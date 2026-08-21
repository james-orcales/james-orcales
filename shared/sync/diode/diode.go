// Package diode is a non-blocking writer that never makes its caller
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
	"errors"
	"sync"
	"sync/atomic"
	"unsafe"

	"local/james-orcales/shared/simulation/aver/default"
	"local/james-orcales/shared/simulation/time"
)

// Ring slot count used when New_Input.Count is unset; 1000 lines of slack absorbs
// typical bursts before the drain has to drop anything.
const DEFAULT_COUNT = 1000

// How long the drain sleeps on an empty ring when New_Input.Poll_Interval is unset.
const DEFAULT_POLL_INTERVAL = 100 * time.MILLISECOND

// Caps the capacity of a line buffer returned to the pool, so one giant line cannot
// bloat every pooled entry (see Go issue 23199).
const MAXIMUM_POOLED_BUFFER = 1 << 16

// SLOT_COUNT_MAXIMUM keeps one ring below the same bounded entry count as one line.
const SLOT_COUNT_MAXIMUM = MAXIMUM_POOLED_BUFFER

// BYTE_RATE_MAXIMUM keeps refill multiplication inside signed duration storage.
const BYTE_RATE_MAXIMUM = MAXIMUM_POOLED_BUFFER * DEFAULT_COUNT

// BURST_SIZE_MAXIMUM shares the rate bound so one full refill cannot overflow.
const BURST_SIZE_MAXIMUM = BYTE_RATE_MAXIMUM

// CONFIGURATION_MINIMUM accepts the one negative value that selects disabled/default behavior.
const CONFIGURATION_MINIMUM = -1

// SLOT_COUNT_MINIMUM is the smallest active ring.
const SLOT_COUNT_MINIMUM = 1

// DATA_SIZE_MINIMUM admits a zero-length writer call.
const DATA_SIZE_MINIMUM = 0

// MISSED_COUNT_MINIMUM is the smallest useful loss report.
const MISSED_COUNT_MINIMUM = 1

// MISSED_COUNT_MAXIMUM is every entry one bounded ring can lose in one observed lap.
const MISSED_COUNT_MAXIMUM = SLOT_COUNT_MAXIMUM

// MESSAGE_SIZE_MINIMUM is the shortest static assertion explanation below.
const MESSAGE_SIZE_MINIMUM = len("diode: Sleep is required")

// MESSAGE_SIZE_MAXIMUM is the longest static assertion explanation below.
const MESSAGE_SIZE_MAXIMUM = len("diode: delivered off the read cursor")

// SEQUENCE_MINIMUM is the first ring cursor.
const SEQUENCE_MINIMUM uint64 = 0

// SEQUENCE_MAXIMUM is the last ring cursor before wrap.
const SEQUENCE_MAXIMUM uint64 = ^uint64(0)

// Missed_Count is one bounded loss report.
type Missed_Count int

// Missed_Count_Invariants bounds one observed lap.
func Missed_Count_Invariants(value Missed_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), MISSED_COUNT_MINIMUM, MISSED_COUNT_MAXIMUM).
		Ensure()
}

// Boolean is one internal ring decision.
type Boolean bool

// Boolean_Invariants requires both decision outcomes.
func Boolean_Invariants(value Boolean, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "A diode decision is true.").
		Ensure()
}

// Line_Size is one bounded queued payload size.
type Line_Size int

// Line_Size_Invariants applies the pooled buffer bound.
func Line_Size_Invariants(value Line_Size, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), DATA_SIZE_MINIMUM, MAXIMUM_POOLED_BUFFER).
		Ensure()
}

// Message is one static assertion explanation.
type Message string

// Message_Invariants spans the static explanations below.
func Message_Invariants(value Message, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), MESSAGE_SIZE_MINIMUM, MESSAGE_SIZE_MAXIMUM).
		Ensure()
}

// ERR_DATA_TOO_LARGE rejects a line that cannot stay inside the pooled byte bound.
var ERR_DATA_TOO_LARGE = errors.New("diode: line exceeds byte limit")

// Drop_Cause distinguishes why a line never reached the sink.
type Drop_Cause int

// Drop_Cause_Invariants admits every declared loss source.
func Drop_Cause_Invariants(value Drop_Cause, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Int(int(value), int(DROP_OVERFLOW), int(DROP_RATE_LIMIT)).
		Ensure()
}

// DROP_OVERFLOW marks a line lost because the ring lapped: the sink could not keep up.
const DROP_OVERFLOW Drop_Cause = 0

// DROP_RATE_LIMIT marks a line shed by the rate limiter.
const DROP_RATE_LIMIT Drop_Cause = 1

// Alerter surfaces dropped lines, tagged with their cause; one is installed per diode.
type Alerter func(missed int, cause Drop_Cause)

// Write is the caller-owned sink procedure.
type Write func(state unsafe.Pointer, data Data) (written Data_Size, err error)

// Close releases the caller-owned sink when it owns a close operation.
type Close func(state unsafe.Pointer) (err error)

// Byte_Rate is a disabled marker or one positive bounded throughput.
type Byte_Rate int

// Byte_Rate_Invariants keeps throughput inside machine storage.
func Byte_Rate_Invariants(value Byte_Rate, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), CONFIGURATION_MINIMUM, BYTE_RATE_MAXIMUM).
		Ensure()
}

// Burst_Size is a default marker or one positive bounded byte budget.
type Burst_Size int

// Burst_Size_Invariants keeps burst size inside machine storage.
func Burst_Size_Invariants(value Burst_Size, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), CONFIGURATION_MINIMUM, BURST_SIZE_MAXIMUM).
		Ensure()
}

// Rate_Limit caps how fast a diode writes to its sink, in bytes per second; lines over the
// budget are shed (reported DROP_RATE_LIMIT), never blocked.
type Rate_Limit struct {
	// Bytes_Per_Second is the sustained throughput to the sink; zero disables the limiter.
	Bytes_Per_Second Byte_Rate
	// Burst is the most bytes that may accumulate while idle; zero defaults to
	// Bytes_Per_Second (one second of slack).
	Burst Burst_Size
}

// Rate_Limit_Invariants composes its two bounded quantities.
func Rate_Limit_Invariants(value Rate_Limit, namespace aver.Namespace) {
	Byte_Rate_Invariants(value.Bytes_Per_Second, namespace)
	Burst_Size_Invariants(value.Burst, namespace)
}

// Writer is compatible with io.Writer. Build it with New; it is always used by pointer because it
// owns a background goroutine and atomic state.
type Writer struct {
	// State stays opaque so ring cursors cannot become caller-owned API.
	State unsafe.Pointer
}

// Writer_Invariants rejects an uninitialized handle.
func Writer_Invariants(value Writer, _ aver.Namespace) {
	aver.Always(value.State != nil, "A diode writer has internal state.")
}

// Poll_Interval is one positive bounded idle wait.
type Poll_Interval time.Duration

// Poll_Interval_Invariants keeps polling below the process uptime bound.
func Poll_Interval_Invariants(value Poll_Interval, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), int64(time.NANOSECOND), int64(time.DAY)).
		Ensure()
}

// Stored_Poll_Interval admits the default marker before New normalizes it.
type Stored_Poll_Interval time.Duration

// Stored_Poll_Interval_Invariants bounds caller configuration.
func Stored_Poll_Interval_Invariants(
	value Stored_Poll_Interval, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), CONFIGURATION_MINIMUM, int64(time.DAY)).
		Ensure()
}

// Slots owns one bounded ring of bucket pointers.
type Slots []unsafe.Pointer

// Slots_Invariants requires active bounded ring storage.
func Slots_Invariants(value Slots, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), SLOT_COUNT_MINIMUM, SLOT_COUNT_MAXIMUM).
		Ensure()
}

// Sequence is one ring cursor.
type Sequence uint64

// Sequence_Invariants states the fixed-width cursor domain.
func Sequence_Invariants(value Sequence, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), SEQUENCE_MINIMUM, SEQUENCE_MAXIMUM).
		Ensure()
}

// Token_Count is one bounded rate-limit balance.
type Token_Count int64

// Token_Count_Invariants keeps the balance inside configured burst bounds.
func Token_Count_Invariants(value Token_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), CONFIGURATION_MINIMUM, int64(BURST_SIZE_MAXIMUM)).
		Ensure()
}

// Writer_State owns the mutable ring behind one opaque Writer handle.
type Writer_State struct {
	// Sink_State keeps callback state explicit instead of captured.
	Sink_State unsafe.Pointer
	// Sink_Write keeps blocking sink ownership outside the ring.
	Sink_Write Write
	// Close_Sink is present only when the composition root transfers ownership.
	Close_Sink Close
	// Clock makes token refill deterministic.
	Clock time.Clock
	// Sleep keeps wall-clock blocking outside this package.
	Sleep func(duration time.Duration)
	// Poll_Interval bounds idle spinning.
	Poll_Interval Poll_Interval
	// Alerter prevents silent loss.
	Alerter Alerter
	// Buffer_Pool amortizes owned line storage.
	Buffer_Pool *sync.Pool
	// Slots bounds unread work.
	Slots Slots
	// Write_Index separates concurrent producer claims.
	Write_Index atomic.Uint64
	// Read_Index belongs to the single drain.
	Read_Index Sequence
	// Rate_Limit is immutable after construction.
	Rate_Limit Rate_Limit
	// Tokens belong to the single drain.
	Tokens Token_Count
	// Last_Refill makes elapsed refill monotonic.
	Last_Refill time.Monotonic_Moment
	// Stop transfers shutdown request to the drain.
	Stop chan struct{}
	// Done acknowledges the final flush.
	Done chan struct{}
}

// Writer_State_Invariants composes the ring's bounded state.
func Writer_State_Invariants(value Writer_State, namespace aver.Namespace) {
	time.Clock_Invariants(value.Clock, namespace)
	Poll_Interval_Invariants(value.Poll_Interval, namespace)
	Slots_Invariants(value.Slots, namespace)
	Sequence_Invariants(value.Read_Index, namespace)
	Rate_Limit_Invariants(value.Rate_Limit, namespace)
	Token_Count_Invariants(value.Tokens, namespace)
	time.Monotonic_Moment_Invariants(value.Last_Refill, namespace)
}

// Slot_Count is a default marker or one positive bounded ring size.
type Slot_Count int

// Slot_Count_Invariants states the complete machine input domain.
func Slot_Count_Invariants(value Slot_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), CONFIGURATION_MINIMUM, SLOT_COUNT_MAXIMUM).
		Ensure()
}

// New_Input configures New.
type New_Input struct {
	// State belongs to the sink procedures and stays opaque to the diode.
	Sink_State unsafe.Pointer
	// Write is the sink to wrap; nil discards.
	Write Write
	// Close releases the sink after the drain stops; nil keeps it borrowed.
	Close Close
	// Clock is the read-only time source the rate limiter refills against; required.
	Clock time.Clock
	// Sleep parks the drain on an empty ring; required (the drain panics without it).
	Sleep func(duration time.Duration)
	// Count is the ring slot count; zero or negative uses DEFAULT_COUNT.
	Count Slot_Count
	// Poll_Interval is the empty-ring sleep; zero or negative uses one hundred milliseconds.
	Poll_Interval Stored_Poll_Interval
	// Rate_Limit caps the sink write rate in bytes per second; the zero value is no limit.
	Rate_Limit Rate_Limit
	// Alerter receives the dropped-entry count and cause; nil installs a no-op.
	Alerter Alerter
}

// New_Input_Invariants composes every caller-stated bounded scalar.
func New_Input_Invariants(value New_Input, namespace aver.Namespace) {
	Slot_Count_Invariants(value.Count, namespace)
	time.Clock_Invariants(value.Clock, namespace)
	Stored_Poll_Interval_Invariants(value.Poll_Interval, namespace)
	Rate_Limit_Invariants(value.Rate_Limit, namespace)
}

// Data is one owned line buffer.
type Data []byte

// Data_Invariants keeps pooled capacity from becoming retained hostile input.
func Data_Invariants(value Data, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), DATA_SIZE_MINIMUM, MAXIMUM_POOLED_BUFFER).
		Ensure()
}

// Data_Size is one bounded line byte count.
type Data_Size int

// Data_Size_Invariants applies the line storage bound.
func Data_Size_Invariants(value Data_Size, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), DATA_SIZE_MINIMUM, MAXIMUM_POOLED_BUFFER).
		Ensure()
}

// One queued line, stored behind an unsafe.Pointer in a ring slot.
// Bucket owns one queued line and its publication sequence.
type Bucket struct {
	// Data is the copied line bytes; it owns its backing array so the pool reuses the
	// bucket and the array together.
	Data Data
	// Sequence is the write index at store time; the drain compares it to Read_Index to
	// tell a fresh entry from a stale leftover or one that lapped it. It is atomic
	// because the recycle path can hand a bucket to a new producer that re-stamps it
	// while another producer still holds a stale pointer to it (from a slot it loaded)
	// and reads Sequence in the collision check — a benign staleness the CAS resolves,
	// but a data race unless the field is atomic.
	Sequence atomic.Uint64
}

// Bucket_Invariants applies the line byte bound.
func Bucket_Invariants(value Bucket, namespace aver.Namespace) {
	Data_Invariants(value.Data, namespace)
}

// New wraps a sink in a non-blocking diode and starts its drain goroutine.
func New(input New_Input) (writer *Writer) {
	defer func() { Writer_Invariants(*writer, "new.writer") }()
	New_Input_Invariants(input, "new.input")
	write := input.Write
	if write == nil {
		write = discard
	}
	count := input.Count
	if count <= 0 {
		count = DEFAULT_COUNT
	}
	interval := input.Poll_Interval
	if interval <= 0 {
		interval = Stored_Poll_Interval(DEFAULT_POLL_INTERVAL)
	}
	alerter := input.Alerter
	if alerter == nil {
		alerter = func(missed int, cause Drop_Cause) {}
	}
	limit := input.Rate_Limit
	if limit.Bytes_Per_Second > 0 {
		if limit.Burst <= 0 {
			limit.Burst = Burst_Size(limit.Bytes_Per_Second)
		}
	}
	assert(input.Sleep != nil, "diode: Sleep is required")
	assert(count > 0, "diode: ring count must be positive")
	state := &Writer_State{
		Sink_State:    input.Sink_State,
		Sink_Write:    write,
		Close_Sink:    input.Close,
		Clock:         input.Clock,
		Sleep:         input.Sleep,
		Poll_Interval: Poll_Interval(interval),
		Alerter:       alerter,
		Rate_Limit:    limit,
		Tokens:        Token_Count(limit.Burst),
		Last_Refill:   time.Clock_Now_Monotonic(input.Clock),
		Buffer_Pool:   &sync.Pool{New: new_bucket},
		Slots:         make(Slots, count),
		Stop:          make(chan struct{}),
		Done:          make(chan struct{}),
	}
	// Start the write cursor one before zero so the first atomic Add yields 0,
	// matching the read cursor's start and keeping slot math symmetric.
	state.Write_Index.Store(^uint64(0))
	writer = &Writer{State: unsafe.Pointer(state)}
	go drain(unsafe.Pointer(state))
	return writer
}

func discard(_ unsafe.Pointer, data Data) (written Data_Size, err error) {
	defer func() { Data_Size_Invariants(written, "discard.written") }()
	Data_Invariants(data, "discard.data")
	return Data_Size(len(data)), nil
}

// Builds a fresh bucket for the pool, with a small starting backing array.
func new_bucket() (item any) {
	return &Bucket{Data: make(Data, 0, 512)}
}

// Write copies p into the ring and returns at once; it never touches the sink, so a
// slow sink never blocks the caller. The copy is mandatory: callers (jlog) reuse p
// the moment Write returns.
func (writer *Writer) Write(p []byte) (n int, err error) {
	state := (*Writer_State)(writer.State)
	item := state.Buffer_Pool.Get().(*Bucket)
	if len(p) > MAXIMUM_POOLED_BUFFER {
		state.Buffer_Pool.Put(item)
		return 0, ERR_DATA_TOO_LARGE
	}
	item.Data = append(item.Data[:0], p...)
	Data_Invariants(item.Data, "write.data")
	assert(len(item.Data) == len(p), "diode: write copy length mismatch")
	ring_set(writer.State, unsafe.Pointer(item))
	return len(p), nil
}

// Close asks the drain to flush what remains and stop, waits for it, then closes the
// wrapped sink through its injected Close procedure. Call it once.
func (writer *Writer) Close() (err error) {
	state := (*Writer_State)(writer.State)
	close(state.Stop)
	<-state.Done
	if state.Close_Sink == nil {
		return nil
	}
	return state.Close_Sink(state.Sink_State)
}

// Stores item in the next ring slot, advancing the shared write cursor atomically so
// concurrent producers never share a slot.
func ring_set(state_pointer unsafe.Pointer, item_pointer unsafe.Pointer) {
	state := (*Writer_State)(state_pointer)
	item := (*Bucket)(item_pointer)
	stored := false
	for !stored {
		index := state.Write_Index.Add(1)
		item.Sequence.Store(index)
		slot := index % uint64(len(state.Slots))
		previous := atomic.LoadPointer(&state.Slots[slot])
		if previous != nil {
			occupant := (*Bucket)(previous)
			lap_start := index - uint64(len(state.Slots))
			if occupant.Sequence.Load() > lap_start {
				// The slot holds an unread bucket from this same lap.
				continue
			}
		}
		stored = atomic.CompareAndSwapPointer(
			&state.Slots[slot], previous, unsafe.Pointer(item))
		if stored {
			assert(previous != unsafe.Pointer(item), "diode: recycled a live bucket")
			recycle_overwritten(state_pointer, previous)
		}
		// A failed CAS means another producer won this slot; loop to the next index.
	}
}

// Returns a bucket that a successful overwrite has dropped back to the pool, keeping a
// diode allocation-free even while shedding load. A successful CAS over a non-nil slot
// proves the drain never took that bucket — a drain read would have nil'd the slot and
// failed the CAS — so the producer holds the only reference and may pool it.
func recycle_overwritten(state_pointer unsafe.Pointer, previous unsafe.Pointer) {
	state := (*Writer_State)(state_pointer)
	if previous == nil {
		return
	}
	dropped := (*Bucket)(previous)
	if cap(dropped.Data) > MAXIMUM_POOLED_BUFFER {
		return
	}
	dropped.Data = dropped.Data[:0]
	state.Buffer_Pool.Put(dropped)
}

// Takes the next entry for the drain, reporting drops when the writer has lapped the
// read cursor. ok is false when nothing fresh is available.
func ring_try_next(
	state_pointer unsafe.Pointer,
) (item_pointer unsafe.Pointer, ok Boolean) {
	defer func() { Boolean_Invariants(ok, "ring_try_next.ok") }()
	state := (*Writer_State)(state_pointer)
	read_index := uint64(state.Read_Index)
	slot := read_index % uint64(len(state.Slots))
	taken := (*Bucket)(atomic.SwapPointer(&state.Slots[slot], nil))
	if taken == nil {
		return nil, false
	}
	sequence := taken.Sequence.Load()
	if sequence < read_index {
		// A stale value from a slot already fast-forwarded past; ignore it.
		return nil, false
	}
	if sequence > read_index {
		dropped := sequence - read_index
		state.Read_Index = Sequence(sequence)
		missed := Missed_Count(MISSED_COUNT_MAXIMUM)
		if dropped <= uint64(MISSED_COUNT_MAXIMUM) {
			missed = Missed_Count(dropped)
		}
		alert(state_pointer, missed, DROP_OVERFLOW)
	}
	assert(
		Boolean(sequence == uint64(state.Read_Index)),
		"diode: delivered off the read cursor",
	)
	state.Read_Index++
	return unsafe.Pointer(taken), true
}

// Drives the single reader: forward entries to the sink, sleep the poll interval on an
// empty ring, and once Close requests a stop flush the remainder and exit.
func drain(state_pointer unsafe.Pointer) {
	state := (*Writer_State)(state_pointer)
	for !is_stopped(state_pointer) {
		item_pointer, ok := ring_try_next(state_pointer)
		if ok {
			forward(state_pointer, item_pointer)
			continue
		}
		state.Sleep(time.Duration(state.Poll_Interval))
	}
	drain_remainder(state_pointer)
	close(state.Done)
}

// Reports whether Close has asked the drain to finish.
func is_stopped(state_pointer unsafe.Pointer) (stopped Boolean) {
	defer func() { Boolean_Invariants(stopped, "is_stopped.stopped") }()
	state := (*Writer_State)(state_pointer)
	select {
	case <-state.Stop:
		return true
	default:
		return false
	}
}

// Flushes whatever is still buffered, used on Close so a clean shutdown does not drop
// already-queued lines.
func drain_remainder(state_pointer unsafe.Pointer) {
	item_pointer, ok := ring_try_next(state_pointer)
	for ok {
		forward(state_pointer, item_pointer)
		item_pointer, ok = ring_try_next(state_pointer)
	}
}

// Writes one line to the sink and returns its bucket to the pool.
func forward(state_pointer unsafe.Pointer, item_pointer unsafe.Pointer) {
	state := (*Writer_State)(state_pointer)
	item := (*Bucket)(item_pointer)
	assert(item != nil, "diode: forward got a nil bucket")
	if rate_limit_sheds(state_pointer, Line_Size(len(item.Data))) {
		alert(state_pointer, 1, DROP_RATE_LIMIT)
	} else {
		// One Write per line, exactly as a synchronous wrapped writer would have seen.
		state.Sink_Write(state.Sink_State, item.Data)
	}
	if cap(item.Data) > MAXIMUM_POOLED_BUFFER {
		return
	}
	item.Data = item.Data[:0]
	state.Buffer_Pool.Put(item)
}

// Reports whether the rate limiter sheds a line of size bytes. It refills the byte token
// bucket from the clock, caps it at the burst, and a non-positive balance means shed (the
// line is dropped, never delayed). A disabled limiter (Bytes_Per_Second <= 0) never sheds.
// Only the drain goroutine calls this, so the bucket state needs no synchronization.
func rate_limit_sheds(
	state_pointer unsafe.Pointer, size Line_Size,
) (shed Boolean) {
	defer func() { Boolean_Invariants(shed, "rate_limit_sheds.shed") }()
	Line_Size_Invariants(size, "rate_limit_sheds.size")
	state := (*Writer_State)(state_pointer)
	rate := int64(state.Rate_Limit.Bytes_Per_Second)
	if rate <= 0 {
		return false
	}
	burst := int64(state.Rate_Limit.Burst)
	now := time.Clock_Now_Monotonic(state.Clock)
	elapsed := int64(now) - int64(state.Last_Refill)
	state.Last_Refill = now
	// Cap elapsed at the time to refill a full burst: beyond that the refill is wasted (the
	// balance is capped at Burst anyway), and the cap keeps elapsed*rate from overflowing
	// int64 after a long idle gap or against a large monotonic clock reading.
	full := burst * int64(time.SECOND) / rate
	if elapsed > full {
		elapsed = full
	}
	state.Tokens += Token_Count(elapsed * rate / int64(time.SECOND))
	if state.Tokens > Token_Count(burst) {
		state.Tokens = Token_Count(burst)
	}
	if state.Tokens <= 0 {
		return true
	}
	state.Tokens -= Token_Count(size)
	return false
}

func alert(state_pointer unsafe.Pointer, missed Missed_Count, cause Drop_Cause) {
	Missed_Count_Invariants(missed, "alert.missed")
	Drop_Cause_Invariants(cause, "alert.cause")
	state := (*Writer_State)(state_pointer)
	state.Alerter(int(missed), cause)
}

// Panics with message when condition is false. Unlike the invariant framework it captures
// no caller site, so it inlines to a single predictable branch over a constant string —
// free on the hot path and allocation-free. The panic's stack trace carries the location;
// message names the invariant.
func assert(condition Boolean, message Message) {
	Boolean_Invariants(condition, "assert.condition")
	Message_Invariants(message, "assert.message")
	if !condition {
		panic(message)
	}
}
