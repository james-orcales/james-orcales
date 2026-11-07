// Package diode is a non-blocking Stream that never makes its caller wait on the inner Stream.
// Each Write copies the bytes into a lock-free ring buffer and retires immediately. The root
// drives the drain: each Writer_Rearm submits one queued line to the inner Stream when nothing
// is in flight. When producers outrun the inner Stream past the ring's capacity, the oldest
// unread entries are overwritten — bounded memory, never blocking — and the count of lost
// entries is reported through an injected Alerter.
//
// This is a house-style port of the diode in CloudFoundry's go-diodes (the same structure
// rs/zerolog wraps). The poller strategy is gone: no goroutine sleeps on an empty ring, and the
// library never pumps itself; the timeline holder calls Writer_Rearm (see inject_io).
//
// A diode trades reliability for non-blocking writes. Lines can be dropped under sustained
// overload (surfaced via the Alerter, never silently), and lines still buffered when the process
// exits are lost unless Close is called to flush. Code that needs guaranteed delivery should
// write to its Stream directly instead.
package diode

import "errors"
import "sync/atomic"
import "unsafe"

import "local/james-orcales/shared/std/sim/aver/default"
import "local/james-orcales/shared/std/sim/nbio"
import "local/james-orcales/shared/std/sim/time"

// Ring slot count a root should size its storage to when it has no better number; 1000 lines
// of slack absorbs typical bursts before the drain has to drop anything.
const DEFAULT_COUNT = 1000

// LINE_BYTES_MAXIMUM is one bucket's fixed storage; a longer line is refused. Buckets are
// preallocated, so this times the bucket count is the diode's whole retained memory.
const LINE_BYTES_MAXIMUM = 1 << 12

// BUCKET_SLACK is how many producers may hold a bucket outside the ring at once (mid-Write)
// before a Write finds none free. Sixty-four exceeds any plausible concurrent producer count.
const BUCKET_SLACK = 64

// SLOT_COUNT_MAXIMUM bounds one ring.
const SLOT_COUNT_MAXIMUM = 1 << 16

// SLOT_COUNT_MINIMUM is the smallest active ring.
const SLOT_COUNT_MINIMUM = 1

// BUCKET_COUNT_MINIMUM is the smallest ring's buckets plus its producer slack.
const BUCKET_COUNT_MINIMUM = SLOT_COUNT_MINIMUM + BUCKET_SLACK

// BUCKET_COUNT_MAXIMUM is the largest ring's buckets plus its producer slack.
const BUCKET_COUNT_MAXIMUM = SLOT_COUNT_MAXIMUM + BUCKET_SLACK

// LINE_ARENA_SIZE_MINIMUM is one full line per bucket of the smallest ring.
const LINE_ARENA_SIZE_MINIMUM = BUCKET_COUNT_MINIMUM * LINE_BYTES_MAXIMUM

// LINE_ARENA_SIZE_MAXIMUM is one full line per bucket of the largest ring.
const LINE_ARENA_SIZE_MAXIMUM = BUCKET_COUNT_MAXIMUM * LINE_BYTES_MAXIMUM

// BUCKET_INDEX_NONE is the free-list terminator and the "nothing in flight" marker; bucket
// indexes are one-based so zero can mean absent.
const BUCKET_INDEX_NONE = 0

// FREE_LIST_RETRIES_MAX bounds a free-list compare-and-swap loop. A retry needs another
// producer to win between the load and the swap; this many losses in a row does not happen
// with BUCKET_SLACK producers, so exhausting it is reported, never spun on.
const FREE_LIST_RETRIES_MAX = 1 << 16

// BYTE_RATE_MAXIMUM keeps refill multiplication inside signed duration storage.
const BYTE_RATE_MAXIMUM = SLOT_COUNT_MAXIMUM * DEFAULT_COUNT

// BURST_SIZE_MAXIMUM shares the rate bound so one full refill cannot overflow.
const BURST_SIZE_MAXIMUM = BYTE_RATE_MAXIMUM

// TOKEN_COUNT_MINIMUM is the deepest overdraft: a positive balance of one spent on a full line.
const TOKEN_COUNT_MINIMUM = 1 - LINE_BYTES_MAXIMUM

// CONFIGURATION_MINIMUM accepts the one negative value that selects disabled/default behavior.
const CONFIGURATION_MINIMUM = -1

// DATA_SIZE_MINIMUM admits a zero-length writer call.
const DATA_SIZE_MINIMUM = 0

// DATA_SIZE_MAXIMUM bounds what a producer may hand over at all; between LINE_BYTES_MAXIMUM
// and here a Write is refused with ERR_DATA_TOO_LARGE rather than failing an invariant.
const DATA_SIZE_MAXIMUM = 1 << 16

// MISSED_COUNT_MINIMUM is the smallest useful loss report.
const MISSED_COUNT_MINIMUM = 1

// MISSED_COUNT_MAXIMUM is every entry one bounded ring can lose in one observed lap.
const MISSED_COUNT_MAXIMUM = SLOT_COUNT_MAXIMUM

// SEQUENCE_MINIMUM is the first ring cursor.
const SEQUENCE_MINIMUM uint64 = 0

// SEQUENCE_MAXIMUM is the last ring cursor before wrap.
const SEQUENCE_MAXIMUM uint64 = ^uint64(0)

// FREE_LIST_INDEX_BITS is the low half of the free-list head; the high half is the ABA tag.
const FREE_LIST_INDEX_BITS = 32

// FREE_LIST_INDEX_MASK extracts the one-based bucket index from the free-list head.
const FREE_LIST_INDEX_MASK = 1<<FREE_LIST_INDEX_BITS - 1

// WRITER_MODES excludes reads and cursor modes because the ring has no position to report.
const WRITER_MODES nbio.Stream_Mode_Set = 1<<nbio.STREAM_MODE_CLOSE |
  1<<nbio.STREAM_MODE_FLUSH |
  1<<nbio.STREAM_MODE_WRITE |
  1<<nbio.STREAM_MODE_DESTROY |
  1<<nbio.STREAM_MODE_QUERY

// CLOSE_OPEN means the inner Close has not been submitted.
const CLOSE_OPEN Close_State = 0

// CLOSE_SUBMITTED means the inner Close went out; nothing more may follow it.
const CLOSE_SUBMITTED Close_State = 1

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

// Line_Size_Invariants applies the bucket storage bound.
func Line_Size_Invariants(value Line_Size, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(int(value), DATA_SIZE_MINIMUM, LINE_BYTES_MAXIMUM).
    Ensure()
}

// ERR_DATA_TOO_LARGE rejects a line that cannot fit one bucket.
var ERR_DATA_TOO_LARGE = errors.New("diode: line exceeds byte limit")

// ERR_NO_BUCKET reports more than BUCKET_SLACK producers inside Write at once.
var ERR_NO_BUCKET = errors.New("diode: no free bucket")

// Drop_Cause distinguishes why a line never reached the inner Stream.
type Drop_Cause int

// Drop_Cause_Invariants admits every declared loss source.
func Drop_Cause_Invariants(value Drop_Cause, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_Int(int(value), int(DROP_OVERFLOW), int(DROP_RATE_LIMIT)).
    Ensure()
}

// DROP_OVERFLOW marks a line lost because the ring lapped: the inner Stream could not keep up.
const DROP_OVERFLOW Drop_Cause = 0

// DROP_RATE_LIMIT marks a line shed by the rate limiter.
const DROP_RATE_LIMIT Drop_Cause = 1

// Alerter surfaces dropped lines, tagged with their cause; one is installed per diode.
type Alerter func(missed int, cause Drop_Cause)

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

// Rate_Limit caps how fast a diode writes to its inner Stream, in bytes per second; lines over
// the budget are shed (reported DROP_RATE_LIMIT), never delayed.
type Rate_Limit struct {
  // Bytes_Per_Second is the sustained throughput; zero disables the limiter.
  Bytes_Per_Second Byte_Rate
  // Burst is the most bytes that may accumulate while idle; zero defaults to
  // Bytes_Per_Second (one second of slack).
  Burst Burst_Size
}

// Rate_Limit_Invariants composes its two bounded quantities.
func Rate_Limit_Invariants(value *Rate_Limit, namespace aver.Namespace) {
  Byte_Rate_Invariants(value.Bytes_Per_Second, namespace)
  Burst_Size_Invariants(value.Burst, namespace)
}

// Writer is the opaque diode handle. New builds it over caller storage; submit to it through
// Writer_To_Stream.
type Writer struct {
  // State stays opaque so ring cursors cannot become caller-owned API, and so the handle's
  // invariants never copy the ring's concurrently written fields.
  State unsafe.Pointer
}

// Writer_Invariants rejects an uninitialized handle.
func Writer_Invariants(value *Writer, _ aver.Namespace) {
  aver.Always(value.State != nil, "A diode writer has internal state.")
}

// Writer_Pointer is the handle every diode operation takes.
type Writer_Pointer *Writer

// Writer_Pointer_Invariants composes the handle only when present.
func Writer_Pointer_Invariants(value Writer_Pointer, namespace aver.Namespace) {
  if value == nil {
    return
  }
  Writer_Invariants(value, namespace)
}

// Free_List_Head is the lock-free stack head over Buckets: one-based index in the low bits, an
// ABA tag in the high bits, BUCKET_INDEX_NONE when empty.
type Free_List_Head uint64

// Free_List_Head_Invariants states the fixed-width head domain.
func Free_List_Head_Invariants(value Free_List_Head, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint64(uint64(value), SEQUENCE_MINIMUM, SEQUENCE_MAXIMUM).
    Ensure()
}

// Close_Flag is how far the producers' Close has progressed, stored 32-bit wide for the atomic
// swap. Accessed only through sync/atomic functions.
type Close_Flag uint32

// Close_Flag_Invariants admits the three publication steps.
func Close_Flag_Invariants(value Close_Flag, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_3_Uint32(
    uint32(value),
    uint32(CLOSE_FLAG_CLEAR),
    uint32(CLOSE_FLAG_CLAIMED),
    uint32(CLOSE_FLAG_PUBLISHED),
  ).
    Ensure()
}

// CLOSE_FLAG_CLEAR means no Close has arrived.
const CLOSE_FLAG_CLEAR Close_Flag = 0

// CLOSE_FLAG_CLAIMED means one Close caller won the swap and is writing its completion and
// callback; later Writes already report Stream_Empty.
const CLOSE_FLAG_CLAIMED Close_Flag = 1

// CLOSE_FLAG_PUBLISHED means the completion and callback are written and the drain may forward
// the Close. Stored after both fields so the drain never reads a half-published Close.
const CLOSE_FLAG_PUBLISHED Close_Flag = 2

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

// Write_Cursor is the producers' shared claim counter; each claim's value becomes that entry's
// Sequence. Its own type because an invariant chain names each subject type once.
type Write_Cursor uint64

// Write_Cursor_Invariants states the fixed-width cursor domain.
func Write_Cursor_Invariants(value Write_Cursor, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint64(uint64(value), SEQUENCE_MINIMUM, SEQUENCE_MAXIMUM).
    Ensure()
}

// Token_Count is one bounded rate-limit balance.
type Token_Count int64

// Token_Count_Invariants keeps the balance inside configured burst bounds.
func Token_Count_Invariants(value Token_Count, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int64(
    int64(value),
    TOKEN_COUNT_MINIMUM,
    int64(BURST_SIZE_MAXIMUM),
  ).
    Ensure()
}

// Bucket_Index is a one-based position in Buckets, or BUCKET_INDEX_NONE.
type Bucket_Index uint32

// Bucket_Index_Invariants keeps an index inside the largest bucket array.
func Bucket_Index_Invariants(value Bucket_Index, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint32(uint32(value), BUCKET_INDEX_NONE, BUCKET_COUNT_MAXIMUM).
    Ensure()
}

// Close_State records whether the inner Close has gone out.
type Close_State int

// Close_State_Invariants admits the two lifecycle positions.
func Close_State_Invariants(value Close_State, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_Int(int(value), int(CLOSE_OPEN), int(CLOSE_SUBMITTED)).
    Ensure()
}

// Line_Bytes is one bucket's fixed storage, a window into the arena New allocates once.
type Line_Bytes []byte

// Line_Bytes_Invariants requires every bucket to hold a full line.
func Line_Bytes_Invariants(value Line_Bytes, namespace aver.Namespace) {
  aver.Always(len(value) == LINE_BYTES_MAXIMUM, "A bucket holds one full line.")
}

// Bucket owns one queued line and its publication sequence.
type Bucket struct {
  // Data is the fixed storage the line is copied into.
  Data Line_Bytes
  // Size is how much of Data the line uses.
  Size Line_Size
  // Sequence is the write index at store time; the drain compares it to Read_Index to
  // tell a fresh entry from a stale leftover or one that lapped it. Accessed only through
  // sync/atomic functions because the recycle path can hand a bucket to a new producer
  // that re-stamps it while another producer still holds a stale pointer to it (from a
  // slot it loaded) and reads Sequence in the collision check — a benign staleness the
  // CAS resolves, but a data race unless the access is atomic. A plain field rather than
  // atomic.Uint64 so the value-typed invariant bundle passes vet's copylocks check.
  Sequence Sequence
  // Next links this bucket into the free list while it is free: the index of the bucket
  // below it, BUCKET_INDEX_NONE at the bottom. Atomic access, as Sequence.
  Next Bucket_Index
  // Index is this bucket's own one-based position, so a pointer taken from a ring slot can
  // be pushed back onto the free list.
  Index Bucket_Index
}

// Bucket_Invariants composes the line and its position. Never called while producers run: the
// copy it takes would race the atomic fields.
func Bucket_Invariants(value *Bucket, namespace aver.Namespace) {
  Line_Bytes_Invariants(value.Data, namespace)
  Line_Size_Invariants(value.Size, namespace)
  Sequence_Invariants(value.Sequence, namespace)
  Bucket_Index_Invariants(value.Next, namespace)
  Bucket_Index_Invariants(value.Index, namespace)
}

// Buckets is every line's storage, caller-provided; the ring and the free list hold indexes
// into it.
type Buckets []Bucket

// Buckets_Invariants bounds the array to a ring plus its producer slack.
func Buckets_Invariants(value Buckets, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(len(value), BUCKET_COUNT_MINIMUM, BUCKET_COUNT_MAXIMUM).
    Ensure()
}

// Line_Arena is every bucket's line storage in one caller-provided run; New windows each
// bucket's Data into it. One arena rather than one array per bucket keeps storage contiguous.
type Line_Arena []byte

// Line_Arena_Invariants bounds the arena to whole lines for the bucket range.
func Line_Arena_Invariants(value Line_Arena, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(
    len(value),
    LINE_ARENA_SIZE_MINIMUM,
    LINE_ARENA_SIZE_MAXIMUM,
  ).
    Ensure()
}

// Writer_State owns the mutable ring behind one opaque Writer handle. The caller owns this
// storage and hands it to New; the ring's slices arrive through New_Input. Fields marked
// drain-owned are touched only by Writer_Rearm and the inner retirement, both of which the root
// runs on its one thread, so they need no synchronization. Fields marked atomic are plain
// integers accessed only through sync/atomic functions: typed atomics carry a noCopy marker
// that vet rejects in the value-typed invariant bundle.
type Writer_State struct {
  // Completion sits first so the retire callback, which receives only the Completion,
  // recovers its Writer_State without a captured closure. Drain-owned.
  Completion nbio.Completion
  // Stream is the inner transport that lines are submitted to.
  Stream nbio.Stream
  // Clock makes token refill deterministic.
  Clock time.Clock
  // Alerter prevents silent loss.
  Alerter Alerter
  // Buckets is every line's storage.
  Buckets Buckets
  // Free is the free list over Buckets. Atomic.
  Free Free_List_Head
  // Slots bounds unread work.
  Slots Slots
  // Write_Index separates concurrent producer claims. Atomic.
  Write_Index Write_Cursor
  // Read_Index is drain-owned.
  Read_Index Sequence
  // Rate_Limit is immutable after construction.
  Rate_Limit Rate_Limit
  // Tokens are drain-owned.
  Tokens Token_Count
  // Last_Refill makes elapsed refill monotonic. Drain-owned.
  Last_Refill time.Monotonic_Moment
  // In_Flight is the bucket the inner Stream still borrows, or BUCKET_INDEX_NONE.
  // Drain-owned.
  In_Flight Bucket_Index
  // Close_Flag admits one Close caller and publishes Close_Completion and Close_Callback to
  // the drain. Atomic.
  Close_Flag Close_Flag
  // Close_Completion is the caller's Close storage, forwarded to the inner Close.
  Close_Completion nbio.Completion_Handle
  // Close_Callback is the caller's Close retirement.
  Close_Callback nbio.Stream_Callback
  // Close is where the inner Close stands. Drain-owned.
  Close Close_State
}

// Writer_State_Invariants composes the ring's bounded state. Never called while producers run:
// the copy it takes would race the atomic fields.
func Writer_State_Invariants(value *Writer_State, namespace aver.Namespace) {
  time.Clock_Invariants(&value.Clock, namespace)
  Buckets_Invariants(value.Buckets, namespace)
  Free_List_Head_Invariants(value.Free, namespace)
  Slots_Invariants(value.Slots, namespace)
  Write_Cursor_Invariants(value.Write_Index, namespace)
  Sequence_Invariants(value.Read_Index, namespace)
  Rate_Limit_Invariants(&value.Rate_Limit, namespace)
  Token_Count_Invariants(value.Tokens, namespace)
  time.Monotonic_Moment_Invariants(value.Last_Refill, namespace)
  Bucket_Index_Invariants(value.In_Flight, namespace)
  Close_Flag_Invariants(value.Close_Flag, namespace)
  Close_State_Invariants(value.Close, namespace)
}

// New_Input configures New. The three storage slices are caller-owned and must agree: one
// bucket per slot plus BUCKET_SLACK, one line of arena per bucket.
type New_Input struct {
  // Stream receives drained lines; the zero Stream retires each line as Stream_Empty.
  Stream nbio.Stream
  // Clock is the read-only time source the rate limiter refills against; required.
  Clock time.Clock
  // Slots is the ring; its length is the ring slot count.
  Slots Slots
  // Buckets is line storage: len(Slots) + BUCKET_SLACK entries.
  Buckets Buckets
  // Arena is len(Buckets) * LINE_BYTES_MAXIMUM bytes.
  Arena Line_Arena
  // Rate_Limit caps the write rate in bytes per second; the zero value is no limit.
  Rate_Limit Rate_Limit
  // Alerter receives the dropped-entry count and cause; nil means drops go unreported.
  Alerter Alerter
}

// New_Input_Invariants composes every caller-stated bounded quantity and ties the three
// storage lengths together.
func New_Input_Invariants(value *New_Input, namespace aver.Namespace) {
  time.Clock_Invariants(&value.Clock, namespace)
  Slots_Invariants(value.Slots, namespace)
  Buckets_Invariants(value.Buckets, namespace)
  Line_Arena_Invariants(value.Arena, namespace)
  Rate_Limit_Invariants(&value.Rate_Limit, namespace)
  aver.Always(
    len(value.Buckets) == len(value.Slots)+BUCKET_SLACK,
    "Buckets hold one entry per slot plus the producer slack.",
  )
  aver.Always(
    len(value.Arena) == len(value.Buckets)*LINE_BYTES_MAXIMUM,
    "The arena holds one full line per bucket.",
  )
}

// Data is one line as the producer hands it over.
type Data []byte

// Data_Invariants bounds what a producer may hand over; the bucket bound is checked after.
func Data_Invariants(value Data, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(len(value), DATA_SIZE_MINIMUM, DATA_SIZE_MAXIMUM).
    Ensure()
}

// Data_Size is one bounded line byte count.
type Data_Size int

// Data_Size_Invariants applies the line storage bound.
func Data_Size_Invariants(value Data_Size, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(int(value), DATA_SIZE_MINIMUM, LINE_BYTES_MAXIMUM).
    Ensure()
}

// New initialises the caller's Writer_State in place, given as storage, and returns the handle
// over it, wrapping an inner Stream in a non-blocking diode. It starts no goroutine and never
// drives: the root calls Writer_Rearm to submit queued lines. Nothing here or afterwards
// allocates. Storage is untyped because the handle keeps the ring opaque, and because typed
// pointer storage would be checked against a ring that is not built yet.
func New(storage unsafe.Pointer, input New_Input) (writer Writer) {
  defer func() { Writer_Invariants(&writer, "new.writer") }()
  New_Input_Invariants(&input, "new.input")
  aver.Always(storage != nil, "diode: New needs caller storage")
  state := (*Writer_State)(storage)
  limit := input.Rate_Limit
  if limit.Bytes_Per_Second > 0 {
    if !(limit.Burst > 0) {
      limit.Burst = Burst_Size(limit.Bytes_Per_Second)
    }
  }
  *state = Writer_State{
    Stream:      input.Stream,
    Clock:       input.Clock,
    Alerter:     input.Alerter,
    Rate_Limit:  limit,
    Tokens:      Token_Count(limit.Burst),
    Last_Refill: time.Clock_Now_Monotonic(input.Clock),
    Buckets:     input.Buckets,
    Slots:       input.Slots,
  }
  // Slots may carry a previous ring's pointers; a stale non-nil slot would be drained.
  for slot_index := range state.Slots {
    state.Slots[slot_index] = nil
  }
  // Every bucket starts free. Pushing in descending order leaves bucket one on top, which
  // keeps the first lines in the lowest storage; nothing depends on it.
  for bucket_count := len(state.Buckets); bucket_count > 0; bucket_count-- {
    bucket := &state.Buckets[bucket_count-1]
    low := (bucket_count - 1) * LINE_BYTES_MAXIMUM
    bucket.Data = Line_Bytes(input.Arena[low : low+LINE_BYTES_MAXIMUM])
    bucket.Size = 0
    bucket.Sequence = 0
    bucket.Index = Bucket_Index(bucket_count)
    bucket_free(unsafe.Pointer(state), unsafe.Pointer(bucket))
  }
  // Start the write cursor one before zero so the first atomic Add yields 0,
  // matching the read cursor's start and keeping slot math symmetric.
  state.Write_Index = Write_Cursor(SEQUENCE_MAXIMUM)
  return Writer{State: storage}
}

// Writer_To_Stream exposes the diode as an nbio.Stream. A Write retires inline once its bytes
// are queued; Writer_Rearm decides when they actually leave.
func Writer_To_Stream(writer Writer_Pointer) (stream nbio.Stream) {
  Writer_Pointer_Invariants(writer, "Writer_To_Stream.writer")
  aver.Always(writer != nil, "diode: writer must not be nil")
  // The literal captures nothing, so it is a static function value and allocates nothing at
  // runtime (Test_Dropping_Does_Not_Allocate proves it), though escape analysis still
  // reports it. It exists only to give nbio its own procedure shape while writer_procedure
  // keeps typed boundaries: a named function with these raw nbio parameter types would fail
  // lint, a generic one is banned, and an alias for each type is rejected by aver.
  return nbio.Stream{
    State: writer,
    Procedure: func(
      state nbio.State,
      completion *nbio.Completion,
      mode nbio.Stream_Mode,
      buffer []byte,
      _ int64,
      _ nbio.Seek_From,
      callback nbio.Stream_Callback,
    ) {
      writer_procedure(state, completion, mode, Data(buffer), callback)
    },
  }
}

// Writer_Rearm is the root's one pump step: with nothing in flight it submits the next queued
// line to the inner Stream, or the pending Close once the ring is empty, and reports whether it
// submitted anything. The root loops it until false. It continues nothing on its own, so an
// immediate inner Stream retires inside this call and the next call submits the next line.
func Writer_Rearm(writer Writer_Pointer) (submitted Boolean) {
  defer func() { Boolean_Invariants(submitted, "Writer_Rearm.submitted") }()
  Writer_Pointer_Invariants(writer, "Writer_Rearm.writer")
  state := (*Writer_State)(writer.State)
  if state.In_Flight != BUCKET_INDEX_NONE {
    return false
  }
  if state.Close == CLOSE_SUBMITTED {
    return false
  }
  // Each pass takes one entry, so the ring bounds the loop; a shed line moves to the next.
  for pass_index := 0; pass_index < len(state.Slots); pass_index++ {
    item_pointer, ok := ring_try_next(unsafe.Pointer(state))
    if !ok {
      return drain_close(unsafe.Pointer(state))
    }
    item := (*Bucket)(item_pointer)
    if rate_limit_sheds(unsafe.Pointer(state), item.Size) {
      alert(unsafe.Pointer(state), 1, DROP_RATE_LIMIT)
      bucket_free(unsafe.Pointer(state), item_pointer)
      continue
    }
    state.In_Flight = item.Index
    nbio.Write(state.Stream, &state.Completion, item.Data[:item.Size], drain_retire)
    return true
  }
  return drain_close(unsafe.Pointer(state))
}

// Writer_Work_Queued reports whether the diode still has work outstanding: a line in flight,
// lines in the ring, or a Close waiting for the ring to empty. The root pumps until it is
// false; it never takes anything from the ring.
func Writer_Work_Queued(writer Writer_Pointer) (queued Boolean) {
  defer func() { Boolean_Invariants(queued, "Writer_Work_Queued.queued") }()
  Writer_Pointer_Invariants(writer, "Writer_Work_Queued.writer")
  state := (*Writer_State)(writer.State)
  if state.In_Flight != BUCKET_INDEX_NONE {
    return true
  }
  if state.Close == CLOSE_SUBMITTED {
    return false
  }
  slot := uint64(state.Read_Index) % uint64(len(state.Slots))
  if atomic.LoadPointer(&state.Slots[slot]) != nil {
    return true
  }
  return atomic.LoadUint32((*uint32)(&state.Close_Flag)) == uint32(CLOSE_FLAG_PUBLISHED)
}

// Pops a free bucket, or BUCKET_INDEX_NONE when BUCKET_SLACK producers already hold one
// mid-Write. The tag in the high bits makes the compare-and-swap safe against a bucket popped,
// pushed, and popped again between this load and this swap.
func bucket_take(state_pointer unsafe.Pointer) (item_pointer unsafe.Pointer) {
  state := (*Writer_State)(state_pointer)
  for retry_index := 0; retry_index < FREE_LIST_RETRIES_MAX; retry_index++ {
    head := atomic.LoadUint64((*uint64)(&state.Free))
    index := head & FREE_LIST_INDEX_MASK
    if index == BUCKET_INDEX_NONE {
      return nil
    }
    item := &state.Buckets[index-1]
    next := uint64(atomic.LoadUint32((*uint32)(&item.Next)))
    tag := head>>FREE_LIST_INDEX_BITS + 1
    if atomic.CompareAndSwapUint64(
      (*uint64)(&state.Free), head, tag<<FREE_LIST_INDEX_BITS|next,
    ) {
      return unsafe.Pointer(item)
    }
  }
  return nil
}

// Pushes a bucket back onto the free list. A successful overwrite may push its victim because
// the CAS over a non-nil slot proves the drain never took that bucket — a drain read would
// have nil'd the slot and failed the CAS.
func bucket_free(state_pointer unsafe.Pointer, item_pointer unsafe.Pointer) {
  state := (*Writer_State)(state_pointer)
  item := (*Bucket)(item_pointer)
  aver.Always(item != nil, "diode: freed a bucket that does not exist")
  index := uint64(item.Index)
  pushed := false
  for retry_index := 0; retry_index < FREE_LIST_RETRIES_MAX; retry_index++ {
    head := atomic.LoadUint64((*uint64)(&state.Free))
    atomic.StoreUint32((*uint32)(&item.Next), uint32(head&FREE_LIST_INDEX_MASK))
    tag := head>>FREE_LIST_INDEX_BITS + 1
    if atomic.CompareAndSwapUint64(
      (*uint64)(&state.Free), head, tag<<FREE_LIST_INDEX_BITS|index,
    ) {
      pushed = true
      break
    }
  }
  aver.Always(pushed, "diode: free list push exhausted its retries")
}

// The diode has no cursor, so the offset and whence nbio carries never reach here.
func writer_procedure(
  state_value nbio.State,
  completion nbio.Completion_Handle,
  mode nbio.Stream_Mode,
  data Data,
  callback nbio.Stream_Callback,
) {
  Data_Invariants(data, "writer_procedure.data")
  state := (*Writer_State)(state_value.(Writer_Pointer).State)
  completion.Data = 0
  completion.Error = nil
  switch mode {
  case nbio.STREAM_MODE_WRITE:
    written, write_err := writer_write(unsafe.Pointer(state), data)
    completion.Data = int(written)
    completion.Error = write_err
  case nbio.STREAM_MODE_FLUSH:
  case nbio.STREAM_MODE_CLOSE, nbio.STREAM_MODE_DESTROY:
    if writer_close(unsafe.Pointer(state), completion, callback) {
      return
    }
  case nbio.STREAM_MODE_QUERY:
    completion.Data = int(WRITER_MODES)
  default:
    completion.Error = nbio.Stream_Empty
  }
  nbio.Stream_Callback_Call(callback, completion)
}

// The copy is mandatory: callers (jlog) reuse data the moment Write retires.
func writer_write(state_pointer unsafe.Pointer, data Data) (written Data_Size, err error) {
  defer func() { Data_Size_Invariants(written, "writer_write.written") }()
  Data_Invariants(data, "writer_write.data")
  state := (*Writer_State)(state_pointer)
  if atomic.LoadUint32((*uint32)(&state.Close_Flag)) != uint32(CLOSE_FLAG_CLEAR) {
    return 0, nbio.Stream_Empty
  }
  if len(data) > LINE_BYTES_MAXIMUM {
    return 0, ERR_DATA_TOO_LARGE
  }
  item_pointer := bucket_take(state_pointer)
  if item_pointer == nil {
    return 0, ERR_NO_BUCKET
  }
  item := (*Bucket)(item_pointer)
  Line_Bytes_Invariants(item.Data, "writer_write.item")
  copied := copy(item.Data, data)
  item.Size = Line_Size(copied)
  aver.Always(copied == len(data), "diode: write copy length mismatch")
  ring_set(state_pointer, unsafe.Pointer(item))
  return Data_Size(copied), nil
}

// Reports owned when Close takes the caller's retirement, which Writer_Rearm delivers once the
// ring is empty; a repeated Close retires inline because Close is idempotent.
func writer_close(
  state_pointer unsafe.Pointer,
  completion nbio.Completion_Handle,
  callback nbio.Stream_Callback,
) (owned Boolean) {
  defer func() { Boolean_Invariants(owned, "writer_close.owned") }()
  state := (*Writer_State)(state_pointer)
  claimed := atomic.CompareAndSwapUint32(
    (*uint32)(&state.Close_Flag),
    uint32(CLOSE_FLAG_CLEAR),
    uint32(CLOSE_FLAG_CLAIMED),
  )
  if !claimed {
    return false
  }
  state.Close_Completion = completion
  state.Close_Callback = callback
  atomic.StoreUint32((*uint32)(&state.Close_Flag), uint32(CLOSE_FLAG_PUBLISHED))
  return true
}

// Stores item in the next ring slot, advancing the shared write cursor atomically so
// concurrent producers never share a slot.
func ring_set(state_pointer unsafe.Pointer, item_pointer unsafe.Pointer) {
  state := (*Writer_State)(state_pointer)
  item := (*Bucket)(item_pointer)
  stored := false
  for !stored {
    index := atomic.AddUint64((*uint64)(&state.Write_Index), 1)
    atomic.StoreUint64((*uint64)(&item.Sequence), index)
    slot := index % uint64(len(state.Slots))
    previous := atomic.LoadPointer(&state.Slots[slot])
    if previous != nil {
      occupant := (*Bucket)(previous)
      lap_start := index - uint64(len(state.Slots))
      if atomic.LoadUint64((*uint64)(&occupant.Sequence)) > lap_start {
        // The slot holds an unread bucket from this same lap.
        continue
      }
    }
    stored = atomic.CompareAndSwapPointer(
      &state.Slots[slot], previous, unsafe.Pointer(item),
    )
    if stored {
      aver.Always(
        previous != unsafe.Pointer(item), "diode: recycled a live bucket",
      )
      if previous != nil {
        bucket_free(state_pointer, previous)
      }
    }
    // A failed CAS means another producer won this slot; loop to the next index.
  }
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
  sequence := atomic.LoadUint64((*uint64)(&taken.Sequence))
  if sequence < read_index {
    // A stale value from a slot already fast-forwarded past; ignore it.
    return nil, false
  }
  if sequence > read_index {
    dropped := sequence - read_index
    state.Read_Index = Sequence(sequence)
    missed := Missed_Count(MISSED_COUNT_MAXIMUM)
    if !(dropped > uint64(MISSED_COUNT_MAXIMUM)) {
      missed = Missed_Count(dropped)
    }
    alert(state_pointer, missed, DROP_OVERFLOW)
  }
  aver.Always(
    sequence == uint64(state.Read_Index), "diode: delivered off the read cursor",
  )
  state.Read_Index++
  return unsafe.Pointer(taken), true
}

// Inner Stream failures have no reporting channel above a diode, so retirement only recycles
// the bucket. It records state and continues nothing: the root's next Writer_Rearm submits.
func drain_retire(completion nbio.Completion_Handle) {
  state := (*Writer_State)(unsafe.Pointer(completion))
  bucket_free(unsafe.Pointer(state), unsafe.Pointer(&state.Buckets[state.In_Flight-1]))
  state.In_Flight = BUCKET_INDEX_NONE
}

// Sends the caller's Close to the inner Stream once the ring has drained.
func drain_close(state_pointer unsafe.Pointer) (submitted Boolean) {
  defer func() { Boolean_Invariants(submitted, "drain_close.submitted") }()
  state := (*Writer_State)(state_pointer)
  if atomic.LoadUint32((*uint32)(&state.Close_Flag)) != uint32(CLOSE_FLAG_PUBLISHED) {
    return false
  }
  state.Close = CLOSE_SUBMITTED
  inner := state.Stream
  if inner.Procedure == nil {
    state.Close_Completion.Data = 0
    state.Close_Completion.Error = nil
    nbio.Stream_Callback_Call(state.Close_Callback, state.Close_Completion)
    return true
  }
  inner.Procedure(
    inner.State,
    state.Close_Completion,
    nbio.STREAM_MODE_CLOSE,
    nil,
    0,
    nbio.SEEK_FROM_START,
    state.Close_Callback,
  )
  return true
}

// Reports whether the rate limiter sheds a line of size bytes. It refills the byte token
// bucket from the clock, caps it at the burst, and a non-positive balance means shed (the
// line is dropped, never delayed). A disabled limiter (Bytes_Per_Second <= 0) never sheds.
// Only the drain owner calls this, so the bucket state needs no synchronization.
func rate_limit_sheds(state_pointer unsafe.Pointer, size Line_Size) (shed Boolean) {
  defer func() { Boolean_Invariants(shed, "rate_limit_sheds.shed") }()
  Line_Size_Invariants(size, "rate_limit_sheds.size")
  state := (*Writer_State)(state_pointer)
  rate := int64(state.Rate_Limit.Bytes_Per_Second)
  if !(rate > 0) {
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
  if !(state.Tokens > 0) {
    return true
  }
  state.Tokens -= Token_Count(size)
  return false
}

func alert(state_pointer unsafe.Pointer, missed Missed_Count, cause Drop_Cause) {
  Missed_Count_Invariants(missed, "alert.missed")
  Drop_Cause_Invariants(cause, "alert.cause")
  state := (*Writer_State)(state_pointer)
  // A nil Alerter is the caller's no-op; a substitute func literal would escape.
  if state.Alerter == nil {
    return
  }
  state.Alerter(int(missed), cause)
}
