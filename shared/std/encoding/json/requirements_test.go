package json_test

import "testing"

import "local/james-orcales/shared/encoding/json"
import "local/james-orcales/shared/testify"

// Test_Validate protects the parser boundary from partial-value acceptance.
func Test_Validate(t *testing.T) {
  valid := [...]string{
    "null",
    "true",
    "false",
    "0",
    "-0",
    "12.5e-2",
    `"a\\b\"c\n\t\u20ac"`,
    `{"a":[1,true,null],"b":{}}`,
    " \n [ ] \r\t",
    `"€"`,
    "\"\xff\"",
  }
  for _, source := range valid {
    position, status := json.Validate(json.Encoded(source))
    testify.Equal(t, json.Position(0), position)
    testify.True(t, bool(status))
  }
  invalid := [...]struct {
    Source   string
    Position json.Position
  }{
    {"", 1},
    {"nul", 4},
    {"null true", 6},
    {"[1,]", 4},
    {`{"a" 1}`, 6},
    {`"\x"`, 3},
    {`"\u20ag"`, 7},
    {"\"\n\"", 2},
    {"01", 2},
    {"1.", 3},
    {"[", 2},
  }
  for _, one := range invalid {
    position, status := json.Validate(json.Encoded(one.Source))
    testify.Equal(t, one.Position, position)
    testify.False(t, bool(status))
  }
}

// Test_Compact protects string whitespace from structural whitespace removal.
func Test_Compact(t *testing.T) {
  source := json.Encoded(" { \"a\" : [ 1, true, null ], \"s\" : \" x \\n \" } \n")
  expected := `{"a":[1,true,null],"s":" x \n "}`
  var destination [json.OUTPUT_SIZE_MAXIMUM]byte
  required, position, size_status := json.Compact_Size(source)
  testify.Equal(t, json.Count(len(expected)), required)
  testify.Equal(t, json.Position(0), position)
  testify.True(t, bool(size_status))
  count, position, status := json.Compact_Into(destination[:required], source)
  testify.Equal(t, required, count)
  testify.Equal(t, json.Position(0), position)
  testify.Equal_Values(t, json.STATUS_OK, status)
  testify.Equal(t, expected, string(destination[:count]))

  destination[0] = TEST_SENTINEL
  count, _, status = json.Compact_Into(destination[:0], source)
  testify.Equal(t, json.Count(0), count)
  testify.Equal_Values(t, json.STATUS_OUTPUT_TOO_SMALL, status)
  testify.Equal(t, TEST_SENTINEL, destination[0])

  source = json.Encoded{'"', 0xff, '"'}
  count, position, status = json.Compact_Into(destination[:len(source)], source)
  testify.Equal(t, json.Count(len(source)), count)
  testify.Equal(t, json.Position(0), position)
  testify.Equal_Values(t, json.STATUS_OK, status)
  testify.Equal(t, byte(0xff), destination[1])
}

// Test_Indent protects caller formatting from hidden package policy.
func Test_Indent(t *testing.T) {
  source := json.Encoded(" \t" + `{"a":[1,2],"b":{}}` + "\n \t")
  prefix := json.Prefix(">")
  indent := json.Indent("  ")
  expected := "{\n>  \"a\": [\n>    1,\n>    2\n>  ],\n>  \"b\": {}\n>}\n \t"
  var destination [json.OUTPUT_SIZE_MAXIMUM]byte
  required, position, size_status := json.Indent_Size(source, prefix, indent)
  testify.Equal(t, json.Count(len(expected)), required)
  testify.Equal(t, json.Position(0), position)
  testify.Equal_Values(t, json.STATUS_OK, size_status)
  count, position, status := json.Indent_Into(
    destination[:required], source, prefix, indent,
  )
  testify.Equal(t, required, count)
  testify.Equal(t, json.Position(0), position)
  testify.Equal_Values(t, json.STATUS_OK, status)
  testify.Equal(t, expected, string(destination[:count]))

  source = json.Encoded{'"', 0xff, '"'}
  count, position, status = json.Indent_Into(
    destination[:len(source)], source, prefix, indent,
  )
  testify.Equal(t, json.Count(len(source)), count)
  testify.Equal(t, json.Position(0), position)
  testify.Equal_Values(t, json.STATUS_OK, status)
  testify.Equal(t, byte(0xff), destination[1])
}

// Test_Bounds protects all caller-controlled work limits.
func Test_Bounds(t *testing.T) {
  test_maximum_source(t)
  test_maximum_depth(t)
  test_refusals(t)
  test_domains(t)
  test_indent_domains(t)
  test_parser_domains(t)
  test_formatter_domains(t)
}

// Test_Decode checks every JSON value and all string escape forms.
func Test_Decode(t *testing.T) {
  scalars := []struct {
    Source string
    Kind   json.Kind
    Text   string
  }{
    {`null`, json.KIND_NULL, ""},
    {`true`, json.KIND_TRUE, ""},
    {`false`, json.KIND_FALSE, ""},
    {`0`, json.KIND_NUMBER, "0"},
    {`-0`, json.KIND_NUMBER, "-0"},
    {`1.234e+9999`, json.KIND_NUMBER, "1.234e+9999"},
    {`""`, json.KIND_STRING, ""},
    {`"ab"`, json.KIND_STRING, "ab"},
    {
      `"a\"\\\/\b\f\n\r\t\u0000\u20AC\uD834\uDD1E"`, json.KIND_STRING,
      "a\"\\/\b\f\n\r\t\x00€𝄞",
    },
    {`"\uD800x\uDC00\uD800\uD800\uDC00"`, json.KIND_STRING, "�x��𐀀"},
    {`"\u0001\u0002\uffff\uDBFF\uDFFF"`, json.KIND_STRING, "\x01\x02\uffff\U0010ffff"},
    {
      "\"\xff\xc0\xaf\xed\xa0\x80\xf4\x90\x80\x80€𐀀\"", json.KIND_STRING,
      "����������€𐀀",
    },
    {
      `"\ud800\u0000\ud800\uffff\ud800abcdef\ud800\n"`, json.KIND_STRING,
      "�\x00�\uffff�abcdef�\n",
    },
  }
  for _, one := range scalars {
    test_decode_values(t, one.Source, []json.Kind{one.Kind}, []string{one.Text})
  }
  test_decode_values(t, `[]`, []json.Kind{json.KIND_ARRAY, json.KIND_ARRAY_END},
    []string{"", ""})
  test_decode_values(t, `{}`, []json.Kind{json.KIND_OBJECT, json.KIND_OBJECT_END},
    []string{"", ""})
  test_decode_values(
    t,
    `[12]`,
    []json.Kind{json.KIND_ARRAY, json.KIND_NUMBER, json.KIND_ARRAY_END},
    []string{"", "12", ""},
  )
  test_decode_values(
    t,
    " \n\t"+`{"a":[null,true,false,-12.50e-2,"x"],"a":{"":[]}}`+"\r",
    []json.Kind{
      json.KIND_OBJECT,
      json.KIND_STRING,
      json.KIND_ARRAY,
      json.KIND_NULL,
      json.KIND_TRUE,
      json.KIND_FALSE,
      json.KIND_NUMBER,
      json.KIND_STRING,
      json.KIND_ARRAY_END,
      json.KIND_STRING,
      json.KIND_OBJECT,
      json.KIND_STRING,
      json.KIND_ARRAY,
      json.KIND_ARRAY_END,
      json.KIND_OBJECT_END,
      json.KIND_OBJECT_END,
    },
    []string{"", "a", "", "", "", "", "-12.50e-2", "x", "", "a", "", "", "", "", "", ""},
  )
}

// Test_Allocation protects the normal assertion path from hidden heap ownership.
func Test_Allocation(t *testing.T) {
  test_validate_allocation(t)
  test_compact_allocation(t)
  test_indent_allocation(t)
}

func test_validate_allocation(t *testing.T) {
  var maximum [json.ENCODED_SIZE_MAXIMUM]byte
  for index := range json.NESTING_DEPTH_MAXIMUM {
    maximum[index] = '['
    maximum[len(maximum)-1-index] = ']'
  }
  valid := json.Encoded(`{"a":[1,2]}`)
  invalid := json.Encoded(`[1,]`)
  var position json.Position
  var status json.Validate_Status
  testify.Zero_Allocation(t, func() {
    allocation_validate(&position, &status, valid)
  })
  testify.True(t, bool(status))
  testify.Zero_Allocation(t, func() {
    allocation_validate(&position, &status, invalid)
  })
  testify.False(t, bool(status))
  testify.True(t, position > 0)
  testify.Zero_Allocation(t, func() {
    allocation_validate(&position, &status, maximum[:])
  })
  testify.True(t, bool(status))
  testify.Equal(t, json.Position(0), position)
}

func test_compact_allocation(t *testing.T) {
  var destination [json.OUTPUT_SIZE_MAXIMUM]byte
  var maximum [json.ENCODED_SIZE_MAXIMUM]byte
  maximum[0] = '"'
  for index := 1; index < len(maximum)-1; index++ {
    maximum[index] = 'a'
  }
  maximum[len(maximum)-1] = '"'
  valid := json.Encoded(`{"a":[1,2]}`)
  invalid := json.Encoded(`[1,]`)
  overlap := json.Encoded(destination[:1])
  var count json.Count
  var position json.Position
  var size_status json.Compact_Size_Status
  var status json.Compact_Status
  testify.Zero_Allocation(t, func() {
    allocation_compact_size(&count, &position, &size_status, valid)
  })
  testify.True(t, bool(size_status))
  testify.Zero_Allocation(t, func() {
    allocation_compact_size(&count, &position, &size_status, invalid)
  })
  testify.False(t, bool(size_status))
  testify.Zero_Allocation(t, func() {
    allocation_compact(&count, &position, &status, destination[:], valid)
  })
  testify.Equal_Values(t, json.STATUS_OK, status)
  testify.Zero_Allocation(t, func() {
    allocation_compact(&count, &position, &status, destination[:], invalid)
  })
  testify.Equal_Values(t, json.STATUS_INPUT_INVALID, status)
  testify.Zero_Allocation(t, func() {
    allocation_compact(&count, &position, &status, destination[:0], valid)
  })
  testify.Equal_Values(t, json.STATUS_OUTPUT_TOO_SMALL, status)
  testify.Zero_Allocation(t, func() {
    allocation_compact(&count, &position, &status, destination[:1], overlap)
  })
  testify.Equal_Values(t, json.STATUS_STORAGE_INVALID, status)
  testify.Zero_Allocation(t, func() {
    allocation_compact(&count, &position, &status, destination[:], maximum[:])
  })
  testify.Equal_Values(t, json.STATUS_OK, status)
  testify.True(t, !(count > json.OUTPUT_SIZE_MAXIMUM))
  testify.True(t, !(position > json.POSITION_MAXIMUM))
}

func test_indent_allocation(t *testing.T) {
  var destination [json.OUTPUT_SIZE_MAXIMUM]byte
  var maximum_indent [json.INDENT_SIZE_MAXIMUM]byte
  var maximum [json.ENCODED_SIZE_MAXIMUM]byte
  maximum[0] = '"'
  for index := 1; index < len(maximum)-1; index++ {
    maximum[index] = 'a'
  }
  maximum[len(maximum)-1] = '"'
  valid := json.Encoded(`{"a":[1,2]}`)
  invalid := json.Encoded(`[1,]`)
  deep := json.Encoded("[[[[[[[[[[[[[[[[0]]]]]]]]]]]]]]]]")
  overlap := json.Encoded(destination[:1])
  prefix := json.Prefix(">")
  indent := json.Indent("  ")
  var count json.Count
  var position json.Position
  var size_status json.Indent_Size_Status
  var status json.Indent_Status
  testify.Zero_Allocation(t, func() {
    allocation_indent_size(&count, &position, &size_status, valid, prefix, indent)
  })
  testify.Equal_Values(t, json.STATUS_OK, size_status)
  testify.Zero_Allocation(t, func() {
    allocation_indent_size(&count, &position, &size_status, invalid, prefix, indent)
  })
  testify.Equal_Values(t, json.STATUS_INPUT_INVALID, size_status)
  testify.Zero_Allocation(t, func() {
    allocation_indent_size(&count, &position, &size_status, deep, nil, maximum_indent[:])
  })
  testify.Equal_Values(t, json.STATUS_OUTPUT_TOO_LARGE, size_status)
  testify.Zero_Allocation(t, func() {
    allocation_indent(&count, &position, &status, destination[:], valid, prefix, indent)
  })
  testify.Equal_Values(t, json.STATUS_OK, status)
  testify.Zero_Allocation(t, func() {
    allocation_indent(&count, &position, &status, destination[:], invalid, prefix, indent)
  })
  testify.Equal_Values(t, json.STATUS_INPUT_INVALID, status)
  testify.Zero_Allocation(t, func() {
    allocation_indent(&count, &position, &status, destination[:0], valid, prefix, indent)
  })
  testify.Equal_Values(t, json.STATUS_OUTPUT_TOO_SMALL, status)
  testify.Zero_Allocation(t, func() {
    allocation_indent(&count, &position, &status, destination[:1], overlap, nil, nil)
  })
  testify.Equal_Values(t, json.STATUS_STORAGE_INVALID, status)
  testify.Zero_Allocation(t, func() {
    allocation_indent(&count, &position, &status,
      destination[:], deep, nil, maximum_indent[:],
    )
  })
  testify.Equal_Values(t, json.STATUS_OUTPUT_TOO_LARGE, status)
  testify.Zero_Allocation(t, func() {
    allocation_indent(&count, &position, &status,
      destination[:], maximum[:], nil, nil,
    )
  })
  testify.Equal_Values(t, json.STATUS_OK, status)
  testify.True(t, !(count > json.OUTPUT_SIZE_MAXIMUM))
  testify.True(t, !(position > json.POSITION_MAXIMUM))
}

func allocation_validate(
  position *json.Position,
  status *json.Validate_Status,
  source json.Encoded,
) {
  *position, *status = json.Validate(source)
}

func allocation_compact_size(
  count *json.Count,
  position *json.Position,
  status *json.Compact_Size_Status,
  source json.Encoded,
) {
  *count, *position, *status = json.Compact_Size(source)
}

func allocation_compact(
  count *json.Count,
  position *json.Position,
  status *json.Compact_Status,
  destination json.Output,
  source json.Encoded,
) {
  *count, *position, *status = json.Compact_Into(destination, source)
}

func allocation_indent_size(
  count *json.Count,
  position *json.Position,
  status *json.Indent_Size_Status,
  source json.Encoded,
  prefix json.Prefix,
  indent json.Indent,
) {
  *count, *position, *status = json.Indent_Size(source, prefix, indent)
}

func allocation_indent(
  count *json.Count,
  position *json.Position,
  status *json.Indent_Status,
  destination json.Output,
  source json.Encoded,
  prefix json.Prefix,
  indent json.Indent,
) {
  *count, *position, *status = json.Indent_Into(destination, source, prefix, indent)
}

const TEST_SENTINEL byte = 0xa5

func test_maximum_source(t *testing.T) {
  var source [json.ENCODED_SIZE_MAXIMUM]byte
  source[0] = '"'
  for index := 1; index < len(source)-1; index++ {
    source[index] = 'a'
  }
  source[len(source)-1] = '"'
  var destination [json.OUTPUT_SIZE_MAXIMUM]byte
  count, _, status := json.Compact_Into(destination[:], source[:])
  testify.Equal(t, json.Count(len(source)), count)
  testify.Equal_Values(t, json.STATUS_OK, status)
}

func test_maximum_depth(t *testing.T) {
  var source [json.ENCODED_SIZE_MAXIMUM]byte
  for index := range json.NESTING_DEPTH_MAXIMUM {
    source[index] = '['
    source[len(source)-1-index] = ']'
  }
  position, status := json.Validate(source[:])
  testify.Equal(t, json.Position(0), position)
  testify.True(t, bool(status))
}

func test_refusals(t *testing.T) {
  var oversized_source [json.ENCODED_SIZE_MAXIMUM + 1]byte
  var oversized_output [json.OUTPUT_SIZE_MAXIMUM + 1]byte
  var oversized_prefix [json.PREFIX_SIZE_MAXIMUM + 1]byte
  var oversized_indent [json.INDENT_SIZE_MAXIMUM + 1]byte
  testify.Panics(t, func() { json.Validate(oversized_source[:]) })
  testify.Panics(t, func() { json.Compact_Into(oversized_output[:], nil) })
  testify.Panics(t, func() {
    json.Indent_Size(nil, oversized_prefix[:], nil)
  })
  testify.Panics(t, func() {
    json.Indent_Size(nil, nil, oversized_indent[:])
  })
  storage := json.Encoded(`[]`)
  _, _, status := json.Compact_Into(json.Output(storage), storage)
  testify.Equal_Values(t, json.STATUS_STORAGE_INVALID, status)
  _, _, indent_status := json.Indent_Into(json.Output(storage), storage, nil, nil)
  testify.Equal_Values(t, json.STATUS_STORAGE_INVALID, indent_status)
}

func test_domains(t *testing.T) {
  var destination [json.OUTPUT_SIZE_MAXIMUM]byte
  for _, source := range [...]json.Encoded{nil, {'0'}, {'[', ']'}} {
    json.Compact_Into(destination[:], source)
  }
  json.Compact_Into(destination[:], json.Encoded{'['})
  json.Compact_Into(destination[:1], json.Encoded("0"))
  invalid := json.Encoded(`{"a":}`)
  count, position, status := json.Compact_Into(destination[:], invalid)
  testify.Equal(t, json.Count(0), count)
  testify.True(t, position > 0)
  testify.Equal_Values(t, json.STATUS_INPUT_INVALID, status)

  var invalid_maximum [json.ENCODED_SIZE_MAXIMUM]byte
  invalid_maximum[0] = '"'
  for index := 1; index < len(invalid_maximum); index++ {
    invalid_maximum[index] = 'a'
  }
  position, validate_status := json.Validate(invalid_maximum[:])
  testify.Equal(t, json.Position(json.POSITION_MAXIMUM), position)
  testify.False(t, bool(validate_status))
  _, position, size_status := json.Compact_Size(invalid_maximum[:])
  testify.Equal(t, json.Position(json.POSITION_MAXIMUM), position)
  testify.False(t, bool(size_status))
  _, position, status = json.Compact_Into(destination[:], invalid_maximum[:])
  testify.Equal(t, json.Position(json.POSITION_MAXIMUM), position)
  testify.Equal_Values(t, json.STATUS_INPUT_INVALID, status)
  _, position, indent_size_status := json.Indent_Size(invalid_maximum[:], nil, nil)
  testify.Equal(t, json.Position(json.POSITION_MAXIMUM), position)
  testify.Equal_Values(t, json.STATUS_INPUT_INVALID, indent_size_status)
  _, position, indent_status := json.Indent_Into(
    destination[:], invalid_maximum[:], nil, nil,
  )
  testify.Equal(t, json.Position(json.POSITION_MAXIMUM), position)
  testify.Equal_Values(t, json.STATUS_INPUT_INVALID, indent_status)

  deep := json.Encoded("[[[[[[[[[[[[[[[[0]]]]]]]]]]]]]]]]")
  _, _, indent_status = json.Indent_Into(destination[:], deep, nil, destination[:])
  testify.Equal_Values(t, json.STATUS_STORAGE_INVALID, indent_status)
  _, _, indent_size_status = json.Indent_Size(deep, nil, destination[:])
  testify.Equal_Values(t, json.STATUS_OUTPUT_TOO_LARGE, indent_size_status)
  var maximum_indent [json.INDENT_SIZE_MAXIMUM]byte
  _, _, indent_status = json.Indent_Into(
    destination[:], deep, nil, maximum_indent[:],
  )
  testify.Equal_Values(t, json.STATUS_OUTPUT_TOO_LARGE, indent_status)
}

func test_indent_domains(t *testing.T) {
  var destination [json.OUTPUT_SIZE_MAXIMUM]byte
  for _, source := range [...]json.Encoded{nil, {'['}} {
    _, position, size_status := json.Indent_Size(source, nil, nil)
    testify.Equal_Values(t, json.STATUS_INPUT_INVALID, size_status)
    testify.True(t, position > 0)
    _, position, status := json.Indent_Into(destination[:], source, nil, nil)
    testify.Equal_Values(t, json.STATUS_INPUT_INVALID, status)
    testify.True(t, position > 0)
  }

  for _, source := range [...]json.Encoded{{'0'}, {'[', ']'}} {
    count, _, size_status := json.Indent_Size(source, nil, nil)
    testify.Equal(t, json.Count(len(source)), count)
    testify.Equal_Values(t, json.STATUS_OK, size_status)
    count, _, status := json.Indent_Into(destination[:len(source)], source, nil, nil)
    testify.Equal(t, json.Count(len(source)), count)
    testify.Equal_Values(t, json.STATUS_OK, status)
  }

  _, _, status := json.Indent_Into(destination[:0], json.Encoded("0"), nil, nil)
  testify.Equal_Values(t, json.STATUS_OUTPUT_TOO_SMALL, status)

  var unit_maximum [json.PREFIX_SIZE_MAXIMUM]byte
  for _, prefix := range [...]json.Prefix{
    nil, {'x'}, {'x', 'x'}, unit_maximum[:],
  } {
    json.Indent_Size(json.Encoded("0"), prefix, nil)
    json.Indent_Into(destination[:1], json.Encoded("0"), prefix, nil)
  }
  for _, indent := range [...]json.Indent{
    nil, {'x'}, {'x', 'x'}, unit_maximum[:],
  } {
    json.Indent_Size(json.Encoded("0"), nil, indent)
    json.Indent_Into(destination[:1], json.Encoded("0"), nil, indent)
  }

  var source_maximum [json.ENCODED_SIZE_MAXIMUM]byte
  source_maximum[0] = '"'
  for index := 1; index < len(source_maximum)-1; index++ {
    source_maximum[index] = 'a'
  }
  source_maximum[len(source_maximum)-1] = '"'
  count, _, size_status := json.Indent_Size(source_maximum[:], nil, nil)
  testify.Equal(t, json.Count(json.OUTPUT_SIZE_MAXIMUM), count)
  testify.Equal_Values(t, json.STATUS_OK, size_status)
  count, _, status = json.Indent_Into(destination[:], source_maximum[:], nil, nil)
  testify.Equal(t, json.Count(json.OUTPUT_SIZE_MAXIMUM), count)
  testify.Equal_Values(t, json.STATUS_OK, status)
}

func test_parser_domains(t *testing.T) {
  for _, source := range [...]json.Encoded{
    {'"'},
    {'"', '"'},
    {'t'},
    {'t', 'r'},
    {'"', '\\'},
    {'{', '}'},
    {'{', '"'},
    {'1'},
  } {
    json.Validate(source)
  }

  var number [json.ENCODED_SIZE_MAXIMUM]byte
  for index := range number {
    number[index] = '1'
  }
  _, status := json.Validate(number[:])
  testify.True(t, bool(status))

  var literal [json.ENCODED_SIZE_MAXIMUM]byte
  copy(literal[:], "true")
  for count := len("true"); count < len(literal); count++ {
    literal[count] = ' '
  }
  _, status = json.Validate(literal[:])
  testify.True(t, bool(status))

  var escaped [json.ENCODED_SIZE_MAXIMUM]byte
  escaped[0] = '"'
  escaped[1] = '\\'
  escaped[2] = 'n'
  for index := 3; index < len(escaped)-1; index++ {
    escaped[index] = 'a'
  }
  escaped[len(escaped)-1] = '"'
  _, status = json.Validate(escaped[:])
  testify.True(t, bool(status))

  var object [json.ENCODED_SIZE_MAXIMUM]byte
  copy(object[:], `{"":0}`)
  for count := len(`{"":0}`); count < len(object); count++ {
    object[count] = ' '
  }
  _, status = json.Validate(object[:])
  testify.True(t, bool(status))

  var destination [json.OUTPUT_SIZE_MAXIMUM]byte
  count, _, compact_status := json.Compact_Into(destination[:2], json.Encoded("[]"))
  testify.Equal(t, json.Count(2), count)
  testify.Equal_Values(t, json.STATUS_OK, compact_status)
}

func test_formatter_domains(t *testing.T) {
  var destination [json.OUTPUT_SIZE_MAXIMUM]byte
  var affix [json.FORMATTING_AFFIX_SIZE_MAXIMUM]byte
  for index := range affix {
    affix[index] = 'x'
  }
  for _, size := range [...]int{0, 1, 2, len(affix)} {
    prefix := json.Prefix(affix[:size])
    required, _, size_status := json.Indent_Size(json.Encoded("[0]"), prefix, nil)
    testify.Equal_Values(t, json.STATUS_OK, size_status)
    output := destination[:required]
    if size == len(affix) {
      output = destination[:]
    }
    _, _, status := json.Indent_Into(output, json.Encoded("[0]"), prefix, nil)
    testify.Equal_Values(t, json.STATUS_OK, status)
  }
  for _, size := range [...]int{0, 1, 2, len(affix)} {
    indent := json.Indent(affix[:size])
    required, _, size_status := json.Indent_Size(json.Encoded("[0]"), nil, indent)
    testify.Equal_Values(t, json.STATUS_OK, size_status)
    _, _, status := json.Indent_Into(
      destination[:required], json.Encoded("[0]"), nil, indent,
    )
    testify.Equal_Values(t, json.STATUS_OK, status)
  }

  var maximum [json.PREFIX_SIZE_MAXIMUM]byte
  _, _, status := json.Indent_Size(json.Encoded("[0]"), maximum[:], nil)
  testify.Equal_Values(t, json.STATUS_OUTPUT_TOO_LARGE, status)
  _, _, status = json.Indent_Size(json.Encoded("[0]"), nil, maximum[:])
  testify.Equal_Values(t, json.STATUS_OUTPUT_TOO_LARGE, status)
}

func test_decode_values(t *testing.T, source string, kinds []json.Kind, texts []string) {
  var tokens [json.TOKEN_COUNT_MAXIMUM]json.Token
  var output [json.OUTPUT_SIZE_MAXIMUM]byte
  required_tokens, required_bytes, position, size_status := json.Decode_Size(json.Encoded(source))
  testify.Equal_Values(t, json.STATUS_OK, size_status)
  testify.Equal(t, json.Position(0), position)
  count, written, position, status := json.Decode_Into(
    tokens[:required_tokens],
    output[:required_bytes],
    json.Encoded(source),
  )
  testify.Equal_Values(t, json.STATUS_OK, status)
  testify.Equal(t, json.Position(0), position)
  testify.Equal(t, required_tokens, count)
  testify.Equal(t, required_bytes, written)
  testify.Equal(t, len(kinds), int(count))
  offset := 0
  for index := range int(count) {
    token := tokens[index]
    text := string(output[token.Start:token.End])
    testify.Equal(t, kinds[index], token.Kind)
    testify.Equal(t, texts[index], text)
    testify.Equal(t, offset, int(token.Start))
    offset += len(texts[index])
    testify.Equal(t, offset, int(token.End))
  }
  testify.Equal(t, offset, int(written))
}

// Test_Decode_Refusals verifies atomic failure, error positions, and storage checks.
func Test_Decode_Refusals(t *testing.T) {
  var tokens [json.TOKEN_COUNT_MAXIMUM]json.Token
  var output [json.OUTPUT_SIZE_MAXIMUM]byte
  tokens[0].Kind = json.KIND_TRUE
  output[0] = 99
  for _, source := range []string{
    "",
    "[",
    "[1,]",
    `{"a":}`,
    `"\x"`,
    "true false",
    `"` + test_decode_repeat("x", json.ENCODED_SIZE_MAXIMUM-1),
  } {
    expected_position, _ := json.Validate(json.Encoded(source))
    count, written, position, status := json.Decode_Into(tokens[:], output[:], json.Encoded(source))
    testify.Equal_Values(t, json.STATUS_INPUT_INVALID, status)
    testify.Equal(t, expected_position, position)
    testify.Equal_Values(t, 0, count)
    testify.Equal_Values(t, 0, written)
    testify.Equal(t, json.KIND_TRUE, tokens[0].Kind)
    testify.Equal(t, byte(99), output[0])
  }
  for _, one := range []struct {
    Tokens int
    Bytes  int
  }{{0, 10}, {10, 0}} {
    _, _, _, status := json.Decode_Into(
      tokens[:one.Tokens],
      output[:one.Bytes],
      json.Encoded(`"x"`),
    )
    testify.Equal_Values(t, json.STATUS_OUTPUT_TOO_SMALL, status)
    testify.Equal(t, json.KIND_TRUE, tokens[0].Kind)
    testify.Equal(t, byte(99), output[0])
  }
  copy(output[:], `"x"`)
  _, _, _, status := json.Decode_Into(tokens[:], output[:], output[:3])
  testify.Equal_Values(t, json.STATUS_STORAGE_INVALID, status)
  testify.Equal(t, `"x"`, string(output[:3]))
  source := json.Encoded("\"" + test_decode_repeat("\xff", json.ENCODED_SIZE_MAXIMUM-2) + "\"")
  _, _, _, status = json.Decode_Into(tokens[:], output[:], source)
  testify.Equal_Values(t, json.STATUS_OUTPUT_TOO_LARGE, status)
  testify.Equal(t, json.KIND_TRUE, tokens[0].Kind)
  testify.Equal(t, `"x"`, string(output[:3]))
  content := test_decode_repeat("\xff", json.OUTPUT_SIZE_MAXIMUM/6+1)
  for _, text := range []string{
    `["` + content + `","` + content + `"]`,
    `["` + test_decode_repeat("\xff", json.OUTPUT_SIZE_MAXIMUM/3) + `a",0]`,
  } {
    before_tokens := tokens
    before_output := output
    count, written, position, expansion_status := json.Decode_Into(
      tokens[:],
      output[:],
      json.Encoded(text),
    )
    testify.Equal_Values(t, json.STATUS_OUTPUT_TOO_LARGE, expansion_status)
    testify.Equal_Values(t, 0, count)
    testify.Equal_Values(t, 0, written)
    testify.Equal_Values(t, 0, position)
    testify.Equal(t, before_tokens, tokens)
    testify.Equal(t, before_output, output)
  }
}

// Test_Decode_Storage protects source independence, output tails, and partial overlaps.
func Test_Decode_Storage(t *testing.T) {
  source := json.Encoded(`["hello",123]`)
  var tokens [json.TOKEN_COUNT_MAXIMUM]json.Token
  var output [json.OUTPUT_SIZE_MAXIMUM]byte
  tokens[4].Kind = json.KIND_TRUE
  output[8] = 99
  count, written, _, status := json.Decode_Into(tokens[:], output[:], source)
  testify.Equal_Values(t, json.STATUS_OK, status)
  testify.Equal_Values(t, 4, count)
  testify.Equal_Values(t, 8, written)
  clear(source)
  testify.Equal(t, "hello123", string(output[:written]))
  testify.Equal(t, json.KIND_TRUE, tokens[count].Kind)
  testify.Equal(t, byte(99), output[written])
  for _, start := range []int{0, 2, 4} {
    copy(output[2:], `"hi"`)
    before_tokens := tokens
    before_output := output
    _, _, _, status = json.Decode_Into(tokens[:], output[start:start+3], output[2:6])
    testify.Equal_Values(t, json.STATUS_STORAGE_INVALID, status)
    testify.Equal(t, before_tokens, tokens)
    testify.Equal(t, before_output, output)
  }
  copy(output[:], `"hi"`)
  _, _, _, status = json.Decode_Into(tokens[:], output[4:], output[:4])
  testify.Equal_Values(t, json.STATUS_OK, status)
  testify.Equal(t, "hi", string(output[4:6]))
}

// Test_Decode_Bounds checks maximum token depth, number size, string output, and caller bounds.
func Test_Decode_Bounds(t *testing.T) {
  kinds := make([]json.Kind, json.TOKEN_COUNT_MAXIMUM)
  for index := range kinds {
    kinds[index] = json.KIND_ARRAY
    if !(index < json.NESTING_DEPTH_MAXIMUM) {
      kinds[index] = json.KIND_ARRAY_END
    }
  }
  source := test_decode_repeat("[", json.NESTING_DEPTH_MAXIMUM) +
    test_decode_repeat("]", json.NESTING_DEPTH_MAXIMUM)
  test_decode_values(t, source, kinds, make([]string, len(kinds)))
  number := "1" + test_decode_repeat("0", json.ENCODED_SIZE_MAXIMUM-1)
  test_decode_values(t, number, []json.Kind{json.KIND_NUMBER}, []string{number})
  content := test_decode_repeat("x", json.ENCODED_SIZE_MAXIMUM-2)
  test_decode_values(t, `"`+content+`"`, []json.Kind{json.KIND_STRING}, []string{content})
  content = test_decode_repeat("x", json.ENCODED_SIZE_MAXIMUM-8)
  test_decode_values(
    t,
    `"\u0041`+content+`"`,
    []json.Kind{json.KIND_STRING},
    []string{"A" + content},
  )
  content = test_decode_repeat("\xff", json.OUTPUT_SIZE_MAXIMUM/3) + "a"
  expected := test_decode_repeat("�", json.OUTPUT_SIZE_MAXIMUM/3) + "a"
  test_decode_values(
    t,
    `["`+content+`",""]`,
    []json.Kind{json.KIND_ARRAY, json.KIND_STRING, json.KIND_STRING, json.KIND_ARRAY_END},
    []string{"", expected, "", ""},
  )
  testify.Panics(t, func() { json.Decode_Size(make(json.Encoded, json.ENCODED_SIZE_MAXIMUM+1)) })
  testify.Panics(t, func() {
    json.Decode_Into(nil, nil, make(json.Encoded, json.ENCODED_SIZE_MAXIMUM+1))
  })
  testify.Panics(t, func() {
    json.Decode_Into(make(json.Tokens, json.TOKEN_COUNT_MAXIMUM+1), nil, nil)
  })
  testify.Panics(t, func() {
    json.Decode_Into(nil, make(json.Output, json.OUTPUT_SIZE_MAXIMUM+1), nil)
  })
}

// Test_Decode_Allocation measures success and each refusal with assertions enabled.
func Test_Decode_Allocation(t *testing.T) {
  var tokens [json.TOKEN_COUNT_MAXIMUM]json.Token
  var output [json.OUTPUT_SIZE_MAXIMUM]byte
  source := json.Encoded(`{"a":["\uD834\uDD1E",123,true,false,null]}`)
  testify.Zero_Allocation(t, func() { json.Decode_Size(source) })
  testify.Zero_Allocation(t, func() { json.Decode_Into(tokens[:], output[:], source) })
  invalid := json.Encoded(`[1,]`)
  testify.Zero_Allocation(t, func() { json.Decode_Into(tokens[:], output[:], invalid) })
  testify.Zero_Allocation(t, func() { json.Decode_Into(nil, nil, source) })
  testify.Zero_Allocation(t, func() { json.Decode_Into(tokens[:], json.Output(source), source) })
  large := json.Encoded("\"" + test_decode_repeat("\xff", json.ENCODED_SIZE_MAXIMUM-2) + "\"")
  testify.Zero_Allocation(t, func() { json.Decode_Into(tokens[:], output[:], large) })
  nested := json.Encoded(
    test_decode_repeat("[", json.NESTING_DEPTH_MAXIMUM) +
      test_decode_repeat("]", json.NESTING_DEPTH_MAXIMUM),
  )
  testify.Zero_Allocation(t, func() { json.Decode_Into(tokens[:], output[:], nested) })
}

func test_decode_repeat(text string, count int) (repeated string) {
  storage := make([]byte, len(text)*count)
  for index := range count {
    copy(storage[index*len(text):], text)
  }
  return string(storage)
}

// Test_Raised_Bounds pins the larger public limits independently of package formulas.
func Test_Raised_Bounds(t *testing.T) {
  testify.Equal(t, 65536, json.ENCODED_SIZE_MAXIMUM)
  testify.Equal(t, 65536, json.OUTPUT_SIZE_MAXIMUM)
  testify.Equal(t, 65536, json.PREFIX_SIZE_MAXIMUM)
  testify.Equal(t, 65536, json.INDENT_SIZE_MAXIMUM)
  testify.Equal(t, 65536, json.TOKEN_COUNT_MAXIMUM)
  testify.Equal(t, 32768, json.NESTING_DEPTH_MAXIMUM)
  content := test_decode_repeat("a", 65534)
  test_decode_values(t, `"`+content+`"`, []json.Kind{json.KIND_STRING}, []string{content})
  content = test_decode_repeat("a", 4093) + "𐀀" + test_decode_repeat("b", 4096)
  test_decode_values(t, `"`+content+`"`, []json.Kind{json.KIND_STRING}, []string{content})
  test_raised_overlap(t)
}

func test_raised_overlap(t *testing.T) {
  var storage [json.OUTPUT_SIZE_MAXIMUM]byte
  var tokens [json.TOKEN_COUNT_MAXIMUM]json.Token
  source := json.Encoded(storage[len(storage)-4:])
  copy(source, `"hi"`)
  _, _, _, status := json.Decode_Into(tokens[:], storage[:], source)
  testify.Equal_Values(t, json.STATUS_STORAGE_INVALID, status)
  _, _, _, status = json.Decode_Into(tokens[:], storage[:len(storage)-4], source)
  testify.Equal_Values(t, json.STATUS_OK, status)
  testify.Equal(t, "hi", string(storage[:2]))
  storage[0] = '"'
  for index := 1; index < len(storage)-1; index++ {
    storage[index] = 'x'
  }
  storage[len(storage)-1] = '"'
  _, _, _, status = json.Decode_Into(tokens[:], storage[len(storage)-2:], storage[:])
  testify.Equal_Values(t, json.STATUS_STORAGE_INVALID, status)
}
