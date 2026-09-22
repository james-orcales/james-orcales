// Package glob compiles bounded glob patterns into caller-owned NFAs.
package glob

import "local/james-orcales/shared/bytes"
import "local/james-orcales/shared/math/bits"
import "local/james-orcales/shared/sim/aver/default"
import "local/james-orcales/shared/unicode/utf8"

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

// CLASS_MEMBER_INDEX_MAXIMUM leaves one validated member inside the range arena.
const CLASS_MEMBER_INDEX_MAXIMUM = CLASS_RANGE_COUNT_MAXIMUM - utf8.CHARACTER_SIZE_MINIMUM

// CLASS_RANGE_START_MAXIMUM leaves room for one closed class after earlier closed classes.
const CLASS_RANGE_START_MAXIMUM = PATTERN_SIZE_MAXIMUM - len("[]") - len("[a]")

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

// ALTERNATIVE_RESULT_PC_MAXIMUM reserves the minimum brace shell, which emits nothing.
const ALTERNATIVE_RESULT_PC_MAXIMUM = INSTRUCTION_PC_MAXIMUM - len("{}")

// ALTERNATIVE_UPDATED_PC_MAXIMUM matches the input bound: a split needs a comma, which costs
// the byte its emission uses.
const ALTERNATIVE_UPDATED_PC_MAXIMUM = ALTERNATIVE_RESULT_PC_MAXIMUM

// SPLIT_BRANCH_PC_MAXIMUM reserves the brace shell and the comma every split needs.
const SPLIT_BRANCH_PC_MAXIMUM = INSTRUCTION_PC_MAXIMUM - len("{,}")

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
    uint8(value),
    uint8(STATUS_OK),
    uint8(STATUS_INPUT_INVALID),
    uint8(STATUS_WORKSPACE_INVALID),
    uint8(STATUS_SYNTAX_INVALID),
  ).
    Ensure()
}

// Syntax_Status classifies parser and compiler transitions.
type Syntax_Status uint8

// Syntax_Status_Invariants admits success or malformed bounded syntax.
func Syntax_Status_Invariants(value Syntax_Status, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_Uint8(
    uint8(value),
    uint8(STATUS_OK),
    uint8(STATUS_SYNTAX_INVALID),
  ).
    Ensure()
}

// Match_Status classifies bounded NFA execution.
type Match_Status uint8

// Match_Status_Invariants lists matcher-visible refusals.
func Match_Status_Invariants(value Match_Status, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_4_Uint8(
    uint8(value),
    uint8(STATUS_OK),
    uint8(STATUS_INPUT_INVALID),
    uint8(STATUS_WORKSPACE_INVALID),
    uint8(STATUS_PATTERN_INVALID),
  ).
    Ensure()
}

// Pattern_Status classifies validated matcher state.
type Pattern_Status uint8

// Pattern_Status_Invariants admits success or hostile compiled state.
func Pattern_Status_Invariants(value Pattern_Status, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_Uint8(
    uint8(value),
    uint8(STATUS_OK),
    uint8(STATUS_PATTERN_INVALID),
  ).
    Ensure()
}

// Quote_Status classifies bounded quote output.
type Quote_Status uint8

// Quote_Status_Invariants lists quoting-visible refusals.
func Quote_Status_Invariants(value Quote_Status, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_3_Uint8(
    uint8(value),
    uint8(STATUS_OK),
    uint8(STATUS_INPUT_INVALID),
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
  value Pattern_Source_Unvalidated,
  namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Int(
    len(value),
    PATTERN_SIZE_MINIMUM,
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
    len(value),
    PATTERN_SIZE_NONEMPTY_MINIMUM,
    PATTERN_SIZE_MAXIMUM,
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
    int(value),
    PATTERN_SIZE_NONEMPTY_MINIMUM,
    PATTERN_SIZE_MAXIMUM,
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
    int(value),
    PATTERN_SIZE_NONEMPTY_MINIMUM,
    PATTERN_INDEX_MAXIMUM,
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
    int(value),
    CLASS_CHARACTER_END_MINIMUM,
    PATTERN_SIZE_MAXIMUM,
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
    int(value),
    PARSER_DEPTH_NONEMPTY_MINIMUM,
    PATTERN_DEPTH_MAXIMUM,
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
    int(value),
    PARSER_DEPTH_NONEMPTY_MINIMUM,
    PATTERN_DEPTH_MAXIMUM,
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
    int(value),
    PATTERN_DEPTH_MINIMUM,
    CLOSED_PARSER_DEPTH_MAXIMUM,
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
    len(value),
    TEXT_SIZE_MINIMUM,
    TEXT_SIZE_UNVALIDATED_MAXIMUM,
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
    len(value),
    SEPARATOR_COUNT_MINIMUM,
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
    uint16(value),
    uint16(bytes.SLICE_SIZE_MINIMUM),
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
    uint16(value),
    uint16(PATTERN_SIZE_MINIMUM),
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
func Diagnostic_Invariants(value *Diagnostic, namespace aver.Namespace) {
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
    uint8(value),
    uint8(NODE_LITERAL),
    uint8(NODE_STAR),
    uint8(NODE_SUPER_STAR),
    uint8(NODE_SINGLE),
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
    uint16(value),
    uint16(NODE_NONE),
    uint16(NODE_COUNT_MAXIMUM),
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
    uint16(value),
    uint16(NODE_ROOT),
    uint16(NODE_COUNT_MAXIMUM),
    uint16(NODE_FIRST_ALTERNATIVE),
    uint16(NODE_FIRST_ALTERNATIVE),
    uint16(NODE_FIRST_ALTERNATIVE),
  ).
    Ensure()
}

// Nested_Sequence_Reference is one sequence inside an open group.
type Nested_Sequence_Reference uint16

// Nested_Sequence_Reference_Invariants excludes root and first alternative.
func Nested_Sequence_Reference_Invariants(
  value Nested_Sequence_Reference,
  namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value),
    uint16(NESTED_SEQUENCE_REFERENCE_MINIMUM),
    uint16(NODE_COUNT_MAXIMUM),
  ).
    Ensure()
}

// Opened_Sequence_Reference is a nested sequence or unchanged full-arena current.
type Opened_Sequence_Reference uint16

// Opened_Sequence_Reference_Invariants follows nested sequence references.
func Opened_Sequence_Reference_Invariants(
  value Opened_Sequence_Reference,
  namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value),
    uint16(NESTED_SEQUENCE_REFERENCE_MINIMUM),
    uint16(NODE_COUNT_MAXIMUM),
  ).
    Ensure()
}

// Branch_Sequence_Reference follows one alternative and its first sequence.
type Branch_Sequence_Reference uint16

// Branch_Sequence_Reference_Invariants starts at the first branch-created sequence.
func Branch_Sequence_Reference_Invariants(
  value Branch_Sequence_Reference,
  namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value),
    uint16(BRANCH_SEQUENCE_REFERENCE_MINIMUM),
    uint16(NODE_COUNT_MAXIMUM),
  ).
    Ensure()
}

// Suspended_Sequence_Reference is one parent saved before an open group pair.
type Suspended_Sequence_Reference uint16

// Suspended_Sequence_Reference_Invariants reserves the opened node pair.
func Suspended_Sequence_Reference_Invariants(
  value Suspended_Sequence_Reference,
  namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Holed_Uint16(
    uint16(value),
    uint16(NODE_ROOT),
    uint16(SUSPENDED_SEQUENCE_REFERENCE_MAXIMUM),
    uint16(NODE_FIRST_ALTERNATIVE),
    uint16(NODE_FIRST_ALTERNATIVE),
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
    uint16(value),
    uint16(NODE_FIRST_ALTERNATIVE),
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
    uint16(value),
    uint16(NODE_ROOT),
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
    uint16(value),
    uint16(NODE_NONE),
    uint16(NODE_COUNT_MAXIMUM),
    uint16(NODE_ROOT),
    uint16(NODE_ROOT),
    uint16(NODE_ROOT),
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
    uint16(value),
    uint16(NODE_NONE),
    uint16(NODE_COUNT_MAXIMUM),
    uint16(NODE_ROOT),
    uint16(NODE_ROOT),
    uint16(NODE_ROOT),
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
    uint16(value),
    uint16(NODE_NONE),
    uint16(NODE_COUNT_MAXIMUM),
    uint16(NODE_ROOT),
    uint16(NODE_FIRST_ALTERNATIVE),
    uint16(NODE_FIRST_ALTERNATIVE),
  ).
    Ensure()
}

// Previous_Sibling_Reference is absent or one earlier nonroot child.
type Previous_Sibling_Reference uint16

// Previous_Sibling_Reference_Invariants leaves one later sibling arena slot.
func Previous_Sibling_Reference_Invariants(
  value Previous_Sibling_Reference,
  namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Holed_Uint16(
    uint16(value),
    uint16(NODE_NONE),
    uint16(CHILD_PARENT_REFERENCE_MAXIMUM),
    uint16(NODE_ROOT),
    uint16(NODE_ROOT),
    uint16(NODE_ROOT),
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
func Node_Links_Invariants(value *Node_Links, namespace aver.Namespace) {
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

// Node_Class_Range_Index_Slot is one value Node_Class_Range_Index_Storage addresses.
type Node_Class_Range_Index_Slot uint16

// Node_Class_Range_Index_Slot_Invariants bounds the slot value.
func Node_Class_Range_Index_Slot_Invariants(
  value Node_Class_Range_Index_Slot,
  namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value),
    uint16(bytes.SLICE_SIZE_MINIMUM),
    uint16(CLASS_RANGE_START_MAXIMUM),
  ).
    Ensure()
}

// Node_Class_Range_Index_Storage retains optional class opening storage.
type Node_Class_Range_Index_Storage *Node_Class_Range_Index_Slot

// Node_Class_Range_Index_Storage_Invariants accepts correctly typed storage.
func Node_Class_Range_Index_Storage_Invariants(
  value Node_Class_Range_Index_Storage,
  namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Node_Class_Range_Index_Slot_Invariants(*value, namespace)
}

// Node_Class_Range_Count_Slot is one value Node_Class_Range_Count_Storage addresses.
type Node_Class_Range_Count_Slot uint16

// Node_Class_Range_Count_Slot_Invariants bounds the slot value.
func Node_Class_Range_Count_Slot_Invariants(
  value Node_Class_Range_Count_Slot,
  namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value),
    uint16(bytes.SLICE_SIZE_MINIMUM),
    uint16(CLASS_RANGE_COUNT_MAXIMUM),
  ).
    Ensure()
}

// Node_Class_Range_Count_Storage retains optional class length storage.
type Node_Class_Range_Count_Storage *Node_Class_Range_Count_Slot

// Node_Class_Range_Count_Storage_Invariants accepts correctly typed storage.
func Node_Class_Range_Count_Storage_Invariants(
  value Node_Class_Range_Count_Storage,
  namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Node_Class_Range_Count_Slot_Invariants(*value, namespace)
}

// Node_Class_Negated_Slot is one value Node_Class_Negated_Storage addresses.
type Node_Class_Negated_Slot uint16

// Node_Class_Negated_Slot_Invariants bounds the slot value.
func Node_Class_Negated_Slot_Invariants(
  value Node_Class_Negated_Slot, namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Enum_Uint16(
    uint16(value),
    uint16(CONTROL_FALSE),
    uint16(CONTROL_TRUE),
  ).
    Ensure()
}

// Node_Class_Negated_Storage retains optional class polarity storage.
type Node_Class_Negated_Storage *Node_Class_Negated_Slot

// Node_Class_Negated_Storage_Invariants accepts correctly typed storage.
func Node_Class_Negated_Storage_Invariants(
  value Node_Class_Negated_Storage,
  namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Node_Class_Negated_Slot_Invariants(*value, namespace)
}

// Node_Class stores class opening, count, and negation.
type Node_Class struct {
  // Range_Index points into caller-owned class metadata.
  Range_Index Node_Class_Range_Index_Storage
  // Range_Count points into caller-owned class metadata.
  Range_Count Node_Class_Range_Count_Storage
  // Negated points into caller-owned class metadata.
  Negated Node_Class_Negated_Storage
}

// Node_Class_Invariants composes caller-owned class metadata pointers.
func Node_Class_Invariants(value *Node_Class, namespace aver.Namespace) {
  Node_Class_Range_Index_Storage_Invariants(value.Range_Index, namespace)
  Node_Class_Range_Count_Storage_Invariants(value.Range_Count, namespace)
  Node_Class_Negated_Storage_Invariants(value.Negated, namespace)
}

// NODE_CHARACTER_FIELD is the sole literal payload slot.
const NODE_CHARACTER_FIELD = bytes.SLICE_SIZE_MINIMUM

// NODE_CHARACTER_FIELD_COUNT fixes literal payload storage.
const NODE_CHARACTER_FIELD_COUNT = NODE_CHARACTER_FIELD + utf8.CHARACTER_SIZE_MINIMUM

// Node_Character_Slot is one value Node_Character_Storage addresses.
type Node_Character_Slot rune

// Node_Character_Slot_Invariants bounds the slot value.
func Node_Character_Slot_Invariants(
  value Node_Character_Slot, namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Int32(
    int32(value),
    utf8.DECODED_CHARACTER_MINIMUM,
    utf8.DECODED_CHARACTER_MAXIMUM,
  ).
    Ensure()
}

// Node_Character_Storage retains optional syntax literal storage.
type Node_Character_Storage *Node_Character_Slot

// Node_Character_Storage_Invariants accepts correctly typed storage.
func Node_Character_Storage_Invariants(
  value Node_Character_Storage, namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Node_Character_Slot_Invariants(*value, namespace)
}

// Node_Character stores one decoded literal.
type Node_Character struct {
  // Value points into caller-owned literal storage.
  Value Node_Character_Storage
}

// Node_Character_Invariants composes caller-owned literal storage.
func Node_Character_Invariants(value *Node_Character, namespace aver.Namespace) {
  Node_Character_Storage_Invariants(value.Value, namespace)
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
func Node_Invariants(value *Node, namespace aver.Namespace) {
  Node_Kind_Invariants(value.Kind, namespace)
  Node_Character_Invariants(&value.Character, namespace)
  Node_Class_Invariants(&value.Class, namespace)
  Node_Links_Invariants(&value.Links, namespace)
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
func Leaf_Node_Invariants(value *Leaf_Node, namespace aver.Namespace) {
  Leaf_Kind_Invariants(value.Kind, namespace)
  Node_Character_Invariants(&value.Character, namespace)
  Node_Class_Invariants(&value.Class, namespace)
}

// Nodes is caller-owned syntax arena.
type Nodes []Node

// Nodes_Invariants fixes complete parser capacity.
func Nodes_Invariants(value Nodes, _ aver.Namespace) {
  aver.Always(len(value) == NODE_COUNT_MAXIMUM, "Glob syntax arena has complete length.")
  aver.Always(cap(value) == NODE_COUNT_MAXIMUM, "Glob syntax arena cannot grow.")
}

// NODE_CHARACTER_VALUE_COUNT gives every syntax node one literal slot.
const NODE_CHARACTER_VALUE_COUNT = NODE_COUNT_MAXIMUM * NODE_CHARACTER_FIELD_COUNT

// Node_Character_Values back every syntax node literal view.
type Node_Character_Values []rune

// Node_Character_Values_Invariants fixes complete literal backing storage.
func Node_Character_Values_Invariants(value Node_Character_Values, _ aver.Namespace) {
  aver.Always(len(value) == NODE_CHARACTER_VALUE_COUNT, "Node literals have complete length.")
  aver.Always(cap(value) == NODE_CHARACTER_VALUE_COUNT, "Node literals cannot grow.")
}

// NODE_CLASS_VALUE_COUNT gives every syntax node complete class metadata.
const NODE_CLASS_VALUE_COUNT = NODE_COUNT_MAXIMUM * NODE_CLASS_FIELD_COUNT

// COMPILE_CONTROL_VALUE_START follows all syntax class metadata.
const COMPILE_CONTROL_VALUE_START = NODE_CLASS_VALUE_COUNT

// PATTERN_CONTROL_VALUE_START follows compiler cursors.
const PATTERN_CONTROL_VALUE_START = COMPILE_CONTROL_VALUE_START + COMPILE_CONTROL_COUNT

// UINT16_VALUE_COUNT includes syntax metadata and both control headers.
const UINT16_VALUE_COUNT = PATTERN_CONTROL_VALUE_START + PATTERN_CONTROL_COUNT

// Uint16_Values back syntax metadata and control pointers.
type Uint16_Values []uint16

// Uint16_Values_Invariants fixes complete 16-bit backing storage.
func Uint16_Values_Invariants(value Uint16_Values, _ aver.Namespace) {
  aver.Always(len(value) == UINT16_VALUE_COUNT, "16-bit values have complete length.")
  aver.Always(cap(value) == UINT16_VALUE_COUNT, "16-bit values cannot grow.")
}

// PARSER_ALTERNATIVE_REFERENCE_MAXIMUM leaves one nested sequence slot.
const PARSER_ALTERNATIVE_REFERENCE_MAXIMUM = NODE_COUNT_MAXIMUM -
  utf8.CHARACTER_SIZE_MINIMUM

// Parser_Alternative_Reference is one alternative container inside a group.
type Parser_Alternative_Reference uint16

// Parser_Alternative_Reference_Invariants reserves its first sequence child.
func Parser_Alternative_Reference_Invariants(
  value Parser_Alternative_Reference,
  namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value),
    uint16(NODE_FIRST_ALTERNATIVE),
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
  value *Parser_References, namespace aver.Namespace,
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
func Parser_Frame_Invariants(value *Parser_Frame, namespace aver.Namespace) {
  Parser_References_Invariants(&value.References, namespace)
}

// Parser_Frames is caller-owned brace-depth stack.
type Parser_Frames []Parser_Frame

// Parser_Frames_Invariants fixes valid nesting capacity.
func Parser_Frames_Invariants(value Parser_Frames, _ aver.Namespace) {
  aver.Always(len(value) == PATTERN_DEPTH_MAXIMUM, "Brace parser stack has complete length.")
  aver.Always(cap(value) == PATTERN_DEPTH_MAXIMUM, "Brace parser stack cannot grow.")
}

// Class_Low stores inclusive range opening.
type Class_Low rune

// Class_Low_Invariants covers decoder output.
func Class_Low_Invariants(value Class_Low, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int32(
    int32(value),
    utf8.DECODED_CHARACTER_MINIMUM,
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
    int32(value),
    utf8.DECODED_CHARACTER_MINIMUM,
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
func Class_Range_Invariants(value *Class_Range, namespace aver.Namespace) {
  Class_Low_Invariants(value.Low, namespace)
  Class_High_Invariants(value.High, namespace)
}

// Class_Ranges is caller-owned hostile parse storage.
type Class_Ranges []Class_Range

// Class_Ranges_Invariants fixes complete class capacity.
func Class_Ranges_Invariants(value Class_Ranges, _ aver.Namespace) {
  aver.Always(
    len(value) == CLASS_RANGE_STORAGE_COUNT_MAXIMUM,
    "Class range arena has complete length.",
  )
  aver.Always(
    cap(value) == CLASS_RANGE_STORAGE_COUNT_MAXIMUM,
    "Class range arena cannot grow.",
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
    uint8(value),
    uint8(INSTRUCTION_MATCH),
    uint8(INSTRUCTION_SPLIT),
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
    uint16(value),
    uint16(INSTRUCTION_PC_MINIMUM),
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
    uint16(value),
    uint16(INSTRUCTION_PC_MINIMUM),
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
    uint16(value),
    uint16(EMITTED_INSTRUCTION_PC_MINIMUM),
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
    uint16(value),
    uint16(INSTRUCTION_PC_MINIMUM),
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
    uint16(value),
    uint16(INSTRUCTION_PC_MINIMUM),
    uint16(ALTERNATIVE_UPDATED_PC_MAXIMUM),
  ).
    Ensure()
}

// INSTRUCTION_KIND_FIELD is the sole operation selector slot.
const INSTRUCTION_KIND_FIELD = bytes.SLICE_SIZE_MINIMUM

// INSTRUCTION_KIND_FIELD_COUNT fixes operation selector storage.
const INSTRUCTION_KIND_FIELD_COUNT = INSTRUCTION_KIND_FIELD + utf8.CHARACTER_SIZE_MINIMUM

// Instruction_Kind_Slot is one value Instruction_Kind_Storage addresses.
type Instruction_Kind_Slot uint8

// Instruction_Kind_Slot_Invariants bounds the slot value.
func Instruction_Kind_Slot_Invariants(
  value Instruction_Kind_Slot, namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint8(uint8(value), bits.WORD_8_MINIMUM, bits.WORD_8_MAXIMUM).
    Ensure()
}

// Instruction_Kind_Storage retains one operation selector.
type Instruction_Kind_Storage *Instruction_Kind_Slot

// Instruction_Kind_Storage_Invariants accepts absent or correctly typed storage.
func Instruction_Kind_Storage_Invariants(
  value Instruction_Kind_Storage, namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Instruction_Kind_Slot_Invariants(*value, namespace)
}

// INSTRUCTION_CHARACTER_FIELD is the sole literal payload slot.
const INSTRUCTION_CHARACTER_FIELD = bytes.SLICE_SIZE_MINIMUM

// INSTRUCTION_CHARACTER_FIELD_COUNT fixes literal payload storage.
const INSTRUCTION_CHARACTER_FIELD_COUNT = INSTRUCTION_CHARACTER_FIELD + utf8.CHARACTER_SIZE_MINIMUM

// INSTRUCTION_TARGET_NEXT retains primary continuation.
const INSTRUCTION_TARGET_NEXT = bytes.SLICE_SIZE_MINIMUM

// INSTRUCTION_TARGET_BRANCH retains alternative continuation.
const INSTRUCTION_TARGET_BRANCH = INSTRUCTION_TARGET_NEXT + utf8.CHARACTER_SIZE_MINIMUM

// INSTRUCTION_TARGET_COUNT fixes both continuation slots.
const INSTRUCTION_TARGET_COUNT = INSTRUCTION_TARGET_BRANCH + utf8.CHARACTER_SIZE_MINIMUM

// Instruction_Next_Slot is one value Instruction_Next_Storage addresses.
type Instruction_Next_Slot uint16

// Instruction_Next_Slot_Invariants bounds the slot value.
func Instruction_Next_Slot_Invariants(
  value Instruction_Next_Slot, namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value),
    bits.WORD_16_MINIMUM,
    bits.WORD_16_MAXIMUM,
  ).
    Ensure()
}

// Instruction_Next_Storage retains optional primary target storage.
type Instruction_Next_Storage *Instruction_Next_Slot

// Instruction_Next_Storage_Invariants accepts correctly typed storage.
func Instruction_Next_Storage_Invariants(value Instruction_Next_Storage, namespace aver.Namespace) {
  if value == nil {
    return
  }
  Instruction_Next_Slot_Invariants(*value, namespace)
}

// Instruction_Branch_Slot is one value Instruction_Branch_Storage addresses.
type Instruction_Branch_Slot uint16

// Instruction_Branch_Slot_Invariants bounds the slot value.
func Instruction_Branch_Slot_Invariants(
  value Instruction_Branch_Slot, namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value),
    bits.WORD_16_MINIMUM,
    bits.WORD_16_MAXIMUM,
  ).
    Ensure()
}

// Instruction_Branch_Storage retains optional alternative target storage.
type Instruction_Branch_Storage *Instruction_Branch_Slot

// Instruction_Branch_Storage_Invariants accepts correctly typed storage.
func Instruction_Branch_Storage_Invariants(
  value Instruction_Branch_Storage,
  namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Instruction_Branch_Slot_Invariants(*value, namespace)
}

// Instruction_Character_Slot is one value Instruction_Character_Storage addresses.
type Instruction_Character_Slot rune

// Instruction_Character_Slot_Invariants bounds the slot value.
func Instruction_Character_Slot_Invariants(
  value Instruction_Character_Slot,
  namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Int32(
    int32(value),
    bits.INTEGER_32_MINIMUM,
    bits.INTEGER_32_MAXIMUM,
  ).
    Ensure()
}

// Instruction_Character_Storage retains optional literal storage.
type Instruction_Character_Storage *Instruction_Character_Slot

// Instruction_Character_Storage_Invariants accepts correctly typed storage.
func Instruction_Character_Storage_Invariants(
  value Instruction_Character_Storage,
  namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Instruction_Character_Slot_Invariants(*value, namespace)
}

// Instruction_Class_Index_Slot is one value Instruction_Class_Index_Storage addresses.
type Instruction_Class_Index_Slot uint16

// Instruction_Class_Index_Slot_Invariants bounds the slot value.
func Instruction_Class_Index_Slot_Invariants(
  value Instruction_Class_Index_Slot,
  namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value),
    bits.WORD_16_MINIMUM,
    bits.WORD_16_MAXIMUM,
  ).
    Ensure()
}

// Instruction_Class_Index_Storage retains optional class opening storage.
type Instruction_Class_Index_Storage *Instruction_Class_Index_Slot

// Instruction_Class_Index_Storage_Invariants accepts correctly typed storage.
func Instruction_Class_Index_Storage_Invariants(
  value Instruction_Class_Index_Storage,
  namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Instruction_Class_Index_Slot_Invariants(*value, namespace)
}

// Instruction_Class_Count_Slot is one value Instruction_Class_Count_Storage addresses.
type Instruction_Class_Count_Slot uint16

// Instruction_Class_Count_Slot_Invariants bounds the slot value.
func Instruction_Class_Count_Slot_Invariants(
  value Instruction_Class_Count_Slot,
  namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value),
    bits.WORD_16_MINIMUM,
    bits.WORD_16_MAXIMUM,
  ).
    Ensure()
}

// Instruction_Class_Count_Storage retains optional class length storage.
type Instruction_Class_Count_Storage *Instruction_Class_Count_Slot

// Instruction_Class_Count_Storage_Invariants accepts correctly typed storage.
func Instruction_Class_Count_Storage_Invariants(
  value Instruction_Class_Count_Storage,
  namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Instruction_Class_Count_Slot_Invariants(*value, namespace)
}

// Instruction_Class_Negated_Slot is one value Instruction_Class_Negated_Storage addresses.
type Instruction_Class_Negated_Slot uint16

// Instruction_Class_Negated_Slot_Invariants bounds the slot value.
func Instruction_Class_Negated_Slot_Invariants(
  value Instruction_Class_Negated_Slot,
  namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value),
    bits.WORD_16_MINIMUM,
    bits.WORD_16_MAXIMUM,
  ).
    Ensure()
}

// Instruction_Class_Negated_Storage retains optional class polarity storage.
type Instruction_Class_Negated_Storage *Instruction_Class_Negated_Slot

// Instruction_Class_Negated_Storage_Invariants accepts correctly typed storage.
func Instruction_Class_Negated_Storage_Invariants(
  value Instruction_Class_Negated_Storage,
  namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Instruction_Class_Negated_Slot_Invariants(*value, namespace)
}

// INSTRUCTION_CLASS_RANGE_INDEX retains first class range.
const INSTRUCTION_CLASS_RANGE_INDEX = bytes.SLICE_SIZE_MINIMUM

// INSTRUCTION_CLASS_RANGE_COUNT retains class range length.
const INSTRUCTION_CLASS_RANGE_COUNT = INSTRUCTION_CLASS_RANGE_INDEX + utf8.CHARACTER_SIZE_MINIMUM

// INSTRUCTION_CLASS_NEGATED retains class polarity.
const INSTRUCTION_CLASS_NEGATED = INSTRUCTION_CLASS_RANGE_COUNT + utf8.CHARACTER_SIZE_MINIMUM

// INSTRUCTION_CLASS_FIELD_COUNT fixes class payload storage.
const INSTRUCTION_CLASS_FIELD_COUNT = INSTRUCTION_CLASS_NEGATED + utf8.CHARACTER_SIZE_MINIMUM

// Instruction is one immutable NFA operation.
type Instruction struct {
  // Kind selects operation.
  Kind Instruction_Kind_Storage
  // Character stores literal payload.
  Character Instruction_Character_Storage
  // Next stores primary continuation.
  Next Instruction_Next_Storage
  // Branch stores alternative continuation.
  Branch Instruction_Branch_Storage
  // Class_Index stores class range opening.
  Class_Index Instruction_Class_Index_Storage
  // Class_Count stores class range count.
  Class_Count Instruction_Class_Count_Storage
  // Class_Negated stores class polarity.
  Class_Negated Instruction_Class_Negated_Storage
}

// Instruction_Invariants composes shape-safe NFA record.
func Instruction_Invariants(value *Instruction, namespace aver.Namespace) {
  Instruction_Kind_Storage_Invariants(value.Kind, namespace)
  Instruction_Character_Storage_Invariants(value.Character, namespace)
  Instruction_Next_Storage_Invariants(value.Next, namespace)
  Instruction_Branch_Storage_Invariants(value.Branch, namespace)
  Instruction_Class_Index_Storage_Invariants(value.Class_Index, namespace)
  Instruction_Class_Count_Storage_Invariants(value.Class_Count, namespace)
  Instruction_Class_Negated_Storage_Invariants(value.Class_Negated, namespace)
}

// Instruction_Kinds stores every operation selector separately from payloads.
type Instruction_Kinds []Instruction_Kind

// Instruction_Kinds_Invariants fixes complete operation capacity.
func Instruction_Kinds_Invariants(
  value Instruction_Kinds, _ aver.Namespace,
) {
  aver.Always(len(value) == INSTRUCTION_COUNT_MAXIMUM, "NFA kind arena has complete length.")
  aver.Always(cap(value) == INSTRUCTION_COUNT_MAXIMUM, "NFA kind arena cannot grow.")
}

// Instruction_Characters stores every literal payload.
type Instruction_Characters []rune

// Instruction_Characters_Invariants fixes complete literal capacity.
func Instruction_Characters_Invariants(
  value Instruction_Characters, _ aver.Namespace,
) {
  aver.Always(
    len(value) == INSTRUCTION_COUNT_MAXIMUM,
    "NFA literal arena has complete length.",
  )
  aver.Always(cap(value) == INSTRUCTION_COUNT_MAXIMUM, "NFA literal arena cannot grow.")
}

// INSTRUCTION_TARGET_VALUE_COUNT gives every instruction both control-flow edges.
const INSTRUCTION_TARGET_VALUE_COUNT = INSTRUCTION_COUNT_MAXIMUM * INSTRUCTION_TARGET_COUNT

// Instruction_Target_Values back every immutable target view.
type Instruction_Target_Values []Instruction_PC

// Instruction_Target_Values_Invariants fixes complete target backing storage.
func Instruction_Target_Values_Invariants(
  value Instruction_Target_Values, _ aver.Namespace,
) {
  aver.Always(
    len(value) == INSTRUCTION_TARGET_VALUE_COUNT,
    "NFA targets have complete length.",
  )
  aver.Always(cap(value) == INSTRUCTION_TARGET_VALUE_COUNT, "NFA targets cannot grow.")
}

// INSTRUCTION_CLASS_VALUE_COUNT gives every instruction complete class metadata.
const INSTRUCTION_CLASS_VALUE_COUNT = INSTRUCTION_COUNT_MAXIMUM * INSTRUCTION_CLASS_FIELD_COUNT

// Instruction_Class_Values back every immutable class view.
type Instruction_Class_Values []uint16

// Instruction_Class_Values_Invariants fixes complete class backing storage.
func Instruction_Class_Values_Invariants(
  value Instruction_Class_Values, _ aver.Namespace,
) {
  aver.Always(
    len(value) == INSTRUCTION_CLASS_VALUE_COUNT,
    "NFA classes have complete length.",
  )
  aver.Always(cap(value) == INSTRUCTION_CLASS_VALUE_COUNT, "NFA classes cannot grow.")
}

// COMPILE_REFERENCE_NODE retains active syntax node.
const COMPILE_REFERENCE_NODE = bytes.SLICE_SIZE_MINIMUM

// COMPILE_REFERENCE_CHILD retains active reverse child.
const COMPILE_REFERENCE_CHILD = COMPILE_REFERENCE_NODE + utf8.CHARACTER_SIZE_MINIMUM

// COMPILE_REFERENCE_COUNT fixes compiler syntax references.
const COMPILE_REFERENCE_COUNT = COMPILE_REFERENCE_CHILD + utf8.CHARACTER_SIZE_MINIMUM

// Compile_Node_Slot is one value Compile_Node_Storage addresses.
type Compile_Node_Slot uint16

// Compile_Node_Slot_Invariants bounds the slot value.
func Compile_Node_Slot_Invariants(
  value Compile_Node_Slot, namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value),
    uint16(NODE_NONE),
    uint16(NODE_COUNT_MAXIMUM),
  ).
    Ensure()
}

// Compile_Node_Storage retains optional active syntax storage.
type Compile_Node_Storage *Compile_Node_Slot

// Compile_Node_Storage_Invariants accepts correctly typed storage.
func Compile_Node_Storage_Invariants(
  value Compile_Node_Storage, namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Compile_Node_Slot_Invariants(*value, namespace)
}

// Compile_Child_Slot is one value Compile_Child_Storage addresses.
type Compile_Child_Slot uint16

// Compile_Child_Slot_Invariants bounds the slot value.
func Compile_Child_Slot_Invariants(
  value Compile_Child_Slot, namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value),
    uint16(NODE_NONE),
    uint16(NODE_COUNT_MAXIMUM),
  ).
    Ensure()
}

// Compile_Child_Storage retains optional reverse child storage.
type Compile_Child_Storage *Compile_Child_Slot

// Compile_Child_Storage_Invariants accepts correctly typed storage.
func Compile_Child_Storage_Invariants(value Compile_Child_Storage, namespace aver.Namespace) {
  if value == nil {
    return
  }
  Compile_Child_Slot_Invariants(*value, namespace)
}

// Compile_References stores active syntax node and reverse child.
type Compile_References struct {
  // Node points into caller-owned frame storage.
  Node Compile_Node_Storage
  // Child points into caller-owned frame storage.
  Child Compile_Child_Storage
}

// Compile_References_Invariants composes compiler traversal storage.
func Compile_References_Invariants(value *Compile_References, namespace aver.Namespace) {
  Compile_Node_Storage_Invariants(value.Node, namespace)
  Compile_Child_Storage_Invariants(value.Child, namespace)
}

// COMPILE_TARGET_CONTINUATION retains caller continuation.
const COMPILE_TARGET_CONTINUATION = bytes.SLICE_SIZE_MINIMUM

// COMPILE_TARGET_ACCUMULATED retains compiled subtree entry.
const COMPILE_TARGET_ACCUMULATED = COMPILE_TARGET_CONTINUATION + utf8.CHARACTER_SIZE_MINIMUM

// COMPILE_TARGET_COUNT fixes compiler target storage.
const COMPILE_TARGET_COUNT = COMPILE_TARGET_ACCUMULATED + utf8.CHARACTER_SIZE_MINIMUM

// Compile_Continuation_Slot is one value Compile_Continuation_Storage addresses.
type Compile_Continuation_Slot uint16

// Compile_Continuation_Slot_Invariants bounds the slot value.
func Compile_Continuation_Slot_Invariants(
  value Compile_Continuation_Slot,
  namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value),
    uint16(INSTRUCTION_PC_MINIMUM),
    uint16(INSTRUCTION_PC_MAXIMUM),
  ).
    Ensure()
}

// Compile_Continuation_Storage retains optional continuation storage.
type Compile_Continuation_Storage *Compile_Continuation_Slot

// Compile_Continuation_Storage_Invariants accepts correctly typed storage.
func Compile_Continuation_Storage_Invariants(
  value Compile_Continuation_Storage,
  namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Compile_Continuation_Slot_Invariants(*value, namespace)
}

// Compile_Accumulated_Slot is one value Compile_Accumulated_Storage addresses.
type Compile_Accumulated_Slot uint16

// Compile_Accumulated_Slot_Invariants bounds the slot value.
func Compile_Accumulated_Slot_Invariants(
  value Compile_Accumulated_Slot, namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value),
    uint16(INSTRUCTION_PC_MINIMUM),
    uint16(INSTRUCTION_PC_MAXIMUM),
  ).
    Ensure()
}

// Compile_Accumulated_Storage retains optional accumulated target storage.
type Compile_Accumulated_Storage *Compile_Accumulated_Slot

// Compile_Accumulated_Storage_Invariants accepts correctly typed storage.
func Compile_Accumulated_Storage_Invariants(
  value Compile_Accumulated_Storage,
  namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Compile_Accumulated_Slot_Invariants(*value, namespace)
}

// Compile_Targets stores continuation and accumulated entry.
type Compile_Targets struct {
  // Continuation points into caller-owned frame storage.
  Continuation Compile_Continuation_Storage
  // Accumulated points into caller-owned frame storage.
  Accumulated Compile_Accumulated_Storage
}

// Compile_Targets_Invariants composes compiler target storage.
func Compile_Targets_Invariants(value *Compile_Targets, namespace aver.Namespace) {
  Compile_Continuation_Storage_Invariants(value.Continuation, namespace)
  Compile_Accumulated_Storage_Invariants(value.Accumulated, namespace)
}

// COMPILE_FRAME_STAGE retains iterative traversal stage.
const COMPILE_FRAME_STAGE = bytes.SLICE_SIZE_MINIMUM

// COMPILE_FRAME_FIRST distinguishes first alternative branch.
const COMPILE_FRAME_FIRST = COMPILE_FRAME_STAGE + utf8.CHARACTER_SIZE_MINIMUM

// COMPILE_FRAME_CONTROL_COUNT fixes compiler control storage.
const COMPILE_FRAME_CONTROL_COUNT = COMPILE_FRAME_FIRST + utf8.CHARACTER_SIZE_MINIMUM

// Compile_Stage_Slot is one value Compile_Stage_Storage addresses.
type Compile_Stage_Slot uint8

// Compile_Stage_Slot_Invariants bounds the slot value.
func Compile_Stage_Slot_Invariants(
  value Compile_Stage_Slot, namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint8(
    uint8(value),
    COMPILE_STAGE_ENTER,
    COMPILE_STAGE_ALTERNATIVE_RETURN,
  ).
    Ensure()
}

// Compile_Stage_Storage retains optional traversal stage storage.
type Compile_Stage_Storage *Compile_Stage_Slot

// Compile_Stage_Storage_Invariants accepts correctly typed storage.
func Compile_Stage_Storage_Invariants(value Compile_Stage_Storage, namespace aver.Namespace) {
  if value == nil {
    return
  }
  Compile_Stage_Slot_Invariants(*value, namespace)
}

// Compile_First_Slot is one value Compile_First_Storage addresses.
type Compile_First_Slot uint8

// Compile_First_Slot_Invariants bounds the slot value.
func Compile_First_Slot_Invariants(
  value Compile_First_Slot, namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Enum_Uint8(uint8(value), CONTROL_FALSE, CONTROL_TRUE).
    Ensure()
}

// Compile_First_Storage retains optional branch-state storage.
type Compile_First_Storage *Compile_First_Slot

// Compile_First_Storage_Invariants accepts correctly typed storage.
func Compile_First_Storage_Invariants(value Compile_First_Storage, namespace aver.Namespace) {
  if value == nil {
    return
  }
  Compile_First_Slot_Invariants(*value, namespace)
}

// Compile_Frame_Control stores traversal stage and first-branch flag.
type Compile_Frame_Control struct {
  // Stage points into caller-owned frame storage.
  Stage Compile_Stage_Storage
  // First points into caller-owned frame storage.
  First Compile_First_Storage
}

// Compile_Frame_Control_Invariants composes compiler control storage.
func Compile_Frame_Control_Invariants(value *Compile_Frame_Control, namespace aver.Namespace) {
  Compile_Stage_Storage_Invariants(value.Stage, namespace)
  Compile_First_Storage_Invariants(value.First, namespace)
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
func Compile_Frame_Invariants(value *Compile_Frame, namespace aver.Namespace) {
  Compile_References_Invariants(&value.References, namespace)
  Compile_Targets_Invariants(&value.Targets, namespace)
  Compile_Frame_Control_Invariants(&value.Control, namespace)
}

// Compile_Frames is caller-owned syntax traversal stack.
type Compile_Frames []Compile_Frame

// Compile_Frames_Invariants fixes nonrecursive compiler depth.
func Compile_Frames_Invariants(value Compile_Frames, _ aver.Namespace) {
  aver.Always(len(value) == NODE_COUNT_MAXIMUM, "NFA compiler stack has complete length.")
  aver.Always(cap(value) == NODE_COUNT_MAXIMUM, "NFA compiler stack cannot grow.")
}

// COMPILE_REFERENCE_VALUE_COUNT gives every frame both syntax references.
const COMPILE_REFERENCE_VALUE_COUNT = NODE_COUNT_MAXIMUM * COMPILE_REFERENCE_COUNT

// Compile_Reference_Values back every compiler reference view.
type Compile_Reference_Values []Node_Reference

// Compile_Reference_Values_Invariants fixes complete reference backing storage.
func Compile_Reference_Values_Invariants(
  value Compile_Reference_Values, _ aver.Namespace,
) {
  aver.Always(
    len(value) == COMPILE_REFERENCE_VALUE_COUNT,
    "Frame references have complete length.",
  )
  aver.Always(cap(value) == COMPILE_REFERENCE_VALUE_COUNT, "Frame references cannot grow.")
}

// COMPILE_TARGET_VALUE_COUNT gives every frame both instruction targets.
const COMPILE_TARGET_VALUE_COUNT = NODE_COUNT_MAXIMUM * COMPILE_TARGET_COUNT

// Compile_Target_Values back every compiler target view.
type Compile_Target_Values []Instruction_PC

// Compile_Target_Values_Invariants fixes complete target backing storage.
func Compile_Target_Values_Invariants(value Compile_Target_Values, _ aver.Namespace) {
  aver.Always(len(value) == COMPILE_TARGET_VALUE_COUNT, "Frame targets have complete length.")
  aver.Always(cap(value) == COMPILE_TARGET_VALUE_COUNT, "Frame targets cannot grow.")
}

// COMPILE_FRAME_CONTROL_VALUE_COUNT gives every frame complete traversal control.
const COMPILE_FRAME_CONTROL_VALUE_COUNT = NODE_COUNT_MAXIMUM * COMPILE_FRAME_CONTROL_COUNT

// Compile_Frame_Control_Values back every compiler control view.
type Compile_Frame_Control_Values []uint8

// Compile_Frame_Control_Values_Invariants fixes complete control backing storage.
func Compile_Frame_Control_Values_Invariants(
  value Compile_Frame_Control_Values, _ aver.Namespace,
) {
  aver.Always(
    len(value) == COMPILE_FRAME_CONTROL_VALUE_COUNT,
    "Frame control has complete length.",
  )
  aver.Always(
    cap(value) == COMPILE_FRAME_CONTROL_VALUE_COUNT,
    "Frame control cannot grow.",
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

// COMPILE_SEQUENCE_DEPTH_MAXIMUM is one frame short of the syntax arena.
const COMPILE_SEQUENCE_DEPTH_MAXIMUM = NODE_COUNT_MAXIMUM - utf8.CHARACTER_SIZE_MINIMUM

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
    int(value),
    COMPILE_DEPTH_NONZERO_MINIMUM,
    NODE_COUNT_MAXIMUM,
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
    int(value),
    COMPILE_DEPTH_NONZERO_MINIMUM,
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
    int(value),
    COMPILE_PUSHED_DEPTH_MINIMUM,
    NODE_COUNT_MAXIMUM,
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
  value Alternative_Compile_Level,
  namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Int(
    int(value),
    PARSER_DEPTH_NONEMPTY_MINIMUM,
    PATTERN_DEPTH_MAXIMUM,
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
  value Alternative_Updated_Level,
  namespace aver.Namespace,
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
    int32(value),
    utf8.CHARACTER_MINIMUM,
    utf8.CHARACTER_MAXIMUM,
  ).
    Ensure()
}

// Separators is caller-owned separator storage.
type Separators []Separator

// Separators_Invariants fixes complete separator capacity.
func Separators_Invariants(value Separators, _ aver.Namespace) {
  aver.Always(len(value) == SEPARATOR_COUNT_MAXIMUM, "Separator storage has complete length.")
  aver.Always(cap(value) == SEPARATOR_COUNT_MAXIMUM, "Separator storage cannot grow.")
}

// Compile_Node_Count_Slot is one value Compile_Node_Count_Storage addresses.
type Compile_Node_Count_Slot uint16

// Compile_Node_Count_Slot_Invariants bounds the slot value.
func Compile_Node_Count_Slot_Invariants(
  value Compile_Node_Count_Slot, namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value),
    uint16(bytes.SLICE_SIZE_MINIMUM),
    uint16(NODE_COUNT_MAXIMUM),
  ).
    Ensure()
}

// Compile_Node_Count_Storage retains optional syntax cursor storage.
type Compile_Node_Count_Storage *Compile_Node_Count_Slot

// Compile_Node_Count_Storage_Invariants accepts correctly typed storage.
func Compile_Node_Count_Storage_Invariants(
  value Compile_Node_Count_Storage,
  namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Compile_Node_Count_Slot_Invariants(*value, namespace)
}

// Compile_Range_Count_Slot is one value Compile_Range_Count_Storage addresses.
type Compile_Range_Count_Slot uint16

// Compile_Range_Count_Slot_Invariants bounds the slot value.
func Compile_Range_Count_Slot_Invariants(
  value Compile_Range_Count_Slot, namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value),
    uint16(bytes.SLICE_SIZE_MINIMUM),
    uint16(CLASS_RANGE_STORAGE_COUNT_MAXIMUM),
  ).
    Ensure()
}

// Compile_Range_Count_Storage retains optional class cursor storage.
type Compile_Range_Count_Storage *Compile_Range_Count_Slot

// Compile_Range_Count_Storage_Invariants accepts correctly typed storage.
func Compile_Range_Count_Storage_Invariants(
  value Compile_Range_Count_Storage,
  namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Compile_Range_Count_Slot_Invariants(*value, namespace)
}

// Compile_Instruction_Count_Slot is one value Compile_Instruction_Count_Storage addresses.
type Compile_Instruction_Count_Slot uint16

// Compile_Instruction_Count_Slot_Invariants bounds the slot value.
func Compile_Instruction_Count_Slot_Invariants(
  value Compile_Instruction_Count_Slot,
  namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value),
    uint16(bytes.SLICE_SIZE_MINIMUM),
    uint16(INSTRUCTION_COUNT_MAXIMUM),
  ).
    Ensure()
}

// Compile_Instruction_Count_Storage retains optional instruction cursor storage.
type Compile_Instruction_Count_Storage *Compile_Instruction_Count_Slot

// Compile_Instruction_Count_Storage_Invariants accepts correctly typed storage.
func Compile_Instruction_Count_Storage_Invariants(
  value Compile_Instruction_Count_Storage,
  namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Compile_Instruction_Count_Slot_Invariants(*value, namespace)
}

// Compile_Separator_Count_Slot is one value Compile_Separator_Count_Storage addresses.
type Compile_Separator_Count_Slot uint16

// Compile_Separator_Count_Slot_Invariants bounds the slot value.
func Compile_Separator_Count_Slot_Invariants(
  value Compile_Separator_Count_Slot,
  namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value),
    uint16(SEPARATOR_COUNT_MINIMUM),
    uint16(SEPARATOR_COUNT_MAXIMUM),
  ).
    Ensure()
}

// Compile_Separator_Count_Storage retains optional separator cursor storage.
type Compile_Separator_Count_Storage *Compile_Separator_Count_Slot

// Compile_Separator_Count_Storage_Invariants accepts correctly typed storage.
func Compile_Separator_Count_Storage_Invariants(
  value Compile_Separator_Count_Storage,
  namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Compile_Separator_Count_Slot_Invariants(*value, namespace)
}

// Compile_Control stores arena cursors only.
type Compile_Control struct {
  // Node_Count points into caller-owned cursor storage.
  Node_Count Compile_Node_Count_Storage
  // Range_Count points into caller-owned cursor storage.
  Range_Count Compile_Range_Count_Storage
  // Instruction_Count points into caller-owned cursor storage.
  Instruction_Count Compile_Instruction_Count_Storage
  // Separator_Count points into caller-owned cursor storage.
  Separator_Count Compile_Separator_Count_Storage
}

// Compile_Control_Invariants composes compiler cursor storage.
func Compile_Control_Invariants(value *Compile_Control, namespace aver.Namespace) {
  Compile_Node_Count_Storage_Invariants(value.Node_Count, namespace)
  Compile_Range_Count_Storage_Invariants(value.Range_Count, namespace)
  Compile_Instruction_Count_Storage_Invariants(value.Instruction_Count, namespace)
  Compile_Separator_Count_Storage_Invariants(value.Separator_Count, namespace)
}

// Compile_Workspace owns parser and immutable NFA storage.
type Compile_Workspace struct {
  // Nodes own flat syntax until NFA construction ends.
  Nodes Nodes
  // Node_Characters back syntax literal views.
  Node_Characters Node_Character_Values
  // Uint16_Values back syntax metadata and control pointers.
  Uint16_Values Uint16_Values
  // Parser_Frames remove recursive brace parsing.
  Parser_Frames Parser_Frames
  // Ranges own character-class intervals.
  Ranges Class_Ranges
  // Instruction_Kinds own immutable operation selectors.
  Instruction_Kinds Instruction_Kinds
  // Instruction_Characters own immutable literal payloads.
  Instruction_Characters Instruction_Characters
  // Instruction_Targets back immutable control-flow views.
  Instruction_Targets Instruction_Target_Values
  // Instruction_Classes back immutable class views.
  Instruction_Classes Instruction_Class_Values
  // Compile_Frames remove recursive syntax compilation.
  Compile_Frames Compile_Frames
  // Compile_References back compiler frame reference views.
  Compile_References Compile_Reference_Values
  // Compile_Targets back compiler frame target views.
  Compile_Targets Compile_Target_Values
  // Compile_Frame_Controls back compiler frame control views.
  Compile_Frame_Controls Compile_Frame_Control_Values
  // Separators own copied caller boundaries.
  Separators Separators
}

// Compile_Workspace_Invariants composes fixed caller-owned storage.
func Compile_Workspace_Invariants(
  value *Compile_Workspace, namespace aver.Namespace,
) {
  Nodes_Invariants(value.Nodes, namespace)
  Node_Character_Values_Invariants(value.Node_Characters, namespace)
  Uint16_Values_Invariants(value.Uint16_Values, namespace)
  Parser_Frames_Invariants(value.Parser_Frames, namespace)
  Class_Ranges_Invariants(value.Ranges, namespace)
  Instruction_Kinds_Invariants(value.Instruction_Kinds, namespace)
  Instruction_Characters_Invariants(value.Instruction_Characters, namespace)
  Instruction_Target_Values_Invariants(value.Instruction_Targets, namespace)
  Instruction_Class_Values_Invariants(value.Instruction_Classes, namespace)
  Compile_Frames_Invariants(value.Compile_Frames, namespace)
  Compile_Reference_Values_Invariants(value.Compile_References, namespace)
  Compile_Target_Values_Invariants(value.Compile_Targets, namespace)
  Compile_Frame_Control_Values_Invariants(value.Compile_Frame_Controls, namespace)
  Separators_Invariants(value.Separators, namespace)
}

// Compile_Workspace_Pointer names mutable compiler arenas.
type Compile_Workspace_Pointer *Compile_Workspace

// Compile_Workspace_Pointer_Invariants composes present compiler arenas.
func Compile_Workspace_Pointer_Invariants(
  value Compile_Workspace_Pointer,
  namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Compile_Workspace_Invariants(value, namespace)
}

// Compile_Workspace_Input carries hostile optional compile storage.
type Compile_Workspace_Input struct {
  // State points at caller storage when present.
  State Compile_Workspace_Pointer
}

// Compile_Workspace_Input_Invariants fixes optional pointer shape.
func Compile_Workspace_Input_Invariants(
  value *Compile_Workspace_Input, namespace aver.Namespace,
) {
  Compile_Workspace_Pointer_Invariants(value.State, namespace)
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
func Compile_Input_Invariants(value *Compile_Input, namespace aver.Namespace) {
  Pattern_Source_Unvalidated_Invariants(value.Source, namespace)
  Separators_Unvalidated_Invariants(value.Separators, namespace)
  Compile_Workspace_Input_Invariants(&value.Workspace, namespace)
}

// Pattern_Start_Slot is one value Pattern_Start_Storage addresses.
type Pattern_Start_Slot uint16

// Pattern_Start_Slot_Invariants bounds the slot value.
func Pattern_Start_Slot_Invariants(
  value Pattern_Start_Slot, namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value),
    uint16(INSTRUCTION_PC_MINIMUM),
    uint16(INSTRUCTION_PC_MAXIMUM),
  ).
    Ensure()
}

// Pattern_Start_Storage retains optional entry storage.
type Pattern_Start_Storage *Pattern_Start_Slot

// Pattern_Start_Storage_Invariants accepts correctly typed storage.
func Pattern_Start_Storage_Invariants(value Pattern_Start_Storage, namespace aver.Namespace) {
  if value == nil {
    return
  }
  Pattern_Start_Slot_Invariants(*value, namespace)
}

// Pattern_Instruction_Count_Slot is one value Pattern_Instruction_Count_Storage addresses.
type Pattern_Instruction_Count_Slot uint16

// Pattern_Instruction_Count_Slot_Invariants bounds the slot value.
func Pattern_Instruction_Count_Slot_Invariants(
  value Pattern_Instruction_Count_Slot,
  namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value),
    uint16(utf8.CHARACTER_SIZE_MINIMUM),
    uint16(INSTRUCTION_COUNT_MAXIMUM),
  ).
    Ensure()
}

// Pattern_Instruction_Count_Storage retains optional instruction length storage.
type Pattern_Instruction_Count_Storage *Pattern_Instruction_Count_Slot

// Pattern_Instruction_Count_Storage_Invariants accepts correctly typed storage.
func Pattern_Instruction_Count_Storage_Invariants(
  value Pattern_Instruction_Count_Storage,
  namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Pattern_Instruction_Count_Slot_Invariants(*value, namespace)
}

// Pattern_Separator_Count_Slot is one value Pattern_Separator_Count_Storage addresses.
type Pattern_Separator_Count_Slot uint16

// Pattern_Separator_Count_Slot_Invariants bounds the slot value.
func Pattern_Separator_Count_Slot_Invariants(
  value Pattern_Separator_Count_Slot,
  namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value),
    uint16(SEPARATOR_COUNT_MINIMUM),
    uint16(SEPARATOR_COUNT_MAXIMUM),
  ).
    Ensure()
}

// Pattern_Separator_Count_Storage retains optional separator length storage.
type Pattern_Separator_Count_Storage *Pattern_Separator_Count_Slot

// Pattern_Separator_Count_Storage_Invariants accepts correctly typed storage.
func Pattern_Separator_Count_Storage_Invariants(
  value Pattern_Separator_Count_Storage,
  namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Pattern_Separator_Count_Slot_Invariants(*value, namespace)
}

// Pattern_Control retains start and populated arena counts.
type Pattern_Control struct {
  // Start points into caller-owned header storage.
  Start Pattern_Start_Storage
  // Instruction_Count points into caller-owned header storage.
  Instruction_Count Pattern_Instruction_Count_Storage
  // Separator_Count points into caller-owned header storage.
  Separator_Count Pattern_Separator_Count_Storage
}

// Pattern_Control_Invariants composes compiled scalar header storage.
func Pattern_Control_Invariants(value *Pattern_Control, namespace aver.Namespace) {
  Pattern_Start_Storage_Invariants(value.Start, namespace)
  Pattern_Instruction_Count_Storage_Invariants(value.Instruction_Count, namespace)
  Pattern_Separator_Count_Storage_Invariants(value.Separator_Count, namespace)
}

// Pattern borrows immutable caller compile storage.
type Pattern struct {
  // Workspace retains compiled instructions and separators.
  Workspace Compile_Workspace_Pointer
  // Control retains start and populated counts.
  Control Pattern_Control
}

// Pattern_Invariants composes borrowed arena and header storage.
func Pattern_Invariants(value *Pattern, namespace aver.Namespace) {
  Compile_Workspace_Pointer_Invariants(value.Workspace, namespace)
  Pattern_Control_Invariants(&value.Control, namespace)
}

// Current_States stores closure before one candidate character.
type Current_States []Instruction_PC

// Current_States_Invariants fixes active-state capacity.
func Current_States_Invariants(value Current_States, _ aver.Namespace) {
  aver.Always(
    len(value) == INSTRUCTION_COUNT_MAXIMUM,
    "Current state set has complete length.",
  )
  aver.Always(cap(value) == INSTRUCTION_COUNT_MAXIMUM, "Current state set cannot grow.")
}

// Next_States stores closure after one candidate character.
type Next_States []Instruction_PC

// Next_States_Invariants fixes next-state capacity.
func Next_States_Invariants(value Next_States, _ aver.Namespace) {
  aver.Always(
    len(value) == INSTRUCTION_COUNT_MAXIMUM,
    "Next state set has complete length.",
  )
  aver.Always(cap(value) == INSTRUCTION_COUNT_MAXIMUM, "Next state set cannot grow.")
}

// Closure_States stores iterative epsilon traversal.
type Closure_States []Instruction_PC

// Closure_States_Invariants fixes epsilon traversal capacity.
func Closure_States_Invariants(value Closure_States, _ aver.Namespace) {
  aver.Always(
    len(value) == INSTRUCTION_COUNT_MAXIMUM,
    "NFA closure stack has complete length.",
  )
  aver.Always(cap(value) == INSTRUCTION_COUNT_MAXIMUM, "NFA closure stack cannot grow.")
}

// State_Generation marks one closure without clearing per transition.
type State_Generation uint16

// State_Generation_Invariants follows candidate characters plus initial closure.
func State_Generation_Invariants(
  value State_Generation, namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value),
    uint16(STATE_GENERATION_MINIMUM),
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
    uint16(value),
    uint16(CLOSURE_GENERATION_MINIMUM),
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
    uint16(value),
    uint16(CONSUME_GENERATION_MINIMUM),
    uint16(STATE_GENERATION_MAXIMUM),
  ).
    Ensure()
}

// State_Generations tracks visited instructions.
type State_Generations []State_Generation

// State_Generations_Invariants fixes visited-set capacity.
func State_Generations_Invariants(
  value State_Generations, _ aver.Namespace,
) {
  aver.Always(len(value) == INSTRUCTION_COUNT_MAXIMUM, "NFA visited set has complete length.")
  aver.Always(cap(value) == INSTRUCTION_COUNT_MAXIMUM, "NFA visited set cannot grow.")
}

// Active_Instruction_States is one bounded populated state prefix.
type Active_Instruction_States []Instruction_PC

// Active_Instruction_States_Invariants follows the compiled instruction bound.
func Active_Instruction_States_Invariants(
  value Active_Instruction_States,
  namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Int(
    len(value),
    bytes.SLICE_SIZE_MINIMUM,
    ACTIVE_STATE_COUNT_MAXIMUM,
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
    int(value),
    bytes.SLICE_SIZE_MINIMUM,
    ACTIVE_STATE_COUNT_MAXIMUM,
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
    int(value),
    bytes.SLICE_SIZE_MINIMUM,
    APPEND_STATE_COUNT_MAXIMUM,
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
    int(value),
    utf8.CHARACTER_SIZE_MINIMUM,
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
    int(value),
    bytes.SLICE_SIZE_MINIMUM,
    CLOSURE_APPEND_COUNT_MAXIMUM,
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
    int(value),
    utf8.CHARACTER_SIZE_MINIMUM,
    INSTRUCTION_COUNT_MAXIMUM,
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
    int(value),
    bytes.SLICE_SIZE_MINIMUM,
    CLASS_RANGE_COUNT_MAXIMUM,
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
    int(value),
    SEPARATOR_COUNT_MINIMUM,
    SEPARATOR_COUNT_MAXIMUM,
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
  Current_States_Invariants(value.Current, namespace)
  Next_States_Invariants(value.Next, namespace)
  Closure_States_Invariants(value.Closure, namespace)
  State_Generations_Invariants(value.Seen, namespace)
}

// Match_Workspace_Pointer names mutable matcher arenas.
type Match_Workspace_Pointer *Match_Workspace

// Match_Workspace_Pointer_Invariants composes present matcher arenas.
func Match_Workspace_Pointer_Invariants(
  value Match_Workspace_Pointer, namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Match_Workspace_Invariants(value, namespace)
}

// Match_Workspace_Input carries hostile optional match storage.
type Match_Workspace_Input struct {
  // State points at caller storage when present.
  State Match_Workspace_Pointer
}

// Match_Workspace_Input_Invariants fixes optional pointer shape.
func Match_Workspace_Input_Invariants(
  value *Match_Workspace_Input, namespace aver.Namespace,
) {
  Match_Workspace_Pointer_Invariants(value.State, namespace)
}

// Match_Input bundles immutable pattern, hostile text, and caller state.
type Match_Input struct {
  // Pattern points at caller compiled NFA.
  Pattern Pattern_Unvalidated
  // Text is hostile borrowed candidate.
  Text Text_Unvalidated
  // Workspace owns mutable NFA state sets.
  Workspace Match_Workspace_Input
}

// Match_Input_Invariants composes hostile match boundary.
func Match_Input_Invariants(value *Match_Input, namespace aver.Namespace) {
  Pattern_Unvalidated_Invariants(&value.Pattern, namespace)
  Text_Unvalidated_Invariants(value.Text, namespace)
  Match_Workspace_Input_Invariants(&value.Workspace, namespace)
}

func compile_control_initialize(
  workspace_state Compile_Workspace_Pointer,
) (control Compile_Control) {
  defer func() {
    Compile_Control_Invariants(&control, "compile_control_initialize.control")
  }()
  Compile_Workspace_Pointer_Invariants(
    workspace_state,
    "compile_control_initialize.workspace_state",
  )
  workspace := (*Compile_Workspace)(workspace_state)
  node_count_index := COMPILE_CONTROL_VALUE_START + COMPILE_CONTROL_NODE_COUNT
  range_count_index := COMPILE_CONTROL_VALUE_START + COMPILE_CONTROL_RANGE_COUNT
  instruction_count_index := COMPILE_CONTROL_VALUE_START + COMPILE_CONTROL_INSTRUCTION_COUNT
  separator_count_index := COMPILE_CONTROL_VALUE_START + COMPILE_CONTROL_SEPARATOR_COUNT
  control = Compile_Control{
    Node_Count: (*Compile_Node_Count_Slot)(
      &workspace.Uint16_Values[node_count_index],
    ),
    Range_Count: (*Compile_Range_Count_Slot)(
      &workspace.Uint16_Values[range_count_index],
    ),
    Instruction_Count: (*Compile_Instruction_Count_Slot)(
      &workspace.Uint16_Values[instruction_count_index],
    ),
    Separator_Count: (*Compile_Separator_Count_Slot)(
      &workspace.Uint16_Values[separator_count_index],
    ),
  }
  for index := PATTERN_CONTROL_VALUE_START; index < UINT16_VALUE_COUNT; index++ {
    workspace.Uint16_Values[index] = bytes.SLICE_SIZE_MINIMUM
  }
  return control
}

// Compile validates and emits one NFA into caller workspace.
func Compile(input Compile_Input) (
  _ Pattern,
  _ Diagnostic,
  status Compile_Status,
) {
  defer func() { Compile_Status_Invariants(status, "Compile.status") }()
  Compile_Input_Invariants(&input, "Compile.input")
  var pattern Pattern
  defer func() { Pattern_Invariants(&pattern, "Compile.pattern") }()
  var diagnostic Diagnostic
  defer func() { Diagnostic_Invariants(&diagnostic, "Compile.diagnostic") }()
  if len(input.Source) > PATTERN_SIZE_MAXIMUM {
    diagnostic = Diagnostic{Code: STATUS_INPUT_INVALID}
    return pattern, diagnostic, STATUS_INPUT_INVALID
  }
  if len(input.Separators) > SEPARATOR_COUNT_MAXIMUM {
    diagnostic = Diagnostic{Code: STATUS_INPUT_INVALID}
    return pattern, diagnostic, STATUS_INPUT_INVALID
  }
  workspace := input.Workspace.State
  if workspace == nil {
    diagnostic = Diagnostic{Code: STATUS_WORKSPACE_INVALID}
    return pattern, diagnostic, STATUS_WORKSPACE_INVALID
  }
  Compile_Workspace_Pointer_Invariants(workspace, "Compile.workspace")
  control := compile_control_initialize(workspace)
  for index := range input.Separators {
    if !bool(utf8.Valid_Character(utf8.Character(input.Separators[index]))) {
      diagnostic = Diagnostic{Code: STATUS_INPUT_INVALID}
      return pattern, diagnostic, STATUS_INPUT_INVALID
    }
    workspace.Separators[index] = Separator(input.Separators[index])
  }
  *control.Separator_Count = Compile_Separator_Count_Slot(uint16(len(input.Separators)))
  root, position, parse_status := pattern_parse(
    workspace.Nodes,
    workspace.Node_Characters,
    workspace.Uint16_Values,
    control.Node_Count,
    workspace.Parser_Frames,
    workspace.Ranges,
    control.Range_Count,
    Pattern_Source(input.Source),
  )
  if parse_status != STATUS_OK {
    diagnostic = Diagnostic{
      Code:     Compile_Status(parse_status),
      Position: Diagnostic_Position(position),
    }
    return pattern, diagnostic, Compile_Status(parse_status)
  }
  start := pattern_compile(
    workspace.Nodes,
    Parsed_Node_Count(*control.Node_Count),
    workspace.Compile_Frames,
    workspace.Compile_References,
    workspace.Compile_Targets,
    workspace.Compile_Frame_Controls,
    workspace.Instruction_Kinds,
    workspace.Instruction_Characters,
    workspace.Instruction_Targets,
    workspace.Instruction_Classes,
    control.Instruction_Count,
    root,
  )
  pattern = Pattern{
    Workspace: workspace,
    Control: pattern_control_store(
      workspace.Uint16_Values,
      start,
      Instruction_Count(*control.Instruction_Count),
      Separator_Count(*control.Separator_Count),
    ),
  }
  return pattern, diagnostic, STATUS_OK
}

// Match executes compiled NFA against one bounded candidate.
func Match(input Match_Input) (_ Matched, status Match_Status) {
  defer func() { Match_Status_Invariants(status, "Match.status") }()
  Match_Input_Invariants(&input, "Match.input")
  matched := Matched(false)
  defer func() { Matched_Invariants(matched, "Match.matched") }()
  if len(input.Text) > TEXT_SIZE_MAXIMUM {
    return false, STATUS_INPUT_INVALID
  }
  state := pattern_validate(input.Pattern)
  if state != STATUS_OK {
    return false, Match_Status(state)
  }
  workspace := input.Workspace.State
  if workspace == nil {
    return false, STATUS_WORKSPACE_INVALID
  }
  Match_Workspace_Pointer_Invariants(workspace, "Match.workspace")
  for index := range workspace.Seen {
    workspace.Seen[index] = State_Generation(STATE_GENERATION_MINIMUM)
  }
  compiled := input.Pattern.Workspace
  kinds, characters := compiled.Instruction_Kinds, compiled.Instruction_Characters
  targets, classes := compiled.Instruction_Targets, compiled.Instruction_Classes
  count := Instruction_Count(uint16(*input.Pattern.Control.Instruction_Count))
  separator_count := Separator_Count(uint16(*input.Pattern.Control.Separator_Count))
  current := Instruction_State_Storage(workspace.Current[:])
  next := Instruction_State_Storage(workspace.Next[:])
  generation := Closure_Generation(CLOSURE_GENERATION_MINIMUM)
  current_count := State_Count(
    state_add(
      kinds,
      characters,
      targets,
      classes,
      count,
      workspace,
      current,
      Append_State_Count(bytes.SLICE_SIZE_MINIMUM),
      Instruction_PC(uint16(*input.Pattern.Control.Start)),
      generation,
    ),
  )
  position := TEXT_SIZE_MINIMUM
  for position < len(input.Text) {
    character, size := utf8.Decode_Character(utf8.Bytes(input.Text[position:]))
    position += int(size)
    generation++
    next_count := state_consume(
      kinds,
      characters,
      targets,
      classes,
      count,
      compiled.Ranges,
      compiled.Separators,
      separator_count,
      workspace,
      Active_Instruction_States(current[:current_count]),
      next,
      utf8.Decoded_Character(character),
      Consume_Generation(generation),
    )
    current, next = next, current
    current_count = next_count
  }
  for index := bytes.SLICE_SIZE_MINIMUM; index < int(current_count); index++ {
    kind := compiled.Instruction_Kinds[current[index]]
    if kind == INSTRUCTION_MATCH {
      matched = true
      return matched, STATUS_OK
    }
  }
  return matched, STATUS_OK
}

// Quote_Meta_Into escapes metacharacters into caller destination atomically.
func Quote_Meta_Into(
  destination Output,
  source Pattern_Source_Unvalidated,
) (
  _ Output_Count,
  status Quote_Status,
) {
  defer func() { Quote_Status_Invariants(status, "Quote_Meta_Into.status") }()
  Output_Invariants(destination, "Quote_Meta_Into.destination")
  Pattern_Source_Unvalidated_Invariants(source, "Quote_Meta_Into.source")
  count := Output_Count(bytes.SLICE_SIZE_MINIMUM)
  defer func() { Output_Count_Invariants(count, "Quote_Meta_Into.count") }()
  if len(source) > PATTERN_SIZE_MAXIMUM {
    return count, STATUS_INPUT_INVALID
  }
  required_count := len(source)
  for index := range source {
    if bool(Special(Character(source[index]))) {
      required_count++
    }
  }
  if len(destination) < required_count {
    return count, STATUS_OUTPUT_TOO_SMALL
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
  count = Output_Count(position)
  return count, STATUS_OK
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
  nodes Nodes,
  characters Node_Character_Values,
  values Uint16_Values,
  node_count Compile_Node_Count_Storage,
  frames Parser_Frames,
  ranges Class_Ranges,
  range_count Compile_Range_Count_Storage,
  source Pattern_Source,
) (_ Root_Node_Reference, _ Pattern_Position, status Syntax_Status) {
  defer func() { Syntax_Status_Invariants(status, "pattern_parse.status") }()
  Nodes_Invariants(nodes, "pattern_parse.nodes")
  Node_Character_Values_Invariants(characters, "pattern_parse.characters")
  Uint16_Values_Invariants(values, "pattern_parse.values")
  Compile_Node_Count_Storage_Invariants(node_count, "pattern_parse.node_count")
  Parser_Frames_Invariants(frames, "pattern_parse.frames")
  Class_Ranges_Invariants(ranges, "pattern_parse.ranges")
  Compile_Range_Count_Storage_Invariants(range_count, "pattern_parse.range_count")
  Pattern_Source_Invariants(source, "pattern_parse.source")
  *node_count = Compile_Node_Count_Slot(bytes.SLICE_SIZE_MINIMUM)
  *range_count = Compile_Range_Count_Slot(bytes.SLICE_SIZE_MINIMUM)
  var root Root_Node_Reference
  defer func() { Root_Node_Reference_Invariants(root, "pattern_parse.root") }()
  position := Pattern_Position(PATTERN_SIZE_MINIMUM)
  defer func() { Pattern_Position_Invariants(position, "pattern_parse.position") }()
  root_reference, status := node_create(nodes, characters, values, node_count, NODE_SEQUENCE)
  root = Root_Node_Reference(root_reference)
  parsed := Parsed_Node_Count_Storage(
    (*Parsed_Node_Count)((*Compile_Node_Count_Slot)(node_count)),
  )
  starts := Class_Member_Index_Storage(
    (*Class_Member_Index)((*Compile_Range_Count_Slot)(range_count)),
  )
  current, depth := Container_Node_Reference(root), Parser_Depth(PATTERN_DEPTH_MINIMUM)
  for status == STATUS_OK {
    if position == Pattern_Position(len(source)) {
      break
    }
    current, depth, position, status = syntax_step(
      nodes,
      characters,
      values,
      parsed,
      frames,
      ranges,
      starts,
      Nonempty_Pattern_Source(source),
      current,
      depth,
      Pattern_Index(position),
    )
  }
  if status == STATUS_OK {
    if depth != PATTERN_DEPTH_MINIMUM {
      status = STATUS_SYNTAX_INVALID
    }
  }
  return root, position, status
}

func star_parse(
  nodes Nodes,
  characters Node_Character_Values,
  values Uint16_Values,
  node_count Parsed_Node_Count_Storage,
  source Nonempty_Pattern_Source,
  parent Container_Node_Reference,
  position Pattern_Index,
) (_ Nonzero_Pattern_Position, status Syntax_Status) {
  defer func() { Syntax_Status_Invariants(status, "star_parse.status") }()
  Nodes_Invariants(nodes, "star_parse.nodes")
  Node_Character_Values_Invariants(characters, "star_parse.characters")
  Uint16_Values_Invariants(values, "star_parse.values")
  Parsed_Node_Count_Storage_Invariants(node_count, "star_parse.node_count")
  Nonempty_Pattern_Source_Invariants(source, "star_parse.source")
  Container_Node_Reference_Invariants(parent, "star_parse.parent")
  Pattern_Index_Invariants(position, "star_parse.position")
  end := Nonzero_Pattern_Position(PATTERN_SIZE_NONEMPTY_MINIMUM)
  defer func() { Nonzero_Pattern_Position_Invariants(end, "star_parse.end") }()
  cursor := Pattern_Position(position) + Pattern_Position(utf8.CHARACTER_SIZE_MINIMUM)
  kind := ATOM_STAR
  if cursor < Pattern_Position(len(source)) {
    if source[cursor] == '*' {
      kind = ATOM_SUPER_STAR
      cursor++
    }
  }
  status = atom_append(
    nodes, characters, values, node_count, parent, kind, ATOM_CHARACTER_EMPTY,
  )
  end = Nonzero_Pattern_Position(cursor)
  return end, status
}

func group_open(
  nodes Nodes,
  characters Node_Character_Values,
  values Uint16_Values,
  node_count Parsed_Node_Count_Storage,
  frames Parser_Frames,
  current Container_Node_Reference,
  depth Parser_Depth,
) (
  _ Opened_Sequence_Reference,
  _ Opened_Parser_Depth,
  status Syntax_Status,
) {
  defer func() { Syntax_Status_Invariants(status, "group_open.status") }()
  Nodes_Invariants(nodes, "group_open.nodes")
  Node_Character_Values_Invariants(characters, "group_open.characters")
  Uint16_Values_Invariants(values, "group_open.values")
  Parsed_Node_Count_Storage_Invariants(node_count, "group_open.node_count")
  Parser_Frames_Invariants(frames, "group_open.frames")
  Container_Node_Reference_Invariants(current, "group_open.current")
  Parser_Depth_Invariants(depth, "group_open.depth")
  var updated_current Opened_Sequence_Reference
  defer func() {
    Opened_Sequence_Reference_Invariants(
      updated_current, "group_open.updated_current",
    )
  }()
  var updated_depth Opened_Parser_Depth
  defer func() {
    Opened_Parser_Depth_Invariants(updated_depth, "group_open.updated_depth")
  }()
  if depth == Parser_Depth(len(frames)) {
    updated_current = Opened_Sequence_Reference(current)
    updated_depth = Opened_Parser_Depth(depth)
    return updated_current, updated_depth, STATUS_SYNTAX_INVALID
  }
  alternative, status := node_create(
    nodes,
    characters,
    values,
    (*Compile_Node_Count_Slot)((*Parsed_Node_Count)(node_count)),
    NODE_ALTERNATIVE,
  )
  if status != STATUS_OK {
    updated_current = Opened_Sequence_Reference(current)
    updated_depth = Opened_Parser_Depth(depth)
    return updated_current, updated_depth, status
  }
  node_append_child(
    nodes, Child_Parent_Reference(current), Child_Node_Reference(alternative),
  )
  sequence, status := node_create(
    nodes,
    characters,
    values,
    (*Compile_Node_Count_Slot)((*Parsed_Node_Count)(node_count)),
    NODE_SEQUENCE,
  )
  if status != STATUS_OK {
    updated_current = Opened_Sequence_Reference(current)
    updated_depth = Opened_Parser_Depth(depth)
    return updated_current, updated_depth, status
  }
  node_append_child(
    nodes, Child_Parent_Reference(alternative), Child_Node_Reference(sequence),
  )
  frames[depth] = Parser_Frame{
    References: Parser_References{
      Parent:      Suspended_Sequence_Reference(current),
      Alternative: Parser_Alternative_Reference(alternative),
      Sequence:    Opened_Sequence_Reference(sequence),
    },
  }
  updated_current = Opened_Sequence_Reference(sequence)
  updated_depth = Opened_Parser_Depth(depth + utf8.CHARACTER_SIZE_MINIMUM)
  return updated_current, updated_depth, STATUS_OK
}

func group_branch(
  nodes Nodes,
  characters Node_Character_Values,
  values Uint16_Values,
  node_count Grouped_Node_Count_Storage,
  frames Parser_Frames,
  current Nested_Sequence_Reference,
  depth Open_Parser_Depth,
) (_ Branch_Sequence_Reference, status Syntax_Status) {
  defer func() { Syntax_Status_Invariants(status, "group_branch.status") }()
  Nodes_Invariants(nodes, "group_branch.nodes")
  Node_Character_Values_Invariants(characters, "group_branch.characters")
  Uint16_Values_Invariants(values, "group_branch.values")
  Grouped_Node_Count_Storage_Invariants(node_count, "group_branch.node_count")
  Parser_Frames_Invariants(frames, "group_branch.frames")
  Nested_Sequence_Reference_Invariants(current, "group_branch.current")
  Open_Parser_Depth_Invariants(depth, "group_branch.depth")
  var updated_current Branch_Sequence_Reference
  defer func() {
    Branch_Sequence_Reference_Invariants(
      updated_current, "group_branch.updated_current",
    )
  }()
  frame := &frames[int(depth)-utf8.CHARACTER_SIZE_MINIMUM]
  sequence, status := node_create(
    nodes,
    characters,
    values,
    (*Compile_Node_Count_Slot)((*Grouped_Node_Count)(node_count)),
    NODE_SEQUENCE,
  )
  if status != STATUS_OK {
    updated_current = Branch_Sequence_Reference(current)
    return updated_current, status
  }
  node_append_child(
    nodes,
    Child_Parent_Reference(frame.References.Alternative),
    Child_Node_Reference(sequence),
  )
  frame.References.Sequence = Opened_Sequence_Reference(sequence)
  updated_current = Branch_Sequence_Reference(sequence)
  return updated_current, STATUS_OK
}

func group_close(
  frames Parser_Frames,
  depth Open_Parser_Depth,
) (_ Suspended_Sequence_Reference, updated_depth Closed_Parser_Depth) {
  defer func() {
    Closed_Parser_Depth_Invariants(updated_depth, "group_close.updated_depth")
  }()
  Parser_Frames_Invariants(frames, "group_close.frames")
  Open_Parser_Depth_Invariants(depth, "group_close.depth")
  var current Suspended_Sequence_Reference
  defer func() {
    Suspended_Sequence_Reference_Invariants(current, "group_close.current")
  }()
  depth--
  current = Suspended_Sequence_Reference(frames[depth].References.Parent)
  return current, Closed_Parser_Depth(depth)
}

func literal_parse(
  nodes Nodes,
  characters Node_Character_Values,
  values Uint16_Values,
  node_count Parsed_Node_Count_Storage,
  source Nonempty_Pattern_Source,
  parent Container_Node_Reference,
  position Pattern_Index,
) (_ Nonzero_Pattern_Position, status Syntax_Status) {
  defer func() { Syntax_Status_Invariants(status, "literal_parse.status") }()
  Nodes_Invariants(nodes, "literal_parse.nodes")
  Node_Character_Values_Invariants(characters, "literal_parse.characters")
  Uint16_Values_Invariants(values, "literal_parse.values")
  Parsed_Node_Count_Storage_Invariants(node_count, "literal_parse.node_count")
  Nonempty_Pattern_Source_Invariants(source, "literal_parse.source")
  Container_Node_Reference_Invariants(parent, "literal_parse.parent")
  Pattern_Index_Invariants(position, "literal_parse.position")
  var updated_position Nonzero_Pattern_Position
  defer func() {
    Nonzero_Pattern_Position_Invariants(
      updated_position, "literal_parse.updated_position",
    )
  }()
  if source[position] == '\\' {
    position++
    if position == Pattern_Index(len(source)) {
      updated_position = Nonzero_Pattern_Position(position)
      return updated_position, STATUS_SYNTAX_INVALID
    }
  }
  character, size := utf8.Decode_Character(utf8.Bytes(source[position:]))
  position += Pattern_Index(size)
  updated_position = Nonzero_Pattern_Position(position)
  return updated_position, atom_append(
    nodes,
    characters,
    values,
    node_count,
    parent,
    ATOM_LITERAL,
    character,
  )
}

func class_parse(
  nodes Nodes,
  characters Node_Character_Values,
  values Uint16_Values,
  node_count Parsed_Node_Count_Storage,
  ranges Class_Ranges,
  range_count_state Class_Member_Index_Storage,
  source Nonempty_Pattern_Source,
  parent Container_Node_Reference,
  position Pattern_Index,
) (_ Nonzero_Pattern_Position, status Syntax_Status) {
  defer func() { Syntax_Status_Invariants(status, "class_parse.status") }()
  Nodes_Invariants(nodes, "class_parse.nodes")
  Node_Character_Values_Invariants(characters, "class_parse.characters")
  Uint16_Values_Invariants(values, "class_parse.values")
  Parsed_Node_Count_Storage_Invariants(node_count, "class_parse.node_count")
  Class_Ranges_Invariants(ranges, "class_parse.ranges")
  Class_Member_Index_Storage_Invariants(range_count_state, "class_parse.range_count_state")
  Nonempty_Pattern_Source_Invariants(source, "class_parse.source")
  Container_Node_Reference_Invariants(parent, "class_parse.parent")
  Pattern_Index_Invariants(position, "class_parse.position")
  var end Nonzero_Pattern_Position
  defer func() { Nonzero_Pattern_Position_Invariants(end, "class_parse.end") }()
  cursor := Pattern_Position(position) + Pattern_Position(utf8.CHARACTER_SIZE_MINIMUM)
  negated_value := Node_Class_Negated_Slot(CONTROL_FALSE)
  if cursor < Pattern_Position(len(source)) {
    if source[cursor] == '!' {
      negated_value = Node_Class_Negated_Slot(CONTROL_TRUE)
      cursor++
    }
  }
  range_index := Node_Class_Range_Index_Slot(*range_count_state)
  end, range_count, status := class_ranges_parse(
    ranges,
    range_count_state,
    source,
    Nonzero_Pattern_Position(cursor),
  )
  if status != STATUS_OK {
    return end, status
  }
  status = class_append(
    nodes,
    characters,
    values,
    node_count,
    parent,
    range_index,
    Class_Member_Count(range_count),
    negated_value,
  )
  return end, status
}

func class_append(
  nodes Nodes,
  characters Node_Character_Values,
  values Uint16_Values,
  node_count Parsed_Node_Count_Storage,
  parent Container_Node_Reference,
  range_index Node_Class_Range_Index_Slot,
  range_count Class_Member_Count,
  negated Node_Class_Negated_Slot,
) (status Syntax_Status) {
  defer func() { Syntax_Status_Invariants(status, "class_append.status") }()
  Nodes_Invariants(nodes, "class_append.nodes")
  Node_Character_Values_Invariants(characters, "class_append.characters")
  Uint16_Values_Invariants(values, "class_append.values")
  Parsed_Node_Count_Storage_Invariants(node_count, "class_append.node_count")
  Container_Node_Reference_Invariants(parent, "class_append.parent")
  Node_Class_Range_Index_Slot_Invariants(range_index, "class_append.range_index")
  Class_Member_Count_Invariants(range_count, "class_append.range_count")
  Node_Class_Negated_Slot_Invariants(negated, "class_append.negated")
  node, create_status := node_create(
    nodes,
    characters,
    values,
    (*Compile_Node_Count_Slot)((*Parsed_Node_Count)(node_count)),
    NODE_CLASS,
  )
  if create_status != STATUS_OK {
    return create_status
  }
  class := nodes[int(node)-utf8.CHARACTER_SIZE_MINIMUM].Class
  *class.Range_Index = range_index
  *class.Range_Count = Node_Class_Range_Count_Slot(range_count)
  *class.Negated = negated
  node_append_child(
    nodes, Child_Parent_Reference(parent), Child_Node_Reference(node),
  )
  return STATUS_OK
}

func class_character(
  source Class_Pattern_Source,
  position_value Class_Character_Index,
) (
  _ utf8.Decoded_Character,
  _ Class_Character_End,
  status Syntax_Status,
) {
  defer func() { Syntax_Status_Invariants(status, "class_character.status") }()
  Class_Pattern_Source_Invariants(source, "class_character.source")
  Class_Character_Index_Invariants(
    position_value, "class_character.position_value",
  )
  character := utf8.Decoded_Character(utf8.DECODED_CHARACTER_MINIMUM)
  defer func() {
    utf8.Decoded_Character_Invariants(character, "class_character.character")
  }()
  var position Class_Character_End
  defer func() {
    Class_Character_End_Invariants(position, "class_character.position")
  }()
  position_index := Pattern_Position(position_value)
  if source[position_index] == '\\' {
    position_index++
    if position_index == Pattern_Position(len(source)) {
      position = Class_Character_End(position_index)
      return character, position, STATUS_SYNTAX_INVALID
    }
  }
  character, size := utf8.Decode_Character(utf8.Bytes(source[position_index:]))
  position = Class_Character_End(position_index + Pattern_Position(size))
  return character, position, STATUS_OK
}

func atom_append(
  nodes Nodes,
  characters Node_Character_Values,
  values Uint16_Values,
  node_count Parsed_Node_Count_Storage,
  parent Container_Node_Reference,
  kind Atom_Kind,
  character utf8.Decoded_Character,
) (status Syntax_Status) {
  defer func() { Syntax_Status_Invariants(status, "atom_append.status") }()
  Nodes_Invariants(nodes, "atom_append.nodes")
  Node_Character_Values_Invariants(characters, "atom_append.characters")
  Uint16_Values_Invariants(values, "atom_append.values")
  Parsed_Node_Count_Storage_Invariants(node_count, "atom_append.node_count")
  Container_Node_Reference_Invariants(parent, "atom_append.parent")
  Atom_Kind_Invariants(kind, "atom_append.kind")
  utf8.Decoded_Character_Invariants(character, "atom_append.character")
  node, create_status := node_create(
    nodes,
    characters,
    values,
    (*Compile_Node_Count_Slot)((*Parsed_Node_Count)(node_count)),
    Node_Kind(kind),
  )
  if create_status != STATUS_OK {
    return create_status
  }
  slot := nodes[int(node)-utf8.CHARACTER_SIZE_MINIMUM].Character.Value
  *slot = Node_Character_Slot(character)
  node_append_child(
    nodes, Child_Parent_Reference(parent), Child_Node_Reference(node),
  )
  return STATUS_OK
}

func node_create(
  nodes Nodes,
  characters Node_Character_Values,
  values Uint16_Values,
  node_count Compile_Node_Count_Storage,
  kind Node_Kind,
) (_ Node_Reference, status Syntax_Status) {
  defer func() { Syntax_Status_Invariants(status, "node_create.status") }()
  Nodes_Invariants(nodes, "node_create.nodes")
  Node_Character_Values_Invariants(characters, "node_create.characters")
  Uint16_Values_Invariants(values, "node_create.values")
  Compile_Node_Count_Storage_Invariants(node_count, "node_create.node_count")
  Node_Kind_Invariants(kind, "node_create.kind")
  reference := NODE_NONE
  defer func() { Node_Reference_Invariants(reference, "node_create.reference") }()
  count := uint16(*node_count)
  if int(count) == len(nodes) {
    return NODE_NONE, STATUS_SYNTAX_INVALID
  }
  character_start := int(count) * NODE_CHARACTER_FIELD_COUNT
  class_start := int(count) * NODE_CLASS_FIELD_COUNT
  nodes[count] = Node{
    Kind: kind,
    Character: Node_Character{
      Value: (*Node_Character_Slot)(&characters[character_start]),
    },
    Class: Node_Class{
      Range_Index: (*Node_Class_Range_Index_Slot)(
        &values[class_start+NODE_CLASS_RANGE_INDEX],
      ),
      Range_Count: (*Node_Class_Range_Count_Slot)(
        &values[class_start+NODE_CLASS_RANGE_COUNT],
      ),
      Negated: (*Node_Class_Negated_Slot)(
        &values[class_start+NODE_CLASS_NEGATED],
      ),
    },
  }
  count++
  *node_count = Compile_Node_Count_Slot(count)
  reference = Node_Reference(count)
  return reference, STATUS_OK
}

func node_append_child(
  nodes Nodes,
  parent Child_Parent_Reference,
  child Child_Node_Reference,
) {
  Nodes_Invariants(nodes, "node_append_child.nodes")
  Child_Parent_Reference_Invariants(parent, "node_append_child.parent")
  Child_Node_Reference_Invariants(child, "node_append_child.child")
  parent_node := &nodes[int(parent)-utf8.CHARACTER_SIZE_MINIMUM]
  last := parent_node.Links.Last_Child
  if last == Last_Child_Reference(NODE_NONE) {
    parent_node.Links.First_Child = First_Child_Reference(child)
  } else {
    nodes[int(last)-utf8.CHARACTER_SIZE_MINIMUM].
      Links.Next_Sibling = Next_Sibling_Reference(child)
    nodes[int(child)-utf8.CHARACTER_SIZE_MINIMUM].
      Links.Previous_Sibling = Previous_Sibling_Reference(last)
  }
  parent_node.Links.Last_Child = Last_Child_Reference(child)
}

func range_append(
  ranges Class_Ranges,
  range_count Node_Class_Range_Count_Storage,
  low utf8.Decoded_Character,
  high utf8.Decoded_Character,
) {
  Class_Ranges_Invariants(ranges, "range_append.ranges")
  Node_Class_Range_Count_Storage_Invariants(range_count, "range_append.range_count")
  utf8.Decoded_Character_Invariants(low, "range_append.low")
  utf8.Decoded_Character_Invariants(high, "range_append.high")
  count := uint16(*range_count)
  aver.Always(
    int(count) < len(ranges),
    "One class shell leaves one formula-derived range slot per source byte.",
  )
  ranges[count] = Class_Range{
    Low:  Class_Low(low),
    High: Class_High(high),
  }
  *range_count = Node_Class_Range_Count_Slot(count + utf8.CHARACTER_SIZE_MINIMUM)
}

func compile_frame_initialize(
  nodes Nodes,
  frames Compile_Frames,
  references Compile_Reference_Values,
  targets Compile_Target_Values,
  controls Compile_Frame_Control_Values,
  index_value Instruction_PC,
  node Frame_Node_Reference,
  continuation Instruction_Continuation,
) {
  Nodes_Invariants(nodes, "compile_frame_initialize.nodes")
  Compile_Frames_Invariants(frames, "compile_frame_initialize.frames")
  Compile_Reference_Values_Invariants(references, "compile_frame_initialize.references")
  Compile_Target_Values_Invariants(targets, "compile_frame_initialize.targets")
  Compile_Frame_Control_Values_Invariants(controls, "compile_frame_initialize.controls")
  Instruction_PC_Invariants(index_value, "compile_frame_initialize.index_value")
  Frame_Node_Reference_Invariants(node, "compile_frame_initialize.node")
  Instruction_Continuation_Invariants(continuation, "compile_frame_initialize.continuation")
  index := int(index_value)
  reference_start := index * COMPILE_REFERENCE_COUNT
  target_start := index * COMPILE_TARGET_COUNT
  control_start := index * COMPILE_FRAME_CONTROL_COUNT
  frame := &frames[index]
  frame.References = Compile_References{
    Node:  (*Compile_Node_Slot)(&references[reference_start+COMPILE_REFERENCE_NODE]),
    Child: (*Compile_Child_Slot)(&references[reference_start+COMPILE_REFERENCE_CHILD]),
  }
  frame.Targets = Compile_Targets{
    Continuation: (*Compile_Continuation_Slot)(
      &targets[target_start+COMPILE_TARGET_CONTINUATION],
    ),
    Accumulated: (*Compile_Accumulated_Slot)(
      &targets[target_start+COMPILE_TARGET_ACCUMULATED],
    ),
  }
  frame.Control = Compile_Frame_Control{
    Stage: (*Compile_Stage_Slot)(&controls[control_start+COMPILE_FRAME_STAGE]),
    First: (*Compile_First_Slot)(&controls[control_start+COMPILE_FRAME_FIRST]),
  }
  syntax := nodes[int(node)-utf8.CHARACTER_SIZE_MINIMUM]
  *frame.References.Node = Compile_Node_Slot(node)
  *frame.References.Child = Compile_Child_Slot(NODE_NONE)
  *frame.Targets.Continuation = Compile_Continuation_Slot(continuation)
  *frame.Targets.Accumulated = Compile_Accumulated_Slot(INSTRUCTION_PC_MINIMUM)
  *frame.Control.Stage = Compile_Stage_Slot(COMPILE_STAGE_ENTER)
  *frame.Control.First = Compile_First_Slot(CONTROL_FALSE)
  switch syntax.Kind {
  case NODE_SEQUENCE:
    *frame.References.Child = Compile_Child_Slot(syntax.Links.Last_Child)
    *frame.Targets.Accumulated = Compile_Accumulated_Slot(continuation)
    *frame.Control.Stage = Compile_Stage_Slot(COMPILE_STAGE_SEQUENCE)
  case NODE_ALTERNATIVE:
    *frame.References.Child = Compile_Child_Slot(syntax.Links.Last_Child)
    *frame.Control.First = Compile_First_Slot(CONTROL_TRUE)
    *frame.Control.Stage = Compile_Stage_Slot(COMPILE_STAGE_ALTERNATIVE)
  }
}

func pattern_compile(
  nodes Nodes,
  node_count Parsed_Node_Count,
  frames Compile_Frames,
  references Compile_Reference_Values,
  targets Compile_Target_Values,
  controls Compile_Frame_Control_Values,
  kinds Instruction_Kinds,
  characters Instruction_Characters,
  instruction_targets Instruction_Target_Values,
  classes Instruction_Class_Values,
  instruction_count Compile_Instruction_Count_Storage,
  root Root_Node_Reference,
) (start Instruction_PC) {
  defer func() { Instruction_PC_Invariants(start, "pattern_compile.start") }()
  Nodes_Invariants(nodes, "pattern_compile.nodes")
  Parsed_Node_Count_Invariants(node_count, "pattern_compile.node_count")
  Compile_Frames_Invariants(frames, "pattern_compile.frames")
  Compile_Reference_Values_Invariants(references, "pattern_compile.references")
  Compile_Target_Values_Invariants(targets, "pattern_compile.targets")
  Compile_Frame_Control_Values_Invariants(controls, "pattern_compile.controls")
  Instruction_Kinds_Invariants(kinds, "pattern_compile.kinds")
  Instruction_Characters_Invariants(characters, "pattern_compile.characters")
  Instruction_Target_Values_Invariants(
    instruction_targets, "pattern_compile.instruction_targets",
  )
  Instruction_Class_Values_Invariants(classes, "pattern_compile.classes")
  Compile_Instruction_Count_Storage_Invariants(
    instruction_count,
    "pattern_compile.instruction_count",
  )
  Root_Node_Reference_Invariants(root, "pattern_compile.root")
  compile_frame_initialize(
    nodes,
    frames,
    references,
    targets,
    controls,
    Instruction_PC(bytes.SLICE_SIZE_MINIMUM),
    Frame_Node_Reference(root),
    Instruction_Continuation(INSTRUCTION_PC_MINIMUM),
  )
  start = compile_frames(
    nodes,
    node_count,
    frames,
    references,
    targets,
    controls,
    kinds,
    characters,
    instruction_targets,
    classes,
    instruction_count,
  )
  return start
}

func compile_enter(
  nodes Nodes,
  kinds Instruction_Kinds,
  characters Instruction_Characters,
  instruction_targets Instruction_Target_Values,
  classes Instruction_Class_Values,
  instruction_count Emitted_Instruction_PC_Storage,
  node Child_Node_Reference,
  continuation Instruction_Continuation,
) (result Emitted_Instruction_PC) {
  defer func() { Emitted_Instruction_PC_Invariants(result, "compile_enter.result") }()
  Nodes_Invariants(nodes, "compile_enter.nodes")
  Instruction_Kinds_Invariants(kinds, "compile_enter.kinds")
  Instruction_Characters_Invariants(characters, "compile_enter.characters")
  Instruction_Target_Values_Invariants(
    instruction_targets, "compile_enter.instruction_targets",
  )
  Instruction_Class_Values_Invariants(classes, "compile_enter.classes")
  Emitted_Instruction_PC_Storage_Invariants(
    instruction_count, "compile_enter.instruction_count",
  )
  Child_Node_Reference_Invariants(node, "compile_enter.node")
  Instruction_Continuation_Invariants(continuation, "compile_enter.continuation")
  syntax := nodes[int(node)-utf8.CHARACTER_SIZE_MINIMUM]
  result = instruction_from_node(
    kinds,
    characters,
    instruction_targets,
    classes,
    instruction_count,
    Leaf_Node{
      Kind: Leaf_Kind(syntax.Kind), Character: syntax.Character,
      Class: syntax.Class,
    },
    continuation,
  )
  return result
}

func compile_sequence(
  nodes Nodes,
  frames Compile_Frames,
  references Compile_Reference_Values,
  targets Compile_Target_Values,
  controls Compile_Frame_Control_Values,
  child_state Last_Child_Reference_Storage,
  accumulated_state Compile_Accumulated_Storage,
  depth Sequence_Compile_Level,
  result Instruction_PC,
) (
  _ Sequence_Updated_Level,
  updated_result Instruction_PC,
) {
  defer func() {
    Instruction_PC_Invariants(updated_result, "compile_sequence.updated_result")
  }()
  Nodes_Invariants(nodes, "compile_sequence.nodes")
  Compile_Frames_Invariants(frames, "compile_sequence.frames")
  Compile_Reference_Values_Invariants(references, "compile_sequence.references")
  Compile_Target_Values_Invariants(targets, "compile_sequence.targets")
  Compile_Frame_Control_Values_Invariants(controls, "compile_sequence.controls")
  Last_Child_Reference_Storage_Invariants(child_state, "compile_sequence.child_state")
  Compile_Accumulated_Storage_Invariants(
    accumulated_state, "compile_sequence.accumulated_state",
  )
  Sequence_Compile_Level_Invariants(depth, "compile_sequence.depth")
  Instruction_PC_Invariants(result, "compile_sequence.result")
  var updated_depth Sequence_Updated_Level
  defer func() {
    Sequence_Updated_Level_Invariants(
      updated_depth, "compile_sequence.updated_depth",
    )
  }()
  child := Node_Reference(*child_state)
  if child == NODE_NONE {
    updated_depth = Sequence_Updated_Level(depth)
    updated_result = Instruction_PC(*accumulated_state)
    return updated_depth, updated_result
  }
  raw_depth := Push_Compile_Depth(
    depth*Sequence_Compile_Level(len("{}")) +
      Sequence_Compile_Level(COMPILE_DEPTH_NONZERO_MINIMUM),
  )
  pushed := compile_push(
    nodes,
    frames,
    references,
    targets,
    controls,
    Child_Node_Reference(child),
    Instruction_Continuation(*accumulated_state),
    raw_depth,
  )
  updated_depth = Sequence_Updated_Level(pushed / Pushed_Compile_Depth(len("{}")))
  updated_result = result
  return updated_depth, updated_result
}

func compile_sequence_return(
  nodes Nodes,
  child Child_Node_Reference,
) (previous Previous_Sibling_Reference) {
  defer func() {
    Previous_Sibling_Reference_Invariants(previous, "compile_sequence_return.previous")
  }()
  Nodes_Invariants(nodes, "compile_sequence_return.nodes")
  Child_Node_Reference_Invariants(child, "compile_sequence_return.child")
  previous = nodes[int(child)-utf8.CHARACTER_SIZE_MINIMUM].Links.Previous_Sibling
  return previous
}

func compile_alternative(
  nodes Nodes,
  frames Compile_Frames,
  references Compile_Reference_Values,
  targets Compile_Target_Values,
  controls Compile_Frame_Control_Values,
  child_state Branch_Reference_Storage,
  first_state Compile_First_Storage,
  accumulated_state Alternative_Updated_PC_Storage,
  continuation_state Alternative_Updated_PC_Storage,
  depth Alternative_Compile_Level,
  result Alternative_Result_PC,
) (
  _ Alternative_Updated_Level,
  updated_result Alternative_Updated_PC,
) {
  defer func() {
    Alternative_Updated_PC_Invariants(
      updated_result, "compile_alternative.updated_result",
    )
  }()
  Nodes_Invariants(nodes, "compile_alternative.nodes")
  Compile_Frames_Invariants(frames, "compile_alternative.frames")
  Compile_Reference_Values_Invariants(references, "compile_alternative.references")
  Compile_Target_Values_Invariants(targets, "compile_alternative.targets")
  Compile_Frame_Control_Values_Invariants(controls, "compile_alternative.controls")
  Branch_Reference_Storage_Invariants(child_state, "compile_alternative.child_state")
  Compile_First_Storage_Invariants(first_state, "compile_alternative.first_state")
  Alternative_Updated_PC_Storage_Invariants(
    accumulated_state,
    "compile_alternative.accumulated_state",
  )
  Alternative_Updated_PC_Storage_Invariants(
    continuation_state,
    "compile_alternative.continuation_state",
  )
  Alternative_Compile_Level_Invariants(depth, "compile_alternative.depth")
  Alternative_Result_PC_Invariants(result, "compile_alternative.result")
  var updated_depth Alternative_Updated_Level
  defer func() {
    Alternative_Updated_Level_Invariants(
      updated_depth, "compile_alternative.updated_depth",
    )
  }()
  child := Node_Reference(*child_state)
  if child == NODE_NONE {
    aver.Always(
      uint8(*first_state) != CONTROL_TRUE,
      "Parsed alternative retains at least one sequence branch.",
    )
    updated_depth = Alternative_Updated_Level(depth - utf8.CHARACTER_SIZE_MINIMUM)
    updated_result = *accumulated_state
    return updated_depth, updated_result
  }
  raw_depth := Push_Compile_Depth(depth * Alternative_Compile_Level(len("{}")))
  pushed := compile_push(
    nodes,
    frames,
    references,
    targets,
    controls,
    Child_Node_Reference(child),
    Instruction_Continuation(*continuation_state),
    raw_depth,
  )
  updated_depth = Alternative_Updated_Level(
    (pushed - Pushed_Compile_Depth(COMPILE_DEPTH_NONZERO_MINIMUM)) /
      Pushed_Compile_Depth(len("{}")),
  )
  updated_result = Alternative_Updated_PC(result)
  return updated_depth, updated_result
}

func compile_alternative_return(
  nodes Nodes,
  kinds Instruction_Kinds,
  characters Instruction_Characters,
  instruction_targets Instruction_Target_Values,
  classes Instruction_Class_Values,
  instruction_count Alternative_Emit_PC_Storage,
  child Opened_Sequence_Reference,
  first_state Compile_First_Storage,
  accumulated Split_Branch_PC,
  result Alternative_Result_PC,
) (previous Previous_Branch_Reference, _ Alternative_Updated_PC) {
  defer func() {
    Previous_Branch_Reference_Invariants(
      previous, "compile_alternative_return.previous",
    )
  }()
  Nodes_Invariants(nodes, "compile_alternative_return.nodes")
  Instruction_Kinds_Invariants(kinds, "compile_alternative_return.kinds")
  Instruction_Characters_Invariants(characters, "compile_alternative_return.characters")
  Instruction_Target_Values_Invariants(
    instruction_targets,
    "compile_alternative_return.instruction_targets",
  )
  Instruction_Class_Values_Invariants(classes, "compile_alternative_return.classes")
  Alternative_Emit_PC_Storage_Invariants(
    instruction_count,
    "compile_alternative_return.instruction_count",
  )
  Opened_Sequence_Reference_Invariants(child, "compile_alternative_return.child")
  Compile_First_Storage_Invariants(first_state, "compile_alternative_return.first_state")
  Split_Branch_PC_Invariants(accumulated, "compile_alternative_return.accumulated")
  Alternative_Result_PC_Invariants(result, "compile_alternative_return.result")
  var updated Alternative_Updated_PC
  defer func() {
    Alternative_Updated_PC_Invariants(updated, "compile_alternative_return.updated")
  }()
  if uint8(*first_state) == CONTROL_TRUE {
    updated = Alternative_Updated_PC(result)
    *first_state = Compile_First_Slot(CONTROL_FALSE)
  } else {
    split := instruction_emit(
      kinds,
      characters,
      instruction_targets,
      classes,
      (*Instruction_PC)((*Alternative_Emit_PC)(instruction_count)),
      INSTRUCTION_SPLIT,
      Node_Character_Slot(utf8.DECODED_CHARACTER_MINIMUM),
      Instruction_Continuation(result),
      accumulated,
      Node_Class_Range_Index_Slot(bytes.SLICE_SIZE_MINIMUM),
      Node_Class_Range_Count_Slot(bytes.SLICE_SIZE_MINIMUM),
      Node_Class_Negated_Slot(CONTROL_FALSE),
    )
    updated = Alternative_Updated_PC(split)
  }
  previous = Previous_Branch_Reference(
    nodes[int(child)-utf8.CHARACTER_SIZE_MINIMUM].Links.Previous_Sibling,
  )
  return previous, updated
}

func compile_push(
  nodes Nodes,
  frames Compile_Frames,
  references Compile_Reference_Values,
  targets Compile_Target_Values,
  controls Compile_Frame_Control_Values,
  node Child_Node_Reference,
  continuation Instruction_Continuation,
  depth Push_Compile_Depth,
) (updated_depth Pushed_Compile_Depth) {
  defer func() {
    Pushed_Compile_Depth_Invariants(updated_depth, "compile_push.updated_depth")
  }()
  Nodes_Invariants(nodes, "compile_push.nodes")
  Compile_Frames_Invariants(frames, "compile_push.frames")
  Compile_Reference_Values_Invariants(references, "compile_push.references")
  Compile_Target_Values_Invariants(targets, "compile_push.targets")
  Compile_Frame_Control_Values_Invariants(controls, "compile_push.controls")
  Child_Node_Reference_Invariants(node, "compile_push.node")
  Instruction_Continuation_Invariants(continuation, "compile_push.continuation")
  Push_Compile_Depth_Invariants(depth, "compile_push.depth")
  aver.Always(
    depth < Push_Compile_Depth(len(frames)),
    "Parsed syntax depth fits formula-derived compiler stack.",
  )
  compile_frame_initialize(
    nodes,
    frames,
    references,
    targets,
    controls,
    Instruction_PC(depth),
    Frame_Node_Reference(node),
    continuation,
  )
  return Pushed_Compile_Depth(depth + utf8.CHARACTER_SIZE_MINIMUM)
}

func instruction_from_node(
  kinds Instruction_Kinds,
  characters Instruction_Characters,
  targets Instruction_Target_Values,
  classes Instruction_Class_Values,
  instruction_count Emitted_Instruction_PC_Storage,
  node Leaf_Node,
  continuation Instruction_Continuation,
) (result Emitted_Instruction_PC) {
  defer func() {
    Emitted_Instruction_PC_Invariants(result, "instruction_from_node.result")
  }()
  Instruction_Kinds_Invariants(kinds, "instruction_from_node.kinds")
  Instruction_Characters_Invariants(characters, "instruction_from_node.characters")
  Instruction_Target_Values_Invariants(targets, "instruction_from_node.targets")
  Instruction_Class_Values_Invariants(classes, "instruction_from_node.classes")
  Emitted_Instruction_PC_Storage_Invariants(
    instruction_count,
    "instruction_from_node.instruction_count",
  )
  Leaf_Node_Invariants(&node, "instruction_from_node.node")
  Instruction_Continuation_Invariants(
    continuation, "instruction_from_node.continuation",
  )
  kind := INSTRUCTION_MATCH
  switch node.Kind {
  case Leaf_Kind(NODE_LITERAL):
    kind = INSTRUCTION_LITERAL
  case Leaf_Kind(NODE_STAR):
    kind = INSTRUCTION_STAR
  case Leaf_Kind(NODE_SUPER_STAR):
    kind = INSTRUCTION_SUPER_STAR
  case Leaf_Kind(NODE_SINGLE):
    kind = INSTRUCTION_SINGLE
  case Leaf_Kind(NODE_CLASS):
    kind = INSTRUCTION_CLASS
  }
  result = Emitted_Instruction_PC(
    instruction_emit(
      kinds,
      characters,
      targets,
      classes,
      (*Instruction_PC)((*Emitted_Instruction_PC)(instruction_count)),
      kind,
      *node.Character.Value,
      continuation,
      Split_Branch_PC(INSTRUCTION_PC_MINIMUM),
      *node.Class.Range_Index,
      *node.Class.Range_Count,
      *node.Class.Negated,
    ),
  )
  return result
}

func instruction_emit(
  kinds Instruction_Kinds,
  characters Instruction_Characters,
  targets Instruction_Target_Values,
  classes Instruction_Class_Values,
  instruction_count Instruction_PC_Storage,
  kind Instruction_Kind,
  character Node_Character_Slot,
  next Instruction_Continuation,
  branch Split_Branch_PC,
  class_index Node_Class_Range_Index_Slot,
  class_count Node_Class_Range_Count_Slot,
  class_negated Node_Class_Negated_Slot,
) (result Instruction_PC) {
  defer func() {
    Instruction_PC_Invariants(result, "instruction_emit.result")
  }()
  Instruction_Kinds_Invariants(kinds, "instruction_emit.kinds")
  Instruction_Characters_Invariants(characters, "instruction_emit.characters")
  Instruction_Target_Values_Invariants(targets, "instruction_emit.targets")
  Instruction_Class_Values_Invariants(classes, "instruction_emit.classes")
  Instruction_PC_Storage_Invariants(
    instruction_count, "instruction_emit.instruction_count",
  )
  Instruction_Kind_Invariants(kind, "instruction_emit.kind")
  Node_Character_Slot_Invariants(character, "instruction_emit.character")
  Instruction_Continuation_Invariants(next, "instruction_emit.next")
  Split_Branch_PC_Invariants(branch, "instruction_emit.branch")
  Node_Class_Range_Index_Slot_Invariants(class_index, "instruction_emit.class_index")
  Node_Class_Range_Count_Slot_Invariants(class_count, "instruction_emit.class_count")
  Node_Class_Negated_Slot_Invariants(class_negated, "instruction_emit.class_negated")
  count := *instruction_count
  aver.Always(
    int(count) < len(kinds),
    "One pattern byte cannot emit more than one instruction.",
  )
  target_start := int(count) * INSTRUCTION_TARGET_COUNT
  class_start := int(count) * INSTRUCTION_CLASS_FIELD_COUNT
  kinds[count] = kind
  characters[count] = rune(character)
  targets[target_start+INSTRUCTION_TARGET_NEXT] = Instruction_PC(next)
  targets[target_start+INSTRUCTION_TARGET_BRANCH] = Instruction_PC(branch)
  classes[class_start+INSTRUCTION_CLASS_RANGE_INDEX] = uint16(class_index)
  classes[class_start+INSTRUCTION_CLASS_RANGE_COUNT] = uint16(class_count)
  classes[class_start+INSTRUCTION_CLASS_NEGATED] = uint16(class_negated)
  *instruction_count = count + Instruction_PC(utf8.CHARACTER_SIZE_MINIMUM)
  return count
}

func instruction_load(
  kinds Instruction_Kinds,
  characters Instruction_Characters,
  targets Instruction_Target_Values,
  classes Instruction_Class_Values,
  pc Instruction_PC,
) (instruction Instruction) {
  defer func() { Instruction_Invariants(&instruction, "instruction_load.instruction") }()
  Instruction_Kinds_Invariants(kinds, "instruction_load.kinds")
  Instruction_Characters_Invariants(characters, "instruction_load.characters")
  Instruction_Target_Values_Invariants(targets, "instruction_load.targets")
  Instruction_Class_Values_Invariants(classes, "instruction_load.classes")
  Instruction_PC_Invariants(pc, "instruction_load.pc")
  position := int(pc)
  target_start := position * INSTRUCTION_TARGET_COUNT
  class_start := position * INSTRUCTION_CLASS_FIELD_COUNT
  next := &targets[target_start+INSTRUCTION_TARGET_NEXT]
  branch := &targets[target_start+INSTRUCTION_TARGET_BRANCH]
  class_index := &classes[class_start+INSTRUCTION_CLASS_RANGE_INDEX]
  class_count := &classes[class_start+INSTRUCTION_CLASS_RANGE_COUNT]
  class_negated := &classes[class_start+INSTRUCTION_CLASS_NEGATED]
  return Instruction{
    Kind:          (*Instruction_Kind_Slot)(&kinds[position]),
    Character:     (*Instruction_Character_Slot)(&characters[position]),
    Next:          (*Instruction_Next_Slot)(next),
    Branch:        (*Instruction_Branch_Slot)(branch),
    Class_Index:   (*Instruction_Class_Index_Slot)(class_index),
    Class_Count:   (*Instruction_Class_Count_Slot)(class_count),
    Class_Negated: (*Instruction_Class_Negated_Slot)(class_negated),
  }
}

func pattern_validate(pattern Pattern_Unvalidated) (status Pattern_Status) {
  defer func() { Pattern_Status_Invariants(status, "pattern_validate.status") }()
  Pattern_Unvalidated_Invariants(&pattern, "pattern_validate.pattern")
  workspace := pattern.Workspace
  if workspace == nil {
    return STATUS_PATTERN_INVALID
  }
  start_state := (*uint16)((*Pattern_Start_Unvalidated)(pattern.Control.Start))
  if start_state == nil {
    return STATUS_PATTERN_INVALID
  }
  instruction_count_state := (*uint16)(
    (*Pattern_Instruction_Count_Unvalidated)(
      pattern.Control.Instruction_Count,
    ),
  )
  if instruction_count_state == nil {
    return STATUS_PATTERN_INVALID
  }
  separator_count_state := (*uint16)(
    (*Pattern_Separator_Count_Unvalidated)(pattern.Control.Separator_Count),
  )
  if separator_count_state == nil {
    return STATUS_PATTERN_INVALID
  }
  instruction_count := int(*instruction_count_state)
  separator_count := int(*separator_count_state)
  start := int(*start_state)
  if instruction_count < utf8.CHARACTER_SIZE_MINIMUM {
    return STATUS_PATTERN_INVALID
  }
  if INSTRUCTION_COUNT_MAXIMUM < instruction_count {
    return STATUS_PATTERN_INVALID
  }
  if !(instruction_count > start) {
    return STATUS_PATTERN_INVALID
  }
  if SEPARATOR_COUNT_MAXIMUM < separator_count {
    return STATUS_PATTERN_INVALID
  }
  values := workspace.Uint16_Values
  control_start := COMPILE_CONTROL_VALUE_START
  if int(values[control_start+COMPILE_CONTROL_INSTRUCTION_COUNT]) != instruction_count {
    return STATUS_PATTERN_INVALID
  }
  if int(values[control_start+COMPILE_CONTROL_SEPARATOR_COUNT]) != separator_count {
    return STATUS_PATTERN_INVALID
  }
  range_count := int(values[control_start+COMPILE_CONTROL_RANGE_COUNT])
  if CLASS_RANGE_COUNT_MAXIMUM < range_count {
    return STATUS_PATTERN_INVALID
  }
  if workspace.Instruction_Kinds[INSTRUCTION_PC_MINIMUM] != INSTRUCTION_MATCH {
    return STATUS_PATTERN_INVALID
  }
  return pattern_storage_validate(
    workspace,
    Instruction_Count(instruction_count),
    Class_Range_Count(range_count),
    Separator_Count(separator_count),
  )
}

func pattern_storage_validate(
  workspace_state Compile_Workspace_Pointer,
  instruction_count Instruction_Count,
  range_count Class_Range_Count,
  separator_count Separator_Count,
) (status Pattern_Status) {
  defer func() {
    Pattern_Status_Invariants(status, "pattern_storage_validate.status")
  }()
  Compile_Workspace_Pointer_Invariants(
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
    instruction := instruction_load(
      workspace.Instruction_Kinds,
      workspace.Instruction_Characters,
      workspace.Instruction_Targets,
      workspace.Instruction_Classes,
      Instruction_PC(index),
    )
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
  Instruction_Invariants(&instruction, "instruction_validate.instruction")
  Instruction_Count_Invariants(instruction_count, "instruction_validate.count")
  Class_Range_Count_Invariants(range_count, "instruction_validate.range_count")
  kind := Instruction_Kind(*instruction.Kind)
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
    if !(int(Instruction_PC(*instruction.Next)) <
      int(instruction_count)) {
      return STATUS_PATTERN_INVALID
    }
    if !(int(Instruction_PC(*instruction.Branch)) <
      int(instruction_count)) {
      return STATUS_PATTERN_INVALID
    }
  default:
    if !(int(Instruction_PC(*instruction.Next)) <
      int(instruction_count)) {
      return STATUS_PATTERN_INVALID
    }
  }
  if kind == INSTRUCTION_LITERAL {
    if !bool(utf8.Valid_Character(
      utf8.Character(rune(*instruction.Character)),
    )) {
      return STATUS_PATTERN_INVALID
    }
  }
  if kind == INSTRUCTION_CLASS {
    class_index := int(uint16(*instruction.Class_Index))
    class_count := int(uint16(*instruction.Class_Count))
    if class_count < utf8.CHARACTER_SIZE_MINIMUM {
      return STATUS_PATTERN_INVALID
    }
    if int(range_count) < class_index+class_count {
      return STATUS_PATTERN_INVALID
    }
    negated := uint16(*instruction.Class_Negated)
    switch negated {
    case uint16(CONTROL_FALSE), uint16(CONTROL_TRUE):
    default:
      return STATUS_PATTERN_INVALID
    }
  }
  return STATUS_OK
}

func state_consume(
  kinds Instruction_Kinds,
  characters Instruction_Characters,
  targets Instruction_Target_Values,
  classes Instruction_Class_Values,
  instruction_count Instruction_Count,
  ranges Class_Ranges,
  separators Separators,
  separator_count Separator_Count,
  workspace_state Match_Workspace_Pointer,
  current Active_Instruction_States,
  next Instruction_State_Storage,
  character utf8.Decoded_Character,
  generation Consume_Generation,
) (next_count State_Count) {
  defer func() {
    State_Count_Invariants(next_count, "state_consume.next_count")
  }()
  Instruction_Kinds_Invariants(kinds, "state_consume.kinds")
  Instruction_Characters_Invariants(characters, "state_consume.characters")
  Instruction_Target_Values_Invariants(targets, "state_consume.targets")
  Instruction_Class_Values_Invariants(classes, "state_consume.classes")
  Instruction_Count_Invariants(instruction_count, "state_consume.instruction_count")
  Class_Ranges_Invariants(ranges, "state_consume.ranges")
  Separators_Invariants(separators, "state_consume.separators")
  Separator_Count_Invariants(separator_count, "state_consume.separator_count")
  Match_Workspace_Pointer_Invariants(workspace_state, "state_consume.workspace_state")
  Active_Instruction_States_Invariants(current, "state_consume.current")
  Instruction_State_Storage_Invariants(next, "state_consume.next")
  utf8.Decoded_Character_Invariants(character, "state_consume.character")
  Consume_Generation_Invariants(generation, "state_consume.generation")
  workspace := (*Match_Workspace)(workspace_state)
  separator := separator_contains(separators, separator_count, character)
  for index := range current {
    pc := current[index]
    aver.Always(
      int(pc) < int(instruction_count),
      "Validated active state references one compiled instruction.",
    )
    instruction := instruction_load(kinds, characters, targets, classes, pc)
    kind := Instruction_Kind(*instruction.Kind)
    consume := Contains(false)
    switch kind {
    case INSTRUCTION_LITERAL:
      literal := rune(*instruction.Character)
      consume = Contains(literal == rune(character))
    case INSTRUCTION_STAR:
      consume = !separator
    case INSTRUCTION_SUPER_STAR:
      consume = true
    case INSTRUCTION_SINGLE:
      consume = !separator
    case INSTRUCTION_CLASS:
      if !separator {
        consume = class_contains(
          ranges,
          Class_Member_Index(*instruction.Class_Index),
          Class_Member_Count(*instruction.Class_Count),
          Node_Class_Negated_Slot(*instruction.Class_Negated),
          character,
        )
      }
    }
    if !consume {
      continue
    }
    target := Instruction_PC(*instruction.Next)
    if kind == INSTRUCTION_STAR {
      target = pc
    }
    if kind == INSTRUCTION_SUPER_STAR {
      target = pc
    }
    next_count = State_Count(
      state_add(
        kinds,
        characters,
        targets,
        classes,
        instruction_count,
        workspace,
        next,
        Append_State_Count(next_count),
        target,
        Closure_Generation(generation),
      ),
    )
  }
  return next_count
}

func state_add(
  kinds Instruction_Kinds,
  characters Instruction_Characters,
  targets Instruction_Target_Values,
  classes Instruction_Class_Values,
  instruction_count Instruction_Count,
  workspace_state Match_Workspace_Pointer,
  states Instruction_State_Storage,
  state_count_value Append_State_Count,
  start Instruction_PC,
  generation Closure_Generation,
) (state_count Nonempty_State_Count) {
  defer func() {
    Nonempty_State_Count_Invariants(state_count, "state_add.state_count")
  }()
  Instruction_Kinds_Invariants(kinds, "state_add.kinds")
  Instruction_Characters_Invariants(characters, "state_add.characters")
  Instruction_Target_Values_Invariants(targets, "state_add.targets")
  Instruction_Class_Values_Invariants(classes, "state_add.classes")
  Instruction_Count_Invariants(instruction_count, "state_add.instruction_count")
  Match_Workspace_Pointer_Invariants(workspace_state, "state_add.workspace_state")
  Instruction_State_Storage_Invariants(states, "state_add.states")
  Append_State_Count_Invariants(state_count_value, "state_add.state_count_value")
  Instruction_PC_Invariants(start, "state_add.start")
  Closure_Generation_Invariants(generation, "state_add.generation")
  workspace := (*Match_Workspace)(workspace_state)
  active_count := State_Count(state_count_value)
  closure := Instruction_State_Storage(workspace.Closure[:])
  closure_count := Closure_Count(bytes.SLICE_SIZE_MINIMUM)
  closure_count = state_enqueue(
    workspace,
    closure,
    Closure_Append_Count(closure_count),
    start,
    generation,
    instruction_count,
  )
  for closure_count != bytes.SLICE_SIZE_MINIMUM {
    closure_count--
    pc := closure[closure_count]
    instruction := instruction_load(kinds, characters, targets, classes, pc)
    switch Instruction_Kind(*instruction.Kind) {
    case INSTRUCTION_SPLIT:
      closure_count = state_enqueue(
        workspace,
        closure,
        Closure_Append_Count(closure_count),
        Instruction_PC(*instruction.Next),
        generation,
        instruction_count,
      )
      closure_count = state_enqueue(
        workspace,
        closure,
        Closure_Append_Count(closure_count),
        Instruction_PC(*instruction.Branch),
        generation,
        instruction_count,
      )
    case INSTRUCTION_STAR, INSTRUCTION_SUPER_STAR:
      active_count = State_Count(state_append(
        states, Append_State_Count(active_count), pc,
      ))
      closure_count = state_enqueue(
        workspace,
        closure,
        Closure_Append_Count(closure_count),
        Instruction_PC(*instruction.Next),
        generation,
        instruction_count,
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
  workspace_state Match_Workspace_Pointer,
  closure Instruction_State_Storage,
  closure_count_value Closure_Append_Count,
  pc Instruction_PC,
  generation Closure_Generation,
  instruction_count Instruction_Count,
) (closure_count Closure_Count) {
  defer func() {
    Closure_Count_Invariants(closure_count, "state_enqueue.closure_count")
  }()
  Match_Workspace_Pointer_Invariants(workspace_state, "state_enqueue.workspace_state")
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
  separators Separators,
  separator_count Separator_Count,
  character utf8.Decoded_Character,
) (contains Contains) {
  defer func() { Contains_Invariants(contains, "separator_contains.contains") }()
  Separators_Invariants(separators, "separator_contains.separators")
  Separator_Count_Invariants(separator_count, "separator_contains.separator_count")
  utf8.Decoded_Character_Invariants(character, "separator_contains.character")
  for index := SEPARATOR_COUNT_MINIMUM; index < int(separator_count); index++ {
    if rune(separators[index]) == rune(character) {
      return true
    }
  }
  return false
}

func class_contains(
  ranges Class_Ranges,
  index Class_Member_Index,
  count Class_Member_Count,
  negated Node_Class_Negated_Slot,
  character utf8.Decoded_Character,
) (contains Contains) {
  defer func() { Contains_Invariants(contains, "class_contains.contains") }()
  Class_Ranges_Invariants(ranges, "class_contains.ranges")
  Class_Member_Index_Invariants(index, "class_contains.index")
  Class_Member_Count_Invariants(count, "class_contains.count")
  Node_Class_Negated_Slot_Invariants(negated, "class_contains.negated")
  utf8.Decoded_Character_Invariants(character, "class_contains.character")
  matched := false
  limit := int(index) + int(count)
  for range_index := int(index); range_index < limit; range_index++ {
    range_value := ranges[range_index]
    if !(rune(range_value.Low) > rune(character)) {
      if !(rune(character) > rune(range_value.High)) {
        matched = true
        break
      }
    }
  }
  if uint16(negated) == uint16(CONTROL_TRUE) {
    matched = !matched
  }
  return Contains(matched)
}

func atom_parse(
  nodes Nodes,
  characters Node_Character_Values,
  values Uint16_Values,
  node_count Parsed_Node_Count_Storage,
  ranges Class_Ranges,
  range_count Class_Member_Index_Storage,
  source Nonempty_Pattern_Source,
  parent Container_Node_Reference,
  position Pattern_Index,
) (_ Nonzero_Pattern_Position, status Syntax_Status) {
  defer func() { Syntax_Status_Invariants(status, "atom_parse.status") }()
  Nodes_Invariants(nodes, "atom_parse.nodes")
  Node_Character_Values_Invariants(characters, "atom_parse.characters")
  Uint16_Values_Invariants(values, "atom_parse.values")
  Parsed_Node_Count_Storage_Invariants(node_count, "atom_parse.node_count")
  Class_Ranges_Invariants(ranges, "atom_parse.ranges")
  Class_Member_Index_Storage_Invariants(range_count, "atom_parse.range_count")
  Nonempty_Pattern_Source_Invariants(source, "atom_parse.source")
  Container_Node_Reference_Invariants(parent, "atom_parse.parent")
  Pattern_Index_Invariants(position, "atom_parse.position")
  var end Nonzero_Pattern_Position
  defer func() { Nonzero_Pattern_Position_Invariants(end, "atom_parse.end") }()
  switch source[position] {
  case '[':
    end, status = class_parse(
      nodes,
      characters,
      values,
      node_count,
      ranges,
      range_count,
      source,
      parent,
      position,
    )
  case '*':
    end, status = star_parse(
      nodes, characters, values, node_count, source, parent, position,
    )
  case '?':
    end = Nonzero_Pattern_Position(position + utf8.CHARACTER_SIZE_MINIMUM)
    status = atom_append(
      nodes,
      characters,
      values,
      node_count,
      parent,
      ATOM_SINGLE,
      ATOM_CHARACTER_EMPTY,
    )
  default:
    end, status = literal_parse(
      nodes, characters, values, node_count, source, parent, position,
    )
  }
  return end, status
}

func class_ranges_parse(
  ranges Class_Ranges,
  range_count_state Class_Member_Index_Storage,
  source Nonempty_Pattern_Source,
  start Nonzero_Pattern_Position,
) (_ Nonzero_Pattern_Position, _ Parsed_Range_Count, status Syntax_Status) {
  defer func() { Syntax_Status_Invariants(status, "class_ranges_parse.status") }()
  Class_Ranges_Invariants(ranges, "class_ranges_parse.ranges")
  Class_Member_Index_Storage_Invariants(
    range_count_state, "class_ranges_parse.range_count_state",
  )
  Nonempty_Pattern_Source_Invariants(source, "class_ranges_parse.source")
  Nonzero_Pattern_Position_Invariants(start, "class_ranges_parse.start")
  var end Nonzero_Pattern_Position
  defer func() { Nonzero_Pattern_Position_Invariants(end, "class_ranges_parse.end") }()
  count := Parsed_Range_Count(bytes.SLICE_SIZE_MINIMUM)
  defer func() { Parsed_Range_Count_Invariants(count, "class_ranges_parse.count") }()
  appended := (*Node_Class_Range_Count_Slot)((*Class_Member_Index)(range_count_state))
  cursor := Pattern_Position(start)
  for status == STATUS_OK {
    if cursor == Pattern_Position(len(source)) {
      status = STATUS_SYNTAX_INVALID
      break
    }
    if source[cursor] == ']' {
      if count == bytes.SLICE_SIZE_MINIMUM {
        status = STATUS_SYNTAX_INVALID
        break
      }
      cursor++
      break
    }
    var low utf8.Decoded_Character
    class_end := Class_Character_End(bytes.SLICE_SIZE_MINIMUM)
    low, class_end, status = class_character(
      Class_Pattern_Source(source),
      Class_Character_Index(cursor),
    )
    cursor = Pattern_Position(class_end)
    if status != STATUS_OK {
      break
    }
    high := low
    if cursor+Pattern_Position(utf8.CHARACTER_SIZE_MINIMUM) <
      Pattern_Position(len(source)) {
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
      break
    }
    if high < low {
      status = STATUS_SYNTAX_INVALID
      break
    }
    range_append(ranges, appended, low, high)
    count++
  }
  end = Nonzero_Pattern_Position(cursor)
  return end, count, status
}

// Class_Member_Index names the first range of one validated class instruction.
type Class_Member_Index uint16

// Class_Member_Index_Invariants leaves room for one member inside the validated range arena.
func Class_Member_Index_Invariants(value Class_Member_Index, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value),
    uint16(bytes.SLICE_SIZE_MINIMUM),
    uint16(CLASS_MEMBER_INDEX_MAXIMUM),
  ).
    Ensure()
}

// Class_Member_Count counts the ranges of one validated class instruction.
type Class_Member_Count uint16

// Class_Member_Count_Invariants requires one member and fits the validated range arena.
func Class_Member_Count_Invariants(value Class_Member_Count, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value),
    uint16(utf8.CHARACTER_SIZE_MINIMUM),
    uint16(CLASS_RANGE_COUNT_MAXIMUM),
  ).
    Ensure()
}

func compile_frames(
  nodes Nodes,
  node_count Parsed_Node_Count,
  frames Compile_Frames,
  references Compile_Reference_Values,
  targets Compile_Target_Values,
  controls Compile_Frame_Control_Values,
  kinds Instruction_Kinds,
  characters Instruction_Characters,
  edges Instruction_Target_Values,
  classes Instruction_Class_Values,
  count Compile_Instruction_Count_Storage,
) (start Instruction_PC) {
  defer func() { Instruction_PC_Invariants(start, "compile_frames.start") }()
  Nodes_Invariants(nodes, "compile_frames.nodes")
  Parsed_Node_Count_Invariants(node_count, "compile_frames.node_count")
  Compile_Frames_Invariants(frames, "compile_frames.frames")
  Compile_Reference_Values_Invariants(references, "compile_frames.references")
  Compile_Target_Values_Invariants(targets, "compile_frames.targets")
  Compile_Frame_Control_Values_Invariants(controls, "compile_frames.controls")
  Instruction_Kinds_Invariants(kinds, "compile_frames.kinds")
  Instruction_Characters_Invariants(characters, "compile_frames.characters")
  Instruction_Target_Values_Invariants(edges, "compile_frames.edges")
  Instruction_Class_Values_Invariants(classes, "compile_frames.classes")
  Compile_Instruction_Count_Storage_Invariants(count, "compile_frames.count")
  compile_match(kinds, characters, edges, classes, count)
  start = Instruction_PC(INSTRUCTION_PC_MINIMUM)
  emitted := (*Emitted_Instruction_PC)((*Compile_Instruction_Count_Slot)(count))
  depth := Compile_Depth(COMPILE_DEPTH_NONZERO_MINIMUM)
  for depth != Compile_Depth(PATTERN_DEPTH_MINIMUM) {
    frame := &frames[depth-utf8.CHARACTER_SIZE_MINIMUM]
    aver.Always(
      !(int(*frame.References.Node) > int(node_count)),
      "Compiler frame reference stays inside parsed syntax.",
    )
    switch uint8(*frame.Control.Stage) {
    case COMPILE_STAGE_ENTER:
      start = Instruction_PC(
        compile_enter(
          nodes,
          kinds,
          characters,
          edges,
          classes,
          emitted,
          Child_Node_Reference(*frame.References.Node),
          Instruction_Continuation(*frame.Targets.Continuation),
        ),
      )
      depth--
    case COMPILE_STAGE_SEQUENCE, COMPILE_STAGE_SEQUENCE_RETURN:
      updated, next, stage := compile_sequence_stage(
        nodes,
        frames,
        references,
        targets,
        controls,
        Sequence_Stage(*frame.Control.Stage),
        (*Last_Child_Reference)(
          (*Compile_Child_Slot)(frame.References.Child),
        ),
        frame.Targets.Accumulated,
        Sequence_Compile_Level(depth/Compile_Depth(len("{}"))),
        start,
      )
      depth, start = Compile_Depth(updated), next
      *frame.Control.Stage = Compile_Stage_Slot(stage)
    default:
      updated, next, stage := compile_alternative_stage(
        nodes,
        frames,
        references,
        targets,
        controls,
        kinds,
        characters,
        edges,
        classes,
        (*Alternative_Emit_PC)(emitted),
        Alternative_Stage(*frame.Control.Stage),
        (*Branch_Reference)((*Compile_Child_Slot)(frame.References.Child)),
        frame.Control.First,
        (*Alternative_Updated_PC)(
          (*Compile_Accumulated_Slot)(frame.Targets.Accumulated),
        ),
        (*Alternative_Updated_PC)(
          (*Compile_Continuation_Slot)(frame.Targets.Continuation),
        ),
        Alternative_Compile_Level(depth/Compile_Depth(len("{}"))),
        Alternative_Result_PC(start),
      )
      depth, start = Compile_Depth(updated), Instruction_PC(next)
      *frame.Control.Stage = Compile_Stage_Slot(stage)
    }
  }
  return start
}

// Parsed_Node_Count counts parsed syntax nodes, root included.
type Parsed_Node_Count uint16

// Parsed_Node_Count_Invariants spans the root alone through a full syntax arena.
func Parsed_Node_Count_Invariants(value Parsed_Node_Count, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value),
    uint16(NODE_ROOT),
    uint16(NODE_COUNT_MAXIMUM),
  ).
    Ensure()
}

// Frame_Node_Reference names the syntax node one compiler frame traverses.
type Frame_Node_Reference uint16

// Frame_Node_Reference_Invariants spans the root through the last parsed node.
func Frame_Node_Reference_Invariants(value Frame_Node_Reference, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value),
    uint16(NODE_ROOT),
    uint16(NODE_COUNT_MAXIMUM),
  ).
    Ensure()
}

func compile_sequence_stage(
  nodes Nodes,
  frames Compile_Frames,
  references Compile_Reference_Values,
  targets Compile_Target_Values,
  controls Compile_Frame_Control_Values,
  stage Sequence_Stage,
  child_state Last_Child_Reference_Storage,
  accumulated_state Compile_Accumulated_Storage,
  level Sequence_Compile_Level,
  result Instruction_PC,
) (_ Sequence_Resume_Depth, _ Instruction_PC, next_stage Sequence_Stage) {
  defer func() {
    Sequence_Stage_Invariants(next_stage, "compile_sequence_stage.next_stage")
  }()
  Nodes_Invariants(nodes, "compile_sequence_stage.nodes")
  Compile_Frames_Invariants(frames, "compile_sequence_stage.frames")
  Compile_Reference_Values_Invariants(references, "compile_sequence_stage.references")
  Compile_Target_Values_Invariants(targets, "compile_sequence_stage.targets")
  Compile_Frame_Control_Values_Invariants(controls, "compile_sequence_stage.controls")
  Sequence_Stage_Invariants(stage, "compile_sequence_stage.stage")
  Last_Child_Reference_Storage_Invariants(child_state, "compile_sequence_stage.child_state")
  Compile_Accumulated_Storage_Invariants(
    accumulated_state,
    "compile_sequence_stage.accumulated_state",
  )
  Sequence_Compile_Level_Invariants(level, "compile_sequence_stage.level")
  Instruction_PC_Invariants(result, "compile_sequence_stage.result")
  var depth Sequence_Resume_Depth
  defer func() { Sequence_Resume_Depth_Invariants(depth, "compile_sequence_stage.depth") }()
  var updated_result Instruction_PC
  defer func() {
    Instruction_PC_Invariants(updated_result, "compile_sequence_stage.updated_result")
  }()
  if stage == Sequence_Stage(COMPILE_STAGE_SEQUENCE_RETURN) {
    *accumulated_state = Compile_Accumulated_Slot(result)
    *child_state = Last_Child_Reference(
      compile_sequence_return(
        nodes, Child_Node_Reference(*child_state),
      ),
    )
    depth = Sequence_Resume_Depth(
      level*Sequence_Compile_Level(len("{}")) +
        Sequence_Compile_Level(COMPILE_DEPTH_NONZERO_MINIMUM),
    )
    updated_result = result
    return depth, updated_result, Sequence_Stage(COMPILE_STAGE_SEQUENCE)
  }
  updated, next := compile_sequence(
    nodes,
    frames,
    references,
    targets,
    controls,
    child_state,
    accumulated_state,
    level,
    result,
  )
  depth = Sequence_Resume_Depth(updated * Sequence_Updated_Level(len("{}")))
  updated_result = next
  return depth, updated_result, Sequence_Stage(COMPILE_STAGE_SEQUENCE_RETURN)
}

func compile_alternative_stage(
  nodes Nodes,
  frames Compile_Frames,
  references Compile_Reference_Values,
  targets Compile_Target_Values,
  controls Compile_Frame_Control_Values,
  kinds Instruction_Kinds,
  characters Instruction_Characters,
  instruction_targets Instruction_Target_Values,
  classes Instruction_Class_Values,
  instruction_count Alternative_Emit_PC_Storage,
  stage Alternative_Stage,
  child_state Branch_Reference_Storage,
  first_state Compile_First_Storage,
  accumulated_state Alternative_Updated_PC_Storage,
  continuation_state Alternative_Updated_PC_Storage,
  level Alternative_Compile_Level,
  result Alternative_Result_PC,
) (_ Nonzero_Compile_Depth, _ Alternative_Updated_PC, next_stage Alternative_Stage) {
  defer func() {
    Alternative_Stage_Invariants(next_stage, "compile_alternative_stage.next_stage")
  }()
  Nodes_Invariants(nodes, "compile_alternative_stage.nodes")
  Compile_Frames_Invariants(frames, "compile_alternative_stage.frames")
  Compile_Reference_Values_Invariants(references, "compile_alternative_stage.references")
  Compile_Target_Values_Invariants(targets, "compile_alternative_stage.targets")
  Compile_Frame_Control_Values_Invariants(controls, "compile_alternative_stage.controls")
  Instruction_Kinds_Invariants(kinds, "compile_alternative_stage.kinds")
  Instruction_Characters_Invariants(characters, "compile_alternative_stage.characters")
  Instruction_Target_Values_Invariants(
    instruction_targets,
    "compile_alternative_stage.instruction_targets",
  )
  Instruction_Class_Values_Invariants(classes, "compile_alternative_stage.classes")
  Alternative_Emit_PC_Storage_Invariants(
    instruction_count,
    "compile_alternative_stage.instruction_count",
  )
  Alternative_Stage_Invariants(stage, "compile_alternative_stage.stage")
  Branch_Reference_Storage_Invariants(child_state, "compile_alternative_stage.child_state")
  Compile_First_Storage_Invariants(first_state, "compile_alternative_stage.first_state")
  Alternative_Updated_PC_Storage_Invariants(
    accumulated_state,
    "compile_alternative_stage.accumulated_state",
  )
  Alternative_Updated_PC_Storage_Invariants(
    continuation_state,
    "compile_alternative_stage.continuation_state",
  )
  Alternative_Compile_Level_Invariants(level, "compile_alternative_stage.level")
  Alternative_Result_PC_Invariants(result, "compile_alternative_stage.result")
  var depth Nonzero_Compile_Depth
  defer func() {
    Nonzero_Compile_Depth_Invariants(depth, "compile_alternative_stage.depth")
  }()
  var updated_result Alternative_Updated_PC
  defer func() {
    Alternative_Updated_PC_Invariants(
      updated_result, "compile_alternative_stage.updated_result",
    )
  }()
  if stage == Alternative_Stage(COMPILE_STAGE_ALTERNATIVE_RETURN) {
    previous, accumulated := compile_alternative_return(
      nodes,
      kinds,
      characters,
      instruction_targets,
      classes,
      instruction_count,
      Opened_Sequence_Reference(*child_state),
      first_state,
      Split_Branch_PC(*accumulated_state),
      result,
    )
    *child_state, *accumulated_state = Branch_Reference(previous), accumulated
    depth = Nonzero_Compile_Depth(level * Alternative_Compile_Level(len("{}")))
    updated_result = Alternative_Updated_PC(result)
    return depth, updated_result, Alternative_Stage(COMPILE_STAGE_ALTERNATIVE)
  }
  updated, next := compile_alternative(
    nodes,
    frames,
    references,
    targets,
    controls,
    child_state,
    first_state,
    accumulated_state,
    continuation_state,
    level,
    result,
  )
  depth = Nonzero_Compile_Depth(
    updated*Alternative_Updated_Level(len("{}")) +
      Alternative_Updated_Level(COMPILE_DEPTH_NONZERO_MINIMUM),
  )
  updated_result = next
  return depth, updated_result, Alternative_Stage(COMPILE_STAGE_ALTERNATIVE_RETURN)
}

// Sequence_Stage is one sequence traversal stage.
type Sequence_Stage uint8

// Sequence_Stage_Invariants admits the two sequence traversal stages.
func Sequence_Stage_Invariants(value Sequence_Stage, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_Uint8(
    uint8(value),
    uint8(COMPILE_STAGE_SEQUENCE),
    uint8(COMPILE_STAGE_SEQUENCE_RETURN),
  ).
    Ensure()
}

// Alternative_Stage is one alternative traversal stage.
type Alternative_Stage uint8

// Alternative_Stage_Invariants admits the two alternative traversal stages.
func Alternative_Stage_Invariants(value Alternative_Stage, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_Uint8(
    uint8(value),
    uint8(COMPILE_STAGE_ALTERNATIVE),
    uint8(COMPILE_STAGE_ALTERNATIVE_RETURN),
  ).
    Ensure()
}

// Parsed_Node_Count_Storage points at the syntax counter once the root exists.
type Parsed_Node_Count_Storage *Parsed_Node_Count

// Parsed_Node_Count_Storage_Invariants accepts absent or bounded syntax counter storage.
func Parsed_Node_Count_Storage_Invariants(
  value Parsed_Node_Count_Storage,
  namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Parsed_Node_Count_Invariants(*value, namespace)
}

// Class_Member_Index_Storage points at the range counter where the next class starts.
type Class_Member_Index_Storage *Class_Member_Index

// Class_Member_Index_Storage_Invariants accepts absent or bounded class start storage.
func Class_Member_Index_Storage_Invariants(
  value Class_Member_Index_Storage,
  namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Class_Member_Index_Invariants(*value, namespace)
}

// Parsed_Range_Count counts the ranges one class parse consumed.
type Parsed_Range_Count uint16

// Parsed_Range_Count_Invariants admits a maximum class missing its closing shell.
func Parsed_Range_Count_Invariants(value Parsed_Range_Count, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value),
    uint16(bytes.SLICE_SIZE_MINIMUM),
    uint16(CLASS_RANGE_STORAGE_COUNT_MAXIMUM),
  ).
    Ensure()
}

func pattern_control_store(
  values Uint16_Values,
  start Instruction_PC,
  instruction_count Instruction_Count,
  separator_count Separator_Count,
) (control Pattern_Control) {
  defer func() { Pattern_Control_Invariants(&control, "pattern_control_store.control") }()
  Uint16_Values_Invariants(values, "pattern_control_store.values")
  Instruction_PC_Invariants(start, "pattern_control_store.start")
  Instruction_Count_Invariants(instruction_count, "pattern_control_store.instruction_count")
  Separator_Count_Invariants(separator_count, "pattern_control_store.separator_count")
  start_index := PATTERN_CONTROL_VALUE_START + PATTERN_CONTROL_START
  instruction_count_index := PATTERN_CONTROL_VALUE_START + PATTERN_CONTROL_INSTRUCTION_COUNT
  separator_count_index := PATTERN_CONTROL_VALUE_START + PATTERN_CONTROL_SEPARATOR_COUNT
  values[start_index] = uint16(start)
  values[instruction_count_index] = uint16(instruction_count)
  values[separator_count_index] = uint16(separator_count)
  control = Pattern_Control{
    Start: (*Pattern_Start_Slot)(&values[start_index]),
    Instruction_Count: (*Pattern_Instruction_Count_Slot)(
      &values[instruction_count_index],
    ),
    Separator_Count: (*Pattern_Separator_Count_Slot)(&values[separator_count_index]),
  }
  return control
}

// Last_Child_Reference_Storage points at the unvisited child cursor of one sequence frame.
type Last_Child_Reference_Storage *Last_Child_Reference

// Last_Child_Reference_Storage_Invariants accepts absent or bounded sequence cursor storage.
func Last_Child_Reference_Storage_Invariants(
  value Last_Child_Reference_Storage,
  namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Last_Child_Reference_Invariants(*value, namespace)
}

// Branch_Reference names an unvisited alternative branch, or none.
type Branch_Reference uint16

// Branch_Reference_Invariants excludes the root and first alternative, which no branch reuses.
func Branch_Reference_Invariants(value Branch_Reference, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Holed_Uint16(
    uint16(value),
    uint16(NODE_NONE),
    uint16(NODE_COUNT_MAXIMUM),
    uint16(NODE_ROOT),
    uint16(NODE_FIRST_ALTERNATIVE),
    uint16(NODE_FIRST_ALTERNATIVE),
  ).
    Ensure()
}

// Branch_Reference_Storage points at the unvisited branch cursor of one alternative frame.
type Branch_Reference_Storage *Branch_Reference

// Branch_Reference_Storage_Invariants accepts absent or bounded alternative cursor storage.
func Branch_Reference_Storage_Invariants(
  value Branch_Reference_Storage, namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Branch_Reference_Invariants(*value, namespace)
}

// Previous_Branch_Reference names the branch before one visited alternative branch, or none.
type Previous_Branch_Reference uint16

// Previous_Branch_Reference_Invariants excludes the root and first alternative before the last.
func Previous_Branch_Reference_Invariants(
  value Previous_Branch_Reference,
  namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Holed_Uint16(
    uint16(value),
    uint16(NODE_NONE),
    uint16(CHILD_PARENT_REFERENCE_MAXIMUM),
    uint16(NODE_ROOT),
    uint16(NODE_FIRST_ALTERNATIVE),
    uint16(NODE_FIRST_ALTERNATIVE),
  ).
    Ensure()
}

// Alternative_Updated_PC_Storage points at one alternative frame target.
type Alternative_Updated_PC_Storage *Alternative_Updated_PC

// Alternative_Updated_PC_Storage_Invariants accepts absent or bounded alternative target storage.
func Alternative_Updated_PC_Storage_Invariants(
  value Alternative_Updated_PC_Storage,
  namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Alternative_Updated_PC_Invariants(*value, namespace)
}

// Instruction_PC_Storage points at the next emission position.
type Instruction_PC_Storage *Instruction_PC

// Instruction_PC_Storage_Invariants accepts absent or bounded emission cursor storage.
func Instruction_PC_Storage_Invariants(
  value Instruction_PC_Storage, namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Instruction_PC_Invariants(*value, namespace)
}

// Emitted_Instruction_PC_Storage points at the next emission position after the terminal match.
type Emitted_Instruction_PC_Storage *Emitted_Instruction_PC

// Emitted_Instruction_PC_Storage_Invariants accepts absent or bounded emission cursor storage.
func Emitted_Instruction_PC_Storage_Invariants(
  value Emitted_Instruction_PC_Storage,
  namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Emitted_Instruction_PC_Invariants(*value, namespace)
}

// Pattern_Unvalidated is one caller-held compiled pattern before header validation.
type Pattern_Unvalidated struct {
  // Workspace retains compiled instructions and separators.
  Workspace Compile_Workspace_Pointer
  // Control retains caller header storage.
  Control Pattern_Control_Unvalidated
}

// Pattern_Unvalidated_Invariants composes caller storage whose header is not yet validated.
func Pattern_Unvalidated_Invariants(value *Pattern_Unvalidated, namespace aver.Namespace) {
  Compile_Workspace_Pointer_Invariants(value.Workspace, namespace)
  Pattern_Control_Unvalidated_Invariants(&value.Control, namespace)
}

// Pattern_Control_Unvalidated is one caller-held pattern header before validation.
type Pattern_Control_Unvalidated struct {
  // Start retains entry storage.
  Start Pattern_Start_Unvalidated_Storage
  // Instruction_Count retains populated instruction count storage.
  Instruction_Count Pattern_Instruction_Count_Unvalidated_Storage
  // Separator_Count retains populated separator count storage.
  Separator_Count Pattern_Separator_Count_Unvalidated_Storage
}

// Pattern_Control_Unvalidated_Invariants composes caller header storage before validation.
func Pattern_Control_Unvalidated_Invariants(
  value *Pattern_Control_Unvalidated,
  namespace aver.Namespace,
) {
  Pattern_Start_Unvalidated_Storage_Invariants(value.Start, namespace)
  Pattern_Instruction_Count_Unvalidated_Storage_Invariants(value.Instruction_Count, namespace)
  Pattern_Separator_Count_Unvalidated_Storage_Invariants(value.Separator_Count, namespace)
}

// Pattern_Start_Unvalidated is one caller-held header value before validation.
type Pattern_Start_Unvalidated uint16

// Pattern_Start_Unvalidated_Invariants admits any stored entry before validation.
func Pattern_Start_Unvalidated_Invariants(
  value Pattern_Start_Unvalidated,
  namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value),
    bits.WORD_16_MINIMUM,
    bits.WORD_16_MAXIMUM,
  ).
    Ensure()
}

// Pattern_Start_Unvalidated_Storage points at one caller-held header value.
type Pattern_Start_Unvalidated_Storage *Pattern_Start_Unvalidated

// Pattern_Start_Unvalidated_Storage_Invariants accepts absent or any caller entry storage.
func Pattern_Start_Unvalidated_Storage_Invariants(
  value Pattern_Start_Unvalidated_Storage,
  namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Pattern_Start_Unvalidated_Invariants(*value, namespace)
}

// Pattern_Instruction_Count_Unvalidated is one caller-held header value before validation.
type Pattern_Instruction_Count_Unvalidated uint16

// Pattern_Instruction_Count_Unvalidated_Invariants admits any stored count before validation.
func Pattern_Instruction_Count_Unvalidated_Invariants(
  value Pattern_Instruction_Count_Unvalidated,
  namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value),
    bits.WORD_16_MINIMUM,
    bits.WORD_16_MAXIMUM,
  ).
    Ensure()
}

// Pattern_Instruction_Count_Unvalidated_Storage points at one caller-held header value.
type Pattern_Instruction_Count_Unvalidated_Storage *Pattern_Instruction_Count_Unvalidated

// Pattern_Instruction_Count_Unvalidated_Storage_Invariants accepts absent or any count storage.
func Pattern_Instruction_Count_Unvalidated_Storage_Invariants(
  value Pattern_Instruction_Count_Unvalidated_Storage,
  namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Pattern_Instruction_Count_Unvalidated_Invariants(*value, namespace)
}

// Pattern_Separator_Count_Unvalidated is one caller-held header value before validation.
type Pattern_Separator_Count_Unvalidated uint16

// Pattern_Separator_Count_Unvalidated_Invariants admits any stored count before validation.
func Pattern_Separator_Count_Unvalidated_Invariants(
  value Pattern_Separator_Count_Unvalidated,
  namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value),
    bits.WORD_16_MINIMUM,
    bits.WORD_16_MAXIMUM,
  ).
    Ensure()
}

// Pattern_Separator_Count_Unvalidated_Storage points at one caller-held header value.
type Pattern_Separator_Count_Unvalidated_Storage *Pattern_Separator_Count_Unvalidated

// Pattern_Separator_Count_Unvalidated_Storage_Invariants accepts absent or any count storage.
func Pattern_Separator_Count_Unvalidated_Storage_Invariants(
  value Pattern_Separator_Count_Unvalidated_Storage,
  namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Pattern_Separator_Count_Unvalidated_Invariants(*value, namespace)
}

func compile_match(
  kinds Instruction_Kinds,
  characters Instruction_Characters,
  targets Instruction_Target_Values,
  classes Instruction_Class_Values,
  instruction_count Compile_Instruction_Count_Storage,
) {
  Instruction_Kinds_Invariants(kinds, "compile_match.kinds")
  Instruction_Characters_Invariants(characters, "compile_match.characters")
  Instruction_Target_Values_Invariants(targets, "compile_match.targets")
  Instruction_Class_Values_Invariants(classes, "compile_match.classes")
  Compile_Instruction_Count_Storage_Invariants(
    instruction_count, "compile_match.instruction_count",
  )
  *instruction_count = Compile_Instruction_Count_Slot(bytes.SLICE_SIZE_MINIMUM)
  match := instruction_emit(
    kinds,
    characters,
    targets,
    classes,
    (*Instruction_PC)((*Compile_Instruction_Count_Slot)(instruction_count)),
    INSTRUCTION_MATCH,
    Node_Character_Slot(utf8.DECODED_CHARACTER_MINIMUM),
    Instruction_Continuation(INSTRUCTION_PC_MINIMUM),
    Split_Branch_PC(INSTRUCTION_PC_MINIMUM),
    Node_Class_Range_Index_Slot(bytes.SLICE_SIZE_MINIMUM),
    Node_Class_Range_Count_Slot(bytes.SLICE_SIZE_MINIMUM),
    Node_Class_Negated_Slot(CONTROL_FALSE),
  )
  aver.Always(
    match == Instruction_PC(INSTRUCTION_PC_MINIMUM),
    "Terminal match occupies the first instruction.",
  )
}

// Alternative_Emit_PC is the next emission position while one alternative compiles.
type Alternative_Emit_PC uint16

// Alternative_Emit_PC_Invariants spans the first emission after the terminal match through the
// last position an alternative can reach before its shell.
func Alternative_Emit_PC_Invariants(value Alternative_Emit_PC, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value),
    uint16(EMITTED_INSTRUCTION_PC_MINIMUM),
    uint16(INSTRUCTION_CONTINUATION_MAXIMUM),
  ).
    Ensure()
}

// Alternative_Emit_PC_Storage points at the emission position while one alternative compiles.
type Alternative_Emit_PC_Storage *Alternative_Emit_PC

// Alternative_Emit_PC_Storage_Invariants accepts absent or bounded alternative emission storage.
func Alternative_Emit_PC_Storage_Invariants(
  value Alternative_Emit_PC_Storage,
  namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Alternative_Emit_PC_Invariants(*value, namespace)
}

// Grouped_Node_Count counts parsed syntax nodes once one group is open.
type Grouped_Node_Count uint16

// Grouped_Node_Count_Invariants starts at the root, one alternative, and one open sequence.
func Grouped_Node_Count_Invariants(value Grouped_Node_Count, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value),
    uint16(NESTED_SEQUENCE_REFERENCE_MINIMUM),
    uint16(NODE_COUNT_MAXIMUM),
  ).
    Ensure()
}

// Grouped_Node_Count_Storage points at the syntax counter once one group is open.
type Grouped_Node_Count_Storage *Grouped_Node_Count

// Grouped_Node_Count_Storage_Invariants accepts absent or bounded grouped counter storage.
func Grouped_Node_Count_Storage_Invariants(
  value Grouped_Node_Count_Storage,
  namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Grouped_Node_Count_Invariants(*value, namespace)
}

// Sequence_Resume_Depth is the compiler depth after one sequence stage.
type Sequence_Resume_Depth int

// Sequence_Resume_Depth_Invariants stops one frame short of the syntax arena, because the
// deepest sequence frame never pushes a child.
func Sequence_Resume_Depth_Invariants(value Sequence_Resume_Depth, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(
    int(value),
    PATTERN_DEPTH_MINIMUM,
    COMPILE_SEQUENCE_DEPTH_MAXIMUM,
  ).
    Ensure()
}

func syntax_step(
  nodes Nodes,
  characters Node_Character_Values,
  values Uint16_Values,
  node_count Parsed_Node_Count_Storage,
  frames Parser_Frames,
  ranges Class_Ranges,
  range_count Class_Member_Index_Storage,
  source Nonempty_Pattern_Source,
  current Container_Node_Reference,
  depth Parser_Depth,
  position Pattern_Index,
) (
  _ Container_Node_Reference, _ Parser_Depth, _ Pattern_Position, status Syntax_Status,
) {
  defer func() { Syntax_Status_Invariants(status, "syntax_step.status") }()
  Nodes_Invariants(nodes, "syntax_step.nodes")
  Node_Character_Values_Invariants(characters, "syntax_step.characters")
  Uint16_Values_Invariants(values, "syntax_step.values")
  Parsed_Node_Count_Storage_Invariants(node_count, "syntax_step.node_count")
  Parser_Frames_Invariants(frames, "syntax_step.frames")
  Class_Ranges_Invariants(ranges, "syntax_step.ranges")
  Class_Member_Index_Storage_Invariants(range_count, "syntax_step.range_count")
  Nonempty_Pattern_Source_Invariants(source, "syntax_step.source")
  Container_Node_Reference_Invariants(current, "syntax_step.current")
  Parser_Depth_Invariants(depth, "syntax_step.depth")
  Pattern_Index_Invariants(position, "syntax_step.position")
  next := current
  defer func() { Container_Node_Reference_Invariants(next, "syntax_step.next") }()
  next_depth := depth
  defer func() { Parser_Depth_Invariants(next_depth, "syntax_step.next_depth") }()
  next_position := Pattern_Position(position) + Pattern_Position(utf8.CHARACTER_SIZE_MINIMUM)
  defer func() { Pattern_Position_Invariants(next_position, "syntax_step.next_position") }()
  switch source[position] {
  case '{':
    opened, opened_depth, state := group_open(
      nodes,
      characters,
      values,
      node_count,
      frames,
      current,
      depth,
    )
    next, next_depth = Container_Node_Reference(opened), Parser_Depth(opened_depth)
    status = state
    return next, next_depth, next_position, status
  case '}':
    if depth == PATTERN_DEPTH_MINIMUM {
      next_position = Pattern_Position(position)
      return next, next_depth, next_position, STATUS_SYNTAX_INVALID
    }
    parent, closed_depth := group_close(frames, Open_Parser_Depth(depth))
    next, next_depth = Container_Node_Reference(parent), Parser_Depth(closed_depth)
    return next, next_depth, next_position, STATUS_OK
  case ',':
    if depth != PATTERN_DEPTH_MINIMUM {
      branch, state := group_branch(
        nodes,
        characters,
        values,
        (*Grouped_Node_Count)((*Parsed_Node_Count)(node_count)),
        frames,
        Nested_Sequence_Reference(current),
        Open_Parser_Depth(depth),
      )
      next, status = Container_Node_Reference(branch), state
      return next, next_depth, next_position, status
    }
  }
  end, status := atom_parse(
    nodes,
    characters,
    values,
    node_count,
    ranges,
    range_count,
    source,
    current,
    position,
  )
  next_position = Pattern_Position(end)
  return next, next_depth, next_position, status
}

// Split_Branch_PC is the alternative target one split branches to.
type Split_Branch_PC uint16

// Split_Branch_PC_Invariants reserves the brace shell and one comma.
func Split_Branch_PC_Invariants(value Split_Branch_PC, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value),
    uint16(INSTRUCTION_PC_MINIMUM),
    uint16(SPLIT_BRANCH_PC_MAXIMUM),
  ).
    Ensure()
}
