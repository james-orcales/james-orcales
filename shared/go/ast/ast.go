// Package ast parses Go source into a node arena. The caller owns one Parse_State and every
// node is a slot in caller storage, thus a parse allocates nothing. The grammar it accepts is the
// subset this repository's linter permits, and it descends by recursion under a depth bound so
// a hostile nesting depth ends the parse instead of the goroutine stack.
package ast

import (
	"local/james-orcales/shared/go/token"
	"local/james-orcales/shared/sim/aver/default"
)

// TOKEN_COUNT_MAXIMUM caps the token run. The densest first-party file in this repository
// spends 50,875 tokens, so this is two and a half times the largest run measured, and a source
// past it fails the parse rather than writing outside the array.
const TOKEN_COUNT_MAXIMUM = 131072

// NODE_COUNT_MAXIMUM caps the arena. It stands below the token bound because a file holds its
// package clause and its end before any node, thus the arena can never outgrow the run.
const NODE_COUNT_MAXIMUM = TOKEN_COUNT_MAXIMUM - 8

// TOKEN_COUNT_MAXIMUM caps the token run. It matches the arena bound, because a parse writes
// at least one node for each token that is not a separator and no source can outrun both.

// DEPTH_MAXIMUM caps how deep the grammar may nest. Recursion carries the descent, so an
// unbounded depth would end the process on the goroutine stack rather than in a diagnostic.
const DEPTH_MAXIMUM = 256

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

// CURSOR_TOKEN_COUNT holds how many tokens the scan wrote.
const CURSOR_TOKEN_COUNT = 0

// CURSOR_POSITION holds the token the next parse step reads.
const CURSOR_POSITION = 1

// CURSOR_FAILURE holds the token the parse refused.
const CURSOR_FAILURE = 2

// CURSOR_BLANK holds one past the final token whose run of empty lines the tree already holds,
// so a step that is asked twice takes that run one time.
const CURSOR_BLANK = 3

// TOKEN_CURSOR_COUNT is the slot count of the token cursor array.
const TOKEN_CURSOR_COUNT = 4

// CURSOR_NODE_COUNT holds how many nodes the parse wrote, counting the absent slot.
const CURSOR_NODE_COUNT = 0

// CURSOR_DEPTH holds how many parents stand open.
const CURSOR_DEPTH = 1

// NODE_CURSOR_COUNT is the slot count of the node cursor array.
const NODE_CURSOR_COUNT = 2

// FLAG_FAILED holds whether the parse met an error, an exhausted arena, or a depth overflow.
const FLAG_FAILED = 0

// FLAG_PLAIN_BRACE holds whether an open brace starts a block rather than a composite literal.
// A control clause sets it, because `for x {` opens a body and never a literal of type x.
const FLAG_PLAIN_BRACE = 1

// FLAG_TYPE_ASSERTION holds whether the clause just parsed held a type assertion over the type
// keyword, which is the one mark that tells a type switch from an expression switch.
const FLAG_TYPE_ASSERTION = 2

// FLAG_COUNT is the slot count of the flag array.
const FLAG_COUNT = 3

// PRECEDENCE_NONE marks a token that binds no binary expression.
const PRECEDENCE_NONE Precedence = 0

// PRECEDENCE_COMPARISON is the level of the four comparison operators this dialect states.
const PRECEDENCE_COMPARISON Precedence = 1

// PRECEDENCE_ADDITION is the level of addition, subtraction, and the two or operators.
const PRECEDENCE_ADDITION Precedence = 2

// PRECEDENCE_MULTIPLICATION is the level of the products, the shifts, and the two and operators.
const PRECEDENCE_MULTIPLICATION Precedence = 3

// PRECEDENCE_MINIMUM binds no binary expression.
const PRECEDENCE_MINIMUM = int(PRECEDENCE_NONE)

// PRECEDENCE_MAXIMUM is the tightest binary level.
const PRECEDENCE_MAXIMUM = int(PRECEDENCE_MULTIPLICATION)

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

// CAUSE_SLOT holds the one cause a parse records.
const CAUSE_SLOT = 0

// CAUSE_SLOT_COUNT is the slot count of the cause array.
const CAUSE_SLOT_COUNT = 1

// FAILURE_NONE marks a parse that met no failure.
const FAILURE_NONE Failure_Code = 0

// FAILURE_SYNTAX marks a token that fits no rule of the grammar.
const FAILURE_SYNTAX Failure_Code = 1

// FAILURE_DECLARATION_GROUP marks a parenthesized const, var, or type group, which this
// repository bans and which the grammar therefore refuses rather than builds.
const FAILURE_DECLARATION_GROUP Failure_Code = 2

// FAILURE_INTERFACE_METHOD marks an interface that holds a method. The linter bans the
// interface, not the method: a method declaration parses and the linter judges it on its own.
const FAILURE_INTERFACE_METHOD Failure_Code = 3

// FAILURE_WIDE_IDENTIFIER marks a name holding a byte at or above 128.
const FAILURE_WIDE_IDENTIFIER Failure_Code = 4

// FAILURE_IOTA marks the iota identifier, which ties a constant value to declaration order.
const FAILURE_IOTA Failure_Code = 5

// FAILURE_PRIVATE_FIELD marks a struct field name that opens with a lowercase letter.
const FAILURE_PRIVATE_FIELD Failure_Code = 6

// FAILURE_REFUSED_SIGN marks a sign this dialect states no form for: the conditional and and the
// conditional or, which are nested ifs written flat, and the two order signs that hold equality,
// which a strict comparison states.
const FAILURE_REFUSED_SIGN Failure_Code = 7

// FAILURE_DOT_IMPORT marks an import that binds no name, which spills a package into the file.
const FAILURE_DOT_IMPORT Failure_Code = 8

// FAILURE_BLANK_IMPORT marks an import bound to the blank name, which is a package taken for its
// effects alone.
const FAILURE_BLANK_IMPORT Failure_Code = 9

// FAILURE_CONSTANT_CASE marks a constant name that is no run of uppercase words.
const FAILURE_CONSTANT_CASE Failure_Code = 10

// FAILURE_BARE_LOOP marks a for that no clause constrains, which runs forever.
const FAILURE_BARE_LOOP Failure_Code = 11

// FAILURE_UNNAMED_RESULT marks a signature result that carries no name.
const FAILURE_UNNAMED_RESULT Failure_Code = 12

// FAILURE_TYPE_ALIAS marks a type declaration bound by the assignment sign, which gives one
// type a second name instead of declaring a type of its own.
const FAILURE_TYPE_ALIAS Failure_Code = 13

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

// REJECT_CAUSE_MINIMUM is the smallest cause that marks one token of the source.
const REJECT_CAUSE_MINIMUM = uint8(FAILURE_SYNTAX)

// REJECT_CAUSE_MAXIMUM is the largest cause that marks one token of the source. A bound that
// the source overruns marks no token, thus it never reaches a rejection.
const REJECT_CAUSE_MAXIMUM = uint8(FAILURE_PACKAGE_CLAUSE)

// CAUSE_MINIMUM excludes the clean parse, because a cause is why a parse stopped.
const CAUSE_MINIMUM = uint8(FAILURE_SYNTAX)

// CAUSE_MAXIMUM is the deepest nesting, the largest cause.
const CAUSE_MAXIMUM = uint8(FAILURE_NESTING_DEPTH)

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

// WRAP_KIND_MINIMUM is the smallest kind that takes back the node parsed before it.
const WRAP_KIND_MINIMUM = int(NODE_GENERIC)

// WRAP_KIND_MAXIMUM is the largest kind that takes back the node parsed before it.
const WRAP_KIND_MAXIMUM = int(NODE_KEY_VALUE)

// PREFIX_KIND_MINIMUM is the smallest type kind whose first token names the whole form.
const PREFIX_KIND_MINIMUM = int(NODE_POINTER_TYPE)

// PREFIX_KIND_MAXIMUM is the largest type kind whose first token names the whole form.
const PREFIX_KIND_MAXIMUM = int(NODE_ELLIPSIS)

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

// GROUP_KIND_PARAMETER is the kind of a name a signature takes in.
const GROUP_KIND_PARAMETER = uint8(NODE_PARAMETER)

// GROUP_KIND_RESULT is the kind of a name a signature sends back.
const GROUP_KIND_RESULT = uint8(NODE_RESULT)

// VALUE_KIND_CONSTANT is the kind of a const declaration.
const VALUE_KIND_CONSTANT = uint8(NODE_CONSTANT)

// VALUE_KIND_VARIABLE is the kind of a var declaration.
const VALUE_KIND_VARIABLE = uint8(NODE_VARIABLE)

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

// NODE_KIND_MAXIMUM is the error kind, the largest kind.
const NODE_KIND_MAXIMUM = uint8(NODE_PARAMETER_NAME)

// Failure_Code names why a parse stopped, or FAILURE_NONE when it did not.
type Failure_Code uint8

// Failure_Code_Invariants states the clean parse and every cause.
func Failure_Code_Invariants(value Failure_Code, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), FAILURE_CODE_MINIMUM, FAILURE_CODE_MAXIMUM).
		Ensure()
}

// Cause is why one parse step stopped, thus never the clean parse.
type Cause uint8

// Cause_Invariants excludes the clean parse, which is no reason to stop.
func Cause_Invariants(value Cause, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), CAUSE_MINIMUM, CAUSE_MAXIMUM).
		Ensure()
}

// Reject_Cause is why one token of the source was refused, thus never an overrun bound.
type Reject_Cause uint8

// Reject_Cause_Invariants states the causes that mark one token.
func Reject_Cause_Invariants(value Reject_Cause, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), REJECT_CAUSE_MINIMUM, REJECT_CAUSE_MAXIMUM).
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
			int32(value), INDEX_MINIMUM, ANCESTOR_MAXIMUM,
			LINK_HOLE_FIRST, LINK_HOLE_FIRST, LINK_HOLE_FIRST, LINK_HOLE_FIRST,
		).
		Ensure()
}

// Head is the first child of one node, or INDEX_ABSENT when it has none.
type Head int32

// Head_Invariants holes the file slot, which is the root and therefore no node's child.
func Head_Invariants(value Head, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Holed_Int32(
			int32(value), INDEX_MINIMUM, INDEX_MAXIMUM,
			LINK_HOLE_ROOT, LINK_HOLE_ROOT, LINK_HOLE_ROOT, LINK_HOLE_ROOT,
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
			int32(value), INDEX_MINIMUM, INDEX_MAXIMUM,
			LINK_HOLE_ROOT, LINK_HOLE_FIRST, LINK_HOLE_FIRST, LINK_HOLE_FIRST,
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

// Wrap_Kind is the class of a node that takes back the node parsed before it, such as a binary
// operator taking the operand read before its own token was seen.
type Wrap_Kind int

// Wrap_Kind_Invariants states the block of kinds that reach back for an earlier node.
func Wrap_Kind_Invariants(value Wrap_Kind, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), WRAP_KIND_MINIMUM, WRAP_KIND_MAXIMUM).
		Ensure()
}

// Prefix_Kind is the class of a type whose first token names the whole form.
type Prefix_Kind int

// Prefix_Kind_Invariants states the block of prefix type kinds.
func Prefix_Kind_Invariants(value Prefix_Kind, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), PREFIX_KIND_MINIMUM, PREFIX_KIND_MAXIMUM).
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

// Group_Kind tells what a signature takes in from what it sends back.
type Group_Kind uint8

// Group_Kind_Invariants states the two halves of a signature.
func Group_Kind_Invariants(value Group_Kind, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value), GROUP_KIND_PARAMETER, GROUP_KIND_RESULT).
		Ensure()
}

// Value_Kind tells a const declaration from a var one, which is what says whether the names
// behind it answer to the constant case rule.
type Value_Kind uint8

// Value_Kind_Invariants states the two declarations that bind a value name.
func Value_Kind_Invariants(value Value_Kind, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value), VALUE_KIND_CONSTANT, VALUE_KIND_VARIABLE).
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
		Enum_3_Int(int(value), SUFFIX_KIND_GENERIC, SUFFIX_KIND_INDEX, SUFFIX_KIND_SLICE).
		Ensure()
}

// Precedence is the binding level of a binary operator.
type Precedence int

// Precedence_Invariants admits the level of a token that binds nothing and the three levels this
// dialect states.
func Precedence_Invariants(value Precedence, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_4_Int(
			int(value), int(PRECEDENCE_NONE), int(PRECEDENCE_COMPARISON),
			int(PRECEDENCE_ADDITION), int(PRECEDENCE_MULTIPLICATION)).
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
func Node_Invariants(value Node, namespace aver.Namespace) {
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
func Token_Storage_Invariants(value Token_Storage, namespace aver.Namespace) {
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
func Node_Storage_Invariants(value Node_Storage, namespace aver.Namespace) {
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
func Parent_Storage_Invariants(value Parent_Storage, namespace aver.Namespace) {
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
func Last_Child_Storage_Invariants(value Last_Child_Storage, namespace aver.Namespace) {
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
	value Earlier_Child_Storage, namespace aver.Namespace,
) {
	Earlier_Children_Invariants(value.Values, namespace)
}

// Token_Count is populated token extent, independent from every token position.
type Token_Count Token_Index

// Token_Count_Invariants follows complete token storage.
func Token_Count_Invariants(value Token_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), TOKEN_INDEX_MINIMUM, TOKEN_COUNT_MAXIMUM).
		Ensure()
}

// Token_Position is next grammar input, independent from populated extent.
type Token_Position Token_Index

// Token_Position_Invariants follows token arena indexes.
func Token_Position_Invariants(value Token_Position, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), TOKEN_INDEX_MINIMUM, TOKEN_INDEX_MAXIMUM).
		Ensure()
}

// Refusal_Token is first rejected token, independent from current input.
type Refusal_Token Token_Index

// Refusal_Token_Invariants follows token arena indexes.
func Refusal_Token_Invariants(value Refusal_Token, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), TOKEN_INDEX_MINIMUM, TOKEN_INDEX_MAXIMUM).
		Ensure()
}

// Blank_Token is trivia progress, independent from current input.
type Blank_Token Token_Index

// Blank_Token_Invariants follows one position past token storage.
func Blank_Token_Invariants(value Blank_Token, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), TOKEN_INDEX_MINIMUM, TOKEN_COUNT_MAXIMUM).
		Ensure()
}

// Token_Cursors names each lexical position so no index hides its role.
type Token_Cursors struct {
	// Count stands alone because it bounds written token storage.
	Count Token_Count
	// Position stands alone because it names next grammar input.
	Position Token_Position
	// Failure stands alone because first refusal survives later parser steps.
	Failure Refusal_Token
	// Blank stands alone because trivia consumption advances independently.
	Blank Blank_Token
}

// Token_Cursors_Invariants composes each independent lexical position.
func Token_Cursors_Invariants(value Token_Cursors, namespace aver.Namespace) {
	Token_Count_Invariants(value.Count, namespace)
	Token_Position_Invariants(value.Position, namespace)
	Refusal_Token_Invariants(value.Failure, namespace)
	Blank_Token_Invariants(value.Blank, namespace)
}

// Node_Count is populated arena extent, independent from nesting depth.
type Node_Count Index

// Node_Count_Invariants follows one position past node storage.
func Node_Count_Invariants(value Node_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), INDEX_MINIMUM, NODE_COUNT_MAXIMUM).
		Ensure()
}

// Parse_Depth is open-parent count, independent from arena extent.
type Parse_Depth Index

// Parse_Depth_Invariants follows parent storage extent.
func Parse_Depth_Invariants(value Parse_Depth, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), INDEX_MINIMUM, DEPTH_MAXIMUM).
		Ensure()
}

// Node_Cursors names arena extent and nesting depth without indexed roles.
type Node_Cursors struct {
	// Count stands alone because it bounds written node storage.
	Count Node_Count
	// Depth stands alone because it bounds open-parent storage.
	Depth Parse_Depth
}

// Node_Cursors_Invariants composes arena extent and nesting depth.
func Node_Cursors_Invariants(value Node_Cursors, namespace aver.Namespace) {
	Node_Count_Invariants(value.Count, namespace)
	Parse_Depth_Invariants(value.Depth, namespace)
}

// Causes leaves no indexed collection around one first-failure value.
type Causes struct {
	// Cause stands alone because parser records only first refusal.
	Cause Failure_Code
}

// Causes_Invariants gives first refusal normal failure obligations.
func Causes_Invariants(value Causes, namespace aver.Namespace) {
	Failure_Code_Invariants(value.Cause, namespace)
}

// Failed is refusal state, independent from grammar ambiguity flags.
type Failed Boolean

// Failed_Invariants states both refusal states.
func Failed_Invariants(value Failed, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Parser has failed.").
		Ensure()
}

// Plain_Brace marks control-clause brace ownership.
type Plain_Brace Boolean

// Plain_Brace_Invariants states both brace readings.
func Plain_Brace_Invariants(value Plain_Brace, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Brace opens plain block.").
		Ensure()
}

// Type_Assertion marks type-switch ambiguity.
type Type_Assertion Boolean

// Type_Assertion_Invariants states both assertion readings.
func Type_Assertion_Invariants(value Type_Assertion, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Clause holds type assertion.").
		Ensure()
}

// Flags names each parser control bit so no index hides its role.
type Flags struct {
	// Failed stands alone because refusal state survives later parser steps.
	Failed Failed
	// Plain_Brace stands alone because control clauses own this ambiguity.
	Plain_Brace Plain_Brace
	// Type_Assertion stands alone because switch classification owns this ambiguity.
	Type_Assertion Type_Assertion
}

// Flags_Invariants composes each independent parser control bit.
func Flags_Invariants(value Flags, namespace aver.Namespace) {
	Failed_Invariants(value.Failed, namespace)
	Plain_Brace_Invariants(value.Plain_Brace, namespace)
	Type_Assertion_Invariants(value.Type_Assertion, namespace)
}

// Parse_State_Fields keeps raw mutable parser state separate from validated transport.
type Parse_State_Fields struct {
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
	// Token_Cursors holds the token count and the read position.
	Token_Cursors Token_Cursors
	// Node_Cursors holds the node count and the nesting depth.
	Node_Cursors Node_Cursors
	// Causes holds why the parse failed, or FAILURE_NONE while it still stands.
	Causes Causes
	// Flags holds whether the parse failed.
	Flags Flags
}

// Parse_State_Fields_Invariants composes raw caller storage before parser lifecycle validation.
func Parse_State_Fields_Invariants(value Parse_State_Fields, namespace aver.Namespace) {
	Token_Storage_Invariants(value.Tokens, namespace)
	Node_Storage_Invariants(value.Nodes, namespace)
	Parent_Storage_Invariants(value.Parents, namespace)
	Last_Child_Storage_Invariants(value.Last_Children, namespace)
	Earlier_Child_Storage_Invariants(value.Earlier_Children, namespace)
	Token_Cursors_Invariants(value.Token_Cursors, namespace)
	Node_Cursors_Invariants(value.Node_Cursors, namespace)
	Causes_Invariants(value.Causes, namespace)
	Flags_Invariants(value.Flags, namespace)
}

// Parse_State_Fields_Stored prevents recursive parser steps from revalidating every store.
type Parse_State_Fields_Stored interface{}

// Parse_State_Fields_Stored_Invariants fixes parse-state representation.
func Parse_State_Fields_Stored_Invariants(value Parse_State_Fields_Stored, _ aver.Namespace) {
	_, valid := value.(Parse_State_Fields)
	aver.Always(valid == (value != nil), "Parse state has expected storage type.")
}

// Parse_State_Envelope keeps caller storage behind one representation boundary.
type Parse_State_Envelope struct {
	// Parse_State_Fields stays embedded so callers retain concrete field access.
	Parse_State_Fields
}

// Parse_State_Envelope_Invariants composes concrete caller storage.
func Parse_State_Envelope_Invariants(value Parse_State_Envelope, namespace aver.Namespace) {
	Parse_State_Fields_Invariants(value.Parse_State_Fields, namespace)
}

// Parse_State is one validated view over caller-owned parser storage.
type Parse_State Parse_State_Envelope

// Parse_State_Invariants fixes representation without revalidating stores in recursive steps.
func Parse_State_Invariants(value Parse_State, namespace aver.Namespace) {
	Parse_State_Fields_Stored_Invariants(
		Parse_State_Fields_Stored(value.Parse_State_Fields), namespace,
	)
}

// Parse_State_Handle gives caller-owned parse storage one identity.
type Parse_State_Handle *Parse_State

// Parse_State_Handle_Invariants composes present parse storage.
func Parse_State_Handle_Invariants(value Parse_State_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Parse_State_Invariants(*value, namespace)
}

// Parse_Result keeps syntax root and refusal state at one output boundary.
type Parse_Result struct {
	// Root remains available after refused parses because partial trees still carry evidence.
	Root Root
	// OK distinguishes accepted syntax from useful partial trees.
	OK Boolean
}

// Parse_Result_Invariants composes one parse outcome.
func Parse_Result_Invariants(value Parse_Result, namespace aver.Namespace) {
	Root_Invariants(value.Root, namespace)
	Boolean_Invariants(value.OK, namespace)
}

// Parse scans a source into the token run and descends it into the arena. A source the grammar
// rejects still returns the tree that was built, with an error node at the token that failed,
// because a linter reads more from a partial tree than from nothing.
func Parse(subject Parse_State_Handle, source token.Source) (result Parse_Result) {
	defer func() { Parse_Result_Invariants(result, "parse.result") }()
	Parse_State_Handle_Invariants(subject, "parse.subject")
	Token_Storage_Invariants(subject.Tokens, "parse.tokens")
	Node_Storage_Invariants(subject.Nodes, "parse.nodes")
	Parent_Storage_Invariants(subject.Parents, "parse.parents")
	Last_Child_Storage_Invariants(subject.Last_Children, "parse.last_children")
	Earlier_Child_Storage_Invariants(subject.Earlier_Children, "parse.earlier_children")
	token.Source_Invariants(source, "parse.source")
	reset(subject)
	if !bool(scan_source(subject, source)) {
		result.Root = Root(INDEX_ABSENT)
		result.OK = false
		return result
	}
	result.Root = Root(open_node(subject, NODE_FILE))
	parse_file(subject)
	close_node(subject)
	name_wide_identifier(subject, source)
	// The import pass runs first, because an import stands ahead of every declaration and the
	// failure a caller reads should be the one its author meets first.
	name_blank_import(subject, source)
	name_constant_case(subject, source)
	name_bare_loop(subject, source)
	name_unnamed_result(subject)
	name_banned_identifier(subject, source)
	result.OK = Boolean(!bool(subject.Flags.Failed))
	return result
}

// Node_At returns one slot of the arena. The parse writes every slot it counts, thus a slot the
// count does not reach is no node and reading it is a caller defect.
func Node_At(subject Parse_State_Handle, index Index) (node Node) {
	defer func() { Node_Invariants(node, "node_at.node") }()
	Parse_State_Handle_Invariants(subject, "node_at.subject")
	Index_Invariants(index, "node_at.index")
	aver.Always(
		index < Index(subject.Node_Cursors.Count),
		"A read node lies inside the count the parse wrote.",
	)
	return subject.Nodes.Values[index]
}

// Token_At returns one token of the run, the token that names a node.
func Token_At(subject Parse_State_Handle, index Token_Index) (one token.Token) {
	defer func() { token.Token_Invariants(one, "token_at.one") }()
	Parse_State_Handle_Invariants(subject, "token_at.subject")
	Token_Index_Invariants(index, "token_at.index")
	aver.Always(
		index < Token_Index(subject.Token_Cursors.Count),
		"A read token lies inside the count the scan wrote.",
	)
	return subject.Tokens.Values[index]
}

// Parse_State_Token_Run borrows only the token prefix the last scan wrote.
func Parse_State_Token_Run(subject Parse_State_Handle) (run Token_Run) {
	defer func() { Token_Run_Invariants(run, "parse_state_token_run.run") }()
	Parse_State_Handle_Invariants(subject, "parse_state_token_run.subject")
	return Token_Run(subject.Tokens.Values[:subject.Token_Cursors.Count])
}

// Clears the cursors and marks slot zero, which holds no node. Slot zero carries the error kind
// so a stray read of the absent slot is loud rather than a plausible node.
func reset(subject Parse_State_Handle) {
	Parse_State_Handle_Invariants(subject, "reset.subject")
	subject.Token_Cursors.Count = 0
	subject.Token_Cursors.Position = 0
	subject.Token_Cursors.Failure = 0
	subject.Token_Cursors.Blank = 0
	subject.Causes.Cause = Failure_Code(FAILURE_NONE)
	subject.Node_Cursors.Count = Node_Count(INDEX_FIRST)
	subject.Node_Cursors.Depth = 0
	subject.Last_Children.Values[0] = INDEX_ABSENT
	subject.Earlier_Children.Values[0] = INDEX_ABSENT
	subject.Flags.Failed = false
	subject.Flags.Plain_Brace = false
	subject.Nodes.Values[INDEX_ABSENT] = Node{
		Kind:        NODE_ERROR,
		Token:       0,
		Parent:      Ancestor(INDEX_ABSENT),
		First_Child: Head(INDEX_ABSENT),
		Next:        Successor(INDEX_ABSENT),
	}
}

// Fills the token run from the source. A run past the bound fails the parse rather than writing
// outside the array.
func scan_source(subject Parse_State_Handle, source token.Source) (ok Boolean) {
	defer func() { Boolean_Invariants(ok, "scan_source.ok") }()
	Parse_State_Handle_Invariants(subject, "scan_source.subject")
	token.Source_Invariants(source, "scan_source.source")
	scanner := token.Scanner{
		Source: source, Offset: 0, Previous: token.KIND_END_OF_FILE, Newline: false,
	}
	count := 0
	for count < TOKEN_COUNT_MAXIMUM {
		one := token.Scan(&scanner)
		subject.Tokens.Values[count] = one
		count++
		if one.Kind == token.KIND_END_OF_FILE {
			subject.Token_Cursors.Count = Token_Count(count)
			return true
		}
	}
	// The run that was written stays readable, so a caller that reads the failure token finds a
	// token there rather than a slot the scan never reached.
	subject.Token_Cursors.Count = Token_Count(count)
	fail(subject, Cause(FAILURE_TOKEN_COUNT))
	return false
}

// Marks the parse failed and records why. The first cause stands, because a later step reads a
// source the first failure already knocked off the rails and its complaint would mislead.
func fail(subject Parse_State_Handle, cause Cause) {
	Parse_State_Handle_Invariants(subject, "fail.subject")
	Cause_Invariants(cause, "fail.cause")
	if bool(subject.Flags.Failed) {
		return
	}
	subject.Flags.Failed = true
	subject.Causes.Cause = Failure_Code(cause)
	subject.Token_Cursors.Failure = Refusal_Token(subject.Token_Cursors.Position)
}

// Reports whether the parse has already failed. Each step asks before it descends, which is what
// keeps the recursion bounded once the arena or the depth runs out.
func failed(subject Parse_State_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "failed.yes") }()
	Parse_State_Handle_Invariants(subject, "failed.subject")
	return Boolean(subject.Flags.Failed)
}

// Marks the current token as the point the grammar rejected and says why.
func reject(subject Parse_State_Handle, cause Reject_Cause) {
	Parse_State_Handle_Invariants(subject, "reject.subject")
	Reject_Cause_Invariants(cause, "reject.cause")
	fail(subject, Cause(cause))
	// An illegal byte is taken so a scan error cannot stall one token forever.
	accept(subject, token.KIND_ILLEGAL)
	open_node(subject, NODE_ERROR)
	close_node(subject)
}

// Reports the kind of the token the cursor stands on.
func at(subject Parse_State_Handle) (kind token.Kind) {
	defer func() { token.Kind_Invariants(kind, "at.kind") }()
	Parse_State_Handle_Invariants(subject, "at.subject")
	return subject.Tokens.Values[subject.Token_Cursors.Position].Kind
}

// Reports the kind of the token after the cursor, which the grammar needs where one token cannot
// tell a label from an expression or a receiver from a parameter.
func after(subject Parse_State_Handle) (kind token.Kind) {
	defer func() { token.Kind_Invariants(kind, "after.kind") }()
	Parse_State_Handle_Invariants(subject, "after.subject")
	position := subject.Token_Cursors.Position
	if int(position)+1 >= int(subject.Token_Cursors.Count) {
		return token.KIND_END_OF_FILE
	}
	return subject.Tokens.Values[position+1].Kind
}

// Steps the cursor past the current token. The run ends in one end-of-file token that the cursor
// never passes, thus a step that misreads the grammar stalls there instead of running off.
func advance(subject Parse_State_Handle) {
	Parse_State_Handle_Invariants(subject, "advance.subject")
	step(subject)
	skip_comments(subject)
}

// Steps the cursor past one token and takes no trivia. The run ends in one end-of-file token
// that the cursor never passes, thus a step that misreads the grammar stalls there.
func step(subject Parse_State_Handle) {
	Parse_State_Handle_Invariants(subject, "step.subject")
	position := subject.Token_Cursors.Position
	if int(position)+1 < int(subject.Token_Cursors.Count) {
		subject.Token_Cursors.Position = position + 1
	}
}

// Takes the run of empty lines that stands before the token at the cursor. The cursor only
// ever moves forward, thus one slot recording how far the tree already holds is enough to keep a
// run out of the tree twice.
func take_blank(subject Parse_State_Handle) {
	Parse_State_Handle_Invariants(subject, "take_blank.subject")
	position := subject.Token_Cursors.Position
	if Token_Index(subject.Token_Cursors.Blank) > Token_Index(position) {
		return
	}
	subject.Token_Cursors.Blank = Blank_Token(position + 1)
	if subject.Tokens.Values[position].Blanks == 0 {
		return
	}
	open_node(subject, NODE_BLANK)
	close_node(subject)
}

// Steps past the current token when it is of this kind.
func accept(subject Parse_State_Handle, kind token.Kind) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "accept.yes") }()
	Parse_State_Handle_Invariants(subject, "accept.subject")
	token.Kind_Invariants(kind, "accept.kind")
	if at(subject) != kind {
		return false
	}
	advance(subject)
	return true
}

// Takes one slot for a node of this kind at the current token. The arena is bounded, thus an
// exhausted arena fails the parse rather than writing past the array.
func allocate(subject Parse_State_Handle, kind Node_Kind) (index Index) {
	defer func() { Index_Invariants(index, "allocate.index") }()
	Parse_State_Handle_Invariants(subject, "allocate.subject")
	Node_Kind_Invariants(kind, "allocate.kind")
	count := subject.Node_Cursors.Count
	if int(count) > INDEX_MAXIMUM {
		fail(subject, Cause(FAILURE_NODE_COUNT))
		return INDEX_ABSENT
	}
	subject.Node_Cursors.Count = count + 1
	subject.Nodes.Values[count] = Node{
		Kind:        kind,
		Token:       Token_Index(subject.Token_Cursors.Position),
		Parent:      Ancestor(INDEX_ABSENT),
		First_Child: Head(INDEX_ABSENT),
		Next:        Successor(INDEX_ABSENT),
	}
	return Index(count)
}

// Links one node under the open parent as its final child.
func attach(subject Parse_State_Handle, child Index) {
	Parse_State_Handle_Invariants(subject, "attach.subject")
	Index_Invariants(child, "attach.child")
	if child == INDEX_ABSENT {
		return
	}
	// The open parent is read inline rather than through a step of its own, because a step
	// that returned it would owe every slot number at one call site and the parent of the
	// first attachment is always the root.
	depth := subject.Node_Cursors.Depth
	if depth == 0 {
		return
	}
	if int(depth) > DEPTH_MAXIMUM {
		return
	}
	parent := subject.Parents.Values[depth-1]
	if parent == INDEX_ABSENT {
		return
	}
	previous := subject.Last_Children.Values[depth-1]
	subject.Nodes.Values[child].Parent = Ancestor(parent)
	if previous == INDEX_ABSENT {
		subject.Nodes.Values[parent].First_Child = Head(child)
	} else {
		subject.Nodes.Values[previous].Next = Successor(child)
	}
	subject.Earlier_Children.Values[depth-1] = previous
	subject.Last_Children.Values[depth-1] = child
}

// Takes one slot, links it under the open parent, and opens it so later nodes attach to it. The
// nesting depth is bounded, thus a hostile depth ends the parse instead of the goroutine stack.
func open_node(subject Parse_State_Handle, kind Node_Kind) (index Index) {
	defer func() { Index_Invariants(index, "open_node.index") }()
	Parse_State_Handle_Invariants(subject, "open_node.subject")
	Node_Kind_Invariants(kind, "open_node.kind")
	index = allocate(subject, kind)
	attach(subject, index)
	depth := subject.Node_Cursors.Depth
	if int(depth) < DEPTH_MAXIMUM {
		subject.Parents.Values[depth] = index
		subject.Last_Children.Values[depth] = INDEX_ABSENT
		subject.Earlier_Children.Values[depth] = INDEX_ABSENT
	} else {
		fail(subject, Cause(FAILURE_NESTING_DEPTH))
	}
	subject.Node_Cursors.Depth = depth + 1
	return index
}

// Closes the open parent. The count is stepped even past the depth bound, so an overflowed open
// and its close stay paired and the parent stack never drifts.
func close_node(subject Parse_State_Handle) {
	Parse_State_Handle_Invariants(subject, "close_node.subject")
	depth := subject.Node_Cursors.Depth
	if depth > 0 {
		subject.Node_Cursors.Depth = depth - 1
	}
}

// Opens one node in the place of the final child of the open parent and gives it that child. A
// binary operator adopts the operand parsed before the operator was seen, and this is the one
// place the tree runs backward. The adopted slot never leaves this step, because a step that
// handed it back would owe every slot number at one call site.
func wrap_last(subject Parse_State_Handle, kind Wrap_Kind) {
	Parse_State_Handle_Invariants(subject, "wrap_last.subject")
	Wrap_Kind_Invariants(kind, "wrap_last.kind")
	last := INDEX_ABSENT
	depth := subject.Node_Cursors.Depth
	if depth > 0 {
		if int(depth) <= DEPTH_MAXIMUM {
			parent := subject.Parents.Values[depth-1]
			last = subject.Last_Children.Values[depth-1]
			previous := subject.Earlier_Children.Values[depth-1]
			subject.Last_Children.Values[depth-1] = previous
			if previous == INDEX_ABSENT {
				subject.Nodes.Values[parent].First_Child = Head(INDEX_ABSENT)
			} else {
				subject.Nodes.Values[previous].Next = Successor(INDEX_ABSENT)
			}
			subject.Nodes.Values[last].Parent = Ancestor(INDEX_ABSENT)
			subject.Nodes.Values[last].Next = Successor(INDEX_ABSENT)
		}
	}
	open_node(subject, Node_Kind(kind))
	attach(subject, last)
}

// Parses the package clause and every declaration of a file.
func parse_file(subject Parse_State_Handle) {
	Parse_State_Handle_Invariants(subject, "parse_file.subject")
	skip_comments(subject)
	if at(subject) != token.KIND_PACKAGE {
		reject(subject, Reject_Cause(FAILURE_PACKAGE_CLAUSE))
		return
	}
	advance(subject)
	parse_name(subject)
	end_line(subject)
	for range TOKEN_COUNT_MAXIMUM {
		if bool(failed(subject)) {
			return
		}
		if bool(accept(subject, token.KIND_END_OF_FILE)) {
			return
		}
		parse_declaration(subject)
	}
	return
}

// Takes every comment at the cursor into the open parent, so a doc comment keeps the place it
// holds above the declaration that follows it.
func skip_comments(subject Parse_State_Handle) {
	Parse_State_Handle_Invariants(subject, "skip_comments.subject")
	// The walk is a loop and never a descent, because a file may hold a run of comments as long
	// as it likes and a descent would spend one stack frame on each.
	for range TOKEN_COUNT_MAXIMUM {
		take_blank(subject)
		if at(subject) != token.KIND_COMMENT {
			return
		}
		open_node(subject, NODE_COMMENT)
		close_node(subject)
		step(subject)
	}
}

// Closes one declaration or statement: any trailing comment, then the semicolon the scanner
// wrote or inserted. A closing bracket stands for the semicolon Go lets a source leave out.
func end_line(subject Parse_State_Handle) (ok Boolean) {
	defer func() { Boolean_Invariants(ok, "end_line.ok") }()
	Parse_State_Handle_Invariants(subject, "end_line.subject")
	skip_comments(subject)
	if bool(accept(subject, token.KIND_SEMICOLON)) {
		return true
	}
	switch at(subject) {
	case token.KIND_BRACE_RIGHT, token.KIND_PARENTHESIS_RIGHT, token.KIND_BRACKET_RIGHT,
		token.KIND_END_OF_FILE, token.KIND_CASE, token.KIND_DEFAULT, token.KIND_COMMA,
		token.KIND_COLON, token.KIND_ELSE:
		return true
	}
	reject(subject, Reject_Cause(FAILURE_SYNTAX))
	return false
}

// Takes one identifier as a node of its own.
func parse_name(subject Parse_State_Handle) {
	Parse_State_Handle_Invariants(subject, "parse_name.subject")
	if at(subject) != token.KIND_IDENTIFIER {
		reject(subject, Reject_Cause(FAILURE_SYNTAX))
		return
	}
	open_node(subject, NODE_IDENTIFIER)
	close_node(subject)
	accept(subject, token.KIND_IDENTIFIER)
	return
}

// Parses one top-level declaration.
func parse_declaration(subject Parse_State_Handle) {
	Parse_State_Handle_Invariants(subject, "parse_declaration.subject")
	skip_comments(subject)
	switch at(subject) {
	case token.KIND_IMPORT:
		parse_import(subject)
		return
	case token.KIND_CONSTANT, token.KIND_VARIABLE:
		parse_value_declaration(subject)
		return
	case token.KIND_TYPE:
		parse_type_declaration(subject)
		return
	case token.KIND_FUNCTION:
		parse_function_declaration(subject)
		return
	case token.KIND_SEMICOLON:
		advance(subject)
		return
	case token.KIND_END_OF_FILE:
		return
	}
	reject(subject, Reject_Cause(FAILURE_SYNTAX))
	return
}

// Reports the kind of the token two ahead of the cursor. A bracket after a type name opens a
// type parameter list or an array capacity, and only the token past the first one tells them
// apart.
// Reports whether the bracket at the cursor opens a type parameter list rather than an array
// capacity. A name followed by the opener of a constraint is a type parameter; anything else is
// the capacity of an array.
func starts_type_parameters(subject Parse_State_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "starts_type_parameters.yes") }()
	Parse_State_Handle_Invariants(subject, "starts_type_parameters.subject")
	if after(subject) != token.KIND_IDENTIFIER {
		return false
	}
	position := subject.Token_Cursors.Position
	if int(position)+2 >= int(subject.Token_Cursors.Count) {
		return false
	}
	// A comma behind the first name says a type parameter list, because an array capacity
	// holds one expression and never a list.
	switch subject.Tokens.Values[position+2].Kind {
	case token.KIND_COMMA:
		return true
	}
	return opens_type(subject.Tokens.Values[position+2].Kind)
}

// Parses one import declaration. A parenthesized group is refused like any other, because one
// declaration for each path is what the reader greps for.
func parse_import(subject Parse_State_Handle) {
	Parse_State_Handle_Invariants(subject, "parse_import.subject")
	advance(subject)
	if at(subject) == token.KIND_PARENTHESIS_LEFT {
		reject(subject, Reject_Cause(FAILURE_DECLARATION_GROUP))
		return
	}
	parse_import_path(subject)
	end_line(subject)
}

// Parses one import path and its optional name. A dot import and a blank import parse, because
// the linter must see the form it rejects.
func parse_import_path(subject Parse_State_Handle) {
	Parse_State_Handle_Invariants(subject, "parse_import_path.subject")
	open_node(subject, NODE_IMPORT)
	if at(subject) == token.KIND_IDENTIFIER {
		open_node(subject, NODE_IMPORT_NAME)
		close_node(subject)
		accept(subject, token.KIND_IDENTIFIER)
	}
	if at(subject) == token.KIND_PERIOD {
		reject(subject, Reject_Cause(FAILURE_DOT_IMPORT))
		close_node(subject)
		return
	}
	if at(subject) != token.KIND_STRING {
		reject(subject, Reject_Cause(FAILURE_SYNTAX))
		close_node(subject)
		return
	}
	open_node(subject, NODE_STRING)
	close_node(subject)
	advance(subject)
	close_node(subject)
}

// Parses one const or var declaration. Each stands alone, because the linter bans the
// parenthesized group.
func parse_value_declaration(subject Parse_State_Handle) {
	Parse_State_Handle_Invariants(subject, "parse_value_declaration.subject")
	kind := NODE_CONSTANT
	if at(subject) == token.KIND_VARIABLE {
		kind = NODE_VARIABLE
	}
	open_node(subject, kind)
	advance(subject)
	if at(subject) == token.KIND_PARENTHESIS_LEFT {
		reject(subject, Reject_Cause(FAILURE_DECLARATION_GROUP))
		close_node(subject)
		return
	}
	parse_value_name(subject, Value_Kind(kind))
	for bool(accept(subject, token.KIND_COMMA)) {
		parse_value_name(subject, Value_Kind(kind))
	}
	switch at(subject) {
	case token.KIND_ASSIGN, token.KIND_SEMICOLON, token.KIND_END_OF_FILE:
	default:
		parse_type(subject)
	}
	if bool(accept(subject, token.KIND_ASSIGN)) {
		parse_expression_list(subject)
	}
	close_node(subject)
	end_line(subject)
	return
}

// Parses one type or alias declaration.
func parse_type_declaration(subject Parse_State_Handle) {
	Parse_State_Handle_Invariants(subject, "parse_type_declaration.subject")
	open_node(subject, NODE_TYPE)
	advance(subject)
	if at(subject) == token.KIND_PARENTHESIS_LEFT {
		reject(subject, Reject_Cause(FAILURE_DECLARATION_GROUP))
		close_node(subject)
		return
	}
	parse_name(subject)
	if at(subject) == token.KIND_BRACKET_LEFT {
		if bool(starts_type_parameters(subject)) {
			parse_type_parameters(subject)
		}
	}
	if at(subject) == token.KIND_ASSIGN {
		reject(subject, Reject_Cause(FAILURE_TYPE_ALIAS))
		close_node(subject)
		return
	}
	parse_type(subject)
	close_node(subject)
	end_line(subject)
	return
}

// Parses one function or method declaration.
func parse_function_declaration(subject Parse_State_Handle) {
	Parse_State_Handle_Invariants(subject, "parse_function_declaration.subject")
	open_node(subject, NODE_FUNCTION)
	advance(subject)
	if at(subject) == token.KIND_PARENTHESIS_LEFT {
		open_node(subject, NODE_RECEIVER)
		parse_parameter_group(subject, Group_Kind(NODE_PARAMETER))
		close_node(subject)
	}
	parse_name(subject)
	if at(subject) == token.KIND_BRACKET_LEFT {
		parse_type_parameters(subject)
	}
	parse_signature(subject)
	if at(subject) == token.KIND_BRACE_LEFT {
		parse_block(subject)
	}
	close_node(subject)
	end_line(subject)
	return
}

// Parses one type parameter list.
func parse_type_parameters(subject Parse_State_Handle) {
	Parse_State_Handle_Invariants(subject, "parse_type_parameters.subject")
	advance(subject)
	for range TOKEN_COUNT_MAXIMUM {
		if bool(failed(subject)) {
			return
		}
		if bool(accept(subject, token.KIND_BRACKET_RIGHT)) {
			return
		}
		open_node(subject, NODE_TYPE_PARAMETER)
		parse_name(subject)
		for bool(accept(subject, token.KIND_COMMA)) {
			if at(subject) == token.KIND_BRACKET_RIGHT {
				break
			}
			parse_name(subject)
		}
		if at(subject) != token.KIND_BRACKET_RIGHT {
			parse_constraint(subject)
		}
		close_node(subject)
		accept(subject, token.KIND_COMMA)
	}
	return
}

// Parses one type constraint: a type, a union of terms, or an approximation under the tilde.
func parse_constraint(subject Parse_State_Handle) {
	Parse_State_Handle_Invariants(subject, "parse_constraint.subject")
	parse_term(subject)
	if at(subject) != token.KIND_OR {
		return
	}
	wrap_last(subject, Wrap_Kind(NODE_UNION))
	for bool(accept(subject, token.KIND_OR)) {
		parse_term(subject)
	}
	close_node(subject)
	return
}

// Parses one constraint term. The term node names the type, because the tilde was already taken.
func parse_term(subject Parse_State_Handle) {
	Parse_State_Handle_Invariants(subject, "parse_term.subject")
	if bool(accept(subject, token.KIND_TILDE)) {
		open_node(subject, NODE_TERM)
		parse_type(subject)
		close_node(subject)
		return
	}
	parse_type(subject)
	return
}

// Parses one signature: its parameter group and its results.
func parse_signature(subject Parse_State_Handle) {
	Parse_State_Handle_Invariants(subject, "parse_signature.subject")
	parse_parameter_group(subject, Group_Kind(NODE_PARAMETER))
	switch at(subject) {
	case token.KIND_PARENTHESIS_LEFT:
		parse_parameter_group(subject, Group_Kind(NODE_RESULT))
		return
	case token.KIND_SEMICOLON, token.KIND_BRACE_LEFT, token.KIND_COMMA, token.KIND_STRING,
		token.KIND_PARENTHESIS_RIGHT, token.KIND_BRACKET_RIGHT, token.KIND_BRACE_RIGHT,
		token.KIND_COMMENT, token.KIND_END_OF_FILE, token.KIND_COLON, token.KIND_ASSIGN:
		return
	}
	open_node(subject, NODE_RESULT)
	parse_type(subject)
	close_node(subject)
	return
}

// Parses one parenthesized parameter or result group.
func parse_parameter_group(subject Parse_State_Handle, kind Group_Kind) {
	Parse_State_Handle_Invariants(subject, "parse_parameter_group.subject")
	Group_Kind_Invariants(kind, "parse_parameter_group.kind")
	if !bool(accept(subject, token.KIND_PARENTHESIS_LEFT)) {
		reject(subject, Reject_Cause(FAILURE_SYNTAX))
		return
	}
	for range TOKEN_COUNT_MAXIMUM {
		if bool(failed(subject)) {
			return
		}
		skip_comments(subject)
		if bool(accept(subject, token.KIND_PARENTHESIS_RIGHT)) {
			return
		}
		parse_parameter(subject, kind)
		accept(subject, token.KIND_COMMA)
	}
	return
}

// Parses one parameter. A name is optional and only its follower tells the two forms apart, so
// an identifier ahead of a type opener is a name and an identifier ahead of anything else is
// the type itself.
func parse_parameter(subject Parse_State_Handle, kind Group_Kind) {
	Parse_State_Handle_Invariants(subject, "parse_parameter.subject")
	Group_Kind_Invariants(kind, "parse_parameter.kind")
	open_node(subject, Node_Kind(kind))
	if at(subject) == token.KIND_IDENTIFIER {
		named := bool(opens_type(after(subject)))
		if after(subject) == token.KIND_COMMA {
			named = true
		}
		if named {
			parse_parameter_name(subject)
			for at(subject) == token.KIND_COMMA {
				if after(subject) != token.KIND_IDENTIFIER {
					break
				}
				if bool(qualifies_after(subject)) {
					break
				}
				advance(subject)
				parse_parameter_name(subject)
			}
		}
	}
	// A run of bare names with no type behind it was a run of types all along, and only the
	// token after the final name tells the two apart. Each name is read back to an identifier
	// here, so a later pass reads a name as a name and never as a type.
	switch at(subject) {
	case token.KIND_COMMA, token.KIND_PARENTHESIS_RIGHT:
		unname_children(subject)
		close_node(subject)
		return
	}
	parse_type(subject)
	close_node(subject)
	return
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

// Parses one type. Every Go type form reaches here, and the composite forms hand off to a step
// of their own so each stays inside the size a reader can hold.
func parse_type(subject Parse_State_Handle) {
	Parse_State_Handle_Invariants(subject, "parse_type.subject")
	if bool(failed(subject)) {
		return
	}
	if !bool(opens_type(at(subject))) {
		reject(subject, Reject_Cause(FAILURE_SYNTAX))
		return
	}
	switch at(subject) {
	case token.KIND_IDENTIFIER:
		parse_type_name(subject)
		return
	case token.KIND_STAR:
		parse_prefix_type(subject, Prefix_Kind(NODE_POINTER_TYPE))
		return
	case token.KIND_ELLIPSIS:
		parse_prefix_type(subject, Prefix_Kind(NODE_ELLIPSIS))
		return
	case token.KIND_BRACKET_LEFT:
		parse_array_type(subject)
		return
	case token.KIND_MAP:
		parse_map_type(subject)
		return
	case token.KIND_CHANNEL:
		parse_channel_type(subject)
		return
	case token.KIND_ARROW:
		parse_prefix_type(subject, Prefix_Kind(NODE_CHANNEL_RECEIVE))
		return
	case token.KIND_FUNCTION:
		parse_function_type(subject)
		return
	case token.KIND_STRUCTURE:
		parse_structure_type(subject)
		return
	case token.KIND_INTERFACE:
		parse_interface_type(subject)
		return
	case token.KIND_PARENTHESIS_LEFT:
		advance(subject)
		parse_type(subject)
		if !bool(accept(subject, token.KIND_PARENTHESIS_RIGHT)) {
			reject(subject, Reject_Cause(FAILURE_SYNTAX))
		}
		return
	}
	reject(subject, Reject_Cause(FAILURE_SYNTAX))
	return
}

// Parses one type whose first token names the whole form and whose body is one further type.
func parse_prefix_type(subject Parse_State_Handle, kind Prefix_Kind) {
	Parse_State_Handle_Invariants(subject, "parse_prefix_type.subject")
	Prefix_Kind_Invariants(kind, "parse_prefix_type.kind")
	open_node(subject, Node_Kind(kind))
	advance(subject)
	accept(subject, token.KIND_CHANNEL)
	parse_type(subject)
	close_node(subject)
	return
}

// Parses one named type, its package qualifier, and its type arguments.
func parse_type_name(subject Parse_State_Handle) {
	Parse_State_Handle_Invariants(subject, "parse_type_name.subject")
	parse_name(subject)
	if at(subject) == token.KIND_PERIOD {
		wrap_last(subject, Wrap_Kind(NODE_SELECTOR))
		advance(subject)
		parse_name(subject)
		close_node(subject)
	}
	if at(subject) != token.KIND_BRACKET_LEFT {
		return
	}
	wrap_last(subject, Wrap_Kind(NODE_GENERIC))
	advance(subject)
	for range TOKEN_COUNT_MAXIMUM {
		if bool(failed(subject)) {
			break
		}
		if bool(accept(subject, token.KIND_BRACKET_RIGHT)) {
			break
		}
		parse_type(subject)
		accept(subject, token.KIND_COMMA)
	}
	close_node(subject)
	return
}

// Parses one array or slice type. The bracket pair alone says slice, and anything inside it says
// array, thus the kind is settled only after the capacity is read.
func parse_array_type(subject Parse_State_Handle) {
	Parse_State_Handle_Invariants(subject, "parse_array_type.subject")
	index := open_node(subject, NODE_SLICE_TYPE)
	advance(subject)
	if !bool(accept(subject, token.KIND_BRACKET_RIGHT)) {
		if index != INDEX_ABSENT {
			subject.Nodes.Values[index].Kind = NODE_ARRAY_TYPE
		}
		if at(subject) == token.KIND_ELLIPSIS {
			open_node(subject, NODE_ELLIPSIS)
			close_node(subject)
			advance(subject)
		} else {
			parse_expression(subject)
		}
		if !bool(accept(subject, token.KIND_BRACKET_RIGHT)) {
			reject(subject, Reject_Cause(FAILURE_SYNTAX))
		}
	}
	parse_type(subject)
	close_node(subject)
	return
}

// Parses one map type, its key, and its value.
func parse_map_type(subject Parse_State_Handle) {
	Parse_State_Handle_Invariants(subject, "parse_map_type.subject")
	open_node(subject, NODE_MAP_TYPE)
	advance(subject)
	if !bool(accept(subject, token.KIND_BRACKET_LEFT)) {
		reject(subject, Reject_Cause(FAILURE_SYNTAX))
	}
	parse_type(subject)
	if !bool(accept(subject, token.KIND_BRACKET_RIGHT)) {
		reject(subject, Reject_Cause(FAILURE_SYNTAX))
	}
	parse_type(subject)
	close_node(subject)
	return
}

// Parses one channel type. An arrow after the keyword makes it send-only.
func parse_channel_type(subject Parse_State_Handle) {
	Parse_State_Handle_Invariants(subject, "parse_channel_type.subject")
	index := open_node(subject, NODE_CHANNEL_TYPE)
	advance(subject)
	if bool(accept(subject, token.KIND_ARROW)) {
		if index != INDEX_ABSENT {
			subject.Nodes.Values[index].Kind = NODE_CHANNEL_SEND
		}
	}
	parse_type(subject)
	close_node(subject)
	return
}

// Parses one function type and its signature.
func parse_function_type(subject Parse_State_Handle) {
	Parse_State_Handle_Invariants(subject, "parse_function_type.subject")
	open_node(subject, NODE_FUNCTION_TYPE)
	advance(subject)
	parse_signature(subject)
	close_node(subject)
	return
}

// Parses one struct type and its fields.
func parse_structure_type(subject Parse_State_Handle) {
	Parse_State_Handle_Invariants(subject, "parse_structure_type.subject")
	open_node(subject, NODE_STRUCTURE_TYPE)
	advance(subject)
	if !bool(accept(subject, token.KIND_BRACE_LEFT)) {
		reject(subject, Reject_Cause(FAILURE_SYNTAX))
	}
	for range TOKEN_COUNT_MAXIMUM {
		if bool(failed(subject)) {
			break
		}
		skip_comments(subject)
		if bool(accept(subject, token.KIND_SEMICOLON)) {
			continue
		}
		if bool(accept(subject, token.KIND_BRACE_RIGHT)) {
			break
		}
		parse_field(subject)
	}
	close_node(subject)
	return
}

// Parses one struct field, its type, and its optional tag. An identifier ahead of a type opener
// is a field name, and an identifier ahead of anything else is an embedded type.
func parse_field(subject Parse_State_Handle) {
	Parse_State_Handle_Invariants(subject, "parse_field.subject")
	open_node(subject, NODE_FIELD)
	if bool(field_has_name(subject)) {
		parse_field_name(subject)
		for bool(accept(subject, token.KIND_COMMA)) {
			parse_field_name(subject)
		}
	}
	parse_type(subject)
	if at(subject) == token.KIND_STRING {
		open_node(subject, NODE_STRING)
		close_node(subject)
		advance(subject)
	}
	close_node(subject)
	end_line(subject)
	return
}

// Parses one interface type. The linter bans a method set, thus every element is a constraint.
func parse_interface_type(subject Parse_State_Handle) {
	Parse_State_Handle_Invariants(subject, "parse_interface_type.subject")
	open_node(subject, NODE_INTERFACE_TYPE)
	advance(subject)
	if !bool(accept(subject, token.KIND_BRACE_LEFT)) {
		reject(subject, Reject_Cause(FAILURE_SYNTAX))
	}
	for range TOKEN_COUNT_MAXIMUM {
		if bool(failed(subject)) {
			break
		}
		skip_comments(subject)
		if bool(accept(subject, token.KIND_SEMICOLON)) {
			continue
		}
		if bool(accept(subject, token.KIND_BRACE_RIGHT)) {
			break
		}
		if at(subject) == token.KIND_IDENTIFIER {
			if after(subject) == token.KIND_PARENTHESIS_LEFT {
				reject(subject, Reject_Cause(FAILURE_INTERFACE_METHOD))
				break
			}
		}
		parse_constraint(subject)
		end_line(subject)
	}
	close_node(subject)
	return
}

// Parses one brace-delimited statement list. A bare semicolon writes no node, because an empty
// statement carries nothing a reader of the tree could want.
func parse_block(subject Parse_State_Handle) {
	Parse_State_Handle_Invariants(subject, "parse_block.subject")
	open_node(subject, NODE_BLOCK)
	if !bool(accept(subject, token.KIND_BRACE_LEFT)) {
		reject(subject, Reject_Cause(FAILURE_SYNTAX))
	}
	plain := subject.Flags.Plain_Brace
	subject.Flags.Plain_Brace = false
	for range TOKEN_COUNT_MAXIMUM {
		if bool(failed(subject)) {
			break
		}
		skip_comments(subject)
		if bool(accept(subject, token.KIND_SEMICOLON)) {
			continue
		}
		if bool(accept(subject, token.KIND_BRACE_RIGHT)) {
			break
		}
		parse_statement(subject)
	}
	subject.Flags.Plain_Brace = plain
	close_node(subject)
	return
}

// Parses one statement.
func parse_statement(subject Parse_State_Handle) {
	Parse_State_Handle_Invariants(subject, "parse_statement.subject")
	if bool(failed(subject)) {
		return
	}
	switch at(subject) {
	case token.KIND_CONSTANT, token.KIND_VARIABLE, token.KIND_TYPE:
		parse_declaration_statement(subject)
		return
	case token.KIND_RETURN:
		parse_return(subject)
		return
	case token.KIND_IF:
		parse_if(subject)
		return
	case token.KIND_FOR:
		parse_for(subject)
		return
	case token.KIND_SWITCH:
		parse_switch(subject)
		return
	case token.KIND_SELECT:
		parse_select(subject)
		return
	case token.KIND_GO, token.KIND_DEFER:
		parse_launch(subject)
		return
	case token.KIND_BREAK, token.KIND_CONTINUE, token.KIND_GOTO:
		parse_jump(subject)
		return
	case token.KIND_FALLTHROUGH:
		open_node(subject, NODE_FALLTHROUGH)
		close_node(subject)
		advance(subject)
		end_line(subject)
		return
	case token.KIND_BRACE_LEFT:
		parse_block(subject)
		end_line(subject)
		return
	case token.KIND_IDENTIFIER:
		if after(subject) == token.KIND_COLON {
			parse_label(subject)
			return
		}
	}
	parse_clause(subject)
	end_line(subject)
	return
}

// Parses one const, var, or type declaration standing inside a body.
func parse_declaration_statement(subject Parse_State_Handle) {
	Parse_State_Handle_Invariants(subject, "parse_declaration_statement.subject")
	open_node(subject, NODE_DECLARATION_STATEMENT)
	if at(subject) == token.KIND_TYPE {
		parse_type_declaration(subject)
	} else {
		parse_value_declaration(subject)
	}
	close_node(subject)
	return
}

// Parses one labelled statement.
func parse_label(subject Parse_State_Handle) {
	Parse_State_Handle_Invariants(subject, "parse_label.subject")
	open_node(subject, NODE_LABEL)
	parse_name(subject)
	accept(subject, token.KIND_COLON)
	// A label may stand at the end of a block with nothing behind it, which Go reads as a
	// labelled empty statement.
	switch at(subject) {
	case token.KIND_BRACE_RIGHT, token.KIND_END_OF_FILE, token.KIND_CASE,
		token.KIND_DEFAULT:
		close_node(subject)
		return
	}
	parse_statement(subject)
	close_node(subject)
	return
}

// Parses one return statement and its results.
func parse_return(subject Parse_State_Handle) {
	Parse_State_Handle_Invariants(subject, "parse_return.subject")
	// A return that names no value parses and holds no child, because the canonical form
	// writes the values the signature names and a parse that refused one writes nothing.
	open_node(subject, NODE_RETURN)
	advance(subject)
	switch at(subject) {
	case token.KIND_SEMICOLON, token.KIND_BRACE_RIGHT, token.KIND_END_OF_FILE,
		token.KIND_COMMENT:
	default:
		parse_expression_list(subject)
	}
	close_node(subject)
	end_line(subject)
	return
}

// Parses one go or defer statement and the call it carries.
func parse_launch(subject Parse_State_Handle) {
	Parse_State_Handle_Invariants(subject, "parse_launch.subject")
	kind := NODE_GO
	if at(subject) == token.KIND_DEFER {
		kind = NODE_DEFER
	}
	open_node(subject, kind)
	advance(subject)
	parse_expression(subject)
	close_node(subject)
	end_line(subject)
	return
}

// Parses one break, continue, or goto statement and its optional label.
func parse_jump(subject Parse_State_Handle) {
	Parse_State_Handle_Invariants(subject, "parse_jump.subject")
	kind := NODE_BREAK
	switch at(subject) {
	case token.KIND_CONTINUE:
		kind = NODE_CONTINUE
	case token.KIND_GOTO:
		kind = NODE_GOTO
	}
	open_node(subject, kind)
	advance(subject)
	if at(subject) == token.KIND_IDENTIFIER {
		parse_name(subject)
	}
	close_node(subject)
	end_line(subject)
	return
}

// Parses one if statement, its optional initializer, its condition, and its branches.
func parse_if(subject Parse_State_Handle) {
	Parse_State_Handle_Invariants(subject, "parse_if.subject")
	open_node(subject, NODE_IF)
	advance(subject)
	plain := subject.Flags.Plain_Brace
	subject.Flags.Plain_Brace = true
	parse_clause(subject)
	if bool(accept(subject, token.KIND_SEMICOLON)) {
		parse_clause(subject)
	}
	subject.Flags.Plain_Brace = plain
	parse_block(subject)
	if bool(accept(subject, token.KIND_ELSE)) {
		if at(subject) == token.KIND_IF {
			parse_if(subject)
			close_node(subject)
			return
		}
		parse_block(subject)
	}
	close_node(subject)
	end_line(subject)
	return
}

// Parses one for statement: a bare loop, a condition, a three-clause head, or a range clause.
func parse_for(subject Parse_State_Handle) {
	Parse_State_Handle_Invariants(subject, "parse_for.subject")
	open_node(subject, NODE_FOR)
	advance(subject)
	plain := subject.Flags.Plain_Brace
	subject.Flags.Plain_Brace = true
	if at(subject) != token.KIND_BRACE_LEFT {
		parse_for_head(subject)
	}
	subject.Flags.Plain_Brace = plain
	parse_block(subject)
	close_node(subject)
	end_line(subject)
	return
}

// Parses the head of a for statement.
func parse_for_head(subject Parse_State_Handle) {
	Parse_State_Handle_Invariants(subject, "parse_for_head.subject")
	if at(subject) == token.KIND_RANGE {
		open_node(subject, NODE_RANGE)
		advance(subject)
		parse_expression(subject)
		close_node(subject)
		return
	}
	if at(subject) != token.KIND_SEMICOLON {
		parse_clause(subject)
	}
	if !bool(accept(subject, token.KIND_SEMICOLON)) {
		return
	}
	if at(subject) != token.KIND_SEMICOLON {
		parse_clause(subject)
	}
	if !bool(accept(subject, token.KIND_SEMICOLON)) {
		return
	}
	if at(subject) != token.KIND_BRACE_LEFT {
		parse_clause(subject)
	}
	return
}

// Parses one expression or type switch. A type assertion over the type keyword in the head is
// the one mark that tells the two apart, so the kind is settled after the head is read.
func parse_switch(subject Parse_State_Handle) {
	Parse_State_Handle_Invariants(subject, "parse_switch.subject")
	index := open_node(subject, NODE_SWITCH)
	advance(subject)
	plain := subject.Flags.Plain_Brace
	subject.Flags.Plain_Brace = true
	subject.Flags.Type_Assertion = false
	if at(subject) != token.KIND_BRACE_LEFT {
		parse_clause(subject)
		if bool(accept(subject, token.KIND_SEMICOLON)) {
			if at(subject) != token.KIND_BRACE_LEFT {
				parse_clause(subject)
			}
		}
	}
	if bool(subject.Flags.Type_Assertion) {
		if index != INDEX_ABSENT {
			subject.Nodes.Values[index].Kind = NODE_TYPE_SWITCH
		}
	}
	subject.Flags.Plain_Brace = plain
	parse_case_block(subject)
	close_node(subject)
	end_line(subject)
	return
}

// Parses one select statement.
func parse_select(subject Parse_State_Handle) {
	Parse_State_Handle_Invariants(subject, "parse_select.subject")
	open_node(subject, NODE_SELECT)
	advance(subject)
	parse_case_block(subject)
	close_node(subject)
	end_line(subject)
	return
}

// Parses the brace-delimited clause list of a switch or a select statement.
func parse_case_block(subject Parse_State_Handle) {
	Parse_State_Handle_Invariants(subject, "parse_case_block.subject")
	if !bool(accept(subject, token.KIND_BRACE_LEFT)) {
		reject(subject, Reject_Cause(FAILURE_SYNTAX))
		return
	}
	for range TOKEN_COUNT_MAXIMUM {
		if bool(failed(subject)) {
			return
		}
		skip_comments(subject)
		if bool(accept(subject, token.KIND_BRACE_RIGHT)) {
			return
		}
		parse_case(subject)
	}
	return
}

// Parses one case or default clause and the statements it holds.
func parse_case(subject Parse_State_Handle) {
	Parse_State_Handle_Invariants(subject, "parse_case.subject")
	if bool(accept(subject, token.KIND_DEFAULT)) {
		open_node(subject, NODE_DEFAULT)
	} else {
		open_node(subject, NODE_CASE)
		if !bool(accept(subject, token.KIND_CASE)) {
			reject(subject, Reject_Cause(FAILURE_SYNTAX))
			close_node(subject)
			return
		}
		parse_clause(subject)
	}
	if !bool(accept(subject, token.KIND_COLON)) {
		reject(subject, Reject_Cause(FAILURE_SYNTAX))
	}
	parse_case_body(subject)
	close_node(subject)
	return
}

// Parses the statements of one case clause, which run to the next clause or the closing brace.
func parse_case_body(subject Parse_State_Handle) {
	Parse_State_Handle_Invariants(subject, "parse_case_body.subject")
	for range TOKEN_COUNT_MAXIMUM {
		if bool(failed(subject)) {
			return
		}
		skip_comments(subject)
		if bool(accept(subject, token.KIND_SEMICOLON)) {
			continue
		}
		switch at(subject) {
		case token.KIND_CASE, token.KIND_DEFAULT, token.KIND_BRACE_RIGHT,
			token.KIND_END_OF_FILE:
			return
		}
		parse_statement(subject)
	}
	return
}

// Parses one simple statement without its closing semicolon: an expression, an assignment, a
// short declaration, an increment, a decrement, or a channel send. The operator that follows the
// left side names the form, thus the kind is settled after that side is read.
func parse_clause(subject Parse_State_Handle) {
	Parse_State_Handle_Invariants(subject, "parse_clause.subject")
	if bool(failed(subject)) {
		return
	}
	index := open_node(subject, NODE_EXPRESSION_STATEMENT)
	parse_expression_list(subject)
	kind := clause_kind(at(subject))
	if int(kind) != CLAUSE_KIND_MINIMUM {
		if index != INDEX_ABSENT {
			subject.Nodes.Values[index].Kind = Node_Kind(kind)
		}
	}
	parse_clause_tail(subject, kind)
	close_node(subject)
	return
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
	case token.KIND_END_OF_FILE, token.KIND_ILLEGAL, token.KIND_IDENTIFIER,
		token.KIND_TILDE:
		return Clause_Kind(NODE_EXPRESSION_STATEMENT)
	}
	return Clause_Kind(NODE_EXPRESSION_STATEMENT)
}

// Parses the right side of a statement whose operator has already named its form.
func parse_clause_tail(subject Parse_State_Handle, kind Clause_Kind) {
	Parse_State_Handle_Invariants(subject, "parse_clause_tail.subject")
	Clause_Kind_Invariants(kind, "parse_clause_tail.kind")
	if int(kind) == CLAUSE_KIND_MINIMUM {
		return
	}
	advance(subject)
	switch kind {
	case Clause_Kind(NODE_INCREMENT), Clause_Kind(NODE_DECREMENT):
		return
	case Clause_Kind(NODE_SEND):
		parse_expression(subject)
		return
	}
	if at(subject) == token.KIND_RANGE {
		open_node(subject, NODE_RANGE)
		advance(subject)
		parse_expression(subject)
		close_node(subject)
		return
	}
	parse_expression_list(subject)
	return
}

// Parses one comma-separated expression list.
func parse_expression_list(subject Parse_State_Handle) {
	Parse_State_Handle_Invariants(subject, "parse_expression_list.subject")
	parse_expression(subject)
	for bool(accept(subject, token.KIND_COMMA)) {
		if bool(failed(subject)) {
			return
		}
		parse_expression(subject)
	}
	return
}

// Parses one expression with Go's own precedence.
func parse_expression(subject Parse_State_Handle) {
	Parse_State_Handle_Invariants(subject, "parse_expression.subject")
	parse_binary(subject, PRECEDENCE_NONE)
	return
}

// Parses a binary expression whose operators bind tighter than one level. The operand parsed
// before an operator is seen is detached and adopted, which is the one place the tree runs back.
func parse_binary(subject Parse_State_Handle, level Precedence) {
	Parse_State_Handle_Invariants(subject, "parse_binary.subject")
	Precedence_Invariants(level, "parse_binary.level")
	if bool(failed(subject)) {
		return
	}
	parse_unary(subject)
	for range TOKEN_COUNT_MAXIMUM {
		// A sign this dialect states no form for fails where it stands, thus every
		// form that holds an operation answers for it and none of them reads it.
		if bool(refuses_sign(subject)) {
			reject(subject, Reject_Cause(FAILURE_REFUSED_SIGN))
			return
		}
		next := binary_precedence(at(subject))
		if bool(failed(subject)) {
			return
		}
		if int(next) <= int(level) {
			return
		}
		wrap_last(subject, Wrap_Kind(NODE_BINARY))
		advance(subject)
		parse_binary(subject, next)
		close_node(subject)
	}
	return
}

// Reports whether the token the cursor stands on states a sign this dialect refuses: the two signs
// that join conditions, which are nested ifs written flat, and the two order signs that hold
// equality, which a strict comparison states.
func refuses_sign(subject Parse_State_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "refuses_sign.yes") }()
	Parse_State_Handle_Invariants(subject, "refuses_sign.subject")
	switch at(subject) {
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

// Parses one unary expression and its operand.
func parse_unary(subject Parse_State_Handle) {
	Parse_State_Handle_Invariants(subject, "parse_unary.subject")
	if bool(failed(subject)) {
		return
	}
	switch at(subject) {
	case token.KIND_PLUS, token.KIND_MINUS, token.KIND_NOT, token.KIND_EXCLUSIVE_OR,
		token.KIND_STAR, token.KIND_AND, token.KIND_ARROW:
		open_node(subject, NODE_UNARY)
		advance(subject)
		parse_unary(subject)
		close_node(subject)
		return
	}
	parse_primary(subject)
	return
}

// Parses one operand and every suffix that binds to it.
func parse_primary(subject Parse_State_Handle) {
	Parse_State_Handle_Invariants(subject, "parse_primary.subject")
	parse_operand(subject)
	for range TOKEN_COUNT_MAXIMUM {
		if bool(failed(subject)) {
			return
		}
		switch at(subject) {
		case token.KIND_PERIOD:
			parse_selector_suffix(subject)
		case token.KIND_BRACKET_LEFT:
			parse_index_suffix(subject)
		case token.KIND_PARENTHESIS_LEFT:
			parse_call_suffix(subject)
		case token.KIND_BRACE_LEFT:
			if bool(subject.Flags.Plain_Brace) {
				if !bool(last_opens_literal(subject)) {
					return
				}
			}
			parse_composite_suffix(subject)
		default:
			return
		}
	}
	return
}

// Parses one operand: a name, a literal, a grouping, a function literal, or a type that stands
// where an expression can, such as the first argument of a conversion or of make.
func parse_operand(subject Parse_State_Handle) {
	Parse_State_Handle_Invariants(subject, "parse_operand.subject")
	if bool(failed(subject)) {
		return
	}
	switch at(subject) {
	case token.KIND_IDENTIFIER:
		parse_name(subject)
		return
	case token.KIND_INTEGER:
		parse_literal(subject, Literal_Kind(NODE_INTEGER))
		return
	case token.KIND_FLOAT:
		parse_literal(subject, Literal_Kind(NODE_FLOAT))
		return
	case token.KIND_IMAGINARY:
		parse_literal(subject, Literal_Kind(NODE_IMAGINARY))
		return
	case token.KIND_CHARACTER:
		parse_literal(subject, Literal_Kind(NODE_CHARACTER))
		return
	case token.KIND_STRING:
		parse_literal(subject, Literal_Kind(NODE_STRING))
		return
	case token.KIND_PARENTHESIS_LEFT:
		parse_grouping(subject)
		return
	case token.KIND_FUNCTION:
		parse_function_literal(subject)
		return
	case token.KIND_BRACKET_LEFT, token.KIND_MAP, token.KIND_CHANNEL, token.KIND_ARROW,
		token.KIND_STRUCTURE, token.KIND_INTERFACE, token.KIND_ELLIPSIS:
		parse_type(subject)
		return
	}
	reject(subject, Reject_Cause(FAILURE_SYNTAX))
	return
}

// Parses one parenthesised expression. The brace rule of the enclosing control clause stops at
// the parenthesis, because a literal inside a grouping is unambiguous again.
func parse_grouping(subject Parse_State_Handle) {
	Parse_State_Handle_Invariants(subject, "parse_grouping.subject")
	open_node(subject, NODE_PARENTHESIS)
	advance(subject)
	plain := subject.Flags.Plain_Brace
	subject.Flags.Plain_Brace = false
	parse_expression(subject)
	subject.Flags.Plain_Brace = plain
	if !bool(accept(subject, token.KIND_PARENTHESIS_RIGHT)) {
		reject(subject, Reject_Cause(FAILURE_SYNTAX))
	}
	close_node(subject)
	return
}

// Parses one function literal and its body.
func parse_function_literal(subject Parse_State_Handle) {
	Parse_State_Handle_Invariants(subject, "parse_function_literal.subject")
	open_node(subject, NODE_FUNCTION_LITERAL)
	advance(subject)
	parse_signature(subject)
	if at(subject) == token.KIND_BRACE_LEFT {
		plain := subject.Flags.Plain_Brace
		subject.Flags.Plain_Brace = false
		parse_block(subject)
		subject.Flags.Plain_Brace = plain
	}
	close_node(subject)
	return
}

// Parses one selector or type assertion over the expression already read.
func parse_selector_suffix(subject Parse_State_Handle) {
	Parse_State_Handle_Invariants(subject, "parse_selector_suffix.subject")
	if after(subject) != token.KIND_PARENTHESIS_LEFT {
		wrap_last(subject, Wrap_Kind(NODE_SELECTOR))
		advance(subject)
		parse_name(subject)
		close_node(subject)
		return
	}
	wrap_last(subject, Wrap_Kind(NODE_ASSERTION))
	advance(subject)
	advance(subject)
	if bool(accept(subject, token.KIND_TYPE)) {
		subject.Flags.Type_Assertion = true
	} else {
		parse_type(subject)
	}
	if !bool(accept(subject, token.KIND_PARENTHESIS_RIGHT)) {
		reject(subject, Reject_Cause(FAILURE_SYNTAX))
	}
	close_node(subject)
	return
}

// Parses one index, slice, or generic instantiation over the expression already read. A colon
// inside the brackets makes it a slice and a comma makes it an instantiation, thus the kind is
// settled only after the brackets close.
func parse_index_suffix(subject Parse_State_Handle) {
	Parse_State_Handle_Invariants(subject, "parse_index_suffix.subject")
	wrap_last(subject, Wrap_Kind(index_suffix_kind(subject)))
	advance(subject)
	plain := subject.Flags.Plain_Brace
	subject.Flags.Plain_Brace = false
	for range TOKEN_COUNT_MAXIMUM {
		if bool(failed(subject)) {
			break
		}
		if bool(accept(subject, token.KIND_BRACKET_RIGHT)) {
			break
		}
		if bool(accept(subject, token.KIND_COLON)) {
			continue
		}
		if bool(accept(subject, token.KIND_COMMA)) {
			continue
		}
		parse_expression(subject)
	}
	subject.Flags.Plain_Brace = plain
	close_node(subject)
	return
}

// Reads ahead to the bracket that closes the suffix at the cursor and names what it opens. A
// colon inside makes a slice and a comma makes an instantiation, so the class is known before
// the node is taken rather than written over it afterwards.
func index_suffix_kind(subject Parse_State_Handle) (kind Suffix_Kind) {
	defer func() { Suffix_Kind_Invariants(kind, "index_suffix_kind.kind") }()
	Parse_State_Handle_Invariants(subject, "index_suffix_kind.subject")
	depth := 0
	kind = Suffix_Kind(NODE_INDEX)
	position := int(subject.Token_Cursors.Position)
	count := int(subject.Token_Cursors.Count)
	for position < count {
		switch subject.Tokens.Values[position].Kind {
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
		position++
	}
	return kind
}

// Parses one call and its arguments over the expression already read.
func parse_call_suffix(subject Parse_State_Handle) {
	Parse_State_Handle_Invariants(subject, "parse_call_suffix.subject")
	wrap_last(subject, Wrap_Kind(NODE_CALL))
	advance(subject)
	plain := subject.Flags.Plain_Brace
	subject.Flags.Plain_Brace = false
	for range TOKEN_COUNT_MAXIMUM {
		if bool(failed(subject)) {
			break
		}
		skip_comments(subject)
		if bool(accept(subject, token.KIND_PARENTHESIS_RIGHT)) {
			break
		}
		parse_expression(subject)
		if bool(accept(subject, token.KIND_ELLIPSIS)) {
			open_node(subject, NODE_ELLIPSIS)
			close_node(subject)
		}
		accept(subject, token.KIND_COMMA)
		accept(subject, token.KIND_SEMICOLON)
	}
	subject.Flags.Plain_Brace = plain
	close_node(subject)
	return
}

// Parses one composite literal over the type already read.
func parse_composite_suffix(subject Parse_State_Handle) {
	Parse_State_Handle_Invariants(subject, "parse_composite_suffix.subject")
	wrap_last(subject, Wrap_Kind(NODE_COMPOSITE))
	parse_composite_body(subject)
	close_node(subject)
	return
}

// Parses the brace-delimited element list of a composite literal.
func parse_composite_body(subject Parse_State_Handle) {
	Parse_State_Handle_Invariants(subject, "parse_composite_body.subject")
	if !bool(accept(subject, token.KIND_BRACE_LEFT)) {
		reject(subject, Reject_Cause(FAILURE_SYNTAX))
		return
	}
	plain := subject.Flags.Plain_Brace
	subject.Flags.Plain_Brace = false
	for range TOKEN_COUNT_MAXIMUM {
		if bool(failed(subject)) {
			break
		}
		skip_comments(subject)
		if bool(accept(subject, token.KIND_BRACE_RIGHT)) {
			break
		}
		parse_element(subject)
		accept(subject, token.KIND_COMMA)
		accept(subject, token.KIND_SEMICOLON)
	}
	subject.Flags.Plain_Brace = plain
	return
}

// Parses one element of a composite literal. An element may leave its type out, which is why a
// brace can open an element of its own.
func parse_element(subject Parse_State_Handle) {
	Parse_State_Handle_Invariants(subject, "parse_element.subject")
	parse_element_value(subject)
	if at(subject) != token.KIND_COLON {
		return
	}
	wrap_last(subject, Wrap_Kind(NODE_KEY_VALUE))
	advance(subject)
	parse_element_value(subject)
	close_node(subject)
	return
}

// Parses one element value, which is either a nested literal body or an expression.
func parse_element_value(subject Parse_State_Handle) {
	Parse_State_Handle_Invariants(subject, "parse_element_value.subject")
	if at(subject) != token.KIND_BRACE_LEFT {
		parse_expression(subject)
		return
	}
	open_node(subject, NODE_COMPOSITE)
	parse_composite_body(subject)
	close_node(subject)
	return
}

// Takes one literal token as a node of its own.
func parse_literal(subject Parse_State_Handle, kind Literal_Kind) {
	Parse_State_Handle_Invariants(subject, "parse_literal.subject")
	Literal_Kind_Invariants(kind, "parse_literal.kind")
	open_node(subject, Node_Kind(kind))
	close_node(subject)
	advance(subject)
}

// Reports whether the node just parsed is a type that opens a composite literal beyond doubt. A
// control clause bars a literal whose type is a bare name, because `for x {` opens a body and
// never a literal of type x, but `for range []T{...} {` reads only one way and Go admits it.
func last_opens_literal(subject Parse_State_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "last_opens_literal.yes") }()
	Parse_State_Handle_Invariants(subject, "last_opens_literal.subject")
	depth := subject.Node_Cursors.Depth
	if depth == 0 {
		return false
	}
	if int(depth) > DEPTH_MAXIMUM {
		return false
	}
	last := subject.Last_Children.Values[depth-1]
	switch subject.Nodes.Values[last].Kind {
	case NODE_SLICE_TYPE, NODE_ARRAY_TYPE, NODE_MAP_TYPE, NODE_STRUCTURE_TYPE:
		return true
	}
	return false
}

// Failure reports why the parse stopped, or FAILURE_NONE when it ran clean.
func Failure(subject Parse_State_Handle) (code Failure_Code) {
	defer func() { Failure_Code_Invariants(code, "failure.code") }()
	Parse_State_Handle_Invariants(subject, "failure.subject")
	return subject.Causes.Cause
}

// Failure_Token reports the run position of the token the parse refused. It carries no meaning
// while Failure reads FAILURE_NONE.
func Failure_Token(subject Parse_State_Handle) (index Token_Index) {
	defer func() { Token_Index_Invariants(index, "failure_token.index") }()
	Parse_State_Handle_Invariants(subject, "failure_token.subject")
	return Token_Index(subject.Token_Cursors.Failure)
}

// Failure_Message reads one failure code as a sentence that says what to do about it. Each
// sentence is a constant, thus naming a failure allocates nothing.
func Failure_Message(code Failure_Code) (text Message) {
	defer func() { Message_Invariants(text, "failure_message.text") }()
	Failure_Code_Invariants(code, "failure_message.code")
	switch code {
	case FAILURE_SYNTAX:
		return "Fix the syntax at this token."
	case FAILURE_DECLARATION_GROUP:
		return "Write one declaration for each name and drop the parentheses."
	case FAILURE_INTERFACE_METHOD:
		return "Declare no interface; a generic constraint is the one exception."
	case FAILURE_WIDE_IDENTIFIER:
		return "Spell the name with ASCII letters, digits, and underscores."
	case FAILURE_IOTA:
		return "Spell the value out; iota ties it to declaration order."
	case FAILURE_PRIVATE_FIELD:
		return "Begin the struct field name with a capital letter."
	case FAILURE_REFUSED_SIGN:
		return "Write nested if statements, or a strict comparison."
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
	case FAILURE_PACKAGE_CLAUSE:
		return "Open the file with the package keyword and one name."
	case FAILURE_TOKEN_COUNT:
		return "Split the file; it holds more tokens than one parse admits."
	case FAILURE_NODE_COUNT:
		return "Split the file; it holds more syntax than one parse arena admits."
	case FAILURE_NESTING_DEPTH:
		return "Flatten the nesting; it runs deeper than one parse admits."
	}
	return "Read the tree."
}

// Names a refused byte at or above 128 for what it is. The scanner calls such a byte illegal,
// and only the source tells a stray byte apart from a name Go would have taken, thus the reading
// happens here where the source is still in reach.
func name_wide_identifier(subject Parse_State_Handle, source token.Source) {
	Parse_State_Handle_Invariants(subject, "name_wide_identifier.subject")
	token.Source_Invariants(source, "name_wide_identifier.source")
	if subject.Causes.Cause != FAILURE_SYNTAX {
		return
	}
	one := subject.Tokens.Values[subject.Token_Cursors.Failure]
	if one.Kind != token.KIND_ILLEGAL {
		return
	}
	if int(one.Offset) >= len(source) {
		return
	}
	if source[one.Offset] < WIDE_BYTE_MINIMUM {
		return
	}
	subject.Causes.Cause = FAILURE_WIDE_IDENTIFIER
}

// Reports whether the name after the comma opens a qualified or generic type rather than one
// more parameter name. A run such as "string, transform.Transformer" is a run of types, and only
// the token past the second name tells it from a run of names sharing one type.
func qualifies_after(subject Parse_State_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "qualifies_after.yes") }()
	Parse_State_Handle_Invariants(subject, "qualifies_after.subject")
	position := subject.Token_Cursors.Position
	if int(position)+2 >= int(subject.Token_Cursors.Count) {
		return false
	}
	switch subject.Tokens.Values[position+2].Kind {
	case token.KIND_PERIOD, token.KIND_BRACKET_LEFT:
		return true
	}
	return false
}

// Takes one struct field name. It stands apart from a plain identifier so a later pass tells a
// field name from the type behind it without guessing which child of the field is which.
func parse_field_name(subject Parse_State_Handle) {
	Parse_State_Handle_Invariants(subject, "parse_field_name.subject")
	if at(subject) != token.KIND_IDENTIFIER {
		reject(subject, Reject_Cause(FAILURE_SYNTAX))
		return
	}
	open_node(subject, NODE_FIELD_NAME)
	close_node(subject)
	accept(subject, token.KIND_IDENTIFIER)
}

// Names the banned identifiers a clean parse still holds. Both read the source, which no parse
// step carries, thus the pass runs here where the source is still in reach. The walk to the name
// a field embeds stays inline, because a step that took a slot number would owe every slot at
// one call site and a field never occupies the first of them.
func name_banned_identifier(subject Parse_State_Handle, source token.Source) {
	Parse_State_Handle_Invariants(subject, "name_banned_identifier.subject")
	token.Source_Invariants(source, "name_banned_identifier.source")
	if subject.Causes.Cause != FAILURE_NONE {
		return
	}
	count := subject.Node_Cursors.Count
	for slot := INDEX_FIRST; slot < Index(count); slot++ {
		target := INDEX_ABSENT
		if subject.Nodes.Values[slot].Kind == NODE_FIELD_NAME {
			target = slot
		}
		if subject.Nodes.Values[slot].Kind == NODE_FIELD {
			walk := Index(subject.Nodes.Values[slot].First_Child)
			for range EMBEDDED_WALK_MAXIMUM {
				if walk == INDEX_ABSENT {
					break
				}
				if subject.Nodes.Values[walk].Kind == NODE_FIELD_NAME {
					break
				}
				if subject.Nodes.Values[walk].Kind == NODE_IDENTIFIER {
					target = walk
					break
				}
				if subject.Nodes.Values[walk].Kind == NODE_SELECTOR {
					head := subject.Nodes.Values[walk].First_Child
					walk = Index(subject.Nodes.Values[head].Next)
					continue
				}
				walk = Index(subject.Nodes.Values[walk].First_Child)
			}
		}
		if target != INDEX_ABSENT {
			one := subject.Tokens.Values[subject.Nodes.Values[target].Token]
			name := source[one.Offset : int(one.Offset)+int(one.Size)]
			// The test stays inline, because a step that took the name would owe every
			// source length at one call site and a field name is never a whole file.
			lower := false
			if name[0] >= 'a' {
				lower = name[0] <= 'z'
			}
			if lower {
				// Cause first: fail names the token the cursor stands on,
				// and the name that broke the rule sits far behind it.
				fail(subject, Cause(FAILURE_PRIVATE_FIELD))
				subject.Token_Cursors.Failure = Refusal_Token(
					subject.Nodes.Values[target].Token)
				return
			}
		}
		if subject.Nodes.Values[slot].Kind != NODE_IDENTIFIER {
			continue
		}
		one := subject.Tokens.Values[subject.Nodes.Values[slot].Token]
		constant := source[one.Offset : int(one.Offset)+int(one.Size)]
		if string(constant) == IOTA_NAME {
			fail(subject, Cause(FAILURE_IOTA))
			subject.Token_Cursors.Failure = Refusal_Token(
				subject.Nodes.Values[slot].Token)
			return
		}
	}
}

// Reports whether the field at the cursor spells its name out. An identifier ahead of a bracket
// reads two ways: a name before an array or slice type, or a generic type the field embeds. The
// token behind the closing bracket tells them apart, because a field that ends there embedded a
// type and one that reads on named it.
func field_has_name(subject Parse_State_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "field_has_name.yes") }()
	Parse_State_Handle_Invariants(subject, "field_has_name.subject")
	if at(subject) != token.KIND_IDENTIFIER {
		return false
	}
	if after(subject) == token.KIND_COMMA {
		return true
	}
	if after(subject) != token.KIND_BRACKET_LEFT {
		return opens_type(after(subject))
	}
	return !closes_field(subject)
}

// Reports whether the bracket run after the name at the cursor ends the field.
func closes_field(subject Parse_State_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "closes_field.yes") }()
	Parse_State_Handle_Invariants(subject, "closes_field.subject")
	depth := 0
	position := int(subject.Token_Cursors.Position) + 1
	count := int(subject.Token_Cursors.Count)
	for position < count {
		switch subject.Tokens.Values[position].Kind {
		case token.KIND_BRACKET_LEFT:
			depth++
		case token.KIND_BRACKET_RIGHT:
			depth--
			if depth == 0 {
				// Inline, because a step taking the kind would owe the end
				// of file at one call site and a closing bracket always draws
				// an inserted semicolon ahead of it.
				switch subject.Tokens.Values[position+1].Kind {
				case token.KIND_SEMICOLON, token.KIND_BRACE_RIGHT,
					token.KIND_STRING, token.KIND_COMMENT:
					return true
				}
				return false
			}
		}
		position++
	}
	return false
}

// Names an import bound to the blank name. The name reads from the source, which no parse step
// carries, thus the pass runs here where the source is still in reach.
func name_blank_import(subject Parse_State_Handle, source token.Source) {
	Parse_State_Handle_Invariants(subject, "name_blank_import.subject")
	token.Source_Invariants(source, "name_blank_import.source")
	if subject.Causes.Cause != FAILURE_NONE {
		return
	}
	count := subject.Node_Cursors.Count
	for slot := INDEX_FIRST; slot < Index(count); slot++ {
		if subject.Nodes.Values[slot].Kind != NODE_IMPORT_NAME {
			continue
		}
		one := subject.Tokens.Values[subject.Nodes.Values[slot].Token]
		name := source[one.Offset : int(one.Offset)+int(one.Size)]
		if string(name) != BLANK_NAME {
			continue
		}
		fail(subject, Cause(FAILURE_BLANK_IMPORT))
		subject.Token_Cursors.Failure = Refusal_Token(subject.Nodes.Values[slot].Token)
		return
	}
}

// Takes one name a const or var declaration binds. A constant name stands apart, because it
// answers to a case rule of its own and no bare identifier run states which it is.
func parse_value_name(subject Parse_State_Handle, kind Value_Kind) {
	Parse_State_Handle_Invariants(subject, "parse_value_name.subject")
	Value_Kind_Invariants(kind, "parse_value_name.kind")
	if at(subject) != token.KIND_IDENTIFIER {
		reject(subject, Reject_Cause(FAILURE_SYNTAX))
		return
	}
	if uint8(kind) == VALUE_KIND_CONSTANT {
		open_node(subject, NODE_CONSTANT_NAME)
	} else {
		open_node(subject, NODE_IDENTIFIER)
	}
	close_node(subject)
	accept(subject, token.KIND_IDENTIFIER)
}

// Names a constant that is no run of uppercase words joined by single underscores. The name
// reads from the source, which no parse step carries, thus the pass runs here.
func name_constant_case(subject Parse_State_Handle, source token.Source) {
	Parse_State_Handle_Invariants(subject, "name_constant_case.subject")
	token.Source_Invariants(source, "name_constant_case.source")
	if subject.Causes.Cause != FAILURE_NONE {
		return
	}
	count := subject.Node_Cursors.Count
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
			if value >= 'A' {
				if value <= 'Z' {
					continue
				}
			}
			if value >= '0' {
				if value <= '9' {
					continue
				}
			}
			screams = false
		}
		if screams {
			continue
		}
		fail(subject, Cause(FAILURE_CONSTANT_CASE))
		subject.Token_Cursors.Failure = Refusal_Token(subject.Nodes.Values[slot].Token)
		return
	}
}

// Reports whether the open function declares a result. A bare return names everything it owes
// only when nothing is owed, thus this is what tells a naked return from a plain one.
// Names a loop that no clause constrains: a bare for, a three-clause for with every clause
// empty, and a for whose only condition is the true literal. A source that means to run forever
// says so with a range, which is a form of its own and reads as the assertion it is.
func name_bare_loop(subject Parse_State_Handle, source token.Source) {
	Parse_State_Handle_Invariants(subject, "name_bare_loop.subject")
	token.Source_Invariants(source, "name_bare_loop.source")
	if subject.Causes.Cause != FAILURE_NONE {
		return
	}
	count := subject.Node_Cursors.Count
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
		fail(subject, Cause(FAILURE_BARE_LOOP))
		subject.Token_Cursors.Failure = Refusal_Token(subject.Nodes.Values[slot].Token)
		return
	}
}

// Names a signature result that carries no name. A result that names itself holds its name and
// its type, thus a result holding one child alone is a type standing on its own.
func name_unnamed_result(subject Parse_State_Handle) {
	Parse_State_Handle_Invariants(subject, "name_unnamed_result.subject")
	if subject.Causes.Cause != FAILURE_NONE {
		return
	}
	count := subject.Node_Cursors.Count
	for slot := INDEX_FIRST; slot < Index(count); slot++ {
		if subject.Nodes.Values[slot].Kind != NODE_RESULT {
			continue
		}
		child := Index(subject.Nodes.Values[slot].First_Child)
		named := false
		for range TOKEN_COUNT_MAXIMUM {
			if child == INDEX_ABSENT {
				break
			}
			if subject.Nodes.Values[child].Kind == NODE_PARAMETER_NAME {
				named = true
				break
			}
			child = Index(subject.Nodes.Values[child].Next)
		}
		if named {
			continue
		}
		fail(subject, Cause(FAILURE_UNNAMED_RESULT))
		subject.Token_Cursors.Failure = Refusal_Token(subject.Nodes.Values[slot].Token)
		return
	}
}

// Takes one name a signature binds.
func parse_parameter_name(subject Parse_State_Handle) {
	Parse_State_Handle_Invariants(subject, "parse_parameter_name.subject")
	if at(subject) != token.KIND_IDENTIFIER {
		reject(subject, Reject_Cause(FAILURE_SYNTAX))
		return
	}
	open_node(subject, NODE_PARAMETER_NAME)
	close_node(subject)
	accept(subject, token.KIND_IDENTIFIER)
}

// Reads every name of the open node back to an identifier, which is what they were.
func unname_children(subject Parse_State_Handle) {
	Parse_State_Handle_Invariants(subject, "unname_children.subject")
	depth := subject.Node_Cursors.Depth
	if depth == 0 {
		return
	}
	if int(depth) > DEPTH_MAXIMUM {
		return
	}
	child := Index(subject.Nodes.Values[subject.Parents.Values[depth-1]].First_Child)
	for range TOKEN_COUNT_MAXIMUM {
		if child == INDEX_ABSENT {
			return
		}
		if subject.Nodes.Values[child].Kind == NODE_PARAMETER_NAME {
			subject.Nodes.Values[child].Kind = NODE_IDENTIFIER
		}
		child = Index(subject.Nodes.Values[child].Next)
	}
}
