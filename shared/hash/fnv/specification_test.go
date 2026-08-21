package fnv_test

import (
	"testing"

	"local/james-orcales/shared/encoding/binary"
	"local/james-orcales/shared/hash/fnv"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/testify"
)

// Test_Package_Owned_State verifies streaming operations need no common dispatcher.
func Test_Package_Owned_State(t *testing.T) {
	// Boolean backing makes third lifecycle state unrepresentable.
	testify.False(t, bool(fnv.READY_EMPTY))
	testify.True(t, bool(fnv.READY_COMPLETE))
	var digest fnv.Digest_32
	fnv.Digest_32_Init(&digest, fnv.KIND_1A)
	count := fnv.Digest_32_Write(&digest, fnv.Source("abc"))
	testify.Equal(t, fnv.Count(len("abc")), count)
	var output [fnv.DIGEST_32_SIZE]byte
	sum_output := fnv.Digest_32_Sum_Into(&digest, output[:])
	testify.Equal(t, fnv.Output_32{
		Count: fnv.OUTPUT_32_COUNT_COMPLETE, Status: fnv.OUTPUT_STATUS_OK,
	}, sum_output)
}

// Test_Reference_Values keeps all six standard algorithms tied to known answers.
func Test_Reference_Values(t *testing.T) {
	tests := [...]struct {
		Kind      fnv.Kind
		Value_32  fnv.Value_32
		Value_64  fnv.Value_64
		Value_128 fnv.Value_128
	}{
		{
			Kind: fnv.KIND_1, Value_32: 0x439c2f4b, Value_64: 0xd8dcca186bafadcb,
			Value_128: fnv.Value_128{High: 0xa68bb2a4348b5822, Low: 0x836dbc78c6aee73b},
		},
		{
			Kind: fnv.KIND_1A, Value_32: 0x1a47e90b, Value_64: 0xe71fa2190541574b,
			Value_128: fnv.Value_128{High: 0xa68d622cec8b5822, Low: 0x836dbc7977af7f3b},
		},
	}
	for _, test := range tests {
		var digest_32 fnv.Digest_32
		fnv.Digest_32_Init(&digest_32, test.Kind)
		fnv.Digest_32_Write(&digest_32, fnv.Source("a"))
		fnv.Digest_32_Write(&digest_32, fnv.Source("bc"))
		testify.Equal(t, test.Value_32, fnv.Digest_32_Sum(&digest_32))

		var digest_64 fnv.Digest_64
		fnv.Digest_64_Init(&digest_64, test.Kind)
		fnv.Digest_64_Write(&digest_64, fnv.Source("a"))
		fnv.Digest_64_Write(&digest_64, fnv.Source("bc"))
		testify.Equal(t, test.Value_64, fnv.Digest_64_Sum(&digest_64))

		var digest_128 fnv.Digest_128
		fnv.Digest_128_Init(&digest_128, test.Kind)
		fnv.Digest_128_Write(&digest_128, fnv.Source("a"))
		fnv.Digest_128_Write(&digest_128, fnv.Source("bc"))
		testify.Equal(t, test.Value_128, fnv.Digest_128_Sum(&digest_128))
	}
}

// Test_Offsets_And_Reset protects width-specific initial state and Kind retention.
func Test_Offsets_And_Reset(t *testing.T) {
	for _, kind := range [...]fnv.Kind{fnv.KIND_1, fnv.KIND_1A} {
		var digest_32 fnv.Digest_32
		fnv.Digest_32_Init(&digest_32, kind)
		testify.Equal(t, fnv.Value_32(fnv.OFFSET_32), fnv.Digest_32_Sum(&digest_32))
		fnv.Digest_32_Write(&digest_32, fnv.Source("used"))
		fnv.Digest_32_Reset(&digest_32)
		testify.Equal(t, fnv.Value_32(fnv.OFFSET_32), fnv.Digest_32_Sum(&digest_32))

		var digest_64 fnv.Digest_64
		fnv.Digest_64_Init(&digest_64, kind)
		testify.Equal(t, fnv.Value_64(fnv.OFFSET_64), fnv.Digest_64_Sum(&digest_64))
		fnv.Digest_64_Write(&digest_64, fnv.Source("used"))
		fnv.Digest_64_Reset(&digest_64)
		testify.Equal(t, fnv.Value_64(fnv.OFFSET_64), fnv.Digest_64_Sum(&digest_64))

		var digest_128 fnv.Digest_128
		fnv.Digest_128_Init(&digest_128, kind)
		testify.Equal(t,
			fnv.Value_128{High: fnv.OFFSET_128_HIGH, Low: fnv.OFFSET_128_LOW},
			fnv.Digest_128_Sum(&digest_128))
		fnv.Digest_128_Write(&digest_128, fnv.Source("used"))
		fnv.Digest_128_Reset(&digest_128)
		testify.Equal(t,
			fnv.Value_128{High: fnv.OFFSET_128_HIGH, Low: fnv.OFFSET_128_LOW},
			fnv.Digest_128_Sum(&digest_128))
	}
}

// Test_Caller_Owned_Output emits each width big-endian and rejects partial output.
func Test_Caller_Owned_Output(t *testing.T) {
	var digest_32 fnv.Digest_32
	fnv.Digest_32_Init(&digest_32, fnv.KIND_1A)
	fnv.Digest_32_Write(&digest_32, fnv.Source("abc"))
	var short_32 [fnv.DIGEST_32_SIZE - 1]byte
	sum_output_32 := fnv.Digest_32_Sum_Into(&digest_32, short_32[:])
	testify.Equal(t, fnv.Output_32{
		Count: fnv.OUTPUT_32_COUNT_EMPTY, Status: fnv.OUTPUT_STATUS_TOO_SMALL,
	}, sum_output_32)
	var output_32 [fnv.DIGEST_32_SIZE]byte
	sum_output_32 = fnv.Digest_32_Sum_Into(&digest_32, output_32[:])
	testify.Equal(t, fnv.Output_32{
		Count: fnv.OUTPUT_32_COUNT_COMPLETE, Status: fnv.OUTPUT_STATUS_OK,
	}, sum_output_32)
	testify.Equal(t, [fnv.DIGEST_32_SIZE]byte{0x1a, 0x47, 0xe9, 0x0b}, output_32)

	var digest_64 fnv.Digest_64
	fnv.Digest_64_Init(&digest_64, fnv.KIND_1A)
	fnv.Digest_64_Write(&digest_64, fnv.Source("abc"))
	var output_64 [fnv.DIGEST_64_SIZE]byte
	sum_output_64 := fnv.Digest_64_Sum_Into(&digest_64, output_64[:])
	testify.Equal(t, fnv.Output_64{
		Count: fnv.OUTPUT_64_COUNT_COMPLETE, Status: fnv.OUTPUT_STATUS_OK,
	}, sum_output_64)
	testify.Equal(t,
		[fnv.DIGEST_64_SIZE]byte{0xe7, 0x1f, 0xa2, 0x19, 0x05, 0x41, 0x57, 0x4b}, output_64)

	var digest_128 fnv.Digest_128
	fnv.Digest_128_Init(&digest_128, fnv.KIND_1A)
	fnv.Digest_128_Write(&digest_128, fnv.Source("abc"))
	var output_128 [fnv.DIGEST_128_SIZE]byte
	sum_output_128 := fnv.Digest_128_Sum_Into(&digest_128, output_128[:])
	testify.Equal(t, fnv.Output_128{
		Count: fnv.OUTPUT_128_COUNT_COMPLETE, Status: fnv.OUTPUT_STATUS_OK,
	}, sum_output_128)
	testify.Equal(t, [fnv.DIGEST_128_SIZE]byte{
		0xa6, 0x8d, 0x62, 0x2c, 0xec, 0x8b, 0x58, 0x22,
		0x83, 0x6d, 0xbc, 0x79, 0x77, 0xaf, 0x7f, 0x3b,
	}, output_128)
}

// Test_State_Compatibility protects all six standard FNV state identities.
func Test_State_Compatibility(t *testing.T) {
	state_32_test(t, fnv.KIND_1, fnv.Source{
		'f', 'n', 'v', 1, 0x05, 0x0c, 0x5d, 0x7e,
	})
	state_32_test(t, fnv.KIND_1A, fnv.Source{
		'f', 'n', 'v', 2, 0xe4, 0x0c, 0x29, 0x2c,
	})
	state_64_test(t, fnv.KIND_1, fnv.Source{
		'f', 'n', 'v', 3, 0xaf, 0x63, 0xbd, 0x4c, 0x86, 0x01, 0xb7, 0xbe,
	})
	state_64_test(t, fnv.KIND_1A, fnv.Source{
		'f', 'n', 'v', 4, 0xaf, 0x63, 0xdc, 0x4c, 0x86, 0x01, 0xec, 0x8c,
	})
	state_128_test(t, fnv.KIND_1, fnv.Source{
		'f', 'n', 'v', 5,
		0xd2, 0x28, 0xcb, 0x69, 0x10, 0x1a, 0x8c, 0xaf,
		0x78, 0x91, 0x2b, 0x70, 0x4e, 0x4a, 0x14, 0x1e,
	})
	state_128_test(t, fnv.KIND_1A, fnv.Source{
		'f', 'n', 'v', 6,
		0xd2, 0x28, 0xcb, 0x69, 0x6f, 0x1a, 0x8c, 0xaf,
		0x78, 0x91, 0x2b, 0x70, 0x4e, 0x4a, 0x89, 0x64,
	})
}

// Test_Clone keeps each caller-owned state independent.
func Test_Clone(t *testing.T) {
	var source_32 fnv.Digest_32
	fnv.Digest_32_Init(&source_32, fnv.KIND_1A)
	fnv.Digest_32_Write(&source_32, fnv.Source("ab"))
	var clone_32 fnv.Digest_32
	fnv.Digest_32_Init(&clone_32, fnv.KIND_1)
	fnv.Digest_32_Clone_Into(&clone_32, &source_32)
	fnv.Digest_32_Write(&source_32, fnv.Source("c"))
	testify.Equal(t, fnv.Value_32(0x4d2505ca), fnv.Digest_32_Sum(&clone_32))

	var source_64 fnv.Digest_64
	fnv.Digest_64_Init(&source_64, fnv.KIND_1A)
	fnv.Digest_64_Write(&source_64, fnv.Source("ab"))
	var clone_64 fnv.Digest_64
	fnv.Digest_64_Init(&clone_64, fnv.KIND_1)
	fnv.Digest_64_Clone_Into(&clone_64, &source_64)
	fnv.Digest_64_Write(&source_64, fnv.Source("c"))
	testify.Equal(t, fnv.Value_64(0x089c4407b545986a), fnv.Digest_64_Sum(&clone_64))

	var source_128 fnv.Digest_128
	fnv.Digest_128_Init(&source_128, fnv.KIND_1A)
	fnv.Digest_128_Write(&source_128, fnv.Source("ab"))
	var clone_128 fnv.Digest_128
	fnv.Digest_128_Init(&clone_128, fnv.KIND_1)
	fnv.Digest_128_Clone_Into(&clone_128, &source_128)
	fnv.Digest_128_Write(&source_128, fnv.Source("c"))
	testify.Equal(t,
		fnv.Value_128{High: 0x08809544bbab1be9, Low: 0x5aa0733055b69a62},
		fnv.Digest_128_Sum(&clone_128))
}

// Test_Bounds rejects caller storage beyond the repository byte boundary.
func Test_Bounds(t *testing.T) {
	var uninitialized_32 fnv.Digest_32
	var uninitialized_64 fnv.Digest_64
	var uninitialized_128 fnv.Digest_128
	testify.Panics(t, func() { fnv.Digest_32_Write(&uninitialized_32, nil) })
	testify.Panics(t, func() { fnv.Digest_64_Write(&uninitialized_64, nil) })
	testify.Panics(t, func() { fnv.Digest_128_Write(&uninitialized_128, nil) })
	testify.Panics(t, func() { fnv.Digest_32_Reset(&uninitialized_32) })
	testify.Panics(t, func() { fnv.Digest_64_Reset(&uninitialized_64) })
	testify.Panics(t, func() { fnv.Digest_128_Reset(&uninitialized_128) })
	testify.Panics(t, func() { fnv.Digest_32_Sum(&uninitialized_32) })
	testify.Panics(t, func() { fnv.Digest_64_Sum(&uninitialized_64) })
	testify.Panics(t, func() { fnv.Digest_128_Sum(&uninitialized_128) })
	testify.Panics(t, func() {
		var destination [fnv.DIGEST_128_SIZE]byte
		fnv.Digest_32_Sum_Into(&uninitialized_32, destination[:])
	})
	testify.Panics(t, func() {
		var destination [fnv.DIGEST_128_SIZE]byte
		fnv.Digest_64_Sum_Into(&uninitialized_64, destination[:])
	})
	testify.Panics(t, func() {
		var destination [fnv.DIGEST_128_SIZE]byte
		fnv.Digest_128_Sum_Into(&uninitialized_128, destination[:])
	})
	testify.Panics(t, func() {
		var destination [fnv.STATE_128_SIZE]byte
		fnv.Digest_32_Marshal_Into(&uninitialized_32, destination[:])
	})
	testify.Panics(t, func() {
		var destination [fnv.STATE_128_SIZE]byte
		fnv.Digest_64_Marshal_Into(&uninitialized_64, destination[:])
	})
	testify.Panics(t, func() {
		var destination [fnv.STATE_128_SIZE]byte
		fnv.Digest_128_Marshal_Into(&uninitialized_128, destination[:])
	})
	testify.Panics(t, func() { fnv.Digest_32_Unmarshal(&uninitialized_32, nil) })
	testify.Panics(t, func() { fnv.Digest_64_Unmarshal(&uninitialized_64, nil) })
	testify.Panics(t, func() { fnv.Digest_128_Unmarshal(&uninitialized_128, nil) })
	testify.Panics(t, func() {
		var destination fnv.Digest_32
		fnv.Digest_32_Clone_Into(&destination, &uninitialized_32)
	})
	testify.Panics(t, func() {
		var destination fnv.Digest_64
		fnv.Digest_64_Clone_Into(&destination, &uninitialized_64)
	})
	testify.Panics(t, func() {
		var destination fnv.Digest_128
		fnv.Digest_128_Clone_Into(&destination, &uninitialized_128)
	})
	testify.Panics(t, func() {
		fnv.Digest_32_Init(&uninitialized_32, fnv.Kind(bits.WORD_8_MAXIMUM))
	})

	var source [fnv.SOURCE_SIZE_MAXIMUM + 1]byte
	var digest_32 fnv.Digest_32
	fnv.Digest_32_Init(&digest_32, fnv.KIND_1)
	fnv.Digest_32_Clone_Into(&uninitialized_32, &digest_32)
	testify.Panics(t, func() { fnv.Digest_32_Write(&digest_32, source[:]) })
	var digest_64 fnv.Digest_64
	fnv.Digest_64_Init(&digest_64, fnv.KIND_1)
	fnv.Digest_64_Clone_Into(&uninitialized_64, &digest_64)
	testify.Panics(t, func() { fnv.Digest_64_Write(&digest_64, source[:]) })
	var digest_128 fnv.Digest_128
	fnv.Digest_128_Init(&digest_128, fnv.KIND_1)
	fnv.Digest_128_Clone_Into(&uninitialized_128, &digest_128)
	testify.Panics(t, func() { fnv.Digest_128_Write(&digest_128, source[:]) })
}

// Test_Invariant_Domains reaches each value, Kind, and caller-byte sentinel.
func Test_Invariant_Domains(t *testing.T) {
	fnv_buffer_domains()
	fnv_32_value_domains(t)
	fnv_64_value_domains(t)
	fnv_128_value_domains(t)
}

// Test_Allocation proves every width and Kind path owns no heap storage.
func Test_Allocation(t *testing.T) {
	fixture := allocation_fixture{
		Source:    fnv.Source("abc"),
		Output:    make(fnv.Destination, fnv.DIGEST_128_SIZE),
		State_32:  make(fnv.Destination, fnv.STATE_32_SIZE),
		State_64:  make(fnv.Destination, fnv.STATE_64_SIZE),
		State_128: make(fnv.Destination, fnv.STATE_128_SIZE),
	}
	fnv.Digest_32_Init(&fixture.Digest_32, fnv.KIND_1A)
	fnv.Digest_64_Init(&fixture.Digest_64, fnv.KIND_1A)
	fnv.Digest_128_Init(&fixture.Digest_128, fnv.KIND_1A)
	fnv.Digest_32_Init(&fixture.Clone_32, fnv.KIND_1A)
	fnv.Digest_64_Init(&fixture.Clone_64, fnv.KIND_1A)
	fnv.Digest_128_Init(&fixture.Clone_128, fnv.KIND_1A)
	testify.Zero_Allocation(t, func() {
		fnv.Digest_32_Init(&fixture.Digest_32, fnv.KIND_1A)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Count = fnv.Digest_32_Write(&fixture.Digest_32, fixture.Source)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Value_32 = fnv.Digest_32_Sum(&fixture.Digest_32)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Output_32 = fnv.Digest_32_Sum_Into(
			&fixture.Digest_32, fixture.Output,
		)
	})
	testify.Zero_Allocation(t, func() {
		fixture.State_32_Output = fnv.Digest_32_Marshal_Into(
			&fixture.Digest_32, fixture.State_32,
		)
	})
	testify.Zero_Allocation(t, func() {
		fixture.State_Input_Status = fnv.Digest_32_Unmarshal(
			&fixture.Clone_32, fnv.Source(fixture.State_32),
		)
	})
	testify.Zero_Allocation(t, func() { fnv.Digest_32_Reset(&fixture.Digest_32) })
	testify.Zero_Allocation(t, func() {
		fnv.Digest_32_Clone_Into(&fixture.Clone_32, &fixture.Digest_32)
	})
	fnv_64_allocation(t, &fixture)
	fnv_128_allocation(t, &fixture)
	testify.True(t, fixture.Count >= 0)
}

func fnv_64_allocation(t *testing.T, fixture *allocation_fixture) {
	testify.Zero_Allocation(t, func() {
		fnv.Digest_64_Init(&fixture.Digest_64, fnv.KIND_1A)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Count = fnv.Digest_64_Write(&fixture.Digest_64, fixture.Source)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Value_64 = fnv.Digest_64_Sum(&fixture.Digest_64)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Output_64 = fnv.Digest_64_Sum_Into(
			&fixture.Digest_64, fixture.Output,
		)
	})
	testify.Zero_Allocation(t, func() {
		fixture.State_64_Output = fnv.Digest_64_Marshal_Into(
			&fixture.Digest_64, fixture.State_64,
		)
	})
	testify.Zero_Allocation(t, func() {
		fixture.State_Input_Status = fnv.Digest_64_Unmarshal(
			&fixture.Clone_64, fnv.Source(fixture.State_64),
		)
	})
	testify.Zero_Allocation(t, func() { fnv.Digest_64_Reset(&fixture.Digest_64) })
	testify.Zero_Allocation(t, func() {
		fnv.Digest_64_Clone_Into(&fixture.Clone_64, &fixture.Digest_64)
	})
}

func fnv_128_allocation(t *testing.T, fixture *allocation_fixture) {
	testify.Zero_Allocation(t, func() {
		fnv.Digest_128_Init(&fixture.Digest_128, fnv.KIND_1A)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Count = fnv.Digest_128_Write(&fixture.Digest_128, fixture.Source)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Value_128 = fnv.Digest_128_Sum(&fixture.Digest_128)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Output_128 = fnv.Digest_128_Sum_Into(
			&fixture.Digest_128, fixture.Output,
		)
	})
	testify.Zero_Allocation(t, func() {
		fixture.State_128_Output = fnv.Digest_128_Marshal_Into(
			&fixture.Digest_128, fixture.State_128,
		)
	})
	testify.Zero_Allocation(t, func() {
		fixture.State_Input_Status = fnv.Digest_128_Unmarshal(
			&fixture.Clone_128, fnv.Source(fixture.State_128),
		)
	})
	testify.Zero_Allocation(t, func() { fnv.Digest_128_Reset(&fixture.Digest_128) })
	testify.Zero_Allocation(t, func() {
		fnv.Digest_128_Clone_Into(&fixture.Clone_128, &fixture.Digest_128)
	})
}

type allocation_fixture struct {
	Digest_32          fnv.Digest_32
	Digest_64          fnv.Digest_64
	Digest_128         fnv.Digest_128
	Clone_32           fnv.Digest_32
	Clone_64           fnv.Digest_64
	Clone_128          fnv.Digest_128
	Source             fnv.Source
	Output             fnv.Destination
	State_32           fnv.Destination
	State_64           fnv.Destination
	State_128          fnv.Destination
	Count              fnv.Count
	Value_32           fnv.Value_32
	Value_64           fnv.Value_64
	Value_128          fnv.Value_128
	Output_32          fnv.Output_32
	Output_64          fnv.Output_64
	Output_128         fnv.Output_128
	State_32_Output    fnv.State_32_Output
	State_64_Output    fnv.State_64_Output
	State_128_Output   fnv.State_128_Output
	State_Input_Status fnv.State_Input_Status
}

func state_32_test(t *testing.T, kind fnv.Kind, want fnv.Source) {
	var digest fnv.Digest_32
	fnv.Digest_32_Init(&digest, kind)
	fnv.Digest_32_Write(&digest, fnv.Source("a"))
	var state [fnv.STATE_32_SIZE]byte
	state_output := fnv.Digest_32_Marshal_Into(&digest, state[:])
	testify.Equal(t, fnv.State_32_Output{
		Count: fnv.STATE_32_COUNT_COMPLETE, Status: fnv.STATE_OUTPUT_STATUS_OK,
	}, state_output)
	testify.Equal(t, want, fnv.Source(state[:]))

	var restored fnv.Digest_32
	fnv.Digest_32_Init(&restored, kind)
	testify.Equal(t, fnv.STATE_INPUT_STATUS_OK,
		fnv.Digest_32_Unmarshal(&restored, state[:]))
	testify.Equal(t, fnv.Digest_32_Sum(&digest), fnv.Digest_32_Sum(&restored))
	previous := restored
	testify.Equal(t, fnv.STATE_INPUT_STATUS_SIZE_INVALID,
		fnv.Digest_32_Unmarshal(&restored, state[:len(state)-1]))
	state[0] ^= 1
	testify.Equal(t, fnv.STATE_INPUT_STATUS_IDENTIFIER_INVALID,
		fnv.Digest_32_Unmarshal(&restored, state[:]))
	testify.Equal(t, previous, restored)
	var short [fnv.STATE_32_SIZE - 1]byte
	state_output = fnv.Digest_32_Marshal_Into(&digest, short[:])
	testify.Equal(t, fnv.State_32_Output{
		Count: fnv.STATE_32_COUNT_EMPTY, Status: fnv.STATE_OUTPUT_STATUS_TOO_SMALL,
	}, state_output)
}

func state_64_test(t *testing.T, kind fnv.Kind, want fnv.Source) {
	var digest fnv.Digest_64
	fnv.Digest_64_Init(&digest, kind)
	fnv.Digest_64_Write(&digest, fnv.Source("a"))
	var state [fnv.STATE_64_SIZE]byte
	state_output := fnv.Digest_64_Marshal_Into(&digest, state[:])
	testify.Equal(t, fnv.State_64_Output{
		Count: fnv.STATE_64_COUNT_COMPLETE, Status: fnv.STATE_OUTPUT_STATUS_OK,
	}, state_output)
	testify.Equal(t, want, fnv.Source(state[:]))

	var restored fnv.Digest_64
	fnv.Digest_64_Init(&restored, kind)
	testify.Equal(t, fnv.STATE_INPUT_STATUS_OK,
		fnv.Digest_64_Unmarshal(&restored, state[:]))
	testify.Equal(t, fnv.Digest_64_Sum(&digest), fnv.Digest_64_Sum(&restored))
	previous := restored
	testify.Equal(t, fnv.STATE_INPUT_STATUS_SIZE_INVALID,
		fnv.Digest_64_Unmarshal(&restored, state[:len(state)-1]))
	state[0] ^= 1
	testify.Equal(t, fnv.STATE_INPUT_STATUS_IDENTIFIER_INVALID,
		fnv.Digest_64_Unmarshal(&restored, state[:]))
	testify.Equal(t, previous, restored)
	var short [fnv.STATE_64_SIZE - 1]byte
	state_output = fnv.Digest_64_Marshal_Into(&digest, short[:])
	testify.Equal(t, fnv.State_64_Output{
		Count: fnv.STATE_64_COUNT_EMPTY, Status: fnv.STATE_OUTPUT_STATUS_TOO_SMALL,
	}, state_output)
}

func state_128_test(t *testing.T, kind fnv.Kind, want fnv.Source) {
	var digest fnv.Digest_128
	fnv.Digest_128_Init(&digest, kind)
	fnv.Digest_128_Write(&digest, fnv.Source("a"))
	var state [fnv.STATE_128_SIZE]byte
	state_output := fnv.Digest_128_Marshal_Into(&digest, state[:])
	testify.Equal(t, fnv.State_128_Output{
		Count: fnv.STATE_128_COUNT_COMPLETE, Status: fnv.STATE_OUTPUT_STATUS_OK,
	}, state_output)
	testify.Equal(t, want, fnv.Source(state[:]))

	var restored fnv.Digest_128
	fnv.Digest_128_Init(&restored, kind)
	testify.Equal(t, fnv.STATE_INPUT_STATUS_OK,
		fnv.Digest_128_Unmarshal(&restored, state[:]))
	testify.Equal(t, fnv.Digest_128_Sum(&digest), fnv.Digest_128_Sum(&restored))
	previous := restored
	testify.Equal(t, fnv.STATE_INPUT_STATUS_SIZE_INVALID,
		fnv.Digest_128_Unmarshal(&restored, state[:len(state)-1]))
	state[0] ^= 1
	testify.Equal(t, fnv.STATE_INPUT_STATUS_IDENTIFIER_INVALID,
		fnv.Digest_128_Unmarshal(&restored, state[:]))
	testify.Equal(t, previous, restored)
	var short [fnv.STATE_128_SIZE - 1]byte
	state_output = fnv.Digest_128_Marshal_Into(&digest, short[:])
	testify.Equal(t, fnv.State_128_Output{
		Count: fnv.STATE_128_COUNT_EMPTY, Status: fnv.STATE_OUTPUT_STATUS_TOO_SMALL,
	}, state_output)
}

func state_32_make(kind fnv.Kind, value fnv.Value_32) (state fnv.Source) {
	state = make(fnv.Source, fnv.STATE_32_SIZE)
	identity := fnv.STATE_32_1_IDENTITY
	if kind == fnv.KIND_1A {
		identity = fnv.STATE_32_1A_IDENTITY
	}
	copy(state[:fnv.STATE_IDENTITY_SIZE], identity)
	for index := range fnv.DIGEST_32_SIZE {
		shift := bits.BIT_COUNT_32_MAXIMUM - binary.BITS_PER_BYTE*(index+1)
		state[fnv.STATE_IDENTITY_SIZE+index] = byte(uint32(value) >> shift)
	}
	return state
}

func state_64_make(kind fnv.Kind, value fnv.Value_64) (state fnv.Source) {
	state = make(fnv.Source, fnv.STATE_64_SIZE)
	identity := fnv.STATE_64_1_IDENTITY
	if kind == fnv.KIND_1A {
		identity = fnv.STATE_64_1A_IDENTITY
	}
	copy(state[:fnv.STATE_IDENTITY_SIZE], identity)
	for index := range fnv.DIGEST_64_SIZE {
		shift := bits.BIT_COUNT_64_MAXIMUM - binary.BITS_PER_BYTE*(index+1)
		state[fnv.STATE_IDENTITY_SIZE+index] = byte(uint64(value) >> shift)
	}
	return state
}

func state_128_make(kind fnv.Kind, value fnv.Value_128) (state fnv.Source) {
	state = make(fnv.Source, fnv.STATE_128_SIZE)
	identity := fnv.STATE_128_1_IDENTITY
	if kind == fnv.KIND_1A {
		identity = fnv.STATE_128_1A_IDENTITY
	}
	copy(state[:fnv.STATE_IDENTITY_SIZE], identity)
	for index := range fnv.DIGEST_64_SIZE {
		shift := bits.BIT_COUNT_64_MAXIMUM - binary.BITS_PER_BYTE*(index+1)
		state[fnv.STATE_IDENTITY_SIZE+index] = byte(uint64(value.High) >> shift)
		state[fnv.STATE_IDENTITY_SIZE+fnv.DIGEST_64_SIZE+index] =
			byte(uint64(value.Low) >> shift)
	}
	return state
}

func fnv_buffer_domains() {
	var source [fnv.SOURCE_SIZE_MAXIMUM]byte
	var output [fnv.DESTINATION_SIZE_MAXIMUM]byte
	for _, size := range [...]int{0, 1, 2, fnv.SOURCE_SIZE_MAXIMUM} {
		for _, kind := range [...]fnv.Kind{fnv.KIND_1, fnv.KIND_1A} {
			var digest_32 fnv.Digest_32
			fnv.Digest_32_Init(&digest_32, kind)
			fnv.Digest_32_Write(&digest_32, source[:size])
			fnv.Digest_32_Unmarshal(&digest_32, source[:size])

			var digest_64 fnv.Digest_64
			fnv.Digest_64_Init(&digest_64, kind)
			fnv.Digest_64_Write(&digest_64, source[:size])
			fnv.Digest_64_Unmarshal(&digest_64, source[:size])

			var digest_128 fnv.Digest_128
			fnv.Digest_128_Init(&digest_128, kind)
			fnv.Digest_128_Write(&digest_128, source[:size])
			fnv.Digest_128_Unmarshal(&digest_128, source[:size])
		}
	}
	for _, size := range [...]int{0, 1, 2, fnv.DESTINATION_SIZE_MAXIMUM} {
		var digest_32 fnv.Digest_32
		fnv.Digest_32_Init(&digest_32, fnv.KIND_1)
		fnv.Digest_32_Sum_Into(&digest_32, output[:size])
		fnv.Digest_32_Marshal_Into(&digest_32, output[:size])
		var digest_64 fnv.Digest_64
		fnv.Digest_64_Init(&digest_64, fnv.KIND_1)
		fnv.Digest_64_Sum_Into(&digest_64, output[:size])
		fnv.Digest_64_Marshal_Into(&digest_64, output[:size])
		var digest_128 fnv.Digest_128
		fnv.Digest_128_Init(&digest_128, fnv.KIND_1)
		fnv.Digest_128_Sum_Into(&digest_128, output[:size])
		fnv.Digest_128_Marshal_Into(&digest_128, output[:size])
	}
}

func fnv_32_value_domains(t *testing.T) {
	values := [...]fnv.Value_32{
		fnv.Value_32(bits.WORD_32_MINIMUM), fnv.Value_32(bits.WORD_32_MINIMUM + 1),
		fnv.Value_32(bits.WORD_32_MINIMUM + 1 + 1), fnv.Value_32(bits.WORD_32_MAXIMUM),
	}
	var output [fnv.DESTINATION_SIZE_MAXIMUM]byte
	for _, kind := range [...]fnv.Kind{fnv.KIND_1, fnv.KIND_1A} {
		for _, value := range values {
			var digest fnv.Digest_32
			fnv.Digest_32_Init(&digest, kind)
			state := state_32_make(kind, value)
			testify.Equal(t, fnv.STATE_INPUT_STATUS_OK,
				fnv.Digest_32_Unmarshal(&digest, state[:]))
			fnv.Digest_32_Unmarshal(&digest, state[:])
			fnv.Digest_32_Write(&digest, nil)
			fnv.Digest_32_Sum(&digest)
			fnv.Digest_32_Sum_Into(&digest, output[:])
			fnv.Digest_32_Marshal_Into(&digest, output[:])
			var clone fnv.Digest_32
			fnv.Digest_32_Init(&clone, kind)
			fnv.Digest_32_Unmarshal(&clone, state[:])
			fnv.Digest_32_Clone_Into(&clone, &digest)
			fnv.Digest_32_Init(&clone, kind)
			fnv.Digest_32_Reset(&digest)
		}
	}
}

func fnv_64_value_domains(t *testing.T) {
	values := [...]fnv.Value_64{
		fnv.Value_64(bits.WORD_64_MINIMUM), fnv.Value_64(bits.WORD_64_MINIMUM + 1),
		fnv.Value_64(bits.WORD_64_MINIMUM + 1 + 1), fnv.Value_64(bits.WORD_64_MAXIMUM),
	}
	var output [fnv.DESTINATION_SIZE_MAXIMUM]byte
	for _, kind := range [...]fnv.Kind{fnv.KIND_1, fnv.KIND_1A} {
		for _, value := range values {
			var digest fnv.Digest_64
			fnv.Digest_64_Init(&digest, kind)
			state := state_64_make(kind, value)
			testify.Equal(t, fnv.STATE_INPUT_STATUS_OK,
				fnv.Digest_64_Unmarshal(&digest, state[:]))
			fnv.Digest_64_Unmarshal(&digest, state[:])
			fnv.Digest_64_Write(&digest, nil)
			fnv.Digest_64_Sum(&digest)
			fnv.Digest_64_Sum_Into(&digest, output[:])
			fnv.Digest_64_Marshal_Into(&digest, output[:])
			var clone fnv.Digest_64
			fnv.Digest_64_Init(&clone, kind)
			fnv.Digest_64_Unmarshal(&clone, state[:])
			fnv.Digest_64_Clone_Into(&clone, &digest)
			fnv.Digest_64_Init(&clone, kind)
			fnv.Digest_64_Reset(&digest)
		}
	}
}

func fnv_128_value_domains(t *testing.T) {
	values := [...]fnv.Value_128{
		{High: fnv.High(bits.WORD_64_MINIMUM), Low: fnv.Low(bits.WORD_64_MINIMUM)},
		{High: fnv.High(bits.WORD_64_MINIMUM + 1), Low: fnv.Low(bits.WORD_64_MINIMUM + 1)},
		{
			High: fnv.High(bits.WORD_64_MINIMUM + 1 + 1),
			Low:  fnv.Low(bits.WORD_64_MINIMUM + 1 + 1),
		},
		{High: fnv.High(bits.WORD_64_MAXIMUM), Low: fnv.Low(bits.WORD_64_MAXIMUM)},
	}
	var output [fnv.DESTINATION_SIZE_MAXIMUM]byte
	for _, kind := range [...]fnv.Kind{fnv.KIND_1, fnv.KIND_1A} {
		for _, value := range values {
			var digest fnv.Digest_128
			fnv.Digest_128_Init(&digest, kind)
			state := state_128_make(kind, value)
			testify.Equal(t, fnv.STATE_INPUT_STATUS_OK,
				fnv.Digest_128_Unmarshal(&digest, state[:]))
			fnv.Digest_128_Unmarshal(&digest, state[:])
			fnv.Digest_128_Write(&digest, nil)
			fnv.Digest_128_Sum(&digest)
			fnv.Digest_128_Sum_Into(&digest, output[:])
			fnv.Digest_128_Marshal_Into(&digest, output[:])
			var clone fnv.Digest_128
			fnv.Digest_128_Init(&clone, kind)
			fnv.Digest_128_Unmarshal(&clone, state[:])
			fnv.Digest_128_Clone_Into(&clone, &digest)
			fnv.Digest_128_Init(&clone, kind)
			fnv.Digest_128_Reset(&digest)
		}
	}
}
