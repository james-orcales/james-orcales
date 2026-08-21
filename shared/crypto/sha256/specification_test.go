package sha256_test

import (
	standard_sha256 "crypto/sha256"
	"testing"

	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/crypto/sha256"
	"local/james-orcales/shared/encoding/binary"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/testify"
)

// Test_Package_Owned_State checks both kinds and fixed geometry without shared dispatch.
func Test_Package_Owned_State(t *testing.T) {
	var digest_224 sha256.Digest
	sha256.Digest_Init(&digest_224, sha256.KIND_SHA_224)
	testify.Equal(t, sha256.Size(sha256.DIGEST_224_SIZE), sha256.Digest_Size(&digest_224))
	var digest_256 sha256.Digest
	sha256.Digest_Init(&digest_256, sha256.KIND_SHA_256)
	testify.Equal(t, sha256.Size(sha256.DIGEST_256_SIZE), sha256.Digest_Size(&digest_256))
	testify.Equal(
		t, sha256.Block_Size(sha256.BLOCK_SIZE), sha256.Digest_Block_Size(&digest_256),
	)
	sha256.Digest_Write(&digest_256, sha256.Source("abc"))
	var clone sha256.Digest
	sha256.Digest_Clone_Into(&clone, &digest_256)
	testify.Equal(t, sha256.Digest_Sum_256(&digest_256), sha256.Digest_Sum_256(&clone))
}

// Test_Formula binds every derived width to shared primitive geometry.
func Test_Formula(t *testing.T) {
	testify.Equal(
		t, sha256.DIGEST_224_LANE_COUNT*bits.BIT_COUNT_32_MAXIMUM,
		sha256.DIGEST_224_BIT_COUNT,
	)
	testify.Equal(
		t, sha256.STATE_LANE_COUNT*bits.BIT_COUNT_32_MAXIMUM,
		sha256.DIGEST_256_BIT_COUNT,
	)
	testify.Equal(t, sha256.DIGEST_224_BIT_COUNT/binary.BITS_PER_BYTE, sha256.DIGEST_224_SIZE)
	testify.Equal(t, sha256.DIGEST_256_BIT_COUNT/binary.BITS_PER_BYTE, sha256.DIGEST_256_SIZE)
	testify.Equal(t, sha256.MESSAGE_WORD_COUNT*binary.UINT_32_SIZE, sha256.BLOCK_SIZE)
	testify.Equal(t, sha256.STATE_LANE_COUNT+binary.UINT_8_SIZE, sha256.STATE_WORD_COUNT)
	testify.Equal(t, sha256.BLOCK_SIZE-sha256.MESSAGE_BIT_COUNT_SIZE, sha256.PADDING_BOUNDARY)
	testify.Equal(t, sha256.BLOCK_SIZE*binary.UINT_16_SIZE, sha256.FINAL_BLOCK_CAPACITY)
	testify.Equal(t, sha256.BLOCK_SIZE-binary.UINT_8_SIZE, sha256.BUFFER_COUNT_MAXIMUM)
	testify.Equal(t, bits.WORD_64_MAXIMUM/binary.BITS_PER_BYTE, sha256.MESSAGE_SIZE_MAXIMUM)
	testify.Equal(
		t, sha256.ROUND_SECTION_COUNT*sha256.ROUND_SECTION_SIZE,
		int(sha256.ROUND_COUNT),
	)
}

// Test_Reference_Values binds both functions to published vectors.
func Test_Reference_Values(t *testing.T) {
	source := sha256.Source("abc")
	want_224 := sha256.Value_224(standard_sha256.Sum224(source))
	want_256 := sha256.Value_256(standard_sha256.Sum256(source))
	testify.Equal(t, want_224, sha256.Checksum_224(source))
	testify.Equal(t, want_256, sha256.Checksum_256(source))
	var digest_224 sha256.Digest
	sha256.Digest_Init(&digest_224, sha256.KIND_SHA_224)
	sha256.Digest_Write(&digest_224, source[:TEST_COUNT_ONE])
	sha256.Digest_Write(&digest_224, source[TEST_COUNT_ONE:])
	testify.Equal(t, want_224, sha256.Digest_Sum_224(&digest_224))
	var digest_256 sha256.Digest
	sha256.Digest_Init(&digest_256, sha256.KIND_SHA_256)
	sha256.Digest_Write(&digest_256, source[:TEST_COUNT_ONE])
	sha256.Digest_Write(&digest_256, source[TEST_COUNT_ONE:])
	testify.Equal(t, want_256, sha256.Digest_Sum_256(&digest_256))
}

// Test_Stream checks direct blocks, buffered tails, and both selected kinds.
func Test_Stream(t *testing.T) {
	var source [sha256.SOURCE_SIZE_MAXIMUM + TEST_COUNT_ONE]byte
	for index := range source {
		source[index] = byte(index)
	}
	for _, size := range [...]int{
		sha256.PADDING_BOUNDARY - TEST_COUNT_ONE, sha256.PADDING_BOUNDARY,
		sha256.BLOCK_SIZE - TEST_COUNT_ONE, sha256.BLOCK_SIZE,
		sha256.BLOCK_SIZE + TEST_COUNT_ONE,
	} {
		testify.Equal(
			t, sha256.Value_224(standard_sha256.Sum224(source[:size])),
			sha256.Checksum_224(source[:size]),
		)
		testify.Equal(
			t, sha256.Value_256(standard_sha256.Sum256(source[:size])),
			sha256.Checksum_256(source[:size]),
		)
	}
	var digest_224 sha256.Digest
	sha256.Digest_Init(&digest_224, sha256.KIND_SHA_224)
	sha256.Digest_Write(&digest_224, source[:sha256.SOURCE_SIZE_MAXIMUM])
	sha256.Digest_Write(&digest_224, source[sha256.SOURCE_SIZE_MAXIMUM:])
	testify.Equal(
		t, sha256.Value_224(standard_sha256.Sum224(source[:])),
		sha256.Digest_Sum_224(&digest_224),
	)
	var digest_256 sha256.Digest
	sha256.Digest_Init(&digest_256, sha256.KIND_SHA_256)
	sha256.Digest_Write(&digest_256, source[:sha256.SOURCE_SIZE_MAXIMUM])
	sha256.Digest_Write(&digest_256, source[sha256.SOURCE_SIZE_MAXIMUM:])
	testify.Equal(
		t, sha256.Value_256(standard_sha256.Sum256(source[:])),
		sha256.Digest_Sum_256(&digest_256),
	)
	want_chunks := sha256.Value_256(
		standard_sha256.Sum256(source[:sha256.SOURCE_SIZE_MAXIMUM]),
	)
	for _, chunk_size := range [...]int{
		TEST_COUNT_ONE, sha256.BLOCK_SIZE - TEST_COUNT_ONE, sha256.BLOCK_SIZE,
		sha256.BLOCK_SIZE + TEST_COUNT_ONE, TEST_MULTI_BLOCK_CHUNK_SIZE,
	} {
		sha256.Digest_Init(&digest_256, sha256.KIND_SHA_256)
		write_chunks(&digest_256, source[:sha256.SOURCE_SIZE_MAXIMUM], chunk_size)
		testify.Equal(t, want_chunks, sha256.Digest_Sum_256(&digest_256), chunk_size)
	}
}

// Test_Caller_Owned_Output keeps short storage untouched and Sum non-consuming.
func Test_Caller_Owned_Output(t *testing.T) {
	var digest sha256.Digest
	sha256.Digest_Init(&digest, sha256.KIND_SHA_224)
	sha256.Digest_Write(&digest, sha256.Source("abc"))
	before := digest
	var short [sha256.DIGEST_224_SIZE - TEST_COUNT_ONE]byte
	count, status := sha256.Digest_Sum_Into(&digest, short[:])
	testify.Equal(t, sha256.OUTPUT_COUNT_224_REQUIRED, count)
	testify.Equal(t, sha256.OUTPUT_STATUS_TOO_SMALL, status)
	var output_224 [sha256.DIGEST_224_SIZE]byte
	count, status = sha256.Digest_Sum_Into(&digest, output_224[:])
	testify.Equal(t, sha256.OUTPUT_COUNT_224_REQUIRED, count)
	testify.Equal(t, sha256.OUTPUT_STATUS_OK, status)
	testify.Equal(t, sha256.Digest_Sum_224(&digest), sha256.Value_224(output_224))
	testify.Equal(t, before, digest)
	sha256.Digest_Init(&digest, sha256.KIND_SHA_256)
	var output_256 [sha256.DIGEST_256_SIZE]byte
	count, status = sha256.Digest_Sum_Into(&digest, output_256[:])
	testify.Equal(t, sha256.OUTPUT_COUNT_256_REQUIRED, count)
	testify.Equal(t, sha256.OUTPUT_STATUS_OK, status)
	sha256.Digest_Reset(&digest)
	testify.Equal(t, sha256.Checksum_256(nil), sha256.Digest_Sum_256(&digest))
}

// Test_Bounds rejects oversized calls, wrong kinds, and uninitialized caller state.
func Test_Bounds(t *testing.T) {
	var digest sha256.Digest
	testify.Panics(t, func() { sha256.Digest_Write(&digest, nil) })
	sha256.Digest_Init(&digest, sha256.KIND_SHA_256)
	var source [sha256.SOURCE_SIZE_MAXIMUM + TEST_COUNT_ONE]byte
	testify.Panics(t, func() { sha256.Digest_Write(&digest, source[:]) })
	testify.Panics(t, func() { sha256.Checksum_256(source[:]) })
	testify.Panics(t, func() { sha256.Digest_Sum_224(&digest) })
	sha256.Digest_Init(&digest, sha256.KIND_SHA_224)
	testify.Panics(t, func() { sha256.Digest_Sum_256(&digest) })
	var destination [sha256.DESTINATION_SIZE_MAXIMUM + TEST_COUNT_ONE]byte
	testify.Panics(t, func() { sha256.Digest_Sum_Into(&digest, destination[:]) })
	digest.Message_Size = sha256.Message_Size(sha256.MESSAGE_SIZE_MAXIMUM)
	digest.Buffer_Count = sha256.Buffer_Count(sha256.MESSAGE_SIZE_MAXIMUM % sha256.BLOCK_SIZE)
	testify.Panics(t, func() { sha256.Digest_Write(&digest, sha256.Source("x")) })
}

// Test_Invariant_Domains drives each valid state and caller-storage boundary through runtime APIs.
func Test_Invariant_Domains(t *testing.T) {
	var source [sha256.SOURCE_SIZE_MAXIMUM]byte
	for _, size := range [...]int{
		sha256.SOURCE_SIZE_MINIMUM,
		sha256.SOURCE_SIZE_MINIMUM + TEST_COUNT_ONE,
		sha256.SOURCE_SIZE_MINIMUM + TEST_COUNT_TWO,
		sha256.SOURCE_SIZE_MAXIMUM,
	} {
		sha256.Checksum_224(source[:size])
		sha256.Checksum_256(source[:size])
		for _, kind := range [...]sha256.Kind{sha256.KIND_SHA_224, sha256.KIND_SHA_256} {
			var digest sha256.Digest
			sha256.Digest_Init(&digest, kind)
			sha256.Digest_Write(&digest, source[:size])
		}
	}

	var destination [sha256.DESTINATION_SIZE_MAXIMUM]byte
	for _, size := range [...]int{
		sha256.DESTINATION_SIZE_MINIMUM,
		sha256.DESTINATION_SIZE_MINIMUM + TEST_COUNT_ONE,
		sha256.DESTINATION_SIZE_MINIMUM + TEST_COUNT_TWO,
		sha256.DESTINATION_SIZE_MAXIMUM,
	} {
		for _, kind := range [...]sha256.Kind{sha256.KIND_SHA_224, sha256.KIND_SHA_256} {
			var digest sha256.Digest
			sha256.Digest_Init(&digest, kind)
			sha256.Digest_Sum_Into(&digest, destination[:size])
		}
	}

	message_sizes := [...]sha256.Message_Size{
		sha256.Message_Size(sha256.MESSAGE_SIZE_MINIMUM),
		sha256.Message_Size(sha256.MESSAGE_SIZE_MINIMUM + TEST_COUNT_ONE),
		sha256.Message_Size(sha256.MESSAGE_SIZE_MINIMUM + TEST_COUNT_TWO),
		sha256.Message_Size(sha256.MESSAGE_SIZE_MAXIMUM),
	}
	for _, kind := range [...]sha256.Kind{sha256.KIND_SHA_224, sha256.KIND_SHA_256} {
		for _, message_size := range message_sizes {
			var source_digest sha256.Digest
			sha256.Digest_Init(&source_digest, kind)
			source_digest.Message_Size = message_size
			source_digest.Buffer_Count = sha256.Buffer_Count(
				uint64(message_size) % sha256.BLOCK_SIZE,
			)

			sha256.Digest_Size(&source_digest)
			sha256.Digest_Block_Size(&source_digest)
			if kind == sha256.KIND_SHA_224 {
				sha256.Digest_Sum_224(&source_digest)
			} else {
				sha256.Digest_Sum_256(&source_digest)
			}
			sha256.Digest_Sum_Into(&source_digest, destination[:])
			sha256.Digest_Write(&source_digest, nil)

			var destination_digest sha256.Digest
			sha256.Digest_Init(&destination_digest, kind)
			destination_digest.Message_Size = message_size
			destination_digest.Buffer_Count = source_digest.Buffer_Count
			sha256.Digest_Clone_Into(&destination_digest, &source_digest)

			reset_digest := source_digest
			sha256.Digest_Reset(&reset_digest)
			init_digest := source_digest
			sha256.Digest_Init(&init_digest, kind)
		}
	}
}

// Test_Allocation proves both algorithms retain caller ownership.
func Test_Allocation(t *testing.T) {
	fixture := allocation_fixture{Source: sha256.Source("abc")}
	sha256.Digest_Init(&fixture.Digest, sha256.KIND_SHA_256)
	testify.Zero_Allocation(t, func() {
		sha256.Digest_Init(&fixture.Digest, sha256.KIND_SHA_256)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Count = sha256.Digest_Write(&fixture.Digest, fixture.Source)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Value_256 = sha256.Digest_Sum_256(&fixture.Digest)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Output_Count, fixture.Output_Status = sha256.Digest_Sum_Into(
			&fixture.Digest, fixture.Output[:],
		)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Value_224 = sha256.Checksum_224(fixture.Source)
		fixture.Value_256 = sha256.Checksum_256(fixture.Source)
	})
}

func write_chunks(digest *sha256.Digest, source sha256.Source, chunk_size int) {
	for offset := TEST_COUNT_ZERO; offset < len(source); offset += chunk_size {
		end_offset := offset + chunk_size
		if end_offset > len(source) {
			end_offset = len(source)
		}
		sha256.Digest_Write(digest, source[offset:end_offset])
	}
}

const TEST_COUNT_ZERO = bytes.SLICE_SIZE_MINIMUM
const TEST_COUNT_ONE = TEST_COUNT_ZERO + binary.UINT_8_SIZE
const TEST_COUNT_TWO = TEST_COUNT_ONE + TEST_COUNT_ONE
const TEST_MULTI_BLOCK_CHUNK_SIZE = sha256.BLOCK_SIZE*binary.UINT_16_SIZE + TEST_COUNT_ONE

type allocation_fixture struct {
	Digest        sha256.Digest
	Source        sha256.Source
	Output        [sha256.DIGEST_256_SIZE]byte
	Value_224     sha256.Value_224
	Value_256     sha256.Value_256
	Count         sha256.Count
	Output_Count  sha256.Output_Count
	Output_Status sha256.Output_Status
}
