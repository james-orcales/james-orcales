package sha512_test

import (
	standard_sha512 "crypto/sha512"
	"testing"

	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/crypto/sha512"
	"local/james-orcales/shared/encoding/binary"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/testify"
)

// Test_Package_Owned_State checks four kinds and fixed geometry without shared dispatch.
func Test_Package_Owned_State(t *testing.T) {
	tests := [...]struct {
		Kind sha512.Kind
		Size sha512.Size
	}{
		{Kind: sha512.KIND_SHA_512_224, Size: sha512.DIGEST_224_SIZE},
		{Kind: sha512.KIND_SHA_512_256, Size: sha512.DIGEST_256_SIZE},
		{Kind: sha512.KIND_SHA_384, Size: sha512.DIGEST_384_SIZE},
		{Kind: sha512.KIND_SHA_512, Size: sha512.DIGEST_512_SIZE},
	}
	for _, test := range tests {
		var digest sha512.Digest
		sha512.Digest_Init(&digest, test.Kind)
		testify.Equal(t, test.Size, sha512.Digest_Size(&digest))
		testify.Equal(
			t, sha512.Block_Size(sha512.BLOCK_SIZE), sha512.Digest_Block_Size(&digest),
		)
	}
	var source sha512.Digest
	sha512.Digest_Init(&source, sha512.KIND_SHA_512)
	sha512.Digest_Write(&source, sha512.Source("abc"))
	var clone sha512.Digest
	sha512.Digest_Clone_Into(&clone, &source)
	testify.Equal(t, sha512.Digest_Sum_512(&source), sha512.Digest_Sum_512(&clone))
}

// Test_Formula binds every derived width to shared primitive geometry.
func Test_Formula(t *testing.T) {
	testify.Equal(
		t, sha512.DIGEST_224_LANE_COUNT*bits.BIT_COUNT_32_MAXIMUM,
		sha512.DIGEST_224_BIT_COUNT,
	)
	testify.Equal(
		t, sha512.DIGEST_256_LANE_COUNT*bits.BIT_COUNT_64_MAXIMUM,
		sha512.DIGEST_256_BIT_COUNT,
	)
	testify.Equal(
		t, sha512.DIGEST_384_LANE_COUNT*bits.BIT_COUNT_64_MAXIMUM,
		sha512.DIGEST_384_BIT_COUNT,
	)
	testify.Equal(
		t, sha512.STATE_LANE_COUNT*bits.BIT_COUNT_64_MAXIMUM,
		sha512.DIGEST_512_BIT_COUNT,
	)
	testify.Equal(t, sha512.DIGEST_224_BIT_COUNT/binary.BITS_PER_BYTE, sha512.DIGEST_224_SIZE)
	testify.Equal(t, sha512.DIGEST_512_BIT_COUNT/binary.BITS_PER_BYTE, sha512.DIGEST_512_SIZE)
	testify.Equal(t, sha512.MESSAGE_WORD_COUNT*binary.UINT_64_SIZE, sha512.BLOCK_SIZE)
	testify.Equal(t, sha512.STATE_LANE_COUNT+binary.UINT_8_SIZE, sha512.STATE_WORD_COUNT)
	testify.Equal(t, sha512.BLOCK_SIZE-sha512.MESSAGE_BIT_COUNT_SIZE, sha512.PADDING_BOUNDARY)
	testify.Equal(t, sha512.BLOCK_SIZE*binary.UINT_16_SIZE, sha512.FINAL_BLOCK_CAPACITY)
	testify.Equal(t, sha512.BLOCK_SIZE-binary.UINT_8_SIZE, sha512.BUFFER_COUNT_MAXIMUM)
	testify.Equal(t, bits.WORD_64_MAXIMUM/binary.BITS_PER_BYTE, sha512.MESSAGE_SIZE_MAXIMUM)
	testify.Equal(
		t, sha512.ROUND_SECTION_COUNT*sha512.ROUND_SECTION_SIZE,
		int(sha512.ROUND_COUNT),
	)
}

// Test_Reference_Values binds all four functions to standard known answers.
func Test_Reference_Values(t *testing.T) {
	source := sha512.Source("abc")
	testify.Equal(
		t, sha512.Value_224(standard_sha512.Sum512_224(source)),
		sha512.Checksum_512_224(source),
	)
	testify.Equal(
		t, sha512.Value_256(standard_sha512.Sum512_256(source)),
		sha512.Checksum_512_256(source),
	)
	testify.Equal(
		t, sha512.Value_384(standard_sha512.Sum384(source)), sha512.Checksum_384(source),
	)
	testify.Equal(
		t, sha512.Value_512(standard_sha512.Sum512(source)), sha512.Checksum_512(source),
	)
}

// Test_Stream checks direct blocks, buffered tails, and every selected kind.
func Test_Stream(t *testing.T) {
	var source [sha512.SOURCE_SIZE_MAXIMUM + TEST_COUNT_ONE]byte
	for index := range source {
		source[index] = byte(index)
	}
	for _, size := range [...]int{
		sha512.PADDING_BOUNDARY - TEST_COUNT_ONE, sha512.PADDING_BOUNDARY,
		sha512.BLOCK_SIZE - TEST_COUNT_ONE, sha512.BLOCK_SIZE,
		sha512.BLOCK_SIZE + TEST_COUNT_ONE,
	} {
		testify.Equal(
			t, sha512.Value_224(standard_sha512.Sum512_224(source[:size])),
			sha512.Checksum_512_224(source[:size]),
		)
		testify.Equal(
			t, sha512.Value_256(standard_sha512.Sum512_256(source[:size])),
			sha512.Checksum_512_256(source[:size]),
		)
		testify.Equal(
			t, sha512.Value_384(standard_sha512.Sum384(source[:size])),
			sha512.Checksum_384(source[:size]),
		)
		testify.Equal(
			t, sha512.Value_512(standard_sha512.Sum512(source[:size])),
			sha512.Checksum_512(source[:size]),
		)
	}
	var digest_224 sha512.Digest
	sha512.Digest_Init(&digest_224, sha512.KIND_SHA_512_224)
	write_divided(&digest_224, source[:])
	testify.Equal(
		t, sha512.Value_224(standard_sha512.Sum512_224(source[:])),
		sha512.Digest_Sum_512_224(&digest_224),
	)
	var digest_256 sha512.Digest
	sha512.Digest_Init(&digest_256, sha512.KIND_SHA_512_256)
	write_divided(&digest_256, source[:])
	testify.Equal(
		t, sha512.Value_256(standard_sha512.Sum512_256(source[:])),
		sha512.Digest_Sum_512_256(&digest_256),
	)
	var digest_384 sha512.Digest
	sha512.Digest_Init(&digest_384, sha512.KIND_SHA_384)
	write_divided(&digest_384, source[:])
	testify.Equal(
		t, sha512.Value_384(standard_sha512.Sum384(source[:])),
		sha512.Digest_Sum_384(&digest_384),
	)
	var digest_512 sha512.Digest
	sha512.Digest_Init(&digest_512, sha512.KIND_SHA_512)
	write_divided(&digest_512, source[:])
	testify.Equal(
		t, sha512.Value_512(standard_sha512.Sum512(source[:])),
		sha512.Digest_Sum_512(&digest_512),
	)
	want_chunks := sha512.Value_512(
		standard_sha512.Sum512(source[:sha512.SOURCE_SIZE_MAXIMUM]),
	)
	for _, chunk_size := range [...]int{
		TEST_COUNT_ONE, sha512.BLOCK_SIZE - TEST_COUNT_ONE, sha512.BLOCK_SIZE,
		sha512.BLOCK_SIZE + TEST_COUNT_ONE, TEST_MULTI_BLOCK_CHUNK_SIZE,
	} {
		sha512.Digest_Init(&digest_512, sha512.KIND_SHA_512)
		write_chunks(&digest_512, source[:sha512.SOURCE_SIZE_MAXIMUM], chunk_size)
		testify.Equal(t, want_chunks, sha512.Digest_Sum_512(&digest_512), chunk_size)
	}
}

// Test_Caller_Owned_Output keeps short storage untouched and reports selected width.
func Test_Caller_Owned_Output(t *testing.T) {
	var digest sha512.Digest
	sha512.Digest_Init(&digest, sha512.KIND_SHA_512_224)
	var short [sha512.DIGEST_224_SIZE - TEST_COUNT_ONE]byte
	count, status := sha512.Digest_Sum_Into(&digest, short[:])
	testify.Equal(t, sha512.OUTPUT_COUNT_224_REQUIRED, count)
	testify.Equal(t, sha512.OUTPUT_STATUS_TOO_SMALL, status)
	var output [sha512.DIGEST_512_SIZE]byte
	count, status = sha512.Digest_Sum_Into(&digest, output[:sha512.DIGEST_224_SIZE])
	testify.Equal(t, sha512.OUTPUT_COUNT_224_REQUIRED, count)
	testify.Equal(t, sha512.OUTPUT_STATUS_OK, status)
	sha512.Digest_Init(&digest, sha512.KIND_SHA_512_256)
	count, status = sha512.Digest_Sum_Into(&digest, output[:sha512.DIGEST_256_SIZE])
	testify.Equal(t, sha512.OUTPUT_COUNT_256_REQUIRED, count)
	testify.Equal(t, sha512.OUTPUT_STATUS_OK, status)
	sha512.Digest_Init(&digest, sha512.KIND_SHA_384)
	count, status = sha512.Digest_Sum_Into(&digest, output[:sha512.DIGEST_384_SIZE])
	testify.Equal(t, sha512.OUTPUT_COUNT_384_REQUIRED, count)
	testify.Equal(t, sha512.OUTPUT_STATUS_OK, status)
	sha512.Digest_Init(&digest, sha512.KIND_SHA_512)
	count, status = sha512.Digest_Sum_Into(&digest, output[:])
	testify.Equal(t, sha512.OUTPUT_COUNT_512_REQUIRED, count)
	testify.Equal(t, sha512.OUTPUT_STATUS_OK, status)
	sha512.Digest_Reset(&digest)
	testify.Equal(t, sha512.Checksum_512(nil), sha512.Digest_Sum_512(&digest))
}

// Test_Bounds rejects oversized calls, wrong kinds, and uninitialized caller state.
func Test_Bounds(t *testing.T) {
	var digest sha512.Digest
	testify.Panics(t, func() { sha512.Digest_Write(&digest, nil) })
	sha512.Digest_Init(&digest, sha512.KIND_SHA_512)
	var source [sha512.SOURCE_SIZE_MAXIMUM + TEST_COUNT_ONE]byte
	testify.Panics(t, func() { sha512.Digest_Write(&digest, source[:]) })
	testify.Panics(t, func() { sha512.Checksum_512(source[:]) })
	testify.Panics(t, func() { sha512.Digest_Sum_384(&digest) })
	var destination [sha512.DESTINATION_SIZE_MAXIMUM + TEST_COUNT_ONE]byte
	testify.Panics(t, func() { sha512.Digest_Sum_Into(&digest, destination[:]) })
	digest.Message_Size = sha512.Message_Size(sha512.MESSAGE_SIZE_MAXIMUM)
	digest.Buffer_Count = sha512.Buffer_Count(sha512.MESSAGE_SIZE_MAXIMUM % sha512.BLOCK_SIZE)
	testify.Panics(t, func() { sha512.Digest_Write(&digest, sha512.Source("x")) })
}

// Test_Invariant_Domains drives each valid state and caller-storage boundary through runtime APIs.
func Test_Invariant_Domains(t *testing.T) {
	kinds := [...]sha512.Kind{
		sha512.KIND_SHA_512_224,
		sha512.KIND_SHA_512_256,
		sha512.KIND_SHA_384,
		sha512.KIND_SHA_512,
	}
	var source [sha512.SOURCE_SIZE_MAXIMUM]byte
	for _, size := range [...]int{
		sha512.SOURCE_SIZE_MINIMUM,
		sha512.SOURCE_SIZE_MINIMUM + TEST_COUNT_ONE,
		sha512.SOURCE_SIZE_MINIMUM + TEST_COUNT_TWO,
		sha512.SOURCE_SIZE_MAXIMUM,
	} {
		sha512.Checksum_512_224(source[:size])
		sha512.Checksum_512_256(source[:size])
		sha512.Checksum_384(source[:size])
		sha512.Checksum_512(source[:size])
		for _, kind := range kinds {
			var digest sha512.Digest
			sha512.Digest_Init(&digest, kind)
			sha512.Digest_Write(&digest, source[:size])
		}
	}

	var destination [sha512.DESTINATION_SIZE_MAXIMUM]byte
	for _, size := range [...]int{
		sha512.DESTINATION_SIZE_MINIMUM,
		sha512.DESTINATION_SIZE_MINIMUM + TEST_COUNT_ONE,
		sha512.DESTINATION_SIZE_MINIMUM + TEST_COUNT_TWO,
		sha512.DESTINATION_SIZE_MAXIMUM,
	} {
		for _, kind := range kinds {
			var digest sha512.Digest
			sha512.Digest_Init(&digest, kind)
			sha512.Digest_Sum_Into(&digest, destination[:size])
		}
	}

	state_domains(t)
}

// Test_Allocation proves all four functions retain caller ownership.
func Test_Allocation(t *testing.T) {
	fixture := allocation_fixture{Source: sha512.Source("abc")}
	sha512.Digest_Init(&fixture.Digest, sha512.KIND_SHA_512)
	testify.Zero_Allocation(t, func() {
		sha512.Digest_Init(&fixture.Digest, sha512.KIND_SHA_512)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Count = sha512.Digest_Write(&fixture.Digest, fixture.Source)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Value_512 = sha512.Digest_Sum_512(&fixture.Digest)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Output_Count, fixture.Output_Status = sha512.Digest_Sum_Into(
			&fixture.Digest, fixture.Output[:],
		)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Value_224 = sha512.Checksum_512_224(fixture.Source)
		fixture.Value_256 = sha512.Checksum_512_256(fixture.Source)
		fixture.Value_384 = sha512.Checksum_384(fixture.Source)
		fixture.Value_512 = sha512.Checksum_512(fixture.Source)
	})
}

func write_divided(digest *sha512.Digest, source sha512.Source) {
	sha512.Digest_Write(digest, source[:sha512.SOURCE_SIZE_MAXIMUM])
	sha512.Digest_Write(digest, source[sha512.SOURCE_SIZE_MAXIMUM:])
}

func write_chunks(digest *sha512.Digest, source sha512.Source, chunk_size int) {
	for offset := TEST_COUNT_ZERO; offset < len(source); offset += chunk_size {
		end_offset := offset + chunk_size
		if end_offset > len(source) {
			end_offset = len(source)
		}
		sha512.Digest_Write(digest, source[offset:end_offset])
	}
}

func state_domains(t *testing.T) {
	kinds := [...]sha512.Kind{
		sha512.KIND_SHA_512_224,
		sha512.KIND_SHA_512_256,
		sha512.KIND_SHA_384,
		sha512.KIND_SHA_512,
	}
	message_sizes := [...]sha512.Message_Size{
		sha512.Message_Size(sha512.MESSAGE_SIZE_MINIMUM),
		sha512.Message_Size(sha512.MESSAGE_SIZE_MINIMUM + TEST_COUNT_ONE),
		sha512.Message_Size(sha512.MESSAGE_SIZE_MINIMUM + TEST_COUNT_TWO),
		sha512.Message_Size(sha512.MESSAGE_SIZE_MAXIMUM),
	}
	var destination [sha512.DESTINATION_SIZE_MAXIMUM]byte
	for _, kind := range kinds {
		for _, message_size := range message_sizes {
			var source_digest sha512.Digest
			sha512.Digest_Init(&source_digest, kind)
			source_digest.Message_Size = message_size
			source_digest.Buffer_Count = sha512.Buffer_Count(
				uint64(message_size) % sha512.BLOCK_SIZE,
			)

			sha512.Digest_Size(&source_digest)
			sha512.Digest_Block_Size(&source_digest)
			if kind == sha512.KIND_SHA_512_224 {
				sha512.Digest_Sum_512_224(&source_digest)
			} else {
				testify.Panics(t, func() {
					sha512.Digest_Sum_512_224(&source_digest)
				})
			}
			if kind == sha512.KIND_SHA_512_256 {
				sha512.Digest_Sum_512_256(&source_digest)
			} else {
				testify.Panics(t, func() {
					sha512.Digest_Sum_512_256(&source_digest)
				})
			}
			if kind == sha512.KIND_SHA_384 {
				sha512.Digest_Sum_384(&source_digest)
			} else {
				testify.Panics(t, func() { sha512.Digest_Sum_384(&source_digest) })
			}
			if kind == sha512.KIND_SHA_512 {
				sha512.Digest_Sum_512(&source_digest)
			} else {
				testify.Panics(t, func() { sha512.Digest_Sum_512(&source_digest) })
			}
			sha512.Digest_Sum_Into(&source_digest, destination[:])
			sha512.Digest_Write(&source_digest, nil)

			var destination_digest sha512.Digest
			sha512.Digest_Init(&destination_digest, kind)
			destination_digest.Message_Size = message_size
			destination_digest.Buffer_Count = source_digest.Buffer_Count
			sha512.Digest_Clone_Into(&destination_digest, &source_digest)

			reset_digest := source_digest
			sha512.Digest_Reset(&reset_digest)
			init_digest := source_digest
			sha512.Digest_Init(&init_digest, kind)
		}
	}
}

const TEST_COUNT_ZERO = bytes.SLICE_SIZE_MINIMUM
const TEST_COUNT_ONE = TEST_COUNT_ZERO + binary.UINT_8_SIZE
const TEST_COUNT_TWO = TEST_COUNT_ONE + TEST_COUNT_ONE
const TEST_MULTI_BLOCK_CHUNK_SIZE = sha512.BLOCK_SIZE*binary.UINT_16_SIZE + TEST_COUNT_ONE

type allocation_fixture struct {
	Digest        sha512.Digest
	Source        sha512.Source
	Output        [sha512.DIGEST_512_SIZE]byte
	Value_224     sha512.Value_224
	Value_256     sha512.Value_256
	Value_384     sha512.Value_384
	Value_512     sha512.Value_512
	Count         sha512.Count
	Output_Count  sha512.Output_Count
	Output_Status sha512.Output_Status
}
