// Package template keeps every lexer role one parse step reads through.
package template

import "local/james-orcales/shared/bytes"
import "local/james-orcales/shared/math/bits"
import "local/james-orcales/shared/sim/aver/default"
import "local/james-orcales/shared/unicode/utf8"

// Quoted_Eof_Token is the empty token one lexer holds while a literal remains.
type Quoted_Eof_Token Eof_Token

// Quoted_Eof_Token_Invariants reserves the closing quote the literal still takes.
func Quoted_Eof_Token_Invariants(value Quoted_Eof_Token, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Start),
    uint16(TOKEN_BOUNDARY_MINIMUM),
    uint16(QUOTED_LEXEME_POSITION_MAXIMUM),
  ).
    Range_Uint16(
    uint16(value.End),
    uint16(TOKEN_BOUNDARY_MINIMUM),
    uint16(QUOTED_LEXEME_POSITION_MAXIMUM),
  ).
    Ensure()
  aver.Always(
    uint8(value.Kind) == uint8(TOKEN_EOF),
    "One quoted cursor holds the empty token its caller read.",
  )
}

// Quoted_Eof_Token_Storage is that empty token inside its cursor field.
type Quoted_Eof_Token_Storage Token_Storage

// Quoted_Eof_Token_Storage_Invariants composes the empty token.
func Quoted_Eof_Token_Storage_Invariants(
  value Quoted_Eof_Token_Storage, namespace aver.Namespace,
) {
  Quoted_Eof_Token_Invariants(Quoted_Eof_Token(value.Value), namespace)
}

// Else_Clause_Token is the alternate keyword one branch body already stands past.
type Else_Clause_Token Else_Token

// Else_Clause_Token_Invariants keeps the opening branch beneath the keyword.
func Else_Clause_Token_Invariants(value Else_Clause_Token, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Start),
    uint16(ELSE_CLAUSE_START_MINIMUM),
    uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM-len(KEYWORD_ELSE)),
  ).
    Range_Uint16(
    uint16(value.End),
    uint16(ELSE_CLAUSE_END_MINIMUM),
    uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM),
  ).
    Ensure()
  aver.Always(
    uint8(value.Kind) == uint8(TOKEN_ELSE),
    "One alternate cursor retains the keyword its caller read.",
  )
}

// Else_Clause_Token_Storage is that keyword inside its cursor field.
type Else_Clause_Token_Storage Token_Storage

// Else_Clause_Token_Storage_Invariants composes the retained keyword.
func Else_Clause_Token_Storage_Invariants(
  value Else_Clause_Token_Storage,
  namespace aver.Namespace,
) {
  Else_Clause_Token_Invariants(Else_Clause_Token(value.Value), namespace)
}

// Else_Clause_Position is where one lexer stands once it read the alternate keyword.
type Else_Clause_Position Else_Position_Stored

// Else_Clause_Position_Invariants keeps the keyword beneath the position.
func Else_Clause_Position_Invariants(value Else_Clause_Position, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(ELSE_CLAUSE_END_MINIMUM),
    uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM),
  ).
    Ensure()
}

// Else_Clause_End is where one lexer stops once it read the alternate keyword.
type Else_Clause_End Else_End_Stored

// Else_Clause_End_Invariants keeps the keyword beneath the boundary.
func Else_Clause_End_Invariants(value Else_Clause_End, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(ELSE_CLAUSE_END_MINIMUM),
    uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM),
  ).
    Ensure()
}

// Else_Clause_Cursor is one lexer that read the alternate keyword of a branch.
type Else_Clause_Cursor Token_Cursor

// Else_Clause_Cursor_Invariants composes lexer state beside the retained keyword.
func Else_Clause_Cursor_Invariants(value Else_Clause_Cursor, namespace aver.Namespace) {
  Else_Clause_Position_Invariants(Else_Clause_Position(value.Position), namespace)
  Else_Clause_End_Invariants(Else_Clause_End(value.End), namespace)
  Else_Clause_Token_Storage_Invariants(Else_Clause_Token_Storage(value.Token), namespace)
  Peeked_Look_Ahead_Storage_Invariants(Peeked_Look_Ahead_Storage(value.Looked), namespace)
  Cursor_Number_Stored_Invariants((*Cursor_Number_Stored)(&value.Number), namespace)
  Function_Function_Storage_Invariants(&value.Function, namespace)
  Else_Last_Start_Invariants(Else_Last_Start(value.Last_Start), namespace)
  Else_Has_Last_Invariants(Else_Has_Last(value.Has_Last), namespace)
}

// Else_Clause_Cursor_Pointer keeps mutation of that lexer typed.
type Else_Clause_Cursor_Pointer *Else_Clause_Cursor

// Else_Clause_Cursor_Pointer_Invariants composes present lexer storage.
func Else_Clause_Cursor_Pointer_Invariants(
  value Else_Clause_Cursor_Pointer,
  namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Else_Clause_Cursor_Invariants(*value, namespace)
}

// PIPE_SPAN_SIZE_MINIMUM spans one term, the pipe, and the term that follows it.
const PIPE_SPAN_SIZE_MINIMUM = IDENTIFIER_SIZE_MINIMUM +
  ACTION_SEPARATOR_SIZE_MINIMUM +
  IDENTIFIER_SIZE_MINIMUM

// COMPARE_NAME_SIZE_MINIMUM is the shortest name one comparison builtin answers.
const COMPARE_NAME_SIZE_MINIMUM = len("eq")

// COMPARE_NAME_SIZE_MAXIMUM is the longest name one comparison builtin answers.
const COMPARE_NAME_SIZE_MAXIMUM = len("not")

// Compare_Name is the builtin name one comparison answers to.
type Compare_Name Function_Name_Validated

// Compare_Name_Invariants keeps every other builtin out of that comparison.
func Compare_Name_Invariants(value Compare_Name, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_Int(
    len(value.Value),
    COMPARE_NAME_SIZE_MINIMUM,
    COMPARE_NAME_SIZE_MAXIMUM,
  ).
    Ensure()
}

// Compare_Status is the outcome one comparison builtin reports.
type Compare_Status Call_Status

// Compare_Status_Invariants keeps the output limit out of that comparison.
func Compare_Status_Invariants(value Compare_Status, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_3_Uint8(
    uint8(value),
    uint8(STATUS_OK),
    uint8(STATUS_FUNCTION_INVALID),
    uint8(STATUS_EXECUTION_INVALID),
  ).
    Ensure()
}

// Delivered_State_Value is the value one frame holds while a result arrives.
type Delivered_State_Value Evaluation_State_Value

// Delivered_State_Value_Invariants keeps that byte inside the destination.
func Delivered_State_Value_Invariants(
  value Delivered_State_Value, namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint8(
    uint8(value.Kind),
    uint8(VALUE_NIL),
    uint8(VALUE_FUNCTION),
  ).
    Range_Uint64(
    uint64(value.Real),
    bits.WORD_64_MINIMUM,
    bits.WORD_64_MAXIMUM,
  ).
    Range_Uint64(
    uint64(value.Imaginary),
    bits.WORD_64_MINIMUM,
    bits.WORD_64_MAXIMUM,
  ).
    Range_Int(
    len(value.Text),
    OUTPUT_SIZE_MINIMUM,
    OUTPUT_SIZE_MAXIMUM,
  ).
    Range_Int(
    len(value.Values),
    bytes.SLICE_SIZE_MINIMUM,
    VALUE_COUNT_MAXIMUM,
  ).
    Range_Int(
    len(value.Fields),
    bytes.SLICE_SIZE_MINIMUM,
    VALUE_COUNT_MAXIMUM,
  ).
    Sometimes(
    value.Function != nil,
    "Delivered state holds a callable.",
  ).
    Ensure()
}

// DELIVERED_DECLARATION_TAIL_SIZE is the shortest declaration whose value one child answers.
const DELIVERED_DECLARATION_TAIL_SIZE = FIRST_DECLARATION_ACTION_SIZE_MINIMUM +
  PARENTHESIS_PAIR_SIZE

// DELIVERED_DECLARATION_PACKED_COUNT packs the densest actions and one text node before that
// declaration.
const DELIVERED_DECLARATION_PACKED_COUNT = ROOT_NODE_COUNT +
  (SOURCE_SIZE_MAXIMUM-DELIVERED_DECLARATION_TAIL_SIZE)/
    SIMPLE_ACTION_SIZE_MINIMUM*
    SIMPLE_ACTION_NODE_COUNT +
  ((SOURCE_SIZE_MAXIMUM-DELIVERED_DECLARATION_TAIL_SIZE)%SIMPLE_ACTION_SIZE_MINIMUM+
    SIMPLE_ACTION_SIZE_MINIMUM-
    SOURCE_POSITION_INCREMENT)/
    SIMPLE_ACTION_SIZE_MINIMUM

// DELIVERED_FIRST_DECLARATION_MAXIMUM is that declaration's name, past its action and pipeline.
const DELIVERED_FIRST_DECLARATION_MAXIMUM = DELIVERED_DECLARATION_PACKED_COUNT +
  ACTION_PIPELINE_NODE_COUNT +
  NODE_COUNT_INCREMENT

// DELIVERED_RANGE_DECLARATION_TAIL_SIZE is the shortest range, with its end, whose two names one
// child answers.
const DELIVERED_RANGE_DECLARATION_TAIL_SIZE = SECOND_DECLARATION_ACTION_SIZE_MINIMUM +
  PARENTHESIS_PAIR_SIZE

// DELIVERED_SECOND_DECLARATION_MAXIMUM is that range's second name behind the densest actions,
// one text node, the range node, its pipeline, and its first name.
const DELIVERED_SECOND_DECLARATION_MAXIMUM = ROOT_NODE_COUNT +
  (SOURCE_SIZE_MAXIMUM-DELIVERED_RANGE_DECLARATION_TAIL_SIZE)/
    SIMPLE_ACTION_SIZE_MINIMUM*
    SIMPLE_ACTION_NODE_COUNT +
  ((SOURCE_SIZE_MAXIMUM-DELIVERED_RANGE_DECLARATION_TAIL_SIZE)%SIMPLE_ACTION_SIZE_MINIMUM+
    SIMPLE_ACTION_SIZE_MINIMUM-
    SOURCE_POSITION_INCREMENT)/
    SIMPLE_ACTION_SIZE_MINIMUM +
  ACTION_PIPELINE_NODE_COUNT +
  NODE_COUNT_INCREMENT +
  NODE_COUNT_INCREMENT

// Delivered_First_Declaration is the first name one frame holds while a child answers.
type Delivered_First_Declaration Declaration_First_Reference_Storage

// Delivered_First_Declaration_Invariants leaves the child's parentheses behind that name.
func Delivered_First_Declaration_Invariants(
  value Delivered_First_Declaration,
  namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Holed_Uint16(
    uint16(value.Value),
    uint16(NODE_REFERENCE_MINIMUM),
    uint16(DELIVERED_FIRST_DECLARATION_MAXIMUM),
    uint16(ROOT_LIST_HOLE),
    uint16(TERM_ACTION_HOLE),
    uint16(TERM_PIPELINE_HOLE),
  ).
    Ensure()
}

// Delivered_Second_Declaration is the second name one frame holds while a child answers.
type Delivered_Second_Declaration Declaration_Second_Reference_Storage

// Delivered_Second_Declaration_Invariants leaves the child's parentheses behind that name.
func Delivered_Second_Declaration_Invariants(
  value Delivered_Second_Declaration,
  namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Holed_Uint16(
    uint16(value.Value),
    uint16(NODE_REFERENCE_MINIMUM),
    uint16(DELIVERED_SECOND_DECLARATION_MAXIMUM),
    uint16(ROOT_LIST_HOLE),
    uint16(TERM_ACTION_HOLE),
    uint16(TERM_PIPELINE_HOLE),
  ).
    Ensure()
}

// Delivered_Declaration_References are the names one frame holds while a child answers.
type Delivered_Declaration_References Declaration_References

// Delivered_Declaration_References_Invariants composes both names.
func Delivered_Declaration_References_Invariants(
  value Delivered_Declaration_References,
  namespace aver.Namespace,
) {
  Delivered_First_Declaration_Invariants(
    Delivered_First_Declaration(value.First), namespace,
  )
  Delivered_Second_Declaration_Invariants(
    Delivered_Second_Declaration(value.Second), namespace,
  )
}

// Delivered_Pipeline_State is the pipeline state one frame holds while a result arrives.
type Delivered_Pipeline_State Pipeline_State_Stored

// Delivered_Pipeline_State_Invariants composes that state's value and declarations.
func Delivered_Pipeline_State_Invariants(
  value Delivered_Pipeline_State, namespace aver.Namespace,
) {
  Delivered_State_Value_Invariants(Delivered_State_Value(value.Value), namespace)
  Declaration_References_Stored_Invariants(
    Declaration_References_Stored(value.Declarations),
    namespace,
  )
  aver.Tree(value, namespace).
    Enum_3_Uint8(
    uint8(value.Declaration_Count),
    uint8(DECLARATION_COUNT_NONE),
    uint8(DECLARATION_COUNT_ONE),
    uint8(DECLARATION_COUNT_TWO),
  ).
    Sometimes(
    bool(value.Assignment),
    "Delivered pipeline updates a variable.",
  ).
    Ensure()
}

// Delivered_Evaluation_States is the state one frame holds while a result arrives.
type Delivered_Evaluation_States Evaluation_States

// Delivered_Evaluation_States_Invariants composes that state's storage.
func Delivered_Evaluation_States_Invariants(
  value Delivered_Evaluation_States,
  namespace aver.Namespace,
) {
  Delivered_Pipeline_State_Invariants(
    Delivered_Pipeline_State(value.Value), namespace,
  )
}

// Child_Delivered_Pipeline_State is the pipeline state one frame holds while its child answers.
type Child_Delivered_Pipeline_State Pipeline_State_Stored

// Child_Delivered_Pipeline_State_Invariants composes that state's value and declarations.
func Child_Delivered_Pipeline_State_Invariants(
  value Child_Delivered_Pipeline_State,
  namespace aver.Namespace,
) {
  Delivered_State_Value_Invariants(Delivered_State_Value(value.Value), namespace)
  Delivered_Declaration_References_Invariants(
    Delivered_Declaration_References(value.Declarations),
    namespace,
  )
  aver.Tree(value, namespace).
    Enum_3_Uint8(
    uint8(value.Declaration_Count),
    uint8(DECLARATION_COUNT_NONE),
    uint8(DECLARATION_COUNT_ONE),
    uint8(DECLARATION_COUNT_TWO),
  ).
    Sometimes(
    bool(value.Assignment),
    "A child answers a variable update.",
  ).
    Ensure()
}

// Child_Delivered_Evaluation_States is the state one frame holds while its child answers.
type Child_Delivered_Evaluation_States Evaluation_States

// Child_Delivered_Evaluation_States_Invariants composes that state's storage.
func Child_Delivered_Evaluation_States_Invariants(
  value Child_Delivered_Evaluation_States,
  namespace aver.Namespace,
) {
  Child_Delivered_Pipeline_State_Invariants(
    Child_Delivered_Pipeline_State(value.Value), namespace,
  )
}

// SCHEDULED_FRAME_COUNT_MINIMUM counts the frame one scheduled range stands inside.
const SCHEDULED_FRAME_COUNT_MINIMUM = FRAME_COUNT_INCREMENT + FRAME_COUNT_INCREMENT

// Scheduled_Frame_Count is the frame cursor one scheduled range reads.
type Scheduled_Frame_Count Bound_Frame_Count

// Scheduled_Frame_Count_Invariants keeps that frame on the stack.
func Scheduled_Frame_Count_Invariants(
  value Scheduled_Frame_Count, namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(SCHEDULED_FRAME_COUNT_MINIMUM),
    uint16(FRAME_COUNT_MAXIMUM-FRAME_COUNT_INCREMENT),
  ).
    Ensure()
}

// Scheduled_Frame_Count_Pointer keeps that cursor mutable for its caller.
type Scheduled_Frame_Count_Pointer *Scheduled_Frame_Count

// Scheduled_Frame_Count_Pointer_Invariants composes one present cursor.
func Scheduled_Frame_Count_Pointer_Invariants(
  value Scheduled_Frame_Count_Pointer,
  namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Scheduled_Frame_Count_Invariants(*value, namespace)
}

// MARKED_NUMBER_START_MAXIMUM leaves the exponent digit one marker stands before.
const MARKED_NUMBER_START_MAXIMUM = NUMBER_START_MAXIMUM - utf8.CHARACTER_SIZE_MINIMUM

// Marked_Number_Span is the span one exponent marker reads inside.
type Marked_Number_Span Number_Span_Validated

// Marked_Number_Span_Invariants composes that span's boundaries.
func Marked_Number_Span_Invariants(value Marked_Number_Span, namespace aver.Namespace) {
  Marked_Number_Boundaries_Invariants(
    Marked_Number_Boundaries(value.Value), namespace,
  )
}

// Marked_Number_Boundaries are the boundaries one exponent marker reads inside.
type Marked_Number_Boundaries Number_Span_Proof_Stored

// Marked_Number_Boundaries_Invariants keeps that digit inside the numeral.
func Marked_Number_Boundaries_Invariants(
  value Marked_Number_Boundaries, namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Start),
    uint16(NUMBER_START_MINIMUM),
    uint16(MARKED_NUMBER_START_MAXIMUM),
  ).
    Range_Uint16(
    uint16(value.End),
    uint16(NUMBER_END_MINIMUM),
    uint16(NUMBER_END_MAXIMUM),
  ).
    Ensure()
}

// CONTROL_ACTION_TAIL_NODE_COUNT counts the branches a clause packs above its action.
const CONTROL_ACTION_TAIL_NODE_COUNT = CONTROL_BRANCH_NODE_COUNT +
  CONTROL_BRANCH_NODE_COUNT

// CLAUSE_TAIL_NODE_COUNT counts three branches and one container above one clause.
const CLAUSE_TAIL_NODE_COUNT = CONTROL_ACTION_TAIL_NODE_COUNT +
  CONTROL_BRANCH_NODE_COUNT +
  SIMPLE_ACTION_CONTAINER_NODE_COUNT

// BLOCK_TAIL_NODE_COUNT counts four branches above one block.
const BLOCK_TAIL_NODE_COUNT = CONTROL_ACTION_TAIL_NODE_COUNT +
  CONTROL_ACTION_TAIL_NODE_COUNT

// TOKENIZED_LAST_START_MAXIMUM leaves the pipe and its term above the last token.
const TOKENIZED_LAST_START_MAXIMUM = PIPELINE_TERM_LAST_START_MAXIMUM -
  (PIPELINE_SEPARATOR_SIZE +
    DOT_SOURCE_SIZE)

// Token_Loop_Cursor is one lexer a pipeline token loop walks.
type Token_Loop_Cursor Pipeline_Term_Cursor

// Token_Loop_Cursor_Invariants composes that lexer's storage.
func Token_Loop_Cursor_Invariants(
  value Token_Loop_Cursor, namespace aver.Namespace,
) {
  Pipeline_Term_Position_Stored_Invariants(
    Pipeline_Term_Position_Stored(value.Position), namespace,
  )
  Pipeline_Term_End_Stored_Invariants(Pipeline_Term_End_Stored(value.End), namespace)
  Pipeline_Term_Token_Storage_Invariants(Pipeline_Term_Token_Storage(value.Token), namespace)
  Peeked_Look_Ahead_Storage_Invariants(Peeked_Look_Ahead_Storage(value.Looked), namespace)
  Cursor_Number_Stored_Invariants((*Cursor_Number_Stored)(&value.Number), namespace)
  Function_Function_Storage_Invariants(&value.Function, namespace)
  Tokenized_Last_Start_Invariants(Tokenized_Last_Start(value.Last_Start), namespace)
  Cursor_Has_Last_Stored_Invariants((*Cursor_Has_Last_Stored)(&value.Has_Last), namespace)
}

// Tokenized_Last_Start is where one pipeline token loop's last token began.
type Tokenized_Last_Start Cursor_Last_Start_Stored

// Tokenized_Last_Start_Invariants keeps the pipe and its term inside the action.
func Tokenized_Last_Start_Invariants(value Tokenized_Last_Start, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(SOURCE_SIZE_MINIMUM),
    uint16(TOKENIZED_LAST_START_MAXIMUM),
  ).
    Ensure()
}

// Token_Loop_Cursor_Pointer keeps mutation of that lexer typed.
type Token_Loop_Cursor_Pointer *Token_Loop_Cursor

// Token_Loop_Cursor_Pointer_Invariants composes present lexer state.
func Token_Loop_Cursor_Pointer_Invariants(
  value Token_Loop_Cursor_Pointer,
  namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Token_Loop_Cursor_Invariants(*value, namespace)
}

// DECLARED_LAST_START_MAXIMUM leaves one bare name and its operator past the parenthesis before it.
const DECLARED_LAST_START_MAXIMUM = ACTION_CONTENT_BOUNDARY_MAXIMUM -
  PARENTHESIS_SIZE -
  VARIABLE_NAME_SIZE_MINIMUM -
  ASSIGNMENT_OPERATOR_SIZE

// Declared_Last_Start is where one declared variable's last token began.
type Declared_Last_Start Cursor_Last_Start_Stored

// Declared_Last_Start_Invariants keeps those actions inside the source.
func Declared_Last_Start_Invariants(value Declared_Last_Start, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(SOURCE_SIZE_MINIMUM),
    uint16(DECLARED_LAST_START_MAXIMUM),
  ).
    Ensure()
}

// KEYWORD_CLAUSE_TAIL_SIZE is the clause and the end action one keyword still owes.
const KEYWORD_CLAUSE_TAIL_SIZE = CONTROL_ACTION_SIZE_MINIMUM +
  len(KEYWORD_END) +
  ACTION_BOUNDARY_SIZE_MINIMUM

// KEYWORD_LAST_START_SECOND stands inside the opening action every later keyword follows.
const KEYWORD_LAST_START_SECOND = SOURCE_SIZE_MINIMUM + utf8.CHARACTER_SIZE_MINIMUM

// KEYWORD_LAST_START_THIRD stands inside that same opening action.
const KEYWORD_LAST_START_THIRD = KEYWORD_LAST_START_SECOND + utf8.CHARACTER_SIZE_MINIMUM

// Keyword_Last_Start is where one control keyword's last token began.
type Keyword_Last_Start Cursor_Last_Start_Stored

// Keyword_Last_Start_Invariants keeps that clause inside the source.
func Keyword_Last_Start_Invariants(value Keyword_Last_Start, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Holed_Uint16(
    uint16(value.Value),
    uint16(SOURCE_SIZE_MINIMUM),
    uint16(ALTERNATE_KEYWORD_LAST_START_MAXIMUM),
    uint16(KEYWORD_LAST_START_SECOND),
    uint16(KEYWORD_LAST_START_THIRD),
    uint16(KEYWORD_LAST_START_THIRD),
  ).
    Ensure()
}

// OPENING_TAIL_NODE_COUNT counts two branches, one action and its pipeline above one opening.
const OPENING_TAIL_NODE_COUNT = CONTROL_ACTION_TAIL_NODE_COUNT +
  ACTION_PIPELINE_NODE_COUNT +
  SIMPLE_ACTION_NODE_COUNT

// Opening_Node_Count is the arena length one control opening reads.
type Opening_Node_Count Node_Count_Allocated

// Opening_Node_Count_Invariants keeps those nodes inside the arena.
func Opening_Node_Count_Invariants(value Opening_Node_Count, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(ROOT_NODE_COUNT),
    uint16(CONTROL_OPENER_COUNT_MAXIMUM),
  ).
    Ensure()
}

// Opening_Action_Count is the arena length one definition opens with.
type Opening_Action_Count Action_Node_Count

// Opening_Action_Count_Invariants keeps those nodes inside the arena.
func Opening_Action_Count_Invariants(value Opening_Action_Count, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(ACTION_NODE_COUNT_MINIMUM),
    uint16(ACTION_NODE_COUNT_LIMIT-OPENING_TAIL_NODE_COUNT),
  ).
    Ensure()
}

// Definition_Node_Count is the arena length one definition leaves.
type Definition_Node_Count Dense_Action_Node_Count

// Definition_Node_Count_Invariants keeps those branches inside the arena.
func Definition_Node_Count_Invariants(
  value Definition_Node_Count, namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(ACTION_NODE_COUNT_MINIMUM),
    uint16(ARGUMENT_NODE_COUNT_LIMIT-CLAUSE_TAIL_NODE_COUNT),
  ).
    Ensure()
}

// Block_Node_Count is the arena length one block invocation leaves.
type Block_Node_Count Dense_Action_Node_Count

// Block_Node_Count_Invariants keeps those branches inside the arena.
func Block_Node_Count_Invariants(value Block_Node_Count, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(ACTION_NODE_COUNT_MINIMUM),
    uint16(ARGUMENT_NODE_COUNT_LIMIT-BLOCK_TAIL_NODE_COUNT),
  ).
    Ensure()
}

// CONTROL_BODY_TAIL_NODE_COUNT counts two branches and one action above one body.
const CONTROL_BODY_TAIL_NODE_COUNT = CONTROL_ACTION_TAIL_NODE_COUNT +
  SIMPLE_ACTION_NODE_COUNT

// CONTROL_DEFINITION_TAIL_NODE_COUNT counts three branches below one definition.
const CONTROL_DEFINITION_TAIL_NODE_COUNT = CONTROL_ACTION_TAIL_NODE_COUNT +
  CONTROL_BRANCH_NODE_COUNT

// Control_Action_Node_Count is the arena length one control action reads.
type Control_Action_Node_Count Node_Count_Allocated

// Control_Action_Node_Count_Invariants keeps those branches inside the arena.
func Control_Action_Node_Count_Invariants(
  value Control_Action_Node_Count,
  namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(ROOT_NODE_COUNT),
    uint16(CONTROL_OPENED_COUNT_MAXIMUM),
  ).
    Ensure()
}

// CONTROL_PIPELINE_TAIL_NODE_COUNT counts the branches a clause packs above its pipeline.
const CONTROL_PIPELINE_TAIL_NODE_COUNT = CONTROL_BRANCH_NODE_COUNT +
  CONTROL_BRANCH_NODE_COUNT +
  ACTION_PIPELINE_NODE_COUNT

// Control_Pipeline_Node_Count is the arena length one control clause reads.
type Control_Pipeline_Node_Count Dense_Action_Node_Count

// Control_Pipeline_Node_Count_Invariants keeps those branches inside the arena.
func Control_Pipeline_Node_Count_Invariants(
  value Control_Pipeline_Node_Count,
  namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(ACTION_NODE_COUNT_MINIMUM),
    uint16(BRANCH_ALLOCATION_LINK_MAXIMUM+PIPELINE_NODE_COUNT+
      COMMAND_NODE_COUNT+SCALAR_NODE_COUNT),
  ).
    Ensure()
}

// Control_Pipeline_Action_Count is the arena length one control clause opens with.
type Control_Pipeline_Action_Count Action_Node_Count

// Control_Pipeline_Action_Count_Invariants keeps those branches inside the arena.
func Control_Pipeline_Action_Count_Invariants(
  value Control_Pipeline_Action_Count,
  namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(ACTION_NODE_COUNT_MINIMUM),
    uint16(BRANCH_ALLOCATION_LINK_MAXIMUM),
  ).
    Ensure()
}

// Control_Pipeline_Root names the pipeline node one control clause opened.
type Control_Pipeline_Root Pipeline_Root_Validated

// Control_Pipeline_Root_Invariants keeps those branches above that pipeline.
func Control_Pipeline_Root_Invariants(
  value Control_Pipeline_Root, namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Holed_Uint16(
    uint16(value.Value),
    uint16(NODE_REFERENCE_MINIMUM),
    uint16(BRANCH_ALLOCATION_LINK_MAXIMUM+PIPELINE_NODE_COUNT),
    uint16(ROOT_LIST_HOLE),
    uint16(TERM_ACTION_HOLE),
    uint16(TERM_ACTION_HOLE),
  ).
    Ensure()
}

// Chain_Push_Status is the outcome one chain push reports.
type Chain_Push_Status Term_Status

// Chain_Push_Status_Invariants keeps every other refusal out of that push.
func Chain_Push_Status_Invariants(value Chain_Push_Status, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_Uint8(
    uint8(value),
    uint8(STATUS_OK),
    uint8(STATUS_PROGRAM_INVALID),
  ).
    Ensure()
}

// Child_Pipe_Node is the piped record one child evaluation reads.
type Child_Pipe_Node Pipe_Node

// Child_Pipe_Node_Invariants leaves the child command the pipe still takes.
func Child_Pipe_Node_Invariants(value Child_Pipe_Node, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Start),
    uint16(ACTION_CONTENT_BOUNDARY_MINIMUM),
    uint16(LEXEME_START_MAXIMUM-ACTION_SEPARATOR_SIZE_MINIMUM-
      IDENTIFIER_SIZE_MINIMUM),
  ).
    Range_Uint16(
    uint16(value.End),
    uint16(ACTION_CONTENT_BOUNDARY_MINIMUM+PIPE_SPAN_SIZE_MINIMUM),
    uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM),
  ).
    Range_Uint16(
    uint16(value.Value_Start),
    uint16(ACTION_CONTENT_BOUNDARY_MINIMUM),
    uint16(LEXEME_START_MAXIMUM-ACTION_SEPARATOR_SIZE_MINIMUM-
      IDENTIFIER_SIZE_MINIMUM),
  ).
    Range_Uint16(
    uint16(value.Value_End),
    uint16(ACTION_CONTENT_BOUNDARY_MINIMUM+PIPE_SPAN_SIZE_MINIMUM),
    uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM),
  ).
    Range_Uint16(
    uint16(value.First_Child),
    uint16(ROOT_NODE_COUNT+SIMPLE_ACTION_CONTAINER_NODE_COUNT),
    uint16(COMMAND_TAIL_REFERENCE_MAXIMUM),
  ).
    Range_Holed_Uint16(
    uint16(value.Next_Sibling),
    uint16(NODE_REFERENCE_MINIMUM),
    uint16(ACTION_TAIL_REFERENCE_MAXIMUM),
    uint16(ROOT_LIST_HOLE),
    uint16(TERM_ACTION_HOLE),
    uint16(TERM_ACTION_HOLE),
  ).
    Ensure()
  aver.Always(
    uint8(value.Kind) == uint8(NODE_PIPE),
    "One child pipe record keeps the kind its caller selected.",
  )
}

// Child_Pipe_Node_Stored keeps that record present for its reader.
type Child_Pipe_Node_Stored *Child_Pipe_Node

// Child_Pipe_Node_Stored_Invariants checks the present record.
func Child_Pipe_Node_Stored_Invariants(value Child_Pipe_Node_Stored, namespace aver.Namespace) {
  if value == nil {
    return
  }
  Child_Pipe_Node_Invariants(*value, namespace)
}

// Child_Pipe_Node_Validated carries deep proof of that record.
type Child_Pipe_Node_Validated struct {
  // Value keeps deep proof attached to the record.
  Value Child_Pipe_Node_Stored
}

// Child_Pipe_Node_Validated_Invariants composes the record it carries.
func Child_Pipe_Node_Validated_Invariants(
  value *Child_Pipe_Node_Validated,
  namespace aver.Namespace,
) {
  Child_Pipe_Node_Stored_Invariants(value.Value, namespace)
}

// CHILD_CHAIN_VALUE_START_MINIMUM follows the parenthesised child the chain reads.
const CHILD_CHAIN_VALUE_START_MINIMUM = ACTION_CONTENT_BOUNDARY_MINIMUM +
  PARENTHESIS_SIZE +
  DOT_SOURCE_SIZE +
  PARENTHESIS_SIZE

// CHILD_CHAIN_END_MINIMUM closes the shortest field that child chain names.
const CHILD_CHAIN_END_MINIMUM = CHILD_CHAIN_VALUE_START_MINIMUM +
  DOT_SOURCE_SIZE +
  IDENTIFIER_SIZE_MINIMUM

// CHILD_CHAIN_FIRST_CHILD_MAXIMUM leaves one field action above the child it names.
const CHILD_CHAIN_FIRST_CHILD_MAXIMUM = NODE_REFERENCE_MAXIMUM -
  ROOT_NODE_COUNT -
  FIELD_ACTION_NODE_COUNT -
  TRAILING_TEXT_NODE_COUNT_MAXIMUM

// Child_Chain_Node is the chain record one child evaluation reads.
type Child_Chain_Node Chain_Node

// Child_Chain_Node_Invariants keeps the parenthesised child beneath the chain.
func Child_Chain_Node_Invariants(value Child_Chain_Node, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Start),
    uint16(ACTION_CONTENT_BOUNDARY_MINIMUM),
    uint16(CHAIN_START_MAXIMUM-DOT_SOURCE_SIZE-ACTION_BOUNDARY_SIZE_MINIMUM),
  ).
    Range_Uint16(
    uint16(value.End),
    uint16(CHILD_CHAIN_END_MINIMUM),
    uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM),
  ).
    Range_Uint16(
    uint16(value.Value_Start),
    uint16(CHILD_CHAIN_VALUE_START_MINIMUM),
    uint16(CHAIN_START_MAXIMUM),
  ).
    Range_Uint16(
    uint16(value.Value_End),
    uint16(CHILD_CHAIN_END_MINIMUM),
    uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM),
  ).
    Range_Uint16(
    uint16(value.First_Child),
    uint16(COMMAND_NODE_COUNT_MINIMUM),
    uint16(CHILD_CHAIN_FIRST_CHILD_MAXIMUM),
  ).
    Range_Holed_Uint16(
    uint16(value.Next_Sibling),
    uint16(NODE_REFERENCE_MINIMUM),
    uint16(ACTION_TAIL_REFERENCE_MAXIMUM),
    uint16(ROOT_LIST_HOLE),
    uint16(TERM_ACTION_HOLE),
    uint16(TERM_PIPELINE_HOLE),
  ).
    Ensure()
  aver.Always(
    uint8(value.Kind) == uint8(NODE_CHAIN),
    "One child chain record keeps the kind its caller selected.",
  )
}

// Child_Chain_Node_Stored keeps that record present for its reader.
type Child_Chain_Node_Stored *Child_Chain_Node

// Child_Chain_Node_Stored_Invariants checks the record when one is present.
func Child_Chain_Node_Stored_Invariants(
  value Child_Chain_Node_Stored, namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Child_Chain_Node_Invariants(*value, namespace)
}

// Child_Chain_Node_Optional carries proof of that record when one is present.
type Child_Chain_Node_Optional struct {
  // Value keeps optional-chain proof attached to the record.
  Value Child_Chain_Node_Stored
}

// Child_Chain_Node_Optional_Invariants checks the record when one is present.
func Child_Chain_Node_Optional_Invariants(
  value *Child_Chain_Node_Optional,
  namespace aver.Namespace,
) {
  Child_Chain_Node_Stored_Invariants(value.Value, namespace)
}

// DEFINITION_NODE_SIZE_MINIMUM spans the shortest named definition.
const DEFINITION_NODE_SIZE_MINIMUM = TEMPLATE_DEFINITION_SIZE_MINIMUM

// FOUND_DEFINITION_START_MAXIMUM leaves one shortest definition at the source end.
const FOUND_DEFINITION_START_MAXIMUM = SOURCE_SIZE_MAXIMUM - TEMPLATE_DEFINITION_SIZE_MINIMUM

// FOUND_DEFINITION_NAME_START_MAXIMUM follows that definition's delimiter, keyword, and
// separator.
const FOUND_DEFINITION_NAME_START_MAXIMUM = FOUND_DEFINITION_START_MAXIMUM +
  CUSTOM_DELIMITER_SIZE_MINIMUM +
  len(KEYWORD_DEFINE) +
  ACTION_SEPARATOR_SIZE_MINIMUM

// FOUND_DEFINITION_NAME_END_MAXIMUM follows that definition's quoted empty name.
const FOUND_DEFINITION_NAME_END_MAXIMUM = FOUND_DEFINITION_NAME_START_MAXIMUM +
  TEMPLATE_NAME_SOURCE_SIZE_MINIMUM

// Found_Definition_Node is the definition record one lookup gives its caller.
type Found_Definition_Node Definition_Node

// Found_Definition_Node_Invariants spans the definition the lookup found.
func Found_Definition_Node_Invariants(value Found_Definition_Node, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Start),
    uint16(SOURCE_SIZE_MINIMUM),
    uint16(FOUND_DEFINITION_START_MAXIMUM),
  ).
    Range_Uint16(
    uint16(value.End),
    uint16(DEFINITION_NODE_SIZE_MINIMUM),
    uint16(SOURCE_SIZE_MAXIMUM),
  ).
    Range_Uint16(
    uint16(value.Value_Start),
    uint16(INVOCATION_NAME_START_MINIMUM),
    uint16(FOUND_DEFINITION_NAME_START_MAXIMUM),
  ).
    Range_Uint16(
    uint16(value.Value_End),
    uint16(INVOCATION_NAME_END_MINIMUM),
    uint16(FOUND_DEFINITION_NAME_END_MAXIMUM),
  ).
    Range_Uint16(
    uint16(value.First_Child),
    uint16(ROOT_NODE_COUNT+ACTION_PIPELINE_NODE_COUNT),
    uint16(DOCUMENT_FIRST_TEMPLATE_REFERENCE_MAXIMUM+NODE_COUNT_INCREMENT),
  ).
    Range_Holed_Uint16(
    uint16(value.Next_Sibling),
    uint16(NODE_REFERENCE_MINIMUM),
    uint16(NAMED_PAIR_LATER_REFERENCE_MAXIMUM),
    uint16(ROOT_LIST_HOLE),
    uint16(TERM_ACTION_HOLE),
    uint16(TERM_ACTION_HOLE),
  ).
    Ensure()
  aver.Always(
    uint8(value.Kind) == uint8(NODE_TEMPLATE),
    "One found definition keeps the kind its parser selected.",
  )
}

// Found_Definition_Node_Stored keeps that record present for its reader.
type Found_Definition_Node_Stored *Found_Definition_Node

// Found_Definition_Node_Stored_Invariants checks the record when one is present.
func Found_Definition_Node_Stored_Invariants(
  value Found_Definition_Node_Stored,
  namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Found_Definition_Node_Invariants(*value, namespace)
}

// Found_Definition_Node_Validated carries deep proof of that record.
type Found_Definition_Node_Validated struct {
  // Value keeps deep proof attached to the record.
  Value Found_Definition_Node_Stored
}

// Found_Definition_Node_Validated_Invariants composes the record it carries.
func Found_Definition_Node_Validated_Invariants(
  value *Found_Definition_Node_Validated,
  namespace aver.Namespace,
) {
  Found_Definition_Node_Stored_Invariants(value.Value, namespace)
}

// Invoking_Source is the source one template invocation stands inside.
type Invoking_Source Source

// Invoking_Source_Invariants keeps the shortest invocation inside that source.
func Invoking_Source_Invariants(value Invoking_Source, namespace aver.Namespace) {
  Invoking_Source_Data_Invariants(Invoking_Source_Data(value.Data), namespace)
}

// Invoking_Source_Data is the byte header one invoking source keeps.
type Invoking_Source_Data Source_Storage

// Invoking_Source_Data_Invariants keeps the shortest invocation inside those bytes.
func Invoking_Source_Data_Invariants(value Invoking_Source_Data, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(
    len(value.Value),
    TEMPLATE_INVOCATION_SIZE_MINIMUM,
    SOURCE_SIZE_MAXIMUM,
  ).
    Ensure()
}

// INVOCATION_ARGUMENT_SIZE is the shortest invocation that passes one argument.
const INVOCATION_ARGUMENT_SIZE = TEMPLATE_INVOCATION_SIZE_MINIMUM +
  ACTION_SEPARATOR_SIZE_MINIMUM +
  DOT_SOURCE_SIZE

// INVOCATION_PIPELINE_REFERENCE_MAXIMUM packs the densest actions and one text node before that
// invocation, then its node and its pipeline.
const INVOCATION_PIPELINE_REFERENCE_MAXIMUM = ROOT_NODE_COUNT +
  (SOURCE_SIZE_MAXIMUM-INVOCATION_ARGUMENT_SIZE)/SIMPLE_ACTION_SIZE_MINIMUM*
    SIMPLE_ACTION_NODE_COUNT +
  ((SOURCE_SIZE_MAXIMUM-INVOCATION_ARGUMENT_SIZE)%SIMPLE_ACTION_SIZE_MINIMUM+
    SIMPLE_ACTION_SIZE_MINIMUM-
    SOURCE_POSITION_INCREMENT)/
    SIMPLE_ACTION_SIZE_MINIMUM +
  NODE_COUNT_INCREMENT +
  PIPELINE_NODE_COUNT

// INVOKING_NODE_COUNT_MAXIMUM packs the densest actions and one text node beside one shortest
// invocation.
const INVOKING_NODE_COUNT_MAXIMUM = ROOT_NODE_COUNT +
  (SOURCE_SIZE_MAXIMUM-TEMPLATE_INVOCATION_SIZE_MINIMUM)/
    SIMPLE_ACTION_SIZE_MINIMUM*
    SIMPLE_ACTION_NODE_COUNT +
  ((SOURCE_SIZE_MAXIMUM-TEMPLATE_INVOCATION_SIZE_MINIMUM)%SIMPLE_ACTION_SIZE_MINIMUM+
    SIMPLE_ACTION_SIZE_MINIMUM-
    SOURCE_POSITION_INCREMENT)/
    SIMPLE_ACTION_SIZE_MINIMUM +
  NODE_COUNT_INCREMENT

// Invoking_Program_Document is arena metadata of a program that holds an invocation.
type Invoking_Program_Document Program_Validated_Document

// Invoking_Program_Document_Invariants keeps the invoked definition inside the arena.
func Invoking_Program_Document_Invariants(
  value Invoking_Program_Document,
  namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Node_Count),
    uint16(ROOT_NODE_COUNT+ACTION_NODE_COUNT),
    uint16(INVOKING_NODE_COUNT_MAXIMUM),
  ).
    Range_Uint16(
    uint16(value.First_Template),
    uint16(
      DOCUMENT_FIRST_TEMPLATE_MINIMUM,
    ),
    uint16(DOCUMENT_FIRST_TEMPLATE_MAXIMUM),
  ).
    Range_Uint16(
    uint16(value.Template_Count),
    uint16(TEMPLATE_COUNT_MINIMUM),
    uint16(
      EXECUTED_TEMPLATE_COUNT_MAXIMUM,
    ),
  ).
    Ensure()
  aver.Always(
    uint16(value.Root) == uint16(ROOT_NODE_COUNT),
    "One invoking program keeps the root in the first syntax arena slot.",
  )
}

// Invoking_Program is one program whose action invokes a named definition.
type Invoking_Program Program_Validated

// Invoking_Program_Invariants composes that source with arena metadata.
func Invoking_Program_Invariants(value Invoking_Program, namespace aver.Namespace) {
  Invoking_Source_Invariants(Invoking_Source(value.Source), namespace)
  Invoking_Program_Document_Invariants(
    Invoking_Program_Document(value.Document), namespace,
  )
  Syntax_Storage_Invariants(&value.Syntax, namespace)
}

// Declaration_Pipeline_Frame_State is frame state one declaration writes into.
type Declaration_Pipeline_Frame_State Pipeline_Frame_State_Stored

// Declaration_Pipeline_Frame_State_Invariants leaves command and argument slots empty.
func Declaration_Pipeline_Frame_State_Invariants(
  value Declaration_Pipeline_Frame_State,
  namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Pipe),
    uint16(PIPELINE_NODE_COUNT_MINIMUM),
    uint16(BINDING_PACKED_COUNT+ACTION_PIPELINE_NODE_COUNT),
  ).
    Range_Holed_Uint16(
    uint16(value.Last_Child),
    uint16(NODE_REFERENCE_MINIMUM),
    uint16(MISPLACED_DECLARATION_COMMAND_MAXIMUM),
    uint16(ROOT_LIST_HOLE),
    uint16(TERM_ACTION_HOLE),
    uint16(TERM_PIPELINE_HOLE),
  ).
    Range_Holed_Uint16(
    uint16(value.Command),
    uint16(NODE_REFERENCE_MINIMUM),
    uint16(MISPLACED_DECLARATION_COMMAND_MAXIMUM),
    uint16(ROOT_LIST_HOLE),
    uint16(TERM_ACTION_HOLE),
    uint16(TERM_PIPELINE_HOLE),
  ).
    Range_Holed_Uint16(
    uint16(value.Last_Argument),
    uint16(NODE_REFERENCE_MINIMUM),
    uint16(MISPLACED_DECLARATION_COMMAND_MAXIMUM-NODE_COUNT_INCREMENT),
    uint16(ROOT_LIST_HOLE),
    uint16(TERM_ACTION_HOLE),
    uint16(TERM_PIPELINE_HOLE),
  ).
    Ensure()
}

// Declaration_Pipeline_Frame is frame storage one declaration writes into.
type Declaration_Pipeline_Frame Pipeline_Frame_Storage

// Declaration_Pipeline_Frame_Invariants composes the frame state it holds.
func Declaration_Pipeline_Frame_Invariants(
  value Declaration_Pipeline_Frame,
  namespace aver.Namespace,
) {
  Declaration_Pipeline_Frame_State_Invariants(
    Declaration_Pipeline_Frame_State(value.Value),
    namespace,
  )
}

// Declaration_Pipeline_Frame_Pointer keeps that frame mutable for its caller.
type Declaration_Pipeline_Frame_Pointer *Declaration_Pipeline_Frame

// Declaration_Pipeline_Frame_Pointer_Invariants composes present frame storage.
func Declaration_Pipeline_Frame_Pointer_Invariants(
  value Declaration_Pipeline_Frame_Pointer,
  namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Declaration_Pipeline_Frame_Invariants(*value, namespace)
}

// BRANCH_BODY_REFERENCE_MINIMUM names the first body list one branch can own.
const BRANCH_BODY_REFERENCE_MINIMUM = ROOT_NODE_COUNT + CONTROL_BRANCH_NODE_COUNT

// Branch_Body_Link is the body list one branch owns.
type Branch_Body_Link Node_Link_Proof_Stored

// Branch_Body_Link_Invariants keeps the branch and its term beneath the body.
func Branch_Body_Link_Invariants(value Branch_Body_Link, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value),
    uint16(BRANCH_BODY_REFERENCE_MINIMUM),
    uint16(BRANCH_BODY_LINK_MAXIMUM),
  ).
    Ensure()
}

// Branch_Body_Validated carries proof of that body list.
type Branch_Body_Validated struct {
  // Value keeps arena proof attached to the link.
  Value Branch_Body_Link
}

// Branch_Body_Validated_Invariants composes the link it carries.
func Branch_Body_Validated_Invariants(value *Branch_Body_Validated, namespace aver.Namespace) {
  Branch_Body_Link_Invariants(value.Value, namespace)
}

// BRANCH_PIPE_REFERENCE_MINIMUM names the first pipeline one branch can own.
const BRANCH_PIPE_REFERENCE_MINIMUM = ROOT_NODE_COUNT + ACTION_PIPELINE_NODE_COUNT

// Branch_Pipe_Reference is the pipeline one branch reads its term through.
type Branch_Pipe_Reference Pipeline_Root_Proof_Stored

// Branch_Pipe_Reference_Invariants keeps the branch action beneath the pipeline.
func Branch_Pipe_Reference_Invariants(value Branch_Pipe_Reference, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value),
    uint16(BRANCH_PIPE_REFERENCE_MINIMUM),
    uint16(BRANCH_PIPE_REFERENCE_MAXIMUM),
  ).
    Ensure()
}

// Branch_Pipe_Validated carries proof of that pipeline.
type Branch_Pipe_Validated struct {
  // Value keeps arena proof attached to the reference.
  Value Branch_Pipe_Reference
}

// Branch_Pipe_Validated_Invariants composes the reference it carries.
func Branch_Pipe_Validated_Invariants(value *Branch_Pipe_Validated, namespace aver.Namespace) {
  Branch_Pipe_Reference_Invariants(value.Value, namespace)
}

// Control_Classified_Token_Kind carries that kind beside its known flag.
type Control_Classified_Token_Kind Classified_Token_Kind

// Control_Classified_Token_Kind_Invariants composes the kind it carries.
func Control_Classified_Token_Kind_Invariants(
  value Control_Classified_Token_Kind,
  namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint8(
    uint8(value.Value),
    uint8(TOKEN_IDENTIFIER),
    uint8(TOKEN_KIND_MAXIMUM),
  ).
    Ensure()
}

// Word_Classified_Token_Kind carries that kind beside its known flag.
type Word_Classified_Token_Kind Classified_Token_Kind

// Word_Classified_Token_Kind_Invariants composes the kind it carries.
func Word_Classified_Token_Kind_Invariants(
  value Word_Classified_Token_Kind,
  namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint8(
    uint8(value.Value),
    uint8(TOKEN_BOOLEAN),
    uint8(TOKEN_KIND_MAXIMUM),
  ).
    Ensure()
}

// COMPLEX_OUTPUT_SIZE_MINIMUM spans the shortest complex one output holds.
const COMPLEX_OUTPUT_SIZE_MINIMUM = len("(0+0i)")

// Complex_Output_Count is the output cursor one complex conversion leaves behind.
type Complex_Output_Count Output_Count_Staged

// Complex_Output_Count_Invariants keeps the shortest complex beneath the cursor.
func Complex_Output_Count_Invariants(value Complex_Output_Count, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(COMPLEX_OUTPUT_SIZE_MINIMUM),
    uint16(OUTPUT_SIZE_MAXIMUM),
  ).
    Ensure()
}

// Quoted_Cursor is one lexer standing at the opening quote of a literal.
type Quoted_Cursor Token_Cursor

// Quoted_Cursor_Invariants composes lexer state beside the empty token.
func Quoted_Cursor_Invariants(value Quoted_Cursor, namespace aver.Namespace) {
  Quoted_Lexeme_Position_Invariants(Quoted_Lexeme_Position(value.Position), namespace)
  Quoted_Boundary_Invariants(Quoted_Boundary(value.End), namespace)
  Quoted_Eof_Token_Storage_Invariants(Quoted_Eof_Token_Storage(value.Token), namespace)
  Unread_Look_Ahead_Storage_Invariants(Unread_Look_Ahead_Storage(value.Looked), namespace)
  Cursor_Number_Stored_Invariants((*Cursor_Number_Stored)(&value.Number), namespace)
  Function_Function_Storage_Invariants(&value.Function, namespace)
  Quoted_Last_Start_Invariants(Quoted_Last_Start(value.Last_Start), namespace)
  Cursor_Has_Last_Stored_Invariants((*Cursor_Has_Last_Stored)(&value.Has_Last), namespace)
}

// Quoted_Cursor_Pointer keeps mutation of that lexer typed.
type Quoted_Cursor_Pointer *Quoted_Cursor

// Quoted_Cursor_Pointer_Invariants composes present lexer storage.
func Quoted_Cursor_Pointer_Invariants(value Quoted_Cursor_Pointer, namespace aver.Namespace) {
  if value == nil {
    return
  }
  Quoted_Cursor_Invariants(*value, namespace)
}

// MERGED_TEMPLATE_REFERENCE_MINIMUM names the first definition one root list holds.
const MERGED_TEMPLATE_REFERENCE_MINIMUM = ROOT_NODE_COUNT + NODE_COUNT_INCREMENT

// Merged_Template_First is the first definition one merge already holds.
type Merged_Template_First Template_First_Stored

// Merged_Template_First_Invariants keeps the root list beneath that definition.
func Merged_Template_First_Invariants(value Merged_Template_First, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(MERGED_TEMPLATE_REFERENCE_MINIMUM),
    uint16(EARLIER_TEMPLATE_REFERENCE_MAXIMUM),
  ).
    Ensure()
}

// Merged_Template_Last is the last definition one merge already holds.
type Merged_Template_Last Template_Last_Stored

// Merged_Template_Last_Invariants keeps the root list beneath that definition.
func Merged_Template_Last_Invariants(value Merged_Template_Last, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(MERGED_TEMPLATE_REFERENCE_MINIMUM),
    uint16(EARLIER_TEMPLATE_REFERENCE_MAXIMUM),
  ).
    Ensure()
}

// Merged_Template_Count counts the definitions one merge already holds.
type Merged_Template_Count Template_Count_Stored

// Merged_Template_Count_Invariants keeps one definition inside that count.
func Merged_Template_Count_Invariants(value Merged_Template_Count, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(TEMPLATE_COUNT_MINIMUM+NODE_COUNT_INCREMENT),
    uint16(EXECUTED_TEMPLATE_COUNT_MAXIMUM),
  ).
    Ensure()
}

// Merged_Template_State is definition state one merge reads.
type Merged_Template_State Template_State

// Merged_Template_State_Invariants composes both definition links beside the count.
func Merged_Template_State_Invariants(value Merged_Template_State, namespace aver.Namespace) {
  Merged_Template_First_Invariants(Merged_Template_First(value.First), namespace)
  Merged_Template_Last_Invariants(Merged_Template_Last(value.Last), namespace)
  Merged_Template_Count_Invariants(Merged_Template_Count(value.Count), namespace)
}

// Merged_Template_State_Pointer keeps that state mutable for its caller.
type Merged_Template_State_Pointer *Merged_Template_State

// Merged_Template_State_Pointer_Invariants composes present definition state.
func Merged_Template_State_Pointer_Invariants(
  value Merged_Template_State_Pointer,
  namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Merged_Template_State_Invariants(*value, namespace)
}

// Refusal_Diagnostic is the refusal one run gives back before it executes a node.
type Refusal_Diagnostic Diagnostic

// Refusal_Diagnostic_Invariants keeps the refusal at no source position.
func Refusal_Diagnostic_Invariants(value Refusal_Diagnostic, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Holed_Uint8(
    uint8(value.Code),
    uint8(EXECUTION_REFUSED_STATUS_VALUE_MINIMUM),
    uint8(STATUS_LIMIT_EXCEEDED),
    uint8(STATUS_OUTPUT_TOO_LARGE),
    uint8(STATUS_OUTPUT_TOO_LARGE),
    uint8(STATUS_OUTPUT_TOO_LARGE),
  ).
    Ensure()
  aver.Always(
    uint16(value.Position) == uint16(bytes.SLICE_SIZE_MINIMUM),
    "One refusal before execution names no source position.",
  )
}

// DECLARED_NAME_START_MAXIMUM leaves the operator the declaration takes.
const DECLARED_NAME_START_MAXIMUM = LEXEME_START_MAXIMUM - ASSIGNMENT_OPERATOR_SIZE

// DECLARED_NAME_END_MAXIMUM closes that name before the operator.
const DECLARED_NAME_END_MAXIMUM = ACTION_CONTENT_BOUNDARY_MAXIMUM -
  ASSIGNMENT_OPERATOR_SIZE

// DECLARED_CURSOR_END_MINIMUM closes the shortest name and operator one action holds.
const DECLARED_CURSOR_END_MINIMUM = ACTION_CONTENT_BOUNDARY_MINIMUM +
  VARIABLE_NAME_SIZE_MINIMUM +
  ASSIGNMENT_OPERATOR_SIZE

// Declared_Name_Token is the variable name one declaration binds.
type Declared_Name_Token Variable_Token

// Declared_Name_Token_Invariants leaves the operator and value above the name.
func Declared_Name_Token_Invariants(value Declared_Name_Token, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Start),
    uint16(TOKEN_BOUNDARY_MINIMUM),
    uint16(DECLARED_NAME_START_MAXIMUM),
  ).
    Range_Uint16(
    uint16(value.End),
    uint16(TOKEN_BOUNDARY_MINIMUM+VARIABLE_NAME_SIZE_MINIMUM),
    uint16(DECLARED_NAME_END_MAXIMUM),
  ).
    Ensure()
  aver.Always(
    uint8(value.Kind) == uint8(TOKEN_VARIABLE),
    "One declaration cursor retains the name its caller read.",
  )
}

// Declared_Name_Token_Storage is that name inside its cursor field.
type Declared_Name_Token_Storage Token_Storage

// Declared_Name_Token_Storage_Invariants composes the retained name.
func Declared_Name_Token_Storage_Invariants(
  value Declared_Name_Token_Storage,
  namespace aver.Namespace,
) {
  Declared_Name_Token_Invariants(Declared_Name_Token(value.Value), namespace)
}

// Declared_Position is where one lexer stands once it read the declared name.
type Declared_Position Cursor_Position_Stored

// Declared_Position_Invariants keeps the name beneath the position.
func Declared_Position_Invariants(value Declared_Position, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(ACTION_CONTENT_BOUNDARY_MINIMUM+VARIABLE_NAME_SIZE_MINIMUM),
    uint16(DECLARED_NAME_END_MAXIMUM),
  ).
    Ensure()
}

// Declared_End is where one lexer stops once it read the declared name.
type Declared_End Cursor_End_Stored

// Declared_End_Invariants keeps the whole declaration beneath the boundary.
func Declared_End_Invariants(value Declared_End, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(DECLARED_CURSOR_END_MINIMUM),
    uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM),
  ).
    Ensure()
}

// Declared_Cursor is one lexer that read the name of a declaration.
type Declared_Cursor Token_Cursor

// Declared_Cursor_Invariants composes lexer state beside the retained name.
func Declared_Cursor_Invariants(value Declared_Cursor, namespace aver.Namespace) {
  Declared_Position_Invariants(Declared_Position(value.Position), namespace)
  Declared_End_Invariants(Declared_End(value.End), namespace)
  Declared_Name_Token_Storage_Invariants(
    Declared_Name_Token_Storage(value.Token), namespace,
  )
  Peeked_Look_Ahead_Storage_Invariants(Peeked_Look_Ahead_Storage(value.Looked), namespace)
  Cursor_Number_Stored_Invariants((*Cursor_Number_Stored)(&value.Number), namespace)
  Function_Function_Storage_Invariants(&value.Function, namespace)
  Declared_Last_Start_Invariants(Declared_Last_Start(value.Last_Start), namespace)
  Cursor_Has_Last_Stored_Invariants((*Cursor_Has_Last_Stored)(&value.Has_Last), namespace)
}

// Declared_Cursor_Pointer keeps mutation of that lexer typed.
type Declared_Cursor_Pointer *Declared_Cursor

// Declared_Cursor_Pointer_Invariants composes present lexer storage.
func Declared_Cursor_Pointer_Invariants(value Declared_Cursor_Pointer, namespace aver.Namespace) {
  if value == nil {
    return
  }
  Declared_Cursor_Invariants(*value, namespace)
}

// CONTROL_ALTERNATE_ABSENT marks the branch that has taken no alternate arm yet.
const CONTROL_ALTERNATE_ABSENT = false

// Alternate_Control_Frame_State is branch state one alternate arm closes over.
type Alternate_Control_Frame_State Control_Frame_State_Stored

// Alternate_Control_Frame_State_Invariants leaves the arm above the branch it opens.
func Alternate_Control_Frame_State_Invariants(
  value Alternate_Control_Frame_State,
  namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Branch),
    uint16(ACTION_NODE_COUNT_MINIMUM),
    uint16(UNCLOSED_ELSE_BRANCH_MAXIMUM),
  ).
    Range_Holed_Uint16(
    uint16(value.Parent_List),
    uint16(ROOT_NODE_COUNT),
    uint16(NESTED_ELSE_PARENT_LIST_MAXIMUM),
    uint16(TERM_ACTION_HOLE),
    uint16(TERM_ACTION_HOLE),
    uint16(TERM_ACTION_HOLE),
  ).
    Range_Uint16(
    uint16(value.Parent_Last_Child),
    uint16(ACTION_NODE_COUNT_MINIMUM),
    uint16(UNCLOSED_ELSE_BRANCH_MAXIMUM),
  ).
    Range_Uint16(
    uint16(value.Parent_Variable_Count),
    uint16(VARIABLE_COUNT_MINIMUM),
    uint16(DEFINITION_CLAUSE_VARIABLE_COUNT_MAXIMUM),
  ).
    Range_Uint16(
    uint16(value.Branch_Variable_Count),
    uint16(VARIABLE_COUNT_MINIMUM),
    uint16(DEFINITION_CLAUSE_VARIABLE_COUNT_MAXIMUM),
  ).
    Range_Uint8(
    uint8(value.Kind),
    uint8(NODE_IF),
    uint8(NODE_WITH),
  ).
    Sometimes(
    bool(value.Shares_End),
    "One alternate arm shares its parent's end.",
  ).
    Ensure()
  aver.Always(
    bool(value.Has_Else) == CONTROL_ALTERNATE_ABSENT,
    "One branch takes its first alternate arm once.",
  )
}

// Alternate_Control_Frame is branch storage one alternate arm writes into.
type Alternate_Control_Frame Control_Frame_Storage

// Alternate_Control_Frame_Invariants composes the branch state it holds.
func Alternate_Control_Frame_Invariants(
  value Alternate_Control_Frame, namespace aver.Namespace,
) {
  Alternate_Control_Frame_State_Invariants(
    Alternate_Control_Frame_State(value.Value), namespace,
  )
}

// Alternate_Control_Frame_Pointer keeps that frame mutable for its caller.
type Alternate_Control_Frame_Pointer *Alternate_Control_Frame

// Alternate_Control_Frame_Pointer_Invariants composes present branch storage.
func Alternate_Control_Frame_Pointer_Invariants(
  value Alternate_Control_Frame_Pointer,
  namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Alternate_Control_Frame_Invariants(*value, namespace)
}

// Stored_First_Declaration is the first name one declaration list binds.
type Stored_First_Declaration Pipeline_First_Declaration

// Stored_First_Declaration_Invariants leaves the operator and value above that name.
func Stored_First_Declaration_Invariants(
  value Stored_First_Declaration, namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Start),
    uint16(TOKEN_BOUNDARY_MINIMUM),
    uint16(DECLARED_NAME_START_MAXIMUM),
  ).
    Range_Uint16(
    uint16(value.End),
    uint16(TOKEN_BOUNDARY_MINIMUM+VARIABLE_NAME_SIZE_MINIMUM),
    uint16(DECLARED_NAME_END_MAXIMUM),
  ).
    Ensure()
  aver.Always(
    uint8(value.Kind) == uint8(TOKEN_VARIABLE),
    "One declaration list names its first binding with a variable token.",
  )
}

// Stored_First_Declaration_Storage keeps that name inside parser scratch.
type Stored_First_Declaration_Storage struct {
  // Value prevents a stale scalar proof from escaping parser storage.
  Value Stored_First_Declaration
}

// Stored_First_Declaration_Storage_Invariants composes the name it keeps.
func Stored_First_Declaration_Storage_Invariants(
  value *Stored_First_Declaration_Storage,
  namespace aver.Namespace,
) {
  Stored_First_Declaration_Invariants(value.Value, namespace)
}

// Stored_Declarations is the declaration list one store step reads.
type Stored_Declarations struct {
  // First retains the required declaration token.
  First Stored_First_Declaration_Storage
  // Second retains an optional range value declaration.
  Second Pipeline_Second_Declaration_Storage
}

// Stored_Declarations_Invariants composes both names it holds.
func Stored_Declarations_Invariants(value *Stored_Declarations, namespace aver.Namespace) {
  Stored_First_Declaration_Storage_Invariants(&value.First, namespace)
  Pipeline_Second_Declaration_Storage_Invariants(&value.Second, namespace)
}

// RANGE_BODY_START_MINIMUM follows the shortest range clause one source holds.
const RANGE_BODY_START_MINIMUM = ACTION_BOUNDARY_SIZE_MINIMUM +
  len(KEYWORD_RANGE) +
  ACTION_SEPARATOR_SIZE_MINIMUM +
  DOT_SOURCE_SIZE

// RANGE_BODY_START_MAXIMUM leaves the close above that start, since a body may be empty.
const RANGE_BODY_START_MAXIMUM = SOURCE_SIZE_MAXIMUM - TEMPLATE_DEFINITION_END_SIZE_MINIMUM

// Ranged_Body_List is the body record one range iteration walks.
type Ranged_Body_List List_Node

// Ranged_Body_List_Invariants keeps the range clause beneath that body.
func Ranged_Body_List_Invariants(value Ranged_Body_List, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Start),
    uint16(RANGE_BODY_START_MINIMUM),
    uint16(RANGE_BODY_START_MAXIMUM),
  ).
    Range_Uint16(
    uint16(value.End),
    uint16(RANGE_BODY_START_MINIMUM),
    uint16(SOURCE_SIZE_MAXIMUM-TEMPLATE_DEFINITION_END_SIZE_MINIMUM),
  ).
    Range_Holed_Uint16(
    uint16(value.First_Child),
    uint16(NODE_REFERENCE_MINIMUM),
    uint16(RANGED_PROGRAM_NODE_COUNT_MAXIMUM),
    uint16(ROOT_LIST_HOLE),
    uint16(TERM_ACTION_HOLE),
    uint16(TERM_ACTION_HOLE),
  ).
    Range_Holed_Uint16(
    uint16(value.Next_Sibling),
    uint16(NODE_REFERENCE_MINIMUM),
    uint16(RANGE_ELSE_REFERENCE_MAXIMUM),
    uint16(ROOT_LIST_HOLE),
    uint16(TERM_ACTION_HOLE),
    uint16(TERM_ACTION_HOLE),
  ).
    Ensure()
  aver.Always(
    uint16(value.Value_Start) == uint16(SOURCE_SIZE_MINIMUM),
    "One ranged body record keeps no payload start.",
  )
  aver.Always(
    uint16(value.Value_End) == uint16(SOURCE_SIZE_MINIMUM),
    "One ranged body record keeps no payload end.",
  )
  aver.Always(
    uint8(value.Kind) == uint8(NODE_LIST),
    "One ranged body record keeps the kind its caller selected.",
  )
}

// Ranged_Body_List_Stored keeps that record opaque between parser and evaluator.
type Ranged_Body_List_Stored *Ranged_Body_List

// Ranged_Body_List_Stored_Invariants checks the present record.
func Ranged_Body_List_Stored_Invariants(
  value Ranged_Body_List_Stored, namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Ranged_Body_List_Invariants(*value, namespace)
}

// Ranged_Body_List_Validated carries deep proof of that record.
type Ranged_Body_List_Validated struct {
  // Value keeps deep proof attached to the record.
  Value Ranged_Body_List_Stored
}

// Ranged_Body_List_Validated_Invariants composes the record it carries.
func Ranged_Body_List_Validated_Invariants(
  value *Ranged_Body_List_Validated,
  namespace aver.Namespace,
) {
  Ranged_Body_List_Stored_Invariants(value.Value, namespace)
}

// ALTERNATE_LIST_REFERENCE_MINIMUM names the body list one alternate arm closes over.
const ALTERNATE_LIST_REFERENCE_MINIMUM = ROOT_NODE_COUNT + CONTROL_BRANCH_NODE_COUNT

// Alternate_List_State is the list state one alternate arm writes into.
type Alternate_List_State List_State_Value_Stored

// Alternate_List_State_Invariants leaves the branch above the list it names.
func Alternate_List_State_Invariants(value Alternate_List_State, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.List),
    uint16(ALTERNATE_LIST_REFERENCE_MINIMUM),
    uint16(UNCLOSED_ELSE_COUNT_MAXIMUM),
  ).
    Range_Holed_Uint16(
    uint16(value.Last_Child),
    uint16(NODE_REFERENCE_MINIMUM),
    uint16(UNCLOSED_ELSE_COUNT_MAXIMUM),
    uint16(ROOT_LIST_HOLE),
    uint16(TERM_ACTION_HOLE),
    uint16(TERM_ACTION_HOLE),
  ).
    Ensure()
}

// Alternate_List_Storage owns that list state inside parser state.
type Alternate_List_Storage List_State_Storage

// Alternate_List_Storage_Invariants composes the list state it owns.
func Alternate_List_Storage_Invariants(
  value Alternate_List_Storage, namespace aver.Namespace,
) {
  Alternate_List_State_Invariants(Alternate_List_State(value.Value), namespace)
}

// Alternate_List_Storage_Pointer keeps that state mutable for its caller.
type Alternate_List_Storage_Pointer *Alternate_List_Storage

// Alternate_List_Storage_Pointer_Invariants composes present list storage.
func Alternate_List_Storage_Pointer_Invariants(
  value Alternate_List_Storage_Pointer,
  namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Alternate_List_Storage_Invariants(*value, namespace)
}

// CALLED_IDENTIFIER_END_HOLE is where no name ends, because one byte opens it.
const CALLED_IDENTIFIER_END_HOLE = ACTION_CONTENT_BOUNDARY_MINIMUM

// CALLED_IDENTIFIER_START_MAXIMUM leaves the argument the called name still takes.
const CALLED_IDENTIFIER_START_MAXIMUM = LEXEME_START_MAXIMUM -
  ACTION_SEPARATOR_SIZE_MINIMUM -
  DOT_SOURCE_SIZE

// CALLED_IDENTIFIER_END_MAXIMUM closes that name before its argument.
const CALLED_IDENTIFIER_END_MAXIMUM = ACTION_CONTENT_BOUNDARY_MAXIMUM -
  ACTION_SEPARATOR_SIZE_MINIMUM -
  DOT_SOURCE_SIZE

// Called_Identifier_Span locates that name inside the source it was read from.
type Called_Identifier_Span Evaluation_Identifier_Span

// Called_Identifier_Span_Invariants composes both name boundaries.
func Called_Identifier_Span_Invariants(
  value Called_Identifier_Span, namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Start),
    uint16(SOURCE_SIZE_MINIMUM),
    uint16(CALLED_IDENTIFIER_START_MAXIMUM),
  ).
    Range_Holed_Uint16(
    uint16(value.End),
    uint16(SOURCE_SIZE_MINIMUM),
    uint16(CALLED_IDENTIFIER_END_MAXIMUM),
    uint16(CALLED_IDENTIFIER_END_HOLE),
    uint16(CALLED_IDENTIFIER_END_HOLE),
    uint16(CALLED_IDENTIFIER_END_HOLE),
  ).
    Ensure()
}

// Called_Evaluation_Nodes keeps that name span beside the chain the call retains.
type Called_Evaluation_Nodes Evaluation_Nodes

// Called_Evaluation_Nodes_Invariants composes the name span beside the chain.
func Called_Evaluation_Nodes_Invariants(
  value Called_Evaluation_Nodes, namespace aver.Namespace,
) {
  Called_Identifier_Span_Invariants(Called_Identifier_Span(value.Identifier), namespace)
  Evaluation_Chain_Span_Invariants(&value.Chain, namespace)
}

// RUNNING_LITERAL_COUNT_MAXIMUM is the whole store one malformed literal fills.
const RUNNING_LITERAL_COUNT_MAXIMUM = LITERAL_DECODED_SIZE_MAXIMUM

// Running_Literal_Count is the decoded cursor one node execution reads.
type Running_Literal_Count Workspace_Literal_Count_Storage

// Running_Literal_Count_Invariants leaves the action that reads the decoded store.
func Running_Literal_Count_Invariants(
  value Running_Literal_Count, namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(bytes.SLICE_SIZE_MINIMUM),
    uint16(RUNNING_LITERAL_COUNT_MAXIMUM),
  ).
    Ensure()
}

// Running_Frame_Count is the traversal cursor one node execution reads.
type Running_Frame_Count Workspace_Frame_Count_Storage

// Running_Frame_Count_Invariants leaves the frame that run still holds.
func Running_Frame_Count_Invariants(value Running_Frame_Count, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(FRAME_COUNT_INCREMENT),
    uint16(RETAINED_FRAME_COUNT_MAXIMUM),
  ).
    Ensure()
}

// Running_Workspace is execution storage one node execution reads.
type Running_Workspace Workspace

// Running_Workspace_Invariants leaves the frame that run still holds.
func Running_Workspace_Invariants(value Running_Workspace, namespace aver.Namespace) {
  aver.Always(
    len(value.Output) == OUTPUT_SIZE_MAXIMUM,
    "One running node holds complete output capacity.",
  )
  aver.Always(
    len(value.Literals) == QUOTED_DECODED_SIZE_MAXIMUM,
    "One running node holds complete decoded-literal capacity.",
  )
  Running_Literal_Count_Invariants(Running_Literal_Count(value.Literal_Count),
    namespace)
  aver.Always(
    len(value.Literal_Counts) == NODE_COUNT_MAXIMUM,
    "One running node holds one decoded length for each node.",
  )
  aver.Always(
    len(value.Literal_Offsets) == NODE_COUNT_MAXIMUM,
    "One running node holds one decoded offset for each node.",
  )
  aver.Always(
    len(value.Literal_Decoded) == NODE_COUNT_MAXIMUM,
    "One running node holds one decoded flag for each node.",
  )
  aver.Always(
    len(value.Name_Left) == QUOTED_DECODED_SIZE_MAXIMUM,
    "One running node holds complete first-name capacity.",
  )
  aver.Always(
    len(value.Name_Right) == QUOTED_DECODED_SIZE_MAXIMUM,
    "One running node holds complete second-name capacity.",
  )
  aver.Always(
    len(value.Number) == NUMBER_SIZE_MAXIMUM,
    "One running node holds complete numeric-text capacity.",
  )
  aver.Always(
    len(value.Generated) == GENERATED_SIZE_MAXIMUM,
    "One running node holds complete derived-text capacity.",
  )
  Workspace_Generated_Count_Storage_Invariants(&value.Generated_Count, namespace)
  aver.Always(
    len(value.Frames) == FRAME_COUNT_MAXIMUM,
    "One running node holds complete frame capacity.",
  )
  Running_Frame_Count_Invariants(Running_Frame_Count(value.Frame_Count), namespace)
  aver.Always(
    len(value.Variables) == VARIABLE_COUNT_MAXIMUM,
    "One running node holds complete binding capacity.",
  )
  Workspace_Variable_Count_Storage_Invariants(&value.Variable_Count, namespace)
  Workspace_Scope_Base_Storage_Invariants(&value.Scope_Base, namespace)
  aver.Always(
    len(value.Arguments) == ARGUMENT_COUNT_MAXIMUM,
    "One running node holds complete argument capacity.",
  )
  aver.Always(
    len(value.Validation) == FRAME_COUNT_MAXIMUM,
    "One running node holds complete validation-frame capacity.",
  )
  aver.Always(
    len(value.Evaluations) == FRAME_COUNT_MAXIMUM,
    "One running node holds complete evaluation-frame capacity.",
  )
  aver.Always(
    len(value.Format) == FRAME_COUNT_MAXIMUM,
    "One running node holds complete format-frame capacity.",
  )
  Workspace_Diagnostic_Storage_Invariants(&value.Diagnostic, namespace)
}

// Running_Workspace_Pointer keeps that storage mutable for its caller.
type Running_Workspace_Pointer *Running_Workspace

// Running_Workspace_Pointer_Invariants composes present execution storage.
func Running_Workspace_Pointer_Invariants(
  value Running_Workspace_Pointer,
  namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Running_Workspace_Invariants(*value, namespace)
}

// COMMENT_ACTION_SIZE_MINIMUM is the shortest one-byte-delimited comment.
const COMMENT_ACTION_SIZE_MINIMUM = ACTION_BOUNDARY_SIZE_MINIMUM + len(`/**/`)

// DENSE_COMMENT_NODE_COUNT_MAXIMUM packs actions beside one shortest comment.
const DENSE_COMMENT_NODE_COUNT_MAXIMUM = ROOT_NODE_COUNT +
  (SOURCE_SIZE_MAXIMUM-COMMENT_ACTION_SIZE_MINIMUM)/
    SIMPLE_ACTION_SIZE_MINIMUM*
    SIMPLE_ACTION_NODE_COUNT

// Dense_Comment_Node_Count carries an arena count one dense source leaves beside a comment.
type Dense_Comment_Node_Count Node_Count_Allocated

// Dense_Comment_Node_Count_Invariants bounds the count by the comment beside it.
func Dense_Comment_Node_Count_Invariants(
  value Dense_Comment_Node_Count, namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(ROOT_NODE_COUNT),
    uint16(DENSE_COMMENT_NODE_COUNT_MAXIMUM),
  ).
    Ensure()
}

// DENSE_BRANCH_NODE_COUNT_MAXIMUM is the arena length once that branch is allocated.
const DENSE_BRANCH_NODE_COUNT_MAXIMUM = BRANCH_ALLOCATION_LINK_MAXIMUM

// Dense_Branch_Node_Count carries an arena count one dense source leaves beside a branch.
type Dense_Branch_Node_Count Node_Count_Allocated

// Dense_Branch_Node_Count_Invariants bounds the count by the branch beside it.
func Dense_Branch_Node_Count_Invariants(
  value Dense_Branch_Node_Count, namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(ROOT_NODE_COUNT),
    uint16(DENSE_BRANCH_NODE_COUNT_MAXIMUM),
  ).
    Ensure()
}

// DENSE_BODY_NODE_COUNT_MAXIMUM is the arena length one unclosed opener leaves.
const DENSE_BODY_NODE_COUNT_MAXIMUM = CONTROL_OPENED_COUNT_MAXIMUM

// Dense_Body_Node_Count carries the count one source leaves beside an open body.
type Dense_Body_Node_Count Node_Count_Allocated

// Dense_Body_Node_Count_Invariants bounds the count by one open branch body beside it.
func Dense_Body_Node_Count_Invariants(value Dense_Body_Node_Count, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(ROOT_NODE_COUNT),
    uint16(DENSE_BODY_NODE_COUNT_MAXIMUM),
  ).
    Ensure()
}

// DENSE_ALTERNATE_NODE_COUNT_MINIMUM counts the root, one branch, and its open body.
const DENSE_ALTERNATE_NODE_COUNT_MINIMUM = ROOT_NODE_COUNT +
  CONTROL_BRANCH_NODE_COUNT +
  NODE_COUNT_INCREMENT

// DENSE_ALTERNATE_NODE_COUNT_MAXIMUM adds the alternate list past that branch's body.
const DENSE_ALTERNATE_NODE_COUNT_MAXIMUM = UNCLOSED_ELSE_COUNT_MAXIMUM + NODE_COUNT_INCREMENT

// Dense_Alternate_Node_Count carries the count one dense source leaves beside an arm.
type Dense_Alternate_Node_Count Node_Count_Allocated

// Dense_Alternate_Node_Count_Invariants bounds the count by one alternate arm beside it.
func Dense_Alternate_Node_Count_Invariants(
  value Dense_Alternate_Node_Count,
  namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(DENSE_ALTERNATE_NODE_COUNT_MINIMUM),
    uint16(DENSE_ALTERNATE_NODE_COUNT_MAXIMUM),
  ).
    Ensure()
}

// DENSE_FLOW_NODE_COUNT_MAXIMUM is the count one stray flow clause leaves as it read it.
const DENSE_FLOW_NODE_COUNT_MAXIMUM = CONTROL_FLOW_COUNT_MAXIMUM

// Dense_Flow_Node_Count carries the count one dense source leaves beside a flow clause.
type Dense_Flow_Node_Count Node_Count_Allocated

// Dense_Flow_Node_Count_Invariants bounds the count by one flow clause inside a branch beside it.
func Dense_Flow_Node_Count_Invariants(value Dense_Flow_Node_Count, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(ROOT_NODE_COUNT),
    uint16(DENSE_FLOW_NODE_COUNT_MAXIMUM),
  ).
    Ensure()
}

// Escaped_Quoted_Position is where one escape stops inside a quoted literal.
type Escaped_Quoted_Position Quoted_Position_Validated

// Escaped_Quoted_Position_Invariants leaves the closing quote above that stop.
func Escaped_Quoted_Position_Invariants(
  value Escaped_Quoted_Position, namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(CONSTANT_END_MINIMUM),
    uint16(QUOTED_ESCAPE_POSITION_MAXIMUM),
  ).
    Ensure()
}

// ESCAPE_GENERATED_COUNT_MAXIMUM leaves the three bytes one escape writes above it.
const ESCAPE_GENERATED_COUNT_MAXIMUM = GENERATED_SIZE_MAXIMUM -
  utf8.CHARACTER_SIZE_MINIMUM -
  utf8.CHARACTER_SIZE_MINIMUM -
  utf8.CHARACTER_SIZE_MINIMUM

// Escape_Generated_Count is the generated cursor one escape writes from.
type Escape_Generated_Count Written_Generated_Count

// Escape_Generated_Count_Invariants keeps that escape inside generated storage.
func Escape_Generated_Count_Invariants(
  value Escape_Generated_Count, namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(utf8.CHARACTER_SIZE_MINIMUM),
    uint16(ESCAPE_GENERATED_COUNT_MAXIMUM),
  ).
    Ensure()
}

// Escape_Generated_Count_Pointer keeps that cursor mutable for its caller.
type Escape_Generated_Count_Pointer *Escape_Generated_Count

// Escape_Generated_Count_Pointer_Invariants composes one present cursor.
func Escape_Generated_Count_Pointer_Invariants(
  value Escape_Generated_Count_Pointer,
  namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Escape_Generated_Count_Invariants(*value, namespace)
}

// APPEND_TEXT_LIST_MAXIMUM is the body list one unclosed opener's text joins.
const APPEND_TEXT_LIST_MAXIMUM = TEXT_BODY_LIST_MAXIMUM

// TEMPLATE_INVOCATION_ARGUMENT_MAXIMUM is the latest invocation node one source holds.
const TEMPLATE_INVOCATION_ARGUMENT_MAXIMUM = INVOKING_NODE_COUNT_MAXIMUM

// APPEND_TEMPLATE_INVOCATION_COUNT_MAXIMUM stands just below that invocation node.
const APPEND_TEMPLATE_INVOCATION_COUNT_MAXIMUM = INVOKING_NODE_COUNT_MAXIMUM -
  NODE_COUNT_INCREMENT

// INVOCATION_LIST_LAST_CHILD_MAXIMUM is the latest shortest action packed below that
// invocation.
const INVOCATION_LIST_LAST_CHILD_MAXIMUM = APPEND_TEMPLATE_INVOCATION_COUNT_MAXIMUM -
  SIMPLE_ACTION_NODE_COUNT +
  NODE_COUNT_INCREMENT

// BRANCH_ELSE_TAIL_SIZE is the shortest branch whose body an alternate list follows.
const BRANCH_ELSE_TAIL_SIZE = CONTROL_ACTION_SIZE_MINIMUM +
  ACTION_BOUNDARY_SIZE_MINIMUM +
  len(KEYWORD_ELSE)

// BODY_LIST_SIBLING_MAXIMUM is the alternate list past the densest branch body.
const BODY_LIST_SIBLING_MAXIMUM = CONTROL_ELSE_COUNT_MAXIMUM + NODE_COUNT_INCREMENT

// NAMES_EQUAL_REFERENCE_MAXIMUM leaves those definitions above that reference.
const NAMES_EQUAL_REFERENCE_MAXIMUM = LATER_TEMPLATE_REFERENCE_MAXIMUM

// RANGED_DEFINITION_TAIL_SIZE is the shortest range followed by the shortest definition.
const RANGED_DEFINITION_TAIL_SIZE = CONTROL_ACTION_SIZE_MINIMUM +
  len(KEYWORD_RANGE) -
  len(KEYWORD_IF) +
  TEMPLATE_DEFINITION_SIZE_MINIMUM

// RANGE_BEGIN_FIRST_TEMPLATE_MAXIMUM is the root-relative offset past the densest actions, one
// text node, and that range's node, pipeline, command, term, and body list.
const RANGE_BEGIN_FIRST_TEMPLATE_MAXIMUM = (SOURCE_SIZE_MAXIMUM-
  RANGED_DEFINITION_TAIL_SIZE)/
  SIMPLE_ACTION_SIZE_MINIMUM*
  SIMPLE_ACTION_NODE_COUNT +
  ((SOURCE_SIZE_MAXIMUM-RANGED_DEFINITION_TAIL_SIZE)%SIMPLE_ACTION_SIZE_MINIMUM+
    SIMPLE_ACTION_SIZE_MINIMUM-
    SOURCE_POSITION_INCREMENT)/
    SIMPLE_ACTION_SIZE_MINIMUM +
  SIMPLE_ACTION_NODE_COUNT +
  NODE_COUNT_INCREMENT +
  NODE_COUNT_INCREMENT

// FORMATTED_INTEGER_TEXT_SIZE_MAXIMUM is the decimal digit count of the widest integer.
const FORMATTED_INTEGER_TEXT_SIZE_MAXIMUM = len("18446744073709551615")

// INVOCATION_TOKEN_START_MAXIMUM leaves the keyword and its delimiter above the start.
const INVOCATION_TOKEN_START_MAXIMUM = SOURCE_SIZE_MAXIMUM -
  (len(KEYWORD_TEMPLATE) +
    utf8.CHARACTER_SIZE_MINIMUM)

// BRANCH_BODY_LINK_MAXIMUM is the body list one unclosed opener's branch ends with.
const BRANCH_BODY_LINK_MAXIMUM = CONTROL_OPENED_COUNT_MAXIMUM

// BRANCH_PIPE_REFERENCE_MAXIMUM is the pipeline that follows that opener's branch.
const BRANCH_PIPE_REFERENCE_MAXIMUM = BRANCH_ALLOCATION_LINK_MAXIMUM + PIPELINE_NODE_COUNT

// BRANCH_ALLOCATION_LINK_MAXIMUM is the branch one unclosed opener allocates behind the densest
// actions.
const BRANCH_ALLOCATION_LINK_MAXIMUM = CONTROL_OPENER_COUNT_MAXIMUM + NODE_COUNT_INCREMENT

// CONTROL_BRANCH_ALLOCATE_COUNT_MAXIMUM is the arena length before that branch.
const CONTROL_BRANCH_ALLOCATE_COUNT_MAXIMUM = BRANCH_ALLOCATION_LINK_MAXIMUM -
  NODE_COUNT_INCREMENT

// TEXT_BRANCH_ELSE_TAIL_SIZE is the shortest branch whose one-byte text body an alternate list
// follows.
const TEXT_BRANCH_ELSE_TAIL_SIZE = BRANCH_ELSE_TAIL_SIZE + utf8.CHARACTER_SIZE_MINIMUM

// CONTROL_ACTION_COUNT_MAXIMUM ends with the text node before one stray shortest keyword action,
// which a hostile source may leave unmatched.
const CONTROL_ACTION_COUNT_MAXIMUM = CONTROL_LIST_LAST_CHILD_MAXIMUM

// CONTROL_ELSE_COUNT_MAXIMUM packs the densest actions and one text node before that branch, then
// its node, pipeline, command, term, body list, and body text.
const CONTROL_ELSE_COUNT_MAXIMUM = ROOT_NODE_COUNT +
  (SOURCE_SIZE_MAXIMUM-TEXT_BRANCH_ELSE_TAIL_SIZE)/
    SIMPLE_ACTION_SIZE_MINIMUM*
    SIMPLE_ACTION_NODE_COUNT +
  ((SOURCE_SIZE_MAXIMUM-TEXT_BRANCH_ELSE_TAIL_SIZE)%SIMPLE_ACTION_SIZE_MINIMUM+
    SIMPLE_ACTION_SIZE_MINIMUM-
    SOURCE_POSITION_INCREMENT)/
    SIMPLE_ACTION_SIZE_MINIMUM +
  SIMPLE_ACTION_NODE_COUNT +
  NODE_COUNT_INCREMENT +
  NODE_COUNT_INCREMENT

// TEMPLATE_BLOCK_INVOCATION_COUNT_MAXIMUM leaves four branches above that count.
const TEMPLATE_BLOCK_INVOCATION_COUNT_MAXIMUM = NODE_REFERENCE_MAXIMUM -
  SIMPLE_ACTION_CONTAINER_NODE_COUNT -
  (CONTROL_BRANCH_NODE_COUNT +
    CONTROL_BRANCH_NODE_COUNT +
    CONTROL_BRANCH_NODE_COUNT +
    CONTROL_BRANCH_NODE_COUNT)

// TEMPLATE_BLOCK_INVOCATION_REFERENCE_MAXIMUM leaves those branches and one action above it.
const TEMPLATE_BLOCK_INVOCATION_REFERENCE_MAXIMUM = NODE_REFERENCE_MAXIMUM -
  SIMPLE_ACTION_CONTAINER_NODE_COUNT -
  (CONTROL_BRANCH_NODE_COUNT +
    CONTROL_BRANCH_NODE_COUNT +
    CONTROL_BRANCH_NODE_COUNT +
    CONTROL_BRANCH_NODE_COUNT)

// FLOW_ACTION_SIZE is the action one break keyword spells alone.
const FLOW_ACTION_SIZE = ACTION_BOUNDARY_SIZE_MINIMUM + len(KEYWORD_BREAK)

// CONTROL_FLOW_COUNT_MAXIMUM packs the densest actions and one text node before one break a
// hostile source leaves outside any range.
const CONTROL_FLOW_COUNT_MAXIMUM = ROOT_NODE_COUNT +
  (SOURCE_SIZE_MAXIMUM-FLOW_ACTION_SIZE)/SIMPLE_ACTION_SIZE_MINIMUM*SIMPLE_ACTION_NODE_COUNT +
  ((SOURCE_SIZE_MAXIMUM-FLOW_ACTION_SIZE)%SIMPLE_ACTION_SIZE_MINIMUM+
    SIMPLE_ACTION_SIZE_MINIMUM-
    SOURCE_POSITION_INCREMENT)/
    SIMPLE_ACTION_SIZE_MINIMUM

// TEMPLATE_KEYWORD_ACTION_SIZE_MINIMUM is the shortest action one template keyword opens.
const TEMPLATE_KEYWORD_ACTION_SIZE_MINIMUM = ACTION_BOUNDARY_SIZE_MINIMUM + len(KEYWORD_BLOCK)

// TEMPLATE_OPEN_COUNT_MAXIMUM packs the densest actions and one text node before one stray
// template keyword, whose name a hostile source may omit.
const TEMPLATE_OPEN_COUNT_MAXIMUM = ROOT_NODE_COUNT +
  (SOURCE_SIZE_MAXIMUM-TEMPLATE_KEYWORD_ACTION_SIZE_MINIMUM)/
    SIMPLE_ACTION_SIZE_MINIMUM*
    SIMPLE_ACTION_NODE_COUNT +
  ((SOURCE_SIZE_MAXIMUM-TEMPLATE_KEYWORD_ACTION_SIZE_MINIMUM)%SIMPLE_ACTION_SIZE_MINIMUM+
    SIMPLE_ACTION_SIZE_MINIMUM-
    SOURCE_POSITION_INCREMENT)/
    SIMPLE_ACTION_SIZE_MINIMUM

// Opened_Template_Node_Count is the arena length one template keyword leaves.
type Opened_Template_Node_Count Node_Count_Allocated

// Opened_Template_Node_Count_Invariants leaves a stray keyword's count unchanged.
func Opened_Template_Node_Count_Invariants(
  value Opened_Template_Node_Count,
  namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(ROOT_NODE_COUNT),
    uint16(TEMPLATE_OPEN_COUNT_MAXIMUM),
  ).
    Ensure()
}

// Else_Node_Count is the arena length one else clause leaves.
type Else_Node_Count Node_Count_Allocated

// Else_Node_Count_Invariants leaves a stray alternate's count as it read it.
func Else_Node_Count_Invariants(value Else_Node_Count, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(ROOT_NODE_COUNT),
    uint16(STRAY_ELSE_COUNT_MAXIMUM),
  ).
    Ensure()
}

// RANGE_BEGIN_NODE_COUNT_MAXIMUM leaves two branches and one action above the range.
const RANGE_BEGIN_NODE_COUNT_MAXIMUM = NODE_REFERENCE_MAXIMUM -
  SIMPLE_ACTION_NODE_COUNT -
  (CONTROL_BRANCH_NODE_COUNT +
    CONTROL_BRANCH_NODE_COUNT)

// CHAINED_COMMAND_NODE_COUNT_MAXIMUM leaves the node one chain adds above it.
const CHAINED_COMMAND_NODE_COUNT_MAXIMUM = ARGUMENT_NODE_COUNT_LIMIT - NODE_COUNT_INCREMENT

// Chained_Command_Node_Count is the arena length one chain reads before it adds.
type Chained_Command_Node_Count Dense_Command_Node_Count

// Chained_Command_Node_Count_Invariants keeps that node inside the arena.
func Chained_Command_Node_Count_Invariants(
  value Chained_Command_Node_Count,
  namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(COMMAND_NODE_COUNT_MINIMUM),
    uint16(CHAINED_COMMAND_NODE_COUNT_MAXIMUM),
  ).
    Ensure()
}

// WALKED_NODE_END_MINIMUM stands past the one byte a walked node covers.
const WALKED_NODE_END_MINIMUM = SOURCE_SIZE_MINIMUM + utf8.CHARACTER_SIZE_MINIMUM

// Referenced_Node is one syntax record one reference names.
type Referenced_Node Node

// Referenced_Node_Invariants keeps that byte and that pipeline inside the record.
func Referenced_Node_Invariants(value Referenced_Node, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint8(
    uint8(value.Kind),
    uint8(NODE_KIND_MINIMUM),
    uint8(NODE_KIND_MAXIMUM),
  ).
    Range_Uint16(
    uint16(value.Start),
    uint16(SOURCE_SIZE_MINIMUM),
    uint16(SOURCE_SIZE_MAXIMUM),
  ).
    Range_Uint16(
    uint16(value.End),
    uint16(WALKED_NODE_END_MINIMUM),
    uint16(SOURCE_SIZE_MAXIMUM),
  ).
    Range_Uint16(
    uint16(value.Value_Start),
    uint16(SOURCE_SIZE_MINIMUM),
    uint16(NODE_VALUE_START_MAXIMUM),
  ).
    Range_Uint16(
    uint16(value.Value_End),
    uint16(SOURCE_SIZE_MINIMUM),
    uint16(SOURCE_SIZE_MAXIMUM),
  ).
    Range_Uint16(
    uint16(value.First_Child),
    uint16(NODE_REFERENCE_MINIMUM),
    uint16(NODE_REFERENCE_MAXIMUM),
  ).
    Range_Uint16(
    uint16(value.Next_Sibling),
    uint16(NODE_REFERENCE_MINIMUM),
    uint16(NODE_REFERENCE_MAXIMUM),
  ).
    Ensure()
}

// Referenced_Node_Stored keeps one referenced record behind its arena pointer.
type Referenced_Node_Stored *Referenced_Node

// Referenced_Node_Stored_Invariants composes one present record.
func Referenced_Node_Stored_Invariants(value Referenced_Node_Stored, namespace aver.Namespace) {
  if value == nil {
    return
  }
  Referenced_Node_Invariants(*value, namespace)
}

// Referenced_Node_Validated carries one referenced record out of the arena.
type Referenced_Node_Validated struct {
  // Value keeps arena proof attached to the record.
  Value Referenced_Node_Stored
}

// Referenced_Node_Validated_Invariants checks transport ownership of that record.
func Referenced_Node_Validated_Invariants(
  value *Referenced_Node_Validated,
  namespace aver.Namespace,
) {
  Referenced_Node_Stored_Invariants(value.Value, namespace)
}

// Walked_Node is one syntax record a walk reads.
type Walked_Node Node

// Walked_Node_Invariants keeps that byte and that pipeline inside the record.
func Walked_Node_Invariants(value Walked_Node, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Holed_Uint8(
    uint8(value.Kind),
    uint8(NODE_KIND_MINIMUM),
    uint8(NODE_KIND_MAXIMUM),
    uint8(NODE_BOOLEAN),
    uint8(NODE_BOOLEAN),
    uint8(NODE_BOOLEAN),
  ).
    Range_Uint16(
    uint16(value.Start),
    uint16(SOURCE_SIZE_MINIMUM),
    uint16(SOURCE_SIZE_MAXIMUM),
  ).
    Range_Uint16(
    uint16(value.End),
    uint16(WALKED_NODE_END_MINIMUM),
    uint16(SOURCE_SIZE_MAXIMUM),
  ).
    Range_Uint16(
    uint16(value.Value_Start),
    uint16(SOURCE_SIZE_MINIMUM),
    uint16(NODE_VALUE_START_MAXIMUM),
  ).
    Range_Uint16(
    uint16(value.Value_End),
    uint16(SOURCE_SIZE_MINIMUM),
    uint16(SOURCE_SIZE_MAXIMUM),
  ).
    Range_Uint16(
    uint16(value.First_Child),
    uint16(NODE_REFERENCE_MINIMUM),
    uint16(COMMAND_TAIL_REFERENCE_MAXIMUM),
  ).
    Range_Uint16(
    uint16(value.Next_Sibling),
    uint16(NODE_REFERENCE_MINIMUM),
    uint16(NODE_REFERENCE_MAXIMUM),
  ).
    Ensure()
}

// Walked_Node_Stored keeps one walked record behind its arena pointer.
type Walked_Node_Stored *Walked_Node

// Walked_Node_Stored_Invariants composes one present record.
func Walked_Node_Stored_Invariants(value Walked_Node_Stored, namespace aver.Namespace) {
  if value == nil {
    return
  }
  Walked_Node_Invariants(*value, namespace)
}

// Walked_Node_Validated carries one walked record through execution.
type Walked_Node_Validated struct {
  // Value keeps arena proof attached to the record.
  Value Walked_Node_Stored
}

// Walked_Node_Validated_Invariants checks transport ownership of that record.
func Walked_Node_Validated_Invariants(
  value *Walked_Node_Validated, namespace aver.Namespace,
) {
  Walked_Node_Stored_Invariants(value.Value, namespace)
}

// WALKED_NODE_COUNT_MINIMUM counts the root and the one node a walk reads under it.
const WALKED_NODE_COUNT_MINIMUM = ROOT_NODE_COUNT + NODE_COUNT_INCREMENT

// Walked_Program_Document is arena metadata of one program a walk reads nodes from.
type Walked_Program_Document Program_Validated_Document

// Walked_Program_Document_Invariants keeps that node inside the arena.
func Walked_Program_Document_Invariants(
  value Walked_Program_Document, namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Node_Count),
    uint16(WALKED_NODE_COUNT_MINIMUM),
    uint16(
      NODE_COUNT_MAXIMUM,
    ),
  ).
    Range_Uint16(
    uint16(value.First_Template),
    uint16(
      DOCUMENT_FIRST_TEMPLATE_MINIMUM,
    ),
    uint16(DOCUMENT_FIRST_TEMPLATE_MAXIMUM),
  ).
    Range_Uint16(
    uint16(value.Template_Count),
    uint16(TEMPLATE_COUNT_MINIMUM),
    uint16(
      EXECUTED_TEMPLATE_COUNT_MAXIMUM,
    ),
  ).
    Ensure()
  aver.Always(
    uint16(value.Root) == uint16(ROOT_NODE_COUNT),
    "Walked program metadata keeps the root in the first syntax arena slot.",
  )
}

// WALKED_SOURCE_SIZE_MINIMUM counts the one byte a walked program's node covers.
const WALKED_SOURCE_SIZE_MINIMUM = SOURCE_SIZE_MINIMUM + utf8.CHARACTER_SIZE_MINIMUM

// Walked_Source is the source one walked program's nodes point into.
type Walked_Source Source

// Walked_Source_Invariants composes the storage one walked program points into.
func Walked_Source_Invariants(value Walked_Source, namespace aver.Namespace) {
  Walked_Source_Storage_Invariants(Walked_Source_Storage(value.Data), namespace)
}

// Walked_Source_Storage is the storage one walked program's nodes point into.
type Walked_Source_Storage Source_Storage

// Walked_Source_Storage_Invariants keeps one byte inside that storage.
func Walked_Source_Storage_Invariants(
  value Walked_Source_Storage, namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Int(
    len(value.Value),
    WALKED_SOURCE_SIZE_MINIMUM,
    SOURCE_SIZE_MAXIMUM,
  ).
    Ensure()
}

// Walked_Program is one checked program whose nodes a walk reads.
type Walked_Program Program_Validated

// Walked_Program_Invariants composes that program's source and metadata.
func Walked_Program_Invariants(value Walked_Program, namespace aver.Namespace) {
  Walked_Source_Invariants(Walked_Source(value.Source), namespace)
  Walked_Program_Document_Invariants(Walked_Program_Document(value.Document), namespace)
  Syntax_Storage_Invariants(&value.Syntax, namespace)
}

// Checked_Document is arena metadata of one program that passed validation.
type Checked_Document Document

// Checked_Document_Invariants keeps the root inside that arena.
func Checked_Document_Invariants(value Checked_Document, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Node_Count),
    uint16(ROOT_NODE_COUNT),
    uint16(NODE_COUNT_MAXIMUM),
  ).
    Range_Uint16(
    uint16(value.First_Template),
    uint16(DOCUMENT_FIRST_TEMPLATE_MINIMUM),
    uint16(DOCUMENT_FIRST_TEMPLATE_MAXIMUM),
  ).
    Range_Uint16(
    uint16(value.Template_Count),
    uint16(TEMPLATE_COUNT_MINIMUM),
    uint16(EXECUTED_TEMPLATE_COUNT_MAXIMUM),
  ).
    Ensure()
  aver.Always(
    uint16(value.Root) == uint16(ROOT_NODE_COUNT),
    "One checked program keeps its root at the first node.",
  )
  aver.Always(
    !(value.Template_Count > Template_Count(value.Node_Count)),
    "One checked program keeps its template chain inside node storage.",
  )
}

// Checked_Program is one program that passed validation.
type Checked_Program Program

// Checked_Program_Invariants composes that program's source and metadata.
func Checked_Program_Invariants(value Checked_Program, namespace aver.Namespace) {
  Source_Invariants(&value.Source, namespace)
  Checked_Document_Invariants(Checked_Document(value.Document), namespace)
  Syntax_Storage_Invariants(&value.Syntax, namespace)
}

// Read_Token_Boundaries are the boundaries one read token covers.
type Read_Token_Boundaries Token_Stored

// Read_Token_Boundaries_Invariants keeps one byte inside that token.
func Read_Token_Boundaries_Invariants(
  value Read_Token_Boundaries, namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint8(
    uint8(value.Kind),
    uint8(TOKEN_KIND_MINIMUM),
    uint8(TOKEN_KIND_MAXIMUM),
  ).
    Range_Uint16(
    uint16(value.Start),
    uint16(TOKEN_BOUNDARY_MINIMUM),
    uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM),
  ).
    Range_Uint16(
    uint16(value.End),
    uint16(READ_CURSOR_BOUNDARY_MINIMUM),
    uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM),
  ).
    Ensure()
}

// Read_Cursor_Token is the token one lexer already read.
type Read_Cursor_Token Cursor_Token_Stored

// Read_Cursor_Token_Invariants composes that token's boundaries.
func Read_Cursor_Token_Invariants(value Read_Cursor_Token, namespace aver.Namespace) {
  Read_Token_Boundaries_Invariants(Read_Token_Boundaries(value.Value), namespace)
}

// Read_Token_Cursor is one action lexer that already read a token.
type Read_Token_Cursor Token_Cursor

// Read_Token_Cursor_Invariants composes that lexer's storage.
func Read_Token_Cursor_Invariants(value Read_Token_Cursor, namespace aver.Namespace) {
  Read_Cursor_Position_Invariants(Read_Cursor_Position(value.Position), namespace)
  Read_Cursor_End_Invariants(Read_Cursor_End(value.End), namespace)
  Read_Cursor_Token_Invariants(Read_Cursor_Token(value.Token), namespace)
  Cursor_Look_Ahead_Stored_Invariants((*Cursor_Look_Ahead_Stored)(&value.Looked), namespace)
  Cursor_Number_Stored_Invariants((*Cursor_Number_Stored)(&value.Number), namespace)
  Function_Function_Storage_Invariants(&value.Function, namespace)
  Cursor_Last_Start_Stored_Invariants((*Cursor_Last_Start_Stored)(&value.Last_Start), namespace)
  Cursor_Has_Last_Stored_Invariants((*Cursor_Has_Last_Stored)(&value.Has_Last), namespace)
}

// Read_Token_Cursor_Pointer keeps mutation of that lexer typed.
type Read_Token_Cursor_Pointer *Read_Token_Cursor

// Read_Token_Cursor_Pointer_Invariants composes present lexer state.
func Read_Token_Cursor_Pointer_Invariants(
  value Read_Token_Cursor_Pointer,
  namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Read_Token_Cursor_Invariants(*value, namespace)
}

// PUSHED_CHAIN_NODE_COUNT_MINIMUM counts the command one pushed chain evaluates.
const PUSHED_CHAIN_NODE_COUNT_MINIMUM = CHAINED_PROGRAM_NODE_COUNT_MINIMUM +
  COMMAND_NODE_COUNT

// Pushed_Chain_Document is arena metadata of a program one pushed chain walks.
type Pushed_Chain_Document Chained_Program_Document

// Pushed_Chain_Document_Invariants keeps that command inside the arena.
func Pushed_Chain_Document_Invariants(value Pushed_Chain_Document, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Node_Count),
    uint16(PUSHED_CHAIN_NODE_COUNT_MINIMUM),
    uint16(CHAINED_PROGRAM_NODE_COUNT_MAXIMUM),
  ).
    Range_Uint16(
    uint16(value.First_Template),
    uint16(DOCUMENT_FIRST_TEMPLATE_MINIMUM),
    uint16(CHAINED_FIRST_TEMPLATE_MAXIMUM),
  ).
    Range_Uint16(
    uint16(value.Template_Count),
    uint16(TEMPLATE_COUNT_MINIMUM),
    uint16(EXECUTED_TEMPLATE_COUNT_MAXIMUM),
  ).
    Ensure()
  aver.Always(
    uint16(value.Root) == uint16(ROOT_NODE_COUNT),
    "One pushed chain keeps the root in the first syntax arena slot.",
  )
}

// Pushed_Chain_Program is one program a pushed chain frame walks.
type Pushed_Chain_Program Chain_Program

// Pushed_Chain_Program_Invariants composes that source with arena metadata.
func Pushed_Chain_Program_Invariants(value Pushed_Chain_Program, namespace aver.Namespace) {
  Chain_Program_Source_Invariants(Chain_Program_Source(value.Source), namespace)
  Pushed_Chain_Document_Invariants(Pushed_Chain_Document(value.Document), namespace)
  Syntax_Storage_Invariants(&value.Syntax, namespace)
}

// TYPED_GENERATED_COUNT_MAXIMUM leaves the two bytes one type name writes above it.
const TYPED_GENERATED_COUNT_MAXIMUM = GENERATED_SIZE_MAXIMUM -
  utf8.CHARACTER_SIZE_MINIMUM -
  utf8.CHARACTER_SIZE_MINIMUM

// Typed_Generated_Count is the generated cursor one type name writes from.
type Typed_Generated_Count Workspace_Generated_Count_Storage

// Typed_Generated_Count_Invariants keeps that name inside generated storage.
func Typed_Generated_Count_Invariants(value Typed_Generated_Count, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(bytes.SLICE_SIZE_MINIMUM),
    uint16(TYPED_GENERATED_COUNT_MAXIMUM),
  ).
    Ensure()
}

// Typed_Generated_Count_Pointer keeps that cursor mutable for its caller.
type Typed_Generated_Count_Pointer *Typed_Generated_Count

// Typed_Generated_Count_Pointer_Invariants composes one present cursor.
func Typed_Generated_Count_Pointer_Invariants(
  value Typed_Generated_Count_Pointer,
  namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Typed_Generated_Count_Invariants(*value, namespace)
}

// UNICODE_GENERATED_COUNT_MAXIMUM leaves the byte one spelled escape adds above it.
const UNICODE_GENERATED_COUNT_MAXIMUM = ESCAPE_GENERATED_COUNT_MAXIMUM -
  utf8.CHARACTER_SIZE_MINIMUM

// Unicode_Generated_Count is the generated cursor one spelled escape writes from.
type Unicode_Generated_Count Written_Generated_Count

// Unicode_Generated_Count_Invariants keeps that escape inside generated storage.
func Unicode_Generated_Count_Invariants(
  value Unicode_Generated_Count, namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(utf8.CHARACTER_SIZE_MINIMUM),
    uint16(UNICODE_GENERATED_COUNT_MAXIMUM),
  ).
    Ensure()
}

// Unicode_Generated_Count_Pointer keeps that cursor mutable for its caller.
type Unicode_Generated_Count_Pointer *Unicode_Generated_Count

// Unicode_Generated_Count_Pointer_Invariants composes one present cursor.
func Unicode_Generated_Count_Pointer_Invariants(
  value Unicode_Generated_Count_Pointer,
  namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Unicode_Generated_Count_Invariants(*value, namespace)
}

// COMMENT_SCAN_START_MAXIMUM leaves the two bytes one comment closes with above it.
const COMMENT_SCAN_START_MAXIMUM = ACTION_SCAN_START_MAXIMUM -
  utf8.CHARACTER_SIZE_MINIMUM -
  utf8.CHARACTER_SIZE_MINIMUM

// Comment_Scan_Span is the span one comment scan reads inside.
type Comment_Scan_Span Action_Scan_Span_Validated

// Comment_Scan_Span_Invariants keeps that closing pair inside the span.
func Comment_Scan_Span_Invariants(value Comment_Scan_Span, namespace aver.Namespace) {
  Comment_Scan_Boundaries_Invariants(Comment_Scan_Boundaries(value.Value), namespace)
}

// Comment_Scan_Boundaries are the boundaries one comment scan reads between.
type Comment_Scan_Boundaries Action_Scan_Span_Proof_Stored

// Comment_Scan_Boundaries_Invariants keeps that closing pair inside the boundaries.
func Comment_Scan_Boundaries_Invariants(
  value Comment_Scan_Boundaries, namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Start),
    uint16(ACTION_SCAN_START_MINIMUM),
    uint16(COMMENT_SCAN_START_MAXIMUM),
  ).
    Range_Uint16(
    uint16(value.Limit),
    uint16(ACTION_SCAN_LIMIT_MINIMUM),
    uint16(ACTION_SCAN_LIMIT_MAXIMUM),
  ).
    Ensure()
}

// Read_Variable_Count is the variable cursor one name read stands under.
type Read_Variable_Count Bound_Variable_Count

// Read_Variable_Count_Invariants keeps that variable inside the table.
func Read_Variable_Count_Invariants(value Read_Variable_Count, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(DECLARATION_COUNT_ONE),
    uint16(VARIABLE_COUNT_MAXIMUM),
  ).
    Ensure()
}

// Read_Variable_Count_Pointer keeps that cursor mutable for its caller.
type Read_Variable_Count_Pointer *Read_Variable_Count

// Read_Variable_Count_Pointer_Invariants composes one present cursor.
func Read_Variable_Count_Pointer_Invariants(
  value Read_Variable_Count_Pointer,
  namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Read_Variable_Count_Invariants(*value, namespace)
}

// Lookup_Scope_Base is the visibility cursor one name lookup reads from.
type Lookup_Scope_Base Workspace_Scope_Base_Storage

// Lookup_Scope_Base_Invariants keeps those two variables above the cursor.
func Lookup_Scope_Base_Invariants(value Lookup_Scope_Base, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(VARIABLE_COUNT_MINIMUM),
    uint16(VARIABLE_COUNT_MAXIMUM-VARIABLE_COUNT_INCREMENT),
  ).
    Ensure()
}

// Lookup_Scope_Base_Pointer keeps mutation of that cursor typed.
type Lookup_Scope_Base_Pointer *Lookup_Scope_Base

// Lookup_Scope_Base_Pointer_Invariants composes one present cursor.
func Lookup_Scope_Base_Pointer_Invariants(
  value Lookup_Scope_Base_Pointer,
  namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Lookup_Scope_Base_Invariants(*value, namespace)
}

// PIPELINE_TERM_LAST_START_MAXIMUM leaves one term byte and one delimiter above it.
const PIPELINE_TERM_LAST_START_MAXIMUM = ACTION_CONTENT_BOUNDARY_MAXIMUM -
  utf8.CHARACTER_SIZE_MINIMUM -
  utf8.CHARACTER_SIZE_MINIMUM

// Pipeline_Term_Last_Start is where one pipeline term's last token began.
type Pipeline_Term_Last_Start Cursor_Last_Start_Stored

// Pipeline_Term_Last_Start_Invariants keeps term and delimiter above that start.
func Pipeline_Term_Last_Start_Invariants(
  value Pipeline_Term_Last_Start, namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(SOURCE_SIZE_MINIMUM),
    uint16(PIPELINE_TERM_LAST_START_MAXIMUM),
  ).
    Ensure()
}

// DECODED_ESCAPE_POSITION_MINIMUM stands past one backslash and its escape letter.
const DECODED_ESCAPE_POSITION_MINIMUM = CONSTANT_END_MINIMUM +
  utf8.CHARACTER_SIZE_MINIMUM +
  utf8.CHARACTER_SIZE_MINIMUM

// Decoded_Escape_Position is where one decoded escape stops.
type Decoded_Escape_Position Quoted_Position_Validated

// Decoded_Escape_Position_Invariants keeps backslash and letter below that stop.
func Decoded_Escape_Position_Invariants(
  value Decoded_Escape_Position, namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(DECODED_ESCAPE_POSITION_MINIMUM),
    uint16(QUOTED_ESCAPE_POSITION_MAXIMUM),
  ).
    Ensure()
}

// DIGIT_ESCAPE_POSITION_MINIMUM stands past the first digit one escape spells.
const DIGIT_ESCAPE_POSITION_MINIMUM = DECODED_ESCAPE_POSITION_MINIMUM +
  utf8.CHARACTER_SIZE_MINIMUM

// Digit_Escape_Position is where one spelled escape stops.
type Digit_Escape_Position Quoted_Position_Validated

// Digit_Escape_Position_Invariants keeps that digit below the stop.
func Digit_Escape_Position_Invariants(value Digit_Escape_Position, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(DIGIT_ESCAPE_POSITION_MINIMUM),
    uint16(QUOTED_ESCAPE_POSITION_MAXIMUM),
  ).
    Ensure()
}

// Lexeme_Token is the token one lexer read while a lexeme remains after it.
type Lexeme_Token Token_Stored

// Lexeme_Token_Invariants reserves the lexeme byte the token stands before.
func Lexeme_Token_Invariants(value Lexeme_Token, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint8(
    uint8(value.Kind),
    uint8(TOKEN_KIND_MINIMUM),
    uint8(TOKEN_KIND_MAXIMUM),
  ).
    Range_Uint16(
    uint16(value.Start),
    uint16(TOKEN_BOUNDARY_MINIMUM),
    uint16(LEXEME_START_MAXIMUM),
  ).
    Range_Uint16(
    uint16(value.End),
    uint16(TOKEN_BOUNDARY_MINIMUM),
    uint16(LEXEME_START_MAXIMUM),
  ).
    Ensure()
}

// Lexeme_Token_Storage is that token inside its cursor field.
type Lexeme_Token_Storage Token_Storage

// Lexeme_Token_Storage_Invariants composes the token it holds.
func Lexeme_Token_Storage_Invariants(value Lexeme_Token_Storage, namespace aver.Namespace) {
  Lexeme_Token_Invariants(Lexeme_Token(value.Value), namespace)
}

// Staged_Argument_Count counts the arguments one nested call opens above.
type Staged_Argument_Count Argument_Count_Tracked

// Staged_Argument_Count_Invariants leaves the nested call room for its own argument.
func Staged_Argument_Count_Invariants(
  value Staged_Argument_Count, namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(ARGUMENT_COUNT_MINIMUM),
    uint16(ARGUMENT_BASE_MAXIMUM),
  ).
    Ensure()
}

// Delimited_Source_Cursor stands past the opening marker of one action.
type Delimited_Source_Cursor Source_Cursor_Validated

// Delimited_Source_Cursor_Invariants keeps that marker beneath the cursor.
func Delimited_Source_Cursor_Invariants(
  value Delimited_Source_Cursor, namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(CUSTOM_DELIMITER_SIZE_MINIMUM),
    uint16(SOURCE_SIZE_MAXIMUM),
  ).
    Ensure()
}

// Running_Frame_Cursor is the traversal cursor one frame push reads.
type Running_Frame_Cursor Bound_Frame_Count

// Running_Frame_Cursor_Invariants leaves the frame that push still takes.
func Running_Frame_Cursor_Invariants(value Running_Frame_Cursor, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(FRAME_COUNT_INCREMENT),
    uint16(RETAINED_FRAME_COUNT_MAXIMUM),
  ).
    Ensure()
}

// Running_Frame_Cursor_Pointer keeps that cursor mutable for its caller.
type Running_Frame_Cursor_Pointer *Running_Frame_Cursor

// Running_Frame_Cursor_Pointer_Invariants composes present cursor storage.
func Running_Frame_Cursor_Pointer_Invariants(
  value Running_Frame_Cursor_Pointer,
  namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Running_Frame_Cursor_Invariants(*value, namespace)
}

// APPENDED_CHILD_REFERENCE_MINIMUM names the first node one list can own.
const APPENDED_CHILD_REFERENCE_MINIMUM = ROOT_NODE_COUNT + NODE_COUNT_INCREMENT

// Appended_Child_Link is the node one append step links into its parent.
type Appended_Child_Link Node_Link_Proof_Stored

// Appended_Child_Link_Invariants keeps the root beneath that node.
func Appended_Child_Link_Invariants(value Appended_Child_Link, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value),
    uint16(APPENDED_CHILD_REFERENCE_MINIMUM),
    uint16(NODE_REFERENCE_MAXIMUM),
  ).
    Ensure()
}

// Appended_Child_Validated carries proof of that node.
type Appended_Child_Validated struct {
  // Value keeps arena proof attached to the link.
  Value Appended_Child_Link
}

// Appended_Child_Validated_Invariants composes the link it carries.
func Appended_Child_Validated_Invariants(
  value *Appended_Child_Validated,
  namespace aver.Namespace,
) {
  Appended_Child_Link_Invariants(value.Value, namespace)
}

// Allocated_Node_Link names one node the parser already put in the arena.
type Allocated_Node_Link Node_Link_Proof_Stored

// Allocated_Node_Link_Invariants keeps the root beneath that node.
func Allocated_Node_Link_Invariants(value Allocated_Node_Link, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value),
    uint16(APPENDED_CHILD_REFERENCE_MINIMUM),
    uint16(NODE_REFERENCE_MAXIMUM),
  ).
    Ensure()
}

// Allocated_Node_Validated carries proof of that node.
type Allocated_Node_Validated struct {
  // Value keeps arena proof attached to the link.
  Value Allocated_Node_Link
}

// Allocated_Node_Validated_Invariants composes the link it carries.
func Allocated_Node_Validated_Invariants(
  value *Allocated_Node_Validated,
  namespace aver.Namespace,
) {
  Allocated_Node_Link_Invariants(value.Value, namespace)
}

// OPENED_PIPELINE_COUNT_MINIMUM counts the root, the action, and its pipeline.
const OPENED_PIPELINE_COUNT_MINIMUM = ROOT_NODE_COUNT + ACTION_PIPELINE_NODE_COUNT

// OPENED_PIPELINE_ACTION_SIZE is the shortest action that opens one parenthesis.
const OPENED_PIPELINE_ACTION_SIZE = ACTION_BOUNDARY_SIZE_MINIMUM +
  PARENTHESIS_SIZE +
  DOT_SOURCE_SIZE +
  PARENTHESIS_SIZE

// OPENED_PIPELINE_COUNT_MAXIMUM packs the densest actions and one text node before that
// action's wrappers.
const OPENED_PIPELINE_COUNT_MAXIMUM = ROOT_NODE_COUNT +
  (SOURCE_SIZE_MAXIMUM-OPENED_PIPELINE_ACTION_SIZE)/
    SIMPLE_ACTION_SIZE_MINIMUM*
    SIMPLE_ACTION_NODE_COUNT +
  ((SOURCE_SIZE_MAXIMUM-OPENED_PIPELINE_ACTION_SIZE)%SIMPLE_ACTION_SIZE_MINIMUM+
    SIMPLE_ACTION_SIZE_MINIMUM-
    SOURCE_POSITION_INCREMENT)/
    SIMPLE_ACTION_SIZE_MINIMUM +
  SIMPLE_ACTION_CONTAINER_NODE_COUNT

// Opened_Pipeline_Count is the arena count one parenthesis open reads.
type Opened_Pipeline_Count Node_Count_Allocated

// Opened_Pipeline_Count_Invariants keeps the action and its pipeline beneath that count.
func Opened_Pipeline_Count_Invariants(
  value Opened_Pipeline_Count, namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(OPENED_PIPELINE_COUNT_MINIMUM),
    uint16(OPENED_PIPELINE_COUNT_MAXIMUM),
  ).
    Ensure()
}

// Skipped_Space_Cursor stands past the spaces one trimmed action left behind.
type Skipped_Space_Cursor Source_Cursor_Validated

// Skipped_Space_Cursor_Invariants keeps the trimmed action beneath that cursor.
func Skipped_Space_Cursor_Invariants(value Skipped_Space_Cursor, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(SKIP_SPACE_CURSOR_VALIDATED_VALUE_MINIMUM),
    uint16(SOURCE_SIZE_MAXIMUM),
  ).
    Ensure()
}

// PUSHED_LIST_REFERENCE_MAXIMUM leaves one branch and its body above that node.
const PUSHED_LIST_REFERENCE_MAXIMUM = NODE_REFERENCE_MAXIMUM -
  (CONTROL_BRANCH_NODE_COUNT +
    CONTROL_BRANCH_NODE_COUNT)

// Pushed_List_Reference names the list node one pushed frame walks.
type Pushed_List_Reference Node_Link_Proof_Stored

// Pushed_List_Reference_Invariants keeps the root, action and pipeline off that node.
func Pushed_List_Reference_Invariants(
  value Pushed_List_Reference, namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Holed_Uint16(
    uint16(value),
    uint16(NODE_REFERENCE_MINIMUM),
    uint16(PUSHED_LIST_REFERENCE_MAXIMUM),
    uint16(ROOT_LIST_HOLE),
    uint16(TERM_ACTION_HOLE),
    uint16(TERM_ACTION_HOLE),
  ).
    Ensure()
}

// Pushed_List_Validated carries proof of that list node.
type Pushed_List_Validated struct {
  // Value keeps arena proof attached to the reference.
  Value Pushed_List_Reference
}

// Pushed_List_Validated_Invariants composes the reference it carries.
func Pushed_List_Validated_Invariants(value *Pushed_List_Validated, namespace aver.Namespace) {
  Pushed_List_Reference_Invariants(value.Value, namespace)
}

// Branch_Allocation_Link names the branch node one clause allocated, or none.
type Branch_Allocation_Link Node_Link_Proof_Stored

// Branch_Allocation_Link_Invariants keeps the root off that node.
func Branch_Allocation_Link_Invariants(
  value Branch_Allocation_Link, namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Holed_Uint16(
    uint16(value),
    uint16(NODE_REFERENCE_MINIMUM),
    uint16(BRANCH_ALLOCATION_LINK_MAXIMUM),
    uint16(ROOT_LIST_HOLE),
    uint16(ROOT_LIST_HOLE),
    uint16(ROOT_LIST_HOLE),
  ).
    Ensure()
}

// Branch_Allocation_Validated carries proof of that branch node.
type Branch_Allocation_Validated struct {
  // Value keeps arena proof attached to the link.
  Value Branch_Allocation_Link
}

// Branch_Allocation_Validated_Invariants composes the link it carries.
func Branch_Allocation_Validated_Invariants(
  value *Branch_Allocation_Validated,
  namespace aver.Namespace,
) {
  Branch_Allocation_Link_Invariants(value.Value, namespace)
}

// Ranged_Scope_Base is the scope one range iteration binds its variables above.
type Ranged_Scope_Base Workspace_Scope_Base_Storage

// Ranged_Scope_Base_Invariants keeps the root scope off that base.
func Ranged_Scope_Base_Invariants(value Ranged_Scope_Base, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(VARIABLE_COUNT_MINIMUM),
    uint16(RANGE_SCOPE_BASE_MAXIMUM),
  ).
    Ensure()
}

// Ranged_Scope_Base_Pointer keeps that base mutable for its caller.
type Ranged_Scope_Base_Pointer *Ranged_Scope_Base

// Ranged_Scope_Base_Pointer_Invariants composes present scope storage.
func Ranged_Scope_Base_Pointer_Invariants(
  value Ranged_Scope_Base_Pointer,
  namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Ranged_Scope_Base_Invariants(*value, namespace)
}

// Quoted_Escape_Decode_Validated is the Quoted Position Validated quoted_escape_decode carries.
type Quoted_Escape_Decode_Validated Quoted_Position_Validated

// Quoted_Escape_Decode_Validated_Invariants keeps quoted-span proof attached to cursor.
func Quoted_Escape_Decode_Validated_Invariants(
  value Quoted_Escape_Decode_Validated,
  namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(ACTION_CONTENT_BOUNDARY_MINIMUM),
    uint16(QUOTED_ESCAPE_POSITION_MAXIMUM-ACTION_BOUNDARY_SIZE_MINIMUM),
  ).
    Ensure()
}

// Quoted_Escape_End_Position_Validated is the Quoted Position Validated quoted_escape_end carries.
type Quoted_Escape_End_Position_Validated Quoted_Position_Validated

// Quoted_Escape_End_Position_Validated_Invariants keeps quoted-span proof attached to cursor.
func Quoted_Escape_End_Position_Validated_Invariants(
  value Quoted_Escape_End_Position_Validated,
  namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(ACTION_CONTENT_BOUNDARY_MINIMUM),
    uint16(QUOTED_ESCAPE_POSITION_MAXIMUM-ACTION_BOUNDARY_SIZE_MINIMUM),
  ).
    Ensure()
}

// Quoted_Hex_Escape_End_Validated is the Quoted Position Validated quoted_hex_escape_end carries.
type Quoted_Hex_Escape_End_Validated Quoted_Position_Validated

// Quoted_Hex_Escape_End_Validated_Invariants keeps quoted-span proof attached to cursor.
func Quoted_Hex_Escape_End_Validated_Invariants(
  value Quoted_Hex_Escape_End_Validated,
  namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(QUOTED_HEX_ESCAPE_END_VALIDATED_VALUE_MINIMUM),
    uint16(QUOTED_ESCAPE_POSITION_MAXIMUM-utf8.CHARACTER_SIZE_MINIMUM),
  ).
    Ensure()
}

// Quoted_Octal_Escape_End_Validated is the Quoted Position Validated quoted_octal_escape_end
// carries.
type Quoted_Octal_Escape_End_Validated Quoted_Position_Validated

// Quoted_Octal_Escape_End_Validated_Invariants keeps quoted-span proof attached to cursor.
func Quoted_Octal_Escape_End_Validated_Invariants(
  value Quoted_Octal_Escape_End_Validated,
  namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(ACTION_CONTENT_BOUNDARY_MINIMUM),
    uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM-SIMPLE_ACTION_SIZE_MINIMUM),
  ).
    Ensure()
}

// Quoted_Character_End_Validated is the Quoted Position Validated quoted_character_end carries.
type Quoted_Character_End_Validated Quoted_Position_Validated

// Quoted_Character_End_Validated_Invariants keeps quoted-span proof attached to cursor.
func Quoted_Character_End_Validated_Invariants(
  value Quoted_Character_End_Validated,
  namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(ACTION_CONTENT_BOUNDARY_MINIMUM),
    uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM-utf8.CHARACTER_SIZE_MINIMUM),
  ).
    Ensure()
}

// Number_Component_Zero_Stored is the Number Span Proof Stored number_component_zero carries.
type Number_Component_Zero_Stored Number_Span_Proof_Stored

// Number_Component_Zero_Stored_Invariants fixes numeric-span representation.
func Number_Component_Zero_Stored_Invariants(
  value Number_Component_Zero_Stored,
  namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Start),
    uint16(NUMBER_START_MINIMUM),
    uint16(NUMBER_COMPONENT_ZERO_STORED_START_MAXIMUM),
  ).
    Range_Uint16(
    uint16(value.End),
    uint16(NUMBER_END_MINIMUM),
    uint16(NUMBER_END_MAXIMUM),
  ).
    Ensure()
}

// Number_Float_Valid_Validated is the Number Span Validated number_float_valid carries.
type Number_Float_Valid_Validated Number_Span_Validated

// Number_Float_Valid_Validated_Invariants checks transport ownership after token bounds validation.
func Number_Float_Valid_Validated_Invariants(
  value Number_Float_Valid_Validated,
  namespace aver.Namespace,
) {
  Number_Float_Valid_Stored_Invariants(Number_Float_Valid_Stored(value.Value), namespace)
}

// Number_Float_Valid_Stored is the Number Span Proof Stored number_float_valid carries.
type Number_Float_Valid_Stored Number_Span_Proof_Stored

// Number_Float_Valid_Stored_Invariants fixes numeric-span representation.
func Number_Float_Valid_Stored_Invariants(
  value Number_Float_Valid_Stored,
  namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Start),
    uint16(NUMBER_START_MINIMUM),
    uint16(NUMBER_FLOAT_VALID_STORED_START_MAXIMUM),
  ).
    Range_Uint16(
    uint16(value.End),
    uint16(NUMBER_END_MINIMUM),
    uint16(NUMBER_END_MAXIMUM),
  ).
    Ensure()
}

// Token_Quoted_Validated is the Lexeme Validated token_quoted carries.
type Token_Quoted_Validated Lexeme_Validated

// Token_Quoted_Validated_Invariants checks transport ownership after source validation.
func Token_Quoted_Validated_Invariants(value Token_Quoted_Validated, namespace aver.Namespace) {
  Token_Quoted_Stored_Invariants(Token_Quoted_Stored(value.Value), namespace)
}

// Token_Quoted_Stored is the Lexeme Proof Stored token_quoted carries.
type Token_Quoted_Stored Lexeme_Proof_Stored

// Token_Quoted_Stored_Invariants fixes lexeme representation.
func Token_Quoted_Stored_Invariants(value Token_Quoted_Stored, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Start),
    uint16(LEXEME_START_MINIMUM),
    uint16(TOKEN_QUOTED_STORED_START_MAXIMUM),
  ).
    Enum_3_Uint8(
    uint8(value.Character),
    uint8(QUOTE_STRING),
    uint8(QUOTE_CHARACTER),
    uint8(QUOTE_RAW_STRING),
  ).
    Ensure()
}

// Number_Lexeme is the lexeme one number opener carries.
type Number_Lexeme Lexeme_Proof_Stored

// Number_Lexeme_Invariants fixes the characters a number opens with.
func Number_Lexeme_Invariants(value Number_Lexeme, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Start),
    uint16(LEXEME_START_MINIMUM),
    uint16(LEXEME_START_MAXIMUM),
  ).
    Range_Uint8(
    uint8(value.Character),
    uint8(SIGN_CHARACTER),
    uint8(DIGIT_LAST_CHARACTER),
  ).
    Ensure()
}

// Closed_Control_Depth is the depth one close clause reads before it pops a branch.
type Closed_Control_Depth Control_Depth_Tracked

// Closed_Control_Depth_Invariants keeps one branch open beneath that depth.
func Closed_Control_Depth_Invariants(value Closed_Control_Depth, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(CONTROL_DEPTH_MINIMUM),
    uint16(CONTROL_DEPTH_MAXIMUM-CONTROL_DEPTH_INCREMENT),
  ).
    Ensure()
}

// Prefixed_Number_Position is where one number stops once its prefix is read.
type Prefixed_Number_Position Number_Position_Validated

// Prefixed_Number_Position_Invariants keeps one digit above that stop.
func Prefixed_Number_Position_Invariants(
  value Prefixed_Number_Position, namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(ACTION_CONTENT_BOUNDARY_MINIMUM),
    uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM-utf8.CHARACTER_SIZE_MINIMUM),
  ).
    Ensure()
}

// Closed_Parenthesis_Depth is the depth one close reads before it pops a parenthesis.
type Closed_Parenthesis_Depth Parenthesis_Depth_Tracked

// Closed_Parenthesis_Depth_Invariants keeps one parenthesis open beneath that depth.
func Closed_Parenthesis_Depth_Invariants(
  value Closed_Parenthesis_Depth, namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(PARENTHESIS_DEPTH_MINIMUM),
    uint16(PARENTHESIS_DEPTH_MAXIMUM-PARENTHESIS_DEPTH_INCREMENT),
  ).
    Ensure()
}

// Compound_Cursor is one lexer holding a token that a compound lexeme follows.
type Compound_Cursor Token_Cursor

// Compound_Cursor_Invariants leaves the compound lexeme byte present.
func Compound_Cursor_Invariants(value Compound_Cursor, namespace aver.Namespace) {
  Lexeme_Position_Invariants(Lexeme_Position(value.Position), namespace)
  Lexeme_End_Invariants(Lexeme_End(value.End), namespace)
  Lexeme_Token_Storage_Invariants(Lexeme_Token_Storage(value.Token), namespace)
  Cursor_Number_Stored_Invariants((*Cursor_Number_Stored)(&value.Number), namespace)
  Function_Function_Storage_Invariants(&value.Function, namespace)
  Lexeme_Last_Start_Invariants(Lexeme_Last_Start(value.Last_Start), namespace)
  Cursor_Has_Last_Stored_Invariants((*Cursor_Has_Last_Stored)(&value.Has_Last), namespace)
  Unread_Look_Ahead_Storage_Invariants(Unread_Look_Ahead_Storage(value.Looked), namespace)
}

// Compound_Cursor_Pointer keeps mutation of that lexer typed.
type Compound_Cursor_Pointer *Compound_Cursor

// Compound_Cursor_Pointer_Invariants composes present lexer storage.
func Compound_Cursor_Pointer_Invariants(value Compound_Cursor_Pointer, namespace aver.Namespace) {
  if value == nil {
    return
  }
  Compound_Cursor_Invariants(*value, namespace)
}

// OPENING_TEMPLATE_COUNT_MAXIMUM leaves room for the definition one open adds.
const OPENING_TEMPLATE_COUNT_MAXIMUM = TEMPLATE_COUNT_MAXIMUM - NODE_COUNT_INCREMENT

// Opening_Template_Count is the definition count one open reads before it adds.
type Opening_Template_Count Template_Count_Stored

// Opening_Template_Count_Invariants keeps that definition inside the table.
func Opening_Template_Count_Invariants(
  value Opening_Template_Count, namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(TEMPLATE_COUNT_MINIMUM),
    uint16(OPENING_TEMPLATE_COUNT_MAXIMUM),
  ).
    Ensure()
}

// Opening_Template_State is the definition chain one open extends.
type Opening_Template_State Template_State

// Opening_Template_State_Invariants bounds that chain's metadata.
func Opening_Template_State_Invariants(
  value Opening_Template_State, namespace aver.Namespace,
) {
  Prior_Template_First_Invariants(Prior_Template_First(value.First), namespace)
  Prior_Template_Last_Invariants(Prior_Template_Last(value.Last), namespace)
  Opening_Template_Count_Invariants(Opening_Template_Count(value.Count), namespace)
}

// Opening_Template_State_Pointer keeps mutation of that chain typed.
type Opening_Template_State_Pointer *Opening_Template_State

// Opening_Template_State_Pointer_Invariants composes one present chain.
func Opening_Template_State_Pointer_Invariants(
  value Opening_Template_State_Pointer,
  namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Opening_Template_State_Invariants(*value, namespace)
}

// INVOKED_IDENTIFIER_START_MINIMUM stands past the delimiter one action opens with.
const INVOKED_IDENTIFIER_START_MINIMUM = SOURCE_SIZE_MINIMUM + utf8.CHARACTER_SIZE_MINIMUM

// INVOKED_IDENTIFIER_END_MINIMUM stands past the first byte one called name takes.
const INVOKED_IDENTIFIER_END_MINIMUM = INVOKED_IDENTIFIER_START_MINIMUM +
  utf8.CHARACTER_SIZE_MINIMUM

// Invoked_Identifier_Span is the span one called name covers.
type Invoked_Identifier_Span Evaluation_Identifier_Span

// Invoked_Identifier_Span_Invariants keeps the delimiter and one byte inside that span.
func Invoked_Identifier_Span_Invariants(value Invoked_Identifier_Span, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Start),
    uint16(INVOKED_IDENTIFIER_START_MINIMUM),
    uint16(IDENTIFIER_START_MAXIMUM),
  ).
    Range_Uint16(
    uint16(value.End),
    uint16(INVOKED_IDENTIFIER_END_MINIMUM),
    uint16(IDENTIFIER_END_MAXIMUM),
  ).
    Ensure()
}

// QUOTED_END_SCAN_MINIMUM stands past the opening quote one literal takes.
const QUOTED_END_SCAN_MINIMUM = CONSTANT_END_MINIMUM + utf8.CHARACTER_SIZE_MINIMUM

// Quoted_End_Scan is where one quoted literal's scan stops.
type Quoted_End_Scan Action_Scan_End_Validated

// Quoted_End_Scan_Invariants keeps that quote below the stop.
func Quoted_End_Scan_Invariants(value Quoted_End_Scan, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(QUOTED_END_SCAN_MINIMUM),
    uint16(ACTION_SCAN_LIMIT_MAXIMUM),
  ).
    Ensure()
}

// Opened_Pipeline_Frame_State is the frame one open parenthesis reads.
type Opened_Pipeline_Frame_State Pipeline_Frame_State_Stored

// Opened_Pipeline_Frame_State_Invariants keeps that parenthesis inside the arena.
func Opened_Pipeline_Frame_State_Invariants(
  value Opened_Pipeline_Frame_State,
  namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Pipe),
    uint16(PIPELINE_NODE_COUNT_MINIMUM),
    uint16(ACTION_COMMAND_TAIL_REFERENCE_MAXIMUM),
  ).
    Range_Holed_Uint16(
    uint16(value.Last_Child),
    uint16(NODE_REFERENCE_MINIMUM),
    uint16(ACTION_TERM_TAIL_REFERENCE_MAXIMUM),
    uint16(ROOT_LIST_HOLE),
    uint16(TERM_ACTION_HOLE),
    uint16(TERM_PIPELINE_HOLE),
  ).
    Range_Holed_Uint16(
    uint16(value.Command),
    uint16(NODE_REFERENCE_MINIMUM),
    uint16(ACTION_TERM_TAIL_REFERENCE_MAXIMUM),
    uint16(ROOT_LIST_HOLE),
    uint16(TERM_ACTION_HOLE),
    uint16(TERM_PIPELINE_HOLE),
  ).
    Range_Holed_Uint16(
    uint16(value.Last_Argument),
    uint16(NODE_REFERENCE_MINIMUM),
    uint16(ACTION_COMMAND_TAIL_REFERENCE_MAXIMUM),
    uint16(ROOT_LIST_HOLE),
    uint16(TERM_ACTION_HOLE),
    uint16(TERM_PIPELINE_HOLE),
  ).
    Ensure()
}

// Opened_Pipeline_Frame is the frame storage one open parenthesis reads.
type Opened_Pipeline_Frame Pipeline_Frame_Storage

// Opened_Pipeline_Frame_Invariants composes that frame's storage.
func Opened_Pipeline_Frame_Invariants(
  value Opened_Pipeline_Frame, namespace aver.Namespace,
) {
  Opened_Pipeline_Frame_State_Invariants(
    Opened_Pipeline_Frame_State(value.Value), namespace,
  )
}

// Opened_Pipeline_Frame_Pointer keeps mutation of that frame typed.
type Opened_Pipeline_Frame_Pointer *Opened_Pipeline_Frame

// Opened_Pipeline_Frame_Pointer_Invariants composes one present frame.
func Opened_Pipeline_Frame_Pointer_Invariants(
  value Opened_Pipeline_Frame_Pointer,
  namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Opened_Pipeline_Frame_Invariants(*value, namespace)
}

// OVERFLOW_NUMBER_SIZE_MINIMUM is the shortest numeral past the float exponent range.
const OVERFLOW_NUMBER_SIZE_MINIMUM = len("1e999999")

// OVERFLOW_NUMBER_SOURCE_SIZE_MINIMUM adds the action that holds that numeral.
const OVERFLOW_NUMBER_SOURCE_SIZE_MINIMUM = ACTION_BOUNDARY_SIZE_MINIMUM +
  OVERFLOW_NUMBER_SIZE_MINIMUM

// PRESENT_DECLARATION_MINIMUM stands past the action one declaration opens with.
const PRESENT_DECLARATION_MINIMUM = ROOT_NODE_COUNT +
  SIMPLE_ACTION_CONTAINER_NODE_COUNT

// Present_Declaration_First is the first variable one read declaration names.
type Present_Declaration_First Declaration_First_Reference_Storage

// Present_Declaration_First_Invariants keeps that action below the variable.
func Present_Declaration_First_Invariants(
  value Present_Declaration_First,
  namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(PRESENT_DECLARATION_MINIMUM),
    uint16(FIRST_DECLARATION_REFERENCE_MAXIMUM),
  ).
    Ensure()
}

// Present_Declaration_References are the variables one read declaration names.
type Present_Declaration_References Declaration_References

// Present_Declaration_References_Invariants composes those variables.
func Present_Declaration_References_Invariants(
  value Present_Declaration_References,
  namespace aver.Namespace,
) {
  Present_Declaration_First_Invariants(
    Present_Declaration_First(value.First), namespace,
  )
  Declaration_Second_Reference_Storage_Invariants(&value.Second, namespace)
}

// Invoked_Program_Document is arena metadata of one program calling a named function.
type Invoked_Program_Document Simple_Program_Document

// Invoked_Program_Document_Invariants keeps that action below the definitions.
func Invoked_Program_Document_Invariants(
  value Invoked_Program_Document, namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Node_Count),
    uint16(SIMPLE_PROGRAM_NODE_COUNT_MINIMUM),
    uint16(SIMPLE_PROGRAM_NODE_COUNT_MAXIMUM),
  ).
    Range_Uint16(
    uint16(value.First_Template),
    uint16(DOCUMENT_FIRST_TEMPLATE_MINIMUM),
    uint16(DOCUMENT_FIRST_TEMPLATE_MAXIMUM),
  ).
    Range_Uint16(
    uint16(value.Template_Count),
    uint16(TEMPLATE_COUNT_MINIMUM),
    uint16(EXECUTED_TEMPLATE_COUNT_MAXIMUM),
  ).
    Ensure()
  aver.Always(
    uint16(value.Root) == uint16(ROOT_NODE_COUNT),
    "One calling program keeps the root in the first syntax arena slot.",
  )
}

// Invoked_Call_Program is one program whose action calls a named function.
type Invoked_Call_Program Simple_Program

// Invoked_Call_Program_Invariants composes that program's source and metadata.
func Invoked_Call_Program_Invariants(value Invoked_Call_Program, namespace aver.Namespace) {
  Simple_Program_Source_Invariants(Simple_Program_Source(value.Source), namespace)
  Invoked_Program_Document_Invariants(
    Invoked_Program_Document(value.Document), namespace,
  )
  Syntax_Storage_Invariants(&value.Syntax, namespace)
}

// LITERAL_REFERENCE_MAXIMUM leaves the literal's own action above the reference.
const LITERAL_REFERENCE_MAXIMUM = LOCATION_REFERENCE_MAXIMUM - SIMPLE_ACTION_NODE_COUNT

// Literal_Location_Place is the arena place one literal term occupies.
type Literal_Location_Place Literal_Location_Stored

// Literal_Location_Place_Invariants keeps that action inside the arena.
func Literal_Location_Place_Invariants(
  value Literal_Location_Place, namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Reference),
    uint16(COMMAND_NODE_COUNT_MINIMUM),
    uint16(LITERAL_REFERENCE_MAXIMUM),
  ).
    Ensure()
  Literal_Node_Stored_Invariants(value.Node, namespace)
}

// Literal_Location is one checked constant location one literal term reads.
type Literal_Location Literal_Location_Validated

// Literal_Location_Invariants checks transport ownership of that place.
func Literal_Location_Invariants(value Literal_Location, namespace aver.Namespace) {
  Literal_Location_Place_Invariants(Literal_Location_Place(value.Value), namespace)
}

// LITERAL_PROGRAM_NODE_COUNT_MAXIMUM leaves the literal's own container above it.
const LITERAL_PROGRAM_NODE_COUNT_MAXIMUM = FIELD_PROGRAM_NODE_COUNT_MAXIMUM -
  SIMPLE_ACTION_CONTAINER_NODE_COUNT

// Literal_Program_Document is arena metadata of one program holding a literal.
type Literal_Program_Document Field_Program_Document

// Literal_Program_Document_Invariants keeps that container inside the arena.
func Literal_Program_Document_Invariants(
  value Literal_Program_Document, namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Node_Count),
    uint16(ACTION_PROGRAM_NODE_COUNT_MINIMUM),
    uint16(LITERAL_PROGRAM_NODE_COUNT_MAXIMUM),
  ).
    Range_Uint16(
    uint16(value.First_Template),
    uint16(DOCUMENT_FIRST_TEMPLATE_MINIMUM),
    uint16(DOCUMENT_FIRST_TEMPLATE_MAXIMUM),
  ).
    Range_Uint16(
    uint16(value.Template_Count),
    uint16(TEMPLATE_COUNT_MINIMUM),
    uint16(EXECUTED_TEMPLATE_COUNT_MAXIMUM),
  ).
    Ensure()
  aver.Always(
    uint16(value.Root) == uint16(ROOT_NODE_COUNT),
    "One literal program keeps the root in the first syntax arena slot.",
  )
}

// Literal_Program is one program whose action holds one literal term.
type Literal_Program Field_Program

// Literal_Program_Invariants composes that program's source and metadata.
func Literal_Program_Invariants(value Literal_Program, namespace aver.Namespace) {
  Field_Program_Source_Invariants(Field_Program_Source(value.Source), namespace)
  Literal_Program_Document_Invariants(
    Literal_Program_Document(value.Document), namespace,
  )
  Syntax_Storage_Invariants(&value.Syntax, namespace)
}

// MERGED_TEMPLATE_MATCH_MINIMUM stands past the root list one merge keeps below it.
const MERGED_TEMPLATE_MATCH_MINIMUM = ROOT_NODE_COUNT + NODE_COUNT_INCREMENT

// Merged_Template_Boundaries are the definitions one merge stands between.
type Merged_Template_Boundaries Template_Search_Proof_Stored

// Merged_Template_Boundaries_Invariants keeps the matched definition present.
func Merged_Template_Boundaries_Invariants(
  value Merged_Template_Boundaries,
  namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Match),
    uint16(MERGED_TEMPLATE_MATCH_MINIMUM),
    uint16(EARLIER_TEMPLATE_REFERENCE_MAXIMUM),
  ).
    Range_Holed_Uint16(
    uint16(value.Previous),
    uint16(NODE_REFERENCE_MINIMUM),
    uint16(MERGE_PREVIOUS_REFERENCE_MAXIMUM),
    uint16(ROOT_LIST_HOLE),
    uint16(ROOT_LIST_HOLE),
    uint16(ROOT_LIST_HOLE),
  ).
    Ensure()
}

// Merged_Template_Search is the search result one merge acts on.
type Merged_Template_Search Template_Search_Validated

// Merged_Template_Search_Invariants composes that result's references.
func Merged_Template_Search_Invariants(
  value Merged_Template_Search, namespace aver.Namespace,
) {
  Merged_Template_Boundaries_Invariants(
    Merged_Template_Boundaries(value.Value), namespace,
  )
}

// MERGED_TEMPLATE_CURRENT_MINIMUM stands past the action one definition opens with.
const MERGED_TEMPLATE_CURRENT_MINIMUM = ROOT_NODE_COUNT +
  SIMPLE_ACTION_CONTAINER_NODE_COUNT

// Merged_Template_Reference is the definition one merge folds into the chain.
type Merged_Template_Reference Template_Reference_Validated

// Merged_Template_Reference_Invariants keeps that action below the definition.
func Merged_Template_Reference_Invariants(
  value Merged_Template_Reference,
  namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(MERGED_TEMPLATE_CURRENT_MINIMUM),
    uint16(NAMES_EQUAL_REFERENCE_MAXIMUM),
  ).
    Ensure()
}

// OVERFLOW_NUMBER_START_MAXIMUM leaves that numeral and its action above the start.
const OVERFLOW_NUMBER_START_MAXIMUM = NUMBER_START_MAXIMUM -
  (OVERFLOW_NUMBER_SIZE_MINIMUM -
    utf8.CHARACTER_SIZE_MINIMUM)

// OVERFLOW_NUMBER_END_MINIMUM stands past that numeral and the delimiter below it.
const OVERFLOW_NUMBER_END_MINIMUM = NUMBER_END_MINIMUM +
  (OVERFLOW_NUMBER_SIZE_MINIMUM -
    utf8.CHARACTER_SIZE_MINIMUM)

// Overflow_Number_Span is the span one overflowing numeral covers.
type Overflow_Number_Span Number_Span_Validated

// Overflow_Number_Span_Invariants composes that numeral's boundaries.
func Overflow_Number_Span_Invariants(value Overflow_Number_Span, namespace aver.Namespace) {
  Overflow_Number_Boundaries_Invariants(
    Overflow_Number_Boundaries(value.Value), namespace,
  )
}

// Overflow_Number_Boundaries are the boundaries one overflowing numeral covers.
type Overflow_Number_Boundaries Number_Span_Proof_Stored

// Overflow_Number_Boundaries_Invariants keeps that numeral inside the action.
func Overflow_Number_Boundaries_Invariants(
  value Overflow_Number_Boundaries,
  namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Start),
    uint16(NUMBER_START_MINIMUM),
    uint16(OVERFLOW_NUMBER_START_MAXIMUM),
  ).
    Range_Uint16(
    uint16(value.End),
    uint16(OVERFLOW_NUMBER_END_MINIMUM),
    uint16(NUMBER_END_MAXIMUM),
  ).
    Ensure()
}

// Overflow_Number_Data is the storage one overflowing numeral stands inside.
type Overflow_Number_Data Branch_Source_Data_Stored

// Overflow_Number_Data_Invariants keeps that numeral inside the source.
func Overflow_Number_Data_Invariants(value Overflow_Number_Data, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(
    len(value.Value),
    OVERFLOW_NUMBER_SOURCE_SIZE_MINIMUM,
    SOURCE_SIZE_MAXIMUM,
  ).
    Ensure()
}

// Overflow_Number_Source is one source that holds one overflowing numeral.
type Overflow_Number_Source Branch_Source

// Overflow_Number_Source_Invariants composes that source's storage.
func Overflow_Number_Source_Invariants(
  value Overflow_Number_Source, namespace aver.Namespace,
) {
  Overflow_Number_Data_Invariants(Overflow_Number_Data(value.Data), namespace)
}

// CALLED_FUNCTION_ACTION_SIZE is the shortest action one function term stands in, as the
// argument of one call.
const CALLED_FUNCTION_ACTION_SIZE = ACTION_BOUNDARY_SIZE_MINIMUM +
  IDENTIFIER_SIZE_MINIMUM +
  ACTION_SEPARATOR_SIZE_MINIMUM +
  IDENTIFIER_SIZE_MINIMUM

// CALLED_FUNCTION_NODE_COUNT_MAXIMUM packs the densest actions and one text node beside that
// action.
const CALLED_FUNCTION_NODE_COUNT_MAXIMUM = ROOT_NODE_COUNT +
  (SOURCE_SIZE_MAXIMUM-CALLED_FUNCTION_ACTION_SIZE)/
    SIMPLE_ACTION_SIZE_MINIMUM*
    SIMPLE_ACTION_NODE_COUNT +
  ((SOURCE_SIZE_MAXIMUM-CALLED_FUNCTION_ACTION_SIZE)%SIMPLE_ACTION_SIZE_MINIMUM+
    SIMPLE_ACTION_SIZE_MINIMUM-
    SOURCE_POSITION_INCREMENT)/
    SIMPLE_ACTION_SIZE_MINIMUM +
  SIMPLE_ACTION_NODE_COUNT +
  SCALAR_NODE_COUNT

// Called_Function_Document is arena metadata of one program naming a function term.
type Called_Function_Document Chained_Program_Document

// Called_Function_Document_Invariants keeps that action inside the arena.
func Called_Function_Document_Invariants(
  value Called_Function_Document, namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Node_Count),
    uint16(CHAINED_PROGRAM_NODE_COUNT_MINIMUM),
    uint16(CALLED_FUNCTION_NODE_COUNT_MAXIMUM),
  ).
    Range_Uint16(
    uint16(value.First_Template),
    uint16(DOCUMENT_FIRST_TEMPLATE_MINIMUM),
    uint16(DOCUMENT_FIRST_TEMPLATE_MAXIMUM),
  ).
    Range_Uint16(
    uint16(value.Template_Count),
    uint16(TEMPLATE_COUNT_MINIMUM),
    uint16(EXECUTED_TEMPLATE_COUNT_MAXIMUM),
  ).
    Ensure()
  aver.Always(
    uint16(value.Root) == uint16(ROOT_NODE_COUNT),
    "One named function program keeps the root in the first syntax arena slot.",
  )
}

// Called_Function_Program is one program whose action names a function term.
type Called_Function_Program Chain_Program

// Called_Function_Program_Invariants composes that program's source and metadata.
func Called_Function_Program_Invariants(
  value Called_Function_Program, namespace aver.Namespace,
) {
  Chain_Program_Source_Invariants(Chain_Program_Source(value.Source), namespace)
  Called_Function_Document_Invariants(Called_Function_Document(value.Document), namespace)
  Syntax_Storage_Invariants(&value.Syntax, namespace)
}

// SEPARATED_LAST_START_MAXIMUM leaves the pipe, its term and the delimiter above it.
const SEPARATED_LAST_START_MAXIMUM = ACTION_CONTENT_BOUNDARY_MAXIMUM -
  (PIPELINE_SEPARATOR_SIZE +
    DOT_SOURCE_SIZE +
    utf8.CHARACTER_SIZE_MINIMUM)

// Separated_Last_Start is where one separated pipeline's last token began.
type Separated_Last_Start Cursor_Last_Start_Stored

// Separated_Last_Start_Invariants keeps that pipe and term inside the action.
func Separated_Last_Start_Invariants(value Separated_Last_Start, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(SOURCE_SIZE_MINIMUM),
    uint16(SEPARATED_LAST_START_MAXIMUM),
  ).
    Ensure()
}

// DELIVERED_IDENTIFIER_START_MAXIMUM leaves the name, its separator, and the shortest
// parenthesised child before the content boundary.
const DELIVERED_IDENTIFIER_START_MAXIMUM = ACTION_CONTENT_BOUNDARY_MAXIMUM -
  (IDENTIFIER_SIZE_MINIMUM +
    ACTION_SEPARATOR_SIZE_MINIMUM +
    PARENTHESIS_SIZE +
    DOT_SOURCE_SIZE +
    PARENTHESIS_SIZE)

// DELIVERED_IDENTIFIER_END_MAXIMUM stands past that shortest name.
const DELIVERED_IDENTIFIER_END_MAXIMUM = DELIVERED_IDENTIFIER_START_MAXIMUM +
  IDENTIFIER_SIZE_MINIMUM

// Delivered_Identifier_Span is the name one frame waiting on a child stands on.
type Delivered_Identifier_Span Called_Identifier_Span

// Delivered_Identifier_Span_Invariants keeps that child call inside the action.
func Delivered_Identifier_Span_Invariants(
  value Delivered_Identifier_Span,
  namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Start),
    uint16(SOURCE_SIZE_MINIMUM),
    uint16(DELIVERED_IDENTIFIER_START_MAXIMUM),
  ).
    Range_Holed_Uint16(
    uint16(value.End),
    uint16(SOURCE_SIZE_MINIMUM),
    uint16(DELIVERED_IDENTIFIER_END_MAXIMUM),
    uint16(CALLED_IDENTIFIER_END_HOLE),
    uint16(CALLED_IDENTIFIER_END_HOLE),
    uint16(CALLED_IDENTIFIER_END_HOLE),
  ).
    Ensure()
}

// Delivered_Evaluation_Nodes are the nodes one frame waiting on a child stands on.
type Delivered_Evaluation_Nodes Called_Evaluation_Nodes

// Delivered_Evaluation_Nodes_Invariants composes that frame's spans.
func Delivered_Evaluation_Nodes_Invariants(
  value Delivered_Evaluation_Nodes,
  namespace aver.Namespace,
) {
  Delivered_Identifier_Span_Invariants(
    Delivered_Identifier_Span(value.Identifier), namespace,
  )
  Evaluation_Chain_Span_Invariants(&value.Chain, namespace)
}

// Delivered_Evaluation_References are the nodes one frame keeps while a child answers.
type Delivered_Evaluation_References Argument_Evaluation_References

// Delivered_Evaluation_References_Invariants keeps the read command out of that frame.
func Delivered_Evaluation_References_Invariants(
  value Delivered_Evaluation_References,
  namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Holed_Uint16(
    uint16(value.Command),
    uint16(NODE_REFERENCE_MINIMUM),
    uint16(CONTAINER_TAIL_REFERENCE_MAXIMUM),
    uint16(ROOT_LIST_HOLE),
    uint16(TERM_ACTION_HOLE),
    uint16(TERM_PIPELINE_HOLE),
  ).
    Range_Holed_Uint16(
    uint16(value.Term),
    uint16(NODE_REFERENCE_MINIMUM),
    uint16(ACTION_TAIL_REFERENCE_MAXIMUM),
    uint16(ROOT_LIST_HOLE),
    uint16(TERM_ACTION_HOLE),
    uint16(TERM_PIPELINE_HOLE),
  ).
    Ensure()
}

// BOUND_RANGE_SOURCE_SIZE_MINIMUM counts the one action one walked body holds.
const BOUND_RANGE_SOURCE_SIZE_MINIMUM = RANGE_PROGRAM_SOURCE_SIZE_MINIMUM +
  utf8.CHARACTER_SIZE_MINIMUM +
  SIMPLE_ACTION_SIZE_MINIMUM

// Bound_Range_Data is the source storage one walked range body points into.
type Bound_Range_Data Range_Program_Data_Stored

// Bound_Range_Data_Invariants keeps that action inside the source.
func Bound_Range_Data_Invariants(value Bound_Range_Data, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(
    len(value.Value),
    BOUND_RANGE_SOURCE_SIZE_MINIMUM,
    SOURCE_SIZE_MAXIMUM,
  ).
    Ensure()
}

// Bound_Range_Source is the source one walked range body stands inside.
type Bound_Range_Source Range_Program_Source

// Bound_Range_Source_Invariants composes that source's storage.
func Bound_Range_Source_Invariants(value Bound_Range_Source, namespace aver.Namespace) {
  Bound_Range_Data_Invariants(Bound_Range_Data(value.Data), namespace)
}

// Bound_Range_Program is one program whose range body a frame walks.
type Bound_Range_Program Range_Program

// Bound_Range_Program_Invariants composes that program's source and metadata.
func Bound_Range_Program_Invariants(value Bound_Range_Program, namespace aver.Namespace) {
  Bound_Range_Source_Invariants(Bound_Range_Source(value.Source), namespace)
  Ranged_Program_Document_Invariants(Ranged_Program_Document(value.Document), namespace)
  Syntax_Storage_Invariants(&value.Syntax, namespace)
}

// Bound_Variable_Document is arena metadata of one program holding a variable term.
type Bound_Variable_Document Simple_Program_Document

// Bound_Variable_Document_Invariants keeps that declaration inside the arena.
func Bound_Variable_Document_Invariants(
  value Bound_Variable_Document, namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Node_Count),
    uint16(SIMPLE_PROGRAM_NODE_COUNT_MINIMUM),
    uint16(SIMPLE_PROGRAM_NODE_COUNT_MAXIMUM),
  ).
    Range_Uint16(
    uint16(value.First_Template),
    uint16(DOCUMENT_FIRST_TEMPLATE_MINIMUM),
    uint16(DOCUMENT_FIRST_TEMPLATE_MAXIMUM),
  ).
    Range_Uint16(
    uint16(value.Template_Count),
    uint16(TEMPLATE_COUNT_MINIMUM),
    uint16(EXECUTED_TEMPLATE_COUNT_MAXIMUM),
  ).
    Ensure()
  aver.Always(
    uint16(value.Root) == uint16(ROOT_NODE_COUNT),
    "One variable program keeps the root in the first syntax arena slot.",
  )
}

// Bound_Variable_Program is one program whose action binds a variable term.
type Bound_Variable_Program Simple_Program

// Bound_Variable_Program_Invariants composes that program's source and metadata.
func Bound_Variable_Program_Invariants(
  value Bound_Variable_Program, namespace aver.Namespace,
) {
  Simple_Program_Source_Invariants(Simple_Program_Source(value.Source), namespace)
  Bound_Variable_Document_Invariants(Bound_Variable_Document(value.Document), namespace)
  Syntax_Storage_Invariants(&value.Syntax, namespace)
}

// CLAUSE_TEXT_TAIL_SIZE is one text byte before one stray shortest keyword action.
const CLAUSE_TEXT_TAIL_SIZE = utf8.CHARACTER_SIZE_MINIMUM + WORD_SOURCE_SIZE_MINIMUM

// CONTROL_LIST_LAST_CHILD_MAXIMUM is that text node behind the densest actions and one text node.
const CONTROL_LIST_LAST_CHILD_MAXIMUM = ROOT_NODE_COUNT +
  (SOURCE_SIZE_MAXIMUM-CLAUSE_TEXT_TAIL_SIZE)/SIMPLE_ACTION_SIZE_MINIMUM*
    SIMPLE_ACTION_NODE_COUNT +
  ((SOURCE_SIZE_MAXIMUM-CLAUSE_TEXT_TAIL_SIZE)%SIMPLE_ACTION_SIZE_MINIMUM+
    SIMPLE_ACTION_SIZE_MINIMUM-
    SOURCE_POSITION_INCREMENT)/
    SIMPLE_ACTION_SIZE_MINIMUM +
  NODE_COUNT_INCREMENT

// Control_List_Value is the list one control clause appends to.
type Control_List_Value List_State_Value_Stored

// Control_List_Value_Invariants keeps that action and pipeline inside the arena.
func Control_List_Value_Invariants(value Control_List_Value, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Holed_Uint16(
    uint16(value.List),
    uint16(ROOT_NODE_COUNT),
    uint16(LIST_REFERENCE_MAXIMUM),
    uint16(ACTION_LIST_HOLE),
    uint16(ACTION_LIST_HOLE),
    uint16(ACTION_LIST_HOLE),
  ).
    Range_Holed_Uint16(
    uint16(value.Last_Child),
    uint16(NODE_REFERENCE_MINIMUM),
    uint16(CONTROL_LIST_LAST_CHILD_MAXIMUM),
    uint16(ROOT_LIST_HOLE),
    uint16(ROOT_LIST_HOLE),
    uint16(ROOT_LIST_HOLE),
  ).
    Ensure()
}

// Control_List_State is the list storage one control clause reads.
type Control_List_State List_State_Storage

// Control_List_State_Invariants composes that list's storage.
func Control_List_State_Invariants(value Control_List_State, namespace aver.Namespace) {
  Control_List_Value_Invariants(Control_List_Value(value.Value), namespace)
}

// Control_List_State_Pointer keeps mutation of that list typed.
type Control_List_State_Pointer *Control_List_State

// Control_List_State_Pointer_Invariants composes one present list.
func Control_List_State_Pointer_Invariants(
  value Control_List_State_Pointer,
  namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Control_List_State_Invariants(*value, namespace)
}

// DECLARED_PIPELINE_NODE_COUNT_MAXIMUM leaves the declaration's action above it.
const DECLARED_PIPELINE_NODE_COUNT_MAXIMUM = BINDING_REFERENCE_MAXIMUM

// Declared_Pipeline_Node_Count is the arena length one declaration reads.
type Declared_Pipeline_Node_Count Dense_Pipeline_Node_Count

// Declared_Pipeline_Node_Count_Invariants keeps that action inside the arena.
func Declared_Pipeline_Node_Count_Invariants(
  value Declared_Pipeline_Node_Count,
  namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(PIPELINE_NODE_COUNT_MINIMUM),
    uint16(DECLARED_PIPELINE_NODE_COUNT_MAXIMUM),
  ).
    Ensure()
}

// OPENED_PIPELINE_NODE_COUNT_MAXIMUM adds the pipeline one open parenthesis pushes.
const OPENED_PIPELINE_NODE_COUNT_MAXIMUM = OPENED_PIPELINE_COUNT_MAXIMUM + PIPELINE_NODE_COUNT

// Opened_Pipeline_Node_Count is the arena length one open parenthesis reads.
type Opened_Pipeline_Node_Count Dense_Pipeline_Node_Count

// Opened_Pipeline_Node_Count_Invariants keeps those nodes inside the arena.
func Opened_Pipeline_Node_Count_Invariants(
  value Opened_Pipeline_Node_Count,
  namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(PIPELINE_NODE_COUNT_MINIMUM),
    uint16(OPENED_PIPELINE_NODE_COUNT_MAXIMUM),
  ).
    Ensure()
}

// OPEN_PAREN_LAST_START_MAXIMUM leaves that one-byte token and the parenthesis above the start.
const OPEN_PAREN_LAST_START_MAXIMUM = ACTION_CONTENT_BOUNDARY_MAXIMUM -
  PARENTHESIS_SIZE -
  utf8.CHARACTER_SIZE_MINIMUM

// Open_Paren_Last_Start is where one open parenthesis clause's last token began.
type Open_Paren_Last_Start Cursor_Last_Start_Stored

// Open_Paren_Last_Start_Invariants keeps that parenthesis inside the action.
func Open_Paren_Last_Start_Invariants(
  value Open_Paren_Last_Start, namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(SOURCE_SIZE_MINIMUM),
    uint16(OPEN_PAREN_LAST_START_MAXIMUM),
  ).
    Ensure()
}

// CALLED_ARGUMENT_COUNT_MAXIMUM leaves the call's own command and its term above it.
const CALLED_ARGUMENT_COUNT_MAXIMUM = ARGUMENT_COUNT_MAXIMUM -
  (COMMAND_NODE_COUNT +
    SCALAR_NODE_COUNT +
    SCALAR_NODE_COUNT)

// Called_Argument_Count is the argument count one call reads before it delivers.
type Called_Argument_Count Argument_Count_Tracked

// Called_Argument_Count_Invariants keeps that command and term inside the action.
func Called_Argument_Count_Invariants(value Called_Argument_Count, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(ARGUMENT_COUNT_MINIMUM),
    uint16(CALLED_ARGUMENT_COUNT_MAXIMUM),
  ).
    Ensure()
}

// DELIVERED_ARGUMENT_COUNT_MAXIMUM leaves the child's own command above it.
const DELIVERED_ARGUMENT_COUNT_MAXIMUM = ARGUMENT_COUNT_MAXIMUM -
  (COMMAND_NODE_COUNT +
    SCALAR_NODE_COUNT)

// Delivered_Argument_Count is the argument count one child result stands under.
type Delivered_Argument_Count Argument_Count_Tracked

// Delivered_Argument_Count_Invariants keeps that command inside the action.
func Delivered_Argument_Count_Invariants(
  value Delivered_Argument_Count, namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(ARGUMENT_COUNT_MINIMUM),
    uint16(DELIVERED_ARGUMENT_COUNT_MAXIMUM),
  ).
    Ensure()
}

// Recorded_Parse_Failure is the outcome one diagnostic keeps once parsing began.
type Recorded_Parse_Failure Parse_Failure_Status_Validated

// Recorded_Parse_Failure_Invariants keeps the refused inputs out of that record.
func Recorded_Parse_Failure_Invariants(
  value Recorded_Parse_Failure, namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Enum_Uint8(
    uint8(value.Value),
    uint8(PARSE_STATUS_SYNTAX_INVALID),
    uint8(PARSE_STATUS_CAPACITY_EXCEEDED),
  ).
    Ensure()
}

// CLAUSE_KEYWORD_SIZE_MINIMUM is the shortest keyword, its delimiter and its space.
const CLAUSE_KEYWORD_SIZE_MINIMUM = len(KEYWORD_IF) +
  utf8.CHARACTER_SIZE_MINIMUM +
  utf8.CHARACTER_SIZE_MINIMUM

// CLAUSE_IDENTIFIER_START_MAXIMUM leaves that keyword's clause above the name.
const CLAUSE_IDENTIFIER_START_MAXIMUM = IDENTIFIER_START_MAXIMUM -
  CLAUSE_KEYWORD_SIZE_MINIMUM

// CLAUSE_IDENTIFIER_END_MINIMUM stands past that keyword's clause.
const CLAUSE_IDENTIFIER_END_MINIMUM = IDENTIFIER_END_MINIMUM + CLAUSE_KEYWORD_SIZE_MINIMUM

// Clause_Identifier_Span is the span one name inside a clause covers.
type Clause_Identifier_Span Identifier_Span_Validated

// Clause_Identifier_Span_Invariants composes that name's boundaries.
func Clause_Identifier_Span_Invariants(value Clause_Identifier_Span, namespace aver.Namespace) {
  Clause_Identifier_Boundaries_Invariants(
    Clause_Identifier_Boundaries(value.Value), namespace,
  )
}

// Clause_Identifier_Boundaries are the boundaries one name inside a clause covers.
type Clause_Identifier_Boundaries Identifier_Span_Proof_Stored

// Clause_Identifier_Boundaries_Invariants keeps that clause outside the name.
func Clause_Identifier_Boundaries_Invariants(
  value Clause_Identifier_Boundaries,
  namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Start),
    uint16(IDENTIFIER_START_MINIMUM),
    uint16(CLAUSE_IDENTIFIER_START_MAXIMUM),
  ).
    Range_Uint16(
    uint16(value.End),
    uint16(CLAUSE_IDENTIFIER_END_MINIMUM),
    uint16(IDENTIFIER_END_MAXIMUM),
  ).
    Ensure()
}

// OCTAL_ESCAPE_POSITION_MAXIMUM leaves the three digits one octal escape spells.
const OCTAL_ESCAPE_POSITION_MAXIMUM = ACTION_CONTENT_BOUNDARY_MAXIMUM -
  int(QUOTED_OCTAL_DIGIT_COUNT)

// Octal_Escape_Position is where one octal escape stops.
type Octal_Escape_Position Quoted_Position_Validated

// Octal_Escape_Position_Invariants keeps those digits inside the literal.
func Octal_Escape_Position_Invariants(value Octal_Escape_Position, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(CONSTANT_END_MINIMUM),
    uint16(OCTAL_ESCAPE_POSITION_MAXIMUM),
  ).
    Ensure()
}

// QUOTED_CHARACTER_START_MAXIMUM leaves one character and its quote above the start.
const QUOTED_CHARACTER_START_MAXIMUM = QUOTED_START_MAXIMUM -
  utf8.CHARACTER_SIZE_MINIMUM -
  utf8.CHARACTER_SIZE_MINIMUM

// QUOTED_CHARACTER_END_MINIMUM stands past the one character the quotes hold.
const QUOTED_CHARACTER_END_MINIMUM = QUOTED_END_MINIMUM + utf8.CHARACTER_SIZE_MINIMUM

// Quoted_Character_Span is the span one quoted character stands inside.
type Quoted_Character_Span Quoted_Span_Validated

// Quoted_Character_Span_Invariants composes that character's boundaries.
func Quoted_Character_Span_Invariants(value Quoted_Character_Span, namespace aver.Namespace) {
  Quoted_Character_Boundaries_Invariants(
    Quoted_Character_Boundaries(value.Value), namespace,
  )
}

// Quoted_Character_Boundaries are the boundaries one quoted character stands inside.
type Quoted_Character_Boundaries Quoted_Span_Proof_Stored

// Quoted_Character_Boundaries_Invariants keeps that character inside the quotes.
func Quoted_Character_Boundaries_Invariants(
  value Quoted_Character_Boundaries,
  namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Start),
    uint16(QUOTED_START_MINIMUM),
    uint16(QUOTED_CHARACTER_START_MAXIMUM),
  ).
    Range_Uint16(
    uint16(value.End),
    uint16(QUOTED_CHARACTER_END_MINIMUM),
    uint16(QUOTED_END_MAXIMUM),
  ).
    Ensure()
}

// SCANNED_MAP_FIELD_COUNT_MINIMUM counts the keys one unordered map holds, because
// one key and two keys always stand in order.
const SCANNED_MAP_FIELD_COUNT_MINIMUM = SORTED_MAP_FIELD_COUNT_MINIMUM +
  MAP_ENTRY_COUNT_INCREMENT +
  MAP_ENTRY_COUNT_INCREMENT

// Scanned_Map_Fields are the keys one scan walks out of order.
type Scanned_Map_Fields Map_Fields_Validated

// Scanned_Map_Fields_Invariants keeps those three keys inside the scan.
func Scanned_Map_Fields_Invariants(value Scanned_Map_Fields, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(
    len(value.Value),
    SCANNED_MAP_FIELD_COUNT_MINIMUM,
    RANGE_MAP_ENTRY_COUNT_MAXIMUM,
  ).
    Ensure()
}

// SORTED_MAP_FIELD_COUNT_MINIMUM counts the one key a direction orders.
const SORTED_MAP_FIELD_COUNT_MINIMUM = bytes.SLICE_SIZE_MINIMUM + 1

// Sorted_Map_Fields are the keys one direction orders.
type Sorted_Map_Fields Map_Fields_Validated

// Sorted_Map_Fields_Invariants keeps one key inside that order.
func Sorted_Map_Fields_Invariants(value Sorted_Map_Fields, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(
    len(value.Value),
    SORTED_MAP_FIELD_COUNT_MINIMUM,
    RANGE_MAP_ENTRY_COUNT_MAXIMUM,
  ).
    Ensure()
}

// Recorded_Failure_Status is the outcome one diagnostic keeps after execution starts.
type Recorded_Failure_Status Failure_Status_Validated

// Recorded_Failure_Status_Invariants keeps a short destination out of that record.
func Recorded_Failure_Status_Invariants(
  value Recorded_Failure_Status, namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint8(
    uint8(value.Value),
    uint8(STATUS_OUTPUT_TOO_LARGE),
    uint8(STATUS_LIMIT_EXCEEDED),
  ).
    Ensure()
}

// IDENTIFIER_CHARACTER_AT_POSITION_MAXIMUM leaves one character above the position.
const IDENTIFIER_CHARACTER_AT_POSITION_MAXIMUM = ACTION_CONTENT_BOUNDARY_MAXIMUM -
  utf8.CHARACTER_SIZE_MINIMUM

// Identifier_Character_At_Position_Storage is where one scan reads its character.
type Identifier_Character_At_Position_Storage Identifier_Search_Position_Storage

// Identifier_Character_At_Position_Storage_Invariants keeps that character readable.
func Identifier_Character_At_Position_Storage_Invariants(
  value Identifier_Character_At_Position_Storage,
  namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(ACTION_CONTENT_BOUNDARY_MINIMUM),
    uint16(IDENTIFIER_CHARACTER_AT_POSITION_MAXIMUM),
  ).
    Ensure()
}

// SEQUENCED_IDENTIFIER_POSITION_MINIMUM stands past the first byte one identifier takes.
const SEQUENCED_IDENTIFIER_POSITION_MINIMUM = ACTION_CONTENT_BOUNDARY_MINIMUM +
  utf8.CHARACTER_SIZE_MINIMUM

// Sequenced_Identifier_Position is where one read identifier stops.
type Sequenced_Identifier_Position Identifier_Search_Position_Storage

// Sequenced_Identifier_Position_Invariants keeps that byte below the position.
func Sequenced_Identifier_Position_Invariants(
  value Sequenced_Identifier_Position,
  namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(SEQUENCED_IDENTIFIER_POSITION_MINIMUM),
    uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM),
  ).
    Ensure()
}

// Sequenced_Identifier_Span is the span one read identifier leaves behind.
type Sequenced_Identifier_Span Identifier_Search_Span

// Sequenced_Identifier_Span_Invariants composes that span's boundaries.
func Sequenced_Identifier_Span_Invariants(
  value Sequenced_Identifier_Span,
  namespace aver.Namespace,
) {
  Sequenced_Identifier_Position_Invariants(
    Sequenced_Identifier_Position(value.Position), namespace,
  )
  Identifier_Search_End_Storage_Invariants(&value.End, namespace)
}

// Sequenced_Identifier_Search is one search one read identifier answers.
type Sequenced_Identifier_Search Identifier_Search_Validated

// Sequenced_Identifier_Search_Invariants checks proof transport ownership.
func Sequenced_Identifier_Search_Invariants(
  value *Sequenced_Identifier_Search,
  namespace aver.Namespace,
) {
  Sequenced_Identifier_Span_Invariants(Sequenced_Identifier_Span(value.Value), namespace)
}

// OPENED_PARENTHESIS_DEPTH_MINIMUM counts the parenthesis one open pushed.
const OPENED_PARENTHESIS_DEPTH_MINIMUM = PARENTHESIS_DEPTH_MINIMUM +
  PARENTHESIS_DEPTH_INCREMENT

// Opened_Parenthesis_Depth is the depth one open clause leaves behind it.
type Opened_Parenthesis_Depth Parenthesis_Depth_Tracked

// Opened_Parenthesis_Depth_Invariants keeps that parenthesis on the stack.
func Opened_Parenthesis_Depth_Invariants(
  value Opened_Parenthesis_Depth, namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(OPENED_PARENTHESIS_DEPTH_MINIMUM),
    uint16(PARENTHESIS_DEPTH_MAXIMUM),
  ).
    Ensure()
}

// NUMBER_BODY_POSITION_MINIMUM stands past the first byte one number takes.
const NUMBER_BODY_POSITION_MINIMUM = ACTION_CONTENT_BOUNDARY_MINIMUM +
  utf8.CHARACTER_SIZE_MINIMUM

// Number_Body_Position is where one number stops once a byte of it is read.
type Number_Body_Position Number_Position_Validated

// Number_Body_Position_Invariants keeps that byte below the position.
func Number_Body_Position_Invariants(value Number_Body_Position, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(NUMBER_BODY_POSITION_MINIMUM),
    uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM),
  ).
    Ensure()
}

// OPENED_SOURCE_CURSOR_MINIMUM stands past the shortest opening delimiter.
const OPENED_SOURCE_CURSOR_MINIMUM = SOURCE_SIZE_MINIMUM + utf8.CHARACTER_SIZE_MINIMUM

// Opened_Source_Cursor is one position past the delimiter that opened it.
type Opened_Source_Cursor Source_Cursor_Validated

// Opened_Source_Cursor_Invariants keeps that delimiter below the position.
func Opened_Source_Cursor_Invariants(value Opened_Source_Cursor, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(OPENED_SOURCE_CURSOR_MINIMUM),
    uint16(SOURCE_SIZE_MAXIMUM),
  ).
    Ensure()
}

// DELIMITER_START_CURSOR_MAXIMUM leaves one shortest delimiter above it.
const DELIMITER_START_CURSOR_MAXIMUM = SOURCE_SIZE_MAXIMUM - utf8.CHARACTER_SIZE_MINIMUM

// Delimiter_Start_Cursor is where one delimiter begins.
type Delimiter_Start_Cursor Source_Cursor_Validated

// Delimiter_Start_Cursor_Invariants keeps that delimiter inside the source.
func Delimiter_Start_Cursor_Invariants(value Delimiter_Start_Cursor, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(SOURCE_SIZE_MINIMUM),
    uint16(DELIMITER_START_CURSOR_MAXIMUM),
  ).
    Ensure()
}

// TRIMMED_END_CURSOR_MINIMUM stands past the shortest opening delimiter.
const TRIMMED_END_CURSOR_MINIMUM = SOURCE_SIZE_MINIMUM + utf8.CHARACTER_SIZE_MINIMUM

// TRIMMED_END_CURSOR_MAXIMUM leaves the shortest closing delimiter above it.
const TRIMMED_END_CURSOR_MAXIMUM = SOURCE_SIZE_MAXIMUM - utf8.CHARACTER_SIZE_MINIMUM

// Trimmed_End_Cursor is where one action's content stops once its space is cut.
type Trimmed_End_Cursor Source_Cursor_Validated

// Trimmed_End_Cursor_Invariants keeps both delimiters outside that content.
func Trimmed_End_Cursor_Invariants(value Trimmed_End_Cursor, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(TRIMMED_END_CURSOR_MINIMUM),
    uint16(TRIMMED_END_CURSOR_MAXIMUM),
  ).
    Ensure()
}

// Identifier_Search_Position keeps search progress independent from its bound.
type Identifier_Search_Position uint16

// Identifier_Search_Position_Invariants bounds search progress.
func Identifier_Search_Position_Invariants(
  value Identifier_Search_Position,
  namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value),
    uint16(SOURCE_SIZE_MINIMUM),
    uint16(SOURCE_SIZE_MAXIMUM),
  ).
    Ensure()
}

// Identifier_Search_End keeps search bound independent from progress.
type Identifier_Search_End uint16

// Identifier_Search_End_Invariants bounds search termination.
func Identifier_Search_End_Invariants(value Identifier_Search_End, namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value),
    uint16(SOURCE_SIZE_MINIMUM),
    uint16(SOURCE_SIZE_MAXIMUM),
  ).
    Ensure()
}

// Identifier_Search_Span retains one nonempty lexer remainder.
type Identifier_Search_Span Identifier_Search_Fields

// Identifier_Search_Span_Invariants composes hostile search boundaries.
func Identifier_Search_Span_Invariants(value Identifier_Search_Span, namespace aver.Namespace,
) {
  Identifier_Search_Position_Storage_Invariants(&value.Position, namespace)
  Identifier_Search_End_Storage_Invariants(&value.End, namespace)
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
  value *Identifier_Search_Validated,
  namespace aver.Namespace,
) {
  Identifier_Search_Span_Invariants(value.Value, namespace)
}

// WRITTEN_OUTPUT_COUNT_MINIMUM counts the shortest byte one write commits.
const WRITTEN_OUTPUT_COUNT_MINIMUM = OUTPUT_SIZE_MINIMUM + utf8.CHARACTER_SIZE_MINIMUM

// Written_Output_Count is the output one finished write has committed.
type Written_Output_Count Output_Count_Staged

// Written_Output_Count_Invariants keeps that write inside the destination.
func Written_Output_Count_Invariants(value Written_Output_Count, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(WRITTEN_OUTPUT_COUNT_MINIMUM),
    uint16(OUTPUT_SIZE_MAXIMUM),
  ).
    Ensure()
}

// WRITTEN_CLOSE_OUTPUT_COUNT_MINIMUM counts the opening delimiter below the close.
const WRITTEN_CLOSE_OUTPUT_COUNT_MINIMUM = WRITTEN_OUTPUT_COUNT_MINIMUM +
  utf8.CHARACTER_SIZE_MINIMUM

// Written_Close_Output_Count is the output one closed composite has committed.
type Written_Close_Output_Count Output_Count_Staged

// Written_Close_Output_Count_Invariants keeps that opening below the close.
func Written_Close_Output_Count_Invariants(
  value Written_Close_Output_Count,
  namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(WRITTEN_CLOSE_OUTPUT_COUNT_MINIMUM),
    uint16(OUTPUT_SIZE_MAXIMUM),
  ).
    Ensure()
}

// Written_Child_Output_Count is the output one composite child has committed.
type Written_Child_Output_Count Output_Count_Staged

// Written_Child_Output_Count_Invariants keeps that delimiter inside the destination.
func Written_Child_Output_Count_Invariants(
  value Written_Child_Output_Count,
  namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(WRITTEN_OUTPUT_COUNT_MINIMUM),
    uint16(OUTPUT_SIZE_MAXIMUM),
  ).
    Ensure()
}

// COMMENT_LIST_TAIL_NODE_COUNT counts two actions above the list one comment joins.
const COMMENT_LIST_TAIL_NODE_COUNT = SIMPLE_ACTION_NODE_COUNT + SIMPLE_ACTION_NODE_COUNT

// FLOW_LIST_TAIL_NODE_COUNT counts two branches and one container above one flow list.
const FLOW_LIST_TAIL_NODE_COUNT = CONTROL_ACTION_TAIL_NODE_COUNT +
  SIMPLE_ACTION_CONTAINER_NODE_COUNT

// DEFINITION_LIST_TAIL_NODE_COUNT counts three branches and one action above one body.
const DEFINITION_LIST_TAIL_NODE_COUNT = CONTROL_DEFINITION_TAIL_NODE_COUNT +
  SIMPLE_ACTION_NODE_COUNT

// Comment_List_Value is the list one comment joins.
type Comment_List_Value List_State_Value_Stored

// Comment_List_Value_Invariants keeps two actions above that list.
func Comment_List_Value_Invariants(value Comment_List_Value, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Holed_Uint16(
    uint16(value.List),
    uint16(ROOT_NODE_COUNT),
    uint16(CONTROL_COMMENT_LIST_MAXIMUM),
    uint16(ACTION_LIST_HOLE),
    uint16(ACTION_LIST_HOLE),
    uint16(ACTION_LIST_HOLE),
  ).
    Range_Holed_Uint16(
    uint16(value.Last_Child),
    uint16(NODE_REFERENCE_MINIMUM),
    uint16(LIST_LAST_CHILD_MAXIMUM-COMMENT_LIST_TAIL_NODE_COUNT),
    uint16(ROOT_LIST_HOLE),
    uint16(ROOT_LIST_HOLE),
    uint16(ROOT_LIST_HOLE),
  ).
    Ensure()
}

// Comment_List_State is the list storage one comment joins.
type Comment_List_State List_State_Storage

// Comment_List_State_Invariants composes that list's storage.
func Comment_List_State_Invariants(value Comment_List_State, namespace aver.Namespace) {
  Comment_List_Value_Invariants(Comment_List_Value(value.Value), namespace)
}

// Comment_List_State_Pointer keeps mutation of that list typed.
type Comment_List_State_Pointer *Comment_List_State

// Comment_List_State_Pointer_Invariants composes one present list.
func Comment_List_State_Pointer_Invariants(
  value Comment_List_State_Pointer,
  namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Comment_List_State_Invariants(*value, namespace)
}

// Opening_List_Value is the list one control opening appends to.
type Opening_List_Value List_State_Value_Stored

// Opening_List_Value_Invariants keeps that opening's branches above the list.
func Opening_List_Value_Invariants(value Opening_List_Value, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Holed_Uint16(
    uint16(value.List),
    uint16(ROOT_NODE_COUNT),
    uint16(NESTED_OPENER_LIST_MAXIMUM),
    uint16(ACTION_LIST_HOLE),
    uint16(ACTION_LIST_HOLE),
    uint16(ACTION_LIST_HOLE),
  ).
    Range_Holed_Uint16(
    uint16(value.Last_Child),
    uint16(NODE_REFERENCE_MINIMUM),
    uint16(OPENER_TEXT_LAST_CHILD_MAXIMUM),
    uint16(ROOT_LIST_HOLE),
    uint16(ROOT_LIST_HOLE),
    uint16(ROOT_LIST_HOLE),
  ).
    Ensure()
}

// Opening_List_State is the list storage one control opening appends to.
type Opening_List_State List_State_Storage

// Opening_List_State_Invariants composes that list's storage.
func Opening_List_State_Invariants(value Opening_List_State, namespace aver.Namespace) {
  Opening_List_Value_Invariants(Opening_List_Value(value.Value), namespace)
}

// Opening_List_State_Pointer keeps mutation of that list typed.
type Opening_List_State_Pointer *Opening_List_State

// Opening_List_State_Pointer_Invariants composes one present list.
func Opening_List_State_Pointer_Invariants(
  value Opening_List_State_Pointer,
  namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Opening_List_State_Invariants(*value, namespace)
}

// Else_List_Value is the list one alternate clause appends to.
type Else_List_Value List_State_Value_Stored

// Else_List_Value_Invariants keeps that clause's branches above the list.
func Else_List_Value_Invariants(value Else_List_Value, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Holed_Uint16(
    uint16(value.List),
    uint16(ROOT_NODE_COUNT),
    uint16(UNCLOSED_ELSE_COUNT_MAXIMUM),
    uint16(ACTION_LIST_HOLE),
    uint16(ACTION_LIST_HOLE),
    uint16(ACTION_LIST_HOLE),
  ).
    Range_Holed_Uint16(
    uint16(value.Last_Child),
    uint16(NODE_REFERENCE_MINIMUM),
    uint16(STRAY_ELSE_COUNT_MAXIMUM),
    uint16(ROOT_LIST_HOLE),
    uint16(ROOT_LIST_HOLE),
    uint16(ROOT_LIST_HOLE),
  ).
    Ensure()
}

// Else_List_State is the list storage one alternate clause appends to.
type Else_List_State List_State_Storage

// Else_List_State_Invariants composes that list's storage.
func Else_List_State_Invariants(value Else_List_State, namespace aver.Namespace) {
  Else_List_Value_Invariants(Else_List_Value(value.Value), namespace)
}

// Else_List_State_Pointer keeps mutation of that list typed.
type Else_List_State_Pointer *Else_List_State

// Else_List_State_Pointer_Invariants composes one present list.
func Else_List_State_Pointer_Invariants(
  value Else_List_State_Pointer, namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Else_List_State_Invariants(*value, namespace)
}

// Flow_List_Value is the list one flow action appends to.
type Flow_List_Value List_State_Value_Stored

// Flow_List_Value_Invariants keeps that flow's branches above the list.
func Flow_List_Value_Invariants(value Flow_List_Value, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Holed_Uint16(
    uint16(value.List),
    uint16(ROOT_NODE_COUNT),
    uint16(BRANCH_FLOW_LIST_MAXIMUM),
    uint16(ACTION_LIST_HOLE),
    uint16(ACTION_LIST_HOLE),
    uint16(ACTION_LIST_HOLE),
  ).
    Range_Holed_Uint16(
    uint16(value.Last_Child),
    uint16(NODE_REFERENCE_MINIMUM),
    uint16(FLOW_TEXT_LAST_CHILD_MAXIMUM),
    uint16(ROOT_LIST_HOLE),
    uint16(ROOT_LIST_HOLE),
    uint16(ROOT_LIST_HOLE),
  ).
    Ensure()
}

// BRANCH_FLOW_TAIL_SIZE is one unclosed opener that holds one break alone.
const BRANCH_FLOW_TAIL_SIZE = CONTROL_OPENER_SIZE_MINIMUM + FLOW_ACTION_SIZE

// BRANCH_FLOW_LIST_MAXIMUM is that opener's body list behind the densest actions, one text node,
// and the branch's node, pipeline, command, and term.
const BRANCH_FLOW_LIST_MAXIMUM = ROOT_NODE_COUNT +
  (SOURCE_SIZE_MAXIMUM-BRANCH_FLOW_TAIL_SIZE)/SIMPLE_ACTION_SIZE_MINIMUM*
    SIMPLE_ACTION_NODE_COUNT +
  ((SOURCE_SIZE_MAXIMUM-BRANCH_FLOW_TAIL_SIZE)%SIMPLE_ACTION_SIZE_MINIMUM+
    SIMPLE_ACTION_SIZE_MINIMUM-
    SOURCE_POSITION_INCREMENT)/
    SIMPLE_ACTION_SIZE_MINIMUM +
  CONTROL_BRANCH_NODE_COUNT

// FLOW_TEXT_TAIL_SIZE is one text byte before one stray break.
const FLOW_TEXT_TAIL_SIZE = utf8.CHARACTER_SIZE_MINIMUM + FLOW_ACTION_SIZE

// FLOW_TEXT_LAST_CHILD_MAXIMUM is that text node behind the densest actions and one text node.
const FLOW_TEXT_LAST_CHILD_MAXIMUM = ROOT_NODE_COUNT +
  (SOURCE_SIZE_MAXIMUM-FLOW_TEXT_TAIL_SIZE)/SIMPLE_ACTION_SIZE_MINIMUM*
    SIMPLE_ACTION_NODE_COUNT +
  ((SOURCE_SIZE_MAXIMUM-FLOW_TEXT_TAIL_SIZE)%SIMPLE_ACTION_SIZE_MINIMUM+
    SIMPLE_ACTION_SIZE_MINIMUM-
    SOURCE_POSITION_INCREMENT)/
    SIMPLE_ACTION_SIZE_MINIMUM +
  NODE_COUNT_INCREMENT

// Flow_List_State is the list storage one flow action appends to.
type Flow_List_State List_State_Storage

// Flow_List_State_Invariants composes that list's storage.
func Flow_List_State_Invariants(value Flow_List_State, namespace aver.Namespace) {
  Flow_List_Value_Invariants(Flow_List_Value(value.Value), namespace)
}

// Flow_List_State_Pointer keeps mutation of that list typed.
type Flow_List_State_Pointer *Flow_List_State

// Flow_List_State_Pointer_Invariants composes one present list.
func Flow_List_State_Pointer_Invariants(
  value Flow_List_State_Pointer, namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Flow_List_State_Invariants(*value, namespace)
}

// Definition_List_Value is the list one definition body closes.
type Definition_List_Value List_State_Value_Stored

// Definition_List_Value_Invariants keeps that body's branches above the list.
func Definition_List_Value_Invariants(value Definition_List_Value, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Holed_Uint16(
    uint16(value.List),
    uint16(ROOT_NODE_COUNT),
    uint16(CONTROL_BLOCK_LIST_MAXIMUM),
    uint16(ACTION_LIST_HOLE),
    uint16(ACTION_LIST_HOLE),
    uint16(ACTION_LIST_HOLE),
  ).
    Range_Holed_Uint16(
    uint16(value.Last_Child),
    uint16(NODE_REFERENCE_MINIMUM),
    uint16(LIST_LAST_CHILD_MAXIMUM-DEFINITION_LIST_TAIL_NODE_COUNT),
    uint16(ROOT_LIST_HOLE),
    uint16(ROOT_LIST_HOLE),
    uint16(ROOT_LIST_HOLE),
  ).
    Ensure()
}

// Definition_List_State is the list storage one definition body closes.
type Definition_List_State List_State_Storage

// Definition_List_State_Invariants composes that list's storage.
func Definition_List_State_Invariants(value Definition_List_State, namespace aver.Namespace) {
  Definition_List_Value_Invariants(Definition_List_Value(value.Value), namespace)
}

// Definition_List_State_Pointer keeps mutation of that list typed.
type Definition_List_State_Pointer *Definition_List_State

// Definition_List_State_Pointer_Invariants composes one present list.
func Definition_List_State_Pointer_Invariants(
  value Definition_List_State_Pointer,
  namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Definition_List_State_Invariants(*value, namespace)
}

// Block_List_Value is the list one named block opens against.
type Block_List_Value List_State_Value_Stored

// Block_List_Value_Invariants keeps that block's branches above the list.
func Block_List_Value_Invariants(value Block_List_Value, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Holed_Uint16(
    uint16(value.List),
    uint16(ROOT_NODE_COUNT),
    uint16(STRAY_BLOCK_LIST_MAXIMUM),
    uint16(ACTION_LIST_HOLE),
    uint16(ACTION_LIST_HOLE),
    uint16(ACTION_LIST_HOLE),
  ).
    Range_Holed_Uint16(
    uint16(value.Last_Child),
    uint16(NODE_REFERENCE_MINIMUM),
    uint16(STRAY_BLOCK_LAST_CHILD_MAXIMUM),
    uint16(ROOT_LIST_HOLE),
    uint16(ROOT_LIST_HOLE),
    uint16(ROOT_LIST_HOLE),
  ).
    Ensure()
}

// CONTROL_OPENER_SIZE_MINIMUM is the shortest action that opens one branch.
const CONTROL_OPENER_SIZE_MINIMUM = KEYWORD_SOURCE_SIZE_MINIMUM +
  ACTION_SEPARATOR_SIZE_MINIMUM +
  DOT_SOURCE_SIZE

// CONTROL_OPENER_COUNT_MAXIMUM packs the densest actions and one text node before that opener,
// which a hostile source may leave unclosed.
const CONTROL_OPENER_COUNT_MAXIMUM = ROOT_NODE_COUNT +
  (SOURCE_SIZE_MAXIMUM-CONTROL_OPENER_SIZE_MINIMUM)/
    SIMPLE_ACTION_SIZE_MINIMUM*
    SIMPLE_ACTION_NODE_COUNT +
  ((SOURCE_SIZE_MAXIMUM-CONTROL_OPENER_SIZE_MINIMUM)%SIMPLE_ACTION_SIZE_MINIMUM+
    SIMPLE_ACTION_SIZE_MINIMUM-
    SOURCE_POSITION_INCREMENT)/
    SIMPLE_ACTION_SIZE_MINIMUM

// CONTROL_OPENED_COUNT_MAXIMUM adds that opener's branch.
const CONTROL_OPENED_COUNT_MAXIMUM = CONTROL_OPENER_COUNT_MAXIMUM + CONTROL_BRANCH_NODE_COUNT

// NESTED_OPENER_TAIL_SIZE is two shortest openers a hostile source leaves unclosed.
const NESTED_OPENER_TAIL_SIZE = CONTROL_OPENER_SIZE_MINIMUM + CONTROL_OPENER_SIZE_MINIMUM

// NESTED_OPENER_LIST_MAXIMUM is the outer opener's body list behind the densest actions, one text
// node, and that branch's node, pipeline, command, and term.
const NESTED_OPENER_LIST_MAXIMUM = ROOT_NODE_COUNT +
  (SOURCE_SIZE_MAXIMUM-NESTED_OPENER_TAIL_SIZE)/SIMPLE_ACTION_SIZE_MINIMUM*
    SIMPLE_ACTION_NODE_COUNT +
  ((SOURCE_SIZE_MAXIMUM-NESTED_OPENER_TAIL_SIZE)%SIMPLE_ACTION_SIZE_MINIMUM+
    SIMPLE_ACTION_SIZE_MINIMUM-
    SOURCE_POSITION_INCREMENT)/
    SIMPLE_ACTION_SIZE_MINIMUM +
  CONTROL_BRANCH_NODE_COUNT

// OPENER_TEXT_TAIL_SIZE is one text byte before one shortest opener.
const OPENER_TEXT_TAIL_SIZE = utf8.CHARACTER_SIZE_MINIMUM + CONTROL_OPENER_SIZE_MINIMUM

// OPENER_TEXT_LAST_CHILD_MAXIMUM is that text node behind the densest actions and one text node.
const OPENER_TEXT_LAST_CHILD_MAXIMUM = ROOT_NODE_COUNT +
  (SOURCE_SIZE_MAXIMUM-OPENER_TEXT_TAIL_SIZE)/SIMPLE_ACTION_SIZE_MINIMUM*
    SIMPLE_ACTION_NODE_COUNT +
  ((SOURCE_SIZE_MAXIMUM-OPENER_TEXT_TAIL_SIZE)%SIMPLE_ACTION_SIZE_MINIMUM+
    SIMPLE_ACTION_SIZE_MINIMUM-
    SOURCE_POSITION_INCREMENT)/
    SIMPLE_ACTION_SIZE_MINIMUM +
  NODE_COUNT_INCREMENT

// ELSE_ACTION_SIZE is the action one else keyword spells alone.
const ELSE_ACTION_SIZE = ACTION_BOUNDARY_SIZE_MINIMUM + len(KEYWORD_ELSE)

// STRAY_ELSE_COUNT_MAXIMUM packs the densest actions and one text node before one else keyword a
// hostile source leaves without a branch.
const STRAY_ELSE_COUNT_MAXIMUM = ROOT_NODE_COUNT +
  (SOURCE_SIZE_MAXIMUM-ELSE_ACTION_SIZE)/SIMPLE_ACTION_SIZE_MINIMUM*SIMPLE_ACTION_NODE_COUNT +
  ((SOURCE_SIZE_MAXIMUM-ELSE_ACTION_SIZE)%SIMPLE_ACTION_SIZE_MINIMUM+
    SIMPLE_ACTION_SIZE_MINIMUM-
    SOURCE_POSITION_INCREMENT)/
    SIMPLE_ACTION_SIZE_MINIMUM

// UNCLOSED_ELSE_BRANCH_MAXIMUM is the branch that unclosed alternate reads.
const UNCLOSED_ELSE_BRANCH_MAXIMUM = UNCLOSED_ELSE_COUNT_MAXIMUM -
  CONTROL_BRANCH_NODE_COUNT +
  NODE_COUNT_INCREMENT

// ALTERNATE_KEYWORD_LAST_START_MAXIMUM is the else keyword of one unclosed alternate branch that
// ends the action content.
const ALTERNATE_KEYWORD_LAST_START_MAXIMUM = ACTION_CONTENT_BOUNDARY_MAXIMUM -
  len(KEYWORD_ELSE) -
  ACTION_SEPARATOR_SIZE_MINIMUM -
  len(KEYWORD_IF) -
  ACTION_SEPARATOR_SIZE_MINIMUM -
  DOT_SOURCE_SIZE

// UNCLOSED_ELSE_TAIL_SIZE is the shortest branch and alternate a hostile source leaves unclosed.
const UNCLOSED_ELSE_TAIL_SIZE = CONTROL_OPENER_SIZE_MINIMUM +
  ACTION_BOUNDARY_SIZE_MINIMUM +
  len(KEYWORD_ELSE)

// UNCLOSED_ELSE_COUNT_MAXIMUM packs the densest actions and one text node before that branch,
// then the branch itself, whose body list the alternate reads.
const UNCLOSED_ELSE_COUNT_MAXIMUM = ROOT_NODE_COUNT +
  (SOURCE_SIZE_MAXIMUM-UNCLOSED_ELSE_TAIL_SIZE)/SIMPLE_ACTION_SIZE_MINIMUM*
    SIMPLE_ACTION_NODE_COUNT +
  ((SOURCE_SIZE_MAXIMUM-UNCLOSED_ELSE_TAIL_SIZE)%SIMPLE_ACTION_SIZE_MINIMUM+
    SIMPLE_ACTION_SIZE_MINIMUM-
    SOURCE_POSITION_INCREMENT)/
    SIMPLE_ACTION_SIZE_MINIMUM +
  CONTROL_BRANCH_NODE_COUNT

// STRAY_BLOCK_TAIL_SIZE is the shortest branch that holds one template keyword alone.
const STRAY_BLOCK_TAIL_SIZE = CONTROL_OPENER_SIZE_MINIMUM + TEMPLATE_KEYWORD_ACTION_SIZE_MINIMUM

// STRAY_BLOCK_LIST_MAXIMUM is that branch's body list behind the densest actions, one text node,
// and the branch's node, pipeline, command, and term.
const STRAY_BLOCK_LIST_MAXIMUM = ROOT_NODE_COUNT +
  (SOURCE_SIZE_MAXIMUM-STRAY_BLOCK_TAIL_SIZE)/SIMPLE_ACTION_SIZE_MINIMUM*
    SIMPLE_ACTION_NODE_COUNT +
  ((SOURCE_SIZE_MAXIMUM-STRAY_BLOCK_TAIL_SIZE)%SIMPLE_ACTION_SIZE_MINIMUM+
    SIMPLE_ACTION_SIZE_MINIMUM-
    SOURCE_POSITION_INCREMENT)/
    SIMPLE_ACTION_SIZE_MINIMUM +
  CONTROL_BRANCH_NODE_COUNT

// STRAY_TEXT_TAIL_SIZE is one text byte before one template keyword alone.
const STRAY_TEXT_TAIL_SIZE = utf8.CHARACTER_SIZE_MINIMUM + TEMPLATE_KEYWORD_ACTION_SIZE_MINIMUM

// STRAY_BLOCK_LAST_CHILD_MAXIMUM is that text node behind the densest actions and one text node.
const STRAY_BLOCK_LAST_CHILD_MAXIMUM = ROOT_NODE_COUNT +
  (SOURCE_SIZE_MAXIMUM-STRAY_TEXT_TAIL_SIZE)/SIMPLE_ACTION_SIZE_MINIMUM*
    SIMPLE_ACTION_NODE_COUNT +
  ((SOURCE_SIZE_MAXIMUM-STRAY_TEXT_TAIL_SIZE)%SIMPLE_ACTION_SIZE_MINIMUM+
    SIMPLE_ACTION_SIZE_MINIMUM-
    SOURCE_POSITION_INCREMENT)/
    SIMPLE_ACTION_SIZE_MINIMUM +
  NODE_COUNT_INCREMENT

// Invoked_Block_List_Value is the list one block invocation joins once its end has room.
type Invoked_Block_List_Value List_State_Value_Stored

// Invoked_Block_List_Value_Invariants keeps that block's tail above the list.
func Invoked_Block_List_Value_Invariants(
  value Invoked_Block_List_Value, namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Holed_Uint16(
    uint16(value.List),
    uint16(ROOT_NODE_COUNT),
    uint16(CONTROL_BLOCK_LIST_MAXIMUM),
    uint16(ACTION_LIST_HOLE),
    uint16(ACTION_LIST_HOLE),
    uint16(ACTION_LIST_HOLE),
  ).
    Range_Holed_Uint16(
    uint16(value.Last_Child),
    uint16(NODE_REFERENCE_MINIMUM),
    uint16(LIST_LAST_CHILD_MAXIMUM-BLOCK_TAIL_NODE_COUNT),
    uint16(ROOT_LIST_HOLE),
    uint16(ROOT_LIST_HOLE),
    uint16(ROOT_LIST_HOLE),
  ).
    Ensure()
}

// Invoked_Block_List_State is that list storage.
type Invoked_Block_List_State List_State_Storage

// Invoked_Block_List_State_Invariants composes that list.
func Invoked_Block_List_State_Invariants(
  value Invoked_Block_List_State, namespace aver.Namespace,
) {
  Invoked_Block_List_Value_Invariants(Invoked_Block_List_Value(value.Value), namespace)
}

// Invoked_Block_List_State_Pointer keeps mutation of that list typed.
type Invoked_Block_List_State_Pointer *Invoked_Block_List_State

// Invoked_Block_List_State_Pointer_Invariants composes one present list.
func Invoked_Block_List_State_Pointer_Invariants(
  value Invoked_Block_List_State_Pointer,
  namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Invoked_Block_List_State_Invariants(*value, namespace)
}

// Block_List_State is the list storage one named block opens against.
type Block_List_State List_State_Storage

// Block_List_State_Invariants composes that list's storage.
func Block_List_State_Invariants(value Block_List_State, namespace aver.Namespace) {
  Block_List_Value_Invariants(Block_List_Value(value.Value), namespace)
}

// Block_List_State_Pointer keeps mutation of that list typed.
type Block_List_State_Pointer *Block_List_State

// Block_List_State_Pointer_Invariants composes one present list.
func Block_List_State_Pointer_Invariants(
  value Block_List_State_Pointer, namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Block_List_State_Invariants(*value, namespace)
}

// Invocation_List_Value is the list one template invocation joins.
type Invocation_List_Value List_State_Value_Stored

// Invocation_List_Value_Invariants keeps that invocation's tail above the list.
func Invocation_List_Value_Invariants(value Invocation_List_Value, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Holed_Uint16(
    uint16(value.List),
    uint16(ROOT_NODE_COUNT),
    uint16(CONTROL_INVOCATION_LIST_MAXIMUM),
    uint16(ACTION_LIST_HOLE),
    uint16(ACTION_LIST_HOLE),
    uint16(ACTION_LIST_HOLE),
  ).
    Range_Holed_Uint16(
    uint16(value.Last_Child),
    uint16(NODE_REFERENCE_MINIMUM),
    uint16(INVOCATION_LIST_LAST_CHILD_MAXIMUM),
    uint16(ROOT_LIST_HOLE),
    uint16(ROOT_LIST_HOLE),
    uint16(ROOT_LIST_HOLE),
  ).
    Ensure()
}

// Invocation_List_State is the list storage one template invocation joins.
type Invocation_List_State List_State_Storage

// Invocation_List_State_Invariants composes that list's storage.
func Invocation_List_State_Invariants(value Invocation_List_State, namespace aver.Namespace) {
  Invocation_List_Value_Invariants(Invocation_List_Value(value.Value), namespace)
}

// Invocation_List_State_Pointer keeps mutation of that list typed.
type Invocation_List_State_Pointer *Invocation_List_State

// Invocation_List_State_Pointer_Invariants composes one present list.
func Invocation_List_State_Pointer_Invariants(
  value Invocation_List_State_Pointer,
  namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Invocation_List_State_Invariants(*value, namespace)
}

// BRANCH_ALLOCATED_LINK_MAXIMUM is the branch one unclosed opener fills.
const BRANCH_ALLOCATED_LINK_MAXIMUM = BRANCH_ALLOCATION_LINK_MAXIMUM

// Branch_Allocated_Node carries proof of the branch one clause fills.
type Branch_Allocated_Node Allocated_Node_Validated

// Branch_Allocated_Node_Invariants keeps three branches above that branch.
func Branch_Allocated_Node_Invariants(value *Branch_Allocated_Node, namespace aver.Namespace) {
  aver.Tree(*value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(APPENDED_CHILD_REFERENCE_MINIMUM),
    uint16(BRANCH_ALLOCATED_LINK_MAXIMUM),
  ).
    Ensure()
}

// DEFINITION_ALLOCATED_LINK_MAXIMUM leaves one body's branches above its definition.
const DEFINITION_ALLOCATED_LINK_MAXIMUM = NODE_REFERENCE_MAXIMUM -
  DEFINITION_LIST_TAIL_NODE_COUNT

// Definition_Allocated_Node carries proof of the definition one body closes.
type Definition_Allocated_Node Allocated_Node_Validated

// Definition_Allocated_Node_Invariants keeps that body's branches above the definition.
func Definition_Allocated_Node_Invariants(
  value *Definition_Allocated_Node,
  namespace aver.Namespace,
) {
  aver.Tree(*value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(APPENDED_CHILD_REFERENCE_MINIMUM),
    uint16(DEFINITION_ALLOCATED_LINK_MAXIMUM),
  ).
    Ensure()
}

// Invocation_Node_Count is the arena length one template invocation leaves.
type Invocation_Node_Count Node_Count_Allocated

// Invocation_Node_Count_Invariants keeps that tail inside the arena.
func Invocation_Node_Count_Invariants(value Invocation_Node_Count, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(ROOT_NODE_COUNT),
    uint16(INVOKING_NODE_COUNT_MAXIMUM),
  ).
    Ensure()
}

// Invocation_Argument_Node_Count is the arena length one invocation argument leaves.
type Invocation_Argument_Node_Count Dense_Action_Node_Count

// Invocation_Argument_Node_Count_Invariants keeps that tail inside the arena.
func Invocation_Argument_Node_Count_Invariants(
  value Invocation_Argument_Node_Count,
  namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(ACTION_NODE_COUNT_MINIMUM),
    uint16(INVOKING_NODE_COUNT_MAXIMUM),
  ).
    Ensure()
}

// Emptied_Template_Reference names one definition a merge checks for an empty body.
type Emptied_Template_Reference Template_Reference_Validated

// Emptied_Template_Reference_Invariants keeps the packed definitions above that one.
func Emptied_Template_Reference_Invariants(
  value Emptied_Template_Reference,
  namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(ACTION_NODE_COUNT_MINIMUM),
    uint16(NAMES_EQUAL_REFERENCE_MAXIMUM),
  ).
    Ensure()
}

// Kinded_Value_Unvalidated is one caller value whose kind already passed validation.
type Kinded_Value_Unvalidated Value_Unvalidated

// Kinded_Value_Unvalidated_Invariants keeps that kind inside the known range.
func Kinded_Value_Unvalidated_Invariants(
  value Kinded_Value_Unvalidated, namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint8(
    uint8(value.Kind),
    uint8(VALUE_NIL),
    uint8(VALUE_FUNCTION),
  ).
    Range_Uint64(
    uint64(value.Real),
    bits.WORD_64_MINIMUM,
    bits.WORD_64_MAXIMUM,
  ).
    Range_Uint64(
    uint64(value.Imaginary),
    bits.WORD_64_MINIMUM,
    bits.WORD_64_MAXIMUM,
  ).
    Range_Int(
    len(value.Text),
    OUTPUT_SIZE_MINIMUM,
    SOURCE_SIZE_UNVALIDATED_MAXIMUM,
  ).
    Range_Int(
    len(value.Values),
    bytes.SLICE_SIZE_MINIMUM,
    VALUE_COUNT_UNVALIDATED_MAXIMUM,
  ).
    Range_Int(
    len(value.Fields),
    bytes.SLICE_SIZE_MINIMUM,
    VALUE_COUNT_UNVALIDATED_MAXIMUM,
  ).
    Sometimes(
    value.Function != nil,
    "Kinded caller value holds a callable.",
  ).
    Ensure()
}

// Chosen_Map_Selection is one map walk already picked.
type Chosen_Map_Selection Map_Selection

// Chosen_Map_Selection_Invariants admits every walk but the unpicked one.
func Chosen_Map_Selection_Invariants(value Chosen_Map_Selection, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_3_Uint8(
    uint8(value),
    uint8(MAP_SELECTION_FORWARD),
    uint8(MAP_SELECTION_BACKWARD),
    uint8(MAP_SELECTION_SCAN),
  ).
    Ensure()
}

// Simple_Classified_Token_Kind carries one kind a single byte selects.
type Simple_Classified_Token_Kind Classified_Token_Kind

// Simple_Classified_Token_Kind_Invariants admits every one-byte kind, and no Boolean word.
func Simple_Classified_Token_Kind_Invariants(
  value Simple_Classified_Token_Kind,
  namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Holed_Uint8(
    uint8(value.Value),
    uint8(TOKEN_KIND_MINIMUM),
    uint8(TOKEN_RIGHT_PAREN),
    uint8(TOKEN_BOOLEAN),
    uint8(TOKEN_BOOLEAN),
    uint8(TOKEN_BOOLEAN),
  ).
    Ensure()
}

// Branch_Node is one control record whose body a condition selects.
type Branch_Node Node

// Branch_Node_Invariants bounds the record and admits the two branches that select a body.
func Branch_Node_Invariants(value Branch_Node, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_Uint8(uint8(value.Kind), uint8(NODE_IF), uint8(NODE_WITH)).
    Range_Uint16(
    uint16(value.Start),
    uint16(SOURCE_SIZE_MINIMUM),
    uint16(SOURCE_SIZE_MAXIMUM-CONTROL_ACTION_SIZE_MINIMUM),
  ).
    Range_Uint16(
    uint16(value.End),
    uint16(CONTROL_ACTION_SIZE_MINIMUM),
    uint16(SOURCE_SIZE_MAXIMUM),
  ).
    Range_Uint16(
    uint16(value.Value_Start),
    uint16(CUSTOM_DELIMITER_SIZE_MINIMUM),
    uint16(SOURCE_SIZE_MAXIMUM-CONTROL_ACTION_SIZE_MINIMUM+
      CUSTOM_DELIMITER_SIZE_MINIMUM),
  ).
    Range_Uint16(
    uint16(value.Value_End),
    uint16(CUSTOM_DELIMITER_SIZE_MINIMUM+len(KEYWORD_IF)),
    uint16(
      SOURCE_SIZE_MAXIMUM-CONTROL_ACTION_SIZE_MINIMUM+
        CUSTOM_DELIMITER_SIZE_MINIMUM+len(KEYWORD_IF),
    ),
  ).
    Range_Uint16(
    uint16(value.First_Child),
    uint16(FIRST_TERM_REFERENCE),
    uint16(NODE_REFERENCE_MAXIMUM-CONTROL_BODY_TAIL_NODE_COUNT),
  ).
    Range_Holed_Uint16(
    uint16(value.Next_Sibling),
    uint16(NODE_REFERENCE_MINIMUM),
    uint16(CONTROL_SIBLING_REFERENCE_MAXIMUM),
    uint16(ROOT_LIST_HOLE),
    uint16(TERM_ACTION_HOLE),
    uint16(TERM_ACTION_HOLE),
  ).
    Ensure()
}

// Branch_Node_Stored keeps one branch record opaque inside the evaluator.
type Branch_Node_Stored *Branch_Node

// Branch_Node_Stored_Invariants checks the present branch record.
func Branch_Node_Stored_Invariants(value Branch_Node_Stored, namespace aver.Namespace) {
  if value == nil {
    return
  }
  Branch_Node_Invariants(*value, namespace)
}

// Branch_Node_Validated carries one branch record into body selection.
type Branch_Node_Validated struct {
  // Value keeps deep proof attached to the branch record.
  Value Branch_Node_Stored
}

// Branch_Node_Validated_Invariants checks transport ownership of that record.
func Branch_Node_Validated_Invariants(value *Branch_Node_Validated, namespace aver.Namespace) {
  Branch_Node_Stored_Invariants(value.Value, namespace)
}

// RANGE_BOUND_INDEX_MINIMUM counts the element one bind already took.
const RANGE_BOUND_INDEX_MINIMUM = bytes.SLICE_SIZE_MINIMUM + 1

// Bound_Range_Frame_State is traversal state whose cursor holds a ranged value.
type Bound_Range_Frame_State Frame_Stored

// Bound_Range_Frame_State_Invariants fixes traversal state beside a ranged cursor.
func Bound_Range_Frame_State_Invariants(value Bound_Range_Frame_State, namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Holed_Uint16(
    uint16(value.Reference),
    uint16(NODE_REFERENCE_MINIMUM),
    uint16(RANGE_BODY_REFERENCE_MAXIMUM),
    uint16(ROOT_LIST_HOLE),
    uint16(TERM_ACTION_HOLE),
    uint16(TERM_ACTION_HOLE),
  ).
    Range_Uint16(
    uint16(value.Index),
    uint16(RANGE_BOUND_INDEX_MINIMUM),
    uint16(
      VALUE_COUNT_MAXIMUM,
    ),
  ).
    Range_Uint16(
    uint16(value.Map_Entry_Index),
    uint16(bytes.SLICE_SIZE_MINIMUM),
    uint16(RANGE_MAP_ENTRY_INDEX_MAXIMUM),
  ).
    Enum_4_Uint8(
    uint8(value.Map_Selected),
    uint8(MAP_SELECTION_NONE),
    uint8(MAP_SELECTION_FORWARD),
    uint8(MAP_SELECTION_BACKWARD),
    uint8(MAP_SELECTION_SCAN),
  ).
    Range_Uint16(
    uint16(value.Variable_Mark),
    uint16(VARIABLE_COUNT_INCREMENT),
    uint16(VARIABLE_COUNT_MAXIMUM),
  ).
    Range_Uint16(
    uint16(value.Iteration_Variables),
    uint16(VARIABLE_COUNT_INCREMENT),
    uint16(VARIABLE_COUNT_MAXIMUM),
  ).
    Range_Uint16(
    uint16(value.Scope_Base),
    uint16(bytes.SLICE_SIZE_MINIMUM),
    uint16(RANGE_SCOPE_BASE_MAXIMUM),
  ).
    Range_Uint16(
    uint16(value.Previous_Scope_Base),
    uint16(bytes.SLICE_SIZE_MINIMUM),
    uint16(RANGE_SCOPE_BASE_MAXIMUM),
  ).
    Enum_3_Uint8(
    uint8(value.Declaration_Count),
    uint8(DECLARATION_COUNT_NONE),
    uint8(
      DECLARATION_COUNT_ONE,
    ),
    uint8(DECLARATION_COUNT_TWO),
  ).
    Sometimes(
    bool(value.Assignment),
    "Pipeline updates existing variable.",
  ).
    Ensure()
  aver.Always(
    bool(value.Restore) == RANGE_FRAME_RESTORE,
    "One bound frame restores the scope its list opened.",
  )
  aver.Always(
    uint8(value.Kind) == uint8(FRAME_RANGE),
    "One bound frame keeps the kind its caller selected.",
  )
  Ranged_Dot_Value_Invariants(Ranged_Dot_Value(value.Dot), namespace)
  Ranged_Declaration_References_Invariants(
    Ranged_Declaration_References(value.Declarations),
    namespace,
  )
}

// Bound_Range_Frame_Storage is caller frame storage a range owns.
type Bound_Range_Frame_Storage Frame_Storage

// Bound_Range_Frame_Storage_Invariants checks ownership of that frame storage.
func Bound_Range_Frame_Storage_Invariants(
  value Bound_Range_Frame_Storage,
  namespace aver.Namespace,
) {
  Bound_Range_Frame_State_Invariants(Bound_Range_Frame_State(value.Value), namespace)
}

// Bound_Range_Frame_Pointer keeps mutation of one range frame typed.
type Bound_Range_Frame_Pointer *Bound_Range_Frame_Storage

// Bound_Range_Frame_Pointer_Invariants composes one present range frame.
func Bound_Range_Frame_Pointer_Invariants(
  value Bound_Range_Frame_Pointer,
  namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Bound_Range_Frame_Storage_Invariants(*value, namespace)
}

// QUOTED_LAST_START_MAXIMUM leaves both quotes and the right delimiter past one token.
const QUOTED_LAST_START_MAXIMUM = TOKEN_QUOTED_STORED_START_MAXIMUM -
  utf8.CHARACTER_SIZE_MINIMUM

// Quoted_Last_Start is where the token before one quoted lexeme began.
type Quoted_Last_Start Cursor_Last_Start_Stored

// Quoted_Last_Start_Invariants keeps that token before the quoted lexeme.
func Quoted_Last_Start_Invariants(value Quoted_Last_Start, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(SOURCE_SIZE_MINIMUM),
    uint16(QUOTED_LAST_START_MAXIMUM),
  ).
    Ensure()
}

// Variable_Last_Start is where the token before one pipeline variable began.
type Variable_Last_Start Cursor_Last_Start_Stored

// Variable_Last_Start_Invariants keeps that token before the variable's one byte.
func Variable_Last_Start_Invariants(value Variable_Last_Start, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(SOURCE_SIZE_MINIMUM),
    uint16(LEXEME_START_MAXIMUM-utf8.CHARACTER_SIZE_MINIMUM),
  ).
    Ensure()
}

// COMMAND_TAIL_REFERENCE_MAXIMUM is the latest reference one command and its term still follow.
const COMMAND_TAIL_REFERENCE_MAXIMUM = NODE_REFERENCE_MAXIMUM -
  COMMAND_NODE_COUNT -
  SCALAR_NODE_COUNT

// CONTAINER_TAIL_REFERENCE_MAXIMUM is the latest reference one pipeline, command, and term
// still follow.
const CONTAINER_TAIL_REFERENCE_MAXIMUM = NODE_REFERENCE_MAXIMUM -
  SIMPLE_ACTION_CONTAINER_NODE_COUNT

// ACTION_TAIL_REFERENCE_MAXIMUM is the latest reference one shortest action's nodes still
// follow.
const ACTION_TAIL_REFERENCE_MAXIMUM = NODE_REFERENCE_MAXIMUM - SIMPLE_ACTION_NODE_COUNT

// ACTION_TERM_TAIL_REFERENCE_MAXIMUM is the latest reference one shortest action's nodes and
// one more term still follow.
const ACTION_TERM_TAIL_REFERENCE_MAXIMUM = ACTION_TAIL_REFERENCE_MAXIMUM - SCALAR_NODE_COUNT

// ACTION_COMMAND_TAIL_REFERENCE_MAXIMUM is the latest reference one shortest action's nodes and
// one more command with its term still follow.
const ACTION_COMMAND_TAIL_REFERENCE_MAXIMUM = ACTION_TAIL_REFERENCE_MAXIMUM -
  COMMAND_NODE_COUNT -
  SCALAR_NODE_COUNT

// Variable_Pipeline_Frame_State is the frame one pipeline variable reads.
type Variable_Pipeline_Frame_State Pipeline_Frame_Stored

// Variable_Pipeline_Frame_State_Invariants leaves the variable's own term past each reference.
func Variable_Pipeline_Frame_State_Invariants(
  value Variable_Pipeline_Frame_State,
  namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Pipe),
    uint16(PIPELINE_NODE_COUNT_MINIMUM),
    uint16(PIPELINE_REFERENCE_MAXIMUM),
  ).
    Range_Holed_Uint16(
    uint16(value.Last_Child),
    uint16(NODE_REFERENCE_MINIMUM),
    uint16(ACTION_TAIL_REFERENCE_MAXIMUM),
    uint16(ROOT_LIST_HOLE),
    uint16(TERM_ACTION_HOLE),
    uint16(TERM_PIPELINE_HOLE),
  ).
    Range_Holed_Uint16(
    uint16(value.Command),
    uint16(NODE_REFERENCE_MINIMUM),
    uint16(ACTION_TAIL_REFERENCE_MAXIMUM),
    uint16(ROOT_LIST_HOLE),
    uint16(TERM_ACTION_HOLE),
    uint16(TERM_PIPELINE_HOLE),
  ).
    Range_Holed_Uint16(
    uint16(value.Last_Argument),
    uint16(NODE_REFERENCE_MINIMUM),
    uint16(ACTION_TERM_TAIL_REFERENCE_MAXIMUM),
    uint16(ROOT_LIST_HOLE),
    uint16(TERM_ACTION_HOLE),
    uint16(TERM_PIPELINE_HOLE),
  ).
    Ensure()
}

// Variable_Pipeline_Frame is the frame storage one pipeline variable reads.
type Variable_Pipeline_Frame Pipeline_Frame_Storage

// Variable_Pipeline_Frame_Invariants composes that frame's state.
func Variable_Pipeline_Frame_Invariants(
  value Variable_Pipeline_Frame, namespace aver.Namespace,
) {
  Variable_Pipeline_Frame_State_Invariants(
    Variable_Pipeline_Frame_State(value.Value), namespace,
  )
}

// Variable_Pipeline_Frame_Pointer keeps that frame mutable for its caller.
type Variable_Pipeline_Frame_Pointer *Variable_Pipeline_Frame

// Variable_Pipeline_Frame_Pointer_Invariants composes present frame storage.
func Variable_Pipeline_Frame_Pointer_Invariants(
  value Variable_Pipeline_Frame_Pointer,
  namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Variable_Pipeline_Frame_Invariants(*value, namespace)
}

// CONTROL_SIBLING_TAIL_SIZE is the shortest branch and the one text byte past it.
const CONTROL_SIBLING_TAIL_SIZE = CONTROL_ACTION_SIZE_MINIMUM + utf8.CHARACTER_SIZE_MINIMUM

// CONTROL_SIBLING_REFERENCE_MAXIMUM packs the densest actions and one text node before that
// branch, then its node, pipeline, command, term, body list, and the text past it.
const CONTROL_SIBLING_REFERENCE_MAXIMUM = ROOT_NODE_COUNT +
  (SOURCE_SIZE_MAXIMUM-CONTROL_SIBLING_TAIL_SIZE)/
    SIMPLE_ACTION_SIZE_MINIMUM*
    SIMPLE_ACTION_NODE_COUNT +
  ((SOURCE_SIZE_MAXIMUM-CONTROL_SIBLING_TAIL_SIZE)%SIMPLE_ACTION_SIZE_MINIMUM+
    SIMPLE_ACTION_SIZE_MINIMUM-
    SOURCE_POSITION_INCREMENT)/
    SIMPLE_ACTION_SIZE_MINIMUM +
  SIMPLE_ACTION_NODE_COUNT +
  NODE_COUNT_INCREMENT +
  NODE_COUNT_INCREMENT

// RANGE_ELSE_TAIL_SIZE is the shortest range whose body an alternate list follows.
const RANGE_ELSE_TAIL_SIZE = CONTROL_ACTION_SIZE_MINIMUM +
  len(KEYWORD_RANGE) -
  len(KEYWORD_IF) +
  ACTION_BOUNDARY_SIZE_MINIMUM +
  len(KEYWORD_ELSE)

// RANGE_ELSE_REFERENCE_MAXIMUM packs the densest actions and one text node before that range,
// then its node, pipeline, command, term, body list, and the alternate list.
const RANGE_ELSE_REFERENCE_MAXIMUM = ROOT_NODE_COUNT +
  (SOURCE_SIZE_MAXIMUM-RANGE_ELSE_TAIL_SIZE)/SIMPLE_ACTION_SIZE_MINIMUM*
    SIMPLE_ACTION_NODE_COUNT +
  ((SOURCE_SIZE_MAXIMUM-RANGE_ELSE_TAIL_SIZE)%SIMPLE_ACTION_SIZE_MINIMUM+
    SIMPLE_ACTION_SIZE_MINIMUM-
    SOURCE_POSITION_INCREMENT)/
    SIMPLE_ACTION_SIZE_MINIMUM +
  SIMPLE_ACTION_NODE_COUNT +
  NODE_COUNT_INCREMENT +
  NODE_COUNT_INCREMENT

// BLOCK_DEFINITION_PAIR_SIZE is the shortest block followed by the shortest definition.
const BLOCK_DEFINITION_PAIR_SIZE = BLOCK_DEFINITION_SIZE_MINIMUM +
  TEMPLATE_DEFINITION_SIZE_MINIMUM

// BLOCK_DEFINITION_PAIR_PACKED_COUNT packs the densest actions and one text node before that pair.
const BLOCK_DEFINITION_PAIR_PACKED_COUNT = ROOT_NODE_COUNT +
  (SOURCE_SIZE_MAXIMUM-BLOCK_DEFINITION_PAIR_SIZE)/
    SIMPLE_ACTION_SIZE_MINIMUM*
    SIMPLE_ACTION_NODE_COUNT +
  ((SOURCE_SIZE_MAXIMUM-BLOCK_DEFINITION_PAIR_SIZE)%SIMPLE_ACTION_SIZE_MINIMUM+
    SIMPLE_ACTION_SIZE_MINIMUM-
    SOURCE_POSITION_INCREMENT)/
    SIMPLE_ACTION_SIZE_MINIMUM

// EARLIER_TEMPLATE_REFERENCE_MAXIMUM is that block's definition, the latest one another
// definition still follows.
const EARLIER_TEMPLATE_REFERENCE_MAXIMUM = BLOCK_DEFINITION_PAIR_PACKED_COUNT +
  SIMPLE_ACTION_NODE_COUNT +
  NODE_COUNT_INCREMENT

// LATER_TEMPLATE_REFERENCE_MAXIMUM is the definition past that block's body list.
const LATER_TEMPLATE_REFERENCE_MAXIMUM = EARLIER_TEMPLATE_REFERENCE_MAXIMUM +
  NODE_COUNT_INCREMENT +
  NODE_COUNT_INCREMENT

// NAMED_PAIR_EARLIER_REFERENCE_MAXIMUM is that block's definition when the later definition
// takes a one-byte name.
const NAMED_PAIR_EARLIER_REFERENCE_MAXIMUM = EARLIER_TEMPLATE_REFERENCE_MAXIMUM -
  NODE_COUNT_INCREMENT

// NAMED_PAIR_LATER_REFERENCE_MAXIMUM is that later named definition.
const NAMED_PAIR_LATER_REFERENCE_MAXIMUM = LATER_TEMPLATE_REFERENCE_MAXIMUM -
  NODE_COUNT_INCREMENT

// Prior_Template_First is the chain head one later definition opens beside.
type Prior_Template_First Template_First_Storage

// Prior_Template_First_Invariants keeps that head at or before the latest earlier definition.
func Prior_Template_First_Invariants(value Prior_Template_First, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Holed_Uint16(
    uint16(value.Value),
    uint16(NODE_REFERENCE_MINIMUM),
    uint16(EARLIER_TEMPLATE_REFERENCE_MAXIMUM),
    uint16(ROOT_LIST_HOLE),
    uint16(ROOT_LIST_HOLE),
    uint16(ROOT_LIST_HOLE),
  ).
    Ensure()
}

// Prior_Template_Last is the chain tail one later definition opens beside.
type Prior_Template_Last Template_Last_Storage

// Prior_Template_Last_Invariants keeps that tail at or before the latest earlier definition.
func Prior_Template_Last_Invariants(value Prior_Template_Last, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Holed_Uint16(
    uint16(value.Value),
    uint16(NODE_REFERENCE_MINIMUM),
    uint16(EARLIER_TEMPLATE_REFERENCE_MAXIMUM),
    uint16(ROOT_LIST_HOLE),
    uint16(ROOT_LIST_HOLE),
    uint16(ROOT_LIST_HOLE),
  ).
    Ensure()
}

// CONTROL_BLOCK_TAIL_SIZE is the shortest branch whose body holds the shortest block.
const CONTROL_BLOCK_TAIL_SIZE = CONTROL_ACTION_SIZE_MINIMUM + BLOCK_DEFINITION_SIZE_MINIMUM

// CONTROL_FIRST_TEMPLATE_MAXIMUM is the root-relative offset past the densest actions, one text
// node, that branch's node, pipeline, command, term, and body list, and the block's action.
const CONTROL_FIRST_TEMPLATE_MAXIMUM = (SOURCE_SIZE_MAXIMUM-
  CONTROL_BLOCK_TAIL_SIZE)/
  SIMPLE_ACTION_SIZE_MINIMUM*
  SIMPLE_ACTION_NODE_COUNT +
  ((SOURCE_SIZE_MAXIMUM-CONTROL_BLOCK_TAIL_SIZE)%SIMPLE_ACTION_SIZE_MINIMUM+
    SIMPLE_ACTION_SIZE_MINIMUM-
    SOURCE_POSITION_INCREMENT)/
    SIMPLE_ACTION_SIZE_MINIMUM +
  SIMPLE_ACTION_NODE_COUNT +
  NODE_COUNT_INCREMENT +
  SIMPLE_ACTION_NODE_COUNT +
  NODE_COUNT_INCREMENT

// CONTROL_BLOCK_LIST_MAXIMUM is the body list of the densest-packed branch that holds one
// shortest block.
const CONTROL_BLOCK_LIST_MAXIMUM = ROOT_NODE_COUNT +
  (SOURCE_SIZE_MAXIMUM-CONTROL_BLOCK_TAIL_SIZE)/SIMPLE_ACTION_SIZE_MINIMUM*
    SIMPLE_ACTION_NODE_COUNT +
  ((SOURCE_SIZE_MAXIMUM-CONTROL_BLOCK_TAIL_SIZE)%SIMPLE_ACTION_SIZE_MINIMUM+
    SIMPLE_ACTION_SIZE_MINIMUM-
    SOURCE_POSITION_INCREMENT)/
    SIMPLE_ACTION_SIZE_MINIMUM +
  SIMPLE_ACTION_NODE_COUNT +
  NODE_COUNT_INCREMENT

// CONTROL_INVOCATION_TAIL_SIZE is the shortest branch whose body holds one shortest invocation.
const CONTROL_INVOCATION_TAIL_SIZE = CONTROL_ACTION_SIZE_MINIMUM +
  TEMPLATE_INVOCATION_SIZE_MINIMUM

// CONTROL_INVOCATION_LIST_MAXIMUM is the body list of the densest-packed branch that holds that
// invocation.
const CONTROL_INVOCATION_LIST_MAXIMUM = ROOT_NODE_COUNT +
  (SOURCE_SIZE_MAXIMUM-CONTROL_INVOCATION_TAIL_SIZE)/
    SIMPLE_ACTION_SIZE_MINIMUM*
    SIMPLE_ACTION_NODE_COUNT +
  ((SOURCE_SIZE_MAXIMUM-CONTROL_INVOCATION_TAIL_SIZE)%SIMPLE_ACTION_SIZE_MINIMUM+
    SIMPLE_ACTION_SIZE_MINIMUM-
    SOURCE_POSITION_INCREMENT)/
    SIMPLE_ACTION_SIZE_MINIMUM +
  SIMPLE_ACTION_NODE_COUNT +
  NODE_COUNT_INCREMENT

// KEPT_TEMPLATE_COUNT_MAXIMUM counts every distinct name one shortest named definition leaves
// room for beside the one empty definition that follows.
const KEPT_TEMPLATE_COUNT_MAXIMUM = (SOURCE_SIZE_MAXIMUM - TEMPLATE_DEFINITION_SIZE_MINIMUM) /
  (TEMPLATE_DEFINITION_SIZE_MINIMUM + IDENTIFIER_SIZE_MINIMUM)

// Kept_Template_Count is the size of the table one search rebuilds.
type Kept_Template_Count Template_Count_Stored

// Kept_Template_Count_Invariants keeps that table below the one definition searched for.
func Kept_Template_Count_Invariants(value Kept_Template_Count, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(TEMPLATE_COUNT_MINIMUM),
    uint16(KEPT_TEMPLATE_COUNT_MAXIMUM),
  ).
    Ensure()
}

// Kept_Template_State is the table one search rebuilds before the definition it looks for.
type Kept_Template_State Template_State

// Kept_Template_State_Invariants keeps each entry before that definition.
func Kept_Template_State_Invariants(value Kept_Template_State, namespace aver.Namespace) {
  Prior_Template_First_Invariants(Prior_Template_First(value.First), namespace)
  Prior_Template_Last_Invariants(Prior_Template_Last(value.Last), namespace)
  Kept_Template_Count_Invariants(Kept_Template_Count(value.Count), namespace)
}

// Kept_Template_State_Pointer keeps mutation of that table typed.
type Kept_Template_State_Pointer *Kept_Template_State

// Kept_Template_State_Pointer_Invariants composes one present table.
func Kept_Template_State_Pointer_Invariants(
  value Kept_Template_State_Pointer,
  namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Kept_Template_State_Invariants(*value, namespace)
}

// MERGE_PREVIOUS_TAIL_SIZE is one shortest named block and two shortest empty definitions.
const MERGE_PREVIOUS_TAIL_SIZE = BLOCK_DEFINITION_SIZE_MINIMUM +
  IDENTIFIER_SIZE_MINIMUM +
  TEMPLATE_DEFINITION_SIZE_MINIMUM +
  TEMPLATE_DEFINITION_SIZE_MINIMUM

// MERGE_PREVIOUS_REFERENCE_MAXIMUM is that block's definition behind the densest actions and one
// text node, the latest entry one merge can stand behind.
const MERGE_PREVIOUS_REFERENCE_MAXIMUM = ROOT_NODE_COUNT +
  (SOURCE_SIZE_MAXIMUM-MERGE_PREVIOUS_TAIL_SIZE)/SIMPLE_ACTION_SIZE_MINIMUM*
    SIMPLE_ACTION_NODE_COUNT +
  ((SOURCE_SIZE_MAXIMUM-MERGE_PREVIOUS_TAIL_SIZE)%SIMPLE_ACTION_SIZE_MINIMUM+
    SIMPLE_ACTION_SIZE_MINIMUM-
    SOURCE_POSITION_INCREMENT)/
    SIMPLE_ACTION_SIZE_MINIMUM +
  SIMPLE_ACTION_NODE_COUNT +
  NODE_COUNT_INCREMENT

// NESTED_ELSE_TAIL_SIZE is the shortest branch whose body holds one shortest branch with an
// alternate arm.
const NESTED_ELSE_TAIL_SIZE = CONTROL_ACTION_SIZE_MINIMUM +
  CONTROL_ACTION_SIZE_MINIMUM +
  ACTION_BOUNDARY_SIZE_MINIMUM +
  len(KEYWORD_ELSE)

// NESTED_ELSE_PARENT_LIST_MAXIMUM is the outer branch's body list behind the densest actions and
// one text node.
const NESTED_ELSE_PARENT_LIST_MAXIMUM = ROOT_NODE_COUNT +
  (SOURCE_SIZE_MAXIMUM-NESTED_ELSE_TAIL_SIZE)/SIMPLE_ACTION_SIZE_MINIMUM*
    SIMPLE_ACTION_NODE_COUNT +
  ((SOURCE_SIZE_MAXIMUM-NESTED_ELSE_TAIL_SIZE)%SIMPLE_ACTION_SIZE_MINIMUM+
    SIMPLE_ACTION_SIZE_MINIMUM-
    SOURCE_POSITION_INCREMENT)/
    SIMPLE_ACTION_SIZE_MINIMUM +
  SIMPLE_ACTION_NODE_COUNT +
  NODE_COUNT_INCREMENT

// ELSE_BRANCH_REFERENCE_MAXIMUM is the branch node of that densest branch whose text body an
// alternate list follows.
const ELSE_BRANCH_REFERENCE_MAXIMUM = CONTROL_ELSE_COUNT_MAXIMUM -
  SIMPLE_ACTION_NODE_COUNT -
  NODE_COUNT_INCREMENT

// Free_Frame_Cursor is the frame cursor one push reads, with one slot still free above it.
type Free_Frame_Cursor Bound_Frame_Count

// Free_Frame_Cursor_Invariants keeps that free slot inside frame storage.
func Free_Frame_Cursor_Invariants(value Free_Frame_Cursor, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(FRAME_COUNT_INCREMENT),
    uint16(FRAME_COUNT_MAXIMUM-FRAME_COUNT_INCREMENT),
  ).
    Ensure()
}

// Free_Frame_Cursor_Pointer keeps that cursor mutable for its caller.
type Free_Frame_Cursor_Pointer *Free_Frame_Cursor

// Free_Frame_Cursor_Pointer_Invariants composes one present cursor.
func Free_Frame_Cursor_Pointer_Invariants(
  value Free_Frame_Cursor_Pointer,
  namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Free_Frame_Cursor_Invariants(*value, namespace)
}

// SECOND_DECLARATION_START_MAXIMUM leaves that name, the operator, one term, and the end action
// one range clause still owes.
const SECOND_DECLARATION_START_MAXIMUM = ACTION_CONTENT_BOUNDARY_MAXIMUM -
  TEMPLATE_DEFINITION_END_SIZE_MINIMUM -
  DOT_SOURCE_SIZE -
  ASSIGNMENT_OPERATOR_SIZE -
  VARIABLE_NAME_SIZE_MINIMUM

// SECOND_DECLARATION_END_MAXIMUM closes that shortest second name.
const SECOND_DECLARATION_END_MAXIMUM = SECOND_DECLARATION_START_MAXIMUM +
  VARIABLE_NAME_SIZE_MINIMUM

// BOUND_FIRST_TEMPLATE_TAIL_SIZE is the shortest block whose pipeline declares one bare name.
const BOUND_FIRST_TEMPLATE_TAIL_SIZE = BLOCK_DEFINITION_SIZE_MINIMUM +
  VARIABLE_NAME_SIZE_MINIMUM +
  ASSIGNMENT_OPERATOR_SIZE

// BOUND_FIRST_TEMPLATE_MAXIMUM is the root-relative offset of that block's definition behind the
// densest actions, one text node, and the block's node, pipeline, name, command, and term.
const BOUND_FIRST_TEMPLATE_MAXIMUM = (SOURCE_SIZE_MAXIMUM-
  BOUND_FIRST_TEMPLATE_TAIL_SIZE)/
  SIMPLE_ACTION_SIZE_MINIMUM*
  SIMPLE_ACTION_NODE_COUNT +
  ((SOURCE_SIZE_MAXIMUM-BOUND_FIRST_TEMPLATE_TAIL_SIZE)%SIMPLE_ACTION_SIZE_MINIMUM+
    SIMPLE_ACTION_SIZE_MINIMUM-
    SOURCE_POSITION_INCREMENT)/
    SIMPLE_ACTION_SIZE_MINIMUM +
  SIMPLE_ACTION_NODE_COUNT +
  NODE_COUNT_INCREMENT +
  NODE_COUNT_INCREMENT

// MISPLACED_DECLARATION_TAIL_SIZE is the shortest action whose binding operator follows one term.
const MISPLACED_DECLARATION_TAIL_SIZE = ACTION_BOUNDARY_SIZE_MINIMUM +
  IDENTIFIER_SIZE_MINIMUM +
  ACTION_SEPARATOR_SIZE_MINIMUM +
  VARIABLE_NAME_SIZE_MINIMUM +
  ASSIGNMENT_OPERATOR_SIZE

// MISPLACED_DECLARATION_PACKED_COUNT packs the densest actions and one text node before that
// action.
const MISPLACED_DECLARATION_PACKED_COUNT = ROOT_NODE_COUNT +
  (SOURCE_SIZE_MAXIMUM-MISPLACED_DECLARATION_TAIL_SIZE)/
    SIMPLE_ACTION_SIZE_MINIMUM*
    SIMPLE_ACTION_NODE_COUNT +
  ((SOURCE_SIZE_MAXIMUM-MISPLACED_DECLARATION_TAIL_SIZE)%SIMPLE_ACTION_SIZE_MINIMUM+
    SIMPLE_ACTION_SIZE_MINIMUM-
    SOURCE_POSITION_INCREMENT)/
    SIMPLE_ACTION_SIZE_MINIMUM

// MISPLACED_DECLARATION_COMMAND_MAXIMUM is that action's command, past its action, pipeline,
// and the term before the declaration.
const MISPLACED_DECLARATION_COMMAND_MAXIMUM = MISPLACED_DECLARATION_PACKED_COUNT +
  SIMPLE_ACTION_NODE_COUNT

// CONTROL_COMMENT_TAIL_SIZE is the shortest branch whose body holds one shortest comment.
const CONTROL_COMMENT_TAIL_SIZE = CONTROL_ACTION_SIZE_MINIMUM + COMMENT_ACTION_SIZE_MINIMUM

// CONTROL_COMMENT_LIST_MAXIMUM is that branch's body list behind the densest actions and one
// text node.
const CONTROL_COMMENT_LIST_MAXIMUM = ROOT_NODE_COUNT +
  (SOURCE_SIZE_MAXIMUM-CONTROL_COMMENT_TAIL_SIZE)/
    SIMPLE_ACTION_SIZE_MINIMUM*
    SIMPLE_ACTION_NODE_COUNT +
  ((SOURCE_SIZE_MAXIMUM-CONTROL_COMMENT_TAIL_SIZE)%SIMPLE_ACTION_SIZE_MINIMUM+
    SIMPLE_ACTION_SIZE_MINIMUM-
    SOURCE_POSITION_INCREMENT)/
    SIMPLE_ACTION_SIZE_MINIMUM +
  SIMPLE_ACTION_NODE_COUNT +
  NODE_COUNT_INCREMENT

// RANGE_DECLARATION_TAIL_SIZE is the shortest range, with its end, that declares one bare name.
const RANGE_DECLARATION_TAIL_SIZE = CONTROL_ACTION_SIZE_MINIMUM +
  len(KEYWORD_RANGE) -
  len(KEYWORD_IF) +
  VARIABLE_NAME_SIZE_MINIMUM +
  ASSIGNMENT_OPERATOR_SIZE

// RANGE_FIRST_DECLARATION_MAXIMUM is that range's name behind the densest actions, one text node,
// the range node, and its pipeline.
const RANGE_FIRST_DECLARATION_MAXIMUM = ROOT_NODE_COUNT +
  (SOURCE_SIZE_MAXIMUM-RANGE_DECLARATION_TAIL_SIZE)/
    SIMPLE_ACTION_SIZE_MINIMUM*
    SIMPLE_ACTION_NODE_COUNT +
  ((SOURCE_SIZE_MAXIMUM-RANGE_DECLARATION_TAIL_SIZE)%SIMPLE_ACTION_SIZE_MINIMUM+
    SIMPLE_ACTION_SIZE_MINIMUM-
    SOURCE_POSITION_INCREMENT)/
    SIMPLE_ACTION_SIZE_MINIMUM +
  ACTION_PIPELINE_NODE_COUNT +
  NODE_COUNT_INCREMENT

// Ranged_First_Declaration_Storage is the first name one range clause declares.
type Ranged_First_Declaration_Storage Declaration_First_Reference_Storage

// Ranged_First_Declaration_Storage_Invariants keeps that name inside the range clause.
func Ranged_First_Declaration_Storage_Invariants(
  value Ranged_First_Declaration_Storage,
  namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Holed_Uint16(
    uint16(value.Value),
    uint16(NODE_REFERENCE_MINIMUM),
    uint16(RANGE_FIRST_DECLARATION_MAXIMUM),
    uint16(ROOT_LIST_HOLE),
    uint16(TERM_ACTION_HOLE),
    uint16(TERM_PIPELINE_HOLE),
  ).
    Ensure()
}

// Ranged_Declaration_References are the names one range clause declares.
type Ranged_Declaration_References Declaration_References

// Ranged_Declaration_References_Invariants composes both names.
func Ranged_Declaration_References_Invariants(
  value Ranged_Declaration_References,
  namespace aver.Namespace,
) {
  Ranged_First_Declaration_Storage_Invariants(
    Ranged_First_Declaration_Storage(value.First),
    namespace,
  )
  Declaration_Second_Reference_Storage_Invariants(&value.Second, namespace)
}

// Ranged_Pipeline_State is the pipeline state one range clause hands its frame.
type Ranged_Pipeline_State Pipeline_State

// Ranged_Pipeline_State_Invariants composes that state's value and range names.
func Ranged_Pipeline_State_Invariants(value Ranged_Pipeline_State, namespace aver.Namespace) {
  Value_Invariants(&value.Value, namespace)
  Ranged_Declaration_References_Invariants(
    Ranged_Declaration_References(value.Declarations),
    namespace,
  )
  aver.Tree(value, namespace).
    Enum_3_Uint8(
    uint8(value.Declaration_Count),
    uint8(DECLARATION_COUNT_NONE),
    uint8(DECLARATION_COUNT_ONE),
    uint8(DECLARATION_COUNT_TWO),
  ).
    Sometimes(
    bool(value.Assignment),
    "Pipeline updates existing variable.",
  ).
    Ensure()
}

// INVOCATION_SIBLING_TAIL_SIZE is the shortest invocation and the one text byte past it.
const INVOCATION_SIBLING_TAIL_SIZE = TEMPLATE_INVOCATION_SIZE_MINIMUM +
  utf8.CHARACTER_SIZE_MINIMUM

// INVOCATION_SIBLING_REFERENCE_MAXIMUM is that text behind the densest actions, one text node,
// and the invocation.
const INVOCATION_SIBLING_REFERENCE_MAXIMUM = ROOT_NODE_COUNT +
  (SOURCE_SIZE_MAXIMUM-INVOCATION_SIBLING_TAIL_SIZE)/
    SIMPLE_ACTION_SIZE_MINIMUM*
    SIMPLE_ACTION_NODE_COUNT +
  ((SOURCE_SIZE_MAXIMUM-INVOCATION_SIBLING_TAIL_SIZE)%SIMPLE_ACTION_SIZE_MINIMUM+
    SIMPLE_ACTION_SIZE_MINIMUM-
    SOURCE_POSITION_INCREMENT)/
    SIMPLE_ACTION_SIZE_MINIMUM +
  NODE_COUNT_INCREMENT +
  NODE_COUNT_INCREMENT

// CHAIN_PARENT_DEPTH_MAXIMUM leaves the chain's pipeline inside the deepest nesting.
const CHAIN_PARENT_DEPTH_MAXIMUM = PARENTHESIS_DEPTH_MAXIMUM - PARENTHESIS_DEPTH_INCREMENT

// Chain_Parent_Depth is the evaluation depth one chain pushes onto, below its own pipeline.
type Chain_Parent_Depth Active_Frame_Count

// Chain_Parent_Depth_Invariants leaves the chain's pipeline inside the deepest nesting.
func Chain_Parent_Depth_Invariants(value Chain_Parent_Depth, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value),
    uint16(ACTIVE_FRAME_COUNT_MINIMUM),
    uint16(CHAIN_PARENT_DEPTH_MAXIMUM),
  ).
    Ensure()
}

// Chain_Depth is the evaluation depth one chain leaves after pushing its pipeline.
type Chain_Depth Active_Frame_Count

// Chain_Depth_Invariants keeps that depth inside the deepest nesting.
func Chain_Depth_Invariants(value Chain_Depth, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value),
    uint16(ACTIVE_FRAME_COUNT_MINIMUM),
    uint16(PARENTHESIS_DEPTH_MAXIMUM),
  ).
    Ensure()
}

// BINDING_SOURCE_SIZE_MINIMUM is the shortest source that holds one variable and its operator.
const BINDING_SOURCE_SIZE_MINIMUM = ACTION_BOUNDARY_SIZE_MINIMUM +
  VARIABLE_NAME_SIZE_MINIMUM +
  ASSIGNMENT_OPERATOR_SIZE

// BINDING_PACKED_COUNT packs the densest actions and one text node before the shortest binding
// operator, whose value a hostile source may omit.
const BINDING_PACKED_COUNT = ROOT_NODE_COUNT +
  (SOURCE_SIZE_MAXIMUM-BINDING_SOURCE_SIZE_MINIMUM)/
    SIMPLE_ACTION_SIZE_MINIMUM*
    SIMPLE_ACTION_NODE_COUNT +
  ((SOURCE_SIZE_MAXIMUM-BINDING_SOURCE_SIZE_MINIMUM)%SIMPLE_ACTION_SIZE_MINIMUM+
    SIMPLE_ACTION_SIZE_MINIMUM-
    SOURCE_POSITION_INCREMENT)/
    SIMPLE_ACTION_SIZE_MINIMUM

// BINDING_REFERENCE_MAXIMUM is that binding's name, past its action and pipeline.
const BINDING_REFERENCE_MAXIMUM = BINDING_PACKED_COUNT +
  ACTION_PIPELINE_NODE_COUNT +
  NODE_COUNT_INCREMENT

// Added_Variable_Reference is the name one parsed binding adds to lexical scope.
type Added_Variable_Reference Declaration_Reference_Validated

// Added_Variable_Reference_Invariants leaves the binding's action and pipeline beneath that name.
func Added_Variable_Reference_Invariants(
  value Added_Variable_Reference, namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(COMMAND_NODE_COUNT_MINIMUM),
    uint16(BINDING_REFERENCE_MAXIMUM),
  ).
    Ensure()
}

// Binding_Source is one source that holds one variable binding operator.
type Binding_Source Source

// Binding_Source_Invariants composes the shortest source one binding operator needs.
func Binding_Source_Invariants(value Binding_Source, namespace aver.Namespace) {
  Binding_Source_Data_Stored_Invariants(Binding_Source_Data_Stored(value.Data), namespace)
}

// Binding_Source_Data_Stored separates its source shape from the borrowed header.
type Binding_Source_Data_Stored Source_Storage

// Binding_Source_Data_Stored_Invariants bounds data by one variable and its operator.
func Binding_Source_Data_Stored_Invariants(
  value Binding_Source_Data_Stored,
  namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Int(
    len(value.Value),
    BINDING_SOURCE_SIZE_MINIMUM,
    SOURCE_SIZE_MAXIMUM,
  ).
    Ensure()
}

// Opened_Definition_End is where one named definition keyword action ends.
type Opened_Definition_End Action_End

// Opened_Definition_End_Invariants leaves that shortest named action beneath the end.
func Opened_Definition_End_Invariants(value Opened_Definition_End, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value),
    uint16(INVOCATION_SOURCE_SIZE_MINIMUM),
    uint16(SOURCE_SIZE_MAXIMUM),
  ).
    Ensure()
}

// Opener_Depth is the nesting one definition opens beneath, which its header leaves below the
// deepest frame.
type Opener_Depth Control_Depth_Tracked

// Opener_Depth_Invariants keeps one frame free for that definition.
func Opener_Depth_Invariants(value Opener_Depth, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Value),
    uint16(CONTROL_DEPTH_MINIMUM),
    uint16(CONTROL_DEPTH_MAXIMUM-CONTROL_DEPTH_INCREMENT),
  ).
    Ensure()
}

// Every frame a clause leaves open owes one end action, so a clause whose source cannot fit
// them is malformed before it allocates.
func opener_room(
  source Invocation_Source,
  end_value Opened_Definition_End,
  depth_value Opener_Depth,
) (room Opener_Room) {
  defer func() { Opener_Room_Invariants(room, "opener_room.room") }()
  Invocation_Source_Invariants(source, "opener_room.source")
  Opened_Definition_End_Invariants(end_value, "opener_room.end_value")
  Opener_Depth_Invariants(depth_value, "opener_room.depth_value")
  owed := int(Control_Depth_Tracked_Value(depth_value.Value)) + CONTROL_DEPTH_INCREMENT
  room = Opener_Room(
    !(len(source.Data.Value)-int(end_value) <
      owed*TEMPLATE_DEFINITION_END_SIZE_MINIMUM),
  )
  return room
}

// CLOSED_BRANCH_BODY_MAXIMUM is the body list of the densest closed branch.
const CLOSED_BRANCH_BODY_MAXIMUM = ROOT_NODE_COUNT +
  (SOURCE_SIZE_MAXIMUM-CONTROL_ACTION_SIZE_MINIMUM)/
    SIMPLE_ACTION_SIZE_MINIMUM*
    SIMPLE_ACTION_NODE_COUNT +
  ((SOURCE_SIZE_MAXIMUM-CONTROL_ACTION_SIZE_MINIMUM)%SIMPLE_ACTION_SIZE_MINIMUM+
    SIMPLE_ACTION_SIZE_MINIMUM-
    SOURCE_POSITION_INCREMENT)/
    SIMPLE_ACTION_SIZE_MINIMUM +
  CONTROL_BRANCH_NODE_COUNT

// Action_Pipe_Node is one pipeline an action or branch owns, which no argument is.
type Action_Pipe_Node Pipe_Node

// Action_Pipe_Node_Invariants leaves a sibling only for the body list one branch owns.
func Action_Pipe_Node_Invariants(value Action_Pipe_Node, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint16(
    uint16(value.Start),
    uint16(ACTION_CONTENT_BOUNDARY_MINIMUM),
    uint16(LEXEME_START_MAXIMUM),
  ).
    Range_Uint16(
    uint16(value.End),
    uint16(CONSTANT_END_MINIMUM),
    uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM),
  ).
    Range_Uint16(
    uint16(value.Value_Start),
    uint16(ACTION_CONTENT_BOUNDARY_MINIMUM),
    uint16(LEXEME_START_MAXIMUM),
  ).
    Range_Uint16(
    uint16(value.Value_End),
    uint16(CONSTANT_END_MINIMUM),
    uint16(ACTION_CONTENT_BOUNDARY_MAXIMUM),
  ).
    Range_Holed_Uint16(
    uint16(value.First_Child),
    uint16(NODE_REFERENCE_MINIMUM),
    uint16(NODE_REFERENCE_MAXIMUM),
    uint16(ROOT_LIST_HOLE),
    uint16(TERM_ACTION_HOLE),
    uint16(TERM_ACTION_HOLE),
  ).
    Range_Holed_Uint16(
    uint16(value.Next_Sibling),
    uint16(NODE_REFERENCE_MINIMUM),
    uint16(CLOSED_BRANCH_BODY_MAXIMUM),
    uint16(ROOT_LIST_HOLE),
    uint16(TERM_ACTION_HOLE),
    uint16(TERM_ACTION_HOLE),
  ).
    Ensure()
  aver.Always(
    uint8(value.Kind) == uint8(NODE_PIPE),
    "An action pipeline record keeps the kind its caller selected.",
  )
}

// Action_Pipe_Node_Stored keeps one such record opaque between parser and evaluator.
type Action_Pipe_Node_Stored *Action_Pipe_Node

// Action_Pipe_Node_Stored_Invariants checks the present record.
func Action_Pipe_Node_Stored_Invariants(value Action_Pipe_Node_Stored, namespace aver.Namespace) {
  if value == nil {
    return
  }
  Action_Pipe_Node_Invariants(*value, namespace)
}

// Action_Pipe_Node_Validated carries one deeply checked action pipeline.
type Action_Pipe_Node_Validated struct {
  // Value keeps deep proof attached to the record.
  Value Action_Pipe_Node_Stored
}

// Action_Pipe_Node_Validated_Invariants checks transport ownership after deep validation.
func Action_Pipe_Node_Validated_Invariants(
  value *Action_Pipe_Node_Validated,
  namespace aver.Namespace,
) {
  Action_Pipe_Node_Stored_Invariants(value.Value, namespace)
}

// TEXT_BODY_TAIL_SIZE is one unclosed opener whose body holds one text byte.
const TEXT_BODY_TAIL_SIZE = CONTROL_OPENER_SIZE_MINIMUM + utf8.CHARACTER_SIZE_MINIMUM

// TEXT_BODY_LIST_MAXIMUM is that opener's body list behind the densest actions, one text node,
// and the branch's node, pipeline, command, and term.
const TEXT_BODY_LIST_MAXIMUM = ROOT_NODE_COUNT +
  (SOURCE_SIZE_MAXIMUM-TEXT_BODY_TAIL_SIZE)/SIMPLE_ACTION_SIZE_MINIMUM*
    SIMPLE_ACTION_NODE_COUNT +
  ((SOURCE_SIZE_MAXIMUM-TEXT_BODY_TAIL_SIZE)%SIMPLE_ACTION_SIZE_MINIMUM+
    SIMPLE_ACTION_SIZE_MINIMUM-
    SOURCE_POSITION_INCREMENT)/
    SIMPLE_ACTION_SIZE_MINIMUM +
  CONTROL_BRANCH_NODE_COUNT

// ACTION_BODY_TAIL_SIZE is one unclosed opener whose body holds one shortest action.
const ACTION_BODY_TAIL_SIZE = CONTROL_OPENER_SIZE_MINIMUM + SIMPLE_ACTION_SIZE_MINIMUM

// ACTION_BODY_LIST_MAXIMUM is that opener's body list behind the densest actions, one text node,
// and the branch's node, pipeline, command, and term.
const ACTION_BODY_LIST_MAXIMUM = ROOT_NODE_COUNT +
  (SOURCE_SIZE_MAXIMUM-ACTION_BODY_TAIL_SIZE)/SIMPLE_ACTION_SIZE_MINIMUM*
    SIMPLE_ACTION_NODE_COUNT +
  ((SOURCE_SIZE_MAXIMUM-ACTION_BODY_TAIL_SIZE)%SIMPLE_ACTION_SIZE_MINIMUM+
    SIMPLE_ACTION_SIZE_MINIMUM-
    SOURCE_POSITION_INCREMENT)/
    SIMPLE_ACTION_SIZE_MINIMUM +
  CONTROL_BRANCH_NODE_COUNT

// Action_List_Value is the list one parsed action joins.
type Action_List_Value List_State_Value_Stored

// Action_List_Value_Invariants leaves that action's own bytes beneath the list.
func Action_List_Value_Invariants(value Action_List_Value, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Holed_Uint16(
    uint16(value.List),
    uint16(ROOT_NODE_COUNT),
    uint16(ACTION_BODY_LIST_MAXIMUM),
    uint16(ACTION_LIST_HOLE),
    uint16(ACTION_LIST_HOLE),
    uint16(ACTION_LIST_HOLE),
  ).
    Range_Holed_Uint16(
    uint16(value.Last_Child),
    uint16(NODE_REFERENCE_MINIMUM),
    uint16(LIST_LAST_CHILD_MAXIMUM),
    uint16(ROOT_LIST_HOLE),
    uint16(ROOT_LIST_HOLE),
    uint16(ROOT_LIST_HOLE),
  ).
    Ensure()
}

// Action_List_State is that list storage.
type Action_List_State List_State_Storage

// Action_List_State_Invariants composes that list.
func Action_List_State_Invariants(value Action_List_State, namespace aver.Namespace) {
  Action_List_Value_Invariants(Action_List_Value(value.Value), namespace)
}

// Action_List_State_Pointer keeps mutation of that list typed.
type Action_List_State_Pointer *Action_List_State

// Action_List_State_Pointer_Invariants composes one present list.
func Action_List_State_Pointer_Invariants(
  value Action_List_State_Pointer,
  namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Action_List_State_Invariants(*value, namespace)
}
