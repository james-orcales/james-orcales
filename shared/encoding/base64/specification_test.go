package base64_test

import (
	"testing"

	"local/james-orcales/shared/encoding/base64"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/testify"
)

// Test_Configuration protects standard, URL, raw, custom, and rejected configurations.
func Test_Configuration(t *testing.T) {
	standard := base64.Standard_Encoding()
	url := base64.URL_Encoding()
	raw_standard := base64.Raw_Standard_Encoding()
	raw_url := base64.Raw_URL_Encoding()
	assert_encode(t, standard, "foobar", "Zm9vYmFy")
	assert_encode(t, url, "\xfb\xff", "-_8=")
	assert_encode(t, raw_standard, "f", "Zg")
	assert_encode(t, raw_url, "\xfb\xff", "-_8")

	alphabet := test_alphabet(TEST_STANDARD_ALPHABET)
	custom, configuration_status := base64.New_Encoding(alphabet, base64.NO_PADDING)
	testify.Equal_Values(t, base64.STATUS_OK, configuration_status)
	assert_encode(t, custom, "f", "Zg")
	padded, padding_status := base64.With_Padding(custom, TEST_PADDING)
	testify.Equal_Values(t, base64.STATUS_OK, padding_status)
	assert_encode(t, padded, "f", "Zg@@")

	duplicate := alphabet
	duplicate[base64.ALPHABET_FINAL_INDEX] = duplicate[0]
	_, configuration_status = base64.New_Encoding(duplicate, base64.STANDARD_PADDING)
	testify.Equal_Values(t, base64.STATUS_ALPHABET_INVALID, configuration_status)
	newline := alphabet
	newline[base64.ALPHABET_FINAL_INDEX] = '\n'
	_, configuration_status = base64.New_Encoding(newline, base64.STANDARD_PADDING)
	testify.Equal_Values(t, base64.STATUS_ALPHABET_INVALID, configuration_status)
	_, configuration_status = base64.New_Encoding(
		alphabet, base64.Padding(bits.INTEGER_16_MINIMUM),
	)
	testify.Equal_Values(t, base64.STATUS_PADDING_INVALID, configuration_status)
	_, configuration_status = base64.New_Encoding(
		alphabet, base64.Padding(bits.INTEGER_16_MAXIMUM),
	)
	testify.Equal_Values(t, base64.STATUS_PADDING_INVALID, configuration_status)
	_, configuration_status = base64.New_Encoding(
		alphabet, base64.Padding(alphabet[0]),
	)
	testify.Equal_Values(t, base64.STATUS_PADDING_INVALID, configuration_status)
	_, padding_status = base64.With_Padding(
		custom, base64.Padding(bits.INTEGER_16_MINIMUM),
	)
	testify.Equal_Values(t, base64.STATUS_PADDING_INVALID, padding_status)
	_, padding_status = base64.With_Padding(
		custom, base64.Padding(bits.INTEGER_16_MAXIMUM),
	)
	testify.Equal_Values(t, base64.STATUS_PADDING_INVALID, padding_status)
	_, padding_status = base64.With_Padding(base64.Encoding{}, base64.NO_PADDING)
	testify.Equal_Values(t, base64.STATUS_ENCODING_INVALID, padding_status)
	for _, padding := range [...]base64.Padding{0, 1, 2} {
		_, configuration_status = base64.New_Encoding(alphabet, padding)
		testify.Equal_Values(t, base64.STATUS_OK, configuration_status)
		_, padding_status = base64.With_Padding(custom, padding)
		testify.Equal_Values(t, base64.STATUS_OK, padding_status)
	}
}

// Test_Encode preserves RFC 4648 wire forms with exact caller capacity.
func Test_Encode(t *testing.T) {
	encoding := base64.Standard_Encoding()
	cases := [...]test_case{
		{Decoded: "", Encoded: ""},
		{Decoded: "f", Encoded: "Zg=="},
		{Decoded: "fo", Encoded: "Zm8="},
		{Decoded: "foo", Encoded: "Zm9v"},
		{Decoded: "foob", Encoded: "Zm9vYg=="},
		{Decoded: "fooba", Encoded: "Zm9vYmE="},
		{Decoded: "foobar", Encoded: "Zm9vYmFy"},
	}
	for _, one := range cases {
		assert_encode(t, encoding, one.Decoded, one.Encoded)
	}

	var destination [base64.ENCODED_SIZE_MAXIMUM]byte
	for index := range destination {
		destination[index] = TEST_SENTINEL
	}
	for _, size := range [...]int{0, 1, 2} {
		count, status := base64.Encode_Into(
			destination[:size], base64.Source("f"), encoding,
		)
		testify.Equal(t, base64.Encoded_Count(0), count)
		testify.Equal_Values(t, base64.STATUS_OUTPUT_TOO_SMALL, status)
	}
	for _, value := range destination {
		testify.Equal(t, TEST_SENTINEL, value)
	}
	count, status := base64.Encode_Into(destination[:1], nil, encoding)
	testify.Equal(t, base64.Encoded_Count(0), count)
	testify.Equal_Values(t, base64.STATUS_OK, status)
}

// Test_Decode preserves padded, unpadded, newline, and malformed input behavior.
func Test_Decode(t *testing.T) {
	encoding := base64.Standard_Encoding()
	cases := [...]test_case{
		{Decoded: "", Encoded: ""},
		{Decoded: "f", Encoded: "Zg=="},
		{Decoded: "fo", Encoded: "Zm8="},
		{Decoded: "foo", Encoded: "Zm9v"},
		{Decoded: "foobar", Encoded: "Zm9vYmFy"},
		{Decoded: "sure", Encoded: "c3V\ryZ\nQ=="},
	}
	for _, one := range cases {
		assert_decode(t, encoding, one.Encoded, one.Decoded)
	}
	raw := base64.Raw_Standard_Encoding()
	assert_decode(t, raw, "Zg", "f")
	assert_decode(t, raw, "Zm8", "fo")

	invalid := [...]string{
		"A", "!!!!", "====", "x===", "=AAA", "A=AA", "AA=A", "AA==A",
		"AAA=AAAA", "AAAAA", "AAAAAA", "A=", "A==", "AA=", "AAAAAA=",
		"YWJjZA=====", "A!\n", "A=\n",
	}
	var destination [base64.DECODED_SIZE_MAXIMUM]byte
	for _, source := range invalid {
		_, status := base64.Decode_Into(destination[:], []byte(source), encoding)
		testify.Equal_Values(t, base64.STATUS_INPUT_INVALID, status)
	}
	count, status := base64.Decode_Into(destination[:1], []byte("Zg=="), encoding)
	testify.Equal(t, base64.Decoded_Count(1), count)
	testify.Equal_Values(t, base64.STATUS_OK, status)
	count, status = base64.Decode_Into(destination[:2], []byte("Zm8="), encoding)
	testify.Equal(t, base64.Decoded_Count(2), count)
	testify.Equal_Values(t, base64.STATUS_OK, status)
	count, status = base64.Decode_Into(destination[:0], []byte("Zg=="), encoding)
	testify.Equal(t, base64.Decoded_Count(0), count)
	testify.Equal_Values(t, base64.STATUS_OUTPUT_TOO_SMALL, status)
}

// Test_Strict_Decode rejects nonzero unused terminal bits only in strict mode.
func Test_Strict_Decode(t *testing.T) {
	standard := base64.Standard_Encoding()
	strict, strict_status := base64.Strict_Encoding(standard)
	testify.Equal_Values(t, base64.STATUS_OK, strict_status)
	var destination [base64.DECODED_SIZE_MAXIMUM]byte
	noncanonical := base64.Encoded("WvLTlMrX9NpYDQlEIFlnDB==")
	_, decode_status := base64.Decode_Into(destination[:], noncanonical, strict)
	testify.Equal_Values(t, base64.STATUS_INPUT_INVALID, decode_status)
	_, decode_status = base64.Decode_Into(destination[:], noncanonical, standard)
	testify.Equal_Values(t, base64.STATUS_OK, decode_status)
	canonical := base64.Encoded("WvLTlMrX9NpYDQlEIFlnDA==")
	_, decode_status = base64.Decode_Into(destination[:], canonical, strict)
	testify.Equal_Values(t, base64.STATUS_OK, decode_status)
	_, strict_status = base64.Strict_Encoding(base64.Encoding{})
	testify.Equal_Values(t, base64.STATUS_ENCODING_INVALID, strict_status)
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
	test_configuration_modifier_allocation(t)
	test_factory_allocation(t)
	test_size_allocation(t)
	test_encode_allocation(t)
	test_decode_allocation(t)
	test_strict_decode_allocation(t)
}

const TEST_STANDARD_ALPHABET = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
const TEST_SENTINEL byte = 0xa5
const TEST_PADDING base64.Padding = '@'

type test_case struct {
	Decoded string
	Encoded string
}

func test_alphabet(text string) (alphabet base64.Alphabet) {
	copy(alphabet[:], text)
	return alphabet
}

func test_new_encoding_allocation(t *testing.T) {
	alphabet := test_alphabet(TEST_STANDARD_ALPHABET)
	duplicate := alphabet
	duplicate[base64.ALPHABET_FINAL_INDEX] = duplicate[0]
	var configured base64.Encoding
	var status base64.Configuration_Status
	testify.Zero_Allocation(t, func() {
		configured, status = base64.New_Encoding(alphabet, base64.STANDARD_PADDING)
	})
	testify.True(t, configured != base64.Encoding{})
	testify.Equal_Values(t, base64.STATUS_OK, status)
	testify.Zero_Allocation(t, func() {
		configured, status = base64.New_Encoding(duplicate, base64.STANDARD_PADDING)
	})
	testify.Equal(t, base64.Encoding{}, configured)
	testify.Equal_Values(t, base64.STATUS_ALPHABET_INVALID, status)
	testify.Zero_Allocation(t, func() {
		configured, status = base64.New_Encoding(
			alphabet, base64.Padding(bits.INTEGER_16_MINIMUM),
		)
	})
	testify.Equal(t, base64.Encoding{}, configured)
	testify.Equal_Values(t, base64.STATUS_PADDING_INVALID, status)
}

func test_configuration_modifier_allocation(t *testing.T) {
	encoding := base64.Standard_Encoding()
	var configured base64.Encoding
	var padding_status base64.Padding_Status
	testify.Zero_Allocation(t, func() {
		configured, padding_status = base64.With_Padding(encoding, base64.NO_PADDING)
	})
	testify.True(t, configured != base64.Encoding{})
	testify.Equal_Values(t, base64.STATUS_OK, padding_status)
	testify.Zero_Allocation(t, func() {
		configured, padding_status = base64.With_Padding(
			base64.Encoding{}, base64.NO_PADDING,
		)
	})
	testify.Equal(t, base64.Encoding{}, configured)
	testify.Equal_Values(t, base64.STATUS_ENCODING_INVALID, padding_status)
	testify.Zero_Allocation(t, func() {
		configured, padding_status = base64.With_Padding(
			encoding, base64.Padding(bits.INTEGER_16_MINIMUM),
		)
	})
	testify.Equal(t, base64.Encoding{}, configured)
	testify.Equal_Values(t, base64.STATUS_PADDING_INVALID, padding_status)
	var strict_status base64.Strict_Status
	testify.Zero_Allocation(t, func() {
		configured, strict_status = base64.Strict_Encoding(encoding)
	})
	testify.True(t, configured != base64.Encoding{})
	testify.Equal_Values(t, base64.STATUS_OK, strict_status)
	testify.Zero_Allocation(t, func() {
		configured, strict_status = base64.Strict_Encoding(base64.Encoding{})
	})
	testify.Equal(t, base64.Encoding{}, configured)
	testify.Equal_Values(t, base64.STATUS_ENCODING_INVALID, strict_status)
}

func test_factory_allocation(t *testing.T) {
	var configured base64.Encoding
	testify.Zero_Allocation(t, func() { configured = base64.Standard_Encoding() })
	testify.True(t, configured != base64.Encoding{})
	testify.Zero_Allocation(t, func() { configured = base64.URL_Encoding() })
	testify.True(t, configured != base64.Encoding{})
	testify.Zero_Allocation(t, func() { configured = base64.Raw_Standard_Encoding() })
	testify.True(t, configured != base64.Encoding{})
	testify.Zero_Allocation(t, func() { configured = base64.Raw_URL_Encoding() })
	testify.True(t, configured != base64.Encoding{})
}

func test_size_allocation(t *testing.T) {
	encoding := base64.Standard_Encoding()
	source_count := base64.Source_Count(len("allocation"))
	encoded_count := base64.Encoded_Input_Count(len("YWxsb2NhdGlvbg=="))
	var encoded_size base64.Encoded_Count
	var decoded_size base64.Decoded_Count
	var status base64.Size_Status
	testify.Zero_Allocation(t, func() {
		encoded_size, status = base64.Encoded_Size(encoding, source_count)
	})
	testify.True(t, encoded_size > 0)
	testify.Equal_Values(t, base64.STATUS_OK, status)
	testify.Zero_Allocation(t, func() {
		encoded_size, status = base64.Encoded_Size(base64.Encoding{}, source_count)
	})
	testify.Equal(t, base64.Encoded_Count(0), encoded_size)
	testify.Equal_Values(t, base64.STATUS_ENCODING_INVALID, status)
	testify.Zero_Allocation(t, func() {
		decoded_size, status = base64.Decoded_Size_Maximum(encoding, encoded_count)
	})
	testify.True(t, decoded_size > 0)
	testify.Equal_Values(t, base64.STATUS_OK, status)
	testify.Zero_Allocation(t, func() {
		decoded_size, status = base64.Decoded_Size_Maximum(
			base64.Encoding{}, encoded_count,
		)
	})
	testify.Equal(t, base64.Decoded_Count(0), decoded_size)
	testify.Equal_Values(t, base64.STATUS_ENCODING_INVALID, status)
}

func test_encode_allocation(t *testing.T) {
	encoding := base64.Standard_Encoding()
	source := base64.Source("allocation")
	var destination [base64.ENCODED_SIZE_MAXIMUM]byte
	var overlap [base64.ENCODED_GROUP_SIZE]byte
	var count base64.Encoded_Count
	var status base64.Encode_Status
	testify.Zero_Allocation(t, func() {
		count, status = base64.Encode_Into(destination[:], source, encoding)
	})
	testify.True(t, count > 0)
	testify.Equal_Values(t, base64.STATUS_OK, status)
	testify.Zero_Allocation(t, func() {
		count, status = base64.Encode_Into(destination[:0], source, encoding)
	})
	testify.Equal(t, base64.Encoded_Count(0), count)
	testify.Equal_Values(t, base64.STATUS_OUTPUT_TOO_SMALL, status)
	testify.Zero_Allocation(t, func() {
		count, status = base64.Encode_Into(
			overlap[:], overlap[:base64.DECODED_GROUP_SIZE], encoding,
		)
	})
	testify.Equal(t, base64.Encoded_Count(0), count)
	testify.Equal_Values(t, base64.STATUS_STORAGE_INVALID, status)
	testify.Zero_Allocation(t, func() {
		count, status = base64.Encode_Into(destination[:], source, base64.Encoding{})
	})
	testify.Equal(t, base64.Encoded_Count(0), count)
	testify.Equal_Values(t, base64.STATUS_ENCODING_INVALID, status)
}

func test_decode_allocation(t *testing.T) {
	encoding := base64.Standard_Encoding()
	source := base64.Encoded("YWxsb2NhdGlvbg==")
	var destination [base64.DECODED_SIZE_MAXIMUM]byte
	var overlap [base64.ENCODED_GROUP_SIZE]byte
	var count base64.Decoded_Count
	var status base64.Decode_Status
	testify.Zero_Allocation(t, func() {
		count, status = base64.Decode_Into(destination[:], source, encoding)
	})
	testify.True(t, count > 0)
	testify.Equal_Values(t, base64.STATUS_OK, status)
	testify.Zero_Allocation(t, func() {
		count, status = base64.Decode_Into(destination[:], []byte("!!!!"), encoding)
	})
	testify.Equal(t, base64.Decoded_Count(0), count)
	testify.Equal_Values(t, base64.STATUS_INPUT_INVALID, status)
	testify.Zero_Allocation(t, func() {
		count, status = base64.Decode_Into(destination[:0], source, encoding)
	})
	testify.Equal(t, base64.Decoded_Count(0), count)
	testify.Equal_Values(t, base64.STATUS_OUTPUT_TOO_SMALL, status)
	testify.Zero_Allocation(t, func() {
		count, status = base64.Decode_Into(
			overlap[:base64.DECODED_GROUP_SIZE], overlap[:], encoding,
		)
	})
	testify.Equal(t, base64.Decoded_Count(0), count)
	testify.Equal_Values(t, base64.STATUS_STORAGE_INVALID, status)
	testify.Zero_Allocation(t, func() {
		count, status = base64.Decode_Into(destination[:], source, base64.Encoding{})
	})
	testify.Equal(t, base64.Decoded_Count(0), count)
	testify.Equal_Values(t, base64.STATUS_ENCODING_INVALID, status)
}

func test_strict_decode_allocation(t *testing.T) {
	strict, strict_status := base64.Strict_Encoding(base64.Standard_Encoding())
	testify.Equal_Values(t, base64.STATUS_OK, strict_status)
	canonical := base64.Encoded("WvLTlMrX9NpYDQlEIFlnDA==")
	noncanonical := base64.Encoded("WvLTlMrX9NpYDQlEIFlnDB==")
	var destination [base64.DECODED_SIZE_MAXIMUM]byte
	var count base64.Decoded_Count
	var status base64.Decode_Status
	testify.Zero_Allocation(t, func() {
		count, status = base64.Decode_Into(destination[:], canonical, strict)
	})
	testify.True(t, count > 0)
	testify.Equal_Values(t, base64.STATUS_OK, status)
	testify.Zero_Allocation(t, func() {
		count, status = base64.Decode_Into(destination[:], noncanonical, strict)
	})
	prefix_count := (len(noncanonical)/base64.ENCODED_GROUP_SIZE - 1) *
		base64.DECODED_GROUP_SIZE
	testify.Equal(t, base64.Decoded_Count(prefix_count), count)
	testify.Equal_Values(t, base64.STATUS_INPUT_INVALID, status)
}

func test_maximum_round_trip(t *testing.T) {
	encoding := base64.Standard_Encoding()
	var source [base64.SOURCE_SIZE_MAXIMUM]byte
	var encoded [base64.ENCODED_SIZE_MAXIMUM]byte
	var decoded [base64.DECODED_SIZE_MAXIMUM]byte
	for index := range source {
		source[index] = byte(index)
	}
	encoded_size, size_status := base64.Encoded_Size(
		encoding, base64.Source_Count(len(source)),
	)
	testify.Equal(t, base64.Encoded_Count(len(encoded)), encoded_size)
	testify.Equal_Values(t, base64.STATUS_OK, size_status)
	encoded_count, encode_status := base64.Encode_Into(encoded[:], source[:], encoding)
	testify.Equal(t, encoded_size, encoded_count)
	testify.Equal_Values(t, base64.STATUS_OK, encode_status)
	decoded_size, size_status := base64.Decoded_Size_Maximum(
		encoding, base64.Encoded_Input_Count(encoded_count),
	)
	testify.Equal(t, base64.Decoded_Count(len(decoded)), decoded_size)
	testify.Equal_Values(t, base64.STATUS_OK, size_status)
	decoded_count, decode_status := base64.Decode_Into(
		decoded[:], encoded[:encoded_count], encoding,
	)
	testify.Equal(t, base64.Decoded_Count(len(source)), decoded_count)
	testify.Equal_Values(t, base64.STATUS_OK, decode_status)
	testify.Equal(t, source[:], decoded[:decoded_count])
}

func test_size_domains(t *testing.T) {
	encoding := base64.Standard_Encoding()
	for _, size := range [...]int{0, 1, 2, base64.SOURCE_SIZE_MAXIMUM} {
		_, status := base64.Encoded_Size(encoding, base64.Source_Count(size))
		testify.Equal_Values(t, base64.STATUS_OK, status)
	}
	for _, size := range [...]int{0, 1, 2, base64.ENCODED_SIZE_MAXIMUM} {
		_, status := base64.Decoded_Size_Maximum(
			encoding, base64.Encoded_Input_Count(size),
		)
		testify.Equal_Values(t, base64.STATUS_OK, status)
	}
	raw := base64.Raw_Standard_Encoding()
	for _, one := range [...]struct {
		Encoded base64.Encoded_Input_Count
		Decoded base64.Decoded_Count
	}{
		{Encoded: base64.ENCODED_TAIL_SIZE_ONE, Decoded: 1},
		{Encoded: base64.ENCODED_TAIL_SIZE_FINAL, Decoded: 2},
	} {
		decoded_size, status := base64.Decoded_Size_Maximum(raw, one.Encoded)
		testify.Equal(t, one.Decoded, decoded_size)
		testify.Equal_Values(t, base64.STATUS_OK, status)
	}
}

func test_bound_refusals(t *testing.T) {
	encoding := base64.Standard_Encoding()
	var encoded [base64.ENCODED_SIZE_MAXIMUM]byte
	var decoded [base64.DECODED_SIZE_MAXIMUM]byte
	var source_oversized [base64.SOURCE_SIZE_MAXIMUM + 1]byte
	var encoded_oversized [base64.ENCODED_SIZE_MAXIMUM + 1]byte
	var decoded_oversized [base64.DECODED_SIZE_MAXIMUM + 1]byte
	testify.Panics(t, func() {
		base64.Encode_Into(encoded[:], source_oversized[:], encoding)
	})
	testify.Panics(t, func() {
		base64.Encode_Into(encoded_oversized[:], nil, encoding)
	})
	testify.Panics(t, func() {
		base64.Decode_Into(decoded[:], encoded_oversized[:], encoding)
	})
	testify.Panics(t, func() {
		base64.Decode_Into(decoded_oversized[:], nil, encoding)
	})
}

func test_storage_and_configuration_statuses(t *testing.T) {
	encoding := base64.Standard_Encoding()
	var encoded [base64.ENCODED_SIZE_MAXIMUM]byte
	var decoded [base64.DECODED_SIZE_MAXIMUM]byte
	var overlap [base64.ENCODED_SIZE_MAXIMUM]byte
	_, encode_status := base64.Encode_Into(
		overlap[:base64.ENCODED_GROUP_SIZE], overlap[:base64.DECODED_GROUP_SIZE], encoding,
	)
	testify.Equal_Values(t, base64.STATUS_STORAGE_INVALID, encode_status)
	_, decode_status := base64.Decode_Into(
		overlap[:base64.DECODED_GROUP_SIZE], overlap[:base64.ENCODED_GROUP_SIZE], encoding,
	)
	testify.Equal_Values(t, base64.STATUS_STORAGE_INVALID, decode_status)
	_, encode_status = base64.Encode_Into(encoded[:], nil, base64.Encoding{})
	testify.Equal_Values(t, base64.STATUS_ENCODING_INVALID, encode_status)
	_, decode_status = base64.Decode_Into(decoded[:], nil, base64.Encoding{})
	testify.Equal_Values(t, base64.STATUS_ENCODING_INVALID, decode_status)
	_, size_status := base64.Encoded_Size(base64.Encoding{}, 0)
	testify.Equal_Values(t, base64.STATUS_ENCODING_INVALID, size_status)
	_, size_status = base64.Decoded_Size_Maximum(base64.Encoding{}, 0)
	testify.Equal_Values(t, base64.STATUS_ENCODING_INVALID, size_status)
}

func assert_encode(t *testing.T, encoding base64.Encoding, source string, expected string) {
	t.Helper()
	var destination [base64.ENCODED_SIZE_MAXIMUM]byte
	required, size_status := base64.Encoded_Size(
		encoding, base64.Source_Count(len(source)),
	)
	testify.Equal_Values(t, base64.STATUS_OK, size_status)
	testify.Equal(t, base64.Encoded_Count(len(expected)), required)
	count, encode_status := base64.Encode_Into(
		destination[:int(required)], []byte(source), encoding,
	)
	testify.Equal_Values(t, base64.STATUS_OK, encode_status)
	testify.Equal(t, required, count)
	testify.Equal(t, expected, string(destination[:count]))
}

func assert_decode(t *testing.T, encoding base64.Encoding, source string, expected string) {
	t.Helper()
	var destination [base64.DECODED_SIZE_MAXIMUM]byte
	count, status := base64.Decode_Into(destination[:], []byte(source), encoding)
	testify.Equal_Values(t, base64.STATUS_OK, status)
	testify.Equal(t, base64.Decoded_Count(len(expected)), count)
	testify.Equal(t, expected, string(destination[:count]))
}
