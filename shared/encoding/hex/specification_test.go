package hex_test

import (
	"testing"

	"local/james-orcales/shared/encoding/hex"
	"local/james-orcales/shared/testify"
)

// Test_Encode preserves lowercase hexadecimal with exact caller capacity.
func Test_Encode(t *testing.T) {
	cases := [...]encode_case{
		{Decoded: "", Encoded: ""},
		{Decoded: "\x00\x01", Encoded: "0001"},
		{Decoded: "\x00\x01\x02\x03", Encoded: "00010203"},
		{Decoded: "\x08\x09\x0a\x0f", Encoded: "08090a0f"},
		{Decoded: "\xf0\xf8\xff", Encoded: "f0f8ff"},
		{Decoded: "g", Encoded: "67"},
	}
	for _, one := range cases {
		assert_encode(t, one)
	}

	var destination [hex.ENCODED_BYTE_SIZE - 1]byte
	destination[0] = TEST_SENTINEL
	count, status := hex.Encode_Into(destination[:], []byte("g"))
	testify.Equal(t, hex.Encoded_Count(0), count)
	testify.Equal_Values(t, hex.STATUS_OUTPUT_TOO_SMALL, status)
	testify.Equal(t, TEST_SENTINEL, destination[0])
}

// Test_Decode preserves uppercase, lowercase, prefix, and malformed-input behavior.
func Test_Decode(t *testing.T) {
	cases := [...]decode_case{
		{Encoded: "", Decoded: "", Status: hex.STATUS_OK},
		{Encoded: "00010203", Decoded: "\x00\x01\x02\x03", Status: hex.STATUS_OK},
		{
			Encoded: "F8F9FAFBFCFDFEFF", Decoded: "\xf8\xf9\xfa\xfb\xfc\xfd\xfe\xff",
			Status: hex.STATUS_OK,
		},
		{Encoded: "0", Decoded: "", Status: hex.STATUS_INPUT_INCOMPLETE},
		{Encoded: "zd4aa", Decoded: "", Status: hex.STATUS_INPUT_INVALID},
		{Encoded: "d4aaz", Decoded: "\xd4\xaa", Status: hex.STATUS_INPUT_INVALID},
		{Encoded: "30313", Decoded: "01", Status: hex.STATUS_INPUT_INCOMPLETE},
		{Encoded: "00gg", Decoded: "\x00", Status: hex.STATUS_INPUT_INVALID},
	}
	for _, one := range cases {
		assert_decode(t, one)
	}

	var destination [hex.DECODED_SIZE_MAXIMUM]byte
	count, status := hex.Decode_Into(destination[:0], []byte("67"))
	testify.Equal(t, hex.Decoded_Count(0), count)
	testify.Equal_Values(t, hex.STATUS_OUTPUT_TOO_SMALL, status)
	count, status = hex.Decode_Into(destination[:2], []byte("0001"))
	testify.Equal(t, hex.Decoded_Count(2), count)
	testify.Equal_Values(t, hex.STATUS_OK, status)
}

// Test_Dump preserves hexdump -C line layout with exact caller storage.
func Test_Dump(t *testing.T) {
	var destination [hex.DUMP_SIZE_MAXIMUM]byte
	count, status := hex.Dump_Into(destination[:], nil)
	testify.Equal(t, hex.Dump_Count(0), count)
	testify.Equal_Values(t, hex.STATUS_OK, status)

	source := hex.Dump_Source("gopher")
	required := hex.Dump_Size(hex.Dump_Source_Count(len(source)))
	count, status = hex.Dump_Into(destination[:int(required)], source)
	testify.Equal(t, required, count)
	testify.Equal_Values(t, hex.STATUS_OK, status)
	expected := "00000000  67 6f 70 68 65 72                                 |gopher|\n"
	testify.Equal(t, expected, string(destination[:count]))
	test_dump_short_domains(t, destination[:])

	var dump_source [TEST_DUMP_SOURCE_SIZE]byte
	source_byte := byte(TEST_DUMP_SOURCE_BYTE_MINIMUM)
	for index := range dump_source {
		dump_source[index] = source_byte
		source_byte++
	}
	required = hex.Dump_Size(hex.Dump_Source_Count(len(dump_source)))
	count, status = hex.Dump_Into(destination[:int(required)], dump_source[:])
	testify.Equal(t, required, count)
	testify.Equal_Values(t, hex.STATUS_OK, status)
	testify.Equal(t, EXPECTED_DUMP, string(destination[:count]))
}

// Test_Bounds reaches every public collection boundary and refusal.
func Test_Bounds(t *testing.T) {
	test_size_boundaries(t)
	test_maximum_round_trip(t)
	test_maximum_dump(t)
	test_overlap(t)
	test_bound_refusals(t)
}

// Test_Allocation measures every public result path with the assertion tracker enabled.
func Test_Allocation(t *testing.T) {
	test_size_allocation(t)
	test_encode_allocation(t)
	test_decode_allocation(t)
	test_dump_allocation(t)
}

const TEST_SENTINEL byte = 0xa5

const TEST_DUMP_SOURCE_SIZE = hex.DUMP_SOURCE_GROUP_SIZE*hex.ENCODED_BYTE_SIZE +
	hex.DUMP_SOURCE_GROUP_SIZE/hex.ENCODED_BYTE_SIZE

const TEST_DUMP_SOURCE_BYTE_MINIMUM = hex.DUMP_SOURCE_GROUP_SIZE*hex.ENCODED_BYTE_SIZE -
	hex.ENCODED_BYTE_SIZE

const EXPECTED_DUMP = `00000000  1e 1f 20 21 22 23 24 25  26 27 28 29 2a 2b 2c 2d  |.. !"#$%&'()*+,-|
00000010  2e 2f 30 31 32 33 34 35  36 37 38 39 3a 3b 3c 3d  |./0123456789:;<=|
00000020  3e 3f 40 41 42 43 44 45                           |>?@ABCDE|
`

type encode_case struct {
	Decoded string
	Encoded string
}

type decode_case struct {
	Encoded string
	Decoded string
	Status  hex.Decode_Status
}

func assert_encode(t *testing.T, one encode_case) {
	t.Helper()
	var destination [hex.ENCODED_SIZE_MAXIMUM]byte
	required := hex.Encoded_Size(hex.Source_Count(len(one.Decoded)))
	testify.Equal(t, hex.Encoded_Count(len(one.Encoded)), required)
	count, status := hex.Encode_Into(destination[:int(required)], []byte(one.Decoded))
	testify.Equal(t, required, count)
	testify.Equal_Values(t, hex.STATUS_OK, status)
	testify.Equal(t, one.Encoded, string(destination[:count]))
}

func assert_decode(t *testing.T, one decode_case) {
	t.Helper()
	var destination [hex.DECODED_SIZE_MAXIMUM]byte
	count, status := hex.Decode_Into(destination[:], []byte(one.Encoded))
	testify.Equal(t, hex.Decoded_Count(len(one.Decoded)), count)
	testify.Equal(t, one.Status, status)
	testify.Equal(t, one.Decoded, string(destination[:count]))
}

func test_size_boundaries(t *testing.T) {
	testify.Equal(
		t, hex.Decoded_Count(0),
		hex.Decoded_Size_Maximum(hex.Encoded_Input_Count(0)),
	)
	testify.Equal(
		t, hex.Decoded_Count(0),
		hex.Decoded_Size_Maximum(hex.Encoded_Input_Count(1)),
	)
	testify.Equal(
		t, hex.Decoded_Count(1),
		hex.Decoded_Size_Maximum(hex.Encoded_Input_Count(hex.ENCODED_BYTE_SIZE)),
	)
	testify.Equal(
		t, hex.Decoded_Count(hex.ENCODED_BYTE_SIZE),
		hex.Decoded_Size_Maximum(
			hex.Encoded_Input_Count(hex.ENCODED_BYTE_SIZE*hex.ENCODED_BYTE_SIZE),
		),
	)
}

func test_dump_short_domains(t *testing.T, destination []byte) {
	var source [hex.ENCODED_BYTE_SIZE]byte
	for source_size := hex.SIZE_MINIMUM + 1; source_size <= len(source); source_size++ {
		required := hex.Dump_Size(hex.Dump_Source_Count(source_size))
		count, status := hex.Dump_Into(destination[:int(required)], source[:source_size])
		testify.Equal(t, required, count)
		testify.Equal_Values(t, hex.STATUS_OK, status)
		count, status = hex.Dump_Into(destination[:source_size], source[:source_size])
		testify.Equal(t, hex.Dump_Count(0), count)
		testify.Equal_Values(t, hex.STATUS_OUTPUT_TOO_SMALL, status)
	}
}

func test_maximum_round_trip(t *testing.T) {
	var source [hex.SOURCE_SIZE_MAXIMUM]byte
	var encoded [hex.ENCODED_SIZE_MAXIMUM]byte
	var decoded [hex.DECODED_SIZE_MAXIMUM]byte
	for index := range source {
		source[index] = byte(index)
	}
	encoded_count, encode_status := hex.Encode_Into(encoded[:], source[:])
	testify.Equal(t, hex.Encoded_Count(len(encoded)), encoded_count)
	testify.Equal_Values(t, hex.STATUS_OK, encode_status)
	decoded_size := hex.Decoded_Size_Maximum(hex.Encoded_Input_Count(len(encoded)))
	testify.Equal(t, hex.Decoded_Count(len(decoded)), decoded_size)
	decoded_count, decode_status := hex.Decode_Into(decoded[:], encoded[:])
	testify.Equal(t, hex.Decoded_Count(len(decoded)), decoded_count)
	testify.Equal_Values(t, hex.STATUS_OK, decode_status)
	testify.Equal(t, source[:], decoded[:])
}

func test_maximum_dump(t *testing.T) {
	var source [hex.DUMP_SOURCE_SIZE_MAXIMUM]byte
	var destination [hex.DUMP_SIZE_MAXIMUM]byte
	required := hex.Dump_Size(hex.Dump_Source_Count(len(source)))
	testify.Equal(t, hex.Dump_Count(len(destination)), required)
	count, status := hex.Dump_Into(destination[:], source[:])
	testify.Equal(t, required, count)
	testify.Equal_Values(t, hex.STATUS_OK, status)
}

func test_overlap(t *testing.T) {
	var encode_overlap [hex.ENCODED_BYTE_SIZE]byte
	_, encode_status := hex.Encode_Into(encode_overlap[:], encode_overlap[:1])
	testify.Equal_Values(t, hex.STATUS_STORAGE_INVALID, encode_status)
	var decode_overlap [hex.ENCODED_BYTE_SIZE]byte
	_, decode_status := hex.Decode_Into(decode_overlap[:1], decode_overlap[:])
	testify.Equal_Values(t, hex.STATUS_STORAGE_INVALID, decode_status)
	var dump_overlap [hex.DUMP_SIZE_MAXIMUM]byte
	_, dump_status := hex.Dump_Into(dump_overlap[:], dump_overlap[:1])
	testify.Equal_Values(t, hex.STATUS_STORAGE_INVALID, dump_status)
}

func test_bound_refusals(t *testing.T) {
	var encoded [hex.ENCODED_SIZE_MAXIMUM]byte
	var decoded [hex.DECODED_SIZE_MAXIMUM]byte
	var dump [hex.DUMP_SIZE_MAXIMUM]byte
	var source_oversized [hex.SOURCE_SIZE_MAXIMUM + 1]byte
	var encoded_oversized [hex.ENCODED_SIZE_MAXIMUM + 1]byte
	var decoded_oversized [hex.DECODED_SIZE_MAXIMUM + 1]byte
	var dump_source_oversized [hex.DUMP_SOURCE_SIZE_MAXIMUM + 1]byte
	testify.Panics(t, func() { hex.Encode_Into(encoded[:], source_oversized[:]) })
	testify.Panics(t, func() { hex.Encode_Into(encoded_oversized[:], nil) })
	testify.Panics(t, func() { hex.Decode_Into(decoded[:], encoded_oversized[:]) })
	testify.Panics(t, func() { hex.Decode_Into(decoded_oversized[:], nil) })
	testify.Panics(t, func() { hex.Dump_Into(dump[:], dump_source_oversized[:]) })
	testify.Panics(t, func() { hex.Dump_Into(encoded_oversized[:], nil) })
}

func test_size_allocation(t *testing.T) {
	var encoded_count hex.Encoded_Count
	var decoded_count hex.Decoded_Count
	var dump_count hex.Dump_Count
	testify.Zero_Allocation(t, func() {
		encoded_count = hex.Encoded_Size(hex.Source_Count(len("allocation")))
	})
	testify.Zero_Allocation(t, func() {
		decoded_count = hex.Decoded_Size_Maximum(
			hex.Encoded_Input_Count(len("616c6c6f636174696f6e")),
		)
	})
	testify.Zero_Allocation(t, func() {
		dump_count = hex.Dump_Size(hex.Dump_Source_Count(len("allocation")))
	})
	testify.True(t, encoded_count > 0)
	testify.True(t, decoded_count > 0)
	testify.True(t, dump_count > 0)
}

func test_encode_allocation(t *testing.T) {
	var destination [hex.ENCODED_SIZE_MAXIMUM]byte
	var overlap [hex.ENCODED_BYTE_SIZE]byte
	source := hex.Source("allocation")
	var count hex.Encoded_Count
	var status hex.Encode_Status
	testify.Zero_Allocation(t, func() {
		count, status = hex.Encode_Into(destination[:], source)
	})
	testify.Equal_Values(t, hex.STATUS_OK, status)
	testify.Zero_Allocation(t, func() {
		count, status = hex.Encode_Into(destination[:0], source)
	})
	testify.Equal_Values(t, hex.STATUS_OUTPUT_TOO_SMALL, status)
	testify.Zero_Allocation(t, func() {
		count, status = hex.Encode_Into(overlap[:], overlap[:1])
	})
	testify.Equal_Values(t, hex.STATUS_STORAGE_INVALID, status)
	testify.True(t, count <= hex.ENCODED_SIZE_MAXIMUM)
}

func test_decode_allocation(t *testing.T) {
	var destination [hex.DECODED_SIZE_MAXIMUM]byte
	var overlap [hex.ENCODED_BYTE_SIZE]byte
	valid := hex.Encoded("616c6c6f636174696f6e")
	var count hex.Decoded_Count
	var status hex.Decode_Status
	testify.Zero_Allocation(t, func() {
		count, status = hex.Decode_Into(destination[:], valid)
	})
	testify.Equal_Values(t, hex.STATUS_OK, status)
	testify.Zero_Allocation(t, func() {
		count, status = hex.Decode_Into(destination[:], []byte("gg"))
	})
	testify.Equal_Values(t, hex.STATUS_INPUT_INVALID, status)
	testify.Zero_Allocation(t, func() {
		count, status = hex.Decode_Into(destination[:], []byte("0"))
	})
	testify.Equal_Values(t, hex.STATUS_INPUT_INCOMPLETE, status)
	testify.Zero_Allocation(t, func() {
		count, status = hex.Decode_Into(destination[:0], valid)
	})
	testify.Equal_Values(t, hex.STATUS_OUTPUT_TOO_SMALL, status)
	testify.Zero_Allocation(t, func() {
		count, status = hex.Decode_Into(overlap[:1], overlap[:])
	})
	testify.Equal_Values(t, hex.STATUS_STORAGE_INVALID, status)
	testify.True(t, count <= hex.DECODED_SIZE_MAXIMUM)
}

func test_dump_allocation(t *testing.T) {
	var destination [hex.DUMP_SIZE_MAXIMUM]byte
	var overlap [hex.DUMP_SIZE_MAXIMUM]byte
	source := hex.Dump_Source("allocation")
	var count hex.Dump_Count
	var status hex.Dump_Status
	testify.Zero_Allocation(t, func() {
		count, status = hex.Dump_Into(destination[:], source)
	})
	testify.Equal_Values(t, hex.STATUS_OK, status)
	testify.Zero_Allocation(t, func() {
		count, status = hex.Dump_Into(destination[:0], source)
	})
	testify.Equal_Values(t, hex.STATUS_OUTPUT_TOO_SMALL, status)
	testify.Zero_Allocation(t, func() {
		count, status = hex.Dump_Into(overlap[:], overlap[:1])
	})
	testify.Equal_Values(t, hex.STATUS_STORAGE_INVALID, status)
	testify.True(t, count <= hex.DUMP_SIZE_MAXIMUM)
}
