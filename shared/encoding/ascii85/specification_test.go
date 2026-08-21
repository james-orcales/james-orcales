package ascii85_test

import (
	"testing"

	"local/james-orcales/shared/encoding/ascii85"
	"local/james-orcales/shared/testify"
)

// Test_Encode keeps upstream wire forms while caller storage owns every result.
func Test_Encode(t *testing.T) {
	cases := [...]encode_case{
		{Decoded: "", Encoded: ""},
		{Decoded: "Man ", Encoded: "9jqo^"},
		{Decoded: "\x00\x00\x00\x00", Encoded: "z"},
		{Decoded: "f", Encoded: "Ac"},
		{Decoded: "fo", Encoded: "Ao@"},
		{Decoded: "foo", Encoded: "AoDS"},
		{Decoded: "foobar", Encoded: "AoDTs@<)"},
	}
	for _, one := range cases {
		assert_encode(t, one)
	}

	var short [len("9jqo^") - 1]byte
	for index := range short {
		short[index] = TEST_SENTINEL
	}
	count, status := ascii85.Encode_Into(short[:], []byte("Man "))
	testify.Equal(t, ascii85.Encoded_Count(0), count)
	testify.Equal_Values(t, ascii85.STATUS_OUTPUT_TOO_SMALL, status)
	expected := [len(short)]byte{
		TEST_SENTINEL, TEST_SENTINEL, TEST_SENTINEL, TEST_SENTINEL,
	}
	testify.Equal(t, expected, short)
}

// Test_Decode preserves whitespace, zero block, partial flush, and corruption behavior.
func Test_Decode(t *testing.T) {
	assert_decode(t, "9jqo^", "Man ", true)
	assert_decode(t, " 9j\nqo^\t", "Man ", true)
	assert_decode(t, "z", "\x00\x00\x00\x00", true)
	assert_decode(t, "Ac", "f", true)

	var destination [ascii85.DECODED_SIZE_MAXIMUM]byte
	decoded, consumed, status := ascii85.Decode_Into(destination[:], []byte("Ac"), false)
	testify.Equal(t, ascii85.Decoded_Count(0), decoded)
	testify.Equal(t, ascii85.Consumed_Count(0), consumed)
	testify.Equal_Values(t, ascii85.STATUS_OK, status)

	invalid := [...]string{"v", "!z!!!!!!!!!", "!"}
	for _, source := range invalid {
		decoded, consumed, status = ascii85.Decode_Into(
			destination[:], []byte(source), true,
		)
		testify.Equal(t, ascii85.Decoded_Count(0), decoded)
		testify.Equal(t, ascii85.Consumed_Count(0), consumed)
		testify.Equal_Values(t, ascii85.STATUS_INPUT_INVALID, status)
	}

	var short [ascii85.DECODED_GROUP_SIZE - 1]byte
	decoded, consumed, status = ascii85.Decode_Into(short[:], []byte("z"), true)
	testify.Equal(t, ascii85.Decoded_Count(0), decoded)
	testify.Equal(t, ascii85.Consumed_Count(0), consumed)
	testify.Equal_Values(t, ascii85.STATUS_OUTPUT_TOO_SMALL, status)
}

// Test_Bounds reaches every declared collection boundary through public operations.
func Test_Bounds(t *testing.T) {
	test_encode_domains(t)
	test_decode_domains(t)
	test_bound_refusals(t)
}

// Test_Allocation measures each public result path with the assertion tracker enabled.
func Test_Allocation(t *testing.T) {
	test_allocation(t)
}

func test_allocation(t *testing.T) {
	var encoded [ascii85.ENCODED_SIZE_MAXIMUM]byte
	var decoded [ascii85.DECODED_SIZE_MAXIMUM]byte
	source := ascii85.Encode_Source("allocation")
	valid := ascii85.Encoded("@4lLN0JP==1c70M")
	invalid := ascii85.Encoded("v")
	var encoded_count ascii85.Encoded_Count
	var maximum_count ascii85.Maximum_Encoded_Count
	var decoded_count ascii85.Decoded_Count
	var consumed_count ascii85.Consumed_Count
	var encode_status ascii85.Encode_Status
	var decode_status ascii85.Status

	testify.Zero_Allocation(t, func() {
		maximum_count = ascii85.Encoded_Size_Maximum(ascii85.Source_Count(len(source)))
	})
	testify.Zero_Allocation(t, func() {
		encoded_count = ascii85.Encoded_Size(source)
	})
	testify.Zero_Allocation(t, func() {
		encoded_count, encode_status = ascii85.Encode_Into(encoded[:], source)
	})
	testify.Zero_Allocation(t, func() {
		encoded_count, encode_status = ascii85.Encode_Into(encoded[:0], source)
	})
	testify.Zero_Allocation(t, func() {
		decoded_count, consumed_count, decode_status = ascii85.Decode_Into(
			decoded[:], valid, true,
		)
	})
	testify.Zero_Allocation(t, func() {
		decoded_count, consumed_count, decode_status = ascii85.Decode_Into(
			decoded[:], invalid, true,
		)
	})
	testify.Zero_Allocation(t, func() {
		decoded_count, consumed_count, decode_status = ascii85.Decode_Into(
			decoded[:0], valid, true,
		)
	})
	testify.Zero_Allocation(t, func() {
		decoded_count, consumed_count, decode_status = ascii85.Decode_Into(
			decoded[:], valid[:1], false,
		)
	})

	testify.True(t, encoded_count <= ascii85.ENCODED_SIZE_MAXIMUM)
	testify.True(t, maximum_count <= ascii85.ENCODED_SIZE_MAXIMUM)
	testify.True(t, decoded_count <= ascii85.DECODED_SIZE_MAXIMUM)
	testify.True(t, consumed_count <= ascii85.ENCODED_INPUT_SIZE_MAXIMUM)
	testify.True(t, encode_status <= ascii85.STATUS_OUTPUT_TOO_SMALL)
	testify.True(t, decode_status <= ascii85.STATUS_OUTPUT_TOO_SMALL)
}

const TEST_SENTINEL byte = 0xa5

type encode_case struct {
	Decoded string
	Encoded string
}

func assert_encode(t *testing.T, one encode_case) {
	t.Helper()
	var destination [ascii85.ENCODED_SIZE_MAXIMUM]byte
	source := ascii85.Encode_Source(one.Decoded)
	maximum := ascii85.Encoded_Size_Maximum(ascii85.Source_Count(len(source)))
	exact := ascii85.Encoded_Size(source)
	count, status := ascii85.Encode_Into(destination[:], source)
	testify.Equal_Values(t, ascii85.STATUS_OK, status)
	testify.Equal(t, ascii85.Encoded_Count(len(one.Encoded)), count)
	testify.Equal(t, ascii85.Encoded_Count(len(one.Encoded)), exact)
	testify.True(t, exact <= ascii85.Encoded_Count(maximum))
	testify.Equal(t, one.Encoded, string(destination[:count]))
}

func assert_decode(t *testing.T, source string, expected string, flush bool) {
	t.Helper()
	var destination [ascii85.DECODED_SIZE_MAXIMUM]byte
	decoded, consumed, status := ascii85.Decode_Into(
		destination[:], []byte(source), ascii85.Flush(flush),
	)
	testify.Equal_Values(t, ascii85.STATUS_OK, status)
	testify.Equal(t, ascii85.Decoded_Count(len(expected)), decoded)
	testify.Equal(t, ascii85.Consumed_Count(len(source)), consumed)
	testify.Equal(t, expected, string(destination[:decoded]))
}

func test_encode_domains(t *testing.T) {
	var source_storage [ascii85.ENCODE_SOURCE_SIZE_MAXIMUM]byte
	var destination [ascii85.ENCODED_INPUT_SIZE_MAXIMUM]byte
	for index := range source_storage {
		source_storage[index] = byte(index%int(TEST_SENTINEL) + 1)
	}
	sizes := [...]int{0, 1, 2, ascii85.ENCODE_SOURCE_SIZE_MAXIMUM}
	for _, size := range sizes {
		source := ascii85.Encode_Source(source_storage[:size])
		maximum := ascii85.Encoded_Size_Maximum(ascii85.Source_Count(size))
		count, status := ascii85.Encode_Into(destination[:int(maximum)], source)
		testify.Equal_Values(t, ascii85.STATUS_OK, status)
		testify.True(t, count <= ascii85.Encoded_Count(maximum))
	}

	var zero_group [ascii85.DECODED_GROUP_SIZE]byte
	count, status := ascii85.Encode_Into(destination[:1], zero_group[:])
	testify.Equal(t, ascii85.Encoded_Count(1), count)
	testify.Equal_Values(t, ascii85.STATUS_OK, status)
	count, status = ascii85.Encode_Into(destination[:2], source_storage[:1])
	testify.Equal(t, ascii85.Encoded_Count(2), count)
	testify.Equal_Values(t, ascii85.STATUS_OK, status)
	count, status = ascii85.Encode_Into(destination[:0], source_storage[:1])
	testify.Equal(t, ascii85.Encoded_Count(0), count)
	testify.Equal_Values(t, ascii85.STATUS_OUTPUT_TOO_SMALL, status)
	count, status = ascii85.Encode_Into(destination[:], nil)
	testify.Equal(t, ascii85.Encoded_Count(0), count)
	testify.Equal_Values(t, ascii85.STATUS_OK, status)
}

func test_decode_domains(t *testing.T) {
	var decoded [ascii85.DECODED_SIZE_MAXIMUM]byte
	var encoded [ascii85.ENCODED_INPUT_SIZE_MAXIMUM]byte
	for index := range encoded {
		encoded[index] = ' '
	}

	decoded_count, consumed, status := ascii85.Decode_Into(decoded[:0], nil, true)
	testify.Equal(t, ascii85.Decoded_Count(0), decoded_count)
	testify.Equal(t, ascii85.Consumed_Count(0), consumed)
	testify.Equal_Values(t, ascii85.STATUS_OK, status)
	decoded_count, consumed, status = ascii85.Decode_Into(decoded[:1], []byte("!!"), true)
	testify.Equal(t, ascii85.Decoded_Count(1), decoded_count)
	testify.Equal(t, ascii85.Consumed_Count(2), consumed)
	testify.Equal_Values(t, ascii85.STATUS_OK, status)
	decoded_count, consumed, status = ascii85.Decode_Into(decoded[:2], []byte("!!!"), true)
	testify.Equal(t, ascii85.Decoded_Count(2), decoded_count)
	testify.Equal(t, ascii85.Consumed_Count(3), consumed)
	testify.Equal_Values(t, ascii85.STATUS_OK, status)

	zero_count := ascii85.DECODED_SIZE_MAXIMUM / ascii85.DECODED_GROUP_SIZE
	for index := 0; index < zero_count; index++ {
		encoded[index] = 'z'
	}
	decoded_count, consumed, status = ascii85.Decode_Into(
		decoded[:], encoded[:zero_count], true,
	)
	testify.Equal(t, ascii85.Decoded_Count(ascii85.DECODED_SIZE_MAXIMUM), decoded_count)
	testify.Equal(t, ascii85.Consumed_Count(zero_count), consumed)
	testify.Equal_Values(t, ascii85.STATUS_OK, status)

	for index := range encoded {
		encoded[index] = ' '
	}
	decoded_count, consumed, status = ascii85.Decode_Into(decoded[:], encoded[:], true)
	testify.Equal(t, ascii85.Decoded_Count(0), decoded_count)
	testify.Equal(t, ascii85.Consumed_Count(ascii85.ENCODED_INPUT_SIZE_MAXIMUM), consumed)
	testify.Equal_Values(t, ascii85.STATUS_OK, status)
}

func test_bound_refusals(t *testing.T) {
	var source_oversized [ascii85.ENCODE_SOURCE_SIZE_MAXIMUM + 1]byte
	var encoded_oversized [ascii85.ENCODED_INPUT_SIZE_MAXIMUM + 1]byte
	var decoded_oversized [ascii85.DECODED_SIZE_MAXIMUM + 1]byte
	var encoded [ascii85.ENCODED_SIZE_MAXIMUM]byte
	var decoded [ascii85.DECODED_SIZE_MAXIMUM]byte
	var encoded_count ascii85.Encoded_Count
	var decoded_count ascii85.Decoded_Count
	var consumed_count ascii85.Consumed_Count
	var encode_status ascii85.Encode_Status
	var decode_status ascii85.Status

	testify.Panics(t, func() {
		encoded_count, encode_status = ascii85.Encode_Into(encoded[:], source_oversized[:])
	})
	testify.Panics(t, func() {
		encoded_count, encode_status = ascii85.Encode_Into(encoded_oversized[:], nil)
	})
	testify.Panics(t, func() {
		decoded_count, consumed_count, decode_status = ascii85.Decode_Into(
			decoded[:], encoded_oversized[:], true,
		)
	})
	testify.Panics(t, func() {
		decoded_count, consumed_count, decode_status = ascii85.Decode_Into(
			decoded_oversized[:], nil, true,
		)
	})

	var overlap [ascii85.ENCODED_GROUP_SIZE]byte
	testify.Panics(t, func() {
		encoded_count, encode_status = ascii85.Encode_Into(
			overlap[:], overlap[:ascii85.DECODED_GROUP_SIZE],
		)
	})
	testify.Panics(t, func() {
		decoded_count, consumed_count, decode_status = ascii85.Decode_Into(
			overlap[:ascii85.DECODED_GROUP_SIZE], overlap[:], true,
		)
	})

	testify.True(t, encoded_count <= ascii85.ENCODED_SIZE_MAXIMUM)
	testify.True(t, decoded_count <= ascii85.DECODED_SIZE_MAXIMUM)
	testify.True(t, consumed_count <= ascii85.ENCODED_INPUT_SIZE_MAXIMUM)
	testify.True(t, encode_status <= ascii85.STATUS_OUTPUT_TOO_SMALL)
	testify.True(t, decode_status <= ascii85.STATUS_OUTPUT_TOO_SMALL)
}
