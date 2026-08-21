// Package glob compiles bounded glob patterns into caller-owned NFAs.
package glob

import (
	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/simulation/aver/default"
	"local/james-orcales/shared/unicode/utf8"
)

// PATTERN_SIZE_MINIMUM admits the empty pattern.
const PATTERN_SIZE_MINIMUM = bytes.SLICE_SIZE_MINIMUM

// PATTERN_SIZE_MAXIMUM follows the repository byte-slice boundary.
const PATTERN_SIZE_MAXIMUM = bytes.SLICE_SIZE_MAXIMUM

// PATTERN_SIZE_UNVALIDATED_MAXIMUM admits the first refused pattern size.
const PATTERN_SIZE_UNVALIDATED_MAXIMUM = PATTERN_SIZE_MAXIMUM + utf8.CHARACTER_SIZE_MINIMUM

// PATTERN_SIZE_NONEMPTY_MINIMUM is one addressable source byte.
const PATTERN_SIZE_NONEMPTY_MINIMUM = PATTERN_SIZE_MINIMUM + utf8.CHARACTER_SIZE_MINIMUM

// PATTERN_INDEX_MAXIMUM is the final byte in maximum validated source.
const PATTERN_INDEX_MAXIMUM = PATTERN_SIZE_MAXIMUM - utf8.CHARACTER_SIZE_MINIMUM

// CLASS_SOURCE_SIZE_MINIMUM contains class opener and one decoded member byte.
const CLASS_SOURCE_SIZE_MINIMUM = len("[a")

// CLASS_CHARACTER_END_MINIMUM follows opener and one decoded member byte.
const CLASS_CHARACTER_END_MINIMUM = CLASS_SOURCE_SIZE_MINIMUM

// TEXT_SIZE_MINIMUM admits the empty candidate.
const TEXT_SIZE_MINIMUM = bytes.SLICE_SIZE_MINIMUM

// TEXT_SIZE_MAXIMUM follows the repository byte-slice boundary.
const TEXT_SIZE_MAXIMUM = bytes.SLICE_SIZE_MAXIMUM

// TEXT_SIZE_UNVALIDATED_MAXIMUM admits the first refused candidate size.
const TEXT_SIZE_UNVALIDATED_MAXIMUM = TEXT_SIZE_MAXIMUM + utf8.CHARACTER_SIZE_MINIMUM

// SEPARATOR_COUNT_MINIMUM admits separator-free matching.
const SEPARATOR_COUNT_MINIMUM = bytes.SLICE_SIZE_MINIMUM

// SEPARATOR_COUNT_MAXIMUM cannot exceed one separator per pattern byte.
const SEPARATOR_COUNT_MAXIMUM = PATTERN_SIZE_MAXIMUM

// SEPARATOR_COUNT_UNVALIDATED_MAXIMUM admits the first refused count.
const SEPARATOR_COUNT_UNVALIDATED_MAXIMUM = SEPARATOR_COUNT_MAXIMUM + utf8.CHARACTER_SIZE_MINIMUM

// PATTERN_DEPTH_MAXIMUM follows the shortest complete brace group.
const PATTERN_DEPTH_MAXIMUM = PATTERN_SIZE_MAXIMUM / len("{}")

// PATTERN_DEPTH_MINIMUM is parser state outside brace alternatives.
const PATTERN_DEPTH_MINIMUM = bytes.SLICE_SIZE_MINIMUM

// PARSER_DEPTH_NONEMPTY_MINIMUM is one suspended brace group.
const PARSER_DEPTH_NONEMPTY_MINIMUM = PATTERN_DEPTH_MINIMUM + utf8.CHARACTER_SIZE_MINIMUM

// CLOSED_PARSER_DEPTH_MAXIMUM follows closing one maximum-depth group.
const CLOSED_PARSER_DEPTH_MAXIMUM = PATTERN_DEPTH_MAXIMUM - utf8.CHARACTER_SIZE_MINIMUM

// NODE_COUNT_MAXIMUM permits one syntax node per byte plus root sequence.
const NODE_COUNT_MAXIMUM = PATTERN_SIZE_MAXIMUM + utf8.CHARACTER_SIZE_MINIMUM

// CLASS_RANGE_COUNT_MAXIMUM spends the remaining bytes after one class shell.
const CLASS_RANGE_COUNT_MAXIMUM = PATTERN_SIZE_MAXIMUM - len("[]")

// CLASS_RANGE_STORAGE_COUNT_MAXIMUM admits a maximum class missing its closing shell.
const CLASS_RANGE_STORAGE_COUNT_MAXIMUM = PATTERN_SIZE_MAXIMUM - len("[")

// INSTRUCTION_COUNT_MAXIMUM permits one operation per byte plus terminal match.
const INSTRUCTION_COUNT_MAXIMUM = PATTERN_SIZE_MAXIMUM + utf8.CHARACTER_SIZE_MINIMUM

// INSTRUCTION_PC_MINIMUM is terminal match instruction index.
const INSTRUCTION_PC_MINIMUM = bytes.SLICE_SIZE_MINIMUM

// INSTRUCTION_PC_MAXIMUM is final populated arena index.
const INSTRUCTION_PC_MAXIMUM = INSTRUCTION_COUNT_MAXIMUM - utf8.CHARACTER_SIZE_MINIMUM

// INSTRUCTION_CONTINUATION_MAXIMUM leaves one slot for current instruction.
const INSTRUCTION_CONTINUATION_MAXIMUM = INSTRUCTION_PC_MAXIMUM - utf8.CHARACTER_SIZE_MINIMUM

// EMITTED_INSTRUCTION_PC_MINIMUM follows the terminal match instruction.
const EMITTED_INSTRUCTION_PC_MINIMUM = INSTRUCTION_PC_MINIMUM + utf8.CHARACTER_SIZE_MINIMUM

// ALTERNATIVE_RESULT_PC_MAXIMUM reserves minimum brace, comma, and branch emissions.
const ALTERNATIVE_RESULT_PC_MAXIMUM = INSTRUCTION_PC_MAXIMUM - len("{,}")

// ALTERNATIVE_UPDATED_PC_MAXIMUM adds one split to an alternative input result.
const ALTERNATIVE_UPDATED_PC_MAXIMUM = ALTERNATIVE_RESULT_PC_MAXIMUM +
	utf8.CHARACTER_SIZE_MINIMUM

// STATE_GENERATION_MINIMUM is the cleared visited-set generation.
const STATE_GENERATION_MINIMUM = bytes.SLICE_SIZE_MINIMUM

// STATE_GENERATION_MAXIMUM covers initial closure plus every candidate byte.
const STATE_GENERATION_MAXIMUM = TEXT_SIZE_MAXIMUM + utf8.CHARACTER_SIZE_MINIMUM

// CLOSURE_GENERATION_MINIMUM is initial NFA closure after cleared generation.
const CLOSURE_GENERATION_MINIMUM = STATE_GENERATION_MINIMUM + utf8.CHARACTER_SIZE_MINIMUM

// CONSUME_GENERATION_MINIMUM is closure after first candidate character.
const CONSUME_GENERATION_MINIMUM = CLOSURE_GENERATION_MINIMUM + utf8.CHARACTER_SIZE_MINIMUM

// ACTIVE_STATE_COUNT_MAXIMUM spends one byte per branch plus terminal continuation.
const ACTIVE_STATE_COUNT_MAXIMUM = PATTERN_DEPTH_MAXIMUM + utf8.CHARACTER_SIZE_MINIMUM

// APPEND_STATE_COUNT_MAXIMUM preserves one active-state slot for the append.
const APPEND_STATE_COUNT_MAXIMUM = ACTIVE_STATE_COUNT_MAXIMUM - utf8.CHARACTER_SIZE_MINIMUM

// CLOSURE_COUNT_MAXIMUM spends one comma per suspended alternative branch.
const CLOSURE_COUNT_MAXIMUM = PATTERN_DEPTH_MAXIMUM - utf8.CHARACTER_SIZE_MINIMUM

// CLOSURE_APPEND_COUNT_MAXIMUM preserves one closure slot for the enqueue.
const CLOSURE_APPEND_COUNT_MAXIMUM = CLOSURE_COUNT_MAXIMUM - utf8.CHARACTER_SIZE_MINIMUM

// QUOTED_SIZE_MAXIMUM follows every pattern byte needing one escape prefix.
const QUOTED_SIZE_MAXIMUM = PATTERN_SIZE_MAXIMUM * len(`\x`)

// WORKSPACE_FIELD is the sole caller workspace pointer slot.
const WORKSPACE_FIELD = bytes.SLICE_SIZE_MINIMUM

// WORKSPACE_FIELD_COUNT fixes hostile workspace pointer storage.
const WORKSPACE_FIELD_COUNT = WORKSPACE_FIELD + utf8.CHARACTER_SIZE_MINIMUM

// PATTERN_WORKSPACE_FIELD is the sole compiled workspace pointer slot.
const PATTERN_WORKSPACE_FIELD = bytes.SLICE_SIZE_MINIMUM

// PATTERN_WORKSPACE_FIELD_COUNT fixes compiled workspace pointer storage.
const PATTERN_WORKSPACE_FIELD_COUNT = PATTERN_WORKSPACE_FIELD + utf8.CHARACTER_SIZE_MINIMUM

// PATTERN_CONTROL_START retains the NFA entry instruction.
const PATTERN_CONTROL_START = bytes.SLICE_SIZE_MINIMUM

// PATTERN_CONTROL_INSTRUCTION_COUNT retains immutable NFA length.
const PATTERN_CONTROL_INSTRUCTION_COUNT = PATTERN_CONTROL_START + utf8.CHARACTER_SIZE_MINIMUM

// PATTERN_CONTROL_SEPARATOR_COUNT retains copied separator length.
const PATTERN_CONTROL_SEPARATOR_COUNT = PATTERN_CONTROL_INSTRUCTION_COUNT +
	utf8.CHARACTER_SIZE_MINIMUM

// PATTERN_CONTROL_COUNT fixes the complete compiled header.
const PATTERN_CONTROL_COUNT = PATTERN_CONTROL_SEPARATOR_COUNT + utf8.CHARACTER_SIZE_MINIMUM

// COMPILE_CONTROL_NODE_COUNT retains the syntax arena cursor.
const COMPILE_CONTROL_NODE_COUNT = bytes.SLICE_SIZE_MINIMUM

// COMPILE_CONTROL_RANGE_COUNT retains the class-range arena cursor.
const COMPILE_CONTROL_RANGE_COUNT = COMPILE_CONTROL_NODE_COUNT + utf8.CHARACTER_SIZE_MINIMUM

// COMPILE_CONTROL_INSTRUCTION_COUNT retains the NFA arena cursor.
const COMPILE_CONTROL_INSTRUCTION_COUNT = COMPILE_CONTROL_RANGE_COUNT + utf8.CHARACTER_SIZE_MINIMUM

// COMPILE_CONTROL_SEPARATOR_COUNT retains copied separator length.
const COMPILE_CONTROL_SEPARATOR_COUNT = COMPILE_CONTROL_INSTRUCTION_COUNT +
	utf8.CHARACTER_SIZE_MINIMUM

// COMPILE_CONTROL_COUNT fixes every compile cursor slot.
const COMPILE_CONTROL_COUNT = COMPILE_CONTROL_SEPARATOR_COUNT + utf8.CHARACTER_SIZE_MINIMUM

// Compile_Status classifies bounded pattern compilation.
type Compile_Status uint8

// Compile_Status_Invariants excludes matcher and output-only refusals.
func Compile_Status_Invariants(
	value Compile_Status, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_4_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_INPUT_INVALID),
			uint8(STATUS_WORKSPACE_INVALID), uint8(STATUS_SYNTAX_INVALID),
		).
		Ensure()
}

// Syntax_Status classifies parser and compiler transitions.
type Syntax_Status uint8

// Syntax_Status_Invariants admits success or malformed bounded syntax.
func Syntax_Status_Invariants(value Syntax_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value), uint8(STATUS_OK), uint8(STATUS_SYNTAX_INVALID)).
		Ensure()
}

// Match_Status classifies bounded NFA execution.
type Match_Status uint8

// Match_Status_Invariants lists matcher-visible refusals.
func Match_Status_Invariants(value Match_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_4_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_INPUT_INVALID),
			uint8(STATUS_WORKSPACE_INVALID), uint8(STATUS_PATTERN_INVALID),
		).
		Ensure()
}

// Pattern_Status classifies validated matcher state.
type Pattern_Status uint8

// Pattern_Status_Invariants admits success or hostile compiled state.
func Pattern_Status_Invariants(value Pattern_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value), uint8(STATUS_OK), uint8(STATUS_PATTERN_INVALID)).
		Ensure()
}

// Quote_Status classifies bounded quote output.
type Quote_Status uint8

// Quote_Status_Invariants lists quoting-visible refusals.
func Quote_Status_Invariants(value Quote_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_INPUT_INVALID),
			uint8(STATUS_OUTPUT_TOO_SMALL),
		).
		Ensure()
}

// STATUS_OK means complete output committed.
const STATUS_OK = bytes.SLICE_SIZE_MINIMUM

// STATUS_INPUT_INVALID refuses hostile input beyond public bounds.
const STATUS_INPUT_INVALID = STATUS_OK + utf8.CHARACTER_SIZE_MINIMUM

// STATUS_WORKSPACE_INVALID reports absent caller storage.
const STATUS_WORKSPACE_INVALID = STATUS_INPUT_INVALID + utf8.CHARACTER_SIZE_MINIMUM

// STATUS_SYNTAX_INVALID reports malformed pattern syntax.
const STATUS_SYNTAX_INVALID = STATUS_WORKSPACE_INVALID + utf8.CHARACTER_SIZE_MINIMUM

// STATUS_OUTPUT_TOO_SMALL preserves short caller output.
const STATUS_OUTPUT_TOO_SMALL = STATUS_SYNTAX_INVALID + utf8.CHARACTER_SIZE_MINIMUM

// STATUS_PATTERN_INVALID refuses forged or stale compiled state.
const STATUS_PATTERN_INVALID = STATUS_OUTPUT_TOO_SMALL + utf8.CHARACTER_SIZE_MINIMUM

// Pattern_Source_Unvalidated is hostile borrowed pattern bytes.
type Pattern_Source_Unvalidated []byte

// Pattern_Source_Unvalidated_Invariants bounds validation witness size.
func Pattern_Source_Unvalidated_Invariants(
	value Pattern_Source_Unvalidated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), PATTERN_SIZE_MINIMUM,
			PATTERN_SIZE_UNVALIDATED_MAXIMUM,
		).
		Ensure()
}

// Pattern_Source excludes the refused oversize witness before parser indexing.
type Pattern_Source []byte

// Pattern_Source_Invariants keeps every parser read inside the public bound.
func Pattern_Source_Invariants(value Pattern_Source, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), PATTERN_SIZE_MINIMUM, PATTERN_SIZE_MAXIMUM).
		Ensure()
}

// Nonempty_Pattern_Source contains at least one parser-dispatched byte.
type Nonempty_Pattern_Source []byte

// Nonempty_Pattern_Source_Invariants excludes the parser-complete empty source.
func Nonempty_Pattern_Source_Invariants(
	value Nonempty_Pattern_Source, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), PATTERN_SIZE_NONEMPTY_MINIMUM, PATTERN_SIZE_MAXIMUM,
		).
		Ensure()
}

// Class_Pattern_Source contains opener and one class member byte.
type Class_Pattern_Source []byte

// Class_Pattern_Source_Invariants follows minimum class-character syntax.
func Class_Pattern_Source_Invariants(
	value Class_Pattern_Source, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), CLASS_SOURCE_SIZE_MINIMUM, PATTERN_SIZE_MAXIMUM).
		Ensure()
}

// Pattern_Position prevents parser cursors from outgrowing validated source storage.
type Pattern_Position int

// Pattern_Position_Invariants shares the exact public pattern bound.
func Pattern_Position_Invariants(value Pattern_Position, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), PATTERN_SIZE_MINIMUM, PATTERN_SIZE_MAXIMUM).
		Ensure()
}

// Pattern_Index is one addressable byte in validated nonempty source.
type Pattern_Index int

// Pattern_Index_Invariants excludes the end boundary.
func Pattern_Index_Invariants(value Pattern_Index, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), PATTERN_SIZE_MINIMUM, PATTERN_INDEX_MAXIMUM).
		Ensure()
}

// Nonzero_Pattern_Position follows at least one consumed source byte.
type Nonzero_Pattern_Position int

// Nonzero_Pattern_Position_Invariants excludes untouched source opening.
func Nonzero_Pattern_Position_Invariants(
	value Nonzero_Pattern_Position, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), PATTERN_SIZE_NONEMPTY_MINIMUM, PATTERN_SIZE_MAXIMUM,
		).
		Ensure()
}

// Class_Character_Index follows a class opener before one decoded member.
type Class_Character_Index int

// Class_Character_Index_Invariants starts after the class opener.
func Class_Character_Index_Invariants(
	value Class_Character_Index, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), PATTERN_SIZE_NONEMPTY_MINIMUM, PATTERN_INDEX_MAXIMUM,
		).
		Ensure()
}

// Class_Character_End follows opener and one decoded member.
type Class_Character_End int

// Class_Character_End_Invariants retains a consumed class character boundary.
func Class_Character_End_Invariants(
	value Class_Character_End, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), CLASS_CHARACTER_END_MINIMUM, PATTERN_SIZE_MAXIMUM,
		).
		Ensure()
}

// Parser_Depth charges one frame per still-open brace group.
type Parser_Depth int

// Parser_Depth_Invariants follows the shortest complete brace-group formula.
func Parser_Depth_Invariants(value Parser_Depth, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), PATTERN_DEPTH_MINIMUM, PATTERN_DEPTH_MAXIMUM).
		Ensure()
}

// Open_Parser_Depth retains at least one suspended brace group.
type Open_Parser_Depth int

// Open_Parser_Depth_Invariants excludes parser state outside groups.
func Open_Parser_Depth_Invariants(
	value Open_Parser_Depth, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), PARSER_DEPTH_NONEMPTY_MINIMUM, PATTERN_DEPTH_MAXIMUM,
		).
		Ensure()
}

// Opened_Parser_Depth follows one successful group opening or full-depth refusal.
type Opened_Parser_Depth int

// Opened_Parser_Depth_Invariants excludes group-free parser state.
func Opened_Parser_Depth_Invariants(
	value Opened_Parser_Depth, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), PARSER_DEPTH_NONEMPTY_MINIMUM, PATTERN_DEPTH_MAXIMUM,
		).
		Ensure()
}

// Closed_Parser_Depth follows one closed group.
type Closed_Parser_Depth int

// Closed_Parser_Depth_Invariants excludes impossible unchanged maximum depth.
func Closed_Parser_Depth_Invariants(
	value Closed_Parser_Depth, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), PATTERN_DEPTH_MINIMUM, CLOSED_PARSER_DEPTH_MAXIMUM,
		).
		Ensure()
}

// Text_Unvalidated is hostile borrowed candidate bytes.
type Text_Unvalidated []byte

// Text_Unvalidated_Invariants bounds validation witness size.
func Text_Unvalidated_Invariants(
	value Text_Unvalidated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), TEXT_SIZE_MINIMUM, TEXT_SIZE_UNVALIDATED_MAXIMUM,
		).
		Ensure()
}

// Separators_Unvalidated is hostile caller separator storage.
type Separators_Unvalidated []rune

// Separators_Unvalidated_Invariants bounds separator validation work.
func Separators_Unvalidated_Invariants(
	value Separators_Unvalidated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), SEPARATOR_COUNT_MINIMUM,
			SEPARATOR_COUNT_UNVALIDATED_MAXIMUM,
		).
		Ensure()
}

// Output is caller-owned escaped output.
type Output []byte

// Output_Invariants follows maximum escaped pattern size.
func Output_Invariants(value Output, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), bytes.SLICE_SIZE_MINIMUM, QUOTED_SIZE_MAXIMUM).
		Ensure()
}

// Output_Count is committed escaped byte count.
type Output_Count uint16

// Output_Count_Invariants follows maximum escaped pattern size.
func Output_Count_Invariants(value Output_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(bytes.SLICE_SIZE_MINIMUM),
			uint16(QUOTED_SIZE_MAXIMUM),
		).
		Ensure()
}

// Diagnostic_Position is first malformed pattern boundary.
type Diagnostic_Position uint16

// Diagnostic_Position_Invariants follows bounded pattern source.
func Diagnostic_Position_Invariants(
	value Diagnostic_Position, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(PATTERN_SIZE_MINIMUM),
			uint16(PATTERN_SIZE_MAXIMUM),
		).
		Ensure()
}

// Diagnostic carries scalar refusal without allocated text.
type Diagnostic struct {
	// Code keeps caller control flow independent from text.
	Code Compile_Status
	// Position locates first certain refusal boundary.
	Position Diagnostic_Position
}

// Diagnostic_Invariants composes refusal code and position.
func Diagnostic_Invariants(value Diagnostic, namespace aver.Namespace) {
	Compile_Status_Invariants(value.Code, namespace)
	Diagnostic_Position_Invariants(value.Position, namespace)
}

// Matched reports complete-pattern match.
type Matched bool

// Matched_Invariants covers matching and rejected candidates.
func Matched_Invariants(value Matched, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Compiled glob matches candidate.").
		Ensure()
}

// Character is one possible metacharacter byte.
type Character byte

// Character_Invariants covers complete byte domain.
func Character_Invariants(value Character, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), bits.WORD_8_MINIMUM, bits.WORD_8_MAXIMUM).
		Ensure()
}

// Special_Character reports glob metacharacter membership.
type Special_Character bool

// Special_Character_Invariants covers ordinary and quoted bytes.
func Special_Character_Invariants(
	value Special_Character, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Byte is glob metacharacter.").
		Ensure()
}

// Node_Kind selects one flat syntax payload.
type Node_Kind uint8

// Node_Kind_Invariants covers complete syntax union.
func Node_Kind_Invariants(value Node_Kind, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), uint8(NODE_SEQUENCE), uint8(NODE_CLASS)).
		Ensure()
}

// NODE_SEQUENCE retains ordered syntax children.
const NODE_SEQUENCE Node_Kind = Node_Kind(bytes.SLICE_SIZE_MINIMUM)

// NODE_ALTERNATIVE retains brace branches.
const NODE_ALTERNATIVE Node_Kind = NODE_SEQUENCE + utf8.CHARACTER_SIZE_MINIMUM

// NODE_LITERAL retains one decoded character.
const NODE_LITERAL Node_Kind = NODE_ALTERNATIVE + utf8.CHARACTER_SIZE_MINIMUM

// NODE_STAR retains a separator-bounded repetition.
const NODE_STAR Node_Kind = NODE_LITERAL + utf8.CHARACTER_SIZE_MINIMUM

// NODE_SUPER_STAR retains an unbounded-separator repetition.
const NODE_SUPER_STAR Node_Kind = NODE_STAR + utf8.CHARACTER_SIZE_MINIMUM

// NODE_SINGLE retains one separator-bounded wildcard.
const NODE_SINGLE Node_Kind = NODE_SUPER_STAR + utf8.CHARACTER_SIZE_MINIMUM

// NODE_CLASS retains one decoded character class.
const NODE_CLASS Node_Kind = NODE_SINGLE + utf8.CHARACTER_SIZE_MINIMUM

// Atom_Kind selects one nonclass parser atom.
type Atom_Kind uint8

// Atom_Kind_Invariants lists exactly the atoms accepted by atom_append.
func Atom_Kind_Invariants(value Atom_Kind, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_4_Uint8(
			uint8(value), uint8(NODE_LITERAL), uint8(NODE_STAR),
			uint8(NODE_SUPER_STAR), uint8(NODE_SINGLE),
		).
		Ensure()
}

// ATOM_LITERAL retains one decoded character.
const ATOM_LITERAL Atom_Kind = Atom_Kind(NODE_LITERAL)

// ATOM_STAR retains separator-bounded repetition.
const ATOM_STAR Atom_Kind = Atom_Kind(NODE_STAR)

// ATOM_SUPER_STAR retains unbounded-separator repetition.
const ATOM_SUPER_STAR Atom_Kind = Atom_Kind(NODE_SUPER_STAR)

// ATOM_SINGLE retains one separator-bounded wildcard.
const ATOM_SINGLE Atom_Kind = Atom_Kind(NODE_SINGLE)

// ATOM_CHARACTER_EMPTY clears payload for nonliteral atoms.
const ATOM_CHARACTER_EMPTY = utf8.Decoded_Character(utf8.DECODED_CHARACTER_MINIMUM)

// Leaf_Kind selects one syntax node that emits one consuming instruction.
type Leaf_Kind uint8

// Leaf_Kind_Invariants excludes sequence and alternative containers.
func Leaf_Kind_Invariants(value Leaf_Kind, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), uint8(NODE_LITERAL), uint8(NODE_CLASS)).
		Ensure()
}

// Node_Reference is one-based syntax slot or absent.
type Node_Reference uint16

// Node_Reference_Invariants includes absent and complete arena boundary.
func Node_Reference_Invariants(
	value Node_Reference, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(NODE_NONE), uint16(NODE_COUNT_MAXIMUM),
		).
		Ensure()
}

// NODE_NONE is the absent one-based syntax reference.
const NODE_NONE Node_Reference = Node_Reference(bytes.SLICE_SIZE_MINIMUM)

// NODE_ROOT is the first one-based syntax reference.
const NODE_ROOT Node_Reference = NODE_NONE + utf8.CHARACTER_SIZE_MINIMUM

// NODE_FIRST_ALTERNATIVE is the only reference excluded from parser containers.
const NODE_FIRST_ALTERNATIVE Node_Reference = NODE_ROOT + utf8.CHARACTER_SIZE_MINIMUM

// NESTED_SEQUENCE_REFERENCE_MINIMUM follows root and first alternative.
const NESTED_SEQUENCE_REFERENCE_MINIMUM = NODE_FIRST_ALTERNATIVE + utf8.CHARACTER_SIZE_MINIMUM

// BRANCH_SEQUENCE_REFERENCE_MINIMUM follows first nested sequence.
const BRANCH_SEQUENCE_REFERENCE_MINIMUM = NESTED_SEQUENCE_REFERENCE_MINIMUM +
	utf8.CHARACTER_SIZE_MINIMUM

// SUSPENDED_SEQUENCE_REFERENCE_MAXIMUM leaves one alternative and sequence pair.
const SUSPENDED_SEQUENCE_REFERENCE_MAXIMUM = NODE_COUNT_MAXIMUM - len("{}")

// CHILD_PARENT_REFERENCE_MAXIMUM leaves one arena slot for its child.
const CHILD_PARENT_REFERENCE_MAXIMUM = NODE_COUNT_MAXIMUM - utf8.CHARACTER_SIZE_MINIMUM

// ROOT_NODE_REFERENCE is the formula-derived parser root value.
const ROOT_NODE_REFERENCE Root_Node_Reference = Root_Node_Reference(NODE_ROOT)

// Root_Node_Reference is the fixed syntax root created before parsing input.
type Root_Node_Reference uint16

// Root_Node_Reference_Invariants fixes the formula-derived root slot.
func Root_Node_Reference_Invariants(
	value Root_Node_Reference, _ aver.Namespace,
) {
	aver.Always(
		uint16(value) == uint16(ROOT_NODE_REFERENCE),
		"Parsed syntax root occupies the first one-based node slot.",
	)
}

// Container_Node_Reference is root or one nested sequence.
type Container_Node_Reference uint16

// Container_Node_Reference_Invariants excludes the first alternative slot.
func Container_Node_Reference_Invariants(
	value Container_Node_Reference, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Holed_Uint16(
			uint16(value), uint16(NODE_ROOT), uint16(NODE_COUNT_MAXIMUM),
			uint16(NODE_FIRST_ALTERNATIVE), uint16(NODE_FIRST_ALTERNATIVE),
			uint16(NODE_FIRST_ALTERNATIVE),
		).
		Ensure()
}

// Nested_Sequence_Reference is one sequence inside an open group.
type Nested_Sequence_Reference uint16

// Nested_Sequence_Reference_Invariants excludes root and first alternative.
func Nested_Sequence_Reference_Invariants(
	value Nested_Sequence_Reference, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(NESTED_SEQUENCE_REFERENCE_MINIMUM),
			uint16(NODE_COUNT_MAXIMUM),
		).
		Ensure()
}

// Opened_Sequence_Reference is a nested sequence or unchanged full-arena current.
type Opened_Sequence_Reference uint16

// Opened_Sequence_Reference_Invariants follows nested sequence references.
func Opened_Sequence_Reference_Invariants(
	value Opened_Sequence_Reference, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(NESTED_SEQUENCE_REFERENCE_MINIMUM),
			uint16(NODE_COUNT_MAXIMUM),
		).
		Ensure()
}

// Branch_Sequence_Reference follows one alternative and its first sequence.
type Branch_Sequence_Reference uint16

// Branch_Sequence_Reference_Invariants starts at the first branch-created sequence.
func Branch_Sequence_Reference_Invariants(
	value Branch_Sequence_Reference, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(BRANCH_SEQUENCE_REFERENCE_MINIMUM),
			uint16(NODE_COUNT_MAXIMUM),
		).
		Ensure()
}

// Suspended_Sequence_Reference is one parent saved before an open group pair.
type Suspended_Sequence_Reference uint16

// Suspended_Sequence_Reference_Invariants reserves the opened node pair.
func Suspended_Sequence_Reference_Invariants(
	value Suspended_Sequence_Reference, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Holed_Uint16(
			uint16(value), uint16(NODE_ROOT),
			uint16(SUSPENDED_SEQUENCE_REFERENCE_MAXIMUM),
			uint16(NODE_FIRST_ALTERNATIVE), uint16(NODE_FIRST_ALTERNATIVE),
			uint16(NODE_FIRST_ALTERNATIVE),
		).
		Ensure()
}

// Child_Node_Reference is one created nonroot syntax node.
type Child_Node_Reference uint16

// Child_Node_Reference_Invariants excludes absent and root references.
func Child_Node_Reference_Invariants(
	value Child_Node_Reference, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(NODE_FIRST_ALTERNATIVE),
			uint16(NODE_COUNT_MAXIMUM),
		).
		Ensure()
}

// Child_Parent_Reference retains one free arena slot for the created child.
type Child_Parent_Reference uint16

// Child_Parent_Reference_Invariants excludes absent and full-arena parent state.
func Child_Parent_Reference_Invariants(
	value Child_Parent_Reference, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(NODE_ROOT),
			uint16(CHILD_PARENT_REFERENCE_MAXIMUM),
		).
		Ensure()
}

// First_Child_Reference is absent or one nonroot child node.
type First_Child_Reference uint16

// First_Child_Reference_Invariants excludes root as its own child.
func First_Child_Reference_Invariants(
	value First_Child_Reference, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Holed_Uint16(
			uint16(value), uint16(NODE_NONE), uint16(NODE_COUNT_MAXIMUM),
			uint16(NODE_ROOT), uint16(NODE_ROOT), uint16(NODE_ROOT),
		).
		Ensure()
}

// Last_Child_Reference is absent or one nonroot child node.
type Last_Child_Reference uint16

// Last_Child_Reference_Invariants excludes root as its own child.
func Last_Child_Reference_Invariants(
	value Last_Child_Reference, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Holed_Uint16(
			uint16(value), uint16(NODE_NONE), uint16(NODE_COUNT_MAXIMUM),
			uint16(NODE_ROOT), uint16(NODE_ROOT), uint16(NODE_ROOT),
		).
		Ensure()
}

// Next_Sibling_Reference is absent or follows one earlier created child.
type Next_Sibling_Reference uint16

// Next_Sibling_Reference_Invariants excludes absent predecessors and root.
func Next_Sibling_Reference_Invariants(
	value Next_Sibling_Reference, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Holed_Uint16(
			uint16(value), uint16(NODE_NONE), uint16(NODE_COUNT_MAXIMUM),
			uint16(NODE_ROOT), uint16(NODE_FIRST_ALTERNATIVE),
			uint16(NODE_FIRST_ALTERNATIVE),
		).
		Ensure()
}

// Previous_Sibling_Reference is absent or one earlier nonroot child.
type Previous_Sibling_Reference uint16

// Previous_Sibling_Reference_Invariants leaves one later sibling arena slot.
func Previous_Sibling_Reference_Invariants(
	value Previous_Sibling_Reference, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Holed_Uint16(
			uint16(value), uint16(NODE_NONE),
			uint16(CHILD_PARENT_REFERENCE_MAXIMUM),
			uint16(NODE_ROOT), uint16(NODE_ROOT), uint16(NODE_ROOT),
		).
		Ensure()
}

// Node_Links stores named child and sibling references.
type Node_Links struct {
	// First_Child opens ordered children.
	First_Child First_Child_Reference
	// Last_Child closes ordered children.
	Last_Child Last_Child_Reference
	// Next_Sibling advances compiler traversal.
	Next_Sibling Next_Sibling_Reference
	// Previous_Sibling reverses compiler traversal.
	Previous_Sibling Previous_Sibling_Reference
}

// Node_Links_Invariants composes exact link roles.
func Node_Links_Invariants(value Node_Links, namespace aver.Namespace) {
	First_Child_Reference_Invariants(value.First_Child, namespace)
	Last_Child_Reference_Invariants(value.Last_Child, namespace)
	Next_Sibling_Reference_Invariants(value.Next_Sibling, namespace)
	Previous_Sibling_Reference_Invariants(value.Previous_Sibling, namespace)
}

// NODE_CLASS_RANGE_INDEX retains first range arena slot.
const NODE_CLASS_RANGE_INDEX = bytes.SLICE_SIZE_MINIMUM

// NODE_CLASS_RANGE_COUNT retains range length.
const NODE_CLASS_RANGE_COUNT = NODE_CLASS_RANGE_INDEX + utf8.CHARACTER_SIZE_MINIMUM

// NODE_CLASS_NEGATED retains class polarity.
const NODE_CLASS_NEGATED = NODE_CLASS_RANGE_COUNT + utf8.CHARACTER_SIZE_MINIMUM

// NODE_CLASS_FIELD_COUNT fixes every class metadata slot.
const NODE_CLASS_FIELD_COUNT = NODE_CLASS_NEGATED + utf8.CHARACTER_SIZE_MINIMUM

// Node_Class stores class opening, count, and negation.
type Node_Class [NODE_CLASS_FIELD_COUNT]uint16

// Node_Class_Invariants fixes class metadata shape.
func Node_Class_Invariants(value Node_Class, _ aver.Namespace) {
	aver.Always(
		len(value) == NODE_CLASS_FIELD_COUNT,
		"Syntax class has complete range metadata.",
	)
}

// NODE_CHARACTER_FIELD is the sole literal payload slot.
const NODE_CHARACTER_FIELD = bytes.SLICE_SIZE_MINIMUM

// NODE_CHARACTER_FIELD_COUNT fixes literal payload storage.
const NODE_CHARACTER_FIELD_COUNT = NODE_CHARACTER_FIELD + utf8.CHARACTER_SIZE_MINIMUM

// Node_Character stores one decoded literal.
type Node_Character [NODE_CHARACTER_FIELD_COUNT]rune

// Node_Character_Invariants fixes literal payload shape.
func Node_Character_Invariants(value Node_Character, _ aver.Namespace) {
	aver.Always(
		len(value) == NODE_CHARACTER_FIELD_COUNT,
		"Syntax literal has one decoded character field.",
	)
}

// Node stores one syntax atom or ordered container.
type Node struct {
	// Kind selects active payload.
	Kind Node_Kind
	// Character stores literal payload.
	Character Node_Character
	// Class stores class payload.
	Class Node_Class
	// Links store child and sibling structure.
	Links Node_Links
}

// Node_Invariants composes bounded syntax record.
func Node_Invariants(value Node, namespace aver.Namespace) {
	Node_Kind_Invariants(value.Kind, namespace)
	Node_Character_Invariants(value.Character, namespace)
	Node_Class_Invariants(value.Class, namespace)
	Node_Links_Invariants(value.Links, namespace)
}

// Leaf_Node is one parsed atom after compiler container dispatch.
type Leaf_Node struct {
	// Kind selects the atomic payload.
	Kind Leaf_Kind
	// Character stores literal payload.
	Character Node_Character
	// Class stores class payload.
	Class Node_Class
}

// Leaf_Node_Invariants composes one compiler leaf.
func Leaf_Node_Invariants(value Leaf_Node, namespace aver.Namespace) {
	Leaf_Kind_Invariants(value.Kind, namespace)
	Node_Character_Invariants(value.Character, namespace)
	Node_Class_Invariants(value.Class, namespace)
}

// Nodes is fixed caller-owned syntax arena.
type Nodes [NODE_COUNT_MAXIMUM]Node

// Nodes_Invariants fixes complete parser capacity.
func Nodes_Invariants(value Nodes, _ aver.Namespace) {
	aver.Always(
		len(value) == NODE_COUNT_MAXIMUM,
		"Glob syntax arena has complete formula-derived capacity.",
	)
}

// PARSER_ALTERNATIVE_REFERENCE_MAXIMUM leaves one nested sequence slot.
const PARSER_ALTERNATIVE_REFERENCE_MAXIMUM = NODE_COUNT_MAXIMUM -
	utf8.CHARACTER_SIZE_MINIMUM

// Parser_Alternative_Reference is one alternative container inside a group.
type Parser_Alternative_Reference uint16

// Parser_Alternative_Reference_Invariants reserves its first sequence child.
func Parser_Alternative_Reference_Invariants(
	value Parser_Alternative_Reference, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(NODE_FIRST_ALTERNATIVE),
			uint16(PARSER_ALTERNATIVE_REFERENCE_MAXIMUM),
		).
		Ensure()
}

// Parser_References stores named group continuations.
type Parser_References struct {
	// Parent resumes the containing sequence.
	Parent Suspended_Sequence_Reference
	// Alternative owns every branch sequence.
	Alternative Parser_Alternative_Reference
	// Sequence is the active branch.
	Sequence Opened_Sequence_Reference
}

// Parser_References_Invariants composes exact continuation roles.
func Parser_References_Invariants(
	value Parser_References, namespace aver.Namespace,
) {
	Suspended_Sequence_Reference_Invariants(value.Parent, namespace)
	Parser_Alternative_Reference_Invariants(value.Alternative, namespace)
	Opened_Sequence_Reference_Invariants(value.Sequence, namespace)
}

// Parser_Frame suspends one brace group without recursion.
type Parser_Frame struct {
	// References retain group continuation.
	References Parser_References
}

// Parser_Frame_Invariants composes fixed parser continuation.
func Parser_Frame_Invariants(value Parser_Frame, namespace aver.Namespace) {
	Parser_References_Invariants(value.References, namespace)
}

// Parser_Frames is fixed brace-depth stack.
type Parser_Frames [PATTERN_DEPTH_MAXIMUM]Parser_Frame

// Parser_Frames_Invariants fixes valid nesting capacity.
func Parser_Frames_Invariants(value Parser_Frames, _ aver.Namespace) {
	aver.Always(
		len(value) == PATTERN_DEPTH_MAXIMUM,
		"Brace parser stack covers maximum valid nesting.",
	)
}

// Class_Low stores inclusive range opening.
type Class_Low rune

// Class_Low_Invariants covers decoder output.
func Class_Low_Invariants(value Class_Low, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(
			int32(value), utf8.DECODED_CHARACTER_MINIMUM,
			utf8.DECODED_CHARACTER_MAXIMUM,
		).
		Ensure()
}

// Class_High stores inclusive range closing.
type Class_High rune

// Class_High_Invariants covers decoder output.
func Class_High_Invariants(value Class_High, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(
			int32(value), utf8.DECODED_CHARACTER_MINIMUM,
			utf8.DECODED_CHARACTER_MAXIMUM,
		).
		Ensure()
}

// Class_Range stores one inclusive decoded interval.
type Class_Range struct {
	// Low opens interval.
	Low Class_Low
	// High closes interval.
	High Class_High
}

// Class_Range_Invariants composes decoded interval boundaries.
func Class_Range_Invariants(value Class_Range, namespace aver.Namespace) {
	Class_Low_Invariants(value.Low, namespace)
	Class_High_Invariants(value.High, namespace)
}

// Class_Ranges is fixed caller-owned hostile parse storage.
type Class_Ranges [CLASS_RANGE_STORAGE_COUNT_MAXIMUM]Class_Range

// Class_Ranges_Invariants fixes complete class capacity.
func Class_Ranges_Invariants(value Class_Ranges, _ aver.Namespace) {
	aver.Always(
		len(value) == CLASS_RANGE_STORAGE_COUNT_MAXIMUM,
		"Class range arena has complete pattern-derived capacity.",
	)
}

// Instruction_Kind selects one NFA operation.
type Instruction_Kind uint8

// Instruction_Kind_Invariants covers complete VM operation set.
func Instruction_Kind_Invariants(
	value Instruction_Kind, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint8(
			uint8(value), uint8(INSTRUCTION_MATCH), uint8(INSTRUCTION_SPLIT),
		).
		Ensure()
}

// INSTRUCTION_MATCH accepts complete candidate consumption.
const INSTRUCTION_MATCH Instruction_Kind = Instruction_Kind(bytes.SLICE_SIZE_MINIMUM)

// INSTRUCTION_LITERAL consumes one equal decoded character.
const INSTRUCTION_LITERAL Instruction_Kind = INSTRUCTION_MATCH + utf8.CHARACTER_SIZE_MINIMUM

// INSTRUCTION_STAR consumes nonseparator characters or nothing.
const INSTRUCTION_STAR Instruction_Kind = INSTRUCTION_LITERAL + utf8.CHARACTER_SIZE_MINIMUM

// INSTRUCTION_SUPER_STAR consumes any characters or nothing.
const INSTRUCTION_SUPER_STAR Instruction_Kind = INSTRUCTION_STAR + utf8.CHARACTER_SIZE_MINIMUM

// INSTRUCTION_SINGLE consumes one nonseparator character.
const INSTRUCTION_SINGLE Instruction_Kind = INSTRUCTION_SUPER_STAR + utf8.CHARACTER_SIZE_MINIMUM

// INSTRUCTION_CLASS consumes one class member.
const INSTRUCTION_CLASS Instruction_Kind = INSTRUCTION_SINGLE + utf8.CHARACTER_SIZE_MINIMUM

// INSTRUCTION_SPLIT exposes both alternative branches.
const INSTRUCTION_SPLIT Instruction_Kind = INSTRUCTION_CLASS + utf8.CHARACTER_SIZE_MINIMUM

// Instruction_PC selects one NFA instruction.
type Instruction_PC uint16

// Instruction_PC_Invariants follows complete instruction arena.
func Instruction_PC_Invariants(value Instruction_PC, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(INSTRUCTION_PC_MINIMUM),
			uint16(INSTRUCTION_PC_MAXIMUM),
		).
		Ensure()
}

// Instruction_Continuation leaves one emission slot after its target.
type Instruction_Continuation uint16

// Instruction_Continuation_Invariants excludes the already-final arena slot.
func Instruction_Continuation_Invariants(
	value Instruction_Continuation, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(INSTRUCTION_PC_MINIMUM),
			uint16(INSTRUCTION_CONTINUATION_MAXIMUM),
		).
		Ensure()
}

// Emitted_Instruction_PC follows the pre-emitted terminal match instruction.
type Emitted_Instruction_PC uint16

// Emitted_Instruction_PC_Invariants excludes terminal match slot zero.
func Emitted_Instruction_PC_Invariants(
	value Emitted_Instruction_PC, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(EMITTED_INSTRUCTION_PC_MINIMUM),
			uint16(INSTRUCTION_PC_MAXIMUM),
		).
		Ensure()
}

// Alternative_Result_PC retains emission cursor before one alternative transition.
type Alternative_Result_PC uint16

// Alternative_Result_PC_Invariants reserves minimum alternative syntax emissions.
func Alternative_Result_PC_Invariants(
	value Alternative_Result_PC, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(INSTRUCTION_PC_MINIMUM),
			uint16(ALTERNATIVE_RESULT_PC_MAXIMUM),
		).
		Ensure()
}

// Alternative_Updated_PC includes one emitted alternative split.
type Alternative_Updated_PC uint16

// Alternative_Updated_PC_Invariants reserves remaining enclosing emission slots.
func Alternative_Updated_PC_Invariants(
	value Alternative_Updated_PC, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(INSTRUCTION_PC_MINIMUM),
			uint16(ALTERNATIVE_UPDATED_PC_MAXIMUM),
		).
		Ensure()
}

// INSTRUCTION_KIND_FIELD is the sole operation selector slot.
const INSTRUCTION_KIND_FIELD = bytes.SLICE_SIZE_MINIMUM

// INSTRUCTION_KIND_FIELD_COUNT fixes operation selector storage.
const INSTRUCTION_KIND_FIELD_COUNT = INSTRUCTION_KIND_FIELD + utf8.CHARACTER_SIZE_MINIMUM

// Instruction_Kind_Storage retains one operation selector.
type Instruction_Kind_Storage [INSTRUCTION_KIND_FIELD_COUNT]Instruction_Kind

// Instruction_Kind_Storage_Invariants fixes operation storage shape.
func Instruction_Kind_Storage_Invariants(
	value Instruction_Kind_Storage, _ aver.Namespace,
) {
	aver.Always(
		len(value) == INSTRUCTION_KIND_FIELD_COUNT,
		"NFA instruction has one operation field.",
	)
}

// INSTRUCTION_CHARACTER_FIELD is the sole literal payload slot.
const INSTRUCTION_CHARACTER_FIELD = bytes.SLICE_SIZE_MINIMUM

// INSTRUCTION_CHARACTER_FIELD_COUNT fixes literal payload storage.
const INSTRUCTION_CHARACTER_FIELD_COUNT = INSTRUCTION_CHARACTER_FIELD + utf8.CHARACTER_SIZE_MINIMUM

// Instruction_Character_Storage retains one literal character.
type Instruction_Character_Storage [INSTRUCTION_CHARACTER_FIELD_COUNT]rune

// Instruction_Character_Storage_Invariants fixes literal storage shape.
func Instruction_Character_Storage_Invariants(
	value Instruction_Character_Storage, _ aver.Namespace,
) {
	aver.Always(
		len(value) == INSTRUCTION_CHARACTER_FIELD_COUNT,
		"NFA instruction has one literal field.",
	)
}

// INSTRUCTION_TARGET_NEXT retains primary continuation.
const INSTRUCTION_TARGET_NEXT = bytes.SLICE_SIZE_MINIMUM

// INSTRUCTION_TARGET_BRANCH retains alternative continuation.
const INSTRUCTION_TARGET_BRANCH = INSTRUCTION_TARGET_NEXT + utf8.CHARACTER_SIZE_MINIMUM

// INSTRUCTION_TARGET_COUNT fixes both continuation slots.
const INSTRUCTION_TARGET_COUNT = INSTRUCTION_TARGET_BRANCH + utf8.CHARACTER_SIZE_MINIMUM

// Instruction_Targets stores continuation and split branch.
type Instruction_Targets [INSTRUCTION_TARGET_COUNT]Instruction_PC

// Instruction_Targets_Invariants fixes control-flow shape.
func Instruction_Targets_Invariants(
	value Instruction_Targets, _ aver.Namespace,
) {
	aver.Always(
		len(value) == INSTRUCTION_TARGET_COUNT,
		"NFA instruction has continuation and branch targets.",
	)
}

// INSTRUCTION_CLASS_RANGE_INDEX retains first class range.
const INSTRUCTION_CLASS_RANGE_INDEX = bytes.SLICE_SIZE_MINIMUM

// INSTRUCTION_CLASS_RANGE_COUNT retains class range length.
const INSTRUCTION_CLASS_RANGE_COUNT = INSTRUCTION_CLASS_RANGE_INDEX + utf8.CHARACTER_SIZE_MINIMUM

// INSTRUCTION_CLASS_NEGATED retains class polarity.
const INSTRUCTION_CLASS_NEGATED = INSTRUCTION_CLASS_RANGE_COUNT + utf8.CHARACTER_SIZE_MINIMUM

// INSTRUCTION_CLASS_FIELD_COUNT fixes class payload storage.
const INSTRUCTION_CLASS_FIELD_COUNT = INSTRUCTION_CLASS_NEGATED + utf8.CHARACTER_SIZE_MINIMUM

// Instruction_Class_Storage stores range opening, count, and negation.
type Instruction_Class_Storage [INSTRUCTION_CLASS_FIELD_COUNT]uint16

// Instruction_Class_Storage_Invariants fixes class payload shape.
func Instruction_Class_Storage_Invariants(
	value Instruction_Class_Storage, _ aver.Namespace,
) {
	aver.Always(
		len(value) == INSTRUCTION_CLASS_FIELD_COUNT,
		"NFA class instruction has complete range metadata.",
	)
}

// Instruction is one immutable NFA operation.
type Instruction struct {
	// Kind selects operation.
	Kind Instruction_Kind_Storage
	// Character stores literal payload.
	Character Instruction_Character_Storage
	// Targets store continuation edges.
	Targets Instruction_Targets
	// Class stores class payload.
	Class Instruction_Class_Storage
}

// Instruction_Invariants composes shape-safe NFA record.
func Instruction_Invariants(value Instruction, namespace aver.Namespace) {
	Instruction_Kind_Storage_Invariants(value.Kind, namespace)
	Instruction_Character_Storage_Invariants(value.Character, namespace)
	Instruction_Targets_Invariants(value.Targets, namespace)
	Instruction_Class_Storage_Invariants(value.Class, namespace)
}

// Instruction_Kinds stores every operation selector separately from payloads.
type Instruction_Kinds [INSTRUCTION_COUNT_MAXIMUM]Instruction_Kind

// Instruction_Kinds_Invariants fixes complete operation capacity.
func Instruction_Kinds_Invariants(
	value Instruction_Kinds, _ aver.Namespace,
) {
	aver.Always(
		len(value) == INSTRUCTION_COUNT_MAXIMUM,
		"NFA kind arena covers every pattern byte plus terminal match.",
	)
}

// Instruction_Characters stores every literal payload.
type Instruction_Characters [INSTRUCTION_COUNT_MAXIMUM]rune

// Instruction_Characters_Invariants fixes complete literal capacity.
func Instruction_Characters_Invariants(
	value Instruction_Characters, _ aver.Namespace,
) {
	aver.Always(
		len(value) == INSTRUCTION_COUNT_MAXIMUM,
		"NFA literal arena covers every pattern byte plus terminal match.",
	)
}

// Instruction_Next_Targets stores every primary continuation.
type Instruction_Next_Targets [INSTRUCTION_COUNT_MAXIMUM]Instruction_PC

// Instruction_Next_Targets_Invariants fixes continuation capacity.
func Instruction_Next_Targets_Invariants(
	value Instruction_Next_Targets, _ aver.Namespace,
) {
	aver.Always(
		len(value) == INSTRUCTION_COUNT_MAXIMUM,
		"NFA continuation arena covers every instruction.",
	)
}

// Instruction_Branch_Targets stores every alternative continuation.
type Instruction_Branch_Targets [INSTRUCTION_COUNT_MAXIMUM]Instruction_PC

// Instruction_Branch_Targets_Invariants fixes branch capacity.
func Instruction_Branch_Targets_Invariants(
	value Instruction_Branch_Targets, _ aver.Namespace,
) {
	aver.Always(
		len(value) == INSTRUCTION_COUNT_MAXIMUM,
		"NFA branch arena covers every instruction.",
	)
}

// Instruction_Class_Indexes stores every class range opening.
type Instruction_Class_Indexes [INSTRUCTION_COUNT_MAXIMUM]uint16

// Instruction_Class_Indexes_Invariants fixes class index capacity.
func Instruction_Class_Indexes_Invariants(
	value Instruction_Class_Indexes, _ aver.Namespace,
) {
	aver.Always(
		len(value) == INSTRUCTION_COUNT_MAXIMUM,
		"NFA class-index arena covers every instruction.",
	)
}

// Instruction_Class_Counts stores every class range count.
type Instruction_Class_Counts [INSTRUCTION_COUNT_MAXIMUM]uint16

// Instruction_Class_Counts_Invariants fixes class count capacity.
func Instruction_Class_Counts_Invariants(
	value Instruction_Class_Counts, _ aver.Namespace,
) {
	aver.Always(
		len(value) == INSTRUCTION_COUNT_MAXIMUM,
		"NFA class-count arena covers every instruction.",
	)
}

// Instruction_Class_Negations stores every class polarity.
type Instruction_Class_Negations [INSTRUCTION_COUNT_MAXIMUM]uint16

// Instruction_Class_Negations_Invariants fixes class polarity capacity.
func Instruction_Class_Negations_Invariants(
	value Instruction_Class_Negations, _ aver.Namespace,
) {
	aver.Always(
		len(value) == INSTRUCTION_COUNT_MAXIMUM,
		"NFA class-polarity arena covers every instruction.",
	)
}

// COMPILE_REFERENCE_NODE retains active syntax node.
const COMPILE_REFERENCE_NODE = bytes.SLICE_SIZE_MINIMUM

// COMPILE_REFERENCE_CHILD retains active reverse child.
const COMPILE_REFERENCE_CHILD = COMPILE_REFERENCE_NODE + utf8.CHARACTER_SIZE_MINIMUM

// COMPILE_REFERENCE_COUNT fixes compiler syntax references.
const COMPILE_REFERENCE_COUNT = COMPILE_REFERENCE_CHILD + utf8.CHARACTER_SIZE_MINIMUM

// Compile_References stores active syntax node and reverse child.
type Compile_References [COMPILE_REFERENCE_COUNT]Node_Reference

// Compile_References_Invariants fixes compiler traversal shape.
func Compile_References_Invariants(
	value Compile_References, _ aver.Namespace,
) {
	aver.Always(
		len(value) == COMPILE_REFERENCE_COUNT,
		"NFA compiler frame has node and child references.",
	)
}

// COMPILE_TARGET_CONTINUATION retains caller continuation.
const COMPILE_TARGET_CONTINUATION = bytes.SLICE_SIZE_MINIMUM

// COMPILE_TARGET_ACCUMULATED retains compiled subtree entry.
const COMPILE_TARGET_ACCUMULATED = COMPILE_TARGET_CONTINUATION + utf8.CHARACTER_SIZE_MINIMUM

// COMPILE_TARGET_COUNT fixes compiler target storage.
const COMPILE_TARGET_COUNT = COMPILE_TARGET_ACCUMULATED + utf8.CHARACTER_SIZE_MINIMUM

// Compile_Targets stores continuation and accumulated entry.
type Compile_Targets [COMPILE_TARGET_COUNT]Instruction_PC

// Compile_Targets_Invariants fixes compiler target shape.
func Compile_Targets_Invariants(value Compile_Targets, _ aver.Namespace) {
	aver.Always(
		len(value) == COMPILE_TARGET_COUNT,
		"NFA compiler frame has continuation and accumulated targets.",
	)
}

// COMPILE_FRAME_STAGE retains iterative traversal stage.
const COMPILE_FRAME_STAGE = bytes.SLICE_SIZE_MINIMUM

// COMPILE_FRAME_FIRST distinguishes first alternative branch.
const COMPILE_FRAME_FIRST = COMPILE_FRAME_STAGE + utf8.CHARACTER_SIZE_MINIMUM

// COMPILE_FRAME_CONTROL_COUNT fixes compiler control storage.
const COMPILE_FRAME_CONTROL_COUNT = COMPILE_FRAME_FIRST + utf8.CHARACTER_SIZE_MINIMUM

// Compile_Frame_Control stores traversal stage and first-branch flag.
type Compile_Frame_Control [COMPILE_FRAME_CONTROL_COUNT]uint8

// Compile_Frame_Control_Invariants fixes compiler control shape.
func Compile_Frame_Control_Invariants(
	value Compile_Frame_Control, _ aver.Namespace,
) {
	aver.Always(
		len(value) == COMPILE_FRAME_CONTROL_COUNT,
		"NFA compiler frame has stage and branch state.",
	)
}

// COMPILE_STAGE_ENTER dispatches syntax node kind.
const COMPILE_STAGE_ENTER uint8 = bits.WORD_8_MINIMUM

// COMPILE_STAGE_SEQUENCE selects next reverse sequence child.
const COMPILE_STAGE_SEQUENCE uint8 = COMPILE_STAGE_ENTER + utf8.CHARACTER_SIZE_MINIMUM

// COMPILE_STAGE_SEQUENCE_RETURN commits compiled child entry.
const COMPILE_STAGE_SEQUENCE_RETURN uint8 = COMPILE_STAGE_SEQUENCE + utf8.CHARACTER_SIZE_MINIMUM

// COMPILE_STAGE_ALTERNATIVE selects next brace branch.
const COMPILE_STAGE_ALTERNATIVE uint8 = COMPILE_STAGE_SEQUENCE_RETURN + utf8.CHARACTER_SIZE_MINIMUM

// COMPILE_STAGE_ALTERNATIVE_RETURN joins compiled branch entry.
const COMPILE_STAGE_ALTERNATIVE_RETURN uint8 = COMPILE_STAGE_ALTERNATIVE +
	utf8.CHARACTER_SIZE_MINIMUM

// CONTROL_FALSE keeps boolean control in fixed integer storage.
const CONTROL_FALSE uint8 = bits.WORD_8_MINIMUM

// CONTROL_TRUE keeps boolean control in fixed integer storage.
const CONTROL_TRUE uint8 = CONTROL_FALSE + utf8.CHARACTER_SIZE_MINIMUM

// Compile_Frame stores one suspended syntax node.
type Compile_Frame struct {
	// References retain reverse traversal.
	References Compile_References
	// Targets retain continuation and result.
	Targets Compile_Targets
	// Control retains traversal state.
	Control Compile_Frame_Control
}

// Compile_Frame_Invariants composes fixed iterative compiler frame.
func Compile_Frame_Invariants(value Compile_Frame, namespace aver.Namespace) {
	Compile_References_Invariants(value.References, namespace)
	Compile_Targets_Invariants(value.Targets, namespace)
	Compile_Frame_Control_Invariants(value.Control, namespace)
}

// Compile_Frames is fixed syntax traversal stack.
type Compile_Frames [NODE_COUNT_MAXIMUM]Compile_Frame

// Compile_Frames_Invariants fixes nonrecursive compiler depth.
func Compile_Frames_Invariants(value Compile_Frames, _ aver.Namespace) {
	aver.Always(
		len(value) == NODE_COUNT_MAXIMUM,
		"NFA compiler stack covers every nested syntax node.",
	)
}

// Compile_Depth prevents iterative syntax traversal from outgrowing its arena.
type Compile_Depth int

// Compile_Depth_Invariants follows the one-frame-per-node formula.
func Compile_Depth_Invariants(value Compile_Depth, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), bytes.SLICE_SIZE_MINIMUM, NODE_COUNT_MAXIMUM,
		).
		Ensure()
}

// COMPILE_DEPTH_NONZERO_MINIMUM is one active compiler frame.
const COMPILE_DEPTH_NONZERO_MINIMUM = bytes.SLICE_SIZE_MINIMUM + utf8.CHARACTER_SIZE_MINIMUM

// COMPILE_PUSH_DEPTH_MAXIMUM leaves one compiler frame slot.
const COMPILE_PUSH_DEPTH_MAXIMUM = NODE_COUNT_MAXIMUM - utf8.CHARACTER_SIZE_MINIMUM

// COMPILE_PUSHED_DEPTH_MINIMUM follows one pushed child frame.
const COMPILE_PUSHED_DEPTH_MINIMUM = COMPILE_DEPTH_NONZERO_MINIMUM +
	utf8.CHARACTER_SIZE_MINIMUM

// Nonzero_Compile_Depth retains at least the root compiler frame.
type Nonzero_Compile_Depth int

// Nonzero_Compile_Depth_Invariants excludes the completed empty stack.
func Nonzero_Compile_Depth_Invariants(
	value Nonzero_Compile_Depth, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), COMPILE_DEPTH_NONZERO_MINIMUM, NODE_COUNT_MAXIMUM,
		).
		Ensure()
}

// Push_Compile_Depth retains one free compiler frame slot.
type Push_Compile_Depth int

// Push_Compile_Depth_Invariants excludes empty and full compiler stacks.
func Push_Compile_Depth_Invariants(
	value Push_Compile_Depth, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), COMPILE_DEPTH_NONZERO_MINIMUM,
			COMPILE_PUSH_DEPTH_MAXIMUM,
		).
		Ensure()
}

// Pushed_Compile_Depth follows one newly suspended child frame.
type Pushed_Compile_Depth int

// Pushed_Compile_Depth_Invariants excludes root-only compiler state.
func Pushed_Compile_Depth_Invariants(
	value Pushed_Compile_Depth, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), COMPILE_PUSHED_DEPTH_MINIMUM, NODE_COUNT_MAXIMUM,
		).
		Ensure()
}

// Sequence_Compile_Level compresses one odd sequence-frame stack depth.
type Sequence_Compile_Level int

// Sequence_Compile_Level_Invariants follows root through maximum brace nesting.
func Sequence_Compile_Level_Invariants(
	value Sequence_Compile_Level, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), PATTERN_DEPTH_MINIMUM, PATTERN_DEPTH_MAXIMUM,
		).
		Ensure()
}

// Alternative_Compile_Level compresses one even alternative-frame stack depth.
type Alternative_Compile_Level int

// Alternative_Compile_Level_Invariants starts at the first open group.
func Alternative_Compile_Level_Invariants(
	value Alternative_Compile_Level, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), PARSER_DEPTH_NONEMPTY_MINIMUM, PATTERN_DEPTH_MAXIMUM,
		).
		Ensure()
}

// Sequence_Updated_Level compresses even depth after sequence transition.
type Sequence_Updated_Level int

// Sequence_Updated_Level_Invariants follows complete or pushed sequence state.
func Sequence_Updated_Level_Invariants(
	value Sequence_Updated_Level, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), PATTERN_DEPTH_MINIMUM, PATTERN_DEPTH_MAXIMUM,
		).
		Ensure()
}

// Alternative_Updated_Level compresses odd depth after alternative transition.
type Alternative_Updated_Level int

// Alternative_Updated_Level_Invariants follows returned or pushed alternative state.
func Alternative_Updated_Level_Invariants(
	value Alternative_Updated_Level, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), PATTERN_DEPTH_MINIMUM, PATTERN_DEPTH_MAXIMUM,
		).
		Ensure()
}

// Separator is one copied path boundary.
type Separator rune

// Separator_Invariants covers complete rune-storage domain.
func Separator_Invariants(value Separator, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(
			int32(value), utf8.CHARACTER_MINIMUM, utf8.CHARACTER_MAXIMUM,
		).
		Ensure()
}

// Separators is fixed caller-owned separator storage.
type Separators [SEPARATOR_COUNT_MAXIMUM]Separator

// Separators_Invariants fixes complete separator capacity.
func Separators_Invariants(value Separators, _ aver.Namespace) {
	aver.Always(
		len(value) == SEPARATOR_COUNT_MAXIMUM,
		"Separator storage covers maximum hostile count.",
	)
}

// Compile_Control stores arena cursors only.
type Compile_Control [COMPILE_CONTROL_COUNT]uint16

// Compile_Control_Invariants fixes compiler cursor shape.
func Compile_Control_Invariants(value Compile_Control, _ aver.Namespace) {
	aver.Always(
		len(value) == COMPILE_CONTROL_COUNT,
		"Compile workspace has complete arena cursors.",
	)
}

// Compile_Workspace owns parser and immutable NFA storage.
type Compile_Workspace struct {
	// Nodes own flat syntax until NFA construction ends.
	Nodes Nodes
	// Parser_Frames remove recursive brace parsing.
	Parser_Frames Parser_Frames
	// Ranges own character-class intervals.
	Ranges Class_Ranges
	// Instruction_Kinds own immutable operation selectors.
	Instruction_Kinds Instruction_Kinds
	// Instruction_Characters own immutable literal payloads.
	Instruction_Characters Instruction_Characters
	// Instruction_Next_Targets own primary continuation edges.
	Instruction_Next_Targets Instruction_Next_Targets
	// Instruction_Branch_Targets own alternative continuation edges.
	Instruction_Branch_Targets Instruction_Branch_Targets
	// Instruction_Class_Indexes own class range openings.
	Instruction_Class_Indexes Instruction_Class_Indexes
	// Instruction_Class_Counts own class range counts.
	Instruction_Class_Counts Instruction_Class_Counts
	// Instruction_Class_Negations own class polarity.
	Instruction_Class_Negations Instruction_Class_Negations
	// Compile_Frames remove recursive syntax compilation.
	Compile_Frames Compile_Frames
	// Separators own copied caller boundaries.
	Separators Separators
	// Control retains arena cursors.
	Control Compile_Control
}

// Compile_Workspace_Invariants composes fixed caller-owned storage.
func Compile_Workspace_Invariants(
	value *Compile_Workspace, namespace aver.Namespace,
) {
	aver.Always(value != nil, "Compile workspace is present.")
	Nodes_Invariants(value.Nodes, namespace)
	Parser_Frames_Invariants(value.Parser_Frames, namespace)
	Class_Ranges_Invariants(value.Ranges, namespace)
	Instruction_Kinds_Invariants(value.Instruction_Kinds, namespace)
	Instruction_Characters_Invariants(value.Instruction_Characters, namespace)
	Instruction_Next_Targets_Invariants(
		value.Instruction_Next_Targets, namespace,
	)
	Instruction_Branch_Targets_Invariants(
		value.Instruction_Branch_Targets, namespace,
	)
	Instruction_Class_Indexes_Invariants(
		value.Instruction_Class_Indexes, namespace,
	)
	Instruction_Class_Counts_Invariants(
		value.Instruction_Class_Counts, namespace,
	)
	Instruction_Class_Negations_Invariants(
		value.Instruction_Class_Negations, namespace,
	)
	Compile_Frames_Invariants(value.Compile_Frames, namespace)
	Separators_Invariants(value.Separators, namespace)
	Compile_Control_Invariants(value.Control, namespace)
}

// Compile_Workspace_Storage admits absent caller storage.
type Compile_Workspace_Storage [WORKSPACE_FIELD_COUNT]*Compile_Workspace

// Compile_Workspace_Storage_Invariants fixes pointer input shape.
func Compile_Workspace_Storage_Invariants(
	value Compile_Workspace_Storage, _ aver.Namespace,
) {
	aver.Always(
		len(value) == WORKSPACE_FIELD_COUNT,
		"Compile input has one workspace pointer field.",
	)
}

// Compile_Workspace_Input carries hostile optional compile storage.
type Compile_Workspace_Input struct {
	// State points at caller storage when present.
	State Compile_Workspace_Storage
}

// Compile_Workspace_Input_Invariants fixes optional pointer shape.
func Compile_Workspace_Input_Invariants(
	value Compile_Workspace_Input, namespace aver.Namespace,
) {
	Compile_Workspace_Storage_Invariants(value.State, namespace)
}

// Compile_Input bundles bounded parser dependencies.
type Compile_Input struct {
	// Source is hostile borrowed pattern bytes.
	Source Pattern_Source_Unvalidated
	// Separators are hostile path boundaries.
	Separators Separators_Unvalidated
	// Workspace owns parse and NFA state.
	Workspace Compile_Workspace_Input
}

// Compile_Input_Invariants composes hostile compile boundary.
func Compile_Input_Invariants(value Compile_Input, namespace aver.Namespace) {
	Pattern_Source_Unvalidated_Invariants(value.Source, namespace)
	Separators_Unvalidated_Invariants(value.Separators, namespace)
	Compile_Workspace_Input_Invariants(value.Workspace, namespace)
}

// Pattern_Workspace_Storage retains one immutable compiled arena.
type Pattern_Workspace_Storage [PATTERN_WORKSPACE_FIELD_COUNT]*Compile_Workspace

// Pattern_Workspace_Storage_Invariants fixes compiled pointer shape.
func Pattern_Workspace_Storage_Invariants(
	value Pattern_Workspace_Storage, _ aver.Namespace,
) {
	aver.Always(
		len(value) == PATTERN_WORKSPACE_FIELD_COUNT,
		"Compiled pattern has one workspace pointer field.",
	)
}

// Pattern_Control retains start and populated arena counts.
type Pattern_Control [PATTERN_CONTROL_COUNT]uint16

// Pattern_Control_Invariants fixes compiled scalar header shape.
func Pattern_Control_Invariants(value Pattern_Control, _ aver.Namespace) {
	aver.Always(
		len(value) == PATTERN_CONTROL_COUNT,
		"Compiled pattern has complete scalar header.",
	)
}

// Pattern borrows immutable caller compile storage.
type Pattern struct {
	// Workspace retains compiled instructions and separators.
	Workspace Pattern_Workspace_Storage
	// Control retains start and populated counts.
	Control Pattern_Control
}

// Pattern_Invariants verifies shape without dereferencing hostile pointer.
func Pattern_Invariants(value Pattern, namespace aver.Namespace) {
	Pattern_Workspace_Storage_Invariants(value.Workspace, namespace)
	Pattern_Control_Invariants(value.Control, namespace)
}

// Current_States stores closure before one candidate character.
type Current_States [INSTRUCTION_COUNT_MAXIMUM]Instruction_PC

// Current_States_Invariants fixes active-state capacity.
func Current_States_Invariants(value Current_States, _ aver.Namespace) {
	aver.Always(
		len(value) == INSTRUCTION_COUNT_MAXIMUM,
		"Current NFA state set covers every instruction once.",
	)
}

// Next_States stores closure after one candidate character.
type Next_States [INSTRUCTION_COUNT_MAXIMUM]Instruction_PC

// Next_States_Invariants fixes next-state capacity.
func Next_States_Invariants(value Next_States, _ aver.Namespace) {
	aver.Always(
		len(value) == INSTRUCTION_COUNT_MAXIMUM,
		"Next NFA state set covers every instruction once.",
	)
}

// Closure_States stores iterative epsilon traversal.
type Closure_States [INSTRUCTION_COUNT_MAXIMUM]Instruction_PC

// Closure_States_Invariants fixes epsilon traversal capacity.
func Closure_States_Invariants(value Closure_States, _ aver.Namespace) {
	aver.Always(
		len(value) == INSTRUCTION_COUNT_MAXIMUM,
		"NFA closure stack covers every instruction once.",
	)
}

// State_Generation marks one closure without clearing per transition.
type State_Generation uint16

// State_Generation_Invariants follows candidate characters plus initial closure.
func State_Generation_Invariants(
	value State_Generation, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(STATE_GENERATION_MINIMUM),
			uint16(STATE_GENERATION_MAXIMUM),
		).
		Ensure()
}

// Closure_Generation follows the initial closure through final text closure.
type Closure_Generation uint16

// Closure_Generation_Invariants excludes the cleared visited-set marker.
func Closure_Generation_Invariants(
	value Closure_Generation, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(CLOSURE_GENERATION_MINIMUM),
			uint16(STATE_GENERATION_MAXIMUM),
		).
		Ensure()
}

// Consume_Generation follows closures after at least one candidate character.
type Consume_Generation uint16

// Consume_Generation_Invariants starts after initial closure generation.
func Consume_Generation_Invariants(
	value Consume_Generation, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(CONSUME_GENERATION_MINIMUM),
			uint16(STATE_GENERATION_MAXIMUM),
		).
		Ensure()
}

// State_Generations tracks visited instructions.
type State_Generations [INSTRUCTION_COUNT_MAXIMUM]State_Generation

// State_Generations_Invariants fixes visited-set capacity.
func State_Generations_Invariants(
	value State_Generations, _ aver.Namespace,
) {
	aver.Always(
		len(value) == INSTRUCTION_COUNT_MAXIMUM,
		"NFA visited set covers every instruction.",
	)
}

// Active_Instruction_States is one bounded populated state prefix.
type Active_Instruction_States []Instruction_PC

// Active_Instruction_States_Invariants follows the compiled instruction bound.
func Active_Instruction_States_Invariants(
	value Active_Instruction_States, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), bytes.SLICE_SIZE_MINIMUM, ACTIVE_STATE_COUNT_MAXIMUM,
		).
		Ensure()
}

// Instruction_State_Storage is one complete caller-owned state arena.
type Instruction_State_Storage []Instruction_PC

// Instruction_State_Storage_Invariants fixes matcher arena capacity.
func Instruction_State_Storage_Invariants(
	value Instruction_State_Storage, _ aver.Namespace,
) {
	aver.Always(
		len(value) == INSTRUCTION_COUNT_MAXIMUM,
		"Matcher state view retains complete instruction-derived capacity.",
	)
}

// State_Count retains a possibly empty active-state prefix.
type State_Count int

// State_Count_Invariants follows maximum simultaneously consuming branches.
func State_Count_Invariants(value State_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), bytes.SLICE_SIZE_MINIMUM, ACTIVE_STATE_COUNT_MAXIMUM,
		).
		Ensure()
}

// Append_State_Count leaves room for one newly reached active state.
type Append_State_Count int

// Append_State_Count_Invariants excludes the already-full active prefix.
func Append_State_Count_Invariants(
	value Append_State_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), bytes.SLICE_SIZE_MINIMUM, APPEND_STATE_COUNT_MAXIMUM,
		).
		Ensure()
}

// Nonempty_State_Count follows one successful state addition.
type Nonempty_State_Count int

// Nonempty_State_Count_Invariants requires one reached instruction.
func Nonempty_State_Count_Invariants(
	value Nonempty_State_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), utf8.CHARACTER_SIZE_MINIMUM,
			ACTIVE_STATE_COUNT_MAXIMUM,
		).
		Ensure()
}

// Closure_Count retains a possibly empty alternative traversal stack.
type Closure_Count int

// Closure_Count_Invariants follows maximum suspended comma branches.
func Closure_Count_Invariants(value Closure_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), bytes.SLICE_SIZE_MINIMUM, CLOSURE_COUNT_MAXIMUM,
		).
		Ensure()
}

// Closure_Append_Count leaves room for one unseen closure instruction.
type Closure_Append_Count int

// Closure_Append_Count_Invariants excludes the already-full closure stack.
func Closure_Append_Count_Invariants(
	value Closure_Append_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), bytes.SLICE_SIZE_MINIMUM, CLOSURE_APPEND_COUNT_MAXIMUM,
		).
		Ensure()
}

// Instruction_Count exists only after forged pattern metadata passed validation.
type Instruction_Count int

// Instruction_Count_Invariants retains terminal match and every bounded instruction.
func Instruction_Count_Invariants(
	value Instruction_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), utf8.CHARACTER_SIZE_MINIMUM, INSTRUCTION_COUNT_MAXIMUM,
		).
		Ensure()
}

// Class_Range_Count exists only after forged range metadata passed validation.
type Class_Range_Count int

// Class_Range_Count_Invariants follows the caller-owned class-range arena.
func Class_Range_Count_Invariants(
	value Class_Range_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), bytes.SLICE_SIZE_MINIMUM, CLASS_RANGE_COUNT_MAXIMUM,
		).
		Ensure()
}

// Separator_Count exists only after forged separator metadata passed validation.
type Separator_Count int

// Separator_Count_Invariants follows the caller-owned separator arena.
func Separator_Count_Invariants(
	value Separator_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), SEPARATOR_COUNT_MINIMUM, SEPARATOR_COUNT_MAXIMUM,
		).
		Ensure()
}

// Contains carries one membership result through caller-owned scalar storage.
type Contains bool

// Contains_Invariants requires both membership outcomes across registered paths.
func Contains_Invariants(value Contains, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Membership succeeds.").
		Ensure()
}

// Match_Workspace owns active, next, pending, and visited NFA states.
type Match_Workspace struct {
	// Current stores closure before one candidate character.
	Current Current_States
	// Next stores closure after one candidate character.
	Next Next_States
	// Closure stores iterative epsilon traversal.
	Closure Closure_States
	// Seen deduplicates each closure.
	Seen State_Generations
}

// Match_Workspace_Invariants composes fixed caller-owned state sets.
func Match_Workspace_Invariants(
	value *Match_Workspace, namespace aver.Namespace,
) {
	aver.Always(value != nil, "Match workspace is present.")
	Current_States_Invariants(value.Current, namespace)
	Next_States_Invariants(value.Next, namespace)
	Closure_States_Invariants(value.Closure, namespace)
	State_Generations_Invariants(value.Seen, namespace)
}

// Match_Workspace_Storage admits absent caller match state.
type Match_Workspace_Storage [WORKSPACE_FIELD_COUNT]*Match_Workspace

// Match_Workspace_Storage_Invariants fixes pointer input shape.
func Match_Workspace_Storage_Invariants(
	value Match_Workspace_Storage, _ aver.Namespace,
) {
	aver.Always(
		len(value) == WORKSPACE_FIELD_COUNT,
		"Match input has one workspace pointer field.",
	)
}

// Match_Workspace_Input carries hostile optional match storage.
type Match_Workspace_Input struct {
	// State points at caller storage when present.
	State Match_Workspace_Storage
}

// Match_Workspace_Input_Invariants fixes optional pointer shape.
func Match_Workspace_Input_Invariants(
	value Match_Workspace_Input, namespace aver.Namespace,
) {
	Match_Workspace_Storage_Invariants(value.State, namespace)
}

// Match_Input bundles immutable pattern, hostile text, and caller state.
type Match_Input struct {
	// Pattern points at caller compiled NFA.
	Pattern Pattern
	// Text is hostile borrowed candidate.
	Text Text_Unvalidated
	// Workspace owns mutable NFA state sets.
	Workspace Match_Workspace_Input
}

// Match_Input_Invariants composes hostile match boundary.
func Match_Input_Invariants(value Match_Input, namespace aver.Namespace) {
	Pattern_Invariants(value.Pattern, namespace)
	Text_Unvalidated_Invariants(value.Text, namespace)
	Match_Workspace_Input_Invariants(value.Workspace, namespace)
}

// Compile validates and emits one NFA into caller workspace.
func Compile(input Compile_Input) (
	pattern Pattern,
	diagnostic Diagnostic,
	status Compile_Status,
) {
	defer func() {
		Pattern_Invariants(pattern, "Compile.pattern")
		Diagnostic_Invariants(diagnostic, "Compile.diagnostic")
		Compile_Status_Invariants(status, "Compile.status")
	}()
	Compile_Input_Invariants(input, "Compile.input")
	if len(input.Source) > PATTERN_SIZE_MAXIMUM {
		return Pattern{}, Diagnostic{Code: STATUS_INPUT_INVALID},
			STATUS_INPUT_INVALID
	}
	if len(input.Separators) > SEPARATOR_COUNT_MAXIMUM {
		return Pattern{}, Diagnostic{Code: STATUS_INPUT_INVALID},
			STATUS_INPUT_INVALID
	}
	workspace := input.Workspace.State[WORKSPACE_FIELD]
	if workspace == nil {
		return Pattern{}, Diagnostic{Code: STATUS_WORKSPACE_INVALID},
			STATUS_WORKSPACE_INVALID
	}
	Compile_Workspace_Invariants(workspace, "Compile.workspace")
	workspace.Control = Compile_Control{}
	for index := range input.Separators {
		if !bool(utf8.Valid_Character(utf8.Character(input.Separators[index]))) {
			return Pattern{}, Diagnostic{Code: STATUS_INPUT_INVALID},
				STATUS_INPUT_INVALID
		}
		workspace.Separators[index] = Separator(input.Separators[index])
	}
	workspace.Control[COMPILE_CONTROL_SEPARATOR_COUNT] =
		uint16(len(input.Separators))
	root, position, parse_status := pattern_parse(workspace, Pattern_Source(input.Source))
	if parse_status != STATUS_OK {
		return Pattern{}, Diagnostic{
			Code: Compile_Status(parse_status), Position: Diagnostic_Position(position),
		}, Compile_Status(parse_status)
	}
	start := pattern_compile(workspace, root)
	pattern = Pattern{Workspace: Pattern_Workspace_Storage{workspace}}
	pattern.Control[PATTERN_CONTROL_START] = uint16(start)
	pattern.Control[PATTERN_CONTROL_INSTRUCTION_COUNT] =
		workspace.Control[COMPILE_CONTROL_INSTRUCTION_COUNT]
	pattern.Control[PATTERN_CONTROL_SEPARATOR_COUNT] =
		workspace.Control[COMPILE_CONTROL_SEPARATOR_COUNT]
	return pattern, Diagnostic{}, STATUS_OK
}

// Match executes compiled NFA against one bounded candidate.
func Match(input Match_Input) (matched Matched, status Match_Status) {
	defer func() {
		Matched_Invariants(matched, "Match.matched")
		Match_Status_Invariants(status, "Match.status")
	}()
	Match_Input_Invariants(input, "Match.input")
	if len(input.Text) > TEXT_SIZE_MAXIMUM {
		return false, STATUS_INPUT_INVALID
	}
	state := pattern_validate(input.Pattern)
	if state != STATUS_OK {
		return false, Match_Status(state)
	}
	workspace := input.Workspace.State[WORKSPACE_FIELD]
	if workspace == nil {
		return false, STATUS_WORKSPACE_INVALID
	}
	Match_Workspace_Invariants(workspace, "Match.workspace")
	for index := range workspace.Seen {
		workspace.Seen[index] = State_Generation(STATE_GENERATION_MINIMUM)
	}
	compiled := input.Pattern.Workspace[PATTERN_WORKSPACE_FIELD]
	current := Instruction_State_Storage(workspace.Current[:])
	next := Instruction_State_Storage(workspace.Next[:])
	current_count := State_Count(bytes.SLICE_SIZE_MINIMUM)
	generation := Closure_Generation(CLOSURE_GENERATION_MINIMUM)
	start := Instruction_PC(input.Pattern.Control[PATTERN_CONTROL_START])
	current_count = State_Count(state_add(
		input.Pattern, workspace, current, Append_State_Count(current_count),
		start, generation,
	))
	position := TEXT_SIZE_MINIMUM
	for position < len(input.Text) {
		character, size := utf8.Decode_Character(utf8.Bytes(input.Text[position:]))
		position += int(size)
		generation++
		next_count := state_consume(
			input.Pattern, workspace,
			Active_Instruction_States(current[:current_count]), next,
			utf8.Decoded_Character(character), Consume_Generation(generation),
		)
		current, next = next, current
		current_count = next_count
	}
	for index := bytes.SLICE_SIZE_MINIMUM; index < int(current_count); index++ {
		kind := compiled.Instruction_Kinds[current[index]]
		if kind == INSTRUCTION_MATCH {
			return true, STATUS_OK
		}
	}
	return false, STATUS_OK
}

// Quote_Meta_Into escapes metacharacters into caller destination atomically.
func Quote_Meta_Into(destination Output, source Pattern_Source_Unvalidated) (
	count Output_Count,
	status Quote_Status,
) {
	defer func() {
		Output_Count_Invariants(count, "Quote_Meta_Into.count")
		Quote_Status_Invariants(status, "Quote_Meta_Into.status")
	}()
	Output_Invariants(destination, "Quote_Meta_Into.destination")
	Pattern_Source_Unvalidated_Invariants(source, "Quote_Meta_Into.source")
	if len(source) > PATTERN_SIZE_MAXIMUM {
		return Output_Count(bytes.SLICE_SIZE_MINIMUM), STATUS_INPUT_INVALID
	}
	required_count := len(source)
	for index := range source {
		if bool(Special(Character(source[index]))) {
			required_count++
		}
	}
	if len(destination) < required_count {
		return Output_Count(bytes.SLICE_SIZE_MINIMUM), STATUS_OUTPUT_TOO_SMALL
	}
	position := bytes.SLICE_SIZE_MINIMUM
	for index := range source {
		if bool(Special(Character(source[index]))) {
			destination[position] = '\\'
			position++
		}
		destination[position] = source[index]
		position++
	}
	return Output_Count(position), STATUS_OK
}

// Special reports whether byte needs glob quoting.
func Special(character Character) (special Special_Character) {
	defer func() {
		Special_Character_Invariants(special, "Special.special")
	}()
	Character_Invariants(character, "Special.character")
	switch character {
	case '*', '?', '\\', '[', ']', '{', '}':
		return true
	}
	return false
}

func pattern_parse(
	workspace_state *Compile_Workspace,
	source Pattern_Source,
) (root Root_Node_Reference, position Pattern_Position, status Syntax_Status) {
	defer func() {
		Root_Node_Reference_Invariants(root, "pattern_parse.root")
		Pattern_Position_Invariants(position, "pattern_parse.position")
		Syntax_Status_Invariants(status, "pattern_parse.status")
	}()
	Compile_Workspace_Invariants(workspace_state, "pattern_parse.workspace_state")
	Pattern_Source_Invariants(source, "pattern_parse.source")
	workspace := (*Compile_Workspace)(workspace_state)
	root_reference, status := node_create(workspace, NODE_SEQUENCE)
	root = Root_Node_Reference(root_reference)
	current, depth := Node_Reference(root), Parser_Depth(PATTERN_DEPTH_MINIMUM)
	nonempty := Nonempty_Pattern_Source(source)
	for status == STATUS_OK {
		if position == Pattern_Position(len(source)) {
			break
		}
		character := source[position]
		index := Pattern_Index(position)
		parent := Container_Node_Reference(current)
		open_depth := Open_Parser_Depth(depth)
		nested := Nested_Sequence_Reference(current)
		switch character {
		case '{':
			opened, next_depth, state := group_open(
				workspace, parent, depth,
			)
			current = Node_Reference(opened)
			depth, status = Parser_Depth(next_depth), state
			position++
		case ',':
			if depth == PATTERN_DEPTH_MINIMUM {
				end, state := literal_parse(workspace, nonempty, parent, index)
				position, status = Pattern_Position(end), state
			} else {
				branch, state := group_branch(workspace, nested, open_depth)
				current, status = Node_Reference(branch), state
				position++
			}
		case '}':
			if depth == PATTERN_DEPTH_MINIMUM {
				status = STATUS_SYNTAX_INVALID
			} else {
				parent, next_depth := group_close(workspace, open_depth)
				current, depth = Node_Reference(parent), Parser_Depth(next_depth)
				position++
			}
		case '[':
			end, state := class_parse(workspace, nonempty, parent, index)
			position, status = Pattern_Position(end), state
		case '*':
			end, state := star_parse(workspace, nonempty, parent, index)
			position, status = Pattern_Position(end), state
		case '?':
			position++
			status = atom_append(workspace, parent, ATOM_SINGLE, ATOM_CHARACTER_EMPTY)
		default:
			end, state := literal_parse(workspace, nonempty, parent, index)
			position, status = Pattern_Position(end), state
		}
	}
	if status == STATUS_OK {
		if depth != PATTERN_DEPTH_MINIMUM {
			status = STATUS_SYNTAX_INVALID
		}
	}
	return root, position, status
}

func star_parse(
	workspace *Compile_Workspace,
	source Nonempty_Pattern_Source,
	parent Container_Node_Reference,
	position Pattern_Index,
) (end Nonzero_Pattern_Position, status Syntax_Status) {
	defer func() {
		Nonzero_Pattern_Position_Invariants(end, "star_parse.end")
		Syntax_Status_Invariants(status, "star_parse.status")
	}()
	Compile_Workspace_Invariants(workspace, "star_parse.workspace")
	Nonempty_Pattern_Source_Invariants(source, "star_parse.source")
	Container_Node_Reference_Invariants(parent, "star_parse.parent")
	Pattern_Index_Invariants(position, "star_parse.position")
	cursor := Pattern_Position(position) + Pattern_Position(utf8.CHARACTER_SIZE_MINIMUM)
	kind := ATOM_STAR
	if cursor < Pattern_Position(len(source)) {
		if source[cursor] == '*' {
			kind = ATOM_SUPER_STAR
			cursor++
		}
	}
	status = atom_append(
		workspace, parent, kind, ATOM_CHARACTER_EMPTY,
	)
	return Nonzero_Pattern_Position(cursor), status
}

func group_open(
	workspace_state *Compile_Workspace,
	current Container_Node_Reference,
	depth Parser_Depth,
) (
	updated_current Opened_Sequence_Reference,
	updated_depth Opened_Parser_Depth,
	status Syntax_Status,
) {
	defer func() {
		Opened_Sequence_Reference_Invariants(
			updated_current, "group_open.updated_current",
		)
		Opened_Parser_Depth_Invariants(updated_depth, "group_open.updated_depth")
		Syntax_Status_Invariants(status, "group_open.status")
	}()
	Compile_Workspace_Invariants(workspace_state, "group_open.workspace_state")
	Container_Node_Reference_Invariants(current, "group_open.current")
	Parser_Depth_Invariants(depth, "group_open.depth")
	workspace := (*Compile_Workspace)(workspace_state)
	if depth == Parser_Depth(len(workspace.Parser_Frames)) {
		return Opened_Sequence_Reference(current), Opened_Parser_Depth(depth),
			STATUS_SYNTAX_INVALID
	}
	alternative, status := node_create(workspace, NODE_ALTERNATIVE)
	if status != STATUS_OK {
		return Opened_Sequence_Reference(current), Opened_Parser_Depth(depth), status
	}
	node_append_child(
		workspace, Child_Parent_Reference(current), Child_Node_Reference(alternative),
	)
	sequence, status := node_create(workspace, NODE_SEQUENCE)
	if status != STATUS_OK {
		return Opened_Sequence_Reference(current), Opened_Parser_Depth(depth), status
	}
	node_append_child(
		workspace, Child_Parent_Reference(alternative), Child_Node_Reference(sequence),
	)
	workspace.Parser_Frames[depth] = Parser_Frame{References: Parser_References{
		Parent:      Suspended_Sequence_Reference(current),
		Alternative: Parser_Alternative_Reference(alternative),
		Sequence:    Opened_Sequence_Reference(sequence),
	}}
	return Opened_Sequence_Reference(sequence),
		Opened_Parser_Depth(depth + utf8.CHARACTER_SIZE_MINIMUM), STATUS_OK
}

func group_branch(
	workspace_state *Compile_Workspace,
	current Nested_Sequence_Reference,
	depth Open_Parser_Depth,
) (updated_current Branch_Sequence_Reference, status Syntax_Status) {
	defer func() {
		Branch_Sequence_Reference_Invariants(
			updated_current, "group_branch.updated_current",
		)
		Syntax_Status_Invariants(status, "group_branch.status")
	}()
	Compile_Workspace_Invariants(workspace_state, "group_branch.workspace_state")
	Nested_Sequence_Reference_Invariants(current, "group_branch.current")
	Open_Parser_Depth_Invariants(depth, "group_branch.depth")
	workspace := (*Compile_Workspace)(workspace_state)
	frame := &workspace.Parser_Frames[int(depth)-utf8.CHARACTER_SIZE_MINIMUM]
	sequence, status := node_create(workspace, NODE_SEQUENCE)
	if status != STATUS_OK {
		return Branch_Sequence_Reference(current), status
	}
	node_append_child(
		workspace,
		Child_Parent_Reference(frame.References.Alternative),
		Child_Node_Reference(sequence),
	)
	frame.References.Sequence = Opened_Sequence_Reference(sequence)
	return Branch_Sequence_Reference(sequence), STATUS_OK
}

func group_close(
	workspace_state *Compile_Workspace,
	depth Open_Parser_Depth,
) (current Suspended_Sequence_Reference, updated_depth Closed_Parser_Depth) {
	defer func() {
		Suspended_Sequence_Reference_Invariants(current, "group_close.current")
		Closed_Parser_Depth_Invariants(updated_depth, "group_close.updated_depth")
	}()
	Compile_Workspace_Invariants(workspace_state, "group_close.workspace_state")
	Open_Parser_Depth_Invariants(depth, "group_close.depth")
	workspace := (*Compile_Workspace)(workspace_state)
	depth--
	return Suspended_Sequence_Reference(
		workspace.Parser_Frames[depth].References.Parent,
	), Closed_Parser_Depth(depth)
}

func literal_parse(
	workspace_state *Compile_Workspace,
	source Nonempty_Pattern_Source,
	parent Container_Node_Reference,
	position Pattern_Index,
) (updated_position Nonzero_Pattern_Position, status Syntax_Status) {
	defer func() {
		Nonzero_Pattern_Position_Invariants(
			updated_position, "literal_parse.updated_position",
		)
		Syntax_Status_Invariants(status, "literal_parse.status")
	}()
	Compile_Workspace_Invariants(workspace_state, "literal_parse.workspace_state")
	Nonempty_Pattern_Source_Invariants(source, "literal_parse.source")
	Container_Node_Reference_Invariants(parent, "literal_parse.parent")
	Pattern_Index_Invariants(position, "literal_parse.position")
	workspace := (*Compile_Workspace)(workspace_state)
	if source[position] == '\\' {
		position++
		if position == Pattern_Index(len(source)) {
			return Nonzero_Pattern_Position(position), STATUS_SYNTAX_INVALID
		}
	}
	character, size := utf8.Decode_Character(utf8.Bytes(source[position:]))
	position += Pattern_Index(size)
	return Nonzero_Pattern_Position(position),
		atom_append(workspace, parent, ATOM_LITERAL, character)
}

func class_parse(
	workspace *Compile_Workspace,
	source Nonempty_Pattern_Source,
	parent Container_Node_Reference,
	position Pattern_Index,
) (end Nonzero_Pattern_Position, status Syntax_Status) {
	defer func() {
		Nonzero_Pattern_Position_Invariants(end, "class_parse.end")
		Syntax_Status_Invariants(status, "class_parse.status")
	}()
	Compile_Workspace_Invariants(workspace, "class_parse.workspace")
	Nonempty_Pattern_Source_Invariants(source, "class_parse.source")
	Container_Node_Reference_Invariants(parent, "class_parse.parent")
	Pattern_Index_Invariants(position, "class_parse.position")
	size := Pattern_Position(len(source))
	cursor := Pattern_Position(position) + Pattern_Position(utf8.CHARACTER_SIZE_MINIMUM)
	negated := false
	if cursor < size {
		negated = source[cursor] == '!'
	}
	if negated {
		cursor++
	}
	range_index := workspace.Control[COMPILE_CONTROL_RANGE_COUNT]
	range_count := uint16(bytes.SLICE_SIZE_MINIMUM)
	for status == STATUS_OK {
		if cursor == size {
			return Nonzero_Pattern_Position(cursor), STATUS_SYNTAX_INVALID
		}
		if source[cursor] == ']' {
			if range_count == bytes.SLICE_SIZE_MINIMUM {
				return Nonzero_Pattern_Position(cursor), STATUS_SYNTAX_INVALID
			}
			cursor++
			break
		}
		var low utf8.Decoded_Character
		class_end := Class_Character_End(bytes.SLICE_SIZE_MINIMUM)
		low, class_end, status = class_character(
			Class_Pattern_Source(source), Class_Character_Index(cursor),
		)
		cursor = Pattern_Position(class_end)
		if status != STATUS_OK {
			return Nonzero_Pattern_Position(cursor), status
		}
		high := low
		if cursor+Pattern_Position(utf8.CHARACTER_SIZE_MINIMUM) < size {
			if source[cursor] == '-' {
				if source[cursor+utf8.CHARACTER_SIZE_MINIMUM] != ']' {
					cursor++
					high, class_end, status = class_character(
						Class_Pattern_Source(source),
						Class_Character_Index(cursor),
					)
					cursor = Pattern_Position(class_end)
				}
			}
		}
		if status != STATUS_OK {
			return Nonzero_Pattern_Position(cursor), status
		}
		if high < low {
			return Nonzero_Pattern_Position(cursor), STATUS_SYNTAX_INVALID
		}
		range_append(workspace, low, high)
		range_count++
	}
	class := Node_Class{range_index, range_count}
	if negated {
		class[NODE_CLASS_NEGATED] = uint16(CONTROL_TRUE)
	}
	return Nonzero_Pattern_Position(cursor),
		class_append(workspace, parent, class)
}

func class_append(
	workspace *Compile_Workspace,
	parent Container_Node_Reference,
	class Node_Class,
) (status Syntax_Status) {
	defer func() { Syntax_Status_Invariants(status, "class_append.status") }()
	Compile_Workspace_Invariants(workspace, "class_append.workspace")
	Container_Node_Reference_Invariants(parent, "class_append.parent")
	Node_Class_Invariants(class, "class_append.class")
	node, create_status := node_create(workspace, NODE_CLASS)
	if create_status != STATUS_OK {
		return create_status
	}
	workspace.Nodes[int(node)-utf8.CHARACTER_SIZE_MINIMUM].Class = class
	node_append_child(
		workspace, Child_Parent_Reference(parent), Child_Node_Reference(node),
	)
	return STATUS_OK
}

func class_character(
	source Class_Pattern_Source,
	position_value Class_Character_Index,
) (
	character utf8.Decoded_Character,
	position Class_Character_End,
	status Syntax_Status,
) {
	defer func() {
		utf8.Decoded_Character_Invariants(character, "class_character.character")
		Class_Character_End_Invariants(position, "class_character.position")
		Syntax_Status_Invariants(status, "class_character.status")
	}()
	Class_Pattern_Source_Invariants(source, "class_character.source")
	Class_Character_Index_Invariants(
		position_value, "class_character.position_value",
	)
	position_index := Pattern_Position(position_value)
	if source[position_index] == '\\' {
		position_index++
		if position_index == Pattern_Position(len(source)) {
			return utf8.Decoded_Character(utf8.DECODED_CHARACTER_MINIMUM),
				Class_Character_End(position_index), STATUS_SYNTAX_INVALID
		}
	}
	character, size := utf8.Decode_Character(utf8.Bytes(source[position_index:]))
	return character, Class_Character_End(position_index + Pattern_Position(size)),
		STATUS_OK
}

func atom_append(
	workspace_state *Compile_Workspace,
	parent Container_Node_Reference,
	kind Atom_Kind,
	character utf8.Decoded_Character,
) (status Syntax_Status) {
	defer func() { Syntax_Status_Invariants(status, "atom_append.status") }()
	Compile_Workspace_Invariants(workspace_state, "atom_append.workspace_state")
	Container_Node_Reference_Invariants(parent, "atom_append.parent")
	Atom_Kind_Invariants(kind, "atom_append.kind")
	utf8.Decoded_Character_Invariants(character, "atom_append.character")
	workspace := (*Compile_Workspace)(workspace_state)
	node, create_status := node_create(workspace, Node_Kind(kind))
	if create_status != STATUS_OK {
		return create_status
	}
	workspace.Nodes[int(node)-utf8.CHARACTER_SIZE_MINIMUM].Character[NODE_CHARACTER_FIELD] =
		rune(character)
	node_append_child(
		workspace, Child_Parent_Reference(parent), Child_Node_Reference(node),
	)
	return STATUS_OK
}

func node_create(
	workspace_state *Compile_Workspace,
	kind Node_Kind,
) (reference Node_Reference, status Syntax_Status) {
	defer func() {
		Node_Reference_Invariants(reference, "node_create.reference")
		Syntax_Status_Invariants(status, "node_create.status")
	}()
	Compile_Workspace_Invariants(workspace_state, "node_create.workspace_state")
	Node_Kind_Invariants(kind, "node_create.kind")
	workspace := (*Compile_Workspace)(workspace_state)
	count := workspace.Control[COMPILE_CONTROL_NODE_COUNT]
	if int(count) == len(workspace.Nodes) {
		return NODE_NONE, STATUS_SYNTAX_INVALID
	}
	workspace.Nodes[count] = Node{Kind: kind}
	count++
	workspace.Control[COMPILE_CONTROL_NODE_COUNT] = count
	return Node_Reference(count), STATUS_OK
}

func node_append_child(
	workspace_state *Compile_Workspace,
	parent Child_Parent_Reference,
	child Child_Node_Reference,
) {
	Compile_Workspace_Invariants(workspace_state, "node_append_child.workspace_state")
	Child_Parent_Reference_Invariants(parent, "node_append_child.parent")
	Child_Node_Reference_Invariants(child, "node_append_child.child")
	workspace := (*Compile_Workspace)(workspace_state)
	parent_node := &workspace.Nodes[int(parent)-utf8.CHARACTER_SIZE_MINIMUM]
	last := parent_node.Links.Last_Child
	if last == Last_Child_Reference(NODE_NONE) {
		parent_node.Links.First_Child = First_Child_Reference(child)
	} else {
		workspace.Nodes[int(last)-utf8.CHARACTER_SIZE_MINIMUM].
			Links.Next_Sibling = Next_Sibling_Reference(child)
		workspace.Nodes[int(child)-utf8.CHARACTER_SIZE_MINIMUM].
			Links.Previous_Sibling = Previous_Sibling_Reference(last)
	}
	parent_node.Links.Last_Child = Last_Child_Reference(child)
}

func range_append(
	workspace_state *Compile_Workspace,
	low utf8.Decoded_Character,
	high utf8.Decoded_Character,
) {
	Compile_Workspace_Invariants(workspace_state, "range_append.workspace_state")
	utf8.Decoded_Character_Invariants(low, "range_append.low")
	utf8.Decoded_Character_Invariants(high, "range_append.high")
	workspace := (*Compile_Workspace)(workspace_state)
	count := workspace.Control[COMPILE_CONTROL_RANGE_COUNT]
	aver.Always(
		int(count) < len(workspace.Ranges),
		"One class shell leaves one formula-derived range slot per source byte.",
	)
	workspace.Ranges[count] = Class_Range{
		Low:  Class_Low(low),
		High: Class_High(high),
	}
	workspace.Control[COMPILE_CONTROL_RANGE_COUNT] = count + utf8.CHARACTER_SIZE_MINIMUM
}

func pattern_compile(
	workspace_state *Compile_Workspace,
	root Root_Node_Reference,
) (start Instruction_PC) {
	defer func() {
		Instruction_PC_Invariants(start, "pattern_compile.start")
	}()
	Compile_Workspace_Invariants(workspace_state, "pattern_compile.workspace_state")
	Root_Node_Reference_Invariants(root, "pattern_compile.root")
	workspace := (*Compile_Workspace)(workspace_state)
	workspace.Control[COMPILE_CONTROL_INSTRUCTION_COUNT] = bytes.SLICE_SIZE_MINIMUM
	match := instruction_emit(workspace, Instruction{
		Kind: Instruction_Kind_Storage{INSTRUCTION_MATCH},
	})
	depth := Compile_Depth(utf8.CHARACTER_SIZE_MINIMUM)
	result := match
	workspace.Compile_Frames[bytes.SLICE_SIZE_MINIMUM] = Compile_Frame{
		References: Compile_References{COMPILE_REFERENCE_NODE: Node_Reference(root)},
		Targets: Compile_Targets{
			COMPILE_TARGET_CONTINUATION: match,
		},
		Control: Compile_Frame_Control{
			COMPILE_FRAME_STAGE: COMPILE_STAGE_ENTER,
		},
	}
	for depth != PATTERN_DEPTH_MINIMUM {
		frame := &workspace.Compile_Frames[depth-utf8.CHARACTER_SIZE_MINIMUM]
		switch frame.Control[COMPILE_FRAME_STAGE] {
		case COMPILE_STAGE_ENTER:
			updated, next := compile_enter(
				workspace, frame, Nonzero_Compile_Depth(depth),
				Instruction_Continuation(result),
			)
			depth, result = Compile_Depth(updated), next
		case COMPILE_STAGE_SEQUENCE:
			level := Sequence_Compile_Level(
				(depth - Compile_Depth(COMPILE_DEPTH_NONZERO_MINIMUM)) /
					Compile_Depth(len("{}")),
			)
			updated, next := compile_sequence(workspace, frame, level, result)
			depth = Compile_Depth(updated * Sequence_Updated_Level(len("{}")))
			result = next
		case COMPILE_STAGE_SEQUENCE_RETURN:
			compile_sequence_return(workspace, frame, result)
		case COMPILE_STAGE_ALTERNATIVE:
			level := Alternative_Compile_Level(depth / Compile_Depth(len("{}")))
			updated, next := compile_alternative(
				workspace, frame, level, Alternative_Result_PC(result),
			)
			depth = Compile_Depth(
				updated*Alternative_Updated_Level(len("{}")) +
					Alternative_Updated_Level(COMPILE_DEPTH_NONZERO_MINIMUM),
			)
			result = Instruction_PC(next)
		case COMPILE_STAGE_ALTERNATIVE_RETURN:
			compile_alternative_return(
				workspace, frame, Alternative_Result_PC(result),
			)
		}
	}
	return result
}

func compile_enter(
	workspace_state *Compile_Workspace,
	frame_state *Compile_Frame,
	depth Nonzero_Compile_Depth,
	result Instruction_Continuation,
) (
	updated_depth Nonzero_Compile_Depth,
	updated_result Instruction_PC,
) {
	defer func() {
		Nonzero_Compile_Depth_Invariants(updated_depth, "compile_enter.updated_depth")
		Instruction_PC_Invariants(updated_result, "compile_enter.updated_result")
	}()
	Compile_Workspace_Invariants(workspace_state, "compile_enter.workspace_state")
	Compile_Frame_Invariants(*frame_state, "compile_enter.frame_state")
	Nonzero_Compile_Depth_Invariants(depth, "compile_enter.depth")
	Instruction_Continuation_Invariants(result, "compile_enter.result")
	workspace := (*Compile_Workspace)(workspace_state)
	frame := (*Compile_Frame)(frame_state)
	updated_result = Instruction_PC(result)
	reference := frame.References[COMPILE_REFERENCE_NODE]
	aver.Always(reference != NODE_NONE, "Compiler frame references one syntax node.")
	aver.Always(
		int(reference) <= int(workspace.Control[COMPILE_CONTROL_NODE_COUNT]),
		"Compiler frame reference stays inside parsed syntax.",
	)
	node := workspace.Nodes[int(reference)-utf8.CHARACTER_SIZE_MINIMUM]
	switch node.Kind {
	case NODE_SEQUENCE:
		frame.References[COMPILE_REFERENCE_CHILD] =
			Node_Reference(node.Links.Last_Child)
		frame.Targets[COMPILE_TARGET_ACCUMULATED] =
			frame.Targets[COMPILE_TARGET_CONTINUATION]
		frame.Control[COMPILE_FRAME_STAGE] = COMPILE_STAGE_SEQUENCE
	case NODE_ALTERNATIVE:
		frame.References[COMPILE_REFERENCE_CHILD] =
			Node_Reference(node.Links.Last_Child)
		frame.Control[COMPILE_FRAME_FIRST] = CONTROL_TRUE
		frame.Control[COMPILE_FRAME_STAGE] = COMPILE_STAGE_ALTERNATIVE
	default:
		updated_result = Instruction_PC(instruction_from_node(
			workspace,
			Leaf_Node{
				Kind: Leaf_Kind(node.Kind), Character: node.Character,
				Class: node.Class,
			},
			Instruction_Continuation(frame.Targets[COMPILE_TARGET_CONTINUATION]),
		))
		depth--
	}
	return depth, updated_result
}

func compile_sequence(
	workspace_state *Compile_Workspace,
	frame_state *Compile_Frame,
	depth Sequence_Compile_Level,
	result Instruction_PC,
) (
	updated_depth Sequence_Updated_Level,
	updated_result Instruction_PC,
) {
	defer func() {
		Sequence_Updated_Level_Invariants(
			updated_depth, "compile_sequence.updated_depth",
		)
		Instruction_PC_Invariants(updated_result, "compile_sequence.updated_result")
	}()
	Compile_Workspace_Invariants(workspace_state, "compile_sequence.workspace_state")
	Compile_Frame_Invariants(*frame_state, "compile_sequence.frame_state")
	Sequence_Compile_Level_Invariants(depth, "compile_sequence.depth")
	Instruction_PC_Invariants(result, "compile_sequence.result")
	workspace := (*Compile_Workspace)(workspace_state)
	frame := (*Compile_Frame)(frame_state)
	child := frame.References[COMPILE_REFERENCE_CHILD]
	if child == NODE_NONE {
		return Sequence_Updated_Level(depth),
			frame.Targets[COMPILE_TARGET_ACCUMULATED]
	}
	frame.Control[COMPILE_FRAME_STAGE] = COMPILE_STAGE_SEQUENCE_RETURN
	raw_depth := Push_Compile_Depth(
		depth*Sequence_Compile_Level(len("{}")) +
			Sequence_Compile_Level(COMPILE_DEPTH_NONZERO_MINIMUM),
	)
	pushed := compile_push(
		workspace,
		Child_Node_Reference(child),
		Instruction_Continuation(frame.Targets[COMPILE_TARGET_ACCUMULATED]),
		raw_depth,
	)
	return Sequence_Updated_Level(pushed / Pushed_Compile_Depth(len("{}"))), result
}

func compile_sequence_return(
	workspace_state *Compile_Workspace,
	frame_state *Compile_Frame,
	result Instruction_PC,
) {
	Compile_Workspace_Invariants(
		workspace_state, "compile_sequence_return.workspace_state",
	)
	Compile_Frame_Invariants(*frame_state, "compile_sequence_return.frame_state")
	Instruction_PC_Invariants(result, "compile_sequence_return.result")
	workspace := (*Compile_Workspace)(workspace_state)
	frame := (*Compile_Frame)(frame_state)
	child := frame.References[COMPILE_REFERENCE_CHILD]
	frame.Targets[COMPILE_TARGET_ACCUMULATED] = result
	frame.References[COMPILE_REFERENCE_CHILD] =
		Node_Reference(workspace.Nodes[int(child)-utf8.CHARACTER_SIZE_MINIMUM].
			Links.Previous_Sibling)
	frame.Control[COMPILE_FRAME_STAGE] = COMPILE_STAGE_SEQUENCE
}

func compile_alternative(
	workspace_state *Compile_Workspace,
	frame_state *Compile_Frame,
	depth Alternative_Compile_Level,
	result Alternative_Result_PC,
) (
	updated_depth Alternative_Updated_Level,
	updated_result Alternative_Updated_PC,
) {
	defer func() {
		Alternative_Updated_Level_Invariants(
			updated_depth, "compile_alternative.updated_depth",
		)
		Alternative_Updated_PC_Invariants(
			updated_result, "compile_alternative.updated_result",
		)
	}()
	Compile_Workspace_Invariants(workspace_state, "compile_alternative.workspace_state")
	Compile_Frame_Invariants(*frame_state, "compile_alternative.frame_state")
	Alternative_Compile_Level_Invariants(depth, "compile_alternative.depth")
	Alternative_Result_PC_Invariants(result, "compile_alternative.result")
	workspace := (*Compile_Workspace)(workspace_state)
	frame := (*Compile_Frame)(frame_state)
	child := frame.References[COMPILE_REFERENCE_CHILD]
	if child == NODE_NONE {
		aver.Always(
			frame.Control[COMPILE_FRAME_FIRST] != CONTROL_TRUE,
			"Parsed alternative retains at least one sequence branch.",
		)
		return Alternative_Updated_Level(depth - utf8.CHARACTER_SIZE_MINIMUM),
			Alternative_Updated_PC(frame.Targets[COMPILE_TARGET_ACCUMULATED])
	}
	frame.Control[COMPILE_FRAME_STAGE] = COMPILE_STAGE_ALTERNATIVE_RETURN
	raw_depth := Push_Compile_Depth(depth * Alternative_Compile_Level(len("{}")))
	pushed := compile_push(
		workspace,
		Child_Node_Reference(child),
		Instruction_Continuation(frame.Targets[COMPILE_TARGET_CONTINUATION]),
		raw_depth,
	)
	return Alternative_Updated_Level(
		(pushed - Pushed_Compile_Depth(COMPILE_DEPTH_NONZERO_MINIMUM)) /
			Pushed_Compile_Depth(len("{}")),
	), Alternative_Updated_PC(result)
}

func compile_alternative_return(
	workspace_state *Compile_Workspace,
	frame_state *Compile_Frame,
	result Alternative_Result_PC,
) {
	Compile_Workspace_Invariants(
		workspace_state, "compile_alternative_return.workspace_state",
	)
	Compile_Frame_Invariants(
		*frame_state, "compile_alternative_return.frame_state",
	)
	Alternative_Result_PC_Invariants(result, "compile_alternative_return.result")
	workspace := (*Compile_Workspace)(workspace_state)
	frame := (*Compile_Frame)(frame_state)
	if frame.Control[COMPILE_FRAME_FIRST] == CONTROL_TRUE {
		frame.Targets[COMPILE_TARGET_ACCUMULATED] = Instruction_PC(result)
		frame.Control[COMPILE_FRAME_FIRST] = CONTROL_FALSE
	} else {
		accumulated := frame.Targets[COMPILE_TARGET_ACCUMULATED]
		instruction := Instruction{
			Kind: Instruction_Kind_Storage{INSTRUCTION_SPLIT},
			Targets: Instruction_Targets{
				INSTRUCTION_TARGET_NEXT:   Instruction_PC(result),
				INSTRUCTION_TARGET_BRANCH: accumulated,
			},
		}
		split := instruction_emit(workspace, instruction)
		frame.Targets[COMPILE_TARGET_ACCUMULATED] = split
	}
	child := frame.References[COMPILE_REFERENCE_CHILD]
	frame.References[COMPILE_REFERENCE_CHILD] =
		Node_Reference(workspace.Nodes[int(child)-utf8.CHARACTER_SIZE_MINIMUM].
			Links.Previous_Sibling)
	frame.Control[COMPILE_FRAME_STAGE] = COMPILE_STAGE_ALTERNATIVE
}

func compile_push(
	workspace_state *Compile_Workspace,
	node Child_Node_Reference,
	continuation Instruction_Continuation,
	depth Push_Compile_Depth,
) (updated_depth Pushed_Compile_Depth) {
	defer func() {
		Pushed_Compile_Depth_Invariants(updated_depth, "compile_push.updated_depth")
	}()
	Compile_Workspace_Invariants(workspace_state, "compile_push.workspace_state")
	Child_Node_Reference_Invariants(node, "compile_push.node")
	Instruction_Continuation_Invariants(continuation, "compile_push.continuation")
	Push_Compile_Depth_Invariants(depth, "compile_push.depth")
	workspace := (*Compile_Workspace)(workspace_state)
	aver.Always(
		depth < Push_Compile_Depth(len(workspace.Compile_Frames)),
		"Parsed syntax depth fits formula-derived compiler stack.",
	)
	workspace.Compile_Frames[depth] = Compile_Frame{
		References: Compile_References{COMPILE_REFERENCE_NODE: Node_Reference(node)},
		Targets: Compile_Targets{
			COMPILE_TARGET_CONTINUATION: Instruction_PC(continuation),
		},
		Control: Compile_Frame_Control{
			COMPILE_FRAME_STAGE: COMPILE_STAGE_ENTER,
		},
	}
	return Pushed_Compile_Depth(depth + utf8.CHARACTER_SIZE_MINIMUM)
}

func instruction_from_node(
	workspace_state *Compile_Workspace,
	node Leaf_Node,
	continuation Instruction_Continuation,
) (result Emitted_Instruction_PC) {
	defer func() {
		Emitted_Instruction_PC_Invariants(result, "instruction_from_node.result")
	}()
	Compile_Workspace_Invariants(
		workspace_state, "instruction_from_node.workspace_state",
	)
	Leaf_Node_Invariants(node, "instruction_from_node.node")
	Instruction_Continuation_Invariants(
		continuation, "instruction_from_node.continuation",
	)
	workspace := (*Compile_Workspace)(workspace_state)
	instruction := Instruction{
		Targets: Instruction_Targets{
			INSTRUCTION_TARGET_NEXT: Instruction_PC(continuation),
		},
	}
	switch node.Kind {
	case Leaf_Kind(NODE_LITERAL):
		instruction.Kind[INSTRUCTION_KIND_FIELD] = INSTRUCTION_LITERAL
		instruction.Character[INSTRUCTION_CHARACTER_FIELD] =
			node.Character[NODE_CHARACTER_FIELD]
	case Leaf_Kind(NODE_STAR):
		instruction.Kind[INSTRUCTION_KIND_FIELD] = INSTRUCTION_STAR
	case Leaf_Kind(NODE_SUPER_STAR):
		instruction.Kind[INSTRUCTION_KIND_FIELD] = INSTRUCTION_SUPER_STAR
	case Leaf_Kind(NODE_SINGLE):
		instruction.Kind[INSTRUCTION_KIND_FIELD] = INSTRUCTION_SINGLE
	case Leaf_Kind(NODE_CLASS):
		instruction.Kind[INSTRUCTION_KIND_FIELD] = INSTRUCTION_CLASS
		instruction.Class = Instruction_Class_Storage{
			INSTRUCTION_CLASS_RANGE_INDEX: node.Class[NODE_CLASS_RANGE_INDEX],
			INSTRUCTION_CLASS_RANGE_COUNT: node.Class[NODE_CLASS_RANGE_COUNT],
			INSTRUCTION_CLASS_NEGATED:     node.Class[NODE_CLASS_NEGATED],
		}
	}
	return Emitted_Instruction_PC(instruction_emit(workspace, instruction))
}

func instruction_emit(
	workspace_state *Compile_Workspace,
	instruction Instruction,
) (result Instruction_PC) {
	defer func() {
		Instruction_PC_Invariants(result, "instruction_emit.result")
	}()
	Compile_Workspace_Invariants(workspace_state, "instruction_emit.workspace_state")
	Instruction_Invariants(instruction, "instruction_emit.instruction")
	workspace := (*Compile_Workspace)(workspace_state)
	count := workspace.Control[COMPILE_CONTROL_INSTRUCTION_COUNT]
	aver.Always(
		int(count) < len(workspace.Instruction_Kinds),
		"One pattern byte cannot emit more than one instruction.",
	)
	workspace.Instruction_Kinds[count] = instruction.Kind[INSTRUCTION_KIND_FIELD]
	workspace.Instruction_Characters[count] =
		instruction.Character[INSTRUCTION_CHARACTER_FIELD]
	workspace.Instruction_Next_Targets[count] =
		instruction.Targets[INSTRUCTION_TARGET_NEXT]
	workspace.Instruction_Branch_Targets[count] =
		instruction.Targets[INSTRUCTION_TARGET_BRANCH]
	workspace.Instruction_Class_Indexes[count] =
		instruction.Class[INSTRUCTION_CLASS_RANGE_INDEX]
	workspace.Instruction_Class_Counts[count] =
		instruction.Class[INSTRUCTION_CLASS_RANGE_COUNT]
	workspace.Instruction_Class_Negations[count] =
		instruction.Class[INSTRUCTION_CLASS_NEGATED]
	workspace.Control[COMPILE_CONTROL_INSTRUCTION_COUNT] = count + utf8.CHARACTER_SIZE_MINIMUM
	return Instruction_PC(count)
}

func instruction_load(
	workspace_state *Compile_Workspace,
	pc Instruction_PC,
) (instruction Instruction) {
	defer func() { Instruction_Invariants(instruction, "instruction_load.instruction") }()
	Compile_Workspace_Invariants(workspace_state, "instruction_load.workspace_state")
	Instruction_PC_Invariants(pc, "instruction_load.pc")
	workspace := (*Compile_Workspace)(workspace_state)
	return Instruction{
		Kind: Instruction_Kind_Storage{workspace.Instruction_Kinds[pc]},
		Character: Instruction_Character_Storage{
			workspace.Instruction_Characters[pc],
		},
		Targets: Instruction_Targets{
			INSTRUCTION_TARGET_NEXT:   workspace.Instruction_Next_Targets[pc],
			INSTRUCTION_TARGET_BRANCH: workspace.Instruction_Branch_Targets[pc],
		},
		Class: Instruction_Class_Storage{
			INSTRUCTION_CLASS_RANGE_INDEX: workspace.Instruction_Class_Indexes[pc],
			INSTRUCTION_CLASS_RANGE_COUNT: workspace.Instruction_Class_Counts[pc],
			INSTRUCTION_CLASS_NEGATED:     workspace.Instruction_Class_Negations[pc],
		},
	}
}

func pattern_validate(pattern Pattern) (status Pattern_Status) {
	defer func() { Pattern_Status_Invariants(status, "pattern_validate.status") }()
	Pattern_Invariants(pattern, "pattern_validate.pattern")
	workspace := pattern.Workspace[PATTERN_WORKSPACE_FIELD]
	if workspace == nil {
		return STATUS_PATTERN_INVALID
	}
	instruction_count := int(
		pattern.Control[PATTERN_CONTROL_INSTRUCTION_COUNT],
	)
	separator_count := int(pattern.Control[PATTERN_CONTROL_SEPARATOR_COUNT])
	start := int(pattern.Control[PATTERN_CONTROL_START])
	if instruction_count < utf8.CHARACTER_SIZE_MINIMUM {
		return STATUS_PATTERN_INVALID
	}
	if INSTRUCTION_COUNT_MAXIMUM < instruction_count {
		return STATUS_PATTERN_INVALID
	}
	if instruction_count <= start {
		return STATUS_PATTERN_INVALID
	}
	if SEPARATOR_COUNT_MAXIMUM < separator_count {
		return STATUS_PATTERN_INVALID
	}
	if int(workspace.Control[COMPILE_CONTROL_INSTRUCTION_COUNT]) !=
		instruction_count {
		return STATUS_PATTERN_INVALID
	}
	if int(workspace.Control[COMPILE_CONTROL_SEPARATOR_COUNT]) !=
		separator_count {
		return STATUS_PATTERN_INVALID
	}
	range_count := int(workspace.Control[COMPILE_CONTROL_RANGE_COUNT])
	if CLASS_RANGE_COUNT_MAXIMUM < range_count {
		return STATUS_PATTERN_INVALID
	}
	if workspace.Instruction_Kinds[INSTRUCTION_PC_MINIMUM] != INSTRUCTION_MATCH {
		return STATUS_PATTERN_INVALID
	}
	return pattern_storage_validate(
		workspace, Instruction_Count(instruction_count),
		Class_Range_Count(range_count), Separator_Count(separator_count),
	)
}

func pattern_storage_validate(
	workspace_state *Compile_Workspace,
	instruction_count Instruction_Count,
	range_count Class_Range_Count,
	separator_count Separator_Count,
) (status Pattern_Status) {
	defer func() {
		Pattern_Status_Invariants(status, "pattern_storage_validate.status")
	}()
	Compile_Workspace_Invariants(
		workspace_state, "pattern_storage_validate.workspace_state",
	)
	Instruction_Count_Invariants(
		instruction_count, "pattern_storage_validate.instruction_count",
	)
	Class_Range_Count_Invariants(
		range_count, "pattern_storage_validate.range_count",
	)
	Separator_Count_Invariants(
		separator_count, "pattern_storage_validate.separator_count",
	)
	workspace := (*Compile_Workspace)(workspace_state)
	for index := INSTRUCTION_PC_MINIMUM; index < int(instruction_count); index++ {
		instruction := instruction_load(workspace, Instruction_PC(index))
		status = instruction_validate(instruction, instruction_count, range_count)
		if status != STATUS_OK {
			return status
		}
	}
	for index := bytes.SLICE_SIZE_MINIMUM; index < int(range_count); index++ {
		low := rune(workspace.Ranges[index].Low)
		high := rune(workspace.Ranges[index].High)
		if !bool(utf8.Valid_Character(utf8.Character(low))) {
			return STATUS_PATTERN_INVALID
		}
		if !bool(utf8.Valid_Character(utf8.Character(high))) {
			return STATUS_PATTERN_INVALID
		}
		if high < low {
			return STATUS_PATTERN_INVALID
		}
	}
	for index := SEPARATOR_COUNT_MINIMUM; index < int(separator_count); index++ {
		if !bool(utf8.Valid_Character(
			utf8.Character(workspace.Separators[index]),
		)) {
			return STATUS_PATTERN_INVALID
		}
	}
	return STATUS_OK
}

func instruction_validate(
	instruction Instruction,
	instruction_count Instruction_Count,
	range_count Class_Range_Count,
) (status Pattern_Status) {
	defer func() {
		Pattern_Status_Invariants(status, "instruction_validate.status")
	}()
	Instruction_Invariants(instruction, "instruction_validate.instruction")
	Instruction_Count_Invariants(instruction_count, "instruction_validate.count")
	Class_Range_Count_Invariants(range_count, "instruction_validate.range_count")
	kind := instruction.Kind[INSTRUCTION_KIND_FIELD]
	if kind < INSTRUCTION_MATCH {
		return STATUS_PATTERN_INVALID
	}
	if INSTRUCTION_SPLIT < kind {
		return STATUS_PATTERN_INVALID
	}
	switch kind {
	case INSTRUCTION_MATCH:
		return STATUS_OK
	case INSTRUCTION_SPLIT:
		if int(instruction.Targets[INSTRUCTION_TARGET_NEXT]) >=
			int(instruction_count) {
			return STATUS_PATTERN_INVALID
		}
		if int(instruction.Targets[INSTRUCTION_TARGET_BRANCH]) >=
			int(instruction_count) {
			return STATUS_PATTERN_INVALID
		}
	default:
		if int(instruction.Targets[INSTRUCTION_TARGET_NEXT]) >=
			int(instruction_count) {
			return STATUS_PATTERN_INVALID
		}
	}
	if kind == INSTRUCTION_LITERAL {
		if !bool(utf8.Valid_Character(utf8.Character(
			instruction.Character[INSTRUCTION_CHARACTER_FIELD],
		))) {
			return STATUS_PATTERN_INVALID
		}
	}
	if kind == INSTRUCTION_CLASS {
		class_index := int(instruction.Class[INSTRUCTION_CLASS_RANGE_INDEX])
		class_count := int(instruction.Class[INSTRUCTION_CLASS_RANGE_COUNT])
		if class_count < utf8.CHARACTER_SIZE_MINIMUM {
			return STATUS_PATTERN_INVALID
		}
		if int(range_count) < class_index+class_count {
			return STATUS_PATTERN_INVALID
		}
		negated := instruction.Class[INSTRUCTION_CLASS_NEGATED]
		switch negated {
		case uint16(CONTROL_FALSE), uint16(CONTROL_TRUE):
		default:
			return STATUS_PATTERN_INVALID
		}
	}
	return STATUS_OK
}

func state_consume(
	pattern Pattern,
	workspace_state *Match_Workspace,
	current Active_Instruction_States,
	next Instruction_State_Storage,
	character utf8.Decoded_Character,
	generation Consume_Generation,
) (next_count State_Count) {
	defer func() {
		State_Count_Invariants(next_count, "state_consume.next_count")
	}()
	Pattern_Invariants(pattern, "state_consume.pattern")
	Match_Workspace_Invariants(workspace_state, "state_consume.workspace_state")
	Active_Instruction_States_Invariants(current, "state_consume.current")
	Instruction_State_Storage_Invariants(next, "state_consume.next")
	utf8.Decoded_Character_Invariants(character, "state_consume.character")
	Consume_Generation_Invariants(generation, "state_consume.generation")
	workspace := (*Match_Workspace)(workspace_state)
	compiled := pattern.Workspace[PATTERN_WORKSPACE_FIELD]
	instruction_count := int(
		pattern.Control[PATTERN_CONTROL_INSTRUCTION_COUNT],
	)
	separator := separator_contains(pattern, character)
	for index := range current {
		pc := current[index]
		aver.Always(
			int(pc) < instruction_count,
			"Validated active state references one compiled instruction.",
		)
		instruction := instruction_load(compiled, pc)
		consume := Contains(false)
		switch instruction.Kind[INSTRUCTION_KIND_FIELD] {
		case INSTRUCTION_LITERAL:
			literal := instruction.Character[INSTRUCTION_CHARACTER_FIELD]
			consume = Contains(literal == rune(character))
		case INSTRUCTION_STAR:
			consume = !separator
		case INSTRUCTION_SUPER_STAR:
			consume = true
		case INSTRUCTION_SINGLE:
			consume = !separator
		case INSTRUCTION_CLASS:
			if !separator {
				consume = class_contains(compiled, instruction, character)
			}
		}
		if !consume {
			continue
		}
		target := instruction.Targets[INSTRUCTION_TARGET_NEXT]
		if instruction.Kind[INSTRUCTION_KIND_FIELD] == INSTRUCTION_STAR {
			target = pc
		}
		if instruction.Kind[INSTRUCTION_KIND_FIELD] == INSTRUCTION_SUPER_STAR {
			target = pc
		}
		next_count = State_Count(state_add(
			pattern, workspace, next, Append_State_Count(next_count),
			target, Closure_Generation(generation),
		))
	}
	return next_count
}

func state_add(
	pattern Pattern,
	workspace_state *Match_Workspace,
	states Instruction_State_Storage,
	state_count_value Append_State_Count,
	start Instruction_PC,
	generation Closure_Generation,
) (state_count Nonempty_State_Count) {
	defer func() {
		Nonempty_State_Count_Invariants(state_count, "state_add.state_count")
	}()
	Pattern_Invariants(pattern, "state_add.pattern")
	Match_Workspace_Invariants(workspace_state, "state_add.workspace_state")
	Instruction_State_Storage_Invariants(states, "state_add.states")
	Append_State_Count_Invariants(state_count_value, "state_add.state_count_value")
	Instruction_PC_Invariants(start, "state_add.start")
	Closure_Generation_Invariants(generation, "state_add.generation")
	workspace := (*Match_Workspace)(workspace_state)
	active_count := State_Count(state_count_value)
	compiled := pattern.Workspace[PATTERN_WORKSPACE_FIELD]
	instruction_count := int(
		pattern.Control[PATTERN_CONTROL_INSTRUCTION_COUNT],
	)
	closure := Instruction_State_Storage(workspace.Closure[:])
	closure_count := Closure_Count(bytes.SLICE_SIZE_MINIMUM)
	closure_count = state_enqueue(
		workspace, closure, Closure_Append_Count(closure_count), start, generation,
		Instruction_Count(instruction_count),
	)
	for closure_count != bytes.SLICE_SIZE_MINIMUM {
		closure_count--
		pc := closure[closure_count]
		instruction := instruction_load(compiled, pc)
		switch instruction.Kind[INSTRUCTION_KIND_FIELD] {
		case INSTRUCTION_SPLIT:
			closure_count = state_enqueue(
				workspace, closure, Closure_Append_Count(closure_count),
				instruction.Targets[INSTRUCTION_TARGET_NEXT], generation,
				Instruction_Count(instruction_count),
			)
			closure_count = state_enqueue(
				workspace, closure, Closure_Append_Count(closure_count),
				instruction.Targets[INSTRUCTION_TARGET_BRANCH], generation,
				Instruction_Count(instruction_count),
			)
		case INSTRUCTION_STAR, INSTRUCTION_SUPER_STAR:
			active_count = State_Count(state_append(
				states, Append_State_Count(active_count), pc,
			))
			closure_count = state_enqueue(
				workspace, closure, Closure_Append_Count(closure_count),
				instruction.Targets[INSTRUCTION_TARGET_NEXT], generation,
				Instruction_Count(instruction_count),
			)
		default:
			active_count = State_Count(state_append(
				states, Append_State_Count(active_count), pc,
			))
		}
	}
	return Nonempty_State_Count(active_count)
}

func state_enqueue(
	workspace_state *Match_Workspace,
	closure Instruction_State_Storage,
	closure_count_value Closure_Append_Count,
	pc Instruction_PC,
	generation Closure_Generation,
	instruction_count Instruction_Count,
) (closure_count Closure_Count) {
	defer func() {
		Closure_Count_Invariants(closure_count, "state_enqueue.closure_count")
	}()
	Match_Workspace_Invariants(workspace_state, "state_enqueue.workspace_state")
	Instruction_State_Storage_Invariants(closure, "state_enqueue.closure")
	Closure_Append_Count_Invariants(
		closure_count_value, "state_enqueue.closure_count_value",
	)
	Instruction_PC_Invariants(pc, "state_enqueue.pc")
	Closure_Generation_Invariants(generation, "state_enqueue.generation")
	Instruction_Count_Invariants(instruction_count, "state_enqueue.instruction_count")
	workspace := (*Match_Workspace)(workspace_state)
	closure_count = Closure_Count(closure_count_value)
	aver.Always(
		int(pc) < int(instruction_count),
		"Validated closure edge references one compiled instruction.",
	)
	if workspace.Seen[pc] == State_Generation(generation) {
		return closure_count
	}
	aver.Always(
		closure_count < Closure_Count(len(closure)),
		"One closure generation cannot contain duplicate instructions.",
	)
	workspace.Seen[pc] = State_Generation(generation)
	closure[closure_count] = pc
	return closure_count + Closure_Count(utf8.CHARACTER_SIZE_MINIMUM)
}

func state_append(
	states Instruction_State_Storage,
	state_count_value Append_State_Count,
	pc Instruction_PC,
) (state_count Nonempty_State_Count) {
	defer func() {
		Nonempty_State_Count_Invariants(state_count, "state_append.state_count")
	}()
	Instruction_State_Storage_Invariants(states, "state_append.states")
	Append_State_Count_Invariants(state_count_value, "state_append.state_count_value")
	Instruction_PC_Invariants(pc, "state_append.pc")
	state_count = Nonempty_State_Count(state_count_value)
	aver.Always(
		state_count < Nonempty_State_Count(len(states)),
		"One active generation cannot contain duplicate instructions.",
	)
	states[state_count] = pc
	return state_count + Nonempty_State_Count(utf8.CHARACTER_SIZE_MINIMUM)
}

func separator_contains(
	pattern Pattern,
	character utf8.Decoded_Character,
) (contains Contains) {
	defer func() { Contains_Invariants(contains, "separator_contains.contains") }()
	Pattern_Invariants(pattern, "separator_contains.pattern")
	utf8.Decoded_Character_Invariants(character, "separator_contains.character")
	workspace := pattern.Workspace[PATTERN_WORKSPACE_FIELD]
	count := int(pattern.Control[PATTERN_CONTROL_SEPARATOR_COUNT])
	for index := bytes.SLICE_SIZE_MINIMUM; index < count; index++ {
		if rune(workspace.Separators[index]) == rune(character) {
			return true
		}
	}
	return false
}

func class_contains(
	workspace_state *Compile_Workspace,
	instruction Instruction,
	character utf8.Decoded_Character,
) (contains Contains) {
	defer func() { Contains_Invariants(contains, "class_contains.contains") }()
	Compile_Workspace_Invariants(workspace_state, "class_contains.workspace_state")
	Instruction_Invariants(instruction, "class_contains.instruction")
	utf8.Decoded_Character_Invariants(character, "class_contains.character")
	workspace := (*Compile_Workspace)(workspace_state)
	index := int(instruction.Class[INSTRUCTION_CLASS_RANGE_INDEX])
	count := int(instruction.Class[INSTRUCTION_CLASS_RANGE_COUNT])
	matched := false
	for range_index := index; range_index < index+count; range_index++ {
		range_value := workspace.Ranges[range_index]
		if rune(range_value.Low) <= rune(character) {
			if rune(character) <= rune(range_value.High) {
				matched = true
				break
			}
		}
	}
	if instruction.Class[INSTRUCTION_CLASS_NEGATED] == uint16(CONTROL_TRUE) {
		matched = !matched
	}
	return Contains(matched)
}
