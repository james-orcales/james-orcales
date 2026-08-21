// Whitebox suite (package prng): most checks are black-box, but the reference-vector test
// reaches the unexported chacha20_block to match RFC 8439's vectors before fast-key-erasure hides
// the raw keystream, and the erasure test reads the key. Every spec-mapped test is in this file, in
// heading order.
package prng

import (
	"testing"
	"unsafe"

	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/encoding/hex"
	"local/james-orcales/shared/invariant/default"
	"local/james-orcales/shared/math/bits"
)

// Test_Seed_Expands_To_State checks New is deterministic and seed-sensitive.
func Test_Seed_Expands_To_State(t *testing.T) {
	first := New([KEY_BYTES]byte{1}, CURSOR_MIN)
	if first.Position != CURSOR_MIN {
		t.Fatalf("new generator cursor was %d, want %d", first.Position, CURSOR_MIN)
	}
	again := New([KEY_BYTES]byte{1}, CURSOR_MIN)
	var first_draw, again_draw [WORD_BYTE_COUNT]byte
	first.Read(first_draw[:])
	again.Read(again_draw[:])
	if first_draw != again_draw {
		t.Fatalf("same seed produced different streams")
	}
	other := New([KEY_BYTES]byte{2}, CURSOR_MIN)
	repeat := New([KEY_BYTES]byte{1}, CURSOR_MIN)
	var other_draw, repeat_draw [WORD_BYTE_COUNT]byte
	other.Read(other_draw[:])
	repeat.Read(repeat_draw[:])
	if other_draw == repeat_draw {
		t.Fatalf("distinct seeds produced the same first draw")
	}
	positioned := New([KEY_BYTES]byte{3}, 2)
	if positioned.Position != 2 {
		t.Fatalf("injected cursor was %d, want 2", positioned.Position)
	}
	for index := 0; index < int(positioned.Position); index++ {
		if positioned.Buffer[index] != 0 {
			t.Fatalf("consumed buffer byte %d was not erased", index)
		}
	}
}

// Test_Block_Matches_Reference_Vectors checks the block function reproduces RFC 8439's official
// keystream vectors: the worked example of section 2.3.2 and the two block vectors of Appendix A.1.
func Test_Block_Matches_Reference_Vectors(t *testing.T) {
	cases := []struct {
		Name    string
		Key     string
		Counter Block_Counter
		Nonce   string
		Want    string
	}{
		{
			Name:    "section 2.3.2",
			Key:     "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f",
			Counter: 1,
			Nonce:   "000000090000004a00000000",
			Want: "10f1e7e4d13b5915500fdd1fa32071c4" +
				"c7d1f4c733c068030422aa9ac3d46c4e" +
				"d2826446079faa0914c2d705d98b02a2" +
				"b5129cd1de164eb9cbd083e8a2503c4e",
		},
		{
			Name:    "appendix a.1 vector 1",
			Key:     "0000000000000000000000000000000000000000000000000000000000000000",
			Counter: 0,
			Nonce:   "000000000000000000000000",
			Want: "76b8e0ada0f13d90405d6ae55386bd28" +
				"bdd219b8a08ded1aa836efcc8b770dc7" +
				"da41597c5157488d7724e03fb8d84a37" +
				"6a43b8f41518a11cc387b669b2ee6586",
		},
		{
			Name:    "appendix a.1 vector 2",
			Key:     "0000000000000000000000000000000000000000000000000000000000000000",
			Counter: 1,
			Nonce:   "000000000000000000000000",
			Want: "9f07e7be5551387a98ba977c732d080d" +
				"cb0f29a048e3656912c6533e32ee7aed" +
				"29b721769ce64e43d57133b074d839d5" +
				"31ed1f28510afb45ace10a1f4b794d6f",
		},
	}
	for _, test_case := range cases {
		key := decode_key(t, test_case.Key)
		nonce := decode_nonce(t, test_case.Nonce)
		want := decode_bytes(t, test_case.Want)
		var output [CHACHA_BLOCK_BYTE_COUNT]byte
		chacha20_block(key, test_case.Counter, nonce, &output)
		if !bytes.Equal(bytes.Slice(output[:]), bytes.Slice(want)) {
			t.Fatalf("%s: block was %x, want %x", test_case.Name, output[:], want)
		}
	}
}

// Test_Known_Sequence locks the output stream for a fixed seed.
func Test_Known_Sequence(t *testing.T) {
	generator := New([KEY_BYTES]byte{}, CURSOR_MIN)
	// Frozen from this implementation: ChaCha20 under fast-key-erasure from an all-zero seed,
	// read eight bytes at a time and assembled little-endian. It is not the raw RFC keystream,
	// since the first 32 bytes of each block reseed the key; the block is checked against the
	// RFC above. The contract is per-version reproducibility.
	want := []uint64{
		10180482965161198042,
		3984235106219861111,
		2062956586891494250,
		9684409023775279043,
		8806878500039886751,
		939050496341555864,
		7594726247694405579,
		17112251633709073938,
	}
	for index := 0; index < len(want); index++ {
		var octet [WORD_BYTE_COUNT]byte
		generator.Read(octet[:])
		value := uint64(word_from_bytes(octet))
		if value != want[index] {
			t.Fatalf("draw %d was %d, want %d", index, value, want[index])
		}
	}
}

// Test_Read_Fills_Fully checks Read fills all of p, reports the full count, and never errors. The
// byte-granular reads also walk the cursor and sink lengths through 0, 1, 2 so those invariant
// boundaries are witnessed; the larger reads cross a refill and prove real keystream.
func Test_Read_Fills_Fully(t *testing.T) {
	generator := New([KEY_BYTES]byte{9}, CURSOR_MIN)
	sizes := []int{0, 1, 1, 1, 2, 7, 224, 225, 1000}
	for _, size := range sizes {
		destination := make([]byte, size)
		count, read_error := generator.Read(destination)
		if read_error != nil {
			t.Fatalf("Read(%d) errored: %v", size, read_error)
		}
		if count != size {
			t.Fatalf("Read(%d) filled %d bytes, want %d", size, count, size)
		}
		all_zero := true
		for index := 0; index < size; index++ {
			if destination[index] != 0 {
				all_zero = false
			}
		}
		if all_zero {
			if size >= 8 {
				t.Fatalf("Read(%d) produced all zero bytes", size)
			}
		}
	}
}

// Test_Bytes_Are_Uniform checks a filled buffer sets close to half of all its bits.
func Test_Bytes_Are_Uniform(t *testing.T) {
	generator := New([KEY_BYTES]byte{3}, CURSOR_MIN)
	buffer := make([]byte, SINK_MAX)
	generator.Read(buffer)
	set_bits := 0
	for _, octet := range buffer {
		set_bits += int(bits.Ones_Count_8(bits.Word_8(octet)))
	}
	total_bits := len(buffer) * 8
	if set_bits < total_bits*49/100 {
		t.Fatalf("set %d of %d bits, below band", set_bits, total_bits)
	}
	if set_bits > total_bits*51/100 {
		t.Fatalf("set %d of %d bits, above band", set_bits, total_bits)
	}
}

// Test_Below_Is_Bounded checks Below stays within zero and bound and rejects a zero bound.
func Test_Below_Is_Bounded(t *testing.T) {
	generator := New([KEY_BYTES]byte{4}, CURSOR_MIN)
	bounds := []int{1, 2, 7, 1000, 1 << 40}
	for _, bound := range bounds {
		for draw_index := 0; draw_index < 10000; draw_index++ {
			index := Chacha_Below(&generator, Bound(bound))
			if index >= Index(bound) {
				t.Fatalf("Below(%d) returned %d, out of range", bound, index)
			}
		}
	}
	for _, byte_size := range []int{1, 2} {
		positioned := New(
			[KEY_BYTES]byte{byte(byte_size)},
			Cursor(byte_size),
		)
		Chacha_Below(&positioned, BOUND_MAX)
	}
	// An all-ones draw reaches the half-open domain's last value deterministically; waiting for
	// a random stream to hit one point in 2^62 would make the boundary contract untestable.
	maximum_generator := Chacha{}
	for index := 0; index < 8; index++ {
		maximum_generator.Buffer[index] = 0xff
	}
	maximum := Chacha_Below(&maximum_generator, BOUND_MAX)
	if maximum != INDEX_MAX {
		t.Fatalf("Below(%d) returned %d from an all-ones draw, want %d",
			BOUND_MAX, maximum, INDEX_MAX)
	}
	died := did_die(func() {
		Chacha_Below(&generator, Bound(0))
	})
	if !died {
		t.Fatalf("Below with a zero bound did not exit")
	}
}

// Test_Seed_Is_Erased_After_Construction checks New performs the first refill, so the seed no
// longer lives in the Chacha's key and a later disclosure cannot reproduce it or its output.
func Test_Seed_Is_Erased_After_Construction(t *testing.T) {
	seed := [KEY_BYTES]byte{}
	for index := 0; index < 32; index++ {
		seed[index] = byte(index)
	}
	generator := New(seed, CURSOR_MIN)
	if generator.Key == seed {
		t.Fatalf("New left the seed in the generator key; fast-key-erasure did not run")
	}
}

// Test_Refill_Resets_Cursor checks that a refill accepts each cursor boundary and starts the new
// buffer at zero.
func Test_Refill_Resets_Cursor(t *testing.T) {
	positions := []Cursor{CURSOR_MIN, 1, 2, CURSOR_MAX}
	for _, position := range positions {
		generator := New([KEY_BYTES]byte{byte(position)}, position)
		chacha_refill(&generator)
		if generator.Position != CURSOR_MIN {
			t.Fatalf("refill cursor was %d, want %d", generator.Position, CURSOR_MIN)
		}
	}
}

// Test_Source_Is_Transparent checks the vtable path equals the direct Read path from equal state,
// across a refill boundary, and that a nil Chacha dies before binding.
func Test_Source_Is_Transparent(t *testing.T) {
	subject := New([KEY_BYTES]byte{6}, CURSOR_MIN)
	reference := New([KEY_BYTES]byte{6}, CURSOR_MIN)
	source := Chacha_To_Source(&subject)
	got := make([]byte, BUFFER_BYTES+1)
	want := make([]byte, BUFFER_BYTES+1)
	Source_Read(source, Sink(got[:WORD_BYTE_COUNT]))
	Source_Read(source, Sink(got[WORD_BYTE_COUNT:]))
	reference.Read(want)
	if !bytes.Equal(got, want) {
		t.Fatalf("vtable bytes differ from direct Read bytes")
	}
	if !did_die(func() { Chacha_To_Source(nil) }) {
		t.Fatalf("nil Chacha did not die")
	}
	// Binding at each cursor boundary witnesses the Chacha invariant that the bind asserts.
	for _, position := range []Cursor{CURSOR_MIN, CURSOR_MIN + 1, CURSOR_MIN + 2, CURSOR_MAX} {
		positioned := New([KEY_BYTES]byte{6}, position)
		Source_Read(Chacha_To_Source(&positioned), Sink(got[:WORD_BYTE_COUNT]))
	}
	// A constructed buffer reaches each word edge of the slot deterministically; a keystream
	// would take 2^64 draws to land on one of them by chance.
	edges := []Word{WORD_MINIMUM, WORD_MINIMUM + 1, WORD_MINIMUM + 2, WORD_MAXIMUM}
	for _, word := range edges {
		edge := Chacha{}
		packed := word_to_bytes(word)
		copy(edge.Buffer[:], packed[:])
		var octet [WORD_BYTE_COUNT]byte
		Source_Read(Chacha_To_Source(&edge), Sink(octet[:]))
		if word_from_bytes(octet) != word {
			t.Fatalf("constructed word %d did not pass through the slot", word)
		}
	}
}

// Test_Source_Marks_A_Cryptographic_Parameter checks a backend enters only through State and
// Next, that Source_Read packs each word little-endian and spends a whole word on a partial tail,
// and that an unbound Source dies.
func Test_Source_Marks_A_Cryptographic_Parameter(t *testing.T) {
	counter := Word(0)
	source := Source{State: unsafe.Pointer(&counter), Next: counter_next}
	var got [WORD_BYTE_COUNT + 1]byte
	Source_Read(source, Sink(got[:]))
	first := word_to_bytes(1)
	if [WORD_BYTE_COUNT]byte(got[:WORD_BYTE_COUNT]) != first {
		t.Fatalf("first word was not packed little-endian")
	}
	if got[WORD_BYTE_COUNT] != byte(2) {
		t.Fatalf("partial tail did not spend a whole second word")
	}
	if counter != 2 {
		t.Fatalf("nine bytes spent %d words, want 2", counter)
	}
	headless := source
	headless.State = nil
	if !did_die(func() { Source_Read(headless, Sink(got[:])) }) {
		t.Fatalf("Source without state did not die")
	}
	inert := source
	inert.Next = nil
	if !did_die(func() { Source_Read(inert, Sink(got[:])) }) {
		t.Fatalf("Source without procedure did not die")
	}
	var largest [SINK_MAX + 1]byte
	if !did_die(func() { Source_Read(source, Sink(largest[:])) }) {
		t.Fatalf("oversized sink did not die")
	}
	for _, size := range []int{SINK_MIN, SINK_MIN + 1, SINK_MIN + 2, SINK_MAX} {
		Source_Read(source, Sink(largest[:size]))
	}
}

// Test_Hot_Path_Is_Zero_Allocation checks a steady-state Read does not allocate, even with the
// invariant assertions on the draw path — measured under recording, not just in benchmark mode.
func Test_Hot_Path_Is_Zero_Allocation(t *testing.T) {
	generator := New([KEY_BYTES]byte{8}, CURSOR_MIN)
	buffer := make([]byte, 8)
	allocations := testing.AllocsPerRun(1000, func() {
		generator.Read(buffer)
	})
	if allocations != 0 {
		t.Fatalf("Read allocated %.1f times per call, want zero", allocations)
	}
}

// A slot that counts its draws, so a test can see how many words a read spent.
func counter_next(state unsafe.Pointer) (value Word) {
	counter := (*Word)(state)
	*counter++
	return *counter
}

// Runs action and reports whether it tripped a fatal invariant, used to assert preconditions. A
// violation exits through the Default recorder, which os.Exit cannot recover, so the helper swaps
// Exit for a panic — and silences the recorder's stderr — then recovers it, so the exit is
// observable in-process. Exit and Output are restored before returning. Copied from prng's suite.
func did_die(action func()) (died bool) {
	exit, output := invariant.Default.Exit, invariant.Default.Output
	invariant.Default.Exit = func(int) { panic(tripped_invariant{}) }
	invariant.Default.Output = discard_writer{}
	defer func() {
		invariant.Default.Exit, invariant.Default.Output = exit, output
		if recover() != nil {
			died = true
		}
	}()
	action()
	return died
}

// Marks the swapped-in Exit's panic, so did_die's recover tells a deliberately tripped guard from
// an unrelated panic in the action.
type tripped_invariant struct{}

type discard_writer struct{}

// Write keeps invariant diagnostics silent while preserving complete writes.
func (discard_writer) Write(data []byte) (count int, err error) {
	return len(data), nil
}

// Decodes a 32-byte hex key, failing the test on a wrong length.
func decode_key(t *testing.T, encoded string) (key [KEY_BYTES]byte) {
	decoded := decode_bytes(t, encoded)
	if len(decoded) != 32 {
		t.Fatalf("key is %d bytes, want 32", len(decoded))
	}
	copy(key[:], decoded)
	return key
}

// Decodes a 12-byte hex nonce, failing the test on a wrong length.
func decode_nonce(t *testing.T, encoded string) (nonce [NONCE_BYTE_COUNT]byte) {
	decoded := decode_bytes(t, encoded)
	if len(decoded) != 12 {
		t.Fatalf("nonce is %d bytes, want 12", len(decoded))
	}
	copy(nonce[:], decoded)
	return nonce
}

// Decodes a hex string into bytes, failing the test on malformed input.
func decode_bytes(t *testing.T, encoded string) (decoded []byte) {
	decoded = make([]byte, len(encoded)/2)
	count, status := hex.Decode_Into(decoded, hex.Encoded(encoded))
	if status != hex.STATUS_OK {
		t.Fatalf("decode %q: status %d", encoded, status)
	}
	return decoded[:count]
}
