// Package diode is a non-blocking io.Writer: a writer that never makes its caller
// wait on the underlying sink. Each Write copies the bytes into a lock-free ring
// buffer and returns immediately; a single background goroutine drains the ring into
// the wrapped writer. When producers outrun the sink past the ring's capacity, the
// oldest unread entries are overwritten — bounded memory, never blocking — and the
// count of lost entries is reported through an injected Alerter.
//
// This is a house-style port of the diode in CloudFoundry's go-diodes (the same
// structure rs/zerolog wraps). It uses the Poller strategy: when the ring is empty
// the drain sleeps for Poll_Interval before checking again. The sleep is taken via
// the injected time.Clock so the library tier holds no impure time dependency and
// the drain loop stays deterministic under a virtual clock in tests.
//
// A diode trades reliability for non-blocking writes. Lines can be dropped under
// sustained overload (surfaced via the Alerter, never silently), and lines still
// buffered when the process exits are lost unless Close is called to flush. Code that
// needs guaranteed delivery should write to its sink synchronously instead.
package diode

import (
	"io"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/james-orcales/james-orcales/shared/time"
)

// Ring slot count used when New_Input.Count is unset; 1000 lines of slack absorbs
// typical bursts before the drain has to drop anything.
const default_count = 1000

// How long the drain sleeps on an empty ring when New_Input.Poll_Interval is unset.
const default_poll_interval = 100 * time.Millisecond

// Caps the capacity of a line buffer returned to the pool, so one giant line cannot
// bloat every pooled entry (see Go issue 23199).
const maximum_pooled_buffer = 1 << 16

// Drop_Cause distinguishes why a line never reached the sink.
type Drop_Cause int

// Drop_Overflow marks a line lost because the ring lapped: the sink could not keep up.
const Drop_Overflow Drop_Cause = 0

// Drop_Rate_Limit marks a line shed by the rate limiter.
const Drop_Rate_Limit Drop_Cause = 1

// Alerter surfaces dropped lines, tagged with their cause; one is installed per diode.
type Alerter func(missed int, cause Drop_Cause)

// Rate_Limit caps how fast a diode writes to its sink, in bytes per second; lines over the
// budget are shed (reported Drop_Rate_Limit), never blocked.
type Rate_Limit struct {
	// Bytes_Per_Second is the sustained throughput to the sink; zero disables the limiter.
	Bytes_Per_Second int
	// Burst is the most bytes that may accumulate while idle; zero defaults to
	// Bytes_Per_Second (one second of slack).
	Burst int
}

// Writer is a non-blocking io.Writer. Build it with New; it is always used by pointer
// because it owns a background goroutine and atomic state.
type Writer struct {
	// Writer is the wrapped sink the drain goroutine forwards finished lines to.
	Writer io.Writer
	// Clock supplies the drain loop's Sleep; the library tier never calls stdlib
	// time, so a real sleep is injected here (and a virtual one in tests).
	Clock time.Clock
	// Poll_Interval is how long the drain sleeps when it finds the ring empty.
	Poll_Interval time.Duration
	// Alerter surfaces dropped lines (overflow or rate-limit) rather than losing them
	// silently; see Drop_Cause.
	Alerter Alerter
	// Buffer_Pool recycles the per-line buckets and their backing arrays so a steady
	// stream of writes allocates nothing once the pool is warm.
	Buffer_Pool *sync.Pool
	// Slots is the ring; each slot holds an unsafe.Pointer to a *bucket or nil.
	Slots []unsafe.Pointer
	// Write_Index is the producers' shared cursor, advanced atomically so many
	// goroutines can claim distinct slots without a lock.
	Write_Index atomic.Uint64
	// Read_Index is the drain's cursor; only the single drain goroutine touches it,
	// so it needs no atomic access.
	Read_Index uint64
	// Rate_Limit caps the sink write rate; zero Bytes_Per_Second disables it.
	Rate_Limit Rate_Limit
	// Tokens is the rate limiter's byte balance, touched only by the drain goroutine.
	Tokens int64
	// Last_Refill is when Tokens were last replenished, for the drain's elapsed-time refill.
	Last_Refill time.Moment
	// Stop is closed by Close to ask the drain to flush and exit.
	Stop chan struct{}
	// Done is closed by the drain once it has flushed and returned.
	Done chan struct{}
}

// New_Input configures New.
type New_Input struct {
	// Writer is the sink to wrap; nil becomes io.Discard.
	Writer io.Writer
	// Clock supplies the drain's Sleep; required (the drain panics without it).
	Clock time.Clock
	// Count is the ring slot count; zero or negative uses default_count.
	Count int
	// Poll_Interval is the empty-ring sleep; zero or negative uses one hundred milliseconds.
	Poll_Interval time.Duration
	// Rate_Limit caps the sink write rate in bytes per second; the zero value is no limit.
	Rate_Limit Rate_Limit
	// Alerter receives the dropped-entry count and cause; nil installs a no-op.
	Alerter Alerter
}

// One queued line, stored behind an unsafe.Pointer in a ring slot.
type bucket struct {
	// Data is the copied line bytes; it owns its backing array so the pool reuses the
	// bucket and the array together.
	Data []byte
	// Sequence is the write index at store time; the drain compares it to Read_Index to
	// tell a fresh entry from a stale leftover or one that lapped it. It is atomic
	// because the recycle path can hand a bucket to a new producer that re-stamps it
	// while another producer still holds a stale pointer to it (from a slot it loaded)
	// and reads Sequence in the collision check — a benign staleness the CAS resolves,
	// but a data race unless the field is atomic.
	Sequence atomic.Uint64
}

// New wraps writer in a non-blocking diode and starts its drain goroutine.
func New(input New_Input) (writer *Writer) {
	sink := input.Writer
	if sink == nil {
		sink = io.Discard
	}
	count := input.Count
	if count <= 0 {
		count = default_count
	}
	interval := input.Poll_Interval
	if interval <= 0 {
		interval = default_poll_interval
	}
	alerter := input.Alerter
	if alerter == nil {
		alerter = func(missed int, cause Drop_Cause) {}
	}
	limit := input.Rate_Limit
	if limit.Bytes_Per_Second > 0 {
		if limit.Burst <= 0 {
			limit.Burst = limit.Bytes_Per_Second
		}
	}
	assert(input.Clock.Sleep != nil, "diode: Clock.Sleep is required")
	assert(count > 0, "diode: ring count must be positive")
	writer = &Writer{
		Writer:        sink,
		Clock:         input.Clock,
		Poll_Interval: interval,
		Alerter:       alerter,
		Rate_Limit:    limit,
		Tokens:        int64(limit.Burst),
		Last_Refill:   input.Clock.Now_Monotonic(),
		Buffer_Pool:   &sync.Pool{New: new_bucket},
		Slots:         make([]unsafe.Pointer, count),
		Stop:          make(chan struct{}),
		Done:          make(chan struct{}),
	}
	// Start the write cursor one before zero so the first atomic Add yields 0,
	// matching the read cursor's start and keeping slot math symmetric.
	writer.Write_Index.Store(^uint64(0))
	go drain(writer)
	return writer
}

// Builds a fresh bucket for the pool, with a small starting backing array.
func new_bucket() (item any) {
	return &bucket{Data: make([]byte, 0, 512)}
}

// Write copies p into the ring and returns at once; it never touches the sink, so a
// slow sink never blocks the caller. The copy is mandatory: callers (jlog) reuse p
// the moment Write returns.
func (writer *Writer) Write(p []byte) (n int, err error) {
	item := writer.Buffer_Pool.Get().(*bucket)
	item.Data = append(item.Data[:0], p...)
	assert(len(item.Data) == len(p), "diode: write copy length mismatch")
	ring_set(writer, item)
	return len(p), nil
}

// Close asks the drain to flush what remains and stop, waits for it, then closes the
// wrapped sink if it implements io.Closer. Call it once.
func (writer *Writer) Close() (err error) {
	close(writer.Stop)
	<-writer.Done
	closer, ok := writer.Writer.(io.Closer)
	if !ok {
		return nil
	}
	return closer.Close()
}

// Stores item in the next ring slot, advancing the shared write cursor atomically so
// concurrent producers never share a slot.
func ring_set(writer *Writer, item *bucket) {
	stored := false
	for !stored {
		index := writer.Write_Index.Add(1)
		item.Sequence.Store(index)
		slot := index % uint64(len(writer.Slots))
		previous := atomic.LoadPointer(&writer.Slots[slot])
		if ring_collides(previous, index, len(writer.Slots)) {
			// The slot holds an unread bucket from this same lap: the ring is too
			// small for the write rate. Skip this index and try the next.
			continue
		}
		stored = atomic.CompareAndSwapPointer(
			&writer.Slots[slot], previous, unsafe.Pointer(item))
		if stored {
			assert(previous != unsafe.Pointer(item), "diode: recycled a live bucket")
			recycle_overwritten(writer, previous)
		}
		// A failed CAS means another producer won this slot; loop to the next index.
	}
}

// Returns a bucket that a successful overwrite has dropped back to the pool, keeping a
// diode allocation-free even while shedding load. A successful CAS over a non-nil slot
// proves the drain never took that bucket — a drain read would have nil'd the slot and
// failed the CAS — so the producer holds the only reference and may pool it.
func recycle_overwritten(writer *Writer, previous unsafe.Pointer) {
	if previous == nil {
		return
	}
	dropped := (*bucket)(previous)
	if cap(dropped.Data) > maximum_pooled_buffer {
		return
	}
	dropped.Data = dropped.Data[:0]
	writer.Buffer_Pool.Put(dropped)
}

// Reports whether the slot's occupant is an unread bucket from the current lap, which
// a store would corrupt.
func ring_collides(previous unsafe.Pointer, index uint64, count int) (collides bool) {
	if previous == nil {
		return false
	}
	occupant := (*bucket)(previous)
	if occupant.Sequence.Load() > index-uint64(count) {
		return true
	}
	return false
}

// Takes the next entry for the drain, reporting drops when the writer has lapped the
// read cursor. ok is false when nothing fresh is available.
func ring_try_next(writer *Writer) (item *bucket, ok bool) {
	slot := writer.Read_Index % uint64(len(writer.Slots))
	taken := (*bucket)(atomic.SwapPointer(&writer.Slots[slot], nil))
	if taken == nil {
		return nil, false
	}
	sequence := taken.Sequence.Load()
	if sequence < writer.Read_Index {
		// A stale value from a slot already fast-forwarded past; ignore it.
		return nil, false
	}
	if sequence > writer.Read_Index {
		dropped := sequence - writer.Read_Index
		writer.Read_Index = sequence
		writer.Alerter(int(dropped), Drop_Overflow)
	}
	assert(sequence == writer.Read_Index, "diode: delivered off the read cursor")
	writer.Read_Index++
	return taken, true
}

// Drives the single reader: forward entries to the sink, sleep the poll interval on an
// empty ring, and once Close requests a stop flush the remainder and exit.
func drain(writer *Writer) {
	for !is_stopped(writer) {
		item, ok := ring_try_next(writer)
		if ok {
			forward(writer, item)
			continue
		}
		writer.Clock.Sleep(writer.Poll_Interval)
	}
	drain_remainder(writer)
	close(writer.Done)
}

// Reports whether Close has asked the drain to finish.
func is_stopped(writer *Writer) (stopped bool) {
	select {
	case <-writer.Stop:
		return true
	default:
		return false
	}
}

// Flushes whatever is still buffered, used on Close so a clean shutdown does not drop
// already-queued lines.
func drain_remainder(writer *Writer) {
	item, ok := ring_try_next(writer)
	for ok {
		forward(writer, item)
		item, ok = ring_try_next(writer)
	}
}

// Writes one line to the sink and returns its bucket to the pool.
func forward(writer *Writer, item *bucket) {
	assert(item != nil, "diode: forward got a nil bucket")
	if rate_limit_sheds(writer, len(item.Data)) {
		writer.Alerter(1, Drop_Rate_Limit)
	} else {
		// One Write per line, exactly as a synchronous wrapped writer would have seen.
		writer.Writer.Write(item.Data)
	}
	if cap(item.Data) > maximum_pooled_buffer {
		return
	}
	item.Data = item.Data[:0]
	writer.Buffer_Pool.Put(item)
}

// Reports whether the rate limiter sheds a line of size bytes. It refills the byte token
// bucket from the clock, caps it at the burst, and a non-positive balance means shed (the
// line is dropped, never delayed). A disabled limiter (Bytes_Per_Second <= 0) never sheds.
// Only the drain goroutine calls this, so the bucket state needs no synchronization.
func rate_limit_sheds(writer *Writer, size int) (shed bool) {
	rate := int64(writer.Rate_Limit.Bytes_Per_Second)
	if rate <= 0 {
		return false
	}
	burst := int64(writer.Rate_Limit.Burst)
	now := writer.Clock.Now_Monotonic()
	elapsed := int64(now) - int64(writer.Last_Refill)
	writer.Last_Refill = now
	// Cap elapsed at the time to refill a full burst: beyond that the refill is wasted (the
	// balance is capped at Burst anyway), and the cap keeps elapsed*rate from overflowing
	// int64 after a long idle gap or against a large monotonic clock reading.
	full := burst * int64(time.Second) / rate
	if elapsed > full {
		elapsed = full
	}
	writer.Tokens += elapsed * rate / int64(time.Second)
	if writer.Tokens > burst {
		writer.Tokens = burst
	}
	if writer.Tokens <= 0 {
		return true
	}
	writer.Tokens -= int64(size)
	return false
}

// Panics with message when condition is false. Unlike the invariant framework it captures
// no caller site, so it inlines to a single predictable branch over a constant string —
// free on the hot path and allocation-free. The panic's stack trace carries the location;
// message names the invariant.
func assert(condition bool, message string) {
	if !condition {
		panic(message)
	}
}
