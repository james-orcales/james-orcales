package lint

import (
	"fmt"

	"local/james-orcales/lint/internal/strings"
	"local/james-orcales/shared/unicode/utf8"
)

func configuration_json_decode(
	source []byte,
) (configuration *Configuration, present map[string]bool, err error) {
	configuration = &Configuration{}
	present = map[string]bool{}
	offset, err := json_byte(source, 0, '{')
	if err != nil {
		return nil, nil, err
	}
	offset = json_space_offset(source, offset)
	if offset < len(source) {
		if source[offset] == '}' {
			return configuration_json_finish(source, offset+1, configuration, present)
		}
	}
	for offset < len(source) {
		key, next, key_err := json_string(source, offset)
		if key_err != nil {
			return nil, nil, key_err
		}
		offset, err = json_byte(source, next, ':')
		if err != nil {
			return nil, nil, err
		}
		offset, err = configuration_json_value(source, offset, key, configuration)
		if err != nil {
			return nil, nil, err
		}
		present[key] = true
		offset = json_space_offset(source, offset)
		if offset >= len(source) {
			return nil, nil, fmt.Errorf("lint.json: unfinished object")
		}
		if source[offset] == '}' {
			return configuration_json_finish(source, offset+1, configuration, present)
		}
		if source[offset] != ',' {
			return nil, nil, fmt.Errorf("lint.json: expected comma at byte %d", offset)
		}
		offset = json_space_offset(source, offset+1)
	}
	return nil, nil, fmt.Errorf("lint.json: unfinished object")
}

func configuration_json_finish(
	source []byte, offset int, configuration *Configuration, present map[string]bool,
) (decoded *Configuration, decoded_present map[string]bool, err error) {
	offset = json_space_offset(source, offset)
	if offset != len(source) {
		return nil, nil, fmt.Errorf("lint.json: trailing data at byte %d", offset)
	}
	return configuration, present, nil
}

func configuration_json_value(
	source []byte, offset int, key string, configuration *Configuration,
) (next int, err error) {
	switch key {
	case "shared_component":
		configuration.Shared_Component, next, err = json_string(source, offset)
	case "instrumentation_packages":
		configuration.Instrumentation_Packages, next, err = json_string_list(source, offset)
	case "pure_but_indeterministic_packages":
		configuration.Pure_But_Indeterministic, next, err = json_string_list(source, offset)
	case "word_replacements":
		configuration.Word_Replacements, next, err = json_string_list_map(source, offset)
	case "ignore":
		configuration.Ignore, next, err = json_string_list(source, offset)
	case "opt_out_assertion_mandate_packages":
		configuration.Invariant_Exempt_Packages, next, err =
			json_string_list(source, offset)
	case "opt_out_recursion_ban":
		configuration.Recursion_Exempt, next, err = json_string_list(source, offset)
	default:
		return 0, fmt.Errorf("lint.json: unknown key %q", key)
	}
	return next, err
}

func json_string_list(source []byte, offset int) (values []string, next int, err error) {
	offset, err = json_byte(source, offset, '[')
	if err != nil {
		return nil, 0, err
	}
	offset = json_space_offset(source, offset)
	if offset < len(source) {
		if source[offset] == ']' {
			return []string{}, offset + 1, nil
		}
	}
	for offset < len(source) {
		value, value_next, value_err := json_string(source, offset)
		if value_err != nil {
			return nil, 0, value_err
		}
		values = append(values, value)
		offset = json_space_offset(source, value_next)
		if offset >= len(source) {
			return nil, 0, fmt.Errorf("lint.json: unfinished array")
		}
		if source[offset] == ']' {
			return values, offset + 1, nil
		}
		if source[offset] != ',' {
			return nil, 0, fmt.Errorf("lint.json: expected comma at byte %d", offset)
		}
		offset = json_space_offset(source, offset+1)
	}
	return nil, 0, fmt.Errorf("lint.json: unfinished array")
}

func json_string_list_map(
	source []byte, offset int,
) (values map[string][]string, next int, err error) {
	values = map[string][]string{}
	offset, err = json_byte(source, offset, '{')
	if err != nil {
		return nil, 0, err
	}
	offset = json_space_offset(source, offset)
	if offset < len(source) {
		if source[offset] == '}' {
			return values, offset + 1, nil
		}
	}
	for offset < len(source) {
		key, key_next, key_err := json_string(source, offset)
		if key_err != nil {
			return nil, 0, key_err
		}
		offset, err = json_byte(source, key_next, ':')
		if err != nil {
			return nil, 0, err
		}
		value, value_next, value_err := json_string_list(source, offset)
		if value_err != nil {
			return nil, 0, value_err
		}
		values[key] = value
		offset = json_space_offset(source, value_next)
		if offset >= len(source) {
			return nil, 0, fmt.Errorf("lint.json: unfinished object")
		}
		if source[offset] == '}' {
			return values, offset + 1, nil
		}
		if source[offset] != ',' {
			return nil, 0, fmt.Errorf("lint.json: expected comma at byte %d", offset)
		}
		offset = json_space_offset(source, offset+1)
	}
	return nil, 0, fmt.Errorf("lint.json: unfinished object")
}

func json_string(source []byte, offset int) (text string, next int, err error) {
	offset = json_space_offset(source, offset)
	if offset >= len(source) {
		return "", 0, fmt.Errorf("lint.json: expected string")
	}
	if source[offset] != '"' {
		return "", 0, fmt.Errorf("lint.json: expected string at byte %d", offset)
	}
	var builder strings.Builder
	for offset = offset + 1; offset < len(source); {
		value := source[offset]
		if value == '"' {
			return builder.String(), offset + 1, nil
		}
		if value == '\\' {
			offset, err = json_escape(source, offset+1, &builder)
			if err != nil {
				return "", 0, err
			}
			continue
		}
		offset, err = json_plain_character(source, offset, &builder)
		if err != nil {
			return "", 0, err
		}
	}
	return "", 0, fmt.Errorf("lint.json: unfinished string")
}

func json_plain_character(
	source []byte, offset int, builder *strings.Builder,
) (next int, err error) {
	if source[offset] < 0x20 {
		return 0, fmt.Errorf("lint.json: control byte in string at byte %d", offset)
	}
	if source[offset] < byte(utf8.CHARACTER_SELF) {
		strings.Builder_Write_Byte(builder, source[offset])
		return offset + 1, nil
	}
	end := offset + utf8.UTF_MAXIMUM
	if end > len(source) {
		end = len(source)
	}
	character, size := utf8.Decode_Character(utf8.Bytes(source[offset:end]))
	if size == utf8.CHARACTER_SIZE_MINIMUM {
		if character == utf8.REPLACEMENT_CHARACTER {
			return 0, fmt.Errorf("lint.json: invalid UTF-8 at byte %d", offset)
		}
	}
	strings.Builder_Write_Text(builder, string(source[offset:offset+int(size)]))
	return offset + int(size), nil
}

func json_escape(
	source []byte, offset int, builder *strings.Builder,
) (next int, err error) {
	if offset >= len(source) {
		return 0, fmt.Errorf("lint.json: unfinished escape")
	}
	switch source[offset] {
	case '"', '\\', '/':
		strings.Builder_Write_Byte(builder, source[offset])
		return offset + 1, nil
	case 'b':
		strings.Builder_Write_Byte(builder, '\b')
	case 'f':
		strings.Builder_Write_Byte(builder, '\f')
	case 'n':
		strings.Builder_Write_Byte(builder, '\n')
	case 'r':
		strings.Builder_Write_Byte(builder, '\r')
	case 't':
		strings.Builder_Write_Byte(builder, '\t')
	case 'u':
		return json_unicode_escape(source, offset+1, builder)
	default:
		return 0, fmt.Errorf("lint.json: invalid escape at byte %d", offset)
	}
	return offset + 1, nil
}

func json_unicode_escape(
	source []byte, offset int, builder *strings.Builder,
) (next int, err error) {
	character, next, err := json_hexadecimal_character(source, offset)
	if err != nil {
		return 0, err
	}
	if 0xd800 <= character {
		if character <= 0xdbff {
			return json_surrogate_pair(source, next, character, builder)
		}
		if character <= 0xdfff {
			return 0, fmt.Errorf("lint.json: lone low surrogate at byte %d", offset)
		}
	}
	json_write_character(builder, character)
	return next, nil
}

func json_surrogate_pair(
	source []byte, offset int, high rune, builder *strings.Builder,
) (next int, err error) {
	if offset+2 > len(source) {
		return 0, fmt.Errorf("lint.json: unfinished surrogate pair")
	}
	if source[offset] != '\\' {
		return 0, fmt.Errorf("lint.json: high surrogate without pair at byte %d", offset)
	}
	if source[offset+1] != 'u' {
		return 0, fmt.Errorf("lint.json: high surrogate without pair at byte %d", offset)
	}
	low, next, err := json_hexadecimal_character(source, offset+2)
	if err != nil {
		return 0, err
	}
	if low < 0xdc00 {
		return 0, fmt.Errorf("lint.json: invalid low surrogate at byte %d", offset)
	}
	if low > 0xdfff {
		return 0, fmt.Errorf("lint.json: invalid low surrogate at byte %d", offset)
	}
	character := rune(0x10000) + (high-0xd800)*0x400 + low - 0xdc00
	json_write_character(builder, character)
	return next, nil
}

func json_hexadecimal_character(
	source []byte, offset int,
) (character rune, next int, err error) {
	final_offset := offset + 4
	if final_offset > len(source) {
		return 0, 0, fmt.Errorf("lint.json: unfinished Unicode escape")
	}
	for digit_offset := offset; digit_offset < final_offset; digit_offset++ {
		digit, valid := json_hexadecimal_digit(source[digit_offset])
		if !valid {
			return 0, 0, fmt.Errorf(
				"lint.json: invalid Unicode escape at byte %d", digit_offset,
			)
		}
		character = character*16 + rune(digit)
	}
	return character, final_offset, nil
}

func json_hexadecimal_digit(value byte) (digit byte, valid bool) {
	if '0' <= value {
		if value <= '9' {
			return value - '0', true
		}
	}
	if 'a' <= value {
		if value <= 'f' {
			return value - 'a' + 10, true
		}
	}
	if 'A' <= value {
		if value <= 'F' {
			return value - 'A' + 10, true
		}
	}
	return 0, false
}

func json_write_character(builder *strings.Builder, character rune) {
	var destination [utf8.UTF_MAXIMUM]byte
	size := utf8.Encode_Character(destination[:], utf8.Character(character))
	strings.Builder_Write_Text(builder, string(destination[:int(size)]))
}

func json_byte(source []byte, offset int, expected byte) (next int, err error) {
	offset = json_space_offset(source, offset)
	if offset >= len(source) {
		return 0, fmt.Errorf("lint.json: expected %q", expected)
	}
	if source[offset] != expected {
		return 0, fmt.Errorf("lint.json: expected %q at byte %d", expected, offset)
	}
	return offset + 1, nil
}

func json_space_offset(source []byte, offset int) (next int) {
	for offset < len(source) {
		switch source[offset] {
		case ' ', '\t', '\n', '\r':
			offset++
		default:
			return offset
		}
	}
	return offset
}
