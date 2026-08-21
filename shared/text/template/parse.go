// Parser stores bounded text/template syntax in caller-owned nodes.
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

// SOURCE_SIZE_MINIMUM keeps an empty template valid.
const SOURCE_SIZE_MINIMUM = bytes.SLICE_SIZE_MINIMUM

// SOURCE_SIZE_MAXIMUM follows the repository byte-slice boundary.
const SOURCE_SIZE_MAXIMUM = bytes.SLICE_SIZE_MAXIMUM

// SOURCE_SIZE_UNVALIDATED_MAXIMUM admits the first refused source size.
const SOURCE_SIZE_UNVALIDATED_MAXIMUM = SOURCE_SIZE_MAXIMUM + utf8.CHARACTER_SIZE_MINIMUM

// SOURCE_POSITION_INCREMENT advances across one single-byte grammar character.
const SOURCE_POSITION_INCREMENT = utf8.CHARACTER_SIZE_MINIMUM

// DELIMITER_SIZE_MINIMUM uses empty input to select standard delimiters.
const DELIMITER_SIZE_MINIMUM = bytes.SLICE_SIZE_MINIMUM

// DELIMITER_SIZE_MAXIMUM cannot usefully exceed bounded source.
const DELIMITER_SIZE_MAXIMUM = SOURCE_SIZE_MAXIMUM

// DELIMITER_SIZE_UNVALIDATED_MAXIMUM admits the first refused delimiter size.
const DELIMITER_SIZE_UNVALIDATED_MAXIMUM = DELIMITER_SIZE_MAXIMUM + utf8.CHARACTER_SIZE_MINIMUM

// IDENTIFIER_SIZE_MINIMUM is the first valid function-name byte count.
const IDENTIFIER_SIZE_MINIMUM = utf8.CHARACTER_SIZE_MINIMUM

// NODE_VALUE_START_MAXIMUM reserves one byte for every nonempty node value.
const NODE_VALUE_START_MAXIMUM = SOURCE_SIZE_MAXIMUM - IDENTIFIER_SIZE_MINIMUM

// ASCII_BYTE_MINIMUM is first single-byte UTF-8 character.
const ASCII_BYTE_MINIMUM = bits.WORD_8_MINIMUM

// ASCII_BYTE_MAXIMUM precedes first multibyte UTF-8 prefix.
const ASCII_BYTE_MAXIMUM = uint8(utf8.CHARACTER_SELF) - utf8.CHARACTER_SIZE_MINIMUM

// NUMBER_BASE_BINARY is radix selected by 0b prefix.
const NUMBER_BASE_BINARY Number_Base = big.BASE_BINARY

// NUMBER_BASE_OCTAL is three binary digits per numeral.
const NUMBER_BASE_OCTAL Number_Base = big.BASE_OCTAL

// NUMBER_BASE_DECIMAL adds two digits to octal alphabet.
const NUMBER_BASE_DECIMAL Number_Base = big.BASE_DECIMAL

// NUMBER_BASE_HEXADECIMAL is four binary digits per numeral.
const NUMBER_BASE_HEXADECIMAL Number_Base = big.BASE_HEXADECIMAL

// NUMBER_BASE_OCTAL_DIGIT_BIT_COUNT is binary payload carried by one octal digit.
const NUMBER_BASE_OCTAL_DIGIT_BIT_COUNT = NUMBER_BASE_OCTAL/NUMBER_BASE_BINARY -
	NUMBER_BASE_BINARY/NUMBER_BASE_BINARY

// NUMBER_BASE_HEXADECIMAL_DIGIT_BIT_COUNT is binary payload carried by one hexadecimal digit.
const NUMBER_BASE_HEXADECIMAL_DIGIT_BIT_COUNT = big.BASE_HEXADECIMAL_DIGIT_BIT_COUNT

// NUMBER_PREFIX_SIZE charges zero and base selector bytes.
const NUMBER_PREFIX_SIZE = len("0x")

// NUMBER_SEQUENCE_REQUIRED needs one digit without prefix underscore.
const NUMBER_SEQUENCE_REQUIRED Number_Sequence_Mode = Number_Sequence_Mode(bytes.SLICE_SIZE_MINIMUM)

// NUMBER_SEQUENCE_PREFIXED permits one underscore immediately after base prefix.
const NUMBER_SEQUENCE_PREFIXED Number_Sequence_Mode = NUMBER_SEQUENCE_REQUIRED +
	utf8.CHARACTER_SIZE_MINIMUM

// NUMBER_SEQUENCE_OPTIONAL permits an empty fractional digit run.
const NUMBER_SEQUENCE_OPTIONAL Number_Sequence_Mode = NUMBER_SEQUENCE_PREFIXED +
	utf8.CHARACTER_SIZE_MINIMUM

// NUMBER_PREFIX_REQUIRED is the ordinary unprefixed digit rule.
const NUMBER_PREFIX_REQUIRED Number_Prefix_Mode = Number_Prefix_Mode(NUMBER_SEQUENCE_REQUIRED)

// NUMBER_PREFIX_PREFIXED permits the underscore following an explicit base.
const NUMBER_PREFIX_PREFIXED Number_Prefix_Mode = Number_Prefix_Mode(NUMBER_SEQUENCE_PREFIXED)

// QUOTED_HEX_DIGIT_COUNT is byte escape hexadecimal width.
const QUOTED_HEX_DIGIT_COUNT Quoted_Digit_Count = strconv.HEXADECIMAL_DIGIT_COUNT

// QUOTED_SHORT_UNICODE_DIGIT_COUNT is short Unicode escape width.
const QUOTED_SHORT_UNICODE_DIGIT_COUNT Quoted_Digit_Count = strconv.SHORT_UNICODE_DIGIT_COUNT

// QUOTED_LONG_UNICODE_DIGIT_COUNT is long Unicode escape width.
const QUOTED_LONG_UNICODE_DIGIT_COUNT Quoted_Digit_Count = strconv.LONG_UNICODE_DIGIT_COUNT

// QUOTED_ESCAPE_PREFIX_SIZE is backslash plus escape selector.
const QUOTED_ESCAPE_PREFIX_SIZE = strconv.ESCAPE_SEQUENCE_SIZE_MINIMUM

// QUOTED_CHARACTER_COUNT_MINIMUM is empty decoded literal body.
const QUOTED_CHARACTER_COUNT_MINIMUM = bytes.SLICE_SIZE_MINIMUM

// QUOTED_CHARACTER_COUNT_REQUIRED is exact character literal payload count.
const QUOTED_CHARACTER_COUNT_REQUIRED = QUOTED_CHARACTER_COUNT_MINIMUM + utf8.CHARACTER_SIZE_MINIMUM

// QUOTE_CHARACTER selects one-character literal grammar.
const QUOTE_CHARACTER Quote = '\''

// QUOTE_STRING selects interpreted string grammar.
const QUOTE_STRING Quote = '"'

// QUOTE_RAW_STRING selects raw string grammar.
const QUOTE_RAW_STRING Quote = '`'

// QUOTED_OCTAL_DIGIT_COUNT follows one byte's three base-eight digits.
const QUOTED_OCTAL_DIGIT_COUNT Quoted_Digit_Count = strconv.OCTAL_ESCAPE_DIGIT_COUNT +
	utf8.CHARACTER_SIZE_MINIMUM

// QUOTED_DECODED_SIZE_MAXIMUM removes both quote marks before malformed UTF-8 expansion.
const QUOTED_DECODED_SIZE_MAXIMUM = (SOURCE_SIZE_MAXIMUM -
	QUOTED_BOUNDARY_SIZE) * utf8.CHARACTER_SIZE_THREE

// TEMPLATE_NAME_SIZE_MAXIMUM removes mandatory quote marks before decoding.
const TEMPLATE_NAME_SIZE_MAXIMUM = QUOTED_DECODED_SIZE_MAXIMUM

// NODE_COUNT_MINIMUM is empty caller node state.
const NODE_COUNT_MINIMUM = bytes.SLICE_SIZE_MINIMUM

// NODE_COUNT_INCREMENT adds one syntax record.
const NODE_COUNT_INCREMENT = 1

// CUSTOM_DELIMITER_SIZE_MINIMUM is shortest nondefault delimiter.
const CUSTOM_DELIMITER_SIZE_MINIMUM = utf8.CHARACTER_SIZE_MINIMUM

// ACTION_LEFT_DELIMITER_COUNT counts action opening boundary.
const ACTION_LEFT_DELIMITER_COUNT = 1

// ACTION_RIGHT_DELIMITER_COUNT counts action closing boundary.
const ACTION_RIGHT_DELIMITER_COUNT = ACTION_LEFT_DELIMITER_COUNT

// ACTION_DELIMITER_COUNT counts both action boundaries.
const ACTION_DELIMITER_COUNT = ACTION_LEFT_DELIMITER_COUNT + ACTION_RIGHT_DELIMITER_COUNT

// ACTION_BOUNDARY_SIZE_MINIMUM keeps every action density formula on one boundary cost.
const ACTION_BOUNDARY_SIZE_MINIMUM = ACTION_DELIMITER_COUNT * CUSTOM_DELIMITER_SIZE_MINIMUM

// ACTION_CONTENT_SIZE_MAXIMUM reserves both shortest hostile-input boundaries.
const ACTION_CONTENT_SIZE_MAXIMUM = SOURCE_SIZE_MAXIMUM - ACTION_BOUNDARY_SIZE_MINIMUM

// ACTION_START_MINIMUM permits action at first source byte.
const ACTION_START_MINIMUM = SOURCE_SIZE_MINIMUM

// ACTION_START_MAXIMUM reserves both shortest action boundaries.
const ACTION_START_MAXIMUM = ACTION_CONTENT_SIZE_MAXIMUM

// ACTION_CONTENT_BOUNDARY_MINIMUM reserves shortest opening delimiter.
const ACTION_CONTENT_BOUNDARY_MINIMUM = CUSTOM_DELIMITER_SIZE_MINIMUM

// ACTION_CONTENT_BOUNDARY_MAXIMUM reserves shortest closing delimiter.
const ACTION_CONTENT_BOUNDARY_MAXIMUM = SOURCE_SIZE_MAXIMUM - CUSTOM_DELIMITER_SIZE_MINIMUM

// ACTION_END_MINIMUM charges both shortest delimiters.
const ACTION_END_MINIMUM = ACTION_BOUNDARY_SIZE_MINIMUM

// ACTION_END_MAXIMUM permits action ending at source boundary.
const ACTION_END_MAXIMUM = SOURCE_SIZE_MAXIMUM

// ACTION_SCAN_START_MINIMUM follows shortest opening action boundary.
const ACTION_SCAN_START_MINIMUM = ACTION_CONTENT_BOUNDARY_MINIMUM

// ACTION_SCAN_START_MAXIMUM reserves one byte before source end.
const ACTION_SCAN_START_MAXIMUM = SOURCE_SIZE_MAXIMUM - utf8.CHARACTER_SIZE_MINIMUM

// ACTION_SCAN_LIMIT_MINIMUM leaves first scanned byte present.
const ACTION_SCAN_LIMIT_MINIMUM = ACTION_SCAN_START_MINIMUM + utf8.CHARACTER_SIZE_MINIMUM

// ACTION_SCAN_LIMIT_MAXIMUM reaches bounded source terminal boundary.
const ACTION_SCAN_LIMIT_MAXIMUM = SOURCE_SIZE_MAXIMUM

// LEXEME_START_MINIMUM follows first legal action-content byte.
const LEXEME_START_MINIMUM = ACTION_CONTENT_BOUNDARY_MINIMUM

// LEXEME_START_MAXIMUM reserves at least one token byte.
const LEXEME_START_MAXIMUM = ACTION_CONTENT_BOUNDARY_MAXIMUM - utf8.CHARACTER_SIZE_MINIMUM

// IDENTIFIER_START_MINIMUM follows first legal lexeme byte.
const IDENTIFIER_START_MINIMUM = LEXEME_START_MINIMUM

// IDENTIFIER_START_MAXIMUM reserves shortest identifier.
const IDENTIFIER_START_MAXIMUM = LEXEME_START_MAXIMUM

// IDENTIFIER_END_MINIMUM closes shortest identifier.
const IDENTIFIER_END_MINIMUM = IDENTIFIER_START_MINIMUM + IDENTIFIER_SIZE_MINIMUM

// IDENTIFIER_END_MAXIMUM reaches final legal action-content boundary.
const IDENTIFIER_END_MAXIMUM = ACTION_CONTENT_BOUNDARY_MAXIMUM

// NUMBER_START_MINIMUM follows first legal action-content byte.
const NUMBER_START_MINIMUM = LEXEME_START_MINIMUM

// NUMBER_START_MAXIMUM reserves at least one numeric byte.
const NUMBER_START_MAXIMUM = LEXEME_START_MAXIMUM

// NUMBER_END_MINIMUM closes shortest numeric token.
const NUMBER_END_MINIMUM = NUMBER_START_MINIMUM + utf8.CHARACTER_SIZE_MINIMUM

// NUMBER_END_MAXIMUM reaches final legal action-content boundary.
const NUMBER_END_MAXIMUM = ACTION_CONTENT_BOUNDARY_MAXIMUM

// QUOTED_BOUNDARY_SIZE charges opening and closing quote bytes.
const QUOTED_BOUNDARY_SIZE = len(`""`)

// QUOTED_START_MINIMUM permits a standalone literal at source start.
const QUOTED_START_MINIMUM = SOURCE_SIZE_MINIMUM

// QUOTED_START_MAXIMUM reserves both quote bytes through source end.
const QUOTED_START_MAXIMUM = SOURCE_SIZE_MAXIMUM - QUOTED_BOUNDARY_SIZE

// QUOTED_END_MINIMUM closes shortest quoted token.
const QUOTED_END_MINIMUM = QUOTED_START_MINIMUM + QUOTED_BOUNDARY_SIZE

// QUOTED_END_MAXIMUM reaches bounded source terminal boundary.
const QUOTED_END_MAXIMUM = SOURCE_SIZE_MAXIMUM

// DOT_SOURCE_SIZE is shortest executable action term.
const DOT_SOURCE_SIZE = len(".")

// SIMPLE_ACTION_SIZE_MINIMUM uses one-byte custom boundaries around dot.
const SIMPLE_ACTION_SIZE_MINIMUM = ACTION_BOUNDARY_SIZE_MINIMUM + DOT_SOURCE_SIZE

// ACTION_NODE_COUNT stores action wrapper.
const ACTION_NODE_COUNT = NODE_COUNT_INCREMENT

// PIPELINE_NODE_COUNT stores action pipeline.
const PIPELINE_NODE_COUNT = NODE_COUNT_INCREMENT

// COMMAND_NODE_COUNT stores pipeline command.
const COMMAND_NODE_COUNT = NODE_COUNT_INCREMENT

// SCALAR_NODE_COUNT stores dot term.
const SCALAR_NODE_COUNT = NODE_COUNT_INCREMENT

// ACTION_PIPELINE_NODE_COUNT charges wrappers shared by every executable action.
const ACTION_PIPELINE_NODE_COUNT = ACTION_NODE_COUNT + PIPELINE_NODE_COUNT

// SIMPLE_ACTION_CONTAINER_NODE_COUNT charges every wrapper around one scalar.
const SIMPLE_ACTION_CONTAINER_NODE_COUNT = ACTION_PIPELINE_NODE_COUNT + COMMAND_NODE_COUNT

// SIMPLE_ACTION_NODE_COUNT is syntax produced by shortest executable action.
const SIMPLE_ACTION_NODE_COUNT = SIMPLE_ACTION_CONTAINER_NODE_COUNT + SCALAR_NODE_COUNT

// ROOT_NODE_COUNT stores document list.
const ROOT_NODE_COUNT = NODE_COUNT_INCREMENT

// TRAILING_TEXT_NODE_COUNT_MAXIMUM stores leftover bytes after packed actions.
const TRAILING_TEXT_NODE_COUNT_MAXIMUM = (SOURCE_SIZE_MAXIMUM%SIMPLE_ACTION_SIZE_MINIMUM +
	SIMPLE_ACTION_SIZE_MINIMUM - SOURCE_POSITION_INCREMENT) /
	SIMPLE_ACTION_SIZE_MINIMUM

// NODE_COUNT_MAXIMUM admits densest one-byte-delimited action packing.
const NODE_COUNT_MAXIMUM = ROOT_NODE_COUNT +
	SOURCE_SIZE_MAXIMUM/SIMPLE_ACTION_SIZE_MINIMUM*SIMPLE_ACTION_NODE_COUNT +
	TRAILING_TEXT_NODE_COUNT_MAXIMUM

// NODE_LIMIT_DEFAULT selects complete fixed arena capacity.
const NODE_LIMIT_DEFAULT Node_Limit = Node_Limit(NODE_COUNT_MINIMUM)

// NODE_LIMIT_UNVALIDATED_MAXIMUM admits first refused caller limit.
const NODE_LIMIT_UNVALIDATED_MAXIMUM = NODE_COUNT_MAXIMUM + NODE_COUNT_INCREMENT

// NODE_REFERENCE_MINIMUM is the absent reference sentinel.
const NODE_REFERENCE_MINIMUM = NODE_COUNT_MINIMUM

// NODE_REFERENCE_MAXIMUM names the final caller node slot.
const NODE_REFERENCE_MAXIMUM = NODE_COUNT_MAXIMUM

// PARENTHESIS_DEPTH_MINIMUM is action state outside parentheses.
const PARENTHESIS_DEPTH_MINIMUM = NODE_COUNT_MINIMUM

// PARENTHESIS_DEPTH_INCREMENT adds one open grouping frame.
const PARENTHESIS_DEPTH_INCREMENT = 1

// PARENTHESIS_DEPTH_INITIAL fixes the empty grouping stack representation.
const PARENTHESIS_DEPTH_INITIAL Parenthesis_Depth_Tracked_Value = PARENTHESIS_DEPTH_MINIMUM

// PARENTHESIS_PAIR_SIZE charges one closing byte for every retained opener.
const PARENTHESIS_PAIR_SIZE = len("()")

// PARENTHESIS_SOURCE_SIZE_AVAILABLE reserves shortest complete action before nesting.
const PARENTHESIS_SOURCE_SIZE_AVAILABLE = SOURCE_SIZE_MAXIMUM - SIMPLE_ACTION_SIZE_MINIMUM

// PARENTHESIS_DEPTH_MAXIMUM leaves delimiters and one executable term present.
const PARENTHESIS_DEPTH_MAXIMUM = PARENTHESIS_SOURCE_SIZE_AVAILABLE / PARENTHESIS_PAIR_SIZE

// VARIABLE_COUNT_MINIMUM is parser state containing only implicit root variable.
const VARIABLE_COUNT_MINIMUM = NODE_COUNT_MINIMUM

// VARIABLE_NAME_SIZE_MINIMUM permits standard root variable spelling.
const VARIABLE_NAME_SIZE_MINIMUM = len("$")

// VARIABLE_DECLARATION_OPERATOR_SIZE charges complete declaration operator.
const VARIABLE_DECLARATION_OPERATOR_SIZE = len(":=")

// VARIABLE_DECLARATION_PAYLOAD_SIZE_MINIMUM reserves name, operator, and value.
const VARIABLE_DECLARATION_PAYLOAD_SIZE_MINIMUM = VARIABLE_NAME_SIZE_MINIMUM +
	VARIABLE_DECLARATION_OPERATOR_SIZE + DOT_SOURCE_SIZE

// VARIABLE_DECLARATION_ACTION_SIZE_MINIMUM is shortest retained declaration.
const VARIABLE_DECLARATION_ACTION_SIZE_MINIMUM = ACTION_BOUNDARY_SIZE_MINIMUM +
	VARIABLE_DECLARATION_PAYLOAD_SIZE_MINIMUM

// VARIABLE_COUNT_MAXIMUM follows packed declarations that remain in root scope.
const SYNTAX_VARIABLE_COUNT_MAXIMUM = SOURCE_SIZE_MAXIMUM / VARIABLE_DECLARATION_ACTION_SIZE_MINIMUM

// PIPELINE_DECLARATION_COUNT_MINIMUM is one ordinary pipeline variable.
const PIPELINE_DECLARATION_COUNT_MINIMUM = 1

// PIPELINE_DECLARATION_COUNT_MAXIMUM follows range key and value declarations.
const PIPELINE_DECLARATION_COUNT_MAXIMUM = PIPELINE_DECLARATION_COUNT_MINIMUM +
	PIPELINE_DECLARATION_COUNT_MINIMUM

// CONTROL_DEPTH_MINIMUM is parser state outside a branch.
const CONTROL_DEPTH_MINIMUM = NODE_COUNT_MINIMUM

// CONTROL_OPEN_PAYLOAD_SIZE_MINIMUM reserves branch keyword and executable term.
const CONTROL_OPEN_PAYLOAD_SIZE_MINIMUM = len("if") + DOT_SOURCE_SIZE

// CONTROL_OPEN_ACTION_SIZE_MINIMUM is shortest executable if action.
const CONTROL_OPEN_ACTION_SIZE_MINIMUM = ACTION_BOUNDARY_SIZE_MINIMUM +
	CONTROL_OPEN_PAYLOAD_SIZE_MINIMUM

// CONTROL_END_ACTION_SIZE_MINIMUM is shortest action that releases one frame.
const CONTROL_END_ACTION_SIZE_MINIMUM = ACTION_BOUNDARY_SIZE_MINIMUM + len("end")

// CONTROL_DEPTH_MAXIMUM leaves one end action for every retained opener.
const CONTROL_DEPTH_MAXIMUM = SOURCE_SIZE_MAXIMUM /
	(CONTROL_OPEN_ACTION_SIZE_MINIMUM + CONTROL_END_ACTION_SIZE_MINIMUM)

// TEMPLATE_COUNT_MINIMUM is a source without named definitions.
const TEMPLATE_COUNT_MINIMUM = NODE_COUNT_MINIMUM

// ACTION_SEPARATOR_SIZE_MINIMUM separates a keyword from its first argument.
const ACTION_SEPARATOR_SIZE_MINIMUM = len(" ")

// TEMPLATE_NAME_SOURCE_SIZE_MINIMUM is one quoted one-byte definition name.
const TEMPLATE_NAME_SOURCE_SIZE_MINIMUM = QUOTED_BOUNDARY_SIZE + IDENTIFIER_SIZE_MINIMUM

// TEMPLATE_DEFINITION_OPEN_SIZE_MINIMUM is the shortest one-byte-delimited opener.
const TEMPLATE_DEFINITION_OPEN_SIZE_MINIMUM = ACTION_BOUNDARY_SIZE_MINIMUM +
	len(KEYWORD_DEFINE) + ACTION_SEPARATOR_SIZE_MINIMUM + TEMPLATE_NAME_SOURCE_SIZE_MINIMUM

// TEMPLATE_DEFINITION_END_SIZE_MINIMUM is the shortest one-byte-delimited close.
const TEMPLATE_DEFINITION_END_SIZE_MINIMUM = ACTION_BOUNDARY_SIZE_MINIMUM + len(KEYWORD_END)

// TEMPLATE_DEFINITION_SIZE_MINIMUM retains one empty named definition.
const TEMPLATE_DEFINITION_SIZE_MINIMUM = TEMPLATE_DEFINITION_OPEN_SIZE_MINIMUM +
	TEMPLATE_DEFINITION_END_SIZE_MINIMUM

// TEMPLATE_COUNT_MAXIMUM packs distinct one-byte names into bounded source.
const TEMPLATE_COUNT_MAXIMUM = SOURCE_SIZE_MAXIMUM / TEMPLATE_DEFINITION_SIZE_MINIMUM

// TEMPLATE_PREFIX_SOURCE_SIZE_MAXIMUM leaves one shortest definition at source end.
const TEMPLATE_PREFIX_SOURCE_SIZE_MAXIMUM = SOURCE_SIZE_MAXIMUM -
	TEMPLATE_DEFINITION_SIZE_MINIMUM

// TEMPLATE_PREFIX_ACTION_COUNT_MAXIMUM packs the densest nodes before a definition.
const TEMPLATE_PREFIX_ACTION_COUNT_MAXIMUM = TEMPLATE_PREFIX_SOURCE_SIZE_MAXIMUM /
	SIMPLE_ACTION_SIZE_MINIMUM

// TEMPLATE_PREFIX_REMAINDER_SIZE_MAXIMUM is text left by packed prefix actions.
const TEMPLATE_PREFIX_REMAINDER_SIZE_MAXIMUM = TEMPLATE_PREFIX_SOURCE_SIZE_MAXIMUM %
	SIMPLE_ACTION_SIZE_MINIMUM

// TEMPLATE_PREFIX_TEXT_NODE_COUNT_MAXIMUM retains any prefix remainder as text.
const TEMPLATE_PREFIX_TEXT_NODE_COUNT_MAXIMUM = (TEMPLATE_PREFIX_REMAINDER_SIZE_MAXIMUM +
	SIMPLE_ACTION_SIZE_MINIMUM - SOURCE_POSITION_INCREMENT) / SIMPLE_ACTION_SIZE_MINIMUM

// DOCUMENT_FIRST_TEMPLATE_MINIMUM is the absent root-relative offset.
const DOCUMENT_FIRST_TEMPLATE_MINIMUM = NODE_REFERENCE_MINIMUM

// DOCUMENT_FIRST_TEMPLATE_REFERENCE_MAXIMUM follows the densest legal definition prefix.
const DOCUMENT_FIRST_TEMPLATE_REFERENCE_MAXIMUM = ROOT_NODE_COUNT +
	TEMPLATE_PREFIX_ACTION_COUNT_MAXIMUM*SIMPLE_ACTION_NODE_COUNT +
	TEMPLATE_PREFIX_TEXT_NODE_COUNT_MAXIMUM + NODE_COUNT_INCREMENT

// DOCUMENT_FIRST_TEMPLATE_MAXIMUM is the largest root-relative definition offset.
const DOCUMENT_FIRST_TEMPLATE_MAXIMUM = DOCUMENT_FIRST_TEMPLATE_REFERENCE_MAXIMUM -
	ROOT_NODE_COUNT

// NO_NODE marks an absent child or sibling.
const NO_NODE Node_Reference = NODE_REFERENCE_MINIMUM

// CONFIGURATION_FIELD is the sole immutable-policy storage position.
const CONFIGURATION_FIELD = bytes.SLICE_SIZE_MINIMUM

// STORAGE_FIELD_COUNT_INCREMENT adds one fixed-array field.
const STORAGE_FIELD_COUNT_INCREMENT = 1

// CONFIGURATION_FIELD_COUNT prevents caller mutation from changing storage shape.
const CONFIGURATION_FIELD_COUNT = CONFIGURATION_FIELD + STORAGE_FIELD_COUNT_INCREMENT

// SOURCE_FIELD is the sole validated source-header position.
const SOURCE_FIELD = bytes.SLICE_SIZE_MINIMUM

// SOURCE_FIELD_COUNT keeps validated source header shape immutable during parsing.
const SOURCE_FIELD_COUNT = SOURCE_FIELD + STORAGE_FIELD_COUNT_INCREMENT

// WORKSPACE_FIELD is the sole caller-state pointer position.
const SYNTAX_WORKSPACE_FIELD = bytes.SLICE_SIZE_MINIMUM

// WORKSPACE_FIELD_COUNT fixes unvalidated pointer storage shape.
const SYNTAX_WORKSPACE_FIELD_COUNT = SYNTAX_WORKSPACE_FIELD + STORAGE_FIELD_COUNT_INCREMENT

// PARSE_COMMENTS retains comments as nodes.
const PARSE_COMMENTS Mode = Mode(utf8.CHARACTER_SIZE_MINIMUM) << bytes.SLICE_SIZE_MINIMUM

// SKIP_FUNCTION_CHECK defers identifier resolution to execution.
const SKIP_FUNCTION_CHECK Mode = PARSE_COMMENTS << utf8.CHARACTER_SIZE_MINIMUM

// MODE_MAXIMUM contains every supported parse mode bit.
const MODE_MAXIMUM = PARSE_COMMENTS | SKIP_FUNCTION_CHECK

// MODE_MINIMUM disables every optional parser feature.
const MODE_MINIMUM Mode = Mode(bytes.SLICE_SIZE_MINIMUM)

// STATUS_OK means the requested operation completed.
const PARSE_STATUS_OK = bytes.SLICE_SIZE_MINIMUM

// STATUS_INPUT_INVALID means hostile source exceeds the byte boundary.
const PARSE_STATUS_INPUT_INVALID = PARSE_STATUS_OK + utf8.CHARACTER_SIZE_MINIMUM

// STATUS_CONFIGURATION_INVALID means delimiter or mode policy is invalid.
const PARSE_STATUS_CONFIGURATION_INVALID = PARSE_STATUS_INPUT_INVALID + utf8.CHARACTER_SIZE_MINIMUM

// STATUS_WORKSPACE_INVALID means caller omitted syntax storage.
const PARSE_STATUS_WORKSPACE_INVALID = PARSE_STATUS_CONFIGURATION_INVALID +
	utf8.CHARACTER_SIZE_MINIMUM

// STATUS_SYNTAX_INVALID means source is not one complete template grammar.
const PARSE_STATUS_SYNTAX_INVALID = PARSE_STATUS_WORKSPACE_INVALID + utf8.CHARACTER_SIZE_MINIMUM

// STATUS_CAPACITY_EXCEEDED means valid syntax cannot fit caller workspace.
const PARSE_STATUS_CAPACITY_EXCEEDED = PARSE_STATUS_SYNTAX_INVALID + utf8.CHARACTER_SIZE_MINIMUM

// DIAGNOSTIC_NONE marks successful parsing.
const DIAGNOSTIC_NONE Diagnostic_Code = PARSE_STATUS_OK

// DIAGNOSTIC_INPUT_INVALID identifies rejected hostile source.
const DIAGNOSTIC_INPUT_INVALID Diagnostic_Code = PARSE_STATUS_INPUT_INVALID

// DIAGNOSTIC_CONFIGURATION_INVALID identifies rejected parser policy.
const DIAGNOSTIC_CONFIGURATION_INVALID Diagnostic_Code = PARSE_STATUS_CONFIGURATION_INVALID

// DIAGNOSTIC_WORKSPACE_INVALID identifies missing caller storage.
const DIAGNOSTIC_WORKSPACE_INVALID Diagnostic_Code = PARSE_STATUS_WORKSPACE_INVALID

// DIAGNOSTIC_SYNTAX_INVALID identifies malformed template grammar.
const DIAGNOSTIC_SYNTAX_INVALID Diagnostic_Code = PARSE_STATUS_SYNTAX_INVALID

// DIAGNOSTIC_CAPACITY_EXCEEDED identifies exhausted bounded storage.
const DIAGNOSTIC_CAPACITY_EXCEEDED Diagnostic_Code = PARSE_STATUS_CAPACITY_EXCEEDED

// Source_Unvalidated is hostile borrowed bytes before size validation.
type Source_Unvalidated []byte

// Source_Unvalidated_Invariants bounds validation witnesses themselves.
func Source_Unvalidated_Invariants(
	value Source_Unvalidated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), SOURCE_SIZE_MINIMUM, SOURCE_SIZE_UNVALIDATED_MAXIMUM,
		).
		Ensure()
}

// Source_Data is borrowed template input bytes.
type Source_Data []byte

// Source_Data_Invariants keeps all source positions representable.
func Source_Data_Invariants(value Source_Data, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), SOURCE_SIZE_MINIMUM, SOURCE_SIZE_MAXIMUM).
		Ensure()
}

// Source_Storage retains one validated borrowed source header.
type Source_Storage_Value []byte

// Source_Storage_Value_Invariants admits any already-validated borrowed header.
func Source_Storage_Value_Invariants(
	value Source_Storage_Value, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), SOURCE_SIZE_MINIMUM, SOURCE_SIZE_UNVALIDATED_MAXIMUM).
		Ensure()
}

// Source_Storage retains one validated borrowed source header.
type Source_Storage struct {
	// Value prevents unvalidated headers from entering validated source.
	Value Source_Storage_Value
}

// Source_Storage_Invariants fixes source-header shape after public validation.
func Source_Storage_Invariants(value Source_Storage, namespace aver.Namespace) {
	Source_Storage_Value_Invariants(value.Value, namespace)
}

// Source_Data_Stored separates source shape from hostile-header validation.
type Source_Data_Stored interface{}

// Source_Data_Stored_Invariants fixes validated source representation.
func Source_Data_Stored_Invariants(value Source_Data_Stored, _ aver.Namespace) {
	_, valid := value.(Source_Storage)
	aver.Always(valid == (value != nil), "Source data has expected storage type.")
}

// Source_Fields keeps borrowed data before public validation.
type Source_Fields struct {
	// Data retains one borrowed byte-slice header.
	Data Source_Storage
}

// Source_Fields_Invariants bounds hostile borrowed source storage.
func Source_Fields_Invariants(value Source_Fields, namespace aver.Namespace) {
	Source_Storage_Invariants(value.Data, namespace)
}

// Source keeps validated borrowed data shape-safe under caller mutation.
type Source Source_Fields

// Source_Invariants verifies source storage shape without trusting mutable header content.
func Source_Invariants(value Source, namespace aver.Namespace) {
	Source_Data_Stored_Invariants(Source_Data_Stored(value.Data), namespace)
}

// Source_Status reports accepted or oversized input.
type Source_Status uint8

// Source_Status_Invariants lists both source validation outcomes.
func Source_Status_Invariants(value Source_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value), uint8(PARSE_STATUS_OK), uint8(PARSE_STATUS_INPUT_INVALID)).
		Ensure()
}

// Opening_Delimiter is an empty default marker or validated borrowed opening marker.
type Opening_Delimiter []byte

// Opening_Delimiter_Invariants keeps opening-marker matching bounded.
func Opening_Delimiter_Invariants(
	value Opening_Delimiter, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), DELIMITER_SIZE_MINIMUM, DELIMITER_SIZE_MAXIMUM).
		Ensure()
}

// Right_Delimiter is an empty default marker or validated borrowed closing marker.
type Right_Delimiter []byte

// Right_Delimiter_Invariants keeps closing-marker matching bounded.
func Right_Delimiter_Invariants(value Right_Delimiter, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), DELIMITER_SIZE_MINIMUM, DELIMITER_SIZE_MAXIMUM).
		Ensure()
}

// Mode selects comment retention and function-name checking.
type Mode uint

// Mode_Invariants keeps hostile caller flag storage visible.
func Mode_Invariants(value Mode, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint(uint(value), bits.WORD_MINIMUM, bits.WORD_MAXIMUM).
		Ensure()
}

// Function_Name is a borrowed identifier presented to caller policy.
type Parse_Function_Name []byte

// Function_Name_Invariants excludes empty function identifiers.
func Parse_Function_Name_Invariants(value Parse_Function_Name, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), IDENTIFIER_SIZE_MINIMUM, ACTION_CONTENT_SIZE_MAXIMUM).
		Ensure()
}

// PARSE_FUNCTION_NAME_VALIDATED_FIELD selects one lexer-proven identifier.
const PARSE_FUNCTION_NAME_VALIDATED_FIELD = bytes.SLICE_SIZE_MINIMUM

// PARSE_FUNCTION_NAME_VALIDATED_FIELD_COUNT fixes identifier proof shape.
const PARSE_FUNCTION_NAME_VALIDATED_FIELD_COUNT = PARSE_FUNCTION_NAME_VALIDATED_FIELD +
	STORAGE_FIELD_COUNT_INCREMENT

// Parse_Function_Name_Validated carries one lexer-proven identifier by value.
type Parse_Function_Name_Validated_Value []byte

// Parse_Function_Name_Validated_Value_Invariants preserves lexer proof storage.
func Parse_Function_Name_Validated_Value_Invariants(
	value Parse_Function_Name_Validated_Value, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), IDENTIFIER_SIZE_MINIMUM, ACTION_CONTENT_SIZE_MAXIMUM).
		Ensure()
}

// Parse_Function_Name_Validated carries one lexer-proven identifier by value.
type Parse_Function_Name_Validated struct {
	// Value cannot share identity with unvalidated input.
	Value Parse_Function_Name_Proof_Stored
}

// Parse_Function_Name_Validated_Invariants checks proof ownership.
func Parse_Function_Name_Validated_Invariants(
	value Parse_Function_Name_Validated, namespace aver.Namespace,
) {
	Parse_Function_Name_Proof_Stored_Invariants(value.Value, namespace)
}

// Variable_Chain is borrowed variable name plus optional field selectors.
type Variable_Chain []byte

// Variable_Chain_Invariants keeps complete variable token inside action content.
func Variable_Chain_Invariants(value Variable_Chain, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), VARIABLE_NAME_SIZE_MINIMUM, ACTION_CONTENT_SIZE_MAXIMUM).
		Ensure()
}

// VARIABLE_CHAIN_VALIDATED_FIELD selects one lexer-proven variable token.
const VARIABLE_CHAIN_VALIDATED_FIELD = bytes.SLICE_SIZE_MINIMUM

// VARIABLE_CHAIN_VALIDATED_FIELD_COUNT fixes variable-token proof shape.
const VARIABLE_CHAIN_VALIDATED_FIELD_COUNT = VARIABLE_CHAIN_VALIDATED_FIELD +
	STORAGE_FIELD_COUNT_INCREMENT

// Variable_Chain_Validated carries one lexer-proven variable token by value.
type Variable_Chain_Validated_Value []byte

// Variable_Chain_Validated_Value_Invariants preserves lexer proof storage.
func Variable_Chain_Validated_Value_Invariants(
	value Variable_Chain_Validated_Value, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), VARIABLE_NAME_SIZE_MINIMUM, ACTION_CONTENT_SIZE_MAXIMUM).
		Ensure()
}

// Variable_Chain_Validated carries one lexer-proven variable token by value.
type Variable_Chain_Validated struct {
	// Value cannot share identity with unvalidated input.
	Value Variable_Chain_Proof_Stored
}

// Variable_Chain_Validated_Invariants checks proof ownership.
func Variable_Chain_Validated_Invariants(
	value Variable_Chain_Validated, namespace aver.Namespace,
) {
	Variable_Chain_Proof_Stored_Invariants(value.Value, namespace)
}

// Function_Known reports whether an identifier names an injected function.
type Function_Known bool

// Function_Known_Invariants covers accepted and unknown function names.
func Function_Known_Invariants(value Function_Known, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Caller policy recognizes the function name.").
		Ensure()
}

// Function_Function injects caller function-name policy without package storage.
type Function_Function func(name Parse_Function_Name) (known Function_Known)

// Function_Function_Invariants observes both absent and injected policy.
func Function_Function_Invariants(value Function_Function, namespace aver.Namespace) {
	Function_Function_Present_Invariants(
		Function_Function_Present(value != nil), namespace,
	)
}

// Function_Function_Present records injected policy presence.
type Function_Function_Present bool

// Function_Function_Present_Invariants observes both policy states.
func Function_Function_Present_Invariants(
	value Function_Function_Present, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Caller supplied function-name policy.").
		Ensure()
}

// Configuration_Input is hostile delimiter and parser policy.
type Configuration_Input struct {
	// Left is empty for standard delimiter or one borrowed custom marker.
	Left Left_Delimiter_Unvalidated
	// Right is empty for standard delimiter or one borrowed custom marker.
	Right Right_Delimiter_Unvalidated
	// Mode selects optional syntax behavior.
	Mode Mode
	// Function checks non-keyword identifiers during parsing.
	Function Function_Function
}

// Configuration_Input_Invariants keeps all hostile policy axes visible.
func Configuration_Input_Invariants(
	value Configuration_Input, namespace aver.Namespace,
) {
	Left_Delimiter_Unvalidated_Invariants(value.Left, namespace)
	Right_Delimiter_Unvalidated_Invariants(value.Right, namespace)
	Mode_Invariants(value.Mode, namespace)
	Function_Function_Invariants(value.Function, namespace)
}

// Left_Delimiter_Unvalidated is hostile borrowed opening-marker input.
type Left_Delimiter_Unvalidated []byte

// Left_Delimiter_Unvalidated_Invariants bounds opening-marker validation work.
func Left_Delimiter_Unvalidated_Invariants(
	value Left_Delimiter_Unvalidated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), DELIMITER_SIZE_MINIMUM,
			DELIMITER_SIZE_UNVALIDATED_MAXIMUM,
		).
		Ensure()
}

// Right_Delimiter_Unvalidated is hostile borrowed closing-marker input.
type Right_Delimiter_Unvalidated []byte

// Right_Delimiter_Unvalidated_Invariants bounds closing-marker validation work.
func Right_Delimiter_Unvalidated_Invariants(
	value Right_Delimiter_Unvalidated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), DELIMITER_SIZE_MINIMUM,
			DELIMITER_SIZE_UNVALIDATED_MAXIMUM,
		).
		Ensure()
}

// Opening_Delimiter_Storage prevents opening-marker storage from changing shape.
type Opening_Delimiter_Storage_Value []byte

// Opening_Delimiter_Storage_Value_Invariants admits validated borrowed headers.
func Opening_Delimiter_Storage_Value_Invariants(
	value Opening_Delimiter_Storage_Value, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), DELIMITER_SIZE_MINIMUM, DELIMITER_SIZE_UNVALIDATED_MAXIMUM).
		Ensure()
}

// Opening_Delimiter_Storage prevents opening-marker storage from changing shape.
type Opening_Delimiter_Storage struct {
	// Value keeps validated policy inside configuration.
	Value Opening_Delimiter_Storage_Value
}

// Opening_Delimiter_Storage_Invariants fixes one borrowed opening-marker field.
func Opening_Delimiter_Storage_Invariants(
	value Opening_Delimiter_Storage, namespace aver.Namespace,
) {
	Opening_Delimiter_Storage_Value_Invariants(value.Value, namespace)
}

// Right_Delimiter_Storage prevents closing-marker storage from changing shape.
type Right_Delimiter_Storage_Value []byte

// Right_Delimiter_Storage_Value_Invariants admits validated borrowed headers.
func Right_Delimiter_Storage_Value_Invariants(
	value Right_Delimiter_Storage_Value, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), DELIMITER_SIZE_MINIMUM, DELIMITER_SIZE_UNVALIDATED_MAXIMUM).
		Ensure()
}

// Right_Delimiter_Storage prevents closing-marker storage from changing shape.
type Right_Delimiter_Storage struct {
	// Value keeps validated policy inside configuration.
	Value Right_Delimiter_Storage_Value
}

// Right_Delimiter_Storage_Invariants fixes one borrowed closing-marker field.
func Right_Delimiter_Storage_Invariants(
	value Right_Delimiter_Storage, namespace aver.Namespace,
) {
	Right_Delimiter_Storage_Value_Invariants(value.Value, namespace)
}

// Mode_Storage_Value retains validated scalar policy storage.
type Mode_Storage_Value uint8

// Mode_Storage_Value_Invariants preserves policy proof storage.
func Mode_Storage_Value_Invariants(value Mode_Storage_Value, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), bits.WORD_8_MINIMUM, bits.WORD_8_MAXIMUM).
		Ensure()
}

// Mode_Storage prevents scalar policy from changing aggregate shape.
type Mode_Storage struct {
	// Value keeps validated policy inside configuration.
	Value Mode_Storage_Value
}

// Mode_Storage_Invariants fixes one mode field.
func Mode_Storage_Invariants(value Mode_Storage, namespace aver.Namespace) {
	Mode_Storage_Value_Invariants(value.Value, namespace)
}

// Function_Storage prevents callback policy from changing aggregate shape.
type Parse_Function_Storage_Value func(name Parse_Function_Name) (known Function_Known)

// Parse_Function_Storage prevents callback policy from changing aggregate shape.
type Parse_Function_Storage struct {
	// Value keeps injected policy inside configuration.
	Value Parse_Function_Storage_Value
}

// Function_Storage_Invariants fixes one callback field.
func Parse_Function_Storage_Invariants(value Parse_Function_Storage, _ aver.Namespace) {
	present := value.Value != nil
	aver.Always(present == present, "Function storage retains caller policy.")
}

// Configuration holds borrowed delimiters and injected name policy.
type Configuration Configuration_Fields

// Configuration_Invariants verifies fixed policy shape and bounded values.
func Configuration_Invariants(value Configuration, namespace aver.Namespace) {
	Configuration_Left_Stored_Invariants(Configuration_Left_Stored(value.Left), namespace)
	Configuration_Right_Stored_Invariants(Configuration_Right_Stored(value.Right), namespace)
	Configuration_Mode_Stored_Invariants(Configuration_Mode_Stored(value.Mode), namespace)
	Parse_Function_Storage_Invariants(value.Function, namespace)
}

// Configuration_Status reports accepted or invalid parser policy.
type Configuration_Status uint8

// Configuration_Status_Invariants lists both construction outcomes.
func Configuration_Status_Invariants(
	value Configuration_Status, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(PARSE_STATUS_OK),
			uint8(PARSE_STATUS_CONFIGURATION_INVALID),
		).
		Ensure()
}

// Configuration_Validity reports whether stored policy remains valid.
type Configuration_Validity bool

// Configuration_Validity_Invariants covers valid and forged policy.
func Configuration_Validity_Invariants(
	value Configuration_Validity, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Stored parser policy is valid.").
		Ensure()
}

// Left_Delimiter selects opening instead of closing marker policy.
type Left_Delimiter bool

// Left_Delimiter_Invariants covers both action marker directions.
func Left_Delimiter_Invariants(value Left_Delimiter, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Delimiter lookup selects the opening marker.").
		Ensure()
}

// Trim_Whitespace records an active standard trim marker.
type Trim_Whitespace bool

// Trim_Whitespace_Invariants covers preserved and removed adjacent whitespace.
func Trim_Whitespace_Invariants(value Trim_Whitespace, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "A standard trim marker removes adjacent whitespace.").
		Ensure()
}

// Delimiter_Match reports whether source begins with selected marker.
type Delimiter_Match bool

// Delimiter_Match_Invariants covers matching and nonmatching source positions.
func Delimiter_Match_Invariants(value Delimiter_Match, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Source position begins the selected delimiter.").
		Ensure()
}

// Input_Byte is one source byte classified for template whitespace.
type Input_Byte byte

// Input_Byte_Invariants covers the complete source-byte domain.
func Input_Byte_Invariants(value Input_Byte, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), bits.WORD_8_MINIMUM, bits.WORD_8_MAXIMUM).
		Ensure()
}

// ASCII_Byte is one single-byte UTF-8 character.
type ASCII_Byte byte

// ASCII_Byte_Invariants excludes multibyte UTF-8 prefixes.
func ASCII_Byte_Invariants(value ASCII_Byte, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), ASCII_BYTE_MINIMUM, ASCII_BYTE_MAXIMUM).
		Ensure()
}

// Lexeme_Start is nonempty token opening boundary.
type Lexeme_Start uint16

// Lexeme_Start_Invariants reserves at least one token byte.
func Lexeme_Start_Invariants(value Lexeme_Start, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(LEXEME_START_MINIMUM), uint16(LEXEME_START_MAXIMUM),
		).
		Ensure()
}

// Lexeme retains first byte and opening boundary for compound dispatch.
type Lexeme struct {
	// Start opens nonempty token.
	Start Lexeme_Start
	// Character selects compound grammar.
	Character Input_Byte
}

// Lexeme_Invariants composes compound dispatch input once.
func Lexeme_Invariants(value Lexeme, namespace aver.Namespace) {
	Lexeme_Start_Invariants(value.Start, namespace)
	Input_Byte_Invariants(value.Character, namespace)
}

// Lexeme_Stored carries one source character after cursor validation.
type Lexeme_Stored Lexeme

// Lexeme_Stored_Invariants preserves cursor-bounded lexical encodings.
func Lexeme_Stored_Invariants(value Lexeme_Stored, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value.Start), uint16(LEXEME_START_MINIMUM),
			uint16(LEXEME_START_MAXIMUM),
		).
		Range_Uint8(
			uint8(value.Character), bits.WORD_8_MINIMUM, bits.WORD_8_MAXIMUM,
		).
		Ensure()
}

// LEXEME_VALIDATED_FIELD selects one nonempty source lexeme.
const LEXEME_VALIDATED_FIELD = bytes.SLICE_SIZE_MINIMUM

// LEXEME_VALIDATED_FIELD_COUNT fixes trusted lexeme transport shape.
const LEXEME_VALIDATED_FIELD_COUNT = LEXEME_VALIDATED_FIELD +
	STORAGE_FIELD_COUNT_INCREMENT

// Lexeme_Validated carries one lexeme bounded by token_read.
type Lexeme_Validated Lexeme_Validated_Fields

// Lexeme_Validated_Invariants checks transport ownership after source validation.
func Lexeme_Validated_Invariants(value Lexeme_Validated, namespace aver.Namespace) {
	Lexeme_Proof_Stored_Invariants(Lexeme_Proof_Stored(value.Value), namespace)
}

// Whitespace reports one standard template whitespace byte.
type Whitespace bool

// Whitespace_Invariants covers whitespace and every retained source byte.
func Whitespace_Invariants(value Whitespace, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Source byte is standard template whitespace.").
		Ensure()
}

// Identifier_Character reports one ASCII identifier byte.
type Identifier_Character bool

// Identifier_Character_Invariants covers identifier and terminating bytes.
func Identifier_Character_Invariants(
	value Identifier_Character, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Source byte belongs to an identifier.").
		Ensure()
}

// Identifier_Size is valid UTF-8 byte width for one identifier character.
type Identifier_Size uint8

// Identifier_Size_Invariants lists UTF-8 encoding widths.
func Identifier_Size_Invariants(value Identifier_Size, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_4_Uint8(
			uint8(value), utf8.CHARACTER_SIZE_MINIMUM, utf8.CHARACTER_SIZE_TWO,
			utf8.CHARACTER_SIZE_THREE, utf8.CHARACTER_SIZE_MAXIMUM,
		).
		Ensure()
}

// Identifier_Start is identifier token opening boundary.
type Identifier_Start uint16

// Identifier_Start_Invariants reserves shortest identifier.
func Identifier_Start_Invariants(value Identifier_Start, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(IDENTIFIER_START_MINIMUM),
			uint16(IDENTIFIER_START_MAXIMUM),
		).
		Ensure()
}

// Identifier_End is identifier token terminal boundary.
type Identifier_End uint16

// Identifier_End_Invariants excludes empty identifier.
func Identifier_End_Invariants(value Identifier_End, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(IDENTIFIER_END_MINIMUM),
			uint16(IDENTIFIER_END_MAXIMUM),
		).
		Ensure()
}

// Identifier_Span retains one validated identifier token boundary pair.
type Identifier_Span struct {
	// Start opens identifier token.
	Start Identifier_Start
	// End closes identifier token.
	End Identifier_End
}

// Identifier_Span_Invariants composes identifier boundaries once per lexical chain.
func Identifier_Span_Invariants(value Identifier_Span, namespace aver.Namespace) {
	Identifier_Start_Invariants(value.Start, namespace)
	Identifier_End_Invariants(value.End, namespace)
}

// Identifier_Span_Stored carries one identifier after lexical validation.
type Identifier_Span_Stored Identifier_Span

// Identifier_Span_Stored_Invariants preserves validated identifier boundaries.
func Identifier_Span_Stored_Invariants(
	value Identifier_Span_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value.Start), uint16(IDENTIFIER_START_MINIMUM),
			uint16(IDENTIFIER_START_MAXIMUM),
		).
		Range_Uint16(
			uint16(value.End), uint16(IDENTIFIER_END_MINIMUM),
			uint16(IDENTIFIER_END_MAXIMUM),
		).
		Ensure()
}

// IDENTIFIER_SPAN_VALIDATED_FIELD selects one token-bounded identifier span.
const IDENTIFIER_SPAN_VALIDATED_FIELD = bytes.SLICE_SIZE_MINIMUM

// IDENTIFIER_SPAN_VALIDATED_FIELD_COUNT fixes trusted identifier transport shape.
const IDENTIFIER_SPAN_VALIDATED_FIELD_COUNT = IDENTIFIER_SPAN_VALIDATED_FIELD +
	STORAGE_FIELD_COUNT_INCREMENT

// Identifier_Span_Validated carries one span bounded by token_identifier.
type Identifier_Span_Validated Identifier_Span_Validated_Fields

// Identifier_Span_Validated_Invariants checks transport ownership after token bounds validation.
func Identifier_Span_Validated_Invariants(
	value Identifier_Span_Validated, namespace aver.Namespace,
) {
	Identifier_Span_Proof_Stored_Invariants(
		Identifier_Span_Proof_Stored(value.Value), namespace,
	)
}

// Identifier_Search_Position keeps search progress independent from its bound.
type Identifier_Search_Position uint16

// Identifier_Search_Position_Invariants bounds search progress.
func Identifier_Search_Position_Invariants(
	value Identifier_Search_Position, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(SOURCE_SIZE_MINIMUM), uint16(SOURCE_SIZE_MAXIMUM),
		).
		Ensure()
}

// Identifier_Search_End keeps search bound independent from progress.
type Identifier_Search_End uint16

// Identifier_Search_End_Invariants bounds search termination.
func Identifier_Search_End_Invariants(
	value Identifier_Search_End, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(SOURCE_SIZE_MINIMUM), uint16(SOURCE_SIZE_MAXIMUM),
		).
		Ensure()
}

// Identifier_Search_Span retains one nonempty lexer remainder.
type Identifier_Search_Span Identifier_Search_Fields

// Identifier_Search_Span_Invariants composes hostile search boundaries.
func Identifier_Search_Span_Invariants(
	value Identifier_Search_Span, namespace aver.Namespace,
) {
	Identifier_Search_Position_Storage_Invariants(value.Position, namespace)
	Identifier_Search_End_Storage_Invariants(value.End, namespace)
}

// IDENTIFIER_SEARCH_FIELD selects one proven lexer remainder.
const IDENTIFIER_SEARCH_FIELD = bytes.SLICE_SIZE_MINIMUM

// IDENTIFIER_SEARCH_FIELD_COUNT fixes search transport shape.
const IDENTIFIER_SEARCH_FIELD_COUNT = IDENTIFIER_SEARCH_FIELD + STORAGE_FIELD_COUNT_INCREMENT

// Identifier_Search_Validated carries one nonempty remainder by value.
type Identifier_Search_Validated struct {
	// Value cannot share identity with unvalidated input.
	Value Identifier_Search_Span
}

// Identifier_Search_Validated_Invariants checks proof transport ownership.
func Identifier_Search_Validated_Invariants(
	value Identifier_Search_Validated, namespace aver.Namespace,
) {
	Identifier_Search_Span_Invariants(value.Value, namespace)
}

// Number_Value_Validity reports whether a parsed numeral fits stdlib value domains.
type Number_Value_Validity bool

// Number_Value_Validity_Invariants covers representable and overflowing numerals.
func Number_Value_Validity_Invariants(
	value Number_Value_Validity, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Parsed numeral fits a standard template numeric value.").
		Ensure()
}

// Number_Base selects numeric digit alphabet.
type Number_Base uint8

// Number_Base_Invariants lists supported Go literal bases.
func Number_Base_Invariants(value Number_Base, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_4_Uint8(
			uint8(value), uint8(NUMBER_BASE_BINARY), uint8(NUMBER_BASE_OCTAL),
			uint8(NUMBER_BASE_DECIMAL), uint8(NUMBER_BASE_HEXADECIMAL),
		).
		Ensure()
}

// Quoted_Escape_Base is one radix used by Go quoted escapes.
type Quoted_Escape_Base uint8

// Quoted_Escape_Base_Invariants lists octal and hexadecimal escape radices.
func Quoted_Escape_Base_Invariants(value Quoted_Escape_Base, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(NUMBER_BASE_OCTAL), uint8(NUMBER_BASE_HEXADECIMAL),
		).
		Ensure()
}

// Number_Validity reports complete numeric token syntax.
type Number_Validity bool

// Number_Validity_Invariants covers accepted and malformed numeric text.
func Number_Validity_Invariants(value Number_Validity, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Numeric token follows Go literal syntax.").
		Ensure()
}

// Number_Component_Validity reports one real or imaginary component syntax.
type Number_Component_Validity bool

// Number_Component_Validity_Invariants covers valid and malformed components.
func Number_Component_Validity_Invariants(
	value Number_Component_Validity, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Numeric component has digits in its selected base.").
		Ensure()
}

// Number_Fraction_Marker reports whether a component contains a radix point.
type Number_Fraction_Marker bool

// Number_Fraction_Marker_Invariants covers integral and fractional components.
func Number_Fraction_Marker_Invariants(
	value Number_Fraction_Marker, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Numeric component contains a radix point.").
		Ensure()
}

// Number_Digit reports digit membership in selected base.
type Number_Digit bool

// Number_Digit_Invariants covers member and nonmember bytes.
func Number_Digit_Invariants(value Number_Digit, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Byte is a digit in selected numeric base.").
		Ensure()
}

// Number_Exponent_Marker reports base-specific exponent prefix.
type Number_Exponent_Marker bool

// Number_Exponent_Marker_Invariants covers exponent and ordinary bytes.
func Number_Exponent_Marker_Invariants(
	value Number_Exponent_Marker, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Byte starts exponent for selected numeric base.").
		Ensure()
}

// Number_Sequence_Mode selects required, prefix-underscored, or optional digits.
type Number_Sequence_Mode uint8

// Number_Sequence_Mode_Invariants lists numeric digit-run grammar contexts.
func Number_Sequence_Mode_Invariants(
	value Number_Sequence_Mode, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(NUMBER_SEQUENCE_REQUIRED),
			uint8(NUMBER_SEQUENCE_PREFIXED), uint8(NUMBER_SEQUENCE_OPTIONAL),
		).
		Ensure()
}

// Number_Prefix_Mode is the subset a base-prefix scan can select.
type Number_Prefix_Mode uint8

// Number_Prefix_Mode_Invariants excludes fractional-only digit policy.
func Number_Prefix_Mode_Invariants(
	value Number_Prefix_Mode, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(NUMBER_PREFIX_REQUIRED),
			uint8(NUMBER_PREFIX_PREFIXED),
		).
		Ensure()
}

// Number_Sequence_Validity reports digit and underscore placement validity.
type Number_Sequence_Validity bool

// Number_Sequence_Validity_Invariants covers accepted and malformed digit runs.
func Number_Sequence_Validity_Invariants(
	value Number_Sequence_Validity, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Numeric digit sequence has legal underscore placement.").
		Ensure()
}

// Quoted_Digit_Count selects octal, byte, or Unicode escape width.
type Quoted_Digit_Count uint8

// Quoted_Digit_Count_Invariants lists supported escape digit widths.
func Quoted_Digit_Count_Invariants(
	value Quoted_Digit_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_4_Uint8(
			uint8(value), uint8(QUOTED_HEX_DIGIT_COUNT),
			uint8(QUOTED_OCTAL_DIGIT_COUNT),
			uint8(QUOTED_SHORT_UNICODE_DIGIT_COUNT),
			uint8(QUOTED_LONG_UNICODE_DIGIT_COUNT),
		).
		Ensure()
}

// Quote selects character, interpreted string, or raw string grammar.
type Quote uint8

// Quote_Invariants lists every template quote byte.
func Quote_Invariants(value Quote, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(QUOTE_STRING), uint8(QUOTE_CHARACTER),
			uint8(QUOTE_RAW_STRING),
		).
		Ensure()
}

// Escaped_Quote selects quotes whose grammar admits escapes.
type Escaped_Quote uint8

// Escaped_Quote_Invariants excludes raw strings before escape scanning.
func Escaped_Quote_Invariants(value Escaped_Quote, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(QUOTE_STRING), uint8(QUOTE_CHARACTER),
		).
		Ensure()
}

// Quoted_Hex_Digit_Count selects hexadecimal escape widths.
type Quoted_Hex_Digit_Count uint8

// Quoted_Hex_Digit_Count_Invariants excludes octal width from hexadecimal scanner.
func Quoted_Hex_Digit_Count_Invariants(
	value Quoted_Hex_Digit_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(QUOTED_HEX_DIGIT_COUNT),
			uint8(QUOTED_SHORT_UNICODE_DIGIT_COUNT),
			uint8(QUOTED_LONG_UNICODE_DIGIT_COUNT),
		).
		Ensure()
}

// Quoted_Validity reports complete string or character literal validity.
type Quoted_Validity bool

// Quoted_Validity_Invariants covers accepted and malformed quoted literals.
func Quoted_Validity_Invariants(value Quoted_Validity, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Quoted token contains valid Go literal body.").
		Ensure()
}

// Quoted_Escape_Validity reports one escape-sequence validity.
type Quoted_Escape_Validity bool

// Quoted_Escape_Validity_Invariants covers accepted and malformed escapes.
func Quoted_Escape_Validity_Invariants(
	value Quoted_Escape_Validity, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Quoted escape decodes one valid character.").
		Ensure()
}

// Quoted_Escape_Value is one decoded byte or Unicode code point.
type Quoted_Escape_Value uint32

// Quoted_Escape_Value_Invariants covers complete Unicode scalar storage.
func Quoted_Escape_Value_Invariants(
	value Quoted_Escape_Value, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint32(uint32(value), bits.WORD_32_MINIMUM, uint32(utf8.RUNE_MAX)).
		Ensure()
}

// Quoted_Escape_Value_Pointer keeps decoded mutation typed.
type Quoted_Escape_Value_Pointer *Quoted_Escape_Value

// Quoted_Escape_Value_Pointer_Invariants composes present decoded storage.
func Quoted_Escape_Value_Pointer_Invariants(
	value Quoted_Escape_Value_Pointer, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Quoted_Escape_Value_Invariants(*value, namespace)
}

// Quoted_Escape_Form selects direct byte or UTF-8 character output.
type Quoted_Escape_Form uint8

// Quoted_Escape_Form_Invariants lists both escape output forms.
func Quoted_Escape_Form_Invariants(
	value Quoted_Escape_Form, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(QUOTED_ESCAPE_BYTE), uint8(QUOTED_ESCAPE_CHARACTER),
		).
		Ensure()
}

// QUOTED_ESCAPE_BYTE writes one literal byte.
const QUOTED_ESCAPE_BYTE Quoted_Escape_Form = Quoted_Escape_Form(bytes.SLICE_SIZE_MINIMUM)

// QUOTED_ESCAPE_CHARACTER writes one UTF-8 code point.
const QUOTED_ESCAPE_CHARACTER Quoted_Escape_Form = QUOTED_ESCAPE_BYTE + utf8.CHARACTER_SIZE_MINIMUM

// Template_Name_Output is complete caller-owned decoded-name storage.
type Template_Name_Output []byte

// Template_Name_Output_Invariants fixes scratch capacity presented as length.
func Template_Name_Output_Invariants(
	value Template_Name_Output, _ aver.Namespace,
) {
	aver.Always(
		len(value) == TEMPLATE_NAME_SIZE_MAXIMUM,
		"Template name decoding receives complete bounded scratch.",
	)
}

// Quoted_Output is complete caller-owned decoded literal storage.
type Quoted_Output []byte

// Quoted_Output_Invariants fixes scratch capacity for worst malformed UTF-8 growth.
func Quoted_Output_Invariants(value Quoted_Output, _ aver.Namespace) {
	aver.Always(
		len(value) == QUOTED_DECODED_SIZE_MAXIMUM,
		"Quoted decoding receives complete bounded scratch.",
	)
}

// Decoded_Output is complete caller-owned decoded text storage.
type Decoded_Output []byte

// Decoded_Output_Invariants fixes one formula shared by both decoders.
func Decoded_Output_Invariants(value Decoded_Output, _ aver.Namespace) {
	aver.Always(
		len(value) == QUOTED_DECODED_SIZE_MAXIMUM,
		"Decoded text receives complete bounded scratch.",
	)
}

// Quoted_Count is populated decoded literal bytes.
type Quoted_Count uint16

// Quoted_Count_Invariants bounds decoded text by source growth.
func Quoted_Count_Invariants(value Quoted_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(SOURCE_SIZE_MINIMUM),
			uint16(QUOTED_DECODED_SIZE_MAXIMUM),
		).
		Ensure()
}

// Template_Name_Count is populated decoded template-name bytes.
type Template_Name_Count uint16

// Template_Name_Count_Invariants bounds decoded names by source.
func Template_Name_Count_Invariants(
	value Template_Name_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(SOURCE_SIZE_MINIMUM),
			uint16(TEMPLATE_NAME_SIZE_MAXIMUM),
		).
		Ensure()
}

// Template_Name_Match reports decoded template-name equality.
type Template_Name_Match bool

// Template_Name_Match_Invariants covers equal and distinct names.
func Template_Name_Match_Invariants(
	value Template_Name_Match, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Decoded template names are equal.").
		Ensure()
}

// Template_Empty reports a body containing only whitespace or comments.
type Template_Empty bool

// Template_Empty_Invariants covers replaceable and concrete definitions.
func Template_Empty_Invariants(value Template_Empty, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Template definition has no executable content.").
		Ensure()
}

// Token_Terminator reports one boundary between action tokens.
type Token_Terminator bool

// Token_Terminator_Invariants covers token content and boundaries.
func Token_Terminator_Invariants(value Token_Terminator, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Source byte terminates the current action token.").
		Ensure()
}

// Node_Kind_Known reports whether a token is one scalar syntax node.
type Node_Kind_Known bool

// Node_Kind_Known_Invariants covers scalar terms and structural tokens.
func Node_Kind_Known_Invariants(value Node_Kind_Known, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Token maps directly to one syntax node.").
		Ensure()
}

// Keyword is one fixed standard action keyword.
type Keyword string

// Keyword_Invariants bounds every standard keyword spelling.
func Keyword_Invariants(value Keyword, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), KEYWORD_SIZE_MINIMUM, KEYWORD_SIZE_MAXIMUM).
		Ensure()
}

// Keyword_Match reports whether one source word equals a fixed keyword.
type Keyword_Match bool

// Keyword_Match_Invariants covers equal and distinct source words.
func Keyword_Match_Invariants(value Keyword_Match, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Source word equals the selected template keyword.").
		Ensure()
}

// KEYWORD_IF begins the shortest standard control action.
const KEYWORD_IF Keyword = "if"

// KEYWORD_NIL is the untyped nil constant.
const KEYWORD_NIL Keyword = "nil"

// KEYWORD_END closes a control or definition.
const KEYWORD_END Keyword = "end"

// KEYWORD_TRUE is the true boolean constant.
const KEYWORD_TRUE Keyword = "true"

// KEYWORD_ELSE begins alternate control output.
const KEYWORD_ELSE Keyword = "else"

// KEYWORD_WITH begins cursor rebinding.
const KEYWORD_WITH Keyword = "with"

// KEYWORD_FALSE is the false boolean constant.
const KEYWORD_FALSE Keyword = "false"

// KEYWORD_RANGE begins iteration.
const KEYWORD_RANGE Keyword = "range"

// KEYWORD_BREAK terminates current iteration.
const KEYWORD_BREAK Keyword = "break"

// KEYWORD_BLOCK defines and invokes one template block.
const KEYWORD_BLOCK Keyword = "block"

// KEYWORD_DEFINE begins a named template definition.
const KEYWORD_DEFINE Keyword = "define"

// KEYWORD_CONTINUE advances current iteration.
const KEYWORD_CONTINUE Keyword = "continue"

// KEYWORD_TEMPLATE invokes one named template.
const KEYWORD_TEMPLATE Keyword = "template"

// KEYWORD_SIZE_MINIMUM follows the shortest standard keyword.
const KEYWORD_SIZE_MINIMUM = len(KEYWORD_IF)

// KEYWORD_SIZE_MAXIMUM follows the longest standard keyword.
const KEYWORD_SIZE_MAXIMUM = len(KEYWORD_CONTINUE)

// Look_Ahead records one token retained by recursive-descent parsing.
type Look_Ahead bool

// Look_Ahead_Invariants covers fresh and retained token cursor state.
func Look_Ahead_Invariants(value Look_Ahead, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Token cursor retains one look-ahead token.").
		Ensure()
}

// Token_Kind identifies one lexical item inside an action.
type Token_Kind uint8

// Token_Kind_Invariants covers every bounded action token.
func Token_Kind_Invariants(value Token_Kind, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(
			uint8(value), uint8(TOKEN_KIND_MINIMUM), uint8(TOKEN_KIND_MAXIMUM),
		).
		Ensure()
}

// Control_Token_Kind is one branch-opening keyword proven by action dispatch.
type Control_Token_Kind uint8

// Control_Token_Kind_Invariants lists every branch-opening keyword.
func Control_Token_Kind_Invariants(value Control_Token_Kind, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(TOKEN_IF), uint8(TOKEN_RANGE), uint8(TOKEN_WITH),
		).
		Ensure()
}

// TOKEN_EOF marks the action-content boundary.
const TOKEN_EOF Token_Kind = Token_Kind(bytes.SLICE_SIZE_MINIMUM)

// TOKEN_KIND_INCREMENT adds one lexical kind.
const TOKEN_KIND_INCREMENT Token_Kind = 1

// TOKEN_ASSIGN marks variable assignment.
const TOKEN_ASSIGN Token_Kind = TOKEN_EOF + TOKEN_KIND_INCREMENT

// TOKEN_BOOLEAN marks true or false.
const TOKEN_BOOLEAN Token_Kind = TOKEN_ASSIGN + TOKEN_KIND_INCREMENT

// TOKEN_CHARACTER marks a quoted character constant.
const TOKEN_CHARACTER Token_Kind = TOKEN_BOOLEAN + TOKEN_KIND_INCREMENT

// TOKEN_COMMA separates range declarations.
const TOKEN_COMMA Token_Kind = TOKEN_CHARACTER + TOKEN_KIND_INCREMENT

// TOKEN_DECLARE marks variable declaration.
const TOKEN_DECLARE Token_Kind = TOKEN_COMMA + TOKEN_KIND_INCREMENT

// TOKEN_DOT marks current execution cursor.
const TOKEN_DOT Token_Kind = TOKEN_DECLARE + TOKEN_KIND_INCREMENT

// TOKEN_FIELD marks a dot-prefixed field.
const TOKEN_FIELD Token_Kind = TOKEN_DOT + TOKEN_KIND_INCREMENT

// TOKEN_IDENTIFIER marks a function name.
const TOKEN_IDENTIFIER Token_Kind = TOKEN_FIELD + TOKEN_KIND_INCREMENT

// TOKEN_LEFT_PAREN opens a nested pipeline.
const TOKEN_LEFT_PAREN Token_Kind = TOKEN_IDENTIFIER + TOKEN_KIND_INCREMENT

// TOKEN_NIL marks untyped nil.
const TOKEN_NIL Token_Kind = TOKEN_LEFT_PAREN + TOKEN_KIND_INCREMENT

// TOKEN_NUMBER marks integer, decimal, complex, or imaginary syntax.
const TOKEN_NUMBER Token_Kind = TOKEN_NIL + TOKEN_KIND_INCREMENT

// TOKEN_PIPE separates commands.
const TOKEN_PIPE Token_Kind = TOKEN_NUMBER + TOKEN_KIND_INCREMENT

// TOKEN_RAW_STRING marks a backquoted string.
const TOKEN_RAW_STRING Token_Kind = TOKEN_PIPE + TOKEN_KIND_INCREMENT

// TOKEN_RIGHT_PAREN closes a nested pipeline.
const TOKEN_RIGHT_PAREN Token_Kind = TOKEN_RAW_STRING + TOKEN_KIND_INCREMENT

// TOKEN_STRING marks a double-quoted string.
const TOKEN_STRING Token_Kind = TOKEN_RIGHT_PAREN + TOKEN_KIND_INCREMENT

// TOKEN_VARIABLE marks a dollar-prefixed variable.
const TOKEN_VARIABLE Token_Kind = TOKEN_STRING + TOKEN_KIND_INCREMENT

// TOKEN_BLOCK marks a block definition and invocation.
const TOKEN_BLOCK Token_Kind = TOKEN_VARIABLE + TOKEN_KIND_INCREMENT

// TOKEN_BREAK marks current range termination.
const TOKEN_BREAK Token_Kind = TOKEN_BLOCK + TOKEN_KIND_INCREMENT

// TOKEN_COMMENT marks one template comment.
const TOKEN_COMMENT Token_Kind = TOKEN_BREAK + TOKEN_KIND_INCREMENT

// TOKEN_CONTINUE marks current range iteration termination.
const TOKEN_CONTINUE Token_Kind = TOKEN_COMMENT + TOKEN_KIND_INCREMENT

// TOKEN_DEFINE marks one named template definition.
const TOKEN_DEFINE Token_Kind = TOKEN_CONTINUE + TOKEN_KIND_INCREMENT

// TOKEN_ELSE marks alternate branch body.
const TOKEN_ELSE Token_Kind = TOKEN_DEFINE + TOKEN_KIND_INCREMENT

// TOKEN_END marks branch or definition closure.
const TOKEN_END Token_Kind = TOKEN_ELSE + TOKEN_KIND_INCREMENT

// TOKEN_IF marks a conditional branch.
const TOKEN_IF Token_Kind = TOKEN_END + TOKEN_KIND_INCREMENT

// TOKEN_RANGE marks an iteration branch.
const TOKEN_RANGE Token_Kind = TOKEN_IF + TOKEN_KIND_INCREMENT

// TOKEN_TEMPLATE marks one named template invocation.
const TOKEN_TEMPLATE Token_Kind = TOKEN_RANGE + TOKEN_KIND_INCREMENT

// TOKEN_WITH marks a cursor-rebinding branch.
const TOKEN_WITH Token_Kind = TOKEN_TEMPLATE + TOKEN_KIND_INCREMENT

// TOKEN_KIND_MINIMUM is the action boundary token.
const TOKEN_KIND_MINIMUM = TOKEN_EOF

// TOKEN_KIND_MAXIMUM is the final lexical token kind.
const TOKEN_KIND_MAXIMUM = TOKEN_WITH

// TOKEN_BOUNDARY_MINIMUM follows first legal action-content byte.
const TOKEN_BOUNDARY_MINIMUM = ACTION_CONTENT_BOUNDARY_MINIMUM

// Token_Start is one token opening source boundary.
type Token_Start uint16

// Token_Start_Invariants keeps token openings inside bounded source.
func Token_Start_Invariants(value Token_Start, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(TOKEN_BOUNDARY_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM),
		).
		Ensure()
}

// Token_End is one token closing source boundary.
type Token_End uint16

// Token_End_Invariants keeps token closings inside bounded source.
func Token_End_Invariants(value Token_End, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(TOKEN_BOUNDARY_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM),
		).
		Ensure()
}

// Token is one borrowed action token span.
type Token struct {
	// Kind selects token grammar meaning.
	Kind Token_Kind
	// Start begins token bytes.
	Start Token_Start
	// End closes token bytes.
	End Token_End
}

// Token_Invariants bounds token kind and borrowed source span.
func Token_Invariants(value Token, namespace aver.Namespace) {
	Token_Kind_Invariants(value.Kind, namespace)
	Token_Start_Invariants(value.Start, namespace)
	Token_End_Invariants(value.End, namespace)
}

// Token_Stored carries lexical state after token validation.
type Token_Stored Token

// Token_Stored_Invariants preserves lexer-proven token encodings.
func Token_Stored_Invariants(value Token_Stored, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(
			uint8(value.Kind), uint8(TOKEN_KIND_MINIMUM), uint8(TOKEN_KIND_MAXIMUM),
		).
		Range_Uint16(
			uint16(value.Start), uint16(SOURCE_SIZE_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM),
		).
		Range_Uint16(
			uint16(value.End), uint16(SOURCE_SIZE_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM),
		).
		Ensure()
}

// TOKEN_VALIDATED_FIELD selects one token produced by token_read.
const TOKEN_VALIDATED_FIELD = bytes.SLICE_SIZE_MINIMUM

// TOKEN_VALIDATED_FIELD_COUNT fixes lexer-proven token transport shape.
const TOKEN_VALIDATED_FIELD_COUNT = TOKEN_VALIDATED_FIELD + STORAGE_FIELD_COUNT_INCREMENT

// Token_Validated_Value keeps optional failure output separate from an ordinary token.
type Token_Validated_Value Token

// Token_Validated_Value_Invariants admits absent failure output and one lexer token.
func Token_Validated_Value_Invariants(
	value Token_Validated_Value, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint8(
			uint8(value.Kind), uint8(TOKEN_KIND_MINIMUM), uint8(TOKEN_KIND_MAXIMUM),
		).
		Range_Uint16(
			uint16(value.Start), uint16(SOURCE_SIZE_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM),
		).
		Range_Uint16(
			uint16(value.End), uint16(SOURCE_SIZE_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM),
		).
		Ensure()
}

// Token_Validated carries one token accepted by lexer.
type Token_Validated Token_Validated_Fields

// Token_Validated_Invariants keeps lexer proof attached to token.
func Token_Validated_Invariants(value Token_Validated, namespace aver.Namespace) {
	Token_Proof_Stored_Invariants(Token_Proof_Stored(value.Value), namespace)
}

// CLASSIFIED_TOKEN_KIND_FIELD selects one lexer-classified token kind.
const CLASSIFIED_TOKEN_KIND_FIELD = bytes.SLICE_SIZE_MINIMUM

// CLASSIFIED_TOKEN_KIND_FIELD_COUNT fixes classifier transport shape.
const CLASSIFIED_TOKEN_KIND_FIELD_COUNT = CLASSIFIED_TOKEN_KIND_FIELD +
	STORAGE_FIELD_COUNT_INCREMENT

// Classified_Token_Kind_Value retains classifier proof storage.
type Classified_Token_Kind_Value uint8

// Classified_Token_Kind_Value_Invariants preserves the classifier encoding.
func Classified_Token_Kind_Value_Invariants(
	value Classified_Token_Kind_Value, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint8(
			uint8(value), uint8(TOKEN_KIND_MINIMUM), uint8(TOKEN_KIND_MAXIMUM),
		).
		Ensure()
}

// Classified_Token_Kind carries one kind selected from validated source.
type Classified_Token_Kind struct {
	// Value keeps classification proof attached to its kind.
	Value Classified_Token_Kind_Proof_Stored
}

// Classified_Token_Kind_Invariants checks lexer classification ownership.
func Classified_Token_Kind_Invariants(
	value Classified_Token_Kind, namespace aver.Namespace,
) {
	Classified_Token_Kind_Proof_Stored_Invariants(value.Value, namespace)
}

// CONTROL_ACTION_TOKEN_KIND_FIELD selects one control-dispatched token kind.
const CONTROL_ACTION_TOKEN_KIND_FIELD = bytes.SLICE_SIZE_MINIMUM

// CONTROL_ACTION_TOKEN_KIND_FIELD_COUNT fixes dispatch transport shape.
const CONTROL_ACTION_TOKEN_KIND_FIELD_COUNT = CONTROL_ACTION_TOKEN_KIND_FIELD +
	STORAGE_FIELD_COUNT_INCREMENT

// Control_Action_Token_Kind_Validated_Value retains dispatch proof storage.
type Control_Action_Token_Kind_Validated_Value uint8

// Control_Action_Token_Kind_Validated_Value_Invariants preserves dispatch encoding.
func Control_Action_Token_Kind_Validated_Value_Invariants(
	value Control_Action_Token_Kind_Validated_Value, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Holed_Uint8(
			uint8(value), uint8(TOKEN_BREAK), uint8(TOKEN_WITH),
			uint8(TOKEN_COMMENT), uint8(TOKEN_DEFINE), uint8(TOKEN_TEMPLATE),
		).
		Ensure()
}

// Control_Action_Token_Kind_Validated carries one control grammar token.
type Control_Action_Token_Kind_Validated struct {
	// Value keeps control dispatch proof attached to its kind.
	Value Control_Action_Token_Kind_Validated_Value
}

// Control_Action_Token_Kind_Validated_Invariants checks dispatch ownership.
func Control_Action_Token_Kind_Validated_Invariants(
	value Control_Action_Token_Kind_Validated, namespace aver.Namespace,
) {
	Control_Action_Token_Kind_Validated_Value_Invariants(value.Value, namespace)
}

// CLASSIFIED_NODE_KIND_FIELD selects one node kind returned by scalar classification.
const CLASSIFIED_NODE_KIND_FIELD = bytes.SLICE_SIZE_MINIMUM

// CLASSIFIED_NODE_KIND_FIELD_COUNT fixes classified node transport shape.
const CLASSIFIED_NODE_KIND_FIELD_COUNT = CLASSIFIED_NODE_KIND_FIELD +
	STORAGE_FIELD_COUNT_INCREMENT

// Classified_Node_Kind_Value retains node classification proof storage.
type Classified_Node_Kind_Value uint8

// Classified_Node_Kind_Value_Invariants preserves node classification encoding.
func Classified_Node_Kind_Value_Invariants(
	value Classified_Node_Kind_Value, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint8(
			uint8(value), uint8(NODE_KIND_MINIMUM), uint8(NODE_VARIABLE),
		).
		Ensure()
}

// Classified_Node_Kind carries one scalar node kind or the unknown sentinel.
type Classified_Node_Kind struct {
	// Value keeps classification proof attached to its kind.
	Value Classified_Node_Kind_Proof_Stored
}

// Classified_Node_Kind_Invariants checks classifier ownership.
func Classified_Node_Kind_Invariants(
	value Classified_Node_Kind, namespace aver.Namespace,
) {
	Classified_Node_Kind_Proof_Stored_Invariants(value.Value, namespace)
}

// Token_Node_Classification retains scalar node mapping and membership.
type Token_Node_Classification struct {
	// Kind is meaningful when Known is true.
	Kind Classified_Node_Kind
	// Known separates scalar terms from grammar punctuation.
	Known Node_Kind_Known
}

// Token_Node_Classification_Invariants composes classification fields once.
func Token_Node_Classification_Invariants(
	value Token_Node_Classification, namespace aver.Namespace,
) {
	Classified_Node_Kind_Invariants(value.Kind, namespace)
	Node_Kind_Known_Invariants(value.Known, namespace)
}

// TOKEN_NODE_CLASSIFICATION_FIELD selects one complete scalar classification.
const TOKEN_NODE_CLASSIFICATION_FIELD = bytes.SLICE_SIZE_MINIMUM

// TOKEN_NODE_CLASSIFICATION_FIELD_COUNT fixes classification transport shape.
const TOKEN_NODE_CLASSIFICATION_FIELD_COUNT = TOKEN_NODE_CLASSIFICATION_FIELD +
	STORAGE_FIELD_COUNT_INCREMENT

// Node_Classification_Validated carries one token classification.
type Node_Classification_Validated struct {
	// Value cannot share identity with unvalidated input.
	Value Token_Node_Classification
}

// Node_Classification_Validated_Invariants checks classifier ownership.
func Node_Classification_Validated_Invariants(
	value Node_Classification_Validated, namespace aver.Namespace,
) {
	Token_Node_Classification_Invariants(value.Value, namespace)
}

// Cursor_Position is next unread action byte.
type Cursor_Position uint16

// Cursor_Position_Invariants keeps reads inside bounded source.
func Cursor_Position_Invariants(
	value Cursor_Position, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(ACTION_CONTENT_BOUNDARY_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM),
		).
		Ensure()
}

// Cursor_End is action-content boundary.
type Cursor_End uint16

// Cursor_End_Invariants keeps action boundaries inside bounded source.
func Cursor_End_Invariants(value Cursor_End, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(ACTION_CONTENT_BOUNDARY_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM),
		).
		Ensure()
}

// Cursor_Last_Start is the latest consumed or rejected token opening.
type Cursor_Last_Start uint16

// Cursor_Last_Start_Invariants keeps diagnostic coordinates inside source.
func Cursor_Last_Start_Invariants(
	value Cursor_Last_Start, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(SOURCE_SIZE_MINIMUM), uint16(SOURCE_SIZE_MAXIMUM),
		).
		Ensure()
}

// Cursor_Has_Last records whether Last_Start identifies a token.
type Cursor_Has_Last bool

// Cursor_Has_Last_Invariants covers fresh and diagnosed token cursors.
func Cursor_Has_Last_Invariants(value Cursor_Has_Last, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Token cursor has a diagnostic token opening.").
		Ensure()
}

// CURSOR_FIELD selects one mutable lexer slot.
const CURSOR_FIELD = bytes.SLICE_SIZE_MINIMUM

// CURSOR_FIELD_COUNT fixes every mutable lexer slot shape.
const CURSOR_FIELD_COUNT = CURSOR_FIELD + STORAGE_FIELD_COUNT_INCREMENT

// Cursor_Position_Storage_Value retains mutable cursor-position storage.
type Cursor_Position_Storage_Value uint16

// Cursor_Position_Storage_Value_Invariants preserves the cursor encoding.
func Cursor_Position_Storage_Value_Invariants(
	value Cursor_Position_Storage_Value, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(SOURCE_SIZE_MINIMUM), uint16(SOURCE_SIZE_MAXIMUM),
		).
		Ensure()
}

// Cursor_Position_Storage owns the next unread boundary without a scalar pointer.
type Cursor_Position_Storage struct {
	// Value keeps mutation inside the cursor aggregate.
	Value Cursor_Position_Storage_Value
}

// Cursor_Position_Storage_Invariants checks bounded ownership, not transient position value.
func Cursor_Position_Storage_Invariants(
	value Cursor_Position_Storage, namespace aver.Namespace,
) {
	Cursor_Position_Storage_Value_Invariants(value.Value, namespace)
}

// Cursor_End_Storage_Value retains mutable cursor-boundary storage.
type Cursor_End_Storage_Value uint16

// Cursor_End_Storage_Value_Invariants preserves the boundary encoding.
func Cursor_End_Storage_Value_Invariants(
	value Cursor_End_Storage_Value, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(SOURCE_SIZE_MINIMUM), uint16(SOURCE_SIZE_MAXIMUM),
		).
		Ensure()
}

// Cursor_End_Storage owns the action boundary without a scalar pointer.
type Cursor_End_Storage struct {
	// Value keeps mutation inside the cursor aggregate.
	Value Cursor_End_Storage_Value
}

// Cursor_End_Storage_Invariants checks bounded ownership, not transient boundary value.
func Cursor_End_Storage_Invariants(value Cursor_End_Storage, namespace aver.Namespace) {
	Cursor_End_Storage_Value_Invariants(value.Value, namespace)
}

// Token_Storage owns one retained look-ahead token.
type Token_Storage struct {
	// Value keeps mutation inside the cursor aggregate.
	Value Token_Stored
}

// Token_Storage_Invariants leaves token semantics to the read that creates it.
func Token_Storage_Invariants(value Token_Storage, namespace aver.Namespace) {
	Token_Stored_Invariants(value.Value, namespace)
}

// Look_Ahead_Storage_Value retains mutable look-ahead storage.
type Look_Ahead_Storage_Value uint8

// Look_Ahead_Storage_Value_Invariants preserves the flag encoding.
func Look_Ahead_Storage_Value_Invariants(
	value Look_Ahead_Storage_Value, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), bits.WORD_8_MINIMUM, uint8(utf8.CHARACTER_SIZE_MINIMUM),
		).
		Ensure()
}

// Look_Ahead_Storage owns one retained-token flag.
type Look_Ahead_Storage struct {
	// Value keeps mutation inside the cursor aggregate.
	Value Look_Ahead_Storage_Value
}

// Look_Ahead_Storage_Invariants checks scalar storage shape.
func Look_Ahead_Storage_Invariants(value Look_Ahead_Storage, namespace aver.Namespace) {
	Look_Ahead_Storage_Value_Invariants(value.Value, namespace)
}

// Number_Workspace_Pointer_Storage owns one parser scratch reference.
type Number_Workspace_Pointer_Storage struct {
	// Value keeps mutation inside the cursor aggregate.
	Value Number_Workspace_Pointer
}

// Number_Workspace_Pointer_Storage_Invariants checks reference storage shape.
func Number_Workspace_Pointer_Storage_Invariants(
	value Number_Workspace_Pointer_Storage, namespace aver.Namespace,
) {
	Number_Workspace_Pointer_Invariants(value.Value, namespace)
}

// Function_Function_Storage owns one lexical function policy.
type Function_Function_Storage_Value func(name Parse_Function_Name) (known Function_Known)

// Function_Function_Storage owns one lexical function policy.
type Function_Function_Storage struct {
	// Value keeps injected policy inside the cursor aggregate.
	Value Function_Function_Storage_Value
}

// Function_Function_Storage_Invariants checks callback storage shape.
func Function_Function_Storage_Invariants(
	value Function_Function_Storage, _ aver.Namespace,
) {
	present := value.Value != nil
	aver.Always(present == present, "Cursor function storage retains caller policy.")
}

// Cursor_Last_Start_Storage_Value retains diagnostic-boundary storage.
type Cursor_Last_Start_Storage_Value uint16

// Cursor_Last_Start_Storage_Value_Invariants preserves the boundary encoding.
func Cursor_Last_Start_Storage_Value_Invariants(
	value Cursor_Last_Start_Storage_Value, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(SOURCE_SIZE_MINIMUM), uint16(SOURCE_SIZE_MAXIMUM),
		).
		Ensure()
}

// Cursor_Last_Start_Storage owns one diagnostic token boundary.
type Cursor_Last_Start_Storage struct {
	// Value keeps mutation inside the cursor aggregate.
	Value Cursor_Last_Start_Storage_Value
}

// Cursor_Last_Start_Storage_Invariants checks scalar storage shape.
func Cursor_Last_Start_Storage_Invariants(
	value Cursor_Last_Start_Storage, namespace aver.Namespace,
) {
	Cursor_Last_Start_Storage_Value_Invariants(value.Value, namespace)
}

// Cursor_Has_Last_Storage_Value retains diagnostic-presence storage.
type Cursor_Has_Last_Storage_Value uint8

// Cursor_Has_Last_Storage_Value_Invariants preserves the flag encoding.
func Cursor_Has_Last_Storage_Value_Invariants(
	value Cursor_Has_Last_Storage_Value, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), bits.WORD_8_MINIMUM, uint8(utf8.CHARACTER_SIZE_MINIMUM),
		).
		Ensure()
}

// Cursor_Has_Last_Storage owns one diagnostic-boundary flag.
type Cursor_Has_Last_Storage struct {
	// Value keeps mutation inside the cursor aggregate.
	Value Cursor_Has_Last_Storage_Value
}

// Cursor_Has_Last_Storage_Invariants checks scalar storage shape.
func Cursor_Has_Last_Storage_Invariants(
	value Cursor_Has_Last_Storage, namespace aver.Namespace,
) {
	Cursor_Has_Last_Storage_Value_Invariants(value.Value, namespace)
}

// Cursor_Position_Stored separates mutable position shape from its transient value.
type Cursor_Position_Stored interface{}

// Cursor_Position_Stored_Invariants fixes cursor-position representation.
func Cursor_Position_Stored_Invariants(value Cursor_Position_Stored, _ aver.Namespace) {
	_, valid := value.(Cursor_Position_Storage)
	aver.Always(valid == (value != nil), "Cursor position has expected storage type.")
}

// Cursor_End_Stored separates mutable boundary shape from its transient value.
type Cursor_End_Stored interface{}

// Cursor_End_Stored_Invariants fixes cursor-boundary representation.
func Cursor_End_Stored_Invariants(value Cursor_End_Stored, _ aver.Namespace) {
	_, valid := value.(Cursor_End_Storage)
	aver.Always(valid == (value != nil), "Cursor end has expected storage type.")
}

// Cursor_Token_Stored separates mutable token shape from lexical validation.
type Cursor_Token_Stored interface{}

// Cursor_Token_Stored_Invariants fixes look-ahead token representation.
func Cursor_Token_Stored_Invariants(value Cursor_Token_Stored, _ aver.Namespace) {
	_, valid := value.(Token_Storage)
	aver.Always(valid == (value != nil), "Cursor token has expected storage type.")
}

// Cursor_Look_Ahead_Stored separates mutable flag shape from its transient value.
type Cursor_Look_Ahead_Stored interface{}

// Cursor_Look_Ahead_Stored_Invariants fixes look-ahead flag representation.
func Cursor_Look_Ahead_Stored_Invariants(value Cursor_Look_Ahead_Stored, _ aver.Namespace) {
	_, valid := value.(Look_Ahead_Storage)
	aver.Always(valid == (value != nil), "Cursor look-ahead has expected storage type.")
}

// Cursor_Number_Stored separates scratch shape from active numeric validation.
type Cursor_Number_Stored interface{}

// Cursor_Number_Stored_Invariants fixes numeric scratch representation.
func Cursor_Number_Stored_Invariants(value Cursor_Number_Stored, _ aver.Namespace) {
	_, valid := value.(Number_Workspace_Pointer_Storage)
	aver.Always(valid == (value != nil), "Cursor number scratch has expected storage type.")
}

// Cursor_Last_Start_Stored separates diagnostic shape from its transient boundary.
type Cursor_Last_Start_Stored interface{}

// Cursor_Last_Start_Stored_Invariants fixes diagnostic-boundary representation.
func Cursor_Last_Start_Stored_Invariants(value Cursor_Last_Start_Stored, _ aver.Namespace) {
	_, valid := value.(Cursor_Last_Start_Storage)
	aver.Always(valid == (value != nil), "Cursor last start has expected storage type.")
}

// Cursor_Has_Last_Stored separates diagnostic shape from its transient flag.
type Cursor_Has_Last_Stored interface{}

// Cursor_Has_Last_Stored_Invariants fixes diagnostic-flag representation.
func Cursor_Has_Last_Stored_Invariants(value Cursor_Has_Last_Stored, _ aver.Namespace) {
	_, valid := value.(Cursor_Has_Last_Storage)
	aver.Always(valid == (value != nil), "Cursor has-last flag has expected storage type.")
}

// Token_Cursor_Fields is one action lexer before parser-state validation.
type Token_Cursor_Fields struct {
	// Position is next unread action byte.
	Position Cursor_Position_Storage
	// End is action-content boundary.
	End Cursor_End_Storage
	// Token retains look-ahead when Looked is true.
	Token Token_Storage
	// Looked distinguishes zero token from retained EOF.
	Looked Look_Ahead_Storage
	// Number owns exact bounded numeric validation scratch.
	Number Number_Workspace_Pointer_Storage
	// Function keeps keyword collision policy beside lexical state.
	Function Function_Function_Storage
	// Last_Start keeps the current diagnostic token opening.
	Last_Start Cursor_Last_Start_Storage
	// Has_Last distinguishes source start from absent token history.
	Has_Last Cursor_Has_Last_Storage
}

// Token_Cursor_Fields_Invariants bounds hostile lexer storage.
func Token_Cursor_Fields_Invariants(value Token_Cursor_Fields, namespace aver.Namespace) {
	Cursor_Position_Storage_Invariants(value.Position, namespace)
	Cursor_End_Storage_Invariants(value.End, namespace)
	Token_Storage_Invariants(value.Token, namespace)
	Look_Ahead_Storage_Invariants(value.Looked, namespace)
	Number_Workspace_Pointer_Storage_Invariants(value.Number, namespace)
	Function_Function_Storage_Invariants(value.Function, namespace)
	Cursor_Last_Start_Storage_Invariants(value.Last_Start, namespace)
	Cursor_Has_Last_Storage_Invariants(value.Has_Last, namespace)
}

// Token_Cursor is one bounded lexer after parser-state validation.
type Token_Cursor Token_Cursor_Fields

// Token_Cursor_Invariants bounds action position and retained token state.
func Token_Cursor_Invariants(value Token_Cursor, namespace aver.Namespace) {
	Cursor_Position_Stored_Invariants(Cursor_Position_Stored(value.Position), namespace)
	Cursor_End_Stored_Invariants(Cursor_End_Stored(value.End), namespace)
	Cursor_Token_Stored_Invariants(Cursor_Token_Stored(value.Token), namespace)
	Cursor_Look_Ahead_Stored_Invariants(Cursor_Look_Ahead_Stored(value.Looked), namespace)
	Cursor_Number_Stored_Invariants(Cursor_Number_Stored(value.Number), namespace)
	Function_Function_Storage_Invariants(value.Function, namespace)
	Cursor_Last_Start_Stored_Invariants(
		Cursor_Last_Start_Stored(value.Last_Start), namespace,
	)
	Cursor_Has_Last_Stored_Invariants(Cursor_Has_Last_Stored(value.Has_Last), namespace)
}

// Token_Cursor_Pointer keeps lexer mutation typed.
type Token_Cursor_Pointer *Token_Cursor

// Token_Cursor_Pointer_Invariants composes present lexer state.
func Token_Cursor_Pointer_Invariants(
	value Token_Cursor_Pointer, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Token_Cursor_Invariants(*value, namespace)
}

// Parenthesis_Depth is populated iterative pipeline-frame count.
type Parenthesis_Depth uint16

// Parenthesis_Depth_Invariants bounds parser nesting by source bytes.
func Parenthesis_Depth_Invariants(
	value Parenthesis_Depth, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(PARENTHESIS_DEPTH_MINIMUM),
			uint16(PARENTHESIS_DEPTH_MAXIMUM),
		).
		Ensure()
}

// PARENTHESIS_DEPTH_TRACKED_FIELD selects active parser nesting.
const PARENTHESIS_DEPTH_TRACKED_FIELD = bytes.SLICE_SIZE_MINIMUM

// PARENTHESIS_DEPTH_TRACKED_FIELD_COUNT fixes nesting transport shape.
const PARENTHESIS_DEPTH_TRACKED_FIELD_COUNT = PARENTHESIS_DEPTH_TRACKED_FIELD +
	STORAGE_FIELD_COUNT_INCREMENT

// Parenthesis_Depth_Tracked_Value retains frame-bound depth storage.
type Parenthesis_Depth_Tracked_Value uint16

// Parenthesis_Depth_Tracked_Value_Invariants preserves depth proof storage.
func Parenthesis_Depth_Tracked_Value_Invariants(
	value Parenthesis_Depth_Tracked_Value, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(PARENTHESIS_DEPTH_MINIMUM),
			uint16(PARENTHESIS_DEPTH_MAXIMUM),
		).
		Ensure()
}

// Parenthesis_Depth_Tracked carries parser-owned nesting by value.
type Parenthesis_Depth_Tracked struct {
	// Value keeps frame-stack proof attached to depth.
	Value Parenthesis_Depth_Proof_Stored
}

// Parenthesis_Depth_Tracked_Invariants checks ownership after frame bounds validation.
func Parenthesis_Depth_Tracked_Invariants(
	value Parenthesis_Depth_Tracked, namespace aver.Namespace,
) {
	Parenthesis_Depth_Proof_Stored_Invariants(value.Value, namespace)
}

// PIPELINE_TERM_FIELD selects one node allocated for the active command.
const PIPELINE_TERM_FIELD = bytes.SLICE_SIZE_MINIMUM

// PIPELINE_TERM_FIELD_COUNT fixes term transport shape.
const PIPELINE_TERM_FIELD_COUNT = PIPELINE_TERM_FIELD + STORAGE_FIELD_COUNT_INCREMENT

// Pipeline_Term_Validated_Value retains arena-bound term storage.
type Pipeline_Term_Validated_Value uint16

// Pipeline_Term_Validated_Value_Invariants preserves the term reference.
func Pipeline_Term_Validated_Value_Invariants(
	value Pipeline_Term_Validated_Value, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(NODE_REFERENCE_MINIMUM),
			uint16(NODE_REFERENCE_MAXIMUM),
		).
		Ensure()
}

// Pipeline_Term_Validated carries one allocated command term by value.
type Pipeline_Term_Validated struct {
	// Value keeps arena proof attached to the term.
	Value Pipeline_Term_Proof_Stored
}

// Pipeline_Term_Validated_Invariants checks ownership after arena allocation.
func Pipeline_Term_Validated_Invariants(
	value Pipeline_Term_Validated, namespace aver.Namespace,
) {
	Pipeline_Term_Proof_Stored_Invariants(value.Value, namespace)
}

// PIPELINE_ROOT_FIELD selects one allocated pipeline root.
const PIPELINE_ROOT_FIELD = bytes.SLICE_SIZE_MINIMUM

// PIPELINE_ROOT_FIELD_COUNT fixes root transport shape.
const PIPELINE_ROOT_FIELD_COUNT = PIPELINE_ROOT_FIELD + STORAGE_FIELD_COUNT_INCREMENT

// Pipeline_Root_Validated_Value retains arena-bound root storage.
type Pipeline_Root_Validated_Value uint16

// Pipeline_Root_Validated_Value_Invariants preserves the root reference.
func Pipeline_Root_Validated_Value_Invariants(
	value Pipeline_Root_Validated_Value, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(NODE_REFERENCE_MINIMUM),
			uint16(NODE_REFERENCE_MAXIMUM),
		).
		Ensure()
}

// Pipeline_Root_Validated carries one allocated pipeline root by value.
type Pipeline_Root_Validated struct {
	// Value keeps arena proof attached to the root.
	Value Pipeline_Root_Proof_Stored
}

// Pipeline_Root_Validated_Invariants checks ownership after arena allocation.
func Pipeline_Root_Validated_Invariants(
	value Pipeline_Root_Validated, namespace aver.Namespace,
) {
	Pipeline_Root_Proof_Stored_Invariants(value.Value, namespace)
}

// PIPELINE_BOUNDARY_FIELD selects one token terminal proven inside source.
const PIPELINE_BOUNDARY_FIELD = bytes.SLICE_SIZE_MINIMUM

// PIPELINE_BOUNDARY_FIELD_COUNT fixes boundary transport shape.
const PIPELINE_BOUNDARY_FIELD_COUNT = PIPELINE_BOUNDARY_FIELD + STORAGE_FIELD_COUNT_INCREMENT

// Pipeline_Boundary_Validated_Value retains token-boundary proof storage.
type Pipeline_Boundary_Validated_Value uint16

// Pipeline_Boundary_Validated_Value_Invariants preserves the token boundary.
func Pipeline_Boundary_Validated_Value_Invariants(
	value Pipeline_Boundary_Validated_Value, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(SOURCE_SIZE_MINIMUM), uint16(SOURCE_SIZE_MAXIMUM),
		).
		Ensure()
}

// Pipeline_Boundary_Validated carries one proven token terminal by value.
type Pipeline_Boundary_Validated struct {
	// Value keeps token proof attached to the boundary.
	Value Pipeline_Boundary_Proof_Stored
}

// Pipeline_Boundary_Validated_Invariants checks ownership after token validation.
func Pipeline_Boundary_Validated_Invariants(
	value Pipeline_Boundary_Validated, namespace aver.Namespace,
) {
	Pipeline_Boundary_Proof_Stored_Invariants(value.Value, namespace)
}

// Pipeline_Reference identifies current pipeline node.
type Pipeline_Reference uint16

// Pipeline_Reference_Invariants keeps current pipeline inside syntax arena.
func Pipeline_Reference_Invariants(
	value Pipeline_Reference, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(NODE_REFERENCE_MINIMUM),
			uint16(NODE_REFERENCE_MAXIMUM),
		).
		Ensure()
}

// Pipeline_Last_Child identifies final declaration or command.
type Pipeline_Last_Child uint16

// Pipeline_Last_Child_Invariants keeps pipeline child links inside syntax arena.
func Pipeline_Last_Child_Invariants(
	value Pipeline_Last_Child, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(NODE_REFERENCE_MINIMUM),
			uint16(NODE_REFERENCE_MAXIMUM),
		).
		Ensure()
}

// Pipeline_Command identifies active command or NO_NODE.
type Pipeline_Command uint16

// Pipeline_Command_Invariants keeps active command inside syntax arena.
func Pipeline_Command_Invariants(
	value Pipeline_Command, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(NODE_REFERENCE_MINIMUM),
			uint16(NODE_REFERENCE_MAXIMUM),
		).
		Ensure()
}

// Pipeline_Last_Argument identifies final active command argument.
type Pipeline_Last_Argument uint16

// Pipeline_Last_Argument_Invariants keeps command arguments inside syntax arena.
func Pipeline_Last_Argument_Invariants(
	value Pipeline_Last_Argument, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(NODE_REFERENCE_MINIMUM),
			uint16(NODE_REFERENCE_MAXIMUM),
		).
		Ensure()
}

// Pipeline_Frame retains one iterative parenthesized pipeline state.
type Pipeline_Frame struct {
	// Pipe is the frame's pipeline node.
	Pipe Pipeline_Reference
	// Last_Child links declarations and commands in lexical order.
	Last_Child Pipeline_Last_Child
	// Command is the active command or NO_NODE after a pipe.
	Command Pipeline_Command
	// Last_Argument links the next command argument.
	Last_Argument Pipeline_Last_Argument
}

// Pipeline_Frame_Invariants bounds every retained syntax reference.
func Pipeline_Frame_Invariants(value Pipeline_Frame, namespace aver.Namespace) {
	Pipeline_Reference_Invariants(value.Pipe, namespace)
	Pipeline_Last_Child_Invariants(value.Last_Child, namespace)
	Pipeline_Command_Invariants(value.Command, namespace)
	Pipeline_Last_Argument_Invariants(value.Last_Argument, namespace)
}

// Pipeline_Frame_Stored carries parser-owned frame state after bounds validation.
type Pipeline_Frame_Stored Pipeline_Frame

// Pipeline_Frame_Stored_Invariants preserves mutable arena references.
func Pipeline_Frame_Stored_Invariants(
	value Pipeline_Frame_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value.Pipe), uint16(NODE_REFERENCE_MINIMUM),
			uint16(NODE_REFERENCE_MAXIMUM),
		).
		Range_Uint16(
			uint16(value.Last_Child), uint16(NODE_REFERENCE_MINIMUM),
			uint16(NODE_REFERENCE_MAXIMUM),
		).
		Range_Uint16(
			uint16(value.Command), uint16(NODE_REFERENCE_MINIMUM),
			uint16(NODE_REFERENCE_MAXIMUM),
		).
		Range_Uint16(
			uint16(value.Last_Argument), uint16(NODE_REFERENCE_MINIMUM),
			uint16(NODE_REFERENCE_MAXIMUM),
		).
		Ensure()
}

// PIPELINE_FRAME_STORAGE_FIELD selects one mutable parenthesis frame.
const PIPELINE_FRAME_STORAGE_FIELD = bytes.SLICE_SIZE_MINIMUM

// PIPELINE_FRAME_STORAGE_FIELD_COUNT fixes frame ownership shape.
const PIPELINE_FRAME_STORAGE_FIELD_COUNT = PIPELINE_FRAME_STORAGE_FIELD +
	STORAGE_FIELD_COUNT_INCREMENT

// Pipeline_Frame_Storage owns one mutable frame without scalar pointers.
type Pipeline_Frame_Storage Pipeline_Frame_Storage_Fields

// Pipeline_Frame_Storage_Invariants checks ownership, not transient parser state.
func Pipeline_Frame_Storage_Invariants(
	value Pipeline_Frame_Storage, namespace aver.Namespace,
) {
	Pipeline_Frame_State_Stored_Invariants(
		Pipeline_Frame_State_Stored(value.Value), namespace,
	)
}

// Pipeline_Frame_Storage_Pointer keeps active frame mutation typed.
type Pipeline_Frame_Storage_Pointer *Pipeline_Frame_Storage

// Pipeline_Frame_Storage_Pointer_Invariants composes present frame storage.
func Pipeline_Frame_Storage_Pointer_Invariants(
	value Pipeline_Frame_Storage_Pointer, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Pipeline_Frame_Storage_Invariants(*value, namespace)
}

// Pipeline_Frames is a source-bounded iterative parenthesis stack.
type Pipeline_Frames []Pipeline_Frame_Storage

// Pipeline_Frames_Invariants fixes stack capacity while parser owns active content.
func Pipeline_Frames_Invariants(value Pipeline_Frames, _ aver.Namespace) {
	aver.Always(
		len(value) == PARENTHESIS_DEPTH_MAXIMUM,
		"Pipeline frame stack has source-bounded capacity.",
	)
	aver.Always(
		cap(value) == PARENTHESIS_DEPTH_MAXIMUM,
		"Pipeline frame stack cannot grow.",
	)
}

// Pipeline_Frames_Pointer keeps stack mutation typed.
type Pipeline_Frames_Pointer *Pipeline_Frames

// Pipeline_Frames_Pointer_Invariants composes present stack storage.
func Pipeline_Frames_Pointer_Invariants(
	value Pipeline_Frames_Pointer, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Pipeline_Frames_Invariants(*value, namespace)
}

// Variable_Count is populated lexical variable-reference count.
type Syntax_Variable_Count uint16

// Variable_Count_Invariants bounds declarations by source bytes.
func Syntax_Variable_Count_Invariants(value Syntax_Variable_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(VARIABLE_COUNT_MINIMUM),
			uint16(SYNTAX_VARIABLE_COUNT_MAXIMUM),
		).
		Ensure()
}

// Variable_Known reports whether a variable is in lexical scope.
type Variable_Known bool

// Variable_Known_Invariants covers declared and undefined variables.
func Variable_Known_Invariants(value Variable_Known, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Variable is in the current lexical scope.").
		Ensure()
}

// Pipeline_Declaration reports a declaration or assignment prefix.
type Pipeline_Declaration bool

// Pipeline_Declaration_Invariants covers command terms and declaration prefixes.
func Pipeline_Declaration_Invariants(
	value Pipeline_Declaration, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Variable begins a pipeline declaration or assignment.").
		Ensure()
}

// Pipeline_Boundary_Validity reports whether separator or terminator follows complete command.
type Pipeline_Boundary_Validity bool

// Pipeline_Boundary_Validity_Invariants covers complete and malformed command boundaries.
func Pipeline_Boundary_Validity_Invariants(
	value Pipeline_Boundary_Validity, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Pipeline boundary follows a complete command.").
		Ensure()
}

// Pipeline_Declaration_Count is one ordinary or two range variables.
type Pipeline_Declaration_Count uint8

// Pipeline_Declaration_Count_Invariants lists both declaration arities.
func Pipeline_Declaration_Count_Invariants(
	value Pipeline_Declaration_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(PIPELINE_DECLARATION_COUNT_MINIMUM),
			uint8(PIPELINE_DECLARATION_COUNT_MAXIMUM),
		).
		Ensure()
}

// Pipeline_First_Declaration retains first declaration-token storage.
type Pipeline_First_Declaration Token

// Pipeline_First_Declaration_Invariants preserves first token encoding.
func Pipeline_First_Declaration_Invariants(
	value Pipeline_First_Declaration, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint8(
			uint8(value.Kind), uint8(TOKEN_KIND_MINIMUM), uint8(TOKEN_KIND_MAXIMUM),
		).
		Range_Uint16(
			uint16(value.Start), uint16(TOKEN_BOUNDARY_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM),
		).
		Range_Uint16(
			uint16(value.End), uint16(TOKEN_BOUNDARY_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM),
		).
		Ensure()
}

// Pipeline_Second_Declaration retains second declaration-token storage.
type Pipeline_Second_Declaration Token

// Pipeline_Second_Declaration_Invariants preserves second token encoding.
func Pipeline_Second_Declaration_Invariants(
	value Pipeline_Second_Declaration, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint8(
			uint8(value.Kind), uint8(TOKEN_KIND_MINIMUM), uint8(TOKEN_KIND_MAXIMUM),
		).
		Range_Uint16(
			uint16(value.Start), uint16(SOURCE_SIZE_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM),
		).
		Range_Uint16(
			uint16(value.End), uint16(SOURCE_SIZE_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM),
		).
		Ensure()
}

// Pipeline_Declarations is exact stack-owned declaration scratch.
type Pipeline_Declarations Pipeline_Declarations_Fields

// Pipeline_Declarations_Invariants fixes declaration scratch shape.
func Pipeline_Declarations_Invariants(
	value Pipeline_Declarations, namespace aver.Namespace,
) {
	Pipeline_First_Declaration_Storage_Invariants(value.First, namespace)
	Pipeline_Second_Declaration_Storage_Invariants(value.Second, namespace)
}

// Variable_References is a source-bounded lexical variable stack.
type Variable_References []Node_Reference

// Variable_References_Invariants fixes stack capacity while parser owns active entries.
func Variable_References_Invariants(value Variable_References, _ aver.Namespace) {
	aver.Always(
		len(value) == SYNTAX_VARIABLE_COUNT_MAXIMUM,
		"Variable stack has source-bounded capacity.",
	)
	aver.Always(
		cap(value) == SYNTAX_VARIABLE_COUNT_MAXIMUM,
		"Variable stack cannot grow.",
	)
}

// PARSER_STATE_FIELD selects one mutable parser scalar slot.
const PARSER_STATE_FIELD = bytes.SLICE_SIZE_MINIMUM

// PARSER_STATE_FIELD_COUNT fixes mutable parser scalar ownership shape.
const PARSER_STATE_FIELD_COUNT = PARSER_STATE_FIELD + STORAGE_FIELD_COUNT_INCREMENT

// Syntax_Variable_Count_Storage_Value retains lexical-count storage.
type Syntax_Variable_Count_Storage_Value uint16

// Syntax_Variable_Count_Storage_Value_Invariants preserves count encoding.
func Syntax_Variable_Count_Storage_Value_Invariants(
	value Syntax_Variable_Count_Storage_Value, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(VARIABLE_COUNT_MINIMUM),
			uint16(SYNTAX_VARIABLE_COUNT_MAXIMUM),
		).
		Ensure()
}

// Syntax_Variable_Count_Storage owns one lexical stack cursor.
type Syntax_Variable_Count_Storage struct {
	// Value keeps mutation inside parser state.
	Value Syntax_Variable_Count_Stored
}

// Syntax_Variable_Count_Storage_Invariants checks ownership, not transient depth.
func Syntax_Variable_Count_Storage_Invariants(
	value Syntax_Variable_Count_Storage, namespace aver.Namespace,
) {
	Syntax_Variable_Count_Stored_Invariants(value.Value, namespace)
}

// Variable_State retains lexical declarations without owned names.
type Variable_State struct {
	// References point at variable nodes whose source spans are names.
	References Variable_References
	// Count bounds populated References.
	Count Syntax_Variable_Count_Storage
}

// Variable_State_Invariants bounds lexical stack shape and count.
func Variable_State_Invariants(value Variable_State, namespace aver.Namespace) {
	Variable_References_Invariants(value.References, namespace)
	Syntax_Variable_Count_Storage_Invariants(value.Count, namespace)
}

// Variable_State_Pointer keeps lexical mutation typed.
type Variable_State_Pointer *Variable_State

// Variable_State_Pointer_Invariants composes present lexical state.
func Variable_State_Pointer_Invariants(
	value Variable_State_Pointer, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Variable_State_Invariants(*value, namespace)
}

// Template_Count is named definitions stored in the syntax arena.
type Template_Count uint16

// Template_Count_Invariants bounds definitions by node capacity.
func Template_Count_Invariants(value Template_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(TEMPLATE_COUNT_MINIMUM),
			uint16(TEMPLATE_COUNT_MAXIMUM),
		).
		Ensure()
}

// Template_First identifies first named definition.
type Template_First uint16

// Template_First_Invariants keeps first definition inside syntax arena.
func Template_First_Invariants(value Template_First, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(NODE_REFERENCE_MINIMUM),
			uint16(NODE_REFERENCE_MAXIMUM),
		).
		Ensure()
}

// Template_Last identifies final named definition.
type Template_Last uint16

// Template_Last_Invariants keeps final definition inside syntax arena.
func Template_Last_Invariants(value Template_Last, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(NODE_REFERENCE_MINIMUM),
			uint16(NODE_REFERENCE_MAXIMUM),
		).
		Ensure()
}

// Template_First_Storage_Value retains definition-opening storage.
type Template_First_Storage_Value uint16

// Template_First_Storage_Value_Invariants preserves the opening reference.
func Template_First_Storage_Value_Invariants(
	value Template_First_Storage_Value, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(NODE_REFERENCE_MINIMUM),
			uint16(NODE_REFERENCE_MAXIMUM),
		).
		Ensure()
}

// Template_First_Storage owns one mutable definition-chain opening.
type Template_First_Storage struct {
	// Value keeps mutation inside parser state.
	Value Template_First_Storage_Value
}

// Template_First_Storage_Invariants checks ownership, not transient reference.
func Template_First_Storage_Invariants(
	value Template_First_Storage, namespace aver.Namespace,
) {
	Template_First_Storage_Value_Invariants(value.Value, namespace)
}

// Template_Last_Storage_Value retains definition-tail storage.
type Template_Last_Storage_Value uint16

// Template_Last_Storage_Value_Invariants preserves the tail reference.
func Template_Last_Storage_Value_Invariants(
	value Template_Last_Storage_Value, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(NODE_REFERENCE_MINIMUM),
			uint16(NODE_REFERENCE_MAXIMUM),
		).
		Ensure()
}

// Template_Last_Storage owns one mutable definition-chain tail.
type Template_Last_Storage struct {
	// Value keeps mutation inside parser state.
	Value Template_Last_Storage_Value
}

// Template_Last_Storage_Invariants checks ownership, not transient reference.
func Template_Last_Storage_Invariants(
	value Template_Last_Storage, namespace aver.Namespace,
) {
	Template_Last_Storage_Value_Invariants(value.Value, namespace)
}

// Template_Count_Storage_Value retains mutable definition-count storage.
type Template_Count_Storage_Value uint16

// Template_Count_Storage_Value_Invariants preserves the count encoding.
func Template_Count_Storage_Value_Invariants(
	value Template_Count_Storage_Value, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(TEMPLATE_COUNT_MINIMUM),
			uint16(TEMPLATE_COUNT_MAXIMUM),
		).
		Ensure()
}

// Template_Count_Storage owns one mutable definition count.
type Template_Count_Storage struct {
	// Value keeps mutation inside parser state.
	Value Template_Count_Storage_Value
}

// Template_Count_Storage_Invariants checks ownership, not transient count.
func Template_Count_Storage_Invariants(
	value Template_Count_Storage, namespace aver.Namespace,
) {
	Template_Count_Storage_Value_Invariants(value.Value, namespace)
}

// Template_Match_Reference identifies one prior equal-name definition.
type Template_Match_Reference uint16

// Template_Match_Reference_Invariants bounds equal-name lookup output.
func Template_Match_Reference_Invariants(
	value Template_Match_Reference, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(NODE_REFERENCE_MINIMUM),
			uint16(NODE_REFERENCE_MAXIMUM),
		).
		Ensure()
}

// Template_Previous_Reference identifies predecessor of an equal definition.
type Template_Previous_Reference uint16

// Template_Previous_Reference_Invariants bounds equal-name predecessor output.
func Template_Previous_Reference_Invariants(
	value Template_Previous_Reference, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(NODE_REFERENCE_MINIMUM),
			uint16(NODE_REFERENCE_MAXIMUM),
		).
		Ensure()
}

// TEMPLATE_REFERENCE_VALIDATED_FIELD selects one definition proven inside syntax.
const TEMPLATE_REFERENCE_VALIDATED_FIELD = bytes.SLICE_SIZE_MINIMUM

// TEMPLATE_REFERENCE_VALIDATED_FIELD_COUNT fixes definition transport shape.
const TEMPLATE_REFERENCE_VALIDATED_FIELD_COUNT = TEMPLATE_REFERENCE_VALIDATED_FIELD +
	STORAGE_FIELD_COUNT_INCREMENT

// Template_Reference_Validated_Value retains arena-bound definition storage.
type Template_Reference_Validated_Value uint16

// Template_Reference_Validated_Value_Invariants preserves the definition reference.
func Template_Reference_Validated_Value_Invariants(
	value Template_Reference_Validated_Value, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(NODE_REFERENCE_MINIMUM),
			uint16(NODE_REFERENCE_MAXIMUM),
		).
		Ensure()
}

// Template_Reference_Validated carries one existing definition by value.
type Template_Reference_Validated struct {
	// Value keeps arena proof attached to the reference.
	Value Template_Reference_Proof_Stored
}

// Template_Reference_Validated_Invariants checks ownership after arena validation.
func Template_Reference_Validated_Invariants(
	value Template_Reference_Validated, namespace aver.Namespace,
) {
	Template_Reference_Proof_Stored_Invariants(value.Value, namespace)
}

// Template_Search retains one optional match and its optional predecessor.
type Template_Search struct {
	// Match is prior equal-name definition or NO_NODE.
	Match Template_Match_Reference
	// Previous is match predecessor or NO_NODE.
	Previous Template_Previous_Reference
}

// Template_Search_Invariants composes hostile search references.
func Template_Search_Invariants(value Template_Search, namespace aver.Namespace) {
	Template_Match_Reference_Invariants(value.Match, namespace)
	Template_Previous_Reference_Invariants(value.Previous, namespace)
}

// Template_Search_Stored carries lookup references after arena traversal.
type Template_Search_Stored Template_Search

// Template_Search_Stored_Invariants preserves validated lookup references.
func Template_Search_Stored_Invariants(
	value Template_Search_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value.Match), uint16(NODE_REFERENCE_MINIMUM),
			uint16(NODE_REFERENCE_MAXIMUM),
		).
		Range_Uint16(
			uint16(value.Previous), uint16(NODE_REFERENCE_MINIMUM),
			uint16(NODE_REFERENCE_MAXIMUM),
		).
		Ensure()
}

// TEMPLATE_SEARCH_VALIDATED_FIELD selects one arena-bounded search result.
const TEMPLATE_SEARCH_VALIDATED_FIELD = bytes.SLICE_SIZE_MINIMUM

// TEMPLATE_SEARCH_VALIDATED_FIELD_COUNT fixes search-result transport shape.
const TEMPLATE_SEARCH_VALIDATED_FIELD_COUNT = TEMPLATE_SEARCH_VALIDATED_FIELD +
	STORAGE_FIELD_COUNT_INCREMENT

// Template_Search_Validated carries one definition search result by value.
type Template_Search_Validated Template_Search_Validated_Fields

// Template_Search_Validated_Invariants checks ownership after bounded traversal.
func Template_Search_Validated_Invariants(
	value Template_Search_Validated, namespace aver.Namespace,
) {
	Template_Search_Proof_Stored_Invariants(
		Template_Search_Proof_Stored(value.Value), namespace,
	)
}

// Template_State links named definitions outside the root output list.
type Template_State Template_State_Fields

// Template_State_Invariants bounds definition chain metadata.
func Template_State_Invariants(value Template_State, namespace aver.Namespace) {
	Template_First_Stored_Invariants(Template_First_Stored(value.First), namespace)
	Template_Last_Stored_Invariants(Template_Last_Stored(value.Last), namespace)
	Template_Count_Stored_Invariants(Template_Count_Stored(value.Count), namespace)
}

// Template_State_Pointer keeps definition mutation typed.
type Template_State_Pointer *Template_State

// Template_State_Pointer_Invariants composes present definition state.
func Template_State_Pointer_Invariants(
	value Template_State_Pointer, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Template_State_Invariants(*value, namespace)
}

// List_Reference identifies current output list.
type List_Reference uint16

// List_Reference_Invariants keeps current output list inside syntax arena.
func List_Reference_Invariants(value List_Reference, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(NODE_REFERENCE_MINIMUM),
			uint16(NODE_REFERENCE_MAXIMUM),
		).
		Ensure()
}

// List_Last_Child identifies final output-list child.
type List_Last_Child uint16

// List_Last_Child_Invariants keeps final list child inside syntax arena.
func List_Last_Child_Invariants(
	value List_Last_Child, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(NODE_REFERENCE_MINIMUM),
			uint16(NODE_REFERENCE_MAXIMUM),
		).
		Ensure()
}

// List_State retains one current list and its final populated child.
type List_State struct {
	// List is the current output node list.
	List List_Reference
	// Last_Child links the next lexical child.
	Last_Child List_Last_Child
}

// List_State_Invariants bounds current list references.
func List_State_Invariants(value List_State, namespace aver.Namespace) {
	List_Reference_Invariants(value.List, namespace)
	List_Last_Child_Invariants(value.Last_Child, namespace)
}

// List_State_Stored carries parser-owned list cursors after bounds validation.
type List_State_Stored List_State

// List_State_Stored_Invariants preserves mutable list references.
func List_State_Stored_Invariants(value List_State_Stored, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value.List), uint16(NODE_REFERENCE_MINIMUM),
			uint16(NODE_REFERENCE_MAXIMUM),
		).
		Range_Uint16(
			uint16(value.Last_Child), uint16(NODE_REFERENCE_MINIMUM),
			uint16(NODE_REFERENCE_MAXIMUM),
		).
		Ensure()
}

// LIST_STATE_STORAGE_FIELD selects one mutable output-list cursor.
const LIST_STATE_STORAGE_FIELD = bytes.SLICE_SIZE_MINIMUM

// LIST_STATE_STORAGE_FIELD_COUNT fixes list cursor ownership shape.
const LIST_STATE_STORAGE_FIELD_COUNT = LIST_STATE_STORAGE_FIELD +
	STORAGE_FIELD_COUNT_INCREMENT

// List_State_Storage owns one mutable list cursor without scalar pointers.
type List_State_Storage List_State_Storage_Fields

// List_State_Storage_Invariants checks ownership, not transient list references.
func List_State_Storage_Invariants(value List_State_Storage, namespace aver.Namespace) {
	List_State_Value_Stored_Invariants(List_State_Value_Stored(value.Value), namespace)
}

// List_State_Storage_Pointer keeps cursor mutation typed.
type List_State_Storage_Pointer *List_State_Storage

// List_State_Storage_Pointer_Invariants composes present cursor storage.
func List_State_Storage_Pointer_Invariants(
	value List_State_Storage_Pointer, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	List_State_Storage_Invariants(*value, namespace)
}

// Else_State records whether a control branch already has an alternate list.
type Else_State bool

// Else_State_Invariants covers branches with and without else lists.
func Else_State_Invariants(value Else_State, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Control branch already contains an else list.").
		Ensure()
}

// Control_Else_Form identifies malformed, list, conditional, or cursor alternate body.
type Control_Else_Form uint8

// Control_Else_Form_Invariants lists every grammar form after else.
func Control_Else_Form_Invariants(value Control_Else_Form, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_4_Uint8(
			uint8(value), uint8(CONTROL_ELSE_FORM_INVALID),
			uint8(CONTROL_ELSE_FORM_LIST), uint8(CONTROL_ELSE_FORM_IF),
			uint8(CONTROL_ELSE_FORM_WITH),
		).
		Ensure()
}

// CONTROL_ELSE_FORM_INVALID rejects every token outside else grammar.
const CONTROL_ELSE_FORM_INVALID Control_Else_Form = Control_Else_Form(bytes.SLICE_SIZE_MINIMUM)

// CONTROL_ELSE_FORM_INCREMENT adds one alternate-body grammar form.
const CONTROL_ELSE_FORM_INCREMENT Control_Else_Form = 1

// CONTROL_ELSE_FORM_LIST begins bare alternate body.
const CONTROL_ELSE_FORM_LIST Control_Else_Form = CONTROL_ELSE_FORM_INVALID +
	CONTROL_ELSE_FORM_INCREMENT

// CONTROL_ELSE_FORM_IF begins chained conditional body.
const CONTROL_ELSE_FORM_IF Control_Else_Form = CONTROL_ELSE_FORM_LIST +
	CONTROL_ELSE_FORM_INCREMENT

// CONTROL_ELSE_FORM_WITH begins chained cursor body.
const CONTROL_ELSE_FORM_WITH Control_Else_Form = CONTROL_ELSE_FORM_IF +
	CONTROL_ELSE_FORM_INCREMENT

// Shared_End records whether else branch closes with parent branch.
type Shared_End bool

// Shared_End_Invariants covers ordinary nesting and else-if syntax sugar.
func Shared_End_Invariants(value Shared_End, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Else branch shares its closing action with parent branch.").
		Ensure()
}

// Range_Scope reports whether current control stack contains iteration.
type Range_Scope bool

// Range_Scope_Invariants covers range and non-range control stacks.
func Range_Scope_Invariants(value Range_Scope, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Current control stack contains a range branch.").
		Ensure()
}

// CONTROL_DEPTH_INCREMENT adds one open branch frame.
const CONTROL_DEPTH_INCREMENT = 1

// Control_Depth is populated iterative branch-frame count.
type Control_Depth uint16

// Control_Depth_Invariants bounds branch nesting by source bytes.
func Control_Depth_Invariants(value Control_Depth, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(CONTROL_DEPTH_MINIMUM),
			uint16(CONTROL_DEPTH_MAXIMUM),
		).
		Ensure()
}

// CONTROL_DEPTH_TRACKED_FIELD selects depth tied to parser frame stack.
const CONTROL_DEPTH_TRACKED_FIELD = bytes.SLICE_SIZE_MINIMUM

// CONTROL_DEPTH_TRACKED_FIELD_COUNT fixes frame-bound depth transport shape.
const CONTROL_DEPTH_TRACKED_FIELD_COUNT = CONTROL_DEPTH_TRACKED_FIELD +
	STORAGE_FIELD_COUNT_INCREMENT

// Control_Depth_Tracked_Value retains branch-depth proof storage.
type Control_Depth_Tracked_Value uint16

// Control_Depth_Tracked_Value_Invariants preserves the depth encoding.
func Control_Depth_Tracked_Value_Invariants(
	value Control_Depth_Tracked_Value, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(CONTROL_DEPTH_MINIMUM), uint16(CONTROL_DEPTH_MAXIMUM),
		).
		Ensure()
}

// Control_Depth_Tracked carries depth updated with Control_Frames.
type Control_Depth_Tracked struct {
	// Value keeps frame-stack proof attached to depth.
	Value Control_Depth_Stored
}

// Control_Depth_Tracked_Invariants keeps frame-stack proof attached to depth.
func Control_Depth_Tracked_Invariants(
	value Control_Depth_Tracked, namespace aver.Namespace,
) {
	Control_Depth_Stored_Invariants(value.Value, namespace)
}

// Control_Branch identifies open branch node.
type Control_Branch uint16

// Control_Branch_Invariants keeps open branch inside syntax arena.
func Control_Branch_Invariants(value Control_Branch, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(NODE_REFERENCE_MINIMUM),
			uint16(NODE_REFERENCE_MAXIMUM),
		).
		Ensure()
}

// Control_Parent_List identifies enclosing output list.
type Control_Parent_List uint16

// Control_Parent_List_Invariants keeps enclosing list inside syntax arena.
func Control_Parent_List_Invariants(
	value Control_Parent_List, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(NODE_REFERENCE_MINIMUM),
			uint16(NODE_REFERENCE_MAXIMUM),
		).
		Ensure()
}

// Control_Parent_Last_Child identifies enclosing list's final child.
type Control_Parent_Last_Child uint16

// Control_Parent_Last_Child_Invariants keeps enclosing child inside syntax arena.
func Control_Parent_Last_Child_Invariants(
	value Control_Parent_Last_Child, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(NODE_REFERENCE_MINIMUM),
			uint16(NODE_REFERENCE_MAXIMUM),
		).
		Ensure()
}

// Control_Parent_Variable_Count is enclosing lexical declaration count.
type Control_Parent_Variable_Count uint16

// Control_Parent_Variable_Count_Invariants bounds enclosing lexical scope.
func Control_Parent_Variable_Count_Invariants(
	value Control_Parent_Variable_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(VARIABLE_COUNT_MINIMUM),
			uint16(SYNTAX_VARIABLE_COUNT_MAXIMUM),
		).
		Ensure()
}

// Control_Branch_Variable_Count is branch lexical declaration count.
type Control_Branch_Variable_Count uint16

// Control_Branch_Variable_Count_Invariants bounds branch lexical scope.
func Control_Branch_Variable_Count_Invariants(
	value Control_Branch_Variable_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(VARIABLE_COUNT_MINIMUM),
			uint16(SYNTAX_VARIABLE_COUNT_MAXIMUM),
		).
		Ensure()
}

// Control_Frame retains one open if, range, or with branch.
type Control_Frame struct {
	// Branch is the open control node.
	Branch Control_Branch
	// Parent_List restores enclosing output list at end.
	Parent_List Control_Parent_List
	// Parent_Last_Child restores enclosing lexical append position.
	Parent_Last_Child Control_Parent_Last_Child
	// Parent_Variable_Count restores enclosing declarations at end.
	Parent_Variable_Count Control_Parent_Variable_Count
	// Branch_Variable_Count restores pipeline declarations before else.
	Branch_Variable_Count Control_Branch_Variable_Count
	// Kind distinguishes range legality for break and continue.
	Kind Node_Kind
	// Has_Else rejects a second alternate branch.
	Has_Else Else_State
	// Shares_End makes else-if and else-with use parent closing action.
	Shares_End Shared_End
}

// Control_Frame_Invariants bounds every retained branch state field.
func Control_Frame_Invariants(value Control_Frame, namespace aver.Namespace) {
	Control_Branch_Invariants(value.Branch, namespace)
	Control_Parent_List_Invariants(value.Parent_List, namespace)
	Control_Parent_Last_Child_Invariants(value.Parent_Last_Child, namespace)
	Control_Parent_Variable_Count_Invariants(value.Parent_Variable_Count, namespace)
	Control_Branch_Variable_Count_Invariants(value.Branch_Variable_Count, namespace)
	Node_Kind_Invariants(value.Kind, namespace)
	Else_State_Invariants(value.Has_Else, namespace)
	Shared_End_Invariants(value.Shares_End, namespace)
}

// Control_Frame_Stored carries parser-owned branch state after bounds validation.
type Control_Frame_Stored Control_Frame

// Control_Frame_Stored_Invariants preserves mutable branch state.
func Control_Frame_Stored_Invariants(
	value Control_Frame_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value.Branch), uint16(NODE_REFERENCE_MINIMUM),
			uint16(NODE_REFERENCE_MAXIMUM),
		).
		Range_Uint16(
			uint16(value.Parent_List), uint16(NODE_REFERENCE_MINIMUM),
			uint16(NODE_REFERENCE_MAXIMUM),
		).
		Range_Uint16(
			uint16(value.Parent_Last_Child), uint16(NODE_REFERENCE_MINIMUM),
			uint16(NODE_REFERENCE_MAXIMUM),
		).
		Range_Uint16(
			uint16(value.Parent_Variable_Count), uint16(VARIABLE_COUNT_MINIMUM),
			uint16(SYNTAX_VARIABLE_COUNT_MAXIMUM),
		).
		Range_Uint16(
			uint16(value.Branch_Variable_Count), uint16(VARIABLE_COUNT_MINIMUM),
			uint16(SYNTAX_VARIABLE_COUNT_MAXIMUM),
		).
		Range_Uint8(
			uint8(value.Kind), uint8(NODE_KIND_MINIMUM), uint8(NODE_KIND_MAXIMUM),
		).
		Sometimes(bool(value.Has_Else), "Control branch has an alternate arm.").
		Sometimes(bool(value.Shares_End), "Control branch shares its parent's end action.").
		Ensure()
}

// CONTROL_FRAME_STORAGE_FIELD selects one mutable branch frame.
const CONTROL_FRAME_STORAGE_FIELD = bytes.SLICE_SIZE_MINIMUM

// CONTROL_FRAME_STORAGE_FIELD_COUNT fixes branch frame ownership shape.
const CONTROL_FRAME_STORAGE_FIELD_COUNT = CONTROL_FRAME_STORAGE_FIELD +
	STORAGE_FIELD_COUNT_INCREMENT

// Control_Frame_Storage owns one mutable branch frame without scalar pointers.
type Control_Frame_Storage Control_Frame_Storage_Fields

// Control_Frame_Storage_Invariants checks ownership, not transient branch state.
func Control_Frame_Storage_Invariants(
	value Control_Frame_Storage, namespace aver.Namespace,
) {
	Control_Frame_State_Stored_Invariants(
		Control_Frame_State_Stored(value.Value), namespace,
	)
}

// Control_Frame_Storage_Pointer keeps active branch mutation typed.
type Control_Frame_Storage_Pointer *Control_Frame_Storage

// Control_Frame_Storage_Pointer_Invariants composes present branch storage.
func Control_Frame_Storage_Pointer_Invariants(
	value Control_Frame_Storage_Pointer, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Control_Frame_Storage_Invariants(*value, namespace)
}

// Control_Frames is a source-bounded iterative branch stack.
type Control_Frames []Control_Frame_Storage

// Control_Frames_Invariants fixes stack capacity while parser owns active frames.
func Control_Frames_Invariants(value Control_Frames, _ aver.Namespace) {
	aver.Always(
		len(value) == CONTROL_DEPTH_MAXIMUM,
		"Control frame stack has source-bounded capacity.",
	)
	aver.Always(
		cap(value) == CONTROL_DEPTH_MAXIMUM,
		"Control frame stack cannot grow.",
	)
}

// Control_Frames_Pointer keeps branch stack mutation typed.
type Control_Frames_Pointer *Control_Frames

// Control_Frames_Pointer_Invariants composes present branch storage.
func Control_Frames_Pointer_Invariants(
	value Control_Frames_Pointer, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Control_Frames_Invariants(*value, namespace)
}

// Node_Kind identifies one standard text/template syntax node.
type Node_Kind uint8

// Node_Kind_Invariants covers the complete standard node-kind domain.
func Node_Kind_Invariants(value Node_Kind, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(
			uint8(value), uint8(NODE_KIND_MINIMUM), uint8(NODE_KIND_MAXIMUM),
		).
		Ensure()
}

// Control_Node_Kind is one branch-opening syntax node.
type Control_Node_Kind uint8

// Control_Node_Kind_Invariants lists every branch-opening node.
func Control_Node_Kind_Invariants(value Control_Node_Kind, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(NODE_IF), uint8(NODE_RANGE), uint8(NODE_WITH),
		).
		Ensure()
}

// NODE_TEXT identifies literal template text.
const NODE_TEXT Node_Kind = Node_Kind(NODE_COUNT_MINIMUM)

// NODE_KIND_INCREMENT adds one syntax kind without changing node population.
const NODE_KIND_INCREMENT Node_Kind = 1

// NODE_ACTION identifies one output pipeline action.
const NODE_ACTION Node_Kind = NODE_TEXT + NODE_KIND_INCREMENT

// NODE_BOOLEAN identifies a boolean constant.
const NODE_BOOLEAN Node_Kind = NODE_ACTION + NODE_KIND_INCREMENT

// NODE_CHAIN identifies fields selected from a parenthesized value.
const NODE_CHAIN Node_Kind = NODE_BOOLEAN + NODE_KIND_INCREMENT

// NODE_COMMAND identifies one pipeline command.
const NODE_COMMAND Node_Kind = NODE_CHAIN + NODE_KIND_INCREMENT

// NODE_DOT identifies current execution cursor.
const NODE_DOT Node_Kind = NODE_COMMAND + NODE_KIND_INCREMENT

// NODE_ELSE identifies an internal else boundary.
const NODE_ELSE Node_Kind = NODE_DOT + NODE_KIND_INCREMENT

// NODE_END identifies an internal end boundary.
const NODE_END Node_Kind = NODE_ELSE + NODE_KIND_INCREMENT

// NODE_FIELD identifies a dot-prefixed field chain.
const NODE_FIELD Node_Kind = NODE_END + NODE_KIND_INCREMENT

// NODE_IDENTIFIER identifies a function name.
const NODE_IDENTIFIER Node_Kind = NODE_FIELD + NODE_KIND_INCREMENT

// NODE_IF identifies a conditional branch.
const NODE_IF Node_Kind = NODE_IDENTIFIER + NODE_KIND_INCREMENT

// NODE_LIST identifies an ordered node list.
const NODE_LIST Node_Kind = NODE_IF + NODE_KIND_INCREMENT

// NODE_NIL identifies untyped nil.
const NODE_NIL Node_Kind = NODE_LIST + NODE_KIND_INCREMENT

// NODE_NUMBER identifies a numeric constant.
const NODE_NUMBER Node_Kind = NODE_NIL + NODE_KIND_INCREMENT

// NODE_PIPE identifies declarations and ordered commands.
const NODE_PIPE Node_Kind = NODE_NUMBER + NODE_KIND_INCREMENT

// NODE_RANGE identifies an iteration branch.
const NODE_RANGE Node_Kind = NODE_PIPE + NODE_KIND_INCREMENT

// NODE_STRING identifies a quoted or raw string constant.
const NODE_STRING Node_Kind = NODE_RANGE + NODE_KIND_INCREMENT

// NODE_TEMPLATE identifies a named template invocation.
const NODE_TEMPLATE Node_Kind = NODE_STRING + NODE_KIND_INCREMENT

// NODE_VARIABLE identifies a dollar-prefixed variable chain.
const NODE_VARIABLE Node_Kind = NODE_TEMPLATE + NODE_KIND_INCREMENT

// NODE_WITH identifies a cursor-rebinding branch.
const NODE_WITH Node_Kind = NODE_VARIABLE + NODE_KIND_INCREMENT

// NODE_COMMENT identifies a retained template comment.
const NODE_COMMENT Node_Kind = NODE_WITH + NODE_KIND_INCREMENT

// NODE_BREAK identifies current range termination.
const NODE_BREAK Node_Kind = NODE_COMMENT + NODE_KIND_INCREMENT

// NODE_CONTINUE identifies current range iteration termination.
const NODE_CONTINUE Node_Kind = NODE_BREAK + NODE_KIND_INCREMENT

// NODE_KIND_MINIMUM is the first standard node kind.
const NODE_KIND_MINIMUM = NODE_TEXT

// NODE_KIND_MAXIMUM is the final standard node kind.
const NODE_KIND_MAXIMUM = NODE_CONTINUE

// Source_Position is one byte boundary inside bounded source.
type Source_Position uint16

// Source_Position_Invariants includes source start and terminal boundary.
func Source_Position_Invariants(value Source_Position, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(SOURCE_SIZE_MINIMUM), uint16(SOURCE_SIZE_MAXIMUM),
		).
		Ensure()
}

// SOURCE_CURSOR_FIELD selects one parser position proven inside source.
const SOURCE_CURSOR_FIELD = bytes.SLICE_SIZE_MINIMUM

// SOURCE_CURSOR_FIELD_COUNT fixes parser cursor transport shape.
const SOURCE_CURSOR_FIELD_COUNT = SOURCE_CURSOR_FIELD + STORAGE_FIELD_COUNT_INCREMENT

// Source_Cursor_Validated_Value retains source-boundary proof storage.
type Source_Cursor_Validated_Value uint16

// Source_Cursor_Validated_Value_Invariants preserves the source boundary.
func Source_Cursor_Validated_Value_Invariants(
	value Source_Cursor_Validated_Value, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(SOURCE_SIZE_MINIMUM), uint16(SOURCE_SIZE_MAXIMUM),
		).
		Ensure()
}

// Source_Cursor_Validated carries one parser position by value.
type Source_Cursor_Validated struct {
	// Value keeps source proof attached to the cursor.
	Value Source_Cursor_Stored
}

// Source_Cursor_Validated_Invariants checks ownership after source validation.
func Source_Cursor_Validated_Invariants(
	value Source_Cursor_Validated, namespace aver.Namespace,
) {
	Source_Cursor_Stored_Invariants(value.Value, namespace)
}

// Text_Span_Stored_Start retains the parser-proven opening boundary.
type Text_Span_Stored_Start uint16

// Text_Span_Stored_Start_Invariants preserves the opening encoding.
func Text_Span_Stored_Start_Invariants(
	value Text_Span_Stored_Start, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(SOURCE_SIZE_MINIMUM), uint16(SOURCE_SIZE_MAXIMUM),
		).
		Ensure()
}

// Text_Span_Stored_End retains the parser-proven closing boundary.
type Text_Span_Stored_End uint16

// Text_Span_Stored_End_Invariants preserves the closing encoding.
func Text_Span_Stored_End_Invariants(
	value Text_Span_Stored_End, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(SOURCE_SIZE_MINIMUM), uint16(SOURCE_SIZE_MAXIMUM),
		).
		Ensure()
}

// Text_Span_Stored carries one literal interval after parser validation.
type Text_Span_Stored struct {
	// Start opens literal text.
	Start Text_Span_Stored_Start
	// End closes literal text.
	End Text_Span_Stored_End
}

// Text_Span_Stored_Invariants preserves validated literal boundaries.
func Text_Span_Stored_Invariants(value Text_Span_Stored, namespace aver.Namespace) {
	Text_Span_Stored_Start_Invariants(value.Start, namespace)
	Text_Span_Stored_End_Invariants(value.End, namespace)
}

// TEXT_SPAN_VALIDATED_FIELD selects one parser-proven literal interval.
const TEXT_SPAN_VALIDATED_FIELD = bytes.SLICE_SIZE_MINIMUM

// TEXT_SPAN_VALIDATED_FIELD_COUNT fixes literal interval transport shape.
const TEXT_SPAN_VALIDATED_FIELD_COUNT = TEXT_SPAN_VALIDATED_FIELD +
	STORAGE_FIELD_COUNT_INCREMENT

// Text_Span_Validated carries one literal interval by value.
type Text_Span_Validated Text_Span_Validated_Fields

// Text_Span_Validated_Invariants checks ownership after parser validation.
func Text_Span_Validated_Invariants(
	value Text_Span_Validated, namespace aver.Namespace,
) {
	Text_Span_Proof_Stored_Invariants(Text_Span_Proof_Stored(value.Value), namespace)
}

// Action_Scan_Start is quoted or comment opening byte.
type Action_Scan_Start uint16

// Action_Scan_Start_Invariants reserves one scanned source byte.
func Action_Scan_Start_Invariants(value Action_Scan_Start, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(ACTION_SCAN_START_MINIMUM),
			uint16(ACTION_SCAN_START_MAXIMUM),
		).
		Ensure()
}

// Action_Scan_Limit is quoted or comment search boundary.
type Action_Scan_Limit uint16

// Action_Scan_Limit_Invariants keeps search inside bounded source.
func Action_Scan_Limit_Invariants(value Action_Scan_Limit, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(ACTION_SCAN_LIMIT_MINIMUM),
			uint16(ACTION_SCAN_LIMIT_MAXIMUM),
		).
		Ensure()
}

// Action_Scan_Span retains bounded quoted or comment search input.
type Action_Scan_Span struct {
	// Start opens quoted or comment text.
	Start Action_Scan_Start
	// Limit stops search before unowned bytes.
	Limit Action_Scan_Limit
}

// Action_Scan_Span_Invariants composes search boundaries once.
func Action_Scan_Span_Invariants(value Action_Scan_Span, namespace aver.Namespace) {
	Action_Scan_Start_Invariants(value.Start, namespace)
	Action_Scan_Limit_Invariants(value.Limit, namespace)
}

// Action_Scan_Span_Stored carries one search interval after source validation.
type Action_Scan_Span_Stored Action_Scan_Span

// Action_Scan_Span_Stored_Invariants preserves validated scan boundaries.
func Action_Scan_Span_Stored_Invariants(
	value Action_Scan_Span_Stored, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value.Start), uint16(ACTION_SCAN_START_MINIMUM),
			uint16(ACTION_SCAN_START_MAXIMUM),
		).
		Range_Uint16(
			uint16(value.Limit), uint16(ACTION_SCAN_LIMIT_MINIMUM),
			uint16(ACTION_SCAN_LIMIT_MAXIMUM),
		).
		Ensure()
}

// ACTION_SCAN_SPAN_VALIDATED_FIELD selects one source-bounded scan span.
const ACTION_SCAN_SPAN_VALIDATED_FIELD = bytes.SLICE_SIZE_MINIMUM

// ACTION_SCAN_SPAN_VALIDATED_FIELD_COUNT fixes trusted scan transport shape.
const ACTION_SCAN_SPAN_VALIDATED_FIELD_COUNT = ACTION_SCAN_SPAN_VALIDATED_FIELD +
	STORAGE_FIELD_COUNT_INCREMENT

// Action_Scan_Span_Validated carries one span built from validated parser state.
type Action_Scan_Span_Validated Action_Scan_Span_Validated_Fields

// Action_Scan_Span_Validated_Invariants checks transport ownership after boundary validation.
func Action_Scan_Span_Validated_Invariants(
	value Action_Scan_Span_Validated, namespace aver.Namespace,
) {
	Action_Scan_Span_Proof_Stored_Invariants(
		Action_Scan_Span_Proof_Stored(value.Value), namespace,
	)
}

// ACTION_SCAN_END_FIELD selects one result bounded by its scan span.
const ACTION_SCAN_END_FIELD = bytes.SLICE_SIZE_MINIMUM

// ACTION_SCAN_END_FIELD_COUNT fixes scan-result transport shape.
const ACTION_SCAN_END_FIELD_COUNT = ACTION_SCAN_END_FIELD + STORAGE_FIELD_COUNT_INCREMENT

// Action_Scan_End_Validated_Value retains scan-boundary proof storage.
type Action_Scan_End_Validated_Value uint16

// Action_Scan_End_Validated_Value_Invariants preserves the scan boundary.
func Action_Scan_End_Validated_Value_Invariants(
	value Action_Scan_End_Validated_Value, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(SOURCE_SIZE_MINIMUM), uint16(SOURCE_SIZE_MAXIMUM),
		).
		Ensure()
}

// Action_Scan_End_Validated carries one bounded scan result by value.
type Action_Scan_End_Validated struct {
	// Value keeps scan proof attached to the boundary.
	Value Action_Scan_End_Proof_Stored
}

// Action_Scan_End_Validated_Invariants checks ownership after bounded scanning.
func Action_Scan_End_Validated_Invariants(
	value Action_Scan_End_Validated, namespace aver.Namespace,
) {
	Action_Scan_End_Proof_Stored_Invariants(value.Value, namespace)
}

// Number_Start is numeric token opening boundary.
type Number_Start uint16

// Number_Start_Invariants reserves one byte before numeric token end.
func Number_Start_Invariants(value Number_Start, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(NUMBER_START_MINIMUM), uint16(NUMBER_START_MAXIMUM),
		).
		Ensure()
}

// Number_End is numeric token terminal boundary.
type Number_End uint16

// Number_End_Invariants excludes empty numeric token.
func Number_End_Invariants(value Number_End, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(NUMBER_END_MINIMUM), uint16(NUMBER_END_MAXIMUM),
		).
		Ensure()
}

// Number_Span retains one validated numeric token boundary pair.
type Number_Span struct {
	// Start opens numeric token.
	Start Number_Start
	// End closes numeric token.
	End Number_End
}

// Number_Span_Invariants composes numeric boundaries once per validation chain.
func Number_Span_Invariants(value Number_Span, namespace aver.Namespace) {
	Number_Start_Invariants(value.Start, namespace)
	Number_End_Invariants(value.End, namespace)
}

// Number_Span_Stored carries one numeric interval after token validation.
type Number_Span_Stored Number_Span

// Number_Span_Stored_Invariants preserves validated numeric boundaries.
func Number_Span_Stored_Invariants(value Number_Span_Stored, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value.Start), uint16(NUMBER_START_MINIMUM),
			uint16(NUMBER_START_MAXIMUM),
		).
		Range_Uint16(
			uint16(value.End), uint16(NUMBER_END_MINIMUM),
			uint16(NUMBER_END_MAXIMUM),
		).
		Ensure()
}

// NUMBER_SPAN_VALIDATED_FIELD selects one token-bounded numeric span.
const NUMBER_SPAN_VALIDATED_FIELD = bytes.SLICE_SIZE_MINIMUM

// NUMBER_SPAN_VALIDATED_FIELD_COUNT fixes trusted numeric span transport shape.
const NUMBER_SPAN_VALIDATED_FIELD_COUNT = NUMBER_SPAN_VALIDATED_FIELD +
	STORAGE_FIELD_COUNT_INCREMENT

// Number_Span_Validated carries one span bounded by token_number.
type Number_Span_Validated Number_Span_Validated_Fields

// Number_Span_Validated_Invariants checks transport ownership after token bounds validation.
func Number_Span_Validated_Invariants(
	value Number_Span_Validated, namespace aver.Namespace,
) {
	Number_Span_Proof_Stored_Invariants(Number_Span_Proof_Stored(value.Value), namespace)
}

// NUMBER_POSITION_VALIDATED_FIELD selects one cursor bounded by numeric span.
const NUMBER_POSITION_VALIDATED_FIELD = bytes.SLICE_SIZE_MINIMUM

// NUMBER_POSITION_VALIDATED_FIELD_COUNT fixes numeric cursor transport shape.
const NUMBER_POSITION_VALIDATED_FIELD_COUNT = NUMBER_POSITION_VALIDATED_FIELD +
	STORAGE_FIELD_COUNT_INCREMENT

// Number_Position_Validated_Value retains numeric-position proof storage.
type Number_Position_Validated_Value uint16

// Number_Position_Validated_Value_Invariants preserves the numeric position.
func Number_Position_Validated_Value_Invariants(
	value Number_Position_Validated_Value, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(SOURCE_SIZE_MINIMUM), uint16(SOURCE_SIZE_MAXIMUM),
		).
		Ensure()
}

// Number_Position_Validated carries one cursor proven inside numeric token.
type Number_Position_Validated struct {
	// Value keeps span proof attached to the cursor.
	Value Number_Position_Stored
}

// Number_Position_Validated_Invariants keeps numeric-span proof attached to cursor.
func Number_Position_Validated_Invariants(
	value Number_Position_Validated, namespace aver.Namespace,
) {
	Number_Position_Stored_Invariants(value.Value, namespace)
}

// Quoted_Start_Unvalidated is hostile quoted opening position.
type Quoted_Start_Unvalidated uint16

// Quoted_Start_Unvalidated_Invariants includes first refused source position.
func Quoted_Start_Unvalidated_Invariants(
	value Quoted_Start_Unvalidated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(SOURCE_SIZE_MINIMUM),
			uint16(SOURCE_SIZE_UNVALIDATED_MAXIMUM),
		).
		Ensure()
}

// Quoted_End_Unvalidated is hostile quoted exclusive end.
type Quoted_End_Unvalidated uint16

// Quoted_End_Unvalidated_Invariants includes first refused source boundary.
func Quoted_End_Unvalidated_Invariants(
	value Quoted_End_Unvalidated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(SOURCE_SIZE_MINIMUM),
			uint16(SOURCE_SIZE_UNVALIDATED_MAXIMUM),
		).
		Ensure()
}

// Quoted_Span_Unvalidated is hostile quoted boundary pair.
type Quoted_Span_Unvalidated struct {
	// Start may not name an opening quote.
	Start Quoted_Start_Unvalidated
	// End may not name a distinct closing quote.
	End Quoted_End_Unvalidated
}

// Quoted_Span_Unvalidated_Invariants composes hostile quoted boundaries.
func Quoted_Span_Unvalidated_Invariants(
	value Quoted_Span_Unvalidated, namespace aver.Namespace,
) {
	Quoted_Start_Unvalidated_Invariants(value.Start, namespace)
	Quoted_End_Unvalidated_Invariants(value.End, namespace)
}

// Quoted_Start is quoted token opening boundary.
type Quoted_Start uint16

// Quoted_Start_Invariants reserves opening and closing quote bytes.
func Quoted_Start_Invariants(value Quoted_Start, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(QUOTED_START_MINIMUM), uint16(QUOTED_START_MAXIMUM),
		).
		Ensure()
}

// Quoted_End is quoted token terminal boundary.
type Quoted_End uint16

// Quoted_End_Invariants excludes unterminated quoted token.
func Quoted_End_Invariants(value Quoted_End, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(QUOTED_END_MINIMUM), uint16(QUOTED_END_MAXIMUM),
		).
		Ensure()
}

// Quoted_Span retains one validated quoted token boundary pair.
type Quoted_Span struct {
	// Start opens quoted token.
	Start Quoted_Start
	// End closes quoted token.
	End Quoted_End
}

// Quoted_Span_Invariants composes quoted boundaries once per validation chain.
func Quoted_Span_Invariants(value Quoted_Span, namespace aver.Namespace) {
	Quoted_Start_Invariants(value.Start, namespace)
	Quoted_End_Invariants(value.End, namespace)
}

// Quoted_Span_Stored carries one quoted interval after source validation.
type Quoted_Span_Stored Quoted_Span

// Quoted_Span_Stored_Invariants preserves validated quoted boundaries.
func Quoted_Span_Stored_Invariants(value Quoted_Span_Stored, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value.Start), uint16(QUOTED_START_MINIMUM),
			uint16(QUOTED_START_MAXIMUM),
		).
		Range_Uint16(
			uint16(value.End), uint16(QUOTED_END_MINIMUM), uint16(QUOTED_END_MAXIMUM),
		).
		Ensure()
}

// QUOTED_SPAN_VALIDATED_FIELD selects one source-checked quoted span.
const QUOTED_SPAN_VALIDATED_FIELD = bytes.SLICE_SIZE_MINIMUM

// QUOTED_SPAN_VALIDATED_FIELD_COUNT fixes trusted quoted transport shape.
const QUOTED_SPAN_VALIDATED_FIELD_COUNT = QUOTED_SPAN_VALIDATED_FIELD +
	STORAGE_FIELD_COUNT_INCREMENT

// Quoted_Span_Validated carries one span accepted by quoted_span_validate.
type Quoted_Span_Validated Quoted_Span_Validated_Fields

// Quoted_Span_Validated_Invariants checks transport ownership after source validation.
func Quoted_Span_Validated_Invariants(
	value Quoted_Span_Validated, namespace aver.Namespace,
) {
	Quoted_Span_Proof_Stored_Invariants(Quoted_Span_Proof_Stored(value.Value), namespace)
}

// QUOTED_POSITION_VALIDATED_FIELD selects one cursor bounded by quoted span.
const QUOTED_POSITION_VALIDATED_FIELD = bytes.SLICE_SIZE_MINIMUM

// QUOTED_POSITION_VALIDATED_FIELD_COUNT fixes quoted cursor transport shape.
const QUOTED_POSITION_VALIDATED_FIELD_COUNT = QUOTED_POSITION_VALIDATED_FIELD +
	STORAGE_FIELD_COUNT_INCREMENT

// Quoted_Position_Validated_Value retains quoted-position proof storage.
type Quoted_Position_Validated_Value uint16

// Quoted_Position_Validated_Value_Invariants preserves the quoted position.
func Quoted_Position_Validated_Value_Invariants(
	value Quoted_Position_Validated_Value, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(SOURCE_SIZE_MINIMUM), uint16(SOURCE_SIZE_MAXIMUM),
		).
		Ensure()
}

// Quoted_Position_Validated carries one cursor proven inside quoted token.
type Quoted_Position_Validated struct {
	// Value keeps span proof attached to the cursor.
	Value Quoted_Position_Stored
}

// Quoted_Position_Validated_Invariants keeps quoted-span proof attached to cursor.
func Quoted_Position_Validated_Invariants(
	value Quoted_Position_Validated, namespace aver.Namespace,
) {
	Quoted_Position_Stored_Invariants(value.Value, namespace)
}

// Action_Start is complete action opening position.
type Action_Start uint16

// Action_Start_Invariants reserves enough bytes for both boundaries.
func Action_Start_Invariants(value Action_Start, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(ACTION_START_MINIMUM), uint16(ACTION_START_MAXIMUM),
		).
		Ensure()
}

// Action_Content_Start is first byte after opening delimiter.
type Action_Content_Start uint16

// Action_Content_Start_Invariants reserves opening and closing boundaries.
func Action_Content_Start_Invariants(
	value Action_Content_Start, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(ACTION_CONTENT_BOUNDARY_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM),
		).
		Ensure()
}

// Action_Content_End is boundary before closing delimiter.
type Action_Content_End uint16

// Action_Content_End_Invariants reserves opening and closing boundaries.
func Action_Content_End_Invariants(value Action_Content_End, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(ACTION_CONTENT_BOUNDARY_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM),
		).
		Ensure()
}

// Action_End is complete action terminal boundary.
type Action_End uint16

// Action_End_Invariants charges both boundaries and permits source end.
func Action_End_Invariants(value Action_End, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(ACTION_END_MINIMUM), uint16(ACTION_END_MAXIMUM),
		).
		Ensure()
}

// Action_Span retains one validated complete action boundary set.
type Action_Span struct {
	// Start is opening delimiter position.
	Start Action_Start
	// Content_Start is first action-content byte.
	Content_Start Action_Content_Start
	// Content_End is action-content terminal boundary.
	Content_End Action_Content_End
	// End is closing delimiter terminal boundary.
	End Action_End
}

// Action_Span_Invariants composes boundaries once before grammar dispatch.
func Action_Span_Invariants(value Action_Span, namespace aver.Namespace) {
	Action_Start_Invariants(value.Start, namespace)
	Action_Content_Start_Invariants(value.Content_Start, namespace)
	Action_Content_End_Invariants(value.Content_End, namespace)
	Action_End_Invariants(value.End, namespace)
}

// Action_Span_Stored carries one complete action after delimiter validation.
type Action_Span_Stored Action_Span

// Action_Span_Stored_Invariants preserves validated action boundaries.
func Action_Span_Stored_Invariants(value Action_Span_Stored, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value.Start), uint16(ACTION_START_MINIMUM),
			uint16(ACTION_START_MAXIMUM),
		).
		Range_Uint16(
			uint16(value.Content_Start), uint16(ACTION_CONTENT_BOUNDARY_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM),
		).
		Range_Uint16(
			uint16(value.Content_End), uint16(ACTION_CONTENT_BOUNDARY_MINIMUM),
			uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM),
		).
		Range_Uint16(
			uint16(value.End), uint16(ACTION_END_MINIMUM), uint16(ACTION_END_MAXIMUM),
		).
		Ensure()
}

// ACTION_SPAN_VALIDATED_FIELD selects one delimiter-checked action span.
const ACTION_SPAN_VALIDATED_FIELD = bytes.SLICE_SIZE_MINIMUM

// ACTION_SPAN_VALIDATED_FIELD_COUNT fixes trusted action transport shape.
const ACTION_SPAN_VALIDATED_FIELD_COUNT = ACTION_SPAN_VALIDATED_FIELD +
	STORAGE_FIELD_COUNT_INCREMENT

// Action_Span_Validated carries one span checked by parse_delimited_action.
type Action_Span_Validated Action_Span_Validated_Fields

// Action_Span_Validated_Invariants checks transport ownership after delimiter validation.
func Action_Span_Validated_Invariants(
	value Action_Span_Validated, namespace aver.Namespace,
) {
	Action_Span_Proof_Stored_Invariants(Action_Span_Proof_Stored(value.Value), namespace)
}

// Node_Reference is a one-based node slot or NO_NODE.
type Node_Reference uint16

// Node_Reference_Invariants includes absent and final node slot.
func Node_Reference_Invariants(value Node_Reference, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(NODE_REFERENCE_MINIMUM),
			uint16(NODE_REFERENCE_MAXIMUM),
		).
		Ensure()
}

// Root_Reference is fixed first syntax arena slot.
type Root_Reference uint16

// Root_Reference_Invariants keeps successful document root canonical.
func Root_Reference_Invariants(value Root_Reference, _ aver.Namespace) {
	aver.Always(
		uint16(value) == uint16(ROOT_NODE_COUNT),
		"Successful document root occupies first syntax arena slot.",
	)
}

// Node_Count is caller workspace slots populated by parsing.
type Node_Count uint16

// Node_Count_Invariants bounds every populated node slot.
func Node_Count_Invariants(value Node_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(NODE_COUNT_MINIMUM), uint16(NODE_COUNT_MAXIMUM),
		).
		Ensure()
}

// Active_Node_Count is one successfully rooted program arena length.
type Active_Node_Count uint16

// Active_Node_Count_Invariants excludes the rejected empty program state.
func Active_Node_Count_Invariants(
	value Active_Node_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(ROOT_NODE_COUNT), uint16(NODE_COUNT_MAXIMUM),
		).
		Ensure()
}

// NODE_COUNT_ALLOCATED_FIELD selects populated caller arena length.
const NODE_COUNT_ALLOCATED_FIELD = bytes.SLICE_SIZE_MINIMUM

// NODE_COUNT_ALLOCATED_FIELD_COUNT fixes arena-bound count transport shape.
const NODE_COUNT_ALLOCATED_FIELD_COUNT = NODE_COUNT_ALLOCATED_FIELD +
	STORAGE_FIELD_COUNT_INCREMENT

// Node_Count_Allocated_Value retains arena-count proof storage.
type Node_Count_Allocated_Value uint16

// Node_Count_Allocated_Value_Invariants preserves the allocated count.
func Node_Count_Allocated_Value_Invariants(
	value Node_Count_Allocated_Value, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(NODE_COUNT_MINIMUM), uint16(NODE_COUNT_MAXIMUM),
		).
		Ensure()
}

// Node_Count_Stored separates arena-count proof shape from transient count values.
type Node_Count_Stored interface{}

// Node_Count_Stored_Invariants accepts absent output or one typed arena count.
func Node_Count_Stored_Invariants(value Node_Count_Stored, _ aver.Namespace) {
	_, valid := value.(Node_Count_Allocated_Value)
	aver.Always(valid == (value != nil), "Node count has expected storage type.")
}

// Node_Count_Allocated carries count checked against caller arena limit.
type Node_Count_Allocated struct {
	// Value keeps arena proof attached to the count.
	Value Node_Count_Stored
}

// Node_Count_Allocated_Invariants keeps count proof attached to parser flow.
func Node_Count_Allocated_Invariants(
	value Node_Count_Allocated, namespace aver.Namespace,
) {
	Node_Count_Stored_Invariants(value.Value, namespace)
}

// Node_Limit is zero for full arena or one explicit caller capacity.
type Node_Limit uint16

// Node_Limit_Invariants bounds caller-selected syntax capacity.
func Node_Limit_Invariants(value Node_Limit, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(NODE_LIMIT_DEFAULT), uint16(NODE_COUNT_MAXIMUM),
		).
		Ensure()
}

// Node_Start is one syntax-node opening source boundary.
type Node_Start uint16

// Node_Start_Invariants keeps node openings inside bounded source.
func Node_Start_Invariants(value Node_Start, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(SOURCE_SIZE_MINIMUM), uint16(SOURCE_SIZE_MAXIMUM),
		).
		Ensure()
}

// Node_End is one syntax-node closing source boundary.
type Node_End uint16

// Node_End_Invariants keeps node closings inside bounded source.
func Node_End_Invariants(value Node_End, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(SOURCE_SIZE_MINIMUM), uint16(SOURCE_SIZE_MAXIMUM),
		).
		Ensure()
}

// Node_Value_Start is one borrowed node-value opening boundary.
type Node_Value_Start uint16

// Node_Value_Start_Invariants keeps node-value openings inside bounded source.
func Node_Value_Start_Invariants(
	value Node_Value_Start, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(SOURCE_SIZE_MINIMUM),
			uint16(NODE_VALUE_START_MAXIMUM),
		).
		Ensure()
}

// Node_Value_End is one borrowed node-value closing boundary.
type Node_Value_End uint16

// Node_Value_End_Invariants keeps node-value closings inside bounded source.
func Node_Value_End_Invariants(
	value Node_Value_End, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(SOURCE_SIZE_MINIMUM), uint16(SOURCE_SIZE_MAXIMUM),
		).
		Ensure()
}

// Node_First_Child identifies first ordered child.
type Node_First_Child uint16

// Node_First_Child_Invariants keeps first child inside syntax arena.
func Node_First_Child_Invariants(
	value Node_First_Child, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(NODE_REFERENCE_MINIMUM),
			uint16(NODE_REFERENCE_MAXIMUM),
		).
		Ensure()
}

// Node_Next_Sibling identifies next ordered sibling.
type Node_Next_Sibling uint16

// Node_Next_Sibling_Invariants keeps next sibling inside syntax arena.
func Node_Next_Sibling_Invariants(
	value Node_Next_Sibling, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(NODE_REFERENCE_MINIMUM),
			uint16(NODE_REFERENCE_MAXIMUM),
		).
		Ensure()
}

// Node stores one flat syntax-tree record using source spans and references.
type Node struct {
	// Kind selects interpretation of every remaining field.
	Kind Node_Kind
	// Start is the node's opening source boundary.
	Start Node_Start
	// End is the node's closing source boundary.
	End Node_End
	// Value_Start begins borrowed lexical payload.
	Value_Start Node_Value_Start
	// Value_End closes borrowed lexical payload.
	Value_End Node_Value_End
	// First_Child begins the node's ordered child list.
	First_Child Node_First_Child
	// Next_Sibling links the containing ordered list.
	Next_Sibling Node_Next_Sibling
}

// Node_Invariants bounds every source span and arena reference.
func Node_Invariants(value Node, namespace aver.Namespace) {
	Node_Kind_Invariants(value.Kind, namespace)
	Node_Start_Invariants(value.Start, namespace)
	Node_End_Invariants(value.End, namespace)
	Node_Value_Start_Invariants(value.Value_Start, namespace)
	Node_Value_End_Invariants(value.Value_End, namespace)
	Node_First_Child_Invariants(value.First_Child, namespace)
	Node_Next_Sibling_Invariants(value.Next_Sibling, namespace)
}

// Node_Stored keeps one checked node opaque between parser and evaluator boundaries.
type Node_Stored interface{}

// Node_Stored_Invariants accepts absent optional storage or one caller-owned node.
func Node_Stored_Invariants(value Node_Stored, _ aver.Namespace) {
	_, valid := value.(*Node)
	aver.Always(valid == (value != nil), "Stored node has expected type.")
}

// NODE_ALLOCATION_FIELD selects one parser-built syntax record.
const NODE_ALLOCATION_FIELD = bytes.SLICE_SIZE_MINIMUM

// NODE_ALLOCATION_FIELD_COUNT fixes parser-built node transport shape.
const NODE_ALLOCATION_FIELD_COUNT = NODE_ALLOCATION_FIELD + STORAGE_FIELD_COUNT_INCREMENT

// Node_Allocation carries one node whose fields derive from validated parser state.
type Node_Allocation struct {
	// Value cannot bypass arena validation.
	Value Node_Allocation_Proof_Stored
}

// Node_Allocation_Invariants keeps parser construction proof attached to node.
func Node_Allocation_Invariants(value Node_Allocation, namespace aver.Namespace) {
	Node_Allocation_Proof_Stored_Invariants(value.Value, namespace)
}

// Nodes is fixed caller-owned syntax arena.
type Nodes []Node

// Nodes_Invariants fixes arena capacity while parser owns populated content.
func Nodes_Invariants(value Nodes, _ aver.Namespace) {
	aver.Always(
		len(value) == NODE_COUNT_MAXIMUM,
		"Syntax node arena has fixed capacity.",
	)
	aver.Always(
		cap(value) == NODE_COUNT_MAXIMUM,
		"Syntax node arena cannot grow.",
	)
}

// Number_Workspace owns exact bounded numeric-validation scratch.
type Number_Workspace big.Float_Parse_Workspace

// Number_Workspace_Invariants preserves shared numeric workspace structure.
func Number_Workspace_Invariants(value Number_Workspace, namespace aver.Namespace) {
	Number_Workspace_Source_Count_Invariants(
		Number_Workspace_Source_Count(len(value.Source)), namespace,
	)
	Number_Workspace_Control_Count_Invariants(
		Number_Workspace_Control_Count(len(value.Control)), namespace,
	)
}

// NUMBER_WORKSPACE_SOURCE_COUNT_COMPLETE is required source scratch size.
const NUMBER_WORKSPACE_SOURCE_COUNT_COMPLETE = Number_Workspace_Source_Count(
	big.FLOAT_PARSE_TEXT_SIZE_MAXIMUM,
)

// Number_Workspace_Source_Count keeps parser source proof independent.
type Number_Workspace_Source_Count int

// Number_Workspace_Source_Count_Invariants requires complete numeric scratch.
func Number_Workspace_Source_Count_Invariants(
	value Number_Workspace_Source_Count, _ aver.Namespace,
) {
	aver.Always(
		int(value) == int(NUMBER_WORKSPACE_SOURCE_COUNT_COMPLETE),
		"Template number source scratch is complete.",
	)
}

// NUMBER_WORKSPACE_CONTROL_COUNT_COMPLETE is required control scratch size.
const NUMBER_WORKSPACE_CONTROL_COUNT_COMPLETE = Number_Workspace_Control_Count(
	big.FLOAT_PARSE_CONTROL_COUNT,
)

// Number_Workspace_Control_Count keeps parser control proof independent.
type Number_Workspace_Control_Count int

// Number_Workspace_Control_Count_Invariants requires complete control scratch.
func Number_Workspace_Control_Count_Invariants(
	value Number_Workspace_Control_Count, _ aver.Namespace,
) {
	aver.Always(
		int(value) == int(NUMBER_WORKSPACE_CONTROL_COUNT_COMPLETE),
		"Template number control scratch is complete.",
	)
}

// Number_Workspace_Pointer keeps internal lexer scratch presence typed.
type Number_Workspace_Pointer *Number_Workspace

// Number_Workspace_Pointer_Invariants requires parser-owned numeric scratch.
func Number_Workspace_Pointer_Invariants(
	value Number_Workspace_Pointer, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Number_Workspace_Invariants(*value, namespace)
}

// Template_Name_Left owns first decoded name in equality work.
type Template_Name_Left []byte

// Template_Name_Left_Invariants fixes first name scratch shape.
func Template_Name_Left_Invariants(value Template_Name_Left, _ aver.Namespace) {
	aver.Always(
		len(value) == TEMPLATE_NAME_SIZE_MAXIMUM,
		"First template name scratch spans maximum source.",
	)
	aver.Always(
		cap(value) == TEMPLATE_NAME_SIZE_MAXIMUM,
		"First template name scratch cannot grow.",
	)
}

// Template_Name_Right owns second decoded name in equality work.
type Template_Name_Right []byte

// Template_Name_Right_Invariants fixes second name scratch shape.
func Template_Name_Right_Invariants(value Template_Name_Right, _ aver.Namespace) {
	aver.Always(
		len(value) == TEMPLATE_NAME_SIZE_MAXIMUM,
		"Second template name scratch spans maximum source.",
	)
	aver.Always(
		cap(value) == TEMPLATE_NAME_SIZE_MAXIMUM,
		"Second template name scratch cannot grow.",
	)
}

// Syntax_Diagnostic_Storage keeps stale failure state outside parser-domain coverage.
type Parse_Diagnostic_Stored interface{}

// Parse_Diagnostic_Stored_Invariants accepts absent scratch or one parser diagnostic.
func Parse_Diagnostic_Stored_Invariants(
	value Parse_Diagnostic_Stored, _ aver.Namespace,
) {
	_, valid := value.(Parse_Diagnostic)
	aver.Always(valid == (value != nil), "Stored parser diagnostic has expected type.")
}

// Syntax_Diagnostic_Storage keeps one stale caller diagnostic as scratch.
type Syntax_Diagnostic_Storage struct {
	// Value keeps mutation inside parser workspace.
	Value Parse_Diagnostic_Stored
}

// Syntax_Diagnostic_Storage_Invariants checks ownership without treating stale scratch as output.
func Syntax_Diagnostic_Storage_Invariants(
	value Syntax_Diagnostic_Storage, namespace aver.Namespace,
) {
	Parse_Diagnostic_Stored_Invariants(value.Value, namespace)
}

// Node_Limit_Storage_Value retains hostile node-limit storage.
type Node_Limit_Storage_Value uint16

// Node_Limit_Storage_Value_Invariants preserves node-limit encoding.
func Node_Limit_Storage_Value_Invariants(
	value Node_Limit_Storage_Value, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(NODE_LIMIT_DEFAULT),
			uint16(NODE_LIMIT_UNVALIDATED_MAXIMUM),
		).
		Ensure()
}

// Node_Limit_Stored keeps hostile policy representation separate from its domain.
type Node_Limit_Stored interface{}

// Node_Limit_Stored_Invariants accepts absent default policy or one typed limit.
func Node_Limit_Stored_Invariants(value Node_Limit_Stored, _ aver.Namespace) {
	_, valid := value.(Node_Limit_Storage_Value)
	aver.Always(valid == (value != nil), "Node limit has expected storage type.")
}

// Node_Limit_Storage keeps one caller policy value inside aggregate storage.
type Node_Limit_Storage struct {
	// Value keeps hostile policy inside parser workspace.
	Value Node_Limit_Stored
}

// Node_Limit_Storage_Invariants leaves hostile policy validation to Parse_Into.
func Node_Limit_Storage_Invariants(value Node_Limit_Storage, namespace aver.Namespace) {
	Node_Limit_Stored_Invariants(value.Value, namespace)
}

// Workspace is caller-owned syntax storage.
type Syntax_Workspace struct {
	// Nodes contains every node referenced by returned Document.
	Nodes Nodes
	// Number owns numeric validation state reused across every token.
	Number Number_Workspace
	// Template_Name_Left owns first decoded template name.
	Template_Name_Left Template_Name_Left
	// Template_Name_Right owns second decoded template name.
	Template_Name_Right Template_Name_Right
	// Diagnostic retains first parser refusal while helpers unwind.
	Parse_Diagnostic Syntax_Diagnostic_Storage
	// Node_Limit selects full arena or a smaller caller work budget.
	Node_Limit Node_Limit_Storage
}

// Workspace_Invariants verifies storage shape without reading stale nodes.
func Syntax_Workspace_Invariants(value Syntax_Workspace, namespace aver.Namespace) {
	Nodes_Invariants(value.Nodes, namespace)
	Number_Workspace_Invariants(value.Number, namespace)
	Template_Name_Left_Invariants(value.Template_Name_Left, namespace)
	Template_Name_Right_Invariants(value.Template_Name_Right, namespace)
	Syntax_Diagnostic_Storage_Invariants(value.Parse_Diagnostic, namespace)
	Node_Limit_Storage_Invariants(value.Node_Limit, namespace)
}

// Syntax_Workspace_Pointer keeps absence typed at the hostile boundary.
type Syntax_Workspace_Pointer *Syntax_Workspace

// Syntax_Workspace_Pointer_Invariants composes present caller storage.
func Syntax_Workspace_Pointer_Invariants(
	value Syntax_Workspace_Pointer, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Syntax_Workspace_Invariants(*value, namespace)
}

// Workspace_Storage retains an absent pointer without changing aggregate shape.
type Syntax_Workspace_Storage struct {
	// Value keeps absence explicit inside hostile input.
	Value Syntax_Workspace_Unvalidated_Pointer
}

// Workspace_Storage_Invariants leaves pointer presence to Parse_Into status.
func Syntax_Workspace_Storage_Invariants(
	value Syntax_Workspace_Storage, namespace aver.Namespace,
) {
	Syntax_Workspace_Unvalidated_Pointer_Invariants(value.Value, namespace)
}

// Syntax_Workspace_Unvalidated_Value retains hostile caller storage.
type Syntax_Workspace_Unvalidated_Value any

// Syntax_Workspace_Unvalidated_Value_Invariants defers shape trust.
func Syntax_Workspace_Unvalidated_Value_Invariants(
	value Syntax_Workspace_Unvalidated_Value, _ aver.Namespace,
) {
	present := value != nil
	aver.Always(present == present, "Syntax workspace input retains caller storage.")
}

// Syntax_Workspace_Unvalidated_Pointer admits absent or malformed caller storage.
type Syntax_Workspace_Unvalidated_Pointer struct {
	// Value delays pointer-shape trust until syntax_workspace_valid.
	Value Syntax_Workspace_Unvalidated_Value
}

// Syntax_Workspace_Unvalidated_Pointer_Invariants defers shape checks to parsing.
func Syntax_Workspace_Unvalidated_Pointer_Invariants(
	value Syntax_Workspace_Unvalidated_Pointer, namespace aver.Namespace,
) {
	Syntax_Workspace_Unvalidated_Value_Invariants(value.Value, namespace)
}

// Workspace_Input retains hostile absent storage until parsing returns status.
type Syntax_Workspace_Input struct {
	// State points at caller-owned syntax storage when present.
	State Syntax_Workspace_Storage
}

// Workspace_Input_Invariants fixes unvalidated pointer storage shape.
func Syntax_Workspace_Input_Invariants(
	value Syntax_Workspace_Input, namespace aver.Namespace,
) {
	Syntax_Workspace_Storage_Invariants(value.State, namespace)
}

// Syntax_Workspace_Validity reports complete caller-owned parser storage.
type Syntax_Workspace_Validity bool

// Syntax_Workspace_Validity_Invariants covers valid and malformed storage.
func Syntax_Workspace_Validity_Invariants(
	value Syntax_Workspace_Validity, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Caller supplied complete parser storage.").
		Ensure()
}

// Document_Root identifies root output list.
type Document_Root uint16

// Document_Root_Invariants lists absent failure root and fixed first success slot.
func Document_Root_Invariants(value Document_Root, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint16(
			uint16(value), uint16(NO_NODE), uint16(ROOT_NODE_COUNT),
		).
		Ensure()
}

// Document_First_Template is zero without definitions, else the root-relative first node.
type Document_First_Template uint16

// Document_First_Template_Invariants keeps definition chain inside syntax arena.
func Document_First_Template_Invariants(
	value Document_First_Template, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(DOCUMENT_FIRST_TEMPLATE_MINIMUM),
			uint16(DOCUMENT_FIRST_TEMPLATE_MAXIMUM),
		).
		Ensure()
}

// Document is the bounded root and populated node count.
type Document struct {
	// Root names the root list node.
	Root Document_Root
	// Node_Count bounds every reference reachable from Root.
	Node_Count Node_Count
	// First_Template begins named definition sibling chain.
	First_Template Document_First_Template
	// Template_Count bounds named definition chain length.
	Template_Count Template_Count
}

// Document_Invariants bounds returned arena metadata.
func Document_Invariants(value Document, namespace aver.Namespace) {
	Document_Root_Invariants(value.Root, namespace)
	Node_Count_Invariants(value.Node_Count, namespace)
	Document_First_Template_Invariants(value.First_Template, namespace)
	Template_Count_Invariants(value.Template_Count, namespace)
	aver.Always(
		value.Template_Count <= Template_Count(value.Node_Count),
		"Template chain cannot exceed populated node storage.",
	)
}

// Diagnostic_Code identifies one bounded parser outcome class.
type Diagnostic_Code uint8

// Diagnostic_Code_Invariants covers success and every parser refusal.
func Diagnostic_Code_Invariants(value Diagnostic_Code, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(
			uint8(value), uint8(DIAGNOSTIC_NONE),
			uint8(DIAGNOSTIC_CAPACITY_EXCEEDED),
		).
		Ensure()
}

// Diagnostic_Position is the first malformed token or refusal boundary.
type Parse_Diagnostic_Position uint16

// Diagnostic_Position_Invariants keeps refusal coordinates inside bounded source.
func Parse_Diagnostic_Position_Invariants(
	value Parse_Diagnostic_Position, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(SOURCE_SIZE_MINIMUM), uint16(SOURCE_SIZE_MAXIMUM),
		).
		Ensure()
}

// PARSE_ERROR_POSITION_MINIMUM is the first post-opening source byte.
const PARSE_ERROR_POSITION_MINIMUM = SOURCE_SIZE_MINIMUM + SOURCE_POSITION_INCREMENT

// Parse_Error_Position begins after parsing has consumed an opening byte.
type Parse_Error_Position uint16

// Parse_Error_Position_Invariants excludes pre-parse validation coordinates.
func Parse_Error_Position_Invariants(
	value Parse_Error_Position, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(PARSE_ERROR_POSITION_MINIMUM),
			uint16(SOURCE_SIZE_MAXIMUM),
		).
		Ensure()
}

// Diagnostic carries one scalar code and source coordinate without text allocation.
type Parse_Diagnostic struct {
	// Code identifies the refusal class.
	Code Diagnostic_Code
	// Position identifies the source boundary where refusal became certain.
	Position Parse_Diagnostic_Position
}

// Diagnostic_Invariants bounds code and source coordinate independently.
func Parse_Diagnostic_Invariants(value Parse_Diagnostic, namespace aver.Namespace) {
	Diagnostic_Code_Invariants(value.Code, namespace)
	Parse_Diagnostic_Position_Invariants(value.Position, namespace)
}

// Parse_Failure_Diagnostic is one non-success parser diagnostic.
type Parse_Failure_Diagnostic Parse_Diagnostic

// Parse_Failure_Diagnostic_Invariants excludes the success code after refusal.
func Parse_Failure_Diagnostic_Invariants(
	value Parse_Failure_Diagnostic, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint8(
			uint8(value.Code), uint8(DIAGNOSTIC_INPUT_INVALID),
			uint8(DIAGNOSTIC_CAPACITY_EXCEEDED),
		).
		Range_Uint16(
			uint16(value.Position), uint16(SOURCE_SIZE_MINIMUM),
			uint16(SOURCE_SIZE_MAXIMUM),
		).
		Ensure()
}

// Parse_Status reports every parser outcome.
type Parse_Status uint8

// Parse_Status_Invariants covers the contiguous parser outcome range.
func Parse_Status_Invariants(value Parse_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(
			uint8(value), uint8(PARSE_STATUS_OK), uint8(PARSE_STATUS_CAPACITY_EXCEEDED),
		).
		Ensure()
}

// Build_Status is success, malformed grammar, or exhausted syntax arena.
type Build_Status uint8

// Build_Status_Invariants excludes validation failures handled before tree construction.
func Build_Status_Invariants(value Build_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(PARSE_STATUS_OK), uint8(PARSE_STATUS_SYNTAX_INVALID),
			uint8(PARSE_STATUS_CAPACITY_EXCEEDED),
		).
		Ensure()
}

// PARSE_FAILURE_STATUS_FIELD selects one parser status proven non-success.
const PARSE_FAILURE_STATUS_FIELD = bytes.SLICE_SIZE_MINIMUM

// PARSE_FAILURE_STATUS_FIELD_COUNT fixes failure transport shape.
const PARSE_FAILURE_STATUS_FIELD_COUNT = PARSE_FAILURE_STATUS_FIELD +
	STORAGE_FIELD_COUNT_INCREMENT

// Parse_Failure_Status_Validated_Value retains refused-status proof storage.
type Parse_Failure_Status_Validated_Value uint8

// Parse_Failure_Status_Validated_Value_Invariants preserves the refused status.
func Parse_Failure_Status_Validated_Value_Invariants(
	value Parse_Failure_Status_Validated_Value, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint8(
			uint8(value), uint8(PARSE_STATUS_INPUT_INVALID),
			uint8(PARSE_STATUS_CAPACITY_EXCEEDED),
		).
		Ensure()
}

// Parse_Failure_Status_Validated carries one refused parser result.
type Parse_Failure_Status_Validated struct {
	// Value keeps failure proof attached to parser status.
	Value Parse_Failure_Status_Proof_Stored
}

// Parse_Failure_Status_Validated_Invariants checks refusal ownership.
func Parse_Failure_Status_Validated_Invariants(
	value Parse_Failure_Status_Validated, namespace aver.Namespace,
) {
	Parse_Failure_Status_Proof_Stored_Invariants(value.Value, namespace)
}

// Lex_Status reports accepted or malformed action text.
type Lex_Status uint8

// Lex_Status_Invariants excludes syntax-tree capacity outcomes.
func Lex_Status_Invariants(value Lex_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(PARSE_STATUS_OK), uint8(PARSE_STATUS_SYNTAX_INVALID),
		).
		Ensure()
}

// Capacity_Status reports success or exhausted caller storage.
type Capacity_Status uint8

// Capacity_Status_Invariants excludes malformed grammar from storage work.
func Capacity_Status_Invariants(value Capacity_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(PARSE_STATUS_OK),
			uint8(PARSE_STATUS_CAPACITY_EXCEEDED),
		).
		Ensure()
}

// DEFAULT_DELIMITER_SIZE is the standard two-byte action marker width.
const DEFAULT_DELIMITER_SIZE = len("{{")

// DEFAULT_LEFT_DELIMITER_FIRST is the first standard action-opening byte.
const DEFAULT_LEFT_DELIMITER_FIRST byte = '{'

// DEFAULT_LEFT_DELIMITER_SECOND is the second standard action-opening byte.
const DEFAULT_LEFT_DELIMITER_SECOND byte = '{'

// DEFAULT_RIGHT_DELIMITER_FIRST is the first standard action-closing byte.
const DEFAULT_RIGHT_DELIMITER_FIRST byte = '}'

// DEFAULT_RIGHT_DELIMITER_SECOND is the second standard action-closing byte.
const DEFAULT_RIGHT_DELIMITER_SECOND byte = '}'

// TRIM_MARKER removes adjacent template whitespace only with a space witness.
const TRIM_MARKER byte = '-'

// Source_Validate changes bounded hostile bytes into parser Source.
func Source_Validate(
	unvalidated Source_Unvalidated,
) (_ Source, status Source_Status) {
	defer func() { Source_Status_Invariants(status, "Source_Validate.status") }()
	Source_Unvalidated_Invariants(unvalidated, "Source_Validate.unvalidated")
	var source Source
	defer func() { Source_Invariants(source, "Source_Validate.source") }()
	if len(unvalidated) > SOURCE_SIZE_MAXIMUM {
		return Source{}, PARSE_STATUS_INPUT_INVALID
	}
	source = Source{
		Data: Source_Storage{Value: Source_Storage_Value(unvalidated)},
	}
	return source, PARSE_STATUS_OK
}

// New_Configuration validates borrowed delimiters and parser mode.
func New_Configuration(
	input Configuration_Input,
) (_ Configuration, status Configuration_Status) {
	defer func() {
		Configuration_Status_Invariants(status, "New_Configuration.status")
	}()
	Configuration_Input_Invariants(input, "New_Configuration.input")
	var configuration Configuration
	defer func() {
		Configuration_Invariants(configuration, "New_Configuration.configuration")
	}()
	if len(input.Left) > DELIMITER_SIZE_MAXIMUM {
		return Configuration{}, PARSE_STATUS_CONFIGURATION_INVALID
	}
	if len(input.Right) > DELIMITER_SIZE_MAXIMUM {
		return Configuration{}, PARSE_STATUS_CONFIGURATION_INVALID
	}
	if (len(input.Left) == DELIMITER_SIZE_MINIMUM) !=
		(len(input.Right) == DELIMITER_SIZE_MINIMUM) {
		return Configuration{}, PARSE_STATUS_CONFIGURATION_INVALID
	}
	if input.Mode&^MODE_MAXIMUM != MODE_MINIMUM {
		return Configuration{}, PARSE_STATUS_CONFIGURATION_INVALID
	}
	configuration.Left.Value = Opening_Delimiter_Storage_Value(input.Left)
	configuration.Right.Value = Right_Delimiter_Storage_Value(input.Right)
	configuration.Mode.Value = Mode_Storage_Value(input.Mode)
	configuration.Function.Value = Parse_Function_Storage_Value(input.Function)
	if !bool(Configuration_Valid(configuration)) {
		return Configuration{}, PARSE_STATUS_CONFIGURATION_INVALID
	}
	return configuration, PARSE_STATUS_OK
}

// Configuration_Valid rechecks mutable borrowed policy before parsing.
func Configuration_Valid(configuration Configuration) (valid Configuration_Validity) {
	defer func() {
		Configuration_Validity_Invariants(valid, "Configuration_Valid.valid")
	}()
	Configuration_Invariants(configuration, "Configuration_Valid.configuration")
	left := configuration.Left.Value
	right := configuration.Right.Value
	if len(left) > DELIMITER_SIZE_MAXIMUM {
		return false
	}
	if len(right) > DELIMITER_SIZE_MAXIMUM {
		return false
	}
	if (len(left) == DELIMITER_SIZE_MINIMUM) != (len(right) == DELIMITER_SIZE_MINIMUM) {
		return false
	}
	if Mode(configuration.Mode.Value)&^MODE_MAXIMUM != MODE_MINIMUM {
		return false
	}
	return true
}

// Quoted_Unquote_Into decodes one validated template literal into caller storage.
func Quoted_Unquote_Into(
	destination Quoted_Output, source Source, unvalidated Quoted_Span_Unvalidated,
) (_ Quoted_Count, status Lex_Status) {
	defer func() { Lex_Status_Invariants(status, "Quoted_Unquote_Into.status") }()
	Quoted_Output_Invariants(destination, "Quoted_Unquote_Into.destination")
	Source_Invariants(source, "Quoted_Unquote_Into.source")
	Quoted_Span_Unvalidated_Invariants(
		unvalidated, "Quoted_Unquote_Into.unvalidated",
	)
	count := Quoted_Count(SOURCE_SIZE_MINIMUM)
	defer func() { Quoted_Count_Invariants(count, "Quoted_Unquote_Into.count") }()
	start := Source_Position(unvalidated.Start)
	end := Source_Position(unvalidated.End)
	data := source.Data.Value
	if end > Source_Position(len(data)) {
		return Quoted_Count(SOURCE_SIZE_MINIMUM), PARSE_STATUS_SYNTAX_INVALID
	}
	if start >= end {
		return Quoted_Count(SOURCE_SIZE_MINIMUM), PARSE_STATUS_SYNTAX_INVALID
	}
	if end-start < Source_Position(QUOTED_BOUNDARY_SIZE) {
		return Quoted_Count(SOURCE_SIZE_MINIMUM), PARSE_STATUS_SYNTAX_INVALID
	}
	quote := Quote(data[start])
	if quote != QUOTE_CHARACTER {
		if quote != QUOTE_STRING {
			if quote != QUOTE_RAW_STRING {
				return Quoted_Count(SOURCE_SIZE_MINIMUM),
					PARSE_STATUS_SYNTAX_INVALID
			}
		}
	}
	if Quote(data[end-SOURCE_POSITION_INCREMENT]) != quote {
		return Quoted_Count(SOURCE_SIZE_MINIMUM), PARSE_STATUS_SYNTAX_INVALID
	}
	span := Quoted_Span{
		Start: Quoted_Start(start), End: Quoted_End(end),
	}
	Quoted_Span_Invariants(span, "Quoted_Unquote_Into.span")
	span_validated := Quoted_Span_Validated{Value: Quoted_Span_Stored(span)}
	valid := quoted_content_valid(source, span_validated, quote)
	if !bool(valid) {
		return Quoted_Count(SOURCE_SIZE_MINIMUM), PARSE_STATUS_SYNTAX_INVALID
	}
	decoded := quoted_unquote_into(
		source, span_validated, Decoded_Output(destination),
	)
	count = Quoted_Count(decoded)
	return count, PARSE_STATUS_OK
}

func syntax_workspace_valid(
	input Syntax_Workspace_Input,
) (valid Syntax_Workspace_Validity) {
	defer func() {
		Syntax_Workspace_Validity_Invariants(valid, "syntax_workspace_valid.valid")
	}()
	Syntax_Workspace_Input_Invariants(input, "syntax_workspace_valid.input")
	workspace, typed := input.State.Value.Value.(*Syntax_Workspace)
	if !typed {
		return false
	}
	if workspace == nil {
		return false
	}
	if len(workspace.Nodes) != NODE_COUNT_MAXIMUM {
		return false
	}
	if len(workspace.Template_Name_Left) != TEMPLATE_NAME_SIZE_MAXIMUM {
		return false
	}
	if len(workspace.Template_Name_Right) != TEMPLATE_NAME_SIZE_MAXIMUM {
		return false
	}
	node_limit, typed_limit := workspace.Node_Limit.Value.(Node_Limit_Storage_Value)
	if workspace.Node_Limit.Value != nil {
		if !typed_limit {
			return false
		}
	}
	if Node_Limit(node_limit) > Node_Limit(NODE_COUNT_MAXIMUM) {
		return false
	}
	number := &workspace.Number
	if len(number.Source) != big.FLOAT_PARSE_TEXT_SIZE_MAXIMUM {
		return false
	}
	if len(number.Control) != big.FLOAT_PARSE_CONTROL_COUNT {
		return false
	}
	if len(number.Parse.Words) != big.WORD_COUNT_MAXIMUM {
		return false
	}
	if len(number.Integers[big.RAT_PARSE_FRACTION_NUMERATOR_INDEX].Words) !=
		big.WORD_COUNT_MAXIMUM {
		return false
	}
	if len(number.Integers[big.RAT_PARSE_FRACTION_DENOMINATOR_INDEX].Words) !=
		big.WORD_COUNT_MAXIMUM {
		return false
	}
	if len(number.Values[big.FLOAT_RAT_NUMERATOR_INDEX].Mantissa.Words) !=
		big.WORD_COUNT_MAXIMUM {
		return false
	}
	if len(number.Values[big.FLOAT_RAT_DENOMINATOR_INDEX].Mantissa.Words) !=
		big.WORD_COUNT_MAXIMUM {
		return false
	}
	if len(number.Values[big.FLOAT_RAT_RESULT_INDEX].Mantissa.Words) !=
		big.WORD_COUNT_MAXIMUM {
		return false
	}
	if len(number.Division.Remainder) != big.FLOAT_DIVISION_WORD_COUNT_MAXIMUM {
		return false
	}
	if len(number.Division.Divisor) != big.FLOAT_DIVISION_WORD_COUNT_MAXIMUM {
		return false
	}
	return true
}

// Parse_Into stores one complete syntax tree in caller workspace.
func Parse_Into(
	source Source, configuration Configuration,
	workspace_input Syntax_Workspace_Input,
) (
	_ Document, _ Parse_Diagnostic, status Parse_Status,
) {
	defer func() { Parse_Status_Invariants(status, "Parse_Into.status") }()
	Source_Invariants(source, "Parse_Into.source")
	Configuration_Invariants(configuration, "Parse_Into.configuration")
	Syntax_Workspace_Input_Invariants(workspace_input, "Parse_Into.workspace")
	var document Document
	defer func() { Document_Invariants(document, "Parse_Into.document") }()
	var diagnostic Parse_Diagnostic
	defer func() {
		Parse_Diagnostic_Invariants(diagnostic, "Parse_Into.diagnostic")
	}()
	failure := func(
		status_value Parse_Failure_Status_Validated_Value,
		position Parse_Diagnostic_Position,
	) (diagnostic_value Parse_Diagnostic) {
		return Parse_Diagnostic(diagnostic_from_status(
			Parse_Failure_Status_Validated{Value: status_value}, position,
		))
	}
	minimum_position := Parse_Diagnostic_Position(SOURCE_SIZE_MINIMUM)
	if len(source.Data.Value) > SOURCE_SIZE_MAXIMUM {
		status = PARSE_STATUS_INPUT_INVALID
		diagnostic = failure(Parse_Failure_Status_Validated_Value(status), minimum_position)
		return document, diagnostic, status
	}
	if !bool(Configuration_Valid(configuration)) {
		status = PARSE_STATUS_CONFIGURATION_INVALID
		diagnostic = failure(Parse_Failure_Status_Validated_Value(status), minimum_position)
		return document, diagnostic, status
	}
	workspace_unvalidated := workspace_input.State.Value
	if workspace_unvalidated.Value == nil {
		status = PARSE_STATUS_WORKSPACE_INVALID
		diagnostic = failure(Parse_Failure_Status_Validated_Value(status), minimum_position)
		return document, diagnostic, status
	}
	if !bool(syntax_workspace_valid(workspace_input)) {
		status = PARSE_STATUS_WORKSPACE_INVALID
		diagnostic = failure(Parse_Failure_Status_Validated_Value(status), minimum_position)
		return document, diagnostic, status
	}
	workspace_value, _ := workspace_unvalidated.Value.(*Syntax_Workspace)
	workspace := Syntax_Workspace_Pointer(workspace_value)
	node_limit, _ := workspace.Node_Limit.Value.(Node_Limit_Storage_Value)
	Node_Limit_Invariants(Node_Limit(node_limit), "Parse_Into.node_limit")
	Syntax_Workspace_Pointer_Invariants(workspace, "Parse_Into.workspace_state")
	workspace.Parse_Diagnostic.Value = Parse_Diagnostic_Stored(Parse_Diagnostic{})
	var build_status Build_Status
	document, build_status = parse_unchecked(source, configuration, workspace)
	status = Parse_Status(build_status)
	if status == PARSE_STATUS_OK {
		return document, Parse_Diagnostic{}, status
	}
	diagnostic = workspace.Parse_Diagnostic.Value.(Parse_Diagnostic)
	if diagnostic.Code == DIAGNOSTIC_NONE {
		diagnostic = failure(
			Parse_Failure_Status_Validated_Value(status),
			Parse_Diagnostic_Position(len(source.Data.Value)),
		)
	}
	return document, diagnostic, status
}

func diagnostic_from_status(
	status_value Parse_Failure_Status_Validated,
	position Parse_Diagnostic_Position,
) (diagnostic Parse_Failure_Diagnostic) {
	defer func() {
		Parse_Failure_Diagnostic_Invariants(
			diagnostic, "diagnostic_from_status.diagnostic",
		)
	}()
	Parse_Failure_Status_Validated_Invariants(
		status_value, "diagnostic_from_status.status_value",
	)
	Parse_Diagnostic_Position_Invariants(position, "diagnostic_from_status.position")
	status := Parse_Status(status_value.Value.(Parse_Failure_Status_Validated_Value))
	return Parse_Failure_Diagnostic{
		Code: Diagnostic_Code(status), Position: Parse_Diagnostic_Position(position),
	}
}

func diagnostic_set(
	workspace_state Syntax_Workspace_Pointer, status_value Parse_Failure_Status_Validated,
	position_value Parse_Error_Position,
) {
	Syntax_Workspace_Pointer_Invariants(workspace_state, "diagnostic_set.workspace_state")
	Parse_Failure_Status_Validated_Invariants(status_value, "diagnostic_set.status_value")
	Parse_Error_Position_Invariants(position_value, "diagnostic_set.position_value")
	workspace := (Syntax_Workspace_Pointer)(workspace_state)
	if workspace.Parse_Diagnostic.Value.(Parse_Diagnostic).Code != DIAGNOSTIC_NONE {
		return
	}
	failure := diagnostic_from_status(
		status_value, Parse_Diagnostic_Position(position_value),
	)
	workspace.Parse_Diagnostic.Value = Parse_Diagnostic_Stored(Parse_Diagnostic(failure))
}

func parse_unchecked(
	source Source, configuration Configuration,
	workspace_state Syntax_Workspace_Pointer,
) (_ Document, status Build_Status) {
	defer func() { Build_Status_Invariants(status, "parse_unchecked.status") }()
	Syntax_Workspace_Pointer_Invariants(workspace_state, "parse_unchecked.workspace_state")
	Source_Invariants(source, "parse_unchecked.source")
	Configuration_Invariants(configuration, "parse_unchecked.configuration")
	var document Document
	defer func() { Document_Invariants(document, "parse_unchecked.document") }()
	workspace, data := (Syntax_Workspace_Pointer)(workspace_state), source.Data.Value
	count := Node_Count_Allocated{Value: Node_Count_Allocated_Value(NODE_COUNT_MINIMUM)}
	root, count, allocation_status := allocate_node(
		workspace, count, Node_Allocation{Value: Node(Node{
			Kind: NODE_LIST, End: Node_End(len(data)),
		})},
	)
	if allocation_status != PARSE_STATUS_OK {
		return document, Build_Status(allocation_status)
	}
	list := List_State_Storage{Value: List_State_Stored(List_State{List: List_Reference(root)})}
	variable_storage := [SYNTAX_VARIABLE_COUNT_MAXIMUM]Node_Reference{}
	variables := Variable_State{
		References: Variable_References(variable_storage[:]),
		Count: Syntax_Variable_Count_Storage{
			Value: Syntax_Variable_Count_Storage_Value(VARIABLE_COUNT_MINIMUM),
		},
	}
	templates := Template_State{}
	Variable_References_Invariants(variables.References, "parse_unchecked.variable_references")
	control_storage := [CONTROL_DEPTH_MAXIMUM]Control_Frame_Storage{}
	controls := Control_Frames(control_storage[:])
	Control_Frames_Invariants(controls, "parse_unchecked.control_frames")
	depth := Control_Depth_Tracked{
		Value: Control_Depth_Tracked_Value(CONTROL_DEPTH_MINIMUM),
	}
	count, depth, status = parse_source(
		source, configuration, workspace, count, &variables, &templates,
		&list, &controls, depth,
	)
	if status != PARSE_STATUS_OK {
		return document, status
	}
	document, parse_status := parse_document(
		source, workspace, Root_Reference(root), count, &templates, depth,
	)
	return document, Build_Status(parse_status)
}

func parse_source(
	source Source, configuration Configuration,
	workspace_state Syntax_Workspace_Pointer, count_value Node_Count_Allocated,
	variables_state Variable_State_Pointer, templates_state Template_State_Pointer,
	list_state List_State_Storage_Pointer, controls_state Control_Frames_Pointer,
	depth_value Control_Depth_Tracked,
) (_ Node_Count_Allocated, _ Control_Depth_Tracked, status Build_Status) {
	defer func() { Build_Status_Invariants(status, "parse_source.status") }()
	Source_Invariants(source, "parse_source.source")
	Configuration_Invariants(configuration, "parse_source.configuration")
	Syntax_Workspace_Pointer_Invariants(workspace_state, "parse_source.workspace_state")
	Node_Count_Allocated_Invariants(count_value, "parse_source.count_value")
	Variable_State_Pointer_Invariants(variables_state, "parse_source.variables_state")
	Template_State_Pointer_Invariants(templates_state, "parse_source.templates_state")
	List_State_Storage_Pointer_Invariants(
		list_state, "parse_source.list_state",
	)
	Control_Frames_Pointer_Invariants(controls_state, "parse_source.controls_state")
	Control_Depth_Tracked_Invariants(depth_value, "parse_source.depth_value")
	count, depth := count_value, depth_value
	defer func() { Node_Count_Allocated_Invariants(count, "parse_source.count") }()
	defer func() { Control_Depth_Tracked_Invariants(depth, "parse_source.depth") }()
	workspace, data := Syntax_Workspace_Pointer(workspace_state), source.Data.Value
	position := Source_Position(SOURCE_SIZE_MINIMUM)
	for int(position) < len(data) {
		cursor := Source_Cursor_Validated{Value: Source_Cursor_Validated_Value(position)}
		left_state := left_delimiter_index(source, cursor, configuration)
		left := Source_Position(left_state.Value.(Source_Cursor_Validated_Value))
		if left == Source_Position(len(data)) {
			var append_status Capacity_Status
			count, append_status = append_source_text(
				source, workspace, count, list_state,
				Text_Span_Stored_Start(position), Text_Span_Stored_End(left), false,
			)
			if status = Build_Status(append_status); status != PARSE_STATUS_OK {
				return count, depth, status
			}
			position = left
			continue
		}
		left_trim, content_start_state := left_trimmed_start(
			source,
			Source_Cursor_Validated{Value: Source_Cursor_Validated_Value(left)},
			configuration,
		)
		var append_status Capacity_Status
		count, append_status = append_source_text(
			source, workspace, count, list_state,
			Text_Span_Stored_Start(position), Text_Span_Stored_End(left), left_trim,
		)
		if status = Build_Status(append_status); status != PARSE_STATUS_OK {
			return count, depth, status
		}
		var right_trim Trim_Whitespace
		var position_state Source_Cursor_Validated
		count, depth, position_state, right_trim, status = parse_delimited_action(
			source, configuration, workspace, count, variables_state, templates_state,
			list_state, controls_state, depth,
			Source_Cursor_Validated{Value: Source_Cursor_Validated_Value(left)},
			content_start_state,
		)
		position = Source_Position(position_state.Value.(Source_Cursor_Validated_Value))
		if status != PARSE_STATUS_OK {
			return count, depth, status
		}
		if right_trim {
			position = Source_Position(skip_space(
				source, Source_Cursor_Validated{
					Value: Source_Cursor_Validated_Value(position),
				},
			).Value.(Source_Cursor_Validated_Value))
		}
	}
	return count, depth, PARSE_STATUS_OK
}

func append_source_text(
	source Source,
	workspace_state Syntax_Workspace_Pointer, count_value Node_Count_Allocated,
	list_state List_State_Storage_Pointer,
	start Text_Span_Start_Proof_Stored, end Text_Span_End_Proof_Stored, trim Trim_Whitespace,
) (_ Node_Count_Allocated, status Capacity_Status) {
	defer func() { Capacity_Status_Invariants(status, "append_source_text.status") }()
	Source_Invariants(source, "append_source_text.source")
	Syntax_Workspace_Pointer_Invariants(workspace_state, "append_source_text.workspace_state")
	Node_Count_Allocated_Invariants(count_value, "append_source_text.count_value")
	List_State_Storage_Pointer_Invariants(list_state, "append_source_text.list_state")
	Text_Span_Start_Proof_Stored_Invariants(start, "append_source_text.start")
	Text_Span_End_Proof_Stored_Invariants(end, "append_source_text.end")
	Trim_Whitespace_Invariants(trim, "append_source_text.trim")
	count := count_value
	defer func() { Node_Count_Allocated_Invariants(count, "append_source_text.count") }()
	span := Text_Span_Validated{Value: Text_Span_Stored{
		Start: start.(Text_Span_Stored_Start), End: end.(Text_Span_Stored_End),
	}}
	var append_status Capacity_Status
	count, append_status = append_text(
		source, workspace_state, count_value, list_state, span, trim,
	)
	return count, append_status
}

func parse_delimited_action(
	source Source, configuration Configuration,
	workspace_state Syntax_Workspace_Pointer, count_value Node_Count_Allocated,
	variables_state Variable_State_Pointer, templates_state Template_State_Pointer,
	list_state List_State_Storage_Pointer, controls_state Control_Frames_Pointer,
	depth_value Control_Depth_Tracked,
	left_state Source_Cursor_Validated, content_start_state Source_Cursor_Validated,
) (_ Node_Count_Allocated, _ Control_Depth_Tracked,
	_ Source_Cursor_Validated, _ Trim_Whitespace, status Build_Status) {
	defer func() { Build_Status_Invariants(status, "parse_delimited_action.status") }()
	Source_Invariants(source, "parse_delimited_action.source")
	Configuration_Invariants(configuration, "parse_delimited_action.configuration")
	Syntax_Workspace_Pointer_Invariants(
		workspace_state, "parse_delimited_action.workspace_state",
	)
	Node_Count_Allocated_Invariants(count_value, "parse_delimited_action.count_value")
	Variable_State_Pointer_Invariants(variables_state, "parse_delimited_action.variables_state")
	Template_State_Pointer_Invariants(templates_state, "parse_delimited_action.templates_state")
	List_State_Storage_Pointer_Invariants(list_state, "parse_delimited_action.list_state")
	Control_Frames_Pointer_Invariants(controls_state, "parse_delimited_action.controls_state")
	Control_Depth_Tracked_Invariants(depth_value, "parse_delimited_action.depth_value")
	Source_Cursor_Validated_Invariants(left_state, "parse_delimited_action.left_state")
	Source_Cursor_Validated_Invariants(
		content_start_state, "parse_delimited_action.content_start_state",
	)
	count, depth := count_value, depth_value
	defer func() { Node_Count_Allocated_Invariants(count, "parse_delimited_action.count") }()
	defer func() { Control_Depth_Tracked_Invariants(depth, "parse_delimited_action.depth") }()
	var end Source_Cursor_Validated
	defer func() { Source_Cursor_Validated_Invariants(end, "parse_delimited_action.end") }()
	var right_trim Trim_Whitespace
	defer func() { Trim_Whitespace_Invariants(right_trim, "parse_delimited_action.trim") }()
	workspace := (Syntax_Workspace_Pointer)(workspace_state)
	left := Source_Position(left_state.Value.(Source_Cursor_Validated_Value))
	content_start := Source_Position(content_start_state.Value.(Source_Cursor_Validated_Value))
	right_state := action_right_index(source, content_start_state, configuration)
	end = right_state
	right := Source_Position(right_state.Value.(Source_Cursor_Validated_Value))
	if right == Source_Position(len(source.Data.Value)) {
		return count_value, depth_value, right_state, false, PARSE_STATUS_SYNTAX_INVALID
	}
	content_end_state, trim := right_trimmed_end(
		source,
		Source_Cursor_Validated{Value: Source_Cursor_Validated_Value(content_start)},
		right_state,
	)
	right_trim = trim
	content_end := Source_Position(content_end_state.Value.(Source_Cursor_Validated_Value))
	cursor := Token_Cursor{
		Position: Cursor_Position_Storage{
			Value: Cursor_Position_Storage_Value(content_start),
		},
		End: Cursor_End_Storage{Value: Cursor_End_Storage_Value(content_end)},
		Token: Token_Storage{Value: Token_Stored(Token{
			Kind: TOKEN_EOF, Start: Token_Start(content_start),
			End: Token_End(content_start),
		})},
		Number: Number_Workspace_Pointer_Storage{
			Value: Number_Workspace_Pointer(&workspace.Number),
		},
		Function: Function_Function_Storage{
			Value: Function_Function_Storage_Value(configuration.Function.Value),
		},
	}
	end = delimiter_end(right_state, configuration, false)
	position := end.Value.(Source_Cursor_Validated_Value)
	span := Action_Span_Validated{Value: Action_Span_Stored(Action_Span{
		Start: Action_Start(left), End: Action_End(position),
		Content_Start: Action_Content_Start(content_start),
		Content_End:   Action_Content_End(content_end),
	})}

	count, depth, status = parse_action(
		source, configuration, workspace, count_value, variables_state, templates_state,
		list_state, controls_state, depth_value, &cursor, span,
	)
	return count, depth, end, right_trim, status
}

func parse_document(
	source Source,
	workspace_state Syntax_Workspace_Pointer, root Root_Reference,
	count Node_Count_Allocated,
	templates_state Template_State_Pointer, depth Control_Depth_Tracked,
) (_ Document, status Lex_Status) {
	defer func() { Lex_Status_Invariants(status, "parse_document.status") }()
	Syntax_Workspace_Pointer_Invariants(workspace_state, "parse_document.workspace_state")
	Node_Count_Allocated_Invariants(count, "parse_document.count")
	Template_State_Pointer_Invariants(templates_state, "parse_document.templates_state")
	Control_Depth_Tracked_Invariants(depth, "parse_document.depth")
	Source_Invariants(source, "parse_document.source")
	Root_Reference_Invariants(root, "parse_document.root")
	var document Document
	defer func() { Document_Invariants(document, "parse_document.document") }()
	templates := (Template_State_Pointer)(templates_state)
	workspace := (Syntax_Workspace_Pointer)(workspace_state)
	if depth.Value.(Control_Depth_Tracked_Value) != CONTROL_DEPTH_MINIMUM {
		diagnostic_set(
			workspace, Parse_Failure_Status_Validated{
				Value: Parse_Failure_Status_Validated_Value(
					PARSE_STATUS_SYNTAX_INVALID,
				),
			},
			Parse_Error_Position(len(source.Data.Value)),
		)
		return document, PARSE_STATUS_SYNTAX_INVALID
	}
	status = templates_compact(source, workspace, templates)
	if status != PARSE_STATUS_OK {
		return document, status
	}
	first_template := Document_First_Template(NO_NODE)
	if Template_First(templates.First.Value) != Template_First(NO_NODE) {
		first_template = Document_First_Template(
			Node_Reference(templates.First.Value) -
				Node_Reference(ROOT_NODE_COUNT),
		)
	}
	document = Document{
		Root:           Document_Root(root),
		Node_Count:     Node_Count(count.Value.(Node_Count_Allocated_Value)),
		First_Template: first_template,
		Template_Count: Template_Count(templates.Count.Value),
	}
	return document, PARSE_STATUS_OK
}

func append_text(
	source Source,
	workspace_state Syntax_Workspace_Pointer, count_value Node_Count_Allocated,
	list_state List_State_Storage_Pointer, span_state Text_Span_Validated,
	trim Trim_Whitespace,
) (_ Node_Count_Allocated, status Capacity_Status) {
	defer func() { Capacity_Status_Invariants(status, "append_text.status") }()
	Syntax_Workspace_Pointer_Invariants(workspace_state, "append_text.workspace_state")
	Node_Count_Allocated_Invariants(count_value, "append_text.count_value")
	List_State_Storage_Pointer_Invariants(list_state, "append_text.list_state")
	Text_Span_Validated_Invariants(span_state, "append_text.span_state")
	Source_Invariants(source, "append_text.source")
	Trim_Whitespace_Invariants(trim, "append_text.trim")
	count := count_value
	defer func() { Node_Count_Allocated_Invariants(count, "append_text.count") }()
	workspace := (Syntax_Workspace_Pointer)(workspace_state)
	if bool(trim) {
		span_state = trim_space_end(source, span_state)
	}
	span := span_state.Value
	start, end := span.Start, span.End
	if Source_Position(start) == Source_Position(end) {
		return count, PARSE_STATUS_OK
	}
	node := Node{
		Kind: NODE_TEXT, Start: Node_Start(start), End: Node_End(end),
		Value_Start: Node_Value_Start(start), Value_End: Node_Value_End(end),
	}
	reference, updated_count, allocation_status := allocate_node(
		workspace, count, Node_Allocation{Value: Node(node)},
	)
	count = updated_count
	if allocation_status != PARSE_STATUS_OK {
		return count, Capacity_Status(allocation_status)
	}
	append_list_child(
		workspace, list_state,
		Node_Link_Validated{Value: Node_Link_Validated_Value(reference)},
	)
	return count, PARSE_STATUS_OK
}

func parse_action(
	source Source, configuration Configuration,
	workspace_state Syntax_Workspace_Pointer, count_value Node_Count_Allocated,
	variables_state Variable_State_Pointer, templates_state Template_State_Pointer,
	list_state List_State_Storage_Pointer, controls_state Control_Frames_Pointer,
	depth_value Control_Depth_Tracked, cursor_state Token_Cursor_Pointer,
	span_state Action_Span_Validated,
) (_ Node_Count_Allocated, _ Control_Depth_Tracked, status Build_Status) {
	defer func() { Build_Status_Invariants(status, "parse_action.status") }()
	Syntax_Workspace_Pointer_Invariants(workspace_state, "parse_action.workspace_state")
	Node_Count_Allocated_Invariants(count_value, "parse_action.count_value")
	Variable_State_Pointer_Invariants(variables_state, "parse_action.variables_state")
	Template_State_Pointer_Invariants(templates_state, "parse_action.templates_state")
	List_State_Storage_Pointer_Invariants(list_state, "parse_action.list_state")
	Control_Frames_Pointer_Invariants(controls_state, "parse_action.controls_state")
	Control_Depth_Tracked_Invariants(depth_value, "parse_action.depth_value")
	Token_Cursor_Pointer_Invariants(cursor_state, "parse_action.cursor_state")
	Action_Span_Validated_Invariants(span_state, "parse_action.span_state")
	Source_Invariants(source, "parse_action.source")
	Configuration_Invariants(configuration, "parse_action.configuration")
	count, depth := count_value, depth_value
	defer func() { Node_Count_Allocated_Invariants(count, "parse_action.count") }()
	defer func() { Control_Depth_Tracked_Invariants(depth, "parse_action.depth") }()
	defer func() {
		diagnostic_set_cursor(
			Syntax_Workspace_Pointer(workspace_state), status,
			Token_Cursor_Pointer(cursor_state),
		)
	}()
	variables := (Variable_State_Pointer)(variables_state)
	templates := (Template_State_Pointer)(templates_state)
	controls := (Control_Frames_Pointer)(controls_state)
	workspace := (Syntax_Workspace_Pointer)(workspace_state)
	cursor := (Token_Cursor_Pointer)(cursor_state)
	status = Build_Status(token_peek(source, cursor))
	if status != PARSE_STATUS_OK {
		return count, depth, status
	}
	first := Token(cursor.Token.Value)
	switch first.Kind {
	case TOKEN_IF, TOKEN_RANGE, TOKEN_WITH, TOKEN_ELSE, TOKEN_END,
		TOKEN_BREAK, TOKEN_CONTINUE:
		count, depth, status = parse_control_action(
			source, configuration, workspace, count, variables, list_state, controls,
			depth, cursor, span_state,
			Control_Action_Token_Kind_Validated{
				Value: Control_Action_Token_Kind_Validated_Value(first.Kind),
			},
		)
		return count, depth, status
	case TOKEN_COMMENT:
		count, status = append_comment(
			source, configuration, workspace, count, list_state,
			cursor, span_state,
		)
		return count, depth, status
	case TOKEN_TEMPLATE:
		count, status = append_template_invocation(
			source, configuration, workspace, count, variables,
			list_state, cursor, span_state,
		)
		return count, depth, status
	case TOKEN_DEFINE, TOKEN_BLOCK:
		count, depth, status = template_open(
			source, configuration, workspace, count, variables,
			templates, list_state, controls, depth, cursor,
			span_state,
		)
		return count, depth, status
	default:
		count, status = append_output_action(
			source, configuration, workspace, count, variables,
			list_state, cursor, span_state,
		)
		return count, depth, status
	}
}

func parse_control_action(
	source Source,
	configuration Configuration,
	workspace_state Syntax_Workspace_Pointer, count_value Node_Count_Allocated,
	variables_state Variable_State_Pointer, list_state List_State_Storage_Pointer,
	controls_state Control_Frames_Pointer, depth_value Control_Depth_Tracked,
	cursor_state Token_Cursor_Pointer, span_state Action_Span_Validated,
	kind_value Control_Action_Token_Kind_Validated,
) (_ Node_Count_Allocated, _ Control_Depth_Tracked, status Build_Status) {
	defer func() { Build_Status_Invariants(status, "parse_control_action.status") }()
	Source_Invariants(source, "parse_control_action.source")
	Configuration_Invariants(configuration, "parse_control_action.configuration")
	Syntax_Workspace_Pointer_Invariants(workspace_state, "parse_control_action.workspace_state")
	Node_Count_Allocated_Invariants(count_value, "parse_control_action.count_value")
	Variable_State_Pointer_Invariants(variables_state, "parse_control_action.variables_state")
	List_State_Storage_Pointer_Invariants(list_state, "parse_control_action.list_state")
	Control_Frames_Pointer_Invariants(controls_state, "parse_control_action.controls_state")
	Control_Depth_Tracked_Invariants(depth_value, "parse_control_action.depth_value")
	Token_Cursor_Pointer_Invariants(cursor_state, "parse_control_action.cursor_state")
	Action_Span_Validated_Invariants(span_state, "parse_control_action.span_state")
	Control_Action_Token_Kind_Validated_Invariants(
		kind_value, "parse_control_action.kind_value",
	)
	count, depth := count_value, depth_value
	defer func() { Node_Count_Allocated_Invariants(count, "parse_control_action.count") }()
	defer func() { Control_Depth_Tracked_Invariants(depth, "parse_control_action.depth") }()
	workspace := (Syntax_Workspace_Pointer)(workspace_state)
	variables := (Variable_State_Pointer)(variables_state)
	controls := Control_Frames_Pointer(controls_state)
	cursor := Token_Cursor_Pointer(cursor_state)
	kind := Token_Kind(kind_value.Value)
	switch kind {
	case TOKEN_IF, TOKEN_RANGE, TOKEN_WITH:
		count, depth, status = control_open(
			source, configuration, workspace, count_value, variables,
			list_state, controls, depth_value, cursor, span_state,
		)
		return count, depth, status
	case TOKEN_ELSE:
		count, depth, status = control_else(
			source, configuration, workspace, count_value, variables, list_state,
			controls, depth_value, cursor, span_state,
		)
		return count, depth, status
	case TOKEN_END:
		var end_status Lex_Status
		depth, end_status = control_end(
			source, workspace, variables, list_state, controls, depth_value, cursor,
			span_state,
		)
		status = Build_Status(end_status)
		return count, depth, status
	case TOKEN_BREAK, TOKEN_CONTINUE:
		count, status = control_flow(
			source, workspace, count_value, list_state, controls, depth_value, cursor,
			span_state,
		)
		return count, depth, status
	default:
		status = PARSE_STATUS_SYNTAX_INVALID
		return count, depth, status
	}
}

func diagnostic_set_cursor(
	workspace_state Syntax_Workspace_Pointer, status Build_Status,
	cursor_state Token_Cursor_Pointer,
) {
	Syntax_Workspace_Pointer_Invariants(
		workspace_state, "diagnostic_set_cursor.workspace_state",
	)
	Build_Status_Invariants(status, "diagnostic_set_cursor.status")
	Token_Cursor_Pointer_Invariants(cursor_state, "diagnostic_set_cursor.cursor_state")
	workspace := (Syntax_Workspace_Pointer)(workspace_state)
	cursor := (Token_Cursor_Pointer)(cursor_state)
	if Build_Status(status) == PARSE_STATUS_OK {
		return
	}
	position := Parse_Error_Position(cursor.Position.Value)
	if cursor.Has_Last.Value != Cursor_Has_Last_Storage_Value(0) {
		position = Parse_Error_Position(cursor.Last_Start.Value)
	}
	diagnostic_set(
		workspace, Parse_Failure_Status_Validated{
			Value: Parse_Failure_Status_Validated_Value(status),
		}, position,
	)
}

func append_output_action(
	source Source,
	configuration Configuration,
	workspace_state Syntax_Workspace_Pointer, count_value Node_Count_Allocated,
	variables_state Variable_State_Pointer, list_state List_State_Storage_Pointer,
	cursor_state Token_Cursor_Pointer, span_state Action_Span_Validated,
) (_ Node_Count_Allocated, status Build_Status) {
	defer func() { Build_Status_Invariants(status, "append_output_action.status") }()
	Syntax_Workspace_Pointer_Invariants(workspace_state, "append_output_action.workspace_state")
	Node_Count_Allocated_Invariants(count_value, "append_output_action.count_value")
	Variable_State_Pointer_Invariants(variables_state, "append_output_action.variables_state")
	List_State_Storage_Pointer_Invariants(list_state, "append_output_action.list_state")
	Token_Cursor_Pointer_Invariants(cursor_state, "append_output_action.cursor_state")
	Action_Span_Validated_Invariants(span_state, "append_output_action.span_state")
	Source_Invariants(source, "append_output_action.source")
	Configuration_Invariants(configuration, "append_output_action.configuration")
	span := span_state.Value
	action_start, action_end := span.Start, span.End
	content_start, content_end := span.Content_Start, span.Content_End
	count := count_value
	defer func() { Node_Count_Allocated_Invariants(count, "append_output_action.count") }()
	variables := (Variable_State_Pointer)(variables_state)
	workspace := (Syntax_Workspace_Pointer)(workspace_state)
	cursor := (Token_Cursor_Pointer)(cursor_state)
	action, updated_count, allocation_status := allocate_node(
		workspace, count, Node_Allocation{Value: Node(Node{
			Kind: NODE_ACTION, Start: Node_Start(action_start),
			End:         Node_End(action_end),
			Value_Start: Node_Value_Start(content_start),
			Value_End:   Node_Value_End(content_end),
		})},
	)
	count = updated_count
	if allocation_status != PARSE_STATUS_OK {
		return count, Build_Status(allocation_status)
	}
	var pipe Pipeline_Root_Validated
	pipe, count, status = parse_pipeline(
		source, configuration, workspace, count, variables,
		PIPELINE_DECLARATION_COUNT_MINIMUM, cursor,
	)
	if status != PARSE_STATUS_OK {
		return count, status
	}
	status = Build_Status(token_take(source, cursor))
	if status != PARSE_STATUS_OK {
		return count, status
	}
	token := Token(cursor.Token.Value)
	if token.Kind != TOKEN_EOF {
		return count, PARSE_STATUS_SYNTAX_INVALID
	}
	workspace.Nodes[action-NODE_COUNT_INCREMENT].First_Child = Node_First_Child(
		pipe.Value.(Pipeline_Root_Validated_Value),
	)
	append_list_child(
		workspace, list_state,
		Node_Link_Validated{Value: Node_Link_Validated_Value(action)},
	)
	return count, PARSE_STATUS_OK
}

func append_comment(
	source Source,
	configuration Configuration,
	workspace_state Syntax_Workspace_Pointer, count_value Node_Count_Allocated,
	list_state List_State_Storage_Pointer, cursor_state Token_Cursor_Pointer,
	span_state Action_Span_Validated,
) (_ Node_Count_Allocated, status Build_Status) {
	defer func() { Build_Status_Invariants(status, "append_comment.status") }()
	Syntax_Workspace_Pointer_Invariants(workspace_state, "append_comment.workspace_state")
	Node_Count_Allocated_Invariants(count_value, "append_comment.count_value")
	List_State_Storage_Pointer_Invariants(list_state, "append_comment.list_state")
	Token_Cursor_Pointer_Invariants(cursor_state, "append_comment.cursor_state")
	Action_Span_Validated_Invariants(span_state, "append_comment.span_state")
	Source_Invariants(source, "append_comment.source")
	Configuration_Invariants(configuration, "append_comment.configuration")
	span := span_state.Value
	action_start, action_end := span.Start, span.End
	count := count_value
	defer func() { Node_Count_Allocated_Invariants(count, "append_comment.count") }()
	workspace := (Syntax_Workspace_Pointer)(workspace_state)
	cursor := (Token_Cursor_Pointer)(cursor_state)
	status = Build_Status(token_take(source, cursor))
	if status != PARSE_STATUS_OK {
		return count, status
	}
	comment := Token(cursor.Token.Value)
	_, content_start_state := left_trimmed_start(
		source,
		Source_Cursor_Validated{
			Value: Source_Cursor_Validated_Value(Source_Position(action_start)),
		},
		configuration,
	)
	content_start := Source_Position(
		content_start_state.Value.(Source_Cursor_Validated_Value),
	)
	if Source_Position(comment.Start) != content_start {
		return count, PARSE_STATUS_SYNTAX_INVALID
	}
	if comment.End != Token_End(cursor.End.Value) {
		return count, PARSE_STATUS_SYNTAX_INVALID
	}
	status = Build_Status(token_take(source, cursor))
	if status != PARSE_STATUS_OK {
		return count, status
	}
	boundary := Token(cursor.Token.Value)
	if boundary.Kind != TOKEN_EOF {
		return count, PARSE_STATUS_SYNTAX_INVALID
	}
	if Mode(configuration.Mode.Value)&PARSE_COMMENTS == MODE_MINIMUM {
		return count, PARSE_STATUS_OK
	}
	reference, updated_count, allocation_status := allocate_node(
		workspace, count, Node_Allocation{Value: Node(Node{
			Kind: NODE_COMMENT, Start: Node_Start(action_start),
			End:         Node_End(action_end),
			Value_Start: Node_Value_Start(comment.Start),
			Value_End:   Node_Value_End(comment.End),
		})},
	)
	count = updated_count
	if allocation_status != PARSE_STATUS_OK {
		return count, Build_Status(allocation_status)
	}
	append_list_child(
		workspace, list_state,
		Node_Link_Validated{Value: Node_Link_Validated_Value(reference)},
	)
	return count, PARSE_STATUS_OK
}

func append_template_invocation(
	source Source, configuration Configuration,
	workspace_state Syntax_Workspace_Pointer, count_value Node_Count_Allocated,
	variables_state Variable_State_Pointer, list_state List_State_Storage_Pointer,
	cursor_state Token_Cursor_Pointer, span_state Action_Span_Validated,
) (_ Node_Count_Allocated, status Build_Status) {
	defer func() { Build_Status_Invariants(status, "append_template_invocation.status") }()
	Syntax_Workspace_Pointer_Invariants(
		workspace_state, "append_template_invocation.workspace_state",
	)
	Node_Count_Allocated_Invariants(count_value, "append_template_invocation.count_value")
	Variable_State_Pointer_Invariants(
		variables_state, "append_template_invocation.variables_state",
	)
	List_State_Storage_Pointer_Invariants(list_state, "append_template_invocation.list_state")
	Token_Cursor_Pointer_Invariants(cursor_state, "append_template_invocation.cursor_state")
	Action_Span_Validated_Invariants(span_state, "append_template_invocation.span_state")
	Source_Invariants(source, "append_template_invocation.source")
	Configuration_Invariants(configuration, "append_template_invocation.configuration")
	count, workspace := count_value, Syntax_Workspace_Pointer(workspace_state)
	defer func() {
		Node_Count_Allocated_Invariants(count, "append_template_invocation.count")
	}()
	cursor := Token_Cursor_Pointer(cursor_state)
	if status = Build_Status(token_take(source, cursor)); status != PARSE_STATUS_OK {
		return count, status
	}
	if status = Build_Status(token_take(source, cursor)); status != PARSE_STATUS_OK {
		return count, status
	}
	name := Token(cursor.Token.Value)
	if name.Kind != TOKEN_STRING {
		if name.Kind != TOKEN_RAW_STRING {
			return count, PARSE_STATUS_SYNTAX_INVALID
		}
	}
	span := span_state.Value
	node := Node{
		Kind: NODE_TEMPLATE, Start: Node_Start(span.Start), End: Node_End(span.End),
		Value_Start: Node_Value_Start(name.Start), Value_End: Node_Value_End(name.End),
	}
	reference, updated_count, allocation_status := allocate_node(
		workspace, count, Node_Allocation{Value: Node(node)},
	)
	count = updated_count
	if allocation_status != PARSE_STATUS_OK {
		return count, Build_Status(allocation_status)
	}
	if status = Build_Status(token_peek(source, cursor)); status != PARSE_STATUS_OK {
		return count, status
	}
	if cursor.Token.Value.Kind != TOKEN_EOF {
		pipe, pipeline_count, pipe_status := parse_pipeline(
			source, configuration, workspace, count, variables_state,
			PIPELINE_DECLARATION_COUNT_MINIMUM, cursor,
		)
		count = pipeline_count
		if pipe_status != PARSE_STATUS_OK {
			return count, pipe_status
		}
		workspace.Nodes[reference-NODE_COUNT_INCREMENT].First_Child =
			Node_First_Child(pipe.Value.(Pipeline_Root_Validated_Value))
	}
	if status = Build_Status(token_take(source, cursor)); status != PARSE_STATUS_OK {
		return count, status
	}
	if cursor.Token.Value.Kind != TOKEN_EOF {
		return count, PARSE_STATUS_SYNTAX_INVALID
	}
	append_list_child(
		workspace, list_state,
		Node_Link_Validated{Value: Node_Link_Validated_Value(reference)},
	)
	return count, PARSE_STATUS_OK
}

func template_open(
	source Source,
	configuration Configuration,
	workspace_state Syntax_Workspace_Pointer, count_value Node_Count_Allocated,
	variables_state Variable_State_Pointer, templates_state Template_State_Pointer,
	list_state List_State_Storage_Pointer, controls_state Control_Frames_Pointer,
	depth_value Control_Depth_Tracked,
	cursor_state Token_Cursor_Pointer, span_state Action_Span_Validated,
) (_ Node_Count_Allocated, _ Control_Depth_Tracked, status Build_Status) {
	defer func() { Build_Status_Invariants(status, "template_open.status") }()
	Syntax_Workspace_Pointer_Invariants(workspace_state, "template_open.workspace_state")
	Node_Count_Allocated_Invariants(count_value, "template_open.count_value")
	Variable_State_Pointer_Invariants(variables_state, "template_open.variables_state")
	Template_State_Pointer_Invariants(templates_state, "template_open.templates_state")
	List_State_Storage_Pointer_Invariants(list_state, "template_open.list_state")
	Control_Frames_Pointer_Invariants(controls_state, "template_open.controls_state")
	Control_Depth_Tracked_Invariants(depth_value, "template_open.depth_value")
	Token_Cursor_Pointer_Invariants(cursor_state, "template_open.cursor_state")
	Action_Span_Validated_Invariants(span_state, "template_open.span_state")
	Source_Invariants(source, "template_open.source")
	Configuration_Invariants(configuration, "template_open.configuration")
	count, depth := count_value, depth_value
	defer func() { Node_Count_Allocated_Invariants(count, "template_open.count") }()
	defer func() { Control_Depth_Tracked_Invariants(depth, "template_open.depth") }()
	span := span_state.Value
	action_start, action_end := span.Start, span.End
	variables := (Variable_State_Pointer)(variables_state)
	templates := (Template_State_Pointer)(templates_state)
	controls := (Control_Frames_Pointer)(controls_state)
	workspace := (Syntax_Workspace_Pointer)(workspace_state)
	cursor := (Token_Cursor_Pointer)(cursor_state)
	keyword_state, name_state, lex_status := template_header(source, cursor, depth)
	if lex_status != PARSE_STATUS_OK {
		return count, depth, Build_Status(lex_status)
	}
	keyword := keyword_state.Value
	name := name_state.Value
	template_node := Node{
		Kind: NODE_TEMPLATE, Start: Node_Start(action_start), End: Node_End(action_end),
		Value_Start: Node_Value_Start(name.Start),
		Value_End:   Node_Value_End(name.End),
	}
	if keyword.Kind == TOKEN_BLOCK {
		invocation, updated_count, allocation_status := allocate_node(
			workspace, count, Node_Allocation{Value: Node(template_node)},
		)
		count = updated_count
		if allocation_status != PARSE_STATUS_OK {
			return count, depth, Build_Status(allocation_status)
		}
		count, status = template_block_invocation(
			source, configuration, workspace, count, variables,
			list_state, cursor,
			Node_Link_Validated{Value: Node_Link_Validated_Value(invocation)},
		)
		if status != PARSE_STATUS_OK {
			return count, depth, status
		}
	} else {
		if !bool(template_boundary(source, cursor)) {
			return count, depth, PARSE_STATUS_SYNTAX_INVALID
		}
	}
	definition, updated_count, allocation_status := allocate_node(
		workspace, count, Node_Allocation{Value: Node(template_node)},
	)
	count = updated_count
	if allocation_status != PARSE_STATUS_OK {
		return count, depth, Build_Status(allocation_status)
	}
	var definition_status Capacity_Status
	count, depth, definition_status = template_definition(
		workspace, count, variables, templates, list_state, controls,
		depth, Node_Link_Validated{Value: Node_Link_Validated_Value(definition)},
	)
	return count, depth, Build_Status(definition_status)
}

func template_header(
	source Source, cursor_state Token_Cursor_Pointer, depth_state Control_Depth_Tracked,
) (_ Token_Validated, _ Token_Validated, status Lex_Status) {
	defer func() { Lex_Status_Invariants(status, "template_header.status") }()
	Source_Invariants(source, "template_header.source")
	Token_Cursor_Pointer_Invariants(cursor_state, "template_header.cursor_state")
	Control_Depth_Tracked_Invariants(depth_state, "template_header.depth_state")
	var keyword Token_Validated
	defer func() { Token_Validated_Invariants(keyword, "template_header.keyword") }()
	var name Token_Validated
	defer func() { Token_Validated_Invariants(name, "template_header.name") }()
	cursor := (Token_Cursor_Pointer)(cursor_state)
	depth := depth_state.Value.(Control_Depth_Tracked_Value)
	status = token_peek(source, cursor)
	if status != PARSE_STATUS_OK {
		return Token_Validated{}, Token_Validated{}, status
	}
	keyword = Token_Validated{Value: Token_Validated_Value(cursor.Token.Value)}
	aver.Always(
		Control_Depth(depth) < Control_Depth(CONTROL_DEPTH_MAXIMUM),
		"Source density leaves one frame for every reachable template opener.",
	)
	if keyword.Value.Kind == TOKEN_DEFINE {
		if depth != CONTROL_DEPTH_MINIMUM {
			return keyword, Token_Validated{}, PARSE_STATUS_SYNTAX_INVALID
		}
	}
	status = token_take(source, cursor)
	if status != PARSE_STATUS_OK {
		return keyword, Token_Validated{}, status
	}
	status = token_take(source, cursor)
	if status != PARSE_STATUS_OK {
		return keyword, Token_Validated{}, status
	}
	name = Token_Validated{Value: Token_Validated_Value(cursor.Token.Value)}
	if name.Value.Kind != TOKEN_STRING {
		if name.Value.Kind != TOKEN_RAW_STRING {
			return keyword, name, PARSE_STATUS_SYNTAX_INVALID
		}
	}
	return keyword, name, PARSE_STATUS_OK
}

func template_block_invocation(
	source Source,
	configuration Configuration,
	workspace_state Syntax_Workspace_Pointer, count_value Node_Count_Allocated,
	variables_state Variable_State_Pointer, list_state List_State_Storage_Pointer,
	cursor_state Token_Cursor_Pointer, invocation_value Node_Link_Validated,
) (_ Node_Count_Allocated, status Build_Status) {
	defer func() { Build_Status_Invariants(status, "template_block_invocation.status") }()
	Syntax_Workspace_Pointer_Invariants(
		workspace_state, "template_block_invocation.workspace_state",
	)
	Node_Count_Allocated_Invariants(count_value, "template_block_invocation.count_value")
	Variable_State_Pointer_Invariants(
		variables_state, "template_block_invocation.variables_state",
	)
	List_State_Storage_Pointer_Invariants(list_state, "template_block_invocation.list_state")
	Token_Cursor_Pointer_Invariants(cursor_state, "template_block_invocation.cursor_state")
	Node_Link_Validated_Invariants(
		invocation_value, "template_block_invocation.invocation_value",
	)
	Source_Invariants(source, "template_block_invocation.source")
	Configuration_Invariants(configuration, "template_block_invocation.configuration")
	count := count_value
	defer func() {
		Node_Count_Allocated_Invariants(count, "template_block_invocation.count")
	}()
	variables := (Variable_State_Pointer)(variables_state)
	workspace := (Syntax_Workspace_Pointer)(workspace_state)
	cursor := (Token_Cursor_Pointer)(cursor_state)
	invocation := invocation_value.Value.(Node_Link_Validated_Value)
	var pipe Pipeline_Root_Validated
	pipe, count, status = parse_pipeline(
		source, configuration, workspace, count, variables,
		PIPELINE_DECLARATION_COUNT_MINIMUM, cursor,
	)
	if status != PARSE_STATUS_OK {
		return count, status
	}
	workspace.Nodes[invocation-NODE_COUNT_INCREMENT].First_Child = Node_First_Child(
		pipe.Value.(Pipeline_Root_Validated_Value),
	)
	if !bool(template_boundary(source, cursor)) {
		return count, PARSE_STATUS_SYNTAX_INVALID
	}
	append_list_child(workspace, list_state, invocation_value)
	return count, PARSE_STATUS_OK
}

func template_boundary(
	source Source, cursor_state Token_Cursor_Pointer) (boundary Keyword_Match) {
	defer func() { Keyword_Match_Invariants(boundary, "template_boundary.boundary") }()
	Token_Cursor_Pointer_Invariants(cursor_state, "template_boundary.cursor_state")
	Source_Invariants(source, "template_boundary.source")
	cursor := (Token_Cursor_Pointer)(cursor_state)
	status := token_take(source, cursor)
	if status != PARSE_STATUS_OK {
		return false
	}
	token := Token(cursor.Token.Value)
	return Keyword_Match(token.Kind == TOKEN_EOF)
}

func template_definition(
	workspace_state Syntax_Workspace_Pointer, count_value Node_Count_Allocated,
	variables_state Variable_State_Pointer, templates_state Template_State_Pointer,
	list_state List_State_Storage_Pointer, controls_state Control_Frames_Pointer,
	depth_value Control_Depth_Tracked,
	definition_value Node_Link_Validated,
) (_ Node_Count_Allocated, _ Control_Depth_Tracked, status Capacity_Status) {
	defer func() { Capacity_Status_Invariants(status, "template_definition.status") }()
	Syntax_Workspace_Pointer_Invariants(workspace_state, "template_definition.workspace_state")
	Node_Count_Allocated_Invariants(count_value, "template_definition.count_value")
	Variable_State_Pointer_Invariants(variables_state, "template_definition.variables_state")
	Template_State_Pointer_Invariants(templates_state, "template_definition.templates_state")
	List_State_Storage_Pointer_Invariants(list_state, "template_definition.list_state")
	Control_Frames_Pointer_Invariants(controls_state, "template_definition.controls_state")
	Control_Depth_Tracked_Invariants(depth_value, "template_definition.depth_value")
	Node_Link_Validated_Invariants(
		definition_value, "template_definition.definition_value",
	)
	count, depth := count_value, depth_value
	defer func() {
		Node_Count_Allocated_Invariants(count, "template_definition.count")
	}()
	defer func() {
		Control_Depth_Tracked_Invariants(depth, "template_definition.depth")
	}()
	variables := (Variable_State_Pointer)(variables_state)
	templates := (Template_State_Pointer)(templates_state)
	list := &list_state.Value
	controls := (Control_Frames_Pointer)(controls_state)
	workspace := (Syntax_Workspace_Pointer)(workspace_state)
	definition := definition_value.Value.(Node_Link_Validated_Value)
	definition_node := workspace.Nodes[definition-NODE_COUNT_INCREMENT]
	action_end := Source_Position(definition_node.End)
	body, updated_count, allocation_status := allocate_node(
		workspace, count, Node_Allocation{Value: Node(Node{
			Kind: NODE_LIST, Start: Node_Start(action_end), End: Node_End(action_end),
		})})
	count = updated_count
	if allocation_status != PARSE_STATUS_OK {
		return count, depth, Capacity_Status(allocation_status)
	}
	workspace.Nodes[definition-NODE_COUNT_INCREMENT].First_Child = Node_First_Child(body)
	if Template_First(templates.First.Value) == Template_First(NO_NODE) {
		templates.First.Value = Template_First_Storage_Value(definition)
	} else {
		last := templates.Last.Value
		workspace.Nodes[last-NODE_COUNT_INCREMENT].Next_Sibling =
			Node_Next_Sibling(definition)
	}
	templates.Last.Value = Template_Last_Storage_Value(definition)
	templates.Count.Value++
	depth_scalar := depth.Value.(Control_Depth_Tracked_Value)
	(*controls)[depth_scalar].Value = Control_Frame_Stored(Control_Frame{
		Branch:            Control_Branch(definition),
		Parent_List:       Control_Parent_List(list.List),
		Parent_Last_Child: Control_Parent_Last_Child(list.Last_Child),
		Parent_Variable_Count: Control_Parent_Variable_Count(
			variables.Count.Value.(Syntax_Variable_Count_Storage_Value),
		),
		Kind: NODE_TEMPLATE,
	})
	depth.Value = depth_scalar + CONTROL_DEPTH_INCREMENT
	variables.Count.Value = Syntax_Variable_Count_Storage_Value(VARIABLE_COUNT_MINIMUM)
	*list = List_State_Stored(List_State{List: List_Reference(body)})
	return count, depth, PARSE_STATUS_OK
}

func templates_compact(
	source Source, workspace_state Syntax_Workspace_Pointer,
	templates_state Template_State_Pointer,
) (status Lex_Status) {
	defer func() { Lex_Status_Invariants(status, "templates_compact.status") }()
	Syntax_Workspace_Pointer_Invariants(workspace_state, "templates_compact.workspace_state")
	Template_State_Pointer_Invariants(templates_state, "templates_compact.templates_state")
	Source_Invariants(source, "templates_compact.source")
	templates := (Template_State_Pointer)(templates_state)
	workspace := (Syntax_Workspace_Pointer)(workspace_state)
	original := Node_Reference(templates.First.Value)
	kept := Template_State{}
	for original != NO_NODE {
		current := Template_Reference_Validated{
			Value: Template_Reference_Validated_Value(original),
		}
		next := Node_Reference(
			workspace.Nodes[original-NODE_COUNT_INCREMENT].Next_Sibling,
		)
		workspace.Nodes[original-NODE_COUNT_INCREMENT].Next_Sibling =
			Node_Next_Sibling(NO_NODE)
		search_state := template_find(source, workspace, &kept, current)
		search := search_state.Value
		if search.Match == Template_Match_Reference(NO_NODE) {
			if Template_First(kept.First.Value) == Template_First(NO_NODE) {
				kept.First.Value = Template_First_Storage_Value(original)
			} else {
				last := Node_Reference(kept.Last.Value)
				workspace.Nodes[last-NODE_COUNT_INCREMENT].Next_Sibling =
					Node_Next_Sibling(original)
			}
			kept.Last.Value = Template_Last_Storage_Value(original)
			kept.Count.Value++
		} else {
			status = template_merge(
				source, workspace, &kept, current, search_state,
			)
			if status != PARSE_STATUS_OK {
				return status
			}
		}
		original = next
	}
	*templates = kept
	return PARSE_STATUS_OK
}

func template_find(
	source Source,
	workspace_state Syntax_Workspace_Pointer,
	templates_state Template_State_Pointer,
	current_value Template_Reference_Validated,
) (search Template_Search_Validated) {
	defer func() {
		Template_Search_Validated_Invariants(search, "template_find.search")
	}()
	Syntax_Workspace_Pointer_Invariants(workspace_state, "template_find.workspace_state")
	Template_State_Pointer_Invariants(templates_state, "template_find.templates_state")
	Template_Reference_Validated_Invariants(current_value, "template_find.current_value")
	Source_Invariants(source, "template_find.source")
	templates := (Template_State_Pointer)(templates_state)
	workspace := (Syntax_Workspace_Pointer)(workspace_state)
	previous := Template_Previous_Reference(NO_NODE)
	reference := Node_Reference(templates.First.Value)
	for reference != NO_NODE {
		if bool(template_names_equal(
			source, workspace, Template_Reference_Validated{
				Value: Template_Reference_Validated_Value(reference),
			}, current_value,
		)) {
			return Template_Search_Validated{Value: Template_Search_Stored(
				Template_Search{
					Match:    Template_Match_Reference(reference),
					Previous: previous,
				})}

		}
		previous = Template_Previous_Reference(reference)
		reference = Node_Reference(
			workspace.Nodes[reference-NODE_COUNT_INCREMENT].Next_Sibling,
		)
	}
	return Template_Search_Validated{Value: Template_Search_Stored(Template_Search{
		Match: Template_Match_Reference(NO_NODE), Previous: previous,
	})}

}

func template_merge(
	source Source,
	workspace_state Syntax_Workspace_Pointer,
	templates_state Template_State_Pointer,
	current_value Template_Reference_Validated,
	search_value Template_Search_Validated,
) (status Lex_Status) {
	defer func() { Lex_Status_Invariants(status, "template_merge.status") }()
	Syntax_Workspace_Pointer_Invariants(workspace_state, "template_merge.workspace_state")
	Template_State_Pointer_Invariants(templates_state, "template_merge.templates_state")
	Template_Reference_Validated_Invariants(current_value, "template_merge.current_value")
	Template_Search_Validated_Invariants(search_value, "template_merge.search_value")
	Source_Invariants(source, "template_merge.source")
	templates := (Template_State_Pointer)(templates_state)
	workspace := (Syntax_Workspace_Pointer)(workspace_state)
	current := current_value.Value.(Template_Reference_Validated_Value)
	search := search_value.Value
	prior := Node_Reference(search.Match)
	prior_empty := template_empty(source, workspace, Template_Reference_Validated{
		Value: Template_Reference_Validated_Value(prior),
	})
	current_empty := template_empty(source, workspace, current_value)
	if !bool(prior_empty) {
		if !bool(current_empty) {
			failure := Parse_Failure_Status_Validated{
				Value: Parse_Failure_Status_Validated_Value(
					PARSE_STATUS_SYNTAX_INVALID,
				),
			}

			diagnostic_set(
				workspace, failure,
				Parse_Error_Position(
					workspace.Nodes[current-NODE_COUNT_INCREMENT].Start,
				),
			)
			return PARSE_STATUS_SYNTAX_INVALID
		}
		return PARSE_STATUS_OK
	}
	successor := workspace.Nodes[prior-NODE_COUNT_INCREMENT].Next_Sibling
	workspace.Nodes[current-NODE_COUNT_INCREMENT].Next_Sibling = successor
	if search.Previous ==
		Template_Previous_Reference(NO_NODE) {
		templates.First.Value = Template_First_Storage_Value(current)
	} else {
		workspace.Nodes[search.Previous-NODE_COUNT_INCREMENT].Next_Sibling =
			Node_Next_Sibling(current)
	}
	if Template_Last(templates.Last.Value) == Template_Last(prior) {
		templates.Last.Value = Template_Last_Storage_Value(current)
	}
	return PARSE_STATUS_OK
}

func template_names_equal(
	source Source,
	workspace_state Syntax_Workspace_Pointer, left_value Template_Reference_Validated,
	right_value Template_Reference_Validated,
) (equal Template_Name_Match) {
	defer func() {
		Template_Name_Match_Invariants(equal, "template_names_equal.equal")
	}()
	Syntax_Workspace_Pointer_Invariants(workspace_state, "template_names_equal.workspace_state")
	Template_Reference_Validated_Invariants(left_value, "template_names_equal.left_value")
	Template_Reference_Validated_Invariants(right_value, "template_names_equal.right_value")
	Source_Invariants(source, "template_names_equal.source")
	workspace := (Syntax_Workspace_Pointer)(workspace_state)
	left := left_value.Value.(Template_Reference_Validated_Value)
	right := right_value.Value.(Template_Reference_Validated_Value)
	Template_Name_Output_Invariants(
		Template_Name_Output(workspace.Template_Name_Left[:]),
		"template_names_equal.left_output",
	)
	left_node := workspace.Nodes[left-NODE_COUNT_INCREMENT]
	right_node := workspace.Nodes[right-NODE_COUNT_INCREMENT]
	left_span := Quoted_Span_Validated{Value: Quoted_Span_Stored(Quoted_Span{
		Start: Quoted_Start(left_node.Value_Start),
		End:   Quoted_End(left_node.Value_End),
	})}

	left_count := quoted_unquote_into(
		source, left_span, Decoded_Output(workspace.Template_Name_Left[:]),
	)
	right_span := Quoted_Span_Validated{Value: Quoted_Span_Stored(Quoted_Span{
		Start: Quoted_Start(right_node.Value_Start),
		End:   Quoted_End(right_node.Value_End),
	})}

	right_count := quoted_unquote_into(
		source, right_span, Decoded_Output(workspace.Template_Name_Right[:]),
	)
	if left_count != right_count {
		return false
	}
	for index := Template_Name_Count(SOURCE_SIZE_MINIMUM); index < left_count; index++ {
		if workspace.Template_Name_Left[index] != workspace.Template_Name_Right[index] {
			return false
		}
	}
	return true
}

func template_empty(
	source Source, workspace_state Syntax_Workspace_Pointer,
	reference_value Template_Reference_Validated,
) (empty Template_Empty) {
	defer func() { Template_Empty_Invariants(empty, "template_empty.empty") }()
	Syntax_Workspace_Pointer_Invariants(workspace_state, "template_empty.workspace_state")
	Template_Reference_Validated_Invariants(reference_value, "template_empty.reference_value")
	Source_Invariants(source, "template_empty.source")
	workspace := (Syntax_Workspace_Pointer)(workspace_state)
	reference := reference_value.Value.(Template_Reference_Validated_Value)
	definition := workspace.Nodes[reference-NODE_COUNT_INCREMENT]
	body := Node_Reference(definition.First_Child)
	child := Node_Reference(workspace.Nodes[body-NODE_COUNT_INCREMENT].First_Child)
	data := source.Data.Value
	for child != NO_NODE {
		node := workspace.Nodes[child-NODE_COUNT_INCREMENT]
		if node.Kind != NODE_COMMENT {
			if node.Kind != NODE_TEXT {
				return false
			}
			start := Source_Position(node.Start)
			end := Source_Position(node.End)
			for position := start; position < end; position++ {
				if !bool(space(Input_Byte(data[position]))) {
					return false
				}
			}
		}
		child = Node_Reference(node.Next_Sibling)
	}
	return true
}

func control_open(
	source Source,
	configuration Configuration,
	workspace_state Syntax_Workspace_Pointer, count_value Node_Count_Allocated,
	variables_state Variable_State_Pointer, list_state List_State_Storage_Pointer,
	controls_state Control_Frames_Pointer, depth_value Control_Depth_Tracked,
	cursor_state Token_Cursor_Pointer, span_state Action_Span_Validated,
) (_ Node_Count_Allocated, _ Control_Depth_Tracked, status Build_Status) {
	defer func() { Build_Status_Invariants(status, "control_open.status") }()
	Syntax_Workspace_Pointer_Invariants(workspace_state, "control_open.workspace_state")
	Node_Count_Allocated_Invariants(count_value, "control_open.count_value")
	Variable_State_Pointer_Invariants(variables_state, "control_open.variables_state")
	List_State_Storage_Pointer_Invariants(list_state, "control_open.list_state")
	Control_Frames_Pointer_Invariants(controls_state, "control_open.controls_state")
	Control_Depth_Tracked_Invariants(depth_value, "control_open.depth_value")
	Token_Cursor_Pointer_Invariants(cursor_state, "control_open.cursor_state")
	Action_Span_Validated_Invariants(span_state, "control_open.span_state")
	Source_Invariants(source, "control_open.source")
	Configuration_Invariants(configuration, "control_open.configuration")
	count, depth := count_value, depth_value
	defer func() { Node_Count_Allocated_Invariants(count, "control_open.count") }()
	defer func() { Control_Depth_Tracked_Invariants(depth, "control_open.depth") }()
	span := span_state.Value
	action_end := span.End
	variables := (Variable_State_Pointer)(variables_state)
	list := &list_state.Value
	controls := (Control_Frames_Pointer)(controls_state)
	keyword_state := control_header(source, cursor_state, depth)
	kind := control_node_kind(Control_Token_Kind(keyword_state.Value.Kind))
	parent_variable_count := variables.Count.Value.(Syntax_Variable_Count_Storage_Value)
	branch_value, count, allocation_status := control_branch_allocate(
		workspace_state, count, keyword_state, span_state,
	)
	if allocation_status != PARSE_STATUS_OK {
		return count, depth, Build_Status(allocation_status)
	}
	branch := branch_value.Value.(Node_Link_Validated_Value)
	pipe, count, status := control_open_pipeline(
		source, configuration, workspace_state, count, variables, cursor_state, kind,
	)
	if status != PARSE_STATUS_OK {
		return count, depth, status
	}
	body, updated_count, allocation_status := allocate_node(
		workspace_state, count, Node_Allocation{Value: Node(Node{
			Kind: NODE_LIST, Start: Node_Start(action_end), End: Node_End(action_end),
		})},
	)
	count = updated_count
	if allocation_status != PARSE_STATUS_OK {
		return count, depth, Build_Status(allocation_status)
	}
	control_branch_children(
		workspace_state, branch_value, pipe,
		Node_Link_Validated{Value: Node_Link_Validated_Value(body)},
	)
	append_list_child(workspace_state, list_state, branch_value)
	depth_scalar := depth.Value.(Control_Depth_Tracked_Value)
	(*controls)[depth_scalar].Value = Control_Frame_Stored(Control_Frame{
		Branch:                Control_Branch(branch),
		Parent_List:           Control_Parent_List(list.List),
		Parent_Last_Child:     Control_Parent_Last_Child(list.Last_Child),
		Parent_Variable_Count: Control_Parent_Variable_Count(parent_variable_count),
		Branch_Variable_Count: Control_Branch_Variable_Count(
			variables.Count.Value.(Syntax_Variable_Count_Storage_Value),
		),
		Kind: Node_Kind(kind),
	})
	depth.Value = depth_scalar + CONTROL_DEPTH_INCREMENT
	*list = List_State_Stored(List_State{List: List_Reference(body)})
	return count, depth, PARSE_STATUS_OK
}

func control_branch_children(
	workspace_state Syntax_Workspace_Pointer, branch_value Node_Link_Validated,
	pipe Pipeline_Root_Validated, body_value Node_Link_Validated,
) {
	Syntax_Workspace_Pointer_Invariants(
		workspace_state, "control_branch_children.workspace_state",
	)
	Node_Link_Validated_Invariants(branch_value, "control_branch_children.branch_value")
	Pipeline_Root_Validated_Invariants(pipe, "control_branch_children.pipe")
	Node_Link_Validated_Invariants(body_value, "control_branch_children.body_value")
	branch := branch_value.Value.(Node_Link_Validated_Value)
	last := Node_Link_Validated{Value: Node_Link_Validated_Value(NO_NODE)}
	last = append_child(
		workspace_state, Node_Link_Validated{Value: Node_Link_Validated_Value(branch)},
		last,
		Node_Link_Validated{Value: Node_Link_Validated_Value(
			pipe.Value.(Pipeline_Root_Validated_Value),
		)},
	)
	append_child(
		workspace_state, Node_Link_Validated{Value: Node_Link_Validated_Value(branch)},
		last, body_value,
	)
}

func control_header(
	source Source, cursor_state Token_Cursor_Pointer, depth_state Control_Depth_Tracked,
) (keyword Token_Validated) {
	defer func() {
		Token_Validated_Invariants(keyword, "control_header.keyword")
	}()
	Source_Invariants(source, "control_header.source")
	Token_Cursor_Pointer_Invariants(cursor_state, "control_header.cursor_state")
	Control_Depth_Tracked_Invariants(depth_state, "control_header.depth_state")
	cursor := (Token_Cursor_Pointer)(cursor_state)
	depth := depth_state.Value.(Control_Depth_Tracked_Value)
	peek_status := token_peek(source, cursor)
	aver.Always(
		peek_status == PARSE_STATUS_OK,
		"Control dispatch retains one lexer-validated keyword.",
	)
	keyword = Token_Validated{Value: Token_Validated_Value(cursor.Token.Value)}
	aver.Always(
		Control_Depth(depth) < Control_Depth(CONTROL_DEPTH_MAXIMUM),
		"Source density leaves one frame for every reachable control opener.",
	)
	take_status := token_take(source, cursor)
	aver.Always(
		take_status == PARSE_STATUS_OK,
		"Retained control keyword remains consumable.",
	)
	return keyword
}

func control_branch_allocate(
	workspace_state Syntax_Workspace_Pointer, count_value Node_Count_Allocated,
	keyword_state Token_Validated,
	span_state Action_Span_Validated,
) (
	_ Node_Link_Validated,
	_ Node_Count_Allocated,
	status Capacity_Status,
) {
	defer func() { Capacity_Status_Invariants(status, "control_branch_allocate.status") }()
	Syntax_Workspace_Pointer_Invariants(
		workspace_state, "control_branch_allocate.workspace_state",
	)
	Node_Count_Allocated_Invariants(count_value, "control_branch_allocate.count_value")
	Token_Validated_Invariants(keyword_state, "control_branch_allocate.keyword_state")
	Action_Span_Validated_Invariants(span_state, "control_branch_allocate.span_state")
	var branch Node_Link_Validated
	defer func() { Node_Link_Validated_Invariants(branch, "control_branch_allocate.branch") }()
	count := count_value
	defer func() { Node_Count_Allocated_Invariants(count, "control_branch_allocate.count") }()
	keyword := keyword_state.Value
	span := span_state.Value
	reference, updated_count, allocation_status := allocate_node(
		workspace_state, count_value, Node_Allocation{Value: Node(Node{
			Kind:  Node_Kind(control_node_kind(Control_Token_Kind(keyword.Kind))),
			Start: Node_Start(span.Start), End: Node_End(span.End),
			Value_Start: Node_Value_Start(keyword.Start),
			Value_End:   Node_Value_End(keyword.End),
		})})
	count, status = updated_count, allocation_status
	branch = Node_Link_Validated{Value: Node_Link_Validated_Value(reference)}
	return branch, count, status
}

func control_open_pipeline(
	source Source, configuration Configuration,
	workspace_state Syntax_Workspace_Pointer, count_value Node_Count_Allocated,
	variables_state Variable_State_Pointer, cursor_state Token_Cursor_Pointer,
	kind Control_Node_Kind,
) (_ Pipeline_Root_Validated, _ Node_Count_Allocated, status Build_Status) {
	defer func() { Build_Status_Invariants(status, "control_open_pipeline.status") }()
	Syntax_Workspace_Pointer_Invariants(
		workspace_state, "control_open_pipeline.workspace_state",
	)
	Node_Count_Allocated_Invariants(count_value, "control_open_pipeline.count_value")
	Variable_State_Pointer_Invariants(variables_state, "control_open_pipeline.variables_state")
	Token_Cursor_Pointer_Invariants(cursor_state, "control_open_pipeline.cursor_state")
	Control_Node_Kind_Invariants(kind, "control_open_pipeline.kind")
	Source_Invariants(source, "control_open_pipeline.source")
	Configuration_Invariants(configuration, "control_open_pipeline.configuration")
	count := count_value
	defer func() { Node_Count_Allocated_Invariants(count, "control_open_pipeline.count") }()
	pipe := Pipeline_Root_Validated{Value: Pipeline_Root_Validated_Value(NO_NODE)}
	defer func() { Pipeline_Root_Validated_Invariants(pipe, "control_open_pipeline.pipe") }()
	variables := (Variable_State_Pointer)(variables_state)
	workspace := (Syntax_Workspace_Pointer)(workspace_state)
	cursor := (Token_Cursor_Pointer)(cursor_state)
	declaration_maximum := Pipeline_Declaration_Count(
		PIPELINE_DECLARATION_COUNT_MINIMUM,
	)
	if Node_Kind(kind) == NODE_RANGE {
		declaration_maximum = PIPELINE_DECLARATION_COUNT_MAXIMUM
	}
	pipe, count, status = parse_pipeline(
		source, configuration, workspace, count, variables,
		declaration_maximum, cursor,
	)
	if status != PARSE_STATUS_OK {
		return pipe, count, status
	}
	status = Build_Status(token_take(source, cursor))
	if status != PARSE_STATUS_OK {
		return pipe, count, status
	}
	boundary := Token(cursor.Token.Value)
	if boundary.Kind != TOKEN_EOF {
		return pipe, count, PARSE_STATUS_SYNTAX_INVALID
	}
	return pipe, count, PARSE_STATUS_OK
}

func control_else(
	source Source, configuration Configuration,
	workspace_state Syntax_Workspace_Pointer, count_value Node_Count_Allocated,
	variables_state Variable_State_Pointer, list_state List_State_Storage_Pointer,
	controls_state Control_Frames_Pointer, depth_value Control_Depth_Tracked,
	cursor_state Token_Cursor_Pointer, span_state Action_Span_Validated,
) (_ Node_Count_Allocated, _ Control_Depth_Tracked, status Build_Status) {
	defer func() { Build_Status_Invariants(status, "control_else.status") }()
	Syntax_Workspace_Pointer_Invariants(workspace_state, "control_else.workspace_state")
	Node_Count_Allocated_Invariants(count_value, "control_else.count_value")
	Variable_State_Pointer_Invariants(variables_state, "control_else.variables_state")
	List_State_Storage_Pointer_Invariants(list_state, "control_else.list_state")
	Control_Frames_Pointer_Invariants(controls_state, "control_else.controls_state")
	Control_Depth_Tracked_Invariants(depth_value, "control_else.depth_value")
	Token_Cursor_Pointer_Invariants(cursor_state, "control_else.cursor_state")
	Action_Span_Validated_Invariants(span_state, "control_else.span_state")
	Source_Invariants(source, "control_else.source")
	Configuration_Invariants(configuration, "control_else.configuration")
	count, depth := count_value, depth_value
	defer func() { Node_Count_Allocated_Invariants(count, "control_else.count") }()
	defer func() { Control_Depth_Tracked_Invariants(depth, "control_else.depth") }()
	variables := (Variable_State_Pointer)(variables_state)
	controls := (Control_Frames_Pointer)(controls_state)
	cursor := (Token_Cursor_Pointer)(cursor_state)
	depth_scalar := depth.Value.(Control_Depth_Tracked_Value)
	if depth_scalar == CONTROL_DEPTH_MINIMUM {
		return count, depth, PARSE_STATUS_SYNTAX_INVALID
	}
	frame_state := &(*controls)[depth_scalar-CONTROL_DEPTH_INCREMENT]
	if frame_state.Value.Kind == NODE_TEMPLATE {
		return count, depth, PARSE_STATUS_SYNTAX_INVALID
	}
	if bool(frame_state.Value.Has_Else) {
		return count, depth, PARSE_STATUS_SYNTAX_INVALID
	}
	form, lex_status := control_else_next(source, cursor)
	if lex_status != PARSE_STATUS_OK {
		return count, depth, Build_Status(lex_status)
	}
	switch form {
	case CONTROL_ELSE_FORM_LIST:
		status = Build_Status(token_take(source, cursor))
		if status != PARSE_STATUS_OK {
			return count, depth, status
		}
		var list_status Capacity_Status
		count, list_status = control_else_list(
			workspace_state, count, variables, list_state, frame_state, span_state,
		)
		return count, depth, Build_Status(list_status)
	case CONTROL_ELSE_FORM_IF, CONTROL_ELSE_FORM_WITH:
		var list_status Capacity_Status
		count, list_status = control_else_list(
			workspace_state, count, variables, list_state, frame_state, span_state,
		)
		status = Build_Status(list_status)
		if status != PARSE_STATUS_OK {
			return count, depth, status
		}
		count, depth, status = control_open(
			source, configuration, workspace_state, count, variables,
			list_state, controls, depth, cursor, span_state,
		)
		if status != PARSE_STATUS_OK {
			return count, depth, status
		}
		depth_scalar = depth.Value.(Control_Depth_Tracked_Value)
		frame_state := &(*controls)[depth_scalar-CONTROL_DEPTH_INCREMENT]
		frame_state.Value.Shares_End = true
		return count, depth, PARSE_STATUS_OK
	default:
		return count, depth, PARSE_STATUS_SYNTAX_INVALID
	}
}

func control_else_next(
	source Source, cursor_state Token_Cursor_Pointer,
) (_ Control_Else_Form, status Lex_Status) {
	defer func() { Lex_Status_Invariants(status, "control_else_next.status") }()
	Token_Cursor_Pointer_Invariants(cursor_state, "control_else_next.cursor_state")
	Source_Invariants(source, "control_else_next.source")
	form := CONTROL_ELSE_FORM_INVALID
	defer func() { Control_Else_Form_Invariants(form, "control_else_next.form") }()
	cursor := (Token_Cursor_Pointer)(cursor_state)
	status = token_take(source, cursor)
	if status != PARSE_STATUS_OK {
		return form, status
	}
	keyword := Token(cursor.Token.Value)
	if keyword.Kind != TOKEN_ELSE {
		return form, PARSE_STATUS_SYNTAX_INVALID
	}
	status = token_peek(source, cursor)
	if status != PARSE_STATUS_OK {
		return form, status
	}
	next := Token(cursor.Token.Value)
	switch next.Kind {
	case TOKEN_EOF:
		form = CONTROL_ELSE_FORM_LIST
	case TOKEN_IF:
		form = CONTROL_ELSE_FORM_IF
	case TOKEN_WITH:
		form = CONTROL_ELSE_FORM_WITH
	}
	return form, PARSE_STATUS_OK
}

func control_else_list(
	workspace_state Syntax_Workspace_Pointer, count_value Node_Count_Allocated,
	variables_state Variable_State_Pointer, list_state List_State_Storage_Pointer,
	frame_state Control_Frame_Storage_Pointer, span_state Action_Span_Validated,
) (_ Node_Count_Allocated, status Capacity_Status) {
	defer func() { Capacity_Status_Invariants(status, "control_else_list.status") }()
	Syntax_Workspace_Pointer_Invariants(workspace_state, "control_else_list.workspace_state")
	Node_Count_Allocated_Invariants(count_value, "control_else_list.count_value")
	Variable_State_Pointer_Invariants(variables_state, "control_else_list.variables_state")
	List_State_Storage_Pointer_Invariants(list_state, "control_else_list.list_state")
	Control_Frame_Storage_Pointer_Invariants(frame_state, "control_else_list.frame_state")
	Action_Span_Validated_Invariants(span_state, "control_else_list.span_state")
	count := count_value
	defer func() { Node_Count_Allocated_Invariants(count, "control_else_list.count") }()
	span := span_state.Value
	action_start, action_end := span.Start, span.End
	frame := &frame_state.Value
	variables := (Variable_State_Pointer)(variables_state)
	list := &list_state.Value
	workspace := (Syntax_Workspace_Pointer)(workspace_state)
	workspace.Nodes[list.List-NODE_COUNT_INCREMENT].End = Node_End(action_start)
	else_list, updated_count, allocation_status := allocate_node(
		workspace, count, Node_Allocation{Value: Node(Node{
			Kind: NODE_LIST, Start: Node_Start(action_end), End: Node_End(action_end),
		})},
	)
	count = updated_count
	if allocation_status != PARSE_STATUS_OK {
		return count, Capacity_Status(allocation_status)
	}
	pipe := Node_Reference(workspace.Nodes[frame.Branch-NODE_COUNT_INCREMENT].First_Child)
	body := Node_Reference(workspace.Nodes[pipe-NODE_COUNT_INCREMENT].Next_Sibling)
	workspace.Nodes[body-NODE_COUNT_INCREMENT].Next_Sibling = Node_Next_Sibling(else_list)
	frame.Has_Else = true
	variables.Count.Value = Syntax_Variable_Count_Storage_Value(
		frame.Branch_Variable_Count,
	)
	*list = List_State_Stored(List_State{List: List_Reference(else_list)})
	return count, PARSE_STATUS_OK
}

func control_end(
	source Source,
	workspace_state Syntax_Workspace_Pointer, variables_state Variable_State_Pointer,
	list_state List_State_Storage_Pointer, controls_state Control_Frames_Pointer,
	depth_value Control_Depth_Tracked,
	cursor_state Token_Cursor_Pointer, span_state Action_Span_Validated,
) (_ Control_Depth_Tracked, status Lex_Status) {
	defer func() { Lex_Status_Invariants(status, "control_end.status") }()
	Syntax_Workspace_Pointer_Invariants(workspace_state, "control_end.workspace_state")
	Variable_State_Pointer_Invariants(variables_state, "control_end.variables_state")
	List_State_Storage_Pointer_Invariants(list_state, "control_end.list_state")
	Control_Frames_Pointer_Invariants(controls_state, "control_end.controls_state")
	Control_Depth_Tracked_Invariants(depth_value, "control_end.depth_value")
	Token_Cursor_Pointer_Invariants(cursor_state, "control_end.cursor_state")
	Action_Span_Validated_Invariants(span_state, "control_end.span_state")
	Source_Invariants(source, "control_end.source")
	depth := depth_value
	defer func() { Control_Depth_Tracked_Invariants(depth, "control_end.depth") }()
	span := span_state.Value
	action_start, action_end := span.Start, span.End
	variables := (Variable_State_Pointer)(variables_state)
	list := &list_state.Value
	controls := (Control_Frames_Pointer)(controls_state)
	workspace := (Syntax_Workspace_Pointer)(workspace_state)
	cursor := (Token_Cursor_Pointer)(cursor_state)
	depth_scalar := depth.Value.(Control_Depth_Tracked_Value)
	if depth_scalar == CONTROL_DEPTH_MINIMUM {
		return depth, PARSE_STATUS_SYNTAX_INVALID
	}
	status = token_take(source, cursor)
	if status != PARSE_STATUS_OK {
		return depth, status
	}
	keyword := Token(cursor.Token.Value)
	if keyword.Kind != TOKEN_END {
		return depth, PARSE_STATUS_SYNTAX_INVALID
	}
	status = token_take(source, cursor)
	if status != PARSE_STATUS_OK {
		return depth, status
	}
	boundary := Token(cursor.Token.Value)
	if boundary.Kind != TOKEN_EOF {
		return depth, PARSE_STATUS_SYNTAX_INVALID
	}
	workspace.Nodes[list.List-NODE_COUNT_INCREMENT].End = Node_End(action_start)
	for depth_scalar > CONTROL_DEPTH_MINIMUM {
		depth_scalar--
		depth.Value = depth_scalar
		frame := (*controls)[depth_scalar].Value
		workspace.Nodes[frame.Branch-NODE_COUNT_INCREMENT].End = Node_End(action_end)
		variables.Count.Value = Syntax_Variable_Count_Storage_Value(
			frame.Parent_Variable_Count,
		)
		*list = List_State_Stored(List_State{
			List:       List_Reference(frame.Parent_List),
			Last_Child: List_Last_Child(frame.Parent_Last_Child),
		})
		if !bool(frame.Shares_End) {
			return depth, PARSE_STATUS_OK
		}
		workspace.Nodes[list.List-NODE_COUNT_INCREMENT].End = Node_End(action_start)
	}
	return depth, PARSE_STATUS_SYNTAX_INVALID
}

func control_flow(
	source Source,
	workspace_state Syntax_Workspace_Pointer, count_value Node_Count_Allocated,
	list_state List_State_Storage_Pointer, controls_state Control_Frames_Pointer,
	depth Control_Depth_Tracked,
	cursor_state Token_Cursor_Pointer, span_state Action_Span_Validated,
) (_ Node_Count_Allocated, status Build_Status) {
	defer func() { Build_Status_Invariants(status, "control_flow.status") }()
	Syntax_Workspace_Pointer_Invariants(workspace_state, "control_flow.workspace_state")
	Node_Count_Allocated_Invariants(count_value, "control_flow.count_value")
	List_State_Storage_Pointer_Invariants(list_state, "control_flow.list_state")
	Control_Frames_Pointer_Invariants(controls_state, "control_flow.controls_state")
	Control_Depth_Tracked_Invariants(depth, "control_flow.depth")
	Token_Cursor_Pointer_Invariants(cursor_state, "control_flow.cursor_state")
	Action_Span_Validated_Invariants(span_state, "control_flow.span_state")
	Source_Invariants(source, "control_flow.source")
	count := count_value
	defer func() { Node_Count_Allocated_Invariants(count, "control_flow.count") }()
	span := span_state.Value
	action_start, action_end := span.Start, span.End
	controls := (Control_Frames_Pointer)(controls_state)
	workspace := (Syntax_Workspace_Pointer)(workspace_state)
	cursor := (Token_Cursor_Pointer)(cursor_state)
	status = Build_Status(token_take(source, cursor))
	if status != PARSE_STATUS_OK {
		return count, status
	}
	token := Token(cursor.Token.Value)
	if !bool(control_range(controls, depth)) {
		return count, PARSE_STATUS_SYNTAX_INVALID
	}
	status = Build_Status(token_take(source, cursor))
	if status != PARSE_STATUS_OK {
		return count, status
	}
	boundary := Token(cursor.Token.Value)
	if boundary.Kind != TOKEN_EOF {
		return count, PARSE_STATUS_SYNTAX_INVALID
	}
	kind := NODE_BREAK
	if token.Kind == TOKEN_CONTINUE {
		kind = NODE_CONTINUE
	}
	reference, updated_count, allocation_status := allocate_node(
		workspace, count, Node_Allocation{Value: Node(Node{
			Kind: kind, Start: Node_Start(action_start), End: Node_End(action_end),
			Value_Start: Node_Value_Start(token.Start),
			Value_End:   Node_Value_End(token.End),
		})},
	)
	count = updated_count
	if allocation_status != PARSE_STATUS_OK {
		return count, Build_Status(allocation_status)
	}
	append_list_child(
		workspace, list_state,
		Node_Link_Validated{Value: Node_Link_Validated_Value(reference)},
	)
	return count, PARSE_STATUS_OK
}

func control_range(
	controls_state Control_Frames_Pointer, depth_state Control_Depth_Tracked,
) (inside Range_Scope) {
	defer func() { Range_Scope_Invariants(inside, "control_range.inside") }()
	Control_Frames_Pointer_Invariants(controls_state, "control_range.controls_state")
	Control_Depth_Tracked_Invariants(depth_state, "control_range.depth_state")
	controls := (Control_Frames_Pointer)(controls_state)
	depth := depth_state.Value.(Control_Depth_Tracked_Value)
	for index := int(depth); index > CONTROL_DEPTH_MINIMUM; index-- {
		frame := (*controls)[index-CONTROL_DEPTH_INCREMENT].Value
		if frame.Kind == NODE_RANGE {
			if !bool(frame.Has_Else) {
				return true
			}
		}
	}
	return false
}

func control_node_kind(kind Control_Token_Kind) (node_kind Control_Node_Kind) {
	defer func() {
		Control_Node_Kind_Invariants(node_kind, "control_node_kind.node_kind")
	}()
	Control_Token_Kind_Invariants(kind, "control_node_kind.kind")
	switch kind {
	case Control_Token_Kind(TOKEN_IF):
		return Control_Node_Kind(NODE_IF)
	case Control_Token_Kind(TOKEN_RANGE):
		return Control_Node_Kind(NODE_RANGE)
	}
	aver.Always(
		kind == Control_Token_Kind(TOKEN_WITH),
		"Action dispatch permits only branch-opening keywords.",
	)
	return Control_Node_Kind(NODE_WITH)
}

func parse_pipeline(
	source Source,
	configuration Configuration,
	workspace_state Syntax_Workspace_Pointer, count_value Node_Count_Allocated,
	variables_state Variable_State_Pointer, declaration_maximum Pipeline_Declaration_Count,
	cursor_state Token_Cursor_Pointer,
) (_ Pipeline_Root_Validated, _ Node_Count_Allocated, status Build_Status) {
	defer func() { Build_Status_Invariants(status, "parse_pipeline.status") }()
	Syntax_Workspace_Pointer_Invariants(workspace_state, "parse_pipeline.workspace_state")
	Node_Count_Allocated_Invariants(count_value, "parse_pipeline.count_value")
	Variable_State_Pointer_Invariants(variables_state, "parse_pipeline.variables_state")
	Token_Cursor_Pointer_Invariants(cursor_state, "parse_pipeline.cursor_state")
	Source_Invariants(source, "parse_pipeline.source")
	Configuration_Invariants(configuration, "parse_pipeline.configuration")
	Pipeline_Declaration_Count_Invariants(
		declaration_maximum, "parse_pipeline.declaration_maximum",
	)
	count := count_value
	defer func() { Node_Count_Allocated_Invariants(count, "parse_pipeline.count") }()
	pipe := Pipeline_Root_Validated{Value: Pipeline_Root_Validated_Value(NO_NODE)}
	defer func() { Pipeline_Root_Validated_Invariants(pipe, "parse_pipeline.pipe") }()
	variables := (Variable_State_Pointer)(variables_state)
	workspace := (Syntax_Workspace_Pointer)(workspace_state)
	cursor := (Token_Cursor_Pointer)(cursor_state)
	status = Build_Status(token_peek(source, cursor))
	if status != PARSE_STATUS_OK {
		return pipe, count, status
	}
	first := Token(cursor.Token.Value)
	if first.Kind == TOKEN_EOF {
		return pipe, count, PARSE_STATUS_SYNTAX_INVALID
	}
	if first.Kind == TOKEN_RIGHT_PAREN {
		return pipe, count, PARSE_STATUS_SYNTAX_INVALID
	}
	allocated_pipe, updated_count, allocation_status := allocate_node(
		workspace, count, Node_Allocation{Value: Node(Node{
			Kind: NODE_PIPE, Start: Node_Start(first.Start),
			End:         Node_End(cursor.End.Value),
			Value_Start: Node_Value_Start(first.Start),
			Value_End:   Node_Value_End(cursor.End.Value),
		})},
	)
	count = updated_count
	if allocation_status != PARSE_STATUS_OK {
		return pipe, count, Build_Status(allocation_status)
	}
	pipe = Pipeline_Root_Validated{Value: Pipeline_Root_Validated_Value(allocated_pipe)}
	count, status = parse_pipeline_tokens(
		source, configuration, workspace, count, variables,
		declaration_maximum, cursor, pipe,
	)
	return pipe, count, status
}

func parse_pipeline_tokens(
	source Source, configuration Configuration,
	workspace_state Syntax_Workspace_Pointer, count_value Node_Count_Allocated,
	variables_state Variable_State_Pointer, declaration_maximum Pipeline_Declaration_Count,
	cursor_state Token_Cursor_Pointer, root_state Pipeline_Root_Validated,
) (_ Node_Count_Allocated, status Build_Status) {
	defer func() { Build_Status_Invariants(status, "parse_pipeline_tokens.status") }()
	Syntax_Workspace_Pointer_Invariants(
		workspace_state, "parse_pipeline_tokens.workspace_state",
	)
	Node_Count_Allocated_Invariants(count_value, "parse_pipeline_tokens.count_value")
	Variable_State_Pointer_Invariants(variables_state, "parse_pipeline_tokens.variables_state")
	Token_Cursor_Pointer_Invariants(cursor_state, "parse_pipeline_tokens.cursor_state")
	Pipeline_Root_Validated_Invariants(root_state, "parse_pipeline_tokens.root_state")
	Source_Invariants(source, "parse_pipeline_tokens.source")
	Configuration_Invariants(configuration, "parse_pipeline_tokens.configuration")
	Pipeline_Declaration_Count_Invariants(
		declaration_maximum, "parse_pipeline_tokens.declaration_maximum",
	)
	count, variables := count_value, Variable_State_Pointer(variables_state)
	defer func() { Node_Count_Allocated_Invariants(count, "parse_pipeline_tokens.count") }()
	workspace, cursor := Syntax_Workspace_Pointer(workspace_state),
		Token_Cursor_Pointer(cursor_state)
	parent_storage := [PARENTHESIS_DEPTH_MAXIMUM]Pipeline_Frame_Storage{}
	parents := Pipeline_Frames(parent_storage[:])
	Pipeline_Frames_Invariants(parents, "parse_pipeline_tokens.parent_frames")
	depth := Parenthesis_Depth_Tracked{Value: PARENTHESIS_DEPTH_INITIAL}
	frame := Pipeline_Frame_Storage{Value: Pipeline_Frame_Stored(Pipeline_Frame{
		Pipe: Pipeline_Reference(root_state.Value.(Pipeline_Root_Validated_Value)),
	})}
	for Source_Position(cursor.Position.Value) <=
		Source_Position(cursor.End.Value) {
		if token_status := token_peek(source, cursor); token_status != PARSE_STATUS_OK {
			return count, Build_Status(token_status)
		}
		switch cursor.Token.Value.Kind {
		case TOKEN_EOF:
			if !bool(pipeline_complete(source, cursor, depth, &frame)) {
				return count, PARSE_STATUS_SYNTAX_INVALID
			}
			return count, PARSE_STATUS_OK
		case TOKEN_PIPE:
			if !bool(pipeline_separated(source, cursor, &frame)) {
				return count, PARSE_STATUS_SYNTAX_INVALID
			}
		case TOKEN_LEFT_PAREN:
			var open_status Capacity_Status
			count, depth, open_status = pipeline_open(
				source, workspace, count, cursor, &parents, depth, &frame,
			)
			status = Build_Status(open_status)
		case TOKEN_RIGHT_PAREN:
			count, depth, status = pipeline_close(
				source, workspace, count, cursor, &parents, depth, &frame,
			)
		case TOKEN_VARIABLE:
			count, status = pipeline_variable(
				source, configuration, workspace, count,
				variables, declaration_maximum, cursor, &frame,
			)
		default:
			count, status = pipeline_scalar(
				source, configuration, workspace, count,
				variables, cursor, &frame,
			)
		}
		if status != PARSE_STATUS_OK {
			return count, status
		}
	}
	return count, PARSE_STATUS_SYNTAX_INVALID
}

func pipeline_complete(
	source Source, cursor_state Token_Cursor_Pointer, depth_state Parenthesis_Depth_Tracked,
	frame_state Pipeline_Frame_Storage_Pointer) (valid Pipeline_Boundary_Validity) {
	defer func() {
		Pipeline_Boundary_Validity_Invariants(valid, "pipeline_complete.valid")
	}()
	Token_Cursor_Pointer_Invariants(cursor_state, "pipeline_complete.cursor_state")
	Parenthesis_Depth_Tracked_Invariants(depth_state, "pipeline_complete.depth_state")
	Pipeline_Frame_Storage_Pointer_Invariants(frame_state, "pipeline_complete.frame_state")
	Source_Invariants(source, "pipeline_complete.source")
	cursor := (Token_Cursor_Pointer)(cursor_state)
	frame := &frame_state.Value
	depth := depth_state.Value.(Parenthesis_Depth_Tracked_Value)
	status := token_take(source, cursor)
	if status != PARSE_STATUS_OK {
		return false
	}
	if depth != PARENTHESIS_DEPTH_MINIMUM {
		return false
	}
	return Pipeline_Boundary_Validity(frame.Command != Pipeline_Command(NO_NODE))
}

func pipeline_separated(
	source Source, cursor_state Token_Cursor_Pointer,
	frame_state Pipeline_Frame_Storage_Pointer,
) (valid Pipeline_Boundary_Validity) {
	defer func() {
		Pipeline_Boundary_Validity_Invariants(valid, "pipeline_separated.valid")
	}()
	Token_Cursor_Pointer_Invariants(cursor_state, "pipeline_separated.cursor_state")
	Pipeline_Frame_Storage_Pointer_Invariants(frame_state, "pipeline_separated.frame_state")
	Source_Invariants(source, "pipeline_separated.source")
	cursor := (Token_Cursor_Pointer)(cursor_state)
	frame := &frame_state.Value
	status := token_take(source, cursor)
	if status != PARSE_STATUS_OK {
		return false
	}
	if frame.Command == Pipeline_Command(NO_NODE) {
		return false
	}
	frame.Command = Pipeline_Command(NO_NODE)
	frame.Last_Argument = Pipeline_Last_Argument(NO_NODE)
	return true
}

func pipeline_open(
	source Source,
	workspace_state Syntax_Workspace_Pointer, count_value Node_Count_Allocated,
	cursor_state Token_Cursor_Pointer, parents_state Pipeline_Frames_Pointer,
	depth_value Parenthesis_Depth_Tracked,
	frame_state Pipeline_Frame_Storage_Pointer,
) (_ Node_Count_Allocated, _ Parenthesis_Depth_Tracked, status Capacity_Status) {
	defer func() { Capacity_Status_Invariants(status, "pipeline_open.status") }()
	Syntax_Workspace_Pointer_Invariants(workspace_state, "pipeline_open.workspace_state")
	Node_Count_Allocated_Invariants(count_value, "pipeline_open.count_value")
	Token_Cursor_Pointer_Invariants(cursor_state, "pipeline_open.cursor_state")
	Pipeline_Frames_Pointer_Invariants(parents_state, "pipeline_open.parents_state")
	Parenthesis_Depth_Tracked_Invariants(depth_value, "pipeline_open.depth_value")
	Pipeline_Frame_Storage_Pointer_Invariants(frame_state, "pipeline_open.frame_state")
	Source_Invariants(source, "pipeline_open.source")
	count, depth := count_value, depth_value
	defer func() { Node_Count_Allocated_Invariants(count, "pipeline_open.count") }()
	defer func() { Parenthesis_Depth_Tracked_Invariants(depth, "pipeline_open.depth") }()
	workspace := (Syntax_Workspace_Pointer)(workspace_state)
	cursor := (Token_Cursor_Pointer)(cursor_state)
	frame := &frame_state.Value
	parents := (Pipeline_Frames_Pointer)(parents_state)
	lex_status := token_take(source, cursor)
	aver.Always(
		lex_status == PARSE_STATUS_OK,
		"Pipeline opener was retained by look-ahead.",
	)
	token := Token(cursor.Token.Value)
	depth_scalar := depth.Value.(Parenthesis_Depth_Tracked_Value)
	if Parenthesis_Depth(depth_scalar) == Parenthesis_Depth(PARENTHESIS_DEPTH_MAXIMUM) {
		return count, depth, PARSE_STATUS_CAPACITY_EXCEEDED
	}
	(*parents)[depth_scalar].Value = *frame
	depth.Value = depth_scalar + PARENTHESIS_DEPTH_INCREMENT
	pipe, updated_count, status := allocate_node(
		workspace, count, Node_Allocation{Value: Node(Node{
			Kind: NODE_PIPE, Start: Node_Start(token.Start),
			End:         Node_End(token.End),
			Value_Start: Node_Value_Start(token.Start),
			Value_End:   Node_Value_End(token.End),
		})})
	count = updated_count
	if status != PARSE_STATUS_OK {
		return count, depth, status
	}
	*frame = Pipeline_Frame_Stored(Pipeline_Frame{Pipe: Pipeline_Reference(pipe)})
	return count, depth, PARSE_STATUS_OK
}

func pipeline_close(
	source Source,
	workspace_state Syntax_Workspace_Pointer, count_value Node_Count_Allocated,
	cursor_state Token_Cursor_Pointer, parents_state Pipeline_Frames_Pointer,
	depth_value Parenthesis_Depth_Tracked,
	frame_state Pipeline_Frame_Storage_Pointer,
) (_ Node_Count_Allocated, _ Parenthesis_Depth_Tracked, status Build_Status) {
	defer func() { Build_Status_Invariants(status, "pipeline_close.status") }()
	Syntax_Workspace_Pointer_Invariants(workspace_state, "pipeline_close.workspace_state")
	Node_Count_Allocated_Invariants(count_value, "pipeline_close.count_value")
	Token_Cursor_Pointer_Invariants(cursor_state, "pipeline_close.cursor_state")
	Pipeline_Frames_Pointer_Invariants(parents_state, "pipeline_close.parents_state")
	Parenthesis_Depth_Tracked_Invariants(depth_value, "pipeline_close.depth_value")
	Pipeline_Frame_Storage_Pointer_Invariants(frame_state, "pipeline_close.frame_state")
	Source_Invariants(source, "pipeline_close.source")
	count, depth := count_value, depth_value
	defer func() { Node_Count_Allocated_Invariants(count, "pipeline_close.count") }()
	defer func() { Parenthesis_Depth_Tracked_Invariants(depth, "pipeline_close.depth") }()
	workspace := (Syntax_Workspace_Pointer)(workspace_state)
	cursor := (Token_Cursor_Pointer)(cursor_state)
	frame := &frame_state.Value
	parents := (Pipeline_Frames_Pointer)(parents_state)
	status = Build_Status(token_take(source, cursor))
	if status != PARSE_STATUS_OK {
		return count, depth, status
	}
	token := Token(cursor.Token.Value)
	depth_scalar := depth.Value.(Parenthesis_Depth_Tracked_Value)
	if depth_scalar == PARENTHESIS_DEPTH_MINIMUM {
		return count, depth, PARSE_STATUS_SYNTAX_INVALID
	}
	if frame.Command == Pipeline_Command(NO_NODE) {
		return count, depth, PARSE_STATUS_SYNTAX_INVALID
	}
	workspace.Nodes[frame.Pipe-NODE_COUNT_INCREMENT].End = Node_End(token.End)
	workspace.Nodes[frame.Pipe-NODE_COUNT_INCREMENT].Value_End = Node_Value_End(token.End)
	term := Pipeline_Term_Validated{
		Value: Pipeline_Term_Validated_Value(Node_Reference(frame.Pipe)),
	}
	term, count, status = pipeline_chain(
		source, workspace, count, cursor, term,
		Pipeline_Boundary_Validated{
			Value: Pipeline_Boundary_Validated_Value(Source_Position(token.End)),
		},
	)
	if status != PARSE_STATUS_OK {
		return count, depth, status
	}
	depth_scalar--
	depth.Value = depth_scalar
	*frame = (*parents)[depth_scalar].Value
	count, status = pipeline_append(workspace, count, frame_state, term)
	return count, depth, status
}

func pipeline_chain(
	source Source,
	workspace_state Syntax_Workspace_Pointer, count_value Node_Count_Allocated,
	cursor_state Token_Cursor_Pointer, reference_value Pipeline_Term_Validated,
	boundary_state Pipeline_Boundary_Validated,
) (_ Pipeline_Term_Validated, _ Node_Count_Allocated, status Build_Status) {
	defer func() { Build_Status_Invariants(status, "pipeline_chain.status") }()
	Syntax_Workspace_Pointer_Invariants(workspace_state, "pipeline_chain.workspace_state")
	Node_Count_Allocated_Invariants(count_value, "pipeline_chain.count_value")
	Token_Cursor_Pointer_Invariants(cursor_state, "pipeline_chain.cursor_state")
	Pipeline_Term_Validated_Invariants(reference_value, "pipeline_chain.reference_value")
	Pipeline_Boundary_Validated_Invariants(boundary_state, "pipeline_chain.boundary_state")
	Source_Invariants(source, "pipeline_chain.source")
	count, reference := count_value, reference_value
	defer func() { Node_Count_Allocated_Invariants(count, "pipeline_chain.count") }()
	defer func() { Pipeline_Term_Validated_Invariants(reference, "pipeline_chain.reference") }()
	workspace := (Syntax_Workspace_Pointer)(workspace_state)
	cursor := (Token_Cursor_Pointer)(cursor_state)
	base := reference.Value.(Pipeline_Term_Validated_Value)
	boundary := boundary_state.Value.(Pipeline_Boundary_Validated_Value)
	status = Build_Status(token_peek(source, cursor))
	if status != PARSE_STATUS_OK {
		return reference, count, status
	}
	next := Token(cursor.Token.Value)
	if next.Kind != TOKEN_FIELD {
		return reference, count, PARSE_STATUS_OK
	}
	if Source_Position(next.Start) != Source_Position(boundary) {
		return reference, count, PARSE_STATUS_OK
	}
	chain, updated_count, allocation_status := allocate_node(
		workspace, count, Node_Allocation{Value: Node(Node{
			Kind: NODE_CHAIN, Start: workspace.Nodes[base-NODE_COUNT_INCREMENT].Start,
			End:         Node_End(boundary),
			Value_Start: Node_Value_Start(next.Start),
			Value_End:   Node_Value_End(next.End),
			First_Child: Node_First_Child(base),
		})})
	count = updated_count
	if allocation_status != PARSE_STATUS_OK {
		return reference, count, Build_Status(allocation_status)
	}
	for next.Kind == TOKEN_FIELD {
		field_status := token_take(source, cursor)
		if field_status != PARSE_STATUS_OK {
			return reference, count, Build_Status(field_status)
		}
		field := Token(cursor.Token.Value)
		workspace.Nodes[chain-NODE_COUNT_INCREMENT].End = Node_End(field.End)
		workspace.Nodes[chain-NODE_COUNT_INCREMENT].Value_End = Node_Value_End(field.End)
		status = Build_Status(token_peek(source, cursor))
		if status != PARSE_STATUS_OK {
			return reference, count, status
		}
		next = Token(cursor.Token.Value)
	}
	reference = Pipeline_Term_Validated{Value: Pipeline_Term_Validated_Value(chain)}
	return reference, count, PARSE_STATUS_OK
}

func pipeline_variable(
	source Source,
	configuration Configuration,
	workspace_state Syntax_Workspace_Pointer, count_value Node_Count_Allocated,
	variables_state Variable_State_Pointer, declaration_maximum Pipeline_Declaration_Count,
	cursor_state Token_Cursor_Pointer, frame_state Pipeline_Frame_Storage_Pointer,
) (_ Node_Count_Allocated, status Build_Status) {
	defer func() { Build_Status_Invariants(status, "pipeline_variable.status") }()
	Syntax_Workspace_Pointer_Invariants(workspace_state, "pipeline_variable.workspace_state")
	Node_Count_Allocated_Invariants(count_value, "pipeline_variable.count_value")
	Variable_State_Pointer_Invariants(variables_state, "pipeline_variable.variables_state")
	Token_Cursor_Pointer_Invariants(cursor_state, "pipeline_variable.cursor_state")
	Pipeline_Frame_Storage_Pointer_Invariants(frame_state, "pipeline_variable.frame_state")
	Source_Invariants(source, "pipeline_variable.source")
	Configuration_Invariants(configuration, "pipeline_variable.configuration")
	Pipeline_Declaration_Count_Invariants(
		declaration_maximum, "pipeline_variable.declaration_maximum",
	)
	count := count_value
	defer func() { Node_Count_Allocated_Invariants(count, "pipeline_variable.count") }()
	variables := (Variable_State_Pointer)(variables_state)
	workspace := (Syntax_Workspace_Pointer)(workspace_state)
	cursor := (Token_Cursor_Pointer)(cursor_state)
	probe := *cursor
	status = Build_Status(token_take(source, &probe))
	if status != PARSE_STATUS_OK {
		return count, status
	}
	status = Build_Status(token_peek(source, &probe))
	if status != PARSE_STATUS_OK {
		return count, status
	}
	next := Token(probe.Token.Value)
	declaration := Pipeline_Declaration(false)
	switch next.Kind {
	case TOKEN_ASSIGN, TOKEN_COMMA, TOKEN_DECLARE:
		declaration = true
	}
	Pipeline_Declaration_Invariants(declaration, "pipeline_variable.declaration")
	if !bool(declaration) {
		count, status = pipeline_scalar(
			source, configuration, workspace, count, variables, cursor, frame_state,
		)
		return count, status
	}
	count, status = pipeline_declaration(
		source, workspace, count, variables,
		declaration_maximum, cursor, frame_state,
	)
	return count, status
}

func pipeline_declaration(
	source Source,
	workspace_state Syntax_Workspace_Pointer, count_value Node_Count_Allocated,
	variables_state Variable_State_Pointer, declaration_maximum Pipeline_Declaration_Count,
	cursor_state Token_Cursor_Pointer, frame_state Pipeline_Frame_Storage_Pointer,
) (_ Node_Count_Allocated, status Build_Status) {
	defer func() { Build_Status_Invariants(status, "pipeline_declaration.status") }()
	Syntax_Workspace_Pointer_Invariants(workspace_state, "pipeline_declaration.workspace_state")
	Node_Count_Allocated_Invariants(count_value, "pipeline_declaration.count_value")
	Variable_State_Pointer_Invariants(variables_state, "pipeline_declaration.variables_state")
	Token_Cursor_Pointer_Invariants(cursor_state, "pipeline_declaration.cursor_state")
	Pipeline_Frame_Storage_Pointer_Invariants(frame_state, "pipeline_declaration.frame_state")
	Source_Invariants(source, "pipeline_declaration.source")
	Pipeline_Declaration_Count_Invariants(
		declaration_maximum, "pipeline_declaration.declaration_maximum",
	)
	count := count_value
	defer func() { Node_Count_Allocated_Invariants(count, "pipeline_declaration.count") }()
	variables := (Variable_State_Pointer)(variables_state)
	workspace := (Syntax_Workspace_Pointer)(workspace_state)
	cursor := (Token_Cursor_Pointer)(cursor_state)
	frame := &frame_state.Value
	status = Build_Status(token_take(source, cursor))
	if status != PARSE_STATUS_OK {
		return count, status
	}
	if frame.Last_Child != Pipeline_Last_Child(NO_NODE) {
		return count, PARSE_STATUS_SYNTAX_INVALID
	}
	declarations := Pipeline_Declarations{First: Pipeline_First_Declaration_Storage{
		Value: Pipeline_First_Declaration(Token(cursor.Token.Value)),
	}}
	declaration_count := Pipeline_Declaration_Count(PIPELINE_DECLARATION_COUNT_MINIMUM)
	status = Build_Status(token_take(source, cursor))
	if status != PARSE_STATUS_OK {
		return count, status
	}
	operator := Token(cursor.Token.Value)
	if operator.Kind == TOKEN_COMMA {
		if declaration_maximum < PIPELINE_DECLARATION_COUNT_MAXIMUM {
			return count, PARSE_STATUS_SYNTAX_INVALID
		}
		second_status := token_take(source, cursor)
		if second_status != PARSE_STATUS_OK {
			return count, Build_Status(second_status)
		}
		second := Token(cursor.Token.Value)
		if second.Kind != TOKEN_VARIABLE {
			return count, PARSE_STATUS_SYNTAX_INVALID
		}
		declarations.Second.Value = Pipeline_Second_Declaration(Token(second))
		declaration_count++
		status = Build_Status(token_take(source, cursor))
		if status != PARSE_STATUS_OK {
			return count, status
		}
		operator = Token(cursor.Token.Value)
	}
	if operator.Kind != TOKEN_ASSIGN {
		if operator.Kind != TOKEN_DECLARE {
			return count, PARSE_STATUS_SYNTAX_INVALID
		}
	}
	workspace.Nodes[frame.Pipe-NODE_COUNT_INCREMENT].Value_Start =
		Node_Value_Start(operator.Start)
	workspace.Nodes[frame.Pipe-NODE_COUNT_INCREMENT].Value_End =
		Node_Value_End(operator.End)
	count, status = pipeline_declaration_store(
		source, workspace, count, variables, frame_state,
		declarations, declaration_count,
	)
	return count, status
}

func pipeline_declaration_store(
	source Source,
	workspace_state Syntax_Workspace_Pointer, count_value Node_Count_Allocated,
	variables_state Variable_State_Pointer, frame_state Pipeline_Frame_Storage_Pointer,
	declarations Pipeline_Declarations,
	declaration_count Pipeline_Declaration_Count,
) (_ Node_Count_Allocated, status Build_Status) {
	defer func() { Build_Status_Invariants(status, "pipeline_declaration_store.status") }()
	Syntax_Workspace_Pointer_Invariants(
		workspace_state, "pipeline_declaration_store.workspace_state",
	)
	Node_Count_Allocated_Invariants(count_value, "pipeline_declaration_store.count_value")
	Variable_State_Pointer_Invariants(
		variables_state, "pipeline_declaration_store.variables_state",
	)
	Pipeline_Frame_Storage_Pointer_Invariants(
		frame_state, "pipeline_declaration_store.frame_state")
	Source_Invariants(source, "pipeline_declaration_store.source")
	Pipeline_Declarations_Invariants(
		declarations, "pipeline_declaration_store.declarations",
	)
	Pipeline_Declaration_Count_Invariants(
		declaration_count, "pipeline_declaration_store.declaration_count",
	)
	count := count_value
	defer func() {
		Node_Count_Allocated_Invariants(count, "pipeline_declaration_store.count")
	}()
	variables := (Variable_State_Pointer)(variables_state)
	workspace := (Syntax_Workspace_Pointer)(workspace_state)
	frame := &frame_state.Value
	data := source.Data.Value
	for index := NODE_COUNT_MINIMUM; index < int(declaration_count); index++ {
		declaration := Token(declarations.First.Value.(Pipeline_First_Declaration))
		if index == int(PIPELINE_DECLARATION_COUNT_MINIMUM) {
			declaration = Token(declarations.Second.Value.(Pipeline_Second_Declaration))
		}
		declaration_start := Source_Position(declaration.Start) + SOURCE_POSITION_INCREMENT
		declaration_end := Source_Position(declaration.End)
		for position := declaration_start; position < declaration_end; position++ {
			if data[position] == '.' {
				return count, PARSE_STATUS_SYNTAX_INVALID
			}
		}
		reference, updated_count, allocation_status := allocate_node(
			workspace, count, Node_Allocation{Value: Node(Node{
				Kind: NODE_VARIABLE, Start: Node_Start(declaration.Start),
				End:         Node_End(declaration.End),
				Value_Start: Node_Value_Start(declaration.Start),
				Value_End:   Node_Value_End(declaration.End),
			})},
		)
		count = updated_count
		if allocation_status != PARSE_STATUS_OK {
			return count, Build_Status(allocation_status)
		}
		last_child := Node_Link_Validated{
			Value: Node_Link_Validated_Value(Node_Reference(frame.Last_Child)),
		}
		last_child = append_child(
			workspace,
			Node_Link_Validated{Value: Node_Link_Validated_Value(frame.Pipe)},
			last_child,
			Node_Link_Validated{Value: Node_Link_Validated_Value(reference)},
		)
		frame.Last_Child = Pipeline_Last_Child(last_child.Value.(Node_Link_Validated_Value))
		variable_add(variables, Declaration_Reference_Validated{
			Value: Declaration_Reference_Validated_Value(reference),
		})
	}
	return count, PARSE_STATUS_OK
}

func variable_add(
	variables_state Variable_State_Pointer, reference_value Declaration_Reference_Validated,
) {
	Variable_State_Pointer_Invariants(variables_state, "variable_add.variables_state")
	Declaration_Reference_Validated_Invariants(reference_value, "variable_add.reference_value")
	variables := (Variable_State_Pointer)(variables_state)
	reference := reference_value.Value.(Declaration_Reference_Validated_Value)
	count := variables.Count.Value.(Syntax_Variable_Count_Storage_Value)
	aver.Always(
		count < Syntax_Variable_Count_Storage_Value(SYNTAX_VARIABLE_COUNT_MAXIMUM),
		"Source density leaves capacity for every reachable declaration.",
	)
	variables.References[count] = Node_Reference(reference)
	variables.Count.Value = count + VARIABLE_COUNT_INCREMENT
}

func variable_known(
	source Source,
	chain_value Variable_Chain_Validated,
	workspace_state Syntax_Workspace_Pointer, variables_state Variable_State_Pointer,
) (known Variable_Known) {
	defer func() { Variable_Known_Invariants(known, "variable_known.known") }()
	Variable_Chain_Validated_Invariants(chain_value, "variable_known.chain_value")
	Syntax_Workspace_Pointer_Invariants(workspace_state, "variable_known.workspace_state")
	Variable_State_Pointer_Invariants(variables_state, "variable_known.variables_state")
	Source_Invariants(source, "variable_known.source")
	variables := (Variable_State_Pointer)(variables_state)
	workspace := (Syntax_Workspace_Pointer)(workspace_state)
	data := source.Data.Value
	chain := chain_value.Value.(Variable_Chain_Validated_Value)
	name_size := len(chain)
	for byte_index := VARIABLE_NAME_SIZE_MINIMUM; byte_index < name_size; byte_index++ {
		if chain[byte_index] == '.' {
			name_size = byte_index
			break
		}
	}
	if name_size == VARIABLE_NAME_SIZE_MINIMUM {
		return true
	}
	variable_index := int(variables.Count.Value.(Syntax_Variable_Count_Storage_Value))
	for variable_index > VARIABLE_COUNT_MINIMUM {
		variable_index--
		reference := variables.References[variable_index]
		declaration := workspace.Nodes[reference-NODE_COUNT_INCREMENT]
		declaration_start := int(declaration.Value_Start)
		declaration_end := int(declaration.Value_End)
		if name_size != declaration_end-declaration_start {
			continue
		}
		equal := true
		for byte_index := SOURCE_SIZE_MINIMUM; byte_index < name_size; byte_index++ {
			if chain[byte_index] != data[declaration_start+byte_index] {
				equal = false
				break
			}
		}
		if equal {
			return true
		}
	}
	return false
}

func pipeline_scalar(
	source Source, configuration Configuration,
	workspace_state Syntax_Workspace_Pointer, count_value Node_Count_Allocated,
	variables_state Variable_State_Pointer, cursor_state Token_Cursor_Pointer,
	frame_state Pipeline_Frame_Storage_Pointer,
) (_ Node_Count_Allocated, status Build_Status) {
	defer func() { Build_Status_Invariants(status, "pipeline_scalar.status") }()
	Syntax_Workspace_Pointer_Invariants(workspace_state, "pipeline_scalar.workspace_state")
	Node_Count_Allocated_Invariants(count_value, "pipeline_scalar.count_value")
	Variable_State_Pointer_Invariants(variables_state, "pipeline_scalar.variables_state")
	Token_Cursor_Pointer_Invariants(cursor_state, "pipeline_scalar.cursor_state")
	Pipeline_Frame_Storage_Pointer_Invariants(frame_state, "pipeline_scalar.frame_state")
	Source_Invariants(source, "pipeline_scalar.source")
	Configuration_Invariants(configuration, "pipeline_scalar.configuration")
	count, workspace := count_value, (Syntax_Workspace_Pointer)(workspace_state)
	defer func() { Node_Count_Allocated_Invariants(count, "pipeline_scalar.count") }()
	cursor := (Token_Cursor_Pointer)(cursor_state)
	status = Build_Status(token_take(source, cursor))
	if status != PARSE_STATUS_OK {
		return count, status
	}
	token := Token(cursor.Token.Value)
	validated := Token_Validated{Value: Token_Validated_Value(token)}
	classification := token_node_kind(validated).Value
	if !bool(classification.Known) {
		return count, PARSE_STATUS_SYNTAX_INVALID
	}
	if token.Kind == TOKEN_IDENTIFIER {
		name := Parse_Function_Name(source.Data.Value[token.Start:token.End])
		if !bool(function_known(Parse_Function_Name_Validated{
			Value: Parse_Function_Name_Validated_Value(name),
		}, configuration)) {
			return count, PARSE_STATUS_SYNTAX_INVALID
		}
	}
	if token.Kind == TOKEN_VARIABLE {
		chain := Variable_Chain(source.Data.Value[token.Start:token.End])
		if !bool(variable_known(
			source, Variable_Chain_Validated{
				Value: Variable_Chain_Validated_Value(chain),
			}, workspace, variables_state,
		)) {
			return count, PARSE_STATUS_SYNTAX_INVALID
		}
	}
	kind := classification.Kind.Value.(Classified_Node_Kind_Value)
	reference, updated_count, allocation_status := allocate_node(
		workspace, count, Node_Allocation{Value: Node(Node{
			Kind:        Node_Kind(kind),
			Start:       Node_Start(token.Start),
			End:         Node_End(token.End),
			Value_Start: Node_Value_Start(token.Start),
			Value_End:   Node_Value_End(token.End),
		})},
	)
	count = updated_count
	if allocation_status != PARSE_STATUS_OK {
		return count, Build_Status(allocation_status)
	}
	term := Pipeline_Term_Validated{Value: Pipeline_Term_Validated_Value(reference)}
	if token.Kind == TOKEN_IDENTIFIER {
		term, count, status = pipeline_chain(
			source, workspace, count, cursor, term,
			Pipeline_Boundary_Validated{
				Value: Pipeline_Boundary_Validated_Value(token.End),
			},
		)
		if status != PARSE_STATUS_OK {
			return count, status
		}
	}
	count, status = pipeline_append(workspace, count, frame_state, term)
	return count, status
}

func pipeline_append(
	workspace_state Syntax_Workspace_Pointer, count_value Node_Count_Allocated,
	frame_state Pipeline_Frame_Storage_Pointer, term_state Pipeline_Term_Validated,
) (_ Node_Count_Allocated, status Build_Status) {
	defer func() { Build_Status_Invariants(status, "pipeline_append.status") }()
	Syntax_Workspace_Pointer_Invariants(workspace_state, "pipeline_append.workspace_state")
	Node_Count_Allocated_Invariants(count_value, "pipeline_append.count_value")
	Pipeline_Frame_Storage_Pointer_Invariants(frame_state, "pipeline_append.frame_state")
	Pipeline_Term_Validated_Invariants(term_state, "pipeline_append.term_state")
	count, workspace := count_value, Syntax_Workspace_Pointer(workspace_state)
	defer func() { Node_Count_Allocated_Invariants(count, "pipeline_append.count") }()
	frame, term := &frame_state.Value, term_state.Value.(Pipeline_Term_Validated_Value)
	if frame.Command == Pipeline_Command(NO_NODE) {
		node := workspace.Nodes[term-NODE_COUNT_INCREMENT]
		if frame.Last_Child != Pipeline_Last_Child(NO_NODE) {
			previous := workspace.Nodes[frame.Last_Child-NODE_COUNT_INCREMENT]
			if previous.Kind == NODE_COMMAND {
				switch node.Kind {
				case NODE_BOOLEAN, NODE_DOT, NODE_NIL, NODE_NUMBER, NODE_STRING:
					return count, PARSE_STATUS_SYNTAX_INVALID
				}
			}
		}
		command, updated_count, allocation_status := allocate_node(
			workspace, count, Node_Allocation{Value: Node(Node{
				Kind: NODE_COMMAND, Start: node.Start, End: node.End,
				Value_Start: Node_Value_Start(node.Start),
				Value_End:   Node_Value_End(node.End),
			})},
		)
		count = updated_count
		if allocation_status != PARSE_STATUS_OK {
			return count, Build_Status(allocation_status)
		}
		last_child := Node_Link_Validated{
			Value: Node_Link_Validated_Value(Node_Reference(frame.Last_Child)),
		}
		last_child = append_child(
			workspace,
			Node_Link_Validated{Value: Node_Link_Validated_Value(frame.Pipe)},
			last_child,
			Node_Link_Validated{Value: Node_Link_Validated_Value(command)},
		)
		frame.Last_Child = Pipeline_Last_Child(last_child.Value.(Node_Link_Validated_Value))
		frame.Command = Pipeline_Command(command)
		frame.Last_Argument = Pipeline_Last_Argument(NO_NODE)
	}
	if frame.Last_Argument != Pipeline_Last_Argument(NO_NODE) {
		previous := workspace.Nodes[frame.Last_Argument-NODE_COUNT_INCREMENT]
		current := workspace.Nodes[term-NODE_COUNT_INCREMENT]
		if previous.End == Node_End(current.Start) {
			return count, PARSE_STATUS_SYNTAX_INVALID
		}
	}
	last := Node_Link_Validated{
		Value: Node_Link_Validated_Value(Node_Reference(frame.Last_Argument)),
	}
	last = append_child(
		workspace,
		Node_Link_Validated{Value: Node_Link_Validated_Value(frame.Command)},
		last,
		Node_Link_Validated{Value: Node_Link_Validated_Value(term)},
	)
	frame.Last_Argument = Pipeline_Last_Argument(last.Value.(Node_Link_Validated_Value))
	term_end := workspace.Nodes[term-NODE_COUNT_INCREMENT].End
	workspace.Nodes[frame.Command-NODE_COUNT_INCREMENT].End = term_end
	workspace.Nodes[frame.Command-NODE_COUNT_INCREMENT].Value_End = Node_Value_End(
		term_end,
	)
	return count, PARSE_STATUS_OK
}

func token_node_kind(
	token_value Token_Validated,
) (classification Node_Classification_Validated) {
	defer func() {
		Node_Classification_Validated_Invariants(
			classification, "token_node_kind.classification",
		)
	}()
	Token_Validated_Invariants(token_value, "token_node_kind.token_value")
	kind := token_value.Value.Kind
	switch Token_Kind(kind) {
	case TOKEN_BOOLEAN:
		return Node_Classification_Validated{Value: Token_Node_Classification{
			Kind: Classified_Node_Kind{
				Value: Classified_Node_Kind_Value(NODE_BOOLEAN),
			}, Known: true,
		}}
	case TOKEN_CHARACTER, TOKEN_NUMBER:
		return Node_Classification_Validated{Value: Token_Node_Classification{
			Kind: Classified_Node_Kind{
				Value: Classified_Node_Kind_Value(NODE_NUMBER),
			}, Known: true,
		}}
	case TOKEN_DOT:
		return Node_Classification_Validated{Value: Token_Node_Classification{
			Kind: Classified_Node_Kind{
				Value: Classified_Node_Kind_Value(NODE_DOT),
			}, Known: true,
		}}
	case TOKEN_FIELD:
		return Node_Classification_Validated{Value: Token_Node_Classification{
			Kind: Classified_Node_Kind{
				Value: Classified_Node_Kind_Value(NODE_FIELD),
			}, Known: true,
		}}
	case TOKEN_IDENTIFIER:
		return Node_Classification_Validated{Value: Token_Node_Classification{
			Kind: Classified_Node_Kind{
				Value: Classified_Node_Kind_Value(NODE_IDENTIFIER),
			}, Known: true,
		}}
	case TOKEN_NIL:
		return Node_Classification_Validated{Value: Token_Node_Classification{
			Kind: Classified_Node_Kind{
				Value: Classified_Node_Kind_Value(NODE_NIL),
			}, Known: true,
		}}
	case TOKEN_RAW_STRING, TOKEN_STRING:
		return Node_Classification_Validated{Value: Token_Node_Classification{
			Kind: Classified_Node_Kind{
				Value: Classified_Node_Kind_Value(NODE_STRING),
			}, Known: true,
		}}
	case TOKEN_VARIABLE:
		return Node_Classification_Validated{Value: Token_Node_Classification{
			Kind: Classified_Node_Kind{
				Value: Classified_Node_Kind_Value(NODE_VARIABLE),
			}, Known: true,
		}}
	default:
		return Node_Classification_Validated{Value: Token_Node_Classification{
			Kind: Classified_Node_Kind{Value: Classified_Node_Kind_Value(NODE_TEXT)},
		}}
	}
}

func function_known(
	name_value Parse_Function_Name_Validated, configuration Configuration,
) (known Function_Known) {
	defer func() { Function_Known_Invariants(known, "function_known.known") }()
	Parse_Function_Name_Validated_Invariants(name_value, "function_known.name_value")
	Configuration_Invariants(configuration, "function_known.configuration")
	name := name_value.Value.(Parse_Function_Name_Validated_Value)
	if Mode(configuration.Mode.Value)&SKIP_FUNCTION_CHECK != MODE_MINIMUM {
		return true
	}
	function := configuration.Function.Value
	if function == nil {
		return false
	}
	known = function(Parse_Function_Name(name))
	Function_Known_Invariants(known, "function_known.callback")
	return known
}

func token_peek(
	source Source, cursor_state Token_Cursor_Pointer) (status Lex_Status) {
	defer func() { Lex_Status_Invariants(status, "token_peek.status") }()
	Token_Cursor_Pointer_Invariants(cursor_state, "token_peek.cursor_state")
	Source_Invariants(source, "token_peek.source")
	cursor := (Token_Cursor_Pointer)(cursor_state)
	if cursor.Looked.Value != Look_Ahead_Storage_Value(0) {
		return PARSE_STATUS_OK
	}
	status = token_read(source, cursor)
	if status != PARSE_STATUS_OK {
		cursor.Last_Start.Value = Cursor_Last_Start_Storage_Value(
			cursor.Position.Value,
		)
		cursor.Has_Last.Value = Cursor_Has_Last_Storage_Value(1)
		return status
	}
	cursor.Looked.Value = Look_Ahead_Storage_Value(1)
	return PARSE_STATUS_OK
}

func token_take(
	source Source, cursor_state Token_Cursor_Pointer) (status Lex_Status) {
	defer func() { Lex_Status_Invariants(status, "token_take.status") }()
	Token_Cursor_Pointer_Invariants(cursor_state, "token_take.cursor_state")
	Source_Invariants(source, "token_take.source")
	cursor := (Token_Cursor_Pointer)(cursor_state)
	status = token_peek(source, cursor)
	if status != PARSE_STATUS_OK {
		return status
	}
	cursor.Last_Start.Value = Cursor_Last_Start_Storage_Value(
		cursor.Token.Value.Start,
	)
	cursor.Has_Last.Value = Cursor_Has_Last_Storage_Value(1)
	cursor.Looked.Value = Look_Ahead_Storage_Value(0)
	return PARSE_STATUS_OK
}

func token_read(
	source Source, cursor_state Token_Cursor_Pointer) (status Lex_Status) {
	defer func() { Lex_Status_Invariants(status, "token_read.status") }()
	Token_Cursor_Pointer_Invariants(cursor_state, "token_read.cursor_state")
	Source_Invariants(source, "token_read.source")
	cursor := (Token_Cursor_Pointer)(cursor_state)
	defer func() {
		Token_Invariants(Token(cursor.Token.Value), "token_read.token")
	}()
	data := source.Data.Value
	position := Source_Position(cursor.Position.Value)
	limit := Source_Position(cursor.End.Value)
	for position < limit {
		if !bool(space(Input_Byte(data[position]))) {
			break
		}
		position++
	}
	if position == limit {
		cursor.Position.Value = Cursor_Position_Storage_Value(position)
		cursor.Token.Value = Token_Stored(Token{
			Kind: TOKEN_EOF, Start: Token_Start(position), End: Token_End(position),
		})
		return PARSE_STATUS_OK
	}
	lexeme := Lexeme_Validated{Value: Lexeme_Stored(Lexeme{
		Start: Lexeme_Start(position), Character: Input_Byte(data[position]),
	})}

	return token_read_compound(source, cursor, lexeme)
}

func simple_token_kind(
	character Input_Byte,
) (_ Classified_Token_Kind, known Node_Kind_Known) {
	defer func() { Node_Kind_Known_Invariants(known, "simple_token_kind.known") }()
	Input_Byte_Invariants(character, "simple_token_kind.character")
	kind := Classified_Token_Kind{Value: Classified_Token_Kind_Value(TOKEN_EOF)}
	defer func() { Classified_Token_Kind_Invariants(kind, "simple_token_kind.kind") }()
	switch character {
	case '=':
		kind.Value = Classified_Token_Kind_Value(TOKEN_ASSIGN)
		known = true
	case ',':
		kind.Value = Classified_Token_Kind_Value(TOKEN_COMMA)
		known = true
	case '(':
		kind.Value = Classified_Token_Kind_Value(TOKEN_LEFT_PAREN)
		known = true
	case ')':
		kind.Value = Classified_Token_Kind_Value(TOKEN_RIGHT_PAREN)
		known = true
	case '|':
		kind.Value = Classified_Token_Kind_Value(TOKEN_PIPE)
		known = true
	}
	return kind, known
}

func token_read_compound(
	source Source,
	cursor_state Token_Cursor_Pointer, lexeme_state Lexeme_Validated,
) (status Lex_Status) {
	defer func() { Lex_Status_Invariants(status, "token_read_compound.status") }()
	Token_Cursor_Pointer_Invariants(cursor_state, "token_read_compound.cursor_state")
	Lexeme_Validated_Invariants(lexeme_state, "token_read_compound.lexeme_state")
	Source_Invariants(source, "token_read_compound.source")
	cursor := (Token_Cursor_Pointer)(cursor_state)
	lexeme := lexeme_state.Value
	position := Source_Position(lexeme.Start)
	simple_kind, simple := simple_token_kind(lexeme.Character)
	if bool(simple) {
		cursor.Position.Value = Cursor_Position_Storage_Value(
			position + SOURCE_POSITION_INCREMENT,
		)
		cursor.Token.Value = Token_Stored(Token{
			Kind:  Token_Kind(simple_kind.Value.(Classified_Token_Kind_Value)),
			Start: Token_Start(position),
			End:   Token_End(position + SOURCE_POSITION_INCREMENT),
		})
		return PARSE_STATUS_OK
	}
	cursor.Token.Value = Token_Stored(Token{
		Kind: TOKEN_EOF, Start: Token_Start(position), End: Token_End(position),
	})
	switch lexeme.Character {
	case ':':
		status = token_declare(source, cursor, lexeme_state)
	case '/':
		status = token_comment(source, cursor, lexeme_state)
	case '.', '$':
		status = token_field_variable_number(source, cursor, lexeme_state)
	case '\'', '"', '`':
		status = token_quoted(source, cursor, lexeme_state)
	case '+', '-':
		status = token_number(source, cursor, lexeme_state)
	default:
		if lexeme.Character >= '0' {
			if lexeme.Character <= '9' {
				status = token_number(source, cursor, lexeme_state)
				break
			}
		}
		identifier, _ := identifier_character_at(
			source, Identifier_Search_Validated{Value: Identifier_Search_Span{
				Position: Identifier_Search_Position_Storage{
					Value: Identifier_Search_Position(position),
				},
				End: Identifier_Search_End_Storage{
					Value: Identifier_Search_End(cursor.End.Value),
				},
			}})
		if bool(identifier) {
			token_identifier(source, cursor, lexeme_state)
			break
		}
		return PARSE_STATUS_SYNTAX_INVALID
	}
	if status != PARSE_STATUS_OK {
		return status
	}
	return PARSE_STATUS_OK
}

func token_comment(
	source Source, cursor_state Token_Cursor_Pointer, lexeme_state Lexeme_Validated,
) (status Lex_Status) {
	defer func() { Lex_Status_Invariants(status, "token_comment.status") }()
	Token_Cursor_Pointer_Invariants(cursor_state, "token_comment.cursor_state")
	Lexeme_Validated_Invariants(lexeme_state, "token_comment.lexeme_state")
	Source_Invariants(source, "token_comment.source")
	cursor := (Token_Cursor_Pointer)(cursor_state)
	lexeme := lexeme_state.Value
	position := Source_Position(lexeme.Start)
	span := Action_Scan_Span_Validated{Value: Action_Scan_Span_Stored(Action_Scan_Span{
		Start: Action_Scan_Start(position),
		Limit: Action_Scan_Limit(cursor.End.Value),
	})}

	end_state, comment_status := comment_end(source, span)
	status = comment_status
	if status != PARSE_STATUS_OK {
		return status
	}
	end := end_state.Value.(Action_Scan_End_Validated_Value)
	cursor.Position.Value = Cursor_Position_Storage_Value(end)
	cursor.Token.Value = Token_Stored(Token{
		Kind: TOKEN_COMMENT, Start: Token_Start(position), End: Token_End(end),
	})
	return PARSE_STATUS_OK
}

func token_declare(
	source Source, cursor_state Token_Cursor_Pointer, lexeme_state Lexeme_Validated,
) (status Lex_Status) {
	defer func() { Lex_Status_Invariants(status, "token_declare.status") }()
	Token_Cursor_Pointer_Invariants(cursor_state, "token_declare.cursor_state")
	Lexeme_Validated_Invariants(lexeme_state, "token_declare.lexeme_state")
	Source_Invariants(source, "token_declare.source")
	cursor := (Token_Cursor_Pointer)(cursor_state)
	lexeme := lexeme_state.Value
	position := Source_Position(lexeme.Start)
	data := source.Data.Value
	end := position + SOURCE_POSITION_INCREMENT
	if end == Source_Position(cursor.End.Value) {
		return PARSE_STATUS_SYNTAX_INVALID
	}
	if data[end] != '=' {
		return PARSE_STATUS_SYNTAX_INVALID
	}
	cursor.Position.Value = Cursor_Position_Storage_Value(end + SOURCE_POSITION_INCREMENT)
	cursor.Token.Value = Token_Stored(Token{
		Kind: TOKEN_DECLARE, Start: Token_Start(position),
		End: Token_End(end + SOURCE_POSITION_INCREMENT),
	})
	return PARSE_STATUS_OK
}

func token_field_variable_number(
	source Source, cursor_state Token_Cursor_Pointer, lexeme_state Lexeme_Validated,
) (status Lex_Status) {
	defer func() { Lex_Status_Invariants(status, "token_field_variable_number.status") }()
	Token_Cursor_Pointer_Invariants(cursor_state, "token_field_variable_number.cursor_state")
	Lexeme_Validated_Invariants(lexeme_state, "token_field_variable_number.lexeme_state")
	Source_Invariants(source, "token_field_variable_number.source")
	cursor := (Token_Cursor_Pointer)(cursor_state)
	lexeme := lexeme_state.Value
	position, character := Source_Position(lexeme.Start), lexeme.Character
	data := source.Data.Value
	end := position + SOURCE_POSITION_INCREMENT
	limit := Source_Position(cursor.End.Value)
	if character == '.' {
		if end < limit {
			if data[end] >= '0' {
				if data[end] <= '9' {
					return token_number(source, cursor, lexeme_state)
				}
			}
		}
	}
	search := Identifier_Search_Validated{Value: Identifier_Search_Span{
		Position: Identifier_Search_Position_Storage{
			Value: Identifier_Search_Position(end),
		},
		End: Identifier_Search_End_Storage{Value: Identifier_Search_End(limit)},
	}}
	search = identifier_sequence(source, search)
	end = Source_Position(search.Value.Position.Value.(Identifier_Search_Position))
	for end < limit {
		if data[end] != '.' {
			break
		}
		if character == '.' {
			if end == position+SOURCE_POSITION_INCREMENT {
				return PARSE_STATUS_SYNTAX_INVALID
			}
		}
		end++
		if end == limit {
			return PARSE_STATUS_SYNTAX_INVALID
		}
		search = Identifier_Search_Validated{Value: Identifier_Search_Span{
			Position: Identifier_Search_Position_Storage{
				Value: Identifier_Search_Position(end),
			},
			End: Identifier_Search_End_Storage{
				Value: Identifier_Search_End(limit),
			},
		}}
		search = identifier_sequence(source, search)
		search_position := search.Value.Position.Value.(Identifier_Search_Position)
		updated_end := Source_Position(search_position)
		if updated_end == end {
			return PARSE_STATUS_SYNTAX_INVALID
		}
		end = updated_end
	}
	kind := TOKEN_VARIABLE
	if character == '.' {
		kind = TOKEN_FIELD
		if end == position+SOURCE_POSITION_INCREMENT {
			kind = TOKEN_DOT
		}
	}
	cursor.Position.Value = Cursor_Position_Storage_Value(end)
	cursor.Token.Value = Token_Stored(Token{
		Kind: kind, Start: Token_Start(position), End: Token_End(end),
	})
	return PARSE_STATUS_OK
}

func token_quoted(
	source Source,
	cursor_state Token_Cursor_Pointer, lexeme_state Lexeme_Validated,
) (status Lex_Status) {
	defer func() { Lex_Status_Invariants(status, "token_quoted.status") }()
	Token_Cursor_Pointer_Invariants(cursor_state, "token_quoted.cursor_state")
	Lexeme_Validated_Invariants(lexeme_state, "token_quoted.lexeme_state")
	Source_Invariants(source, "token_quoted.source")
	cursor := (Token_Cursor_Pointer)(cursor_state)
	lexeme := lexeme_state.Value
	position := Source_Position(lexeme.Start)
	quote := Quote(lexeme.Character)
	search := Action_Scan_Span_Validated{Value: Action_Scan_Span_Stored(Action_Scan_Span{
		Start: Action_Scan_Start(position),
		Limit: Action_Scan_Limit(cursor.End.Value),
	})}

	end_state, quoted_status := quoted_end(source, search, quote)
	status = quoted_status
	if status != PARSE_STATUS_OK {
		return status
	}
	end := end_state.Value.(Action_Scan_End_Validated_Value)
	span := Quoted_Span{Start: Quoted_Start(position), End: Quoted_End(end)}
	span_validated := Quoted_Span_Validated{Value: Quoted_Span_Stored(span)}
	valid := quoted_content_valid(source, span_validated, quote)
	if !bool(valid) {
		return PARSE_STATUS_SYNTAX_INVALID
	}
	kind := TOKEN_STRING
	if quote == '\'' {
		kind = TOKEN_CHARACTER
	}
	if quote == '`' {
		kind = TOKEN_RAW_STRING
	}
	cursor.Position.Value = Cursor_Position_Storage_Value(end)
	cursor.Token.Value = Token_Stored(Token{
		Kind: kind, Start: Token_Start(position), End: Token_End(end),
	})
	return PARSE_STATUS_OK
}

func quoted_end(
	source Source,
	span_state Action_Scan_Span_Validated,
	quote Quote,
) (_ Action_Scan_End_Validated, status Lex_Status) {
	defer func() { Lex_Status_Invariants(status, "quoted_end.status") }()
	Action_Scan_Span_Validated_Invariants(span_state, "quoted_end.span_state")
	Source_Invariants(source, "quoted_end.source")
	Quote_Invariants(quote, "quoted_end.quote")
	var end_state Action_Scan_End_Validated
	defer func() { Action_Scan_End_Validated_Invariants(end_state, "quoted_end.end_state") }()
	span := span_state.Value
	start, limit := Source_Position(span.Start), Source_Position(span.Limit)
	data := source.Data.Value
	end := start + SOURCE_POSITION_INCREMENT
	for end < limit {
		character := Input_Byte(data[end])
		end++
		if Quote(character) == quote {
			end_state = Action_Scan_End_Validated{
				Value: Action_Scan_End_Validated_Value(end),
			}
			return end_state, PARSE_STATUS_OK
		}
		if quote != '`' {
			if character == '\n' {
				end_state = Action_Scan_End_Validated{
					Value: Action_Scan_End_Validated_Value(end),
				}
				return end_state, PARSE_STATUS_SYNTAX_INVALID
			}
			if character == '\\' {
				if end == limit {
					end_state = Action_Scan_End_Validated{
						Value: Action_Scan_End_Validated_Value(end),
					}
					return end_state, PARSE_STATUS_SYNTAX_INVALID
				}
				end++
			}
		}
	}
	end_state = Action_Scan_End_Validated{
		Value: Action_Scan_End_Validated_Value(end),
	}
	return end_state, PARSE_STATUS_SYNTAX_INVALID
}

func quoted_content_valid(
	source Source, span_state Quoted_Span_Validated, quote Quote,
) (valid Quoted_Validity) {
	defer func() { Quoted_Validity_Invariants(valid, "quoted_content_valid.valid") }()
	Source_Invariants(source, "quoted_content_valid.source")
	Quoted_Span_Validated_Invariants(span_state, "quoted_content_valid.span_state")
	Quote_Invariants(quote, "quoted_content_valid.quote")
	span := span_state.Value
	start, end := Source_Position(span.Start), Source_Position(span.End)
	data := source.Data.Value
	if quote == '`' {
		return true
	}
	position := start + SOURCE_POSITION_INCREMENT
	limit := end - SOURCE_POSITION_INCREMENT
	character_count := QUOTED_CHARACTER_COUNT_MINIMUM
	for position < limit {
		if data[position] == '\\' {
			var escape_valid Quoted_Escape_Validity
			position_state, escape_valid := quoted_escape_end(
				source, span_state, Quoted_Position_Validated{
					Value: Quoted_Position_Validated_Value(position),
				},
				Escaped_Quote(quote),
			)
			position = Source_Position(
				position_state.Value.(Quoted_Position_Validated_Value),
			)
			if !bool(escape_valid) {
				return false
			}
		} else {
			position_state := quoted_character_end(
				source, span_state, Quoted_Position_Validated{
					Value: Quoted_Position_Validated_Value(position),
				},
			)
			position = Source_Position(
				position_state.Value.(Quoted_Position_Validated_Value),
			)
		}
		character_count++
		if quote == '\'' {
			if character_count > QUOTED_CHARACTER_COUNT_REQUIRED {
				return false
			}
		}
	}
	if quote == '\'' {
		return Quoted_Validity(character_count == QUOTED_CHARACTER_COUNT_REQUIRED)
	}
	return true
}

func quoted_escape_end(
	source Source, span_state Quoted_Span_Validated,
	position_value Quoted_Position_Validated, quote Escaped_Quote,
) (_ Quoted_Position_Validated, valid Quoted_Escape_Validity) {
	defer func() { Quoted_Escape_Validity_Invariants(valid, "quoted_escape_end.valid") }()
	Quoted_Span_Validated_Invariants(span_state, "quoted_escape_end.span_state")
	Quoted_Position_Validated_Invariants(position_value, "quoted_escape_end.position_value")
	Escaped_Quote_Invariants(quote, "quoted_escape_end.quote")
	Source_Invariants(source, "quoted_escape_end.source")
	position := position_value
	defer func() {
		Quoted_Position_Validated_Invariants(position, "quoted_escape_end.position")
	}()
	position_raw := Source_Position(position_value.Value.(Quoted_Position_Validated_Value))
	limit := Source_Position(span_state.Value.End) -
		SOURCE_POSITION_INCREMENT
	if position_raw+SOURCE_POSITION_INCREMENT >= limit {
		return position, false
	}
	data := source.Data.Value
	first := Input_Byte(data[position_raw+SOURCE_POSITION_INCREMENT])
	switch first {
	case 'a', 'b', 'f', 'n', 'r', 't', 'v', '\\':
		position_raw += Source_Position(QUOTED_ESCAPE_PREFIX_SIZE)
		position = Quoted_Position_Validated{
			Value: Quoted_Position_Validated_Value(position_raw),
		}
		return position, true
	case '\'', '"':
		if Escaped_Quote(first) != quote {
			return position, false
		}
		position_raw += Source_Position(QUOTED_ESCAPE_PREFIX_SIZE)
		position = Quoted_Position_Validated{
			Value: Quoted_Position_Validated_Value(position_raw),
		}
		return position, true
	case 'x':
		position_raw += Source_Position(QUOTED_ESCAPE_PREFIX_SIZE)
		position, valid = quoted_hex_escape_end(
			source, span_state, Quoted_Position_Validated{
				Value: Quoted_Position_Validated_Value(position_raw),
			},
			Quoted_Hex_Digit_Count(QUOTED_HEX_DIGIT_COUNT),
		)
		return position, valid
	case 'u':
		position_raw += Source_Position(QUOTED_ESCAPE_PREFIX_SIZE)
		position, valid = quoted_hex_escape_end(
			source, span_state, Quoted_Position_Validated{
				Value: Quoted_Position_Validated_Value(position_raw),
			},
			Quoted_Hex_Digit_Count(QUOTED_SHORT_UNICODE_DIGIT_COUNT),
		)
		return position, valid
	case 'U':
		position_raw += Source_Position(QUOTED_ESCAPE_PREFIX_SIZE)
		position, valid = quoted_hex_escape_end(
			source, span_state, Quoted_Position_Validated{
				Value: Quoted_Position_Validated_Value(position_raw),
			},
			Quoted_Hex_Digit_Count(QUOTED_LONG_UNICODE_DIGIT_COUNT),
		)
		return position, valid
	}
	position, valid = quoted_octal_escape_end(source, span_state, position_value)
	return position, valid
}

func quoted_octal_escape_end(
	source Source, span_state Quoted_Span_Validated,
	position_value Quoted_Position_Validated,
) (_ Quoted_Position_Validated, valid Quoted_Escape_Validity) {
	defer func() { Quoted_Escape_Validity_Invariants(valid, "quoted_octal_escape_end.valid") }()
	Quoted_Span_Validated_Invariants(span_state, "quoted_octal_escape_end.span_state")
	Quoted_Position_Validated_Invariants(
		position_value, "quoted_octal_escape_end.position_value",
	)
	Source_Invariants(source, "quoted_octal_escape_end.source")
	position := position_value
	defer func() {
		Quoted_Position_Validated_Invariants(position, "quoted_octal_escape_end.position")
	}()
	position_raw := Source_Position(position_value.Value.(Quoted_Position_Validated_Value))
	data := source.Data.Value
	first := Input_Byte(data[position_raw+SOURCE_POSITION_INCREMENT])
	if first < '0' {
		return position_value, false
	}
	if first > '7' {
		return position_value, false
	}
	limit := Source_Position(span_state.Value.End) -
		SOURCE_POSITION_INCREMENT
	octal_end := position_raw + Source_Position(QUOTED_OCTAL_DIGIT_COUNT) +
		SOURCE_POSITION_INCREMENT
	if octal_end > limit {
		return position_value, false
	}
	value := uint16(bits.WORD_16_MINIMUM)
	digit_count := Source_Position(QUOTED_OCTAL_DIGIT_COUNT)
	for index := Source_Position(SOURCE_POSITION_INCREMENT); index <= digit_count; index++ {
		digit := data[position_raw+index]
		if digit < '0' {
			return position_value, false
		}
		if digit > '7' {
			return position_value, false
		}
		value = value<<NUMBER_BASE_OCTAL_DIGIT_BIT_COUNT | uint16(digit-'0')
	}
	if value > uint16(bits.WORD_8_MAXIMUM) {
		return position_value, false
	}
	position = Quoted_Position_Validated{Value: Quoted_Position_Validated_Value(octal_end)}
	return position, true
}

func quoted_hex_escape_end(
	source Source,
	span_state Quoted_Span_Validated,
	position_value Quoted_Position_Validated,
	digit_count Quoted_Hex_Digit_Count,
) (_ Quoted_Position_Validated, valid Quoted_Escape_Validity) {
	defer func() { Quoted_Escape_Validity_Invariants(valid, "quoted_hex_escape_end.valid") }()
	Quoted_Span_Validated_Invariants(span_state, "quoted_hex_escape_end.span_state")
	Quoted_Position_Validated_Invariants(position_value, "quoted_hex_escape_end.position_value")
	Quoted_Hex_Digit_Count_Invariants(
		digit_count, "quoted_hex_escape_end.digit_count",
	)
	Source_Invariants(source, "quoted_hex_escape_end.source")
	span := span_state.Value
	position_raw := Source_Position(position_value.Value.(Quoted_Position_Validated_Value))
	invalid := Quoted_Position_Validated{
		Value: Quoted_Position_Validated_Value(position_raw),
	}
	position := invalid
	defer func() {
		Quoted_Position_Validated_Invariants(position, "quoted_hex_escape_end.position")
	}()
	limit := Source_Position(span.End) - SOURCE_POSITION_INCREMENT
	if int(position_raw)+int(digit_count) > int(limit) {
		return invalid, false
	}
	data := source.Data.Value
	value := uint32(bits.WORD_32_MINIMUM)
	index := Source_Position(SOURCE_SIZE_MINIMUM)
	for index < Source_Position(digit_count) {
		character := Input_Byte(data[position_raw+index])
		digit := uint32(bits.WORD_32_MINIMUM)
		if character >= '0' {
			if character <= '9' {
				digit = uint32(character - '0')
			} else if character >= 'a' {
				if character > 'f' {
					return invalid, false
				}
				digit = uint32(character-'a') + uint32(NUMBER_BASE_DECIMAL)
			} else if character >= 'A' {
				if character > 'F' {
					return invalid, false
				}
				digit = uint32(character-'A') + uint32(NUMBER_BASE_DECIMAL)
			} else {
				return invalid, false
			}
		} else {
			return invalid, false
		}
		value = value<<NUMBER_BASE_HEXADECIMAL_DIGIT_BIT_COUNT | digit
		index++
	}
	if Quoted_Digit_Count(digit_count) != QUOTED_HEX_DIGIT_COUNT {
		if !bool(utf8.Valid_Character(utf8.Character(value))) {
			return invalid, false
		}
	}
	position = Quoted_Position_Validated{
		Value: Quoted_Position_Validated_Value(
			position_raw + Source_Position(digit_count),
		),
	}
	return position, true
}

func quoted_character_end(
	source Source, span_state Quoted_Span_Validated,
	position_value Quoted_Position_Validated,
) (position Quoted_Position_Validated) {
	defer func() {
		Quoted_Position_Validated_Invariants(position, "quoted_character_end.position")
	}()
	Quoted_Span_Validated_Invariants(span_state, "quoted_character_end.span_state")
	Quoted_Position_Validated_Invariants(position_value, "quoted_character_end.position_value")
	Source_Invariants(source, "quoted_character_end.source")
	span := span_state.Value
	position_raw := Source_Position(position_value.Value.(Quoted_Position_Validated_Value))
	limit := Source_Position(span.End) - SOURCE_POSITION_INCREMENT
	data := source.Data.Value
	_, size := utf8.Decode_Character(utf8.Bytes(data[position_raw:limit]))
	// Standard unquoting treats one malformed byte as one replacement character.
	return Quoted_Position_Validated{
		Value: Quoted_Position_Validated_Value(position_raw + Source_Position(size)),
	}
}

func quoted_unquote_into(
	source Source,
	span_state Quoted_Span_Validated,
	output Decoded_Output,
) (count Template_Name_Count) {
	defer func() {
		Template_Name_Count_Invariants(count, "quoted_unquote_into.count")
	}()
	Quoted_Span_Validated_Invariants(span_state, "quoted_unquote_into.span_state")
	Decoded_Output_Invariants(output, "quoted_unquote_into.output")
	Source_Invariants(source, "quoted_unquote_into.source")
	span := span_state.Value
	aver.Always(
		len(output) >= QUOTED_DECODED_SIZE_MAXIMUM,
		"Quoted decoding storage holds worst source expansion.",
	)
	data := source.Data.Value
	start, end := Source_Position(span.Start), Source_Position(span.End)
	quote := data[start]
	position := start + SOURCE_POSITION_INCREMENT
	limit := end - SOURCE_POSITION_INCREMENT
	for position < limit {
		if quote == '`' {
			if data[position] != '\r' {
				output[count] = data[position]
				count++
			}
			position++
			continue
		}
		if data[position] != '\\' {
			character, size := utf8.Decode_Character(
				utf8.Bytes(data[position:limit]),
			)
			character_size := utf8.Character_Size(utf8.Character(character))
			encode_limit := count + Template_Name_Count(character_size)
			encoded_size := utf8.Encode_Character(
				utf8.Bytes(output[count:encode_limit]), utf8.Character(character),
			)
			count += Template_Name_Count(encoded_size)
			position += Source_Position(size)
			continue
		}
		var value Quoted_Escape_Value
		var form Quoted_Escape_Form
		position_state, decoded_value, decoded_form := quoted_escape_decode(
			source, Quoted_Position_Validated{
				Value: Quoted_Position_Validated_Value(position),
			},
		)
		position = Source_Position(position_state.Value.(Quoted_Position_Validated_Value))
		value, form = decoded_value, decoded_form
		if form == QUOTED_ESCAPE_BYTE {
			output[count] = byte(value)
			count++
		} else {
			character_size := utf8.Character_Size(utf8.Character(value))
			encode_limit := count + Template_Name_Count(character_size)
			size := utf8.Encode_Character(
				utf8.Bytes(output[count:encode_limit]), utf8.Character(value),
			)
			count += Template_Name_Count(size)
		}
	}
	return count
}

func quoted_escape_decode(
	source Source, position_value Quoted_Position_Validated,
) (
	_ Quoted_Position_Validated,
	_ Quoted_Escape_Value,
	form Quoted_Escape_Form,
) {
	defer func() { Quoted_Escape_Form_Invariants(form, "quoted_escape_decode.form") }()
	Quoted_Position_Validated_Invariants(position_value, "quoted_escape_decode.position_value")
	Source_Invariants(source, "quoted_escape_decode.source")
	var position Quoted_Position_Validated
	defer func() {
		Quoted_Position_Validated_Invariants(position, "quoted_escape_decode.position")
	}()
	var value Quoted_Escape_Value
	defer func() { Quoted_Escape_Value_Invariants(value, "quoted_escape_decode.value") }()
	position_raw := Source_Position(position_value.Value.(Quoted_Position_Validated_Value))
	data := source.Data.Value
	start := position_raw
	selector := data[start+SOURCE_POSITION_INCREMENT]
	position_raw += Source_Position(QUOTED_ESCAPE_PREFIX_SIZE)
	position = Quoted_Position_Validated{Value: Quoted_Position_Validated_Value(position_raw)}
	switch selector {
	case 'a':
		value, form = '\a', QUOTED_ESCAPE_BYTE
	case 'b':
		value, form = '\b', QUOTED_ESCAPE_BYTE
	case 'f':
		value, form = '\f', QUOTED_ESCAPE_BYTE
	case 'n':
		value, form = '\n', QUOTED_ESCAPE_BYTE
	case 'r':
		value, form = '\r', QUOTED_ESCAPE_BYTE
	case 't':
		value, form = '\t', QUOTED_ESCAPE_BYTE
	case 'v':
		value, form = '\v', QUOTED_ESCAPE_BYTE
	case '\\', '\'', '"':
		value, form = Quoted_Escape_Value(selector), QUOTED_ESCAPE_BYTE
	case 'x':
		position, value, form = quoted_escape_digits(
			source, position, QUOTED_HEX_DIGIT_COUNT,
			Quoted_Escape_Base(NUMBER_BASE_HEXADECIMAL),
			QUOTED_ESCAPE_BYTE,
		)
	case 'u':
		position, value, form = quoted_escape_digits(
			source, position, QUOTED_SHORT_UNICODE_DIGIT_COUNT,
			Quoted_Escape_Base(NUMBER_BASE_HEXADECIMAL), QUOTED_ESCAPE_CHARACTER,
		)
	case 'U':
		position, value, form = quoted_escape_digits(
			source, position, QUOTED_LONG_UNICODE_DIGIT_COUNT,
			Quoted_Escape_Base(NUMBER_BASE_HEXADECIMAL), QUOTED_ESCAPE_CHARACTER,
		)
	default:
		position, value, form = quoted_escape_digits(
			source, Quoted_Position_Validated{
				Value: Quoted_Position_Validated_Value(
					start + SOURCE_POSITION_INCREMENT,
				),
			},
			QUOTED_OCTAL_DIGIT_COUNT,
			Quoted_Escape_Base(NUMBER_BASE_OCTAL), QUOTED_ESCAPE_BYTE,
		)
	}
	return position, value, form
}

func quoted_escape_digits(
	source Source,
	position_value Quoted_Position_Validated,
	digit_count Quoted_Digit_Count,
	base Quoted_Escape_Base,
	form Quoted_Escape_Form,
) (
	_ Quoted_Position_Validated,
	_ Quoted_Escape_Value,
	output_form Quoted_Escape_Form,
) {
	defer func() {
		Quoted_Escape_Form_Invariants(output_form, "quoted_escape_digits.output_form")
	}()
	Quoted_Position_Validated_Invariants(position_value, "quoted_escape_digits.position_value")
	Quoted_Digit_Count_Invariants(digit_count, "quoted_escape_digits.digit_count")
	Quoted_Escape_Base_Invariants(base, "quoted_escape_digits.base")
	Quoted_Escape_Form_Invariants(form, "quoted_escape_digits.form")
	Source_Invariants(source, "quoted_escape_digits.source")
	var position Quoted_Position_Validated
	defer func() {
		Quoted_Position_Validated_Invariants(position, "quoted_escape_digits.position")
	}()
	value := Quoted_Escape_Value(bits.WORD_32_MINIMUM)
	defer func() { Quoted_Escape_Value_Invariants(value, "quoted_escape_digits.value") }()
	output_form = form
	position_raw := Source_Position(position_value.Value.(Quoted_Position_Validated_Value))
	data := source.Data.Value
	index := Source_Position(SOURCE_SIZE_MINIMUM)
	for index < Source_Position(digit_count) {
		character := data[position_raw+index]
		digit := Quoted_Escape_Value(character - '0')
		if character >= 'a' {
			digit = Quoted_Escape_Value(character-'a') +
				Quoted_Escape_Value(NUMBER_BASE_DECIMAL)
		} else if character >= 'A' {
			digit = Quoted_Escape_Value(character-'A') +
				Quoted_Escape_Value(NUMBER_BASE_DECIMAL)
		}
		value = value*Quoted_Escape_Value(base) + digit
		index++
	}
	position = Quoted_Position_Validated{
		Value: Quoted_Position_Validated_Value(
			position_raw + Source_Position(digit_count),
		),
	}
	return position, value, output_form
}

func token_number(
	source Source, cursor_state Token_Cursor_Pointer, lexeme_state Lexeme_Validated,
) (status Lex_Status) {
	defer func() { Lex_Status_Invariants(status, "token_number.status") }()
	Token_Cursor_Pointer_Invariants(cursor_state, "token_number.cursor_state")
	Lexeme_Validated_Invariants(lexeme_state, "token_number.lexeme_state")
	Source_Invariants(source, "token_number.source")
	cursor := (Token_Cursor_Pointer)(cursor_state)
	lexeme := lexeme_state.Value
	position := Source_Position(lexeme.Start)
	data := source.Data.Value
	end := position + SOURCE_POSITION_INCREMENT
	for end < Source_Position(cursor.End.Value) {
		if bool(token_terminator(Input_Byte(data[end]))) {
			break
		}
		end++
	}
	span := Number_Span_Validated{Value: Number_Span_Stored(Number_Span{
		Start: Number_Start(position), End: Number_End(end),
	})}
	if !bool(number_valid(source, span, cursor.Number.Value)) {
		return PARSE_STATUS_SYNTAX_INVALID
	}
	cursor.Position.Value = Cursor_Position_Storage_Value(end)
	cursor.Token.Value = Token_Stored(Token{
		Kind: TOKEN_NUMBER, Start: Token_Start(position), End: Token_End(end),
	})
	return PARSE_STATUS_OK
}

func number_valid(
	source Source,
	span_state Number_Span_Validated,
	workspace Number_Workspace_Pointer,
) (valid Number_Validity) {
	defer func() { Number_Validity_Invariants(valid, "number_valid.valid") }()
	Number_Span_Validated_Invariants(span_state, "number_valid.span_state")
	Source_Invariants(source, "number_valid.source")
	Number_Workspace_Pointer_Invariants(workspace, "number_valid.workspace")
	span := span_state.Value
	data := source.Data.Value
	start, end := Source_Position(span.Start), Source_Position(span.End)
	position_state := Number_Position_Validated{Value: Number_Position_Validated_Value(start)}
	var component_valid Number_Component_Validity
	position_state, component_valid = number_component(source, position_state, span_state)
	if !bool(component_valid) {
		return false
	}
	position := Source_Position(position_state.Value.(Number_Position_Validated_Value))
	first_end := position
	if position == end {
		value_span := Number_Span_Validated{Value: Number_Span_Stored(Number_Span{
			Start: span.Start, End: Number_End(first_end),
		})}

		return Number_Validity(number_value_valid(
			source, value_span, workspace,
			data[first_end-SOURCE_POSITION_INCREMENT] == 'i',
		))
	}
	if data[position-SOURCE_POSITION_INCREMENT] == 'i' {
		return false
	}
	if data[position] != '+' {
		if data[position] != '-' {
			return false
		}
	}
	second_start := position
	position_state, component_valid = number_component(
		source, Number_Position_Validated{
			Value: Number_Position_Validated_Value(position),
		}, span_state,
	)
	position = Source_Position(position_state.Value.(Number_Position_Validated_Value))
	if !bool(component_valid) {
		return false
	}
	if position != end {
		return false
	}
	if data[end-SOURCE_POSITION_INCREMENT] != 'i' {
		return false
	}
	first_span := Number_Span_Validated{Value: Number_Span_Stored(Number_Span{
		Start: span.Start, End: Number_End(first_end),
	})}

	if !bool(number_value_valid(source, first_span, workspace, true)) {
		return false
	}
	second_span := Number_Span_Validated{Value: Number_Span_Stored(Number_Span{
		Start: Number_Start(second_start), End: span.End,
	})}

	return Number_Validity(number_value_valid(
		source, second_span, workspace, true,
	))
}

func number_value_valid(
	source Source,
	span_state Number_Span_Validated,
	workspace Number_Workspace_Pointer,
	float_domain Number_Value_Validity,
) (valid Number_Value_Validity) {
	defer func() {
		Number_Value_Validity_Invariants(valid, "number_value_valid.valid")
	}()
	Number_Span_Validated_Invariants(span_state, "number_value_valid.span_state")
	Source_Invariants(source, "number_value_valid.source")
	Number_Workspace_Pointer_Invariants(workspace, "number_value_valid.workspace")
	Number_Value_Validity_Invariants(float_domain, "number_value_valid.float_domain")
	span := span_state.Value
	data := source.Data.Value
	end := Source_Position(span.End)
	if data[end-SOURCE_POSITION_INCREMENT] == 'i' {
		end--
		span.End = Number_End(end)
	}
	if !bool(float_domain) {
		float_domain = number_has_float_marker(
			source, Number_Span_Validated{Value: Number_Span_Stored(span)},
		)
	}
	if !bool(float_domain) {
		return number_integer_valid(
			source, Number_Span_Validated{Value: Number_Span_Stored(span)},
		)
	}
	return number_float_valid(
		source, Number_Span_Validated{Value: Number_Span_Stored(span)}, workspace,
	)
}

func number_has_float_marker(
	source Source, span_state Number_Span_Validated,
) (found Number_Value_Validity) {
	defer func() {
		Number_Value_Validity_Invariants(found, "number_has_float_marker.found")
	}()
	Number_Span_Validated_Invariants(span_state, "number_has_float_marker.span_state")
	Source_Invariants(source, "number_has_float_marker.source")
	span := span_state.Value
	start, end := Source_Position(span.Start), Source_Position(span.End)
	data := source.Data.Value
	for position := start; position < end; position++ {
		switch data[position] {
		case '.', 'e', 'E', 'p', 'P':
			return true
		}
	}
	return false
}

func number_integer_valid(
	source Source, span_state Number_Span_Validated,
) (valid Number_Value_Validity) {
	defer func() {
		Number_Value_Validity_Invariants(valid, "number_integer_valid.valid")
	}()
	Number_Span_Validated_Invariants(span_state, "number_integer_valid.span_state")
	Source_Invariants(source, "number_integer_valid.source")
	span := span_state.Value
	start, end := Source_Position(span.Start), Source_Position(span.End)
	data := source.Data.Value
	position := start
	negative := false
	signed := false
	if data[position] == '+' {
		signed = true
		position++
	} else if data[position] == '-' {
		signed = true
		negative = true
		position++
	}
	digits := position
	digits_state, base, _ := number_prefix(
		source, Number_Position_Validated{
			Value: Number_Position_Validated_Value(digits),
		}, span_state,
	)
	digits = Source_Position(digits_state.Value.(Number_Position_Validated_Value))
	if digits == position {
		if data[position] == '0' {
			if position+SOURCE_POSITION_INCREMENT < end {
				base = NUMBER_BASE_OCTAL
			}
		}
	}
	limit := uint64(bits.WORD_64_MAXIMUM)
	if signed {
		limit >>= utf8.CHARACTER_SIZE_MINIMUM
		if negative {
			limit++
		}
	}
	value := uint64(bits.WORD_64_MINIMUM)
	for digits < end {
		character := Input_Byte(data[digits])
		digits++
		if character == '_' {
			continue
		}
		digit := uint64(character - '0')
		if character >= 'a' {
			digit = uint64(character-'a') + uint64(NUMBER_BASE_DECIMAL)
		} else if character >= 'A' {
			digit = uint64(character-'A') + uint64(NUMBER_BASE_DECIMAL)
		}
		if digit >= uint64(base) {
			return false
		}
		if value > (limit-digit)/uint64(base) {
			return false
		}
		value = value*uint64(base) + digit
	}
	return true
}

func number_float_valid(
	source Source,
	span_state Number_Span_Validated,
	workspace Number_Workspace_Pointer,
) (valid Number_Value_Validity) {
	defer func() {
		Number_Value_Validity_Invariants(valid, "number_float_valid.valid")
	}()
	Number_Span_Validated_Invariants(span_state, "number_float_valid.span_state")
	Source_Invariants(source, "number_float_valid.source")
	Number_Workspace_Pointer_Invariants(workspace, "number_float_valid.workspace")
	span := span_state.Value
	start, end := Source_Position(span.Start), Source_Position(span.End)
	data := source.Data.Value
	if bool(number_component_zero(source, span_state)) {
		return true
	}
	local_workspace := (*Number_Workspace)(workspace)
	number_workspace := (*big.Float_Parse_Workspace)(local_workspace)
	value := &number_workspace.Values[big.FLOAT_RAT_RESULT_INDEX]
	value.Precision = big.Float_Precision(big.FLOAT_64_VALUE_MANTISSA_BIT_COUNT)
	value.Mode = big.Rounding_Mode(big.ROUND_TO_NEAREST_EVEN)
	_, status := big.Float_Parse(
		value, big.Float_Parse_Text_Unvalidated(data[start:end]),
		big.BASE_AUTOMATIC, number_workspace,
	)
	if status != big.Parse_Status(big.STATUS_OK) {
		if status == big.Parse_Status(big.STATUS_VALUE_OVERFLOW) {
			return number_exponent_negative(source, span_state)
		}
		return false
	}
	encoding, _ := big.Float_Float_64_Bits(value)
	magnitude := uint64(encoding) &^ big.FLOAT_64_SIGN_MASK
	return Number_Value_Validity(magnitude != big.FLOAT_64_POSITIVE_INFINITY_BITS)
}

func number_component_zero(
	source Source, span_state Number_Span_Validated,
) (zero Number_Value_Validity) {
	defer func() {
		Number_Value_Validity_Invariants(zero, "number_component_zero.zero")
	}()
	Number_Span_Validated_Invariants(span_state, "number_component_zero.span_state")
	Source_Invariants(source, "number_component_zero.source")
	span := span_state.Value
	start, end := Source_Position(span.Start), Source_Position(span.End)
	data := source.Data.Value
	position_state := number_sign_end(
		source, Number_Position_Validated{
			Value: Number_Position_Validated_Value(start),
		}, span_state,
	)
	position_state, base, _ := number_prefix(source, position_state, span_state)
	position := Source_Position(position_state.Value.(Number_Position_Validated_Value))
	for position < end {
		character := Input_Byte(data[position])
		if bool(number_exponent_marker(
			source, Number_Position_Validated{
				Value: Number_Position_Validated_Value(position),
			}, span_state, base,
		)) {
			break
		}
		if bool(number_digit(
			source, Number_Position_Validated{
				Value: Number_Position_Validated_Value(position),
			}, span_state, base,
		)) {
			if character != '0' {
				return false
			}
		}
		position++
	}
	return true
}

func number_exponent_negative(
	source Source, span_state Number_Span_Validated,
) (negative Number_Value_Validity) {
	defer func() {
		Number_Value_Validity_Invariants(negative, "number_exponent_negative.negative")
	}()
	Number_Span_Validated_Invariants(span_state, "number_exponent_negative.span_state")
	Source_Invariants(source, "number_exponent_negative.source")
	span := span_state.Value
	start, end := Source_Position(span.Start), Source_Position(span.End)
	data := source.Data.Value
	position_state := number_sign_end(
		source, Number_Position_Validated{
			Value: Number_Position_Validated_Value(start),
		}, span_state,
	)
	position_state, base, _ := number_prefix(source, position_state, span_state)
	position := Source_Position(position_state.Value.(Number_Position_Validated_Value))
	for position < end {
		if bool(number_exponent_marker(
			source, Number_Position_Validated{
				Value: Number_Position_Validated_Value(position),
			}, span_state, base,
		)) {
			position++
			if position < end {
				return Number_Value_Validity(data[position] == '-')
			}
			return false
		}
		position++
	}
	return false
}

func number_component(
	source Source, position_value Number_Position_Validated, span_state Number_Span_Validated,
) (_ Number_Position_Validated, valid Number_Component_Validity) {
	defer func() { Number_Component_Validity_Invariants(valid, "number_component.valid") }()
	Number_Position_Validated_Invariants(position_value, "number_component.position_value")
	Number_Span_Validated_Invariants(span_state, "number_component.span")
	Source_Invariants(source, "number_component.source")
	position := position_value
	defer func() {
		Number_Position_Validated_Invariants(position, "number_component.position")
	}()
	end, data := Source_Position(span_state.Value.End), source.Data.Value
	position = number_sign_end(source, position_value, span_state)
	cursor := Source_Position(position.Value.(Number_Position_Validated_Value))
	if cursor == end {
		return position, false
	}
	position, base, prefix_mode := number_prefix(source, position, span_state)
	cursor = Source_Position(position.Value.(Number_Position_Validated_Value))
	integer_start := cursor
	if data[cursor] != '.' {
		var sequence_valid Number_Sequence_Validity
		position, sequence_valid = number_sequence(
			source, position, span_state, base, Number_Sequence_Mode(prefix_mode),
		)
		cursor = Source_Position(position.Value.(Number_Position_Validated_Value))
		if !bool(sequence_valid) {
			return position, false
		}
	}
	dotted := Number_Fraction_Marker(false)
	if cursor < end {
		if data[cursor] == '.' {
			dotted = true
			cursor++
			mode := Number_Sequence_Mode(NUMBER_SEQUENCE_OPTIONAL)
			if cursor == integer_start+SOURCE_POSITION_INCREMENT {
				mode = NUMBER_SEQUENCE_REQUIRED
			}
			var sequence_valid Number_Sequence_Validity
			position, sequence_valid = number_sequence(
				source, Number_Position_Validated{
					Value: Number_Position_Validated_Value(cursor),
				}, span_state, base, mode,
			)
			cursor = Source_Position(
				position.Value.(Number_Position_Validated_Value),
			)
			if !bool(sequence_valid) {
				return position, false
			}
		}
	}
	position, exponent, exponent_valid := number_exponent(
		source, Number_Position_Validated{
			Value: Number_Position_Validated_Value(cursor),
		}, span_state, base,
	)
	cursor = Source_Position(position.Value.(Number_Position_Validated_Value))
	if !bool(exponent_valid) {
		return position, false
	}
	if !bool(number_hexadecimal_component_valid(base, dotted, exponent)) {
		return position, false
	}
	position = number_suffix_end(source, position, span_state)
	return position, true
}

func number_suffix_end(
	source Source, position_value Number_Position_Validated, span_state Number_Span_Validated,
) (position Number_Position_Validated) {
	defer func() {
		Number_Position_Validated_Invariants(position, "number_suffix_end.position")
	}()
	Source_Invariants(source, "number_suffix_end.source")
	Number_Position_Validated_Invariants(position_value, "number_suffix_end.position_value")
	Number_Span_Validated_Invariants(span_state, "number_suffix_end.span")
	position = position_value
	cursor := Source_Position(position.Value.(Number_Position_Validated_Value))
	if cursor < Source_Position(span_state.Value.End) {
		if source.Data.Value[cursor] == 'i' {
			position.Value = Number_Position_Validated_Value(
				cursor + SOURCE_POSITION_INCREMENT,
			)
		}
	}
	return position
}

func number_hexadecimal_component_valid(
	base Number_Base, dotted Number_Fraction_Marker, exponent Number_Exponent_Marker,
) (valid Number_Component_Validity) {
	defer func() {
		Number_Component_Validity_Invariants(
			valid, "number_hexadecimal_component_valid.valid",
		)
	}()
	Number_Base_Invariants(base, "number_hexadecimal_component_valid.base")
	Number_Fraction_Marker_Invariants(
		dotted, "number_hexadecimal_component_valid.dotted",
	)
	Number_Exponent_Marker_Invariants(
		exponent, "number_hexadecimal_component_valid.exponent",
	)
	if base != NUMBER_BASE_HEXADECIMAL {
		return true
	}
	if !dotted {
		return true
	}
	return Number_Component_Validity(exponent)
}

func number_exponent(
	source Source,
	position_value Number_Position_Validated,
	span_state Number_Span_Validated,
	base Number_Base,
) (
	_ Number_Position_Validated,
	_ Number_Exponent_Marker,
	valid Number_Component_Validity,
) {
	defer func() { Number_Component_Validity_Invariants(valid, "number_exponent.valid") }()
	Source_Invariants(source, "number_exponent.source")
	Number_Position_Validated_Invariants(position_value, "number_exponent.position_value")
	Number_Span_Validated_Invariants(span_state, "number_exponent.span")
	Number_Base_Invariants(base, "number_exponent.base")
	position := position_value
	defer func() {
		Number_Position_Validated_Invariants(position, "number_exponent.position")
	}()
	var exponent Number_Exponent_Marker
	defer func() { Number_Exponent_Marker_Invariants(exponent, "number_exponent.exponent") }()
	span := span_state.Value
	cursor := Source_Position(position_value.Value.(Number_Position_Validated_Value))
	if cursor == Source_Position(span.End) {
		return position, exponent, true
	}
	exponent = number_exponent_marker(source, position_value, span_state, base)
	if !bool(exponent) {
		return position, exponent, true
	}
	cursor++
	position = number_sign_end(
		source, Number_Position_Validated{
			Value: Number_Position_Validated_Value(cursor),
		}, span_state,
	)
	position, sequence_valid := number_sequence(
		source, position, span_state, NUMBER_BASE_DECIMAL, NUMBER_SEQUENCE_REQUIRED,
	)
	return position, exponent, Number_Component_Validity(sequence_valid)
}

func number_sign_end(
	source Source, position_value Number_Position_Validated,
	span_state Number_Span_Validated,
) (position Number_Position_Validated) {
	defer func() {
		Number_Position_Validated_Invariants(position, "number_sign_end.position")
	}()
	Number_Position_Validated_Invariants(position_value, "number_sign_end.position_value")
	Number_Span_Validated_Invariants(span_state, "number_sign_end.span")
	Source_Invariants(source, "number_sign_end.source")
	span := span_state.Value
	cursor := Source_Position(position_value.Value.(Number_Position_Validated_Value))
	if cursor == Source_Position(span.End) {
		return Number_Position_Validated{Value: Number_Position_Validated_Value(cursor)}
	}
	data := source.Data.Value
	if data[cursor] == '+' {
		return Number_Position_Validated{
			Value: Number_Position_Validated_Value(cursor + SOURCE_POSITION_INCREMENT),
		}
	}
	if data[cursor] == '-' {
		return Number_Position_Validated{
			Value: Number_Position_Validated_Value(cursor + SOURCE_POSITION_INCREMENT),
		}
	}
	return Number_Position_Validated{Value: Number_Position_Validated_Value(cursor)}
}

func number_prefix(
	source Source, position_value Number_Position_Validated,
	span_state Number_Span_Validated,
) (_ Number_Position_Validated, _ Number_Base, mode Number_Prefix_Mode) {
	defer func() { Number_Prefix_Mode_Invariants(mode, "number_prefix.mode") }()
	Number_Position_Validated_Invariants(position_value, "number_prefix.position_value")
	Number_Span_Validated_Invariants(span_state, "number_prefix.span")
	Source_Invariants(source, "number_prefix.source")
	position := position_value
	defer func() { Number_Position_Validated_Invariants(position, "number_prefix.position") }()
	base := Number_Base(NUMBER_BASE_DECIMAL)
	defer func() { Number_Base_Invariants(base, "number_prefix.base") }()
	span := span_state.Value
	cursor := Source_Position(position_value.Value.(Number_Position_Validated_Value))
	end := Source_Position(span.End)
	mode = NUMBER_PREFIX_REQUIRED
	if cursor+SOURCE_POSITION_INCREMENT >= end {
		return position, base, mode
	}
	data := source.Data.Value
	if data[cursor] != '0' {
		return position, base, mode
	}
	switch data[cursor+SOURCE_POSITION_INCREMENT] {
	case 'b', 'B':
		base = NUMBER_BASE_BINARY
	case 'o', 'O':
		base = NUMBER_BASE_OCTAL
	case 'x', 'X':
		base = NUMBER_BASE_HEXADECIMAL
	default:
		return position, base, mode
	}
	cursor += Source_Position(NUMBER_PREFIX_SIZE)
	mode = NUMBER_PREFIX_PREFIXED
	position = Number_Position_Validated{Value: Number_Position_Validated_Value(cursor)}
	return position, base, mode
}

func number_sequence(
	source Source,
	position_value Number_Position_Validated,
	span_state Number_Span_Validated,
	base Number_Base,
	mode Number_Sequence_Mode,
) (_ Number_Position_Validated, valid Number_Sequence_Validity) {
	defer func() { Number_Sequence_Validity_Invariants(valid, "number_sequence.valid") }()
	Number_Position_Validated_Invariants(position_value, "number_sequence.position_value")
	Number_Span_Validated_Invariants(span_state, "number_sequence.span")
	Source_Invariants(source, "number_sequence.source")
	Number_Base_Invariants(base, "number_sequence.base")
	Number_Sequence_Mode_Invariants(mode, "number_sequence.mode")
	position := position_value
	defer func() {
		Number_Position_Validated_Invariants(position, "number_sequence.position")
	}()
	span := span_state.Value
	position_raw := Source_Position(position_value.Value.(Number_Position_Validated_Value))
	end := Source_Position(span.End)
	data := source.Data.Value
	start := position_raw
	cursor := start
	digit_seen := false
	previous_digit := false
	for cursor < end {
		character := Input_Byte(data[cursor])
		if character == '_' {
			if !previous_digit {
				if mode != NUMBER_SEQUENCE_PREFIXED {
					position.Value = Number_Position_Validated_Value(cursor)
					return position, false
				}
				if cursor != start {
					position.Value = Number_Position_Validated_Value(cursor)
					return position, false
				}
			}
			previous_digit = false
			cursor++
			continue
		}
		if !bool(number_digit(
			source, Number_Position_Validated{
				Value: Number_Position_Validated_Value(cursor),
			}, span_state, base,
		)) {
			break
		}
		digit_seen = true
		previous_digit = true
		cursor++
	}
	position.Value = Number_Position_Validated_Value(cursor)
	if !digit_seen {
		if mode == NUMBER_SEQUENCE_OPTIONAL {
			return position, true
		}
		return position, false
	}
	if !previous_digit {
		return position, false
	}
	return position, true
}

func number_exponent_marker(
	source Source, position_state Number_Position_Validated,
	span_state Number_Span_Validated,
	base Number_Base,
) (yes Number_Exponent_Marker) {
	defer func() {
		Number_Exponent_Marker_Invariants(yes, "number_exponent_marker.yes")
	}()
	Number_Position_Validated_Invariants(
		position_state, "number_exponent_marker.position_state",
	)
	Number_Span_Validated_Invariants(span_state, "number_exponent_marker.span")
	Number_Base_Invariants(base, "number_exponent_marker.base")
	Source_Invariants(source, "number_exponent_marker.source")
	span := span_state.Value
	position := Source_Position(position_state.Value.(Number_Position_Validated_Value))
	if position >= Source_Position(span.End) {
		return false
	}
	character := Input_Byte(source.Data.Value[position])
	if base == NUMBER_BASE_HEXADECIMAL {
		if character == 'p' {
			return true
		}
		return Number_Exponent_Marker(character == 'P')
	}
	if base != NUMBER_BASE_DECIMAL {
		return false
	}
	if character == 'e' {
		return true
	}
	return Number_Exponent_Marker(character == 'E')
}

func number_digit(
	source Source, position_state Number_Position_Validated,
	span_state Number_Span_Validated,
	base Number_Base,
) (digit Number_Digit) {
	defer func() { Number_Digit_Invariants(digit, "number_digit.digit") }()
	Number_Position_Validated_Invariants(position_state, "number_digit.position_state")
	Number_Span_Validated_Invariants(span_state, "number_digit.span")
	Source_Invariants(source, "number_digit.source")
	Number_Base_Invariants(base, "number_digit.base")
	span := span_state.Value
	position := Source_Position(position_state.Value.(Number_Position_Validated_Value))
	if position >= Source_Position(span.End) {
		return false
	}
	character := Input_Byte(source.Data.Value[position])
	if character >= '0' {
		if character <= '9' {
			return Number_Digit(character-'0' < Input_Byte(base))
		}
	}
	if base != NUMBER_BASE_HEXADECIMAL {
		return false
	}
	if character >= 'a' {
		if character <= 'f' {
			return true
		}
	}
	if character >= 'A' {
		return Number_Digit(character <= 'F')
	}
	return false
}

func token_identifier(
	source Source, cursor_state Token_Cursor_Pointer, lexeme_state Lexeme_Validated,
) {
	Token_Cursor_Pointer_Invariants(cursor_state, "token_identifier.cursor_state")
	Lexeme_Validated_Invariants(lexeme_state, "token_identifier.lexeme_state")
	Source_Invariants(source, "token_identifier.source")
	cursor := (Token_Cursor_Pointer)(cursor_state)
	lexeme := lexeme_state.Value
	position := Source_Position(lexeme.Start)
	limit := Source_Position(cursor.End.Value)
	search := Identifier_Search_Validated{Value: Identifier_Search_Span{
		Position: Identifier_Search_Position_Storage{
			Value: Identifier_Search_Position(position),
		},
		End: Identifier_Search_End_Storage{Value: Identifier_Search_End(limit)},
	}}
	search = identifier_sequence(source, search)
	end := Source_Position(search.Value.Position.Value.(Identifier_Search_Position))
	span := Identifier_Span_Validated{Value: Identifier_Span_Stored(Identifier_Span{
		Start: Identifier_Start(position), End: Identifier_End(end),
	})}

	kind := word_token_kind(source, *cursor, span)
	cursor.Position.Value = Cursor_Position_Storage_Value(end)
	cursor.Token.Value = Token_Stored(Token{
		Kind:  Token_Kind(kind.Value.(Classified_Token_Kind_Value)),
		Start: Token_Start(position), End: Token_End(end),
	})
}

func identifier_sequence(
	source Source, search_state Identifier_Search_Validated,
) (search Identifier_Search_Validated) {
	defer func() {
		Identifier_Search_Validated_Invariants(
			search, "identifier_sequence.search",
		)
	}()
	Source_Invariants(source, "identifier_sequence.source")
	Identifier_Search_Validated_Invariants(
		search_state, "identifier_sequence.search_state",
	)
	search = search_state
	position := search.Value.Position.Value.(Identifier_Search_Position)
	end := search.Value.End.Value.(Identifier_Search_End)
	for uint16(position) < uint16(end) {
		identifier, size := identifier_character_at(source, search)
		if !bool(identifier) {
			return search
		}
		position += Identifier_Search_Position(size)
		search.Value.Position.Value = position
	}
	return search
}

func identifier_character_at(
	source Source, span_state Identifier_Search_Validated,
) (_ Identifier_Character, size Identifier_Size) {
	defer func() { Identifier_Size_Invariants(size, "identifier_character_at.size") }()
	Identifier_Search_Validated_Invariants(
		span_state, "identifier_character_at.span_state",
	)
	Source_Invariants(source, "identifier_character_at.source")
	var identifier Identifier_Character
	defer func() {
		Identifier_Character_Invariants(
			identifier, "identifier_character_at.identifier",
		)
	}()
	span := span_state.Value
	position := span.Position.Value.(Identifier_Search_Position)
	end := span.End.Value.(Identifier_Search_End)
	data := source.Data.Value
	character, decoded_size := utf8.Decode_Character(
		utf8.Bytes(data[position:end]),
	)
	size = Identifier_Size(decoded_size)
	if character < utf8.Decoded_Character(utf8.CHARACTER_SELF) {
		identifier = identifier_character(ASCII_Byte(character))
		return identifier, size
	}
	code_point := ucd.Character(character)
	if bool(ucd.Is_Letter(code_point)) {
		identifier = true
		return identifier, size
	}
	identifier = Identifier_Character(ucd.Is_Digit(code_point))
	return identifier, size
}

func word_token_kind(
	source Source,
	cursor Token_Cursor,
	span_state Identifier_Span_Validated,
) (kind Classified_Token_Kind) {
	defer func() {
		Classified_Token_Kind_Invariants(kind, "word_token_kind.kind")
	}()
	Token_Cursor_Invariants(cursor, "word_token_kind.cursor")
	Identifier_Span_Validated_Invariants(span_state, "word_token_kind.span")
	Source_Invariants(source, "word_token_kind.source")
	if bool(word_matches(source, span_state, KEYWORD_NIL)) {
		return Classified_Token_Kind{Value: Classified_Token_Kind_Value(TOKEN_NIL)}
	}
	if bool(word_matches(source, span_state, KEYWORD_TRUE)) {
		return Classified_Token_Kind{Value: Classified_Token_Kind_Value(TOKEN_BOOLEAN)}
	}
	if bool(word_matches(source, span_state, KEYWORD_FALSE)) {
		return Classified_Token_Kind{Value: Classified_Token_Kind_Value(TOKEN_BOOLEAN)}
	}
	return control_token_kind(source, cursor, span_state)
}

func control_token_kind(
	source Source,
	cursor Token_Cursor,
	span_state Identifier_Span_Validated,
) (kind Classified_Token_Kind) {
	defer func() {
		Classified_Token_Kind_Invariants(kind, "control_token_kind.kind")
	}()
	Token_Cursor_Invariants(cursor, "control_token_kind.cursor")
	Identifier_Span_Validated_Invariants(span_state, "control_token_kind.span")
	Source_Invariants(source, "control_token_kind.source")
	if bool(word_matches(source, span_state, KEYWORD_IF)) {
		return Classified_Token_Kind{Value: Classified_Token_Kind_Value(TOKEN_IF)}
	}
	if bool(word_matches(source, span_state, KEYWORD_RANGE)) {
		return Classified_Token_Kind{Value: Classified_Token_Kind_Value(TOKEN_RANGE)}
	}
	if bool(word_matches(source, span_state, KEYWORD_WITH)) {
		return Classified_Token_Kind{Value: Classified_Token_Kind_Value(TOKEN_WITH)}
	}
	if bool(word_matches(source, span_state, KEYWORD_ELSE)) {
		return Classified_Token_Kind{Value: Classified_Token_Kind_Value(TOKEN_ELSE)}
	}
	if bool(word_matches(source, span_state, KEYWORD_END)) {
		return Classified_Token_Kind{Value: Classified_Token_Kind_Value(TOKEN_END)}
	}
	if bool(word_matches(source, span_state, KEYWORD_BREAK)) {
		if bool(cursor_function_known(source, cursor, span_state)) {
			return Classified_Token_Kind{
				Value: Classified_Token_Kind_Value(TOKEN_IDENTIFIER),
			}
		}
		return Classified_Token_Kind{Value: Classified_Token_Kind_Value(TOKEN_BREAK)}
	}
	if bool(word_matches(source, span_state, KEYWORD_CONTINUE)) {
		if bool(cursor_function_known(source, cursor, span_state)) {
			return Classified_Token_Kind{
				Value: Classified_Token_Kind_Value(TOKEN_IDENTIFIER),
			}
		}
		return Classified_Token_Kind{Value: Classified_Token_Kind_Value(TOKEN_CONTINUE)}
	}
	if bool(word_matches(source, span_state, KEYWORD_TEMPLATE)) {
		return Classified_Token_Kind{Value: Classified_Token_Kind_Value(TOKEN_TEMPLATE)}
	}
	if bool(word_matches(source, span_state, KEYWORD_DEFINE)) {
		return Classified_Token_Kind{Value: Classified_Token_Kind_Value(TOKEN_DEFINE)}
	}
	if bool(word_matches(source, span_state, KEYWORD_BLOCK)) {
		return Classified_Token_Kind{Value: Classified_Token_Kind_Value(TOKEN_BLOCK)}
	}
	return Classified_Token_Kind{Value: Classified_Token_Kind_Value(TOKEN_IDENTIFIER)}
}

func cursor_function_known(
	source Source,
	cursor Token_Cursor,
	span_state Identifier_Span_Validated,
) (known Function_Known) {
	defer func() {
		Function_Known_Invariants(known, "cursor_function_known.known")
	}()
	Token_Cursor_Invariants(cursor, "cursor_function_known.cursor")
	Identifier_Span_Validated_Invariants(
		span_state, "cursor_function_known.span_state",
	)
	Source_Invariants(source, "cursor_function_known.source")
	if cursor.Function.Value == nil {
		return false
	}
	span := span_state.Value
	start, end := Source_Position(span.Start), Source_Position(span.End)
	data := source.Data.Value
	known = cursor.Function.Value(Parse_Function_Name(data[start:end]))
	Function_Known_Invariants(known, "cursor_function_known.callback")
	return known
}

func word_matches(
	source Source,
	span_state Identifier_Span_Validated,
	keyword Keyword,
) (matched Keyword_Match) {
	defer func() { Keyword_Match_Invariants(matched, "word_matches.matched") }()
	Identifier_Span_Validated_Invariants(span_state, "word_matches.span_state")
	Source_Invariants(source, "word_matches.source")
	Keyword_Invariants(keyword, "word_matches.keyword")
	span := span_state.Value
	start, end := Source_Position(span.Start), Source_Position(span.End)
	data := source.Data.Value
	if int(end-start) != len(keyword) {
		return false
	}
	for index := SOURCE_SIZE_MINIMUM; index < len(keyword); index++ {
		if data[start+Source_Position(index)] != keyword[index] {
			return false
		}
	}
	return true
}
func token_terminator(character Input_Byte) (terminates Token_Terminator) {
	defer func() {
		Token_Terminator_Invariants(terminates, "token_terminator.terminates")
	}()
	Input_Byte_Invariants(character, "token_terminator.character")
	if bool(space(character)) {
		return true
	}
	switch character {
	case ',', '|', '(', ')':
		return true
	default:
		return false
	}
}

func identifier_character(
	character ASCII_Byte,
) (identifier Identifier_Character) {
	defer func() {
		Identifier_Character_Invariants(identifier, "identifier_character.identifier")
	}()
	ASCII_Byte_Invariants(character, "identifier_character.character")
	if character >= '0' {
		if character <= '9' {
			return true
		}
	}
	if character == '_' {
		return true
	}
	if character >= 'a' {
		if character <= 'z' {
			return true
		}
	}
	if character >= 'A' {
		if character <= 'Z' {
			return true
		}
	}
	return false
}

func allocate_node(
	workspace_state Syntax_Workspace_Pointer, count_value Node_Count_Allocated,
	node_value Node_Allocation,
) (_ Node_Reference, _ Node_Count_Allocated, status Capacity_Status) {
	defer func() { Capacity_Status_Invariants(status, "allocate_node.status") }()
	Syntax_Workspace_Pointer_Invariants(workspace_state, "allocate_node.workspace_state")
	Node_Count_Allocated_Invariants(count_value, "allocate_node.count_value")
	Node_Allocation_Invariants(node_value, "allocate_node.node_value")
	reference, count := Node_Reference(NO_NODE), count_value
	defer func() { Node_Reference_Invariants(reference, "allocate_node.reference") }()
	defer func() { Node_Count_Allocated_Invariants(count, "allocate_node.count") }()
	count_scalar := count.Value.(Node_Count_Allocated_Value)
	node := node_value.Value.(Node)
	Node_Kind_Invariants(node.Kind, "allocate_node.node.kind")
	Node_Start_Invariants(node.Start, "allocate_node.node.start")
	Node_End_Invariants(node.End, "allocate_node.node.end")
	Node_Value_Start_Invariants(node.Value_Start, "allocate_node.node.value_start")
	Node_Value_End_Invariants(node.Value_End, "allocate_node.node.value_end")
	aver.Always(
		Node_Reference(node.First_Child) <= Node_Reference(count_scalar),
		"A seeded child was allocated before its parent node.",
	)
	aver.Always(
		node.Next_Sibling == Node_Next_Sibling(NO_NODE),
		"Sibling links are created only after node allocation.",
	)
	workspace := (Syntax_Workspace_Pointer)(workspace_state)
	limit, _ := workspace.Node_Limit.Value.(Node_Limit_Storage_Value)
	if Node_Limit(limit) == NODE_LIMIT_DEFAULT {
		limit = Node_Limit_Storage_Value(NODE_COUNT_MAXIMUM)
	}
	if Node_Limit_Storage_Value(count_scalar) == limit {
		return NO_NODE, count, PARSE_STATUS_CAPACITY_EXCEEDED
	}
	workspace.Nodes[count_scalar] = Node(node)
	count_scalar++
	count.Value = count_scalar
	reference = Node_Reference(count_scalar)
	return reference, count, PARSE_STATUS_OK
}

func append_child(
	workspace_state Syntax_Workspace_Pointer, parent_value Node_Link_Validated,
	last_child_value Node_Link_Validated,
	child_value Node_Link_Validated,
) (last_child Node_Link_Validated) {
	defer func() {
		Node_Link_Validated_Invariants(last_child, "append_child.last_child")
	}()
	Syntax_Workspace_Pointer_Invariants(workspace_state, "append_child.workspace_state")
	Node_Link_Validated_Invariants(parent_value, "append_child.parent_value")
	Node_Link_Validated_Invariants(last_child_value, "append_child.last_child_value")
	Node_Link_Validated_Invariants(child_value, "append_child.child_value")
	parent := parent_value.Value.(Node_Link_Validated_Value)
	last_child = last_child_value
	child := child_value.Value.(Node_Link_Validated_Value)
	workspace := (Syntax_Workspace_Pointer)(workspace_state)
	if Node_Reference(last_child.Value.(Node_Link_Validated_Value)) == NO_NODE {
		workspace.Nodes[parent-NODE_COUNT_INCREMENT].First_Child = Node_First_Child(child)
	} else {
		last_reference := last_child.Value.(Node_Link_Validated_Value)
		workspace.Nodes[last_reference-NODE_COUNT_INCREMENT].Next_Sibling =
			Node_Next_Sibling(child)
	}
	return child_value
}

func append_list_child(
	workspace_state Syntax_Workspace_Pointer,
	list_state List_State_Storage_Pointer, child_value Node_Link_Validated,
) {
	Syntax_Workspace_Pointer_Invariants(workspace_state, "append_list_child.workspace_state")
	List_State_Storage_Pointer_Invariants(list_state, "append_list_child.list_state")
	Node_Link_Validated_Invariants(child_value, "append_list_child.child_value")
	list := &list_state.Value
	workspace := (Syntax_Workspace_Pointer)(workspace_state)
	last_child := Node_Link_Validated{
		Value: Node_Link_Validated_Value(Node_Reference(list.Last_Child)),
	}
	last_child = append_child(
		workspace, Node_Link_Validated{Value: Node_Link_Validated_Value(list.List)},
		last_child, child_value,
	)
	list.Last_Child = List_Last_Child(last_child.Value.(Node_Link_Validated_Value))
}

func action_right_index(
	source Source, start_state Source_Cursor_Validated, configuration Configuration,
) (position_state Source_Cursor_Validated) {
	defer func() {
		Source_Cursor_Validated_Invariants(
			position_state, "action_right_index.position_state",
		)
	}()
	Source_Cursor_Validated_Invariants(start_state, "action_right_index.start_state")
	Source_Invariants(source, "action_right_index.source")
	Configuration_Invariants(configuration, "action_right_index.configuration")
	data := source.Data.Value
	start := Source_Position(start_state.Value.(Source_Cursor_Validated_Value))
	position := start
	limit := Source_Position(len(data))
	for position < limit {
		position_state = Source_Cursor_Validated{
			Value: Source_Cursor_Validated_Value(position),
		}
		if delimiter_matches(source, position_state, configuration, false) {
			return position_state
		}
		character := Input_Byte(data[position])
		switch character {
		case '\'', '"', '`':
			span := Action_Scan_Span_Validated{Value: Action_Scan_Span_Stored(
				Action_Scan_Span{
					Start: Action_Scan_Start(position),
					Limit: Action_Scan_Limit(limit),
				})}

			end_state, status := quoted_end(source, span, Quote(character))
			if status != PARSE_STATUS_OK {
				return Source_Cursor_Validated{
					Value: Source_Cursor_Validated_Value(limit),
				}
			}
			end := end_state.Value.(Action_Scan_End_Validated_Value)
			position = Source_Position(end)
		case '/':
			span := Action_Scan_Span_Validated{Value: Action_Scan_Span_Stored(
				Action_Scan_Span{
					Start: Action_Scan_Start(position),
					Limit: Action_Scan_Limit(limit),
				})}

			end_state, status := comment_end(source, span)
			if status != PARSE_STATUS_OK {
				position++
				continue
			}
			end := end_state.Value.(Action_Scan_End_Validated_Value)
			position = Source_Position(end)
		default:
			position++
		}
	}
	return Source_Cursor_Validated{Value: Source_Cursor_Validated_Value(limit)}
}

func comment_end(
	source Source, span_state Action_Scan_Span_Validated,
) (_ Action_Scan_End_Validated, status Lex_Status) {
	defer func() { Lex_Status_Invariants(status, "comment_end.status") }()
	Action_Scan_Span_Validated_Invariants(span_state, "comment_end.span_state")
	Source_Invariants(source, "comment_end.source")
	span := span_state.Value
	start, limit := Source_Position(span.Start), Source_Position(span.Limit)
	end_state := Action_Scan_End_Validated{
		Value: Action_Scan_End_Validated_Value(start),
	}
	defer func() { Action_Scan_End_Validated_Invariants(end_state, "comment_end.end_state") }()
	data := source.Data.Value
	if start+SOURCE_POSITION_INCREMENT >= limit {
		return end_state, PARSE_STATUS_SYNTAX_INVALID
	}
	if data[start] != '/' {
		return end_state, PARSE_STATUS_SYNTAX_INVALID
	}
	if data[start+SOURCE_POSITION_INCREMENT] != '*' {
		return end_state, PARSE_STATUS_SYNTAX_INVALID
	}
	end := start + Source_Position(NUMBER_PREFIX_SIZE)
	for end+SOURCE_POSITION_INCREMENT < limit {
		if data[end] == '*' {
			if data[end+SOURCE_POSITION_INCREMENT] == '/' {
				end += Source_Position(NUMBER_PREFIX_SIZE)
				end_state = Action_Scan_End_Validated{
					Value: Action_Scan_End_Validated_Value(end),
				}
				return end_state, PARSE_STATUS_OK
			}
		}
		end++
	}
	end_state = Action_Scan_End_Validated{
		Value: Action_Scan_End_Validated_Value(end),
	}
	return end_state, PARSE_STATUS_SYNTAX_INVALID
}

func left_delimiter_index(
	source Source,
	start_state Source_Cursor_Validated,
	configuration Configuration,
) (position_state Source_Cursor_Validated) {
	defer func() {
		Source_Cursor_Validated_Invariants(
			position_state, "left_delimiter_index.position_state",
		)
	}()
	Source_Cursor_Validated_Invariants(start_state, "left_delimiter_index.start_state")
	Source_Invariants(source, "left_delimiter_index.source")
	Configuration_Invariants(configuration, "left_delimiter_index.configuration")
	data := source.Data.Value
	start := start_state.Value.(Source_Cursor_Validated_Value)
	for index := int(start); index < len(data); index++ {
		position_state = Source_Cursor_Validated{
			Value: Source_Cursor_Validated_Value(Source_Position(index)),
		}
		if delimiter_matches(source, position_state, configuration, true) {
			return position_state
		}
	}
	return Source_Cursor_Validated{
		Value: Source_Cursor_Validated_Value(Source_Position(len(data))),
	}
}

func delimiter_matches(
	source Source,
	position_state Source_Cursor_Validated,
	configuration Configuration,
	left Left_Delimiter,
) (matched Delimiter_Match) {
	defer func() { Delimiter_Match_Invariants(matched, "delimiter_matches.matched") }()
	Source_Cursor_Validated_Invariants(position_state, "delimiter_matches.position_state")
	Source_Invariants(source, "delimiter_matches.source")
	Configuration_Invariants(configuration, "delimiter_matches.configuration")
	Left_Delimiter_Invariants(left, "delimiter_matches.left")
	position := Source_Position(position_state.Value.(Source_Cursor_Validated_Value))
	data := source.Data.Value
	delimiter := []byte(configuration.Right.Value)
	if bool(left) {
		delimiter = []byte(configuration.Left.Value)
	}
	if len(delimiter) == DELIMITER_SIZE_MINIMUM {
		return default_delimiter_matches(source, position_state, left)
	}
	if int(position)+len(delimiter) > len(data) {
		return false
	}
	for index, character := range delimiter {
		if data[int(position)+index] != character {
			return false
		}
	}
	return true
}

func default_delimiter_matches(
	source Source, position_state Source_Cursor_Validated, left Left_Delimiter,
) (matched Delimiter_Match) {
	defer func() {
		Delimiter_Match_Invariants(matched, "default_delimiter_matches.matched")
	}()
	Source_Cursor_Validated_Invariants(
		position_state, "default_delimiter_matches.position_state",
	)
	Source_Invariants(source, "default_delimiter_matches.source")
	Left_Delimiter_Invariants(left, "default_delimiter_matches.left")
	position := Source_Position(position_state.Value.(Source_Cursor_Validated_Value))
	data := source.Data.Value
	if int(position)+DEFAULT_DELIMITER_SIZE > len(data) {
		return false
	}
	first, second := DEFAULT_RIGHT_DELIMITER_FIRST, DEFAULT_RIGHT_DELIMITER_SECOND
	if bool(left) {
		first, second = DEFAULT_LEFT_DELIMITER_FIRST, DEFAULT_LEFT_DELIMITER_SECOND
	}
	if data[position] != first {
		return false
	}
	return Delimiter_Match(data[position+SOURCE_POSITION_INCREMENT] == second)
}

func delimiter_end(
	position_state Source_Cursor_Validated, configuration Configuration,
	left Left_Delimiter,
) (end_state Source_Cursor_Validated) {
	defer func() {
		Source_Cursor_Validated_Invariants(end_state, "delimiter_end.end_state")
	}()
	Source_Cursor_Validated_Invariants(position_state, "delimiter_end.position_state")
	Configuration_Invariants(configuration, "delimiter_end.configuration")
	Left_Delimiter_Invariants(left, "delimiter_end.left")
	position := Source_Position(position_state.Value.(Source_Cursor_Validated_Value))
	delimiter := []byte(configuration.Right.Value)
	if bool(left) {
		delimiter = []byte(configuration.Left.Value)
	}
	if len(delimiter) == DELIMITER_SIZE_MINIMUM {
		return Source_Cursor_Validated{
			Value: Source_Cursor_Validated_Value(
				position + Source_Position(DEFAULT_DELIMITER_SIZE),
			),
		}

	}
	return Source_Cursor_Validated{
		Value: Source_Cursor_Validated_Value(position + Source_Position(len(delimiter))),
	}
}

func left_trimmed_start(
	source Source, left_state Source_Cursor_Validated, configuration Configuration,
) (_ Trim_Whitespace, start_state Source_Cursor_Validated) {
	defer func() {
		Source_Cursor_Validated_Invariants(
			start_state, "left_trimmed_start.start_state",
		)
	}()
	Source_Cursor_Validated_Invariants(left_state, "left_trimmed_start.left_state")
	Source_Invariants(source, "left_trimmed_start.source")
	Configuration_Invariants(configuration, "left_trimmed_start.configuration")
	var trim Trim_Whitespace
	defer func() { Trim_Whitespace_Invariants(trim, "left_trimmed_start.trim") }()
	data := source.Data.Value
	start_state = delimiter_end(left_state, configuration, true)
	start := Source_Position(start_state.Value.(Source_Cursor_Validated_Value))
	if int(start)+SOURCE_POSITION_INCREMENT >= len(data) {
		return false, start_state
	}
	if data[start] == TRIM_MARKER {
		if bool(space(Input_Byte(data[start+SOURCE_POSITION_INCREMENT]))) {
			trim = true
			start_state = Source_Cursor_Validated{
				Value: Source_Cursor_Validated_Value(
					start + Source_Position(NUMBER_PREFIX_SIZE),
				),
			}
			return trim, start_state
		}
	}
	return false, start_state
}

func right_trimmed_end(
	source Source, start_state Source_Cursor_Validated,
	right_state Source_Cursor_Validated,
) (_ Source_Cursor_Validated, trim Trim_Whitespace) {
	defer func() { Trim_Whitespace_Invariants(trim, "right_trimmed_end.trim") }()
	Source_Cursor_Validated_Invariants(start_state, "right_trimmed_end.start_state")
	Source_Cursor_Validated_Invariants(right_state, "right_trimmed_end.right_state")
	Source_Invariants(source, "right_trimmed_end.source")
	end_state := right_state
	defer func() {
		Source_Cursor_Validated_Invariants(end_state, "right_trimmed_end.end_state")
	}()
	start := Source_Position(start_state.Value.(Source_Cursor_Validated_Value))
	right := Source_Position(right_state.Value.(Source_Cursor_Validated_Value))
	data := source.Data.Value
	if right-start < Source_Position(NUMBER_PREFIX_SIZE) {
		return right_state, false
	}
	if data[right-SOURCE_POSITION_INCREMENT] == TRIM_MARKER {
		if bool(space(Input_Byte(data[right-Source_Position(NUMBER_PREFIX_SIZE)]))) {
			end_state = Source_Cursor_Validated{
				Value: Source_Cursor_Validated_Value(
					right - Source_Position(NUMBER_PREFIX_SIZE),
				),
			}
			trim = true
			return end_state, trim
		}
	}
	return right_state, false
}

func skip_space(
	source Source, start_state Source_Cursor_Validated,
) (end_state Source_Cursor_Validated) {
	defer func() {
		Source_Cursor_Validated_Invariants(end_state, "skip_space.end_state")
	}()
	Source_Cursor_Validated_Invariants(start_state, "skip_space.start_state")
	Source_Invariants(source, "skip_space.source")
	data := source.Data.Value
	end := start_state.Value.(Source_Cursor_Validated_Value)
	for int(end) < len(data) {
		if !bool(space(Input_Byte(data[end]))) {
			break
		}
		end++
	}
	return Source_Cursor_Validated{Value: Source_Cursor_Validated_Value(end)}
}

func trim_space_end(
	source Source, span_state Text_Span_Validated,
) (trimmed Text_Span_Validated) {
	defer func() {
		Text_Span_Validated_Invariants(trimmed, "trim_space_end.trimmed")
	}()
	Text_Span_Validated_Invariants(span_state, "trim_space_end.span_state")
	Source_Invariants(source, "trim_space_end.source")
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

func output_text(
	workspace_state Workspace_Pointer, count_value Output_Count_Staged,
	source Output_Text,
) (_ Output_Count_Staged, status Output_Status) {
	defer func() { Output_Status_Invariants(status, "output_text.status") }()
	Workspace_Pointer_Invariants(workspace_state, "output_text.workspace_state")
	Output_Count_Staged_Invariants(count_value, "output_text.count_value")
	Output_Text_Invariants(source, "output_text.source")
	written := count_value
	defer func() { Output_Count_Staged_Invariants(written, "output_text.written") }()
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

func bytes_equal(left Text, right Text) (
	equal Bytes_Equality,
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
