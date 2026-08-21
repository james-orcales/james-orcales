package sha1_test

import (
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
	testify.Equal(t, digest_sum(&source), digest_sum(&destination))
}

// Test_Formula binds every derived width to shared primitive geometry.
func Test_Formula(t *testing.T) {
	testify.Equal(t, sha1.STATE_LANE_COUNT*bits.BIT_COUNT_32_MAXIMUM, sha1.DIGEST_BIT_COUNT)
	testify.Equal(t, sha1.DIGEST_BIT_COUNT/binary.BITS_PER_BYTE, sha1.DIGEST_SIZE)
	testify.Equal(t, sha1.MESSAGE_WORD_COUNT*binary.UINT_32_SIZE, sha1.BLOCK_SIZE)
	testify.Equal(t, sha1.BLOCK_SIZE*binary.BITS_PER_BYTE, sha1.BLOCK_BIT_COUNT)
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
	tests := [...]struct {
		Source sha1.Source
		Want   test_output
	}{
		{Source: sha1.Source(""), Want: test_output(REFERENCE_EMPTY)},
		{Source: sha1.Source("abc"), Want: test_output(REFERENCE_ABC)},
	}
	for _, test := range tests {
		testify.Equal(t, test.Want, checksum(test.Source))
		var digest sha1.Digest
		sha1.Digest_Init(&digest)
		midpoint := len(test.Source) / TEST_COUNT_TWO
		testify.Equal(
			t, sha1.Count(midpoint), sha1.Digest_Write(&digest, test.Source[:midpoint]),
		)
		testify.Equal(
			t, sha1.Count(len(test.Source)-midpoint),
			sha1.Digest_Write(&digest, test.Source[midpoint:]),
		)
		testify.Equal(t, test.Want, digest_sum(&digest))
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
		want := checksum(source[:size])
		var digest sha1.Digest
		sha1.Digest_Init(&digest)
		sha1.Digest_Write(&digest, source[:size])
		testify.Equal(t, want, digest_sum(&digest))
	}
	var reference sha1.Digest
	sha1.Digest_Init(&reference)
	write_chunks(&reference, source[:], sha1.BLOCK_SIZE+TEST_COUNT_ONE)
	want := digest_sum(&reference)
	var digest sha1.Digest
	sha1.Digest_Init(&digest)
	sha1.Digest_Write(&digest, source[:sha1.SOURCE_SIZE_MAXIMUM])
	sha1.Digest_Write(&digest, source[sha1.SOURCE_SIZE_MAXIMUM:])
	testify.Equal(t, want, digest_sum(&digest))
	sha1.Digest_Write(&digest, sha1.Source("x"))
	sha1.Digest_Write(&reference, sha1.Source("x"))
	testify.Equal(t, digest_sum(&reference), digest_sum(&digest))
	want_chunks := checksum(source[:sha1.SOURCE_SIZE_MAXIMUM])
	for _, chunk_size := range [...]int{
		TEST_COUNT_ONE, sha1.BLOCK_SIZE - TEST_COUNT_ONE, sha1.BLOCK_SIZE,
		sha1.BLOCK_SIZE + TEST_COUNT_ONE, TEST_MULTI_BLOCK_CHUNK_SIZE,
	} {
		sha1.Digest_Init(&digest)
		write_chunks(&digest, source[:sha1.SOURCE_SIZE_MAXIMUM], chunk_size)
		testify.Equal(t, want_chunks, digest_sum(&digest), chunk_size)
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
	testify.Equal(t, test_output(REFERENCE_ABC), test_output(output[:]))
	testify.Equal(t, before, digest)
	sha1.Digest_Reset(&digest)
	testify.Equal(t, checksum(nil), digest_sum(&digest))
}

// Test_Bounds rejects oversized calls and uninitialized caller state.
func Test_Bounds(t *testing.T) {
	var digest sha1.Digest
	testify.Panics(t, func() { sha1.Digest_Write(&digest, nil) })
	sha1.Digest_Init(&digest)
	var source [sha1.SOURCE_SIZE_MAXIMUM + TEST_COUNT_ONE]byte
	testify.Panics(t, func() { sha1.Digest_Write(&digest, source[:]) })
	testify.Panics(t, func() { sha1.Checksum_Into(nil, source[:]) })
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
		var checksum_output [sha1.DIGEST_SIZE]byte
		sha1.Checksum_Into(checksum_output[:], source[:size])
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
		sha1.Checksum_Into(destination[:size], nil)
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
	test_state_invariant_domains(t)
}

// Test_Allocation proves every exported runtime operation remains stack-owned.
func Test_Allocation(t *testing.T) {
	fixture := allocation_fixture{
		Source: sha1.Source("abc"), Output: make(sha1.Destination, sha1.DIGEST_SIZE),
	}
	sha1.Digest_Init(&fixture.Digest)
	testify.Zero_Allocation(t, func() { sha1.Digest_Init(&fixture.Digest) })
	testify.Zero_Allocation(t, func() {
		fixture.Count = sha1.Digest_Write(&fixture.Digest, fixture.Source)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Output_Count, fixture.Output_Status = sha1.Digest_Sum_Into(
			&fixture.Digest, fixture.Output,
		)
	})
	testify.Zero_Allocation(t, func() { sha1.Digest_Reset(&fixture.Digest) })
	testify.Zero_Allocation(t, func() {
		sha1.Digest_Clone_Into(&fixture.Clone, &fixture.Digest)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Size = sha1.Digest_Size(&fixture.Digest)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Block_Size = sha1.Digest_Block_Size(&fixture.Digest)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Output_Count, fixture.Output_Status = sha1.Checksum_Into(
			fixture.Output, fixture.Source,
		)
	})
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

const REFERENCE_EMPTY = "\xda\x39\xa3\xee\x5e\x6b\x4b\x0d\x32\x55" +
	"\xbf\xef\x95\x60\x18\x90\xaf\xd8\x07\x09"

const REFERENCE_ABC = "\xa9\x99\x3e\x36\x47\x06\x81\x6a\xba\x3e" +
	"\x25\x71\x78\x50\xc2\x6c\x9c\xd0\xd8\x9d"

type allocation_fixture struct {
	Digest        sha1.Digest
	Clone         sha1.Digest
	Source        sha1.Source
	Output        sha1.Destination
	Count         sha1.Count
	Output_Count  sha1.Output_Count
	Output_Status sha1.Output_Status
	Size          sha1.Size
	Block_Size    sha1.Block_Size
}

type test_output []byte

func checksum(source sha1.Source) (output test_output) {
	output = make(test_output, sha1.DIGEST_SIZE)
	sha1.Checksum_Into(sha1.Destination(output), source)
	return output
}

func digest_sum(digest *sha1.Digest) (output test_output) {
	output = make(test_output, sha1.DIGEST_SIZE)
	sha1.Digest_Sum_Into(digest, sha1.Destination(output))
	return output
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
		for _, buffer_count := range [...]sha1.Buffer_Count{
			sha1.Buffer_Count(binary.UINT_8_SIZE),
			sha1.Buffer_Count(binary.UINT_64_SIZE),
		} {
			buffered := digest
			buffered.Buffer_Count = buffer_count
			buffered.Message_Size = sha1.Message_Size(buffer_count)
			sha1.Digest_Write(&buffered, sha1.Source{byte(value)})
		}
	}

	var uninitialized sha1.Digest
	testify.Panics(t, func() { sha1.Digest_Reset(&uninitialized) })
	testify.Panics(t, func() { sha1.Digest_Sum_Into(&uninitialized, nil) })
	testify.Panics(t, func() { sha1.Digest_Size(&uninitialized) })
	testify.Panics(t, func() { sha1.Digest_Block_Size(&uninitialized) })
	initialized := domain_digest(bits.WORD_64_MINIMUM)
	sha1.Digest_Clone_Into(&uninitialized, &initialized)
	var invalid_source sha1.Digest
	testify.Panics(t, func() { sha1.Digest_Clone_Into(&initialized, &invalid_source) })
}

func domain_digest(value uint64) (digest sha1.Digest) {
	digest.State = sha1.State{
		Lane_0: sha1.State_Lane_0(value),
		Lane_1: sha1.State_Lane_1(value),
		Lane_2: sha1.State_Lane_2(value),
		Lane_3: sha1.State_Lane_3(value),
		Lane_4: sha1.State_Lane_4(value),
	}
	digest.Buffer = sha1.Buffer{
		Lane_1: sha1.Buffer_Lane_1(value),
		Lane_2: sha1.Buffer_Lane_2(value),
		Lane_3: sha1.Buffer_Lane_3(value),
		Lane_4: sha1.Buffer_Lane_4(value),
		Lane_5: sha1.Buffer_Lane_5(value),
		Lane_6: sha1.Buffer_Lane_6(value),
		Lane_7: sha1.Buffer_Lane_7(value),
		Lane_8: sha1.Buffer_Lane_8(value),
	}
	digest.Ready = true
	return digest
}

func drive_digest_operations(digest *sha1.Digest) {
	var destination [sha1.DIGEST_SIZE]byte
	sha1.Digest_Write(digest, nil)
	sha1.Digest_Sum_Into(digest, destination[:])
	sha1.Digest_Size(digest)
	sha1.Digest_Block_Size(digest)
	clone_source := *digest
	clone_destination := *digest
	sha1.Digest_Clone_Into(&clone_destination, &clone_source)
	reset := *digest
	sha1.Digest_Reset(&reset)
	initialized := *digest
	sha1.Digest_Init(&initialized)
}
