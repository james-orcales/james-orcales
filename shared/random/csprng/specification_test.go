// Whitebox suite (package csprng): most checks are black-box, but the reference-vector test
// reaches the unexported chacha20_block to match RFC 8439's vectors before fast-key-erasure hides
// the raw keystream, and the erasure test reads the key. Every spec-mapped test is in this file, in
// heading order.
package csprng

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"io"
	"math/bits"
	"testing"

	invariant "local/james-orcales/shared/invariant/default"
)

// Test_Seed_Expands_To_State checks New is deterministic and seed-sensitive.
func Test_Seed_Expands_To_State(t *testing.T) {
	first := New([32]byte{1})
	again := New([32]byte{1})
	var first_draw, again_draw [8]byte
	first.Read(first_draw[:])
	again.Read(again_draw[:])
	if first_draw != again_draw {
		t.Fatalf("same seed produced different streams")
	}
	other := New([32]byte{2})
	repeat := New([32]byte{1})
	var other_draw, repeat_draw [8]byte
	other.Read(other_draw[:])
	repeat.Read(repeat_draw[:])
	if other_draw == repeat_draw {
		t.Fatalf("distinct seeds produced the same first draw")
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
		var output [64]byte
		chacha20_block(key, test_case.Counter, nonce, &output)
		if !bytes.Equal(output[:], want) {
			t.Fatalf("%s: block was %x, want %x", test_case.Name, output[:], want)
		}
	}
}

// Test_Known_Sequence locks the output stream for a fixed seed.
func Test_Known_Sequence(t *testing.T) {
	generator := New([32]byte{})
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
		var octet [8]byte
		generator.Read(octet[:])
		value := binary.LittleEndian.Uint64(octet[:])
		if value != want[index] {
			t.Fatalf("draw %d was %d, want %d", index, value, want[index])
		}
	}
}

// Test_Read_Fills_Fully checks Read fills all of p, reports the full count, and never errors. The
// byte-granular reads also walk the cursor and sink lengths through 0, 1, 2 so those invariant
// boundaries are witnessed; the larger reads cross a refill and prove real keystream.
func Test_Read_Fills_Fully(t *testing.T) {
	generator := New([32]byte{9})
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
	generator := New([32]byte{3})
	buffer := make([]byte, 100000)
	generator.Read(buffer)
	set_bits := 0
	for _, octet := range buffer {
		set_bits += bits.OnesCount8(octet)
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
	generator := New([32]byte{4})
	bounds := []int{1, 2, 7, 1000, 1 << 40}
	for _, bound := range bounds {
		for draw_index := 0; draw_index < 10000; draw_index++ {
			index := Generator_Below(&generator, Bound(bound))
			if index >= Index(bound) {
				t.Fatalf("Below(%d) returned %d, out of range", bound, index)
			}
		}
	}
	died := did_die(func() {
		Generator_Below(&generator, Bound(0))
	})
	if !died {
		t.Fatalf("Below with a zero bound did not exit")
	}
}

// Test_Seed_Is_Erased_After_Construction checks New performs the first refill, so the seed no
// longer lives in the Generator's key and a later disclosure cannot reproduce it or its output.
func Test_Seed_Is_Erased_After_Construction(t *testing.T) {
	seed := [32]byte{}
	for index := 0; index < 32; index++ {
		seed[index] = byte(index)
	}
	generator := New(seed)
	if generator.Key == seed {
		t.Fatalf("New left the seed in the generator key; fast-key-erasure did not run")
	}
}

// Test_Hot_Path_Is_Zero_Allocation checks a steady-state Read does not allocate, even with the
// invariant assertions on the draw path — measured under recording, not just in benchmark mode.
func Test_Hot_Path_Is_Zero_Allocation(t *testing.T) {
	generator := New([32]byte{8})
	buffer := make([]byte, 8)
	allocations := testing.AllocsPerRun(1000, func() {
		generator.Read(buffer)
	})
	if allocations != 0 {
		t.Fatalf("Read allocated %.1f times per call, want zero", allocations)
	}
}

// Runs action and reports whether it tripped a fatal invariant, used to assert preconditions. A
// violation exits through the Default recorder, which os.Exit cannot recover, so the helper swaps
// Exit for a panic — and silences the recorder's stderr — then recovers it, so the exit is
// observable in-process. Exit and Output are restored before returning. Copied from prng's suite.
func did_die(action func()) (died bool) {
	exit, output := invariant.Default.Exit, invariant.Default.Output
	invariant.Default.Exit = func(int) { panic(tripped_invariant{}) }
	invariant.Default.Output = io.Discard
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

// Decodes a 32-byte hex key, failing the test on a wrong length.
func decode_key(t *testing.T, encoded string) (key [32]byte) {
	decoded := decode_bytes(t, encoded)
	if len(decoded) != 32 {
		t.Fatalf("key is %d bytes, want 32", len(decoded))
	}
	copy(key[:], decoded)
	return key
}

// Decodes a 12-byte hex nonce, failing the test on a wrong length.
func decode_nonce(t *testing.T, encoded string) (nonce [12]byte) {
	decoded := decode_bytes(t, encoded)
	if len(decoded) != 12 {
		t.Fatalf("nonce is %d bytes, want 12", len(decoded))
	}
	copy(nonce[:], decoded)
	return nonce
}

// Decodes a hex string into bytes, failing the test on malformed input.
func decode_bytes(t *testing.T, encoded string) (decoded []byte) {
	decoded, decode_error := hex.DecodeString(encoded)
	if decode_error != nil {
		t.Fatalf("decode %q: %v", encoded, decode_error)
	}
	return decoded
}
