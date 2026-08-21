package md5_test

import (
	standard_md5 "crypto/md5"
	"testing"

	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/crypto/md5"
	"local/james-orcales/shared/encoding/binary"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/testify"
)

// Test_Package_Owned_State checks lifecycle and fixed geometry without shared dispatch.
func Test_Package_Owned_State(t *testing.T) {
	var source md5.Digest
	md5.Digest_Init(&source)
	testify.Equal(t, md5.Size(md5.DIGEST_SIZE), md5.Digest_Size(&source))
	testify.Equal(t, md5.Block_Size(md5.BLOCK_SIZE), md5.Digest_Block_Size(&source))
	md5.Digest_Write(&source, md5.Source("abc"))
	var destination md5.Digest
	md5.Digest_Clone_Into(&destination, &source)
	testify.Equal(t, md5.Digest_Sum(&source), md5.Digest_Sum(&destination))
}

// Test_Formula binds every derived width to shared primitive geometry.
func Test_Formula(t *testing.T) {
	testify.Equal(t, md5.STATE_LANE_COUNT*bits.BIT_COUNT_32_MAXIMUM, md5.DIGEST_BIT_COUNT)
	testify.Equal(t, md5.DIGEST_BIT_COUNT/binary.BITS_PER_BYTE, md5.DIGEST_SIZE)
	testify.Equal(t, md5.MESSAGE_WORD_COUNT*binary.UINT_32_SIZE, md5.BLOCK_SIZE)
	testify.Equal(t, md5.BLOCK_SIZE*binary.BITS_PER_BYTE, md5.BLOCK_BIT_COUNT)
	testify.Equal(t, md5.STATE_LANE_COUNT+binary.UINT_8_SIZE, md5.STATE_WORD_COUNT)
	testify.Equal(t, md5.BLOCK_SIZE-md5.MESSAGE_BIT_COUNT_SIZE, md5.PADDING_BOUNDARY)
	testify.Equal(t, md5.BLOCK_SIZE*binary.UINT_16_SIZE, md5.FINAL_BLOCK_CAPACITY)
	testify.Equal(t, md5.BLOCK_SIZE-binary.UINT_8_SIZE, md5.BUFFER_COUNT_MAXIMUM)
	testify.Equal(t, bits.WORD_64_MAXIMUM/binary.BITS_PER_BYTE, md5.MESSAGE_SIZE_MAXIMUM)
	testify.Equal(t, md5.ROUND_QUARTER_COUNT*md5.MESSAGE_WORD_COUNT, int(md5.ROUND_COUNT))
	testify.Equal(t, ^md5.INITIAL_STATE_0, md5.INITIAL_STATE_2)
	testify.Equal(t, ^md5.INITIAL_STATE_1, md5.INITIAL_STATE_3)
}

// Test_Reference_Values binds output to RFC 1321 vectors.
func Test_Reference_Values(t *testing.T) {
	sources := [...]md5.Source{md5.Source(""), md5.Source("abc")}
	for _, source := range sources {
		value := md5.Value(standard_md5.Sum(source))
		testify.Equal(t, value, md5.Checksum(source))
		var digest md5.Digest
		md5.Digest_Init(&digest)
		midpoint := len(source) / TEST_COUNT_TWO
		testify.Equal(
			t, md5.Count(midpoint), md5.Digest_Write(&digest, source[:midpoint]),
		)
		testify.Equal(
			t, md5.Count(len(source)-midpoint),
			md5.Digest_Write(&digest, source[midpoint:]),
		)
		testify.Equal(t, value, md5.Digest_Sum(&digest))
	}
}

// Test_Stream checks direct blocks, buffered tails, and repeated bounded writes.
func Test_Stream(t *testing.T) {
	var source [md5.SOURCE_SIZE_MAXIMUM + TEST_COUNT_ONE]byte
	for index := range source {
		source[index] = byte(index)
	}
	for _, size := range [...]int{
		md5.PADDING_BOUNDARY - TEST_COUNT_ONE, md5.PADDING_BOUNDARY,
		md5.BLOCK_SIZE - TEST_COUNT_ONE, md5.BLOCK_SIZE,
		md5.BLOCK_SIZE + TEST_COUNT_ONE,
	} {
		testify.Equal(
			t, md5.Value(standard_md5.Sum(source[:size])), md5.Checksum(source[:size]),
		)
	}
	want := md5.Value(standard_md5.Sum(source[:]))
	var digest md5.Digest
	md5.Digest_Init(&digest)
	md5.Digest_Write(&digest, source[:md5.SOURCE_SIZE_MAXIMUM])
	md5.Digest_Write(&digest, source[md5.SOURCE_SIZE_MAXIMUM:])
	testify.Equal(t, want, md5.Digest_Sum(&digest))
	md5.Digest_Write(&digest, md5.Source("x"))
	testify.Equal(
		t, md5.Value(standard_md5.Sum(append(source[:], 'x'))), md5.Digest_Sum(&digest),
	)
	want_chunks := md5.Value(standard_md5.Sum(source[:md5.SOURCE_SIZE_MAXIMUM]))
	for _, chunk_size := range [...]int{
		TEST_COUNT_ONE, md5.BLOCK_SIZE - TEST_COUNT_ONE, md5.BLOCK_SIZE,
		md5.BLOCK_SIZE + TEST_COUNT_ONE, TEST_MULTI_BLOCK_CHUNK_SIZE,
	} {
		md5.Digest_Init(&digest)
		write_chunks(&digest, source[:md5.SOURCE_SIZE_MAXIMUM], chunk_size)
		testify.Equal(t, want_chunks, md5.Digest_Sum(&digest), chunk_size)
	}
}

// Test_Caller_Owned_Output keeps short storage untouched and Sum non-consuming.
func Test_Caller_Owned_Output(t *testing.T) {
	var digest md5.Digest
	md5.Digest_Init(&digest)
	md5.Digest_Write(&digest, md5.Source("abc"))
	before := digest
	var short [md5.DIGEST_SIZE - TEST_COUNT_ONE]byte
	count, status := md5.Digest_Sum_Into(&digest, short[:])
	testify.Equal(t, md5.OUTPUT_COUNT_REQUIRED, count)
	testify.Equal(t, md5.OUTPUT_STATUS_TOO_SMALL, status)
	testify.Equal(t, [md5.DIGEST_SIZE - TEST_COUNT_ONE]byte{}, short)
	var output [md5.DIGEST_SIZE]byte
	count, status = md5.Digest_Sum_Into(&digest, output[:])
	testify.Equal(t, md5.OUTPUT_COUNT_REQUIRED, count)
	testify.Equal(t, md5.OUTPUT_STATUS_OK, status)
	testify.Equal(t, md5.Value(output), md5.Digest_Sum(&digest))
	testify.Equal(t, before, digest)
	md5.Digest_Reset(&digest)
	testify.Equal(t, md5.Checksum(nil), md5.Digest_Sum(&digest))
}

// Test_Bounds rejects oversized calls and uninitialized caller state.
func Test_Bounds(t *testing.T) {
	var digest md5.Digest
	testify.Panics(t, func() { md5.Digest_Write(&digest, nil) })
	md5.Digest_Init(&digest)
	var source [md5.SOURCE_SIZE_MAXIMUM + TEST_COUNT_ONE]byte
	testify.Panics(t, func() { md5.Digest_Write(&digest, source[:]) })
	testify.Panics(t, func() { md5.Checksum(source[:]) })
	var destination [md5.DESTINATION_SIZE_MAXIMUM + TEST_COUNT_ONE]byte
	testify.Panics(t, func() { md5.Digest_Sum_Into(&digest, destination[:]) })
	digest.Message_Size = md5.Message_Size(md5.MESSAGE_SIZE_MAXIMUM)
	digest.Buffer_Count = md5.Buffer_Count(md5.MESSAGE_SIZE_MAXIMUM % md5.BLOCK_SIZE)
	testify.Panics(t, func() { md5.Digest_Write(&digest, md5.Source("x")) })
}

// Test_Invariant_Domains drives each valid state and caller-storage boundary through runtime APIs.
func Test_Invariant_Domains(t *testing.T) {
	var source [md5.SOURCE_SIZE_MAXIMUM]byte
	for _, size := range [...]int{
		md5.SOURCE_SIZE_MINIMUM,
		md5.SOURCE_SIZE_MINIMUM + TEST_COUNT_ONE,
		md5.SOURCE_SIZE_MINIMUM + TEST_COUNT_TWO,
		md5.SOURCE_SIZE_MAXIMUM,
	} {
		md5.Checksum(source[:size])
		var digest md5.Digest
		md5.Digest_Init(&digest)
		md5.Digest_Write(&digest, source[:size])
	}

	var destination [md5.DESTINATION_SIZE_MAXIMUM]byte
	for _, size := range [...]int{
		md5.DESTINATION_SIZE_MINIMUM,
		md5.DESTINATION_SIZE_MINIMUM + TEST_COUNT_ONE,
		md5.DESTINATION_SIZE_MINIMUM + TEST_COUNT_TWO,
		md5.DESTINATION_SIZE_MAXIMUM,
	} {
		var digest md5.Digest
		md5.Digest_Init(&digest)
		md5.Digest_Sum_Into(&digest, destination[:size])
	}

	message_sizes := [...]md5.Message_Size{
		md5.Message_Size(md5.MESSAGE_SIZE_MINIMUM),
		md5.Message_Size(md5.MESSAGE_SIZE_MINIMUM + TEST_COUNT_ONE),
		md5.Message_Size(md5.MESSAGE_SIZE_MINIMUM + TEST_COUNT_TWO),
		md5.Message_Size(md5.MESSAGE_SIZE_MAXIMUM),
	}
	for _, message_size := range message_sizes {
		var source_digest md5.Digest
		md5.Digest_Init(&source_digest)
		source_digest.Message_Size = message_size
		source_digest.Buffer_Count = md5.Buffer_Count(uint64(message_size) % md5.BLOCK_SIZE)

		md5.Digest_Size(&source_digest)
		md5.Digest_Block_Size(&source_digest)
		md5.Digest_Sum(&source_digest)
		md5.Digest_Sum_Into(&source_digest, destination[:])
		md5.Digest_Write(&source_digest, nil)

		var destination_digest md5.Digest
		md5.Digest_Init(&destination_digest)
		destination_digest.Message_Size = message_size
		destination_digest.Buffer_Count = source_digest.Buffer_Count
		md5.Digest_Clone_Into(&destination_digest, &source_digest)

		reset_digest := source_digest
		md5.Digest_Reset(&reset_digest)
		init_digest := source_digest
		md5.Digest_Init(&init_digest)
	}
}

// Test_Allocation proves caller state and result storage remain on stack.
func Test_Allocation(t *testing.T) {
	fixture := allocation_fixture{Source: md5.Source("abc")}
	md5.Digest_Init(&fixture.Digest)
	testify.Zero_Allocation(t, func() { md5.Digest_Init(&fixture.Digest) })
	testify.Zero_Allocation(t, func() {
		fixture.Count = md5.Digest_Write(&fixture.Digest, fixture.Source)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Value = md5.Digest_Sum(&fixture.Digest)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Output_Count, fixture.Output_Status = md5.Digest_Sum_Into(
			&fixture.Digest, fixture.Output[:],
		)
	})
	testify.Zero_Allocation(t, func() { fixture.Value = md5.Checksum(fixture.Source) })
}

func write_chunks(digest *md5.Digest, source md5.Source, chunk_size int) {
	for offset := TEST_COUNT_ZERO; offset < len(source); offset += chunk_size {
		end_offset := offset + chunk_size
		if end_offset > len(source) {
			end_offset = len(source)
		}
		md5.Digest_Write(digest, source[offset:end_offset])
	}
}

const TEST_COUNT_ZERO = bytes.SLICE_SIZE_MINIMUM
const TEST_COUNT_ONE = TEST_COUNT_ZERO + binary.UINT_8_SIZE
const TEST_COUNT_TWO = TEST_COUNT_ONE + TEST_COUNT_ONE
const TEST_MULTI_BLOCK_CHUNK_SIZE = md5.BLOCK_SIZE*binary.UINT_16_SIZE + TEST_COUNT_ONE

type allocation_fixture struct {
	Digest        md5.Digest
	Source        md5.Source
	Output        [md5.DIGEST_SIZE]byte
	Value         md5.Value
	Count         md5.Count
	Output_Count  md5.Output_Count
	Output_Status md5.Output_Status
}
