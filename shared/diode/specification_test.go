package diode_test

import (
	"encoding/binary"
	"io"
	"strconv"
	"sync"
	"testing"

	"github.com/james-orcales/james-orcales/shared/diode"
	jtime "github.com/james-orcales/james-orcales/shared/time"
)

// Test_Write_Forwards_To_Sink checks that a written line reaches the wrapped sink.
func Test_Write_Forwards_To_Sink(t *testing.T) {
	sink := &recording_sink{Written: make(chan string, 4)}
	writer := diode.New(diode.New_Input{Writer: sink, Clock: instant_clock(), Count: 8})
	writer.Write([]byte("hello"))
	if got := <-sink.Written; got != "hello" {
		t.Fatalf("forwarded %q, want hello", got)
	}
	writer.Close()
}

// Test_Write_Does_Not_Block checks that Write returns even while the sink is stuck; a
// synchronous wrapper would deadlock here and the test would time out.
func Test_Write_Does_Not_Block(t *testing.T) {
	sink := &blocking_sink{Release: make(chan struct{})}
	writer := diode.New(diode.New_Input{Writer: sink, Clock: instant_clock(), Count: 4})
	for index := 0; index < 16; index++ {
		n, err := writer.Write([]byte("x"))
		if n != 1 {
			t.Fatalf("write %d returned n=%d, want 1", index, n)
		}
		if err != nil {
			t.Fatalf("write %d returned err=%v, want nil", index, err)
		}
	}
	close(sink.Release)
	writer.Close()
}

// Test_Overflow_Drops_Oldest checks that overrunning capacity keeps the newest
// entries and discards the oldest.
func Test_Overflow_Drops_Oldest(t *testing.T) {
	sink := &recording_sink{Written: make(chan string, 8)}
	clock, parked, resume := gated_clock()
	writer := diode.New(diode.New_Input{Writer: sink, Clock: clock, Count: 4})
	<-parked
	for index := 0; index < 8; index++ {
		writer.Write([]byte(decimal(index)))
	}
	close(resume)
	for _, want := range []string{"4", "5", "6", "7"} {
		if got := <-sink.Written; got != want {
			t.Fatalf("survivor %q, want %q", got, want)
		}
	}
	writer.Close()
}

// Test_Drop_Count_Is_Reported checks that the Alerter receives the number of
// overwritten entries when the writer laps the drain.
func Test_Drop_Count_Is_Reported(t *testing.T) {
	dropped := make(chan int, 8)
	causes := make(chan diode.Drop_Cause, 8)
	sink := &recording_sink{Written: make(chan string, 8)}
	clock, parked, resume := gated_clock()
	writer := diode.New(diode.New_Input{
		Writer: sink,
		Clock:  clock,
		Count:  4,
		Alerter: func(missed int, cause diode.Drop_Cause) {
			dropped <- missed
			causes <- cause
		},
	})
	<-parked
	for index := 0; index < 8; index++ {
		writer.Write([]byte(decimal(index)))
	}
	close(resume)
	if missed := <-dropped; missed != 4 {
		t.Fatalf("alerter reported %d drops, want 4", missed)
	}
	if cause := <-causes; cause != diode.Drop_Overflow {
		t.Fatalf("drop cause %v, want Drop_Overflow", cause)
	}
	writer.Close()
}

// Test_Order_Is_Preserved checks that delivered entries keep write order.
func Test_Order_Is_Preserved(t *testing.T) {
	sink := &recording_sink{Written: make(chan string, 16)}
	writer := diode.New(diode.New_Input{Writer: sink, Clock: instant_clock(), Count: 16})
	for index := 0; index < 10; index++ {
		writer.Write([]byte(decimal(index)))
	}
	for index := 0; index < 10; index++ {
		if got := <-sink.Written; got != decimal(index) {
			t.Fatalf("position %d = %q, want %q", index, got, decimal(index))
		}
	}
	writer.Close()
}

// Test_Poll_Interval_Is_Configurable checks the empty-ring sleep duration: the
// default is one hundred milliseconds, and an explicit interval is honored.
func Test_Poll_Interval_Is_Configurable(t *testing.T) {
	if observed := capture_interval(t, 0); observed != 100*jtime.Millisecond {
		t.Fatalf("default interval %d, want %d", observed, 100*jtime.Millisecond)
	}
	observed := capture_interval(t, 250*jtime.Millisecond)
	if observed != 250*jtime.Millisecond {
		t.Fatalf("custom interval %d, want %d", observed, 250*jtime.Millisecond)
	}
}

// Test_Close_Flushes_And_Stops checks that entries still buffered at Close reach the
// sink and that Close then returns (the drain goroutine has exited).
func Test_Close_Flushes_And_Stops(t *testing.T) {
	sink := &recording_sink{Written: make(chan string, 4)}
	clock, parked, resume := gated_clock()
	writer := diode.New(diode.New_Input{Writer: sink, Clock: clock, Count: 8})
	<-parked
	for index := 0; index < 3; index++ {
		writer.Write([]byte(decimal(index)))
	}
	closed := make(chan struct{})
	go func() {
		writer.Close()
		close(closed)
	}()
	close(resume)
	for index := 0; index < 3; index++ {
		if got := <-sink.Written; got != decimal(index) {
			t.Fatalf("flushed position %d = %q, want %q", index, got, decimal(index))
		}
	}
	<-closed
}

// Test_Dropping_Does_Not_Allocate checks that overwriting unread entries recycles
// their buckets, so a diode shedding load allocates nothing per dropped line.
func Test_Dropping_Does_Not_Allocate(t *testing.T) {
	clock, parked, resume := gated_clock()
	writer := diode.New(diode.New_Input{Writer: io.Discard, Clock: clock, Count: 8})
	<-parked
	line := []byte("a dropped line")
	// Warm past one full lap so the pool reaches steady-state recycling before the
	// measured runs; the parked drain consumes nothing, so every write here drops.
	for index := 0; index < 64; index++ {
		writer.Write(line)
	}
	allocs := testing.AllocsPerRun(1000, func() {
		writer.Write(line)
	})
	if allocs != 0 {
		t.Fatalf("dropping allocated %.2f per write, want 0", allocs)
	}
	close(resume)
	writer.Close()
}

// Test_Rate_Limit_Sheds_By_Bytes checks the byte budget: up to the burst is delivered and
// the rest is shed with cause Drop_Rate_Limit, while a clock that advances refills tokens so
// every line passes.
func Test_Rate_Limit_Sheds_By_Bytes(t *testing.T) {
	sink := &recording_sink{Written: make(chan string, 16)}
	causes := make(chan diode.Drop_Cause, 16)
	writer := diode.New(diode.New_Input{
		Writer:     sink,
		Clock:      instant_clock(),
		Count:      16,
		Rate_Limit: diode.Rate_Limit{Bytes_Per_Second: 1, Burst: 6},
		Alerter: func(missed int, cause diode.Drop_Cause) {
			for index := 0; index < missed; index++ {
				causes <- cause
			}
		},
	})
	for index := 0; index < 8; index++ {
		writer.Write([]byte("ab"))
	}
	for index := 0; index < 3; index++ {
		if got := <-sink.Written; got != "ab" {
			t.Fatalf("delivered %q, want ab", got)
		}
	}
	for index := 0; index < 5; index++ {
		if cause := <-causes; cause != diode.Drop_Rate_Limit {
			t.Fatalf("shed cause %v, want Drop_Rate_Limit", cause)
		}
	}
	writer.Close()

	steady_sink := &recording_sink{Written: make(chan string, 16)}
	steady := diode.New(diode.New_Input{
		Writer:     steady_sink,
		Clock:      stepping_clock(jtime.Second),
		Count:      16,
		Rate_Limit: diode.Rate_Limit{Bytes_Per_Second: 1000, Burst: 2},
		Alerter: func(missed int, cause diode.Drop_Cause) {
			t.Errorf("unexpected drop of %d (%v); tokens should refill", missed, cause)
		},
	})
	for index := 0; index < 8; index++ {
		steady.Write([]byte("ab"))
	}
	for index := 0; index < 8; index++ {
		if got := <-steady_sink.Written; got != "ab" {
			t.Fatalf("steady delivery %q, want ab", got)
		}
	}
	steady.Close()
}

// Test_Rate_Limit_Survives_A_Large_Clock guards the refill math against int64 overflow: a real
// monotonic clock reads large (nanoseconds since boot) and can gap by hours when idle, which a
// naive elapsed*rate would overflow into a negative balance that then sheds every line.
func Test_Rate_Limit_Survives_A_Large_Clock(t *testing.T) {
	sink := &recording_sink{Written: make(chan string, 16)}
	writer := diode.New(diode.New_Input{
		Writer:     sink,
		Clock:      stepping_clock(10_000 * jtime.Second),
		Count:      16,
		Rate_Limit: diode.Rate_Limit{Bytes_Per_Second: 1 << 20, Burst: 1 << 20},
		Alerter: func(missed int, cause diode.Drop_Cause) {
			t.Errorf("unexpected drop of %d (%v) under a large clock", missed, cause)
		},
	})
	for index := 0; index < 4; index++ {
		writer.Write([]byte("ab"))
	}
	for index := 0; index < 4; index++ {
		if got := <-sink.Written; got != "ab" {
			t.Fatalf("large-clock delivery %q, want ab", got)
		}
	}
	writer.Close()
}

// A sink that forwards every line the drain writes onto Written so a test can observe
// deliveries without polling.
type recording_sink struct {
	// Written receives one copy of each forwarded line.
	Written chan string
}

func (sink *recording_sink) Write(p []byte) (n int, err error) {
	sink.Written <- string(p)
	return len(p), nil
}

// A sink that models a stuck downstream: every Write blocks until Release closes.
type blocking_sink struct {
	// Release unblocks all pending and future writes when closed.
	Release chan struct{}
}

func (sink *blocking_sink) Write(p []byte) (n int, err error) {
	<-sink.Release
	return len(p), nil
}

// A clock whose Sleep returns immediately, so the drain spins and tests synchronize on
// the sink rather than on wall-clock time.
func instant_clock() (clock jtime.Clock) {
	return jtime.Clock{
		Now_Monotonic: func() (moment jtime.Moment) { return 0 },
		Now_Realtime:  func() (moment jtime.Moment) { return 0 },
		Tick:          func() {},
		Sleep:         func(duration jtime.Duration) {},
	}
}

// A clock that parks the drain inside its first Sleep: the first call closes parked
// (so a test learns the drain is idle on an empty ring) and every call blocks until
// resume closes. After resume closes, Sleep returns immediately.
func gated_clock() (clock jtime.Clock, parked chan struct{}, resume chan struct{}) {
	parked = make(chan struct{})
	resume = make(chan struct{})
	var once sync.Once
	clock = jtime.Clock{
		Now_Monotonic: func() (moment jtime.Moment) { return 0 },
		Now_Realtime:  func() (moment jtime.Moment) { return 0 },
		Tick:          func() {},
		Sleep: func(duration jtime.Duration) {
			once.Do(func() { close(parked) })
			<-resume
		},
	}
	return clock, parked, resume
}

// A clock whose monotonic reading advances by step on every read, so a test can drive the
// rate limiter's token refill deterministically; Sleep returns at once like instant_clock.
// Only the single drain goroutine reads Now_Monotonic, so the captured counter needs no
// synchronization.
func stepping_clock(step jtime.Duration) (clock jtime.Clock) {
	elapsed := int64(0)
	return jtime.Clock{
		Now_Monotonic: func() (moment jtime.Moment) {
			elapsed += int64(step)
			return jtime.Moment(elapsed)
		},
		Now_Realtime: func() (moment jtime.Moment) { return 0 },
		Tick:         func() {},
		Sleep:        func(duration jtime.Duration) {},
	}
}

// Builds a diode with the configured interval and returns the duration the drain
// actually passed to Sleep on its first empty poll.
func capture_interval(t *testing.T, configured jtime.Duration) (observed jtime.Duration) {
	t.Helper()
	intervals := make(chan jtime.Duration, 1)
	clock := jtime.Clock{
		Now_Monotonic: func() (moment jtime.Moment) { return 0 },
		Now_Realtime:  func() (moment jtime.Moment) { return 0 },
		Tick:          func() {},
		Sleep: func(duration jtime.Duration) {
			select {
			case intervals <- duration:
			default:
			}
		},
	}
	writer := diode.New(diode.New_Input{
		Writer:        io.Discard,
		Clock:         clock,
		Poll_Interval: configured,
	})
	observed = <-intervals
	writer.Close()
	return observed
}

// Renders index as its base-ten string, a stable per-entry marker for tests.
func decimal(index int) (text string) {
	return strconv.Itoa(index)
}

// Matches the realistic message length used across the repo's logging benchmarks so
// diode numbers compare against jlog's.
const benchmark_line = "Test logging, but use a somewhat realistic message length."

// Benchmark_Write measures the producer cost with a roomy ring the drain can keep up
// with, so it reflects the steady-state, no-drop path.
func Benchmark_Write(b *testing.B) {
	writer := diode.New(diode.New_Input{
		Writer: io.Discard, Clock: instant_clock(), Count: 1024,
	})
	line := []byte(benchmark_line)
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			writer.Write(line)
		}
	})
	b.StopTimer()
	writer.Close()
}

// Benchmark_Write_Serial measures the single-producer cost, isolating it from the
// atomic-counter contention that the parallel benchmark adds.
func Benchmark_Write_Serial(b *testing.B) {
	writer := diode.New(diode.New_Input{
		Writer: io.Discard, Clock: instant_clock(), Count: 1024,
	})
	line := []byte(benchmark_line)
	b.ReportAllocs()
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		writer.Write(line)
	}
	b.StopTimer()
	writer.Close()
}

// Benchmark_Write_Full_Ring measures the producer cost when a tiny ring is saturated
// and dropping, the worst case for collision retries and bucket churn.
func Benchmark_Write_Full_Ring(b *testing.B) {
	writer := diode.New(diode.New_Input{Writer: io.Discard, Clock: instant_clock(), Count: 8})
	line := []byte(benchmark_line)
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			writer.Write(line)
		}
	})
	b.StopTimer()
	writer.Close()
}

// The diode's sink during fuzzing: it decodes each delivered line back to its
// (producer, sequence) and records the first invariant it sees broken. Only the drain
// goroutine touches it, and the test reads it after Close.
type ring_witness struct {
	// Producers is how many concurrent writers this run spawned.
	Producers int
	// Writes is how many lines each producer emits.
	Writes int
	// Last is the highest sequence delivered per producer, -1 before any.
	Last []int
	// Delivered counts the lines forwarded to this sink.
	Delivered int
	// Fault holds the first broken invariant observed, empty when clean.
	Fault string
}

func (witness *ring_witness) Write(p []byte) (n int, err error) {
	if len(p) != 8 {
		note_fault(witness, "torn line length")
		return len(p), nil
	}
	producer := int(binary.LittleEndian.Uint32(p[0:4]))
	sequence := int(binary.LittleEndian.Uint32(p[4:8]))
	if producer >= witness.Producers {
		note_fault(witness, "producer id out of range")
		return len(p), nil
	}
	if sequence >= witness.Writes {
		note_fault(witness, "sequence out of range")
		return len(p), nil
	}
	if sequence <= witness.Last[producer] {
		note_fault(witness, "duplicate or out-of-order delivery")
		return len(p), nil
	}
	witness.Last[producer] = sequence
	witness.Delivered++
	return len(p), nil
}

// Records the first broken invariant; later ones stay quiet so the failure the test
// reports is the earliest and most diagnostic.
func note_fault(witness *ring_witness, fault string) {
	if witness.Fault != "" {
		return
	}
	witness.Fault = fault
}

// Fuzz_Ring drives many producers into a tiny ring so overwrites, collision retries, and
// the recycle path all fire under contention. The invariants: every delivered line
// decodes to a real (producer, sequence), and each producer's deliveries are strictly
// increasing — both break if a recycled bucket is ever handed out twice. Run it with
// -race and -fuzz to explore parameters and interleavings; the seed corpus also runs as
// an ordinary concurrent regression test.
func Fuzz_Ring(f *testing.F) {
	f.Add(uint8(0), uint16(3), uint32(400))
	f.Add(uint8(7), uint16(1), uint32(900))
	f.Fuzz(func(t *testing.T, ring_byte uint8, producer_word uint16, write_word uint32) {
		count := 1 + int(ring_byte%8)
		producer_count := 1 + int(producer_word%8)
		writes := 1 + int(write_word%256)
		witness := &ring_witness{
			Producers: producer_count,
			Writes:    writes,
			Last:      make([]int, producer_count),
		}
		for index := range witness.Last {
			witness.Last[index] = -1
		}
		writer := diode.New(diode.New_Input{
			Writer:  witness,
			Clock:   instant_clock(),
			Count:   count,
			Alerter: func(missed int, cause diode.Drop_Cause) {},
		})
		var crew sync.WaitGroup
		for producer_index := 0; producer_index < producer_count; producer_index++ {
			crew.Add(1)
			go func(id int) {
				defer crew.Done()
				line := make([]byte, 8)
				binary.LittleEndian.PutUint32(line[0:4], uint32(id))
				for sequence_index := 0; sequence_index < writes; sequence_index++ {
					stamp := uint32(sequence_index)
					binary.LittleEndian.PutUint32(line[4:8], stamp)
					writer.Write(line)
				}
			}(producer_index)
		}
		crew.Wait()
		writer.Close()
		if witness.Fault != "" {
			t.Fatalf("%s (count=%d producers=%d writes=%d)",
				witness.Fault, count, producer_count, writes)
		}
		if witness.Delivered > producer_count*writes {
			t.Fatalf("delivered %d exceeds %d written",
				witness.Delivered, producer_count*writes)
		}
	})
}
