package md5_test

import (
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
	var source_output [md5.DIGEST_SIZE]byte
	var destination_output [md5.DIGEST_SIZE]byte
	md5.Digest_Sum_Into(&source, source_output[:])
	md5.Digest_Sum_Into(&destination, destination_output[:])
	testify.Equal(t, source_output, destination_output)
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
	tests := [...]struct {
		Source md5.Source
		Want   [md5.DIGEST_SIZE]byte
	}{
		{
			Source: md5.Source(""),
			Want: [md5.DIGEST_SIZE]byte{
				0xd4, 0x1d, 0x8c, 0xd9, 0x8f, 0x00, 0xb2, 0x04,
				0xe9, 0x80, 0x09, 0x98, 0xec, 0xf8, 0x42, 0x7e,
			},
		},
		{
			Source: md5.Source("abc"),
			Want: [md5.DIGEST_SIZE]byte{
				0x90, 0x01, 0x50, 0x98, 0x3c, 0xd2, 0x4f, 0xb0,
				0xd6, 0x96, 0x3f, 0x7d, 0x28, 0xe1, 0x7f, 0x72,
			},
		},
	}
	for _, test := range tests {
		testify.Equal(t, test.Want[:], checksum(test.Source))
		var digest md5.Digest
		md5.Digest_Init(&digest)
		midpoint := len(test.Source) / TEST_COUNT_TWO
		testify.Equal(
			t, md5.Count(midpoint), md5.Digest_Write(&digest, test.Source[:midpoint]),
		)
		testify.Equal(
			t, md5.Count(len(test.Source)-midpoint),
			md5.Digest_Write(&digest, test.Source[midpoint:]),
		)
		var output [md5.DIGEST_SIZE]byte
		md5.Digest_Sum_Into(&digest, output[:])
		testify.Equal(t, test.Want, output)
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
		want := checksum(source[:size])
		var digest md5.Digest
		md5.Digest_Init(&digest)
		md5.Digest_Write(&digest, source[:size])
		var output [md5.DIGEST_SIZE]byte
		md5.Digest_Sum_Into(&digest, output[:])
		testify.Equal(t, want, output[:])
	}
	var reference md5.Digest
	md5.Digest_Init(&reference)
	write_chunks(&reference, source[:], md5.BLOCK_SIZE+TEST_COUNT_ONE)
	var want [md5.DIGEST_SIZE]byte
	md5.Digest_Sum_Into(&reference, want[:])
	var digest md5.Digest
	md5.Digest_Init(&digest)
	md5.Digest_Write(&digest, source[:md5.SOURCE_SIZE_MAXIMUM])
	md5.Digest_Write(&digest, source[md5.SOURCE_SIZE_MAXIMUM:])
	var output [md5.DIGEST_SIZE]byte
	md5.Digest_Sum_Into(&digest, output[:])
	testify.Equal(t, want, output)
	md5.Digest_Write(&digest, md5.Source("x"))
	md5.Digest_Write(&reference, md5.Source("x"))
	var want_extended [md5.DIGEST_SIZE]byte
	md5.Digest_Sum_Into(&reference, want_extended[:])
	md5.Digest_Sum_Into(&digest, output[:])
	testify.Equal(t, want_extended, output)
	want_chunks := checksum(source[:md5.SOURCE_SIZE_MAXIMUM])
	for _, chunk_size := range [...]int{
		TEST_COUNT_ONE, md5.BLOCK_SIZE - TEST_COUNT_ONE, md5.BLOCK_SIZE,
		md5.BLOCK_SIZE + TEST_COUNT_ONE, TEST_MULTI_BLOCK_CHUNK_SIZE,
	} {
		md5.Digest_Init(&digest)
		write_chunks(&digest, source[:md5.SOURCE_SIZE_MAXIMUM], chunk_size)
		md5.Digest_Sum_Into(&digest, output[:])
		testify.Equal(t, want_chunks, output[:], chunk_size)
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
	var repeated [md5.DIGEST_SIZE]byte
	md5.Digest_Sum_Into(&digest, repeated[:])
	testify.Equal(t, output, repeated)
	testify.Equal(t, before, digest)
	md5.Digest_Reset(&digest)
	var empty_output [md5.DIGEST_SIZE]byte
	md5.Checksum_Into(empty_output[:], nil)
	md5.Digest_Sum_Into(&digest, repeated[:])
	testify.Equal(t, empty_output, repeated)
}

// Test_Bounds rejects oversized calls and uninitialized caller state.
func Test_Bounds(t *testing.T) {
	var digest md5.Digest
	testify.Panics(t, func() { md5.Digest_Write(&digest, nil) })
	md5.Digest_Init(&digest)
	var source [md5.SOURCE_SIZE_MAXIMUM + TEST_COUNT_ONE]byte
	testify.Panics(t, func() { md5.Digest_Write(&digest, source[:]) })
	var output [md5.DIGEST_SIZE]byte
	testify.Panics(t, func() { md5.Checksum_Into(output[:], source[:]) })
	var destination [md5.DESTINATION_SIZE_MAXIMUM + TEST_COUNT_ONE]byte
	testify.Panics(t, func() { md5.Digest_Sum_Into(&digest, destination[:]) })
	digest.Message_Size = md5.Message_Size(md5.MESSAGE_SIZE_MAXIMUM)
	digest.Buffer_Count = md5.Buffer_Count(md5.MESSAGE_SIZE_MAXIMUM % md5.BLOCK_SIZE)
	testify.Panics(t, func() { md5.Digest_Write(&digest, md5.Source("x")) })
}

// Test_Invariant_Domains drives each valid state and caller-storage boundary through runtime APIs.
func Test_Invariant_Domains(t *testing.T) {
	var source [md5.SOURCE_SIZE_MAXIMUM]byte
	var destination [md5.DESTINATION_SIZE_MAXIMUM]byte
	for _, size := range [...]int{
		md5.SOURCE_SIZE_MINIMUM,
		md5.SOURCE_SIZE_MINIMUM + TEST_COUNT_ONE,
		md5.SOURCE_SIZE_MINIMUM + TEST_COUNT_TWO,
		md5.SOURCE_SIZE_MAXIMUM,
	} {
		md5.Checksum_Into(destination[:], source[:size])
		var digest md5.Digest
		md5.Digest_Init(&digest)
		md5.Digest_Write(&digest, source[:size])
	}

	for _, size := range [...]int{
		md5.DESTINATION_SIZE_MINIMUM,
		md5.DESTINATION_SIZE_MINIMUM + TEST_COUNT_ONE,
		md5.DESTINATION_SIZE_MINIMUM + TEST_COUNT_TWO,
		md5.DESTINATION_SIZE_MAXIMUM,
	} {
		var digest md5.Digest
		md5.Digest_Init(&digest)
		md5.Digest_Sum_Into(&digest, destination[:size])
		md5.Checksum_Into(destination[:size], nil)
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
	test_state_invariant_domains(t)
}

// Test_Allocation proves every exported runtime operation remains stack-owned.
func Test_Allocation(t *testing.T) {
	fixture := allocation_fixture{
		Source: md5.Source("abc"), Output: make(md5.Destination, md5.DIGEST_SIZE),
	}
	md5.Digest_Init(&fixture.Digest)
	testify.Zero_Allocation(t, func() { md5.Digest_Init(&fixture.Digest) })
	testify.Zero_Allocation(t, func() {
		fixture.Count = md5.Digest_Write(&fixture.Digest, fixture.Source)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Output_Count, fixture.Output_Status = md5.Digest_Sum_Into(
			&fixture.Digest, fixture.Output,
		)
	})
	testify.Zero_Allocation(t, func() { md5.Digest_Reset(&fixture.Digest) })
	testify.Zero_Allocation(t, func() {
		md5.Digest_Clone_Into(&fixture.Clone, &fixture.Digest)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Size = md5.Digest_Size(&fixture.Digest)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Block_Size = md5.Digest_Block_Size(&fixture.Digest)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Output_Count, fixture.Output_Status = md5.Checksum_Into(
			fixture.Output, fixture.Source,
		)
	})
}

func checksum(source md5.Source) (output []byte) {
	output = make([]byte, md5.DIGEST_SIZE)
	md5.Checksum_Into(output, source)
	return output
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
	Clone         md5.Digest
	Source        md5.Source
	Output        md5.Destination
	Count         md5.Count
	Output_Count  md5.Output_Count
	Output_Status md5.Output_Status
	Size          md5.Size
	Block_Size    md5.Block_Size
}

func test_state_invariant_domains(t *testing.T) {
	for _, value := range [...]uint64{
		bits.WORD_64_MINIMUM,
		uint64(binary.UINT_8_SIZE),
		uint64(binary.UINT_16_SIZE),
		bits.WORD_64_MAXIMUM,
	} {
		digest := domain_digest(value)
		drive_digest_operations(&digest)
		for _, buffer_count := range [...]md5.Buffer_Count{
			md5.Buffer_Count(binary.UINT_8_SIZE),
			md5.Buffer_Count(binary.UINT_64_SIZE),
		} {
			buffered := digest
			buffered.Buffer_Count = buffer_count
			buffered.Message_Size = md5.Message_Size(buffer_count)
			md5.Digest_Write(&buffered, md5.Source{byte(value)})
		}
	}

	var uninitialized md5.Digest
	testify.Panics(t, func() { md5.Digest_Reset(&uninitialized) })
	testify.Panics(t, func() { md5.Digest_Sum_Into(&uninitialized, nil) })
	testify.Panics(t, func() { md5.Digest_Size(&uninitialized) })
	testify.Panics(t, func() { md5.Digest_Block_Size(&uninitialized) })
	initialized := domain_digest(bits.WORD_64_MINIMUM)
	md5.Digest_Clone_Into(&uninitialized, &initialized)
	var invalid_source md5.Digest
	testify.Panics(t, func() { md5.Digest_Clone_Into(&initialized, &invalid_source) })
}

func domain_digest(value uint64) (digest md5.Digest) {
	digest.State = md5.State{
		Lane_0: md5.State_Lane_0(value),
		Lane_1: md5.State_Lane_1(value),
		Lane_2: md5.State_Lane_2(value),
		Lane_3: md5.State_Lane_3(value),
	}
	digest.Buffer = md5.Buffer{
		Lane_1: md5.Buffer_Lane_1(value),
		Lane_2: md5.Buffer_Lane_2(value),
		Lane_3: md5.Buffer_Lane_3(value),
		Lane_4: md5.Buffer_Lane_4(value),
		Lane_5: md5.Buffer_Lane_5(value),
		Lane_6: md5.Buffer_Lane_6(value),
		Lane_7: md5.Buffer_Lane_7(value),
		Lane_8: md5.Buffer_Lane_8(value),
	}
	digest.Ready = true
	return digest
}

func drive_digest_operations(digest *md5.Digest) {
	var destination [md5.DIGEST_SIZE]byte
	md5.Digest_Write(digest, nil)
	md5.Digest_Sum_Into(digest, destination[:])
	md5.Digest_Size(digest)
	md5.Digest_Block_Size(digest)
	clone_source := *digest
	clone_destination := *digest
	md5.Digest_Clone_Into(&clone_destination, &clone_source)
	reset := *digest
	md5.Digest_Reset(&reset)
	initialized := *digest
	md5.Digest_Init(&initialized)
}
