package base64_test

import (
	"testing"

	"local/james-orcales/shared/encoding/base64"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/testify"
)

// Test_Configuration protects standard, URL, custom, and rejected alphabets.
func Test_Configuration(t *testing.T) {
	test_alphabet_size_validation(t)
	standard := standard_encoding(t)
	url := url_encoding(t)
	assert_encode(t, standard, base64.STANDARD_PADDING, "foobar", "Zm9vYmFy")
	assert_encode(t, url, base64.STANDARD_PADDING, "\xfb\xff", "-_8=")
	assert_encode(t, standard, base64.NO_PADDING, "f", "Zg")
	assert_encode(t, url, base64.NO_PADDING, "\xfb\xff", "-_8")

	alphabet := test_alphabet(TEST_STANDARD_ALPHABET)
	custom, configuration_status := base64.New_Encoding(alphabet)
	testify.Equal_Values(t, base64.STATUS_OK, configuration_status)
	assert_encode(t, custom, base64.NO_PADDING, "f", "Zg")
	assert_encode(t, custom, TEST_PADDING, "f", "Zg@@")

	duplicate := duplicate_alphabet(alphabet)
	_, configuration_status = base64.New_Encoding(duplicate)
	testify.Equal_Values(t, base64.STATUS_ALPHABET_INVALID, configuration_status)
	newline := base64.Alphabet(alphabet[:base64.ALPHABET_FINAL_INDEX] + "\n")
	_, configuration_status = base64.New_Encoding(newline)
	testify.Equal_Values(t, base64.STATUS_ALPHABET_INVALID, configuration_status)
	test_padding_refusals(t, custom, alphabet)
	for _, padding := range [...]base64.Padding{0, 1, 2, base64.PADDING_BYTE_MAXIMUM} {
		test_padding_round_trip(t, custom, padding)
	}
}

// Test_Encode preserves RFC 4648 wire forms with exact caller capacity.
func Test_Encode(t *testing.T) {
	encoding := standard_encoding(t)
	cases := [...]test_case{
		{Decoded: "", Encoded: ""},
		{Decoded: "f", Encoded: "Zg=="},
		{Decoded: "fo", Encoded: "Zm8="},
		{Decoded: "foo", Encoded: "Zm9v"},
		{Decoded: "\x00\x00\x01", Encoded: "AAAB"},
		{Decoded: "\x00\x00\x02", Encoded: "AAAC"},
		{Decoded: "\xff\xff\xff", Encoded: "////"},
		{Decoded: "foob", Encoded: "Zm9vYg=="},
		{Decoded: "fooba", Encoded: "Zm9vYmE="},
		{Decoded: "foobar", Encoded: "Zm9vYmFy"},
	}
	for _, one := range cases {
		assert_encode(t, encoding, base64.STANDARD_PADDING, one.Decoded, one.Encoded)
	}

	var destination [base64.ENCODED_SIZE_MAXIMUM]byte
	for index := range destination {
		destination[index] = TEST_SENTINEL
	}
	for _, size := range [...]int{0, 1, 2} {
		count, status := base64.Encode_Into(
			destination[:size], base64.Source("f"), encoding, base64.STANDARD_PADDING,
		)
		testify.Equal(t, base64.Encoded_Count(0), count)
		testify.Equal_Values(t, base64.STATUS_OUTPUT_TOO_SMALL, status)
	}
	for _, value := range destination {
		testify.Equal(t, TEST_SENTINEL, value)
	}
	count, status := base64.Encode_Into(
		destination[:1], nil, encoding, base64.STANDARD_PADDING,
	)
	testify.Equal(t, base64.Encoded_Count(0), count)
	testify.Equal_Values(t, base64.STATUS_OK, status)
}

// Test_Decode preserves padded, unpadded, newline, and malformed input behavior.
func Test_Decode(t *testing.T) {
	encoding := standard_encoding(t)
	cases := [...]test_case{
		{Decoded: "", Encoded: ""},
		{Decoded: "f", Encoded: "Zg=="},
		{Decoded: "fo", Encoded: "Zm8="},
		{Decoded: "foo", Encoded: "Zm9v"},
		{Decoded: "\x00\x00\x01", Encoded: "AAAB"},
		{Decoded: "\x00\x00\x02", Encoded: "AAAC"},
		{Decoded: "\xff\xff\xff", Encoded: "////"},
		{Decoded: "foobar", Encoded: "Zm9vYmFy"},
		{Decoded: "foobarfoobar", Encoded: "Zm9vYmFyZm9vYmFy"},
		{
			Decoded: "foobarfoobarfoobarfoobar",
			Encoded: "Zm9vYmFyZm9vYmFyZm9vYmFyZm9vYmFy",
		},
		{Decoded: "sure", Encoded: "c3V\ryZ\nQ=="},
	}
	for _, one := range cases {
		assert_decode(t, encoding, base64.STANDARD_PADDING, one.Encoded, one.Decoded)
	}
	assert_decode(t, encoding, base64.NO_PADDING, "Zg", "f")
	assert_decode(t, encoding, base64.NO_PADDING, "Zm8", "fo")
	assert_decode(t, encoding, base64.NO_PADDING, "AAB", "\x00\x00")
	assert_decode(t, encoding, base64.NO_PADDING, "AAC", "\x00\x00")
	assert_decode(t, encoding, base64.NO_PADDING, "///", "\xff\xff")

	invalid := [...]string{
		"A", "!!!!", "====", "x===", "=AAA", "A=AA", "AA=A", "AA==A",
		"AAA=AAAA", "AAAAA", "AAAAAA", "A=", "A==", "AA=", "AAAAAA=",
		"YWJjZA=====", "A!\n", "A=\n",
	}
	var destination [base64.DECODED_SIZE_MAXIMUM]byte
	for _, source := range invalid {
		_, status := base64.Decode_Into(
			destination[:], []byte(source), encoding, base64.STANDARD_PADDING,
		)
		testify.Equal_Values(t, base64.STATUS_INPUT_INVALID, status)
	}
	count, status := base64.Decode_Into(
		destination[:1], []byte("Zg=="), encoding, base64.STANDARD_PADDING,
	)
	testify.Equal(t, base64.Decoded_Count(1), count)
	testify.Equal_Values(t, base64.STATUS_OK, status)
	count, status = base64.Decode_Into(
		destination[:2], []byte("Zm8="), encoding, base64.STANDARD_PADDING,
	)
	testify.Equal(t, base64.Decoded_Count(2), count)
	testify.Equal_Values(t, base64.STATUS_OK, status)
	count, status = base64.Decode_Into(
		destination[:2], []byte("Zm8=\n"), encoding, base64.STANDARD_PADDING,
	)
	testify.Equal(t, base64.Decoded_Count(2), count)
	testify.Equal_Values(t, base64.STATUS_OK, status)
	count, status = base64.Decode_Into(
		destination[:0], []byte("Zg=="), encoding, base64.STANDARD_PADDING,
	)
	testify.Equal(t, base64.Decoded_Count(0), count)
	testify.Equal_Values(t, base64.STATUS_OUTPUT_TOO_SMALL, status)
	test_decode_symbol_domain(t, encoding, destination[:])
	test_decode_leaves_tail_untouched(t, encoding)
}

// Test_Strict_Decode rejects nonzero unused terminal bits only in strict mode.
func Test_Strict_Decode(t *testing.T) {
	standard := standard_encoding(t)
	var destination [base64.DECODED_SIZE_MAXIMUM]byte
	noncanonical := base64.Encoded("WvLTlMrX9NpYDQlEIFlnDB==")
	_, decode_status := base64.Decode_Strict_Into(
		destination[:], noncanonical, standard, base64.STANDARD_PADDING,
	)
	testify.Equal_Values(t, base64.STATUS_INPUT_INVALID, decode_status)
	_, decode_status = base64.Decode_Into(
		destination[:], noncanonical, standard, base64.STANDARD_PADDING,
	)
	testify.Equal_Values(t, base64.STATUS_OK, decode_status)
	canonical := base64.Encoded("WvLTlMrX9NpYDQlEIFlnDA==")
	_, decode_status = base64.Decode_Strict_Into(
		destination[:], canonical, standard, base64.STANDARD_PADDING,
	)
	testify.Equal_Values(t, base64.STATUS_OK, decode_status)
	_, decode_status = base64.Decode_Strict_Into(
		destination[:], canonical, base64.Encoding{}, base64.STANDARD_PADDING,
	)
	testify.Equal_Values(t, base64.STATUS_ENCODING_INVALID, decode_status)
	_, decode_status = base64.Decode_Strict_Into(
		destination[:], canonical, standard, base64.Padding(bits.INTEGER_16_MAXIMUM),
	)
	testify.Equal_Values(t, base64.STATUS_PADDING_INVALID, decode_status)
	test_strict_decode_shapes(t, standard)
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
	return base64.Alphabet(text)
}

func standard_encoding(t *testing.T) (encoding base64.Encoding) {
	t.Helper()
	encoding, status := base64.New_Encoding(base64.STANDARD_ALPHABET)
	testify.Equal_Values(t, base64.STATUS_OK, status)
	return encoding
}

func url_encoding(t *testing.T) (encoding base64.Encoding) {
	t.Helper()
	encoding, status := base64.New_Encoding(base64.URL_ALPHABET)
	testify.Equal_Values(t, base64.STATUS_OK, status)
	return encoding
}

func duplicate_alphabet(alphabet base64.Alphabet) (duplicate base64.Alphabet) {
	return base64.Alphabet(
		alphabet[:base64.ALPHABET_FINAL_INDEX] + alphabet[:1],
	)
}

func test_alphabet_size_validation(t *testing.T) {
	maximum := make(base64.Encoded, base64.ENCODED_SIZE_MAXIMUM)
	for _, alphabet := range [...]base64.Alphabet{
		"",
		"A",
		"AB",
		base64.Alphabet(string(maximum)),
	} {
		_, status := base64.New_Encoding(alphabet)
		testify.Equal_Values(t, base64.STATUS_ALPHABET_INVALID, status)
	}
}

// Padding reaches every operation, so every operation refuses a padding that is outside the
// wire-byte domain, is a line terminator, or is already an alphabet symbol.
func test_padding_refusals(t *testing.T, encoding base64.Encoding, alphabet base64.Alphabet) {
	t.Helper()
	var encoded [base64.ENCODED_SIZE_MAXIMUM]byte
	var decoded [base64.DECODED_SIZE_MAXIMUM]byte
	for _, padding := range [...]base64.Padding{
		base64.Padding(bits.INTEGER_16_MINIMUM),
		base64.Padding(bits.INTEGER_16_MAXIMUM),
		base64.Padding(alphabet[0]),
		base64.Padding('\r'),
		base64.Padding('\n'),
	} {
		_, size_status := base64.Encoded_Size(encoding, padding, 1)
		testify.Equal_Values(t, base64.STATUS_PADDING_INVALID, size_status)
		_, size_status = base64.Decoded_Size_Maximum(encoding, padding, 1)
		testify.Equal_Values(t, base64.STATUS_PADDING_INVALID, size_status)
		_, encode_status := base64.Encode_Into(
			encoded[:], base64.Source("f"), encoding, padding,
		)
		testify.Equal_Values(t, base64.STATUS_PADDING_INVALID, encode_status)
		_, decode_status := base64.Decode_Into(
			decoded[:], base64.Encoded("Zg"), encoding, padding,
		)
		testify.Equal_Values(t, base64.STATUS_PADDING_INVALID, decode_status)
		_, decode_status = base64.Decode_Strict_Into(
			decoded[:], base64.Encoded("Zg"), encoding, padding,
		)
		testify.Equal_Values(t, base64.STATUS_PADDING_INVALID, decode_status)
	}
}

func test_decode_symbol_domain(
	t *testing.T, encoding base64.Encoding, destination base64.Decoded,
) {
	for _, source_byte := range [...]byte{0, 1, 2, 255} {
		_, status := base64.Decode_Into(
			destination, base64.Encoded{source_byte}, encoding, base64.STANDARD_PADDING,
		)
		testify.Equal_Values(t, base64.STATUS_INPUT_INVALID, status)
	}
}

// A decode writes only the bytes it reports, so storage past the reported count keeps its
// sentinel.
func test_decode_leaves_tail_untouched(t *testing.T, encoding base64.Encoding) {
	t.Helper()
	var exact [base64.ENCODED_GROUP_SIZE + base64.ENCODED_GROUP_SIZE]byte
	for index := range exact {
		exact[index] = TEST_SENTINEL
	}
	count, status := base64.Decode_Into(
		exact[:], []byte("Zm9vYmFy"), encoding, base64.STANDARD_PADDING,
	)
	testify.Equal(
		t, base64.Decoded_Count(base64.DECODED_GROUP_SIZE+base64.DECODED_GROUP_SIZE),
		count,
	)
	testify.Equal_Values(t, base64.STATUS_OK, status)
	for _, value := range exact[count:] {
		testify.Equal(t, TEST_SENTINEL, value)
	}
}

func test_new_encoding_allocation(t *testing.T) {
	alphabet := test_alphabet(TEST_STANDARD_ALPHABET)
	duplicate := duplicate_alphabet(alphabet)
	var configured base64.Encoding
	var status base64.Configuration_Status
	testify.Zero_Allocation(t, func() {
		configured, status = base64.New_Encoding(alphabet)
	})
	testify.True(t, configured != base64.Encoding{})
	testify.Equal_Values(t, base64.STATUS_OK, status)
	testify.Zero_Allocation(t, func() {
		configured, status = base64.New_Encoding(duplicate)
	})
	testify.Equal(t, base64.Encoding{}, configured)
	testify.Equal_Values(t, base64.STATUS_ALPHABET_INVALID, status)
}

// A refused padding costs no allocation either: every operation reports it as a status.
func test_padding_allocation(t *testing.T) {
	encoding := standard_encoding(t)
	invalid := base64.Padding(bits.INTEGER_16_MINIMUM)
	var encoded [base64.ENCODED_SIZE_MAXIMUM]byte
	var size_status base64.Size_Status
	var encode_status base64.Encode_Status
	testify.Zero_Allocation(t, func() {
		_, size_status = base64.Encoded_Size(encoding, invalid, 1)
	})
	testify.Equal_Values(t, base64.STATUS_PADDING_INVALID, size_status)
	testify.Zero_Allocation(t, func() {
		_, size_status = base64.Decoded_Size_Maximum(encoding, invalid, 1)
	})
	testify.Equal_Values(t, base64.STATUS_PADDING_INVALID, size_status)
	testify.Zero_Allocation(t, func() {
		_, encode_status = base64.Encode_Into(
			encoded[:], base64.Source("f"), encoding, invalid,
		)
	})
	testify.Equal_Values(t, base64.STATUS_PADDING_INVALID, encode_status)
}

// Measures New_Encoding directly: a test helper's t.Helper call allocates on its own.
func test_factory_allocation(t *testing.T) {
	var configured base64.Encoding
	var status base64.Configuration_Status
	testify.Zero_Allocation(t, func() {
		configured, status = base64.New_Encoding(base64.STANDARD_ALPHABET)
	})
	testify.Equal_Values(t, base64.STATUS_OK, status)
	testify.True(t, configured != base64.Encoding{})
	testify.Zero_Allocation(t, func() {
		configured, status = base64.New_Encoding(base64.URL_ALPHABET)
	})
	testify.Equal_Values(t, base64.STATUS_OK, status)
	testify.True(t, configured != base64.Encoding{})
}

func test_size_allocation(t *testing.T) {
	encoding := standard_encoding(t)
	source_count := base64.Source_Count(len("allocation"))
	encoded_count := base64.Encoded_Input_Count(len("YWxsb2NhdGlvbg=="))
	var encoded_size base64.Encoded_Count
	var decoded_size base64.Decoded_Count
	var status base64.Size_Status
	testify.Zero_Allocation(t, func() {
		encoded_size, status = base64.Encoded_Size(
			encoding, base64.STANDARD_PADDING, source_count,
		)
	})
	testify.True(t, encoded_size > 0)
	testify.Equal_Values(t, base64.STATUS_OK, status)
	testify.Zero_Allocation(t, func() {
		encoded_size, status = base64.Encoded_Size(
			base64.Encoding{}, base64.STANDARD_PADDING, source_count,
		)
	})
	testify.Equal(t, base64.Encoded_Count(0), encoded_size)
	testify.Equal_Values(t, base64.STATUS_ENCODING_INVALID, status)
	testify.Zero_Allocation(t, func() {
		decoded_size, status = base64.Decoded_Size_Maximum(
			encoding, base64.STANDARD_PADDING, encoded_count,
		)
	})
	testify.True(t, decoded_size > 0)
	testify.Equal_Values(t, base64.STATUS_OK, status)
	testify.Zero_Allocation(t, func() {
		decoded_size, status = base64.Decoded_Size_Maximum(
			base64.Encoding{}, base64.STANDARD_PADDING, encoded_count,
		)
	})
	testify.Equal(t, base64.Decoded_Count(0), decoded_size)
	testify.Equal_Values(t, base64.STATUS_ENCODING_INVALID, status)
}

func test_encode_allocation(t *testing.T) {
	encoding := standard_encoding(t)
	source := base64.Source("allocation")
	var destination [base64.ENCODED_SIZE_MAXIMUM]byte
	var overlap [base64.ENCODED_GROUP_SIZE]byte
	var count base64.Encoded_Count
	var status base64.Encode_Status
	testify.Zero_Allocation(t, func() {
		count, status = base64.Encode_Into(
			destination[:], source, encoding, base64.STANDARD_PADDING,
		)
	})
	testify.True(t, count > 0)
	testify.Equal_Values(t, base64.STATUS_OK, status)
	testify.Zero_Allocation(t, func() {
		count, status = base64.Encode_Into(
			destination[:0], source, encoding, base64.STANDARD_PADDING,
		)
	})
	testify.Equal(t, base64.Encoded_Count(0), count)
	testify.Equal_Values(t, base64.STATUS_OUTPUT_TOO_SMALL, status)
	testify.Zero_Allocation(t, func() {
		count, status = base64.Encode_Into(
			overlap[:], overlap[:base64.DECODED_GROUP_SIZE], encoding,
			base64.STANDARD_PADDING,
		)
	})
	testify.Equal(t, base64.Encoded_Count(0), count)
	testify.Equal_Values(t, base64.STATUS_STORAGE_INVALID, status)
	testify.Zero_Allocation(t, func() {
		count, status = base64.Encode_Into(
			destination[:], source, base64.Encoding{}, base64.STANDARD_PADDING,
		)
	})
	testify.Equal(t, base64.Encoded_Count(0), count)
	testify.Equal_Values(t, base64.STATUS_ENCODING_INVALID, status)
}

func test_decode_allocation(t *testing.T) {
	encoding := standard_encoding(t)
	source := base64.Encoded("YWxsb2NhdGlvbg==")
	var destination [base64.DECODED_SIZE_MAXIMUM]byte
	var overlap [base64.ENCODED_GROUP_SIZE]byte
	var count base64.Decoded_Count
	var status base64.Decode_Status
	testify.Zero_Allocation(t, func() {
		count, status = base64.Decode_Into(
			destination[:], source, encoding, base64.STANDARD_PADDING,
		)
	})
	testify.True(t, count > 0)
	testify.Equal_Values(t, base64.STATUS_OK, status)
	testify.Zero_Allocation(t, func() {
		count, status = base64.Decode_Into(
			destination[:], []byte("!!!!"), encoding, base64.STANDARD_PADDING,
		)
	})
	testify.Equal(t, base64.Decoded_Count(0), count)
	testify.Equal_Values(t, base64.STATUS_INPUT_INVALID, status)
	testify.Zero_Allocation(t, func() {
		count, status = base64.Decode_Into(
			destination[:0], source, encoding, base64.STANDARD_PADDING,
		)
	})
	testify.Equal(t, base64.Decoded_Count(0), count)
	testify.Equal_Values(t, base64.STATUS_OUTPUT_TOO_SMALL, status)
	testify.Zero_Allocation(t, func() {
		count, status = base64.Decode_Into(
			overlap[:base64.DECODED_GROUP_SIZE], overlap[:], encoding,
			base64.STANDARD_PADDING,
		)
	})
	testify.Equal(t, base64.Decoded_Count(0), count)
	testify.Equal_Values(t, base64.STATUS_STORAGE_INVALID, status)
	testify.Zero_Allocation(t, func() {
		count, status = base64.Decode_Into(
			destination[:], source, base64.Encoding{}, base64.STANDARD_PADDING,
		)
	})
	testify.Equal(t, base64.Decoded_Count(0), count)
	testify.Equal_Values(t, base64.STATUS_ENCODING_INVALID, status)
	testify.Zero_Allocation(t, func() {
		count, status = base64.Decode_Into(
			destination[:], source, encoding, base64.Padding(bits.INTEGER_16_MAXIMUM),
		)
	})
	testify.Equal(t, base64.Decoded_Count(0), count)
	testify.Equal_Values(t, base64.STATUS_PADDING_INVALID, status)
}

func test_strict_decode_allocation(t *testing.T) {
	strict := standard_encoding(t)
	canonical := base64.Encoded("WvLTlMrX9NpYDQlEIFlnDA==")
	noncanonical := base64.Encoded("WvLTlMrX9NpYDQlEIFlnDB==")
	var destination [base64.DECODED_SIZE_MAXIMUM]byte
	var count base64.Decoded_Count
	var status base64.Decode_Status
	testify.Zero_Allocation(t, func() {
		count, status = base64.Decode_Strict_Into(
			destination[:], canonical, strict, base64.STANDARD_PADDING,
		)
	})
	testify.True(t, count > 0)
	testify.Equal_Values(t, base64.STATUS_OK, status)
	testify.Zero_Allocation(t, func() {
		count, status = base64.Decode_Strict_Into(
			destination[:], noncanonical, strict, base64.STANDARD_PADDING,
		)
	})
	prefix_count := (len(noncanonical)/base64.ENCODED_GROUP_SIZE - 1) *
		base64.DECODED_GROUP_SIZE
	testify.Equal(t, base64.Decoded_Count(prefix_count), count)
	testify.Equal_Values(t, base64.STATUS_INPUT_INVALID, status)
}

func test_maximum_round_trip(t *testing.T) {
	encoding := standard_encoding(t)
	var source [base64.SOURCE_SIZE_MAXIMUM]byte
	var encoded [base64.ENCODED_SIZE_MAXIMUM]byte
	var decoded [base64.DECODED_SIZE_MAXIMUM]byte
	for index := range source {
		source[index] = byte(index)
	}
	encoded_size, size_status := base64.Encoded_Size(
		encoding, base64.STANDARD_PADDING, base64.Source_Count(len(source)),
	)
	testify.Equal(t, base64.Encoded_Count(len(encoded)), encoded_size)
	testify.Equal_Values(t, base64.STATUS_OK, size_status)
	encoded_count, encode_status := base64.Encode_Into(
		encoded[:], source[:], encoding, base64.STANDARD_PADDING,
	)
	testify.Equal(t, encoded_size, encoded_count)
	testify.Equal_Values(t, base64.STATUS_OK, encode_status)
	decoded_size, size_status := base64.Decoded_Size_Maximum(
		encoding, base64.STANDARD_PADDING, base64.Encoded_Input_Count(encoded_count),
	)
	testify.Equal(t, base64.Decoded_Count(len(decoded)), decoded_size)
	testify.Equal_Values(t, base64.STATUS_OK, size_status)
	decoded_count, decode_status := base64.Decode_Into(
		decoded[:], encoded[:encoded_count], encoding, base64.STANDARD_PADDING,
	)
	testify.Equal(t, base64.Decoded_Count(len(source)), decoded_count)
	testify.Equal_Values(t, base64.STATUS_OK, decode_status)
	testify.Equal(t, source[:], decoded[:decoded_count])
	strict_count, strict_status := base64.Decode_Strict_Into(
		decoded[:], encoded[:encoded_count], encoding, base64.STANDARD_PADDING,
	)
	testify.Equal(t, base64.Decoded_Count(len(source)), strict_count)
	testify.Equal_Values(t, base64.STATUS_OK, strict_status)
}

func test_size_domains(t *testing.T) {
	encoding := standard_encoding(t)
	for _, size := range [...]int{0, 1, 2, base64.SOURCE_SIZE_MAXIMUM} {
		_, status := base64.Encoded_Size(
			encoding, base64.STANDARD_PADDING, base64.Source_Count(size),
		)
		testify.Equal_Values(t, base64.STATUS_OK, status)
	}
	for _, size := range [...]int{0, 1, 2, base64.ENCODED_SIZE_MAXIMUM} {
		_, status := base64.Decoded_Size_Maximum(
			encoding, base64.STANDARD_PADDING, base64.Encoded_Input_Count(size),
		)
		testify.Equal_Values(t, base64.STATUS_OK, status)
	}
	for _, one := range [...]struct {
		Encoded base64.Encoded_Input_Count
		Decoded base64.Decoded_Count
	}{
		{Encoded: base64.ENCODED_TAIL_SIZE_ONE, Decoded: 1},
		{Encoded: base64.ENCODED_TAIL_SIZE_FINAL, Decoded: 2},
	} {
		decoded_size, status := base64.Decoded_Size_Maximum(
			encoding, base64.NO_PADDING, one.Encoded,
		)
		testify.Equal(t, one.Decoded, decoded_size)
		testify.Equal_Values(t, base64.STATUS_OK, status)
	}
}

func test_bound_refusals(t *testing.T) {
	encoding := standard_encoding(t)
	var encoded [base64.ENCODED_SIZE_MAXIMUM]byte
	var decoded [base64.DECODED_SIZE_MAXIMUM]byte
	var source_oversized [base64.SOURCE_SIZE_MAXIMUM + 1]byte
	var encoded_oversized [base64.ENCODED_SIZE_MAXIMUM + 1]byte
	var decoded_oversized [base64.DECODED_SIZE_MAXIMUM + 1]byte
	testify.Panics(t, func() {
		base64.Encode_Into(
			encoded[:], source_oversized[:], encoding, base64.STANDARD_PADDING,
		)
	})
	testify.Panics(t, func() {
		base64.Encode_Into(encoded_oversized[:], nil, encoding, base64.STANDARD_PADDING)
	})
	testify.Panics(t, func() {
		base64.Decode_Into(
			decoded[:], encoded_oversized[:], encoding, base64.STANDARD_PADDING,
		)
	})
	testify.Panics(t, func() {
		base64.Decode_Into(decoded_oversized[:], nil, encoding, base64.STANDARD_PADDING)
	})
}

func test_storage_and_configuration_statuses(t *testing.T) {
	encoding := standard_encoding(t)
	var encoded [base64.ENCODED_SIZE_MAXIMUM]byte
	var decoded [base64.DECODED_SIZE_MAXIMUM]byte
	var overlap [base64.ENCODED_SIZE_MAXIMUM]byte
	_, encode_status := base64.Encode_Into(
		overlap[:base64.ENCODED_GROUP_SIZE], overlap[:base64.DECODED_GROUP_SIZE],
		encoding, base64.STANDARD_PADDING,
	)
	testify.Equal_Values(t, base64.STATUS_STORAGE_INVALID, encode_status)
	_, decode_status := base64.Decode_Into(
		overlap[:base64.DECODED_GROUP_SIZE], overlap[:base64.ENCODED_GROUP_SIZE],
		encoding, base64.STANDARD_PADDING,
	)
	testify.Equal_Values(t, base64.STATUS_STORAGE_INVALID, decode_status)
	_, encode_status = base64.Encode_Into(
		encoded[:], nil, base64.Encoding{}, base64.STANDARD_PADDING,
	)
	testify.Equal_Values(t, base64.STATUS_ENCODING_INVALID, encode_status)
	_, decode_status = base64.Decode_Into(
		decoded[:], nil, base64.Encoding{}, base64.STANDARD_PADDING,
	)
	testify.Equal_Values(t, base64.STATUS_ENCODING_INVALID, decode_status)
	_, size_status := base64.Encoded_Size(base64.Encoding{}, base64.STANDARD_PADDING, 0)
	testify.Equal_Values(t, base64.STATUS_ENCODING_INVALID, size_status)
	_, size_status = base64.Decoded_Size_Maximum(
		base64.Encoding{}, base64.STANDARD_PADDING, 0,
	)
	testify.Equal_Values(t, base64.STATUS_ENCODING_INVALID, size_status)
}

func assert_encode(
	t *testing.T, encoding base64.Encoding, padding base64.Padding,
	source string, expected string,
) {
	t.Helper()
	var destination [base64.ENCODED_SIZE_MAXIMUM]byte
	required, size_status := base64.Encoded_Size(
		encoding, padding, base64.Source_Count(len(source)),
	)
	testify.Equal_Values(t, base64.STATUS_OK, size_status)
	testify.Equal(t, base64.Encoded_Count(len(expected)), required)
	count, encode_status := base64.Encode_Into(
		destination[:int(required)], []byte(source), encoding, padding,
	)
	testify.Equal_Values(t, base64.STATUS_OK, encode_status)
	testify.Equal(t, required, count)
	testify.Equal(t, expected, string(destination[:count]))
}

func assert_decode(
	t *testing.T, encoding base64.Encoding, padding base64.Padding,
	source string, expected string,
) {
	t.Helper()
	var destination [base64.DECODED_SIZE_MAXIMUM]byte
	count, status := base64.Decode_Into(destination[:], []byte(source), encoding, padding)
	testify.Equal_Values(t, base64.STATUS_OK, status)
	testify.Equal(t, base64.Decoded_Count(len(expected)), count)
	testify.Equal(t, expected, string(destination[:count]))
}

// Every wire byte a padding can hold, the edges and the smallest values, must round-trip through
// each operation, not only the RFC padding.
func test_padding_round_trip(t *testing.T, encoding base64.Encoding, padding base64.Padding) {
	t.Helper()
	const UNPADDED = "Zm9vYmFyZg"
	const SOURCE = "foobarf"
	expected := []byte(UNPADDED + "==")
	for count := len(UNPADDED); count < len(expected); count++ {
		expected[count] = byte(padding)
	}
	assert_encode(t, encoding, padding, SOURCE, string(expected))
	assert_decode(t, encoding, padding, string(expected), SOURCE)
	maximum, size_status := base64.Decoded_Size_Maximum(
		encoding, padding, base64.Encoded_Input_Count(len(expected)),
	)
	testify.Equal_Values(t, base64.STATUS_OK, size_status)
	testify.True(t, int(maximum) >= len(SOURCE))
	var destination [base64.DECODED_SIZE_MAXIMUM]byte
	strict_count, strict_status := base64.Decode_Strict_Into(
		destination[:], expected, encoding, padding,
	)
	testify.Equal_Values(t, base64.STATUS_OK, strict_status)
	testify.Equal(t, base64.Decoded_Count(len(SOURCE)), strict_count)
	_, decode_status := base64.Decode_Into(
		destination[:], []byte(UNPADDED), encoding, padding,
	)
	testify.Equal_Values(t, base64.STATUS_INPUT_INVALID, decode_status)
}

// The strict entry point takes every source and destination shape the ordinary one takes: empty,
// one byte, two bytes, a raw unpadded encoding, a short destination, and overlapping storage.
func test_strict_decode_shapes(t *testing.T, standard base64.Encoding) {
	t.Helper()
	var destination [base64.DECODED_SIZE_MAXIMUM]byte
	count, status := base64.Decode_Strict_Into(
		destination[:0], base64.Encoded{}, standard, base64.STANDARD_PADDING,
	)
	testify.Equal(t, base64.Decoded_Count(0), count)
	testify.Equal_Values(t, base64.STATUS_OK, status)
	count, status = base64.Decode_Strict_Into(
		destination[:1], []byte("Zg=="), standard, base64.STANDARD_PADDING,
	)
	testify.Equal(t, base64.Decoded_Count(1), count)
	testify.Equal_Values(t, base64.STATUS_OK, status)
	count, status = base64.Decode_Strict_Into(
		destination[:2], []byte("Zm8="), standard, base64.STANDARD_PADDING,
	)
	testify.Equal(t, base64.Decoded_Count(2), count)
	testify.Equal_Values(t, base64.STATUS_OK, status)
	_, status = base64.Decode_Strict_Into(
		destination[:], []byte("Z"), standard, base64.STANDARD_PADDING,
	)
	testify.Equal_Values(t, base64.STATUS_INPUT_INVALID, status)
	_, status = base64.Decode_Strict_Into(
		destination[:], []byte("Zm"), standard, base64.STANDARD_PADDING,
	)
	testify.Equal_Values(t, base64.STATUS_INPUT_INVALID, status)
	_, status = base64.Decode_Strict_Into(
		destination[:1], []byte("Zm9v"), standard, base64.STANDARD_PADDING,
	)
	testify.Equal_Values(t, base64.STATUS_OUTPUT_TOO_SMALL, status)
	_, status = base64.Decode_Strict_Into(
		destination[:base64.DECODED_GROUP_SIZE], destination[:base64.ENCODED_GROUP_SIZE],
		standard, base64.STANDARD_PADDING,
	)
	testify.Equal_Values(t, base64.STATUS_STORAGE_INVALID, status)
	count, status = base64.Decode_Strict_Into(
		destination[:], []byte("Zm9v"), standard, base64.NO_PADDING,
	)
	testify.Equal(t, base64.Decoded_Count(3), count)
	testify.Equal_Values(t, base64.STATUS_OK, status)
}
