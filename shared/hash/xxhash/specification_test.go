package xxhash_test

import (
	"testing"

	"local/james-orcales/shared/hash/xxhash"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/sim/prng"
	"local/james-orcales/shared/testify"
)

// Test_Hash_Matches_Reference_Vectors checks the one-shot Hash against the published XXH64 vectors.
func Test_Hash_Matches_Reference_Vectors(t *testing.T) {
	for _, test_case := range reference_cases() {
		got := xxhash.Hash(xxhash.Source(test_case.Input), test_case.Seed)
		testify.Equal(t, test_case.Want, got, test_case.Name)
	}
	xxhash_hash_domains(t)
}

// Test_Digest_Matches_Reference_Vectors checks the streaming Digest reproduces those vectors when
// fed in chunks of every size, exercising the partial-stripe buffering across Write boundaries.
func Test_Digest_Matches_Reference_Vectors(t *testing.T) {
	chunk_sizes := []int{1, 2, 3, 7, 13, 32, 100}
	for _, test_case := range reference_cases() {
		for _, chunk := range chunk_sizes {
			digest := xxhash.New_Digest(test_case.Seed)
			input := xxhash.Source(test_case.Input)
			for offset := 0; offset < len(input); offset += chunk {
				end := offset + chunk
				if end > len(input) {
					end = len(input)
				}
				digest.Write(input[offset:end])
			}
			got := xxhash.Digest_Sum_64(&digest)
			testify.Equal(t, test_case.Want, got, test_case.Name, chunk)
		}
	}
	xxhash_digest_domains()
}

// Test_Digest_Equals_One_Shot checks streaming and one-shot agree on a longer input split at many
// chunk sizes — the case fixed vectors underweight.
func Test_Digest_Equals_One_Shot(t *testing.T) {
	generator := prng.New(99)
	data := make(xxhash.Source, 1000)
	for index := 0; index < len(data); index++ {
		data[index] = byte(prng.Xoshiro_Next(&generator))
	}
	seed := xxhash.Seed(0xabcdef)
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
		got := xxhash.Digest_Sum_64(&digest)
		testify.Equal(t, want, got, chunk)
	}
}

// Test_Write_Reports_Full_Count checks Write consumes and reports every byte and never errors,
// and that *Digest works as an io.Writer, so io.Copy produces the same result as Hash.
func Test_Write_Reports_Full_Count(t *testing.T) {
	digest := xxhash.New_Digest(xxhash.Seed(0))
	count, write_error := digest.Write(make([]byte, 50))
	testify.No_Error(t, write_error)
	testify.Equal(t, 50, count)

	data := xxhash.Source("streamed through writer method into digest")
	streamed := xxhash.New_Digest(xxhash.Seed(0))
	var write func([]byte) (count int, write_error error) = streamed.Write
	count, write_error = write(data)
	testify.No_Error(t, write_error)
	testify.Equal(t, len(data), count)
	testify.Equal(
		t, xxhash.Hash(data, xxhash.Seed(0)), xxhash.Digest_Sum_64(&streamed),
	)
}

// Test_Reset_Restores_Initial_State checks Digest_Reset returns a used Digest to the state of a
// fresh one with the same seed.
func Test_Reset_Restores_Initial_State(t *testing.T) {
	digest := xxhash.New_Digest(xxhash.Seed(7))
	digest.Write([]byte("garbage that should be forgotten on reset"))
	xxhash.Digest_Reset(&digest)
	digest.Write([]byte("asdf"))
	got := xxhash.Digest_Sum_64(&digest)

	fresh := xxhash.New_Digest(xxhash.Seed(7))
	fresh.Write([]byte("asdf"))
	want := xxhash.Digest_Sum_64(&fresh)
	testify.Equal(t, want, got)
}

// Test_Hot_Path_Is_Zero_Allocation checks a one-shot Hash of a preallocated slice never allocates.
func Test_Hot_Path_Is_Zero_Allocation(t *testing.T) {
	fixture := xxhash_allocation_fixture{Source: make(xxhash.Source, 64)}
	fixture.Digest = xxhash.New_Digest(0)
	testify.Zero_Allocation(t, func() {
		fixture.Value = xxhash.Hash(fixture.Source, fixture.Seed)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Digest = xxhash.New_Digest(fixture.Seed)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Count, fixture.Error = fixture.Digest.Write(fixture.Source)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Value = xxhash.Digest_Sum_64(&fixture.Digest)
	})
	testify.Zero_Allocation(t, func() {
		xxhash.Digest_Reset(&fixture.Digest)
	})
	testify.No_Error(t, fixture.Error)
	testify.Equal(t, len(fixture.Source), fixture.Count)
}

type xxhash_allocation_fixture struct {
	Source xxhash.Source
	Digest xxhash.Digest
	Seed   xxhash.Seed
	Value  xxhash.Value
	Count  int
	Error  error
}

func xxhash_hash_domains(t *testing.T) {
	var maximum_source [xxhash.SOURCE_SIZE_MAXIMUM]byte
	for _, size := range [...]int{0, 1, 2, xxhash.BUFFER_FILL_MAXIMUM} {
		xxhash.Hash(maximum_source[:size], 0)
	}
	xxhash.Hash(maximum_source[:], 0)
	for _, seed := range xxhash_words() {
		xxhash.Hash(nil, xxhash.Seed(seed))
		xxhash.New_Digest(xxhash.Seed(seed))
	}
	for _, target := range xxhash_words() {
		accumulator_seed := xxhash.Seed(target - xxhash.PRIME64_5)
		accumulator_digest := xxhash.New_Digest(accumulator_seed)
		xxhash.Digest_Sum_64(&accumulator_digest)
		seed := xxhash.Seed(xxhash_avalanche_inverse(target) - xxhash.PRIME64_5)
		testify.Equal(t, xxhash.Value(target), xxhash.Hash(nil, seed))
		digest := xxhash.New_Digest(seed)
		testify.Equal(t, xxhash.Value(target), xxhash.Digest_Sum_64(&digest))
	}
}

func xxhash_digest_domains() {
	var maximum_source [xxhash.SOURCE_SIZE_MAXIMUM]byte
	digest := xxhash.New_Digest(0)
	digest.Write(nil)
	digest.Write(maximum_source[:])
	for _, target := range xxhash_words() {
		var stripe [xxhash.STRIPE_BYTES]byte
		for lane_index := range xxhash.STRIPE_LANE_COUNT {
			for byte_index := range 8 {
				stripe[lane_index*8+byte_index] = byte(target >> (8 * byte_index))
			}
		}
		xxhash_write_state(target, target, stripe[:])
		preimage := uint64(bits.Rotate_Left_64(
			bits.Word_64(target*xxhash_odd_inverse(xxhash.PRIME64_1)), -31,
		)) - target*xxhash.PRIME64_2
		xxhash_write_state(preimage, target, stripe[:])
		xxhash_merge_state(target, 0)
		merge_input := (target - xxhash.PRIME64_4) *
			xxhash_odd_inverse(xxhash.PRIME64_1)
		round_zero := uint64(bits.Rotate_Left_64(
			bits.Word_64(target*xxhash.PRIME64_2), 31,
		)) * xxhash.PRIME64_1
		xxhash_merge_state(merge_input^round_zero, target)
	}
}

func xxhash_write_state(accumulator uint64, word uint64, stripe []byte) {
	value := xxhash.Digest{}
	for index := range xxhash.STRIPE_LANE_COUNT {
		value.State[index] = accumulator
	}
	for lane_index := range xxhash.STRIPE_LANE_COUNT {
		for byte_index := range 8 {
			stripe[lane_index*8+byte_index] = byte(word >> (8 * byte_index))
		}
	}
	value.Write(stripe)
}

func xxhash_merge_state(accumulator uint64, word uint64) {
	value := xxhash.Digest{}
	value.State[xxhash.STATE_TOTAL_BYTES_INDEX] = xxhash.STRIPE_BYTES
	value.State[xxhash.STATE_ACCUMULATOR_1_INDEX] = word
	rotated_word := bits.Rotate_Left_64(bits.Word_64(word), 1)
	value.State[xxhash.STATE_ACCUMULATOR_2_INDEX] = uint64(bits.Rotate_Left_64(
		bits.Word_64(accumulator-uint64(rotated_word)), -7,
	))
	xxhash.Digest_Sum_64(&value)
}

func xxhash_words() (values [xxhash.STRIPE_LANE_COUNT]uint64) {
	return [xxhash.STRIPE_LANE_COUNT]uint64{0, 1, 2, ^uint64(0)}
}

func xxhash_odd_inverse(value uint64) (inverse uint64) {
	inverse = value
	for range 6 {
		inverse *= 2 - value*inverse
	}
	return inverse
}

func xxhash_xor_shift_right_inverse(value uint64, shift int) (inverse uint64) {
	inverse = value
	for distance := shift; distance < 64; distance *= 2 {
		inverse ^= inverse >> distance
	}
	return inverse
}

func xxhash_avalanche_inverse(value uint64) (inverse uint64) {
	inverse = xxhash_xor_shift_right_inverse(value, 32)
	inverse *= xxhash_odd_inverse(xxhash.PRIME64_3)
	inverse = xxhash_xor_shift_right_inverse(inverse, 29)
	inverse *= xxhash_odd_inverse(xxhash.PRIME64_2)
	return xxhash_xor_shift_right_inverse(inverse, 33)
}

// SENTENCE_63 is a 63-byte input: long enough to run one full 32-byte stripe and then a 31-byte
// tail that exercises every remainder branch (8, 8, 8, 4, 3). Borrowed from the canonical xxHash
// Go port's vectors so the frozen hashes can be trusted against the C reference.
const SENTENCE_63 = "Call me Ishmael. Some years ago--never mind how long precisely-"

// One published (input, seed) to XXH64 known answer.
type reference_case struct {
	Name  string
	Input string
	Seed  xxhash.Seed
	Want  xxhash.Value
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
		{
			Name: "asdf/max", Input: "asdf",
			Seed: xxhash.Seed(^uint64(0)), Want: 0x9a2fd8473be539b6,
		},
		{Name: "sentence/seed", Input: SENTENCE_63, Seed: 54321, Want: 0x1736d186daf5d1cd},
	}
}

// Benchmark_Hash measures one-shot hash over mid-sized bounded buffer.
func Benchmark_Hash(b *testing.B) {
	data := make(xxhash.Source, 1024)
	b.SetBytes(int64(len(data)))
	for b.Loop() {
		xxhash.Hash(data, xxhash.Seed(0))
	}
}

// Benchmark_Digest measures streaming same buffer through Write then Sum.
func Benchmark_Digest(b *testing.B) {
	data := make(xxhash.Source, 1024)
	b.SetBytes(int64(len(data)))
	for b.Loop() {
		digest := xxhash.New_Digest(xxhash.Seed(0))
		digest.Write(data)
		xxhash.Digest_Sum_64(&digest)
	}
}
