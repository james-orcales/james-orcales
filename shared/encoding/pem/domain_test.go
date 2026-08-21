package pem

import "testing"

// Test_Internal_Boundaries reaches internal scalar boundary contracts through production code.
func Test_Internal_Boundaries(_ *testing.T) {
	var encoded_storage [ENCODED_SIZE_MAXIMUM]byte
	var decoded_storage [DECODED_SIZE_MAXIMUM]byte
	source := Block_Source(encoded_storage[:])
	begin_source := Begin_Source(encoded_storage[:])
	section_source := Section_Source(encoded_storage[:])
	output := Block_Output(encoded_storage[:])
	positions := [...]Internal_Position{0, 1, 2, ENCODED_SIZE_MAXIMUM}
	for _, position := range positions {
		body_decoded_size(source, position, position)
		body_encoding_valid(source, position, position)
		decode_body(Decoded(decoded_storage[:]), source, position, position, 0)
		encode_body(output, position, nil)
		end_line_valid(section_source, position, position, position, position)
		fill_headers(nil, source, position, 0)
		find_begin(Encoded(encoded_storage[:]), position)
		line_bounds(begin_source, position)
		line_colon(section_source, position, position)
		parse_sections(section_source, position, position, position)
		text_at(begin_source, position, BEGIN_PREFIX)
		trim_right(begin_source, position)
	}
	for _, value := range [...]PEM_Byte{0, 1, 2, BYTE_MAXIMUM} {
		body_space(value)
		header_space(value)
		horizontal_space(value)
	}
	for _, position := range [...]Internal_Position{0, 1, 2} {
		encoded_storage[position] = LINE_FEED
		line_bounds(begin_source, position)
		encoded_storage[position] = HEADER_KEY_SEPARATOR
		line_colon(section_source, position, position+1)
		encoded_storage[position] = 0
	}
	encoded_storage[COLON_POSITION_MAXIMUM] = HEADER_KEY_SEPARATOR
	line_colon(
		section_source, COLON_POSITION_MAXIMUM, Internal_Position(ENCODED_SIZE_MAXIMUM),
	)
}
