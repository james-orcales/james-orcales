package crc64_test

import (
	"testing"

	"local/james-orcales/shared/hash/crc64"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/testify"
)

// Test_Package_Owned_State verifies streaming operations need no common dispatcher.
func Test_Package_Owned_State(t *testing.T) {
	var table crc64.Table
	crc64.Table_Make_Into(&table, crc64.ISO)
	var digest crc64.Digest
	crc64.Digest_Init(&digest, &table)
	count := crc64.Digest_Write(&digest, crc64.Source("abc"))
	testify.Equal(t, crc64.Count(len("abc")), count)
	var output [crc64.DIGEST_SIZE]byte
	output_count, status := crc64.Digest_Sum_Into(&digest, output[:])
	testify.Equal(t, crc64.OUTPUT_COUNT_COMPLETE, output_count)
	testify.Equal(t, crc64.OUTPUT_STATUS_OK, status)
	crc64.Digest_Reset(&digest)
	testify.Equal(t, crc64.Digest_Value(0), crc64.Digest_Sum_64(&digest))
}

// Test_Reference_Values keeps both standard polynomials tied to known answers.
func Test_Reference_Values(t *testing.T) {
	tests := [...]struct {
		Text string
		ISO  crc64.Digest_Value
		ECMA crc64.Digest_Value
	}{
		{Text: "", ISO: 0, ECMA: 0},
		{Text: "a", ISO: 0x3420000000000000, ECMA: 0x330284772e652b05},
		{Text: "abc", ISO: 0x3776c42000000000, ECMA: 0x2cd8094a1a277627},
		{Text: "abcdefghij", ISO: 0x7f5b6e21b002d367, ECMA: 0x32093a2ecd5773f4},
	}
	var iso crc64.Table
	crc64.Table_Make_Into(&iso, crc64.ISO)
	var ecma crc64.Table
	crc64.Table_Make_Into(&ecma, crc64.ECMA)
	for _, test := range tests {
		source := crc64.Source(test.Text)
		testify.Equal(t, test.ISO, crc64.Checksum(source, &iso))
		testify.Equal(t, test.ECMA, crc64.Checksum(source, &ecma))
	}
}

// Test_Table matches an independent bit walk for caller-selected polynomial.
func Test_Table(t *testing.T) {
	polynomials := [...]crc64.Polynomial{crc64.ISO, crc64.ECMA, 0x777}
	source := crc64.Source("table-driven reference")
	for _, polynomial := range polynomials {
		var table crc64.Table
		crc64.Table_Make_Into(&table, polynomial)
		testify.Equal(t, reference_checksum(source, polynomial),
			crc64.Checksum(source, &table))
	}
}

// Test_Streaming_And_Update protects write-division independence and seeded continuation.
func Test_Streaming_And_Update(t *testing.T) {
	var table crc64.Table
	crc64.Table_Make_Into(&table, crc64.ECMA)
	prefix := crc64.Source("bounded ")
	suffix := crc64.Source("crc64")
	want := crc64.Checksum(crc64.Source("bounded crc64"), &table)
	testify.Equal(t, want, crc64.Update(crc64.Checksum(prefix, &table), &table, suffix))
	var digest crc64.Digest
	crc64.Digest_Init(&digest, &table)
	crc64.Digest_Write(&digest, prefix)
	crc64.Digest_Write(&digest, suffix)
	testify.Equal(t, want, crc64.Digest_Sum_64(&digest))
	crc64.Digest_Reset(&digest)
	testify.Equal(t, crc64.Digest_Value(0), crc64.Digest_Sum_64(&digest))
}

// Test_Caller_Owned_Output keeps short output untouched and emits big-endian state.
func Test_Caller_Owned_Output(t *testing.T) {
	var table crc64.Table
	crc64.Table_Make_Into(&table, crc64.ECMA)
	var digest crc64.Digest
	crc64.Digest_Init(&digest, &table)
	crc64.Digest_Write(&digest, crc64.Source("abc"))
	var short [crc64.DIGEST_SIZE - 1]byte
	count, status := crc64.Digest_Sum_Into(&digest, short[:])
	testify.Equal(t, crc64.OUTPUT_COUNT_EMPTY, count)
	testify.Equal(t, crc64.OUTPUT_STATUS_TOO_SMALL, status)
	var output [crc64.DIGEST_SIZE]byte
	count, status = crc64.Digest_Sum_Into(&digest, output[:])
	testify.Equal(t, crc64.OUTPUT_COUNT_COMPLETE, count)
	testify.Equal(t, crc64.OUTPUT_STATUS_OK, status)
	testify.Equal(t,
		[crc64.DIGEST_SIZE]byte{0x2c, 0xd8, 0x09, 0x4a, 0x1a, 0x27, 0x76, 0x27}, output)
}

// Test_State_Compatibility protects standard state bytes and rejects hostile replacement.
func Test_State_Compatibility(t *testing.T) {
	var iso crc64.Table
	crc64.Table_Make_Into(&iso, crc64.ISO)
	var digest crc64.Digest
	crc64.Digest_Init(&digest, &iso)
	crc64.Digest_Write(&digest, crc64.Source("ab"))

	var state [crc64.STATE_SIZE]byte
	count, status := crc64.Digest_Marshal_Into(&digest, state[:])
	testify.Equal(t, crc64.STATE_COUNT_COMPLETE, count)
	testify.Equal(t, crc64.STATE_OUTPUT_STATUS_OK, status)
	testify.Equal(t, [crc64.STATE_SIZE]byte{
		'c', 'r', 'c', 2,
		0x73, 0xba, 0x84, 0x84, 0xbb, 0xcd, 0x5d, 0xef,
		0x36, 0xc4, 0x20, 0, 0, 0, 0, 0,
	}, state)

	var restored crc64.Digest
	crc64.Digest_Init(&restored, &iso)
	testify.Equal(t, crc64.STATE_INPUT_STATUS_OK,
		crc64.Digest_Unmarshal(&restored, state[:]))
	testify.Equal(t, crc64.Digest_Sum_64(&digest), crc64.Digest_Sum_64(&restored))

	previous := restored
	testify.Equal(t, crc64.STATE_INPUT_STATUS_SIZE_INVALID,
		crc64.Digest_Unmarshal(&restored, state[:len(state)-1]))
	testify.Equal(t, previous, restored)
	state[0] ^= 1
	testify.Equal(t, crc64.STATE_INPUT_STATUS_IDENTIFIER_INVALID,
		crc64.Digest_Unmarshal(&restored, state[:]))
	testify.Equal(t, previous, restored)
	state[0] ^= 1

	var ecma crc64.Table
	crc64.Table_Make_Into(&ecma, crc64.ECMA)
	var other crc64.Digest
	crc64.Digest_Init(&other, &ecma)
	other_before := other
	testify.Equal(t, crc64.STATE_INPUT_STATUS_TABLE_INVALID,
		crc64.Digest_Unmarshal(&other, state[:]))
	testify.Equal(t, other_before, other)

	var short [crc64.STATE_SIZE - 1]byte
	count, status = crc64.Digest_Marshal_Into(&digest, short[:])
	testify.Equal(t, crc64.STATE_COUNT_EMPTY, count)
	testify.Equal(t, crc64.STATE_OUTPUT_STATUS_TOO_SMALL, status)
}

// Test_Clone keeps table and checksum state caller-owned and independent.
func Test_Clone(t *testing.T) {
	var table crc64.Table
	crc64.Table_Make_Into(&table, crc64.ISO)
	var source crc64.Digest
	crc64.Digest_Init(&source, &table)
	crc64.Digest_Write(&source, crc64.Source("ab"))
	var destination crc64.Digest
	crc64.Digest_Init(&destination, &table)
	crc64.Digest_Clone_Into(&destination, &source)
	crc64.Digest_Write(&source, crc64.Source("c"))
	testify.Equal(t, crc64.Digest_Value(0x3776c42000000000), crc64.Digest_Sum_64(&source))
	testify.Equal(t, crc64.Digest_Value(0x36c4200000000000),
		crc64.Digest_Sum_64(&destination))
}

// Test_Bounds rejects caller storage beyond the repository byte boundary.
func Test_Bounds(t *testing.T) {
	var uninitialized_table crc64.Table
	var uninitialized_digest crc64.Digest
	testify.Panics(t, func() { crc64.Checksum(nil, &uninitialized_table) })
	testify.Panics(t, func() { crc64.Digest_Write(&uninitialized_digest, nil) })

	var table crc64.Table
	crc64.Table_Make_Into(&table, crc64.ECMA)
	var digest crc64.Digest
	crc64.Digest_Init(&digest, &table)
	var source [crc64.SOURCE_SIZE_MAXIMUM + 1]byte
	testify.Panics(t, func() { crc64.Checksum(source[:], &table) })
	testify.Panics(t, func() { crc64.Update(0, &table, source[:]) })
	testify.Panics(t, func() { crc64.Digest_Write(&digest, source[:]) })
}

// Test_Invariant_Domains reaches each full-width state and bounded-byte sentinel.
func Test_Invariant_Domains(t *testing.T) {
	var source [crc64.SOURCE_SIZE_MAXIMUM]byte
	var output [crc64.DESTINATION_SIZE_MAXIMUM]byte
	crc64_polynomial_domains(output[:])

	var ecma crc64.Table
	crc64.Table_Make_Into(&ecma, crc64.ECMA)
	for _, size := range [...]int{0, 1, 2, crc64.SOURCE_SIZE_MAXIMUM} {
		crc64.Checksum(source[:size], &ecma)
		crc64.Update(0, &ecma, source[:size])
		var digest crc64.Digest
		crc64.Digest_Init(&digest, &ecma)
		crc64.Digest_Write(&digest, source[:size])
		crc64.Digest_Unmarshal(&digest, source[:size])
	}
	for _, size := range [...]int{0, 1, 2, crc64.DESTINATION_SIZE_MAXIMUM} {
		var digest crc64.Digest
		crc64.Digest_Init(&digest, &ecma)
		crc64.Digest_Sum_Into(&digest, output[:size])
		crc64.Digest_Marshal_Into(&digest, output[:size])
	}
	preimages := [...][crc64.DIGEST_SIZE]byte{
		{0x05, 0xca, 0x60, 0x8a, 0xe7, 0x13, 0x6a, 0x0f},
		{0x80, 0xd4, 0x6e, 0x25, 0xcc, 0xbc, 0xb2, 0x9d},
		{0x8a, 0xe9, 0x72, 0x7b, 0x9b, 0xe2, 0x03, 0xb8},
		{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
	}
	values := [...]crc64.Digest_Value{
		crc64.Digest_Value(bits.WORD_64_MINIMUM),
		crc64.Digest_Value(bits.WORD_64_MINIMUM + 1),
		crc64.Digest_Value(bits.WORD_64_MINIMUM + 1 + 1),
		crc64.Digest_Value(bits.WORD_64_MAXIMUM),
	}
	for index, preimage := range preimages {
		testify.Equal(t, values[index], crc64.Checksum(preimage[:], &ecma))
		testify.Equal(t, values[index], crc64.Update(values[index], &ecma, nil))
		var digest crc64.Digest
		crc64.Digest_Init(&digest, &ecma)
		crc64.Digest_Write(&digest, preimage[:])
		crc64.Digest_Write(&digest, nil)
		crc64.Digest_Sum_64(&digest)
		crc64.Digest_Sum_Into(&digest, output[:])
		var state [crc64.STATE_SIZE]byte
		crc64.Digest_Marshal_Into(&digest, state[:])
		var restored crc64.Digest
		crc64.Digest_Init(&restored, &ecma)
		crc64.Digest_Clone_Into(&restored, &digest)
		crc64.Digest_Unmarshal(&restored, state[:])
		var clone crc64.Digest
		crc64.Digest_Init(&clone, &ecma)
		crc64.Digest_Write(&clone, preimage[:])
		crc64.Digest_Clone_Into(&clone, &digest)
		crc64.Digest_Init(&clone, &ecma)
		crc64.Digest_Reset(&digest)
	}
}

// Test_Allocation proves table, one-shot, streaming, output, and clone paths own no heap storage.
func Test_Allocation(t *testing.T) {
	fixture := allocation_fixture{Source: crc64.Source("abc")}
	crc64.Table_Make_Into(&fixture.Table, crc64.ECMA)
	crc64.Digest_Init(&fixture.Digest, &fixture.Table)
	crc64.Digest_Init(&fixture.Clone, &fixture.Table)
	testify.Zero_Allocation(t, func() {
		crc64.Table_Make_Into(&fixture.Table, crc64.ECMA)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Value = crc64.Checksum(fixture.Source, &fixture.Table)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Value = crc64.Update(fixture.Value, &fixture.Table, fixture.Source)
	})
	testify.Zero_Allocation(t, func() {
		crc64.Digest_Init(&fixture.Digest, &fixture.Table)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Count = crc64.Digest_Write(&fixture.Digest, fixture.Source)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Value = crc64.Digest_Sum_64(&fixture.Digest)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Output_Count, fixture.Output_Status = crc64.Digest_Sum_Into(
			&fixture.Digest, fixture.Output[:],
		)
	})
	testify.Zero_Allocation(t, func() {
		fixture.State_Count, fixture.State_Output_Status = crc64.Digest_Marshal_Into(
			&fixture.Digest, fixture.State[:],
		)
	})
	testify.Zero_Allocation(t, func() {
		fixture.State_Input_Status = crc64.Digest_Unmarshal(
			&fixture.Clone, fixture.State[:],
		)
	})
	testify.Zero_Allocation(t, func() { crc64.Digest_Reset(&fixture.Digest) })
	testify.Zero_Allocation(t, func() {
		crc64.Digest_Clone_Into(&fixture.Clone, &fixture.Digest)
	})
	testify.True(t, fixture.Value >= 0)
}

type allocation_fixture struct {
	Table               crc64.Table
	Digest              crc64.Digest
	Clone               crc64.Digest
	Source              crc64.Source
	Output              [crc64.DIGEST_SIZE]byte
	State               [crc64.STATE_SIZE]byte
	Count               crc64.Count
	Value               crc64.Digest_Value
	Output_Count        crc64.Output_Count
	Output_Status       crc64.Output_Status
	State_Count         crc64.State_Count
	State_Output_Status crc64.State_Output_Status
	State_Input_Status  crc64.State_Input_Status
}

func reference_checksum(
	source crc64.Source, polynomial crc64.Polynomial,
) (value crc64.Digest_Value) {
	checksum := uint64(bits.WORD_64_MAXIMUM)
	for _, source_byte := range source {
		checksum ^= uint64(source_byte)
		for range bits.BIT_COUNT_8_MAXIMUM {
			if checksum&1 == 0 {
				checksum >>= 1
				continue
			}
			checksum = checksum>>1 ^ uint64(polynomial)
		}
	}
	return crc64.Digest_Value(^checksum)
}

func crc64_polynomial_domains(output crc64.Destination) {
	polynomials := [...]crc64.Polynomial{
		crc64.Polynomial(bits.WORD_64_MINIMUM),
		crc64.Polynomial(bits.WORD_64_MINIMUM + 1),
		crc64.Polynomial(bits.WORD_64_MINIMUM + 1 + 1),
		crc64.Polynomial(bits.WORD_64_MAXIMUM),
	}
	var table crc64.Table
	for _, polynomial := range polynomials {
		crc64.Table_Make_Into(&table, polynomial)
		crc64.Checksum(nil, &table)
		crc64.Update(0, &table, nil)
		var digest crc64.Digest
		crc64.Digest_Init(&digest, &table)
		crc64.Digest_Init(&digest, &table)
		crc64.Digest_Write(&digest, nil)
		crc64.Digest_Sum_64(&digest)
		crc64.Digest_Sum_Into(&digest, output)
		var state [crc64.STATE_SIZE]byte
		crc64.Digest_Marshal_Into(&digest, state[:])
		crc64.Digest_Unmarshal(&digest, state[:])
		var clone crc64.Digest
		crc64.Digest_Init(&clone, &table)
		crc64.Digest_Clone_Into(&clone, &digest)
		crc64.Digest_Reset(&digest)
	}
	crc64.Table_Make_Into(&table, polynomials[0])
}
