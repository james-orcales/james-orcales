package csv

import (
	"testing"

	"local/james-orcales/shared/unicode/utf8"
)

// DOMAIN_STORAGE_SIZE reaches zero, one, and two positions.
const DOMAIN_STORAGE_SIZE = 3

// Test_Internal_Boundaries reaches internal parser boundaries through production code.
func Test_Internal_Boundaries(_ *testing.T) {
	parsers := [...]Parser{
		{},
		{
			Source_Position: 1, Decoded_Count: 1, Line: 1, Column: 1,
			Record_Done: true,
		},
		{Source_Position: 2, Decoded_Count: 2, Line: 2, Column: 2},
		{
			Source_Position: ENCODED_SIZE_MAXIMUM,
			Decoded_Count:   DECODED_SIZE_MAXIMUM,
			Line:            LINE_MAXIMUM,
			Column:          COLUMN_MAXIMUM,
			Record_Done:     true,
		},
	}
	field_statuses := [...]Field_Status{
		STATUS_OK, STATUS_OUTPUT_TOO_SMALL, STATUS_INPUT_INVALID, STATUS_OK,
	}
	for index, parser := range parsers {
		new_field_result(
			parser, Error_Line(parser.Line), Error_Column(parser.Column),
			field_statuses[index],
		)
	}
	new_decode_result(0, 0, 0, 0, STATUS_OK)
	new_decode_result(1, 1, 1, 1, STATUS_OUTPUT_TOO_SMALL)
	new_decode_result(2, 2, 2, 2, STATUS_END)
	new_decode_result(
		FIELD_COUNT_MAXIMUM, ENCODED_SIZE_MAXIMUM, LINE_MAXIMUM,
		COLUMN_MAXIMUM, STATUS_FIELDS_TOO_SMALL,
	)
	var source_storage [ENCODED_SIZE_MAXIMUM]byte
	var decoded_storage [DECODED_SIZE_MAXIMUM]byte
	source := Nonempty_Encoded(source_storage[:])
	decoded := Decoded(decoded_storage[:])
	delimiter := Delimiter_Character(STANDARD_DELIMITER)
	decode_field(nil, source[:1], delimiter, false, Parser{Line: 1, Column: 1})
	decode_field(decoded, source, delimiter, false, parsers[3])
	invalid := Nonempty_Encoded([]byte{'a', QUOTE})
	decode_field(decoded, invalid, delimiter, false, Parser{Line: 1, Column: 1})
	for _, parser := range parsers {
		trim_prefix_space(source, parser, false)
		decode_unquoted(decoded, source, delimiter, false, parser)
		decode_quoted_content(decoded, source, delimiter, false, parser)
		decode_quoted_content(decoded, source, delimiter, true, parser)
	}
	decode_unquoted(nil, source[:1], delimiter, false, Parser{})
	decode_quoted_content(nil, source[:1], delimiter, false, Parser{})
	decode_field(decoded, source, delimiter, false, Parser{})
	decode_field(decoded, source[:1], delimiter, false, Parser{Source_Position: 1})
	decode_quoted_content(
		decoded, source, delimiter, false,
		Parser{Source_Position: ENCODED_SIZE_MAXIMUM, Decoded_Count: 2},
	)
	for _, parser := range [...]Parser{
		{Line: LINE_MAXIMUM, Column: COLUMN_MAXIMUM},
		{Line: 2, Column: 1},
	} {
		decode_unquoted(decoded, invalid[1:], delimiter, false, parser)
	}
}

// Test_Internal_Record_Ending_Boundaries reaches record ending helper boundaries.
func Test_Internal_Record_Ending_Boundaries(_ *testing.T) {
	var decoded_storage [DECODED_SIZE_MAXIMUM]byte
	decoded := Decoded(decoded_storage[:])
	parsers := [...]Parser{
		{},
		{Source_Position: 2, Decoded_Count: 2, Line: 2, Column: 2},
		{
			Source_Position: ENCODED_SIZE_MAXIMUM,
			Decoded_Count:   DECODED_SIZE_MAXIMUM,
			Line:            LINE_MAXIMUM,
			Column:          COLUMN_MAXIMUM,
			Record_Done:     true,
		},
	}
	var ending_storage [ENCODED_SIZE_MAXIMUM]byte
	ending_storage[2] = LINE_FEED
	ending_source := Nonempty_Encoded(ending_storage[:])
	consume_record_ending(ending_source[:1], Parser{})
	consume_record_ending(ending_source[:DOMAIN_STORAGE_SIZE], parsers[1])
	consume_record_ending(ending_source, parsers[2])
	copy_record_ending(nil, ending_source[:1], Parser{})
	copy_record_ending(decoded[:2], ending_source[:DOMAIN_STORAGE_SIZE], parsers[1])
	copy_record_ending(decoded, ending_source, parsers[2])
	var plain_storage [DOMAIN_STORAGE_SIZE]byte
	plain_source := Nonempty_Encoded(plain_storage[:])
	consume_record_ending(
		plain_source[:2], Parser{Source_Position: 1, Line: 1, Column: 1},
	)
	consume_record_ending(plain_source, Parser{Source_Position: 2, Column: 2})
	copy_record_ending(
		decoded[:1], plain_source[:2],
		Parser{Source_Position: 1, Decoded_Count: 1, Column: 1},
	)
}

// Test_Internal_Delimiter_Boundaries reaches delimiter and character parser boundaries.
func Test_Internal_Delimiter_Boundaries(_ *testing.T) {
	var source_storage [ENCODED_SIZE_MAXIMUM]byte
	source := Nonempty_Encoded(source_storage[:])
	delimiters := [...]Delimiter_Character{
		1, 2, 3, Delimiter_Character(utf8.DECODED_CHARACTER_MAXIMUM),
	}
	positions := [...]Source_Position{0, 1, 2, ENCODED_SIZE_MAXIMUM}
	comments := [...]Comment_Character{
		0, 1, 2, Comment_Character(utf8.DECODED_CHARACTER_MAXIMUM),
	}
	next_lines := [...]Next_Line{0, 1, 2, NEXT_LINE_MAXIMUM}
	for index, position := range positions {
		delimiter_at(source, position, delimiters[index])
		encode_field(
			Nonempty_Encoded(source_storage[:]), position, nil,
			delimiters[index], false,
		)
		skip_ignored(Encoded(source), position, next_lines[index], comments[index])
	}
	for _, delimiter := range delimiters {
		encoded_field_size(nil, delimiter, false)
		parser := Parser{
			Source_Position: ENCODED_SIZE_MAXIMUM,
			Decoded_Count:   DECODED_SIZE_MAXIMUM,
			Line:            LINE_MAXIMUM,
			Column:          COLUMN_MAXIMUM,
		}
		decode_field(nil, source, delimiter, false, parser)
		decode_unquoted(nil, source, delimiter, false, parser)
		decode_quoted_content(nil, source, delimiter, false, parser)
	}
	source_storage[0] = LINE_FEED
	physical_line(source, 0)
	physical_line(source, ENCODED_SIZE_MAXIMUM)
	record_ending_at(source, 0)
	record_ending_at(source, ENCODED_SIZE_MAXIMUM-1)
}
