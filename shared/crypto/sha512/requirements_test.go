package sha512_test

import (
	"testing"

	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/crypto/sha512"
	"local/james-orcales/shared/encoding/binary"
	"local/james-orcales/shared/encoding/hex"
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
	testify.Equal(t, digest_sum(&source), digest_sum(&clone))
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
	tests := [...]struct {
		Kind sha512.Kind
		Want string
	}{
		{Kind: sha512.KIND_SHA_512_224, Want: REFERENCE_512_224_ABC},
		{Kind: sha512.KIND_SHA_512_256, Want: REFERENCE_512_256_ABC},
		{Kind: sha512.KIND_SHA_384, Want: REFERENCE_384_ABC},
		{Kind: sha512.KIND_SHA_512, Want: REFERENCE_512_ABC},
	}
	for _, test := range tests {
		testify.Equal(t, test.Want, encoded(t, checksum(test.Kind, source)))
	}
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
		for _, kind := range test_kinds() {
			want := checksum(kind, source[:size])
			var digest sha512.Digest
			sha512.Digest_Init(&digest, kind)
			sha512.Digest_Write(&digest, source[:size])
			testify.Equal(t, want, digest_sum(&digest))
		}
	}
	var reference_224 sha512.Digest
	sha512.Digest_Init(&reference_224, sha512.KIND_SHA_512_224)
	write_chunks(&reference_224, source[:], sha512.BLOCK_SIZE+TEST_COUNT_ONE)
	var digest_224 sha512.Digest
	sha512.Digest_Init(&digest_224, sha512.KIND_SHA_512_224)
	write_divided(&digest_224, source[:])
	testify.Equal(
		t, digest_sum(&reference_224), digest_sum(&digest_224),
	)
	var reference_256 sha512.Digest
	sha512.Digest_Init(&reference_256, sha512.KIND_SHA_512_256)
	write_chunks(&reference_256, source[:], sha512.BLOCK_SIZE+TEST_COUNT_ONE)
	var digest_256 sha512.Digest
	sha512.Digest_Init(&digest_256, sha512.KIND_SHA_512_256)
	write_divided(&digest_256, source[:])
	testify.Equal(
		t, digest_sum(&reference_256), digest_sum(&digest_256),
	)
	var reference_384 sha512.Digest
	sha512.Digest_Init(&reference_384, sha512.KIND_SHA_384)
	write_chunks(&reference_384, source[:], sha512.BLOCK_SIZE+TEST_COUNT_ONE)
	var digest_384 sha512.Digest
	sha512.Digest_Init(&digest_384, sha512.KIND_SHA_384)
	write_divided(&digest_384, source[:])
	testify.Equal(
		t, digest_sum(&reference_384), digest_sum(&digest_384),
	)
	var reference_512 sha512.Digest
	sha512.Digest_Init(&reference_512, sha512.KIND_SHA_512)
	write_chunks(&reference_512, source[:], sha512.BLOCK_SIZE+TEST_COUNT_ONE)
	var digest_512 sha512.Digest
	sha512.Digest_Init(&digest_512, sha512.KIND_SHA_512)
	write_divided(&digest_512, source[:])
	testify.Equal(
		t, digest_sum(&reference_512), digest_sum(&digest_512),
	)
	want_chunks := checksum(sha512.KIND_SHA_512, source[:sha512.SOURCE_SIZE_MAXIMUM])
	for _, chunk_size := range [...]int{
		TEST_COUNT_ONE, sha512.BLOCK_SIZE - TEST_COUNT_ONE, sha512.BLOCK_SIZE,
		sha512.BLOCK_SIZE + TEST_COUNT_ONE, TEST_MULTI_BLOCK_CHUNK_SIZE,
	} {
		sha512.Digest_Init(&digest_512, sha512.KIND_SHA_512)
		write_chunks(&digest_512, source[:sha512.SOURCE_SIZE_MAXIMUM], chunk_size)
		testify.Equal(t, want_chunks, digest_sum(&digest_512), chunk_size)
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
	testify.Equal(t, [sha512.DIGEST_224_SIZE - TEST_COUNT_ONE]byte{}, short)
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
	testify.Equal(t, checksum(sha512.KIND_SHA_512, nil), digest_sum(&digest))
}

// Test_Bounds rejects oversized calls, wrong kinds, and uninitialized caller state.
func Test_Bounds(t *testing.T) {
	var digest sha512.Digest
	testify.Panics(t, func() { sha512.Digest_Write(&digest, nil) })
	sha512.Digest_Init(&digest, sha512.KIND_SHA_512)
	var source [sha512.SOURCE_SIZE_MAXIMUM + TEST_COUNT_ONE]byte
	testify.Panics(t, func() { sha512.Digest_Write(&digest, source[:]) })
	var output [sha512.DIGEST_512_SIZE]byte
	testify.Panics(t, func() {
		sha512.Checksum_Into(output[:], sha512.KIND_SHA_512, source[:])
	})
	testify.Panics(t, func() {
		sha512.Digest_Init(&digest, sha512.Kind(bits.WORD_8_MAXIMUM))
	})
	var destination [sha512.DESTINATION_SIZE_MAXIMUM + TEST_COUNT_ONE]byte
	testify.Panics(t, func() { sha512.Digest_Sum_Into(&digest, destination[:]) })
	digest.Message_Size = sha512.Message_Size(sha512.MESSAGE_SIZE_MAXIMUM)
	digest.Buffer_Count = sha512.Buffer_Count(sha512.MESSAGE_SIZE_MAXIMUM % sha512.BLOCK_SIZE)
	testify.Panics(t, func() { sha512.Digest_Write(&digest, sha512.Source("x")) })
}

// Test_Invariant_Domains drives each valid state and caller-storage boundary through runtime APIs.
func Test_Invariant_Domains(t *testing.T) {
	kinds := test_kinds()
	var source [sha512.SOURCE_SIZE_MAXIMUM]byte
	for _, size := range [...]int{
		sha512.SOURCE_SIZE_MINIMUM,
		sha512.SOURCE_SIZE_MINIMUM + TEST_COUNT_ONE,
		sha512.SOURCE_SIZE_MINIMUM + TEST_COUNT_TWO,
		sha512.SOURCE_SIZE_MAXIMUM,
	} {
		for _, kind := range kinds {
			var output [sha512.DIGEST_512_SIZE]byte
			sha512.Checksum_Into(output[:], kind, source[:size])
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
			sha512.Checksum_Into(destination[:size], kind, nil)
		}
	}

	state_domains(t)
}

// Test_Allocation proves every exported runtime operation remains stack-owned.
func Test_Allocation(t *testing.T) {
	for _, kind := range test_kinds() {
		fixture := allocation_fixture{
			Source: sha512.Source("abc"),
			Output: make(sha512.Destination, sha512.DIGEST_512_SIZE),
		}
		sha512.Digest_Init(&fixture.Digest, kind)
		testify.Zero_Allocation(t, func() { sha512.Digest_Init(&fixture.Digest, kind) })
		testify.Zero_Allocation(t, func() {
			fixture.Count = sha512.Digest_Write(&fixture.Digest, fixture.Source)
		})
		testify.Zero_Allocation(t, func() {
			fixture.Output_Count, fixture.Output_Status = sha512.Digest_Sum_Into(
				&fixture.Digest, fixture.Output,
			)
		})
		testify.Zero_Allocation(t, func() { sha512.Digest_Reset(&fixture.Digest) })
		testify.Zero_Allocation(t, func() {
			sha512.Digest_Clone_Into(&fixture.Clone, &fixture.Digest)
		})
		testify.Zero_Allocation(t, func() {
			fixture.Size = sha512.Digest_Size(&fixture.Digest)
		})
		testify.Zero_Allocation(t, func() {
			fixture.Block_Size = sha512.Digest_Block_Size(&fixture.Digest)
		})
		testify.Zero_Allocation(t, func() {
			fixture.Output_Count, fixture.Output_Status = sha512.Checksum_Into(
				fixture.Output, kind, fixture.Source,
			)
		})
	}
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
	kinds := test_kinds()
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
	test_state_invariant_domains(t)
}

const TEST_COUNT_ZERO = bytes.SLICE_SIZE_MINIMUM
const TEST_COUNT_ONE = TEST_COUNT_ZERO + binary.UINT_8_SIZE
const TEST_COUNT_TWO = TEST_COUNT_ONE + TEST_COUNT_ONE
const TEST_MULTI_BLOCK_CHUNK_SIZE = sha512.BLOCK_SIZE*binary.UINT_16_SIZE + TEST_COUNT_ONE

const REFERENCE_512_224_ABC = "4634270f707b6a54daae7530460842e2" +
	"0e37ed265ceee9a43e8924aa"

const REFERENCE_512_256_ABC = "53048e2681941ef99b2e29b76b4c7dab" +
	"e4c2d0c634fc6d46e0e2f13107e7af23"

const REFERENCE_384_ABC = "cb00753f45a35e8bb5a03d699ac65007" +
	"272c32ab0eded1631a8b605a43ff5bed" +
	"8086072ba1e7cc2358baeca134c825a7"

const REFERENCE_512_ABC = "ddaf35a193617abacc417349ae204131" +
	"12e6fa4e89a97ea20a9eeee64b55d39a" +
	"2192992a274fc1a836ba3c23a3feebbd" +
	"454d4423643ce80e2a9ac94fa54ca49f"

type allocation_fixture struct {
	Digest        sha512.Digest
	Clone         sha512.Digest
	Source        sha512.Source
	Output        sha512.Destination
	Count         sha512.Count
	Output_Count  sha512.Output_Count
	Output_Status sha512.Output_Status
	Size          sha512.Size
	Block_Size    sha512.Block_Size
}

type test_output []byte

type kind_list []sha512.Kind

func test_kinds() (kinds kind_list) {
	return kind_list{
		sha512.KIND_SHA_512_224,
		sha512.KIND_SHA_512_256,
		sha512.KIND_SHA_384,
		sha512.KIND_SHA_512,
	}
}

func checksum(kind sha512.Kind, source sha512.Source) (output test_output) {
	output = make(test_output, int(output_size(kind)))
	sha512.Checksum_Into(sha512.Destination(output), kind, source)
	return output
}

func encoded(t *testing.T, source test_output) (value string) {
	t.Helper()
	var output [sha512.DIGEST_512_SIZE * hex.ENCODED_BYTE_SIZE]byte
	count, status := hex.Encode_Into(output[:], hex.Source(source))
	testify.Equal(t, hex.Encode_Status(hex.STATUS_OK), status)
	return string(output[:count])
}

func digest_sum(digest *sha512.Digest) (output test_output) {
	output = make(test_output, int(sha512.Digest_Size(digest)))
	sha512.Digest_Sum_Into(digest, sha512.Destination(output))
	return output
}

func output_size(kind sha512.Kind) (size sha512.Size) {
	switch kind {
	case sha512.KIND_SHA_512_224:
		return sha512.DIGEST_224_SIZE
	case sha512.KIND_SHA_512_256:
		return sha512.DIGEST_256_SIZE
	case sha512.KIND_SHA_384:
		return sha512.DIGEST_384_SIZE
	default:
		return sha512.DIGEST_512_SIZE
	}
}

func test_state_invariant_domains(t *testing.T) {
	for _, value := range [...]uint64{
		bits.WORD_64_MINIMUM,
		uint64(binary.UINT_8_SIZE),
		uint64(binary.UINT_16_SIZE),
		bits.WORD_64_MAXIMUM,
	} {
		for _, kind := range test_kinds() {
			digest := domain_digest(value, kind)
			drive_digest_operations(&digest, kind)
			for _, buffer_count := range [...]sha512.Buffer_Count{
				sha512.Buffer_Count(binary.UINT_8_SIZE),
				sha512.Buffer_Count(binary.UINT_64_SIZE),
			} {
				buffered := digest
				buffered.Buffer_Count = buffer_count
				buffered.Message_Size = sha512.Message_Size(buffer_count)
				sha512.Digest_Write(&buffered, sha512.Source{byte(value)})
			}
		}
	}

	var uninitialized sha512.Digest
	testify.Panics(t, func() { sha512.Digest_Reset(&uninitialized) })
	testify.Panics(t, func() { sha512.Digest_Sum_Into(&uninitialized, nil) })
	testify.Panics(t, func() { sha512.Digest_Size(&uninitialized) })
	testify.Panics(t, func() { sha512.Digest_Block_Size(&uninitialized) })
	initialized := domain_digest(bits.WORD_64_MINIMUM, sha512.KIND_SHA_512)
	sha512.Digest_Clone_Into(&uninitialized, &initialized)
	var invalid_source sha512.Digest
	testify.Panics(t, func() { sha512.Digest_Clone_Into(&initialized, &invalid_source) })
}

func domain_digest(value uint64, kind sha512.Kind) (digest sha512.Digest) {
	digest.State = sha512.State{
		Lane_0: sha512.State_Lane_0(value),
		Lane_1: sha512.State_Lane_1(value),
		Lane_2: sha512.State_Lane_2(value),
		Lane_3: sha512.State_Lane_3(value),
		Lane_4: sha512.State_Lane_4(value),
		Lane_5: sha512.State_Lane_5(value),
		Lane_6: sha512.State_Lane_6(value),
		Lane_7: sha512.State_Lane_7(value),
	}
	digest.Buffer = sha512.Buffer{
		Lane_1:  sha512.Buffer_Lane_1(value),
		Lane_2:  sha512.Buffer_Lane_2(value),
		Lane_3:  sha512.Buffer_Lane_3(value),
		Lane_4:  sha512.Buffer_Lane_4(value),
		Lane_5:  sha512.Buffer_Lane_5(value),
		Lane_6:  sha512.Buffer_Lane_6(value),
		Lane_7:  sha512.Buffer_Lane_7(value),
		Lane_8:  sha512.Buffer_Lane_8(value),
		Lane_9:  sha512.Buffer_Lane_9(value),
		Lane_10: sha512.Buffer_Lane_10(value),
		Lane_11: sha512.Buffer_Lane_11(value),
		Lane_12: sha512.Buffer_Lane_12(value),
		Lane_13: sha512.Buffer_Lane_13(value),
		Lane_14: sha512.Buffer_Lane_14(value),
		Lane_15: sha512.Buffer_Lane_15(value),
		Lane_16: sha512.Buffer_Lane_16(value),
	}
	digest.Kind = kind
	digest.Ready = true
	return digest
}

func drive_digest_operations(digest *sha512.Digest, kind sha512.Kind) {
	var destination [sha512.DIGEST_512_SIZE]byte
	sha512.Digest_Write(digest, nil)
	sha512.Digest_Sum_Into(digest, destination[:])
	sha512.Digest_Size(digest)
	sha512.Digest_Block_Size(digest)
	clone_source := *digest
	clone_destination := *digest
	sha512.Digest_Clone_Into(&clone_destination, &clone_source)
	reset := *digest
	sha512.Digest_Reset(&reset)
	initialized := *digest
	sha512.Digest_Init(&initialized, kind)
}
