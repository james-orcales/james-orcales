package json_test

import (
	"testing"

	"local/james-orcales/shared/encoding/json"
	"local/james-orcales/shared/testify"
)

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
		position, status = json.Validate(valid)
	})
	testify.True(t, bool(status))
	testify.Zero_Allocation(t, func() {
		position, status = json.Validate(invalid)
	})
	testify.False(t, bool(status))
	testify.True(t, position > 0)
	testify.Zero_Allocation(t, func() {
		position, status = json.Validate(maximum[:])
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
		count, position, size_status = json.Compact_Size(valid)
	})
	testify.True(t, bool(size_status))
	testify.Zero_Allocation(t, func() {
		count, position, size_status = json.Compact_Size(invalid)
	})
	testify.False(t, bool(size_status))
	testify.Zero_Allocation(t, func() {
		count, position, status = json.Compact_Into(destination[:], valid)
	})
	testify.Equal_Values(t, json.STATUS_OK, status)
	testify.Zero_Allocation(t, func() {
		count, position, status = json.Compact_Into(destination[:], invalid)
	})
	testify.Equal_Values(t, json.STATUS_INPUT_INVALID, status)
	testify.Zero_Allocation(t, func() {
		count, position, status = json.Compact_Into(destination[:0], valid)
	})
	testify.Equal_Values(t, json.STATUS_OUTPUT_TOO_SMALL, status)
	testify.Zero_Allocation(t, func() {
		count, position, status = json.Compact_Into(destination[:1], overlap)
	})
	testify.Equal_Values(t, json.STATUS_STORAGE_INVALID, status)
	testify.Zero_Allocation(t, func() {
		count, position, status = json.Compact_Into(destination[:], maximum[:])
	})
	testify.Equal_Values(t, json.STATUS_OK, status)
	testify.True(t, count <= json.OUTPUT_SIZE_MAXIMUM)
	testify.True(t, position <= json.POSITION_MAXIMUM)
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
		count, position, size_status = json.Indent_Size(valid, prefix, indent)
	})
	testify.Equal_Values(t, json.STATUS_OK, size_status)
	testify.Zero_Allocation(t, func() {
		count, position, size_status = json.Indent_Size(invalid, prefix, indent)
	})
	testify.Equal_Values(t, json.STATUS_INPUT_INVALID, size_status)
	testify.Zero_Allocation(t, func() {
		count, position, size_status = json.Indent_Size(deep, nil, maximum_indent[:])
	})
	testify.Equal_Values(t, json.STATUS_OUTPUT_TOO_LARGE, size_status)
	testify.Zero_Allocation(t, func() {
		count, position, status = json.Indent_Into(destination[:], valid, prefix, indent)
	})
	testify.Equal_Values(t, json.STATUS_OK, status)
	testify.Zero_Allocation(t, func() {
		count, position, status = json.Indent_Into(destination[:], invalid, prefix, indent)
	})
	testify.Equal_Values(t, json.STATUS_INPUT_INVALID, status)
	testify.Zero_Allocation(t, func() {
		count, position, status = json.Indent_Into(destination[:0], valid, prefix, indent)
	})
	testify.Equal_Values(t, json.STATUS_OUTPUT_TOO_SMALL, status)
	testify.Zero_Allocation(t, func() {
		count, position, status = json.Indent_Into(destination[:1], overlap, nil, nil)
	})
	testify.Equal_Values(t, json.STATUS_STORAGE_INVALID, status)
	testify.Zero_Allocation(t, func() {
		count, position, status = json.Indent_Into(
			destination[:], deep, nil, maximum_indent[:],
		)
	})
	testify.Equal_Values(t, json.STATUS_OUTPUT_TOO_LARGE, status)
	testify.Zero_Allocation(t, func() {
		count, position, status = json.Indent_Into(
			destination[:], maximum[:], nil, nil,
		)
	})
	testify.Equal_Values(t, json.STATUS_OK, status)
	testify.True(t, count <= json.OUTPUT_SIZE_MAXIMUM)
	testify.True(t, position <= json.POSITION_MAXIMUM)
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
