// Whitebox suite (package prng): most checks are black-box, but the reference-vector test
// reaches the unexported chacha20_block to match RFC 8439's vectors before fast-key-erasure hides
// the raw keystream, and the erasure test reads the key. Every spec-mapped test is in this file, in
// heading order.
package prng

import (
	"testing"

	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/encoding/hex"
	"local/james-orcales/shared/math/bits"
)

// Test_Seed_Expands_To_State checks Chacha_Init is deterministic and seed-sensitive.
func Test_Seed_Expands_To_State(t *testing.T) {
	first := test_chacha(test_seed(1), CURSOR_MIN)
	if first.Position != CURSOR_MIN {
		t.Fatalf("new generator cursor was %d, want %d", first.Position, CURSOR_MIN)
	}
	again := test_chacha(test_seed(1), CURSOR_MIN)
	var first_draw, again_draw [WORD_BYTE_COUNT]byte
	Chacha_Read(&first, Sink(first_draw[:]))
	Chacha_Read(&again, Sink(again_draw[:]))
	if first_draw != again_draw {
		t.Fatalf("same seed produced different streams")
	}
	other := test_chacha(test_seed(2), CURSOR_MIN)
	repeat := test_chacha(test_seed(1), CURSOR_MIN)
	var other_draw, repeat_draw [WORD_BYTE_COUNT]byte
	Chacha_Read(&other, Sink(other_draw[:]))
	Chacha_Read(&repeat, Sink(repeat_draw[:]))
	if other_draw == repeat_draw {
		t.Fatalf("distinct seeds produced the same first draw")
	}
	positioned := test_chacha(test_seed(3), 2)
	if positioned.Position != 2 {
		t.Fatalf("injected cursor was %d, want 2", positioned.Position)
	}
	var positioned_buffer [BUFFER_BYTES]byte
	buffer_copy(positioned.Buffer, Buffer_Destination(positioned_buffer[:]))
	for index := 0; index < int(positioned.Position); index++ {
		if positioned_buffer[index] != 0 {
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
		chacha20_block(key, test_case.Counter, nonce, output[:])
		if !bytes.Equal(bytes.Slice(output[:]), bytes.Slice(want)) {
			t.Fatalf("%s: block was %x, want %x", test_case.Name, output[:], want)
		}
	}
}

// Test_Known_Sequence locks the output stream for a fixed seed.
func Test_Known_Sequence(t *testing.T) {
	generator := test_chacha(test_seed(0), CURSOR_MIN)
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
		Chacha_Read(&generator, Sink(octet[:]))
		value := uint64(test_word_from_bytes(octet[:]))
		if value != want[index] {
			t.Fatalf("draw %d was %d, want %d", index, value, want[index])
		}
	}
}

// Test_Chacha_Read_Fills_Fully checks Read fills all of p and reports the full count. The
// byte-granular reads also walk the cursor and sink lengths through 0, 1, 2 so those invariant
// boundaries are witnessed; the larger reads cross a refill and prove real keystream.
func Test_Chacha_Read_Fills_Fully(t *testing.T) {
	generator := test_chacha(test_seed(9), CURSOR_MIN)
	sizes := []int{0, 1, 1, 1, 2, 7, 224, 225, 1000}
	for _, size := range sizes {
		destination := make([]byte, size)
		count := Chacha_Read(&generator, Sink(destination))
		if int(count) != size {
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
	generator := test_chacha(test_seed(3), CURSOR_MIN)
	buffer := make([]byte, SINK_MAX)
	Chacha_Read(&generator, Sink(buffer))
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
	generator := test_chacha(test_seed(4), CURSOR_MIN)
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
		positioned := test_chacha(
			test_seed(byte(byte_size)),
			Cursor(byte_size),
		)
		Chacha_Below(&positioned, BOUND_MAX)
	}
	// An all-ones draw reaches the half-open domain's last value deterministically; waiting for
	// a random stream to hit one point in 2^62 would make the boundary contract untestable.
	maximum_generator := Chacha{}
	maximum_generator.Buffer.Lane_0 = Buffer_Lane_0(bits.WORD_64_MAXIMUM)
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

// Test_Seed_Is_Erased_After_Construction checks initialization performs the first refill, so the
// seed no longer lives in state and a later disclosure cannot reproduce it or prior output.
func Test_Seed_Is_Erased_After_Construction(t *testing.T) {
	seed := make(Seed, KEY_BYTES)
	for index := 0; index < 32; index++ {
		seed[index] = byte(index)
	}
	generator := test_chacha(seed, CURSOR_MIN)
	var live_seed [KEY_BYTES]byte
	key_copy(generator.Key, Key_Destination(live_seed[:]))
	if bytes.Equal(bytes.Slice(live_seed[:]), bytes.Slice(seed)) {
		t.Fatalf("New left the seed in the generator key; fast-key-erasure did not run")
	}
}

// Test_Refill_Resets_Cursor checks that a refill accepts each cursor boundary and starts the new
// buffer at zero.
func Test_Refill_Resets_Cursor(t *testing.T) {
	positions := []Cursor{CURSOR_MIN, 1, 2, CURSOR_MAX}
	for _, position := range positions {
		generator := test_chacha(test_seed(byte(position)), position)
		chacha_refill(&generator)
		if generator.Position != CURSOR_MIN {
			t.Fatalf("refill cursor was %d, want %d", generator.Position, CURSOR_MIN)
		}
	}
}

// Test_Source_Is_Transparent checks the vtable path equals the direct Read path from equal state,
// across a refill boundary, and that a nil Chacha dies before binding.
func Test_Source_Is_Transparent(t *testing.T) {
	subject := test_chacha(test_seed(6), CURSOR_MIN)
	reference := test_chacha(test_seed(6), CURSOR_MIN)
	source := Chacha_To_Source(&subject)
	got := make([]byte, BUFFER_BYTES+1)
	want := make([]byte, BUFFER_BYTES+1)
	Source_Read(source, Sink(got[:WORD_BYTE_COUNT]))
	Source_Read(source, Sink(got[WORD_BYTE_COUNT:]))
	Chacha_Read(&reference, Sink(want))
	if !bytes.Equal(got, want) {
		t.Fatalf("vtable bytes differ from direct Read bytes")
	}
	if !did_die(func() { Chacha_To_Source(nil) }) {
		t.Fatalf("nil Chacha did not die")
	}
	// Binding at each cursor boundary witnesses the Chacha invariant that the bind asserts.
	for _, position := range []Cursor{CURSOR_MIN, CURSOR_MIN + 1, CURSOR_MIN + 2, CURSOR_MAX} {
		positioned := test_chacha(test_seed(6), position)
		Source_Read(Chacha_To_Source(&positioned), Sink(got[:WORD_BYTE_COUNT]))
	}
	// A constructed buffer reaches each word edge of the slot deterministically; a keystream
	// would take 2^64 draws to land on one of them by chance.
	edges := []Word{WORD_MINIMUM, WORD_MINIMUM + 1, WORD_MINIMUM + 2, WORD_MAXIMUM}
	for _, word := range edges {
		edge := Chacha{}
		edge.Buffer.Lane_0 = Buffer_Lane_0(word)
		var octet [WORD_BYTE_COUNT]byte
		Source_Read(Chacha_To_Source(&edge), Sink(octet[:]))
		if test_word_from_bytes(octet[:]) != word {
			t.Fatalf("constructed word %d did not pass through the slot", word)
		}
	}
}

// Test_Source_Marks_A_Cryptographic_Parameter checks a backend enters only through State and
// Next, that Source_Read packs each word little-endian and spends a whole word on a partial tail,
// and that an unbound Source dies.
func Test_Source_Marks_A_Cryptographic_Parameter(t *testing.T) {
	counter := Word(0)
	source := Source{State: &counter, Next: counter_next}
	var got [WORD_BYTE_COUNT + 1]byte
	Source_Read(source, Sink(got[:]))
	first := [WORD_BYTE_COUNT]byte{1}
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

// Test_Allocation keeps every exported entropy path allocation-free with assertions active.
func Test_Allocation(t *testing.T) {
	seed := test_seed(8)
	var initialized Chacha
	require_zero_allocation(t, "Chacha_Init", func() {
		Chacha_Init(&initialized, seed, CURSOR_MIN)
	})
	read := test_chacha(seed, CURSOR_MIN)
	var sink [WORD_BYTE_COUNT]byte
	var count Count
	require_zero_allocation(t, "Chacha_Read", func() {
		count = Chacha_Read(&read, Sink(sink[:]))
	})
	source_generator := test_chacha(seed, CURSOR_MIN)
	var source Source
	require_zero_allocation(t, "Chacha_To_Source", func() {
		source = Chacha_To_Source(&source_generator)
	})
	require_zero_allocation(t, "Source_Read", func() { Source_Read(source, Sink(sink[:])) })
	below := test_chacha(seed, CURSOR_MIN)
	var index Index
	require_zero_allocation(t, "Chacha_Below", func() {
		index = Chacha_Below(&below, BOUND_MIN)
	})
	if count == Count(len(sink)) {
		if index < Index(BOUND_MIN) {
			test_state_invariant_domains()
			return
		}
	}
	t.Fatal("allocation probes did not complete")
}

type allocation_operation func()

func require_zero_allocation(t *testing.T, name string, operation allocation_operation) {
	t.Helper()
	allocations := testing.AllocsPerRun(1, operation)
	if allocations != 0 {
		t.Fatalf("%s allocated %.1f times per call; want zero", name, allocations)
	}
}

func test_state_invariant_domains() {
	for _, value := range [...]uint64{
		bits.WORD_64_MINIMUM,
		1,
		2,
		bits.WORD_64_MAXIMUM,
	} {
		generator := domain_chacha(value)

		initialized := generator
		Chacha_Init(&initialized, test_seed(1), CURSOR_MIN)

		read := generator
		Chacha_Read(&read, nil)

		bounded := generator
		Chacha_Below(&bounded, BOUND_MIN)

		bound := generator
		Chacha_To_Source(&bound)

		refilled := generator
		chacha_refill(&refilled)

		var serialized [BUFFER_BYTES]byte
		buffer_copy(generator.Buffer, Buffer_Destination(serialized[:]))
		replaced := generator.Buffer
		buffer_replace(Buffer_Handle(&replaced), Buffer_Source(serialized[:]))
	}
	for _, position := range [...]Cursor{CURSOR_MIN, 1, 2, CURSOR_MAX} {
		generator := domain_chacha(bits.WORD_64_MINIMUM)
		generator.Position = position
		Chacha_Read(&generator, nil)
		Chacha_Init(&generator, test_seed(1), CURSOR_MIN)
	}
}

func domain_chacha(value uint64) (generator Chacha) {
	generator.Key = Key{
		Lane_0: Key_Lane_0(value),
		Lane_1: Key_Lane_1(value),
		Lane_2: Key_Lane_2(value),
		Lane_3: Key_Lane_3(value),
	}
	generator.Buffer = Buffer{
		Lane_0:  Buffer_Lane_0(value),
		Lane_1:  Buffer_Lane_1(value),
		Lane_2:  Buffer_Lane_2(value),
		Lane_3:  Buffer_Lane_3(value),
		Lane_4:  Buffer_Lane_4(value),
		Lane_5:  Buffer_Lane_5(value),
		Lane_6:  Buffer_Lane_6(value),
		Lane_7:  Buffer_Lane_7(value),
		Lane_8:  Buffer_Lane_8(value),
		Lane_9:  Buffer_Lane_9(value),
		Lane_10: Buffer_Lane_10(value),
		Lane_11: Buffer_Lane_11(value),
		Lane_12: Buffer_Lane_12(value),
		Lane_13: Buffer_Lane_13(value),
		Lane_14: Buffer_Lane_14(value),
		Lane_15: Buffer_Lane_15(value),
		Lane_16: Buffer_Lane_16(value),
		Lane_17: Buffer_Lane_17(value),
		Lane_18: Buffer_Lane_18(value),
		Lane_19: Buffer_Lane_19(value),
		Lane_20: Buffer_Lane_20(value),
		Lane_21: Buffer_Lane_21(value),
		Lane_22: Buffer_Lane_22(value),
		Lane_23: Buffer_Lane_23(value),
		Lane_24: Buffer_Lane_24(value),
		Lane_25: Buffer_Lane_25(value),
		Lane_26: Buffer_Lane_26(value),
		Lane_27: Buffer_Lane_27(value),
	}
	return generator
}

// A slot that counts its draws, so a test can see how many words a read spent.
func counter_next(state Backend_State) (value Word) {
	counter := state.(*Word)
	*counter++
	return *counter
}

// Runs action and reports whether it tripped a fatal invariant.
func did_die(action func()) (died bool) {
	defer func() { died = recover() != nil }()
	action()
	return died
}

// Decodes a 32-byte hex key, failing the test on a wrong length.
func decode_key(t *testing.T, encoded string) (key Key_Source) {
	decoded := decode_bytes(t, encoded)
	if len(decoded) != 32 {
		t.Fatalf("key is %d bytes, want 32", len(decoded))
	}
	return Key_Source(decoded)
}

// Decodes a 12-byte hex nonce, failing the test on a wrong length.
func decode_nonce(t *testing.T, encoded string) (nonce Nonce) {
	decoded := decode_bytes(t, encoded)
	if len(decoded) != 12 {
		t.Fatalf("nonce is %d bytes, want 12", len(decoded))
	}
	return Nonce(decoded)
}

func test_seed(octet byte) (seed Seed) {
	seed = make(Seed, KEY_BYTES)
	seed[0] = octet
	return seed
}

func test_chacha(seed Seed, position Cursor) (generator Chacha) {
	Chacha_Init(&generator, seed, position)
	return generator
}

func test_word_from_bytes(octet []byte) (word Word) {
	for index := WORD_BYTE_COUNT - 1; index >= 0; index-- {
		word = word<<bits.BIT_COUNT_8_MAXIMUM | Word(octet[index])
	}
	return word
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
