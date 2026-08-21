package zlib_test

import (
	"testing"

	"local/james-orcales/shared/compress/zlib"
	"local/james-orcales/shared/testify"
)

// Test_Bounded_Decompression protects caller-owned output bound.
func Test_Bounded_Decompression(t *testing.T) {
	compressed := test_zlib_compressed()
	var destination [TEST_ZLIB_OUTPUT_SIZE]byte
	count, status := zlib.Decode_Into(destination[:], compressed[:])
	if status != zlib.STATUS_OK {
		t.Fatalf("Decode_Into status = %d, want STATUS_OK", status)
	}
	if int(count) != len(destination) {
		t.Fatalf("Decode_Into count = %d, want %d", count, len(destination))
	}
	if string(destination[:]) != "hello bounded zlib world\n" {
		t.Fatalf("Decode_Into output = %q", destination[:])
	}

	var short [TEST_ZLIB_SHORT_OUTPUT_SIZE]byte
	count, status = zlib.Decode_Into(short[:], compressed[:])
	if status != zlib.STATUS_OUTPUT_TOO_SMALL {
		t.Fatalf("short output status = %d", status)
	}
	if int(count) != len(short) {
		t.Fatalf("short output count = %d", count)
	}

	stored := test_zlib_stored()
	assert_zlib_decode(t, stored[:], "stored block payload")
	fixed := test_zlib_fixed()
	assert_zlib_decode(t, fixed[:], "fixed huffman payload fixed huffman payload")
	dynamic := test_zlib_dynamic()
	assert_zlib_decode(t, dynamic[:], "aaaabbbbccccddddeeeeffffgggghhhh")
	empty := test_zlib_empty()
	assert_zlib_decode(t, empty[:], "")
}

// Test_Allocation proves every result path owns no heap storage.
func Test_Allocation(t *testing.T) {
	compressed := test_zlib_compressed()
	stored := test_zlib_stored()
	fixed := test_zlib_fixed()
	dynamic := test_zlib_dynamic()
	var destination [TEST_ZLIB_VARIANT_OUTPUT_SIZE]byte
	var observed_count zlib.Count
	var observed_status zlib.Status
	testify.Zero_Allocation(t, func() {
		observed_count, observed_status = zlib.Decode_Into(
			destination[:], compressed[:],
		)
	})
	testify.Zero_Allocation(t, func() {
		observed_count, observed_status = zlib.Decode_Into(
			destination[:TEST_ZLIB_SHORT_OUTPUT_SIZE], compressed[:],
		)
	})
	testify.Zero_Allocation(t, func() {
		observed_count, observed_status = zlib.Decode_Into(destination[:], nil)
	})
	testify.Zero_Allocation(t, func() {
		observed_count, observed_status = zlib.Decode_Into(destination[:], stored[:])
	})
	testify.Zero_Allocation(t, func() {
		observed_count, observed_status = zlib.Decode_Into(destination[:], fixed[:])
	})
	testify.Zero_Allocation(t, func() {
		observed_count, observed_status = zlib.Decode_Into(destination[:], dynamic[:])
	})
	if observed_count == -1 {
		t.Fatal("allocation calls produced impossible count")
	}
	if observed_status > zlib.STATUS_OUTPUT_TOO_SMALL {
		t.Fatal("allocation calls produced impossible observation")
	}
}

// Test_Untrusted_Input protects malformed-input boundary.
func Test_Untrusted_Input(t *testing.T) {
	compressed_input := test_zlib_compressed()
	var destination [TEST_ZLIB_OUTPUT_SIZE]byte
	suffix_input := [TEST_ZLIB_COMPRESSED_SIZE + 1]byte{}
	copy(suffix_input[:], compressed_input[:])
	inputs := [][]byte{
		nil,
		{0},
		compressed_input[:len(compressed_input)-1],
		suffix_input[:],
	}
	for _, compressed := range inputs {
		_, status := zlib.Decode_Into(destination[:], compressed)
		if status != zlib.STATUS_INPUT_INVALID {
			t.Fatalf("malformed status = %d, want STATUS_INPUT_INVALID", status)
		}
	}
	for index := 0; index < len(compressed_input); index++ {
		_, status := zlib.Decode_Into(destination[:], compressed_input[:index])
		if status != zlib.STATUS_INPUT_INVALID {
			t.Fatalf("truncated size %d status = %d", index, status)
		}
	}
	checksum_input := compressed_input
	checksum_input[len(checksum_input)-1] ^= 1
	_, status := zlib.Decode_Into(destination[:], checksum_input[:])
	if status != zlib.STATUS_INPUT_INVALID {
		t.Fatalf("checksum status = %d", status)
	}
	stored_variant := test_zlib_stored()
	fixed_variant := test_zlib_fixed()
	dynamic_variant := test_zlib_dynamic()
	variants := [][]byte{stored_variant[:], fixed_variant[:], dynamic_variant[:]}
	for _, original := range variants {
		candidate := make([]byte, len(original))
		for index := range original {
			for value_index := 0; value_index < 256; value_index++ {
				copy(candidate, original)
				candidate[index] = byte(value_index)
				observed_count, observed_status := zlib.Decode_Into(
					destination[:], candidate,
				)
				assert_zlib_domain_result(t, observed_count, observed_status)
			}
		}
	}
}

// Test_Invariant_Domains drives every bounded decoder position through production input.
func Test_Invariant_Domains(t *testing.T) {
	stored := test_zlib_stored()
	fixed := test_zlib_fixed()
	dynamic := test_zlib_dynamic()
	fixtures := [][]byte{stored[:], fixed[:], dynamic[:]}
	maximum_destination := make([]byte, zlib.BYTE_COUNT_MAXIMUM)
	for _, compressed := range fixtures {
		for _, size := range []int{0, 1, 2, zlib.BYTE_COUNT_MAXIMUM} {
			count, status := zlib.Decode_Into(maximum_destination[:size], compressed)
			assert_zlib_domain_result(t, count, status)
		}
	}

	maximum_compressed := make([]byte, zlib.BYTE_COUNT_MAXIMUM)
	for _, compressed := range fixtures {
		clear(maximum_compressed)
		copy(maximum_compressed, compressed)
		count, status := zlib.Decode_Into(maximum_destination, maximum_compressed)
		assert_zlib_domain_result(t, count, status)
	}

	for _, size := range []int{1, 2} {
		compressed := test_zlib_fixed_zeros(size)
		count, status := zlib.Decode_Into(maximum_destination, compressed)
		if status != zlib.STATUS_OK {
			t.Fatalf("small domain status = %d, want STATUS_OK", status)
		}
		if int(count) != size {
			t.Fatalf("small domain decode = (%d, %d), want (%d, STATUS_OK)",
				count, status, size)
		}
	}

	stored_maximum := test_zlib_stored_zeros(1<<16 - 1)
	count, status := zlib.Decode_Into(maximum_destination, stored_maximum)
	if status != zlib.STATUS_OK {
		t.Fatalf("maximum stored status = %d", status)
	}
	if int(count) != 1<<16-1 {
		t.Fatalf("maximum stored decode = (%d, %d)", count, status)
	}

	huffman_maximum := test_zlib_fixed_zeros(zlib.BYTE_COUNT_MAXIMUM)
	count, status = zlib.Decode_Into(maximum_destination, huffman_maximum)
	if status != zlib.STATUS_OK {
		t.Fatalf("maximum Huffman status = %d", status)
	}
	if int(count) != zlib.BYTE_COUNT_MAXIMUM {
		t.Fatalf("maximum Huffman decode = (%d, %d)", count, status)
	}

	for _, prefix := range [][]byte{{'a'}, {'a', 'b'}} {
		for _, suffix := range [][]byte{fixed[:], dynamic[:]} {
			compressed := test_zlib_prefix(prefix, suffix)
			observed_count, observed_status := zlib.Decode_Into(
				maximum_destination, compressed,
			)
			assert_zlib_domain_result(t, observed_count, observed_status)
		}
	}
	maximum_distance := test_zlib_maximum_distance()
	observed_count, observed_status := zlib.Decode_Into(
		maximum_destination, maximum_distance,
	)
	assert_zlib_domain_result(t, observed_count, observed_status)
}

// Test_Invariant_Reader_Domains drives malformed bit-reader boundary states.
func Test_Invariant_Reader_Domains(t *testing.T) {
	maximum_destination := make([]byte, zlib.BYTE_COUNT_MAXIMUM)
	stored_minimum := []byte{120, 1, 1, 0, 0, 0}
	observed_count, observed_status := zlib.Decode_Into(
		maximum_destination, stored_minimum,
	)
	assert_zlib_domain_result(t, observed_count, observed_status)
	for literal_index := 0; literal_index < 8; literal_index++ {
		stored := test_zlib_fixed_prefix(literal_index, 1)
		observed_count, observed_status = zlib.Decode_Into(maximum_destination, stored)
		assert_zlib_domain_result(t, observed_count, observed_status)
		dynamic := test_zlib_fixed_prefix(literal_index, 5)
		observed_count, observed_status = zlib.Decode_Into(maximum_destination, dynamic)
		assert_zlib_domain_result(t, observed_count, observed_status)
	}
	fixed := test_zlib_fixed_prefix(1, 3)
	observed_count, observed_status = zlib.Decode_Into(maximum_destination, fixed)
	assert_zlib_domain_result(t, observed_count, observed_status)

	for _, total_size := range []int{6, 7, 8, zlib.BYTE_COUNT_MAXIMUM} {
		compressed := test_zlib_dynamic_probe(total_size, 0, 0, 4, 0)
		probe_count, probe_status := zlib.Decode_Into(
			maximum_destination, compressed,
		)
		assert_zlib_domain_result(t, probe_count, probe_status)
	}
	for _, code_count := range []int{5, 7, 8} {
		tail_value := uint32(0)
		if code_count == 7 {
			tail_value = 2
		}
		if code_count == 8 {
			tail_value = 127
		}
		compressed := test_zlib_dynamic_probe(16, 0, 0, code_count, tail_value)
		probe_count, probe_status := zlib.Decode_Into(
			maximum_destination, compressed,
		)
		assert_zlib_domain_result(t, probe_count, probe_status)
	}
	maximum_dynamic_sizes := test_zlib_dynamic_probe(16, 29, 29, 8, 127)
	observed_count, observed_status = zlib.Decode_Into(
		maximum_destination, maximum_dynamic_sizes,
	)
	assert_zlib_domain_result(t, observed_count, observed_status)

	short_huffman := test_zlib_fixed_zeros(2)
	for compressed_index := 6; compressed_index < len(short_huffman); compressed_index++ {
		observed_count, observed_status = zlib.Decode_Into(
			maximum_destination, short_huffman[:compressed_index],
		)
		assert_zlib_domain_result(t, observed_count, observed_status)
	}
}

const TEST_ZLIB_OUTPUT_SIZE = 25
const TEST_ZLIB_SHORT_OUTPUT_SIZE = 5
const TEST_ZLIB_COMPRESSED_SIZE = 33
const TEST_ZLIB_STORED_SIZE = 31
const TEST_ZLIB_FIXED_SIZE = 33
const TEST_ZLIB_DYNAMIC_SIZE = 35
const TEST_ZLIB_EMPTY_SIZE = 8
const TEST_ZLIB_VARIANT_OUTPUT_SIZE = 64

func test_zlib_compressed() (compressed [TEST_ZLIB_COMPRESSED_SIZE]byte) {
	return [TEST_ZLIB_COMPRESSED_SIZE]byte{
		120, 156, 203, 72, 205, 201, 201, 87, 72, 202, 47, 205, 75, 73, 77,
		81, 168, 202, 201, 76, 82, 40, 207, 47, 202, 73, 225, 2, 0, 123, 233,
		9, 57,
	}
}

func test_zlib_stored() (compressed [TEST_ZLIB_STORED_SIZE]byte) {
	return [TEST_ZLIB_STORED_SIZE]byte{
		120, 1, 1, 20, 0, 235, 255, 115, 116, 111, 114, 101, 100, 32, 98, 108,
		111, 99, 107, 32, 112, 97, 121, 108, 111, 97, 100, 82, 62, 7, 199,
	}
}

func test_zlib_fixed() (compressed [TEST_ZLIB_FIXED_SIZE]byte) {
	return [TEST_ZLIB_FIXED_SIZE]byte{
		120, 1, 75, 203, 172, 72, 77, 81, 200, 40, 77, 75, 203, 77, 204, 83,
		40, 72, 172, 204, 201, 79, 76, 81, 72, 195, 38, 10, 0, 103, 86, 16,
		95,
	}
}

func test_zlib_dynamic() (compressed [TEST_ZLIB_DYNAMIC_SIZE]byte) {
	return [TEST_ZLIB_DYNAMIC_SIZE]byte{
		120, 1, 5, 193, 7, 1, 0, 0, 12, 2, 160, 172, 234, 60, 253, 19,
		12, 0, 128, 36, 37, 233, 238, 206, 182, 147, 164, 109, 183, 237, 1,
		204, 200, 12, 145,
	}
}

func test_zlib_empty() (compressed [TEST_ZLIB_EMPTY_SIZE]byte) {
	return [TEST_ZLIB_EMPTY_SIZE]byte{120, 156, 3, 0, 0, 0, 0, 1}
}

func assert_zlib_decode(t *testing.T, compressed []byte, want string) {
	t.Helper()
	var destination [TEST_ZLIB_VARIANT_OUTPUT_SIZE]byte
	count, status := zlib.Decode_Into(destination[:], compressed)
	if status != zlib.STATUS_OK {
		t.Fatalf("Decode_Into status = %d", status)
	}
	if string(destination[:count]) != want {
		t.Fatalf("Decode_Into output = %q", destination[:count])
	}
}

func test_zlib_prefix(prefix []byte, suffix []byte) (compressed []byte) {
	compressed = make([]byte, 0, len(prefix)+len(suffix)+5)
	compressed = append(compressed, 120, 1, 0, byte(len(prefix)), 0)
	compressed = append(compressed, byte(^uint16(len(prefix))), 255)
	compressed = append(compressed, prefix...)
	compressed = append(compressed, suffix[2:len(suffix)-4]...)
	decoded := make([]byte, 0, len(prefix)+TEST_ZLIB_VARIANT_OUTPUT_SIZE)
	decoded = append(decoded, prefix...)
	if len(suffix) == TEST_ZLIB_FIXED_SIZE {
		decoded = append(decoded, []byte("fixed huffman payload fixed huffman payload")...)
	} else {
		decoded = append(decoded, []byte("aaaabbbbccccddddeeeeffffgggghhhh")...)
	}
	checksum := test_zlib_adler32(decoded)
	return append(
		compressed, byte(checksum>>24), byte(checksum>>16),
		byte(checksum>>8), byte(checksum),
	)
}

func test_zlib_fixed_zeros(count int) (compressed []byte) {
	compressed = make([]byte, 0, count/100+16)
	compressed = append(compressed, 120, 1)
	bits := uint64(0)
	bit_count := uint(0)
	compressed, bits, bit_count = test_zlib_append_bits(
		compressed, bits, bit_count, 2, 3,
	)
	tail_count := count
	if tail_count > 0 {
		compressed, bits, bit_count = test_zlib_append_fixed_symbol(
			compressed, bits, bit_count, 0,
		)
		tail_count--
	}
	for tail_count >= 258 {
		compressed, bits, bit_count = test_zlib_append_fixed_symbol(
			compressed, bits, bit_count, 285,
		)
		compressed, bits, bit_count = test_zlib_append_bits(
			compressed, bits, bit_count, 0, 5,
		)
		tail_count -= 258
	}
	for tail_count > 0 {
		compressed, bits, bit_count = test_zlib_append_fixed_symbol(
			compressed, bits, bit_count, 0,
		)
		tail_count--
	}
	compressed, bits, bit_count = test_zlib_append_fixed_symbol(
		compressed, bits, bit_count, 256,
	)
	compressed, bits, bit_count = test_zlib_append_bits(
		compressed, bits, bit_count, 3, 3,
	)
	compressed, bits, bit_count = test_zlib_append_fixed_symbol(
		compressed, bits, bit_count, 256,
	)
	if bit_count > 0 {
		compressed = append(compressed, byte(bits))
	}
	checksum := uint32(count%zlib.ADLER_MODULUS)<<16 | 1
	return append(
		compressed, byte(checksum>>24), byte(checksum>>16),
		byte(checksum>>8), byte(checksum),
	)
}

func test_zlib_stored_zeros(size int) (compressed []byte) {
	encoded_size := uint16(size)
	inverse := ^encoded_size
	compressed = make([]byte, 0, size+11)
	compressed = append(
		compressed, 120, 1, 1, byte(encoded_size), byte(encoded_size>>8),
		byte(inverse), byte(inverse>>8),
	)
	compressed = append(compressed, make([]byte, size)...)
	checksum := uint32(size%zlib.ADLER_MODULUS)<<16 | 1
	return append(
		compressed, byte(checksum>>24), byte(checksum>>16),
		byte(checksum>>8), byte(checksum),
	)
}

func test_zlib_dynamic_probe(
	total_size int, literal_bits uint32, distance_bits uint32, code_count int,
	tail_value uint32,
) (compressed []byte) {
	var payload []byte
	bits := uint64(0)
	bit_count := uint(0)
	payload, bits, bit_count = test_zlib_append_bits(payload, bits, bit_count, 5, 3)
	payload, bits, bit_count = test_zlib_append_bits(
		payload, bits, bit_count, literal_bits, 5,
	)
	payload, bits, bit_count = test_zlib_append_bits(
		payload, bits, bit_count, distance_bits, 5,
	)
	payload, bits, bit_count = test_zlib_append_bits(
		payload, bits, bit_count, uint32(code_count-4), 4,
	)
	for index := 0; index < code_count; index++ {
		size := uint32(0)
		if index < 2 {
			size = 1
		}
		payload, bits, bit_count = test_zlib_append_bits(
			payload, bits, bit_count, size, 3,
		)
	}
	if bit_count > 0 {
		bits |= uint64(tail_value) << bit_count
		payload = append(payload, byte(bits))
	}
	minimum_size := len(payload) + 2
	if total_size < minimum_size {
		total_size = minimum_size
	}
	compressed = make([]byte, total_size)
	compressed[0] = 120
	compressed[1] = 1
	copy(compressed[2:], payload)
	return compressed
}

func test_zlib_maximum_distance() (compressed []byte) {
	compressed = append(compressed, 120, 1)
	bits := uint64(0)
	bit_count := uint(0)
	compressed, bits, bit_count = test_zlib_append_bits(
		compressed, bits, bit_count, 3, 3,
	)
	compressed, bits, bit_count = test_zlib_append_fixed_symbol(
		compressed, bits, bit_count, 257,
	)
	compressed, bits, bit_count = test_zlib_append_bits(
		compressed, bits, bit_count, test_zlib_reverse_bits(29, 5), 5,
	)
	compressed, bits, bit_count = test_zlib_append_bits(
		compressed, bits, bit_count, 1<<13-1, 13,
	)
	if bit_count > 0 {
		compressed = append(compressed, byte(bits))
	}
	for len(compressed) < 6 {
		compressed = append(compressed, 0)
	}
	return compressed
}

func test_zlib_fixed_prefix(literal_count int, block_kind uint32) (compressed []byte) {
	compressed = append(compressed, 120, 1)
	bits := uint64(0)
	bit_count := uint(0)
	compressed, bits, bit_count = test_zlib_append_bits(
		compressed, bits, bit_count, 2, 3,
	)
	for literal_index := 0; literal_index < literal_count; literal_index++ {
		compressed, bits, bit_count = test_zlib_append_fixed_symbol(
			compressed, bits, bit_count, 144,
		)
	}
	compressed, bits, bit_count = test_zlib_append_fixed_symbol(
		compressed, bits, bit_count, 256,
	)
	compressed, bits, bit_count = test_zlib_append_bits(
		compressed, bits, bit_count, block_kind, 3,
	)
	switch block_kind {
	case 1:
		padding_count := uint(8) - bit_count
		compressed, bits, bit_count = test_zlib_append_bits(
			compressed, bits, bit_count, 1<<padding_count-1, padding_count,
		)
		compressed = append(compressed, 0, 0, 255, 255)
	case 3:
		compressed, bits, bit_count = test_zlib_append_fixed_symbol(
			compressed, bits, bit_count, 256,
		)
	case 5:
		compressed, bits, bit_count = test_zlib_append_bits(
			compressed, bits, bit_count, 1<<14-1, 14,
		)
	}
	if bit_count > 0 {
		compressed = append(compressed, byte(bits))
	}
	return append(compressed, 0, 0, 0, 0, 0, 0, 0, 0)
}

func test_zlib_append_fixed_symbol(
	destination []byte, bits uint64, bit_count uint, symbol uint32,
) (next []byte, next_bits uint64, next_bit_count uint) {
	code := uint32(0)
	width := uint(0)
	switch {
	case symbol <= 143:
		code = 0x30 + symbol
		width = 8
	case symbol <= 255:
		code = 0x190 + symbol - 144
		width = 9
	case symbol <= 279:
		code = symbol - 256
		width = 7
	default:
		code = 0xc0 + symbol - 280
		width = 8
	}
	code = test_zlib_reverse_bits(code, width)
	return test_zlib_append_bits(destination, bits, bit_count, code, width)
}

func test_zlib_append_bits(
	destination []byte, bits uint64, bit_count uint, value uint32, count uint,
) (next []byte, next_bits uint64, next_bit_count uint) {
	bits |= uint64(value) << bit_count
	bit_count += count
	for bit_count >= 8 {
		destination = append(destination, byte(bits))
		bits >>= 8
		bit_count -= 8
	}
	return destination, bits, bit_count
}

func test_zlib_reverse_bits(value uint32, count uint) (reversed uint32) {
	for index := uint(0); index < count; index++ {
		reversed = reversed<<1 | value&1
		value >>= 1
	}
	return reversed
}

func test_zlib_adler32(value []byte) (checksum uint32) {
	first := uint32(1)
	second := uint32(0)
	for _, one := range value {
		first = (first + uint32(one)) % zlib.ADLER_MODULUS
		second = (second + first) % zlib.ADLER_MODULUS
	}
	return second<<16 | first
}

func assert_zlib_domain_result(t *testing.T, count zlib.Count, status zlib.Status) {
	t.Helper()
	if count < 0 {
		t.Fatalf("domain count = %d", count)
	}
	if status > zlib.STATUS_OUTPUT_TOO_SMALL {
		t.Fatalf("domain status = %d", status)
	}
}
