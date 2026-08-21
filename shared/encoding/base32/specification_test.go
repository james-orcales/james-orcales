package base32_test

import (
	"testing"

	"local/james-orcales/shared/encoding/base32"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/testify"
)

// Test_Configuration protects custom alphabets without hidden pointer ownership.
func Test_Configuration(t *testing.T) {
	standard := base32.Standard_Encoding()
	hexadecimal := base32.Hexadecimal_Encoding()
	assert_encode(t, standard, "foobar", "MZXW6YTBOI======")
	assert_encode(t, hexadecimal, "foobar", "CPNMUOJ1E8======")

	alphabet := test_alphabet(TEST_STANDARD_ALPHABET)
	custom, configuration_status := base32.New_Encoding(alphabet, base32.NO_PADDING)
	testify.Equal_Values(t, base32.STATUS_OK, configuration_status)
	assert_encode(t, custom, "f", "MY")
	assert_encode(t, custom, "foobar", "MZXW6YTBOI")

	padded, padding_status := base32.With_Padding(custom, base32.STANDARD_PADDING)
	testify.Equal_Values(t, base32.STATUS_OK, padding_status)
	assert_encode(t, padded, "f", "MY======")

	duplicate := alphabet
	duplicate[base32.ALPHABET_SIZE-1] = duplicate[0]
	_, configuration_status = base32.New_Encoding(duplicate, base32.STANDARD_PADDING)
	testify.Equal_Values(t, base32.STATUS_ALPHABET_INVALID, configuration_status)
	newline := alphabet
	newline[base32.ALPHABET_SIZE-1] = '\n'
	_, configuration_status = base32.New_Encoding(newline, base32.STANDARD_PADDING)
	testify.Equal_Values(t, base32.STATUS_ALPHABET_INVALID, configuration_status)
	_, configuration_status = base32.New_Encoding(
		alphabet, base32.Padding(bits.INTEGER_16_MINIMUM),
	)
	testify.Equal_Values(t, base32.STATUS_PADDING_INVALID, configuration_status)
	_, configuration_status = base32.New_Encoding(
		alphabet, base32.Padding(bits.INTEGER_16_MAXIMUM),
	)
	testify.Equal_Values(t, base32.STATUS_PADDING_INVALID, configuration_status)
	_, configuration_status = base32.New_Encoding(
		alphabet, base32.Padding(alphabet[0]),
	)
	testify.Equal_Values(t, base32.STATUS_PADDING_INVALID, configuration_status)
	_, padding_status = base32.With_Padding(
		custom, base32.Padding(bits.INTEGER_16_MINIMUM),
	)
	testify.Equal_Values(t, base32.STATUS_PADDING_INVALID, padding_status)
	_, padding_status = base32.With_Padding(
		custom, base32.Padding(bits.INTEGER_16_MAXIMUM),
	)
	testify.Equal_Values(t, base32.STATUS_PADDING_INVALID, padding_status)
	_, padding_status = base32.With_Padding(base32.Encoding{}, base32.NO_PADDING)
	testify.Equal_Values(t, base32.STATUS_ENCODING_INVALID, padding_status)
	for _, padding := range [...]base32.Padding{0, 1, 2} {
		_, configuration_status = base32.New_Encoding(alphabet, padding)
		testify.Equal_Values(t, base32.STATUS_OK, configuration_status)
		_, padding_status = base32.With_Padding(custom, padding)
		testify.Equal_Values(t, base32.STATUS_OK, padding_status)
	}
}

// Test_Encode preserves RFC 4648 wire forms with exact caller capacity.
func Test_Encode(t *testing.T) {
	encoding := base32.Standard_Encoding()
	cases := [...]test_case{
		{Decoded: "", Encoded: ""},
		{Decoded: "f", Encoded: "MY======"},
		{Decoded: "fo", Encoded: "MZXQ===="},
		{Decoded: "foo", Encoded: "MZXW6==="},
		{Decoded: "foob", Encoded: "MZXW6YQ="},
		{Decoded: "fooba", Encoded: "MZXW6YTB"},
		{Decoded: "foobar", Encoded: "MZXW6YTBOI======"},
	}
	for _, one := range cases {
		assert_encode(t, encoding, one.Decoded, one.Encoded)
	}

	var destination [base32.ENCODED_SIZE_MAXIMUM]byte
	for index := range destination {
		destination[index] = TEST_SENTINEL
	}
	required, size_status := base32.Encoded_Size(encoding, base32.Source_Count(len("f")))
	testify.Equal_Values(t, base32.STATUS_OK, size_status)
	count, encode_status := base32.Encode_Into(
		destination[:int(required)-1], base32.Source("f"), encoding,
	)
	testify.Equal(t, base32.Encoded_Count(0), count)
	testify.Equal_Values(t, base32.STATUS_OUTPUT_TOO_SMALL, encode_status)
	for _, value := range destination {
		testify.Equal(t, TEST_SENTINEL, value)
	}
	for _, size := range [...]int{1, 2} {
		count, encode_status = base32.Encode_Into(
			destination[:size], base32.Source("f"), encoding,
		)
		testify.Equal(t, base32.Encoded_Count(0), count)
		testify.Equal_Values(t, base32.STATUS_OUTPUT_TOO_SMALL, encode_status)
	}
	count, encode_status = base32.Encode_Into(destination[:1], nil, encoding)
	testify.Equal(t, base32.Encoded_Count(0), count)
	testify.Equal_Values(t, base32.STATUS_OK, encode_status)
}

// Test_Decode preserves padded, unpadded, newline, and malformed input behavior.
func Test_Decode(t *testing.T) {
	encoding := base32.Standard_Encoding()
	cases := [...]test_case{
		{Decoded: "", Encoded: ""},
		{Decoded: "f", Encoded: "MY======"},
		{Decoded: "fo", Encoded: "MZXQ===="},
		{Decoded: "foo", Encoded: "MZXW6==="},
		{Decoded: "foob", Encoded: "MZXW6YQ="},
		{Decoded: "fooba", Encoded: "MZXW6YTB"},
		{Decoded: "foobar", Encoded: "MZXW6YTBOI======"},
		{Decoded: "sure", Encoded: "ON2X\rEZ\nI="},
		{Decoded: "fooba", Encoded: "\nMZXW6YTB"},
	}
	for _, one := range cases {
		assert_decode(t, encoding, one.Encoded, one.Decoded)
	}

	unpadded, padding_status := base32.With_Padding(encoding, base32.NO_PADDING)
	testify.Equal_Values(t, base32.STATUS_OK, padding_status)
	assert_decode(t, unpadded, "MZXW6YTBOI", "foobar")

	invalid := [...]string{
		"A", "!!!!", "x===", "AA=A====", "AAA=AAAA", "MMMMMMMMM", "MMMMMM",
		"A=", "AA=", "AA==", "AA===", "AAAA=", "AAAA==", "AAAAA=",
		"AAAAA==", "A=======", "AAA=====", "AAAAAA==", "========",
	}
	var destination [base32.DECODED_SIZE_MAXIMUM]byte
	for _, source := range invalid {
		_, decode_status := base32.Decode_Into(
			destination[:], base32.Encoded(source), encoding,
		)
		testify.Equal_Values(t, base32.STATUS_INPUT_INVALID, decode_status)
	}

	count, decode_status := base32.Decode_Into(
		destination[:1], base32.Encoded("MY======"), encoding,
	)
	testify.Equal(t, base32.Decoded_Count(1), count)
	testify.Equal_Values(t, base32.STATUS_OK, decode_status)
	count, decode_status = base32.Decode_Into(
		destination[:2], base32.Encoded("MZXQ===="), encoding,
	)
	testify.Equal(t, base32.Decoded_Count(2), count)
	testify.Equal_Values(t, base32.STATUS_OK, decode_status)
	count, decode_status = base32.Decode_Into(
		destination[:0], base32.Encoded("MY======"), encoding,
	)
	testify.Equal(t, base32.Decoded_Count(0), count)
	testify.Equal_Values(t, base32.STATUS_OUTPUT_TOO_SMALL, decode_status)
}

// Test_Bounds reaches every public collection boundary and scalar result.
func Test_Bounds(t *testing.T) {
	test_maximum_round_trip(t)
	test_size_domains(t)
	test_bound_refusals(t)
	test_storage_and_configuration_statuses(t)
}

// Test_Allocation measures every public result path with the assertion tracker enabled.
func Test_Allocation(t *testing.T) {
	test_new_encoding_allocation(t)
	test_padding_allocation(t)
	test_factory_allocation(t)
	test_size_allocation(t)
	test_encode_allocation(t)
	test_decode_allocation(t)
}

func test_maximum_round_trip(t *testing.T) {
	encoding := base32.Standard_Encoding()
	var source [base32.SOURCE_SIZE_MAXIMUM]byte
	var encoded [base32.ENCODED_SIZE_MAXIMUM]byte
	var decoded [base32.DECODED_SIZE_MAXIMUM]byte
	for index := range source {
		source[index] = byte(index)
	}

	encoded_size, size_status := base32.Encoded_Size(
		encoding, base32.Source_Count(len(source)),
	)
	testify.Equal(t, base32.Encoded_Count(len(encoded)), encoded_size)
	testify.Equal_Values(t, base32.STATUS_OK, size_status)
	encoded_count, encode_status := base32.Encode_Into(encoded[:], source[:], encoding)
	testify.Equal(t, encoded_size, encoded_count)
	testify.Equal_Values(t, base32.STATUS_OK, encode_status)
	decoded_size, size_status := base32.Decoded_Size_Maximum(
		encoding, base32.Encoded_Input_Count(encoded_count),
	)
	testify.Equal(t, base32.Decoded_Count(len(decoded)), decoded_size)
	testify.Equal_Values(t, base32.STATUS_OK, size_status)
	decoded_count, decode_status := base32.Decode_Into(
		decoded[:], encoded[:encoded_count], encoding,
	)
	testify.Equal(t, base32.Decoded_Count(len(source)), decoded_count)
	testify.Equal_Values(t, base32.STATUS_OK, decode_status)
	testify.Equal(t, source[:], decoded[:decoded_count])
}

func test_size_domains(t *testing.T) {
	encoding := base32.Standard_Encoding()
	for _, size := range [...]int{0, 1, 2, base32.SOURCE_SIZE_MAXIMUM} {
		_, size_status := base32.Encoded_Size(encoding, base32.Source_Count(size))
		testify.Equal_Values(t, base32.STATUS_OK, size_status)
	}
	for _, size := range [...]int{0, 1, 2, base32.ENCODED_SIZE_MAXIMUM} {
		_, size_status := base32.Decoded_Size_Maximum(
			encoding, base32.Encoded_Input_Count(size),
		)
		testify.Equal_Values(t, base32.STATUS_OK, size_status)
	}
	unpadded, padding_status := base32.With_Padding(encoding, base32.NO_PADDING)
	testify.Equal_Values(t, base32.STATUS_OK, padding_status)
	for _, one := range [...]struct {
		Encoded base32.Encoded_Input_Count
		Decoded base32.Decoded_Count
	}{
		{Encoded: base32.ENCODED_TAIL_SIZE_ONE, Decoded: 1},
		{Encoded: base32.ENCODED_TAIL_SIZE_TWO, Decoded: 2},
	} {
		decoded_size, size_status := base32.Decoded_Size_Maximum(unpadded, one.Encoded)
		testify.Equal(t, one.Decoded, decoded_size)
		testify.Equal_Values(t, base32.STATUS_OK, size_status)
	}
}

func test_bound_refusals(t *testing.T) {
	encoding := base32.Standard_Encoding()
	var encoded [base32.ENCODED_SIZE_MAXIMUM]byte
	var decoded [base32.DECODED_SIZE_MAXIMUM]byte
	var source_oversized [base32.SOURCE_SIZE_MAXIMUM + 1]byte
	var encoded_oversized [base32.ENCODED_SIZE_MAXIMUM + 1]byte
	var decoded_oversized [base32.DECODED_SIZE_MAXIMUM + 1]byte
	testify.Panics(t, func() {
		base32.Encode_Into(encoded[:], source_oversized[:], encoding)
	})
	testify.Panics(t, func() {
		base32.Encode_Into(encoded_oversized[:], nil, encoding)
	})
	testify.Panics(t, func() {
		base32.Decode_Into(decoded[:], encoded_oversized[:], encoding)
	})
	testify.Panics(t, func() {
		base32.Decode_Into(decoded_oversized[:], nil, encoding)
	})
}

func test_storage_and_configuration_statuses(t *testing.T) {
	encoding := base32.Standard_Encoding()
	var encoded [base32.ENCODED_SIZE_MAXIMUM]byte
	var decoded [base32.DECODED_SIZE_MAXIMUM]byte
	var overlap [base32.ENCODED_SIZE_MAXIMUM]byte
	_, encode_status := base32.Encode_Into(
		overlap[:base32.ENCODED_GROUP_SIZE], overlap[:base32.DECODED_GROUP_SIZE], encoding,
	)
	testify.Equal_Values(t, base32.STATUS_STORAGE_INVALID, encode_status)
	_, decode_status := base32.Decode_Into(
		overlap[:base32.DECODED_GROUP_SIZE], overlap[:base32.ENCODED_GROUP_SIZE], encoding,
	)
	testify.Equal_Values(t, base32.STATUS_STORAGE_INVALID, decode_status)

	_, encode_status = base32.Encode_Into(encoded[:], nil, base32.Encoding{})
	testify.Equal_Values(t, base32.STATUS_ENCODING_INVALID, encode_status)
	_, decode_status = base32.Decode_Into(decoded[:], nil, base32.Encoding{})
	testify.Equal_Values(t, base32.STATUS_ENCODING_INVALID, decode_status)
	_, size_status := base32.Encoded_Size(base32.Encoding{}, 0)
	testify.Equal_Values(t, base32.STATUS_ENCODING_INVALID, size_status)
	_, size_status = base32.Decoded_Size_Maximum(base32.Encoding{}, 0)
	testify.Equal_Values(t, base32.STATUS_ENCODING_INVALID, size_status)
}

const TEST_STANDARD_ALPHABET = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567"
const TEST_SENTINEL byte = 0xa5

type test_case struct {
	Decoded string
	Encoded string
}

func test_alphabet(text string) (alphabet base32.Alphabet) {
	copy(alphabet[:], text)
	return alphabet
}

func test_new_encoding_allocation(t *testing.T) {
	alphabet := test_alphabet(TEST_STANDARD_ALPHABET)
	duplicate := alphabet
	duplicate[base32.ALPHABET_FINAL_INDEX] = duplicate[0]
	var configured base32.Encoding
	var status base32.Configuration_Status
	testify.Zero_Allocation(t, func() {
		configured, status = base32.New_Encoding(alphabet, base32.STANDARD_PADDING)
	})
	testify.True(t, configured != base32.Encoding{})
	testify.Equal_Values(t, base32.STATUS_OK, status)
	testify.Zero_Allocation(t, func() {
		configured, status = base32.New_Encoding(duplicate, base32.STANDARD_PADDING)
	})
	testify.Equal(t, base32.Encoding{}, configured)
	testify.Equal_Values(t, base32.STATUS_ALPHABET_INVALID, status)
	testify.Zero_Allocation(t, func() {
		configured, status = base32.New_Encoding(
			alphabet, base32.Padding(bits.INTEGER_16_MINIMUM),
		)
	})
	testify.Equal(t, base32.Encoding{}, configured)
	testify.Equal_Values(t, base32.STATUS_PADDING_INVALID, status)
}

func test_padding_allocation(t *testing.T) {
	encoding := base32.Standard_Encoding()
	var configured base32.Encoding
	var status base32.Padding_Status
	testify.Zero_Allocation(t, func() {
		configured, status = base32.With_Padding(encoding, base32.NO_PADDING)
	})
	testify.True(t, configured != base32.Encoding{})
	testify.Equal_Values(t, base32.STATUS_OK, status)
	testify.Zero_Allocation(t, func() {
		configured, status = base32.With_Padding(base32.Encoding{}, base32.NO_PADDING)
	})
	testify.Equal(t, base32.Encoding{}, configured)
	testify.Equal_Values(t, base32.STATUS_ENCODING_INVALID, status)
	testify.Zero_Allocation(t, func() {
		configured, status = base32.With_Padding(
			encoding, base32.Padding(bits.INTEGER_16_MINIMUM),
		)
	})
	testify.Equal(t, base32.Encoding{}, configured)
	testify.Equal_Values(t, base32.STATUS_PADDING_INVALID, status)
}

func test_factory_allocation(t *testing.T) {
	var configured base32.Encoding
	testify.Zero_Allocation(t, func() { configured = base32.Standard_Encoding() })
	testify.True(t, configured != base32.Encoding{})
	testify.Zero_Allocation(t, func() { configured = base32.Hexadecimal_Encoding() })
	testify.True(t, configured != base32.Encoding{})
}

func test_size_allocation(t *testing.T) {
	encoding := base32.Standard_Encoding()
	source_count := base32.Source_Count(len("allocation"))
	encoded_count := base32.Encoded_Input_Count(len("MFWHA3LPNRSXQ2LO"))
	var encoded_size base32.Encoded_Count
	var decoded_size base32.Decoded_Count
	var status base32.Size_Status
	testify.Zero_Allocation(t, func() {
		encoded_size, status = base32.Encoded_Size(encoding, source_count)
	})
	testify.True(t, encoded_size > 0)
	testify.Equal_Values(t, base32.STATUS_OK, status)
	testify.Zero_Allocation(t, func() {
		encoded_size, status = base32.Encoded_Size(base32.Encoding{}, source_count)
	})
	testify.Equal(t, base32.Encoded_Count(0), encoded_size)
	testify.Equal_Values(t, base32.STATUS_ENCODING_INVALID, status)
	testify.Zero_Allocation(t, func() {
		decoded_size, status = base32.Decoded_Size_Maximum(encoding, encoded_count)
	})
	testify.True(t, decoded_size > 0)
	testify.Equal_Values(t, base32.STATUS_OK, status)
	testify.Zero_Allocation(t, func() {
		decoded_size, status = base32.Decoded_Size_Maximum(base32.Encoding{}, encoded_count)
	})
	testify.Equal(t, base32.Decoded_Count(0), decoded_size)
	testify.Equal_Values(t, base32.STATUS_ENCODING_INVALID, status)
}

func test_encode_allocation(t *testing.T) {
	encoding := base32.Standard_Encoding()
	source := base32.Source("allocation")
	var destination [base32.ENCODED_SIZE_MAXIMUM]byte
	var overlap [base32.ENCODED_GROUP_SIZE]byte
	var count base32.Encoded_Count
	var status base32.Encode_Status
	testify.Zero_Allocation(t, func() {
		count, status = base32.Encode_Into(destination[:], source, encoding)
	})
	testify.True(t, count > 0)
	testify.Equal_Values(t, base32.STATUS_OK, status)
	testify.Zero_Allocation(t, func() {
		count, status = base32.Encode_Into(destination[:0], source, encoding)
	})
	testify.Equal(t, base32.Encoded_Count(0), count)
	testify.Equal_Values(t, base32.STATUS_OUTPUT_TOO_SMALL, status)
	testify.Zero_Allocation(t, func() {
		count, status = base32.Encode_Into(
			overlap[:], overlap[:base32.DECODED_GROUP_SIZE], encoding,
		)
	})
	testify.Equal(t, base32.Encoded_Count(0), count)
	testify.Equal_Values(t, base32.STATUS_STORAGE_INVALID, status)
	testify.Zero_Allocation(t, func() {
		count, status = base32.Encode_Into(destination[:], source, base32.Encoding{})
	})
	testify.Equal(t, base32.Encoded_Count(0), count)
	testify.Equal_Values(t, base32.STATUS_ENCODING_INVALID, status)
}

func test_decode_allocation(t *testing.T) {
	encoding := base32.Standard_Encoding()
	source := base32.Encoded("MFWHA3LPNRSXQ2LO")
	malformed := base32.Encoded("!!!!!!!!")
	var destination [base32.DECODED_SIZE_MAXIMUM]byte
	var overlap [base32.ENCODED_GROUP_SIZE]byte
	var count base32.Decoded_Count
	var status base32.Decode_Status
	testify.Zero_Allocation(t, func() {
		count, status = base32.Decode_Into(destination[:], source, encoding)
	})
	testify.True(t, count > 0)
	testify.Equal_Values(t, base32.STATUS_OK, status)
	testify.Zero_Allocation(t, func() {
		count, status = base32.Decode_Into(destination[:], malformed, encoding)
	})
	testify.Equal(t, base32.Decoded_Count(0), count)
	testify.Equal_Values(t, base32.STATUS_INPUT_INVALID, status)
	testify.Zero_Allocation(t, func() {
		count, status = base32.Decode_Into(destination[:0], source, encoding)
	})
	testify.Equal(t, base32.Decoded_Count(0), count)
	testify.Equal_Values(t, base32.STATUS_OUTPUT_TOO_SMALL, status)
	testify.Zero_Allocation(t, func() {
		count, status = base32.Decode_Into(
			overlap[:base32.DECODED_GROUP_SIZE], overlap[:], encoding,
		)
	})
	testify.Equal(t, base32.Decoded_Count(0), count)
	testify.Equal_Values(t, base32.STATUS_STORAGE_INVALID, status)
	testify.Zero_Allocation(t, func() {
		count, status = base32.Decode_Into(destination[:], source, base32.Encoding{})
	})
	testify.Equal(t, base32.Decoded_Count(0), count)
	testify.Equal_Values(t, base32.STATUS_ENCODING_INVALID, status)
}

func assert_encode(t *testing.T, encoding base32.Encoding, source string, expected string) {
	t.Helper()
	var destination [base32.ENCODED_SIZE_MAXIMUM]byte
	required, size_status := base32.Encoded_Size(
		encoding, base32.Source_Count(len(source)),
	)
	testify.Equal_Values(t, base32.STATUS_OK, size_status)
	testify.Equal(t, base32.Encoded_Count(len(expected)), required)
	count, encode_status := base32.Encode_Into(
		destination[:int(required)], []byte(source), encoding,
	)
	testify.Equal_Values(t, base32.STATUS_OK, encode_status)
	testify.Equal(t, required, count)
	testify.Equal(t, expected, string(destination[:count]))
}

func assert_decode(t *testing.T, encoding base32.Encoding, source string, expected string) {
	t.Helper()
	var destination [base32.DECODED_SIZE_MAXIMUM]byte
	count, status := base32.Decode_Into(destination[:], []byte(source), encoding)
	testify.Equal_Values(t, base32.STATUS_OK, status)
	testify.Equal(t, base32.Decoded_Count(len(expected)), count)
	testify.Equal(t, expected, string(destination[:count]))
}
