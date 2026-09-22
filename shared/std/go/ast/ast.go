// Package ast parses Go source into a node arena. The caller owns one Parse_State and every
// node is a slot in caller storage, thus a parse allocates nothing. The grammar it accepts is the
// subset this repository's linter permits. The grammar is a program of small steps rather than a
// function for each rule, because a rule function runs only where its syntax stands and would owe
// every read position and every arena count at a call site that never meets most of them.
package ast

import "local/james-orcales/shared/std/go/token"
import "local/james-orcales/shared/std/sim/aver/default"

// TOKEN_COUNT_MAXIMUM caps the token run. The densest first-party file in this repository
// spends 50,875 tokens, so this is two and a half times the largest run measured, and a source
// past it fails the parse rather than writing outside the array.
const TOKEN_COUNT_MAXIMUM = 131072

// NODE_COUNT_MAXIMUM caps the arena. It stands below the token bound because a file holds its
// package clause and its end before any node, thus the arena can never outgrow the run.
const NODE_COUNT_MAXIMUM = TOKEN_COUNT_MAXIMUM - 8

// DEPTH_MAXIMUM caps how deep the tree may nest. The parent stack holds one slot for each open
// node, and a hostile nesting depth ends the parse rather than writing past that stack.
const DEPTH_MAXIMUM = 256

// FRAME_MAXIMUM caps how deep the grammar program may call itself. A grouped type opens no node,
// thus the tree depth alone never bounds the recursion, and a hostile run of parentheses ends the
// parse here instead of on the goroutine stack. Four frames stand for each open node, which is
// more than the deepest chain of rules one node ever needs.
const FRAME_MAXIMUM = DEPTH_MAXIMUM * 4

// INDEX_ABSENT names no node. Slot zero holds no node, thus an absent child needs no separate
// flag and a zero Node_Kind reads as absent.
const INDEX_ABSENT Index = 0

// INDEX_FIRST is the first slot a node can occupy.
const INDEX_FIRST Index = 1

// INDEX_MINIMUM is INDEX_ABSENT, the smallest slot number.
const INDEX_MINIMUM = int32(INDEX_ABSENT)

// INDEX_MAXIMUM is the final slot of the arena.
const INDEX_MAXIMUM = NODE_COUNT_MAXIMUM - 1

// ANCESTOR_MAXIMUM is one slot below the arena end, because a parent takes its slot before its
// children and the final slot therefore holds no parent.
const ANCESTOR_MAXIMUM = NODE_COUNT_MAXIMUM - 2

// TOKEN_INDEX_MINIMUM is the first token of the run.
const TOKEN_INDEX_MINIMUM = 0

// TOKEN_INDEX_MAXIMUM is the final token of the run.
const TOKEN_INDEX_MAXIMUM = TOKEN_COUNT_MAXIMUM - 1

// TOKEN_RUN_SIZE_MINIMUM is the end token every completed scan writes.
const TOKEN_RUN_SIZE_MINIMUM = 1

// NODE_COUNT_BARE is the count a file node alone would give. No parse returns it: the package
// clause opens with one token the tree holds, either the name the clause states or the error node
// that refuses it.
const NODE_COUNT_BARE = 2

// PRECEDENCE_NONE marks a token that binds no binary expression.
const PRECEDENCE_NONE Precedence = 0

// PRECEDENCE_COMPARISON is the level of the four comparison operators this dialect states.
const PRECEDENCE_COMPARISON Precedence = 1

// PRECEDENCE_ADDITION is the level of addition, subtraction, and the two or operators.
const PRECEDENCE_ADDITION Precedence = 2

// PRECEDENCE_MULTIPLICATION is the level of the products, the shifts, and the two and operators.
const PRECEDENCE_MULTIPLICATION Precedence = 3

// EMBEDDED_WALK_MAXIMUM caps the walk from a field to the identifier it embeds. A pointer, a
// qualifier, and a type argument list each add one step, and nothing Go admits adds many.
const EMBEDDED_WALK_MAXIMUM = 8

// BLANK_NAME is the name a caller binds a package to when it wants none.
const BLANK_NAME = "_"

// TRUE_NAME is the identifier a loop reads as a condition that constrains nothing.
const TRUE_NAME = "true"

// IOTA_NAME is the identifier that ties a constant value to declaration order.
const IOTA_NAME = "iota"

// WIDE_BYTE_MINIMUM is the first byte that opens no ASCII name.
const WIDE_BYTE_MINIMUM = 128

// FAILURE_NONE marks a parse that met no failure.
const FAILURE_NONE Failure_Code = 0

// FAILURE_SYNTAX marks a token that fits no rule of the grammar.
const FAILURE_SYNTAX Failure_Code = 1

// FAILURE_INTERFACE_METHOD marks an interface that holds a method. The linter bans the
// interface, not the method: a method declaration parses and the linter judges it on its own.
const FAILURE_INTERFACE_METHOD Failure_Code = 2

// FAILURE_WIDE_IDENTIFIER marks a name holding a byte at or above 128.
const FAILURE_WIDE_IDENTIFIER Failure_Code = 3

// FAILURE_IOTA marks the iota identifier, which ties a constant value to declaration order.
const FAILURE_IOTA Failure_Code = 4

// FAILURE_REFUSED_SIGN marks a sign this dialect states no form for: the conditional and and the
// conditional or, which are nested ifs written flat, and the two order signs that hold equality,
// which a strict comparison states.
const FAILURE_REFUSED_SIGN Failure_Code = 5

// FAILURE_DOT_IMPORT marks an import that binds no name, which spills a package into the file.
const FAILURE_DOT_IMPORT Failure_Code = 6

// FAILURE_BLANK_IMPORT marks an import bound to the blank name, which is a package taken for its
// effects alone.
const FAILURE_BLANK_IMPORT Failure_Code = 7

// FAILURE_CONSTANT_CASE marks a constant name that is no run of uppercase words.
const FAILURE_CONSTANT_CASE Failure_Code = 8

// FAILURE_BARE_LOOP marks a for that no clause constrains, which runs forever.
const FAILURE_BARE_LOOP Failure_Code = 9

// FAILURE_UNNAMED_RESULT marks a signature result that carries no name.
const FAILURE_UNNAMED_RESULT Failure_Code = 10

// FAILURE_TYPE_ALIAS marks a type declaration bound by the assignment sign, which gives one
// type a second name instead of declaring a type of its own.
const FAILURE_TYPE_ALIAS Failure_Code = 11

// FAILURE_CLOSURE_BODY marks a function literal whose body holds anything but calls, which
// hides logic a named function would lay open to a test.
const FAILURE_CLOSURE_BODY Failure_Code = 12

// FAILURE_FALLTHROUGH marks a case that runs on into the case behind it, which states a flow no
// reader of the case alone can see.
const FAILURE_FALLTHROUGH Failure_Code = 13

// FAILURE_PACKAGE_CLAUSE marks a file that opens with anything but the package keyword.
const FAILURE_PACKAGE_CLAUSE Failure_Code = 14

// FAILURE_TOKEN_COUNT marks a source holding more tokens than one run admits.
const FAILURE_TOKEN_COUNT Failure_Code = 15

// FAILURE_NODE_COUNT marks a source holding more syntax than the arena admits.
const FAILURE_NODE_COUNT Failure_Code = 16

// FAILURE_NESTING_DEPTH marks a source nesting deeper than one parse admits.
const FAILURE_NESTING_DEPTH Failure_Code = 17

// FAILURE_CODE_MINIMUM is the clean parse, the smallest code.
const FAILURE_CODE_MINIMUM = uint8(FAILURE_NONE)

// FAILURE_CODE_MAXIMUM is the deepest nesting, the largest code.
const FAILURE_CODE_MAXIMUM = uint8(FAILURE_NESTING_DEPTH)

// MESSAGE_SIZE_MINIMUM is the byte count of the shortest failure message.
const MESSAGE_SIZE_MINIMUM = 14

// MESSAGE_SIZE_MAXIMUM is the byte count of the longest failure message.
const MESSAGE_SIZE_MAXIMUM = 65

// NODE_FILE is one parsed source file.
const NODE_FILE Node_Kind = 1

// NODE_COMMENT is one line comment or general comment.
const NODE_COMMENT Node_Kind = 2

// NODE_IMPORT is one import declaration.
const NODE_IMPORT Node_Kind = 3

// NODE_CONSTANT is one const declaration.
const NODE_CONSTANT Node_Kind = 4

// NODE_VARIABLE is one var declaration.
const NODE_VARIABLE Node_Kind = 5

// NODE_TYPE is one defined type declaration.
const NODE_TYPE Node_Kind = 6

// NODE_FUNCTION is one function or method declaration.
const NODE_FUNCTION Node_Kind = 7

// NODE_RECEIVER is the receiver of a method declaration.
const NODE_RECEIVER Node_Kind = 8

// NODE_TYPE_PARAMETER is one type parameter and its constraint.
const NODE_TYPE_PARAMETER Node_Kind = 9

// NODE_PARAMETER is one parameter or one result of a signature.
const NODE_PARAMETER Node_Kind = 10

// NODE_FIELD is one struct field, its type, and its optional tag.
const NODE_FIELD Node_Kind = 11

// NODE_POINTER_TYPE is a pointer type.
const NODE_POINTER_TYPE Node_Kind = 12

// NODE_ARRAY_TYPE is an array type and its capacity.
const NODE_ARRAY_TYPE Node_Kind = 13

// NODE_SLICE_TYPE is a slice type.
const NODE_SLICE_TYPE Node_Kind = 14

// NODE_MAP_TYPE is a map type, its key, and its value.
const NODE_MAP_TYPE Node_Kind = 15

// NODE_CHANNEL_TYPE is a two-way channel type.
const NODE_CHANNEL_TYPE Node_Kind = 16

// NODE_CHANNEL_SEND is a send-only channel type.
const NODE_CHANNEL_SEND Node_Kind = 17

// NODE_CHANNEL_RECEIVE is a receive-only channel type.
const NODE_CHANNEL_RECEIVE Node_Kind = 18

// NODE_FUNCTION_TYPE is a function type and its signature.
const NODE_FUNCTION_TYPE Node_Kind = 19

// NODE_STRUCTURE_TYPE is a struct type and its fields.
const NODE_STRUCTURE_TYPE Node_Kind = 20

// NODE_INTERFACE_TYPE is an interface type and its elements.
const NODE_INTERFACE_TYPE Node_Kind = 21

// NODE_ELLIPSIS is the ellipsis of a variadic parameter or an inferred array capacity.
const NODE_ELLIPSIS Node_Kind = 22

// NODE_GENERIC is one instantiated generic type and its arguments.
const NODE_GENERIC Node_Kind = 23

// NODE_TERM is one approximation term of a type constraint.
const NODE_TERM Node_Kind = 24

// NODE_UNION is a union of type constraint terms.
const NODE_UNION Node_Kind = 25

// NODE_BLOCK is one brace-delimited statement list.
const NODE_BLOCK Node_Kind = 26

// NODE_EXPRESSION_STATEMENT is one expression standing as a statement.
const NODE_EXPRESSION_STATEMENT Node_Kind = 27

// NODE_ASSIGN is one assignment statement.
const NODE_ASSIGN Node_Kind = 28

// NODE_DEFINE is one short variable declaration.
const NODE_DEFINE Node_Kind = 29

// NODE_OPERATION_ASSIGN is one compound assignment such as an addition assignment.
const NODE_OPERATION_ASSIGN Node_Kind = 30

// NODE_INCREMENT is one increment statement.
const NODE_INCREMENT Node_Kind = 31

// NODE_DECREMENT is one decrement statement.
const NODE_DECREMENT Node_Kind = 32

// NODE_SEND is one channel send statement.
const NODE_SEND Node_Kind = 33

// NODE_RETURN is one return statement and its results.
const NODE_RETURN Node_Kind = 34

// NODE_IF is one if statement, its optional initializer, its condition, and its branches.
const NODE_IF Node_Kind = 35

// NODE_FOR is one for statement and its clauses.
const NODE_FOR Node_Kind = 36

// NODE_RANGE is one range clause of a for statement.
const NODE_RANGE Node_Kind = 37

// NODE_SWITCH is one expression switch statement.
const NODE_SWITCH Node_Kind = 38

// NODE_TYPE_SWITCH is one type switch statement.
const NODE_TYPE_SWITCH Node_Kind = 39

// NODE_CASE is one case clause of a switch or a select statement.
const NODE_CASE Node_Kind = 40

// NODE_DEFAULT is one default clause of a switch or a select statement.
const NODE_DEFAULT Node_Kind = 41

// NODE_SELECT is one select statement.
const NODE_SELECT Node_Kind = 42

// NODE_GO is one go statement.
const NODE_GO Node_Kind = 43

// NODE_DEFER is one defer statement.
const NODE_DEFER Node_Kind = 44

// NODE_BREAK is one break statement and its optional label.
const NODE_BREAK Node_Kind = 45

// NODE_CONTINUE is one continue statement and its optional label.
const NODE_CONTINUE Node_Kind = 46

// NODE_GOTO is one goto statement and its label.
const NODE_GOTO Node_Kind = 47

// NODE_FALLTHROUGH is one fallthrough statement.
const NODE_FALLTHROUGH Node_Kind = 48

// NODE_LABEL is one labelled statement.
const NODE_LABEL Node_Kind = 49

// NODE_DECLARATION_STATEMENT is one const, var, or type declaration inside a body.
const NODE_DECLARATION_STATEMENT Node_Kind = 50

// NODE_IDENTIFIER is one name.
const NODE_IDENTIFIER Node_Kind = 51

// NODE_INTEGER is one integer literal.
const NODE_INTEGER Node_Kind = 52

// NODE_FLOAT is one floating-point literal.
const NODE_FLOAT Node_Kind = 53

// NODE_IMAGINARY is one imaginary literal.
const NODE_IMAGINARY Node_Kind = 54

// NODE_CHARACTER is one character literal.
const NODE_CHARACTER Node_Kind = 55

// NODE_STRING is one string literal.
const NODE_STRING Node_Kind = 56

// NODE_BINARY is one binary expression and its two operands.
const NODE_BINARY Node_Kind = 57

// NODE_UNARY is one unary expression and its operand.
const NODE_UNARY Node_Kind = 58

// NODE_CALL is one call and its arguments.
const NODE_CALL Node_Kind = 59

// NODE_INDEX is one index expression or one generic instantiation.
const NODE_INDEX Node_Kind = 60

// NODE_SLICE_EXPRESSION is one slice expression and its bounds.
const NODE_SLICE_EXPRESSION Node_Kind = 61

// NODE_SELECTOR is one selector expression.
const NODE_SELECTOR Node_Kind = 62

// NODE_ASSERTION is one type assertion.
const NODE_ASSERTION Node_Kind = 63

// NODE_COMPOSITE is one composite literal and its elements.
const NODE_COMPOSITE Node_Kind = 64

// NODE_KEY_VALUE is one keyed element of a composite literal.
const NODE_KEY_VALUE Node_Kind = 65

// NODE_PARENTHESIS is one parenthesised expression.
const NODE_PARENTHESIS Node_Kind = 66

// NODE_FUNCTION_LITERAL is one function literal and its body.
const NODE_FUNCTION_LITERAL Node_Kind = 67

// NODE_ERROR marks the token at which the grammar rejected the source.
const NODE_ERROR Node_Kind = 68

// NODE_BLANK marks a run of empty lines. Its token is the token that follows the run, and that
// token counts the lines, thus a caller writes the spacing back out from the tree alone.
const NODE_BLANK Node_Kind = 69

// NODE_FIELD_NAME is one name of a struct field. It stands apart from an identifier so a caller
// tells a field name from the type behind it, which no bare identifier run states.
const NODE_FIELD_NAME Node_Kind = 70

// NODE_IMPORT_NAME is the name an import binds its package to.
const NODE_IMPORT_NAME Node_Kind = 71

// NODE_CONSTANT_NAME is one name a const declaration binds. It stands apart from an identifier
// because a constant answers to a case rule of its own, whatever its scope.
const NODE_CONSTANT_NAME Node_Kind = 72

// NODE_RESULT is one result of a signature. It stands apart from a parameter so a caller reads
// whether a function sends anything back, which a bare parameter run never states.
const NODE_RESULT Node_Kind = 73

// NODE_PARAMETER_NAME is one name a signature binds. A run of bare types wears the same shape
// until the token behind it settles the question, thus a name says so only once it is settled.
const NODE_PARAMETER_NAME Node_Kind = 74

// CLAUSE_KIND_MINIMUM is the simple statement that carries no operator.
const CLAUSE_KIND_MINIMUM = int(NODE_EXPRESSION_STATEMENT)

// CLAUSE_KIND_MAXIMUM is the largest simple statement kind.
const CLAUSE_KIND_MAXIMUM = int(NODE_SEND)

// SUFFIX_KIND_INDEX names the bracket suffix that reads one element.
const SUFFIX_KIND_INDEX = int(NODE_INDEX)

// SUFFIX_KIND_SLICE names the bracket suffix that holds a colon.
const SUFFIX_KIND_SLICE = int(NODE_SLICE_EXPRESSION)

// SUFFIX_KIND_GENERIC names the bracket suffix that holds a comma.
const SUFFIX_KIND_GENERIC = int(NODE_GENERIC)

// LITERAL_KIND_MINIMUM is the smallest literal kind.
const LITERAL_KIND_MINIMUM = int(NODE_INTEGER)

// LITERAL_KIND_MAXIMUM is the largest literal kind.
const LITERAL_KIND_MAXIMUM = int(NODE_STRING)

// LINK_HOLE_ROOT is the file slot. It is the root, thus no node holds it as a link.
const LINK_HOLE_ROOT = 1

// LINK_HOLE_FIRST is the slot after the file. It opens the child chain of the file, thus no node
// holds it as a following sibling.
const LINK_HOLE_FIRST = 2

// NODE_KIND_MINIMUM is the file kind, the smallest kind. No kind stands for an absent node,
// because INDEX_ABSENT already names one and slot zero carries NODE_ERROR so a stray read of it
// is loud rather than silent.
const NODE_KIND_MINIMUM = uint8(NODE_FILE)

// NODE_KIND_MAXIMUM is the parameter name kind, the largest kind.
const NODE_KIND_MAXIMUM = uint8(NODE_PARAMETER_NAME)

// Failure_Code names why a parse stopped, or FAILURE_NONE when it did not.
type Failure_Code uint8

// Failure_Code_Invariants states the clean parse and every cause.
func Failure_Code_Invariants(value Failure_Code, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint8(uint8(value), FAILURE_CODE_MINIMUM, FAILURE_CODE_MAXIMUM).
    Ensure()
}

// Message is the sentence one failure code reads as. Each is a compile-time constant, thus
// naming a failure allocates nothing.
type Message string

// Message_Invariants states the band every failure sentence spans.
func Message_Invariants(value Message, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(len(value), MESSAGE_SIZE_MINIMUM, MESSAGE_SIZE_MAXIMUM).
    Ensure()
}

// Node_Kind is the syntactic class of one node.
type Node_Kind uint8

// Node_Kind_Invariants states the complete syntactic class domain.
func Node_Kind_Invariants(value Node_Kind, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint8(uint8(value), NODE_KIND_MINIMUM, NODE_KIND_MAXIMUM).
    Ensure()
}

// Index is one slot of the node arena, or INDEX_ABSENT for no node.
type Index int32

// Index_Invariants admits the absent slot, because a missing child is a slot number of its own.
func Index_Invariants(value Index, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int32(int32(value), INDEX_MINIMUM, INDEX_MAXIMUM).
    Ensure()
}

// Token_Index is one slot of the token run.
type Token_Index int32

// Token_Index_Invariants states the complete token run domain.
func Token_Index_Invariants(value Token_Index, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int32(int32(value), TOKEN_INDEX_MINIMUM, TOKEN_INDEX_MAXIMUM).
    Ensure()
}

// Token_Run is the borrowed token prefix one parse wrote.
type Token_Run []token.Token

// Token_Run_Invariants bounds the borrowed prefix inside Parse_State storage.
func Token_Run_Invariants(value Token_Run, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(len(value), TOKEN_RUN_SIZE_MINIMUM, TOKEN_COUNT_MAXIMUM).
    Ensure()
}

// Ancestor is the slot that holds one node as a child, or INDEX_ABSENT for a root. A parent
// takes its slot before its children, thus no parent can occupy the final slot of the arena.
type Ancestor int32

// Ancestor_Invariants excludes the final slot, which no node can hold as its parent, and the
// slot after the file, which opens the child chain and therefore holds no child of its own.
func Ancestor_Invariants(value Ancestor, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Holed_Int32(
    int32(value),
    INDEX_MINIMUM,
    ANCESTOR_MAXIMUM,
    LINK_HOLE_FIRST,
    LINK_HOLE_FIRST,
    LINK_HOLE_FIRST,
    LINK_HOLE_FIRST,
  ).
    Ensure()
}

// Head is the first child of one node, or INDEX_ABSENT when it has none.
type Head int32

// Head_Invariants holes the file slot, which is the root and therefore no node's child.
func Head_Invariants(value Head, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Holed_Int32(
    int32(value),
    INDEX_MINIMUM,
    INDEX_MAXIMUM,
    LINK_HOLE_ROOT,
    LINK_HOLE_ROOT,
    LINK_HOLE_ROOT,
    LINK_HOLE_ROOT,
  ).
    Ensure()
}

// Successor is the sibling after one node, or INDEX_ABSENT when it is the final child.
type Successor int32

// Successor_Invariants holes the file slot and the slot after it: the first opens no chain it
// belongs to and the second opens the chain of the file, so neither follows a sibling.
func Successor_Invariants(value Successor, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Holed_Int32(
    int32(value),
    INDEX_MINIMUM,
    INDEX_MAXIMUM,
    LINK_HOLE_ROOT,
    LINK_HOLE_FIRST,
    LINK_HOLE_FIRST,
    LINK_HOLE_FIRST,
  ).
    Ensure()
}

// Root is the file node one parse builds, or INDEX_ABSENT when the scan refused the source.
type Root int32

// Root_Invariants states the two slots a parse can return.
func Root_Invariants(value Root, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_Int32(int32(value), INDEX_MINIMUM, int32(INDEX_FIRST)).
    Ensure()
}

// Clause_Kind is the class of one simple statement.
type Clause_Kind int

// Clause_Kind_Invariants states the block of simple statement kinds.
func Clause_Kind_Invariants(value Clause_Kind, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(int(value), CLAUSE_KIND_MINIMUM, CLAUSE_KIND_MAXIMUM).
    Ensure()
}

// Literal_Kind is the class of one literal token.
type Literal_Kind int

// Literal_Kind_Invariants states the block of literal kinds.
func Literal_Kind_Invariants(value Literal_Kind, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(int(value), LITERAL_KIND_MINIMUM, LITERAL_KIND_MAXIMUM).
    Ensure()
}

// Suffix_Kind is the class a bracket suffix opens.
type Suffix_Kind int

// Suffix_Kind_Invariants states the three classes a bracket suffix can open.
func Suffix_Kind_Invariants(value Suffix_Kind, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_3_Int(
    int(value),
    SUFFIX_KIND_GENERIC,
    SUFFIX_KIND_INDEX,
    SUFFIX_KIND_SLICE,
  ).
    Ensure()
}

// Precedence is the binding level of a binary operator.
type Precedence int

// Precedence_Invariants admits the level of a token that binds nothing and the three levels this
// dialect states.
func Precedence_Invariants(value Precedence, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_4_Int(
    int(value),
    int(PRECEDENCE_NONE),
    int(PRECEDENCE_COMPARISON),
    int(PRECEDENCE_ADDITION),
    int(PRECEDENCE_MULTIPLICATION),
  ).
    Ensure()
}

// Boolean is a true or false report about one parse step.
type Boolean bool

// Boolean_Invariants states both parse reports as obligations.
func Boolean_Invariants(value Boolean, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Sometimes(bool(value), "The parse step report is true.").
    Ensure()
}

// Node is one slot of the arena: a syntactic class, the token that names it, and its place in
// the tree. Children hang off First_Child and thread through Next, thus a node of any arity
// costs the same seven fields. Every slot number and the token position stand over four-byte
// integers and the class over one byte, which is the whole reason a whole-file arena fits in
// single-digit megabytes; the class stands last so the six wide fields pack ahead of it.
type Node struct {
  // Token is the run position of the token that names this node.
  Token Token_Index
  // Parent is the slot that holds this node as a child, or INDEX_ABSENT for the root.
  Parent Ancestor
  // First_Child is the first child of this node, or INDEX_ABSENT when it has none.
  First_Child Head
  // Next is the sibling after this node, or INDEX_ABSENT when it is the final child.
  Next Successor
  // Kind is the syntactic class this node belongs to.
  Kind Node_Kind
}

// Node_Invariants composes the class, the naming token, and the five tree links of one node.
// Each link carries a type of its own, because one assertion tree holds one type at one
// position and five links of one type would give five paths one identity.
func Node_Invariants(value *Node, namespace aver.Namespace) {
  Node_Kind_Invariants(value.Kind, namespace)
  Token_Index_Invariants(value.Token, namespace)
  Ancestor_Invariants(value.Parent, namespace)
  Head_Invariants(value.First_Child, namespace)
  Successor_Invariants(value.Next, namespace)
}

// Tokens keeps complete lexical storage caller-owned and unable to grow.
type Tokens []token.Token

// Tokens_Invariants fixes lexical storage to parser bound.
func Tokens_Invariants(value Tokens, _ aver.Namespace) {
  aver.Always(len(value) == TOKEN_COUNT_MAXIMUM, "Token storage has complete length.")
  aver.Always(cap(value) == TOKEN_COUNT_MAXIMUM, "Token storage cannot grow.")
}

// Token_Storage keeps token slice representation composable under parse state.
type Token_Storage struct {
  // Values stays wrapped because validated state inherits this field.
  Values Tokens
}

// Token_Storage_Invariants fixes caller token memory.
func Token_Storage_Invariants(value *Token_Storage, namespace aver.Namespace) {
  Tokens_Invariants(value.Values, namespace)
}

// Nodes keeps complete syntax arena caller-owned and unable to grow.
type Nodes []Node

// Nodes_Invariants fixes syntax storage to parser bound.
func Nodes_Invariants(value Nodes, _ aver.Namespace) {
  aver.Always(len(value) == NODE_COUNT_MAXIMUM, "Node storage has complete length.")
  aver.Always(cap(value) == NODE_COUNT_MAXIMUM, "Node storage cannot grow.")
}

// Node_Storage keeps arena slice representation composable under parse state.
type Node_Storage struct {
  // Values stays wrapped because validated state inherits this field.
  Values Nodes
}

// Node_Storage_Invariants fixes caller arena memory.
func Node_Storage_Invariants(value *Node_Storage, namespace aver.Namespace) {
  Nodes_Invariants(value.Values, namespace)
}

// Parents keeps open-node ancestry caller-owned and unable to grow.
type Parents []Index

// Parents_Invariants fixes ancestry storage to nesting bound.
func Parents_Invariants(value Parents, _ aver.Namespace) {
  aver.Always(len(value) == DEPTH_MAXIMUM, "Parent storage has complete length.")
  aver.Always(cap(value) == DEPTH_MAXIMUM, "Parent storage cannot grow.")
}

// Parent_Storage keeps ancestry slice representation composable under parse state.
type Parent_Storage struct {
  // Values stays wrapped because validated state inherits this field.
  Values Parents
}

// Parent_Storage_Invariants fixes caller ancestry memory.
func Parent_Storage_Invariants(value *Parent_Storage, namespace aver.Namespace) {
  Parents_Invariants(value.Values, namespace)
}

// Last_Children keeps active sibling tails caller-owned and unable to grow.
type Last_Children []Index

// Last_Children_Invariants fixes sibling-tail storage to nesting bound.
func Last_Children_Invariants(value Last_Children, _ aver.Namespace) {
  aver.Always(len(value) == DEPTH_MAXIMUM, "Last-child storage has complete length.")
  aver.Always(cap(value) == DEPTH_MAXIMUM, "Last-child storage cannot grow.")
}

// Last_Child_Storage keeps sibling-tail slice representation composable under parse state.
type Last_Child_Storage struct {
  // Values stays wrapped because validated state inherits this field.
  Values Last_Children
}

// Last_Child_Storage_Invariants fixes caller sibling-tail memory.
func Last_Child_Storage_Invariants(value *Last_Child_Storage, namespace aver.Namespace) {
  Last_Children_Invariants(value.Values, namespace)
}

// Earlier_Children keeps prior sibling tails caller-owned and unable to grow.
type Earlier_Children []Index

// Earlier_Children_Invariants fixes prior-tail storage to nesting bound.
func Earlier_Children_Invariants(value Earlier_Children, _ aver.Namespace) {
  aver.Always(len(value) == DEPTH_MAXIMUM, "Earlier-child storage has complete length.")
  aver.Always(cap(value) == DEPTH_MAXIMUM, "Earlier-child storage cannot grow.")
}

// Earlier_Child_Storage keeps prior-tail slice representation composable under parse state.
type Earlier_Child_Storage struct {
  // Values stays wrapped because validated state inherits this field.
  Values Earlier_Children
}

// Earlier_Child_Storage_Invariants fixes caller prior-tail memory.
func Earlier_Child_Storage_Invariants(
  value *Earlier_Child_Storage, namespace aver.Namespace,
) {
  Earlier_Children_Invariants(value.Values, namespace)
}

// Syntax_Tree is the token run and the node arena one parse writes, which is all a reader of the
// tree walks.
type Syntax_Tree struct {
  // Tokens is the token run of the source, ending in one end-of-file token.
  Tokens Token_Storage
  // Nodes is the node arena. Slot zero holds no node.
  Nodes Node_Storage
  // Parents holds the node left open at each nesting depth.
  Parents Parent_Storage
  // Last_Children holds the final child of the open node at each nesting depth. It stands
  // here rather than in every node, because only an open node ever gains a child and a
  // field of its own would cost four bytes for every slot of the arena.
  Last_Children Last_Child_Storage
  // Earlier_Children holds the child before the final one at each nesting depth, which is
  // what an operator needs to unlink the operand it takes back.
  Earlier_Children Earlier_Child_Storage
}

// Syntax_Tree_Invariants composes every store the caller supplies.
func Syntax_Tree_Invariants(value *Syntax_Tree, namespace aver.Namespace) {
  Token_Storage_Invariants(&value.Tokens, namespace)
  Node_Storage_Invariants(&value.Nodes, namespace)
  Parent_Storage_Invariants(&value.Parents, namespace)
  Last_Child_Storage_Invariants(&value.Last_Children, namespace)
  Earlier_Child_Storage_Invariants(&value.Earlier_Children, namespace)
}

// Syntax_Tree_Handle gives the caller-owned tree storage one identity.
type Syntax_Tree_Handle *Syntax_Tree

// Syntax_Tree_Handle_Invariants composes present tree storage.
func Syntax_Tree_Handle_Invariants(value Syntax_Tree_Handle, namespace aver.Namespace) {
  if value == nil {
    return
  }
  Syntax_Tree_Invariants(value, namespace)
}

// Parse_State is the caller-owned parser storage.
type Parse_State struct {
  // Syntax_Tree is the tree the parse writes.
  Syntax_Tree
  // Failure is why the last parse stopped, or FAILURE_NONE when it ran clean.
  Failure Failure_Code
  // Failure_Token is the run position of the token the last parse refused. It carries no
  // meaning while Failure reads FAILURE_NONE.
  Failure_Token Token_Index
}

// Parse_State_Invariants composes the tree and the report.
func Parse_State_Invariants(value *Parse_State, namespace aver.Namespace) {
  Syntax_Tree_Invariants(&value.Syntax_Tree, namespace)
  Failure_Code_Invariants(value.Failure, namespace)
  Token_Index_Invariants(value.Failure_Token, namespace)
}

// Parse_State_Handle gives caller-owned parse storage one identity.
type Parse_State_Handle *Parse_State

// Parse_State_Handle_Invariants composes present parse storage.
func Parse_State_Handle_Invariants(value Parse_State_Handle, namespace aver.Namespace) {
  if value == nil {
    return
  }
  Parse_State_Invariants(value, namespace)
}

// Token_Count is how many tokens one scan wrote, the end token among them.
type Token_Count Token_Index

// Token_Count_Invariants states every run from the end token alone to the widest run.
func Token_Count_Invariants(value Token_Count, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int32(int32(value), TOKEN_RUN_SIZE_MINIMUM, TOKEN_COUNT_MAXIMUM).
    Ensure()
}

// Token_Position is the token the read stands on.
type Token_Position Token_Index

// Token_Position_Invariants states every token of the run.
func Token_Position_Invariants(value Token_Position, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int32(int32(value), TOKEN_INDEX_MINIMUM, TOKEN_INDEX_MAXIMUM).
    Ensure()
}

// Blank_Token is one past the final token whose run of empty lines the tree already holds.
type Blank_Token Token_Index

// Blank_Token_Invariants states one position past every token of the run.
func Blank_Token_Invariants(value Blank_Token, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int32(int32(value), TOKEN_INDEX_MINIMUM, TOKEN_COUNT_MAXIMUM).
    Ensure()
}

// Run_Cursors names each place the read of the run stands at.
type Run_Cursors struct {
  // Count bounds every read, because the run ends at the token the scan wrote last.
  Count Token_Count
  // Position is the token the next step reads.
  Position Token_Position
  // Blank keeps a run of empty lines out of the tree twice, because a step may ask for the
  // trivia at one token more than once.
  Blank Blank_Token
}

// Run_Cursors_Invariants composes each place the read stands at.
func Run_Cursors_Invariants(value *Run_Cursors, namespace aver.Namespace) {
  Token_Count_Invariants(value.Count, namespace)
  Token_Position_Invariants(value.Position, namespace)
  Blank_Token_Invariants(value.Blank, namespace)
}

// Arena_Count is how many slots the arena holds, the absent slot among them.
type Arena_Count Index

// Arena_Count_Invariants states every arena from the absent slot alone to the widest arena.
func Arena_Count_Invariants(value Arena_Count, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int32(
    int32(value),
    int32(ARENA_COUNT_MINIMUM),
    NODE_COUNT_MAXIMUM,
  ).
    Ensure()
}

// ARENA_COUNT_MINIMUM is the absent slot and the file. The file opens ahead of the program, thus
// no step of it reads an arena narrower.
const ARENA_COUNT_MINIMUM Arena_Count = 2

// DEPTH_MINIMUM is the file alone standing open, which is the shallowest a step of the program
// reads, because the file opens ahead of the program and never closes inside it.
const DEPTH_MINIMUM Parse_Depth = 1

// Parse_Depth is how many nodes stand open.
type Parse_Depth Index

// Parse_Depth_Invariants states every depth the parent stack holds. An open past the stack fails
// the parse before it counts, thus the count never passes the stack.
func Parse_Depth_Invariants(value Parse_Depth, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int32(int32(value), int32(DEPTH_MINIMUM), DEPTH_MAXIMUM).
    Ensure()
}

// Arena_Cursors names the extent of the arena and the depth of the open nodes.
type Arena_Cursors struct {
  // Count bounds every write, because the arena ends at the slot the parse took last.
  Count Arena_Count
  // Depth bounds the parent stack.
  Depth Parse_Depth
}

// Arena_Cursors_Invariants composes the arena extent and the open depth.
func Arena_Cursors_Invariants(value *Arena_Cursors, namespace aver.Namespace) {
  Arena_Count_Invariants(value.Count, namespace)
  Parse_Depth_Invariants(value.Depth, namespace)
}

// Plain_Brace marks a control clause, where an open brace starts a block rather than a composite
// literal, because `for x {` opens a body and never a literal of type x.
type Plain_Brace Boolean

// Plain_Brace_Invariants states both brace readings.
func Plain_Brace_Invariants(value Plain_Brace, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Sometimes(bool(value), "Brace opens plain block.").
    Ensure()
}

// Type_Assertion marks a switch head holding a type assertion over the type keyword, which is
// the one mark that tells a type switch from an expression switch.
type Type_Assertion Boolean

// Type_Assertion_Invariants states both assertion readings.
func Type_Assertion_Invariants(value Type_Assertion, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Sometimes(bool(value), "Clause holds type assertion.").
    Ensure()
}

// Machine_Flags names each reading the grammar carries across its rules.
type Machine_Flags struct {
  // Plain_Brace stands alone because control clauses own this ambiguity.
  Plain_Brace Plain_Brace
  // Type_Assertion stands alone because switch classification owns this ambiguity.
  Type_Assertion Type_Assertion
}

// Machine_Flags_Invariants composes each reading.
func Machine_Flags_Invariants(value *Machine_Flags, namespace aver.Namespace) {
  Plain_Brace_Invariants(value.Plain_Brace, namespace)
  Type_Assertion_Invariants(value.Type_Assertion, namespace)
}

// Arena is the tree one run writes: the slots, the open nodes, and their extent. It stands apart
// from the read of the run, because a write into the tree owes no place the read stands at.
type Arena struct {
  // Nodes is the arena the run writes.
  Nodes Node_Storage
  // Parents holds the node left open at each depth.
  Parents Parent_Storage
  // Last_Children holds the final child of the open node at each depth.
  Last_Children Last_Child_Storage
  // Earlier_Children holds the child before the final one at each depth.
  Earlier_Children Earlier_Child_Storage
  // Cursors holds the extent of the arena and the depth of the open nodes.
  Cursors Arena_Cursors
}

// Arena_Invariants composes every store and both cursors of the tree one run writes.
func Arena_Invariants(value *Arena, namespace aver.Namespace) {
  Node_Storage_Invariants(&value.Nodes, namespace)
  Parent_Storage_Invariants(&value.Parents, namespace)
  Last_Child_Storage_Invariants(&value.Last_Children, namespace)
  Earlier_Child_Storage_Invariants(&value.Earlier_Children, namespace)
  Arena_Cursors_Invariants(&value.Cursors, namespace)
}

// Arena_Handle gives the tree one run writes one identity.
type Arena_Handle *Arena

// Arena_Handle_Invariants composes a present tree.
func Arena_Handle_Invariants(value Arena_Handle, namespace aver.Namespace) {
  if value == nil {
    return
  }
  Arena_Invariants(value, namespace)
}

// Opened_Index is the slot a step of the program opened, or INDEX_ABSENT where it opened none.
// The file takes the first slot ahead of the program, thus no step opens it.
type Opened_Index Index

// Opened_Index_Invariants states every slot a step opens.
func Opened_Index_Invariants(value Opened_Index, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Holed_Int32(
    int32(value),
    INDEX_MINIMUM,
    INDEX_MAXIMUM,
    LINK_HOLE_ROOT,
    LINK_HOLE_ROOT,
    LINK_HOLE_ROOT,
    LINK_HOLE_ROOT,
  ).
    Ensure()
}

// Machine_Flags_Handle gives the readings of one run one identity.
type Machine_Flags_Handle *Machine_Flags

// Machine_Flags_Handle_Invariants composes present readings.
func Machine_Flags_Handle_Invariants(value Machine_Flags_Handle, namespace aver.Namespace) {
  if value == nil {
    return
  }
  Machine_Flags_Invariants(value, namespace)
}

// Machine is one run of the grammar program over one scanned source. It lives in the frame of
// the parse that runs it, thus its cursors never outlive the parse and a reader of the tree owes
// none of them.
type Machine struct {
  // Tokens is the run the scan wrote.
  Tokens Token_Storage
  // Run holds the places the read stands at.
  Run Run_Cursors
  // Arena holds the tree the run writes.
  Arena Arena
  // Flags holds the readings the grammar carries across its rules.
  Flags Machine_Flags
}

// Machine_Invariants composes every store and every cursor of one run.
func Machine_Invariants(value *Machine, namespace aver.Namespace) {
  Token_Storage_Invariants(&value.Tokens, namespace)
  Run_Cursors_Invariants(&value.Run, namespace)
  Arena_Invariants(&value.Arena, namespace)
  Machine_Flags_Invariants(&value.Flags, namespace)
}

// Machine_Handle gives one run of the grammar program one identity.
type Machine_Handle *Machine

// Machine_Handle_Invariants composes a present run.
func Machine_Handle_Invariants(value Machine_Handle, namespace aver.Namespace) {
  if value == nil {
    return
  }
  Machine_Invariants(value, namespace)
}

// Node_Count is how many slots one parse wrote, the absent slot among them.
type Node_Count Index

// Node_Count_Invariants states every arena a parse returns. A refused scan writes no node, and a
// parse that runs writes the file and the first token of its clause, thus no parse returns the
// file node alone.
func Node_Count_Invariants(value Node_Count, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Holed_Int32(
    int32(value),
    int32(INDEX_FIRST),
    NODE_COUNT_MAXIMUM,
    NODE_COUNT_BARE,
    NODE_COUNT_BARE,
    NODE_COUNT_BARE,
    NODE_COUNT_BARE,
  ).
    Ensure()
}

// Parse_Result is what one parse returns: the root it built, the extent of the run and the arena it
// wrote, and why it stopped. It stands apart from the caller storage, because a reader of the
// storage walks the tree by its links and never owes a count one parse wrote.
type Parse_Result struct {
  // Root is the file node, or INDEX_ABSENT when the scan refused the source.
  Root Root
  // Token_Count bounds the run the scan wrote.
  Token_Count Token_Count
  // Node_Count bounds the arena the parse wrote.
  Node_Count Node_Count
  // Failure is why the parse stopped, or FAILURE_NONE when it ran clean.
  Failure Failure_Code
  // Failure_Token is the run position of the token the parse refused. It carries no meaning
  // while Failure reads FAILURE_NONE.
  Failure_Token Token_Index
  // OK reports whether the parse ran clean, which is Failure reading FAILURE_NONE.
  OK Boolean
}

// Parse_Result_Invariants composes one parse outcome.
func Parse_Result_Invariants(value *Parse_Result, namespace aver.Namespace) {
  Root_Invariants(value.Root, namespace)
  Token_Count_Invariants(value.Token_Count, namespace)
  Node_Count_Invariants(value.Node_Count, namespace)
  Failure_Code_Invariants(value.Failure, namespace)
  Token_Index_Invariants(value.Failure_Token, namespace)
  Boolean_Invariants(value.OK, namespace)
}

// Refusal is why one run of the grammar stopped, and the token it stopped at.
type Refusal struct {
  // Cause is why the run stopped, or FAILURE_NONE when it returned.
  Cause Failure_Code
  // Token is the token the run stood on when it stopped.
  Token Token_Index
}

// Refusal_Invariants composes one stop.
func Refusal_Invariants(value *Refusal, namespace aver.Namespace) {
  Failure_Code_Invariants(value.Cause, namespace)
  Token_Index_Invariants(value.Token, namespace)
}

// Opened_Kind is the class of a node one step of the program opens. The file opens ahead of the
// program, thus the comment is the smallest class a step opens.
type Opened_Kind Node_Kind

// Opened_Kind_Invariants states every class a step opens.
func Opened_Kind_Invariants(value Opened_Kind, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint8(uint8(value), uint8(NODE_COMMENT), NODE_KIND_MAXIMUM).
    Ensure()
}

// Arena_Failure is why one write into the arena stopped: the arena ran out, the parent stack ran
// out, or neither.
type Arena_Failure Failure_Code

// Arena_Failure_Invariants states the two bounds one write can overrun and the write that
// overran neither.
func Arena_Failure_Invariants(value Arena_Failure, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_3_Uint8(
    uint8(value),
    uint8(FAILURE_NONE),
    uint8(FAILURE_NODE_COUNT),
    uint8(FAILURE_NESTING_DEPTH),
  ).
    Ensure()
}

// ARENA_INTACT is a write that overran neither bound.
const ARENA_INTACT = Arena_Failure(FAILURE_NONE)

// Name_Failure is why one run of names stopped: a token that is no name, an exhausted arena, an
// exhausted parent stack, or none of them.
type Name_Failure Failure_Code

// Name_Failure_Invariants states the four ways a run of names ends.
func Name_Failure_Invariants(value Name_Failure, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_4_Uint8(
    uint8(value),
    uint8(FAILURE_NONE),
    uint8(FAILURE_SYNTAX),
    uint8(FAILURE_NODE_COUNT),
    uint8(FAILURE_NESTING_DEPTH),
  ).
    Ensure()
}

// Wrap_Failure is why one wrap stopped: the arena ran out, or it did not.
type Wrap_Failure Failure_Code

// Wrap_Failure_Invariants states the one bound a wrap can overrun and the wrap that overran none.
func Wrap_Failure_Invariants(value Wrap_Failure, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_Uint8(
    uint8(value),
    uint8(FAILURE_NONE),
    uint8(FAILURE_NODE_COUNT),
  ).
    Ensure()
}

// WRAP_INTACT is a wrap that overran no bound.
const WRAP_INTACT = Wrap_Failure(FAILURE_NONE)

// Child is one slot a node is linked into. The file takes the first slot ahead of the program and
// links nowhere, thus a link names the slot after it at the least.
type Child Index

// Child_Invariants states every slot a link can name.
func Child_Invariants(value Child, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int32(int32(value), LINK_HOLE_FIRST, INDEX_MAXIMUM).
    Ensure()
}

// OPCODE_RETURN ends the rule that runs and hands the read back to the rule that called it.
const OPCODE_RETURN Opcode = 0

// OPCODE_CALL runs the rule its operand names and resumes behind it once that rule returns.
const OPCODE_CALL Opcode = 1

// OPCODE_TAIL runs the rule its operand names in the place of the one that runs, which is how a
// rule repeats itself without one frame for each round.
const OPCODE_TAIL Opcode = 2

// OPCODE_WHEN runs the step behind it only where the last test held.
const OPCODE_WHEN Opcode = 3

// OPCODE_UNLESS runs the step behind it only where the last test failed.
const OPCODE_UNLESS Opcode = 4

// OPCODE_AT tests whether the token at the read is of the kind its operand names.
const OPCODE_AT Opcode = 5

// OPCODE_OR_AT holds where the last test held, or where the token at the read is of the kind its
// operand names.
const OPCODE_OR_AT Opcode = 6

// OPCODE_AFTER tests whether the token behind the read is of the kind its operand names.
const OPCODE_AFTER Opcode = 7

// OPCODE_OR_AFTER holds where the last test held, or where the token behind the read is of the
// kind its operand names.
const OPCODE_OR_AFTER Opcode = 8

// OPCODE_AND_AFTER holds where the last test held and the token behind the read is of the kind
// its operand names.
const OPCODE_AND_AFTER Opcode = 9

// OPCODE_ACCEPT tests whether the token at the read is of the kind its operand names, and steps
// past it where it is.
const OPCODE_ACCEPT Opcode = 10

// OPCODE_OPENS_TYPE tests whether the token at the read can open a type.
const OPCODE_OPENS_TYPE Opcode = 11

// OPCODE_AFTER_OPENS_TYPE tests whether the token behind the read can open a type.
const OPCODE_AFTER_OPENS_TYPE Opcode = 12

// OPCODE_ASSERTED tests whether the switch head just read held a type assertion over the type
// keyword.
const OPCODE_ASSERTED Opcode = 13

// OPCODE_REFUSES_SIGN tests whether the token at the read is a sign this dialect refuses.
const OPCODE_REFUSES_SIGN Opcode = 14

// OPCODE_ABOVE tests whether the operator at the read binds tighter than the level the rule
// runs at.
const OPCODE_ABOVE Opcode = 15

// OPCODE_CLAUSE_OPERATOR tests whether the token at the read names the form of a simple statement.
const OPCODE_CLAUSE_OPERATOR Opcode = 16

// OPCODE_TYPE_PARAMETERS tests whether the bracket at the read opens a type parameter list
// rather than an array capacity.
const OPCODE_TYPE_PARAMETERS Opcode = 17

// OPCODE_QUALIFIES tests whether the name behind the comma at the read opens a qualified or a
// generic type rather than one more parameter name.
const OPCODE_QUALIFIES Opcode = 18

// OPCODE_FIELD_NAMED tests whether the field at the read spells its name out.
const OPCODE_FIELD_NAMED Opcode = 19

// OPCODE_BARRED_BRACE tests whether a control clause bars the brace at the read from opening a
// composite literal.
const OPCODE_BARRED_BRACE Opcode = 20

// OPCODE_ADVANCE steps past the token at the read and takes the trivia behind it.
const OPCODE_ADVANCE Opcode = 21

// OPCODE_SKIP_COMMENTS takes the trivia at the read into the open node.
const OPCODE_SKIP_COMMENTS Opcode = 22

// OPCODE_OPEN opens a node of the kind its operand names at the read.
const OPCODE_OPEN Opcode = 23

// OPCODE_CLOSE closes the open node.
const OPCODE_CLOSE Opcode = 24

// OPCODE_LEAF writes a node of the kind its operand names at the read, holding no child.
const OPCODE_LEAF Opcode = 25

// OPCODE_LITERAL writes a node of the literal kind the token at the read names and steps past
// that token.
const OPCODE_LITERAL Opcode = 26

// OPCODE_WRAP opens a node of the kind its operand names in the place of the final child of the
// open node and gives it that child.
const OPCODE_WRAP Opcode = 27

// OPCODE_WRAP_SUFFIX wraps the final child in the kind the bracket suffix at the read opens.
const OPCODE_WRAP_SUFFIX Opcode = 28

// OPCODE_RETYPE gives the node the rule opened the kind its operand names.
const OPCODE_RETYPE Opcode = 29

// OPCODE_RETYPE_CLAUSE gives the node the rule opened the form the operator at the read names.
const OPCODE_RETYPE_CLAUSE Opcode = 30

// OPCODE_UNNAME reads every name of the open node back to an identifier.
const OPCODE_UNNAME Opcode = 31

// OPCODE_REJECT refuses the token at the read for the cause its operand names.
const OPCODE_REJECT Opcode = 32

// OPCODE_SET_PLAIN marks the read as standing in a control clause.
const OPCODE_SET_PLAIN Opcode = 33

// OPCODE_CLEAR_PLAIN marks the read as standing in no control clause.
const OPCODE_CLEAR_PLAIN Opcode = 34

// OPCODE_SAVE_PLAIN keeps the control clause reading for the rule that runs.
const OPCODE_SAVE_PLAIN Opcode = 35

// OPCODE_RESTORE_PLAIN gives the read back the control clause reading the rule kept.
const OPCODE_RESTORE_PLAIN Opcode = 36

// OPCODE_SET_ASSERTION marks the switch head as holding a type assertion.
const OPCODE_SET_ASSERTION Opcode = 37

// OPCODE_CLEAR_ASSERTION marks the switch head as holding no type assertion.
const OPCODE_CLEAR_ASSERTION Opcode = 38

// OPCODE_HOLD keeps the level the operator at the read binds at.
const OPCODE_HOLD Opcode = 39

// OPCODE_CALL_HELD runs the binary rule at the level the rule kept.
const OPCODE_CALL_HELD Opcode = 40

// OPCODE_SKIP_EMPTY takes the trivia at the read and every empty statement among it.
const OPCODE_SKIP_EMPTY Opcode = 41

// OPCODE_UNARY_SIGN tests whether the token at the read is a sign a unary expression opens with.
const OPCODE_UNARY_SIGN Opcode = 42

// OPCODE_SUFFIX_SIGN tests whether the token at the read opens a suffix: a selector, a bracket, a
// call, or a composite literal.
const OPCODE_SUFFIX_SIGN Opcode = 43

// OPCODE_TAKE writes a node of the kind its operand names at the read, holding no child, and
// steps past the token it stands at.
const OPCODE_TAKE Opcode = 44

// OPCODE_NAME takes the identifier at the read as a node of the kind its operand names, and
// refuses any other token for its syntax.
const OPCODE_NAME Opcode = 45

// OPCODE_NAMES takes each identifier of one comma-joined run as a node of the kind its operand
// names, and refuses a run that holds anything else for its syntax.
const OPCODE_NAMES Opcode = 46

// OPCODE_STARS opens a pointer type at each star of one run, stepping past the channel keyword
// a star stands ahead of, and keeps the first node it opened.
const OPCODE_STARS Opcode = 47

// OPCODE_CLOSE_STARS closes every pointer type the run opened, down to the first.
const OPCODE_CLOSE_STARS Opcode = 48

// OPCODE_MINIMUM is the return, the smallest opcode.
const OPCODE_MINIMUM = uint8(OPCODE_RETURN)

// OPCODE_MAXIMUM is the close of a run of stars, the largest opcode.
const OPCODE_MAXIMUM = uint8(OPCODE_CLOSE_STARS)

// Opcode is the action of one step of the grammar program.
type Opcode uint8

// Opcode_Invariants states every action the program holds.
func Opcode_Invariants(value Opcode, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint8(uint8(value), OPCODE_MINIMUM, OPCODE_MAXIMUM).
    Ensure()
}

// OPERAND_MINIMUM is the end-of-file kind, the smallest operand.
const OPERAND_MINIMUM = uint8(token.KIND_END_OF_FILE)

// OPERAND_MAXIMUM is the final rule, the largest operand. Every token kind and every node kind
// stands below it, thus one byte names whatever a step acts on.
const OPERAND_MAXIMUM = uint8(PRODUCTION_GROUP_VALUE_TAIL)

// Operand is what one step of the grammar program acts on: a token kind, a node kind, a cause,
// or a rule, as its action reads it.
type Operand uint8

// Operand_Invariants states every operand the program holds.
func Operand_Invariants(value Operand, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint8(uint8(value), OPERAND_MINIMUM, OPERAND_MAXIMUM).
    Ensure()
}

// PRODUCTION_MINIMUM is the file, the smallest rule.
const PRODUCTION_MINIMUM = uint8(PRODUCTION_FILE)

// PRODUCTION_MAXIMUM is the value of one grouped member, the largest rule.
const PRODUCTION_MAXIMUM = uint8(PRODUCTION_GROUP_VALUE_TAIL)

// Production is one rule of the grammar program.
type Production uint8

// Production_Invariants states every rule the program holds.
func Production_Invariants(value Production, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint8(uint8(value), PRODUCTION_MINIMUM, PRODUCTION_MAXIMUM).
    Ensure()
}

// STEP_SIZE is the byte count of one step of the program: its action and its operand.
const STEP_SIZE = 2

// STEP_COUNT is how many steps the program holds. Registration reads no length of a string,
// thus the count is spelled out and the bounds test holds it to the program.
const STEP_COUNT = 959

// PLACE_MINIMUM is the first step of the program.
const PLACE_MINIMUM = 0

// PLACE_MAXIMUM is the final step of the program.
const PLACE_MAXIMUM = STEP_COUNT - 1

// Program_Place is the step of the program one rule reads.
type Program_Place int32

// Program_Place_Invariants states every step of the program.
func Program_Place_Invariants(value Program_Place, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int32(int32(value), PLACE_MINIMUM, int32(PLACE_MAXIMUM)).
    Ensure()
}

// ENTRY_PLACE_SECOND is the step behind the file's first one, which opens no rule.
const ENTRY_PLACE_SECOND = 1

// ENTRY_PLACE_THIRD is the step two behind the file's first one, which opens no rule.
const ENTRY_PLACE_THIRD = 2

// Entry_Place is the step one rule opens at.
type Entry_Place int32

// Entry_Place_Invariants states the step each rule opens at. The file rule runs longer than two
// steps, thus the rule behind it opens past the second and the third step of the program.
func Entry_Place_Invariants(value Entry_Place, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Holed_Int32(
    int32(value),
    ENTRY_FILE,
    int32(ENTRY_MAXIMUM),
    ENTRY_PLACE_SECOND,
    ENTRY_PLACE_THIRD,
    ENTRY_PLACE_THIRD,
    ENTRY_PLACE_THIRD,
  ).
    Ensure()
}

// FRAME_MINIMUM is the file rule, which stands beneath no other.
const FRAME_MINIMUM = 0

// Frame_Depth is how many rules stand open beneath the one that runs.
type Frame_Depth int32

// Frame_Depth_Invariants states every depth the program may call itself to.
func Frame_Depth_Invariants(value Frame_Depth, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int32(int32(value), FRAME_MINIMUM, FRAME_MAXIMUM).
    Ensure()
}

// Held_Level is the level of the operator one binary rule stepped past.
type Held_Level Precedence

// Held_Level_Invariants admits the level of a token that binds nothing and the three levels this
// dialect states.
func Held_Level_Invariants(value Held_Level, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_4_Int(
    int(value),
    int(PRECEDENCE_NONE),
    int(PRECEDENCE_COMPARISON),
    int(PRECEDENCE_ADDITION),
    int(PRECEDENCE_MULTIPLICATION),
  ).
    Ensure()
}

// Callee_Level is the level the rule one step calls runs at.
type Callee_Level Precedence

// Callee_Level_Invariants admits the level of a rule that holds no operator and the three levels
// this dialect states.
func Callee_Level_Invariants(value Callee_Level, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_4_Int(
    int(value),
    int(PRECEDENCE_NONE),
    int(PRECEDENCE_COMPARISON),
    int(PRECEDENCE_ADDITION),
    int(PRECEDENCE_MULTIPLICATION),
  ).
    Ensure()
}

// CONTROL_CONTINUE reads the next step of the rule that runs.
const CONTROL_CONTINUE Control = 0

// CONTROL_RETURN ends the rule that runs.
const CONTROL_RETURN Control = 1

// CONTROL_CALL runs the rule the step named and resumes behind it.
const CONTROL_CALL Control = 2

// CONTROL_REFUSE stops the whole run.
const CONTROL_REFUSE Control = 3

// Control is what the rule that runs does once one step of it ran.
type Control uint8

// Control_Invariants states the four things a rule does next.
func Control_Invariants(value Control, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_4_Uint8(
    uint8(value),
    uint8(CONTROL_CONTINUE),
    uint8(CONTROL_RETURN),
    uint8(CONTROL_CALL),
    uint8(CONTROL_REFUSE),
  ).
    Ensure()
}

// Frame is the state one rule keeps for itself while it runs. It lives in the frame of the rule,
// thus a rule it calls holds a frame of its own and hands this one back untouched.
type Frame struct {
  // Place is the step the rule reads next.
  Place Program_Place
  // Holds is whether the last test held.
  Holds Boolean
  // Opened is the node the rule opened last, which a later step may give another kind.
  Opened Opened_Index
  // Plain is the control clause reading the rule kept.
  Plain Plain_Brace
  // Held is the level of the operator the rule stepped past last.
  Held Held_Level
  // Level is the level the rule runs at, which only a binary rule reads.
  Level Precedence
  // Depth is how many rules stand open beneath this one.
  Depth Frame_Depth
  // Callee is the rule a call step named.
  Callee Production
  // Callee_Level is the level the rule a call step named runs at.
  Callee_Level Callee_Level
}

// Frame_Invariants composes the state of one rule.
func Frame_Invariants(value *Frame, namespace aver.Namespace) {
  Program_Place_Invariants(value.Place, namespace)
  Boolean_Invariants(value.Holds, namespace)
  Opened_Index_Invariants(value.Opened, namespace)
  Plain_Brace_Invariants(value.Plain, namespace)
  Held_Level_Invariants(value.Held, namespace)
  Precedence_Invariants(value.Level, namespace)
  Frame_Depth_Invariants(value.Depth, namespace)
  Production_Invariants(value.Callee, namespace)
  Callee_Level_Invariants(value.Callee_Level, namespace)
}

// Frame_Handle gives the state of one rule one identity.
type Frame_Handle *Frame

// Frame_Handle_Invariants composes a present rule state.
func Frame_Handle_Invariants(value Frame_Handle, namespace aver.Namespace) {
  if value == nil {
    return
  }
  Frame_Invariants(value, namespace)
}

// STEP_BUDGET caps the steps one rule runs. Every rule steps the read or returns within a bounded
// run of steps, thus a run that spends the budget misreads the grammar and stops rather than
// spinning.
const STEP_BUDGET = TOKEN_COUNT_MAXIMUM * 64

// Parse scans a source into the token run and runs the grammar program over it into the arena. A
// source the grammar rejects still returns the tree that was built, with an error node at the
// token that failed, because a linter reads more from a partial tree than from nothing.
func Parse(subject Parse_State_Handle, source token.Source) (tree Parse_Result) {
  defer func() { Parse_Result_Invariants(&tree, "parse.tree") }()
  Parse_State_Handle_Invariants(subject, "parse.subject")
  token.Source_Invariants(source, "parse.source")
  // Slot zero carries the error kind so a stray read of the absent slot is loud rather than a
  // plausible node.
  subject.Nodes.Values[INDEX_ABSENT] = Node{
    Kind:        NODE_ERROR,
    Token:       0,
    Parent:      Ancestor(INDEX_ABSENT),
    First_Child: Head(INDEX_ABSENT),
    Next:        Successor(INDEX_ABSENT),
  }
  held := &subject.Syntax_Tree
  count, scanned := scan_source(held, source)
  tree = Parse_Result{
    Root:          Root(INDEX_ABSENT),
    Token_Count:   count,
    Node_Count:    Node_Count(INDEX_FIRST),
    Failure:       FAILURE_TOKEN_COUNT,
    Failure_Token: 0,
  }
  opened := Arena_Cursors{Count: ARENA_COUNT_MINIMUM, Depth: DEPTH_MINIMUM}
  machine := Machine{
    Tokens: subject.Tokens,
    Run:    Run_Cursors{Count: count, Position: 0, Blank: 0},
    Arena: Arena{
      Nodes:            subject.Nodes,
      Parents:          subject.Parents,
      Last_Children:    subject.Last_Children,
      Earlier_Children: subject.Earlier_Children,
      Cursors:          opened,
    },
    Flags: Machine_Flags{Plain_Brace: false, Type_Assertion: false},
  }
  if bool(scanned) {
    // The file opens ahead of the program, thus every step of it reads an arena that
    // holds the file and stands inside it.
    subject.Nodes.Values[INDEX_FIRST] = Node{
      Kind:        NODE_FILE,
      Token:       0,
      Parent:      Ancestor(INDEX_ABSENT),
      First_Child: Head(INDEX_ABSENT),
      Next:        Successor(INDEX_ABSENT),
    }
    subject.Parents.Values[0] = INDEX_FIRST
    subject.Last_Children.Values[0] = INDEX_ABSENT
    subject.Earlier_Children.Values[0] = INDEX_ABSENT
    refusal := descend(&machine, PRODUCTION_FILE, PRECEDENCE_NONE, 0)
    tree.Root = Root(INDEX_FIRST)
    tree.Node_Count = Node_Count(machine.Arena.Cursors.Count)
    tree.Failure = refusal.Cause
    tree.Failure_Token = refusal.Token
    tree.OK = Boolean(refusal.Cause == FAILURE_NONE)
  }
  tree = name_wide_identifier(held, tree, source)
  // The import pass runs first, because an import stands ahead of every declaration and the
  // failure a caller reads should be the one its author meets first.
  tree = name_blank_import(held, tree, source)
  tree = name_constant_case(held, tree, source)
  tree = name_bare_loop(held, tree, source)
  tree = name_fallthrough(held, tree)
  tree = name_unnamed_result(held, tree)
  tree = name_closure_statement(held, tree)
  tree = name_closure_call(held, tree)
  tree = name_banned_identifier(held, tree, source)
  subject.Failure = tree.Failure
  subject.Failure_Token = tree.Failure_Token
  empty_tokens(held, tree.Token_Count)
  empty_nodes(held, tree.Node_Count)
  return tree
}

// Empties every token past the run one parse wrote, up to the first slot no parse wrote. A parse
// before this one may have written a longer run, and a token it left would read as part of this
// one.
func empty_tokens(subject Syntax_Tree_Handle, count Token_Count) {
  Syntax_Tree_Handle_Invariants(subject, "empty_tokens.subject")
  Token_Count_Invariants(count, "empty_tokens.count")
  for slot := int(count); slot < TOKEN_COUNT_MAXIMUM; slot++ {
    if subject.Tokens.Values[slot] == (token.Token{}) {
      return
    }
    subject.Tokens.Values[slot] = token.Token{}
  }
}

// Empties every slot past the arena one parse wrote, up to the first slot no parse wrote. A parse
// before this one may have written a wider arena, and a slot it left would read as a node of this
// one.
func empty_nodes(subject Syntax_Tree_Handle, count Node_Count) {
  Syntax_Tree_Handle_Invariants(subject, "empty_nodes.subject")
  Node_Count_Invariants(count, "empty_nodes.count")
  for slot := int(count); slot < NODE_COUNT_MAXIMUM; slot++ {
    if subject.Nodes.Values[slot] == (Node{}) {
      return
    }
    subject.Nodes.Values[slot] = Node{}
  }
}

// Node_At returns one slot of the arena. The parse writes every slot it counts, thus a slot the
// count does not reach is no node and reading it is a caller defect.
func Node_At(subject Parse_State_Handle, index Index) (node Node) {
  defer func() { Node_Invariants(&node, "node_at.node") }()
  Parse_State_Handle_Invariants(subject, "node_at.subject")
  Index_Invariants(index, "node_at.index")
  return Syntax_Node_At(&subject.Syntax_Tree, index)
}

// Token_At returns one token of the run, the token that names a node.
func Token_At(subject Parse_State_Handle, index Token_Index) (one token.Token) {
  defer func() { token.Token_Invariants(&one, "token_at.one") }()
  Parse_State_Handle_Invariants(subject, "token_at.subject")
  Token_Index_Invariants(index, "token_at.index")
  return Syntax_Token_At(&subject.Syntax_Tree, index)
}

// Syntax_Node_At returns one slot of the arena for a reader that holds the tree alone.
func Syntax_Node_At(subject Syntax_Tree_Handle, index Index) (node Node) {
  defer func() { Node_Invariants(&node, "syntax_node_at.node") }()
  Syntax_Tree_Handle_Invariants(subject, "syntax_node_at.subject")
  Index_Invariants(index, "syntax_node_at.index")
  // Parse empties every slot past the arena it wrote, and no slot it wrote is empty.
  aver.Always(
    subject.Nodes.Values[index] != (Node{}),
    "A read node lies inside the count the parse wrote.",
  )
  return subject.Nodes.Values[index]
}

// Syntax_Token_At returns one token of the run for a reader that holds the tree alone.
func Syntax_Token_At(subject Syntax_Tree_Handle, index Token_Index) (one token.Token) {
  defer func() { token.Token_Invariants(&one, "syntax_token_at.one") }()
  Syntax_Tree_Handle_Invariants(subject, "syntax_token_at.subject")
  Token_Index_Invariants(index, "syntax_token_at.index")
  // The run closes at its end token, and Parse empties every token past it, thus a token
  // that stands behind an end token lies past the run.
  inside := index == 0
  if !inside {
    inside = subject.Tokens.Values[index-1].Kind != token.KIND_END_OF_FILE
  }
  aver.Always(inside, "A read token lies inside the count the scan wrote.")
  return subject.Tokens.Values[index]
}

// Parse_State_Token_Run borrows the token run the last parse wrote. The run closes at its end
// token, or fills the whole storage where the scan ran past the bound, thus a caller that holds
// the state alone reads it.
func Parse_State_Token_Run(subject Parse_State_Handle) (run Token_Run) {
  defer func() { Token_Run_Invariants(run, "parse_state_token_run.run") }()
  Parse_State_Handle_Invariants(subject, "parse_state_token_run.subject")
  for index := range TOKEN_COUNT_MAXIMUM {
    if subject.Tokens.Values[index].Kind == token.KIND_END_OF_FILE {
      return Token_Run(subject.Tokens.Values[:index+1])
    }
  }
  return Token_Run(subject.Tokens.Values)
}

// Failure names why the last parse on the state stopped, or FAILURE_NONE when it ran clean.
func Failure(subject Parse_State_Handle) (code Failure_Code) {
  defer func() { Failure_Code_Invariants(code, "failure.code") }()
  Parse_State_Handle_Invariants(subject, "failure.subject")
  return subject.Failure
}

// Failure_Token names the run position of the token the last parse on the state refused. It
// carries no meaning while Failure reads FAILURE_NONE.
func Failure_Token(subject Parse_State_Handle) (position Token_Index) {
  defer func() { Token_Index_Invariants(position, "failure_token.position") }()
  Parse_State_Handle_Invariants(subject, "failure_token.subject")
  return subject.Failure_Token
}

// Fills the token run from the source and reports how many tokens it wrote. A run past the bound
// fails the parse rather than writing outside the array, and the run it wrote stays readable, so a
// caller that reads the failure token finds a token there.
func scan_source(
  subject Syntax_Tree_Handle,
  source token.Source,
) (count Token_Count, scanned Boolean) {
  defer func() {
    Token_Count_Invariants(count, "scan_source.count")
    Boolean_Invariants(scanned, "scan_source.scanned")
  }()
  Syntax_Tree_Handle_Invariants(subject, "scan_source.subject")
  token.Source_Invariants(source, "scan_source.source")
  scanner := token.Scanner{
    Source:   source,
    Offset:   0,
    Previous: token.KIND_END_OF_FILE,
    Newline:  false,
  }
  for written := range TOKEN_COUNT_MAXIMUM {
    one := token.Scan(&scanner)
    subject.Tokens.Values[written] = one
    if one.Kind == token.KIND_END_OF_FILE {
      return Token_Count(written + 1), true
    }
  }
  return TOKEN_COUNT_MAXIMUM, false
}

// Runs one rule of the grammar program until it returns, and reports why the run stopped, with the
// token it stopped at, or FAILURE_NONE when the rule returned.
func descend(
  machine Machine_Handle,
  production Production,
  level Precedence,
  depth Frame_Depth,
) (refusal Refusal) {
  defer func() { Refusal_Invariants(&refusal, "descend.refusal") }()
  Machine_Handle_Invariants(machine, "descend.machine")
  Production_Invariants(production, "descend.production")
  Precedence_Invariants(level, "descend.level")
  Frame_Depth_Invariants(depth, "descend.depth")
  frame := Frame{
    Place:        Program_Place(production_entry(production)),
    Holds:        false,
    Opened:       Opened_Index(INDEX_ABSENT),
    Plain:        machine.Flags.Plain_Brace,
    Held:         0,
    Level:        level,
    Depth:        depth,
    Callee:       PRODUCTION_FILE,
    Callee_Level: 0,
  }
  for range STEP_BUDGET {
    frame.Place = guarded(frame.Place, frame.Holds)
    control, stopped := step(machine, &frame)
    if control == CONTROL_REFUSE {
      return stopped
    }
    if control == CONTROL_RETURN {
      return Refusal{Cause: FAILURE_NONE, Token: 0}
    }
    if control == CONTROL_CALL {
      callee := Precedence(frame.Callee_Level)
      inner := descend(machine, frame.Callee, callee, frame.Depth+1)
      if inner.Cause != FAILURE_NONE {
        return inner
      }
    }
  }
  return Refusal{Cause: FAILURE_SYNTAX, Token: Token_Index(machine.Run.Position)}
}

// Runs one step of the rule a frame holds and names what the rule does next. Every step runs
// through every unit and each unit acts on its own actions alone, because a unit that ran for its
// own actions alone would owe every place the read stands at where it never stands.
func step(machine Machine_Handle, frame Frame_Handle) (control Control, refusal Refusal) {
  defer func() {
    Control_Invariants(control, "step.control")
    Refusal_Invariants(&refusal, "step.refusal")
  }()
  Machine_Handle_Invariants(machine, "step.machine")
  Frame_Handle_Invariants(frame, "step.frame")
  op, operand := decode(frame.Place)
  position := Token_Index(machine.Run.Position)
  at, after, second := window(machine.Tokens, machine.Run)
  closes := closes_field(machine.Tokens, machine.Run)
  barred := barred_brace(
    machine.Arena.Nodes,
    machine.Arena.Last_Children,
    machine.Arena.Cursors,
    machine.Flags.Plain_Brace,
  )
  asserted := machine.Flags.Type_Assertion
  frame.Holds = test(op, operand, frame.Holds, frame.Level, at, after, asserted)
  frame.Holds = test_scan(op, frame.Holds, at, after, second, closes, barred)
  frame.Held = hold(op, frame.Held, at)
  frame.Plain = keep_plain(&machine.Flags, op, frame.Plain)
  suffix := bracket_kind(machine.Tokens, machine.Run, op)
  // A leaf stands at the token the step reads, thus the shape runs ahead of the move that
  // steps past that token.
  cause := Failure_Code(reshape(machine, op, operand, suffix))
  named := take_names(machine, op, operand)
  starred, run := open_stars(machine, op, frame.Opened)
  opened, shaped := shape(machine, op, operand, starred)
  frame.Opened = opened
  moved := move(machine, op, frame.Holds)
  if cause == FAILURE_NONE {
    cause = shaped
  }
  if cause == FAILURE_NONE {
    cause = Failure_Code(named)
  }
  if cause == FAILURE_NONE {
    cause = Failure_Code(run)
  }
  if cause == FAILURE_NONE {
    cause = Failure_Code(moved)
  }
  if cause != FAILURE_NONE {
    return CONTROL_REFUSE, Refusal{Cause: cause, Token: position}
  }
  calls := op == OPCODE_CALL
  switch op {
  case OPCODE_RETURN:
    return CONTROL_RETURN, Refusal{Cause: FAILURE_NONE, Token: 0}
  case OPCODE_TAIL:
    frame.Place = Program_Place(production_entry(Production(operand)))
    return CONTROL_CONTINUE, Refusal{Cause: FAILURE_NONE, Token: 0}
  case OPCODE_CALL:
    frame.Callee = Production(operand)
    frame.Callee_Level = Callee_Level(PRECEDENCE_NONE)
  case OPCODE_CALL_HELD:
    calls = true
    frame.Callee = PRODUCTION_BINARY
    frame.Callee_Level = Callee_Level(frame.Held)
  }
  frame.Place = frame.Place + 1
  if !calls {
    return CONTROL_CONTINUE, Refusal{Cause: FAILURE_NONE, Token: 0}
  }
  // A grouped type opens no node, thus the tree depth alone never bounds the calls.
  if frame.Depth == FRAME_MAXIMUM {
    return CONTROL_REFUSE, Refusal{Cause: FAILURE_NESTING_DEPTH, Token: position}
  }
  return CONTROL_CALL, Refusal{Cause: FAILURE_NONE, Token: 0}
}

// Names the step a rule runs next: the one it stands at, or where a guard stands there, the step
// behind the guard or the one past it as the test the guard reads says. A guard only chooses, thus
// it runs inside the step it chooses rather than as one of its own.
func guarded(place Program_Place, holds Boolean) (next Program_Place) {
  defer func() { Program_Place_Invariants(next, "guarded.next") }()
  Program_Place_Invariants(place, "guarded.place")
  Boolean_Invariants(holds, "guarded.holds")
  next = place
  // No rule stacks more than a few guards, and the program closes with a step that is no
  // guard, thus the walk stops inside the program.
  for range STEP_COUNT {
    op := Opcode(PROGRAM[int(next)*STEP_SIZE])
    runs := bool(holds)
    if op == OPCODE_UNLESS {
      runs = !runs
    } else if op != OPCODE_WHEN {
      return next
    }
    next = next + 1
    if !runs {
      next = next + 1
    }
  }
  return next
}

// Reads the action and the operand of one step of the program.
func decode(place Program_Place) (op Opcode, operand Operand) {
  defer func() {
    Opcode_Invariants(op, "decode.op")
    Operand_Invariants(operand, "decode.operand")
  }()
  Program_Place_Invariants(place, "decode.place")
  offset := int(place) * STEP_SIZE
  return Opcode(PROGRAM[offset]), Operand(PROGRAM[offset+1])
}

// Names the step one rule opens at. Each entry spans two bytes of seven bits, because a byte of a
// constant string above that width would be written as two.
func production_entry(production Production) (place Entry_Place) {
  defer func() { Entry_Place_Invariants(place, "production_entry.place") }()
  Production_Invariants(production, "production_entry.production")
  offset := int(production) * STEP_SIZE
  high := int32(ENTRIES[offset])
  low := int32(ENTRIES[offset+1])
  return Entry_Place(high*ENTRY_BASE + low)
}

// Reads the token at the read, the token behind it, and the token behind that one. A read past
// the run names the end token, which is what the run holds there.
func window(
  tokens Token_Storage,
  run Run_Cursors,
) (at token.Kind, after token.Kind, second token.Kind) {
  defer func() {
    token.Kind_Invariants(at, "window.at")
    token.Kind_Invariants(after, "window.after")
    token.Kind_Invariants(second, "window.second")
  }()
  Token_Storage_Invariants(&tokens, "window.tokens")
  Run_Cursors_Invariants(&run, "window.run")
  position := int(run.Position)
  count := int(run.Count)
  at = tokens.Values[position].Kind
  after = token.KIND_END_OF_FILE
  if position+1 < count {
    after = tokens.Values[position+1].Kind
  }
  second = token.KIND_END_OF_FILE
  if position+2 < count {
    second = tokens.Values[position+2].Kind
  }
  return at, after, second
}

// Runs one test over the token at the read, the token behind it, and the readings the grammar
// carries, and reports whether it held. A step that tests nothing hands the last answer on.
func test(
  op Opcode,
  operand Operand,
  holds Boolean,
  level Precedence,
  at token.Kind,
  after token.Kind,
  asserted Type_Assertion,
) (result Boolean) {
  defer func() { Boolean_Invariants(result, "test.result") }()
  Opcode_Invariants(op, "test.op")
  Operand_Invariants(operand, "test.operand")
  Boolean_Invariants(holds, "test.holds")
  Precedence_Invariants(level, "test.level")
  token.Kind_Invariants(at, "test.at")
  token.Kind_Invariants(after, "test.after")
  Type_Assertion_Invariants(asserted, "test.asserted")
  kind := token.Kind(operand)
  opens := opens_type(at)
  follows := opens_type(after)
  binds := binary_precedence(at)
  clause := clause_kind(at)
  refused := refuses_sign(at)
  switch op {
  case OPCODE_AT, OPCODE_ACCEPT:
    return Boolean(at == kind)
  case OPCODE_OR_AT:
    if bool(holds) {
      return true
    }
    return Boolean(at == kind)
  case OPCODE_AFTER:
    return Boolean(after == kind)
  case OPCODE_OR_AFTER:
    if bool(holds) {
      return true
    }
    return Boolean(after == kind)
  case OPCODE_AND_AFTER:
    if !bool(holds) {
      return false
    }
    return Boolean(after == kind)
  case OPCODE_OPENS_TYPE:
    return opens
  case OPCODE_AFTER_OPENS_TYPE:
    return follows
  case OPCODE_ASSERTED:
    return Boolean(asserted)
  case OPCODE_REFUSES_SIGN:
    return refused
  case OPCODE_ABOVE:
    return Boolean(int(binds) > int(level))
  case OPCODE_CLAUSE_OPERATOR:
    return Boolean(int(clause) != CLAUSE_KIND_MINIMUM)
  }
  return holds
}

// Runs one test that reads further than the token behind the read: past a bracket, past a comma,
// or back to the node the open parent holds last. A step that tests nothing hands the last answer
// on.
func test_scan(
  op Opcode,
  holds Boolean,
  at token.Kind,
  after token.Kind,
  second token.Kind,
  closes Boolean,
  barred Boolean,
) (result Boolean) {
  defer func() { Boolean_Invariants(result, "test_scan.result") }()
  Opcode_Invariants(op, "test_scan.op")
  Boolean_Invariants(holds, "test_scan.holds")
  token.Kind_Invariants(at, "test_scan.at")
  token.Kind_Invariants(after, "test_scan.after")
  token.Kind_Invariants(second, "test_scan.second")
  Boolean_Invariants(closes, "test_scan.closes")
  Boolean_Invariants(barred, "test_scan.barred")
  follows := opens_type(after)
  later := opens_type(second)
  unary := unary_sign(at)
  suffixed := suffix_sign(at)
  switch op {
  case OPCODE_UNARY_SIGN:
    return unary
  case OPCODE_SUFFIX_SIGN:
    return suffixed
  case OPCODE_TYPE_PARAMETERS:
    // A comma behind the first name says a type parameter list, because an array
    // capacity holds one expression and never a list.
    if at != token.KIND_BRACKET_LEFT {
      return false
    }
    if after != token.KIND_IDENTIFIER {
      return false
    }
    if second == token.KIND_COMMA {
      return true
    }
    return later
  case OPCODE_QUALIFIES:
    switch second {
    case token.KIND_PERIOD, token.KIND_BRACKET_LEFT:
      return true
    }
    return false
  case OPCODE_FIELD_NAMED:
    if at != token.KIND_IDENTIFIER {
      return false
    }
    if after == token.KIND_COMMA {
      return true
    }
    if after != token.KIND_BRACKET_LEFT {
      return follows
    }
    return !closes
  case OPCODE_BARRED_BRACE:
    return barred
  }
  return holds
}

// Reports whether the bracket run behind the name at the read ends the field. An identifier ahead
// of a bracket reads two ways: a name before an array or slice type, or a generic type the field
// embeds. The token behind the closing bracket tells them apart, because a field that ends there
// embedded a type and one that reads on named it.
func closes_field(tokens Token_Storage, run Run_Cursors) (yes Boolean) {
  defer func() { Boolean_Invariants(yes, "closes_field.yes") }()
  Token_Storage_Invariants(&tokens, "closes_field.tokens")
  Run_Cursors_Invariants(&run, "closes_field.run")
  values := tokens.Values
  position := int(run.Position)
  count := int(run.Count)
  if values[position].Kind != token.KIND_IDENTIFIER {
    return false
  }
  if !(position+1 < count) {
    return false
  }
  if values[position+1].Kind != token.KIND_BRACKET_LEFT {
    return false
  }
  depth := 0
  for place := position + 1; place < count; place++ {
    switch values[place].Kind {
    case token.KIND_BRACKET_LEFT:
      depth++
    case token.KIND_BRACKET_RIGHT:
      depth--
      if depth == 0 {
        // A closing bracket always draws a token behind it, the end
        // token at the least, thus the read stays inside the run.
        switch values[place+1].Kind {
        case token.KIND_SEMICOLON, token.KIND_BRACE_RIGHT:
          return true
        case token.KIND_STRING, token.KIND_COMMENT:
          return true
        }
        return false
      }
    }
  }
  return false
}

// Reports whether a control clause bars the brace at the read from opening a composite literal. A
// clause bars a literal whose type is a bare name, because `for x {` opens a body and never a
// literal of type x, but `for range []T{...} {` reads only one way and Go admits it.
func barred_brace(
  nodes Node_Storage,
  last_children Last_Child_Storage,
  arena Arena_Cursors,
  plain Plain_Brace,
) (yes Boolean) {
  defer func() { Boolean_Invariants(yes, "barred_brace.yes") }()
  Node_Storage_Invariants(&nodes, "barred_brace.nodes")
  Last_Child_Storage_Invariants(&last_children, "barred_brace.last_children")
  Arena_Cursors_Invariants(&arena, "barred_brace.arena")
  Plain_Brace_Invariants(plain, "barred_brace.plain")
  if !bool(plain) {
    return false
  }
  depth := arena.Depth
  last := last_children.Values[depth-1]
  switch nodes.Values[last].Kind {
  case NODE_SLICE_TYPE, NODE_ARRAY_TYPE, NODE_MAP_TYPE, NODE_STRUCTURE_TYPE:
    return false
  }
  return true
}

// Keeps the level the operator at the read binds at, where the step asks for it.
func hold(op Opcode, held Held_Level, at token.Kind) (level Held_Level) {
  defer func() { Held_Level_Invariants(level, "hold.level") }()
  Opcode_Invariants(op, "hold.op")
  Held_Level_Invariants(held, "hold.held")
  token.Kind_Invariants(at, "hold.at")
  binds := binary_precedence(at)
  if op == OPCODE_HOLD {
    return Held_Level(binds)
  }
  return held
}

// Sets, clears, keeps, and restores the readings the grammar carries across its rules, and hands
// back the control clause reading the rule keeps.
func keep_plain(flags Machine_Flags_Handle, op Opcode, plain Plain_Brace) (kept Plain_Brace) {
  defer func() { Plain_Brace_Invariants(kept, "keep_plain.kept") }()
  Machine_Flags_Handle_Invariants(flags, "keep_plain.flags")
  Opcode_Invariants(op, "keep_plain.op")
  Plain_Brace_Invariants(plain, "keep_plain.plain")
  switch op {
  case OPCODE_SET_PLAIN:
    flags.Plain_Brace = true
  case OPCODE_CLEAR_PLAIN:
    flags.Plain_Brace = false
  case OPCODE_SAVE_PLAIN:
    return flags.Plain_Brace
  case OPCODE_RESTORE_PLAIN:
    flags.Plain_Brace = plain
  case OPCODE_SET_ASSERTION:
    flags.Type_Assertion = true
  case OPCODE_CLEAR_ASSERTION:
    flags.Type_Assertion = false
  }
  return plain
}

// Reads ahead to the bracket that closes the suffix at the read and names what it opens. A colon
// inside makes a slice and a comma makes an instantiation, so the class is known before the node
// is taken rather than written over it afterwards.
func bracket_kind(tokens Token_Storage, run Run_Cursors, op Opcode) (kind Suffix_Kind) {
  defer func() { Suffix_Kind_Invariants(kind, "bracket_kind.kind") }()
  Token_Storage_Invariants(&tokens, "bracket_kind.tokens")
  Run_Cursors_Invariants(&run, "bracket_kind.run")
  Opcode_Invariants(op, "bracket_kind.op")
  kind = Suffix_Kind(NODE_INDEX)
  if op != OPCODE_WRAP_SUFFIX {
    return kind
  }
  depth := 0
  count := int(run.Count)
  for place := int(run.Position); place < count; place++ {
    switch tokens.Values[place].Kind {
    case token.KIND_BRACKET_LEFT, token.KIND_PARENTHESIS_LEFT, token.KIND_BRACE_LEFT:
      depth++
    case token.KIND_PARENTHESIS_RIGHT, token.KIND_BRACE_RIGHT:
      depth--
    case token.KIND_BRACKET_RIGHT:
      depth--
      if depth == 0 {
        return kind
      }
    case token.KIND_COLON:
      if depth == 1 {
        kind = Suffix_Kind(NODE_SLICE_EXPRESSION)
      }
    case token.KIND_COMMA:
      if depth == 1 {
        kind = Suffix_Kind(NODE_GENERIC)
      }
    }
  }
  return kind
}

// Reports whether one step moves the read past the token it stands at. A name the shape refused
// stands where it stood, because the run stops there.
func steps_past(op Opcode, holds Boolean, at token.Kind) (yes Boolean) {
  defer func() { Boolean_Invariants(yes, "steps_past.yes") }()
  Opcode_Invariants(op, "steps_past.op")
  Boolean_Invariants(holds, "steps_past.holds")
  token.Kind_Invariants(at, "steps_past.at")
  switch op {
  case OPCODE_ADVANCE, OPCODE_TAKE, OPCODE_LITERAL:
    return true
  case OPCODE_ACCEPT:
    return holds
  case OPCODE_NAME:
    return Boolean(at == token.KIND_IDENTIFIER)
  }
  return false
}

// Moves the read through the source: past one token, past the trivia at the read, or past a run
// of empty statements. A run of trivia opens nodes, thus an exhausted arena stops the move.
func move(machine Machine_Handle, op Opcode, holds Boolean) (cause Arena_Failure) {
  defer func() { Arena_Failure_Invariants(cause, "move.cause") }()
  Machine_Handle_Invariants(machine, "move.machine")
  Opcode_Invariants(op, "move.op")
  Boolean_Invariants(holds, "move.holds")
  stepped := steps_past(op, holds, machine.Tokens.Values[machine.Run.Position].Kind)
  trivia := stepped
  if op == OPCODE_SKIP_COMMENTS {
    trivia = true
  }
  empty := Boolean(op == OPCODE_SKIP_EMPTY)
  if bool(empty) {
    trivia = true
  }
  // The run ends in one end-of-file token that the read never passes, thus a step that
  // misreads the grammar stalls there instead of running off.
  if stepped {
    if int(machine.Run.Position)+1 < int(machine.Run.Count) {
      machine.Run.Position = machine.Run.Position + 1
    }
  }
  if !trivia {
    return ARENA_INTACT
  }
  return take_trivia(machine, empty)
}

// Takes the trivia at the read into the open node: each comment, each run of empty lines, and
// where the step asks for it, each empty statement among them. The walk is a loop and never a
// descent, because a file may hold a run of comments or of empty statements as long as it likes.
func take_trivia(machine Machine_Handle, empty Boolean) (cause Arena_Failure) {
  defer func() { Arena_Failure_Invariants(cause, "take_trivia.cause") }()
  Machine_Handle_Invariants(machine, "take_trivia.machine")
  Boolean_Invariants(empty, "take_trivia.empty")
  tokens := machine.Tokens.Values
  for range TOKEN_COUNT_MAXIMUM {
    position := machine.Run.Position
    // The read only ever moves forward, thus one mark of how far the tree already holds
    // is enough to keep a run of empty lines out of the tree twice.
    if !(Token_Index(machine.Run.Blank) > Token_Index(position)) {
      machine.Run.Blank = Blank_Token(position + 1)
      if tokens[position].Blanks != 0 {
        span := Opened_Kind(NODE_BLANK)
        _, blank := open_node(&machine.Arena, machine.Run.Position, span)
        if blank != ARENA_INTACT {
          return blank
        }
        machine.Arena.Cursors.Depth = machine.Arena.Cursors.Depth - 1
      }
    }
    kind := tokens[position].Kind
    passed := kind == token.KIND_COMMENT
    if bool(empty) {
      if kind == token.KIND_SEMICOLON {
        passed = true
      }
    }
    if !passed {
      return ARENA_INTACT
    }
    if kind == token.KIND_COMMENT {
      note := Opened_Kind(NODE_COMMENT)
      _, comment := open_node(&machine.Arena, machine.Run.Position, note)
      if comment != ARENA_INTACT {
        return comment
      }
      machine.Arena.Cursors.Depth = machine.Arena.Cursors.Depth - 1
    }
    if int(position)+1 < int(machine.Run.Count) {
      machine.Run.Position = position + 1
    }
  }
  return ARENA_INTACT
}

// Takes each name of one comma-joined run as a node of the kind the operand names. A run of
// names is how a file spends its name storage, thus it costs one step rather than one rule for
// each name.
func take_names(machine Machine_Handle, op Opcode, operand Operand) (cause Name_Failure) {
  defer func() { Name_Failure_Invariants(cause, "take_names.cause") }()
  Machine_Handle_Invariants(machine, "take_names.machine")
  Opcode_Invariants(op, "take_names.op")
  Operand_Invariants(operand, "take_names.operand")
  if op != OPCODE_NAMES {
    return Name_Failure(FAILURE_NONE)
  }
  tokens := machine.Tokens.Values
  for range TOKEN_COUNT_MAXIMUM {
    kind := Opened_Kind(operand)
    if tokens[machine.Run.Position].Kind != token.KIND_IDENTIFIER {
      kind = Opened_Kind(NODE_ERROR)
    }
    _, failure := open_node(&machine.Arena, machine.Run.Position, kind)
    if failure != ARENA_INTACT {
      return Name_Failure(failure)
    }
    machine.Arena.Cursors.Depth = machine.Arena.Cursors.Depth - 1
    if kind == Opened_Kind(NODE_ERROR) {
      return Name_Failure(FAILURE_SYNTAX)
    }
    if int(machine.Run.Position)+1 < int(machine.Run.Count) {
      machine.Run.Position = machine.Run.Position + 1
    }
    passed := take_trivia(machine, false)
    if passed != ARENA_INTACT {
      return Name_Failure(passed)
    }
    if tokens[machine.Run.Position].Kind != token.KIND_COMMA {
      return Name_Failure(FAILURE_NONE)
    }
    if int(machine.Run.Position)+1 < int(machine.Run.Count) {
      machine.Run.Position = machine.Run.Position + 1
    }
    passed = take_trivia(machine, false)
    if passed != ARENA_INTACT {
      return Name_Failure(passed)
    }
  }
  return Name_Failure(FAILURE_NONE)
}

// Opens a pointer type at each star of one run and hands back the first node it opened, which the
// close of the run closes down to. A star the channel keyword stands behind takes that keyword
// too, as one prefix type does.
func open_stars(
  machine Machine_Handle,
  op Opcode,
  opened Opened_Index,
) (first Opened_Index, cause Arena_Failure) {
  defer func() {
    Opened_Index_Invariants(first, "open_stars.first")
    Arena_Failure_Invariants(cause, "open_stars.cause")
  }()
  Machine_Handle_Invariants(machine, "open_stars.machine")
  Opcode_Invariants(op, "open_stars.op")
  Opened_Index_Invariants(opened, "open_stars.opened")
  if op != OPCODE_STARS {
    return opened, ARENA_INTACT
  }
  tokens := machine.Tokens.Values
  first = Opened_Index(INDEX_ABSENT)
  for range TOKEN_COUNT_MAXIMUM {
    if tokens[machine.Run.Position].Kind != token.KIND_STAR {
      return first, ARENA_INTACT
    }
    pointer := Opened_Kind(NODE_POINTER_TYPE)
    index, failure := open_node(&machine.Arena, machine.Run.Position, pointer)
    if failure != ARENA_INTACT {
      return first, failure
    }
    if first == Opened_Index(INDEX_ABSENT) {
      first = index
    }
    if int(machine.Run.Position)+1 < int(machine.Run.Count) {
      machine.Run.Position = machine.Run.Position + 1
    }
    passed := take_trivia(machine, false)
    if passed != ARENA_INTACT {
      return first, passed
    }
    if tokens[machine.Run.Position].Kind == token.KIND_CHANNEL {
      if int(machine.Run.Position)+1 < int(machine.Run.Count) {
        machine.Run.Position = machine.Run.Position + 1
      }
      passed = take_trivia(machine, false)
      if passed != ARENA_INTACT {
        return first, passed
      }
    }
  }
  return first, ARENA_INTACT
}

// Opens, closes, writes, and retypes nodes as one step asks, and hands back the node the rule
// opened last. A refusal writes its error node here and names its own cause, because the first
// cause stands and an arena the error node overran is no reason the source was refused.
func shape(
  machine Machine_Handle,
  op Opcode,
  operand Operand,
  opened Opened_Index,
) (index Opened_Index, cause Failure_Code) {
  defer func() {
    Opened_Index_Invariants(index, "shape.index")
    Failure_Code_Invariants(cause, "shape.cause")
  }()
  Machine_Handle_Invariants(machine, "shape.machine")
  Opcode_Invariants(op, "shape.op")
  Operand_Invariants(operand, "shape.operand")
  Opened_Index_Invariants(opened, "shape.opened")
  at := machine.Tokens.Values[machine.Run.Position].Kind
  literal := literal_kind(at)
  clause := clause_kind(at)
  kind := Node_Kind(operand)
  switch op {
  case OPCODE_OPEN:
    node, failure := open_node(&machine.Arena, machine.Run.Position, Opened_Kind(kind))
    return node, Failure_Code(failure)
  case OPCODE_CLOSE:
    machine.Arena.Cursors.Depth = machine.Arena.Cursors.Depth - 1
    return opened, FAILURE_NONE
  case OPCODE_LEAF, OPCODE_TAKE, OPCODE_NAME, OPCODE_LITERAL, OPCODE_REJECT:
    if op == OPCODE_LITERAL {
      kind = Node_Kind(literal)
    }
    if op == OPCODE_NAME {
      if at != token.KIND_IDENTIFIER {
        op = OPCODE_REJECT
        operand = Operand(FAILURE_SYNTAX)
      }
    }
    if op == OPCODE_REJECT {
      kind = NODE_ERROR
    }
    leaf, failure := open_node(&machine.Arena, machine.Run.Position, Opened_Kind(kind))
    if failure == ARENA_INTACT {
      machine.Arena.Cursors.Depth = machine.Arena.Cursors.Depth - 1
    }
    if op == OPCODE_REJECT {
      return opened, Failure_Code(operand)
    }
    // A leaf is what a later retype of the rule would name, thus it
    // stands as the node the rule opened last.
    return leaf, Failure_Code(failure)
  case OPCODE_CLOSE_STARS:
    for range DEPTH_MAXIMUM {
      depth := machine.Arena.Cursors.Depth
      top := machine.Arena.Parents.Values[depth-1]
      machine.Arena.Cursors.Depth = depth - 1
      if Opened_Index(top) == opened {
        break
      }
    }
  case OPCODE_RETYPE:
    machine.Arena.Nodes.Values[opened].Kind = kind
  case OPCODE_RETYPE_CLAUSE:
    machine.Arena.Nodes.Values[opened].Kind = Node_Kind(clause)
  }
  return opened, FAILURE_NONE
}

// Takes back the final child of the open node into a node opened in its place, or reads every
// name of the open node back to an identifier. A binary operator adopts the operand parsed before
// the operator was seen, and this is the one place the tree runs backward.
func reshape(
  machine Machine_Handle,
  op Opcode,
  operand Operand,
  suffix Suffix_Kind,
) (cause Wrap_Failure) {
  defer func() { Wrap_Failure_Invariants(cause, "reshape.cause") }()
  Machine_Handle_Invariants(machine, "reshape.machine")
  Opcode_Invariants(op, "reshape.op")
  Operand_Invariants(operand, "reshape.operand")
  Suffix_Kind_Invariants(suffix, "reshape.suffix")
  nodes := machine.Arena.Nodes.Values
  depth := machine.Arena.Cursors.Depth
  if op == OPCODE_UNNAME {
    // A run of bare names with no type behind it was a run of types all along, and
    // only the token after the final name tells the two apart.
    child := Index(nodes[machine.Arena.Parents.Values[depth-1]].First_Child)
    for range TOKEN_COUNT_MAXIMUM {
      if child == INDEX_ABSENT {
        return WRAP_INTACT
      }
      if nodes[child].Kind == NODE_PARAMETER_NAME {
        nodes[child].Kind = NODE_IDENTIFIER
      }
      child = Index(nodes[child].Next)
    }
    return WRAP_INTACT
  }
  kind := Node_Kind(operand)
  if op == OPCODE_WRAP_SUFFIX {
    kind = Node_Kind(suffix)
  } else if op != OPCODE_WRAP {
    return WRAP_INTACT
  }
  parent := machine.Arena.Parents.Values[depth-1]
  last := machine.Arena.Last_Children.Values[depth-1]
  previous := machine.Arena.Earlier_Children.Values[depth-1]
  machine.Arena.Last_Children.Values[depth-1] = previous
  if previous == INDEX_ABSENT {
    nodes[parent].First_Child = Head(INDEX_ABSENT)
  } else {
    nodes[previous].Next = Successor(INDEX_ABSENT)
  }
  nodes[last].Parent = Ancestor(INDEX_ABSENT)
  nodes[last].Next = Successor(INDEX_ABSENT)
  _, failure := open_node(&machine.Arena, machine.Run.Position, Opened_Kind(kind))
  // The operand a wrap adopts opened at the depth the wrap opens at, thus a depth that
  // admitted the operand admits the wrap and only the arena can run out here.
  if failure != ARENA_INTACT {
    return Wrap_Failure(failure)
  }
  attach(&machine.Arena, Child(last))
  return WRAP_INTACT
}

// Takes one slot for a node of this kind at the read, links it under the open node, and opens it
// so later nodes attach to it. The arena and the parent stack are both bounded, thus an exhausted
// one stops the run rather than writing past its array. A node past the parent stack still takes
// its slot and its link, because it stands in the tree its parent holds and only its own children
// have no room.
func open_node(
  arena Arena_Handle,
  position Token_Position,
  kind Opened_Kind,
) (index Opened_Index, cause Arena_Failure) {
  defer func() {
    Opened_Index_Invariants(index, "open_node.index")
    Arena_Failure_Invariants(cause, "open_node.cause")
  }()
  Arena_Handle_Invariants(arena, "open_node.arena")
  Token_Position_Invariants(position, "open_node.position")
  Opened_Kind_Invariants(kind, "open_node.kind")
  count := arena.Cursors.Count
  if int(count) > INDEX_MAXIMUM {
    return Opened_Index(INDEX_ABSENT), Arena_Failure(FAILURE_NODE_COUNT)
  }
  arena.Nodes.Values[count] = Node{
    Kind:        Node_Kind(kind),
    Token:       Token_Index(position),
    Parent:      Ancestor(INDEX_ABSENT),
    First_Child: Head(INDEX_ABSENT),
    Next:        Successor(INDEX_ABSENT),
  }
  attach(arena, Child(count))
  arena.Cursors.Count = count + 1
  depth := arena.Cursors.Depth
  if !(int(depth) < DEPTH_MAXIMUM) {
    return Opened_Index(count), Arena_Failure(FAILURE_NESTING_DEPTH)
  }
  arena.Parents.Values[depth] = Index(count)
  arena.Last_Children.Values[depth] = INDEX_ABSENT
  arena.Earlier_Children.Values[depth] = INDEX_ABSENT
  arena.Cursors.Depth = depth + 1
  return Opened_Index(count), ARENA_INTACT
}

// Links one node under the open node as its final child.
func attach(arena Arena_Handle, child Child) {
  Arena_Handle_Invariants(arena, "attach.arena")
  Child_Invariants(child, "attach.child")
  depth := arena.Cursors.Depth
  nodes := arena.Nodes.Values
  parent := arena.Parents.Values[depth-1]
  previous := arena.Last_Children.Values[depth-1]
  nodes[child].Parent = Ancestor(parent)
  if previous == INDEX_ABSENT {
    nodes[parent].First_Child = Head(child)
  } else {
    nodes[previous].Next = Successor(child)
  }
  arena.Earlier_Children.Values[depth-1] = previous
  arena.Last_Children.Values[depth-1] = Index(child)
}

// Reports whether this kind can open a type, which is what tells a named parameter from a bare
// one and a named struct field from an embedded one.
func opens_type(kind token.Kind) (yes Boolean) {
  defer func() { Boolean_Invariants(yes, "opens_type.yes") }()
  token.Kind_Invariants(kind, "opens_type.kind")
  switch kind {
  case token.KIND_IDENTIFIER, token.KIND_STAR, token.KIND_BRACKET_LEFT, token.KIND_MAP,
    token.KIND_CHANNEL, token.KIND_FUNCTION, token.KIND_STRUCTURE, token.KIND_TILDE,
    token.KIND_INTERFACE, token.KIND_ARROW, token.KIND_ELLIPSIS,
    token.KIND_PARENTHESIS_LEFT:
    return true
  }
  return false
}

// Reports whether this kind is a sign a unary expression opens with.
func unary_sign(kind token.Kind) (yes Boolean) {
  defer func() { Boolean_Invariants(yes, "unary_sign.yes") }()
  token.Kind_Invariants(kind, "unary_sign.kind")
  switch kind {
  case token.KIND_PLUS, token.KIND_MINUS, token.KIND_NOT, token.KIND_EXCLUSIVE_OR,
    token.KIND_STAR, token.KIND_AND, token.KIND_ARROW:
    return true
  }
  return false
}

// Reports whether this kind opens a suffix over the operand already read.
func suffix_sign(kind token.Kind) (yes Boolean) {
  defer func() { Boolean_Invariants(yes, "suffix_sign.yes") }()
  token.Kind_Invariants(kind, "suffix_sign.kind")
  switch kind {
  case token.KIND_PERIOD, token.KIND_BRACKET_LEFT, token.KIND_PARENTHESIS_LEFT,
    token.KIND_BRACE_LEFT:
    return true
  }
  return false
}

// Reports whether this kind is a sign this dialect refuses: the two signs that join conditions,
// which are nested ifs written flat, and the two order signs that hold equality, which a strict
// comparison states.
func refuses_sign(kind token.Kind) (yes Boolean) {
  defer func() { Boolean_Invariants(yes, "refuses_sign.yes") }()
  token.Kind_Invariants(kind, "refuses_sign.kind")
  switch kind {
  case token.KIND_LOGICAL_AND, token.KIND_LOGICAL_OR, token.KIND_LESS_EQUAL,
    token.KIND_GREATER_EQUAL:
    return true
  }
  return false
}

// Reports the level at which this operator binds, or PRECEDENCE_NONE when it binds nothing.
func binary_precedence(kind token.Kind) (level Precedence) {
  defer func() { Precedence_Invariants(level, "binary_precedence.level") }()
  token.Kind_Invariants(kind, "binary_precedence.kind")
  switch kind {
  case token.KIND_EQUAL, token.KIND_NOT_EQUAL, token.KIND_LESS, token.KIND_GREATER:
    return PRECEDENCE_COMPARISON
  case token.KIND_PLUS, token.KIND_MINUS, token.KIND_OR, token.KIND_EXCLUSIVE_OR:
    return PRECEDENCE_ADDITION
  case token.KIND_STAR, token.KIND_SLASH, token.KIND_PERCENT, token.KIND_SHIFT_LEFT,
    token.KIND_SHIFT_RIGHT, token.KIND_AND, token.KIND_AND_NOT:
    return PRECEDENCE_MULTIPLICATION
  }
  return PRECEDENCE_NONE
}

// Names the statement form the operator after the left side opens.
func clause_kind(kind token.Kind) (result Clause_Kind) {
  defer func() { Clause_Kind_Invariants(result, "clause_kind.result") }()
  token.Kind_Invariants(kind, "clause_kind.kind")
  switch kind {
  case token.KIND_ASSIGN:
    return Clause_Kind(NODE_ASSIGN)
  case token.KIND_DEFINE:
    return Clause_Kind(NODE_DEFINE)
  case token.KIND_INCREMENT:
    return Clause_Kind(NODE_INCREMENT)
  case token.KIND_DECREMENT:
    return Clause_Kind(NODE_DECREMENT)
  case token.KIND_ARROW:
    return Clause_Kind(NODE_SEND)
  case token.KIND_PLUS_ASSIGN, token.KIND_MINUS_ASSIGN, token.KIND_STAR_ASSIGN,
    token.KIND_SLASH_ASSIGN, token.KIND_PERCENT_ASSIGN, token.KIND_AND_ASSIGN,
    token.KIND_OR_ASSIGN, token.KIND_EXCLUSIVE_OR_ASSIGN, token.KIND_SHIFT_LEFT_ASSIGN,
    token.KIND_SHIFT_RIGHT_ASSIGN, token.KIND_AND_NOT_ASSIGN:
    return Clause_Kind(NODE_OPERATION_ASSIGN)
  }
  return Clause_Kind(NODE_EXPRESSION_STATEMENT)
}

// Names the node kind a literal token opens. A token that is no literal names the integer kind,
// which no step asks of it.
func literal_kind(kind token.Kind) (literal Literal_Kind) {
  defer func() { Literal_Kind_Invariants(literal, "literal_kind.literal") }()
  token.Kind_Invariants(kind, "literal_kind.kind")
  switch kind {
  case token.KIND_FLOAT:
    return Literal_Kind(NODE_FLOAT)
  case token.KIND_IMAGINARY:
    return Literal_Kind(NODE_IMAGINARY)
  case token.KIND_CHARACTER:
    return Literal_Kind(NODE_CHARACTER)
  case token.KIND_STRING:
    return Literal_Kind(NODE_STRING)
  }
  return Literal_Kind(NODE_INTEGER)
}

// Failure_Message reads one failure code as a sentence that says what to do about it. Each
// sentence is a constant, thus naming a failure allocates nothing.
func Failure_Message(code Failure_Code) (text Message) {
  defer func() { Message_Invariants(text, "failure_message.text") }()
  Failure_Code_Invariants(code, "failure_message.code")
  switch code {
  case FAILURE_SYNTAX:
    return "Fix the syntax at this token."
  case FAILURE_INTERFACE_METHOD:
    return "Declare no interface; a generic constraint is the one exception."
  case FAILURE_WIDE_IDENTIFIER:
    return "Spell the name with ASCII letters, digits, and underscores."
  case FAILURE_IOTA:
    return "Spell the value out; iota ties it to declaration order."
  case FAILURE_REFUSED_SIGN:
    return "Write nested if statements."
  case FAILURE_DOT_IMPORT:
    return "Name the package; a dot import hides where a name came from."
  case FAILURE_BLANK_IMPORT:
    return "Call the package by name; a blank import hides an effect."
  case FAILURE_CONSTANT_CASE:
    return "Spell the constant name in SCREAMING_SNAKE_CASE."
  case FAILURE_BARE_LOOP:
    return "Give the loop a condition, a range, or a post clause."
  case FAILURE_UNNAMED_RESULT:
    return "Name every result the signature sends back."
  case FAILURE_TYPE_ALIAS:
    return "Declare a type of its own instead of an alias."
  case FAILURE_CLOSURE_BODY:
    return "Move closure logic into a named function; keep only calls."
  case FAILURE_PACKAGE_CLAUSE:
    return "Open the file with the package keyword and one name."
  case FAILURE_TOKEN_COUNT:
    return "Split the file; it holds more tokens than one parse admits."
  case FAILURE_NODE_COUNT:
    return "Split the file; it holds more syntax than one parse arena admits."
  case FAILURE_NESTING_DEPTH:
    return "Flatten the nesting; it runs deeper than one parse admits."
  case FAILURE_FALLTHROUGH:
    return "State each case on its own; a fallthrough hides the flow."
  }
  return "Read the tree."
}

// Names a refused byte at or above 128 for what it is. The scanner calls such a byte illegal,
// and only the source tells a stray byte apart from a name Go would have taken, thus the reading
// happens here where the source is still in reach.
func name_wide_identifier(
  subject Syntax_Tree_Handle,
  tree Parse_Result,
  source token.Source,
) (named Parse_Result) {
  defer func() { Parse_Result_Invariants(&named, "name_wide_identifier.named") }()
  Syntax_Tree_Handle_Invariants(subject, "name_wide_identifier.subject")
  Parse_Result_Invariants(&tree, "name_wide_identifier.tree")
  token.Source_Invariants(source, "name_wide_identifier.source")
  if tree.Failure != FAILURE_SYNTAX {
    return tree
  }
  one := subject.Tokens.Values[tree.Failure_Token]
  if one.Kind != token.KIND_ILLEGAL {
    return tree
  }
  if !(int(one.Offset) < len(source)) {
    return tree
  }
  if source[one.Offset] < WIDE_BYTE_MINIMUM {
    return tree
  }
  tree.Failure = FAILURE_WIDE_IDENTIFIER
  tree.OK = false
  return tree
}

// Names an import bound to the blank name. The name reads from the source, which no parse step
// carries, thus the pass runs here where the source is still in reach.
func name_blank_import(
  subject Syntax_Tree_Handle,
  tree Parse_Result,
  source token.Source,
) (named Parse_Result) {
  defer func() { Parse_Result_Invariants(&named, "name_blank_import.named") }()
  Syntax_Tree_Handle_Invariants(subject, "name_blank_import.subject")
  Parse_Result_Invariants(&tree, "name_blank_import.tree")
  token.Source_Invariants(source, "name_blank_import.source")
  if tree.Failure != FAILURE_NONE {
    return tree
  }
  count := tree.Node_Count
  for slot := INDEX_FIRST; slot < Index(count); slot++ {
    if subject.Nodes.Values[slot].Kind != NODE_IMPORT_NAME {
      continue
    }
    one := subject.Tokens.Values[subject.Nodes.Values[slot].Token]
    name := source[one.Offset : int(one.Offset)+int(one.Size)]
    if string(name) != BLANK_NAME {
      continue
    }
    tree.Failure = FAILURE_BLANK_IMPORT
    tree.OK = false
    tree.Failure_Token = Token_Index(subject.Nodes.Values[slot].Token)
    return tree
  }
  return tree
}

// Names a constant that is no run of uppercase words joined by single underscores. The name
// reads from the source, which no parse step carries, thus the pass runs here.
func name_constant_case(
  subject Syntax_Tree_Handle,
  tree Parse_Result,
  source token.Source,
) (named Parse_Result) {
  defer func() { Parse_Result_Invariants(&named, "name_constant_case.named") }()
  Syntax_Tree_Handle_Invariants(subject, "name_constant_case.subject")
  Parse_Result_Invariants(&tree, "name_constant_case.tree")
  token.Source_Invariants(source, "name_constant_case.source")
  if tree.Failure != FAILURE_NONE {
    return tree
  }
  count := tree.Node_Count
  for slot := INDEX_FIRST; slot < Index(count); slot++ {
    if subject.Nodes.Values[slot].Kind != NODE_CONSTANT_NAME {
      continue
    }
    one := subject.Tokens.Values[subject.Nodes.Values[slot].Token]
    name := source[one.Offset : int(one.Offset)+int(one.Size)]
    screams := true
    if name[0] < 'A' {
      screams = false
    }
    if name[0] > 'Z' {
      screams = false
    }
    // The byte test stays inline, because a step taking the name would owe every
    // source length at one call site and a constant name is never a whole file.
    for index := 1; index < len(name); index++ {
      value := name[index]
      if value == '_' {
        if index+1 == len(name) {
          screams = false
        }
        if name[index-1] == '_' {
          screams = false
        }
        continue
      }
      if !(value < 'A') {
        if !(value > 'Z') {
          continue
        }
      }
      if !(value < '0') {
        if !(value > '9') {
          continue
        }
      }
      screams = false
    }
    if screams {
      continue
    }
    tree.Failure = FAILURE_CONSTANT_CASE
    tree.OK = false
    tree.Failure_Token = Token_Index(subject.Nodes.Values[slot].Token)
    return tree
  }
  return tree
}

// Names a loop that no clause constrains: a bare for, a three-clause for with every clause
// empty, and a for whose only condition is the true literal. A source that means to run forever
// says so with a range, which is a form of its own and reads as the assertion it is.
func name_bare_loop(
  subject Syntax_Tree_Handle,
  tree Parse_Result,
  source token.Source,
) (named Parse_Result) {
  defer func() { Parse_Result_Invariants(&named, "name_bare_loop.named") }()
  Syntax_Tree_Handle_Invariants(subject, "name_bare_loop.subject")
  Parse_Result_Invariants(&tree, "name_bare_loop.tree")
  token.Source_Invariants(source, "name_bare_loop.source")
  if tree.Failure != FAILURE_NONE {
    return tree
  }
  count := tree.Node_Count
  for slot := INDEX_FIRST; slot < Index(count); slot++ {
    if subject.Nodes.Values[slot].Kind != NODE_FOR {
      continue
    }
    head := Index(subject.Nodes.Values[slot].First_Child)
    if head == INDEX_ABSENT {
      continue
    }
    bare := subject.Nodes.Values[head].Kind == NODE_BLOCK
    if subject.Nodes.Values[head].Kind == NODE_EXPRESSION_STATEMENT {
      inner := Index(subject.Nodes.Values[head].First_Child)
      if inner != INDEX_ABSENT {
        if subject.Nodes.Values[inner].Kind == NODE_IDENTIFIER {
          name_token := subject.Nodes.Values[inner].Token
          one := subject.Tokens.Values[name_token]
          name := source[one.Offset : int(one.Offset)+int(one.Size)]
          bare = string(name) == TRUE_NAME
        }
      }
    }
    if !bare {
      continue
    }
    tree.Failure = FAILURE_BARE_LOOP
    tree.OK = false
    tree.Failure_Token = Token_Index(subject.Nodes.Values[slot].Token)
    return tree
  }
  return tree
}

// Names a signature result that carries no name. A result that names itself holds its name and
// its type, thus a result holding one child alone is a type standing on its own.
func name_unnamed_result(subject Syntax_Tree_Handle, tree Parse_Result) (named Parse_Result) {
  defer func() { Parse_Result_Invariants(&named, "name_unnamed_result.named") }()
  Syntax_Tree_Handle_Invariants(subject, "name_unnamed_result.subject")
  Parse_Result_Invariants(&tree, "name_unnamed_result.tree")
  if tree.Failure != FAILURE_NONE {
    return tree
  }
  count := tree.Node_Count
  for slot := INDEX_FIRST; slot < Index(count); slot++ {
    if subject.Nodes.Values[slot].Kind != NODE_RESULT {
      continue
    }
    child := Index(subject.Nodes.Values[slot].First_Child)
    carries_name := false
    for range TOKEN_COUNT_MAXIMUM {
      if child == INDEX_ABSENT {
        break
      }
      if subject.Nodes.Values[child].Kind == NODE_PARAMETER_NAME {
        carries_name = true
        break
      }
      child = Index(subject.Nodes.Values[child].Next)
    }
    if carries_name {
      continue
    }
    tree.Failure = FAILURE_UNNAMED_RESULT
    tree.OK = false
    tree.Failure_Token = Token_Index(subject.Nodes.Values[slot].Token)
    return tree
  }
  return tree
}

// Names a statement of a function literal that is no call and no return, and a return that
// stands ahead of another statement, because a call behind a return never runs. Trivia states
// nothing the closure runs, thus a comment or an empty line between calls passes.
func name_closure_statement(subject Syntax_Tree_Handle, tree Parse_Result) (named Parse_Result) {
  defer func() { Parse_Result_Invariants(&named, "name_closure_statement.named") }()
  Syntax_Tree_Handle_Invariants(subject, "name_closure_statement.subject")
  Parse_Result_Invariants(&tree, "name_closure_statement.tree")
  if tree.Failure != FAILURE_NONE {
    return tree
  }
  nodes := subject.Nodes.Values
  count := tree.Node_Count
  for slot := INDEX_FIRST; slot < Index(count); slot++ {
    block := Index(nodes[slot].Parent)
    if nodes[block].Kind != NODE_BLOCK {
      continue
    }
    if nodes[nodes[block].Parent].Kind != NODE_FUNCTION_LITERAL {
      continue
    }
    kind := nodes[slot].Kind
    if kind == NODE_COMMENT {
      continue
    }
    if kind == NODE_BLANK {
      continue
    }
    refused := kind != NODE_EXPRESSION_STATEMENT
    if kind == NODE_RETURN {
      refused = false
      for sibling := Index(nodes[slot].Next); sibling != INDEX_ABSENT; {
        if nodes[sibling].Kind != NODE_COMMENT {
          if nodes[sibling].Kind != NODE_BLANK {
            refused = true
            break
          }
        }
        sibling = Index(nodes[sibling].Next)
      }
    }
    if !refused {
      continue
    }
    tree.Failure = FAILURE_CLOSURE_BODY
    tree.OK = false
    tree.Failure_Token = Token_Index(nodes[slot].Token)
    return tree
  }
  return tree
}

// Names a statement of a function literal, or a grouping inside one, that holds anything but one
// call or one grouping. A return of two values holds two children, thus it fails here too, and
// a grouping climbs to the statement it stands in, because only that statement says it is judged.
func name_closure_call(subject Syntax_Tree_Handle, tree Parse_Result) (named Parse_Result) {
  defer func() { Parse_Result_Invariants(&named, "name_closure_call.named") }()
  Syntax_Tree_Handle_Invariants(subject, "name_closure_call.subject")
  Parse_Result_Invariants(&tree, "name_closure_call.tree")
  if tree.Failure != FAILURE_NONE {
    return tree
  }
  nodes := subject.Nodes.Values
  count := tree.Node_Count
  for slot := INDEX_FIRST; slot < Index(count); slot++ {
    // The statement pass refuses every other statement kind, and trivia holds no
    // child, thus only these three stand to be judged.
    kind := nodes[slot].Kind
    judged := kind == NODE_EXPRESSION_STATEMENT
    if kind == NODE_RETURN {
      judged = true
    }
    if kind == NODE_PARENTHESIS {
      judged = true
    }
    if !judged {
      continue
    }
    statement := slot
    for nodes[statement].Kind == NODE_PARENTHESIS {
      statement = Index(nodes[statement].Parent)
    }
    block := Index(nodes[statement].Parent)
    if nodes[block].Kind != NODE_BLOCK {
      continue
    }
    if nodes[nodes[block].Parent].Kind != NODE_FUNCTION_LITERAL {
      continue
    }
    held := 0
    call := false
    for child := Index(nodes[slot].First_Child); child != INDEX_ABSENT; {
      child_kind := nodes[child].Kind
      if child_kind != NODE_COMMENT {
        if child_kind != NODE_BLANK {
          held++
          call = child_kind == NODE_CALL
          if child_kind == NODE_PARENTHESIS {
            call = true
          }
        }
      }
      child = Index(nodes[child].Next)
    }
    if held == 1 {
      if call {
        continue
      }
    }
    tree.Failure = FAILURE_CLOSURE_BODY
    tree.OK = false
    tree.Failure_Token = Token_Index(nodes[slot].Token)
    return tree
  }
  return tree
}

// Names the banned identifier a clean parse still holds. It reads the source, which no parse step
// carries, thus the pass runs here where the source is still in reach.
func name_banned_identifier(
  subject Syntax_Tree_Handle,
  tree Parse_Result,
  source token.Source,
) (named Parse_Result) {
  defer func() { Parse_Result_Invariants(&named, "name_banned_identifier.named") }()
  Syntax_Tree_Handle_Invariants(subject, "name_banned_identifier.subject")
  Parse_Result_Invariants(&tree, "name_banned_identifier.tree")
  token.Source_Invariants(source, "name_banned_identifier.source")
  if tree.Failure != FAILURE_NONE {
    return tree
  }
  count := tree.Node_Count
  for slot := INDEX_FIRST; slot < Index(count); slot++ {
    if subject.Nodes.Values[slot].Kind != NODE_IDENTIFIER {
      continue
    }
    one := subject.Tokens.Values[subject.Nodes.Values[slot].Token]
    constant := source[one.Offset : int(one.Offset)+int(one.Size)]
    if string(constant) == IOTA_NAME {
      tree.Failure = FAILURE_IOTA
      tree.OK = false
      tree.Failure_Token = Token_Index(
        subject.Nodes.Values[slot].Token,
      )
      return tree
    }
  }
  return tree
}

// ENTRY_BASE is one more than the widest seven-bit byte. A byte of a constant string above
// that width would be written as two, thus an entry spans two bytes of seven bits.
const ENTRY_BASE = 128

// NO_OPERAND pads a step whose action reads no operand, thus every step spans two bytes.
const NO_OPERAND = "\x00"

// RETURN is the step of OPCODE_RETURN.
const RETURN = string(rune(OPCODE_RETURN)) + NO_OPERAND

// CALL is the action of OPCODE_CALL; its operand follows it.
const CALL = string(rune(OPCODE_CALL))

// TAIL is the action of OPCODE_TAIL; its operand follows it.
const TAIL = string(rune(OPCODE_TAIL))

// WHEN is the step of OPCODE_WHEN.
const WHEN = string(rune(OPCODE_WHEN)) + NO_OPERAND

// UNLESS is the step of OPCODE_UNLESS.
const UNLESS = string(rune(OPCODE_UNLESS)) + NO_OPERAND

// AT is the action of OPCODE_AT; its operand follows it.
const AT = string(rune(OPCODE_AT))

// OR_AT is the action of OPCODE_OR_AT; its operand follows it.
const OR_AT = string(rune(OPCODE_OR_AT))

// AFTER is the action of OPCODE_AFTER; its operand follows it.
const AFTER = string(rune(OPCODE_AFTER))

// OR_AFTER is the action of OPCODE_OR_AFTER; its operand follows it.
const OR_AFTER = string(rune(OPCODE_OR_AFTER))

// AND_AFTER is the action of OPCODE_AND_AFTER; its operand follows it.
const AND_AFTER = string(rune(OPCODE_AND_AFTER))

// ACCEPT is the action of OPCODE_ACCEPT; its operand follows it.
const ACCEPT = string(rune(OPCODE_ACCEPT))

// OPENS_TYPE is the step of OPCODE_OPENS_TYPE.
const OPENS_TYPE = string(rune(OPCODE_OPENS_TYPE)) + NO_OPERAND

// AFTER_OPENS_TYPE is the step of OPCODE_AFTER_OPENS_TYPE.
const AFTER_OPENS_TYPE = string(rune(OPCODE_AFTER_OPENS_TYPE)) + NO_OPERAND

// ASSERTED is the step of OPCODE_ASSERTED.
const ASSERTED = string(rune(OPCODE_ASSERTED)) + NO_OPERAND

// REFUSES_SIGN is the step of OPCODE_REFUSES_SIGN.
const REFUSES_SIGN = string(rune(OPCODE_REFUSES_SIGN)) + NO_OPERAND

// ABOVE is the step of OPCODE_ABOVE.
const ABOVE = string(rune(OPCODE_ABOVE)) + NO_OPERAND

// CLAUSE_OPERATOR is the step of OPCODE_CLAUSE_OPERATOR.
const CLAUSE_OPERATOR = string(rune(OPCODE_CLAUSE_OPERATOR)) + NO_OPERAND

// TYPE_PARAMETERS is the step of OPCODE_TYPE_PARAMETERS.
const TYPE_PARAMETERS = string(rune(OPCODE_TYPE_PARAMETERS)) + NO_OPERAND

// QUALIFIES is the step of OPCODE_QUALIFIES.
const QUALIFIES = string(rune(OPCODE_QUALIFIES)) + NO_OPERAND

// FIELD_NAMED is the step of OPCODE_FIELD_NAMED.
const FIELD_NAMED = string(rune(OPCODE_FIELD_NAMED)) + NO_OPERAND

// BARRED_BRACE is the step of OPCODE_BARRED_BRACE.
const BARRED_BRACE = string(rune(OPCODE_BARRED_BRACE)) + NO_OPERAND

// ADVANCE is the step of OPCODE_ADVANCE.
const ADVANCE = string(rune(OPCODE_ADVANCE)) + NO_OPERAND

// SKIP_COMMENTS is the step of OPCODE_SKIP_COMMENTS.
const SKIP_COMMENTS = string(rune(OPCODE_SKIP_COMMENTS)) + NO_OPERAND

// OPEN is the action of OPCODE_OPEN; its operand follows it.
const OPEN = string(rune(OPCODE_OPEN))

// CLOSE is the step of OPCODE_CLOSE.
const CLOSE = string(rune(OPCODE_CLOSE)) + NO_OPERAND

// LEAF is the action of OPCODE_LEAF; its operand follows it.
const LEAF = string(rune(OPCODE_LEAF))

// LITERAL is the step of OPCODE_LITERAL.
const LITERAL = string(rune(OPCODE_LITERAL)) + NO_OPERAND

// WRAP is the action of OPCODE_WRAP; its operand follows it.
const WRAP = string(rune(OPCODE_WRAP))

// WRAP_SUFFIX is the step of OPCODE_WRAP_SUFFIX.
const WRAP_SUFFIX = string(rune(OPCODE_WRAP_SUFFIX)) + NO_OPERAND

// RETYPE is the action of OPCODE_RETYPE; its operand follows it.
const RETYPE = string(rune(OPCODE_RETYPE))

// RETYPE_CLAUSE is the step of OPCODE_RETYPE_CLAUSE.
const RETYPE_CLAUSE = string(rune(OPCODE_RETYPE_CLAUSE)) + NO_OPERAND

// UNNAME is the step of OPCODE_UNNAME.
const UNNAME = string(rune(OPCODE_UNNAME)) + NO_OPERAND

// REJECT is the action of OPCODE_REJECT; its operand follows it.
const REJECT = string(rune(OPCODE_REJECT))

// SET_PLAIN is the step of OPCODE_SET_PLAIN.
const SET_PLAIN = string(rune(OPCODE_SET_PLAIN)) + NO_OPERAND

// CLEAR_PLAIN is the step of OPCODE_CLEAR_PLAIN.
const CLEAR_PLAIN = string(rune(OPCODE_CLEAR_PLAIN)) + NO_OPERAND

// SAVE_PLAIN is the step of OPCODE_SAVE_PLAIN.
const SAVE_PLAIN = string(rune(OPCODE_SAVE_PLAIN)) + NO_OPERAND

// RESTORE_PLAIN is the step of OPCODE_RESTORE_PLAIN.
const RESTORE_PLAIN = string(rune(OPCODE_RESTORE_PLAIN)) + NO_OPERAND

// SET_ASSERTION is the step of OPCODE_SET_ASSERTION.
const SET_ASSERTION = string(rune(OPCODE_SET_ASSERTION)) + NO_OPERAND

// CLEAR_ASSERTION is the step of OPCODE_CLEAR_ASSERTION.
const CLEAR_ASSERTION = string(rune(OPCODE_CLEAR_ASSERTION)) + NO_OPERAND

// HOLD is the step of OPCODE_HOLD.
const HOLD = string(rune(OPCODE_HOLD)) + NO_OPERAND

// CALL_HELD is the step of OPCODE_CALL_HELD.
const CALL_HELD = string(rune(OPCODE_CALL_HELD)) + NO_OPERAND

// SKIP_EMPTY is the step of OPCODE_SKIP_EMPTY.
const SKIP_EMPTY = string(rune(OPCODE_SKIP_EMPTY)) + NO_OPERAND

// UNARY_SIGN is the step of OPCODE_UNARY_SIGN.
const UNARY_SIGN = string(rune(OPCODE_UNARY_SIGN)) + NO_OPERAND

// SUFFIX_SIGN is the step of OPCODE_SUFFIX_SIGN.
const SUFFIX_SIGN = string(rune(OPCODE_SUFFIX_SIGN)) + NO_OPERAND

// TAKE is the action of OPCODE_TAKE; its operand follows it.
const TAKE = string(rune(OPCODE_TAKE))

// NAME is the action of OPCODE_NAME; its operand follows it.
const NAME = string(rune(OPCODE_NAME))

// NAMES is the action of OPCODE_NAMES; its operand follows it.
const NAMES = string(rune(OPCODE_NAMES))

// STARS is the step of OPCODE_STARS.
const STARS = string(rune(OPCODE_STARS)) + NO_OPERAND

// CLOSE_STARS is the step of OPCODE_CLOSE_STARS.
const CLOSE_STARS = string(rune(OPCODE_CLOSE_STARS)) + NO_OPERAND

// PACKAGE_TOKEN names token.KIND_PACKAGE as an operand.
const PACKAGE_TOKEN = string(rune(token.KIND_PACKAGE))

// IDENTIFIER_TOKEN names token.KIND_IDENTIFIER as an operand.
const IDENTIFIER_TOKEN = string(rune(token.KIND_IDENTIFIER))

// END_OF_FILE_TOKEN names token.KIND_END_OF_FILE as an operand.
const END_OF_FILE_TOKEN = string(rune(token.KIND_END_OF_FILE))

// IMPORT_TOKEN names token.KIND_IMPORT as an operand.
const IMPORT_TOKEN = string(rune(token.KIND_IMPORT))

// CONSTANT_TOKEN names token.KIND_CONSTANT as an operand.
const CONSTANT_TOKEN = string(rune(token.KIND_CONSTANT))

// VARIABLE_TOKEN names token.KIND_VARIABLE as an operand.
const VARIABLE_TOKEN = string(rune(token.KIND_VARIABLE))

// TYPE_TOKEN names token.KIND_TYPE as an operand.
const TYPE_TOKEN = string(rune(token.KIND_TYPE))

// FUNCTION_TOKEN names token.KIND_FUNCTION as an operand.
const FUNCTION_TOKEN = string(rune(token.KIND_FUNCTION))

// SEMICOLON_TOKEN names token.KIND_SEMICOLON as an operand.
const SEMICOLON_TOKEN = string(rune(token.KIND_SEMICOLON))

// BRACE_RIGHT_TOKEN names token.KIND_BRACE_RIGHT as an operand.
const BRACE_RIGHT_TOKEN = string(rune(token.KIND_BRACE_RIGHT))

// PARENTHESIS_RIGHT_TOKEN names token.KIND_PARENTHESIS_RIGHT as an operand.
const PARENTHESIS_RIGHT_TOKEN = string(rune(token.KIND_PARENTHESIS_RIGHT))

// BRACKET_RIGHT_TOKEN names token.KIND_BRACKET_RIGHT as an operand.
const BRACKET_RIGHT_TOKEN = string(rune(token.KIND_BRACKET_RIGHT))

// CASE_TOKEN names token.KIND_CASE as an operand.
const CASE_TOKEN = string(rune(token.KIND_CASE))

// DEFAULT_TOKEN names token.KIND_DEFAULT as an operand.
const DEFAULT_TOKEN = string(rune(token.KIND_DEFAULT))

// COMMA_TOKEN names token.KIND_COMMA as an operand.
const COMMA_TOKEN = string(rune(token.KIND_COMMA))

// COLON_TOKEN names token.KIND_COLON as an operand.
const COLON_TOKEN = string(rune(token.KIND_COLON))

// ELSE_TOKEN names token.KIND_ELSE as an operand.
const ELSE_TOKEN = string(rune(token.KIND_ELSE))

// PARENTHESIS_LEFT_TOKEN names token.KIND_PARENTHESIS_LEFT as an operand.
const PARENTHESIS_LEFT_TOKEN = string(rune(token.KIND_PARENTHESIS_LEFT))

// PERIOD_TOKEN names token.KIND_PERIOD as an operand.
const PERIOD_TOKEN = string(rune(token.KIND_PERIOD))

// STRING_TOKEN names token.KIND_STRING as an operand.
const STRING_TOKEN = string(rune(token.KIND_STRING))

// ASSIGN_TOKEN names token.KIND_ASSIGN as an operand.
const ASSIGN_TOKEN = string(rune(token.KIND_ASSIGN))

// BRACKET_LEFT_TOKEN names token.KIND_BRACKET_LEFT as an operand.
const BRACKET_LEFT_TOKEN = string(rune(token.KIND_BRACKET_LEFT))

// OR_TOKEN names token.KIND_OR as an operand.
const OR_TOKEN = string(rune(token.KIND_OR))

// TILDE_TOKEN names token.KIND_TILDE as an operand.
const TILDE_TOKEN = string(rune(token.KIND_TILDE))

// BRACE_LEFT_TOKEN names token.KIND_BRACE_LEFT as an operand.
const BRACE_LEFT_TOKEN = string(rune(token.KIND_BRACE_LEFT))

// COMMENT_TOKEN names token.KIND_COMMENT as an operand.
const COMMENT_TOKEN = string(rune(token.KIND_COMMENT))

// STAR_TOKEN names token.KIND_STAR as an operand.
const STAR_TOKEN = string(rune(token.KIND_STAR))

// ELLIPSIS_TOKEN names token.KIND_ELLIPSIS as an operand.
const ELLIPSIS_TOKEN = string(rune(token.KIND_ELLIPSIS))

// MAP_TOKEN names token.KIND_MAP as an operand.
const MAP_TOKEN = string(rune(token.KIND_MAP))

// CHANNEL_TOKEN names token.KIND_CHANNEL as an operand.
const CHANNEL_TOKEN = string(rune(token.KIND_CHANNEL))

// ARROW_TOKEN names token.KIND_ARROW as an operand.
const ARROW_TOKEN = string(rune(token.KIND_ARROW))

// STRUCTURE_TOKEN names token.KIND_STRUCTURE as an operand.
const STRUCTURE_TOKEN = string(rune(token.KIND_STRUCTURE))

// INTERFACE_TOKEN names token.KIND_INTERFACE as an operand.
const INTERFACE_TOKEN = string(rune(token.KIND_INTERFACE))

// RETURN_TOKEN names token.KIND_RETURN as an operand.
const RETURN_TOKEN = string(rune(token.KIND_RETURN))

// IF_TOKEN names token.KIND_IF as an operand.
const IF_TOKEN = string(rune(token.KIND_IF))

// FOR_TOKEN names token.KIND_FOR as an operand.
const FOR_TOKEN = string(rune(token.KIND_FOR))

// SWITCH_TOKEN names token.KIND_SWITCH as an operand.
const SWITCH_TOKEN = string(rune(token.KIND_SWITCH))

// SELECT_TOKEN names token.KIND_SELECT as an operand.
const SELECT_TOKEN = string(rune(token.KIND_SELECT))

// GO_TOKEN names token.KIND_GO as an operand.
const GO_TOKEN = string(rune(token.KIND_GO))

// DEFER_TOKEN names token.KIND_DEFER as an operand.
const DEFER_TOKEN = string(rune(token.KIND_DEFER))

// BREAK_TOKEN names token.KIND_BREAK as an operand.
const BREAK_TOKEN = string(rune(token.KIND_BREAK))

// CONTINUE_TOKEN names token.KIND_CONTINUE as an operand.
const CONTINUE_TOKEN = string(rune(token.KIND_CONTINUE))

// GOTO_TOKEN names token.KIND_GOTO as an operand.
const GOTO_TOKEN = string(rune(token.KIND_GOTO))

// FALLTHROUGH_TOKEN names token.KIND_FALLTHROUGH as an operand.
const FALLTHROUGH_TOKEN = string(rune(token.KIND_FALLTHROUGH))

// RANGE_TOKEN names token.KIND_RANGE as an operand.
const RANGE_TOKEN = string(rune(token.KIND_RANGE))

// INCREMENT_TOKEN names token.KIND_INCREMENT as an operand.
const INCREMENT_TOKEN = string(rune(token.KIND_INCREMENT))

// DECREMENT_TOKEN names token.KIND_DECREMENT as an operand.
const DECREMENT_TOKEN = string(rune(token.KIND_DECREMENT))

// INTEGER_TOKEN names token.KIND_INTEGER as an operand.
const INTEGER_TOKEN = string(rune(token.KIND_INTEGER))

// FLOAT_TOKEN names token.KIND_FLOAT as an operand.
const FLOAT_TOKEN = string(rune(token.KIND_FLOAT))

// IMAGINARY_TOKEN names token.KIND_IMAGINARY as an operand.
const IMAGINARY_TOKEN = string(rune(token.KIND_IMAGINARY))

// CHARACTER_TOKEN names token.KIND_CHARACTER as an operand.
const CHARACTER_TOKEN = string(rune(token.KIND_CHARACTER))

// IDENTIFIER_NODE names NODE_IDENTIFIER as an operand.
const IDENTIFIER_NODE = string(rune(NODE_IDENTIFIER))

// IMPORT_NODE names NODE_IMPORT as an operand.
const IMPORT_NODE = string(rune(NODE_IMPORT))

// IMPORT_NAME_NODE names NODE_IMPORT_NAME as an operand.
const IMPORT_NAME_NODE = string(rune(NODE_IMPORT_NAME))

// STRING_NODE names NODE_STRING as an operand.
const STRING_NODE = string(rune(NODE_STRING))

// CONSTANT_NODE names NODE_CONSTANT as an operand.
const CONSTANT_NODE = string(rune(NODE_CONSTANT))

// CONSTANT_NAME_NODE names NODE_CONSTANT_NAME as an operand.
const CONSTANT_NAME_NODE = string(rune(NODE_CONSTANT_NAME))

// VARIABLE_NODE names NODE_VARIABLE as an operand.
const VARIABLE_NODE = string(rune(NODE_VARIABLE))

// TYPE_NODE names NODE_TYPE as an operand.
const TYPE_NODE = string(rune(NODE_TYPE))

// FUNCTION_NODE names NODE_FUNCTION as an operand.
const FUNCTION_NODE = string(rune(NODE_FUNCTION))

// RECEIVER_NODE names NODE_RECEIVER as an operand.
const RECEIVER_NODE = string(rune(NODE_RECEIVER))

// TYPE_PARAMETER_NODE names NODE_TYPE_PARAMETER as an operand.
const TYPE_PARAMETER_NODE = string(rune(NODE_TYPE_PARAMETER))

// UNION_NODE names NODE_UNION as an operand.
const UNION_NODE = string(rune(NODE_UNION))

// TERM_NODE names NODE_TERM as an operand.
const TERM_NODE = string(rune(NODE_TERM))

// RESULT_NODE names NODE_RESULT as an operand.
const RESULT_NODE = string(rune(NODE_RESULT))

// PARAMETER_NODE names NODE_PARAMETER as an operand.
const PARAMETER_NODE = string(rune(NODE_PARAMETER))

// PARAMETER_NAME_NODE names NODE_PARAMETER_NAME as an operand.
const PARAMETER_NAME_NODE = string(rune(NODE_PARAMETER_NAME))

// ELLIPSIS_NODE names NODE_ELLIPSIS as an operand.
const ELLIPSIS_NODE = string(rune(NODE_ELLIPSIS))

// CHANNEL_RECEIVE_NODE names NODE_CHANNEL_RECEIVE as an operand.
const CHANNEL_RECEIVE_NODE = string(rune(NODE_CHANNEL_RECEIVE))

// GENERIC_NODE names NODE_GENERIC as an operand.
const GENERIC_NODE = string(rune(NODE_GENERIC))

// SELECTOR_NODE names NODE_SELECTOR as an operand.
const SELECTOR_NODE = string(rune(NODE_SELECTOR))

// SLICE_TYPE_NODE names NODE_SLICE_TYPE as an operand.
const SLICE_TYPE_NODE = string(rune(NODE_SLICE_TYPE))

// ARRAY_TYPE_NODE names NODE_ARRAY_TYPE as an operand.
const ARRAY_TYPE_NODE = string(rune(NODE_ARRAY_TYPE))

// CHANNEL_TYPE_NODE names NODE_CHANNEL_TYPE as an operand.
const CHANNEL_TYPE_NODE = string(rune(NODE_CHANNEL_TYPE))

// CHANNEL_SEND_NODE names NODE_CHANNEL_SEND as an operand.
const CHANNEL_SEND_NODE = string(rune(NODE_CHANNEL_SEND))

// MAP_TYPE_NODE names NODE_MAP_TYPE as an operand.
const MAP_TYPE_NODE = string(rune(NODE_MAP_TYPE))

// FUNCTION_TYPE_NODE names NODE_FUNCTION_TYPE as an operand.
const FUNCTION_TYPE_NODE = string(rune(NODE_FUNCTION_TYPE))

// STRUCTURE_TYPE_NODE names NODE_STRUCTURE_TYPE as an operand.
const STRUCTURE_TYPE_NODE = string(rune(NODE_STRUCTURE_TYPE))

// FIELD_NODE names NODE_FIELD as an operand.
const FIELD_NODE = string(rune(NODE_FIELD))

// FIELD_NAME_NODE names NODE_FIELD_NAME as an operand.
const FIELD_NAME_NODE = string(rune(NODE_FIELD_NAME))

// INTERFACE_TYPE_NODE names NODE_INTERFACE_TYPE as an operand.
const INTERFACE_TYPE_NODE = string(rune(NODE_INTERFACE_TYPE))

// BLOCK_NODE names NODE_BLOCK as an operand.
const BLOCK_NODE = string(rune(NODE_BLOCK))

// GO_NODE names NODE_GO as an operand.
const GO_NODE = string(rune(NODE_GO))

// DEFER_NODE names NODE_DEFER as an operand.
const DEFER_NODE = string(rune(NODE_DEFER))

// BREAK_NODE names NODE_BREAK as an operand.
const BREAK_NODE = string(rune(NODE_BREAK))

// CONTINUE_NODE names NODE_CONTINUE as an operand.
const CONTINUE_NODE = string(rune(NODE_CONTINUE))

// GOTO_NODE names NODE_GOTO as an operand.
const GOTO_NODE = string(rune(NODE_GOTO))

// FALLTHROUGH_NODE names NODE_FALLTHROUGH as an operand.
const FALLTHROUGH_NODE = string(rune(NODE_FALLTHROUGH))

// DECLARATION_STATEMENT_NODE names NODE_DECLARATION_STATEMENT as an operand.
const DECLARATION_STATEMENT_NODE = string(rune(NODE_DECLARATION_STATEMENT))

// LABEL_NODE names NODE_LABEL as an operand.
const LABEL_NODE = string(rune(NODE_LABEL))

// RETURN_NODE names NODE_RETURN as an operand.
const RETURN_NODE = string(rune(NODE_RETURN))

// IF_NODE names NODE_IF as an operand.
const IF_NODE = string(rune(NODE_IF))

// FOR_NODE names NODE_FOR as an operand.
const FOR_NODE = string(rune(NODE_FOR))

// RANGE_NODE names NODE_RANGE as an operand.
const RANGE_NODE = string(rune(NODE_RANGE))

// SWITCH_NODE names NODE_SWITCH as an operand.
const SWITCH_NODE = string(rune(NODE_SWITCH))

// TYPE_SWITCH_NODE names NODE_TYPE_SWITCH as an operand.
const TYPE_SWITCH_NODE = string(rune(NODE_TYPE_SWITCH))

// SELECT_NODE names NODE_SELECT as an operand.
const SELECT_NODE = string(rune(NODE_SELECT))

// DEFAULT_NODE names NODE_DEFAULT as an operand.
const DEFAULT_NODE = string(rune(NODE_DEFAULT))

// CASE_NODE names NODE_CASE as an operand.
const CASE_NODE = string(rune(NODE_CASE))

// EXPRESSION_STATEMENT_NODE names NODE_EXPRESSION_STATEMENT as an operand.
const EXPRESSION_STATEMENT_NODE = string(rune(NODE_EXPRESSION_STATEMENT))

// BINARY_NODE names NODE_BINARY as an operand.
const BINARY_NODE = string(rune(NODE_BINARY))

// UNARY_NODE names NODE_UNARY as an operand.
const UNARY_NODE = string(rune(NODE_UNARY))

// PARENTHESIS_NODE names NODE_PARENTHESIS as an operand.
const PARENTHESIS_NODE = string(rune(NODE_PARENTHESIS))

// FUNCTION_LITERAL_NODE names NODE_FUNCTION_LITERAL as an operand.
const FUNCTION_LITERAL_NODE = string(rune(NODE_FUNCTION_LITERAL))

// ASSERTION_NODE names NODE_ASSERTION as an operand.
const ASSERTION_NODE = string(rune(NODE_ASSERTION))

// CALL_NODE names NODE_CALL as an operand.
const CALL_NODE = string(rune(NODE_CALL))

// COMPOSITE_NODE names NODE_COMPOSITE as an operand.
const COMPOSITE_NODE = string(rune(NODE_COMPOSITE))

// KEY_VALUE_NODE names NODE_KEY_VALUE as an operand.
const KEY_VALUE_NODE = string(rune(NODE_KEY_VALUE))

// SYNTAX_CAUSE names FAILURE_SYNTAX as an operand.
const SYNTAX_CAUSE = string(rune(FAILURE_SYNTAX))

// INTERFACE_METHOD_CAUSE names FAILURE_INTERFACE_METHOD as an operand.
const INTERFACE_METHOD_CAUSE = string(rune(FAILURE_INTERFACE_METHOD))

// REFUSED_SIGN_CAUSE names FAILURE_REFUSED_SIGN as an operand.
const REFUSED_SIGN_CAUSE = string(rune(FAILURE_REFUSED_SIGN))

// DOT_IMPORT_CAUSE names FAILURE_DOT_IMPORT as an operand.
const DOT_IMPORT_CAUSE = string(rune(FAILURE_DOT_IMPORT))

// TYPE_ALIAS_CAUSE names FAILURE_TYPE_ALIAS as an operand.
const TYPE_ALIAS_CAUSE = string(rune(FAILURE_TYPE_ALIAS))

// PACKAGE_CLAUSE_CAUSE names FAILURE_PACKAGE_CLAUSE as an operand.
const PACKAGE_CLAUSE_CAUSE = string(rune(FAILURE_PACKAGE_CLAUSE))

// PRODUCTION_FILE is the package clause and every declaration of one file.
const PRODUCTION_FILE Production = 0

// PRODUCTION_PACKAGE_CLAUSE is the package clause of one file.
const PRODUCTION_PACKAGE_CLAUSE Production = 1

// PRODUCTION_NAME is one identifier as a node of its own.
const PRODUCTION_NAME Production = 2

// PRODUCTION_DECLARATIONS is the run of declarations up to the end of the file.
const PRODUCTION_DECLARATIONS Production = 3

// PRODUCTION_DECLARATION is one top-level declaration.
const PRODUCTION_DECLARATION Production = 4

// PRODUCTION_END_LINE is the trivia and the semicolon that close one declaration or statement.
const PRODUCTION_END_LINE Production = 5

// PRODUCTION_IMPORT is one import declaration.
const PRODUCTION_IMPORT Production = 6

// PRODUCTION_IMPORT_PATH is one import path and its optional name.
const PRODUCTION_IMPORT_PATH Production = 7

// PRODUCTION_VALUE_DECLARATION is one const or var declaration.
const PRODUCTION_VALUE_DECLARATION Production = 8

// PRODUCTION_CONSTANT_DECLARATION is one const declaration.
const PRODUCTION_CONSTANT_DECLARATION Production = 9

// PRODUCTION_VARIABLE_DECLARATION is one var declaration.
const PRODUCTION_VARIABLE_DECLARATION Production = 10

// PRODUCTION_VALUE_TAIL is the type and the values behind the names of a value declaration.
const PRODUCTION_VALUE_TAIL Production = 11

// PRODUCTION_TYPE_DECLARATION is one type declaration.
const PRODUCTION_TYPE_DECLARATION Production = 12

// PRODUCTION_FUNCTION_DECLARATION is one function or method declaration.
const PRODUCTION_FUNCTION_DECLARATION Production = 13

// PRODUCTION_RECEIVER is the receiver of one method declaration.
const PRODUCTION_RECEIVER Production = 14

// PRODUCTION_TYPE_PARAMETER_LIST is one type parameter list.
const PRODUCTION_TYPE_PARAMETER_LIST Production = 15

// PRODUCTION_TYPE_PARAMETER_ITEMS is the type parameters of one list.
const PRODUCTION_TYPE_PARAMETER_ITEMS Production = 16

// PRODUCTION_TYPE_PARAMETER_NAMES is the names behind the first one a type parameter binds.
const PRODUCTION_TYPE_PARAMETER_NAMES Production = 17

// PRODUCTION_CONSTRAINT is one type constraint.
const PRODUCTION_CONSTRAINT Production = 18

// PRODUCTION_UNION_TERMS is the terms behind the first one a union holds.
const PRODUCTION_UNION_TERMS Production = 19

// PRODUCTION_TERM is one constraint term.
const PRODUCTION_TERM Production = 20

// PRODUCTION_SIGNATURE is one signature: its parameters and its results.
const PRODUCTION_SIGNATURE Production = 21

// PRODUCTION_PARAMETERS is one parenthesized parameter group.
const PRODUCTION_PARAMETERS Production = 22

// PRODUCTION_PARAMETER_ITEMS is the parameters of one group.
const PRODUCTION_PARAMETER_ITEMS Production = 23

// PRODUCTION_RESULTS is one parenthesized result group.
const PRODUCTION_RESULTS Production = 24

// PRODUCTION_RESULT_ITEMS is the results of one group.
const PRODUCTION_RESULT_ITEMS Production = 25

// PRODUCTION_PARAMETER is one parameter.
const PRODUCTION_PARAMETER Production = 26

// PRODUCTION_RESULT_PARAMETER is one result.
const PRODUCTION_RESULT_PARAMETER Production = 27

// PRODUCTION_PARAMETER_BODY is the names and the type of one parameter or result.
const PRODUCTION_PARAMETER_BODY Production = 28

// PRODUCTION_PARAMETER_NAMES is the names behind the first one a parameter binds.
const PRODUCTION_PARAMETER_NAMES Production = 29

// PRODUCTION_PARAMETER_TYPE is the type one parameter or result closes with.
const PRODUCTION_PARAMETER_TYPE Production = 30

// PRODUCTION_PARAMETER_NAME is one name a signature binds.
const PRODUCTION_PARAMETER_NAME Production = 31

// PRODUCTION_TYPE is one type.
const PRODUCTION_TYPE Production = 32

// PRODUCTION_POINTERS is one run of pointer types and the type they point at.
const PRODUCTION_POINTERS Production = 33

// PRODUCTION_PREFIX_TYPE is the type behind a pointer, an ellipsis, or a receive arrow.
const PRODUCTION_PREFIX_TYPE Production = 34

// PRODUCTION_GROUPED_TYPE is one parenthesized type.
const PRODUCTION_GROUPED_TYPE Production = 35

// PRODUCTION_TYPE_NAME is one named type, its qualifier, and its type arguments.
const PRODUCTION_TYPE_NAME Production = 36

// PRODUCTION_QUALIFIED_NAME is the name behind the qualifier of one named type.
const PRODUCTION_QUALIFIED_NAME Production = 37

// PRODUCTION_GENERIC_ARGUMENTS is the type arguments of one generic instance.
const PRODUCTION_GENERIC_ARGUMENTS Production = 38

// PRODUCTION_ARRAY_TYPE is one array or slice type.
const PRODUCTION_ARRAY_TYPE Production = 39

// PRODUCTION_ELEMENT_TYPE is the type the element of one composite type holds.
const PRODUCTION_ELEMENT_TYPE Production = 40

// PRODUCTION_CHANNEL_TYPE is one channel type.
const PRODUCTION_CHANNEL_TYPE Production = 41

// PRODUCTION_MAP_TYPE is one map type.
const PRODUCTION_MAP_TYPE Production = 42

// PRODUCTION_FUNCTION_TYPE is one function type.
const PRODUCTION_FUNCTION_TYPE Production = 43

// PRODUCTION_STRUCTURE_TYPE is one struct type.
const PRODUCTION_STRUCTURE_TYPE Production = 44

// PRODUCTION_FIELDS is the fields of one struct type.
const PRODUCTION_FIELDS Production = 45

// PRODUCTION_FIELD is one struct field.
const PRODUCTION_FIELD Production = 46

// PRODUCTION_FIELD_TYPE is the type and the tag one field closes with.
const PRODUCTION_FIELD_TYPE Production = 47

// PRODUCTION_INTERFACE_TYPE is one interface type.
const PRODUCTION_INTERFACE_TYPE Production = 48

// PRODUCTION_ELEMENTS is the elements of one interface type.
const PRODUCTION_ELEMENTS Production = 49

// PRODUCTION_BLOCK is one brace-delimited statement list.
const PRODUCTION_BLOCK Production = 50

// PRODUCTION_STATEMENTS is the statements of one block.
const PRODUCTION_STATEMENTS Production = 51

// PRODUCTION_STATEMENT is one statement.
const PRODUCTION_STATEMENT Production = 52

// PRODUCTION_DECLARATION_STATEMENT is one declaration standing inside a body.
const PRODUCTION_DECLARATION_STATEMENT Production = 53

// PRODUCTION_LABEL is one labelled statement.
const PRODUCTION_LABEL Production = 54

// PRODUCTION_RETURN_STATEMENT is one return statement.
const PRODUCTION_RETURN_STATEMENT Production = 55

// PRODUCTION_LAUNCH is the call one go or defer statement carries.
const PRODUCTION_LAUNCH Production = 56

// PRODUCTION_JUMP is the label one break, continue, or goto statement carries.
const PRODUCTION_JUMP Production = 57

// PRODUCTION_IF_STATEMENT is one if statement.
const PRODUCTION_IF_STATEMENT Production = 58

// PRODUCTION_IF_END is the close of one if statement.
const PRODUCTION_IF_END Production = 59

// PRODUCTION_FOR_STATEMENT is one for statement.
const PRODUCTION_FOR_STATEMENT Production = 60

// PRODUCTION_FOR_HEAD is the head of one for statement.
const PRODUCTION_FOR_HEAD Production = 61

// PRODUCTION_RANGE_CLAUSE is one range clause.
const PRODUCTION_RANGE_CLAUSE Production = 62

// PRODUCTION_SWITCH_STATEMENT is one expression or type switch.
const PRODUCTION_SWITCH_STATEMENT Production = 63

// PRODUCTION_SWITCH_HEAD is the head of one switch.
const PRODUCTION_SWITCH_HEAD Production = 64

// PRODUCTION_SELECT_STATEMENT is one select statement.
const PRODUCTION_SELECT_STATEMENT Production = 65

// PRODUCTION_CASE_BLOCK is the clause list of one switch or select.
const PRODUCTION_CASE_BLOCK Production = 66

// PRODUCTION_CASES is the clauses of one clause list.
const PRODUCTION_CASES Production = 67

// PRODUCTION_CASE is one case or default clause.
const PRODUCTION_CASE Production = 68

// PRODUCTION_CASE_COLON is the colon and the statements of one clause.
const PRODUCTION_CASE_COLON Production = 69

// PRODUCTION_CASE_BODY is the statements of one clause.
const PRODUCTION_CASE_BODY Production = 70

// PRODUCTION_CLAUSE is one simple statement without its closing semicolon.
const PRODUCTION_CLAUSE Production = 71

// PRODUCTION_EXPRESSION_LIST is one comma-separated expression list.
const PRODUCTION_EXPRESSION_LIST Production = 72

// PRODUCTION_BINARY is one expression whose operators bind tighter than the level the rule runs at.
const PRODUCTION_BINARY Production = 73

// PRODUCTION_BINARY_OPERATORS is the operators behind the first operand of one binary expression.
const PRODUCTION_BINARY_OPERATORS Production = 74

// PRODUCTION_UNARY is one unary expression.
const PRODUCTION_UNARY Production = 75

// PRODUCTION_PRIMARY is one operand and every suffix that binds to it.
const PRODUCTION_PRIMARY Production = 76

// PRODUCTION_SUFFIXES is the suffixes that bind to one operand.
const PRODUCTION_SUFFIXES Production = 77

// PRODUCTION_OPERAND is one operand.
const PRODUCTION_OPERAND Production = 78

// PRODUCTION_GROUPING is one parenthesized expression.
const PRODUCTION_GROUPING Production = 79

// PRODUCTION_FUNCTION_LITERAL is one function literal.
const PRODUCTION_FUNCTION_LITERAL Production = 80

// PRODUCTION_SELECTOR_SUFFIX is one selector over the expression already read.
const PRODUCTION_SELECTOR_SUFFIX Production = 81

// PRODUCTION_ASSERTION_SUFFIX is one type assertion over the expression already read.
const PRODUCTION_ASSERTION_SUFFIX Production = 82

// PRODUCTION_INDEX_SUFFIX is one index, slice, or instantiation over the expression already read.
const PRODUCTION_INDEX_SUFFIX Production = 83

// PRODUCTION_INDEX_ITEMS is the bounds of one bracket suffix.
const PRODUCTION_INDEX_ITEMS Production = 84

// PRODUCTION_CALL_SUFFIX is one call over the expression already read.
const PRODUCTION_CALL_SUFFIX Production = 85

// PRODUCTION_ARGUMENTS is the arguments of one call.
const PRODUCTION_ARGUMENTS Production = 86

// PRODUCTION_COMPOSITE_SUFFIX is one composite literal over the type already read.
const PRODUCTION_COMPOSITE_SUFFIX Production = 87

// PRODUCTION_COMPOSITE_BODY is the element list of one composite literal.
const PRODUCTION_COMPOSITE_BODY Production = 88

// PRODUCTION_COMPOSITE_ITEMS is the elements of one composite literal.
const PRODUCTION_COMPOSITE_ITEMS Production = 89

// PRODUCTION_ELEMENT is one element of a composite literal.
const PRODUCTION_ELEMENT Production = 90

// PRODUCTION_ELEMENT_VALUE is one element value.
const PRODUCTION_ELEMENT_VALUE Production = 91

// PRODUCTION_CONSTANT_GROUP is the parenthesized run of constants.
const PRODUCTION_CONSTANT_GROUP Production = 92

// PRODUCTION_CONSTANT_MEMBERS is one constant of that run and the run behind it.
const PRODUCTION_CONSTANT_MEMBERS Production = 93

// PRODUCTION_VARIABLE_GROUP is the parenthesized run of variables.
const PRODUCTION_VARIABLE_GROUP Production = 94

// PRODUCTION_VARIABLE_MEMBERS is one variable of that run and the run behind it.
const PRODUCTION_VARIABLE_MEMBERS Production = 95

// PRODUCTION_TYPE_GROUP is the parenthesized run of types.
const PRODUCTION_TYPE_GROUP Production = 96

// PRODUCTION_TYPE_MEMBERS is one type of that run and the run behind it.
const PRODUCTION_TYPE_MEMBERS Production = 97

// PRODUCTION_IMPORT_GROUP is the parenthesized run of imports.
const PRODUCTION_IMPORT_GROUP Production = 98

// PRODUCTION_IMPORT_MEMBERS is one import of that run and the run behind it.
const PRODUCTION_IMPORT_MEMBERS Production = 99

// PRODUCTION_GROUP_VALUE_TAIL is the type and the values one grouped member states. Every run
// calls it, thus it is the rule a call reaches last.
const PRODUCTION_GROUP_VALUE_TAIL Production = 100

// PACKAGE_CLAUSE_PRODUCTION names PRODUCTION_PACKAGE_CLAUSE as an operand.
const PACKAGE_CLAUSE_PRODUCTION = string(rune(PRODUCTION_PACKAGE_CLAUSE))

// NAME_PRODUCTION names PRODUCTION_NAME as an operand.
const NAME_PRODUCTION = string(rune(PRODUCTION_NAME))

// DECLARATIONS_PRODUCTION names PRODUCTION_DECLARATIONS as an operand.
const DECLARATIONS_PRODUCTION = string(rune(PRODUCTION_DECLARATIONS))

// DECLARATION_PRODUCTION names PRODUCTION_DECLARATION as an operand.
const DECLARATION_PRODUCTION = string(rune(PRODUCTION_DECLARATION))

// END_LINE_PRODUCTION names PRODUCTION_END_LINE as an operand.
const END_LINE_PRODUCTION = string(rune(PRODUCTION_END_LINE))

// IMPORT_PRODUCTION names PRODUCTION_IMPORT as an operand.
const IMPORT_PRODUCTION = string(rune(PRODUCTION_IMPORT))

// IMPORT_PATH_PRODUCTION names PRODUCTION_IMPORT_PATH as an operand.
const IMPORT_PATH_PRODUCTION = string(rune(PRODUCTION_IMPORT_PATH))

// VALUE_DECLARATION_PRODUCTION names PRODUCTION_VALUE_DECLARATION as an operand.
const VALUE_DECLARATION_PRODUCTION = string(rune(PRODUCTION_VALUE_DECLARATION))

// CONSTANT_DECLARATION_PRODUCTION names PRODUCTION_CONSTANT_DECLARATION as an operand.
const CONSTANT_DECLARATION_PRODUCTION = string(rune(PRODUCTION_CONSTANT_DECLARATION))

// CONSTANT_GROUP_PRODUCTION names PRODUCTION_CONSTANT_GROUP as an operand.
const CONSTANT_GROUP_PRODUCTION = string(rune(PRODUCTION_CONSTANT_GROUP))

// CONSTANT_MEMBERS_PRODUCTION names PRODUCTION_CONSTANT_MEMBERS as an operand.
const CONSTANT_MEMBERS_PRODUCTION = string(rune(PRODUCTION_CONSTANT_MEMBERS))

// GROUP_VALUE_TAIL_PRODUCTION names PRODUCTION_GROUP_VALUE_TAIL as an operand.
const GROUP_VALUE_TAIL_PRODUCTION = string(rune(PRODUCTION_GROUP_VALUE_TAIL))

// VARIABLE_GROUP_PRODUCTION names PRODUCTION_VARIABLE_GROUP as an operand.
const VARIABLE_GROUP_PRODUCTION = string(rune(PRODUCTION_VARIABLE_GROUP))

// VARIABLE_MEMBERS_PRODUCTION names PRODUCTION_VARIABLE_MEMBERS as an operand.
const VARIABLE_MEMBERS_PRODUCTION = string(rune(PRODUCTION_VARIABLE_MEMBERS))

// TYPE_GROUP_PRODUCTION names PRODUCTION_TYPE_GROUP as an operand.
const TYPE_GROUP_PRODUCTION = string(rune(PRODUCTION_TYPE_GROUP))

// TYPE_MEMBERS_PRODUCTION names PRODUCTION_TYPE_MEMBERS as an operand.
const TYPE_MEMBERS_PRODUCTION = string(rune(PRODUCTION_TYPE_MEMBERS))

// IMPORT_GROUP_PRODUCTION names PRODUCTION_IMPORT_GROUP as an operand.
const IMPORT_GROUP_PRODUCTION = string(rune(PRODUCTION_IMPORT_GROUP))

// IMPORT_MEMBERS_PRODUCTION names PRODUCTION_IMPORT_MEMBERS as an operand.
const IMPORT_MEMBERS_PRODUCTION = string(rune(PRODUCTION_IMPORT_MEMBERS))

// VARIABLE_DECLARATION_PRODUCTION names PRODUCTION_VARIABLE_DECLARATION as an operand.
const VARIABLE_DECLARATION_PRODUCTION = string(rune(PRODUCTION_VARIABLE_DECLARATION))

// VALUE_TAIL_PRODUCTION names PRODUCTION_VALUE_TAIL as an operand.
const VALUE_TAIL_PRODUCTION = string(rune(PRODUCTION_VALUE_TAIL))

// TYPE_DECLARATION_PRODUCTION names PRODUCTION_TYPE_DECLARATION as an operand.
const TYPE_DECLARATION_PRODUCTION = string(rune(PRODUCTION_TYPE_DECLARATION))

// FUNCTION_DECLARATION_PRODUCTION names PRODUCTION_FUNCTION_DECLARATION as an operand.
const FUNCTION_DECLARATION_PRODUCTION = string(rune(PRODUCTION_FUNCTION_DECLARATION))

// RECEIVER_PRODUCTION names PRODUCTION_RECEIVER as an operand.
const RECEIVER_PRODUCTION = string(rune(PRODUCTION_RECEIVER))

// TYPE_PARAMETER_LIST_PRODUCTION names PRODUCTION_TYPE_PARAMETER_LIST as an operand.
const TYPE_PARAMETER_LIST_PRODUCTION = string(rune(PRODUCTION_TYPE_PARAMETER_LIST))

// TYPE_PARAMETER_ITEMS_PRODUCTION names PRODUCTION_TYPE_PARAMETER_ITEMS as an operand.
const TYPE_PARAMETER_ITEMS_PRODUCTION = string(rune(PRODUCTION_TYPE_PARAMETER_ITEMS))

// TYPE_PARAMETER_NAMES_PRODUCTION names PRODUCTION_TYPE_PARAMETER_NAMES as an operand.
const TYPE_PARAMETER_NAMES_PRODUCTION = string(rune(PRODUCTION_TYPE_PARAMETER_NAMES))

// CONSTRAINT_PRODUCTION names PRODUCTION_CONSTRAINT as an operand.
const CONSTRAINT_PRODUCTION = string(rune(PRODUCTION_CONSTRAINT))

// UNION_TERMS_PRODUCTION names PRODUCTION_UNION_TERMS as an operand.
const UNION_TERMS_PRODUCTION = string(rune(PRODUCTION_UNION_TERMS))

// TERM_PRODUCTION names PRODUCTION_TERM as an operand.
const TERM_PRODUCTION = string(rune(PRODUCTION_TERM))

// SIGNATURE_PRODUCTION names PRODUCTION_SIGNATURE as an operand.
const SIGNATURE_PRODUCTION = string(rune(PRODUCTION_SIGNATURE))

// PARAMETERS_PRODUCTION names PRODUCTION_PARAMETERS as an operand.
const PARAMETERS_PRODUCTION = string(rune(PRODUCTION_PARAMETERS))

// PARAMETER_ITEMS_PRODUCTION names PRODUCTION_PARAMETER_ITEMS as an operand.
const PARAMETER_ITEMS_PRODUCTION = string(rune(PRODUCTION_PARAMETER_ITEMS))

// RESULTS_PRODUCTION names PRODUCTION_RESULTS as an operand.
const RESULTS_PRODUCTION = string(rune(PRODUCTION_RESULTS))

// RESULT_ITEMS_PRODUCTION names PRODUCTION_RESULT_ITEMS as an operand.
const RESULT_ITEMS_PRODUCTION = string(rune(PRODUCTION_RESULT_ITEMS))

// PARAMETER_PRODUCTION names PRODUCTION_PARAMETER as an operand.
const PARAMETER_PRODUCTION = string(rune(PRODUCTION_PARAMETER))

// RESULT_PARAMETER_PRODUCTION names PRODUCTION_RESULT_PARAMETER as an operand.
const RESULT_PARAMETER_PRODUCTION = string(rune(PRODUCTION_RESULT_PARAMETER))

// PARAMETER_BODY_PRODUCTION names PRODUCTION_PARAMETER_BODY as an operand.
const PARAMETER_BODY_PRODUCTION = string(rune(PRODUCTION_PARAMETER_BODY))

// PARAMETER_NAMES_PRODUCTION names PRODUCTION_PARAMETER_NAMES as an operand.
const PARAMETER_NAMES_PRODUCTION = string(rune(PRODUCTION_PARAMETER_NAMES))

// PARAMETER_TYPE_PRODUCTION names PRODUCTION_PARAMETER_TYPE as an operand.
const PARAMETER_TYPE_PRODUCTION = string(rune(PRODUCTION_PARAMETER_TYPE))

// PARAMETER_NAME_PRODUCTION names PRODUCTION_PARAMETER_NAME as an operand.
const PARAMETER_NAME_PRODUCTION = string(rune(PRODUCTION_PARAMETER_NAME))

// TYPE_PRODUCTION names PRODUCTION_TYPE as an operand.
const TYPE_PRODUCTION = string(rune(PRODUCTION_TYPE))

// POINTERS_PRODUCTION names PRODUCTION_POINTERS as an operand.
const POINTERS_PRODUCTION = string(rune(PRODUCTION_POINTERS))

// PREFIX_TYPE_PRODUCTION names PRODUCTION_PREFIX_TYPE as an operand.
const PREFIX_TYPE_PRODUCTION = string(rune(PRODUCTION_PREFIX_TYPE))

// GROUPED_TYPE_PRODUCTION names PRODUCTION_GROUPED_TYPE as an operand.
const GROUPED_TYPE_PRODUCTION = string(rune(PRODUCTION_GROUPED_TYPE))

// TYPE_NAME_PRODUCTION names PRODUCTION_TYPE_NAME as an operand.
const TYPE_NAME_PRODUCTION = string(rune(PRODUCTION_TYPE_NAME))

// QUALIFIED_NAME_PRODUCTION names PRODUCTION_QUALIFIED_NAME as an operand.
const QUALIFIED_NAME_PRODUCTION = string(rune(PRODUCTION_QUALIFIED_NAME))

// GENERIC_ARGUMENTS_PRODUCTION names PRODUCTION_GENERIC_ARGUMENTS as an operand.
const GENERIC_ARGUMENTS_PRODUCTION = string(rune(PRODUCTION_GENERIC_ARGUMENTS))

// ARRAY_TYPE_PRODUCTION names PRODUCTION_ARRAY_TYPE as an operand.
const ARRAY_TYPE_PRODUCTION = string(rune(PRODUCTION_ARRAY_TYPE))

// ELEMENT_TYPE_PRODUCTION names PRODUCTION_ELEMENT_TYPE as an operand.
const ELEMENT_TYPE_PRODUCTION = string(rune(PRODUCTION_ELEMENT_TYPE))

// CHANNEL_TYPE_PRODUCTION names PRODUCTION_CHANNEL_TYPE as an operand.
const CHANNEL_TYPE_PRODUCTION = string(rune(PRODUCTION_CHANNEL_TYPE))

// MAP_TYPE_PRODUCTION names PRODUCTION_MAP_TYPE as an operand.
const MAP_TYPE_PRODUCTION = string(rune(PRODUCTION_MAP_TYPE))

// FUNCTION_TYPE_PRODUCTION names PRODUCTION_FUNCTION_TYPE as an operand.
const FUNCTION_TYPE_PRODUCTION = string(rune(PRODUCTION_FUNCTION_TYPE))

// STRUCTURE_TYPE_PRODUCTION names PRODUCTION_STRUCTURE_TYPE as an operand.
const STRUCTURE_TYPE_PRODUCTION = string(rune(PRODUCTION_STRUCTURE_TYPE))

// FIELDS_PRODUCTION names PRODUCTION_FIELDS as an operand.
const FIELDS_PRODUCTION = string(rune(PRODUCTION_FIELDS))

// FIELD_PRODUCTION names PRODUCTION_FIELD as an operand.
const FIELD_PRODUCTION = string(rune(PRODUCTION_FIELD))

// FIELD_TYPE_PRODUCTION names PRODUCTION_FIELD_TYPE as an operand.
const FIELD_TYPE_PRODUCTION = string(rune(PRODUCTION_FIELD_TYPE))

// INTERFACE_TYPE_PRODUCTION names PRODUCTION_INTERFACE_TYPE as an operand.
const INTERFACE_TYPE_PRODUCTION = string(rune(PRODUCTION_INTERFACE_TYPE))

// ELEMENTS_PRODUCTION names PRODUCTION_ELEMENTS as an operand.
const ELEMENTS_PRODUCTION = string(rune(PRODUCTION_ELEMENTS))

// BLOCK_PRODUCTION names PRODUCTION_BLOCK as an operand.
const BLOCK_PRODUCTION = string(rune(PRODUCTION_BLOCK))

// STATEMENTS_PRODUCTION names PRODUCTION_STATEMENTS as an operand.
const STATEMENTS_PRODUCTION = string(rune(PRODUCTION_STATEMENTS))

// STATEMENT_PRODUCTION names PRODUCTION_STATEMENT as an operand.
const STATEMENT_PRODUCTION = string(rune(PRODUCTION_STATEMENT))

// DECLARATION_STATEMENT_PRODUCTION names PRODUCTION_DECLARATION_STATEMENT as an operand.
const DECLARATION_STATEMENT_PRODUCTION = string(rune(PRODUCTION_DECLARATION_STATEMENT))

// LABEL_PRODUCTION names PRODUCTION_LABEL as an operand.
const LABEL_PRODUCTION = string(rune(PRODUCTION_LABEL))

// RETURN_STATEMENT_PRODUCTION names PRODUCTION_RETURN_STATEMENT as an operand.
const RETURN_STATEMENT_PRODUCTION = string(rune(PRODUCTION_RETURN_STATEMENT))

// LAUNCH_PRODUCTION names PRODUCTION_LAUNCH as an operand.
const LAUNCH_PRODUCTION = string(rune(PRODUCTION_LAUNCH))

// JUMP_PRODUCTION names PRODUCTION_JUMP as an operand.
const JUMP_PRODUCTION = string(rune(PRODUCTION_JUMP))

// IF_STATEMENT_PRODUCTION names PRODUCTION_IF_STATEMENT as an operand.
const IF_STATEMENT_PRODUCTION = string(rune(PRODUCTION_IF_STATEMENT))

// IF_END_PRODUCTION names PRODUCTION_IF_END as an operand.
const IF_END_PRODUCTION = string(rune(PRODUCTION_IF_END))

// FOR_STATEMENT_PRODUCTION names PRODUCTION_FOR_STATEMENT as an operand.
const FOR_STATEMENT_PRODUCTION = string(rune(PRODUCTION_FOR_STATEMENT))

// FOR_HEAD_PRODUCTION names PRODUCTION_FOR_HEAD as an operand.
const FOR_HEAD_PRODUCTION = string(rune(PRODUCTION_FOR_HEAD))

// RANGE_CLAUSE_PRODUCTION names PRODUCTION_RANGE_CLAUSE as an operand.
const RANGE_CLAUSE_PRODUCTION = string(rune(PRODUCTION_RANGE_CLAUSE))

// SWITCH_STATEMENT_PRODUCTION names PRODUCTION_SWITCH_STATEMENT as an operand.
const SWITCH_STATEMENT_PRODUCTION = string(rune(PRODUCTION_SWITCH_STATEMENT))

// SWITCH_HEAD_PRODUCTION names PRODUCTION_SWITCH_HEAD as an operand.
const SWITCH_HEAD_PRODUCTION = string(rune(PRODUCTION_SWITCH_HEAD))

// SELECT_STATEMENT_PRODUCTION names PRODUCTION_SELECT_STATEMENT as an operand.
const SELECT_STATEMENT_PRODUCTION = string(rune(PRODUCTION_SELECT_STATEMENT))

// CASE_BLOCK_PRODUCTION names PRODUCTION_CASE_BLOCK as an operand.
const CASE_BLOCK_PRODUCTION = string(rune(PRODUCTION_CASE_BLOCK))

// CASES_PRODUCTION names PRODUCTION_CASES as an operand.
const CASES_PRODUCTION = string(rune(PRODUCTION_CASES))

// CASE_PRODUCTION names PRODUCTION_CASE as an operand.
const CASE_PRODUCTION = string(rune(PRODUCTION_CASE))

// CASE_COLON_PRODUCTION names PRODUCTION_CASE_COLON as an operand.
const CASE_COLON_PRODUCTION = string(rune(PRODUCTION_CASE_COLON))

// CASE_BODY_PRODUCTION names PRODUCTION_CASE_BODY as an operand.
const CASE_BODY_PRODUCTION = string(rune(PRODUCTION_CASE_BODY))

// CLAUSE_PRODUCTION names PRODUCTION_CLAUSE as an operand.
const CLAUSE_PRODUCTION = string(rune(PRODUCTION_CLAUSE))

// EXPRESSION_LIST_PRODUCTION names PRODUCTION_EXPRESSION_LIST as an operand.
const EXPRESSION_LIST_PRODUCTION = string(rune(PRODUCTION_EXPRESSION_LIST))

// BINARY_PRODUCTION names PRODUCTION_BINARY as an operand.
const BINARY_PRODUCTION = string(rune(PRODUCTION_BINARY))

// BINARY_OPERATORS_PRODUCTION names PRODUCTION_BINARY_OPERATORS as an operand.
const BINARY_OPERATORS_PRODUCTION = string(rune(PRODUCTION_BINARY_OPERATORS))

// UNARY_PRODUCTION names PRODUCTION_UNARY as an operand.
const UNARY_PRODUCTION = string(rune(PRODUCTION_UNARY))

// PRIMARY_PRODUCTION names PRODUCTION_PRIMARY as an operand.
const PRIMARY_PRODUCTION = string(rune(PRODUCTION_PRIMARY))

// SUFFIXES_PRODUCTION names PRODUCTION_SUFFIXES as an operand.
const SUFFIXES_PRODUCTION = string(rune(PRODUCTION_SUFFIXES))

// OPERAND_PRODUCTION names PRODUCTION_OPERAND as an operand.
const OPERAND_PRODUCTION = string(rune(PRODUCTION_OPERAND))

// GROUPING_PRODUCTION names PRODUCTION_GROUPING as an operand.
const GROUPING_PRODUCTION = string(rune(PRODUCTION_GROUPING))

// FUNCTION_LITERAL_PRODUCTION names PRODUCTION_FUNCTION_LITERAL as an operand.
const FUNCTION_LITERAL_PRODUCTION = string(rune(PRODUCTION_FUNCTION_LITERAL))

// SELECTOR_SUFFIX_PRODUCTION names PRODUCTION_SELECTOR_SUFFIX as an operand.
const SELECTOR_SUFFIX_PRODUCTION = string(rune(PRODUCTION_SELECTOR_SUFFIX))

// ASSERTION_SUFFIX_PRODUCTION names PRODUCTION_ASSERTION_SUFFIX as an operand.
const ASSERTION_SUFFIX_PRODUCTION = string(rune(PRODUCTION_ASSERTION_SUFFIX))

// INDEX_SUFFIX_PRODUCTION names PRODUCTION_INDEX_SUFFIX as an operand.
const INDEX_SUFFIX_PRODUCTION = string(rune(PRODUCTION_INDEX_SUFFIX))

// INDEX_ITEMS_PRODUCTION names PRODUCTION_INDEX_ITEMS as an operand.
const INDEX_ITEMS_PRODUCTION = string(rune(PRODUCTION_INDEX_ITEMS))

// CALL_SUFFIX_PRODUCTION names PRODUCTION_CALL_SUFFIX as an operand.
const CALL_SUFFIX_PRODUCTION = string(rune(PRODUCTION_CALL_SUFFIX))

// ARGUMENTS_PRODUCTION names PRODUCTION_ARGUMENTS as an operand.
const ARGUMENTS_PRODUCTION = string(rune(PRODUCTION_ARGUMENTS))

// COMPOSITE_SUFFIX_PRODUCTION names PRODUCTION_COMPOSITE_SUFFIX as an operand.
const COMPOSITE_SUFFIX_PRODUCTION = string(rune(PRODUCTION_COMPOSITE_SUFFIX))

// COMPOSITE_BODY_PRODUCTION names PRODUCTION_COMPOSITE_BODY as an operand.
const COMPOSITE_BODY_PRODUCTION = string(rune(PRODUCTION_COMPOSITE_BODY))

// COMPOSITE_ITEMS_PRODUCTION names PRODUCTION_COMPOSITE_ITEMS as an operand.
const COMPOSITE_ITEMS_PRODUCTION = string(rune(PRODUCTION_COMPOSITE_ITEMS))

// ELEMENT_PRODUCTION names PRODUCTION_ELEMENT as an operand.
const ELEMENT_PRODUCTION = string(rune(PRODUCTION_ELEMENT))

// ELEMENT_VALUE_PRODUCTION names PRODUCTION_ELEMENT_VALUE as an operand.
const ELEMENT_VALUE_PRODUCTION = string(rune(PRODUCTION_ELEMENT_VALUE))

// ENTRY_FILE is the step PRODUCTION_FILE opens at.
const ENTRY_FILE = 0

// ENTRY_PACKAGE_CLAUSE is the step PRODUCTION_PACKAGE_CLAUSE opens at.
const ENTRY_PACKAGE_CLAUSE = ENTRY_FILE +
  len(FILE_PROGRAM)/STEP_SIZE

// ENTRY_NAME is the step PRODUCTION_NAME opens at.
const ENTRY_NAME = ENTRY_PACKAGE_CLAUSE +
  len(PACKAGE_CLAUSE_PROGRAM)/STEP_SIZE

// ENTRY_DECLARATIONS is the step PRODUCTION_DECLARATIONS opens at.
const ENTRY_DECLARATIONS = ENTRY_NAME +
  len(NAME_PROGRAM)/STEP_SIZE

// ENTRY_DECLARATION is the step PRODUCTION_DECLARATION opens at.
const ENTRY_DECLARATION = ENTRY_DECLARATIONS +
  len(DECLARATIONS_PROGRAM)/STEP_SIZE

// ENTRY_END_LINE is the step PRODUCTION_END_LINE opens at.
const ENTRY_END_LINE = ENTRY_DECLARATION +
  len(DECLARATION_PROGRAM)/STEP_SIZE

// ENTRY_IMPORT is the step PRODUCTION_IMPORT opens at.
const ENTRY_IMPORT = ENTRY_END_LINE +
  len(END_LINE_PROGRAM)/STEP_SIZE

// ENTRY_IMPORT_PATH is the step PRODUCTION_IMPORT_PATH opens at.
const ENTRY_IMPORT_PATH = ENTRY_IMPORT +
  len(IMPORT_PROGRAM)/STEP_SIZE

// ENTRY_VALUE_DECLARATION is the step PRODUCTION_VALUE_DECLARATION opens at.
const ENTRY_VALUE_DECLARATION = ENTRY_IMPORT_PATH +
  len(IMPORT_PATH_PROGRAM)/STEP_SIZE

// ENTRY_CONSTANT_DECLARATION is the step PRODUCTION_CONSTANT_DECLARATION opens at.
const ENTRY_CONSTANT_DECLARATION = ENTRY_VALUE_DECLARATION +
  len(VALUE_DECLARATION_PROGRAM)/STEP_SIZE

// ENTRY_VARIABLE_DECLARATION is the step PRODUCTION_VARIABLE_DECLARATION opens at.
const ENTRY_VARIABLE_DECLARATION = ENTRY_CONSTANT_DECLARATION +
  len(CONSTANT_DECLARATION_PROGRAM)/STEP_SIZE

// ENTRY_VALUE_TAIL is the step PRODUCTION_VALUE_TAIL opens at.
const ENTRY_VALUE_TAIL = ENTRY_VARIABLE_DECLARATION +
  len(VARIABLE_DECLARATION_PROGRAM)/STEP_SIZE

// ENTRY_TYPE_DECLARATION is the step PRODUCTION_TYPE_DECLARATION opens at.
const ENTRY_TYPE_DECLARATION = ENTRY_VALUE_TAIL +
  len(VALUE_TAIL_PROGRAM)/STEP_SIZE

// ENTRY_FUNCTION_DECLARATION is the step PRODUCTION_FUNCTION_DECLARATION opens at.
const ENTRY_FUNCTION_DECLARATION = ENTRY_TYPE_DECLARATION +
  len(TYPE_DECLARATION_PROGRAM)/STEP_SIZE

// ENTRY_RECEIVER is the step PRODUCTION_RECEIVER opens at.
const ENTRY_RECEIVER = ENTRY_FUNCTION_DECLARATION +
  len(FUNCTION_DECLARATION_PROGRAM)/STEP_SIZE

// ENTRY_TYPE_PARAMETER_LIST is the step PRODUCTION_TYPE_PARAMETER_LIST opens at.
const ENTRY_TYPE_PARAMETER_LIST = ENTRY_RECEIVER +
  len(RECEIVER_PROGRAM)/STEP_SIZE

// ENTRY_TYPE_PARAMETER_ITEMS is the step PRODUCTION_TYPE_PARAMETER_ITEMS opens at.
const ENTRY_TYPE_PARAMETER_ITEMS = ENTRY_TYPE_PARAMETER_LIST +
  len(TYPE_PARAMETER_LIST_PROGRAM)/STEP_SIZE

// ENTRY_TYPE_PARAMETER_NAMES is the step PRODUCTION_TYPE_PARAMETER_NAMES opens at.
const ENTRY_TYPE_PARAMETER_NAMES = ENTRY_TYPE_PARAMETER_ITEMS +
  len(TYPE_PARAMETER_ITEMS_PROGRAM)/STEP_SIZE

// ENTRY_CONSTRAINT is the step PRODUCTION_CONSTRAINT opens at.
const ENTRY_CONSTRAINT = ENTRY_TYPE_PARAMETER_NAMES +
  len(TYPE_PARAMETER_NAMES_PROGRAM)/STEP_SIZE

// ENTRY_UNION_TERMS is the step PRODUCTION_UNION_TERMS opens at.
const ENTRY_UNION_TERMS = ENTRY_CONSTRAINT +
  len(CONSTRAINT_PROGRAM)/STEP_SIZE

// ENTRY_TERM is the step PRODUCTION_TERM opens at.
const ENTRY_TERM = ENTRY_UNION_TERMS +
  len(UNION_TERMS_PROGRAM)/STEP_SIZE

// ENTRY_SIGNATURE is the step PRODUCTION_SIGNATURE opens at.
const ENTRY_SIGNATURE = ENTRY_TERM +
  len(TERM_PROGRAM)/STEP_SIZE

// ENTRY_PARAMETERS is the step PRODUCTION_PARAMETERS opens at.
const ENTRY_PARAMETERS = ENTRY_SIGNATURE +
  len(SIGNATURE_PROGRAM)/STEP_SIZE

// ENTRY_PARAMETER_ITEMS is the step PRODUCTION_PARAMETER_ITEMS opens at.
const ENTRY_PARAMETER_ITEMS = ENTRY_PARAMETERS +
  len(PARAMETERS_PROGRAM)/STEP_SIZE

// ENTRY_RESULTS is the step PRODUCTION_RESULTS opens at.
const ENTRY_RESULTS = ENTRY_PARAMETER_ITEMS +
  len(PARAMETER_ITEMS_PROGRAM)/STEP_SIZE

// ENTRY_RESULT_ITEMS is the step PRODUCTION_RESULT_ITEMS opens at.
const ENTRY_RESULT_ITEMS = ENTRY_RESULTS +
  len(RESULTS_PROGRAM)/STEP_SIZE

// ENTRY_PARAMETER is the step PRODUCTION_PARAMETER opens at.
const ENTRY_PARAMETER = ENTRY_RESULT_ITEMS +
  len(RESULT_ITEMS_PROGRAM)/STEP_SIZE

// ENTRY_RESULT_PARAMETER is the step PRODUCTION_RESULT_PARAMETER opens at.
const ENTRY_RESULT_PARAMETER = ENTRY_PARAMETER +
  len(PARAMETER_PROGRAM)/STEP_SIZE

// ENTRY_PARAMETER_BODY is the step PRODUCTION_PARAMETER_BODY opens at.
const ENTRY_PARAMETER_BODY = ENTRY_RESULT_PARAMETER +
  len(RESULT_PARAMETER_PROGRAM)/STEP_SIZE

// ENTRY_PARAMETER_NAMES is the step PRODUCTION_PARAMETER_NAMES opens at.
const ENTRY_PARAMETER_NAMES = ENTRY_PARAMETER_BODY +
  len(PARAMETER_BODY_PROGRAM)/STEP_SIZE

// ENTRY_PARAMETER_TYPE is the step PRODUCTION_PARAMETER_TYPE opens at.
const ENTRY_PARAMETER_TYPE = ENTRY_PARAMETER_NAMES +
  len(PARAMETER_NAMES_PROGRAM)/STEP_SIZE

// ENTRY_PARAMETER_NAME is the step PRODUCTION_PARAMETER_NAME opens at.
const ENTRY_PARAMETER_NAME = ENTRY_PARAMETER_TYPE +
  len(PARAMETER_TYPE_PROGRAM)/STEP_SIZE

// ENTRY_TYPE is the step PRODUCTION_TYPE opens at.
const ENTRY_TYPE = ENTRY_PARAMETER_NAME +
  len(PARAMETER_NAME_PROGRAM)/STEP_SIZE

// ENTRY_POINTERS is the step PRODUCTION_POINTERS opens at.
const ENTRY_POINTERS = ENTRY_TYPE +
  len(TYPE_PROGRAM)/STEP_SIZE

// ENTRY_PREFIX_TYPE is the step PRODUCTION_PREFIX_TYPE opens at.
const ENTRY_PREFIX_TYPE = ENTRY_POINTERS +
  len(POINTERS_PROGRAM)/STEP_SIZE

// ENTRY_GROUPED_TYPE is the step PRODUCTION_GROUPED_TYPE opens at.
const ENTRY_GROUPED_TYPE = ENTRY_PREFIX_TYPE +
  len(PREFIX_TYPE_PROGRAM)/STEP_SIZE

// ENTRY_TYPE_NAME is the step PRODUCTION_TYPE_NAME opens at.
const ENTRY_TYPE_NAME = ENTRY_GROUPED_TYPE +
  len(GROUPED_TYPE_PROGRAM)/STEP_SIZE

// ENTRY_QUALIFIED_NAME is the step PRODUCTION_QUALIFIED_NAME opens at.
const ENTRY_QUALIFIED_NAME = ENTRY_TYPE_NAME +
  len(TYPE_NAME_PROGRAM)/STEP_SIZE

// ENTRY_GENERIC_ARGUMENTS is the step PRODUCTION_GENERIC_ARGUMENTS opens at.
const ENTRY_GENERIC_ARGUMENTS = ENTRY_QUALIFIED_NAME +
  len(QUALIFIED_NAME_PROGRAM)/STEP_SIZE

// ENTRY_ARRAY_TYPE is the step PRODUCTION_ARRAY_TYPE opens at.
const ENTRY_ARRAY_TYPE = ENTRY_GENERIC_ARGUMENTS +
  len(GENERIC_ARGUMENTS_PROGRAM)/STEP_SIZE

// ENTRY_ELEMENT_TYPE is the step PRODUCTION_ELEMENT_TYPE opens at.
const ENTRY_ELEMENT_TYPE = ENTRY_ARRAY_TYPE +
  len(ARRAY_TYPE_PROGRAM)/STEP_SIZE

// ENTRY_CHANNEL_TYPE is the step PRODUCTION_CHANNEL_TYPE opens at.
const ENTRY_CHANNEL_TYPE = ENTRY_ELEMENT_TYPE +
  len(ELEMENT_TYPE_PROGRAM)/STEP_SIZE

// ENTRY_MAP_TYPE is the step PRODUCTION_MAP_TYPE opens at.
const ENTRY_MAP_TYPE = ENTRY_CHANNEL_TYPE +
  len(CHANNEL_TYPE_PROGRAM)/STEP_SIZE

// ENTRY_FUNCTION_TYPE is the step PRODUCTION_FUNCTION_TYPE opens at.
const ENTRY_FUNCTION_TYPE = ENTRY_MAP_TYPE +
  len(MAP_TYPE_PROGRAM)/STEP_SIZE

// ENTRY_STRUCTURE_TYPE is the step PRODUCTION_STRUCTURE_TYPE opens at.
const ENTRY_STRUCTURE_TYPE = ENTRY_FUNCTION_TYPE +
  len(FUNCTION_TYPE_PROGRAM)/STEP_SIZE

// ENTRY_FIELDS is the step PRODUCTION_FIELDS opens at.
const ENTRY_FIELDS = ENTRY_STRUCTURE_TYPE +
  len(STRUCTURE_TYPE_PROGRAM)/STEP_SIZE

// ENTRY_FIELD is the step PRODUCTION_FIELD opens at.
const ENTRY_FIELD = ENTRY_FIELDS +
  len(FIELDS_PROGRAM)/STEP_SIZE

// ENTRY_FIELD_TYPE is the step PRODUCTION_FIELD_TYPE opens at.
const ENTRY_FIELD_TYPE = ENTRY_FIELD +
  len(FIELD_PROGRAM)/STEP_SIZE

// ENTRY_INTERFACE_TYPE is the step PRODUCTION_INTERFACE_TYPE opens at.
const ENTRY_INTERFACE_TYPE = ENTRY_FIELD_TYPE +
  len(FIELD_TYPE_PROGRAM)/STEP_SIZE

// ENTRY_ELEMENTS is the step PRODUCTION_ELEMENTS opens at.
const ENTRY_ELEMENTS = ENTRY_INTERFACE_TYPE +
  len(INTERFACE_TYPE_PROGRAM)/STEP_SIZE

// ENTRY_BLOCK is the step PRODUCTION_BLOCK opens at.
const ENTRY_BLOCK = ENTRY_ELEMENTS +
  len(ELEMENTS_PROGRAM)/STEP_SIZE

// ENTRY_STATEMENTS is the step PRODUCTION_STATEMENTS opens at.
const ENTRY_STATEMENTS = ENTRY_BLOCK +
  len(BLOCK_PROGRAM)/STEP_SIZE

// ENTRY_STATEMENT is the step PRODUCTION_STATEMENT opens at.
const ENTRY_STATEMENT = ENTRY_STATEMENTS +
  len(STATEMENTS_PROGRAM)/STEP_SIZE

// ENTRY_DECLARATION_STATEMENT is the step PRODUCTION_DECLARATION_STATEMENT opens at.
const ENTRY_DECLARATION_STATEMENT = ENTRY_STATEMENT +
  len(STATEMENT_PROGRAM)/STEP_SIZE

// ENTRY_LABEL is the step PRODUCTION_LABEL opens at.
const ENTRY_LABEL = ENTRY_DECLARATION_STATEMENT +
  len(DECLARATION_STATEMENT_PROGRAM)/STEP_SIZE

// ENTRY_RETURN_STATEMENT is the step PRODUCTION_RETURN_STATEMENT opens at.
const ENTRY_RETURN_STATEMENT = ENTRY_LABEL +
  len(LABEL_PROGRAM)/STEP_SIZE

// ENTRY_LAUNCH is the step PRODUCTION_LAUNCH opens at.
const ENTRY_LAUNCH = ENTRY_RETURN_STATEMENT +
  len(RETURN_STATEMENT_PROGRAM)/STEP_SIZE

// ENTRY_JUMP is the step PRODUCTION_JUMP opens at.
const ENTRY_JUMP = ENTRY_LAUNCH +
  len(LAUNCH_PROGRAM)/STEP_SIZE

// ENTRY_IF_STATEMENT is the step PRODUCTION_IF_STATEMENT opens at.
const ENTRY_IF_STATEMENT = ENTRY_JUMP +
  len(JUMP_PROGRAM)/STEP_SIZE

// ENTRY_IF_END is the step PRODUCTION_IF_END opens at.
const ENTRY_IF_END = ENTRY_IF_STATEMENT +
  len(IF_STATEMENT_PROGRAM)/STEP_SIZE

// ENTRY_FOR_STATEMENT is the step PRODUCTION_FOR_STATEMENT opens at.
const ENTRY_FOR_STATEMENT = ENTRY_IF_END +
  len(IF_END_PROGRAM)/STEP_SIZE

// ENTRY_FOR_HEAD is the step PRODUCTION_FOR_HEAD opens at.
const ENTRY_FOR_HEAD = ENTRY_FOR_STATEMENT +
  len(FOR_STATEMENT_PROGRAM)/STEP_SIZE

// ENTRY_RANGE_CLAUSE is the step PRODUCTION_RANGE_CLAUSE opens at.
const ENTRY_RANGE_CLAUSE = ENTRY_FOR_HEAD +
  len(FOR_HEAD_PROGRAM)/STEP_SIZE

// ENTRY_SWITCH_STATEMENT is the step PRODUCTION_SWITCH_STATEMENT opens at.
const ENTRY_SWITCH_STATEMENT = ENTRY_RANGE_CLAUSE +
  len(RANGE_CLAUSE_PROGRAM)/STEP_SIZE

// ENTRY_SWITCH_HEAD is the step PRODUCTION_SWITCH_HEAD opens at.
const ENTRY_SWITCH_HEAD = ENTRY_SWITCH_STATEMENT +
  len(SWITCH_STATEMENT_PROGRAM)/STEP_SIZE

// ENTRY_SELECT_STATEMENT is the step PRODUCTION_SELECT_STATEMENT opens at.
const ENTRY_SELECT_STATEMENT = ENTRY_SWITCH_HEAD +
  len(SWITCH_HEAD_PROGRAM)/STEP_SIZE

// ENTRY_CASE_BLOCK is the step PRODUCTION_CASE_BLOCK opens at.
const ENTRY_CASE_BLOCK = ENTRY_SELECT_STATEMENT +
  len(SELECT_STATEMENT_PROGRAM)/STEP_SIZE

// ENTRY_CASES is the step PRODUCTION_CASES opens at.
const ENTRY_CASES = ENTRY_CASE_BLOCK +
  len(CASE_BLOCK_PROGRAM)/STEP_SIZE

// ENTRY_CASE is the step PRODUCTION_CASE opens at.
const ENTRY_CASE = ENTRY_CASES +
  len(CASES_PROGRAM)/STEP_SIZE

// ENTRY_CASE_COLON is the step PRODUCTION_CASE_COLON opens at.
const ENTRY_CASE_COLON = ENTRY_CASE +
  len(CASE_PROGRAM)/STEP_SIZE

// ENTRY_CASE_BODY is the step PRODUCTION_CASE_BODY opens at.
const ENTRY_CASE_BODY = ENTRY_CASE_COLON +
  len(CASE_COLON_PROGRAM)/STEP_SIZE

// ENTRY_CLAUSE is the step PRODUCTION_CLAUSE opens at.
const ENTRY_CLAUSE = ENTRY_CASE_BODY +
  len(CASE_BODY_PROGRAM)/STEP_SIZE

// ENTRY_EXPRESSION_LIST is the step PRODUCTION_EXPRESSION_LIST opens at.
const ENTRY_EXPRESSION_LIST = ENTRY_CLAUSE +
  len(CLAUSE_PROGRAM)/STEP_SIZE

// ENTRY_BINARY is the step PRODUCTION_BINARY opens at.
const ENTRY_BINARY = ENTRY_EXPRESSION_LIST +
  len(EXPRESSION_LIST_PROGRAM)/STEP_SIZE

// ENTRY_BINARY_OPERATORS is the step PRODUCTION_BINARY_OPERATORS opens at.
const ENTRY_BINARY_OPERATORS = ENTRY_BINARY +
  len(BINARY_PROGRAM)/STEP_SIZE

// ENTRY_UNARY is the step PRODUCTION_UNARY opens at.
const ENTRY_UNARY = ENTRY_BINARY_OPERATORS +
  len(BINARY_OPERATORS_PROGRAM)/STEP_SIZE

// ENTRY_PRIMARY is the step PRODUCTION_PRIMARY opens at.
const ENTRY_PRIMARY = ENTRY_UNARY +
  len(UNARY_PROGRAM)/STEP_SIZE

// ENTRY_SUFFIXES is the step PRODUCTION_SUFFIXES opens at.
const ENTRY_SUFFIXES = ENTRY_PRIMARY +
  len(PRIMARY_PROGRAM)/STEP_SIZE

// ENTRY_OPERAND is the step PRODUCTION_OPERAND opens at.
const ENTRY_OPERAND = ENTRY_SUFFIXES +
  len(SUFFIXES_PROGRAM)/STEP_SIZE

// ENTRY_GROUPING is the step PRODUCTION_GROUPING opens at.
const ENTRY_GROUPING = ENTRY_OPERAND +
  len(OPERAND_PROGRAM)/STEP_SIZE

// ENTRY_FUNCTION_LITERAL is the step PRODUCTION_FUNCTION_LITERAL opens at.
const ENTRY_FUNCTION_LITERAL = ENTRY_GROUPING +
  len(GROUPING_PROGRAM)/STEP_SIZE

// ENTRY_SELECTOR_SUFFIX is the step PRODUCTION_SELECTOR_SUFFIX opens at.
const ENTRY_SELECTOR_SUFFIX = ENTRY_FUNCTION_LITERAL +
  len(FUNCTION_LITERAL_PROGRAM)/STEP_SIZE

// ENTRY_ASSERTION_SUFFIX is the step PRODUCTION_ASSERTION_SUFFIX opens at.
const ENTRY_ASSERTION_SUFFIX = ENTRY_SELECTOR_SUFFIX +
  len(SELECTOR_SUFFIX_PROGRAM)/STEP_SIZE

// ENTRY_INDEX_SUFFIX is the step PRODUCTION_INDEX_SUFFIX opens at.
const ENTRY_INDEX_SUFFIX = ENTRY_ASSERTION_SUFFIX +
  len(ASSERTION_SUFFIX_PROGRAM)/STEP_SIZE

// ENTRY_INDEX_ITEMS is the step PRODUCTION_INDEX_ITEMS opens at.
const ENTRY_INDEX_ITEMS = ENTRY_INDEX_SUFFIX +
  len(INDEX_SUFFIX_PROGRAM)/STEP_SIZE

// ENTRY_CALL_SUFFIX is the step PRODUCTION_CALL_SUFFIX opens at.
const ENTRY_CALL_SUFFIX = ENTRY_INDEX_ITEMS +
  len(INDEX_ITEMS_PROGRAM)/STEP_SIZE

// ENTRY_ARGUMENTS is the step PRODUCTION_ARGUMENTS opens at.
const ENTRY_ARGUMENTS = ENTRY_CALL_SUFFIX +
  len(CALL_SUFFIX_PROGRAM)/STEP_SIZE

// ENTRY_COMPOSITE_SUFFIX is the step PRODUCTION_COMPOSITE_SUFFIX opens at.
const ENTRY_COMPOSITE_SUFFIX = ENTRY_ARGUMENTS +
  len(ARGUMENTS_PROGRAM)/STEP_SIZE

// ENTRY_COMPOSITE_BODY is the step PRODUCTION_COMPOSITE_BODY opens at.
const ENTRY_COMPOSITE_BODY = ENTRY_COMPOSITE_SUFFIX +
  len(COMPOSITE_SUFFIX_PROGRAM)/STEP_SIZE

// ENTRY_COMPOSITE_ITEMS is the step PRODUCTION_COMPOSITE_ITEMS opens at.
const ENTRY_COMPOSITE_ITEMS = ENTRY_COMPOSITE_BODY +
  len(COMPOSITE_BODY_PROGRAM)/STEP_SIZE

// ENTRY_ELEMENT is the step PRODUCTION_ELEMENT opens at.
const ENTRY_ELEMENT = ENTRY_COMPOSITE_ITEMS +
  len(COMPOSITE_ITEMS_PROGRAM)/STEP_SIZE

// ENTRY_ELEMENT_VALUE is the step PRODUCTION_ELEMENT_VALUE opens at.
const ENTRY_ELEMENT_VALUE = ENTRY_ELEMENT +
  len(ELEMENT_PROGRAM)/STEP_SIZE

// ENTRY_CONSTANT_GROUP is the step PRODUCTION_CONSTANT_GROUP opens at.
const ENTRY_CONSTANT_GROUP = ENTRY_ELEMENT_VALUE +
  len(ELEMENT_VALUE_PROGRAM)/STEP_SIZE

// ENTRY_CONSTANT_MEMBERS is the step PRODUCTION_CONSTANT_MEMBERS opens at.
const ENTRY_CONSTANT_MEMBERS = ENTRY_CONSTANT_GROUP +
  len(CONSTANT_GROUP_PROGRAM)/STEP_SIZE

// ENTRY_GROUP_VALUE_TAIL is the step PRODUCTION_GROUP_VALUE_TAIL opens at.
const ENTRY_GROUP_VALUE_TAIL = ENTRY_CONSTANT_MEMBERS +
  len(CONSTANT_MEMBERS_PROGRAM)/STEP_SIZE

// ENTRY_VARIABLE_GROUP is the step PRODUCTION_VARIABLE_GROUP opens at.
const ENTRY_VARIABLE_GROUP = ENTRY_GROUP_VALUE_TAIL +
  len(GROUP_VALUE_TAIL_PROGRAM)/STEP_SIZE

// ENTRY_VARIABLE_MEMBERS is the step PRODUCTION_VARIABLE_MEMBERS opens at.
const ENTRY_VARIABLE_MEMBERS = ENTRY_VARIABLE_GROUP +
  len(VARIABLE_GROUP_PROGRAM)/STEP_SIZE

// ENTRY_TYPE_GROUP is the step PRODUCTION_TYPE_GROUP opens at.
const ENTRY_TYPE_GROUP = ENTRY_VARIABLE_MEMBERS +
  len(VARIABLE_MEMBERS_PROGRAM)/STEP_SIZE

// ENTRY_TYPE_MEMBERS is the step PRODUCTION_TYPE_MEMBERS opens at.
const ENTRY_TYPE_MEMBERS = ENTRY_TYPE_GROUP +
  len(TYPE_GROUP_PROGRAM)/STEP_SIZE

// ENTRY_IMPORT_GROUP is the step PRODUCTION_IMPORT_GROUP opens at.
const ENTRY_IMPORT_GROUP = ENTRY_TYPE_MEMBERS +
  len(TYPE_MEMBERS_PROGRAM)/STEP_SIZE

// ENTRY_IMPORT_MEMBERS is the step PRODUCTION_IMPORT_MEMBERS opens at.
const ENTRY_IMPORT_MEMBERS = ENTRY_IMPORT_GROUP +
  len(IMPORT_GROUP_PROGRAM)/STEP_SIZE

// ENTRY_MAXIMUM is the step the final rule opens at. Registration reads no length of a string,
// thus the step is spelled out and the bounds test holds it to the program.
const ENTRY_MAXIMUM = 953

// ENTRIES names the step each rule opens at, in the order of the rules.
const ENTRIES = string(rune(ENTRY_FILE/ENTRY_BASE)) +
  string(rune(ENTRY_FILE%ENTRY_BASE)) +
  string(rune(ENTRY_PACKAGE_CLAUSE/ENTRY_BASE)) +
  string(rune(ENTRY_PACKAGE_CLAUSE%ENTRY_BASE)) +
  string(rune(ENTRY_NAME/ENTRY_BASE)) +
  string(rune(ENTRY_NAME%ENTRY_BASE)) +
  string(rune(ENTRY_DECLARATIONS/ENTRY_BASE)) +
  string(rune(ENTRY_DECLARATIONS%ENTRY_BASE)) +
  string(rune(ENTRY_DECLARATION/ENTRY_BASE)) +
  string(rune(ENTRY_DECLARATION%ENTRY_BASE)) +
  string(rune(ENTRY_END_LINE/ENTRY_BASE)) +
  string(rune(ENTRY_END_LINE%ENTRY_BASE)) +
  string(rune(ENTRY_IMPORT/ENTRY_BASE)) +
  string(rune(ENTRY_IMPORT%ENTRY_BASE)) +
  string(rune(ENTRY_IMPORT_PATH/ENTRY_BASE)) +
  string(rune(ENTRY_IMPORT_PATH%ENTRY_BASE)) +
  string(rune(ENTRY_VALUE_DECLARATION/ENTRY_BASE)) +
  string(rune(ENTRY_VALUE_DECLARATION%ENTRY_BASE)) +
  string(rune(ENTRY_CONSTANT_DECLARATION/ENTRY_BASE)) +
  string(rune(ENTRY_CONSTANT_DECLARATION%ENTRY_BASE)) +
  string(rune(ENTRY_VARIABLE_DECLARATION/ENTRY_BASE)) +
  string(rune(ENTRY_VARIABLE_DECLARATION%ENTRY_BASE)) +
  string(rune(ENTRY_VALUE_TAIL/ENTRY_BASE)) +
  string(rune(ENTRY_VALUE_TAIL%ENTRY_BASE)) +
  string(rune(ENTRY_TYPE_DECLARATION/ENTRY_BASE)) +
  string(rune(ENTRY_TYPE_DECLARATION%ENTRY_BASE)) +
  string(rune(ENTRY_FUNCTION_DECLARATION/ENTRY_BASE)) +
  string(rune(ENTRY_FUNCTION_DECLARATION%ENTRY_BASE)) +
  string(rune(ENTRY_RECEIVER/ENTRY_BASE)) +
  string(rune(ENTRY_RECEIVER%ENTRY_BASE)) +
  string(rune(ENTRY_TYPE_PARAMETER_LIST/ENTRY_BASE)) +
  string(rune(ENTRY_TYPE_PARAMETER_LIST%ENTRY_BASE)) +
  string(rune(ENTRY_TYPE_PARAMETER_ITEMS/ENTRY_BASE)) +
  string(rune(ENTRY_TYPE_PARAMETER_ITEMS%ENTRY_BASE)) +
  string(rune(ENTRY_TYPE_PARAMETER_NAMES/ENTRY_BASE)) +
  string(rune(ENTRY_TYPE_PARAMETER_NAMES%ENTRY_BASE)) +
  string(rune(ENTRY_CONSTRAINT/ENTRY_BASE)) +
  string(rune(ENTRY_CONSTRAINT%ENTRY_BASE)) +
  string(rune(ENTRY_UNION_TERMS/ENTRY_BASE)) +
  string(rune(ENTRY_UNION_TERMS%ENTRY_BASE)) +
  string(rune(ENTRY_TERM/ENTRY_BASE)) +
  string(rune(ENTRY_TERM%ENTRY_BASE)) +
  string(rune(ENTRY_SIGNATURE/ENTRY_BASE)) +
  string(rune(ENTRY_SIGNATURE%ENTRY_BASE)) +
  string(rune(ENTRY_PARAMETERS/ENTRY_BASE)) +
  string(rune(ENTRY_PARAMETERS%ENTRY_BASE)) +
  string(rune(ENTRY_PARAMETER_ITEMS/ENTRY_BASE)) +
  string(rune(ENTRY_PARAMETER_ITEMS%ENTRY_BASE)) +
  string(rune(ENTRY_RESULTS/ENTRY_BASE)) +
  string(rune(ENTRY_RESULTS%ENTRY_BASE)) +
  string(rune(ENTRY_RESULT_ITEMS/ENTRY_BASE)) +
  string(rune(ENTRY_RESULT_ITEMS%ENTRY_BASE)) +
  string(rune(ENTRY_PARAMETER/ENTRY_BASE)) +
  string(rune(ENTRY_PARAMETER%ENTRY_BASE)) +
  string(rune(ENTRY_RESULT_PARAMETER/ENTRY_BASE)) +
  string(rune(ENTRY_RESULT_PARAMETER%ENTRY_BASE)) +
  string(rune(ENTRY_PARAMETER_BODY/ENTRY_BASE)) +
  string(rune(ENTRY_PARAMETER_BODY%ENTRY_BASE)) +
  string(rune(ENTRY_PARAMETER_NAMES/ENTRY_BASE)) +
  string(rune(ENTRY_PARAMETER_NAMES%ENTRY_BASE)) +
  string(rune(ENTRY_PARAMETER_TYPE/ENTRY_BASE)) +
  string(rune(ENTRY_PARAMETER_TYPE%ENTRY_BASE)) +
  string(rune(ENTRY_PARAMETER_NAME/ENTRY_BASE)) +
  string(rune(ENTRY_PARAMETER_NAME%ENTRY_BASE)) +
  string(rune(ENTRY_TYPE/ENTRY_BASE)) +
  string(rune(ENTRY_TYPE%ENTRY_BASE)) +
  string(rune(ENTRY_POINTERS/ENTRY_BASE)) +
  string(rune(ENTRY_POINTERS%ENTRY_BASE)) +
  string(rune(ENTRY_PREFIX_TYPE/ENTRY_BASE)) +
  string(rune(ENTRY_PREFIX_TYPE%ENTRY_BASE)) +
  string(rune(ENTRY_GROUPED_TYPE/ENTRY_BASE)) +
  string(rune(ENTRY_GROUPED_TYPE%ENTRY_BASE)) +
  string(rune(ENTRY_TYPE_NAME/ENTRY_BASE)) +
  string(rune(ENTRY_TYPE_NAME%ENTRY_BASE)) +
  string(rune(ENTRY_QUALIFIED_NAME/ENTRY_BASE)) +
  string(rune(ENTRY_QUALIFIED_NAME%ENTRY_BASE)) +
  string(rune(ENTRY_GENERIC_ARGUMENTS/ENTRY_BASE)) +
  string(rune(ENTRY_GENERIC_ARGUMENTS%ENTRY_BASE)) +
  string(rune(ENTRY_ARRAY_TYPE/ENTRY_BASE)) +
  string(rune(ENTRY_ARRAY_TYPE%ENTRY_BASE)) +
  string(rune(ENTRY_ELEMENT_TYPE/ENTRY_BASE)) +
  string(rune(ENTRY_ELEMENT_TYPE%ENTRY_BASE)) +
  string(rune(ENTRY_CHANNEL_TYPE/ENTRY_BASE)) +
  string(rune(ENTRY_CHANNEL_TYPE%ENTRY_BASE)) +
  string(rune(ENTRY_MAP_TYPE/ENTRY_BASE)) +
  string(rune(ENTRY_MAP_TYPE%ENTRY_BASE)) +
  string(rune(ENTRY_FUNCTION_TYPE/ENTRY_BASE)) +
  string(rune(ENTRY_FUNCTION_TYPE%ENTRY_BASE)) +
  string(rune(ENTRY_STRUCTURE_TYPE/ENTRY_BASE)) +
  string(rune(ENTRY_STRUCTURE_TYPE%ENTRY_BASE)) +
  string(rune(ENTRY_FIELDS/ENTRY_BASE)) +
  string(rune(ENTRY_FIELDS%ENTRY_BASE)) +
  string(rune(ENTRY_FIELD/ENTRY_BASE)) +
  string(rune(ENTRY_FIELD%ENTRY_BASE)) +
  string(rune(ENTRY_FIELD_TYPE/ENTRY_BASE)) +
  string(rune(ENTRY_FIELD_TYPE%ENTRY_BASE)) +
  string(rune(ENTRY_INTERFACE_TYPE/ENTRY_BASE)) +
  string(rune(ENTRY_INTERFACE_TYPE%ENTRY_BASE)) +
  string(rune(ENTRY_ELEMENTS/ENTRY_BASE)) +
  string(rune(ENTRY_ELEMENTS%ENTRY_BASE)) +
  string(rune(ENTRY_BLOCK/ENTRY_BASE)) +
  string(rune(ENTRY_BLOCK%ENTRY_BASE)) +
  string(rune(ENTRY_STATEMENTS/ENTRY_BASE)) +
  string(rune(ENTRY_STATEMENTS%ENTRY_BASE)) +
  string(rune(ENTRY_STATEMENT/ENTRY_BASE)) +
  string(rune(ENTRY_STATEMENT%ENTRY_BASE)) +
  string(rune(ENTRY_DECLARATION_STATEMENT/ENTRY_BASE)) +
  string(rune(ENTRY_DECLARATION_STATEMENT%ENTRY_BASE)) +
  string(rune(ENTRY_LABEL/ENTRY_BASE)) +
  string(rune(ENTRY_LABEL%ENTRY_BASE)) +
  string(rune(ENTRY_RETURN_STATEMENT/ENTRY_BASE)) +
  string(rune(ENTRY_RETURN_STATEMENT%ENTRY_BASE)) +
  string(rune(ENTRY_LAUNCH/ENTRY_BASE)) +
  string(rune(ENTRY_LAUNCH%ENTRY_BASE)) +
  string(rune(ENTRY_JUMP/ENTRY_BASE)) +
  string(rune(ENTRY_JUMP%ENTRY_BASE)) +
  string(rune(ENTRY_IF_STATEMENT/ENTRY_BASE)) +
  string(rune(ENTRY_IF_STATEMENT%ENTRY_BASE)) +
  string(rune(ENTRY_IF_END/ENTRY_BASE)) +
  string(rune(ENTRY_IF_END%ENTRY_BASE)) +
  string(rune(ENTRY_FOR_STATEMENT/ENTRY_BASE)) +
  string(rune(ENTRY_FOR_STATEMENT%ENTRY_BASE)) +
  string(rune(ENTRY_FOR_HEAD/ENTRY_BASE)) +
  string(rune(ENTRY_FOR_HEAD%ENTRY_BASE)) +
  string(rune(ENTRY_RANGE_CLAUSE/ENTRY_BASE)) +
  string(rune(ENTRY_RANGE_CLAUSE%ENTRY_BASE)) +
  string(rune(ENTRY_SWITCH_STATEMENT/ENTRY_BASE)) +
  string(rune(ENTRY_SWITCH_STATEMENT%ENTRY_BASE)) +
  string(rune(ENTRY_SWITCH_HEAD/ENTRY_BASE)) +
  string(rune(ENTRY_SWITCH_HEAD%ENTRY_BASE)) +
  string(rune(ENTRY_SELECT_STATEMENT/ENTRY_BASE)) +
  string(rune(ENTRY_SELECT_STATEMENT%ENTRY_BASE)) +
  string(rune(ENTRY_CASE_BLOCK/ENTRY_BASE)) +
  string(rune(ENTRY_CASE_BLOCK%ENTRY_BASE)) +
  string(rune(ENTRY_CASES/ENTRY_BASE)) +
  string(rune(ENTRY_CASES%ENTRY_BASE)) +
  string(rune(ENTRY_CASE/ENTRY_BASE)) +
  string(rune(ENTRY_CASE%ENTRY_BASE)) +
  string(rune(ENTRY_CASE_COLON/ENTRY_BASE)) +
  string(rune(ENTRY_CASE_COLON%ENTRY_BASE)) +
  string(rune(ENTRY_CASE_BODY/ENTRY_BASE)) +
  string(rune(ENTRY_CASE_BODY%ENTRY_BASE)) +
  string(rune(ENTRY_CLAUSE/ENTRY_BASE)) +
  string(rune(ENTRY_CLAUSE%ENTRY_BASE)) +
  string(rune(ENTRY_EXPRESSION_LIST/ENTRY_BASE)) +
  string(rune(ENTRY_EXPRESSION_LIST%ENTRY_BASE)) +
  string(rune(ENTRY_BINARY/ENTRY_BASE)) +
  string(rune(ENTRY_BINARY%ENTRY_BASE)) +
  string(rune(ENTRY_BINARY_OPERATORS/ENTRY_BASE)) +
  string(rune(ENTRY_BINARY_OPERATORS%ENTRY_BASE)) +
  string(rune(ENTRY_UNARY/ENTRY_BASE)) +
  string(rune(ENTRY_UNARY%ENTRY_BASE)) +
  string(rune(ENTRY_PRIMARY/ENTRY_BASE)) +
  string(rune(ENTRY_PRIMARY%ENTRY_BASE)) +
  string(rune(ENTRY_SUFFIXES/ENTRY_BASE)) +
  string(rune(ENTRY_SUFFIXES%ENTRY_BASE)) +
  string(rune(ENTRY_OPERAND/ENTRY_BASE)) +
  string(rune(ENTRY_OPERAND%ENTRY_BASE)) +
  string(rune(ENTRY_GROUPING/ENTRY_BASE)) +
  string(rune(ENTRY_GROUPING%ENTRY_BASE)) +
  string(rune(ENTRY_FUNCTION_LITERAL/ENTRY_BASE)) +
  string(rune(ENTRY_FUNCTION_LITERAL%ENTRY_BASE)) +
  string(rune(ENTRY_SELECTOR_SUFFIX/ENTRY_BASE)) +
  string(rune(ENTRY_SELECTOR_SUFFIX%ENTRY_BASE)) +
  string(rune(ENTRY_ASSERTION_SUFFIX/ENTRY_BASE)) +
  string(rune(ENTRY_ASSERTION_SUFFIX%ENTRY_BASE)) +
  string(rune(ENTRY_INDEX_SUFFIX/ENTRY_BASE)) +
  string(rune(ENTRY_INDEX_SUFFIX%ENTRY_BASE)) +
  string(rune(ENTRY_INDEX_ITEMS/ENTRY_BASE)) +
  string(rune(ENTRY_INDEX_ITEMS%ENTRY_BASE)) +
  string(rune(ENTRY_CALL_SUFFIX/ENTRY_BASE)) +
  string(rune(ENTRY_CALL_SUFFIX%ENTRY_BASE)) +
  string(rune(ENTRY_ARGUMENTS/ENTRY_BASE)) +
  string(rune(ENTRY_ARGUMENTS%ENTRY_BASE)) +
  string(rune(ENTRY_COMPOSITE_SUFFIX/ENTRY_BASE)) +
  string(rune(ENTRY_COMPOSITE_SUFFIX%ENTRY_BASE)) +
  string(rune(ENTRY_COMPOSITE_BODY/ENTRY_BASE)) +
  string(rune(ENTRY_COMPOSITE_BODY%ENTRY_BASE)) +
  string(rune(ENTRY_COMPOSITE_ITEMS/ENTRY_BASE)) +
  string(rune(ENTRY_COMPOSITE_ITEMS%ENTRY_BASE)) +
  string(rune(ENTRY_ELEMENT/ENTRY_BASE)) +
  string(rune(ENTRY_ELEMENT%ENTRY_BASE)) +
  string(rune(ENTRY_ELEMENT_VALUE/ENTRY_BASE)) +
  string(rune(ENTRY_ELEMENT_VALUE%ENTRY_BASE)) +
  string(rune(ENTRY_CONSTANT_GROUP/ENTRY_BASE)) +
  string(rune(ENTRY_CONSTANT_GROUP%ENTRY_BASE)) +
  string(rune(ENTRY_CONSTANT_MEMBERS/ENTRY_BASE)) +
  string(rune(ENTRY_CONSTANT_MEMBERS%ENTRY_BASE)) +
  string(rune(ENTRY_VARIABLE_GROUP/ENTRY_BASE)) +
  string(rune(ENTRY_VARIABLE_GROUP%ENTRY_BASE)) +
  string(rune(ENTRY_VARIABLE_MEMBERS/ENTRY_BASE)) +
  string(rune(ENTRY_VARIABLE_MEMBERS%ENTRY_BASE)) +
  string(rune(ENTRY_TYPE_GROUP/ENTRY_BASE)) +
  string(rune(ENTRY_TYPE_GROUP%ENTRY_BASE)) +
  string(rune(ENTRY_TYPE_MEMBERS/ENTRY_BASE)) +
  string(rune(ENTRY_TYPE_MEMBERS%ENTRY_BASE)) +
  string(rune(ENTRY_IMPORT_GROUP/ENTRY_BASE)) +
  string(rune(ENTRY_IMPORT_GROUP%ENTRY_BASE)) +
  string(rune(ENTRY_IMPORT_MEMBERS/ENTRY_BASE)) +
  string(rune(ENTRY_IMPORT_MEMBERS%ENTRY_BASE)) +
  string(rune(ENTRY_GROUP_VALUE_TAIL/ENTRY_BASE)) +
  string(rune(ENTRY_GROUP_VALUE_TAIL%ENTRY_BASE))

// PROGRAM is the grammar: the steps of every rule, in the order of the rules.
const PROGRAM = FILE_PROGRAM +
  PACKAGE_CLAUSE_PROGRAM +
  NAME_PROGRAM +
  DECLARATIONS_PROGRAM +
  DECLARATION_PROGRAM +
  END_LINE_PROGRAM +
  IMPORT_PROGRAM +
  IMPORT_PATH_PROGRAM +
  VALUE_DECLARATION_PROGRAM +
  CONSTANT_DECLARATION_PROGRAM +
  VARIABLE_DECLARATION_PROGRAM +
  VALUE_TAIL_PROGRAM +
  TYPE_DECLARATION_PROGRAM +
  FUNCTION_DECLARATION_PROGRAM +
  RECEIVER_PROGRAM +
  TYPE_PARAMETER_LIST_PROGRAM +
  TYPE_PARAMETER_ITEMS_PROGRAM +
  TYPE_PARAMETER_NAMES_PROGRAM +
  CONSTRAINT_PROGRAM +
  UNION_TERMS_PROGRAM +
  TERM_PROGRAM +
  SIGNATURE_PROGRAM +
  PARAMETERS_PROGRAM +
  PARAMETER_ITEMS_PROGRAM +
  RESULTS_PROGRAM +
  RESULT_ITEMS_PROGRAM +
  PARAMETER_PROGRAM +
  RESULT_PARAMETER_PROGRAM +
  PARAMETER_BODY_PROGRAM +
  PARAMETER_NAMES_PROGRAM +
  PARAMETER_TYPE_PROGRAM +
  PARAMETER_NAME_PROGRAM +
  TYPE_PROGRAM +
  POINTERS_PROGRAM +
  PREFIX_TYPE_PROGRAM +
  GROUPED_TYPE_PROGRAM +
  TYPE_NAME_PROGRAM +
  QUALIFIED_NAME_PROGRAM +
  GENERIC_ARGUMENTS_PROGRAM +
  ARRAY_TYPE_PROGRAM +
  ELEMENT_TYPE_PROGRAM +
  CHANNEL_TYPE_PROGRAM +
  MAP_TYPE_PROGRAM +
  FUNCTION_TYPE_PROGRAM +
  STRUCTURE_TYPE_PROGRAM +
  FIELDS_PROGRAM +
  FIELD_PROGRAM +
  FIELD_TYPE_PROGRAM +
  INTERFACE_TYPE_PROGRAM +
  ELEMENTS_PROGRAM +
  BLOCK_PROGRAM +
  STATEMENTS_PROGRAM +
  STATEMENT_PROGRAM +
  DECLARATION_STATEMENT_PROGRAM +
  LABEL_PROGRAM +
  RETURN_STATEMENT_PROGRAM +
  LAUNCH_PROGRAM +
  JUMP_PROGRAM +
  IF_STATEMENT_PROGRAM +
  IF_END_PROGRAM +
  FOR_STATEMENT_PROGRAM +
  FOR_HEAD_PROGRAM +
  RANGE_CLAUSE_PROGRAM +
  SWITCH_STATEMENT_PROGRAM +
  SWITCH_HEAD_PROGRAM +
  SELECT_STATEMENT_PROGRAM +
  CASE_BLOCK_PROGRAM +
  CASES_PROGRAM +
  CASE_PROGRAM +
  CASE_COLON_PROGRAM +
  CASE_BODY_PROGRAM +
  CLAUSE_PROGRAM +
  EXPRESSION_LIST_PROGRAM +
  BINARY_PROGRAM +
  BINARY_OPERATORS_PROGRAM +
  UNARY_PROGRAM +
  PRIMARY_PROGRAM +
  SUFFIXES_PROGRAM +
  OPERAND_PROGRAM +
  GROUPING_PROGRAM +
  FUNCTION_LITERAL_PROGRAM +
  SELECTOR_SUFFIX_PROGRAM +
  ASSERTION_SUFFIX_PROGRAM +
  INDEX_SUFFIX_PROGRAM +
  INDEX_ITEMS_PROGRAM +
  CALL_SUFFIX_PROGRAM +
  ARGUMENTS_PROGRAM +
  COMPOSITE_SUFFIX_PROGRAM +
  COMPOSITE_BODY_PROGRAM +
  COMPOSITE_ITEMS_PROGRAM +
  ELEMENT_PROGRAM +
  ELEMENT_VALUE_PROGRAM +
  CONSTANT_GROUP_PROGRAM +
  CONSTANT_MEMBERS_PROGRAM +
  GROUP_VALUE_TAIL_PROGRAM +
  VARIABLE_GROUP_PROGRAM +
  VARIABLE_MEMBERS_PROGRAM +
  TYPE_GROUP_PROGRAM +
  TYPE_MEMBERS_PROGRAM +
  IMPORT_GROUP_PROGRAM +
  IMPORT_MEMBERS_PROGRAM

// FILE_PROGRAM takes the trivia ahead of the package clause into the file, and reads the clause
// and every declaration behind it.
const FILE_PROGRAM = SKIP_COMMENTS +
  CALL +
  PACKAGE_CLAUSE_PRODUCTION +
  TAIL +
  DECLARATIONS_PRODUCTION

// PACKAGE_CLAUSE_PROGRAM refuses a file that opens with anything but the package keyword, and
// reads the one name the clause states.
const PACKAGE_CLAUSE_PROGRAM = AT +
  PACKAGE_TOKEN +
  UNLESS +
  REJECT +
  PACKAGE_CLAUSE_CAUSE +
  ADVANCE +
  CALL +
  NAME_PRODUCTION +
  TAIL +
  END_LINE_PRODUCTION

// DECLARATIONS_PROGRAM reads declarations until the end of the file. A run of empty declarations
// writes no node, thus the run passes in one step.
const DECLARATIONS_PROGRAM = SKIP_EMPTY +
  ACCEPT +
  END_OF_FILE_TOKEN +
  WHEN +
  RETURN +
  CALL +
  DECLARATION_PRODUCTION +
  TAIL +
  DECLARATIONS_PRODUCTION

// DECLARATION_PROGRAM hands one declaration to the rule its keyword names.
const DECLARATION_PROGRAM = AT +
  IMPORT_TOKEN +
  WHEN +
  TAIL +
  IMPORT_PRODUCTION +
  AT +
  CONSTANT_TOKEN +
  OR_AT +
  VARIABLE_TOKEN +
  WHEN +
  TAIL +
  VALUE_DECLARATION_PRODUCTION +
  AT +
  TYPE_TOKEN +
  WHEN +
  TAIL +
  TYPE_DECLARATION_PRODUCTION +
  AT +
  FUNCTION_TOKEN +
  WHEN +
  TAIL +
  FUNCTION_DECLARATION_PRODUCTION +
  REJECT +
  SYNTAX_CAUSE

// END_LINE_PROGRAM takes any trailing comment, then the semicolon the scanner wrote or inserted.
// A closing bracket stands for the semicolon Go lets a source leave out.
const END_LINE_PROGRAM = SKIP_COMMENTS +
  ACCEPT +
  SEMICOLON_TOKEN +
  WHEN +
  RETURN +
  AT +
  BRACE_RIGHT_TOKEN +
  OR_AT +
  PARENTHESIS_RIGHT_TOKEN +
  OR_AT +
  BRACKET_RIGHT_TOKEN +
  OR_AT +
  END_OF_FILE_TOKEN +
  OR_AT +
  CASE_TOKEN +
  OR_AT +
  DEFAULT_TOKEN +
  OR_AT +
  COMMA_TOKEN +
  OR_AT +
  COLON_TOKEN +
  OR_AT +
  ELSE_TOKEN +
  WHEN +
  RETURN +
  REJECT +
  SYNTAX_CAUSE

// NAME_PROGRAM takes one identifier as a node of its own.
const NAME_PROGRAM = NAME + IDENTIFIER_NODE + RETURN

// IMPORT_PROGRAM reads one import, or the run of them a group holds. The run states one import
// for each path, thus the tree holds what the print writes.
const IMPORT_PROGRAM = AFTER +
  PARENTHESIS_LEFT_TOKEN +
  WHEN +
  TAIL +
  IMPORT_GROUP_PRODUCTION +
  ADVANCE +
  CALL +
  IMPORT_PATH_PRODUCTION +
  TAIL +
  END_LINE_PRODUCTION

// IMPORT_GROUP_PROGRAM steps past the keyword and the bracket the run opens at.
const IMPORT_GROUP_PROGRAM = ADVANCE + ADVANCE + TAIL + IMPORT_MEMBERS_PRODUCTION

// IMPORT_MEMBERS_PROGRAM reads one import of the run and the run behind it.
const IMPORT_MEMBERS_PROGRAM = ACCEPT +
  PARENTHESIS_RIGHT_TOKEN +
  WHEN +
  TAIL +
  END_LINE_PRODUCTION +
  CALL +
  IMPORT_PATH_PRODUCTION +
  CALL +
  END_LINE_PRODUCTION +
  TAIL +
  IMPORT_MEMBERS_PRODUCTION

// IMPORT_PATH_PROGRAM reads one path and its optional name. A blank import parses, because the
// linter must see the form it rejects.
const IMPORT_PATH_PROGRAM = OPEN +
  IMPORT_NODE +
  AT +
  IDENTIFIER_TOKEN +
  WHEN +
  TAKE +
  IMPORT_NAME_NODE +
  AT +
  PERIOD_TOKEN +
  WHEN +
  REJECT +
  DOT_IMPORT_CAUSE +
  AT +
  STRING_TOKEN +
  UNLESS +
  REJECT +
  SYNTAX_CAUSE +
  TAKE +
  STRING_NODE +
  CLOSE +
  RETURN

// VALUE_DECLARATION_PROGRAM hands one value declaration to the rule its keyword names. A constant
// name answers to a case rule a variable name does not, thus the two read their names apart.
const VALUE_DECLARATION_PROGRAM = AT +
  CONSTANT_TOKEN +
  WHEN +
  TAIL +
  CONSTANT_DECLARATION_PRODUCTION +
  TAIL +
  VARIABLE_DECLARATION_PRODUCTION

// CONSTANT_DECLARATION_PROGRAM reads one const declaration, or the parenthesized run of them the
// author wrote. The run states one declaration for each name it holds, thus the tree holds what
// the print writes and the group stands nowhere in it.
const CONSTANT_DECLARATION_PROGRAM = AFTER +
  PARENTHESIS_LEFT_TOKEN +
  WHEN +
  TAIL +
  CONSTANT_GROUP_PRODUCTION +
  OPEN +
  CONSTANT_NODE +
  ADVANCE +
  NAMES +
  CONSTANT_NAME_NODE +
  TAIL +
  VALUE_TAIL_PRODUCTION

// CONSTANT_GROUP_PROGRAM steps past the keyword and the bracket the run opens at.
const CONSTANT_GROUP_PROGRAM = ADVANCE + ADVANCE + TAIL + CONSTANT_MEMBERS_PRODUCTION

// CONSTANT_MEMBERS_PROGRAM reads one constant of the run and the run behind it. The bracket that
// closes the run closes the line it stands on, thus the read hands back where the run ends.
const CONSTANT_MEMBERS_PROGRAM = ACCEPT +
  PARENTHESIS_RIGHT_TOKEN +
  WHEN +
  TAIL +
  END_LINE_PRODUCTION +
  OPEN +
  CONSTANT_NODE +
  NAMES +
  CONSTANT_NAME_NODE +
  CALL +
  GROUP_VALUE_TAIL_PRODUCTION +
  TAIL +
  CONSTANT_MEMBERS_PRODUCTION

// GROUP_VALUE_TAIL_PROGRAM reads the type and the values one member of a run states, and hands
// the read back to the run rather than to the rule that opened it.
const GROUP_VALUE_TAIL_PROGRAM = AT +
  ASSIGN_TOKEN +
  OR_AT +
  SEMICOLON_TOKEN +
  OR_AT +
  END_OF_FILE_TOKEN +
  OR_AT +
  PARENTHESIS_RIGHT_TOKEN +
  UNLESS +
  CALL +
  TYPE_PRODUCTION +
  ACCEPT +
  ASSIGN_TOKEN +
  WHEN +
  CALL +
  EXPRESSION_LIST_PRODUCTION +
  CLOSE +
  CALL +
  END_LINE_PRODUCTION +
  RETURN

// VARIABLE_DECLARATION_PROGRAM reads one var declaration, or the run of them a group holds.
const VARIABLE_DECLARATION_PROGRAM = AFTER +
  PARENTHESIS_LEFT_TOKEN +
  WHEN +
  TAIL +
  VARIABLE_GROUP_PRODUCTION +
  OPEN +
  VARIABLE_NODE +
  ADVANCE +
  NAMES +
  IDENTIFIER_NODE +
  TAIL +
  VALUE_TAIL_PRODUCTION

// VARIABLE_GROUP_PROGRAM steps past the keyword and the bracket the run opens at.
const VARIABLE_GROUP_PROGRAM = ADVANCE + ADVANCE + TAIL + VARIABLE_MEMBERS_PRODUCTION

// VARIABLE_MEMBERS_PROGRAM reads one variable of the run and the run behind it.
const VARIABLE_MEMBERS_PROGRAM = ACCEPT +
  PARENTHESIS_RIGHT_TOKEN +
  WHEN +
  TAIL +
  END_LINE_PRODUCTION +
  OPEN +
  VARIABLE_NODE +
  NAMES +
  IDENTIFIER_NODE +
  CALL +
  GROUP_VALUE_TAIL_PRODUCTION +
  TAIL +
  VARIABLE_MEMBERS_PRODUCTION

// VALUE_TAIL_PROGRAM reads the optional type and the optional values behind the names.
const VALUE_TAIL_PROGRAM = AT +
  ASSIGN_TOKEN +
  OR_AT +
  SEMICOLON_TOKEN +
  OR_AT +
  END_OF_FILE_TOKEN +
  UNLESS +
  CALL +
  TYPE_PRODUCTION +
  ACCEPT +
  ASSIGN_TOKEN +
  WHEN +
  CALL +
  EXPRESSION_LIST_PRODUCTION +
  CLOSE +
  TAIL +
  END_LINE_PRODUCTION

// TYPE_DECLARATION_PROGRAM reads one type declaration, or the run of them a group holds, and
// refuses an alias wherever it stands.
const TYPE_DECLARATION_PROGRAM = AFTER +
  PARENTHESIS_LEFT_TOKEN +
  WHEN +
  TAIL +
  TYPE_GROUP_PRODUCTION +
  OPEN +
  TYPE_NODE +
  ADVANCE +
  CALL +
  NAME_PRODUCTION +
  TYPE_PARAMETERS +
  WHEN +
  CALL +
  TYPE_PARAMETER_LIST_PRODUCTION +
  AT +
  ASSIGN_TOKEN +
  WHEN +
  REJECT +
  TYPE_ALIAS_CAUSE +
  CALL +
  TYPE_PRODUCTION +
  CLOSE +
  TAIL +
  END_LINE_PRODUCTION

// TYPE_GROUP_PROGRAM steps past the keyword and the bracket the run opens at.
const TYPE_GROUP_PROGRAM = ADVANCE + ADVANCE + TAIL + TYPE_MEMBERS_PRODUCTION

// TYPE_MEMBERS_PROGRAM reads one type of the run and the run behind it.
const TYPE_MEMBERS_PROGRAM = ACCEPT +
  PARENTHESIS_RIGHT_TOKEN +
  WHEN +
  TAIL +
  END_LINE_PRODUCTION +
  OPEN +
  TYPE_NODE +
  CALL +
  NAME_PRODUCTION +
  TYPE_PARAMETERS +
  WHEN +
  CALL +
  TYPE_PARAMETER_LIST_PRODUCTION +
  AT +
  ASSIGN_TOKEN +
  WHEN +
  REJECT +
  TYPE_ALIAS_CAUSE +
  CALL +
  TYPE_PRODUCTION +
  CLOSE +
  CALL +
  END_LINE_PRODUCTION +
  TAIL +
  TYPE_MEMBERS_PRODUCTION

// FUNCTION_DECLARATION_PROGRAM reads one function or method declaration.
const FUNCTION_DECLARATION_PROGRAM = OPEN +
  FUNCTION_NODE +
  ADVANCE +
  AT +
  PARENTHESIS_LEFT_TOKEN +
  WHEN +
  CALL +
  RECEIVER_PRODUCTION +
  CALL +
  NAME_PRODUCTION +
  AT +
  BRACKET_LEFT_TOKEN +
  WHEN +
  CALL +
  TYPE_PARAMETER_LIST_PRODUCTION +
  CALL +
  SIGNATURE_PRODUCTION +
  AT +
  BRACE_LEFT_TOKEN +
  WHEN +
  CALL +
  BLOCK_PRODUCTION +
  CLOSE +
  TAIL +
  END_LINE_PRODUCTION

// RECEIVER_PROGRAM reads the receiver of one method.
const RECEIVER_PROGRAM = OPEN + RECEIVER_NODE + CALL + PARAMETERS_PRODUCTION + CLOSE + RETURN

// TYPE_PARAMETER_LIST_PROGRAM steps past the bracket that opens the list.
const TYPE_PARAMETER_LIST_PROGRAM = ADVANCE + TAIL + TYPE_PARAMETER_ITEMS_PRODUCTION

// TYPE_PARAMETER_ITEMS_PROGRAM reads each type parameter up to the closing bracket.
const TYPE_PARAMETER_ITEMS_PROGRAM = ACCEPT +
  BRACKET_RIGHT_TOKEN +
  WHEN +
  RETURN +
  OPEN +
  TYPE_PARAMETER_NODE +
  CALL +
  NAME_PRODUCTION +
  CALL +
  TYPE_PARAMETER_NAMES_PRODUCTION +
  AT +
  BRACKET_RIGHT_TOKEN +
  UNLESS +
  CALL +
  CONSTRAINT_PRODUCTION +
  CLOSE +
  ACCEPT +
  COMMA_TOKEN +
  TAIL +
  TYPE_PARAMETER_ITEMS_PRODUCTION

// TYPE_PARAMETER_NAMES_PROGRAM reads every name behind a comma, and stops at a comma the list
// closes behind.
const TYPE_PARAMETER_NAMES_PROGRAM = ACCEPT +
  COMMA_TOKEN +
  UNLESS +
  RETURN +
  AT +
  BRACKET_RIGHT_TOKEN +
  WHEN +
  RETURN +
  NAME +
  IDENTIFIER_NODE +
  TAIL +
  TYPE_PARAMETER_NAMES_PRODUCTION

// CONSTRAINT_PROGRAM reads one term, and a union where a bar follows it.
const CONSTRAINT_PROGRAM = CALL +
  TERM_PRODUCTION +
  AT +
  OR_TOKEN +
  UNLESS +
  RETURN +
  WRAP +
  UNION_NODE +
  TAIL +
  UNION_TERMS_PRODUCTION

// UNION_TERMS_PROGRAM reads every term behind a bar.
const UNION_TERMS_PROGRAM = ACCEPT +
  OR_TOKEN +
  WHEN +
  CALL +
  TERM_PRODUCTION +
  WHEN +
  TAIL +
  UNION_TERMS_PRODUCTION +
  CLOSE +
  RETURN

// TERM_PROGRAM reads one term. The term node names the type, because the tilde was already taken.
const TERM_PROGRAM = ACCEPT +
  TILDE_TOKEN +
  UNLESS +
  TAIL +
  TYPE_PRODUCTION +
  OPEN +
  TERM_NODE +
  CALL +
  TYPE_PRODUCTION +
  CLOSE +
  RETURN

// SIGNATURE_PROGRAM reads the parameters and the results. A result that stands as one type opens
// a node of its own, and a token that closes the signature leaves no result at all.
const SIGNATURE_PROGRAM = CALL +
  PARAMETERS_PRODUCTION +
  AT +
  PARENTHESIS_LEFT_TOKEN +
  WHEN +
  TAIL +
  RESULTS_PRODUCTION +
  AT +
  SEMICOLON_TOKEN +
  OR_AT +
  BRACE_LEFT_TOKEN +
  OR_AT +
  COMMA_TOKEN +
  OR_AT +
  STRING_TOKEN +
  OR_AT +
  PARENTHESIS_RIGHT_TOKEN +
  OR_AT +
  BRACKET_RIGHT_TOKEN +
  OR_AT +
  BRACE_RIGHT_TOKEN +
  OR_AT +
  COMMENT_TOKEN +
  OR_AT +
  END_OF_FILE_TOKEN +
  OR_AT +
  COLON_TOKEN +
  OR_AT +
  ASSIGN_TOKEN +
  WHEN +
  RETURN +
  OPEN +
  RESULT_NODE +
  CALL +
  TYPE_PRODUCTION +
  CLOSE +
  RETURN

// PARAMETERS_PROGRAM steps past the parenthesis that opens a parameter group.
const PARAMETERS_PROGRAM = ACCEPT +
  PARENTHESIS_LEFT_TOKEN +
  UNLESS +
  REJECT +
  SYNTAX_CAUSE +
  TAIL +
  PARAMETER_ITEMS_PRODUCTION

// PARAMETER_ITEMS_PROGRAM reads each parameter up to the closing parenthesis.
const PARAMETER_ITEMS_PROGRAM = SKIP_COMMENTS +
  ACCEPT +
  PARENTHESIS_RIGHT_TOKEN +
  WHEN +
  RETURN +
  CALL +
  PARAMETER_PRODUCTION +
  ACCEPT +
  COMMA_TOKEN +
  TAIL +
  PARAMETER_ITEMS_PRODUCTION

// RESULTS_PROGRAM steps past the parenthesis that opens a result group.
const RESULTS_PROGRAM = ACCEPT +
  PARENTHESIS_LEFT_TOKEN +
  UNLESS +
  REJECT +
  SYNTAX_CAUSE +
  TAIL +
  RESULT_ITEMS_PRODUCTION

// RESULT_ITEMS_PROGRAM reads each result up to the closing parenthesis.
const RESULT_ITEMS_PROGRAM = SKIP_COMMENTS +
  ACCEPT +
  PARENTHESIS_RIGHT_TOKEN +
  WHEN +
  RETURN +
  CALL +
  RESULT_PARAMETER_PRODUCTION +
  ACCEPT +
  COMMA_TOKEN +
  TAIL +
  RESULT_ITEMS_PRODUCTION

// PARAMETER_PROGRAM opens one parameter.
const PARAMETER_PROGRAM = OPEN + PARAMETER_NODE + TAIL + PARAMETER_BODY_PRODUCTION

// RESULT_PARAMETER_PROGRAM opens one result.
const RESULT_PARAMETER_PROGRAM = OPEN + RESULT_NODE + TAIL + PARAMETER_BODY_PRODUCTION

// PARAMETER_BODY_PROGRAM reads the names ahead of the type. A name is optional and only its
// follower tells the two forms apart, so an identifier ahead of a type opener is a name and an
// identifier ahead of anything else is the type itself.
const PARAMETER_BODY_PROGRAM = AT +
  IDENTIFIER_TOKEN +
  UNLESS +
  TAIL +
  PARAMETER_TYPE_PRODUCTION +
  AFTER_OPENS_TYPE +
  OR_AFTER +
  COMMA_TOKEN +
  UNLESS +
  TAIL +
  PARAMETER_TYPE_PRODUCTION +
  CALL +
  PARAMETER_NAME_PRODUCTION +
  TAIL +
  PARAMETER_NAMES_PRODUCTION

// PARAMETER_NAMES_PROGRAM reads every name behind a comma, and stops at a comma a qualified or a
// generic type stands behind.
const PARAMETER_NAMES_PROGRAM = AT +
  COMMA_TOKEN +
  UNLESS +
  TAIL +
  PARAMETER_TYPE_PRODUCTION +
  AFTER +
  IDENTIFIER_TOKEN +
  UNLESS +
  TAIL +
  PARAMETER_TYPE_PRODUCTION +
  QUALIFIES +
  WHEN +
  TAIL +
  PARAMETER_TYPE_PRODUCTION +
  ADVANCE +
  NAME +
  PARAMETER_NAME_NODE +
  TAIL +
  PARAMETER_NAMES_PRODUCTION

// PARAMETER_TYPE_PROGRAM reads the type behind the names. A run of bare names with no type behind
// it was a run of types all along, and only the token after the final name tells the two apart.
const PARAMETER_TYPE_PROGRAM = AT +
  COMMA_TOKEN +
  OR_AT +
  PARENTHESIS_RIGHT_TOKEN +
  WHEN +
  UNNAME +
  WHEN +
  CLOSE +
  WHEN +
  RETURN +
  CALL +
  TYPE_PRODUCTION +
  CLOSE +
  RETURN

// PARAMETER_NAME_PROGRAM takes one name a signature binds.
const PARAMETER_NAME_PROGRAM = NAME + PARAMETER_NAME_NODE + RETURN

// TYPE_PROGRAM hands one type to the rule its first token names.
const TYPE_PROGRAM = OPENS_TYPE +
  UNLESS +
  REJECT +
  SYNTAX_CAUSE +
  AT +
  IDENTIFIER_TOKEN +
  WHEN +
  TAIL +
  TYPE_NAME_PRODUCTION +
  AT +
  STAR_TOKEN +
  WHEN +
  TAIL +
  POINTERS_PRODUCTION +
  AT +
  ELLIPSIS_TOKEN +
  WHEN +
  OPEN +
  ELLIPSIS_NODE +
  WHEN +
  TAIL +
  PREFIX_TYPE_PRODUCTION +
  AT +
  BRACKET_LEFT_TOKEN +
  WHEN +
  TAIL +
  ARRAY_TYPE_PRODUCTION +
  AT +
  MAP_TOKEN +
  WHEN +
  TAIL +
  MAP_TYPE_PRODUCTION +
  AT +
  CHANNEL_TOKEN +
  WHEN +
  TAIL +
  CHANNEL_TYPE_PRODUCTION +
  AT +
  ARROW_TOKEN +
  WHEN +
  OPEN +
  CHANNEL_RECEIVE_NODE +
  WHEN +
  TAIL +
  PREFIX_TYPE_PRODUCTION +
  AT +
  FUNCTION_TOKEN +
  WHEN +
  TAIL +
  FUNCTION_TYPE_PRODUCTION +
  AT +
  STRUCTURE_TOKEN +
  WHEN +
  TAIL +
  STRUCTURE_TYPE_PRODUCTION +
  AT +
  INTERFACE_TOKEN +
  WHEN +
  TAIL +
  INTERFACE_TYPE_PRODUCTION +
  AT +
  PARENTHESIS_LEFT_TOKEN +
  WHEN +
  TAIL +
  GROUPED_TYPE_PRODUCTION +
  REJECT +
  SYNTAX_CAUSE

// POINTERS_PROGRAM reads one run of pointer types in one step, the type they point at, and closes
// the run. A long run of stars is how a file spends its type storage, thus it costs one step
// rather than one rule for each star.
const POINTERS_PROGRAM = STARS + CALL + TYPE_PRODUCTION + CLOSE_STARS + RETURN

// PREFIX_TYPE_PROGRAM reads the type behind the token that named the form.
const PREFIX_TYPE_PROGRAM = ADVANCE +
  ACCEPT +
  CHANNEL_TOKEN +
  CALL +
  TYPE_PRODUCTION +
  CLOSE +
  RETURN

// GROUPED_TYPE_PROGRAM reads one parenthesized type, which opens no node.
const GROUPED_TYPE_PROGRAM = ADVANCE +
  CALL +
  TYPE_PRODUCTION +
  ACCEPT +
  PARENTHESIS_RIGHT_TOKEN +
  UNLESS +
  REJECT +
  SYNTAX_CAUSE +
  RETURN

// TYPE_NAME_PROGRAM reads one named type, its qualifier, and its type arguments.
const TYPE_NAME_PROGRAM = CALL +
  NAME_PRODUCTION +
  AT +
  PERIOD_TOKEN +
  WHEN +
  CALL +
  QUALIFIED_NAME_PRODUCTION +
  AT +
  BRACKET_LEFT_TOKEN +
  UNLESS +
  RETURN +
  WRAP +
  GENERIC_NODE +
  ADVANCE +
  TAIL +
  GENERIC_ARGUMENTS_PRODUCTION

// QUALIFIED_NAME_PROGRAM takes back the qualifier and reads the name behind it.
const QUALIFIED_NAME_PROGRAM = WRAP +
  SELECTOR_NODE +
  ADVANCE +
  CALL +
  NAME_PRODUCTION +
  CLOSE +
  RETURN

// GENERIC_ARGUMENTS_PROGRAM reads each type argument up to the closing bracket.
const GENERIC_ARGUMENTS_PROGRAM = ACCEPT +
  BRACKET_RIGHT_TOKEN +
  WHEN +
  CLOSE +
  WHEN +
  RETURN +
  CALL +
  TYPE_PRODUCTION +
  ACCEPT +
  COMMA_TOKEN +
  TAIL +
  GENERIC_ARGUMENTS_PRODUCTION

// ARRAY_TYPE_PROGRAM reads one array or slice type. The bracket pair alone says slice, and anything
// inside it says array, thus the kind is settled only after the capacity is read.
const ARRAY_TYPE_PROGRAM = OPEN +
  SLICE_TYPE_NODE +
  ADVANCE +
  ACCEPT +
  BRACKET_RIGHT_TOKEN +
  WHEN +
  TAIL +
  ELEMENT_TYPE_PRODUCTION +
  RETYPE +
  ARRAY_TYPE_NODE +
  AT +
  ELLIPSIS_TOKEN +
  WHEN +
  TAKE +
  ELLIPSIS_NODE +
  UNLESS +
  CALL +
  BINARY_PRODUCTION +
  ACCEPT +
  BRACKET_RIGHT_TOKEN +
  UNLESS +
  REJECT +
  SYNTAX_CAUSE +
  TAIL +
  ELEMENT_TYPE_PRODUCTION

// ELEMENT_TYPE_PROGRAM reads the type a composite type holds and closes that type.
const ELEMENT_TYPE_PROGRAM = CALL + TYPE_PRODUCTION + CLOSE + RETURN

// CHANNEL_TYPE_PROGRAM reads one channel type. An arrow after the keyword makes it send-only.
const CHANNEL_TYPE_PROGRAM = OPEN +
  CHANNEL_TYPE_NODE +
  ADVANCE +
  ACCEPT +
  ARROW_TOKEN +
  WHEN +
  RETYPE +
  CHANNEL_SEND_NODE +
  TAIL +
  ELEMENT_TYPE_PRODUCTION

// MAP_TYPE_PROGRAM reads one map type, its key, and its value.
const MAP_TYPE_PROGRAM = OPEN +
  MAP_TYPE_NODE +
  ADVANCE +
  ACCEPT +
  BRACKET_LEFT_TOKEN +
  UNLESS +
  REJECT +
  SYNTAX_CAUSE +
  CALL +
  TYPE_PRODUCTION +
  ACCEPT +
  BRACKET_RIGHT_TOKEN +
  UNLESS +
  REJECT +
  SYNTAX_CAUSE +
  TAIL +
  ELEMENT_TYPE_PRODUCTION

// FUNCTION_TYPE_PROGRAM reads one function type and its signature.
const FUNCTION_TYPE_PROGRAM = OPEN +
  FUNCTION_TYPE_NODE +
  ADVANCE +
  CALL +
  SIGNATURE_PRODUCTION +
  CLOSE +
  RETURN

// STRUCTURE_TYPE_PROGRAM steps past the brace that opens a struct type.
const STRUCTURE_TYPE_PROGRAM = OPEN +
  STRUCTURE_TYPE_NODE +
  ADVANCE +
  ACCEPT +
  BRACE_LEFT_TOKEN +
  UNLESS +
  REJECT +
  SYNTAX_CAUSE +
  TAIL +
  FIELDS_PRODUCTION

// FIELDS_PROGRAM reads each field up to the closing brace.
const FIELDS_PROGRAM = SKIP_EMPTY +
  ACCEPT +
  BRACE_RIGHT_TOKEN +
  WHEN +
  CLOSE +
  WHEN +
  RETURN +
  CALL +
  FIELD_PRODUCTION +
  TAIL +
  FIELDS_PRODUCTION

// FIELD_PROGRAM reads one struct field. An identifier ahead of a type opener is a field name, and
// an identifier ahead of anything else is an embedded type.
const FIELD_PROGRAM = OPEN +
  FIELD_NODE +
  FIELD_NAMED +
  UNLESS +
  TAIL +
  FIELD_TYPE_PRODUCTION +
  NAMES +
  FIELD_NAME_NODE +
  TAIL +
  FIELD_TYPE_PRODUCTION

// FIELD_TYPE_PROGRAM reads the type and the optional tag, and closes the field.
const FIELD_TYPE_PROGRAM = CALL +
  TYPE_PRODUCTION +
  AT +
  STRING_TOKEN +
  WHEN +
  TAKE +
  STRING_NODE +
  CLOSE +
  TAIL +
  END_LINE_PRODUCTION

// INTERFACE_TYPE_PROGRAM steps past the brace that opens an interface type.
const INTERFACE_TYPE_PROGRAM = OPEN +
  INTERFACE_TYPE_NODE +
  ADVANCE +
  ACCEPT +
  BRACE_LEFT_TOKEN +
  UNLESS +
  REJECT +
  SYNTAX_CAUSE +
  TAIL +
  ELEMENTS_PRODUCTION

// ELEMENTS_PROGRAM reads each element up to the closing brace. The linter bans a method set, thus
// every element is a constraint.
const ELEMENTS_PROGRAM = SKIP_EMPTY +
  ACCEPT +
  BRACE_RIGHT_TOKEN +
  WHEN +
  CLOSE +
  WHEN +
  RETURN +
  AT +
  IDENTIFIER_TOKEN +
  AND_AFTER +
  PARENTHESIS_LEFT_TOKEN +
  WHEN +
  REJECT +
  INTERFACE_METHOD_CAUSE +
  CALL +
  CONSTRAINT_PRODUCTION +
  CALL +
  END_LINE_PRODUCTION +
  TAIL +
  ELEMENTS_PRODUCTION

// BLOCK_PROGRAM steps past the brace that opens a block. The brace rule of an enclosing control
// clause stops at the block, because a literal inside it is unambiguous again.
const BLOCK_PROGRAM = OPEN +
  BLOCK_NODE +
  ACCEPT +
  BRACE_LEFT_TOKEN +
  UNLESS +
  REJECT +
  SYNTAX_CAUSE +
  SAVE_PLAIN +
  CLEAR_PLAIN +
  TAIL +
  STATEMENTS_PRODUCTION

// STATEMENTS_PROGRAM reads each statement up to the closing brace. An empty statement writes no
// node, because it carries nothing a reader of the tree could want.
const STATEMENTS_PROGRAM = SKIP_EMPTY +
  ACCEPT +
  BRACE_RIGHT_TOKEN +
  WHEN +
  RESTORE_PLAIN +
  WHEN +
  CLOSE +
  WHEN +
  RETURN +
  CALL +
  STATEMENT_PRODUCTION +
  TAIL +
  STATEMENTS_PRODUCTION

// STATEMENT_PROGRAM hands one statement to the rule its first token names. A statement most
// often opens with a name, thus the rule asks that first.
const STATEMENT_PROGRAM = AT +
  IDENTIFIER_TOKEN +
  AND_AFTER +
  COLON_TOKEN +
  WHEN +
  TAIL +
  LABEL_PRODUCTION +
  AT +
  IDENTIFIER_TOKEN +
  WHEN +
  CALL +
  CLAUSE_PRODUCTION +
  WHEN +
  TAIL +
  END_LINE_PRODUCTION +
  AT +
  CONSTANT_TOKEN +
  OR_AT +
  VARIABLE_TOKEN +
  OR_AT +
  TYPE_TOKEN +
  WHEN +
  TAIL +
  DECLARATION_STATEMENT_PRODUCTION +
  AT +
  RETURN_TOKEN +
  WHEN +
  TAIL +
  RETURN_STATEMENT_PRODUCTION +
  AT +
  IF_TOKEN +
  WHEN +
  TAIL +
  IF_STATEMENT_PRODUCTION +
  AT +
  FOR_TOKEN +
  WHEN +
  TAIL +
  FOR_STATEMENT_PRODUCTION +
  AT +
  SWITCH_TOKEN +
  WHEN +
  TAIL +
  SWITCH_STATEMENT_PRODUCTION +
  AT +
  SELECT_TOKEN +
  WHEN +
  TAIL +
  SELECT_STATEMENT_PRODUCTION +
  AT +
  GO_TOKEN +
  WHEN +
  OPEN +
  GO_NODE +
  WHEN +
  TAIL +
  LAUNCH_PRODUCTION +
  AT +
  DEFER_TOKEN +
  WHEN +
  OPEN +
  DEFER_NODE +
  WHEN +
  TAIL +
  LAUNCH_PRODUCTION +
  AT +
  BREAK_TOKEN +
  WHEN +
  OPEN +
  BREAK_NODE +
  WHEN +
  TAIL +
  JUMP_PRODUCTION +
  AT +
  CONTINUE_TOKEN +
  WHEN +
  OPEN +
  CONTINUE_NODE +
  WHEN +
  TAIL +
  JUMP_PRODUCTION +
  AT +
  GOTO_TOKEN +
  WHEN +
  OPEN +
  GOTO_NODE +
  WHEN +
  TAIL +
  JUMP_PRODUCTION +
  AT +
  FALLTHROUGH_TOKEN +
  WHEN +
  TAKE +
  FALLTHROUGH_NODE +
  WHEN +
  TAIL +
  END_LINE_PRODUCTION +
  AT +
  BRACE_LEFT_TOKEN +
  WHEN +
  CALL +
  BLOCK_PRODUCTION +
  WHEN +
  TAIL +
  END_LINE_PRODUCTION +
  CALL +
  CLAUSE_PRODUCTION +
  TAIL +
  END_LINE_PRODUCTION

// DECLARATION_STATEMENT_PROGRAM reads one const, var, or type declaration standing inside a body.
const DECLARATION_STATEMENT_PROGRAM = OPEN +
  DECLARATION_STATEMENT_NODE +
  AT +
  TYPE_TOKEN +
  WHEN +
  CALL +
  TYPE_DECLARATION_PRODUCTION +
  UNLESS +
  CALL +
  VALUE_DECLARATION_PRODUCTION +
  CLOSE +
  RETURN

// LABEL_PROGRAM reads one labelled statement. A label may stand at the end of a block with nothing
// behind it, which Go reads as a labelled empty statement.
const LABEL_PROGRAM = OPEN +
  LABEL_NODE +
  CALL +
  NAME_PRODUCTION +
  ACCEPT +
  COLON_TOKEN +
  AT +
  BRACE_RIGHT_TOKEN +
  OR_AT +
  END_OF_FILE_TOKEN +
  OR_AT +
  CASE_TOKEN +
  OR_AT +
  DEFAULT_TOKEN +
  WHEN +
  CLOSE +
  WHEN +
  RETURN +
  CALL +
  STATEMENT_PRODUCTION +
  CLOSE +
  RETURN

// RETURN_STATEMENT_PROGRAM reads one return statement. A return that names no value parses and
// holds no child, because the canonical form writes the values the signature names.
const RETURN_STATEMENT_PROGRAM = OPEN +
  RETURN_NODE +
  ADVANCE +
  AT +
  SEMICOLON_TOKEN +
  OR_AT +
  BRACE_RIGHT_TOKEN +
  OR_AT +
  END_OF_FILE_TOKEN +
  OR_AT +
  COMMENT_TOKEN +
  UNLESS +
  CALL +
  EXPRESSION_LIST_PRODUCTION +
  CLOSE +
  TAIL +
  END_LINE_PRODUCTION

// LAUNCH_PROGRAM reads the call one go or defer statement carries.
const LAUNCH_PROGRAM = ADVANCE + CALL + BINARY_PRODUCTION + CLOSE + TAIL + END_LINE_PRODUCTION

// JUMP_PROGRAM reads the optional label one jump carries.
const JUMP_PROGRAM = ADVANCE +
  AT +
  IDENTIFIER_TOKEN +
  WHEN +
  CALL +
  NAME_PRODUCTION +
  CLOSE +
  TAIL +
  END_LINE_PRODUCTION

// IF_STATEMENT_PROGRAM reads one if statement, its optional initializer, its condition, and its
// branches. An else if closes with the if it holds.
const IF_STATEMENT_PROGRAM = OPEN +
  IF_NODE +
  ADVANCE +
  SAVE_PLAIN +
  SET_PLAIN +
  CALL +
  CLAUSE_PRODUCTION +
  ACCEPT +
  SEMICOLON_TOKEN +
  WHEN +
  CALL +
  CLAUSE_PRODUCTION +
  RESTORE_PLAIN +
  CALL +
  BLOCK_PRODUCTION +
  ACCEPT +
  ELSE_TOKEN +
  UNLESS +
  TAIL +
  IF_END_PRODUCTION +
  AT +
  IF_TOKEN +
  WHEN +
  CALL +
  IF_STATEMENT_PRODUCTION +
  WHEN +
  CLOSE +
  WHEN +
  RETURN +
  CALL +
  BLOCK_PRODUCTION +
  TAIL +
  IF_END_PRODUCTION

// IF_END_PROGRAM closes one if statement.
const IF_END_PROGRAM = CLOSE + TAIL + END_LINE_PRODUCTION

// FOR_STATEMENT_PROGRAM reads one for statement: a bare loop, a condition, a three-clause head, or
// a range clause.
const FOR_STATEMENT_PROGRAM = OPEN +
  FOR_NODE +
  ADVANCE +
  SAVE_PLAIN +
  SET_PLAIN +
  AT +
  BRACE_LEFT_TOKEN +
  UNLESS +
  CALL +
  FOR_HEAD_PRODUCTION +
  RESTORE_PLAIN +
  CALL +
  BLOCK_PRODUCTION +
  CLOSE +
  TAIL +
  END_LINE_PRODUCTION

// FOR_HEAD_PROGRAM reads the head of one for statement.
const FOR_HEAD_PROGRAM = AT +
  RANGE_TOKEN +
  WHEN +
  TAIL +
  RANGE_CLAUSE_PRODUCTION +
  AT +
  SEMICOLON_TOKEN +
  UNLESS +
  CALL +
  CLAUSE_PRODUCTION +
  ACCEPT +
  SEMICOLON_TOKEN +
  UNLESS +
  RETURN +
  AT +
  SEMICOLON_TOKEN +
  UNLESS +
  CALL +
  CLAUSE_PRODUCTION +
  ACCEPT +
  SEMICOLON_TOKEN +
  UNLESS +
  RETURN +
  AT +
  BRACE_LEFT_TOKEN +
  UNLESS +
  CALL +
  CLAUSE_PRODUCTION +
  RETURN

// RANGE_CLAUSE_PROGRAM reads one range clause.
const RANGE_CLAUSE_PROGRAM = OPEN + RANGE_NODE + ADVANCE +
  CALL + BINARY_PRODUCTION + CLOSE + RETURN

// SWITCH_STATEMENT_PROGRAM reads one expression or type switch. A type assertion over the type
// keyword in the head is the one mark that tells the two apart, so the kind is settled after the
// head is read.
const SWITCH_STATEMENT_PROGRAM = OPEN +
  SWITCH_NODE +
  ADVANCE +
  SAVE_PLAIN +
  SET_PLAIN +
  CLEAR_ASSERTION +
  AT +
  BRACE_LEFT_TOKEN +
  UNLESS +
  CALL +
  SWITCH_HEAD_PRODUCTION +
  ASSERTED +
  WHEN +
  RETYPE +
  TYPE_SWITCH_NODE +
  RESTORE_PLAIN +
  CALL +
  CASE_BLOCK_PRODUCTION +
  CLOSE +
  TAIL +
  END_LINE_PRODUCTION

// SWITCH_HEAD_PROGRAM reads the initializer and the tag of one switch.
const SWITCH_HEAD_PROGRAM = CALL +
  CLAUSE_PRODUCTION +
  ACCEPT +
  SEMICOLON_TOKEN +
  UNLESS +
  RETURN +
  AT +
  BRACE_LEFT_TOKEN +
  UNLESS +
  CALL +
  CLAUSE_PRODUCTION +
  RETURN

// SELECT_STATEMENT_PROGRAM reads one select statement.
const SELECT_STATEMENT_PROGRAM = OPEN +
  SELECT_NODE +
  ADVANCE +
  CALL +
  CASE_BLOCK_PRODUCTION +
  CLOSE +
  TAIL +
  END_LINE_PRODUCTION

// CASE_BLOCK_PROGRAM steps past the brace that opens a clause list.
const CASE_BLOCK_PROGRAM = ACCEPT +
  BRACE_LEFT_TOKEN +
  UNLESS +
  REJECT +
  SYNTAX_CAUSE +
  TAIL +
  CASES_PRODUCTION

// CASES_PROGRAM reads each clause up to the closing brace.
const CASES_PROGRAM = SKIP_COMMENTS +
  ACCEPT +
  BRACE_RIGHT_TOKEN +
  WHEN +
  RETURN +
  CALL +
  CASE_PRODUCTION +
  TAIL +
  CASES_PRODUCTION

// CASE_PROGRAM opens one case or default clause. The default node stands at the token behind its
// keyword, because the keyword is taken before the node opens.
const CASE_PROGRAM = ACCEPT +
  DEFAULT_TOKEN +
  WHEN +
  OPEN +
  DEFAULT_NODE +
  WHEN +
  TAIL +
  CASE_COLON_PRODUCTION +
  OPEN +
  CASE_NODE +
  ACCEPT +
  CASE_TOKEN +
  UNLESS +
  REJECT +
  SYNTAX_CAUSE +
  CALL +
  CLAUSE_PRODUCTION +
  TAIL +
  CASE_COLON_PRODUCTION

// CASE_COLON_PROGRAM reads the colon and the statements of one clause, and closes it.
const CASE_COLON_PROGRAM = ACCEPT +
  COLON_TOKEN +
  UNLESS +
  REJECT +
  SYNTAX_CAUSE +
  CALL +
  CASE_BODY_PRODUCTION +
  CLOSE +
  RETURN

// CASE_BODY_PROGRAM reads the statements of one clause, which run to the next clause or the
// closing brace.
const CASE_BODY_PROGRAM = SKIP_EMPTY +
  AT +
  CASE_TOKEN +
  OR_AT +
  DEFAULT_TOKEN +
  OR_AT +
  BRACE_RIGHT_TOKEN +
  OR_AT +
  END_OF_FILE_TOKEN +
  WHEN +
  RETURN +
  CALL +
  STATEMENT_PRODUCTION +
  TAIL +
  CASE_BODY_PRODUCTION

// CLAUSE_PROGRAM reads one simple statement. The operator that follows the left side names the
// form, thus the kind is settled after that side is read.
const CLAUSE_PROGRAM = OPEN +
  EXPRESSION_STATEMENT_NODE +
  CALL +
  EXPRESSION_LIST_PRODUCTION +
  CLAUSE_OPERATOR +
  UNLESS +
  CLOSE +
  UNLESS +
  RETURN +
  RETYPE_CLAUSE +
  AT +
  INCREMENT_TOKEN +
  OR_AT +
  DECREMENT_TOKEN +
  WHEN +
  ADVANCE +
  WHEN +
  CLOSE +
  WHEN +
  RETURN +
  AT +
  ARROW_TOKEN +
  WHEN +
  ADVANCE +
  WHEN +
  CALL +
  BINARY_PRODUCTION +
  WHEN +
  CLOSE +
  WHEN +
  RETURN +
  ADVANCE +
  AT +
  RANGE_TOKEN +
  WHEN +
  CALL +
  RANGE_CLAUSE_PRODUCTION +
  UNLESS +
  CALL +
  EXPRESSION_LIST_PRODUCTION +
  CLOSE +
  RETURN

// EXPRESSION_LIST_PROGRAM reads one comma-separated expression list.
const EXPRESSION_LIST_PROGRAM = CALL +
  BINARY_PRODUCTION +
  ACCEPT +
  COMMA_TOKEN +
  WHEN +
  TAIL +
  EXPRESSION_LIST_PRODUCTION +
  RETURN

// BINARY_PROGRAM reads the first operand of one binary expression.
const BINARY_PROGRAM = CALL + UNARY_PRODUCTION + TAIL + BINARY_OPERATORS_PRODUCTION

// BINARY_OPERATORS_PROGRAM reads each operator that binds tighter than the level the rule runs at.
// A sign this dialect states no form for fails where it stands, thus every form that holds an
// operation answers for it and none of them reads it.
const BINARY_OPERATORS_PROGRAM = REFUSES_SIGN +
  WHEN +
  REJECT +
  REFUSED_SIGN_CAUSE +
  ABOVE +
  UNLESS +
  RETURN +
  HOLD +
  WRAP +
  BINARY_NODE +
  ADVANCE +
  CALL_HELD +
  CLOSE +
  TAIL +
  BINARY_OPERATORS_PRODUCTION

// UNARY_PROGRAM reads one unary expression and its operand.
const UNARY_PROGRAM = UNARY_SIGN +
  UNLESS +
  TAIL +
  PRIMARY_PRODUCTION +
  OPEN +
  UNARY_NODE +
  ADVANCE +
  CALL +
  UNARY_PRODUCTION +
  CLOSE +
  RETURN

// PRIMARY_PROGRAM reads one operand and every suffix that binds to it.
const PRIMARY_PROGRAM = CALL + OPERAND_PRODUCTION + TAIL + SUFFIXES_PRODUCTION

// SUFFIXES_PROGRAM reads each suffix that binds to the operand already read. An operand most
// often binds none, thus the rule asks that first.
const SUFFIXES_PROGRAM = SUFFIX_SIGN +
  UNLESS +
  RETURN +
  AT +
  PERIOD_TOKEN +
  WHEN +
  CALL +
  SELECTOR_SUFFIX_PRODUCTION +
  WHEN +
  TAIL +
  SUFFIXES_PRODUCTION +
  AT +
  BRACKET_LEFT_TOKEN +
  WHEN +
  CALL +
  INDEX_SUFFIX_PRODUCTION +
  WHEN +
  TAIL +
  SUFFIXES_PRODUCTION +
  AT +
  PARENTHESIS_LEFT_TOKEN +
  WHEN +
  CALL +
  CALL_SUFFIX_PRODUCTION +
  WHEN +
  TAIL +
  SUFFIXES_PRODUCTION +
  AT +
  BRACE_LEFT_TOKEN +
  UNLESS +
  RETURN +
  BARRED_BRACE +
  WHEN +
  RETURN +
  CALL +
  COMPOSITE_SUFFIX_PRODUCTION +
  TAIL +
  SUFFIXES_PRODUCTION

// OPERAND_PROGRAM reads one operand: a name, a literal, a grouping, a function literal, or a type
// that stands where an expression can, such as the first argument of a conversion or of make.
const OPERAND_PROGRAM = AT +
  IDENTIFIER_TOKEN +
  WHEN +
  TAIL +
  NAME_PRODUCTION +
  AT +
  INTEGER_TOKEN +
  OR_AT +
  FLOAT_TOKEN +
  OR_AT +
  IMAGINARY_TOKEN +
  OR_AT +
  CHARACTER_TOKEN +
  OR_AT +
  STRING_TOKEN +
  WHEN +
  LITERAL +
  WHEN +
  RETURN +
  AT +
  PARENTHESIS_LEFT_TOKEN +
  WHEN +
  TAIL +
  GROUPING_PRODUCTION +
  AT +
  FUNCTION_TOKEN +
  WHEN +
  TAIL +
  FUNCTION_LITERAL_PRODUCTION +
  AT +
  BRACKET_LEFT_TOKEN +
  OR_AT +
  MAP_TOKEN +
  OR_AT +
  CHANNEL_TOKEN +
  OR_AT +
  ARROW_TOKEN +
  OR_AT +
  STRUCTURE_TOKEN +
  OR_AT +
  INTERFACE_TOKEN +
  OR_AT +
  ELLIPSIS_TOKEN +
  WHEN +
  TAIL +
  TYPE_PRODUCTION +
  REJECT +
  SYNTAX_CAUSE

// GROUPING_PROGRAM reads one parenthesized expression. The brace rule of the enclosing control
// clause stops at the parenthesis, because a literal inside a grouping is unambiguous again.
const GROUPING_PROGRAM = OPEN +
  PARENTHESIS_NODE +
  ADVANCE +
  SAVE_PLAIN +
  CLEAR_PLAIN +
  CALL +
  BINARY_PRODUCTION +
  RESTORE_PLAIN +
  ACCEPT +
  PARENTHESIS_RIGHT_TOKEN +
  UNLESS +
  REJECT +
  SYNTAX_CAUSE +
  CLOSE +
  RETURN

// FUNCTION_LITERAL_PROGRAM reads one function literal and its body.
const FUNCTION_LITERAL_PROGRAM = OPEN +
  FUNCTION_LITERAL_NODE +
  ADVANCE +
  CALL +
  SIGNATURE_PRODUCTION +
  AT +
  BRACE_LEFT_TOKEN +
  WHEN +
  SAVE_PLAIN +
  WHEN +
  CLEAR_PLAIN +
  WHEN +
  CALL +
  BLOCK_PRODUCTION +
  WHEN +
  RESTORE_PLAIN +
  CLOSE +
  RETURN

// SELECTOR_SUFFIX_PROGRAM reads one selector over the expression already read.
const SELECTOR_SUFFIX_PROGRAM = AFTER +
  PARENTHESIS_LEFT_TOKEN +
  WHEN +
  TAIL +
  ASSERTION_SUFFIX_PRODUCTION +
  WRAP +
  SELECTOR_NODE +
  ADVANCE +
  CALL +
  NAME_PRODUCTION +
  CLOSE +
  RETURN

// ASSERTION_SUFFIX_PROGRAM reads one type assertion over the expression already read.
const ASSERTION_SUFFIX_PROGRAM = WRAP +
  ASSERTION_NODE +
  ADVANCE +
  ADVANCE +
  ACCEPT +
  TYPE_TOKEN +
  WHEN +
  SET_ASSERTION +
  UNLESS +
  CALL +
  TYPE_PRODUCTION +
  ACCEPT +
  PARENTHESIS_RIGHT_TOKEN +
  UNLESS +
  REJECT +
  SYNTAX_CAUSE +
  CLOSE +
  RETURN

// INDEX_SUFFIX_PROGRAM reads one index, slice, or instantiation over the expression already read.
const INDEX_SUFFIX_PROGRAM = WRAP_SUFFIX +
  ADVANCE +
  SAVE_PLAIN +
  CLEAR_PLAIN +
  TAIL +
  INDEX_ITEMS_PRODUCTION

// INDEX_ITEMS_PROGRAM reads each bound up to the closing bracket.
const INDEX_ITEMS_PROGRAM = ACCEPT +
  BRACKET_RIGHT_TOKEN +
  WHEN +
  RESTORE_PLAIN +
  WHEN +
  CLOSE +
  WHEN +
  RETURN +
  ACCEPT +
  COLON_TOKEN +
  WHEN +
  TAIL +
  INDEX_ITEMS_PRODUCTION +
  ACCEPT +
  COMMA_TOKEN +
  WHEN +
  TAIL +
  INDEX_ITEMS_PRODUCTION +
  CALL +
  BINARY_PRODUCTION +
  TAIL +
  INDEX_ITEMS_PRODUCTION

// CALL_SUFFIX_PROGRAM reads one call over the expression already read.
const CALL_SUFFIX_PROGRAM = WRAP +
  CALL_NODE +
  ADVANCE +
  SAVE_PLAIN +
  CLEAR_PLAIN +
  TAIL +
  ARGUMENTS_PRODUCTION

// ARGUMENTS_PROGRAM reads each argument up to the closing parenthesis.
const ARGUMENTS_PROGRAM = SKIP_COMMENTS +
  ACCEPT +
  PARENTHESIS_RIGHT_TOKEN +
  WHEN +
  RESTORE_PLAIN +
  WHEN +
  CLOSE +
  WHEN +
  RETURN +
  CALL +
  BINARY_PRODUCTION +
  ACCEPT +
  ELLIPSIS_TOKEN +
  WHEN +
  LEAF +
  ELLIPSIS_NODE +
  ACCEPT +
  COMMA_TOKEN +
  ACCEPT +
  SEMICOLON_TOKEN +
  TAIL +
  ARGUMENTS_PRODUCTION

// COMPOSITE_SUFFIX_PROGRAM reads one composite literal over the type already read.
const COMPOSITE_SUFFIX_PROGRAM = WRAP +
  COMPOSITE_NODE +
  CALL +
  COMPOSITE_BODY_PRODUCTION +
  CLOSE +
  RETURN

// COMPOSITE_BODY_PROGRAM steps past the brace that opens an element list.
const COMPOSITE_BODY_PROGRAM = ACCEPT +
  BRACE_LEFT_TOKEN +
  UNLESS +
  REJECT +
  SYNTAX_CAUSE +
  SAVE_PLAIN +
  CLEAR_PLAIN +
  TAIL +
  COMPOSITE_ITEMS_PRODUCTION

// COMPOSITE_ITEMS_PROGRAM reads each element up to the closing brace.
const COMPOSITE_ITEMS_PROGRAM = SKIP_COMMENTS +
  ACCEPT +
  BRACE_RIGHT_TOKEN +
  WHEN +
  RESTORE_PLAIN +
  WHEN +
  RETURN +
  CALL +
  ELEMENT_PRODUCTION +
  ACCEPT +
  COMMA_TOKEN +
  ACCEPT +
  SEMICOLON_TOKEN +
  TAIL +
  COMPOSITE_ITEMS_PRODUCTION

// ELEMENT_PROGRAM reads one element and the value a colon keys to it.
const ELEMENT_PROGRAM = CALL +
  ELEMENT_VALUE_PRODUCTION +
  AT +
  COLON_TOKEN +
  UNLESS +
  RETURN +
  WRAP +
  KEY_VALUE_NODE +
  ADVANCE +
  CALL +
  ELEMENT_VALUE_PRODUCTION +
  CLOSE +
  RETURN

// ELEMENT_VALUE_PROGRAM reads one element value. An element may leave its type out, which is why a
// brace can open an element of its own.
const ELEMENT_VALUE_PROGRAM = AT +
  BRACE_LEFT_TOKEN +
  UNLESS +
  TAIL +
  BINARY_PRODUCTION +
  OPEN +
  COMPOSITE_NODE +
  CALL +
  COMPOSITE_BODY_PRODUCTION +
  CLOSE +
  RETURN

// Names a case that runs on into the case behind it. A case closes where its statements close,
// thus a reader of one case reads the whole of what it runs, and a word that runs on states a
// flow that reader never sees.
func name_fallthrough(subject Syntax_Tree_Handle, tree Parse_Result) (named Parse_Result) {
  defer func() { Parse_Result_Invariants(&named, "name_fallthrough.named") }()
  Syntax_Tree_Handle_Invariants(subject, "name_fallthrough.subject")
  Parse_Result_Invariants(&tree, "name_fallthrough.tree")
  if tree.Failure != FAILURE_NONE {
    return tree
  }
  count := tree.Node_Count
  for slot := INDEX_FIRST; slot < Index(count); slot++ {
    if subject.Nodes.Values[slot].Kind != NODE_FALLTHROUGH {
      continue
    }
    tree.Failure = FAILURE_FALLTHROUGH
    tree.OK = false
    tree.Failure_Token = Token_Index(subject.Nodes.Values[slot].Token)
    return tree
  }
  return tree
}
