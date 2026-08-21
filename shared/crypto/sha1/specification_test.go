package sha1_test

import (
	standard_sha1 "crypto/sha1"
	"testing"

	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/crypto/sha1"
	"local/james-orcales/shared/encoding/binary"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/testify"
)

// Test_Package_Owned_State checks lifecycle and fixed geometry without shared dispatch.
func Test_Package_Owned_State(t *testing.T) {
	var source sha1.Digest
	sha1.Digest_Init(&source)
	testify.Equal(t, sha1.Size(sha1.DIGEST_SIZE), sha1.Digest_Size(&source))
	testify.Equal(t, sha1.Block_Size(sha1.BLOCK_SIZE), sha1.Digest_Block_Size(&source))
	sha1.Digest_Write(&source, sha1.Source("abc"))
	var destination sha1.Digest
	sha1.Digest_Clone_Into(&destination, &source)
	testify.Equal(t, sha1.Digest_Sum(&source), sha1.Digest_Sum(&destination))
}

// Test_Formula binds every derived width to shared primitive geometry.
func Test_Formula(t *testing.T) {
	testify.Equal(t, sha1.STATE_LANE_COUNT*bits.BIT_COUNT_32_MAXIMUM, sha1.DIGEST_BIT_COUNT)
	testify.Equal(t, sha1.DIGEST_BIT_COUNT/binary.BITS_PER_BYTE, sha1.DIGEST_SIZE)
	testify.Equal(t, sha1.MESSAGE_WORD_COUNT*binary.UINT_32_SIZE, sha1.BLOCK_SIZE)
	testify.Equal(t, sha1.BLOCK_SIZE*binary.BITS_PER_BYTE, sha1.BLOCK_BIT_COUNT)
	testify.Equal(t, sha1.STATE_LANE_COUNT+binary.UINT_8_SIZE, sha1.STATE_WORD_COUNT)
	testify.Equal(t, sha1.BLOCK_SIZE-sha1.MESSAGE_BIT_COUNT_SIZE, sha1.PADDING_BOUNDARY)
	testify.Equal(t, sha1.BLOCK_SIZE*binary.UINT_16_SIZE, sha1.FINAL_BLOCK_CAPACITY)
	testify.Equal(t, sha1.BLOCK_SIZE-binary.UINT_8_SIZE, sha1.BUFFER_COUNT_MAXIMUM)
	testify.Equal(t, bits.WORD_64_MAXIMUM/binary.BITS_PER_BYTE, sha1.MESSAGE_SIZE_MAXIMUM)
	testify.Equal(t, sha1.ROUND_SECTION_COUNT*sha1.ROUND_SECTION_SIZE, int(sha1.ROUND_COUNT))
	testify.Equal(t, ^sha1.INITIAL_STATE_0, sha1.INITIAL_STATE_2)
	testify.Equal(t, ^sha1.INITIAL_STATE_1, sha1.INITIAL_STATE_3)
}

// Test_Reference_Values binds output to published SHA-1 vectors.
func Test_Reference_Values(t *testing.T) {
	sources := [...]sha1.Source{sha1.Source(""), sha1.Source("abc")}
	for _, source := range sources {
		value := sha1.Value(standard_sha1.Sum(source))
		testify.Equal(t, value, sha1.Checksum(source))
		var digest sha1.Digest
		sha1.Digest_Init(&digest)
		midpoint := len(source) / TEST_COUNT_TWO
		testify.Equal(
			t, sha1.Count(midpoint), sha1.Digest_Write(&digest, source[:midpoint]),
		)
		testify.Equal(
			t, sha1.Count(len(source)-midpoint),
			sha1.Digest_Write(&digest, source[midpoint:]),
		)
		testify.Equal(t, value, sha1.Digest_Sum(&digest))
	}
}

// Test_Stream checks direct blocks, buffered tails, and repeated bounded writes.
func Test_Stream(t *testing.T) {
	var source [sha1.SOURCE_SIZE_MAXIMUM + TEST_COUNT_ONE]byte
	for index := range source {
		source[index] = byte(index)
	}
	for _, size := range [...]int{
		sha1.PADDING_BOUNDARY - TEST_COUNT_ONE, sha1.PADDING_BOUNDARY,
		sha1.BLOCK_SIZE - TEST_COUNT_ONE, sha1.BLOCK_SIZE,
		sha1.BLOCK_SIZE + TEST_COUNT_ONE,
	} {
		testify.Equal(
			t, sha1.Value(standard_sha1.Sum(source[:size])),
			sha1.Checksum(source[:size]),
		)
	}
	want := sha1.Value(standard_sha1.Sum(source[:]))
	var digest sha1.Digest
	sha1.Digest_Init(&digest)
	sha1.Digest_Write(&digest, source[:sha1.SOURCE_SIZE_MAXIMUM])
	sha1.Digest_Write(&digest, source[sha1.SOURCE_SIZE_MAXIMUM:])
	testify.Equal(t, want, sha1.Digest_Sum(&digest))
	sha1.Digest_Write(&digest, sha1.Source("x"))
	testify.Equal(
		t, sha1.Value(standard_sha1.Sum(append(source[:], 'x'))), sha1.Digest_Sum(&digest),
	)
	want_chunks := sha1.Value(standard_sha1.Sum(source[:sha1.SOURCE_SIZE_MAXIMUM]))
	for _, chunk_size := range [...]int{
		TEST_COUNT_ONE, sha1.BLOCK_SIZE - TEST_COUNT_ONE, sha1.BLOCK_SIZE,
		sha1.BLOCK_SIZE + TEST_COUNT_ONE, TEST_MULTI_BLOCK_CHUNK_SIZE,
	} {
		sha1.Digest_Init(&digest)
		write_chunks(&digest, source[:sha1.SOURCE_SIZE_MAXIMUM], chunk_size)
		testify.Equal(t, want_chunks, sha1.Digest_Sum(&digest), chunk_size)
	}
}

// Test_Caller_Owned_Output keeps short storage untouched and Sum non-consuming.
func Test_Caller_Owned_Output(t *testing.T) {
	var digest sha1.Digest
	sha1.Digest_Init(&digest)
	sha1.Digest_Write(&digest, sha1.Source("abc"))
	before := digest
	var short [sha1.DIGEST_SIZE - TEST_COUNT_ONE]byte
	count, status := sha1.Digest_Sum_Into(&digest, short[:])
	testify.Equal(t, sha1.OUTPUT_COUNT_REQUIRED, count)
	testify.Equal(t, sha1.OUTPUT_STATUS_TOO_SMALL, status)
	testify.Equal(t, [sha1.DIGEST_SIZE - TEST_COUNT_ONE]byte{}, short)
	var output [sha1.DIGEST_SIZE]byte
	count, status = sha1.Digest_Sum_Into(&digest, output[:])
	testify.Equal(t, sha1.OUTPUT_COUNT_REQUIRED, count)
	testify.Equal(t, sha1.OUTPUT_STATUS_OK, status)
	testify.Equal(t, sha1.Value(output), sha1.Digest_Sum(&digest))
	testify.Equal(t, before, digest)
	sha1.Digest_Reset(&digest)
	testify.Equal(t, sha1.Checksum(nil), sha1.Digest_Sum(&digest))
}

// Test_Bounds rejects oversized calls and uninitialized caller state.
func Test_Bounds(t *testing.T) {
	var digest sha1.Digest
	testify.Panics(t, func() { sha1.Digest_Write(&digest, nil) })
	sha1.Digest_Init(&digest)
	var source [sha1.SOURCE_SIZE_MAXIMUM + TEST_COUNT_ONE]byte
	testify.Panics(t, func() { sha1.Digest_Write(&digest, source[:]) })
	testify.Panics(t, func() { sha1.Checksum(source[:]) })
	var destination [sha1.DESTINATION_SIZE_MAXIMUM + TEST_COUNT_ONE]byte
	testify.Panics(t, func() { sha1.Digest_Sum_Into(&digest, destination[:]) })
	digest.Message_Size = sha1.Message_Size(sha1.MESSAGE_SIZE_MAXIMUM)
	digest.Buffer_Count = sha1.Buffer_Count(sha1.MESSAGE_SIZE_MAXIMUM % sha1.BLOCK_SIZE)
	testify.Panics(t, func() { sha1.Digest_Write(&digest, sha1.Source("x")) })
}

// Test_Invariant_Domains drives each valid state and caller-storage boundary through runtime APIs.
func Test_Invariant_Domains(t *testing.T) {
	var source [sha1.SOURCE_SIZE_MAXIMUM]byte
	for _, size := range [...]int{
		sha1.SOURCE_SIZE_MINIMUM,
		sha1.SOURCE_SIZE_MINIMUM + TEST_COUNT_ONE,
		sha1.SOURCE_SIZE_MINIMUM + TEST_COUNT_TWO,
		sha1.SOURCE_SIZE_MAXIMUM,
	} {
		sha1.Checksum(source[:size])
		var digest sha1.Digest
		sha1.Digest_Init(&digest)
		sha1.Digest_Write(&digest, source[:size])
	}

	var destination [sha1.DESTINATION_SIZE_MAXIMUM]byte
	for _, size := range [...]int{
		sha1.DESTINATION_SIZE_MINIMUM,
		sha1.DESTINATION_SIZE_MINIMUM + TEST_COUNT_ONE,
		sha1.DESTINATION_SIZE_MINIMUM + TEST_COUNT_TWO,
		sha1.DESTINATION_SIZE_MAXIMUM,
	} {
		var digest sha1.Digest
		sha1.Digest_Init(&digest)
		sha1.Digest_Sum_Into(&digest, destination[:size])
	}

	message_sizes := [...]sha1.Message_Size{
		sha1.Message_Size(sha1.MESSAGE_SIZE_MINIMUM),
		sha1.Message_Size(sha1.MESSAGE_SIZE_MINIMUM + TEST_COUNT_ONE),
		sha1.Message_Size(sha1.MESSAGE_SIZE_MINIMUM + TEST_COUNT_TWO),
		sha1.Message_Size(sha1.MESSAGE_SIZE_MAXIMUM),
	}
	for _, message_size := range message_sizes {
		var source_digest sha1.Digest
		sha1.Digest_Init(&source_digest)
		source_digest.Message_Size = message_size
		source_digest.Buffer_Count = sha1.Buffer_Count(
			uint64(message_size) % sha1.BLOCK_SIZE,
		)

		sha1.Digest_Size(&source_digest)
		sha1.Digest_Block_Size(&source_digest)
		sha1.Digest_Sum(&source_digest)
		sha1.Digest_Sum_Into(&source_digest, destination[:])
		sha1.Digest_Write(&source_digest, nil)

		var destination_digest sha1.Digest
		sha1.Digest_Init(&destination_digest)
		destination_digest.Message_Size = message_size
		destination_digest.Buffer_Count = source_digest.Buffer_Count
		sha1.Digest_Clone_Into(&destination_digest, &source_digest)

		reset_digest := source_digest
		sha1.Digest_Reset(&reset_digest)
		init_digest := source_digest
		sha1.Digest_Init(&init_digest)
	}
}

// Test_Allocation proves caller state and result storage remain on stack.
func Test_Allocation(t *testing.T) {
	fixture := allocation_fixture{Source: sha1.Source("abc")}
	sha1.Digest_Init(&fixture.Digest)
	testify.Zero_Allocation(t, func() { sha1.Digest_Init(&fixture.Digest) })
	testify.Zero_Allocation(t, func() {
		fixture.Count = sha1.Digest_Write(&fixture.Digest, fixture.Source)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Value = sha1.Digest_Sum(&fixture.Digest)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Output_Count, fixture.Output_Status = sha1.Digest_Sum_Into(
			&fixture.Digest, fixture.Output[:],
		)
	})
	testify.Zero_Allocation(t, func() { fixture.Value = sha1.Checksum(fixture.Source) })
}

func write_chunks(digest *sha1.Digest, source sha1.Source, chunk_size int) {
	for offset := TEST_COUNT_ZERO; offset < len(source); offset += chunk_size {
		end_offset := offset + chunk_size
		if end_offset > len(source) {
			end_offset = len(source)
		}
		sha1.Digest_Write(digest, source[offset:end_offset])
	}
}

const TEST_COUNT_ZERO = bytes.SLICE_SIZE_MINIMUM
const TEST_COUNT_ONE = TEST_COUNT_ZERO + binary.UINT_8_SIZE
const TEST_COUNT_TWO = TEST_COUNT_ONE + TEST_COUNT_ONE
const TEST_MULTI_BLOCK_CHUNK_SIZE = sha1.BLOCK_SIZE*binary.UINT_16_SIZE + TEST_COUNT_ONE

type allocation_fixture struct {
	Digest        sha1.Digest
	Source        sha1.Source
	Output        [sha1.DIGEST_SIZE]byte
	Value         sha1.Value
	Count         sha1.Count
	Output_Count  sha1.Output_Count
	Output_Status sha1.Output_Status
}
