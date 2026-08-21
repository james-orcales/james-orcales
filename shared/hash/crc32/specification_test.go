package crc32_test

import (
	"testing"

	"local/james-orcales/shared/hash/crc32"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/testify"
)

// Test_Package_Owned_State verifies streaming operations need no common dispatcher.
func Test_Package_Owned_State(t *testing.T) {
	var table crc32.Table
	crc32.Table_Make_Into(&table, crc32.IEEE)
	var digest crc32.Digest
	crc32.Digest_Init(&digest, &table)
	count := crc32.Digest_Write(&digest, crc32.Source("abc"))
	testify.Equal(t, crc32.Count(len("abc")), count)
	var output [crc32.DIGEST_SIZE]byte
	output_count, status := crc32.Digest_Sum_Into(&digest, output[:])
	testify.Equal(t, crc32.OUTPUT_COUNT_COMPLETE, output_count)
	testify.Equal(t, crc32.OUTPUT_STATUS_OK, status)
	crc32.Digest_Reset(&digest)
	testify.Equal(t, crc32.Digest_Value(0), crc32.Digest_Sum_32(&digest))
}

// Test_Reference_Values keeps both standard polynomials tied to known answers.
func Test_Reference_Values(t *testing.T) {
	tests := [...]struct {
		Text       string
		IEEE       crc32.Digest_Value
		Castagnoli crc32.Digest_Value
	}{
		{Text: "", IEEE: 0, Castagnoli: 0},
		{Text: "a", IEEE: 0xe8b7be43, Castagnoli: 0xc1d04330},
		{Text: "abc", IEEE: 0x352441c2, Castagnoli: 0x364b3fb7},
		{Text: "123456789", IEEE: 0xcbf43926, Castagnoli: 0xe3069283},
	}
	var ieee crc32.Table
	crc32.Table_Make_Into(&ieee, crc32.IEEE)
	var castagnoli crc32.Table
	crc32.Table_Make_Into(&castagnoli, crc32.CASTAGNOLI)
	for _, test := range tests {
		source := crc32.Source(test.Text)
		testify.Equal(t, test.IEEE, crc32.Checksum(source, &ieee))
		testify.Equal(t, test.IEEE, crc32.Checksum_IEEE(source))
		testify.Equal(t, test.Castagnoli, crc32.Checksum(source, &castagnoli))
	}
}

// Test_Table matches an independent bit walk for custom polynomials.
func Test_Table(t *testing.T) {
	polynomials := [...]crc32.Polynomial{
		crc32.IEEE, crc32.CASTAGNOLI, crc32.KOOPMAN, 0xd5828281,
	}
	source := crc32.Source("table-driven reference")
	for _, polynomial := range polynomials {
		var table crc32.Table
		crc32.Table_Make_Into(&table, polynomial)
		testify.Equal(t, reference_checksum(source, polynomial),
			crc32.Checksum(source, &table))
	}
}

// Test_Streaming_And_Update protects write-division independence and seeded continuation.
func Test_Streaming_And_Update(t *testing.T) {
	var table crc32.Table
	crc32.Table_Make_Into(&table, crc32.IEEE)
	prefix := crc32.Source("bounded ")
	suffix := crc32.Source("crc32")
	want := crc32.Checksum(crc32.Source("bounded crc32"), &table)
	continued := crc32.Update(crc32.Checksum(prefix, &table), &table, suffix)
	testify.Equal(t, want, continued)

	var digest crc32.Digest
	crc32.Digest_Init(&digest, &table)
	crc32.Digest_Write(&digest, prefix)
	crc32.Digest_Write(&digest, suffix)
	testify.Equal(t, want, crc32.Digest_Sum_32(&digest))
	crc32.Digest_Reset(&digest)
	testify.Equal(t, crc32.Digest_Value(0), crc32.Digest_Sum_32(&digest))
}

// Test_Caller_Owned_Output keeps short output untouched and checksum state reusable.
func Test_Caller_Owned_Output(t *testing.T) {
	var table crc32.Table
	crc32.Table_Make_Into(&table, crc32.IEEE)
	var digest crc32.Digest
	crc32.Digest_Init(&digest, &table)
	crc32.Digest_Write(&digest, crc32.Source("abc"))
	var short [crc32.DIGEST_SIZE - 1]byte
	count, status := crc32.Digest_Sum_Into(&digest, short[:])
	testify.Equal(t, crc32.OUTPUT_COUNT_EMPTY, count)
	testify.Equal(t, crc32.OUTPUT_STATUS_TOO_SMALL, status)
	var output [crc32.DIGEST_SIZE]byte
	count, status = crc32.Digest_Sum_Into(&digest, output[:])
	testify.Equal(t, crc32.OUTPUT_COUNT_COMPLETE, count)
	testify.Equal(t, crc32.OUTPUT_STATUS_OK, status)
	testify.Equal(t, [crc32.DIGEST_SIZE]byte{0x35, 0x24, 0x41, 0xc2}, output)
}

// Test_State_Compatibility protects standard state bytes and rejects hostile replacement.
func Test_State_Compatibility(t *testing.T) {
	var ieee crc32.Table
	crc32.Table_Make_Into(&ieee, crc32.IEEE)
	var digest crc32.Digest
	crc32.Digest_Init(&digest, &ieee)
	crc32.Digest_Write(&digest, crc32.Source("ab"))

	var state [crc32.STATE_SIZE]byte
	count, status := crc32.Digest_Marshal_Into(&digest, state[:])
	testify.Equal(t, crc32.STATE_COUNT_COMPLETE, count)
	testify.Equal(t, crc32.STATE_OUTPUT_STATUS_OK, status)
	testify.Equal(t, [crc32.STATE_SIZE]byte{
		'c', 'r', 'c', 1, 0xca, 0x87, 0x91, 0x4d, 0x9e, 0x83, 0x48, 0x6d,
	}, state)

	var restored crc32.Digest
	crc32.Digest_Init(&restored, &ieee)
	testify.Equal(t, crc32.STATE_INPUT_STATUS_OK,
		crc32.Digest_Unmarshal(&restored, state[:]))
	testify.Equal(t, crc32.Digest_Sum_32(&digest), crc32.Digest_Sum_32(&restored))

	previous := restored
	testify.Equal(t, crc32.STATE_INPUT_STATUS_SIZE_INVALID,
		crc32.Digest_Unmarshal(&restored, state[:len(state)-1]))
	testify.Equal(t, previous, restored)
	state[0] ^= 1
	testify.Equal(t, crc32.STATE_INPUT_STATUS_IDENTIFIER_INVALID,
		crc32.Digest_Unmarshal(&restored, state[:]))
	testify.Equal(t, previous, restored)
	state[0] ^= 1

	var castagnoli crc32.Table
	crc32.Table_Make_Into(&castagnoli, crc32.CASTAGNOLI)
	var other crc32.Digest
	crc32.Digest_Init(&other, &castagnoli)
	other_before := other
	testify.Equal(t, crc32.STATE_INPUT_STATUS_TABLE_INVALID,
		crc32.Digest_Unmarshal(&other, state[:]))
	testify.Equal(t, other_before, other)

	var short [crc32.STATE_SIZE - 1]byte
	count, status = crc32.Digest_Marshal_Into(&digest, short[:])
	testify.Equal(t, crc32.STATE_COUNT_EMPTY, count)
	testify.Equal(t, crc32.STATE_OUTPUT_STATUS_TOO_SMALL, status)
}

// Test_Clone keeps table and checksum state caller-owned and independent.
func Test_Clone(t *testing.T) {
	var table crc32.Table
	crc32.Table_Make_Into(&table, crc32.IEEE)
	var source crc32.Digest
	crc32.Digest_Init(&source, &table)
	crc32.Digest_Write(&source, crc32.Source("ab"))
	var destination crc32.Digest
	crc32.Digest_Init(&destination, &table)
	crc32.Digest_Clone_Into(&destination, &source)
	crc32.Digest_Write(&source, crc32.Source("c"))
	testify.Equal(t, crc32.Digest_Value(0x352441c2), crc32.Digest_Sum_32(&source))
	testify.Equal(t, crc32.Digest_Value(0x9e83486d), crc32.Digest_Sum_32(&destination))
}

// Test_Bounds rejects caller storage beyond the repository byte boundary.
func Test_Bounds(t *testing.T) {
	var uninitialized_table crc32.Table
	var uninitialized_digest crc32.Digest
	testify.Panics(t, func() { crc32.Checksum(nil, &uninitialized_table) })
	testify.Panics(t, func() { crc32.Digest_Write(&uninitialized_digest, nil) })

	var table crc32.Table
	crc32.Table_Make_Into(&table, crc32.IEEE)
	var digest crc32.Digest
	crc32.Digest_Init(&digest, &table)
	var source [crc32.SOURCE_SIZE_MAXIMUM + 1]byte
	testify.Panics(t, func() { crc32.Checksum(source[:], &table) })
	testify.Panics(t, func() { crc32.Update(0, &table, source[:]) })
	testify.Panics(t, func() { crc32.Digest_Write(&digest, source[:]) })
}

// Test_Invariant_Domains reaches each full-width state and bounded-byte sentinel.
func Test_Invariant_Domains(t *testing.T) {
	var source [crc32.SOURCE_SIZE_MAXIMUM]byte
	var output [crc32.DESTINATION_SIZE_MAXIMUM]byte
	crc32_polynomial_domains(output[:])

	var ieee crc32.Table
	crc32.Table_Make_Into(&ieee, crc32.IEEE)
	for _, size := range [...]int{0, 1, 2, crc32.SOURCE_SIZE_MAXIMUM} {
		crc32.Checksum(source[:size], &ieee)
		crc32.Checksum_IEEE(source[:size])
		crc32.Update(0, &ieee, source[:size])
		var digest crc32.Digest
		crc32.Digest_Init(&digest, &ieee)
		crc32.Digest_Write(&digest, source[:size])
		crc32.Digest_Unmarshal(&digest, source[:size])
	}
	for _, size := range [...]int{0, 1, 2, crc32.DESTINATION_SIZE_MAXIMUM} {
		var digest crc32.Digest
		crc32.Digest_Init(&digest, &ieee)
		crc32.Digest_Sum_Into(&digest, output[:size])
		crc32.Digest_Marshal_Into(&digest, output[:size])
	}
	preimages := [...][crc32.DIGEST_SIZE]byte{
		{0x9d, 0x0a, 0xd9, 0x6d},
		{0xdc, 0x0c, 0xa8, 0xb6},
		{0x5e, 0x00, 0x4a, 0x00},
		{0xff, 0xff, 0xff, 0xff},
	}
	values := [...]crc32.Digest_Value{
		crc32.Digest_Value(bits.WORD_32_MINIMUM),
		crc32.Digest_Value(bits.WORD_32_MINIMUM + 1),
		crc32.Digest_Value(bits.WORD_32_MINIMUM + 1 + 1),
		crc32.Digest_Value(bits.WORD_32_MAXIMUM),
	}
	for index, preimage := range preimages {
		testify.Equal(t, values[index], crc32.Checksum(preimage[:], &ieee))
		testify.Equal(t, values[index], crc32.Checksum_IEEE(preimage[:]))
		testify.Equal(t, values[index], crc32.Update(0, &ieee, preimage[:]))
		testify.Equal(t, values[index], crc32.Update(values[index], &ieee, nil))
		var digest crc32.Digest
		crc32.Digest_Init(&digest, &ieee)
		crc32.Digest_Write(&digest, preimage[:])
		crc32.Digest_Write(&digest, nil)
		crc32.Digest_Sum_32(&digest)
		crc32.Digest_Sum_Into(&digest, output[:])
		var state [crc32.STATE_SIZE]byte
		crc32.Digest_Marshal_Into(&digest, state[:])
		var restored crc32.Digest
		crc32.Digest_Init(&restored, &ieee)
		crc32.Digest_Clone_Into(&restored, &digest)
		crc32.Digest_Unmarshal(&restored, state[:])
		var clone crc32.Digest
		crc32.Digest_Init(&clone, &ieee)
		crc32.Digest_Write(&clone, preimage[:])
		crc32.Digest_Clone_Into(&clone, &digest)
		crc32.Digest_Init(&clone, &ieee)
		crc32.Digest_Reset(&digest)
	}
}

// Test_Allocation proves table, one-shot, streaming, output, and clone paths own no heap storage.
func Test_Allocation(t *testing.T) {
	fixture := allocation_fixture{Source: crc32.Source("abc")}
	crc32.Table_Make_Into(&fixture.Table, crc32.IEEE)
	crc32.Digest_Init(&fixture.Digest, &fixture.Table)
	crc32.Digest_Init(&fixture.Clone, &fixture.Table)
	testify.Zero_Allocation(t, func() {
		crc32.Table_Make_Into(&fixture.Table, crc32.IEEE)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Value = crc32.Checksum(fixture.Source, &fixture.Table)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Value = crc32.Checksum_IEEE(fixture.Source)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Value = crc32.Update(fixture.Value, &fixture.Table, fixture.Source)
	})
	testify.Zero_Allocation(t, func() {
		crc32.Digest_Init(&fixture.Digest, &fixture.Table)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Count = crc32.Digest_Write(&fixture.Digest, fixture.Source)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Value = crc32.Digest_Sum_32(&fixture.Digest)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Output_Count, fixture.Output_Status = crc32.Digest_Sum_Into(
			&fixture.Digest, fixture.Output[:],
		)
	})
	testify.Zero_Allocation(t, func() {
		fixture.State_Count, fixture.State_Output_Status = crc32.Digest_Marshal_Into(
			&fixture.Digest, fixture.State[:],
		)
	})
	testify.Zero_Allocation(t, func() {
		fixture.State_Input_Status = crc32.Digest_Unmarshal(
			&fixture.Clone, fixture.State[:],
		)
	})
	testify.Zero_Allocation(t, func() { crc32.Digest_Reset(&fixture.Digest) })
	testify.Zero_Allocation(t, func() {
		crc32.Digest_Clone_Into(&fixture.Clone, &fixture.Digest)
	})
	testify.True(t, fixture.Value >= 0)
}

type allocation_fixture struct {
	Table               crc32.Table
	Digest              crc32.Digest
	Clone               crc32.Digest
	Source              crc32.Source
	Output              [crc32.DIGEST_SIZE]byte
	State               [crc32.STATE_SIZE]byte
	Count               crc32.Count
	Value               crc32.Digest_Value
	Output_Count        crc32.Output_Count
	Output_Status       crc32.Output_Status
	State_Count         crc32.State_Count
	State_Output_Status crc32.State_Output_Status
	State_Input_Status  crc32.State_Input_Status
}

func reference_checksum(
	source crc32.Source, polynomial crc32.Polynomial,
) (value crc32.Digest_Value) {
	checksum := uint32(bits.WORD_32_MAXIMUM)
	for _, source_byte := range source {
		checksum ^= uint32(source_byte)
		for range bits.BIT_COUNT_8_MAXIMUM {
			if checksum&1 == 0 {
				checksum >>= 1
				continue
			}
			checksum = checksum>>1 ^ uint32(polynomial)
		}
	}
	return crc32.Digest_Value(^checksum)
}

func crc32_polynomial_domains(output crc32.Destination) {
	polynomials := [...]crc32.Polynomial{
		crc32.Polynomial(bits.WORD_32_MINIMUM),
		crc32.Polynomial(bits.WORD_32_MINIMUM + 1),
		crc32.Polynomial(bits.WORD_32_MINIMUM + 1 + 1),
		crc32.Polynomial(bits.WORD_32_MAXIMUM),
	}
	var table crc32.Table
	for _, polynomial := range polynomials {
		crc32.Table_Make_Into(&table, polynomial)
		crc32.Checksum(nil, &table)
		crc32.Update(0, &table, nil)
		var digest crc32.Digest
		crc32.Digest_Init(&digest, &table)
		crc32.Digest_Init(&digest, &table)
		crc32.Digest_Write(&digest, nil)
		crc32.Digest_Sum_32(&digest)
		crc32.Digest_Sum_Into(&digest, output)
		var state [crc32.STATE_SIZE]byte
		crc32.Digest_Marshal_Into(&digest, state[:])
		crc32.Digest_Unmarshal(&digest, state[:])
		var clone crc32.Digest
		crc32.Digest_Init(&clone, &table)
		crc32.Digest_Clone_Into(&clone, &digest)
		crc32.Digest_Reset(&digest)
	}
	crc32.Table_Make_Into(&table, polynomials[0])
}
