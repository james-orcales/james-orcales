package template

import (
	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/math/big"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/sim/aver/default"
	"local/james-orcales/shared/strconv"
	"local/james-orcales/shared/unicode/ucd"
	"local/james-orcales/shared/unicode/utf8"
)

func output_composite_child(output Output_Storage, number Number_Storage,
	count_value Output_Composite_Child_Staged,
	frame_state Value_Format_Frame_Storage_Pointer, float_parse Float_Parse_Workspace_Pointer,
	float_text Float_Text_Workspace_Pointer,
) (_ Value, _ Output_Count_Staged, status Output_Status) {
	defer func() { Output_Status_Invariants(status, "output_composite_child.status") }()
	Output_Storage_Invariants(output, "output_composite_child.output")
	Number_Storage_Invariants(number, "output_composite_child.number")
	Output_Composite_Child_Staged_Invariants(count_value, "output_composite_child.count_value")
	Value_Format_Frame_Storage_Pointer_Invariants(
		frame_state, "output_composite_child.frame_state")
	Float_Parse_Workspace_Pointer_Invariants(float_parse, "output_composite_child.float_parse")
	Float_Text_Workspace_Pointer_Invariants(float_text, "output_composite_child.float_text")
	var child Value
	defer func() { Value_Invariants(child, "output_composite_child.child") }()
	written := Output_Count_Staged(count_value)
	defer func() {
		Written_Child_Output_Count_Invariants(
			Written_Child_Output_Count(written), "output_composite_child.written")
	}()
	frame := &frame_state.Value
	value := frame.Value
	index := frame.Index
	frame.Index++
	kind := Value_Kind(value.Kind)
	Composite_Kind_Invariants(Composite_Kind(kind), "output_composite_child.kind")
	switch kind {
	case VALUE_SEQUENCE:
		child = Values(value.Values)[index]
		return child, written, OUTPUT_STATUS_OK
	case VALUE_OBJECT:
		child = Fields(value.Fields)[index].Value.Data
		return child, written, OUTPUT_STATUS_OK
	case VALUE_MAP:
		key, item, selected, selection := range_map_next(
			Mapped_Value(value), Prior_Map_Entry_Index(frame.Map_Entry_Index),
			frame.Map_Selected)
		frame.Map_Entry_Index = selected
		frame.Map_Selected = selection
		var write_status Output_Status
		written, write_status = output_scalar(
			output, number, Output_Count_Staged(count_value), Scalar_Value(key),
			float_parse, float_text)
		if write_status != OUTPUT_STATUS_OK {
			return child, written, OUTPUT_STATUS_TOO_LARGE
		}
		written, write_status = output_text(output, written, ":")
		if write_status != OUTPUT_STATUS_OK {
			return child, written, OUTPUT_STATUS_TOO_LARGE
		}
		child = item
		return child, written, OUTPUT_STATUS_OK
	}
	return child, written, OUTPUT_STATUS_OK
}

func output_complex_bits(output Output_Storage,
	count_value Output_Count_Staged, real Float, imaginary Float,
	float_parse Float_Parse_Workspace_Pointer, float_text Float_Text_Workspace_Pointer,
) (_ Output_Count_Staged, status Output_Status) {
	defer func() { Output_Status_Invariants(status, "output_complex_bits.status") }()
	Output_Storage_Invariants(output, "output_complex_bits.output")
	Output_Count_Staged_Invariants(count_value, "output_complex_bits.count_value")
	Float_Invariants(real, "output_complex_bits.real")
	Float_Invariants(imaginary, "output_complex_bits.imaginary")
	Float_Parse_Workspace_Pointer_Invariants(float_parse, "output_complex_bits.float_parse")
	Float_Text_Workspace_Pointer_Invariants(float_text, "output_complex_bits.float_text")
	written := count_value
	defer func() {
		Complex_Output_Count_Invariants(Complex_Output_Count(written),
			"output_complex_bits.written")
	}()
	var write_status Output_Status
	written, write_status = output_text(output, count_value, "(")
	if write_status != OUTPUT_STATUS_OK {
		return written, OUTPUT_STATUS_TOO_LARGE
	}
	written, write_status = output_float_bits(output, written, real, float_parse, float_text)
	if write_status != OUTPUT_STATUS_OK {
		return written, OUTPUT_STATUS_TOO_LARGE
	}
	if uint64(imaginary)&FLOAT_64_SIGN_MASK == bits.WORD_64_MINIMUM {
		written, write_status = output_text(output, written, "+")
		if write_status != OUTPUT_STATUS_OK {
			return written, OUTPUT_STATUS_TOO_LARGE
		}
	}
	written, write_status = output_float_bits(
		output, written, imaginary, float_parse, float_text)
	if write_status != OUTPUT_STATUS_OK {
		return written, OUTPUT_STATUS_TOO_LARGE
	}
	written, status = output_text(output, written, "i)")
	return written, status
}

func output_float_bits(output Output_Storage, count_value Output_Count_Staged, encoding Float,
	float_parse Float_Parse_Workspace_Pointer, float_text Float_Text_Workspace_Pointer,
) (_ Output_Count_Staged, status Output_Status) {
	defer func() { Output_Status_Invariants(status, "output_float_bits.status") }()
	Output_Storage_Invariants(output, "output_float_bits.output")
	Output_Count_Staged_Invariants(count_value, "output_float_bits.count_value")
	Float_Invariants(encoding, "output_float_bits.encoding")
	Float_Parse_Workspace_Pointer_Invariants(float_parse, "output_float_bits.float_parse")
	Float_Text_Workspace_Pointer_Invariants(float_text, "output_float_bits.float_text")
	count := count_value
	defer func() {
		Written_Output_Count_Invariants(
			Written_Output_Count(count), "output_float_bits.written")
	}()
	if float64_nan(encoding) {
		count, status = output_text(output, count, "NaN")
		return count, status
	}
	float_value := (*big.Float)(&float_parse.Values.Result)
	text_workspace := (*big.Float_Text_Workspace)((*Float_Text_Workspace)(float_text))
	// Reset semantics must not discard caller-owned bounded mantissa storage.
	words := float_value.Mantissa.Words
	*float_value = big.Float{}
	float_value.Mantissa.Words = words
	set_status := big.Float_Set_Float_64_Bits(float_value, big.Float_64_Bits(encoding))
	aver.Always(set_status == big.STATUS_OK,
		"A non-NaN binary64 encoding is always a valid shared big float.")
	start := int(Output_Count_Staged_Value(count.Value))
	formatted_count, text_status := big.Float_Text_Into(big.Text(output[start:]), float_value,
		big.FLOAT_TEXT_FORMAT_DECIMAL_GENERAL, big.FLOAT_TEXT_PRECISION_MINIMUM,
		text_workspace)
	if text_status == big.STATUS_DESTINATION_TOO_SMALL {
		return count, OUTPUT_STATUS_TOO_LARGE
	}
	aver.Always(text_status == big.STATUS_OK,
		"Validated shortest binary64 formatting has no other failure.")
	count = Output_Count_Staged{
		Value: Output_Count_Proof_Stored(Output_Count_Staged_Value(count.Value) +
			Output_Count_Staged_Value(formatted_count)),
	}
	return count, OUTPUT_STATUS_OK
}

func output_bytes(output Output_Storage, count_value Output_Count_Staged, source Text,
) (_ Output_Count_Staged, status Output_Status) {
	defer func() { Output_Status_Invariants(status, "output_bytes.status") }()
	Output_Storage_Invariants(output, "output_bytes.output")
	Output_Count_Staged_Invariants(count_value, "output_bytes.count_value")
	Text_Invariants(source, "output_bytes.source")
	written := count_value
	defer func() { Output_Count_Staged_Invariants(written, "output_bytes.written") }()
	count := int(Output_Count_Staged_Value(count_value.Value))
	if count > len(output)-len(source) {
		return count_value, OUTPUT_STATUS_TOO_LARGE
	}
	copy(output[count:], source)
	written = Output_Count_Staged{
		Value: Output_Count_Proof_Stored(Output_Count_Staged_Value(count_value.Value) +
			Output_Count_Staged_Value(len(source))),
	}
	return written, OUTPUT_STATUS_OK
}

// Float_Parse_Unvalidated_Pointer points at caller float parser scratch before validation.
type Float_Parse_Unvalidated_Pointer *Float_Parse_Unvalidated

// Float_Parse_Unvalidated_Pointer_Invariants checks present caller parser scratch.
func Float_Parse_Unvalidated_Pointer_Invariants(
	value Float_Parse_Unvalidated_Pointer, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Float_Parse_Unvalidated_Invariants(*value, namespace)
}

// Float_Text_Unvalidated_Pointer points at caller float formatter scratch before validation.
type Float_Text_Unvalidated_Pointer *Float_Text_Unvalidated

// Float_Text_Unvalidated_Pointer_Invariants checks present caller formatter scratch.
func Float_Text_Unvalidated_Pointer_Invariants(
	value Float_Text_Unvalidated_Pointer, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Float_Text_Unvalidated_Invariants(*value, namespace)
}

// Float_Parse_Workspace_Pointer points at validated float parser scratch.
type Float_Parse_Workspace_Pointer *Float_Parse_Workspace

// Float_Parse_Workspace_Pointer_Invariants checks validated parser scratch.
func Float_Parse_Workspace_Pointer_Invariants(
	value Float_Parse_Workspace_Pointer, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Float_Parse_Workspace_Invariants(*value, namespace)
}

// Float_Text_Workspace_Pointer points at validated float formatter scratch.
type Float_Text_Workspace_Pointer *Float_Text_Workspace

// Float_Text_Workspace_Pointer_Invariants checks validated formatter scratch.
func Float_Text_Workspace_Pointer_Invariants(
	value Float_Text_Workspace_Pointer, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Float_Text_Workspace_Invariants(*value, namespace)
}

// Workspace_Literal_Count_Storage_Pointer points at one workspace cursor or record.
type Workspace_Literal_Count_Storage_Pointer *Workspace_Literal_Count_Storage

// Workspace_Literal_Count_Storage_Pointer_Invariants checks present workspace storage.
func Workspace_Literal_Count_Storage_Pointer_Invariants(
	value Workspace_Literal_Count_Storage_Pointer, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Workspace_Literal_Count_Storage_Invariants(*value, namespace)
}

// Workspace_Generated_Count_Storage_Pointer points at one workspace cursor or record.
type Workspace_Generated_Count_Storage_Pointer *Workspace_Generated_Count_Storage

// Workspace_Generated_Count_Storage_Pointer_Invariants checks present workspace storage.
func Workspace_Generated_Count_Storage_Pointer_Invariants(
	value Workspace_Generated_Count_Storage_Pointer, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Workspace_Generated_Count_Storage_Invariants(*value, namespace)
}

// Written_Generated_Count is the derived cursor one escape step continues from.
type Written_Generated_Count Workspace_Generated_Count_Storage

// Written_Generated_Count_Invariants keeps the byte its opening step already wrote.
func Written_Generated_Count_Invariants(value Written_Generated_Count,
	namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(utf8.CHARACTER_SIZE_MINIMUM),
			uint16(GENERATED_SIZE_MAXIMUM-utf8.CHARACTER_SIZE_MINIMUM)).
		Ensure()
}

// Written_Generated_Count_Pointer keeps that cursor mutable for its caller.
type Written_Generated_Count_Pointer *Written_Generated_Count

// Written_Generated_Count_Pointer_Invariants checks the present cursor.
func Written_Generated_Count_Pointer_Invariants(
	value Written_Generated_Count_Pointer, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Written_Generated_Count_Invariants(*value, namespace)
}

// Escaped_Generated_Count is the derived cursor one escape starts from.
type Escaped_Generated_Count Workspace_Generated_Count_Storage

// Escaped_Generated_Count_Invariants leaves room for the text one escape derives.
func Escaped_Generated_Count_Invariants(value Escaped_Generated_Count,
	namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(bytes.SLICE_SIZE_MINIMUM),
			uint16(GENERATED_SIZE_MAXIMUM-SOURCE_SIZE_MAXIMUM)).
		Ensure()
}

// Escaped_Generated_Count_Pointer keeps that cursor mutable for its caller.
type Escaped_Generated_Count_Pointer *Escaped_Generated_Count

// Escaped_Generated_Count_Pointer_Invariants checks the present cursor.
func Escaped_Generated_Count_Pointer_Invariants(
	value Escaped_Generated_Count_Pointer, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Escaped_Generated_Count_Invariants(*value, namespace)
}

// Workspace_Frame_Count_Storage_Pointer points at one workspace cursor or record.
type Workspace_Frame_Count_Storage_Pointer *Workspace_Frame_Count_Storage

// Workspace_Frame_Count_Storage_Pointer_Invariants checks present workspace storage.
func Workspace_Frame_Count_Storage_Pointer_Invariants(
	value Workspace_Frame_Count_Storage_Pointer, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Workspace_Frame_Count_Storage_Invariants(*value, namespace)
}

// Workspace_Variable_Count_Storage_Pointer points at one workspace cursor or record.
type Workspace_Variable_Count_Storage_Pointer *Workspace_Variable_Count_Storage

// Workspace_Variable_Count_Storage_Pointer_Invariants checks present workspace storage.
func Workspace_Variable_Count_Storage_Pointer_Invariants(
	value Workspace_Variable_Count_Storage_Pointer, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Workspace_Variable_Count_Storage_Invariants(*value, namespace)
}

// Workspace_Scope_Base_Storage_Pointer points at one workspace cursor or record.
type Workspace_Scope_Base_Storage_Pointer *Workspace_Scope_Base_Storage

// Workspace_Scope_Base_Storage_Pointer_Invariants checks present workspace storage.
func Workspace_Scope_Base_Storage_Pointer_Invariants(
	value Workspace_Scope_Base_Storage_Pointer, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Workspace_Scope_Base_Storage_Invariants(*value, namespace)
}

// Workspace_Diagnostic_Storage_Pointer points at one workspace cursor or record.
type Workspace_Diagnostic_Storage_Pointer *Workspace_Diagnostic_Storage

// Workspace_Diagnostic_Storage_Pointer_Invariants checks present workspace storage.
func Workspace_Diagnostic_Storage_Pointer_Invariants(
	value Workspace_Diagnostic_Storage_Pointer, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Workspace_Diagnostic_Storage_Invariants(*value, namespace)
}

func frame_advance(
	frames Frames, frame_count Bound_Frame_Count_Pointer, variables Variables,
	variable_count Bound_Variable_Count_Pointer,
	scope_base Workspace_Scope_Base_Storage_Pointer, program_value Program_Validated,
	frame_state Frame_Storage_Pointer,
) (advanced Frame_Advanced) {
	defer func() { Frame_Advanced_Invariants(advanced, "frame_advance.advanced") }()
	Frames_Invariants(frames, "frame_advance.frames")
	Bound_Frame_Count_Pointer_Invariants(frame_count, "frame_advance.frame_count")
	Variables_Invariants(variables, "frame_advance.variables")
	Bound_Variable_Count_Pointer_Invariants(variable_count, "frame_advance.variable_count")
	Workspace_Scope_Base_Storage_Pointer_Invariants(scope_base, "frame_advance.scope_base")
	Program_Validated_Invariants(program_value, "frame_advance.program_value")
	Frame_Storage_Pointer_Invariants(frame_state, "frame_advance.frame_state")
	frame := &frame_state.Value
	if frame.Kind == FRAME_RANGE {
		ranged_base := (*Ranged_Scope_Base)((*Workspace_Scope_Base_Storage)(scope_base))
		scheduled := (*Scheduled_Frame_Count)((*Bound_Frame_Count)(frame_count))
		range_schedule(frames, scheduled, variables, variable_count, ranged_base,
			Bound_Range_Program(program_value),
			(*Range_Frame_Storage)((*Frame_Storage)(frame_state)))
		return true
	}
	if frame.Reference == NO_NODE {
		frame_pop(frames, frame_count, variable_count, scope_base)
		return true
	}
	return false
}

// Frame_Advanced reports a frame that scheduled or finished without executing a node.
type Frame_Advanced bool

// Frame_Advanced_Invariants covers both scheduling outcomes.
func Frame_Advanced_Invariants(value Frame_Advanced, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Traversal frame advanced without executing a node.").
		Ensure()
}

func integer_format_base(verb Integer_Format_Verb) (base Integer_Format_Base,
	_ Integer_Format_Uppercase,
) {
	defer func() { Integer_Format_Base_Invariants(base, "integer_format_base.base") }()
	Integer_Format_Verb_Invariants(verb, "integer_format_base.verb")
	uppercase := Integer_Format_Uppercase(false)
	defer func() {
		Integer_Format_Uppercase_Invariants(uppercase, "integer_format_base.uppercase")
	}()
	base = INTEGER_FORMAT_BASE_DECIMAL
	if verb == 'b' {
		base = INTEGER_FORMAT_BASE_BINARY
	} else if verb == 'o' {
		base = INTEGER_FORMAT_BASE_OCTAL
	} else if verb == 'x' {
		base = INTEGER_FORMAT_BASE_HEXADECIMAL
	} else if verb == 'X' {
		base = INTEGER_FORMAT_BASE_HEXADECIMAL
		uppercase = true
	}
	return base, uppercase
}

// Strictly increasing names cannot repeat, thus they skip the pairwise duplicate scan.
// OBJECT_FIELD_INCREMENT steps to the next object member.
const OBJECT_FIELD_INCREMENT = 1

func object_names_sorted(field_values Bounded_Fields) (sorted Object_Names_Sorted) {
	defer func() {
		Object_Names_Sorted_Invariants(sorted, "object_names_sorted.sorted")
	}()
	Bounded_Fields_Invariants(field_values, "object_names_sorted.field_values")
	fields := Fields_Storage_Value(field_values.Value)
	for index := OBJECT_FIELD_INCREMENT; index < len(fields); index++ {
		previous := fields[index-OBJECT_FIELD_INCREMENT].Name
		current := fields[index].Name
		position := SOURCE_SIZE_MINIMUM
		for position < len(previous) {
			if position == len(current) {
				return false
			}
			if previous[position] != current[position] {
				break
			}
			position++
		}
		if position == len(previous) {
			if position == len(current) {
				return false
			}
			continue
		}
		if previous[position] > current[position] {
			return false
		}
	}
	return true
}

// Object_Names_Sorted reports object member names in strictly increasing byte order.
type Object_Names_Sorted bool

// Object_Names_Sorted_Invariants covers sorted and unsorted member names.
func Object_Names_Sorted_Invariants(value Object_Names_Sorted, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Object member names ascend strictly.").
		Ensure()
}

// Nil_Value is one caller value a constructor built.
type Nil_Value Value

// Nil_Value_Invariants fixes the selector and the empty payloads of a absent constructor.
func Nil_Value_Invariants(value Nil_Value, namespace aver.Namespace) {
	aver.Always(
		uint8(value.Kind) == uint8(VALUE_NIL), "A absent constructor selects its own kind.",
	)
	aver.Always(uint64(value.Real) == bits.WORD_64_MINIMUM,
		"A absent constructor leaves the primary word empty.")
	aver.Always(uint64(value.Imaginary) == bits.WORD_64_MINIMUM,
		"A absent constructor leaves the imaginary word empty.")
	aver.Always(
		len(value.Text) == OUTPUT_SIZE_MINIMUM, "A absent constructor leaves text empty.",
	)
	aver.Always(len(value.Values) == bytes.SLICE_SIZE_MINIMUM,
		"A absent constructor leaves the sequence empty.")
	aver.Always(len(value.Fields) == bytes.SLICE_SIZE_MINIMUM,
		"A absent constructor leaves the members empty.")
}

// Boolean_Value is one caller value a constructor built.
type Boolean_Value Value

// Boolean_Value_Invariants fixes the selector and the empty payloads of a truth constructor.
func Boolean_Value_Invariants(value Boolean_Value, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint64(uint64(value.Real), bits.WORD_64_MINIMUM, bits.CARRY_MAXIMUM).
		Ensure()
	aver.Always(uint8(value.Kind) == uint8(VALUE_BOOLEAN),
		"A truth constructor selects its own kind.")
	aver.Always(uint64(value.Imaginary) == bits.WORD_64_MINIMUM,
		"A truth constructor leaves the imaginary word empty.")
	aver.Always(
		len(value.Text) == OUTPUT_SIZE_MINIMUM, "A truth constructor leaves text empty.",
	)
	aver.Always(len(value.Values) == bytes.SLICE_SIZE_MINIMUM,
		"A truth constructor leaves the sequence empty.")
	aver.Always(len(value.Fields) == bytes.SLICE_SIZE_MINIMUM,
		"A truth constructor leaves the members empty.")
}

// Integer_Value is one caller value a constructor built.
type Integer_Value Value

// Integer_Value_Invariants fixes the selector and the empty payloads of a signed constructor.
func Integer_Value_Invariants(value Integer_Value, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value.Real), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
	aver.Always(uint8(value.Kind) == uint8(VALUE_INTEGER),
		"A signed constructor selects its own kind.")
	aver.Always(uint64(value.Imaginary) == bits.WORD_64_MINIMUM,
		"A signed constructor leaves the imaginary word empty.")
	aver.Always(
		len(value.Text) == OUTPUT_SIZE_MINIMUM, "A signed constructor leaves text empty.",
	)
	aver.Always(len(value.Values) == bytes.SLICE_SIZE_MINIMUM,
		"A signed constructor leaves the sequence empty.")
	aver.Always(len(value.Fields) == bytes.SLICE_SIZE_MINIMUM,
		"A signed constructor leaves the members empty.")
}

// Unsigned_Value is one caller value a constructor built.
type Unsigned_Value Value

// Unsigned_Value_Invariants fixes the selector and the empty payloads of a unsigned constructor.
func Unsigned_Value_Invariants(value Unsigned_Value, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value.Real), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
	aver.Always(uint8(value.Kind) == uint8(VALUE_UNSIGNED),
		"A unsigned constructor selects its own kind.")
	aver.Always(uint64(value.Imaginary) == bits.WORD_64_MINIMUM,
		"A unsigned constructor leaves the imaginary word empty.")
	aver.Always(
		len(value.Text) == OUTPUT_SIZE_MINIMUM, "A unsigned constructor leaves text empty.",
	)
	aver.Always(len(value.Values) == bytes.SLICE_SIZE_MINIMUM,
		"A unsigned constructor leaves the sequence empty.")
	aver.Always(len(value.Fields) == bytes.SLICE_SIZE_MINIMUM,
		"A unsigned constructor leaves the members empty.")
}

// Float_Value is one caller value a constructor built.
type Float_Value Value

// Float_Value_Invariants fixes the selector and the empty payloads of a binary64 constructor.
func Float_Value_Invariants(value Float_Value, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value.Real), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
	aver.Always(uint8(value.Kind) == uint8(VALUE_FLOAT),
		"A binary64 constructor selects its own kind.")
	aver.Always(uint64(value.Imaginary) == bits.WORD_64_MINIMUM,
		"A binary64 constructor leaves the imaginary word empty.")
	aver.Always(
		len(value.Text) == OUTPUT_SIZE_MINIMUM, "A binary64 constructor leaves text empty.",
	)
	aver.Always(len(value.Values) == bytes.SLICE_SIZE_MINIMUM,
		"A binary64 constructor leaves the sequence empty.")
	aver.Always(len(value.Fields) == bytes.SLICE_SIZE_MINIMUM,
		"A binary64 constructor leaves the members empty.")
}

// Complex_Value is one caller value a constructor built.
type Complex_Value Value

// Complex_Value_Invariants fixes the selector and the empty payloads of a complex constructor.
func Complex_Value_Invariants(value Complex_Value, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value.Real), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Imaginary), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
	aver.Always(uint8(value.Kind) == uint8(VALUE_COMPLEX),
		"A complex constructor selects its own kind.")
	aver.Always(
		len(value.Text) == OUTPUT_SIZE_MINIMUM, "A complex constructor leaves text empty.",
	)
	aver.Always(len(value.Values) == bytes.SLICE_SIZE_MINIMUM,
		"A complex constructor leaves the sequence empty.")
	aver.Always(len(value.Fields) == bytes.SLICE_SIZE_MINIMUM,
		"A complex constructor leaves the members empty.")
}

// Text_Value is one caller value a constructor built.
type Text_Value Value

// Text_Value_Invariants fixes the selector and the empty payloads of a text constructor.
func Text_Value_Invariants(value Text_Value, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value.Text), OUTPUT_SIZE_MINIMUM,
			OUTPUT_SIZE_UNVALIDATED_MAXIMUM-utf8.CHARACTER_SIZE_MINIMUM).
		Ensure()
	aver.Always(
		uint8(value.Kind) == uint8(VALUE_TEXT), "A text constructor selects its own kind.",
	)
	aver.Always(uint64(value.Real) == bits.WORD_64_MINIMUM,
		"A text constructor leaves the primary word empty.")
	aver.Always(uint64(value.Imaginary) == bits.WORD_64_MINIMUM,
		"A text constructor leaves the imaginary word empty.")
	aver.Always(len(value.Values) == bytes.SLICE_SIZE_MINIMUM,
		"A text constructor leaves the sequence empty.")
	aver.Always(len(value.Fields) == bytes.SLICE_SIZE_MINIMUM,
		"A text constructor leaves the members empty.")
}

// Sequence_Value is one caller value a constructor built.
type Sequence_Value Value

// Sequence_Value_Invariants fixes the selector and the empty payloads of a sequence constructor.
func Sequence_Value_Invariants(value Sequence_Value, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value.Values), bytes.SLICE_SIZE_MINIMUM,
			VALUE_COUNT_UNVALIDATED_MAXIMUM).
		Ensure()
	aver.Always(uint8(value.Kind) == uint8(VALUE_SEQUENCE),
		"A sequence constructor selects its own kind.")
	aver.Always(uint64(value.Real) == bits.WORD_64_MINIMUM,
		"A sequence constructor leaves the primary word empty.")
	aver.Always(uint64(value.Imaginary) == bits.WORD_64_MINIMUM,
		"A sequence constructor leaves the imaginary word empty.")
	aver.Always(
		len(value.Text) == OUTPUT_SIZE_MINIMUM, "A sequence constructor leaves text empty.",
	)
	aver.Always(len(value.Fields) == bytes.SLICE_SIZE_MINIMUM,
		"A sequence constructor leaves the members empty.")
}

// Object_Value is one caller value a constructor built.
type Object_Value Value

// Object_Value_Invariants fixes the selector and the empty payloads of a object constructor.
func Object_Value_Invariants(value Object_Value, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value.Fields), bytes.SLICE_SIZE_MINIMUM,
			VALUE_COUNT_UNVALIDATED_MAXIMUM).
		Ensure()
	aver.Always(uint8(value.Kind) == uint8(VALUE_OBJECT),
		"A object constructor selects its own kind.")
	aver.Always(uint64(value.Real) == bits.WORD_64_MINIMUM,
		"A object constructor leaves the primary word empty.")
	aver.Always(uint64(value.Imaginary) == bits.WORD_64_MINIMUM,
		"A object constructor leaves the imaginary word empty.")
	aver.Always(
		len(value.Text) == OUTPUT_SIZE_MINIMUM, "A object constructor leaves text empty.",
	)
	aver.Always(len(value.Values) == bytes.SLICE_SIZE_MINIMUM,
		"A object constructor leaves the sequence empty.")
}

// Map_Value is one caller value a constructor built.
type Map_Value Value

// Map_Value_Invariants fixes the selector and the empty payloads of a map constructor.
func Map_Value_Invariants(value Map_Value, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value.Fields), bytes.SLICE_SIZE_MINIMUM,
			VALUE_COUNT_UNVALIDATED_MAXIMUM).
		Ensure()
	aver.Always(
		uint8(value.Kind) == uint8(VALUE_MAP), "A map constructor selects its own kind.",
	)
	aver.Always(uint64(value.Real) == bits.WORD_64_MINIMUM,
		"A map constructor leaves the primary word empty.")
	aver.Always(uint64(value.Imaginary) == bits.WORD_64_MINIMUM,
		"A map constructor leaves the imaginary word empty.")
	aver.Always(len(value.Text) == OUTPUT_SIZE_MINIMUM, "A map constructor leaves text empty.")
	aver.Always(len(value.Values) == bytes.SLICE_SIZE_MINIMUM,
		"A map constructor leaves the sequence empty.")
}

// Function_Value is one caller value a constructor built.
type Function_Value Value

// Function_Value_Invariants fixes the selector and the empty payloads of a callable constructor.
func Function_Value_Invariants(value Function_Value, namespace aver.Namespace) {
	aver.Always(uint8(value.Kind) == uint8(VALUE_FUNCTION),
		"A callable constructor selects its own kind.")
	aver.Always(uint64(value.Real) == bits.WORD_64_MINIMUM,
		"A callable constructor leaves the primary word empty.")
	aver.Always(uint64(value.Imaginary) == bits.WORD_64_MINIMUM,
		"A callable constructor leaves the imaginary word empty.")
	aver.Always(
		len(value.Text) == OUTPUT_SIZE_MINIMUM, "A callable constructor leaves text empty.",
	)
	aver.Always(len(value.Values) == bytes.SLICE_SIZE_MINIMUM,
		"A callable constructor leaves the sequence empty.")
	aver.Always(len(value.Fields) == bytes.SLICE_SIZE_MINIMUM,
		"A callable constructor leaves the members empty.")
}

// Workspace_Cursor_Unvalidated_Pointer points at one caller cursor that execution resets.
type Workspace_Cursor_Unvalidated_Pointer *Workspace_Cursor_Unvalidated

// Workspace_Cursor_Unvalidated_Pointer_Invariants checks one present caller cursor.
func Workspace_Cursor_Unvalidated_Pointer_Invariants(
	value Workspace_Cursor_Unvalidated_Pointer, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Workspace_Cursor_Unvalidated_Invariants(*value, namespace)
}

// Workspace_Generated_Cursor_Unvalidated_Pointer points at one caller cursor that execution resets.
type Workspace_Generated_Cursor_Unvalidated_Pointer *Workspace_Generated_Cursor_Unvalidated

// Workspace_Generated_Cursor_Unvalidated_Pointer_Invariants checks one present caller cursor.
func Workspace_Generated_Cursor_Unvalidated_Pointer_Invariants(
	value Workspace_Generated_Cursor_Unvalidated_Pointer, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Workspace_Generated_Cursor_Unvalidated_Invariants(*value, namespace)
}

// Workspace_Frame_Cursor_Unvalidated_Pointer points at one caller cursor that execution resets.
type Workspace_Frame_Cursor_Unvalidated_Pointer *Workspace_Frame_Cursor_Unvalidated

// Workspace_Frame_Cursor_Unvalidated_Pointer_Invariants checks one present caller cursor.
func Workspace_Frame_Cursor_Unvalidated_Pointer_Invariants(
	value Workspace_Frame_Cursor_Unvalidated_Pointer, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Workspace_Frame_Cursor_Unvalidated_Invariants(*value, namespace)
}

// Workspace_Variable_Cursor_Unvalidated_Pointer points at one caller cursor that execution resets.
type Workspace_Variable_Cursor_Unvalidated_Pointer *Workspace_Variable_Cursor_Unvalidated

// Workspace_Variable_Cursor_Unvalidated_Pointer_Invariants checks one present caller cursor.
func Workspace_Variable_Cursor_Unvalidated_Pointer_Invariants(
	value Workspace_Variable_Cursor_Unvalidated_Pointer, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Workspace_Variable_Cursor_Unvalidated_Invariants(*value, namespace)
}

// Workspace_Scope_Cursor_Unvalidated_Pointer points at one caller cursor that execution resets.
type Workspace_Scope_Cursor_Unvalidated_Pointer *Workspace_Scope_Cursor_Unvalidated

// Workspace_Scope_Cursor_Unvalidated_Pointer_Invariants checks one present caller cursor.
func Workspace_Scope_Cursor_Unvalidated_Pointer_Invariants(
	value Workspace_Scope_Cursor_Unvalidated_Pointer, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Workspace_Scope_Cursor_Unvalidated_Invariants(*value, namespace)
}

// Workspace_Diagnostic_Unvalidated_Pointer points at one caller cursor that execution resets.
type Workspace_Diagnostic_Unvalidated_Pointer *Workspace_Diagnostic_Unvalidated

// Workspace_Diagnostic_Unvalidated_Pointer_Invariants checks one present caller cursor.
func Workspace_Diagnostic_Unvalidated_Pointer_Invariants(
	value Workspace_Diagnostic_Unvalidated_Pointer, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Workspace_Diagnostic_Unvalidated_Invariants(*value, namespace)
}

func execution_refused(diagnostic Workspace_Diagnostic_Storage_Pointer,
	code Execution_Refused_Status,
) (refusal Refusal_Diagnostic) {
	defer func() { Refusal_Diagnostic_Invariants(refusal, "execution_refused.refusal") }()
	Workspace_Diagnostic_Storage_Pointer_Invariants(diagnostic, "execution_refused.diagnostic")
	Execution_Refused_Status_Invariants(code, "execution_refused.code")
	execution_refusal(diagnostic, code)
	refusal = Refusal_Diagnostic(diagnostic.Value)
	return refusal
}

// Source_Header_Unvalidated is one caller source header before its size is checked.
type Source_Header_Unvalidated Source

// Source_Header_Unvalidated_Invariants admits a caller header up to its first refused size.
func Source_Header_Unvalidated_Invariants(
	value Source_Header_Unvalidated, namespace aver.Namespace,
) {
	Source_Storage_Invariants(value.Data, namespace)
}

// Document_Unvalidated is one caller document header before program_valid checks it.
type Document_Unvalidated Document

// Document_Unvalidated_Invariants admits every caller document encoding.
func Document_Unvalidated_Invariants(value Document_Unvalidated, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Root), bits.WORD_16_MINIMUM, bits.WORD_16_MAXIMUM).
		Range_Uint16(uint16(value.Node_Count), bits.WORD_16_MINIMUM, bits.WORD_16_MAXIMUM).
		Range_Uint16(
			uint16(value.First_Template), bits.WORD_16_MINIMUM, bits.WORD_16_MAXIMUM,
		).
		Range_Uint16(
			uint16(value.Template_Count), bits.WORD_16_MINIMUM, bits.WORD_16_MAXIMUM,
		).
		Ensure()
}

// Program_Unvalidated is one caller program before program_valid checks it.
type Program_Unvalidated Program

// Program_Unvalidated_Invariants composes one caller program before program_valid.
func Program_Unvalidated_Invariants(value Program_Unvalidated, namespace aver.Namespace) {
	Source_Header_Unvalidated_Invariants(Source_Header_Unvalidated(value.Source), namespace)
	Document_Unvalidated_Invariants(Document_Unvalidated(value.Document), namespace)
	Syntax_Storage_Invariants(value.Syntax, namespace)
}

// Fields_Proof_Stored separates traversal proof from transient slice headers.
type Fields_Proof_Stored Fields_Storage_Value

// Fields_Proof_Stored_Invariants fixes borrowed-field representation.
func Fields_Proof_Stored_Invariants(value Fields_Proof_Stored, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), bytes.SLICE_SIZE_MINIMUM, VALUE_COUNT_MAXIMUM).
		Ensure()
}

// Evaluation_Frame_Delivery_Pointer carries one caller-owned suspended frame.
type Evaluation_Frame_Delivery_Pointer *Evaluation_Frame

// Evaluation_Frame_Delivery_Pointer_Invariants fixes delivery-frame representation.
func Evaluation_Frame_Delivery_Pointer_Invariants(
	value Evaluation_Frame_Delivery_Pointer, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Evaluation_Frame_Invariants(*value, namespace)
}

// Text_Span_Start_Proof_Stored separates literal proof from transient boundaries.
type Text_Span_Start_Proof_Stored Text_Span_Stored_Start

// Text_Span_Start_Proof_Stored_Invariants fixes literal-start representation.
func Text_Span_Start_Proof_Stored_Invariants(
	value Text_Span_Start_Proof_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(SOURCE_SIZE_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM),
		).
		Ensure()
}

// Text_Span_End_Proof_Stored separates literal proof from transient boundaries.
type Text_Span_End_Proof_Stored Text_Span_Stored_End

// Text_Span_End_Proof_Stored_Invariants fixes literal-end representation.
func Text_Span_End_Proof_Stored_Invariants(
	value Text_Span_End_Proof_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(SOURCE_SIZE_MINIMUM), uint16(SOURCE_SIZE_MAXIMUM),
		).
		Ensure()
}

// Parse_Failure_Status_Proof_Stored separates refusal proof from transient statuses.
type Parse_Failure_Status_Proof_Stored Parse_Failure_Status_Validated_Value

// Parse_Failure_Status_Proof_Stored_Invariants fixes parser-refusal representation.
func Parse_Failure_Status_Proof_Stored_Invariants(
	value Parse_Failure_Status_Proof_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), uint8(PARSE_STATUS_INPUT_INVALID), uint8(
			PARSE_STATUS_CAPACITY_EXCEEDED)).
		Ensure()
}

// Classified_Node_Kind_Proof_Stored separates classification proof from transient kinds.
type Classified_Node_Kind_Proof_Stored Classified_Node_Kind_Value

// Classified_Node_Kind_Proof_Stored_Invariants fixes node-kind representation.
func Classified_Node_Kind_Proof_Stored_Invariants(
	value Classified_Node_Kind_Proof_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Holed_Uint8(
			uint8(value), uint8(NODE_KIND_MINIMUM),
			uint8(NODE_VARIABLE), uint8(NODE_ACTION),
			uint8(NODE_ACTION), uint8(NODE_ACTION)).
		Ensure()
}

// Generated_Start_Proof_Stored separates output proof from transient boundaries.
type Generated_Start_Proof_Stored Generated_Start_Validated_Value

// Generated_Start_Proof_Stored_Invariants fixes generated-start representation.
func Generated_Start_Proof_Stored_Invariants(
	value Generated_Start_Proof_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value), uint16(bytes.SLICE_SIZE_MINIMUM), uint16(
			GENERATED_SIZE_MAXIMUM)).
		Ensure()
}

// Formatted_Integer_Text_Proof_Stored separates conversion proof from transient text.
type Formatted_Integer_Text_Proof_Stored Formatted_Integer_Text_Validated_Value

// Formatted_Integer_Text_Proof_Stored_Invariants fixes formatted-text representation.
func Formatted_Integer_Text_Proof_Stored_Invariants(
	value Formatted_Integer_Text_Proof_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), FORMATTED_INTEGER_TEXT_SIZE_MINIMUM,
			FORMATTED_INTEGER_TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Variable_Chain_Proof_Stored separates lexer proof from transient source slices.
type Variable_Chain_Proof_Stored Variable_Chain_Validated_Value

// Variable_Chain_Proof_Stored_Invariants fixes variable-chain representation.
func Variable_Chain_Proof_Stored_Invariants(
	value Variable_Chain_Proof_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), VARIABLE_NAME_SIZE_MINIMUM, ACTION_CONTENT_SIZE_MAXIMUM).
		Ensure()
}

// Parse_Function_Name_Proof_Stored separates policy proof from transient source slices.
type Parse_Function_Name_Proof_Stored Parse_Function_Name_Validated_Value

// Parse_Function_Name_Proof_Stored_Invariants fixes parser-name representation.
func Parse_Function_Name_Proof_Stored_Invariants(
	value Parse_Function_Name_Proof_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), IDENTIFIER_SIZE_MINIMUM, ACTION_CONTENT_SIZE_MAXIMUM).
		Ensure()
}

// BUILTIN_NAME_SIZE_MAXIMUM is the longest builtin name one call names.
const BUILTIN_NAME_SIZE_MAXIMUM = len("urlquery")

// Function_Name_Proof_Stored separates callback proof from transient source slices.
type Function_Name_Proof_Stored Function_Name_Validated_Value

// Function_Name_Proof_Stored_Invariants fixes callback-name representation.
func Function_Name_Proof_Stored_Invariants(
	value Function_Name_Proof_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), IDENTIFIER_SIZE_MINIMUM, BUILTIN_NAME_SIZE_MAXIMUM).
		Ensure()
}

// LOCATION_REFERENCE_MAXIMUM stands beneath the term the location names.
const LOCATION_REFERENCE_MAXIMUM = NODE_REFERENCE_MAXIMUM - SCALAR_NODE_COUNT

// Node_Reference_Proof_Stored separates location proof from transient arena slots.
type Node_Reference_Proof_Stored Node_Reference

// Node_Reference_Proof_Stored_Invariants fixes location-reference representation.
func Node_Reference_Proof_Stored_Invariants(
	value Node_Reference_Proof_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value), uint16(COMMAND_NODE_COUNT_MINIMUM), uint16(
			LOCATION_REFERENCE_MAXIMUM)).
		Ensure()
}

// Execution_Failure_Position_Proof_Stored separates diagnostic proof from transient positions.
type Execution_Failure_Position_Proof_Stored Execution_Failure_Position_Value

// Execution_Failure_Position_Proof_Stored_Invariants fixes diagnostic position representation.
func Execution_Failure_Position_Proof_Stored_Invariants(
	value Execution_Failure_Position_Proof_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(SOURCE_SIZE_MINIMUM), uint16(SOURCE_SIZE_MAXIMUM),
		).
		Ensure()
}

// Action_Scan_End_Proof_Stored separates scan proof from transient boundaries.
type Action_Scan_End_Proof_Stored Action_Scan_End_Validated_Value

// Action_Scan_End_Proof_Stored_Invariants fixes scan boundary representation.
func Action_Scan_End_Proof_Stored_Invariants(
	value Action_Scan_End_Proof_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value), uint16(CONSTANT_END_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Ensure()
}

// Classified_Token_Kind_Proof_Stored separates classification proof from transient kinds.
type Classified_Token_Kind_Proof_Stored Classified_Token_Kind_Value

// Classified_Token_Kind_Proof_Stored_Invariants fixes token-kind representation.
func Classified_Token_Kind_Proof_Stored_Invariants(
	value Classified_Token_Kind_Proof_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint8(
			uint8(value), uint8(TOKEN_KIND_MINIMUM), uint8(TOKEN_RIGHT_PAREN)).
		Ensure()
}

// Number_Text_Proof_Stored separates numeric proof from transient source slices.
type Number_Text_Proof_Stored Number_Text_Validated_Value

// Number_Text_Proof_Stored_Invariants fixes numeric-text representation.
func Number_Text_Proof_Stored_Invariants(value Number_Text_Proof_Stored, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), IDENTIFIER_SIZE_MINIMUM, ACTION_CONTENT_SIZE_MAXIMUM).
		Ensure()
}

// Failure_Status_Proof_Stored separates refusal proof from transient statuses.
type Failure_Status_Proof_Stored Failure_Status_Validated_Value

// Failure_Status_Proof_Stored_Invariants fixes execution refusal representation.
func Failure_Status_Proof_Stored_Invariants(
	value Failure_Status_Proof_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint8(
			uint8(value), uint8(STATUS_OUTPUT_TOO_SMALL), uint8(STATUS_LIMIT_EXCEEDED),
		).
		Ensure()
}

// Parenthesis_Depth_Proof_Stored separates parser proof from transient depth.
type Parenthesis_Depth_Proof_Stored Parenthesis_Depth_Tracked_Value

// Parenthesis_Depth_Proof_Stored_Invariants fixes parenthesis-depth representation.
func Parenthesis_Depth_Proof_Stored_Invariants(
	value Parenthesis_Depth_Proof_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value), uint16(PARENTHESIS_DEPTH_MINIMUM), uint16(
			PARENTHESIS_DEPTH_MAXIMUM)).
		Ensure()
}

// Pipeline_Boundary_Proof_Stored separates token proof from transient boundaries.
type Pipeline_Boundary_Proof_Stored Pipeline_Boundary_Validated_Value

// Pipeline_Boundary_Proof_Stored_Invariants fixes pipeline-boundary representation.
func Pipeline_Boundary_Proof_Stored_Invariants(
	value Pipeline_Boundary_Proof_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(SOURCE_SIZE_MINIMUM), uint16(SOURCE_SIZE_MAXIMUM),
		).
		Ensure()
}

// Parsed_Float_Proof_Stored separates conversion proof from transient encodings.
type Parsed_Float_Proof_Stored Parsed_Float_Validated_Value

// Parsed_Float_Proof_Stored_Invariants fixes binary64 representation.
func Parsed_Float_Proof_Stored_Invariants(
	value Parsed_Float_Proof_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Holed_Uint64(
			uint64(value), bits.WORD_64_MINIMUM,
			FLOAT_BITS_PARSED_MAXIMUM, FLOAT_BITS_SUBNORMAL_HOLE,
			FLOAT_BITS_SUBNORMAL_NEXT, FLOAT_BITS_SUBNORMAL_NEXT).
		Ensure()
}

// Pipeline_Root_Proof_Stored separates root proof shape from transient references.
type Pipeline_Root_Proof_Stored Pipeline_Root_Validated_Value

// Pipeline_Root_Proof_Stored_Invariants accepts absent output or one typed reference.
func Pipeline_Root_Proof_Stored_Invariants(
	value Pipeline_Root_Proof_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Holed_Uint16(
			uint16(value), uint16(NODE_REFERENCE_MINIMUM),
			uint16(PIPELINE_REFERENCE_MAXIMUM), uint16(ROOT_LIST_HOLE),
			uint16(TERM_ACTION_HOLE), uint16(TERM_ACTION_HOLE)).
		Ensure()
}

// TERM_REFERENCE_MAXIMUM leaves the node that follows one pipeline term.
const TERM_REFERENCE_MAXIMUM = NODE_REFERENCE_MAXIMUM - NODE_COUNT_INCREMENT

// Pipeline_Term_Proof_Stored separates term proof shape from transient references.
type Pipeline_Term_Proof_Stored Pipeline_Term_Validated_Value

// Pipeline_Term_Proof_Stored_Invariants accepts absent output or one typed reference.
func Pipeline_Term_Proof_Stored_Invariants(
	value Pipeline_Term_Proof_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(COMMAND_NODE_COUNT_MINIMUM),
			uint16(TERM_REFERENCE_MAXIMUM)).
		Ensure()
}

// Template_Reference_Proof_Stored separates definition proof shape from transient references.
type Template_Reference_Proof_Stored Template_Reference_Validated_Value

// Template_Reference_Proof_Stored_Invariants accepts absent output or one typed reference.
func Template_Reference_Proof_Stored_Invariants(
	value Template_Reference_Proof_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value), uint16(ACTION_NODE_COUNT_MINIMUM),
			uint16(DOCUMENT_FIRST_TEMPLATE_REFERENCE_MAXIMUM)).
		Ensure()
}

// DECLARATION_REFERENCE_MAXIMUM is the latest name one declaration reference can hold.
const DECLARATION_REFERENCE_MAXIMUM = FIRST_DECLARATION_REFERENCE_MAXIMUM

// Declaration_Reference_Proof_Stored separates binding proof shape from transient references.
type Declaration_Reference_Proof_Stored Declaration_Reference_Validated_Value

// Declaration_Reference_Proof_Stored_Invariants accepts absent output or one typed reference.
func Declaration_Reference_Proof_Stored_Invariants(
	value Declaration_Reference_Proof_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value), uint16(COMMAND_NODE_COUNT_MINIMUM), uint16(
			DECLARATION_REFERENCE_MAXIMUM)).
		Ensure()
}

// Argument_Count_Proof_Stored separates callback proof shape from transient progress.
type Argument_Count_Proof_Stored Argument_Count_Tracked_Value

// Argument_Count_Proof_Stored_Invariants accepts absent output or one typed count.
func Argument_Count_Proof_Stored_Invariants(
	value Argument_Count_Proof_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value), uint16(ARGUMENT_COUNT_MINIMUM), uint16(
			ARGUMENT_COUNT_MAXIMUM)).
		Ensure()
}

// Pipeline_Second_Declaration_Storage owns one optional scratch token.
type Pipeline_Second_Declaration_Storage struct {
	// Value prevents a stale scalar proof from escaping parser storage.
	Value Pipeline_Second_Declaration_Stored
}

// Pipeline_Second_Declaration_Storage_Invariants fixes second-token ownership.
func Pipeline_Second_Declaration_Storage_Invariants(
	value Pipeline_Second_Declaration_Storage, namespace aver.Namespace,
) {
	Pipeline_Second_Declaration_Stored_Invariants(value.Value, namespace)
}

// Pipeline_Second_Declaration_Stored separates scratch shape from an optional token.
type Pipeline_Second_Declaration_Stored Pipeline_Second_Declaration

// Pipeline_Second_Declaration_Stored_Invariants fixes second-token representation.
func Pipeline_Second_Declaration_Stored_Invariants(
	value Pipeline_Second_Declaration_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value.Kind), uint8(TOKEN_KIND_MINIMUM), uint8(TOKEN_VARIABLE)).
		Range_Holed_Uint16(
			uint16(value.Start), uint16(SOURCE_SIZE_MINIMUM),
			uint16(SECOND_DECLARATION_START_MAXIMUM), uint16(ROOT_LIST_HOLE),
			uint16(TERM_ACTION_HOLE), uint16(TERM_ACTION_HOLE)).
		Range_Holed_Uint16(
			uint16(value.End), uint16(SOURCE_SIZE_MINIMUM),
			uint16(SECOND_DECLARATION_END_MAXIMUM),
			uint16(ROOT_LIST_HOLE), uint16(TERM_ACTION_HOLE),
			uint16(TERM_ACTION_HOLE)).
		Ensure()
}

// Declaration_References_Fields carries raw evaluator binding references.
type Declaration_References_Fields struct {
	// First retains the range index identity.
	First Declaration_First_Reference_Storage
	// Second retains the range value identity.
	Second Declaration_Second_Reference_Storage
}

// Declaration_References_Fields_Invariants bounds hostile binding references.
func Declaration_References_Fields_Invariants(
	value Declaration_References_Fields, namespace aver.Namespace,
) {
	Declaration_First_Reference_Storage_Invariants(value.First, namespace)
	Declaration_Second_Reference_Storage_Invariants(value.Second, namespace)
}

// Declaration_First_Reference_Storage owns one optional index binding.
type Declaration_First_Reference_Storage struct {
	// Value prevents stale reference proof from escaping evaluator storage.
	Value Declaration_First_Reference_Stored
}

// Declaration_First_Reference_Storage_Invariants fixes index-binding ownership.
func Declaration_First_Reference_Storage_Invariants(
	value Declaration_First_Reference_Storage, namespace aver.Namespace,
) {
	Declaration_First_Reference_Stored_Invariants(value.Value, namespace)
}

// Declaration_First_Reference_Stored separates binding shape from an optional index.
type Declaration_First_Reference_Stored Declaration_First_Reference

// Declaration_First_Reference_Stored_Invariants fixes index-binding representation.
func Declaration_First_Reference_Stored_Invariants(
	value Declaration_First_Reference_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Holed_Uint16(uint16(value), uint16(NODE_REFERENCE_MINIMUM),
			uint16(FIRST_DECLARATION_REFERENCE_MAXIMUM), uint16(ROOT_LIST_HOLE),
			uint16(TERM_ACTION_HOLE), uint16(TERM_PIPELINE_HOLE)).
		Ensure()
}

// Declaration_Second_Reference_Storage owns one optional value binding.
type Declaration_Second_Reference_Storage struct {
	// Value prevents stale reference proof from escaping evaluator storage.
	Value Declaration_Second_Reference_Stored
}

// Declaration_Second_Reference_Storage_Invariants fixes value-binding ownership.
func Declaration_Second_Reference_Storage_Invariants(
	value Declaration_Second_Reference_Storage, namespace aver.Namespace,
) {
	Declaration_Second_Reference_Stored_Invariants(value.Value, namespace)
}

// Declaration_Second_Reference_Stored separates binding shape from an optional value.
type Declaration_Second_Reference_Stored Declaration_Second_Reference

// Declaration_Second_Reference_Stored_Invariants fixes value-binding representation.
func Declaration_Second_Reference_Stored_Invariants(
	value Declaration_Second_Reference_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Holed_Uint16(uint16(value), uint16(NODE_REFERENCE_MINIMUM),
			uint16(SECOND_DECLARATION_REFERENCE_MAXIMUM), uint16(ROOT_LIST_HOLE),
			uint16(TERM_ACTION_HOLE), uint16(TERM_PIPELINE_HOLE)).
		Ensure()
}

// Identifier_Search_Fields carries raw lexer search boundaries.
type Identifier_Search_Fields struct {
	// Position retains candidate character start.
	Position Identifier_Search_Position_Storage
	// End retains action content termination.
	End Identifier_Search_End_Storage
}

// Identifier_Search_Fields_Invariants bounds hostile lexer search boundaries.
func Identifier_Search_Fields_Invariants(value Identifier_Search_Fields, namespace aver.Namespace,
) {
	Identifier_Search_Position_Storage_Invariants(value.Position, namespace)
	Identifier_Search_End_Storage_Invariants(value.End, namespace)
}

// Identifier_Search_Position_Storage owns transient lexer progress.
type Identifier_Search_Position_Storage struct {
	// Value prevents a stale scalar proof from escaping lexer storage.
	Value Identifier_Search_Position_Stored
}

// Identifier_Search_Position_Storage_Invariants fixes progress ownership.
func Identifier_Search_Position_Storage_Invariants(
	value Identifier_Search_Position_Storage, namespace aver.Namespace,
) {
	Identifier_Search_Position_Stored_Invariants(value.Value, namespace)
}

// Identifier_Search_Position_Stored separates search shape from transient progress.
type Identifier_Search_Position_Stored Identifier_Search_Position

// Identifier_Search_Position_Stored_Invariants fixes progress representation.
func Identifier_Search_Position_Stored_Invariants(
	value Identifier_Search_Position_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value), uint16(ACTION_CONTENT_BOUNDARY_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Ensure()
}

// Identifier_Search_End_Storage owns one transient lexer bound.
type Identifier_Search_End_Storage struct {
	// Value prevents a stale scalar proof from escaping lexer storage.
	Value Identifier_Search_End_Stored
}

// Identifier_Search_End_Storage_Invariants fixes termination ownership.
func Identifier_Search_End_Storage_Invariants(
	value Identifier_Search_End_Storage, namespace aver.Namespace,
) {
	Identifier_Search_End_Stored_Invariants(value.Value, namespace)
}

// IDENTIFIER_SEARCH_END_MINIMUM stops after the first identifier byte.
const IDENTIFIER_SEARCH_END_MINIMUM = ACTION_CONTENT_BOUNDARY_MINIMUM +
	utf8.CHARACTER_SIZE_MINIMUM

// Identifier_Search_End_Stored separates search shape from its transient bound.
type Identifier_Search_End_Stored Identifier_Search_End

// Identifier_Search_End_Stored_Invariants fixes termination representation.
func Identifier_Search_End_Stored_Invariants(
	value Identifier_Search_End_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(IDENTIFIER_SEARCH_END_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM),
		).
		Ensure()
}

// Arguments_Proof_Stored separates callback proof shape from transient arguments.
type Arguments_Proof_Stored Arguments_Staged_Value

// Arguments_Proof_Stored_Invariants accepts absent output or typed arguments.
func Arguments_Proof_Stored_Invariants(value Arguments_Proof_Stored, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), ARGUMENT_COUNT_MINIMUM, ARGUMENT_COUNT_MAXIMUM).
		Ensure()
}

// Functions_Proof_Stored separates registry proof shape from transient functions.
type Functions_Proof_Stored Functions_Validated_Value

// Functions_Proof_Stored_Invariants accepts absent output or one typed registry.
func Functions_Proof_Stored_Invariants(value Functions_Proof_Stored, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), bytes.SLICE_SIZE_MINIMUM, FUNCTION_COUNT_MAXIMUM).
		Ensure()
}

// Node_Link_Proof_Stored separates link proof shape from transient references.
type Node_Link_Proof_Stored Node_Link_Validated_Value

// Node_Link_Proof_Stored_Invariants accepts absent output or one typed reference.
func Node_Link_Proof_Stored_Invariants(value Node_Link_Proof_Stored, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value), uint16(NODE_REFERENCE_MINIMUM), uint16(
			NODE_REFERENCE_MAXIMUM)).
		Ensure()
}

// Output_Count_Proof_Stored separates writer proof shape from transient progress.
type Output_Count_Proof_Stored Output_Count_Staged_Value

// Output_Count_Proof_Stored_Invariants accepts absent output or one typed count.
func Output_Count_Proof_Stored_Invariants(
	value Output_Count_Proof_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(OUTPUT_SIZE_MINIMUM), uint16(OUTPUT_SIZE_MAXIMUM),
		).
		Ensure()
}

// Source_Cursor_Stored separates source proof shape from transient positions.
type Source_Cursor_Stored Source_Cursor_Validated_Value

// Source_Cursor_Stored_Invariants accepts absent output or one typed position.
func Source_Cursor_Stored_Invariants(value Source_Cursor_Stored, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(SOURCE_SIZE_MINIMUM), uint16(SOURCE_SIZE_MAXIMUM),
		).
		Ensure()
}

// Number_Position_Stored separates numeric proof shape from transient positions.
type Number_Position_Stored Number_Position_Validated_Value

// Number_Position_Stored_Invariants accepts absent output or one typed position.
func Number_Position_Stored_Invariants(value Number_Position_Stored, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value), uint16(ACTION_CONTENT_BOUNDARY_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Ensure()
}

// QUOTED_ESCAPE_POSITION_MAXIMUM leaves the closing quote the literal still takes.
const QUOTED_ESCAPE_POSITION_MAXIMUM = ACTION_CONTENT_BOUNDARY_MAXIMUM -
	utf8.CHARACTER_SIZE_MINIMUM

// Quoted_Position_Stored separates quoted proof shape from transient positions.
type Quoted_Position_Stored Quoted_Position_Validated_Value

// Quoted_Position_Stored_Invariants accepts absent output or one typed position.
func Quoted_Position_Stored_Invariants(value Quoted_Position_Stored, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value), uint16(CONSTANT_END_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Ensure()
}

// Syntax_Variable_Count_Stored separates stack shape from transient population.
type Syntax_Variable_Count_Stored Syntax_Variable_Count_Storage_Value

// Syntax_Variable_Count_Stored_Invariants accepts absent zero or one typed count.
func Syntax_Variable_Count_Stored_Invariants(
	value Syntax_Variable_Count_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value), uint16(VARIABLE_COUNT_MINIMUM), uint16(
			SYNTAX_VARIABLE_COUNT_MAXIMUM)).
		Ensure()
}

// Control_Depth_Stored separates branch proof shape from transient depth.
type Control_Depth_Stored Control_Depth_Tracked_Value

// Control_Depth_Stored_Invariants accepts absent output or one typed depth.
func Control_Depth_Stored_Invariants(value Control_Depth_Stored, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(CONTROL_DEPTH_MINIMUM), uint16(CONTROL_DEPTH_MAXIMUM),
		).
		Ensure()
}

// Declaration_References_Stored separates binding shape from transient references.
type Declaration_References_Stored Declaration_References

// Declaration_References_Stored_Invariants fixes declaration representation.
func Declaration_References_Stored_Invariants(
	value Declaration_References_Stored, namespace aver.Namespace,
) {
	Declaration_First_Reference_Storage_Invariants(value.First, namespace)
	Declaration_Second_Reference_Storage_Invariants(value.Second, namespace)
}

// Frame_Storage_Fields carries raw mutable traversal storage.
type Frame_Storage_Fields struct {
	// Value keeps mutation inside traversal storage.
	Value Frame_Stored
}

// Frame_Storage_Fields_Invariants bounds hostile traversal storage.
func Frame_Storage_Fields_Invariants(value Frame_Storage_Fields, namespace aver.Namespace) {
	Frame_Stored_Invariants(value.Value, namespace)
}

// Frame_State_Stored separates traversal shape from transient state.
type Frame_State_Stored Frame_Stored

// Frame_State_Stored_Invariants fixes traversal-frame representation.
func Frame_State_Stored_Invariants(value Frame_State_Stored, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value.Kind), uint8(FRAME_LIST), uint8(FRAME_RANGE)).
		Range_Uint16(uint16(value.Reference), uint16(NODE_REFERENCE_MINIMUM), uint16(
			NODE_REFERENCE_MAXIMUM)).
		Range_Uint16(uint16(value.Index), uint16(bytes.SLICE_SIZE_MINIMUM), uint16(
			VALUE_COUNT_MAXIMUM)).
		Range_Uint16(uint16(value.Map_Entry_Index), uint16(bytes.SLICE_SIZE_MINIMUM),
			uint16(RANGE_MAP_ENTRY_INDEX_MAXIMUM)).
		Enum_4_Uint8(uint8(value.Map_Selected), uint8(MAP_SELECTION_NONE),
			uint8(MAP_SELECTION_FORWARD), uint8(MAP_SELECTION_BACKWARD),
			uint8(MAP_SELECTION_SCAN)).
		Range_Uint16(uint16(value.Variable_Mark), uint16(bytes.SLICE_SIZE_MINIMUM),
			uint16(VARIABLE_COUNT_MAXIMUM)).
		Range_Uint16(uint16(value.Iteration_Variables), uint16(bytes.SLICE_SIZE_MINIMUM),
			uint16(VARIABLE_COUNT_MAXIMUM)).
		Range_Uint16(
			uint16(value.Scope_Base), uint16(bytes.SLICE_SIZE_MINIMUM),
			uint16(VARIABLE_COUNT_MAXIMUM-VARIABLE_COUNT_INCREMENT)).
		Range_Uint16(
			uint16(value.Previous_Scope_Base), uint16(bytes.SLICE_SIZE_MINIMUM),
			uint16(VARIABLE_COUNT_MAXIMUM-VARIABLE_COUNT_INCREMENT)).
		Enum_3_Uint8(uint8(value.Declaration_Count), uint8(DECLARATION_COUNT_NONE), uint8(
			DECLARATION_COUNT_ONE), uint8(DECLARATION_COUNT_TWO)).
		Sometimes(bool(value.Assignment), "Pipeline updates existing variable.").
		Sometimes(bool(value.Restore), "Template Boolean is true.").
		Ensure()
	Value_Invariants(value.Dot, namespace)
	Ranged_Declaration_References_Invariants(
		Ranged_Declaration_References(value.Declarations), namespace)
}

// Value_Format_Frame_Storage_Fields carries raw mutable formatter storage.
type Value_Format_Frame_Storage_Fields struct {
	// Value keeps mutation inside formatter storage.
	Value Value_Format_Frame_Stored
}

// Value_Format_Frame_Storage_Fields_Invariants bounds hostile formatter storage.
func Value_Format_Frame_Storage_Fields_Invariants(
	value Value_Format_Frame_Storage_Fields, namespace aver.Namespace,
) {
	Value_Format_Frame_Stored_Invariants(value.Value, namespace)
}

// COMPOSITE_FRAME_OPENED marks one frame past its opening delimiter.
const COMPOSITE_FRAME_OPENED = true

// Value_Format_Frame_State_Stored separates formatter shape from transient state.
type Value_Format_Frame_State_Stored Value_Format_Frame_Stored

// Value_Format_Frame_State_Stored_Invariants fixes formatter-frame representation.
func Value_Format_Frame_State_Stored_Invariants(
	value Value_Format_Frame_State_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Index), uint16(bytes.SLICE_SIZE_MINIMUM),
			uint16(VALUE_COUNT_MAXIMUM-VALUE_COUNT_INCREMENT)).
		Range_Uint16(uint16(value.Map_Entry_Index), uint16(bytes.SLICE_SIZE_MINIMUM),
			uint16(RANGE_MAP_ENTRY_INDEX_MAXIMUM)).
		Enum_4_Uint8(uint8(value.Map_Selected), uint8(MAP_SELECTION_NONE),
			uint8(MAP_SELECTION_FORWARD), uint8(MAP_SELECTION_BACKWARD),
			uint8(MAP_SELECTION_SCAN)).
		Ensure()
	aver.Always(bool(value.Opened) == COMPOSITE_FRAME_OPENED,
		"One child frame follows its opening delimiter.")
	Composite_Value_Invariants(value.Value, namespace)
}

// Control_Frame_Storage_Fields carries raw mutable branch storage.
type Control_Frame_Storage_Fields struct {
	// Value keeps mutation inside the frame stack.
	Value Control_Frame_Stored
}

// Control_Frame_Storage_Fields_Invariants bounds hostile branch storage.
func Control_Frame_Storage_Fields_Invariants(
	value Control_Frame_Storage_Fields, namespace aver.Namespace,
) {
	Control_Frame_Stored_Invariants(value.Value, namespace)
}

// Control_Frame_State_Stored separates branch shape from transient state.
type Control_Frame_State_Stored Control_Frame_Stored

// Control_Frame_State_Stored_Invariants fixes branch-frame representation.
func Control_Frame_State_Stored_Invariants(
	value Control_Frame_State_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Branch), uint16(ACTION_NODE_COUNT_MINIMUM), uint16(
			NODE_REFERENCE_MAXIMUM)).
		Range_Holed_Uint16(
			uint16(value.Parent_List), uint16(ROOT_NODE_COUNT),
			uint16(NODE_REFERENCE_MAXIMUM), uint16(TERM_ACTION_HOLE),
			uint16(TERM_ACTION_HOLE), uint16(TERM_ACTION_HOLE)).
		Range_Uint16(uint16(value.Parent_Last_Child), uint16(ACTION_NODE_COUNT_MINIMUM),
			uint16(NODE_REFERENCE_MAXIMUM)).
		Range_Uint16(uint16(value.Parent_Variable_Count), uint16(VARIABLE_COUNT_MINIMUM),
			uint16(SYNTAX_VARIABLE_COUNT_MAXIMUM)).
		Range_Uint16(uint16(value.Branch_Variable_Count), uint16(VARIABLE_COUNT_MINIMUM),
			uint16(SYNTAX_VARIABLE_COUNT_MAXIMUM)).
		Range_Uint8(uint8(value.Kind), uint8(NODE_IF), uint8(NODE_WITH)).
		Sometimes(bool(value.Has_Else), "Control branch has an alternate arm.").
		Sometimes(bool(value.Shares_End), "Control branch shares its parent's end action.").
		Ensure()
}

// Evaluation_Frame_Fields carries raw mutable evaluator storage.
type Evaluation_Frame_Fields struct {
	// Nodes retain caller parser records while parent evaluation waits.
	Nodes Evaluation_Nodes
	// Values retain dot and command head while child evaluation runs.
	Values Evaluation_Values
	// References retain next sibling work.
	References Evaluation_References
	// States retain pipeline value and declarations.
	States Evaluation_States
	// Argument_Bases isolate nested callback argument regions.
	Argument_Bases Evaluation_Argument_Bases
	// Control retains iterative evaluator state.
	Control Evaluation_Control
}

// Evaluation_Frame_Fields_Invariants bounds hostile evaluator storage.
func Evaluation_Frame_Fields_Invariants(value Evaluation_Frame_Fields, namespace aver.Namespace,
) {
	Evaluation_Nodes_Invariants(value.Nodes, namespace)
	Evaluation_Values_Invariants(value.Values, namespace)
	Evaluation_References_Invariants(value.References, namespace)
	Evaluation_States_Invariants(value.States, namespace)
	Evaluation_Argument_Bases_Invariants(value.Argument_Bases, namespace)
	Evaluation_Control_Invariants(value.Control, namespace)
}

// Evaluation_References_Stored separates evaluator shape from transient references.
type Evaluation_References_Stored Evaluation_References

// Evaluation_References_Stored_Invariants fixes evaluator-reference representation.
func Evaluation_References_Stored_Invariants(
	value Evaluation_References_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Holed_Uint16(
			uint16(value.Command), uint16(NODE_REFERENCE_MINIMUM),
			uint16(COMMAND_REFERENCE_MAXIMUM), uint16(ROOT_LIST_HOLE),
			uint16(TERM_ACTION_HOLE), uint16(TERM_ACTION_HOLE)).
		Range_Holed_Uint16(
			uint16(value.Term), uint16(NODE_REFERENCE_MINIMUM),
			uint16(ACTION_TAIL_REFERENCE_MAXIMUM),
			uint16(ROOT_LIST_HOLE), uint16(TERM_ACTION_HOLE),
			uint16(TERM_ACTION_HOLE)).
		Ensure()
}

// Evaluation_Argument_Bases_Stored separates evaluator shape from transient stack state.
type Evaluation_Argument_Bases_Stored Evaluation_Argument_Bases

// Evaluation_Argument_Bases_Stored_Invariants fixes argument-base representation.
func Evaluation_Argument_Bases_Stored_Invariants(
	value Evaluation_Argument_Bases_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(ARGUMENT_COUNT_MINIMUM), uint16(
			ARGUMENT_BASE_MAXIMUM)).
		Ensure()
}

// Evaluation_Control_Stored separates evaluator shape from transient control state.
type Evaluation_Control_Stored Evaluation_Control

// Evaluation_Control_Stored_Invariants fixes evaluator-control representation.
func Evaluation_Control_Stored_Invariants(
	value Evaluation_Control_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value.Stage), uint8(EVALUATION_STAGE_COMMAND), uint8(
			EVALUATION_STAGE_DONE)).
		Enum_Uint8(uint8(value.Mode), uint8(EVALUATION_MODE_VALUE), uint8(
			EVALUATION_MODE_IDENTIFIER)).
		Enum_Uint8(uint8(value.Present), uint8(EVALUATION_CONTROL_FALSE), uint8(
			EVALUATION_CONTROL_TRUE)).
		Enum_Uint8(uint8(value.Chain), uint8(EVALUATION_CONTROL_FALSE), uint8(
			EVALUATION_CONTROL_TRUE)).
		Ensure()
}

// Configuration_Fields carries raw parser policy storage.
type Configuration_Fields struct {
	// Left retains the validated custom marker or empty default marker.
	Left Opening_Delimiter_Storage
	// Right retains the validated custom marker or empty default marker.
	Right Right_Delimiter_Storage
	// Mode retains validated parse flags.
	Mode Mode_Storage
	// Function retains caller name policy.
	Function Parse_Function_Storage
}

// Configuration_Fields_Invariants bounds hostile parser policy storage.
func Configuration_Fields_Invariants(value Configuration_Fields, namespace aver.Namespace) {
	Opening_Delimiter_Storage_Invariants(value.Left, namespace)
	Right_Delimiter_Storage_Invariants(value.Right, namespace)
	Mode_Storage_Invariants(value.Mode, namespace)
	Parse_Function_Storage_Invariants(value.Function, namespace)
}

// Configuration_Left_Stored separates policy shape from delimiter contents.
type Configuration_Left_Stored Opening_Delimiter_Storage

// Configuration_Left_Stored_Invariants fixes opening-delimiter representation.
func Configuration_Left_Stored_Invariants(
	value Configuration_Left_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value.Value), DELIMITER_SIZE_MINIMUM, DELIMITER_SIZE_MAXIMUM).
		Ensure()
}

// Configuration_Right_Stored separates policy shape from delimiter contents.
type Configuration_Right_Stored Right_Delimiter_Storage

// Configuration_Right_Stored_Invariants fixes closing-delimiter representation.
func Configuration_Right_Stored_Invariants(
	value Configuration_Right_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value.Value), DELIMITER_SIZE_MINIMUM, DELIMITER_SIZE_MAXIMUM).
		Ensure()
}

// Configuration_Mode_Stored separates policy shape from transient flags.
type Configuration_Mode_Stored Mode_Storage

// Configuration_Mode_Stored_Invariants fixes mode representation.
func Configuration_Mode_Stored_Invariants(
	value Configuration_Mode_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_4_Uint8(uint8(value.Value), uint8(MODE_MINIMUM), uint8(PARSE_COMMENTS),
			uint8(SKIP_FUNCTION_CHECK), uint8(MODE_MAXIMUM)).
		Ensure()
}

// Program_Validated_Fields carries raw validated-program proof storage.
type Program_Validated_Fields struct {
	// Source retains checked borrowed input.
	Source Source
	// Document retains checked arena metadata.
	Document Program_Validated_Document
	// Syntax retains the checked caller node arena.
	Syntax Syntax_Storage
}

// Program_Validated_Fields_Invariants bounds hostile program proof storage.
func Program_Validated_Fields_Invariants(value Program_Validated_Fields, namespace aver.Namespace,
) {
	Source_Invariants(value.Source, namespace)
	Program_Validated_Document_Invariants(value.Document, namespace)
	Syntax_Storage_Invariants(value.Syntax, namespace)
}

// Program_Validated_Document stores metadata already checked by program_valid.
type Program_Validated_Document Document

// Program_Validated_Document_Invariants fixes validated metadata representation.
func Program_Validated_Document_Invariants(
	value Program_Validated_Document, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Node_Count), uint16(ROOT_NODE_COUNT), uint16(
			NODE_COUNT_MAXIMUM)).
		Range_Uint16(uint16(value.First_Template), uint16(
			DOCUMENT_FIRST_TEMPLATE_MINIMUM), uint16(DOCUMENT_FIRST_TEMPLATE_MAXIMUM)).
		Range_Uint16(uint16(value.Template_Count), uint16(TEMPLATE_COUNT_MINIMUM), uint16(
			EXECUTED_TEMPLATE_COUNT_MAXIMUM)).
		Ensure()
	aver.Always(uint16(value.Root) == uint16(ROOT_NODE_COUNT),
		"Stored program metadata keeps the root in the first syntax arena slot.")
}

// Token_Validated_Fields carries raw lexer proof storage.
type Token_Validated_Fields struct {
	// Value cannot share identity with unvalidated input.
	Value Token_Validated_Value
}

// Token_Validated_Fields_Invariants bounds hostile lexer proof storage.
func Token_Validated_Fields_Invariants(value Token_Validated_Fields, namespace aver.Namespace) {
	Token_Validated_Value_Invariants(value.Value, namespace)
}

// Token_Proof_Stored separates lexer proof shape from its transient span.
type Token_Proof_Stored Token_Validated_Value

// Token_Proof_Stored_Invariants fixes lexer proof representation.
func Token_Proof_Stored_Invariants(value Token_Proof_Stored, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(
			uint8(value.Kind), uint8(TOKEN_KIND_MINIMUM), uint8(TOKEN_KIND_MAXIMUM),
		).
		Range_Uint16(uint16(value.Start), uint16(SOURCE_SIZE_MINIMUM), uint16(
			ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Range_Uint16(uint16(value.End), uint16(SOURCE_SIZE_MINIMUM), uint16(
			ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Ensure()
}

// Pipeline_Frame_Storage_Fields carries raw mutable frame storage.
type Pipeline_Frame_Storage_Fields struct {
	// Value keeps mutation inside the frame stack.
	Value Pipeline_Frame_Stored
}

// Pipeline_Frame_Storage_Fields_Invariants bounds hostile frame storage.
func Pipeline_Frame_Storage_Fields_Invariants(
	value Pipeline_Frame_Storage_Fields, namespace aver.Namespace,
) {
	Pipeline_Frame_Stored_Invariants(value.Value, namespace)
}

// Pipeline_Frame_State_Stored separates frame shape from transient references.
type Pipeline_Frame_State_Stored Pipeline_Frame_Stored

// Pipeline_Frame_State_Stored_Invariants fixes pipeline-frame representation.
func Pipeline_Frame_State_Stored_Invariants(
	value Pipeline_Frame_State_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Pipe), uint16(PIPELINE_NODE_COUNT_MINIMUM),
			uint16(PIPELINE_REFERENCE_MAXIMUM)).
		Range_Holed_Uint16(uint16(value.Last_Child), uint16(NODE_REFERENCE_MINIMUM),
			uint16(COMMAND_REFERENCE_MAXIMUM), uint16(ROOT_LIST_HOLE),
			uint16(TERM_ACTION_HOLE), uint16(TERM_PIPELINE_HOLE)).
		Range_Holed_Uint16(uint16(value.Command), uint16(NODE_REFERENCE_MINIMUM),
			uint16(COMMAND_REFERENCE_MAXIMUM), uint16(ROOT_LIST_HOLE),
			uint16(TERM_ACTION_HOLE), uint16(TERM_PIPELINE_HOLE)).
		Range_Holed_Uint16(uint16(value.Last_Argument), uint16(NODE_REFERENCE_MINIMUM),
			uint16(ARGUMENT_REFERENCE_MAXIMUM), uint16(ROOT_LIST_HOLE),
			uint16(TERM_ACTION_HOLE), uint16(TERM_PIPELINE_HOLE)).
		Ensure()
}

// List_State_Storage_Fields carries raw mutable list storage.
type List_State_Storage_Fields struct {
	// Value keeps mutation inside parser state.
	Value List_State_Stored
}

// List_State_Storage_Fields_Invariants bounds hostile list storage.
func List_State_Storage_Fields_Invariants(
	value List_State_Storage_Fields, namespace aver.Namespace,
) {
	List_State_Stored_Invariants(value.Value, namespace)
}

// LIST_REFERENCE_MAXIMUM packs actions beside the shortest branch that opens a list.
const LIST_REFERENCE_MAXIMUM = ROOT_NODE_COUNT + CONTROL_BRANCH_NODE_COUNT +
	(SOURCE_SIZE_MAXIMUM-CONTROL_ACTION_SIZE_MINIMUM)/SIMPLE_ACTION_SIZE_MINIMUM*
		SIMPLE_ACTION_NODE_COUNT + TRAILING_TEXT_NODE_COUNT_MAXIMUM

// LIST_LAST_CHILD_MAXIMUM leaves the terms and the trailing text one action still owes.
const LIST_LAST_CHILD_MAXIMUM = NODE_COUNT_MAXIMUM - PIPELINE_NODE_COUNT -
	COMMAND_NODE_COUNT - SCALAR_NODE_COUNT - TRAILING_TEXT_NODE_COUNT_MAXIMUM

// List_State_Value_Stored separates list shape from transient references.
type List_State_Value_Stored List_State_Stored

// List_State_Value_Stored_Invariants fixes list-state representation.
func List_State_Value_Stored_Invariants(value List_State_Value_Stored, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Holed_Uint16(
			uint16(value.List), uint16(ROOT_NODE_COUNT), uint16(TEXT_BODY_LIST_MAXIMUM),
			uint16(ACTION_LIST_HOLE), uint16(ACTION_LIST_HOLE),
			uint16(ACTION_LIST_HOLE)).
		Range_Holed_Uint16(uint16(value.Last_Child), uint16(NODE_REFERENCE_MINIMUM),
			uint16(LIST_LAST_CHILD_MAXIMUM), uint16(ROOT_LIST_HOLE),
			uint16(ROOT_LIST_HOLE), uint16(ROOT_LIST_HOLE)).
		Ensure()
}

// Number_Span_Validated_Fields carries raw numeric-span proof storage.
type Number_Span_Validated_Fields struct {
	// Value keeps token proof attached to the span.
	Value Number_Span_Stored
}

// Number_Span_Validated_Fields_Invariants bounds hostile numeric proof storage.
func Number_Span_Validated_Fields_Invariants(
	value Number_Span_Validated_Fields, namespace aver.Namespace,
) {
	Number_Span_Stored_Invariants(value.Value, namespace)
}

// Number_Span_Proof_Stored separates numeric proof shape from transient boundaries.
type Number_Span_Proof_Stored Number_Span_Stored

// Number_Span_Proof_Stored_Invariants fixes numeric-span representation.
func Number_Span_Proof_Stored_Invariants(value Number_Span_Proof_Stored, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Start), uint16(NUMBER_START_MINIMUM),
			uint16(NUMBER_START_MAXIMUM)).
		Range_Uint16(uint16(value.End), uint16(NUMBER_END_MINIMUM),
			uint16(NUMBER_END_MAXIMUM)).
		Ensure()
}

// Quoted_Span_Validated_Fields carries raw quoted-span proof storage.
type Quoted_Span_Validated_Fields struct {
	// Value keeps source proof attached to the span.
	Value Quoted_Span_Stored
}

// Quoted_Span_Validated_Fields_Invariants bounds hostile quoted proof storage.
func Quoted_Span_Validated_Fields_Invariants(
	value Quoted_Span_Validated_Fields, namespace aver.Namespace,
) {
	Quoted_Span_Stored_Invariants(value.Value, namespace)
}

// Quoted_Span_Proof_Stored separates quoted proof shape from transient boundaries.
type Quoted_Span_Proof_Stored Quoted_Span_Stored

// Quoted_Span_Proof_Stored_Invariants fixes quoted-span representation.
func Quoted_Span_Proof_Stored_Invariants(value Quoted_Span_Proof_Stored, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Start), uint16(QUOTED_START_MINIMUM), uint16(
			QUOTED_START_MAXIMUM)).
		Range_Uint16(
			uint16(value.End), uint16(QUOTED_END_MINIMUM), uint16(QUOTED_END_MAXIMUM),
		).
		Ensure()
}

// Action_Span_Validated_Fields carries raw action-span proof storage.
type Action_Span_Validated_Fields struct {
	// Value keeps delimiter proof attached to the span.
	Value Action_Span_Stored
}

// Action_Span_Validated_Fields_Invariants bounds hostile action proof storage.
func Action_Span_Validated_Fields_Invariants(
	value Action_Span_Validated_Fields, namespace aver.Namespace,
) {
	Action_Span_Stored_Invariants(value.Value, namespace)
}

// Action_Span_Proof_Stored separates action proof shape from transient boundaries.
type Action_Span_Proof_Stored Action_Span_Stored

// Action_Span_Proof_Stored_Invariants fixes action-span representation.
func Action_Span_Proof_Stored_Invariants(value Action_Span_Proof_Stored, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Start), uint16(ACTION_START_MINIMUM), uint16(
			ACTION_START_MAXIMUM)).
		Range_Uint16(uint16(value.Content_Start), uint16(ACTION_CONTENT_BOUNDARY_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Range_Uint16(uint16(value.Content_End), uint16(ACTION_CONTENT_BOUNDARY_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Range_Uint16(
			uint16(value.End), uint16(ACTION_END_MINIMUM), uint16(ACTION_END_MAXIMUM),
		).
		Ensure()
}

// Lexeme_Validated_Fields carries raw source-span proof storage.
type Lexeme_Validated_Fields struct {
	// Value cannot share identity with unvalidated input.
	Value Lexeme_Stored
}

// Lexeme_Validated_Fields_Invariants bounds hostile lexeme proof storage.
func Lexeme_Validated_Fields_Invariants(value Lexeme_Validated_Fields, namespace aver.Namespace,
) {
	Lexeme_Stored_Invariants(value.Value, namespace)
}

// Lexeme_Proof_Stored separates lexeme proof shape from transient boundaries.
type Lexeme_Proof_Stored Lexeme_Stored

// Lexeme_Proof_Stored_Invariants fixes lexeme representation.
func Lexeme_Proof_Stored_Invariants(value Lexeme_Proof_Stored, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Start), uint16(LEXEME_START_MINIMUM), uint16(
			LEXEME_START_MAXIMUM)).
		Range_Uint8(uint8(value.Character), bits.WORD_8_MINIMUM, bits.WORD_8_MAXIMUM).
		Ensure()
}

// Identifier_Span_Validated_Fields carries raw identifier proof storage.
type Identifier_Span_Validated_Fields struct {
	// Value cannot share identity with unvalidated input.
	Value Identifier_Span_Stored
}

// Identifier_Span_Validated_Fields_Invariants bounds hostile identifier proof storage.
func Identifier_Span_Validated_Fields_Invariants(
	value Identifier_Span_Validated_Fields, namespace aver.Namespace,
) {
	Identifier_Span_Stored_Invariants(value.Value, namespace)
}

// Identifier_Span_Proof_Stored separates identifier proof shape from transient boundaries.
type Identifier_Span_Proof_Stored Identifier_Span_Stored

// Identifier_Span_Proof_Stored_Invariants fixes identifier-span representation.
func Identifier_Span_Proof_Stored_Invariants(
	value Identifier_Span_Proof_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Start), uint16(IDENTIFIER_START_MINIMUM), uint16(
			IDENTIFIER_START_MAXIMUM)).
		Range_Uint16(uint16(value.End), uint16(IDENTIFIER_END_MINIMUM), uint16(
			IDENTIFIER_END_MAXIMUM)).
		Ensure()
}

// Text_Span_Validated_Fields carries raw text proof storage.
type Text_Span_Validated_Fields struct {
	// Value keeps parser proof attached to the span.
	Value Text_Span_Stored
}

// Text_Span_Validated_Fields_Invariants bounds hostile text proof storage.
func Text_Span_Validated_Fields_Invariants(
	value Text_Span_Validated_Fields, namespace aver.Namespace,
) {
	Text_Span_Stored_Invariants(value.Value, namespace)
}

// TRIM_MARKER_SIZE is the byte one trimming action opens with.
const TRIM_MARKER_SIZE = utf8.CHARACTER_SIZE_MINIMUM

// TRIMMED_TEXT_BOUNDARY_MAXIMUM leaves the trimming action the text stands before.
const TRIMMED_TEXT_BOUNDARY_MAXIMUM = SOURCE_SIZE_MAXIMUM - ACTION_BOUNDARY_SIZE_MINIMUM -
	TRIM_MARKER_SIZE - ACTION_SEPARATOR_SIZE_MINIMUM - DOT_SOURCE_SIZE

// Text_Span_Proof_Stored separates text proof shape from transient boundaries.
type Text_Span_Proof_Stored Text_Span_Stored

// Text_Span_Proof_Stored_Invariants fixes text-span representation.
func Text_Span_Proof_Stored_Invariants(value Text_Span_Proof_Stored, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Holed_Uint16(
			uint16(value.Start), uint16(SOURCE_SIZE_MINIMUM),
			uint16(TRIMMED_TEXT_BOUNDARY_MAXIMUM), uint16(ROOT_LIST_HOLE),
			uint16(TERM_ACTION_HOLE), uint16(TERM_ACTION_HOLE)).
		Range_Uint16(
			uint16(value.End), uint16(SOURCE_SIZE_MINIMUM),
			uint16(TRIMMED_TEXT_BOUNDARY_MAXIMUM),
		).
		Ensure()
}

// Action_Scan_Span_Validated_Fields carries raw scan proof storage.
type Action_Scan_Span_Validated_Fields struct {
	// Value keeps source proof attached to the span.
	Value Action_Scan_Span_Stored
}

// Action_Scan_Span_Validated_Fields_Invariants bounds hostile scan proof storage.
func Action_Scan_Span_Validated_Fields_Invariants(
	value Action_Scan_Span_Validated_Fields, namespace aver.Namespace,
) {
	Action_Scan_Span_Stored_Invariants(value.Value, namespace)
}

// Action_Scan_Span_Proof_Stored separates scan proof shape from transient boundaries.
type Action_Scan_Span_Proof_Stored Action_Scan_Span_Stored

// Action_Scan_Span_Proof_Stored_Invariants fixes scan-span representation.
func Action_Scan_Span_Proof_Stored_Invariants(
	value Action_Scan_Span_Proof_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Start), uint16(ACTION_SCAN_START_MINIMUM), uint16(
			LEXEME_START_MAXIMUM)).
		Range_Uint16(uint16(value.Limit), uint16(ACTION_SCAN_LIMIT_MINIMUM), uint16(
			ACTION_SCAN_LIMIT_MAXIMUM)).
		Ensure()
}

// Template_Search_Validated_Fields carries raw lookup proof storage.
type Template_Search_Validated_Fields struct {
	// Value keeps traversal proof attached to the result.
	Value Template_Search_Stored
}

// Template_Search_Validated_Fields_Invariants bounds hostile lookup proof storage.
func Template_Search_Validated_Fields_Invariants(
	value Template_Search_Validated_Fields, namespace aver.Namespace,
) {
	Template_Search_Stored_Invariants(value.Value, namespace)
}

// Template_Search_Proof_Stored separates lookup proof shape from transient references.
type Template_Search_Proof_Stored Template_Search_Stored

// Template_Search_Proof_Stored_Invariants fixes lookup representation.
func Template_Search_Proof_Stored_Invariants(
	value Template_Search_Proof_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Holed_Uint16(
			uint16(value.Match), uint16(NODE_REFERENCE_MINIMUM),
			uint16(EARLIER_TEMPLATE_REFERENCE_MAXIMUM), uint16(ROOT_LIST_HOLE),
			uint16(ROOT_LIST_HOLE), uint16(ROOT_LIST_HOLE)).
		Range_Holed_Uint16(
			uint16(value.Previous), uint16(NODE_REFERENCE_MINIMUM),
			uint16(NAMED_PAIR_EARLIER_REFERENCE_MAXIMUM), uint16(ROOT_LIST_HOLE),
			uint16(ROOT_LIST_HOLE), uint16(ROOT_LIST_HOLE)).
		Ensure()
}

// Scope_Restore_Validated_Fields carries raw scope-restoration proof storage.
type Scope_Restore_Validated_Fields struct {
	// Value keeps workspace proof attached to restoration state.
	Value Scope_Restore_Stored
}

// Scope_Restore_Validated_Fields_Invariants bounds hostile restoration proof storage.
func Scope_Restore_Validated_Fields_Invariants(
	value Scope_Restore_Validated_Fields, namespace aver.Namespace,
) {
	Scope_Restore_Stored_Invariants(value.Value, namespace)
}

// Scope_Restore_Proof_Stored separates restoration proof shape from transient cursors.
type Scope_Restore_Proof_Stored Scope_Restore_Stored

// Scope_Restore_Proof_Stored_Invariants fixes restoration representation.
func Scope_Restore_Proof_Stored_Invariants(
	value Scope_Restore_Proof_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Mark), uint16(VARIABLE_COUNT_INCREMENT), uint16(
			VARIABLE_COUNT_MAXIMUM)).
		Range_Uint16(uint16(value.Previous), uint16(bytes.SLICE_SIZE_MINIMUM), uint16(
			VARIABLE_COUNT_MAXIMUM-VARIABLE_COUNT_INCREMENT)).
		Ensure()
}

// Template_State_Fields carries raw definition-chain storage.
type Template_State_Fields struct {
	// First begins definition sibling chain.
	First Template_First_Storage
	// Last links next definition.
	Last Template_Last_Storage
	// Count bounds definition chain length.
	Count Template_Count_Storage
}

// Template_State_Fields_Invariants bounds hostile definition-chain storage.
func Template_State_Fields_Invariants(value Template_State_Fields, namespace aver.Namespace) {
	Template_First_Storage_Invariants(value.First, namespace)
	Template_Last_Storage_Invariants(value.Last, namespace)
	Template_Count_Storage_Invariants(value.Count, namespace)
}

// Template_First_Stored separates chain shape from transient references.
type Template_First_Stored Template_First_Storage

// Template_First_Stored_Invariants fixes first-reference representation.
func Template_First_Stored_Invariants(value Template_First_Stored, namespace aver.Namespace) {
	// A definition stands inside the root list, never in the slot the list holds.
	aver.Tree(value, namespace).
		Range_Holed_Uint16(uint16(value.Value), uint16(NODE_REFERENCE_MINIMUM),
			uint16(DOCUMENT_FIRST_TEMPLATE_REFERENCE_MAXIMUM), uint16(ROOT_LIST_HOLE),
			uint16(ROOT_LIST_HOLE), uint16(ROOT_LIST_HOLE)).
		Ensure()
}

// Template_Last_Stored separates chain shape from transient references.
type Template_Last_Stored Template_Last_Storage

// Template_Last_Stored_Invariants fixes last-reference representation.
func Template_Last_Stored_Invariants(value Template_Last_Stored, namespace aver.Namespace) {
	// A definition stands inside the root list, never in the slot the list holds.
	aver.Tree(value, namespace).
		Range_Holed_Uint16(uint16(value.Value), uint16(NODE_REFERENCE_MINIMUM),
			uint16(DOCUMENT_FIRST_TEMPLATE_REFERENCE_MAXIMUM), uint16(ROOT_LIST_HOLE),
			uint16(ROOT_LIST_HOLE), uint16(ROOT_LIST_HOLE)).
		Ensure()
}

// Template_Count_Stored separates chain shape from transient population.
type Template_Count_Stored Template_Count_Storage

// Template_Count_Stored_Invariants fixes definition-count representation.
func Template_Count_Stored_Invariants(value Template_Count_Stored, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(TEMPLATE_COUNT_MINIMUM), uint16(
			TEMPLATE_COUNT_MAXIMUM)).
		Ensure()
}

func delimiter_end(
	position_state Delimiter_Start_Cursor, configuration Configuration, left Left_Delimiter,
) (end_state Opened_Source_Cursor) {
	defer func() {
		Opened_Source_Cursor_Invariants(end_state, "delimiter_end.end_state")
	}()
	Delimiter_Start_Cursor_Invariants(position_state, "delimiter_end.position_state")
	Configuration_Invariants(configuration, "delimiter_end.configuration")
	Left_Delimiter_Invariants(left, "delimiter_end.left")
	position := Source_Position(Source_Cursor_Validated_Value(position_state.Value))
	delimiter := []byte(configuration.Right.Value)
	if bool(left) {
		delimiter = []byte(configuration.Left.Value)
	}
	if len(delimiter) == DELIMITER_SIZE_MINIMUM {
		return Opened_Source_Cursor{
			Value: Source_Cursor_Stored(
				position + Source_Position(DEFAULT_DELIMITER_SIZE)),
		}

	}
	return Opened_Source_Cursor{
		Value: Source_Cursor_Stored(position + Source_Position(len(delimiter))),
	}
}

func left_trimmed_start(
	source Text_Source, left_state Left_Trimmed_Start_Validated, configuration Configuration,
) (_ Trim_Whitespace, start_state Opened_Source_Cursor) {
	defer func() {
		Opened_Source_Cursor_Invariants(start_state, "left_trimmed_start.start_state")
	}()
	Left_Trimmed_Start_Validated_Invariants(left_state, "left_trimmed_start.left_state")
	Text_Source_Invariants(source, "left_trimmed_start.source")
	Configuration_Invariants(configuration, "left_trimmed_start.configuration")
	var trim Trim_Whitespace
	defer func() { Trim_Whitespace_Invariants(trim, "left_trimmed_start.trim") }()
	data := source.Data.Value
	start_state = delimiter_end(Delimiter_Start_Cursor(left_state), configuration, true)
	start := Source_Position(Source_Cursor_Validated_Value(start_state.Value))
	if int(start)+SOURCE_POSITION_INCREMENT >= len(data) {
		return false, start_state
	}
	if data[start] == TRIM_MARKER {
		if bool(space(Input_Byte(data[start+SOURCE_POSITION_INCREMENT]))) {
			trim = true
			start_state = Opened_Source_Cursor{
				Value: Source_Cursor_Stored(
					start + Source_Position(NUMBER_PREFIX_SIZE)),
			}
			return trim, start_state
		}
	}
	return false, start_state
}

func right_trimmed_end(source Action_Source, start_state Right_Trimmed_End_Source_Cursor_Validated,
	right_state Right_Trimmed_End_Cursor_Validated,
) (_ Source_Cursor_Validated, trim Trim_Whitespace) {
	defer func() { Trim_Whitespace_Invariants(trim, "right_trimmed_end.trim") }()
	Right_Trimmed_End_Source_Cursor_Validated_Invariants(start_state,
		"right_trimmed_end.start_state")
	Right_Trimmed_End_Cursor_Validated_Invariants(right_state, "right_trimmed_end.right_state")
	Action_Source_Invariants(source, "right_trimmed_end.source")
	end_state := Source_Cursor_Validated(right_state)
	defer func() {
		Trimmed_End_Cursor_Invariants(
			Trimmed_End_Cursor(end_state), "right_trimmed_end.end_state")
	}()
	start := Source_Position(Source_Cursor_Validated_Value(Source_Cursor_Validated(
		start_state).Value))
	right := Source_Position(Source_Cursor_Validated_Value(Source_Cursor_Validated(
		right_state).Value))
	data := source.Data.Value
	if right-start < Source_Position(NUMBER_PREFIX_SIZE) {
		return Source_Cursor_Validated(right_state), false
	}
	if data[right-SOURCE_POSITION_INCREMENT] == TRIM_MARKER {
		if bool(space(Input_Byte(data[right-Source_Position(NUMBER_PREFIX_SIZE)]))) {
			end_state = Source_Cursor_Validated{
				Value: Source_Cursor_Stored(
					right - Source_Position(NUMBER_PREFIX_SIZE)),
			}
			trim = true
			return end_state, trim
		}
	}
	return Source_Cursor_Validated(right_state), false
}

func skip_space(source Word_Source, start_state Skip_Space_Cursor_Validated,
) (end_state Skipped_Space_Cursor) {
	defer func() {
		Skipped_Space_Cursor_Invariants(end_state, "skip_space.end_state")
	}()
	Skip_Space_Cursor_Validated_Invariants(start_state, "skip_space.start_state")
	Word_Source_Invariants(source, "skip_space.source")
	data := source.Data.Value
	end := Source_Cursor_Validated_Value(Source_Cursor_Validated(start_state).Value)
	for int(end) < len(data) {
		if !bool(space(Input_Byte(data[end]))) {
			break
		}
		end++
	}
	return Skipped_Space_Cursor{Value: Source_Cursor_Stored(end)}
}

func trim_space_end(source Word_Source, span_state Text_Span_Validated,
) (trimmed Text_Span_Validated) {
	defer func() {
		Text_Span_Validated_Invariants(trimmed, "trim_space_end.trimmed")
	}()
	Text_Span_Validated_Invariants(span_state, "trim_space_end.span_state")
	Word_Source_Invariants(source, "trim_space_end.source")
	data := source.Data.Value
	trimmed = span_state
	span := trimmed.Value
	for uint16(span.End) > uint16(span.Start) {
		if !bool(space(Input_Byte(data[span.End-SOURCE_POSITION_INCREMENT]))) {
			break
		}
		span.End--
	}
	trimmed.Value = span
	return trimmed
}

func output_text(output Output_Storage, count_value Output_Count_Staged, source Output_Text,
) (_ Output_Count_Staged, status Output_Status) {
	defer func() { Output_Status_Invariants(status, "output_text.status") }()
	Output_Storage_Invariants(output, "output_text.output")

	Output_Count_Staged_Invariants(count_value, "output_text.count_value")
	Output_Text_Invariants(source, "output_text.source")
	written := count_value
	defer func() {
		Written_Output_Count_Invariants(
			Written_Output_Count(written), "output_text.written")
	}()

	count := int(Output_Count_Staged_Value(count_value.Value))
	if count > len(output)-len(source) {
		return count_value, OUTPUT_STATUS_TOO_LARGE
	}
	copy(output[count:], source)
	written = Output_Count_Staged{
		Value: Output_Count_Proof_Stored(Output_Count_Staged_Value(count_value.Value) +
			Output_Count_Staged_Value(len(source))),
	}
	return written, OUTPUT_STATUS_OK
}

func bytes_equal(left Text, right Text) (equal Bytes_Equality,
) {
	defer func() { Bytes_Equality_Invariants(equal, "bytes_equal.equal") }()
	Text_Invariants(left, "bytes_equal.left")
	Text_Invariants(right, "bytes_equal.right")
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func space(character Input_Byte) (whitespace Whitespace) {
	defer func() { Whitespace_Invariants(whitespace, "space.whitespace") }()
	Input_Byte_Invariants(character, "space.character")
	switch character {
	case ' ', '\t', '\r', '\n':
		return true
	default:
		return false
	}
}

// Evaluation_Identifier_Start begins the retained callable name.
type Evaluation_Identifier_Start uint16

// Evaluation_Identifier_Start_Invariants admits reset storage and every identifier start.
func Evaluation_Identifier_Start_Invariants(
	value Evaluation_Identifier_Start, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value), uint16(SOURCE_SIZE_MINIMUM),
			uint16(IDENTIFIER_START_MAXIMUM)).
		Ensure()
}

// Evaluation_Identifier_End closes the retained callable name.
type Evaluation_Identifier_End uint16

// Evaluation_Identifier_End_Invariants admits reset storage and every identifier end.
func Evaluation_Identifier_End_Invariants(
	value Evaluation_Identifier_End, namespace aver.Namespace,
) {
	// One delimiter and one name byte stand before the end this span records.
	aver.Tree(value, namespace).
		Range_Holed_Uint16(
			uint16(value), uint16(SOURCE_SIZE_MINIMUM),
			uint16(IDENTIFIER_END_MAXIMUM),
			uint16(CUSTOM_DELIMITER_SIZE_MINIMUM),
			uint16(CUSTOM_DELIMITER_SIZE_MINIMUM),
			uint16(CUSTOM_DELIMITER_SIZE_MINIMUM),
		).
		Ensure()
}

// Evaluation_Identifier_Span locates the callable name while arguments run.
type Evaluation_Identifier_Span struct {
	// Start begins the name.
	Start Evaluation_Identifier_Start
	// End closes the name.
	End Evaluation_Identifier_End
}

// Evaluation_Identifier_Span_Invariants composes both name boundaries.
func Evaluation_Identifier_Span_Invariants(
	value Evaluation_Identifier_Span, namespace aver.Namespace,
) {
	Evaluation_Identifier_Start_Invariants(value.Start, namespace)
	Evaluation_Identifier_End_Invariants(value.End, namespace)
}

// CHAIN_START_HOLE is where no chain begins, because one term stands before it.
const CHAIN_START_HOLE = CUSTOM_DELIMITER_SIZE_MINIMUM + utf8.CHARACTER_SIZE_MINIMUM

// CHAIN_START_MAXIMUM reserves the shortest field name the chain retains.
const CHAIN_START_MAXIMUM = LEXEME_START_MAXIMUM - IDENTIFIER_SIZE_MINIMUM

// Evaluation_Chain_Start begins the retained field path.
type Evaluation_Chain_Start uint16

// Evaluation_Chain_Start_Invariants admits reset storage and every lexeme start.
func Evaluation_Chain_Start_Invariants(value Evaluation_Chain_Start, namespace aver.Namespace) {
	// One term stands between the opening delimiter and the chain that follows it.
	aver.Tree(value, namespace).
		Range_Holed_Uint16(
			uint16(value), uint16(SOURCE_SIZE_MINIMUM),
			uint16(CHAIN_START_MAXIMUM),
			uint16(CUSTOM_DELIMITER_SIZE_MINIMUM),
			uint16(CHAIN_START_HOLE),
			uint16(CHAIN_START_HOLE),
		).
		Ensure()
}

// Evaluation_Chain_End closes the retained field path.
type Evaluation_Chain_End uint16

// Evaluation_Chain_End_Invariants admits reset storage and every action content end.
func Evaluation_Chain_End_Invariants(value Evaluation_Chain_End, namespace aver.Namespace) {
	// One term and one field name stand between the delimiter and the chain end.
	aver.Tree(value, namespace).
		Range_Holed_Uint16(uint16(value), uint16(SOURCE_SIZE_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM),
			uint16(CUSTOM_DELIMITER_SIZE_MINIMUM),
			uint16(CUSTOM_DELIMITER_SIZE_MINIMUM+IDENTIFIER_SIZE_MINIMUM),
			uint16(CUSTOM_DELIMITER_SIZE_MINIMUM+IDENTIFIER_SIZE_MINIMUM+
				IDENTIFIER_SIZE_MINIMUM)).
		Ensure()
}

// Evaluation_Chain_Span locates the pending field path while a child runs.
type Evaluation_Chain_Span struct {
	// Start begins the path.
	Start Evaluation_Chain_Start
	// End closes the path.
	End Evaluation_Chain_End
}

// Evaluation_Chain_Span_Invariants composes both path boundaries.
func Evaluation_Chain_Span_Invariants(value Evaluation_Chain_Span, namespace aver.Namespace) {
	Evaluation_Chain_Start_Invariants(value.Start, namespace)
	Evaluation_Chain_End_Invariants(value.End, namespace)
}

// Evaluation_Nodes stores active identifier and chain source spans.
type Evaluation_Nodes struct {
	// Identifier retains callable name while arguments run.
	Identifier Evaluation_Identifier_Span
	// Chain retains pending field path while a child runs.
	Chain Evaluation_Chain_Span
}

// Evaluation_Nodes_Invariants fixes iterative syntax-frame shape.
func Evaluation_Nodes_Invariants(value Evaluation_Nodes, namespace aver.Namespace) {
	Evaluation_Identifier_Span_Invariants(value.Identifier, namespace)
	Evaluation_Chain_Span_Invariants(value.Chain, namespace)
}

// Evaluation_Dot_Value keeps cursor identity independent.
type Evaluation_Dot_Value Value

// Evaluation_Dot_Value_Invariants bounds every payload field inline for its own role.
func Evaluation_Dot_Value_Invariants(value Evaluation_Dot_Value, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value.Kind), uint8(VALUE_NIL), uint8(VALUE_FUNCTION)).
		Range_Uint64(uint64(value.Real), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Imaginary), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Int(len(value.Text), OUTPUT_SIZE_MINIMUM, OUTPUT_SIZE_MAXIMUM).
		Range_Int(len(value.Values), bytes.SLICE_SIZE_MINIMUM, VALUE_COUNT_MAXIMUM).
		Range_Int(len(value.Fields), bytes.SLICE_SIZE_MINIMUM, VALUE_COUNT_MAXIMUM).
		Sometimes(value.Function != nil, "Evaluation dot holds a callable.").
		Ensure()
}

// Evaluation_Dot_Kind_Storage prevents a mutable cursor from proving stale contents.
type Evaluation_Dot_Kind_Storage Value_Kind_Storage

// Evaluation_Dot_Kind_Storage_Invariants fixes the cursor selector representation.
func Evaluation_Dot_Kind_Storage_Invariants(
	value Evaluation_Dot_Kind_Storage, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value.Value), bits.WORD_8_MINIMUM, bits.WORD_8_MAXIMUM).
		Ensure()
}

// Evaluation_Dot_Scalar_Storage prevents a mutable cursor from proving stale contents.
type Evaluation_Dot_Scalar_Storage Value_Scalar

// Evaluation_Dot_Scalar_Storage_Invariants fixes the cursor scalar representation.
func Evaluation_Dot_Scalar_Storage_Invariants(
	value Evaluation_Dot_Scalar_Storage, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value.Real), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Imaginary), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Evaluation_Dot_Text_Storage prevents a mutable cursor from proving stale contents.
type Evaluation_Dot_Text_Storage Text_Storage

// Evaluation_Dot_Text_Storage_Invariants fixes the cursor text representation.
func Evaluation_Dot_Text_Storage_Invariants(
	value Evaluation_Dot_Text_Storage, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value.Value), OUTPUT_SIZE_MINIMUM, SOURCE_SIZE_UNVALIDATED_MAXIMUM).
		Ensure()
}

// Evaluation_Dot_Values_Storage prevents a mutable cursor from proving stale contents.
type Evaluation_Dot_Values_Storage Values_Storage

// Evaluation_Dot_Values_Storage_Invariants fixes the cursor sequence representation.
func Evaluation_Dot_Values_Storage_Invariants(
	value Evaluation_Dot_Values_Storage, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value.Value), bytes.SLICE_SIZE_MINIMUM, VALUE_COUNT_UNVALIDATED_MAXIMUM,
		).
		Ensure()
}

// Evaluation_Dot_Fields_Storage prevents a mutable cursor from proving stale contents.
type Evaluation_Dot_Fields_Storage Fields_Storage

// Evaluation_Dot_Fields_Storage_Invariants fixes the cursor field representation.
func Evaluation_Dot_Fields_Storage_Invariants(
	value Evaluation_Dot_Fields_Storage, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value.Value), bytes.SLICE_SIZE_MINIMUM, VALUE_COUNT_UNVALIDATED_MAXIMUM,
		).
		Ensure()
}

// Evaluation_Dot_Function_Storage prevents a mutable cursor from proving stale contents.
type Evaluation_Dot_Function_Storage Function_Storage

// Evaluation_Dot_Function_Storage_Invariants fixes the cursor callable representation.
func Evaluation_Dot_Function_Storage_Invariants(
	value Evaluation_Dot_Function_Storage, namespace aver.Namespace,
) {
	Function_Storage_Invariants(Function_Storage(value), namespace)
}

// Evaluation_Current_Value keeps command result identity independent.
type Evaluation_Current_Value Value

// Evaluation_Current_Value_Invariants bounds every payload field inline for its own role.
func Evaluation_Current_Value_Invariants(value Evaluation_Current_Value, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value.Kind), uint8(VALUE_NIL), uint8(VALUE_FUNCTION)).
		Range_Uint64(uint64(value.Real), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Imaginary), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Int(len(value.Text), OUTPUT_SIZE_MINIMUM, OUTPUT_SIZE_MAXIMUM).
		Range_Int(len(value.Values), bytes.SLICE_SIZE_MINIMUM, VALUE_COUNT_MAXIMUM).
		Range_Int(len(value.Fields), bytes.SLICE_SIZE_MINIMUM, VALUE_COUNT_MAXIMUM).
		Sometimes(value.Function != nil, "Evaluation result holds a callable.").
		Ensure()
}

// Evaluation_Current_Kind_Storage prevents a mutable result from proving stale contents.
type Evaluation_Current_Kind_Storage Value_Kind_Storage

// Evaluation_Current_Kind_Storage_Invariants fixes the current selector representation.
func Evaluation_Current_Kind_Storage_Invariants(
	value Evaluation_Current_Kind_Storage, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value.Value), bits.WORD_8_MINIMUM, bits.WORD_8_MAXIMUM).
		Ensure()
}

// Evaluation_Current_Scalar_Storage prevents a mutable result from proving stale contents.
type Evaluation_Current_Scalar_Storage Value_Scalar

// Evaluation_Current_Scalar_Storage_Invariants fixes the current scalar representation.
func Evaluation_Current_Scalar_Storage_Invariants(
	value Evaluation_Current_Scalar_Storage, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value.Real), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Imaginary), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Evaluation_Current_Text_Storage prevents a mutable result from proving stale contents.
type Evaluation_Current_Text_Storage Text_Storage

// Evaluation_Current_Text_Storage_Invariants fixes the current text representation.
func Evaluation_Current_Text_Storage_Invariants(
	value Evaluation_Current_Text_Storage, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value.Value), OUTPUT_SIZE_MINIMUM, SOURCE_SIZE_UNVALIDATED_MAXIMUM).
		Ensure()
}

// Evaluation_Current_Values_Storage prevents a mutable result from proving stale contents.
type Evaluation_Current_Values_Storage Values_Storage

// Evaluation_Current_Values_Storage_Invariants fixes the current sequence representation.
func Evaluation_Current_Values_Storage_Invariants(
	value Evaluation_Current_Values_Storage, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value.Value), bytes.SLICE_SIZE_MINIMUM, VALUE_COUNT_UNVALIDATED_MAXIMUM,
		).
		Ensure()
}

// Evaluation_Current_Fields_Storage prevents a mutable result from proving stale contents.
type Evaluation_Current_Fields_Storage Fields_Storage

// Evaluation_Current_Fields_Storage_Invariants fixes the current field representation.
func Evaluation_Current_Fields_Storage_Invariants(
	value Evaluation_Current_Fields_Storage, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value.Value), bytes.SLICE_SIZE_MINIMUM, VALUE_COUNT_UNVALIDATED_MAXIMUM,
		).
		Ensure()
}

// Evaluation_Current_Function_Storage prevents a mutable result from proving stale contents.
type Evaluation_Current_Function_Storage Function_Storage

// Evaluation_Current_Function_Storage_Invariants fixes the current callable representation.
func Evaluation_Current_Function_Storage_Invariants(
	value Evaluation_Current_Function_Storage, namespace aver.Namespace,
) {
	Function_Storage_Invariants(Function_Storage(value), namespace)
}

// Evaluation_Values stores dot and current command value.
type Evaluation_Values struct {
	// Dot retains the pipeline cursor.
	Dot Evaluation_Dot_Value
	// Current retains the first command result.
	Current Evaluation_Current_Value
}

// Evaluation_Values_Invariants fixes iterative value-frame shape.
func Evaluation_Values_Invariants(value Evaluation_Values, namespace aver.Namespace) {
	Evaluation_Dot_Value_Invariants(value.Dot, namespace)
	Evaluation_Current_Value_Invariants(value.Current, namespace)
}

// Evaluation_Command_Reference keeps command continuation identity independent.
type Evaluation_Command_Reference Node_Reference

// Evaluation_Command_Reference_Invariants composes command continuation.
func Evaluation_Command_Reference_Invariants(
	value Evaluation_Command_Reference, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value), uint16(NODE_REFERENCE_MINIMUM),
			uint16(NODE_REFERENCE_MAXIMUM)).
		Ensure()
}

// Evaluation_Term_Reference keeps term continuation identity independent.
type Evaluation_Term_Reference Node_Reference

// Evaluation_Term_Reference_Invariants composes term continuation.
func Evaluation_Term_Reference_Invariants(
	value Evaluation_Term_Reference, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value), uint16(NODE_REFERENCE_MINIMUM),
			uint16(NODE_REFERENCE_MAXIMUM)).
		Ensure()
}

// Evaluation_References stores next command and next term.
type Evaluation_References struct {
	// Command retains the next pipeline command.
	Command Evaluation_Command_Reference
	// Term retains the next command argument.
	Term Evaluation_Term_Reference
}

// Evaluation_References_Invariants fixes iterative reference-frame shape.
func Evaluation_References_Invariants(value Evaluation_References, namespace aver.Namespace,
) {
	Evaluation_Command_Reference_Invariants(value.Command, namespace)
	Evaluation_Term_Reference_Invariants(value.Term, namespace)
}

// Evaluation_State_Value keeps pipeline state independent from both evaluation cursors.
type Evaluation_State_Value Value

// Evaluation_State_Value_Invariants bounds every payload field inline for its own role.
func Evaluation_State_Value_Invariants(value Evaluation_State_Value, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value.Kind), uint8(VALUE_NIL), uint8(VALUE_FUNCTION)).
		Range_Uint64(uint64(value.Real), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Imaginary), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Int(len(value.Text), OUTPUT_SIZE_MINIMUM, OUTPUT_SIZE_MAXIMUM).
		Range_Int(len(value.Values), bytes.SLICE_SIZE_MINIMUM, VALUE_COUNT_MAXIMUM).
		Range_Int(len(value.Fields), bytes.SLICE_SIZE_MINIMUM, VALUE_COUNT_MAXIMUM).
		Sometimes(value.Function != nil, "Evaluation state holds a callable.").
		Ensure()
}

// Evaluation_State_Kind_Storage prevents mutable state from proving stale contents.
type Evaluation_State_Kind_Storage Value_Kind_Storage

// Evaluation_State_Kind_Storage_Invariants fixes the state selector representation.
func Evaluation_State_Kind_Storage_Invariants(
	value Evaluation_State_Kind_Storage, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value.Value), bits.WORD_8_MINIMUM, bits.WORD_8_MAXIMUM).
		Ensure()
}

// Evaluation_State_Scalar_Storage prevents mutable state from proving stale contents.
type Evaluation_State_Scalar_Storage Value_Scalar

// Evaluation_State_Scalar_Storage_Invariants fixes the state scalar representation.
func Evaluation_State_Scalar_Storage_Invariants(
	value Evaluation_State_Scalar_Storage, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value.Real), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Imaginary), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Evaluation_State_Text_Storage prevents mutable state from proving stale contents.
type Evaluation_State_Text_Storage Text_Storage

// Evaluation_State_Text_Storage_Invariants fixes the state text representation.
func Evaluation_State_Text_Storage_Invariants(
	value Evaluation_State_Text_Storage, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value.Value), OUTPUT_SIZE_MINIMUM, SOURCE_SIZE_UNVALIDATED_MAXIMUM).
		Ensure()
}

// Evaluation_State_Values_Storage prevents mutable state from proving stale contents.
type Evaluation_State_Values_Storage Values_Storage

// Evaluation_State_Values_Storage_Invariants fixes the state sequence representation.
func Evaluation_State_Values_Storage_Invariants(
	value Evaluation_State_Values_Storage, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value.Value), bytes.SLICE_SIZE_MINIMUM, VALUE_COUNT_UNVALIDATED_MAXIMUM,
		).
		Ensure()
}

// Evaluation_State_Fields_Storage prevents mutable state from proving stale contents.
type Evaluation_State_Fields_Storage Fields_Storage

// Evaluation_State_Fields_Storage_Invariants fixes the state field representation.
func Evaluation_State_Fields_Storage_Invariants(
	value Evaluation_State_Fields_Storage, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value.Value), bytes.SLICE_SIZE_MINIMUM, VALUE_COUNT_UNVALIDATED_MAXIMUM,
		).
		Ensure()
}

// Evaluation_State_Function_Storage prevents mutable state from proving stale contents.
type Evaluation_State_Function_Storage Function_Storage

// Evaluation_State_Function_Storage_Invariants fixes the state callable representation.
func Evaluation_State_Function_Storage_Invariants(
	value Evaluation_State_Function_Storage, namespace aver.Namespace,
) {
	Function_Storage_Invariants(Function_Storage(value), namespace)
}

// Evaluation_States stores one pipeline state.
type Pipeline_State_Stored Pipeline_State

// Pipeline_State_Stored_Invariants preserves mutable pipeline state.
func Pipeline_State_Stored_Invariants(value Pipeline_State_Stored, namespace aver.Namespace,
) {
	Evaluation_State_Value_Invariants(Evaluation_State_Value(value.Value), namespace)
	Declaration_References_Stored_Invariants(
		Declaration_References_Stored(value.Declarations), namespace)
	aver.Tree(value, namespace).
		Enum_3_Uint8(uint8(value.Declaration_Count), uint8(DECLARATION_COUNT_NONE),
			uint8(DECLARATION_COUNT_ONE), uint8(DECLARATION_COUNT_TWO)).
		Sometimes(bool(value.Assignment), "Pipeline updates an existing variable.").
		Ensure()
}

// Evaluation_States stores one pipeline state.
type Evaluation_States struct {
	// Value retains pipeline result and declarations.
	Value Pipeline_State_Stored
}

// Evaluation_States_Invariants fixes iterative pipeline-state shape.
func Evaluation_States_Invariants(value Evaluation_States, namespace aver.Namespace) {
	Pipeline_State_Stored_Invariants(value.Value, namespace)
}

// Evaluation_Argument_Bases_Value retains argument-base storage.
type Evaluation_Argument_Bases_Value uint16

// Evaluation_Argument_Bases_Value_Invariants preserves the stack boundary.
func Evaluation_Argument_Bases_Value_Invariants(
	value Evaluation_Argument_Bases_Value, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value), uint16(ARGUMENT_COUNT_MINIMUM),
			uint16(ARGUMENT_COUNT_MAXIMUM)).
		Ensure()
}

// Evaluation_Argument_Bases stores one argument-stack opening.
type Evaluation_Argument_Bases struct {
	// Value isolates nested callback argument regions.
	Value Evaluation_Argument_Bases_Value
}

// Evaluation_Argument_Bases_Invariants fixes argument opening shape.
func Evaluation_Argument_Bases_Invariants(
	value Evaluation_Argument_Bases, namespace aver.Namespace,
) {
	Evaluation_Argument_Bases_Value_Invariants(value.Value, namespace)
}

// Evaluation_Control stores stage, mode, and state-machine flags.
type Evaluation_Control struct {
	// Stage drives the evaluator state machine.
	Stage Evaluation_Stage
	// Mode separates identifier calls from value calls.
	Mode Evaluation_Mode
	// Present records a prior pipeline result.
	Present Evaluation_Present
	// Chain records pending field selection.
	Chain Evaluation_Chain
}

// Evaluation_Control_Invariants fixes iterative control shape.
func Evaluation_Control_Invariants(value Evaluation_Control, namespace aver.Namespace) {
	Evaluation_Stage_Invariants(value.Stage, namespace)
	Evaluation_Mode_Invariants(value.Mode, namespace)
	Evaluation_Present_Invariants(value.Present, namespace)
	Evaluation_Chain_Invariants(value.Chain, namespace)
}

// Evaluation_Frame stores one suspended nested pipeline without recursion.
type Evaluation_Frame Evaluation_Frame_Fields

// Evaluation_Frame_Invariants composes fixed caller-owned frame storage.
func Evaluation_Frame_Invariants(value Evaluation_Frame, namespace aver.Namespace) {
	Evaluation_Nodes_Invariants(value.Nodes, namespace)
	Evaluation_Values_Invariants(value.Values, namespace)
	Evaluation_References_Stored_Invariants(
		Evaluation_References_Stored(value.References), namespace)
	Evaluation_States_Invariants(value.States, namespace)
	Evaluation_Argument_Bases_Stored_Invariants(
		Evaluation_Argument_Bases_Stored(value.Argument_Bases), namespace)
	Evaluation_Control_Stored_Invariants(Evaluation_Control_Stored(value.Control), namespace)
}

func generated_format_value(
	output Output_Storage, number Number_Storage, generated Generated_Storage,
	generated_count Workspace_Generated_Count_Storage_Pointer, format Value_Format_Frames,
	value_state Value, verb Format_Verb, width Format_Width, zero_padding Format_Zero_Padding,
	alternate Format_Alternate, float_parse Float_Parse_Workspace_Pointer,
	float_text Float_Text_Workspace_Pointer,
) (status Formatting_Status) {
	defer func() {
		Formatting_Status_Invariants(status, "generated_format_value.status")
	}()
	Output_Storage_Invariants(output, "generated_format_value.output")
	Number_Storage_Invariants(number, "generated_format_value.number")
	Generated_Storage_Invariants(generated, "generated_format_value.generated")
	Workspace_Generated_Count_Storage_Pointer_Invariants(
		generated_count, "generated_format_value.generated_count")
	Value_Format_Frames_Invariants(format, "generated_format_value.format")
	Value_Invariants(value_state, "generated_format_value.value_state")
	Format_Verb_Invariants(verb, "generated_format_value.verb")
	Format_Width_Invariants(width, "generated_format_value.width")
	Format_Zero_Padding_Invariants(zero_padding, "generated_format_value.zero_padding")
	Format_Alternate_Invariants(alternate, "generated_format_value.alternate")
	Float_Parse_Workspace_Pointer_Invariants(float_parse, "generated_format_value.float_parse")
	Float_Text_Workspace_Pointer_Invariants(float_text, "generated_format_value.float_text")
	value := Value(value_state)
	kind := Value_Kind(value.Kind)
	switch verb {
	case 'v', 'g':
		if verb == 'g' {
			if kind < VALUE_INTEGER {
				return FORMATTING_STATUS_INVALID
			}
			if kind > VALUE_COMPLEX {
				return FORMATTING_STATUS_INVALID
			}
		}
		return Formatting_Status(generated_append_value(
			output, number, generated, generated_count, format, value, float_parse,
			float_text))
	case 's':
		if kind != VALUE_TEXT {
			return FORMATTING_STATUS_INVALID
		}
		return Formatting_Status(
			generated_append_bytes(generated, generated_count, Text(value.Text)))
	case 'q':
		if kind != VALUE_TEXT {
			return FORMATTING_STATUS_INVALID
		}
		return Formatting_Status(generated_format_quote(
			generated, generated_count, Text(value.Text), alternate))
	case 'b', 'd', 'o', 'x', 'X':
		if kind != VALUE_INTEGER {
			if kind != VALUE_UNSIGNED {
				return FORMATTING_STATUS_INVALID
			}
		}
		base, uppercase := integer_format_base(Integer_Format_Verb(verb))
		return Formatting_Status(generated_format_integer(
			number, generated, generated_count, Integer_Word_Value(value), base,
			uppercase, width, zero_padding))
	case 'T':
		storage := (*Workspace_Generated_Count_Storage)(generated_count)
		return Formatting_Status(generated_format_type(
			generated, (*Typed_Generated_Count)(storage), kind))
	}
	return FORMATTING_STATUS_INVALID
}

func generated_format_integer(number Number_Storage, generated Generated_Storage,
	generated_count Workspace_Generated_Count_Storage_Pointer, value_state Integer_Word_Value,
	base Integer_Format_Base, uppercase Integer_Format_Uppercase, width Format_Width,
	zero_padding Format_Zero_Padding,
) (status Generated_Status) {
	defer func() {
		Generated_Status_Invariants(status, "generated_format_integer.status")
	}()
	Number_Storage_Invariants(number, "generated_format_integer.number")
	Generated_Storage_Invariants(generated, "generated_format_integer.generated")
	Workspace_Generated_Count_Storage_Pointer_Invariants(
		generated_count, "generated_format_integer.generated_count")
	Integer_Word_Value_Invariants(value_state, "generated_format_integer.value_state")
	Integer_Format_Base_Invariants(base, "generated_format_integer.base")
	Integer_Format_Uppercase_Invariants(uppercase, "generated_format_integer.uppercase")
	Format_Width_Invariants(width, "generated_format_integer.width")
	Format_Zero_Padding_Invariants(zero_padding, "generated_format_integer.zero_padding")
	value := Value(value_state)
	count := SOURCE_SIZE_MINIMUM
	if Value_Kind(value.Kind) == VALUE_INTEGER {
		count = int(strconv.Format_Integer_Into(
			strconv.Buffer(number), strconv.Signed_Integer(int64(value.Real)),
			strconv.Base(base)))
	} else {
		count = int(strconv.Format_Unsigned_Integer_Into(
			strconv.Buffer(number), strconv.Unsigned_Integer(value.Real),
			strconv.Base(base)))
	}
	if bool(uppercase) {
		for digit_index := SOURCE_SIZE_MINIMUM; digit_index < count; digit_index++ {
			if number[digit_index] >= 'a' {
				number[digit_index] -= 'a' - 'A'
			}
		}
	}
	return generated_append_padded(generated, generated_count, Formatted_Integer_Text_Validated{
		Value: Formatted_Integer_Text_Proof_Stored(number[:count]),
	}, width, zero_padding)
}

func generated_append_padded(
	generated Generated_Storage, generated_count Workspace_Generated_Count_Storage_Pointer,
	source_value Formatted_Integer_Text_Validated, width Format_Width,
	zero_padding Format_Zero_Padding,
) (status Generated_Status) {
	defer func() {
		Generated_Status_Invariants(status, "generated_append_padded.status")
	}()
	Generated_Storage_Invariants(generated, "generated_append_padded.generated")
	Workspace_Generated_Count_Storage_Pointer_Invariants(
		generated_count, "generated_append_padded.generated_count")
	Formatted_Integer_Text_Validated_Invariants(
		source_value, "generated_append_padded.source_value")
	Format_Width_Invariants(width, "generated_append_padded.width")
	Format_Zero_Padding_Invariants(zero_padding, "generated_append_padded.zero_padding")
	source := Text(Formatted_Integer_Text_Validated_Value(source_value.Value))
	padding_count := int(width) - len(source)
	if padding_count < OUTPUT_SIZE_MINIMUM {
		padding_count = OUTPUT_SIZE_MINIMUM
	}
	padding := " "
	if bool(zero_padding) {
		padding = "0"
	}
	negative := len(source) > SOURCE_SIZE_MINIMUM
	if negative {
		negative = source[SOURCE_SIZE_MINIMUM] == '-'
	}
	if negative {
		if bool(zero_padding) {
			if generated_append_bytes(
				generated, generated_count, source[:SOURCE_POSITION_INCREMENT],
			) != GENERATED_STATUS_OK {
				return GENERATED_STATUS_LIMIT_EXCEEDED
			}
			source = source[SOURCE_POSITION_INCREMENT:]
		}
	}
	for padding_index := OUTPUT_SIZE_MINIMUM; padding_index < padding_count; padding_index++ {
		append_status := generated_append_text(
			generated, generated_count, Generated_Text(padding))
		if append_status != GENERATED_STATUS_OK {
			return GENERATED_STATUS_LIMIT_EXCEEDED
		}
	}
	return generated_append_bytes(generated, generated_count, source)
}

func generated_format_type(
	generated Generated_Storage, generated_count Typed_Generated_Count_Pointer,
	kind Value_Kind,
) (status Generated_Status) {
	defer func() {
		Generated_Status_Invariants(status, "generated_format_type.status")
	}()
	Generated_Storage_Invariants(generated, "generated_format_type.generated")
	Typed_Generated_Count_Pointer_Invariants(
		generated_count, "generated_format_type.generated_count")
	Value_Kind_Invariants(kind, "generated_format_type.kind")
	name := ""
	switch kind {
	case VALUE_NIL:
		name = "<nil>"
	case VALUE_BOOLEAN:
		name = "bool"
	case VALUE_INTEGER:
		name = "int"
	case VALUE_UNSIGNED:
		name = "uint64"
	case VALUE_FLOAT:
		name = "float64"
	case VALUE_COMPLEX:
		name = "complex128"
	case VALUE_TEXT:
		name = "string"
	case VALUE_SEQUENCE:
		name = "[]template.Value"
	case VALUE_OBJECT:
		name = "template.Object"
	case VALUE_MAP:
		name = "template.Map"
	case VALUE_FUNCTION:
		name = "template.Function"
	}
	typed := (*Typed_Generated_Count)(generated_count)
	return generated_append_text(generated,
		(*Workspace_Generated_Count_Storage)(typed), Generated_Text(name))
}

func generated_format_quote(
	generated Generated_Storage, generated_count Workspace_Generated_Count_Storage_Pointer,
	source Text, alternate Format_Alternate,
) (status Generated_Status) {
	defer func() {
		Generated_Status_Invariants(status, "generated_format_quote.status")
	}()
	Generated_Storage_Invariants(generated, "generated_format_quote.generated")
	Workspace_Generated_Count_Storage_Pointer_Invariants(
		generated_count, "generated_format_quote.generated_count")
	Text_Invariants(source, "generated_format_quote.source")
	Format_Alternate_Invariants(alternate, "generated_format_quote.alternate")
	step := Written_Generated_Count_Pointer(
		(*Written_Generated_Count)((*Workspace_Generated_Count_Storage)(generated_count)))
	if bool(alternate) {
		if bool(quote_backquote_safe(source)) {
			if generated_append_text(generated, generated_count,
				"`") != GENERATED_STATUS_OK {
				return GENERATED_STATUS_LIMIT_EXCEEDED
			}
			if generated_append_bytes(generated, generated_count,
				source) != GENERATED_STATUS_OK {
				return GENERATED_STATUS_LIMIT_EXCEEDED
			}
			return generated_append_text(generated, generated_count, "`")
		}
	}
	if generated_append_text(generated, generated_count, `"`) != GENERATED_STATUS_OK {
		return GENERATED_STATUS_LIMIT_EXCEEDED
	}
	for source_index := SOURCE_SIZE_MINIMUM; source_index < len(source); {
		value := source[source_index]
		if value < byte(utf8.CHARACTER_SELF) {
			status = generated_append_quoted_ascii(generated,
				(*Escape_Generated_Count)((*Written_Generated_Count)(step)),
				ASCII_Byte(value))
			source_index++
		} else {
			character, size := utf8.Decode_Character(utf8.Bytes(source[source_index:]))
			if bool(ucd.Is_Print(ucd.Character(character))) {
				status = generated_append_bytes(generated, generated_count,
					source[source_index:source_index+int(size)])
			} else {
				written := (*Written_Generated_Count)(step)
				status = generated_append_js_unicode(generated,
					(*Unicode_Generated_Count)(written),
					utf8.Decoded_Character(character))
			}
			source_index += int(size)
		}
		if status != GENERATED_STATUS_OK {
			return status
		}
	}
	return generated_append_text(generated, generated_count, `"`)
}

func generated_append_quoted_ascii(
	generated Generated_Storage, generated_count Escape_Generated_Count_Pointer,
	value_value ASCII_Byte,
) (status Generated_Status) {
	defer func() {
		Generated_Status_Invariants(status, "generated_append_quoted_ascii.status")
	}()
	Generated_Storage_Invariants(generated, "generated_append_quoted_ascii.generated")
	Escape_Generated_Count_Pointer_Invariants(
		generated_count, "generated_append_quoted_ascii.generated_count")
	ASCII_Byte_Invariants(value_value, "generated_append_quoted_ascii.value_value")
	derived := Workspace_Generated_Count_Storage_Pointer(
		(*Workspace_Generated_Count_Storage)((*Escape_Generated_Count)(generated_count)))
	value := byte(value_value)
	replacement := ""
	switch value {
	case '\\':
		replacement = `\\`
	case '"':
		replacement = `\"`
	case '\n':
		replacement = `\n`
	case '\r':
		replacement = `\r`
	case '\t':
		replacement = `\t`
	case '\b':
		replacement = `\b`
	case '\f':
		replacement = `\f`
	default:
		if value < ' ' {
			return generated_append_control_byte(
				generated, generated_count, Quoted_Control_Byte(value))
		}
		if value == byte(utf8.CHARACTER_SELF)-utf8.CHARACTER_SIZE_MINIMUM {
			return generated_append_text(generated, derived, `\x7F`)
		}
	}
	if replacement != "" {
		return generated_append_text(
			generated, derived, Generated_Text(replacement))
	}
	return generated_append_bytes(generated, derived, []byte{value})
}

func generated_append_control_byte(
	generated Generated_Storage, generated_count Escape_Generated_Count_Pointer,
	value_value Quoted_Control_Byte,
) (status Generated_Status) {
	defer func() {
		Generated_Status_Invariants(status, "generated_append_control_byte.status")
	}()
	Generated_Storage_Invariants(generated, "generated_append_control_byte.generated")
	Escape_Generated_Count_Pointer_Invariants(
		generated_count, "generated_append_control_byte.generated_count")
	Quoted_Control_Byte_Invariants(value_value, "generated_append_control_byte.value_value")
	derived := Workspace_Generated_Count_Storage_Pointer(
		(*Workspace_Generated_Count_Storage)((*Escape_Generated_Count)(generated_count)))
	value := byte(value_value)
	encoded := [len(`\x00`)]byte{
		'\\', 'x',
		TEMPLATE_HEXADECIMAL_DIGITS[value>>NUMBER_BASE_HEXADECIMAL_DIGIT_BIT_COUNT],
		TEMPLATE_HEXADECIMAL_DIGITS[value&HEXADECIMAL_DIGIT_MASK],
	}
	return generated_append_bytes(generated, derived, encoded[:])
}

func generated_append_value(
	workspace_output Output_Storage, number Number_Storage, generated Generated_Storage,
	generated_count Workspace_Generated_Count_Storage_Pointer, format Value_Format_Frames,
	value_state Value, float_parse Float_Parse_Workspace_Pointer,
	float_text Float_Text_Workspace_Pointer,
) (status Generated_Status) {
	defer func() {
		Generated_Status_Invariants(status, "generated_append_value.status")
	}()
	Output_Storage_Invariants(workspace_output, "generated_append_value.workspace_output")
	Number_Storage_Invariants(number, "generated_append_value.number")
	Generated_Storage_Invariants(generated, "generated_append_value.generated")
	Workspace_Generated_Count_Storage_Pointer_Invariants(
		generated_count, "generated_append_value.generated_count")
	Value_Format_Frames_Invariants(format, "generated_append_value.format")
	Value_Invariants(value_state, "generated_append_value.value_state")
	Float_Parse_Workspace_Pointer_Invariants(float_parse, "generated_append_value.float_parse")
	Float_Text_Workspace_Pointer_Invariants(float_text, "generated_append_value.float_text")
	start := int(Workspace_Generated_Count_Storage_Value(generated_count.Value))
	// One value formats into a full output window, so a short remainder refuses.
	end := start + OUTPUT_SIZE_MAXIMUM
	if end > len(generated) {
		return GENERATED_STATUS_LIMIT_EXCEEDED
	}
	output := workspace_output
	workspace_output = Output_Storage(generated[start:end])
	written, value_status := output_value(workspace_output, number, format,
		Output_Count_Staged{Value: Output_Count_Proof_Stored(OUTPUT_SIZE_MINIMUM)},
		Value(value_state), float_parse, float_text)
	workspace_output = output
	if value_status == OUTPUT_STATUS_TOO_LARGE {
		return GENERATED_STATUS_LIMIT_EXCEEDED
	}
	aver.Always(value_status == OUTPUT_STATUS_OK,
		"Validated builtin values cannot fail scalar or composite formatting.")
	generated_count.Value = generated_count.
		Value + Workspace_Generated_Count_Stored(Output_Count_Staged_Value(written.Value))
	return GENERATED_STATUS_OK
}

func generated_append_bytes(
	generated Generated_Storage, generated_count Workspace_Generated_Count_Storage_Pointer,
	source Text,
) (status Generated_Status) {
	defer func() {
		Generated_Status_Invariants(status, "generated_append_bytes.status")
	}()
	Generated_Storage_Invariants(generated, "generated_append_bytes.generated")
	Workspace_Generated_Count_Storage_Pointer_Invariants(
		generated_count, "generated_append_bytes.generated_count")
	Text_Invariants(source, "generated_append_bytes.source")
	count := int(Workspace_Generated_Count_Storage_Value(generated_count.Value))
	if count > len(generated)-len(source) {
		return GENERATED_STATUS_LIMIT_EXCEEDED
	}
	copy(generated[count:], source)
	generated_count.Value = generated_count.
		Value + Workspace_Generated_Count_Stored(len(source))
	return GENERATED_STATUS_OK
}

func generated_append_text(
	generated Generated_Storage, generated_count Workspace_Generated_Count_Storage_Pointer,
	source Generated_Text,
) (status Generated_Status) {
	defer func() {
		Generated_Status_Invariants(status, "generated_append_text.status")
	}()
	Generated_Storage_Invariants(generated, "generated_append_text.generated")
	Workspace_Generated_Count_Storage_Pointer_Invariants(
		generated_count, "generated_append_text.generated_count")
	Generated_Text_Invariants(source, "generated_append_text.source")
	count := int(Workspace_Generated_Count_Storage_Value(generated_count.Value))
	if count > len(generated)-len(source) {
		return GENERATED_STATUS_LIMIT_EXCEEDED
	}
	copy(generated[count:], source)
	generated_count.Value = generated_count.
		Value + Workspace_Generated_Count_Stored(len(source))
	return GENERATED_STATUS_OK
}

func generated_text_value(
	generated Generated_Storage, generated_count Workspace_Generated_Count_Storage_Pointer,
	start_value Generated_Start_Validated,
) (_ Generated_Text_Value, status Generated_Status) {
	defer func() { Generated_Status_Invariants(status, "generated_text_value.status") }()
	Generated_Storage_Invariants(generated, "generated_text_value.generated")
	Workspace_Generated_Count_Storage_Pointer_Invariants(
		generated_count, "generated_text_value.generated_count")
	Generated_Start_Validated_Invariants(start_value, "generated_text_value.start_value")
	var result Generated_Text_Value
	defer func() { Generated_Text_Value_Invariants(result, "generated_text_value.result") }()
	start := Generated_Count(Generated_Start_Validated_Value(start_value.Value))
	end := Generated_Count(Workspace_Generated_Count_Storage_Value(generated_count.Value))
	if int(end-start) > OUTPUT_SIZE_MAXIMUM {
		return Generated_Text_Value{}, GENERATED_STATUS_LIMIT_EXCEEDED
	}
	result = Generated_Text_Value(Value_Of_Text(Text(generated[start:end])))
	return result, GENERATED_STATUS_OK
}

func generated_escape_html(
	generated Generated_Storage, generated_count Escaped_Generated_Count_Pointer,
	source Text,
) (status Generated_Status) {
	defer func() {
		Generated_Status_Invariants(status, "generated_escape_html.status")
	}()
	Generated_Storage_Invariants(generated, "generated_escape_html.generated")
	Escaped_Generated_Count_Pointer_Invariants(
		generated_count, "generated_escape_html.generated_count")
	Text_Invariants(source, "generated_escape_html.source")
	plain := Workspace_Generated_Count_Storage_Pointer(
		(*Workspace_Generated_Count_Storage)((*Escaped_Generated_Count)(generated_count)))
	for source_index := range source {
		replacement := ""
		switch source[source_index] {
		case '&':
			replacement = "&amp;"
		case '\'':
			replacement = "&#39;"
		case '<':
			replacement = "&lt;"
		case '>':
			replacement = "&gt;"
		case '"':
			replacement = "&#34;"
		}
		if replacement != "" {
			if generated_append_text(
				generated, plain, Generated_Text(replacement),
			) != GENERATED_STATUS_OK {
				return GENERATED_STATUS_LIMIT_EXCEEDED
			}
		} else if generated_append_bytes(generated, plain,
			source[source_index:source_index+SOURCE_POSITION_INCREMENT],
		) != GENERATED_STATUS_OK {
			return GENERATED_STATUS_LIMIT_EXCEEDED
		}
	}
	return GENERATED_STATUS_OK
}

func generated_escape_js(
	generated Generated_Storage, generated_count Escaped_Generated_Count_Pointer,
	source Text,
) (status Generated_Status) {
	defer func() {
		Generated_Status_Invariants(status, "generated_escape_js.status")
	}()
	Generated_Storage_Invariants(generated, "generated_escape_js.generated")
	Escaped_Generated_Count_Pointer_Invariants(
		generated_count, "generated_escape_js.generated_count")
	Text_Invariants(source, "generated_escape_js.source")
	plain := Workspace_Generated_Count_Storage_Pointer(
		(*Workspace_Generated_Count_Storage)((*Escaped_Generated_Count)(generated_count)))
	step := Written_Generated_Count_Pointer(
		(*Written_Generated_Count)((*Workspace_Generated_Count_Storage)(plain)))
	for source_index := SOURCE_SIZE_MINIMUM; source_index < len(source); {
		value := source[source_index]
		if value < byte(utf8.CHARACTER_SELF) {
			status = generated_escape_js_ascii(
				generated, step, ASCII_Byte(value))
			if status != GENERATED_STATUS_OK {
				return status
			}
			source_index++
			continue
		}
		character, size := utf8.Decode_Character(utf8.Bytes(source[source_index:]))
		if bool(ucd.Is_Print(ucd.Character(character))) {
			status = generated_append_bytes(generated, plain,
				source[source_index:source_index+int(size)])
		} else {
			status = generated_append_js_unicode(generated,
				(*Unicode_Generated_Count)((*Written_Generated_Count)(step)),
				utf8.Decoded_Character(character))
		}
		if status != GENERATED_STATUS_OK {
			return status
		}
		source_index += int(size)
	}
	return GENERATED_STATUS_OK
}

func generated_escape_js_ascii(
	generated Generated_Storage, generated_count Written_Generated_Count_Pointer,
	value_value ASCII_Byte,
) (status Generated_Status) {
	defer func() {
		Generated_Status_Invariants(status, "generated_escape_js_ascii.status")
	}()
	Generated_Storage_Invariants(generated, "generated_escape_js_ascii.generated")
	Written_Generated_Count_Pointer_Invariants(
		generated_count, "generated_escape_js_ascii.generated_count")
	ASCII_Byte_Invariants(value_value, "generated_escape_js_ascii.value_value")
	derived := Workspace_Generated_Count_Storage_Pointer(
		(*Workspace_Generated_Count_Storage)((*Written_Generated_Count)(generated_count)))
	value := byte(value_value)
	replacement := ""
	switch value {
	case '\\':
		replacement = `\\`
	case '\'':
		replacement = `\'`
	case '"':
		replacement = `\"`
	case '<':
		replacement = `\u003C`
	case '>':
		replacement = `\u003E`
	case '&':
		replacement = `\u0026`
	case '=':
		replacement = `\u003D`
	default:
		if value < ' ' {
			written := (*Written_Generated_Count)(generated_count)
			return generated_append_js_unicode(generated,
				(*Unicode_Generated_Count)(written),
				utf8.Decoded_Character(value))
		}
	}
	if replacement != "" {
		return generated_append_text(
			generated, derived, Generated_Text(replacement))
	}
	return generated_append_bytes(generated, derived, []byte{value})
}

func generated_append_js_unicode(
	generated Generated_Storage, generated_count Unicode_Generated_Count_Pointer,
	value_value utf8.Decoded_Character,
) (status Generated_Status) {
	defer func() {
		Generated_Status_Invariants(status, "generated_append_js_unicode.status")
	}()
	Generated_Storage_Invariants(generated, "generated_append_js_unicode.generated")
	Unicode_Generated_Count_Pointer_Invariants(
		generated_count, "generated_append_js_unicode.generated_count")
	utf8.Decoded_Character_Invariants(value_value, "generated_append_js_unicode.value_value")
	derived := Workspace_Generated_Count_Storage_Pointer(
		(*Workspace_Generated_Count_Storage)((*Unicode_Generated_Count)(generated_count)))
	value := uint32(value_value)
	hexadecimal_count := strconv.SHORT_UNICODE_DIGIT_COUNT
	for value >= uint32(bits.CARRY_MAXIMUM)<<
		(hexadecimal_count*NUMBER_BASE_HEXADECIMAL_DIGIT_BIT_COUNT) {
		hexadecimal_count++
	}
	if generated_append_text(generated, derived, `\u`) != GENERATED_STATUS_OK {
		return GENERATED_STATUS_LIMIT_EXCEEDED
	}
	digit_index := hexadecimal_count - HEXADECIMAL_DIGIT_COUNT_INCREMENT
	for digit_index >= SOURCE_SIZE_MINIMUM {
		digit := value >> (digit_index * NUMBER_BASE_HEXADECIMAL_DIGIT_BIT_COUNT) &
			HEXADECIMAL_DIGIT_MASK
		if generated_append_bytes(
			generated, derived, []byte{TEMPLATE_HEXADECIMAL_DIGITS[digit]},
		) != GENERATED_STATUS_OK {
			return GENERATED_STATUS_LIMIT_EXCEEDED
		}
		digit_index--
	}
	return GENERATED_STATUS_OK
}

func generated_escape_url_query(
	generated Generated_Storage, generated_count Escaped_Generated_Count_Pointer,
	source Text,
) (status Generated_Status) {
	defer func() {
		Generated_Status_Invariants(status, "generated_escape_url_query.status")
	}()
	Generated_Storage_Invariants(generated, "generated_escape_url_query.generated")
	Escaped_Generated_Count_Pointer_Invariants(
		generated_count, "generated_escape_url_query.generated_count")
	Text_Invariants(source, "generated_escape_url_query.source")
	plain := Workspace_Generated_Count_Storage_Pointer(
		(*Workspace_Generated_Count_Storage)((*Escaped_Generated_Count)(generated_count)))
	for source_index := range source {
		value := source[source_index]
		if bool(url_query_safe(utf8.Byte(value))) {
			status = generated_append_bytes(generated, plain,
				source[source_index:source_index+SOURCE_POSITION_INCREMENT])
		} else if value == ' ' {
			status = generated_append_text(generated, plain, "+")
		} else {
			high := value >> NUMBER_BASE_HEXADECIMAL_DIGIT_BIT_COUNT
			encoded := [URL_QUERY_ESCAPE_SIZE]byte{
				'%', TEMPLATE_HEXADECIMAL_DIGITS[high],
				TEMPLATE_HEXADECIMAL_DIGITS[value&HEXADECIMAL_DIGIT_MASK],
			}
			status = generated_append_bytes(generated, plain, encoded[:])
		}
		if status != GENERATED_STATUS_OK {
			return status
		}
	}
	return GENERATED_STATUS_OK
}

// Truth_Value is one builtin comparison result or the empty value a failure leaves.
type Truth_Value Value

// Truth_Value_Invariants fixes every payload except the truth word.
func Truth_Value_Invariants(value Truth_Value, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value.Kind), uint8(VALUE_NIL), uint8(VALUE_BOOLEAN)).
		Enum_Uint64(uint64(value.Real), bits.WORD_64_MINIMUM, bits.CARRY_MAXIMUM).
		Ensure()
	aver.Always(uint64(value.Imaginary) == bits.WORD_64_MINIMUM,
		"A builtin comparison leaves the imaginary word empty.")
	aver.Always(
		len(value.Text) == OUTPUT_SIZE_MINIMUM, "A builtin comparison leaves text empty.",
	)
	aver.Always(len(value.Values) == bytes.SLICE_SIZE_MINIMUM,
		"A builtin comparison leaves the sequence empty.")
	aver.Always(len(value.Fields) == bytes.SLICE_SIZE_MINIMUM,
		"A builtin comparison leaves the members empty.")
}

// Count_Value is one builtin element count or the empty value a failure leaves.
type Count_Value Value

// Count_Value_Invariants bounds the count by the largest collection or text.
func Count_Value_Invariants(value Count_Value, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value.Kind), uint8(VALUE_NIL), uint8(VALUE_INTEGER)).
		Range_Uint64(uint64(value.Real), bits.WORD_64_MINIMUM, uint64(VALUE_COUNT_MAXIMUM)).
		Ensure()
	aver.Always(uint64(value.Imaginary) == bits.WORD_64_MINIMUM,
		"A builtin count leaves the imaginary word empty.")
	aver.Always(len(value.Text) == OUTPUT_SIZE_MINIMUM, "A builtin count leaves text empty.")
	aver.Always(len(value.Values) == bytes.SLICE_SIZE_MINIMUM,
		"A builtin count leaves the sequence empty.")
	aver.Always(len(value.Fields) == bytes.SLICE_SIZE_MINIMUM,
		"A builtin count leaves the members empty.")
}

// Generated_Text_Value is one generated builtin text or the empty value a failure leaves.
type Generated_Text_Value Value

// Generated_Text_Value_Invariants bounds generated text by execution output.
func Generated_Text_Value_Invariants(value Generated_Text_Value, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value.Kind), uint8(VALUE_NIL), uint8(VALUE_TEXT)).
		Range_Int(len(value.Text), OUTPUT_SIZE_MINIMUM, OUTPUT_SIZE_MAXIMUM).
		Ensure()
	aver.Always(uint64(value.Real) == bits.WORD_64_MINIMUM,
		"Generated builtin text leaves the primary word empty.")
	aver.Always(uint64(value.Imaginary) == bits.WORD_64_MINIMUM,
		"Generated builtin text leaves the imaginary word empty.")
	aver.Always(len(value.Values) == bytes.SLICE_SIZE_MINIMUM,
		"Generated builtin text leaves the sequence empty.")
	aver.Always(len(value.Fields) == bytes.SLICE_SIZE_MINIMUM,
		"Generated builtin text leaves the members empty.")
}

// Each kind owns its payload fields; a stray payload marks a forged value.
func value_payload_owned(value_state Kinded_Value_Unvalidated) (owned Value_Validity) {
	defer func() { Value_Validity_Invariants(owned, "value_payload_owned.owned") }()
	Kinded_Value_Unvalidated_Invariants(value_state, "value_payload_owned.value_state")
	value := Value(value_state)
	kind := Value_Kind(value.Kind)
	real_owned := kind >= VALUE_BOOLEAN
	if kind > VALUE_COMPLEX {
		real_owned = false
	}
	if !real_owned {
		if value.Real != Value_Scalar_Real(bits.WORD_64_MINIMUM) {
			return false
		}
	}
	if kind == VALUE_BOOLEAN {
		if value.Real > Value_Scalar_Real(bits.CARRY_MAXIMUM) {
			return false
		}
	}
	if kind != VALUE_COMPLEX {
		if value.Imaginary != Value_Scalar_Imaginary(bits.WORD_64_MINIMUM) {
			return false
		}
	}
	if kind != VALUE_TEXT {
		if len(Text(value.Text)) != OUTPUT_SIZE_MINIMUM {
			return false
		}
	}
	if kind != VALUE_SEQUENCE {
		if len(Values(value.Values)) != bytes.SLICE_SIZE_MINIMUM {
			return false
		}
	}
	if kind != VALUE_OBJECT {
		if kind != VALUE_MAP {
			if len(Fields(value.Fields)) != bytes.SLICE_SIZE_MINIMUM {
				return false
			}
		}
	}
	if kind != VALUE_FUNCTION {
		return Value_Validity(Function_Call(value.Function) == nil)
	}
	return true
}

// Composite_Value is one sequence, object, or map the formatter walks.
type Composite_Value Value

// Composite_Value_Invariants fixes collection kinds and their unowned scalar payloads.
func Composite_Value_Invariants(value Composite_Value, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(uint8(value.Kind), uint8(VALUE_SEQUENCE), uint8(VALUE_OBJECT),
			uint8(VALUE_MAP)).
		Range_Int(len(value.Values), bytes.SLICE_SIZE_MINIMUM, VALUE_COUNT_MAXIMUM).
		Range_Int(len(value.Fields), bytes.SLICE_SIZE_MINIMUM, VALUE_COUNT_MAXIMUM).
		Ensure()
	aver.Always(uint64(value.Real) == bits.WORD_64_MINIMUM,
		"Validation leaves a collection's primary word empty.")
	aver.Always(uint64(value.Imaginary) == bits.WORD_64_MINIMUM,
		"Validation leaves a collection's imaginary word empty.")
	aver.Always(len(value.Text) == OUTPUT_SIZE_MINIMUM,
		"Validation leaves a collection's text empty.")
}

// Scalar_Value is one value the formatter writes without walking children.
type Scalar_Value Value

// Scalar_Value_Invariants fixes the empty collection payloads of every scalar kind.
func Scalar_Value_Invariants(value Scalar_Value, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value.Kind), uint8(VALUE_NIL), uint8(VALUE_FUNCTION)).
		Range_Uint64(uint64(value.Real), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Imaginary), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Int(len(value.Text), OUTPUT_SIZE_MINIMUM, OUTPUT_SIZE_MAXIMUM).
		Ensure()
	aver.Always(len(value.Values) == bytes.SLICE_SIZE_MINIMUM,
		"Validation leaves a scalar's sequence empty.")
	aver.Always(len(value.Fields) == bytes.SLICE_SIZE_MINIMUM,
		"Validation leaves a scalar's members empty.")
}

// Integer_Word_Value is one signed or unsigned integer the formatter spells.
type Integer_Word_Value Value

// Integer_Word_Value_Invariants fixes the two integer kinds and their unowned payloads.
func Integer_Word_Value_Invariants(value Integer_Word_Value, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value.Kind), uint8(VALUE_INTEGER), uint8(VALUE_UNSIGNED)).
		Range_Uint64(uint64(value.Real), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
	aver.Always(uint64(value.Imaginary) == bits.WORD_64_MINIMUM,
		"Validation leaves an integer's imaginary word empty.")
	aver.Always(len(value.Text) == OUTPUT_SIZE_MINIMUM,
		"Validation leaves an integer's text empty.")
	aver.Always(len(value.Values) == bytes.SLICE_SIZE_MINIMUM,
		"Validation leaves an integer's sequence empty.")
	aver.Always(len(value.Fields) == bytes.SLICE_SIZE_MINIMUM,
		"Validation leaves an integer's members empty.")
}

// Range_Key_Value is one range key: an element position, or nil beside an integer count.
type Range_Key_Value Value

// Range_Key_Value_Invariants bounds a position by the largest collection.
func Range_Key_Value_Invariants(value Range_Key_Value, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value.Kind), uint8(VALUE_NIL), uint8(VALUE_INTEGER)).
		Range_Uint64(
			uint64(value.Real), bits.WORD_64_MINIMUM, uint64(MAP_ENTRY_INDEX_MAXIMUM),
		).
		Ensure()
	aver.Always(uint64(value.Imaginary) == bits.WORD_64_MINIMUM,
		"A range key leaves the imaginary word empty.")
	aver.Always(len(value.Text) == OUTPUT_SIZE_MINIMUM, "A range key leaves text empty.")
	aver.Always(len(value.Values) == bytes.SLICE_SIZE_MINIMUM,
		"A range key leaves the sequence empty.")
	aver.Always(len(value.Fields) == bytes.SLICE_SIZE_MINIMUM,
		"A range key leaves the members empty.")
}

// Integer_Result_Value is one parsed signed integer, or the empty value a failure leaves.
type Integer_Result_Value Value

// Integer_Result_Value_Invariants fixes the payloads a signed result never owns.
func Integer_Result_Value_Invariants(value Integer_Result_Value, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value.Kind), uint8(VALUE_NIL), uint8(VALUE_INTEGER)).
		Range_Uint64(uint64(value.Real), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
	aver.Always(uint64(value.Imaginary) == bits.WORD_64_MINIMUM,
		"A parsed integer leaves the imaginary word empty.")
	aver.Always(len(value.Text) == OUTPUT_SIZE_MINIMUM, "A parsed integer leaves text empty.")
	aver.Always(len(value.Values) == bytes.SLICE_SIZE_MINIMUM,
		"A parsed integer leaves the sequence empty.")
	aver.Always(len(value.Fields) == bytes.SLICE_SIZE_MINIMUM,
		"A parsed integer leaves the members empty.")
}

// Complex_Result_Value is one parsed complex number, or the empty value a failure leaves.
type Complex_Result_Value Value

// Complex_Result_Value_Invariants fixes the payloads a complex result never owns.
func Complex_Result_Value_Invariants(value Complex_Result_Value, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value.Kind), uint8(VALUE_NIL), uint8(VALUE_COMPLEX)).
		Range_Holed_Uint64(
			uint64(value.Real), bits.WORD_64_MINIMUM,
			FLOAT_BITS_PARSED_MAXIMUM, FLOAT_BITS_SUBNORMAL_HOLE,
			FLOAT_BITS_SUBNORMAL_NEXT, FLOAT_BITS_SUBNORMAL_NEXT).
		Range_Holed_Uint64(
			uint64(value.Imaginary), bits.WORD_64_MINIMUM,
			FLOAT_BITS_PARSED_MAXIMUM, FLOAT_BITS_SUBNORMAL_HOLE,
			FLOAT_BITS_SUBNORMAL_NEXT, FLOAT_BITS_SUBNORMAL_NEXT).
		Ensure()
	aver.Always(len(value.Text) == OUTPUT_SIZE_MINIMUM, "A parsed complex leaves text empty.")
	aver.Always(len(value.Values) == bytes.SLICE_SIZE_MINIMUM,
		"A parsed complex leaves the sequence empty.")
	aver.Always(len(value.Fields) == bytes.SLICE_SIZE_MINIMUM,
		"A parsed complex leaves the members empty.")
}

// Number_Result_Value is one parsed number, or the empty value a failure leaves.
type Number_Result_Value Value

// Number_Result_Value_Invariants fixes every numeric kind a literal can carry.
func Number_Result_Value_Invariants(value Number_Result_Value, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_4_Uint8(uint8(value.Kind), uint8(VALUE_NIL), uint8(VALUE_INTEGER),
			uint8(VALUE_FLOAT), uint8(VALUE_COMPLEX)).
		Range_Uint64(uint64(value.Real), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Holed_Uint64(
			uint64(value.Imaginary), bits.WORD_64_MINIMUM,
			FLOAT_BITS_PARSED_MAXIMUM, FLOAT_BITS_SUBNORMAL_HOLE,
			FLOAT_BITS_SUBNORMAL_NEXT, FLOAT_BITS_SUBNORMAL_NEXT).
		Ensure()
	aver.Always(len(value.Text) == OUTPUT_SIZE_MINIMUM, "A parsed number leaves text empty.")
	aver.Always(len(value.Values) == bytes.SLICE_SIZE_MINIMUM,
		"A parsed number leaves the sequence empty.")
	aver.Always(len(value.Fields) == bytes.SLICE_SIZE_MINIMUM,
		"A parsed number leaves the members empty.")
}

// Literal_Text_Value is one decoded source literal, or the empty value a failure leaves.
type Literal_Text_Value Value

// Literal_Text_Value_Invariants bounds decoded text by the largest quoted source span.
func Literal_Text_Value_Invariants(value Literal_Text_Value, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value.Kind), uint8(VALUE_NIL), uint8(VALUE_TEXT)).
		Range_Int(len(value.Text), OUTPUT_SIZE_MINIMUM, LITERAL_TEXT_SIZE_MAXIMUM).
		Ensure()
	aver.Always(uint64(value.Real) == bits.WORD_64_MINIMUM,
		"A decoded literal leaves the primary word empty.")
	aver.Always(uint64(value.Imaginary) == bits.WORD_64_MINIMUM,
		"A decoded literal leaves the imaginary word empty.")
	aver.Always(len(value.Values) == bytes.SLICE_SIZE_MINIMUM,
		"A decoded literal leaves the sequence empty.")
	aver.Always(len(value.Fields) == bytes.SLICE_SIZE_MINIMUM,
		"A decoded literal leaves the members empty.")
}

// Found_Function_Value is one registered callable, or the empty value an absent name leaves.
type Found_Function_Value Value

// Found_Function_Value_Invariants fixes every payload a callable never owns.
func Found_Function_Value_Invariants(value Found_Function_Value, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value.Kind), uint8(VALUE_NIL), uint8(VALUE_FUNCTION)).
		Ensure()
	aver.Always(uint64(value.Real) == bits.WORD_64_MINIMUM,
		"A looked-up callable leaves the primary word empty.")
	aver.Always(uint64(value.Imaginary) == bits.WORD_64_MINIMUM,
		"A looked-up callable leaves the imaginary word empty.")
	aver.Always(
		len(value.Text) == OUTPUT_SIZE_MINIMUM, "A looked-up callable leaves text empty.",
	)
	aver.Always(len(value.Values) == bytes.SLICE_SIZE_MINIMUM,
		"A looked-up callable leaves the sequence empty.")
	aver.Always(len(value.Fields) == bytes.SLICE_SIZE_MINIMUM,
		"A looked-up callable leaves the members empty.")
}

// Sliced_Value is one sliced text or sequence, or the empty value a failure leaves.
type Sliced_Value Value

// Sliced_Value_Invariants fixes the two sliceable kinds and their unowned payloads.
func Sliced_Value_Invariants(value Sliced_Value, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(uint8(value.Kind), uint8(VALUE_NIL), uint8(VALUE_TEXT),
			uint8(VALUE_SEQUENCE)).
		Range_Int(len(value.Text), OUTPUT_SIZE_MINIMUM, OUTPUT_SIZE_MAXIMUM).
		Range_Int(len(value.Values), bytes.SLICE_SIZE_MINIMUM, VALUE_COUNT_MAXIMUM).
		Ensure()
	aver.Always(uint64(value.Real) == bits.WORD_64_MINIMUM,
		"A sliced value leaves the primary word empty.")
	aver.Always(uint64(value.Imaginary) == bits.WORD_64_MINIMUM,
		"A sliced value leaves the imaginary word empty.")
	aver.Always(len(value.Fields) == bytes.SLICE_SIZE_MINIMUM,
		"A sliced value leaves the members empty.")
}

func int64_order(left Integer, right Integer) (order Order) {
	defer func() { Order_Invariants(order, "int64_order.order") }()
	Integer_Invariants(left, "int64_order.left")
	Integer_Invariants(right, "int64_order.right")
	if left < right {
		return ORDER_BEFORE
	}
	if left > right {
		return ORDER_AFTER
	}
	return ORDER_SAME
}

func uint64_order(left Unsigned, right Unsigned) (order Order) {
	defer func() { Order_Invariants(order, "uint64_order.order") }()
	Unsigned_Invariants(left, "uint64_order.left")
	Unsigned_Invariants(right, "uint64_order.right")
	if left < right {
		return ORDER_BEFORE
	}
	if left > right {
		return ORDER_AFTER
	}
	return ORDER_SAME
}

// Signed range is asymmetric, so the magnitude limit differs by sign.
func integer_signed(magnitude_value Integer_Magnitude, negative Integer_Negative,
) (_ Integer_Result_Value, valid Value_Validity) {
	defer func() { Value_Validity_Invariants(valid, "integer_signed.valid") }()
	Integer_Magnitude_Invariants(magnitude_value, "integer_signed.magnitude_value")
	Integer_Negative_Invariants(negative, "integer_signed.negative")
	var result Integer_Result_Value
	defer func() { Integer_Result_Value_Invariants(result, "integer_signed.result") }()
	magnitude := uint64(magnitude_value)
	if bool(negative) {
		if magnitude > INTEGER_64_NEGATIVE_MAGNITUDE_MAXIMUM {
			return Integer_Result_Value{}, false
		}
		if magnitude == INTEGER_64_NEGATIVE_MAGNITUDE_MAXIMUM {
			result = Integer_Result_Value(
				Value_Of_Integer(Integer(bits.INTEGER_64_MINIMUM)))
			return result, true
		}
		result = Integer_Result_Value(Value_Of_Integer(Integer(-int64(magnitude))))
		return result, true
	}
	if magnitude > uint64(bits.INTEGER_MAXIMUM) {
		return Integer_Result_Value{}, false
	}
	result = Integer_Result_Value(Value_Of_Integer(Integer(magnitude)))
	return result, true
}

func float64_sort_order(left_value Float, right_value Float) (order Order) {
	defer func() { Order_Invariants(order, "float64_sort_order.order") }()
	Float_Invariants(left_value, "float64_sort_order.left_value")
	Float_Invariants(right_value, "float64_sort_order.right_value")
	left := Float(left_value)
	right := Float(right_value)
	left_nan := float64_nan(left)
	right_nan := float64_nan(right)
	if left_nan {
		if right_nan {
			return ORDER_SAME
		}
		return ORDER_BEFORE
	}
	if right_nan {
		return ORDER_AFTER
	}
	order, comparable := float64_order(left, right)
	aver.Always(comparable, "Non-NaN binary64 values admit ordering.")
	return order
}

// TERM_SOURCE_SIZE_MINIMUM is the shortest source that holds one action term.
const TERM_SOURCE_SIZE_MINIMUM = ACTION_BOUNDARY_SIZE_MINIMUM + IDENTIFIER_SIZE_MINIMUM

// Term_Source is one source that holds one action term.
type Term_Source Source

// Term_Source_Invariants composes the shortest source one action term needs.
func Term_Source_Invariants(value Term_Source, namespace aver.Namespace) {
	Term_Source_Data_Stored_Invariants(Term_Source_Data_Stored(value.Data), namespace)
}

// Term_Source_Data_Stored separates its source shape from the borrowed header.
type Term_Source_Data_Stored Source_Storage

// Term_Source_Data_Stored_Invariants bounds data by one action term.
func Term_Source_Data_Stored_Invariants(value Term_Source_Data_Stored, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value.Value), TERM_SOURCE_SIZE_MINIMUM, SOURCE_SIZE_MAXIMUM).
		Ensure()
}

// KEYWORD_SOURCE_SIZE_MINIMUM is the shortest source that holds the shortest keyword action.
const KEYWORD_SOURCE_SIZE_MINIMUM = ACTION_BOUNDARY_SIZE_MINIMUM + len(KEYWORD_IF)

// Keyword_Source is one source that holds the shortest keyword action.
type Keyword_Source Source

// Keyword_Source_Invariants composes the shortest source the shortest keyword action needs.
func Keyword_Source_Invariants(value Keyword_Source, namespace aver.Namespace) {
	Keyword_Source_Data_Stored_Invariants(Keyword_Source_Data_Stored(value.Data), namespace)
}

// Keyword_Source_Data_Stored separates its source shape from the borrowed header.
type Keyword_Source_Data_Stored Source_Storage

// Keyword_Source_Data_Stored_Invariants bounds data by the shortest keyword action.
func Keyword_Source_Data_Stored_Invariants(value Keyword_Source_Data_Stored,
	namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value.Value), KEYWORD_SOURCE_SIZE_MINIMUM, SOURCE_SIZE_MAXIMUM).
		Ensure()
}

// WORD_SOURCE_SIZE_MINIMUM is the shortest source that holds one closing keyword action.
const WORD_SOURCE_SIZE_MINIMUM = ACTION_BOUNDARY_SIZE_MINIMUM + len(KEYWORD_END)

// Word_Source is one source that holds one closing keyword action.
type Word_Source Source

// Word_Source_Invariants composes the shortest source one closing keyword action needs.
func Word_Source_Invariants(value Word_Source, namespace aver.Namespace) {
	Word_Source_Data_Stored_Invariants(Word_Source_Data_Stored(value.Data), namespace)
}

// Word_Source_Data_Stored separates its source shape from the borrowed header.
type Word_Source_Data_Stored Source_Storage

// Word_Source_Data_Stored_Invariants bounds data by one closing keyword action.
func Word_Source_Data_Stored_Invariants(value Word_Source_Data_Stored, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value.Value), WORD_SOURCE_SIZE_MINIMUM, SOURCE_SIZE_MAXIMUM).
		Ensure()
}

// COMMENT_SOURCE_SIZE_MINIMUM is the shortest source that holds one comment action.
const COMMENT_SOURCE_SIZE_MINIMUM = ACTION_BOUNDARY_SIZE_MINIMUM + len(`/**/`)

// Comment_Source is one source that holds one comment action.
type Comment_Source Source

// Comment_Source_Invariants composes the shortest source one comment action needs.
func Comment_Source_Invariants(value Comment_Source, namespace aver.Namespace) {
	Comment_Source_Data_Stored_Invariants(Comment_Source_Data_Stored(value.Data), namespace)
}

// Comment_Source_Data_Stored separates its source shape from the borrowed header.
type Comment_Source_Data_Stored Source_Storage

// Comment_Source_Data_Stored_Invariants bounds data by one comment action.
func Comment_Source_Data_Stored_Invariants(value Comment_Source_Data_Stored,
	namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value.Value), COMMENT_SOURCE_SIZE_MINIMUM, SOURCE_SIZE_MAXIMUM).
		Ensure()
}

// CLAUSE_SOURCE_SIZE_MINIMUM is the shortest source that holds one iteration keyword action.
const CLAUSE_SOURCE_SIZE_MINIMUM = ACTION_BOUNDARY_SIZE_MINIMUM + len(KEYWORD_RANGE)

// Clause_Source is one source that holds one iteration keyword action.
type Clause_Source Source

// Clause_Source_Invariants composes the shortest source one iteration keyword action needs.
func Clause_Source_Invariants(value Clause_Source, namespace aver.Namespace) {
	Clause_Source_Data_Stored_Invariants(Clause_Source_Data_Stored(value.Data), namespace)
}

// Clause_Source_Data_Stored separates its source shape from the borrowed header.
type Clause_Source_Data_Stored Source_Storage

// Clause_Source_Data_Stored_Invariants bounds data by one iteration keyword action.
func Clause_Source_Data_Stored_Invariants(value Clause_Source_Data_Stored,
	namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value.Value), CLAUSE_SOURCE_SIZE_MINIMUM, SOURCE_SIZE_MAXIMUM).
		Ensure()
}

// INVOCATION_SOURCE_SIZE_MINIMUM is the shortest source holding one invocation keyword.
const INVOCATION_SOURCE_SIZE_MINIMUM = ACTION_BOUNDARY_SIZE_MINIMUM + len(KEYWORD_TEMPLATE)

// Invocation_Source is one source that holds one invocation keyword action.
type Invocation_Source Source

// Invocation_Source_Invariants composes the shortest source one invocation needs.
func Invocation_Source_Invariants(value Invocation_Source, namespace aver.Namespace) {
	Invocation_Source_Data_Stored_Invariants(Invocation_Source_Data_Stored(value.Data),
		namespace)
}

// Invocation_Source_Data_Stored separates its source shape from the borrowed header.
type Invocation_Source_Data_Stored Source_Storage

// Invocation_Source_Data_Stored_Invariants bounds data by one invocation keyword.
func Invocation_Source_Data_Stored_Invariants(value Invocation_Source_Data_Stored,
	namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value.Value), INVOCATION_SOURCE_SIZE_MINIMUM, SOURCE_SIZE_MAXIMUM).
		Ensure()
}

// HEADER_SOURCE_SIZE_MINIMUM is the shortest source that holds one definition header.
const HEADER_SOURCE_SIZE_MINIMUM = ACTION_BOUNDARY_SIZE_MINIMUM + len(KEYWORD_DEFINE) +
	ACTION_SEPARATOR_SIZE_MINIMUM + QUOTED_BOUNDARY_SIZE

// Header_Source is one source that holds one definition header.
type Header_Source Source

// Header_Source_Invariants composes the shortest source one definition header needs.
func Header_Source_Invariants(value Header_Source, namespace aver.Namespace) {
	Header_Source_Data_Stored_Invariants(Header_Source_Data_Stored(value.Data), namespace)
}

// Header_Source_Data_Stored separates its source shape from the borrowed header.
type Header_Source_Data_Stored Source_Storage

// Header_Source_Data_Stored_Invariants bounds data by one definition header.
func Header_Source_Data_Stored_Invariants(value Header_Source_Data_Stored,
	namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value.Value), OWED_HEADER_SOURCE_SIZE_MINIMUM, SOURCE_SIZE_MAXIMUM).
		Ensure()
}

// OWED_HEADER_SOURCE_SIZE_MINIMUM adds the end one definition header still owes.
const OWED_HEADER_SOURCE_SIZE_MINIMUM = HEADER_SOURCE_SIZE_MINIMUM +
	TEMPLATE_DEFINITION_END_SIZE_MINIMUM

// BLOCK_SOURCE_SIZE_MINIMUM is the shortest source that holds one named block and its end.
const BLOCK_SOURCE_SIZE_MINIMUM = ACTION_BOUNDARY_SIZE_MINIMUM + len(KEYWORD_BLOCK) +
	ACTION_SEPARATOR_SIZE_MINIMUM + TEMPLATE_NAME_SOURCE_SIZE_MINIMUM +
	TEMPLATE_DEFINITION_END_SIZE_MINIMUM

// Block_Source is one source that holds one named block and the end it owes.
type Block_Source Source

// Block_Source_Invariants composes the shortest source one block needs.
func Block_Source_Invariants(value Block_Source, namespace aver.Namespace) {
	Block_Source_Data_Stored_Invariants(Block_Source_Data_Stored(value.Data), namespace)
}

// Block_Source_Data_Stored separates its source shape from the borrowed header.
type Block_Source_Data_Stored Source_Storage

// Block_Source_Data_Stored_Invariants bounds data by one block and its end.
func Block_Source_Data_Stored_Invariants(value Block_Source_Data_Stored,
	namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value.Value), BLOCK_SOURCE_SIZE_MINIMUM, SOURCE_SIZE_MAXIMUM).
		Ensure()
}

// BRANCH_SOURCE_SIZE_MINIMUM is the shortest source that holds one branch and its alternate.
const BRANCH_SOURCE_SIZE_MINIMUM = KEYWORD_SOURCE_SIZE_MINIMUM + ACTION_SEPARATOR_SIZE_MINIMUM +
	DOT_SOURCE_SIZE + ACTION_BOUNDARY_SIZE_MINIMUM + len(KEYWORD_ELSE)

// Branch_Source is one source that holds one branch and its alternate.
type Branch_Source Source

// Branch_Source_Invariants composes the shortest source one branch and its alternate needs.
func Branch_Source_Invariants(value Branch_Source, namespace aver.Namespace) {
	Branch_Source_Data_Stored_Invariants(Branch_Source_Data_Stored(value.Data), namespace)
}

// Branch_Source_Data_Stored separates its source shape from the borrowed header.
type Branch_Source_Data_Stored Source_Storage

// Branch_Source_Data_Stored_Invariants bounds data by one branch and its alternate.
func Branch_Source_Data_Stored_Invariants(value Branch_Source_Data_Stored,
	namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value.Value), BRANCH_SOURCE_SIZE_MINIMUM, SOURCE_SIZE_MAXIMUM).
		Ensure()
}

// DEFINITION_SOURCE_SIZE_MINIMUM is the shortest source that holds one complete definition.
const DEFINITION_SOURCE_SIZE_MINIMUM = HEADER_SOURCE_SIZE_MINIMUM + WORD_SOURCE_SIZE_MINIMUM

// Definition_Source is one source that holds one complete definition.
type Definition_Source Source

// Definition_Source_Invariants composes the shortest source one complete definition needs.
func Definition_Source_Invariants(value Definition_Source, namespace aver.Namespace) {
	Definition_Source_Data_Stored_Invariants(Definition_Source_Data_Stored(value.Data),
		namespace)
}

// Definition_Source_Data_Stored separates its source shape from the borrowed header.
type Definition_Source_Data_Stored Source_Storage

// Definition_Source_Data_Stored_Invariants bounds data by one complete definition.
func Definition_Source_Data_Stored_Invariants(value Definition_Source_Data_Stored,
	namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value.Value), DEFINITION_SOURCE_SIZE_MINIMUM, SOURCE_SIZE_MAXIMUM).
		Ensure()
}

// MERGE_SOURCE_SIZE_MINIMUM is the shortest source that holds two complete definitions.
const MERGE_SOURCE_SIZE_MINIMUM = DEFINITION_SOURCE_SIZE_MINIMUM + DEFINITION_SOURCE_SIZE_MINIMUM

// Merge_Source is one source that holds two complete definitions.
type Merge_Source Source

// Merge_Source_Invariants composes the shortest source two complete definitions needs.
func Merge_Source_Invariants(value Merge_Source, namespace aver.Namespace) {
	Merge_Source_Data_Stored_Invariants(Merge_Source_Data_Stored(value.Data), namespace)
}

// Merge_Source_Data_Stored separates its source shape from the borrowed header.
type Merge_Source_Data_Stored Source_Storage

// Merge_Source_Data_Stored_Invariants bounds data by two complete definitions.
func Merge_Source_Data_Stored_Invariants(value Merge_Source_Data_Stored, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value.Value), MERGE_SOURCE_SIZE_MINIMUM, SOURCE_SIZE_MAXIMUM).
		Ensure()
}

// ACTION_PROGRAM_NODE_COUNT_MINIMUM counts the root and the shortest action's nodes.
const ACTION_PROGRAM_NODE_COUNT_MINIMUM = ROOT_NODE_COUNT + SIMPLE_ACTION_NODE_COUNT

// EXECUTED_TEMPLATE_COUNT_MAXIMUM names every body one executable source can define.
const EXECUTED_TEMPLATE_COUNT_MAXIMUM = (SOURCE_SIZE_MAXIMUM - SIMPLE_ACTION_SIZE_MINIMUM) /
	(TEMPLATE_DEFINITION_SIZE_MINIMUM + IDENTIFIER_SIZE_MINIMUM)

// RANGED_PROGRAM_NODE_COUNT_MINIMUM counts the root beside one range body.
const RANGED_PROGRAM_NODE_COUNT_MINIMUM = ROOT_NODE_COUNT + CONTROL_BRANCH_NODE_COUNT

// RANGED_PROGRAM_NODE_COUNT_MAXIMUM packs actions beside one range body.
const RANGED_PROGRAM_NODE_COUNT_MAXIMUM = ROOT_NODE_COUNT + CONTROL_BRANCH_NODE_COUNT +
	NODE_COUNT_INCREMENT +
	(SOURCE_SIZE_MAXIMUM-RANGE_PROGRAM_SOURCE_SIZE_MINIMUM-utf8.CHARACTER_SIZE_MINIMUM)/
		SIMPLE_ACTION_SIZE_MINIMUM*SIMPLE_ACTION_NODE_COUNT +
	TRAILING_TEXT_NODE_COUNT_MAXIMUM

// Ranged_Program_Document is arena metadata of a program that holds one range body.
type Ranged_Program_Document Program_Validated_Document

// Ranged_Program_Document_Invariants bounds counts by one range body.
func Ranged_Program_Document_Invariants(
	value Ranged_Program_Document, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Node_Count),
			uint16(RANGED_PROGRAM_NODE_COUNT_MINIMUM),
			uint16(RANGED_PROGRAM_NODE_COUNT_MAXIMUM)).
		Range_Uint16(
			uint16(value.First_Template), uint16(DOCUMENT_FIRST_TEMPLATE_MINIMUM),
			uint16(RANGE_BEGIN_FIRST_TEMPLATE_MAXIMUM)).
		Range_Uint16(uint16(value.Template_Count), uint16(TEMPLATE_COUNT_MINIMUM),
			uint16(EXECUTED_TEMPLATE_COUNT_MAXIMUM)).
		Ensure()
	aver.Always(uint16(value.Root) == uint16(ROOT_NODE_COUNT),
		"One range program keeps the root in the first syntax arena slot.")
}

// SIMPLE_PROGRAM_NODE_COUNT_MINIMUM counts the root beside one evaluated term.
const SIMPLE_PROGRAM_NODE_COUNT_MINIMUM = ACTION_PROGRAM_NODE_COUNT_MINIMUM

// SIMPLE_PROGRAM_NODE_COUNT_MAXIMUM fills the arena the evaluated term stands in.
const SIMPLE_PROGRAM_NODE_COUNT_MAXIMUM = NODE_COUNT_MAXIMUM

// Simple_Program_Document is arena metadata of a program that holds one evaluated term.
type Simple_Program_Document Program_Validated_Document

// Simple_Program_Document_Invariants bounds counts by one evaluated term.
func Simple_Program_Document_Invariants(
	value Simple_Program_Document, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Node_Count),
			uint16(SIMPLE_PROGRAM_NODE_COUNT_MINIMUM),
			uint16(SIMPLE_PROGRAM_NODE_COUNT_MAXIMUM)).
		Range_Uint16(
			uint16(value.First_Template), uint16(DOCUMENT_FIRST_TEMPLATE_MINIMUM),
			uint16(DOCUMENT_FIRST_TEMPLATE_MAXIMUM)).
		Range_Uint16(uint16(value.Template_Count), uint16(TEMPLATE_COUNT_MINIMUM),
			uint16(EXECUTED_TEMPLATE_COUNT_MAXIMUM)).
		Ensure()
	aver.Always(uint16(value.Root) == uint16(ROOT_NODE_COUNT),
		"One evaluated program keeps the root in the first syntax arena slot.")
}

// CHAINED_PROGRAM_NODE_COUNT_MINIMUM counts the root beside one chained term.
const CHAINED_PROGRAM_NODE_COUNT_MINIMUM = ACTION_PROGRAM_NODE_COUNT_MINIMUM +
	SCALAR_NODE_COUNT

// CHAINED_PROGRAM_NODE_COUNT_MAXIMUM packs actions beside one chained term.
const CHAINED_PROGRAM_NODE_COUNT_MAXIMUM = NODE_COUNT_MAXIMUM - COMMAND_NODE_COUNT -
	SCALAR_NODE_COUNT

// CHAINED_FIRST_TEMPLATE_MAXIMUM follows the shortest chained block, which packs one node
// fewer than the shortest block.
const CHAINED_FIRST_TEMPLATE_MAXIMUM = DOCUMENT_FIRST_TEMPLATE_MAXIMUM - NODE_COUNT_INCREMENT

// Chained_Program_Document is arena metadata of a program that holds one chained term.
type Chained_Program_Document Program_Validated_Document

// Chained_Program_Document_Invariants bounds counts by one chained term.
func Chained_Program_Document_Invariants(
	value Chained_Program_Document, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Node_Count),
			uint16(CHAINED_PROGRAM_NODE_COUNT_MINIMUM),
			uint16(CHAINED_PROGRAM_NODE_COUNT_MAXIMUM)).
		Range_Uint16(
			uint16(value.First_Template), uint16(DOCUMENT_FIRST_TEMPLATE_MINIMUM),
			uint16(CHAINED_FIRST_TEMPLATE_MAXIMUM)).
		Range_Uint16(uint16(value.Template_Count), uint16(TEMPLATE_COUNT_MINIMUM),
			uint16(EXECUTED_TEMPLATE_COUNT_MAXIMUM)).
		Ensure()
	aver.Always(uint16(value.Root) == uint16(ROOT_NODE_COUNT),
		"One chained program keeps the root in the first syntax arena slot.")
}

// BOUND_PROGRAM_NODE_COUNT_MINIMUM counts the root beside one variable binding.
const BOUND_PROGRAM_NODE_COUNT_MINIMUM = ACTION_PROGRAM_NODE_COUNT_MINIMUM +
	SCALAR_NODE_COUNT

// BOUND_PROGRAM_NODE_COUNT_MAXIMUM packs actions beside one variable binding whose value is one
// parenthesized term, which pays two bytes for its pipeline and command.
const BOUND_PROGRAM_NODE_COUNT_MAXIMUM = DELIVERED_DECLARATION_PACKED_COUNT +
	SIMPLE_ACTION_NODE_COUNT + NODE_COUNT_INCREMENT + PIPELINE_NODE_COUNT + COMMAND_NODE_COUNT

// Bound_Program_Document is arena metadata of a program that holds one variable binding.
type Bound_Program_Document Program_Validated_Document

// Bound_Program_Document_Invariants bounds counts by one variable binding.
func Bound_Program_Document_Invariants(
	value Bound_Program_Document, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Node_Count),
			uint16(BOUND_PROGRAM_NODE_COUNT_MINIMUM),
			uint16(BOUND_PROGRAM_NODE_COUNT_MAXIMUM)).
		Range_Uint16(
			uint16(value.First_Template), uint16(DOCUMENT_FIRST_TEMPLATE_MINIMUM),
			uint16(BOUND_FIRST_TEMPLATE_MAXIMUM)).
		Range_Uint16(uint16(value.Template_Count), uint16(TEMPLATE_COUNT_MINIMUM),
			uint16(EXECUTED_TEMPLATE_COUNT_MAXIMUM)).
		Ensure()
	aver.Always(uint16(value.Root) == uint16(ROOT_NODE_COUNT),
		"One binding program keeps the root in the first syntax arena slot.")
}

// Eof_Token is one retained token holding no retained token.
type Eof_Token Token

// Eof_Token_Invariants fixes the kind one eof cursor retains.
func Eof_Token_Invariants(value Eof_Token, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Start), uint16(TOKEN_BOUNDARY_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Range_Uint16(uint16(value.End), uint16(TOKEN_BOUNDARY_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Ensure()
	aver.Always(uint8(value.Kind) == uint8(TOKEN_EOF),
		"A eof cursor retains the token its caller read.")
}

// Eof_Token_Storage is that retained token inside its cursor field.
type Eof_Token_Storage Token_Storage

// Eof_Token_Storage_Invariants composes the retained token.
func Eof_Token_Storage_Invariants(value Eof_Token_Storage, namespace aver.Namespace) {
	Eof_Token_Invariants(Eof_Token(value.Value), namespace)
}

// Eof_Cursor is one lexer whose retained token holds no retained token.
type Eof_Cursor Token_Cursor

// Eof_Cursor_Invariants composes lexer state beside the retained token.
func Eof_Cursor_Invariants(value Eof_Cursor, namespace aver.Namespace) {
	Lexeme_Position_Invariants(Lexeme_Position(value.Position), namespace)
	Lexeme_End_Invariants(Lexeme_End(value.End), namespace)
	Lexeme_Eof_Token_Storage_Invariants(Lexeme_Eof_Token_Storage(value.Token), namespace)
	Unread_Look_Ahead_Storage_Invariants(Unread_Look_Ahead_Storage(value.Looked), namespace)
	Cursor_Number_Stored_Invariants(Cursor_Number_Stored(value.Number), namespace)
	Function_Function_Storage_Invariants(value.Function, namespace)
	Lexeme_Last_Start_Invariants(Lexeme_Last_Start(value.Last_Start), namespace)
	Cursor_Has_Last_Stored_Invariants(Cursor_Has_Last_Stored(value.Has_Last), namespace)
}

// Fresh_Cursor is one action lexer before it reads its first token.
type Fresh_Cursor Eof_Cursor

// Fresh_Cursor_Invariants composes lexer state that holds no token history.
func Fresh_Cursor_Invariants(value Fresh_Cursor, namespace aver.Namespace) {
	Cursor_Position_Stored_Invariants(Cursor_Position_Stored(value.Position), namespace)
	Cursor_End_Stored_Invariants(Cursor_End_Stored(value.End), namespace)
	Eof_Token_Storage_Invariants(Eof_Token_Storage(value.Token), namespace)
	Unread_Look_Ahead_Storage_Invariants(Unread_Look_Ahead_Storage(value.Looked), namespace)
	Cursor_Number_Stored_Invariants(Cursor_Number_Stored(value.Number), namespace)
	Function_Function_Storage_Invariants(value.Function, namespace)
	Fresh_Last_Start_Stored_Invariants(Fresh_Last_Start_Stored(value.Last_Start), namespace)
	Fresh_Has_Last_Stored_Invariants(Fresh_Has_Last_Stored(value.Has_Last), namespace)
}

// Fresh_Cursor_Pointer keeps mutation of that lexer typed.
type Fresh_Cursor_Pointer *Fresh_Cursor

// Fresh_Cursor_Pointer_Invariants composes present fresh lexer state.
func Fresh_Cursor_Pointer_Invariants(value Fresh_Cursor_Pointer, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Fresh_Cursor_Invariants(*value, namespace)
}

// Fresh_Last_Start_Stored is the diagnostic boundary one fresh cursor carries.
type Fresh_Last_Start_Stored Cursor_Last_Start_Storage

// Fresh_Last_Start_Stored_Invariants fixes the boundary before the first token.
func Fresh_Last_Start_Stored_Invariants(value Fresh_Last_Start_Stored,
	namespace aver.Namespace,
) {
	aver.Always(uint16(value.Value) == SOURCE_SIZE_MINIMUM,
		"One fresh cursor opens its diagnostic at the source start.")
}

// Read_Has_Last_Stored is the token history one cursor keeps after its first read.
type Read_Has_Last_Stored Cursor_Has_Last_Storage

// Read_Has_Last_Stored_Invariants fixes that history present.
func Read_Has_Last_Stored_Invariants(value Read_Has_Last_Stored, _ aver.Namespace) {
	aver.Always(uint8(value.Value) == uint8(utf8.CHARACTER_SIZE_MINIMUM),
		"One cursor past its first token keeps that token in its history.")
}

// Fresh_Has_Last_Stored is the token history one fresh cursor carries.
type Fresh_Has_Last_Stored Cursor_Has_Last_Storage

// Fresh_Has_Last_Stored_Invariants fixes the absent history of a fresh cursor.
func Fresh_Has_Last_Stored_Invariants(value Fresh_Has_Last_Stored, namespace aver.Namespace) {
	aver.Always(uint8(value.Value) == bits.WORD_8_MINIMUM,
		"One fresh cursor holds no token history.")
}

// Eof_Cursor_Pointer keeps mutation of that lexer typed.
type Eof_Cursor_Pointer *Eof_Cursor

// Eof_Cursor_Pointer_Invariants composes present lexer state.
func Eof_Cursor_Pointer_Invariants(value Eof_Cursor_Pointer, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Eof_Cursor_Invariants(*value, namespace)
}

// Else_Token is one retained token holding one else keyword.
type Else_Token Token

// Else_Token_Invariants fixes the kind one else cursor retains.
func Else_Token_Invariants(value Else_Token, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Start), uint16(TOKEN_BOUNDARY_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM-len(KEYWORD_ELSE))).
		Range_Uint16(uint16(value.End), uint16(ELSE_TOKEN_END_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Ensure()
	aver.Always(uint8(value.Kind) == uint8(TOKEN_ELSE),
		"A else cursor retains the token its caller read.")
}

// Else_Token_Storage is that retained token inside its cursor field.
type Else_Token_Storage Token_Storage

// Else_Token_Storage_Invariants composes the retained token.
func Else_Token_Storage_Invariants(value Else_Token_Storage, namespace aver.Namespace) {
	Else_Token_Invariants(Else_Token(value.Value), namespace)
}

// Else_Cursor is one lexer whose retained token holds one else keyword.
type Else_Cursor Token_Cursor

// Else_Cursor_Invariants composes lexer state beside the retained token.
func Else_Cursor_Invariants(value Else_Cursor, namespace aver.Namespace) {
	Else_Position_Stored_Invariants(Else_Position_Stored(value.Position), namespace)
	Else_End_Stored_Invariants(Else_End_Stored(value.End), namespace)
	Else_Token_Storage_Invariants(Else_Token_Storage(value.Token), namespace)
	Peeked_Look_Ahead_Storage_Invariants(Peeked_Look_Ahead_Storage(value.Looked), namespace)
	Cursor_Number_Stored_Invariants(Cursor_Number_Stored(value.Number), namespace)
	Function_Function_Storage_Invariants(value.Function, namespace)
	Else_Last_Start_Invariants(Else_Last_Start(value.Last_Start), namespace)
	Else_Has_Last_Invariants(Else_Has_Last(value.Has_Last), namespace)
}

// Else_Cursor_Pointer keeps mutation of that lexer typed.
type Else_Cursor_Pointer *Else_Cursor

// Else_Cursor_Pointer_Invariants composes present lexer state.
func Else_Cursor_Pointer_Invariants(value Else_Cursor_Pointer, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Else_Cursor_Invariants(*value, namespace)
}

// End_Token is one retained token holding one end keyword.
type End_Token Token

// End_Token_Invariants fixes the kind one end cursor retains.
func End_Token_Invariants(value End_Token, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value.Start), uint16(TOKEN_BOUNDARY_MINIMUM),
			uint16(LEXEME_START_MAXIMUM-ACTION_BOUNDARY_SIZE_MINIMUM)).
		Range_Uint16(uint16(value.End), uint16(END_TOKEN_END_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Ensure()
	aver.Always(uint8(value.Kind) == uint8(TOKEN_END),
		"A end cursor retains the token its caller read.")
}

// End_Token_Storage is that retained token inside its cursor field.
type End_Token_Storage Token_Storage

// End_Token_Storage_Invariants composes the retained token.
func End_Token_Storage_Invariants(value End_Token_Storage, namespace aver.Namespace) {
	End_Token_Invariants(End_Token(value.Value), namespace)
}

// End_Cursor is one lexer whose retained token holds one end keyword.
type End_Cursor Token_Cursor

// End_Cursor_Invariants composes lexer state beside the retained token.
func End_Cursor_Invariants(value End_Cursor, namespace aver.Namespace) {
	End_Position_Stored_Invariants(End_Position_Stored(value.Position), namespace)
	End_End_Stored_Invariants(End_End_Stored(value.End), namespace)
	End_Token_Storage_Invariants(End_Token_Storage(value.Token), namespace)
	Peeked_Look_Ahead_Storage_Invariants(Peeked_Look_Ahead_Storage(value.Looked), namespace)
	Cursor_Number_Stored_Invariants(Cursor_Number_Stored(value.Number), namespace)
	Function_Function_Storage_Invariants(value.Function, namespace)
	End_Clause_Last_Start_Invariants(End_Clause_Last_Start(value.Last_Start), namespace)
	End_Clause_Has_Last_Invariants(End_Clause_Has_Last(value.Has_Last), namespace)
}

// End_Cursor_Pointer keeps mutation of that lexer typed.
type End_Cursor_Pointer *End_Cursor

// End_Cursor_Pointer_Invariants composes present lexer state.
func End_Cursor_Pointer_Invariants(value End_Cursor_Pointer, namespace aver.Namespace) {
	if value == nil {
		return
	}
	End_Cursor_Invariants(*value, namespace)
}

// Comment_Token is one retained token holding one comment.
type Comment_Token Token

// Comment_Token_Invariants fixes the kind one comment cursor retains.
func Comment_Token_Invariants(value Comment_Token, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value.Start), uint16(TOKEN_BOUNDARY_MINIMUM),
			uint16(LEXEME_START_MAXIMUM-SIMPLE_ACTION_SIZE_MINIMUM)).
		Range_Uint16(uint16(value.End), uint16(COMMENT_TOKEN_END_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Ensure()
	aver.Always(uint8(value.Kind) == uint8(TOKEN_COMMENT),
		"A comment cursor retains the token its caller read.")
}

// Comment_Token_Storage is that retained token inside its cursor field.
type Comment_Token_Storage Token_Storage

// Comment_Token_Storage_Invariants composes the retained token.
func Comment_Token_Storage_Invariants(value Comment_Token_Storage, namespace aver.Namespace) {
	Comment_Token_Invariants(Comment_Token(value.Value), namespace)
}

// Comment_Cursor is one lexer whose retained token holds one comment.
type Comment_Cursor Token_Cursor

// Comment_Cursor_Invariants composes lexer state beside the retained token.
func Comment_Cursor_Invariants(value Comment_Cursor, namespace aver.Namespace) {
	Comment_Position_Stored_Invariants(Comment_Position_Stored(value.Position), namespace)
	Comment_End_Stored_Invariants(Comment_End_Stored(value.End), namespace)
	Comment_Token_Storage_Invariants(Comment_Token_Storage(value.Token), namespace)
	Peeked_Look_Ahead_Storage_Invariants(Peeked_Look_Ahead_Storage(value.Looked), namespace)
	Cursor_Number_Stored_Invariants(Cursor_Number_Stored(value.Number), namespace)
	Function_Function_Storage_Invariants(value.Function, namespace)
	Comment_Last_Start_Invariants(Comment_Last_Start(value.Last_Start), namespace)
	Comment_Has_Last_Invariants(Comment_Has_Last(value.Has_Last), namespace)
}

// Comment_Cursor_Pointer keeps mutation of that lexer typed.
type Comment_Cursor_Pointer *Comment_Cursor

// Comment_Cursor_Pointer_Invariants composes present lexer state.
func Comment_Cursor_Pointer_Invariants(value Comment_Cursor_Pointer, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Comment_Cursor_Invariants(*value, namespace)
}

// Invocation_Token is one retained token holding one template keyword.
type Invocation_Token Token

// Invocation_Token_Invariants fixes the kind one invocation cursor retains.
func Invocation_Token_Invariants(value Invocation_Token, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value.Start), uint16(TOKEN_BOUNDARY_MINIMUM),
			uint16(INVOCATION_TOKEN_START_MAXIMUM)).
		Range_Uint16(uint16(value.End), uint16(INVOCATION_TOKEN_END_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Ensure()
	aver.Always(uint8(value.Kind) == uint8(TOKEN_TEMPLATE),
		"A invocation cursor retains the token its caller read.")
}

// Invocation_Token_Storage is that retained token inside its cursor field.
type Invocation_Token_Storage Token_Storage

// Invocation_Token_Storage_Invariants composes the retained token.
func Invocation_Token_Storage_Invariants(value Invocation_Token_Storage, namespace aver.Namespace) {
	Invocation_Token_Invariants(Invocation_Token(value.Value), namespace)
}

// Invocation_Cursor is one lexer whose retained token holds one template keyword.
type Invocation_Cursor Token_Cursor

// Invocation_Cursor_Invariants composes lexer state beside the retained token.
func Invocation_Cursor_Invariants(value Invocation_Cursor, namespace aver.Namespace) {
	Invocation_Position_Stored_Invariants(
		Invocation_Position_Stored(value.Position), namespace)
	Invocation_End_Stored_Invariants(Invocation_End_Stored(value.End), namespace)
	Invocation_Token_Storage_Invariants(Invocation_Token_Storage(value.Token), namespace)
	Peeked_Look_Ahead_Storage_Invariants(Peeked_Look_Ahead_Storage(value.Looked), namespace)
	Cursor_Number_Stored_Invariants(Cursor_Number_Stored(value.Number), namespace)
	Function_Function_Storage_Invariants(value.Function, namespace)
	Invocation_Last_Start_Invariants(Invocation_Last_Start(value.Last_Start), namespace)
	Invocation_Has_Last_Invariants(Invocation_Has_Last(value.Has_Last), namespace)
}

// Invocation_Cursor_Pointer keeps mutation of that lexer typed.
type Invocation_Cursor_Pointer *Invocation_Cursor

// Invocation_Cursor_Pointer_Invariants composes present lexer state.
func Invocation_Cursor_Pointer_Invariants(value Invocation_Cursor_Pointer,
	namespace aver.Namespace) {
	if value == nil {
		return
	}
	Invocation_Cursor_Invariants(*value, namespace)
}

// Block_Token is one retained token holding one quoted definition name.
type Block_Token Token

// Block_Token_Invariants fixes the kind one block cursor retains.
func Block_Token_Invariants(value Block_Token, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value.Kind), uint8(TOKEN_RAW_STRING), uint8(TOKEN_STRING)).
		Range_Uint16(uint16(value.Start), uint16(INVOCATION_NAME_START_MINIMUM),
			uint16(BLOCK_LAST_START_MAXIMUM)).
		Range_Uint16(uint16(value.End), uint16(BLOCK_END_MINIMUM),
			uint16(OWED_END_BOUNDARY_MAXIMUM)).
		Ensure()
}

// Block_Token_Storage is that retained token inside its cursor field.
type Block_Token_Storage Token_Storage

// Block_Token_Storage_Invariants composes the retained token.
func Block_Token_Storage_Invariants(value Block_Token_Storage, namespace aver.Namespace) {
	Block_Token_Invariants(Block_Token(value.Value), namespace)
}

// Block_Cursor is one lexer whose retained token holds one quoted definition name.
type Block_Cursor Token_Cursor

// Block_Cursor_Invariants composes lexer state beside the retained token.
func Block_Cursor_Invariants(value Block_Cursor, namespace aver.Namespace) {
	Block_Position_Stored_Invariants(Block_Position_Stored(value.Position), namespace)
	Block_End_Stored_Invariants(Block_End_Stored(value.End), namespace)
	Block_Token_Storage_Invariants(Block_Token_Storage(value.Token), namespace)
	Unread_Look_Ahead_Storage_Invariants(Unread_Look_Ahead_Storage(value.Looked), namespace)
	Cursor_Number_Stored_Invariants(Cursor_Number_Stored(value.Number), namespace)
	Function_Function_Storage_Invariants(value.Function, namespace)
	Block_Last_Start_Invariants(Block_Last_Start(value.Last_Start), namespace)
	Read_Has_Last_Stored_Invariants(Read_Has_Last_Stored(value.Has_Last), namespace)
}

// Block_Cursor_Pointer keeps mutation of that lexer typed.
type Block_Cursor_Pointer *Block_Cursor

// Block_Cursor_Pointer_Invariants composes present lexer state.
func Block_Cursor_Pointer_Invariants(value Block_Cursor_Pointer, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Block_Cursor_Invariants(*value, namespace)
}

// Open_Paren_Token is one retained token holding one opening parenthesis.
type Open_Paren_Token Token

// Open_Paren_Token_Invariants fixes the kind one open paren cursor retains.
func Open_Paren_Token_Invariants(value Open_Paren_Token, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value.Start), uint16(TOKEN_BOUNDARY_MINIMUM),
			uint16(LEXEME_START_MAXIMUM)).
		Range_Uint16(
			uint16(value.End),
			uint16(TOKEN_BOUNDARY_MINIMUM+utf8.CHARACTER_SIZE_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Ensure()
	aver.Always(uint8(value.Kind) == uint8(TOKEN_LEFT_PAREN),
		"A open paren cursor retains the token its caller read.")
}

// Open_Paren_Token_Storage is that retained token inside its cursor field.
type Open_Paren_Token_Storage Token_Storage

// Open_Paren_Token_Storage_Invariants composes the retained token.
func Open_Paren_Token_Storage_Invariants(value Open_Paren_Token_Storage, namespace aver.Namespace) {
	Open_Paren_Token_Invariants(Open_Paren_Token(value.Value), namespace)
}

// Open_Paren_Cursor is one lexer whose retained token holds one opening parenthesis.
type Open_Paren_Cursor Token_Cursor

// Open_Paren_Cursor_Invariants composes lexer state beside the retained token.
func Open_Paren_Cursor_Invariants(value Open_Paren_Cursor, namespace aver.Namespace) {
	Open_Paren_Position_Stored_Invariants(
		Open_Paren_Position_Stored(value.Position), namespace)
	Open_Paren_End_Stored_Invariants(Open_Paren_End_Stored(value.End), namespace)
	Open_Paren_Token_Storage_Invariants(Open_Paren_Token_Storage(value.Token), namespace)
	Peeked_Look_Ahead_Storage_Invariants(Peeked_Look_Ahead_Storage(value.Looked), namespace)
	Cursor_Number_Stored_Invariants(Cursor_Number_Stored(value.Number), namespace)
	Function_Function_Storage_Invariants(value.Function, namespace)
	Open_Paren_Last_Start_Invariants(Open_Paren_Last_Start(value.Last_Start), namespace)
	Cursor_Has_Last_Stored_Invariants(Cursor_Has_Last_Stored(value.Has_Last), namespace)
}

// Open_Paren_Cursor_Pointer keeps mutation of that lexer typed.
type Open_Paren_Cursor_Pointer *Open_Paren_Cursor

// Open_Paren_Cursor_Pointer_Invariants composes present lexer state.
func Open_Paren_Cursor_Pointer_Invariants(value Open_Paren_Cursor_Pointer,
	namespace aver.Namespace) {
	if value == nil {
		return
	}
	Open_Paren_Cursor_Invariants(*value, namespace)
}

// Close_Paren_Token is one retained token holding one closing parenthesis.
type Close_Paren_Token Token

// Close_Paren_Token_Invariants fixes the kind one close paren cursor retains.
func Close_Paren_Token_Invariants(value Close_Paren_Token, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Start),
			uint16(CLOSE_PAREN_TOKEN_END_MINIMUM-PARENTHESIS_SIZE),
			uint16(LEXEME_START_MAXIMUM)).
		Range_Uint16(uint16(value.End), uint16(CLOSE_PAREN_TOKEN_END_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Ensure()
	aver.Always(uint8(value.Kind) == uint8(TOKEN_RIGHT_PAREN),
		"A close paren cursor retains the token its caller read.")
}

// Close_Paren_Token_Storage is that retained token inside its cursor field.
type Close_Paren_Token_Storage Token_Storage

// Close_Paren_Token_Storage_Invariants composes the retained token.
func Close_Paren_Token_Storage_Invariants(value Close_Paren_Token_Storage,
	namespace aver.Namespace) {
	Close_Paren_Token_Invariants(Close_Paren_Token(value.Value), namespace)
}

// Close_Paren_Cursor is one lexer whose retained token holds one closing parenthesis.
type Close_Paren_Cursor Token_Cursor

// Close_Paren_Cursor_Invariants composes lexer state beside the retained token.
func Close_Paren_Cursor_Invariants(value Close_Paren_Cursor, namespace aver.Namespace) {
	Close_Paren_Position_Stored_Invariants(
		Close_Paren_Position_Stored(value.Position), namespace)
	Close_Paren_End_Stored_Invariants(Close_Paren_End_Stored(value.End), namespace)
	Close_Paren_Token_Storage_Invariants(Close_Paren_Token_Storage(value.Token), namespace)
	Peeked_Look_Ahead_Storage_Invariants(Peeked_Look_Ahead_Storage(value.Looked), namespace)
	Cursor_Number_Stored_Invariants(Cursor_Number_Stored(value.Number), namespace)
	Function_Function_Storage_Invariants(value.Function, namespace)
	Close_Paren_Last_Start_Invariants(Close_Paren_Last_Start(value.Last_Start), namespace)
	Read_Has_Last_Stored_Invariants(Read_Has_Last_Stored(value.Has_Last), namespace)
}

// Close_Paren_Cursor_Pointer keeps mutation of that lexer typed.
type Close_Paren_Cursor_Pointer *Close_Paren_Cursor

// Close_Paren_Cursor_Pointer_Invariants composes present lexer state.
func Close_Paren_Cursor_Pointer_Invariants(value Close_Paren_Cursor_Pointer,
	namespace aver.Namespace) {
	if value == nil {
		return
	}
	Close_Paren_Cursor_Invariants(*value, namespace)
}

// Separator_Token is one retained token holding one pipe separator.
type Separator_Token Token

// Separator_Token_Invariants fixes the kind one separator cursor retains.
func Separator_Token_Invariants(value Separator_Token, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value.Start),
			uint16(TOKEN_BOUNDARY_MINIMUM+utf8.CHARACTER_SIZE_MINIMUM),
			uint16(LEXEME_START_MAXIMUM-utf8.CHARACTER_SIZE_MINIMUM)).
		Range_Uint16(
			uint16(value.End), uint16(SEPARATOR_TOKEN_END_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM-utf8.CHARACTER_SIZE_MINIMUM)).
		Ensure()
	aver.Always(uint8(value.Kind) == uint8(TOKEN_PIPE),
		"A separator cursor retains the token its caller read.")
}

// Separator_Token_Storage is that retained token inside its cursor field.
type Separator_Token_Storage Token_Storage

// Separator_Token_Storage_Invariants composes the retained token.
func Separator_Token_Storage_Invariants(value Separator_Token_Storage, namespace aver.Namespace) {
	Separator_Token_Invariants(Separator_Token(value.Value), namespace)
}

// Separator_Cursor is one lexer whose retained token holds one pipe separator.
type Separator_Cursor Token_Cursor

// Separator_Cursor_Invariants composes lexer state beside the retained token.
func Separator_Cursor_Invariants(value Separator_Cursor, namespace aver.Namespace) {
	Separator_Position_Stored_Invariants(Separator_Position_Stored(value.Position), namespace)
	Separator_End_Stored_Invariants(Separator_End_Stored(value.End), namespace)
	Separator_Token_Storage_Invariants(Separator_Token_Storage(value.Token), namespace)
	Peeked_Look_Ahead_Storage_Invariants(Peeked_Look_Ahead_Storage(value.Looked), namespace)
	Cursor_Number_Stored_Invariants(Cursor_Number_Stored(value.Number), namespace)
	Function_Function_Storage_Invariants(value.Function, namespace)
	Separated_Last_Start_Invariants(Separated_Last_Start(value.Last_Start), namespace)
	Cursor_Has_Last_Stored_Invariants(Cursor_Has_Last_Stored(value.Has_Last), namespace)
}

// Separator_Cursor_Pointer keeps mutation of that lexer typed.
type Separator_Cursor_Pointer *Separator_Cursor

// Separator_Cursor_Pointer_Invariants composes present lexer state.
func Separator_Cursor_Pointer_Invariants(value Separator_Cursor_Pointer, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Separator_Cursor_Invariants(*value, namespace)
}

// Variable_Token is one retained token holding one variable name.
type Variable_Token Token

// Variable_Token_Invariants fixes the kind one variable cursor retains.
func Variable_Token_Invariants(value Variable_Token, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Start), uint16(TOKEN_BOUNDARY_MINIMUM),
			uint16(LEXEME_START_MAXIMUM)).
		Range_Uint16(uint16(value.End),
			uint16(TOKEN_BOUNDARY_MINIMUM+VARIABLE_NAME_SIZE_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Ensure()
	aver.Always(uint8(value.Kind) == uint8(TOKEN_VARIABLE),
		"A variable cursor retains the token its caller read.")
}

// Variable_Token_Storage is that retained token inside its cursor field.
type Variable_Token_Storage Token_Storage

// Variable_Token_Storage_Invariants composes the retained token.
func Variable_Token_Storage_Invariants(value Variable_Token_Storage, namespace aver.Namespace) {
	Variable_Token_Invariants(Variable_Token(value.Value), namespace)
}

// Variable_Cursor is one lexer whose retained token holds one variable name.
type Variable_Cursor Token_Cursor

// Variable_Cursor_Invariants composes lexer state beside the retained token.
func Variable_Cursor_Invariants(value Variable_Cursor, namespace aver.Namespace) {
	Read_Cursor_Position_Invariants(Read_Cursor_Position(value.Position), namespace)
	Read_Cursor_End_Invariants(Read_Cursor_End(value.End), namespace)
	Variable_Token_Storage_Invariants(Variable_Token_Storage(value.Token), namespace)
	Peeked_Look_Ahead_Storage_Invariants(Peeked_Look_Ahead_Storage(value.Looked), namespace)
	Cursor_Number_Stored_Invariants(Cursor_Number_Stored(value.Number), namespace)
	Function_Function_Storage_Invariants(value.Function, namespace)
	Variable_Last_Start_Invariants(Variable_Last_Start(value.Last_Start), namespace)
	Cursor_Has_Last_Stored_Invariants(Cursor_Has_Last_Stored(value.Has_Last), namespace)
}

// Variable_Cursor_Pointer keeps mutation of that lexer typed.
type Variable_Cursor_Pointer *Variable_Cursor

// Variable_Cursor_Pointer_Invariants composes present lexer state.
func Variable_Cursor_Pointer_Invariants(value Variable_Cursor_Pointer, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Variable_Cursor_Invariants(*value, namespace)
}

// PIPELINE_LAST_START_MAXIMUM is the latest shortest keyword a pipeline follows.
const PIPELINE_LAST_START_MAXIMUM = ACTION_CONTENT_BOUNDARY_MAXIMUM - KEYWORD_SIZE_MINIMUM

// Pipeline_Last_Start is the last token opening one pipeline begins after.
type Pipeline_Last_Start Cursor_Last_Start_Storage

// Pipeline_Last_Start_Invariants leaves the shortest keyword before the action end.
func Pipeline_Last_Start_Invariants(value Pipeline_Last_Start, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(SOURCE_SIZE_MINIMUM),
			uint16(PIPELINE_LAST_START_MAXIMUM)).
		Ensure()
}

// Pipeline_Cursor is one lexer a pipeline begins on.
type Pipeline_Cursor Token_Cursor

// Pipeline_Cursor_Invariants composes lexer state beside the token the pipeline follows.
func Pipeline_Cursor_Invariants(value Pipeline_Cursor, namespace aver.Namespace) {
	Cursor_Position_Stored_Invariants(Cursor_Position_Stored(value.Position), namespace)
	Cursor_End_Stored_Invariants(Cursor_End_Stored(value.End), namespace)
	Cursor_Token_Stored_Invariants(Cursor_Token_Stored(value.Token), namespace)
	Cursor_Look_Ahead_Stored_Invariants(Cursor_Look_Ahead_Stored(value.Looked), namespace)
	Cursor_Number_Stored_Invariants(Cursor_Number_Stored(value.Number), namespace)
	Function_Function_Storage_Invariants(value.Function, namespace)
	Pipeline_Last_Start_Invariants(Pipeline_Last_Start(value.Last_Start), namespace)
	Cursor_Has_Last_Stored_Invariants(Cursor_Has_Last_Stored(value.Has_Last), namespace)
}

// Pipeline_Cursor_Pointer keeps mutation of that lexer typed.
type Pipeline_Cursor_Pointer *Pipeline_Cursor

// Pipeline_Cursor_Pointer_Invariants composes present lexer state.
func Pipeline_Cursor_Pointer_Invariants(value Pipeline_Cursor_Pointer, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Pipeline_Cursor_Invariants(*value, namespace)
}

// Peeked_Cursor is one lexer that holds one token its caller already read.
type Peeked_Cursor Token_Cursor

// Peeked_Cursor_Invariants composes lexer state beside its look-ahead flag.
func Peeked_Cursor_Invariants(value Peeked_Cursor, namespace aver.Namespace) {
	Cursor_Position_Stored_Invariants(Cursor_Position_Stored(value.Position), namespace)
	Cursor_End_Stored_Invariants(Cursor_End_Stored(value.End), namespace)
	Cursor_Token_Stored_Invariants(Cursor_Token_Stored(value.Token), namespace)
	Cursor_Number_Stored_Invariants(Cursor_Number_Stored(value.Number), namespace)
	Function_Function_Storage_Invariants(value.Function, namespace)
	Cursor_Last_Start_Stored_Invariants(Cursor_Last_Start_Stored(value.Last_Start), namespace)
	Cursor_Has_Last_Stored_Invariants(Cursor_Has_Last_Stored(value.Has_Last), namespace)
	Peeked_Look_Ahead_Storage_Invariants(Peeked_Look_Ahead_Storage(value.Looked), namespace)
}

// Peeked_Cursor_Pointer keeps mutation of that lexer typed.
type Peeked_Cursor_Pointer *Peeked_Cursor

// Peeked_Cursor_Pointer_Invariants composes present lexer state.
func Peeked_Cursor_Pointer_Invariants(value Peeked_Cursor_Pointer, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Peeked_Cursor_Invariants(*value, namespace)
}

// Unread_Cursor is one lexer that holds no token yet.
type Unread_Cursor Token_Cursor

// Unread_Cursor_Invariants composes lexer state beside its look-ahead flag.
func Unread_Cursor_Invariants(value Unread_Cursor, namespace aver.Namespace) {
	Cursor_Position_Stored_Invariants(Cursor_Position_Stored(value.Position), namespace)
	Cursor_End_Stored_Invariants(Cursor_End_Stored(value.End), namespace)
	Cursor_Token_Stored_Invariants(Cursor_Token_Stored(value.Token), namespace)
	Cursor_Number_Stored_Invariants(Cursor_Number_Stored(value.Number), namespace)
	Function_Function_Storage_Invariants(value.Function, namespace)
	Cursor_Last_Start_Stored_Invariants(Cursor_Last_Start_Stored(value.Last_Start), namespace)
	Cursor_Has_Last_Stored_Invariants(Cursor_Has_Last_Stored(value.Has_Last), namespace)
	Unread_Look_Ahead_Storage_Invariants(Unread_Look_Ahead_Storage(value.Looked), namespace)
}

// Unread_Cursor_Pointer keeps mutation of that lexer typed.
type Unread_Cursor_Pointer *Unread_Cursor

// Unread_Cursor_Pointer_Invariants composes present lexer state.
func Unread_Cursor_Pointer_Invariants(value Unread_Cursor_Pointer, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Unread_Cursor_Invariants(*value, namespace)
}

// Exhausted_Token is one retained token holding no retained token.
type Exhausted_Token Token

// Exhausted_Token_Invariants fixes the kind one exhausted cursor retains.
func Exhausted_Token_Invariants(value Exhausted_Token, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Start),
			uint16(TOKEN_BOUNDARY_MINIMUM+utf8.CHARACTER_SIZE_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Range_Uint16(uint16(value.End),
			uint16(TOKEN_BOUNDARY_MINIMUM+utf8.CHARACTER_SIZE_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Ensure()
	aver.Always(uint8(value.Kind) == uint8(TOKEN_EOF),
		"A exhausted cursor retains the token its caller read.")
}

// Exhausted_Token_Storage is that retained token inside its cursor field.
type Exhausted_Token_Storage Token_Storage

// Exhausted_Token_Storage_Invariants composes the retained token.
func Exhausted_Token_Storage_Invariants(value Exhausted_Token_Storage, namespace aver.Namespace) {
	Exhausted_Token_Invariants(Exhausted_Token(value.Value), namespace)
}

// Exhausted_Cursor is one lexer whose retained token holds no retained token.
type Exhausted_Cursor Token_Cursor

// Exhausted_Cursor_Invariants composes lexer state beside the retained token.
func Exhausted_Cursor_Invariants(value Exhausted_Cursor, namespace aver.Namespace) {
	Exhausted_Position_Stored_Invariants(Exhausted_Position_Stored(value.Position), namespace)
	Exhausted_End_Stored_Invariants(Exhausted_End_Stored(value.End), namespace)
	Exhausted_Token_Storage_Invariants(Exhausted_Token_Storage(value.Token), namespace)
	Peeked_Look_Ahead_Storage_Invariants(Peeked_Look_Ahead_Storage(value.Looked), namespace)
	Cursor_Number_Stored_Invariants(Cursor_Number_Stored(value.Number), namespace)
	Function_Function_Storage_Invariants(value.Function, namespace)
	Exhausted_Last_Start_Invariants(Exhausted_Last_Start(value.Last_Start), namespace)
	Read_Has_Last_Stored_Invariants(Read_Has_Last_Stored(value.Has_Last), namespace)
}

// Exhausted_Cursor_Pointer keeps mutation of that lexer typed.
type Exhausted_Cursor_Pointer *Exhausted_Cursor

// Exhausted_Cursor_Pointer_Invariants composes present lexer state.
func Exhausted_Cursor_Pointer_Invariants(value Exhausted_Cursor_Pointer, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Exhausted_Cursor_Invariants(*value, namespace)
}

// Peeked_Look_Ahead_Storage is the flag of a cursor whose next token is already read.
type Peeked_Look_Ahead_Storage Look_Ahead_Storage

// Peeked_Look_Ahead_Storage_Invariants fixes the flag one peek sets.
func Peeked_Look_Ahead_Storage_Invariants(value Peeked_Look_Ahead_Storage, _ aver.Namespace,
) {
	aver.Always(uint8(value.Value) == uint8(utf8.CHARACTER_SIZE_MINIMUM),
		"A peeked cursor keeps the token its caller read.")
}

// Unread_Look_Ahead_Storage is the flag of a cursor holding no token.
type Unread_Look_Ahead_Storage Look_Ahead_Storage

// Unread_Look_Ahead_Storage_Invariants fixes the flag one read clears.
func Unread_Look_Ahead_Storage_Invariants(value Unread_Look_Ahead_Storage, _ aver.Namespace,
) {
	aver.Always(uint8(value.Value) == uint8(bytes.SLICE_SIZE_MINIMUM),
		"An unread cursor keeps no token.")
}

// List_Node is one syntax record holding one branch or document body.
type List_Node Node

// List_Node_Invariants bounds the record and fixes the kind its role admits.
func List_Node_Invariants(value List_Node, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Start), uint16(CONTROL_BODY_START_MINIMUM),
			uint16(SOURCE_SIZE_MAXIMUM-TEMPLATE_DEFINITION_END_SIZE_MINIMUM)).
		Range_Uint16(uint16(value.End), uint16(CONTROL_BODY_START_MINIMUM),
			uint16(SOURCE_SIZE_MAXIMUM-TEMPLATE_DEFINITION_END_SIZE_MINIMUM)).
		Range_Holed_Uint16(
			uint16(value.First_Child), uint16(NODE_REFERENCE_MINIMUM),
			uint16(NODE_REFERENCE_MAXIMUM-(CONTROL_BRANCH_NODE_COUNT+
				CONTROL_BRANCH_NODE_COUNT)),
			uint16(ROOT_LIST_HOLE), uint16(TERM_ACTION_HOLE),
			uint16(TERM_ACTION_HOLE)).
		Range_Holed_Uint16(
			uint16(value.Next_Sibling), uint16(NODE_REFERENCE_MINIMUM),
			uint16(BODY_LIST_SIBLING_MAXIMUM), uint16(ROOT_LIST_HOLE),
			uint16(TERM_ACTION_HOLE), uint16(TERM_ACTION_HOLE)).
		Ensure()
	aver.Always(uint16(value.Value_Start) == uint16(SOURCE_SIZE_MINIMUM),
		"A body record keeps no payload start.")
	aver.Always(uint16(value.Value_End) == uint16(SOURCE_SIZE_MINIMUM),
		"A body record keeps no payload end.")
	aver.Always(uint8(value.Kind) == uint8(NODE_LIST),
		"A body record keeps the kind its caller selected.")
}

// List_Node_Stored keeps one such record opaque between parser and evaluator.
type List_Node_Stored *List_Node

// List_Node_Stored_Invariants checks the present record.
func List_Node_Stored_Invariants(value List_Node_Stored, namespace aver.Namespace) {
	if value == nil {
		return
	}
	List_Node_Invariants(*value, namespace)
}

// List_Node_Validated carries one deeply checked record of that kind.
type List_Node_Validated struct {
	// Value keeps deep proof attached to the record.
	Value List_Node_Stored
}

// List_Node_Validated_Invariants checks transport ownership after deep validation.
func List_Node_Validated_Invariants(value List_Node_Validated, namespace aver.Namespace) {
	List_Node_Stored_Invariants(value.Value, namespace)
}

// TEMPLATE_INVOCATION_SIZE_MINIMUM is the shortest one-byte-delimited invocation.
const TEMPLATE_INVOCATION_SIZE_MINIMUM = ACTION_BOUNDARY_SIZE_MINIMUM +
	len(KEYWORD_TEMPLATE) + ACTION_SEPARATOR_SIZE_MINIMUM + QUOTED_BOUNDARY_SIZE

// INVOCATION_NAME_START_MINIMUM follows the shortest keyword one name can stand behind.
const INVOCATION_NAME_START_MINIMUM = CUSTOM_DELIMITER_SIZE_MINIMUM + len(KEYWORD_BLOCK) +
	ACTION_SEPARATOR_SIZE_MINIMUM

// INVOCATION_NAME_END_MINIMUM closes the shortest quoted name one invocation carries.
const INVOCATION_NAME_END_MINIMUM = INVOCATION_NAME_START_MINIMUM + QUOTED_BOUNDARY_SIZE +
	IDENTIFIER_SIZE_MINIMUM

// Definition_Node is one syntax record holding one template definition or invocation.
type Definition_Node Node

// Definition_Node_Invariants bounds the record and fixes the kind its role admits.
func Definition_Node_Invariants(value Definition_Node, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Start), uint16(SOURCE_SIZE_MINIMUM),
			uint16(SOURCE_SIZE_MAXIMUM-TEMPLATE_INVOCATION_SIZE_MINIMUM)).
		Range_Uint16(uint16(value.End), uint16(TEMPLATE_INVOCATION_SIZE_MINIMUM),
			uint16(SOURCE_SIZE_MAXIMUM)).
		Range_Uint16(uint16(value.Value_Start), uint16(INVOCATION_NAME_START_MINIMUM),
			uint16(SOURCE_SIZE_MAXIMUM-QUOTED_BOUNDARY_SIZE-IDENTIFIER_SIZE_MINIMUM)).
		Range_Uint16(uint16(value.Value_End), uint16(INVOCATION_NAME_END_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Range_Holed_Uint16(
			uint16(value.First_Child), uint16(NODE_REFERENCE_MINIMUM),
			uint16(INVOCATION_PIPELINE_REFERENCE_MAXIMUM), uint16(ROOT_LIST_HOLE),
			uint16(TERM_ACTION_HOLE), uint16(TERM_ACTION_HOLE)).
		Range_Holed_Uint16(uint16(value.Next_Sibling), uint16(NODE_REFERENCE_MINIMUM),
			uint16(INVOCATION_SIBLING_REFERENCE_MAXIMUM), uint16(ROOT_LIST_HOLE),
			uint16(TERM_ACTION_HOLE), uint16(TERM_ACTION_HOLE)).
		Ensure()
	aver.Always(uint8(value.Kind) == uint8(NODE_TEMPLATE),
		"A definition record keeps the kind its caller selected.")
}

// Definition_Node_Stored keeps one such record opaque between parser and evaluator.
type Definition_Node_Stored *Definition_Node

// Definition_Node_Stored_Invariants checks the present record.
func Definition_Node_Stored_Invariants(value Definition_Node_Stored, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Definition_Node_Invariants(*value, namespace)
}

// Definition_Node_Validated carries one deeply checked record of that kind.
type Definition_Node_Validated struct {
	// Value keeps deep proof attached to the record.
	Value Definition_Node_Stored
}

// Definition_Node_Validated_Invariants checks transport ownership after deep validation.
func Definition_Node_Validated_Invariants(value Definition_Node_Validated,
	namespace aver.Namespace) {
	Definition_Node_Stored_Invariants(value.Value, namespace)
}

// Action_Node is one syntax record holding a delimited action.
type Action_Node Node

// Action_Node_Invariants bounds the record and fixes the kind actions carry.
func Action_Node_Invariants(value Action_Node, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Start), uint16(SOURCE_SIZE_MINIMUM),
			uint16(SOURCE_SIZE_MAXIMUM)).
		Range_Uint16(uint16(value.End), uint16(SIMPLE_ACTION_SIZE_MINIMUM),
			uint16(SOURCE_SIZE_MAXIMUM)).
		Range_Uint16(
			uint16(value.Value_Start), uint16(ACTION_CONTENT_BOUNDARY_MINIMUM),
			uint16(SOURCE_SIZE_MAXIMUM-ACTION_BOUNDARY_SIZE_MINIMUM)).
		Range_Uint16(
			uint16(value.Value_End), uint16(CONSTANT_END_MINIMUM),
			uint16(SOURCE_SIZE_MAXIMUM-utf8.CHARACTER_SIZE_MINIMUM)).
		Range_Holed_Uint16(
			uint16(value.First_Child), uint16(NODE_REFERENCE_MINIMUM),
			uint16(NODE_REFERENCE_MAXIMUM-ACTION_PIPELINE_NODE_COUNT),
			uint16(ROOT_LIST_HOLE), uint16(TERM_ACTION_HOLE),
			uint16(TERM_ACTION_HOLE)).
		Range_Holed_Uint16(uint16(value.Next_Sibling), uint16(NODE_REFERENCE_MINIMUM),
			uint16(NODE_REFERENCE_MAXIMUM), uint16(ROOT_LIST_HOLE),
			uint16(TERM_ACTION_HOLE), uint16(TERM_ACTION_HOLE)).
		Ensure()
	aver.Always(uint8(value.Kind) == uint8(NODE_ACTION),
		"An action record keeps the kind its caller selected.")
}

// Action_Node_Stored keeps one action record opaque between parser and evaluator.
type Action_Node_Stored *Action_Node

// Action_Node_Stored_Invariants checks the present action record.
func Action_Node_Stored_Invariants(value Action_Node_Stored, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Action_Node_Invariants(*value, namespace)
}

// Action_Node_Validated carries one deeply checked action record.
type Action_Node_Validated struct {
	// Value keeps deep proof attached to the action record.
	Value Action_Node_Stored
}

// Action_Node_Validated_Invariants checks transport ownership after deep validation.
func Action_Node_Validated_Invariants(value Action_Node_Validated,
	namespace aver.Namespace) {
	Action_Node_Stored_Invariants(value.Value, namespace)
}

// CONTROL_ACTION_SIZE_MINIMUM spans the shortest branch beside the close it owes.
const CONTROL_ACTION_SIZE_MINIMUM = ACTION_BOUNDARY_SIZE_MINIMUM + len(KEYWORD_IF) +
	ACTION_SEPARATOR_SIZE_MINIMUM + DOT_SOURCE_SIZE + TEMPLATE_DEFINITION_END_SIZE_MINIMUM

// Control_Node is one syntax record holding a branch that owns a body.
type Control_Node Node

// Control_Node_Invariants bounds the record and fixes the branches it admits.
func Control_Node_Invariants(value Control_Node, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(uint8(value.Kind), uint8(NODE_IF), uint8(NODE_RANGE),
			uint8(NODE_WITH)).
		Range_Uint16(uint16(value.Start), uint16(SOURCE_SIZE_MINIMUM),
			uint16(SOURCE_SIZE_MAXIMUM-CONTROL_ACTION_SIZE_MINIMUM)).
		Range_Uint16(uint16(value.End), uint16(CONTROL_ACTION_SIZE_MINIMUM),
			uint16(SOURCE_SIZE_MAXIMUM)).
		Range_Uint16(uint16(value.Value_Start), uint16(CUSTOM_DELIMITER_SIZE_MINIMUM),
			uint16(SOURCE_SIZE_MAXIMUM-CONTROL_ACTION_SIZE_MINIMUM+
				CUSTOM_DELIMITER_SIZE_MINIMUM)).
		Range_Uint16(uint16(value.Value_End),
			uint16(CUSTOM_DELIMITER_SIZE_MINIMUM+len(KEYWORD_IF)),
			uint16(SOURCE_SIZE_MAXIMUM-CONTROL_ACTION_SIZE_MINIMUM+
				CUSTOM_DELIMITER_SIZE_MINIMUM+len(KEYWORD_IF))).
		Range_Uint16(uint16(value.First_Child), uint16(FIRST_TERM_REFERENCE),
			uint16(NODE_REFERENCE_MAXIMUM-CONTROL_BODY_TAIL_NODE_COUNT)).
		Range_Holed_Uint16(uint16(value.Next_Sibling), uint16(NODE_REFERENCE_MINIMUM),
			uint16(CONTROL_SIBLING_REFERENCE_MAXIMUM), uint16(ROOT_LIST_HOLE),
			uint16(TERM_ACTION_HOLE), uint16(TERM_ACTION_HOLE)).
		Ensure()
}

// Control_Node_Stored keeps one branch record opaque between parser and evaluator.
type Control_Node_Stored *Control_Node

// Control_Node_Stored_Invariants checks the present branch record.
func Control_Node_Stored_Invariants(value Control_Node_Stored, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Control_Node_Invariants(*value, namespace)
}

// Control_Node_Validated carries one deeply checked branch record.
type Control_Node_Validated struct {
	// Value keeps deep proof attached to the branch record.
	Value Control_Node_Stored
}

// Control_Node_Validated_Invariants checks transport ownership after deep validation.
func Control_Node_Validated_Invariants(value Control_Node_Validated,
	namespace aver.Namespace) {
	Control_Node_Stored_Invariants(value.Value, namespace)
}

// Pipe_Node is one syntax record holding one pipeline.
type Pipe_Node Node

// Pipe_Node_Invariants bounds the record and fixes the kind its role admits.
func Pipe_Node_Invariants(value Pipe_Node, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Start), uint16(ACTION_CONTENT_BOUNDARY_MINIMUM),
			uint16(LEXEME_START_MAXIMUM)).
		Range_Uint16(uint16(value.End), uint16(CONSTANT_END_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Range_Uint16(uint16(value.Value_Start), uint16(ACTION_CONTENT_BOUNDARY_MINIMUM),
			uint16(LEXEME_START_MAXIMUM)).
		Range_Uint16(uint16(value.Value_End), uint16(CONSTANT_END_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Range_Holed_Uint16(
			uint16(value.First_Child), uint16(NODE_REFERENCE_MINIMUM),
			uint16(NODE_REFERENCE_MAXIMUM), uint16(ROOT_LIST_HOLE),
			uint16(TERM_ACTION_HOLE), uint16(TERM_ACTION_HOLE)).
		Range_Holed_Uint16(
			uint16(value.Next_Sibling), uint16(NODE_REFERENCE_MINIMUM),
			uint16(ACTION_TAIL_REFERENCE_MAXIMUM),
			uint16(ROOT_LIST_HOLE), uint16(TERM_ACTION_HOLE),
			uint16(TERM_ACTION_HOLE)).
		Ensure()
	aver.Always(uint8(value.Kind) == uint8(NODE_PIPE),
		"A pipeline record keeps the kind its caller selected.")
}

// Pipe_Node_Stored keeps one such record opaque between parser and evaluator.
type Pipe_Node_Stored *Pipe_Node

// Pipe_Node_Stored_Invariants checks the present record.
func Pipe_Node_Stored_Invariants(value Pipe_Node_Stored, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Pipe_Node_Invariants(*value, namespace)
}

// Pipe_Node_Validated carries one deeply checked record of that kind.
type Pipe_Node_Validated struct {
	// Value keeps deep proof attached to the record.
	Value Pipe_Node_Stored
}

// Pipe_Node_Validated_Invariants checks transport ownership after deep validation.
func Pipe_Node_Validated_Invariants(value Pipe_Node_Validated, namespace aver.Namespace) {
	Pipe_Node_Stored_Invariants(value.Value, namespace)
}

// FLOAT_BITS_SUBNORMAL_HOLE is the bit pattern no parsed number lands on.
const FLOAT_BITS_SUBNORMAL_HOLE = bits.WORD_64_MINIMUM + bits.CARRY_MAXIMUM

// FLOAT_BITS_SUBNORMAL_NEXT is the second bit pattern no parsed number lands on.
const FLOAT_BITS_SUBNORMAL_NEXT = FLOAT_BITS_SUBNORMAL_HOLE + bits.CARRY_MAXIMUM

// FLOAT_BITS_PARSED_MAXIMUM is the negative largest finite binary64; syntax bars infinity.
const FLOAT_BITS_PARSED_MAXIMUM = big.FLOAT_64_NEGATIVE_INFINITY_BITS - bits.CARRY_MAXIMUM

// ROOT_LIST_HOLE is the root list slot, which holds no term and no child.
const ROOT_LIST_HOLE = ROOT_NODE_COUNT

// TERM_ACTION_HOLE is the action slot, which holds no term.
const TERM_ACTION_HOLE = ROOT_NODE_COUNT + ACTION_NODE_COUNT

// TERM_PIPELINE_HOLE is the pipeline slot, which holds no term.
const TERM_PIPELINE_HOLE = ROOT_NODE_COUNT + ACTION_PIPELINE_NODE_COUNT

// TERM_COMMAND_HOLE is the command slot, which holds no argument.
const TERM_COMMAND_HOLE = ROOT_NODE_COUNT + SIMPLE_ACTION_CONTAINER_NODE_COUNT

// FIRST_TERM_REFERENCE is the earliest slot a child of a first action can hold.
const FIRST_TERM_REFERENCE = ROOT_NODE_COUNT + ACTION_PIPELINE_NODE_COUNT

// FIRST_ARGUMENT_REFERENCE is the earliest slot a command argument can hold.
const FIRST_ARGUMENT_REFERENCE = ROOT_NODE_COUNT + SIMPLE_ACTION_CONTAINER_NODE_COUNT

// Chain_Node is one syntax record holding one field chain.
type Chain_Node Node

// Chain_Node_Invariants bounds the record and fixes the kind its role admits.
func Chain_Node_Invariants(value Chain_Node, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Start), uint16(ACTION_CONTENT_BOUNDARY_MINIMUM),
			uint16(CHAIN_START_MAXIMUM-DOT_SOURCE_SIZE)).
		Range_Uint16(uint16(value.End), uint16(CALLABLE_END_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Range_Uint16(uint16(value.Value_Start), uint16(CONSTANT_END_MINIMUM),
			uint16(CHAIN_START_MAXIMUM)).
		Range_Uint16(uint16(value.Value_End), uint16(CALLABLE_END_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Range_Uint16(
			uint16(value.First_Child), uint16(COMMAND_NODE_COUNT_MINIMUM),
			uint16(
				NODE_REFERENCE_MAXIMUM-CONTROL_BRANCH_NODE_COUNT-
					NODE_COUNT_INCREMENT)).
		Range_Holed_Uint16(
			uint16(value.Next_Sibling), uint16(NODE_REFERENCE_MINIMUM),
			uint16(ACTION_TAIL_REFERENCE_MAXIMUM), uint16(ROOT_LIST_HOLE),
			uint16(TERM_ACTION_HOLE), uint16(TERM_PIPELINE_HOLE)).
		Ensure()
	aver.Always(uint8(value.Kind) == uint8(NODE_CHAIN),
		"A chain record keeps the kind its caller selected.")
}

// Chain_Node_Stored keeps one such record opaque between parser and evaluator.
type Chain_Node_Stored *Chain_Node

// Chain_Node_Stored_Invariants checks the present record.
func Chain_Node_Stored_Invariants(value Chain_Node_Stored, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Chain_Node_Invariants(*value, namespace)
}

// Chain_Node_Validated carries one deeply checked record of that kind.
type Chain_Node_Validated struct {
	// Value keeps deep proof attached to the record.
	Value Chain_Node_Stored
}

// Chain_Node_Validated_Invariants checks transport ownership after deep validation.
func Chain_Node_Validated_Invariants(value Chain_Node_Validated, namespace aver.Namespace) {
	Chain_Node_Stored_Invariants(value.Value, namespace)
}

// FIELD_END_MINIMUM closes the shortest field access one action can hold.
const FIELD_END_MINIMUM = ACTION_CONTENT_BOUNDARY_MINIMUM + DOT_SOURCE_SIZE +
	IDENTIFIER_SIZE_MINIMUM

// FIELD_START_MAXIMUM leaves the shortest field access beside its closing delimiter.
const FIELD_START_MAXIMUM = ACTION_CONTENT_BOUNDARY_MAXIMUM - DOT_SOURCE_SIZE -
	IDENTIFIER_SIZE_MINIMUM

// Field_Node is one syntax record holding one field access.
type Field_Node Node

// Field_Node_Invariants bounds the record and fixes the kind its role admits.
func Field_Node_Invariants(value Field_Node, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Start), uint16(ACTION_CONTENT_BOUNDARY_MINIMUM),
			uint16(FIELD_START_MAXIMUM)).
		Range_Uint16(uint16(value.End), uint16(FIELD_END_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Range_Uint16(uint16(value.Value_Start), uint16(SOURCE_SIZE_MINIMUM),
			uint16(FIELD_START_MAXIMUM)).
		Range_Uint16(uint16(value.Value_End), uint16(FIELD_END_MINIMUM),
			uint16(SOURCE_SIZE_MAXIMUM)).
		Range_Holed_Uint16(
			uint16(value.Next_Sibling), uint16(NODE_REFERENCE_MINIMUM),
			uint16(ACTION_TAIL_REFERENCE_MAXIMUM),
			uint16(ROOT_LIST_HOLE), uint16(TERM_ACTION_HOLE),
			uint16(TERM_ACTION_HOLE)).
		Ensure()
	aver.Always(uint16(value.First_Child) == uint16(NO_NODE),
		"A field record has no child record.")
	aver.Always(uint8(value.Kind) == uint8(NODE_FIELD),
		"A field record keeps the kind its caller selected.")
}

// Field_Node_Stored keeps one such record opaque between parser and evaluator.
type Field_Node_Stored *Field_Node

// Field_Node_Stored_Invariants checks the present record.
func Field_Node_Stored_Invariants(value Field_Node_Stored, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Field_Node_Invariants(*value, namespace)
}

// Field_Node_Validated carries one deeply checked record of that kind.
type Field_Node_Validated struct {
	// Value keeps deep proof attached to the record.
	Value Field_Node_Stored
}

// Field_Node_Validated_Invariants checks transport ownership after deep validation.
func Field_Node_Validated_Invariants(value Field_Node_Validated, namespace aver.Namespace) {
	Field_Node_Stored_Invariants(value.Value, namespace)
}

// CALLABLE_START_MINIMUM follows one opening delimiter, one command, and one separator.
const CALLABLE_START_MINIMUM = ACTION_CONTENT_BOUNDARY_MINIMUM + IDENTIFIER_SIZE_MINIMUM +
	ACTION_SEPARATOR_SIZE_MINIMUM

// CALLABLE_END_MINIMUM closes the shortest callable one argument position holds.
const CALLABLE_END_MINIMUM = CALLABLE_START_MINIMUM + IDENTIFIER_SIZE_MINIMUM

// Callable_Node is one syntax record holding one callable name.
type Callable_Node Node

// Callable_Node_Invariants bounds the record and fixes the kind its role admits.
func Callable_Node_Invariants(value Callable_Node, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Start), uint16(CALLABLE_START_MINIMUM),
			uint16(LEXEME_START_MAXIMUM)).
		Range_Uint16(uint16(value.End), uint16(CALLABLE_END_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Range_Uint16(uint16(value.Value_Start), uint16(CALLABLE_START_MINIMUM),
			uint16(LEXEME_START_MAXIMUM)).
		Range_Uint16(uint16(value.Value_End), uint16(CALLABLE_END_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Range_Holed_Uint16(
			uint16(value.Next_Sibling), uint16(NODE_REFERENCE_MINIMUM),
			uint16(CONTAINER_TAIL_REFERENCE_MAXIMUM), uint16(ROOT_LIST_HOLE),
			uint16(TERM_ACTION_HOLE), uint16(TERM_ACTION_HOLE)).
		Ensure()
	aver.Always(uint16(value.First_Child) == uint16(NO_NODE),
		"A callable record has no child record.")
	aver.Always(uint8(value.Kind) == uint8(NODE_IDENTIFIER),
		"A callable record keeps the kind its caller selected.")
}

// Callable_Node_Stored keeps one such record opaque between parser and evaluator.
type Callable_Node_Stored *Callable_Node

// Callable_Node_Stored_Invariants checks the present record.
func Callable_Node_Stored_Invariants(value Callable_Node_Stored, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Callable_Node_Invariants(*value, namespace)
}

// Callable_Node_Validated carries one deeply checked record of that kind.
type Callable_Node_Validated struct {
	// Value keeps deep proof attached to the record.
	Value Callable_Node_Stored
}

// Callable_Node_Validated_Invariants checks transport ownership after deep validation.
func Callable_Node_Validated_Invariants(value Callable_Node_Validated, namespace aver.Namespace) {
	Callable_Node_Stored_Invariants(value.Value, namespace)
}

// Variable_Node is one syntax record holding one variable name.
type Variable_Node Node

// Variable_Node_Invariants bounds the record and fixes the kind its role admits.
func Variable_Node_Invariants(value Variable_Node, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Start), uint16(ACTION_CONTENT_BOUNDARY_MINIMUM),
			uint16(LEXEME_START_MAXIMUM)).
		Range_Uint16(uint16(value.End),
			uint16(ACTION_CONTENT_BOUNDARY_MINIMUM+VARIABLE_NAME_SIZE_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Range_Uint16(uint16(value.Value_Start), uint16(SOURCE_SIZE_MINIMUM),
			uint16(LEXEME_START_MAXIMUM)).
		Range_Uint16(uint16(value.Value_End),
			uint16(ACTION_CONTENT_BOUNDARY_MINIMUM+VARIABLE_NAME_SIZE_MINIMUM),
			uint16(SOURCE_SIZE_MAXIMUM)).
		Range_Holed_Uint16(
			uint16(value.Next_Sibling), uint16(NODE_REFERENCE_MINIMUM),
			uint16(CONTAINER_TAIL_REFERENCE_MAXIMUM), uint16(ROOT_LIST_HOLE),
			uint16(TERM_ACTION_HOLE), uint16(TERM_ACTION_HOLE)).
		Ensure()
	aver.Always(uint16(value.First_Child) == uint16(NO_NODE),
		"A variable record has no child record.")
	aver.Always(uint8(value.Kind) == uint8(NODE_VARIABLE),
		"A variable record keeps the kind its caller selected.")
}

// Variable_Node_Stored keeps one such record opaque between parser and evaluator.
type Variable_Node_Stored *Variable_Node

// Variable_Node_Stored_Invariants checks the present record.
func Variable_Node_Stored_Invariants(value Variable_Node_Stored, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Variable_Node_Invariants(*value, namespace)
}

// Variable_Node_Validated carries one deeply checked record of that kind.
type Variable_Node_Validated struct {
	// Value keeps deep proof attached to the record.
	Value Variable_Node_Stored
}

// Variable_Node_Validated_Invariants checks transport ownership after deep validation.
func Variable_Node_Validated_Invariants(value Variable_Node_Validated, namespace aver.Namespace) {
	Variable_Node_Stored_Invariants(value.Value, namespace)
}

// One chain evaluates its pipeline first and keeps the field path for the result.
func evaluation_chain_push(
	evaluations Evaluations, program_value Chain_Program, chain_value Chain_Node_Validated,
	depth_value Chain_Parent_Depth, argument_count_value Chained_Argument_Count,
) (_ Chain_Depth, status Chain_Push_Status) {
	defer func() { Chain_Push_Status_Invariants(status, "evaluation_chain_push.status") }()
	Evaluations_Invariants(evaluations, "evaluation_chain_push.evaluations")
	Chain_Program_Invariants(program_value, "evaluation_chain_push.program_value")
	Chain_Node_Validated_Invariants(chain_value, "evaluation_chain_push.chain_value")
	Chain_Parent_Depth_Invariants(depth_value, "evaluation_chain_push.depth_value")
	Chained_Argument_Count_Invariants(
		argument_count_value, "evaluation_chain_push.argument_count_value")
	depth := Chain_Depth(depth_value)
	defer func() { Chain_Depth_Invariants(depth, "evaluation_chain_push.depth") }()
	chain := (*Node)((*Chain_Node)(chain_value.Value))
	reference := Node_Link_Validated{Value: Node_Link_Proof_Stored(chain.First_Child)}
	pipe_value, pipe_found := program_node(Walked_Program(program_value), reference)
	pipe := (*Node)(pipe_value.Value)
	if pipe.Kind != NODE_PIPE {
		return depth, Chain_Push_Status(TERM_STATUS_PROGRAM_INVALID)
	}
	if !bool(pipe_found) {
		return depth, Chain_Push_Status(TERM_STATUS_PROGRAM_INVALID)
	}
	child_depth, child_status := evaluation_child_push(
		evaluations, Pushed_Chain_Program(program_value),
		Child_Pipe_Node_Validated{Value: (*Child_Pipe_Node)(pipe)},
		Child_Chain_Node_Optional{Value: (*Child_Chain_Node)(chain)},
		Active_Frame_Count(depth), Staged_Argument_Count(argument_count_value))
	depth = Chain_Depth(child_depth)
	return depth, Chain_Push_Status(child_status)
}

// Chained_Argument_Count is the argument list staged beneath one chained term.
type Chained_Argument_Count Argument_Count_Tracked

// Chained_Argument_Count_Invariants leaves the shortest chained term beside those arguments.
func Chained_Argument_Count_Invariants(value Chained_Argument_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(ARGUMENT_COUNT_MINIMUM),
			uint16(ARGUMENT_BASE_MAXIMUM)).
		Ensure()
}

// Pushed_Evaluation_Frame is one frame that has only its pipeline and dot.
type Pushed_Evaluation_Frame Evaluation_Frame

// Pushed_Evaluation_Frame_Invariants fixes every field no command has filled yet.
func Pushed_Evaluation_Frame_Invariants(value Pushed_Evaluation_Frame, namespace aver.Namespace,
) {
	Pushed_Evaluation_Nodes_Invariants(Pushed_Evaluation_Nodes(value.Nodes), namespace)
	Pushed_Evaluation_Values_Invariants(Pushed_Evaluation_Values(value.Values), namespace)
	Pushed_Evaluation_References_Invariants(
		Pushed_Evaluation_References(value.References), namespace)
	Pushed_Evaluation_States_Invariants(Pushed_Evaluation_States(value.States), namespace)
	Evaluation_Argument_Bases_Stored_Invariants(
		Evaluation_Argument_Bases_Stored(value.Argument_Bases), namespace)
	Pushed_Evaluation_Control_Invariants(Pushed_Evaluation_Control(value.Control), namespace)
}

// Pushed_Evaluation_State is the pipeline state of a frame before its first command.
type Pushed_Evaluation_State Pipeline_State_Stored

// Pushed_Evaluation_State_Invariants keeps declarations and fixes the empty result.
func Pushed_Evaluation_State_Invariants(value Pushed_Evaluation_State, namespace aver.Namespace,
) {
	Declaration_References_Stored_Invariants(
		Declaration_References_Stored(value.Declarations), namespace)
	aver.Tree(value, namespace).
		Enum_3_Uint8(uint8(value.Declaration_Count), uint8(DECLARATION_COUNT_NONE),
			uint8(DECLARATION_COUNT_ONE), uint8(DECLARATION_COUNT_TWO)).
		Sometimes(bool(value.Assignment), "A pushed frame updates an existing variable.").
		Ensure()
	Pushed_Pipeline_Value_Invariants(Pushed_Pipeline_Value(value.Value), namespace)
}

// Pushed_Evaluation_Frame_Pointer keeps mutation of a pushed frame typed.
type Pushed_Evaluation_Frame_Pointer *Pushed_Evaluation_Frame

// Pushed_Evaluation_Frame_Pointer_Invariants composes one present pushed frame.
func Pushed_Evaluation_Frame_Pointer_Invariants(
	value Pushed_Evaluation_Frame_Pointer, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Pushed_Evaluation_Frame_Invariants(*value, namespace)
}

// Pushed_Evaluation_Nodes is the syntax of a frame before any command reads a name.
type Pushed_Evaluation_Nodes Evaluation_Nodes

// Pushed_Evaluation_Nodes_Invariants fixes both spans as empty.
func Pushed_Evaluation_Nodes_Invariants(value Pushed_Evaluation_Nodes, namespace aver.Namespace,
) {
	Pushed_Identifier_Span_Invariants(Pushed_Identifier_Span(value.Identifier), namespace)
	Pushed_Chain_Span_Invariants(Pushed_Chain_Span(value.Chain), namespace)
}

// Pushed_Identifier_Span is the callable name of a frame that read none.
type Pushed_Identifier_Span Evaluation_Identifier_Span

// Pushed_Identifier_Span_Invariants fixes the empty name span.
func Pushed_Identifier_Span_Invariants(value Pushed_Identifier_Span, _ aver.Namespace) {
	aver.Always(uint16(value.Start) == uint16(SOURCE_SIZE_MINIMUM),
		"A pushed frame keeps no callable name start.")
	aver.Always(uint16(value.End) == uint16(SOURCE_SIZE_MINIMUM),
		"A pushed frame keeps no callable name end.")
}

// Pushed_Chain_Span is the field path of a frame that read none.
type Pushed_Chain_Span Evaluation_Chain_Span

// Pushed_Chain_Span_Invariants fixes the empty path span.
func Pushed_Chain_Span_Invariants(value Pushed_Chain_Span, _ aver.Namespace) {
	aver.Always(uint16(value.Start) == uint16(SOURCE_SIZE_MINIMUM),
		"A pushed frame keeps no field path start.")
	aver.Always(uint16(value.End) == uint16(SOURCE_SIZE_MINIMUM),
		"A pushed frame keeps no field path end.")
}

// Pushed_Evaluation_Values holds the pipeline cursor and no command result.
type Pushed_Evaluation_Values Evaluation_Values

// Pushed_Evaluation_Values_Invariants keeps the cursor and fixes the empty result.
func Pushed_Evaluation_Values_Invariants(value Pushed_Evaluation_Values, namespace aver.Namespace,
) {
	Evaluation_Dot_Value_Invariants(value.Dot, namespace)
	Pushed_Current_Value_Invariants(Pushed_Current_Value(value.Current), namespace)
}

// Pushed_Current_Value is the first command result of a frame that ran none.
type Pushed_Current_Value Evaluation_Current_Value

// Pushed_Current_Value_Invariants fixes the empty result.
func Pushed_Current_Value_Invariants(value Pushed_Current_Value, _ aver.Namespace) {
	aver.Always(uint8(value.Kind) == uint8(VALUE_NIL),
		"A pushed frame keeps no first command result.")
	aver.Always(uint64(value.Real) == bits.WORD_64_MINIMUM,
		"A pushed frame leaves its command result's primary word empty.")
	aver.Always(uint64(value.Imaginary) == bits.WORD_64_MINIMUM,
		"A pushed frame leaves its command result's imaginary word empty.")
	aver.Always(len(value.Text) == OUTPUT_SIZE_MINIMUM,
		"A pushed frame leaves its command result's text empty.")
	aver.Always(len(value.Values) == bytes.SLICE_SIZE_MINIMUM,
		"A pushed frame leaves its command result's sequence empty.")
	aver.Always(len(value.Fields) == bytes.SLICE_SIZE_MINIMUM,
		"A pushed frame leaves its command result's members empty.")
}

// Pushed_Evaluation_References holds the first command and no argument.
type Pushed_Evaluation_References Evaluation_References

// Pushed_Evaluation_References_Invariants bounds the command and fixes the argument.
func Pushed_Evaluation_References_Invariants(
	value Pushed_Evaluation_References, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Holed_Uint16(
			uint16(value.Command), uint16(NODE_REFERENCE_MINIMUM),
			uint16(NODE_REFERENCE_MAXIMUM), uint16(ROOT_LIST_HOLE),
			uint16(TERM_ACTION_HOLE), uint16(TERM_ACTION_HOLE)).
		Ensure()
	aver.Always(uint16(value.Term) == uint16(NO_NODE),
		"A pushed frame has no argument to evaluate yet.")
}

// Pushed_Evaluation_Control is the state machine of a frame at its first command.
type Pushed_Evaluation_Control Evaluation_Control

// Pushed_Evaluation_Control_Invariants fixes every flag one push leaves.
func Pushed_Evaluation_Control_Invariants(value Pushed_Evaluation_Control, _ aver.Namespace) {
	aver.Always(uint8(value.Stage) == uint8(EVALUATION_STAGE_COMMAND),
		"A pushed frame starts at its first command.")
	aver.Always(uint8(value.Mode) == uint8(EVALUATION_MODE_VALUE),
		"A pushed frame has read no callable name.")
	aver.Always(uint8(value.Present) == uint8(EVALUATION_CONTROL_FALSE),
		"A pushed frame holds no earlier command result.")
	aver.Always(uint8(value.Chain) == uint8(EVALUATION_CONTROL_FALSE),
		"A pushed frame holds no pending field path.")
}

// Pushed_Evaluation_States is the pipeline state storage of a pushed frame.
type Pushed_Evaluation_States Evaluation_States

// Pushed_Evaluation_States_Invariants composes the state before any command.
func Pushed_Evaluation_States_Invariants(value Pushed_Evaluation_States, namespace aver.Namespace,
) {
	Pushed_Evaluation_State_Invariants(Pushed_Evaluation_State(value.Value), namespace)
}

// Pushed_Pipeline_Value is the pipeline result of a frame that ran no command.
type Pushed_Pipeline_Value Value

// Pushed_Pipeline_Value_Invariants fixes every payload as empty.
func Pushed_Pipeline_Value_Invariants(value Pushed_Pipeline_Value, _ aver.Namespace) {
	aver.Always(
		uint8(value.Kind) == uint8(VALUE_NIL), "A pushed frame keeps no pipeline result.",
	)
	aver.Always(uint64(value.Real) == bits.WORD_64_MINIMUM,
		"A pushed frame leaves its pipeline result's primary word empty.")
	aver.Always(uint64(value.Imaginary) == bits.WORD_64_MINIMUM,
		"A pushed frame leaves its pipeline result's imaginary word empty.")
	aver.Always(len(value.Text) == OUTPUT_SIZE_MINIMUM,
		"A pushed frame leaves its pipeline result's text empty.")
	aver.Always(len(value.Values) == bytes.SLICE_SIZE_MINIMUM,
		"A pushed frame leaves its pipeline result's sequence empty.")
	aver.Always(len(value.Fields) == bytes.SLICE_SIZE_MINIMUM,
		"A pushed frame leaves its pipeline result's members empty.")
}

// READ_CURSOR_BOUNDARY_MINIMUM is the first boundary of an action holding one byte.
const READ_CURSOR_BOUNDARY_MINIMUM = ACTION_CONTENT_BOUNDARY_MINIMUM +
	utf8.CHARACTER_SIZE_MINIMUM

// Read_Cursor_Position is the next unread byte of an action that holds content.
type Read_Cursor_Position Cursor_Position_Storage

// Read_Cursor_Position_Invariants bounds the position by an action with content.
func Read_Cursor_Position_Invariants(value Read_Cursor_Position, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(READ_CURSOR_BOUNDARY_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Ensure()
}

// Read_Cursor_End is the content boundary of an action that holds content.
type Read_Cursor_End Cursor_End_Storage

// Read_Cursor_End_Invariants bounds the boundary by an action with content.
func Read_Cursor_End_Invariants(value Read_Cursor_End, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(READ_CURSOR_BOUNDARY_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Ensure()
}

// Chain_Node_Optional_Validated carries an absent chain or one checked chain record.
type Chain_Node_Optional_Validated struct {
	// Value keeps optional-chain proof attached to the record.
	Value Chain_Node_Stored
}

// Chain_Node_Optional_Validated_Invariants checks the record when one is present.
func Chain_Node_Optional_Validated_Invariants(
	value Chain_Node_Optional_Validated, namespace aver.Namespace,
) {
	Chain_Node_Stored_Invariants(value.Value, namespace)
}

// CONTROL_BODY_START_MINIMUM follows the shortest control action one source holds.
const CONTROL_BODY_START_MINIMUM = KEYWORD_SOURCE_SIZE_MINIMUM +
	ACTION_SEPARATOR_SIZE_MINIMUM + DOT_SOURCE_SIZE

// PIPELINE_SEPARATOR_SIZE is one pipe byte.
const PIPELINE_SEPARATOR_SIZE = len("|")

// PARENTHESIS_SIZE is one parenthesis byte.
const PARENTHESIS_SIZE = len("(")

// ELSE_TOKEN_END_MINIMUM ends after one else keyword.
const ELSE_TOKEN_END_MINIMUM = ACTION_CONTENT_BOUNDARY_MINIMUM + len(KEYWORD_ELSE)

// END_TOKEN_END_MINIMUM ends after one end keyword.
const END_TOKEN_END_MINIMUM = ACTION_CONTENT_BOUNDARY_MINIMUM + len(KEYWORD_END)

// COMMENT_TOKEN_END_MINIMUM ends after one comment.
const COMMENT_TOKEN_END_MINIMUM = ACTION_CONTENT_BOUNDARY_MINIMUM + len(`/**/`)

// INVOCATION_TOKEN_END_MINIMUM ends after one template keyword.
const INVOCATION_TOKEN_END_MINIMUM = ACTION_CONTENT_BOUNDARY_MINIMUM + len(KEYWORD_TEMPLATE)

// SEPARATOR_TOKEN_END_MINIMUM ends after one pipe after one term.
const SEPARATOR_TOKEN_END_MINIMUM = ACTION_CONTENT_BOUNDARY_MINIMUM + IDENTIFIER_SIZE_MINIMUM +
	PIPELINE_SEPARATOR_SIZE

// CLOSE_PAREN_TOKEN_END_MINIMUM ends after one closing parenthesis after one term.
const CLOSE_PAREN_TOKEN_END_MINIMUM = ACTION_CONTENT_BOUNDARY_MINIMUM +
	PARENTHESIS_SIZE + IDENTIFIER_SIZE_MINIMUM + PARENTHESIS_SIZE

// IDENTIFIER_ACTION_SIZE_MINIMUM is the shortest one-byte-delimited call.
const IDENTIFIER_ACTION_SIZE_MINIMUM = ACTION_BOUNDARY_SIZE_MINIMUM + IDENTIFIER_SIZE_MINIMUM

// PIPELINE_REFERENCE_MAXIMUM leaves room for one command and one term after a pipeline.
const PIPELINE_REFERENCE_MAXIMUM = ROOT_NODE_COUNT + ACTION_PIPELINE_NODE_COUNT +
	(SOURCE_SIZE_MAXIMUM-IDENTIFIER_ACTION_SIZE_MINIMUM)/SIMPLE_ACTION_SIZE_MINIMUM*
		SIMPLE_ACTION_NODE_COUNT + TRAILING_TEXT_NODE_COUNT_MAXIMUM

// COMMAND_REFERENCE_MAXIMUM leaves room for one term after a command.
const COMMAND_REFERENCE_MAXIMUM = NODE_COUNT_MAXIMUM

// ARGUMENT_REFERENCE_MAXIMUM leaves room for the text one dense source ends with.
const ARGUMENT_REFERENCE_MAXIMUM = COMMAND_REFERENCE_MAXIMUM -
	TRAILING_TEXT_NODE_COUNT_MAXIMUM

// Mapped_Value is one map a range walks entry by entry.
type Mapped_Value Value

// Mapped_Value_Invariants fixes the kind and every payload a map never owns.
func Mapped_Value_Invariants(value Mapped_Value, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value.Fields), MAP_ENTRY_COUNT_INCREMENT, RANGE_MAP_ENTRY_COUNT_MAXIMUM,
		).
		Ensure()
	aver.Always(uint8(value.Kind) == uint8(VALUE_MAP), "A ranged map keeps its own kind.")
	aver.Always(uint64(value.Real) == bits.WORD_64_MINIMUM,
		"Validation leaves a ranged map's primary word empty.")
	aver.Always(uint64(value.Imaginary) == bits.WORD_64_MINIMUM,
		"Validation leaves a ranged map's imaginary word empty.")
	aver.Always(len(value.Text) == OUTPUT_SIZE_MINIMUM,
		"Validation leaves a ranged map's text empty.")
	aver.Always(len(value.Values) == bytes.SLICE_SIZE_MINIMUM,
		"Validation leaves a ranged map's sequence empty.")
}

// Iterable_Value is one sequence or counted integer a range walks by position.
type Iterable_Value Value

// Iterable_Value_Invariants fixes every kind a positional range admits.
func Iterable_Value_Invariants(value Iterable_Value, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(uint8(value.Kind), uint8(VALUE_INTEGER), uint8(VALUE_UNSIGNED),
			uint8(VALUE_SEQUENCE)).
		Range_Uint64(uint64(value.Real), bits.WORD_64_MINIMUM, uint64(VALUE_COUNT_MAXIMUM)).
		Range_Int(len(value.Values), bytes.SLICE_SIZE_MINIMUM, VALUE_COUNT_MAXIMUM).
		Ensure()
	aver.Always(uint64(value.Imaginary) == bits.WORD_64_MINIMUM,
		"Validation leaves an iterated value's imaginary word empty.")
	aver.Always(len(value.Text) == OUTPUT_SIZE_MINIMUM,
		"Validation leaves an iterated value's text empty.")
	aver.Always(len(value.Fields) == bytes.SLICE_SIZE_MINIMUM,
		"Validation leaves an iterated value's members empty.")
}

// VARIABLE_MARK_SIZE is the byte that opens one variable name.
const VARIABLE_MARK_SIZE = len("$")

// FIRST_DECLARATION_ACTION_SIZE_MINIMUM is the shortest action that binds one bare name.
const FIRST_DECLARATION_ACTION_SIZE_MINIMUM = ACTION_BOUNDARY_SIZE_MINIMUM +
	VARIABLE_MARK_SIZE + ASSIGNMENT_OPERATOR_SIZE + DOT_SOURCE_SIZE

// FIRST_DECLARATION_PACKED_COUNT packs the densest actions and one text node before that
// action.
const FIRST_DECLARATION_PACKED_COUNT = ROOT_NODE_COUNT +
	(SOURCE_SIZE_MAXIMUM-FIRST_DECLARATION_ACTION_SIZE_MINIMUM)/SIMPLE_ACTION_SIZE_MINIMUM*
		SIMPLE_ACTION_NODE_COUNT +
	((SOURCE_SIZE_MAXIMUM-FIRST_DECLARATION_ACTION_SIZE_MINIMUM)%SIMPLE_ACTION_SIZE_MINIMUM+
		SIMPLE_ACTION_SIZE_MINIMUM-SOURCE_POSITION_INCREMENT)/SIMPLE_ACTION_SIZE_MINIMUM

// FIRST_DECLARATION_REFERENCE_MAXIMUM is that action's name, past its action and pipeline.
const FIRST_DECLARATION_REFERENCE_MAXIMUM = FIRST_DECLARATION_PACKED_COUNT +
	ACTION_PIPELINE_NODE_COUNT + NODE_COUNT_INCREMENT

// SECOND_DECLARATION_ACTION_SIZE_MINIMUM is the shortest range, with its end, that declares two
// bare names.
const SECOND_DECLARATION_ACTION_SIZE_MINIMUM = FIRST_DECLARATION_ACTION_SIZE_MINIMUM +
	len(KEYWORD_RANGE) + ACTION_SEPARATOR_SIZE_MINIMUM + VARIABLE_MARK_SIZE +
	DECLARATION_SEPARATOR_SIZE + TEMPLATE_DEFINITION_END_SIZE_MINIMUM

// DECLARATION_SEPARATOR_SIZE is the comma between two declared names.
const DECLARATION_SEPARATOR_SIZE = len(",")

// SECOND_DECLARATION_REFERENCE_MAXIMUM is that range's second name behind the densest actions,
// one text node, its range node, its pipeline, and its first name.
const SECOND_DECLARATION_REFERENCE_MAXIMUM = ROOT_NODE_COUNT +
	(SOURCE_SIZE_MAXIMUM-SECOND_DECLARATION_ACTION_SIZE_MINIMUM)/SIMPLE_ACTION_SIZE_MINIMUM*
		SIMPLE_ACTION_NODE_COUNT +
	((SOURCE_SIZE_MAXIMUM-SECOND_DECLARATION_ACTION_SIZE_MINIMUM)%SIMPLE_ACTION_SIZE_MINIMUM+
		SIMPLE_ACTION_SIZE_MINIMUM-SOURCE_POSITION_INCREMENT)/SIMPLE_ACTION_SIZE_MINIMUM +
	ACTION_PIPELINE_NODE_COUNT + NODE_COUNT_INCREMENT + NODE_COUNT_INCREMENT

// ACTION_LIST_HOLE is the action record that always precedes a branch body.
const ACTION_LIST_HOLE = ROOT_NODE_COUNT + ACTION_NODE_COUNT

// Ranged_Dot_Value is the collection or count one range frame walks.
type Ranged_Dot_Value Value

// Ranged_Dot_Value_Invariants fixes every kind and payload a range admits.
func Ranged_Dot_Value_Invariants(value Ranged_Dot_Value, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value.Kind), uint8(VALUE_INTEGER), uint8(VALUE_MAP)).
		Range_Uint64(uint64(value.Real), bits.WORD_64_MINIMUM, uint64(VALUE_COUNT_MAXIMUM)).
		Range_Int(len(value.Values), bytes.SLICE_SIZE_MINIMUM, VALUE_COUNT_MAXIMUM).
		Range_Int(len(value.Fields), bytes.SLICE_SIZE_MINIMUM,
			RANGE_MAP_ENTRY_COUNT_MAXIMUM).
		Ensure()
	aver.Always(uint64(value.Imaginary) == bits.WORD_64_MINIMUM,
		"A ranged value leaves its imaginary word empty.")
	aver.Always(
		len(value.Text) == OUTPUT_SIZE_MINIMUM, "A ranged value leaves its text empty.")
}

// RANGE_INVOCATION_SOURCE_SIZE_MINIMUM is the shortest named range beside its invocation.
const RANGE_INVOCATION_SOURCE_SIZE_MINIMUM = TEMPLATE_DEFINITION_OPEN_SIZE_MINIMUM +
	RANGE_BODY_SOURCE_SIZE_MINIMUM + TEMPLATE_DEFINITION_END_SIZE_MINIMUM +
	TEMPLATE_INVOCATION_SIZE_MINIMUM + ACTION_SEPARATOR_SIZE_MINIMUM + DOT_SOURCE_SIZE

// RANGE_SCOPE_BASE_MAXIMUM leaves the root variable one invocation still binds above that base.
const RANGE_SCOPE_BASE_MAXIMUM = VARIABLE_COUNT_MAXIMUM - VARIABLE_COUNT_INCREMENT

// RANGE_BODY_SOURCE_SIZE_MINIMUM is the shortest range beside one body byte.
const RANGE_BODY_SOURCE_SIZE_MINIMUM = ACTION_BOUNDARY_SIZE_MINIMUM + len(KEYWORD_RANGE) +
	ACTION_SEPARATOR_SIZE_MINIMUM + DOT_SOURCE_SIZE + utf8.CHARACTER_SIZE_MINIMUM +
	TEMPLATE_DEFINITION_END_SIZE_MINIMUM

// RANGE_BODY_REFERENCE_MINIMUM follows the range, its pipeline, its command, and its term.
const RANGE_BODY_REFERENCE_MINIMUM = ROOT_NODE_COUNT + ACTION_PIPELINE_NODE_COUNT +
	COMMAND_NODE_COUNT + SCALAR_NODE_COUNT + NODE_COUNT_INCREMENT

// RANGE_BODY_REFERENCE_MAXIMUM packs actions beside the shortest range and its body.
const RANGE_BODY_REFERENCE_MAXIMUM = RANGE_BODY_REFERENCE_MINIMUM + NODE_COUNT_INCREMENT +
	(SOURCE_SIZE_MAXIMUM-RANGE_BODY_SOURCE_SIZE_MINIMUM)/SIMPLE_ACTION_SIZE_MINIMUM*
		SIMPLE_ACTION_NODE_COUNT + TRAILING_TEXT_NODE_COUNT_MAXIMUM

// RANGE_FRAME_RESTORE marks that a range frame restores the scope its list opened.
const RANGE_FRAME_RESTORE = true

// Range_Frame_State_Stored is traversal state whose cursor holds a ranged value.
type Range_Frame_State_Stored Frame_Stored

// Range_Frame_State_Stored_Invariants fixes traversal state beside a ranged cursor.
func Range_Frame_State_Stored_Invariants(value Range_Frame_State_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Holed_Uint16(
			uint16(value.Reference), uint16(NODE_REFERENCE_MINIMUM),
			uint16(RANGE_BODY_REFERENCE_MAXIMUM), uint16(ROOT_LIST_HOLE),
			uint16(TERM_ACTION_HOLE), uint16(TERM_ACTION_HOLE)).
		Range_Uint16(uint16(value.Index), uint16(bytes.SLICE_SIZE_MINIMUM), uint16(
			VALUE_COUNT_MAXIMUM)).
		Range_Uint16(uint16(value.Map_Entry_Index), uint16(bytes.SLICE_SIZE_MINIMUM),
			uint16(RANGE_MAP_ENTRY_INDEX_MAXIMUM)).
		Enum_4_Uint8(uint8(value.Map_Selected), uint8(MAP_SELECTION_NONE),
			uint8(MAP_SELECTION_FORWARD), uint8(MAP_SELECTION_BACKWARD),
			uint8(MAP_SELECTION_SCAN)).
		Range_Uint16(uint16(value.Variable_Mark), uint16(VARIABLE_COUNT_INCREMENT),
			uint16(VARIABLE_COUNT_MAXIMUM)).
		Range_Uint16(uint16(value.Iteration_Variables), uint16(VARIABLE_COUNT_INCREMENT),
			uint16(VARIABLE_COUNT_MAXIMUM)).
		Range_Uint16(
			uint16(value.Scope_Base), uint16(bytes.SLICE_SIZE_MINIMUM),
			uint16(RANGE_SCOPE_BASE_MAXIMUM)).
		Range_Uint16(
			uint16(value.Previous_Scope_Base), uint16(bytes.SLICE_SIZE_MINIMUM),
			uint16(RANGE_SCOPE_BASE_MAXIMUM)).
		Enum_3_Uint8(uint8(value.Declaration_Count), uint8(DECLARATION_COUNT_NONE), uint8(
			DECLARATION_COUNT_ONE), uint8(DECLARATION_COUNT_TWO)).
		Sometimes(bool(value.Assignment), "Pipeline updates existing variable.").
		Ensure()
	aver.Always(bool(value.Restore) == RANGE_FRAME_RESTORE,
		"One ranged frame restores the scope its list opened.")
	aver.Always(uint8(value.Kind) == uint8(FRAME_RANGE),
		"One ranged frame keeps the kind its caller selected.")
	Ranged_Dot_Value_Invariants(Ranged_Dot_Value(value.Dot), namespace)
	Ranged_Declaration_References_Invariants(
		Ranged_Declaration_References(value.Declarations), namespace)
}

// Range_Frame_Storage is caller frame storage a range owns.
type Range_Frame_Storage Frame_Storage

// Range_Frame_Storage_Invariants checks ownership of that frame storage.
func Range_Frame_Storage_Invariants(value Range_Frame_Storage, namespace aver.Namespace) {
	Range_Frame_State_Stored_Invariants(Range_Frame_State_Stored(value.Value), namespace)
}

// Range_Frame_Pointer keeps mutation of one range frame typed.
type Range_Frame_Pointer *Range_Frame_Storage

// Range_Frame_Pointer_Invariants composes one present range frame.
func Range_Frame_Pointer_Invariants(value Range_Frame_Pointer, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Range_Frame_Storage_Invariants(*value, namespace)
}

// Command_Evaluation_References holds the next command and no argument yet.
type Command_Evaluation_References Evaluation_References

// Command_Evaluation_References_Invariants bounds the command and fixes the argument.
func Command_Evaluation_References_Invariants(
	value Command_Evaluation_References, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Holed_Uint16(
			uint16(value.Command), uint16(NODE_REFERENCE_MINIMUM),
			uint16(COMMAND_REFERENCE_MAXIMUM), uint16(ROOT_LIST_HOLE),
			uint16(TERM_ACTION_HOLE), uint16(TERM_ACTION_HOLE)).
		Ensure()
	aver.Always(uint16(value.Term) == uint16(NO_NODE),
		"A frame at its next command has no argument to evaluate.")
}

// Command_Evaluation_Control is the state machine of a frame at its next command.
type Command_Evaluation_Control Evaluation_Control

// Command_Evaluation_Control_Invariants fixes the stage and the pending field path.
func Command_Evaluation_Control_Invariants(
	value Command_Evaluation_Control, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value.Mode), uint8(EVALUATION_MODE_VALUE),
			uint8(EVALUATION_MODE_IDENTIFIER)).
		Enum_Uint8(uint8(value.Present), uint8(EVALUATION_CONTROL_FALSE),
			uint8(EVALUATION_CONTROL_TRUE)).
		Ensure()
	aver.Always(uint8(value.Stage) == uint8(EVALUATION_STAGE_COMMAND),
		"A frame at its next command stays at the command stage.")
	aver.Always(uint8(value.Chain) == uint8(EVALUATION_CONTROL_FALSE),
		"A frame at its next command holds no pending field path.")
}

// Command_Evaluation_Frame is one frame stopped at its next command.
type Command_Evaluation_Frame Evaluation_Frame

// Command_Evaluation_Frame_Invariants composes that frame's storage.
func Command_Evaluation_Frame_Invariants(value Command_Evaluation_Frame, namespace aver.Namespace,
) {
	Evaluation_Nodes_Invariants(value.Nodes, namespace)
	Evaluation_Values_Invariants(value.Values, namespace)
	Command_Evaluation_References_Invariants(
		Command_Evaluation_References(value.References), namespace)
	Evaluation_States_Invariants(value.States, namespace)
	Evaluation_Argument_Bases_Stored_Invariants(
		Evaluation_Argument_Bases_Stored(value.Argument_Bases), namespace)
	Command_Evaluation_Control_Invariants(
		Command_Evaluation_Control(value.Control), namespace)
}

// Command_Evaluation_Frame_Pointer keeps mutation of that frame typed.
type Command_Evaluation_Frame_Pointer *Command_Evaluation_Frame

// Command_Evaluation_Frame_Pointer_Invariants composes one present frame.
func Command_Evaluation_Frame_Pointer_Invariants(
	value Command_Evaluation_Frame_Pointer, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Command_Evaluation_Frame_Invariants(*value, namespace)
}

// Call_Evaluation_References holds no argument for one frame stopped before its call.
type Call_Evaluation_References Evaluation_References

// Call_Evaluation_References_Invariants bounds the command and fixes the argument.
func Call_Evaluation_References_Invariants(
	value Call_Evaluation_References, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Holed_Uint16(
			uint16(value.Command), uint16(NODE_REFERENCE_MINIMUM),
			uint16(COMMAND_TAIL_REFERENCE_MAXIMUM),
			uint16(ROOT_LIST_HOLE), uint16(TERM_ACTION_HOLE),
			uint16(TERM_ACTION_HOLE)).
		Ensure()
	aver.Always(uint16(value.Term) == uint16(NO_NODE),
		"One frame stopped before its call keeps no argument reference.")
}

// Call_Evaluation_Control is the state machine of one frame stopped before its call.
type Call_Evaluation_Control Evaluation_Control

// Call_Evaluation_Control_Invariants fixes the stage that frame holds.
func Call_Evaluation_Control_Invariants(value Call_Evaluation_Control, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value.Mode), uint8(EVALUATION_MODE_VALUE),
			uint8(EVALUATION_MODE_IDENTIFIER)).
		Enum_Uint8(uint8(value.Present), uint8(EVALUATION_CONTROL_FALSE),
			uint8(EVALUATION_CONTROL_TRUE)).
		Ensure()
	aver.Always(uint8(value.Chain) == uint8(EVALUATION_CONTROL_FALSE),
		"One frame stopped before its call carries no chain.")
	aver.Always(uint8(value.Stage) == uint8(EVALUATION_STAGE_CALL),
		"One frame stopped before its call stays at its own stage.")
}

// Call_Evaluation_Frame is one frame stopped before its call.
type Call_Evaluation_Frame Evaluation_Frame

// Call_Evaluation_Frame_Invariants composes that frame's storage.
func Call_Evaluation_Frame_Invariants(value Call_Evaluation_Frame, namespace aver.Namespace,
) {
	Evaluation_Nodes_Invariants(value.Nodes, namespace)
	Evaluation_Values_Invariants(value.Values, namespace)
	Call_Evaluation_References_Invariants(
		Call_Evaluation_References(value.References), namespace)
	Evaluation_States_Invariants(value.States, namespace)
	Evaluation_Argument_Bases_Stored_Invariants(
		Evaluation_Argument_Bases_Stored(value.Argument_Bases), namespace)
	Call_Evaluation_Control_Invariants(Call_Evaluation_Control(value.Control), namespace)
}

// Call_Evaluation_Frame_Pointer keeps mutation of that frame typed.
type Call_Evaluation_Frame_Pointer *Call_Evaluation_Frame

// Call_Evaluation_Frame_Pointer_Invariants composes one present frame.
func Call_Evaluation_Frame_Pointer_Invariants(
	value Call_Evaluation_Frame_Pointer, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Call_Evaluation_Frame_Invariants(*value, namespace)
}

// Done_Evaluation_References holds no argument for one frame past its last command.
type Done_Evaluation_References Evaluation_References

// Done_Evaluation_References_Invariants bounds the command and fixes the argument.
func Done_Evaluation_References_Invariants(
	value Done_Evaluation_References, namespace aver.Namespace,
) {
	aver.Always(uint16(value.Command) == uint16(NO_NODE),
		"One frame past its last command keeps no command reference.")
	aver.Always(uint16(value.Term) == uint16(NO_NODE),
		"One frame past its last command keeps no argument reference.")
}

// Done_Evaluation_Control is the state machine of one frame past its last command.
type Done_Evaluation_Control Evaluation_Control

// Done_Evaluation_Control_Invariants fixes the stage that frame holds.
func Done_Evaluation_Control_Invariants(value Done_Evaluation_Control, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value.Mode), uint8(EVALUATION_MODE_VALUE),
			uint8(EVALUATION_MODE_IDENTIFIER)).
		Enum_Uint8(uint8(value.Present), uint8(EVALUATION_CONTROL_FALSE),
			uint8(EVALUATION_CONTROL_TRUE)).
		Ensure()
	aver.Always(uint8(value.Chain) == uint8(EVALUATION_CONTROL_FALSE),
		"One frame past its last command carries no chain.")
	aver.Always(uint8(value.Stage) == uint8(EVALUATION_STAGE_DONE),
		"One frame past its last command stays at its own stage.")
}

// Done_Evaluation_Frame is one frame past its last command.
type Done_Evaluation_Frame Evaluation_Frame

// Done_Evaluation_Frame_Invariants composes that frame's storage.
func Done_Evaluation_Frame_Invariants(value Done_Evaluation_Frame, namespace aver.Namespace,
) {
	Evaluation_Nodes_Invariants(value.Nodes, namespace)
	Evaluation_Values_Invariants(value.Values, namespace)
	Done_Evaluation_References_Invariants(
		Done_Evaluation_References(value.References), namespace)
	Evaluation_States_Invariants(value.States, namespace)
	Evaluation_Argument_Bases_Stored_Invariants(
		Evaluation_Argument_Bases_Stored(value.Argument_Bases), namespace)
	Done_Evaluation_Control_Invariants(Done_Evaluation_Control(value.Control), namespace)
}

// Done_Evaluation_Frame_Pointer keeps mutation of that frame typed.
type Done_Evaluation_Frame_Pointer *Done_Evaluation_Frame

// Done_Evaluation_Frame_Pointer_Invariants composes one present frame.
func Done_Evaluation_Frame_Pointer_Invariants(
	value Done_Evaluation_Frame_Pointer, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Done_Evaluation_Frame_Invariants(*value, namespace)
}

// Argument_Evaluation_References holds the next command and the next argument.
type Argument_Evaluation_References Evaluation_References

// Argument_Evaluation_References_Invariants bounds both references.
func Argument_Evaluation_References_Invariants(
	value Argument_Evaluation_References, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Holed_Uint16(uint16(value.Command), uint16(NODE_REFERENCE_MINIMUM),
			uint16(COMMAND_TAIL_REFERENCE_MAXIMUM), uint16(ROOT_LIST_HOLE),
			uint16(TERM_ACTION_HOLE), uint16(TERM_PIPELINE_HOLE)).
		Range_Holed_Uint16(uint16(value.Term), uint16(NODE_REFERENCE_MINIMUM),
			uint16(CONTAINER_TAIL_REFERENCE_MAXIMUM), uint16(ROOT_LIST_HOLE),
			uint16(TERM_ACTION_HOLE), uint16(TERM_PIPELINE_HOLE)).
		Ensure()
}

// Argument_Evaluation_Control is the state machine of one frame taking one argument.
type Argument_Evaluation_Control Evaluation_Control

// Argument_Evaluation_Control_Invariants fixes the stage that frame holds.
func Argument_Evaluation_Control_Invariants(
	value Argument_Evaluation_Control, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value.Mode), uint8(EVALUATION_MODE_VALUE),
			uint8(EVALUATION_MODE_IDENTIFIER)).
		Enum_Uint8(uint8(value.Present), uint8(EVALUATION_CONTROL_FALSE),
			uint8(EVALUATION_CONTROL_TRUE)).
		Enum_Uint8(uint8(value.Stage), uint8(EVALUATION_STAGE_FIRST),
			uint8(EVALUATION_STAGE_ARGUMENT)).
		Ensure()
	aver.Always(uint8(value.Chain) == uint8(EVALUATION_CONTROL_FALSE),
		"One frame taking an argument holds no chain of its own.")
}

// Delivery_Evaluation_Control is the state machine one frame taking a child result holds.
type Delivery_Evaluation_Control Evaluation_Control

// Delivery_Evaluation_Control_Invariants fixes the stage that frame holds.
func Delivery_Evaluation_Control_Invariants(
	value Delivery_Evaluation_Control, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value.Mode), uint8(EVALUATION_MODE_VALUE),
			uint8(EVALUATION_MODE_IDENTIFIER)).
		Enum_Uint8(uint8(value.Present), uint8(EVALUATION_CONTROL_FALSE),
			uint8(EVALUATION_CONTROL_TRUE)).
		Enum_Uint8(uint8(value.Chain), uint8(EVALUATION_CONTROL_FALSE),
			uint8(EVALUATION_CONTROL_TRUE)).
		Enum_Uint8(uint8(value.Stage), uint8(EVALUATION_STAGE_FIRST),
			uint8(EVALUATION_STAGE_ARGUMENT)).
		Ensure()
}

// Delivery_Evaluation_Frame is one frame taking one child result.
type Delivery_Evaluation_Frame Evaluation_Frame

// Delivery_Evaluation_Frame_Invariants composes that frame's storage.
func Delivery_Evaluation_Frame_Invariants(
	value Delivery_Evaluation_Frame, namespace aver.Namespace,
) {
	Delivered_Evaluation_Nodes_Invariants(
		Delivered_Evaluation_Nodes(value.Nodes), namespace)
	Evaluation_Values_Invariants(value.Values, namespace)
	Delivered_Evaluation_References_Invariants(
		Delivered_Evaluation_References(value.References), namespace)
	Delivered_Evaluation_States_Invariants(
		Delivered_Evaluation_States(value.States), namespace)
	Evaluation_Argument_Bases_Stored_Invariants(
		Evaluation_Argument_Bases_Stored(value.Argument_Bases), namespace)
	Delivery_Evaluation_Control_Invariants(
		Delivery_Evaluation_Control(value.Control), namespace)
}

// Short_Circuit_Control is the state machine one frame past its argument holds.
type Short_Circuit_Control Evaluation_Control

// Short_Circuit_Control_Invariants fixes the stage that frame reached.
func Short_Circuit_Control_Invariants(
	value Short_Circuit_Control, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value.Mode), uint8(EVALUATION_MODE_VALUE),
			uint8(EVALUATION_MODE_IDENTIFIER)).
		Enum_Uint8(uint8(value.Present), uint8(EVALUATION_CONTROL_FALSE),
			uint8(EVALUATION_CONTROL_TRUE)).
		Ensure()
	aver.Always(uint8(value.Chain) == uint8(EVALUATION_CONTROL_FALSE),
		"One frame past its argument holds no chain of its own.")
	aver.Always(uint8(value.Stage) == uint8(EVALUATION_STAGE_ARGUMENT),
		"One frame past its argument stands at the argument stage.")
}

// Short_Circuit_Frame is one frame that has taken its argument.
type Short_Circuit_Frame Evaluation_Frame

// Short_Circuit_Frame_Invariants composes that frame's storage.
func Short_Circuit_Frame_Invariants(value Short_Circuit_Frame, namespace aver.Namespace) {
	Called_Evaluation_Nodes_Invariants(Called_Evaluation_Nodes(value.Nodes), namespace)
	Evaluation_Values_Invariants(value.Values, namespace)
	Argument_Evaluation_References_Invariants(
		Argument_Evaluation_References(value.References), namespace)
	Delivered_Evaluation_States_Invariants(
		Delivered_Evaluation_States(value.States), namespace)
	Evaluation_Argument_Bases_Stored_Invariants(
		Evaluation_Argument_Bases_Stored(value.Argument_Bases), namespace)
	Short_Circuit_Control_Invariants(Short_Circuit_Control(value.Control), namespace)
}

// Short_Circuit_Frame_Pointer keeps mutation of that frame typed.
type Short_Circuit_Frame_Pointer *Short_Circuit_Frame

// Short_Circuit_Frame_Pointer_Invariants composes one present frame.
func Short_Circuit_Frame_Pointer_Invariants(
	value Short_Circuit_Frame_Pointer, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Short_Circuit_Frame_Invariants(*value, namespace)
}

// Argument_Evaluation_Frame is one frame taking one argument.
type Argument_Evaluation_Frame Evaluation_Frame

// Argument_Evaluation_Frame_Invariants composes that frame's storage.
func Argument_Evaluation_Frame_Invariants(
	value Argument_Evaluation_Frame, namespace aver.Namespace,
) {
	Called_Evaluation_Nodes_Invariants(Called_Evaluation_Nodes(value.Nodes), namespace)
	Evaluation_Values_Invariants(value.Values, namespace)
	Argument_Evaluation_References_Invariants(
		Argument_Evaluation_References(value.References), namespace)
	Delivered_Evaluation_States_Invariants(
		Delivered_Evaluation_States(value.States), namespace)
	Evaluation_Argument_Bases_Stored_Invariants(
		Evaluation_Argument_Bases_Stored(value.Argument_Bases), namespace)
	Argument_Evaluation_Control_Invariants(
		Argument_Evaluation_Control(value.Control), namespace)
}

// Argument_Evaluation_Frame_Pointer keeps mutation of that frame typed.
type Argument_Evaluation_Frame_Pointer *Argument_Evaluation_Frame

// Argument_Evaluation_Frame_Pointer_Invariants composes one present frame.
func Argument_Evaluation_Frame_Pointer_Invariants(
	value Argument_Evaluation_Frame_Pointer, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Argument_Evaluation_Frame_Invariants(*value, namespace)
}

// Child_Delivery_Frame is one frame taking one parenthesized child result.
type Child_Delivery_Frame Evaluation_Frame

// Child_Delivery_Frame_Invariants composes that frame's storage.
func Child_Delivery_Frame_Invariants(value Child_Delivery_Frame, namespace aver.Namespace) {
	Delivered_Evaluation_Nodes_Invariants(
		Delivered_Evaluation_Nodes(value.Nodes), namespace)
	Evaluation_Values_Invariants(value.Values, namespace)
	Delivered_Evaluation_References_Invariants(
		Delivered_Evaluation_References(value.References), namespace)
	Child_Delivered_Evaluation_States_Invariants(
		Child_Delivered_Evaluation_States(value.States), namespace)
	Evaluation_Argument_Bases_Stored_Invariants(
		Evaluation_Argument_Bases_Stored(value.Argument_Bases), namespace)
	Delivery_Evaluation_Control_Invariants(
		Delivery_Evaluation_Control(value.Control), namespace)
}
// Argument_Evaluation_Frame_Delivery_Pointer carries one caller frame taking a child result.
type Argument_Evaluation_Frame_Delivery_Pointer *Child_Delivery_Frame

// Argument_Evaluation_Frame_Delivery_Pointer_Invariants composes that frame.
func Argument_Evaluation_Frame_Delivery_Pointer_Invariants(
	value Argument_Evaluation_Frame_Delivery_Pointer, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Child_Delivery_Frame_Invariants(*value, namespace)
}

// Control_Keyword_Token is one retained token holding one control keyword.
type Control_Keyword_Token Token

// Control_Keyword_Token_Invariants fixes the kinds that role admits.
func Control_Keyword_Token_Invariants(value Control_Keyword_Token, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(uint8(value.Kind), uint8(TOKEN_IF), uint8(TOKEN_RANGE),
			uint8(TOKEN_WITH)).
		Range_Uint16(uint16(value.Start), uint16(TOKEN_BOUNDARY_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM-len(KEYWORD_IF))).
		Range_Uint16(uint16(value.End),
			uint16(ACTION_CONTENT_BOUNDARY_MINIMUM+len(KEYWORD_IF)),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Ensure()
}

// Control_Keyword_Token_Storage is that retained token inside its cursor field.
type Control_Keyword_Token_Storage Token_Storage

// Control_Keyword_Token_Storage_Invariants composes the retained token.
func Control_Keyword_Token_Storage_Invariants(value Control_Keyword_Token_Storage,
	namespace aver.Namespace) {
	Control_Keyword_Token_Invariants(Control_Keyword_Token(value.Value), namespace)
}

// Control_Keyword_Cursor is one lexer whose retained token holds one control keyword.
type Control_Keyword_Cursor Token_Cursor

// Control_Keyword_Cursor_Invariants composes lexer state beside the retained token.
func Control_Keyword_Cursor_Invariants(value Control_Keyword_Cursor, namespace aver.Namespace) {
	Control_Keyword_Position_Stored_Invariants(
		Control_Keyword_Position_Stored(value.Position), namespace)
	Control_Keyword_End_Stored_Invariants(Control_Keyword_End_Stored(value.End), namespace)
	Control_Keyword_Token_Storage_Invariants(Control_Keyword_Token_Storage(value.Token),
		namespace)
	Peeked_Look_Ahead_Storage_Invariants(Peeked_Look_Ahead_Storage(value.Looked), namespace)
	Cursor_Number_Stored_Invariants(Cursor_Number_Stored(value.Number), namespace)
	Function_Function_Storage_Invariants(value.Function, namespace)
	Keyword_Last_Start_Invariants(Keyword_Last_Start(value.Last_Start), namespace)
	Cursor_Has_Last_Stored_Invariants(Cursor_Has_Last_Stored(value.Has_Last), namespace)
}

// Control_Keyword_Cursor_Pointer keeps mutation of that lexer typed.
type Control_Keyword_Cursor_Pointer *Control_Keyword_Cursor

// Control_Keyword_Cursor_Pointer_Invariants composes present lexer state.
func Control_Keyword_Cursor_Pointer_Invariants(value Control_Keyword_Cursor_Pointer,
	namespace aver.Namespace) {
	if value == nil {
		return
	}
	Control_Keyword_Cursor_Invariants(*value, namespace)
}

// Branch_Keyword_Token is one retained token holding one branch keyword.
type Branch_Keyword_Token Token

// Branch_Keyword_Token_Invariants fixes the kinds that role admits.
func Branch_Keyword_Token_Invariants(value Branch_Keyword_Token, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value.Kind), uint8(TOKEN_BREAK), uint8(TOKEN_CONTINUE)).
		Range_Uint16(
			uint16(value.Start), uint16(TOKEN_BOUNDARY_MINIMUM),
			uint16(
				ACTION_CONTENT_BOUNDARY_MAXIMUM-
					TEMPLATE_DEFINITION_END_SIZE_MINIMUM)).
		Range_Uint16(uint16(value.End), uint16(BRANCH_KEYWORD_END_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Ensure()
}

// Branch_Keyword_Token_Storage is that retained token inside its cursor field.
type Branch_Keyword_Token_Storage Token_Storage

// Branch_Keyword_Token_Storage_Invariants composes the retained token.
func Branch_Keyword_Token_Storage_Invariants(value Branch_Keyword_Token_Storage,
	namespace aver.Namespace) {
	Branch_Keyword_Token_Invariants(Branch_Keyword_Token(value.Value), namespace)
}

// Branch_Keyword_Cursor is one lexer whose retained token holds one branch keyword.
type Branch_Keyword_Cursor Token_Cursor

// Branch_Keyword_Cursor_Invariants composes lexer state beside the retained token.
func Branch_Keyword_Cursor_Invariants(value Branch_Keyword_Cursor, namespace aver.Namespace) {
	Branch_Keyword_Position_Stored_Invariants(
		Branch_Keyword_Position_Stored(value.Position), namespace)
	Branch_Keyword_End_Stored_Invariants(Branch_Keyword_End_Stored(value.End), namespace)
	Branch_Keyword_Token_Storage_Invariants(Branch_Keyword_Token_Storage(value.Token),
		namespace)
	Peeked_Look_Ahead_Storage_Invariants(Peeked_Look_Ahead_Storage(value.Looked), namespace)
	Cursor_Number_Stored_Invariants(Cursor_Number_Stored(value.Number), namespace)
	Function_Function_Storage_Invariants(value.Function, namespace)
	Branch_Keyword_Last_Start_Invariants(Branch_Keyword_Last_Start(value.Last_Start), namespace)
	Branch_Keyword_Has_Last_Invariants(Branch_Keyword_Has_Last(value.Has_Last), namespace)
}

// Branch_Keyword_Cursor_Pointer keeps mutation of that lexer typed.
type Branch_Keyword_Cursor_Pointer *Branch_Keyword_Cursor

// Branch_Keyword_Cursor_Pointer_Invariants composes present lexer state.
func Branch_Keyword_Cursor_Pointer_Invariants(value Branch_Keyword_Cursor_Pointer,
	namespace aver.Namespace) {
	if value == nil {
		return
	}
	Branch_Keyword_Cursor_Invariants(*value, namespace)
}

// Definition_Keyword_Token is one retained token holding one definition keyword.
type Definition_Keyword_Token Token

// Definition_Keyword_Token_Invariants fixes the kinds that role admits.
func Definition_Keyword_Token_Invariants(value Definition_Keyword_Token, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value.Kind), uint8(TOKEN_BLOCK), uint8(TOKEN_DEFINE)).
		Range_Uint16(uint16(value.Start), uint16(TOKEN_BOUNDARY_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM-len(KEYWORD_BLOCK))).
		Range_Uint16(uint16(value.End),
			uint16(ACTION_CONTENT_BOUNDARY_MINIMUM+len(KEYWORD_BLOCK)),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Ensure()
}

// Definition_Keyword_Token_Storage is that retained token inside its cursor field.
type Definition_Keyword_Token_Storage Token_Storage

// Definition_Keyword_Token_Storage_Invariants composes the retained token.
func Definition_Keyword_Token_Storage_Invariants(value Definition_Keyword_Token_Storage,
	namespace aver.Namespace) {
	Definition_Keyword_Token_Invariants(Definition_Keyword_Token(value.Value), namespace)
}

// Definition_Keyword_Cursor is one lexer whose retained token holds one definition keyword.
type Definition_Keyword_Cursor Token_Cursor

// Definition_Keyword_Cursor_Invariants composes lexer state beside the retained token.
func Definition_Keyword_Cursor_Invariants(value Definition_Keyword_Cursor,
	namespace aver.Namespace) {
	Definition_Keyword_Position_Stored_Invariants(
		Definition_Keyword_Position_Stored(value.Position), namespace)
	Definition_Keyword_End_Stored_Invariants(
		Definition_Keyword_End_Stored(value.End), namespace)
	Definition_Keyword_Token_Storage_Invariants(Definition_Keyword_Token_Storage(value.Token),
		namespace)
	Peeked_Look_Ahead_Storage_Invariants(Peeked_Look_Ahead_Storage(value.Looked), namespace)
	Cursor_Number_Stored_Invariants(Cursor_Number_Stored(value.Number), namespace)
	Function_Function_Storage_Invariants(value.Function, namespace)
	Definition_Keyword_Last_Start_Invariants(Definition_Keyword_Last_Start(value.Last_Start),
		namespace)
	Definition_Keyword_Has_Last_Invariants(Definition_Keyword_Has_Last(value.Has_Last),
		namespace)
}

// Definition_Keyword_Cursor_Pointer keeps mutation of that lexer typed.
type Definition_Keyword_Cursor_Pointer *Definition_Keyword_Cursor

// Definition_Keyword_Cursor_Pointer_Invariants composes present lexer state.
func Definition_Keyword_Cursor_Pointer_Invariants(value Definition_Keyword_Cursor_Pointer,
	namespace aver.Namespace) {
	if value == nil {
		return
	}
	Definition_Keyword_Cursor_Invariants(*value, namespace)
}

// Name_Boundary_Token is one retained token holding one token that closes a template name.
type Name_Boundary_Token Token

// Name_Boundary_Token_Invariants fixes the kinds that role admits.
func Name_Boundary_Token_Invariants(value Name_Boundary_Token, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value.Kind), uint8(TOKEN_RAW_STRING), uint8(TOKEN_STRING)).
		Range_Uint16(
			uint16(value.Start),
			uint16(NAME_BOUNDARY_POSITION_MINIMUM-QUOTED_BOUNDARY_SIZE),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM-ACTION_BOUNDARY_SIZE_MINIMUM)).
		Range_Uint16(uint16(value.End), uint16(NAME_BOUNDARY_END_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Ensure()
}

// Name_Boundary_Token_Storage is that retained token inside its cursor field.
type Name_Boundary_Token_Storage Token_Storage

// Name_Boundary_Token_Storage_Invariants composes the retained token.
func Name_Boundary_Token_Storage_Invariants(value Name_Boundary_Token_Storage,
	namespace aver.Namespace) {
	Name_Boundary_Token_Invariants(Name_Boundary_Token(value.Value), namespace)
}

// Name_Boundary_Cursor is one lexer whose retained token closes a template name.
type Name_Boundary_Cursor Token_Cursor

// Name_Boundary_Cursor_Invariants composes lexer state beside the retained token.
func Name_Boundary_Cursor_Invariants(value Name_Boundary_Cursor, namespace aver.Namespace) {
	Name_Boundary_Position_Stored_Invariants(
		Name_Boundary_Position_Stored(value.Position), namespace)
	Name_Boundary_End_Stored_Invariants(Name_Boundary_End_Stored(value.End), namespace)
	Name_Boundary_Token_Storage_Invariants(Name_Boundary_Token_Storage(value.Token), namespace)
	Unread_Look_Ahead_Storage_Invariants(Unread_Look_Ahead_Storage(value.Looked), namespace)
	Cursor_Number_Stored_Invariants(Cursor_Number_Stored(value.Number), namespace)
	Function_Function_Storage_Invariants(value.Function, namespace)
	Name_Boundary_Last_Start_Invariants(Name_Boundary_Last_Start(value.Last_Start), namespace)
	Read_Has_Last_Stored_Invariants(Read_Has_Last_Stored(value.Has_Last), namespace)
}

// Name_Boundary_Cursor_Pointer keeps mutation of that lexer typed.
type Name_Boundary_Cursor_Pointer *Name_Boundary_Cursor

// Name_Boundary_Cursor_Pointer_Invariants composes present lexer state.
func Name_Boundary_Cursor_Pointer_Invariants(value Name_Boundary_Cursor_Pointer,
	namespace aver.Namespace) {
	if value == nil {
		return
	}
	Name_Boundary_Cursor_Invariants(*value, namespace)
}

// Chain_Term_Token is one retained token holding one term a chain may follow.
type Chain_Term_Token Token

// Chain_Term_Token_Invariants fixes the kinds that role admits.
func Chain_Term_Token_Invariants(value Chain_Term_Token, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value.Kind), uint8(TOKEN_IDENTIFIER), uint8(TOKEN_RIGHT_PAREN)).
		Range_Uint16(
			uint16(value.Start), uint16(TOKEN_BOUNDARY_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM-utf8.CHARACTER_SIZE_MINIMUM)).
		Range_Uint16(uint16(value.End),
			uint16(TOKEN_BOUNDARY_MINIMUM+utf8.CHARACTER_SIZE_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Ensure()
}

// Chain_Term_Token_Storage is that retained token inside its cursor field.
type Chain_Term_Token_Storage Token_Storage

// Chain_Term_Token_Storage_Invariants composes the retained token.
func Chain_Term_Token_Storage_Invariants(value Chain_Term_Token_Storage, namespace aver.Namespace) {
	Chain_Term_Token_Invariants(Chain_Term_Token(value.Value), namespace)
}

// Chain_Term_Cursor is one lexer whose retained token holds one term a chain may follow.
type Chain_Term_Cursor Token_Cursor

// Chain_Term_Cursor_Invariants composes lexer state beside the retained token.
func Chain_Term_Cursor_Invariants(value Chain_Term_Cursor, namespace aver.Namespace) {
	Chain_Term_Position_Stored_Invariants(
		Chain_Term_Position_Stored(value.Position), namespace)
	Chain_Term_End_Stored_Invariants(Chain_Term_End_Stored(value.End), namespace)
	Chain_Term_Token_Storage_Invariants(Chain_Term_Token_Storage(value.Token), namespace)
	Unread_Look_Ahead_Storage_Invariants(Unread_Look_Ahead_Storage(value.Looked), namespace)
	Cursor_Number_Stored_Invariants(Cursor_Number_Stored(value.Number), namespace)
	Function_Function_Storage_Invariants(value.Function, namespace)
	Chain_Term_Last_Start_Invariants(Chain_Term_Last_Start(value.Last_Start), namespace)
	Read_Has_Last_Stored_Invariants(Read_Has_Last_Stored(value.Has_Last), namespace)
}

// Chain_Term_Cursor_Pointer keeps mutation of that lexer typed.
type Chain_Term_Cursor_Pointer *Chain_Term_Cursor

// Chain_Term_Cursor_Pointer_Invariants composes present lexer state.
func Chain_Term_Cursor_Pointer_Invariants(value Chain_Term_Cursor_Pointer,
	namespace aver.Namespace) {
	if value == nil {
		return
	}
	Chain_Term_Cursor_Invariants(*value, namespace)
}

// Pipeline_Term_Token is one retained token holding one term a pipeline admits.
type Pipeline_Term_Token Token

// Pipeline_Term_Token_Invariants fixes the kinds that role admits.
func Pipeline_Term_Token_Invariants(value Pipeline_Term_Token, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value.Kind), uint8(TOKEN_ASSIGN), uint8(TOKEN_VARIABLE)).
		Range_Uint16(uint16(value.Start), uint16(TOKEN_BOUNDARY_MINIMUM),
			uint16(LEXEME_START_MAXIMUM)).
		Range_Uint16(uint16(value.End),
			uint16(TOKEN_BOUNDARY_MINIMUM+utf8.CHARACTER_SIZE_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Ensure()
}

// Pipeline_Term_Token_Storage is that retained token inside its cursor field.
type Pipeline_Term_Token_Storage Token_Storage

// Pipeline_Term_Token_Storage_Invariants composes the retained token.
func Pipeline_Term_Token_Storage_Invariants(value Pipeline_Term_Token_Storage,
	namespace aver.Namespace) {
	Pipeline_Term_Token_Invariants(Pipeline_Term_Token(value.Value), namespace)
}

// Pipeline_Term_Cursor is one lexer whose retained token holds one term a pipeline admits.
type Pipeline_Term_Cursor Token_Cursor

// Pipeline_Term_Cursor_Invariants composes lexer state beside the retained token.
func Pipeline_Term_Cursor_Invariants(value Pipeline_Term_Cursor, namespace aver.Namespace) {
	Pipeline_Term_Position_Stored_Invariants(
		Pipeline_Term_Position_Stored(value.Position), namespace)
	Pipeline_Term_End_Stored_Invariants(Pipeline_Term_End_Stored(value.End), namespace)
	Pipeline_Term_Token_Storage_Invariants(Pipeline_Term_Token_Storage(value.Token), namespace)
	Peeked_Look_Ahead_Storage_Invariants(Peeked_Look_Ahead_Storage(value.Looked), namespace)
	Cursor_Number_Stored_Invariants(Cursor_Number_Stored(value.Number), namespace)
	Function_Function_Storage_Invariants(value.Function, namespace)
	Pipeline_Term_Last_Start_Invariants(
		Pipeline_Term_Last_Start(value.Last_Start), namespace)
	Cursor_Has_Last_Stored_Invariants(Cursor_Has_Last_Stored(value.Has_Last), namespace)
}

// Pipeline_Term_Cursor_Pointer keeps mutation of that lexer typed.
type Pipeline_Term_Cursor_Pointer *Pipeline_Term_Cursor

// Pipeline_Term_Cursor_Pointer_Invariants composes present lexer state.
func Pipeline_Term_Cursor_Pointer_Invariants(value Pipeline_Term_Cursor_Pointer,
	namespace aver.Namespace) {
	if value == nil {
		return
	}
	Pipeline_Term_Cursor_Invariants(*value, namespace)
}

// Action_Term_Token is one retained token holding one term or the action end.
type Action_Term_Token Token

// Action_Term_Token_Invariants fixes the kinds that role admits.
func Action_Term_Token_Invariants(value Action_Term_Token, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value.Kind), uint8(TOKEN_EOF), uint8(TOKEN_VARIABLE)).
		Range_Uint16(uint16(value.Start), uint16(TOKEN_BOUNDARY_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Range_Uint16(uint16(value.End), uint16(TOKEN_BOUNDARY_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Ensure()
}

// Action_Term_Token_Storage is that retained token inside its cursor field.
type Action_Term_Token_Storage Token_Storage

// Action_Term_Token_Storage_Invariants composes the retained token.
func Action_Term_Token_Storage_Invariants(value Action_Term_Token_Storage,
	namespace aver.Namespace) {
	Action_Term_Token_Invariants(Action_Term_Token(value.Value), namespace)
}

// Action_Term_Cursor is one lexer whose retained token holds one term or the action end.
type Action_Term_Cursor Token_Cursor

// Action_Term_Cursor_Invariants composes lexer state beside the retained token.
func Action_Term_Cursor_Invariants(value Action_Term_Cursor, namespace aver.Namespace) {
	Cursor_Position_Stored_Invariants(Cursor_Position_Stored(value.Position), namespace)
	Cursor_End_Stored_Invariants(Cursor_End_Stored(value.End), namespace)
	Action_Term_Token_Storage_Invariants(Action_Term_Token_Storage(value.Token), namespace)
	Peeked_Look_Ahead_Storage_Invariants(Peeked_Look_Ahead_Storage(value.Looked), namespace)
	Cursor_Number_Stored_Invariants(Cursor_Number_Stored(value.Number), namespace)
	Function_Function_Storage_Invariants(value.Function, namespace)
	Action_Term_Last_Start_Invariants(Action_Term_Last_Start(value.Last_Start), namespace)
	Action_Term_Has_Last_Invariants(Action_Term_Has_Last(value.Has_Last), namespace)
}

// Action_Term_Cursor_Pointer keeps mutation of that lexer typed.
type Action_Term_Cursor_Pointer *Action_Term_Cursor

// Action_Term_Cursor_Pointer_Invariants composes present lexer state.
func Action_Term_Cursor_Pointer_Invariants(value Action_Term_Cursor_Pointer,
	namespace aver.Namespace) {
	if value == nil {
		return
	}
	Action_Term_Cursor_Invariants(*value, namespace)
}

// Control_Term_Token is one retained token holding one control keyword or its argument.
type Control_Term_Token Token

// Control_Term_Token_Invariants fixes the kinds that role admits.
func Control_Term_Token_Invariants(value Control_Term_Token, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value.Kind), uint8(TOKEN_BREAK), uint8(TOKEN_WITH)).
		Range_Uint16(
			uint16(value.Start), uint16(TOKEN_BOUNDARY_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM-ACTION_BOUNDARY_SIZE_MINIMUM)).
		Range_Uint16(uint16(value.End), uint16(CONTROL_KEYWORD_END_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Ensure()
}

// Control_Term_Token_Storage is that retained token inside its cursor field.
type Control_Term_Token_Storage Token_Storage

// Control_Term_Token_Storage_Invariants composes the retained token.
func Control_Term_Token_Storage_Invariants(value Control_Term_Token_Storage,
	namespace aver.Namespace) {
	Control_Term_Token_Invariants(Control_Term_Token(value.Value), namespace)
}

// Control_Term_Cursor is one lexer whose retained token holds one control keyword or its argument.
type Control_Term_Cursor Token_Cursor

// Control_Term_Cursor_Invariants composes lexer state beside the retained token.
func Control_Term_Cursor_Invariants(value Control_Term_Cursor, namespace aver.Namespace) {
	Control_Term_Position_Stored_Invariants(
		Control_Term_Position_Stored(value.Position), namespace)
	Control_Term_End_Stored_Invariants(Control_Term_End_Stored(value.End), namespace)
	Control_Term_Token_Storage_Invariants(Control_Term_Token_Storage(value.Token), namespace)
	Peeked_Look_Ahead_Storage_Invariants(Peeked_Look_Ahead_Storage(value.Looked), namespace)
	Cursor_Number_Stored_Invariants(Cursor_Number_Stored(value.Number), namespace)
	Function_Function_Storage_Invariants(value.Function, namespace)
	Control_Term_Last_Start_Invariants(Control_Term_Last_Start(value.Last_Start), namespace)
	Control_Term_Has_Last_Invariants(Control_Term_Has_Last(value.Has_Last), namespace)
}

// Control_Term_Cursor_Pointer keeps mutation of that lexer typed.
type Control_Term_Cursor_Pointer *Control_Term_Cursor

// Control_Term_Cursor_Pointer_Invariants composes present lexer state.
func Control_Term_Cursor_Pointer_Invariants(value Control_Term_Cursor_Pointer,
	namespace aver.Namespace) {
	if value == nil {
		return
	}
	Control_Term_Cursor_Invariants(*value, namespace)
}

// DEFINITION_NAME_START_MINIMUM follows the keyword one definition name stands behind.
const DEFINITION_NAME_START_MINIMUM = CUSTOM_DELIMITER_SIZE_MINIMUM + len(KEYWORD_DEFINE) +
	ACTION_SEPARATOR_SIZE_MINIMUM

// Definition_Boundary_Token is one retained token that closes a definition header.
type Definition_Boundary_Token Token

// Definition_Boundary_Token_Invariants fixes the kinds that role admits.
func Definition_Boundary_Token_Invariants(value Definition_Boundary_Token,
	namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(uint8(value.Kind), uint8(TOKEN_EOF), uint8(TOKEN_RAW_STRING),
			uint8(TOKEN_STRING)).
		Range_Uint16(uint16(value.Start), uint16(DEFINITION_NAME_START_MINIMUM),
			uint16(OWED_END_BOUNDARY_MAXIMUM)).
		Range_Uint16(uint16(value.End),
			uint16(DEFINITION_NAME_START_MINIMUM+QUOTED_BOUNDARY_SIZE),
			uint16(OWED_END_BOUNDARY_MAXIMUM)).
		Ensure()
}

// Definition_Boundary_Token_Storage is that retained token inside its cursor field.
type Definition_Boundary_Token_Storage Token_Storage

// Definition_Boundary_Token_Storage_Invariants composes the retained token.
func Definition_Boundary_Token_Storage_Invariants(value Definition_Boundary_Token_Storage,
	namespace aver.Namespace) {
	Definition_Boundary_Token_Invariants(Definition_Boundary_Token(value.Value), namespace)
}

// Definition_Boundary_Cursor is one lexer whose retained token closes a definition header.
type Definition_Boundary_Cursor Token_Cursor

// Definition_Boundary_Cursor_Invariants composes lexer state beside the retained token.
func Definition_Boundary_Cursor_Invariants(value Definition_Boundary_Cursor,
	namespace aver.Namespace) {
	Definition_Boundary_Position_Stored_Invariants(
		Definition_Boundary_Position_Stored(value.Position), namespace)
	Definition_Boundary_End_Stored_Invariants(
		Definition_Boundary_End_Stored(value.End), namespace)
	Definition_Boundary_Token_Storage_Invariants(Definition_Boundary_Token_Storage(value.Token),
		namespace)
	Unread_Look_Ahead_Storage_Invariants(Unread_Look_Ahead_Storage(value.Looked), namespace)
	Cursor_Number_Stored_Invariants(Cursor_Number_Stored(value.Number), namespace)
	Function_Function_Storage_Invariants(value.Function, namespace)
	Definition_Boundary_Last_Start_Invariants(Definition_Boundary_Last_Start(value.Last_Start),
		namespace)
	Read_Has_Last_Stored_Invariants(Read_Has_Last_Stored(value.Has_Last), namespace)
}

// Definition_Boundary_Cursor_Pointer keeps mutation of that lexer typed.
type Definition_Boundary_Cursor_Pointer *Definition_Boundary_Cursor

// Definition_Boundary_Cursor_Pointer_Invariants composes present lexer state.
func Definition_Boundary_Cursor_Pointer_Invariants(value Definition_Boundary_Cursor_Pointer,
	namespace aver.Namespace) {
	if value == nil {
		return
	}
	Definition_Boundary_Cursor_Invariants(*value, namespace)
}

// COMMENT_POSITION_MINIMUM bounds the read offset inside one comment action.
const COMMENT_POSITION_MINIMUM = CUSTOM_DELIMITER_SIZE_MINIMUM + len(`/**/`)

// Comment_Position_Stored is the read offset inside one comment action.
type Comment_Position_Stored Cursor_Position_Storage

// Comment_Position_Stored_Invariants fixes that offset against its clause.
func Comment_Position_Stored_Invariants(value Comment_Position_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(COMMENT_POSITION_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Ensure()
}

// COMMENT_END_MINIMUM bounds the parse boundary of one comment action.
const COMMENT_END_MINIMUM = CUSTOM_DELIMITER_SIZE_MINIMUM + len(`/**/`)

// Comment_End_Stored is the parse boundary of one comment action.
type Comment_End_Stored Cursor_End_Storage

// Comment_End_Stored_Invariants fixes that boundary against its clause.
func Comment_End_Stored_Invariants(value Comment_End_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(COMMENT_END_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Ensure()
}

// INVOCATION_POSITION_MINIMUM bounds the read offset inside one template invocation.
const INVOCATION_POSITION_MINIMUM = CUSTOM_DELIMITER_SIZE_MINIMUM + len(KEYWORD_TEMPLATE)

// Invocation_Position_Stored is the read offset inside one template invocation.
type Invocation_Position_Stored Cursor_Position_Storage

// Invocation_Position_Stored_Invariants fixes that offset against its clause.
func Invocation_Position_Stored_Invariants(
	value Invocation_Position_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(INVOCATION_POSITION_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Ensure()
}

// INVOCATION_END_MINIMUM bounds the parse boundary of one template invocation.
const INVOCATION_END_MINIMUM = CUSTOM_DELIMITER_SIZE_MINIMUM + len(KEYWORD_TEMPLATE)

// Invocation_End_Stored is the parse boundary of one template invocation.
type Invocation_End_Stored Cursor_End_Storage

// Invocation_End_Stored_Invariants fixes that boundary against its clause.
func Invocation_End_Stored_Invariants(value Invocation_End_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(INVOCATION_END_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Ensure()
}

// ELSE_CLAUSE_START_MINIMUM opens the alternate arm one branch already holds.
const ELSE_CLAUSE_START_MINIMUM = ACTION_BOUNDARY_SIZE_MINIMUM + len(KEYWORD_IF) +
	ACTION_SEPARATOR_SIZE_MINIMUM + DOT_SOURCE_SIZE + CUSTOM_DELIMITER_SIZE_MINIMUM

// ELSE_CLAUSE_END_MINIMUM closes that alternate keyword.
const ELSE_CLAUSE_END_MINIMUM = ELSE_CLAUSE_START_MINIMUM + len(KEYWORD_ELSE)

// ELSE_POSITION_MINIMUM bounds the read offset inside one else clause.
const ELSE_POSITION_MINIMUM = CUSTOM_DELIMITER_SIZE_MINIMUM + len(KEYWORD_ELSE)

// Else_Position_Stored is the read offset inside one else clause.
type Else_Position_Stored Cursor_Position_Storage

// Else_Position_Stored_Invariants fixes that offset against its clause.
func Else_Position_Stored_Invariants(value Else_Position_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(ELSE_POSITION_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Ensure()
}

// ELSE_END_MINIMUM bounds the parse boundary of one else clause.
const ELSE_END_MINIMUM = CUSTOM_DELIMITER_SIZE_MINIMUM + len(KEYWORD_ELSE)

// Else_End_Stored is the parse boundary of one else clause.
type Else_End_Stored Cursor_End_Storage

// Else_End_Stored_Invariants fixes that boundary against its clause.
func Else_End_Stored_Invariants(value Else_End_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(ELSE_END_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Ensure()
}

// END_POSITION_MINIMUM bounds the read offset inside one end clause.
const END_POSITION_MINIMUM = CUSTOM_DELIMITER_SIZE_MINIMUM + len(KEYWORD_END)

// End_Position_Stored is the read offset inside one end clause.
type End_Position_Stored Cursor_Position_Storage

// End_Position_Stored_Invariants fixes that offset against its clause.
func End_Position_Stored_Invariants(value End_Position_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(END_POSITION_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Ensure()
}

// END_END_MINIMUM bounds the parse boundary of one end clause.
const END_END_MINIMUM = CUSTOM_DELIMITER_SIZE_MINIMUM + len(KEYWORD_END)

// End_End_Stored is the parse boundary of one end clause.
type End_End_Stored Cursor_End_Storage

// End_End_Stored_Invariants fixes that boundary against its clause.
func End_End_Stored_Invariants(value End_End_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(END_END_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Ensure()
}

// BRANCH_KEYWORD_POSITION_MINIMUM bounds the read offset inside one branch clause.
const BRANCH_KEYWORD_POSITION_MINIMUM = CUSTOM_DELIMITER_SIZE_MINIMUM + len(KEYWORD_BREAK)

// Branch_Keyword_Position_Stored is the read offset inside one branch clause.
type Branch_Keyword_Position_Stored Cursor_Position_Storage

// Branch_Keyword_Position_Stored_Invariants fixes that offset against its clause.
func Branch_Keyword_Position_Stored_Invariants(
	value Branch_Keyword_Position_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(BRANCH_KEYWORD_POSITION_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Ensure()
}

// BRANCH_KEYWORD_END_MINIMUM bounds the parse boundary of one branch clause.
const BRANCH_KEYWORD_END_MINIMUM = CUSTOM_DELIMITER_SIZE_MINIMUM + len(KEYWORD_BREAK)

// Branch_Keyword_End_Stored is the parse boundary of one branch clause.
type Branch_Keyword_End_Stored Cursor_End_Storage

// Branch_Keyword_End_Stored_Invariants fixes that boundary against its clause.
func Branch_Keyword_End_Stored_Invariants(
	value Branch_Keyword_End_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(BRANCH_KEYWORD_END_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Ensure()
}

// CONTROL_KEYWORD_POSITION_MINIMUM bounds the read offset inside one control header.
const CONTROL_KEYWORD_POSITION_MINIMUM = CUSTOM_DELIMITER_SIZE_MINIMUM + len(KEYWORD_IF)

// Control_Keyword_Position_Stored is the read offset inside one control header.
type Control_Keyword_Position_Stored Cursor_Position_Storage

// Control_Keyword_Position_Stored_Invariants fixes that offset against its clause.
func Control_Keyword_Position_Stored_Invariants(
	value Control_Keyword_Position_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(CONTROL_KEYWORD_POSITION_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Ensure()
}

// CONTROL_KEYWORD_END_MINIMUM bounds the parse boundary of one control header.
const CONTROL_KEYWORD_END_MINIMUM = CUSTOM_DELIMITER_SIZE_MINIMUM + len(KEYWORD_IF)

// Control_Keyword_End_Stored is the parse boundary of one control header.
type Control_Keyword_End_Stored Cursor_End_Storage

// Control_Keyword_End_Stored_Invariants fixes that boundary against its clause.
func Control_Keyword_End_Stored_Invariants(
	value Control_Keyword_End_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(CONTROL_KEYWORD_END_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Ensure()
}

// CONTROL_TERM_POSITION_MINIMUM bounds the read offset inside one control action.
const CONTROL_TERM_POSITION_MINIMUM = CUSTOM_DELIMITER_SIZE_MINIMUM + len(KEYWORD_IF)

// Control_Term_Position_Stored is the read offset inside one control action.
type Control_Term_Position_Stored Cursor_Position_Storage

// Control_Term_Position_Stored_Invariants fixes that offset against its clause.
func Control_Term_Position_Stored_Invariants(
	value Control_Term_Position_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(CONTROL_TERM_POSITION_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Ensure()
}

// CONTROL_TERM_END_MINIMUM bounds the parse boundary of one control action.
const CONTROL_TERM_END_MINIMUM = CUSTOM_DELIMITER_SIZE_MINIMUM + len(KEYWORD_IF)

// Control_Term_End_Stored is the parse boundary of one control action.
type Control_Term_End_Stored Cursor_End_Storage

// Control_Term_End_Stored_Invariants fixes that boundary against its clause.
func Control_Term_End_Stored_Invariants(value Control_Term_End_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(CONTROL_TERM_END_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Ensure()
}

// CONTROL_PIPELINE_POSITION_MINIMUM bounds the read offset inside one control pipeline.
const CONTROL_PIPELINE_POSITION_MINIMUM = CUSTOM_DELIMITER_SIZE_MINIMUM + len(KEYWORD_IF)

// Control_Pipeline_Position_Stored is the read offset inside one control pipeline.
type Control_Pipeline_Position_Stored Cursor_Position_Storage

// Control_Pipeline_Position_Stored_Invariants fixes that offset against its clause.
func Control_Pipeline_Position_Stored_Invariants(
	value Control_Pipeline_Position_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(CONTROL_PIPELINE_POSITION_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Ensure()
}

// CONTROL_PIPELINE_END_MINIMUM bounds the parse boundary of one control pipeline.
const CONTROL_PIPELINE_END_MINIMUM = CUSTOM_DELIMITER_SIZE_MINIMUM + len(KEYWORD_IF)

// Control_Pipeline_End_Stored is the parse boundary of one control pipeline.
type Control_Pipeline_End_Stored Cursor_End_Storage

// Control_Pipeline_End_Stored_Invariants fixes that boundary against its clause.
func Control_Pipeline_End_Stored_Invariants(
	value Control_Pipeline_End_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(CONTROL_PIPELINE_END_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Ensure()
}

// PIPELINE_TERM_POSITION_MINIMUM bounds the read offset inside one pipeline term.
const PIPELINE_TERM_POSITION_MINIMUM = CUSTOM_DELIMITER_SIZE_MINIMUM + IDENTIFIER_SIZE_MINIMUM

// Pipeline_Term_Position_Stored is the read offset inside one pipeline term.
type Pipeline_Term_Position_Stored Cursor_Position_Storage

// Pipeline_Term_Position_Stored_Invariants fixes that offset against its clause.
func Pipeline_Term_Position_Stored_Invariants(
	value Pipeline_Term_Position_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(PIPELINE_TERM_POSITION_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Ensure()
}

// PIPELINE_TERM_END_MINIMUM bounds the parse boundary of one pipeline term.
const PIPELINE_TERM_END_MINIMUM = CUSTOM_DELIMITER_SIZE_MINIMUM + IDENTIFIER_SIZE_MINIMUM

// Pipeline_Term_End_Stored is the parse boundary of one pipeline term.
type Pipeline_Term_End_Stored Cursor_End_Storage

// Pipeline_Term_End_Stored_Invariants fixes that boundary against its clause.
func Pipeline_Term_End_Stored_Invariants(value Pipeline_Term_End_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(PIPELINE_TERM_END_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Ensure()
}

// CHAIN_TERM_POSITION_MINIMUM bounds the read offset inside one chained term.
const CHAIN_TERM_POSITION_MINIMUM = CUSTOM_DELIMITER_SIZE_MINIMUM + IDENTIFIER_SIZE_MINIMUM

// Chain_Term_Position_Stored is the read offset inside one chained term.
type Chain_Term_Position_Stored Cursor_Position_Storage

// Chain_Term_Position_Stored_Invariants fixes that offset against its clause.
func Chain_Term_Position_Stored_Invariants(
	value Chain_Term_Position_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(CHAIN_TERM_POSITION_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Ensure()
}

// CHAIN_TERM_END_MINIMUM bounds the parse boundary of one chained term.
const CHAIN_TERM_END_MINIMUM = CUSTOM_DELIMITER_SIZE_MINIMUM + IDENTIFIER_SIZE_MINIMUM

// Chain_Term_End_Stored is the parse boundary of one chained term.
type Chain_Term_End_Stored Cursor_End_Storage

// Chain_Term_End_Stored_Invariants fixes that boundary against its clause.
func Chain_Term_End_Stored_Invariants(value Chain_Term_End_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(CHAIN_TERM_END_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Ensure()
}

// EXHAUSTED_POSITION_MINIMUM bounds the read offset inside one finished pipeline.
const EXHAUSTED_POSITION_MINIMUM = CUSTOM_DELIMITER_SIZE_MINIMUM + IDENTIFIER_SIZE_MINIMUM

// Exhausted_Position_Stored is the read offset inside one finished pipeline.
type Exhausted_Position_Stored Cursor_Position_Storage

// Exhausted_Position_Stored_Invariants fixes that offset against its clause.
func Exhausted_Position_Stored_Invariants(
	value Exhausted_Position_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(EXHAUSTED_POSITION_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Ensure()
}

// EXHAUSTED_END_MINIMUM bounds the parse boundary of one finished pipeline.
const EXHAUSTED_END_MINIMUM = CUSTOM_DELIMITER_SIZE_MINIMUM + IDENTIFIER_SIZE_MINIMUM

// Exhausted_End_Stored is the parse boundary of one finished pipeline.
type Exhausted_End_Stored Cursor_End_Storage

// Exhausted_End_Stored_Invariants fixes that boundary against its clause.
func Exhausted_End_Stored_Invariants(value Exhausted_End_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(EXHAUSTED_END_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Ensure()
}

// CLOSE_PAREN_POSITION_MINIMUM bounds the read offset inside one closed parenthesis.
const CLOSE_PAREN_POSITION_MINIMUM = CUSTOM_DELIMITER_SIZE_MINIMUM + PARENTHESIS_SIZE +
	IDENTIFIER_SIZE_MINIMUM +
	PARENTHESIS_SIZE

// Close_Paren_Position_Stored is the read offset inside one closed parenthesis.
type Close_Paren_Position_Stored Cursor_Position_Storage

// Close_Paren_Position_Stored_Invariants fixes that offset against its clause.
func Close_Paren_Position_Stored_Invariants(
	value Close_Paren_Position_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(CLOSE_PAREN_POSITION_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Ensure()
}

// CLOSE_PAREN_END_MINIMUM bounds the parse boundary of one closed parenthesis.
const CLOSE_PAREN_END_MINIMUM = CUSTOM_DELIMITER_SIZE_MINIMUM + PARENTHESIS_SIZE +
	IDENTIFIER_SIZE_MINIMUM +
	PARENTHESIS_SIZE

// Close_Paren_End_Stored is the parse boundary of one closed parenthesis.
type Close_Paren_End_Stored Cursor_End_Storage

// Close_Paren_End_Stored_Invariants fixes that boundary against its clause.
func Close_Paren_End_Stored_Invariants(value Close_Paren_End_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(CLOSE_PAREN_END_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Ensure()
}

// OPEN_PAREN_POSITION_MINIMUM bounds the read offset inside one open parenthesis.
const OPEN_PAREN_POSITION_MINIMUM = CUSTOM_DELIMITER_SIZE_MINIMUM + PARENTHESIS_SIZE

// Open_Paren_Position_Stored is the read offset inside one open parenthesis.
type Open_Paren_Position_Stored Cursor_Position_Storage

// Open_Paren_Position_Stored_Invariants fixes that offset against its clause.
func Open_Paren_Position_Stored_Invariants(
	value Open_Paren_Position_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value.Value), uint16(OPEN_PAREN_POSITION_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Ensure()
}

// OPEN_PAREN_END_MINIMUM bounds the parse boundary of one open parenthesis.
const OPEN_PAREN_END_MINIMUM = CUSTOM_DELIMITER_SIZE_MINIMUM + PARENTHESIS_SIZE +
	IDENTIFIER_SIZE_MINIMUM

// Open_Paren_End_Stored is the parse boundary of one open parenthesis.
type Open_Paren_End_Stored Cursor_End_Storage

// Open_Paren_End_Stored_Invariants fixes that boundary against its clause.
func Open_Paren_End_Stored_Invariants(value Open_Paren_End_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(OPEN_PAREN_END_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Ensure()
}

// SEPARATOR_POSITION_MINIMUM bounds the read offset inside one pipeline separator.
const SEPARATOR_POSITION_MINIMUM = CUSTOM_DELIMITER_SIZE_MINIMUM + IDENTIFIER_SIZE_MINIMUM +
	PIPELINE_SEPARATOR_SIZE

// Separator_Position_Stored is the read offset inside one pipeline separator.
type Separator_Position_Stored Cursor_Position_Storage

// Separator_Position_Stored_Invariants fixes that offset against its clause.
func Separator_Position_Stored_Invariants(
	value Separator_Position_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value.Value), uint16(SEPARATOR_POSITION_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM-utf8.CHARACTER_SIZE_MINIMUM)).
		Ensure()
}

// SEPARATOR_END_MINIMUM bounds the parse boundary of one pipeline separator.
const SEPARATOR_END_MINIMUM = CUSTOM_DELIMITER_SIZE_MINIMUM + IDENTIFIER_SIZE_MINIMUM +
	PIPELINE_SEPARATOR_SIZE

// Separator_End_Stored is the parse boundary of one pipeline separator.
type Separator_End_Stored Cursor_End_Storage

// Separator_End_Stored_Invariants fixes that boundary against its clause.
func Separator_End_Stored_Invariants(value Separator_End_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(SEPARATOR_END_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Ensure()
}

// BLOCK_POSITION_MINIMUM bounds the read offset inside one block body.
const BLOCK_POSITION_MINIMUM = CUSTOM_DELIMITER_SIZE_MINIMUM + len(KEYWORD_BLOCK) +
	ACTION_SEPARATOR_SIZE_MINIMUM + TEMPLATE_NAME_SOURCE_SIZE_MINIMUM

// Block_Position_Stored is the read offset inside one block body.
type Block_Position_Stored Cursor_Position_Storage

// Block_Position_Stored_Invariants fixes that offset against its clause.
func Block_Position_Stored_Invariants(value Block_Position_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(BLOCK_POSITION_MINIMUM),
			uint16(OWED_END_BOUNDARY_MAXIMUM)).
		Ensure()
}

// BLOCK_END_MINIMUM bounds the parse boundary of one block body.
const BLOCK_END_MINIMUM = CUSTOM_DELIMITER_SIZE_MINIMUM + len(KEYWORD_BLOCK) +
	ACTION_SEPARATOR_SIZE_MINIMUM + TEMPLATE_NAME_SOURCE_SIZE_MINIMUM

// Block_End_Stored is the parse boundary of one block body.
type Block_End_Stored Cursor_End_Storage

// Block_End_Stored_Invariants fixes that boundary against its clause.
func Block_End_Stored_Invariants(value Block_End_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(BLOCK_END_MINIMUM),
			uint16(OWED_END_BOUNDARY_MAXIMUM)).
		Ensure()
}

// DEFINITION_BOUNDARY_POSITION_MINIMUM bounds the read offset inside one definition header.
const DEFINITION_BOUNDARY_POSITION_MINIMUM = CUSTOM_DELIMITER_SIZE_MINIMUM + len(KEYWORD_DEFINE) +
	ACTION_SEPARATOR_SIZE_MINIMUM + TEMPLATE_NAME_SOURCE_SIZE_MINIMUM

// Definition_Boundary_Position_Stored is the read offset inside one definition header.
type Definition_Boundary_Position_Stored Cursor_Position_Storage

// Definition_Boundary_Position_Stored_Invariants fixes that offset against its clause.
func Definition_Boundary_Position_Stored_Invariants(
	value Definition_Boundary_Position_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(DEFINITION_BOUNDARY_POSITION_MINIMUM),
			uint16(OWED_END_BOUNDARY_MAXIMUM)).
		Ensure()
}

// DEFINITION_BOUNDARY_END_MINIMUM bounds the parse boundary of one definition header.
const DEFINITION_BOUNDARY_END_MINIMUM = CUSTOM_DELIMITER_SIZE_MINIMUM + len(KEYWORD_DEFINE) +
	ACTION_SEPARATOR_SIZE_MINIMUM + TEMPLATE_NAME_SOURCE_SIZE_MINIMUM

// Definition_Boundary_End_Stored is the parse boundary of one definition header.
type Definition_Boundary_End_Stored Cursor_End_Storage

// Definition_Boundary_End_Stored_Invariants fixes that boundary against its clause.
func Definition_Boundary_End_Stored_Invariants(
	value Definition_Boundary_End_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(DEFINITION_BOUNDARY_END_MINIMUM),
			uint16(OWED_END_BOUNDARY_MAXIMUM)).
		Ensure()
}

// DEFINITION_KEYWORD_POSITION_MINIMUM bounds the read offset inside one definition keyword.
const DEFINITION_KEYWORD_POSITION_MINIMUM = CUSTOM_DELIMITER_SIZE_MINIMUM + len(KEYWORD_BLOCK)

// Definition_Keyword_Position_Stored is the read offset inside one definition keyword.
type Definition_Keyword_Position_Stored Cursor_Position_Storage

// Definition_Keyword_Position_Stored_Invariants fixes that offset against its clause.
func Definition_Keyword_Position_Stored_Invariants(
	value Definition_Keyword_Position_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(DEFINITION_KEYWORD_POSITION_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Ensure()
}

// DEFINITION_KEYWORD_END_MINIMUM bounds the parse boundary of one definition keyword.
const DEFINITION_KEYWORD_END_MINIMUM = CUSTOM_DELIMITER_SIZE_MINIMUM + len(KEYWORD_BLOCK)

// Definition_Keyword_End_Stored is the parse boundary of one definition keyword.
type Definition_Keyword_End_Stored Cursor_End_Storage

// Definition_Keyword_End_Stored_Invariants fixes that boundary against its clause.
func Definition_Keyword_End_Stored_Invariants(
	value Definition_Keyword_End_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(DEFINITION_KEYWORD_END_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Ensure()
}

// NAME_BOUNDARY_POSITION_MINIMUM bounds the read offset inside one invocation name.
const NAME_BOUNDARY_POSITION_MINIMUM = CUSTOM_DELIMITER_SIZE_MINIMUM + len(KEYWORD_TEMPLATE) +
	ACTION_SEPARATOR_SIZE_MINIMUM + TEMPLATE_NAME_SOURCE_SIZE_MINIMUM

// Name_Boundary_Position_Stored is the read offset inside one invocation name.
type Name_Boundary_Position_Stored Cursor_Position_Storage

// Name_Boundary_Position_Stored_Invariants fixes that offset against its clause.
func Name_Boundary_Position_Stored_Invariants(
	value Name_Boundary_Position_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(NAME_BOUNDARY_POSITION_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Ensure()
}

// NAME_BOUNDARY_END_MINIMUM bounds the parse boundary of one invocation name.
const NAME_BOUNDARY_END_MINIMUM = CUSTOM_DELIMITER_SIZE_MINIMUM + len(KEYWORD_TEMPLATE) +
	ACTION_SEPARATOR_SIZE_MINIMUM + TEMPLATE_NAME_SOURCE_SIZE_MINIMUM

// Name_Boundary_End_Stored is the parse boundary of one invocation name.
type Name_Boundary_End_Stored Cursor_End_Storage

// Name_Boundary_End_Stored_Invariants fixes that boundary against its clause.
func Name_Boundary_End_Stored_Invariants(value Name_Boundary_End_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(NAME_BOUNDARY_END_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Ensure()
}

// Control_Pipeline_Token is one retained token holding one control keyword before its pipeline.
type Control_Pipeline_Token Token

// Control_Pipeline_Token_Invariants fixes the kinds that role admits.
func Control_Pipeline_Token_Invariants(value Control_Pipeline_Token, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(uint8(value.Kind), uint8(TOKEN_IF), uint8(TOKEN_RANGE),
			uint8(TOKEN_WITH)).
		Range_Uint16(
			uint16(value.Start), uint16(TOKEN_BOUNDARY_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM-ACTION_BOUNDARY_SIZE_MINIMUM)).
		Range_Uint16(uint16(value.End), uint16(CONTROL_KEYWORD_END_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Ensure()
}

// Control_Pipeline_Token_Storage is that retained token inside its cursor field.
type Control_Pipeline_Token_Storage Token_Storage

// Control_Pipeline_Token_Storage_Invariants composes the retained token.
func Control_Pipeline_Token_Storage_Invariants(value Control_Pipeline_Token_Storage,
	namespace aver.Namespace) {
	Control_Pipeline_Token_Invariants(Control_Pipeline_Token(value.Value), namespace)
}

// Control_Pipeline_Cursor is one lexer whose retained token holds one control keyword.
type Control_Pipeline_Cursor Token_Cursor

// Control_Pipeline_Cursor_Invariants composes lexer state beside the retained token.
func Control_Pipeline_Cursor_Invariants(value Control_Pipeline_Cursor, namespace aver.Namespace) {
	Control_Pipeline_Position_Stored_Invariants(
		Control_Pipeline_Position_Stored(value.Position), namespace)
	Control_Pipeline_End_Stored_Invariants(Control_Pipeline_End_Stored(value.End), namespace)
	Control_Pipeline_Token_Storage_Invariants(Control_Pipeline_Token_Storage(value.Token),
		namespace)
	Unread_Look_Ahead_Storage_Invariants(Unread_Look_Ahead_Storage(value.Looked), namespace)
	Cursor_Number_Stored_Invariants(Cursor_Number_Stored(value.Number), namespace)
	Function_Function_Storage_Invariants(value.Function, namespace)
	Control_Pipeline_Last_Start_Invariants(Control_Pipeline_Last_Start(value.Last_Start),
		namespace)
	Read_Has_Last_Stored_Invariants(Read_Has_Last_Stored(value.Has_Last), namespace)
}

// Control_Pipeline_Cursor_Pointer keeps mutation of that lexer typed.
type Control_Pipeline_Cursor_Pointer *Control_Pipeline_Cursor

// Control_Pipeline_Cursor_Pointer_Invariants composes present lexer state.
func Control_Pipeline_Cursor_Pointer_Invariants(value Control_Pipeline_Cursor_Pointer,
	namespace aver.Namespace) {
	if value == nil {
		return
	}
	Control_Pipeline_Cursor_Invariants(*value, namespace)
}

// Comment_Last_Start is the unset diagnostic boundary of one comment action.
type Comment_Last_Start Cursor_Last_Start_Storage

// Comment_Last_Start_Invariants fixes that boundary as unset.
func Comment_Last_Start_Invariants(value Comment_Last_Start, _ aver.Namespace) {
	aver.Always(uint16(value.Value) == uint16(bits.WORD_16_MINIMUM),
		"One comment action reads before any diagnostic token.")
}

// Comment_Has_Last is the unset diagnostic flag of one comment action.
type Comment_Has_Last Cursor_Has_Last_Storage

// Comment_Has_Last_Invariants fixes that flag as unset.
func Comment_Has_Last_Invariants(value Comment_Has_Last, _ aver.Namespace) {
	aver.Always(uint8(value.Value) == bits.WORD_8_MINIMUM,
		"One comment action holds no diagnostic token.")
}

// Else_Last_Start is the unset diagnostic boundary of one else clause.
type Else_Last_Start Cursor_Last_Start_Storage

// Else_Last_Start_Invariants fixes that boundary as unset.
func Else_Last_Start_Invariants(value Else_Last_Start, _ aver.Namespace) {
	aver.Always(uint16(value.Value) == uint16(bits.WORD_16_MINIMUM),
		"One else clause reads before any diagnostic token.")
}

// Else_Has_Last is the unset diagnostic flag of one else clause.
type Else_Has_Last Cursor_Has_Last_Storage

// Else_Has_Last_Invariants fixes that flag as unset.
func Else_Has_Last_Invariants(value Else_Has_Last, _ aver.Namespace) {
	aver.Always(uint8(value.Value) == bits.WORD_8_MINIMUM,
		"One else clause holds no diagnostic token.")
}

// Action_Term_Last_Start is the unset diagnostic boundary of one output action.
type Action_Term_Last_Start Cursor_Last_Start_Storage

// Action_Term_Last_Start_Invariants fixes that boundary as unset.
func Action_Term_Last_Start_Invariants(value Action_Term_Last_Start, _ aver.Namespace) {
	aver.Always(uint16(value.Value) == uint16(bits.WORD_16_MINIMUM),
		"One output action reads before any diagnostic token.")
}

// Action_Term_Has_Last is the unset diagnostic flag of one output action.
type Action_Term_Has_Last Cursor_Has_Last_Storage

// Action_Term_Has_Last_Invariants fixes that flag as unset.
func Action_Term_Has_Last_Invariants(value Action_Term_Has_Last, _ aver.Namespace) {
	aver.Always(uint8(value.Value) == bits.WORD_8_MINIMUM,
		"One output action holds no diagnostic token.")
}

// Invocation_Last_Start is the unset diagnostic boundary of one template invocation.
type Invocation_Last_Start Cursor_Last_Start_Storage

// Invocation_Last_Start_Invariants fixes that boundary as unset.
func Invocation_Last_Start_Invariants(value Invocation_Last_Start, _ aver.Namespace) {
	aver.Always(uint16(value.Value) == uint16(bits.WORD_16_MINIMUM),
		"One template invocation reads before any diagnostic token.")
}

// Invocation_Has_Last is the unset diagnostic flag of one template invocation.
type Invocation_Has_Last Cursor_Has_Last_Storage

// Invocation_Has_Last_Invariants fixes that flag as unset.
func Invocation_Has_Last_Invariants(value Invocation_Has_Last, _ aver.Namespace) {
	aver.Always(uint8(value.Value) == bits.WORD_8_MINIMUM,
		"One template invocation holds no diagnostic token.")
}

// End_Clause_Last_Start is the unset diagnostic boundary of one end clause.
type End_Clause_Last_Start Cursor_Last_Start_Storage

// End_Clause_Last_Start_Invariants fixes that boundary as unset.
func End_Clause_Last_Start_Invariants(value End_Clause_Last_Start, _ aver.Namespace) {
	aver.Always(uint16(value.Value) == uint16(bits.WORD_16_MINIMUM),
		"One end clause reads before any diagnostic token.")
}

// End_Clause_Has_Last is the unset diagnostic flag of one end clause.
type End_Clause_Has_Last Cursor_Has_Last_Storage

// End_Clause_Has_Last_Invariants fixes that flag as unset.
func End_Clause_Has_Last_Invariants(value End_Clause_Has_Last, _ aver.Namespace) {
	aver.Always(uint8(value.Value) == bits.WORD_8_MINIMUM,
		"One end clause holds no diagnostic token.")
}

// Branch_Keyword_Last_Start is the unset diagnostic boundary of one branch clause.
type Branch_Keyword_Last_Start Cursor_Last_Start_Storage

// Branch_Keyword_Last_Start_Invariants fixes that boundary as unset.
func Branch_Keyword_Last_Start_Invariants(value Branch_Keyword_Last_Start, _ aver.Namespace) {
	aver.Always(uint16(value.Value) == uint16(bits.WORD_16_MINIMUM),
		"One branch clause reads before any diagnostic token.")
}

// Branch_Keyword_Has_Last is the unset diagnostic flag of one branch clause.
type Branch_Keyword_Has_Last Cursor_Has_Last_Storage

// Branch_Keyword_Has_Last_Invariants fixes that flag as unset.
func Branch_Keyword_Has_Last_Invariants(value Branch_Keyword_Has_Last, _ aver.Namespace) {
	aver.Always(uint8(value.Value) == bits.WORD_8_MINIMUM,
		"One branch clause holds no diagnostic token.")
}

// Control_Term_Last_Start is the unset diagnostic boundary of one control action.
type Control_Term_Last_Start Cursor_Last_Start_Storage

// Control_Term_Last_Start_Invariants fixes that boundary as unset.
func Control_Term_Last_Start_Invariants(value Control_Term_Last_Start, _ aver.Namespace) {
	aver.Always(uint16(value.Value) == uint16(bits.WORD_16_MINIMUM),
		"One control action reads before any diagnostic token.")
}

// Control_Term_Has_Last is the unset diagnostic flag of one control action.
type Control_Term_Has_Last Cursor_Has_Last_Storage

// Control_Term_Has_Last_Invariants fixes that flag as unset.
func Control_Term_Has_Last_Invariants(value Control_Term_Has_Last, _ aver.Namespace) {
	aver.Always(uint8(value.Value) == bits.WORD_8_MINIMUM,
		"One control action holds no diagnostic token.")
}

// Definition_Keyword_Last_Start is the unset diagnostic boundary of one definition keyword.
type Definition_Keyword_Last_Start Cursor_Last_Start_Storage

// Definition_Keyword_Last_Start_Invariants fixes that boundary as unset.
func Definition_Keyword_Last_Start_Invariants(value Definition_Keyword_Last_Start,
	_ aver.Namespace) {
	aver.Always(uint16(value.Value) == uint16(bits.WORD_16_MINIMUM),
		"One definition keyword reads before any diagnostic token.")
}

// Definition_Keyword_Has_Last is the unset diagnostic flag of one definition keyword.
type Definition_Keyword_Has_Last Cursor_Has_Last_Storage

// Definition_Keyword_Has_Last_Invariants fixes that flag as unset.
func Definition_Keyword_Has_Last_Invariants(value Definition_Keyword_Has_Last, _ aver.Namespace) {
	aver.Always(uint8(value.Value) == bits.WORD_8_MINIMUM,
		"One definition keyword holds no diagnostic token.")
}

// CONTROL_PIPELINE_LAST_START_MINIMUM bounds the diagnostic boundary of one control pipeline.
const CONTROL_PIPELINE_LAST_START_MINIMUM = CUSTOM_DELIMITER_SIZE_MINIMUM

// CONTROL_PIPELINE_LAST_START_MAXIMUM bounds that boundary against the clause end.
const CONTROL_PIPELINE_LAST_START_MAXIMUM = ACTION_CONTENT_BOUNDARY_MAXIMUM -
	IDENTIFIER_SIZE_MINIMUM - utf8.CHARACTER_SIZE_MINIMUM

// Control_Pipeline_Last_Start is the diagnostic boundary of one control pipeline.
type Control_Pipeline_Last_Start Cursor_Last_Start_Storage

// Control_Pipeline_Last_Start_Invariants fixes that boundary inside its clause.
func Control_Pipeline_Last_Start_Invariants(value Control_Pipeline_Last_Start,
	namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(CONTROL_PIPELINE_LAST_START_MINIMUM),
			uint16(CONTROL_PIPELINE_LAST_START_MAXIMUM)).
		Ensure()
}

// CHAIN_TERM_LAST_START_MINIMUM bounds the diagnostic boundary of one chained term.
const CHAIN_TERM_LAST_START_MINIMUM = CUSTOM_DELIMITER_SIZE_MINIMUM

// CHAIN_TERM_LAST_START_MAXIMUM bounds that boundary against the clause end.
const CHAIN_TERM_LAST_START_MAXIMUM = ACTION_CONTENT_BOUNDARY_MAXIMUM - IDENTIFIER_SIZE_MINIMUM

// Chain_Term_Last_Start is the diagnostic boundary of one chained term.
type Chain_Term_Last_Start Cursor_Last_Start_Storage

// Chain_Term_Last_Start_Invariants fixes that boundary inside its clause.
func Chain_Term_Last_Start_Invariants(value Chain_Term_Last_Start, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(CHAIN_TERM_LAST_START_MINIMUM),
			uint16(CHAIN_TERM_LAST_START_MAXIMUM)).
		Ensure()
}

// CLOSE_PAREN_LAST_START_MINIMUM bounds the diagnostic boundary of one closed parenthesis.
const CLOSE_PAREN_LAST_START_MINIMUM = CUSTOM_DELIMITER_SIZE_MINIMUM + PARENTHESIS_SIZE

// CLOSE_PAREN_LAST_START_MAXIMUM bounds that boundary against the clause end.
const CLOSE_PAREN_LAST_START_MAXIMUM = ACTION_CONTENT_BOUNDARY_MAXIMUM -
	IDENTIFIER_SIZE_MINIMUM - utf8.CHARACTER_SIZE_MINIMUM

// Close_Paren_Last_Start is the diagnostic boundary of one closed parenthesis.
type Close_Paren_Last_Start Cursor_Last_Start_Storage

// Close_Paren_Last_Start_Invariants fixes that boundary inside its clause.
func Close_Paren_Last_Start_Invariants(value Close_Paren_Last_Start, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(CLOSE_PAREN_LAST_START_MINIMUM),
			uint16(CLOSE_PAREN_LAST_START_MAXIMUM)).
		Ensure()
}

// EXHAUSTED_LAST_START_MINIMUM bounds the diagnostic boundary of one finished pipeline.
const EXHAUSTED_LAST_START_MINIMUM = CUSTOM_DELIMITER_SIZE_MINIMUM

// EXHAUSTED_LAST_START_MAXIMUM bounds that boundary against the clause end.
const EXHAUSTED_LAST_START_MAXIMUM = ACTION_CONTENT_BOUNDARY_MAXIMUM - IDENTIFIER_SIZE_MINIMUM

// Exhausted_Last_Start is the diagnostic boundary of one finished pipeline.
type Exhausted_Last_Start Cursor_Last_Start_Storage

// Exhausted_Last_Start_Invariants fixes that boundary inside its clause.
func Exhausted_Last_Start_Invariants(value Exhausted_Last_Start, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(EXHAUSTED_LAST_START_MINIMUM),
			uint16(EXHAUSTED_LAST_START_MAXIMUM)).
		Ensure()
}

// BLOCK_LAST_START_MINIMUM bounds the diagnostic boundary of one block body.
const BLOCK_LAST_START_MINIMUM = CUSTOM_DELIMITER_SIZE_MINIMUM + len(KEYWORD_BLOCK) +
	CUSTOM_DELIMITER_SIZE_MINIMUM

// BLOCK_LAST_START_MAXIMUM bounds that boundary against the clause end.
const BLOCK_LAST_START_MAXIMUM = OWED_END_BOUNDARY_MAXIMUM - QUOTED_BOUNDARY_SIZE

// OWED_END_BOUNDARY_MAXIMUM is the last content boundary one definition reaches while it still
// leaves room for the end it owes.
const OWED_END_BOUNDARY_MAXIMUM = ACTION_CONTENT_BOUNDARY_MAXIMUM -
	TEMPLATE_DEFINITION_END_SIZE_MINIMUM

// Block_Last_Start is the diagnostic boundary of one block body.
type Block_Last_Start Cursor_Last_Start_Storage

// Block_Last_Start_Invariants fixes that boundary inside its clause.
func Block_Last_Start_Invariants(value Block_Last_Start, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(BLOCK_LAST_START_MINIMUM),
			uint16(BLOCK_LAST_START_MAXIMUM)).
		Ensure()
}

// DEFINITION_BOUNDARY_LAST_START_MINIMUM bounds the diagnostic boundary of one definition header.
const DEFINITION_BOUNDARY_LAST_START_MINIMUM = CUSTOM_DELIMITER_SIZE_MINIMUM +
	len(KEYWORD_DEFINE) + CUSTOM_DELIMITER_SIZE_MINIMUM

// DEFINITION_BOUNDARY_LAST_START_MAXIMUM bounds that boundary against the clause end.
const DEFINITION_BOUNDARY_LAST_START_MAXIMUM = OWED_END_BOUNDARY_MAXIMUM

// Definition_Boundary_Last_Start is the diagnostic boundary of one definition header.
type Definition_Boundary_Last_Start Cursor_Last_Start_Storage

// Definition_Boundary_Last_Start_Invariants fixes that boundary inside its clause.
func Definition_Boundary_Last_Start_Invariants(value Definition_Boundary_Last_Start,
	namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(DEFINITION_BOUNDARY_LAST_START_MINIMUM),
			uint16(DEFINITION_BOUNDARY_LAST_START_MAXIMUM)).
		Ensure()
}

// NAME_BOUNDARY_LAST_START_MINIMUM bounds the diagnostic boundary of one invocation name.
const NAME_BOUNDARY_LAST_START_MINIMUM = CUSTOM_DELIMITER_SIZE_MINIMUM + len(KEYWORD_TEMPLATE) +
	CUSTOM_DELIMITER_SIZE_MINIMUM

// NAME_BOUNDARY_LAST_START_MAXIMUM bounds that boundary against the clause end.
const NAME_BOUNDARY_LAST_START_MAXIMUM = ACTION_CONTENT_BOUNDARY_MAXIMUM - QUOTED_BOUNDARY_SIZE

// Name_Boundary_Last_Start is the diagnostic boundary of one invocation name.
type Name_Boundary_Last_Start Cursor_Last_Start_Storage

// Name_Boundary_Last_Start_Invariants fixes that boundary inside its clause.
func Name_Boundary_Last_Start_Invariants(value Name_Boundary_Last_Start, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(NAME_BOUNDARY_LAST_START_MINIMUM),
			uint16(NAME_BOUNDARY_LAST_START_MAXIMUM)).
		Ensure()
}

// SIMPLE_PROGRAM_SOURCE_SIZE_MINIMUM is the shortest source that holds one bare term.
const SIMPLE_PROGRAM_SOURCE_SIZE_MINIMUM = ACTION_BOUNDARY_SIZE_MINIMUM + IDENTIFIER_SIZE_MINIMUM

// Simple_Program_Source is one source that holds one bare term.
type Simple_Program_Source Source

// Simple_Program_Source_Invariants bounds that source by its shortest action.
func Simple_Program_Source_Invariants(value Simple_Program_Source, namespace aver.Namespace) {
	Simple_Program_Data_Stored_Invariants(Simple_Program_Data_Stored(value.Data), namespace)
}

// Simple_Program_Data_Stored separates that source shape from its borrowed header.
type Simple_Program_Data_Stored Source_Storage

// Simple_Program_Data_Stored_Invariants bounds data by the shortest action.
func Simple_Program_Data_Stored_Invariants(
	value Simple_Program_Data_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value.Value), SIMPLE_PROGRAM_SOURCE_SIZE_MINIMUM,
			SOURCE_SIZE_MAXIMUM).
		Ensure()
}

// Simple_Program is one program whose action holds one bare term.
type Simple_Program Program_Validated

// Simple_Program_Invariants composes that source with arena metadata.
func Simple_Program_Invariants(value Simple_Program, namespace aver.Namespace) {
	Simple_Program_Source_Invariants(Simple_Program_Source(value.Source), namespace)
	Simple_Program_Document_Invariants(Simple_Program_Document(value.Document), namespace)
	Syntax_Storage_Invariants(value.Syntax, namespace)
}

// FIELD_PROGRAM_SOURCE_SIZE_MINIMUM is the shortest source that holds one field term.
const FIELD_PROGRAM_SOURCE_SIZE_MINIMUM = SIMPLE_PROGRAM_SOURCE_SIZE_MINIMUM +
	IDENTIFIER_SIZE_MINIMUM

// Field_Program_Source is one source that holds one field term.
type Field_Program_Source Source

// Field_Program_Source_Invariants bounds that source by its shortest action.
func Field_Program_Source_Invariants(value Field_Program_Source, namespace aver.Namespace) {
	Field_Program_Data_Stored_Invariants(Field_Program_Data_Stored(value.Data), namespace)
}

// Field_Program_Data_Stored separates that source shape from its borrowed header.
type Field_Program_Data_Stored Source_Storage

// Field_Program_Data_Stored_Invariants bounds data by the shortest action.
func Field_Program_Data_Stored_Invariants(
	value Field_Program_Data_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value.Value), FIELD_PROGRAM_SOURCE_SIZE_MINIMUM, SOURCE_SIZE_MAXIMUM).
		Ensure()
}

// Field_Program is one program whose action holds one field term.
type Field_Program Program_Validated

// Field_Program_Invariants composes that source with arena metadata.
func Field_Program_Invariants(value Field_Program, namespace aver.Namespace) {
	Field_Program_Source_Invariants(Field_Program_Source(value.Source), namespace)
	Field_Program_Document_Invariants(Field_Program_Document(value.Document), namespace)
	Syntax_Storage_Invariants(value.Syntax, namespace)
}

// FIELD_ACTION_SIZE_MINIMUM is the shortest one-byte-delimited field access.
const FIELD_ACTION_SIZE_MINIMUM = ACTION_BOUNDARY_SIZE_MINIMUM + DOT_SOURCE_SIZE +
	IDENTIFIER_SIZE_MINIMUM

// FIELD_ACTION_NODE_COUNT counts one action, its pipeline, its command, and its field.
const FIELD_ACTION_NODE_COUNT = ACTION_PIPELINE_NODE_COUNT + COMMAND_NODE_COUNT +
	SCALAR_NODE_COUNT

// FIELD_PROGRAM_NODE_COUNT_MAXIMUM packs actions beside the shortest field access.
const FIELD_PROGRAM_NODE_COUNT_MAXIMUM = ROOT_NODE_COUNT + FIELD_ACTION_NODE_COUNT +
	(SOURCE_SIZE_MAXIMUM-FIELD_ACTION_SIZE_MINIMUM)/SIMPLE_ACTION_SIZE_MINIMUM*
		SIMPLE_ACTION_NODE_COUNT

// Field_Program_Document is arena metadata of a program that holds one field access.
type Field_Program_Document Program_Validated_Document

// Field_Program_Document_Invariants bounds counts by the shortest field access.
func Field_Program_Document_Invariants(
	value Field_Program_Document, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Node_Count), uint16(ACTION_PROGRAM_NODE_COUNT_MINIMUM),
			uint16(FIELD_PROGRAM_NODE_COUNT_MAXIMUM)).
		Range_Uint16(
			uint16(value.First_Template), uint16(DOCUMENT_FIRST_TEMPLATE_MINIMUM),
			uint16(DOCUMENT_FIRST_TEMPLATE_MAXIMUM-CONTROL_BRANCH_NODE_COUNT)).
		Range_Uint16(uint16(value.Template_Count), uint16(TEMPLATE_COUNT_MINIMUM),
			uint16(EXECUTED_TEMPLATE_COUNT_MAXIMUM)).
		Ensure()
	aver.Always(uint16(value.Root) == uint16(ROOT_NODE_COUNT),
		"One field program keeps the root in the first syntax arena slot.")
}

// CHAIN_PROGRAM_SOURCE_SIZE_MINIMUM is the shortest source that holds one chained term.
const CHAIN_PROGRAM_SOURCE_SIZE_MINIMUM = FIELD_PROGRAM_SOURCE_SIZE_MINIMUM +
	IDENTIFIER_SIZE_MINIMUM

// Chain_Program_Source is one source that holds one chained term.
type Chain_Program_Source Source

// Chain_Program_Source_Invariants bounds that source by its shortest action.
func Chain_Program_Source_Invariants(value Chain_Program_Source, namespace aver.Namespace) {
	Chain_Program_Data_Stored_Invariants(Chain_Program_Data_Stored(value.Data), namespace)
}

// Chain_Program_Data_Stored separates that source shape from its borrowed header.
type Chain_Program_Data_Stored Source_Storage

// Chain_Program_Data_Stored_Invariants bounds data by the shortest action.
func Chain_Program_Data_Stored_Invariants(
	value Chain_Program_Data_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value.Value), CHAIN_PROGRAM_SOURCE_SIZE_MINIMUM, SOURCE_SIZE_MAXIMUM).
		Ensure()
}

// Chain_Program is one program whose action holds one chained term.
type Chain_Program Program_Validated

// Chain_Program_Invariants composes that source with arena metadata.
func Chain_Program_Invariants(value Chain_Program, namespace aver.Namespace) {
	Chain_Program_Source_Invariants(Chain_Program_Source(value.Source), namespace)
	Chained_Program_Document_Invariants(Chained_Program_Document(value.Document), namespace)
	Syntax_Storage_Invariants(value.Syntax, namespace)
}

// VARIABLE_PROGRAM_SOURCE_SIZE_MINIMUM is the shortest source that holds one variable term.
const VARIABLE_PROGRAM_SOURCE_SIZE_MINIMUM = CHAIN_PROGRAM_SOURCE_SIZE_MINIMUM +
	IDENTIFIER_SIZE_MINIMUM

// Variable_Program_Source is one source that holds one variable term.
type Variable_Program_Source Source

// Variable_Program_Source_Invariants bounds that source by its shortest action.
func Variable_Program_Source_Invariants(value Variable_Program_Source, namespace aver.Namespace) {
	Variable_Program_Data_Stored_Invariants(Variable_Program_Data_Stored(value.Data), namespace)
}

// Variable_Program_Data_Stored separates that source shape from its borrowed header.
type Variable_Program_Data_Stored Source_Storage

// Variable_Program_Data_Stored_Invariants bounds data by the shortest action.
func Variable_Program_Data_Stored_Invariants(
	value Variable_Program_Data_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value.Value), VARIABLE_PROGRAM_SOURCE_SIZE_MINIMUM,
			SOURCE_SIZE_MAXIMUM).
		Ensure()
}

// Variable_Program is one program whose action holds one variable term.
type Variable_Program Program_Validated

// Variable_Program_Invariants composes that source with arena metadata.
func Variable_Program_Invariants(value Variable_Program, namespace aver.Namespace) {
	Variable_Program_Source_Invariants(Variable_Program_Source(value.Source), namespace)
	Bound_Program_Document_Invariants(Bound_Program_Document(value.Document), namespace)
	Syntax_Storage_Invariants(value.Syntax, namespace)
}

// CONTROL_PROGRAM_SOURCE_SIZE_MINIMUM is the shortest source that holds one control body.
const CONTROL_PROGRAM_SOURCE_SIZE_MINIMUM = KEYWORD_SOURCE_SIZE_MINIMUM +
	ACTION_SEPARATOR_SIZE_MINIMUM + IDENTIFIER_SIZE_MINIMUM +
	WORD_SOURCE_SIZE_MINIMUM

// Control_Program_Source is one source that holds one control body.
type Control_Program_Source Source

// Control_Program_Source_Invariants bounds that source by its shortest action.
func Control_Program_Source_Invariants(value Control_Program_Source, namespace aver.Namespace) {
	Control_Program_Data_Stored_Invariants(Control_Program_Data_Stored(value.Data), namespace)
}

// Control_Program_Data_Stored separates that source shape from its borrowed header.
type Control_Program_Data_Stored Source_Storage

// Control_Program_Data_Stored_Invariants bounds data by the shortest action.
func Control_Program_Data_Stored_Invariants(
	value Control_Program_Data_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value.Value), CONTROL_PROGRAM_SOURCE_SIZE_MINIMUM,
			SOURCE_SIZE_MAXIMUM).
		Ensure()
}

// CONTROL_BRANCH_NODE_COUNT counts one branch, its pipeline, its term, and its body list.
const CONTROL_BRANCH_NODE_COUNT = ACTION_PIPELINE_NODE_COUNT + COMMAND_NODE_COUNT +
	SCALAR_NODE_COUNT + NODE_COUNT_INCREMENT

// Control_Program is one program whose action holds one control body.
type Control_Program Program_Validated

// Control_Program_Invariants composes that source with arena metadata.
func Control_Program_Invariants(value Control_Program, namespace aver.Namespace) {
	Control_Program_Source_Invariants(Control_Program_Source(value.Source), namespace)
	Control_Program_Document_Invariants(
		Control_Program_Document(value.Document), namespace)
	Syntax_Storage_Invariants(value.Syntax, namespace)
}

// CONTROL_PROGRAM_NODE_COUNT_MINIMUM counts the root, one branch, and its empty body.
const CONTROL_PROGRAM_NODE_COUNT_MINIMUM = ROOT_NODE_COUNT + CONTROL_BRANCH_NODE_COUNT

// CONTROL_BRANCH_SOURCE_SIZE_MINIMUM is the shortest branch beside one body byte.
const CONTROL_BRANCH_SOURCE_SIZE_MINIMUM = CONTROL_ACTION_SIZE_MINIMUM +
	utf8.CHARACTER_SIZE_MINIMUM

// CONTROL_PROGRAM_NODE_COUNT_MAXIMUM packs actions beside the shortest branch.
const CONTROL_PROGRAM_NODE_COUNT_MAXIMUM = ROOT_NODE_COUNT + CONTROL_BRANCH_NODE_COUNT +
	NODE_COUNT_INCREMENT +
	(SOURCE_SIZE_MAXIMUM-CONTROL_BRANCH_SOURCE_SIZE_MINIMUM)/SIMPLE_ACTION_SIZE_MINIMUM*
		SIMPLE_ACTION_NODE_COUNT + TRAILING_TEXT_NODE_COUNT_MAXIMUM

// Control_Program_Document is arena metadata of a program that holds one branch.
type Control_Program_Document Program_Validated_Document

// Control_Program_Document_Invariants bounds counts by the shortest branch.
func Control_Program_Document_Invariants(
	value Control_Program_Document, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Node_Count),
			uint16(CONTROL_PROGRAM_NODE_COUNT_MINIMUM),
			uint16(CONTROL_PROGRAM_NODE_COUNT_MAXIMUM)).
		Range_Uint16(
			uint16(value.First_Template), uint16(DOCUMENT_FIRST_TEMPLATE_MINIMUM),
			uint16(CONTROL_FIRST_TEMPLATE_MAXIMUM)).
		Range_Uint16(uint16(value.Template_Count), uint16(TEMPLATE_COUNT_MINIMUM),
			uint16(EXECUTED_TEMPLATE_COUNT_MAXIMUM)).
		Ensure()
	aver.Always(uint16(value.Root) == uint16(ROOT_NODE_COUNT),
		"One branch program keeps the root in the first syntax arena slot.")
}

// RANGE_PROGRAM_SOURCE_SIZE_MINIMUM is the shortest source that holds one range body.
const RANGE_PROGRAM_SOURCE_SIZE_MINIMUM = CLAUSE_SOURCE_SIZE_MINIMUM +
	ACTION_SEPARATOR_SIZE_MINIMUM + IDENTIFIER_SIZE_MINIMUM + WORD_SOURCE_SIZE_MINIMUM

// Range_Program_Source is one source that holds one range body.
type Range_Program_Source Source

// Range_Program_Source_Invariants bounds that source by its shortest action.
func Range_Program_Source_Invariants(value Range_Program_Source, namespace aver.Namespace) {
	Range_Program_Data_Stored_Invariants(Range_Program_Data_Stored(value.Data), namespace)
}

// Range_Program_Data_Stored separates that source shape from its borrowed header.
type Range_Program_Data_Stored Source_Storage

// Range_Program_Data_Stored_Invariants bounds data by the shortest action.
func Range_Program_Data_Stored_Invariants(
	value Range_Program_Data_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value.Value),
			RANGE_PROGRAM_SOURCE_SIZE_MINIMUM+utf8.CHARACTER_SIZE_MINIMUM,
			SOURCE_SIZE_MAXIMUM).
		Ensure()
}

// Range_Program is one program whose action holds one range body.
type Range_Program Program_Validated

// Range_Program_Invariants composes that source with arena metadata.
func Range_Program_Invariants(value Range_Program, namespace aver.Namespace) {
	Range_Program_Source_Invariants(Range_Program_Source(value.Source), namespace)
	Ranged_Program_Document_Invariants(Ranged_Program_Document(value.Document), namespace)
	Syntax_Storage_Invariants(value.Syntax, namespace)
}

// ACTION_RIGHT_INDEX_CURSOR_VALIDATED_VALUE_MINIMUM is the floor this call site starts from.
const ACTION_RIGHT_INDEX_CURSOR_VALIDATED_VALUE_MINIMUM = SOURCE_SIZE_MINIMUM + 1

// Action_Right_Index_Cursor_Validated is the Source Cursor Validated action_right_index carries.
type Action_Right_Index_Cursor_Validated Source_Cursor_Validated

// Action_Right_Index_Cursor_Validated_Invariants checks ownership after source validation.
func Action_Right_Index_Cursor_Validated_Invariants(
	value Action_Right_Index_Cursor_Validated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value),
			uint16(ACTION_RIGHT_INDEX_CURSOR_VALIDATED_VALUE_MINIMUM),
			uint16(SOURCE_SIZE_MAXIMUM)).
		Ensure()
}

// Allocate_Node_Count is the Unrooted Node Count allocate_node carries.
type Allocate_Node_Count Unrooted_Node_Count

// Allocate_Node_Count_Invariants bounds a count the root may not occupy yet.
func Allocate_Node_Count_Invariants(value Allocate_Node_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(NODE_COUNT_MINIMUM),
			uint16(NODE_COUNT_MAXIMUM-NODE_COUNT_INCREMENT)).
		Ensure()
}

// Allocate_Node_Allocation is the parser-built node allocate_node carries.
type Allocate_Node_Allocation struct {
	// Value cannot bypass arena validation.
	Value Allocate_Node_Stored
}

// Allocate_Node_Allocation_Invariants keeps parser construction proof attached to node.
func Allocate_Node_Allocation_Invariants(value Allocate_Node_Allocation, namespace aver.Namespace) {
	Allocate_Node_Stored_Invariants(value.Value, namespace)
}

// Allocate_Node_Stored is the allocation record allocate_node carries.
type Allocate_Node_Stored Node

// Allocate_Node_Stored_Invariants fixes allocation-record representation.
func Allocate_Node_Stored_Invariants(value Allocate_Node_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value.Kind), uint8(NODE_KIND_MINIMUM), uint8(NODE_KIND_MAXIMUM)).
		Range_Uint16(uint16(value.Start), uint16(SOURCE_SIZE_MINIMUM), uint16(
			SOURCE_SIZE_MAXIMUM)).
		Range_Uint16(
			uint16(value.End), uint16(SOURCE_SIZE_MINIMUM), uint16(SOURCE_SIZE_MAXIMUM),
		).
		Range_Uint16(uint16(value.Value_Start), uint16(SOURCE_SIZE_MINIMUM), uint16(
			NODE_VALUE_START_MAXIMUM)).
		Range_Uint16(uint16(value.Value_End), uint16(SOURCE_SIZE_MINIMUM), uint16(
			SOURCE_SIZE_MAXIMUM)).
		Range_Holed_Uint16(
			uint16(value.First_Child), uint16(NODE_REFERENCE_MINIMUM),
			uint16(
				NODE_REFERENCE_MAXIMUM-CONTROL_BRANCH_NODE_COUNT-
					NODE_COUNT_INCREMENT), uint16(ROOT_LIST_HOLE),
			uint16(TERM_ACTION_HOLE), uint16(TERM_ACTION_HOLE)).
		Ensure()
	// Every allocation leaves the list link for the append that follows it.
	aver.Always(uint16(value.Next_Sibling) == uint16(NO_NODE),
		"One allocation record carries no sibling link for allocate_node.")
}

// APPEND_CHILD_VALIDATED_VALUE_MINIMUM is the floor this call site starts from.
const APPEND_CHILD_VALIDATED_VALUE_MINIMUM = NODE_REFERENCE_MINIMUM + 2

// Append_Child_Validated is the Node Link Validated append_child carries.
type Append_Child_Validated Node_Link_Validated

// Append_Child_Validated_Invariants checks ownership after program validation.
func Append_Child_Validated_Invariants(value Append_Child_Validated, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(APPEND_CHILD_VALIDATED_VALUE_MINIMUM),
			uint16(NODE_REFERENCE_MAXIMUM)).
		Ensure()
}

// Append_Child_Link_Validated is the Node Link Validated append_child carries.
type Append_Child_Link_Validated Node_Link_Validated

// Append_Child_Link_Validated_Invariants checks ownership after program validation.
func Append_Child_Link_Validated_Invariants(value Append_Child_Link_Validated,
	namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Holed_Uint16(
			uint16(value.Value), uint16(NODE_REFERENCE_MINIMUM),
			uint16(ACTION_TAIL_REFERENCE_MAXIMUM),
			uint16(ROOT_LIST_HOLE), uint16(ROOT_LIST_HOLE),
			uint16(ROOT_LIST_HOLE)).
		Ensure()
}

// APPEND_CHILD_PARENT_VALUE_VALIDATED_VALUE_MINIMUM is the floor this call site starts from.
const APPEND_CHILD_PARENT_VALUE_VALIDATED_VALUE_MINIMUM = NODE_REFERENCE_MINIMUM + 1

// Append_Child_Parent_Value_Validated is the Node Link Validated append_child carries.
type Append_Child_Parent_Value_Validated Node_Link_Validated

// Append_Child_Parent_Value_Validated_Invariants checks ownership after program validation.
func Append_Child_Parent_Value_Validated_Invariants(value Append_Child_Parent_Value_Validated,
	namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value),
			uint16(APPEND_CHILD_PARENT_VALUE_VALIDATED_VALUE_MINIMUM),
			uint16(NODE_REFERENCE_MAXIMUM)).
		Ensure()
}

// Append_Comment_Allocated is the Node Count Allocated append_comment carries.
type Append_Comment_Allocated Node_Count_Allocated

// Append_Comment_Allocated_Invariants keeps count proof attached to parser flow.
func Append_Comment_Allocated_Invariants(value Append_Comment_Allocated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value.Value), uint16(ROOT_NODE_COUNT),
			uint16(NODE_COUNT_MAXIMUM-(SIMPLE_ACTION_NODE_COUNT+
				CONTROL_BRANCH_NODE_COUNT))).
		Ensure()
}

// Append_Comment_Validated is the Action Span Validated append_comment carries.
type Append_Comment_Validated Action_Span_Validated

// Append_Comment_Validated_Invariants checks transport ownership after delimiter validation.
func Append_Comment_Validated_Invariants(value Append_Comment_Validated, namespace aver.Namespace,
) {
	Append_Comment_Proof_Stored_Invariants(Append_Comment_Proof_Stored(value.Value), namespace)
}

// APPEND_COMMENT_PROOF_STORED_CONTENT_END_MINIMUM is the floor this call site starts from.
const APPEND_COMMENT_PROOF_STORED_CONTENT_END_MINIMUM = ACTION_CONTENT_BOUNDARY_MINIMUM + 4

// APPEND_COMMENT_PROOF_STORED_END_MINIMUM is the floor this call site starts from.
const APPEND_COMMENT_PROOF_STORED_END_MINIMUM = ACTION_END_MINIMUM + 4

// Append_Comment_Proof_Stored is the Action Span Proof Stored append_comment carries.
type Append_Comment_Proof_Stored Action_Span_Proof_Stored

// Append_Comment_Proof_Stored_Invariants fixes action-span representation.
func Append_Comment_Proof_Stored_Invariants(value Append_Comment_Proof_Stored,
	namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value.Start), uint16(ACTION_START_MINIMUM),
			uint16(ACTION_START_MAXIMUM-FIELD_ACTION_SIZE_MINIMUM)).
		Range_Uint16(
			uint16(value.Content_Start), uint16(ACTION_CONTENT_BOUNDARY_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM-FIELD_ACTION_SIZE_MINIMUM)).
		Range_Uint16(uint16(value.Content_End),
			uint16(APPEND_COMMENT_PROOF_STORED_CONTENT_END_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Range_Uint16(uint16(value.End), uint16(APPEND_COMMENT_PROOF_STORED_END_MINIMUM),
			uint16(ACTION_END_MAXIMUM)).
		Ensure()
}

// Append_Output_Action_Allocated is the Node Count Allocated append_output_action carries.
type Append_Output_Action_Allocated Node_Count_Allocated

// Append_Output_Action_Allocated_Invariants keeps count proof attached to parser flow.
func Append_Output_Action_Allocated_Invariants(
	value Append_Output_Action_Allocated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value.Value), uint16(ROOT_NODE_COUNT),
			uint16(NODE_COUNT_MAXIMUM-SIMPLE_ACTION_NODE_COUNT)).
		Ensure()
}

// Append_Source_Text_Allocated is the Node Count Allocated append_source_text carries.
type Append_Source_Text_Allocated Node_Count_Allocated

// Append_Source_Text_Allocated_Invariants keeps count proof attached to parser flow.
func Append_Source_Text_Allocated_Invariants(
	value Append_Source_Text_Allocated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(ROOT_NODE_COUNT),
			uint16(NODE_COUNT_MAXIMUM-NODE_COUNT_INCREMENT)).
		Ensure()
}

// TEXT_SPAN_START_MAXIMUM leaves the text byte one span still holds.
const TEXT_SPAN_START_MAXIMUM = SOURCE_SIZE_MAXIMUM - utf8.CHARACTER_SIZE_MINIMUM

// Append_Source_Text_Proof_Stored is the Text Span Start Proof Stored append_source_text carries.
type Append_Source_Text_Proof_Stored Text_Span_Start_Proof_Stored

// Append_Source_Text_Proof_Stored_Invariants fixes literal-start representation.
func Append_Source_Text_Proof_Stored_Invariants(
	value Append_Source_Text_Proof_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Holed_Uint16(
			uint16(value), uint16(SOURCE_SIZE_MINIMUM),
			uint16(TEXT_SPAN_START_MAXIMUM), uint16(ROOT_LIST_HOLE),
			uint16(TERM_ACTION_HOLE), uint16(TERM_ACTION_HOLE)).
		Ensure()
}

// Append_Template_Invocation_Allocated is the Node Count Allocated append_template_invocation
// carries.
type Append_Template_Invocation_Allocated Node_Count_Allocated

// Append_Template_Invocation_Allocated_Invariants keeps count proof attached to parser flow.
func Append_Template_Invocation_Allocated_Invariants(
	value Append_Template_Invocation_Allocated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value.Value), uint16(ROOT_NODE_COUNT),
			uint16(APPEND_TEMPLATE_INVOCATION_COUNT_MAXIMUM)).
		Ensure()
}

// NUMBER_SUFFIX_END_VALIDATED_VALUE_MINIMUM is the floor this call site starts from.
const NUMBER_SUFFIX_END_VALIDATED_VALUE_MINIMUM = ACTION_CONTENT_BOUNDARY_MINIMUM + 1

// Number_Suffix_End_Validated is the Number Position Validated number_suffix_end carries.
type Number_Suffix_End_Validated Number_Position_Validated

// Number_Suffix_End_Validated_Invariants keeps numeric-span proof attached to cursor.
func Number_Suffix_End_Validated_Invariants(
	value Number_Suffix_End_Validated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value),
			uint16(NUMBER_SUFFIX_END_VALIDATED_VALUE_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Ensure()
}

// Number_Value_Program is the Simple Program number_value carries.
type Number_Value_Program Simple_Program

// Number_Value_Program_Invariants composes that source with arena metadata.
func Number_Value_Program_Invariants(value Number_Value_Program, namespace aver.Namespace) {
	Simple_Program_Source_Invariants(Simple_Program_Source(value.Source), namespace)
	Number_Value_Document_Invariants(Number_Value_Document(value.Document), namespace)
	Syntax_Storage_Invariants(value.Syntax, namespace)
}

// Number_Value_Document is the program document number_value carries.
type Number_Value_Document Program_Validated_Document

// Number_Value_Document_Invariants bounds counts by the shortest action.
func Number_Value_Document_Invariants(value Number_Value_Document, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Node_Count), uint16(ACTION_PROGRAM_NODE_COUNT_MINIMUM),
			uint16(NODE_COUNT_MAXIMUM)).
		Range_Uint16(
			uint16(value.First_Template), uint16(DOCUMENT_FIRST_TEMPLATE_MINIMUM),
			uint16(DOCUMENT_FIRST_TEMPLATE_MAXIMUM)).
		Range_Uint16(
			uint16(value.Template_Count), uint16(TEMPLATE_COUNT_MINIMUM),
			uint16(TEMPLATE_COUNT_MAXIMUM-
				TEMPLATE_DEFINITION_SIZE_MINIMUM)).
		Ensure()
	aver.Always(uint16(value.Root) == uint16(ROOT_NODE_COUNT),
		"An action program keeps the root in the first syntax arena slot for number_value.")
}

// OUTPUT_COMPOSITE_CHILD_STAGED_VALUE_MINIMUM is the floor this call site starts from.
const OUTPUT_COMPOSITE_CHILD_STAGED_VALUE_MINIMUM = OUTPUT_SIZE_MINIMUM + 1

// Output_Composite_Child_Staged is the Output Count Staged output_composite_child carries.
type Output_Composite_Child_Staged Output_Count_Staged

// Output_Composite_Child_Staged_Invariants checks transport ownership after writer validation.
func Output_Composite_Child_Staged_Invariants(value Output_Composite_Child_Staged,
	namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value.Value), uint16(OUTPUT_COMPOSITE_CHILD_STAGED_VALUE_MINIMUM),
			uint16(OUTPUT_SIZE_MAXIMUM)).
		Ensure()
}

// OUTPUT_COMPOSITE_CLOSE_STAGED_VALUE_MINIMUM counts the opening brace one close follows.
const OUTPUT_COMPOSITE_CLOSE_STAGED_VALUE_MINIMUM = OUTPUT_SIZE_MINIMUM +
	utf8.CHARACTER_SIZE_MINIMUM

// Output_Composite_Close_Staged is the Output Count Staged output_composite_close carries.
type Output_Composite_Close_Staged Output_Count_Staged

// Output_Composite_Close_Staged_Invariants checks transport ownership after writer validation.
func Output_Composite_Close_Staged_Invariants(value Output_Composite_Close_Staged,
	namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value),
			uint16(OUTPUT_COMPOSITE_CLOSE_STAGED_VALUE_MINIMUM),
			uint16(OUTPUT_SIZE_MAXIMUM)).
		Ensure()
}

// Parse_Action_Allocated is the Node Count Allocated parse_action carries.
type Parse_Action_Allocated Node_Count_Allocated

// Parse_Action_Allocated_Invariants keeps count proof attached to parser flow.
func Parse_Action_Allocated_Invariants(value Parse_Action_Allocated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value.Value), uint16(ROOT_NODE_COUNT),
			uint16(NODE_COUNT_MAXIMUM-SIMPLE_ACTION_NODE_COUNT)).
		Ensure()
}

// Parse_Control_Action_Allocated is the Node Count Allocated parse_control_action carries.
type Parse_Control_Action_Allocated Node_Count_Allocated

// Parse_Control_Action_Allocated_Invariants keeps count proof attached to parser flow.
func Parse_Control_Action_Allocated_Invariants(value Parse_Control_Action_Allocated,
	namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(ROOT_NODE_COUNT),
			uint16(CONTROL_ACTION_COUNT_MAXIMUM)).
		Ensure()
}

// Parse_Control_Action_Validated is the Action Span Validated parse_control_action carries.
type Parse_Control_Action_Validated Action_Span_Validated

// Parse_Control_Action_Validated_Invariants checks transport ownership after delimiter validation.
func Parse_Control_Action_Validated_Invariants(value Parse_Control_Action_Validated,
	namespace aver.Namespace,
) {
	Parse_Control_Action_Proof_Stored_Invariants(Parse_Control_Action_Proof_Stored(
		value.Value), namespace)
}

// PARSE_CONTROL_ACTION_PROOF_STORED_CONTENT_END_MINIMUM is the floor this call site starts from.
const PARSE_CONTROL_ACTION_PROOF_STORED_CONTENT_END_MINIMUM = ACTION_CONTENT_BOUNDARY_MINIMUM + 2

// PARSE_CONTROL_ACTION_PROOF_STORED_END_MINIMUM is the floor this call site starts from.
const PARSE_CONTROL_ACTION_PROOF_STORED_END_MINIMUM = ACTION_END_MINIMUM + 2

// Parse_Control_Action_Proof_Stored is the Action Span Proof Stored parse_control_action carries.
type Parse_Control_Action_Proof_Stored Action_Span_Proof_Stored

// Parse_Control_Action_Proof_Stored_Invariants fixes action-span representation.
func Parse_Control_Action_Proof_Stored_Invariants(value Parse_Control_Action_Proof_Stored,
	namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value.Start), uint16(ACTION_START_MINIMUM),
			uint16(ACTION_START_MAXIMUM-ACTION_BOUNDARY_SIZE_MINIMUM)).
		Range_Uint16(
			uint16(value.Content_Start), uint16(ACTION_CONTENT_BOUNDARY_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM-ACTION_BOUNDARY_SIZE_MINIMUM)).
		Range_Uint16(uint16(value.Content_End),
			uint16(PARSE_CONTROL_ACTION_PROOF_STORED_CONTENT_END_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Range_Uint16(uint16(value.End),
			uint16(PARSE_CONTROL_ACTION_PROOF_STORED_END_MINIMUM),
			uint16(ACTION_END_MAXIMUM)).
		Ensure()
}

// PARSE_DELIMITED_ACTION_VALIDATED_VALUE_MINIMUM is the floor this call site starts from.
const PARSE_DELIMITED_ACTION_VALIDATED_VALUE_MINIMUM = SOURCE_SIZE_MINIMUM + 1

// Parse_Delimited_Action_Validated is the Source Cursor Validated parse_delimited_action carries.
type Parse_Delimited_Action_Validated Source_Cursor_Validated

// Parse_Delimited_Action_Validated_Invariants checks ownership after source validation.
func Parse_Delimited_Action_Validated_Invariants(value Parse_Delimited_Action_Validated,
	namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value),
			uint16(PARSE_DELIMITED_ACTION_VALIDATED_VALUE_MINIMUM),
			uint16(SOURCE_SIZE_MAXIMUM)).
		Ensure()
}

// Parse_Delimited_Action_Allocated is the Node Count Allocated parse_delimited_action carries.
type Parse_Delimited_Action_Allocated Node_Count_Allocated

// Parse_Delimited_Action_Allocated_Invariants keeps count proof attached to parser flow.
func Parse_Delimited_Action_Allocated_Invariants(value Parse_Delimited_Action_Allocated,
	namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value.Value), uint16(ROOT_NODE_COUNT),
			uint16(NODE_COUNT_MAXIMUM-SIMPLE_ACTION_NODE_COUNT)).
		Ensure()
}

// Parse_Delimited_Action_Source_Cursor_Validated is the Source Cursor Validated
// parse_delimited_action carries.
type Parse_Delimited_Action_Source_Cursor_Validated Source_Cursor_Validated

// Parse_Delimited_Action_Source_Cursor_Validated_Invariants checks ownership after source
// validation.
func Parse_Delimited_Action_Source_Cursor_Validated_Invariants(
	value Parse_Delimited_Action_Source_Cursor_Validated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value.Value), uint16(SOURCE_SIZE_MINIMUM),
			uint16(SOURCE_SIZE_MAXIMUM-ACTION_BOUNDARY_SIZE_MINIMUM)).
		Ensure()
}

// PARSE_PIPELINE_TOKENS_VALIDATED_VALUE_MINIMUM is the floor this call site starts from.
const PARSE_PIPELINE_TOKENS_VALIDATED_VALUE_MINIMUM = NODE_REFERENCE_MINIMUM + 3

// Parse_Pipeline_Tokens_Validated is the Pipeline Root Validated parse_pipeline_tokens carries.
type Parse_Pipeline_Tokens_Validated Pipeline_Root_Validated

// Parse_Pipeline_Tokens_Validated_Invariants checks ownership after arena allocation.
func Parse_Pipeline_Tokens_Validated_Invariants(value Parse_Pipeline_Tokens_Validated,
	namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value),
			uint16(PARSE_PIPELINE_TOKENS_VALIDATED_VALUE_MINIMUM),
			uint16(PIPELINE_REFERENCE_MAXIMUM)).
		Ensure()
}

// Parse_Unchecked_Configuration is the Configuration parse_unchecked carries.
type Parse_Unchecked_Configuration Configuration

// Parse_Unchecked_Configuration_Invariants verifies fixed policy shape and bounded values.
func Parse_Unchecked_Configuration_Invariants(value Parse_Unchecked_Configuration,
	namespace aver.Namespace) {
	Parse_Unchecked_Stored_Invariants(Parse_Unchecked_Stored(value.Left), namespace)
	Configuration_Right_Stored_Invariants(Configuration_Right_Stored(value.Right), namespace)
	Configuration_Mode_Stored_Invariants(Configuration_Mode_Stored(value.Mode), namespace)
	Parse_Function_Storage_Invariants(value.Function, namespace)
}

// Parse_Unchecked_Stored is the Configuration Left Stored parse_unchecked carries.
type Parse_Unchecked_Stored Configuration_Left_Stored

// Parse_Unchecked_Stored_Invariants fixes opening-delimiter representation.
func Parse_Unchecked_Stored_Invariants(value Parse_Unchecked_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value.Value), DELIMITER_SIZE_MINIMUM, DELIMITER_SIZE_MAXIMUM).
		Ensure()
}

// Pipeline_Append_Validated is the Pipeline Term Validated pipeline_append carries.
type Pipeline_Append_Validated Pipeline_Term_Validated

// Pipeline_Append_Validated_Invariants checks ownership after arena allocation.
func Pipeline_Append_Validated_Invariants(value Pipeline_Append_Validated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value.Value), uint16(COMMAND_NODE_COUNT_MINIMUM),
			uint16(NODE_REFERENCE_MAXIMUM-NODE_COUNT_INCREMENT)).
		Ensure()
}

// PIPELINE_CHAIN_VALIDATED_VALUE_MINIMUM is the floor this call site starts from.
const PIPELINE_CHAIN_VALIDATED_VALUE_MINIMUM = SOURCE_SIZE_MINIMUM + 2

// Pipeline_Chain_Validated is the Pipeline Boundary Validated pipeline_chain carries.
type Pipeline_Chain_Validated Pipeline_Boundary_Validated

// Pipeline_Chain_Validated_Invariants checks ownership after token validation.
func Pipeline_Chain_Validated_Invariants(value Pipeline_Chain_Validated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value.Value), uint16(PIPELINE_CHAIN_VALIDATED_VALUE_MINIMUM),
			uint16(SOURCE_SIZE_MAXIMUM-utf8.CHARACTER_SIZE_MINIMUM)).
		Ensure()
}

// Pipeline_Chain_Count is the Command Node Count pipeline_chain carries.
type Pipeline_Chain_Count Command_Node_Count

// Pipeline_Chain_Count_Invariants bounds the count by one command node.
func Pipeline_Chain_Count_Invariants(value Pipeline_Chain_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(COMMAND_NODE_COUNT_MINIMUM),
			uint16(COMMAND_NODE_COUNT_LIMIT)).
		Ensure()
}

// Pipeline_Chain_Term_Validated is the Pipeline Term Validated pipeline_chain carries.
type Pipeline_Chain_Term_Validated Pipeline_Term_Validated

// Pipeline_Chain_Term_Validated_Invariants checks ownership after arena allocation.
func Pipeline_Chain_Term_Validated_Invariants(value Pipeline_Chain_Term_Validated,
	namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value.Value), uint16(COMMAND_NODE_COUNT_MINIMUM),
			uint16(NODE_REFERENCE_MAXIMUM-NODE_COUNT_INCREMENT)).
		Ensure()
}

// PIPELINE_CLOSE_COUNT_MAXIMUM adds the shortest parenthesised pipeline to the densest open.
const PIPELINE_CLOSE_COUNT_MAXIMUM = OPENED_PIPELINE_COUNT_MAXIMUM + PIPELINE_NODE_COUNT +
	COMMAND_NODE_COUNT + SCALAR_NODE_COUNT

// Pipeline_Close_Count is the Argument Node Count pipeline_close carries.
type Pipeline_Close_Count Argument_Node_Count

// Pipeline_Close_Count_Invariants bounds the count by one complete simple action.
func Pipeline_Close_Count_Invariants(value Pipeline_Close_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value.Value), uint16(ARGUMENT_NODE_COUNT_MINIMUM),
			uint16(PIPELINE_CLOSE_COUNT_MAXIMUM)).
		Ensure()
}

// Pipeline_Declaration_Node_Count is the Pipeline Node Count pipeline_declaration carries.
type Pipeline_Declaration_Node_Count Pipeline_Node_Count

// Pipeline_Declaration_Node_Count_Invariants bounds the count by one pipeline node.
func Pipeline_Declaration_Node_Count_Invariants(value Pipeline_Declaration_Node_Count,
	namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value.Value), uint16(PIPELINE_NODE_COUNT_MINIMUM),
			uint16(BINDING_PACKED_COUNT+ACTION_PIPELINE_NODE_COUNT)).
		Ensure()
}

// Pipeline_Declaration_Store_Count is the Pipeline Node Count pipeline_declaration_store carries.
type Pipeline_Declaration_Store_Count Pipeline_Node_Count

// Pipeline_Declaration_Store_Count_Invariants bounds the count by one pipeline node.
func Pipeline_Declaration_Store_Count_Invariants(value Pipeline_Declaration_Store_Count,
	namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value.Value), uint16(PIPELINE_NODE_COUNT_MINIMUM),
			uint16(BINDING_PACKED_COUNT+ACTION_PIPELINE_NODE_COUNT)).
		Ensure()
}

// Quoted_Escape_End_Validated is the Quoted Span Validated quoted_escape_end carries.
type Quoted_Escape_End_Validated Quoted_Span_Validated

// Quoted_Escape_End_Validated_Invariants checks transport ownership after source validation.
func Quoted_Escape_End_Validated_Invariants(value Quoted_Escape_End_Validated,
	namespace aver.Namespace,
) {
	Quoted_Escape_End_Stored_Invariants(Quoted_Escape_End_Stored(value.Value), namespace)
}

// QUOTED_ESCAPE_END_STORED_END_MINIMUM is the floor this call site starts from.
const QUOTED_ESCAPE_END_STORED_END_MINIMUM = QUOTED_END_MINIMUM + 3

// Quoted_Escape_End_Stored is the Quoted Span Proof Stored quoted_escape_end carries.
type Quoted_Escape_End_Stored Quoted_Span_Proof_Stored

// Quoted_Escape_End_Stored_Invariants fixes quoted-span representation.
func Quoted_Escape_End_Stored_Invariants(value Quoted_Escape_End_Stored, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value.Start), uint16(QUOTED_START_MINIMUM),
			uint16(QUOTED_START_MAXIMUM-SIMPLE_ACTION_SIZE_MINIMUM)).
		Range_Uint16(
			uint16(value.End), uint16(QUOTED_ESCAPE_END_STORED_END_MINIMUM),
			uint16(QUOTED_END_MAXIMUM-utf8.CHARACTER_SIZE_MINIMUM)).
		Ensure()
}

// QUOTED_HEX_ESCAPE_END_VALIDATED_VALUE_MINIMUM is the floor this call site starts from.
const QUOTED_HEX_ESCAPE_END_VALIDATED_VALUE_MINIMUM = ACTION_CONTENT_BOUNDARY_MINIMUM + 2

// Quoted_Hex_Escape_End_Span_Validated is the Quoted Span Validated quoted_hex_escape_end carries.
type Quoted_Hex_Escape_End_Span_Validated Quoted_Span_Validated

// Quoted_Hex_Escape_End_Span_Validated_Invariants checks transport ownership after source
// validation.
func Quoted_Hex_Escape_End_Span_Validated_Invariants(value Quoted_Hex_Escape_End_Span_Validated,
	namespace aver.Namespace,
) {
	Quoted_Hex_Escape_End_Proof_Stored_Invariants(Quoted_Hex_Escape_End_Proof_Stored(
		value.Value), namespace)
}

// QUOTED_HEX_ESCAPE_END_PROOF_STORED_END_MINIMUM is the floor this call site starts from.
const QUOTED_HEX_ESCAPE_END_PROOF_STORED_END_MINIMUM = QUOTED_END_MINIMUM + 4

// Quoted_Hex_Escape_End_Proof_Stored is the Quoted Span Proof Stored quoted_hex_escape_end carries.
type Quoted_Hex_Escape_End_Proof_Stored Quoted_Span_Proof_Stored

// Quoted_Hex_Escape_End_Proof_Stored_Invariants fixes quoted-span representation.
func Quoted_Hex_Escape_End_Proof_Stored_Invariants(value Quoted_Hex_Escape_End_Proof_Stored,
	namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value.Start), uint16(QUOTED_START_MINIMUM),
			uint16(QUOTED_START_MAXIMUM-FIELD_ACTION_SIZE_MINIMUM)).
		Range_Uint16(
			uint16(value.End), uint16(QUOTED_HEX_ESCAPE_END_PROOF_STORED_END_MINIMUM),
			uint16(QUOTED_END_MAXIMUM-utf8.CHARACTER_SIZE_MINIMUM)).
		Ensure()
}

// Quoted_Octal_Escape_End_Span_Validated is the Quoted Span Validated quoted_octal_escape_end
// carries.
type Quoted_Octal_Escape_End_Span_Validated Quoted_Span_Validated

// Quoted_Octal_Escape_End_Span_Validated_Invariants checks transport ownership after source
// validation.
func Quoted_Octal_Escape_End_Span_Validated_Invariants(
	value Quoted_Octal_Escape_End_Span_Validated, namespace aver.Namespace,
) {
	Quoted_Octal_Escape_End_Proof_Stored_Invariants(Quoted_Octal_Escape_End_Proof_Stored(
		value.Value), namespace)
}

// QUOTED_OCTAL_ESCAPE_END_PROOF_STORED_END_MINIMUM is the floor this call site starts from.
const QUOTED_OCTAL_ESCAPE_END_PROOF_STORED_END_MINIMUM = QUOTED_END_MINIMUM + 3

// Quoted_Octal_Escape_End_Proof_Stored is the Quoted Span Proof Stored quoted_octal_escape_end
// carries.
type Quoted_Octal_Escape_End_Proof_Stored Quoted_Span_Proof_Stored

// Quoted_Octal_Escape_End_Proof_Stored_Invariants fixes quoted-span representation.
func Quoted_Octal_Escape_End_Proof_Stored_Invariants(value Quoted_Octal_Escape_End_Proof_Stored,
	namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value.Start), uint16(QUOTED_START_MINIMUM),
			uint16(QUOTED_START_MAXIMUM-SIMPLE_ACTION_SIZE_MINIMUM)).
		Range_Uint16(
			uint16(value.End), uint16(QUOTED_OCTAL_ESCAPE_END_PROOF_STORED_END_MINIMUM),
			uint16(QUOTED_END_MAXIMUM-utf8.CHARACTER_SIZE_MINIMUM)).
		Ensure()
}

// Range_Begin_Program is the Range Program range_begin carries.
type Range_Begin_Program Range_Program

// Range_Begin_Program_Invariants composes that source with arena metadata.
func Range_Begin_Program_Invariants(value Range_Begin_Program, namespace aver.Namespace) {
	Range_Program_Source_Invariants(Range_Program_Source(value.Source), namespace)
	Range_Begin_Document_Invariants(Range_Begin_Document(value.Document), namespace)
	Syntax_Storage_Invariants(value.Syntax, namespace)
}

// RANGE_BEGIN_DOCUMENT_NODE_COUNT_MINIMUM is the floor this call site starts from.
const RANGE_BEGIN_DOCUMENT_NODE_COUNT_MINIMUM = ACTION_PROGRAM_NODE_COUNT_MINIMUM + 1

// Range_Begin_Document is the program document range_begin carries.
type Range_Begin_Document Program_Validated_Document

// Range_Begin_Document_Invariants bounds counts by the shortest action.
func Range_Begin_Document_Invariants(value Range_Begin_Document, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value.Node_Count), uint16(RANGE_BEGIN_DOCUMENT_NODE_COUNT_MINIMUM),
			uint16(RANGE_BEGIN_NODE_COUNT_MAXIMUM)).
		Range_Uint16(
			uint16(value.First_Template), uint16(DOCUMENT_FIRST_TEMPLATE_MINIMUM),
			uint16(RANGE_BEGIN_FIRST_TEMPLATE_MAXIMUM)).
		Range_Uint16(
			uint16(value.Template_Count), uint16(TEMPLATE_COUNT_MINIMUM),
			uint16(TEMPLATE_COUNT_MAXIMUM-
				TEMPLATE_DEFINITION_SIZE_MINIMUM)).
		Ensure()
	aver.Always(uint16(value.Root) == uint16(ROOT_NODE_COUNT),
		"An action program keeps the root in the first syntax arena slot for range_begin.")
}

// Template_Invocation_Argument_Count is the Action Node Count template_invocation_argument carries.
type Template_Invocation_Argument_Count Action_Node_Count

// Template_Invocation_Argument_Count_Invariants bounds the count by one action node.
func Template_Invocation_Argument_Count_Invariants(value Template_Invocation_Argument_Count,
	namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value.Value), uint16(ACTION_NODE_COUNT_MINIMUM),
			uint16(TEMPLATE_INVOCATION_ARGUMENT_MAXIMUM)).
		Ensure()
}

// TEMPLATE_INVOCATION_ARGUMENT_VALIDATED_VALUE_MINIMUM is the floor this call site starts from.
const TEMPLATE_INVOCATION_ARGUMENT_VALIDATED_VALUE_MINIMUM = NODE_REFERENCE_MINIMUM + 2

// Template_Invocation_Argument_Validated is the Node Link Validated template_invocation_argument
// carries.
type Template_Invocation_Argument_Validated Node_Link_Validated

// Template_Invocation_Argument_Validated_Invariants checks ownership after program validation.
func Template_Invocation_Argument_Validated_Invariants(
	value Template_Invocation_Argument_Validated, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value.Value),
			uint16(TEMPLATE_INVOCATION_ARGUMENT_VALIDATED_VALUE_MINIMUM),
			uint16(TEMPLATE_INVOCATION_ARGUMENT_MAXIMUM)).
		Ensure()
}

// Template_Invocation_Argument_Source is the Action Source template_invocation_argument carries.
type Template_Invocation_Argument_Source Action_Source

// Template_Invocation_Argument_Source_Invariants composes the shortest source one action needs.
func Template_Invocation_Argument_Source_Invariants(value Template_Invocation_Argument_Source,
	namespace aver.Namespace) {
	Template_Invocation_Argument_Data_Stored_Invariants(
		Template_Invocation_Argument_Data_Stored(value.Data), namespace)
}

// TEMPLATE_INVOCATION_ARGUMENT_DATA_STORED_VALUE_MINIMUM is the floor this call site starts from.
const TEMPLATE_INVOCATION_ARGUMENT_DATA_STORED_VALUE_MINIMUM = ACTION_BOUNDARY_SIZE_MINIMUM + 11

// Template_Invocation_Argument_Data_Stored is the Action Source Data Stored
// template_invocation_argument carries.
type Template_Invocation_Argument_Data_Stored Action_Source_Data_Stored

// Template_Invocation_Argument_Data_Stored_Invariants bounds data by the shortest action.
func Template_Invocation_Argument_Data_Stored_Invariants(
	value Template_Invocation_Argument_Data_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value.Value),
			TEMPLATE_INVOCATION_ARGUMENT_DATA_STORED_VALUE_MINIMUM,
			SOURCE_SIZE_MAXIMUM).
		Ensure()
}

// Template_Merge_Source is the Action Source template_merge carries.
type Template_Merge_Source Action_Source

// Template_Merge_Source_Invariants composes the shortest source one action needs.
func Template_Merge_Source_Invariants(value Template_Merge_Source, namespace aver.Namespace) {
	Template_Merge_Stored_Invariants(Template_Merge_Stored(value.Data), namespace)
}

// TEMPLATE_MERGE_STORED_VALUE_MINIMUM is the floor this call site starts from.
const TEMPLATE_MERGE_STORED_VALUE_MINIMUM = ACTION_BOUNDARY_SIZE_MINIMUM + 30

// Template_Merge_Stored is the Action Source Data Stored template_merge carries.
type Template_Merge_Stored Action_Source_Data_Stored

// Template_Merge_Stored_Invariants bounds data by the shortest action.
func Template_Merge_Stored_Invariants(value Template_Merge_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value.Value), TEMPLATE_MERGE_STORED_VALUE_MINIMUM,
			SOURCE_SIZE_MAXIMUM).
		Ensure()
}

// Template_Names_Equal_Validated is the Template Reference Validated template_names_equal carries.
type Template_Names_Equal_Validated Template_Reference_Validated

// Template_Names_Equal_Validated_Invariants checks ownership after arena validation.
func Template_Names_Equal_Validated_Invariants(
	value Template_Names_Equal_Validated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(ACTION_NODE_COUNT_MINIMUM),
			uint16(EARLIER_TEMPLATE_REFERENCE_MAXIMUM)).
		Ensure()
}

// TEMPLATE_NAMES_EQUAL_REFERENCE_VALIDATED_VALUE_MINIMUM is the floor this call site starts from.
const TEMPLATE_NAMES_EQUAL_REFERENCE_VALIDATED_VALUE_MINIMUM = ACTION_NODE_COUNT_MINIMUM + 2

// Template_Names_Equal_Reference_Validated is the Template Reference Validated
// template_names_equal carries.
type Template_Names_Equal_Reference_Validated Template_Reference_Validated

// Template_Names_Equal_Reference_Validated_Invariants checks ownership after arena validation.
func Template_Names_Equal_Reference_Validated_Invariants(
	value Template_Names_Equal_Reference_Validated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value.Value),
			uint16(TEMPLATE_NAMES_EQUAL_REFERENCE_VALIDATED_VALUE_MINIMUM),
			uint16(NAMES_EQUAL_REFERENCE_MAXIMUM)).
		Ensure()
}

// Template_Names_Equal_Source is the Action Source template_names_equal carries.
type Template_Names_Equal_Source Action_Source

// Template_Names_Equal_Source_Invariants composes the shortest source one action needs.
func Template_Names_Equal_Source_Invariants(value Template_Names_Equal_Source,
	namespace aver.Namespace) {
	Template_Names_Equal_Data_Stored_Invariants(Template_Names_Equal_Data_Stored(value.Data),
		namespace)
}

// TEMPLATE_NAMES_EQUAL_DATA_STORED_VALUE_MINIMUM is the floor this call site starts from.
const TEMPLATE_NAMES_EQUAL_DATA_STORED_VALUE_MINIMUM = ACTION_BOUNDARY_SIZE_MINIMUM + 30

// Template_Names_Equal_Data_Stored is the Action Source Data Stored template_names_equal carries.
type Template_Names_Equal_Data_Stored Action_Source_Data_Stored

// Template_Names_Equal_Data_Stored_Invariants bounds data by the shortest action.
func Template_Names_Equal_Data_Stored_Invariants(
	value Template_Names_Equal_Data_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value.Value), TEMPLATE_NAMES_EQUAL_DATA_STORED_VALUE_MINIMUM,
			SOURCE_SIZE_MAXIMUM).
		Ensure()
}

// Template_Open_Allocated is the Node Count Allocated template_open carries.
type Template_Open_Allocated Node_Count_Allocated

// Template_Open_Allocated_Invariants keeps count proof attached to parser flow.
func Template_Open_Allocated_Invariants(value Template_Open_Allocated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(ROOT_NODE_COUNT),
			uint16(TEMPLATE_OPEN_COUNT_MAXIMUM)).
		Ensure()
}

// Template_Open_Validated is the Action Span Validated template_open carries.
type Template_Open_Validated Action_Span_Validated

// Template_Open_Validated_Invariants checks transport ownership after delimiter validation.
func Template_Open_Validated_Invariants(value Template_Open_Validated, namespace aver.Namespace,
) {
	Template_Open_Proof_Stored_Invariants(Template_Open_Proof_Stored(value.Value), namespace)
}

// TEMPLATE_OPEN_PROOF_STORED_CONTENT_END_MINIMUM is the floor this call site starts from.
const TEMPLATE_OPEN_PROOF_STORED_CONTENT_END_MINIMUM = ACTION_CONTENT_BOUNDARY_MINIMUM + 5

// TEMPLATE_OPEN_PROOF_STORED_END_MINIMUM is the floor this call site starts from.
const TEMPLATE_OPEN_PROOF_STORED_END_MINIMUM = ACTION_END_MINIMUM + 5

// Template_Open_Proof_Stored is the Action Span Proof Stored template_open carries.
type Template_Open_Proof_Stored Action_Span_Proof_Stored

// Template_Open_Proof_Stored_Invariants fixes action-span representation.
func Template_Open_Proof_Stored_Invariants(value Template_Open_Proof_Stored,
	namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Start), uint16(ACTION_START_MINIMUM),
			uint16(ACTION_START_MAXIMUM-len(KEYWORD_BLOCK))).
		Range_Uint16(uint16(value.Content_Start), uint16(ACTION_CONTENT_BOUNDARY_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM-len(KEYWORD_BLOCK))).
		Range_Uint16(uint16(value.Content_End),
			uint16(TEMPLATE_OPEN_PROOF_STORED_CONTENT_END_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Range_Uint16(uint16(value.End), uint16(TEMPLATE_OPEN_PROOF_STORED_END_MINIMUM),
			uint16(ACTION_END_MAXIMUM)).
		Ensure()
}

// Append_Text_Allocated is the Node Count Allocated append_text carries.
type Append_Text_Allocated Node_Count_Allocated

// Append_Text_Allocated_Invariants keeps count proof attached to parser flow.
func Append_Text_Allocated_Invariants(value Append_Text_Allocated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(ROOT_NODE_COUNT),
			uint16(NODE_COUNT_MAXIMUM-NODE_COUNT_INCREMENT)).
		Ensure()
}

// Control_Branch_Allocate_Allocated is the Node Count Allocated control_branch_allocate carries.
type Control_Branch_Allocate_Allocated Node_Count_Allocated

// Control_Branch_Allocate_Allocated_Invariants keeps count proof attached to parser flow.
func Control_Branch_Allocate_Allocated_Invariants(value Control_Branch_Allocate_Allocated,
	namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(ROOT_NODE_COUNT),
			uint16(CONTROL_BRANCH_ALLOCATE_COUNT_MAXIMUM)).
		Ensure()
}

// Control_Else_Allocated is the Node Count Allocated control_else carries.
type Control_Else_Allocated Node_Count_Allocated

// Control_Else_Allocated_Invariants keeps count proof attached to parser flow.
func Control_Else_Allocated_Invariants(value Control_Else_Allocated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(ROOT_NODE_COUNT),
			uint16(STRAY_ELSE_COUNT_MAXIMUM)).
		Ensure()
}

// CONTROL_ELSE_LIST_ALLOCATED_VALUE_MINIMUM is the floor this call site starts from.
const CONTROL_ELSE_LIST_ALLOCATED_VALUE_MINIMUM = ROOT_NODE_COUNT + 5

// Control_Else_List_Allocated is the Node Count Allocated control_else_list carries.
type Control_Else_List_Allocated Node_Count_Allocated

// Control_Else_List_Allocated_Invariants keeps count proof attached to parser flow.
func Control_Else_List_Allocated_Invariants(value Control_Else_List_Allocated,
	namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value),
			uint16(CONTROL_ELSE_LIST_ALLOCATED_VALUE_MINIMUM),
			uint16(UNCLOSED_ELSE_COUNT_MAXIMUM)).
		Ensure()
}

// Control_Flow_Allocated is the Node Count Allocated control_flow carries.
type Control_Flow_Allocated Node_Count_Allocated

// Control_Flow_Allocated_Invariants keeps count proof attached to parser flow.
func Control_Flow_Allocated_Invariants(value Control_Flow_Allocated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(ROOT_NODE_COUNT),
			uint16(CONTROL_FLOW_COUNT_MAXIMUM)).
		Ensure()
}

// Left_Trimmed_Start_Validated is the Source Cursor Validated left_trimmed_start carries.
type Left_Trimmed_Start_Validated Source_Cursor_Validated

// Left_Trimmed_Start_Validated_Invariants checks ownership after source validation.
func Left_Trimmed_Start_Validated_Invariants(value Left_Trimmed_Start_Validated,
	namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value.Value), uint16(SOURCE_SIZE_MINIMUM),
			uint16(SOURCE_SIZE_MAXIMUM-ACTION_BOUNDARY_SIZE_MINIMUM)).
		Ensure()
}

// Map_Field_Order_Index is the Range Map Entry Index map_field_order carries.
type Map_Field_Order_Index Range_Map_Entry_Index

// Map_Field_Order_Index_Invariants follows the largest map fitting validation storage.
func Map_Field_Order_Index_Invariants(value Map_Field_Order_Index, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value), uint16(bytes.SLICE_SIZE_MINIMUM),
			uint16(RANGE_MAP_ENTRY_INDEX_MAXIMUM)).
		Ensure()
}

// Map_Field_Order_Entry_Index is the Prior Map Entry Index map_field_order carries.
type Map_Field_Order_Entry_Index Prior_Map_Entry_Index

// Map_Field_Order_Entry_Index_Invariants follows every rangeable physical map slot.
func Map_Field_Order_Entry_Index_Invariants(value Map_Field_Order_Entry_Index,
	namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value), uint16(bytes.SLICE_SIZE_MINIMUM),
			uint16(PRIOR_MAP_ENTRY_INDEX_MAXIMUM)).
		Ensure()
}

// Map_Scan_Next_Index is the Prior Map Entry Index map_scan_next carries.
type Map_Scan_Next_Index Prior_Map_Entry_Index

// Map_Scan_Next_Index_Invariants follows every rangeable physical map slot.
func Map_Scan_Next_Index_Invariants(value Map_Scan_Next_Index, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value), uint16(bytes.SLICE_SIZE_MINIMUM),
			uint16(PRIOR_MAP_ENTRY_INDEX_MAXIMUM)).
		Ensure()
}

// Number_Component_Validated is the Number Position Validated number_component carries.
type Number_Component_Validated Number_Position_Validated

// Number_Component_Validated_Invariants keeps numeric-span proof attached to cursor.
func Number_Component_Validated_Invariants(
	value Number_Component_Validated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(ACTION_CONTENT_BOUNDARY_MINIMUM),
			uint16(QUOTED_ESCAPE_POSITION_MAXIMUM)).
		Ensure()
}

// Number_Digit_Validated is the Number Position Validated number_digit carries.
type Number_Digit_Validated Number_Position_Validated

// Number_Digit_Validated_Invariants keeps numeric-span proof attached to cursor.
func Number_Digit_Validated_Invariants(value Number_Digit_Validated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value.Value), uint16(ACTION_CONTENT_BOUNDARY_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM-utf8.CHARACTER_SIZE_MINIMUM)).
		Ensure()
}

// NUMBER_EXPONENT_VALIDATED_VALUE_MINIMUM is the floor this call site starts from.
const NUMBER_EXPONENT_VALIDATED_VALUE_MINIMUM = ACTION_CONTENT_BOUNDARY_MINIMUM + 1

// Number_Exponent_Validated is the Number Position Validated number_exponent carries.
type Number_Exponent_Validated Number_Position_Validated

// Number_Exponent_Validated_Invariants keeps numeric-span proof attached to cursor.
func Number_Exponent_Validated_Invariants(
	value Number_Exponent_Validated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(NUMBER_EXPONENT_VALIDATED_VALUE_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Ensure()
}

// Number_Exponent_Marker_Validated is the Number Position Validated number_exponent_marker carries.
type Number_Exponent_Marker_Validated Number_Position_Validated

// Number_Exponent_Marker_Validated_Invariants keeps numeric-span proof attached to cursor.
func Number_Exponent_Marker_Validated_Invariants(
	value Number_Exponent_Marker_Validated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value.Value), uint16(ACTION_CONTENT_BOUNDARY_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM-utf8.CHARACTER_SIZE_MINIMUM)).
		Ensure()
}

// Number_Prefix_Validated is the Number Position Validated number_prefix carries.
type Number_Prefix_Validated Number_Position_Validated

// Number_Prefix_Validated_Invariants keeps numeric-span proof attached to cursor.
func Number_Prefix_Validated_Invariants(value Number_Prefix_Validated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value.Value), uint16(ACTION_CONTENT_BOUNDARY_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM-utf8.CHARACTER_SIZE_MINIMUM)).
		Ensure()
}

// Number_Sign_End_Validated is the Number Position Validated number_sign_end carries.
type Number_Sign_End_Validated Number_Position_Validated

// Number_Sign_End_Validated_Invariants keeps numeric-span proof attached to cursor.
func Number_Sign_End_Validated_Invariants(
	value Number_Sign_End_Validated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value.Value), uint16(ACTION_CONTENT_BOUNDARY_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM-utf8.CHARACTER_SIZE_MINIMUM)).
		Ensure()
}

// QUOTED_ESCAPE_DIGITS_VALIDATED_VALUE_MINIMUM is the floor this call site starts from.
const QUOTED_ESCAPE_DIGITS_VALIDATED_VALUE_MINIMUM = ACTION_CONTENT_BOUNDARY_MINIMUM + 1

// Quoted_Escape_Digits_Validated is the Quoted Position Validated quoted_escape_digits carries.
type Quoted_Escape_Digits_Validated Quoted_Position_Validated

// Quoted_Escape_Digits_Validated_Invariants keeps quoted-span proof attached to cursor.
func Quoted_Escape_Digits_Validated_Invariants(
	value Quoted_Escape_Digits_Validated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value.Value), uint16(QUOTED_ESCAPE_DIGITS_VALIDATED_VALUE_MINIMUM),
			uint16(QUOTED_ESCAPE_POSITION_MAXIMUM-ACTION_BOUNDARY_SIZE_MINIMUM)).
		Ensure()
}

// RIGHT_TRIMMED_END_CURSOR_VALIDATED_VALUE_MINIMUM is the floor this call site starts from.
const RIGHT_TRIMMED_END_CURSOR_VALIDATED_VALUE_MINIMUM = SOURCE_SIZE_MINIMUM + 1

// Right_Trimmed_End_Cursor_Validated is the Source Cursor Validated right_trimmed_end carries.
type Right_Trimmed_End_Cursor_Validated Source_Cursor_Validated

// Right_Trimmed_End_Cursor_Validated_Invariants checks ownership after source validation.
func Right_Trimmed_End_Cursor_Validated_Invariants(value Right_Trimmed_End_Cursor_Validated,
	namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value.Value),
			uint16(RIGHT_TRIMMED_END_CURSOR_VALIDATED_VALUE_MINIMUM),
			uint16(SOURCE_SIZE_MAXIMUM-utf8.CHARACTER_SIZE_MINIMUM)).
		Ensure()
}

// RIGHT_TRIMMED_END_SOURCE_CURSOR_VALIDATED_VALUE_MINIMUM is the floor this call site starts from.
const RIGHT_TRIMMED_END_SOURCE_CURSOR_VALIDATED_VALUE_MINIMUM = SOURCE_SIZE_MINIMUM + 1

// Right_Trimmed_End_Source_Cursor_Validated is the Source Cursor Validated right_trimmed_end
// carries.
type Right_Trimmed_End_Source_Cursor_Validated Source_Cursor_Validated

// Right_Trimmed_End_Source_Cursor_Validated_Invariants checks ownership after source validation.
func Right_Trimmed_End_Source_Cursor_Validated_Invariants(
	value Right_Trimmed_End_Source_Cursor_Validated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value.Value),
			uint16(RIGHT_TRIMMED_END_SOURCE_CURSOR_VALIDATED_VALUE_MINIMUM),
			uint16(SOURCE_SIZE_MAXIMUM-utf8.CHARACTER_SIZE_MINIMUM)).
		Ensure()
}

// SKIP_SPACE_CURSOR_VALIDATED_VALUE_MINIMUM is the floor this call site starts from.
const SKIP_SPACE_CURSOR_VALIDATED_VALUE_MINIMUM = SOURCE_SIZE_MINIMUM + 5

// Skip_Space_Cursor_Validated is the Source Cursor Validated skip_space carries.
type Skip_Space_Cursor_Validated Source_Cursor_Validated

// Skip_Space_Cursor_Validated_Invariants checks ownership after source validation.
func Skip_Space_Cursor_Validated_Invariants(value Skip_Space_Cursor_Validated,
	namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value),
			uint16(SKIP_SPACE_CURSOR_VALIDATED_VALUE_MINIMUM),
			uint16(SOURCE_SIZE_MAXIMUM)).
		Ensure()
}

// SLICE_POSITION_VALUE_KIND_MINIMUM admits every kind one caller can name a bound with.
const SLICE_POSITION_VALUE_KIND_MINIMUM = VALUE_NIL

// Slice_Position_Value is the Value slice_position carries.
type Slice_Position_Value Value

// Slice_Position_Value_Invariants bounds the kind and the scalar one bound reads.
func Slice_Position_Value_Invariants(value Slice_Position_Value, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value.Kind), uint8(SLICE_POSITION_VALUE_KIND_MINIMUM),
			uint8(VALUE_FUNCTION)).
		Range_Uint64(uint64(value.Real), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Holed_Uint64(
			uint64(value.Imaginary), bits.WORD_64_MINIMUM,
			bits.WORD_64_MAXIMUM, FLOAT_BITS_SUBNORMAL_HOLE,
			FLOAT_BITS_SUBNORMAL_NEXT, FLOAT_BITS_SUBNORMAL_NEXT).
		Range_Int(len(value.Text), OUTPUT_SIZE_MINIMUM, OUTPUT_SIZE_MAXIMUM).
		Range_Int(len(value.Values), bytes.SLICE_SIZE_MINIMUM, VALUE_COUNT_MAXIMUM).
		Range_Int(len(value.Fields), bytes.SLICE_SIZE_MINIMUM, VALUE_COUNT_MAXIMUM).
		Ensure()
}

// Template_Block_Invocation_Count is the Action Node Count template_block_invocation carries.
type Template_Block_Invocation_Count Action_Node_Count

// Template_Block_Invocation_Count_Invariants bounds the count by one action node.
func Template_Block_Invocation_Count_Invariants(value Template_Block_Invocation_Count,
	namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(ACTION_NODE_COUNT_MINIMUM),
			uint16(TEMPLATE_BLOCK_INVOCATION_COUNT_MAXIMUM)).
		Ensure()
}

// TEMPLATE_BLOCK_INVOCATION_VALIDATED_VALUE_MINIMUM is the floor this call site starts from.
const TEMPLATE_BLOCK_INVOCATION_VALIDATED_VALUE_MINIMUM = NODE_REFERENCE_MINIMUM + 2

// Template_Block_Invocation_Validated is the Node Link Validated template_block_invocation carries.
type Template_Block_Invocation_Validated Node_Link_Validated

// Template_Block_Invocation_Validated_Invariants checks ownership after program validation.
func Template_Block_Invocation_Validated_Invariants(value Template_Block_Invocation_Validated,
	namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value),
			uint16(TEMPLATE_BLOCK_INVOCATION_VALIDATED_VALUE_MINIMUM),
			uint16(TEMPLATE_BLOCK_INVOCATION_REFERENCE_MAXIMUM)).
		Ensure()
}

// Variable_Bind_Reference_Validated is the Declaration Reference Validated
// variable_bind_reference carries.
type Variable_Bind_Reference_Validated Declaration_Reference_Validated

// Variable_Bind_Reference_Validated_Invariants checks ownership after program traversal.
func Variable_Bind_Reference_Validated_Invariants(
	value Variable_Bind_Reference_Validated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(COMMAND_NODE_COUNT_MINIMUM),
			uint16(DECLARATION_REFERENCE_MAXIMUM)).
		Ensure()
}

// EXECUTION_REFUSED_STATUS_VALUE_MINIMUM is the floor this call site starts from.
const EXECUTION_REFUSED_STATUS_VALUE_MINIMUM = STATUS_OK + 1

// Execution_Refused_Status is the Status execution_refused carries.
type Execution_Refused_Status Status

// Execution_Refused_Status_Invariants covers every bounded execution outcome.
func Execution_Refused_Status_Invariants(value Execution_Refused_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Holed_Uint8(uint8(value), uint8(EXECUTION_REFUSED_STATUS_VALUE_MINIMUM),
			uint8(STATUS_LIMIT_EXCEEDED), uint8(STATUS_OUTPUT_TOO_LARGE),
			uint8(STATUS_OUTPUT_TOO_LARGE), uint8(STATUS_OUTPUT_TOO_LARGE)).
		Ensure()
}

// INDEX_POSITION_VALUE_KIND_MINIMUM admits every kind one caller can name an element with.
const INDEX_POSITION_VALUE_KIND_MINIMUM = VALUE_NIL

// Index_Position_Value is the Value index_position carries.
type Index_Position_Value Value

// Index_Position_Value_Invariants bounds the kind and the scalar one position reads.
func Index_Position_Value_Invariants(value Index_Position_Value, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value.Kind), uint8(INDEX_POSITION_VALUE_KIND_MINIMUM),
			uint8(VALUE_FUNCTION)).
		Range_Uint64(uint64(value.Real), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Holed_Uint64(
			uint64(value.Imaginary), bits.WORD_64_MINIMUM,
			bits.WORD_64_MAXIMUM, FLOAT_BITS_SUBNORMAL_HOLE,
			FLOAT_BITS_SUBNORMAL_NEXT, FLOAT_BITS_SUBNORMAL_NEXT).
		Range_Int(len(value.Text), OUTPUT_SIZE_MINIMUM, OUTPUT_SIZE_MAXIMUM).
		Range_Int(len(value.Values), bytes.SLICE_SIZE_MINIMUM, VALUE_COUNT_MAXIMUM).
		Range_Int(len(value.Fields), bytes.SLICE_SIZE_MINIMUM, VALUE_COUNT_MAXIMUM).
		Ensure()
}

// Left_Delimiter_Index_Validated is the Source Cursor Validated left_delimiter_index carries.
type Left_Delimiter_Index_Validated Source_Cursor_Validated

// Left_Delimiter_Index_Validated_Invariants checks ownership after source validation.
func Left_Delimiter_Index_Validated_Invariants(value Left_Delimiter_Index_Validated,
	namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Holed_Uint16(
			uint16(value.Value), uint16(SOURCE_SIZE_MINIMUM),
			uint16(TEXT_SPAN_START_MAXIMUM), uint16(ROOT_LIST_HOLE),
			uint16(TERM_ACTION_HOLE), uint16(TERM_ACTION_HOLE)).
		Ensure()
}

// Append_Template_Invocation_Validated is the Action Span Validated append_template_invocation
// carries.
type Append_Template_Invocation_Validated Action_Span_Validated

// Append_Template_Invocation_Validated_Invariants checks transport ownership after delimiter
// validation.
func Append_Template_Invocation_Validated_Invariants(value Append_Template_Invocation_Validated,
	namespace aver.Namespace,
) {
	Append_Template_Invocation_Stored_Invariants(Append_Template_Invocation_Stored(
		value.Value), namespace)
}

// APPEND_TEMPLATE_INVOCATION_STORED_CONTENT_END_MINIMUM is the floor this call site starts from.
const APPEND_TEMPLATE_INVOCATION_STORED_CONTENT_END_MINIMUM = ACTION_CONTENT_BOUNDARY_MINIMUM + 8

// APPEND_TEMPLATE_INVOCATION_STORED_END_MINIMUM is the floor this call site starts from.
const APPEND_TEMPLATE_INVOCATION_STORED_END_MINIMUM = ACTION_END_MINIMUM + 8

// Append_Template_Invocation_Stored is the Action Span Proof Stored append_template_invocation
// carries.
type Append_Template_Invocation_Stored Action_Span_Proof_Stored

// Append_Template_Invocation_Stored_Invariants fixes action-span representation.
func Append_Template_Invocation_Stored_Invariants(value Append_Template_Invocation_Stored,
	namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value.Start), uint16(ACTION_START_MINIMUM),
			uint16(ACTION_START_MAXIMUM-(QUOTED_BOUNDARY_SIZE+
				COMMENT_ACTION_SIZE_MINIMUM))).
		Range_Uint16(
			uint16(value.Content_Start), uint16(ACTION_CONTENT_BOUNDARY_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM-(QUOTED_BOUNDARY_SIZE+
				COMMENT_ACTION_SIZE_MINIMUM))).
		Range_Uint16(uint16(value.Content_End),
			uint16(APPEND_TEMPLATE_INVOCATION_STORED_CONTENT_END_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Range_Uint16(uint16(value.End),
			uint16(APPEND_TEMPLATE_INVOCATION_STORED_END_MINIMUM),
			uint16(ACTION_END_MAXIMUM)).
		Ensure()
}

// Append_Text_Pointer is the List State Storage Pointer append_text carries.
type Append_Text_Pointer *Append_Text_Storage

// Append_Text_Pointer_Invariants composes present cursor storage.
func Append_Text_Pointer_Invariants(value Append_Text_Pointer, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Append_Text_Storage_Invariants(*value, namespace)
}

// Append_Text_Storage is the List State Storage append_text carries.
type Append_Text_Storage List_State_Storage

// Append_Text_Storage_Invariants checks ownership, not transient list references.
func Append_Text_Storage_Invariants(value Append_Text_Storage, namespace aver.Namespace) {
	Append_Text_Stored_Invariants(Append_Text_Stored(value.Value), namespace)
}

// Append_Text_Stored is the List State Value Stored append_text carries.
type Append_Text_Stored List_State_Value_Stored

// Append_Text_Stored_Invariants fixes list-state representation.
func Append_Text_Stored_Invariants(value Append_Text_Stored, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Holed_Uint16(
			uint16(value.List), uint16(ROOT_NODE_COUNT),
			uint16(APPEND_TEXT_LIST_MAXIMUM),
			uint16(ACTION_LIST_HOLE), uint16(ACTION_LIST_HOLE),
			uint16(ACTION_LIST_HOLE)).
		Range_Holed_Uint16(
			uint16(value.Last_Child), uint16(NODE_REFERENCE_MINIMUM),
			uint16(ACTION_TAIL_REFERENCE_MAXIMUM),
			uint16(ROOT_LIST_HOLE), uint16(ROOT_LIST_HOLE),
			uint16(ROOT_LIST_HOLE)).
		Ensure()
}

// Append_Text_Validated is the Text Span Validated append_text carries.
type Append_Text_Validated Text_Span_Validated

// Append_Text_Validated_Invariants checks ownership after parser validation.
func Append_Text_Validated_Invariants(value Append_Text_Validated, namespace aver.Namespace,
) {
	Append_Text_Proof_Stored_Invariants(Append_Text_Proof_Stored(value.Value), namespace)
}

// Append_Text_Proof_Stored is the Text Span Proof Stored append_text carries.
type Append_Text_Proof_Stored Text_Span_Proof_Stored

// Append_Text_Proof_Stored_Invariants fixes text-span representation.
func Append_Text_Proof_Stored_Invariants(value Append_Text_Proof_Stored, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Holed_Uint16(
			uint16(value.Start), uint16(SOURCE_SIZE_MINIMUM),
			uint16(TEXT_SPAN_START_MAXIMUM), uint16(ROOT_LIST_HOLE),
			uint16(TERM_ACTION_HOLE), uint16(TERM_ACTION_HOLE)).
		Range_Uint16(
			uint16(value.End), uint16(SOURCE_SIZE_MINIMUM), uint16(SOURCE_SIZE_MAXIMUM),
		).
		Ensure()
}

// CONTROL_FLOW_TRACKED_VALUE_MAXIMUM is the ceiling this call site stays under.
const CONTROL_FLOW_TRACKED_VALUE_MAXIMUM = CONTROL_DEPTH_MAXIMUM - 1

// Control_Flow_Tracked is the Control Depth Tracked control_flow carries.
type Control_Flow_Tracked Control_Depth_Tracked

// Control_Flow_Tracked_Invariants keeps frame-stack proof attached to depth.
func Control_Flow_Tracked_Invariants(value Control_Flow_Tracked, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(CONTROL_DEPTH_MINIMUM),
			uint16(CONTROL_FLOW_TRACKED_VALUE_MAXIMUM)).
		Ensure()
}

// CONTROL_HEADER_TRACKED_VALUE_MAXIMUM is the ceiling this call site stays under.
const CONTROL_HEADER_TRACKED_VALUE_MAXIMUM = CONTROL_DEPTH_MAXIMUM - 1

// Control_Header_Tracked is the Control Depth Tracked control_header carries.
type Control_Header_Tracked Control_Depth_Tracked

// Control_Header_Tracked_Invariants keeps frame-stack proof attached to depth.
func Control_Header_Tracked_Invariants(value Control_Header_Tracked, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(CONTROL_DEPTH_MINIMUM),
			uint16(CONTROL_HEADER_TRACKED_VALUE_MAXIMUM)).
		Ensure()
}

// TEMPLATE_HEADER_TRACKED_VALUE_MAXIMUM is the ceiling this call site stays under.
const TEMPLATE_HEADER_TRACKED_VALUE_MAXIMUM = CONTROL_DEPTH_MAXIMUM

// Template_Header_Tracked is the Control Depth Tracked template_header carries.
type Template_Header_Tracked Control_Depth_Tracked

// Template_Header_Tracked_Invariants keeps frame-stack proof attached to depth.
func Template_Header_Tracked_Invariants(value Template_Header_Tracked, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(CONTROL_DEPTH_MINIMUM),
			uint16(TEMPLATE_HEADER_TRACKED_VALUE_MAXIMUM)).
		Ensure()
}

// TEMPLATE_DEFINITION_TRACKED_VALUE_MAXIMUM is the deepest nesting beside which one block and
// every end it owes still fit.
const TEMPLATE_DEFINITION_TRACKED_VALUE_MAXIMUM = (SOURCE_SIZE_MAXIMUM -
	BLOCK_DEFINITION_SIZE_MINIMUM) /
	(CONTROL_OPEN_ACTION_SIZE_MINIMUM + CONTROL_END_ACTION_SIZE_MINIMUM)

// Defined_Depth is the nesting one definition leaves open.
type Defined_Depth Control_Depth_Tracked

// Defined_Depth_Invariants adds that definition's own frame to its caller's depth.
func Defined_Depth_Invariants(value Defined_Depth, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(CONTROL_DEPTH_MINIMUM),
			uint16(TEMPLATE_DEFINITION_TRACKED_VALUE_MAXIMUM+CONTROL_DEPTH_INCREMENT)).
		Ensure()
}

// Template_Definition_Tracked is the Control Depth Tracked template_definition carries.
type Template_Definition_Tracked Control_Depth_Tracked

// Template_Definition_Tracked_Invariants keeps frame-stack proof attached to depth.
func Template_Definition_Tracked_Invariants(
	value Template_Definition_Tracked, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(CONTROL_DEPTH_MINIMUM),
			uint16(TEMPLATE_DEFINITION_TRACKED_VALUE_MAXIMUM)).
		Ensure()
}

// CONTROL_OPEN_TRACKED_VALUE_MAXIMUM is the ceiling this call site stays under.
const CONTROL_OPEN_TRACKED_VALUE_MAXIMUM = CONTROL_DEPTH_MAXIMUM

// Control_Open_Tracked is the Control Depth Tracked control_open carries.
type Control_Open_Tracked Control_Depth_Tracked

// Control_Open_Tracked_Invariants keeps frame-stack proof attached to depth.
func Control_Open_Tracked_Invariants(value Control_Open_Tracked, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(CONTROL_DEPTH_MINIMUM),
			uint16(CONTROL_OPEN_TRACKED_VALUE_MAXIMUM)).
		Ensure()
}

// CONTROL_RANGE_TRACKED_VALUE_MAXIMUM is the ceiling this call site stays under.
const CONTROL_RANGE_TRACKED_VALUE_MAXIMUM = CONTROL_DEPTH_MAXIMUM - 1

// Control_Range_Tracked is the Control Depth Tracked control_range carries.
type Control_Range_Tracked Control_Depth_Tracked

// Control_Range_Tracked_Invariants keeps frame-stack proof attached to depth.
func Control_Range_Tracked_Invariants(value Control_Range_Tracked, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(CONTROL_DEPTH_MINIMUM),
			uint16(CONTROL_RANGE_TRACKED_VALUE_MAXIMUM)).
		Ensure()
}

// Control_Token_Kind_Cursor is the Token Cursor control_token_kind carries.
type Control_Token_Kind_Cursor Token_Cursor

// Control_Token_Kind_Cursor_Invariants bounds action position and retained token state.
func Control_Token_Kind_Cursor_Invariants(value Control_Token_Kind_Cursor,
	namespace aver.Namespace) {
	Control_Token_Kind_Stored_Invariants(Control_Token_Kind_Stored(value.Position), namespace)
	Cursor_End_Stored_Invariants(Cursor_End_Stored(value.End), namespace)
	Cursor_Token_Stored_Invariants(Cursor_Token_Stored(value.Token), namespace)
	Cursor_Look_Ahead_Stored_Invariants(Cursor_Look_Ahead_Stored(value.Looked), namespace)
	Cursor_Number_Stored_Invariants(Cursor_Number_Stored(value.Number), namespace)
	Function_Function_Storage_Invariants(value.Function, namespace)
	Cursor_Last_Start_Stored_Invariants(Cursor_Last_Start_Stored(value.Last_Start), namespace)
	Cursor_Has_Last_Stored_Invariants(Cursor_Has_Last_Stored(value.Has_Last), namespace)
}

// CONTROL_TOKEN_KIND_STORED_VALUE_MAXIMUM is the ceiling this call site stays under.
const CONTROL_TOKEN_KIND_STORED_VALUE_MAXIMUM = ACTION_CONTENT_BOUNDARY_MAXIMUM - 1

// Control_Token_Kind_Stored is the Cursor Position Stored control_token_kind carries.
type Control_Token_Kind_Stored Cursor_Position_Stored

// Control_Token_Kind_Stored_Invariants fixes cursor-position representation.
func Control_Token_Kind_Stored_Invariants(value Control_Token_Kind_Stored,
	namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(ACTION_CONTENT_BOUNDARY_MINIMUM),
			uint16(CONTROL_TOKEN_KIND_STORED_VALUE_MAXIMUM)).
		Ensure()
}

// DEFAULT_DELIMITER_MATCHES_VALIDATED_VALUE_MAXIMUM is the ceiling this call site stays under.
const DEFAULT_DELIMITER_MATCHES_VALIDATED_VALUE_MAXIMUM = SOURCE_SIZE_MAXIMUM - 1

// Default_Delimiter_Matches_Validated is the Source Cursor Validated default_delimiter_matches
// carries.
type Default_Delimiter_Matches_Validated Source_Cursor_Validated

// Default_Delimiter_Matches_Validated_Invariants checks ownership after source validation.
func Default_Delimiter_Matches_Validated_Invariants(value Default_Delimiter_Matches_Validated,
	namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(SOURCE_SIZE_MINIMUM),
			uint16(DEFAULT_DELIMITER_MATCHES_VALIDATED_VALUE_MAXIMUM)).
		Ensure()
}

// DELIMITER_MATCHES_VALIDATED_VALUE_MAXIMUM is the ceiling this call site stays under.
const DELIMITER_MATCHES_VALIDATED_VALUE_MAXIMUM = SOURCE_SIZE_MAXIMUM - 1

// Delimiter_Matches_Validated is the Source Cursor Validated delimiter_matches carries.
type Delimiter_Matches_Validated Source_Cursor_Validated

// Delimiter_Matches_Validated_Invariants checks ownership after source validation.
func Delimiter_Matches_Validated_Invariants(value Delimiter_Matches_Validated,
	namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(SOURCE_SIZE_MINIMUM),
			uint16(DELIMITER_MATCHES_VALIDATED_VALUE_MAXIMUM)).
		Ensure()
}

// Identifier_Character_At_Validated is the Identifier Search Validated identifier_character_at
// carries.
type Identifier_Character_At_Validated Identifier_Search_Validated

// Identifier_Character_At_Validated_Invariants checks proof transport ownership.
func Identifier_Character_At_Validated_Invariants(
	value Identifier_Character_At_Validated, namespace aver.Namespace,
) {
	Identifier_Character_At_Span_Invariants(Identifier_Character_At_Span(value.Value),
		namespace)
}

// Identifier_Character_At_Span is the Identifier Search Span identifier_character_at carries.
type Identifier_Character_At_Span Identifier_Search_Span

// Identifier_Character_At_Span_Invariants composes hostile search boundaries.
func Identifier_Character_At_Span_Invariants(value Identifier_Character_At_Span,
	namespace aver.Namespace,
) {
	Identifier_Character_At_Position_Storage_Invariants(
		Identifier_Character_At_Position_Storage(value.Position), namespace)
	Identifier_Character_At_Storage_Invariants(Identifier_Character_At_Storage(value.End),
		namespace)
}

// IDENTIFIER_CHARACTER_AT_STORAGE_VALUE_MAXIMUM is the ceiling this call site stays under.
const IDENTIFIER_CHARACTER_AT_STORAGE_VALUE_MAXIMUM = SOURCE_SIZE_MAXIMUM - 1

// Identifier_Character_At_Storage is the Identifier Search End Storage identifier_character_at
// carries.
type Identifier_Character_At_Storage Identifier_Search_End_Storage

// Identifier_Character_At_Storage_Invariants fixes termination ownership.
func Identifier_Character_At_Storage_Invariants(
	value Identifier_Character_At_Storage, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value),
			uint16(ACTION_CONTENT_BOUNDARY_MINIMUM+utf8.CHARACTER_SIZE_MINIMUM),
			uint16(IDENTIFIER_CHARACTER_AT_STORAGE_VALUE_MAXIMUM)).
		Ensure()
}

// Identifier_Sequence_Search_State_Validated is the Identifier Search Validated
// identifier_sequence carries.
type Identifier_Sequence_Search_State_Validated Identifier_Search_Validated

// Identifier_Sequence_Search_State_Validated_Invariants checks proof transport ownership.
func Identifier_Sequence_Search_State_Validated_Invariants(
	value Identifier_Sequence_Search_State_Validated, namespace aver.Namespace,
) {
	Identifier_Sequence_Search_State_Span_Invariants(Identifier_Sequence_Search_State_Span(
		value.Value), namespace)
}

// Identifier_Sequence_Search_State_Span is the Identifier Search Span identifier_sequence carries.
type Identifier_Sequence_Search_State_Span Identifier_Search_Span

// Identifier_Sequence_Search_State_Span_Invariants composes hostile search boundaries.
func Identifier_Sequence_Search_State_Span_Invariants(value Identifier_Sequence_Search_State_Span,
	namespace aver.Namespace,
) {
	Identifier_Search_Position_Storage_Invariants(value.Position, namespace)
	Identifier_Sequence_End_Storage_Invariants(Identifier_Sequence_End_Storage(value.End),
		namespace)
}

// IDENTIFIER_SEQUENCE_END_STORAGE_VALUE_MAXIMUM is the ceiling this call site stays under.
const IDENTIFIER_SEQUENCE_END_STORAGE_VALUE_MAXIMUM = SOURCE_SIZE_MAXIMUM - 1

// Identifier_Sequence_End_Storage is the Identifier Search End Storage identifier_sequence carries.
type Identifier_Sequence_End_Storage Identifier_Search_End_Storage

// Identifier_Sequence_End_Storage_Invariants fixes termination ownership.
func Identifier_Sequence_End_Storage_Invariants(
	value Identifier_Sequence_End_Storage, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value),
			uint16(ACTION_CONTENT_BOUNDARY_MINIMUM+utf8.CHARACTER_SIZE_MINIMUM),
			uint16(IDENTIFIER_SEQUENCE_END_STORAGE_VALUE_MAXIMUM)).
		Ensure()
}

// Number_Component_Zero_Validated is the Number Span Validated number_component_zero carries.
type Number_Component_Zero_Validated Number_Span_Validated

// Number_Component_Zero_Validated_Invariants checks transport ownership after token bounds
// validation.
func Number_Component_Zero_Validated_Invariants(value Number_Component_Zero_Validated,
	namespace aver.Namespace,
) {
	Number_Component_Zero_Stored_Invariants(Number_Component_Zero_Stored(value.Value),
		namespace)
}

// NUMBER_COMPONENT_ZERO_STORED_START_MAXIMUM is the ceiling this call site stays under.
const NUMBER_COMPONENT_ZERO_STORED_START_MAXIMUM = NUMBER_START_MAXIMUM - 1

// NUMBER_FLOAT_VALID_STORED_START_MAXIMUM is the ceiling this call site stays under.
const NUMBER_FLOAT_VALID_STORED_START_MAXIMUM = NUMBER_START_MAXIMUM - 1

// PIPELINE_OPEN_TRACKED_VALUE_MAXIMUM is the ceiling this call site stays under.
const PIPELINE_OPEN_TRACKED_VALUE_MAXIMUM = PARENTHESIS_DEPTH_MAXIMUM

// Pipeline_Open_Tracked is the Parenthesis Depth Tracked pipeline_open carries.
type Pipeline_Open_Tracked Parenthesis_Depth_Tracked

// Pipeline_Open_Tracked_Invariants checks ownership after frame bounds validation.
func Pipeline_Open_Tracked_Invariants(value Pipeline_Open_Tracked, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(PARENTHESIS_DEPTH_MINIMUM),
			uint16(PIPELINE_OPEN_TRACKED_VALUE_MAXIMUM)).
		Ensure()
}

// Token_Node_Kind_Validated is the Token Validated token_node_kind carries.
type Token_Node_Kind_Validated Token_Validated

// Token_Node_Kind_Validated_Invariants keeps lexer proof attached to token.
func Token_Node_Kind_Validated_Invariants(value Token_Node_Kind_Validated,
	namespace aver.Namespace) {
	Token_Node_Kind_Stored_Invariants(Token_Node_Kind_Stored(value.Value), namespace)
}

// TOKEN_NODE_KIND_STORED_START_MAXIMUM is the ceiling this call site stays under.
const TOKEN_NODE_KIND_STORED_START_MAXIMUM = ACTION_CONTENT_BOUNDARY_MAXIMUM - 1

// Token_Node_Kind_Stored is the Token Proof Stored token_node_kind carries.
type Token_Node_Kind_Stored Token_Proof_Stored

// Token_Node_Kind_Stored_Invariants fixes lexer proof representation.
func Token_Node_Kind_Stored_Invariants(value Token_Node_Kind_Stored, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(
			uint8(value.Kind), uint8(TOKEN_KIND_MINIMUM+TOKEN_KIND_INCREMENT),
			uint8(TOKEN_VARIABLE)).
		Range_Uint16(uint16(value.Start), uint16(ACTION_CONTENT_BOUNDARY_MINIMUM),
			uint16(TOKEN_NODE_KIND_STORED_START_MAXIMUM)).
		Range_Uint16(uint16(value.End),
			uint16(ACTION_CONTENT_BOUNDARY_MINIMUM+utf8.CHARACTER_SIZE_MINIMUM), uint16(
				ACTION_CONTENT_BOUNDARY_MAXIMUM)).
		Ensure()
}

// TOKEN_QUOTED_STORED_START_MAXIMUM is the ceiling this call site stays under.
const TOKEN_QUOTED_STORED_START_MAXIMUM = LEXEME_START_MAXIMUM - 1

// DECLARE_CHARACTER opens one declaration lexeme.
const DECLARE_CHARACTER = ':'

// COMMENT_CHARACTER opens one comment lexeme.
const COMMENT_CHARACTER = '/'

// FIELD_CHARACTER opens one field lexeme.
const FIELD_CHARACTER = '.'

// VARIABLE_CHARACTER opens one variable lexeme.
const VARIABLE_CHARACTER = '$'

// SIGN_CHARACTER is the first character one signed number lexeme admits.
const SIGN_CHARACTER = '+'

// DIGIT_LAST_CHARACTER is the last character one number lexeme admits.
const DIGIT_LAST_CHARACTER = '9'

// Declare_Lexeme is the lexeme one declaration opener carries.
type Declare_Lexeme Lexeme_Proof_Stored

// Declare_Lexeme_Invariants fixes the character a declaration opens with.
func Declare_Lexeme_Invariants(value Declare_Lexeme, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Start), uint16(LEXEME_START_MINIMUM),
			uint16(LEXEME_START_MAXIMUM)).
		Ensure()
	aver.Always(uint8(value.Character) == uint8(DECLARE_CHARACTER),
		"One declaration lexeme opens with the character its caller read.")
}

// Declare_Lexeme_Validated carries that lexeme after source validation.
type Declare_Lexeme_Validated Lexeme_Validated_Fields

// Declare_Lexeme_Validated_Invariants checks transport ownership of that lexeme.
func Declare_Lexeme_Validated_Invariants(
	value Declare_Lexeme_Validated, namespace aver.Namespace,
) {
	Declare_Lexeme_Invariants(Declare_Lexeme(value.Value), namespace)
}

// Comment_Lexeme is the lexeme one comment opener carries.
type Comment_Lexeme Lexeme_Proof_Stored

// Comment_Lexeme_Invariants fixes the character a comment opens with.
func Comment_Lexeme_Invariants(value Comment_Lexeme, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value.Start), uint16(LEXEME_START_MINIMUM),
			uint16(LEXEME_START_MAXIMUM-SIMPLE_ACTION_SIZE_MINIMUM)).
		Ensure()
	aver.Always(uint8(value.Character) == uint8(COMMENT_CHARACTER),
		"One comment lexeme opens with the character its caller read.")
}

// Comment_Lexeme_Validated carries that lexeme after source validation.
type Comment_Lexeme_Validated Lexeme_Validated_Fields

// Comment_Lexeme_Validated_Invariants checks transport ownership of that lexeme.
func Comment_Lexeme_Validated_Invariants(
	value Comment_Lexeme_Validated, namespace aver.Namespace,
) {
	Comment_Lexeme_Invariants(Comment_Lexeme(value.Value), namespace)
}

// Field_Lexeme is the lexeme one field or variable opener carries.
type Field_Lexeme Lexeme_Proof_Stored

// Field_Lexeme_Invariants fixes the characters a field or variable opens with.
func Field_Lexeme_Invariants(value Field_Lexeme, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Start), uint16(LEXEME_START_MINIMUM),
			uint16(LEXEME_START_MAXIMUM)).
		Enum_Uint8(uint8(value.Character), uint8(VARIABLE_CHARACTER),
			uint8(FIELD_CHARACTER)).
		Ensure()
}

// Field_Lexeme_Validated carries that lexeme after source validation.
type Field_Lexeme_Validated Lexeme_Validated_Fields

// Field_Lexeme_Validated_Invariants checks transport ownership of that lexeme.
func Field_Lexeme_Validated_Invariants(
	value Field_Lexeme_Validated, namespace aver.Namespace,
) {
	Field_Lexeme_Invariants(Field_Lexeme(value.Value), namespace)
}

// IDENTIFIER_FIRST_CHARACTER is the first character one identifier lexeme admits.
const IDENTIFIER_FIRST_CHARACTER = 'A'

// IDENTIFIER_LAST_CHARACTER is the last lead byte one identifier lexeme admits.
const IDENTIFIER_LAST_CHARACTER = 0xF4

// Identifier_Lexeme is the lexeme one identifier opener carries.
type Identifier_Lexeme Lexeme_Proof_Stored

// Identifier_Lexeme_Invariants fixes the characters an identifier opens with.
func Identifier_Lexeme_Invariants(value Identifier_Lexeme, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Start), uint16(LEXEME_START_MINIMUM),
			uint16(LEXEME_START_MAXIMUM)).
		Range_Uint8(
			uint8(value.Character), uint8(IDENTIFIER_FIRST_CHARACTER),
			uint8(IDENTIFIER_LAST_CHARACTER-
				FIELD_ACTION_SIZE_MINIMUM)).
		Ensure()
}

// Identifier_Lexeme_Validated carries that lexeme after source validation.
type Identifier_Lexeme_Validated Lexeme_Validated_Fields

// Identifier_Lexeme_Validated_Invariants checks transport ownership of that lexeme.
func Identifier_Lexeme_Validated_Invariants(
	value Identifier_Lexeme_Validated, namespace aver.Namespace,
) {
	Identifier_Lexeme_Invariants(Identifier_Lexeme(value.Value), namespace)
}

// Number_Lexeme_Validated carries that lexeme after source validation.
type Number_Lexeme_Validated Lexeme_Validated_Fields

// Number_Lexeme_Validated_Invariants checks transport ownership of that lexeme.
func Number_Lexeme_Validated_Invariants(
	value Number_Lexeme_Validated, namespace aver.Namespace,
) {
	Number_Lexeme_Invariants(Number_Lexeme(value.Value), namespace)
}

// Word_Token_Kind_Cursor is the Token Cursor word_token_kind carries.
type Word_Token_Kind_Cursor Token_Cursor

// Word_Token_Kind_Cursor_Invariants bounds action position and retained token state.
func Word_Token_Kind_Cursor_Invariants(value Word_Token_Kind_Cursor, namespace aver.Namespace) {
	Word_Token_Kind_Stored_Invariants(Word_Token_Kind_Stored(value.Position), namespace)
	Cursor_End_Stored_Invariants(Cursor_End_Stored(value.End), namespace)
	Cursor_Token_Stored_Invariants(Cursor_Token_Stored(value.Token), namespace)
	Cursor_Look_Ahead_Stored_Invariants(Cursor_Look_Ahead_Stored(value.Looked), namespace)
	Cursor_Number_Stored_Invariants(Cursor_Number_Stored(value.Number), namespace)
	Function_Function_Storage_Invariants(value.Function, namespace)
	Cursor_Last_Start_Stored_Invariants(Cursor_Last_Start_Stored(value.Last_Start), namespace)
	Cursor_Has_Last_Stored_Invariants(Cursor_Has_Last_Stored(value.Has_Last), namespace)
}

// WORD_TOKEN_KIND_STORED_VALUE_MAXIMUM is the ceiling this call site stays under.
const WORD_TOKEN_KIND_STORED_VALUE_MAXIMUM = ACTION_CONTENT_BOUNDARY_MAXIMUM - 1

// Word_Token_Kind_Stored is the Cursor Position Stored word_token_kind carries.
type Word_Token_Kind_Stored Cursor_Position_Stored

// Word_Token_Kind_Stored_Invariants fixes cursor-position representation.
func Word_Token_Kind_Stored_Invariants(value Word_Token_Kind_Stored, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(ACTION_CONTENT_BOUNDARY_MINIMUM),
			uint16(WORD_TOKEN_KIND_STORED_VALUE_MAXIMUM)).
		Ensure()
}

// Word_Token_Kind_Cursor_Cursor is the Word Token Kind Cursor word_token_kind carries.
type Word_Token_Kind_Cursor_Cursor Word_Token_Kind_Cursor

// Word_Token_Kind_Cursor_Cursor_Invariants bounds action position and retained token state.
func Word_Token_Kind_Cursor_Cursor_Invariants(value Word_Token_Kind_Cursor_Cursor,
	namespace aver.Namespace) {
	Word_Token_Kind_Stored_Invariants(Word_Token_Kind_Stored(value.Position), namespace)
	Cursor_End_Stored_Invariants(Cursor_End_Stored(value.End), namespace)
	Word_Token_Kind_Cursor_Stored_Invariants(Word_Token_Kind_Cursor_Stored(value.Token),
		namespace)
	Cursor_Look_Ahead_Stored_Invariants(Cursor_Look_Ahead_Stored(value.Looked), namespace)
	Cursor_Number_Stored_Invariants(Cursor_Number_Stored(value.Number), namespace)
	Function_Function_Storage_Invariants(value.Function, namespace)
	Cursor_Last_Start_Stored_Invariants(Cursor_Last_Start_Stored(value.Last_Start), namespace)
	Cursor_Has_Last_Stored_Invariants(Cursor_Has_Last_Stored(value.Has_Last), namespace)
}

// Word_Token_Kind_Cursor_Stored is the Cursor Token Stored word_token_kind carries.
type Word_Token_Kind_Cursor_Stored Cursor_Token_Stored

// Word_Token_Kind_Cursor_Stored_Invariants fixes look-ahead token representation.
func Word_Token_Kind_Cursor_Stored_Invariants(value Word_Token_Kind_Cursor_Stored,
	namespace aver.Namespace) {
	Token_Stored_Invariants(value.Value, namespace)
}

// Control_Token_Kind_Cursor_Cursor is the Control Token Kind Cursor control_token_kind carries.
type Control_Token_Kind_Cursor_Cursor Control_Token_Kind_Cursor

// Control_Token_Kind_Cursor_Cursor_Invariants bounds action position and retained token state.
func Control_Token_Kind_Cursor_Cursor_Invariants(value Control_Token_Kind_Cursor_Cursor,
	namespace aver.Namespace) {
	Control_Token_Kind_Stored_Invariants(Control_Token_Kind_Stored(value.Position), namespace)
	Cursor_End_Stored_Invariants(Cursor_End_Stored(value.End), namespace)
	Control_Token_Kind_Cursor_Stored_Invariants(
		Control_Token_Kind_Cursor_Stored(value.Token), namespace)
	Cursor_Look_Ahead_Stored_Invariants(Cursor_Look_Ahead_Stored(value.Looked), namespace)
	Cursor_Number_Stored_Invariants(Cursor_Number_Stored(value.Number), namespace)
	Function_Function_Storage_Invariants(value.Function, namespace)
	Cursor_Last_Start_Stored_Invariants(Cursor_Last_Start_Stored(value.Last_Start), namespace)
	Cursor_Has_Last_Stored_Invariants(Cursor_Has_Last_Stored(value.Has_Last), namespace)
}

// Control_Token_Kind_Cursor_Stored is the Cursor Token Stored control_token_kind carries.
type Control_Token_Kind_Cursor_Stored Cursor_Token_Stored

// Control_Token_Kind_Cursor_Stored_Invariants fixes look-ahead token representation.
func Control_Token_Kind_Cursor_Stored_Invariants(value Control_Token_Kind_Cursor_Stored,
	namespace aver.Namespace) {
	Token_Stored_Invariants(value.Value, namespace)
}

// Evaluation_Complete_Pointer is the Workspace Scope Base Storage Pointer evaluation_complete
// carries.
type Evaluation_Complete_Pointer *Evaluation_Complete_Storage

// Evaluation_Complete_Pointer_Invariants checks present workspace storage.
func Evaluation_Complete_Pointer_Invariants(
	value Evaluation_Complete_Pointer, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Evaluation_Complete_Storage_Invariants(*value, namespace)
}

// EVALUATION_COMPLETE_STORAGE_VALUE_MAXIMUM is the ceiling this call site stays under.
const EVALUATION_COMPLETE_STORAGE_VALUE_MAXIMUM = VARIABLE_COUNT_MAXIMUM - 1

// Evaluation_Complete_Storage is the Workspace Scope Base Storage evaluation_complete carries.
type Evaluation_Complete_Storage Workspace_Scope_Base_Storage

// Evaluation_Complete_Storage_Invariants fixes visibility cursor shape.
func Evaluation_Complete_Storage_Invariants(
	value Evaluation_Complete_Storage, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value.Value), uint16(VARIABLE_COUNT_MINIMUM),
			uint16(EVALUATION_COMPLETE_STORAGE_VALUE_MAXIMUM)).
		Ensure()
}

// Word_Token_Kind_Cursor_Cursor_Cursor is the Word Token Kind Cursor Cursor word_token_kind
// carries.
type Word_Token_Kind_Cursor_Cursor_Cursor Word_Token_Kind_Cursor_Cursor

// Word_Token_Kind_Cursor_Cursor_Cursor_Invariants bounds action position and retained token state.
func Word_Token_Kind_Cursor_Cursor_Cursor_Invariants(value Word_Token_Kind_Cursor_Cursor_Cursor,
	namespace aver.Namespace) {
	Word_Token_Kind_Stored_Invariants(Word_Token_Kind_Stored(value.Position), namespace)
	Cursor_End_Stored_Invariants(Cursor_End_Stored(value.End), namespace)
	Word_Token_Kind_Cursor_Cursor_Stored_Invariants(Word_Token_Kind_Cursor_Cursor_Stored(
		value.Token), namespace)
	Cursor_Look_Ahead_Stored_Invariants(Cursor_Look_Ahead_Stored(value.Looked), namespace)
	Cursor_Number_Stored_Invariants(Cursor_Number_Stored(value.Number), namespace)
	Function_Function_Storage_Invariants(value.Function, namespace)
	Cursor_Last_Start_Stored_Invariants(Cursor_Last_Start_Stored(value.Last_Start), namespace)
	Cursor_Has_Last_Stored_Invariants(Cursor_Has_Last_Stored(value.Has_Last), namespace)
}

// Word_Token_Kind_Cursor_Cursor_Stored is the Word Token Kind Cursor Stored word_token_kind
// carries.
type Word_Token_Kind_Cursor_Cursor_Stored Word_Token_Kind_Cursor_Stored

// Word_Token_Kind_Cursor_Cursor_Stored_Invariants fixes look-ahead token representation.
func Word_Token_Kind_Cursor_Cursor_Stored_Invariants(value Word_Token_Kind_Cursor_Cursor_Stored,
	namespace aver.Namespace) {
	Word_Token_Kind_Cursor_Token_Stored_Invariants(
		Word_Token_Kind_Cursor_Token_Stored(value.Value), namespace)
}

// WORD_TOKEN_KIND_CURSOR_TOKEN_STORED_START_MAXIMUM is the ceiling this call site stays under.
const WORD_TOKEN_KIND_CURSOR_TOKEN_STORED_START_MAXIMUM = ACTION_CONTENT_BOUNDARY_MAXIMUM - 1

// WORD_TOKEN_KIND_CURSOR_TOKEN_STORED_END_MAXIMUM is the ceiling this call site stays under.
const WORD_TOKEN_KIND_CURSOR_TOKEN_STORED_END_MAXIMUM = ACTION_CONTENT_BOUNDARY_MAXIMUM - 1

// Word_Token_Kind_Cursor_Token_Stored is the Token Stored word_token_kind carries.
type Word_Token_Kind_Cursor_Token_Stored Token_Stored

// Word_Token_Kind_Cursor_Token_Stored_Invariants preserves lexer-proven token encodings.
func Word_Token_Kind_Cursor_Token_Stored_Invariants(value Word_Token_Kind_Cursor_Token_Stored,
	namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(
			uint8(value.Kind), uint8(TOKEN_KIND_MINIMUM), uint8(TOKEN_KIND_MAXIMUM),
		).
		Range_Uint16(uint16(value.Start), uint16(TOKEN_BOUNDARY_MINIMUM),
			uint16(WORD_TOKEN_KIND_CURSOR_TOKEN_STORED_START_MAXIMUM)).
		Range_Uint16(uint16(value.End), uint16(TOKEN_BOUNDARY_MINIMUM),
			uint16(WORD_TOKEN_KIND_CURSOR_TOKEN_STORED_END_MAXIMUM)).
		Ensure()
}
