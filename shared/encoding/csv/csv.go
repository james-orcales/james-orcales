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

// NONEMPTY_SIZE_MINIMUM is one addressable byte.
const NONEMPTY_SIZE_MINIMUM = 1

// RECORD_FIELD_VALUE_SIZE_MAXIMUM leaves one byte for the mandatory record ending.
const RECORD_FIELD_VALUE_SIZE_MAXIMUM = FIELD_VALUE_SIZE_MAXIMUM - LINE_FEED_SIZE

// RECORD_FIELD_COUNT_MAXIMUM leaves one byte for the mandatory record ending.
const RECORD_FIELD_COUNT_MAXIMUM = FIELD_COUNT_MAXIMUM - LINE_FEED_SIZE

// RECORD_POSITION_MAXIMUM is final addressable encoded byte.
const RECORD_POSITION_MAXIMUM = ENCODED_SIZE_MAXIMUM - 1

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

// Delimiter_Storage hides raw policy domain from fixed configuration output.
type Delimiter_Storage interface{}

// Comment_Storage hides raw policy domain from fixed configuration output.
type Comment_Storage interface{}

// Fields_Per_Record_Storage hides raw width domain from fixed configuration output.
type Fields_Per_Record_Storage interface{}

// Lazy_Quotes_Storage hides raw policy domain from fixed configuration output.
type Lazy_Quotes_Storage interface{}

// Trim_Leading_Space_Storage hides raw policy domain from fixed configuration output.
type Trim_Leading_Space_Storage interface{}

// Use_CRLF_Storage hides raw policy domain from fixed configuration output.
type Use_CRLF_Storage interface{}

// Next_Line_Storage hides raw line domain from fixed decoder output.
type Next_Line_Storage interface{}

// Configuration stores validated policy entirely by value.
type Configuration struct {
	// Delimiter retains one code point for local wire encoding.
	Delimiter Delimiter_Storage
	// Comment is zero when comments are disabled.
	Comment Comment_Storage
	// Fields_Per_Record selects width validation.
	Fields_Per_Record Fields_Per_Record_Storage
	// Lazy_Quotes admits otherwise malformed quotes.
	Lazy_Quotes Lazy_Quotes_Storage
	// Trim_Leading_Space removes Unicode space before fields.
	Trim_Leading_Space Trim_Leading_Space_Storage
	// Use_CRLF selects carriage-return line feeds while encoding.
	Use_CRLF Use_CRLF_Storage
}

// Configuration_Invariants admits zero failure output and typed configured storage.
func Configuration_Invariants(value Configuration, _ aver.Namespace) {
	_, delimiter_valid := value.Delimiter.(Delimiter)
	_, comment_valid := value.Comment.(Comment)
	_, fields_valid := value.Fields_Per_Record.(Fields_Per_Record)
	_, lazy_valid := value.Lazy_Quotes.(Lazy_Quotes)
	_, trim_valid := value.Trim_Leading_Space.(Trim_Leading_Space)
	_, crlf_valid := value.Use_CRLF.(Use_CRLF)
	aver.Always(
		delimiter_valid == (value.Delimiter != nil),
		"Configuration delimiter storage has expected type.",
	)
	aver.Always(
		comment_valid == (value.Comment != nil),
		"Configuration comment storage has expected type.",
	)
	aver.Always(
		fields_valid == (value.Fields_Per_Record != nil),
		"Configuration field count storage has expected type.",
	)
	aver.Always(
		lazy_valid == (value.Lazy_Quotes != nil),
		"Configuration lazy quote storage has expected type.",
	)
	aver.Always(
		trim_valid == (value.Trim_Leading_Space != nil),
		"Configuration trim storage has expected type.",
	)
	aver.Always(
		crlf_valid == (value.Use_CRLF != nil),
		"Configuration line ending storage has expected type.",
	)
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

// Record_Field_Value fits beside the mandatory record ending.
type Record_Field_Value []byte

// Record_Field_Value_Invariants excludes a field that cannot reach encoding.
func Record_Field_Value_Invariants(value Record_Field_Value, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), bytes.SLICE_SIZE_MINIMUM, RECORD_FIELD_VALUE_SIZE_MAXIMUM,
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

// Error_Line is an absent zero or one-based field parser error line.
type Error_Line int

// Error_Line_Invariants keeps field error lines inside bounded input.
func Error_Line_Invariants(value Error_Line, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), bytes.SLICE_SIZE_MINIMUM, LINE_MAXIMUM).
		Ensure()
}

// Error_Column is an absent zero or one-based field parser error column.
type Error_Column int

// Error_Column_Invariants keeps field error columns inside bounded input.
func Error_Column_Invariants(value Error_Column, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), bytes.SLICE_SIZE_MINIMUM, COLUMN_MAXIMUM).
		Ensure()
}

// Source_Position is one zero-based encoded-storage boundary.
type Source_Position int

// Source_Position_Invariants keeps parser progress inside bounded input.
func Source_Position_Invariants(value Source_Position, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), bytes.SLICE_SIZE_MINIMUM, ENCODED_SIZE_MAXIMUM).
		Ensure()
}

// Decoded_Count is one zero-based decoded-storage boundary.
type Decoded_Count int

// Decoded_Count_Invariants keeps parser writes inside decoded storage.
func Decoded_Count_Invariants(value Decoded_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), bytes.SLICE_SIZE_MINIMUM, DECODED_SIZE_MAXIMUM).
		Ensure()
}

// Next_Line is the next one-based physical line, or zero before initialization.
type Next_Line int

// Next_Line_Invariants keeps cross-record line state bounded.
func Next_Line_Invariants(value Next_Line, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), bytes.SLICE_SIZE_MINIMUM, NEXT_LINE_MAXIMUM).
		Ensure()
}

// Record_Done reports whether one parser record ended.
type Record_Done bool

// Record_Done_Invariants covers active and completed records.
func Record_Done_Invariants(value Record_Done, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "CSV parser completed one record.").
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

// Record_Fields fit with their separators and mandatory ending.
type Record_Fields []Field

// Record_Fields_Invariants excludes a field count no bounded record can hold.
func Record_Fields_Invariants(value Record_Fields, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), bytes.SLICES_COUNT_MINIMUM, RECORD_FIELD_COUNT_MAXIMUM).
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

// Nonempty_Encoded has at least one addressable byte.
type Nonempty_Encoded []byte

// Nonempty_Encoded_Invariants rejects empty storage after size success.
func Nonempty_Encoded_Invariants(value Nonempty_Encoded, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), NONEMPTY_SIZE_MINIMUM, ENCODED_SIZE_MAXIMUM).
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

// Record_Encoded_Count includes the mandatory record ending.
type Record_Encoded_Count int

// Record_Encoded_Count_Invariants rejects the zero refusal sentinel after size success.
func Record_Encoded_Count_Invariants(
	value Record_Encoded_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), NONEMPTY_SIZE_MINIMUM, ENCODED_SIZE_MAXIMUM).
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

// Decoder retains only caller-owned cross-record policy state.
type Decoder struct {
	// Delimiter remains immutable after construction.
	Delimiter Delimiter_Storage
	// Comment remains immutable after construction.
	Comment Comment_Storage
	// Lazy_Quotes remains immutable after construction.
	Lazy_Quotes Lazy_Quotes_Storage
	// Trim_Leading_Space remains immutable after construction.
	Trim_Leading_Space Trim_Leading_Space_Storage
	// Fields_Per_Record stores unchecked, inferred, or learned width.
	Fields_Per_Record Fields_Per_Record_Storage
	// Next_Line is the first line in the next caller source suffix.
	Next_Line Next_Line_Storage
}

// Decoder_Invariants admits zero state and typed initialized storage.
func Decoder_Invariants(value Decoder, _ aver.Namespace) {
	_, delimiter_valid := value.Delimiter.(Delimiter)
	_, comment_valid := value.Comment.(Comment)
	_, lazy_valid := value.Lazy_Quotes.(Lazy_Quotes)
	_, trim_valid := value.Trim_Leading_Space.(Trim_Leading_Space)
	_, fields_valid := value.Fields_Per_Record.(Fields_Per_Record)
	_, line_valid := value.Next_Line.(Next_Line)
	aver.Always(
		delimiter_valid == (value.Delimiter != nil),
		"Decoder delimiter storage has expected type.",
	)
	aver.Always(
		comment_valid == (value.Comment != nil),
		"Decoder comment storage has expected type.",
	)
	aver.Always(
		lazy_valid == (value.Lazy_Quotes != nil),
		"Decoder lazy quote storage has expected type.",
	)
	aver.Always(
		trim_valid == (value.Trim_Leading_Space != nil),
		"Decoder trim storage has expected type.",
	)
	aver.Always(
		fields_valid == (value.Fields_Per_Record != nil),
		"Decoder field count storage has expected type.",
	)
	aver.Always(
		line_valid == (value.Next_Line != nil),
		"Decoder line storage has expected type.",
	)
}

// Decoder_Handle keeps caller-owned decoder identity across records.
type Decoder_Handle *Decoder

// Decoder_Handle_Invariants rejects absent dependency injection state.
func Decoder_Handle_Invariants(value Decoder_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Decoder_Invariants(*value, namespace)
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

// Record_Status reports outcomes produced after public preflight.
type Record_Status uint8

// Record_Status_Invariants excludes public-only preflight failures.
func Record_Status_Invariants(value Record_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Holed_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_FIELDS_TOO_SMALL),
			uint8(STATUS_STORAGE_INVALID), uint8(STATUS_CONFIGURATION_INVALID),
			uint8(STATUS_RECORD_TOO_LARGE),
		).
		Ensure()
}

// Field_Status reports successful, full-output, or malformed field parsing.
type Field_Status uint8

// Field_Status_Invariants lists every field parser outcome.
func Field_Status_Invariants(value Field_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_OUTPUT_TOO_SMALL),
			uint8(STATUS_INPUT_INVALID),
		).
		Ensure()
}

// Physical_Next follows at least one examined source byte.
type Physical_Next int

// Physical_Next_Invariants excludes absent physical-line progress.
func Physical_Next_Invariants(value Physical_Next, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), NONEMPTY_SIZE_MINIMUM, ENCODED_SIZE_MAXIMUM).
		Ensure()
}

// Record_Position is one addressable source byte.
type Record_Position int

// Record_Position_Invariants excludes the position after source storage.
func Record_Position_Invariants(value Record_Position, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), bytes.SLICE_SIZE_MINIMUM, RECORD_POSITION_MAXIMUM,
		).
		Ensure()
}

// Parser keeps temporary state bounded without heap-owned buffers.
type Parser struct {
	// Source_Position prevents rescanning bytes before current field.
	Source_Position Source_Position
	// Decoded_Count prevents overwriting preceding caller output.
	Decoded_Count Decoded_Count
	// Line preserves physical position across multiline fields.
	Line Line
	// Column preserves byte position across delimiter widths.
	Column Column
	// Record_Done separates field delimiter from record termination.
	Record_Done Record_Done
}

// Parser_Invariants keeps temporary indexes inside bounded source and output domains.
func Parser_Invariants(value Parser, namespace aver.Namespace) {
	Source_Position_Invariants(value.Source_Position, namespace)
	Decoded_Count_Invariants(value.Decoded_Count, namespace)
	Line_Invariants(value.Line, namespace)
	Column_Invariants(value.Column, namespace)
	Record_Done_Invariants(value.Record_Done, namespace)
}

// Decode_Result avoids raw internal returns without widening public state.
type Decode_Result struct {
	// Field_Count crosses parser boundary without raw scalar return.
	Field_Count Field_Count
	// Consumed crosses parser boundary without raw scalar return.
	Consumed Consumed_Count
	// Error_Line keeps absent zero representable beside physical lines.
	Error_Line Line
	// Error_Column keeps absent zero representable beside byte columns.
	Error_Column Column
	// Status keeps parser outcomes independent from public conversion.
	Status Record_Status
}

// Decode_Result_Invariants keeps parser results nonnegative before public conversion.
func Decode_Result_Invariants(value Decode_Result, namespace aver.Namespace) {
	Field_Count_Invariants(value.Field_Count, namespace)
	Consumed_Count_Invariants(value.Consumed, namespace)
	Line_Invariants(value.Error_Line, namespace)
	Column_Invariants(value.Error_Column, namespace)
	Record_Status_Invariants(value.Status, namespace)
}

func new_decode_result(
	field_count Field_Count, consumed Consumed_Count,
	error_line Line, error_column Column, status Record_Status,
) (result Decode_Result) {
	defer func() { Decode_Result_Invariants(result, "new_decode_result.result") }()
	Field_Count_Invariants(field_count, "new_decode_result.field_count")
	Consumed_Count_Invariants(consumed, "new_decode_result.consumed")
	Line_Invariants(error_line, "new_decode_result.error_line")
	Column_Invariants(error_column, "new_decode_result.error_column")
	Record_Status_Invariants(status, "new_decode_result.status")
	result.Field_Count = field_count
	result.Consumed = consumed
	result.Error_Line = error_line
	result.Error_Column = error_column
	result.Status = status
	return result
}

// Field_Decode_Result keeps parser refusal state in bounded value storage.
type Field_Decode_Result struct {
	// Parser carries caller-storage offsets without hidden buffers.
	Parser Parser
	// Error_Line keeps absent zero representable on successful fields.
	Error_Line Error_Line
	// Error_Column keeps absent zero representable on successful fields.
	Error_Column Error_Column
	// Status keeps field outcomes independent from record outcomes.
	Status Field_Status
}

// Field_Decode_Result_Invariants fixes storage shape between parser stages.
func Field_Decode_Result_Invariants(
	value Field_Decode_Result, namespace aver.Namespace,
) {
	Parser_Invariants(value.Parser, namespace)
	Error_Line_Invariants(value.Error_Line, namespace)
	Error_Column_Invariants(value.Error_Column, namespace)
	Field_Status_Invariants(value.Status, namespace)
}

func new_field_result(
	parser Parser, error_line Error_Line,
	error_column Error_Column, status Field_Status,
) (result Field_Decode_Result) {
	defer func() {
		Field_Decode_Result_Invariants(result, "new_field_result.result")
	}()
	Parser_Invariants(parser, "new_field_result.parser")
	Error_Line_Invariants(error_line, "new_field_result.error_line")
	Error_Column_Invariants(error_column, "new_field_result.error_column")
	Field_Status_Invariants(status, "new_field_result.status")
	result.Parser = parser
	result.Error_Line = error_line
	result.Error_Column = error_column
	result.Status = status
	return result
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
	if !bool(character_allowed(utf8.Character(input.Delimiter))) {
		return Configuration{}, STATUS_CONFIGURATION_INVALID
	}
	if input.Comment != COMMENT_DISABLED {
		if !bool(character_allowed(utf8.Character(input.Comment))) {
			return Configuration{}, STATUS_CONFIGURATION_INVALID
		}
		if int32(input.Comment) == int32(input.Delimiter) {
			return Configuration{}, STATUS_CONFIGURATION_INVALID
		}
	}
	configuration.Delimiter = input.Delimiter
	configuration.Comment = input.Comment
	configuration.Fields_Per_Record = input.Fields_Per_Record
	configuration.Lazy_Quotes = input.Lazy_Quotes
	configuration.Trim_Leading_Space = input.Trim_Leading_Space
	configuration.Use_CRLF = input.Use_CRLF
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
	value, _ := configuration.Delimiter.(Delimiter)
	return Delimiter_Character(value)
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
	value, _ := configuration.Comment.(Comment)
	return Comment_Character(value)
}

// Decoder_Init leaves invalid policy unable to mutate caller state.
func Decoder_Init(
	decoder Decoder_Handle, configuration Configuration,
) (status Decoder_Status) {
	defer func() {
		Decoder_Invariants(*decoder, "Decoder_Init.decoder_result")
		Decoder_Status_Invariants(status, "Decoder_Init.status")
	}()
	Decoder_Handle_Invariants(decoder, "Decoder_Init.decoder")
	Configuration_Invariants(configuration, "Decoder_Init.configuration")
	if !bool(configuration_valid(configuration)) {
		return STATUS_CONFIGURATION_INVALID
	}
	decoder.Delimiter = configuration.Delimiter
	decoder.Comment = configuration.Comment
	decoder.Lazy_Quotes = configuration.Lazy_Quotes
	decoder.Trim_Leading_Space = configuration.Trim_Leading_Space
	decoder.Fields_Per_Record = configuration.Fields_Per_Record
	decoder.Next_Line = Next_Line(LINE_FIRST)
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
	delimiter, _ := configuration.Delimiter.(Delimiter)
	use_crlf, _ := configuration.Use_CRLF.(Use_CRLF)
	var delimiter_storage [utf8.CHARACTER_SIZE_MAXIMUM]byte
	delimiter_size := utf8.Encode_Character(
		utf8.Bytes(delimiter_storage[:]), utf8.Character(delimiter),
	)
	required := int(line_ending_size(use_crlf))
	if len(fields) > bytes.SLICES_COUNT_MINIMUM {
		required += (len(fields) - 1) * int(delimiter_size)
	}
	for _, field := range fields {
		Field_Value_Invariants(field.Value, "Encoded_Size.field")
		required += int(encoded_field_size(
			field.Value, Delimiter_Character(delimiter), use_crlf,
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
	encode_unchecked(
		Nonempty_Encoded(destination), Record_Fields(fields), configuration,
		Record_Encoded_Count(required),
	)
	return required, STATUS_OK
}

// Decode_Record_Into parses one record and mutates only caller-owned decoder state.
func Decode_Record_Into(
	destination Decoded, fields Fields, source Encoded, decoder Decoder_Handle,
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
	Decoder_Handle_Invariants(decoder, "Decode_Record_Into.decoder")
	Decoder_Invariants(*decoder, "Decode_Record_Into.decoder_value")
	if !bool(decoder_valid(decoder)) {
		return 0, 0, Parse_Position{}, STATUS_CONFIGURATION_INVALID
	}
	if bytes.Overlap(bytes.Slice(destination), bytes.Slice(source)) {
		return 0, 0, Parse_Position{}, STATUS_STORAGE_INVALID
	}
	result := decode_record_unchecked(destination, fields, source, decoder)
	return Field_Count(result.Field_Count),
		Consumed_Count(result.Consumed),
		Parse_Position{
			Line:   Line(result.Error_Line),
			Column: Column(result.Error_Column),
		},
		Decode_Status(result.Status)
}

func decoder_valid(decoder Decoder_Handle) (valid Decoder_Valid) {
	defer func() { Decoder_Valid_Invariants(valid, "decoder_valid.valid") }()
	Decoder_Handle_Invariants(decoder, "decoder_valid.decoder")
	Decoder_Invariants(*decoder, "decoder_valid.decoder_value")
	configuration := Configuration{
		Delimiter:          decoder.Delimiter,
		Comment:            decoder.Comment,
		Fields_Per_Record:  decoder.Fields_Per_Record,
		Lazy_Quotes:        decoder.Lazy_Quotes,
		Trim_Leading_Space: decoder.Trim_Leading_Space,
		Use_CRLF:           Use_CRLF(false),
	}
	if !bool(configuration_valid(configuration)) {
		return false
	}
	fields_per_record, fields_valid := decoder.Fields_Per_Record.(Fields_Per_Record)
	if !fields_valid {
		return false
	}
	if fields_per_record < FIELDS_PER_RECORD_UNCHECKED {
		return false
	}
	if fields_per_record > FIELD_COUNT_MAXIMUM {
		return false
	}
	next_line, line_valid := decoder.Next_Line.(Next_Line)
	if !line_valid {
		return false
	}
	if next_line < LINE_FIRST {
		return false
	}
	return Decoder_Valid(next_line <= NEXT_LINE_MAXIMUM)
}

func character_allowed(character utf8.Character) (allowed Boolean) {
	defer func() { Boolean_Invariants(allowed, "character_allowed.allowed") }()
	utf8.Character_Invariants(character, "character_allowed.character")
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
	delimiter_value, delimiter_valid := configuration.Delimiter.(Delimiter)
	comment_value, comment_valid := configuration.Comment.(Comment)
	fields_per_record, fields_valid := configuration.Fields_Per_Record.(Fields_Per_Record)
	_, lazy_valid := configuration.Lazy_Quotes.(Lazy_Quotes)
	_, trim_valid := configuration.Trim_Leading_Space.(Trim_Leading_Space)
	_, crlf_valid := configuration.Use_CRLF.(Use_CRLF)
	if !delimiter_valid {
		return false
	}
	if !comment_valid {
		return false
	}
	if !fields_valid {
		return false
	}
	if !lazy_valid {
		return false
	}
	if !trim_valid {
		return false
	}
	if !crlf_valid {
		return false
	}
	delimiter := utf8.Character(delimiter_value)
	if !bool(character_allowed(delimiter)) {
		return false
	}
	if fields_per_record < FIELDS_PER_RECORD_UNCHECKED {
		return false
	}
	if fields_per_record > FIELD_COUNT_MAXIMUM {
		return false
	}
	if comment_value == COMMENT_DISABLED {
		return true
	}
	comment := utf8.Character(comment_value)
	if !bool(character_allowed(comment)) {
		return false
	}
	if comment == delimiter {
		return false
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

func line_ending_size(use_crlf Use_CRLF) (size Line_Ending_Size_Count) {
	defer func() {
		Line_Ending_Size_Count_Invariants(size, "line_ending_size.size")
	}()
	Use_CRLF_Invariants(use_crlf, "line_ending_size.use_crlf")
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

func encoded_field_size(
	value Field_Value, delimiter Delimiter_Character, use_crlf Use_CRLF,
) (size Encoded_Field_Size_Count) {
	defer func() {
		Encoded_Field_Size_Count_Invariants(size, "encoded_field_size.size")
	}()
	Field_Value_Invariants(value, "encoded_field_size.value")
	Delimiter_Character_Invariants(delimiter, "encoded_field_size.delimiter")
	Use_CRLF_Invariants(use_crlf, "encoded_field_size.use_crlf")
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

func field_needs_quotes(value Field_Value, delimiter Delimiter_Character) (needed Boolean) {
	defer func() { Boolean_Invariants(needed, "field_needs_quotes.needed") }()
	Field_Value_Invariants(value, "field_needs_quotes.value")
	Delimiter_Character_Invariants(delimiter, "field_needs_quotes.delimiter")
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
		if delimiter_at(Nonempty_Encoded(value), Source_Position(index), delimiter) {
			return true
		}
	}
	character, _ := utf8.Decode_Character(utf8.Bytes(value))
	return Boolean(ucd.Is_Space(ucd.Character(character)))
}

func encode_unchecked(
	destination Nonempty_Encoded, fields Record_Fields,
	configuration Configuration, required Record_Encoded_Count,
) {
	Nonempty_Encoded_Invariants(destination, "encode_unchecked.destination")
	Record_Fields_Invariants(fields, "encode_unchecked.fields")
	Configuration_Invariants(configuration, "encode_unchecked.configuration")
	Record_Encoded_Count_Invariants(required, "encode_unchecked.required")
	delimiter_value, _ := configuration.Delimiter.(Delimiter)
	use_crlf, _ := configuration.Use_CRLF.(Use_CRLF)
	position := bytes.SLICE_SIZE_MINIMUM
	var delimiter_storage [utf8.CHARACTER_SIZE_MAXIMUM]byte
	delimiter_size := utf8.Encode_Character(
		utf8.Bytes(delimiter_storage[:]), utf8.Character(delimiter_value),
	)
	delimiter := utf8.Bytes(delimiter_storage[:delimiter_size])
	for index, field := range fields {
		if index > bytes.SLICES_COUNT_MINIMUM {
			position += copy(destination[position:], delimiter)
		}
		position = int(encode_field(
			destination,
			Source_Position(position),
			Record_Field_Value(field.Value),
			Delimiter_Character(delimiter_value),
			use_crlf,
		))
	}
	if use_crlf {
		destination[position] = CARRIAGE_RETURN
		position++
	}
	destination[position] = LINE_FEED
	position++
	aver.Always(position == int(required), "CSV encoding writes its exact reported size.")
}

func encode_field(
	destination Nonempty_Encoded, destination_position Source_Position,
	value Record_Field_Value, delimiter Delimiter_Character, use_crlf Use_CRLF,
) (next_position Source_Position) {
	defer func() {
		Source_Position_Invariants(next_position, "encode_field.next_position")
	}()
	Nonempty_Encoded_Invariants(destination, "encode_field.destination")
	Source_Position_Invariants(destination_position, "encode_field.destination_position")
	Record_Field_Value_Invariants(value, "encode_field.value")
	Delimiter_Character_Invariants(delimiter, "encode_field.delimiter")
	Use_CRLF_Invariants(use_crlf, "encode_field.use_crlf")
	next := destination_position
	if !field_needs_quotes(Field_Value(value), delimiter) {
		return next + Source_Position(copy(destination[next:], value))
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
	return next + 1
}

func decode_record_unchecked(
	destination Decoded, fields Fields, source Encoded, decoder Decoder_Handle,
) (result Decode_Result) {
	defer func() { Decode_Result_Invariants(result, "decode_record_unchecked.result") }()
	Decoded_Invariants(destination, "decode_record_unchecked.destination")
	Fields_Invariants(fields, "decode_record_unchecked.fields")
	Encoded_Invariants(source, "decode_record_unchecked.source")
	Decoder_Handle_Invariants(decoder, "decode_record_unchecked.decoder")
	Decoder_Invariants(*decoder, "decode_record_unchecked.decoder_value")
	next_line, _ := decoder.Next_Line.(Next_Line)
	comment, _ := decoder.Comment.(Comment)
	trim, _ := decoder.Trim_Leading_Space.(Trim_Leading_Space)
	delimiter, _ := decoder.Delimiter.(Delimiter)
	lazy, _ := decoder.Lazy_Quotes.(Lazy_Quotes)
	fields_per_record, _ := decoder.Fields_Per_Record.(Fields_Per_Record)
	source_position, line := skip_ignored(source, 0, next_line, Comment_Character(comment))
	if int(source_position) == len(source) {
		decoder.Next_Line = line
		return new_decode_result(0, Consumed_Count(source_position), 0, 0, STATUS_END)
	}
	parser := Parser{
		Source_Position: source_position, Line: Line(line), Column: COLUMN_FIRST,
	}
	field_count := bytes.SLICES_COUNT_MINIMUM
	for !parser.Record_Done {
		if field_count >= len(fields) {
			return new_decode_result(
				Field_Count(field_count), 0, 0, 0, STATUS_FIELDS_TOO_SMALL,
			)
		}
		parser = trim_prefix_space(Nonempty_Encoded(source), parser, trim)
		field_start := parser.Decoded_Count
		field_line := parser.Line
		field_column := parser.Column
		field_result := decode_field(
			destination, Nonempty_Encoded(source),
			Delimiter_Character(delimiter), lazy, parser,
		)
		parser = field_result.Parser
		if field_result.Status != STATUS_OK {
			status := Record_Status(field_result.Status)
			return new_decode_result(
				Field_Count(field_count), 0, Line(field_result.Error_Line),
				Column(field_result.Error_Column), status,
			)
		}
		field_end := parser.Decoded_Count
		fields[field_count] = Field{
			Value:  Field_Value(destination[field_start:field_end]),
			Line:   Line(field_line),
			Column: Column(field_column),
		}
		field_count++
	}
	decoder.Next_Line = Next_Line(parser.Line)
	consumed := parser.Source_Position
	if fields_per_record == FIELDS_PER_RECORD_INFERRED {
		decoder.Fields_Per_Record = Fields_Per_Record(field_count)
	} else if fields_per_record > FIELDS_PER_RECORD_INFERRED {
		if int(fields_per_record) != field_count {
			return new_decode_result(
				Field_Count(field_count),
				Consumed_Count(consumed), Line(line), COLUMN_FIRST,
				STATUS_FIELD_COUNT_INVALID,
			)
		}
	}
	return new_decode_result(
		Field_Count(field_count), Consumed_Count(consumed), 0, 0, STATUS_OK,
	)
}

func decode_field(
	destination Decoded, source Nonempty_Encoded, delimiter Delimiter_Character,
	lazy Lazy_Quotes, parser Parser,
) (result Field_Decode_Result) {
	defer func() {
		Field_Decode_Result_Invariants(result, "decode_field.result")
	}()
	Decoded_Invariants(destination, "decode_field.destination")
	Nonempty_Encoded_Invariants(source, "decode_field.source")
	Delimiter_Character_Invariants(delimiter, "decode_field.delimiter")
	Lazy_Quotes_Invariants(lazy, "decode_field.lazy")
	Parser_Invariants(parser, "decode_field.parser")
	if int(parser.Source_Position) >= len(source) {
		result.Parser = parser
		result.Parser.Record_Done = true
		return result
	}
	if source[parser.Source_Position] == QUOTE {
		parser.Source_Position++
		parser.Column++
		return decode_quoted_content(destination, source, delimiter, lazy, parser)
	}
	return decode_unquoted(destination, source, delimiter, lazy, parser)
}

func skip_ignored(
	source Encoded, start Source_Position, starting_line Next_Line,
	comment Comment_Character,
) (next_position Source_Position, next_line Next_Line) {
	defer func() {
		Source_Position_Invariants(next_position, "skip_ignored.next_position")
		Next_Line_Invariants(next_line, "skip_ignored.next_line")
	}()
	Encoded_Invariants(source, "skip_ignored.source")
	Source_Position_Invariants(start, "skip_ignored.start")
	Next_Line_Invariants(starting_line, "skip_ignored.starting_line")
	Comment_Character_Invariants(comment, "skip_ignored.comment")
	position := start
	line := starting_line
	for int(position) < len(source) {
		line_end, next := physical_line(Nonempty_Encoded(source), position)
		content_end := int(line_end)
		if content_end > int(position) {
			if source[content_end-1] == CARRIAGE_RETURN {
				content_end--
			}
		}
		ignored := content_end == int(position)
		if comment != Comment_Character(COMMENT_DISABLED) {
			if delimiter_at(
				Nonempty_Encoded(source), position, Delimiter_Character(comment),
			) {
				ignored = true
			}
		}
		if !ignored {
			break
		}
		position = Source_Position(next)
		if Source_Position(next) > line_end {
			line++
		}
	}
	return position, line
}

func trim_prefix_space(
	source Nonempty_Encoded, parser Parser, trim Trim_Leading_Space,
) (next Parser) {
	defer func() { Parser_Invariants(next, "trim_prefix_space.next") }()
	Nonempty_Encoded_Invariants(source, "trim_prefix_space.source")
	Parser_Invariants(parser, "trim_prefix_space.parser")
	Trim_Leading_Space_Invariants(trim, "trim_prefix_space.trim")
	next = parser
	if !trim {
		return next
	}
	for int(next.Source_Position) < len(source) {
		position := next.Source_Position
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
		next.Source_Position += Source_Position(size)
		next.Column += Column(size)
	}
	return next
}

func decode_unquoted(
	destination Decoded, source Nonempty_Encoded, delimiter Delimiter_Character,
	lazy Lazy_Quotes, parser Parser,
) (result Field_Decode_Result) {
	defer func() {
		Field_Decode_Result_Invariants(result, "decode_unquoted.result")
	}()
	Decoded_Invariants(destination, "decode_unquoted.destination")
	Nonempty_Encoded_Invariants(source, "decode_unquoted.source")
	Delimiter_Character_Invariants(delimiter, "decode_unquoted.delimiter")
	Lazy_Quotes_Invariants(lazy, "decode_unquoted.lazy")
	Parser_Invariants(parser, "decode_unquoted.parser")
	next := parser
	for int(next.Source_Position) < len(source) {
		source_index := next.Source_Position
		if delimiter_at(source, source_index, delimiter) {
			delimiter_size := int(utf8.Character_Size(utf8.Character(delimiter)))
			next.Source_Position += Source_Position(delimiter_size)
			next.Column += Column(delimiter_size)
			return new_field_result(next, 0, 0, STATUS_OK)
		}
		if record_ending_at(source, Record_Position(source_index)) {
			next.Record_Done = true
			if source[next.Source_Position] == CARRIAGE_RETURN {
				next.Source_Position++
				if int(next.Source_Position) == len(source) {
					return new_field_result(next, 0, 0, STATUS_OK)
				}
			}
			next.Source_Position++
			next.Line++
			next.Column = COLUMN_FIRST
			return new_field_result(
				next, 0, 0, STATUS_OK,
			)
		}
		value := source[source_index]
		if value == QUOTE {
			if !lazy {
				return new_field_result(
					next, Error_Line(next.Line),
					Error_Column(next.Column), STATUS_INPUT_INVALID,
				)
			}
		}
		if int(next.Decoded_Count) >= len(destination) {
			return new_field_result(next, 0, 0, STATUS_OUTPUT_TOO_SMALL)
		}
		destination[next.Decoded_Count] = value
		next.Decoded_Count++
		next.Source_Position++
		next.Column++
	}
	next.Record_Done = true
	return new_field_result(next, 0, 0, STATUS_OK)
}

func consume_record_ending(
	source Nonempty_Encoded, parser Parser,
) (next Parser) {
	defer func() { Parser_Invariants(next, "consume_record_ending.next") }()
	Nonempty_Encoded_Invariants(source, "consume_record_ending.source")
	Parser_Invariants(parser, "consume_record_ending.parser")
	next = parser
	if int(next.Source_Position) >= len(source) {
		return next
	}
	if !record_ending_at(source, Record_Position(next.Source_Position)) {
		return next
	}
	next.Record_Done = true
	if source[next.Source_Position] == CARRIAGE_RETURN {
		next.Source_Position++
		if int(next.Source_Position) == len(source) {
			return next
		}
	}
	next.Source_Position++
	next.Line++
	next.Column = COLUMN_FIRST
	return next
}

func copy_record_ending(
	destination Decoded, source Nonempty_Encoded, parser Parser,
) (next Parser, available Boolean) {
	defer func() {
		Parser_Invariants(next, "copy_record_ending.next")
		Boolean_Invariants(available, "copy_record_ending.available")
	}()
	Decoded_Invariants(destination, "copy_record_ending.destination")
	Nonempty_Encoded_Invariants(source, "copy_record_ending.source")
	Parser_Invariants(parser, "copy_record_ending.parser")
	next = parser
	if int(next.Decoded_Count) >= len(destination) {
		return next, false
	}
	if source[next.Source_Position] == CARRIAGE_RETURN {
		next.Source_Position++
		if int(next.Source_Position) == len(source) {
			return next, true
		}
	}
	next.Source_Position++
	destination[next.Decoded_Count] = LINE_FEED
	next.Decoded_Count++
	next.Line++
	next.Column = COLUMN_FIRST
	return next, true
}

func decode_quoted_content(
	dst Decoded, source Nonempty_Encoded, delimiter Delimiter_Character,
	lazy Lazy_Quotes, parser Parser,
) (result Field_Decode_Result) {
	defer func() { Field_Decode_Result_Invariants(result, "decode_quoted_content.result") }()
	Decoded_Invariants(dst, "decode_quoted_content.destination")
	Nonempty_Encoded_Invariants(source, "decode_quoted_content.source")
	Delimiter_Character_Invariants(delimiter, "decode_quoted_content.delimiter")
	Lazy_Quotes_Invariants(lazy, "decode_quoted_content.lazy")
	Parser_Invariants(parser, "decode_quoted_content.parser")
	next := parser
	for int(next.Source_Position) < len(source) {
		if source[next.Source_Position] == QUOTE {
			next.Source_Position, next.Column = next.Source_Position+1, next.Column+1
			if int(next.Source_Position) == len(source) {
				next.Record_Done = true
				return new_field_result(next, 0, 0, STATUS_OK)
			}
			after_quote := next.Source_Position
			if source[after_quote] == QUOTE {
				if int(next.Decoded_Count) >= len(dst) {
					return new_field_result(next, 0, 0, STATUS_OUTPUT_TOO_SMALL)
				}
				dst[next.Decoded_Count] = QUOTE
				next.Decoded_Count, next.Source_Position, next.Column =
					next.Decoded_Count+1, next.Source_Position+1, next.Column+1
				continue
			}
			if delimiter_at(source, after_quote, delimiter) {
				size := int(utf8.Character_Size(utf8.Character(delimiter)))
				next.Source_Position += Source_Position(size)
				next.Column += Column(size)
				return new_field_result(next, 0, 0, STATUS_OK)
			}
			if record_ending_at(source, Record_Position(after_quote)) {
				next = consume_record_ending(source, next)
				return new_field_result(next, 0, 0, STATUS_OK)
			}
			if !lazy {
				column := Error_Column(next.Column - 1)
				return new_field_result(next, Error_Line(next.Line), column,
					STATUS_INPUT_INVALID)
			}
			if int(next.Decoded_Count) >= len(dst) {
				return new_field_result(next, 0, 0, STATUS_OUTPUT_TOO_SMALL)
			}
			dst[next.Decoded_Count] = QUOTE
			next.Decoded_Count++
			continue
		}
		if record_ending_at(source, Record_Position(next.Source_Position)) {
			var available Boolean
			next, available = copy_record_ending(dst, source, next)
			if !available {
				return new_field_result(next, 0, 0, STATUS_OUTPUT_TOO_SMALL)
			}
			continue
		}
		if int(next.Decoded_Count) >= len(dst) {
			return new_field_result(next, 0, 0, STATUS_OUTPUT_TOO_SMALL)
		}
		dst[next.Decoded_Count] = source[next.Source_Position]
		next.Decoded_Count, next.Source_Position, next.Column =
			next.Decoded_Count+1, next.Source_Position+1, next.Column+1
	}
	if lazy {
		next.Record_Done = true
		return new_field_result(next, 0, 0, STATUS_OK)
	}
	return new_field_result(
		next, Error_Line(next.Line), Error_Column(next.Column), STATUS_INPUT_INVALID,
	)
}

func physical_line(
	source Nonempty_Encoded, start Source_Position,
) (line_end Source_Position, next Physical_Next) {
	defer func() {
		Source_Position_Invariants(line_end, "physical_line.line_end")
		Physical_Next_Invariants(next, "physical_line.next")
	}()
	Nonempty_Encoded_Invariants(source, "physical_line.source")
	Source_Position_Invariants(start, "physical_line.start")
	for position := start; int(position) < len(source); position++ {
		if source[position] == LINE_FEED {
			return position, Physical_Next(position + LINE_FEED_SIZE)
		}
	}
	return Source_Position(len(source)), Physical_Next(len(source))
}

func delimiter_at(
	source Nonempty_Encoded, position Source_Position, delimiter Delimiter_Character,
) (present Boolean) {
	defer func() { Boolean_Invariants(present, "delimiter_at.present") }()
	Nonempty_Encoded_Invariants(source, "delimiter_at.source")
	Source_Position_Invariants(position, "delimiter_at.position")
	Delimiter_Character_Invariants(delimiter, "delimiter_at.delimiter")
	if position < bytes.SLICE_SIZE_MINIMUM {
		return false
	}
	var delimiter_storage [utf8.CHARACTER_SIZE_MAXIMUM]byte
	delimiter_size := int(utf8.Encode_Character(
		utf8.Bytes(delimiter_storage[:]), utf8.Character(delimiter),
	))
	if len(source)-int(position) < delimiter_size {
		return false
	}
	for index := range delimiter_size {
		if source[position+Source_Position(index)] != delimiter_storage[index] {
			return false
		}
	}
	return true
}

func record_ending_at(source Nonempty_Encoded, position Record_Position) (ending Boolean) {
	defer func() { Boolean_Invariants(ending, "record_ending_at.ending") }()
	Nonempty_Encoded_Invariants(source, "record_ending_at.source")
	Record_Position_Invariants(position, "record_ending_at.position")
	if source[position] == LINE_FEED {
		return true
	}
	if source[position] != CARRIAGE_RETURN {
		return false
	}
	if int(position)+1 == len(source) {
		return true
	}
	return source[position+1] == LINE_FEED
}
