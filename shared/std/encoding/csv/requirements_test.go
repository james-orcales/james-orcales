package csv_test

import "testing"

import "local/james-orcales/shared/std/encoding/csv"
import "local/james-orcales/shared/std/math/bits"
import "local/james-orcales/shared/std/testify"
import "local/james-orcales/shared/std/unicode/utf8"

// Test_Configuration preserves standard and custom dialect validation.
func Test_Configuration(t *testing.T) {
  standard := standard_configuration(t)
  testify.Equal(t, csv.Delimiter_Character(','), csv.Configuration_Delimiter(standard))
  testify.Equal(t, csv.Comment_Character(0), csv.Configuration_Comment(standard))

  custom, status := csv.New_Configuration(
    csv.Configuration_Input{
      Delimiter:          '£',
      Comment:            '€',
      Fields_Per_Record:  csv.FIELDS_PER_RECORD_UNCHECKED,
      Lazy_Quotes:        true,
      Trim_Leading_Space: true,
      Use_CRLF:           true,
    },
  )
  testify.Equal_Values(t, csv.STATUS_OK, status)
  testify.Equal(t, csv.Delimiter_Character('£'), csv.Configuration_Delimiter(custom))
  testify.Equal(t, csv.Comment_Character('€'), csv.Configuration_Comment(custom))

  _, status = csv.New_Configuration(csv.Configuration_Input{Delimiter: '\n'})
  testify.Equal_Values(t, csv.STATUS_CONFIGURATION_INVALID, status)
  _, status = csv.New_Configuration(csv.Configuration_Input{
    Delimiter: ',', Comment: ',',
  })
  testify.Equal_Values(t, csv.STATUS_CONFIGURATION_INVALID, status)
  test_configuration_domains(t)
}

// Test_Encode preserves quoting, escaping, custom delimiters, and line endings.
func Test_Encode(t *testing.T) {
  var destination [csv.ENCODED_SIZE_MAXIMUM]byte
  fields := [...]csv.Field{
    {Value: csv.Field_Value("a")},
    {Value: csv.Field_Value("b,c")},
    {Value: csv.Field_Value("a\"b")},
    {Value: csv.Field_Value("line\nnext")},
    {Value: csv.Field_Value(" leading")},
    {Value: csv.Field_Value(`\.`)},
    {},
  }
  configuration := standard_configuration(t)
  required, size_status := csv.Encoded_Size(fields[:], configuration)
  testify.Equal_Values(t, csv.STATUS_OK, size_status)
  count, status := csv.Encode_Into(destination[:int(required)], fields[:], configuration)
  testify.Equal(t, required, count)
  testify.Equal_Values(t, csv.STATUS_OK, status)
  testify.Equal(t, ENCODED_STANDARD, string(destination[:count]))

  custom := custom_configuration(t)
  custom_fields := [...]csv.Field{
    {Value: csv.Field_Value("a")},
    {Value: csv.Field_Value("b£c")},
    {Value: csv.Field_Value("d\ne")},
  }
  required, size_status = csv.Encoded_Size(custom_fields[:], custom)
  testify.Equal_Values(t, csv.STATUS_OK, size_status)
  count, status = csv.Encode_Into(destination[:int(required)], custom_fields[:], custom)
  testify.Equal_Values(t, csv.STATUS_OK, status)
  testify.Equal(t, "a£\"b£c\"£\"d\r\ne\"\r\n", string(destination[:count]))

  destination[0] = TEST_SENTINEL
  count, status = csv.Encode_Into(destination[:0], fields[:], configuration)
  testify.Equal(t, csv.Encoded_Count(0), count)
  testify.Equal_Values(t, csv.STATUS_OUTPUT_TOO_SMALL, status)
  testify.Equal(t, TEST_SENTINEL, destination[0])
}

// Test_Decode preserves records, positions, comments, multiline fields, and parser statuses.
func Test_Decode(t *testing.T) {
  test_decode_standard(t)
  test_decode_custom(t)
  test_decode_field_count(t)
  test_decode_invalid(t)
}

// Test_Bounds reaches exact collection maxima and every refusal.
func Test_Bounds(t *testing.T) {
  test_maximum_encoded(t)
  test_maximum_decoded(t)
  test_maximum_fields(t)
  test_overlap(t)
  test_bound_refusals(t)
  test_public_domains(t)
}

// Test_Allocation measures every public status path with assertion tracking enabled.
func Test_Allocation(t *testing.T) {
  test_configuration_allocation(t)
  test_encode_allocation(t)
  test_decode_allocation(t)
}

const TEST_SENTINEL byte = 0xa5

const ENCODED_STANDARD = "a,\"b,c\",\"a\"\"b\",\"line\nnext\",\" leading\",\"\\.\",\n"

func standard_configuration(t *testing.T) (configuration csv.Configuration) {
  t.Helper()
  configuration, status := csv.New_Configuration(
    csv.Configuration_Input{
      Delimiter:         csv.STANDARD_DELIMITER,
      Fields_Per_Record: csv.FIELDS_PER_RECORD_INFERRED,
    },
  )
  testify.Equal_Values(t, csv.STATUS_OK, status)
  return configuration
}

func custom_configuration(t *testing.T) (configuration csv.Configuration) {
  t.Helper()
  configuration, status := csv.New_Configuration(
    csv.Configuration_Input{
      Delimiter:          '£',
      Comment:            '€',
      Fields_Per_Record:  csv.FIELDS_PER_RECORD_UNCHECKED,
      Lazy_Quotes:        true,
      Trim_Leading_Space: true,
      Use_CRLF:           true,
    },
  )
  testify.Equal_Values(t, csv.STATUS_OK, status)
  return configuration
}

func test_decoder(t *testing.T, configuration csv.Configuration) (decoder csv.Decoder) {
  t.Helper()
  status := csv.Decoder_Init(&decoder, configuration)
  testify.Equal_Values(t, csv.STATUS_OK, status)
  return decoder
}

func test_configuration_domains(t *testing.T) {
  t.Helper()
  tests := [...]struct {
    Input  csv.Configuration_Input
    Status csv.Configuration_Status
  }{
    {
      csv.Configuration_Input{Delimiter: csv.Delimiter(bits.INTEGER_32_MINIMUM)},
      csv.STATUS_CONFIGURATION_INVALID,
    },
    {
      csv.Configuration_Input{Delimiter: csv.Delimiter(bits.INTEGER_32_MAXIMUM)},
      csv.STATUS_CONFIGURATION_INVALID,
    },
    {csv.Configuration_Input{Delimiter: 0}, csv.STATUS_CONFIGURATION_INVALID},
    {csv.Configuration_Input{Delimiter: 1}, csv.STATUS_OK},
    {csv.Configuration_Input{Delimiter: 2}, csv.STATUS_OK},
    {csv.Configuration_Input{Delimiter: -1}, csv.STATUS_CONFIGURATION_INVALID},
    {
      csv.Configuration_Input{
        Delimiter: ',', Comment: csv.Comment(bits.INTEGER_32_MINIMUM),
      },
      csv.STATUS_CONFIGURATION_INVALID,
    },
    {
      csv.Configuration_Input{
        Delimiter: ',', Comment: csv.Comment(bits.INTEGER_32_MAXIMUM),
      },
      csv.STATUS_CONFIGURATION_INVALID,
    },
    {csv.Configuration_Input{Delimiter: ',', Comment: 1}, csv.STATUS_OK},
    {csv.Configuration_Input{Delimiter: ',', Comment: 2}, csv.STATUS_OK},
    {
      csv.Configuration_Input{Delimiter: ',', Comment: -1},
      csv.STATUS_CONFIGURATION_INVALID,
    },
    {csv.Configuration_Input{Delimiter: ',', Fields_Per_Record: 1}, csv.STATUS_OK},
    {csv.Configuration_Input{Delimiter: ',', Fields_Per_Record: 2}, csv.STATUS_OK},
    {
      csv.Configuration_Input{
        Delimiter: ',', Fields_Per_Record: csv.FIELD_COUNT_MAXIMUM,
      },
      csv.STATUS_OK,
    },
  }
  for _, one := range tests {
    _, status := csv.New_Configuration(one.Input)
    testify.Equal_Values(t, one.Status, status)
  }

  standard := standard_configuration(t)
  testify.Equal(t, csv.Comment_Character(0), csv.Configuration_Comment(standard))
  for _, character := range [...]int32{1, 2, utf8.DECODED_CHARACTER_MAXIMUM} {
    configuration, status := csv.New_Configuration(
      csv.Configuration_Input{
        Delimiter: csv.Delimiter(character),
      },
    )
    testify.Equal_Values(t, csv.STATUS_OK, status)
    testify.Equal(
      t, csv.Delimiter_Character(character),
      csv.Configuration_Delimiter(configuration),
    )

    configuration, status = csv.New_Configuration(
      csv.Configuration_Input{
        Delimiter: ',', Comment: csv.Comment(character),
      },
    )
    testify.Equal_Values(t, csv.STATUS_OK, status)
    testify.Equal(
      t, csv.Comment_Character(character),
      csv.Configuration_Comment(configuration),
    )
  }

  var invalid_decoder csv.Decoder
  decoder_status := csv.Decoder_Init(&invalid_decoder, csv.Configuration{})
  testify.Equal_Values(t, csv.STATUS_CONFIGURATION_INVALID, decoder_status)
  testify.Panics(t, func() { csv.Configuration_Delimiter(csv.Configuration{}) })
  testify.Panics(t, func() { csv.Configuration_Comment(csv.Configuration{}) })
  test_stored_domains(t)
}

// Every stored scalar has boundary branches in each namespace a configuration or
// decoder passes through, so one sweep drives every boundary through every entry.
func test_stored_domains(t *testing.T) {
  t.Helper()
  inputs := [...]csv.Configuration_Input{
    {Delimiter: 1, Fields_Per_Record: csv.FIELDS_PER_RECORD_UNCHECKED},
    {
      Delimiter:         2,
      Comment:           csv.Comment(utf8.DECODED_CHARACTER_MAXIMUM),
      Fields_Per_Record: 1,
      Lazy_Quotes:       true,
    },
    {
      Delimiter:          csv.Delimiter(utf8.DECODED_CHARACTER_MAXIMUM),
      Comment:            1,
      Fields_Per_Record:  2,
      Trim_Leading_Space: true,
    },
    {
      Delimiter:          ',',
      Comment:            2,
      Fields_Per_Record:  csv.FIELD_COUNT_MAXIMUM,
      Lazy_Quotes:        true,
      Trim_Leading_Space: true,
      Use_CRLF:           true,
    },
  }
  var configurations [len(inputs) + 1]csv.Configuration
  for index, input := range inputs {
    configuration, status := csv.New_Configuration(input)
    testify.Equal_Values(t, csv.STATUS_OK, status)
    configurations[index] = configuration
    csv.Configuration_Delimiter(configuration)
    csv.Configuration_Comment(configuration)
  }
  for _, configuration := range configurations {
    test_stored_domain_entries(t, configuration)
  }
}

func test_stored_domain_entries(t *testing.T, configuration csv.Configuration) {
  t.Helper()
  var encoded [csv.ENCODED_SIZE_MAXIMUM]byte
  var decoded [csv.DECODED_SIZE_MAXIMUM]byte
  var fields [csv.FIELD_COUNT_MAXIMUM]csv.Field
  two_fields := [...]csv.Field{{Value: csv.Field_Value("a")}, {Value: csv.Field_Value("b")}}
  csv.Encoded_Size(two_fields[:], configuration)
  csv.Encode_Into(encoded[:], two_fields[:], configuration)
  var decoder csv.Decoder
  csv.Decoder_Init(&decoder, configuration)
  csv.Decoder_Init(&decoder, configuration)
  csv.Decode_Record_Into(decoded[:], fields[:], csv.Encoded("a\n"), &decoder)
  csv.Decoder_Init(&decoder, csv.Configuration{})
  csv.Decode_Record_Into(decoded[:], fields[:], csv.Encoded("a"), &decoder)
  csv.Decode_Record_Into(decoded[:0], fields[:], csv.Encoded("a"), &decoder)
  csv.Decode_Record_Into(decoded[:], fields[:0], csv.Encoded("a"), &decoder)
  // A source made only of line endings drives a fresh cursor to its maximum, and
  // the decoder then carries that maximum into every later entry.
  csv.Decoder_Init(&decoder, configuration)
  var line_endings [csv.ENCODED_SIZE_MAXIMUM]byte
  for index := range line_endings {
    line_endings[index] = '\n'
  }
  csv.Decode_Record_Into(decoded[:], fields[:], line_endings[:], &decoder)
  csv.Decode_Record_Into(decoded[:], fields[:], nil, &decoder)
  csv.Decoder_Init(&decoder, csv.Configuration{})
  csv.Decoder_Init(&decoder, configuration)
}

func test_decode_standard(t *testing.T) {
  configuration := standard_configuration(t)
  decoder := test_decoder(t, configuration)
  var decoded [csv.DECODED_SIZE_MAXIMUM]byte
  var fields [csv.FIELD_COUNT_MAXIMUM]csv.Field
  source := csv.Encoded("a,\"b\nbb\",\"c\"\"d\"\r\ntail")
  count, consumed, position, status := csv.Decode_Record_Into(
    decoded[:],
    fields[:],
    source,
    &decoder,
  )
  testify.Equal_Values(t, csv.STATUS_OK, status)
  testify.Equal(t, csv.Field_Count(3), count)
  testify.Equal(t, csv.Consumed_Count(len(source)-len("tail")), consumed)
  testify.Equal(t, csv.Parse_Position{}, position)
  testify.Equal(t, csv.Field_Value("a"), fields[0].Value)
  testify.Equal(t, csv.Line(1), fields[0].Line)
  testify.Equal(t, csv.Column(1), fields[0].Column)
  testify.Equal(t, csv.Field_Value("b\nbb"), fields[1].Value)
  testify.Equal(t, csv.Line(1), fields[1].Line)
  testify.Equal(t, csv.Column(3), fields[1].Column)
  testify.Equal(t, csv.Field_Value("c\"d"), fields[2].Value)
}

func test_decode_custom(t *testing.T) {
  configuration := custom_configuration(t)
  decoder := test_decoder(t, configuration)
  var decoded [csv.DECODED_SIZE_MAXIMUM]byte
  var fields [csv.FIELD_COUNT_MAXIMUM]csv.Field
  source := csv.Encoded("\r\n€ ignored\r\n  a£\"b£c\"£d\"e\r\n")
  count, consumed, _, status := csv.Decode_Record_Into(
    decoded[:], fields[:], source, &decoder,
  )
  testify.Equal_Values(t, csv.STATUS_OK, status)
  testify.Equal(t, csv.Field_Count(3), count)
  testify.Equal(t, csv.Consumed_Count(len(source)), consumed)
  testify.Equal(t, csv.Field_Value("a"), fields[0].Value)
  testify.Equal(t, csv.Field_Value("b£c"), fields[1].Value)
  testify.Equal(t, csv.Field_Value("d\"e"), fields[2].Value)

  configuration, configuration_status := csv.New_Configuration(
    csv.Configuration_Input{
      Delimiter:         ',',
      Fields_Per_Record: csv.FIELDS_PER_RECORD_UNCHECKED,
      Lazy_Quotes:       true,
    },
  )
  testify.Equal_Values(t, csv.STATUS_OK, configuration_status)
  decoder = test_decoder(t, configuration)
  source = csv.Encoded("\"a\"x,b\r")
  count, consumed, _, status = csv.Decode_Record_Into(
    decoded[:], fields[:], source, &decoder,
  )
  testify.Equal_Values(t, csv.STATUS_OK, status)
  testify.Equal(t, csv.Field_Count(1), count)
  testify.Equal(t, csv.Consumed_Count(len(source)), consumed)
  testify.Equal(t, csv.Field_Value("a\"x,b"), fields[0].Value)
}

func test_decode_field_count(t *testing.T) {
  configuration := standard_configuration(t)
  decoder := test_decoder(t, configuration)
  var decoded [csv.DECODED_SIZE_MAXIMUM]byte
  var fields [csv.FIELD_COUNT_MAXIMUM]csv.Field
  first := csv.Encoded("a,b,c\n")
  count, _, _, status := csv.Decode_Record_Into(decoded[:], fields[:], first, &decoder)
  testify.Equal(t, csv.Field_Count(3), count)
  testify.Equal_Values(t, csv.STATUS_OK, status)
  second := csv.Encoded("d,e\n")
  count, _, position, status := csv.Decode_Record_Into(
    decoded[:], fields[:], second, &decoder,
  )
  testify.Equal(t, csv.Field_Count(2), count)
  testify.Equal_Values(t, csv.STATUS_FIELD_COUNT_INVALID, status)
  testify.Equal(t, csv.Line(2), position.Line)
  testify.Equal(t, csv.Column(1), position.Column)
  // The rejected record was consumed, so the decoder's line cursor must move past it.
  third := csv.Encoded("f,g,h\n")
  count, _, _, status = csv.Decode_Record_Into(decoded[:], fields[:], third, &decoder)
  testify.Equal(t, csv.Field_Count(3), count)
  testify.Equal_Values(t, csv.STATUS_OK, status)
  testify.Equal(t, csv.Line(3), fields[0].Line)
}

func test_decode_invalid(t *testing.T) {
  configuration := standard_configuration(t)
  decoder := test_decoder(t, configuration)
  var decoded [csv.DECODED_SIZE_MAXIMUM]byte
  var fields [csv.FIELD_COUNT_MAXIMUM]csv.Field
  count, _, position, status := csv.Decode_Record_Into(
    decoded[:],
    fields[:],
    csv.Encoded("a,ba\"d\n"),
    &decoder,
  )
  testify.Equal(t, csv.Field_Count(1), count)
  testify.Equal_Values(t, csv.STATUS_INPUT_INVALID, status)
  testify.Equal(t, csv.Line(1), position.Line)
  testify.Equal(t, csv.Column(5), position.Column)

  decoder = test_decoder(t, configuration)
  count, consumed, _, status := csv.Decode_Record_Into(
    decoded[:],
    fields[:],
    csv.Encoded("\n\r\n"),
    &decoder,
  )
  testify.Equal(t, csv.Field_Count(0), count)
  testify.Equal(t, csv.Consumed_Count(len("\n\r\n")), consumed)
  testify.Equal_Values(t, csv.STATUS_END, status)
}

func test_maximum_encoded(t *testing.T) {
  var value [csv.ENCODED_SIZE_MAXIMUM - 1]byte
  var destination [csv.ENCODED_SIZE_MAXIMUM]byte
  for index := range value {
    value[index] = 'a'
  }
  fields := [...]csv.Field{{Value: value[:]}}
  configuration := standard_configuration(t)
  required, size_status := csv.Encoded_Size(fields[:], configuration)
  testify.Equal(t, csv.Encoded_Count(len(destination)), required)
  testify.Equal_Values(t, csv.STATUS_OK, size_status)
  count, status := csv.Encode_Into(destination[:], fields[:], configuration)
  testify.Equal(t, required, count)
  testify.Equal_Values(t, csv.STATUS_OK, status)
}

func test_maximum_decoded(t *testing.T) {
  var source [csv.ENCODED_SIZE_MAXIMUM]byte
  var decoded [csv.DECODED_SIZE_MAXIMUM]byte
  var fields [csv.FIELD_COUNT_MAXIMUM]csv.Field
  for index := range source {
    source[index] = 'a'
  }
  decoder := test_decoder(t, standard_configuration(t))
  count, consumed, _, status := csv.Decode_Record_Into(
    decoded[:], fields[:], source[:], &decoder,
  )
  testify.Equal_Values(t, csv.STATUS_OK, status)
  testify.Equal(t, csv.Field_Count(1), count)
  testify.Equal(t, csv.Consumed_Count(len(source)), consumed)
  testify.Equal(t, csv.Field_Value(source[:]), fields[0].Value)
}

func test_maximum_fields(t *testing.T) {
  var source [csv.ENCODED_SIZE_MAXIMUM]byte
  var fields [csv.FIELD_COUNT_MAXIMUM]csv.Field
  for index := range source {
    source[index] = ','
  }
  decoder := test_decoder(t, standard_configuration(t))
  count, consumed, _, status := csv.Decode_Record_Into(nil, fields[:], source[:], &decoder)
  testify.Equal_Values(t, csv.STATUS_OK, status)
  testify.Equal(t, csv.Field_Count(len(fields)), count)
  testify.Equal(t, csv.Consumed_Count(len(source)), consumed)
  _, size_status := csv.Encoded_Size(fields[:], standard_configuration(t))
  testify.Equal_Values(t, csv.STATUS_RECORD_TOO_LARGE, size_status)
}

func test_overlap(t *testing.T) {
  var storage [csv.ENCODED_SIZE_MAXIMUM]byte
  copy(storage[:], "a,b\n")
  var fields [csv.FIELD_COUNT_MAXIMUM]csv.Field
  decoder := test_decoder(t, standard_configuration(t))
  _, _, _, decode_status := csv.Decode_Record_Into(
    storage[:],
    fields[:],
    storage[:len("a,b\n")],
    &decoder,
  )
  testify.Equal_Values(t, csv.STATUS_STORAGE_INVALID, decode_status)
  encode_fields := [...]csv.Field{{Value: storage[:1]}}
  _, encode_status := csv.Encode_Into(
    storage[:], encode_fields[:], standard_configuration(t),
  )
  testify.Equal_Values(t, csv.STATUS_STORAGE_INVALID, encode_status)
}

func test_bound_refusals(t *testing.T) {
  var encoded [csv.ENCODED_SIZE_MAXIMUM]byte
  var decoded [csv.DECODED_SIZE_MAXIMUM]byte
  var fields [csv.FIELD_COUNT_MAXIMUM]csv.Field
  var encoded_oversized [csv.ENCODED_SIZE_MAXIMUM + 1]byte
  var decoded_oversized [csv.DECODED_SIZE_MAXIMUM + 1]byte
  var fields_oversized [csv.FIELD_COUNT_MAXIMUM + 1]csv.Field
  var value_oversized [csv.FIELD_VALUE_SIZE_MAXIMUM + 1]byte
  decoder := test_decoder(t, standard_configuration(t))
  testify.Panics(t, func() {
    csv.Encode_Into(encoded_oversized[:], nil, standard_configuration(t))
  })
  testify.Panics(t, func() {
    csv.Encode_Into(encoded[:], fields_oversized[:], standard_configuration(t))
  })
  testify.Panics(t, func() {
    csv.Encode_Into(
      encoded[:], []csv.Field{{Value: value_oversized[:]}},
      standard_configuration(t),
    )
  })
  testify.Panics(t, func() {
    csv.Decode_Record_Into(decoded_oversized[:], fields[:], nil, &decoder)
  })
  testify.Panics(t, func() {
    csv.Decode_Record_Into(decoded[:], fields_oversized[:], nil, &decoder)
  })
  testify.Panics(t, func() {
    csv.Decode_Record_Into(decoded[:], fields[:], encoded_oversized[:], &decoder)
  })
}

func test_public_domains(t *testing.T) {
  t.Helper()
  test_encode_domains(t)
  test_decode_domains(t)
}

func test_encode_domains(t *testing.T) {
  t.Helper()
  configuration := standard_configuration(t)
  var encoded [csv.ENCODED_SIZE_MAXIMUM]byte
  count, size_status := csv.Encoded_Size(nil, configuration)
  testify.Equal(t, csv.Encoded_Count(1), count)
  testify.Equal_Values(t, csv.STATUS_OK, size_status)
  count, encode_status := csv.Encode_Into(encoded[:1], nil, configuration)
  testify.Equal(t, csv.Encoded_Count(1), count)
  testify.Equal_Values(t, csv.STATUS_OK, encode_status)
  empty_field := [...]csv.Field{{}}
  count, encode_status = csv.Encode_Into(encoded[:1], empty_field[:], configuration)
  testify.Equal(t, csv.Encoded_Count(1), count)
  testify.Equal_Values(t, csv.STATUS_OK, encode_status)
  one_field := [...]csv.Field{{Value: csv.Field_Value("a")}}
  count, encode_status = csv.Encode_Into(encoded[:2], one_field[:], configuration)
  testify.Equal(t, csv.Encoded_Count(2), count)
  testify.Equal_Values(t, csv.STATUS_OK, encode_status)
  two_byte_field := [...]csv.Field{{Value: csv.Field_Value("aa")}}
  _, size_status = csv.Encoded_Size(two_byte_field[:], configuration)
  testify.Equal_Values(t, csv.STATUS_OK, size_status)
  var quote_field_value [csv.FIELD_VALUE_SIZE_MAXIMUM]byte
  for index := range quote_field_value {
    quote_field_value[index] = '"'
  }
  quote_field := [...]csv.Field{{Value: quote_field_value[:]}}
  _, size_status = csv.Encoded_Size(quote_field[:], configuration)
  testify.Equal_Values(t, csv.STATUS_RECORD_TOO_LARGE, size_status)
  var maximum_fields [csv.FIELD_COUNT_MAXIMUM]csv.Field
  _, encode_status = csv.Encode_Into(encoded[:], maximum_fields[:], configuration)
  testify.Equal_Values(t, csv.STATUS_RECORD_TOO_LARGE, encode_status)
  count, encode_status = csv.Encode_Into(
    encoded[:],
    maximum_fields[:len(maximum_fields)-1],
    configuration,
  )
  testify.Equal(t, csv.Encoded_Count(csv.ENCODED_SIZE_MAXIMUM), count)
  testify.Equal_Values(t, csv.STATUS_OK, encode_status)
  invalid_delimiter := configuration
  invalid_delimiter.Delimiter = csv.Delimiter_Stored('\n')
  _, size_status = csv.Encoded_Size(nil, invalid_delimiter)
  testify.Equal_Values(t, csv.STATUS_CONFIGURATION_INVALID, size_status)
  invalid_comment := custom_configuration(t)
  invalid_comment.Comment = csv.Comment_Character('\n')
  _, size_status = csv.Encoded_Size(nil, invalid_comment)
  testify.Equal_Values(t, csv.STATUS_CONFIGURATION_INVALID, size_status)
}

func test_decode_domains(t *testing.T) {
  t.Helper()
  test_decode_slice_domains(t)
  test_decode_position_domains(t)
}

func test_decode_slice_domains(t *testing.T) {
  t.Helper()
  configuration := standard_configuration(t)
  var decoded [csv.DECODED_SIZE_MAXIMUM]byte
  var fields [csv.FIELD_COUNT_MAXIMUM]csv.Field
  decoder := test_decoder(t, configuration)
  field_count, consumed, _, decode_status := csv.Decode_Record_Into(
    decoded[:0],
    fields[:0],
    nil,
    &decoder,
  )
  testify.Equal(t, csv.Field_Count(0), field_count)
  testify.Equal(t, csv.Consumed_Count(0), consumed)
  testify.Equal_Values(t, csv.STATUS_END, decode_status)

  decoder = test_decoder(t, configuration)
  field_count, consumed, _, decode_status = csv.Decode_Record_Into(
    decoded[:1],
    fields[:1],
    csv.Encoded("a\n"),
    &decoder,
  )
  testify.Equal(t, csv.Field_Count(1), field_count)
  testify.Equal(t, csv.Consumed_Count(2), consumed)
  testify.Equal_Values(t, csv.STATUS_OK, decode_status)

  decoder = test_decoder(t, configuration)
  field_count, consumed, _, decode_status = csv.Decode_Record_Into(
    decoded[:1],
    fields[:1],
    csv.Encoded("a"),
    &decoder,
  )
  testify.Equal(t, csv.Field_Count(1), field_count)
  testify.Equal(t, csv.Consumed_Count(1), consumed)
  testify.Equal_Values(t, csv.STATUS_OK, decode_status)

  for _, size := range [...]int{0, 1, 2} {
    decoder = test_decoder(t, configuration)
    field_count, consumed, _, decode_status = csv.Decode_Record_Into(
      decoded[:size],
      fields[:1],
      csv.Encoded(`""`),
      &decoder,
    )
    testify.Equal(t, csv.Field_Count(1), field_count)
    testify.Equal(t, csv.Consumed_Count(2), consumed)
    testify.Equal_Values(t, csv.STATUS_OK, decode_status)
  }

  for _, size := range [...]int{1, 2} {
    decoder = test_decoder(t, configuration)
    _, _, _, decode_status = csv.Decode_Record_Into(
      decoded[:size],
      fields[:1],
      csv.Encoded("\"\n"),
      &decoder,
    )
    testify.Equal_Values(t, csv.STATUS_INPUT_INVALID, decode_status)
  }

  decoder = test_decoder(t, configuration)
  field_count, consumed, _, decode_status = csv.Decode_Record_Into(
    decoded[:0],
    fields[:1],
    csv.Encoded("\"\"\n"),
    &decoder,
  )
  testify.Equal(t, csv.Field_Count(1), field_count)
  testify.Equal(t, csv.Consumed_Count(3), consumed)
  testify.Equal_Values(t, csv.STATUS_OK, decode_status)

  decoder = test_decoder(t, configuration)
  field_count, _, _, decode_status = csv.Decode_Record_Into(
    decoded[:2],
    fields[:2],
    csv.Encoded("a,b\n"),
    &decoder,
  )
  testify.Equal(t, csv.Field_Count(2), field_count)
  testify.Equal_Values(t, csv.STATUS_OK, decode_status)
}

func test_decode_position_domains(t *testing.T) {
  t.Helper()
  configuration := standard_configuration(t)
  var decoded [csv.DECODED_SIZE_MAXIMUM]byte
  var fields [csv.FIELD_COUNT_MAXIMUM]csv.Field
  decoder := test_decoder(t, configuration)
  _, _, position, decode_status := csv.Decode_Record_Into(
    decoded[:],
    fields[:],
    csv.Encoded("\""),
    &decoder,
  )
  testify.Equal_Values(t, csv.STATUS_INPUT_INVALID, decode_status)
  testify.Equal(t, csv.Column(2), position.Column)

  var maximum_column_source [csv.ENCODED_SIZE_MAXIMUM]byte
  maximum_column_source[0] = '"'
  for index := 1; index < len(maximum_column_source); index++ {
    maximum_column_source[index] = 'a'
  }
  decoder = test_decoder(t, configuration)
  _, _, position, decode_status = csv.Decode_Record_Into(
    decoded[:],
    fields[:1],
    maximum_column_source[:],
    &decoder,
  )
  testify.Equal_Values(t, csv.STATUS_INPUT_INVALID, decode_status)
  testify.Equal(t, csv.Column(csv.COLUMN_MAXIMUM), position.Column)

  var maximum_line_source [csv.ENCODED_SIZE_MAXIMUM]byte
  for index := range maximum_line_source {
    maximum_line_source[index] = '\n'
  }
  maximum_line_source[len(maximum_line_source)-1] = '"'
  decoder = test_decoder(t, configuration)
  _, _, position, decode_status = csv.Decode_Record_Into(
    decoded[:],
    fields[:1],
    maximum_line_source[:],
    &decoder,
  )
  testify.Equal_Values(t, csv.STATUS_INPUT_INVALID, decode_status)
  testify.Equal(t, csv.Line(csv.LINE_MAXIMUM), position.Line)

  var maximum_quoted_source [csv.ENCODED_SIZE_MAXIMUM]byte
  for index := range maximum_quoted_source {
    maximum_quoted_source[index] = 'a'
  }
  maximum_quoted_source[0] = '"'
  maximum_quoted_source[1] = '\n'
  maximum_quoted_source[2] = '"'
  maximum_quoted_source[3] = '\n'
  decoder = test_decoder(t, configuration)
  field_count, consumed, _, decode_status := csv.Decode_Record_Into(
    decoded[:],
    fields[:1],
    maximum_quoted_source[:],
    &decoder,
  )
  testify.Equal(t, csv.Field_Count(1), field_count)
  testify.Equal(t, csv.Consumed_Count(4), consumed)
  testify.Equal_Values(t, csv.STATUS_OK, decode_status)
}

func test_configuration_allocation(t *testing.T) {
  input := csv.Configuration_Input{Delimiter: ',', Comment: '#'}
  invalid := csv.Configuration_Input{Delimiter: '\n'}
  var configuration csv.Configuration
  var status csv.Configuration_Status
  testify.Zero_Allocation(t, func() {
    allocation_configuration(&configuration, &status, input)
  })
  testify.Equal_Values(t, csv.STATUS_OK, status)
  testify.Zero_Allocation(t, func() {
    allocation_configuration(&configuration, &status, invalid)
  })
  testify.Equal_Values(t, csv.STATUS_CONFIGURATION_INVALID, status)
  standard := csv.Configuration_Input{
    Delimiter:         csv.STANDARD_DELIMITER,
    Fields_Per_Record: csv.FIELDS_PER_RECORD_INFERRED,
  }
  testify.Zero_Allocation(t, func() {
    allocation_configuration(&configuration, &status, standard)
  })
  testify.Equal_Values(t, csv.STATUS_OK, status)
  testify.True(t, csv.Configuration_Delimiter(configuration) == ',')
  var decoder csv.Decoder
  var decoder_status csv.Decoder_Status
  testify.Zero_Allocation(t, func() {
    allocation_decoder_init(&decoder_status, &decoder, configuration)
  })
  testify.Equal_Values(t, csv.STATUS_OK, decoder_status)
  testify.Zero_Allocation(t, func() {
    allocation_decoder_init(&decoder_status, &decoder, csv.Configuration{})
  })
  testify.Equal_Values(t, csv.STATUS_CONFIGURATION_INVALID, decoder_status)
}

func test_encode_allocation(t *testing.T) {
  var destination [csv.ENCODED_SIZE_MAXIMUM]byte
  var large [csv.FIELD_VALUE_SIZE_MAXIMUM]byte
  fields := [...]csv.Field{
    {Value: csv.Field_Value("a")},
    {Value: csv.Field_Value("b")},
  }
  large_fields := [...]csv.Field{{Value: large[:]}}
  overlap_fields := [...]csv.Field{{Value: destination[:1]}}
  configuration := standard_configuration(t)
  invalid := csv.Configuration{}
  var count csv.Encoded_Count
  var status csv.Encode_Status
  testify.Zero_Allocation(t, func() {
    allocation_encode(&count, &status, destination[:], fields[:], configuration)
  })
  testify.Equal_Values(t, csv.STATUS_OK, status)
  testify.Zero_Allocation(t, func() {
    allocation_encode(&count, &status, destination[:0], fields[:], configuration)
  })
  testify.Equal_Values(t, csv.STATUS_OUTPUT_TOO_SMALL, status)
  testify.Zero_Allocation(t, func() {
    allocation_encode(&count, &status, destination[:], large_fields[:], configuration)
  })
  testify.Equal_Values(t, csv.STATUS_RECORD_TOO_LARGE, status)
  testify.Zero_Allocation(t, func() {
    allocation_encode(&count, &status, destination[:], fields[:], invalid)
  })
  testify.Equal_Values(t, csv.STATUS_CONFIGURATION_INVALID, status)
  testify.Zero_Allocation(t, func() {
    allocation_encode(&count, &status, destination[:], overlap_fields[:], configuration)
  })
  testify.Equal_Values(t, csv.STATUS_STORAGE_INVALID, status)
  testify.True(t, !(count > csv.ENCODED_SIZE_MAXIMUM))
}

func test_decode_allocation(t *testing.T) {
  var decoded [csv.DECODED_SIZE_MAXIMUM]byte
  var fields [csv.FIELD_COUNT_MAXIMUM]csv.Field
  valid := csv.Encoded("a,b\n")
  invalid := csv.Encoded("a,ba\"d\n")
  empty := csv.Encoded("\n")
  configuration := standard_configuration(t)
  var count csv.Field_Count
  var consumed csv.Consumed_Count
  var position csv.Parse_Position
  var status csv.Decode_Status
  decoder := test_decoder(t, configuration)
  testify.Zero_Allocation(t, func() {
    allocation_decode(&count, &consumed, &position, &status,
      decoded[:], fields[:], valid, &decoder,
    )
  })
  testify.Equal_Values(t, csv.STATUS_OK, status)
  decoder = test_decoder(t, configuration)
  testify.Zero_Allocation(t, func() {
    allocation_decode(
      &count,
      &consumed,
      &position,
      &status,
      decoded[:],
      fields[:],
      invalid,
      &decoder,
    )
  })
  testify.Equal_Values(t, csv.STATUS_INPUT_INVALID, status)
  decoder = test_decoder(t, configuration)
  testify.Zero_Allocation(t, func() {
    allocation_decode(&count, &consumed, &position, &status,
      decoded[:], fields[:], empty, &decoder,
    )
  })
  testify.Equal_Values(t, csv.STATUS_END, status)
  decoder = test_decoder(t, configuration)
  testify.Zero_Allocation(t, func() {
    allocation_decode(
      &count,
      &consumed,
      &position,
      &status,
      decoded[:0],
      fields[:],
      valid,
      &decoder,
    )
  })
  testify.Equal_Values(t, csv.STATUS_OUTPUT_TOO_SMALL, status)
  decoder = test_decoder(t, configuration)
  testify.Zero_Allocation(t, func() {
    allocation_decode(
      &count,
      &consumed,
      &position,
      &status,
      decoded[:],
      fields[:0],
      valid,
      &decoder,
    )
  })
  testify.Equal_Values(t, csv.STATUS_FIELDS_TOO_SMALL, status)
  decoder = test_decoder(t, configuration)
  testify.Zero_Allocation(t, func() {
    allocation_decode(
      &count,
      &consumed,
      &position,
      &status,
      csv.Decoded(valid),
      fields[:],
      valid,
      &decoder,
    )
  })
  testify.Equal_Values(t, csv.STATUS_STORAGE_INVALID, status)
  var invalid_decoder csv.Decoder
  testify.Zero_Allocation(t, func() {
    allocation_decode(
      &count,
      &consumed,
      &position,
      &status,
      decoded[:],
      fields[:],
      valid,
      &invalid_decoder,
    )
  })
  testify.Equal_Values(t, csv.STATUS_CONFIGURATION_INVALID, status)
  testify.True(t, !(count > csv.FIELD_COUNT_MAXIMUM))
  testify.True(t, !(consumed > csv.ENCODED_SIZE_MAXIMUM))
  testify.True(t, !(position.Line > csv.LINE_MAXIMUM))
  test_decode_many_lines_allocation(t)
}

// Line numbers past 255 leave the runtime small-integer cache, so any boxing of
// decoder line state shows up only here.
func test_decode_many_lines_allocation(t *testing.T) {
  t.Helper()
  var decoded [csv.DECODED_SIZE_MAXIMUM]byte
  var fields [csv.FIELD_COUNT_MAXIMUM]csv.Field
  var many_lines [TEST_LINE_COUNT_LARGE + len("a\n")]byte
  for index := range many_lines {
    many_lines[index] = '\n'
  }
  many_lines[TEST_LINE_COUNT_LARGE] = 'a'
  decoder := test_decoder(t, standard_configuration(t))
  var status csv.Decode_Status
  testify.Zero_Allocation(t, func() {
    allocation_decode_status(&status,
      decoded[:], fields[:], many_lines[:], &decoder,
    )
  })
  testify.Equal_Values(t, csv.STATUS_OK, status)
  // The callback runs more than once on the same decoder, so only the lower bound holds.
  testify.True(t, fields[0].Line > csv.Line(TEST_LINE_COUNT_LARGE))
}

// TEST_LINE_COUNT_LARGE exceeds the 256-entry small-integer cache the runtime boxes for free.
const TEST_LINE_COUNT_LARGE = 300

func allocation_configuration(
  configuration *csv.Configuration,
  status *csv.Configuration_Status,
  input csv.Configuration_Input,
) {
  *configuration, *status = csv.New_Configuration(input)
}

func allocation_decoder_init(
  status *csv.Decoder_Status,
  decoder *csv.Decoder,
  configuration csv.Configuration,
) {
  *status = csv.Decoder_Init(decoder, configuration)
}

func allocation_encode(
  count *csv.Encoded_Count,
  status *csv.Encode_Status,
  destination csv.Encoded,
  fields csv.Fields,
  configuration csv.Configuration,
) {
  *count, *status = csv.Encode_Into(destination, fields, configuration)
}

func allocation_decode(
  count *csv.Field_Count,
  consumed *csv.Consumed_Count,
  position *csv.Parse_Position,
  status *csv.Decode_Status,
  destination csv.Decoded,
  fields csv.Fields,
  source csv.Encoded,
  decoder csv.Decoder_Handle,
) {
  *count, *consumed, *position, *status = csv.Decode_Record_Into(
    destination,
    fields,
    source,
    decoder,
  )
}

func allocation_decode_status(
  status *csv.Decode_Status,
  destination csv.Decoded,
  fields csv.Fields,
  source csv.Encoded,
  decoder csv.Decoder_Handle,
) {
  _, _, _, *status = csv.Decode_Record_Into(destination, fields, source, decoder)
}
