// Package printer writes the canonical form of a parse tree. It exists because a linter that
// judges whether a file is clean needs the form the formatter would write, and reading that form
// out of the tree costs nothing the caller did not already pay for. Every store is fixed storage
// the caller owns, thus a print allocates nothing.
package printer

import (
	"local/james-orcales/shared/go/ast"
	"local/james-orcales/shared/go/token"
	"local/james-orcales/shared/sim/aver/default"
)

// FORM_SIZE_MINIMUM admits no output.
const FORM_SIZE_MINIMUM = 0

// FORM_SIZE_MAXIMUM holds the widest form one source prints to. Indentation is the one thing a
// print adds to a source, thus two source widths hold every form a source of the admitted width
// can take.
const FORM_SIZE_MAXIMUM = token.SOURCE_SIZE_MAXIMUM * 2

// DEPTH_MAXIMUM caps the nest one print walks. A run of one sign nests one node inside the
// next, thus a tree stands deeper than the parse ever holds open, and the print refuses a tree
// nested deeper than the walk holds.
const DEPTH_MAXIMUM = 4096

// DEPTH_DEEPEST is the deepest slot of the walk.
const DEPTH_DEEPEST = DEPTH_MAXIMUM - 1

// WIDTH_MAXIMUM is the widest width one measure answers with. A part wider than it aligns as a
// part that wide, which no line a person reads reaches.
const WIDTH_MAXIMUM = 1 << 16

// COLUMN_WIDTH_MAXIMUM is the widest line the canonical form writes. A form that would run past it
// states one part to a line instead, and a line that holds one token past it stands as it is,
// because no break of any kind shortens one word.
const COLUMN_WIDTH_MAXIMUM = 100

// TAB_WIDTH is the columns one tab stands in, which is what a reader of the form sees and what the
// linter counts. A tab stands in the form only inside a string or a note the author wrote.
const TAB_WIDTH = 8

// INDENT_SYMBOL is the byte one line opens with for each depth the walk stands under.
const INDENT_SYMBOL = Symbol(32)

// INDENT_WIDTH is how many such bytes one line opens at for each depth the walk stands under,
// which is also the columns they stand in.
const INDENT_WIDTH = 2

// CHARACTER_TAIL_MARK marks the bytes of one character that stand behind its first, which state
// the two high bits of a continuation and stand in no column of their own.
const CHARACTER_TAIL_MARK = 0x80

// CHARACTER_TAIL_MASK reads the two high bits one byte states.
const CHARACTER_TAIL_MASK = 0xC0

// CASE_WIDTH_MAXIMUM is the widest head a case writes on one line. A head wider than this reads
// no better closed up than broken, thus the print keeps the lines the author wrote for it.
const CASE_WIDTH_MAXIMUM = 60

// POSITION_MINIMUM is the first token of the run.
const POSITION_MINIMUM = 0

// COUNT_MAXIMUM caps every counter one print keeps, and the form is the widest thing it counts.
const COUNT_MAXIMUM = FORM_SIZE_MAXIMUM

// COUNT_WRITTEN holds how many bytes the print wrote.
const COUNT_WRITTEN = 0

// COUNT_DEPTH holds how deep the walk stands.
const COUNT_DEPTH = 1

// COUNT_INDENT holds how many tabs a line opens with.
const COUNT_INDENT = 2

// COUNT_WIDTH holds the widest name the alignment run holds.
const COUNT_WIDTH = 3

// COUNT_NEST holds how deep inside another form the expression the print stands on stands. A
// statement opens at one, and the canonical form closes up the signs of an operation that stands
// deeper than that.
const COUNT_NEST = 4

// COUNT_COLUMN holds the column the comments of the run the print stands in open at. Zero names
// a print stand in no run of comments.
const COUNT_COLUMN = 5

// COUNT_COLUMN_INDENT holds the tabs the form the print stands in opened its line at.
const COUNT_COLUMN_INDENT = 6

// COUNT_DECLARED holds the byte of the form the run of declarations the print stands on opened
// at, which is where the empty line between two spread declarations stands.
const COUNT_DECLARED = 7

// COUNT_SPLIT holds the binding of the run of operations the print breaks at every sign, which is
// what says whether an operation inside it stands in that run or opens one of its own.
const COUNT_SPLIT = 8

// COUNT_COLUMNS holds the columns the line the print stands on already holds. The print counts
// them as it writes, thus a read of the line it stands on costs no walk of that line.
const COUNT_COLUMNS = 9

// COUNT_CLOSER_NODE holds the node whose closing bracket the print read last, counted from one
// so that a printer that read none holds nothing at all.
const COUNT_CLOSER_NODE = 10

// COUNT_CLOSER_PLACE holds the position that read found.
const COUNT_CLOSER_PLACE = 11

// COUNT_LINE holds how many lines the print closed, which states whether a form the print wrote
// stands over lines of its own.
const COUNT_LINE = 12

// COUNT_DECLARED_COUNT holds how many declarations of the file the print closed, which states
// that it wrote each one the file holds and wrote none of them twice.
const COUNT_DECLARED_COUNT = 13

// COUNT_NOTE holds how many notes the print wrote, which states that the form holds every note
// the author wrote.
const COUNT_NOTE = 14

// COUNT_CLAUSE holds how many package clauses the print wrote, which one file states once.
const COUNT_CLAUSE = 15

// COUNT_FIELD holds how many fields the print wrote, which states that the form holds every
// field the structures of the file state.
const COUNT_FIELD = 16

// COUNT_PARAMETER holds how many parameters and results the print wrote, which states that the
// form holds every one the signatures of the file state.
const COUNT_PARAMETER = 17

// COUNT_CASE holds how many cases the print wrote, which states that the form holds every case
// the switches and the selects of the file state.
const COUNT_CASE = 18

// COUNT_STATEMENT holds how many statements the print wrote, which states that the form holds
// every statement the bodies of the file state.
const COUNT_STATEMENT = 19

// COUNT_PAIR holds how many pairs the print wrote, which states that the form holds every pair
// the literals of the file state.
const COUNT_PAIR = 20

// COUNT_TERM holds how many terms the print wrote, which states that the form holds every term
// the constraints of the file state.
const COUNT_TERM = 21

// COUNT_LABEL holds how many labels the print wrote, which states that the form holds every one
// the statements of the file state.
const COUNT_LABEL = 22

// COUNT_UNION holds how many unions the print wrote, which states that the form holds every one
// the constraints of the file state.
const COUNT_UNION = 23

// COUNT_TYPE_PARAMETER holds how many type parameters the print wrote, which states that the
// form holds every one the signatures of the file state.
const COUNT_TYPE_PARAMETER = 24

// COUNT_BLOCK holds how many blocks the print wrote, which states that the form holds every body
// the file states.
const COUNT_BLOCK = 25

// COUNT_FORM holds how many literals, selectors, assertions, slices, function literals, and
// ranges the print wrote, which the canonical form states wherever the tree holds them.
const COUNT_FORM = 26

// COUNT_OPERATION holds how many operations the print wrote, which the canonical form states
// wherever the tree holds one.
const COUNT_OPERATION = 27

// COUNT_SELECTOR holds how many selectors the print wrote.
const COUNT_SELECTOR = 28

// COUNT_ASSERTION holds how many assertions the print wrote.
const COUNT_ASSERTION = 29

// COUNT_LITERAL holds how many composite literals the print wrote.
const COUNT_LITERAL = 30

// COUNT_SLICE holds how many slice expressions the print wrote.
const COUNT_SLICE = 31

// COUNT_FUNCTION holds how many function literals the print wrote.
const COUNT_FUNCTION = 32

// COUNT_RANGE holds how many ranges the print wrote.
const COUNT_RANGE = 33

// COUNT_DEFAULT holds how many default cases the print wrote, which a switch states apart from
// the cases that name the values they stand for.
const COUNT_DEFAULT = 34

// COUNT_RESULT holds how many results the print wrote, which a signature states apart from the
// parameters it binds.
const COUNT_RESULT = 35

// COUNT_IMPORT holds how many imports the print wrote, which a file states apart from the other
// declarations it holds.
const COUNT_IMPORT = 36

// COUNT_CONSTANT holds how many constant declarations the print wrote.
const COUNT_CONSTANT = 37

// COUNT_VARIABLE holds how many variable declarations the print wrote.
const COUNT_VARIABLE = 38

// COUNT_TYPE holds how many type declarations the print wrote.
const COUNT_TYPE = 39

// COUNT_FUNCTION_DECLARATION holds how many function declarations the print wrote.
const COUNT_FUNCTION_DECLARATION = 40

// COUNT_IF holds how many checks the print wrote, which a body states apart from the loops and
// the switches standing in it.
const COUNT_IF = 41

// COUNT_FOR holds how many loops the print wrote.
const COUNT_FOR = 42

// COUNT_SWITCH holds how many switches of values the print wrote.
const COUNT_SWITCH = 43

// COUNT_TYPE_SWITCH holds how many switches of types the print wrote.
const COUNT_TYPE_SWITCH = 44

// COUNT_SELECT holds how many selects the print wrote.
const COUNT_SELECT = 45

// COUNT_RETURN holds how many returns the print wrote, which a body states apart from the other
// words that close a run of statements.
const COUNT_RETURN = 46

// COUNT_GO holds how many go statements the print wrote.
const COUNT_GO = 47

// COUNT_DEFER holds how many defer statements the print wrote.
const COUNT_DEFER = 48

// COUNT_BREAK holds how many breaks the print wrote.
const COUNT_BREAK = 49

// COUNT_CONTINUE holds how many continues the print wrote.
const COUNT_CONTINUE = 50

// COUNT_GOTO holds how many gotos the print wrote.
const COUNT_GOTO = 51

// COUNT_FALLTHROUGH holds how many fallthroughs the print wrote.
const COUNT_FALLTHROUGH = 52

// COUNT_ASSIGN holds how many assignments of a value the print wrote, which a body states apart
// from the bindings that name what they bind.
const COUNT_ASSIGN = 53

// COUNT_DEFINE holds how many bindings of a name the print wrote.
const COUNT_DEFINE = 54

// COUNT_OPERATION_ASSIGN holds how many assignments of an operation the print wrote.
const COUNT_OPERATION_ASSIGN = 55

// COUNT_SEND holds how many sends the print wrote.
const COUNT_SEND = 56

// COUNT_INCREMENT holds how many steps up the print wrote.
const COUNT_INCREMENT = 57

// COUNT_DECREMENT holds how many steps down the print wrote.
const COUNT_DECREMENT = 58

// COUNT_EXPRESSION_STATEMENT holds how many statements of one expression the print wrote.
const COUNT_EXPRESSION_STATEMENT = 59

// COUNT_DECLARATION_STATEMENT holds how many statements of one declaration the print wrote.
const COUNT_DECLARATION_STATEMENT = 60

// COUNT_FIELD_NAME holds how many names of fields the print wrote, which states that the form
// binds every one a structure states.
const COUNT_FIELD_NAME = 61

// COUNT_PARAMETER_NAME holds how many names of parameters the print wrote, which states that the
// form binds every one a signature states.
const COUNT_PARAMETER_NAME = 62

// COUNT_TYPE_FORM holds how many types the print wrote of the kinds that spell a form of their
// own: the pointers, the slices, the arrays, the maps, the channels and the rest.
const COUNT_TYPE_FORM = 63

// COUNT_TYPE_DROPPED holds how many of those types the print dropped, which it does where the
// value a declaration binds names the type the author wrote ahead of it.
const COUNT_TYPE_DROPPED = 64

// COUNT_RECEIVER holds how many receivers the print wrote, which a method states ahead of its
// name.
const COUNT_RECEIVER = 65

// COUNT_GROUP holds how many groupings the print wrote, which the author states as a value
// between brackets.
const COUNT_GROUP = 66

// COUNT_UNARY holds how many signs of one value the print wrote, which stand ahead of the value
// they bind.
const COUNT_UNARY = 67

// COUNT_CALL holds how many calls the print wrote.
const COUNT_CALL = 68

// COUNT_INDEX holds how many reads of one slot the print wrote, the reads that name a type
// parameter among them.
const COUNT_INDEX = 69

// COUNT_CALL_DROPPED holds how many calls the print dropped, which it does where the call names
// the length of the value the brackets around it close.
const COUNT_CALL_DROPPED = 70

// COUNT_POINTER holds how many pointer types the print read, the ones it wrote and the ones it
// dropped alike.
const COUNT_POINTER = 71

// COUNT_SLICE_TYPE holds how many slice types the print read.
const COUNT_SLICE_TYPE = 72

// COUNT_ELLIPSIS holds how many spreads the print read, which a signature states ahead of the
// type its last parameter binds.
const COUNT_ELLIPSIS = 73

// COUNT_CHANNEL holds how many channel types the print read, the ones one way stands on among
// them.
const COUNT_CHANNEL = 74

// COUNT_ARRAY holds how many array types the print read.
const COUNT_ARRAY = 75

// COUNT_MAP holds how many map types the print read.
const COUNT_MAP = 76

// COUNT_FUNCTION_TYPE holds how many function types the print read.
const COUNT_FUNCTION_TYPE = 77

// COUNT_STRUCTURE holds how many structure types the print read.
const COUNT_STRUCTURE = 78

// COUNT_INTERFACE holds how many interface types the print read.
const COUNT_INTERFACE = 79

// COUNT_NAME holds how many names the print wrote, which the author spells as an identifier.
const COUNT_NAME = 80

// COUNT_NAME_DROPPED holds how many names the print dropped, which it does where the form the
// name stands in states the name nowhere.
const COUNT_NAME_DROPPED = 81

// COUNT_MAIN names the function one program enters at, which the file states ahead of every
// other function it declares. The absent slot names none, and names one the file holds where the
// author already wrote it ahead of them.
const COUNT_MAIN = 82

// COUNT_MAIN_PLACE names the first child of the run the first function of the file opens, which
// is where the print writes the function one program enters at.
const COUNT_MAIN_PLACE = 83

// COUNT_SLOT_COUNT is the counter count one printer holds. Every slot counts what the print
// wrote, thus a measure restores each one.
const COUNT_SLOT_COUNT = 84

// INVARIANTS_TAIL is the text the name of a function that states the invariants of a type closes
// with. A type states its invariants in one function of its own name, thus the canonical form
// stands that function under the type it answers for.
const INVARIANTS_TAIL = "_Invariants"

// PAIR_COUNT_MAXIMUM caps the functions of one file that state the invariants of a type. A file
// that states more of them writes the ones past this count where the author wrote them.
const PAIR_COUNT_MAXIMUM = 1024

// IMPORT_COUNT_MAXIMUM caps the imports of one file whose name the print drops. A file that binds
// more names than this states them as the author wrote them, because the storage that holds them
// is fixed and a print allocates nothing.
const IMPORT_COUNT_MAXIMUM = 64

// DEFAULT_TAIL is the element a path closes with where the package it names stands one element
// ahead of it, which is the directory holding the build of a package rather than a package of its
// own.
const DEFAULT_TAIL = "default"

// INTERNAL_TAIL is the other such element, which holds what one component keeps to itself.
const INTERNAL_TAIL = "internal"

// RESULT_COUNT_MAXIMUM caps the result names one signature binds that a return writes for it. A
// signature that binds more states a form no return of this dialect writes, thus a return there
// stands as the author wrote it.
const RESULT_COUNT_MAXIMUM = 16

// FLAG_FULL marks a print that met the end of the storage the caller supplied, or a line that
// carries more comments than the print holds.
const FLAG_FULL = 0

// FLAG_DEEP marks a print that met a tree nested deeper than the walk holds.
const FLAG_DEEP = 1

// FLAG_LINE marks a print that stands at the opening of a line.
const FLAG_LINE = 2

// FLAG_BLANK marks a print that owes one empty line behind the line it stands on.
const FLAG_BLANK = 3

// FLAG_BROKEN marks a statement the author already broke across lines, thus its continuations
// stand one tab deeper however many breaks it holds.
const FLAG_BROKEN = 4

// FLAG_TIGHT marks a body that states no empty line against the brackets that hold it: a function
// body, a body of one statement alone, a literal, and a run of fields.
const FLAG_TIGHT = 5

// FLAG_BODY marks the block one function declaration states as its body, which the block itself
// cannot tell from any other block it holds.
const FLAG_BODY = 6

// FLAG_FLAT marks a print that takes no break the author wrote, which is the print one measure of
// a form runs: a measure answers how wide the form reads on one line, thus it reads no line the
// author closed.
const FLAG_FLAT = 8

// FLAG_APART marks a form the print writes one part to a line because the parts would otherwise
// run past the widest line. A run of pairs reads it: the pairs of such a form each open a line of
// their own however the author wrote them, thus they align in one column.
const FLAG_APART = 10

// FLAG_SPLIT marks a run of one operation the print breaks at every sign it states, which is the
// run that would otherwise write a line past the widest one. A form that opens brackets of its own
// closes the mark, because the values inside those brackets answer for their own lines.
const FLAG_SPLIT = 9

// FLAG_SHORT marks a variable declaration the short sign binds, which is the form the canonical
// one states for a variable of values and no type inside a body.
const FLAG_SHORT = 7

// FLAG_NOTELESS marks a print that reads the width one statement writes. A note stands behind
// the statement it follows rather than inside it, thus a read of that width writes none.
const FLAG_NOTELESS = 12

// FLAG_REFUSED marks a print of a tree the parse refused. Such a tree holds forms the grammar
// states nothing for, thus the form the print writes states what the tree holds and none of the
// rules the canonical form states.
const FLAG_REFUSED = 11

// FLAG_COUNT is the flag count one printer holds.
const FLAG_COUNT = 13

// COMMENT_COUNT_MAXIMUM caps the comments one line carries behind it.
const COMMENT_COUNT_MAXIMUM = 16

// SYMBOL_MINIMUM is the first byte a form can hold.
const SYMBOL_MINIMUM = 0

// SYMBOL_MAXIMUM is the last byte a form can hold.
const SYMBOL_MAXIMUM = 255

// Symbol is one byte of the form a print writes. A source holds any byte inside a literal, thus
// every byte stands as a symbol the print may write.
type Symbol byte

// Symbol_Invariants states every byte one form holds.
func Symbol_Invariants(value Symbol, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), SYMBOL_MINIMUM, SYMBOL_MAXIMUM).
		Ensure()
}

// BASE_NONE names a literal that states no base prefix, which is a decimal or an octal one the
// source opens with a zero alone.
const BASE_NONE Base = 0

// BASE_HEXADECIMAL names the base a literal opening in 0x states.
const BASE_HEXADECIMAL Base = 'x'

// BASE_OCTAL names the base a literal opening in 0o states.
const BASE_OCTAL Base = 'o'

// BASE_BINARY names the base a literal opening in 0b states.
const BASE_BINARY Base = 'b'

// Base names the base one number literal states, in the case the canonical form writes it.
type Base uint8

// Base_Invariants states every base a number literal names of its own.
func Base_Invariants(value Base, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_4_Uint8(uint8(value), uint8(BASE_NONE), uint8(BASE_BINARY),
			uint8(BASE_OCTAL), uint8(BASE_HEXADECIMAL)).
		Ensure()
}

// EXPONENT_NONE names a literal that states no exponent at all.
const EXPONENT_NONE Exponent = 0

// EXPONENT_DECIMAL names the exponent mark a decimal literal states.
const EXPONENT_DECIMAL Exponent = 'E'

// EXPONENT_POWER names the exponent mark a hexadecimal literal states.
const EXPONENT_POWER Exponent = 'P'

// Exponent names the mark one number literal states its exponent behind, in the case the source
// may write it.
type Exponent uint8

// Exponent_Invariants states every exponent mark a number literal states.
func Exponent_Invariants(value Exponent, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(uint8(value), uint8(EXPONENT_NONE), uint8(EXPONENT_DECIMAL),
			uint8(EXPONENT_POWER)).
		Ensure()
}

// Boolean is a true or false report about one print.
type Boolean bool

// Boolean_Invariants states both print reports as obligations.
func Boolean_Invariants(value Boolean, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The print report is true.").
		Ensure()
}

// Form is caller storage one print writes itself into.
type Form []byte

// Form_Invariants states the storage the widest form needs.
func Form_Invariants(value Form, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), FORM_SIZE_MINIMUM, FORM_SIZE_MAXIMUM).
		Ensure()
}

// Form_Count is the byte count one print wrote.
type Form_Count int

// Form_Count_Invariants states the byte count of the widest form.
func Form_Count_Invariants(value Form_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), FORM_SIZE_MINIMUM, FORM_SIZE_MAXIMUM).
		Ensure()
}

// Count is one counter of the print. Every counter lives in a slot, thus no body owes the whole
// counter domain at a step that can only see one part of it.
type Count int

// Count_Invariants states the widest thing one counter counts.
func Count_Invariants(value Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), FORM_SIZE_MINIMUM, COUNT_MAXIMUM).
		Ensure()
}

// Width is the byte count one name spends, which is what an alignment run pads against.
type Width int32

// Width_Invariants states every width a name spends.
func Width_Invariants(value Width, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), FORM_SIZE_MINIMUM, WIDTH_MAXIMUM).
		Ensure()
}

// Node_Stack keeps bounded traversal state in caller storage.
type Node_Stack []ast.Index

// Node_Stack_Invariants fixes traversal depth and prevents growth.
func Node_Stack_Invariants(value Node_Stack, _ aver.Namespace) {
	aver.Always(len(value) == DEPTH_MAXIMUM, "Printer node stack has complete length.")
	aver.Always(cap(value) == DEPTH_MAXIMUM, "Printer node stack cannot grow.")
}

// Measured_Width is the width returned by one measurement.
type Measured_Width Width

// Measured_Width_Invariants states every width a measurement can return.
func Measured_Width_Invariants(value Measured_Width, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), FORM_SIZE_MINIMUM, WIDTH_MAXIMUM).
		Ensure()
}

// PART_WIDTH_MINIMUM is the narrowest part the print pads behind. A name, a key, a type, and the
// code a note follows each write one byte at least, and the parser writes nothing behind a part it
// refused, thus no pad follows an empty part.
const PART_WIDTH_MINIMUM = 1

// Part_Width is the width of one part the print already wrote on the line it pads.
type Part_Width Width

// Part_Width_Invariants states every width of a part the print pads behind.
func Part_Width_Invariants(value Part_Width, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), PART_WIDTH_MINIMUM, WIDTH_MAXIMUM).
		Ensure()
}

// Column_Width is the column at which one measured form begins.
type Column_Width Width

// Column_Width_Invariants states every measured starting column.
func Column_Width_Invariants(value Column_Width, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), FORM_SIZE_MINIMUM, WIDTH_MAXIMUM).
		Ensure()
}

// Line_Width is the width already spent on one line. A form the print writes one part to a line
// closes on a line that holds the bracket closing it alone, thus one column is a width of its own.
type Line_Width Width

// Line_Width_Invariants states every occupied line width.
func Line_Width_Invariants(value Line_Width, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), FORM_SIZE_MINIMUM, WIDTH_MAXIMUM).
		Ensure()
}

// Import_Aliases keeps bounded imported-name tokens in caller storage.
type Import_Aliases []ast.Token_Index

// Import_Aliases_Invariants fixes imported-name storage and prevents growth.
func Import_Aliases_Invariants(value Import_Aliases, _ aver.Namespace) {
	aver.Always(len(value) == IMPORT_COUNT_MAXIMUM, "Import aliases have complete length.")
	aver.Always(cap(value) == IMPORT_COUNT_MAXIMUM, "Import aliases cannot grow.")
}

// Import_Paths keeps bounded import-path tokens in caller storage.
type Import_Paths []ast.Index

// Import_Paths_Invariants fixes import-path storage and prevents growth.
func Import_Paths_Invariants(value Import_Paths, _ aver.Namespace) {
	aver.Always(len(value) == IMPORT_COUNT_MAXIMUM, "Import paths have complete length.")
	aver.Always(cap(value) == IMPORT_COUNT_MAXIMUM, "Import paths cannot grow.")
}

// Pair_Functions keeps the functions that state the invariants of a type in caller storage.
type Pair_Functions []ast.Index

// Pair_Functions_Invariants fixes pair storage and prevents growth.
func Pair_Functions_Invariants(value Pair_Functions, _ aver.Namespace) {
	aver.Always(len(value) == PAIR_COUNT_MAXIMUM, "Pair functions have complete length.")
	aver.Always(cap(value) == PAIR_COUNT_MAXIMUM, "Pair functions cannot grow.")
}

// Pair_Heads keeps the note that opens each such function in caller storage.
type Pair_Heads []ast.Index

// Pair_Heads_Invariants fixes pair-head storage and prevents growth.
func Pair_Heads_Invariants(value Pair_Heads, _ aver.Namespace) {
	aver.Always(len(value) == PAIR_COUNT_MAXIMUM, "Pair heads have complete length.")
	aver.Always(cap(value) == PAIR_COUNT_MAXIMUM, "Pair heads cannot grow.")
}

// Pair_Storage keeps caller-owned pair storage distinct from its view.
type Pair_Storage struct {
	// Functions holds each function of the file that states the invariants of a type.
	Functions Pair_Functions
	// Heads holds the note that opens each of those functions, or the absent slot.
	Heads Pair_Heads
}

// Pair_Storage_Invariants composes pair storage.
func Pair_Storage_Invariants(value Pair_Storage, namespace aver.Namespace) {
	Pair_Functions_Invariants(value.Functions, namespace)
	Pair_Heads_Invariants(value.Heads, namespace)
}

// DIRECTIVE_SIZE_MAXIMUM is the widest directive prefix this printer recognizes.
const DIRECTIVE_SIZE_MAXIMUM = len("export ")

// Comment_Positions keeps bounded trailing comments in caller storage.
type Comment_Positions []ast.Index

// Comment_Positions_Invariants fixes trailing-comment storage and prevents growth.
func Comment_Positions_Invariants(value Comment_Positions, _ aver.Namespace) {
	aver.Always(len(value) == COMMENT_COUNT_MAXIMUM, "Comment positions have complete length.")
	aver.Always(cap(value) == COMMENT_COUNT_MAXIMUM, "Comment positions cannot grow.")
}

// Counts keeps bounded printer cursors in caller storage.
type Counts []Count

// Counts_Invariants fixes cursor storage and prevents growth.
func Counts_Invariants(value Counts, _ aver.Namespace) {
	aver.Always(len(value) == COUNT_SLOT_COUNT, "Printer counts have complete length.")
	aver.Always(cap(value) == COUNT_SLOT_COUNT, "Printer counts cannot grow.")
}

// Flags keeps bounded printer control bits in caller storage.
type Flags []Boolean

// Flags_Invariants fixes control storage and prevents growth.
func Flags_Invariants(value Flags, _ aver.Namespace) {
	aver.Always(len(value) == FLAG_COUNT, "Printer flags have complete length.")
	aver.Always(cap(value) == FLAG_COUNT, "Printer flags cannot grow.")
}

// Node_Stack_Storage keeps the caller-owned traversal stack distinct from its view.
type Node_Stack_Storage struct {
	// Values makes ownership explicit at the package boundary.
	Values Node_Stack
}

// Node_Stack_Storage_Invariants composes traversal storage.
func Node_Stack_Storage_Invariants(value Node_Stack_Storage, namespace aver.Namespace) {
	Node_Stack_Invariants(value.Values, namespace)
}

// Import_Alias_Storage keeps caller-owned alias storage distinct from its view.
type Import_Alias_Storage struct {
	// Values makes ownership explicit at the package boundary.
	Values Import_Aliases
}

// Import_Alias_Storage_Invariants composes import-alias storage.
func Import_Alias_Storage_Invariants(value Import_Alias_Storage, namespace aver.Namespace) {
	Import_Aliases_Invariants(value.Values, namespace)
}

// Import_Path_Storage keeps caller-owned path storage distinct from its view.
type Import_Path_Storage struct {
	// Values makes ownership explicit at the package boundary.
	Values Import_Paths
}

// Import_Path_Storage_Invariants composes import-path storage.
func Import_Path_Storage_Invariants(value Import_Path_Storage, namespace aver.Namespace) {
	Import_Paths_Invariants(value.Values, namespace)
}

// Comment_Position_Storage keeps caller-owned comment storage distinct from its view.
type Comment_Position_Storage struct {
	// Values makes ownership explicit at the package boundary.
	Values Comment_Positions
}

// Comment_Position_Storage_Invariants composes trailing-comment storage.
func Comment_Position_Storage_Invariants(value Comment_Position_Storage, namespace aver.Namespace) {
	Comment_Positions_Invariants(value.Values, namespace)
}

// Count_Storage keeps caller-owned counters distinct from their view.
type Count_Storage struct {
	// Values makes ownership explicit at the package boundary.
	Values Counts
}

// Count_Storage_Invariants composes counter storage.
func Count_Storage_Invariants(value Count_Storage, namespace aver.Namespace) {
	Counts_Invariants(value.Values, namespace)
}

// Flag_Storage keeps caller-owned control bits distinct from their view.
type Flag_Storage struct {
	// Values makes ownership explicit at the package boundary.
	Values Flags
}

// Flag_Storage_Invariants composes control storage.
func Flag_Storage_Invariants(value Flag_Storage, namespace aver.Namespace) {
	Flags_Invariants(value.Values, namespace)
}

// Source_Bytes keeps one copy of the source the print reads, caller-owned and unable to grow. It
// spans the widest source and a counter names how much of it the source fills, thus every step
// reads storage of one width whatever source the caller hands over.
type Source_Bytes []byte

// Source_Bytes_Invariants fixes the source storage to the widest source.
func Source_Bytes_Invariants(value Source_Bytes, _ aver.Namespace) {
	aver.Always(len(value) == token.SOURCE_SIZE_MAXIMUM, "Printer source spans its storage.")
	aver.Always(cap(value) == token.SOURCE_SIZE_MAXIMUM, "Printer source storage cannot grow.")
}

// Source_Storage keeps the source copy composable under printer state.
type Source_Storage struct {
	// Values makes ownership explicit at the package boundary.
	Values Source_Bytes
}

// Source_Storage_Invariants composes the source copy.
func Source_Storage_Invariants(value Source_Storage, namespace aver.Namespace) {
	Source_Bytes_Invariants(value.Values, namespace)
}

// Form_Bytes keeps the form the print writes, caller-owned and unable to grow. It spans the widest
// form and a counter names how much of it the storage the caller supplied admits.
type Form_Bytes []byte

// Form_Bytes_Invariants fixes the form storage to the widest form.
func Form_Bytes_Invariants(value Form_Bytes, _ aver.Namespace) {
	aver.Always(len(value) == FORM_SIZE_MAXIMUM, "Printer form storage has complete length.")
	aver.Always(cap(value) == FORM_SIZE_MAXIMUM, "Printer form storage cannot grow.")
}

// Form_Storage keeps the written form composable under printer state.
type Form_Storage struct {
	// Values makes ownership explicit at the package boundary.
	Values Form_Bytes
}

// Form_Storage_Invariants composes the written form.
func Form_Storage_Invariants(value Form_Storage, namespace aver.Namespace) {
	Form_Bytes_Invariants(value.Values, namespace)
}

// Form_Handle names the form storage a printer holds, or nil where the caller lends the
// destination of each print instead.
type Form_Handle *Form_Storage

// Form_Handle_Invariants composes present form storage.
func Form_Handle_Invariants(value Form_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Form_Storage_Invariants(*value, namespace)
}

// Source_Handle names the source storage a printer holds, or nil where the caller lends the
// source buffer of each print instead.
type Source_Handle *Source_Storage

// Source_Handle_Invariants composes present source storage.
func Source_Handle_Invariants(value Source_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Source_Storage_Invariants(*value, namespace)
}

// Points the form and source storage a caller left absent at the buffers it hands the print: the
// destination, which spans FORM_SIZE_MAXIMUM, and the source buffer, whose capacity spans
// SOURCE_SIZE_MAXIMUM. A caller that owns one buffer of each thus lends them rather than holding
// a second copy of each, and the print copies each onto itself.
func lend(subject Printer_Handle, destination Form, source token.Source) {
	Printer_Handle_Invariants(subject, "lend.subject")
	Form_Invariants(destination, "lend.destination")
	token.Source_Invariants(source, "lend.source")
	if subject.Form == nil {
		subject.Form = &Form_Storage{
			Values: Form_Bytes(destination[:FORM_SIZE_MAXIMUM:FORM_SIZE_MAXIMUM]),
		}
	}
	if subject.Source == nil {
		subject.Source = &Source_Storage{
			Values: Source_Bytes(
				source[:token.SOURCE_SIZE_MAXIMUM:token.SOURCE_SIZE_MAXIMUM]),
		}
	}
}

// RESERVED_NAME_SIZE_MINIMUM is the width of the narrowest reserved note name, the sys word.
const RESERVED_NAME_SIZE_MINIMUM = len("sys ")

// RESERVED_NAME_SIZE_LINE is the width of the line word.
const RESERVED_NAME_SIZE_LINE = len("line ")

// RESERVED_NAME_SIZE_SYSNB is the width of the sysnb word, which the nolint word shares.
const RESERVED_NAME_SIZE_SYSNB = len("sysnb ")

// Reserved_Name is a name the toolchain reserved for a directive of its own.
type Reserved_Name string

// Reserved_Name_Invariants states the four widths the reserved names span.
func Reserved_Name_Invariants(value Reserved_Name, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_4_Int(
			len(value), RESERVED_NAME_SIZE_MINIMUM, RESERVED_NAME_SIZE_LINE,
			RESERVED_NAME_SIZE_SYSNB, DIRECTIVE_SIZE_MAXIMUM).
		Ensure()
}

// Printer is the caller-owned formatting storage.
type Printer struct {
	// Nodes holds the tree slot the walk stands on at each depth.
	Nodes Node_Stack_Storage
	// Aliases holds the name tokens of the imports whose name the print drops.
	Aliases Import_Alias_Storage
	// Pairs holds the functions of the file that state the invariants of a type.
	Pairs Pair_Storage
	// Paths holds the path token of each of those imports, which names the package the forms
	// that read the name now name.
	Paths Import_Path_Storage
	// Comments holds the run position of each comment the line the print stands on carries.
	Comments Comment_Position_Storage
	// Form holds the form the print writes, which it hands to the caller once it ran.
	Form Form_Handle
	// Source holds the source the tree stands for.
	Source Source_Handle
	// Counts holds every counter and cursor the print keeps.
	Counts Count_Storage
	// Flags holds what the print met: full storage, deep nest, and a fresh line.
	Flags Flag_Storage
}

// Printer_Invariants composes every concrete field once.
func Printer_Invariants(value Printer, namespace aver.Namespace) {
	Node_Stack_Storage_Invariants(value.Nodes, namespace)
	Import_Alias_Storage_Invariants(value.Aliases, namespace)
	Pair_Storage_Invariants(value.Pairs, namespace)
	Import_Path_Storage_Invariants(value.Paths, namespace)
	Comment_Position_Storage_Invariants(value.Comments, namespace)
	Form_Handle_Invariants(value.Form, namespace)
	Source_Handle_Invariants(value.Source, namespace)
	Count_Storage_Invariants(value.Counts, namespace)
	Flag_Storage_Invariants(value.Flags, namespace)
}

// Printer_Handle gives caller-owned formatting state one identity.
type Printer_Handle *Printer

// Printer_Handle_Invariants composes present printer storage.
func Printer_Handle_Invariants(value Printer_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Printer_Invariants(*value, namespace)
}

// Print_Result keeps written size and storage outcome at one output boundary.
type Print_Result struct {
	// Count remains available when caller storage fills.
	Count Form_Count
	// OK distinguishes complete forms from bounded prefixes.
	OK Boolean
}

// Print_Result_Invariants composes one print outcome.
func Print_Result_Invariants(value Print_Result, namespace aver.Namespace) {
	Form_Count_Invariants(value.Count, namespace)
	Boolean_Invariants(value.OK, namespace)
}

// Print writes the canonical form of one tree into caller storage and reports whether the whole
// form fit. A tree the parser refused still prints what it holds, because a caller reads more
// from a partial print than from nothing.
func Print(
	subject Printer_Handle, destination Form, tree ast.Parse_State_Handle, source token.Source,
) (result Print_Result) {
	defer func() { Print_Result_Invariants(result, "print.result") }()
	Printer_Handle_Invariants(subject, "print.subject")
	Node_Stack_Storage_Invariants(subject.Nodes, "print.nodes")
	Import_Alias_Storage_Invariants(subject.Aliases, "print.aliases")
	Import_Path_Storage_Invariants(subject.Paths, "print.paths")
	Comment_Position_Storage_Invariants(subject.Comments, "print.comments")
	Form_Handle_Invariants(subject.Form, "print.form")
	Source_Handle_Invariants(subject.Source, "print.source_storage")
	Count_Storage_Invariants(subject.Counts, "print.counts")
	Flag_Storage_Invariants(subject.Flags, "print.flags")
	Form_Invariants(destination, "print.destination")
	ast.Parse_State_Handle_Invariants(tree, "print.tree")
	token.Source_Invariants(source, "print.source")
	lend(subject, destination, source)
	// The print writes into its own storage and hands the form over once it ran, thus every
	// step reads storage of one width whatever the caller supplied.
	copy(subject.Source.Values, source)
	open_counts(subject, tree)
	layout_run(subject, &tree.Syntax_Tree, RULE_FILE, false)
	// The print writes into storage of the widest form whatever the caller hands over, thus a
	// form that runs past the destination reads as the bytes the destination holds of it.
	written := int(written_load(subject.Counts.Values))
	// Every line of the form closes on a line feed, thus the form a caller reads closes on
	// one too.
	if !bool(flag_load(subject.Flags.Values, FLAG_FULL)) {
		if written > 0 {
			aver.Always(subject.Form.Values[written-1] == '\n',
				"The form closes on a line feed.")
		}
	}
	closes_file(subject, &tree.Syntax_Tree)
	// A tree that stands for anything but a file states no form this print writes, thus the
	// print writes nothing at all for it.
	if node_kind(subject, &tree.Syntax_Tree) != ast.NODE_FILE {
		aver.Always(written == 0, "A tree that states no file writes no byte.")
	}
	held := !flag_load(subject.Flags.Values, FLAG_FULL)
	held = held && !flag_load(subject.Flags.Values, FLAG_DEEP)
	held = held && Boolean(written <= len(destination))
	count := min(written, len(destination))
	// A print that states it wrote the form wrote every byte of it into the storage the
	// caller handed over, thus the count it reports names the whole form.
	if bool(held) {
		aver.Always(count == written, "A print that holds reports the whole form it wrote.")
	}
	copy(destination, subject.Form.Values[:count])
	// The caller reads the form out of its own storage, thus the bytes that storage holds
	// close the way the form the print wrote closes.
	if bool(held) {
		if count > 0 {
			aver.Always(destination[count-1] == '\n',
				"The form the caller reads closes on a line feed.")
		}
	}
	return Print_Result{Count: Form_Count(count), OK: held}
}

// Reads how many bytes the print wrote.
func written_load(counts Counts) (written Form_Count) {
	defer func() { Form_Count_Invariants(written, "written_load.written") }()
	Counts_Invariants(counts, "written_load.counts")
	return Form_Count(counts[COUNT_WRITTEN])
}

// Stands how many bytes the print wrote.
func written_store(counts Counts, written Form_Count) {
	Counts_Invariants(counts, "written_store.counts")
	Form_Count_Invariants(written, "written_store.written")
	counts[COUNT_WRITTEN] = Count(written)
}

// DECLARATION_PLACE_MINIMUM is the first byte a declaration opens at: the word "package", a space,
// a name of one byte, and the line feed that closes the clause.
const DECLARATION_PLACE_MINIMUM = 10

// Declaration_Place is the byte of the form a run of declarations opens at.
type Declaration_Place int

// Declaration_Place_Invariants states every byte a run of declarations opens at.
func Declaration_Place_Invariants(value Declaration_Place, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), DECLARATION_PLACE_MINIMUM, FORM_SIZE_MAXIMUM).
		Ensure()
}

// Reads the byte of the form the run of declarations the print stands on opened at.
func declared_load(counts Counts) (declared Declaration_Place) {
	defer func() { Declaration_Place_Invariants(declared, "declared_load.declared") }()
	Counts_Invariants(counts, "declared_load.counts")
	return Declaration_Place(counts[COUNT_DECLARED])
}

// Stands the byte of the form the run of declarations the print stands on opened at.
func declared_store(counts Counts, declared Declaration_Place) {
	Counts_Invariants(counts, "declared_store.counts")
	Declaration_Place_Invariants(declared, "declared_store.declared")
	counts[COUNT_DECLARED] = Count(declared)
}

// NOTE_COLUMN_HOLE is the narrowest column no run of notes opens at. A run opens behind the widest
// line it holds, and every line of a run holds the bytes its line opens at, a byte of the form,
// and the space that parts that byte from the note.
const NOTE_COLUMN_HOLE = 1

// NOTE_COLUMN_HOLE_LAST is the widest column no run of notes opens at, which is the last column
// the bytes one line opens at stand in. A note standing there would stand inside the indent.
const NOTE_COLUMN_HOLE_LAST = INDENT_WIDTH

// Note_Column is the column the notes of one run open at. Zero names no run.
type Note_Column Width

// Note_Column_Invariants states every column a run of notes opens at.
func Note_Column_Invariants(value Note_Column, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Holed_Int32(
			int32(value), FORM_SIZE_MINIMUM, WIDTH_MAXIMUM,
			NOTE_COLUMN_HOLE, NOTE_COLUMN_HOLE+1, NOTE_COLUMN_HOLE_LAST,
			NOTE_COLUMN_HOLE_LAST,
		).
		Ensure()
}

// Reads the column the notes of the run the print stands in open at. Zero names no run.
func column_load(counts Counts) (column Note_Column) {
	defer func() { Note_Column_Invariants(column, "column_load.column") }()
	Counts_Invariants(counts, "column_load.counts")
	return Note_Column(counts[COUNT_COLUMN])
}

// Stands the column the notes of the run the print stands in open at. Zero names no run.
func column_store(counts Counts, column Note_Column) {
	Counts_Invariants(counts, "column_store.counts")
	Note_Column_Invariants(column, "column_store.column")
	counts[COUNT_COLUMN] = Count(column)
}

// INDENT_MAXIMUM is the most tabs one line opens at. A line that deep already runs past the widest
// line by far, thus the print refuses a form nested deeper rather than write it.
const INDENT_MAXIMUM = 128

// Indent is the tabs one line opens at.
type Indent int

// Indent_Invariants states every count of tabs one line opens at.
func Indent_Invariants(value Indent, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), FORM_SIZE_MINIMUM, INDENT_MAXIMUM).
		Ensure()
}

// Reads the tabs the line the print stands on opens at.
func indent_load(counts Counts) (indent Indent) {
	defer func() { Indent_Invariants(indent, "indent_load.indent") }()
	Counts_Invariants(counts, "indent_load.counts")
	return Indent(counts[COUNT_INDENT])
}

// Stands the tabs the line the print stands on opens at.
func indent_store(counts Counts, indent Indent) {
	Counts_Invariants(counts, "indent_store.counts")
	Indent_Invariants(indent, "indent_store.indent")
	counts[COUNT_INDENT] = Count(indent)
}

// Opens the line one tab deeper than the tabs it stood at. A form nested past the deepest indent
// marks the print deep, and the line holds the deepest indent.
func indent_deepen(subject Printer_Handle, indent Indent) {
	Printer_Handle_Invariants(subject, "indent_deepen.subject")
	Indent_Invariants(indent, "indent_deepen.indent")
	if indent == INDENT_MAXIMUM {
		flag_store(subject.Flags.Values, FLAG_DEEP, true)
		indent_store(subject.Counts.Values, indent)
		return
	}
	indent_store(subject.Counts.Values, indent+1)
	// One form of the tree opens each tab a line owes, thus the tabs the walk stands under
	// never pass the nodes it stands inside.
	aver.Always(int(indent_load(subject.Counts.Values)) <=
		int(depth_load(subject.Counts.Values)),
		"A step deeper stands at no more tabs than the forms the walk stands inside.")
}

// Closes the line one tab shallower than the tabs it stood at. A print marked deep held the
// deepest indent through every tab past it, thus it closes those tabs at no tab rather than fewer.
func indent_shallow(subject Printer_Handle, indent Indent) {
	Printer_Handle_Invariants(subject, "indent_shallow.subject")
	Indent_Invariants(indent, "indent_shallow.indent")
	if indent == 0 {
		aver.Always(bool(flag_load(subject.Flags.Values, FLAG_DEEP)),
			"Only a print marked deep closes a tab it never opened.")
		return
	}
	indent_store(subject.Counts.Values, indent-1)
	// A form that closes the tabs it opened closes them inside the form that holds it, thus
	// the tabs the walk stands under stay inside the forms it stands inside.
	aver.Always(int(indent_load(subject.Counts.Values)) <=
		int(depth_load(subject.Counts.Values)),
		"A step shallower stands inside the forms the walk stands inside.")
}

// Reads the tabs the form the print stands in opened its line at.
func column_indent_load(counts Counts) (indent Indent) {
	defer func() { Indent_Invariants(indent, "column_indent_load.indent") }()
	Counts_Invariants(counts, "column_indent_load.counts")
	return Indent(counts[COUNT_COLUMN_INDENT])
}

// Stands the tabs the form the print stands in opened its line at.
func column_indent_store(counts Counts, indent Indent) {
	Counts_Invariants(counts, "column_indent_store.counts")
	Indent_Invariants(indent, "column_indent_store.indent")
	counts[COUNT_COLUMN_INDENT] = Count(indent)
}

// Reads the widest key the run of pairs the print stands in holds. Zero names no run.
func key_width_load(counts Counts) (width Measured_Width) {
	defer func() { Measured_Width_Invariants(width, "key_width_load.width") }()
	Counts_Invariants(counts, "key_width_load.counts")
	return Measured_Width(counts[COUNT_WIDTH])
}

// Stands the widest key the run of pairs the print stands in holds. Zero names no run.
func key_width_store(counts Counts, width Measured_Width) {
	Counts_Invariants(counts, "key_width_store.counts")
	Measured_Width_Invariants(width, "key_width_store.width")
	counts[COUNT_WIDTH] = Count(width)
}

// NEST_MINIMUM is the nest a statement opens its expression at.
const NEST_MINIMUM = 1

// NEST_MAXIMUM is the deepest one expression stands inside the forms around it. A form nested that
// deep already reads as no table a reader follows, thus the print refuses one nested deeper
// rather than space it.
const NEST_MAXIMUM = 64

// Nest is how deep inside another form one expression stands. A statement opens at one, and the
// canonical form closes up the signs of an operation that stands deeper than that.
type Nest int

// Nest_Invariants states every nest one expression stands at.
func Nest_Invariants(value Nest, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), NEST_MINIMUM, NEST_MAXIMUM).
		Ensure()
}

// Reads how deep inside another form the expression the print stands on stands.
func nest_load(counts Counts) (nest Nest) {
	defer func() { Nest_Invariants(nest, "nest_load.nest") }()
	Counts_Invariants(counts, "nest_load.counts")
	return Nest(counts[COUNT_NEST])
}

// Stands how deep inside another form the expression the print stands on stands.
func nest_store(counts Counts, nest Nest) {
	Counts_Invariants(counts, "nest_store.counts")
	Nest_Invariants(nest, "nest_store.nest")
	counts[COUNT_NEST] = Count(nest)
}

// INNER_NEST_MINIMUM is the nest of a form that stands inside the one a statement opens.
const INNER_NEST_MINIMUM = NEST_MINIMUM + 1

// Inner_Nest is the nest of a form that stands inside another one.
type Inner_Nest Nest

// Inner_Nest_Invariants states every nest a form inside another one stands at.
func Inner_Nest_Invariants(value Inner_Nest, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), INNER_NEST_MINIMUM, NEST_MAXIMUM).
		Ensure()
}

// Names the nest one form deeper than the one handed over. A form nested past the deepest nest
// marks the print deep, and the form holds the deepest nest.
func nest_inner(subject Printer_Handle, nest Nest) (inner Inner_Nest) {
	defer func() { Inner_Nest_Invariants(inner, "nest_inner.inner") }()
	Printer_Handle_Invariants(subject, "nest_inner.subject")
	Nest_Invariants(nest, "nest_inner.nest")
	if nest == NEST_MAXIMUM {
		flag_store(subject.Flags.Values, FLAG_DEEP, true)
		return NEST_MAXIMUM
	}
	return Inner_Nest(nest + 1)
}

// Depth is the slot of the walk the node the print stands on sits in.
type Depth int

// Depth_Invariants states every slot of the walk.
func Depth_Invariants(value Depth, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), FORM_SIZE_MINIMUM, DEPTH_DEEPEST).
		Ensure()
}

// Reads the slot of the walk the node the print stands on sits in.
func depth_load(counts Counts) (depth Depth) {
	defer func() { Depth_Invariants(depth, "depth_load.depth") }()
	Counts_Invariants(counts, "depth_load.counts")
	return Depth(counts[COUNT_DEPTH])
}

// Stands the slot of the walk the node the print stands on sits in.
func depth_store(counts Counts, depth Depth) {
	Counts_Invariants(counts, "depth_store.counts")
	Depth_Invariants(depth, "depth_store.depth")
	counts[COUNT_DEPTH] = Count(depth)
}

// FLAG_LAST is the last slot of the flags.
const FLAG_LAST = FLAG_COUNT - 1

// Flag is one slot of the flags.
type Flag uint8

// Flag_Invariants states every slot of the flags.
func Flag_Invariants(value Flag, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), FLAG_FULL, FLAG_LAST).
		Ensure()
}

// Reads one flag of the print.
func flag_load(flags Flags, slot Flag) (value Boolean) {
	defer func() { Boolean_Invariants(value, "flag_load.value") }()
	Flags_Invariants(flags, "flag_load.flags")
	Flag_Invariants(slot, "flag_load.slot")
	return flags[slot]
}

// Stands one flag of the print.
func flag_store(flags Flags, slot Flag, value Boolean) {
	Flags_Invariants(flags, "flag_store.flags")
	Flag_Invariants(slot, "flag_store.slot")
	Boolean_Invariants(value, "flag_store.value")
	flags[slot] = value
}

// Stands every counter of the print back where a measure found it.
// States what one flat print wrote: a run of bytes holding no line feed. A flat print reads the
// columns one form spends on one line, thus a feed inside it would state a width no line holds.
func closes_flat(subject Printer_Handle, held Counts) {
	Printer_Handle_Invariants(subject, "closes_flat.subject")
	Counts_Invariants(held, "closes_flat.held")
	form := subject.Form.Values
	limit := int(written_load(subject.Counts.Values))
	for place := int(held[COUNT_WRITTEN]); place < limit; place++ {
		aver.Always(form[place] != '\n', "A flat print writes no line feed.")
	}
}

// States what one measure hands back: the form the print stood on before it ran. A measure writes
// what it reads past the bytes the print already wrote, thus every byte it wrote stands dropped.
func closes_measure(subject Printer_Handle, held Counts) {
	Printer_Handle_Invariants(subject, "closes_measure.subject")
	Counts_Invariants(held, "closes_measure.held")
	aver.Always(int(written_load(subject.Counts.Values)) == int(held[COUNT_WRITTEN]),
		"A measure hands back the form the print wrote.")
	// The lines a measure closed stand dropped with the bytes it wrote, thus the count of
	// lines reads what the print closed and never what a measure did.
	aver.Always(subject.Counts.Values[COUNT_LINE] == held[COUNT_LINE],
		"A measure hands back the lines the print closed.")
}

func restore_counts(subject Printer_Handle, held Counts) {
	Printer_Handle_Invariants(subject, "restore_counts.subject")
	Counts_Invariants(held, "restore_counts.held")
	counts := subject.Counts.Values
	written_store(counts, Form_Count(held[COUNT_WRITTEN]))
	depth_store(counts, Depth(held[COUNT_DEPTH]))
	indent_store(counts, Indent(held[COUNT_INDENT]))
	key_width_store(counts, Measured_Width(held[COUNT_WIDTH]))
	nest_store(counts, Nest(held[COUNT_NEST]))
	column_store(counts, Note_Column(held[COUNT_COLUMN]))
	column_indent_store(counts, Indent(held[COUNT_COLUMN_INDENT]))
	columns_store(counts, Line_Width(held[COUNT_COLUMNS]))
	// A measure writes the lines it reads past the form the print wrote, thus every count it
	// spent stands back, the count of the declarations it opened aside.
	for slot := range COUNT_SLOT_COUNT {
		if slot == COUNT_DECLARED {
			continue
		}
		counts[slot] = held[slot]
	}
	aver.Always(counts[COUNT_DECLARED] == held[COUNT_DECLARED],
		"A measure opens no run of declarations.")
	split_store(subject, Level(held[COUNT_SPLIT]))
}

// Stands every flag of the print back where a measure found it.
func restore_flags(subject Printer_Handle, held Flags) {
	Printer_Handle_Invariants(subject, "restore_flags.subject")
	Flags_Invariants(held, "restore_flags.held")
	for slot := range FLAG_COUNT {
		flag_store(subject.Flags.Values, Flag(slot), held[slot])
	}
}

// States what one print hands back once it wrote the whole file: the walk it opened at the file
// itself, the tabs and the nest it stands at, and the runs of keys and of notes it opened. A
// print that keeps any of them wrote a form some rule of it never closed.
func closes_file(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "closes_file.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "closes_file.tree")
	if !bool(writes_canonical(subject.Flags.Values)) {
		return
	}
	aver.Always(depth_load(subject.Counts.Values) == 0,
		"A print closes on the walk it opened at the file.")
	aver.Always(indent_load(subject.Counts.Values) == 0,
		"A print closes at the tabs the file opens at.")
	aver.Always(nest_load(subject.Counts.Values) == NEST_MINIMUM,
		"A print closes at the nest the file opens at.")
	aver.Always(key_width_load(subject.Counts.Values) == 0,
		"A print closes every run of keys it opened.")
	aver.Always(column_load(subject.Counts.Values) == 0,
		"A print closes every run of notes it opened.")
	// Every note the author wrote stands in the form, thus the storage that holds the notes
	// of one line carries none once the file closed.
	aver.Always(comment_count(subject) == COMMENT_COUNT_MINIMUM,
		"A print closes carrying no note it still owes.")
	aver.Always(!bool(flag_load(subject.Flags.Values, FLAG_BLANK)),
		"A print closes owing no empty line.")
	// The print states every declaration the file holds and states none of them twice, thus
	// the count of the ones it closed reads the count the file states. A tree that states
	// anything but a file writes no declaration at all.
	if ast.Syntax_Node_At(tree, subject.Nodes.Values[0]).Kind != ast.NODE_FILE {
		return
	}
	closes_declarations(subject, tree)
	// Every law that states what the print wrote reads one tally of the nodes, thus the laws
	// cost one walk of the arena rather than a walk for each of them.
	var held [COUNT_SLOT_COUNT]Count
	closes_tally(subject, tree, held[:])
	closes_declared_kinds(subject, held[:])
	closes_nodes(subject, held[:])
	closes_forms(subject, held[:])
	// One file names the package it binds once, thus the print writes that clause once.
	aver.Always(subject.Counts.Values[COUNT_CLAUSE] == 1,
		"A print writes the clause one file opens on, once.")
	closes_pairs(subject)
	closes_lines(subject)
}

// States what the table of the functions that state the invariants of a type holds once the print
// closed. The table fills from its first slot, thus the functions it holds stand in one run and
// a slot past that run holds none.
func closes_pairs(subject Printer_Handle) {
	Printer_Handle_Invariants(subject, "closes_pairs.subject")
	empty := Boolean(false)
	filled := Boolean(false)
	for slot := range len(subject.Pairs.Functions) {
		if subject.Pairs.Functions[slot] == ast.INDEX_ABSENT {
			empty = true
			continue
		}
		if bool(empty) {
			filled = true
		}
	}
	aver.Always(!bool(filled),
		"The table of the functions one file pairs holds its functions in one run.")
	// The table names the note that opens each function it holds, thus a slot holding no
	// function names no note either.
	stray := Boolean(false)
	for slot := range len(subject.Pairs.Heads) {
		if subject.Pairs.Functions[slot] != ast.INDEX_ABSENT {
			continue
		}
		if subject.Pairs.Heads[slot] != ast.INDEX_ABSENT {
			stray = true
		}
	}
	aver.Always(!bool(stray),
		"The table of the functions one file pairs names no note past its functions.")
}

// TREE_ROOT is the arena slot the parser writes the file node into, thus a caller hands over the
// tree alone and never the root it already knows.
const TREE_ROOT ast.Index = 1

// Writes one byte into the storage. A storage the byte no longer fits marks the print full, thus
// every later write is a test and never a write past the array.
func emit_byte(subject Printer_Handle, value Symbol) {
	Symbol_Invariants(value, "emit_byte.value")
	Printer_Handle_Invariants(subject, "emit_byte.subject")
	written := int(written_load(subject.Counts.Values))
	if written >= FORM_SIZE_MAXIMUM {
		flag_store(subject.Flags.Values, FLAG_FULL, true)
		return
	}
	subject.Form.Values[written] = byte(value)
	written_store(subject.Counts.Values, Form_Count(written+1))
	flag_store(subject.Flags.Values, FLAG_LINE, false)
	// A line feed opens a line of no columns, a tab stands in the columns a reader sees, and
	// a byte that continues a character stands in none of its own.
	spent := int(columns_load(subject.Counts.Values)) + 1
	if value == '\t' {
		spent = int(columns_load(subject.Counts.Values)) + TAB_WIDTH
	}
	if value&CHARACTER_TAIL_MASK == CHARACTER_TAIL_MARK {
		spent = int(columns_load(subject.Counts.Values))
	}
	if value == '\n' {
		spent = 0
		subject.Counts.Values[COUNT_LINE] = subject.Counts.Values[COUNT_LINE] + 1
	}
	columns_store(subject.Counts.Values, Line_Width(min(spent, WIDTH_MAXIMUM)))
}

// Reports whether the print writes the canonical form of a tree. A print that filled its storage
// or met a tree deeper than the walk holds writes a form no caller reads, and a print of a tree
// the parse refused writes what that tree holds rather than the form the rules state.
func writes_canonical(flags Flags) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "writes_canonical.yes") }()
	Flags_Invariants(flags, "writes_canonical.flags")
	if bool(flag_load(flags, FLAG_FULL)) {
		return false
	}
	if bool(flag_load(flags, FLAG_DEEP)) {
		return false
	}
	return !flag_load(flags, FLAG_REFUSED)
}

// SYMBOL_SPACE is the byte that holds two bytes of one line apart.
const SYMBOL_SPACE = Symbol(32)

// SYMBOL_DELETE is the byte standing past every sign a reader reads, which no step of the
// program writes.
const SYMBOL_DELETE = 127

// SYMBOL_COMMA is the byte that closes one part of a form.
const SYMBOL_COMMA = Symbol(44)

// Writes the comma that closes one part of a form. A comma closes what stands ahead of it, thus
// the line it stands on already holds a byte.
func emit_comma(subject Printer_Handle) {
	Printer_Handle_Invariants(subject, "emit_comma.subject")
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(!bool(flag_load(subject.Flags.Values, FLAG_LINE)),
			"A comma closes a byte of its line.")
		// A comma closes what stands ahead of it, thus no space holds the two apart.
		aver.Always(!bool(stands_after_space(subject)),
			"A comma stands against the byte ahead of it.")
	}
	stood := written_load(subject.Counts.Values)
	emit_byte(subject, SYMBOL_COMMA)
	// A comma spells one byte, thus the form holds one byte more than it held.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(written_load(subject.Counts.Values) == stood+1,
			"A comma that parts two parts spells one byte of its own.")
	}
}

// Writes one space.
func emit_space(subject Printer_Handle) {
	Printer_Handle_Invariants(subject, "emit_space.subject")
	// A space holds two bytes of one line apart, thus the line already holds a byte where
	// the print writes one, and no second space stands behind it.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(!bool(flag_load(subject.Flags.Values, FLAG_LINE)),
			"A space stands behind a byte of its line.")
		aver.Always(!bool(stands_after_space(subject)),
			"A space stands behind a byte and never behind a space.")
		aver.Always(take_line_width(subject.Counts.Values) > 0,
			"A space stands on a line that already holds a column.")
	}
	emit_byte(subject, SYMBOL_SPACE)
	// A print that still holds storage lands every byte it writes, thus the line the space
	// opened closes on that space.
	if !bool(flag_load(subject.Flags.Values, FLAG_FULL)) {
		aver.Always(subject.Form.Values[int(written_load(subject.Counts.Values))-1] == ' ',
			"A space lands in the form as the byte its line closes on.")
	}
}

// Writes one line feed and marks the line that follows as still unopened. The canonical form
// states no byte a reader cannot see at the end of a line, thus the line closes on the last byte
// it holds and never on a space or a tab.
func emit_line(subject Printer_Handle) {
	Printer_Handle_Invariants(subject, "emit_line.subject")
	// A print that filled its storage writes a form no caller reads, and a tree the parse
	// refused holds forms the canonical form states nothing for, thus only a whole print of a
	// tree the parse holds answers for the byte its line closes on.
	if bool(writes_canonical(subject.Flags.Values)) {
		drop_trailing_space(subject)
		aver.Always(!bool(stands_after_space(subject)),
			"A line closes on no space and no tab.")
		// The form opens on the first byte a declaration writes, thus no line of it closes
		// on nothing at all.
		aver.Always(written_load(subject.Counts.Values) > 0,
			"The form opens on a byte and not on a line feed.")
		// One empty line stands between two forms and no run of them does, thus the
		// feed this line closes on opens no second empty line.
		written := int(written_load(subject.Counts.Values))
		if written > 1 {
			form := subject.Form.Values
			if form[written-1] == '\n' {
				aver.Always(form[written-2] != '\n',
					"No two empty lines stand together.")
			}
		}
	}
	emit_byte(subject, '\n')
	// A print that still holds storage lands every byte it writes, thus the feed closes the
	// line the caller closed.
	if !bool(flag_load(subject.Flags.Values, FLAG_FULL)) {
		aver.Always(subject.Form.Values[int(written_load(subject.Counts.Values))-1] == '\n',
			"A feed lands in the form as the byte its line closes on.")
	}
	flag_store(subject.Flags.Values, FLAG_LINE, true)
}

// Takes back every space and tab the line closing here holds at its end. A tree the parse refused
// states parts of no width, and a run that pads for one of them writes what no reader sees, thus
// the line closes on the last byte a reader reads.
func drop_trailing_space(subject Printer_Handle) {
	Printer_Handle_Invariants(subject, "drop_trailing_space.subject")
	for bool(stands_after_space(subject)) {
		written := written_load(subject.Counts.Values)
		// A space stands in one column and a tab in the columns a reader sees, thus the
		// line hands back what the byte it drops spent.
		spent := 1
		if subject.Form.Values[int(written)-1] == '\t' {
			spent = TAB_WIDTH
		}
		held := int(columns_load(subject.Counts.Values)) - spent
		columns_store(subject.Counts.Values, Line_Width(max(held, 0)))
		written_store(subject.Counts.Values, written-1)
	}
}

// Writes the tabs the open line owes, which is one for each depth the walk stands under. A line
// that already holds a byte owes none, thus a caller writes the indent before anything else.
func emit_indent(subject Printer_Handle) {
	Printer_Handle_Invariants(subject, "emit_indent.subject")
	if !bool(flag_load(subject.Flags.Values, FLAG_LINE)) {
		return
	}
	// The bytes one line opens at stand ahead of every byte that line holds, thus the print
	// writes them where the line it stands on holds none.
	if bool(writes_canonical(subject.Flags.Values)) {
		opened := int(written_load(subject.Counts.Values))
		if opened > 0 {
			aver.Always(subject.Form.Values[opened-1] == '\n',
				"The bytes one line opens at stand ahead of every byte it holds.")
		}
	}
	written := int(written_load(subject.Counts.Values))
	indent_count := int(indent_load(subject.Counts.Values))
	// One line opens at as many bytes as the depths it stands under state, each of them as
	// wide as one column.
	spent_count := indent_count * INDENT_WIDTH
	available_count := FORM_SIZE_MAXIMUM - written
	if available_count < spent_count {
		spent_count = available_count
		indent_count = spent_count / INDENT_WIDTH
		flag_store(subject.Flags.Values, FLAG_FULL, true)
	}
	for index := range spent_count {
		subject.Form.Values[written+index] = byte(INDENT_SYMBOL)
	}
	if spent_count > 0 {
		written_store(subject.Counts.Values, Form_Count(written+spent_count))
		// The bytes the line opens on stand in the columns a reader sees.
		spent := int(columns_load(subject.Counts.Values)) + spent_count
		columns_store(subject.Counts.Values, Line_Width(min(spent, WIDTH_MAXIMUM)))
		flag_store(subject.Flags.Values, FLAG_LINE, false)
	}
	// A line opens at one byte for each depth the walk stands under, thus a print that still
	// holds storage writes every one of them.
	if !bool(flag_load(subject.Flags.Values, FLAG_FULL)) {
		aver.Always(indent_count == int(indent_load(subject.Counts.Values)),
			"A line opens at one byte for each depth the walk stands under.")
		// Each such byte stands in one column, thus the line they open holds the columns
		// the walk stands under and no byte more.
		aver.Always(int(take_line_width(subject.Counts.Values)) ==
			indent_count*INDENT_WIDTH,
			"A line that opens on those bytes holds the columns they span.")
	}
}

// Writes the source bytes the token of the node the walk stands on spans. A number states its base
// and its exponent in the case the canonical form states them, thus the print writes the literal
// the source means rather than the bytes the author typed, and a note wraps where its words run
// past the widest line.
func emit_token(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "emit_token.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "emit_token.tree")
	cursor := ast.Syntax_Node_At(tree, subject.Nodes.Values[depth_load(subject.Counts.Values)])
	// Every token the print writes passes here, thus the count of the literals reads the
	// class of the node it stands on once.
	if bool(states_literal(cursor.Kind)) {
		subject.Counts.Values[COUNT_TYPE_FORM] += 1
	}
	if cursor.Kind == ast.NODE_IDENTIFIER {
		subject.Counts.Values[COUNT_NAME] += 1
	}
	one := ast.Syntax_Token_At(tree, cursor.Token)
	switch one.Kind {
	case token.KIND_INTEGER, token.KIND_FLOAT, token.KIND_IMAGINARY:
		emit_number(subject, tree)
		return
	case token.KIND_COMMENT:
		emit_note(subject, tree)
		return
	case token.KIND_STRING:
		emit_string(subject, tree)
		return
	}
	text := token.Text(token.Source(subject.Source.Values), one)
	stood := int(written_load(subject.Counts.Values))
	for index := range len(text) {
		emit_byte(subject, Symbol(text[index]))
	}
	// A token stands in the form as the author spelled it, thus every byte of it lands there
	// wherever the storage the caller supplied still holds room for them.
	if !bool(flag_load(subject.Flags.Values, FLAG_FULL)) {
		aver.Always(int(written_load(subject.Counts.Values))-stood == len(text),
			"A token the print writes lands whole in the form.")
	}
}

// Holds one comment or one empty line until the line it stands behind closes. The parser hangs
// each one behind the node whose span it follows, thus a form that still owes a bracket writes
// that bracket first and the trivia behind it.
func print_trivia(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "print_trivia.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "print_trivia.tree")
	// A walk over trivia meets a note or an empty line alone, thus a part of the form
	// standing here would stand in no line the print writes.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(bool(trivia_kind(subject, tree)),
			"A node the print holds as trivia states trivia.")
		Trivia_Part_Invariants(trivia_part(subject, tree), "print_trivia.parted")
	}
	kind := node_kind(subject, tree)
	if kind == ast.NODE_BLANK {
		// A body opens at its first part and closes at its last one, thus an empty line
		// standing against the bracket on either side states nothing at all.
		if bool(drops_blank(subject, tree)) {
			return
		}
		flag_store(subject.Flags.Values, FLAG_BLANK, true)
		return
	}
	if kind != ast.NODE_COMMENT {
		return
	}
	// A comment past the storage would drop what the author wrote, thus the print stops as
	// it does at the end of the storage the caller supplied.
	count := int(comment_count(subject))
	if count >= COMMENT_COUNT_MAXIMUM {
		flag_store(subject.Flags.Values, FLAG_FULL, true)
		return
	}
	comment := subject.Nodes.Values[depth_load(subject.Counts.Values)]
	// The count names the first empty slot of the storage, thus a note lands where no note
	// stands and the run of notes one line carries holds every one of them.
	aver.Always(subject.Comments.Values[count] == ast.INDEX_ABSENT,
		"A note the line takes lands on an empty slot of the storage.")
	subject.Comments.Values[count] = comment
	// The note stands in the storage where the count named an empty slot, thus the line
	// carries one note more than it carried.
	aver.Always(int(comment_count(subject)) == count+1,
		"A note the line takes leaves the storage carrying it.")
	// An empty line ahead of a note belongs to that note, and the writing of the note reads it
	// from the source, thus the line the print still owes is the one behind every note.
	flag_store(subject.Flags.Values, FLAG_BLANK, false)
}

// TRIVIA_PART_NOTE names a note the author wrote, which the print writes where it stands.
const TRIVIA_PART_NOTE = 0

// TRIVIA_PART_LINE names an empty line the author left, which the print owes the line behind it.
const TRIVIA_PART_LINE = 1

// Trivia_Part names which of the two the print holds: a note or an empty line.
type Trivia_Part uint8

// Trivia_Part_Invariants states both forms of trivia one line carries.
func Trivia_Part_Invariants(value Trivia_Part, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value), TRIVIA_PART_NOTE, TRIVIA_PART_LINE).
		Ensure()
}

// Reads which of the two forms of trivia the walk stands on.
func trivia_part(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (parted Trivia_Part) {
	defer func() { Trivia_Part_Invariants(parted, "trivia_part.parted") }()
	Printer_Handle_Invariants(subject, "trivia_part.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "trivia_part.tree")
	if node_kind(subject, tree) == ast.NODE_BLANK {
		return TRIVIA_PART_LINE
	}
	return TRIVIA_PART_NOTE
}

// Closes the line the print stands on: the comments it carries stand at its end, and the empty
// line it owes stands behind it.
func close_line(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "close_line.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "close_line.tree")
	// The columns the print counted as it wrote state the line it closes here.
	reads_columns(subject)
	flush_comments(subject, tree, CLOSER_POSITION_NONE)
	finish_line(subject)
	// Every note the line owed stands written, thus the close carries none to the line
	// behind it.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(comment_count(subject) == COMMENT_COUNT_MINIMUM,
			"A line the print closed carries no note it still owes.")
	}
}

// Writes the line feed one closed line takes and the empty line it owes behind it.
func finish_line(subject Printer_Handle) {
	Printer_Handle_Invariants(subject, "finish_line.subject")
	// Two empty lines together state no more than one stands, thus a close of a line that
	// holds nothing behind an empty line writes no feed at all.
	if !bool(stands_after_line(subject)) {
		emit_line(subject)
	}
	if bool(flag_load(subject.Flags.Values, FLAG_BLANK)) {
		if !bool(stands_after_line(subject)) {
			emit_line(subject)
		}
		flag_store(subject.Flags.Values, FLAG_BLANK, false)
		key_width_store(subject.Counts.Values, 0)
	}
	// The line the print stands on behind a close is the one the feed opened, thus it holds
	// no column of its own.
	aver.Always(bool(flag_load(subject.Flags.Values, FLAG_LINE)),
		"A close of one line leaves the print at the opening of the next.")
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(take_line_width(subject.Counts.Values) == 0,
			"A line the print opens holds no column at all.")
	}
	aver.Always(!bool(flag_load(subject.Flags.Values, FLAG_BLANK)),
		"A close of one line owes no empty line behind it.")
}

// Writes one comment: it takes the tabs of a line it opens, and one space where it follows what
// already stands on the line.
func emit_comment(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "emit_comment.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "emit_comment.tree")
	// A read of the width one statement writes reads the statement alone.
	if bool(flag_load(subject.Flags.Values, FLAG_NOTELESS)) {
		return
	}
	// The indent the print writes opens the line, thus the report the writing reads must be
	// the one that stood before it wrote anything.
	opening := flag_load(subject.Flags.Values, FLAG_LINE)
	if bool(opening) {
		emit_indent(subject)
	}
	// Every comment of one run opens at one column, thus a reader follows the notes down the
	// run rather than hunting for each one at the end of its line.
	if !bool(opening) {
		column := column_load(subject.Counts.Values)
		reads_columns(subject)
		// A note that stands in a run pads to the column that run opens at, thus the
		// print reads that the cursor states the line the form holds.
		if column != 0 {
			// The run opens its column behind the widest line it holds, thus the
			// line this note closes stands inside that column. A line that already
			// carries a note stands past that column, where the note it took ahead
			// of this one names the line.
			stood := int(take_line_width(subject.Counts.Values))
			held := Boolean(stood < WIDTH_MAXIMUM)
			if bool(held) {
				held = Boolean(!bool(carries_note(subject)))
			}
			if bool(held) {
				held = writes_canonical(subject.Flags.Values)
			}
			if bool(held) {
				aver.Always(stood <= int(column),
					"A line of one run stands inside its column.")
			}
		}
		width := Part_Width(min(int(take_line_width(subject.Counts.Values)), WIDTH_MAXIMUM))
		emit_column(subject, width, Column_Width(column))
		// A note stands apart from what it follows, thus one space at least stands
		// between the byte the line held and the note the print writes here.
		aver.Always(bool(stands_after_space(subject)),
			"A note that follows a byte of its line stands behind a space.")
		// Every note of one run opens at one column, and a line past that column opens
		// its note one space behind the byte it closed on.
		stood := int(take_line_width(subject.Counts.Values))
		if stood < WIDTH_MAXIMUM {
			if bool(writes_canonical(subject.Flags.Values)) {
				aver.Always(stood == max(int(column), int(width))+1,
					"A note of one run opens at the column the run states.")
			}
		}
	}
	// A note the print holds until the line closes reads the same as one it writes where it
	// stands, thus both run through one writing and a note of a line of its own wraps here
	// too.
	emit_note(subject, tree)
}

// Moves the walk to the first child that states a part of the form, and writes every comment and
// empty line it steps over. The parser hangs trivia behind the node its span follows, thus a walk
// over the parts of a form meets trivia anywhere and prints it where it stands.
func descend_part(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (ok Boolean) {
	defer func() { Boolean_Invariants(ok, "descend_part.ok") }()
	Printer_Handle_Invariants(subject, "descend_part.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "descend_part.tree")
	held_depth := depth_load(subject.Counts.Values)
	if !bool(descend(subject, tree)) {
		aver.Always(depth_load(subject.Counts.Values) == held_depth,
			"A step down that finds no part of one form leaves the depth it stood at.")
		return false
	}
	if bool(skip_trivia(subject, tree)) {
		// The step down stands on a child of the form, thus it stands one slot below it
		// however much trivia it wrote on the way.
		aver.Always(depth_load(subject.Counts.Values) == held_depth+1,
			"A step down that finds a part of one form stands one slot below it.")
		return true
	}
	// A form of trivia alone holds no part, and a caller that reads no part takes no step
	// back, thus the step down closes here.
	ascend(subject)
	aver.Always(depth_load(subject.Counts.Values) == held_depth,
		"A step back over the trivia of one form leaves the depth it stood at.")
	return false
}

// Moves the walk to the sibling that states the next part of the form, and writes every comment
// and empty line it steps over.
func advance_part(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (ok Boolean) {
	defer func() { Boolean_Invariants(ok, "advance_part.ok") }()
	Printer_Handle_Invariants(subject, "advance_part.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "advance_part.tree")
	held_depth := depth_load(subject.Counts.Values)
	held_node := subject.Nodes.Values[held_depth]
	if !bool(advance(subject, tree)) {
		aver.Always(depth_load(subject.Counts.Values) == held_depth,
			"A step that finds no next part of one form leaves the depth it stood at.")
		return false
	}
	// The step stands on a sibling of the node it stood on, thus it holds the slot it began
	// at and stands on another node of the tree.
	aver.Always(depth_load(subject.Counts.Values) == held_depth,
		"A step to the next part of one form holds the slot it stood on.")
	aver.Always(subject.Nodes.Values[held_depth] != held_node,
		"A step to the next part of one form leaves the part it stood on.")
	return skip_trivia(subject, tree)
}

// Steps past every part and every piece of trivia the walk still stands ahead of, thus a form
// that stopped at its final part still holds the comments and empty lines behind it.
func finish_parts(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "finish_parts.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "finish_parts.tree")
	for bool(advance_part(subject, tree)) {
		continue
	}
	// Every part of the form ran through the walk, thus the walk stands on the last one and
	// no part of the chain stands behind it unwritten.
	index := subject.Nodes.Values[depth_load(subject.Counts.Values)]
	aver.Always(ast.Index(ast.Syntax_Node_At(tree, index).Next) == 0,
		"A walk that finished the parts stands on the last of them.")
}

// Writes the trivia the walk stands on and steps past it, and reports whether a part stands where
// the walk stopped.
func skip_trivia(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (ok Boolean) {
	defer func() { Boolean_Invariants(ok, "skip_trivia.ok") }()
	Printer_Handle_Invariants(subject, "skip_trivia.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "skip_trivia.tree")
	for bool(trivia_kind(subject, tree)) {
		print_trivia(subject, tree)
		if !bool(advance(subject, tree)) {
			return false
		}
	}
	return true
}

// Writes every comment and empty line the walk still stands ahead of. A line already closed
// stands behind them, thus each one takes a line of its own rather than the end of that line.
func drain_trivia(subject Printer_Handle, tree ast.Syntax_Tree_Handle, open Boolean) {
	Printer_Handle_Invariants(subject, "drain_trivia.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "drain_trivia.tree")
	Boolean_Invariants(open, "drain_trivia.open")
	stand := open
	for bool(stand) {
		print_behind(subject, tree)
		stand = advance(subject, tree)
	}
}

// Writes one comment or one empty line that stands behind a line the print already closed.
func print_behind(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "print_behind.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "print_behind.tree")
	// A walk that closed its line meets trivia alone behind it, thus a part of the form
	// standing here would write no line at all.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(bool(trivia_kind(subject, tree)),
			"A node standing behind a closed line states trivia.")
		Trivia_Part_Invariants(trivia_part(subject, tree), "print_behind.parted")
	}
	kind := node_kind(subject, tree)
	if kind == ast.NODE_BLANK {
		if bool(stands_against(subject, tree)) {
			return
		}
		if bool(stands_before_check(subject, tree)) {
			return
		}
		if bool(stands_behind_sign(subject, tree)) {
			return
		}
		emit_line(subject)
		return
	}
	if kind != ast.NODE_COMMENT {
		return
	}
	emit_indent(subject)
	emit_token(subject, tree)
	emit_line(subject)
}

// Reads the class of the node the walk stands on.
func node_kind(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (kind ast.Node_Kind) {
	defer func() { ast.Node_Kind_Invariants(kind, "node_kind.kind") }()
	Printer_Handle_Invariants(subject, "node_kind.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "node_kind.tree")
	current := subject.Nodes.Values[depth_load(subject.Counts.Values)]
	return ast.Syntax_Node_At(tree, current).Kind
}

// Moves the walk to the first child of the node it stands on.
func descend(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (ok Boolean) {
	defer func() { Boolean_Invariants(ok, "descend.ok") }()
	Printer_Handle_Invariants(subject, "descend.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "descend.tree")
	index := subject.Nodes.Values[depth_load(subject.Counts.Values)]
	child := ast.Syntax_Node_At(tree, index).First_Child
	if ast.Index(child) == 0 {
		return false
	}
	if int(depth_load(subject.Counts.Values))+1 >= DEPTH_MAXIMUM {
		flag_store(subject.Flags.Values, FLAG_DEEP, true)
		return false
	}
	// The chain the walk steps into hangs under the node it stood on.
	aver.Always(ast.Index(ast.Syntax_Node_At(tree, ast.Index(child)).Parent) == index,
		"A step down the walk stands on a child of the node it left.")
	depth_store(subject.Counts.Values, depth_load(subject.Counts.Values)+1)
	subject.Nodes.Values[depth_load(subject.Counts.Values)] = ast.Index(child)
	return true
}

// Moves the walk to the sibling after the node it stands on.
func advance(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (ok Boolean) {
	defer func() { Boolean_Invariants(ok, "advance.ok") }()
	Printer_Handle_Invariants(subject, "advance.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "advance.tree")
	current := subject.Nodes.Values[depth_load(subject.Counts.Values)]
	next := ast.Syntax_Node_At(tree, current).Next
	if ast.Index(next) == 0 {
		return false
	}
	// Two siblings hang under one node, thus a step along the chain leaves the walk where
	// the step down the walk took opened it.
	aver.Always(ast.Syntax_Node_At(tree, ast.Index(next)).Parent ==
		ast.Syntax_Node_At(tree, current).Parent,
		"A step along the walk stands on a node of the chain it stood in.")
	// The parse hangs the nodes of one chain in the order the source spells them, thus a
	// step along that chain stands on a node the parse made behind the one it left.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(ast.Index(next) > current,
			"A step along the walk stands on a node the parse made behind it.")
	}
	subject.Nodes.Values[depth_load(subject.Counts.Values)] = ast.Index(next)
	return true
}

// Leaves the child chain the walk stands in.
func ascend(subject Printer_Handle) {
	Printer_Handle_Invariants(subject, "ascend.subject")
	// The walk stands on the file at no depth at all, and every step down it takes stands
	// behind a step up, thus a step up from the file writes a walk no step down opened.
	aver.Always(depth_load(subject.Counts.Values) > 0,
		"A step up the walk stands behind a step down.")
	depth_store(subject.Counts.Values, depth_load(subject.Counts.Values)-1)
}

// Closes the file on one line feed. Each declaration carries the empty line that stood behind it,
// and the declaration the author wrote last carries none, thus a file whose last declaration now
// stands under a type would close on a line that states nothing.
func close_file(subject Printer_Handle) {
	Printer_Handle_Invariants(subject, "close_file.subject")
	form := subject.Form.Values
	// Every line of the form closes on a line feed, thus a file that left one open closes it
	// before the print takes back the empty lines behind it.
	if !bool(flag_load(subject.Flags.Values, FLAG_LINE)) {
		emit_line(subject)
	}
	// The form closes on one line feed, thus a reader of it reads whole lines.
	if written_load(subject.Counts.Values) > 0 {
		if bool(writes_canonical(subject.Flags.Values)) {
			aver.Always(form[int(written_load(subject.Counts.Values))-1] == '\n',
				"The form closes on the line feed its last line takes.")
		}
	}
	for written_load(subject.Counts.Values) > 1 {
		written := int(written_load(subject.Counts.Values))
		if form[written-1] != '\n' {
			return
		}
		if form[written-2] != '\n' {
			return
		}
		written_store(subject.Counts.Values, Form_Count(written-1))
		// The drop takes back one feed the print wrote, thus the count of the lines it
		// closed reads the feeds the form still holds.
		subject.Counts.Values[COUNT_LINE] = subject.Counts.Values[COUNT_LINE] - 1
		// The drop closes the form on the feed behind an empty line, thus the line the
		// print stands on holds no column at all.
		columns_store(subject.Counts.Values, 0)
	}
}

// Writes the run of declarations that states the invariants of the type the walk stands on, which
// is the function of that name and the notes above it. It reports whether the run it wrote spans
// more than one line, which is what the declaration behind it stands apart from.
func print_invariant_run(
	subject Printer_Handle, tree ast.Syntax_Tree_Handle, spread Boolean,
) (held Boolean, moved Boolean) {
	defer func() {
		Boolean_Invariants(held, "print_invariant_run.held")
		Boolean_Invariants(moved, "print_invariant_run.moved")
	}()
	Printer_Handle_Invariants(subject, "print_invariant_run.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "print_invariant_run.tree")
	Boolean_Invariants(spread, "print_invariant_run.spread")
	stand := subject.Nodes.Values[depth_load(subject.Counts.Values)]
	stood := written_load(subject.Counts.Values)
	node := ast.INDEX_ABSENT
	if bool(take_invariant_run(subject, tree)) {
		node = subject.Nodes.Values[depth_load(subject.Counts.Values)]
	}
	commented := Boolean(false)
	for node != ast.INDEX_ABSENT {
		subject.Nodes.Values[depth_load(subject.Counts.Values)] = node
		kind := node_kind(subject, tree)
		// The run stands under the type the walk wrote last.
		open_declaration(subject, tree, commented, Boolean(kind == ast.NODE_TYPE))
		if !bool(commented) {
			declared_store(subject.Counts.Values,
				Declaration_Place(written_load(subject.Counts.Values)))
		}
		print_declaration(subject, tree)
		// The run wrote one declaration of the file here, thus the count of them reads
		// this one the way it reads the ones the file walk wrote.
		if !bool(trivia_kind(subject, tree)) {
			subject.Counts.Values[COUNT_DECLARED_COUNT] =
				subject.Counts.Values[COUNT_DECLARED_COUNT] + 1
			counts_declaration(subject, tree)
		}
		commented = Boolean(kind == ast.NODE_COMMENT)
		if !bool(commented) {
			spread = open_spread(subject, spread)
			moved = true
			break
		}
		node = ast.Index(ast.Syntax_Node_At(tree, node).Next)
	}
	subject.Nodes.Values[depth_load(subject.Counts.Values)] = stand
	// A run that moved a declaration under the type it names wrote that declaration, thus
	// the form holds the bytes of every declaration the run took.
	if bool(moved) {
		aver.Always(written_load(subject.Counts.Values) > stood,
			"A run that moves a declaration writes the bytes of it.")
	}
	return spread, moved
}

// Reports whether the declaration the walk stands on stands under a type of the file, reading the
// run of notes that opens such a declaration one time. Every note of one run answers the same way,
// thus the walk holds that answer through the run rather than reading the run again at each note.
func notes_move(
	subject Printer_Handle, tree ast.Syntax_Tree_Handle, run Notes_Run, moved Moved_Notes,
) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "notes_move.yes") }()
	Printer_Handle_Invariants(subject, "notes_move.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "notes_move.tree")
	Notes_Run_Invariants(run, "notes_move.run")
	Moved_Notes_Invariants(moved, "notes_move.moved")
	if bool(run) {
		if node_kind(subject, tree) == ast.NODE_COMMENT {
			return Boolean(moved)
		}
	}
	if bool(moves_under_type(subject, tree)) {
		return true
	}
	return moves_first(subject, tree)
}

// Reports whether the declaration the walk stands on stands under a type of the file rather than
// where the author wrote it: a function that states the invariants of such a type, or a note that
// opens one. The type carries both, thus the walk writes neither where it finds them.
func moves_under_type(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "moves_under_type.yes") }()
	Printer_Handle_Invariants(subject, "moves_under_type.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "moves_under_type.tree")
	stood := written_load(subject.Counts.Values)
	defer func() {
		aver.Always(written_load(subject.Counts.Values) == stood,
			"A read of the type one run moves under writes no byte of its own.")
	}()
	depth := depth_load(subject.Counts.Values)
	stand := subject.Nodes.Values[depth]
	one := stand
	for range ast.NODE_COUNT_MAXIMUM {
		if ast.Syntax_Node_At(tree, one).Kind != ast.NODE_COMMENT {
			break
		}
		one = ast.Index(ast.Syntax_Node_At(tree, one).Next)
		if one == ast.INDEX_ABSENT {
			return false
		}
	}
	subject.Nodes.Values[depth] = one
	yes = states_own_type(subject, tree)
	subject.Nodes.Values[depth] = stand
	return yes
}

// Moves the walk from the type it stands on to the run of declarations that states the invariants
// of that type: the function of that name and the notes standing above it. A type that states
// none names no run at all, thus the walk stands where it stood and the read reports none.
func take_invariant_run(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (found Boolean) {
	defer func() { Boolean_Invariants(found, "take_invariant_run.found") }()
	Printer_Handle_Invariants(subject, "take_invariant_run.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "take_invariant_run.tree")
	depth := depth_load(subject.Counts.Values)
	named := subject.Nodes.Values[depth]
	if ast.Syntax_Node_At(tree, named).Kind != ast.NODE_TYPE {
		return false
	}
	// One function of invariants stands under one type, thus the first type of its name
	// takes it and every other type of that name takes none. The read reads the types the
	// file declares and writes none of them.
	written := written_load(subject.Counts.Values)
	named_first := first_named_type(subject, tree)
	aver.Always(written_load(subject.Counts.Values) == written,
		"A read for the first type of one name writes no byte.")
	if !bool(named_first) {
		return false
	}
	for slot := range len(subject.Pairs.Functions) {
		function := subject.Pairs.Functions[slot]
		if function == ast.INDEX_ABSENT {
			return false
		}
		subject.Nodes.Values[depth+1] = function
		depth_store(subject.Counts.Values, depth+1)
		paired := names_pair(subject, tree)
		depth_store(subject.Counts.Values, depth)
		if !bool(paired) {
			continue
		}
		// The notes that open the function stand under the type with it.
		head := subject.Pairs.Heads[slot]
		subject.Nodes.Values[depth] = function
		if head != ast.INDEX_ABSENT {
			subject.Nodes.Values[depth] = head
			// A run of notes opens the declaration it stands above, thus the run
			// the walk takes opens above the function the table paired.
			aver.Always(head < function,
				"The notes of one pair stand above the function they open.")
		}
		return true
	}
	return false
}

// Writes one node that stands ahead of the declarations: the doc comment of the file, or the
// package clause that closes the opening. It reports whether the clause now stands written.
func print_opening(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (named Boolean) {
	defer func() { Boolean_Invariants(named, "print_opening.named") }()
	Printer_Handle_Invariants(subject, "print_opening.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "print_opening.tree")
	kind := node_kind(subject, tree)
	// A file opens on the notes and the empty lines its author wrote ahead of the clause
	// that names the package it binds, thus every child it opens on states one of the three.
	if bool(writes_canonical(subject.Flags.Values)) {
		Opening_Part_Invariants(opening_part(subject, tree), "print_opening.parted")
	}
	// A file opens at its first line however many empty lines stand ahead of it, thus a blank
	// the print has written nothing before is no line of the form.
	if kind == ast.NODE_BLANK {
		if written_load(subject.Counts.Values) == 0 {
			return false
		}
		print_behind(subject, tree)
		return false
	}
	if kind == ast.NODE_COMMENT {
		print_behind(subject, tree)
		return false
	}
	emit_word(subject, WORD_PACKAGE)
	emit_space(subject)
	stood := written_load(subject.Counts.Values)
	emit_token(subject, tree)
	// A file opens on the word it states and the name it binds, thus the name spells bytes
	// of its own behind that word.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(written_load(subject.Counts.Values) > stood,
			"A file names the package it binds behind the word it opens on.")
	}
	// A note the author left on the line the clause closes keeps that line, thus the print
	// takes it before it closes that line.
	take_statement_note(subject, tree)
	close_line(subject, tree)
	return true
}

// OPENING_PART_NAME names the clause that binds the package the file states.
const OPENING_PART_NAME = 0

// OPENING_PART_NOTE names a note the author wrote ahead of that clause.
const OPENING_PART_NOTE = 1

// OPENING_PART_LINE names an empty line the author left ahead of that clause.
const OPENING_PART_LINE = 2

// Opening_Part names which of the three a file opens on.
type Opening_Part uint8

// Opening_Part_Invariants states every part one file opens on.
func Opening_Part_Invariants(value Opening_Part, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(uint8(value), OPENING_PART_NAME, OPENING_PART_NOTE, OPENING_PART_LINE).
		Ensure()
}

// WORD_PACKAGE opens a file.
const WORD_PACKAGE Word = 0

// WORD_IMPORT opens an import.
const WORD_IMPORT Word = 1

// WORD_CONSTANT opens a constant declaration.
const WORD_CONSTANT Word = 2

// WORD_VARIABLE opens a variable declaration.
const WORD_VARIABLE Word = 3

// WORD_TYPE opens a type declaration.
const WORD_TYPE Word = 4

// WORD_FUNCTION opens a function declaration.
const WORD_FUNCTION Word = 5

// WORD_STRUCTURE opens a struct type.
const WORD_STRUCTURE Word = 6

// WORD_INTERFACE opens a constraint.
const WORD_INTERFACE Word = 7

// WORD_MAP opens a map type.
const WORD_MAP Word = 8

// WORD_CHANNEL opens a channel type.
const WORD_CHANNEL Word = 9

// WORD_RANGE opens a range clause.
const WORD_RANGE Word = 10

// WORD_ELSE opens the branch an if statement takes when its condition fails.
const WORD_ELSE Word = 11

// WORD_DEFAULT opens the case a switch takes when every other case fails.
const WORD_DEFAULT Word = 12

// WORD_MINIMUM is the first word the print writes of its own.
const WORD_MINIMUM = uint8(WORD_PACKAGE)

// WORD_MAXIMUM is the last word the print writes of its own.
const WORD_MAXIMUM = uint8(WORD_DEFAULT)

// Word names one keyword the print writes of its own rather than reads from the source. A
// keyword the tree drops is a keyword the print owes, thus each one stands here.
type Word uint8

// Word_Invariants states every keyword the print writes of its own.
func Word_Invariants(value Word, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), WORD_MINIMUM, WORD_MAXIMUM).
		Ensure()
}

// Writes one keyword the print owes.
func emit_word(subject Printer_Handle, word Word) {
	Printer_Handle_Invariants(subject, "emit_word.subject")
	Word_Invariants(word, "emit_word.word")
	// One file binds one package, thus the count of the clauses the print wrote reads one
	// once it closed the file.
	if word == WORD_PACKAGE {
		subject.Counts.Values[COUNT_CLAUSE] = subject.Counts.Values[COUNT_CLAUSE] + 1
	}
	text := string(word_text(word))
	stood := int(written_load(subject.Counts.Values))
	for index := range len(text) {
		emit_byte(subject, Symbol(text[index]))
	}
	// The tree drops the word a declaration opens with, thus the print states it whole or
	// the storage the caller supplied holds no room for it.
	if !bool(flag_load(subject.Flags.Values, FLAG_FULL)) {
		aver.Always(int(written_load(subject.Counts.Values))-stood == len(text),
			"A word the print states lands whole in the form.")
	}
}

// Writes the one declaration the walk stands on, by the rule of its kind.
func print_declaration(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "print_declaration.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "print_declaration.tree")
	stood := written_load(subject.Counts.Values)
	held_depth := depth_load(subject.Counts.Values)
	held_indent := indent_load(subject.Counts.Values)
	layout_run(subject, tree, RULE_DECLARATION, false)
	// A declaration states what it binds, thus the form holds the bytes it wrote for it.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(written_load(subject.Counts.Values) > stood,
			"A declaration writes bytes of its own.")
	}
	// Every form inside the declaration hands back the walk and the tabs it was handed, thus
	// the declaration behind it opens where this one opened.
	aver.Always(depth_load(subject.Counts.Values) == held_depth,
		"A declaration closes at the depth it opened at.")
	aver.Always(indent_load(subject.Counts.Values) == held_indent,
		"A declaration closes at the tabs it opened at.")
}

// Writes one declaration of a kind no rule writes yet: a type, a function, or a statement.
func print_other_declaration(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "print_other_declaration.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "print_other_declaration.tree")
	print_statement(subject, tree)
}

func binds_same_name(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "binds_same_name.yes") }()
	Printer_Handle_Invariants(subject, "binds_same_name.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "binds_same_name.tree")
	depth := depth_load(subject.Counts.Values)
	other := ast.Syntax_Node_At(tree, subject.Nodes.Values[depth])
	own := ast.Syntax_Node_At(tree, subject.Nodes.Values[depth-1])
	if ast.Index(other.First_Child) == ast.INDEX_ABSENT {
		return false
	}
	var flags [FLAG_COUNT]Boolean
	copy(flags[:], subject.Flags.Values)
	var comments [COMMENT_COUNT_MAXIMUM]ast.Index
	copy(comments[:], subject.Comments.Values)
	start := written_load(subject.Counts.Values)
	// The read writes the two names into the form and takes them back, thus every count of
	// what the print wrote stands back with them.
	var counts [COUNT_SLOT_COUNT]Count
	copy(counts[:], subject.Counts.Values)
	// An import states its name where the author wrote one and states the name of its path
	// where the author wrote none.
	subject.Nodes.Values[depth+1] = ast.Index(other.First_Child)
	depth_store(subject.Counts.Values, depth+1)
	enter_named(subject, tree)
	if node_kind(subject, tree) == ast.NODE_IMPORT_NAME {
		emit_token(subject, tree)
	}
	if node_kind(subject, tree) != ast.NODE_IMPORT_NAME {
		emit_package_name(subject, tree)
	}
	middle := written_load(subject.Counts.Values)
	subject.Nodes.Values[depth+1] = ast.Index(own.First_Child)
	enter_named(subject, tree)
	subject.Nodes.Values[depth+1] = ast.Index(
		ast.Syntax_Node_At(tree, subject.Nodes.Values[depth+1]).Next)
	enter_named(subject, tree)
	emit_package_name(subject, tree)
	end := written_load(subject.Counts.Values)
	depth_store(subject.Counts.Values, depth)
	// The measure wrote the names past the form the print stands on, thus the line it stands
	// on holds the columns it held before the measure ran.
	restore_counts(subject, counts[:])
	restore_flags(subject, flags[:])
	copy(subject.Comments.Values, comments[:])
	if bool(flag_load(subject.Flags.Values, FLAG_FULL)) {
		return false
	}
	// A name spells a byte at least, thus the read holds two names of its own and never two
	// empty runs that read as one name.
	aver.Always(int(middle) > int(start),
		"A name one import states writes a byte of its own.")
	aver.Always(int(end) > int(middle),
		"A name one path states writes a byte of its own.")
	form := subject.Form.Values
	return Boolean(string(form[start:middle]) == string(form[middle:end]))
}

// Reads the functions of the file that state the invariants of a type, with the note that opens
// each one. A type reads this table rather than the declarations of the file, thus a file of many
// declarations costs one read of them and no read for each type it holds.
func take_pairs(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "take_pairs.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "take_pairs.tree")
	for slot := range len(subject.Pairs.Functions) {
		subject.Pairs.Functions[slot] = ast.INDEX_ABSENT
		subject.Pairs.Heads[slot] = ast.INDEX_ABSENT
	}
	depth := depth_load(subject.Counts.Values)
	file := subject.Nodes.Values[depth]
	child := ast.Index(ast.Syntax_Node_At(tree, file).First_Child)
	head := ast.INDEX_ABSENT
	count := 0
	for range ast.NODE_COUNT_MAXIMUM {
		if child == ast.INDEX_ABSENT {
			// The table fills from its first slot, thus the slot behind the last pair
			// holds nothing and every read of the table closes there.
			if count < len(subject.Pairs.Functions) {
				aver.Always(subject.Pairs.Functions[count] == ast.INDEX_ABSENT,
					"The slot behind the last pair holds no function.")
			}
			return
		}
		held := ast.Syntax_Node_At(tree, child)
		if held.Kind == ast.NODE_COMMENT {
			if head == ast.INDEX_ABSENT {
				head = child
			}
			child = ast.Index(held.Next)
			continue
		}
		subject.Nodes.Values[depth+1] = child
		depth_store(subject.Counts.Values, depth+1)
		named := names_tail(subject, tree)
		depth_store(subject.Counts.Values, depth)
		if bool(named) {
			if count < len(subject.Pairs.Functions) {
				// The read walks the declarations of the file in the order the
				// author wrote them, thus the table takes them in that order.
				if count > 0 {
					behind := subject.Pairs.Functions[count-1]
					aver.Always(behind < child,
						"A pair stands behind the pair ahead of it.")
				}
				// A pair names the function that states the invariants of
				// one type and the notes standing above it.
				aver.Always(held.Kind == ast.NODE_FUNCTION,
					"A pair of the table names a function of the file.")
				if head != ast.INDEX_ABSENT {
					aver.Always(
						ast.Syntax_Node_At(tree, head).Kind ==
							ast.NODE_COMMENT,
						"A head of one pair names a note of the file.")
				}
				subject.Pairs.Functions[count] = child
				subject.Pairs.Heads[count] = head
				count = count + 1
			}
		}
		head = ast.INDEX_ABSENT
		child = ast.Index(held.Next)
	}
}

// Reports whether the table of pairs holds the declaration the walk stands on, which is what a
// function that stands under a type stands in.
func holds_pair(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "holds_pair.yes") }()
	Printer_Handle_Invariants(subject, "holds_pair.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "holds_pair.tree")
	if !bool(names_tail(subject, tree)) {
		return false
	}
	one := subject.Nodes.Values[depth_load(subject.Counts.Values)]
	// A declaration whose name closes on the tail every function of invariants states is a
	// function itself, thus the read stands on one where it reads the table.
	aver.Always(ast.Syntax_Node_At(tree, one).Kind == ast.NODE_FUNCTION,
		"A read of the table stands on a function the file declares.")
	source := token.Source(subject.Source.Values)
	child := ast.Index(ast.Syntax_Node_At(tree, one).First_Child)
	for range ast.NODE_COUNT_MAXIMUM {
		if child == ast.INDEX_ABSENT {
			return false
		}
		kind := ast.Syntax_Node_At(tree, child).Kind
		if kind != ast.NODE_COMMENT {
			if kind != ast.NODE_BLANK {
				break
			}
		}
		child = ast.Index(ast.Syntax_Node_At(tree, child).Next)
	}
	held := ast.Syntax_Node_At(tree, child)
	name := string(token.Text(source, ast.Syntax_Token_At(tree, held.Token)))
	// A file that names the invariants of one type twice states no one run to stand under
	// it, thus the print writes both where the author wrote them.
	for slot := range len(subject.Pairs.Functions) {
		function := subject.Pairs.Functions[slot]
		if function == ast.INDEX_ABSENT {
			return false
		}
		if function == one {
			return true
		}
		// One type takes one function of invariants under it, thus a second function of
		// that name stands where the author wrote it.
		other := ast.Index(ast.Syntax_Node_At(tree, function).First_Child)
		for range ast.NODE_COUNT_MAXIMUM {
			if other == ast.INDEX_ABSENT {
				break
			}
			kind := ast.Syntax_Node_At(tree, other).Kind
			if kind != ast.NODE_COMMENT {
				if kind != ast.NODE_BLANK {
					break
				}
			}
			other = ast.Index(ast.Syntax_Node_At(tree, other).Next)
		}
		if other == ast.INDEX_ABSENT {
			continue
		}
		named := ast.Syntax_Node_At(tree, other)
		if string(token.Text(source, ast.Syntax_Token_At(tree, named.Token))) == name {
			return false
		}
	}
	return false
}

// SEPARATOR_COMMA_WIDTH is what the comma between two values of one run spends: the comma and
// the space behind it.
const SEPARATOR_COMMA_WIDTH = 2

// SEPARATOR_SIGN_WIDTH is what the sign that binds a run spends: the space, the sign, and the
// space behind them.
const SEPARATOR_SIGN_WIDTH = 3

// SEPARATOR_SHORT_WIDTH is what the short sign that binds a run spends, which states one byte
// past the sign the author could have written.
const SEPARATOR_SHORT_WIDTH = 4

// Reads that the part one run states stands behind what parts it from the part ahead of it: the
// sign that binds the values of a declaration, or the comma that parts two of them.
func reads_separator(subject Printer_Handle) {
	Printer_Handle_Invariants(subject, "reads_separator.subject")
	if !bool(writes_canonical(subject.Flags.Values)) {
		return
	}
	aver.Always(bool(stands_after_space(subject)),
		"A part of one run stands behind the space that parts it.")
}

// Reports whether one child of a value declaration binds a name of the run of names.
func names_one(named Boolean, valued Boolean) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "names_one.yes") }()
	Boolean_Invariants(named, "names_one.named")
	Boolean_Invariants(valued, "names_one.valued")
	if bool(valued) {
		return false
	}
	return named
}

// Reports whether the bracket that closes the list the walk stands in the first part of stands on
// a line of its own. The list itself holds no node, thus the print counts brackets from the part
// it stands on rather than from a bracket a node names.
func closes_list(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "closes_list.yes") }()
	Printer_Handle_Invariants(subject, "closes_list.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "closes_list.tree")
	if bool(flag_load(subject.Flags.Values, FLAG_FLAT)) {
		return false
	}
	depth := 1
	held_depth := depth_load(subject.Counts.Values)
	enter_leftmost(subject, tree)
	// The read steps into the run and never out of it, thus it stands at the slot it opened
	// or below it, and the walk stands back once it read the token.
	aver.Always(depth_load(subject.Counts.Values) >= held_depth,
		"A read of the token one list opens with stands below the slot it opened.")
	leftmost := subject.Nodes.Values[depth_load(subject.Counts.Values)]
	depth_store(subject.Counts.Values, held_depth)
	found := Count(ast.Syntax_Node_At(tree, leftmost).Token)
	opening := int(found)
	for position := opening; position <= ast.TOKEN_INDEX_MAXIMUM; position++ {
		kind := ast.Syntax_Token_At(tree, ast.Token_Index(position)).Kind
		if kind == token.KIND_END_OF_FILE {
			return false
		}
		step := bracket_step(token.Scanned_Kind(kind))
		if step == BRACKET_OPENS {
			depth = depth + 1
		}
		if step != BRACKET_CLOSES {
			continue
		}
		depth = depth - 1
		if depth > 0 {
			continue
		}
		// The read opens at the token the list opens on, thus the bracket that closes
		// the list stands behind that token.
		aver.Always(position > opening,
			"A list closes on a bracket behind the token it opens on.")
		behind := ast.Syntax_Token_At(tree, ast.Token_Index(position-1))
		ahead := ast.Syntax_Token_At(tree, ast.Token_Index(position))
		from := int(behind.Offset) + int(behind.Size)
		// A line feed between the two bytes states a break the author wrote.
		from_offset, to_offset := int(from), int(ahead.Offset)
		for place_offset := from_offset; place_offset < to_offset; place_offset++ {
			if subject.Source.Values[place_offset] == '\n' {
				return true
			}
		}
		return false
	}
	return false
}

// Reports whether the brace of a body stands on the line the bracket of the list the walk stands
// in the first part of closes. A signature the author broke across lines closes on a line of its
// own, thus a list that carries the brace of the body opens one line for that brace.
func closes_signature(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "closes_signature.yes") }()
	Printer_Handle_Invariants(subject, "closes_signature.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "closes_signature.tree")
	depth := 1
	held_depth := depth_load(subject.Counts.Values)
	enter_leftmost(subject, tree)
	leftmost := subject.Nodes.Values[depth_load(subject.Counts.Values)]
	depth_store(subject.Counts.Values, held_depth)
	found := Count(ast.Syntax_Node_At(tree, leftmost).Token)
	closer := 0
	// The run closes at its end token, thus a read that meets it has read the whole run.
	run := tree.Tokens.Values
	first := int(found)
	for position := first; position < ast.TOKEN_COUNT_MAXIMUM; position++ {
		kind := run[position].Kind
		if kind == token.KIND_END_OF_FILE {
			break
		}
		if bool(opens_bracket(token.Scanned_Kind(kind))) {
			depth = depth + 1
		}
		held := token.Scanned_Kind(kind)
		if bool(closes_bracket(held)) {
			depth = depth - 1
			if depth == 0 {
				closer = position
				break
			}
		}
	}
	if closer == 0 {
		return false
	}
	// The read stops at the bracket that closes the list the signature states.
	aver.Always(bool(closes_bracket(token.Scanned_Kind(run[closer].Kind))),
		"A signature closes its list on a bracket that closes one.")
	// Every bracket the signature states behind the one that closed the list closes again
	// before the body opens, thus the read counts them and stops at the first brace that
	// stands outside them all.
	behind := ast.Syntax_Token_At(tree, ast.Token_Index(closer))
	depth = 0
	for position := closer + 1; position < ast.TOKEN_COUNT_MAXIMUM; position++ {
		one := run[position]
		if one.Kind == token.KIND_END_OF_FILE {
			return false
		}
		if one.Kind == token.KIND_BRACE_LEFT {
			if depth == 0 {
				// A line feed between the two states a break the author wrote.
				place := int(behind.Offset) + int(behind.Size)
				for ; place < int(one.Offset); place++ {
					if subject.Source.Values[place] == '\n' {
						return false
					}
				}
				return true
			}
		}
		step := bracket_step(token.Scanned_Kind(one.Kind))
		if step == BRACKET_OPENS {
			depth = depth + 1
		}
		if step == BRACKET_CLOSES {
			depth = depth - 1
		}
	}
	return false
}

// RESULT_COUNT_MINIMUM is the count a return that wrote no name yet holds.
const RESULT_COUNT_MINIMUM = 0

// RESULT_WRITTEN_MAXIMUM is the count a return holds as it writes the last name a signature binds,
// which is one below the names it admits.
const RESULT_WRITTEN_MAXIMUM = RESULT_COUNT_MAXIMUM - 1

// Result_Count is how many names of one signature a return already wrote.
type Result_Count int

// Result_Count_Invariants states every count of names one return writes.
func Result_Count_Invariants(value Result_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), RESULT_COUNT_MINIMUM, RESULT_WRITTEN_MAXIMUM).
		Ensure()
}

// Opens one name a return states for the signature that bound it: the comma that closes the name
// before it, where one stands, and the space that holds the two apart.
func open_result_name(subject Printer_Handle, written Result_Count) {
	Printer_Handle_Invariants(subject, "open_result_name.subject")
	Result_Count_Invariants(written, "open_result_name.written")
	if written > RESULT_COUNT_MINIMUM {
		emit_comma(subject)
	}
	emit_space(subject)
}

// WORD_STATEMENT_MINIMUM is the first statement of one word the print writes past a return, which
// is the go statement.
const WORD_STATEMENT_MINIMUM = uint8(ast.NODE_GO)

// WORD_STATEMENT_MAXIMUM is the last statement of one word the canonical form holds, which is the
// goto. The dialect refuses the fallthrough behind it, thus a tree the parse holds states none.
const WORD_STATEMENT_MAXIMUM = uint8(ast.NODE_GOTO)

// Word_Statement_Kind names which statement of one word the print writes: a go, a defer, a break,
// a continue, a goto, or a fallthrough. A return states the values it hands back and stands apart.
type Word_Statement_Kind uint8

// Word_Statement_Kind_Invariants states every statement of one word past the return.
func Word_Statement_Kind_Invariants(value Word_Statement_Kind, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), WORD_STATEMENT_MINIMUM, WORD_STATEMENT_MAXIMUM).
		Ensure()
}

// Writes the result names the signature around a return binds, behind a return that names no value
// of its own. The nearest function around the return states those names in its results. A name
// spelled as the blank names nothing a return can read, and a signature that binds more names than
// a return writes states a form no return writes, thus either one writes the word alone.
func emit_result_names(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "emit_result_names.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "emit_result_names.tree")
	if node_kind(subject, tree) != ast.NODE_RETURN {
		return
	}
	function := ast.INDEX_ABSENT
	for level := int(depth_load(subject.Counts.Values)) - 1; level >= 0; level-- {
		one := subject.Nodes.Values[level]
		switch ast.Syntax_Node_At(tree, one).Kind {
		case ast.NODE_FUNCTION, ast.NODE_FUNCTION_LITERAL:
			function = one
			level = -1
		}
	}
	if function == ast.INDEX_ABSENT {
		return
	}
	source, stood := token.Source(subject.Source.Values), written_load(subject.Counts.Values)
	counted := Result_Count(RESULT_COUNT_MINIMUM)
	for pass := range 2 {
		names := Result_Count(RESULT_COUNT_MINIMUM)
		child := ast.Index(ast.Syntax_Node_At(tree, function).First_Child)
		for child != ast.INDEX_ABSENT {
			node := ast.Syntax_Node_At(tree, child)
			child = ast.Index(node.Next)
			if node.Kind != ast.NODE_RESULT {
				continue
			}
			name := ast.Index(node.First_Child)
			for name != ast.INDEX_ABSENT {
				held := ast.Syntax_Node_At(tree, name)
				name = ast.Index(held.Next)
				switch held.Kind {
				case ast.NODE_BLANK, ast.NODE_COMMENT:
					continue
				}
				if held.Kind != ast.NODE_PARAMETER_NAME {
					break
				}
				text := token.Text(source, ast.Syntax_Token_At(tree, held.Token))
				if string(text) == "_" {
					return
				}
				if names == RESULT_COUNT_MAXIMUM {
					return
				}
				if pass == 1 {
					open_result_name(subject, names)
					for index := range len(text) {
						emit_byte(subject, Symbol(text[index]))
					}
				}
				names = names + 1
			}
		}
		if pass == 1 {
			aver.Always(names == counted,
				"Both passes over the results of one signature read one run.")
		}
		counted = names
	}
	if counted > RESULT_COUNT_MINIMUM {
		if bool(writes_canonical(subject.Flags.Values)) {
			aver.Always(written_load(subject.Counts.Values) > stood,
				"A return of no value writes the names its signature bound.")
		}
	}
}

// Reports whether one signature states a single result that binds no name. Parentheses around
// such a result state nothing the reader needs, thus the canonical form drops them.
func bare_result(subject Printer_Handle, tree ast.Syntax_Tree_Handle, open Boolean) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "bare_result.yes") }()
	Printer_Handle_Invariants(subject, "bare_result.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "bare_result.tree")
	Boolean_Invariants(open, "bare_result.open")
	written := written_load(subject.Counts.Values)
	defer func() {
		aver.Always(written_load(subject.Counts.Values) == written,
			"A read of the result one signature states alone writes no byte.")
	}()
	if !bool(open) {
		return false
	}
	if node_kind(subject, tree) != ast.NODE_RESULT {
		return false
	}
	if bool(named_result(subject, tree)) {
		return false
	}
	stood := depth_load(subject.Counts.Values)
	held := subject.Nodes.Values[stood]
	alone := Boolean(true)
	for bool(advance_quiet(subject, tree)) {
		if node_kind(subject, tree) == ast.NODE_RESULT {
			alone = false
			break
		}
	}
	subject.Nodes.Values[depth_load(subject.Counts.Values)] = held
	// The read steps along the run with a step that holds the slot it stood on, thus it
	// hands back the walk it opened.
	aver.Always(depth_load(subject.Counts.Values) == stood,
		"A read of one run of results holds the slot it opened.")
	return alone
}

// Reports whether the result the walk stands on binds a name.
func named_result(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "named_result.yes") }()
	Printer_Handle_Invariants(subject, "named_result.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "named_result.tree")
	written := written_load(subject.Counts.Values)
	defer func() {
		aver.Always(written_load(subject.Counts.Values) == written,
			"A read of the names one signature binds writes no byte of its own.")
	}()
	if !bool(descend_quiet(subject, tree)) {
		return false
	}
	held := Boolean(node_kind(subject, tree) == ast.NODE_PARAMETER_NAME)
	ascend(subject)
	return held
}

// Writes one block by the rule that writes blocks.
func print_block(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "print_block.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "print_block.tree")
	// A block holds its statements between braces, thus the walk stands on a block where the
	// print writes one.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(node_kind(subject, tree) == ast.NODE_BLOCK,
			"A block the print writes stands on a block of the tree.")
	}
	// A block indents the statements it holds and closes its brace at the tabs it opened at,
	// thus the tabs it hands back are the ones it was handed.
	held := indent_load(subject.Counts.Values)
	stood := depth_load(subject.Counts.Values)
	opened := int(written_load(subject.Counts.Values))
	tight := flag_load(subject.Flags.Values, FLAG_TIGHT)
	subject.Counts.Values[COUNT_BLOCK] += 1
	layout_run(subject, tree, RULE_BLOCK, false)
	if bool(writes_canonical(subject.Flags.Values)) {
		// A block opens on the brace that holds its statements.
		aver.Always(subject.Form.Values[opened] == '{',
			"A block opens on the brace that holds its statements.")
		// The brace closes the statements the block holds, thus the last byte the block
		// wrote states that brace.
		aver.Always(subject.Form.Values[int(written_load(subject.Counts.Values))-1] == '}',
			"A block closes on the brace that holds its statements.")
		aver.Always(indent_load(subject.Counts.Values) == held,
			"A block closes at the tabs it opened at.")
		// Every statement the block holds hands back the walk it was handed, thus the
		// block closes on the node it opened on.
		aver.Always(depth_load(subject.Counts.Values) == stood,
			"A block closes on the walk it opened.")
		// The block the body mark names stands written, thus the mark closes with it and
		// no block behind it reads the mark this one was handed.
		aver.Always(!bool(flag_load(subject.Flags.Values, FLAG_BODY)),
			"A block closes the body mark it stood under.")
		aver.Always(!bool(flag_load(subject.Flags.Values, FLAG_SHORT)),
			"A block closes the short sign it stood under.")
		// A block states its own answer for the lines against its braces and hands back
		// the answer the form around it stood under.
		aver.Always(flag_load(subject.Flags.Values, FLAG_TIGHT) == tight,
			"A block closes the tightness it stood under.")
	}
}

// Pushes onto the walk the statement the node the walk stands on belongs to: the nearest node
// around it, itself counted, that stands in a block, in a case, or behind a label, and the node
// that holds that one beneath it. The walk stays where it stands where no such node holds it.
func enter_statement(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "enter_statement.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "enter_statement.tree")
	depth := depth_load(subject.Counts.Values)
	if int(depth)+2 >= DEPTH_MAXIMUM {
		return
	}
	for statement := depth; statement > 0; statement-- {
		held := Statement_Holder(false)
		switch ast.Syntax_Node_At(tree, subject.Nodes.Values[statement-1]).Kind {
		case ast.NODE_BLOCK, ast.NODE_CASE, ast.NODE_DEFAULT, ast.NODE_LABEL:
			held = true
		}
		Statement_Holder_Invariants(held, "enter_statement.held")
		if bool(held) {
			subject.Nodes.Values[depth+1] = subject.Nodes.Values[statement-1]
			subject.Nodes.Values[depth+2] = subject.Nodes.Values[statement]
			depth_store(subject.Counts.Values, depth+2)
			// The walk holds the node that holds each one below it, thus the
			// statement the step stands on hangs under the node it opened.
			stood := ast.Syntax_Node_At(tree, subject.Nodes.Values[depth+2])
			aver.Always(ast.Index(stood.Parent) == subject.Nodes.Values[depth+1],
				"A step to one statement stands on a part of its holder.")
			return
		}
	}
}

// Statement_Holder names a node that holds statements of its own: a block, a case, and the label
// that names the statement behind it.
type Statement_Holder Boolean

// Statement_Holder_Invariants states both stands of the node.
func Statement_Holder_Invariants(value Statement_Holder, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The node holds statements of its own.").
		Ensure()
}

// KEPT_NONE names an assignment that writes none of its names, which is a range clause that binds
// nothing or binds only names that read nothing.
const KEPT_NONE = 0

// KEPT_FIRST names a range clause that writes its key and drops the value it binds to nothing.
const KEPT_FIRST = 1

// KEPT_ALL names an assignment that writes every name the author wrote.
const KEPT_ALL = 2

// Kept_Names is which names of one assignment the print writes. A range clause binds the names the
// author wrote to values the body may read, thus a name that reads nothing states nothing and the
// canonical form drops it along with the sign it stood behind. A range clause binds two names at
// most, thus it drops the value alone or both names.
type Kept_Names int

// Kept_Names_Invariants states every run of names one assignment writes.
func Kept_Names_Invariants(value Kept_Names, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Int(int(value), KEPT_NONE, KEPT_FIRST, KEPT_ALL).
		Ensure()
}

// Reads which names of the assignment the walk stands on the print writes.
func take_names(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (kept Kept_Names) {
	defer func() { Kept_Names_Invariants(kept, "take_names.kept") }()
	Printer_Handle_Invariants(subject, "take_names.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "take_names.tree")
	held := subject.Nodes.Values[depth_load(subject.Counts.Values)]
	depth := depth_load(subject.Counts.Values)
	count := Part_Count(PART_COUNT_NONE)
	ranged := Boolean(false)
	first := ast.Token_Index(0)
	last := ast.Token_Index(0)
	open := descend(subject, tree)
	for bool(open) {
		// A note the author left among the names states none of them, thus the read
		// steps over it rather than reading it as the value the sign binds.
		if !bool(trivia_kind(subject, tree)) {
			if bool(opens_bound(subject, tree)) {
				ranged = Boolean(node_kind(subject, tree) == ast.NODE_RANGE)
				break
			}
			one := subject.Nodes.Values[depth_load(subject.Counts.Values)]
			last = ast.Syntax_Node_At(tree, one).Token
			if count == PART_COUNT_NONE {
				first = last
			}
			if count != PART_COUNT_THREE {
				count = count + 1
			}
		}
		open = advance(subject, tree)
	}
	depth_store(subject.Counts.Values, depth)
	subject.Nodes.Values[depth] = held
	// An assignment that binds no name at all writes none.
	if count == PART_COUNT_NONE {
		return KEPT_NONE
	}
	// The names one assignment binds stand in the run of tokens in the order the walk meets
	// them, thus the first of them opens no later than the last.
	aver.Always(first <= last,
		"The first name one assignment binds opens ahead of the last it binds.")
	source := token.Source(subject.Source.Values)
	blank_head := Boolean(string(token.Text(source, ast.Syntax_Token_At(tree, first))) == "_")
	blank_tail := Boolean(string(token.Text(source, ast.Syntax_Token_At(tree, last))) == "_")
	return kept_names(Name_Count(count), ranged, blank_head, blank_tail)
}

// NAME_COUNT_ONE names an assignment that binds one name.
const NAME_COUNT_ONE = 1

// NAME_COUNT_TWO names one that binds two.
const NAME_COUNT_TWO = 2

// NAME_COUNT_MANY names one that binds three names or more.
const NAME_COUNT_MANY = 3

// Name_Count is how many names one assignment binds, counted up to three.
type Name_Count uint8

// Name_Count_Invariants states every count of names one assignment binds.
func Name_Count_Invariants(value Name_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(uint8(value), NAME_COUNT_ONE, NAME_COUNT_TWO, NAME_COUNT_MANY).
		Ensure()
}

// Names which names of one assignment the print writes. A range clause binds the names the author
// wrote to values the body may read, thus a name that reads nothing states nothing, and a clause
// that binds its key alone writes that key.
func kept_names(
	count Name_Count, ranged Boolean, blank_head Boolean, blank_tail Boolean,
) (kept Kept_Names) {
	defer func() { Kept_Names_Invariants(kept, "kept_names.kept") }()
	Name_Count_Invariants(count, "kept_names.count")
	Boolean_Invariants(ranged, "kept_names.ranged")
	Boolean_Invariants(blank_head, "kept_names.blank_head")
	Boolean_Invariants(blank_tail, "kept_names.blank_tail")
	if !bool(ranged) {
		return KEPT_ALL
	}
	if count == NAME_COUNT_TWO {
		if !bool(blank_tail) {
			return KEPT_ALL
		}
		if !bool(blank_head) {
			return KEPT_FIRST
		}
		return KEPT_NONE
	}
	if bool(blank_head) {
		return KEPT_NONE
	}
	return KEPT_ALL
}

// Pushes onto the walk the nearest assignment around the node the walk stands on. The walk stays
// where it stands where no assignment holds it.
func enter_assignment(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "enter_assignment.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "enter_assignment.tree")
	depth := depth_load(subject.Counts.Values)
	if int(depth)+1 >= DEPTH_MAXIMUM {
		return
	}
	for statement := depth - 1; statement >= 0; statement-- {
		held := Bound_Values(false)
		switch ast.Syntax_Node_At(tree, subject.Nodes.Values[statement]).Kind {
		case ast.NODE_ASSIGN, ast.NODE_DEFINE, ast.NODE_OPERATION_ASSIGN, ast.NODE_SEND:
			held = true
		}
		Bound_Values_Invariants(held, "enter_assignment.held")
		if bool(held) {
			subject.Nodes.Values[depth+1] = subject.Nodes.Values[statement]
			depth_store(subject.Counts.Values, depth+1)
			return
		}
	}
}

// Bound_Values names a node that binds a run of values to the names ahead of it, which is what
// every assignment of the tree states by the sign it holds.
type Bound_Values Boolean

// Bound_Values_Invariants states both stands of the node.
func Bound_Values_Invariants(value Bound_Values, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The node binds a run of values.").
		Ensure()
}

// Reports whether the form the walk stands on holds two parts alone, counting neither the empty
// lines nor the notes standing between them. A check of two parts states a condition and a block
// and no branch behind them.
func holds_two_parts(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "holds_two_parts.yes") }()
	Printer_Handle_Invariants(subject, "holds_two_parts.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "holds_two_parts.tree")
	stood := written_load(subject.Counts.Values)
	defer func() {
		aver.Always(written_load(subject.Counts.Values) == stood,
			"A read of the two parts one form holds writes no byte of its own.")
	}()
	held := subject.Nodes.Values[depth_load(subject.Counts.Values)]
	depth := depth_load(subject.Counts.Values)
	parts := 0
	open := descend(subject, tree)
	for bool(open) {
		if !bool(trivia_kind(subject, tree)) {
			parts = parts + 1
		}
		open = advance(subject, tree)
	}
	depth_store(subject.Counts.Values, depth)
	subject.Nodes.Values[depth] = held
	return Boolean(parts == 2)
}

// Moves the walk from a note or an empty line to the last statement ahead of it in the node that
// holds both, which the walk stands on beneath it. A statement stays where it stands, and so does
// trivia no statement stands ahead of.
func enter_prior(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "enter_prior.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "enter_prior.tree")
	if !bool(trivia_kind(subject, tree)) {
		return
	}
	depth := depth_load(subject.Counts.Values)
	statement := subject.Nodes.Values[depth]
	holder := ast.Syntax_Node_At(tree, subject.Nodes.Values[depth-1])
	subject.Nodes.Values[depth] = ast.Index(holder.First_Child)
	prior := statement
	for range ast.NODE_COUNT_MAXIMUM {
		if subject.Nodes.Values[depth] == statement {
			break
		}
		if !bool(trivia_kind(subject, tree)) {
			prior = subject.Nodes.Values[depth]
		}
		if !bool(advance(subject, tree)) {
			break
		}
	}
	subject.Nodes.Values[depth] = prior
}

// Reports whether the empty line the walk stands on stands ahead of the simple error check behind
// the statement that holds it. The author writes the check of an error against the call that
// states it, thus the empty line between the two states nothing.
func stands_before_check(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "stands_before_check.yes") }()
	Printer_Handle_Invariants(subject, "stands_before_check.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "stands_before_check.tree")
	stood := written_load(subject.Counts.Values)
	defer func() {
		aver.Always(written_load(subject.Counts.Values) == stood,
			"A read of the check one line stands ahead of writes no byte.")
	}()
	if node_kind(subject, tree) != ast.NODE_BLANK {
		return false
	}
	// The empty line names the token ahead of it. A semicolon a line feed stands for spans no
	// byte, and a note between the two forms stands with the check rather than with the value
	// the check reads, thus the read steps over both to the word the check opens with.
	node := ast.Syntax_Node_At(tree, subject.Nodes.Values[depth_load(subject.Counts.Values)])
	position := int(node.Token)
	// The end token is neither a note nor an empty semicolon, thus the read stops there at the
	// latest.
	for range ast.TOKEN_COUNT_MAXIMUM {
		one := ast.Syntax_Token_At(tree, ast.Token_Index(position))
		held := one.Kind == token.KIND_COMMENT
		if one.Size == 0 {
			held = one.Kind == token.KIND_SEMICOLON
		}
		if !held {
			break
		}
		position = position + 1
	}
	// The read steps over every note standing between the two forms, thus it stops on a
	// token that states none.
	aver.Always(ast.Syntax_Token_At(tree, ast.Token_Index(position)).Kind !=
		token.KIND_COMMENT,
		"A read for the check ahead stops past the notes between the two forms.")
	held_depth := depth_load(subject.Counts.Values)
	enter_statement(subject, tree)
	if depth_load(subject.Counts.Values) == held_depth {
		return false
	}
	enter_prior(subject, tree)
	found := false
	if bool(assigns_error(subject, tree)) {
		// A note between the two forms names the check rather than the value it stands
		// behind, thus the walk steps over the trivia to the statement written next.
		open := advance(subject, tree)
		for bool(open) {
			if !bool(trivia_kind(subject, tree)) {
				break
			}
			open = advance(subject, tree)
		}
		if bool(open) {
			check := ast.Syntax_Node_At(tree, subject.Nodes.Values[held_depth+2]).Token
			found = bool(states_check(subject, tree)) && int(check) == position
		}
	}
	depth_store(subject.Counts.Values, held_depth)
	return Boolean(found)
}

// Reports whether the node the walk stands on is an empty line the scanner counted behind the
// brace that closes the block rather than inside it.
func closes_block(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "closes_block.yes") }()
	Printer_Handle_Invariants(subject, "closes_block.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "closes_block.tree")
	if node_kind(subject, tree) != ast.NODE_BLANK {
		return false
	}
	// An empty line the scanner counted at a token the brace of the block closes ahead of
	// stands inside the block, and one it counted past that brace stands behind the block.
	return stands_behind(subject, tree)
}

// Writes one statement of a block, on a line of its own.
func print_statement(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "print_statement.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "print_statement.tree")
	if bool(print_bare_statement(subject, tree)) {
		return
	}
	// A statement that the author broke across lines indents its continuations, and the tabs
	// it owed stand again at the statement behind it.
	held := indent_load(subject.Counts.Values)
	flag_store(subject.Flags.Values, FLAG_BROKEN, false)
	// A label names the statement behind it rather than stand inside it, thus it stands one
	// tab shallower than the statements around it and closes no line of its own.
	kind := node_kind(subject, tree)
	labelled := kind == ast.NODE_LABEL
	if labelled {
		indent := indent_load(subject.Counts.Values)
		if indent > 0 {
			indent_store(subject.Counts.Values, indent-1)
		}
	}
	// A clause closes its own line, because the comments behind its body take lines of their
	// own rather than the end of the line its brace closes.
	closed := labelled || bool(closes_own_line(subject, tree))
	emit_indent(subject)
	stood := depth_load(subject.Counts.Values)
	written := written_load(subject.Counts.Values)
	print_statement_form(subject, tree)
	// A statement states a form of its own, thus the line it opens holds bytes of it.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(written_load(subject.Counts.Values) > written,
			"A statement writes bytes of its own.")
	}
	// A statement opens a form of its own, and every form inside it hands back the nest it
	// was handed, thus the statement closes at the nest it opened at, holding no run of keys
	// a literal inside it opened.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(nest_load(subject.Counts.Values) == NEST_MINIMUM,
			"A statement closes at the nest it opened at.")
		aver.Always(key_width_load(subject.Counts.Values) == 0,
			"A statement closes the run of keys it opened.")
		aver.Always(depth_load(subject.Counts.Values) == stood,
			"A statement closes on the walk it opened.")
		aver.Always(!bool(flag_load(subject.Flags.Values, FLAG_SPLIT)),
			"A statement closes the run of signs it broke.")
		aver.Always(!bool(flag_load(subject.Flags.Values, FLAG_SHORT)),
			"A statement closes the short sign it stood under.")
	}
	if !closed {
		// A statement that holds no part of its own carries its note as a sibling, thus
		// the print takes that note before it closes the line the statement stands on.
		take_statement_note(subject, tree)
		close_line(subject, tree)
	}
	indent_store(subject.Counts.Values, held)
	flag_store(subject.Flags.Values, FLAG_BROKEN, false)
}

// Writes one statement that states no form of its own: an empty line, a note, a declaration, or a
// block. It reports whether it wrote one, because every other statement writes a form.
func print_bare_statement(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (held Boolean) {
	defer func() { Boolean_Invariants(held, "print_bare_statement.held") }()
	Printer_Handle_Invariants(subject, "print_bare_statement.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "print_bare_statement.tree")
	switch node_kind(subject, tree) {
	case ast.NODE_BLANK:
		if bool(stands_against(subject, tree)) {
			return true
		}
		if bool(stands_before_check(subject, tree)) {
			return true
		}
		close_line(subject, tree)
		return true
	case ast.NODE_COMMENT:
		stood := written_load(subject.Counts.Values)
		emit_indent(subject)
		emit_token(subject, tree)
		// A note that stands as a statement of its own spells the bytes the author wrote.
		if bool(writes_canonical(subject.Flags.Values)) {
			aver.Always(written_load(subject.Counts.Values) > stood,
				"A note standing as one statement writes the note it states.")
		}
		close_line(subject, tree)
		return true
	case ast.NODE_DECLARATION_STATEMENT:
		counts_binding(subject, tree)
		print_declaration_statement(subject, tree)
		return true
	case ast.NODE_BLOCK:
		emit_indent(subject)
		print_block(subject, tree)
		close_line(subject, tree)
		return true
	}
	return false
}

// Writes the expression one statement stands for, which is the one child it holds.
func print_inner(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "print_inner.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "print_inner.tree")
	// A statement of one expression writes that expression alone, thus the walk stands on
	// such a statement where the print writes one.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(node_kind(subject, tree) == ast.NODE_EXPRESSION_STATEMENT,
			"A statement of one expression stands on such a statement.")
	}
	if !bool(descend_part(subject, tree)) {
		return
	}
	// A statement of one expression holds that expression alone where the source spells a
	// statement the language admits, and the print writes every value a wider one holds
	// rather than dropping what the author wrote.
	print_expression(subject, tree)
	for bool(advance_part(subject, tree)) {
		emit_comma(subject)
		emit_space(subject)
		print_expression(subject, tree)
	}
	finish_parts(subject, tree)
	ascend(subject)
}

// Writes one local declaration, which states the same form a file-level one does.
func print_declaration_statement(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "print_declaration_statement.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "print_declaration_statement.tree")
	if !bool(descend_part(subject, tree)) {
		return
	}
	held_short := flag_load(subject.Flags.Values, FLAG_SHORT)
	flag_store(subject.Flags.Values, FLAG_SHORT, states_short(subject, tree))
	// A statement that states a declaration writes that declaration alone, thus the walk
	// stands on the declaration it holds rather than on trivia standing inside it.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(!bool(trivia_kind(subject, tree)),
			"A statement of one declaration stands on the declaration it states.")
	}
	// A body states declarations of the same kinds the file states, thus the count of each
	// kind reads every one the tree holds rather than the ones the file binds alone.
	counts_declaration(subject, tree)
	print_declaration(subject, tree)
	finish_parts(subject, tree)
	// The declaration closed its own line, thus the notes the walk stepped over behind it
	// take lines of their own rather than the end of the line the statement behind it writes.
	close_debts(subject, tree)
	ascend(subject)
	flag_store(subject.Flags.Values, FLAG_SHORT, held_short)
}

// Writes one assignment: the values on its left, the sign the source states, and the values on
// its right.
func print_assignment(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "print_assignment.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "print_assignment.tree")
	// An assignment binds a run of values to a run of names, and the sign it states names
	// which binding it holds, thus the walk stands on one of them where the print writes one.
	if bool(writes_canonical(subject.Flags.Values)) {
		Assignment_Kind_Invariants(assignment_kind(subject, tree), "print_assignment.bound")
	}
	// An assignment of a run of values against a run of names states each value as one part of
	// a wider form, thus its signs close up the way the parts of any other form do.
	stood := depth_load(subject.Counts.Values)
	held_nest := nest_load(subject.Counts.Values)
	nest := held_nest
	if bool(assigns_runs(subject, tree)) {
		nest = Nest(nest_inner(subject, nest))
	}
	// The values of the assignment stand at the nest it opens for them, and the statement
	// behind it stands at the nest it was handed.
	defer nest_store(subject.Counts.Values, held_nest)
	kept := take_names(subject, tree)
	named := Named_Run(kept != KEPT_NONE)
	if !bool(descend_part(subject, tree)) {
		return
	}
	// The tree holds no node for the sign of an assignment, thus the print reads it from the
	// token that stands ahead of the first value on the right.
	written := Value_Written(false)
	open := Boolean(true)
	// The names on the left indent their own continuations, and the values on the right answer
	// to the statement rather than to the line the names ended on.
	indent := indent_load(subject.Counts.Values)
	marked := flag_load(subject.Flags.Values, FLAG_BROKEN)
	for bool(open) {
		right := opens_right(subject, tree)
		// A name a range clause binds to nothing states nothing, thus the print steps
		// over it and over the sign that stood behind the run it closed.
		if !bool(right) {
			if bool(keeps_names(kept, written)) {
				// A name the print steps over states nothing at all, thus the
				// form holds no byte of it and the count reads it dropped.
				counts_dropped_type(subject, tree)
				stepped := written_load(subject.Counts.Values)
				open = advance_part(subject, tree)
				aver.Always(written_load(subject.Counts.Values) == stepped,
					"A name a range binds to nothing writes no byte.")
				continue
			}
		}
		if bool(right) {
			open_sign(subject, tree, named)
			indent_store(subject.Counts.Values, indent)
			flag_store(subject.Flags.Values, FLAG_BROKEN, marked)
			written = false
		}
		nest_store(subject.Counts.Values, nest)
		print_assigned_value(subject, tree, written, named, right)
		written = true
		open = advance_part(subject, tree)
	}
	finish_parts(subject, tree)
	ascend(subject)
	// Every value the assignment binds hands back the walk it was handed, thus the statement
	// closes on the node the assignment opened on.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(depth_load(subject.Counts.Values) == stood,
			"An assignment closes on the walk it opened.")
	}
}

// Writes one value of an assignment: the comma that holds it off the value ahead of it, the break
// the author left, and the form itself. A value the author wrote on a line of its own keeps that
// line, and it stands one tab deeper than the statement that holds it. The first name on the left
// opens the statement, thus it takes neither that break nor the space a value behind a sign takes.
func print_assigned_value(
	subject Printer_Handle, tree ast.Syntax_Tree_Handle, written Value_Written,
	named Named_Run, right Boolean,
) {
	Printer_Handle_Invariants(subject, "print_assigned_value.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "print_assigned_value.tree")
	Value_Written_Invariants(written, "print_assigned_value.written")
	Named_Run_Invariants(named, "print_assigned_value.named")
	Boolean_Invariants(right, "print_assigned_value.right")
	if bool(written) {
		emit_comma(subject)
	}
	behind := Boolean(written)
	if bool(right) {
		behind = Boolean(named)
	}
	open_value(subject, tree, behind, right)
	stood := written_load(subject.Counts.Values)
	print_expression(subject, tree)
	// A value one assignment binds stands for a form the author wrote, thus the print
	// writes the bytes of it.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(written_load(subject.Counts.Values) > stood,
			"A value of one assignment writes bytes of its own.")
	}
}

// ASSIGNMENT_BINDS names an assignment that binds values to names the source already stated.
const ASSIGNMENT_BINDS = 0

// ASSIGNMENT_DEFINES names an assignment that binds values to names it states itself.
const ASSIGNMENT_DEFINES = 1

// ASSIGNMENT_OPERATES names an assignment that reads its names and binds what the sign states.
const ASSIGNMENT_OPERATES = 2

// ASSIGNMENT_SENDS names a send, which binds one value to the channel it names.
const ASSIGNMENT_SENDS = 3

// Assignment_Kind names which binding one assignment states by the sign it holds.
type Assignment_Kind uint8

// Assignment_Kind_Invariants states every binding one assignment holds.
func Assignment_Kind_Invariants(value Assignment_Kind, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_4_Uint8(uint8(value), ASSIGNMENT_BINDS, ASSIGNMENT_DEFINES,
			ASSIGNMENT_OPERATES, ASSIGNMENT_SENDS).
		Ensure()
}

// Reads which binding the assignment the walk stands on states.
func assignment_kind(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (bound Assignment_Kind) {
	defer func() { Assignment_Kind_Invariants(bound, "assignment_kind.bound") }()
	Printer_Handle_Invariants(subject, "assignment_kind.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "assignment_kind.tree")
	if node_kind(subject, tree) == ast.NODE_ASSIGN {
		return ASSIGNMENT_BINDS
	}
	if node_kind(subject, tree) == ast.NODE_DEFINE {
		return ASSIGNMENT_DEFINES
	}
	if node_kind(subject, tree) == ast.NODE_OPERATION_ASSIGN {
		return ASSIGNMENT_OPERATES
	}
	return ASSIGNMENT_SENDS
}

// Writes the sign of one assignment and the space that holds it off the names before it. An
// assignment that writes no name of its own states its sign against nothing, thus the print
// writes neither.
func open_sign(subject Printer_Handle, tree ast.Syntax_Tree_Handle, named Named_Run) {
	Printer_Handle_Invariants(subject, "open_sign.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "open_sign.tree")
	Named_Run_Invariants(named, "open_sign.named")
	if !bool(named) {
		return
	}
	emit_space(subject)
	stood := written_load(subject.Counts.Values)
	emit_sign(subject, tree)
	// An assignment states the sign that binds its values, thus the sign spells bytes of its
	// own behind the names the assignment wrote.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(written_load(subject.Counts.Values) > stood,
			"The sign of one assignment spells bytes of its own.")
	}
}

// Named_Run names an assignment that writes a name of its own, which is every one but a range
// clause that binds each name it states to nothing.
type Named_Run Boolean

// Named_Run_Invariants states both runs of names one assignment writes.
func Named_Run_Invariants(value Named_Run, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The assignment writes a name of its own.").
		Ensure()
}

// Value_Written names a run of one assignment that already wrote a value.
type Value_Written Boolean

// Value_Written_Invariants states both stands of the run.
func Value_Written_Invariants(value Value_Written, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The run already wrote a value.").
		Ensure()
}

// Reports whether the run of names already wrote every name the print keeps. A range clause that
// binds a name to nothing writes no name at all, and one that keeps its key writes that key
// alone, thus the run closes at the name the print keeps last.
func keeps_names(kept Kept_Names, written Value_Written) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "keeps_names.yes") }()
	Kept_Names_Invariants(kept, "keeps_names.kept")
	Value_Written_Invariants(written, "keeps_names.written")
	if kept == KEPT_NONE {
		return true
	}
	if kept == KEPT_ALL {
		return false
	}
	return Boolean(written)
}

// Opens the line one value of an assignment stands on: the break the author wrote ahead of it, or
// the space that holds it against the value before it. The value a sign binds stands on the line
// the sign closes, thus a break the author wrote behind the sign itself states nothing at all.
func open_value(
	subject Printer_Handle, tree ast.Syntax_Tree_Handle, behind Boolean, right Boolean,
) {
	Printer_Handle_Invariants(subject, "open_value.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "open_value.tree")
	Boolean_Invariants(behind, "open_value.behind")
	Boolean_Invariants(right, "open_value.right")
	here := Boolean(false)
	if bool(behind) {
		if !bool(right) {
			here = breaks_before(subject, tree)
		}
	}
	open_continuation(subject, tree, here)
	if bool(behind) {
		if !bool(here) {
			emit_space(subject)
		}
	}
	// A value that stands behind another one stands apart from it, thus a space or the tabs
	// of a line of its own stand between the two.
	if bool(behind) {
		if bool(writes_canonical(subject.Flags.Values)) {
			aver.Always(bool(stands_after_space(subject)),
				"A value behind another stands apart from it.")
		}
	}
}

// Moves the walk from the form it stands on down to the node that names the first token the form
// spans. A node names the token its own form opens with, and an operation names its sign and never
// its left value, thus the walk steps into each first child that names an earlier token. A form
// whose first child names a later token opens at its own token, and so does every child of it.
func enter_leftmost(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "enter_leftmost.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "enter_leftmost.tree")
	for range DEPTH_MAXIMUM {
		depth := depth_load(subject.Counts.Values)
		node := ast.Syntax_Node_At(tree, subject.Nodes.Values[depth])
		child := ast.Index(node.First_Child)
		if child == ast.INDEX_ABSENT {
			return
		}
		if ast.Syntax_Node_At(tree, child).Token >= node.Token {
			return
		}
		if int(depth)+1 >= DEPTH_MAXIMUM {
			return
		}
		// The walk steps down into the chain that hangs under the node it stands on.
		aver.Always(ast.Index(ast.Syntax_Node_At(tree, child).Parent) ==
			subject.Nodes.Values[depth],
			"A step down to the first part stands on a child of the node it left.")
		subject.Nodes.Values[depth+1] = child
		depth_store(subject.Counts.Values, depth+1)
	}
}

// Writes the sign that stands ahead of the node the walk stands on.
func emit_sign(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "emit_sign.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "emit_sign.tree")
	held_depth := depth_load(subject.Counts.Values)
	enter_leftmost(subject, tree)
	leftmost := subject.Nodes.Values[depth_load(subject.Counts.Values)]
	depth_store(subject.Counts.Values, held_depth)
	found := Count(ast.Syntax_Node_At(tree, leftmost).Token)
	// A note the author left behind the sign stands between the sign and the value it binds,
	// thus the read steps over every note it meets.
	behind := int(found) - 1
	for behind > 0 {
		if ast.Syntax_Token_At(tree, ast.Token_Index(behind)).Kind != token.KIND_COMMENT {
			break
		}
		behind = behind - 1
	}
	// The value one assignment binds stands behind the sign, thus a run position of nothing
	// at all names no sign the print writes.
	aver.Always(behind > 0, "A sign the print writes stands behind the value it binds.")
	one := ast.Syntax_Token_At(tree, ast.Token_Index(behind))
	text := token.Text(token.Source(subject.Source.Values), one)
	// A sign spells bytes of its own, thus the assignment it closes reads as the author
	// wrote it and never as one value standing against another.
	aver.Always(len(text) > 0, "A sign the print writes spells a byte of the source.")
	for index := range len(text) {
		emit_byte(subject, Symbol(text[index]))
	}
}

// STEP_UP names a step that counts the value it holds one higher.
const STEP_UP = 0

// STEP_DOWN names a step that counts the value it holds one lower.
const STEP_DOWN = 1

// Step_Kind names which way one step counts the value it holds.
type Step_Kind uint8

// Step_Kind_Invariants states both ways one step counts.
func Step_Kind_Invariants(value Step_Kind, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value), STEP_UP, STEP_DOWN).
		Ensure()
}

// Reads which way the step the walk stands on counts.
func step_kind(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (stepped Step_Kind) {
	defer func() { Step_Kind_Invariants(stepped, "step_kind.stepped") }()
	Printer_Handle_Invariants(subject, "step_kind.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "step_kind.tree")
	if node_kind(subject, tree) == ast.NODE_INCREMENT {
		return STEP_UP
	}
	return STEP_DOWN
}

// CLAUSE_KIND_MINIMUM is the first clause the print writes, which is the test one if states.
const CLAUSE_KIND_MINIMUM = uint8(ast.NODE_IF)

// CLAUSE_KIND_MAXIMUM is the last clause the print writes, which is the select.
const CLAUSE_KIND_MAXIMUM = uint8(ast.NODE_SELECT)

// CLAUSE_KIND_RANGE names the range clause, which stands inside a loop and opens no clause of
// its own.
const CLAUSE_KIND_RANGE = uint8(ast.NODE_RANGE)

// CLAUSE_KIND_CASE names one case of a switch, which the clause holds rather than states.
const CLAUSE_KIND_CASE = uint8(ast.NODE_CASE)

// CLAUSE_KIND_DEFAULT names the default of a switch, which the clause holds rather than states.
const CLAUSE_KIND_DEFAULT = uint8(ast.NODE_DEFAULT)

// Clause_Kind names which clause the print writes: a test, a loop, a switch of values or of
// types, or a select. The classes of the tree hold the range and the cases of a switch between
// them, and a clause states neither.
type Clause_Kind uint8

// Clause_Kind_Invariants states every clause one statement opens.
func Clause_Kind_Invariants(value Clause_Kind, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Holed_Uint8(
			uint8(value), CLAUSE_KIND_MINIMUM, CLAUSE_KIND_MAXIMUM,
			CLAUSE_KIND_RANGE, CLAUSE_KIND_CASE, CLAUSE_KIND_DEFAULT,
		).
		Ensure()
}

// Writes one clause statement: the word it opens with, the header it states, and the block behind
// it.
func print_clause(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "print_clause.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "print_clause.tree")
	// The tree holds no node for the word an else branch opens with, thus the print writes it
	// wherever a second branch stands behind the first.
	kind := node_kind(subject, tree)
	// A clause states a test, a loop, a switch of values or of types, or a select, thus the
	// walk stands on one of them where the print writes one.
	if bool(writes_canonical(subject.Flags.Values)) {
		Clause_Kind_Invariants(Clause_Kind(kind), "print_clause.kind")
	}
	// A body of one case alone closes up against the braces that hold it, the way a body of
	// one statement does; a body of more cases keeps the lines the author wrote.
	defer restore_tight(subject, hold_tight(subject, holds_one_case(subject, tree)))
	emit_token(subject, tree)
	open := descend_part(subject, tree)
	entered := open
	bodied, cased := Bodied(false), Cased(false)
	// A header the author broke across lines indents its continuations, and the body it opens
	// stands at the tabs the clause itself opened at.
	indent := indent_load(subject.Counts.Values)
	headed := Clause_Headed(false)
	closed := Branch_Closed(false)
	for bool(open) {
		conditional := Boolean(kind == ast.NODE_IF)
		if bool(opens_branch(subject, tree, bodied, conditional)) {
			emit_space(subject)
			emit_word(subject, WORD_ELSE)
		}
		// A branch that states a further clause closes its own line, thus the clause
		// around it writes no line of its own behind it.
		if bool(opens_else_clause(subject, tree, bodied)) {
			emit_space(subject)
			print_statement_form(subject, tree)
			closed = Branch_Closed(true)
			open = advance_part(subject, tree)
			continue
		}
		if bool(opens_case(subject, tree)) {
			column_indent_store(subject.Counts.Values, indent)
			print_switch_case(subject, tree, cased)
			cased = true
			open = advance_part(subject, tree)
			continue
		}
		if node_kind(subject, tree) == ast.NODE_BLOCK {
			column_indent_store(subject.Counts.Values, indent)
			open_body(subject, tree)
			print_block(subject, tree)
			bodied = true
			open = advance_part(subject, tree)
			continue
		}
		print_header(subject, tree, headed)
		headed = true
		open = advance_part(subject, tree)
	}
	// Every part of one clause hands back the tabs it opened at.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(indent_load(subject.Counts.Values) == indent,
			"A clause closes at the tabs it opened at.")
	}
	close_empty_clause(subject, bodied, cased)
	Branch_Closed_Invariants(closed, "print_clause.closed")
	close_clause(subject, tree, Boolean(cased), Boolean(closed))
	if bool(entered) {
		drain_trivia(subject, tree, advance(subject, tree))
		ascend(subject)
	}
}

// Writes the braces of one clause that states no case and no body of its own, the way every
// other empty body closes.
func close_empty_clause(subject Printer_Handle, bodied Bodied, cased Cased) {
	Printer_Handle_Invariants(subject, "close_empty_clause.subject")
	Bodied_Invariants(bodied, "close_empty_clause.bodied")
	Cased_Invariants(cased, "close_empty_clause.cased")
	if bool(bodied) {
		return
	}
	if bool(cased) {
		return
	}
	stood := written_load(subject.Counts.Values)
	emit_space(subject)
	emit_byte(subject, '{')
	emit_byte(subject, '}')
	// A clause of no body states the braces that hold none, thus the line it stands on closes
	// on the brace that closes them and holds the space and the braces it wrote.
	if !bool(flag_load(subject.Flags.Values, FLAG_FULL)) {
		aver.Always(subject.Form.Values[int(written_load(subject.Counts.Values))-1] == '}',
			"A clause of no body closes on the brace that holds none.")
		aver.Always(written_load(subject.Counts.Values) == stood+EMPTY_BODY_SIZE,
			"A clause of no body spells the space and the braces it states.")
	}
}

// Branch_Closed names a clause whose branch closed the line of its own, which is every branch
// that states a further clause behind the word else.
type Branch_Closed Boolean

// Branch_Closed_Invariants states both stands of the clause.
func Branch_Closed_Invariants(value Branch_Closed, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The branch closed the line of its own.").
		Ensure()
}

// Closes one clause: the brace its cases stand between, the line it wrote, and the comments and
// the empty line a branch that closed its own line left standing.
func close_clause(
	subject Printer_Handle, tree ast.Syntax_Tree_Handle, cased Boolean, closed Boolean,
) {
	Printer_Handle_Invariants(subject, "close_clause.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "close_clause.tree")
	Boolean_Invariants(cased, "close_clause.cased")
	Boolean_Invariants(closed, "close_clause.closed")
	if bool(cased) {
		emit_indent(subject)
		// The brace that closes a switch stands on a line of its own, at the tabs the
		// switch itself opened at.
		if bool(writes_canonical(subject.Flags.Values)) {
			aver.Always(int(take_line_width(subject.Counts.Values)) ==
				int(indent_load(subject.Counts.Values))*INDENT_WIDTH,
				"A switch closes its brace on a line of its own.")
		}
		emit_byte(subject, '}')
	}
	if !bool(closed) {
		close_line(subject, tree)
		return
	}
	close_debts(subject, tree)
}

// Writes what a closed line still owes: the comments the walk stepped over and the empty line
// behind them. A branch that closed its own line leaves them stand, thus the clause that holds
// it writes them rather than the statement behind it.
func close_debts(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "close_debts.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "close_debts.tree")
	if comment_count(subject) > 0 {
		flush_comments(subject, tree, CLOSER_POSITION_NONE)
		emit_line(subject)
	}
	if bool(flag_load(subject.Flags.Values, FLAG_BLANK)) {
		emit_line(subject)
		flag_store(subject.Flags.Values, FLAG_BLANK, false)
		key_width_store(subject.Counts.Values, 0)
	}
	// The close writes every note the print still owed and the empty line it owed behind
	// them, thus nothing the walk passed stands unwritten.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(comment_count(subject) == COMMENT_COUNT_MINIMUM,
			"A close of the debts one print holds carries no note behind it.")
		aver.Always(!bool(flag_load(subject.Flags.Values, FLAG_BLANK)),
			"A close of the debts one print holds owes no empty line.")
	}
}

// Reports whether one statement closes the line it stands on of its own. A clause writes a body
// and the comments behind that body, thus the statement that holds it closes no line for it.
func closes_own_line(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "closes_own_line.yes") }()
	Printer_Handle_Invariants(subject, "closes_own_line.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "closes_own_line.tree")
	written := written_load(subject.Counts.Values)
	defer func() {
		aver.Always(written_load(subject.Counts.Values) == written,
			"A read of the line one statement closes writes no byte of its own.")
	}()
	kind := node_kind(subject, tree)
	if kind == ast.NODE_IF {
		return true
	}
	if kind == ast.NODE_FOR {
		return true
	}
	if kind == ast.NODE_SWITCH {
		return true
	}
	if kind == ast.NODE_TYPE_SWITCH {
		return true
	}
	return Boolean(kind == ast.NODE_SELECT)
}

// CASE_NAMES names a case that states the values it stands for.
const CASE_NAMES = 0

// CASE_HOLDS names a default, which states the statements every other case leaves.
const CASE_HOLDS = 1

// Case_Kind names which of the two cases one switch or select states.
type Case_Kind uint8

// Case_Kind_Invariants states both cases a switch holds.
func Case_Kind_Invariants(value Case_Kind, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value), CASE_NAMES, CASE_HOLDS).
		Ensure()
}

// Reads which of the two cases the walk stands on.
func case_kind(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (cased Case_Kind) {
	defer func() { Case_Kind_Invariants(cased, "case_kind.cased") }()
	Printer_Handle_Invariants(subject, "case_kind.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "case_kind.tree")
	if node_kind(subject, tree) == ast.NODE_CASE {
		return CASE_NAMES
	}
	return CASE_HOLDS
}

// Writes one case of a switch or a select: the word, the values it names, and the statements it
// holds one tab deeper.
func print_case(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (closed Boolean) {
	defer func() { Boolean_Invariants(closed, "print_case.closed") }()
	Printer_Handle_Invariants(subject, "print_case.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "print_case.tree")
	// A case names the values it stands for, and a default names none, thus the walk stands
	// on one of the two where the print writes one.
	if bool(writes_canonical(subject.Flags.Values)) {
		Case_Kind_Invariants(case_kind(subject, tree), "print_case.cased")
	}
	stood := written_load(subject.Counts.Values)
	walked := depth_load(subject.Counts.Values)
	// The print writes one case of a switch here, thus the count of them reads what the
	// switches of the file state once the print closed. A default states no value of its own.
	if node_kind(subject, tree) == ast.NODE_DEFAULT {
		subject.Counts.Values[COUNT_DEFAULT] += 1
	}
	if node_kind(subject, tree) != ast.NODE_DEFAULT {
		subject.Counts.Values[COUNT_CASE] += 1
	}
	walk := Case_Walk{Body: false, Printed: false}
	// The tree names a default case by the colon it stands on and holds no node for the word
	// it opens with, thus the print writes that word of its own.
	if node_kind(subject, tree) == ast.NODE_DEFAULT {
		emit_word(subject, WORD_DEFAULT)
	}
	if node_kind(subject, tree) != ast.NODE_DEFAULT {
		emit_token(subject, tree)
	}
	// A case of few values reads on one line however the author wrote it, thus the print
	// closes up a head that fits and keeps the lines of a head that does not.
	flat := fits_case(subject, tree)
	open := descend(subject, tree)
	entered := open
	for bool(open) {
		walk = print_case_child(subject, tree, walk, flat)
		open = advance(subject, tree)
	}
	if !bool(walk.Printed) {
		emit_byte(subject, ':')
	}
	if bool(entered) {
		ascend(subject)
	}
	if bool(walk.Printed) {
		indent_shallow(subject, indent_load(subject.Counts.Values))
	}
	// A case states the values it stands for, or the word a default opens with, and the
	// colon that closes them, thus it writes bytes of its own for both.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(int(written_load(subject.Counts.Values)) > int(stood)+1,
			"A case writes the values it stands for and the colon behind them.")
		// Every child of the case hands back the walk it was handed.
		aver.Always(depth_load(subject.Counts.Values) == walked,
			"A case closes on the walk it opened.")
		// A case whose body the walk never opened closes on the colon the print wrote
		// for it, thus the statements behind it open on the line that colon closed.
		if !bool(walk.Printed) {
			held := int(written_load(subject.Counts.Values))
			aver.Always(subject.Form.Values[held-1] == ':',
				"A case closes on the colon behind the values it stands for.")
		}
	}
	return Boolean(walk.Printed)
}

// Body_Opened names a case whose body the walk already opened, which is every child behind the
// colon that closes the values the case stands for.
type Body_Opened Boolean

// Body_Opened_Invariants states both stands of the walk.
func Body_Opened_Invariants(value Body_Opened, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The walk stands inside the body of the case.").
		Ensure()
}

// Case_Walk states what the walk of one case holds: whether the body opened and whether the print
// wrote a statement of that body.
type Case_Walk struct {
	// Body names a walk that stands behind the colon of the case.
	Body Body_Opened
	// Printed names a case whose body already wrote a statement.
	Printed Statement_Written
}

// Case_Walk_Invariants composes both stands the walk of one case holds.
func Case_Walk_Invariants(value Case_Walk, namespace aver.Namespace) {
	Body_Opened_Invariants(value.Body, namespace)
	Statement_Written_Invariants(value.Printed, namespace)
}

// Writes one child of a case: a statement of its body, the trivia that stands inside it, or one
// value the case stands for. The colon closes the values, thus every child behind it states a
// statement of the body, a note the body opens with counted.
func print_case_child(
	subject Printer_Handle, tree ast.Syntax_Tree_Handle, walk Case_Walk, flat Boolean,
) (next Case_Walk) {
	defer func() { Case_Walk_Invariants(next, "print_case_child.next") }()
	Printer_Handle_Invariants(subject, "print_case_child.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "print_case_child.tree")
	Case_Walk_Invariants(walk, "print_case_child.walk")
	Boolean_Invariants(flat, "print_case_child.flat")
	if !bool(walk.Body) {
		walk.Body = Body_Opened(opens_body(subject, tree))
	}
	if bool(walk.Body) {
		print_case_body(subject, tree, Boolean(walk.Printed))
		return Case_Walk{Body: walk.Body, Printed: true}
	}
	if bool(trivia_kind(subject, tree)) {
		print_trivia(subject, tree)
		return walk
	}
	print_case_values(subject, tree, flat)
	return walk
}

// Writes one case of a switch on the line it opens. A case that holds statements closes its own
// line, thus the print owes a line only to a case that holds none.
func print_case_line(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "print_case_line.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "print_case_line.tree")
	held := indent_load(subject.Counts.Values)
	flag_store(subject.Flags.Values, FLAG_BROKEN, false)
	emit_indent(subject)
	closed := print_case(subject, tree)
	if !bool(closed) {
		close_line(subject, tree)
	}
	// A case body indents the statements it holds and hands back the tabs it was handed,
	// thus the case behind it opens at the tabs its switch stands at.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(indent_load(subject.Counts.Values) <= held,
			"A case closes at the tabs the switch opened it at.")
	}
	indent_store(subject.Counts.Values, held)
	flag_store(subject.Flags.Values, FLAG_BROKEN, false)
}

// Reports whether the node the walk stands on opens the body of a case rather than one more value
// the case names. A colon stands between the values and the body outside every bracket a value
// opens, thus a colon of the case ahead of the first token of one node states which side of it
// that node stands on.
func opens_body(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "opens_body.yes") }()
	Printer_Handle_Invariants(subject, "opens_body.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "opens_body.tree")
	held_depth := depth_load(subject.Counts.Values)
	enter_leftmost(subject, tree)
	leftmost := subject.Nodes.Values[depth_load(subject.Counts.Values)]
	depth_store(subject.Counts.Values, held_depth)
	found := int(ast.Syntax_Node_At(tree, leftmost).Token)
	opening := int(ast.Syntax_Node_At(tree, subject.Nodes.Values[held_depth-1]).Token)
	// The part stands inside the form around it, thus that form opens on a token no later
	// than the one the part opens on.
	aver.Always(opening <= found,
		"A part of one form stands behind the token that form opens on.")
	level := 0
	for place := opening; place < found; place = place + 1 {
		kind := ast.Syntax_Token_At(tree, ast.Token_Index(place)).Kind
		step := bracket_step(token.Scanned_Kind(kind))
		if step == BRACKET_OPENS {
			level = level + 1
		}
		if step == BRACKET_CLOSES {
			level = level - 1
		}
		if kind != token.KIND_COLON {
			continue
		}
		if level == 0 {
			return true
		}
	}
	return false
}

// Writes one expression, which every kind states its own way.
func print_expression(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "print_expression.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "print_expression.tree")
	// Every form inside one expression hands back the walk and the tabs it was handed, thus
	// the expression closes where it opened.
	stood := depth_load(subject.Counts.Values)
	written := written_load(subject.Counts.Values)
	tabbed := indent_load(subject.Counts.Values)
	nested := nest_load(subject.Counts.Values)
	keyed := key_width_load(subject.Counts.Values)
	columned := column_load(subject.Counts.Values)
	tight := flag_load(subject.Flags.Values, FLAG_TIGHT)
	counts_expression(subject, tree)
	switch node_kind(subject, tree) {
	case ast.NODE_BINARY:
		print_binary(subject, tree)
	case ast.NODE_UNARY:
		print_unary(subject, tree)
	case ast.NODE_CALL:
		print_wrapped(subject, tree, WRAP_CALL)
	case ast.NODE_INDEX, ast.NODE_GENERIC:
		print_wrapped(subject, tree, WRAP_INDEX)
	case ast.NODE_SLICE_EXPRESSION:
		print_slice_expression(subject, tree)
	case ast.NODE_SELECTOR:
		print_selector(subject, tree)
	case ast.NODE_ASSERTION:
		print_assertion(subject, tree)
	case ast.NODE_COMPOSITE:
		print_composite(subject, tree)
	case ast.NODE_KEY_VALUE:
		print_key_value(subject, tree)
	case ast.NODE_PARENTHESIS:
		print_group(subject, tree)
	case ast.NODE_FUNCTION_LITERAL:
		print_function_literal(subject, tree)
	case ast.NODE_RANGE:
		print_range(subject, tree)
	default:
		print_type_form(subject, tree)
	}
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(depth_load(subject.Counts.Values) == stood,
			"An expression closes on the walk it opened.")
		// An expression states a form the source spells, thus the print writes bytes.
		aver.Always(written_load(subject.Counts.Values) > written,
			"An expression writes bytes of its own.")
		// A form inside the expression indents its own lines and nests its own parts,
		// thus the expression hands back the tabs and the nest it was handed.
		aver.Always(indent_load(subject.Counts.Values) == tabbed,
			"An expression closes at the tabs it opened at.")
		aver.Always(nest_load(subject.Counts.Values) == nested,
			"An expression closes at the nest it opened at.")
		// A literal inside the expression opens a run of keys and a column of notes of
		// its own and holds its lines against its braces, thus each one closes with it.
		aver.Always(key_width_load(subject.Counts.Values) == keyed,
			"An expression closes the run of keys it opened.")
		aver.Always(column_load(subject.Counts.Values) == columned,
			"An expression closes the run of notes it opened.")
		aver.Always(flag_load(subject.Flags.Values, FLAG_TIGHT) == tight,
			"An expression closes the tightness it stood under.")
	}
}

// WRAPPED_CALL names a call, which wraps the values it hands over in parentheses.
const WRAPPED_CALL = 0

// WRAPPED_INDEX names an index, which wraps the value it reads at in brackets.
const WRAPPED_INDEX = 1

// WRAPPED_GENERIC names a generic, which wraps the types it names in brackets.
const WRAPPED_GENERIC = 2

// Wrapped_Kind names which form of the three the print writes brackets around.
type Wrapped_Kind uint8

// Wrapped_Kind_Invariants states every form that wraps its parts in brackets.
func Wrapped_Kind_Invariants(value Wrapped_Kind, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(uint8(value), WRAPPED_CALL, WRAPPED_INDEX, WRAPPED_GENERIC).
		Ensure()
}

// Reads which form of the three the walk stands on.
func wrapped_kind(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (held Wrapped_Kind) {
	defer func() { Wrapped_Kind_Invariants(held, "wrapped_kind.held") }()
	Printer_Handle_Invariants(subject, "wrapped_kind.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "wrapped_kind.tree")
	if node_kind(subject, tree) == ast.NODE_CALL {
		return WRAPPED_CALL
	}
	if node_kind(subject, tree) == ast.NODE_INDEX {
		return WRAPPED_INDEX
	}
	return WRAPPED_GENERIC
}

// WRAP_CALL names the parentheses a call states behind its head.
const WRAP_CALL Wrap = 0

// WRAP_INDEX names the brackets an index states behind its head.
const WRAP_INDEX Wrap = 1

// Wrap names the brackets one form states around what stands inside it.
type Wrap uint8

// Wrap_Invariants states every bracket a form wraps its parts in.
func Wrap_Invariants(value Wrap, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value), uint8(WRAP_CALL), uint8(WRAP_INDEX)).
		Ensure()
}

// Writes one form that wraps its parts in brackets: a call, an index, or a grouping.
func print_wrapped(subject Printer_Handle, tree ast.Syntax_Tree_Handle, wrap Wrap) {
	Printer_Handle_Invariants(subject, "print_wrapped.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "print_wrapped.tree")
	Wrap_Invariants(wrap, "print_wrapped.wrap")
	// A wrapped form states a call, an index, or the types one generic names.
	if bool(writes_canonical(subject.Flags.Values)) {
		Wrapped_Kind_Invariants(wrapped_kind(subject, tree), "print_wrapped.wrapped")
	}
	counts_type(subject, tree)
	walked := depth_load(subject.Counts.Values)
	// A call and an index state parts of an expression and never a body, thus an empty line
	// the author left against their brackets stands where the author put it.
	defer restore_tight(subject, hold_tight(subject, false))
	// A form past the widest line states one part to a line, thus the print measures it
	// first, and a run of one operation outside these brackets states nothing of the runs
	// the parts inside them hold.
	over := runs_over(subject, tree)
	defer restore_split(subject, hold_split(subject, false))
	tail := closes_line(subject, tree)
	indent := indent_load(subject.Counts.Values)
	marked := flag_load(subject.Flags.Values, FLAG_BROKEN)
	held := nest_load(subject.Counts.Values)
	inner := part_inner_nest(subject, tree, wrap, held)
	// The notes of the parts inside these brackets stand in a column of their own.
	defer column_store(subject.Counts.Values, column_load(subject.Counts.Values))
	open := descend_part(subject, tree)
	entered := open
	place, broken, opened := Part_Place(PART_HEAD), Boolean(false), Boolean(false)
	// A call the author broke behind its parenthesis closes on a line of its own, thus the
	// print reads that break at the first argument and holds it past the arguments behind it.
	heads := Boolean(false)
	for bool(open) {
		// The points of a spread stand against the argument ahead of them, thus a
		// spread behind one takes neither a comma nor a break of its own.
		spread := stands_spread(subject, tree, place)
		if place == PART_FIRST {
			result := open_wrapped(subject, tree, wrap, over, tail)
			indent, marked = result.Indent, Boolean(result.Marked)
			opened, broken = Boolean(result.Opened), Boolean(result.Broken)
			heads = Boolean(result.Heads)
		}
		if place == PART_BEHIND {
			column_indent_store(subject.Counts.Values, indent)
			result := open_argument(subject, tree, opened, broken, marked, over)
			opened, broken = result.Stand, Boolean(result.Spread)
		}
		if bool(spread) {
			print_spread(subject, tree)
			place = Part_Place(next_part_place(place))
			open = advance_part(subject, tree)
			continue
		}
		nest_store(subject.Counts.Values, part_nest(place, wrap, inner))
		print_part(subject, tree, opened, place)
		nest_store(subject.Counts.Values, held)
		place = Part_Place(next_part_place(place))
		open = advance_part(subject, tree)
	}
	if bool(broken) {
		column_indent_store(subject.Counts.Values, indent)
		close_broken(subject, tree, tail || heads, form_closer(subject, tree))
	}
	close_wrapped(subject, tree, wrap, entered, Boolean(place != PART_BEHIND))
	// Every part the brackets hold hands back the walk it was handed, thus the form closes
	// on the node it opened on.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(depth_load(subject.Counts.Values) == walked,
			"A wrapped form closes on the walk it opened.")
	}
}

// Names the nest one part of a wrapped form stands at. The value an index reads stands as plainly
// as a statement does, thus the value that index names opens at the nest a statement opens at.
func part_nest(place Part_Place, wrap Wrap, inner Nest) (nest Nest) {
	defer func() { Nest_Invariants(nest, "part_nest.nest") }()
	Part_Place_Invariants(place, "part_nest.place")
	Wrap_Invariants(wrap, "part_nest.wrap")
	Nest_Invariants(inner, "part_nest.inner")
	if place != PART_HEAD {
		return inner
	}
	if wrap != WRAP_INDEX {
		return inner
	}
	return NEST_MINIMUM
}

// MARKED_BREAK names a form the print already broke ahead of the bracket one form opens at.
type Marked_Break Boolean

// Marked_Break_Invariants states both stands of the break a form opens at.
func Marked_Break_Invariants(value Marked_Break, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The form stood broken where its bracket opened.").
		Ensure()
}

// Opened_Break names a bracket the author left a break behind.
type Opened_Break Boolean

// Opened_Break_Invariants states both stands of the bracket.
func Opened_Break_Invariants(value Opened_Break, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The bracket opened a line of its own.").
		Ensure()
}

// Heads_Line names a call whose head stands on the line its bracket closes.
type Heads_Line Boolean

// Heads_Line_Invariants states both stands of the head.
func Heads_Line_Invariants(value Heads_Line, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The head of the call keeps the line its bracket closes.").
		Ensure()
}

// Broken_Form names a form the print writes one part to a line.
type Broken_Form Boolean

// Broken_Form_Invariants states both stands of the form.
func Broken_Form_Invariants(value Broken_Form, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The form stands one part to a line.").
		Ensure()
}

// Wrapped_Open states how the brackets of one wrapped form opened: the indent its parts answer
// to, the break that stood ahead of it, the break the author left behind the bracket, whether
// the head keeps the line the bracket closes, and whether the parts stand one to a line.
type Wrapped_Open struct {
	// Indent holds the tabs the bracket opened at.
	Indent Indent
	// Marked reports whether the print already stood broken where the bracket opened.
	Marked Marked_Break
	// Opened reports whether the author left a break behind the bracket.
	Opened Opened_Break
	// Heads reports whether the head of a call keeps the line its bracket closes.
	Heads Heads_Line
	// Broken reports whether the parts stand one to a line.
	Broken Broken_Form
}

// Wrapped_Open_Invariants composes one opening of a wrapped form.
func Wrapped_Open_Invariants(value Wrapped_Open, namespace aver.Namespace) {
	Indent_Invariants(value.Indent, namespace)
	Marked_Break_Invariants(value.Marked, namespace)
	Opened_Break_Invariants(value.Opened, namespace)
	Heads_Line_Invariants(value.Heads, namespace)
	Broken_Form_Invariants(value.Broken, namespace)
}

// Opens the brackets of one wrapped form at its first part: the bracket, the break the author
// left behind it, and the indent the parts answer to. The head of the form may have indented what
// stands behind it, thus the parts answer to the indent the bracket opened at, and a bracket the
// author left on a line of its own closes a broken form however the parts before it stand.
func open_wrapped(
	subject Printer_Handle, tree ast.Syntax_Tree_Handle, wrap Wrap, over Boolean, tail Boolean,
) (result Wrapped_Open) {
	defer func() { Wrapped_Open_Invariants(result, "open_wrapped.result") }()
	Printer_Handle_Invariants(subject, "open_wrapped.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "open_wrapped.tree")
	Wrap_Invariants(wrap, "open_wrapped.wrap")
	Boolean_Invariants(over, "open_wrapped.over")
	Boolean_Invariants(tail, "open_wrapped.tail")
	emit_open(subject, wrap)
	indent := indent_load(subject.Counts.Values)
	marked := flag_load(subject.Flags.Values, FLAG_BROKEN)
	opened := breaks_before(subject, tree) || over
	heads := opened && Boolean(wrap == WRAP_CALL)
	print_first_break(subject, tree, opened)
	return Wrapped_Open{
		Indent: indent, Marked: Marked_Break(marked), Opened: Opened_Break(opened),
		Heads: Heads_Line(heads), Broken: Broken_Form(opened || tail),
	}
}

// Closes one wrapped form: the parts the walk still stands ahead of, the bracket a form of one
// part alone still owes, and the bracket every form closes with.
func close_wrapped(
	subject Printer_Handle,
	tree ast.Syntax_Tree_Handle,
	wrap Wrap,
	entered Boolean,
	alone Boolean,
) {
	Printer_Handle_Invariants(subject, "close_wrapped.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "close_wrapped.tree")
	Wrap_Invariants(wrap, "close_wrapped.wrap")
	Boolean_Invariants(entered, "close_wrapped.entered")
	Boolean_Invariants(alone, "close_wrapped.alone")
	if bool(entered) {
		finish_parts(subject, tree)
		ascend(subject)
	}
	if bool(alone) {
		emit_open(subject, wrap)
	}
	stood := written_load(subject.Counts.Values)
	emit_close(subject, wrap)
	// The bracket that closes one wrapped form spells one byte, thus the form holds it and
	// the print stands behind it.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(written_load(subject.Counts.Values) == stood+1,
			"A wrapped form closes on one byte of its own.")
	}
	closes_bracket_byte(subject, wrap)
}

// Opens the line the first part of a broken form stands on and indents its parts one level.
func print_first_break(subject Printer_Handle, tree ast.Syntax_Tree_Handle, broken Boolean) {
	Printer_Handle_Invariants(subject, "print_first_break.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "print_first_break.tree")
	Boolean_Invariants(broken, "print_first_break.broken")
	if !bool(broken) {
		return
	}
	stood := written_load(subject.Counts.Values)
	indent_deepen(subject, indent_load(subject.Counts.Values))
	close_line(subject, tree)
	emit_indent(subject)
	// A form the print breaks at its bracket opens a line for its first part, thus the
	// break writes the feed that line takes.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(written_load(subject.Counts.Values) > stood,
			"A break ahead of the first part writes the line it opens.")
	}
}

// Writes what stands between two parts of a form: a comma and a space, or a comma and the line
// the author broke.
func print_element_break(subject Printer_Handle, tree ast.Syntax_Tree_Handle, broken Boolean) {
	Printer_Handle_Invariants(subject, "print_element_break.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "print_element_break.tree")
	Boolean_Invariants(broken, "print_element_break.broken")
	stood := written_load(subject.Counts.Values)
	if !bool(broken) {
		emit_comma(subject)
		emit_space(subject)
		// A part standing behind another takes the comma that parts them and the space
		// behind it, thus the print writes bytes of its own between the two.
		if bool(writes_canonical(subject.Flags.Values)) {
			aver.Always(written_load(subject.Counts.Values) > stood,
				"A part behind another stands behind the comma that parts them.")
		}
		return
	}
	emit_break(subject, tree)
}

// Closes a broken form. A bracket the author left on the line of the final part stays there and
// takes no comma; a bracket the author gave a line of its own keeps it, and the part before it
// takes the comma that a line feed there demands.
func close_broken(
	subject Printer_Handle, tree ast.Syntax_Tree_Handle, tail Boolean, closer Closer_Position,
) {
	Printer_Handle_Invariants(subject, "close_broken.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "close_broken.tree")
	Boolean_Invariants(tail, "close_broken.tail")
	Closer_Position_Invariants(closer, "close_broken.closer")
	// A part the author broke across lines of its own indented what stood behind it, thus the
	// bracket that closes the form stands where the form opened and not where a part left it.
	if !bool(tail) {
		indent_store(subject.Counts.Values, column_indent_load(subject.Counts.Values))
		return
	}
	// A part that wrote nothing a reader reads on the line it opened takes no comma, thus
	// a form whose final part states nothing closes its bracket on that line.
	if !bool(stands_after_space(subject)) {
		if !bool(flag_load(subject.Flags.Values, FLAG_LINE)) {
			emit_comma(subject)
		}
	}
	// An empty line the author left behind the final part belongs to the statement that
	// holds the form and not to the line the bracket closes, thus the print still owes it.
	// A note the author left past the bracket stands on the line that bracket closes, and a
	// note standing on a line of its own stands at the tabs the parts stand at.
	flush_comments(subject, tree, closer)
	indent_store(subject.Counts.Values, column_indent_load(subject.Counts.Values))
	// A part that stated nothing opened a line holding tabs alone, thus the print takes
	// those tabs back and closes the bracket on the line that part opened.
	if bool(writes_canonical(subject.Flags.Values)) {
		drop_trailing_space(subject)
	}
	if !bool(opens_form_line(subject)) {
		emit_line(subject)
	}
	emit_indent(subject)
	// The bracket that closes a broken form stands on a line of its own, at the tabs the
	// form opened at and behind nothing else.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(int(take_line_width(subject.Counts.Values)) ==
			int(indent_load(subject.Counts.Values))*INDENT_WIDTH,
			"A broken form closes its bracket on a line of its own.")
	}
}

// Writes the comments the line the print stands on carries behind it. A comment the author wrote
// on a line of its own keeps that line, thus a note about the statement behind it never lands at
// the end of the statement ahead of it. The walk stands on each comment while it writes it.
func flush_comments(subject Printer_Handle, tree ast.Syntax_Tree_Handle, closer Closer_Position) {
	Printer_Handle_Invariants(subject, "flush_comments.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "flush_comments.tree")
	Closer_Position_Invariants(closer, "flush_comments.closer")
	depth := depth_load(subject.Counts.Values)
	count := int(comment_count(subject))
	stood := written_load(subject.Counts.Values)
	kept := 0
	lined := Boolean(false)
	behind := ast.Token_Index(0)
	for slot := range count {
		subject.Nodes.Values[depth+1] = subject.Comments.Values[slot]
		depth_store(subject.Counts.Values, depth+1)
		// The storage holds the notes one line carries and nothing else.
		aver.Always(node_kind(subject, tree) == ast.NODE_COMMENT,
			"A note the line carries names a note of the tree.")
		// The notes one line carries stand in the order the author wrote them, thus the
		// storage holds each one behind the one it took before it.
		position := ast.Syntax_Node_At(tree,
			subject.Nodes.Values[depth+1]).Token
		aver.Always(position >= behind,
			"A note the line carries stands behind the note it took ahead of it.")
		behind = position
		if bool(stands_past(subject, tree, closer)) {
			subject.Comments.Values[kept] = subject.Comments.Values[slot]
			kept = kept + 1
			depth_store(subject.Counts.Values, depth)
			continue
		}
		if bool(stands_alone(subject, tree)) {
			if !bool(flag_load(subject.Flags.Values, FLAG_LINE)) {
				emit_line(subject)
			}
		}
		if bool(opens_run(subject, tree)) {
			emit_line(subject)
		}
		// A note the author closed at the line swallows the bytes behind it, thus a note
		// standing behind one on a line opens a line of its own.
		if bool(lined) {
			if !bool(flag_load(subject.Flags.Values, FLAG_LINE)) {
				emit_line(subject)
			}
		}
		lined = closes_at_line(subject, tree)
		emit_comment(subject, tree)
		depth_store(subject.Counts.Values, depth)
	}
	for slot := kept; slot < count; slot++ {
		subject.Comments.Values[slot] = ast.INDEX_ABSENT
	}
	// The notes of one line stand on that line alone, thus the storage that held them takes
	// the notes of the line the print opens next and the notes it held back.
	aver.Always(int(comment_count(subject)) == kept,
		"A flush of the notes one line carries holds back what stands past the bracket.")
	// A note spells bytes of the source, thus a flush that wrote one lands them in the form.
	// A measure of one statement writes no note at all, thus it lands none.
	if kept < count {
		if !bool(flag_load(subject.Flags.Values, FLAG_NOTELESS)) {
			if bool(writes_canonical(subject.Flags.Values)) {
				aver.Always(written_load(subject.Counts.Values) > stood,
					"A flush that writes a note lands bytes of it.")
			}
		}
	}
}

// COMMENT_COUNT_MINIMUM is the count of a line that carries no comment.
const COMMENT_COUNT_MINIMUM = 0

// Comment_Count is how many comments the line the print stands on carries.
type Comment_Count int

// Comment_Count_Invariants states every count of comments the storage holds.
func Comment_Count_Invariants(value Comment_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), COMMENT_COUNT_MINIMUM, COMMENT_COUNT_MAXIMUM).
		Ensure()
}

// Reports whether the comment at one run position opens the line it stands on.
func stands_alone(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "stands_alone.yes") }()
	Printer_Handle_Invariants(subject, "stands_alone.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "stands_alone.tree")
	written := written_load(subject.Counts.Values)
	defer func() {
		aver.Always(written_load(subject.Counts.Values) == written,
			"A read of the line one note opens writes no byte of its own.")
	}()
	cursor := ast.Syntax_Node_At(tree, subject.Nodes.Values[depth_load(subject.Counts.Values)])
	position := cursor.Token
	if position == 0 {
		return false
	}
	// A semicolon the scanner states for the author spends no byte of the source and stands
	// where the token behind it opens, thus the line opens at the last token that spent one.
	place := int(position) - 1
	behind := ast.Syntax_Token_At(tree, ast.Token_Index(place))
	for place > POSITION_MINIMUM {
		if behind.Size > 0 {
			break
		}
		place = place - 1
		behind = ast.Syntax_Token_At(tree, ast.Token_Index(place))
	}
	ahead := ast.Syntax_Token_At(tree, ast.Token_Index(position))
	// No two tokens of the run share a byte of the source, thus the one the node names opens
	// behind the end of the one standing ahead of it.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(int(ahead.Offset) >= int(behind.Offset)+int(behind.Size),
			"A token of the run opens behind the end of the token ahead of it.")
	}
	// A line feed between the two bytes states a break the author wrote.
	opened := Boolean(false)
	from_offset, to_offset := int(int(behind.Offset)+int(behind.Size)), int(ahead.Offset)
	for place_offset := from_offset; place_offset < to_offset; place_offset++ {
		if subject.Source.Values[place_offset] == '\n' {
			opened = true
		}
	}
	if !bool(opened) {
		return false
	}
	// A note that opens a line and carries the source of that line behind it stands with
	// that source, thus it takes no line of its own.
	next := ast.Syntax_Token_At(tree, ast.Token_Index(position+1))
	from_offset, to_offset = int(int(ahead.Offset)+int(ahead.Size)), int(next.Offset)
	for place_offset := from_offset; place_offset < to_offset; place_offset++ {
		if subject.Source.Values[place_offset] == '\n' {
			return true
		}
	}
	return false
}

// Writes the bracket that opens one wrapped form.
func emit_open(subject Printer_Handle, wrap Wrap) {
	Printer_Handle_Invariants(subject, "emit_open.subject")
	Wrap_Invariants(wrap, "emit_open.wrap")
	// A bracket that wraps the parts of one form stands against the head it follows, thus no
	// space holds the two apart.
	written := int(written_load(subject.Counts.Values))
	if written > 0 {
		if bool(writes_canonical(subject.Flags.Values)) {
			aver.Always(subject.Form.Values[written-1] != byte(SYMBOL_SPACE),
				"A bracket opens against the head it follows.")
		}
	}
	stood := written_load(subject.Counts.Values)
	if wrap == WRAP_INDEX {
		emit_byte(subject, '[')
	}
	if wrap != WRAP_INDEX {
		emit_byte(subject, '(')
	}
	opens_bracket_byte(subject, wrap)
	// A bracket spells one byte, thus the form holds one byte more than it held.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(written_load(subject.Counts.Values) == stood+1,
			"A bracket that opens one form spells one byte of its own.")
	}
}

// Writes the bracket that closes one wrapped form.
func emit_close(subject Printer_Handle, wrap Wrap) {
	Printer_Handle_Invariants(subject, "emit_close.subject")
	Wrap_Invariants(wrap, "emit_close.wrap")
	// A bracket closes what stands ahead of it, thus no space holds the two apart. A bracket
	// that opens a line of its own stands behind the bytes that line opens at and behind no
	// space of its own.
	written := int(written_load(subject.Counts.Values))
	if written > 0 {
		if !bool(opens_line(subject)) {
			if bool(writes_canonical(subject.Flags.Values)) {
				aver.Always(subject.Form.Values[written-1] != byte(SYMBOL_SPACE),
					"A bracket closes against the byte ahead of it.")
			}
		}
	}
	stood := written_load(subject.Counts.Values)
	if wrap == WRAP_INDEX {
		emit_byte(subject, ']')
	}
	if wrap != WRAP_INDEX {
		emit_byte(subject, ')')
	}
	// A bracket spells one byte, thus the form holds one byte more than it held.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(written_load(subject.Counts.Values) == stood+1,
			"A bracket that closes one form spells one byte of its own.")
	}
}

// Writes one operation over two values. A sign that binds loosely stands between spaces and one
// that binds tightly stands against its values, thus the reader sees which value binds to which
// sign without counting precedence.
func print_binary(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "print_binary.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "print_binary.tree")
	// An operation binds two values with one sign, thus the walk stands on one where the
	// print writes one.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(node_kind(subject, tree) == ast.NODE_BINARY,
			"An operation the print writes stands on an operation of the tree.")
	}
	subject.Counts.Values[COUNT_FORM] += 1
	subject.Counts.Values[COUNT_OPERATION] += 1
	spaced := binary_spaced(subject, tree)
	level := precedence(subject, tree)
	// A run of one operation breaks at every sign it states or at none of them, thus the
	// outermost operation of the run reads the width and the operations below it hold that
	// answer. An operation that binds tighter states a run of its own and reads its own width,
	// because a value the run holds reads as one value however the run around it stands.
	held_level := split_load(subject)
	split := binary_split(subject, tree, level)
	split_store(subject, level)
	defer restore_split(subject, hold_split(subject, split))
	held := nest_load(subject.Counts.Values)
	// An operation binds one value ahead of its sign and one behind it, thus the walk steps
	// down to the value the sign reads first.
	opened := descend_part(subject, tree)
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(bool(opened),
			"An operation the print writes binds a value ahead of its sign.")
	}
	if !bool(opened) {
		split_store(subject, held_level)
		return
	}
	// A left value that states the same sign holds the same nest, thus a run of one sign
	// spaces the same way however long it runs.
	nest_store(subject.Counts.Values, Nest(nest_inner(subject, held)))
	if bool(left_holds(subject, tree, level)) {
		nest_store(subject.Counts.Values, held)
	}
	print_expression(subject, tree)
	nest_store(subject.Counts.Values, held)
	if spaced {
		emit_space(subject)
	}
	// A sign that spaces neither side stands against the value ahead of it, thus the value
	// closes on a byte of its own and leaves no space behind.
	if !spaced {
		if bool(writes_canonical(subject.Flags.Values)) {
			aver.Always(!bool(stands_after_space(subject)),
				"A sign that takes no space stands against its value.")
		}
	}
	// A value the operation holds states a sign of its own while it prints, thus the slot
	// states this sign again before the print writes it.
	emit_parent_token(subject, tree)
	behind := advance_part(subject, tree)
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(bool(behind),
			"An operation the print writes binds a value behind its sign.")
	}
	if bool(behind) {
		print_right_value(subject, tree, split, Boolean(spaced), held)
	}
	finish_parts(subject, tree)
	ascend(subject)
	split_store(subject, held_level)
}

// Writes the value one operation binds behind its sign. An operation the author broke behind its
// sign keeps that break, and its right value stands one tab deeper than the operation that holds
// it. The tab closes with the operation, thus an operation inside another one steps deeper again
// while a run of one sign holds one column.
func print_right_value(
	subject Printer_Handle, tree ast.Syntax_Tree_Handle, split Boolean, spaced Boolean,
	held Nest,
) {
	Printer_Handle_Invariants(subject, "print_right_value.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "print_right_value.tree")
	Boolean_Invariants(split, "print_right_value.split")
	Boolean_Invariants(spaced, "print_right_value.spaced")
	Nest_Invariants(held, "print_right_value.held")
	indent := indent_load(subject.Counts.Values)
	here := Boolean(bool(breaks_before(subject, tree)) || bool(split))
	if bool(here) {
		indent_deepen(subject, indent)
		close_line(subject, tree)
		emit_indent(subject)
		// A run of one sign breaks at every sign it states, thus the value behind the
		// sign opens a line holding the tabs of that run and nothing else.
		if bool(writes_canonical(subject.Flags.Values)) {
			aver.Always(int(take_line_width(subject.Counts.Values)) ==
				int(indent_load(subject.Counts.Values))*INDENT_WIDTH,
				"A value behind a broken sign opens a line of its own.")
		}
	}
	if bool(spaced) {
		if !bool(here) {
			emit_space(subject)
		}
	}
	nest_store(subject.Counts.Values, Nest(nest_inner(subject, held)))
	print_expression(subject, tree)
	nest_store(subject.Counts.Values, held)
	indent_store(subject.Counts.Values, indent)
}

// Reads the binding of the run of operations the print breaks at every sign.
func split_load(subject Printer_Handle) (level Level) {
	defer func() { Level_Invariants(level, "split_load.level") }()
	Printer_Handle_Invariants(subject, "split_load.subject")
	return Level(subject.Counts.Values[COUNT_SPLIT])
}

// Stands the binding of the run of operations the print breaks at every sign.
func split_store(subject Printer_Handle, level Level) {
	Printer_Handle_Invariants(subject, "split_store.subject")
	Level_Invariants(level, "split_store.level")
	subject.Counts.Values[COUNT_SPLIT] = Count(level)
}

// Writes the sign of the operation the walk stands inside, which is the token of the node one
// depth above the walk. An operation writes its sign between its values, thus the walk already
// stands on the left value when the sign is written.
func emit_parent_token(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "emit_parent_token.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "emit_parent_token.tree")
	parent := subject.Nodes.Values[depth_load(subject.Counts.Values)-1]
	one := ast.Syntax_Token_At(tree, ast.Syntax_Node_At(tree, parent).Token)
	text := token.Text(token.Source(subject.Source.Values), one)
	// The token of the form around the walk spells bytes of the source, thus the print
	// writes the sign that form states rather than nothing at all.
	aver.Always(len(text) > 0, "A token the form around the walk states spells a byte.")
	for index := range len(text) {
		emit_byte(subject, Symbol(text[index]))
	}
}

// LEVEL_MINIMUM names a token that binds no values, which is a token that is no operation sign.
const LEVEL_MINIMUM = 0

// LEVEL_COMPARISON names the binding of the four signs that compare two values.
const LEVEL_COMPARISON = 1

// LEVEL_ADDITION names the binding of addition, subtraction, and the two or signs.
const LEVEL_ADDITION = 2

// LEVEL_SIGN_MAXIMUM names the tightest binding a sign of an operation states, which is the
// binding of the products, the shifts, and the two and signs.
const LEVEL_SIGN_MAXIMUM = 3

// CUT_MINIMUM is the loosest binding a form closes its signs up at.
const CUT_MINIMUM = 2

// CUT_MIDDLE closes up the signs that bind tightest and spaces every looser one, which is what a
// form that mixes the two levels the canonical form spaces by states.
const CUT_MIDDLE = 3

// CUT_MAXIMUM stands above every sign, which is the cut a form that spaces every sign it holds
// states, and no sign binds that tightly.
const CUT_MAXIMUM = 4

// CLASH_NONE names an operation no reading of which runs two signs together.
const CLASH_NONE = 0

// CLASH_LOOSE names a clash the looser of the two spaced levels settles.
const CLASH_LOOSE = 2

// CLASH_TIGHT names a clash only the tighter of the two spaced levels settles.
const CLASH_TIGHT = 3

// Clash names the level a sign must space at to stand apart from the sign of the value behind it.
type Clash uint8

// Clash_Invariants states every clash two signs standing together state.
func Clash_Invariants(value Clash, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(uint8(value), CLASH_NONE, CLASH_LOOSE, CLASH_TIGHT).
		Ensure()
}

// Cut names the binding at and above which one form closes up the signs it holds.
type Cut uint8

// Cut_Invariants states every cut one form closes its signs up at.
func Cut_Invariants(value Cut, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(uint8(value), CUT_MINIMUM, CUT_MIDDLE, CUT_MAXIMUM).
		Ensure()
}

// Level names how tightly one sign binds the values it stands between.
type Level uint8

// Level_Invariants states the binding of a token that binds nothing and the three bindings the
// signs of this dialect hold.
func Level_Invariants(value Level, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_4_Uint8(uint8(value), LEVEL_MINIMUM, LEVEL_COMPARISON, LEVEL_ADDITION,
			LEVEL_SIGN_MAXIMUM).
		Ensure()
}

// Reports whether the left value of one operation states the sign that operation states. Such a
// value stands at the level the operation stands at, which keeps one run of a sign spacing as one
// form.
func left_holds(subject Printer_Handle, tree ast.Syntax_Tree_Handle, level Level) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "left_holds.yes") }()
	Printer_Handle_Invariants(subject, "left_holds.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "left_holds.tree")
	Level_Invariants(level, "left_holds.level")
	stood := written_load(subject.Counts.Values)
	defer func() {
		aver.Always(written_load(subject.Counts.Values) == stood,
			"A read of the sign one value on the left states writes no byte.")
	}()
	if node_kind(subject, tree) != ast.NODE_BINARY {
		return false
	}
	return Boolean(precedence(subject, tree) == level)
}

// Names how tightly the sign of the operation the walk stands on binds its values. The minimum
// names a node whose token is no operation sign.
func precedence(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (level Level) {
	defer func() { Level_Invariants(level, "precedence.level") }()
	Printer_Handle_Invariants(subject, "precedence.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "precedence.tree")
	node := ast.Syntax_Node_At(tree, subject.Nodes.Values[depth_load(subject.Counts.Values)])
	switch ast.Syntax_Token_At(tree, node.Token).Kind {
	case token.KIND_STAR, token.KIND_SLASH, token.KIND_PERCENT, token.KIND_SHIFT_LEFT,
		token.KIND_SHIFT_RIGHT, token.KIND_AND, token.KIND_AND_NOT:
		return LEVEL_SIGN_MAXIMUM
	case token.KIND_PLUS, token.KIND_MINUS, token.KIND_OR, token.KIND_EXCLUSIVE_OR:
		return LEVEL_ADDITION
	case token.KIND_EQUAL, token.KIND_NOT_EQUAL, token.KIND_LESS, token.KIND_GREATER:
		return LEVEL_COMPARISON
	}
	return LEVEL_MINIMUM
}

// CLASH_SIGN_OTHER names an operation whose sign runs together with no unary sign behind it.
const CLASH_SIGN_OTHER = 0

// CLASH_SIGN_SLASH names a division, whose sign runs together with a dereference behind it.
const CLASH_SIGN_SLASH = 1

// CLASH_SIGN_AND names a bitwise and, whose sign runs together with an address or a complement.
const CLASH_SIGN_AND = 2

// CLASH_SIGN_PLUS names an addition, whose sign runs together with a unary plus.
const CLASH_SIGN_PLUS = 3

// CLASH_SIGN_MINUS names a subtraction, whose sign runs together with a negation.
const CLASH_SIGN_MINUS = 4

// Clash_Sign names the operation sign a unary sign behind it may run together with.
type Clash_Sign uint8

// Clash_Sign_Invariants states every operation sign a unary sign may run together with.
func Clash_Sign_Invariants(value Clash_Sign, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), CLASH_SIGN_OTHER, CLASH_SIGN_MINUS).
		Ensure()
}

// Names the operation sign of the node the walk stands on that a unary sign behind it may run
// together with.
func clash_sign(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (sign Clash_Sign) {
	defer func() { Clash_Sign_Invariants(sign, "clash_sign.sign") }()
	Printer_Handle_Invariants(subject, "clash_sign.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "clash_sign.tree")
	node := ast.Syntax_Node_At(tree, subject.Nodes.Values[depth_load(subject.Counts.Values)])
	switch ast.Syntax_Token_At(tree, node.Token).Kind {
	case token.KIND_SLASH:
		return CLASH_SIGN_SLASH
	case token.KIND_AND:
		return CLASH_SIGN_AND
	case token.KIND_PLUS:
		return CLASH_SIGN_PLUS
	case token.KIND_MINUS:
		return CLASH_SIGN_MINUS
	}
	return CLASH_SIGN_OTHER
}

// Reports whether the sign of the operation the walk stands on takes spaces around it. The
// canonical form spaces every sign of an operation a statement states on its own, and closes up
// the tighter signs of an operation that stands inside another form or mixes two levels.
// Reports whether one operation breaks at every sign it states: a run of the binding the print
// already broke at, or one whose line runs past the widest line.
func binary_split(
	subject Printer_Handle, tree ast.Syntax_Tree_Handle, level Level,
) (splits Boolean) {
	defer func() { Boolean_Invariants(splits, "binary_split.splits") }()
	Printer_Handle_Invariants(subject, "binary_split.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "binary_split.tree")
	Level_Invariants(level, "binary_split.level")
	if bool(flag_load(subject.Flags.Values, FLAG_SPLIT)) {
		if split_load(subject) == level {
			return true
		}
	}
	return runs_over(subject, tree)
}

func binary_spaced(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "binary_spaced.yes") }()
	Printer_Handle_Invariants(subject, "binary_spaced.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "binary_spaced.tree")
	stood := written_load(subject.Counts.Values)
	defer func() {
		aver.Always(written_load(subject.Counts.Values) == stood,
			"A read of the spacing one operation states writes no byte of its own.")
	}()
	levels := binary_levels(subject, tree)
	adds, multiplies, clash := levels.Adds, levels.Multiplies, levels.Clash
	// A sign that would read as another sign against the value behind it takes spaces
	// however deep it stands, thus the clash sets the cut on its own.
	cut := Cut(CUT_MINIMUM)
	if clash > 0 {
		cut = Cut(clash) + 1
	}
	if clash == 0 {
		cut = plain_cut(subject, adds, Boolean(multiplies))
	}
	return Boolean(Cut(precedence(subject, tree)) < cut)
}

// Names the level at and above which the signs of one operation close up. An operation that mixes
// the two levels the canonical form spaces by states the tighter of them, and one that holds a
// single level spaces every sign a statement states on its own.
func plain_cut(subject Printer_Handle, adds Boolean, multiplies Boolean) (cut Cut) {
	defer func() { Cut_Invariants(cut, "plain_cut.cut") }()
	Printer_Handle_Invariants(subject, "plain_cut.subject")
	Boolean_Invariants(adds, "plain_cut.adds")
	Boolean_Invariants(multiplies, "plain_cut.multiplies")
	plain := nest_load(subject.Counts.Values) == 1
	if bool(adds) {
		if bool(multiplies) {
			if plain {
				return CUT_MIDDLE
			}
			return CUT_MINIMUM
		}
	}
	if plain {
		return CUT_MAXIMUM
	}
	return CUT_MINIMUM
}

// Names the level an operation sign must space at to stand apart from the unary sign of the value
// the walk stands on. Zero names an operation no reading of which runs the two signs together.
func clash_level(
	subject Printer_Handle, tree ast.Syntax_Tree_Handle, operation Clash_Sign,
) (level Clash) {
	defer func() { Clash_Invariants(level, "clash_level.level") }()
	Printer_Handle_Invariants(subject, "clash_level.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "clash_level.tree")
	Clash_Sign_Invariants(operation, "clash_level.operation")
	node := ast.Syntax_Node_At(tree, subject.Nodes.Values[depth_load(subject.Counts.Values)])
	unary := ast.Syntax_Token_At(tree, node.Token).Kind
	switch operation {
	case CLASH_SIGN_SLASH:
		if unary == token.KIND_STAR {
			return CLASH_TIGHT
		}
	case CLASH_SIGN_AND:
		if unary == token.KIND_AND {
			return CLASH_TIGHT
		}
		if unary == token.KIND_EXCLUSIVE_OR {
			return CLASH_TIGHT
		}
	case CLASH_SIGN_PLUS:
		if unary == token.KIND_PLUS {
			return CLASH_LOOSE
		}
	case CLASH_SIGN_MINUS:
		if unary == token.KIND_MINUS {
			return CLASH_LOOSE
		}
	}
	return CLASH_NONE
}

// Multiplication_Level distinguishes tighter-sign presence from addition presence.
type Multiplication_Level Boolean

// Multiplication_Level_Invariants states both tighter-level outcomes.
func Multiplication_Level_Invariants(value Multiplication_Level, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Operation contains multiplication level.").
		Ensure()
}

// Binary_Levels_Result keeps spacing evidence at one output boundary.
type Binary_Levels_Result struct {
	// Adds states loose-sign presence independently from tighter signs.
	Adds Boolean
	// Multiplies states tighter-sign presence independently from loose signs.
	Multiplies Multiplication_Level
	// Clash names adjacent unary signs requiring spaces.
	Clash Clash
}

// Binary_Levels_Result_Invariants composes one spacing classification.
func Binary_Levels_Result_Invariants(value Binary_Levels_Result, namespace aver.Namespace) {
	Boolean_Invariants(value.Adds, namespace)
	Multiplication_Level_Invariants(value.Multiplies, namespace)
	Clash_Invariants(value.Clash, namespace)
}

// Reports which of the two levels the canonical form spaces by stand inside the operation the
// walk stands on. A value the source parenthesised states its own form, thus the walk stops at
// it and never reads the levels it holds.
func binary_levels(
	subject Printer_Handle, tree ast.Syntax_Tree_Handle,
) (result Binary_Levels_Result) {
	defer func() { Binary_Levels_Result_Invariants(result, "binary_levels.result") }()
	Printer_Handle_Invariants(subject, "binary_levels.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "binary_levels.tree")
	stood := depth_load(subject.Counts.Values)
	operation := clash_sign(subject, tree)
	level := precedence(subject, tree)
	adds := Boolean(level == LEVEL_ADDITION)
	multiplies := Boolean(level == LEVEL_SIGN_MAXIMUM)
	clash := Clash(CLASH_NONE)
	if !bool(descend_quiet(subject, tree)) {
		return Binary_Levels_Result{
			Adds: adds, Multiplies: Multiplication_Level(multiplies), Clash: clash,
		}
	}
	if bool(reads_levels(subject, tree, level, false)) {
		nested := binary_levels(subject, tree)
		adds = adds || nested.Adds
		multiplies = multiplies || Boolean(nested.Multiplies)
		clash = max(clash, nested.Clash)
	}
	if bool(advance_quiet(subject, tree)) {
		if bool(reads_levels(subject, tree, level, true)) {
			nested := binary_levels(subject, tree)
			adds = adds || nested.Adds
			multiplies = multiplies || Boolean(nested.Multiplies)
			clash = max(clash, nested.Clash)
		}
		if node_kind(subject, tree) == ast.NODE_UNARY {
			clash = max(clash, clash_level(subject, tree, operation))
		}
	}
	ascend(subject)
	// The read steps down into the values one operation holds and climbs back out of each
	// of them, thus it closes on the walk it opened.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(depth_load(subject.Counts.Values) == stood,
			"A read of the levels one operation holds closes on the walk it opened.")
	}
	return Binary_Levels_Result{
		Adds: adds, Multiplies: Multiplication_Level(multiplies), Clash: clash,
	}
}

// Reports whether the value the walk stands on states an operation the form above it spaces as
// one. A value the printer will parenthesise stands as its own form, thus the levels it holds
// say nothing about the form above it.
func reads_levels(
	subject Printer_Handle, tree ast.Syntax_Tree_Handle, level Level, loose Boolean,
) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "reads_levels.yes") }()
	Printer_Handle_Invariants(subject, "reads_levels.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "reads_levels.tree")
	Level_Invariants(level, "reads_levels.level")
	Boolean_Invariants(loose, "reads_levels.loose")
	if node_kind(subject, tree) != ast.NODE_BINARY {
		return false
	}
	// The right value of an operation takes parentheses at the level of that operation,
	// thus it reads the levels of the operation above it only when it binds tighter.
	held := precedence(subject, tree)
	if bool(loose) {
		return Boolean(held >= level)
	}
	return Boolean(held > level)
}

// Writes one operation over a single value, which stands against the value it reads.
func print_unary(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "print_unary.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "print_unary.tree")
	// A sign of one value stands ahead of the value it binds, thus the walk stands on one
	// where the print writes one.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(node_kind(subject, tree) == ast.NODE_UNARY,
			"A sign of one value stands on a sign of the tree.")
	}
	counts_type(subject, tree)
	// A literal that states the address of the type the run it stands in names repeats that
	// type twice over, thus the canonical form writes neither the sign nor the type.
	stood := int(written_load(subject.Counts.Values))
	if !bool(repeats_context(subject, tree, true)) {
		emit_token(subject, tree)
	}
	// Two signs standing together spell a third sign or the mark a comment opens with, thus
	// the sign of one value never closes up against the sign ahead of it.
	if bool(writes_canonical(subject.Flags.Values)) {
		written := int(written_load(subject.Counts.Values))
		if written > stood {
			pair := int(subject.Form.Values[max(stood-1, 0)])<<BYTE_SHIFT |
				int(subject.Form.Values[stood])
			clashed := Boolean(false)
			switch pair {
			case '/'<<BYTE_SHIFT | '/', '/'<<BYTE_SHIFT | '*',
				'+'<<BYTE_SHIFT | '+', '-'<<BYTE_SHIFT | '-',
				'&'<<BYTE_SHIFT | '^', '<'<<BYTE_SHIFT | '-':
				clashed = true
			}
			aver.Always(!bool(clashed),
				"A sign of one value spells no sign with the one ahead of it.")
		}
	}
	// A sign of one value binds the value standing behind it, thus the walk steps down to
	// that value.
	opened := descend_part(subject, tree)
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(bool(opened), "A sign of one value binds a value of the tree.")
	}
	if !bool(opened) {
		return
	}
	// A sign of one value stands against the value it binds, thus no space holds the two
	// apart however the author spaced them.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(!bool(stands_after_space(subject)),
			"A sign of one value stands against the value it binds.")
	}
	print_expression(subject, tree)
	finish_parts(subject, tree)
	ascend(subject)
}

// Reports whether the colons of one slice stand between spaces. Two bounds one of which states
// an operation read harder closed up than spaced, thus the canonical form spaces them.
func slice_blanks(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "slice_blanks.yes") }()
	Printer_Handle_Invariants(subject, "slice_blanks.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "slice_blanks.tree")
	stood := written_load(subject.Counts.Values)
	defer func() {
		aver.Always(written_load(subject.Counts.Values) == stood,
			"A read of the bounds one slice leaves out writes no byte.")
	}()
	if nest_load(subject.Counts.Values) > 1 {
		return false
	}
	// A slice of two parts states one bound, and the blanks the canonical form drops stand
	// only where the slice states a bound of its own on either side.
	if count_parts(subject, tree) != PART_COUNT_THREE {
		return false
	}
	return holds_operation(subject, tree)
}

// Reports whether one part of the form the walk stands on states an operation over two values.
func holds_operation(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "holds_operation.yes") }()
	Printer_Handle_Invariants(subject, "holds_operation.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "holds_operation.tree")
	written := written_load(subject.Counts.Values)
	defer func() {
		aver.Always(written_load(subject.Counts.Values) == written,
			"A read of the operation one value states writes no byte of its own.")
	}()
	held := subject.Nodes.Values[depth_load(subject.Counts.Values)]
	depth := depth_load(subject.Counts.Values)
	found := Boolean(false)
	open := descend_quiet(subject, tree)
	for bool(open) {
		if node_kind(subject, tree) == ast.NODE_BINARY {
			found = true
			break
		}
		open = advance_quiet(subject, tree)
	}
	depth_store(subject.Counts.Values, depth)
	subject.Nodes.Values[depth] = held
	return found
}

// Writes one selector: the value it reads, the period, and the name behind it.
func print_selector(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "print_selector.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "print_selector.tree")
	// A selector reads one name from the form ahead of it, thus the walk stands on one where
	// the print writes one.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(node_kind(subject, tree) == ast.NODE_SELECTOR,
			"A selector the print writes stands on a selector of the tree.")
	}
	subject.Counts.Values[COUNT_FORM] += 1
	subject.Counts.Values[COUNT_SELECTOR] += 1

	if !bool(descend_part(subject, tree)) {
		return
	}
	// A form that read an import by the name the author wrote for it names the package the
	// path states, because the import states that name no longer.
	if !bool(emit_alias(subject, tree)) {
		print_expression(subject, tree)
	}
	// A selector binds the name behind it to the form ahead of it, thus no space holds the
	// dot apart from what it selects from.
	held := int(written_load(subject.Counts.Values))
	if held > 0 {
		if bool(writes_canonical(subject.Flags.Values)) {
			aver.Always(subject.Form.Values[held-1] != byte(SYMBOL_SPACE),
				"The dot of one selector stands against the form it reads.")
		}
	}
	emit_byte(subject, '.')
	// A selector reads one name of the form ahead of it, thus a name stands behind the dot.
	named := advance_part(subject, tree)
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(bool(named), "A selector the print writes reads a name of its form.")
	}
	tabbed := indent_load(subject.Counts.Values)
	marked := flag_load(subject.Flags.Values, FLAG_BROKEN)
	if bool(named) {
		open_continuation(subject, tree, breaks_before(subject, tree))
		emit_token(subject, tree)
	}
	finish_parts(subject, tree)
	// A selector that broke ahead of the name it reads opened the line that name stands on,
	// thus it hands back the tabs and the mark the form around it stood under.
	indent_store(subject.Counts.Values, tabbed)
	flag_store(subject.Flags.Values, FLAG_BROKEN, marked)
	ascend(subject)
}

// Opens the line a continuation of the statement stands on. One statement takes one tab however
// many lines the author broke it into, which is what the canonical form states for a chain.
func open_continuation(subject Printer_Handle, tree ast.Syntax_Tree_Handle, broken Boolean) {
	Printer_Handle_Invariants(subject, "open_continuation.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "open_continuation.tree")
	Boolean_Invariants(broken, "open_continuation.broken")
	if !bool(broken) {
		return
	}
	if !bool(flag_load(subject.Flags.Values, FLAG_BROKEN)) {
		indent_deepen(subject, indent_load(subject.Counts.Values))
		flag_store(subject.Flags.Values, FLAG_BROKEN, true)
	}
	close_line(subject, tree)
	emit_indent(subject)
	// A continuation stands on a line of its own, thus that line holds the tabs it owes and
	// no byte a reader reads. A line as wide as the widest the print reads saturates that
	// read, thus it states nothing.
	owed := int(indent_load(subject.Counts.Values)) * INDENT_WIDTH
	if owed < WIDTH_MAXIMUM {
		if bool(writes_canonical(subject.Flags.Values)) {
			aver.Always(int(take_line_width(subject.Counts.Values)) == owed,
				"A continuation opens a line holding the tabs it owes.")
		}
	}
}

// Writes one assertion: the value it reads and the type it names between brackets.
func print_assertion(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "print_assertion.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "print_assertion.tree")
	// An assertion reads one type from the value ahead of it, thus the walk stands on one
	// where the print writes one.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(node_kind(subject, tree) == ast.NODE_ASSERTION,
			"An assertion the print writes stands on an assertion of the tree.")
	}
	subject.Counts.Values[COUNT_FORM] += 1
	subject.Counts.Values[COUNT_ASSERTION] += 1
	if !bool(descend_part(subject, tree)) {
		return
	}
	print_expression(subject, tree)
	// An assertion reads the type it names from the value ahead of it, thus no space holds
	// the dot apart from that value.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(!bool(stands_after_space(subject)),
			"The dot of one assertion stands against the value it reads.")
	}
	emit_byte(subject, '.')
	emit_byte(subject, '(')
	// A type switch states the word type where an assertion states a type, and the tree holds
	// no node for that word, thus the print writes it wherever no type stands.
	typed := advance_part(subject, tree)
	if bool(typed) {
		print_expression(subject, tree)
	}
	if !bool(typed) {
		emit_word(subject, WORD_TYPE)
	}
	finish_parts(subject, tree)
	ascend(subject)
	emit_byte(subject, ')')
	// An assertion holds the type it names between brackets, thus it closes on the bracket
	// that closes them and never on a space.
	if bool(writes_canonical(subject.Flags.Values)) {
		held := int(written_load(subject.Counts.Values))
		aver.Always(subject.Form.Values[held-1] == ')',
			"An assertion closes on the bracket that holds the type it names.")
	}
}

// Writes one composite literal: the type it names and the elements between its braces.
func print_composite(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "print_composite.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "print_composite.tree")
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(node_kind(subject, tree) == ast.NODE_COMPOSITE,
			"A literal the print writes stands on a literal of the tree.")
	}
	counts := subject.Counts.Values
	counts[COUNT_FORM], counts[COUNT_LITERAL] = counts[COUNT_FORM]+1, counts[COUNT_LITERAL]+1
	width, walked, tabbed := key_width_load(counts), depth_load(counts), indent_load(counts)
	key_width_store(counts, 0)
	// The notes inside these braces stand in a column of their own.
	defer column_store(counts, column_load(counts))
	defer restore_tight(subject, hold_tight(subject, true))
	// A literal running past the widest line states one element to a line.
	over := runs_over(subject, tree)
	defer restore_split(subject, hold_split(subject, false))
	held_apart := flag_load(subject.Flags.Values, FLAG_APART)
	flag_store(subject.Flags.Values, FLAG_APART, over)
	spread_result := take_spread(subject, tree, closes_line(subject, tree))
	force := spread_result.Force || over
	between := Boolean(spread_result.Between) || over
	indent := indent_load(subject.Counts.Values)
	marked := flag_load(subject.Flags.Values, FLAG_BROKEN)
	open := descend_part(subject, tree)
	entered := open
	headed := opens_type(subject, tree, open)
	open = headed.Stand
	place := Literal_Place(LITERAL_FIRST)
	if bool(headed.Headed) {
		place = LITERAL_TYPE
	}
	broken := Boolean(false)
	opened := Boolean(false)
	after := Boolean(false)
	for bool(open) {
		if place == LITERAL_FIRST {
			result := open_braces(subject, tree, force)
			indent, marked = result.Indent, Boolean(result.Marked)
			opened, broken = Boolean(result.Opened), Boolean(result.Broken)
		}
		column_indent_store(subject.Counts.Values, indent)
		if bool(literal_inside(place)) {
			result := open_element(
				subject, tree, opened, broken, marked, between, after, over,
			)
			opened, broken = result.Stand, Boolean(result.Spread)
		}
		after = print_literal_element(subject, tree, opened, over)
		// Every element behind the first stands the same way, thus the walk holds
		// them as one place.
		next := Literal_Place(LITERAL_BEHIND)
		if place == LITERAL_TYPE {
			next = LITERAL_FIRST
		}
		place = next
		open = advance_part(subject, tree)
	}
	column_indent_store(subject.Counts.Values, indent)
	close_literal(subject, tree, broken, entered, Boolean(!bool(literal_inside(place))), force)
	key_width_store(subject.Counts.Values, width)
	flag_store(subject.Flags.Values, FLAG_APART, held_apart)
	// A literal hands back the walk and the tabs it was handed.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(depth_load(subject.Counts.Values) == walked,
			"A literal closes on the walk it opened.")
		aver.Always(indent_load(subject.Counts.Values) == tabbed,
			"A literal closes at the tabs it opened at.")
	}
}

// Writes one element of a literal and the run of keys it closes, and reports whether that element
// states a literal of its own. An element opens a form of its own, thus its parts space their
// signs the way a statement does however deep the literal that holds it stands.
func print_literal_element(
	subject Printer_Handle, tree ast.Syntax_Tree_Handle, opened Boolean, over Boolean,
) (after Boolean) {
	defer func() { Boolean_Invariants(after, "print_literal_element.after") }()
	Printer_Handle_Invariants(subject, "print_literal_element.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "print_literal_element.tree")
	Boolean_Invariants(opened, "print_literal_element.opened")
	Boolean_Invariants(over, "print_literal_element.over")
	held := Boolean(node_kind(subject, tree) == ast.NODE_COMPOSITE)
	nest := nest_load(subject.Counts.Values)
	nest_store(subject.Counts.Values, NEST_MINIMUM)
	// A pair that runs past the widest line stands in no column with the pairs around it.
	spans := Boolean(false)
	if bool(over) {
		spans = runs_over(subject, tree)
	}
	// Every element of one literal stands behind the brace that opens it.
	stood := written_load(subject.Counts.Values)
	lines := subject.Counts.Values[COUNT_LINE]
	print_part(subject, tree, opened, PART_BEHIND)
	// An element states a form of its own, thus it writes the bytes of that form and
	// closes every nest its parts opened.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(written_load(subject.Counts.Values) > stood,
			"An element of one literal writes bytes of its own.")
		aver.Always(nest_load(subject.Counts.Values) == NEST_MINIMUM,
			"An element of one literal closes at the nest it opened at.")
	}
	nest_store(subject.Counts.Values, nest)
	// A run closes at an element that stands over lines of its own, which the lines the
	// element wrote state rather than the lines the author left inside it.
	if subject.Counts.Values[COUNT_LINE] != lines {
		spans = true
	}
	close_key_run(subject, tree, spans)
	return held
}

// LITERAL_TYPE names the type one literal states ahead of its brace.
const LITERAL_TYPE = 0

// LITERAL_FIRST names the element the brace of one literal opens at.
const LITERAL_FIRST = 1

// LITERAL_BEHIND names every element standing behind that first one.
const LITERAL_BEHIND = 2

// Literal_Place names where one part of a literal stands: the type it names, the element its
// brace opens at, or an element behind that one. Every element behind the first stands the same
// way, thus the walk holds all of them as one place.
type Literal_Place uint8

// Literal_Place_Invariants states every place one part of a literal stands at.
func Literal_Place_Invariants(value Literal_Place, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(uint8(value), LITERAL_TYPE, LITERAL_FIRST, LITERAL_BEHIND).
		Ensure()
}

// Reports whether the part the walk stands on stands behind the brace one literal opens, which
// is every part but the type it names and the part its brace opens at.
func literal_inside(place Literal_Place) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "literal_inside.yes") }()
	Literal_Place_Invariants(place, "literal_inside.place")
	return Boolean(place == LITERAL_BEHIND)
}

// Literal_Open states how the braces of one literal opened: the indent its elements answer to,
// the break that stood ahead of it, the break the author left behind the brace, and whether the
// elements stand one to a line.
type Literal_Open struct {
	// Indent holds the tabs the brace opened at.
	Indent Indent
	// Marked reports whether the print already stood broken where the brace opened.
	Marked Marked_Break
	// Opened reports whether the author left a break behind the brace.
	Opened Opened_Break
	// Broken reports whether the elements stand one to a line.
	Broken Broken_Form
}

// Literal_Open_Invariants composes one opening of a literal.
func Literal_Open_Invariants(value Literal_Open, namespace aver.Namespace) {
	Indent_Invariants(value.Indent, namespace)
	Marked_Break_Invariants(value.Marked, namespace)
	Opened_Break_Invariants(value.Opened, namespace)
	Broken_Form_Invariants(value.Broken, namespace)
}

// Opens the body of one literal: the break the author left behind the brace, and the widths the
// run of pairs behind it aligns on. It reports whether that break stands.
func open_literal(
	subject Printer_Handle, tree ast.Syntax_Tree_Handle, force Boolean,
) (opened Boolean) {
	defer func() { Boolean_Invariants(opened, "open_literal.opened") }()
	Printer_Handle_Invariants(subject, "open_literal.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "open_literal.tree")
	Boolean_Invariants(force, "open_literal.force")
	opened = breaks_before(subject, tree) || force
	print_first_break(subject, tree, opened)
	if bool(opened) {
		take_key_width(subject, tree)
	}
	return opened
}

// Spread distinguishes a broken form from one that stays on its opening line.
type Spread Boolean

// Spread_Invariants states both layout outcomes.
func Spread_Invariants(value Spread, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Form stands broken.").
		Ensure()
}

// Open_Result keeps current-line and whole-form layout at one output boundary.
type Open_Result struct {
	// Stand reports whether current part opened another line.
	Stand Boolean
	// Spread reports whether any part broke whole form.
	Spread Spread
}

// Open_Result_Invariants composes one layout step.
func Open_Result_Invariants(value Open_Result, namespace aver.Namespace) {
	Boolean_Invariants(value.Stand, namespace)
	Spread_Invariants(value.Spread, namespace)
}

// Opens the line one element of a literal stands on and reports what the run states behind it:
// whether a break now stands ahead of the elements, and whether the literal stands broken.
func open_element(
	subject Printer_Handle, tree ast.Syntax_Tree_Handle, opened Boolean, broken Boolean,
	marked Boolean, between Boolean, after Boolean, over Boolean,
) (result Open_Result) {
	defer func() { Open_Result_Invariants(result, "open_element.result") }()
	Printer_Handle_Invariants(subject, "open_element.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "open_element.tree")
	Boolean_Invariants(opened, "open_element.opened")
	Boolean_Invariants(broken, "open_element.broken")
	Boolean_Invariants(marked, "open_element.marked")
	Boolean_Invariants(between, "open_element.between")
	Boolean_Invariants(after, "open_element.after")
	Boolean_Invariants(over, "open_element.over")
	here := breaks_before(subject, tree) || over
	// A literal that states one element on a line of its own states every literal it holds on
	// a line of its own, thus a run the author packed behind a break breaks with it.
	if bool(between) {
		if bool(after) {
			here = true
		}
		if node_kind(subject, tree) == ast.NODE_COMPOSITE {
			here = true
		}
	}
	if bool(open_column(here, opened)) {
		indent_deepen(subject, column_indent_load(subject.Counts.Values))
		flag_store(subject.Flags.Values, FLAG_BROKEN, marked)
	}
	print_element_break(subject, tree, here)
	// An element the print breaks to stands on a line of its own, thus the line it opens
	// holds the tabs of the run and nothing the element wrote yet.
	if bool(here) {
		if bool(writes_canonical(subject.Flags.Values)) {
			aver.Always(int(take_line_width(subject.Counts.Values)) ==
				int(indent_load(subject.Counts.Values))*INDENT_WIDTH,
				"An element the print breaks to opens a line of its own.")
		}
	}
	open_key_run(subject, tree, here)
	return Open_Result{Stand: opened || here, Spread: Spread(broken || here)}
}

// Between distinguishes interior breaks from opening and closing breaks.
type Between Boolean

// Between_Invariants states both interior-layout outcomes.
func Between_Invariants(value Between, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Break stands between parts.").
		Ensure()
}

// Take_Spread_Result keeps forced and interior breaks at one output boundary.
type Take_Spread_Result struct {
	// Force reports whether literal must open broken.
	Force Boolean
	// Between reports whether any two elements have a break.
	Between Between
}

// Take_Spread_Result_Invariants composes one literal-layout reading.
func Take_Spread_Result_Invariants(value Take_Spread_Result, namespace aver.Namespace) {
	Boolean_Invariants(value.Force, namespace)
	Between_Invariants(value.Between, namespace)
}

// Element_Written names a literal whose run the walk already read one element of, which every
// element behind the first stands behind.
type Element_Written Boolean

// Element_Written_Invariants states both stands of the run.
func Element_Written_Invariants(value Element_Written, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The run already holds an element.").
		Ensure()
}

// Reads how the elements of one literal stand: whether the braces or the elements open lines of
// their own, and whether a break stands between two elements. A literal the author broke states
// one element to a line, thus the print reads the whole run before it writes the first element.
func take_spread(
	subject Printer_Handle, tree ast.Syntax_Tree_Handle, tail Boolean,
) (result Take_Spread_Result) {
	defer func() { Take_Spread_Result_Invariants(result, "take_spread.result") }()
	Printer_Handle_Invariants(subject, "take_spread.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "take_spread.tree")
	Boolean_Invariants(tail, "take_spread.tail")
	stood := written_load(subject.Counts.Values)
	defer func() {
		aver.Always(written_load(subject.Counts.Values) == stood,
			"A read of the lines one literal spreads writes no byte of its own.")
	}()
	if bool(flag_load(subject.Flags.Values, FLAG_FLAT)) {
		return Take_Spread_Result{Force: false, Between: false}
	}
	held := subject.Nodes.Values[depth_load(subject.Counts.Values)]
	depth := depth_load(subject.Counts.Values)
	brace := ast.Syntax_Node_At(tree, held).Token
	force := tail
	separated := Boolean(false)
	written := Element_Written(false)
	open := descend_quiet(subject, tree)
	for bool(open) {
		held_depth := depth_load(subject.Counts.Values)
		enter_leftmost(subject, tree)
		leftmost := subject.Nodes.Values[depth_load(subject.Counts.Values)]
		depth_store(subject.Counts.Values, held_depth)
		found := ast.Syntax_Node_At(tree, leftmost).Token
		// The type one literal names stands ahead of its brace, thus a part that opens
		// behind the brace is an element and every other one names the type.
		if found > brace {
			here := breaks_before(subject, tree)
			if bool(written) {
				separated = separated || here
			}
			if !bool(written) {
				force = force || here
			}
			written = true
		}
		open = advance_quiet(subject, tree)
	}
	Element_Written_Invariants(written, "take_spread.written")
	// A break between two elements stands where an element already stood, thus a literal
	// that holds such a break holds an element of its own.
	if bool(separated) {
		aver.Always(bool(written), "A literal that breaks between elements holds one.")
	}
	depth_store(subject.Counts.Values, depth)
	subject.Nodes.Values[depth] = held
	return Take_Spread_Result{Force: force || separated, Between: Between(separated)}
}

// Opens the line one part of a call or an index stands on and reports what the form now states:
// whether a break stands ahead of the parts, and whether the form stands broken. Each part stands
// where the author put it, on the line the part before it holds or on a line of its own, and a
// part that indented lines of its own leaves them behind at the column the form opened at.
func open_argument(
	subject Printer_Handle,
	tree ast.Syntax_Tree_Handle,
	opened Boolean,
	broken Boolean,
	marked Boolean,
	over Boolean,
) (result Open_Result) {
	defer func() { Open_Result_Invariants(result, "open_argument.result") }()
	Printer_Handle_Invariants(subject, "open_argument.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "open_argument.tree")
	Boolean_Invariants(opened, "open_argument.opened")
	Boolean_Invariants(broken, "open_argument.broken")
	Boolean_Invariants(marked, "open_argument.marked")
	Boolean_Invariants(over, "open_argument.over")
	// The points the author wrote behind an argument stand against that argument, thus they
	// take neither a comma nor a break of their own. Points that hold the value they spread
	// state an argument of their own and take both.
	if bool(trails_spread(subject, tree)) {
		return Open_Result{Stand: opened, Spread: Spread(broken)}
	}
	here := breaks_before(subject, tree) || over
	if bool(open_column(here, opened)) {
		indent_deepen(subject, column_indent_load(subject.Counts.Values))
		flag_store(subject.Flags.Values, FLAG_BROKEN, marked)
	}
	print_element_break(subject, tree, here)
	// An argument the print breaks to stands on a line of its own, thus the line it opens
	// holds the tabs of the call and nothing the argument wrote yet.
	if bool(here) {
		if bool(writes_canonical(subject.Flags.Values)) {
			aver.Always(int(take_line_width(subject.Counts.Values)) ==
				int(indent_load(subject.Counts.Values))*INDENT_WIDTH,
				"An argument the print breaks to opens a line of its own.")
		}
	}
	return Open_Result{Stand: opened || here, Spread: Spread(broken || here)}
}

// Pushes onto the walk stack the type the literal the walk reads inherits from its context: the
// type the elements of the nearest literal around it wear, or its keys where the literal stands
// as a key, and the type a pointer names where the address of the literal stands as such an
// element. The walk stands on the type the literal names, or on the sign that takes the address
// of the literal where address holds. A literal no context names a type for pushes nothing.
func push_context(
	subject Printer_Handle, tree ast.Syntax_Tree_Handle, address Boolean,
) (found Boolean) {
	defer func() { Boolean_Invariants(found, "push_context.found") }()
	Printer_Handle_Invariants(subject, "push_context.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "push_context.tree")
	Boolean_Invariants(address, "push_context.address")
	depth := int(depth_load(subject.Counts.Values))
	above, sign_place := depth-2, depth-2
	if bool(address) {
		above, sign_place = depth-1, depth
	}
	context, keyed, brace := ast.INDEX_ABSENT, ast.INDEX_ABSENT, ast.Token_Index(0)
	for step := 1; step <= above; step++ {
		node := ast.Syntax_Node_At(tree, subject.Nodes.Values[step])
		if node.Kind == ast.NODE_KEY_VALUE {
			if ast.Index(node.First_Child) == subject.Nodes.Values[step+1] {
				context = keyed
			}
		}
		if node.Kind != ast.NODE_COMPOSITE {
			continue
		}
		// A literal that states no type of its own opens at its brace, thus a head behind
		// that brace states elements and names no type for them.
		context, keyed = ast.INDEX_ABSENT, ast.INDEX_ABSENT
		brace = node.Token
		if ast.Index(node.First_Child) == ast.INDEX_ABSENT {
			continue
		}
		named := ast.Syntax_Node_At(tree, ast.Index(node.First_Child))
		if named.Token >= node.Token {
			continue
		}
		// The element of an array stands behind the length it states, and the value of a
		// map stands behind the key it reads, thus the last child names the type of an
		// element and the first one names the type of a key.
		switch named.Kind {
		case ast.NODE_ARRAY_TYPE, ast.NODE_SLICE_TYPE, ast.NODE_MAP_TYPE:
			context = ast.Index(named.First_Child)
			for ast.Index(ast.Syntax_Node_At(tree, context).Next) != ast.INDEX_ABSENT {
				context = ast.Index(ast.Syntax_Node_At(tree, context).Next)
			}
			if named.Kind == ast.NODE_MAP_TYPE {
				keyed = ast.Index(named.First_Child)
			}
		}
	}
	// The address of a literal wears the pointer its context names, thus the literal itself
	// wears the type that pointer names.
	pointer := ast.Syntax_Node_At(tree, context)
	sign := ast.Syntax_Node_At(tree, subject.Nodes.Values[sign_place])
	if sign.Kind == ast.NODE_UNARY {
		if ast.Syntax_Token_At(tree, sign.Token).Kind == token.KIND_AND {
			if pointer.Kind == ast.NODE_POINTER_TYPE {
				context = ast.Index(pointer.First_Child)
			}
		}
	}
	if context == ast.INDEX_ABSENT {
		return false
	}
	// A literal wears the type the head behind its brace names.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(ast.Syntax_Node_At(tree, context).Token < brace,
			"The type a literal wears stands ahead of the brace it opens at.")
	}

	subject.Nodes.Values[depth+1] = context
	depth_store(subject.Counts.Values, Depth(depth+1))
	return true
}

// Reports whether the type the literal the walk reads names is the type its context already
// names, which the canonical form writes once. The walk stands on the type the literal names, or
// on the sign that takes the address of the literal where address holds.
func repeats_context(
	subject Printer_Handle, tree ast.Syntax_Tree_Handle, address Boolean,
) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "repeats_context.yes") }()
	Printer_Handle_Invariants(subject, "repeats_context.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "repeats_context.tree")
	Boolean_Invariants(address, "repeats_context.address")
	depth := int(depth_load(subject.Counts.Values))
	own := subject.Nodes.Values[depth]
	if bool(address) {
		sign := ast.Syntax_Node_At(tree, own)
		if ast.Syntax_Token_At(tree, sign.Token).Kind != token.KIND_AND {
			return false
		}
		if ast.Index(sign.First_Child) == ast.INDEX_ABSENT {
			return false
		}
		operand := ast.Syntax_Node_At(tree, ast.Index(sign.First_Child))
		if operand.Kind != ast.NODE_COMPOSITE {
			return false
		}
		// A literal that states no type of its own opens at its brace, thus a head behind
		// that brace states elements and never a type.
		own = ast.Index(operand.First_Child)
		if own == ast.INDEX_ABSENT {
			return false
		}
		if ast.Syntax_Node_At(tree, own).Token >= operand.Token {
			return false
		}
	}
	if depth+2 >= DEPTH_MAXIMUM {
		return false
	}
	if !bool(push_context(subject, tree, address)) {
		return false
	}
	subject.Nodes.Values[depth+2] = own
	depth_store(subject.Counts.Values, Depth(depth+2))
	yes = same_form(subject, tree)
	// The read pushes the type its context names and the type the literal states, and the
	// compare of the two hands back the walk it was handed.
	aver.Always(depth_load(subject.Counts.Values) == Depth(depth+2),
		"A compare of the type one context names hands back the walk it was handed.")
	depth_store(subject.Counts.Values, Depth(depth))
	return yes
}

// Headed distinguishes a literal whose print writes a type ahead of its brace from one whose
// print writes the brace alone.
type Headed Boolean

// Headed_Invariants states both heads a literal writes.
func Headed_Invariants(value Headed, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Literal writes a type ahead of its brace.").
		Ensure()
}

// Head_Result keeps whether the walk stands on a part apart from whether a type heads the literal.
type Head_Result struct {
	// Stand reports whether the walk stands on a part of the literal.
	Stand Boolean
	// Headed reports whether the part the walk stands on is the type the print writes.
	Headed Headed
}

// Head_Result_Invariants composes one read of a literal head.
func Head_Result_Invariants(value Head_Result, namespace aver.Namespace) {
	Boolean_Invariants(value.Stand, namespace)
	Headed_Invariants(value.Headed, namespace)
}

// Reports whether the walk stands on a part of the literal it entered, and whether that part is a
// type the print writes. The walk steps over the type the literal names where the literal around
// it already named that type, which the canonical form writes once. A literal that stands as an
// element of a slice of pointers states the type its pointer names, thus the print reads no type
// for it however the literal that holds it stands.
func opens_type(
	subject Printer_Handle, tree ast.Syntax_Tree_Handle, open Boolean,
) (result Head_Result) {
	defer func() { Head_Result_Invariants(result, "opens_type.result") }()
	Printer_Handle_Invariants(subject, "opens_type.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "opens_type.tree")
	Boolean_Invariants(open, "opens_type.open")
	if !bool(typed_head(subject, tree, open)) {
		return Head_Result{Stand: open, Headed: false}
	}
	if bool(repeats_context(subject, tree, false)) {
		// The run the element stands in already names the type, thus the print drops it
		// and writes no byte for the head the author wrote.
		counts_dropped_type(subject, tree)
		stood := written_load(subject.Counts.Values)
		held := Head_Result{Stand: advance_part(subject, tree), Headed: false}
		aver.Always(written_load(subject.Counts.Values) == stood,
			"An element that repeats the type of its run writes no byte for it.")
		return held
	}
	return Head_Result{Stand: open, Headed: true}
}

// Reads the widest key of the run of pairs the walk stands at the opening of. Every value of one
// run stands in the same column, thus the print owes the widest key before it writes the first
// one. A run stops where a pair stops, because a literal that holds anything else aligns nothing.
func take_key_width(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "take_key_width.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "take_key_width.tree")
	held := subject.Nodes.Values[depth_load(subject.Counts.Values)]
	widest := Measured_Width(0)
	// A pair pads its value to the column the widest key states, and a pair of that key
	// takes no padding, thus the widest padded line states what the column may spend.
	slack, widest_slack := 0, 0
	paired, seen, open := Boolean(false), Boolean(false), Boolean(true)
	for bool(open) {
		if node_kind(subject, tree) != ast.NODE_KEY_VALUE {
			break
		}
		seen = true
		width := Measured_Width(key_width(subject, tree))
		// A pair over lines of its own stands in no column with the pairs around it.
		if bool(spans_value(subject, tree)) {
			if !bool(paired) {
				widest = width
			}
			break
		}

		closes := breaks_run(subject, tree)
		held_slack := int(element_width(subject, tree)) - int(width)
		if width > widest {
			slack = max(slack, widest_slack)
			widest_slack = held_slack
		}
		if width <= widest {
			slack = max(slack, held_slack)
		}
		widest = max(widest, width)
		paired = true
		if bool(closes) {
			break
		}
		open = advance(subject, tree)
		for range ast.NODE_COUNT_MAXIMUM {
			if !bool(open) {
				break
			}
			if !bool(trails_note(subject, tree)) {
				break
			}
			open = advance(subject, tree)
		}
		// Two pairs that share a line stand in no column.
		if bool(open) {
			if !bool(flag_load(subject.Flags.Values, FLAG_APART)) {
				if !bool(breaks_before(subject, tree)) {
					break
				}
			}
		}
	}
	subject.Nodes.Values[depth_load(subject.Counts.Values)] = held
	// A column that carries a padded line past the widest line a reader reads pads nothing,
	// and a run that pads no line at all states the column its keys spell.
	if slack > 0 {
		if slack+int(widest) > COLUMN_WIDTH_MAXIMUM {
			widest = 0
		}
	}
	key_width_store(subject.Counts.Values, Measured_Width(widest))
	// A run that states a column stood on a pair to read the widest key from.
	if key_width_load(subject.Counts.Values) != 0 {
		aver.Always(bool(seen), "A run of pairs that states a column holds a pair.")
	}
}

// Reads the width the key of the pair the walk stands on writes. A key states a whole expression
// rather than one word, thus the column it opens the value at answers for the form the print
// writes and not for the token the key names.
func key_width(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (width Part_Width) {
	defer func() { Part_Width_Invariants(width, "key_width.width") }()
	Printer_Handle_Invariants(subject, "key_width.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "key_width.tree")
	stand := descend_quiet(subject, tree)
	aver.Always(bool(stand), "A pair of a literal holds the key it opens.")
	nest := nest_load(subject.Counts.Values)
	nest_store(subject.Counts.Values, NEST_MINIMUM)
	measured := measure_expression(subject, tree)
	nest_store(subject.Counts.Values, nest)
	ascend(subject)
	// A key states a whole expression, thus the column it opens the value at stands past the
	// colon that closes it and never against the key itself.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(measured > 0, "A key of one pair states a width of its own.")
	}
	return Part_Width(measured)
}

// Reports whether the form the walk stands on writes over more than one line.
func spans_form(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "spans_form.yes") }()
	Printer_Handle_Invariants(subject, "spans_form.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "spans_form.tree")
	depth := depth_load(subject.Counts.Values)
	stood := written_load(subject.Counts.Values)
	defer func() {
		aver.Always(written_load(subject.Counts.Values) == stood,
			"A read of the lines one form spans writes no byte of its own.")
	}()
	enter_leftmost(subject, tree)
	first := ast.Syntax_Node_At(tree, subject.Nodes.Values[depth_load(subject.Counts.Values)])
	depth_store(subject.Counts.Values, depth)
	enter_rightmost(subject, tree)
	final := ast.Syntax_Node_At(tree, subject.Nodes.Values[depth_load(subject.Counts.Values)])
	depth_store(subject.Counts.Values, depth)
	opening := ast.Syntax_Token_At(tree, first.Token)
	last := ast.Syntax_Token_At(tree, final.Token)
	// A raw literal spans the lines it holds, thus the span closes at the end of the final
	// token rather than at the byte that token opens.
	aver.Always(opening.Offset <= last.Offset,
		"A form opens at a byte ahead of the one its last token opens at.")
	for place := int(opening.Offset); place < int(last.Offset)+int(last.Size); place++ {
		if subject.Source.Values[place] == '\n' {
			return true
		}
	}
	return false
}

// Reports whether the pair the walk stands on closes the run it stands in. A comment or an empty
// line behind a pair closes the run, thus the pairs behind that line align on a width of their
// own.
func breaks_run(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "breaks_run.yes") }()
	Printer_Handle_Invariants(subject, "breaks_run.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "breaks_run.tree")
	written := written_load(subject.Counts.Values)
	defer func() {
		aver.Always(written_load(subject.Counts.Values) == written,
			"A read of the line that breaks one run writes no byte of its own.")
	}()
	stood := depth_load(subject.Counts.Values)
	held := subject.Nodes.Values[stood]
	stand := advance(subject, tree)
	// Every note the pair carries behind it stands in a column of its own rather than
	// last the run, thus the walk steps over them to read what stands behind the pair.
	for range ast.NODE_COUNT_MAXIMUM {
		if !bool(stand) {
			break
		}
		if !bool(trails_note(subject, tree)) {
			break
		}
		stand = advance(subject, tree)
	}
	closes := closes_run(subject, tree, stand)
	subject.Nodes.Values[depth_load(subject.Counts.Values)] = held
	// The read steps along the run with a step that holds the slot it stood on, thus it
	// hands back the walk it opened.
	aver.Always(depth_load(subject.Counts.Values) == stood,
		"A read of one run of pairs holds the slot it opened.")
	return closes
}

// Reports whether the run of pairs closes where the walk stands. Nothing behind the pair, trivia
// behind it, or a pair that shares its line each close the run the pair stood in.
func closes_run(subject Printer_Handle, tree ast.Syntax_Tree_Handle, stand Boolean) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "closes_run.yes") }()
	Printer_Handle_Invariants(subject, "closes_run.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "closes_run.tree")
	Boolean_Invariants(stand, "closes_run.stand")
	stood := written_load(subject.Counts.Values)
	defer func() {
		aver.Always(written_load(subject.Counts.Values) == stood,
			"A read of the pair that closes one run writes no byte of its own.")
	}()
	if !bool(stand) {
		return true
	}
	if bool(trivia_kind(subject, tree)) {
		return true
	}
	// A form the print writes one part to a line opens a line for every pair it holds, thus
	// the run closes at nothing the author wrote on one line.
	if bool(flag_load(subject.Flags.Values, FLAG_APART)) {
		return false
	}
	return !breaks_before(subject, tree)
}

// Reports whether the node the walk stands on states a note that closes the line before it.
func trails_note(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "trails_note.yes") }()
	Printer_Handle_Invariants(subject, "trails_note.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "trails_note.tree")
	written := written_load(subject.Counts.Values)
	defer func() {
		aver.Always(written_load(subject.Counts.Values) == written,
			"A read of the note one pair carries writes no byte of its own.")
	}()
	if node_kind(subject, tree) != ast.NODE_COMMENT {
		return false
	}
	return !stands_alone(subject, tree)
}

// FUNCTION_HOLDS names a function literal, which states the block it holds behind its signature.
const FUNCTION_HOLDS = 0

// FUNCTION_STATES names a function type, which states the signature alone.
const FUNCTION_STATES = 1

// Function_Kind names which of the two forms one signature opens.
type Function_Kind uint8

// Function_Kind_Invariants states both forms a signature opens.
func Function_Kind_Invariants(value Function_Kind, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value), FUNCTION_HOLDS, FUNCTION_STATES).
		Ensure()
}

// Reads which of the two forms the walk stands on.
func function_kind(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (held Function_Kind) {
	defer func() { Function_Kind_Invariants(held, "function_kind.held") }()
	Printer_Handle_Invariants(subject, "function_kind.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "function_kind.tree")
	if node_kind(subject, tree) == ast.NODE_FUNCTION_LITERAL {
		return FUNCTION_HOLDS
	}
	return FUNCTION_STATES
}

// Writes one function literal: the word, its signature, and the block behind it.
func print_function_literal(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "print_function_literal.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "print_function_literal.tree")
	// A function literal states a signature and a body, and a function type states the
	// signature alone, thus the walk stands on one of the two where the print writes one.
	if bool(writes_canonical(subject.Flags.Values)) {
		Function_Kind_Invariants(function_kind(subject, tree), "print_literal.held")
	}
	if node_kind(subject, tree) == ast.NODE_FUNCTION_LITERAL {
		subject.Counts.Values[COUNT_FORM] += 1
		subject.Counts.Values[COUNT_FUNCTION] += 1
	}
	stood := written_load(subject.Counts.Values)
	// The body the literal holds spaces its statements at the nest a statement stands at,
	// thus the literal hands back the nest the value that holds it stood at.
	defer nest_store(subject.Counts.Values, nest_load(subject.Counts.Values))
	emit_word(subject, WORD_FUNCTION)
	open := descend_part(subject, tree)
	entered := open
	layout_run(subject, tree, RULE_SIGNATURE, open)
	if bool(entered) {
		ascend(subject)
	}
	// A function literal states the word it opens with and the signature behind it, thus it
	// writes bytes of its own however the author wrote it.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(written_load(subject.Counts.Values) > stood,
			"A function the print writes spells the word it opens with.")
	}
}

// Writes one range clause: the word and the value it steps over.
func print_range(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "print_range.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "print_range.tree")
	// A range reads the values one form holds, thus the walk stands on one where the print
	// writes one.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(node_kind(subject, tree) == ast.NODE_RANGE,
			"A range the print writes stands on a range of the tree.")
	}
	subject.Counts.Values[COUNT_FORM] += 1
	subject.Counts.Values[COUNT_RANGE] += 1
	emit_word(subject, WORD_RANGE)
	if !bool(descend_part(subject, tree)) {
		return
	}
	emit_space(subject)
	stood := written_load(subject.Counts.Values)
	print_expression(subject, tree)
	// A range reads the values one form holds, thus that form spells bytes behind the word
	// the range opens with.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(written_load(subject.Counts.Values) > stood,
			"A range names the form it reads behind the word it opens with.")
	}
	finish_parts(subject, tree)
	ascend(subject)
}

// Writes one type form, which is what an expression that names a type states.
func print_type_form(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "print_type_form.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "print_type_form.tree")
	stood := written_load(subject.Counts.Values)
	counts_type(subject, tree)
	switch node_kind(subject, tree) {
	case ast.NODE_POINTER_TYPE:
		print_prefixed(subject, tree, PREFIX_POINTER)
	case ast.NODE_SLICE_TYPE:
		print_prefixed(subject, tree, PREFIX_SLICE)
	case ast.NODE_ELLIPSIS:
		print_prefixed(subject, tree, PREFIX_ELLIPSIS)
	case ast.NODE_CHANNEL_TYPE:
		print_prefixed(subject, tree, PREFIX_CHANNEL)
	case ast.NODE_CHANNEL_SEND:
		print_prefixed(subject, tree, PREFIX_SEND)
	case ast.NODE_CHANNEL_RECEIVE:
		print_prefixed(subject, tree, PREFIX_RECEIVE)
	case ast.NODE_ARRAY_TYPE:
		print_array(subject, tree)
	case ast.NODE_MAP_TYPE:
		print_map(subject, tree)
	case ast.NODE_FUNCTION_TYPE:
		print_function_literal(subject, tree)
	case ast.NODE_STRUCTURE_TYPE:
		print_structure(subject, tree)
	case ast.NODE_INTERFACE_TYPE:
		print_constraint(subject, tree)
	case ast.NODE_UNION:
		print_union(subject, tree)
	case ast.NODE_TERM:
		print_term(subject, tree)
	default:
		emit_token(subject, tree)
	}
	// Every type of the tree stands as a run of bytes a reader reads, thus a type that wrote
	// none would leave the form it stands in reading as the form around it.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(written_load(subject.Counts.Values) > stood,
			"A type the print writes spells a byte of its own.")
	}
}

// PREFIX_POINTER opens a pointer type.
const PREFIX_POINTER Prefix = 0

// PREFIX_SLICE opens a slice type.
const PREFIX_SLICE Prefix = 1

// PREFIX_ELLIPSIS opens a run of arguments.
const PREFIX_ELLIPSIS Prefix = 2

// PREFIX_CHANNEL opens a channel of both directions.
const PREFIX_CHANNEL Prefix = 3

// PREFIX_SEND opens a channel a caller only sends on.
const PREFIX_SEND Prefix = 4

// PREFIX_RECEIVE opens a channel a caller only reads from.
const PREFIX_RECEIVE Prefix = 5

// PREFIX_MINIMUM is the first prefix a type opens with.
const PREFIX_MINIMUM = uint8(PREFIX_POINTER)

// PREFIX_MAXIMUM is the last prefix a type opens with.
const PREFIX_MAXIMUM = uint8(PREFIX_RECEIVE)

// Prefix names the bytes one type opens with, which stand ahead of the type they read.
type Prefix uint8

// Prefix_Invariants states every prefix a type opens with.
func Prefix_Invariants(value Prefix, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), PREFIX_MINIMUM, PREFIX_MAXIMUM).
		Ensure()
}

// Writes one type that opens with a prefix and reads one type behind it.
func print_prefixed(subject Printer_Handle, tree ast.Syntax_Tree_Handle, prefix Prefix) {
	Printer_Handle_Invariants(subject, "print_prefixed.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "print_prefixed.tree")
	Prefix_Invariants(prefix, "print_prefixed.prefix")
	// A type that opens on a mark of its own states the type it holds behind that mark, thus
	// the walk stands on one of those types where the print writes one.
	if bool(writes_canonical(subject.Flags.Values)) {
		held := node_kind(subject, tree)
		stated := held == ast.NODE_POINTER_TYPE || held == ast.NODE_SLICE_TYPE
		stated = stated || held == ast.NODE_ELLIPSIS || held == ast.NODE_CHANNEL_TYPE
		stated = stated || held == ast.NODE_CHANNEL_SEND
		stated = stated || held == ast.NODE_CHANNEL_RECEIVE
		aver.Always(stated, "A type that opens on a mark stands on such a type.")
	}
	emit_prefix(subject, prefix)
	if !bool(descend_part(subject, tree)) {
		return
	}
	print_expression(subject, tree)
	finish_parts(subject, tree)
	ascend(subject)
}

// Writes the bytes one prefix spells.
func emit_prefix(subject Printer_Handle, prefix Prefix) {
	Printer_Handle_Invariants(subject, "emit_prefix.subject")
	Prefix_Invariants(prefix, "emit_prefix.prefix")
	stood := take_line_width(subject.Counts.Values)
	switch prefix {
	case PREFIX_POINTER:
		emit_byte(subject, '*')
	case PREFIX_SLICE:
		emit_byte(subject, '[')
		emit_byte(subject, ']')
	case PREFIX_ELLIPSIS:
		emit_byte(subject, '.')
		emit_byte(subject, '.')
		emit_byte(subject, '.')
	case PREFIX_CHANNEL:
		emit_word(subject, WORD_CHANNEL)
		emit_space(subject)
	case PREFIX_SEND:
		emit_word(subject, WORD_CHANNEL)
		emit_byte(subject, '<')
		emit_byte(subject, '-')
		emit_space(subject)
	case PREFIX_RECEIVE:
		emit_byte(subject, '<')
		emit_byte(subject, '-')
		emit_word(subject, WORD_CHANNEL)
		emit_space(subject)
	}
	// Every mark one type opens with spells bytes of its own, thus the line holds them.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(take_line_width(subject.Counts.Values) > stood,
			"A mark one type opens with spends columns of its own.")
		spent := int(take_line_width(subject.Counts.Values)) - int(stood)
		aver.Always(spent == int(prefix_width(prefix)),
			"A mark one type opens with spends the columns it states.")
	}
}

// ARROW_SIZE is how many bytes the arrow of a channel spends.
const ARROW_SIZE = 2

// PREFIX_WIDTH_MINIMUM is the columns the narrowest mark spends, which is the star of a pointer.
const PREFIX_WIDTH_MINIMUM = 1

// PREFIX_WIDTH_MAXIMUM is the columns the widest mark spends, which is the word and the arrow of
// a channel that sends or reads alone.
const PREFIX_WIDTH_MAXIMUM = 7

// Prefix_Width is how many columns one mark a type opens with spends.
type Prefix_Width int

// Prefix_Width_Invariants states every width one mark a type opens with spends.
func Prefix_Width_Invariants(value Prefix_Width, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), PREFIX_WIDTH_MINIMUM, PREFIX_WIDTH_MAXIMUM).
		Ensure()
}

// Writes one array type: the count between brackets and the element behind them.
func print_array(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "print_array.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "print_array.tree")
	// An array names the count it holds and the type of its elements, thus the walk stands
	// on one where the print writes one.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(node_kind(subject, tree) == ast.NODE_ARRAY_TYPE,
			"An array the print writes stands on an array of the tree.")
	}
	emit_byte(subject, '[')
	if !bool(descend_part(subject, tree)) {
		emit_byte(subject, ']')
		return
	}
	print_expression(subject, tree)
	// An array states the count it holds between its brackets, thus the bracket closes
	// against that count and no space stands between the two.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(!bool(stands_after_space(subject)),
			"An array closes its bracket against the count it holds.")
	}
	emit_byte(subject, ']')
	if bool(advance_part(subject, tree)) {
		print_expression(subject, tree)
	}
	finish_parts(subject, tree)
	ascend(subject)
}

// Writes one map type: the key between brackets and the value behind them.
func print_map(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "print_map.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "print_map.tree")
	// A map names the type of its keys and the type of its values, thus the walk stands on
	// one where the print writes one.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(node_kind(subject, tree) == ast.NODE_MAP_TYPE,
			"A map the print writes stands on a map of the tree.")
	}
	emit_word(subject, WORD_MAP)
	emit_byte(subject, '[')
	if !bool(descend_part(subject, tree)) {
		emit_byte(subject, ']')
		return
	}
	print_expression(subject, tree)
	// A map states the type of its keys between brackets, thus the bracket closes against
	// that type and no space stands between the two.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(!bool(stands_after_space(subject)),
			"A map closes its bracket against the type of its keys.")
	}
	emit_byte(subject, ']')
	if bool(advance_part(subject, tree)) {
		print_expression(subject, tree)
	}
	finish_parts(subject, tree)
	ascend(subject)
}

// Writes one union of terms, which stand between signs.
func print_union(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "print_union.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "print_union.tree")
	// A union holds the terms one constraint admits, thus the walk stands on a union where
	// the print writes one.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(node_kind(subject, tree) == ast.NODE_UNION,
			"A union the print writes stands on a union of the tree.")
	}
	subject.Counts.Values[COUNT_UNION] += 1
	open := descend_part(subject, tree)
	entered := open
	written := Term_Written(false)
	for bool(open) {
		Term_Written_Invariants(written, "print_union.written")
		if bool(written) {
			emit_space(subject)
			emit_byte(subject, '|')
			emit_space(subject)
		}
		stood := written_load(subject.Counts.Values)
		print_expression(subject, tree)
		written = true
		// A sign of one union holds two terms apart, thus every term it states writes
		// bytes of its own.
		if bool(writes_canonical(subject.Flags.Values)) {
			aver.Always(written_load(subject.Counts.Values) > stood,
				"A term of one union writes bytes of its own.")
		}
		open = advance_part(subject, tree)
	}
	if bool(entered) {
		finish_parts(subject, tree)
		ascend(subject)
	}
}

// Term_Written names a union that already wrote a term of its own, which every term behind the
// first stands behind.
type Term_Written Boolean

// Term_Written_Invariants states both stands of the union.
func Term_Written_Invariants(value Term_Written, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The union already wrote a term.").
		Ensure()
}

// Writes one term of a constraint, which the tilde opens where it admits a type of its own shape.
func print_term(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "print_term.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "print_term.tree")
	// A term names one type a constraint admits, thus the walk stands on one where the print
	// writes one.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(node_kind(subject, tree) == ast.NODE_TERM,
			"A term the print writes stands on a term of the tree.")
	}
	subject.Counts.Values[COUNT_TERM] += 1
	if bool(approximate(subject, tree)) {
		emit_byte(subject, '~')
	}
	if !bool(descend_part(subject, tree)) {
		return
	}
	print_expression(subject, tree)
	finish_parts(subject, tree)
	ascend(subject)
}

// Reports whether one term admits every type of its shape, which the tilde ahead of it states.
func approximate(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "approximate.yes") }()
	Printer_Handle_Invariants(subject, "approximate.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "approximate.tree")
	written := written_load(subject.Counts.Values)
	defer func() {
		aver.Always(written_load(subject.Counts.Values) == written,
			"A read of the width one form spends writes no byte of its own.")
	}()
	index := subject.Nodes.Values[depth_load(subject.Counts.Values)]
	position := ast.Syntax_Node_At(tree, index).Token
	if position == 0 {
		return false
	}
	return ast.Syntax_Token_At(tree, position-1).Kind == token.KIND_TILDE
}

// Writes one struct type: the word, the fields one tab deeper, and the brace that closes it. A
// struct that states no field stands on one line, which is the form the canonical printer writes.
func print_structure(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "print_structure.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "print_structure.tree")
	// A structure holds its fields between braces, thus the walk stands on one where the
	// print writes one.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(node_kind(subject, tree) == ast.NODE_STRUCTURE_TYPE,
			"A structure the print writes stands on a structure of the tree.")
	}
	emit_word(subject, WORD_STRUCTURE)
	if bool(spans_one_line(subject, tree)) {
		print_inline_structure(subject, tree)
		return
	}
	if bool(holds_no_part(subject, tree)) {
		emit_byte(subject, '{')
		emit_byte(subject, '}')
		close_empty_list(subject, tree)
		return
	}
	stood := descend(subject, tree)
	aver.Always(bool(stood), "A structure that holds a field opens on that field.")
	emit_space(subject)
	emit_byte(subject, '{')
	// A note the author left on the line the brace opens keeps that line, thus the print
	// writes it before it closes the line.
	stood = open_brace_note(subject, tree)
	close_line(subject, tree)
	indent_deepen(subject, indent_load(subject.Counts.Values))
	// A run of fields closes up against its braces, thus the struct opens at its first field
	// and closes at its last one.
	held_tight := flag_load(subject.Flags.Values, FLAG_TIGHT)
	flag_store(subject.Flags.Values, FLAG_TIGHT, true)
	run := Run_Widths{Names: 0, Tags: 0}
	open := stood
	for bool(open) {
		// A note the author left on the line the brace closes keeps that line, thus
		// the struct holds it and the line that brace closes carries it.
		if bool(trails_close(subject, tree)) {
			print_trivia(subject, tree)
			open = advance(subject, tree)
			continue
		}
		run = print_field(subject, tree, run)
		open = advance(subject, tree)
	}
	flag_store(subject.Flags.Values, FLAG_TIGHT, held_tight)
	ascend(subject)
	indent_shallow(subject, indent_load(subject.Counts.Values))
	emit_indent(subject)
	closes_body(subject)
	emit_byte(subject, '}')
	// The brace that closes a structure stands on a line of its own, at the tabs the
	// structure opened at and behind nothing else.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(int(take_line_width(subject.Counts.Values)) ==
			int(indent_load(subject.Counts.Values))*INDENT_WIDTH+1,
			"A structure closes its brace on a line of its own.")
	}
}

// Name_Column is the column the types of one run of fields align at, which is the width of the
// widest run of names in it. Zero names no run of fields.
type Name_Column Width

// Name_Column_Invariants states every column a run of fields aligns its types at.
func Name_Column_Invariants(value Name_Column, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), FORM_SIZE_MINIMUM, WIDTH_MAXIMUM).
		Ensure()
}

// Tag_Column is the column the tags of one run of fields align at, which is the width of the
// widest type in it. Zero names a run that carries no tag.
type Tag_Column Width

// Tag_Column_Invariants states every column a run of fields aligns its tags at.
func Tag_Column_Invariants(value Tag_Column, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), FORM_SIZE_MINIMUM, WIDTH_MAXIMUM).
		Ensure()
}

// Run_Widths keeps the two columns one run of fields aligns at.
type Run_Widths struct {
	// Names is the column the types align at.
	Names Name_Column
	// Tags is the column the tags align at.
	Tags Tag_Column
}

// Run_Widths_Invariants composes both columns of one run.
func Run_Widths_Invariants(value Run_Widths, namespace aver.Namespace) {
	Name_Column_Invariants(value.Names, namespace)
	Tag_Column_Invariants(value.Tags, namespace)
}

// Writes one member of a struct: a field, a comment, or an empty line, and returns the run of
// fields the member behind it stands in. A comment and an empty line each close the run of fields
// that stood before them, thus the run behind one aligns on a width of its own.
func print_field(
	subject Printer_Handle, tree ast.Syntax_Tree_Handle, run Run_Widths,
) (next Run_Widths) {
	defer func() { Run_Widths_Invariants(next, "print_field.next") }()
	Printer_Handle_Invariants(subject, "print_field.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "print_field.tree")
	Run_Widths_Invariants(run, "print_field.run")
	closed := Run_Widths{Names: 0, Tags: 0}
	kind := node_kind(subject, tree)
	// A structure holds fields, notes, and empty lines alone, thus every part it holds
	// states one of the three.
	if bool(writes_canonical(subject.Flags.Values)) {
		Field_Part_Invariants(field_part(subject, tree), "print_field.parted")
	}
	if kind == ast.NODE_BLANK {
		if bool(stands_against(subject, tree)) {
			return closed
		}
		if bool(stands_before_brace(subject, tree)) {
			return closed
		}
		// An empty line the scanner counted behind the brace stands behind the struct, thus
		// the print owes it to the line the declaration closes and not to this one.
		if bool(closes_block(subject, tree)) {
			flag_store(subject.Flags.Values, FLAG_BLANK, true)
			return closed
		}
		close_line(subject, tree)
		return closed
	}
	if kind == ast.NODE_COMMENT {
		emit_indent(subject)
		emit_token(subject, tree)
		close_line(subject, tree)
		return closed
	}
	// A structure holds fields, notes, and empty lines alone, thus a part of any other kind
	// stands in no line the print writes.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(kind == ast.NODE_FIELD, "A part of one structure states a field.")
	}
	if kind != ast.NODE_FIELD {
		return run
	}
	// The print writes one field of a structure here, thus the count of them reads what the
	// structures of the file state once the print closed.
	subject.Counts.Values[COUNT_FIELD] = subject.Counts.Values[COUNT_FIELD] + 1
	// A field that runs over lines states its type where that type falls, thus it closes the
	// run ahead of it and stands in no column at all.
	spans := field_spans(subject, tree)
	if bool(spans) {
		run = Run_Widths{Names: 0, Tags: 0}
	}
	if run.Names == 0 {
		if !bool(spans) {
			run = take_run_width(subject, tree)
		}
	}
	if bool(print_field_parts(subject, tree, run)) {
		return closed
	}
	return run
}

// FIELD_PART_NAMED names a field, which states the names it binds and the type behind them.
const FIELD_PART_NAMED = 0

// FIELD_PART_NOTE names a note the author left inside the structure.
const FIELD_PART_NOTE = 1

// FIELD_PART_LINE names an empty line the author left inside the structure.
const FIELD_PART_LINE = 2

// Field_Part names which of the three parts one structure holds.
type Field_Part uint8

// Field_Part_Invariants states every part one structure holds.
func Field_Part_Invariants(value Field_Part, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(uint8(value), FIELD_PART_NAMED, FIELD_PART_NOTE, FIELD_PART_LINE).
		Ensure()
}

// Reads which of the three parts of a structure the walk stands on.
func field_part(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (parted Field_Part) {
	defer func() { Field_Part_Invariants(parted, "field_part.parted") }()
	Printer_Handle_Invariants(subject, "field_part.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "field_part.tree")
	if node_kind(subject, tree) == ast.NODE_BLANK {
		return FIELD_PART_LINE
	}
	if node_kind(subject, tree) == ast.NODE_COMMENT {
		return FIELD_PART_NOTE
	}
	return FIELD_PART_NAMED
}

// Reads the columns of the run of fields the walk opens. A field that states no name of its own
// closes the run, thus it aligns nothing and nothing aligns on it.
func take_run_width(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (run Run_Widths) {
	defer func() { Run_Widths_Invariants(run, "take_run_width.run") }()
	Printer_Handle_Invariants(subject, "take_run_width.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "take_run_width.tree")
	stood := depth_load(subject.Counts.Values)
	held := subject.Nodes.Values[stood]
	widest := Width(0)
	types := Width(0)
	tagged := Run_Tagged(false)
	open := Boolean(true)
	for bool(open) {
		if node_kind(subject, tree) != ast.NODE_FIELD {
			break
		}
		measured, named := field_width(subject, tree)
		width := Width(measured)
		if !bool(named) {
			break
		}
		// A field that runs over lines stands in no column, thus the run closes ahead of
		// it and its own width states nothing.
		if bool(field_spans(subject, tree)) {
			break
		}
		if width > widest {
			widest = width
		}
		// Every field of one run binds a name, thus the column the run opens its types at
		// stands past the widest of those names and never at no column at all.
		aver.Always(widest >= NAME_WIDTH_MINIMUM,
			"A run of fields opens its types past the widest name it holds.")
		// A tag stands in a column of its own behind the types, thus the run answers for
		// the widest type as well as for the widest run of names.
		typed, states := type_width(subject, tree)
		one := Width(typed)
		if one > types {
			types = one
		}
		tagged = Run_Tagged(bool(tagged) || bool(states))
		// A field that carries a comment or an empty line behind it closes the run, thus
		// the run behind that line aligns on a width of its own.
		if bool(field_breaks(subject, tree)) {
			break
		}
		open = advance(subject, tree)
	}
	subject.Nodes.Values[stood] = held
	// The read steps along the run with a step that holds the slot it stood on, thus it
	// hands back the walk it opened.
	aver.Always(depth_load(subject.Counts.Values) == stood,
		"A read of one run of fields stands at the depth it opened at.")
	run = Run_Widths{Names: Name_Column(min(widest, WIDTH_MAXIMUM)), Tags: 0}
	if bool(tagged) {
		run.Tags = Tag_Column(min(types, WIDTH_MAXIMUM))
	}
	Run_Tagged_Invariants(tagged, "take_run_width.tagged")
	return run
}

// Run_Tagged names a run of fields that carries a tag behind one type of it.
type Run_Tagged Boolean

// Run_Tagged_Invariants states both stands of the run.
func Run_Tagged_Invariants(value Run_Tagged, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The run carries a tag behind one type.").
		Ensure()
}

// Reads how wide the type of one field writes and whether the field carries a tag behind it.
func type_width(
	subject Printer_Handle, tree ast.Syntax_Tree_Handle,
) (width Measured_Width, tagged Boolean) {
	defer func() {
		Measured_Width_Invariants(width, "type_width.width")
		Boolean_Invariants(tagged, "type_width.tagged")
	}()
	Printer_Handle_Invariants(subject, "type_width.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "type_width.tree")
	if !bool(descend_quiet(subject, tree)) {
		return 0, false
	}
	open := Boolean(true)
	for bool(open) {
		if node_kind(subject, tree) != ast.NODE_FIELD_NAME {
			break
		}
		open = advance_quiet(subject, tree)
	}
	if !bool(open) {
		ascend(subject)
		return 0, false
	}
	width = measure_expression(subject, tree)
	tagged = advance_quiet(subject, tree)
	ascend(subject)
	// A field that states a type spells that type, thus the run of tags behind the types
	// aligns on a width of one column at least. A tree the parse refused holds a field whose
	// type spells nothing at all.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(width > 0, "A field that states a type states a width of its own.")
	}
	return width, tagged
}

// Reports whether the field the walk stands on carries a comment or an empty line behind it.
func field_breaks(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "field_breaks.yes") }()
	Printer_Handle_Invariants(subject, "field_breaks.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "field_breaks.tree")
	stood := written_load(subject.Counts.Values)
	defer func() {
		aver.Always(written_load(subject.Counts.Values) == stood,
			"A read of the line one field closes its run on writes no byte.")
	}()
	if !bool(descend(subject, tree)) {
		return false
	}
	closed := Field_Close(FIELD_CLOSE_NONE)
	open := Boolean(true)
	for bool(open) {
		kind := node_kind(subject, tree)
		if kind == ast.NODE_BLANK {
			closed = FIELD_CLOSE_LINE
		}
		open = advance(subject, tree)
	}
	ascend(subject)
	// A run of fields closes at an empty line the author left inside the field, and a note
	// the field carries closes none: a note standing on a line of its own stands in a field
	// that runs over lines, which closes the run where the print reads its width.
	Field_Close_Invariants(closed, "field_breaks.closed")
	return Boolean(closed != FIELD_CLOSE_NONE)
}

// Reads the width of the names one field states, and reports whether it states one at all.
func field_width(
	subject Printer_Handle, tree ast.Syntax_Tree_Handle,
) (width Measured_Width, named Boolean) {
	defer func() {
		Measured_Width_Invariants(width, "field_width.width")
		Boolean_Invariants(named, "field_width.named")
	}()
	Printer_Handle_Invariants(subject, "field_width.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "field_width.tree")
	if !bool(descend(subject, tree)) {
		return 0, false
	}
	held := Boolean(node_kind(subject, tree) == ast.NODE_FIELD_NAME)
	width = names_width(subject, tree)
	ascend(subject)
	if !bool(held) {
		return 0, false
	}
	// A field that binds a name spells that name, thus the run it stands in aligns on a
	// width of one column at least.
	aver.Always(width >= NAME_WIDTH_MINIMUM,
		"A field that binds a name states a width of its own.")
	return width, true
}

// Reads how wide the names of one field write: every name, and a comma and a space between two of
// them. The run stops at the first part that names no field.
func names_width(
	subject Printer_Handle, tree ast.Syntax_Tree_Handle,
) (width Measured_Width) {
	defer func() { Measured_Width_Invariants(width, "names_width.width") }()
	Printer_Handle_Invariants(subject, "names_width.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "names_width.tree")
	stood := depth_load(subject.Counts.Values)
	held := subject.Nodes.Values[stood]
	total := 0
	named := Field_Named(false)
	open := Boolean(true)
	for bool(open) {
		// A note the author left between the names stands on the line the field closes
		// and spends no column of the names, thus the read steps over it.
		if bool(trivia_kind(subject, tree)) {
			open = advance(subject, tree)
			continue
		}
		if node_kind(subject, tree) != ast.NODE_FIELD_NAME {
			break
		}
		// A comma and a space stand between two names of one field.
		if bool(named) {
			total = total + NAME_GAP_WIDTH
		}
		total = total + int(name_width(subject, tree))
		named = true
		open = advance(subject, tree)
	}
	subject.Nodes.Values[depth_load(subject.Counts.Values)] = held
	// The read steps along the names with a step that holds the slot it stood on, thus it
	// hands back the walk it opened.
	aver.Always(depth_load(subject.Counts.Values) == stood,
		"A read of the names one field binds holds the slot it opened.")
	return Measured_Width(min(total, WIDTH_MAXIMUM))
}

// Field_Named names a field that already wrote a name of its own.
type Field_Named Boolean

// Field_Named_Invariants states both stands of the field.
func Field_Named_Invariants(value Field_Named, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The field already wrote a name.").
		Ensure()
}

// Opens one name of a field: the comma that closes the name before it, where one stands, and the
// space that holds the two apart.
func open_field_name(subject Printer_Handle, named Field_Named) {
	Printer_Handle_Invariants(subject, "open_field_name.subject")
	Field_Named_Invariants(named, "open_field_name.named")
	if !bool(named) {
		return
	}
	stood := take_line_width(subject.Counts.Values)
	emit_comma(subject)
	emit_space(subject)
	// The measure of a run of names counts this gap for every name behind the first, thus
	// the bytes the print writes here and the width that measure reads state one gap.
	if bool(writes_canonical(subject.Flags.Values)) {
		spent := int(take_line_width(subject.Counts.Values)) - int(stood)
		aver.Always(spent == NAME_GAP_WIDTH,
			"The gap between two names of one field spends the width it states.")
	}
}

// NAME_WIDTH_MINIMUM is the narrowest name one field states, which spells one byte.
const NAME_WIDTH_MINIMUM = 1

// NAME_GAP_WIDTH is what stands between two names of one field: the comma and the space behind it.
const NAME_GAP_WIDTH = 2

// Name_Width is the width one name of a field writes.
type Name_Width int32

// Name_Width_Invariants states every width one name writes.
func Name_Width_Invariants(value Name_Width, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), NAME_WIDTH_MINIMUM, WIDTH_MAXIMUM).
		Ensure()
}

// Reads the width the name the walk stands on writes, which is the bytes its token spells.
func name_width(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (width Name_Width) {
	defer func() { Name_Width_Invariants(width, "name_width.width") }()
	Printer_Handle_Invariants(subject, "name_width.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "name_width.tree")
	node := ast.Syntax_Node_At(tree, subject.Nodes.Values[depth_load(subject.Counts.Values)])
	spelled := int(ast.Syntax_Token_At(tree, node.Token).Size)
	return Name_Width(min(max(spelled, NAME_WIDTH_MINIMUM), WIDTH_MAXIMUM))
}

// Writes the name of one field, the spaces that align the type behind it, and that type, and
// reports whether the field closes the run it stands in.
func print_field_parts(
	subject Printer_Handle, tree ast.Syntax_Tree_Handle, run Run_Widths,
) (closes Boolean) {
	defer func() { Boolean_Invariants(closes, "print_field_parts.closes") }()
	Printer_Handle_Invariants(subject, "print_field_parts.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "print_field_parts.tree")
	Run_Widths_Invariants(run, "print_field_parts.run")
	// A field that states no part at all writes nothing, thus the line it would open holds
	// no tab of its own.
	if !bool(descend_part(subject, tree)) {
		return false
	}
	emit_indent(subject)
	if node_kind(subject, tree) != ast.NODE_FIELD_NAME {
		print_bare_field(subject, tree)
		return true
	}
	width := names_width(subject, tree)
	print_field_names(subject, tree)
	// A field that states a part behind its names stands on that part, and one that states
	// none stands on the last name it binds.
	open := Boolean(node_kind(subject, tree) != ast.NODE_FIELD_NAME)
	if bool(open) {
		// A field that states no type pads for none, thus the line it writes closes on
		// the last name it holds.
		emit_column(subject, Part_Width(width), Column_Width(run.Names))
		// Every type of one run opens at the column the run states, and a run of names
		// past that column opens one space behind the names it wrote. A run as wide as
		// the widest line the print reads saturates that read, thus it states nothing.
		opened := int(indent_load(subject.Counts.Values))*INDENT_WIDTH +
			max(int(run.Names), int(width)) + 1
		if opened < WIDTH_MAXIMUM {
			if bool(writes_canonical(subject.Flags.Values)) {
				aver.Always(int(take_line_width(subject.Counts.Values)) == opened,
					"A type of one run opens at its column.")
			}
		}
		span := measure_expression(subject, tree)
		print_expression(subject, tree)
		open = advance(subject, tree)
		// A tag stands in the column the widest type of the run opens, thus every tag of
		// one run reads as one column rather than as a ragged edge. The node behind the
		// type is the tag where the author wrote one, and trivia where none stands.
		open = enter_tag(subject, tree, open)
		if bool(open) {
			if !bool(trivia_kind(subject, tree)) {
				emit_column(subject, Part_Width(span), Column_Width(run.Tags))
				held := opened + max(int(run.Tags), int(span)) + 1
				if held < WIDTH_MAXIMUM {
					spent := int(take_line_width(subject.Counts.Values))
					aver.Always(spent == held,
						"A tag of one run opens at its column.")
				}
				emit_token(subject, tree)
				open = advance(subject, tree)
			}
		}
	}
	open = take_trailing_note(subject, tree, open)
	close_line(subject, tree)
	broken := open
	drain_trivia(subject, tree, open)
	ascend(subject)
	return broken
}

// Writes one field that binds no name of its own. Such a field still carries the notes and the
// empty lines the author wrote behind it, thus the walk steps over them rather than dropping them.
func print_bare_field(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "print_bare_field.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "print_bare_field.tree")
	// A field that binds no name of its own states the type it embeds, thus the walk stands
	// on a part the print writes rather than on the trivia standing behind it.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(!bool(trivia_kind(subject, tree)),
			"A field that binds no name states a type of its own.")
	}
	print_expression(subject, tree)
	stand := advance(subject, tree)
	// A field that binds no name carries its tag behind the type it embeds, and the node
	// behind that type is the tag where the author wrote one and trivia where none stands.
	stand = enter_tag(subject, tree, stand)
	if bool(stand) {
		if !bool(trivia_kind(subject, tree)) {
			emit_space(subject)
			emit_token(subject, tree)
			stand = advance(subject, tree)
		}
	}
	stand = take_trailing_note(subject, tree, stand)
	close_line(subject, tree)
	drain_trivia(subject, tree, stand)
	ascend(subject)
}

// CONSTRAINT_PART_TERM names one term the constraint admits.
const CONSTRAINT_PART_TERM = 0

// CONSTRAINT_PART_NOTE names a note the author left inside the constraint.
const CONSTRAINT_PART_NOTE = 1

// CONSTRAINT_PART_LINE names an empty line the author left inside the constraint.
const CONSTRAINT_PART_LINE = 2

// Constraint_Part names which of the three parts one constraint holds.
type Constraint_Part uint8

// Constraint_Part_Invariants states every part one constraint holds.
func Constraint_Part_Invariants(value Constraint_Part, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(uint8(value), CONSTRAINT_PART_TERM, CONSTRAINT_PART_NOTE,
			CONSTRAINT_PART_LINE).
		Ensure()
}

// Reads which of the three parts of a constraint the walk stands on.
func constraint_part(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (parted Constraint_Part) {
	defer func() { Constraint_Part_Invariants(parted, "constraint_part.parted") }()
	Printer_Handle_Invariants(subject, "constraint_part.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "constraint_part.tree")
	if node_kind(subject, tree) == ast.NODE_BLANK {
		return CONSTRAINT_PART_LINE
	}
	if node_kind(subject, tree) == ast.NODE_COMMENT {
		return CONSTRAINT_PART_NOTE
	}
	return CONSTRAINT_PART_TERM
}

// Writes one constraint: the word, the terms one tab deeper, and the brace that closes it.
func print_constraint(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "print_constraint.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "print_constraint.tree")
	// A constraint holds the terms it admits between braces, thus the walk stands on one
	// where the print writes one.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(node_kind(subject, tree) == ast.NODE_INTERFACE_TYPE,
			"A constraint the print writes stands on a constraint of the tree.")
	}
	emit_word(subject, WORD_INTERFACE)
	if bool(holds_no_part(subject, tree)) {
		emit_byte(subject, '{')
		emit_byte(subject, '}')
		close_empty_list(subject, tree)
		return
	}
	stood := descend(subject, tree)
	aver.Always(bool(stood), "A constraint that holds a term opens on that term.")
	emit_space(subject)
	emit_byte(subject, '{')
	// A note the author left on the line the brace opens keeps that line.
	stood = open_brace_note(subject, tree)
	close_line(subject, tree)
	indent_deepen(subject, indent_load(subject.Counts.Values))
	held_tight := flag_load(subject.Flags.Values, FLAG_TIGHT)
	flag_store(subject.Flags.Values, FLAG_TIGHT, true)
	open := stood
	for bool(open) {
		// A note the author left on the line the brace closes keeps that line, thus
		// the constraint holds it and the line that brace closes carries it.
		if bool(trails_close(subject, tree)) {
			print_trivia(subject, tree)
			open = advance(subject, tree)
			continue
		}
		print_element(subject, tree)
		open = advance(subject, tree)
	}
	flag_store(subject.Flags.Values, FLAG_TIGHT, held_tight)
	ascend(subject)
	indent_shallow(subject, indent_load(subject.Counts.Values))
	emit_indent(subject)
	closes_body(subject)
	emit_byte(subject, '}')
	// The brace that closes a constraint stands on a line of its own, at the tabs the
	// constraint opened at and behind nothing else.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(int(take_line_width(subject.Counts.Values)) ==
			int(indent_load(subject.Counts.Values))*INDENT_WIDTH+1,
			"A constraint closes its brace on a line of its own.")
	}
}

// Writes one element of a constraint on a line of its own.
func print_element(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "print_element.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "print_element.tree")
	kind := node_kind(subject, tree)
	// A constraint holds terms, notes, and empty lines alone, thus every part it holds
	// states one of the three.
	if bool(writes_canonical(subject.Flags.Values)) {
		Constraint_Part_Invariants(constraint_part(subject, tree), "print_element.parted")
	}
	if kind == ast.NODE_BLANK {
		if bool(stands_before_brace(subject, tree)) {
			return
		}
		if bool(closes_block(subject, tree)) {
			flag_store(subject.Flags.Values, FLAG_BLANK, true)
			return
		}
		close_line(subject, tree)
		return
	}
	emit_indent(subject)
	if kind == ast.NODE_COMMENT {
		emit_token(subject, tree)
		close_line(subject, tree)
		return
	}
	stood := written_load(subject.Counts.Values)
	print_expression(subject, tree)
	// An element states one term the constraint admits, thus the print writes bytes for it.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(written_load(subject.Counts.Values) > stood,
			"An element of one constraint writes bytes of its own.")
	}
	close_line(subject, tree)
}

// Reports whether the source states a line feed ahead of the node the walk stands on. The
// canonical form keeps the line the author broke, thus a print reads the break out of the source
// rather than inventing one of its own.
func breaks_before(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "breaks_before.yes") }()
	Printer_Handle_Invariants(subject, "breaks_before.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "breaks_before.tree")
	written := written_load(subject.Counts.Values)
	defer func() {
		aver.Always(written_load(subject.Counts.Values) == written,
			"A read of the break one author left writes no byte of its own.")
	}()
	// A measure asks how wide the form reads on one line, thus it reads no break at all.
	if bool(flag_load(subject.Flags.Values, FLAG_FLAT)) {
		return false
	}
	held_depth := depth_load(subject.Counts.Values)
	enter_leftmost(subject, tree)
	leftmost := subject.Nodes.Values[depth_load(subject.Counts.Values)]
	depth_store(subject.Counts.Values, held_depth)
	found := Count(ast.Syntax_Node_At(tree, leftmost).Token)
	position := found
	if position == 0 {
		return false
	}
	behind := ast.Syntax_Token_At(tree, ast.Token_Index(position-1))
	ahead := ast.Syntax_Token_At(tree, ast.Token_Index(position))
	// A line feed between the two bytes states a break the author wrote.
	from_offset, to_offset := int(int(behind.Offset)+int(behind.Size)), int(ahead.Offset)
	// No two tokens of the run share a byte of the source, thus the token the form opens on
	// stands behind the end of the token standing ahead of it.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(to_offset >= from_offset,
			"A form opens behind the end of the token ahead of it.")
	}
	for place_offset := from_offset; place_offset < to_offset; place_offset++ {
		if subject.Source.Values[place_offset] == '\n' {
			return true
		}
	}
	return false
}

// Moves the walk from the form it stands on down to the node that names the final token the form
// spans. The last child of a node spans the tokens behind every other child, and a child that
// names a token ahead of its parent stands wholly ahead of that token, thus the walk steps into
// each last child that names a later token.
func enter_rightmost(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "enter_rightmost.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "enter_rightmost.tree")
	for range DEPTH_MAXIMUM {
		depth := depth_load(subject.Counts.Values)
		node := ast.Syntax_Node_At(tree, subject.Nodes.Values[depth])
		last := ast.Index(node.First_Child)
		if last == ast.INDEX_ABSENT {
			return
		}
		// An empty line and a note the author left behind the last part of a form
		// stand behind that form, thus the part it closes on is the last one that
		// states a byte of the form itself.
		held := last
		for ast.Index(ast.Syntax_Node_At(tree, last).Next) != ast.INDEX_ABSENT {
			last = ast.Index(ast.Syntax_Node_At(tree, last).Next)
			kind := ast.Syntax_Node_At(tree, last).Kind
			if kind == ast.NODE_BLANK {
				continue
			}
			if kind == ast.NODE_COMMENT {
				continue
			}
			held = last
		}
		last = held
		if ast.Syntax_Node_At(tree, last).Token <= node.Token {
			return
		}
		if int(depth)+1 >= DEPTH_MAXIMUM {
			return
		}
		// The walk steps down into the chain that hangs under the node it stands on.
		aver.Always(ast.Index(ast.Syntax_Node_At(tree, last).Parent) ==
			subject.Nodes.Values[depth],
			"A step down to the last part stands on a child of the node it left.")
		subject.Nodes.Values[depth+1] = last
		depth_store(subject.Counts.Values, depth+1)
	}
}

// Reports whether the bracket that closes the form the walk stands on stands on a line of its
// own. A bracket the author left behind the final part stays there, thus the print states the
// break the author stated and never one of its own.
func closes_line(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "closes_line.yes") }()
	Printer_Handle_Invariants(subject, "closes_line.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "closes_line.tree")
	if bool(flag_load(subject.Flags.Values, FLAG_FLAT)) {
		return false
	}
	position := closer_position(subject, tree)
	if position == 0 {
		return false
	}
	behind := ast.Syntax_Token_At(tree, ast.Token_Index(position-1))
	ahead := ast.Syntax_Token_At(tree, ast.Token_Index(position))
	// The read of a closer hands back where a bracket closes the form, thus the token the
	// position names closes one.
	aver.Always(bool(closes_bracket(token.Scanned_Kind(ahead.Kind))),
		"A closer the read hands back stands on a bracket that closes a form.")
	// A line feed between the two bytes states a break the author wrote.
	from_offset, to_offset := int(int(behind.Offset)+int(behind.Size)), int(ahead.Offset)
	for place_offset := from_offset; place_offset < to_offset; place_offset++ {
		if subject.Source.Values[place_offset] == '\n' {
			return true
		}
	}
	return false
}

// CLOSER_HOLE_NAME and CLOSER_HOLE_MARK are the positions the package clause spends on its name
// and on the semicolon behind it, which no bracket stands at.
const CLOSER_HOLE_NAME = 1

// CLOSER_HOLE_MARK is the position of the semicolon the package clause closes on.
const CLOSER_HOLE_MARK = 2

// CLOSER_POSITION_MAXIMUM is the last position a bracket stands at. A file closes on the
// semicolon its end inserts behind the final bracket and on its end token.
const CLOSER_POSITION_MAXIMUM = ast.TOKEN_INDEX_MAXIMUM - 2

// Closer_Position is the run position of the bracket that closes one form, or zero where the run
// holds no such bracket.
type Closer_Position ast.Token_Index

// Closer_Position_Invariants states every position a closing bracket stands at.
func Closer_Position_Invariants(value Closer_Position, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Holed_Int32(
			int32(value), POSITION_MINIMUM, CLOSER_POSITION_MAXIMUM,
			CLOSER_HOLE_NAME, CLOSER_HOLE_MARK, CLOSER_HOLE_MARK, CLOSER_HOLE_MARK,
		).
		Ensure()
}

// Names the run position of the bracket that closes the form the walk stands on. The node names
// the bracket its form opens with, thus a count over the brackets behind that one finds the
// bracket that closes it. Zero names a form the run holds no last bracket for.
func closer_position(
	subject Printer_Handle, tree ast.Syntax_Tree_Handle,
) (closer Closer_Position) {
	defer func() { Closer_Position_Invariants(closer, "closer_position.closer") }()
	Printer_Handle_Invariants(subject, "closer_position.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "closer_position.tree")
	stood := written_load(subject.Counts.Values)
	defer func() {
		aver.Always(written_load(subject.Counts.Values) == stood,
			"A read of the bracket one form closes on writes no byte of its own.")
	}()
	index := subject.Nodes.Values[depth_load(subject.Counts.Values)]
	counts := subject.Counts.Values
	// Every form of one file closes on one bracket, thus a read of the form the print
	// already read hands back what that read found.
	if counts[COUNT_CLOSER_NODE] == Count(index)+1 {
		return Closer_Position(counts[COUNT_CLOSER_PLACE])
	}
	node := ast.Syntax_Node_At(tree, index)
	depth := 0
	place := 0
	run := tree.Tokens.Values
	for offset := int(node.Token); offset < ast.TOKEN_COUNT_MAXIMUM; offset++ {
		kind := run[offset].Kind
		if kind == token.KIND_END_OF_FILE {
			break
		}
		// The walk reads every token of a body once for each block around it, thus the
		// bracket test stays inline rather than paying a checked step for each token.
		switch kind {
		case token.KIND_PARENTHESIS_LEFT, token.KIND_BRACKET_LEFT, token.KIND_BRACE_LEFT:
			depth = depth + 1
		case token.KIND_PARENTHESIS_RIGHT, token.KIND_BRACKET_RIGHT, token.KIND_BRACE_RIGHT:
			depth = depth - 1
			if depth <= 0 {
				place = offset
			}
		}
		if place != 0 {
			break
		}
	}
	// The read opens at the token the node names, thus the bracket it closes on stands
	// behind that token. A tree the parse refused holds a form whose own token closes it.
	if place != 0 {
		if bool(writes_canonical(subject.Flags.Values)) {
			aver.Always(place > int(node.Token),
				"A form closes on a bracket behind its token.")
		}
	}
	counts[COUNT_CLOSER_NODE] = Count(index) + 1
	counts[COUNT_CLOSER_PLACE] = Count(place)
	return Closer_Position(place)
}

// Opens the line one part of a broken form stands on: the comma that closes the part before it,
// the line feed, and the tabs one level deeper than the form.
func emit_break(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "emit_break.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "emit_break.tree")
	emit_comma(subject)
	close_line(subject, tree)
	emit_indent(subject)
	// A part the break opens stands behind the tabs of its line and behind nothing else.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(int(take_line_width(subject.Counts.Values)) ==
			int(indent_load(subject.Counts.Values))*INDENT_WIDTH,
			"A break opens a line holding the tabs the walk stands under.")
	}
}

// Statement_Written names a body that already wrote a statement of its own.
type Statement_Written Boolean

// Statement_Written_Invariants states both stands of the body.
func Statement_Written_Invariants(value Statement_Written, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The body already wrote a statement.").
		Ensure()
}

// Opens the statement one inline body writes: the semicolon that closes the statement before it,
// where one stands, and the space that holds it off the brace or that semicolon.
func open_inline_statement(subject Printer_Handle, written Statement_Written) {
	Printer_Handle_Invariants(subject, "open_inline_statement.subject")
	Statement_Written_Invariants(written, "open_inline_statement.written")
	stood := take_line_width(subject.Counts.Values)
	if bool(written) {
		emit_byte(subject, ';')
	}
	emit_space(subject)
	// A statement behind another on one line stands behind the semicolon that closes it and
	// the space behind that, and the first stands behind the space alone.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(int(take_line_width(subject.Counts.Values)) > int(stood),
			"A statement on one line stands behind what holds it apart.")
	}
}

// Writes one block whose statements stand on the line its brace opens.
func print_inline_block(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "print_inline_block.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "print_inline_block.tree")
	// The block the body mark names stands written here, thus the mark closes with it and
	// the block behind it reads the lines the author wrote rather than this mark.
	flag_store(subject.Flags.Values, FLAG_BODY, false)
	stood := written_load(subject.Counts.Values)
	emit_byte(subject, '{')
	open := descend(subject, tree)
	entered := open
	written := Statement_Written(false)
	for bool(open) {
		if bool(trivia_kind(subject, tree)) {
			print_trivia(subject, tree)
			open = advance(subject, tree)
			continue
		}
		open_inline_statement(subject, written)
		// The print writes one statement of a body here, thus the count of them reads
		// this one the way it reads the statements a body over lines states.
		subject.Counts.Values[COUNT_STATEMENT] =
			subject.Counts.Values[COUNT_STATEMENT] + 1
		print_statement_form(subject, tree)
		written = true
		open = advance(subject, tree)
	}
	if bool(entered) {
		ascend(subject)
	}
	// A statement that closed the line it stood on leaves the brace the line it opens, thus
	// the brace takes the tabs of that line rather than a space behind a byte.
	if bool(written) {
		opening := flag_load(subject.Flags.Values, FLAG_LINE)
		if bool(opening) {
			emit_indent(subject)
		}
		if !bool(opening) {
			emit_space(subject)
		}
	}
	emit_byte(subject, '}')
	// A block the author stated on one line writes the braces it holds its statements
	// between, thus the form holds both however many statements stand inside them.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(written_load(subject.Counts.Values) > stood+1,
			"A block on one line writes the braces it holds its statements between.")
	}
}

// Indents the parts of a form that the author broke behind its first part. A form broken at its
// head takes its indent when it opens, thus only a form broken later owes one here.
func open_break(subject Printer_Handle, here Boolean, broken Boolean) {
	Printer_Handle_Invariants(subject, "open_break.subject")
	Boolean_Invariants(here, "open_break.here")
	Boolean_Invariants(broken, "open_break.broken")
	if !bool(here) {
		return
	}
	if bool(broken) {
		return
	}
	indent_deepen(subject, indent_load(subject.Counts.Values))
}

// Reads the nest the parts inside the brackets of one form stand at: one deeper than the form
// where the form deepens its parts, and the nest of the form itself where it does not.
func part_inner_nest(
	subject Printer_Handle, tree ast.Syntax_Tree_Handle, wrap Wrap, held Nest,
) (inner Nest) {
	defer func() { Nest_Invariants(inner, "part_inner_nest.inner") }()
	Printer_Handle_Invariants(subject, "part_inner_nest.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "part_inner_nest.tree")
	Wrap_Invariants(wrap, "part_inner_nest.wrap")
	Nest_Invariants(held, "part_inner_nest.held")
	if !bool(deepens_parts(subject, tree, wrap)) {
		return held
	}
	return Nest(nest_inner(subject, held))
}

// Reports whether the parts inside the brackets of one form stand one nest deeper than the form.
// An index always does, and a call of one argument states that argument as plainly as a statement
// does, thus only a call of two or more closes up the signs its arguments hold.
func deepens_parts(subject Printer_Handle, tree ast.Syntax_Tree_Handle, wrap Wrap) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "deepens_parts.yes") }()
	Printer_Handle_Invariants(subject, "deepens_parts.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "deepens_parts.tree")
	Wrap_Invariants(wrap, "deepens_parts.wrap")
	if wrap == WRAP_INDEX {
		return true
	}
	return holds_three_parts(subject, tree)
}

// PART_COUNT_NONE names a form that holds no part at all.
const PART_COUNT_NONE = 0

// PART_COUNT_ONE names a form that holds one part.
const PART_COUNT_ONE = 1

// PART_COUNT_TWO names a form that holds two.
const PART_COUNT_TWO = 2

// PART_COUNT_THREE names a form that holds three parts or more. A call holds its callee as one
// part, thus three parts state two arguments, and the print reads no further.
const PART_COUNT_THREE = 3

// Part_Count is how many parts one form holds, counted up to three.
type Part_Count uint8

// Part_Count_Invariants states every count of parts one read states.
func Part_Count_Invariants(value Part_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_4_Uint8(uint8(value), PART_COUNT_NONE, PART_COUNT_ONE, PART_COUNT_TWO,
			PART_COUNT_THREE).
		Ensure()
}

// Reports whether one child of a clause opens the branch its condition fails into. Only an if
// statement holds a second branch, and only a block or a further if statement states one.
func opens_branch(
	subject Printer_Handle, tree ast.Syntax_Tree_Handle, bodied Bodied, conditional Boolean,
) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "opens_branch.yes") }()
	Printer_Handle_Invariants(subject, "opens_branch.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "opens_branch.tree")
	Bodied_Invariants(bodied, "opens_branch.bodied")
	Boolean_Invariants(conditional, "opens_branch.conditional")
	held := Boolean(bodied)
	if !bool(conditional) {
		return false
	}
	if !bool(held) {
		return false
	}
	child := node_kind(subject, tree)
	return Boolean(child == ast.NODE_BLOCK || child == ast.NODE_IF)
}

// Reports whether the node the walk stands on states one case of a switch or a select.
func opens_case(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "opens_case.yes") }()
	Printer_Handle_Invariants(subject, "opens_case.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "opens_case.tree")
	kind := node_kind(subject, tree)
	return Boolean(kind == ast.NODE_CASE || kind == ast.NODE_DEFAULT)
}

// Bodied names a clause that already wrote a body of its own.
type Bodied Boolean

// Bodied_Invariants states both stands of the clause.
func Bodied_Invariants(value Bodied, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The clause already wrote a body.").
		Ensure()
}

// Cased names a clause that already wrote a case.
type Cased Boolean

// Cased_Invariants states both stands of the clause.
func Cased_Invariants(value Cased, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The clause already wrote a case.").
		Ensure()
}

// Clause_Headed names a clause that already wrote a header.
type Clause_Headed Boolean

// Clause_Headed_Invariants states both stands of the clause.
func Clause_Headed_Invariants(value Clause_Headed, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The clause already wrote a header.").
		Ensure()
}

// Reports whether the clause the walk stands on holds one case alone, which is the body that
// closes up against the braces that hold it.
func holds_one_case(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "holds_one_case.yes") }()
	Printer_Handle_Invariants(subject, "holds_one_case.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "holds_one_case.tree")
	stood := written_load(subject.Counts.Values)
	defer func() {
		aver.Always(written_load(subject.Counts.Values) == stood,
			"A read of the case one switch holds writes no byte of its own.")
	}()
	held := subject.Nodes.Values[depth_load(subject.Counts.Values)]
	depth := depth_load(subject.Counts.Values)
	cases := 0
	open := descend_quiet(subject, tree)
	for bool(open) {
		if bool(opens_case(subject, tree)) {
			cases = cases + 1
		}
		open = advance_quiet(subject, tree)
	}
	depth_store(subject.Counts.Values, depth)
	subject.Nodes.Values[depth] = held
	return Boolean(cases == 1)
}

// Writes the values one case stands for. The tree hangs the whole run off one node, thus the
// print walks that node rather than the case itself.
func print_case_values(subject Printer_Handle, tree ast.Syntax_Tree_Handle, flat Boolean) {
	Printer_Handle_Invariants(subject, "print_case_values.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "print_case_values.tree")
	Boolean_Invariants(flat, "print_case_values.flat")
	// A select states a whole statement where a switch states a value, thus the print writes
	// the form of the child rather than an expression the child may not be.
	if node_kind(subject, tree) != ast.NODE_EXPRESSION_STATEMENT {
		emit_space(subject)
		print_statement_form(subject, tree)
		return
	}
	// A case states the values it stands for as one statement of expressions, thus the print
	// writes that statement here the way a body writes one of its own.
	counts_binding(subject, tree)
	open := descend_part(subject, tree)
	entered := open
	written := Value_Written(false)
	broken := Boolean(false)
	for bool(open) {
		Value_Written_Invariants(written, "print_case_values.written")
		if bool(written) {
			emit_comma(subject)
		}
		here := Boolean(false)
		if bool(written) {
			here = breaks_before(subject, tree)
		}
		if bool(flat) {
			here = false
		}
		open_break(subject, here, broken)
		broken = broken || here
		if bool(here) {
			close_line(subject, tree)
			emit_indent(subject)
		}
		if !bool(here) {
			emit_space(subject)
		}
		print_expression(subject, tree)
		written = true
		open = advance_part(subject, tree)
	}
	// A case head the print reads inside the widest head it admits stands on one line, thus
	// the values it names take no break at all.
	if bool(flat) {
		if bool(writes_canonical(subject.Flags.Values)) {
			aver.Always(!bool(broken), "A case head that fits takes no break.")
		}
	}
	if bool(broken) {
		indent_shallow(subject, indent_load(subject.Counts.Values))
	}
	if bool(entered) {
		finish_parts(subject, tree)
		ascend(subject)
	}
}

// Reports whether the form the walk stands on writes every part of itself on the line it opens
// on. A struct the author wrote on one line stands on one line, thus a table of cases reads as a
// table rather than as a page of fields.
func spans_one_line(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "spans_one_line.yes") }()
	Printer_Handle_Invariants(subject, "spans_one_line.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "spans_one_line.tree")
	written := written_load(subject.Counts.Values)
	defer func() {
		aver.Always(written_load(subject.Counts.Values) == written,
			"A read of the line one body stands on writes no byte of its own.")
	}()
	closer := closer_position(subject, tree)
	if closer == 0 {
		return false
	}
	node := ast.Syntax_Node_At(tree, subject.Nodes.Values[depth_load(subject.Counts.Values)])
	opening := ast.Syntax_Token_At(tree, node.Token)
	ahead := ast.Syntax_Token_At(tree, ast.Token_Index(closer))
	// A line feed between the two bytes states a break the author wrote.
	from_offset, to_offset := int(int(opening.Offset)+int(opening.Size)), int(ahead.Offset)
	for place_offset := from_offset; place_offset < to_offset; place_offset++ {
		if subject.Source.Values[place_offset] == '\n' {
			return false
		}
	}
	// A declaration and a clause each close the line they stand on, thus a body that holds
	// one writes no line of its own and stands as a body of many lines does.
	return !holds_closing_part(subject, tree)
}

// Writes one struct type the author stated on one line, the fields between semicolons.
func print_inline_structure(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "print_inline_structure.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "print_inline_structure.tree")
	stood := written_load(subject.Counts.Values)
	emit_byte(subject, '{')
	open := descend(subject, tree)
	entered := open
	written := Field_Written(false)
	for bool(open) {
		if node_kind(subject, tree) != ast.NODE_FIELD {
			// The empty line behind the brace stands behind the declaration the
			// struct closes, thus the print still owes it.
			print_trivia(subject, tree)
			open = advance(subject, tree)
			continue
		}
		if bool(written) {
			emit_byte(subject, ';')
		}
		emit_space(subject)
		// The print writes one field of a structure here, thus the count of them reads
		// this one the way it reads the fields a struct over lines states.
		subject.Counts.Values[COUNT_FIELD] = subject.Counts.Values[COUNT_FIELD] + 1
		print_inline_field(subject, tree)
		written = true
		open = advance(subject, tree)
	}
	if bool(entered) {
		ascend(subject)
	}
	if bool(written) {
		emit_space(subject)
	}
	emit_byte(subject, '}')
	Field_Written_Invariants(written, "print_inline_structure.written")
	// A structure the author stated on one line writes the braces it holds its fields
	// between, thus the form holds both however many fields stand inside them.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(written_load(subject.Counts.Values) > stood+1,
			"A structure on one line writes the braces it holds its fields between.")
	}
}

// Field_Written names a structure on one line that already wrote a field of its own.
type Field_Written Boolean

// Field_Written_Invariants states both stands of the structure.
func Field_Written_Invariants(value Field_Written, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The structure already wrote a field.").
		Ensure()
}

// Writes one field of a struct the author stated on one line: the names it binds and the type
// behind them.
func print_inline_field(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "print_inline_field.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "print_inline_field.tree")
	stood := written_load(subject.Counts.Values)
	open := descend_quiet(subject, tree)
	entered := open
	written := Field_Named(false)
	for bool(open) {
		if node_kind(subject, tree) != ast.NODE_FIELD_NAME {
			break
		}
		if bool(written) {
			emit_comma(subject)
			emit_space(subject)
		}
		subject.Counts.Values[COUNT_FIELD_NAME] += 1
		emit_token(subject, tree)
		written = true
		open = advance_quiet(subject, tree)
	}
	if bool(open) {
		if bool(written) {
			emit_space(subject)
		}
		print_expression(subject, tree)
	}
	if bool(entered) {
		ascend(subject)
	}
	Field_Named_Invariants(written, "print_inline_field.written")
	// A field of a structure on one line states the names it binds and the type behind them,
	// thus the line it stands on holds bytes of it.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(written_load(subject.Counts.Values) > stood,
			"A field on one line writes bytes of its own.")
	}
}

// MARKS_NONE, MARKS_ONE, and MARKS_TWO are the counts of semicolons one header of a clause stands
// behind. A clause states three headers at most, thus two semicolons stand between them at most.
const MARKS_NONE = 0

// MARKS_ONE is the count of the semicolon that closes one header.
const MARKS_ONE = 1

// MARKS_TWO is the count of the semicolons that close two headers.
const MARKS_TWO = 2

// Marks is how many semicolons the author wrote ahead of one header or one body of a clause.
type Marks int

// Marks_Invariants states every count of semicolons a clause writes between two of its parts.
func Marks_Invariants(value Marks, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Int(int(value), MARKS_NONE, MARKS_ONE, MARKS_TWO).
		Ensure()
}

// Writes the semicolons the read ahead of one header counted, each with the space behind it.
func emit_marks(subject Printer_Handle, marks Marks) {
	Printer_Handle_Invariants(subject, "emit_marks.subject")
	Marks_Invariants(marks, "emit_marks.marks")
	stood := take_line_width(subject.Counts.Values)
	for range int(marks) {
		emit_byte(subject, ';')
		emit_space(subject)
	}
	// Each mark states a semicolon and the space behind it, thus the line spends two columns
	// for every mark the header holds.
	if bool(writes_canonical(subject.Flags.Values)) {
		spent := int(take_line_width(subject.Counts.Values)) - int(stood)
		aver.Always(spent == int(marks)*MARK_WIDTH,
			"A run of marks spends two columns for each mark it states.")
	}
}

// MARK_WIDTH is what one mark of a clause spends: the semicolon and the space behind it.
const MARK_WIDTH = 2

// Reads how many semicolons the author wrote ahead of the node the walk stands on. The tree drops
// the semicolons a clause states between its headers, and a loop that leaves a header out states
// a semicolon the tree names nothing at all for, thus the print reads them from the token run.
// Every such semicolon stands in one run behind the form ahead of it, and the first case of a
// switch stands behind the brace that closes the headers, thus the read steps over that brace and
// over every note. A semicolon a line feed stands for spans no byte, thus only one the author
// wrote counts.
func take_marks(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (marks Marks) {
	defer func() { Marks_Invariants(marks, "take_marks.marks") }()
	Printer_Handle_Invariants(subject, "take_marks.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "take_marks.tree")
	held_depth := depth_load(subject.Counts.Values)
	enter_leftmost(subject, tree)
	leftmost := subject.Nodes.Values[depth_load(subject.Counts.Values)]
	depth_store(subject.Counts.Values, held_depth)
	count := Marks(MARKS_NONE)
	braces := 0
	opening := int(ast.Syntax_Node_At(tree, leftmost).Token)
	// The body of one clause stands behind the word that clause opens with, thus the token
	// the body opens on names no position at the head of the run.
	aver.Always(opening > POSITION_MINIMUM,
		"A body of one clause opens behind the token the clause opens on.")
	for position := opening - 1; position > 0; position-- {
		one := ast.Syntax_Token_At(tree, ast.Token_Index(position))
		if one.Kind == token.KIND_COMMENT {
			continue
		}
		if one.Kind == token.KIND_BRACE_LEFT {
			braces = braces + 1
			if braces > 1 {
				break
			}
			continue
		}
		if one.Kind != token.KIND_SEMICOLON {
			break
		}
		if one.Size != 0 {
			count = count + 1
		}
	}
	return count
}

// Reports whether one assignment states more than one name and more than one value. The values of
// such an assignment stand as parts of a run rather than as statements of their own.
func assigns_runs(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "assigns_runs.yes") }()
	Printer_Handle_Invariants(subject, "assigns_runs.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "assigns_runs.tree")
	held := subject.Nodes.Values[depth_load(subject.Counts.Values)]
	depth := depth_load(subject.Counts.Values)
	names := 0
	values := 0
	crossed := false
	open := descend_quiet(subject, tree)
	for bool(open) {
		if bool(opens_right(subject, tree)) {
			crossed = true
		}
		if crossed {
			values = values + 1
		}
		if !crossed {
			names = names + 1
		}
		open = advance_quiet(subject, tree)
	}
	// The read steps into the parts of the assignment and never out of it, and an assignment
	// that crossed its sign holds a value behind it.
	aver.Always(depth_load(subject.Counts.Values) >= depth,
		"A read of the runs one assignment binds stands below the slot it opened.")
	depth_store(subject.Counts.Values, depth)
	subject.Nodes.Values[depth] = held
	if crossed {
		aver.Always(values > 0, "An assignment that crossed its sign holds a value.")
	}
	return Boolean(names > 1 && values > 1)
}

// Reports whether the signature the walk stands at the opening of runs the line past the widest
// line. The signature holds no node of its own, thus the measure writes the parameters and the
// results the walk stands ahead of and reads the columns they wrote.
func runs_over_signature(
	subject Printer_Handle, tree ast.Syntax_Tree_Handle, open Boolean,
) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "runs_over_signature.yes") }()
	Printer_Handle_Invariants(subject, "runs_over_signature.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "runs_over_signature.tree")
	Boolean_Invariants(open, "runs_over_signature.open")
	if bool(flag_load(subject.Flags.Values, FLAG_FLAT)) {
		return false
	}
	var counts [COUNT_SLOT_COUNT]Count
	copy(counts[:], subject.Counts.Values)
	var flags [FLAG_COUNT]Boolean
	copy(flags[:], subject.Flags.Values)
	var comments [COMMENT_COUNT_MAXIMUM]ast.Index
	copy(comments[:], subject.Comments.Values)
	node := subject.Nodes.Values[depth_load(subject.Counts.Values)]

	flag_store(subject.Flags.Values, FLAG_FLAT, true)
	layout_run(subject, tree, RULE_HEAD, open)
	// The flat print writes no line feed, thus the line it closes holds the columns the line
	// held ahead of it and every column the print writes.
	columns := int(take_line_width(subject.Counts.Values))
	// A flat print writes the signature on the line it stood on, thus the line it closes
	// stands no nearer than the line the print opened on.
	aver.Always(columns >= int(counts[COUNT_COLUMNS]),
		"A flat read of one signature closes no nearer than the line it opened on.")
	closes_flat(subject, counts[:])
	restore_counts(subject, counts[:])
	restore_flags(subject, flags[:])
	copy(subject.Comments.Values, comments[:])
	subject.Nodes.Values[depth_load(subject.Counts.Values)] = node
	closes_measure(subject, counts[:])
	// The brace of the body stands on the line the signature closes, thus it counts against
	// the widest line the same way every other byte of that line does.
	return Boolean(columns+len(BODY_HEAD) > COLUMN_WIDTH_MAXIMUM)
}

// Reports whether the results the walk stands at the opening of run the line past the widest one.
// The measure writes the results flat and reads the columns they wrote, and a measure inside a
// measure answers false, thus the writing it runs states no measure of its own.
func runs_over_results(
	subject Printer_Handle, tree ast.Syntax_Tree_Handle, open Boolean,
) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "runs_over_results.yes") }()
	Printer_Handle_Invariants(subject, "runs_over_results.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "runs_over_results.tree")
	Boolean_Invariants(open, "runs_over_results.open")
	if bool(flag_load(subject.Flags.Values, FLAG_FLAT)) {
		return false
	}
	var counts [COUNT_SLOT_COUNT]Count
	copy(counts[:], subject.Counts.Values)
	var flags [FLAG_COUNT]Boolean
	copy(flags[:], subject.Flags.Values)
	var comments [COMMENT_COUNT_MAXIMUM]ast.Index
	copy(comments[:], subject.Comments.Values)
	node := subject.Nodes.Values[depth_load(subject.Counts.Values)]

	flag_store(subject.Flags.Values, FLAG_FLAT, true)
	layout_run(subject, tree, RULE_RESULTS, open)
	// The flat print writes no line feed, thus the line it closes holds the columns the line
	// held ahead of it and every column the print writes.
	columns := int(take_line_width(subject.Counts.Values))
	// A flat print writes the results on the line it stood on, thus the line it closes
	// stands no nearer than the line the print opened on.
	aver.Always(columns >= int(counts[COUNT_COLUMNS]),
		"A flat read of the results closes no nearer than the line it opened on.")
	closes_flat(subject, counts[:])
	restore_counts(subject, counts[:])
	restore_flags(subject, flags[:])
	copy(subject.Comments.Values, comments[:])
	subject.Nodes.Values[depth_load(subject.Counts.Values)] = node
	closes_measure(subject, counts[:])
	return Boolean(columns+len(BODY_HEAD) > COLUMN_WIDTH_MAXIMUM)
}

// BODY_HEAD is the text that stands between one signature and the body it opens.
const BODY_HEAD = " {"

// Reports whether the form the walk stands on runs the line it opens past the widest line, which
// is what states one part of that form to a line. The measure writes the form flat past the bytes
// the print already wrote and reads the columns back, thus it costs no storage of its own.
func runs_over(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "runs_over.yes") }()
	Printer_Handle_Invariants(subject, "runs_over.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "runs_over.tree")
	// A measure inside a measure answers nothing the outer one asks, because the outer print
	// already stands flat and holds every break it would take.
	if bool(flag_load(subject.Flags.Values, FLAG_FLAT)) {
		return false
	}
	var counts [COUNT_SLOT_COUNT]Count
	copy(counts[:], subject.Counts.Values)
	var flags [FLAG_COUNT]Boolean
	copy(flags[:], subject.Flags.Values)
	var comments [COMMENT_COUNT_MAXIMUM]ast.Index
	copy(comments[:], subject.Comments.Values)
	node := subject.Nodes.Values[depth_load(subject.Counts.Values)]

	flag_store(subject.Flags.Values, FLAG_FLAT, true)
	print_expression(subject, tree)
	// A form that holds a body of its own writes the lines of that body however flat the
	// print stands, thus the read counts the line the form opens and stops at the first feed
	// standing behind it.
	held_written := written_load(subject.Counts.Values)
	// Every form the print measures flat writes bytes of its own, thus the read stands
	// behind what the flat print wrote.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(int(held_written) > int(counts[COUNT_WRITTEN]),
			"A flat measure of one form writes the form it reads.")
	}
	place := int(counts[COUNT_WRITTEN])
	for place < int(held_written) {
		if subject.Form.Values[place] == '\n' {
			break
		}
		place = place + 1
	}
	written_store(subject.Counts.Values, Form_Count(place))
	// The line the form opens holds the columns the print stood at and every column the
	// flat print wrote ahead of the feed it took.
	stood := int(counts[COUNT_COLUMNS]) + int(measured_columns(subject, counts[:]))
	columns := Line_Width(min(stood, WIDTH_MAXIMUM))
	written_store(subject.Counts.Values, held_written)
	restore_counts(subject, counts[:])
	restore_flags(subject, flags[:])
	copy(subject.Comments.Values, comments[:])
	subject.Nodes.Values[depth_load(subject.Counts.Values)] = node
	closes_measure(subject, counts[:])
	// A literal writes one comma behind every element it states, thus the line an element
	// closes holds that comma and the read counts it.
	spent := int(columns)
	if bool(owes_comma(subject, tree)) {
		spent = spent + COMMA_WIDTH
	}
	return Boolean(spent > COLUMN_WIDTH_MAXIMUM)
}

// COMMA_WIDTH is what the comma behind one element of a literal spends on the line it closes.
const COMMA_WIDTH = 1

// Reports whether the form the walk stands on stands as one element of a literal, which the
// literal writes one comma behind. A pair states its value on the line the pair closes, thus a
// value of one pair owes that comma the way an element of its own does.
func owes_comma(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "owes_comma.yes") }()
	Printer_Handle_Invariants(subject, "owes_comma.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "owes_comma.tree")
	index := subject.Nodes.Values[depth_load(subject.Counts.Values)]
	parent := ast.Index(ast.Syntax_Node_At(tree, index).Parent)
	held := ast.Syntax_Node_At(tree, parent).Kind
	if held == ast.NODE_COMPOSITE {
		return true
	}
	if held != ast.NODE_KEY_VALUE {
		return false
	}
	above := ast.Index(ast.Syntax_Node_At(tree, parent).Parent)
	return Boolean(ast.Syntax_Node_At(tree, above).Kind == ast.NODE_COMPOSITE)
}

// Reads how wide the expression the walk stands on writes and leaves the print stand where it
// stood.
func measure_expression(
	subject Printer_Handle, tree ast.Syntax_Tree_Handle,
) (width Measured_Width) {
	defer func() { Measured_Width_Invariants(width, "measure_expression.width") }()
	Printer_Handle_Invariants(subject, "measure_expression.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "measure_expression.tree")
	var counts [COUNT_SLOT_COUNT]Count
	copy(counts[:], subject.Counts.Values)
	var flags [FLAG_COUNT]Boolean
	copy(flags[:], subject.Flags.Values)
	var comments [COMMENT_COUNT_MAXIMUM]ast.Index
	copy(comments[:], subject.Comments.Values)
	node := subject.Nodes.Values[depth_load(subject.Counts.Values)]
	print_expression(subject, tree)
	written := measured_columns(subject, counts[:])
	// An expression writes bytes of its own, thus the measure reads a column the reader of
	// that expression reads.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(written > 0, "A measure of one expression reads a column of it.")
	}
	restore_counts(subject, counts[:])
	restore_flags(subject, flags[:])
	copy(subject.Comments.Values, comments[:])
	subject.Nodes.Values[depth_load(subject.Counts.Values)] = node
	closes_measure(subject, counts[:])
	return Measured_Width(min(int(written), WIDTH_MAXIMUM))
}

// Reads the columns the print counted for the line it stands on.
func columns_load(counts Counts) (width Line_Width) {
	defer func() { Line_Width_Invariants(width, "columns_load.width") }()
	Counts_Invariants(counts, "columns_load.counts")
	return Line_Width(counts[COUNT_COLUMNS])
}

// Stands the columns the line the print stands on holds.
func columns_store(counts Counts, width Line_Width) {
	Counts_Invariants(counts, "columns_store.counts")
	Line_Width_Invariants(width, "columns_store.width")
	counts[COUNT_COLUMNS] = Count(width)
}

// STATEMENT_WIDTH_MINIMUM is the narrowest line one statement of a block writes: the tab the
// block opens its statements at, which stands in the columns a reader sees, and one byte of the
// statement itself.
const STATEMENT_WIDTH_MINIMUM = INDENT_WIDTH + 1

// Statement_Width is the width the line one statement of a block opens writes.
type Statement_Width int32

// Statement_Width_Invariants states every width one statement writes.
func Statement_Width_Invariants(value Statement_Width, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), STATEMENT_WIDTH_MINIMUM, WIDTH_MAXIMUM).
		Ensure()
}

// Statement_Measure states what one statement writes: the width of the line it opens and whether
// the print breaks it over lines of its own.
type Statement_Measure struct {
	// Width holds the width the statement writes.
	Width Statement_Width
	// Spans reports whether the print breaks the statement over lines of its own.
	Spans Broken_Form
}

// Statement_Measure_Invariants composes one measure of a statement.
func Statement_Measure_Invariants(value Statement_Measure, namespace aver.Namespace) {
	Statement_Width_Invariants(value.Width, namespace)
	Broken_Form_Invariants(value.Spans, namespace)
}

// Measures the statement the walk stands on by writing it past the bytes the print already wrote
// and reading what it spans. The measure costs no storage of its own, thus it takes the print
// back to where it stood once it read the width.
func measure_statement(
	subject Printer_Handle, tree ast.Syntax_Tree_Handle,
) (measured Statement_Measure) {
	defer func() { Statement_Measure_Invariants(measured, "measure_statement.measured") }()
	Printer_Handle_Invariants(subject, "measure_statement.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "measure_statement.tree")
	var counts [COUNT_SLOT_COUNT]Count
	copy(counts[:], subject.Counts.Values)
	var flags [FLAG_COUNT]Boolean
	copy(flags[:], subject.Flags.Values)
	var comments [COMMENT_COUNT_MAXIMUM]ast.Index
	copy(comments[:], subject.Comments.Values)
	node := subject.Nodes.Values[depth_load(subject.Counts.Values)]
	start := written_load(subject.Counts.Values)
	flag_store(subject.Flags.Values, FLAG_NOTELESS, true)
	emit_indent(subject)
	print_statement_form(subject, tree)
	flag_store(subject.Flags.Values, FLAG_NOTELESS, false)
	spans := Boolean(false)
	shaped := subject.Form.Values
	limit := int(written_load(subject.Counts.Values))
	// A declaration closes the line it stands on, and an empty line the author left behind
	// the statement stands behind it, thus every feed the print wrote last closes the
	// statement rather than breaking it.
	for limit > int(start) {
		if shaped[limit-1] != '\n' {
			break
		}
		limit = limit - 1
	}
	for place := int(start); place < limit; place++ {
		if shaped[place] == '\n' {
			spans = true
			break
		}
	}
	// One character stands in one column however many bytes it spans, and a tab stands in
	// the columns a reader sees, thus the width counts columns and not bytes.
	width := 0
	for place := int(start); place < limit; place++ {
		if shaped[place] == '\t' {
			width = width + TAB_WIDTH
			continue
		}
		if shaped[place]&CHARACTER_TAIL_MASK == CHARACTER_TAIL_MARK {
			continue
		}
		width = width + 1
	}
	// Every statement writes bytes of its own behind the tabs its line opens at, thus the
	// measure reads a line the statement wrote and never an empty one.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(limit > int(start)+int(indent_load(subject.Counts.Values)),
			"A measure of one statement reads the bytes that statement wrote.")
	}
	restore_counts(subject, counts[:])
	restore_flags(subject, flags[:])
	copy(subject.Comments.Values, comments[:])
	subject.Nodes.Values[depth_load(subject.Counts.Values)] = node
	closes_measure(subject, counts[:])
	return Statement_Measure{
		Width: Statement_Width(min(max(width, STATEMENT_WIDTH_MINIMUM), WIDTH_MAXIMUM)),
		Spans: Broken_Form(spans),
	}
}

// Reports whether a note stands behind the node the walk stands on, on the line that node closes.
func trails_comment(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "trails_comment.yes") }()
	Printer_Handle_Invariants(subject, "trails_comment.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "trails_comment.tree")
	stood := written_load(subject.Counts.Values)
	defer func() {
		aver.Always(written_load(subject.Counts.Values) == stood,
			"A read of the note one node closes its line with writes no byte.")
	}()
	// The tree hangs a note wherever the walk reached it, thus the run of tokens states which
	// line the note stands on rather than the child chain that holds it.
	depth := depth_load(subject.Counts.Values)
	held := subject.Nodes.Values[depth]
	enter_leftmost(subject, tree)
	first := subject.Nodes.Values[depth_load(subject.Counts.Values)]
	start := int(ast.Syntax_Node_At(tree, first).Token)
	depth_store(subject.Counts.Values, depth)
	// A source the author left open holds forms that close on the end of the file, thus the
	// node the walk stands on names that end and no token stands behind it.
	if ast.Syntax_Token_At(tree, ast.Token_Index(start)).Kind == token.KIND_END_OF_FILE {
		subject.Nodes.Values[depth] = held
		return false
	}
	limit := ast.TOKEN_INDEX_MAXIMUM
	last := !bool(advance(subject, tree))
	if !last {
		// A list hangs a note beside the element it follows, thus the note stands as
		// the sibling of that element rather than inside it.
		if node_kind(subject, tree) == ast.NODE_COMMENT {
			alone := stands_alone(subject, tree)
			subject.Nodes.Values[depth] = held
			return !alone
		}
		// A statement that opens with a word of its own names that word, and the walk
		// reaches the name behind it, thus the run closes at whichever stands first.
		own := int(ast.Syntax_Node_At(tree, subject.Nodes.Values[depth]).Token)
		enter_leftmost(subject, tree)
		leftmost := subject.Nodes.Values[depth_load(subject.Counts.Values)]
		depth_store(subject.Counts.Values, depth)
		limit = min(own, int(ast.Syntax_Node_At(tree, leftmost).Token))
	}
	subject.Nodes.Values[depth] = held
	// A read of the run behind the node leaves the walk where it stood.
	aver.Always(depth_load(subject.Counts.Values) == depth,
		"A read of the run behind a node leaves the walk at the depth it stood on.")
	source := subject.Source.Values
	for position := start + 1; position < limit; position++ {
		one := ast.Syntax_Token_At(tree, ast.Token_Index(position))
		if one.Kind == token.KIND_END_OF_FILE {
			return false
		}
		// A line feed between the token and the one ahead of it opens a line of its own.
		behind := ast.Syntax_Token_At(tree, ast.Token_Index(position-1))
		from_offset := int(behind.Offset) + int(behind.Size)
		for place_offset := from_offset; place_offset < int(one.Offset); place_offset++ {
			if source[place_offset] == '\n' {
				return false
			}
		}
		if one.Kind != token.KIND_COMMENT {
			continue
		}
		// A note the print holds lands at the end of the line the statement writes,
		// wherever the author left it inside that statement.
		return true
	}
	return false
}

// Reports whether an empty line stands ahead of the comment at one run position.
func opens_run(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "opens_run.yes") }()
	Printer_Handle_Invariants(subject, "opens_run.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "opens_run.tree")
	cursor := ast.Syntax_Node_At(tree, subject.Nodes.Values[depth_load(subject.Counts.Values)])
	position := cursor.Token
	if position == 0 {
		return false
	}
	return Boolean(ast.Syntax_Token_At(tree, ast.Token_Index(position)).Blanks > 0)
}

// Reports whether the node the walk stands on names a token behind the bracket that closes the
// block around it.
func stands_behind(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "stands_behind.yes") }()
	Printer_Handle_Invariants(subject, "stands_behind.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "stands_behind.tree")
	depth := depth_load(subject.Counts.Values)
	node := ast.Syntax_Node_At(tree, subject.Nodes.Values[depth])
	depth_store(subject.Counts.Values, depth-1)
	closer := closer_position(subject, tree)
	depth_store(subject.Counts.Values, depth)
	// A form that closes on no bracket at all holds every node it opened, thus no node of it
	// stands behind a bracket the form never states.
	if closer == CLOSER_POSITION_NONE {
		return false
	}
	return Boolean(Closer_Position(node.Token) > closer)
}

// Reads the column the notes of the run of elements that opens where the walk stands write at.
// An element that carries no note closes the run, thus the notes of one broken list read as one
// column rather than as a ragged edge.
func take_element_column(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "take_element_column.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "take_element_column.tree")
	stood := depth_load(subject.Counts.Values)
	held := subject.Nodes.Values[stood]
	widest := Width(0)
	open := Boolean(true)
	for bool(open) {
		if bool(trivia_kind(subject, tree)) {
			break
		}
		if !bool(trails_comment(subject, tree)) {
			break
		}
		if bool(spans_form(subject, tree)) {
			break
		}
		// Every line of a run holds the tab of its form, a byte of the element it states,
		// and the comma that closes it, thus the run opens its column past those tabs.
		held_width := element_width(subject, tree)
		aver.Always(int(held_width) >= ELEMENT_WIDTH_MINIMUM,
			"An element of one run states the width its line holds.")
		// An element that runs the line past the widest one stands over lines of its
		// own once the print breaks it, thus it closes the run and states no column.
		if int(held_width) > COLUMN_WIDTH_MAXIMUM {
			break
		}
		widest = max(widest, Width(held_width))
		open = advance(subject, tree)
		if bool(open) {
			if bool(trivia_kind(subject, tree)) {
				open = advance(subject, tree)
			}
		}
	}
	subject.Nodes.Values[depth_load(subject.Counts.Values)] = held
	column_store(subject.Counts.Values, Note_Column(min(int(widest), WIDTH_MAXIMUM)))
	// The read steps along the run with a step that holds the slot it stood on, thus it
	// hands back the walk it opened.
	aver.Always(depth_load(subject.Counts.Values) == stood,
		"A read of one run of elements holds the slot it opened.")
	// A run opens its column behind the widest line it holds, and every line of a run of
	// elements holds the tab of its form, a byte of the element, and the comma behind it.
	if column_load(subject.Counts.Values) != 0 {
		aver.Always(int(column_load(subject.Counts.Values)) >= ELEMENT_WIDTH_MINIMUM,
			"A run of notes on elements opens past the tabs of its lines.")
	}
}

// ELEMENT_WIDTH_MINIMUM is the narrowest line one element of a list writes. A list that carries a
// note on an element stands one element to a line, thus the element stands behind one tab at
// least, which stands in the columns a reader sees, and the comma closes it.
const ELEMENT_WIDTH_MINIMUM = INDENT_WIDTH + 2

// Element_Width is the width the line of one element of a list writes.
type Element_Width int32

// Element_Width_Invariants states every width one element writes.
func Element_Width_Invariants(value Element_Width, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), ELEMENT_WIDTH_MINIMUM, WIDTH_MAXIMUM).
		Ensure()
}

// Reads the width the line of one element writes: the tabs the list opens at, the element, and
// the comma that closes it.
func element_width(
	subject Printer_Handle, tree ast.Syntax_Tree_Handle,
) (width Element_Width) {
	defer func() { Element_Width_Invariants(width, "element_width.width") }()
	Printer_Handle_Invariants(subject, "element_width.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "element_width.tree")
	measured := int(measure_expression(subject, tree))
	// A tab stands in the columns a reader sees, thus the width of the line counts those
	// columns the way the read of a written line counts them.
	total := int(indent_load(subject.Counts.Values))*INDENT_WIDTH + measured + 1
	return Element_Width(min(max(total, ELEMENT_WIDTH_MINIMUM), WIDTH_MAXIMUM))
}

// Writes the notes the author left on the line a brace opens, and steps the walk past them. It
// reports whether a part stands behind them, because a form of notes alone holds none.
func open_brace_note(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (stood Boolean) {
	defer func() { Boolean_Invariants(stood, "open_brace_note.stood") }()
	Printer_Handle_Invariants(subject, "open_brace_note.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "open_brace_note.tree")
	for bool(trails_open_brace(subject, tree)) {
		print_trivia(subject, tree)
		if !bool(advance(subject, tree)) {
			return false
		}
	}
	return true
}

// Reports whether the node the walk stands on states a note the author left on the line the
// brace of the form opens. A note standing past the brace that closes the form stands on the
// line that brace closes, thus the brace that opens the form takes none of it.
func trails_open_brace(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "trails_open_brace.yes") }()
	Printer_Handle_Invariants(subject, "trails_open_brace.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "trails_open_brace.tree")
	if !bool(trails_line(subject, tree)) {
		return false
	}
	return Boolean(!bool(trails_close(subject, tree)))
}

// Reports whether the node the walk stands on states a note on the line that already stands
// open, which the note closes.
func trails_line(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "trails_line.yes") }()
	Printer_Handle_Invariants(subject, "trails_line.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "trails_line.tree")
	written := written_load(subject.Counts.Values)
	defer func() {
		aver.Always(written_load(subject.Counts.Values) == written,
			"A read of the line one node closes writes no byte of its own.")
	}()
	if node_kind(subject, tree) != ast.NODE_COMMENT {
		return false
	}
	return !stands_alone(subject, tree)
}

// BRACKET_NEITHER names a token that opens and closes no bracketed form.
const BRACKET_NEITHER = 0

// BRACKET_OPENS names a token that opens one bracketed form.
const BRACKET_OPENS = 1

// BRACKET_CLOSES names a token that closes one.
const BRACKET_CLOSES = 2

// Bracket_Step names what one token does to the brackets that stand open around it.
type Bracket_Step uint8

// Bracket_Step_Invariants states every step one token takes over the brackets.
func Bracket_Step_Invariants(value Bracket_Step, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(uint8(value), BRACKET_NEITHER, BRACKET_OPENS, BRACKET_CLOSES).
		Ensure()
}

// Names what one token does to the brackets that stand open around it, which is how a read over a
// run of tokens holds the forms the brackets state apart from the form that holds them.
func bracket_step(sign token.Scanned_Kind) (step Bracket_Step) {
	defer func() { Bracket_Step_Invariants(step, "bracket_step.step") }()
	token.Scanned_Kind_Invariants(sign, "bracket_step.sign")
	if bool(opens_bracket(sign)) {
		return BRACKET_OPENS
	}
	if bool(closes_bracket(sign)) {
		return BRACKET_CLOSES
	}
	return BRACKET_NEITHER
}

// Reports whether the child the walk stands on states the further clause an else branch opens
// with, which closes its own line and leaves the clause around it none to close.
func opens_else_clause(
	subject Printer_Handle, tree ast.Syntax_Tree_Handle, bodied Bodied,
) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "opens_else_clause.yes") }()
	Printer_Handle_Invariants(subject, "opens_else_clause.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "opens_else_clause.tree")
	Bodied_Invariants(bodied, "opens_else_clause.bodied")
	held := Boolean(bodied)
	if !bool(held) {
		return false
	}
	return Boolean(node_kind(subject, tree) == ast.NODE_IF)
}

// Reads the column the notes of a broken form open at, which only a broken form holds.
func take_notes(subject Printer_Handle, tree ast.Syntax_Tree_Handle, opened Boolean) {
	Printer_Handle_Invariants(subject, "take_notes.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "take_notes.tree")
	Boolean_Invariants(opened, "take_notes.opened")
	if !bool(opened) {
		return
	}
	if column_load(subject.Counts.Values) != 0 {
		return
	}
	take_element_column(subject, tree)
}

// Closes the column of notes wherever the part the print wrote carries none behind it.
func close_notes(subject Printer_Handle, opened Boolean, trails Boolean) {
	Printer_Handle_Invariants(subject, "close_notes.subject")
	Boolean_Invariants(opened, "close_notes.opened")
	Boolean_Invariants(trails, "close_notes.trails")
	if !bool(opened) {
		return
	}
	if bool(trails) {
		return
	}
	column_store(subject.Counts.Values, 0)
}

// Writes the semicolons and the space that stand between the headers of one clause and the body
// it opens, and returns the tabs and the break state the body stands at.
func open_body(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "open_body.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "open_body.tree")
	stood := written_load(subject.Counts.Values)
	indent := column_indent_load(subject.Counts.Values)
	marks := take_marks(subject, tree)
	emit_marks(subject, marks)
	indent_store(subject.Counts.Values, indent)
	// The body of a clause opens a line of its own, thus no continuation of the header stands
	// open where it opens.
	flag_store(subject.Flags.Values, FLAG_BROKEN, false)
	if marks == 0 {
		emit_space(subject)
	}
	// The body of a clause stands behind the header it opens, thus the opening writes the
	// space or the semicolons that part the two.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(written_load(subject.Counts.Values) > stood,
			"A body of one clause stands behind what parts it from the header.")
		// Each mark states the space behind it, and a clause of no mark writes that space
		// of its own, thus the body opens on a space either way.
		aver.Always(bool(stands_after_space(subject)),
			"A body of one clause opens on the space that parts it from the header.")
	}
}

// Writes one header of a clause and the semicolons that stand ahead of it.
func print_header(subject Printer_Handle, tree ast.Syntax_Tree_Handle, headed Clause_Headed) {
	Printer_Handle_Invariants(subject, "print_header.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "print_header.tree")
	Clause_Headed_Invariants(headed, "print_header.headed")
	first := Boolean(!bool(headed))
	marks := take_marks(subject, tree)
	// A loop that leaves a header out still states the semicolon that header stands between,
	// thus a space stands on each side of that semicolon.
	if bool(first) {
		if marks > 0 {
			emit_space(subject)
		}
	}
	emit_marks(subject, marks)
	if marks == 0 {
		emit_space(subject)
	}
	// Each mark states the space behind it, and a header no mark stands ahead of writes that
	// space of its own, thus the header opens on a space either way.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(bool(stands_after_space(subject)),
			"A header of one clause opens on the space that parts it from the word.")
	}
	stood := written_load(subject.Counts.Values)
	print_statement_form(subject, tree)
	// A header states the form the clause reads, thus the line the clause opens holds the
	// bytes that form wrote.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(written_load(subject.Counts.Values) > stood,
			"A header of one clause writes bytes of its own.")
	}
}

// PART_HEAD names the part one form opens with, which is the value its brackets read.
const PART_HEAD = 0

// PART_FIRST names the part that stands first inside the brackets of one form.
const PART_FIRST = 1

// PART_BEHIND names every part that stands behind the first one inside those brackets.
const PART_BEHIND = 2

// Part_Place names where one part of a wrapped form stands. Every part behind the first stands
// the same way, thus the walk holds all of them as one place.
type Part_Place uint8

// Part_Place_Invariants states every place one part of a wrapped form stands at.
func Part_Place_Invariants(value Part_Place, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(uint8(value), PART_HEAD, PART_FIRST, PART_BEHIND).
		Ensure()
}

// Reports whether the node the walk stands on spreads the part behind it. The spread of a call
// stands as a node of its own behind the value it spreads, thus it takes no comma and no space
// of its own, and one standing where the form opens names what that form reads instead.
// Reports whether the node the walk stands on states the points the author wrote behind the
// argument ahead of them, which hold no value of their own.
func trails_spread(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "trails_spread.yes") }()
	Printer_Handle_Invariants(subject, "trails_spread.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "trails_spread.tree")
	if node_kind(subject, tree) != ast.NODE_ELLIPSIS {
		return false
	}
	index := subject.Nodes.Values[depth_load(subject.Counts.Values)]
	return Boolean(ast.Index(ast.Syntax_Node_At(tree, index).First_Child) == ast.INDEX_ABSENT)
}

func stands_spread(
	subject Printer_Handle, tree ast.Syntax_Tree_Handle, place Part_Place,
) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "stands_spread.yes") }()
	Printer_Handle_Invariants(subject, "stands_spread.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "stands_spread.tree")
	Part_Place_Invariants(place, "stands_spread.place")
	if place == PART_HEAD {
		return false
	}
	return Boolean(node_kind(subject, tree) == ast.NODE_ELLIPSIS)
}

// Writes the points of one spread and the value the tree hangs on them. A source that states a
// value behind the points hangs that value on the node the points stand for, thus the print
// writes it rather than dropping it.
func print_spread(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "print_spread.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "print_spread.tree")
	// The points the author wrote behind an argument stand against that argument, thus no
	// space holds the two apart. Points that hold the value they spread open an argument of
	// their own and stand behind the comma that closes the one ahead of them.
	if bool(trails_spread(subject, tree)) {
		if bool(writes_canonical(subject.Flags.Values)) {
			aver.Always(!bool(stands_after_space(subject)),
				"The points behind one argument stand against it.")
		}
	}
	// The points of a spread stand on a node of the tree the way the points of a variadic
	// parameter do, thus the count of them reads both.
	counts_type(subject, tree)
	emit_spread(subject)
	if !bool(descend_part(subject, tree)) {
		return
	}
	print_expression(subject, tree)
	finish_parts(subject, tree)
	ascend(subject)
}

// Writes the three points one call states behind the value it spreads. The tree holds a node
// behind that value rather than on the points, thus the print writes them of its own.
func emit_spread(subject Printer_Handle) {
	Printer_Handle_Invariants(subject, "emit_spread.subject")
	stood := written_load(subject.Counts.Values)
	emit_byte(subject, '.')
	emit_byte(subject, '.')
	emit_byte(subject, '.')
	// The points of a spread spell three bytes of their own.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(int(written_load(subject.Counts.Values)) == int(stood)+SPREAD_SIZE,
			"A spread the print writes spells the points it states.")
	}
}

// SPREAD_SIZE is how many bytes the points of one spread spend.
const SPREAD_SIZE = 3

// Writes one part of a broken form and holds the column its note stands in.
func print_part(
	subject Printer_Handle, tree ast.Syntax_Tree_Handle, opened Boolean, place Part_Place,
) {
	Printer_Handle_Invariants(subject, "print_part.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "print_part.tree")
	Boolean_Invariants(opened, "print_part.opened")
	Part_Place_Invariants(place, "print_part.place")
	behind := Boolean(place != PART_HEAD)
	if bool(behind) {
		take_notes(subject, tree, opened)
	}
	trails := trails_comment(subject, tree)
	stood := written_load(subject.Counts.Values)
	print_expression(subject, tree)
	// A part states a form the whole holds, thus the print writes bytes for every one.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(written_load(subject.Counts.Values) > stood,
			"A part of one form writes bytes of its own.")
	}
	close_notes(subject, opened, trails)
}

// Opens the column the values of a run of pairs stand in. A run holds the pairs that each open a
// line of their own, thus the pairs behind a closed run answer for a width of their own, and a
// pair that runs over lines of its own opens a run rather than joining one.
func open_key_run(subject Printer_Handle, tree ast.Syntax_Tree_Handle, here Boolean) {
	Printer_Handle_Invariants(subject, "open_key_run.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "open_key_run.tree")
	Boolean_Invariants(here, "open_key_run.here")
	if !bool(here) {
		return
	}
	if bool(spans_value(subject, tree)) {
		key_width_store(subject.Counts.Values, 0)
	}
	if key_width_load(subject.Counts.Values) != 0 {
		return
	}
	stood := written_load(subject.Counts.Values)
	take_key_width(subject, tree)
	// The read of one run states a column and never a byte, thus the form stands as wide
	// once the read closed as it stood when the read opened.
	aver.Always(written_load(subject.Counts.Values) == stood,
		"A read of the column one run of pairs states writes no byte.")
}

// Closes the column of a run of pairs wherever the element the print wrote closed the run: an
// element that carries trivia behind it, and one that ran over lines of its own.
func close_key_run(subject Printer_Handle, tree ast.Syntax_Tree_Handle, spans Boolean) {
	Printer_Handle_Invariants(subject, "close_key_run.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "close_key_run.tree")
	Boolean_Invariants(spans, "close_key_run.spans")
	if bool(breaks_run(subject, tree)) {
		key_width_store(subject.Counts.Values, 0)
	}
	if bool(spans) {
		key_width_store(subject.Counts.Values, 0)
	}
}

// Reports whether one literal states a type ahead of its brace. A literal that stands as an
// element of another one states none, thus the print reads whether a part stands ahead of the
// brace the literal names rather than assuming one does.
func typed_head(subject Printer_Handle, tree ast.Syntax_Tree_Handle, open Boolean) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "typed_head.yes") }()
	Printer_Handle_Invariants(subject, "typed_head.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "typed_head.tree")
	Boolean_Invariants(open, "typed_head.open")
	if !bool(open) {
		return false
	}
	held_depth := depth_load(subject.Counts.Values)
	// The head stands inside the form that holds it, thus the walk stands on a slot with a
	// form above it to read the brace from.
	aver.Always(held_depth > 0, "A head the print reads stands under the form that holds it.")
	brace := ast.Syntax_Node_At(tree, subject.Nodes.Values[held_depth-1]).Token
	enter_leftmost(subject, tree)
	// The read steps into the form and never out of it, thus it stands at the slot it opened
	// or below it.
	aver.Always(depth_load(subject.Counts.Values) >= held_depth,
		"A read of the leftmost token of one head stands below the slot it opened.")
	leftmost := subject.Nodes.Values[depth_load(subject.Counts.Values)]
	depth_store(subject.Counts.Values, held_depth)
	return Boolean(ast.Syntax_Node_At(tree, leftmost).Token < brace)
}

// Writes one case of a switch, and the brace its run of cases opens at. The tree hangs the cases
// of a switch off the switch itself, thus the print writes the braces they stand between.
func print_switch_case(subject Printer_Handle, tree ast.Syntax_Tree_Handle, cased Cased) {
	Printer_Handle_Invariants(subject, "print_switch_case.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "print_switch_case.tree")
	Cased_Invariants(cased, "print_switch_case.cased")
	stood := written_load(subject.Counts.Values)
	if !bool(cased) {
		open_body(subject, tree)
		emit_byte(subject, '{')
		close_line(subject, tree)
		// The brace of the body stands ahead of the first case the switch holds.
		if bool(writes_canonical(subject.Flags.Values)) {
			aver.Always(written_load(subject.Counts.Values) > stood,
				"A switch opens the brace its first case stands behind.")
		}
	}
	print_case_line(subject, tree)
}

// Reads whether the run of declarations the print just wrote spans more than one line, and opens
// the empty line that stands between it and the run behind it where both do. A declaration states
// the lines it spans only once the print wrote it, thus the line stands where the run opened
// rather than where the print now stands.
func open_spread(subject Printer_Handle, held Boolean) (spread Boolean) {
	defer func() { Boolean_Invariants(spread, "open_spread.spread") }()
	Printer_Handle_Invariants(subject, "open_spread.subject")
	Boolean_Invariants(held, "open_spread.held")
	place := int(declared_load(subject.Counts.Values))
	form := subject.Form.Values
	if run_lines(subject) != LINE_COUNT_MANY {
		return false
	}
	if !bool(held) {
		return true
	}
	// A run the author already stood apart from states the line the rule asks for, thus the
	// print opens no second one.
	if place < 2 {
		return true
	}
	if form[place-2] == '\n' {
		return true
	}
	open_line_at(subject)
	// The run that spans lines stands an empty line apart from the run ahead of it, thus the
	// byte the run opens at closes a line that holds nothing.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(form[place-1] == '\n',
			"A run that spans lines opens behind an empty line.")
	}
	return true
}

// LINE_COUNT_NONE names a run of declarations that closed no line of its own.
const LINE_COUNT_NONE = 0

// LINE_COUNT_ONE names one that closed one line.
const LINE_COUNT_ONE = 1

// LINE_COUNT_MANY names one that closed two lines or more, which is a run that spans lines.
const LINE_COUNT_MANY = 2

// Line_Count is how many lines one run of declarations closed, counted up to two.
type Line_Count uint8

// Line_Count_Invariants states every count of lines one run closes.
func Line_Count_Invariants(value Line_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(uint8(value), LINE_COUNT_NONE, LINE_COUNT_ONE, LINE_COUNT_MANY).
		Ensure()
}

// Reads how many lines the run of declarations the print wrote last closed, counting up to two.
// A run that spans lines stands apart from the run behind it, thus the read stops at the count
// that states the spread.
func run_lines(subject Printer_Handle) (count Line_Count) {
	defer func() { Line_Count_Invariants(count, "run_lines.count") }()
	Printer_Handle_Invariants(subject, "run_lines.subject")
	place := int(declared_load(subject.Counts.Values))
	written := int(written_load(subject.Counts.Values))
	// The run of declarations opened at a byte the print already wrote, thus the read of the
	// lines it spans walks the form and never past its end.
	aver.Always(place <= written, "A run of declarations opens inside the form.")
	form := subject.Form.Values
	count = LINE_COUNT_NONE
	for step := place; step < written; step = step + 1 {
		if form[step] == '\n' {
			count = count + 1
		}
		if count == LINE_COUNT_MANY {
			return count
		}
	}
	return count
}

// Opens one empty line at the byte the run of declarations the print wrote stands at, moving
// everything behind it one byte on.
func open_line_at(subject Printer_Handle) {
	Printer_Handle_Invariants(subject, "open_line_at.subject")
	place := int(declared_load(subject.Counts.Values))
	written := int(written_load(subject.Counts.Values))
	form := subject.Form.Values
	if written >= len(form) {
		flag_store(subject.Flags.Values, FLAG_FULL, true)
		return
	}
	// The line lands where the run of declarations opened, thus it stands inside the form the
	// print already wrote and the bytes behind it move one place along.
	aver.Always(place < written, "A line the print opens inside the form lands behind its end.")
	copy(form[place+1:written+1], form[place:written])
	form[place] = '\n'
	written_store(subject.Counts.Values, Form_Count(written+1))
	// The line the print opened stands in the form, thus the count of the lines it closed
	// reads that one along with every other feed.
	subject.Counts.Values[COUNT_LINE] = subject.Counts.Values[COUNT_LINE] + 1
}

// Reports whether the print stands behind an empty line, which is the line a run of declarations
// opens at.
func stands_after_blank(subject Printer_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "stands_after_blank.yes") }()
	Printer_Handle_Invariants(subject, "stands_after_blank.subject")
	written := int(written_load(subject.Counts.Values))
	if written < 2 {
		return false
	}
	form := subject.Form.Values
	if form[written-1] != '\n' {
		return false
	}
	return Boolean(form[written-2] == '\n')
}

// Reports whether one note opens with the word the slot names, behind the slashes.
func opens_word(
	subject Printer_Handle, tree ast.Syntax_Tree_Handle, word Reserved_Name,
) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "opens_word.yes") }()
	Printer_Handle_Invariants(subject, "opens_word.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "opens_word.tree")
	Reserved_Name_Invariants(word, "opens_word.word")
	cursor := ast.Syntax_Node_At(tree, subject.Nodes.Values[depth_load(subject.Counts.Values)])
	one := ast.Syntax_Token_At(tree, cursor.Token)
	text := token.Text(token.Source(subject.Source.Values), one)
	if len(text)-2 < len(word) {
		return false
	}
	for index := range len(word) {
		if text[2+index] != word[index] {
			return false
		}
	}
	return true
}

// Sets the flag one form prints under and hands back the flag standing before it, which the
// caller hands to a restore of its own. A body that closes up against its brackets drops the
// empty lines the author left there; a literal does the same, and a call does not, thus each
// form states its own answer rather than reading the one its parent stood under.
func hold_tight(subject Printer_Handle, tight Boolean) (held Boolean) {
	defer func() { Boolean_Invariants(held, "hold_tight.held") }()
	Printer_Handle_Invariants(subject, "hold_tight.subject")
	Boolean_Invariants(tight, "hold_tight.tight")
	held = flag_load(subject.Flags.Values, FLAG_TIGHT)
	flag_store(subject.Flags.Values, FLAG_TIGHT, tight)
	return held
}

// Marks the run of one operation the print breaks at every sign, and hands back the mark that
// stood before it, which the caller stands back.
func hold_split(subject Printer_Handle, split Boolean) (held Boolean) {
	defer func() { Boolean_Invariants(held, "hold_split.held") }()
	Printer_Handle_Invariants(subject, "hold_split.subject")
	Boolean_Invariants(split, "hold_split.split")
	held = flag_load(subject.Flags.Values, FLAG_SPLIT)
	flag_store(subject.Flags.Values, FLAG_SPLIT, split)
	return held
}

// Stands the split mark back where a form found it.
func restore_split(subject Printer_Handle, held Boolean) {
	Printer_Handle_Invariants(subject, "restore_split.subject")
	Boolean_Invariants(held, "restore_split.held")
	flag_store(subject.Flags.Values, FLAG_SPLIT, held)
}

// Stands the tight flag back where a form found it.
func restore_tight(subject Printer_Handle, held Boolean) {
	Printer_Handle_Invariants(subject, "restore_tight.subject")
	Boolean_Invariants(held, "restore_tight.held")
	flag_store(subject.Flags.Values, FLAG_TIGHT, held)
}

// Reports whether the node the walk stands on states an empty line standing against a brace,
// which is a line the canonical form drops: the body opens at its first statement and closes at
// its last one.
func stands_against(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "stands_against.yes") }()
	Printer_Handle_Invariants(subject, "stands_against.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "stands_against.tree")
	if node_kind(subject, tree) != ast.NODE_BLANK {
		return false
	}
	// Only a body that closes up against its brackets drops the line: every other body keeps
	// the empty lines the author wrote wherever the author wrote them. A form inside a body
	// states its own answer, thus a line deep inside a statement reads the body around it.
	if !bool(flag_load(subject.Flags.Values, FLAG_TIGHT)) {
		if !bool(closes_tight_body(subject, tree)) {
			return false
		}
	}
	index := subject.Nodes.Values[depth_load(subject.Counts.Values)]
	position := int(ast.Syntax_Node_At(tree, index).Token)
	// An empty line of the tree names the token the scanner counted it at, thus that token
	// carries the count of the lines standing ahead of it.
	aver.Always(ast.Syntax_Token_At(tree, ast.Token_Index(position)).Blanks > 0,
		"An empty line of the tree names a token the scanner counted lines at.")
	// The empty line names the token behind it, thus a closing bracket there states a line
	// against the close and an opening bracket ahead of it states a line against the open. A
	// semicolon the line feed itself stands for spans no byte and states no bracket either
	// way, thus the read steps over it.
	ahead := position
	for ahead < ast.TOKEN_INDEX_MAXIMUM {
		one := ast.Syntax_Token_At(tree, ast.Token_Index(ahead))
		if one.Size != 0 {
			break
		}
		if one.Kind != token.KIND_SEMICOLON {
			break
		}
		ahead = ahead + 1
	}
	// The end of the file spells no byte and closes no bracket, and a source the author left
	// open stands its forms against that end.
	closer := ast.Syntax_Token_At(tree, ast.Token_Index(ahead)).Kind
	if closer != token.KIND_END_OF_FILE {
		if bool(closes_bracket(token.Scanned_Kind(closer))) {
			return true
		}
	}
	if position == POSITION_MINIMUM {
		return false
	}
	behind := position - 1
	for behind > POSITION_MINIMUM {
		one := ast.Syntax_Token_At(tree, ast.Token_Index(behind))
		if one.Size != 0 {
			break
		}
		if one.Kind != token.KIND_SEMICOLON {
			break
		}
		behind = behind - 1
	}
	opener := ast.Syntax_Token_At(tree, ast.Token_Index(behind)).Kind
	if opener == token.KIND_END_OF_FILE {
		return false
	}
	return opens_bracket(token.Scanned_Kind(opener))
}

// Marks the empty line a field list of no fields closes on. The line the author left ahead of the
// next declaration stands inside the list the parser read, thus a list that writes no body of its
// own hands that line to the print that follows it.
func close_empty_list(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "close_empty_list.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "close_empty_list.tree")
	if !bool(descend(subject, tree)) {
		return
	}
	stood := written_load(subject.Counts.Values)
	open := Boolean(true)
	for bool(open) {
		if bool(closes_block(subject, tree)) {
			flag_store(subject.Flags.Values, FLAG_BLANK, true)
		}
		open = advance(subject, tree)
	}
	ascend(subject)
	// A list of no parts closes on the line it opened on, thus the walk over the empty lines
	// it holds writes no byte and takes the line it owes as a debt of the declaration.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(written_load(subject.Counts.Values) == stood,
			"A list of no parts writes no byte for the lines it holds.")
	}
}

// LAYOUT_RETURN ends the rule that runs.
const LAYOUT_RETURN Layout_Op = 0

// LAYOUT_DOWN moves the walk to the first child of the node it stands on, and tests whether one
// stands.
const LAYOUT_DOWN Layout_Op = 1

// LAYOUT_NEXT moves the walk to the sibling behind the node it stands on, and tests whether one
// stands.
const LAYOUT_NEXT Layout_Op = 2

// LAYOUT_UP moves the walk back to the node it descended from.
const LAYOUT_UP Layout_Op = 3

// LAYOUT_WHEN runs the step behind it only where the last test held.
const LAYOUT_WHEN Layout_Op = 4

// LAYOUT_UNLESS runs the step behind it only where the last test failed.
const LAYOUT_UNLESS Layout_Op = 5

// LAYOUT_JUMP resumes the program at the step the operand names.
const LAYOUT_JUMP Layout_Op = 6

// LAYOUT_JUMP_UNLESS resumes the program at the step the operand names where the last test failed.
const LAYOUT_JUMP_UNLESS Layout_Op = 7

// LAYOUT_CALL runs the rule the operand names over the node the walk stands on, handing it the
// last test, and resumes behind it with the test that rule ran last.
const LAYOUT_CALL Layout_Op = 8

// LAYOUT_FILE tests whether the node the walk stands on is a file.
const LAYOUT_FILE Layout_Op = 21

// LAYOUT_ALIASES reads the import names the print drops.
const LAYOUT_ALIASES Layout_Op = 22

// LAYOUT_OPENING writes one node ahead of the declarations and notes whether it was the package
// clause.
const LAYOUT_OPENING Layout_Op = 23

// LAYOUT_NAMED tests whether the walk wrote the package clause.
const LAYOUT_NAMED Layout_Op = 24

// LAYOUT_MOVES tests whether the declaration the walk stands on stands under a type instead.
const LAYOUT_MOVES Layout_Op = 25

// LAYOUT_OPEN_DECLARATION writes the empty line one declaration owes the one ahead of it.
const LAYOUT_OPEN_DECLARATION Layout_Op = 26

// LAYOUT_CLOSE_DECLARATION reads what the declaration the print wrote leaves for the one behind
// it, and tests whether it is a type, which carries the run of invariants that stands under it.
const LAYOUT_CLOSE_DECLARATION Layout_Op = 27

// LAYOUT_INVARIANT_RUN writes the run of invariants that stands under the type the walk stands on.
const LAYOUT_INVARIANT_RUN Layout_Op = 28

// LAYOUT_CLOSE_FILE closes the file on one line feed.
const LAYOUT_CLOSE_FILE Layout_Op = 29

// LAYOUT_HOLD_FORM holds the indent one form opens at and clears the break it inherits.
const LAYOUT_HOLD_FORM Layout_Op = 9

// LAYOUT_RESTORE_FORM stands the indent the rule held back and clears the break.
const LAYOUT_RESTORE_FORM Layout_Op = 10

// LAYOUT_IS tests whether the node the walk stands on is of the kind the operand names.
const LAYOUT_IS Layout_Op = 11

// LAYOUT_TOKEN writes the token the node the walk stands on names.
const LAYOUT_TOKEN Layout_Op = 12

// LAYOUT_WORD writes the keyword the operand names, which the tree drops.
const LAYOUT_WORD Layout_Op = 13

// LAYOUT_BYTE writes the byte the operand names.
const LAYOUT_BYTE Layout_Op = 14

// LAYOUT_INDENT writes the tabs the line the print opens owes.
const LAYOUT_INDENT Layout_Op = 15

// LAYOUT_LINE closes the line the print stands on.
const LAYOUT_LINE Layout_Op = 16

// LAYOUT_DOWN_PART moves the walk to the first child that states a part of the form, writing the
// trivia it steps over, and tests whether one stands.
const LAYOUT_DOWN_PART Layout_Op = 30

// LAYOUT_NEXT_PART moves the walk to the next sibling that states a part of the form, writing the
// trivia it steps over, and tests whether one stands.
const LAYOUT_NEXT_PART Layout_Op = 31

// LAYOUT_TAKE_TRIVIA steps past the node the walk stands on and reads every piece of trivia
// behind it into the line the print stands on.
const LAYOUT_TAKE_TRIVIA Layout_Op = 32

// LAYOUT_IMPORT_NAME writes the name an import binds, unless the print drops it.
const LAYOUT_IMPORT_NAME Layout_Op = 33

// LAYOUT_PASS writes the declaration the walk stands on through the recursive print, for the
// kinds no rule writes yet.
const LAYOUT_PASS Layout_Op = 34

// LAYOUT_VALUE_WORD writes the word the operand names, which opens one value declaration, unless
// the short sign binds it.
const LAYOUT_VALUE_WORD Layout_Op = 17

// LAYOUT_VALUES_OPEN opens the walk over the names, the type, and the values of a declaration.
const LAYOUT_VALUES_OPEN Layout_Op = 35

// LAYOUT_TRIVIA tests whether the node the walk stands on is a comment or an empty line.
const LAYOUT_TRIVIA Layout_Op = 36

// LAYOUT_VALUE_CHILD writes what stands ahead of one child of a value declaration, and tests
// whether the child is an expression rather than a name.
const LAYOUT_VALUE_CHILD Layout_Op = 37

// LAYOUT_EXPRESSION writes the expression the walk stands on through the recursive print.
const LAYOUT_EXPRESSION Layout_Op = 38

// LAYOUT_VALUES_NEXT steps to the child behind the one the print wrote, and tests whether one
// stands.
const LAYOUT_VALUES_NEXT Layout_Op = 39

// LAYOUT_TAKE_HERE reads the trivia the walk stands on, where the last test held, into the line.
const LAYOUT_TAKE_HERE Layout_Op = 40

// LAYOUT_TRUE makes the last test hold.
const LAYOUT_TRUE Layout_Op = 18

// LAYOUT_SIGNATURE_OVER reads whether the signature runs past the widest line.
const LAYOUT_SIGNATURE_OVER Layout_Op = 41

// LAYOUT_RESULTS_OVER reads whether the results run past the widest line, and whether they state
// one bare result.
const LAYOUT_RESULTS_OVER Layout_Op = 42

// LAYOUT_LIST_BEGIN opens a list that holds no part yet at the indent the line stands at.
const LAYOUT_LIST_BEGIN Layout_Op = 43

// LAYOUT_LIST_ITEM writes what stands ahead of one part of the list the operand names.
const LAYOUT_LIST_ITEM Layout_Op = 44

// LAYOUT_LIST_END closes the list the operand names.
const LAYOUT_LIST_END Layout_Op = 45

// LAYOUT_BODY writes the body of a function through the recursive print.
const LAYOUT_BODY Layout_Op = 46

// LAYOUT_PARTS_OPEN opens a run of names and a run of types that hold none yet.
const LAYOUT_PARTS_OPEN Layout_Op = 47

// LAYOUT_NAME_MARK writes the comma ahead of every name of a run but the first.
const LAYOUT_NAME_MARK Layout_Op = 48

// LAYOUT_NAMES_SPACE writes the space between a run of names and the type behind it.
const LAYOUT_NAMES_SPACE Layout_Op = 49

// LAYOUT_TYPE_MARK writes the comma ahead of every type of a run but the first.
const LAYOUT_TYPE_MARK Layout_Op = 50

// LAYOUT_STANDS_LAST tests whether no part stands behind the one the walk stands on.
const LAYOUT_STANDS_LAST Layout_Op = 51

// LAYOUT_FINISH_PARTS steps past every part and piece of trivia the walk still stands ahead of.
const LAYOUT_FINISH_PARTS Layout_Op = 52

// LAYOUT_PARAMETER_MARK writes the bracket ahead of the first type parameter of a type
// declaration or the comma ahead of any other.
const LAYOUT_PARAMETER_MARK Layout_Op = 53

// LAYOUT_PARAMETERS_CLOSE writes the bracket that closes the type parameters of a type
// declaration where it states one.
const LAYOUT_PARAMETERS_CLOSE Layout_Op = 54

// LAYOUT_OP_MINIMUM is the first action the program holds.
const LAYOUT_OP_MINIMUM = uint8(LAYOUT_RETURN)

// LAYOUT_SPANS_ONE_LINE tests whether the block the walk stands on reads on one line.
const LAYOUT_SPANS_ONE_LINE Layout_Op = 55

// LAYOUT_INLINE_BLOCK writes one block whose statements stand on the line its brace opens.
const LAYOUT_INLINE_BLOCK Layout_Op = 56

// LAYOUT_BLOCK_OPEN reads whether the block closes up against its braces, holding the tightness
// of the form around it back.
const LAYOUT_BLOCK_OPEN Layout_Op = 57

// LAYOUT_ENTERED notes whether the walk stepped into the block.
const LAYOUT_ENTERED Layout_Op = 58

// LAYOUT_TRAILS_BRACE tests whether the note the walk stands on stands on the line the brace opens.
const LAYOUT_TRAILS_BRACE Layout_Op = 59

// LAYOUT_TRIVIA_HERE holds the comment or the empty line the walk stands on until the line closes.
const LAYOUT_TRIVIA_HERE Layout_Op = 20

// LAYOUT_INDENT_IN indents every line behind it one tab deeper.
const LAYOUT_INDENT_IN Layout_Op = 19

// LAYOUT_BLOCK_SKIP tests whether the block writes no line for the node the walk stands on: an
// empty line behind the brace that closes it, or one standing against a brace.
const LAYOUT_BLOCK_SKIP Layout_Op = 60

// LAYOUT_COLUMN_OPEN reads the column the notes of the run of statements the walk stands in open
// at, and whether a note trails the statement.
const LAYOUT_COLUMN_OPEN Layout_Op = 61

// LAYOUT_STATEMENT writes the statement the walk stands on through the recursive print.
const LAYOUT_STATEMENT Layout_Op = 62

// LAYOUT_COLUMN_CLOSE closes the run of notes where the statement carried none.
const LAYOUT_COLUMN_CLOSE Layout_Op = 63

// LAYOUT_BLOCK_CLOSE closes one block: its indent, the walk, and the brace on a line of its own.
const LAYOUT_BLOCK_CLOSE Layout_Op = 64

// LAYOUT_OP_MAXIMUM is the last action the program holds.
const LAYOUT_OP_MAXIMUM = uint8(LAYOUT_BLOCK_CLOSE)

// Layout_Op is one action of the layout program.
type Layout_Op uint8

// Layout_Op_Invariants states every action the program holds.
func Layout_Op_Invariants(value Layout_Op, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), LAYOUT_OP_MINIMUM, LAYOUT_OP_MAXIMUM).
		Ensure()
}

// Layout_Rule is one rule of the layout program.
type Layout_Rule uint8

// Layout_Rule_Invariants states every rule the program holds.
func Layout_Rule_Invariants(value Layout_Rule, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), LAYOUT_RULE_MINIMUM, LAYOUT_RULE_MAXIMUM).
		Ensure()
}

// Layout_Frame is one rule of the program standing open over one node.
type Layout_Frame struct {
	// Place is the step of the program the rule reads next.
	Place Layout_Place
	// Holds is whether the last test held.
	Holds Boolean
	// File is what a walk over the declarations of one file carries from one to the next.
	File File_Walk
	// Values is what a walk over the children of a value declaration carries.
	Values Values_Walk
	// List is the layout of the list the rule writes.
	List List_Walk
	// Parts is what a walk over a run of names and a run of types carries.
	Parts Parts_Walk
	// Block is what a walk over the statements of one block carries.
	Block Block_Walk
}

// Layout_Frame_Invariants composes one open rule.
func Layout_Frame_Invariants(value Layout_Frame, namespace aver.Namespace) {
	Layout_Place_Invariants(value.Place, namespace)
	Boolean_Invariants(value.Holds, namespace)
	File_Walk_Invariants(value.File, namespace)
	Values_Walk_Invariants(value.Values, namespace)
	List_Walk_Invariants(value.List, namespace)
	Parts_Walk_Invariants(value.Parts, namespace)
	Block_Walk_Invariants(value.Block, namespace)
}

// Block_Entered distinguishes a block the walk stepped into from one that holds no node.
type Block_Entered Boolean

// Block_Entered_Invariants states both states of the walk.
func Block_Entered_Invariants(value Block_Entered, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The walk stands inside the block.").
		Ensure()
}

// Block_Trail distinguishes a block that carries an empty line behind its brace.
type Block_Trail Boolean

// Block_Trail_Invariants states both endings of a block.
func Block_Trail_Invariants(value Block_Trail, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "An empty line stands behind the block.").
		Ensure()
}

// Block_Trails distinguishes a statement a note trails on the line it closes.
type Block_Trails Boolean

// Block_Trails_Invariants states both endings of a statement.
func Block_Trails_Invariants(value Block_Trails, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "A note trails the statement.").
		Ensure()
}

// Block_Held distinguishes a form around the block that closes up against its brackets.
type Block_Held Boolean

// Block_Held_Invariants states both forms around a block.
func Block_Held_Invariants(value Block_Held, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The form around the block stands tight.").
		Ensure()
}

// Block_Walk is what a walk over the statements of one block carries.
type Block_Walk struct {
	// Entered reports whether the walk stepped into the block.
	Entered Block_Entered
	// Trail reports whether an empty line stands behind the block.
	Trail Block_Trail
	// Trails reports whether a note trails the statement the walk stands on.
	Trails Block_Trails
	// Held is the tightness of the form around the block, which the block stands back.
	Held Block_Held
}

// Block_Walk_Invariants composes one walk over a block.
func Block_Walk_Invariants(value Block_Walk, namespace aver.Namespace) {
	Block_Entered_Invariants(value.Entered, namespace)
	Block_Trail_Invariants(value.Trail, namespace)
	Block_Trails_Invariants(value.Trails, namespace)
	Block_Held_Invariants(value.Held, namespace)
}

// List_Started distinguishes a list that holds a part from one that holds none yet.
type List_Started Boolean

// List_Started_Invariants states both states of a list.
func List_Started_Invariants(value List_Started, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The list holds a part.").
		Ensure()
}

// List_Opened distinguishes a list whose parts stand on lines of their own.
type List_Opened Boolean

// List_Opened_Invariants states both layouts of the parts.
func List_Opened_Invariants(value List_Opened, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The parts stand on lines of their own.").
		Ensure()
}

// List_Broken distinguishes a list that stands broken across lines.
type List_Broken Boolean

// List_Broken_Invariants states both layouts of a list.
func List_Broken_Invariants(value List_Broken, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The list stands broken.").
		Ensure()
}

// List_Tail distinguishes a list whose closing bracket the author left on a line of its own.
type List_Tail Boolean

// List_Tail_Invariants states both places of the closing bracket.
func List_Tail_Invariants(value List_Tail, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The closing bracket stands on a line of its own.").
		Ensure()
}

// List_Carries distinguishes a list that carries the brace of a body on the line it closes.
type List_Carries Boolean

// List_Carries_Invariants states both endings of a list.
func List_Carries_Invariants(value List_Carries, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The list carries the brace of the body.").
		Ensure()
}

// List_Over distinguishes a list that runs past the widest line.
type List_Over Boolean

// List_Over_Invariants states both widths of a list.
func List_Over_Invariants(value List_Over, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The list runs past the widest line.").
		Ensure()
}

// List_Bare distinguishes a list of one result that binds no name.
type List_Bare Boolean

// List_Bare_Invariants states both forms of the results.
func List_Bare_Invariants(value List_Bare, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The results stand bare.").
		Ensure()
}

// List_Walk is the layout of one list a rule writes. The indent the list opened at lives in the
// run of the rule, as any other local does.
type List_Walk struct {
	// Started reports whether the list holds a part.
	Started List_Started
	// Opened reports whether the parts stand on lines of their own.
	Opened List_Opened
	// Broken reports whether the list stands broken.
	Broken List_Broken
	// Tail reports whether the closing bracket stands on a line of its own.
	Tail List_Tail
	// Carries reports whether the list carries the brace of a body.
	Carries List_Carries
	// Over reports whether the list runs past the widest line.
	Over List_Over
	// Bare reports whether the results stand bare.
	Bare List_Bare
}

// List_Walk_Invariants composes the layout of one list.
func List_Walk_Invariants(value List_Walk, namespace aver.Namespace) {
	List_Started_Invariants(value.Started, namespace)
	List_Opened_Invariants(value.Opened, namespace)
	List_Broken_Invariants(value.Broken, namespace)
	List_Tail_Invariants(value.Tail, namespace)
	List_Carries_Invariants(value.Carries, namespace)
	List_Over_Invariants(value.Over, namespace)
	List_Bare_Invariants(value.Bare, namespace)
}

// List_Walk_Handle gives the layout of one list one identity.
type List_Walk_Handle *List_Walk

// List_Walk_Handle_Invariants composes a present layout.
func List_Walk_Handle_Invariants(value List_Walk_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	List_Walk_Invariants(*value, namespace)
}

// Part_Names distinguishes a run of names that holds a name.
type Part_Names Boolean

// Part_Names_Invariants states both states of the run of names.
func Part_Names_Invariants(value Part_Names, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The run holds a name.").
		Ensure()
}

// Part_Types distinguishes a run of types that holds a type.
type Part_Types Boolean

// Part_Types_Invariants states both states of the run of types.
func Part_Types_Invariants(value Part_Types, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The run holds a type.").
		Ensure()
}

// Parts_Walk is what a walk over a run of names and a run of types carries.
type Parts_Walk struct {
	// Names reports whether the run of names holds a name.
	Names Part_Names
	// Types reports whether the run of types holds a type.
	Types Part_Types
}

// Parts_Walk_Invariants composes one walk over names and types.
func Parts_Walk_Invariants(value Parts_Walk, namespace aver.Namespace) {
	Part_Names_Invariants(value.Names, namespace)
	Part_Types_Invariants(value.Types, namespace)
}

// Parts_Walk_Handle gives one walk over names and types one identity.
type Parts_Walk_Handle *Parts_Walk

// Parts_Walk_Handle_Invariants composes a present walk.
func Parts_Walk_Handle_Invariants(value Parts_Walk_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Parts_Walk_Invariants(*value, namespace)
}

// Layout_Frame_Handle gives one open rule one identity.
type Layout_Frame_Handle *Layout_Frame

// Layout_Frame_Handle_Invariants composes a present open rule.
func Layout_Frame_Handle_Invariants(value Layout_Frame_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Layout_Frame_Invariants(*value, namespace)
}

// Named distinguishes a walk that wrote the package clause from one still ahead of it.
type Named Boolean

// Named_Invariants states both sides of the package clause.
func Named_Invariants(value Named, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The package clause stands written.").
		Ensure()
}

// Commented distinguishes a declaration a note opens from one that opens a run of its own.
type Commented Boolean

// Commented_Invariants states both openings of a declaration.
func Commented_Invariants(value Commented, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "A note opened the declaration.").
		Ensure()
}

// Spread_Run distinguishes a run of declarations that spans more than one line.
type Spread_Run Boolean

// Spread_Run_Invariants states both spans of a run.
func Spread_Run_Invariants(value Spread_Run, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The run spans lines.").
		Ensure()
}

// File_Walk is what a walk over the declarations of one file carries from one to the next.
type File_Walk struct {
	// Named reports whether the package clause stands written.
	Named Named
	// Commented reports whether a note opened the declaration the walk stands on.
	Commented Commented
	// Spread reports whether the run the print wrote last spans more than one line.
	Spread Spread_Run
	// Previous names the kind of the declaration the print wrote last.
	Previous Declared_Kind
	// Run reports whether the walk stands inside a run of notes it already read.
	Run Notes_Run
	// Moved reports whether the run of notes the walk stands inside stands under a type.
	Moved Moved_Notes
}

// File_Walk_Invariants composes one walk over the declarations of a file.
func File_Walk_Invariants(value File_Walk, namespace aver.Namespace) {
	Named_Invariants(value.Named, namespace)
	Commented_Invariants(value.Commented, namespace)
	Spread_Run_Invariants(value.Spread, namespace)
	Declared_Kind_Invariants(value.Previous, namespace)
	Notes_Run_Invariants(value.Run, namespace)
	Moved_Notes_Invariants(value.Moved, namespace)
}

// DECLARED_OTHER names a declaration of any other kind, which is what stands ahead of the first
// declaration of a file and what a refused tree holds.
const DECLARED_OTHER = 0

// DECLARED_IMPORT names one import.
const DECLARED_IMPORT = 1

// DECLARED_CONSTANT names one constant declaration.
const DECLARED_CONSTANT = 2

// DECLARED_VARIABLE names one variable declaration.
const DECLARED_VARIABLE = 3

// DECLARED_TYPE names one type declaration.
const DECLARED_TYPE = 4

// DECLARED_FUNCTION names one function declaration.
const DECLARED_FUNCTION = 5

// Declared_Kind names the kind of one declaration of a file, which is what a run of declarations
// holds one of.
type Declared_Kind uint8

// Declared_Kind_Invariants states every kind a declaration of a file states.
func Declared_Kind_Invariants(value Declared_Kind, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), DECLARED_OTHER, DECLARED_FUNCTION).
		Ensure()
}

// Notes_Run distinguishes a walk that stands inside a run of notes it read from one that
// stands on the note that opens a run.
type Notes_Run Boolean

// Notes_Run_Invariants states both stands of the walk.
func Notes_Run_Invariants(value Notes_Run, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The walk stands inside a run of notes it read.").
		Ensure()
}

// Moved_Notes distinguishes a run of notes that stands under a type from one that stands where
// the author wrote it.
type Moved_Notes Boolean

// Moved_Notes_Invariants states both stands of a run of notes.
func Moved_Notes_Invariants(value Moved_Notes, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The run of notes stands under a type.").
		Ensure()
}

// Names the kind of one declaration of a file, which is how a run holds declarations of one kind.
func declared_kind(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (declared Declared_Kind) {
	defer func() { Declared_Kind_Invariants(declared, "declared_kind.declared") }()
	Printer_Handle_Invariants(subject, "declared_kind.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "declared_kind.tree")
	switch node_kind(subject, tree) {
	case ast.NODE_IMPORT:
		return DECLARED_IMPORT
	case ast.NODE_CONSTANT:
		return DECLARED_CONSTANT
	case ast.NODE_VARIABLE:
		return DECLARED_VARIABLE
	case ast.NODE_TYPE:
		return DECLARED_TYPE
	case ast.NODE_FUNCTION:
		return DECLARED_FUNCTION
	}
	return DECLARED_OTHER
}

// File_Walk_Handle gives one walk over a file one identity.
type File_Walk_Handle *File_Walk

// File_Walk_Handle_Invariants composes a present walk.
func File_Walk_Handle_Invariants(value File_Walk_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	File_Walk_Invariants(*value, namespace)
}

// Value_Named distinguishes a child that stands in the run of names.
type Value_Named Boolean

// Value_Named_Invariants states both sides of the run of names.
func Value_Named_Invariants(value Value_Named, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The child stands in the run of names.").
		Ensure()
}

// Value_Valued distinguishes a child that stands in the run of values.
type Value_Valued Boolean

// Value_Valued_Invariants states both sides of the run of values.
func Value_Valued_Invariants(value Value_Valued, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The child stands in the run of values.").
		Ensure()
}

// Value_Names distinguishes a run of names the print already wrote a name of.
type Value_Names Boolean

// Value_Names_Invariants states both states of the run of names.
func Value_Names_Invariants(value Value_Names, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "A name stands written.").
		Ensure()
}

// Value_Values distinguishes a run of values the print already wrote a value of.
type Value_Values Boolean

// Value_Values_Invariants states both states of the run of values.
func Value_Values_Invariants(value Value_Values, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "A value stands written.").
		Ensure()
}

// Values_Walk is what a walk over the children of a value declaration carries: the names of a
// declaration stand in one run and its values in another, thus each run states of its own
// whether the child the print writes stands behind a child of it.
type Values_Walk struct {
	// Named reports whether the child stands in the run of names.
	Named Value_Named
	// Valued reports whether the child stands in the run of values.
	Valued Value_Valued
	// Names reports whether a name stands written.
	Names Value_Names
	// Values reports whether a value stands written.
	Values Value_Values
}

// Values_Walk_Invariants composes one walk over a value declaration.
func Values_Walk_Invariants(value Values_Walk, namespace aver.Namespace) {
	Value_Named_Invariants(value.Named, namespace)
	Value_Valued_Invariants(value.Valued, namespace)
	Value_Names_Invariants(value.Names, namespace)
	Value_Values_Invariants(value.Values, namespace)
}

// Values_Walk_Handle gives one walk over a value declaration one identity.
type Values_Walk_Handle *Values_Walk

// Values_Walk_Handle_Invariants composes a present walk.
func Values_Walk_Handle_Invariants(value Values_Walk_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Values_Walk_Invariants(*value, namespace)
}

// BYTE_SHIFT is how far one byte of a pair stands above the byte behind it, which is how a read
// holds two bytes of the form as one value.
const BYTE_SHIFT = 8

