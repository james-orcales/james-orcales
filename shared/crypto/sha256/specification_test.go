package sha256_test

import (
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
	testify.Equal(t, digest_sum(&digest_256), digest_sum(&clone))
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
	want_224 := test_output(REFERENCE_224_ABC)
	want_256 := test_output(REFERENCE_256_ABC)
	testify.Equal(t, want_224, checksum(sha256.KIND_SHA_224, source))
	testify.Equal(t, want_256, checksum(sha256.KIND_SHA_256, source))
	var digest_224 sha256.Digest
	sha256.Digest_Init(&digest_224, sha256.KIND_SHA_224)
	sha256.Digest_Write(&digest_224, source[:TEST_COUNT_ONE])
	sha256.Digest_Write(&digest_224, source[TEST_COUNT_ONE:])
	testify.Equal(t, want_224, digest_sum(&digest_224))
	var digest_256 sha256.Digest
	sha256.Digest_Init(&digest_256, sha256.KIND_SHA_256)
	sha256.Digest_Write(&digest_256, source[:TEST_COUNT_ONE])
	sha256.Digest_Write(&digest_256, source[TEST_COUNT_ONE:])
	testify.Equal(t, want_256, digest_sum(&digest_256))
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
		for _, kind := range [...]sha256.Kind{
			sha256.KIND_SHA_224, sha256.KIND_SHA_256,
		} {
			want := checksum(kind, source[:size])
			var digest sha256.Digest
			sha256.Digest_Init(&digest, kind)
			sha256.Digest_Write(&digest, source[:size])
			testify.Equal(t, want, digest_sum(&digest))
		}
	}
	var reference_224 sha256.Digest
	sha256.Digest_Init(&reference_224, sha256.KIND_SHA_224)
	write_chunks(&reference_224, source[:], sha256.BLOCK_SIZE+TEST_COUNT_ONE)
	var digest_224 sha256.Digest
	sha256.Digest_Init(&digest_224, sha256.KIND_SHA_224)
	sha256.Digest_Write(&digest_224, source[:sha256.SOURCE_SIZE_MAXIMUM])
	sha256.Digest_Write(&digest_224, source[sha256.SOURCE_SIZE_MAXIMUM:])
	testify.Equal(
		t, digest_sum(&reference_224),
		digest_sum(&digest_224),
	)
	var reference_256 sha256.Digest
	sha256.Digest_Init(&reference_256, sha256.KIND_SHA_256)
	write_chunks(&reference_256, source[:], sha256.BLOCK_SIZE+TEST_COUNT_ONE)
	var digest_256 sha256.Digest
	sha256.Digest_Init(&digest_256, sha256.KIND_SHA_256)
	sha256.Digest_Write(&digest_256, source[:sha256.SOURCE_SIZE_MAXIMUM])
	sha256.Digest_Write(&digest_256, source[sha256.SOURCE_SIZE_MAXIMUM:])
	testify.Equal(
		t, digest_sum(&reference_256),
		digest_sum(&digest_256),
	)
	want_chunks := checksum(sha256.KIND_SHA_256, source[:sha256.SOURCE_SIZE_MAXIMUM])
	for _, chunk_size := range [...]int{
		TEST_COUNT_ONE, sha256.BLOCK_SIZE - TEST_COUNT_ONE, sha256.BLOCK_SIZE,
		sha256.BLOCK_SIZE + TEST_COUNT_ONE, TEST_MULTI_BLOCK_CHUNK_SIZE,
	} {
		sha256.Digest_Init(&digest_256, sha256.KIND_SHA_256)
		write_chunks(&digest_256, source[:sha256.SOURCE_SIZE_MAXIMUM], chunk_size)
		testify.Equal(t, want_chunks, digest_sum(&digest_256), chunk_size)
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
	testify.Equal(t, [sha256.DIGEST_224_SIZE - TEST_COUNT_ONE]byte{}, short)
	var output_224 [sha256.DIGEST_224_SIZE]byte
	count, status = sha256.Digest_Sum_Into(&digest, output_224[:])
	testify.Equal(t, sha256.OUTPUT_COUNT_224_REQUIRED, count)
	testify.Equal(t, sha256.OUTPUT_STATUS_OK, status)
	testify.Equal(
		t, test_output(REFERENCE_224_ABC),
		test_output(output_224[:]),
	)
	testify.Equal(t, before, digest)
	sha256.Digest_Init(&digest, sha256.KIND_SHA_256)
	var output_256 [sha256.DIGEST_256_SIZE]byte
	count, status = sha256.Digest_Sum_Into(&digest, output_256[:])
	testify.Equal(t, sha256.OUTPUT_COUNT_256_REQUIRED, count)
	testify.Equal(t, sha256.OUTPUT_STATUS_OK, status)
	sha256.Digest_Reset(&digest)
	testify.Equal(t, checksum(sha256.KIND_SHA_256, nil), digest_sum(&digest))
}

// Test_Bounds rejects oversized calls, wrong kinds, and uninitialized caller state.
func Test_Bounds(t *testing.T) {
	var digest sha256.Digest
	testify.Panics(t, func() { sha256.Digest_Write(&digest, nil) })
	sha256.Digest_Init(&digest, sha256.KIND_SHA_256)
	var source [sha256.SOURCE_SIZE_MAXIMUM + TEST_COUNT_ONE]byte
	testify.Panics(t, func() { sha256.Digest_Write(&digest, source[:]) })
	var output [sha256.DIGEST_256_SIZE]byte
	testify.Panics(t, func() {
		sha256.Checksum_Into(output[:], sha256.KIND_SHA_256, source[:])
	})
	testify.Panics(t, func() {
		sha256.Digest_Init(&digest, sha256.Kind(bits.WORD_8_MAXIMUM))
	})
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
		for _, kind := range [...]sha256.Kind{sha256.KIND_SHA_224, sha256.KIND_SHA_256} {
			var output [sha256.DIGEST_256_SIZE]byte
			sha256.Checksum_Into(output[:], kind, source[:size])
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
			sha256.Checksum_Into(destination[:size], kind, nil)
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
	test_state_invariant_domains(t)
}

// Test_Allocation proves every exported runtime operation remains stack-owned.
func Test_Allocation(t *testing.T) {
	for _, kind := range [...]sha256.Kind{sha256.KIND_SHA_224, sha256.KIND_SHA_256} {
		fixture := allocation_fixture{
			Source: sha256.Source("abc"),
			Output: make(sha256.Destination, sha256.DIGEST_256_SIZE),
		}
		sha256.Digest_Init(&fixture.Digest, kind)
		testify.Zero_Allocation(t, func() { sha256.Digest_Init(&fixture.Digest, kind) })
		testify.Zero_Allocation(t, func() {
			fixture.Count = sha256.Digest_Write(&fixture.Digest, fixture.Source)
		})
		testify.Zero_Allocation(t, func() {
			fixture.Output_Count, fixture.Output_Status = sha256.Digest_Sum_Into(
				&fixture.Digest, fixture.Output,
			)
		})
		testify.Zero_Allocation(t, func() { sha256.Digest_Reset(&fixture.Digest) })
		testify.Zero_Allocation(t, func() {
			sha256.Digest_Clone_Into(&fixture.Clone, &fixture.Digest)
		})
		testify.Zero_Allocation(t, func() {
			fixture.Size = sha256.Digest_Size(&fixture.Digest)
		})
		testify.Zero_Allocation(t, func() {
			fixture.Block_Size = sha256.Digest_Block_Size(&fixture.Digest)
		})
		testify.Zero_Allocation(t, func() {
			fixture.Output_Count, fixture.Output_Status = sha256.Checksum_Into(
				fixture.Output, kind, fixture.Source,
			)
		})
	}
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

const REFERENCE_224_ABC = "\x23\x09\x7d\x22\x34\x05\xd8\x22" +
	"\x86\x42\xa4\x77\xbd\xa2\x55\xb3\x2a\xad\xbc\xe4" +
	"\xbd\xa0\xb3\xf7\xe3\x6c\x9d\xa7"

const REFERENCE_256_ABC = "\xba\x78\x16\xbf\x8f\x01\xcf\xea" +
	"\x41\x41\x40\xde\x5d\xae\x22\x23\xb0\x03\x61\xa3" +
	"\x96\x17\x7a\x9c\xb4\x10\xff\x61\xf2\x00\x15\xad"

type allocation_fixture struct {
	Digest        sha256.Digest
	Clone         sha256.Digest
	Source        sha256.Source
	Output        sha256.Destination
	Count         sha256.Count
	Output_Count  sha256.Output_Count
	Output_Status sha256.Output_Status
	Size          sha256.Size
	Block_Size    sha256.Block_Size
}

type test_output []byte

func checksum(kind sha256.Kind, source sha256.Source) (output test_output) {
	output = make(test_output, int(output_size(kind)))
	sha256.Checksum_Into(sha256.Destination(output), kind, source)
	return output
}

func digest_sum(digest *sha256.Digest) (output test_output) {
	output = make(test_output, int(sha256.Digest_Size(digest)))
	sha256.Digest_Sum_Into(digest, sha256.Destination(output))
	return output
}

func output_size(kind sha256.Kind) (size sha256.Size) {
	if kind == sha256.KIND_SHA_224 {
		return sha256.DIGEST_224_SIZE
	}
	return sha256.DIGEST_256_SIZE
}

func test_state_invariant_domains(t *testing.T) {
	for _, value := range [...]uint64{
		bits.WORD_64_MINIMUM,
		uint64(binary.UINT_8_SIZE),
		uint64(binary.UINT_16_SIZE),
		bits.WORD_64_MAXIMUM,
	} {
		for _, kind := range [...]sha256.Kind{sha256.KIND_SHA_224, sha256.KIND_SHA_256} {
			digest := domain_digest(value, kind)
			drive_digest_operations(&digest, kind)
			for _, buffer_count := range [...]sha256.Buffer_Count{
				sha256.Buffer_Count(binary.UINT_8_SIZE),
				sha256.Buffer_Count(binary.UINT_64_SIZE),
			} {
				buffered := digest
				buffered.Buffer_Count = buffer_count
				buffered.Message_Size = sha256.Message_Size(buffer_count)
				sha256.Digest_Write(&buffered, sha256.Source{byte(value)})
			}
		}
	}

	var uninitialized sha256.Digest
	testify.Panics(t, func() { sha256.Digest_Reset(&uninitialized) })
	testify.Panics(t, func() { sha256.Digest_Sum_Into(&uninitialized, nil) })
	testify.Panics(t, func() { sha256.Digest_Size(&uninitialized) })
	testify.Panics(t, func() { sha256.Digest_Block_Size(&uninitialized) })
	initialized := domain_digest(bits.WORD_64_MINIMUM, sha256.KIND_SHA_256)
	sha256.Digest_Clone_Into(&uninitialized, &initialized)
	var invalid_source sha256.Digest
	testify.Panics(t, func() { sha256.Digest_Clone_Into(&initialized, &invalid_source) })
}

func domain_digest(value uint64, kind sha256.Kind) (digest sha256.Digest) {
	digest.State = sha256.State{
		Lane_0: sha256.State_Lane_0(value),
		Lane_1: sha256.State_Lane_1(value),
		Lane_2: sha256.State_Lane_2(value),
		Lane_3: sha256.State_Lane_3(value),
		Lane_4: sha256.State_Lane_4(value),
		Lane_5: sha256.State_Lane_5(value),
		Lane_6: sha256.State_Lane_6(value),
		Lane_7: sha256.State_Lane_7(value),
	}
	digest.Buffer = sha256.Buffer{
		Lane_1: sha256.Buffer_Lane_1(value),
		Lane_2: sha256.Buffer_Lane_2(value),
		Lane_3: sha256.Buffer_Lane_3(value),
		Lane_4: sha256.Buffer_Lane_4(value),
		Lane_5: sha256.Buffer_Lane_5(value),
		Lane_6: sha256.Buffer_Lane_6(value),
		Lane_7: sha256.Buffer_Lane_7(value),
		Lane_8: sha256.Buffer_Lane_8(value),
	}
	digest.Kind = kind
	digest.Ready = true
	return digest
}

func drive_digest_operations(digest *sha256.Digest, kind sha256.Kind) {
	var destination [sha256.DIGEST_256_SIZE]byte
	sha256.Digest_Write(digest, nil)
	sha256.Digest_Sum_Into(digest, destination[:])
	sha256.Digest_Size(digest)
	sha256.Digest_Block_Size(digest)
	clone_source := *digest
	clone_destination := *digest
	sha256.Digest_Clone_Into(&clone_destination, &clone_source)
	reset := *digest
	sha256.Digest_Reset(&reset)
	initialized := *digest
	sha256.Digest_Init(&initialized, kind)
}
