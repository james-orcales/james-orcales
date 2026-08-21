// Package json implements bounded RFC 8259 transforms on caller-owned storage.
package json

import (
	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/simulation/aver/default"
)

// ENCODED_SIZE_MAXIMUM follows shared byte-slice boundary.
const ENCODED_SIZE_MAXIMUM = bytes.SLICE_SIZE_MAXIMUM

// OUTPUT_SIZE_MAXIMUM prevents formatting from growing caller work without bound.
const OUTPUT_SIZE_MAXIMUM = bytes.SLICE_SIZE_MAXIMUM

// PREFIX_SIZE_MAXIMUM keeps every formatting input on shared byte-slice boundary.
const PREFIX_SIZE_MAXIMUM = bytes.SLICE_SIZE_MAXIMUM

// INDENT_SIZE_MAXIMUM keeps every formatting input on shared byte-slice boundary.
const INDENT_SIZE_MAXIMUM = bytes.SLICE_SIZE_MAXIMUM

// POSITION_MAXIMUM includes unexpected end immediately after maximum input.
const POSITION_MAXIMUM = ENCODED_SIZE_MAXIMUM + 1

// NESTING_DEPTH_MAXIMUM follows maximum complete empty-container nesting.
const NESTING_DEPTH_MAXIMUM = ENCODED_SIZE_MAXIMUM / 2

// VALUE_SIZE_MINIMUM is the shortest complete JSON value.
const VALUE_SIZE_MINIMUM = bytes.SLICE_SIZE_MINIMUM + 1

// SCALAR_STORAGE_SIZE keeps internal results structural instead of semantically broad.
const SCALAR_STORAGE_SIZE = bytes.SLICE_SIZE_MINIMUM + 1

// SCALAR_STORAGE_INDEX avoids alternate indexing for fixed scalar storage.
const SCALAR_STORAGE_INDEX = bytes.SLICE_SIZE_MINIMUM

// STATUS_OK means operation completed.
const STATUS_OK = 0

// STATUS_INPUT_INVALID means source is not one complete JSON value.
const STATUS_INPUT_INVALID = STATUS_OK + 1

// STATUS_OUTPUT_TOO_SMALL means caller output cannot hold exact result.
const STATUS_OUTPUT_TOO_SMALL = STATUS_INPUT_INVALID + 1

// STATUS_STORAGE_INVALID means output overlaps one read-only input.
const STATUS_STORAGE_INVALID = STATUS_OUTPUT_TOO_SMALL + 1

// STATUS_OUTPUT_TOO_LARGE means formatting exceeds package output bound.
const STATUS_OUTPUT_TOO_LARGE = STATUS_STORAGE_INVALID + 1

// ARRAY_VALUE_OR_END permits the empty-array closing delimiter.
const ARRAY_VALUE_OR_END byte = 0

// ARRAY_VALUE requires a value after a comma.
const ARRAY_VALUE = ARRAY_VALUE_OR_END + 1

// ARRAY_AFTER_VALUE requires a comma or closing delimiter.
const ARRAY_AFTER_VALUE = ARRAY_VALUE + 1

// OBJECT_KEY_OR_END permits the empty-object closing delimiter.
const OBJECT_KEY_OR_END = ARRAY_AFTER_VALUE + 1

// OBJECT_KEY requires a quoted key after a comma.
const OBJECT_KEY = OBJECT_KEY_OR_END + 1

// OBJECT_COLON requires the separator after a key.
const OBJECT_COLON = OBJECT_KEY + 1

// OBJECT_VALUE requires a value after the separator.
const OBJECT_VALUE = OBJECT_COLON + 1

// OBJECT_AFTER_VALUE requires a comma or closing delimiter.
const OBJECT_AFTER_VALUE = OBJECT_VALUE + 1

// Encoded keeps malicious source bounded before syntax validation.
type Encoded []byte

// Encoded_Invariants enforces shared source boundary.
func Encoded_Invariants(value Encoded, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), bytes.SLICE_SIZE_MINIMUM, ENCODED_SIZE_MAXIMUM).
		Ensure()
}

// Output keeps caller destination bounded before capacity checks.
type Output []byte

// Output_Invariants enforces shared destination boundary.
func Output_Invariants(value Output, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), bytes.SLICE_SIZE_MINIMUM, OUTPUT_SIZE_MAXIMUM).
		Ensure()
}

// Prefix keeps each repeated line prefix bounded.
type Prefix []byte

// Prefix_Invariants enforces shared formatting boundary.
func Prefix_Invariants(value Prefix, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), bytes.SLICE_SIZE_MINIMUM, PREFIX_SIZE_MAXIMUM).
		Ensure()
}

// Indent keeps each repeated depth unit bounded.
type Indent []byte

// Indent_Invariants enforces shared formatting boundary.
func Indent_Invariants(value Indent, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), bytes.SLICE_SIZE_MINIMUM, INDENT_SIZE_MAXIMUM).
		Ensure()
}

// Count is exact bounded output bytes or zero on refusal.
type Count int

// Count_Invariants keeps public results inside writable output bound.
func Count_Invariants(value Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), bytes.SLICE_SIZE_MINIMUM, OUTPUT_SIZE_MAXIMUM).
		Ensure()
}

// Compact_Count excludes zero because valid JSON always contains one value byte.
type Compact_Count int

// Compact_Count_Invariants follows compact output limits for valid input.
func Compact_Count_Invariants(value Compact_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), VALUE_SIZE_MINIMUM, OUTPUT_SIZE_MAXIMUM).
		Ensure()
}

// Position is one-based invalid byte or zero when absent.
type Position int

// Position_Invariants includes unexpected end after maximum source.
func Position_Invariants(value Position, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), bytes.SLICE_SIZE_MINIMUM, POSITION_MAXIMUM).
		Ensure()
}

// Validate_Status excludes storage outcomes irrelevant to syntax checks.
type Validate_Status uint8

// Validate_Status_Invariants lists valid and invalid source.
func Validate_Status_Invariants(value Validate_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value), uint8(STATUS_OK), uint8(STATUS_INPUT_INVALID)).
		Ensure()
}

// Compact_Size_Status excludes growth because compact output never exceeds source.
type Compact_Size_Status uint8

// Compact_Size_Status_Invariants lists compact sizing outcomes.
func Compact_Size_Status_Invariants(
	value Compact_Size_Status, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value), uint8(STATUS_OK), uint8(STATUS_INPUT_INVALID)).
		Ensure()
}

// Indent_Size_Status admits bounded growth refusal after valid syntax.
type Indent_Size_Status uint8

// Indent_Size_Status_Invariants lists indentation sizing outcomes.
func Indent_Size_Status_Invariants(
	value Indent_Size_Status, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_INPUT_INVALID),
			uint8(STATUS_OUTPUT_TOO_LARGE),
		).
		Ensure()
}

// Compact_Status includes syntax, output, and overlap refusals.
type Compact_Status uint8

// Compact_Status_Invariants lists every compact outcome.
func Compact_Status_Invariants(value Compact_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_4_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_INPUT_INVALID),
			uint8(STATUS_OUTPUT_TOO_SMALL), uint8(STATUS_STORAGE_INVALID),
		).
		Ensure()
}

// Indent_Status includes every shared transform refusal.
type Indent_Status uint8

// Indent_Status_Invariants covers contiguous indentation outcomes.
func Indent_Status_Invariants(value Indent_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), uint8(STATUS_OK), uint8(STATUS_OUTPUT_TOO_LARGE)).
		Ensure()
}

// Parse_Result keeps recursive parser return state bounded by value.
type Parse_Result struct {
	// Next avoids retaining source suffixes across recursive calls.
	Next [SCALAR_STORAGE_SIZE]int
	// Error keeps unexpected end representable beside byte positions.
	Error [SCALAR_STORAGE_SIZE]int
	// Valid avoids error interfaces and owned messages.
	Valid [SCALAR_STORAGE_SIZE]bool
}

// Parse_Result_Invariants fixes result shape before public scalar conversion.
func Parse_Result_Invariants(value Parse_Result, _ aver.Namespace) {
	aver.Always(
		len(value.Next) == SCALAR_STORAGE_SIZE,
		"JSON parser next position stays in fixed scalar storage.",
	)
	aver.Always(
		len(value.Error) == SCALAR_STORAGE_SIZE,
		"JSON parser error position stays in fixed scalar storage.",
	)
	aver.Always(
		len(value.Valid) == SCALAR_STORAGE_SIZE,
		"JSON parser validity stays in fixed scalar storage.",
	)
}

// Boolean gives lexical decisions independent coverage identity.
type Boolean bool

// Boolean_Invariants covers both lexical outcomes.
func Boolean_Invariants(value Boolean, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "JSON lexical decision is positive.").
		Ensure()
}

// Validate reports first invalid byte without owned diagnostics.
func Validate(source Encoded) (position Position, status Validate_Status) {
	defer func() {
		Position_Invariants(position, "Validate.position")
		Validate_Status_Invariants(status, "Validate.status")
	}()
	Encoded_Invariants(source, "Validate.source")
	result := validate_unchecked(source)
	if !result.Valid[SCALAR_STORAGE_INDEX] {
		return Position(result.Error[SCALAR_STORAGE_INDEX]), STATUS_INPUT_INVALID
	}
	return 0, STATUS_OK
}

// Compact_Size reports exact output before any caller storage can change.
func Compact_Size(
	source Encoded,
) (count Count, position Position, status Compact_Size_Status) {
	defer func() {
		Count_Invariants(count, "Compact_Size.count")
		Position_Invariants(position, "Compact_Size.position")
		Compact_Size_Status_Invariants(status, "Compact_Size.status")
	}()
	Encoded_Invariants(source, "Compact_Size.source")
	result := validate_unchecked(source)
	if !result.Valid[SCALAR_STORAGE_INDEX] {
		return 0, Position(result.Error[SCALAR_STORAGE_INDEX]), STATUS_INPUT_INVALID
	}
	return Count(compact_size_unchecked(source)), 0, STATUS_OK
}

// Compact_Into rejects every refusal before changing caller output.
func Compact_Into(
	destination Output, source Encoded,
) (count Count, position Position, status Compact_Status) {
	defer func() {
		Count_Invariants(count, "Compact_Into.count")
		Position_Invariants(position, "Compact_Into.position")
		Compact_Status_Invariants(status, "Compact_Into.status")
	}()
	Output_Invariants(destination, "Compact_Into.destination")
	Encoded_Invariants(source, "Compact_Into.source")
	if bytes.Overlap(bytes.Slice(destination), bytes.Slice(source)) {
		return 0, 0, STATUS_STORAGE_INVALID
	}
	required, invalid_position, size_status := Compact_Size(source)
	if size_status != STATUS_OK {
		return 0, invalid_position, STATUS_INPUT_INVALID
	}
	if len(destination) < int(required) {
		return 0, 0, STATUS_OUTPUT_TOO_SMALL
	}
	compact_unchecked(destination, source)
	return required, 0, STATUS_OK
}

// Indent_Size refuses unbounded formatting growth before output access.
func Indent_Size(
	source Encoded, prefix Prefix, indent Indent,
) (count Count, position Position, status Indent_Size_Status) {
	defer func() {
		Count_Invariants(count, "Indent_Size.count")
		Position_Invariants(position, "Indent_Size.position")
		Indent_Size_Status_Invariants(status, "Indent_Size.status")
	}()
	Encoded_Invariants(source, "Indent_Size.source")
	Prefix_Invariants(prefix, "Indent_Size.prefix")
	Indent_Invariants(indent, "Indent_Size.indent")
	result := validate_unchecked(source)
	if !result.Valid[SCALAR_STORAGE_INDEX] {
		return 0, Position(result.Error[SCALAR_STORAGE_INDEX]), STATUS_INPUT_INVALID
	}
	required, too_large := indent_size_unchecked(
		source, result.Next[SCALAR_STORAGE_INDEX], prefix, indent,
	)
	if too_large {
		return 0, 0, STATUS_OUTPUT_TOO_LARGE
	}
	return required, 0, STATUS_OK
}

// Indent_Into rejects every refusal before changing caller output.
func Indent_Into(
	destination Output, source Encoded, prefix Prefix, indent Indent,
) (count Count, position Position, status Indent_Status) {
	defer func() {
		Count_Invariants(count, "Indent_Into.count")
		Position_Invariants(position, "Indent_Into.position")
		Indent_Status_Invariants(status, "Indent_Into.status")
	}()
	Output_Invariants(destination, "Indent_Into.destination")
	Encoded_Invariants(source, "Indent_Into.source")
	Prefix_Invariants(prefix, "Indent_Into.prefix")
	Indent_Invariants(indent, "Indent_Into.indent")
	if transform_storage_overlaps(destination, source, prefix, indent) {
		return 0, 0, STATUS_STORAGE_INVALID
	}
	result := validate_unchecked(source)
	if !result.Valid[SCALAR_STORAGE_INDEX] {
		return 0, Position(result.Error[SCALAR_STORAGE_INDEX]), STATUS_INPUT_INVALID
	}
	value_end := result.Next[SCALAR_STORAGE_INDEX]
	required, too_large := indent_size_unchecked(source, value_end, prefix, indent)
	if too_large {
		return 0, 0, STATUS_OUTPUT_TOO_LARGE
	}
	if len(destination) < int(required) {
		return 0, 0, STATUS_OUTPUT_TOO_SMALL
	}
	indent_unchecked(destination, source, value_end, prefix, indent)
	return required, 0, STATUS_OK
}

func new_parse_result[Next ~int, Error ~int, Valid ~bool](
	next Next, error_position Error, valid Valid,
) (result Parse_Result) {
	defer func() { Parse_Result_Invariants(result, "new_parse_result.result") }()
	result.Next[SCALAR_STORAGE_INDEX] = int(next)
	result.Error[SCALAR_STORAGE_INDEX] = int(error_position)
	result.Valid[SCALAR_STORAGE_INDEX] = bool(valid)
	return result
}

func validate_unchecked[Source ~[]byte](source Source) (result Parse_Result) {
	defer func() { Parse_Result_Invariants(result, "validate_unchecked.result") }()
	var kinds [NESTING_DEPTH_MAXIMUM]byte
	var states [NESTING_DEPTH_MAXIMUM]byte
	position := int(skip_space(source, 0))
	if position == len(source) {
		return new_parse_result(0, len(source)+1, false)
	}
	depth := bytes.SLICE_SIZE_MINIMUM
	need_value := true
	for position <= len(source) {
		value_complete := false
		if need_value {
			result = parse_value_start(source, position, &kinds, &states, &depth)
			if !result.Valid[SCALAR_STORAGE_INDEX] {
				return result
			}
			position = result.Next[SCALAR_STORAGE_INDEX]
			if result.Error[SCALAR_STORAGE_INDEX] == STATUS_INPUT_INVALID {
				need_value = false
				continue
			}
			value_complete = true
		} else {
			position = int(skip_space(source, position))
			if position == len(source) {
				return new_parse_result(0, len(source)+1, false)
			}
			result = parse_container(
				source, position, &kinds, &states, &depth, &need_value,
			)
			if !result.Valid[SCALAR_STORAGE_INDEX] {
				return result
			}
			position = result.Next[SCALAR_STORAGE_INDEX]
			value_complete = result.Error[SCALAR_STORAGE_INDEX] == STATUS_OK
		}
		if !value_complete {
			continue
		}
		if depth == bytes.SLICE_SIZE_MINIMUM {
			return validate_complete(source, position)
		}
		mark_parent_complete(kinds[:], states[:], depth)
		need_value = false
	}
	return new_parse_result(0, position+1, false)
}

func parse_value_start[Source ~[]byte, Position_Value ~int, Depth ~int](
	source Source, start Position_Value, kinds *[NESTING_DEPTH_MAXIMUM]byte,
	states *[NESTING_DEPTH_MAXIMUM]byte, depth *Depth,
) (result Parse_Result) {
	defer func() { Parse_Result_Invariants(result, "parse_value_start.result") }()
	position := int(skip_space(source, start))
	if position == len(source) {
		return new_parse_result(0, len(source)+1, false)
	}
	switch source[position] {
	case '{':
		return push_container(position, byte('{'), kinds, states, depth)
	case '[':
		return push_container(position, byte('['), kinds, states, depth)
	case '"':
		return parse_string(source, position)
	case 't':
		return parse_literal(source, position, "true")
	case 'f':
		return parse_literal(source, position, "false")
	case 'n':
		return parse_literal(source, position, "null")
	default:
		return parse_number(source, position)
	}
}

func push_container[Position_Value ~int, Kind ~byte, Depth ~int](
	position Position_Value, kind Kind, kinds *[NESTING_DEPTH_MAXIMUM]byte,
	states *[NESTING_DEPTH_MAXIMUM]byte, depth *Depth,
) (result Parse_Result) {
	defer func() { Parse_Result_Invariants(result, "push_container.result") }()
	if int(*depth) == NESTING_DEPTH_MAXIMUM {
		return new_parse_result(0, int(position)+1, false)
	}
	kinds[*depth] = byte(kind)
	if kind == '[' {
		states[*depth] = ARRAY_VALUE_OR_END
	} else {
		states[*depth] = OBJECT_KEY_OR_END
	}
	*depth++
	return new_parse_result(int(position)+1, STATUS_INPUT_INVALID, true)
}

func parse_container[Source ~[]byte, Position_Value ~int, Depth ~int, Need_Value ~bool](
	source Source, position Position_Value, kinds *[NESTING_DEPTH_MAXIMUM]byte,
	states *[NESTING_DEPTH_MAXIMUM]byte, depth *Depth, need_value *Need_Value,
) (result Parse_Result) {
	defer func() { Parse_Result_Invariants(result, "parse_container.result") }()
	index := int(*depth) - 1
	if kinds[index] == '[' {
		return parse_array_state(source, position, states, depth, need_value)
	}
	return parse_object_state(source, position, states, depth, need_value)
}

func parse_array_state[
	Source ~[]byte, Position_Value ~int, Depth ~int, Need_Value ~bool,
](
	source Source, position Position_Value, states *[NESTING_DEPTH_MAXIMUM]byte,
	depth *Depth, need_value *Need_Value,
) (result Parse_Result) {
	defer func() { Parse_Result_Invariants(result, "parse_array_state.result") }()
	index := int(*depth) - 1
	if states[index] == ARRAY_VALUE_OR_END {
		if source[position] == ']' {
			*depth--
			return new_parse_result(position+1, STATUS_OK, true)
		}
		*need_value = true
		return new_parse_result(position, STATUS_INPUT_INVALID, true)
	}
	if states[index] == ARRAY_VALUE {
		*need_value = true
		return new_parse_result(position, STATUS_INPUT_INVALID, true)
	}
	if source[position] == ']' {
		*depth--
		return new_parse_result(position+1, STATUS_OK, true)
	}
	if source[position] != ',' {
		return new_parse_result(0, position+1, false)
	}
	states[index] = ARRAY_VALUE
	return new_parse_result(position+1, STATUS_INPUT_INVALID, true)
}

func parse_object_state[
	Source ~[]byte, Position_Value ~int, Depth ~int, Need_Value ~bool,
](
	source Source, position Position_Value, states *[NESTING_DEPTH_MAXIMUM]byte,
	depth *Depth, need_value *Need_Value,
) (result Parse_Result) {
	defer func() { Parse_Result_Invariants(result, "parse_object_state.result") }()
	index := int(*depth) - 1
	switch states[index] {
	case OBJECT_KEY_OR_END:
		if source[position] == '}' {
			*depth--
			return new_parse_result(position+1, STATUS_OK, true)
		}
		return parse_object_key(source, position, states, index)
	case OBJECT_KEY:
		return parse_object_key(source, position, states, index)
	case OBJECT_COLON:
		if source[position] != ':' {
			return new_parse_result(0, position+1, false)
		}
		states[index] = OBJECT_VALUE
		*need_value = true
		return new_parse_result(position+1, STATUS_INPUT_INVALID, true)
	default:
		return parse_object_end(position, source[position], states, depth)
	}
}

func parse_object_key[Source ~[]byte, Position_Value ~int, Index ~int](
	source Source, position Position_Value,
	states *[NESTING_DEPTH_MAXIMUM]byte, index Index,
) (result Parse_Result) {
	defer func() { Parse_Result_Invariants(result, "parse_object_key.result") }()
	if source[position] != '"' {
		return new_parse_result(0, position+1, false)
	}
	result = parse_string(source, position)
	if result.Valid[SCALAR_STORAGE_INDEX] {
		states[index] = OBJECT_COLON
		result.Error[SCALAR_STORAGE_INDEX] = STATUS_INPUT_INVALID
	}
	return result
}

func parse_object_end[Position_Value ~int, Value ~byte, Depth ~int](
	position Position_Value, value Value, states *[NESTING_DEPTH_MAXIMUM]byte,
	depth *Depth,
) (result Parse_Result) {
	defer func() { Parse_Result_Invariants(result, "parse_object_end.result") }()
	if value == '}' {
		*depth--
		return new_parse_result(int(position)+1, STATUS_OK, true)
	}
	if value != ',' {
		return new_parse_result(0, int(position)+1, false)
	}
	states[int(*depth)-1] = OBJECT_KEY
	return new_parse_result(int(position)+1, STATUS_INPUT_INVALID, true)
}

func mark_parent_complete[Kinds ~[]byte, States ~[]byte, Depth ~int](
	kinds Kinds, states States, depth Depth,
) {
	index := int(depth) - 1
	if kinds[index] == '[' {
		states[index] = ARRAY_AFTER_VALUE
	} else {
		states[index] = OBJECT_AFTER_VALUE
	}
}

func validate_complete[Source ~[]byte, Position_Value ~int](
	source Source, start Position_Value,
) (result Parse_Result) {
	defer func() { Parse_Result_Invariants(result, "validate_complete.result") }()
	position := int(skip_space(source, start))
	if position != len(source) {
		return new_parse_result(0, position+1, false)
	}
	return new_parse_result(start, 0, true)
}

func parse_string[Source ~[]byte, Position_Value ~int](
	source Source, start Position_Value,
) (result Parse_Result) {
	defer func() { Parse_Result_Invariants(result, "parse_string.result") }()
	for position := int(start) + 1; position < len(source); position++ {
		value := source[position]
		if value == '"' {
			return new_parse_result(position+1, 0, true)
		}
		if value < ' ' {
			return new_parse_result(0, position+1, false)
		}
		if value == '\\' {
			escape := parse_escape(source, position)
			if !escape.Valid[SCALAR_STORAGE_INDEX] {
				return escape
			}
			position = escape.Next[SCALAR_STORAGE_INDEX] - 1
			continue
		}
	}
	return new_parse_result(0, len(source)+1, false)
}

func parse_escape[Source ~[]byte, Position_Value ~int](
	source Source, slash Position_Value,
) (result Parse_Result) {
	defer func() { Parse_Result_Invariants(result, "parse_escape.result") }()
	position := int(slash) + 1
	if position == len(source) {
		return new_parse_result(0, len(source)+1, false)
	}
	value := source[position]
	switch value {
	case '"', '\\', '/', 'b', 'f', 'n', 'r', 't':
		return new_parse_result(position+1, 0, true)
	}
	if value != 'u' {
		return new_parse_result(0, position+1, false)
	}
	for index := position + 1; index < position+5; index++ {
		if index >= len(source) {
			return new_parse_result(0, len(source)+1, false)
		}
		if !bool(hexadecimal(source[index])) {
			return new_parse_result(0, index+1, false)
		}
	}
	return new_parse_result(position+5, 0, true)
}

func parse_literal[Source ~[]byte, Position_Value ~int, Literal ~string](
	source Source, start Position_Value, literal Literal,
) (result Parse_Result) {
	defer func() { Parse_Result_Invariants(result, "parse_literal.result") }()
	for index := range len(literal) {
		position := int(start) + index
		if position >= len(source) {
			return new_parse_result(0, len(source)+1, false)
		}
		if source[position] != literal[index] {
			return new_parse_result(0, position+1, false)
		}
	}
	return new_parse_result(int(start)+len(literal), 0, true)
}

func parse_number[Source ~[]byte, Position_Value ~int](
	source Source, start Position_Value,
) (result Parse_Result) {
	defer func() { Parse_Result_Invariants(result, "parse_number.result") }()
	position := int(start)
	if source[position] == '-' {
		position++
		if position == len(source) {
			return new_parse_result(0, len(source)+1, false)
		}
	}
	if source[position] == '0' {
		position++
	} else {
		if source[position] < '1' {
			return new_parse_result(0, position+1, false)
		}
		if source[position] > '9' {
			return new_parse_result(0, position+1, false)
		}
		position = int(digits_end(source, position))
	}
	if position < len(source) {
		if source[position] == '.' {
			position++
			if position == len(source) {
				return new_parse_result(0, position+1, false)
			}
			if !bool(decimal(source[position])) {
				return new_parse_result(0, position+1, false)
			}
			position = int(digits_end(source, position))
		}
	}
	return parse_exponent(source, position)
}

func parse_exponent[Source ~[]byte, Position_Value ~int](
	source Source, start Position_Value,
) (result Parse_Result) {
	defer func() { Parse_Result_Invariants(result, "parse_exponent.result") }()
	position := int(start)
	if position == len(source) {
		return new_parse_result(position, 0, true)
	}
	if source[position] != 'e' {
		if source[position] != 'E' {
			return new_parse_result(position, 0, true)
		}
	}
	position++
	if position < len(source) {
		switch source[position] {
		case '+', '-':
			position++
		}
	}
	if position == len(source) {
		return new_parse_result(0, len(source)+1, false)
	}
	if !bool(decimal(source[position])) {
		return new_parse_result(0, position+1, false)
	}
	return new_parse_result(digits_end(source, position), 0, true)
}

func skip_space[Source ~[]byte, Position_Value ~int](
	source Source, start Position_Value,
) (next Position_Value) {
	position := int(start)
	for position < len(source) {
		if !bool(space(source[position])) {
			break
		}
		position++
	}
	return Position_Value(position)
}

func digits_end[Source ~[]byte, Position_Value ~int](
	source Source, start Position_Value,
) (next Position_Value) {
	position := int(start)
	for position < len(source) {
		if !bool(decimal(source[position])) {
			break
		}
		position++
	}
	return Position_Value(position)
}

func space[Value ~byte](value Value) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "space.yes") }()
	switch value {
	case ' ', '\t', '\r', '\n':
		return true
	default:
		return false
	}
}

func decimal[Value ~byte](value Value) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "decimal.yes") }()
	if value < '0' {
		return false
	}
	return Boolean(value <= '9')
}

func hexadecimal[Value ~byte](value Value) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "hexadecimal.yes") }()
	if value >= '0' {
		if value <= '9' {
			return true
		}
	}
	if value >= 'a' {
		if value <= 'f' {
			return true
		}
	}
	if value < 'A' {
		return false
	}
	return Boolean(value <= 'F')
}

func compact_size_unchecked[Source ~[]byte](source Source) (size Compact_Count) {
	defer func() {
		Compact_Count_Invariants(size, "compact_size_unchecked.size")
	}()
	quoted := false
	escaped := false
	for _, value := range source {
		if quoted {
			size++
			if escaped {
				escaped = false
				continue
			}
			if value == '\\' {
				escaped = true
				continue
			}
			if value == '"' {
				quoted = false
			}
			continue
		}
		if value == '"' {
			quoted = true
			size++
			continue
		}
		if !bool(space(value)) {
			size++
		}
	}
	return size
}

func compact_unchecked[Destination ~[]byte, Source ~[]byte](
	destination Destination, source Source,
) {
	position := bytes.SLICE_SIZE_MINIMUM
	quoted := false
	escaped := false
	for _, value := range source {
		if quoted {
			destination[position] = value
			position++
		} else if !bool(space(value)) {
			destination[position] = value
			position++
		}
		if quoted {
			if escaped {
				escaped = false
				continue
			}
			if value == '\\' {
				escaped = true
				continue
			}
			if value == '"' {
				quoted = false
			}
		} else if value == '"' {
			quoted = true
		}
	}
}

func indent_size_unchecked[
	Source ~[]byte, End ~int, Prefix_Value ~[]byte, Indent_Value ~[]byte,
](
	source Source, value_end End, prefix Prefix_Value, indent Indent_Value,
) (size Count, too_large Boolean) {
	defer func() {
		Count_Invariants(size, "indent_size_unchecked.size")
		Boolean_Invariants(too_large, "indent_size_unchecked.too_large")
	}()
	calculated := bytes.SLICE_SIZE_MINIMUM
	quoted := false
	escaped := false
	depth := bytes.SLICE_SIZE_MINIMUM
	need_indent := false
	for _, value := range source[:value_end] {
		if quoted {
			calculated++
			quoted, escaped = quote_state(value, quoted, escaped)
			continue
		}
		if bool(space(value)) {
			continue
		}
		if value == '"' {
			quoted = true
		}
		if need_indent {
			if value != '}' {
				if value != ']' {
					need_indent = false
					depth++
					calculated += newline_size(prefix, indent, depth)
				}
			}
		}
		calculated, depth, need_indent = indent_symbol_size(
			calculated, depth, need_indent, value, prefix, indent,
		)
		if calculated > OUTPUT_SIZE_MAXIMUM {
			return 0, true
		}
	}
	calculated += len(source) - int(value_end)
	if calculated > OUTPUT_SIZE_MAXIMUM {
		return 0, true
	}
	return Count(calculated), false
}

func indent_unchecked[
	Destination ~[]byte, Source ~[]byte, End ~int,
	Prefix_Value ~[]byte, Indent_Value ~[]byte,
](
	destination Destination, source Source, value_end End,
	prefix Prefix_Value, indent Indent_Value,
) {
	position := bytes.SLICE_SIZE_MINIMUM
	quoted := false
	escaped := false
	depth := bytes.SLICE_SIZE_MINIMUM
	need_indent := false
	for _, value := range source[:value_end] {
		if quoted {
			destination[position] = value
			position++
			quoted, escaped = quote_state(value, quoted, escaped)
			continue
		}
		if bool(space(value)) {
			continue
		}
		if value == '"' {
			quoted = true
		}
		if need_indent {
			if value != '}' {
				if value != ']' {
					need_indent = false
					depth++
					position = append_newline(
						destination, position, prefix, indent, depth,
					)
				}
			}
		}
		position, depth, need_indent = indent_symbol_write(
			destination, position, depth, need_indent, value, prefix, indent,
		)
	}
	copy(destination[position:], source[value_end:])
}

func quote_state[Value ~byte, Quoted ~bool, Escaped ~bool](
	value Value, quoted Quoted, escaped Escaped,
) (next_quoted Quoted, next_escaped Escaped) {
	if escaped {
		return quoted, false
	}
	if value == '\\' {
		return quoted, true
	}
	if value == '"' {
		return false, false
	}
	return quoted, false
}

func newline_size[Prefix_Value ~[]byte, Indent_Value ~[]byte, Depth ~int](
	prefix Prefix_Value, indent Indent_Value, depth Depth,
) (size Depth) {
	return Depth(1 + len(prefix) + int(depth)*len(indent))
}

func indent_symbol_size[
	Size ~int, Depth ~int, Need_Indent ~bool, Value ~byte,
	Prefix_Value ~[]byte, Indent_Value ~[]byte,
](
	size Size, depth Depth, need_indent Need_Indent, value Value,
	prefix Prefix_Value, indent Indent_Value,
) (next_size Size, next_depth Depth, next_need_indent Need_Indent) {
	next_size = size + 1
	next_depth = depth
	next_need_indent = need_indent
	switch value {
	case '{', '[':
		next_need_indent = true
	case ',':
		next_size += Size(newline_size(prefix, indent, depth))
	case ':':
		next_size++
	case '}', ']':
		if need_indent {
			next_need_indent = false
		} else {
			next_depth--
			next_size += Size(newline_size(prefix, indent, next_depth))
		}
	}
	return next_size, next_depth, next_need_indent
}

func append_newline[
	Destination ~[]byte, Position_Value ~int,
	Prefix_Value ~[]byte, Indent_Value ~[]byte, Depth ~int,
](
	destination Destination, start Position_Value,
	prefix Prefix_Value, indent Indent_Value, depth Depth,
) (next Position_Value) {
	position := int(start)
	destination[position] = '\n'
	position++
	position += copy(destination[position:], prefix)
	for range int(depth) {
		position += copy(destination[position:], indent)
	}
	return Position_Value(position)
}

func indent_symbol_write[
	Destination ~[]byte, Position_Value ~int, Depth ~int, Need_Indent ~bool,
	Value ~byte,
	Prefix_Value ~[]byte, Indent_Value ~[]byte,
](
	destination Destination, position Position_Value, depth Depth,
	need_indent Need_Indent, value Value, prefix Prefix_Value, indent Indent_Value,
) (next Position_Value, next_depth Depth, next_need_indent Need_Indent) {
	next = position
	next_depth = depth
	next_need_indent = need_indent
	switch value {
	case '{', '[':
		destination[next] = byte(value)
		next++
		next_need_indent = true
	case ',':
		destination[next] = byte(value)
		next++
		next = append_newline(destination, next, prefix, indent, depth)
	case ':':
		destination[next] = byte(value)
		destination[next+1] = ' '
		next += 2
	case '}', ']':
		if need_indent {
			next_need_indent = false
		} else {
			next_depth--
			next = append_newline(destination, next, prefix, indent, next_depth)
		}
		destination[next] = byte(value)
		next++
	default:
		destination[next] = byte(value)
		next++
	}
	return next, next_depth, next_need_indent
}

func transform_storage_overlaps(
	destination Output, source Encoded, prefix Prefix, indent Indent,
) (overlaps Boolean) {
	defer func() { Boolean_Invariants(overlaps, "transform_storage_overlaps.overlaps") }()
	Output_Invariants(destination, "transform_storage_overlaps.destination")
	Encoded_Invariants(source, "transform_storage_overlaps.source")
	Prefix_Invariants(prefix, "transform_storage_overlaps.prefix")
	Indent_Invariants(indent, "transform_storage_overlaps.indent")
	if bytes.Overlap(bytes.Slice(destination), bytes.Slice(source)) {
		return true
	}
	if bytes.Overlap(bytes.Slice(destination), bytes.Slice(prefix)) {
		return true
	}
	return Boolean(bytes.Overlap(bytes.Slice(destination), bytes.Slice(indent)))
}
