// Package csv implements bounded comma-separated records on caller-owned storage.
package csv

import (
	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/sim/aver/default"
	"local/james-orcales/shared/unicode/ucd"
	"local/james-orcales/shared/unicode/utf8"
)

// ENCODED_SIZE_MAXIMUM follows the repository byte-slice boundary.
const ENCODED_SIZE_MAXIMUM = bytes.SLICE_SIZE_MAXIMUM

// DECODED_SIZE_MAXIMUM follows the largest unquoted input field.
const DECODED_SIZE_MAXIMUM = bytes.SLICE_SIZE_MAXIMUM

// FIELD_VALUE_SIZE_MAXIMUM follows the repository byte-slice boundary.
const FIELD_VALUE_SIZE_MAXIMUM = bytes.SLICE_SIZE_MAXIMUM

// FIELD_COUNT_MAXIMUM includes every empty field around every source separator.
const FIELD_COUNT_MAXIMUM = bytes.SLICES_COUNT_MAXIMUM

// LINE_MAXIMUM is the final line where bounded input can still hold an error byte.
const LINE_MAXIMUM = ENCODED_SIZE_MAXIMUM

// NEXT_LINE_MAXIMUM admits terminal position after input made only of line endings.
const NEXT_LINE_MAXIMUM = bytes.SLICES_COUNT_MAXIMUM

// COLUMN_MAXIMUM includes the position after the largest unterminated line.
const COLUMN_MAXIMUM = bytes.SLICES_COUNT_MAXIMUM

// LINE_FIRST is the first one-based source line.
const LINE_FIRST = bytes.SLICE_SIZE_MINIMUM + 1

// COLUMN_FIRST is the first one-based byte column.
const COLUMN_FIRST = bytes.SLICE_SIZE_MINIMUM + 1

// FIELDS_PER_RECORD_UNCHECKED disables record-width validation.
const FIELDS_PER_RECORD_UNCHECKED Fields_Per_Record = -1

// FIELDS_PER_RECORD_INFERRED learns width from the first decoded record.
const FIELDS_PER_RECORD_INFERRED Fields_Per_Record = 0

// STANDARD_DELIMITER follows RFC 4180.
const STANDARD_DELIMITER Delimiter = ','

// COMMENT_DISABLED leaves every nonblank line visible to the parser.
const COMMENT_DISABLED Comment = 0

// QUOTE delimits and escapes CSV fields.
const QUOTE byte = '"'

// CARRIAGE_RETURN participates in carriage-return line feeds.
const CARRIAGE_RETURN byte = '\r'

// LINE_FEED terminates standard encoded records.
const LINE_FEED byte = '\n'

// SPACE starts fields that require quoting and participates in Unicode trimming.
const SPACE byte = ' '

// ESCAPED_QUOTE_SIZE doubles a quote inside a quoted field.
const ESCAPED_QUOTE_SIZE = 2

// LINE_FEED_SIZE is the standard record terminator width.
const LINE_FEED_SIZE = 1

// CRLF_SIZE is the alternate record terminator width.
const CRLF_SIZE = 2

// QUOTED_FIELD_BOUNDARY_SIZE includes opening and closing quotes.
const QUOTED_FIELD_BOUNDARY_SIZE = 2

// POSTGRES_TERMINATOR_SIZE is the two-byte field requiring forced quotes.
const POSTGRES_TERMINATOR_SIZE = 2

// ENCODED_FIELD_SIZE_MAXIMUM includes doubling every byte plus quote boundaries.
const ENCODED_FIELD_SIZE_MAXIMUM = FIELD_VALUE_SIZE_MAXIMUM*ESCAPED_QUOTE_SIZE +
	QUOTED_FIELD_BOUNDARY_SIZE

// SCALAR_STORAGE_SIZE keeps caller-mutated policy invalidity status-reportable.
const SCALAR_STORAGE_SIZE = bytes.SLICE_SIZE_MINIMUM + 1

// SCALAR_STORAGE_INDEX avoids a second representation for fixed scalar storage.
const SCALAR_STORAGE_INDEX = bytes.SLICE_SIZE_MINIMUM

// DELIMITER_CHARACTER_MINIMUM excludes disabled zero from validated delimiters.
const DELIMITER_CHARACTER_MINIMUM int32 = int32(bytes.SLICE_SIZE_MINIMUM + 1)

// STATUS_OK means the requested operation completed.
const STATUS_OK = 0

// STATUS_OUTPUT_TOO_SMALL means caller byte storage cannot hold the next field.
const STATUS_OUTPUT_TOO_SMALL = STATUS_OK + 1

// STATUS_STORAGE_INVALID means encoded and decoded byte storage overlap.
const STATUS_STORAGE_INVALID = STATUS_OUTPUT_TOO_SMALL + 1

// STATUS_CONFIGURATION_INVALID means delimiter or comment state is unusable.
const STATUS_CONFIGURATION_INVALID = STATUS_STORAGE_INVALID + 1

// STATUS_RECORD_TOO_LARGE means bounded fields combine beyond output capacity.
const STATUS_RECORD_TOO_LARGE = STATUS_CONFIGURATION_INVALID + 1

// STATUS_END means no record remains after ignored lines.
const STATUS_END = STATUS_RECORD_TOO_LARGE + 1

// STATUS_INPUT_INVALID means quote placement violated the selected dialect.
const STATUS_INPUT_INVALID = STATUS_END + 1

// STATUS_FIELD_COUNT_INVALID means a record differs from the configured width.
const STATUS_FIELD_COUNT_INVALID = STATUS_INPUT_INVALID + 1

// STATUS_FIELDS_TOO_SMALL means caller field slots cannot hold the next field.
const STATUS_FIELDS_TOO_SMALL = STATUS_FIELD_COUNT_INVALID + 1

// Delimiter_Character excludes disabled sentinel from validated accessor results.
type Delimiter_Character int32

// Delimiter_Character_Invariants keeps accessor output inside Unicode scalar bounds.
func Delimiter_Character_Invariants(
	value Delimiter_Character, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int32(
			int32(value), DELIMITER_CHARACTER_MINIMUM,
			utf8.DECODED_CHARACTER_MAXIMUM,
		).
		Ensure()
}

// Comment_Character retains zero because comments can be disabled.
type Comment_Character int32

// Comment_Character_Invariants keeps disabled and Unicode results bounded.
func Comment_Character_Invariants(
	value Comment_Character, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int32(
			int32(value), int32(bytes.SLICE_SIZE_MINIMUM),
			utf8.DECODED_CHARACTER_MAXIMUM,
		).
		Ensure()
}

// Delimiter retains malicious scalar input until construction can return status.
type Delimiter int32

// Delimiter_Invariants covers storage domain before validation.
func Delimiter_Invariants(value Delimiter, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), bits.INTEGER_32_MINIMUM, bits.INTEGER_32_MAXIMUM).
		Ensure()
}

// Comment retains malicious scalar input until construction can return status.
type Comment int32

// Comment_Invariants covers storage domain before validation.
func Comment_Invariants(value Comment, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), bits.INTEGER_32_MINIMUM, bits.INTEGER_32_MAXIMUM).
		Ensure()
}

// Boolean carries one explicit dialect or parser decision.
type Boolean bool

// Boolean_Invariants reaches both decision outcomes.
func Boolean_Invariants(value Boolean, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "A CSV decision is positive.").
		Ensure()
}

// Lazy_Quotes gives permissive quote policy independent coverage identity.
type Lazy_Quotes bool

// Lazy_Quotes_Invariants covers strict and permissive policy.
func Lazy_Quotes_Invariants(value Lazy_Quotes, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Lazy quote parsing is enabled.").
		Ensure()
}

// Trim_Leading_Space gives whitespace policy independent coverage identity.
type Trim_Leading_Space bool

// Trim_Leading_Space_Invariants covers retained and removed prefixes.
func Trim_Leading_Space_Invariants(
	value Trim_Leading_Space, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Leading space trimming is enabled.").
		Ensure()
}

// Use_CRLF gives line-ending policy independent coverage identity.
type Use_CRLF bool

// Use_CRLF_Invariants covers LF and CRLF output.
func Use_CRLF_Invariants(value Use_CRLF, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "CRLF encoding is enabled.").
		Ensure()
}

// Fields_Per_Record selects unchecked, inferred, or exact record width.
type Fields_Per_Record int

// Fields_Per_Record_Invariants bounds width by maximum decoded field slots.
func Fields_Per_Record_Invariants(
	value Fields_Per_Record, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), int(FIELDS_PER_RECORD_UNCHECKED), FIELD_COUNT_MAXIMUM,
		).
		Ensure()
}

// Configuration_Input is caller policy before delimiter validation.
type Configuration_Input struct {
	// Delimiter separates adjacent fields.
	Delimiter Delimiter
	// Comment skips matching physical lines when nonzero.
	Comment Comment
	// Fields_Per_Record selects width validation.
	Fields_Per_Record Fields_Per_Record
	// Lazy_Quotes admits otherwise malformed quotes.
	Lazy_Quotes Lazy_Quotes
	// Trim_Leading_Space removes Unicode space before each field.
	Trim_Leading_Space Trim_Leading_Space
	// Use_CRLF selects carriage-return line feeds while encoding.
	Use_CRLF Use_CRLF
}

// Configuration_Input_Invariants composes complete unvalidated scalar domains.
func Configuration_Input_Invariants(
	value Configuration_Input, namespace aver.Namespace,
) {
	Delimiter_Invariants(value.Delimiter, namespace)
	Comment_Invariants(value.Comment, namespace)
	Fields_Per_Record_Invariants(value.Fields_Per_Record, namespace)
	Lazy_Quotes_Invariants(value.Lazy_Quotes, namespace)
	Trim_Leading_Space_Invariants(value.Trim_Leading_Space, namespace)
	Use_CRLF_Invariants(value.Use_CRLF, namespace)
}

// Delimiter_Storage avoids owned memory for maximum UTF-8 delimiter width.
type Delimiter_Storage [utf8.CHARACTER_SIZE_MAXIMUM]byte

// Delimiter_Storage_Invariants fixes capacity while operations validate content.
func Delimiter_Storage_Invariants(value Delimiter_Storage, _ aver.Namespace) {
	aver.Always(
		len(value) == utf8.CHARACTER_SIZE_MAXIMUM,
		"Delimiter storage holds one maximum UTF-8 encoding.",
	)
}

// Comment_Storage avoids owned memory for maximum UTF-8 comment width.
type Comment_Storage [utf8.CHARACTER_SIZE_MAXIMUM]byte

// Comment_Storage_Invariants fixes capacity while operations validate content.
func Comment_Storage_Invariants(value Comment_Storage, _ aver.Namespace) {
	aver.Always(
		len(value) == utf8.CHARACTER_SIZE_MAXIMUM,
		"Comment storage holds one maximum UTF-8 encoding.",
	)
}

// Delimiter_Size_Storage keeps corrupt caller state representable for status return.
type Delimiter_Size_Storage [SCALAR_STORAGE_SIZE]uint8

// Delimiter_Size_Storage_Invariants leaves content checks on operation paths.
func Delimiter_Size_Storage_Invariants(
	value Delimiter_Size_Storage, _ aver.Namespace,
) {
	aver.Always(
		len(value) == SCALAR_STORAGE_SIZE,
		"Delimiter size storage keeps validity outside structural invariants.",
	)
}

// Comment_Size_Storage keeps disabled and corrupt state status-reportable.
type Comment_Size_Storage [SCALAR_STORAGE_SIZE]uint8

// Comment_Size_Storage_Invariants leaves content checks on operation paths.
func Comment_Size_Storage_Invariants(
	value Comment_Size_Storage, _ aver.Namespace,
) {
	aver.Always(
		len(value) == SCALAR_STORAGE_SIZE,
		"Comment size storage keeps disabled state outside structural invariants.",
	)
}

// Fields_Per_Record_Storage keeps corrupt caller width status-reportable.
type Fields_Per_Record_Storage [SCALAR_STORAGE_SIZE]int

// Fields_Per_Record_Storage_Invariants leaves content checks on operation paths.
func Fields_Per_Record_Storage_Invariants(
	value Fields_Per_Record_Storage, _ aver.Namespace,
) {
	aver.Always(
		len(value) == SCALAR_STORAGE_SIZE,
		"Record width storage keeps caller mutation status-reportable.",
	)
}

// Lazy_Quotes_Storage avoids hidden mutable policy.
type Lazy_Quotes_Storage [SCALAR_STORAGE_SIZE]bool

// Lazy_Quotes_Storage_Invariants fixes visible policy storage shape.
func Lazy_Quotes_Storage_Invariants(
	value Lazy_Quotes_Storage, _ aver.Namespace,
) {
	aver.Always(
		len(value) == SCALAR_STORAGE_SIZE,
		"Lazy quote storage avoids hidden mutable policy.",
	)
}

// Trim_Leading_Space_Storage avoids hidden mutable policy.
type Trim_Leading_Space_Storage [SCALAR_STORAGE_SIZE]bool

// Trim_Leading_Space_Storage_Invariants fixes visible policy storage shape.
func Trim_Leading_Space_Storage_Invariants(
	value Trim_Leading_Space_Storage, _ aver.Namespace,
) {
	aver.Always(
		len(value) == SCALAR_STORAGE_SIZE,
		"Trim storage avoids hidden mutable policy.",
	)
}

// Use_CRLF_Storage avoids hidden mutable policy.
type Use_CRLF_Storage [SCALAR_STORAGE_SIZE]bool

// Use_CRLF_Storage_Invariants fixes visible policy storage shape.
func Use_CRLF_Storage_Invariants(
	value Use_CRLF_Storage, _ aver.Namespace,
) {
	aver.Always(
		len(value) == SCALAR_STORAGE_SIZE,
		"Line-ending storage avoids hidden mutable policy.",
	)
}

// Configuration stores validated policy entirely by value.
type Configuration struct {
	// Delimiter retains the encoded field separator.
	Delimiter Delimiter_Storage
	// Delimiter_Size selects its meaningful prefix.
	Delimiter_Size Delimiter_Size_Storage
	// Comment retains the encoded line prefix when enabled.
	Comment Comment_Storage
	// Comment_Size is zero when comments are disabled.
	Comment_Size Comment_Size_Storage
	// Fields_Per_Record selects width validation.
	Fields_Per_Record Fields_Per_Record_Storage
	// Lazy_Quotes admits otherwise malformed quotes.
	Lazy_Quotes Lazy_Quotes_Storage
	// Trim_Leading_Space removes Unicode space before fields.
	Trim_Leading_Space Trim_Leading_Space_Storage
	// Use_CRLF selects carriage-return line feeds while encoding.
	Use_CRLF Use_CRLF_Storage
}

// Configuration_Invariants composes storage without assuming caller validity.
func Configuration_Invariants(value Configuration, namespace aver.Namespace) {
	Delimiter_Storage_Invariants(value.Delimiter, namespace)
	Delimiter_Size_Storage_Invariants(value.Delimiter_Size, namespace)
	Comment_Storage_Invariants(value.Comment, namespace)
	Comment_Size_Storage_Invariants(value.Comment_Size, namespace)
	Fields_Per_Record_Storage_Invariants(value.Fields_Per_Record, namespace)
	Lazy_Quotes_Storage_Invariants(value.Lazy_Quotes, namespace)
	Trim_Leading_Space_Storage_Invariants(value.Trim_Leading_Space, namespace)
	Use_CRLF_Storage_Invariants(value.Use_CRLF, namespace)
}

// Configuration_Valid reports complete storage validation without an error interface.
type Configuration_Valid bool

// Configuration_Valid_Invariants reaches valid and corrupted configurations.
func Configuration_Valid_Invariants(
	value Configuration_Valid, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "CSV configuration storage is valid.").
		Ensure()
}

// Configuration_Status reports construction success or invalid policy.
type Configuration_Status uint8

// Configuration_Status_Invariants lists both construction outcomes.
func Configuration_Status_Invariants(
	value Configuration_Status, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_CONFIGURATION_INVALID),
		).
		Ensure()
}

// Field_Value is decoded or caller-supplied field bytes.
type Field_Value []byte

// Field_Value_Invariants follows the repository byte-slice boundary.
func Field_Value_Invariants(value Field_Value, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), bytes.SLICE_SIZE_MINIMUM, FIELD_VALUE_SIZE_MAXIMUM,
		).
		Ensure()
}

// Line is a one-based field or error line, with zero meaning absent.
type Line int

// Line_Invariants covers absent through final bounded input position.
func Line_Invariants(value Line, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), bytes.SLICE_SIZE_MINIMUM, LINE_MAXIMUM).
		Ensure()
}

// Column is a one-based byte column, with zero meaning absent.
type Column int

// Column_Invariants covers absent through the boundary after bounded input.
func Column_Invariants(value Column, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), bytes.SLICE_SIZE_MINIMUM, COLUMN_MAXIMUM).
		Ensure()
}

// Field retains decoded bytes and their source start.
type Field struct {
	// Value borrows caller input or decoded storage.
	Value Field_Value
	// Line is one-based for decoded output and zero for encode-only input.
	Line Line
	// Column is one-based for decoded output and zero for encode-only input.
	Column Column
}

// Field_Invariants composes value and optional position.
func Field_Invariants(value Field, namespace aver.Namespace) {
	Field_Value_Invariants(value.Value, namespace)
	Line_Invariants(value.Line, namespace)
	Column_Invariants(value.Column, namespace)
}

// Fields is caller-owned record input or decoded output slots.
type Fields []Field

// Fields_Invariants follows the separator-derived maximum field count.
func Fields_Invariants(value Fields, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), bytes.SLICES_COUNT_MINIMUM, FIELD_COUNT_MAXIMUM).
		Ensure()
}

// Encoded is bounded CSV input or writable output.
type Encoded []byte

// Encoded_Invariants follows the repository byte-slice boundary.
func Encoded_Invariants(value Encoded, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), bytes.SLICE_SIZE_MINIMUM, ENCODED_SIZE_MAXIMUM).
		Ensure()
}

// Decoded is caller-owned unescaped field storage.
type Decoded []byte

// Decoded_Invariants follows the largest bounded input contraction.
func Decoded_Invariants(value Decoded, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), bytes.SLICE_SIZE_MINIMUM, DECODED_SIZE_MAXIMUM).
		Ensure()
}

// Encoded_Count is exact required or written record bytes, or zero on refusal.
type Encoded_Count int

// Encoded_Count_Invariants follows bounded output.
func Encoded_Count_Invariants(value Encoded_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), bytes.SLICE_SIZE_MINIMUM, ENCODED_SIZE_MAXIMUM).
		Ensure()
}

// Field_Count is decoded or required field slots.
type Field_Count int

// Field_Count_Invariants follows the separator-derived maximum.
func Field_Count_Invariants(value Field_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), bytes.SLICES_COUNT_MINIMUM, FIELD_COUNT_MAXIMUM).
		Ensure()
}

// Consumed_Count is source bytes through one record or ignored-input end.
type Consumed_Count int

// Consumed_Count_Invariants follows bounded source.
func Consumed_Count_Invariants(value Consumed_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), bytes.SLICE_SIZE_MINIMUM, ENCODED_SIZE_MAXIMUM).
		Ensure()
}

// Parse_Position names one error location or the absent zero position.
type Parse_Position struct {
	// Line is one-based when present.
	Line Line
	// Column is a one-based byte position when present.
	Column Column
}

// Parse_Position_Invariants keeps absent and present coordinates bounded together.
func Parse_Position_Invariants(value Parse_Position, namespace aver.Namespace) {
	Line_Invariants(value.Line, namespace)
	Column_Invariants(value.Column, namespace)
}

// Expected_Fields_Per_Record_Storage keeps learned width caller-owned and visible.
type Expected_Fields_Per_Record_Storage [SCALAR_STORAGE_SIZE]int

// Expected_Fields_Per_Record_Storage_Invariants leaves validity on decode paths.
func Expected_Fields_Per_Record_Storage_Invariants(
	value Expected_Fields_Per_Record_Storage, _ aver.Namespace,
) {
	aver.Always(
		len(value) == SCALAR_STORAGE_SIZE,
		"Learned record width storage keeps caller mutation status-reportable.",
	)
}

// Next_Line_Storage keeps cross-call location caller-owned and visible.
type Next_Line_Storage [SCALAR_STORAGE_SIZE]int

// Next_Line_Storage_Invariants leaves validity on decode paths.
func Next_Line_Storage_Invariants(value Next_Line_Storage, _ aver.Namespace) {
	aver.Always(
		len(value) == SCALAR_STORAGE_SIZE,
		"Line storage keeps caller mutation status-reportable.",
	)
}

// Decoder retains only caller-owned cross-record policy state.
type Decoder struct {
	// Configuration remains immutable after construction.
	Configuration Configuration
	// Fields_Per_Record stores unchecked, inferred, or learned width.
	Fields_Per_Record Expected_Fields_Per_Record_Storage
	// Next_Line is the first line in the next caller source suffix.
	Next_Line Next_Line_Storage
}

// Decoder_Invariants composes representable initialized and zero states.
func Decoder_Invariants(value *Decoder, namespace aver.Namespace) {
	aver.Always(value != nil, "Decoder storage exists.")
	Configuration_Invariants(value.Configuration, namespace)
	Expected_Fields_Per_Record_Storage_Invariants(value.Fields_Per_Record, namespace)
	Next_Line_Storage_Invariants(value.Next_Line, namespace)
}

// Decoder_Status reports initialized or invalid configuration.
type Decoder_Status uint8

// Decoder_Status_Invariants lists both construction outcomes.
func Decoder_Status_Invariants(value Decoder_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_CONFIGURATION_INVALID),
		).
		Ensure()
}

// Decoder_Valid separates caller corruption from structural storage bounds.
type Decoder_Valid bool

// Decoder_Valid_Invariants covers usable and corrupt caller state.
func Decoder_Valid_Invariants(value Decoder_Valid, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Decoder storage is valid.").
		Ensure()
}

// Size_Status reports exact size, invalid configuration, or aggregate overflow.
type Size_Status uint8

// Size_Status_Invariants lists every sizing outcome.
func Size_Status_Invariants(value Size_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_CONFIGURATION_INVALID),
			uint8(STATUS_RECORD_TOO_LARGE),
		).
		Ensure()
}

// Encode_Status adds byte storage failures to size outcomes.
type Encode_Status uint8

// Encode_Status_Invariants excludes decode-only outcomes from shared status values.
func Encode_Status_Invariants(value Encode_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), uint8(STATUS_OK), uint8(STATUS_RECORD_TOO_LARGE)).
		Ensure()
}

// Decode_Status reports every parser, storage, and configuration outcome.
type Decode_Status uint8

// Decode_Status_Invariants covers its contiguous outcome range.
func Decode_Status_Invariants(value Decode_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Holed_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_FIELDS_TOO_SMALL),
			uint8(STATUS_RECORD_TOO_LARGE), uint8(STATUS_RECORD_TOO_LARGE),
			uint8(STATUS_RECORD_TOO_LARGE),
		).
		Ensure()
}

// Parser keeps temporary state bounded without heap-owned buffers.
type Parser struct {
	// Source_Position prevents rescanning bytes before current field.
	Source_Position [SCALAR_STORAGE_SIZE]int
	// Decoded_Count prevents overwriting preceding caller output.
	Decoded_Count [SCALAR_STORAGE_SIZE]int
	// Line preserves physical position across multiline fields.
	Line [SCALAR_STORAGE_SIZE]int
	// Column preserves byte position across delimiter widths.
	Column [SCALAR_STORAGE_SIZE]int
	// Record_Done separates field delimiter from record termination.
	Record_Done [SCALAR_STORAGE_SIZE]bool
}

// Parser_Invariants fixes storage shape while public output checks scalar bounds.
func Parser_Invariants(value Parser, _ aver.Namespace) {
	aver.Always(
		len(value.Source_Position) == SCALAR_STORAGE_SIZE,
		"Parser source position stays in caller-independent scalar storage.",
	)
	aver.Always(
		len(value.Decoded_Count) == SCALAR_STORAGE_SIZE,
		"Parser decoded position stays in caller-independent scalar storage.",
	)
	aver.Always(
		len(value.Line) == SCALAR_STORAGE_SIZE,
		"Parser line stays in caller-independent scalar storage.",
	)
	aver.Always(
		len(value.Column) == SCALAR_STORAGE_SIZE,
		"Parser column stays in caller-independent scalar storage.",
	)
	aver.Always(
		len(value.Record_Done) == SCALAR_STORAGE_SIZE,
		"Parser completion stays in caller-independent scalar storage.",
	)
}

// Decode_Result avoids raw internal returns without widening public state.
type Decode_Result struct {
	// Field_Count crosses parser boundary without raw scalar return.
	Field_Count [SCALAR_STORAGE_SIZE]int
	// Consumed crosses parser boundary without raw scalar return.
	Consumed [SCALAR_STORAGE_SIZE]int
	// Error_Line keeps absent zero representable beside physical lines.
	Error_Line [SCALAR_STORAGE_SIZE]int
	// Error_Column keeps absent zero representable beside byte columns.
	Error_Column [SCALAR_STORAGE_SIZE]int
	// Status keeps parser outcomes independent from public conversion.
	Status [SCALAR_STORAGE_SIZE]uint8
}

// Decode_Result_Invariants fixes scalar storage shape before public conversion.
func Decode_Result_Invariants(value Decode_Result, _ aver.Namespace) {
	aver.Always(
		len(value.Field_Count) == SCALAR_STORAGE_SIZE,
		"Decode field count stays in caller-independent scalar storage.",
	)
	aver.Always(
		len(value.Consumed) == SCALAR_STORAGE_SIZE,
		"Decode consumed count stays in caller-independent scalar storage.",
	)
	aver.Always(
		len(value.Error_Line) == SCALAR_STORAGE_SIZE,
		"Decode error line stays in caller-independent scalar storage.",
	)
	aver.Always(
		len(value.Error_Column) == SCALAR_STORAGE_SIZE,
		"Decode error column stays in caller-independent scalar storage.",
	)
	aver.Always(
		len(value.Status) == SCALAR_STORAGE_SIZE,
		"Decode status stays in caller-independent scalar storage.",
	)
}

func new_decode_result[
	Field_Count_Value ~int, Consumed_Value ~int,
	Error_Line_Value ~int, Error_Column_Value ~int, Status_Value ~int | ~uint8,
](
	field_count Field_Count_Value, consumed Consumed_Value,
	error_line Error_Line_Value, error_column Error_Column_Value, status Status_Value,
) (result Decode_Result) {
	defer func() { Decode_Result_Invariants(result, "new_decode_result.result") }()
	result.Field_Count[SCALAR_STORAGE_INDEX] = int(field_count)
	result.Consumed[SCALAR_STORAGE_INDEX] = int(consumed)
	result.Error_Line[SCALAR_STORAGE_INDEX] = int(error_line)
	result.Error_Column[SCALAR_STORAGE_INDEX] = int(error_column)
	result.Status[SCALAR_STORAGE_INDEX] = uint8(status)
	return result
}

// Field_Decode_Result keeps parser refusal state in bounded value storage.
type Field_Decode_Result struct {
	// Parser carries caller-storage offsets without hidden buffers.
	Parser Parser
	// Error_Line keeps absent zero representable on successful fields.
	Error_Line [SCALAR_STORAGE_SIZE]int
	// Error_Column keeps absent zero representable on successful fields.
	Error_Column [SCALAR_STORAGE_SIZE]int
	// Status keeps field outcomes independent from record outcomes.
	Status [SCALAR_STORAGE_SIZE]uint8
}

// Field_Decode_Result_Invariants fixes storage shape between parser stages.
func Field_Decode_Result_Invariants(
	value Field_Decode_Result, namespace aver.Namespace,
) {
	Parser_Invariants(value.Parser, namespace)
	aver.Always(
		len(value.Error_Line) == SCALAR_STORAGE_SIZE,
		"Field error line stays in caller-independent scalar storage.",
	)
	aver.Always(
		len(value.Error_Column) == SCALAR_STORAGE_SIZE,
		"Field error column stays in caller-independent scalar storage.",
	)
	aver.Always(
		len(value.Status) == SCALAR_STORAGE_SIZE,
		"Field status stays in caller-independent scalar storage.",
	)
}

func new_field_decode_result[
	Error_Line_Value ~int, Error_Column_Value ~int, Status_Value ~int | ~uint8,
](
	parser Parser, error_line Error_Line_Value,
	error_column Error_Column_Value, status Status_Value,
) (result Field_Decode_Result) {
	defer func() {
		Field_Decode_Result_Invariants(result, "new_field_decode_result.result")
	}()
	Parser_Invariants(parser, "new_field_decode_result.parser")
	result.Parser = parser
	result.Error_Line[SCALAR_STORAGE_INDEX] = int(error_line)
	result.Error_Column[SCALAR_STORAGE_INDEX] = int(error_column)
	result.Status[SCALAR_STORAGE_INDEX] = uint8(status)
	return result
}

func new_field_position_result[Status_Value ~int | ~uint8](
	parser Parser, status Status_Value,
) (result Field_Decode_Result) {
	defer func() {
		Field_Decode_Result_Invariants(result, "new_field_position_result.result")
	}()
	Parser_Invariants(parser, "new_field_position_result.parser")
	return new_field_decode_result(
		parser, parser.Line[SCALAR_STORAGE_INDEX], parser.Column[SCALAR_STORAGE_INDEX],
		status,
	)
}

// New_Configuration validates policy before publishing encoded delimiter storage.
func New_Configuration(
	input Configuration_Input,
) (configuration Configuration, status Configuration_Status) {
	defer func() {
		Configuration_Invariants(configuration, "New_Configuration.configuration")
		Configuration_Status_Invariants(status, "New_Configuration.status")
	}()
	Configuration_Input_Invariants(input, "New_Configuration.input")
	if !bool(character_allowed(input.Delimiter)) {
		return Configuration{}, STATUS_CONFIGURATION_INVALID
	}
	if input.Comment != COMMENT_DISABLED {
		if !bool(character_allowed(input.Comment)) {
			return Configuration{}, STATUS_CONFIGURATION_INVALID
		}
		if int32(input.Comment) == int32(input.Delimiter) {
			return Configuration{}, STATUS_CONFIGURATION_INVALID
		}
	}
	delimiter_size := utf8.Encode_Character(
		utf8.Bytes(configuration.Delimiter[:]), utf8.Character(input.Delimiter),
	)
	configuration.Delimiter_Size[SCALAR_STORAGE_INDEX] = uint8(delimiter_size)
	if input.Comment != COMMENT_DISABLED {
		comment_size := utf8.Encode_Character(
			utf8.Bytes(configuration.Comment[:]), utf8.Character(input.Comment),
		)
		configuration.Comment_Size[SCALAR_STORAGE_INDEX] = uint8(comment_size)
	}
	configuration.Fields_Per_Record[SCALAR_STORAGE_INDEX] = int(input.Fields_Per_Record)
	configuration.Lazy_Quotes[SCALAR_STORAGE_INDEX] = bool(input.Lazy_Quotes)
	configuration.Trim_Leading_Space[SCALAR_STORAGE_INDEX] = bool(input.Trim_Leading_Space)
	configuration.Use_CRLF[SCALAR_STORAGE_INDEX] = bool(input.Use_CRLF)
	return configuration, STATUS_OK
}

// Standard_Configuration returns independent immutable-by-value RFC policy.
func Standard_Configuration() (configuration Configuration) {
	defer func() {
		Configuration_Invariants(configuration, "Standard_Configuration.configuration")
	}()
	configuration, status := New_Configuration(Configuration_Input{
		Delimiter: STANDARD_DELIMITER, Fields_Per_Record: FIELDS_PER_RECORD_INFERRED,
	})
	aver.Always(status == STATUS_OK, "Standard CSV configuration is valid.")
	return configuration
}

// Configuration_Delimiter returns the validated delimiter character.
func Configuration_Delimiter(
	configuration Configuration,
) (delimiter Delimiter_Character) {
	defer func() {
		Delimiter_Character_Invariants(delimiter, "Configuration_Delimiter.delimiter")
	}()
	Configuration_Invariants(configuration, "Configuration_Delimiter.configuration")
	aver.Always(
		bool(configuration_valid(configuration)),
		"Delimiter configuration is valid.",
	)
	delimiter_size := configuration.Delimiter_Size[SCALAR_STORAGE_INDEX]
	character, _ := utf8.Decode_Character(
		utf8.Bytes(configuration.Delimiter[:delimiter_size]),
	)
	return Delimiter_Character(character)
}

// Configuration_Comment returns zero when comments are disabled.
func Configuration_Comment(configuration Configuration) (comment Comment_Character) {
	defer func() {
		Comment_Character_Invariants(comment, "Configuration_Comment.comment")
	}()
	Configuration_Invariants(configuration, "Configuration_Comment.configuration")
	aver.Always(
		bool(configuration_valid(configuration)),
		"Comment configuration is valid.",
	)
	comment_size := configuration.Comment_Size[SCALAR_STORAGE_INDEX]
	if comment_size == 0 {
		return Comment_Character(COMMENT_DISABLED)
	}
	character, _ := utf8.Decode_Character(
		utf8.Bytes(configuration.Comment[:comment_size]),
	)
	return Comment_Character(character)
}

// Decoder_Init leaves invalid policy unable to mutate caller state.
func Decoder_Init(
	decoder *Decoder, configuration Configuration,
) (status Decoder_Status) {
	defer func() {
		Decoder_Invariants(decoder, "Decoder_Init.decoder_result")
		Decoder_Status_Invariants(status, "Decoder_Init.status")
	}()
	Decoder_Invariants(decoder, "Decoder_Init.decoder")
	Configuration_Invariants(configuration, "Decoder_Init.configuration")
	if !bool(configuration_valid(configuration)) {
		return STATUS_CONFIGURATION_INVALID
	}
	decoder.Configuration = configuration
	decoder.Fields_Per_Record[SCALAR_STORAGE_INDEX] =
		configuration.Fields_Per_Record[SCALAR_STORAGE_INDEX]
	decoder.Next_Line[SCALAR_STORAGE_INDEX] = LINE_FIRST
	return STATUS_OK
}

// Encoded_Size reports exact storage for one caller record.
func Encoded_Size(
	fields Fields, configuration Configuration,
) (count Encoded_Count, status Size_Status) {
	defer func() {
		Encoded_Count_Invariants(count, "Encoded_Size.count")
		Size_Status_Invariants(status, "Encoded_Size.status")
	}()
	Fields_Invariants(fields, "Encoded_Size.fields")
	Configuration_Invariants(configuration, "Encoded_Size.configuration")
	if !bool(configuration_valid(configuration)) {
		return 0, STATUS_CONFIGURATION_INVALID
	}
	delimiter_size := configuration.Delimiter_Size[SCALAR_STORAGE_INDEX]
	required := int(line_ending_size(configuration.Use_CRLF[SCALAR_STORAGE_INDEX]))
	if len(fields) > bytes.SLICES_COUNT_MINIMUM {
		required += (len(fields) - 1) * int(delimiter_size)
	}
	delimiter := configuration.Delimiter[:delimiter_size]
	for _, field := range fields {
		Field_Value_Invariants(field.Value, "Encoded_Size.field")
		required += int(encoded_field_size(
			field.Value, delimiter, configuration.Use_CRLF[SCALAR_STORAGE_INDEX],
		))
	}
	if required > ENCODED_SIZE_MAXIMUM {
		return 0, STATUS_RECORD_TOO_LARGE
	}
	return Encoded_Count(required), STATUS_OK
}

// Encode_Into checks all refusals before changing caller output.
func Encode_Into(
	destination Encoded, fields Fields, configuration Configuration,
) (count Encoded_Count, status Encode_Status) {
	defer func() {
		Encoded_Count_Invariants(count, "Encode_Into.count")
		Encode_Status_Invariants(status, "Encode_Into.status")
	}()
	Encoded_Invariants(destination, "Encode_Into.destination")
	Fields_Invariants(fields, "Encode_Into.fields")
	Configuration_Invariants(configuration, "Encode_Into.configuration")
	required, size_status := Encoded_Size(fields, configuration)
	if size_status == STATUS_CONFIGURATION_INVALID {
		return 0, STATUS_CONFIGURATION_INVALID
	}
	if size_status == STATUS_RECORD_TOO_LARGE {
		return 0, STATUS_RECORD_TOO_LARGE
	}
	for _, field := range fields {
		if bytes.Overlap(bytes.Slice(destination), bytes.Slice(field.Value)) {
			return 0, STATUS_STORAGE_INVALID
		}
	}
	if len(destination) < int(required) {
		return 0, STATUS_OUTPUT_TOO_SMALL
	}
	encode_unchecked(destination, fields, configuration, required)
	return required, STATUS_OK
}

// Decode_Record_Into parses one record and mutates only caller-owned decoder state.
func Decode_Record_Into(
	destination Decoded, fields Fields, source Encoded, decoder *Decoder,
) (
	field_count Field_Count, consumed Consumed_Count,
	position Parse_Position, status Decode_Status,
) {
	defer func() {
		Field_Count_Invariants(field_count, "Decode_Record_Into.field_count")
		Consumed_Count_Invariants(consumed, "Decode_Record_Into.consumed")
		Parse_Position_Invariants(position, "Decode_Record_Into.position")
		Decode_Status_Invariants(status, "Decode_Record_Into.status")
	}()
	Decoded_Invariants(destination, "Decode_Record_Into.destination")
	Fields_Invariants(fields, "Decode_Record_Into.fields")
	Encoded_Invariants(source, "Decode_Record_Into.source")
	Decoder_Invariants(decoder, "Decode_Record_Into.decoder")
	if !bool(decoder_valid(decoder)) {
		return 0, 0, Parse_Position{}, STATUS_CONFIGURATION_INVALID
	}
	if bytes.Overlap(bytes.Slice(destination), bytes.Slice(source)) {
		return 0, 0, Parse_Position{}, STATUS_STORAGE_INVALID
	}
	result := decode_record_unchecked(destination, fields, source, decoder)
	return Field_Count(result.Field_Count[SCALAR_STORAGE_INDEX]),
		Consumed_Count(result.Consumed[SCALAR_STORAGE_INDEX]),
		Parse_Position{
			Line:   Line(result.Error_Line[SCALAR_STORAGE_INDEX]),
			Column: Column(result.Error_Column[SCALAR_STORAGE_INDEX]),
		},
		Decode_Status(result.Status[SCALAR_STORAGE_INDEX])
}

func decoder_valid(decoder *Decoder) (valid Decoder_Valid) {
	defer func() { Decoder_Valid_Invariants(valid, "decoder_valid.valid") }()
	Decoder_Invariants(decoder, "decoder_valid.decoder")
	if !bool(configuration_valid(decoder.Configuration)) {
		return false
	}
	fields_per_record := decoder.Fields_Per_Record[SCALAR_STORAGE_INDEX]
	if fields_per_record < int(FIELDS_PER_RECORD_UNCHECKED) {
		return false
	}
	if fields_per_record > FIELD_COUNT_MAXIMUM {
		return false
	}
	next_line := decoder.Next_Line[SCALAR_STORAGE_INDEX]
	if next_line < LINE_FIRST {
		return false
	}
	return Decoder_Valid(next_line <= NEXT_LINE_MAXIMUM)
}

func character_allowed[Character_Value ~int32](
	value Character_Value,
) (allowed Boolean) {
	defer func() { Boolean_Invariants(allowed, "character_allowed.allowed") }()
	character := utf8.Character(value)
	if !bool(utf8.Valid_Character(character)) {
		return false
	}
	if character == 0 {
		return false
	}
	if character == utf8.Character(QUOTE) {
		return false
	}
	if character == utf8.Character(CARRIAGE_RETURN) {
		return false
	}
	if character == utf8.Character(LINE_FEED) {
		return false
	}
	if character == utf8.Character(utf8.REPLACEMENT_CHARACTER) {
		return false
	}
	return true
}

func configuration_valid(configuration Configuration) (valid Configuration_Valid) {
	defer func() {
		Configuration_Valid_Invariants(valid, "configuration_valid.valid")
	}()
	Configuration_Invariants(configuration, "configuration_valid.configuration")
	delimiter_size_stored := configuration.Delimiter_Size[SCALAR_STORAGE_INDEX]
	if delimiter_size_stored == 0 {
		return false
	}
	if delimiter_size_stored > utf8.CHARACTER_SIZE_MAXIMUM {
		return false
	}
	delimiter, delimiter_size := utf8.Decode_Character(
		utf8.Bytes(configuration.Delimiter[:delimiter_size_stored]),
	)
	if int(delimiter_size) != int(delimiter_size_stored) {
		return false
	}
	if !bool(character_allowed(int32(delimiter))) {
		return false
	}
	if !delimiter_tail_zero(configuration.Delimiter, int(delimiter_size_stored)) {
		return false
	}
	fields_per_record := configuration.Fields_Per_Record[SCALAR_STORAGE_INDEX]
	if fields_per_record < int(FIELDS_PER_RECORD_UNCHECKED) {
		return false
	}
	if fields_per_record > FIELD_COUNT_MAXIMUM {
		return false
	}
	comment_size_stored := configuration.Comment_Size[SCALAR_STORAGE_INDEX]
	if comment_size_stored > utf8.CHARACTER_SIZE_MAXIMUM {
		return false
	}
	if comment_size_stored == 0 {
		return Configuration_Valid(comment_tail_zero(configuration.Comment, 0))
	}
	comment, comment_size := utf8.Decode_Character(
		utf8.Bytes(configuration.Comment[:comment_size_stored]),
	)
	if int(comment_size) != int(comment_size_stored) {
		return false
	}
	if !bool(character_allowed(int32(comment))) {
		return false
	}
	if comment == delimiter {
		return false
	}
	return Configuration_Valid(
		comment_tail_zero(configuration.Comment, int(comment_size_stored)),
	)
}

// Delimiter_Tail_Zero_State exposes corrupted unused bytes to coverage.
type Delimiter_Tail_Zero_State bool

// Delimiter_Tail_Zero_State_Invariants covers clean and corrupt storage.
func Delimiter_Tail_Zero_State_Invariants(
	value Delimiter_Tail_Zero_State, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Delimiter unused storage is zero.").
		Ensure()
}

func delimiter_tail_zero[Size ~int](
	storage Delimiter_Storage, size Size,
) (zero Delimiter_Tail_Zero_State) {
	defer func() {
		Delimiter_Tail_Zero_State_Invariants(zero, "delimiter_tail_zero.zero")
	}()
	Delimiter_Storage_Invariants(storage, "delimiter_tail_zero.storage")
	for index := int(size); index < len(storage); index++ {
		if storage[index] != 0 {
			return false
		}
	}
	return true
}

// Comment_Tail_Zero_State exposes corrupted unused bytes to coverage.
type Comment_Tail_Zero_State bool

// Comment_Tail_Zero_State_Invariants covers clean and corrupt storage.
func Comment_Tail_Zero_State_Invariants(
	value Comment_Tail_Zero_State, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Comment unused storage is zero.").
		Ensure()
}

func comment_tail_zero[Size ~int](
	storage Comment_Storage, size Size,
) (zero Comment_Tail_Zero_State) {
	defer func() {
		Comment_Tail_Zero_State_Invariants(zero, "comment_tail_zero.zero")
	}()
	Comment_Storage_Invariants(storage, "comment_tail_zero.storage")
	for index := int(size); index < len(storage); index++ {
		if storage[index] != 0 {
			return false
		}
	}
	return true
}

// Line_Ending_Size_Count excludes sizes no configured line ending can produce.
type Line_Ending_Size_Count int

// Line_Ending_Size_Count_Invariants covers LF and CRLF widths.
func Line_Ending_Size_Count_Invariants(
	value Line_Ending_Size_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_Int(int(value), LINE_FEED_SIZE, CRLF_SIZE).
		Ensure()
}

func line_ending_size[Use_CRLF ~bool](
	use_crlf Use_CRLF,
) (size Line_Ending_Size_Count) {
	defer func() {
		Line_Ending_Size_Count_Invariants(size, "line_ending_size.size")
	}()
	if use_crlf {
		return CRLF_SIZE
	}
	return LINE_FEED_SIZE
}

// Encoded_Field_Size_Count includes quote doubling before record refusal.
type Encoded_Field_Size_Count int

// Encoded_Field_Size_Count_Invariants bounds worst-case escaped field growth.
func Encoded_Field_Size_Count_Invariants(
	value Encoded_Field_Size_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), bytes.SLICE_SIZE_MINIMUM, ENCODED_FIELD_SIZE_MAXIMUM,
		).
		Ensure()
}

func encoded_field_size[
	Value ~[]byte, Delimiter ~[]byte, Use_CRLF ~bool,
](
	value Value, delimiter Delimiter, use_crlf Use_CRLF,
) (size Encoded_Field_Size_Count) {
	defer func() {
		Encoded_Field_Size_Count_Invariants(size, "encoded_field_size.size")
	}()
	if !field_needs_quotes(value, delimiter) {
		return Encoded_Field_Size_Count(len(value))
	}
	size = QUOTED_FIELD_BOUNDARY_SIZE
	for _, value_byte := range value {
		switch value_byte {
		case QUOTE:
			size += ESCAPED_QUOTE_SIZE
		case CARRIAGE_RETURN:
			if !use_crlf {
				size++
			}
		case LINE_FEED:
			size += Encoded_Field_Size_Count(line_ending_size(use_crlf))
		default:
			size++
		}
	}
	return size
}

func field_needs_quotes[Value ~[]byte, Delimiter ~[]byte](
	value Value, delimiter Delimiter,
) (needed Boolean) {
	defer func() { Boolean_Invariants(needed, "field_needs_quotes.needed") }()
	if len(value) == bytes.SLICE_SIZE_MINIMUM {
		return false
	}
	if len(value) == POSTGRES_TERMINATOR_SIZE {
		if value[0] == '\\' {
			if value[1] == '.' {
				return true
			}
		}
	}
	for index, value_byte := range value {
		if value_byte == QUOTE {
			return true
		}
		if value_byte == CARRIAGE_RETURN {
			return true
		}
		if value_byte == LINE_FEED {
			return true
		}
		if delimiter_at(value, index, delimiter) {
			return true
		}
	}
	character, _ := utf8.Decode_Character(utf8.Bytes(value))
	return Boolean(ucd.Is_Space(ucd.Character(character)))
}

func encode_unchecked[
	Destination ~[]byte, Field_Values ~[]Field, Required ~int,
](
	destination Destination, fields Field_Values,
	configuration Configuration, required Required,
) {
	Configuration_Invariants(configuration, "encode_unchecked.configuration")
	position := bytes.SLICE_SIZE_MINIMUM
	delimiter_size := configuration.Delimiter_Size[SCALAR_STORAGE_INDEX]
	delimiter := configuration.Delimiter[:delimiter_size]
	for index, field := range fields {
		if index > bytes.SLICES_COUNT_MINIMUM {
			position += copy(destination[position:], delimiter)
		}
		position = encode_field(
			destination, position, field.Value, delimiter,
			configuration.Use_CRLF[SCALAR_STORAGE_INDEX],
		)
	}
	if configuration.Use_CRLF[SCALAR_STORAGE_INDEX] {
		destination[position] = CARRIAGE_RETURN
		position++
	}
	destination[position] = LINE_FEED
	position++
	aver.Always(position == int(required), "CSV encoding writes its exact reported size.")
}

func encode_field[
	Destination ~[]byte, Position ~int, Value ~[]byte,
	Delimiter ~[]byte, Use_CRLF ~bool,
](
	destination Destination, destination_position Position, value Value,
	delimiter Delimiter, use_crlf Use_CRLF,
) (next_position Position) {
	next := int(destination_position)
	if !field_needs_quotes(value, delimiter) {
		return Position(next + copy(destination[next:], value))
	}
	destination[next] = QUOTE
	next++
	for _, value_byte := range value {
		switch value_byte {
		case QUOTE:
			destination[next] = QUOTE
			destination[next+1] = QUOTE
			next += ESCAPED_QUOTE_SIZE
		case CARRIAGE_RETURN:
			if !use_crlf {
				destination[next] = value_byte
				next++
			}
		case LINE_FEED:
			if use_crlf {
				destination[next] = CARRIAGE_RETURN
				next++
			}
			destination[next] = LINE_FEED
			next++
		default:
			destination[next] = value_byte
			next++
		}
	}
	destination[next] = QUOTE
	return Position(next + 1)
}

func decode_record_unchecked[
	Destination ~[]byte, Field_Storage ~[]Field, Source ~[]byte,
](
	destination Destination, fields Field_Storage, source Source, decoder *Decoder,
) (result Decode_Result) {
	defer func() {
		Decode_Result_Invariants(result, "decode_record_unchecked.result")
	}()
	Decoder_Invariants(decoder, "decode_record_unchecked.decoder")
	configuration := decoder.Configuration
	comment_size := configuration.Comment_Size[SCALAR_STORAGE_INDEX]
	comment := configuration.Comment[:comment_size]
	source_position, line := skip_ignored(
		source, 0, decoder.Next_Line[SCALAR_STORAGE_INDEX], comment,
	)
	if source_position == len(source) {
		decoder.Next_Line[SCALAR_STORAGE_INDEX] = line
		return new_decode_result(0, source_position, 0, 0, STATUS_END)
	}
	record_line := line
	var parser Parser
	parser.Source_Position[SCALAR_STORAGE_INDEX] = source_position
	parser.Line[SCALAR_STORAGE_INDEX] = line
	parser.Column[SCALAR_STORAGE_INDEX] = COLUMN_FIRST
	delimiter_size := configuration.Delimiter_Size[SCALAR_STORAGE_INDEX]
	delimiter := configuration.Delimiter[:delimiter_size]
	field_count := bytes.SLICES_COUNT_MINIMUM
	for !parser.Record_Done[SCALAR_STORAGE_INDEX] {
		if field_count >= len(fields) {
			return new_decode_result(field_count, 0, 0, 0, STATUS_FIELDS_TOO_SMALL)
		}
		parser = trim_prefix_space(
			source, parser, configuration.Trim_Leading_Space[SCALAR_STORAGE_INDEX],
		)
		field_start := parser.Decoded_Count[SCALAR_STORAGE_INDEX]
		field_line := parser.Line[SCALAR_STORAGE_INDEX]
		field_column := parser.Column[SCALAR_STORAGE_INDEX]
		field_result := decode_field(
			destination, source, delimiter,
			configuration.Lazy_Quotes[SCALAR_STORAGE_INDEX], parser,
		)
		parser = field_result.Parser
		if field_result.Status[SCALAR_STORAGE_INDEX] != STATUS_OK {
			return new_decode_result(
				field_count, 0, field_result.Error_Line[SCALAR_STORAGE_INDEX],
				field_result.Error_Column[SCALAR_STORAGE_INDEX],
				field_result.Status[SCALAR_STORAGE_INDEX],
			)
		}
		field_end := parser.Decoded_Count[SCALAR_STORAGE_INDEX]
		fields[field_count] = Field{
			Value:  Field_Value(destination[field_start:field_end]),
			Line:   Line(field_line),
			Column: Column(field_column),
		}
		field_count++
	}
	decoder.Next_Line[SCALAR_STORAGE_INDEX] = parser.Line[SCALAR_STORAGE_INDEX]
	consumed := parser.Source_Position[SCALAR_STORAGE_INDEX]
	fields_per_record := decoder.Fields_Per_Record[SCALAR_STORAGE_INDEX]
	if fields_per_record == int(FIELDS_PER_RECORD_INFERRED) {
		decoder.Fields_Per_Record[SCALAR_STORAGE_INDEX] = field_count
	} else if fields_per_record > int(FIELDS_PER_RECORD_INFERRED) {
		if fields_per_record != field_count {
			return new_decode_result(
				field_count, consumed, record_line, COLUMN_FIRST,
				STATUS_FIELD_COUNT_INVALID,
			)
		}
	}
	return new_decode_result(field_count, consumed, 0, 0, STATUS_OK)
}

func decode_field[
	Destination ~[]byte, Source ~[]byte, Delimiter ~[]byte, Lazy ~bool,
](
	destination Destination, source Source, delimiter Delimiter,
	lazy Lazy, parser Parser,
) (result Field_Decode_Result) {
	defer func() {
		Field_Decode_Result_Invariants(result, "decode_field.result")
	}()
	Parser_Invariants(parser, "decode_field.parser")
	if parser.Source_Position[SCALAR_STORAGE_INDEX] >= len(source) {
		result.Parser = parser
		result.Parser.Record_Done[SCALAR_STORAGE_INDEX] = true
		return result
	}
	if source[parser.Source_Position[SCALAR_STORAGE_INDEX]] == QUOTE {
		return decode_quoted(destination, source, delimiter, lazy, parser)
	}
	return decode_unquoted(destination, source, delimiter, lazy, parser)
}

func skip_ignored[
	Source ~[]byte, Position ~int, Line_Value ~int, Comment ~[]byte,
](
	source Source, start Position, starting_line Line_Value, comment Comment,
) (next_position Position, next_line Line_Value) {
	position := int(start)
	line := int(starting_line)
	for position < len(source) {
		line_end, next := physical_line(source, position)
		content_end := int(line_end)
		if content_end > position {
			if source[content_end-1] == CARRIAGE_RETURN {
				content_end--
			}
		}
		ignored := content_end == position
		if len(comment) > bytes.SLICE_SIZE_MINIMUM {
			if delimiter_at(source, position, comment) {
				ignored = true
			}
		}
		if !ignored {
			break
		}
		position = int(next)
		if int(next) > int(line_end) {
			line++
		}
	}
	return Position(position), Line_Value(line)
}

func trim_prefix_space[Source ~[]byte, Trim ~bool](
	source Source, parser Parser, trim Trim,
) (next Parser) {
	defer func() { Parser_Invariants(next, "trim_prefix_space.next") }()
	Parser_Invariants(parser, "trim_prefix_space.parser")
	next = parser
	if !trim {
		return next
	}
	for next.Source_Position[SCALAR_STORAGE_INDEX] < len(source) {
		position := next.Source_Position[SCALAR_STORAGE_INDEX]
		value := source[position]
		if value == LINE_FEED {
			break
		}
		if value == CARRIAGE_RETURN {
			break
		}
		character, size := utf8.Decode_Character(utf8.Bytes(source[position:]))
		if !bool(ucd.Is_Space(ucd.Character(character))) {
			break
		}
		next.Source_Position[SCALAR_STORAGE_INDEX] += int(size)
		next.Column[SCALAR_STORAGE_INDEX] += int(size)
	}
	return next
}

func decode_unquoted[
	Destination ~[]byte, Source ~[]byte, Delimiter ~[]byte, Lazy ~bool,
](
	destination Destination, source Source, delimiter Delimiter,
	lazy Lazy, parser Parser,
) (result Field_Decode_Result) {
	defer func() {
		Field_Decode_Result_Invariants(result, "decode_unquoted.result")
	}()
	Parser_Invariants(parser, "decode_unquoted.parser")
	next := parser
	for next.Source_Position[SCALAR_STORAGE_INDEX] < len(source) {
		source_index := next.Source_Position[SCALAR_STORAGE_INDEX]
		if delimiter_at(source, source_index, delimiter) {
			next.Source_Position[SCALAR_STORAGE_INDEX] += len(delimiter)
			next.Column[SCALAR_STORAGE_INDEX] += len(delimiter)
			return new_field_decode_result(next, 0, 0, STATUS_OK)
		}
		if record_ending_at(source, source_index) {
			return new_field_decode_result(
				consume_record_ending(source, next), 0, 0, STATUS_OK,
			)
		}
		value := source[source_index]
		if value == QUOTE {
			if !lazy {
				return new_field_decode_result(
					next, next.Line[SCALAR_STORAGE_INDEX],
					next.Column[SCALAR_STORAGE_INDEX], STATUS_INPUT_INVALID,
				)
			}
		}
		if next.Decoded_Count[SCALAR_STORAGE_INDEX] >= len(destination) {
			return new_field_decode_result(next, 0, 0, STATUS_OUTPUT_TOO_SMALL)
		}
		destination[next.Decoded_Count[SCALAR_STORAGE_INDEX]] = value
		next.Decoded_Count[SCALAR_STORAGE_INDEX]++
		next.Source_Position[SCALAR_STORAGE_INDEX]++
		next.Column[SCALAR_STORAGE_INDEX]++
	}
	next.Record_Done[SCALAR_STORAGE_INDEX] = true
	return new_field_decode_result(next, 0, 0, STATUS_OK)
}

func decode_quoted[
	Destination ~[]byte, Source ~[]byte, Delimiter ~[]byte, Lazy ~bool,
](
	destination Destination, source Source, delimiter Delimiter,
	lazy Lazy, parser Parser,
) (result Field_Decode_Result) {
	defer func() {
		Field_Decode_Result_Invariants(result, "decode_quoted.result")
	}()
	Parser_Invariants(parser, "decode_quoted.parser")
	next := parser
	next.Source_Position[SCALAR_STORAGE_INDEX]++
	next.Column[SCALAR_STORAGE_INDEX]++
	return decode_quoted_content(destination, source, delimiter, lazy, next)
}

func decode_quoted_content[
	Destination ~[]byte, Source ~[]byte, Delimiter ~[]byte, Lazy ~bool,
](
	destination Destination, source Source, delimiter Delimiter, lazy Lazy, parser Parser,
) (result Field_Decode_Result) {
	defer func() { Field_Decode_Result_Invariants(result, "decode_quoted_content.result") }()
	Parser_Invariants(parser, "decode_quoted_content.parser")
	next := parser
	for next.Source_Position[SCALAR_STORAGE_INDEX] < len(source) {
		source_index := next.Source_Position[SCALAR_STORAGE_INDEX]
		if source[source_index] == QUOTE {
			quote_line := next.Line[SCALAR_STORAGE_INDEX]
			quote_column := next.Column[SCALAR_STORAGE_INDEX]
			next.Source_Position[SCALAR_STORAGE_INDEX]++
			next.Column[SCALAR_STORAGE_INDEX]++
			if next.Source_Position[SCALAR_STORAGE_INDEX] == len(source) {
				next.Record_Done[SCALAR_STORAGE_INDEX] = true
				return new_field_decode_result(next, 0, 0, STATUS_OK)
			}
			after_quote := next.Source_Position[SCALAR_STORAGE_INDEX]
			if source[after_quote] == QUOTE {
				if next.Decoded_Count[SCALAR_STORAGE_INDEX] >= len(destination) {
					return new_field_decode_result(
						next, 0, 0, STATUS_OUTPUT_TOO_SMALL,
					)
				}
				destination[next.Decoded_Count[SCALAR_STORAGE_INDEX]] = QUOTE
				next.Decoded_Count[SCALAR_STORAGE_INDEX]++
				next.Source_Position[SCALAR_STORAGE_INDEX]++
				next.Column[SCALAR_STORAGE_INDEX]++
				continue
			}
			if delimiter_at(source, after_quote, delimiter) {
				next.Source_Position[SCALAR_STORAGE_INDEX] += len(delimiter)
				next.Column[SCALAR_STORAGE_INDEX] += len(delimiter)
				return new_field_decode_result(next, 0, 0, STATUS_OK)
			}
			if record_ending_at(source, after_quote) {
				return new_field_decode_result(
					consume_record_ending(source, next), 0, 0, STATUS_OK)
			}
			if !lazy {
				return new_field_decode_result(
					next, quote_line, quote_column, STATUS_INPUT_INVALID)
			}
			if next.Decoded_Count[SCALAR_STORAGE_INDEX] >= len(destination) {
				return new_field_decode_result(next, 0, 0, STATUS_OUTPUT_TOO_SMALL)
			}
			destination[next.Decoded_Count[SCALAR_STORAGE_INDEX]] = QUOTE
			next.Decoded_Count[SCALAR_STORAGE_INDEX]++
			continue
		}
		if record_ending_at(source, source_index) {
			if next.Decoded_Count[SCALAR_STORAGE_INDEX] >= len(destination) {
				return new_field_decode_result(next, 0, 0, STATUS_OUTPUT_TOO_SMALL)
			}
			next = copy_quoted_line_ending(source, destination, next)
			continue
		}
		if next.Decoded_Count[SCALAR_STORAGE_INDEX] >= len(destination) {
			return new_field_decode_result(next, 0, 0, STATUS_OUTPUT_TOO_SMALL)
		}
		destination[next.Decoded_Count[SCALAR_STORAGE_INDEX]] = source[source_index]
		next.Decoded_Count[SCALAR_STORAGE_INDEX]++
		next.Source_Position[SCALAR_STORAGE_INDEX]++
		next.Column[SCALAR_STORAGE_INDEX]++
	}
	if lazy {
		next.Record_Done[SCALAR_STORAGE_INDEX] = true
		return new_field_decode_result(next, 0, 0, STATUS_OK)
	}
	return new_field_position_result(next, STATUS_INPUT_INVALID)
}

func copy_quoted_line_ending[
	Source ~[]byte, Destination ~[]byte,
](
	source Source, destination Destination, parser Parser,
) (next Parser) {
	defer func() { Parser_Invariants(next, "copy_quoted_line_ending.next") }()
	Parser_Invariants(parser, "copy_quoted_line_ending.parser")
	next = parser
	if source[next.Source_Position[SCALAR_STORAGE_INDEX]] == CARRIAGE_RETURN {
		next.Source_Position[SCALAR_STORAGE_INDEX]++
		if next.Source_Position[SCALAR_STORAGE_INDEX] == len(source) {
			return next
		}
	}
	next.Source_Position[SCALAR_STORAGE_INDEX]++
	destination[next.Decoded_Count[SCALAR_STORAGE_INDEX]] = LINE_FEED
	next.Decoded_Count[SCALAR_STORAGE_INDEX]++
	next.Line[SCALAR_STORAGE_INDEX]++
	next.Column[SCALAR_STORAGE_INDEX] = COLUMN_FIRST
	return next
}

func consume_record_ending[Source ~[]byte](source Source, parser Parser) (next Parser) {
	defer func() { Parser_Invariants(next, "consume_record_ending.next") }()
	Parser_Invariants(parser, "consume_record_ending.parser")
	next = parser
	if source[next.Source_Position[SCALAR_STORAGE_INDEX]] == CARRIAGE_RETURN {
		next.Source_Position[SCALAR_STORAGE_INDEX]++
		if next.Source_Position[SCALAR_STORAGE_INDEX] == len(source) {
			next.Record_Done[SCALAR_STORAGE_INDEX] = true
			return next
		}
	}
	next.Source_Position[SCALAR_STORAGE_INDEX]++
	next.Line[SCALAR_STORAGE_INDEX]++
	next.Column[SCALAR_STORAGE_INDEX] = COLUMN_FIRST
	next.Record_Done[SCALAR_STORAGE_INDEX] = true
	return next
}

func physical_line[Source ~[]byte, Position ~int](
	source Source, start Position,
) (line_end Position, next Position) {
	for position := int(start); position < len(source); position++ {
		if source[position] == LINE_FEED {
			return Position(position), Position(position + LINE_FEED_SIZE)
		}
	}
	return Position(len(source)), Position(len(source))
}

func delimiter_at[Source ~[]byte, Position ~int, Delimiter ~[]byte](
	source Source, position Position, delimiter Delimiter,
) (present Boolean) {
	defer func() { Boolean_Invariants(present, "delimiter_at.present") }()
	if int(position) < bytes.SLICE_SIZE_MINIMUM {
		return false
	}
	if len(source)-int(position) < len(delimiter) {
		return false
	}
	for index := range len(delimiter) {
		if source[int(position)+index] != delimiter[index] {
			return false
		}
	}
	return true
}

func record_ending_at[Source ~[]byte, Position ~int](
	source Source, position Position,
) (ending Boolean) {
	defer func() { Boolean_Invariants(ending, "record_ending_at.ending") }()
	if source[int(position)] == LINE_FEED {
		return true
	}
	if source[int(position)] != CARRIAGE_RETURN {
		return false
	}
	if int(position)+1 == len(source) {
		return true
	}
	return source[int(position)+1] == LINE_FEED
}
