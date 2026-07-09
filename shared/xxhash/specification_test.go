package xxhash_test

import (
	"bytes"
	"io"
	"testing"

	"local/james-orcales/shared/prng"
	"local/james-orcales/shared/xxhash"
)

// Test_Hash_Matches_Reference_Vectors checks the one-shot Hash against the published XXH64 vectors.
func Test_Hash_Matches_Reference_Vectors(t *testing.T) {
	for _, test_case := range reference_cases() {
		got := xxhash.Hash([]byte(test_case.Input), test_case.Seed)
		if got != test_case.Want {
			t.Fatalf("Hash %q seed %d = %#016x, want %#016x",
				test_case.Name, test_case.Seed, got, test_case.Want)
		}
	}
}

// Test_Digest_Matches_Reference_Vectors checks the streaming Digest reproduces those vectors when
// fed in chunks of every size, exercising the partial-stripe buffering across Write boundaries.
func Test_Digest_Matches_Reference_Vectors(t *testing.T) {
	chunk_sizes := []int{1, 2, 3, 7, 13, 32, 100}
	for _, test_case := range reference_cases() {
		for _, chunk := range chunk_sizes {
			digest := xxhash.New_Digest(test_case.Seed)
			input := []byte(test_case.Input)
			for offset := 0; offset < len(input); offset += chunk {
				end := offset + chunk
				if end > len(input) {
					end = len(input)
				}
				digest.Write(input[offset:end])
			}
			got := xxhash.Digest_Sum64(&digest)
			if got != test_case.Want {
				t.Fatalf("Digest %q seed %d chunk %d = %#016x, want %#016x",
					test_case.Name, test_case.Seed, chunk, got, test_case.Want)
			}
		}
	}
}

// Test_Digest_Equals_One_Shot checks streaming and one-shot agree on a longer input split at many
// chunk sizes — the case fixed vectors underweight.
func Test_Digest_Equals_One_Shot(t *testing.T) {
	generator := prng.New(99)
	data := make([]byte, 1000)
	for index := 0; index < len(data); index++ {
		data[index] = byte(prng.Generator_Next(&generator))
	}
	seed := uint64(0xabcdef)
	want := xxhash.Hash(data, seed)
	chunk_sizes := []int{1, 5, 8, 31, 32, 33, 64, 257}
	for _, chunk := range chunk_sizes {
		digest := xxhash.New_Digest(seed)
		for offset := 0; offset < len(data); offset += chunk {
			end := offset + chunk
			if end > len(data) {
				end = len(data)
			}
			digest.Write(data[offset:end])
		}
		got := xxhash.Digest_Sum64(&digest)
		if got != want {
			t.Fatalf("Digest chunk %d = %#016x, want one-shot %#016x", chunk, got, want)
		}
	}
}

// Test_Write_Reports_Full_Count checks Write consumes and reports every byte and never errors,
// and that *Digest works as an io.Writer, so io.Copy produces the same result as Hash.
func Test_Write_Reports_Full_Count(t *testing.T) {
	digest := xxhash.New_Digest(0)
	count, write_error := digest.Write(make([]byte, 50))
	if write_error != nil {
		t.Fatalf("Write errored: %v", write_error)
	}
	if count != 50 {
		t.Fatalf("Write reported %d bytes, want 50", count)
	}

	data := []byte("streamed through io.Copy into the digest")
	streamed := xxhash.New_Digest(0)
	var writer io.Writer = &streamed
	_, copy_error := io.CopyN(writer, bytes.NewReader(data), int64(len(data)))
	if copy_error != nil {
		t.Fatalf("io.CopyN errored: %v", copy_error)
	}
	if got, want := xxhash.Digest_Sum64(&streamed), xxhash.Hash(data, 0); got != want {
		t.Fatalf("io.Copy digest = %#016x, want %#016x", got, want)
	}
}

// Test_Reset_Restores_Initial_State checks Digest_Reset returns a used Digest to the state of a
// fresh one with the same seed.
func Test_Reset_Restores_Initial_State(t *testing.T) {
	digest := xxhash.New_Digest(7)
	digest.Write([]byte("garbage that should be forgotten on reset"))
	xxhash.Digest_Reset(&digest)
	digest.Write([]byte("asdf"))
	got := xxhash.Digest_Sum64(&digest)

	fresh := xxhash.New_Digest(7)
	fresh.Write([]byte("asdf"))
	want := xxhash.Digest_Sum64(&fresh)

	if got != want {
		t.Fatalf("after reset = %#016x, want fresh %#016x", got, want)
	}
}

// Test_Hot_Path_Is_Zero_Allocation checks a one-shot Hash of a preallocated slice never allocates.
func Test_Hot_Path_Is_Zero_Allocation(t *testing.T) {
	data := make([]byte, 64)
	allocations := testing.AllocsPerRun(1000, func() {
		xxhash.Hash(data, 0)
	})
	if allocations != 0 {
		t.Fatalf("Hash allocated %.1f times per call, want zero", allocations)
	}
}

// SENTENCE_63 is a 63-byte input: long enough to run one full 32-byte stripe and then a 31-byte
// tail that exercises every remainder branch (8, 8, 8, 4, 3). Borrowed from the canonical xxHash
// Go port's vectors so the frozen hashes can be trusted against the C reference.
const SENTENCE_63 = "Call me Ishmael. Some years ago--never mind how long precisely-"

// One published (input, seed) to XXH64 known answer.
type reference_case struct {
	Name  string
	Input string
	Seed  uint64
	Want  uint64
}

// The official XXH64 vectors (from cespare/xxhash, tested against the C reference): empty,
// sub-word, word-sized, and 63-byte inputs across several seeds.
func reference_cases() (cases []reference_case) {
	return []reference_case{
		{Name: "empty", Input: "", Seed: 0, Want: 0xef46db3751d8e999},
		{Name: "a", Input: "a", Seed: 0, Want: 0xd24ec4f1a98c6e5b},
		{Name: "as", Input: "as", Seed: 0, Want: 0x1c330fb2d66be179},
		{Name: "asd", Input: "asd", Seed: 0, Want: 0x631c37ce72a97393},
		{Name: "asdf", Input: "asdf", Seed: 0, Want: 0x415872f599cea71e},
		{Name: "sentence", Input: SENTENCE_63, Seed: 0, Want: 0x02a2e85470d6fd96},
		{Name: "empty/123", Input: "", Seed: 123, Want: 0xe0db84de91f3e198},
		{Name: "asdf/max", Input: "asdf", Seed: ^uint64(0), Want: 0x9a2fd8473be539b6},
		{Name: "sentence/seed", Input: SENTENCE_63, Seed: 54321, Want: 0x1736d186daf5d1cd},
	}
}
