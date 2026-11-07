package base32_test

import "testing"

import "local/james-orcales/shared/std/encoding/base32"
import "local/james-orcales/shared/std/math/bits"
import "local/james-orcales/shared/std/testify"

// Test_Configuration protects custom alphabets without hidden pointer ownership.
func Test_Configuration(t *testing.T) {
  test_alphabet_size_validation(t)
  standard := standard_encoding(t)
  hexadecimal := hexadecimal_encoding(t)
  assert_encode(t, standard, base32.STANDARD_PADDING, "foobar", "MZXW6YTBOI======")
  assert_encode(t, hexadecimal, base32.STANDARD_PADDING, "foobar", "CPNMUOJ1E8======")

  alphabet := test_alphabet(TEST_STANDARD_ALPHABET)
  custom, configuration_status := base32.New_Encoding(alphabet)
  testify.Equal_Values(t, base32.STATUS_OK, configuration_status)
  assert_encode(t, custom, base32.NO_PADDING, "f", "MY")
  assert_encode(t, custom, base32.NO_PADDING, "foobar", "MZXW6YTBOI")
  assert_encode(t, custom, base32.STANDARD_PADDING, "f", "MY======")

  duplicate := duplicate_alphabet(alphabet)
  _, configuration_status = base32.New_Encoding(duplicate)
  testify.Equal_Values(t, base32.STATUS_ALPHABET_INVALID, configuration_status)
  newline := base32.Alphabet(
    alphabet[:base32.ALPHABET_FINAL_INDEX] + "\n",
  )
  _, configuration_status = base32.New_Encoding(newline)
  testify.Equal_Values(t, base32.STATUS_ALPHABET_INVALID, configuration_status)
  test_padding_refusals(t, custom, alphabet)
  for _, padding := range [...]base32.Padding{0, 1, 2, base32.PADDING_BYTE_MAXIMUM} {
    test_padding_round_trip(t, custom, padding)
  }
}

// Test_Encode preserves RFC 4648 wire forms with exact caller capacity.
func Test_Encode(t *testing.T) {
  encoding := standard_encoding(t)
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
    assert_encode(t, encoding, base32.STANDARD_PADDING, one.Decoded, one.Encoded)
  }

  var destination [base32.ENCODED_SIZE_MAXIMUM]byte
  for index := range destination {
    destination[index] = TEST_SENTINEL
  }
  required, size_status := base32.Encoded_Size(
    encoding,
    base32.STANDARD_PADDING,
    base32.Source_Count(len("f")),
  )
  testify.Equal_Values(t, base32.STATUS_OK, size_status)
  count, encode_status := base32.Encode_Into(
    destination[:int(required)-1],
    base32.Source("f"),
    encoding,
    base32.STANDARD_PADDING,
  )
  testify.Equal(t, base32.Encoded_Count(0), count)
  testify.Equal_Values(t, base32.STATUS_OUTPUT_TOO_SMALL, encode_status)
  for _, value := range destination {
    testify.Equal(t, TEST_SENTINEL, value)
  }
  for _, size := range [...]int{1, 2} {
    count, encode_status = base32.Encode_Into(
      destination[:size],
      base32.Source("f"),
      encoding,
      base32.STANDARD_PADDING,
    )
    testify.Equal(t, base32.Encoded_Count(0), count)
    testify.Equal_Values(t, base32.STATUS_OUTPUT_TOO_SMALL, encode_status)
  }
  count, encode_status = base32.Encode_Into(
    destination[:1], nil, encoding, base32.STANDARD_PADDING,
  )
  testify.Equal(t, base32.Encoded_Count(0), count)
  testify.Equal_Values(t, base32.STATUS_OK, encode_status)
}

// Test_Decode preserves padded, unpadded, newline, and malformed input behavior.
func Test_Decode(t *testing.T) {
  encoding := standard_encoding(t)
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
    assert_decode(t, encoding, base32.STANDARD_PADDING, one.Encoded, one.Decoded)
  }
  assert_decode(t, encoding, base32.NO_PADDING, "MZXW6YTBOI", "foobar")

  invalid := [...]string{
    "A",
    "!!!!",
    "x===",
    "AA=A====",
    "AAA=AAAA",
    "MMMMMMMMM",
    "MMMMMM",
    "A=",
    "AA=",
    "AA==",
    "AA===",
    "AAAA=",
    "AAAA==",
    "AAAAA=",
    "AAAAA==",
    "A=======",
    "AAA=====",
    "AAAAAA==",
    "========",
  }
  var destination [base32.DECODED_SIZE_MAXIMUM]byte
  for _, source := range invalid {
    _, decode_status := base32.Decode_Into(
      destination[:],
      base32.Encoded(source),
      encoding,
      base32.STANDARD_PADDING,
    )
    testify.Equal_Values(t, base32.STATUS_INPUT_INVALID, decode_status)
  }

  count, decode_status := base32.Decode_Into(
    destination[:1],
    base32.Encoded("MY======"),
    encoding,
    base32.STANDARD_PADDING,
  )
  testify.Equal(t, base32.Decoded_Count(1), count)
  testify.Equal_Values(t, base32.STATUS_OK, decode_status)
  count, decode_status = base32.Decode_Into(
    destination[:2],
    base32.Encoded("MZXQ===="),
    encoding,
    base32.STANDARD_PADDING,
  )
  testify.Equal(t, base32.Decoded_Count(2), count)
  testify.Equal_Values(t, base32.STATUS_OK, decode_status)
  count, decode_status = base32.Decode_Into(
    destination[:0],
    base32.Encoded("MY======"),
    encoding,
    base32.STANDARD_PADDING,
  )
  testify.Equal(t, base32.Decoded_Count(0), count)
  testify.Equal_Values(t, base32.STATUS_OUTPUT_TOO_SMALL, decode_status)
  test_decode_domain(t, encoding, destination[:])
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
  encoding := standard_encoding(t)
  var source [base32.SOURCE_SIZE_MAXIMUM]byte
  var encoded [base32.ENCODED_SIZE_MAXIMUM]byte
  var decoded [base32.DECODED_SIZE_MAXIMUM]byte
  for index := range source {
    source[index] = byte(index)
  }

  encoded_size, size_status := base32.Encoded_Size(
    encoding,
    base32.STANDARD_PADDING,
    base32.Source_Count(len(source)),
  )
  testify.Equal(t, base32.Encoded_Count(len(encoded)), encoded_size)
  testify.Equal_Values(t, base32.STATUS_OK, size_status)
  encoded_count, encode_status := base32.Encode_Into(
    encoded[:],
    source[:],
    encoding,
    base32.STANDARD_PADDING,
  )
  testify.Equal(t, encoded_size, encoded_count)
  testify.Equal_Values(t, base32.STATUS_OK, encode_status)
  decoded_size, size_status := base32.Decoded_Size_Maximum(
    encoding,
    base32.STANDARD_PADDING,
    base32.Encoded_Input_Count(encoded_count),
  )
  testify.Equal(t, base32.Decoded_Count(len(decoded)), decoded_size)
  testify.Equal_Values(t, base32.STATUS_OK, size_status)
  decoded_count, decode_status := base32.Decode_Into(
    decoded[:],
    encoded[:encoded_count],
    encoding,
    base32.STANDARD_PADDING,
  )
  testify.Equal(t, base32.Decoded_Count(len(source)), decoded_count)
  testify.Equal_Values(t, base32.STATUS_OK, decode_status)
  testify.Equal(t, source[:], decoded[:decoded_count])
}

func test_size_domains(t *testing.T) {
  encoding := standard_encoding(t)
  for _, size := range [...]int{0, 1, 2, base32.SOURCE_SIZE_MAXIMUM} {
    _, size_status := base32.Encoded_Size(
      encoding,
      base32.STANDARD_PADDING,
      base32.Source_Count(size),
    )
    testify.Equal_Values(t, base32.STATUS_OK, size_status)
  }
  for _, size := range [...]int{0, 1, 2, base32.ENCODED_SIZE_MAXIMUM} {
    _, size_status := base32.Decoded_Size_Maximum(
      encoding,
      base32.STANDARD_PADDING,
      base32.Encoded_Input_Count(size),
    )
    testify.Equal_Values(t, base32.STATUS_OK, size_status)
  }
  for _, one := range [...]struct {
    Encoded base32.Encoded_Input_Count
    Decoded base32.Decoded_Count
  }{
    {Encoded: base32.ENCODED_TAIL_SIZE_ONE, Decoded: 1},
    {Encoded: base32.ENCODED_TAIL_SIZE_TWO, Decoded: 2},
  } {
    decoded_size, size_status := base32.Decoded_Size_Maximum(
      encoding,
      base32.NO_PADDING,
      one.Encoded,
    )
    testify.Equal(t, one.Decoded, decoded_size)
    testify.Equal_Values(t, base32.STATUS_OK, size_status)
  }
}

func test_bound_refusals(t *testing.T) {
  encoding := standard_encoding(t)
  var encoded [base32.ENCODED_SIZE_MAXIMUM]byte
  var decoded [base32.DECODED_SIZE_MAXIMUM]byte
  var source_oversized [base32.SOURCE_SIZE_MAXIMUM + 1]byte
  var encoded_oversized [base32.ENCODED_SIZE_MAXIMUM + 1]byte
  var decoded_oversized [base32.DECODED_SIZE_MAXIMUM + 1]byte
  testify.Panics(t, func() {
    base32.Encode_Into(
      encoded[:], source_oversized[:], encoding, base32.STANDARD_PADDING,
    )
  })
  testify.Panics(t, func() {
    base32.Encode_Into(encoded_oversized[:], nil, encoding, base32.STANDARD_PADDING)
  })
  testify.Panics(t, func() {
    base32.Decode_Into(
      decoded[:], encoded_oversized[:], encoding, base32.STANDARD_PADDING,
    )
  })
  testify.Panics(t, func() {
    base32.Decode_Into(decoded_oversized[:], nil, encoding, base32.STANDARD_PADDING)
  })
}

func test_storage_and_configuration_statuses(t *testing.T) {
  encoding := standard_encoding(t)
  var encoded [base32.ENCODED_SIZE_MAXIMUM]byte
  var decoded [base32.DECODED_SIZE_MAXIMUM]byte
  var overlap [base32.ENCODED_SIZE_MAXIMUM]byte
  _, encode_status := base32.Encode_Into(
    overlap[:base32.ENCODED_GROUP_SIZE],
    overlap[:base32.DECODED_GROUP_SIZE],
    encoding,
    base32.STANDARD_PADDING,
  )
  testify.Equal_Values(t, base32.STATUS_STORAGE_INVALID, encode_status)
  _, decode_status := base32.Decode_Into(
    overlap[:base32.DECODED_GROUP_SIZE],
    overlap[:base32.ENCODED_GROUP_SIZE],
    encoding,
    base32.STANDARD_PADDING,
  )
  testify.Equal_Values(t, base32.STATUS_STORAGE_INVALID, decode_status)

  _, encode_status = base32.Encode_Into(
    encoded[:], nil, base32.Encoding{}, base32.STANDARD_PADDING,
  )
  testify.Equal_Values(t, base32.STATUS_ENCODING_INVALID, encode_status)
  _, decode_status = base32.Decode_Into(
    decoded[:], nil, base32.Encoding{}, base32.STANDARD_PADDING,
  )
  testify.Equal_Values(t, base32.STATUS_ENCODING_INVALID, decode_status)
  _, size_status := base32.Encoded_Size(base32.Encoding{}, base32.STANDARD_PADDING, 0)
  testify.Equal_Values(t, base32.STATUS_ENCODING_INVALID, size_status)
  _, size_status = base32.Decoded_Size_Maximum(
    base32.Encoding{}, base32.STANDARD_PADDING, 0,
  )
  testify.Equal_Values(t, base32.STATUS_ENCODING_INVALID, size_status)
}

const TEST_STANDARD_ALPHABET = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567"
const TEST_SENTINEL byte = 0xa5

type test_case struct {
  Decoded string
  Encoded string
}

func test_alphabet(text string) (alphabet base32.Alphabet) {
  return base32.Alphabet(text)
}

func duplicate_alphabet(alphabet base32.Alphabet) (duplicate base32.Alphabet) {
  return base32.Alphabet(
    alphabet[:base32.ALPHABET_FINAL_INDEX] + alphabet[:1],
  )
}

func test_alphabet_size_validation(t *testing.T) {
  maximum := make(base32.Encoded, base32.ENCODED_SIZE_MAXIMUM)
  for _, alphabet := range [...]base32.Alphabet{
    "",
    "A",
    "AB",
    base32.Alphabet(string(maximum)),
  } {
    _, status := base32.New_Encoding(alphabet)
    testify.Equal_Values(t, base32.STATUS_ALPHABET_INVALID, status)
  }
}

// Padding reaches every operation, so every operation refuses a padding that is outside the
// wire-byte domain, is a line terminator, or is already an alphabet symbol.
func test_padding_refusals(t *testing.T, encoding base32.Encoding, alphabet base32.Alphabet) {
  t.Helper()
  var encoded [base32.ENCODED_SIZE_MAXIMUM]byte
  var decoded [base32.DECODED_SIZE_MAXIMUM]byte
  for _, padding := range [...]base32.Padding{
    base32.Padding(bits.INTEGER_16_MINIMUM),
    base32.Padding(bits.INTEGER_16_MAXIMUM),
    base32.Padding(alphabet[0]),
    base32.Padding('\r'),
    base32.Padding('\n'),
  } {
    _, size_status := base32.Encoded_Size(encoding, padding, 1)
    testify.Equal_Values(t, base32.STATUS_PADDING_INVALID, size_status)
    _, size_status = base32.Decoded_Size_Maximum(encoding, padding, 1)
    testify.Equal_Values(t, base32.STATUS_PADDING_INVALID, size_status)
    _, encode_status := base32.Encode_Into(
      encoded[:], base32.Source("f"), encoding, padding,
    )
    testify.Equal_Values(t, base32.STATUS_PADDING_INVALID, encode_status)
    _, decode_status := base32.Decode_Into(
      decoded[:], base32.Encoded("MY"), encoding, padding,
    )
    testify.Equal_Values(t, base32.STATUS_PADDING_INVALID, decode_status)
  }
}

func test_decode_domain(
  t *testing.T, encoding base32.Encoding, destination base32.Decoded,
) {
  for _, source_byte := range [...]byte{0, 1, 2, 255} {
    _, status := base32.Decode_Into(
      destination,
      base32.Encoded{source_byte},
      encoding,
      base32.STANDARD_PADDING,
    )
    testify.Equal_Values(t, base32.STATUS_INPUT_INVALID, status)
  }
  for _, source := range [...]base32.Encoded{
    base32.Encoded("\nAAAAAAAA"),
    base32.Encoded("\nAAAAAAAB"),
    base32.Encoded("\nAAAAAAAC"),
    base32.Encoded("\n77777777"),
  } {
    count, status := base32.Decode_Into(
      destination, source, encoding, base32.STANDARD_PADDING,
    )
    testify.Equal(t, base32.Decoded_Count(5), count)
    testify.Equal_Values(t, base32.STATUS_OK, status)
  }
  for _, source := range [...]base32.Encoded{
    base32.Encoded("\nAAAAAAABA"),
    base32.Encoded("\nAAAAAAACA"),
    base32.Encoded("\n77777777A"),
  } {
    count, status := base32.Decode_Into(
      destination, source, encoding, base32.STANDARD_PADDING,
    )
    testify.Equal(t, base32.Decoded_Count(5), count)
    testify.Equal_Values(t, base32.STATUS_INPUT_INVALID, status)
  }
}

func allocation_encoding(
  encoding *base32.Encoding,
  status *base32.Configuration_Status,
  alphabet base32.Alphabet,
) {
  *encoding, *status = base32.New_Encoding(alphabet)
}

func allocation_encoded_size(
  count *base32.Encoded_Count,
  status *base32.Size_Status,
  encoding base32.Encoding,
  padding base32.Padding,
  source_count base32.Source_Count,
) {
  *count, *status = base32.Encoded_Size(encoding, padding, source_count)
}

func allocation_decoded_size(
  count *base32.Decoded_Count,
  status *base32.Size_Status,
  encoding base32.Encoding,
  padding base32.Padding,
  source_count base32.Encoded_Input_Count,
) {
  *count, *status = base32.Decoded_Size_Maximum(encoding, padding, source_count)
}

func allocation_encode(
  count *base32.Encoded_Count,
  status *base32.Encode_Status,
  destination base32.Encoded,
  source base32.Source,
  encoding base32.Encoding,
  padding base32.Padding,
) {
  *count, *status = base32.Encode_Into(destination, source, encoding, padding)
}

func allocation_decode(
  count *base32.Decoded_Count,
  status *base32.Decode_Status,
  destination base32.Decoded,
  source base32.Encoded,
  encoding base32.Encoding,
  padding base32.Padding,
) {
  *count, *status = base32.Decode_Into(destination, source, encoding, padding)
}

func test_new_encoding_allocation(t *testing.T) {
  alphabet := test_alphabet(TEST_STANDARD_ALPHABET)
  duplicate := duplicate_alphabet(alphabet)
  var configured base32.Encoding
  var status base32.Configuration_Status
  testify.Zero_Allocation(t, func() {
    allocation_encoding(&configured, &status, alphabet)
  })
  testify.True(t, configured != base32.Encoding{})
  testify.Equal_Values(t, base32.STATUS_OK, status)
  testify.Zero_Allocation(t, func() {
    allocation_encoding(&configured, &status, duplicate)
  })
  testify.Equal(t, base32.Encoding{}, configured)
  testify.Equal_Values(t, base32.STATUS_ALPHABET_INVALID, status)
}

// A refused padding costs no allocation either: every operation reports it as a status.
func test_padding_allocation(t *testing.T) {
  encoding := standard_encoding(t)
  invalid := base32.Padding(bits.INTEGER_16_MINIMUM)
  var encoded [base32.ENCODED_SIZE_MAXIMUM]byte
  var size_status base32.Size_Status
  var encode_status base32.Encode_Status
  var encoded_count base32.Encoded_Count
  var decoded_count base32.Decoded_Count
  testify.Zero_Allocation(t, func() {
    allocation_encoded_size(&encoded_count, &size_status, encoding, invalid, 1)
  })
  testify.Equal_Values(t, base32.STATUS_PADDING_INVALID, size_status)
  testify.Zero_Allocation(t, func() {
    allocation_decoded_size(&decoded_count, &size_status, encoding, invalid, 1)
  })
  testify.Equal_Values(t, base32.STATUS_PADDING_INVALID, size_status)
  testify.Zero_Allocation(t, func() {
    allocation_encode(
      &encoded_count,
      &encode_status,
      encoded[:],
      base32.Source("f"),
      encoding,
      invalid,
    )
  })
  testify.Equal_Values(t, base32.STATUS_PADDING_INVALID, encode_status)
}

// Measures New_Encoding directly: a test helper's t.Helper call allocates on its own.
func test_factory_allocation(t *testing.T) {
  var configured base32.Encoding
  var status base32.Configuration_Status
  testify.Zero_Allocation(t, func() {
    allocation_encoding(&configured, &status, base32.STANDARD_ALPHABET)
  })
  testify.Equal_Values(t, base32.STATUS_OK, status)
  testify.True(t, configured != base32.Encoding{})
  testify.Zero_Allocation(t, func() {
    allocation_encoding(&configured, &status, base32.HEXADECIMAL_ALPHABET)
  })
  testify.Equal_Values(t, base32.STATUS_OK, status)
  testify.True(t, configured != base32.Encoding{})
}

func test_size_allocation(t *testing.T) {
  encoding := standard_encoding(t)
  source_count := base32.Source_Count(len("allocation"))
  encoded_count := base32.Encoded_Input_Count(len("MFWHA3LPNRSXQ2LO"))
  var encoded_size base32.Encoded_Count
  var decoded_size base32.Decoded_Count
  var status base32.Size_Status
  testify.Zero_Allocation(t, func() {
    allocation_encoded_size(&encoded_size, &status,
      encoding, base32.STANDARD_PADDING, source_count,
    )
  })
  testify.True(t, encoded_size > 0)
  testify.Equal_Values(t, base32.STATUS_OK, status)
  testify.Zero_Allocation(t, func() {
    allocation_encoded_size(
      &encoded_size,
      &status,
      base32.Encoding{},
      base32.STANDARD_PADDING,
      source_count,
    )
  })
  testify.Equal(t, base32.Encoded_Count(0), encoded_size)
  testify.Equal_Values(t, base32.STATUS_ENCODING_INVALID, status)
  testify.Zero_Allocation(t, func() {
    allocation_decoded_size(
      &decoded_size,
      &status,
      encoding,
      base32.STANDARD_PADDING,
      encoded_count,
    )
  })
  testify.True(t, decoded_size > 0)
  testify.Equal_Values(t, base32.STATUS_OK, status)
  testify.Zero_Allocation(t, func() {
    allocation_decoded_size(
      &decoded_size,
      &status,
      base32.Encoding{},
      base32.STANDARD_PADDING,
      encoded_count,
    )
  })
  testify.Equal(t, base32.Decoded_Count(0), decoded_size)
  testify.Equal_Values(t, base32.STATUS_ENCODING_INVALID, status)
}

func test_encode_allocation(t *testing.T) {
  encoding := standard_encoding(t)
  source := base32.Source("allocation")
  var destination [base32.ENCODED_SIZE_MAXIMUM]byte
  var overlap [base32.ENCODED_GROUP_SIZE]byte
  var count base32.Encoded_Count
  var status base32.Encode_Status
  testify.Zero_Allocation(t, func() {
    allocation_encode(&count, &status,
      destination[:], source, encoding, base32.STANDARD_PADDING,
    )
  })
  testify.True(t, count > 0)
  testify.Equal_Values(t, base32.STATUS_OK, status)
  testify.Zero_Allocation(t, func() {
    allocation_encode(&count, &status,
      destination[:0], source, encoding, base32.STANDARD_PADDING,
    )
  })
  testify.Equal(t, base32.Encoded_Count(0), count)
  testify.Equal_Values(t, base32.STATUS_OUTPUT_TOO_SMALL, status)
  testify.Zero_Allocation(t, func() {
    allocation_encode(
      &count,
      &status,
      overlap[:],
      overlap[:base32.DECODED_GROUP_SIZE],
      encoding,
      base32.STANDARD_PADDING,
    )
  })
  testify.Equal(t, base32.Encoded_Count(0), count)
  testify.Equal_Values(t, base32.STATUS_STORAGE_INVALID, status)
  testify.Zero_Allocation(t, func() {
    allocation_encode(
      &count,
      &status,
      destination[:],
      source,
      base32.Encoding{},
      base32.STANDARD_PADDING,
    )
  })
  testify.Equal(t, base32.Encoded_Count(0), count)
  testify.Equal_Values(t, base32.STATUS_ENCODING_INVALID, status)
}

func test_decode_allocation(t *testing.T) {
  encoding := standard_encoding(t)
  source := base32.Encoded("MFWHA3LPNRSXQ2LO")
  malformed := base32.Encoded("!!!!!!!!")
  var destination [base32.DECODED_SIZE_MAXIMUM]byte
  var overlap [base32.ENCODED_GROUP_SIZE]byte
  var count base32.Decoded_Count
  var status base32.Decode_Status
  testify.Zero_Allocation(t, func() {
    allocation_decode(&count, &status,
      destination[:], source, encoding, base32.STANDARD_PADDING,
    )
  })
  testify.True(t, count > 0)
  testify.Equal_Values(t, base32.STATUS_OK, status)
  testify.Zero_Allocation(t, func() {
    allocation_decode(&count, &status,
      destination[:], malformed, encoding, base32.STANDARD_PADDING,
    )
  })
  testify.Equal(t, base32.Decoded_Count(0), count)
  testify.Equal_Values(t, base32.STATUS_INPUT_INVALID, status)
  testify.Zero_Allocation(t, func() {
    allocation_decode(&count, &status,
      destination[:0], source, encoding, base32.STANDARD_PADDING,
    )
  })
  testify.Equal(t, base32.Decoded_Count(0), count)
  testify.Equal_Values(t, base32.STATUS_OUTPUT_TOO_SMALL, status)
  testify.Zero_Allocation(t, func() {
    allocation_decode(
      &count,
      &status,
      overlap[:base32.DECODED_GROUP_SIZE],
      overlap[:],
      encoding,
      base32.STANDARD_PADDING,
    )
  })
  testify.Equal(t, base32.Decoded_Count(0), count)
  testify.Equal_Values(t, base32.STATUS_STORAGE_INVALID, status)
  testify.Zero_Allocation(t, func() {
    allocation_decode(
      &count,
      &status,
      destination[:],
      source,
      base32.Encoding{},
      base32.STANDARD_PADDING,
    )
  })
  testify.Equal(t, base32.Decoded_Count(0), count)
  testify.Equal_Values(t, base32.STATUS_ENCODING_INVALID, status)
  testify.Zero_Allocation(t, func() {
    allocation_decode(
      &count,
      &status,
      destination[:],
      source,
      encoding,
      base32.Padding(bits.INTEGER_16_MAXIMUM),
    )
  })
  testify.Equal(t, base32.Decoded_Count(0), count)
  testify.Equal_Values(t, base32.STATUS_PADDING_INVALID, status)
}

func assert_encode(
  t *testing.T,
  encoding base32.Encoding,
  padding base32.Padding,
  source string,
  expected string,
) {
  t.Helper()
  var destination [base32.ENCODED_SIZE_MAXIMUM]byte
  required, size_status := base32.Encoded_Size(
    encoding, padding, base32.Source_Count(len(source)),
  )
  testify.Equal_Values(t, base32.STATUS_OK, size_status)
  testify.Equal(t, base32.Encoded_Count(len(expected)), required)
  count, encode_status := base32.Encode_Into(
    destination[:int(required)],
    []byte(source),
    encoding,
    padding,
  )
  testify.Equal_Values(t, base32.STATUS_OK, encode_status)
  testify.Equal(t, required, count)
  testify.Equal(t, expected, string(destination[:count]))
}

func assert_decode(
  t *testing.T,
  encoding base32.Encoding,
  padding base32.Padding,
  source string,
  expected string,
) {
  t.Helper()
  var destination [base32.DECODED_SIZE_MAXIMUM]byte
  count, status := base32.Decode_Into(destination[:], []byte(source), encoding, padding)
  testify.Equal_Values(t, base32.STATUS_OK, status)
  testify.Equal(t, base32.Decoded_Count(len(expected)), count)
  testify.Equal(t, expected, string(destination[:count]))
}

// Every wire byte a padding can hold, the edges and the smallest values, must round-trip
// through each operation, not only the RFC padding.
func test_padding_round_trip(t *testing.T, encoding base32.Encoding, padding base32.Padding) {
  t.Helper()
  const UNPADDED = "MZXW6YTBOI"
  expected := []byte(UNPADDED + "======")
  for count := len(UNPADDED); count < len(expected); count++ {
    expected[count] = byte(padding)
  }
  assert_encode(t, encoding, padding, "foobar", string(expected))
  assert_decode(t, encoding, padding, string(expected), "foobar")
  maximum, size_status := base32.Decoded_Size_Maximum(
    encoding,
    padding,
    base32.Encoded_Input_Count(len(expected)),
  )
  testify.Equal_Values(t, base32.STATUS_OK, size_status)
  testify.True(t, !(int(maximum) < len("foobar")))
  var destination [base32.DECODED_SIZE_MAXIMUM]byte
  _, decode_status := base32.Decode_Into(
    destination[:], []byte(UNPADDED), encoding, padding,
  )
  testify.Equal_Values(t, base32.STATUS_INPUT_INVALID, decode_status)
}

func standard_encoding(t *testing.T) (encoding base32.Encoding) {
  t.Helper()
  encoding, status := base32.New_Encoding(base32.STANDARD_ALPHABET)
  testify.Equal_Values(t, base32.STATUS_OK, status)
  return encoding
}

func hexadecimal_encoding(t *testing.T) (encoding base32.Encoding) {
  t.Helper()
  encoding, status := base32.New_Encoding(base32.HEXADECIMAL_ALPHABET)
  testify.Equal_Values(t, base32.STATUS_OK, status)
  return encoding
}
