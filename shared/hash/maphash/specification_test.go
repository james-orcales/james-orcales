package maphash_test

import (
	"testing"

	"local/james-orcales/shared/hash/maphash"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/testify"
)

// Test_Package_Owned_State verifies streaming operations need no common dispatcher.
func Test_Package_Owned_State(t *testing.T) {
	// Boolean backing makes third lifecycle state unrepresentable.
	testify.False(t, bool(maphash.READY_EMPTY))
	testify.True(t, bool(maphash.READY_COMPLETE))
	var value maphash.Hash
	maphash.Hash_Init(&value, test_seed())
	write_output := maphash.Hash_Write(&value, maphash.Source("abc"))
	testify.Equal(t, maphash.Write_Output{
		Count: maphash.Count(len("abc")), Status: maphash.WRITE_STATUS_OK,
	}, write_output)
	var output [maphash.DIGEST_SIZE]byte
	sum_output := maphash.Hash_Sum_Into(&value, output[:])
	testify.Equal(t, maphash.Output{
		Count: maphash.OUTPUT_COUNT_COMPLETE, Status: maphash.OUTPUT_STATUS_OK,
	}, sum_output)
}

// Test_Reference_Values keeps the injected-key algorithm tied to SipHash-2-4 vectors.
func Test_Reference_Values(t *testing.T) {
	seed := maphash.Seed{
		Key_0: 0x0706050403020100,
		Key_1: 0x0f0e0d0c0b0a0908,
	}
	tests := [...]struct {
		Size int
		Want maphash.Value
	}{
		{Size: 0, Want: 0x726fdb47dd0e0e31},
		{Size: 1, Want: 0x74f839c593dc67fd},
		{Size: 2, Want: 0x0d6c8009d9a94f5a},
		{Size: 3, Want: 0x85676696d7fb7e2d},
		{Size: 4, Want: 0xcf2794e0277187b7},
		{Size: 8, Want: 0x93f5f5799a932462},
	}
	var source [maphash.BLOCK_SIZE]byte
	for index := range source {
		source[index] = byte(index)
	}
	for _, test := range tests {
		testify.Equal(t, test.Want, maphash.Bytes(seed, source[:test.Size]))
	}
}

// Test_Write_Division proves bytes, text, single bytes, and chunks form one stream.
func Test_Write_Division(t *testing.T) {
	seed := test_seed()
	want := maphash.Bytes(seed, maphash.Source("write division is irrelevant"))
	testify.Equal(t, want, maphash.String(seed, maphash.Text("write division is irrelevant")))

	var value maphash.Hash
	maphash.Hash_Init(&value, seed)
	maphash.Hash_Write(&value, maphash.Source("write "))
	maphash.Hash_Write_Text(&value, maphash.Text("division "))
	for _, item := range []byte("is irrelevant") {
		maphash.Hash_Write_Byte(&value, maphash.Byte(item))
	}
	testify.Equal(t, want, maphash.Hash_Sum_64(&value))
}

// Test_Seed_Reset_And_Clone protects explicit function identity and caller-owned state.
func Test_Seed_Reset_And_Clone(t *testing.T) {
	seed := test_seed()
	other := maphash.Seed{
		Key_0: seed.Key_0 + 1, Key_1: seed.Key_1, Output_Mask: seed.Output_Mask,
	}
	message := maphash.Source("seeded")
	testify.Not_Equal(t, maphash.Bytes(seed, message), maphash.Bytes(other, message))

	var source maphash.Hash
	maphash.Hash_Init(&source, seed)
	maphash.Hash_Write(&source, maphash.Source("seed"))
	var clone maphash.Hash
	maphash.Hash_Init(&clone, other)
	maphash.Hash_Clone_Into(&clone, &source)
	maphash.Hash_Write(&source, maphash.Source("ed"))
	testify.Equal(t, maphash.Bytes(seed, maphash.Source("seed")), maphash.Hash_Sum_64(&clone))
	testify.Equal(t, seed, maphash.Hash_Seed(&clone))

	maphash.Hash_Reset(&source)
	maphash.Hash_Write(&source, message)
	testify.Equal(t, maphash.Bytes(seed, message), maphash.Hash_Sum_64(&source))
	maphash.Hash_Set_Seed(&source, other)
	maphash.Hash_Write(&source, message)
	testify.Equal(t, maphash.Bytes(other, message), maphash.Hash_Sum_64(&source))
}

// Test_Caller_Owned_Output uses standard maphash little-endian order.
func Test_Caller_Owned_Output(t *testing.T) {
	seed := test_seed()
	var value maphash.Hash
	maphash.Hash_Init(&value, seed)
	maphash.Hash_Write(&value, maphash.Source("output"))
	want := maphash.Hash_Sum_64(&value)
	var short [maphash.DIGEST_SIZE - 1]byte
	sum_output := maphash.Hash_Sum_Into(&value, short[:])
	testify.Equal(t, maphash.Output{
		Count: maphash.OUTPUT_COUNT_EMPTY, Status: maphash.OUTPUT_STATUS_TOO_SMALL,
	}, sum_output)
	var output [maphash.DIGEST_SIZE]byte
	sum_output = maphash.Hash_Sum_Into(&value, output[:])
	testify.Equal(t, maphash.Output{
		Count: maphash.OUTPUT_COUNT_COMPLETE, Status: maphash.OUTPUT_STATUS_OK,
	}, sum_output)
	for index := range output {
		testify.Equal(t, byte(uint64(want)>>(index*maphash.BITS_PER_BYTE)), output[index])
	}
}

// Test_Bounds rejects oversized calls, total overflow, and an unkeyed seed.
func Test_Bounds(t *testing.T) {
	seed := test_seed()
	var uninitialized maphash.Hash
	testify.Panics(t, func() { maphash.Hash_Write(&uninitialized, nil) })
	testify.Panics(t, func() { maphash.Hash_Write_Text(&uninitialized, "") })
	testify.Panics(t, func() { maphash.Hash_Write_Byte(&uninitialized, 0) })
	testify.Panics(t, func() { maphash.Hash_Sum_64(&uninitialized) })
	testify.Panics(t, func() {
		var destination [maphash.DIGEST_SIZE]byte
		maphash.Hash_Sum_Into(&uninitialized, destination[:])
	})
	testify.Panics(t, func() { maphash.Hash_Seed(&uninitialized) })
	testify.Panics(t, func() { maphash.Hash_Message_Size_Maximum(&uninitialized) })
	testify.Panics(t, func() { maphash.Hash_Reset(&uninitialized) })
	testify.Panics(t, func() { maphash.Hash_Set_Seed(&uninitialized, seed) })
	testify.Panics(t, func() {
		var destination maphash.Hash
		maphash.Hash_Clone_Into(&destination, &uninitialized)
	})
	var value maphash.Hash
	maphash.Hash_Init(&value, seed)
	maphash.Hash_Clone_Into(&uninitialized, &value)
	var source [maphash.SOURCE_SIZE_MAXIMUM + 1]byte
	var text [maphash.TEXT_SIZE_MAXIMUM + 1]byte
	testify.Panics(t, func() { maphash.Bytes(seed, source[:]) })
	testify.Panics(t, func() { maphash.String(seed, maphash.Text(string(text[:]))) })
	testify.Panics(t, func() { maphash.Hash_Write(&value, source[:]) })
	testify.Panics(t, func() { maphash.Hash_Write_Text(&value, maphash.Text(string(text[:]))) })
	testify.Panics(t, func() { maphash.Hash_Init(&value, maphash.Seed{}) })

	maphash.Hash_Init_Bounded(&value, seed, 0)
	before := value
	write_output := maphash.Hash_Write(&value, maphash.Source{1})
	testify.Equal(t, maphash.Write_Output{
		Count: maphash.Count(0), Status: maphash.WRITE_STATUS_MESSAGE_TOO_LARGE,
	}, write_output)
	testify.Equal(t, before, value)
	write_output = maphash.Hash_Write_Text(&value, "x")
	testify.Equal(t, maphash.Write_Output{
		Count: maphash.Count(0), Status: maphash.WRITE_STATUS_MESSAGE_TOO_LARGE,
	}, write_output)
	write_status := maphash.Hash_Write_Byte(&value, 'x')
	testify.Equal(t, maphash.WRITE_STATUS_MESSAGE_TOO_LARGE, write_status)
}

// Test_Invariant_Domains reaches seed, state, count, tail position, and caller-byte sentinels.
func Test_Invariant_Domains(t *testing.T) {
	var source [maphash.SOURCE_SIZE_MAXIMUM]byte
	var output [maphash.DESTINATION_SIZE_MAXIMUM]byte
	maphash_seed_domains(source[:], output[:])
	maphash_state_domains(t, output[:])
	maphash_tail_byte_domains(test_seed())
	seed := test_seed()
	for _, size := range [...]int{0, 1, 2, maphash.DESTINATION_SIZE_MAXIMUM} {
		var value maphash.Hash
		maphash.Hash_Init(&value, seed)
		maphash.Hash_Sum_Into(&value, output[:size])
	}
	states := [...]uint64{
		bits.WORD_64_MINIMUM,
		bits.WORD_64_MINIMUM + 1,
		bits.WORD_64_MINIMUM + 1 + 1,
		bits.WORD_64_MAXIMUM,
	}
	for _, maximum := range [...]maphash.Message_Size_Maximum{
		maphash.Message_Size_Maximum(maphash.TOTAL_COUNT_MINIMUM),
		maphash.Message_Size_Maximum(maphash.TOTAL_COUNT_MINIMUM + 1),
		maphash.Message_Size_Maximum(maphash.TOTAL_COUNT_MINIMUM + 1 + 1),
		maphash.Message_Size_Maximum(maphash.TOTAL_COUNT_MAXIMUM),
	} {
		var value maphash.Hash
		maphash.Hash_Init_Bounded(&value, seed, maximum)
		testify.Equal(t, maximum, maphash.Hash_Message_Size_Maximum(&value))
		maphash.Hash_Init_Bounded(&value, seed, maximum)
	}
	var byte_value maphash.Hash
	maphash.Hash_Init(&byte_value, seed)
	maphash.Hash_Write_Byte(&byte_value, maphash.Byte(bits.WORD_8_MAXIMUM))
	base_seed := test_seed()
	base := maphash.Bytes(base_seed, nil)
	for _, target := range states {
		masked := base_seed
		masked.Output_Mask = maphash.Output_Mask(
			uint64(base) ^ uint64(base_seed.Output_Mask) ^ target,
		)
		testify.Equal(t, maphash.Value(target), maphash.Bytes(masked, nil))
		testify.Equal(t, maphash.Value(target), maphash.String(masked, ""))
		var value maphash.Hash
		maphash.Hash_Init(&value, masked)
		testify.Equal(t, maphash.Value(target), maphash.Hash_Sum_64(&value))
	}
}

// Test_Allocation proves byte, text, stream, clone, reset, and output paths own no heap storage.
func Test_Allocation(t *testing.T) {
	// Output storage is made here, outside every Zero_Allocation closure.
	fixture := allocation_fixture{
		Seed: test_seed(), Source: maphash.Source("allocation"), Text: "allocation",
		Output:               make(maphash.Destination, maphash.DIGEST_SIZE),
		Message_Size_Maximum: maphash.Message_Size_Maximum(maphash.SOURCE_SIZE_MAXIMUM),
	}
	maphash.Hash_Init(&fixture.Hash, fixture.Seed)
	maphash.Hash_Init(&fixture.Clone, fixture.Seed)
	testify.Zero_Allocation(t, func() {
		fixture.Value = maphash.Bytes(fixture.Seed, fixture.Source)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Value = maphash.String(fixture.Seed, fixture.Text)
	})
	testify.Zero_Allocation(t, func() { maphash.Hash_Init(&fixture.Hash, fixture.Seed) })
	testify.Zero_Allocation(t, func() {
		maphash.Hash_Init_Bounded(&fixture.Hash, fixture.Seed, fixture.Message_Size_Maximum)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Write_Output = maphash.Hash_Write(
			&fixture.Hash, fixture.Source,
		)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Write_Output = maphash.Hash_Write_Text(
			&fixture.Hash, fixture.Text,
		)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Write_Status = maphash.Hash_Write_Byte(&fixture.Hash, 'x')
	})
	testify.Zero_Allocation(t, func() {
		fixture.Value = maphash.Hash_Sum_64(&fixture.Hash)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Output_Result = maphash.Hash_Sum_Into(
			&fixture.Hash, fixture.Output,
		)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Observed_Seed = maphash.Hash_Seed(&fixture.Hash)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Observed_Message_Size_Maximum = maphash.Hash_Message_Size_Maximum(
			&fixture.Hash,
		)
	})
	testify.Zero_Allocation(t, func() { maphash.Hash_Reset(&fixture.Hash) })
	testify.Zero_Allocation(t, func() {
		maphash.Hash_Set_Seed(&fixture.Hash, fixture.Seed)
	})
	testify.Zero_Allocation(t, func() {
		maphash.Hash_Clone_Into(&fixture.Clone, &fixture.Hash)
	})
	testify.True(t, fixture.Value >= 0)
}

// Full-domain state is caller-owned, so empty writes expose every legal stored bit pattern.
func maphash_state_domains(t *testing.T, output maphash.Destination) {
	words := [...]uint64{
		bits.WORD_64_MINIMUM,
		bits.WORD_64_MINIMUM + 1,
		bits.WORD_64_MINIMUM + 1 + 1,
		bits.WORD_64_MAXIMUM,
	}
	counts := [...]uint32{
		maphash.TOTAL_COUNT_MINIMUM,
		maphash.TOTAL_COUNT_MINIMUM + 1,
		maphash.TOTAL_COUNT_MINIMUM + 1 + 1,
		maphash.TOTAL_COUNT_MAXIMUM,
	}
	tails := [...]int{
		maphash.TAIL_COUNT_MINIMUM,
		maphash.TAIL_COUNT_MINIMUM + 1,
		maphash.TAIL_COUNT_MINIMUM + 1 + 1,
		maphash.TAIL_COUNT_MAXIMUM,
	}
	// Every position takes the row byte, because stale bytes past Tail_Count are legal state
	// and the top position is otherwise only reachable through a completed word.
	tail_bytes := [...]uint8{
		maphash.TAIL_BYTE_MINIMUM,
		maphash.TAIL_BYTE_MINIMUM + 1,
		maphash.TAIL_BYTE_MINIMUM + 1 + 1,
		maphash.TAIL_BYTE_MAXIMUM,
	}
	for index, word := range words {
		value := maphash.Hash{
			Seed: maphash.Seed{
				Key_0:       maphash.Key_0(word),
				Key_1:       maphash.Key_1(word),
				Output_Mask: maphash.Output_Mask(word),
			},
			State_0:              maphash.State_0(word),
			State_1:              maphash.State_1(word),
			State_2:              maphash.State_2(word),
			State_3:              maphash.State_3(word),
			Tail:                 test_tail(tail_bytes[index]),
			Tail_Count:           maphash.Tail_Count(tails[index]),
			Total_Count:          maphash.Total_Count(counts[index]),
			Message_Size_Maximum: maphash.Message_Size_Maximum(counts[index]),
			Ready:                maphash.READY_COMPLETE,
		}
		maphash.Hash_Write(&value, nil)
		maphash.Hash_Write_Text(&value, "")
		byte_value := value
		maphash.Hash_Write_Byte(&byte_value, maphash.Byte(index))
		maphash.Hash_Sum_64(&value)
		maphash.Hash_Sum_Into(&value, output)
		maphash.Hash_Seed(&value)
		maphash.Hash_Message_Size_Maximum(&value)
		clone := value
		maphash.Hash_Clone_Into(&clone, &value)
		reset := value
		if index == 0 {
			testify.Panics(t, func() { maphash.Hash_Reset(&reset) })
		} else {
			maphash.Hash_Reset(&reset)
		}
		set_seed := value
		maphash.Hash_Set_Seed(&set_seed, test_seed())
		initialized := value
		maphash.Hash_Init(&initialized, test_seed())
		maphash.Hash_Init_Bounded(
			&initialized, test_seed(), maphash.MESSAGE_SIZE_MAXIMUM,
		)
	}
}

// A completed word leaves every position holding its byte, so the next write sees each sentinel
// in every position, the top one included.
func maphash_tail_byte_domains(seed maphash.Seed) {
	for _, item := range [...]uint8{
		maphash.TAIL_BYTE_MINIMUM,
		maphash.TAIL_BYTE_MINIMUM + 1,
		maphash.TAIL_BYTE_MINIMUM + 1 + 1,
		maphash.TAIL_BYTE_MAXIMUM,
	} {
		source := make(maphash.Source, maphash.BLOCK_SIZE)
		for index := range source {
			source[index] = item
		}
		var value maphash.Hash
		maphash.Hash_Init(&value, seed)
		maphash.Hash_Write(&value, source[:maphash.TAIL_COUNT_MAXIMUM])
		maphash.Hash_Sum_64(&value)
		maphash.Hash_Write_Byte(&value, maphash.Byte(item))
		maphash.Hash_Write_Byte(&value, maphash.Byte(item))
	}
}

func test_tail(item uint8) (tail maphash.Tail) {
	return maphash.Tail{
		Byte_0: maphash.Tail_Byte_0(item),
		Byte_1: maphash.Tail_Byte_1(item),
		Byte_2: maphash.Tail_Byte_2(item),
		Byte_3: maphash.Tail_Byte_3(item),
		Byte_4: maphash.Tail_Byte_4(item),
		Byte_5: maphash.Tail_Byte_5(item),
		Byte_6: maphash.Tail_Byte_6(item),
		Byte_7: maphash.Tail_Byte_7(item),
	}
}

func test_seed() (seed maphash.Seed) {
	return maphash.Seed{
		Key_0:       0x0706050403020100,
		Key_1:       0x0f0e0d0c0b0a0908,
		Output_Mask: 0x9e3779b97f4a7c15,
	}
}

func maphash_seed_domains(source maphash.Source, output maphash.Destination) {
	seeds := [...]maphash.Seed{
		{Key_0: 0, Key_1: 1, Output_Mask: 0},
		{Key_0: 1, Key_1: 0, Output_Mask: 1},
		{Key_0: 2, Key_1: 2, Output_Mask: 2},
		{
			Key_0:       maphash.Key_0(bits.WORD_64_MAXIMUM),
			Key_1:       maphash.Key_1(bits.WORD_64_MAXIMUM),
			Output_Mask: maphash.Output_Mask(bits.WORD_64_MAXIMUM),
		},
	}
	for _, seed := range seeds {
		for _, size := range [...]int{0, 1, 2, maphash.SOURCE_SIZE_MAXIMUM} {
			maphash.Bytes(seed, source[:size])
			maphash.String(seed, maphash.Text(string(source[:size])))
			var value maphash.Hash
			maphash.Hash_Init(&value, seed)
			maphash.Hash_Write(&value, source[:size])
			maphash.Hash_Init(&value, seed)
			maphash.Hash_Write_Text(&value, maphash.Text(string(source[:size])))
			maphash.Hash_Init_Bounded(&value, seed, maphash.MESSAGE_SIZE_MAXIMUM)
		}
		maphash_tail_domains(seed, source, output)
	}
}

func maphash_tail_domains(
	seed maphash.Seed, source maphash.Source, output maphash.Destination,
) {
	for _, tail_count := range [...]int{
		maphash.TAIL_COUNT_MINIMUM,
		maphash.TAIL_COUNT_MINIMUM + 1,
		maphash.TAIL_COUNT_MINIMUM + 1 + 1,
		maphash.TAIL_COUNT_MAXIMUM,
	} {
		var value maphash.Hash
		maphash.Hash_Init(&value, seed)
		maphash.Hash_Write(&value, source[:tail_count])
		maphash.Hash_Write(&value, nil)
		maphash.Hash_Write_Text(&value, "")
		maphash.Hash_Sum_64(&value)
		maphash.Hash_Sum_Into(&value, output)
		maphash.Hash_Seed(&value)
		maphash.Hash_Message_Size_Maximum(&value)
		var clone maphash.Hash
		maphash.Hash_Clone_Into(&clone, &value)
		byte_value := value
		maphash.Hash_Write_Byte(&byte_value, maphash.Byte(tail_count))
		seed_value := value
		maphash.Hash_Set_Seed(&seed_value, seed)
		maphash.Hash_Init_Bounded(&value, seed, maphash.MESSAGE_SIZE_MAXIMUM)
		maphash.Hash_Reset(&value)
	}
}

type allocation_fixture struct {
	Seed                          maphash.Seed
	Observed_Seed                 maphash.Seed
	Hash                          maphash.Hash
	Clone                         maphash.Hash
	Source                        maphash.Source
	Text                          maphash.Text
	Output                        maphash.Destination
	Value                         maphash.Value
	Write_Output                  maphash.Write_Output
	Write_Status                  maphash.Write_Status
	Output_Result                 maphash.Output
	Message_Size_Maximum          maphash.Message_Size_Maximum
	Observed_Message_Size_Maximum maphash.Message_Size_Maximum
}
