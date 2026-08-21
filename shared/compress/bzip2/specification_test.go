package bzip2_test

import (
	"testing"

	"local/james-orcales/shared/compress/bzip2"
	"local/james-orcales/shared/testify"
)

// Test_Bounded_Decompression protects caller-owned output and workspace bounds.
func Test_Bounded_Decompression(t *testing.T) {
	compressed := test_bzip2_compressed()
	var destination [TEST_BZIP2_OUTPUT_SIZE]byte
	var transform [TEST_BZIP2_BLOCK_SIZE]uint32
	count, status := bzip2.Decode_Into(
		destination[:], transform[:], compressed[:],
	)
	if status != bzip2.STATUS_OK {
		t.Fatalf("Decode_Into status = %d, want STATUS_OK", status)
	}
	if int(count) != len(destination) {
		t.Fatalf("Decode_Into count = %d, want %d", count, len(destination))
	}
	if string(destination[:]) != "hello bounded bzip2 world\n" {
		t.Fatalf("Decode_Into output = %q", destination[:])
	}

	var short [TEST_BZIP2_SHORT_OUTPUT_SIZE]byte
	count, status = bzip2.Decode_Into(short[:], transform[:], compressed[:])
	if status != bzip2.STATUS_OUTPUT_TOO_SMALL {
		t.Fatalf("short output status = %d", status)
	}
	if int(count) != len(short) {
		t.Fatalf("short output count = %d", count)
	}

	count, status = bzip2.Decode_Into(destination[:], transform[:1], compressed[:])
	if status != bzip2.STATUS_WORKSPACE_TOO_SMALL {
		t.Fatalf("short workspace status = %d", status)
	}
	if count != 0 {
		t.Fatalf("short workspace count = %d", count)
	}
	runs := test_bzip2_runs()
	assert_bzip2_runs(t, runs[:], transform[:])
}

// Test_Allocation proves every result path owns no heap storage.
func Test_Allocation(t *testing.T) {
	compressed := test_bzip2_compressed()
	runs := test_bzip2_runs()
	var destination [TEST_BZIP2_OUTPUT_SIZE]byte
	var runs_destination [TEST_BZIP2_RUNS_OUTPUT_SIZE]byte
	var transform [TEST_BZIP2_BLOCK_SIZE]uint32
	var observed_count bzip2.Count
	var observed_status bzip2.Status
	testify.Zero_Allocation(t, func() {
		observed_count, observed_status = bzip2.Decode_Into(
			destination[:], transform[:], compressed[:],
		)
	})
	testify.Zero_Allocation(t, func() {
		observed_count, observed_status = bzip2.Decode_Into(
			destination[:TEST_BZIP2_SHORT_OUTPUT_SIZE], transform[:], compressed[:],
		)
	})
	testify.Zero_Allocation(t, func() {
		observed_count, observed_status = bzip2.Decode_Into(
			destination[:], transform[:1], compressed[:],
		)
	})
	testify.Zero_Allocation(t, func() {
		observed_count, observed_status = bzip2.Decode_Into(
			destination[:], transform[:], nil,
		)
	})
	testify.Zero_Allocation(t, func() {
		observed_count, observed_status = bzip2.Decode_Into(
			runs_destination[:], transform[:], runs[:],
		)
	})
	if observed_count == -1 {
		t.Fatal("allocation calls produced impossible count")
	}
	if observed_status > bzip2.STATUS_WORKSPACE_TOO_SMALL {
		t.Fatal("allocation calls produced impossible observation")
	}
}

// Test_Untrusted_Input protects malformed-input boundary.
func Test_Untrusted_Input(t *testing.T) {
	compressed_input := test_bzip2_compressed()
	var destination [TEST_BZIP2_OUTPUT_SIZE]byte
	var transform [TEST_BZIP2_BLOCK_SIZE]uint32
	suffix_input := [TEST_BZIP2_COMPRESSED_SIZE + 1]byte{}
	copy(suffix_input[:], compressed_input[:])
	inputs := [][]byte{
		nil,
		{0},
		compressed_input[:len(compressed_input)-1],
		suffix_input[:],
	}
	for _, compressed := range inputs {
		_, status := bzip2.Decode_Into(destination[:], transform[:], compressed)
		if status != bzip2.STATUS_INPUT_INVALID {
			t.Fatalf("malformed status = %d, want STATUS_INPUT_INVALID", status)
		}
	}
	for index := 0; index < len(compressed_input); index++ {
		_, status := bzip2.Decode_Into(
			destination[:], transform[:], compressed_input[:index],
		)
		if status != bzip2.STATUS_INPUT_INVALID {
			t.Fatalf("truncated size %d status = %d", index, status)
		}
	}
	checksum_input := compressed_input
	checksum_input[TEST_BZIP2_BLOCK_CHECKSUM_INDEX] ^= 1
	_, status := bzip2.Decode_Into(destination[:], transform[:], checksum_input[:])
	if status != bzip2.STATUS_INPUT_INVALID {
		t.Fatalf("checksum status = %d", status)
	}
}

// Test_Invariant_Domains drives every public bound through decoder input.
func Test_Invariant_Domains(t *testing.T) {
	compressed := test_bzip2_compressed()
	maximum_destination := make([]byte, bzip2.BYTE_COUNT_MAXIMUM)
	maximum_transform := make([]uint32, TEST_BZIP2_BLOCK_SIZE)
	for _, size := range []int{0, 1, 2, bzip2.BYTE_COUNT_MAXIMUM} {
		count, status := bzip2.Decode_Into(
			maximum_destination[:size], maximum_transform, compressed[:],
		)
		assert_bzip2_domain_result(t, count, status)
	}
	for _, size := range []int{0, 1, 2, TEST_BZIP2_BLOCK_SIZE} {
		count, status := bzip2.Decode_Into(
			maximum_destination, maximum_transform[:size], compressed[:],
		)
		assert_bzip2_domain_result(t, count, status)
	}

	maximum_compressed := make([]byte, bzip2.BYTE_COUNT_MAXIMUM)
	empty := test_bzip2_empty()
	one := test_bzip2_one()
	for _, source := range [][]byte{empty[:], one[:], compressed[:]} {
		clear(maximum_compressed)
		copy(maximum_compressed, source)
		count, status := bzip2.Decode_Into(
			maximum_destination, maximum_transform, maximum_compressed,
		)
		assert_bzip2_domain_result(t, count, status)
	}

	level_one := compressed
	level_one[3] = '1'
	count, status := bzip2.Decode_Into(
		maximum_destination, maximum_transform[:100_000], level_one[:],
	)
	if status != bzip2.STATUS_OK {
		t.Fatalf("level one status = %d", status)
	}
	if int(count) != TEST_BZIP2_OUTPUT_SIZE {
		t.Fatalf("level one decode = (%d, %d)", count, status)
	}

	assert_bzip2_size(t, maximum_destination, maximum_transform, empty[:], 0)
	assert_bzip2_size(t, maximum_destination, maximum_transform, one[:], 1)
	two := test_bzip2_two()
	assert_bzip2_size(t, maximum_destination, maximum_transform, two[:], 2)
	bytes := test_bzip2_bytes()
	assert_bzip2_size(t, maximum_destination, maximum_transform, bytes[:], 4)
	maximum := test_bzip2_maximum()
	assert_bzip2_size(
		t, maximum_destination, maximum_transform, maximum[:],
		bzip2.BYTE_COUNT_MAXIMUM,
	)
}

const TEST_BZIP2_BLOCK_SIZE = 900_000
const TEST_BZIP2_OUTPUT_SIZE = 26
const TEST_BZIP2_SHORT_OUTPUT_SIZE = 5
const TEST_BZIP2_COMPRESSED_SIZE = 65
const TEST_BZIP2_RUNS_SIZE = 58
const TEST_BZIP2_RUNS_OUTPUT_SIZE = 5_000
const TEST_BZIP2_RUN_START_INDEX = 1_000
const TEST_BZIP2_BLOCK_CHECKSUM_INDEX = 10
const TEST_BZIP2_EMPTY_SIZE = 14
const TEST_BZIP2_ONE_SIZE = 37
const TEST_BZIP2_TWO_SIZE = 37
const TEST_BZIP2_BYTES_SIZE = 42
const TEST_BZIP2_MAXIMUM_SIZE = 79

func test_bzip2_compressed() (compressed [TEST_BZIP2_COMPRESSED_SIZE]byte) {
	return [TEST_BZIP2_COMPRESSED_SIZE]byte{
		66, 90, 104, 57, 49, 65, 89, 38, 83, 89, 172, 2, 115, 97, 0, 0,
		6, 89, 128, 0, 16, 64, 0, 16, 0, 22, 101, 210, 144, 32, 0, 34,
		38, 141, 52, 61, 6, 161, 76, 0, 19, 70, 10, 209, 180, 29, 135, 30,
		52, 252, 181, 54, 74, 86, 190, 46, 228, 138, 112, 161, 33, 88, 4, 230,
		194,
	}
}

func test_bzip2_runs() (compressed [TEST_BZIP2_RUNS_SIZE]byte) {
	return [TEST_BZIP2_RUNS_SIZE]byte{
		66, 90, 104, 57, 49, 65, 89, 38, 83, 89, 150, 112, 152, 11, 0, 0,
		1, 132, 1, 190, 0, 0, 128, 0, 8, 32, 0, 84, 67, 0, 38, 170,
		140, 201, 172, 254, 159, 10, 13, 237, 72, 50, 164, 27, 82, 12, 187,
		11, 185, 34, 156, 40, 72, 75, 56, 76, 5, 128,
	}
}

func test_bzip2_empty() (compressed [TEST_BZIP2_EMPTY_SIZE]byte) {
	return [TEST_BZIP2_EMPTY_SIZE]byte{
		66, 90, 104, 57, 23, 114, 69, 56, 80, 144, 0, 0, 0, 0,
	}
}

func test_bzip2_one() (compressed [TEST_BZIP2_ONE_SIZE]byte) {
	return [TEST_BZIP2_ONE_SIZE]byte{
		66, 90, 104, 57, 49, 65, 89, 38, 83, 89, 177, 247, 64, 75, 0, 0,
		0, 64, 0, 64, 0, 32, 0, 33, 24, 70, 130, 238, 72, 167, 10, 18,
		22, 62, 232, 9, 96,
	}
}

func test_bzip2_two() (compressed [TEST_BZIP2_TWO_SIZE]byte) {
	return [TEST_BZIP2_TWO_SIZE]byte{
		66, 90, 104, 57, 49, 65, 89, 38, 83, 89, 255, 72, 155, 130, 0, 0,
		0, 192, 0, 64, 0, 32, 0, 33, 24, 70, 194, 238, 72, 167, 10, 18,
		31, 233, 19, 112, 64,
	}
}

func test_bzip2_bytes() (compressed [TEST_BZIP2_BYTES_SIZE]byte) {
	return [TEST_BZIP2_BYTES_SIZE]byte{
		66, 90, 104, 57, 49, 65, 89, 38, 83, 89, 40, 38, 80, 184, 0, 0,
		0, 64, 0, 240, 0, 0, 0, 160, 0, 33, 154, 104, 51, 77, 19, 60,
		93, 201, 20, 225, 66, 64, 160, 153, 66, 224,
	}
}

func test_bzip2_maximum() (compressed [TEST_BZIP2_MAXIMUM_SIZE]byte) {
	return [TEST_BZIP2_MAXIMUM_SIZE]byte{
		66, 90, 104, 57, 49, 65, 89, 38, 83, 89, 14, 9, 226, 223, 1, 95,
		142, 64, 0, 192, 0, 0, 8, 32, 0, 48, 128, 77, 70, 66, 160, 37,
		169, 10, 128, 151, 49, 65, 89, 38, 83, 89, 188, 4, 181, 195, 0, 162,
		117, 192, 0, 192, 0, 0, 8, 32, 0, 32, 164, 8, 54, 50, 138, 136,
		77, 42, 42, 33, 56, 187, 146, 41, 194, 132, 133, 0, 187, 131, 232,
	}
}

func assert_bzip2_runs(t *testing.T, compressed []byte, transform []uint32) {
	t.Helper()
	var destination [TEST_BZIP2_RUNS_OUTPUT_SIZE]byte
	count, status := bzip2.Decode_Into(destination[:], transform, compressed)
	if status != bzip2.STATUS_OK {
		t.Fatalf("runs status = %d", status)
	}
	if int(count) != len(destination) {
		t.Fatalf("runs count = %d", count)
	}
	for index := 0; index < TEST_BZIP2_RUN_START_INDEX; index++ {
		if destination[index] != 'A' {
			t.Fatalf("runs prefix byte %d = %d", index, destination[index])
		}
	}
	const CYCLE = "BCDE"
	for index := TEST_BZIP2_RUN_START_INDEX; index < len(destination); index++ {
		want := CYCLE[(index-TEST_BZIP2_RUN_START_INDEX)%len(CYCLE)]
		if destination[index] != want {
			t.Fatalf("runs byte %d = %d, want %d", index, destination[index], want)
		}
	}
}

func assert_bzip2_size(
	t *testing.T, destination []byte, transform []uint32,
	compressed []byte, want int,
) {
	t.Helper()
	count, status := bzip2.Decode_Into(destination, transform, compressed)
	if status != bzip2.STATUS_OK {
		t.Fatalf("domain status = %d, want STATUS_OK", status)
	}
	if int(count) != want {
		t.Fatalf("domain decode = (%d, %d), want (%d, STATUS_OK)", count, status, want)
	}
}

func assert_bzip2_domain_result(t *testing.T, count bzip2.Count, status bzip2.Status) {
	t.Helper()
	if count < 0 {
		t.Fatalf("domain count = %d", count)
	}
	if int(count) > bzip2.BYTE_COUNT_MAXIMUM {
		t.Fatalf("domain count = %d", count)
	}
	if status > bzip2.STATUS_WORKSPACE_TOO_SMALL {
		t.Fatalf("domain status = %d", status)
	}
}
