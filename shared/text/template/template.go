// Package template executes bounded text/template programs in caller storage.
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

// OUTPUT_SIZE_MAXIMUM follows repository byte-slice boundary.
const OUTPUT_SIZE_MAXIMUM = bytes.SLICE_SIZE_MAXIMUM

// OUTPUT_SIZE_MINIMUM is empty execution output.
const OUTPUT_SIZE_MINIMUM = bytes.SLICE_SIZE_MINIMUM

// GENERATED_SIZE_MAXIMUM retains one source and one bounded derived result.
const GENERATED_SIZE_MAXIMUM = OUTPUT_SIZE_MAXIMUM + SOURCE_SIZE_MAXIMUM

// VALUE_COUNT_MAXIMUM follows maximum caller-owned collection length.
const VALUE_COUNT_MAXIMUM = bytes.SLICE_SIZE_MAXIMUM

// VALUE_COUNT_INCREMENT adds one caller value.
const VALUE_COUNT_INCREMENT = 1

// VALUE_COUNT_UNVALIDATED_MAXIMUM admits first refused collection length.
const VALUE_COUNT_UNVALIDATED_MAXIMUM = VALUE_COUNT_MAXIMUM + VALUE_COUNT_INCREMENT

// MAP_ENTRY_COUNT_INCREMENT adds one caller map field.
const MAP_ENTRY_COUNT_INCREMENT = 1

// MAP_ENTRY_INDEX_MAXIMUM is the final bounded caller map entry.
const MAP_ENTRY_INDEX_MAXIMUM = VALUE_COUNT_MAXIMUM - MAP_ENTRY_COUNT_INCREMENT

// FUNCTION_COUNT_MAXIMUM cannot exceed one identifier per source byte.
const FUNCTION_COUNT_MAXIMUM = SOURCE_SIZE_MAXIMUM

// FUNCTION_COUNT_INCREMENT adds one injected function.
const FUNCTION_COUNT_INCREMENT = 1

// FUNCTION_COUNT_UNVALIDATED_MAXIMUM admits first refused function count.
const FUNCTION_COUNT_UNVALIDATED_MAXIMUM = FUNCTION_COUNT_MAXIMUM + FUNCTION_COUNT_INCREMENT

// FRAME_COUNT_MAXIMUM permits one active frame per syntax node.
const FRAME_COUNT_MAXIMUM = NODE_COUNT_MAXIMUM

// FRAME_COUNT_INCREMENT adds one traversal frame.
const FRAME_COUNT_INCREMENT = 1

// STEP_COUNT_MAXIMUM charges complete syntax plus complete caller data once.
const STEP_COUNT_MAXIMUM = NODE_COUNT_MAXIMUM + VALUE_COUNT_MAXIMUM

// ARGUMENT_COUNT_MAXIMUM permits every syntax node to contribute one argument.
const ARGUMENT_COUNT_MAXIMUM = NODE_COUNT_MAXIMUM

// ARGUMENT_COUNT_MINIMUM is empty callback argument storage.
const ARGUMENT_COUNT_MINIMUM = bytes.SLICE_SIZE_MINIMUM

// ARGUMENT_COUNT_INCREMENT adds one evaluated argument.
const ARGUMENT_COUNT_INCREMENT = 1

// BUILTIN_ARGUMENT_COUNT_NONE is empty builtin call payload.
const BUILTIN_ARGUMENT_COUNT_NONE = ARGUMENT_COUNT_MINIMUM

// BUILTIN_ARGUMENT_COUNT_ONE is one builtin operand.
const BUILTIN_ARGUMENT_COUNT_ONE = BUILTIN_ARGUMENT_COUNT_NONE + ARGUMENT_COUNT_INCREMENT

// BUILTIN_ARGUMENT_COUNT_TWO is one binary builtin operand pair.
const BUILTIN_ARGUMENT_COUNT_TWO = BUILTIN_ARGUMENT_COUNT_ONE + ARGUMENT_COUNT_INCREMENT

// BUILTIN_ARGUMENT_FIRST selects first operand.
const BUILTIN_ARGUMENT_FIRST = bytes.SLICE_SIZE_MINIMUM

// BUILTIN_ARGUMENT_SECOND selects second operand.
const BUILTIN_ARGUMENT_SECOND = BUILTIN_ARGUMENT_FIRST + ARGUMENT_COUNT_INCREMENT

// SLICE_INDEX_START selects low slice boundary.
const SLICE_INDEX_START = bytes.SLICE_SIZE_MINIMUM

// SLICE_INDEX_INCREMENT advances one optional slice boundary.
const SLICE_INDEX_INCREMENT = 1

// SLICE_INDEX_END selects high slice boundary.
const SLICE_INDEX_END = SLICE_INDEX_START + SLICE_INDEX_INCREMENT

// SLICE_INDEX_CAPACITY selects full-slice capacity boundary.
const SLICE_INDEX_CAPACITY = SLICE_INDEX_END + SLICE_INDEX_INCREMENT

// SLICE_INDEX_COUNT stores every standard full-slice boundary.
const SLICE_INDEX_COUNT = SLICE_INDEX_CAPACITY + SLICE_INDEX_INCREMENT

// SLICE_ARGUMENT_COUNT_MINIMUM contains collection operand only.
const SLICE_ARGUMENT_COUNT_MINIMUM = BUILTIN_ARGUMENT_COUNT_ONE

// SLICE_ARGUMENT_COUNT_MAXIMUM contains collection operand and every full-slice index.
const SLICE_ARGUMENT_COUNT_MAXIMUM = SLICE_ARGUMENT_COUNT_MINIMUM + SLICE_INDEX_COUNT

// ESCAPE_HTML selects plain HTML text escaping.
const ESCAPE_HTML Escape_Kind = Escape_Kind(bytes.SLICE_SIZE_MINIMUM)

// ESCAPE_KIND_INCREMENT adds one text transformation kind.
const ESCAPE_KIND_INCREMENT Escape_Kind = 1

// ESCAPE_JS selects JavaScript string escaping.
const ESCAPE_JS Escape_Kind = ESCAPE_HTML + ESCAPE_KIND_INCREMENT

// ESCAPE_URL_QUERY selects URL query component escaping.
const ESCAPE_URL_QUERY Escape_Kind = ESCAPE_JS + ESCAPE_KIND_INCREMENT

// BUILTIN_ORDER_LESS selects strict less-than comparison.
const BUILTIN_ORDER_LESS Builtin_Order_Kind = Builtin_Order_Kind(bytes.SLICE_SIZE_MINIMUM)

// BUILTIN_ORDER_KIND_INCREMENT adds one comparison predicate.
const BUILTIN_ORDER_KIND_INCREMENT Builtin_Order_Kind = Builtin_Order_Kind(
	1,
)

// BUILTIN_ORDER_LESS_EQUAL selects inclusive less-than comparison.
const BUILTIN_ORDER_LESS_EQUAL Builtin_Order_Kind = BUILTIN_ORDER_LESS +
	BUILTIN_ORDER_KIND_INCREMENT

// BUILTIN_ORDER_GREATER selects strict greater-than comparison.
const BUILTIN_ORDER_GREATER Builtin_Order_Kind = BUILTIN_ORDER_LESS_EQUAL +
	BUILTIN_ORDER_KIND_INCREMENT

// BUILTIN_ORDER_GREATER_EQUAL selects inclusive greater-than comparison.
const BUILTIN_ORDER_GREATER_EQUAL Builtin_Order_Kind = BUILTIN_ORDER_GREATER +
	BUILTIN_ORDER_KIND_INCREMENT

// BUILTIN_ORDER_NAME_SIZE follows every standard comparison identifier.
const BUILTIN_ORDER_NAME_SIZE = len("lt")

// TEMPLATE_HEXADECIMAL_DIGITS supplies uppercase escaped byte digits.
const TEMPLATE_HEXADECIMAL_DIGITS = "0123456789ABCDEF"

// HEXADECIMAL_DIGIT_MASK selects one hexadecimal digit.
const HEXADECIMAL_DIGIT_MASK = big.BASE_HEXADECIMAL - big.BASE_BINARY/big.BASE_BINARY

// HEXADECIMAL_DIGIT_COUNT_INCREMENT adds one formatted hexadecimal digit.
const HEXADECIMAL_DIGIT_COUNT_INCREMENT = 1

// URL_QUERY_ESCAPE_SIZE is percent plus two hexadecimal digits.
const URL_QUERY_ESCAPE_SIZE = len("%00")

// QUOTED_CONTROL_BYTE_MINIMUM is the first ASCII control byte.
const QUOTED_CONTROL_BYTE_MINIMUM = bits.WORD_8_MINIMUM

// QUOTED_CONTROL_BYTE_MAXIMUM is the byte immediately before ASCII space.
const QUOTED_CONTROL_BYTE_MAXIMUM = ' ' - utf8.CHARACTER_SIZE_MINIMUM

// NUMBER_SIZE_MAXIMUM is widest formatted integer.
const NUMBER_SIZE_MAXIMUM = strconv.INTEGER_TEXT_SIZE_MAXIMUM

// FLOAT_64_SIGN_MASK selects IEEE binary64 sign field.
const FLOAT_64_SIGN_MASK = big.FLOAT_64_SIGN_MASK

// FLOAT_64_FRACTION_MASK selects IEEE binary64 fraction field.
const FLOAT_64_FRACTION_MASK = big.FLOAT_64_MANTISSA_MASK

// FLOAT_64_EXPONENT_MASK selects IEEE binary64 exponent field.
const FLOAT_64_EXPONENT_MASK = uint64(big.FLOAT_64_EXPONENT_MASK) <<
	big.FLOAT_64_EXPONENT_SHIFT

// INTEGER_64_NEGATIVE_MAGNITUDE_MAXIMUM includes signed minimum magnitude.
const INTEGER_64_NEGATIVE_MAGNITUDE_MAXIMUM = uint64(bits.INTEGER_64_MAXIMUM) +
	bits.CARRY_MAXIMUM

// ORDER_INCREMENT separates adjacent three-way comparison results.
const ORDER_INCREMENT Order = 1

// ORDER_BEFORE places left value before right value.
const ORDER_BEFORE Order = -ORDER_INCREMENT

// ORDER_SAME means both values compare equal.
const ORDER_SAME Order = Order(bytes.SLICE_SIZE_MINIMUM)

// ORDER_AFTER places left value after right value.
const ORDER_AFTER Order = ORDER_SAME + ORDER_INCREMENT

// VARIABLE_COUNT_MAXIMUM follows parser lexical declaration bound.
const VARIABLE_COUNT_MAXIMUM = SYNTAX_VARIABLE_COUNT_MAXIMUM

// VARIABLE_COUNT_INCREMENT adds one lexical binding.
const VARIABLE_COUNT_INCREMENT = 1

// DECLARATION_COUNT_NONE means pipeline binds no variable.
const DECLARATION_COUNT_NONE Declaration_Count = Declaration_Count(bytes.SLICE_SIZE_MINIMUM)

// DECLARATION_COUNT_INCREMENT adds one pipeline binding.
const DECLARATION_COUNT_INCREMENT Declaration_Count = Declaration_Count(
	1,
)

// DECLARATION_COUNT_ONE means pipeline binds one variable.
const DECLARATION_COUNT_ONE Declaration_Count = DECLARATION_COUNT_NONE +
	DECLARATION_COUNT_INCREMENT

// DECLARATION_COUNT_TWO means range pipeline binds key and value.
const DECLARATION_COUNT_TWO Declaration_Count = DECLARATION_COUNT_ONE +
	DECLARATION_COUNT_INCREMENT

// DECLARATION_REFERENCE_COUNT stores maximum declaration arity.
const DECLARATION_REFERENCE_COUNT = int(DECLARATION_COUNT_TWO)

// SYNTAX_FIELD is sole caller syntax pointer slot.
const SYNTAX_FIELD = bytes.SLICE_SIZE_MINIMUM

// SYNTAX_FIELD_COUNT fixes hostile syntax pointer storage.
const SYNTAX_FIELD_COUNT = SYNTAX_FIELD + STORAGE_FIELD_COUNT_INCREMENT

// WORKSPACE_FIELD is sole caller execution pointer slot.
const WORKSPACE_FIELD = bytes.SLICE_SIZE_MINIMUM

// WORKSPACE_FIELD_COUNT fixes hostile execution pointer storage.
const WORKSPACE_FIELD_COUNT = WORKSPACE_FIELD + STORAGE_FIELD_COUNT_INCREMENT

// STATUS_OK means complete output committed.
const STATUS_OK Status = Status(bytes.SLICE_SIZE_MINIMUM)

// STATUS_INCREMENT adds one bounded execution outcome.
const STATUS_INCREMENT Status = 1

// STATUS_OUTPUT_TOO_SMALL preserves short caller destination.
const STATUS_OUTPUT_TOO_SMALL Status = STATUS_OK + STATUS_INCREMENT

// STATUS_OUTPUT_TOO_LARGE refuses output beyond package bound.
const STATUS_OUTPUT_TOO_LARGE Status = STATUS_OUTPUT_TOO_SMALL + STATUS_INCREMENT

// STATUS_PROGRAM_INVALID refuses forged or stale syntax.
const STATUS_PROGRAM_INVALID Status = STATUS_OUTPUT_TOO_LARGE + STATUS_INCREMENT

// STATUS_WORKSPACE_INVALID reports absent caller execution state.
const STATUS_WORKSPACE_INVALID Status = STATUS_PROGRAM_INVALID + STATUS_INCREMENT

// STATUS_VALUE_INVALID refuses forged caller data.
const STATUS_VALUE_INVALID Status = STATUS_WORKSPACE_INVALID + STATUS_INCREMENT

// STATUS_FUNCTION_INVALID refuses malformed function registry or callback result.
const STATUS_FUNCTION_INVALID Status = STATUS_VALUE_INVALID + STATUS_INCREMENT

// STATUS_FUNCTION_ABSENT reports unresolved identifier.
const STATUS_FUNCTION_ABSENT Status = STATUS_FUNCTION_INVALID + STATUS_INCREMENT

// STATUS_EXECUTION_INVALID reports valid syntax unsupported by current values.
const STATUS_EXECUTION_INVALID Status = STATUS_FUNCTION_ABSENT + STATUS_INCREMENT

// STATUS_LIMIT_EXCEEDED stops data-amplified work at caller-visible bound.
const STATUS_LIMIT_EXCEEDED Status = STATUS_EXECUTION_INVALID + STATUS_INCREMENT

// FAILURE_STATUS_VALIDATED_FIELD selects one caller-visible non-success status.
const FAILURE_STATUS_VALIDATED_FIELD = bytes.SLICE_SIZE_MINIMUM

// FAILURE_STATUS_VALIDATED_FIELD_COUNT fixes failure transport shape.
const FAILURE_STATUS_VALIDATED_FIELD_COUNT = FAILURE_STATUS_VALIDATED_FIELD +
	STORAGE_FIELD_COUNT_INCREMENT

// Failure_Status_Validated_Value retains failure-status proof storage.
type Failure_Status_Validated_Value uint8

// Failure_Status_Validated_Value_Invariants preserves the status encoding.
func Failure_Status_Validated_Value_Invariants(
	value Failure_Status_Validated_Value, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint8(
			uint8(value), uint8(STATUS_OUTPUT_TOO_SMALL), uint8(STATUS_LIMIT_EXCEEDED),
		).
		Ensure()
}

// Failure_Status_Validated carries one status proven non-success by dispatch.
type Failure_Status_Validated struct {
	// Value keeps failure proof attached to execution status.
	Value Failure_Status_Proof_Stored
}

// Failure_Status_Validated_Invariants checks failure ownership.
func Failure_Status_Validated_Invariants(
	value Failure_Status_Validated, namespace aver.Namespace,
) {
	Failure_Status_Proof_Stored_Invariants(value.Value, namespace)
}

// EXECUTION_FAILURE_POSITION_FIELD selects one proven diagnostic position.
const EXECUTION_FAILURE_POSITION_FIELD = bytes.SLICE_SIZE_MINIMUM

// EXECUTION_FAILURE_POSITION_FIELD_COUNT fixes position proof shape.
const EXECUTION_FAILURE_POSITION_FIELD_COUNT = EXECUTION_FAILURE_POSITION_FIELD +
	STORAGE_FIELD_COUNT_INCREMENT

// Execution_Failure_Position_Value retains diagnostic-position proof storage.
type Execution_Failure_Position_Value uint16

// Execution_Failure_Position_Value_Invariants preserves position encoding.
func Execution_Failure_Position_Value_Invariants(
	value Execution_Failure_Position_Value, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(SOURCE_SIZE_MINIMUM), uint16(SOURCE_SIZE_MAXIMUM),
		).
		Ensure()
}

// Execution_Failure_Position carries one validated diagnostic position by value.
type Execution_Failure_Position struct {
	// Value keeps program proof attached to diagnostic position.
	Value Execution_Failure_Position_Proof_Stored
}

// Execution_Failure_Position_Invariants checks proof ownership.
func Execution_Failure_Position_Invariants(
	value Execution_Failure_Position, namespace aver.Namespace,
) {
	Execution_Failure_Position_Proof_Stored_Invariants(value.Value, namespace)
}

// FUNCTION_STATUS_OK accepts callback result.
const FUNCTION_STATUS_OK Function_Status = Function_Status(STATUS_OK)

// FUNCTION_STATUS_INCREMENT adds one callback outcome.
const FUNCTION_STATUS_INCREMENT Function_Status = Function_Status(
	1,
)

// FUNCTION_STATUS_INVALID rejects callback result.
const FUNCTION_STATUS_INVALID Function_Status = FUNCTION_STATUS_OK + FUNCTION_STATUS_INCREMENT

// VALUE_NIL is absent template data.
const VALUE_NIL Value_Kind = Value_Kind(bytes.SLICE_SIZE_MINIMUM)

// VALUE_KIND_INCREMENT adds one closed-union payload kind.
const VALUE_KIND_INCREMENT Value_Kind = 1

// VALUE_BOOLEAN is one truth value.
const VALUE_BOOLEAN Value_Kind = VALUE_NIL + VALUE_KIND_INCREMENT

// VALUE_INTEGER is signed binary integer.
const VALUE_INTEGER Value_Kind = VALUE_BOOLEAN + VALUE_KIND_INCREMENT

// VALUE_UNSIGNED is unsigned binary integer.
const VALUE_UNSIGNED Value_Kind = VALUE_INTEGER + VALUE_KIND_INCREMENT

// VALUE_FLOAT is exact IEEE binary64 bits.
const VALUE_FLOAT Value_Kind = VALUE_UNSIGNED + VALUE_KIND_INCREMENT

// VALUE_COMPLEX is two exact IEEE binary64 components.
const VALUE_COMPLEX Value_Kind = VALUE_FLOAT + VALUE_KIND_INCREMENT

// VALUE_TEXT is borrowed untrusted text.
const VALUE_TEXT Value_Kind = VALUE_COMPLEX + VALUE_KIND_INCREMENT

// VALUE_SEQUENCE is ordered caller-owned data.
const VALUE_SEQUENCE Value_Kind = VALUE_TEXT + VALUE_KIND_INCREMENT

// VALUE_OBJECT is named caller-owned fields.
const VALUE_OBJECT Value_Kind = VALUE_SEQUENCE + VALUE_KIND_INCREMENT

// VALUE_MAP is ordered-at-execution caller-owned entries.
const VALUE_MAP Value_Kind = VALUE_OBJECT + VALUE_KIND_INCREMENT

// VALUE_FUNCTION is injected callable data.
const VALUE_FUNCTION Value_Kind = VALUE_MAP + VALUE_KIND_INCREMENT

// VALUE_SCALAR_REAL is sole scalar or complex real word.
const VALUE_SCALAR_REAL = bytes.SLICE_SIZE_MINIMUM

// VALUE_SCALAR_WORD_COUNT_INCREMENT adds one binary64 component word.
const VALUE_SCALAR_WORD_COUNT_INCREMENT = 1

// VALUE_SCALAR_IMAGINARY is complex imaginary word.
const VALUE_SCALAR_IMAGINARY = VALUE_SCALAR_REAL + VALUE_SCALAR_WORD_COUNT_INCREMENT

// VALUE_SCALAR_WORD_COUNT stores both complex components.
const VALUE_SCALAR_WORD_COUNT = VALUE_SCALAR_IMAGINARY + VALUE_SCALAR_WORD_COUNT_INCREMENT

// VALUE_PAYLOAD_FIELD is sole storage slot for each union payload.
const VALUE_PAYLOAD_FIELD = bytes.SLICE_SIZE_MINIMUM

// VALUE_PAYLOAD_FIELD_COUNT fixes each union payload shape.
const VALUE_PAYLOAD_FIELD_COUNT = VALUE_PAYLOAD_FIELD + STORAGE_FIELD_COUNT_INCREMENT

// Value_Kind selects one reflection-free payload.
type Value_Kind uint8

// Value_Kind_Invariants covers complete closed union.
func Value_Kind_Invariants(value Value_Kind, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), uint8(VALUE_NIL), uint8(VALUE_FUNCTION)).
		Ensure()
}

// Range_Value_Kind is one iterable kind with an index-derived value.
type Range_Value_Kind uint8

// Range_Value_Kind_Invariants excludes nil and separately ordered maps.
func Range_Value_Kind_Invariants(
	value Range_Value_Kind, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(VALUE_INTEGER), uint8(VALUE_UNSIGNED),
			uint8(VALUE_SEQUENCE),
		).
		Ensure()
}

// Composite_Kind is one recursively formatted collection kind.
type Composite_Kind uint8

// Composite_Kind_Invariants covers every collection formatter shape.
func Composite_Kind_Invariants(value Composite_Kind, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(VALUE_SEQUENCE), uint8(VALUE_OBJECT),
			uint8(VALUE_MAP),
		).
		Ensure()
}

// Escape_Kind selects one standard text transformation.
type Escape_Kind uint8

// Escape_Kind_Invariants lists complete builtin escape domain.
func Escape_Kind_Invariants(value Escape_Kind, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(ESCAPE_HTML), uint8(ESCAPE_JS),
			uint8(ESCAPE_URL_QUERY),
		).
		Ensure()
}

// Builtin_Order_Kind selects one standard binary comparison.
type Builtin_Order_Kind uint8

// Builtin_Order_Kind_Invariants lists the complete comparison domain.
func Builtin_Order_Kind_Invariants(value Builtin_Order_Kind, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_4_Uint8(
			uint8(value), uint8(BUILTIN_ORDER_LESS), uint8(BUILTIN_ORDER_LESS_EQUAL),
			uint8(BUILTIN_ORDER_GREATER), uint8(BUILTIN_ORDER_GREATER_EQUAL),
		).
		Ensure()
}

// BUILTIN_ORDER_NAME_GREATER is the greater comparison prefix.
const BUILTIN_ORDER_NAME_GREATER = 'g'

// BUILTIN_ORDER_NAME_LESS is the less comparison prefix.
const BUILTIN_ORDER_NAME_LESS = 'l'

// Builtin_Order_Name_First selects less or greater comparison family.
type Builtin_Order_Name_First byte

// Builtin_Order_Name_First_Invariants rejects non-builtin prefixes.
func Builtin_Order_Name_First_Invariants(
	value Builtin_Order_Name_First, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(BUILTIN_ORDER_NAME_GREATER),
			uint8(BUILTIN_ORDER_NAME_LESS),
		).
		Ensure()
}

// BUILTIN_ORDER_NAME_EQUAL is the inclusive comparison suffix.
const BUILTIN_ORDER_NAME_EQUAL = 'e'

// BUILTIN_ORDER_NAME_THAN is the strict comparison suffix.
const BUILTIN_ORDER_NAME_THAN = 't'

// Builtin_Order_Name_Second selects strict or inclusive comparison.
type Builtin_Order_Name_Second byte

// Builtin_Order_Name_Second_Invariants rejects non-builtin suffixes.
func Builtin_Order_Name_Second_Invariants(
	value Builtin_Order_Name_Second, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(BUILTIN_ORDER_NAME_EQUAL),
			uint8(BUILTIN_ORDER_NAME_THAN),
		).
		Ensure()
}

// Builtin_Order_Name_Validated carries one comparison identifier proven by dispatch.
type Builtin_Order_Name_Validated struct {
	// First can only select the less or greater family.
	First Builtin_Order_Name_First
	// Second can only select strict or inclusive comparison.
	Second Builtin_Order_Name_Second
}

// Builtin_Order_Name_Validated_Invariants fixes complete comparison-name transport.
func Builtin_Order_Name_Validated_Invariants(
	value Builtin_Order_Name_Validated, namespace aver.Namespace,
) {
	Builtin_Order_Name_First_Invariants(value.First, namespace)
	Builtin_Order_Name_Second_Invariants(value.Second, namespace)
}

// Quoted_Control_Byte is one ASCII control requiring hexadecimal quoting.
type Quoted_Control_Byte uint8

// Quoted_Control_Byte_Invariants follows the ASCII control block.
func Quoted_Control_Byte_Invariants(value Quoted_Control_Byte, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(
			uint8(value), uint8(QUOTED_CONTROL_BYTE_MINIMUM),
			uint8(QUOTED_CONTROL_BYTE_MAXIMUM),
		).
		Ensure()
}

// Format_Verb is one hostile printf conversion byte.
type Format_Verb byte

// Format_Verb_Invariants keeps format refusal constant work.
func Format_Verb_Invariants(value Format_Verb, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), bits.WORD_8_MINIMUM, bits.WORD_8_MAXIMUM).
		Ensure()
}

// Integer_Format_Base is one supported positional integer radix.
type Integer_Format_Base uint8

// Integer_Format_Base_Invariants admits only implemented integer radices.
func Integer_Format_Base_Invariants(
	value Integer_Format_Base, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_4_Uint8(
			uint8(value), uint8(INTEGER_FORMAT_BASE_BINARY),
			uint8(INTEGER_FORMAT_BASE_OCTAL), uint8(INTEGER_FORMAT_BASE_DECIMAL),
			uint8(INTEGER_FORMAT_BASE_HEXADECIMAL),
		).
		Ensure()
}

// INTEGER_FORMAT_BASE_BINARY selects base two.
const INTEGER_FORMAT_BASE_BINARY Integer_Format_Base = big.BASE_BINARY

// INTEGER_FORMAT_BASE_OCTAL selects base eight.
const INTEGER_FORMAT_BASE_OCTAL Integer_Format_Base = big.BASE_OCTAL

// INTEGER_FORMAT_BASE_DECIMAL selects base ten.
const INTEGER_FORMAT_BASE_DECIMAL Integer_Format_Base = strconv.DECIMAL_BASE

// INTEGER_FORMAT_BASE_HEXADECIMAL selects base sixteen.
const INTEGER_FORMAT_BASE_HEXADECIMAL Integer_Format_Base = big.BASE_HEXADECIMAL

// Integer_Format_Uppercase selects uppercase hexadecimal digits.
type Integer_Format_Uppercase bool

// Integer_Format_Uppercase_Invariants covers both hexadecimal alphabets.
func Integer_Format_Uppercase_Invariants(
	value Integer_Format_Uppercase, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Integer formatting uses uppercase hexadecimal digits.").
		Ensure()
}

// Format_Width is one bounded printf field width.
type Format_Width uint16

// Format_Width_Invariants prevents padding beyond output capacity.
func Format_Width_Invariants(value Format_Width, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(bytes.SLICE_SIZE_MINIMUM),
			uint16(OUTPUT_SIZE_MAXIMUM),
		).
		Ensure()
}

// Format_Zero_Padding selects zero instead of space padding.
type Format_Zero_Padding bool

// Format_Zero_Padding_Invariants covers both standard padding modes.
func Format_Zero_Padding_Invariants(
	value Format_Zero_Padding, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Printf field uses zero padding.").
		Ensure()
}

// Format_Alternate selects one alternate printf representation.
type Format_Alternate bool

// Format_Alternate_Invariants covers ordinary and alternate forms.
func Format_Alternate_Invariants(
	value Format_Alternate, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Printf field uses its alternate representation.").
		Ensure()
}

// Format_Count is one bounded printf directive byte count.
type Format_Count uint16

// Format_Count_Invariants reserves the leading percent byte outside directive input.
func Format_Count_Invariants(value Format_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(bytes.SLICE_SIZE_MINIMUM),
			uint16(FORMAT_COUNT_MAXIMUM),
		).
		Ensure()
}

// FORMAT_COUNT_MAXIMUM reserves the leading percent byte outside directive input.
const FORMAT_COUNT_MAXIMUM = SOURCE_SIZE_MAXIMUM - len("%")

// Format_Source is one printf suffix after its leading percent byte.
type Format_Source []byte

// Format_Source_Invariants follows the remaining bounded format bytes.
func Format_Source_Invariants(value Format_Source, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), bytes.SLICE_SIZE_MINIMUM, FORMAT_COUNT_MAXIMUM,
		).
		Ensure()
}

// Boolean is template truth storage.
type Boolean bool

// Boolean_Invariants covers both truth states.
func Boolean_Invariants(value Boolean, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Template Boolean is true.").
		Ensure()
}

// Integer is complete signed template integer.
type Integer int64

// Integer_Invariants covers complete signed width.
func Integer_Invariants(value Integer, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), bits.INTEGER_64_MINIMUM, bits.INTEGER_64_MAXIMUM).
		Ensure()
}

// Unsigned is complete unsigned template integer.
type Unsigned uint64

// Unsigned_Invariants covers complete unsigned width.
func Unsigned_Invariants(value Unsigned, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Float stores every IEEE binary64 bit pattern.
type Float uint64

// Float_Invariants includes finite, infinite, and NaN encodings.
func Float_Invariants(value Float, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Complex_Real stores real binary64 bits.
type Complex_Real uint64

// Complex_Real_Invariants includes every binary64 encoding.
func Complex_Real_Invariants(value Complex_Real, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Complex_Imaginary stores imaginary binary64 bits.
type Complex_Imaginary uint64

// Complex_Imaginary_Invariants includes every binary64 encoding.
func Complex_Imaginary_Invariants(value Complex_Imaginary, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Complex stores exact real and imaginary binary64 bits.
type Complex struct {
	// Real stores real component encoding.
	Real Complex_Real
	// Imaginary stores imaginary component encoding.
	Imaginary Complex_Imaginary
}

// Complex_Invariants composes both independent binary64 components.
func Complex_Invariants(value Complex, namespace aver.Namespace) {
	Complex_Real_Invariants(value.Real, namespace)
	Complex_Imaginary_Invariants(value.Imaginary, namespace)
}

// Text is borrowed caller bytes.
type Text []byte

// Text_Invariants bounds every scalar and field name.
func Text_Invariants(value Text, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), bytes.SLICE_SIZE_MINIMUM, bytes.SLICE_SIZE_MAXIMUM,
		).
		Ensure()
}

// NUMBER_TEXT_VALIDATED_FIELD selects one lexer-proven numeric token or component.
const NUMBER_TEXT_VALIDATED_FIELD = bytes.SLICE_SIZE_MINIMUM

// NUMBER_TEXT_VALIDATED_FIELD_COUNT fixes numeric text transport shape.
const NUMBER_TEXT_VALIDATED_FIELD_COUNT = NUMBER_TEXT_VALIDATED_FIELD +
	STORAGE_FIELD_COUNT_INCREMENT

// Number_Text_Validated carries one lexer-proven numeric token by value.
type Number_Text_Validated_Value []byte

// Number_Text_Validated_Value_Invariants admits text after numeric syntax validation.
func Number_Text_Validated_Value_Invariants(
	value Number_Text_Validated_Value, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), IDENTIFIER_SIZE_MINIMUM, ACTION_CONTENT_SIZE_MAXIMUM).
		Ensure()
}

// Number_Text_Validated carries one lexer-proven numeric token by value.
type Number_Text_Validated struct {
	// Value keeps syntax proof attached to numeric text.
	Value Number_Text_Proof_Stored
}

// Number_Text_Validated_Invariants checks ownership after numeric syntax validation.
func Number_Text_Validated_Invariants(
	value Number_Text_Validated, namespace aver.Namespace,
) {
	Number_Text_Proof_Stored_Invariants(value.Value, namespace)
}

// PARSED_FLOAT_VALIDATED_FIELD selects one accepted binary64 encoding.
const PARSED_FLOAT_VALIDATED_FIELD = bytes.SLICE_SIZE_MINIMUM

// PARSED_FLOAT_VALIDATED_FIELD_COUNT fixes parsed encoding transport shape.
const PARSED_FLOAT_VALIDATED_FIELD_COUNT = PARSED_FLOAT_VALIDATED_FIELD +
	STORAGE_FIELD_COUNT_INCREMENT

// Parsed_Float_Validated_Value retains parsed-float proof storage.
type Parsed_Float_Validated_Value uint64

// Parsed_Float_Validated_Value_Invariants preserves every float bit.
func Parsed_Float_Validated_Value_Invariants(
	value Parsed_Float_Validated_Value, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Parsed_Float_Validated carries one accepted binary64 encoding by value.
type Parsed_Float_Validated struct {
	// Value keeps conversion proof attached to float bits.
	Value Parsed_Float_Proof_Stored
}

// Parsed_Float_Validated_Invariants checks ownership after numeric conversion.
func Parsed_Float_Validated_Invariants(
	value Parsed_Float_Validated, namespace aver.Namespace,
) {
	Parsed_Float_Proof_Stored_Invariants(value.Value, namespace)
}

// Field_Name is borrowed object metadata.
type Field_Name []byte

// Field_Name_Invariants admits absent map metadata and bounded object names.
func Field_Name_Invariants(value Field_Name, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), bytes.SLICE_SIZE_MINIMUM, bytes.SLICE_SIZE_MAXIMUM,
		).
		Ensure()
}

// Values is caller-owned sequence storage.
type Values []Value

// Values_Invariants bounds hostile collection traversal.
func Values_Invariants(value Values, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), bytes.SLICE_SIZE_MINIMUM,
			VALUE_COUNT_UNVALIDATED_MAXIMUM,
		).
		Ensure()
}

// ARGUMENTS_STAGED_FIELD selects one slice bounded by execution workspace.
const ARGUMENTS_STAGED_FIELD = bytes.SLICE_SIZE_MINIMUM

// ARGUMENTS_STAGED_FIELD_COUNT fixes callback argument transport shape.
const ARGUMENTS_STAGED_FIELD_COUNT = ARGUMENTS_STAGED_FIELD + STORAGE_FIELD_COUNT_INCREMENT

// Arguments_Staged carries active callback arguments by value.
type Arguments_Staged_Value []Value

// Arguments_Staged_Value_Invariants admits a header after workspace bounds validation.
func Arguments_Staged_Value_Invariants(
	value Arguments_Staged_Value, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), ARGUMENT_COUNT_MINIMUM, ARGUMENT_COUNT_MAXIMUM).
		Ensure()
}

// Arguments_Staged carries active callback arguments by value.
type Arguments_Staged struct {
	// Value keeps workspace proof attached to callback arguments.
	Value Arguments_Proof_Stored
}

// Arguments_Staged_Invariants checks ownership after workspace bounds enforcement.
func Arguments_Staged_Invariants(value Arguments_Staged, namespace aver.Namespace) {
	Arguments_Proof_Stored_Invariants(value.Value, namespace)
}

// Fields is caller-owned object or map storage.
type Fields []Field

// Fields_Invariants bounds hostile field traversal.
func Fields_Invariants(value Fields, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), bytes.SLICE_SIZE_MINIMUM,
			VALUE_COUNT_UNVALIDATED_MAXIMUM,
		).
		Ensure()
}

// BOUNDED_FIELDS_FIELD selects fields accepted by active-value validation.
const BOUNDED_FIELDS_FIELD = bytes.SLICE_SIZE_MINIMUM

// BOUNDED_FIELDS_FIELD_COUNT fixes bounded field proof shape.
const BOUNDED_FIELDS_FIELD_COUNT = BOUNDED_FIELDS_FIELD + STORAGE_FIELD_COUNT_INCREMENT

// Bounded_Fields carries one active-value-bounded field slice by value.
type Bounded_Fields struct {
	// Value keeps traversal proof attached to caller fields.
	Value Fields_Proof_Stored
}

// Bounded_Fields_Invariants checks proof ownership.
func Bounded_Fields_Invariants(value Bounded_Fields, namespace aver.Namespace) {
	Fields_Proof_Stored_Invariants(value.Value, namespace)
}

// MAP_KEYS_VALIDATED_FIELD selects fields whose map keys were individually checked.
const MAP_KEYS_VALIDATED_FIELD = bytes.SLICE_SIZE_MINIMUM

// MAP_KEYS_VALIDATED_FIELD_COUNT fixes map-key proof shape.
const MAP_KEYS_VALIDATED_FIELD_COUNT = MAP_KEYS_VALIDATED_FIELD + STORAGE_FIELD_COUNT_INCREMENT

// Map_Keys_Validated carries individually sortable map keys by value.
type Map_Keys_Validated struct {
	// Value keeps key proof attached to caller fields.
	Value Fields_Proof_Stored
}

// Map_Keys_Validated_Invariants checks proof ownership.
func Map_Keys_Validated_Invariants(value Map_Keys_Validated, namespace aver.Namespace) {
	Fields_Proof_Stored_Invariants(value.Value, namespace)
}

// Map_Keys_Monotonic reports strict ascending or descending caller key order.
type Map_Keys_Monotonic bool

// Map_Keys_Monotonic_Invariants covers ordered and unordered caller maps.
func Map_Keys_Monotonic_Invariants(
	value Map_Keys_Monotonic, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Caller map keys have strict monotonic order.").
		Ensure()
}

// MAP_FIELDS_VALIDATED_FIELD selects one deeply validated map field slice.
const MAP_FIELDS_VALIDATED_FIELD = bytes.SLICE_SIZE_MINIMUM

// MAP_FIELDS_VALIDATED_FIELD_COUNT fixes map field proof shape.
const MAP_FIELDS_VALIDATED_FIELD_COUNT = MAP_FIELDS_VALIDATED_FIELD +
	STORAGE_FIELD_COUNT_INCREMENT

// Map_Fields_Validated carries map fields whose names and keys were checked.
type Map_Fields_Validated struct {
	// Value keeps deep proof attached to caller fields.
	Value Fields_Proof_Stored
}

// Map_Fields_Validated_Invariants checks proof ownership.
func Map_Fields_Validated_Invariants(
	value Map_Fields_Validated, namespace aver.Namespace,
) {
	Fields_Proof_Stored_Invariants(value.Value, namespace)
}

// MAP_KEY_VALIDATED_FIELD selects one deeply validated sortable map key.
const MAP_KEY_VALIDATED_FIELD = bytes.SLICE_SIZE_MINIMUM

// MAP_KEY_VALIDATED_FIELD_COUNT fixes map key proof shape.
const MAP_KEY_VALIDATED_FIELD_COUNT = MAP_KEY_VALIDATED_FIELD + STORAGE_FIELD_COUNT_INCREMENT

// Map_Key_Validated carries one sortable map key by value.
type Map_Key_Validated struct {
	// Value keeps ordering proof attached to the map key.
	Value Value
}

// Map_Key_Validated_Invariants checks proof ownership.
func Map_Key_Validated_Invariants(value Map_Key_Validated, namespace aver.Namespace) {
	Value_Invariants(value.Value, namespace)
}

// Function_Name is borrowed callable identifier.
type Function_Name []byte

// Function_Name_Invariants bounds hostile registry names.
func Function_Name_Invariants(value Function_Name, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), bytes.SLICE_SIZE_MINIMUM, SOURCE_SIZE_MAXIMUM,
		).
		Ensure()
}

// FUNCTION_NAME_VALIDATED_FIELD selects one lexer-proven identifier.
const FUNCTION_NAME_VALIDATED_FIELD = bytes.SLICE_SIZE_MINIMUM

// FUNCTION_NAME_VALIDATED_FIELD_COUNT fixes identifier transport shape.
const FUNCTION_NAME_VALIDATED_FIELD_COUNT = FUNCTION_NAME_VALIDATED_FIELD +
	STORAGE_FIELD_COUNT_INCREMENT

// Function_Name_Validated carries one lexer-proven identifier by value.
type Function_Name_Validated_Value []byte

// Function_Name_Validated_Value_Invariants admits a name after lexical validation.
func Function_Name_Validated_Value_Invariants(
	value Function_Name_Validated_Value, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), IDENTIFIER_SIZE_MINIMUM, ACTION_CONTENT_SIZE_MAXIMUM).
		Ensure()
}

// Function_Name_Validated carries one lexer-proven identifier by value.
type Function_Name_Validated struct {
	// Value keeps token proof attached to the function name.
	Value Function_Name_Proof_Stored
}

// Function_Name_Validated_Invariants checks ownership after token validation.
func Function_Name_Validated_Invariants(
	value Function_Name_Validated, namespace aver.Namespace,
) {
	Function_Name_Proof_Stored_Invariants(value.Value, namespace)
}

// Function_Call executes one injected bounded function.
type Function_Call func(input Function_Input) (result Function_Result)

// Function_Call_Invariants rejects behavior that cannot be invoked.
func Function_Call_Invariants(value Function_Call, namespace aver.Namespace) {
	aver.Always(
		value != nil,
		"Injected function behavior must exist before construction or invocation.",
	)
}

// Value_Kind_Storage retains one active union selector.
type Value_Kind_Storage_Value uint8

// Value_Kind_Storage_Value_Invariants admits any selector after union validation.
func Value_Kind_Storage_Value_Invariants(
	value Value_Kind_Storage_Value, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), bits.WORD_8_MINIMUM, bits.WORD_8_MAXIMUM).
		Ensure()
}

// Value_Kind_Storage retains one active union selector.
type Value_Kind_Storage struct {
	// Value keeps the selector inside union storage.
	Value Value_Kind_Storage_Value
}

// Value_Kind_Storage_Invariants fixes selector storage shape.
func Value_Kind_Storage_Invariants(
	value Value_Kind_Storage, namespace aver.Namespace,
) {
	Value_Kind_Storage_Value_Invariants(value.Value, namespace)
}

// Value_Scalar_Real keeps the primary word's invariant identity independent.
type Value_Scalar_Real uint64

// Value_Scalar_Real_Invariants admits every caller bit pattern.
func Value_Scalar_Real_Invariants(value Value_Scalar_Real, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Value_Scalar_Imaginary keeps the secondary word's invariant identity independent.
type Value_Scalar_Imaginary uint64

// Value_Scalar_Imaginary_Invariants admits every caller bit pattern.
func Value_Scalar_Imaginary_Invariants(
	value Value_Scalar_Imaginary, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Value_Scalar stores one scalar or both complex component words.
type Value_Scalar struct {
	// Real cannot share invariant identity with the imaginary word.
	Real Value_Scalar_Real
	// Imaginary cannot share invariant identity with the real word.
	Imaginary Value_Scalar_Imaginary
}

// Value_Scalar_Invariants fixes closed scalar storage shape.
func Value_Scalar_Invariants(value Value_Scalar, namespace aver.Namespace) {
	Value_Scalar_Real_Invariants(value.Real, namespace)
	Value_Scalar_Imaginary_Invariants(value.Imaginary, namespace)
}

// Text_Storage retains one borrowed text payload.
type Text_Storage_Value []byte

// Text_Storage_Value_Invariants admits any header after active-union validation.
func Text_Storage_Value_Invariants(
	value Text_Storage_Value, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), OUTPUT_SIZE_MINIMUM, SOURCE_SIZE_UNVALIDATED_MAXIMUM).
		Ensure()
}

// Text_Storage retains one borrowed text payload.
type Text_Storage struct {
	// Value keeps borrowed text inside union storage.
	Value Text_Storage_Value
}

// Text_Storage_Invariants fixes text payload shape.
func Text_Storage_Invariants(value Text_Storage, namespace aver.Namespace) {
	Text_Storage_Value_Invariants(value.Value, namespace)
}

// Values_Storage retains one borrowed sequence payload.
type Values_Storage_Value []Value

// Values_Storage_Value_Invariants admits any header after active-union validation.
func Values_Storage_Value_Invariants(
	value Values_Storage_Value, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), bytes.SLICE_SIZE_MINIMUM, VALUE_COUNT_UNVALIDATED_MAXIMUM).
		Ensure()
}

// Values_Storage retains one borrowed sequence payload.
type Values_Storage struct {
	// Value keeps borrowed sequence inside union storage.
	Value Values_Storage_Value
}

// Values_Storage_Invariants fixes sequence payload shape.
func Values_Storage_Invariants(value Values_Storage, namespace aver.Namespace) {
	Values_Storage_Value_Invariants(value.Value, namespace)
}

// Fields_Storage retains one borrowed object or map payload.
type Fields_Storage_Value []Field

// Fields_Storage_Value_Invariants admits any header after active-union validation.
func Fields_Storage_Value_Invariants(
	value Fields_Storage_Value, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), bytes.SLICE_SIZE_MINIMUM, VALUE_COUNT_UNVALIDATED_MAXIMUM).
		Ensure()
}

// Fields_Storage retains one borrowed object or map payload.
type Fields_Storage struct {
	// Value keeps borrowed fields inside union storage.
	Value Fields_Storage_Value
}

// Fields_Storage_Invariants fixes field payload shape.
func Fields_Storage_Invariants(value Fields_Storage, namespace aver.Namespace) {
	Fields_Storage_Value_Invariants(value.Value, namespace)
}

// Function_Storage retains one callable payload.
type Function_Storage_Value func(input Function_Input) (result Function_Result)

// Function_Storage retains one callable payload.
type Function_Storage struct {
	// Value keeps injected behavior inside union storage.
	Value Function_Storage_Value
}

// Function_Storage_Invariants fixes callable payload shape.
func Function_Storage_Invariants(value Function_Storage, _ aver.Namespace) {
	present := value.Value != nil
	aver.Always(present == present, "Function payload retains caller behavior.")
}

// Value_Kind_Stored separates union shape from active selector validation.
type Value_Kind_Stored interface{}

// Value_Kind_Stored_Invariants fixes union-selector representation.
func Value_Kind_Stored_Invariants(value Value_Kind_Stored, _ aver.Namespace) {
	_, valid := value.(Value_Kind_Storage)
	aver.Always(valid == (value != nil), "Value selector has expected storage type.")
}

// Value_Scalar_Stored separates union shape from active scalar validation.
type Value_Scalar_Stored interface{}

// Value_Scalar_Stored_Invariants fixes union-scalar representation.
func Value_Scalar_Stored_Invariants(value Value_Scalar_Stored, _ aver.Namespace) {
	_, valid := value.(Value_Scalar)
	aver.Always(valid == (value != nil), "Value scalar has expected storage type.")
}

// Value_Text_Stored separates union shape from active text validation.
type Value_Text_Stored interface{}

// Value_Text_Stored_Invariants fixes union-text representation.
func Value_Text_Stored_Invariants(value Value_Text_Stored, _ aver.Namespace) {
	_, valid := value.(Text_Storage)
	aver.Always(valid == (value != nil), "Value text has expected storage type.")
}

// Value_Values_Stored separates union shape from active sequence validation.
type Value_Values_Stored interface{}

// Value_Values_Stored_Invariants fixes union-sequence representation.
func Value_Values_Stored_Invariants(value Value_Values_Stored, _ aver.Namespace) {
	_, valid := value.(Values_Storage)
	aver.Always(valid == (value != nil), "Value sequence has expected storage type.")
}

// Value_Fields_Stored separates union shape from active field validation.
type Value_Fields_Stored interface{}

// Value_Fields_Stored_Invariants fixes union-field representation.
func Value_Fields_Stored_Invariants(value Value_Fields_Stored, _ aver.Namespace) {
	_, valid := value.(Fields_Storage)
	aver.Always(valid == (value != nil), "Value fields have expected storage type.")
}

// Value_Function_Stored separates union shape from active callable validation.
type Value_Function_Stored interface{}

// Value_Function_Stored_Invariants fixes union-callable representation.
func Value_Function_Stored_Invariants(value Value_Function_Stored, _ aver.Namespace) {
	_, valid := value.(Function_Storage)
	aver.Always(valid == (value != nil), "Value function has expected storage type.")
}

// Value_Fields is reflection-free template data before active validation.
type Value_Fields struct {
	// Kind selects active payload.
	Kind Value_Kind_Storage
	// Scalar stores Boolean, numeric, or complex bits.
	Scalar Value_Scalar
	// Text stores borrowed scalar payload.
	Text Text_Storage
	// Values stores sequence payload.
	Values Values_Storage
	// Fields stores object or map payload.
	Fields Fields_Storage
	// Function stores callable payload.
	Function Function_Storage
}

// Value_Fields_Invariants admits hostile union storage before active validation.
func Value_Fields_Invariants(value Value_Fields, namespace aver.Namespace) {
	Value_Kind_Storage_Invariants(value.Kind, namespace)
	Value_Scalar_Invariants(value.Scalar, namespace)
	Text_Storage_Invariants(value.Text, namespace)
	Values_Storage_Invariants(value.Values, namespace)
	Fields_Storage_Invariants(value.Fields, namespace)
	Function_Storage_Invariants(value.Function, namespace)
}

// Value is reflection-free template data after active validation.
type Value Value_Fields

// Value_Invariants keeps closed-union storage bounded before active validation.
func Value_Invariants(value Value, namespace aver.Namespace) {
	Value_Kind_Stored_Invariants(Value_Kind_Stored(value.Kind), namespace)
	Value_Scalar_Stored_Invariants(Value_Scalar_Stored(value.Scalar), namespace)
	Value_Text_Stored_Invariants(Value_Text_Stored(value.Text), namespace)
	Value_Values_Stored_Invariants(Value_Values_Stored(value.Values), namespace)
	Value_Fields_Stored_Invariants(Value_Fields_Stored(value.Fields), namespace)
	Function_Storage_Invariants(value.Function, namespace)
}

// Value_Unvalidated carries hostile callback or nested caller data.
type Value_Unvalidated Value

// Value_Unvalidated_Invariants admits every kind until active validation.
func Value_Unvalidated_Invariants(
	value Value_Unvalidated, namespace aver.Namespace,
) {
	Value_Kind_Stored_Invariants(Value_Kind_Stored(value.Kind), namespace)
	Value_Scalar_Stored_Invariants(Value_Scalar_Stored(value.Scalar), namespace)
	Value_Text_Stored_Invariants(Value_Text_Stored(value.Text), namespace)
	Value_Values_Stored_Invariants(Value_Values_Stored(value.Values), namespace)
	Value_Fields_Stored_Invariants(Value_Fields_Stored(value.Fields), namespace)
	Value_Function_Stored_Invariants(Value_Function_Stored(value.Function), namespace)
}

// Field_Key stores one map key without repeating Value in Field invariant chain.
type Field_Key struct {
	// Data is map key.
	Data Value
}

// Field_Key_Invariants composes bounded key header.
func Field_Key_Invariants(value Field_Key, namespace aver.Namespace) {
	Value_Invariants(value.Data, namespace)
}

// Field_Value stores one object member or map value.
type Field_Value struct {
	// Data is selected value.
	Data Value
}

// Field_Value_Invariants composes bounded member header.
func Field_Value_Invariants(value Field_Value, namespace aver.Namespace) {
	Value_Invariants(value.Data, namespace)
}

// Field is one object member or map entry.
type Field struct {
	// Name identifies object member; empty selects Key for maps.
	Name Field_Name
	// Key stores map key.
	Key Field_Key
	// Value stores member value.
	Value Field_Value
}

// Field_Invariants composes metadata and both independent value roles.
func Field_Invariants(value Field, namespace aver.Namespace) {
	Field_Name_Invariants(value.Name, namespace)
	Field_Key_Invariants(value.Key, namespace)
	Field_Value_Invariants(value.Value, namespace)
}

// Function is one injected named callable.
type Function struct {
	// Name is exact template identifier.
	Name Function_Name
	// Call receives bounded borrowed arguments.
	Call Function_Call
}

// Function_Invariants bounds registry metadata.
func Function_Invariants(value Function, namespace aver.Namespace) {
	Function_Name_Invariants(value.Name, namespace)
	Function_Call_Invariants(value.Call, namespace)
}

// Functions is caller-owned function registry.
type Functions []Function

// Functions_Invariants bounds registry search.
func Functions_Invariants(value Functions, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), bytes.SLICE_SIZE_MINIMUM,
			FUNCTION_COUNT_UNVALIDATED_MAXIMUM,
		).
		Ensure()
}

// FUNCTIONS_VALIDATED_FIELD selects one registry accepted by registry_valid.
const FUNCTIONS_VALIDATED_FIELD = bytes.SLICE_SIZE_MINIMUM

// FUNCTIONS_VALIDATED_FIELD_COUNT fixes trusted registry transport shape.
const FUNCTIONS_VALIDATED_FIELD_COUNT = FUNCTIONS_VALIDATED_FIELD +
	STORAGE_FIELD_COUNT_INCREMENT

// Functions_Validated_Value retains one registry after deep validation.
type Functions_Validated_Value []Function

// Functions_Validated_Value_Invariants admits a header after registry validation.
func Functions_Validated_Value_Invariants(
	value Functions_Validated_Value, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), bytes.SLICE_SIZE_MINIMUM, FUNCTION_COUNT_MAXIMUM).
		Ensure()
}

// Functions_Validated carries one deeply checked function registry.
type Functions_Validated struct {
	// Value keeps registry proof attached to caller functions.
	Value Functions_Proof_Stored
}

// Functions_Validated_Invariants checks transport ownership after registry validation.
func Functions_Validated_Invariants(value Functions_Validated, namespace aver.Namespace) {
	Functions_Proof_Stored_Invariants(value.Value, namespace)
}

// Function_Input borrows arguments only during callback.
type Function_Input struct {
	// Arguments are evaluated left-to-right.
	Arguments Values
}

// Function_Input_Invariants bounds callback argument view.
func Function_Input_Invariants(value Function_Input, namespace aver.Namespace) {
	Values_Invariants(value.Arguments, namespace)
}

// Function_Status accepts or rejects callback result.
type Function_Status uint8

// Function_Status_Invariants covers callback outcomes.
func Function_Status_Invariants(value Function_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(FUNCTION_STATUS_OK),
			uint8(FUNCTION_STATUS_INVALID),
		).
		Ensure()
}

// Function_Result returns one value or scalar refusal.
type Function_Result struct {
	// Value is callback output on success.
	Value Value
	// Status classifies callback completion.
	Status Function_Status
}

// Function_Result_Invariants composes callback output.
func Function_Result_Invariants(value Function_Result, namespace aver.Namespace) {
	Value_Invariants(value.Value, namespace)
	Function_Status_Invariants(value.Status, namespace)
}

// Syntax_Storage admits absent caller syntax state.
type Syntax_Storage struct {
	// Value keeps absence explicit inside hostile input.
	Value Syntax_Workspace_Pointer
}

// Syntax_Storage_Invariants fixes pointer input shape.
func Syntax_Storage_Invariants(value Syntax_Storage, namespace aver.Namespace) {
	Syntax_Workspace_Pointer_Invariants(value.Value, namespace)
}

// Syntax_Input carries hostile optional parser storage.
type Syntax_Input struct {
	// State points at caller-owned parser storage when present.
	State Syntax_Storage
}

// Syntax_Input_Invariants fixes pointer input shape.
func Syntax_Input_Invariants(value Syntax_Input, namespace aver.Namespace) {
	Syntax_Storage_Invariants(value.State, namespace)
}

// Compile_Input is bounded source and parser policy.
type Compile_Input struct {
	// Source is hostile borrowed template bytes.
	Source Source_Unvalidated
	// Left is optional custom opening delimiter.
	Left Left_Delimiter_Unvalidated
	// Right is optional custom closing delimiter.
	Right Right_Delimiter_Unvalidated
	// Mode selects retained comments.
	Mode Mode
	// Workspace owns syntax nodes.
	Workspace Syntax_Input
}

// Compile_Input_Invariants composes hostile compile boundary.
func Compile_Input_Invariants(value Compile_Input, namespace aver.Namespace) {
	Source_Unvalidated_Invariants(value.Source, namespace)
	Left_Delimiter_Unvalidated_Invariants(value.Left, namespace)
	Right_Delimiter_Unvalidated_Invariants(value.Right, namespace)
	Mode_Invariants(value.Mode, namespace)
	Syntax_Input_Invariants(value.Workspace, namespace)
}

// Program retains borrowed source and caller syntax storage.
type Program struct {
	// Source is validated borrowed source.
	Source Source
	// Document identifies populated syntax.
	Document Document
	// Syntax points at nodes backing Document.
	Syntax Syntax_Storage
}

// Program_Invariants composes bounded headers without dereferencing hostile state.
func Program_Invariants(value Program, namespace aver.Namespace) {
	Source_Invariants(value.Source, namespace)
	Document_Invariants(value.Document, namespace)
	Syntax_Storage_Invariants(value.Syntax, namespace)
}

// PROGRAM_VALIDATED_FIELD selects one program proven safe by program_valid.
const PROGRAM_VALIDATED_FIELD = bytes.SLICE_SIZE_MINIMUM

// PROGRAM_VALIDATED_FIELD_COUNT fixes trusted program transport shape.
const PROGRAM_VALIDATED_FIELD_COUNT = PROGRAM_VALIDATED_FIELD + STORAGE_FIELD_COUNT_INCREMENT

// Program_Validated_Root retains checked root storage.
type Program_Validated_Root uint16

// Program_Validated_Root_Invariants preserves the canonical root encoding.
func Program_Validated_Root_Invariants(value Program_Validated_Root, _ aver.Namespace) {
	aver.Always(
		uint16(value) == uint16(ROOT_NODE_COUNT),
		"Validated program root occupies the first syntax arena slot.",
	)
}

// Program_Validated_Node_Count retains checked arena-count storage.
type Program_Validated_Node_Count uint16

// Program_Validated_Node_Count_Invariants preserves the arena count.
func Program_Validated_Node_Count_Invariants(
	value Program_Validated_Node_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(ROOT_NODE_COUNT), uint16(NODE_COUNT_MAXIMUM),
		).
		Ensure()
}

// Program_Validated_First_Template retains checked definition storage.
type Program_Validated_First_Template uint16

// Program_Validated_First_Template_Invariants preserves the first definition.
func Program_Validated_First_Template_Invariants(
	value Program_Validated_First_Template, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(DOCUMENT_FIRST_TEMPLATE_MINIMUM),
			uint16(DOCUMENT_FIRST_TEMPLATE_MAXIMUM),
		).
		Ensure()
}

// Program_Validated_Template_Count retains checked definition-count storage.
type Program_Validated_Template_Count uint16

// Program_Validated_Template_Count_Invariants preserves the definition count.
func Program_Validated_Template_Count_Invariants(
	value Program_Validated_Template_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(TEMPLATE_COUNT_MINIMUM),
			uint16(TEMPLATE_COUNT_MAXIMUM),
		).
		Ensure()
}

// Program_Validated_Document stores metadata already checked by program_valid.
type Program_Validated_Document struct {
	// Root retains the checked root encoding.
	Root Program_Validated_Root
	// Node_Count retains the checked populated arena size.
	Node_Count Program_Validated_Node_Count
	// First_Template retains the checked definition-chain opening.
	First_Template Program_Validated_First_Template
	// Template_Count retains the checked definition count.
	Template_Count Program_Validated_Template_Count
}

// Program_Validated_Document_Invariants fixes proof transport without repeating validation.
func Program_Validated_Document_Invariants(
	value Program_Validated_Document, namespace aver.Namespace,
) {
	Program_Validated_Root_Invariants(value.Root, namespace)
	Program_Validated_Node_Count_Invariants(value.Node_Count, namespace)
	Program_Validated_First_Template_Invariants(value.First_Template, namespace)
	Program_Validated_Template_Count_Invariants(value.Template_Count, namespace)
}

// Program_Validated carries one deeply checked program through execution.
type Program_Validated Program_Validated_Fields

// Program_Validated_Invariants checks transport ownership after deep validation.
func Program_Validated_Invariants(value Program_Validated, namespace aver.Namespace) {
	Source_Invariants(value.Source, namespace)
	Program_Document_Stored_Invariants(Program_Document_Stored(value.Document), namespace)
	Syntax_Storage_Invariants(value.Syntax, namespace)
}

// NODE_VALIDATED_FIELD selects one syntax node checked by program_valid.
const NODE_VALIDATED_FIELD = bytes.SLICE_SIZE_MINIMUM

// NODE_VALIDATED_FIELD_COUNT fixes trusted node transport shape.
const NODE_VALIDATED_FIELD_COUNT = NODE_VALIDATED_FIELD + STORAGE_FIELD_COUNT_INCREMENT

// Node_Validated carries one deeply checked syntax record through execution.
type Node_Validated struct {
	// Value keeps deep proof attached to the node.
	Value Node_Stored
}

// Node_Validated_Invariants checks transport ownership after deep validation.
func Node_Validated_Invariants(value Node_Validated, namespace aver.Namespace) {
	Node_Stored_Invariants(value.Value, namespace)
}

// Node_Optional_Validated carries an absent sentinel or one deeply checked chain.
type Node_Optional_Validated struct {
	// Value keeps optional-chain proof attached to the node.
	Value Node_Stored
}

// Node_Optional_Validated_Invariants checks trusted optional-node transport shape.
func Node_Optional_Validated_Invariants(
	value Node_Optional_Validated, namespace aver.Namespace,
) {
	Node_Stored_Invariants(value.Value, namespace)
}

// Node_Location binds one validated syntax node to its arena slot.
type Node_Location struct {
	// Reference selects the node's caller-owned arena slot.
	Reference Node_Reference
	// Node is the syntax record validated for that slot.
	Node Node_Stored
}

// Node_Location_Invariants composes hostile location parts before deep validation.
func Node_Location_Invariants(value Node_Location, namespace aver.Namespace) {
	Node_Reference_Invariants(value.Reference, namespace)
	Node_Stored_Invariants(value.Node, namespace)
}

// Node_Location_Stored carries one location after program validation.
type Node_Location_Stored struct {
	// Reference retains the validated caller-owned arena slot.
	Reference Node_Reference_Proof_Stored
	// Node retains the caller-owned node without copying it.
	Node Node_Stored
}

// Node_Location_Stored_Invariants preserves location encodings without repeated checks.
func Node_Location_Stored_Invariants(
	value Node_Location_Stored, namespace aver.Namespace,
) {
	Node_Reference_Proof_Stored_Invariants(value.Reference, namespace)
	Node_Stored_Invariants(value.Node, namespace)
}

// NODE_LOCATION_VALIDATED_FIELD selects one deeply checked node location.
const NODE_LOCATION_VALIDATED_FIELD = bytes.SLICE_SIZE_MINIMUM

// NODE_LOCATION_VALIDATED_FIELD_COUNT fixes trusted location transport shape.
const NODE_LOCATION_VALIDATED_FIELD_COUNT = NODE_LOCATION_VALIDATED_FIELD +
	STORAGE_FIELD_COUNT_INCREMENT

// Node_Location_Validated carries one arena-bound node by value.
type Node_Location_Validated struct {
	// Value keeps arena proof attached to the node location.
	Value Node_Location_Stored
}

// Node_Location_Validated_Invariants checks transport ownership after program validation.
func Node_Location_Validated_Invariants(
	value Node_Location_Validated, namespace aver.Namespace,
) {
	Node_Location_Stored_Invariants(value.Value, namespace)
}

// NODE_LINK_VALIDATED_FIELD selects one optional link from a validated node.
const NODE_LINK_VALIDATED_FIELD = bytes.SLICE_SIZE_MINIMUM

// NODE_LINK_VALIDATED_FIELD_COUNT fixes trusted link transport shape.
const NODE_LINK_VALIDATED_FIELD_COUNT = NODE_LINK_VALIDATED_FIELD +
	STORAGE_FIELD_COUNT_INCREMENT

// Node_Link_Validated_Value retains program-link proof storage.
type Node_Link_Validated_Value uint16

// Node_Link_Validated_Value_Invariants preserves link encoding.
func Node_Link_Validated_Value_Invariants(
	value Node_Link_Validated_Value, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(NODE_REFERENCE_MINIMUM),
			uint16(NODE_REFERENCE_MAXIMUM),
		).
		Ensure()
}

// Node_Link_Validated carries one optional program link by value.
type Node_Link_Validated struct {
	// Value keeps link proof attached to the reference.
	Value Node_Link_Proof_Stored
}

// Node_Link_Validated_Invariants checks ownership after program validation.
func Node_Link_Validated_Invariants(value Node_Link_Validated, namespace aver.Namespace) {
	Node_Link_Proof_Stored_Invariants(value.Value, namespace)
}

// Output is caller destination.
type Output []byte

// Output_Invariants bounds destination before commit.
func Output_Invariants(value Output, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), bytes.SLICE_SIZE_MINIMUM, OUTPUT_SIZE_MAXIMUM,
		).
		Ensure()
}

// Output_Count is committed byte count.
type Output_Count uint16

// Output_Count_Invariants follows bounded staged output.
func Output_Count_Invariants(value Output_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(bytes.SLICE_SIZE_MINIMUM),
			uint16(OUTPUT_SIZE_MAXIMUM),
		).
		Ensure()
}

// OUTPUT_COUNT_STAGED_FIELD selects one count proven to fit staged output.
const OUTPUT_COUNT_STAGED_FIELD = bytes.SLICE_SIZE_MINIMUM

// OUTPUT_COUNT_STAGED_FIELD_COUNT fixes staged count transport shape.
const OUTPUT_COUNT_STAGED_FIELD_COUNT = OUTPUT_COUNT_STAGED_FIELD +
	STORAGE_FIELD_COUNT_INCREMENT

// Output_Count_Staged_Value retains staged-count proof storage.
type Output_Count_Staged_Value uint16

// Output_Count_Staged_Value_Invariants preserves the staged count.
func Output_Count_Staged_Value_Invariants(
	value Output_Count_Staged_Value, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(OUTPUT_SIZE_MINIMUM), uint16(OUTPUT_SIZE_MAXIMUM),
		).
		Ensure()
}

// Output_Count_Staged carries committed staged output progress by value.
type Output_Count_Staged struct {
	// Value keeps destination proof attached to output progress.
	Value Output_Count_Proof_Stored
}

// Output_Count_Staged_Invariants checks transport ownership after writer validation.
func Output_Count_Staged_Invariants(
	value Output_Count_Staged, namespace aver.Namespace,
) {
	Output_Count_Proof_Stored_Invariants(value.Value, namespace)
}

// Status classifies execution without allocated error text.
type Status uint8

// Status_Invariants covers every bounded execution outcome.
func Status_Invariants(value Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_LIMIT_EXCEEDED),
		).
		Ensure()
}

// Node_Execution_Status excludes destination refusal and pre-execution validation failures.
type Node_Execution_Status uint8

// Node_Execution_Status_Invariants admits only results produced after execution begins.
func Node_Execution_Status_Invariants(
	value Node_Execution_Status, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Holed_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_LIMIT_EXCEEDED),
			uint8(STATUS_OUTPUT_TOO_SMALL), uint8(STATUS_WORKSPACE_INVALID),
			uint8(STATUS_VALUE_INVALID),
		).
		Ensure()
}

// NODE_EXECUTION_STATUS_OK completes internal execution.
const NODE_EXECUTION_STATUS_OK = Node_Execution_Status(STATUS_OK)

// NODE_EXECUTION_STATUS_PROGRAM_INVALID refuses forged syntax during traversal.
const NODE_EXECUTION_STATUS_PROGRAM_INVALID = Node_Execution_Status(STATUS_PROGRAM_INVALID)

// NODE_EXECUTION_STATUS_LIMIT_EXCEEDED stops amplified traversal.
const NODE_EXECUTION_STATUS_LIMIT_EXCEEDED = Node_Execution_Status(STATUS_LIMIT_EXCEEDED)

// Value_Text_Status excludes execution failures that a scalar projection cannot produce.
type Value_Text_Status uint8

// Value_Text_Status_Invariants covers exact text and wrong value kind.
func Value_Text_Status_Invariants(
	value Value_Text_Status, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_VALUE_INVALID),
		).
		Ensure()
}

// VALUE_TEXT_STATUS_OK accepts one exact text value.
const VALUE_TEXT_STATUS_OK = Value_Text_Status(STATUS_OK)

// VALUE_TEXT_STATUS_INVALID rejects every non-text kind.
const VALUE_TEXT_STATUS_INVALID = Value_Text_Status(STATUS_VALUE_INVALID)

// Format_Status excludes failures unrelated to one bounded directive.
type Format_Status uint8

// Format_Status_Invariants covers accepted, malformed, and oversized directives.
func Format_Status_Invariants(value Format_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_EXECUTION_INVALID),
			uint8(STATUS_LIMIT_EXCEEDED),
		).
		Ensure()
}

// FORMAT_STATUS_OK accepts one complete directive.
const FORMAT_STATUS_OK = Format_Status(STATUS_OK)

// FORMAT_STATUS_INVALID rejects an incomplete directive.
const FORMAT_STATUS_INVALID = Format_Status(STATUS_EXECUTION_INVALID)

// FORMAT_STATUS_LIMIT_EXCEEDED rejects width beyond staged output.
const FORMAT_STATUS_LIMIT_EXCEEDED = Format_Status(STATUS_LIMIT_EXCEEDED)

// Formatting_Status excludes failures beyond directive compatibility and generated storage.
type Formatting_Status uint8

// Formatting_Status_Invariants covers accepted, incompatible, and oversized formatting.
func Formatting_Status_Invariants(
	value Formatting_Status, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_EXECUTION_INVALID),
			uint8(STATUS_LIMIT_EXCEEDED),
		).
		Ensure()
}

// FORMATTING_STATUS_OK produces one derived value.
const FORMATTING_STATUS_OK = Formatting_Status(STATUS_OK)

// FORMATTING_STATUS_INVALID rejects directive or value incompatibility.
const FORMATTING_STATUS_INVALID = Formatting_Status(STATUS_EXECUTION_INVALID)

// FORMATTING_STATUS_LIMIT_EXCEEDED rejects generated-storage exhaustion.
const FORMATTING_STATUS_LIMIT_EXCEEDED = Formatting_Status(STATUS_LIMIT_EXCEEDED)

// Function_Lookup_Status excludes call failures from registry lookup.
type Function_Lookup_Status uint8

// Function_Lookup_Status_Invariants covers found and absent names.
func Function_Lookup_Status_Invariants(
	value Function_Lookup_Status, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_FUNCTION_ABSENT),
		).
		Ensure()
}

// FUNCTION_LOOKUP_STATUS_OK accepts one registered function.
const FUNCTION_LOOKUP_STATUS_OK = Function_Lookup_Status(STATUS_OK)

// FUNCTION_LOOKUP_STATUS_ABSENT reports a missing function name.
const FUNCTION_LOOKUP_STATUS_ABSENT = Function_Lookup_Status(STATUS_FUNCTION_ABSENT)

// Function_Call_Status excludes lookup and execution-engine failures from an injected call.
type Function_Call_Status uint8

// Function_Call_Status_Invariants covers accepted and rejected callback results.
func Function_Call_Status_Invariants(
	value Function_Call_Status, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_FUNCTION_INVALID),
		).
		Ensure()
}

// FUNCTION_CALL_STATUS_OK accepts one validated callback result.
const FUNCTION_CALL_STATUS_OK = Function_Call_Status(STATUS_OK)

// FUNCTION_CALL_STATUS_INVALID rejects callback status or value shape.
const FUNCTION_CALL_STATUS_INVALID = Function_Call_Status(STATUS_FUNCTION_INVALID)

// Builtin_Argument_Status excludes failures beyond fixed builtin arity.
type Builtin_Argument_Status uint8

// Builtin_Argument_Status_Invariants covers accepted and malformed argument counts.
func Builtin_Argument_Status_Invariants(
	value Builtin_Argument_Status, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_FUNCTION_INVALID),
		).
		Ensure()
}

// BUILTIN_ARGUMENT_STATUS_OK accepts the builtin argument count.
const BUILTIN_ARGUMENT_STATUS_OK = Builtin_Argument_Status(STATUS_OK)

// BUILTIN_ARGUMENT_STATUS_INVALID rejects the builtin argument count.
const BUILTIN_ARGUMENT_STATUS_INVALID = Builtin_Argument_Status(STATUS_FUNCTION_INVALID)

// Builtin_Value_Status excludes failures beyond builtin arity and value kind.
type Builtin_Value_Status uint8

// Builtin_Value_Status_Invariants covers accepted, malformed, and unsupported arguments.
func Builtin_Value_Status_Invariants(
	value Builtin_Value_Status, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_FUNCTION_INVALID),
			uint8(STATUS_EXECUTION_INVALID),
		).
		Ensure()
}

// BUILTIN_VALUE_STATUS_OK accepts the builtin values.
const BUILTIN_VALUE_STATUS_OK = Builtin_Value_Status(STATUS_OK)

// BUILTIN_VALUE_STATUS_FUNCTION_INVALID rejects the builtin call contract.
const BUILTIN_VALUE_STATUS_FUNCTION_INVALID = Builtin_Value_Status(STATUS_FUNCTION_INVALID)

// BUILTIN_VALUE_STATUS_KIND_INVALID rejects unsupported builtin values.
const BUILTIN_VALUE_STATUS_KIND_INVALID = Builtin_Value_Status(STATUS_EXECUTION_INVALID)

// Index_Status excludes failures beyond one value traversal.
type Index_Status uint8

// Index_Status_Invariants covers accepted and invalid receiver-index pairs.
func Index_Status_Invariants(value Index_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_EXECUTION_INVALID),
		).
		Ensure()
}

// INDEX_STATUS_OK selects one value.
const INDEX_STATUS_OK = Index_Status(STATUS_OK)

// INDEX_STATUS_INVALID rejects receiver kind, index kind, or index bounds.
const INDEX_STATUS_INVALID = Index_Status(STATUS_EXECUTION_INVALID)

// Range_Status excludes failures beyond iterable kind and bounded count.
type Range_Status uint8

// Range_Status_Invariants covers accepted, unsupported, and oversized iterables.
func Range_Status_Invariants(value Range_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_EXECUTION_INVALID),
			uint8(STATUS_LIMIT_EXCEEDED),
		).
		Ensure()
}

// RANGE_STATUS_OK accepts one bounded iterable.
const RANGE_STATUS_OK = Range_Status(STATUS_OK)

// RANGE_STATUS_KIND_INVALID rejects a non-iterable value.
const RANGE_STATUS_KIND_INVALID = Range_Status(STATUS_EXECUTION_INVALID)

// RANGE_STATUS_LIMIT_EXCEEDED rejects a negative or oversized integer range.
const RANGE_STATUS_LIMIT_EXCEEDED = Range_Status(STATUS_LIMIT_EXCEEDED)

// Value_Graph_Status excludes failures beyond hostile value shape and work bounds.
type Value_Graph_Status uint8

// Value_Graph_Status_Invariants covers accepted, malformed, and oversized graphs.
func Value_Graph_Status_Invariants(
	value Value_Graph_Status, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_VALUE_INVALID),
			uint8(STATUS_LIMIT_EXCEEDED),
		).
		Ensure()
}

// VALUE_GRAPH_STATUS_OK accepts one bounded closed-union graph.
const VALUE_GRAPH_STATUS_OK = Value_Graph_Status(STATUS_OK)

// VALUE_GRAPH_STATUS_INVALID rejects malformed active values.
const VALUE_GRAPH_STATUS_INVALID = Value_Graph_Status(STATUS_VALUE_INVALID)

// VALUE_GRAPH_STATUS_LIMIT_EXCEEDED rejects traversal beyond caller storage or work.
const VALUE_GRAPH_STATUS_LIMIT_EXCEEDED = Value_Graph_Status(STATUS_LIMIT_EXCEEDED)

// Program_Status excludes failures beyond one compiled-program lookup or decode.
type Program_Status uint8

// Program_Status_Invariants covers accepted and stale compiled state.
func Program_Status_Invariants(value Program_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_PROGRAM_INVALID),
		).
		Ensure()
}

// PROGRAM_STATUS_OK accepts compiled state.
const PROGRAM_STATUS_OK = Program_Status(STATUS_OK)

// PROGRAM_STATUS_INVALID rejects stale or forged compiled state.
const PROGRAM_STATUS_INVALID = Program_Status(STATUS_PROGRAM_INVALID)

// Program_Execution_Status is one compiled-state or runtime-value decision.
type Program_Execution_Status uint8

// Program_Execution_Status_Invariants covers its exact outcome set.
func Program_Execution_Status_Invariants(
	value Program_Execution_Status, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_PROGRAM_INVALID),
			uint8(STATUS_EXECUTION_INVALID),
		).
		Ensure()
}

// PROGRAM_EXECUTION_STATUS_OK accepts compiled state and runtime value.
const PROGRAM_EXECUTION_STATUS_OK = Program_Execution_Status(STATUS_OK)

// PROGRAM_EXECUTION_STATUS_PROGRAM_INVALID rejects stale compiled state.
const PROGRAM_EXECUTION_STATUS_PROGRAM_INVALID = Program_Execution_Status(
	STATUS_PROGRAM_INVALID,
)

// PROGRAM_EXECUTION_STATUS_EXECUTION_INVALID rejects incompatible runtime state.
const PROGRAM_EXECUTION_STATUS_EXECUTION_INVALID = Program_Execution_Status(
	STATUS_EXECUTION_INVALID,
)

// Limit_Status is one bounded work-budget decision.
type Limit_Status uint8

// Limit_Status_Invariants covers accepted and exhausted work.
func Limit_Status_Invariants(value Limit_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_LIMIT_EXCEEDED),
		).
		Ensure()
}

// LIMIT_STATUS_OK accepts work inside caller storage.
const LIMIT_STATUS_OK = Limit_Status(STATUS_OK)

// LIMIT_STATUS_EXCEEDED rejects exhausted caller work storage.
const LIMIT_STATUS_EXCEEDED = Limit_Status(STATUS_LIMIT_EXCEEDED)

// Execution_Limit_Status is one runtime-value or work-budget decision.
type Execution_Limit_Status uint8

// Execution_Limit_Status_Invariants covers its exact outcome set.
func Execution_Limit_Status_Invariants(
	value Execution_Limit_Status, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_EXECUTION_INVALID),
			uint8(STATUS_LIMIT_EXCEEDED),
		).
		Ensure()
}

// EXECUTION_LIMIT_STATUS_OK accepts runtime state inside caller storage.
const EXECUTION_LIMIT_STATUS_OK = Execution_Limit_Status(STATUS_OK)

// EXECUTION_LIMIT_STATUS_EXECUTION_INVALID rejects incompatible runtime state.
const EXECUTION_LIMIT_STATUS_EXECUTION_INVALID = Execution_Limit_Status(
	STATUS_EXECUTION_INVALID,
)

// EXECUTION_LIMIT_STATUS_LIMIT_EXCEEDED rejects exhausted caller work storage.
const EXECUTION_LIMIT_STATUS_LIMIT_EXCEEDED = Execution_Limit_Status(
	STATUS_LIMIT_EXCEEDED,
)

// Program_Limit_Status is one compiled-state or work-budget decision.
type Program_Limit_Status uint8

// Program_Limit_Status_Invariants covers its exact outcome set.
func Program_Limit_Status_Invariants(
	value Program_Limit_Status, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_PROGRAM_INVALID),
			uint8(STATUS_LIMIT_EXCEEDED),
		).
		Ensure()
}

// PROGRAM_LIMIT_STATUS_OK accepts compiled state inside work budget.
const PROGRAM_LIMIT_STATUS_OK = Program_Limit_Status(STATUS_OK)

// PROGRAM_LIMIT_STATUS_PROGRAM_INVALID rejects stale compiled state.
const PROGRAM_LIMIT_STATUS_PROGRAM_INVALID = Program_Limit_Status(
	STATUS_PROGRAM_INVALID,
)

// PROGRAM_LIMIT_STATUS_LIMIT_EXCEEDED rejects exhausted caller work storage.
const PROGRAM_LIMIT_STATUS_LIMIT_EXCEEDED = Program_Limit_Status(
	STATUS_LIMIT_EXCEEDED,
)

// Program_Execution_Limit_Status combines compiled, runtime, and budget decisions.
type Program_Execution_Limit_Status uint8

// Program_Execution_Limit_Status_Invariants covers its exact outcome set.
func Program_Execution_Limit_Status_Invariants(
	value Program_Execution_Limit_Status, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_4_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_PROGRAM_INVALID),
			uint8(STATUS_EXECUTION_INVALID), uint8(STATUS_LIMIT_EXCEEDED),
		).
		Ensure()
}

// PROGRAM_EXECUTION_LIMIT_STATUS_OK accepts all three decision domains.
const PROGRAM_EXECUTION_LIMIT_STATUS_OK = Program_Execution_Limit_Status(STATUS_OK)

// PROGRAM_EXECUTION_LIMIT_STATUS_PROGRAM_INVALID rejects stale compiled state.
const PROGRAM_EXECUTION_LIMIT_STATUS_PROGRAM_INVALID = Program_Execution_Limit_Status(
	STATUS_PROGRAM_INVALID,
)

// PROGRAM_EXECUTION_LIMIT_STATUS_EXECUTION_INVALID rejects incompatible runtime state.
const PROGRAM_EXECUTION_LIMIT_STATUS_EXECUTION_INVALID = Program_Execution_Limit_Status(
	STATUS_EXECUTION_INVALID,
)

// PROGRAM_EXECUTION_LIMIT_STATUS_LIMIT_EXCEEDED rejects exhausted work storage.
const PROGRAM_EXECUTION_LIMIT_STATUS_LIMIT_EXCEEDED = Program_Execution_Limit_Status(
	STATUS_LIMIT_EXCEEDED,
)

// Call_Status is one callable or builtin completion decision.
type Call_Status uint8

// Call_Status_Invariants excludes failures outside callable execution.
func Call_Status_Invariants(value Call_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Holed_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_LIMIT_EXCEEDED),
			uint8(STATUS_OUTPUT_TOO_SMALL), uint8(STATUS_OUTPUT_TOO_LARGE),
			uint8(STATUS_PROGRAM_INVALID),
		).
		Ensure()
	aver.Always(
		value != Call_Status(STATUS_WORKSPACE_INVALID),
		"Callable execution starts after workspace validation.",
	)
	aver.Always(
		value != Call_Status(STATUS_VALUE_INVALID),
		"Callable execution starts after input-value validation.",
	)
}

// CALL_STATUS_OK accepts one callable result.
const CALL_STATUS_OK = Call_Status(STATUS_OK)

// CALL_STATUS_FUNCTION_INVALID rejects a callback contract failure.
const CALL_STATUS_FUNCTION_INVALID = Call_Status(STATUS_FUNCTION_INVALID)

// CALL_STATUS_FUNCTION_ABSENT rejects an unresolved callable name.
const CALL_STATUS_FUNCTION_ABSENT = Call_Status(STATUS_FUNCTION_ABSENT)

// CALL_STATUS_EXECUTION_INVALID rejects incompatible arguments or values.
const CALL_STATUS_EXECUTION_INVALID = Call_Status(STATUS_EXECUTION_INVALID)

// CALL_STATUS_LIMIT_EXCEEDED rejects generated output or argument exhaustion.
const CALL_STATUS_LIMIT_EXCEEDED = Call_Status(STATUS_LIMIT_EXCEEDED)

// Term_Scalar_Status is one scalar syntax-term evaluation decision.
type Term_Scalar_Status uint8

// Term_Scalar_Status_Invariants covers its exact outcome set.
func Term_Scalar_Status_Invariants(
	value Term_Scalar_Status, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_4_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_PROGRAM_INVALID),
			uint8(STATUS_FUNCTION_ABSENT), uint8(STATUS_EXECUTION_INVALID),
		).
		Ensure()
}

// TERM_SCALAR_STATUS_OK accepts one scalar term.
const TERM_SCALAR_STATUS_OK = Term_Scalar_Status(STATUS_OK)

// TERM_SCALAR_STATUS_PROGRAM_INVALID rejects stale scalar syntax.
const TERM_SCALAR_STATUS_PROGRAM_INVALID = Term_Scalar_Status(STATUS_PROGRAM_INVALID)

// TERM_SCALAR_STATUS_FUNCTION_ABSENT rejects an unresolved function value.
const TERM_SCALAR_STATUS_FUNCTION_ABSENT = Term_Scalar_Status(STATUS_FUNCTION_ABSENT)

// TERM_SCALAR_STATUS_EXECUTION_INVALID rejects incompatible runtime data.
const TERM_SCALAR_STATUS_EXECUTION_INVALID = Term_Scalar_Status(STATUS_EXECUTION_INVALID)

// Term_Status is one syntax-term traversal decision.
type Term_Status uint8

// Term_Status_Invariants excludes failures outside term traversal.
func Term_Status_Invariants(value Term_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Holed_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_LIMIT_EXCEEDED),
			uint8(STATUS_OUTPUT_TOO_SMALL), uint8(STATUS_OUTPUT_TOO_LARGE),
			uint8(STATUS_WORKSPACE_INVALID),
		).
		Ensure()
	aver.Always(
		value != Term_Status(STATUS_VALUE_INVALID),
		"Term traversal starts after input-value validation.",
	)
	aver.Always(
		value != Term_Status(STATUS_FUNCTION_INVALID),
		"Term traversal resolves function values without calling them.",
	)
}

// TERM_STATUS_OK accepts one traversed term.
const TERM_STATUS_OK = Term_Status(STATUS_OK)

// TERM_STATUS_PROGRAM_INVALID rejects stale term syntax.
const TERM_STATUS_PROGRAM_INVALID = Term_Status(STATUS_PROGRAM_INVALID)

// TERM_STATUS_FUNCTION_ABSENT rejects an unresolved function value.
const TERM_STATUS_FUNCTION_ABSENT = Term_Status(STATUS_FUNCTION_ABSENT)

// TERM_STATUS_EXECUTION_INVALID rejects incompatible runtime data.
const TERM_STATUS_EXECUTION_INVALID = Term_Status(STATUS_EXECUTION_INVALID)

// TERM_STATUS_LIMIT_EXCEEDED rejects traversal storage exhaustion.
const TERM_STATUS_LIMIT_EXCEEDED = Term_Status(STATUS_LIMIT_EXCEEDED)

// Evaluation_Status is one complete pipeline evaluation decision.
type Evaluation_Status uint8

// Evaluation_Status_Invariants excludes failures outside pipeline evaluation.
func Evaluation_Status_Invariants(
	value Evaluation_Status, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Holed_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_LIMIT_EXCEEDED),
			uint8(STATUS_OUTPUT_TOO_SMALL), uint8(STATUS_OUTPUT_TOO_LARGE),
			uint8(STATUS_WORKSPACE_INVALID),
		).
		Ensure()
	aver.Always(
		value != Evaluation_Status(STATUS_VALUE_INVALID),
		"Pipeline evaluation cannot fail output, workspace, or input validation.",
	)
}

// EVALUATION_STATUS_OK accepts one complete pipeline.
const EVALUATION_STATUS_OK = Evaluation_Status(STATUS_OK)

// EVALUATION_STATUS_PROGRAM_INVALID rejects stale pipeline syntax.
const EVALUATION_STATUS_PROGRAM_INVALID = Evaluation_Status(STATUS_PROGRAM_INVALID)

// EVALUATION_STATUS_FUNCTION_INVALID rejects a callback contract failure.
const EVALUATION_STATUS_FUNCTION_INVALID = Evaluation_Status(STATUS_FUNCTION_INVALID)

// EVALUATION_STATUS_FUNCTION_ABSENT rejects an unresolved callable name.
const EVALUATION_STATUS_FUNCTION_ABSENT = Evaluation_Status(STATUS_FUNCTION_ABSENT)

// EVALUATION_STATUS_EXECUTION_INVALID rejects incompatible runtime data.
const EVALUATION_STATUS_EXECUTION_INVALID = Evaluation_Status(STATUS_EXECUTION_INVALID)

// EVALUATION_STATUS_LIMIT_EXCEEDED rejects evaluation storage exhaustion.
const EVALUATION_STATUS_LIMIT_EXCEEDED = Evaluation_Status(STATUS_LIMIT_EXCEEDED)

// Number_Status excludes failures beyond one compiled numeric literal.
type Number_Status uint8

// Number_Status_Invariants covers accepted, stale, and invalid numeric text.
func Number_Status_Invariants(value Number_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_PROGRAM_INVALID),
			uint8(STATUS_EXECUTION_INVALID),
		).
		Ensure()
}

// NUMBER_STATUS_OK accepts one scalar literal.
const NUMBER_STATUS_OK = Number_Status(STATUS_OK)

// NUMBER_STATUS_PROGRAM_INVALID rejects stale compiled literal structure.
const NUMBER_STATUS_PROGRAM_INVALID = Number_Status(STATUS_PROGRAM_INVALID)

// NUMBER_STATUS_VALUE_INVALID rejects malformed numeric payload.
const NUMBER_STATUS_VALUE_INVALID = Number_Status(STATUS_EXECUTION_INVALID)

// Definition_Status excludes failures beyond one compiled template-name lookup.
type Definition_Status uint8

// Definition_Status_Invariants covers found, absent, and stale definition names.
func Definition_Status_Invariants(
	value Definition_Status, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_PROGRAM_INVALID),
			uint8(STATUS_EXECUTION_INVALID),
		).
		Ensure()
}

// DEFINITION_STATUS_OK finds one definition.
const DEFINITION_STATUS_OK = Definition_Status(STATUS_OK)

// DEFINITION_STATUS_PROGRAM_INVALID rejects stale compiled names.
const DEFINITION_STATUS_PROGRAM_INVALID = Definition_Status(STATUS_PROGRAM_INVALID)

// DEFINITION_STATUS_ABSENT reports an undefined template name.
const DEFINITION_STATUS_ABSENT = Definition_Status(STATUS_EXECUTION_INVALID)

// Value_Access_Status excludes failures beyond one compiled field path and receiver.
type Value_Access_Status uint8

// Value_Access_Status_Invariants covers found, stale, and incompatible traversal.
func Value_Access_Status_Invariants(
	value Value_Access_Status, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_PROGRAM_INVALID),
			uint8(STATUS_EXECUTION_INVALID),
		).
		Ensure()
}

// VALUE_ACCESS_STATUS_OK resolves a field or the standard missing nil value.
const VALUE_ACCESS_STATUS_OK = Value_Access_Status(STATUS_OK)

// VALUE_ACCESS_STATUS_PROGRAM_INVALID rejects stale path syntax.
const VALUE_ACCESS_STATUS_PROGRAM_INVALID = Value_Access_Status(STATUS_PROGRAM_INVALID)

// VALUE_ACCESS_STATUS_RECEIVER_INVALID rejects a non-object receiver.
const VALUE_ACCESS_STATUS_RECEIVER_INVALID = Value_Access_Status(STATUS_EXECUTION_INVALID)

// Variable_Status excludes failures beyond one compiled variable path and binding.
type Variable_Status uint8

// Variable_Status_Invariants covers found, stale, and absent variable access.
func Variable_Status_Invariants(value Variable_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_PROGRAM_INVALID),
			uint8(STATUS_EXECUTION_INVALID),
		).
		Ensure()
}

// VARIABLE_STATUS_OK resolves one binding.
const VARIABLE_STATUS_OK = Variable_Status(STATUS_OK)

// VARIABLE_STATUS_PROGRAM_INVALID rejects a stale field suffix.
const VARIABLE_STATUS_PROGRAM_INVALID = Variable_Status(STATUS_PROGRAM_INVALID)

// VARIABLE_STATUS_ABSENT reports an unavailable binding.
const VARIABLE_STATUS_ABSENT = Variable_Status(STATUS_EXECUTION_INVALID)

// Output_Status is the exact bounded staging-write outcome set.
type Output_Status uint8

// Output_Status_Invariants covers success and exhausted staging.
func Output_Status_Invariants(
	value Output_Status, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_OUTPUT_TOO_LARGE),
		).
		Ensure()
}

// OUTPUT_STATUS_OK commits one complete staging write.
const OUTPUT_STATUS_OK = Output_Status(STATUS_OK)

// OUTPUT_STATUS_TOO_LARGE rejects staging overflow.
const OUTPUT_STATUS_TOO_LARGE = Output_Status(STATUS_OUTPUT_TOO_LARGE)

// Generated_Status is the exact derived-text storage outcome set.
type Generated_Status uint8

// Generated_Status_Invariants covers success and exhausted derived storage.
func Generated_Status_Invariants(
	value Generated_Status, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_LIMIT_EXCEEDED),
		).
		Ensure()
}

// GENERATED_STATUS_OK commits one complete derived-text write.
const GENERATED_STATUS_OK = Generated_Status(STATUS_OK)

// GENERATED_STATUS_LIMIT_EXCEEDED rejects derived-text exhaustion.
const GENERATED_STATUS_LIMIT_EXCEEDED = Generated_Status(STATUS_LIMIT_EXCEEDED)

// Diagnostic_Position is source boundary nearest refusal.
type Diagnostic_Position uint16

// Diagnostic_Position_Invariants stays inside source maximum.
func Diagnostic_Position_Invariants(
	value Diagnostic_Position, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(bytes.SLICE_SIZE_MINIMUM),
			uint16(SOURCE_SIZE_MAXIMUM),
		).
		Ensure()
}

// Diagnostic carries scalar execution refusal.
type Diagnostic struct {
	// Code identifies refusal class.
	Code Status
	// Position identifies responsible syntax boundary.
	Position Diagnostic_Position
}

// Diagnostic_Invariants composes execution diagnostic.
func Diagnostic_Invariants(value Diagnostic, namespace aver.Namespace) {
	Status_Invariants(value.Code, namespace)
	Diagnostic_Position_Invariants(value.Position, namespace)
}

// Value_Nil constructs absent data.
func Value_Nil() (result Value) {
	defer func() { Value_Invariants(result, "Value_Nil.result") }()
	return Value{Kind: Value_Kind_Storage{Value: Value_Kind_Storage_Value(VALUE_NIL)}}
}

// Value_Of_Boolean constructs truth data.
func Value_Of_Boolean(value Boolean) (result Value) {
	defer func() { Value_Invariants(result, "Value_Of_Boolean.result") }()
	Boolean_Invariants(value, "Value_Of_Boolean.value")
	scalar := uint64(bits.WORD_64_MINIMUM)
	if bool(value) {
		scalar = bits.CARRY_MAXIMUM
	}
	return Value{
		Kind:   Value_Kind_Storage{Value: Value_Kind_Storage_Value(VALUE_BOOLEAN)},
		Scalar: Value_Scalar{Real: Value_Scalar_Real(scalar)},
	}
}

// Value_Of_Integer constructs signed data.
func Value_Of_Integer(value Integer) (result Value) {
	defer func() { Value_Invariants(result, "Value_Of_Integer.result") }()
	Integer_Invariants(value, "Value_Of_Integer.value")
	return Value{
		Kind:   Value_Kind_Storage{Value: Value_Kind_Storage_Value(VALUE_INTEGER)},
		Scalar: Value_Scalar{Real: Value_Scalar_Real(value)},
	}
}

// Value_Of_Unsigned constructs unsigned data.
func Value_Of_Unsigned(value Unsigned) (result Value) {
	defer func() { Value_Invariants(result, "Value_Of_Unsigned.result") }()
	Unsigned_Invariants(value, "Value_Of_Unsigned.value")
	return Value{
		Kind:   Value_Kind_Storage{Value: Value_Kind_Storage_Value(VALUE_UNSIGNED)},
		Scalar: Value_Scalar{Real: Value_Scalar_Real(value)},
	}
}

// Value_Of_Float constructs binary64 data from exact bits.
func Value_Of_Float(value Float) (result Value) {
	defer func() { Value_Invariants(result, "Value_Of_Float.result") }()
	Float_Invariants(value, "Value_Of_Float.value")
	return Value{
		Kind:   Value_Kind_Storage{Value: Value_Kind_Storage_Value(VALUE_FLOAT)},
		Scalar: Value_Scalar{Real: Value_Scalar_Real(value)},
	}
}

// Value_Of_Complex constructs complex data from exact bits.
func Value_Of_Complex(value Complex) (result Value) {
	defer func() { Value_Invariants(result, "Value_Of_Complex.result") }()
	Complex_Invariants(value, "Value_Of_Complex.value")
	return Value{
		Kind: Value_Kind_Storage{Value: Value_Kind_Storage_Value(VALUE_COMPLEX)},
		Scalar: Value_Scalar{
			Real:      Value_Scalar_Real(value.Real),
			Imaginary: Value_Scalar_Imaginary(value.Imaginary),
		},
	}
}

// Value_Of_Text borrows bounded text.
func Value_Of_Text(value Text) (result Value) {
	defer func() { Value_Invariants(result, "Value_Of_Text.result") }()
	Text_Invariants(value, "Value_Of_Text.value")
	return Value{
		Kind: Value_Kind_Storage{Value: Value_Kind_Storage_Value(VALUE_TEXT)},
		Text: Text_Storage{Value: Text_Storage_Value(value)},
	}
}

// Value_Of_Sequence borrows ordered values.
func Value_Of_Sequence(value Values) (result Value) {
	defer func() { Value_Invariants(result, "Value_Of_Sequence.result") }()
	Values_Invariants(value, "Value_Of_Sequence.value")
	return Value{
		Kind:   Value_Kind_Storage{Value: Value_Kind_Storage_Value(VALUE_SEQUENCE)},
		Values: Values_Storage{Value: Values_Storage_Value(value)},
	}
}

// Value_Of_Object borrows named fields.
func Value_Of_Object(value Fields) (result Value) {
	defer func() { Value_Invariants(result, "Value_Of_Object.result") }()
	Fields_Invariants(value, "Value_Of_Object.value")
	return Value{
		Kind:   Value_Kind_Storage{Value: Value_Kind_Storage_Value(VALUE_OBJECT)},
		Fields: Fields_Storage{Value: Fields_Storage_Value(value)},
	}
}

// Value_Of_Map borrows key-value entries.
func Value_Of_Map(value Fields) (result Value) {
	defer func() { Value_Invariants(result, "Value_Of_Map.result") }()
	Fields_Invariants(value, "Value_Of_Map.value")
	return Value{
		Kind:   Value_Kind_Storage{Value: Value_Kind_Storage_Value(VALUE_MAP)},
		Fields: Fields_Storage{Value: Fields_Storage_Value(value)},
	}
}

// Value_Of_Function stores injected callable data.
func Value_Of_Function(value Function_Call) (result Value) {
	defer func() { Value_Invariants(result, "Value_Of_Function.result") }()
	Function_Call_Invariants(value, "Value_Of_Function.value")
	return Value{
		Kind:     Value_Kind_Storage{Value: Value_Kind_Storage_Value(VALUE_FUNCTION)},
		Function: Function_Storage{Value: Function_Storage_Value(value)},
	}
}

// Value_As_Text returns text only for exact text kind.
func Value_As_Text(value Value) (_ Text, status Value_Text_Status) {
	defer func() { Value_Text_Status_Invariants(status, "Value_As_Text.status") }()
	Value_Invariants(value, "Value_As_Text.value")
	var result Text
	defer func() { Text_Invariants(result, "Value_As_Text.result") }()
	if Value_Kind(value.Kind.Value) != VALUE_TEXT {
		return result, VALUE_TEXT_STATUS_INVALID
	}
	result = Text(value.Text.Value)
	return result, VALUE_TEXT_STATUS_OK
}

// Compile parses one bounded program into caller syntax storage.
func Compile(
	input Compile_Input,
) (_ Program, _ Parse_Diagnostic, status Parse_Status) {
	defer func() { Parse_Status_Invariants(status, "Compile.status") }()
	Compile_Input_Invariants(input, "Compile.input")
	var program Program
	defer func() { Program_Invariants(program, "Compile.program") }()
	var diagnostic Parse_Diagnostic
	defer func() { Parse_Diagnostic_Invariants(diagnostic, "Compile.diagnostic") }()
	source, source_status := Source_Validate(input.Source)
	if source_status != PARSE_STATUS_OK {
		diagnostic.Code = DIAGNOSTIC_INPUT_INVALID
		return program, diagnostic, PARSE_STATUS_INPUT_INVALID
	}
	configuration, configuration_status := New_Configuration(
		Configuration_Input{
			Left: input.Left, Right: input.Right,
			Mode: input.Mode | SKIP_FUNCTION_CHECK,
		},
	)
	if configuration_status != PARSE_STATUS_OK {
		diagnostic.Code = DIAGNOSTIC_CONFIGURATION_INVALID
		return program, diagnostic, PARSE_STATUS_CONFIGURATION_INVALID
	}
	syntax := input.Workspace.State.Value
	var workspace_input Syntax_Workspace_Input
	workspace_input.State.Value = Syntax_Workspace_Unvalidated_Pointer{
		Value: (*Syntax_Workspace)(syntax),
	}
	document, diagnostic, status := Parse_Into(
		source, configuration, workspace_input,
	)
	if status != PARSE_STATUS_OK {
		return program, diagnostic, status
	}
	program = Program{
		Source: source, Document: document,
		Syntax: Syntax_Storage{Value: Syntax_Workspace_Pointer(syntax)},
	}
	return program, diagnostic, status
}

// Output_Storage is caller-owned atomic staging memory.
type Output_Storage []byte

// Output_Storage_Invariants bounds hostile staging memory.
func Output_Storage_Invariants(value Output_Storage, namespace aver.Namespace) {
	aver.Always(
		len(value) == OUTPUT_SIZE_MAXIMUM,
		"Output staging has complete bounded capacity.",
	)
}

// Literal_Storage is caller-owned decoded-literal memory.
type Literal_Storage []byte

// Literal_Storage_Invariants bounds hostile literal staging.
func Literal_Storage_Invariants(value Literal_Storage, namespace aver.Namespace) {
	aver.Always(
		len(value) == QUOTED_DECODED_SIZE_MAXIMUM,
		"Literal staging has complete decoded-source capacity.",
	)
}

// Literal_Count is populated decoded literal storage.
type Literal_Count uint16

// Literal_Count_Invariants follows exact source-derived decode maximum.
func Literal_Count_Invariants(value Literal_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(bytes.SLICE_SIZE_MINIMUM),
			uint16(QUOTED_DECODED_SIZE_MAXIMUM),
		).
		Ensure()
}

// Literal_Counts stores decoded bytes per syntax node.
type Literal_Counts []Literal_Count

// Literal_Counts_Invariants bounds hostile cache storage.
func Literal_Counts_Invariants(value Literal_Counts, namespace aver.Namespace) {
	aver.Always(
		len(value) == NODE_COUNT_MAXIMUM,
		"Literal count cache covers complete syntax arena.",
	)
}

// Literal_Offsets stores decoded opening per syntax node.
type Literal_Offsets []Literal_Count

// Literal_Offsets_Invariants bounds hostile cache storage.
func Literal_Offsets_Invariants(value Literal_Offsets, namespace aver.Namespace) {
	aver.Always(
		len(value) == NODE_COUNT_MAXIMUM,
		"Literal offset cache covers complete syntax arena.",
	)
}

// Literal_Decoded stores cache presence per syntax node.
type Literal_Decoded []Boolean

// Literal_Decoded_Invariants bounds hostile cache storage.
func Literal_Decoded_Invariants(value Literal_Decoded, namespace aver.Namespace) {
	aver.Always(
		len(value) == NODE_COUNT_MAXIMUM,
		"Literal presence cache covers complete syntax arena.",
	)
}

// Name_Left_Storage decodes one invocation name.
type Name_Left_Storage []byte

// Name_Left_Storage_Invariants bounds hostile invocation-name memory.
func Name_Left_Storage_Invariants(
	value Name_Left_Storage, namespace aver.Namespace,
) {
	aver.Always(
		len(value) == QUOTED_DECODED_SIZE_MAXIMUM,
		"Invocation-name staging has complete decoded-source capacity.",
	)
}

// Name_Right_Storage decodes one definition name.
type Name_Right_Storage []byte

// Name_Right_Storage_Invariants bounds hostile definition-name memory.
func Name_Right_Storage_Invariants(
	value Name_Right_Storage, namespace aver.Namespace,
) {
	aver.Always(
		len(value) == QUOTED_DECODED_SIZE_MAXIMUM,
		"Definition-name staging has complete decoded-source capacity.",
	)
}

// Number_Storage formats widest scalar.
type Number_Storage []byte

// Number_Storage_Invariants bounds hostile number staging.
func Number_Storage_Invariants(value Number_Storage, namespace aver.Namespace) {
	aver.Always(
		len(value) == NUMBER_SIZE_MAXIMUM,
		"Number staging has widest scalar text capacity.",
	)
}

// Generated_Storage is caller-owned derived builtin text.
type Generated_Storage []byte

// Generated_Storage_Invariants fixes source-plus-result scratch capacity.
func Generated_Storage_Invariants(
	value Generated_Storage, _ aver.Namespace,
) {
	aver.Always(
		len(value) == GENERATED_SIZE_MAXIMUM,
		"Generated text retains one source and one bounded derived result.",
	)
}

// Generated_Count is populated derived builtin text.
type Generated_Count uint16

// Generated_Count_Invariants follows complete generated scratch capacity.
func Generated_Count_Invariants(
	value Generated_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(bytes.SLICE_SIZE_MINIMUM),
			uint16(GENERATED_SIZE_MAXIMUM),
		).
		Ensure()
}

// Workspace_Generated_Count_Storage_Value retains generated-cursor storage.
type Workspace_Generated_Count_Storage_Value uint16

// Workspace_Generated_Count_Storage_Value_Invariants preserves cursor encoding.
func Workspace_Generated_Count_Storage_Value_Invariants(
	value Workspace_Generated_Count_Storage_Value, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(bytes.SLICE_SIZE_MINIMUM),
			uint16(GENERATED_SIZE_MAXIMUM),
		).
		Ensure()
}

// Workspace_Generated_Count_Stored keeps stale caller state opaque until reset.
type Workspace_Generated_Count_Stored interface{}

// Workspace_Generated_Count_Stored_Invariants accepts absent or typed generated state.
func Workspace_Generated_Count_Stored_Invariants(
	value Workspace_Generated_Count_Stored, _ aver.Namespace,
) {
	_, valid := value.(Workspace_Generated_Count_Storage_Value)
	aver.Always(valid == (value != nil), "Stored generated cursor has expected type.")
}

// Workspace_Generated_Count_Storage keeps generated cursor caller-owned.
type Workspace_Generated_Count_Storage struct {
	// Value keeps mutation inside execution workspace.
	Value Workspace_Generated_Count_Stored
}

// Workspace_Generated_Count_Storage_Invariants fixes scalar shape.
func Workspace_Generated_Count_Storage_Invariants(
	value Workspace_Generated_Count_Storage, namespace aver.Namespace,
) {
	Workspace_Generated_Count_Stored_Invariants(value.Value, namespace)
}

// Frame_Kind distinguishes list and range continuation.
type Frame_Kind uint8

// Frame_Kind_Invariants covers both execution frames.
func Frame_Kind_Invariants(value Frame_Kind, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value), uint8(FRAME_LIST), uint8(FRAME_RANGE)).
		Ensure()
}

// FRAME_LIST walks one ordered syntax list.
const FRAME_LIST Frame_Kind = Frame_Kind(bytes.SLICE_SIZE_MINIMUM)

// FRAME_KIND_INCREMENT adds one traversal-frame kind.
const FRAME_KIND_INCREMENT Frame_Kind = 1

// FRAME_RANGE schedules next collection item.
const FRAME_RANGE Frame_Kind = FRAME_LIST + FRAME_KIND_INCREMENT

// Composite_Opened reports emitted opening delimiter.
type Composite_Opened bool

// Composite_Opened_Invariants covers fresh and active format frames.
func Composite_Opened_Invariants(
	value Composite_Opened, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Composite opening delimiter was emitted.").
		Ensure()
}

// Frame_Index is active caller-owned frame position.
type Frame_Index uint16

// Frame_Index_Invariants bounds frame stack cursor.
func Frame_Index_Invariants(value Frame_Index, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(bytes.SLICE_SIZE_MINIMUM),
			uint16(FRAME_COUNT_MAXIMUM),
		).
		Ensure()
}

// ACTIVE_FRAME_COUNT_MINIMUM is one active evaluation frame.
const ACTIVE_FRAME_COUNT_MINIMUM Active_Frame_Count = FRAME_COUNT_INCREMENT

// Active_Frame_Count is a nonempty evaluation stack depth.
type Active_Frame_Count uint16

// Active_Frame_Count_Invariants excludes the empty state handled by the caller.
func Active_Frame_Count_Invariants(
	value Active_Frame_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(ACTIVE_FRAME_COUNT_MINIMUM),
			uint16(FRAME_COUNT_MAXIMUM),
		).
		Ensure()
}

// Collection_Index is next range item.
type Collection_Index uint16

// Collection_Index_Invariants includes complete collection boundary.
func Collection_Index_Invariants(value Collection_Index, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(bytes.SLICE_SIZE_MINIMUM),
			uint16(VALUE_COUNT_MAXIMUM),
		).
		Ensure()
}

// Collection_Element_Index selects one populated bounded collection element.
type Collection_Element_Index uint16

// Collection_Element_Index_Invariants excludes the complete collection boundary.
func Collection_Element_Index_Invariants(
	value Collection_Element_Index, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(bytes.SLICE_SIZE_MINIMUM),
			uint16(MAP_ENTRY_INDEX_MAXIMUM),
		).
		Ensure()
}

// Range_Map_Entry_Index is one map entry reachable after complete graph validation.
type Range_Map_Entry_Index uint16

// Range_Map_Entry_Index_Invariants follows the largest map fitting validation storage.
func Range_Map_Entry_Index_Invariants(
	value Range_Map_Entry_Index, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(bytes.SLICE_SIZE_MINIMUM),
			uint16(RANGE_MAP_ENTRY_INDEX_MAXIMUM),
		).
		Ensure()
}

// MAP_ENTRY_GRAPH_VALUE_COUNT charges one key and one value.
const MAP_ENTRY_GRAPH_VALUE_COUNT = VALUE_PAYLOAD_FIELD_COUNT + VALUE_PAYLOAD_FIELD_COUNT

// RANGE_MAP_ENTRY_COUNT_MAXIMUM fits complete map entries in validation storage.
const RANGE_MAP_ENTRY_COUNT_MAXIMUM = FRAME_COUNT_MAXIMUM / MAP_ENTRY_GRAPH_VALUE_COUNT

// RANGE_MAP_ENTRY_INDEX_MAXIMUM is the final rangeable caller map entry.
const RANGE_MAP_ENTRY_INDEX_MAXIMUM = RANGE_MAP_ENTRY_COUNT_MAXIMUM -
	MAP_ENTRY_COUNT_INCREMENT

// PRIOR_MAP_ENTRY_INDEX_MAXIMUM permits any physical slot selected by key order.
const PRIOR_MAP_ENTRY_INDEX_MAXIMUM = RANGE_MAP_ENTRY_INDEX_MAXIMUM

// Prior_Map_Entry_Index is the physical slot selected on the prior iteration.
type Prior_Map_Entry_Index uint16

// Prior_Map_Entry_Index_Invariants follows every rangeable physical map slot.
func Prior_Map_Entry_Index_Invariants(
	value Prior_Map_Entry_Index, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(bytes.SLICE_SIZE_MINIMUM),
			uint16(PRIOR_MAP_ENTRY_INDEX_MAXIMUM),
		).
		Ensure()
}

// Map_Selection reports whether a previous sorted key exists.
type Map_Selection bool

// Map_Selection_Invariants covers first and later range iterations.
func Map_Selection_Invariants(value Map_Selection, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Map range has selected its first key.").
		Ensure()
}

// Collection_Count is bounded range item count.
type Collection_Count uint16

// Collection_Count_Invariants follows caller value bound.
func Collection_Count_Invariants(
	value Collection_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(bytes.SLICE_SIZE_MINIMUM),
			uint16(VALUE_COUNT_MAXIMUM),
		).
		Ensure()
}

// Variable_Count is populated variable storage.
type Variable_Count uint16

// Variable_Count_Invariants follows parser lexical bound.
func Variable_Count_Invariants(value Variable_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(bytes.SLICE_SIZE_MINIMUM),
			uint16(VARIABLE_COUNT_MAXIMUM),
		).
		Ensure()
}

// VARIABLE_INDEX_ABSENT means lookup found no binding.
const VARIABLE_INDEX_ABSENT Variable_Index = Variable_Index(bytes.SLICE_SIZE_MINIMUM)

// Variable_Index is one-based lexical storage slot or absent.
type Variable_Index uint16

// Variable_Index_Invariants follows lexical storage plus absent sentinel.
func Variable_Index_Invariants(value Variable_Index, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(VARIABLE_INDEX_ABSENT),
			uint16(VARIABLE_COUNT_MAXIMUM),
		).
		Ensure()
}

// Scope_Base is first variable visible inside invoked template.
type Scope_Base uint16

// Scope_Base_Invariants follows lexical variable storage.
func Scope_Base_Invariants(value Scope_Base, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(bytes.SLICE_SIZE_MINIMUM),
			uint16(VARIABLE_COUNT_MAXIMUM),
		).
		Ensure()
}

// Scope_Restore retains one lexical state captured before nested execution.
type Scope_Restore struct {
	// Mark restores populated variable storage.
	Mark Variable_Count
	// Previous restores the enclosing visibility boundary.
	Previous Scope_Base
}

// Scope_Restore_Invariants composes hostile lexical restoration parts.
func Scope_Restore_Invariants(value Scope_Restore, namespace aver.Namespace) {
	Variable_Count_Invariants(value.Mark, namespace)
	Scope_Base_Invariants(value.Previous, namespace)
}

// Scope_Restore_Stored carries lexical cursors after workspace validation.
type Scope_Restore_Stored Scope_Restore

// Scope_Restore_Stored_Invariants preserves validated lexical cursors.
func Scope_Restore_Stored_Invariants(
	value Scope_Restore_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value.Mark), uint16(VARIABLE_COUNT_MINIMUM),
			uint16(VARIABLE_COUNT_MAXIMUM),
		).
		Range_Uint16(
			uint16(value.Previous), uint16(bytes.SLICE_SIZE_MINIMUM),
			uint16(VARIABLE_COUNT_MAXIMUM),
		).
		Ensure()
}

// SCOPE_RESTORE_VALIDATED_FIELD selects one state captured from workspace.
const SCOPE_RESTORE_VALIDATED_FIELD = bytes.SLICE_SIZE_MINIMUM

// SCOPE_RESTORE_VALIDATED_FIELD_COUNT fixes trusted restoration transport shape.
const SCOPE_RESTORE_VALIDATED_FIELD_COUNT = SCOPE_RESTORE_VALIDATED_FIELD +
	STORAGE_FIELD_COUNT_INCREMENT

// Scope_Restore_Validated carries one captured lexical state by value.
type Scope_Restore_Validated Scope_Restore_Validated_Fields

// Scope_Restore_Validated_Invariants checks ownership after workspace validation.
func Scope_Restore_Validated_Invariants(
	value Scope_Restore_Validated, namespace aver.Namespace,
) {
	Scope_Restore_Proof_Stored_Invariants(
		Scope_Restore_Proof_Stored(value.Value), namespace,
	)
}

// Frame_Variable_Mark restores enclosing variable count.
type Frame_Variable_Mark uint16

// Frame_Variable_Mark_Invariants follows lexical variable storage.
func Frame_Variable_Mark_Invariants(
	value Frame_Variable_Mark, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(bytes.SLICE_SIZE_MINIMUM),
			uint16(VARIABLE_COUNT_MAXIMUM),
		).
		Ensure()
}

// Frame_Iteration_Variable_Count restores range declarations.
type Frame_Iteration_Variable_Count uint16

// Frame_Iteration_Variable_Count_Invariants follows lexical variable storage.
func Frame_Iteration_Variable_Count_Invariants(
	value Frame_Iteration_Variable_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(bytes.SLICE_SIZE_MINIMUM),
			uint16(VARIABLE_COUNT_MAXIMUM),
		).
		Ensure()
}

// Frame_Scope_Base retains range body scope.
type Frame_Scope_Base uint16

// Frame_Scope_Base_Invariants follows lexical variable storage.
func Frame_Scope_Base_Invariants(
	value Frame_Scope_Base, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(bytes.SLICE_SIZE_MINIMUM),
			uint16(VARIABLE_COUNT_MAXIMUM),
		).
		Ensure()
}

// Frame_Previous_Scope_Base restores caller template scope.
type Frame_Previous_Scope_Base uint16

// Frame_Previous_Scope_Base_Invariants follows lexical variable storage.
func Frame_Previous_Scope_Base_Invariants(
	value Frame_Previous_Scope_Base, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(bytes.SLICE_SIZE_MINIMUM),
			uint16(VARIABLE_COUNT_MAXIMUM),
		).
		Ensure()
}

// Declaration_Count is zero, one, or two pipeline names.
type Declaration_Count uint8

// Declaration_Count_Invariants covers standard declaration arity.
func Declaration_Count_Invariants(value Declaration_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(DECLARATION_COUNT_NONE),
			uint8(DECLARATION_COUNT_ONE), uint8(DECLARATION_COUNT_TWO),
		).
		Ensure()
}

// Declaration_Index selects one populated declaration slot.
type Declaration_Index uint8

// Declaration_Index_Invariants covers the two fixed declaration slots.
func Declaration_Index_Invariants(value Declaration_Index, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(DECLARATION_COUNT_NONE),
			uint8(DECLARATION_COUNT_ONE),
		).
		Ensure()
}

// Assignment reports equals instead of declaration.
type Assignment bool

// Assignment_Invariants covers both pipeline binding forms.
func Assignment_Invariants(value Assignment, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Pipeline updates existing variable.").
		Ensure()
}

// Scope_Root reports synthetic dollar variable.
type Scope_Root bool

// Scope_Root_Invariants covers root and named variables.
func Scope_Root_Invariants(value Scope_Root, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Variable is template root dollar.").
		Ensure()
}

// Variable stores one source name and current value.
type Variable struct {
	// Start begins borrowed name.
	Start Node_Value_Start
	// End closes borrowed name.
	End Node_Value_End
	// Root distinguishes dollar from absent zero span.
	Root Scope_Root
	// Value is current binding.
	Value Value
}

// Variable_Invariants composes name metadata and current value.
func Variable_Invariants(value Variable, namespace aver.Namespace) {
	Node_Value_Start_Invariants(value.Start, namespace)
	Node_Value_End_Invariants(value.End, namespace)
	Scope_Root_Invariants(value.Root, namespace)
	Value_Invariants(value.Value, namespace)
}

// DECLARATION_REFERENCE_VALIDATED_FIELD selects one proven variable node.
const DECLARATION_REFERENCE_VALIDATED_FIELD = bytes.SLICE_SIZE_MINIMUM

// DECLARATION_REFERENCE_VALIDATED_FIELD_COUNT fixes declaration transport shape.
const DECLARATION_REFERENCE_VALIDATED_FIELD_COUNT = DECLARATION_REFERENCE_VALIDATED_FIELD +
	STORAGE_FIELD_COUNT_INCREMENT

// Declaration_Reference_Validated_Value retains declaration proof storage.
type Declaration_Reference_Validated_Value uint16

// Declaration_Reference_Validated_Value_Invariants preserves the declaration reference.
func Declaration_Reference_Validated_Value_Invariants(
	value Declaration_Reference_Validated_Value, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(NODE_REFERENCE_MINIMUM),
			uint16(NODE_REFERENCE_MAXIMUM),
		).
		Ensure()
}

// Declaration_Reference_Validated carries one proven variable node by value.
type Declaration_Reference_Validated struct {
	// Value keeps program proof attached to the declaration.
	Value Declaration_Reference_Proof_Stored
}

// Declaration_Reference_Validated_Invariants checks ownership after program traversal.
func Declaration_Reference_Validated_Invariants(
	value Declaration_Reference_Validated, namespace aver.Namespace,
) {
	Declaration_Reference_Proof_Stored_Invariants(value.Value, namespace)
}

// Declaration_First_Reference keeps range index proof independent.
type Declaration_First_Reference uint16

// Declaration_First_Reference_Invariants composes range index proof.
func Declaration_First_Reference_Invariants(
	value Declaration_First_Reference, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(NODE_REFERENCE_MINIMUM),
			uint16(NODE_REFERENCE_MAXIMUM),
		).
		Ensure()
}

// Declaration_Second_Reference keeps range value proof independent.
type Declaration_Second_Reference uint16

// Declaration_Second_Reference_Invariants composes range value proof.
func Declaration_Second_Reference_Invariants(
	value Declaration_Second_Reference, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(NODE_REFERENCE_MINIMUM),
			uint16(NODE_REFERENCE_MAXIMUM),
		).
		Ensure()
}

// Declaration_References stores both range names.
type Declaration_References Declaration_References_Fields

// Declaration_References_Invariants fixes standard range arity storage.
func Declaration_References_Invariants(
	value Declaration_References, namespace aver.Namespace,
) {
	Declaration_First_Reference_Storage_Invariants(value.First, namespace)
	Declaration_Second_Reference_Storage_Invariants(value.Second, namespace)
}

// Separate fields keep declaration identity without dynamic short storage.
func declaration_reference(
	value Declaration_References, index Declaration_Index,
) (reference Declaration_Reference_Validated) {
	defer func() {
		Declaration_Reference_Validated_Invariants(
			reference, "declaration_reference.reference",
		)
	}()
	Declaration_References_Invariants(value, "declaration_reference.value")
	Declaration_Index_Invariants(index, "declaration_reference.index")
	if index == Declaration_Index(DECLARATION_COUNT_NONE) {
		return Declaration_Reference_Validated{
			Value: Declaration_Reference_Validated_Value(
				value.First.Value.(Declaration_First_Reference),
			),
		}
	}
	return Declaration_Reference_Validated{
		Value: Declaration_Reference_Validated_Value(
			value.Second.Value.(Declaration_Second_Reference),
		),
	}
}

// Frame keeps one iterative list or range continuation.
type Frame struct {
	// Kind selects traversal state.
	Kind Frame_Kind
	// Reference is next list child or range body.
	Reference Node_Reference
	// Dot is current cursor or ranged collection.
	Dot Value
	// Index is next range item.
	Index Collection_Index
	// Map_Entry_Index is the previous sorted caller entry.
	Map_Entry_Index Range_Map_Entry_Index
	// Map_Selected distinguishes first map selection from entry zero.
	Map_Selected Map_Selection
	// Variable_Mark restores enclosing bindings.
	Variable_Mark Frame_Variable_Mark
	// Iteration_Variables restores body-local declarations.
	Iteration_Variables Frame_Iteration_Variable_Count
	// Scope_Base restores template variable visibility.
	Scope_Base Frame_Scope_Base
	// Previous_Scope_Base restores caller template scope.
	Previous_Scope_Base Frame_Previous_Scope_Base
	// Declarations retain range variable names.
	Declarations Declaration_References
	// Declaration_Count selects populated names.
	Declaration_Count Declaration_Count
	// Assignment selects range update behavior.
	Assignment Assignment
	// Restore reports scope restoration when frame completes.
	Restore Boolean
}

// Frame_Invariants composes scalar traversal state.
func Frame_Invariants(value Frame, namespace aver.Namespace) {
	Frame_Kind_Invariants(value.Kind, namespace)
	Node_Reference_Invariants(value.Reference, namespace)
	Value_Invariants(value.Dot, namespace)
	Collection_Index_Invariants(value.Index, namespace)
	Range_Map_Entry_Index_Invariants(value.Map_Entry_Index, namespace)
	Map_Selection_Invariants(value.Map_Selected, namespace)
	Frame_Variable_Mark_Invariants(value.Variable_Mark, namespace)
	Frame_Iteration_Variable_Count_Invariants(value.Iteration_Variables, namespace)
	Frame_Scope_Base_Invariants(value.Scope_Base, namespace)
	Frame_Previous_Scope_Base_Invariants(value.Previous_Scope_Base, namespace)
	Declaration_References_Invariants(value.Declarations, namespace)
	Declaration_Count_Invariants(value.Declaration_Count, namespace)
	Assignment_Invariants(value.Assignment, namespace)
	Boolean_Invariants(value.Restore, namespace)
}

// Frame_Stored carries traversal state after workspace validation.
type Frame_Stored Frame

// Frame_Stored_Invariants preserves mutable traversal state.
func Frame_Stored_Invariants(value Frame_Stored, namespace aver.Namespace) {
	Value_Invariants(value.Dot, namespace)
	Declaration_References_Stored_Invariants(
		Declaration_References_Stored(value.Declarations), namespace,
	)
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value.Kind), uint8(FRAME_LIST), uint8(FRAME_RANGE)).
		Range_Uint16(
			uint16(value.Reference), uint16(NODE_REFERENCE_MINIMUM),
			uint16(NODE_REFERENCE_MAXIMUM),
		).
		Range_Uint16(
			uint16(value.Index), uint16(bytes.SLICE_SIZE_MINIMUM),
			uint16(VALUE_COUNT_MAXIMUM),
		).
		Range_Uint16(
			uint16(value.Map_Entry_Index), uint16(bytes.SLICE_SIZE_MINIMUM),
			uint16(RANGE_MAP_ENTRY_INDEX_MAXIMUM),
		).
		Sometimes(bool(value.Map_Selected), "Map traversal selected its first entry.").
		Range_Uint16(
			uint16(value.Variable_Mark), uint16(bytes.SLICE_SIZE_MINIMUM),
			uint16(VARIABLE_COUNT_MAXIMUM),
		).
		Range_Uint16(
			uint16(value.Iteration_Variables), uint16(bytes.SLICE_SIZE_MINIMUM),
			uint16(VARIABLE_COUNT_MAXIMUM),
		).
		Range_Uint16(
			uint16(value.Scope_Base), uint16(bytes.SLICE_SIZE_MINIMUM),
			uint16(VARIABLE_COUNT_MAXIMUM),
		).
		Range_Uint16(
			uint16(value.Previous_Scope_Base), uint16(bytes.SLICE_SIZE_MINIMUM),
			uint16(VARIABLE_COUNT_MAXIMUM),
		).
		Enum_3_Uint8(
			uint8(value.Declaration_Count), uint8(DECLARATION_COUNT_NONE),
			uint8(DECLARATION_COUNT_ONE), uint8(DECLARATION_COUNT_TWO),
		).
		Sometimes(bool(value.Assignment), "Frame updates an existing variable.").
		Sometimes(bool(value.Restore), "Frame restores an enclosing scope.").
		Ensure()
}

// FRAME_STORAGE_FIELD selects one mutable traversal frame.
const FRAME_STORAGE_FIELD = bytes.SLICE_SIZE_MINIMUM

// FRAME_STORAGE_FIELD_COUNT fixes traversal frame ownership shape.
const FRAME_STORAGE_FIELD_COUNT = FRAME_STORAGE_FIELD + STORAGE_FIELD_COUNT_INCREMENT

// Frame_Storage owns one mutable traversal frame without scalar pointers.
type Frame_Storage Frame_Storage_Fields

// Frame_Storage_Invariants checks ownership, not transient traversal state.
func Frame_Storage_Invariants(value Frame_Storage, namespace aver.Namespace) {
	Frame_State_Stored_Invariants(Frame_State_Stored(value.Value), namespace)
}

// Frame_Storage_Pointer keeps traversal mutation typed.
type Frame_Storage_Pointer *Frame_Storage

// Frame_Storage_Pointer_Invariants composes present traversal storage.
func Frame_Storage_Pointer_Invariants(
	value Frame_Storage_Pointer, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Frame_Storage_Invariants(*value, namespace)
}

// Value_Format_Frame keeps one iterative composite output continuation.
type Value_Format_Frame struct {
	// Value is the composite being formatted.
	Value Value
	// Index is the next child or completed child count.
	Index Collection_Index
	// Map_Entry_Index is the previous sorted caller map entry.
	Map_Entry_Index Range_Map_Entry_Index
	// Map_Selected distinguishes first map selection from entry zero.
	Map_Selected Map_Selection
	// Opened reports emitted opening delimiter.
	Opened Composite_Opened
}

// Value_Format_Frame_Invariants composes bounded iterative formatting state.
func Value_Format_Frame_Invariants(
	value Value_Format_Frame, namespace aver.Namespace,
) {
	Value_Invariants(value.Value, namespace)
	Collection_Index_Invariants(value.Index, namespace)
	Range_Map_Entry_Index_Invariants(value.Map_Entry_Index, namespace)
	Map_Selection_Invariants(value.Map_Selected, namespace)
	Composite_Opened_Invariants(value.Opened, namespace)
}

// Value_Format_Frame_Stored carries formatter state after workspace validation.
type Value_Format_Frame_Stored Value_Format_Frame

// Value_Format_Frame_Stored_Invariants preserves mutable formatter state.
func Value_Format_Frame_Stored_Invariants(
	value Value_Format_Frame_Stored, namespace aver.Namespace,
) {
	Value_Invariants(value.Value, namespace)
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value.Index), uint16(bytes.SLICE_SIZE_MINIMUM),
			uint16(VALUE_COUNT_MAXIMUM),
		).
		Range_Uint16(
			uint16(value.Map_Entry_Index), uint16(bytes.SLICE_SIZE_MINIMUM),
			uint16(RANGE_MAP_ENTRY_INDEX_MAXIMUM),
		).
		Sometimes(bool(value.Map_Selected), "Map formatting selected its first entry.").
		Sometimes(
			bool(value.Opened), "Composite formatting emitted its opening delimiter.",
		).
		Ensure()
}

// VALUE_FORMAT_FRAME_STORAGE_FIELD selects one mutable formatter frame.
const VALUE_FORMAT_FRAME_STORAGE_FIELD = bytes.SLICE_SIZE_MINIMUM

// VALUE_FORMAT_FRAME_STORAGE_FIELD_COUNT fixes formatter frame ownership shape.
const VALUE_FORMAT_FRAME_STORAGE_FIELD_COUNT = VALUE_FORMAT_FRAME_STORAGE_FIELD +
	STORAGE_FIELD_COUNT_INCREMENT

// Value_Format_Frame_Storage owns one mutable formatter frame.
type Value_Format_Frame_Storage Value_Format_Frame_Storage_Fields

// Value_Format_Frame_Storage_Invariants checks ownership, not transient formatter state.
func Value_Format_Frame_Storage_Invariants(
	value Value_Format_Frame_Storage, namespace aver.Namespace,
) {
	Value_Format_Frame_State_Stored_Invariants(
		Value_Format_Frame_State_Stored(value.Value), namespace,
	)
}

// Value_Format_Frame_Storage_Pointer keeps formatter mutation typed.
type Value_Format_Frame_Storage_Pointer *Value_Format_Frame_Storage

// Value_Format_Frame_Storage_Pointer_Invariants composes present formatter storage.
func Value_Format_Frame_Storage_Pointer_Invariants(
	value Value_Format_Frame_Storage_Pointer, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Value_Format_Frame_Storage_Invariants(*value, namespace)
}

// Value_Format_Frames is caller-owned composite output stack.
type Value_Format_Frames []Value_Format_Frame_Storage

// Value_Format_Frames_Invariants fixes syntax-derived depth capacity.
func Value_Format_Frames_Invariants(
	value Value_Format_Frames, _ aver.Namespace,
) {
	aver.Always(
		len(value) == FRAME_COUNT_MAXIMUM,
		"Composite output stack covers complete validated value depth.",
	)
}

// Frames is caller-owned traversal stack.
type Frames []Frame_Storage

// Frames_Invariants bounds hostile traversal storage.
func Frames_Invariants(value Frames, namespace aver.Namespace) {
	aver.Always(
		len(value) == FRAME_COUNT_MAXIMUM,
		"Frame storage covers complete syntax-derived bound.",
	)
}

// Variables is caller-owned lexical storage.
type Variables []Variable

// Variables_Invariants bounds hostile lexical storage.
func Variables_Invariants(value Variables, namespace aver.Namespace) {
	aver.Always(
		len(value) == VARIABLE_COUNT_MAXIMUM,
		"Variable storage covers complete lexical bound.",
	)
}

// Arguments is caller-owned function argument storage.
type Arguments []Value

// Arguments_Invariants bounds hostile argument storage.
func Arguments_Invariants(value Arguments, namespace aver.Namespace) {
	aver.Always(
		len(value) == ARGUMENT_COUNT_MAXIMUM,
		"Argument storage covers complete syntax-derived bound.",
	)
}

// Validation_Frames is caller-owned hostile value traversal stack.
type Validation_Frames []Value

// Validation_Frames_Invariants bounds hostile validation storage.
func Validation_Frames_Invariants(
	value Validation_Frames, namespace aver.Namespace,
) {
	aver.Always(
		len(value) == FRAME_COUNT_MAXIMUM,
		"Validation storage covers complete traversal bound.",
	)
}

// WORKSPACE_SCALAR_FIELD selects one internal cursor or diagnostic.
const WORKSPACE_SCALAR_FIELD = bytes.SLICE_SIZE_MINIMUM

// WORKSPACE_SCALAR_FIELD_COUNT fixes mutable scalar storage shape.
const WORKSPACE_SCALAR_FIELD_COUNT = WORKSPACE_SCALAR_FIELD + STORAGE_FIELD_COUNT_INCREMENT

// EVALUATION_NODE_PIPE stores active pipeline syntax.
const EVALUATION_NODE_PIPE = bytes.SLICE_SIZE_MINIMUM

// EVALUATION_NODE_SLOT_INCREMENT adds one syntax slot to evaluation frame.
const EVALUATION_NODE_SLOT_INCREMENT = 1

// EVALUATION_NODE_IDENTIFIER stores callable identifier syntax.
const EVALUATION_NODE_IDENTIFIER = EVALUATION_NODE_PIPE + EVALUATION_NODE_SLOT_INCREMENT

// EVALUATION_NODE_CHAIN stores field path following nested pipeline.
const EVALUATION_NODE_CHAIN = EVALUATION_NODE_IDENTIFIER + EVALUATION_NODE_SLOT_INCREMENT

// EVALUATION_NODE_COUNT fixes syntax slots per evaluation frame.
const EVALUATION_NODE_COUNT = EVALUATION_NODE_CHAIN + EVALUATION_NODE_SLOT_INCREMENT

// EVALUATION_VALUE_DOT stores pipeline cursor.
const EVALUATION_VALUE_DOT = bytes.SLICE_SIZE_MINIMUM

// EVALUATION_VALUE_SLOT_INCREMENT adds one value slot to evaluation frame.
const EVALUATION_VALUE_SLOT_INCREMENT = 1

// EVALUATION_VALUE_CURRENT stores first command result.
const EVALUATION_VALUE_CURRENT = EVALUATION_VALUE_DOT + EVALUATION_VALUE_SLOT_INCREMENT

// EVALUATION_VALUE_COUNT fixes value slots per evaluation frame.
const EVALUATION_VALUE_COUNT = EVALUATION_VALUE_CURRENT + EVALUATION_VALUE_SLOT_INCREMENT

// EVALUATION_REFERENCE_COMMAND stores next pipeline command.
const EVALUATION_REFERENCE_COMMAND = bytes.SLICE_SIZE_MINIMUM

// EVALUATION_REFERENCE_SLOT_INCREMENT adds one syntax-reference slot.
const EVALUATION_REFERENCE_SLOT_INCREMENT = 1

// EVALUATION_REFERENCE_TERM stores next command term.
const EVALUATION_REFERENCE_TERM = EVALUATION_REFERENCE_COMMAND +
	EVALUATION_REFERENCE_SLOT_INCREMENT

// EVALUATION_REFERENCE_COUNT fixes reference slots per evaluation frame.
const EVALUATION_REFERENCE_COUNT = EVALUATION_REFERENCE_TERM +
	EVALUATION_REFERENCE_SLOT_INCREMENT

// EVALUATION_STATE_FIELD stores pipeline result and declarations.
const EVALUATION_STATE_FIELD = bytes.SLICE_SIZE_MINIMUM

// EVALUATION_STATE_FIELD_COUNT fixes pipeline state storage shape.
const EVALUATION_STATE_FIELD_COUNT = EVALUATION_STATE_FIELD + STORAGE_FIELD_COUNT_INCREMENT

// EVALUATION_ARGUMENT_BASE_FIELD stores caller argument-stack opening.
const EVALUATION_ARGUMENT_BASE_FIELD = bytes.SLICE_SIZE_MINIMUM

// EVALUATION_ARGUMENT_BASE_FIELD_COUNT fixes argument opening storage shape.
const EVALUATION_ARGUMENT_BASE_FIELD_COUNT = EVALUATION_ARGUMENT_BASE_FIELD +
	STORAGE_FIELD_COUNT_INCREMENT

// EVALUATION_CONTROL_STAGE stores current state-machine stage.
const EVALUATION_CONTROL_STAGE = bytes.SLICE_SIZE_MINIMUM

// EVALUATION_CONTROL_SLOT_INCREMENT adds one state-machine metadata slot.
const EVALUATION_CONTROL_SLOT_INCREMENT = 1

// EVALUATION_CONTROL_MODE stores identifier or value call mode.
const EVALUATION_CONTROL_MODE = EVALUATION_CONTROL_STAGE + EVALUATION_CONTROL_SLOT_INCREMENT

// EVALUATION_CONTROL_PRESENT stores prior pipeline result presence.
const EVALUATION_CONTROL_PRESENT = EVALUATION_CONTROL_MODE + EVALUATION_CONTROL_SLOT_INCREMENT

// EVALUATION_CONTROL_CHAIN stores pending field-chain presence.
const EVALUATION_CONTROL_CHAIN = EVALUATION_CONTROL_PRESENT + EVALUATION_CONTROL_SLOT_INCREMENT

// EVALUATION_CONTROL_COUNT fixes state-machine metadata shape.
const EVALUATION_CONTROL_COUNT = EVALUATION_CONTROL_CHAIN + EVALUATION_CONTROL_SLOT_INCREMENT

// Evaluation_Stage identifies one evaluator state-machine step.
type Evaluation_Stage uint8

// Evaluation_Stage_Invariants bounds suspended evaluator progress.
func Evaluation_Stage_Invariants(value Evaluation_Stage, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(
			uint8(value), uint8(EVALUATION_STAGE_COMMAND), uint8(EVALUATION_STAGE_DONE),
		).
		Ensure()
}

// Evaluation_Mode selects identifier or evaluated-value command head.
type Evaluation_Mode uint8

// Evaluation_Mode_Invariants lists both command-head forms.
func Evaluation_Mode_Invariants(value Evaluation_Mode, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(EVALUATION_MODE_VALUE),
			uint8(EVALUATION_MODE_IDENTIFIER),
		).
		Ensure()
}

// Evaluation_Present records prior pipeline result presence.
type Evaluation_Present uint8

// Evaluation_Present_Invariants lists both result states.
func Evaluation_Present_Invariants(value Evaluation_Present, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(EVALUATION_CONTROL_FALSE),
			uint8(EVALUATION_CONTROL_TRUE),
		).
		Ensure()
}

// Evaluation_Chain records pending field selection.
type Evaluation_Chain uint8

// Evaluation_Chain_Invariants lists both chain states.
func Evaluation_Chain_Invariants(value Evaluation_Chain, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(EVALUATION_CONTROL_FALSE),
			uint8(EVALUATION_CONTROL_TRUE),
		).
		Ensure()
}

// EVALUATION_STAGE_COMMAND begins next command or finishes pipeline.
const EVALUATION_STAGE_COMMAND Evaluation_Stage = Evaluation_Stage(bits.WORD_8_MINIMUM)

// EVALUATION_STAGE_INCREMENT advances one pipeline-evaluation stage.
const EVALUATION_STAGE_INCREMENT Evaluation_Stage = 1

// EVALUATION_STAGE_FIRST evaluates nonidentifier command head.
const EVALUATION_STAGE_FIRST = Evaluation_Stage(
	EVALUATION_STAGE_COMMAND + EVALUATION_STAGE_INCREMENT,
)

// EVALUATION_STAGE_ARGUMENT evaluates remaining command terms.
const EVALUATION_STAGE_ARGUMENT = Evaluation_Stage(
	EVALUATION_STAGE_FIRST + EVALUATION_STAGE_INCREMENT,
)

// EVALUATION_STAGE_CALL invokes command target.
const EVALUATION_STAGE_CALL = Evaluation_Stage(
	EVALUATION_STAGE_ARGUMENT + EVALUATION_STAGE_INCREMENT,
)

// EVALUATION_STAGE_DONE binds declarations and returns pipeline value.
const EVALUATION_STAGE_DONE Evaluation_Stage = EVALUATION_STAGE_CALL + EVALUATION_STAGE_INCREMENT

// EVALUATION_MODE_VALUE treats command head as evaluated value.
const EVALUATION_MODE_VALUE Evaluation_Mode = Evaluation_Mode(bits.WORD_8_MINIMUM)

// EVALUATION_MODE_INCREMENT adds one command-head evaluation mode.
const EVALUATION_MODE_INCREMENT Evaluation_Mode = 1

// EVALUATION_MODE_IDENTIFIER treats command head as named function.
const EVALUATION_MODE_IDENTIFIER Evaluation_Mode = EVALUATION_MODE_VALUE + EVALUATION_MODE_INCREMENT

// EVALUATION_CONTROL_FALSE clears one state-machine flag.
const EVALUATION_CONTROL_FALSE = 0

// EVALUATION_CONTROL_TRUE sets one state-machine flag.
const EVALUATION_CONTROL_TRUE = 1

// Evaluation_Pipe_Node keeps an optional caller-owned pipeline node.
type Evaluation_Pipe_Node interface{}

// Evaluation_Pipe_Node_Invariants accepts absent scratch or one pipeline-node pointer.
func Evaluation_Pipe_Node_Invariants(
	value Evaluation_Pipe_Node, _ aver.Namespace,
) {
	_, valid := value.(*Node)
	aver.Always(valid == (value != nil), "Evaluation pipeline node has expected type.")
}

// Evaluation_Identifier_Node keeps an optional caller-owned callable node.
type Evaluation_Identifier_Node interface{}

// Evaluation_Identifier_Node_Invariants accepts absent scratch or one callable-node pointer.
func Evaluation_Identifier_Node_Invariants(
	value Evaluation_Identifier_Node, _ aver.Namespace,
) {
	_, valid := value.(*Node)
	aver.Always(valid == (value != nil), "Evaluation callable node has expected type.")
}

// Evaluation_Chain_Node keeps an optional caller-owned field-chain node.
type Evaluation_Chain_Node interface{}

// Evaluation_Chain_Node_Invariants accepts absent scratch or one chain-node pointer.
func Evaluation_Chain_Node_Invariants(
	value Evaluation_Chain_Node, _ aver.Namespace,
) {
	_, valid := value.(*Node)
	aver.Always(valid == (value != nil), "Evaluation chain node has expected type.")
}

// Evaluation_Nodes stores active pipe, identifier, and chain syntax.
type Evaluation_Nodes struct {
	// Pipe retains pipeline syntax while a child waits.
	Pipe Evaluation_Pipe_Node
	// Identifier retains callable syntax while arguments run.
	Identifier Evaluation_Identifier_Node
	// Chain retains pending field syntax while a child runs.
	Chain Evaluation_Chain_Node
}

// Evaluation_Nodes_Invariants fixes iterative syntax-frame shape.
func Evaluation_Nodes_Invariants(value Evaluation_Nodes, namespace aver.Namespace) {
	Evaluation_Pipe_Node_Invariants(value.Pipe, namespace)
	Evaluation_Identifier_Node_Invariants(value.Identifier, namespace)
	Evaluation_Chain_Node_Invariants(value.Chain, namespace)
}

// Evaluation_Dot_Value keeps cursor identity independent.
type Evaluation_Dot_Value Value

// Evaluation_Dot_Value_Invariants composes cursor value.
func Evaluation_Dot_Value_Invariants(value Evaluation_Dot_Value, namespace aver.Namespace) {
	Evaluation_Dot_Kind_Storage_Invariants(
		Evaluation_Dot_Kind_Storage(value.Kind), namespace,
	)
	Evaluation_Dot_Scalar_Storage_Invariants(
		Evaluation_Dot_Scalar_Storage(value.Scalar), namespace,
	)
	Evaluation_Dot_Text_Storage_Invariants(
		Evaluation_Dot_Text_Storage(value.Text), namespace,
	)
	Evaluation_Dot_Values_Storage_Invariants(
		Evaluation_Dot_Values_Storage(value.Values), namespace,
	)
	Evaluation_Dot_Fields_Storage_Invariants(
		Evaluation_Dot_Fields_Storage(value.Fields), namespace,
	)
	Evaluation_Dot_Function_Storage_Invariants(
		Evaluation_Dot_Function_Storage(value.Function), namespace,
	)
}

// Evaluation_Dot_Kind_Storage prevents a mutable cursor from proving stale contents.
type Evaluation_Dot_Kind_Storage interface{}

// Evaluation_Dot_Kind_Storage_Invariants fixes the cursor selector representation.
func Evaluation_Dot_Kind_Storage_Invariants(value Evaluation_Dot_Kind_Storage, _ aver.Namespace) {
	_, valid := value.(Value_Kind_Storage)
	aver.Always(valid == (value != nil), "Evaluation cursor kind storage has expected type.")
}

// Evaluation_Dot_Scalar_Storage prevents a mutable cursor from proving stale contents.
type Evaluation_Dot_Scalar_Storage interface{}

// Evaluation_Dot_Scalar_Storage_Invariants fixes the cursor scalar representation.
func Evaluation_Dot_Scalar_Storage_Invariants(
	value Evaluation_Dot_Scalar_Storage, _ aver.Namespace,
) {
	_, valid := value.(Value_Scalar)
	aver.Always(valid == (value != nil), "Evaluation cursor scalar storage has expected type.")
}

// Evaluation_Dot_Text_Storage prevents a mutable cursor from proving stale contents.
type Evaluation_Dot_Text_Storage interface{}

// Evaluation_Dot_Text_Storage_Invariants fixes the cursor text representation.
func Evaluation_Dot_Text_Storage_Invariants(value Evaluation_Dot_Text_Storage, _ aver.Namespace) {
	_, valid := value.(Text_Storage)
	aver.Always(valid == (value != nil), "Evaluation cursor text storage has expected type.")
}

// Evaluation_Dot_Values_Storage prevents a mutable cursor from proving stale contents.
type Evaluation_Dot_Values_Storage interface{}

// Evaluation_Dot_Values_Storage_Invariants fixes the cursor sequence representation.
func Evaluation_Dot_Values_Storage_Invariants(
	value Evaluation_Dot_Values_Storage, _ aver.Namespace,
) {
	_, valid := value.(Values_Storage)
	aver.Always(
		valid == (value != nil), "Evaluation cursor sequence storage has expected type.",
	)
}

// Evaluation_Dot_Fields_Storage prevents a mutable cursor from proving stale contents.
type Evaluation_Dot_Fields_Storage interface{}

// Evaluation_Dot_Fields_Storage_Invariants fixes the cursor field representation.
func Evaluation_Dot_Fields_Storage_Invariants(
	value Evaluation_Dot_Fields_Storage, _ aver.Namespace,
) {
	_, valid := value.(Fields_Storage)
	aver.Always(valid == (value != nil), "Evaluation cursor field storage has expected type.")
}

// Evaluation_Dot_Function_Storage prevents a mutable cursor from proving stale contents.
type Evaluation_Dot_Function_Storage interface{}

// Evaluation_Dot_Function_Storage_Invariants fixes the cursor callable representation.
func Evaluation_Dot_Function_Storage_Invariants(
	value Evaluation_Dot_Function_Storage, _ aver.Namespace,
) {
	_, valid := value.(Function_Storage)
	aver.Always(
		valid == (value != nil), "Evaluation cursor callable storage has expected type.",
	)
}

// Evaluation_Current_Value keeps command result identity independent.
type Evaluation_Current_Value Value

// Evaluation_Current_Value_Invariants composes command result.
func Evaluation_Current_Value_Invariants(
	value Evaluation_Current_Value, namespace aver.Namespace,
) {
	Evaluation_Current_Kind_Storage_Invariants(
		Evaluation_Current_Kind_Storage(value.Kind), namespace,
	)
	Evaluation_Current_Scalar_Storage_Invariants(
		Evaluation_Current_Scalar_Storage(value.Scalar), namespace,
	)
	Evaluation_Current_Text_Storage_Invariants(
		Evaluation_Current_Text_Storage(value.Text), namespace,
	)
	Evaluation_Current_Values_Storage_Invariants(
		Evaluation_Current_Values_Storage(value.Values), namespace,
	)
	Evaluation_Current_Fields_Storage_Invariants(
		Evaluation_Current_Fields_Storage(value.Fields), namespace,
	)
	Evaluation_Current_Function_Storage_Invariants(
		Evaluation_Current_Function_Storage(value.Function), namespace,
	)
}

// Evaluation_Current_Kind_Storage prevents a mutable result from proving stale contents.
type Evaluation_Current_Kind_Storage interface{}

// Evaluation_Current_Kind_Storage_Invariants fixes the current selector representation.
func Evaluation_Current_Kind_Storage_Invariants(
	value Evaluation_Current_Kind_Storage, _ aver.Namespace,
) {
	_, valid := value.(Value_Kind_Storage)
	aver.Always(valid == (value != nil), "Evaluation current kind storage has expected type.")
}

// Evaluation_Current_Scalar_Storage prevents a mutable result from proving stale contents.
type Evaluation_Current_Scalar_Storage interface{}

// Evaluation_Current_Scalar_Storage_Invariants fixes the current scalar representation.
func Evaluation_Current_Scalar_Storage_Invariants(
	value Evaluation_Current_Scalar_Storage, _ aver.Namespace,
) {
	_, valid := value.(Value_Scalar)
	aver.Always(valid == (value != nil), "Evaluation current scalar storage has expected type.")
}

// Evaluation_Current_Text_Storage prevents a mutable result from proving stale contents.
type Evaluation_Current_Text_Storage interface{}

// Evaluation_Current_Text_Storage_Invariants fixes the current text representation.
func Evaluation_Current_Text_Storage_Invariants(
	value Evaluation_Current_Text_Storage, _ aver.Namespace,
) {
	_, valid := value.(Text_Storage)
	aver.Always(valid == (value != nil), "Evaluation current text storage has expected type.")
}

// Evaluation_Current_Values_Storage prevents a mutable result from proving stale contents.
type Evaluation_Current_Values_Storage interface{}

// Evaluation_Current_Values_Storage_Invariants fixes the current sequence representation.
func Evaluation_Current_Values_Storage_Invariants(
	value Evaluation_Current_Values_Storage, _ aver.Namespace,
) {
	_, valid := value.(Values_Storage)
	aver.Always(
		valid == (value != nil), "Evaluation current sequence storage has expected type.",
	)
}

// Evaluation_Current_Fields_Storage prevents a mutable result from proving stale contents.
type Evaluation_Current_Fields_Storage interface{}

// Evaluation_Current_Fields_Storage_Invariants fixes the current field representation.
func Evaluation_Current_Fields_Storage_Invariants(
	value Evaluation_Current_Fields_Storage, _ aver.Namespace,
) {
	_, valid := value.(Fields_Storage)
	aver.Always(valid == (value != nil), "Evaluation current field storage has expected type.")
}

// Evaluation_Current_Function_Storage prevents a mutable result from proving stale contents.
type Evaluation_Current_Function_Storage interface{}

// Evaluation_Current_Function_Storage_Invariants fixes the current callable representation.
func Evaluation_Current_Function_Storage_Invariants(
	value Evaluation_Current_Function_Storage, _ aver.Namespace,
) {
	_, valid := value.(Function_Storage)
	aver.Always(
		valid == (value != nil), "Evaluation current callable storage has expected type.",
	)
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
		Range_Uint16(
			uint16(value), uint16(NODE_REFERENCE_MINIMUM),
			uint16(NODE_REFERENCE_MAXIMUM),
		).
		Ensure()
}

// Evaluation_Term_Reference keeps term continuation identity independent.
type Evaluation_Term_Reference Node_Reference

// Evaluation_Term_Reference_Invariants composes term continuation.
func Evaluation_Term_Reference_Invariants(
	value Evaluation_Term_Reference, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(NODE_REFERENCE_MINIMUM),
			uint16(NODE_REFERENCE_MAXIMUM),
		).
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
func Evaluation_References_Invariants(
	value Evaluation_References, namespace aver.Namespace,
) {
	Evaluation_Command_Reference_Invariants(value.Command, namespace)
	Evaluation_Term_Reference_Invariants(value.Term, namespace)
}

// Evaluation_State_Value keeps pipeline state independent from both evaluation cursors.
type Evaluation_State_Value Value

// Evaluation_State_Value_Invariants composes pipeline state through unique proof subjects.
func Evaluation_State_Value_Invariants(
	value Evaluation_State_Value, namespace aver.Namespace,
) {
	Evaluation_State_Kind_Storage_Invariants(
		Evaluation_State_Kind_Storage(value.Kind), namespace,
	)
	Evaluation_State_Scalar_Storage_Invariants(
		Evaluation_State_Scalar_Storage(value.Scalar), namespace,
	)
	Evaluation_State_Text_Storage_Invariants(
		Evaluation_State_Text_Storage(value.Text), namespace,
	)
	Evaluation_State_Values_Storage_Invariants(
		Evaluation_State_Values_Storage(value.Values), namespace,
	)
	Evaluation_State_Fields_Storage_Invariants(
		Evaluation_State_Fields_Storage(value.Fields), namespace,
	)
	Evaluation_State_Function_Storage_Invariants(
		Evaluation_State_Function_Storage(value.Function), namespace,
	)
}

// Evaluation_State_Kind_Storage prevents mutable state from proving stale contents.
type Evaluation_State_Kind_Storage interface{}

// Evaluation_State_Kind_Storage_Invariants fixes the state selector representation.
func Evaluation_State_Kind_Storage_Invariants(
	value Evaluation_State_Kind_Storage, _ aver.Namespace,
) {
	_, valid := value.(Value_Kind_Storage)
	aver.Always(valid == (value != nil), "Evaluation state kind storage has expected type.")
}

// Evaluation_State_Scalar_Storage prevents mutable state from proving stale contents.
type Evaluation_State_Scalar_Storage interface{}

// Evaluation_State_Scalar_Storage_Invariants fixes the state scalar representation.
func Evaluation_State_Scalar_Storage_Invariants(
	value Evaluation_State_Scalar_Storage, _ aver.Namespace,
) {
	_, valid := value.(Value_Scalar)
	aver.Always(valid == (value != nil), "Evaluation state scalar storage has expected type.")
}

// Evaluation_State_Text_Storage prevents mutable state from proving stale contents.
type Evaluation_State_Text_Storage interface{}

// Evaluation_State_Text_Storage_Invariants fixes the state text representation.
func Evaluation_State_Text_Storage_Invariants(
	value Evaluation_State_Text_Storage, _ aver.Namespace,
) {
	_, valid := value.(Text_Storage)
	aver.Always(valid == (value != nil), "Evaluation state text storage has expected type.")
}

// Evaluation_State_Values_Storage prevents mutable state from proving stale contents.
type Evaluation_State_Values_Storage interface{}

// Evaluation_State_Values_Storage_Invariants fixes the state sequence representation.
func Evaluation_State_Values_Storage_Invariants(
	value Evaluation_State_Values_Storage, _ aver.Namespace,
) {
	_, valid := value.(Values_Storage)
	aver.Always(valid == (value != nil), "Evaluation state sequence storage has expected type.")
}

// Evaluation_State_Fields_Storage prevents mutable state from proving stale contents.
type Evaluation_State_Fields_Storage interface{}

// Evaluation_State_Fields_Storage_Invariants fixes the state field representation.
func Evaluation_State_Fields_Storage_Invariants(
	value Evaluation_State_Fields_Storage, _ aver.Namespace,
) {
	_, valid := value.(Fields_Storage)
	aver.Always(valid == (value != nil), "Evaluation state field storage has expected type.")
}

// Evaluation_State_Function_Storage prevents mutable state from proving stale contents.
type Evaluation_State_Function_Storage interface{}

// Evaluation_State_Function_Storage_Invariants fixes the state callable representation.
func Evaluation_State_Function_Storage_Invariants(
	value Evaluation_State_Function_Storage, _ aver.Namespace,
) {
	_, valid := value.(Function_Storage)
	aver.Always(valid == (value != nil), "Evaluation state callable storage has expected type.")
}

// Evaluation_States stores one pipeline state.
type Pipeline_State_Stored Pipeline_State

// Pipeline_State_Stored_Invariants preserves mutable pipeline state.
func Pipeline_State_Stored_Invariants(
	value Pipeline_State_Stored, namespace aver.Namespace,
) {
	Evaluation_State_Value_Invariants(Evaluation_State_Value(value.Value), namespace)
	Declaration_References_Stored_Invariants(
		Declaration_References_Stored(value.Declarations), namespace,
	)
	aver.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value.Declaration_Count), uint8(DECLARATION_COUNT_NONE),
			uint8(DECLARATION_COUNT_ONE), uint8(DECLARATION_COUNT_TWO),
		).
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
		Range_Uint16(
			uint16(value), uint16(ARGUMENT_COUNT_MINIMUM),
			uint16(ARGUMENT_COUNT_MAXIMUM),
		).
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
		Evaluation_References_Stored(value.References), namespace,
	)
	Evaluation_States_Invariants(value.States, namespace)
	Evaluation_Argument_Bases_Stored_Invariants(
		Evaluation_Argument_Bases_Stored(value.Argument_Bases), namespace,
	)
	Evaluation_Control_Stored_Invariants(Evaluation_Control_Stored(value.Control), namespace)
}

// Evaluation_Frame_Pointer keeps evaluator mutation typed.
type Evaluation_Frame_Pointer *Evaluation_Frame

// Evaluation_Frame_Pointer_Invariants composes present evaluator storage.
func Evaluation_Frame_Pointer_Invariants(
	value Evaluation_Frame_Pointer, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Evaluation_Frame_Invariants(*value, namespace)
}

// Evaluations is caller-owned nested pipeline storage.
type Evaluations []Evaluation_Frame

// Evaluations_Invariants requires complete syntax-derived nesting capacity.
func Evaluations_Invariants(value Evaluations, _ aver.Namespace) {
	aver.Always(
		len(value) == FRAME_COUNT_MAXIMUM,
		"Evaluation storage covers complete syntax-derived depth.",
	)
}

// Workspace_Literal_Count_Storage_Value retains decoded-cursor storage.
type Workspace_Literal_Count_Storage_Value uint16

// Workspace_Literal_Count_Storage_Value_Invariants preserves cursor encoding.
func Workspace_Literal_Count_Storage_Value_Invariants(
	value Workspace_Literal_Count_Storage_Value, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(bytes.SLICE_SIZE_MINIMUM),
			uint16(QUOTED_DECODED_SIZE_MAXIMUM),
		).
		Ensure()
}

// Workspace_Literal_Count_Storage keeps decoded cursor corruption detectable.
type Workspace_Literal_Count_Stored interface{}

// Workspace_Literal_Count_Stored_Invariants accepts absent or typed literal state.
func Workspace_Literal_Count_Stored_Invariants(
	value Workspace_Literal_Count_Stored, _ aver.Namespace,
) {
	_, valid := value.(Workspace_Literal_Count_Storage_Value)
	aver.Always(valid == (value != nil), "Stored literal cursor has expected type.")
}

// Workspace_Literal_Count_Storage keeps decoded cursor corruption detectable.
type Workspace_Literal_Count_Storage struct {
	// Value keeps mutation inside execution workspace.
	Value Workspace_Literal_Count_Stored
}

// Workspace_Literal_Count_Storage_Invariants fixes decoded cursor shape.
func Workspace_Literal_Count_Storage_Invariants(
	value Workspace_Literal_Count_Storage, namespace aver.Namespace,
) {
	Workspace_Literal_Count_Stored_Invariants(value.Value, namespace)
}

// Workspace_Frame_Count_Storage_Value retains traversal-cursor storage.
type Workspace_Frame_Count_Storage_Value uint16

// Workspace_Frame_Count_Storage_Value_Invariants preserves cursor encoding.
func Workspace_Frame_Count_Storage_Value_Invariants(
	value Workspace_Frame_Count_Storage_Value, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(bytes.SLICE_SIZE_MINIMUM),
			uint16(FRAME_COUNT_MAXIMUM),
		).
		Ensure()
}

// Workspace_Frame_Count_Storage keeps traversal cursor corruption detectable.
type Workspace_Frame_Count_Stored interface{}

// Workspace_Frame_Count_Stored_Invariants accepts absent or typed frame state.
func Workspace_Frame_Count_Stored_Invariants(
	value Workspace_Frame_Count_Stored, _ aver.Namespace,
) {
	_, valid := value.(Workspace_Frame_Count_Storage_Value)
	aver.Always(valid == (value != nil), "Stored frame cursor has expected type.")
}

// Workspace_Frame_Count_Storage keeps traversal cursor corruption detectable.
type Workspace_Frame_Count_Storage struct {
	// Value keeps mutation inside execution workspace.
	Value Workspace_Frame_Count_Stored
}

// Workspace_Frame_Count_Storage_Invariants fixes traversal cursor shape.
func Workspace_Frame_Count_Storage_Invariants(
	value Workspace_Frame_Count_Storage, namespace aver.Namespace,
) {
	Workspace_Frame_Count_Stored_Invariants(value.Value, namespace)
}

// Workspace_Variable_Count_Storage_Value retains binding-cursor storage.
type Workspace_Variable_Count_Storage_Value uint16

// Workspace_Variable_Count_Storage_Value_Invariants preserves cursor encoding.
func Workspace_Variable_Count_Storage_Value_Invariants(
	value Workspace_Variable_Count_Storage_Value, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(VARIABLE_COUNT_MINIMUM),
			uint16(VARIABLE_COUNT_MAXIMUM),
		).
		Ensure()
}

// Workspace_Variable_Count_Storage keeps binding cursor corruption detectable.
type Workspace_Variable_Count_Stored interface{}

// Workspace_Variable_Count_Stored_Invariants accepts absent or typed binding state.
func Workspace_Variable_Count_Stored_Invariants(
	value Workspace_Variable_Count_Stored, _ aver.Namespace,
) {
	_, valid := value.(Workspace_Variable_Count_Storage_Value)
	aver.Always(valid == (value != nil), "Stored variable cursor has expected type.")
}

// Workspace_Variable_Count_Storage keeps binding cursor corruption detectable.
type Workspace_Variable_Count_Storage struct {
	// Value keeps mutation inside execution workspace.
	Value Workspace_Variable_Count_Stored
}

// Workspace_Variable_Count_Storage_Invariants fixes binding cursor shape.
func Workspace_Variable_Count_Storage_Invariants(
	value Workspace_Variable_Count_Storage, namespace aver.Namespace,
) {
	Workspace_Variable_Count_Stored_Invariants(value.Value, namespace)
}

// Workspace_Scope_Base_Storage_Value retains visibility-cursor storage.
type Workspace_Scope_Base_Storage_Value uint16

// Workspace_Scope_Base_Storage_Value_Invariants preserves cursor encoding.
func Workspace_Scope_Base_Storage_Value_Invariants(
	value Workspace_Scope_Base_Storage_Value, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(VARIABLE_COUNT_MINIMUM),
			uint16(VARIABLE_COUNT_MAXIMUM),
		).
		Ensure()
}

// Workspace_Scope_Base_Storage keeps visibility cursor corruption detectable.
type Workspace_Scope_Base_Stored interface{}

// Workspace_Scope_Base_Stored_Invariants accepts absent or typed scope state.
func Workspace_Scope_Base_Stored_Invariants(
	value Workspace_Scope_Base_Stored, _ aver.Namespace,
) {
	_, valid := value.(Workspace_Scope_Base_Storage_Value)
	aver.Always(valid == (value != nil), "Stored scope cursor has expected type.")
}

// Workspace_Scope_Base_Storage keeps visibility cursor corruption detectable.
type Workspace_Scope_Base_Storage struct {
	// Value keeps mutation inside execution workspace.
	Value Workspace_Scope_Base_Stored
}

// Workspace_Scope_Base_Storage_Invariants fixes visibility cursor shape.
func Workspace_Scope_Base_Storage_Invariants(
	value Workspace_Scope_Base_Storage, namespace aver.Namespace,
) {
	Workspace_Scope_Base_Stored_Invariants(value.Value, namespace)
}

// Workspace_Diagnostic_Storage keeps first refusal corruption detectable.
type Diagnostic_Stored Diagnostic

// Diagnostic_Stored_Invariants preserves bounded diagnostic storage.
func Diagnostic_Stored_Invariants(value Diagnostic_Stored, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(
			uint8(value.Code), uint8(STATUS_OK), uint8(STATUS_LIMIT_EXCEEDED),
		).
		Range_Uint16(
			uint16(value.Position), uint16(bytes.SLICE_SIZE_MINIMUM),
			uint16(SOURCE_SIZE_MAXIMUM),
		).
		Ensure()
}

// Workspace_Diagnostic_Fields keeps first refusal state before workspace validation.
type Workspace_Diagnostic_Fields struct {
	// Value keeps mutation inside execution workspace.
	Value Diagnostic_Stored
}

// Workspace_Diagnostic_Fields_Invariants bounds hostile diagnostic storage.
func Workspace_Diagnostic_Fields_Invariants(
	value Workspace_Diagnostic_Fields, namespace aver.Namespace,
) {
	Diagnostic_Stored_Invariants(value.Value, namespace)
}

// Workspace_Diagnostic_Stored separates stale failure state from active diagnostics.
type Workspace_Diagnostic_Stored interface{}

// Workspace_Diagnostic_Stored_Invariants fixes diagnostic representation.
func Workspace_Diagnostic_Stored_Invariants(value Workspace_Diagnostic_Stored, _ aver.Namespace) {
	_, valid := value.(Diagnostic_Stored)
	aver.Always(valid == (value != nil), "Workspace diagnostic has expected storage type.")
}

// Workspace_Diagnostic_Storage keeps first refusal corruption detectable.
type Workspace_Diagnostic_Storage Workspace_Diagnostic_Fields

// Workspace_Diagnostic_Storage_Invariants fixes first-refusal shape.
func Workspace_Diagnostic_Storage_Invariants(
	value Workspace_Diagnostic_Storage, namespace aver.Namespace,
) {
	Workspace_Diagnostic_Stored_Invariants(
		Workspace_Diagnostic_Stored(value.Value), namespace,
	)
}

// Float_Parse_Workspace keeps parser scratch inline in execution storage.
type Float_Parse_Workspace big.Float_Parse_Workspace

// Float_Parse_Workspace_Invariants preserves parser storage checks without a staged copy.
func Float_Parse_Workspace_Invariants(
	value Float_Parse_Workspace, namespace aver.Namespace,
) {
	Float_Parse_Source_Count_Invariants(
		Float_Parse_Source_Count(len(value.Source)), namespace,
	)
	Float_Parse_Control_Count_Invariants(
		Float_Parse_Control_Count(len(value.Control)), namespace,
	)
}

// FLOAT_PARSE_SOURCE_COUNT_COMPLETE is required source scratch size.
const FLOAT_PARSE_SOURCE_COUNT_COMPLETE Float_Parse_Source_Count = big.FLOAT_PARSE_TEXT_SIZE_MAXIMUM

// Float_Parse_Source_Count keeps parser source proof independent.
type Float_Parse_Source_Count int

// Float_Parse_Source_Count_Invariants requires complete source scratch.
func Float_Parse_Source_Count_Invariants(
	value Float_Parse_Source_Count, _ aver.Namespace,
) {
	aver.Always(
		int(value) == int(FLOAT_PARSE_SOURCE_COUNT_COMPLETE),
		"Execution float source scratch is complete.",
	)
}

// FLOAT_PARSE_CONTROL_COUNT_COMPLETE is required control scratch size.
const FLOAT_PARSE_CONTROL_COUNT_COMPLETE Float_Parse_Control_Count = big.FLOAT_PARSE_CONTROL_COUNT

// Float_Parse_Control_Count keeps parser control proof independent.
type Float_Parse_Control_Count int

// Float_Parse_Control_Count_Invariants requires complete control scratch.
func Float_Parse_Control_Count_Invariants(
	value Float_Parse_Control_Count, _ aver.Namespace,
) {
	aver.Always(
		int(value) == int(FLOAT_PARSE_CONTROL_COUNT_COMPLETE),
		"Execution float control scratch is complete.",
	)
}

// Float_Text_Workspace keeps formatter scratch inline in execution storage.
type Float_Text_Workspace big.Float_Text_Workspace

// Float_Text_Workspace_Invariants preserves formatter shape without a staged copy.
func Float_Text_Workspace_Invariants(
	value Float_Text_Workspace, _ aver.Namespace,
) {
	aver.Always(
		len(value.Integers) == big.FLOAT_TEXT_INTEGER_COUNT,
		"Float text scratch owns both integer conversions.",
	)
	aver.Always(
		len(value.Values) == big.FLOAT_TEXT_VALUE_COUNT,
		"Float text scratch retains source and rounded values.",
	)
	aver.Always(
		len(value.Text) == big.FLOAT_TEXT_INTEGER_WORKSPACE_COUNT,
		"Float text scratch owns both integer text conversions.",
	)
	aver.Always(
		len(value.Control) == big.FLOAT_TEXT_CONTROL_COUNT,
		"Float text scratch owns its bounded scalar state.",
	)
	aver.Always(
		len(value.Decimals) == big.FLOAT_DECIMAL_COUNT,
		"Float text scratch owns exact value and rounding bounds.",
	)
	aver.Always(
		len(value.Magnitude.Words) == big.FLOAT_TEXT_MAGNITUDE_WORD_COUNT_MAXIMUM,
		"Float text midpoint retains its guard bit.",
	)
	aver.Always(
		len(value.Magnitude.Control) == big.FLOAT_TEXT_MAGNITUDE_CONTROL_COUNT,
		"Float text midpoint retains its active word count.",
	)
	aver.Always(
		len(value.Output) == big.FLOAT_TEXT_SIZE_MAXIMUM,
		"Float text output remains transactional.",
	)
}

// Workspace owns every execution mutation and scratch byte.
type Workspace struct {
	// Output stages atomic result.
	Output Output_Storage
	// Literals decode quoted source without ownership.
	Literals Literal_Storage
	// Literal_Count is populated decoded storage.
	Literal_Count Workspace_Literal_Count_Storage
	// Literal_Counts stores decoded bytes per node.
	Literal_Counts Literal_Counts
	// Literal_Offsets stores decoded opening per node.
	Literal_Offsets Literal_Offsets
	// Literal_Decoded avoids repeat work inside loops.
	Literal_Decoded Literal_Decoded
	// Name_Left decodes invocation name.
	Name_Left Name_Left_Storage
	// Name_Right decodes definition name.
	Name_Right Name_Right_Storage
	// Number formats integer output.
	Number Number_Storage
	// Generated retains derived builtin text for one pipeline.
	Generated Generated_Storage
	// Generated_Count is populated derived text.
	Generated_Count Workspace_Generated_Count_Storage
	// Float_Parse stays inline because workspace already owns scratch lifetime.
	Float_Parse Float_Parse_Workspace
	// Float_Text stays inline because workspace already owns scratch lifetime.
	Float_Text Float_Text_Workspace
	// Frames hold iterative traversal.
	Frames Frames
	// Frame_Count is active traversal depth.
	Frame_Count Workspace_Frame_Count_Storage
	// Variables hold lexical bindings.
	Variables Variables
	// Variable_Count is populated binding count.
	Variable_Count Workspace_Variable_Count_Storage
	// Scope_Base hides caller variables during template invocation.
	Scope_Base Workspace_Scope_Base_Storage
	// Arguments stage callback values.
	Arguments Arguments
	// Validation holds hostile nested values.
	Validation Validation_Frames
	// Evaluations hold suspended nested pipelines.
	Evaluations Evaluations
	// Format holds iterative composite output continuations.
	Format Value_Format_Frames
	// Diagnostic retains first execution refusal.
	Diagnostic Workspace_Diagnostic_Storage
}

// Workspace_Invariants verifies caller storage shape and scalar cursors.
func Workspace_Invariants(value Workspace, namespace aver.Namespace) {
	Output_Storage_Invariants(value.Output, namespace)
	Literal_Storage_Invariants(value.Literals, namespace)
	Workspace_Literal_Count_Storage_Invariants(value.Literal_Count, namespace)
	Literal_Counts_Invariants(value.Literal_Counts, namespace)
	Literal_Offsets_Invariants(value.Literal_Offsets, namespace)
	Literal_Decoded_Invariants(value.Literal_Decoded, namespace)
	Name_Left_Storage_Invariants(value.Name_Left, namespace)
	Name_Right_Storage_Invariants(value.Name_Right, namespace)
	Number_Storage_Invariants(value.Number, namespace)
	Generated_Storage_Invariants(value.Generated, namespace)
	Workspace_Generated_Count_Storage_Invariants(value.Generated_Count, namespace)
	Float_Parse_Workspace_Invariants(value.Float_Parse, namespace)
	Float_Text_Workspace_Invariants(value.Float_Text, namespace)
	Frames_Invariants(value.Frames, namespace)
	Workspace_Frame_Count_Storage_Invariants(value.Frame_Count, namespace)
	Variables_Invariants(value.Variables, namespace)
	Workspace_Variable_Count_Storage_Invariants(value.Variable_Count, namespace)
	Workspace_Scope_Base_Storage_Invariants(value.Scope_Base, namespace)
	Arguments_Invariants(value.Arguments, namespace)
	Validation_Frames_Invariants(value.Validation, namespace)
	Evaluations_Invariants(value.Evaluations, namespace)
	Value_Format_Frames_Invariants(value.Format, namespace)
	Workspace_Diagnostic_Storage_Invariants(value.Diagnostic, namespace)
}

// Workspace_Pointer keeps optional execution storage typed.
type Workspace_Pointer *Workspace

// Workspace_Pointer_Invariants composes present caller storage.
func Workspace_Pointer_Invariants(value Workspace_Pointer, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Workspace_Invariants(*value, namespace)
}

// Workspace_Storage admits absent caller execution state.
type Workspace_Storage struct {
	// Value keeps absence explicit inside hostile input.
	Value Workspace_Unvalidated_Pointer
}

// Workspace_Storage_Invariants fixes pointer input shape.
func Workspace_Storage_Invariants(value Workspace_Storage, namespace aver.Namespace) {
	Workspace_Unvalidated_Pointer_Invariants(value.Value, namespace)
}

// Workspace_Unvalidated_Value retains hostile caller storage.
type Workspace_Unvalidated_Value any

// Workspace_Unvalidated_Value_Invariants defers shape trust.
func Workspace_Unvalidated_Value_Invariants(
	value Workspace_Unvalidated_Value, _ aver.Namespace,
) {
	present := value != nil
	aver.Always(present == present, "Execution workspace input retains caller storage.")
}

// Workspace_Unvalidated_Pointer admits absent or malformed caller storage.
type Workspace_Unvalidated_Pointer struct {
	// Value delays pointer-shape trust until workspace_valid.
	Value Workspace_Unvalidated_Value
}

// Workspace_Unvalidated_Pointer_Invariants defers shape checks to workspace_valid.
func Workspace_Unvalidated_Pointer_Invariants(
	value Workspace_Unvalidated_Pointer, namespace aver.Namespace,
) {
	Workspace_Unvalidated_Value_Invariants(value.Value, namespace)
}

// Workspace_Input carries hostile optional execution state.
type Workspace_Input struct {
	// State points at caller-owned execution storage when present.
	State Workspace_Storage
}

// Workspace_Input_Invariants fixes pointer input shape.
func Workspace_Input_Invariants(value Workspace_Input, namespace aver.Namespace) {
	Workspace_Storage_Invariants(value.State, namespace)
}

// Execute_Input bundles bounded execution dependencies.
type Execute_Input struct {
	// Destination receives complete output only on success.
	Destination Output
	// Program retains borrowed source and syntax.
	Program Program
	// Value is root dot.
	Value Value
	// Functions contains injected named callables.
	Functions Functions
	// Workspace owns execution mutation.
	Workspace Workspace_Input
}

// Execute_Input_Invariants composes public execution boundary.
func Execute_Input_Invariants(value Execute_Input, namespace aver.Namespace) {
	Output_Invariants(value.Destination, namespace)
	Program_Invariants(value.Program, namespace)
	Value_Invariants(value.Value, namespace)
	Functions_Invariants(value.Functions, namespace)
	Workspace_Input_Invariants(value.Workspace, namespace)
}

// Program_Validity reports safe syntax traversal.
type Program_Validity bool

// Program_Validity_Invariants covers parsed and forged programs.
func Program_Validity_Invariants(value Program_Validity, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Program syntax is safe to traverse.").
		Ensure()
}

// Node_Validity reports safe node access.
type Node_Validity bool

// Node_Validity_Invariants covers valid and forged node records.
func Node_Validity_Invariants(value Node_Validity, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Syntax node and references are valid.").
		Ensure()
}

// Value_Validity reports safe active payload.
type Value_Validity bool

// Value_Validity_Invariants covers valid and forged caller data.
func Value_Validity_Invariants(value Value_Validity, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Value active payload is valid.").
		Ensure()
}

// Number_Float_Syntax reports a real floating-point marker.
type Number_Float_Syntax bool

// Number_Float_Syntax_Invariants covers integer and floating syntax.
func Number_Float_Syntax_Invariants(
	value Number_Float_Syntax, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Number syntax contains a floating-point marker.").
		Ensure()
}

// Registry_Validity reports safe function lookup.
type Registry_Validity bool

// Registry_Validity_Invariants covers valid and malformed registries.
func Registry_Validity_Invariants(value Registry_Validity, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Function registry is valid.").
		Ensure()
}

// Workspace_Validity reports complete caller execution storage.
type Workspace_Validity bool

// Workspace_Validity_Invariants covers complete and malformed storage.
func Workspace_Validity_Invariants(
	value Workspace_Validity, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Execution workspace has complete bounded storage.").
		Ensure()
}

// STEP_COUNT_MINIMUM is empty execution work.
const STEP_COUNT_MINIMUM = ARGUMENT_COUNT_MINIMUM

// Step_Count bounds data-amplified execution.
type Step_Count uint16

// Step_Count_Invariants includes empty and refusal boundaries.
func Step_Count_Invariants(value Step_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(STEP_COUNT_MINIMUM), uint16(STEP_COUNT_MAXIMUM),
		).
		Ensure()
}

// ACTIVE_STEP_COUNT_INCREMENT charges one completed work item.
const ACTIVE_STEP_COUNT_INCREMENT Active_Step_Count = 1

// Active_Step_Count counts work after the mandatory root visit.
type Active_Step_Count uint16

// Active_Step_Count_Invariants excludes the pre-loop zero state from completed work.
func Active_Step_Count_Invariants(
	value Active_Step_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(ACTIVE_STEP_COUNT_MINIMUM),
			uint16(STEP_COUNT_MAXIMUM),
		).
		Ensure()
}

// ACTIVE_STEP_COUNT_MINIMUM charges the mandatory root frame or value.
const ACTIVE_STEP_COUNT_MINIMUM = Active_Step_Count(STEP_COUNT_MINIMUM) +
	ACTIVE_STEP_COUNT_INCREMENT

// Flow is normal traversal, break, or continue.
type Flow uint8

// Flow_Invariants covers standard range control.
func Flow_Invariants(value Flow, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(FLOW_NORMAL), uint8(FLOW_BREAK),
			uint8(FLOW_CONTINUE),
		).
		Ensure()
}

// FLOW_NORMAL keeps current traversal.
const FLOW_NORMAL Flow = Flow(bytes.SLICE_SIZE_MINIMUM)

// FLOW_INCREMENT adds one range-control result.
const FLOW_INCREMENT Flow = 1

// FLOW_BREAK exits nearest range.
const FLOW_BREAK Flow = FLOW_NORMAL + FLOW_INCREMENT

// FLOW_CONTINUE advances nearest range.
const FLOW_CONTINUE Flow = FLOW_BREAK + FLOW_INCREMENT

// Range_Flow is break or continue after normal traversal was excluded.
type Range_Flow uint8

// Range_Flow_Invariants lists both range-control states.
func Range_Flow_Invariants(value Range_Flow, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value), uint8(FLOW_BREAK), uint8(FLOW_CONTINUE)).
		Ensure()
}

// Result_Present distinguishes absent first pipeline input from nil.
type Result_Present bool

// Result_Present_Invariants covers first and later pipeline commands.
func Result_Present_Invariants(value Result_Present, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Pipeline has prior command result.").
		Ensure()
}

// Argument_Count is active callback argument storage.
type Argument_Count uint16

// Argument_Count_Invariants follows syntax-node argument maximum.
func Argument_Count_Invariants(value Argument_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(ARGUMENT_COUNT_MINIMUM),
			uint16(ARGUMENT_COUNT_MAXIMUM),
		).
		Ensure()
}

// ARGUMENT_COUNT_TRACKED_FIELD selects one count bounded by callback storage.
const ARGUMENT_COUNT_TRACKED_FIELD = bytes.SLICE_SIZE_MINIMUM

// ARGUMENT_COUNT_TRACKED_FIELD_COUNT fixes evaluation count transport shape.
const ARGUMENT_COUNT_TRACKED_FIELD_COUNT = ARGUMENT_COUNT_TRACKED_FIELD +
	STORAGE_FIELD_COUNT_INCREMENT

// Argument_Count_Tracked_Value retains callback-count proof storage.
type Argument_Count_Tracked_Value uint16

// Argument_Count_Tracked_Value_Invariants preserves the callback count.
func Argument_Count_Tracked_Value_Invariants(
	value Argument_Count_Tracked_Value, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(ARGUMENT_COUNT_MINIMUM),
			uint16(ARGUMENT_COUNT_MAXIMUM),
		).
		Ensure()
}

// Argument_Count_Tracked carries callback staging progress by value.
type Argument_Count_Tracked struct {
	// Value keeps staging proof attached to argument progress.
	Value Argument_Count_Proof_Stored
}

// Argument_Count_Tracked_Invariants checks transport ownership after storage validation.
func Argument_Count_Tracked_Invariants(
	value Argument_Count_Tracked, namespace aver.Namespace,
) {
	Argument_Count_Proof_Stored_Invariants(value.Value, namespace)
}

// Function_Found reports registry or builtin resolution.
type Function_Found bool

// Function_Found_Invariants covers known and missing identifiers.
func Function_Found_Invariants(value Function_Found, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Function name resolves.").
		Ensure()
}

// Truth reports standard template truth.
type Truth bool

// Truth_Invariants covers empty and nonempty values.
func Truth_Invariants(value Truth, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Template value is truthy.").
		Ensure()
}

// Equality reports standard comparable equality.
type Equality bool

// Equality_Invariants covers equal and unequal values.
func Equality_Invariants(value Equality, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Template values are equal.").
		Ensure()
}

// Bytes_Equality reports exact borrowed-byte equality.
type Bytes_Equality bool

// Bytes_Equality_Invariants covers matching and mismatching bytes.
func Bytes_Equality_Invariants(value Bytes_Equality, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Borrowed bytes are equal.").
		Ensure()
}

// Order prevents comparison helpers from returning values outside three-way ordering.
type Order int8

// Order_Invariants admits only the formula-derived three-way results.
func Order_Invariants(value Order, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Int8(
			int8(value), int8(ORDER_BEFORE), int8(ORDER_SAME), int8(ORDER_AFTER),
		).
		Ensure()
}

// Comparable keeps kind compatibility separate from the three-way result.
type Comparable bool

// Comparable_Invariants requires both compatible and incompatible value pairs.
func Comparable_Invariants(value Comparable, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Value kinds admit ordering.").
		Ensure()
}

// Na_N keeps binary64 classification distinct from ordering compatibility.
type Na_N bool

// Na_N_Invariants requires ordinary and not-a-number binary64 encodings.
func Na_N_Invariants(value Na_N, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Binary64 encoding is not a number.").
		Ensure()
}

// Collection_Limit prevents index conversion from exceeding bounded caller storage.
type Collection_Limit int

// Collection_Limit_Invariants follows the shared bounded slice formula.
func Collection_Limit_Invariants(
	value Collection_Limit, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), bytes.SLICE_SIZE_MINIMUM, bytes.SLICE_SIZE_MAXIMUM,
		).
		Ensure()
}

// Collection_Position remains inside the same caller-owned collection bound.
type Collection_Position int

// Collection_Position_Invariants follows the shared bounded slice formula.
func Collection_Position_Invariants(
	value Collection_Position, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), bytes.SLICE_SIZE_MINIMUM, bytes.SLICE_SIZE_MAXIMUM,
		).
		Ensure()
}

// Collection_Position_Validity separates refused scalar conversion from position zero.
type Collection_Position_Validity bool

// Collection_Position_Validity_Invariants requires accepted and refused indexes.
func Collection_Position_Validity_Invariants(
	value Collection_Position_Validity, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Collection position is valid.").
		Ensure()
}

// GENERATED_TEXT_SIZE_MINIMUM follows shortest static formatter fragment.
const GENERATED_TEXT_SIZE_MINIMUM = len(" ")

// GENERATED_TEXT_SIZE_MAXIMUM follows longest static formatter fragment.
const GENERATED_TEXT_SIZE_MAXIMUM = len("template.Function")

// Generated_Text avoids allocating byte conversions for static formatter text.
type Generated_Text string

// Generated_Text_Invariants follows exact internal formatter fragments.
func Generated_Text_Invariants(value Generated_Text, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), GENERATED_TEXT_SIZE_MINIMUM, GENERATED_TEXT_SIZE_MAXIMUM,
		).
		Ensure()
}

// OUTPUT_TEXT_SIZE_MINIMUM follows shortest structural output fragment.
const OUTPUT_TEXT_SIZE_MINIMUM = len("[")

// OUTPUT_TEXT_SIZE_MAXIMUM follows longest scalar output fragment.
const OUTPUT_TEXT_SIZE_MAXIMUM = len("<no value>")

// Output_Text is static formatting text bounded by caller output storage.
type Output_Text string

// Output_Text_Invariants follows exact scalar and structural fragments.
func Output_Text_Invariants(value Output_Text, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), OUTPUT_TEXT_SIZE_MINIMUM, OUTPUT_TEXT_SIZE_MAXIMUM,
		).
		Ensure()
}

// FORMATTED_INTEGER_TEXT_FIELD selects one completed integer conversion.
const FORMATTED_INTEGER_TEXT_FIELD = bytes.SLICE_SIZE_MINIMUM

// FORMATTED_INTEGER_TEXT_SIZE_MINIMUM is the shortest completed conversion.
const FORMATTED_INTEGER_TEXT_SIZE_MINIMUM = len("0")

// FORMATTED_INTEGER_TEXT_FIELD_COUNT fixes conversion transport shape.
const FORMATTED_INTEGER_TEXT_FIELD_COUNT = FORMATTED_INTEGER_TEXT_FIELD +
	STORAGE_FIELD_COUNT_INCREMENT

// Formatted_Integer_Text_Validated_Value retains completed conversion storage.
type Formatted_Integer_Text_Validated_Value []byte

// Formatted_Integer_Text_Validated_Value_Invariants preserves conversion proof.
func Formatted_Integer_Text_Validated_Value_Invariants(
	value Formatted_Integer_Text_Validated_Value, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), FORMATTED_INTEGER_TEXT_SIZE_MINIMUM, NUMBER_SIZE_MAXIMUM,
		).
		Ensure()
}

// Formatted_Integer_Text_Validated carries one completed integer conversion.
type Formatted_Integer_Text_Validated struct {
	// Value keeps conversion proof attached to output text.
	Value Formatted_Integer_Text_Proof_Stored
}

// Formatted_Integer_Text_Validated_Invariants checks conversion ownership.
func Formatted_Integer_Text_Validated_Invariants(
	value Formatted_Integer_Text_Validated, namespace aver.Namespace,
) {
	Formatted_Integer_Text_Proof_Stored_Invariants(value.Value, namespace)
}

// GENERATED_START_VALIDATED_FIELD selects one captured arena boundary.
const GENERATED_START_VALIDATED_FIELD = bytes.SLICE_SIZE_MINIMUM

// GENERATED_START_VALIDATED_FIELD_COUNT fixes boundary transport shape.
const GENERATED_START_VALIDATED_FIELD_COUNT = GENERATED_START_VALIDATED_FIELD +
	STORAGE_FIELD_COUNT_INCREMENT

// Generated_Start_Validated_Value retains generated-boundary proof storage.
type Generated_Start_Validated_Value uint16

// Generated_Start_Validated_Value_Invariants preserves the generated boundary.
func Generated_Start_Validated_Value_Invariants(
	value Generated_Start_Validated_Value, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(bytes.SLICE_SIZE_MINIMUM),
			uint16(GENERATED_SIZE_MAXIMUM),
		).
		Ensure()
}

// Generated_Start_Validated carries one workspace-proven arena boundary.
type Generated_Start_Validated struct {
	// Value keeps workspace proof attached to the arena boundary.
	Value Generated_Start_Proof_Stored
}

// Generated_Start_Validated_Invariants checks captured-boundary ownership.
func Generated_Start_Validated_Invariants(
	value Generated_Start_Validated, namespace aver.Namespace,
) {
	Generated_Start_Proof_Stored_Invariants(value.Value, namespace)
}

// Pipeline_State retains result and declarations without tuple widening.
type Pipeline_State struct {
	// Value is final command result.
	Value Value
	// Declarations retain variable names.
	Declarations Declaration_References
	// Declaration_Count selects populated names.
	Declaration_Count Declaration_Count
	// Assignment selects update instead of declaration.
	Assignment Assignment
}

// Pipeline_State_Invariants composes scalar evaluation metadata.
func Pipeline_State_Invariants(value Pipeline_State, namespace aver.Namespace) {
	Value_Invariants(value.Value, namespace)
	Declaration_References_Invariants(value.Declarations, namespace)
	Declaration_Count_Invariants(value.Declaration_Count, namespace)
	Assignment_Invariants(value.Assignment, namespace)
}

// Execute_Into stages and atomically commits one bounded execution.
func Execute_Into(input Execute_Input) (_ Output_Count, _ Diagnostic, status Status) {
	defer func() { Status_Invariants(status, "Execute_Into.status") }()
	Execute_Input_Invariants(input, "Execute_Into.input")
	count := Output_Count(OUTPUT_SIZE_MINIMUM)
	defer func() { Output_Count_Invariants(count, "Execute_Into.count") }()
	var diagnostic Diagnostic
	defer func() { Diagnostic_Invariants(diagnostic, "Execute_Into.diagnostic") }()
	workspace_input := input.Workspace.State.Value
	if workspace_input.Value == nil {
		diagnostic.Code = STATUS_WORKSPACE_INVALID
		return count, diagnostic, STATUS_WORKSPACE_INVALID
	}
	if !bool(workspace_valid(input.Workspace)) {
		diagnostic.Code = STATUS_WORKSPACE_INVALID
		return count, diagnostic, STATUS_WORKSPACE_INVALID
	}
	workspace := workspace_begin(workspace_input)
	failure := Failure_Status_Validated{}
	refuse := func() { execution_refusal(workspace, failure) }
	if !bool(program_valid(input.Program)) {
		status = STATUS_PROGRAM_INVALID
	}
	if status == STATUS_OK {
		value_status := value_graph_validate(workspace, Value_Unvalidated(input.Value))
		if value_status != VALUE_GRAPH_STATUS_OK {
			status = Status(value_status)
		}
	}
	if status == STATUS_OK {
		if !bool(registry_valid(input.Functions)) {
			status = STATUS_FUNCTION_INVALID
		}
	}
	if status != STATUS_OK {
		failure.Value = Failure_Status_Validated_Value(status)
		refuse()
		diagnostic = Diagnostic(workspace.Diagnostic.Value)
		return count, diagnostic, status
	}
	for index := NODE_COUNT_MINIMUM; index < int(input.Program.Document.Node_Count); index++ {
		workspace.Literal_Decoded[index] = false
	}
	document_input := input.Program.Document
	document := Program_Validated_Document{
		Root:           Program_Validated_Root(document_input.Root),
		Node_Count:     Program_Validated_Node_Count(document_input.Node_Count),
		First_Template: Program_Validated_First_Template(document_input.First_Template),
		Template_Count: Program_Validated_Template_Count(document_input.Template_Count),
	}
	program := program_validate(input.Program.Source, document, input.Program.Syntax)
	functions := Functions_Validated{Value: Functions_Validated_Value(input.Functions)}
	written_state, execution_status, _ := execute_unchecked(
		workspace, program, input.Value, functions,
	)
	if execution_status != NODE_EXECUTION_STATUS_OK {
		diagnostic = Diagnostic(workspace.Diagnostic.Value)
		return count, diagnostic, Status(execution_status)
	}
	written := written_state.Value.(Output_Count_Staged_Value)
	if len(input.Destination) < int(written) {
		status = STATUS_OUTPUT_TOO_SMALL
		failure.Value = Failure_Status_Validated_Value(status)
		refuse()
		diagnostic = Diagnostic(workspace.Diagnostic.Value)
		return count, diagnostic, status
	}
	copy(input.Destination, workspace.Output[:written])
	count = Output_Count(written)
	return count, diagnostic, STATUS_OK
}

func workspace_begin(input Workspace_Unvalidated_Pointer) (workspace Workspace_Pointer) {
	defer func() {
		Workspace_Pointer_Invariants(workspace, "workspace_begin.workspace")
	}()
	Workspace_Unvalidated_Pointer_Invariants(input, "workspace_begin.input")
	workspace_value, _ := input.Value.(*Workspace)
	workspace = Workspace_Pointer(workspace_value)
	workspace.Frame_Count.Value = Workspace_Frame_Count_Storage_Value(bytes.SLICE_SIZE_MINIMUM)
	workspace.Variable_Count.Value = Workspace_Variable_Count_Storage_Value(
		VARIABLE_COUNT_MINIMUM,
	)
	workspace.Scope_Base.Value = Workspace_Scope_Base_Storage_Value(VARIABLE_COUNT_MINIMUM)
	workspace.Literal_Count.Value = Workspace_Literal_Count_Storage_Value(SOURCE_SIZE_MINIMUM)
	workspace.Generated_Count.Value = Workspace_Generated_Count_Storage_Value(
		OUTPUT_SIZE_MINIMUM,
	)
	workspace.Diagnostic.Value = Diagnostic_Stored(Diagnostic{})
	return workspace
}

func program_validate(
	source Source, document Program_Validated_Document, syntax Syntax_Storage,
) (validated Program_Validated) {
	defer func() {
		Program_Validated_Invariants(validated, "program_validate.validated")
	}()
	Source_Invariants(source, "program_validate.source")
	Program_Validated_Document_Invariants(document, "program_validate.document")
	Syntax_Storage_Invariants(syntax, "program_validate.syntax")
	return Program_Validated{
		Source: source, Document: document, Syntax: syntax,
	}
}

func workspace_valid(
	input Workspace_Input,
) (valid Workspace_Validity) {
	defer func() {
		Workspace_Validity_Invariants(valid, "workspace_valid.valid")
	}()
	Workspace_Input_Invariants(input, "workspace_valid.input")
	workspace, typed := input.State.Value.Value.(*Workspace)
	if !typed {
		return false
	}
	if workspace == nil {
		return false
	}
	if len(workspace.Output) != OUTPUT_SIZE_MAXIMUM {
		return false
	}
	if len(workspace.Literals) != QUOTED_DECODED_SIZE_MAXIMUM {
		return false
	}
	if len(workspace.Literal_Counts) != NODE_COUNT_MAXIMUM {
		return false
	}
	if len(workspace.Literal_Offsets) != NODE_COUNT_MAXIMUM {
		return false
	}
	if len(workspace.Literal_Decoded) != NODE_COUNT_MAXIMUM {
		return false
	}
	if len(workspace.Name_Left) != QUOTED_DECODED_SIZE_MAXIMUM {
		return false
	}
	if len(workspace.Name_Right) != QUOTED_DECODED_SIZE_MAXIMUM {
		return false
	}
	if len(workspace.Number) != NUMBER_SIZE_MAXIMUM {
		return false
	}
	if len(workspace.Generated) != GENERATED_SIZE_MAXIMUM {
		return false
	}
	if len(workspace.Frames) != FRAME_COUNT_MAXIMUM {
		return false
	}
	if len(workspace.Variables) != VARIABLE_COUNT_MAXIMUM {
		return false
	}
	if len(workspace.Arguments) != ARGUMENT_COUNT_MAXIMUM {
		return false
	}
	if len(workspace.Validation) != FRAME_COUNT_MAXIMUM {
		return false
	}
	if len(workspace.Evaluations) != FRAME_COUNT_MAXIMUM {
		return false
	}
	if len(workspace.Format) != FRAME_COUNT_MAXIMUM {
		return false
	}
	return true
}

func execution_refusal(
	workspace_state Workspace_Pointer, status_value Failure_Status_Validated,
) {
	Workspace_Pointer_Invariants(workspace_state, "execution_refusal.workspace_state")
	Failure_Status_Validated_Invariants(status_value, "execution_refusal.status_value")
	workspace := (Workspace_Pointer)(workspace_state)
	status := Status(status_value.Value.(Failure_Status_Validated_Value))
	set_diagnostic(
		workspace, Failure_Status_Validated{Value: Failure_Status_Validated_Value(status)},
		Diagnostic_Position(SOURCE_SIZE_MINIMUM),
	)
}

func set_diagnostic(
	workspace_state Workspace_Pointer, status_value Failure_Status_Validated,
	position_value Diagnostic_Position,
) {
	Workspace_Pointer_Invariants(workspace_state, "set_diagnostic.workspace_state")
	Failure_Status_Validated_Invariants(status_value, "set_diagnostic.status_value")
	Diagnostic_Position_Invariants(position_value, "set_diagnostic.position_value")
	workspace := (Workspace_Pointer)(workspace_state)
	status := Status(status_value.Value.(Failure_Status_Validated_Value))
	position := Diagnostic_Position(position_value)
	if workspace.Diagnostic.Value.Code != STATUS_OK {
		return
	}
	workspace.Diagnostic.Value = Diagnostic_Stored(Diagnostic{
		Code: status, Position: position,
	})
}

func program_valid(program_value Program) (
	valid Program_Validity,
) {
	defer func() { Program_Validity_Invariants(valid, "program_valid.valid") }()
	Program_Invariants(program_value, "program_valid.program_value")
	program := Program(program_value)
	syntax := program.Syntax.Value
	if syntax == nil {
		return false
	}
	data := program.Source.Data.Value
	if len(data) > SOURCE_SIZE_MAXIMUM {
		return false
	}
	document := program.Document
	if document.Root != Document_Root(ROOT_NODE_COUNT) {
		return false
	}
	if document.Node_Count < Node_Count(ROOT_NODE_COUNT) {
		return false
	}
	if document.Node_Count > Node_Count(NODE_COUNT_MAXIMUM) {
		return false
	}
	if document.Template_Count > Template_Count(document.Node_Count) {
		return false
	}
	if (document.Template_Count == Template_Count(TEMPLATE_COUNT_MINIMUM)) !=
		(document.First_Template == Document_First_Template(NO_NODE)) {
		return false
	}
	active_count := Active_Node_Count(document.Node_Count)
	for index := Node_Count(NODE_COUNT_MINIMUM); index < document.Node_Count; index++ {
		if !bool(node_record_valid(
			syntax.Nodes[index], Source_Position(len(data)), active_count,
		)) {
			return false
		}
	}
	root := syntax.Nodes[ROOT_NODE_COUNT-NODE_COUNT_INCREMENT]
	return Program_Validity(root.Kind == NODE_LIST)
}

func node_record_valid(
	node_value Node,
	source_size_value Source_Position,
	node_count_value Active_Node_Count,
) (valid Node_Validity) {
	defer func() { Node_Validity_Invariants(valid, "node_record_valid.valid") }()
	Node_Invariants(node_value, "node_record_valid.node_value")
	Source_Position_Invariants(source_size_value, "node_record_valid.source_size_value")
	Active_Node_Count_Invariants(node_count_value, "node_record_valid.node_count_value")
	node := Node(node_value)
	source_size := int(source_size_value)
	node_count := Active_Node_Count(node_count_value)
	if node.Kind < NODE_KIND_MINIMUM {
		return false
	}
	if node.Kind > NODE_KIND_MAXIMUM {
		return false
	}
	if int(node.Start) > source_size {
		return false
	}
	if int(node.End) > source_size {
		return false
	}
	if node.Start > Node_Start(node.End) {
		return false
	}
	if int(node.Value_Start) > source_size {
		return false
	}
	if int(node.Value_End) > source_size {
		return false
	}
	if node.Value_Start > Node_Value_Start(node.Value_End) {
		return false
	}
	if Node_Reference(node.First_Child) > Node_Reference(node_count) {
		return false
	}
	if Node_Reference(node.Next_Sibling) > Node_Reference(node_count) {
		return false
	}
	return true
}

func program_node(
	program_value Program_Validated,
	reference_value Node_Link_Validated,
) (_ Node_Validated, valid Node_Validity) {
	defer func() { Node_Validity_Invariants(valid, "program_node.valid") }()
	Program_Validated_Invariants(program_value, "program_node.program_value")
	Node_Link_Validated_Invariants(reference_value, "program_node.reference_value")
	var node Node_Validated
	defer func() { Node_Validated_Invariants(node, "program_node.node") }()
	program := program_value
	reference := reference_value.Value.(Node_Link_Validated_Value)
	if Node_Reference(reference) == NO_NODE {
		return Node_Validated{}, false
	}
	if Node_Reference(reference) > Node_Reference(program.Document.Node_Count) {
		return Node_Validated{}, false
	}
	node = Node_Validated{Value: Node_Stored(
		&program.Syntax.Value.Nodes[reference-NODE_COUNT_INCREMENT],
	)}

	node_record := node.Value.(*Node)
	if !bool(node_record_valid(
		*node_record, Source_Position(len(program.Source.Data.Value)),
		Active_Node_Count(program.Document.Node_Count),
	)) {
		return Node_Validated{}, false
	}
	return node, true
}

func value_graph_validate(
	workspace_state Workspace_Pointer, root_value Value_Unvalidated,
) (status Value_Graph_Status) {
	defer func() {
		Value_Graph_Status_Invariants(status, "value_graph_validate.status")
	}()
	Workspace_Pointer_Invariants(workspace_state, "value_graph_validate.workspace_state")
	Value_Unvalidated_Invariants(root_value, "value_graph_validate.root_value")
	workspace, root := Workspace_Pointer(workspace_state), Value(root_value)
	workspace.Validation[bytes.SLICE_SIZE_MINIMUM] = root
	count, steps := VALUE_COUNT_INCREMENT, Active_Step_Count(STEP_COUNT_MINIMUM)
	defer func() {
		Active_Step_Count_Invariants(steps, "value_graph_validate.steps")
	}()
	for count > NODE_COUNT_MINIMUM {
		if steps == Active_Step_Count(STEP_COUNT_MAXIMUM) {
			return VALUE_GRAPH_STATUS_LIMIT_EXCEEDED
		}
		steps++
		count--
		current := workspace.Validation[count]
		if !bool(value_active_valid(Value_Unvalidated(current))) {
			return VALUE_GRAPH_STATUS_INVALID
		}
		switch Value_Kind(current.Kind.Value) {
		case VALUE_SEQUENCE:
			for index := range Values(current.Values.Value) {
				if count == len(workspace.Validation) {
					return VALUE_GRAPH_STATUS_LIMIT_EXCEEDED
				}
				workspace.Validation[count] = Values(current.Values.Value)[index]
				count++
			}
		case VALUE_OBJECT:
			fields := Fields(current.Fields.Value)
			if !bool(object_fields_valid(Bounded_Fields{
				Value: Fields_Storage_Value(fields),
			})) {
				return VALUE_GRAPH_STATUS_INVALID
			}
			for index := range fields {
				field := fields[index]
				if count == len(workspace.Validation) {
					return VALUE_GRAPH_STATUS_LIMIT_EXCEEDED
				}
				workspace.Validation[count] = field.Value.Data
				count++
			}
		case VALUE_MAP:
			fields := Fields(current.Fields.Value)
			if len(fields) > RANGE_MAP_ENTRY_COUNT_MAXIMUM {
				return VALUE_GRAPH_STATUS_LIMIT_EXCEEDED
			}
			if !bool(map_fields_valid(Bounded_Fields{
				Value: Fields_Storage_Value(fields),
			})) {
				return VALUE_GRAPH_STATUS_INVALID
			}
			for index := range fields {
				field := fields[index]
				if count > len(workspace.Validation)-MAP_ENTRY_GRAPH_VALUE_COUNT {
					return VALUE_GRAPH_STATUS_LIMIT_EXCEEDED
				}
				workspace.Validation[count] = field.Key.Data
				workspace.Validation[count+VALUE_PAYLOAD_FIELD_COUNT] =
					field.Value.Data
				count += MAP_ENTRY_GRAPH_VALUE_COUNT
			}
		}
	}
	return VALUE_GRAPH_STATUS_OK
}

func object_fields_valid(field_values Bounded_Fields) (
	valid Value_Validity,
) {
	defer func() { Value_Validity_Invariants(valid, "object_fields_valid.valid") }()
	Bounded_Fields_Invariants(field_values, "object_fields_valid.field_values")
	fields := field_values.Value.(Fields_Storage_Value)
	for index := range fields {
		field := fields[index]
		if len(field.Name) == SOURCE_SIZE_MINIMUM {
			return false
		}
		if len(field.Name) > SOURCE_SIZE_MAXIMUM {
			return false
		}
		previous_index := SOURCE_SIZE_MINIMUM
		for previous_index < index {
			if bool(bytes_equal(
				Text(fields[previous_index].Name), Text(field.Name),
			)) {
				return false
			}
			previous_index++
		}
	}
	return true
}

func map_fields_valid(field_values Bounded_Fields) (
	valid Value_Validity,
) {
	defer func() { Value_Validity_Invariants(valid, "map_fields_valid.valid") }()
	Bounded_Fields_Invariants(field_values, "map_fields_valid.field_values")
	fields := field_values.Value.(Fields_Storage_Value)
	for index := range fields {
		field := fields[index]
		if len(field.Name) != SOURCE_SIZE_MINIMUM {
			return false
		}
		if !bool(map_key_valid(field.Key.Data)) {
			return false
		}
	}
	if bool(map_keys_monotonic(Map_Keys_Validated{Value: Fields_Storage_Value(fields)})) {
		return true
	}
	for index := range fields {
		field := fields[index]
		previous_index := SOURCE_SIZE_MINIMUM
		for previous_index < index {
			if bool(values_equal(
				fields[previous_index].Key.Data, field.Key.Data,
			)) {
				return false
			}
			previous_index++
		}
	}
	return true
}

func map_keys_monotonic(
	field_values Map_Keys_Validated,
) (monotonic Map_Keys_Monotonic) {
	defer func() {
		Map_Keys_Monotonic_Invariants(monotonic, "map_keys_monotonic.monotonic")
	}()
	Map_Keys_Validated_Invariants(field_values, "map_keys_monotonic.field_values")
	fields := field_values.Value.(Fields_Storage_Value)
	direction := ORDER_SAME
	monotonic = true
	for index := MAP_ENTRY_COUNT_INCREMENT; index < len(fields); index++ {
		previous := fields[index-MAP_ENTRY_COUNT_INCREMENT].Key.Data
		current := fields[index].Key.Data
		if bool(values_equal(previous, current)) {
			return false
		}
		order := map_key_order(
			Map_Key_Validated{Value: previous}, Map_Key_Validated{Value: current},
		)
		if order == ORDER_SAME {
			monotonic = false
			continue
		}
		if direction == ORDER_SAME {
			direction = order
			continue
		}
		if order != direction {
			monotonic = false
		}
	}
	return monotonic
}

func value_active_valid(value_state Value_Unvalidated) (
	valid Value_Validity,
) {
	defer func() { Value_Validity_Invariants(valid, "value_active_valid.valid") }()
	Value_Unvalidated_Invariants(value_state, "value_active_valid.value_state")
	value := Value(value_state)
	kind := Value_Kind(value.Kind.Value)
	if kind < VALUE_NIL {
		return false
	}
	if kind > VALUE_FUNCTION {
		return false
	}
	Value_Kind_Invariants(kind, "value_active_valid.kind")
	switch kind {
	case VALUE_TEXT:
		return Value_Validity(
			len(Text(value.Text.Value)) <= OUTPUT_SIZE_MAXIMUM,
		)
	case VALUE_SEQUENCE:
		return Value_Validity(len(Values(value.Values.Value)) <= VALUE_COUNT_MAXIMUM)
	case VALUE_OBJECT, VALUE_MAP:
		return Value_Validity(len(Fields(value.Fields.Value)) <= VALUE_COUNT_MAXIMUM)
	case VALUE_FUNCTION:
		return Value_Validity(Function_Call(value.Function.Value) != nil)
	}
	return true
}

func map_key_valid(value_state Value) (
	valid Value_Validity,
) {
	defer func() { Value_Validity_Invariants(valid, "map_key_valid.valid") }()
	Value_Invariants(value_state, "map_key_valid.value_state")
	value := Value(value_state)
	kind := Value_Kind(value.Kind.Value)
	if kind < VALUE_NIL {
		return false
	}
	return Value_Validity(kind <= VALUE_TEXT)
}

func registry_valid(function_values Functions) (
	valid Registry_Validity,
) {
	defer func() { Registry_Validity_Invariants(valid, "registry_valid.valid") }()
	Functions_Invariants(function_values, "registry_valid.function_values")
	functions := Functions(function_values)
	if len(functions) > FUNCTION_COUNT_MAXIMUM {
		return false
	}
	for index := range functions {
		current := functions[index]
		if len(current.Name) == SOURCE_SIZE_MINIMUM {
			return false
		}
		if len(current.Name) > SOURCE_SIZE_MAXIMUM {
			return false
		}
		if current.Call == nil {
			return false
		}
		previous_index := SOURCE_SIZE_MINIMUM
		for previous_index < index {
			if bytes_equal(
				Text(functions[previous_index].Name), Text(current.Name),
			) {
				return false
			}
			previous_index++
		}
	}
	return true
}

func execute_unchecked(
	workspace_state Workspace_Pointer, program_value Program_Validated,
	root_value Value,
	function_values Functions_Validated,
) (
	_ Output_Count_Staged,
	status Node_Execution_Status,
	_ Active_Step_Count,
) {
	defer func() { Node_Execution_Status_Invariants(status, "execute_unchecked.status") }()
	Workspace_Pointer_Invariants(workspace_state, "execute_unchecked.workspace_state")
	Program_Validated_Invariants(program_value, "execute_unchecked.program_value")
	Value_Invariants(root_value, "execute_unchecked.root_value")
	Functions_Validated_Invariants(function_values, "execute_unchecked.function_values")
	count := Output_Count_Staged{Value: Output_Count_Staged_Value(OUTPUT_SIZE_MINIMUM)}
	defer func() { Output_Count_Staged_Invariants(count, "execute_unchecked.count") }()
	steps := Active_Step_Count(STEP_COUNT_MINIMUM)
	defer func() { Active_Step_Count_Invariants(steps, "execute_unchecked.steps") }()
	workspace, root := (Workspace_Pointer)(workspace_state), Value(root_value)
	execute_begin(workspace, program_value, root)
	limit_failure := Failure_Status_Validated{
		Value: Failure_Status_Validated_Value(STATUS_LIMIT_EXCEEDED)}
	program_failure := Failure_Status_Validated{
		Value: Failure_Status_Validated_Value(STATUS_PROGRAM_INVALID)}
	initial_position := Execution_Failure_Position{
		Value: Execution_Failure_Position_Value(SOURCE_SIZE_MINIMUM)}
	for workspace.Frame_Count.Value.(Workspace_Frame_Count_Storage_Value) >
		Workspace_Frame_Count_Storage_Value(NODE_COUNT_MINIMUM) {
		if steps == Active_Step_Count(STEP_COUNT_MAXIMUM) {
			execution_failure_at(workspace, limit_failure, initial_position)
			count = Output_Count_Staged{}
			return count, NODE_EXECUTION_STATUS_LIMIT_EXCEEDED, steps
		}
		steps++
		frame_count := workspace.Frame_Count.Value.(Workspace_Frame_Count_Storage_Value)
		frame_index := int(frame_count) - FRAME_COUNT_INCREMENT
		frame_state := &workspace.Frames[frame_index]
		frame := &frame_state.Value
		if frame.Kind == FRAME_RANGE {
			range_schedule(workspace, program_value, frame_state)
			continue
		}
		if frame.Reference == NO_NODE {
			frame_pop(workspace)
			continue
		}
		reference := frame.Reference
		node_value, found := program_node(
			program_value,
			Node_Link_Validated{Value: Node_Link_Validated_Value(reference)},
		)
		if !bool(found) {
			execution_failure_at(workspace, program_failure, initial_position)
			count = Output_Count_Staged{}
			return count, NODE_EXECUTION_STATUS_PROGRAM_INVALID, steps
		}
		node := node_value.Value.(*Node)
		frame.Reference = Node_Reference(node.Next_Sibling)
		flow := FLOW_NORMAL
		count, flow, status = execute_node(
			workspace, program_value, node_value, frame.Dot, function_values, count,
		)
		if status != NODE_EXECUTION_STATUS_OK {
			count = Output_Count_Staged{}
			return count, status, steps
		}
		if flow != FLOW_NORMAL {
			if !bool(range_unwind(workspace, Range_Flow(flow))) {
				node_position := Execution_Failure_Position{
					Value: Execution_Failure_Position_Value(node.Start)}
				execution_failure_at(workspace, program_failure, node_position)
				count = Output_Count_Staged{}
				return count, NODE_EXECUTION_STATUS_PROGRAM_INVALID, steps
			}
		}
	}
	return count, NODE_EXECUTION_STATUS_OK, steps
}

func execution_failure_at(
	workspace_state Workspace_Pointer, status_value Failure_Status_Validated,
	position_value Execution_Failure_Position,
) {
	Workspace_Pointer_Invariants(workspace_state, "execution_failure_at.workspace_state")
	Failure_Status_Validated_Invariants(status_value, "execution_failure_at.status_value")
	Execution_Failure_Position_Invariants(
		position_value, "execution_failure_at.position_value",
	)
	execution_failure(workspace_state, status_value, position_value)
}

func execution_failure(
	workspace_state Workspace_Pointer, status_value Failure_Status_Validated,
	position_value Execution_Failure_Position,
) {
	Workspace_Pointer_Invariants(workspace_state, "execution_failure.workspace_state")
	Failure_Status_Validated_Invariants(status_value, "execution_failure.status_value")
	Execution_Failure_Position_Invariants(
		position_value, "execution_failure.position_value",
	)
	set_diagnostic(
		workspace_state, status_value,
		Diagnostic_Position(position_value.Value.(Execution_Failure_Position_Value)),
	)
}

func execute_begin(
	workspace_state Workspace_Pointer, program_value Program_Validated, root_value Value,
) {
	Workspace_Pointer_Invariants(workspace_state, "execute_begin.workspace_state")
	Program_Validated_Invariants(program_value, "execute_begin.program_value")
	Value_Invariants(root_value, "execute_begin.root_value")
	workspace := (Workspace_Pointer)(workspace_state)
	program := program_value
	root := Value(root_value)
	workspace.Variable_Count.Value = Workspace_Variable_Count_Storage_Value(
		DECLARATION_COUNT_ONE,
	)
	workspace.Scope_Base.Value = Workspace_Scope_Base_Storage_Value(VARIABLE_COUNT_MINIMUM)
	workspace.Variables[VARIABLE_COUNT_MINIMUM] = Variable{Root: true, Value: root}
	workspace.Frame_Count.Value = Workspace_Frame_Count_Storage_Value(FRAME_COUNT_INCREMENT)
	root_node := program.Syntax.Value.Nodes[program.Document.Root-NODE_COUNT_INCREMENT]
	workspace.Frames[bytes.SLICE_SIZE_MINIMUM].Value = Frame_Stored(Frame{
		Kind: FRAME_LIST, Reference: Node_Reference(root_node.First_Child),
		Dot: root,
	})
}

func frame_pop(workspace_state Workspace_Pointer) {
	Workspace_Pointer_Invariants(workspace_state, "frame_pop.workspace_state")
	workspace := (Workspace_Pointer)(workspace_state)
	index := int(workspace.Frame_Count.Value.(Workspace_Frame_Count_Storage_Value)) -
		FRAME_COUNT_INCREMENT
	frame := workspace.Frames[index].Value
	if bool(frame.Restore) {
		workspace.Variable_Count.Value = Workspace_Variable_Count_Storage_Value(
			frame.Variable_Mark,
		)
		workspace.Scope_Base.Value = Workspace_Scope_Base_Storage_Value(
			frame.Previous_Scope_Base,
		)
	}
	workspace.Frame_Count.Value =
		workspace.Frame_Count.Value.(Workspace_Frame_Count_Storage_Value) -
			Workspace_Frame_Count_Storage_Value(FRAME_COUNT_INCREMENT)
}

func frame_push_list(
	workspace_state Workspace_Pointer, reference_value Node_Link_Validated,
	dot_value Value,
	restore_value Scope_Restore_Validated,
) {
	Workspace_Pointer_Invariants(workspace_state, "frame_push_list.workspace_state")
	Node_Link_Validated_Invariants(reference_value, "frame_push_list.reference_value")
	Value_Invariants(dot_value, "frame_push_list.dot_value")
	Scope_Restore_Validated_Invariants(restore_value, "frame_push_list.restore_value")
	workspace := (Workspace_Pointer)(workspace_state)
	reference := reference_value.Value.(Node_Link_Validated_Value)
	dot := Value(dot_value)
	restore := restore_value.Value
	frame_count := workspace.Frame_Count.Value.(Workspace_Frame_Count_Storage_Value)
	aver.Always(
		int(frame_count) < len(workspace.Frames),
		"Source and variable bounds exhaust before caller frame storage.",
	)
	workspace.Frames[frame_count].Value = Frame_Stored(Frame{
		Kind: FRAME_LIST, Reference: Node_Reference(reference), Dot: dot,
		Restore: true, Variable_Mark: Frame_Variable_Mark(restore.Mark),
		Previous_Scope_Base: Frame_Previous_Scope_Base(restore.Previous),
	})
	workspace.Frame_Count.Value = frame_count +
		Workspace_Frame_Count_Storage_Value(FRAME_COUNT_INCREMENT)
}

func execute_node(
	workspace_state Workspace_Pointer, program_value Program_Validated,
	node_value Node_Validated, dot_value Value,
	function_values Functions_Validated, count_value Output_Count_Staged,
) (_ Output_Count_Staged, _ Flow, status Node_Execution_Status) {
	defer func() { Node_Execution_Status_Invariants(status, "execute_node.status") }()
	Workspace_Pointer_Invariants(workspace_state, "execute_node.workspace_state")
	Program_Validated_Invariants(program_value, "execute_node.program_value")
	Node_Validated_Invariants(node_value, "execute_node.node_value")
	Value_Invariants(dot_value, "execute_node.dot_value")
	Functions_Validated_Invariants(function_values, "execute_node.function_values")
	Output_Count_Staged_Invariants(count_value, "execute_node.count_value")
	written, flow := count_value, Flow(FLOW_NORMAL)
	defer func() { Output_Count_Staged_Invariants(written, "execute_node.written") }()
	defer func() { Flow_Invariants(flow, "execute_node.flow") }()
	workspace := (Workspace_Pointer)(workspace_state)
	node, dot := node_value.Value.(*Node), Value(dot_value)
	switch node.Kind {
	case NODE_TEXT:
		text := program_value.Source.Data.Value[node.Value_Start:node.Value_End]
		var output_status Output_Status
		written, output_status = output_bytes(workspace, written, Text(text))
		status = Node_Execution_Status(output_status)
		return written, flow, status
	case NODE_COMMENT:
		return written, flow, NODE_EXECUTION_STATUS_OK
	case NODE_ACTION:
		written, status = execute_action(
			workspace, program_value, node_value, dot, function_values, written,
		)
		return written, flow, status
	case NODE_IF, NODE_WITH, NODE_RANGE:
		control_status := execute_control(
			workspace, program_value, node_value, dot, function_values,
		)
		if control_status != EVALUATION_STATUS_OK {
			failure := Failure_Status_Validated{
				Value: Failure_Status_Validated_Value(control_status),
			}
			execution_node_error(workspace, failure, node_value)
			written = Output_Count_Staged{}
			return written, flow, Node_Execution_Status(control_status)
		}
		return written, flow, NODE_EXECUTION_STATUS_OK
	case NODE_TEMPLATE:
		template_status := execute_template(
			workspace, program_value, node_value, dot, function_values,
		)
		if template_status != EVALUATION_STATUS_OK {
			failure := Failure_Status_Validated{
				Value: Failure_Status_Validated_Value(template_status),
			}
			execution_node_error(workspace, failure, node_value)
			written = Output_Count_Staged{}
			return written, flow, Node_Execution_Status(template_status)
		}
		return written, flow, NODE_EXECUTION_STATUS_OK
	case NODE_BREAK:
		flow = FLOW_BREAK
		return written, flow, NODE_EXECUTION_STATUS_OK
	case NODE_CONTINUE:
		flow = FLOW_CONTINUE
		return written, flow, NODE_EXECUTION_STATUS_OK
	}
	execution_node_error(
		workspace, Failure_Status_Validated{
			Value: Failure_Status_Validated_Value(STATUS_PROGRAM_INVALID),
		}, node_value,
	)
	written = Output_Count_Staged{}
	return written, flow, NODE_EXECUTION_STATUS_PROGRAM_INVALID
}

func execute_action(
	workspace_state Workspace_Pointer, program_value Program_Validated,
	node_value Node_Validated,
	dot_value Value,
	function_values Functions_Validated,
	count_value Output_Count_Staged,
) (_ Output_Count_Staged, status Node_Execution_Status) {
	defer func() { Node_Execution_Status_Invariants(status, "execute_action.status") }()
	Workspace_Pointer_Invariants(workspace_state, "execute_action.workspace_state")
	Program_Validated_Invariants(program_value, "execute_action.program_value")
	Node_Validated_Invariants(node_value, "execute_action.node_value")
	Value_Invariants(dot_value, "execute_action.dot_value")
	Functions_Validated_Invariants(function_values, "execute_action.function_values")
	Output_Count_Staged_Invariants(count_value, "execute_action.count_value")
	count := count_value
	defer func() { Output_Count_Staged_Invariants(count, "execute_action.count") }()
	workspace := (Workspace_Pointer)(workspace_state)
	node, dot := node_value.Value.(*Node), Value(dot_value)
	functions := function_values
	pipe_value, found := program_node(
		program_value,
		Node_Link_Validated{Value: Node_Link_Validated_Value(node.First_Child)},
	)
	if !bool(found) {
		execution_node_error(
			workspace, Failure_Status_Validated{
				Value: Failure_Status_Validated_Value(STATUS_PROGRAM_INVALID),
			}, node_value,
		)
		count = Output_Count_Staged{}
		return count, NODE_EXECUTION_STATUS_PROGRAM_INVALID
	}
	pipe := pipe_value.Value.(*Node)
	if pipe.Kind != NODE_PIPE {
		execution_node_error(
			workspace, Failure_Status_Validated{
				Value: Failure_Status_Validated_Value(STATUS_PROGRAM_INVALID),
			}, node_value,
		)
		count = Output_Count_Staged{}
		return count, NODE_EXECUTION_STATUS_PROGRAM_INVALID
	}
	state, pipeline_status := pipeline_evaluate(
		workspace, program_value, pipe_value, dot, functions,
	)
	if pipeline_status != EVALUATION_STATUS_OK {
		execution_node_error(
			workspace, Failure_Status_Validated{
				Value: Failure_Status_Validated_Value(pipeline_status),
			}, node_value,
		)
		count = Output_Count_Staged{}
		return count, Node_Execution_Status(pipeline_status)
	}
	if state.Declaration_Count != DECLARATION_COUNT_NONE {
		return count, NODE_EXECUTION_STATUS_OK
	}
	count, output_status := output_value(workspace, count, state.Value)
	if output_status != OUTPUT_STATUS_OK {
		execution_node_error(
			workspace, Failure_Status_Validated{
				Value: Failure_Status_Validated_Value(output_status),
			}, node_value,
		)
		count = Output_Count_Staged{}
		return count, Node_Execution_Status(output_status)
	}
	return count, NODE_EXECUTION_STATUS_OK
}

func execution_node_error(
	workspace_state Workspace_Pointer, status_value Failure_Status_Validated,
	node_value Node_Validated,
) {
	Workspace_Pointer_Invariants(workspace_state, "execution_node_error.workspace_state")
	Failure_Status_Validated_Invariants(
		status_value, "execution_node_error.status_value",
	)
	Node_Validated_Invariants(node_value, "execution_node_error.node_value")
	workspace := (Workspace_Pointer)(workspace_state)
	status := status_value.Value.(Failure_Status_Validated_Value)
	node := node_value.Value.(*Node)
	set_diagnostic(
		workspace, Failure_Status_Validated{Value: Failure_Status_Validated_Value(status)},
		Diagnostic_Position(node.Start),
	)
}

func execute_control(
	workspace_state Workspace_Pointer, program_value Program_Validated,
	control_value Node_Validated,
	dot_value Value,
	function_values Functions_Validated,
) (status Evaluation_Status) {
	defer func() {
		Evaluation_Status_Invariants(status, "execute_control.status")
	}()
	Workspace_Pointer_Invariants(workspace_state, "execute_control.workspace_state")
	Program_Validated_Invariants(program_value, "execute_control.program_value")
	Node_Validated_Invariants(control_value, "execute_control.control_value")
	Value_Invariants(dot_value, "execute_control.dot_value")
	Functions_Validated_Invariants(function_values, "execute_control.function_values")
	workspace := (Workspace_Pointer)(workspace_state)
	control := control_value.Value.(*Node)
	pipe_value, found := program_node(
		program_value,
		Node_Link_Validated{Value: Node_Link_Validated_Value(control.First_Child)},
	)
	if !bool(found) {
		return EVALUATION_STATUS_PROGRAM_INVALID
	}
	pipe := pipe_value.Value.(*Node)
	if pipe.Kind != NODE_PIPE {
		return EVALUATION_STATUS_PROGRAM_INVALID
	}
	mark := workspace.Variable_Count.Value.(Workspace_Variable_Count_Storage_Value)
	previous_scope := workspace.Scope_Base.Value.(Workspace_Scope_Base_Storage_Value)
	restore := Scope_Restore_Validated{Value: Scope_Restore_Stored(
		Scope_Restore{Mark: Variable_Count(mark), Previous: Scope_Base(previous_scope)})}

	state, pipeline_status := pipeline_evaluate(
		workspace, program_value, pipe_value, dot_value, function_values,
	)
	if pipeline_status != EVALUATION_STATUS_OK {
		return pipeline_status
	}
	body_value, found := program_node(
		program_value,
		Node_Link_Validated{Value: Node_Link_Validated_Value(pipe.Next_Sibling)},
	)
	if !bool(found) {
		return EVALUATION_STATUS_PROGRAM_INVALID
	}
	body := body_value.Value.(*Node)
	if body.Kind != NODE_LIST {
		return EVALUATION_STATUS_PROGRAM_INVALID
	}
	if control.Kind == NODE_RANGE {
		range_status := range_begin(
			workspace, program_value, body_value, dot_value, state, restore,
		)
		return Evaluation_Status(range_status)
	}
	branch_status := control_body_select(
		workspace, program_value, control_value, body_value,
		dot_value, state.Value, restore,
	)
	return Evaluation_Status(branch_status)
}

func control_body_select(
	workspace_state Workspace_Pointer, program_value Program_Validated,
	control_value Node_Validated,
	body_value Node_Validated,
	dot_value Value,
	condition_value Value,
	restore_value Scope_Restore_Validated,
) (status Program_Status) {
	defer func() { Program_Status_Invariants(status, "control_body_select.status") }()
	Workspace_Pointer_Invariants(workspace_state, "control_body_select.workspace_state")
	Program_Validated_Invariants(program_value, "control_body_select.program_value")
	Node_Validated_Invariants(control_value, "control_body_select.control_value")
	Node_Validated_Invariants(body_value, "control_body_select.body_value")
	Value_Invariants(dot_value, "control_body_select.dot_value")
	Value_Invariants(condition_value, "control_body_select.condition_value")
	Scope_Restore_Validated_Invariants(restore_value, "control_body_select.restore_value")
	workspace := (Workspace_Pointer)(workspace_state)
	control := control_value.Value.(*Node)
	body := body_value.Value.(*Node)
	condition := Value(condition_value)
	restore := restore_value.Value
	selected_value := body_value
	selected := body
	selected_dot := dot_value
	if !bool(value_truth(condition)) {
		alternate := Node_Reference(body.Next_Sibling)
		if alternate == NO_NODE {
			workspace.Variable_Count.Value = Workspace_Variable_Count_Storage_Value(
				restore.Mark,
			)
			workspace.Scope_Base.Value = Workspace_Scope_Base_Storage_Value(
				restore.Previous,
			)
			return PROGRAM_STATUS_OK
		}
		var selected_valid Node_Validity
		selected_value, selected_valid = program_node(
			program_value,
			Node_Link_Validated{Value: Node_Link_Validated_Value(alternate)},
		)
		if !bool(selected_valid) {
			return PROGRAM_STATUS_INVALID
		}
		selected = selected_value.Value.(*Node)
		if selected.Kind != NODE_LIST {
			return PROGRAM_STATUS_INVALID
		}
	} else if control.Kind == NODE_WITH {
		selected_dot = condition
	}
	frame_push_list(
		workspace,
		Node_Link_Validated{Value: Node_Link_Validated_Value(selected.First_Child)},
		selected_dot,
		restore_value,
	)
	return PROGRAM_STATUS_OK
}

func range_begin(
	workspace_state Workspace_Pointer,
	program_value Program_Validated, body_value Node_Validated, dot_value Value,
	state_value Pipeline_State, restore_value Scope_Restore_Validated,
) (status Program_Execution_Limit_Status) {
	defer func() { Program_Execution_Limit_Status_Invariants(status, "range_begin.status") }()
	Workspace_Pointer_Invariants(workspace_state, "range_begin.workspace_state")
	Program_Validated_Invariants(program_value, "range_begin.program_value")
	Node_Validated_Invariants(body_value, "range_begin.body_value")
	Value_Invariants(dot_value, "range_begin.dot_value")
	Pipeline_State_Invariants(state_value, "range_begin.state_value")
	Scope_Restore_Validated_Invariants(restore_value, "range_begin.restore_value")
	workspace, body := Workspace_Pointer(workspace_state), body_value.Value.(*Node)
	dot, state := Value(dot_value), Pipeline_State(state_value)
	mark, previous_scope := restore_value.Value.Mark, restore_value.Value.Previous
	count, count_status := range_value_count(state.Value)
	if count_status != RANGE_STATUS_OK {
		return Program_Execution_Limit_Status(count_status)
	}
	if count == Collection_Count(bytes.SLICE_SIZE_MINIMUM) {
		alternate_reference := Node_Reference(body.Next_Sibling)
		if alternate_reference == NO_NODE {
			workspace.Variable_Count.Value = Workspace_Variable_Count_Storage_Value(
				mark,
			)
			workspace.Scope_Base.Value = Workspace_Scope_Base_Storage_Value(
				previous_scope,
			)
			return PROGRAM_EXECUTION_LIMIT_STATUS_OK
		}
		alternate_value, found := program_node(
			program_value,
			Node_Link_Validated{Value: Node_Link_Validated_Value(alternate_reference)},
		)
		if !bool(found) {
			return PROGRAM_EXECUTION_LIMIT_STATUS_PROGRAM_INVALID
		}
		alternate := alternate_value.Value.(*Node)
		if alternate.Kind != NODE_LIST {
			return PROGRAM_EXECUTION_LIMIT_STATUS_PROGRAM_INVALID
		}
		frame_push_list(
			workspace,
			Node_Link_Validated{
				Value: Node_Link_Validated_Value(alternate.First_Child),
			},
			dot,
			restore_value,
		)
		return PROGRAM_EXECUTION_LIMIT_STATUS_OK
	}
	frame_count := workspace.Frame_Count.Value.(Workspace_Frame_Count_Storage_Value)
	variable_count := workspace.Variable_Count.Value.(Workspace_Variable_Count_Storage_Value)
	scope_base := workspace.Scope_Base.Value.(Workspace_Scope_Base_Storage_Value)
	if int(frame_count) == len(workspace.Frames) {
		return PROGRAM_EXECUTION_LIMIT_STATUS_LIMIT_EXCEEDED
	}
	workspace.Frames[frame_count].Value = Frame_Stored(Frame{
		Kind: FRAME_RANGE, Reference: Node_Reference(body.First_Child),
		Dot: state.Value, Variable_Mark: Frame_Variable_Mark(mark),
		Iteration_Variables: Frame_Iteration_Variable_Count(variable_count),
		Scope_Base:          Frame_Scope_Base(scope_base),
		Previous_Scope_Base: Frame_Previous_Scope_Base(previous_scope),
		Declarations:        state.Declarations,
		Declaration_Count:   state.Declaration_Count,
		Assignment:          state.Assignment, Restore: true,
	})
	workspace.Frame_Count.Value = frame_count +
		Workspace_Frame_Count_Storage_Value(FRAME_COUNT_INCREMENT)
	return PROGRAM_EXECUTION_LIMIT_STATUS_OK
}

func range_schedule(
	workspace_state Workspace_Pointer, program_value Program_Validated,
	frame_state Frame_Storage_Pointer) {
	Workspace_Pointer_Invariants(workspace_state, "range_schedule.workspace_state")
	Program_Validated_Invariants(program_value, "range_schedule.program_value")
	Frame_Storage_Pointer_Invariants(frame_state, "range_schedule.frame_state")
	workspace := (Workspace_Pointer)(workspace_state)
	frame := &frame_state.Value
	count, count_status := range_value_count(frame.Dot)
	aver.Always(
		count_status == RANGE_STATUS_OK,
		"Range frame retains the iterable accepted when the frame was created.",
	)
	if Collection_Count(frame.Index) == count {
		frame_pop(workspace)
		return
	}
	index := frame.Index
	key := Value{}
	item := Value{}
	if Value_Kind(frame.Dot.Kind.Value) == VALUE_MAP {
		selected := Range_Map_Entry_Index(bytes.SLICE_SIZE_MINIMUM)
		key, item, selected = range_map_next(
			frame.Dot, Prior_Map_Entry_Index(frame.Map_Entry_Index),
			frame.Map_Selected,
		)
		frame.Map_Entry_Index = selected
		frame.Map_Selected = true
	} else {
		key, item = range_value_at(frame.Dot, Collection_Element_Index(index))
	}
	frame.Index++
	variable_count := workspace.Variable_Count.Value.(Workspace_Variable_Count_Storage_Value)
	scope_base := workspace.Scope_Base.Value.(Workspace_Scope_Base_Storage_Value)
	range_bind(workspace, program_value, frame_state, key, item)
	frame_push_list(
		workspace, Node_Link_Validated{Value: Node_Link_Validated_Value(frame.Reference)},
		item,
		Scope_Restore_Validated{Value: Scope_Restore_Stored(Scope_Restore{
			Mark:     Variable_Count(variable_count),
			Previous: Scope_Base(scope_base),
		})},
	)
}

func range_bind(
	workspace_state Workspace_Pointer, program_value Program_Validated,
	frame_state Frame_Storage_Pointer, key_value Value,
	item_value Value,
) {
	Workspace_Pointer_Invariants(workspace_state, "range_bind.workspace_state")
	Program_Validated_Invariants(program_value, "range_bind.program_value")
	Frame_Storage_Pointer_Invariants(frame_state, "range_bind.frame_state")
	Value_Invariants(key_value, "range_bind.key_value")
	Value_Invariants(item_value, "range_bind.item_value")
	workspace, frame := Workspace_Pointer(workspace_state), &frame_state.Value
	key, item := Value(key_value), Value(item_value)
	workspace.Variable_Count.Value = Workspace_Variable_Count_Storage_Value(
		frame.Iteration_Variables,
	)
	workspace.Scope_Base.Value = Workspace_Scope_Base_Storage_Value(frame.Scope_Base)
	declaration_start := int(frame.Variable_Mark)
	declaration_end := declaration_start
	if !bool(frame.Assignment) {
		declaration_end += int(frame.Declaration_Count)
	}
	aver.Always(
		declaration_end <= int(frame.Iteration_Variables),
		"Range frame retains the declarations bound before iteration.",
	)
	if bool(frame.Assignment) {
		for index := DECLARATION_COUNT_NONE; index < frame.Declaration_Count; index++ {
			value := item
			if frame.Declaration_Count == DECLARATION_COUNT_TWO {
				if index == DECLARATION_COUNT_NONE {
					value = key
				}
			}
			status := variable_bind_reference(
				workspace, program_value,
				declaration_reference(frame.Declarations, Declaration_Index(index)),
				value, frame.Assignment,
			)
			aver.Always(
				status == EXECUTION_LIMIT_STATUS_OK,
				"Parsed range assignment retains its visible binding.",
			)
		}
		return
	}
	if frame.Declaration_Count == DECLARATION_COUNT_ONE {
		workspace.Variables[declaration_start].Value = item
	} else if frame.Declaration_Count == DECLARATION_COUNT_TWO {
		workspace.Variables[declaration_start].Value = key
		workspace.Variables[declaration_start+VARIABLE_COUNT_INCREMENT].Value = item
	}
}

func range_map_next(
	value_state Value,
	previous_index_value Prior_Map_Entry_Index,
	previous_selection_value Map_Selection,
) (
	_ Value,
	_ Value,
	selected Range_Map_Entry_Index,
) {
	defer func() { Range_Map_Entry_Index_Invariants(selected, "range_map_next.selected") }()
	Value_Invariants(value_state, "range_map_next.value_state")
	Prior_Map_Entry_Index_Invariants(
		previous_index_value, "range_map_next.previous_index_value",
	)
	Map_Selection_Invariants(
		previous_selection_value, "range_map_next.previous_selection_value",
	)
	var key Value
	defer func() { Value_Invariants(key, "range_map_next.key") }()
	var item Value
	defer func() { Value_Invariants(item, "range_map_next.item") }()
	value := Value(value_state)
	fields := Fields(value.Fields.Value)
	validated_fields := Map_Fields_Validated{Value: Fields_Storage_Value(fields)}
	previous_index := int(previous_index_value)
	previous_selected := bool(previous_selection_value)
	found := false
	selected_index := bytes.SLICE_SIZE_MINIMUM
	for candidate_index := range fields {
		if previous_selected {
			order := map_field_order(
				validated_fields, Range_Map_Entry_Index(candidate_index),
				Prior_Map_Entry_Index(previous_index),
			)
			if order < ORDER_SAME {
				continue
			}
			if order == ORDER_SAME {
				if candidate_index <= previous_index {
					continue
				}
			}
		}
		if found {
			order := map_field_order(
				validated_fields, Range_Map_Entry_Index(candidate_index),
				Prior_Map_Entry_Index(selected_index),
			)
			if order > ORDER_SAME {
				continue
			}
			if order == ORDER_SAME {
				if candidate_index > selected_index {
					continue
				}
			}
		}
		selected_index = candidate_index
		found = true
	}
	aver.Always(found, "Counted map iteration requests one remaining entry.")
	field := fields[selected_index]
	key, item = field.Key.Data, field.Value.Data
	selected = Range_Map_Entry_Index(selected_index)
	return key, item, selected
}

func map_field_order(
	fields_value Map_Fields_Validated,
	left_index Range_Map_Entry_Index,
	right_index Prior_Map_Entry_Index,
) (order Order) {
	defer func() { Order_Invariants(order, "map_field_order.order") }()
	Map_Fields_Validated_Invariants(fields_value, "map_field_order.fields_value")
	Range_Map_Entry_Index_Invariants(left_index, "map_field_order.left_index")
	Prior_Map_Entry_Index_Invariants(right_index, "map_field_order.right_index")
	fields := fields_value.Value.(Fields_Storage_Value)
	return map_key_order(
		Map_Key_Validated{Value: fields[left_index].Key.Data},
		Map_Key_Validated{Value: fields[right_index].Key.Data},
	)
}

func map_key_order(
	left_value Map_Key_Validated,
	right_value Map_Key_Validated,
) (order Order) {
	defer func() { Order_Invariants(order, "map_key_order.order") }()
	Map_Key_Validated_Invariants(left_value, "map_key_order.left_value")
	Map_Key_Validated_Invariants(right_value, "map_key_order.right_value")
	left := left_value.Value
	right := right_value.Value
	left_kind := Value_Kind(left.Kind.Value)
	right_kind := Value_Kind(right.Kind.Value)
	if left_kind != right_kind {
		return uint64_order(Unsigned(left_kind), Unsigned(right_kind))
	}
	switch left_kind {
	case VALUE_NIL:
		return ORDER_SAME
	case VALUE_BOOLEAN, VALUE_UNSIGNED:
		return uint64_order(
			Unsigned(left.Scalar.Real),
			Unsigned(right.Scalar.Real),
		)
	case VALUE_INTEGER:
		return int64_order(
			Integer(left.Scalar.Real),
			Integer(right.Scalar.Real),
		)
	case VALUE_FLOAT:
		return float64_sort_order(
			Float(left.Scalar.Real),
			Float(right.Scalar.Real),
		)
	case VALUE_COMPLEX:
		order = float64_sort_order(
			Float(left.Scalar.Real),
			Float(right.Scalar.Real),
		)
		if order == ORDER_SAME {
			order = float64_sort_order(
				Float(left.Scalar.Imaginary),
				Float(right.Scalar.Imaginary),
			)
		}
		return order
	case VALUE_TEXT:
		return bytes_order(
			Text(left.Text.Value), Text(right.Text.Value),
		)
	}
	return ORDER_SAME
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

func range_value_count(value_state Value) (
	_ Collection_Count,
	status Range_Status,
) {
	defer func() { Range_Status_Invariants(status, "range_value_count.status") }()
	Value_Invariants(value_state, "range_value_count.value_state")
	count := Collection_Count(bytes.SLICE_SIZE_MINIMUM)
	defer func() { Collection_Count_Invariants(count, "range_value_count.count") }()
	value := Value(value_state)
	switch Value_Kind(value.Kind.Value) {
	case VALUE_NIL:
		return count, RANGE_STATUS_OK
	case VALUE_SEQUENCE:
		count = Collection_Count(len(Values(value.Values.Value)))
		return count, RANGE_STATUS_OK
	case VALUE_MAP:
		count = Collection_Count(len(Fields(value.Fields.Value)))
		return count, RANGE_STATUS_OK
	case VALUE_INTEGER:
		integer := Integer(int64(value.Scalar.Real))
		if integer < Integer(bits.WORD_64_MINIMUM) {
			return count, RANGE_STATUS_OK
		}
		if integer > Integer(VALUE_COUNT_MAXIMUM) {
			return count, RANGE_STATUS_LIMIT_EXCEEDED
		}
		count = Collection_Count(integer)
		return count, RANGE_STATUS_OK
	case VALUE_UNSIGNED:
		unsigned := Unsigned(value.Scalar.Real)
		if unsigned > Unsigned(VALUE_COUNT_MAXIMUM) {
			return count, RANGE_STATUS_LIMIT_EXCEEDED
		}
		count = Collection_Count(unsigned)
		return count, RANGE_STATUS_OK
	}
	return count, RANGE_STATUS_KIND_INVALID
}

func range_value_at(
	value_state Value,
	index_value Collection_Element_Index,
) (_ Value, item Value) {
	defer func() { Value_Invariants(item, "range_value_at.item") }()
	Value_Invariants(value_state, "range_value_at.value_state")
	Collection_Element_Index_Invariants(index_value, "range_value_at.index_value")
	var key Value
	defer func() { Value_Invariants(key, "range_value_at.key") }()
	value := Value(value_state)
	index := Collection_Element_Index(index_value)
	kind := Value_Kind(value.Kind.Value)
	Range_Value_Kind_Invariants(Range_Value_Kind(kind), "range_value_at.kind")
	switch kind {
	case VALUE_SEQUENCE:
		key = Value_Of_Integer(Integer(index))
		item = Values(value.Values.Value)[index]
	case VALUE_INTEGER, VALUE_UNSIGNED:
		key, item = Value_Nil(), Value_Of_Integer(Integer(index))
	}
	return key, item
}

func range_unwind(
	workspace_state Workspace_Pointer, flow_value Range_Flow,
) (found Function_Found) {
	defer func() { Function_Found_Invariants(found, "range_unwind.found") }()
	Workspace_Pointer_Invariants(workspace_state, "range_unwind.workspace_state")
	Range_Flow_Invariants(flow_value, "range_unwind.flow_value")
	workspace := (Workspace_Pointer)(workspace_state)
	flow := Range_Flow(flow_value)
	for workspace.Frame_Count.Value.(Workspace_Frame_Count_Storage_Value) >
		Workspace_Frame_Count_Storage_Value(
			bytes.SLICE_SIZE_MINIMUM,
		) {
		index := int(workspace.Frame_Count.Value.(Workspace_Frame_Count_Storage_Value)) -
			FRAME_COUNT_INCREMENT
		frame := workspace.Frames[index].Value
		if frame.Kind == FRAME_RANGE {
			workspace.Variable_Count.Value = Workspace_Variable_Count_Storage_Value(
				frame.Iteration_Variables,
			)
			workspace.Scope_Base.Value = Workspace_Scope_Base_Storage_Value(
				frame.Scope_Base,
			)
			if flow == Range_Flow(FLOW_BREAK) {
				workspace.Variable_Count.Value =
					Workspace_Variable_Count_Storage_Value(
						frame.Variable_Mark,
					)
				workspace.Scope_Base.Value = Workspace_Scope_Base_Storage_Value(
					frame.Previous_Scope_Base,
				)
				stored_count := workspace.Frame_Count.Value
				frame_count := stored_count.(Workspace_Frame_Count_Storage_Value)
				workspace.Frame_Count.Value = frame_count -
					Workspace_Frame_Count_Storage_Value(FRAME_COUNT_INCREMENT)
			}
			return true
		}
		frame_pop(workspace)
	}
	return false
}

func execute_template(
	workspace_state Workspace_Pointer, program_value Program_Validated,
	invocation_value Node_Validated, dot_value Value,
	function_values Functions_Validated,
) (status Evaluation_Status) {
	defer func() { Evaluation_Status_Invariants(status, "execute_template.status") }()
	Workspace_Pointer_Invariants(workspace_state, "execute_template.workspace_state")
	Program_Validated_Invariants(program_value, "execute_template.program_value")
	Node_Validated_Invariants(invocation_value, "execute_template.invocation_value")
	Value_Invariants(dot_value, "execute_template.dot_value")
	Functions_Validated_Invariants(function_values, "execute_template.function_values")
	workspace, invocation := Workspace_Pointer(workspace_state), invocation_value.Value.(*Node)
	dot, cursor := Value(dot_value), Value_Nil()
	if invocation.First_Child != Node_First_Child(NO_NODE) {
		pipe_value, found := program_node(
			program_value,
			Node_Link_Validated{
				Value: Node_Link_Validated_Value(invocation.First_Child),
			},
		)
		if !bool(found) {
			return EVALUATION_STATUS_PROGRAM_INVALID
		}
		pipe := pipe_value.Value.(*Node)
		if pipe.Kind != NODE_PIPE {
			return EVALUATION_STATUS_PROGRAM_INVALID
		}
		state, pipeline_status := pipeline_evaluate(
			workspace, program_value, pipe_value, dot, function_values,
		)
		if pipeline_status != EVALUATION_STATUS_OK {
			return pipeline_status
		}
		cursor = state.Value
	}
	definition, definition_status := definition_find(
		workspace, program_value, invocation_value,
	)
	if definition_status != DEFINITION_STATUS_OK {
		return Evaluation_Status(definition_status)
	}
	body_value, found := program_node(
		program_value,
		Node_Link_Validated{
			Value: Node_Link_Validated_Value(definition.Value.(*Node).First_Child),
		},
	)
	if !bool(found) {
		return EVALUATION_STATUS_PROGRAM_INVALID
	}
	if body_value.Value.(*Node).Kind != NODE_LIST {
		return EVALUATION_STATUS_PROGRAM_INVALID
	}
	mark := workspace.Variable_Count.Value.(Workspace_Variable_Count_Storage_Value)
	previous_scope := workspace.Scope_Base.Value.(Workspace_Scope_Base_Storage_Value)
	if int(mark) == len(workspace.Variables) {
		return EVALUATION_STATUS_LIMIT_EXCEEDED
	}
	workspace.Scope_Base.Value = Workspace_Scope_Base_Storage_Value(mark)
	workspace.Variables[mark] = Variable{
		Root: true, Value: cursor,
	}
	workspace.Variable_Count.Value = mark +
		Workspace_Variable_Count_Storage_Value(VARIABLE_COUNT_INCREMENT)
	restore := Scope_Restore_Validated{Value: Scope_Restore_Stored(Scope_Restore{
		Mark: Variable_Count(mark), Previous: Scope_Base(previous_scope),
	})}
	frame_push_list(
		workspace, Node_Link_Validated{
			Value: Node_Link_Validated_Value(body_value.Value.(*Node).First_Child),
		}, cursor, restore,
	)
	return EVALUATION_STATUS_OK
}

func definition_find(
	workspace_state Workspace_Pointer, program_value Program_Validated,
	invocation_value Node_Validated,
) (_ Node_Validated, status Definition_Status) {
	defer func() { Definition_Status_Invariants(status, "definition_find.status") }()
	Workspace_Pointer_Invariants(workspace_state, "definition_find.workspace_state")
	Program_Validated_Invariants(program_value, "definition_find.program_value")
	Node_Validated_Invariants(invocation_value, "definition_find.invocation_value")
	var definition Node_Validated
	defer func() { Node_Validated_Invariants(definition, "definition_find.definition") }()
	workspace := (Workspace_Pointer)(workspace_state)
	program := program_value
	invocation := invocation_value.Value.(*Node)
	left_count, left_status := Quoted_Unquote_Into(
		Quoted_Output(workspace.Name_Left), program.Source,
		Quoted_Span_Unvalidated{
			Start: Quoted_Start_Unvalidated(invocation.Value_Start),
			End:   Quoted_End_Unvalidated(invocation.Value_End),
		},
	)
	if left_status != PARSE_STATUS_OK {
		return Node_Validated{}, DEFINITION_STATUS_PROGRAM_INVALID
	}
	reference := Node_Reference(program.Document.First_Template)
	if reference != NO_NODE {
		reference += Node_Reference(ROOT_NODE_COUNT)
	}
	index := Template_Count(TEMPLATE_COUNT_MINIMUM)
	for index < Template_Count(program.Document.Template_Count) {
		candidate_value, found := program_node(
			program_value,
			Node_Link_Validated{Value: Node_Link_Validated_Value(reference)},
		)
		if !bool(found) {
			return Node_Validated{}, DEFINITION_STATUS_PROGRAM_INVALID
		}
		candidate := candidate_value.Value.(*Node)
		if candidate.Kind != NODE_TEMPLATE {
			return Node_Validated{}, DEFINITION_STATUS_PROGRAM_INVALID
		}
		right_count, right_status := Quoted_Unquote_Into(
			Quoted_Output(workspace.Name_Right), program.Source,
			Quoted_Span_Unvalidated{
				Start: Quoted_Start_Unvalidated(candidate.Value_Start),
				End:   Quoted_End_Unvalidated(candidate.Value_End),
			},
		)
		if right_status != PARSE_STATUS_OK {
			return Node_Validated{}, DEFINITION_STATUS_PROGRAM_INVALID
		}
		if left_count == right_count {
			left_name := workspace.Name_Left[:left_count]
			right_name := workspace.Name_Right[:right_count]
			if bytes_equal(
				Text(left_name), Text(right_name),
			) {
				definition = candidate_value
				return definition, DEFINITION_STATUS_OK
			}
		}
		reference = Node_Reference(candidate.Next_Sibling)
		index++
	}
	return Node_Validated{}, DEFINITION_STATUS_ABSENT
}

func pipeline_evaluate(
	workspace_state Workspace_Pointer, program_value Program_Validated,
	pipe_value Node_Validated,
	dot_value Value,
	function_values Functions_Validated,
) (_ Pipeline_State, status Evaluation_Status) {
	defer func() { Evaluation_Status_Invariants(status, "pipeline_evaluate.status") }()
	Workspace_Pointer_Invariants(workspace_state, "pipeline_evaluate.workspace_state")
	Program_Validated_Invariants(program_value, "pipeline_evaluate.program_value")
	Node_Validated_Invariants(pipe_value, "pipeline_evaluate.pipe_value")
	Value_Invariants(dot_value, "pipeline_evaluate.dot_value")
	Functions_Validated_Invariants(function_values, "pipeline_evaluate.function_values")
	var state Pipeline_State
	defer func() { Pipeline_State_Invariants(state, "pipeline_evaluate.state") }()
	workspace := (Workspace_Pointer)(workspace_state)
	functions := function_values
	depth := ACTIVE_FRAME_COUNT_MINIMUM
	argument_count := Argument_Count_Tracked{
		Value: Argument_Count_Tracked_Value(ARGUMENT_COUNT_MINIMUM),
	}
	push_status := evaluation_push(
		&workspace.Evaluations[bytes.SLICE_SIZE_MINIMUM], program_value,
		pipe_value,
		Value(dot_value), argument_count,
	)
	status = Evaluation_Status(push_status)
	for status == EVALUATION_STATUS_OK {
		frame := &workspace.Evaluations[depth-ACTIVE_FRAME_COUNT_MINIMUM]
		switch frame.Control.Stage {
		case EVALUATION_STAGE_COMMAND:
			command_status := evaluation_command_begin(
				frame, program_value, argument_count,
			)
			status = Evaluation_Status(command_status)
		case EVALUATION_STAGE_FIRST, EVALUATION_STAGE_ARGUMENT:
			var term_status Term_Status
			depth, argument_count, term_status = evaluation_term_step(
				workspace, program_value, functions, depth, argument_count,
			)
			status = Evaluation_Status(term_status)
		case EVALUATION_STAGE_CALL:
			var call_status Call_Status
			argument_count, call_status = evaluation_command_call(
				frame, workspace, program_value, functions, argument_count,
			)
			status = Evaluation_Status(call_status)
		case EVALUATION_STAGE_DONE:
			complete_state, complete_status := evaluation_complete(
				frame, workspace, program_value,
			)
			state = complete_state
			status = Evaluation_Status(complete_status)
			if depth == ACTIVE_FRAME_COUNT_MINIMUM {
				return state, status
			}
			depth--
			if complete_status == PROGRAM_EXECUTION_LIMIT_STATUS_OK {
				var deliver_status Program_Execution_Status
				parent_index := depth - ACTIVE_FRAME_COUNT_MINIMUM
				parent := &workspace.Evaluations[parent_index]
				argument_count, deliver_status = evaluation_child_deliver(
					Evaluation_Frame_Pointer(parent), workspace, program_value,
					state.Value, argument_count,
				)
				status = Evaluation_Status(deliver_status)
			}
		default:
			status = EVALUATION_STATUS_PROGRAM_INVALID
		}
	}
	return Pipeline_State{}, status
}

func evaluation_push(
	frame_state Evaluation_Frame_Pointer, program_value Program_Validated,
	pipe_value Node_Validated,
	dot_value Value,
	argument_base_value Argument_Count_Tracked,
) (status Program_Status) {
	defer func() { Program_Status_Invariants(status, "evaluation_push.status") }()
	Evaluation_Frame_Pointer_Invariants(frame_state, "evaluation_push.frame_state")
	Program_Validated_Invariants(program_value, "evaluation_push.program_value")
	Node_Validated_Invariants(pipe_value, "evaluation_push.pipe_value")
	Value_Invariants(dot_value, "evaluation_push.dot_value")
	Argument_Count_Tracked_Invariants(
		argument_base_value, "evaluation_push.argument_base_value",
	)
	frame := (Evaluation_Frame_Pointer)(frame_state)
	program := program_value
	pipe := pipe_value.Value.(*Node)
	*frame = Evaluation_Frame{}
	frame.Nodes.Pipe = Evaluation_Pipe_Node(pipe)
	frame.Values.Dot = Evaluation_Dot_Value(dot_value)
	frame.Argument_Bases.Value = Evaluation_Argument_Bases_Value(
		argument_base_value.Value.(Argument_Count_Tracked_Value),
	)
	frame.Control.Stage = EVALUATION_STAGE_COMMAND
	reference := Node_Reference(pipe.First_Child)
	state := &frame.States.Value
	for reference != NO_NODE {
		node_value, found := program_node(
			program_value,
			Node_Link_Validated{Value: Node_Link_Validated_Value(reference)},
		)
		if !bool(found) {
			return PROGRAM_STATUS_INVALID
		}
		node := node_value.Value.(*Node)
		if node.Kind != NODE_VARIABLE {
			break
		}
		if state.Declaration_Count == DECLARATION_COUNT_TWO {
			return PROGRAM_STATUS_INVALID
		}
		declaration := Declaration_Reference_Validated{
			Value: Declaration_Reference_Validated_Value(reference),
		}
		if state.Declaration_Count == DECLARATION_COUNT_NONE {
			state.Declarations.First.Value =
				Declaration_First_Reference(
					declaration.Value.(Declaration_Reference_Validated_Value),
				)
		} else {
			state.Declarations.Second.Value =
				Declaration_Second_Reference(
					declaration.Value.(Declaration_Reference_Validated_Value),
				)
		}
		state.Declaration_Count++
		reference = Node_Reference(node.Next_Sibling)
	}
	operator := program.Source.Data.Value[pipe.Value_Start:pipe.Value_End]
	if state.Declaration_Count > DECLARATION_COUNT_NONE {
		if bytes_equal(Text(operator), Text("=")) {
			state.Assignment = true
		} else if !bytes_equal(Text(operator), Text(":=")) {
			return PROGRAM_STATUS_INVALID
		}
	}
	frame.References.Command = Evaluation_Command_Reference(reference)
	Evaluation_Frame_Pointer_Invariants(frame, "evaluation_push.frame")
	return PROGRAM_STATUS_OK
}

func evaluation_command_begin(
	frame_state Evaluation_Frame_Pointer, program_value Program_Validated,
	argument_count_value Argument_Count_Tracked,
) (status Program_Status) {
	defer func() {
		Program_Status_Invariants(status, "evaluation_command_begin.status")
	}()
	Evaluation_Frame_Pointer_Invariants(frame_state, "evaluation_command_begin.frame_state")
	Program_Validated_Invariants(program_value, "evaluation_command_begin.program_value")
	Argument_Count_Tracked_Invariants(
		argument_count_value, "evaluation_command_begin.argument_count_value",
	)
	frame := (Evaluation_Frame_Pointer)(frame_state)
	reference := Node_Reference(frame.References.Command)
	if reference == NO_NODE {
		frame.Control.Stage = EVALUATION_STAGE_DONE
		return PROGRAM_STATUS_OK
	}
	command_value, found := program_node(
		program_value, Node_Link_Validated{Value: Node_Link_Validated_Value(reference)},
	)
	if !bool(found) {
		return PROGRAM_STATUS_INVALID
	}
	command := command_value.Value.(*Node)
	if command.Kind != NODE_COMMAND {
		return PROGRAM_STATUS_INVALID
	}
	first_reference := Node_Reference(command.First_Child)
	if first_reference == NO_NODE {
		return PROGRAM_STATUS_INVALID
	}
	first_value, found := program_node(
		program_value,
		Node_Link_Validated{Value: Node_Link_Validated_Value(first_reference)},
	)
	if !bool(found) {
		return PROGRAM_STATUS_INVALID
	}
	first := first_value.Value.(*Node)
	frame.References.Command =
		Evaluation_Command_Reference(command.Next_Sibling)
	frame.Argument_Bases.Value = Evaluation_Argument_Bases_Value(
		argument_count_value.Value.(Argument_Count_Tracked_Value),
	)
	if first.Kind == NODE_IDENTIFIER {
		frame.Nodes.Identifier = Evaluation_Identifier_Node(first)
		frame.References.Term =
			Evaluation_Term_Reference(first.Next_Sibling)
		frame.Control.Mode = EVALUATION_MODE_IDENTIFIER
		frame.Control.Stage = EVALUATION_STAGE_ARGUMENT
		return PROGRAM_STATUS_OK
	}
	frame.References.Term = Evaluation_Term_Reference(first_reference)
	frame.Control.Mode = EVALUATION_MODE_VALUE
	frame.Control.Stage = EVALUATION_STAGE_FIRST
	return PROGRAM_STATUS_OK
}

func evaluation_term_step(
	workspace_state Workspace_Pointer, program_value Program_Validated,
	function_values Functions_Validated, depth_value Active_Frame_Count,
	argument_count_value Argument_Count_Tracked,
) (_ Active_Frame_Count, _ Argument_Count_Tracked, status Term_Status) {
	defer func() { Term_Status_Invariants(status, "evaluation_term_step.status") }()
	Workspace_Pointer_Invariants(workspace_state, "evaluation_term_step.workspace_state")
	Program_Validated_Invariants(program_value, "evaluation_term_step.program_value")
	Functions_Validated_Invariants(function_values, "evaluation_term_step.function_values")
	Active_Frame_Count_Invariants(depth_value, "evaluation_term_step.depth_value")
	Argument_Count_Tracked_Invariants(
		argument_count_value, "evaluation_term_step.argument_count_value",
	)
	depth, count := depth_value, argument_count_value
	defer func() { Active_Frame_Count_Invariants(depth, "evaluation_term_step.depth") }()
	defer func() { Argument_Count_Tracked_Invariants(count, "evaluation_term_step.count") }()
	frame := &workspace_state.Evaluations[depth-ACTIVE_FRAME_COUNT_MINIMUM]
	reference := Node_Reference(frame.References.Term)
	if reference == NO_NODE {
		frame.Control.Stage = EVALUATION_STAGE_CALL
		return depth, count, TERM_STATUS_OK
	}
	reference_value := Node_Link_Validated{Value: Node_Link_Validated_Value(reference)}
	node_value, found := program_node(program_value, reference_value)
	if !bool(found) {
		return depth, count, TERM_STATUS_PROGRAM_INVALID
	}
	node := node_value.Value.(*Node)
	frame.References.Term = Evaluation_Term_Reference(node.Next_Sibling)
	if node.Kind == NODE_PIPE {
		child_depth, child_status := evaluation_child_push(
			workspace_state, program_value, node_value,
			Node_Optional_Validated{}, depth, count,
		)
		depth, status = child_depth, Term_Status(child_status)
		return depth, count, status
	}
	if node.Kind == NODE_CHAIN {
		pipe_reference := Node_Link_Validated{
			Value: Node_Link_Validated_Value(node.First_Child),
		}
		pipe_value, pipe_found := program_node(program_value, pipe_reference)
		if !bool(pipe_found) {
			return depth, count, TERM_STATUS_PROGRAM_INVALID
		}
		pipe := pipe_value.Value.(*Node)
		if pipe.Kind != NODE_PIPE {
			return depth, count, TERM_STATUS_PROGRAM_INVALID
		}
		child_depth, child_status := evaluation_child_push(
			workspace_state, program_value, pipe_value,
			Node_Optional_Validated{Value: Node_Stored(node)}, depth, count,
		)
		depth, status = child_depth, Term_Status(child_status)
		return depth, count, status
	}
	location := Node_Location_Validated{Value: Node_Location_Stored{
		Reference: reference, Node: Node_Stored(node)}}
	value, term_status := term_scalar_evaluate(
		workspace_state, program_value, location,
		Value(frame.Values.Dot), function_values,
	)
	if term_status != TERM_SCALAR_STATUS_OK {
		return depth, count, Term_Status(term_status)
	}
	var delivery_status Limit_Status
	count, delivery_status = evaluation_value_deliver(
		frame, workspace_state, value, count,
	)
	if delivery_status == LIMIT_STATUS_OK {
		evaluation_short_circuit(frame, program_value, value)
	}
	return depth, count, Term_Status(delivery_status)
}

func evaluation_child_push(
	workspace_state Workspace_Pointer, program_value Program_Validated,
	pipe_value Node_Validated,
	chain_value Node_Optional_Validated,
	depth_value Active_Frame_Count,
	argument_count_value Argument_Count_Tracked,
) (_ Active_Frame_Count, status Program_Limit_Status) {
	defer func() { Program_Limit_Status_Invariants(status, "evaluation_child_push.status") }()
	Workspace_Pointer_Invariants(workspace_state, "evaluation_child_push.workspace_state")
	Program_Validated_Invariants(program_value, "evaluation_child_push.program_value")
	Node_Validated_Invariants(pipe_value, "evaluation_child_push.pipe_value")
	Node_Optional_Validated_Invariants(
		chain_value, "evaluation_child_push.chain_value",
	)
	Active_Frame_Count_Invariants(depth_value, "evaluation_child_push.depth_value")
	Argument_Count_Tracked_Invariants(
		argument_count_value, "evaluation_child_push.argument_count_value",
	)
	workspace := (Workspace_Pointer)(workspace_state)
	depth := Active_Frame_Count(depth_value)
	defer func() { Active_Frame_Count_Invariants(depth, "evaluation_child_push.depth") }()
	if int(depth) == len(workspace.Evaluations) {
		return depth, PROGRAM_LIMIT_STATUS_LIMIT_EXCEEDED
	}
	parent := &workspace.Evaluations[depth-ACTIVE_FRAME_COUNT_MINIMUM]
	chain, _ := chain_value.Value.(*Node)
	parent.Control.Chain = EVALUATION_CONTROL_FALSE
	if chain != nil {
		if chain.Kind == NODE_CHAIN {
			parent.Nodes.Chain = Evaluation_Chain_Node(chain)
			parent.Control.Chain = EVALUATION_CONTROL_TRUE
		}
	}
	push_status := evaluation_push(
		&workspace.Evaluations[depth], program_value, pipe_value,
		Value(parent.Values.Dot), argument_count_value,
	)
	status = Program_Limit_Status(push_status)
	if push_status == PROGRAM_STATUS_OK {
		depth++
	}
	return depth, status
}

func evaluation_value_deliver(
	frame_state Evaluation_Frame_Pointer, workspace_state Workspace_Pointer, value_state Value,
	argument_count_value Argument_Count_Tracked,
) (_ Argument_Count_Tracked, status Limit_Status) {
	defer func() { Limit_Status_Invariants(status, "evaluation_value_deliver.status") }()
	Evaluation_Frame_Pointer_Invariants(frame_state, "evaluation_value_deliver.frame_state")
	Workspace_Pointer_Invariants(workspace_state, "evaluation_value_deliver.workspace_state")
	Value_Invariants(value_state, "evaluation_value_deliver.value_state")
	Argument_Count_Tracked_Invariants(
		argument_count_value, "evaluation_value_deliver.argument_count_value",
	)
	argument_count := argument_count_value
	defer func() {
		Argument_Count_Tracked_Invariants(
			argument_count, "evaluation_value_deliver.argument_count",
		)
	}()
	frame := (Evaluation_Frame_Pointer)(frame_state)
	value := Value(value_state)
	if frame.Control.Stage == EVALUATION_STAGE_FIRST {
		frame.Values.Current = Evaluation_Current_Value(value)
		frame.Control.Stage = EVALUATION_STAGE_ARGUMENT
		return argument_count, LIMIT_STATUS_OK
	}
	aver.Always(
		frame.Control.Stage == EVALUATION_STAGE_ARGUMENT,
		"Evaluation state machine delivers only first values and arguments.",
	)
	workspace := (Workspace_Pointer)(workspace_state)
	count := argument_count.Value.(Argument_Count_Tracked_Value)
	if int(count) == len(workspace.Arguments) {
		return argument_count, LIMIT_STATUS_EXCEEDED
	}
	workspace.Arguments[count] = value
	argument_count.Value = count + ARGUMENT_COUNT_INCREMENT
	return argument_count, LIMIT_STATUS_OK
}

func evaluation_child_deliver(
	frame_state Evaluation_Frame_Delivery_Pointer, workspace_state Workspace_Pointer,
	program_value Program_Validated,
	value_state Value,
	argument_count_value Argument_Count_Tracked,
) (_ Argument_Count_Tracked, status Program_Execution_Status) {
	defer func() {
		Program_Execution_Status_Invariants(status, "evaluation_child_deliver.status")
	}()
	Evaluation_Frame_Delivery_Pointer_Invariants(
		frame_state, "evaluation_child_deliver.frame_state",
	)
	Workspace_Pointer_Invariants(workspace_state, "evaluation_child_deliver.workspace_state")
	Program_Validated_Invariants(program_value, "evaluation_child_deliver.program_value")
	Value_Invariants(value_state, "evaluation_child_deliver.value_state")
	Argument_Count_Tracked_Invariants(
		argument_count_value, "evaluation_child_deliver.argument_count_value",
	)
	argument_count := argument_count_value
	defer func() {
		Argument_Count_Tracked_Invariants(
			argument_count, "evaluation_child_deliver.argument_count",
		)
	}()
	frame := frame_state.(Evaluation_Frame_Pointer)
	value := Value(value_state)
	if frame.Control.Chain == EVALUATION_CONTROL_TRUE {
		chain, _ := frame.Nodes.Chain.(*Node)
		program := program_value
		path := program.Source.Data.Value[chain.Value_Start:chain.Value_End]
		access_value, access_status := field_path(Text(path), value)
		if access_status != VALUE_ACCESS_STATUS_OK {
			return argument_count, Program_Execution_Status(access_status)
		}
		value = access_value
		frame.Control.Chain = EVALUATION_CONTROL_FALSE
	}
	var deliver_status Limit_Status
	argument_count, deliver_status = evaluation_value_deliver(
		frame, (Workspace_Pointer)(workspace_state), value,
		argument_count,
	)
	aver.Always(
		deliver_status == LIMIT_STATUS_OK,
		"One child result cannot exhaust syntax-sized argument storage.",
	)
	evaluation_short_circuit(frame, program_value, value)
	return argument_count, Program_Execution_Status(deliver_status)
}

func evaluation_short_circuit(
	frame_state Evaluation_Frame_Pointer, program_value Program_Validated,
	value_state Value,
) {
	Evaluation_Frame_Pointer_Invariants(frame_state, "evaluation_short_circuit.frame_state")
	Program_Validated_Invariants(program_value, "evaluation_short_circuit.program_value")
	Value_Invariants(value_state, "evaluation_short_circuit.value_state")
	frame := (Evaluation_Frame_Pointer)(frame_state)
	if frame.Control.Mode != EVALUATION_MODE_IDENTIFIER {
		return
	}
	identifier := frame.Nodes.Identifier.(*Node)
	program := program_value
	source := program.Source.Data.Value
	name := source[identifier.Value_Start:identifier.Value_End]
	truth := bool(value_truth(Value(value_state)))
	if bool(bytes_equal(Text(name), Text("and"))) {
		if !truth {
			frame.References.Term = Evaluation_Term_Reference(NO_NODE)
		}
		return
	}
	if bool(bytes_equal(Text(name), Text("or"))) {
		if truth {
			frame.References.Term = Evaluation_Term_Reference(NO_NODE)
		}
	}
}

func evaluation_command_call(
	frame_state Evaluation_Frame_Pointer, workspace_state Workspace_Pointer,
	program_value Program_Validated,
	function_values Functions_Validated,
	argument_count_value Argument_Count_Tracked,
) (_ Argument_Count_Tracked, status Call_Status) {
	defer func() { Call_Status_Invariants(status, "evaluation_command_call.status") }()
	Evaluation_Frame_Pointer_Invariants(frame_state, "evaluation_command_call.frame_state")
	Workspace_Pointer_Invariants(workspace_state, "evaluation_command_call.workspace_state")
	Program_Validated_Invariants(program_value, "evaluation_command_call.program_value")
	Functions_Validated_Invariants(function_values, "evaluation_command_call.function_values")
	Argument_Count_Tracked_Invariants(
		argument_count_value, "evaluation_command_call.argument_count_value",
	)
	argument_count := argument_count_value
	defer func() {
		Argument_Count_Tracked_Invariants(
			argument_count, "evaluation_command_call.argument_count",
		)
	}()
	frame := (Evaluation_Frame_Pointer)(frame_state)
	workspace := (Workspace_Pointer)(workspace_state)
	base := frame.Argument_Bases.Value
	state := &frame.States.Value
	count := argument_count.Value.(Argument_Count_Tracked_Value)
	if frame.Control.Present == EVALUATION_CONTROL_TRUE {
		if int(count) == len(workspace.Arguments) {
			return argument_count, CALL_STATUS_LIMIT_EXCEEDED
		}
		workspace.Arguments[count] = state.Value
		count++
	}
	argument_count.Value = count
	arguments := Arguments_Staged{Value: Arguments_Staged_Value(
		Values(workspace.Arguments[base:count]),
	)}
	var result Value
	if frame.Control.Mode == EVALUATION_MODE_IDENTIFIER {
		result, status = function_call_identifier(
			workspace, program_value,
			Node_Validated{Value: Node_Stored(frame.Nodes.Identifier.(*Node))},
			function_values, arguments,
		)
	} else {
		result = Value(frame.Values.Current)
		if count != Argument_Count_Tracked_Value(base) {
			if Value_Kind(result.Kind.Value) != VALUE_FUNCTION {
				status = CALL_STATUS_EXECUTION_INVALID
			} else if Function_Call(result.Function.Value) == nil {
				status = CALL_STATUS_EXECUTION_INVALID
			} else {
				var call_status Function_Call_Status
				result, call_status = callback_call(
					workspace, Function_Call(result.Function.Value), arguments,
				)
				status = Call_Status(call_status)
			}
		}
	}
	argument_count = Argument_Count_Tracked{Value: Argument_Count_Tracked_Value(base)}
	if status != CALL_STATUS_OK {
		return argument_count, status
	}
	state.Value = result
	frame.Control.Present = EVALUATION_CONTROL_TRUE
	frame.Control.Stage = EVALUATION_STAGE_COMMAND
	return argument_count, CALL_STATUS_OK
}

func evaluation_complete(
	frame_state Evaluation_Frame_Pointer, workspace_state Workspace_Pointer,
	program_value Program_Validated,
) (_ Pipeline_State, status Program_Execution_Limit_Status) {
	defer func() {
		Program_Execution_Limit_Status_Invariants(status, "evaluation_complete.status")
	}()
	Evaluation_Frame_Pointer_Invariants(frame_state, "evaluation_complete.frame_state")
	Workspace_Pointer_Invariants(workspace_state, "evaluation_complete.workspace_state")
	Program_Validated_Invariants(program_value, "evaluation_complete.program_value")
	var state Pipeline_State
	defer func() { Pipeline_State_Invariants(state, "evaluation_complete.state") }()
	frame := (Evaluation_Frame_Pointer)(frame_state)
	if frame.Control.Present != EVALUATION_CONTROL_TRUE {
		return Pipeline_State{}, PROGRAM_EXECUTION_LIMIT_STATUS_PROGRAM_INVALID
	}
	state = Pipeline_State(frame.States.Value)
	for index := DECLARATION_COUNT_NONE; index < state.Declaration_Count; index++ {
		bind_status := variable_bind_reference(
			(Workspace_Pointer)(workspace_state), program_value,
			declaration_reference(state.Declarations, Declaration_Index(index)),
			state.Value,
			state.Assignment,
		)
		status = Program_Execution_Limit_Status(bind_status)
		if bind_status != EXECUTION_LIMIT_STATUS_OK {
			state = Pipeline_State{}
			return state, status
		}
	}
	return state, PROGRAM_EXECUTION_LIMIT_STATUS_OK
}

func term_scalar_evaluate(
	workspace_state Workspace_Pointer, program_value Program_Validated,
	location_value Node_Location_Validated,
	dot_value Value,
	function_values Functions_Validated,
) (_ Value, status Term_Scalar_Status) {
	defer func() { Term_Scalar_Status_Invariants(status, "term_scalar_evaluate.status") }()
	Workspace_Pointer_Invariants(workspace_state, "term_scalar_evaluate.workspace_state")
	Program_Validated_Invariants(program_value, "term_scalar_evaluate.program_value")
	Node_Location_Validated_Invariants(
		location_value, "term_scalar_evaluate.location_value",
	)
	Value_Invariants(dot_value, "term_scalar_evaluate.dot_value")
	Functions_Validated_Invariants(function_values, "term_scalar_evaluate.function_values")
	var result Value
	defer func() { Value_Invariants(result, "term_scalar_evaluate.result") }()
	workspace := (Workspace_Pointer)(workspace_state)
	program := program_value
	location := location_value.Value
	node_value := Node_Validated{Value: location.Node}
	node := location.Node.(*Node)
	dot := Value(dot_value)
	functions := function_values
	switch node.Kind {
	case NODE_DOT:
		result = dot
	case NODE_BOOLEAN:
		data := program.Source.Data.Value[node.Value_Start:node.Value_End]
		result = Value_Of_Boolean(
			Boolean(bytes_equal(Text(data), Text("true"))),
		)
	case NODE_NIL:
		result = Value_Nil()
	case NODE_STRING:
		var literal_status Program_Status
		result, literal_status = literal_value(
			workspace, program_value, location_value,
		)
		return result, Term_Scalar_Status(literal_status)
	case NODE_NUMBER:
		var number_status Number_Status
		result, number_status = number_value(
			workspace, program_value, location_value,
		)
		return result, Term_Scalar_Status(number_status)
	case NODE_FIELD:
		var access_status Value_Access_Status
		result, access_status = field_value(program_value, node_value, dot)
		return result, Term_Scalar_Status(access_status)
	case NODE_VARIABLE:
		var variable_status Variable_Status
		result, variable_status = variable_value(
			workspace, program_value, node_value,
		)
		return result, Term_Scalar_Status(variable_status)
	case NODE_IDENTIFIER:
		var lookup_status Function_Lookup_Status
		result, lookup_status = function_value(program_value, node_value, functions)
		return result, Term_Scalar_Status(lookup_status)
	default:
		return result, TERM_SCALAR_STATUS_PROGRAM_INVALID
	}
	return result, TERM_SCALAR_STATUS_OK
}

func literal_value(
	workspace_state Workspace_Pointer, program_value Program_Validated,
	location_value Node_Location_Validated,
) (_ Value, status Program_Status) {
	defer func() { Program_Status_Invariants(status, "literal_value.status") }()
	Workspace_Pointer_Invariants(workspace_state, "literal_value.workspace_state")
	Program_Validated_Invariants(program_value, "literal_value.program_value")
	Node_Location_Validated_Invariants(location_value, "literal_value.location_value")
	var result Value
	defer func() { Value_Invariants(result, "literal_value.result") }()
	workspace := (Workspace_Pointer)(workspace_state)
	program := program_value
	location := location_value.Value
	reference, node := location.Reference.(Node_Reference), location.Node.(*Node)
	index := int(reference) - NODE_COUNT_INCREMENT
	if bool(workspace.Literal_Decoded[index]) {
		offset := workspace.Literal_Offsets[index]
		count := workspace.Literal_Counts[index]
		literal := workspace.Literals[offset:]
		result = Value_Of_Text(Text(literal[:count]))
		return result, PROGRAM_STATUS_OK
	}
	decoded, decode_status := Quoted_Unquote_Into(
		Quoted_Output(workspace.Name_Left), program.Source,
		Quoted_Span_Unvalidated{
			Start: Quoted_Start_Unvalidated(node.Value_Start),
			End:   Quoted_End_Unvalidated(node.Value_End),
		},
	)
	if decode_status != PARSE_STATUS_OK {
		return Value{}, PROGRAM_STATUS_INVALID
	}
	literal_count := workspace.Literal_Count.Value.(Workspace_Literal_Count_Storage_Value)
	if int(literal_count)+int(decoded) > len(workspace.Literals) {
		return Value{}, PROGRAM_STATUS_INVALID
	}
	offset := literal_count
	copy(workspace.Literals[offset:], workspace.Name_Left[:decoded])
	workspace.Literal_Offsets[index] = Literal_Count(offset)
	workspace.Literal_Counts[index] = Literal_Count(decoded)
	workspace.Literal_Decoded[index] = true
	workspace.Literal_Count.Value = literal_count +
		Workspace_Literal_Count_Storage_Value(decoded)
	result = Value_Of_Text(Text(
		workspace.Literals[offset : offset+Workspace_Literal_Count_Storage_Value(decoded)],
	))
	return result, PROGRAM_STATUS_OK
}

func number_value(
	workspace_state Workspace_Pointer, program_value Program_Validated,
	location_value Node_Location_Validated,
) (_ Value, status Number_Status) {
	defer func() { Number_Status_Invariants(status, "number_value.status") }()
	Workspace_Pointer_Invariants(workspace_state, "number_value.workspace_state")
	Program_Validated_Invariants(program_value, "number_value.program_value")
	Node_Location_Validated_Invariants(location_value, "number_value.location_value")
	var result Value
	defer func() { Value_Invariants(result, "number_value.result") }()
	workspace := (Workspace_Pointer)(workspace_state)
	program := program_value
	location := location_value.Value
	node := location.Node.(*Node)
	data := program.Source.Data.Value[node.Value_Start:node.Value_End]
	number_text := Number_Text_Validated{Value: Number_Text_Validated_Value(Text(data))}
	if len(data) == SOURCE_SIZE_MINIMUM {
		return Value{}, NUMBER_STATUS_PROGRAM_INVALID
	}
	if data[SOURCE_SIZE_MINIMUM] == '\'' {
		decoded, decode_status := literal_value(
			workspace, program_value, location_value,
		)
		text := Text(decoded.Text.Value)
		if decode_status != PROGRAM_STATUS_OK {
			return Value{}, NUMBER_STATUS_PROGRAM_INVALID
		}
		if len(text) == SOURCE_SIZE_MINIMUM {
			return Value{}, NUMBER_STATUS_PROGRAM_INVALID
		}
		character, size := utf8.Decode_Character(utf8.Bytes(text))
		if int(size) != len(text) {
			return Value{}, NUMBER_STATUS_PROGRAM_INVALID
		}
		result = Value_Of_Integer(Integer(character))
		return result, NUMBER_STATUS_OK
	}
	if data[len(data)-SOURCE_POSITION_INCREMENT] == 'i' {
		value, valid := complex_parse(
			workspace, Number_Text_Validated{Value: Number_Text_Validated_Value(
				Text(data[:len(data)-SOURCE_POSITION_INCREMENT]),
			)},
		)
		if !bool(valid) {
			return Value{}, NUMBER_STATUS_VALUE_INVALID
		}
		result = value
		return result, NUMBER_STATUS_OK
	}
	if bool(number_float_syntax(number_text)) {
		encoding_value, valid := float_bits_parse(workspace, number_text)
		if !valid {
			return Value{}, NUMBER_STATUS_VALUE_INVALID
		}
		result = Value_Of_Float(
			Float(encoding_value.Value.(Parsed_Float_Validated_Value)),
		)
		return result, NUMBER_STATUS_OK
	}
	value, valid := integer_parse(number_text)
	if !bool(valid) {
		return Value{}, NUMBER_STATUS_VALUE_INVALID
	}
	result = value
	return result, NUMBER_STATUS_OK
}

func number_float_syntax(data_value Number_Text_Validated) (found Number_Float_Syntax) {
	defer func() {
		Number_Float_Syntax_Invariants(found, "number_float_syntax.found")
	}()
	Number_Text_Validated_Invariants(data_value, "number_float_syntax.data_value")
	data := data_value.Value.(Number_Text_Validated_Value)
	position_index := SOURCE_SIZE_MINIMUM
	if data[position_index] == '+' {
		position_index++
	} else if data[position_index] == '-' {
		position_index++
	}
	hexadecimal := false
	if position_index < len(data)-SOURCE_POSITION_INCREMENT {
		if data[position_index] == '0' {
			next := data[position_index+SOURCE_POSITION_INCREMENT]
			if next == 'x' {
				hexadecimal = true
			} else if next == 'X' {
				hexadecimal = true
			}
		}
	}
	for ; position_index < len(data); position_index++ {
		switch data[position_index] {
		case '.', 'p', 'P':
			return true
		case 'e', 'E':
			if !hexadecimal {
				return true
			}
		}
	}
	return false
}

func complex_parse(
	workspace_state Workspace_Pointer, data_value Number_Text_Validated,
) (_ Value, valid Value_Validity) {
	defer func() { Value_Validity_Invariants(valid, "complex_parse.valid") }()
	Workspace_Pointer_Invariants(workspace_state, "complex_parse.workspace_state")
	Number_Text_Validated_Invariants(data_value, "complex_parse.data_value")
	var result Value
	defer func() { Value_Invariants(result, "complex_parse.result") }()
	workspace := (Workspace_Pointer)(workspace_state)
	data := data_value.Value.(Number_Text_Validated_Value)
	if len(data) == SOURCE_SIZE_MINIMUM {
		return Value{}, false
	}
	separator := SOURCE_SIZE_MINIMUM
	for position := SOURCE_POSITION_INCREMENT; position < len(data); position++ {
		if data[position] != '+' {
			if data[position] != '-' {
				continue
			}
		}
		previous := data[position-SOURCE_POSITION_INCREMENT]
		switch previous {
		case 'e', 'E', 'p', 'P':
			continue
		}
		separator = position
		break
	}
	real := Parsed_Float_Validated{
		Value: Parsed_Float_Validated_Value(Float(bits.WORD_64_MINIMUM)),
	}
	if separator != SOURCE_SIZE_MINIMUM {
		var component_valid Value_Validity
		real, component_valid = float_bits_parse(
			workspace, Number_Text_Validated{
				Value: Number_Text_Validated_Value(Text(data[:separator])),
			},
		)
		if !component_valid {
			return Value{}, false
		}
	}
	imaginary, component_valid := float_bits_parse(
		workspace, Number_Text_Validated{
			Value: Number_Text_Validated_Value(Text(data[separator:])),
		},
	)
	if !component_valid {
		return Value{}, false
	}
	result = Value_Of_Complex(Complex{
		Real: Complex_Real(real.Value.(Parsed_Float_Validated_Value)),
		Imaginary: Complex_Imaginary(
			imaginary.Value.(Parsed_Float_Validated_Value),
		),
	})
	return result, true
}

func float_bits_parse(
	workspace_state Workspace_Pointer, data_value Number_Text_Validated,
) (_ Parsed_Float_Validated, valid Value_Validity) {
	defer func() { Value_Validity_Invariants(valid, "float_bits_parse.valid") }()
	Workspace_Pointer_Invariants(workspace_state, "float_bits_parse.workspace_state")
	Number_Text_Validated_Invariants(data_value, "float_bits_parse.data_value")
	var encoding Parsed_Float_Validated
	defer func() { Parsed_Float_Validated_Invariants(encoding, "float_bits_parse.encoding") }()
	workspace := (Workspace_Pointer)(workspace_state)
	data := data_value.Value.(Number_Text_Validated_Value)
	float_value := &workspace.Float_Parse.Values[big.FLOAT_RAT_RESULT_INDEX]
	float_parse := (*big.Float_Parse_Workspace)(&workspace.Float_Parse)
	// Reset semantics must not discard caller-owned bounded mantissa storage.
	words := float_value.Mantissa.Words
	*float_value = big.Float{}
	float_value.Mantissa.Words = words
	if big.Float_Set_Precision(
		float_value, big.FLOAT_64_VALUE_MANTISSA_BIT_COUNT,
	) != big.STATUS_OK {
		return Parsed_Float_Validated{}, false
	}
	_, status := big.Float_Parse(
		float_value, big.Float_Parse_Text_Unvalidated(data),
		big.BASE_AUTOMATIC, float_parse,
	)
	if status != big.STATUS_OK {
		return Parsed_Float_Validated{}, false
	}
	value, _ := big.Float_Float_64_Bits(float_value)
	encoding = Parsed_Float_Validated{Value: Parsed_Float_Validated_Value(Float(value))}
	return encoding, true
}

func integer_parse(data_value Number_Text_Validated) (_ Value, valid Value_Validity) {
	defer func() { Value_Validity_Invariants(valid, "integer_parse.valid") }()
	Number_Text_Validated_Invariants(data_value, "integer_parse.data_value")
	var result Value
	defer func() { Value_Invariants(result, "integer_parse.result") }()
	data := data_value.Value.(Number_Text_Validated_Value)
	position, negative := SOURCE_SIZE_MINIMUM, false
	if data[position] == '+' {
		position++
	} else if data[position] == '-' {
		negative = true
		position++
	}
	base := uint64(big.BASE_DECIMAL)
	if position+SOURCE_POSITION_INCREMENT < len(data) {
		if data[position] == '0' {
			switch data[position+SOURCE_POSITION_INCREMENT] {
			case 'b', 'B':
				base = uint64(big.BASE_BINARY)
				position += NUMBER_PREFIX_SIZE
			case 'o', 'O':
				base = uint64(big.BASE_OCTAL)
				position += NUMBER_PREFIX_SIZE
			case 'x', 'X':
				base = uint64(big.BASE_HEXADECIMAL)
				position += NUMBER_PREFIX_SIZE
			default:
				if len(data)-position > SOURCE_POSITION_INCREMENT {
					base = uint64(big.BASE_OCTAL)
				}
			}
		}
	}
	magnitude := uint64(bits.WORD_64_MINIMUM)
	for ; position < len(data); position++ {
		character := data[position]
		if character == '_' {
			continue
		}
		digit := uint64(character - '0')
		if character >= 'a' {
			digit = uint64(character-'a') + big.BASE_DECIMAL
		} else if character >= 'A' {
			digit = uint64(character-'A') + big.BASE_DECIMAL
		}
		if digit >= base {
			return Value{}, false
		}
		if magnitude > (bits.WORD_64_MAXIMUM-digit)/base {
			return Value{}, false
		}
		magnitude = magnitude*base + digit
	}
	if negative {
		if magnitude > INTEGER_64_NEGATIVE_MAGNITUDE_MAXIMUM {
			return Value{}, false
		}
		if magnitude == INTEGER_64_NEGATIVE_MAGNITUDE_MAXIMUM {
			result = Value_Of_Integer(Integer(bits.INTEGER_64_MINIMUM))
			return result, true
		}
		result = Value_Of_Integer(Integer(-int64(magnitude)))
		return result, true
	}
	if magnitude > uint64(bits.INTEGER_MAXIMUM) {
		return Value{}, false
	}
	result = Value_Of_Integer(Integer(magnitude))
	return result, true
}

func field_value(
	program_value Program_Validated, node_value Node_Validated, value_state Value,
) (
	_ Value,
	status Value_Access_Status,
) {
	defer func() { Value_Access_Status_Invariants(status, "field_value.status") }()
	Program_Validated_Invariants(program_value, "field_value.program_value")
	Node_Validated_Invariants(node_value, "field_value.node_value")
	Value_Invariants(value_state, "field_value.value_state")
	var result Value
	defer func() { Value_Invariants(result, "field_value.result") }()
	program := program_value
	node := node_value.Value.(*Node)
	value := Value(value_state)
	path := program.Source.Data.Value[node.Value_Start:node.Value_End]
	result, status = field_path(Text(path), value)
	return result, status
}

func field_path(
	path Text, value_state Value,
) (_ Value, status Value_Access_Status) {
	defer func() { Value_Access_Status_Invariants(status, "field_path.status") }()
	Text_Invariants(path, "field_path.path")
	Value_Invariants(value_state, "field_path.value_state")
	var result Value
	defer func() { Value_Invariants(result, "field_path.result") }()
	value := Value(value_state)
	if len(path) < NUMBER_PREFIX_SIZE {
		return Value{}, VALUE_ACCESS_STATUS_PROGRAM_INVALID
	}
	if path[SOURCE_SIZE_MINIMUM] != '.' {
		return Value{}, VALUE_ACCESS_STATUS_PROGRAM_INVALID
	}
	position := SOURCE_POSITION_INCREMENT
	current := value
	for position < len(path) {
		end := position
		for end < len(path) {
			if path[end] == '.' {
				break
			}
			end++
		}
		if end == position {
			return Value{}, VALUE_ACCESS_STATUS_PROGRAM_INVALID
		}
		kind := Value_Kind(current.Kind.Value)
		if kind != VALUE_OBJECT {
			if kind != VALUE_MAP {
				return Value{}, VALUE_ACCESS_STATUS_RECEIVER_INVALID
			}
		}
		found := false
		fields := Fields(current.Fields.Value)
		for index := range fields {
			field := fields[index]
			if kind == VALUE_OBJECT {
				if bytes_equal(path[position:end], Text(field.Name)) {
					current = field.Value.Data
					found = true
					break
				}
			} else if Value_Kind(field.Key.Data.Kind.Value) == VALUE_TEXT {
				key_text := Text(field.Key.Data.Text.Value)
				if bytes_equal(
					path[position:end], key_text,
				) {
					current = field.Value.Data
					found = true
					break
				}
			}
		}
		if !found {
			result = Value_Nil()
			return result, VALUE_ACCESS_STATUS_OK
		}
		position = end + SOURCE_POSITION_INCREMENT
	}
	result = current
	return result, VALUE_ACCESS_STATUS_OK
}

func variable_value(
	workspace_state Workspace_Pointer, program_value Program_Validated,
	node_value Node_Validated,
) (_ Value, status Variable_Status) {
	defer func() { Variable_Status_Invariants(status, "variable_value.status") }()
	Workspace_Pointer_Invariants(workspace_state, "variable_value.workspace_state")
	Program_Validated_Invariants(program_value, "variable_value.program_value")
	Node_Validated_Invariants(node_value, "variable_value.node_value")
	var result Value
	defer func() { Value_Invariants(result, "variable_value.result") }()
	workspace := (Workspace_Pointer)(workspace_state)
	program := program_value
	node := node_value.Value.(*Node)
	data := program.Source.Data.Value[node.Value_Start:node.Value_End]
	name_end := SOURCE_SIZE_MINIMUM
	for name_end < len(data) && data[name_end] != '.' {
		name_end++
	}
	index := variable_find(workspace, program_value, Text(data[:name_end]))
	if index == VARIABLE_INDEX_ABSENT {
		return Value{}, VARIABLE_STATUS_ABSENT
	}
	result = workspace.Variables[index-VARIABLE_COUNT_INCREMENT].Value
	if name_end < len(data) {
		access_result, access_status := field_path(Text(data[name_end:]), result)
		if access_status == VALUE_ACCESS_STATUS_PROGRAM_INVALID {
			return Value{}, VARIABLE_STATUS_PROGRAM_INVALID
		}
		if access_status == VALUE_ACCESS_STATUS_RECEIVER_INVALID {
			return Value{}, VARIABLE_STATUS_ABSENT
		}
		result = access_result
		return result, VARIABLE_STATUS_OK
	}
	return result, VARIABLE_STATUS_OK
}

func variable_find(workspace_state Workspace_Pointer, program_value Program_Validated, name Text) (
	result Variable_Index,
) {
	defer func() { Variable_Index_Invariants(result, "variable_find.result") }()
	Workspace_Pointer_Invariants(workspace_state, "variable_find.workspace_state")
	Program_Validated_Invariants(program_value, "variable_find.program_value")
	Text_Invariants(name, "variable_find.name")
	workspace := (Workspace_Pointer)(workspace_state)
	program := program_value
	data := program.Source.Data.Value
	variable_count := workspace.Variable_Count.Value.(Workspace_Variable_Count_Storage_Value)
	scope_base := workspace.Scope_Base.Value.(Workspace_Scope_Base_Storage_Value)
	minimum := int(scope_base)
	for index := int(variable_count) - VARIABLE_COUNT_INCREMENT; index >= minimum; index-- {
		variable := workspace.Variables[index]
		if bool(variable.Root) {
			if len(name) == VARIABLE_NAME_SIZE_MINIMUM {
				if name[SOURCE_SIZE_MINIMUM] == '$' {
					return Variable_Index(index + VARIABLE_COUNT_INCREMENT)
				}
			}
			continue
		}
		if bytes_equal(name, Text(data[variable.Start:variable.End])) {
			return Variable_Index(index + VARIABLE_COUNT_INCREMENT)
		}
	}
	return VARIABLE_INDEX_ABSENT
}

func variable_bind_reference(
	workspace_state Workspace_Pointer, program_value Program_Validated,
	reference_value Declaration_Reference_Validated,
	value_state Value,
	assignment_value Assignment,
) (status Execution_Limit_Status) {
	defer func() {
		Execution_Limit_Status_Invariants(status, "variable_bind_reference.status")
	}()
	Workspace_Pointer_Invariants(workspace_state, "variable_bind_reference.workspace_state")
	Program_Validated_Invariants(program_value, "variable_bind_reference.program_value")
	Declaration_Reference_Validated_Invariants(
		reference_value, "variable_bind_reference.reference_value",
	)
	Value_Invariants(value_state, "variable_bind_reference.value_state")
	Assignment_Invariants(assignment_value, "variable_bind_reference.assignment_value")
	workspace := (Workspace_Pointer)(workspace_state)
	program := program_value
	reference := reference_value.Value.(Declaration_Reference_Validated_Value)
	value := Value(value_state)
	assignment := Assignment(assignment_value)
	node_value, found := program_node(
		program_value, Node_Link_Validated{Value: Node_Link_Validated_Value(reference)},
	)
	aver.Always(
		bool(found),
		"Evaluation retained the declaration node before binding its value.",
	)
	if !bool(found) {
		return EXECUTION_LIMIT_STATUS_EXECUTION_INVALID
	}
	node := node_value.Value.(*Node)
	aver.Always(
		node.Kind == NODE_VARIABLE,
		"Evaluation retains only variable nodes as declarations.",
	)
	if node.Kind != NODE_VARIABLE {
		return EXECUTION_LIMIT_STATUS_EXECUTION_INVALID
	}
	name := program.Source.Data.Value[node.Value_Start:node.Value_End]
	if bool(assignment) {
		index := variable_find(workspace, program_value, Text(name))
		if index == VARIABLE_INDEX_ABSENT {
			return EXECUTION_LIMIT_STATUS_EXECUTION_INVALID
		}
		workspace.Variables[index-VARIABLE_COUNT_INCREMENT].Value = value
		return EXECUTION_LIMIT_STATUS_OK
	}
	variable_count := workspace.Variable_Count.Value.(Workspace_Variable_Count_Storage_Value)
	if int(variable_count) == len(workspace.Variables) {
		return EXECUTION_LIMIT_STATUS_LIMIT_EXCEEDED
	}
	workspace.Variables[variable_count] = Variable{
		Start: node.Value_Start, End: node.Value_End, Value: value,
	}
	workspace.Variable_Count.Value = variable_count +
		Workspace_Variable_Count_Storage_Value(VARIABLE_COUNT_INCREMENT)
	return EXECUTION_LIMIT_STATUS_OK
}

func function_value(
	program_value Program_Validated,
	node_value Node_Validated,
	function_values Functions_Validated,
) (_ Value, status Function_Lookup_Status) {
	defer func() { Function_Lookup_Status_Invariants(status, "function_value.status") }()
	Program_Validated_Invariants(program_value, "function_value.program_value")
	Node_Validated_Invariants(node_value, "function_value.node_value")
	Functions_Validated_Invariants(function_values, "function_value.function_values")
	var result Value
	defer func() { Value_Invariants(result, "function_value.result") }()
	program := program_value
	node := node_value.Value.(*Node)
	functions := function_values.Value.(Functions_Validated_Value)
	name := program.Source.Data.Value[node.Value_Start:node.Value_End]
	for index := range functions {
		if bytes_equal(Text(name), Text(functions[index].Name)) {
			result = Value_Of_Function(functions[index].Call)
			return result, FUNCTION_LOOKUP_STATUS_OK
		}
	}
	return Value{}, FUNCTION_LOOKUP_STATUS_ABSENT
}

func function_call_identifier(
	workspace_state Workspace_Pointer, program_value Program_Validated,
	identifier_value Node_Validated,
	function_values Functions_Validated,
	argument_values Arguments_Staged,
) (_ Value, status Call_Status) {
	defer func() { Call_Status_Invariants(status, "function_call_identifier.status") }()
	Workspace_Pointer_Invariants(workspace_state, "function_call_identifier.workspace_state")
	Program_Validated_Invariants(program_value, "function_call_identifier.program_value")
	Node_Validated_Invariants(
		identifier_value, "function_call_identifier.identifier_value",
	)
	Functions_Validated_Invariants(function_values, "function_call_identifier.function_values")
	Arguments_Staged_Invariants(argument_values, "function_call_identifier.argument_values")
	var result Value
	defer func() { Value_Invariants(result, "function_call_identifier.result") }()
	workspace := (Workspace_Pointer)(workspace_state)
	program := program_value
	identifier := identifier_value.Value.(*Node)
	functions := function_values.Value.(Functions_Validated_Value)
	arguments := argument_values
	name := program.Source.Data.Value[identifier.Value_Start:identifier.Value_End]
	for index := range functions {
		if bytes_equal(Text(name), Text(functions[index].Name)) {
			call_result, call_status := callback_call(
				workspace, functions[index].Call, arguments,
			)
			result, status = call_result, Call_Status(call_status)
			return result, status
		}
	}
	result, status = builtin_call(
		workspace, Function_Name_Validated{
			Value: Function_Name_Validated_Value(Function_Name(name)),
		}, arguments,
	)
	return result, status
}

func callback_call(
	workspace_state Workspace_Pointer, call_value Function_Call,
	argument_values Arguments_Staged,
) (
	_ Value,
	status Function_Call_Status,
) {
	defer func() { Function_Call_Status_Invariants(status, "callback_call.status") }()
	Workspace_Pointer_Invariants(workspace_state, "callback_call.workspace_state")
	Function_Call_Invariants(call_value, "callback_call.call_value")
	Arguments_Staged_Invariants(argument_values, "callback_call.argument_values")
	var result Value
	defer func() { Value_Invariants(result, "callback_call.result") }()
	workspace := (Workspace_Pointer)(workspace_state)
	call := Function_Call(call_value)
	arguments := Values(argument_values.Value.(Arguments_Staged_Value))
	output := call(Function_Input{Arguments: arguments})
	if output.Status != FUNCTION_STATUS_OK {
		return Value{}, FUNCTION_CALL_STATUS_INVALID
	}
	if !bool(value_active_valid(Value_Unvalidated(output.Value))) {
		return Value{}, FUNCTION_CALL_STATUS_INVALID
	}
	if value_graph_validate(
		workspace, Value_Unvalidated(output.Value),
	) != VALUE_GRAPH_STATUS_OK {
		return Value{}, FUNCTION_CALL_STATUS_INVALID
	}
	result = output.Value
	return result, FUNCTION_CALL_STATUS_OK
}

func builtin_call(
	workspace_state Workspace_Pointer, name_value Function_Name_Validated,
	argument_values Arguments_Staged,
) (_ Value, status Call_Status) {
	defer func() { Call_Status_Invariants(status, "builtin_call.status") }()
	Workspace_Pointer_Invariants(workspace_state, "builtin_call.workspace_state")
	Function_Name_Validated_Invariants(name_value, "builtin_call.name_value")
	Arguments_Staged_Invariants(argument_values, "builtin_call.argument_values")
	var result Value
	defer func() { Value_Invariants(result, "builtin_call.result") }()
	workspace, arguments := (Workspace_Pointer)(workspace_state), argument_values
	name := name_value.Value.(Function_Name_Validated_Value)
	switch {
	case bool(bytes_equal(Text(name), Text("and"))):
		result, argument_status := builtin_and(arguments)
		return result, Call_Status(argument_status)
	case bool(bytes_equal(Text(name), Text("or"))):
		result, argument_status := builtin_or(arguments)
		return result, Call_Status(argument_status)
	case bool(bytes_equal(Text(name), Text("call"))):
		result, value_status := builtin_call_function(workspace, arguments)
		return result, Call_Status(value_status)
	case bool(bytes_equal(Text(name), Text("index"))):
		result, value_status := builtin_index(arguments)
		return result, Call_Status(value_status)
	case bool(bytes_equal(Text(name), Text("slice"))):
		result, value_status := builtin_slice(arguments)
		return result, Call_Status(value_status)
	case bool(bytes_equal(Text(name), Text("eq"))):
		result, argument_status := builtin_equal(arguments)
		return result, Call_Status(argument_status)
	case bool(bytes_equal(Text(name), Text("ne"))):
		result, argument_status := builtin_not_equal(arguments)
		return result, Call_Status(argument_status)
	case bool(bytes_equal(Text(name), Text("lt"))),
		bool(bytes_equal(Text(name), Text("le"))),
		bool(bytes_equal(Text(name), Text("gt"))),
		bool(bytes_equal(Text(name), Text("ge"))):
		order_name := Builtin_Order_Name_Validated{
			First:  Builtin_Order_Name_First(name[BUILTIN_ARGUMENT_FIRST]),
			Second: Builtin_Order_Name_Second(name[BUILTIN_ARGUMENT_SECOND]),
		}
		kind := builtin_order_kind(order_name)
		result, value_status := builtin_order(kind, arguments)
		return result, Call_Status(value_status)
	case bool(bytes_equal(Text(name), Text("not"))):
		result, argument_status := builtin_not(arguments)
		return result, Call_Status(argument_status)
	case bool(bytes_equal(Text(name), Text("len"))):
		result, value_status := builtin_count(arguments)
		return result, Call_Status(value_status)
	case bool(bytes_equal(Text(name), Text("print"))):
		result, generated_status := builtin_print(workspace, arguments, false)
		return result, Call_Status(generated_status)
	case bool(bytes_equal(Text(name), Text("println"))):
		result, generated_status := builtin_print(workspace, arguments, true)
		return result, Call_Status(generated_status)
	case bool(bytes_equal(Text(name), Text("printf"))):
		result, formatting_status := builtin_printf(workspace, arguments)
		return result, Call_Status(formatting_status)
	case bool(bytes_equal(Text(name), Text("html"))):
		result, generated_status := builtin_escape(workspace, arguments, ESCAPE_HTML)
		return result, Call_Status(generated_status)
	case bool(bytes_equal(Text(name), Text("js"))):
		result, generated_status := builtin_escape(workspace, arguments, ESCAPE_JS)
		return result, Call_Status(generated_status)
	case bool(bytes_equal(Text(name), Text("urlquery"))):
		result, generated_status := builtin_escape(workspace, arguments, ESCAPE_URL_QUERY)
		return result, Call_Status(generated_status)
	}
	return Value{}, CALL_STATUS_FUNCTION_ABSENT
}

func builtin_and(argument_values Arguments_Staged) (
	_ Value,
	status Builtin_Argument_Status,
) {
	defer func() { Builtin_Argument_Status_Invariants(status, "builtin_and.status") }()
	Arguments_Staged_Invariants(argument_values, "builtin_and.argument_values")
	var result Value
	defer func() { Value_Invariants(result, "builtin_and.result") }()
	arguments := argument_values.Value.(Arguments_Staged_Value)
	if len(arguments) == BUILTIN_ARGUMENT_COUNT_NONE {
		return Value{}, BUILTIN_ARGUMENT_STATUS_INVALID
	}
	for index := range arguments {
		result = arguments[index]
		if !bool(value_truth(result)) {
			return result, BUILTIN_ARGUMENT_STATUS_OK
		}
	}
	return result, BUILTIN_ARGUMENT_STATUS_OK
}

func builtin_or(argument_values Arguments_Staged) (
	_ Value,
	status Builtin_Argument_Status,
) {
	defer func() { Builtin_Argument_Status_Invariants(status, "builtin_or.status") }()
	Arguments_Staged_Invariants(argument_values, "builtin_or.argument_values")
	var result Value
	defer func() { Value_Invariants(result, "builtin_or.result") }()
	arguments := argument_values.Value.(Arguments_Staged_Value)
	if len(arguments) == BUILTIN_ARGUMENT_COUNT_NONE {
		return Value{}, BUILTIN_ARGUMENT_STATUS_INVALID
	}
	for index := range arguments {
		result = arguments[index]
		if bool(value_truth(result)) {
			return result, BUILTIN_ARGUMENT_STATUS_OK
		}
	}
	return result, BUILTIN_ARGUMENT_STATUS_OK
}

func builtin_call_function(
	workspace_state Workspace_Pointer, argument_values Arguments_Staged,
) (
	_ Value,
	status Builtin_Value_Status,
) {
	defer func() { Builtin_Value_Status_Invariants(status, "builtin_call_function.status") }()
	Workspace_Pointer_Invariants(workspace_state, "builtin_call_function.workspace_state")
	Arguments_Staged_Invariants(argument_values, "builtin_call_function.argument_values")
	var result Value
	defer func() { Value_Invariants(result, "builtin_call_function.result") }()
	workspace := (Workspace_Pointer)(workspace_state)
	arguments := argument_values.Value.(Arguments_Staged_Value)
	if len(arguments) == BUILTIN_ARGUMENT_COUNT_NONE {
		return Value{}, BUILTIN_VALUE_STATUS_FUNCTION_INVALID
	}
	function := arguments[BUILTIN_ARGUMENT_FIRST]
	if Value_Kind(function.Kind.Value) != VALUE_FUNCTION {
		return Value{}, BUILTIN_VALUE_STATUS_KIND_INVALID
	}
	call := Function_Call(function.Function.Value)
	if call == nil {
		return Value{}, BUILTIN_VALUE_STATUS_KIND_INVALID
	}
	callback_arguments := Arguments_Staged{Value: Arguments_Staged_Value(
		Values(arguments[BUILTIN_ARGUMENT_COUNT_ONE:]),
	)}

	result, call_status := callback_call(workspace, call, callback_arguments)
	if call_status != FUNCTION_CALL_STATUS_OK {
		return Value{}, BUILTIN_VALUE_STATUS_FUNCTION_INVALID
	}
	return result, BUILTIN_VALUE_STATUS_OK
}

func builtin_equal(argument_values Arguments_Staged) (
	_ Value,
	status Builtin_Argument_Status,
) {
	defer func() { Builtin_Argument_Status_Invariants(status, "builtin_equal.status") }()
	Arguments_Staged_Invariants(argument_values, "builtin_equal.argument_values")
	var result Value
	defer func() { Value_Invariants(result, "builtin_equal.result") }()
	arguments := argument_values.Value.(Arguments_Staged_Value)
	if len(arguments) < BUILTIN_ARGUMENT_COUNT_TWO {
		return Value{}, BUILTIN_ARGUMENT_STATUS_INVALID
	}
	equal := false
	for index := BUILTIN_ARGUMENT_SECOND; index < len(arguments); index++ {
		if bool(values_equal(arguments[BUILTIN_ARGUMENT_FIRST], arguments[index])) {
			equal = true
			break
		}
	}
	result = Value_Of_Boolean(Boolean(equal))
	return result, BUILTIN_ARGUMENT_STATUS_OK
}

func builtin_not_equal(argument_values Arguments_Staged) (
	_ Value,
	status Builtin_Argument_Status,
) {
	defer func() { Builtin_Argument_Status_Invariants(status, "builtin_not_equal.status") }()
	Arguments_Staged_Invariants(argument_values, "builtin_not_equal.argument_values")
	var result Value
	defer func() { Value_Invariants(result, "builtin_not_equal.result") }()
	arguments := argument_values.Value.(Arguments_Staged_Value)
	if len(arguments) != BUILTIN_ARGUMENT_COUNT_TWO {
		return Value{}, BUILTIN_ARGUMENT_STATUS_INVALID
	}
	equal := bool(values_equal(
		arguments[BUILTIN_ARGUMENT_FIRST], arguments[BUILTIN_ARGUMENT_SECOND],
	))
	result = Value_Of_Boolean(Boolean(!equal))
	return result, BUILTIN_ARGUMENT_STATUS_OK
}

func builtin_not(argument_values Arguments_Staged) (
	_ Value,
	status Builtin_Argument_Status,
) {
	defer func() { Builtin_Argument_Status_Invariants(status, "builtin_not.status") }()
	Arguments_Staged_Invariants(argument_values, "builtin_not.argument_values")
	var result Value
	defer func() { Value_Invariants(result, "builtin_not.result") }()
	arguments := argument_values.Value.(Arguments_Staged_Value)
	if len(arguments) != BUILTIN_ARGUMENT_COUNT_ONE {
		return Value{}, BUILTIN_ARGUMENT_STATUS_INVALID
	}
	truth := bool(value_truth(arguments[BUILTIN_ARGUMENT_FIRST]))
	result = Value_Of_Boolean(Boolean(!truth))
	return result, BUILTIN_ARGUMENT_STATUS_OK
}

func builtin_count(argument_values Arguments_Staged) (
	_ Value,
	status Builtin_Value_Status,
) {
	defer func() { Builtin_Value_Status_Invariants(status, "builtin_count.status") }()
	Arguments_Staged_Invariants(argument_values, "builtin_count.argument_values")
	var result Value
	defer func() { Value_Invariants(result, "builtin_count.result") }()
	arguments := argument_values.Value.(Arguments_Staged_Value)
	if len(arguments) != BUILTIN_ARGUMENT_COUNT_ONE {
		return Value{}, BUILTIN_VALUE_STATUS_FUNCTION_INVALID
	}
	value := arguments[BUILTIN_ARGUMENT_FIRST]
	switch Value_Kind(value.Kind.Value) {
	case VALUE_TEXT:
		result = Value_Of_Integer(Integer(len(Text(value.Text.Value))))
	case VALUE_SEQUENCE:
		result = Value_Of_Integer(Integer(len(Values(value.Values.Value))))
	case VALUE_OBJECT, VALUE_MAP:
		result = Value_Of_Integer(Integer(len(Fields(value.Fields.Value))))
	default:
		return result, BUILTIN_VALUE_STATUS_KIND_INVALID
	}
	return result, BUILTIN_VALUE_STATUS_OK
}

func builtin_print(
	workspace_state Workspace_Pointer, argument_values Arguments_Staged,
	newline Boolean,
) (_ Value, status Generated_Status) {
	defer func() { Generated_Status_Invariants(status, "builtin_print.status") }()
	Workspace_Pointer_Invariants(workspace_state, "builtin_print.workspace_state")
	Arguments_Staged_Invariants(argument_values, "builtin_print.argument_values")
	Boolean_Invariants(newline, "builtin_print.newline")
	var result Value
	defer func() { Value_Invariants(result, "builtin_print.result") }()
	workspace := (Workspace_Pointer)(workspace_state)
	arguments := argument_values.Value.(Arguments_Staged_Value)
	start := Generated_Start_Validated{
		Value: Generated_Start_Validated_Value(
			workspace.Generated_Count.Value.(Workspace_Generated_Count_Storage_Value),
		),
	}

	for argument_index := range arguments {
		if argument_index > BUILTIN_ARGUMENT_FIRST {
			separator_required := bool(newline)
			if !separator_required {
				previous := arguments[argument_index-ARGUMENT_COUNT_INCREMENT]
				previous_kind := Value_Kind(previous.Kind.Value)
				current_kind := Value_Kind(arguments[argument_index].Kind.Value)
				separator_required = previous_kind != VALUE_TEXT
				if current_kind == VALUE_TEXT {
					separator_required = false
				}
			}
			if separator_required {
				if generated_append_text(workspace, " ") != GENERATED_STATUS_OK {
					return Value{}, GENERATED_STATUS_LIMIT_EXCEEDED
				}
			}
		}
		if Value_Kind(arguments[argument_index].Kind.Value) == VALUE_NIL {
			if generated_append_text(workspace, "<nil>") != GENERATED_STATUS_OK {
				return Value{}, GENERATED_STATUS_LIMIT_EXCEEDED
			}
			continue
		}
		if generated_append_value(
			workspace, arguments[argument_index],
		) != GENERATED_STATUS_OK {
			return Value{}, GENERATED_STATUS_LIMIT_EXCEEDED
		}
	}
	if bool(newline) {
		if generated_append_text(workspace, "\n") != GENERATED_STATUS_OK {
			return Value{}, GENERATED_STATUS_LIMIT_EXCEEDED
		}
	}
	result, generated_status := generated_text_value(workspace, start)
	return result, generated_status
}

func builtin_printf(workspace_state Workspace_Pointer, argument_values Arguments_Staged) (
	_ Value,
	status Formatting_Status,
) {
	defer func() { Formatting_Status_Invariants(status, "builtin_printf.status") }()
	Workspace_Pointer_Invariants(workspace_state, "builtin_printf.workspace_state")
	Arguments_Staged_Invariants(argument_values, "builtin_printf.argument_values")
	var result Value
	defer func() { Value_Invariants(result, "builtin_printf.result") }()
	workspace := (Workspace_Pointer)(workspace_state)
	arguments := argument_values.Value.(Arguments_Staged_Value)
	if len(arguments) == BUILTIN_ARGUMENT_COUNT_NONE {
		return Value{}, FORMATTING_STATUS_INVALID
	}
	format, text_status := Value_As_Text(arguments[BUILTIN_ARGUMENT_FIRST])
	if text_status != VALUE_TEXT_STATUS_OK {
		return Value{}, FORMATTING_STATUS_INVALID
	}
	start := Generated_Start_Validated{
		Value: Generated_Start_Validated_Value(
			workspace.Generated_Count.Value.(Workspace_Generated_Count_Storage_Value),
		),
	}

	argument_index := BUILTIN_ARGUMENT_SECOND
	for format_index := SOURCE_SIZE_MINIMUM; format_index < len(format); format_index++ {
		if format[format_index] != '%' {
			character := format[format_index : format_index+SOURCE_POSITION_INCREMENT]
			if generated_append_bytes(
				workspace, character,
			) != GENERATED_STATUS_OK {
				return Value{}, FORMATTING_STATUS_LIMIT_EXCEEDED
			}
			continue
		}
		verb, width, zero_padding, alternate, consumed, directive_status :=
			format_directive_parse(
				Format_Source(format[format_index+SOURCE_POSITION_INCREMENT:]),
			)
		if directive_status != FORMAT_STATUS_OK {
			return Value{}, Formatting_Status(directive_status)
		}
		format_index += int(consumed)
		if verb == '%' {
			if generated_append_text(workspace, "%") != GENERATED_STATUS_OK {
				return Value{}, FORMATTING_STATUS_LIMIT_EXCEEDED
			}
			continue
		}
		if argument_index == len(arguments) {
			return Value{}, FORMATTING_STATUS_INVALID
		}
		status = generated_format_value(
			workspace, arguments[argument_index], verb, width,
			zero_padding, alternate,
		)
		if status != FORMATTING_STATUS_OK {
			return Value{}, status
		}
		argument_index++
	}
	if argument_index != len(arguments) {
		return Value{}, FORMATTING_STATUS_INVALID
	}
	result, generated_status := generated_text_value(workspace, start)
	return result, Formatting_Status(generated_status)
}

func format_directive_parse(source Format_Source) (
	_ Format_Verb,
	_ Format_Width,
	_ Format_Zero_Padding,
	_ Format_Alternate,
	_ Format_Count,
	status Format_Status,
) {
	defer func() { Format_Status_Invariants(status, "format_directive_parse.status") }()
	Format_Source_Invariants(source, "format_directive_parse.source")
	verb := Format_Verb(bits.WORD_8_MINIMUM)
	defer func() { Format_Verb_Invariants(verb, "format_directive_parse.verb") }()
	width := Format_Width(OUTPUT_SIZE_MINIMUM)
	defer func() { Format_Width_Invariants(width, "format_directive_parse.width") }()
	var zero_padding Format_Zero_Padding
	defer func() {
		Format_Zero_Padding_Invariants(zero_padding, "format_directive_parse.zero_padding")
	}()
	var alternate Format_Alternate
	defer func() {
		Format_Alternate_Invariants(alternate, "format_directive_parse.alternate")
	}()
	consumed := Format_Count(SOURCE_SIZE_MINIMUM)
	defer func() { Format_Count_Invariants(consumed, "format_directive_parse.consumed") }()
	if len(source) == SOURCE_SIZE_MINIMUM {
		return verb, width, zero_padding, alternate, consumed, FORMAT_STATUS_INVALID
	}
	position_index := SOURCE_SIZE_MINIMUM
	flags := true
	for flags {
		if source[position_index] == '0' {
			zero_padding = true
			position_index++
		} else if source[position_index] == '#' {
			alternate = true
			position_index++
		} else {
			flags = false
		}
		if position_index == len(source) {
			verb = Format_Verb(bits.WORD_8_MINIMUM)
			width = Format_Width(OUTPUT_SIZE_MINIMUM)
			zero_padding, alternate = false, false
			return verb, width, zero_padding, alternate, consumed, FORMAT_STATUS_INVALID
		}
	}
	for position_index < len(source) {
		character := source[position_index]
		if character < '0' {
			break
		}
		if character > '9' {
			break
		}
		width = width*strconv.DECIMAL_BASE + Format_Width(character-'0')
		if width > Format_Width(OUTPUT_SIZE_MAXIMUM) {
			verb = Format_Verb(bits.WORD_8_MINIMUM)
			width = Format_Width(OUTPUT_SIZE_MINIMUM)
			zero_padding, alternate = false, false
			return verb, width, zero_padding, alternate, consumed,
				FORMAT_STATUS_LIMIT_EXCEEDED
		}
		position_index++
	}
	if position_index == len(source) {
		verb = Format_Verb(bits.WORD_8_MINIMUM)
		width = Format_Width(OUTPUT_SIZE_MINIMUM)
		zero_padding, alternate = false, false
		return verb, width, zero_padding, alternate, consumed, FORMAT_STATUS_INVALID
	}
	verb = Format_Verb(source[position_index])
	consumed = Format_Count(position_index + SOURCE_POSITION_INCREMENT)
	return verb, width, zero_padding, alternate, consumed, FORMAT_STATUS_OK
}

func generated_format_value(
	workspace_state Workspace_Pointer, value_state Value,
	verb Format_Verb,
	width Format_Width,
	zero_padding Format_Zero_Padding,
	alternate Format_Alternate,
) (
	status Formatting_Status,
) {
	defer func() {
		Formatting_Status_Invariants(status, "generated_format_value.status")
	}()
	Workspace_Pointer_Invariants(workspace_state, "generated_format_value.workspace_state")
	Value_Invariants(value_state, "generated_format_value.value_state")
	Format_Verb_Invariants(verb, "generated_format_value.verb")
	Format_Width_Invariants(width, "generated_format_value.width")
	Format_Zero_Padding_Invariants(
		zero_padding, "generated_format_value.zero_padding",
	)
	Format_Alternate_Invariants(alternate, "generated_format_value.alternate")
	workspace := (Workspace_Pointer)(workspace_state)
	value := Value(value_state)
	kind := Value_Kind(value.Kind.Value)
	switch verb {
	case 'v':
		return Formatting_Status(generated_append_value(workspace, value))
	case 's':
		if kind != VALUE_TEXT {
			return FORMATTING_STATUS_INVALID
		}
		return Formatting_Status(generated_append_bytes(
			workspace, Text(value.Text.Value),
		))
	case 'q':
		if kind != VALUE_TEXT {
			return FORMATTING_STATUS_INVALID
		}
		return Formatting_Status(generated_format_quote(
			workspace, Text(value.Text.Value), alternate,
		))
	case 'b', 'd', 'o', 'x', 'X':
		if kind != VALUE_INTEGER {
			if kind != VALUE_UNSIGNED {
				return FORMATTING_STATUS_INVALID
			}
		}
		base := INTEGER_FORMAT_BASE_DECIMAL
		uppercase := Integer_Format_Uppercase(false)
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
		return Formatting_Status(generated_format_integer(
			workspace, value, base, uppercase, width, zero_padding,
		))
	case 'g':
		if kind < VALUE_INTEGER {
			return FORMATTING_STATUS_INVALID
		}
		if kind > VALUE_COMPLEX {
			return FORMATTING_STATUS_INVALID
		}
		return Formatting_Status(generated_append_value(workspace, value))
	case 'T':
		return Formatting_Status(generated_format_type(workspace, kind))
	}
	return FORMATTING_STATUS_INVALID
}

func generated_format_integer(
	workspace_state Workspace_Pointer, value_state Value,
	base Integer_Format_Base,
	uppercase Integer_Format_Uppercase,
	width Format_Width,
	zero_padding Format_Zero_Padding,
) (status Generated_Status) {
	defer func() {
		Generated_Status_Invariants(status, "generated_format_integer.status")
	}()
	Workspace_Pointer_Invariants(workspace_state, "generated_format_integer.workspace_state")
	Value_Invariants(value_state, "generated_format_integer.value_state")
	Integer_Format_Base_Invariants(base, "generated_format_integer.base")
	Integer_Format_Uppercase_Invariants(
		uppercase, "generated_format_integer.uppercase",
	)
	Format_Width_Invariants(width, "generated_format_integer.width")
	Format_Zero_Padding_Invariants(
		zero_padding, "generated_format_integer.zero_padding",
	)
	workspace := (Workspace_Pointer)(workspace_state)
	value := Value(value_state)
	count := SOURCE_SIZE_MINIMUM
	if Value_Kind(value.Kind.Value) == VALUE_INTEGER {
		count = int(strconv.Format_Integer_Into(
			strconv.Buffer(workspace.Number),
			strconv.Signed_Integer(int64(value.Scalar.Real)),
			strconv.Base(base),
		))
	} else {
		count = int(strconv.Format_Unsigned_Integer_Into(
			strconv.Buffer(workspace.Number),
			strconv.Unsigned_Integer(value.Scalar.Real),
			strconv.Base(base),
		))
	}
	if bool(uppercase) {
		for digit_index := SOURCE_SIZE_MINIMUM; digit_index < count; digit_index++ {
			if workspace.Number[digit_index] >= 'a' {
				workspace.Number[digit_index] -= 'a' - 'A'
			}
		}
	}
	return generated_append_padded(
		workspace, Formatted_Integer_Text_Validated{
			Value: Formatted_Integer_Text_Validated_Value(workspace.Number[:count]),
		},

		width, zero_padding,
	)
}

func generated_append_padded(
	workspace_state Workspace_Pointer, source_value Formatted_Integer_Text_Validated,
	width Format_Width,
	zero_padding Format_Zero_Padding,
) (status Generated_Status) {
	defer func() {
		Generated_Status_Invariants(status, "generated_append_padded.status")
	}()
	Workspace_Pointer_Invariants(workspace_state, "generated_append_padded.workspace_state")
	Formatted_Integer_Text_Validated_Invariants(
		source_value, "generated_append_padded.source_value",
	)
	Format_Width_Invariants(width, "generated_append_padded.width")
	Format_Zero_Padding_Invariants(
		zero_padding, "generated_append_padded.zero_padding",
	)
	workspace := (Workspace_Pointer)(workspace_state)
	source := Text(source_value.Value.(Formatted_Integer_Text_Validated_Value))
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
				workspace, source[:SOURCE_POSITION_INCREMENT],
			) != GENERATED_STATUS_OK {
				return GENERATED_STATUS_LIMIT_EXCEEDED
			}
			source = source[SOURCE_POSITION_INCREMENT:]
		}
	}
	for padding_index := OUTPUT_SIZE_MINIMUM; padding_index < padding_count; padding_index++ {
		append_status := generated_append_text(workspace, Generated_Text(padding))
		if append_status != GENERATED_STATUS_OK {
			return GENERATED_STATUS_LIMIT_EXCEEDED
		}
	}
	return generated_append_bytes(workspace, source)
}

func generated_format_type(
	workspace_state Workspace_Pointer, kind Value_Kind,
) (status Generated_Status) {
	defer func() {
		Generated_Status_Invariants(status, "generated_format_type.status")
	}()
	Workspace_Pointer_Invariants(workspace_state, "generated_format_type.workspace_state")
	Value_Kind_Invariants(kind, "generated_format_type.kind")
	workspace := (Workspace_Pointer)(workspace_state)
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
	return generated_append_text(workspace, Generated_Text(name))
}

func generated_format_quote(
	workspace_state Workspace_Pointer, source Text,
	alternate Format_Alternate,
) (status Generated_Status) {
	defer func() {
		Generated_Status_Invariants(status, "generated_format_quote.status")
	}()
	Workspace_Pointer_Invariants(workspace_state, "generated_format_quote.workspace_state")
	Text_Invariants(source, "generated_format_quote.source")
	Format_Alternate_Invariants(alternate, "generated_format_quote.alternate")
	workspace := (Workspace_Pointer)(workspace_state)
	if bool(alternate) {
		if bool(quote_backquote_safe(source)) {
			if generated_append_text(workspace, "`") != GENERATED_STATUS_OK {
				return GENERATED_STATUS_LIMIT_EXCEEDED
			}
			if generated_append_bytes(workspace, source) != GENERATED_STATUS_OK {
				return GENERATED_STATUS_LIMIT_EXCEEDED
			}
			return generated_append_text(workspace, "`")
		}
	}
	if generated_append_text(workspace, `"`) != GENERATED_STATUS_OK {
		return GENERATED_STATUS_LIMIT_EXCEEDED
	}
	for source_index := SOURCE_SIZE_MINIMUM; source_index < len(source); {
		value := source[source_index]
		if value < byte(utf8.CHARACTER_SELF) {
			status = generated_append_quoted_ascii(workspace, ASCII_Byte(value))
			source_index++
		} else {
			character, size := utf8.Decode_Character(utf8.Bytes(source[source_index:]))
			if bool(ucd.Is_Print(ucd.Character(character))) {
				status = generated_append_bytes(
					workspace, source[source_index:source_index+int(size)],
				)
			} else {
				status = generated_append_js_unicode(
					workspace, utf8.Decoded_Character(character),
				)
			}
			source_index += int(size)
		}
		if status != GENERATED_STATUS_OK {
			return status
		}
	}
	return generated_append_text(workspace, `"`)
}

func generated_append_quoted_ascii(
	workspace_state Workspace_Pointer, value_value ASCII_Byte,
) (status Generated_Status) {
	defer func() {
		Generated_Status_Invariants(status, "generated_append_quoted_ascii.status")
	}()
	Workspace_Pointer_Invariants(
		workspace_state, "generated_append_quoted_ascii.workspace_state",
	)
	ASCII_Byte_Invariants(value_value, "generated_append_quoted_ascii.value_value")
	workspace := (Workspace_Pointer)(workspace_state)
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
			return generated_append_control_byte(workspace, Quoted_Control_Byte(value))
		}
		if value == byte(utf8.CHARACTER_SELF)-utf8.CHARACTER_SIZE_MINIMUM {
			return generated_append_text(workspace, `\x7F`)
		}
	}
	if replacement != "" {
		return generated_append_text(workspace, Generated_Text(replacement))
	}
	return generated_append_bytes(workspace, []byte{value})
}

func generated_append_control_byte(
	workspace_state Workspace_Pointer, value_value Quoted_Control_Byte,
) (status Generated_Status) {
	defer func() {
		Generated_Status_Invariants(
			status, "generated_append_control_byte.status",
		)
	}()
	Workspace_Pointer_Invariants(
		workspace_state, "generated_append_control_byte.workspace_state",
	)
	Quoted_Control_Byte_Invariants(value_value, "generated_append_control_byte.value_value")
	workspace := (Workspace_Pointer)(workspace_state)
	value := byte(value_value)
	encoded := [len(`\x00`)]byte{
		'\\', 'x',
		TEMPLATE_HEXADECIMAL_DIGITS[value>>NUMBER_BASE_HEXADECIMAL_DIGIT_BIT_COUNT],
		TEMPLATE_HEXADECIMAL_DIGITS[value&HEXADECIMAL_DIGIT_MASK],
	}
	return generated_append_bytes(workspace, encoded[:])
}

func quote_backquote_safe(source Text) (safe Boolean) {
	defer func() { Boolean_Invariants(safe, "quote_backquote_safe.safe") }()
	Text_Invariants(source, "quote_backquote_safe.source")
	for source_index := SOURCE_SIZE_MINIMUM; source_index < len(source); {
		value := source[source_index]
		if value == '`' {
			return false
		}
		if value < byte(utf8.CHARACTER_SELF) {
			if value != '\t' {
				if value < ' ' {
					return false
				}
			}
			if value == byte(utf8.CHARACTER_SELF)-utf8.CHARACTER_SIZE_MINIMUM {
				return false
			}
			source_index++
			continue
		}
		character, size := utf8.Decode_Character(utf8.Bytes(source[source_index:]))
		if !bool(ucd.Is_Print(ucd.Character(character))) {
			return false
		}
		source_index += int(size)
	}
	return true
}

func generated_append_value(
	workspace_state Workspace_Pointer, value_state Value,
) (status Generated_Status) {
	defer func() {
		Generated_Status_Invariants(status, "generated_append_value.status")
	}()
	Workspace_Pointer_Invariants(workspace_state, "generated_append_value.workspace_state")
	Value_Invariants(value_state, "generated_append_value.value_state")
	workspace := (Workspace_Pointer)(workspace_state)
	start := int(workspace.Generated_Count.Value.(Workspace_Generated_Count_Storage_Value))
	end := start + OUTPUT_SIZE_MAXIMUM
	if end > len(workspace.Generated) {
		end = len(workspace.Generated)
	}
	if start == end {
		return GENERATED_STATUS_LIMIT_EXCEEDED
	}
	output := workspace.Output
	workspace.Output = Output_Storage(workspace.Generated[start:end])
	written, value_status := output_value(
		workspace,
		Output_Count_Staged{Value: Output_Count_Staged_Value(OUTPUT_SIZE_MINIMUM)},
		Value(value_state),
	)
	workspace.Output = output
	if value_status == OUTPUT_STATUS_TOO_LARGE {
		return GENERATED_STATUS_LIMIT_EXCEEDED
	}
	aver.Always(
		value_status == OUTPUT_STATUS_OK,
		"Validated builtin values cannot fail scalar or composite formatting.",
	)
	workspace.Generated_Count.Value =
		workspace.Generated_Count.Value.(Workspace_Generated_Count_Storage_Value) +
			Workspace_Generated_Count_Storage_Value(
				written.Value.(Output_Count_Staged_Value),
			)
	return GENERATED_STATUS_OK
}

func generated_append_bytes(
	workspace_state Workspace_Pointer, source Text,
) (status Generated_Status) {
	defer func() {
		Generated_Status_Invariants(status, "generated_append_bytes.status")
	}()
	Workspace_Pointer_Invariants(workspace_state, "generated_append_bytes.workspace_state")
	Text_Invariants(source, "generated_append_bytes.source")
	workspace := (Workspace_Pointer)(workspace_state)
	count := int(workspace.Generated_Count.Value.(Workspace_Generated_Count_Storage_Value))
	if count > len(workspace.Generated)-len(source) {
		return GENERATED_STATUS_LIMIT_EXCEEDED
	}
	copy(workspace.Generated[count:], source)
	workspace.Generated_Count.Value =
		workspace.Generated_Count.Value.(Workspace_Generated_Count_Storage_Value) +
			Workspace_Generated_Count_Storage_Value(len(source))
	return GENERATED_STATUS_OK
}

func generated_append_text(
	workspace_state Workspace_Pointer, source Generated_Text,
) (status Generated_Status) {
	defer func() {
		Generated_Status_Invariants(status, "generated_append_text.status")
	}()
	Workspace_Pointer_Invariants(workspace_state, "generated_append_text.workspace_state")
	Generated_Text_Invariants(source, "generated_append_text.source")
	workspace := (Workspace_Pointer)(workspace_state)
	count := int(workspace.Generated_Count.Value.(Workspace_Generated_Count_Storage_Value))
	if count > len(workspace.Generated)-len(source) {
		return GENERATED_STATUS_LIMIT_EXCEEDED
	}
	copy(workspace.Generated[count:], source)
	workspace.Generated_Count.Value =
		workspace.Generated_Count.Value.(Workspace_Generated_Count_Storage_Value) +
			Workspace_Generated_Count_Storage_Value(len(source))
	return GENERATED_STATUS_OK
}

func generated_text_value(
	workspace_state Workspace_Pointer, start_value Generated_Start_Validated,
) (
	_ Value,
	status Generated_Status,
) {
	defer func() { Generated_Status_Invariants(status, "generated_text_value.status") }()
	Workspace_Pointer_Invariants(workspace_state, "generated_text_value.workspace_state")
	Generated_Start_Validated_Invariants(
		start_value, "generated_text_value.start_value",
	)
	var result Value
	defer func() { Value_Invariants(result, "generated_text_value.result") }()
	workspace := (Workspace_Pointer)(workspace_state)
	start := Generated_Count(start_value.Value.(Generated_Start_Validated_Value))
	end := Generated_Count(
		workspace.Generated_Count.Value.(Workspace_Generated_Count_Storage_Value),
	)
	if int(end-start) > OUTPUT_SIZE_MAXIMUM {
		return Value{}, GENERATED_STATUS_LIMIT_EXCEEDED
	}
	result = Value_Of_Text(Text(workspace.Generated[start:end]))
	return result, GENERATED_STATUS_OK
}

func builtin_escape(
	workspace_state Workspace_Pointer, argument_values Arguments_Staged,
	kind Escape_Kind,
) (_ Value, status Generated_Status) {
	defer func() { Generated_Status_Invariants(status, "builtin_escape.status") }()
	Workspace_Pointer_Invariants(workspace_state, "builtin_escape.workspace_state")
	Arguments_Staged_Invariants(argument_values, "builtin_escape.argument_values")
	Escape_Kind_Invariants(kind, "builtin_escape.kind")
	var result Value
	defer func() { Value_Invariants(result, "builtin_escape.result") }()
	workspace := (Workspace_Pointer)(workspace_state)
	source, print_status := builtin_print(workspace, argument_values, false)
	if print_status != GENERATED_STATUS_OK {
		return Value{}, print_status
	}
	text := Text(source.Text.Value)
	start := Generated_Start_Validated{
		Value: Generated_Start_Validated_Value(
			workspace.Generated_Count.Value.(Workspace_Generated_Count_Storage_Value),
		),
	}

	generated_status := GENERATED_STATUS_OK
	switch kind {
	case ESCAPE_HTML:
		generated_status = generated_escape_html(workspace, text)
	case ESCAPE_JS:
		generated_status = generated_escape_js(workspace, text)
	case ESCAPE_URL_QUERY:
		generated_status = generated_escape_url_query(workspace, text)
	}
	if generated_status != GENERATED_STATUS_OK {
		return Value{}, generated_status
	}
	result, value_status := generated_text_value(workspace, start)
	return result, value_status
}

func generated_escape_html(
	workspace_state Workspace_Pointer, source Text,
) (status Generated_Status) {
	defer func() {
		Generated_Status_Invariants(status, "generated_escape_html.status")
	}()
	Workspace_Pointer_Invariants(workspace_state, "generated_escape_html.workspace_state")
	Text_Invariants(source, "generated_escape_html.source")
	workspace := (Workspace_Pointer)(workspace_state)
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
				workspace, Generated_Text(replacement),
			) != GENERATED_STATUS_OK {
				return GENERATED_STATUS_LIMIT_EXCEEDED
			}
		} else if generated_append_bytes(
			workspace, source[source_index:source_index+SOURCE_POSITION_INCREMENT],
		) != GENERATED_STATUS_OK {
			return GENERATED_STATUS_LIMIT_EXCEEDED
		}
	}
	return GENERATED_STATUS_OK
}

func generated_escape_js(
	workspace_state Workspace_Pointer, source Text,
) (status Generated_Status) {
	defer func() {
		Generated_Status_Invariants(status, "generated_escape_js.status")
	}()
	Workspace_Pointer_Invariants(workspace_state, "generated_escape_js.workspace_state")
	Text_Invariants(source, "generated_escape_js.source")
	workspace := (Workspace_Pointer)(workspace_state)
	for source_index := SOURCE_SIZE_MINIMUM; source_index < len(source); {
		value := source[source_index]
		if value < byte(utf8.CHARACTER_SELF) {
			status = generated_escape_js_ascii(workspace, ASCII_Byte(value))
			if status != GENERATED_STATUS_OK {
				return status
			}
			source_index++
			continue
		}
		character, size := utf8.Decode_Character(utf8.Bytes(source[source_index:]))
		if bool(ucd.Is_Print(ucd.Character(character))) {
			status = generated_append_bytes(
				workspace, source[source_index:source_index+int(size)],
			)
		} else {
			status = generated_append_js_unicode(
				workspace, utf8.Decoded_Character(character),
			)
		}
		if status != GENERATED_STATUS_OK {
			return status
		}
		source_index += int(size)
	}
	return GENERATED_STATUS_OK
}

func generated_escape_js_ascii(
	workspace_state Workspace_Pointer, value_value ASCII_Byte,
) (status Generated_Status) {
	defer func() {
		Generated_Status_Invariants(status, "generated_escape_js_ascii.status")
	}()
	Workspace_Pointer_Invariants(workspace_state, "generated_escape_js_ascii.workspace_state")
	ASCII_Byte_Invariants(value_value, "generated_escape_js_ascii.value_value")
	workspace := (Workspace_Pointer)(workspace_state)
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
			return generated_append_js_unicode(
				workspace, utf8.Decoded_Character(value),
			)
		}
	}
	if replacement != "" {
		return generated_append_text(workspace, Generated_Text(replacement))
	}
	return generated_append_bytes(workspace, []byte{value})
}

func generated_append_js_unicode(
	workspace_state Workspace_Pointer, value_value utf8.Decoded_Character,
) (status Generated_Status) {
	defer func() {
		Generated_Status_Invariants(status, "generated_append_js_unicode.status")
	}()
	Workspace_Pointer_Invariants(workspace_state, "generated_append_js_unicode.workspace_state")
	utf8.Decoded_Character_Invariants(
		value_value, "generated_append_js_unicode.value_value",
	)
	workspace := (Workspace_Pointer)(workspace_state)
	value := uint32(value_value)
	hexadecimal_count := strconv.SHORT_UNICODE_DIGIT_COUNT
	for value >= uint32(bits.CARRY_MAXIMUM)<<
		(hexadecimal_count*NUMBER_BASE_HEXADECIMAL_DIGIT_BIT_COUNT) {
		hexadecimal_count++
	}
	if generated_append_text(workspace, `\u`) != GENERATED_STATUS_OK {
		return GENERATED_STATUS_LIMIT_EXCEEDED
	}
	digit_index := hexadecimal_count - HEXADECIMAL_DIGIT_COUNT_INCREMENT
	for digit_index >= SOURCE_SIZE_MINIMUM {
		digit := value >> (digit_index * NUMBER_BASE_HEXADECIMAL_DIGIT_BIT_COUNT) &
			HEXADECIMAL_DIGIT_MASK
		if generated_append_bytes(
			workspace, []byte{TEMPLATE_HEXADECIMAL_DIGITS[digit]},
		) != GENERATED_STATUS_OK {
			return GENERATED_STATUS_LIMIT_EXCEEDED
		}
		digit_index--
	}
	return GENERATED_STATUS_OK
}

func generated_escape_url_query(
	workspace_state Workspace_Pointer, source Text,
) (status Generated_Status) {
	defer func() {
		Generated_Status_Invariants(status, "generated_escape_url_query.status")
	}()
	Workspace_Pointer_Invariants(workspace_state, "generated_escape_url_query.workspace_state")
	Text_Invariants(source, "generated_escape_url_query.source")
	workspace := (Workspace_Pointer)(workspace_state)
	for source_index := range source {
		value := source[source_index]
		if bool(url_query_safe(utf8.Byte(value))) {
			status = generated_append_bytes(
				workspace,
				source[source_index:source_index+SOURCE_POSITION_INCREMENT],
			)
		} else if value == ' ' {
			status = generated_append_text(workspace, "+")
		} else {
			high := value >> NUMBER_BASE_HEXADECIMAL_DIGIT_BIT_COUNT
			encoded := [URL_QUERY_ESCAPE_SIZE]byte{
				'%',
				TEMPLATE_HEXADECIMAL_DIGITS[high],
				TEMPLATE_HEXADECIMAL_DIGITS[value&HEXADECIMAL_DIGIT_MASK],
			}
			status = generated_append_bytes(workspace, encoded[:])
		}
		if status != GENERATED_STATUS_OK {
			return status
		}
	}
	return GENERATED_STATUS_OK
}

func url_query_safe(value_value utf8.Byte) (safe Boolean) {
	defer func() { Boolean_Invariants(safe, "url_query_safe.safe") }()
	utf8.Byte_Invariants(value_value, "url_query_safe.value_value")
	value := byte(value_value)
	switch value {
	case '-', '.', '_', '~':
		return true
	}
	if value >= 'a' {
		if value <= 'z' {
			return true
		}
	}
	if value >= 'A' {
		if value <= 'Z' {
			return true
		}
	}
	if value >= '0' {
		return Boolean(value <= '9')
	}
	return false
}

func builtin_order(
	kind Builtin_Order_Kind,
	argument_values Arguments_Staged,
) (_ Value, status Builtin_Value_Status) {
	defer func() { Builtin_Value_Status_Invariants(status, "builtin_order.status") }()
	Builtin_Order_Kind_Invariants(kind, "builtin_order.kind")
	Arguments_Staged_Invariants(argument_values, "builtin_order.argument_values")
	var result Value
	defer func() { Value_Invariants(result, "builtin_order.result") }()
	arguments := argument_values.Value.(Arguments_Staged_Value)
	if len(arguments) != BUILTIN_ARGUMENT_COUNT_TWO {
		return Value{}, BUILTIN_VALUE_STATUS_FUNCTION_INVALID
	}
	order, comparable := values_order(
		arguments[BUILTIN_ARGUMENT_FIRST], arguments[BUILTIN_ARGUMENT_SECOND],
	)
	if !comparable {
		return Value{}, BUILTIN_VALUE_STATUS_KIND_INVALID
	}
	matched := false
	if kind == BUILTIN_ORDER_LESS {
		matched = order < ORDER_SAME
	} else if kind == BUILTIN_ORDER_LESS_EQUAL {
		matched = order <= ORDER_SAME
	} else if kind == BUILTIN_ORDER_GREATER {
		matched = order > ORDER_SAME
	} else {
		aver.Always(
			kind == BUILTIN_ORDER_GREATER_EQUAL,
			"Builtin dispatch permits only standard comparison kinds.",
		)
		matched = order >= ORDER_SAME
	}
	result = Value_Of_Boolean(Boolean(matched))
	return result, BUILTIN_VALUE_STATUS_OK
}

func builtin_order_kind(name Builtin_Order_Name_Validated) (kind Builtin_Order_Kind) {
	defer func() { Builtin_Order_Kind_Invariants(kind, "builtin_order_kind.kind") }()
	Builtin_Order_Name_Validated_Invariants(name, "builtin_order_kind.name")
	if name.First == 'l' {
		if name.Second == 't' {
			return BUILTIN_ORDER_LESS
		}
		return BUILTIN_ORDER_LESS_EQUAL
	}
	if name.Second == 't' {
		return BUILTIN_ORDER_GREATER
	}
	aver.Always(
		name.First == 'g',
		"Builtin dispatch permits only standard comparison names.",
	)
	return BUILTIN_ORDER_GREATER_EQUAL
}

func values_order(
	left Value,
	right Value,
) (_ Order, comparable Comparable) {
	defer func() { Comparable_Invariants(comparable, "values_order.comparable") }()
	Value_Invariants(left, "values_order.left")
	Value_Invariants(right, "values_order.right")
	var order Order
	defer func() { Order_Invariants(order, "values_order.order") }()
	left_kind := Value_Kind(left.Kind.Value)
	right_kind := Value_Kind(right.Kind.Value)
	if left_kind == VALUE_INTEGER {
		if right_kind == VALUE_UNSIGNED {
			order = integer_unsigned_order(left, right)
			return order, true
		}
	}
	if left_kind == VALUE_UNSIGNED {
		if right_kind == VALUE_INTEGER {
			order = -integer_unsigned_order(right, left)
			return order, true
		}
	}
	if left_kind != right_kind {
		return ORDER_SAME, false
	}
	switch left_kind {
	case VALUE_INTEGER:
		order = int64_order(
			Integer(left.Scalar.Real),
			Integer(right.Scalar.Real),
		)
		return order, true
	case VALUE_UNSIGNED:
		order = uint64_order(
			Unsigned(left.Scalar.Real),
			Unsigned(right.Scalar.Real),
		)
		return order, true
	case VALUE_FLOAT:
		order, comparable = float64_order(
			Float(left.Scalar.Real),
			Float(right.Scalar.Real),
		)
		return order, comparable
	case VALUE_TEXT:
		order = bytes_order(
			Text(left.Text.Value), Text(right.Text.Value),
		)
		return order, true
	default:
		return ORDER_SAME, false
	}
}

func integer_unsigned_order(
	integer Value,
	unsigned Value,
) (order Order) {
	defer func() {
		Order_Invariants(order, "integer_unsigned_order.order")
	}()
	Value_Invariants(integer, "integer_unsigned_order.integer")
	Value_Invariants(unsigned, "integer_unsigned_order.unsigned")
	integer_scalar := Integer(integer.Scalar.Real)
	unsigned_scalar := Unsigned(unsigned.Scalar.Real)
	if integer_scalar < Integer(bits.WORD_64_MINIMUM) {
		return ORDER_BEFORE
	}
	return uint64_order(Unsigned(integer_scalar), unsigned_scalar)
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

func float64_order(
	left Float,
	right Float,
) (_ Order, comparable Comparable) {
	defer func() { Comparable_Invariants(comparable, "float64_order.comparable") }()
	Float_Invariants(left, "float64_order.left")
	Float_Invariants(right, "float64_order.right")
	var order Order
	defer func() { Order_Invariants(order, "float64_order.order") }()
	if float64_nan(left) {
		return ORDER_SAME, false
	}
	if float64_nan(right) {
		return ORDER_SAME, false
	}
	left_encoding := uint64(left)
	right_encoding := uint64(right)
	if left_encoding&^FLOAT_64_SIGN_MASK == bits.WORD_64_MINIMUM {
		if right_encoding&^FLOAT_64_SIGN_MASK == bits.WORD_64_MINIMUM {
			return ORDER_SAME, true
		}
	}
	left_negative := left_encoding&FLOAT_64_SIGN_MASK != bits.WORD_64_MINIMUM
	right_negative := right_encoding&FLOAT_64_SIGN_MASK != bits.WORD_64_MINIMUM
	if left_negative != right_negative {
		if left_negative {
			order = ORDER_BEFORE
			return order, true
		}
		order = ORDER_AFTER
		return order, true
	}
	order = uint64_order(Unsigned(left), Unsigned(right))
	if left_negative {
		order = -order
	}
	return order, true
}

func float64_nan(value Float) (nan Na_N) {
	defer func() { Na_N_Invariants(nan, "float64_nan.nan") }()
	Float_Invariants(value, "float64_nan.value")
	encoding := uint64(value)
	return Na_N(
		encoding&FLOAT_64_EXPONENT_MASK == FLOAT_64_EXPONENT_MASK &&
			encoding&FLOAT_64_FRACTION_MASK != bits.WORD_64_MINIMUM,
	)
}

func bytes_order(left Text, right Text) (order Order) {
	defer func() { Order_Invariants(order, "bytes_order.order") }()
	Text_Invariants(left, "bytes_order.left")
	Text_Invariants(right, "bytes_order.right")
	count := len(left)
	if len(right) < count {
		count = len(right)
	}
	for index := SOURCE_SIZE_MINIMUM; index < count; index++ {
		if left[index] < right[index] {
			return ORDER_BEFORE
		}
		if left[index] > right[index] {
			return ORDER_AFTER
		}
	}
	return int64_order(Integer(len(left)), Integer(len(right)))
}

func builtin_index(argument_values Arguments_Staged) (
	_ Value,
	status Builtin_Value_Status,
) {
	defer func() { Builtin_Value_Status_Invariants(status, "builtin_index.status") }()
	Arguments_Staged_Invariants(argument_values, "builtin_index.argument_values")
	var result Value
	defer func() { Value_Invariants(result, "builtin_index.result") }()
	arguments := argument_values.Value.(Arguments_Staged_Value)
	if len(arguments) < BUILTIN_ARGUMENT_COUNT_TWO {
		return Value{}, BUILTIN_VALUE_STATUS_FUNCTION_INVALID
	}
	result = arguments[BUILTIN_ARGUMENT_FIRST]
	for index := BUILTIN_ARGUMENT_SECOND; index < len(arguments); index++ {
		indexed, index_status := value_index(result, arguments[index])
		if index_status != INDEX_STATUS_OK {
			return Value{}, BUILTIN_VALUE_STATUS_KIND_INVALID
		}
		result = indexed
	}
	return result, BUILTIN_VALUE_STATUS_OK
}

func value_index(
	item Value,
	index Value,
) (_ Value, status Index_Status) {
	defer func() { Index_Status_Invariants(status, "value_index.status") }()
	Value_Invariants(item, "value_index.item")
	Value_Invariants(index, "value_index.index")
	var result Value
	defer func() { Value_Invariants(result, "value_index.result") }()
	switch Value_Kind(item.Kind.Value) {
	case VALUE_TEXT:
		position, valid := index_position(
			index, Collection_Limit(len(Text(item.Text.Value))),
		)
		if !valid {
			return Value{}, INDEX_STATUS_INVALID
		}
		result = Value_Of_Unsigned(Unsigned(
			Text(item.Text.Value)[position],
		))
		return result, INDEX_STATUS_OK
	case VALUE_SEQUENCE:
		position, valid := index_position(
			index, Collection_Limit(len(Values(item.Values.Value))),
		)
		if !valid {
			return Value{}, INDEX_STATUS_INVALID
		}
		result = Values(item.Values.Value)[position]
		return result, INDEX_STATUS_OK
	case VALUE_MAP:
		fields := Fields(item.Fields.Value)
		for field_index := range fields {
			if bool(values_equal(fields[field_index].Key.Data, index)) {
				result = fields[field_index].Value.Data
				return result, INDEX_STATUS_OK
			}
		}
		result = Value_Nil()
		return result, INDEX_STATUS_OK
	}
	return Value{}, INDEX_STATUS_INVALID
}

func index_position(
	index Value,
	limit Collection_Limit,
) (_ Collection_Element_Index, valid Collection_Position_Validity) {
	defer func() { Collection_Position_Validity_Invariants(valid, "index_position.valid") }()
	Value_Invariants(index, "index_position.index")
	Collection_Limit_Invariants(limit, "index_position.limit")
	position := Collection_Element_Index(bytes.SLICE_SIZE_MINIMUM)
	defer func() { Collection_Element_Index_Invariants(position, "index_position.position") }()
	switch Value_Kind(index.Kind.Value) {
	case VALUE_INTEGER:
		value := Integer(index.Scalar.Real)
		if value < Integer(bits.WORD_64_MINIMUM) {
			return Collection_Element_Index(bytes.SLICE_SIZE_MINIMUM), false
		}
		if value >= Integer(limit) {
			return Collection_Element_Index(bytes.SLICE_SIZE_MINIMUM), false
		}
		position = Collection_Element_Index(value)
		return position, true
	case VALUE_UNSIGNED:
		value := Unsigned(index.Scalar.Real)
		if value >= Unsigned(limit) {
			return Collection_Element_Index(bytes.SLICE_SIZE_MINIMUM), false
		}
		position = Collection_Element_Index(value)
		return position, true
	}
	return Collection_Element_Index(bytes.SLICE_SIZE_MINIMUM), false
}

func builtin_slice(argument_values Arguments_Staged) (
	_ Value,
	status Builtin_Value_Status,
) {
	defer func() { Builtin_Value_Status_Invariants(status, "builtin_slice.status") }()
	Arguments_Staged_Invariants(argument_values, "builtin_slice.argument_values")
	var result Value
	defer func() { Value_Invariants(result, "builtin_slice.result") }()
	arguments := argument_values.Value.(Arguments_Staged_Value)
	if len(arguments) < SLICE_ARGUMENT_COUNT_MINIMUM {
		return Value{}, BUILTIN_VALUE_STATUS_FUNCTION_INVALID
	}
	if len(arguments) > SLICE_ARGUMENT_COUNT_MAXIMUM {
		return Value{}, BUILTIN_VALUE_STATUS_FUNCTION_INVALID
	}
	item := arguments[BUILTIN_ARGUMENT_FIRST]
	count := VALUE_COUNT_MAXIMUM - VALUE_COUNT_MAXIMUM
	capacity := VALUE_COUNT_MAXIMUM - VALUE_COUNT_MAXIMUM
	switch Value_Kind(item.Kind.Value) {
	case VALUE_TEXT:
		count = len(Text(item.Text.Value))
		capacity = count
	case VALUE_SEQUENCE:
		count = len(Values(item.Values.Value))
		capacity = cap(Values(item.Values.Value))
		if capacity > VALUE_COUNT_MAXIMUM {
			capacity = VALUE_COUNT_MAXIMUM
		}
	default:
		return Value{}, BUILTIN_VALUE_STATUS_KIND_INVALID
	}
	indexes := [SLICE_INDEX_COUNT]Collection_Position{
		SLICE_INDEX_START:    bytes.SLICE_SIZE_MINIMUM,
		SLICE_INDEX_END:      Collection_Position(count),
		SLICE_INDEX_CAPACITY: Collection_Position(capacity),
	}
	for index := SLICE_ARGUMENT_COUNT_MINIMUM; index < len(arguments); index++ {
		position, valid := slice_position(
			arguments[index], Collection_Limit(capacity),
		)
		if !valid {
			return Value{}, BUILTIN_VALUE_STATUS_KIND_INVALID
		}
		indexes[index-SLICE_ARGUMENT_COUNT_MINIMUM] = position
	}
	if indexes[SLICE_INDEX_START] > indexes[SLICE_INDEX_END] {
		return Value{}, BUILTIN_VALUE_STATUS_KIND_INVALID
	}
	if len(arguments) == SLICE_ARGUMENT_COUNT_MAXIMUM {
		if Value_Kind(item.Kind.Value) == VALUE_TEXT {
			return Value{}, BUILTIN_VALUE_STATUS_KIND_INVALID
		}
		if indexes[SLICE_INDEX_END] > indexes[SLICE_INDEX_CAPACITY] {
			return Value{}, BUILTIN_VALUE_STATUS_KIND_INVALID
		}
	}
	start := indexes[SLICE_INDEX_START]
	end := indexes[SLICE_INDEX_END]
	if Value_Kind(item.Kind.Value) == VALUE_TEXT {
		result = Value_Of_Text(Text(item.Text.Value)[start:end])
		return result, BUILTIN_VALUE_STATUS_OK
	}
	values := Values(item.Values.Value)
	if len(arguments) == SLICE_ARGUMENT_COUNT_MAXIMUM {
		maximum := indexes[SLICE_INDEX_CAPACITY]
		result = Value_Of_Sequence(values[start:end:maximum])
		return result, BUILTIN_VALUE_STATUS_OK
	}
	result = Value_Of_Sequence(values[start:end])
	return result, BUILTIN_VALUE_STATUS_OK
}

func slice_position(
	index Value,
	limit Collection_Limit,
) (_ Collection_Position, valid Collection_Position_Validity) {
	defer func() { Collection_Position_Validity_Invariants(valid, "slice_position.valid") }()
	Value_Invariants(index, "slice_position.index")
	Collection_Limit_Invariants(limit, "slice_position.limit")
	position := Collection_Position(bytes.SLICE_SIZE_MINIMUM)
	defer func() { Collection_Position_Invariants(position, "slice_position.position") }()
	switch Value_Kind(index.Kind.Value) {
	case VALUE_INTEGER:
		value := Integer(index.Scalar.Real)
		if value < Integer(bits.WORD_64_MINIMUM) {
			return Collection_Position(bytes.SLICE_SIZE_MINIMUM), false
		}
		if value > Integer(limit) {
			return Collection_Position(bytes.SLICE_SIZE_MINIMUM), false
		}
		position = Collection_Position(value)
		return position, true
	case VALUE_UNSIGNED:
		value := Unsigned(index.Scalar.Real)
		if value > Unsigned(limit) {
			return Collection_Position(bytes.SLICE_SIZE_MINIMUM), false
		}
		position = Collection_Position(value)
		return position, true
	}
	return Collection_Position(bytes.SLICE_SIZE_MINIMUM), false
}

func value_truth(value_state Value) (truth Truth) {
	defer func() { Truth_Invariants(truth, "value_truth.truth") }()
	Value_Invariants(value_state, "value_truth.value_state")
	value := Value(value_state)
	switch Value_Kind(value.Kind.Value) {
	case VALUE_NIL:
		return false
	case VALUE_BOOLEAN:
		return Truth(uint64(value.Scalar.Real) != bits.WORD_64_MINIMUM)
	case VALUE_INTEGER:
		return Truth(int64(value.Scalar.Real) != int64(bits.WORD_64_MINIMUM))
	case VALUE_UNSIGNED:
		return Truth(uint64(value.Scalar.Real) != bits.WORD_64_MINIMUM)
	case VALUE_FLOAT:
		return Truth(
			uint64(value.Scalar.Real)&^FLOAT_64_SIGN_MASK != bits.WORD_64_MINIMUM,
		)
	case VALUE_COMPLEX:
		if uint64(value.Scalar.Real)&^FLOAT_64_SIGN_MASK != bits.WORD_64_MINIMUM {
			return true
		}
		return Truth(
			uint64(value.Scalar.Imaginary)&^FLOAT_64_SIGN_MASK !=
				bits.WORD_64_MINIMUM,
		)
	case VALUE_TEXT:
		return Truth(len(Text(value.Text.Value)) != OUTPUT_SIZE_MINIMUM)
	case VALUE_SEQUENCE:
		return Truth(len(Values(value.Values.Value)) != ARGUMENT_COUNT_MINIMUM)
	case VALUE_MAP:
		return Truth(len(Fields(value.Fields.Value)) != ARGUMENT_COUNT_MINIMUM)
	case VALUE_OBJECT, VALUE_FUNCTION:
		return true
	}
	return false
}

func values_equal(
	left_value Value, right_value Value,
) (equal Equality) {
	defer func() { Equality_Invariants(equal, "values_equal.equal") }()
	Value_Invariants(left_value, "values_equal.left_value")
	Value_Invariants(right_value, "values_equal.right_value")
	left := Value(left_value)
	right := Value(right_value)
	left_kind := Value_Kind(left.Kind.Value)
	right_kind := Value_Kind(right.Kind.Value)
	if left_kind == VALUE_INTEGER {
		if right_kind == VALUE_UNSIGNED {
			left_integer := Integer(int64(left.Scalar.Real))
			if left_integer < Integer(bits.WORD_64_MINIMUM) {
				return false
			}
			return Equality(
				Unsigned(left_integer) == Unsigned(right.Scalar.Real),
			)
		}
	}
	if left_kind == VALUE_UNSIGNED {
		if right_kind == VALUE_INTEGER {
			right_integer := Integer(int64(right.Scalar.Real))
			if right_integer < Integer(bits.WORD_64_MINIMUM) {
				return false
			}
			return Equality(
				Unsigned(left.Scalar.Real) == Unsigned(right_integer),
			)
		}
	}
	if left_kind != right_kind {
		return false
	}
	switch left_kind {
	case VALUE_NIL:
		return true
	case VALUE_BOOLEAN:
		return Equality(
			left.Scalar.Real == right.Scalar.Real,
		)
	case VALUE_INTEGER:
		return Equality(
			left.Scalar.Real == right.Scalar.Real,
		)
	case VALUE_UNSIGNED:
		return Equality(
			left.Scalar.Real == right.Scalar.Real,
		)
	case VALUE_FLOAT:
		return float64_equal(
			Float(left.Scalar.Real),
			Float(right.Scalar.Real),
		)
	case VALUE_COMPLEX:
		return Equality(
			bool(float64_equal(
				Float(left.Scalar.Real),
				Float(right.Scalar.Real),
			)) && bool(float64_equal(
				Float(left.Scalar.Imaginary),
				Float(right.Scalar.Imaginary),
			)),
		)
	case VALUE_TEXT:
		return Equality(bytes_equal(
			Text(left.Text.Value), Text(right.Text.Value),
		))
	}
	return false
}

func float64_equal(
	left Float, right Float,
) (equal Equality) {
	defer func() { Equality_Invariants(equal, "float64_equal.equal") }()
	Float_Invariants(left, "float64_equal.left")
	Float_Invariants(right, "float64_equal.right")
	order, comparable := float64_order(left, right)
	return Equality(comparable && order == ORDER_SAME)
}

func output_value(
	workspace_state Workspace_Pointer, count_value Output_Count_Staged,
	value_state Value,
) (_ Output_Count_Staged, status Output_Status) {
	defer func() { Output_Status_Invariants(status, "output_value.status") }()
	Workspace_Pointer_Invariants(workspace_state, "output_value.workspace_state")
	Output_Count_Staged_Invariants(count_value, "output_value.count_value")
	Value_Invariants(value_state, "output_value.value_state")
	written := count_value
	defer func() { Output_Count_Staged_Invariants(written, "output_value.written") }()
	workspace := (Workspace_Pointer)(workspace_state)
	value := Value(value_state)
	kind := Value_Kind(value.Kind.Value)
	if kind == VALUE_SEQUENCE {
		written, status = output_composite(workspace, count_value, value)
		return written, status
	}
	if kind == VALUE_OBJECT {
		written, status = output_composite(workspace, count_value, value)
		return written, status
	}
	if kind == VALUE_MAP {
		written, status = output_composite(workspace, count_value, value)
		return written, status
	}
	written, status = output_scalar(workspace, count_value, value)
	return written, status
}

func output_scalar(
	workspace_state Workspace_Pointer, count_value Output_Count_Staged,
	value_state Value,
) (_ Output_Count_Staged, status Output_Status) {
	defer func() { Output_Status_Invariants(status, "output_scalar.status") }()
	Workspace_Pointer_Invariants(workspace_state, "output_scalar.workspace_state")
	Output_Count_Staged_Invariants(count_value, "output_scalar.count_value")
	Value_Invariants(value_state, "output_scalar.value_state")
	written := count_value
	defer func() { Output_Count_Staged_Invariants(written, "output_scalar.written") }()
	workspace := (Workspace_Pointer)(workspace_state)
	value := Value(value_state)
	kind := Value_Kind(value.Kind.Value)
	aver.Always(
		kind != VALUE_SEQUENCE,
		"Sequences are dispatched before scalar formatting.",
	)
	aver.Always(
		kind != VALUE_OBJECT,
		"Objects are dispatched before scalar formatting.",
	)
	aver.Always(
		kind != VALUE_MAP,
		"Maps are dispatched before scalar formatting.",
	)
	switch kind {
	case VALUE_NIL:
		written, status = output_text(workspace, written, "<no value>")
	case VALUE_BOOLEAN:
		if uint64(value.Scalar.Real) != bits.WORD_64_MINIMUM {
			written, status = output_text(workspace, written, "true")
			return written, status
		}
		written, status = output_text(workspace, written, "false")
	case VALUE_INTEGER:
		integer := Integer(int64(value.Scalar.Real))
		count := strconv.Format_Integer_Into(
			strconv.Buffer(workspace.Number), strconv.Signed_Integer(integer),
			strconv.DECIMAL_BASE,
		)
		written, status = output_bytes(
			workspace, count_value, Text(workspace.Number[:count]),
		)
	case VALUE_UNSIGNED:
		unsigned := Unsigned(value.Scalar.Real)
		count := strconv.Format_Unsigned_Integer_Into(
			strconv.Buffer(workspace.Number), strconv.Unsigned_Integer(unsigned),
			strconv.DECIMAL_BASE,
		)
		written, status = output_bytes(
			workspace, count_value, Text(workspace.Number[:count]),
		)
	case VALUE_FLOAT:
		written, status = output_float_bits(
			workspace, written, Float(value.Scalar.Real),
		)
	case VALUE_COMPLEX:
		written, status = output_complex_bits(
			workspace, written, Float(value.Scalar.Real),
			Float(value.Scalar.Imaginary),
		)
	case VALUE_TEXT:
		written, status = output_bytes(
			workspace, written, Text(value.Text.Value),
		)
	case VALUE_FUNCTION:
		written, status = output_text(workspace, written, "<function>")
	}
	return written, status
}

func output_composite(
	workspace_state Workspace_Pointer, count_value Output_Count_Staged,
	value_state Value,
) (_ Output_Count_Staged, status Output_Status) {
	defer func() { Output_Status_Invariants(status, "output_composite.status") }()
	Workspace_Pointer_Invariants(workspace_state, "output_composite.workspace_state")
	Output_Count_Staged_Invariants(count_value, "output_composite.count_value")
	Value_Invariants(value_state, "output_composite.value_state")
	workspace := (Workspace_Pointer)(workspace_state)
	count := count_value
	defer func() { Output_Count_Staged_Invariants(count, "output_composite.written") }()
	workspace.Format[bytes.SLICE_SIZE_MINIMUM].Value =
		Value_Format_Frame_Stored(Value_Format_Frame{Value: Value(value_state)})
	depth := Frame_Index(FRAME_COUNT_INCREMENT)
	for depth > Frame_Index(bytes.SLICE_SIZE_MINIMUM) {
		frame_state := &workspace.Format[depth-Frame_Index(FRAME_COUNT_INCREMENT)]
		frame := &frame_state.Value
		if !bool(frame.Opened) {
			var write_status Output_Status
			count, write_status = output_composite_open(workspace, count, frame.Value)
			if write_status != OUTPUT_STATUS_OK {
				return count, write_status
			}
			frame.Opened = true
		}
		child_count := output_composite_count(frame.Value)
		if Collection_Count(frame.Index) == child_count {
			var write_status Output_Status
			count, write_status = output_composite_close(workspace, count, frame.Value)
			if write_status != OUTPUT_STATUS_OK {
				return count, write_status
			}
			depth--
			continue
		}
		if frame.Index > Collection_Index(bytes.SLICE_SIZE_MINIMUM) {
			var write_status Output_Status
			count, write_status = output_text(workspace, count, " ")
			if write_status != OUTPUT_STATUS_OK {
				return count, OUTPUT_STATUS_TOO_LARGE
			}
		}
		child, child_written, child_status := output_composite_child(
			workspace, count, frame_state,
		)
		count = child_written
		if child_status != OUTPUT_STATUS_OK {
			return count, child_status
		}
		kind := Value_Kind(child.Kind.Value)
		composite := kind == VALUE_SEQUENCE || kind == VALUE_OBJECT ||
			kind == VALUE_MAP
		if composite {
			aver.Always(
				int(depth) < len(workspace.Format),
				"Value validation rejects depth beyond format storage.",
			)
			workspace.Format[depth].Value =
				Value_Format_Frame_Stored(Value_Format_Frame{Value: child})
			depth++
			continue
		}
		var write_status Output_Status
		count, write_status = output_scalar(workspace, count, child)
		if write_status != OUTPUT_STATUS_OK {
			return count, OUTPUT_STATUS_TOO_LARGE
		}
	}
	return count, OUTPUT_STATUS_OK
}

func output_composite_count(value_state Value) (
	count Collection_Count,
) {
	defer func() {
		Collection_Count_Invariants(count, "output_composite_count.count")
	}()
	Value_Invariants(value_state, "output_composite_count.value_state")
	value := Value(value_state)
	kind := Value_Kind(value.Kind.Value)
	Composite_Kind_Invariants(
		Composite_Kind(kind), "output_composite_count.kind",
	)
	switch kind {
	case VALUE_SEQUENCE:
		return Collection_Count(len(Values(value.Values.Value)))
	case VALUE_OBJECT, VALUE_MAP:
		return Collection_Count(len(Fields(value.Fields.Value)))
	}
	return VALUE_COUNT_MAXIMUM - VALUE_COUNT_MAXIMUM
}

func output_composite_open(
	workspace_state Workspace_Pointer, count_value Output_Count_Staged, value_state Value,
) (
	_ Output_Count_Staged, status Output_Status,
) {
	defer func() { Output_Status_Invariants(status, "output_composite_open.status") }()
	Workspace_Pointer_Invariants(workspace_state, "output_composite_open.workspace_state")
	Output_Count_Staged_Invariants(count_value, "output_composite_open.count_value")
	Value_Invariants(value_state, "output_composite_open.value_state")
	written := count_value
	defer func() { Output_Count_Staged_Invariants(written, "output_composite_open.written") }()
	workspace := (Workspace_Pointer)(workspace_state)
	value := Value(value_state)
	if Value_Kind(value.Kind.Value) == VALUE_MAP {
		written, status = output_text(workspace, count_value, "map[")
		return written, status
	}
	if Value_Kind(value.Kind.Value) == VALUE_OBJECT {
		written, status = output_text(workspace, count_value, "{")
		return written, status
	}
	written, status = output_text(workspace, count_value, "[")
	return written, status
}

func output_composite_close(
	workspace_state Workspace_Pointer, count_value Output_Count_Staged, value_state Value,
) (
	_ Output_Count_Staged, status Output_Status,
) {
	defer func() { Output_Status_Invariants(status, "output_composite_close.status") }()
	Workspace_Pointer_Invariants(workspace_state, "output_composite_close.workspace_state")
	Output_Count_Staged_Invariants(count_value, "output_composite_close.count_value")
	Value_Invariants(value_state, "output_composite_close.value_state")
	written := count_value
	defer func() { Output_Count_Staged_Invariants(written, "output_composite_close.written") }()
	workspace := (Workspace_Pointer)(workspace_state)
	value := Value(value_state)
	if Value_Kind(value.Kind.Value) == VALUE_OBJECT {
		written, status = output_text(workspace, count_value, "}")
		return written, status
	}
	written, status = output_text(workspace, count_value, "]")
	return written, status
}

func output_composite_child(
	workspace_state Workspace_Pointer, count_value Output_Count_Staged,
	frame_state Value_Format_Frame_Storage_Pointer,
) (_ Value, _ Output_Count_Staged, status Output_Status) {
	defer func() { Output_Status_Invariants(status, "output_composite_child.status") }()
	Workspace_Pointer_Invariants(workspace_state, "output_composite_child.workspace_state")
	Output_Count_Staged_Invariants(count_value, "output_composite_child.count_value")
	Value_Format_Frame_Storage_Pointer_Invariants(
		frame_state, "output_composite_child.frame_state")
	var child Value
	defer func() { Value_Invariants(child, "output_composite_child.child") }()
	written := count_value
	defer func() { Output_Count_Staged_Invariants(written, "output_composite_child.written") }()

	workspace := (Workspace_Pointer)(workspace_state)
	frame := &frame_state.Value
	value := frame.Value
	index := frame.Index
	frame.Index++
	kind := Value_Kind(value.Kind.Value)
	Composite_Kind_Invariants(
		Composite_Kind(kind), "output_composite_child.kind",
	)
	switch kind {
	case VALUE_SEQUENCE:
		child = Values(value.Values.Value)[index]
		return child, written, OUTPUT_STATUS_OK
	case VALUE_OBJECT:
		child = Fields(value.Fields.Value)[index].Value.Data
		return child, written, OUTPUT_STATUS_OK
	case VALUE_MAP:
		key, item, selected := range_map_next(
			value, Prior_Map_Entry_Index(frame.Map_Entry_Index),
			frame.Map_Selected,
		)
		frame.Map_Entry_Index = selected
		frame.Map_Selected = true
		var write_status Output_Status
		written, write_status = output_scalar(workspace, count_value, key)
		if write_status != OUTPUT_STATUS_OK {
			return child, written, OUTPUT_STATUS_TOO_LARGE
		}
		written, write_status = output_text(workspace, written, ":")
		if write_status != OUTPUT_STATUS_OK {
			return child, written, OUTPUT_STATUS_TOO_LARGE
		}
		child = item
		return child, written, OUTPUT_STATUS_OK
	}
	return child, written, OUTPUT_STATUS_OK
}

func output_complex_bits(
	workspace_state Workspace_Pointer, count_value Output_Count_Staged,
	real Float,
	imaginary Float,
) (_ Output_Count_Staged, status Output_Status) {
	defer func() { Output_Status_Invariants(status, "output_complex_bits.status") }()
	Workspace_Pointer_Invariants(workspace_state, "output_complex_bits.workspace_state")
	Output_Count_Staged_Invariants(count_value, "output_complex_bits.count_value")
	Float_Invariants(real, "output_complex_bits.real")
	Float_Invariants(imaginary, "output_complex_bits.imaginary")
	written := count_value
	defer func() { Output_Count_Staged_Invariants(written, "output_complex_bits.written") }()
	workspace := (Workspace_Pointer)(workspace_state)
	var write_status Output_Status
	written, write_status = output_text(workspace, count_value, "(")
	if write_status != OUTPUT_STATUS_OK {
		return written, OUTPUT_STATUS_TOO_LARGE
	}
	written, write_status = output_float_bits(workspace, written, real)
	if write_status != OUTPUT_STATUS_OK {
		return written, OUTPUT_STATUS_TOO_LARGE
	}
	if uint64(imaginary)&FLOAT_64_SIGN_MASK == bits.WORD_64_MINIMUM {
		written, write_status = output_text(workspace, written, "+")
		if write_status != OUTPUT_STATUS_OK {
			return written, OUTPUT_STATUS_TOO_LARGE
		}
	}
	written, write_status = output_float_bits(workspace, written, imaginary)
	if write_status != OUTPUT_STATUS_OK {
		return written, OUTPUT_STATUS_TOO_LARGE
	}
	written, status = output_text(workspace, written, "i)")
	return written, status
}

func output_float_bits(
	workspace_state Workspace_Pointer, count_value Output_Count_Staged,
	encoding Float,
) (_ Output_Count_Staged, status Output_Status) {
	defer func() { Output_Status_Invariants(status, "output_float_bits.status") }()
	Workspace_Pointer_Invariants(workspace_state, "output_float_bits.workspace_state")
	Output_Count_Staged_Invariants(count_value, "output_float_bits.count_value")
	Float_Invariants(encoding, "output_float_bits.encoding")
	workspace := (Workspace_Pointer)(workspace_state)
	count := count_value
	defer func() { Output_Count_Staged_Invariants(count, "output_float_bits.written") }()
	if float64_nan(encoding) {
		count, status = output_text(workspace, count, "NaN")
		return count, status
	}
	float_value := &workspace.Float_Parse.Values[big.FLOAT_RAT_RESULT_INDEX]
	float_text := (*big.Float_Text_Workspace)(&workspace.Float_Text)
	// Reset semantics must not discard caller-owned bounded mantissa storage.
	words := float_value.Mantissa.Words
	*float_value = big.Float{}
	float_value.Mantissa.Words = words
	set_status := big.Float_Set_Float_64_Bits(
		float_value, big.Float_64_Bits(encoding),
	)
	aver.Always(
		set_status == big.STATUS_OK,
		"A non-NaN binary64 encoding is always a valid shared big float.",
	)
	start := int(count.Value.(Output_Count_Staged_Value))
	formatted_count, text_status := big.Float_Text_Into(
		big.Text(workspace.Output[start:]), float_value,
		big.FLOAT_TEXT_FORMAT_DECIMAL_GENERAL, big.FLOAT_TEXT_PRECISION_MINIMUM,
		float_text,
	)
	if text_status == big.STATUS_DESTINATION_TOO_SMALL {
		return count, OUTPUT_STATUS_TOO_LARGE
	}
	aver.Always(
		text_status == big.STATUS_OK,
		"Validated shortest binary64 formatting has no other failure.",
	)
	count = Output_Count_Staged{
		Value: count.Value.(Output_Count_Staged_Value) +
			Output_Count_Staged_Value(formatted_count),
	}
	return count, OUTPUT_STATUS_OK
}

func output_bytes(
	workspace_state Workspace_Pointer, count_value Output_Count_Staged,
	source Text,
) (_ Output_Count_Staged, status Output_Status) {
	defer func() { Output_Status_Invariants(status, "output_bytes.status") }()
	Workspace_Pointer_Invariants(workspace_state, "output_bytes.workspace_state")
	Output_Count_Staged_Invariants(count_value, "output_bytes.count_value")
	Text_Invariants(source, "output_bytes.source")
	written := count_value
	defer func() { Output_Count_Staged_Invariants(written, "output_bytes.written") }()
	workspace := (Workspace_Pointer)(workspace_state)
	count := int(count_value.Value.(Output_Count_Staged_Value))
	if count > len(workspace.Output)-len(source) {
		return count_value, OUTPUT_STATUS_TOO_LARGE
	}
	copy(workspace.Output[count:], source)
	written = Output_Count_Staged{
		Value: count_value.Value.(Output_Count_Staged_Value) +
			Output_Count_Staged_Value(len(source)),
	}
	return written, OUTPUT_STATUS_OK
}
