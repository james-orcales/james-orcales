package json

import "testing"

// Test_Internal_Boundaries reaches internal scalar boundaries through production code.
func Test_Internal_Boundaries(_ *testing.T) {
	for _, value := range [...]JSON_Byte{0, 1, 2, BYTE_MAXIMUM} {
		decimal(value)
		hexadecimal(value)
		space(value)
		for _, quoted := range [...]Boolean{false, true} {
			for _, escaped := range [...]Boolean{false, true} {
				quote_state(value, quoted, escaped)
			}
		}
		indent_symbol_size(0, 0, false, value, nil, nil)
	}
	quote_state('\\', true, false)
	indent_symbol_size(FORMATTING_SIZE_MAXIMUM-1, NESTING_DEPTH_MAXIMUM, true, 0, nil, nil)
	newline_size(nil, nil, 0)
	var affix [INDENT_SIZE_MAXIMUM]byte
	newline_size(affix[:1], nil, 0)
	newline_size(affix[:2], nil, 0)
	newline_size(affix[:], affix[:], NESTING_DEPTH_MAXIMUM)
	var source_storage [ENCODED_SIZE_MAXIMUM]byte
	source := Encoded(source_storage[:])
	nonempty := Nonempty_Encoded(source_storage[:])
	for _, position := range [...]Internal_Position{0, 1, 2, POSITION_MAXIMUM} {
		skip_space(source, position)
		digits_end(nonempty, position)
	}
	for _, position := range [...]Source_Index{0, 1, 2, SOURCE_INDEX_MAXIMUM} {
		parse_literal(nonempty, position, "true")
		parse_string(nonempty, position)
		parse_number(nonempty, position)
	}
}

// Test_Internal_Parser_Boundaries drives parser state edges.
func Test_Internal_Parser_Boundaries(_ *testing.T) {
	var source_storage [ENCODED_SIZE_MAXIMUM]byte
	var kind_storage [NESTING_DEPTH_MAXIMUM]byte
	var state_storage [NESTING_DEPTH_MAXIMUM]byte
	source := Nonempty_Encoded(source_storage[:])
	container := Container_Encoded(source_storage[:])
	kinds := Container_Kinds(kind_storage[:])
	states := Container_States(state_storage[:])

	for _, position := range [...]Source_Index{0, 1, 2, SOURCE_INDEX_MAXIMUM} {
		parse_value_start(source, position, kinds, states, 0)
		push_container(position, Container_Kind('['), kinds, states, 0)
		push_container(position, Container_Kind('{'), kinds, states, 0)
	}
	push_container(0, Container_Kind('['), kinds, states, NESTING_DEPTH_MAXIMUM)
	source_storage[SOURCE_INDEX_MAXIMUM] = '-'
	parse_number(source, SOURCE_INDEX_MAXIMUM)
	parse_value_start(source, SOURCE_INDEX_MAXIMUM, kinds, states, 0)
	source_storage[0] = '{'
	parse_value_start(source, 0, kinds, states, NESTING_DEPTH_MAXIMUM)
	source_storage[0] = 0
	source_storage[SOURCE_INDEX_MAXIMUM] = 't'
	parse_literal(source, SOURCE_INDEX_MAXIMUM, "true")
	source_storage[SOURCE_INDEX_MAXIMUM] = 0

	for _, position := range [...]Container_Position{1, 2, SOURCE_INDEX_MAXIMUM} {
		source_storage[position] = ']'
		states[0] = ARRAY_VALUE_OR_END
		parse_array_state(container, position, states, 1)
		kinds[0] = '['
		parse_container(container, position, kinds, states, 1)
		source_storage[position] = 0
		parse_array_state(container, position, states, 1)
		states[0] = ARRAY_AFTER_VALUE
		parse_array_state(container, position, states, 1)
		source_storage[position] = ','
		parse_array_state(container, position, states, 1)

		source_storage[position] = '}'
		states[0] = OBJECT_KEY_OR_END
		parse_object_state(container, position, states, 1)
		kinds[0] = '{'
		parse_container(container, position, kinds, states, 1)
		source_storage[position] = 0
		parse_object_state(container, position, states, 1)
	}
}

// Test_Internal_Parser_Result_Boundaries drives parser result edges.
func Test_Internal_Parser_Result_Boundaries(_ *testing.T) {
	var source_storage [ENCODED_SIZE_MAXIMUM]byte
	var kind_storage [NESTING_DEPTH_MAXIMUM]byte
	var state_storage [NESTING_DEPTH_MAXIMUM]byte
	source := Nonempty_Encoded(source_storage[:])
	container := Container_Encoded(source_storage[:])
	kinds := Container_Kinds(kind_storage[:])
	states := Container_States(state_storage[:])
	for _, index := range [...]Container_Index{0, 1, 2, CONTAINER_INDEX_MAXIMUM} {
		source_storage[1] = 0
		parse_object_key(container, 1, states, index)
		source_storage[1], source_storage[2] = '"', '"'
		parse_object_key(container, 1, states, index)
	}
	source_storage[SOURCE_INDEX_MAXIMUM] = '"'
	parse_object_key(container, SOURCE_INDEX_MAXIMUM, states, 0)

	for _, position := range [...]Container_Position{1, 2, SOURCE_INDEX_MAXIMUM} {
		parse_escape(Escape_Encoded(source), position)
	}
	for _, position := range [...]End_Position{1, 2, ENCODED_SIZE_MAXIMUM} {
		parse_exponent(source, position)
		validate_complete(source, position)
	}
	validate_complete(source[:1], 1)
	source_storage[SOURCE_INDEX_MAXIMUM] = 'e'
	parse_exponent(source, ENCODED_SIZE_MAXIMUM-1)

	for _, depth := range [...]Active_Depth{1, 2, NESTING_DEPTH_MAXIMUM} {
		kinds[depth-1] = '['
		states[depth-1] = ARRAY_AFTER_VALUE
		parse_container(container, 1, kinds, states, depth)
		parse_array_state(container, 1, states, depth)
		parse_object_state(container, 1, states, depth)
		mark_parent_complete(kinds, states, depth)
	}
}

// Test_Internal_Formatting_Boundaries drives formatter state edges.
func Test_Internal_Formatting_Boundaries(_ *testing.T) {
	var source_storage [ENCODED_SIZE_MAXIMUM]byte
	var output_storage [OUTPUT_SIZE_MAXIMUM]byte
	source := Nonempty_Encoded(source_storage[:])
	output := Nonempty_Output(output_storage[:])
	formatted := Formatted_Output(output_storage[:])

	for _, value_end := range [...]End_Position{1, 2, ENCODED_SIZE_MAXIMUM} {
		indent_size_unchecked(source, value_end, nil, nil)
		indent_unchecked(output, source, value_end, nil, nil)
	}
	for _, start := range [...]Nonzero_Output_Index{1, 2, OUTPUT_INDEX_MAXIMUM} {
		append_newline(formatted, start, nil, nil, 0)
	}
	append_newline(formatted, 1, nil, nil, NESTING_DEPTH_MAXIMUM)
	for _, value := range [...]JSON_Byte{0, 1, 2, BYTE_MAXIMUM} {
		indent_symbol_write(output, 0, 0, false, value, nil, nil)
	}
	indent_symbol_write(output, OUTPUT_INDEX_MAXIMUM, 0, false, 0, nil, nil)
	indent_symbol_write(output, 0, NESTING_DEPTH_MAXIMUM, false, 0, nil, nil)
}
