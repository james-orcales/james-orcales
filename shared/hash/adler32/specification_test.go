package adler32_test

import (
	"testing"

	"local/james-orcales/shared/hash/adler32"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/testify"
)

// Test_Package_Owned_State verifies streaming operations need no common dispatcher.
func Test_Package_Owned_State(t *testing.T) {
	// Boolean backing makes third lifecycle state unrepresentable.
	testify.False(t, bool(adler32.READY_EMPTY))
	testify.True(t, bool(adler32.READY_COMPLETE))
	var digest adler32.Digest
	adler32.Digest_Init(&digest)
	count := adler32.Digest_Write(&digest, adler32.Source("abc"))
	testify.Equal(t, adler32.Count(len("abc")), count)

	var output [adler32.DIGEST_SIZE]byte
	output_count, status := adler32.Digest_Sum_Into(&digest, output[:])
	testify.Equal(t, adler32.OUTPUT_COUNT_COMPLETE, output_count)
	testify.Equal(t, adler32.OUTPUT_STATUS_OK, status)
	adler32.Digest_Reset(&digest)
	testify.Equal(t, adler32.Digest_Value(1), adler32.Digest_Sum_32(&digest))
}

// Test_Reference_Values keeps the implementation tied to published results.
func Test_Reference_Values(t *testing.T) {
	tests := [...]struct {
		Text string
		Want adler32.Value
	}{
		{Text: "", Want: reference_value(0x00000001)},
		{Text: "a", Want: reference_value(0x00620062)},
		{Text: "abc", Want: reference_value(0x024d0127)},
		{Text: "abcdefghij", Want: reference_value(0x158603f8)},
		{
			Text: "Discard medicine more than two years old.",
			Want: reference_value(0x3f090f02),
		},
	}
	for _, test := range tests {
		testify.Equal(t, test.Want, adler32.Checksum(adler32.Source(test.Text)))
		var digest adler32.Digest
		adler32.Digest_Init(&digest)
		midpoint := len(test.Text) / 2
		adler32.Digest_Write(&digest, adler32.Source(test.Text[:midpoint]))
		adler32.Digest_Write(&digest, adler32.Source(test.Text[midpoint:]))
		testify.Equal(t, test.Want,
			reference_value(uint32(adler32.Digest_Sum_32(&digest))))
	}
}

// Test_Reduction_Boundary protects reduction across the bounded-call boundary.
func Test_Reduction_Boundary(t *testing.T) {
	var source [adler32.SOURCE_SIZE_MAXIMUM + 1]byte
	for index := range source {
		source[index] = 0xff
	}
	var digest adler32.Digest
	adler32.Digest_Init(&digest)
	adler32.Digest_Write(&digest, source[:adler32.SOURCE_SIZE_MAXIMUM])
	adler32.Digest_Write(&digest, source[adler32.SOURCE_SIZE_MAXIMUM:])
	testify.Equal(t, reference_checksum(source[:]),
		reference_value(uint32(adler32.Digest_Sum_32(&digest))))
}

// Test_Caller_Owned_Output keeps output bounded and state unchanged.
func Test_Caller_Owned_Output(t *testing.T) {
	var digest adler32.Digest
	adler32.Digest_Init(&digest)
	adler32.Digest_Write(&digest, adler32.Source("abc"))
	before := digest
	var short [adler32.DIGEST_SIZE - 1]byte
	count, status := adler32.Digest_Sum_Into(&digest, short[:])
	testify.Equal(t, adler32.OUTPUT_COUNT_EMPTY, count)
	testify.Equal(t, adler32.OUTPUT_STATUS_TOO_SMALL, status)
	var output [adler32.DIGEST_SIZE]byte
	count, status = adler32.Digest_Sum_Into(&digest, output[:])
	testify.Equal(t, adler32.OUTPUT_COUNT_COMPLETE, count)
	testify.Equal(t, adler32.OUTPUT_STATUS_OK, status)
	testify.Equal(t, [adler32.DIGEST_SIZE]byte{0x02, 0x4d, 0x01, 0x27}, output)
	testify.Equal(t, before, digest)
}

// Test_State_Compatibility protects the standard state format and hostile input checks.
func Test_State_Compatibility(t *testing.T) {
	var digest adler32.Digest
	adler32.Digest_Init(&digest)
	adler32.Digest_Write(&digest, adler32.Source("ab"))
	var state [adler32.STATE_SIZE]byte
	count, status := adler32.Digest_Marshal_Into(&digest, state[:])
	testify.Equal(t, adler32.State_Count(adler32.STATE_SIZE), count)
	testify.Equal(t, adler32.STATE_OUTPUT_STATUS_OK, status)
	testify.Equal(t, [adler32.STATE_SIZE]byte{'a', 'd', 'l', 1, 0x01, 0x26, 0x00, 0xc4}, state)

	var restored adler32.Digest
	input_status := adler32.Digest_Unmarshal(&restored, state[:])
	testify.Equal(t, adler32.STATE_INPUT_STATUS_OK, input_status)
	testify.Equal(t, digest, restored)
	previous := restored
	testify.Equal(t, adler32.STATE_INPUT_STATUS_IDENTIFIER_INVALID,
		adler32.Digest_Unmarshal(&restored, nil))
	testify.Equal(t, previous, restored)
	testify.Equal(t, adler32.STATE_INPUT_STATUS_SIZE_INVALID,
		adler32.Digest_Unmarshal(&restored, state[:len(state)-1]))
	testify.Equal(t, previous, restored)
	state[0] ^= 1
	testify.Equal(t, adler32.STATE_INPUT_STATUS_IDENTIFIER_INVALID,
		adler32.Digest_Unmarshal(&restored, state[:]))
	testify.Equal(t, previous, restored)
	var short [adler32.STATE_SIZE - 1]byte
	count, status = adler32.Digest_Marshal_Into(&digest, short[:])
	testify.Equal(t, adler32.State_Count(0), count)
	testify.Equal(t, adler32.STATE_OUTPUT_STATUS_TOO_SMALL, status)
}

// Test_Clone preserves independent caller-owned state.
func Test_Clone(t *testing.T) {
	var source adler32.Digest
	adler32.Digest_Init(&source)
	testify.Equal(t, adler32.Count(2),
		adler32.Digest_Write(&source, adler32.Source("ab")))

	var destination adler32.Digest
	adler32.Digest_Init(&destination)
	adler32.Digest_Clone_Into(&destination, &source)
	adler32.Digest_Write(&source, adler32.Source("c"))
	testify.Equal(t, adler32.Digest_Value(0x024d0127), adler32.Digest_Sum_32(&source))
	testify.Equal(t, adler32.Digest_Value(0x012600c4),
		adler32.Digest_Sum_32(&destination))
}

// Test_Bounds rejects a caller crossing the shared byte boundary.
func Test_Bounds(t *testing.T) {
	var uninitialized adler32.Digest
	testify.Panics(t, func() { adler32.Digest_Write(&uninitialized, nil) })
	testify.Panics(t, func() { adler32.Digest_Reset(&uninitialized) })
	testify.Panics(t, func() { adler32.Digest_Sum_32(&uninitialized) })
	testify.Panics(t, func() {
		var destination [adler32.DIGEST_SIZE]byte
		adler32.Digest_Sum_Into(&uninitialized, destination[:])
	})
	testify.Panics(t, func() {
		var destination [adler32.STATE_SIZE]byte
		adler32.Digest_Marshal_Into(&uninitialized, destination[:])
	})
	testify.Panics(t, func() {
		var destination adler32.Digest
		adler32.Digest_Clone_Into(&destination, &uninitialized)
	})
	testify.Equal(t, adler32.STATE_INPUT_STATUS_IDENTIFIER_INVALID,
		adler32.Digest_Unmarshal(&uninitialized, nil))

	var digest adler32.Digest
	adler32.Digest_Init(&digest)
	adler32.Digest_Clone_Into(&uninitialized, &digest)
	var source [adler32.SOURCE_SIZE_MAXIMUM + 1]byte
	testify.Panics(t, func() { adler32.Digest_Write(&digest, source[:]) })
	testify.Panics(t, func() { adler32.Checksum(source[:]) })
}

// Test_Invariant_Domains reaches range and enum sentinels through public operations.
func Test_Invariant_Domains(t *testing.T) {
	var source [adler32.SOURCE_SIZE_MAXIMUM]byte
	for _, size := range [...]int{0, 1, 2, adler32.SOURCE_SIZE_MAXIMUM} {
		var digest adler32.Digest
		adler32.Digest_Init(&digest)
		adler32.Digest_Write(&digest, source[:size])
		adler32.Checksum(source[:size])
	}
	adler32.Checksum(adler32.Source{1})
	const MAXIMUM_SOURCE_BYTE = bits.WORD_8_MAXIMUM - 3
	const MAXIMUM_SOURCE_SIZE = (adler32.MODULUS - 1) / (int(MAXIMUM_SOURCE_BYTE) / 2)
	var maximum_source [MAXIMUM_SOURCE_SIZE]byte
	for index := range maximum_source {
		maximum_source[index] = MAXIMUM_SOURCE_BYTE
	}
	adler32.Checksum(maximum_source[:])
	const MINIMUM_SOURCE_FULL_BYTES = (adler32.MODULUS - 1) / int(bits.WORD_8_MAXIMUM)
	const MINIMUM_SOURCE_REMAINDER = (adler32.MODULUS - 1) % int(bits.WORD_8_MAXIMUM)
	var minimum_source [MINIMUM_SOURCE_FULL_BYTES + 1]byte
	for index := range MINIMUM_SOURCE_FULL_BYTES {
		minimum_source[index] = bits.WORD_8_MAXIMUM
	}
	minimum_source[len(minimum_source)-1] = byte(MINIMUM_SOURCE_REMAINDER)
	adler32.Checksum(minimum_source[:])

	var output [adler32.DESTINATION_SIZE_MAXIMUM]byte
	var digest adler32.Digest
	adler32.Digest_Init(&digest)
	for _, size := range [...]int{0, 1, 2, adler32.DESTINATION_SIZE_MAXIMUM} {
		adler32.Digest_Sum_Into(&digest, output[:size])
		adler32.Digest_Marshal_Into(&digest, output[:size])
		adler32.Digest_Unmarshal(&digest, source[:size])
	}
	states := [...][adler32.STATE_SIZE]byte{
		{'a', 'd', 'l', 1, 0, 0, 0, 0},
		{'a', 'd', 'l', 1, 0, 0, 0, 1},
		{'a', 'd', 'l', 1, 0, 0, 0, 2},
		{'a', 'd', 'l', 1, 0xff, 0xff, 0xff, 0xff},
	}
	for _, state := range states {
		adler32.Digest_Unmarshal(&digest, state[:])
		adler32.Digest_Unmarshal(&digest, state[:])
		adler32.Digest_Write(&digest, nil)
		adler32.Digest_Sum_32(&digest)
		adler32.Digest_Sum_Into(&digest, output[:])
		adler32.Digest_Marshal_Into(&digest, output[:])
		var clone adler32.Digest
		adler32.Digest_Unmarshal(&clone, state[:])
		adler32.Digest_Clone_Into(&clone, &digest)
		adler32.Digest_Init(&clone)
		adler32.Digest_Reset(&digest)
	}
	adler32.Digest_Unmarshal(&digest, nil)
}

// Test_Allocation proves every runtime path owns no heap storage.
func Test_Allocation(t *testing.T) {
	// Buffers made here, outside every measured closure, so each closure sees
	// caller storage and only production allocation would register.
	fixture := allocation_fixture{
		Source: adler32.Source("abc"),
		Output: make(adler32.Destination, adler32.DIGEST_SIZE),
		State:  make(adler32.Destination, adler32.STATE_SIZE),
	}
	adler32.Digest_Init(&fixture.Digest)
	testify.Zero_Allocation(t, func() { adler32.Digest_Init(&fixture.Digest) })
	testify.Zero_Allocation(t, func() {
		fixture.Count = adler32.Digest_Write(&fixture.Digest, fixture.Source)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Digest_32 = adler32.Digest_Sum_32(&fixture.Digest)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Output_Count, fixture.Output_Status = adler32.Digest_Sum_Into(
			&fixture.Digest, fixture.Output,
		)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Value = adler32.Checksum(fixture.Source)
	})
	testify.Zero_Allocation(t, func() {
		fixture.State_Count, fixture.State_Output_Status = adler32.Digest_Marshal_Into(
			&fixture.Digest, fixture.State,
		)
	})
	testify.Zero_Allocation(t, func() {
		fixture.State_Input_Status = adler32.Digest_Unmarshal(
			&fixture.Clone_Digest, adler32.Source(fixture.State),
		)
	})
	testify.Zero_Allocation(t, func() {
		adler32.Digest_Clone_Into(&fixture.Clone_Digest, &fixture.Digest)
	})
	testify.True(t, fixture.Digest_32 >= 0)
}

type allocation_fixture struct {
	Digest              adler32.Digest
	Clone_Digest        adler32.Digest
	Source              adler32.Source
	Output              adler32.Destination
	State               adler32.Destination
	Count               adler32.Count
	Output_Count        adler32.Output_Count
	Output_Status       adler32.Output_Status
	State_Count         adler32.State_Count
	State_Output_Status adler32.State_Output_Status
	State_Input_Status  adler32.State_Input_Status
	Digest_32           adler32.Digest_Value
	Value               adler32.Value
}

// Reference checksum reduces every byte, unlike bounded production loop.
func reference_checksum(source []byte) (checksum adler32.Value) {
	sum_1 := uint32(1)
	sum_2 := uint32(0)
	for _, value := range source {
		sum_1 = (sum_1 + uint32(value)) % adler32.MODULUS
		sum_2 = (sum_2 + sum_1) % adler32.MODULUS
	}
	return adler32.Value{Sum_1: adler32.Sum_1(sum_1), Sum_2: adler32.Sum_2(sum_2)}
}

func reference_value(encoded uint32) (value adler32.Value) {
	return adler32.Value{
		Sum_1: adler32.Sum_1(encoded & uint32(bits.WORD_16_MAXIMUM)),
		Sum_2: adler32.Sum_2(encoded >> adler32.COMPONENT_SIZE),
	}
}
