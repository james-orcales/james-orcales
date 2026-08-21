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
// next, thus a tree stands as deep as it holds nodes and the walk answers for every one of them.
const DEPTH_MAXIMUM = ast.NODE_COUNT_MAXIMUM

// FIELD_COUNT_MAXIMUM caps the fields one alignment run holds.
const FIELD_COUNT_MAXIMUM = 1024

// WIDTH_MAXIMUM is the widest name one alignment run answers for.
const WIDTH_MAXIMUM = token.SOURCE_SIZE_MAXIMUM

// COLUMN_WIDTH_MAXIMUM is the widest line the canonical form writes. A form that would run past it
// states one part to a line instead, and a line that holds one token past it stands as it is,
// because no break of any kind shortens one word.
const COLUMN_WIDTH_MAXIMUM = 100

// TAB_WIDTH is the columns one tab stands in, which is what a reader of the form sees and what the
// linter counts.
const TAB_WIDTH = 8

// CHARACTER_TAIL_MARK marks the bytes of one character that stand behind its first, which state
// the two high bits of a continuation and stand in no column of their own.
const CHARACTER_TAIL_MARK = 0x80

// CHARACTER_TAIL_MASK reads the two high bits one byte states.
const CHARACTER_TAIL_MASK = 0xC0

// CASE_WIDTH_MAXIMUM is the widest head a case writes on one line. A head wider than this reads
// no better closed up than broken, thus the print keeps the lines the author wrote for it.
const CASE_WIDTH_MAXIMUM = 60

// SPAN_FOUND holds the width one measure answered with.
const SPAN_FOUND = 0

// SPAN_COLUMN holds the column one run of lines aligns its notes at.
const SPAN_COLUMN = 1

// SPAN_LINE holds the columns the line the print stands on already holds.
const SPAN_LINE = 2

// SPAN_SLOT_COUNT is the width slot count one printer holds.
const SPAN_SLOT_COUNT = 3

// POSITION_FOUND holds the run position one search answered with.
const POSITION_FOUND = 0

// POSITION_FROM holds the run position one scan opens at.
const POSITION_FROM = 1

// POSITION_TO holds the run position one scan closes at.
const POSITION_TO = 2

// POSITION_CHECK holds the token one simple error check opens at, which names the empty line the
// canonical form drops. No check opens at the first token of a file, thus that position states
// that no check stands ahead at all.
const POSITION_CHECK = 3

// POSITION_BOUND holds the token the values of one assignment open at, which names the empty line
// the canonical form drops between the sign and the value it binds.
const POSITION_BOUND = 4

// POSITION_SLOT_COUNT is the run position slot count one printer holds.
const POSITION_SLOT_COUNT = 5

// POSITION_MINIMUM is the first token of the run.
const POSITION_MINIMUM = 0

// POSITION_MAXIMUM is the last token the run admits.
const POSITION_MAXIMUM = ast.TOKEN_INDEX_MAXIMUM

// OFFSET_FROM holds the byte one read of the source opens at.
const OFFSET_FROM = 0

// OFFSET_TO holds the byte one read of the source closes at.
const OFFSET_TO = 1

// OFFSET_SLOT_COUNT is the source byte slot count one printer holds.
const OFFSET_SLOT_COUNT = 2

// KIND_CLAUSE holds the class of the clause the print stands inside of.
const KIND_CLAUSE = 0

// KIND_DECLARATION holds the class of the declaration the print wrote last.
const KIND_DECLARATION = 1

// KIND_SLOT_COUNT is the node class slot count one printer holds.
const KIND_SLOT_COUNT = 2

// MARK_SIGN holds the token of the sign one operation states between its values.
const MARK_SIGN = 0

// MARK_TEXT holds the token whose text the print writes.
const MARK_TEXT = 1

// MARK_SLOT_COUNT is the token slot count one printer holds.
const MARK_SLOT_COUNT = 2

// SIGN_HELD holds the class of the token one reader stands on.
const SIGN_HELD = 0

// SIGN_BEHIND holds the class of the token that stands behind the one a reader stands on.
const SIGN_BEHIND = 1

// SIGN_SLOT_COUNT is the token class slot count one printer holds.
const SIGN_SLOT_COUNT = 2

// WORD_SLOT holds the keyword the form the print stands on opens with.
const WORD_SLOT = 0

// WORD_SLOT_COUNT is the keyword slot count one printer holds.
const WORD_SLOT_COUNT = 1

// WORD_DIRECTIVE holds the name one read of a note asks that note to open with.
const WORD_DIRECTIVE = 0

// DIRECTIVE_SLOT_COUNT is the directive name slot count one printer holds.
const DIRECTIVE_SLOT_COUNT = 1

// COUNT_MAXIMUM caps every counter one print keeps, and the form is the widest thing it counts.
const COUNT_MAXIMUM = FORM_SIZE_MAXIMUM

// COUNT_WRITTEN holds how many bytes the print wrote.
const COUNT_WRITTEN = 0

// COUNT_DEPTH holds how deep the walk stands.
const COUNT_DEPTH = 1

// COUNT_INDENT holds how many tabs a line opens with.
const COUNT_INDENT = 2

// COUNT_RUN holds how many fields the alignment run ahead holds.
const COUNT_RUN = 3

// COUNT_FIELD holds which field of the run the print stands on.
const COUNT_FIELD = 4

// COUNT_WIDTH holds the widest name the alignment run holds.
const COUNT_WIDTH = 5

// COUNT_COMMENT holds how many comments the line the print stands on carries.
const COUNT_COMMENT = 6

// COUNT_NEST holds how deep inside another form the expression the print stands on stands. A
// statement opens at one, and the canonical form closes up the signs of an operation that stands
// deeper than that.
const COUNT_NEST = 7

// COUNT_TYPE holds the widest type the alignment run holds, which is the column a tag opens at.
// Zero names a run no field of which states a tag.
const COUNT_TYPE = 8

// COUNT_COLUMN holds the column the comments of the run the print stands in open at. Zero names
// a print stand in no run of comments.
const COUNT_COLUMN = 9

// COUNT_OPENING holds how many bytes stood written when the line the print stands on opened.
const COUNT_OPENING = 10

// COUNT_MARK holds how many semicolons the source states between two run positions.
const COUNT_MARK = 11

// COUNT_COLUMN_INDENT holds the tabs the form the print stands in opened its line at.
const COUNT_COLUMN_INDENT = 12

// COUNT_PART holds how many parts the form the print measured holds.
const COUNT_PART = 13

// COUNT_INNER holds the nest the parts inside the brackets of one form stand at.
const COUNT_INNER = 14

// COUNT_HEAD holds the nest the head of one form stands at.
const COUNT_HEAD = 15

// COUNT_SPAN holds how many bytes one measure wrote.
const COUNT_SPAN = 16

// COUNT_DIGIT holds the byte of one number literal its digits open at.
const COUNT_DIGIT = 17

// COUNT_DECLARED holds the byte of the form the run of declarations the print stands on opened
// at, which is where the empty line between two spread declarations stands.
const COUNT_DECLARED = 18

// COUNT_NAME holds how many names of one assignment the print writes, which is every name the
// author wrote except the ones a range clause binds to nothing.
const COUNT_NAME = 19

// COUNT_BOUND holds how many bounds of one slice the print writes, which is every bound the
// author wrote except a closing bound that states the length of the value the slice reads.
const COUNT_BOUND = 20

// COUNT_RESULT holds how many result names the signature the print stands inside of binds, which
// is what a return that names no value writes.
const COUNT_RESULT = 21

// COUNT_COLUMNS holds the columns one read of the form counted, which is what a reader of the line
// sees rather than the bytes the storage spends.
const COUNT_COLUMNS = 22

// COUNT_SPLIT holds the binding of the run of operations the print breaks at every sign, which is
// what says whether an operation inside it stands in that run or opens one of its own.
const COUNT_SPLIT = 23

// COUNT_ALIAS holds how many import names the print drops, which the run of names it holds states.
const COUNT_ALIAS = 24

// COUNT_SLOT_COUNT is the counter count one printer holds.
const COUNT_SLOT_COUNT = 25

// INVARIANTS_TAIL is the text the name of a function that states the invariants of a type closes
// with. A type states its invariants in one function of its own name, thus the canonical form
// stands that function under the type it answers for.
const INVARIANTS_TAIL = "_Invariants"

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

// TYPE_ELEMENT holds the tree slot of the type each element of the literal the print stands in
// wears, and no slot at all where the literal names no type its elements repeat.
const TYPE_ELEMENT = 0

// TYPE_KEY holds the tree slot of the type each key of that literal wears.
const TYPE_KEY = 1

// TYPE_LEFT holds one of the two forms a comparison of types reads.
const TYPE_LEFT = 2

// TYPE_RIGHT holds the other form that comparison reads.
const TYPE_RIGHT = 3

// TYPE_NAMED holds the tree slot of the declaration one read of the file stands on.
const TYPE_NAMED = 4

// TYPE_FOUND holds the tree slot one search of the file answered with.
const TYPE_FOUND = 5

// TYPE_OWN holds the tree slot of the type one literal names for the elements it holds.
const TYPE_OWN = 6

// TYPE_SLOT_COUNT is the type slot count one printer holds.
const TYPE_SLOT_COUNT = 7

// FORM_SLOT holds the storage the print writes into.
const FORM_SLOT = 0

// FORM_SLOT_COUNT is the storage slot count one printer holds.
const FORM_SLOT_COUNT = 1

// SOURCE_SLOT holds the source the tree stands for.
const SOURCE_SLOT = 0

// SOURCE_TEXT holds the bytes of the token the print writes, which the writing reads over and
// over and the tree answers for once.
const SOURCE_TEXT = 1

// SOURCE_NAME holds the bytes of the name one declaration binds.
const SOURCE_NAME = 2

// SOURCE_SLOT_COUNT is the source slot count one printer holds.
const SOURCE_SLOT_COUNT = 3

// FLAG_FULL marks a print that met the end of the storage the caller supplied.
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

// FLAG_COUNT is the flag count one printer holds.
const FLAG_COUNT = 11

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

// Printer is the whole print. Every field is an array, and every cursor lives in an array slot
// rather than in a field of its own, because a cursor field would owe its whole domain at every
// step that receives the state and a step deep in one declaration can never see a cursor at zero.
type Printer struct {
	// Nodes holds the tree slot the walk stands on at each depth.
	Nodes [DEPTH_MAXIMUM]ast.Index
	// Widths holds the name width of each field of the alignment run ahead.
	Widths [FIELD_COUNT_MAXIMUM]Width
	// Spans holds the widths one measure answers with, which a caller reads rather than
	// takes, because a width the print states in a signature owes a domain no source reaches.
	Spans [SPAN_SLOT_COUNT]Width
	// Positions holds the run positions one search answers with and one scan runs between.
	Positions [POSITION_SLOT_COUNT]Position
	// Offsets holds the source bytes one read of the source runs between.
	Offsets [OFFSET_SLOT_COUNT]token.Offset
	// Kinds holds the class of the form the print stands inside of.
	Kinds [KIND_SLOT_COUNT]ast.Node_Kind
	// Types holds the tree slots of the type forms one elision of a repeated type reads.
	Types [TYPE_SLOT_COUNT]ast.Index
	// Marks holds the run positions of the tokens the print writes of its own.
	Marks [MARK_SLOT_COUNT]ast.Token_Index
	// Results holds the tokens of the result names the signature the print stands inside of
	// binds, which a return that names no value writes.
	Results [RESULT_COUNT_MAXIMUM]ast.Token_Index
	// Aliases holds the name tokens of the imports whose name the print drops.
	Aliases [IMPORT_COUNT_MAXIMUM]ast.Token_Index
	// Paths holds the path token of each of those imports, which names the package the forms
	// that read the name now name.
	Paths [IMPORT_COUNT_MAXIMUM]ast.Token_Index
	// Signs holds the classes of the tokens one reader of the run stands between.
	Signs [SIGN_SLOT_COUNT]token.Kind
	// Words holds the keyword one form opens with, which the tree drops.
	Words [WORD_SLOT_COUNT]Word
	// Directives holds the name one read of a note asks that note to open with.
	Directives [DIRECTIVE_SLOT_COUNT]string
	// Comments holds the run position of each comment the line the print stands on carries.
	Comments [COMMENT_COUNT_MAXIMUM]Position
	// Forms holds the storage the print writes into.
	Forms [FORM_SLOT_COUNT]Form
	// Sources holds the source the tree stands for.
	Sources [SOURCE_SLOT_COUNT]token.Source
	// Counts holds every counter and cursor the print keeps.
	Counts [COUNT_SLOT_COUNT]Count
	// Flags holds what the print met: full storage, deep nest, and a fresh line.
	Flags [FLAG_COUNT]Boolean
}

// Printer_Invariants states the storage the caller supplies. Each cursor is an array slot, thus
// it proves its own domain where a body reads it.
func Printer_Invariants(subject *Printer, namespace aver.Namespace) {
	aver.Always(
		len(subject.Nodes) == DEPTH_MAXIMUM,
		"A printer holds one walk slot for every admitted depth.",
	)
}

// Print writes the canonical form of one tree into caller storage and reports whether the whole
// form fit. A tree the parser refused still prints what it holds, because a caller reads more
// from a partial print than from nothing.
func Print(
	subject *Printer, destination Form, tree *ast.Parse_State, source token.Source,
) (count Form_Count, ok Boolean) {
	defer func() {
		Form_Count_Invariants(count, "print.count")
		Boolean_Invariants(ok, "print.ok")
	}()
	Printer_Invariants(subject, "print.subject")
	Form_Invariants(destination, "print.destination")
	ast.Parse_State_Invariants(tree, "print.tree")
	token.Source_Invariants(source, "print.source")
	subject.Forms[FORM_SLOT] = destination
	subject.Sources[SOURCE_SLOT] = source
	for slot := range subject.Counts {
		subject.Counts[slot] = 0
	}
	for slot := range subject.Types {
		subject.Types[slot] = ast.INDEX_ABSENT
	}
	// An expression a declaration or a statement states stands at one, which is the nest
	// the canonical form spaces every sign at.
	subject.Counts[COUNT_NEST] = 1
	subject.Flags[FLAG_FULL] = false
	subject.Flags[FLAG_DEEP] = false
	subject.Flags[FLAG_LINE] = true
	subject.Nodes[0] = TREE_ROOT
	print_file(subject, tree)
	held := !subject.Flags[FLAG_FULL]
	held = held && !subject.Flags[FLAG_DEEP]
	return Form_Count(subject.Counts[COUNT_WRITTEN]), held
}

// TREE_ROOT is the arena slot the parser writes the file node into, thus a caller hands over the
// tree alone and never the root it already knows.
const TREE_ROOT ast.Index = 1

// Writes one byte into the storage. A storage the byte no longer fits marks the print full, thus
// every later write is a test and never a write past the array.
func emit_byte(subject *Printer, value Symbol) {
	Symbol_Invariants(value, "emit_byte.value")
	Printer_Invariants(subject, "emit_byte.subject")
	written := int(subject.Counts[COUNT_WRITTEN])
	if written >= len(subject.Forms[FORM_SLOT]) {
		subject.Flags[FLAG_FULL] = true
		return
	}
	subject.Forms[FORM_SLOT][written] = byte(value)
	subject.Counts[COUNT_WRITTEN] = Count(written + 1)
	subject.Flags[FLAG_LINE] = false
}

// Writes one space.
func emit_space(subject *Printer) {
	Printer_Invariants(subject, "emit_space.subject")
	emit_byte(subject, ' ')
}

// Writes one line feed and marks the line that follows as still unopened.
func emit_line(subject *Printer) {
	Printer_Invariants(subject, "emit_line.subject")
	emit_byte(subject, '\n')
	subject.Flags[FLAG_LINE] = true
	subject.Counts[COUNT_OPENING] = subject.Counts[COUNT_WRITTEN]
}

// Writes the tabs the open line owes, which is one for each depth the walk stands under. A line
// that already holds a byte owes none, thus a caller writes the indent before anything else.
func emit_indent(subject *Printer) {
	Printer_Invariants(subject, "emit_indent.subject")
	if !bool(subject.Flags[FLAG_LINE]) {
		return
	}
	written := int(subject.Counts[COUNT_WRITTEN])
	indent_count := int(subject.Counts[COUNT_INDENT])
	available_count := len(subject.Forms[FORM_SLOT]) - written
	if available_count < indent_count {
		indent_count = available_count
		subject.Flags[FLAG_FULL] = true
	}
	for index := range indent_count {
		subject.Forms[FORM_SLOT][written+index] = '\t'
	}
	if indent_count > 0 {
		subject.Counts[COUNT_WRITTEN] = Count(written + indent_count)
		subject.Flags[FLAG_LINE] = false
	}
}

// Writes the source bytes the token of the node the walk stands on spans.
func emit_token(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "emit_token.subject")
	ast.Parse_State_Invariants(tree, "emit_token.tree")
	node := ast.Node_At(tree, subject.Nodes[subject.Counts[COUNT_DEPTH]])
	subject.Marks[MARK_TEXT] = node.Token
	emit_text(subject, tree)
}

// Writes the text of the token the slot names. A number states its base and its exponent in the
// case the canonical form states them, thus the print writes the literal the source means rather
// than the bytes the author typed.
func emit_text(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "emit_text.subject")
	ast.Parse_State_Invariants(tree, "emit_text.tree")
	one := ast.Token_At(tree, subject.Marks[MARK_TEXT])
	subject.Sources[SOURCE_TEXT] = token.Text(subject.Sources[SOURCE_SLOT], one)
	kind := one.Kind
	number := Boolean(false)
	if kind == token.KIND_INTEGER {
		number = true
	}
	if kind == token.KIND_FLOAT {
		number = true
	}
	if kind == token.KIND_IMAGINARY {
		number = true
	}
	if bool(number) {
		emit_number(subject, tree)
		return
	}
	if kind == token.KIND_COMMENT {
		emit_note(subject)
		return
	}
	text := subject.Sources[SOURCE_TEXT]
	for index := range len(text) {
		emit_byte(subject, Symbol(text[index]))
	}
}

// Holds one comment or one empty line until the line it stands behind closes. The parser hangs
// each one behind the node whose span it follows, thus a form that still owes a bracket writes
// that bracket first and the trivia behind it.
func print_trivia(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "print_trivia.subject")
	ast.Parse_State_Invariants(tree, "print_trivia.tree")
	kind := node_kind(subject, tree)
	if kind == ast.NODE_BLANK {
		// A body opens at its first part and closes at its last one, thus an empty line
		// standing against the bracket on either side states nothing at all.
		if bool(stands_against(subject, tree)) {
			return
		}
		if bool(stands_before_check(subject, tree)) {
			return
		}
		if bool(stands_behind_sign(subject, tree)) {
			return
		}
		subject.Flags[FLAG_BLANK] = true
		return
	}
	if kind != ast.NODE_COMMENT {
		return
	}
	if int(subject.Counts[COUNT_COMMENT]) >= COMMENT_COUNT_MAXIMUM {
		return
	}
	node := ast.Node_At(tree, subject.Nodes[subject.Counts[COUNT_DEPTH]])
	subject.Comments[subject.Counts[COUNT_COMMENT]] = Position(node.Token)
	subject.Counts[COUNT_COMMENT] = subject.Counts[COUNT_COMMENT] + 1
	// An empty line ahead of a note belongs to that note, and the writing of the note reads it
	// from the source, thus the line the print still owes is the one behind every note.
	subject.Flags[FLAG_BLANK] = false
}

// Closes the line the print stands on: the comments it carries stand at its end, and the empty
// line it owes stands behind it.
func close_line(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "close_line.subject")
	ast.Parse_State_Invariants(tree, "close_line.tree")
	flush_comments(subject, tree)
	emit_line(subject)
	if bool(subject.Flags[FLAG_BLANK]) {
		emit_line(subject)
		subject.Flags[FLAG_BLANK] = false
		subject.Counts[COUNT_WIDTH] = 0
	}
}

// Writes one comment: it takes the tabs of a line it opens, and one space where it follows what
// already stands on the line.
func emit_comment(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "emit_comment.subject")
	ast.Parse_State_Invariants(tree, "emit_comment.tree")
	position := subject.Positions[POSITION_FOUND]
	// The indent the print writes opens the line, thus the report the writing reads must be
	// the one that stood before it wrote anything.
	opening := subject.Flags[FLAG_LINE]
	if bool(opening) {
		emit_indent(subject)
	}
	// Every comment of one run opens at one column, thus a reader follows the notes down the
	// run rather than hunting for each one at the end of its line.
	if !bool(opening) {
		line_width(subject)
		subject.Spans[SPAN_COLUMN] = Width(subject.Counts[COUNT_COLUMN])
		emit_column(subject)
	}
	// A note the print holds until the line closes reads the same as one it writes where it
	// stands, thus both run through one writing and a note of a line of its own wraps here
	// too.
	one := ast.Token_At(tree, ast.Token_Index(position))
	subject.Sources[SOURCE_TEXT] = token.Text(subject.Sources[SOURCE_SLOT], one)
	emit_note(subject)
}

// Moves the walk to the first child that states a part of the form, and writes every comment and
// empty line it steps over. The parser hangs trivia behind the node its span follows, thus a walk
// over the parts of a form meets trivia anywhere and prints it where it stands.
func descend_part(subject *Printer, tree *ast.Parse_State) (ok Boolean) {
	defer func() { Boolean_Invariants(ok, "descend_part.ok") }()
	Printer_Invariants(subject, "descend_part.subject")
	ast.Parse_State_Invariants(tree, "descend_part.tree")
	if !bool(descend(subject, tree)) {
		return false
	}
	if bool(skip_trivia(subject, tree)) {
		return true
	}
	// A form of trivia alone holds no part, and a caller that reads no part takes no step
	// back, thus the step down closes here.
	ascend(subject)
	return false
}

// Moves the walk to the sibling that states the next part of the form, and writes every comment
// and empty line it steps over.
func advance_part(subject *Printer, tree *ast.Parse_State) (ok Boolean) {
	defer func() { Boolean_Invariants(ok, "advance_part.ok") }()
	Printer_Invariants(subject, "advance_part.subject")
	ast.Parse_State_Invariants(tree, "advance_part.tree")
	if !bool(advance(subject, tree)) {
		return false
	}
	return skip_trivia(subject, tree)
}

// Steps past every part and every piece of trivia the walk still stands ahead of, thus a form
// that stopped at its final part still holds the comments and empty lines behind it.
func finish_parts(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "finish_parts.subject")
	ast.Parse_State_Invariants(tree, "finish_parts.tree")
	for bool(advance_part(subject, tree)) {
		continue
	}
}

// Writes the trivia the walk stands on and steps past it, and reports whether a part stands where
// the walk stopped.
func skip_trivia(subject *Printer, tree *ast.Parse_State) (ok Boolean) {
	defer func() { Boolean_Invariants(ok, "skip_trivia.ok") }()
	Printer_Invariants(subject, "skip_trivia.subject")
	ast.Parse_State_Invariants(tree, "skip_trivia.tree")
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
func drain_trivia(subject *Printer, tree *ast.Parse_State, open Boolean) {
	Printer_Invariants(subject, "drain_trivia.subject")
	ast.Parse_State_Invariants(tree, "drain_trivia.tree")
	Boolean_Invariants(open, "drain_trivia.open")
	stand := open
	for bool(stand) {
		print_behind(subject, tree)
		stand = advance(subject, tree)
	}
}

// Writes one comment or one empty line that stands behind a line the print already closed.
func print_behind(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "print_behind.subject")
	ast.Parse_State_Invariants(tree, "print_behind.tree")
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
func node_kind(subject *Printer, tree *ast.Parse_State) (kind ast.Node_Kind) {
	defer func() { ast.Node_Kind_Invariants(kind, "node_kind.kind") }()
	Printer_Invariants(subject, "node_kind.subject")
	ast.Parse_State_Invariants(tree, "node_kind.tree")
	return ast.Node_At(tree, subject.Nodes[subject.Counts[COUNT_DEPTH]]).Kind
}

// Reads the byte count the token of the node the walk stands on spans.
func node_width(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "node_width.subject")
	ast.Parse_State_Invariants(tree, "node_width.tree")
	node := ast.Node_At(tree, subject.Nodes[subject.Counts[COUNT_DEPTH]])
	text := token.Text(subject.Sources[SOURCE_SLOT], ast.Token_At(tree, node.Token))
	subject.Spans[SPAN_FOUND] = Width(len(text))
}

// Moves the walk to the first child of the node it stands on.
func descend(subject *Printer, tree *ast.Parse_State) (ok Boolean) {
	defer func() { Boolean_Invariants(ok, "descend.ok") }()
	Printer_Invariants(subject, "descend.subject")
	ast.Parse_State_Invariants(tree, "descend.tree")
	child := ast.Node_At(tree, subject.Nodes[subject.Counts[COUNT_DEPTH]]).First_Child
	if ast.Index(child) == 0 {
		return false
	}
	if int(subject.Counts[COUNT_DEPTH])+1 >= DEPTH_MAXIMUM {
		subject.Flags[FLAG_DEEP] = true
		return false
	}
	subject.Counts[COUNT_DEPTH] = subject.Counts[COUNT_DEPTH] + 1
	subject.Nodes[subject.Counts[COUNT_DEPTH]] = ast.Index(child)
	return true
}

// Moves the walk to the sibling after the node it stands on.
func advance(subject *Printer, tree *ast.Parse_State) (ok Boolean) {
	defer func() { Boolean_Invariants(ok, "advance.ok") }()
	Printer_Invariants(subject, "advance.subject")
	ast.Parse_State_Invariants(tree, "advance.tree")
	next := ast.Node_At(tree, subject.Nodes[subject.Counts[COUNT_DEPTH]]).Next
	if ast.Index(next) == 0 {
		return false
	}
	subject.Nodes[subject.Counts[COUNT_DEPTH]] = ast.Index(next)
	return true
}

// Leaves the child chain the walk stands in.
func ascend(subject *Printer) {
	Printer_Invariants(subject, "ascend.subject")
	if subject.Counts[COUNT_DEPTH] == 0 {
		return
	}
	subject.Counts[COUNT_DEPTH] = subject.Counts[COUNT_DEPTH] - 1
}

// Writes one file: the package clause and each declaration behind it.
func print_file(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "print_file.subject")
	ast.Parse_State_Invariants(tree, "print_file.tree")
	// A print walks one file and the declarations it holds, thus a tree standing for anything
	// else writes nothing at all.
	if node_kind(subject, tree) != ast.NODE_FILE {
		return
	}
	take_aliases(subject, tree)
	if !bool(descend(subject, tree)) {
		return
	}
	// A comment ahead of the package clause is the doc of the file, and no empty line stands
	// between the two, thus the print holds back the blank the scanner counted there. Every
	// declaration behind the clause states its own trivia, thus this walk takes plain steps.
	open := Boolean(true)
	named := Boolean(false)
	// No declaration stands yet, thus the first one opens a run of its own and stands behind
	// the empty line the package clause closes on.
	subject.Kinds[KIND_DECLARATION] = ast.NODE_FILE
	commented := Boolean(false)
	// Two declarations that each span more than one line stand apart, thus the print reads
	// the lines the run it wrote spans and the lines the run behind it spanned.
	spread := Boolean(false)
	for bool(open) {
		if !bool(named) {
			named = print_opening(subject, tree)
			open = advance(subject, tree)
			continue
		}
		kind := node_kind(subject, tree)
		// A function that states the invariants of a type stands under that type, thus the
		// walk writes nothing where the author wrote it and the type writes it instead.
		if bool(moves_under_type(subject, tree)) {
			open = advance(subject, tree)
			continue
		}
		open_declaration(subject, tree, commented)
		if !bool(commented) {
			subject.Counts[COUNT_DECLARED] = subject.Counts[COUNT_WRITTEN]
		}
		print_declaration(subject, tree)
		commented = Boolean(kind == ast.NODE_COMMENT)
		if !bool(commented) {
			spread = open_spread(subject, spread)
		}
		if kind != ast.NODE_COMMENT {
			if kind != ast.NODE_BLANK {
				subject.Kinds[KIND_DECLARATION] = kind
			}
		}
		if kind == ast.NODE_TYPE {
			spread = print_invariant_run(subject, tree, spread)
		}
		open = advance(subject, tree)
	}
	close_file(subject)
	ascend(subject)
}

// Closes the file on one line feed. Each declaration carries the empty line that stood behind it,
// and the declaration the author wrote last carries none, thus a file whose last declaration now
// stands under a type would close on a line that states nothing.
func close_file(subject *Printer) {
	Printer_Invariants(subject, "close_file.subject")
	form := subject.Forms[FORM_SLOT]
	for subject.Counts[COUNT_WRITTEN] > 1 {
		written := int(subject.Counts[COUNT_WRITTEN])
		if written > len(form) {
			return
		}
		if form[written-1] != '\n' {
			return
		}
		if form[written-2] != '\n' {
			return
		}
		subject.Counts[COUNT_WRITTEN] = Count(written - 1)
	}
}

// Writes the run of declarations that states the invariants of the type the walk stands on, which
// is the function of that name and the notes above it. It reports whether the run it wrote spans
// more than one line, which is what the declaration behind it stands apart from.
func print_invariant_run(subject *Printer, tree *ast.Parse_State, spread Boolean) (held Boolean) {
	defer func() { Boolean_Invariants(held, "print_invariant_run.held") }()
	Printer_Invariants(subject, "print_invariant_run.subject")
	ast.Parse_State_Invariants(tree, "print_invariant_run.tree")
	Boolean_Invariants(spread, "print_invariant_run.spread")
	subject.Types[TYPE_NAMED] = subject.Nodes[subject.Counts[COUNT_DEPTH]]
	take_invariant_run(subject, tree)
	node := subject.Types[TYPE_FOUND]
	stand := subject.Nodes[subject.Counts[COUNT_DEPTH]]
	commented := Boolean(false)
	for node != ast.INDEX_ABSENT {
		subject.Nodes[subject.Counts[COUNT_DEPTH]] = node
		kind := node_kind(subject, tree)
		open_declaration(subject, tree, commented)
		if !bool(commented) {
			subject.Counts[COUNT_DECLARED] = subject.Counts[COUNT_WRITTEN]
		}
		print_declaration(subject, tree)
		commented = Boolean(kind == ast.NODE_COMMENT)
		if !bool(commented) {
			spread = open_spread(subject, spread)
			subject.Kinds[KIND_DECLARATION] = kind
			break
		}
		node = ast.Index(ast.Node_At(tree, node).Next)
	}
	subject.Nodes[subject.Counts[COUNT_DEPTH]] = stand
	return spread
}

// Reports whether the declaration the walk stands on stands under a type of the file rather than
// where the author wrote it: a function that states the invariants of such a type, or a note that
// opens one. The type carries both, thus the walk writes neither where it finds them.
func moves_under_type(subject *Printer, tree *ast.Parse_State) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "moves_under_type.yes") }()
	Printer_Invariants(subject, "moves_under_type.subject")
	ast.Parse_State_Invariants(tree, "moves_under_type.tree")
	one := subject.Nodes[subject.Counts[COUNT_DEPTH]]
	for range ast.NODE_COUNT_MAXIMUM {
		if ast.Node_At(tree, one).Kind != ast.NODE_COMMENT {
			break
		}
		one = ast.Index(ast.Node_At(tree, one).Next)
		if one == ast.INDEX_ABSENT {
			return false
		}
	}
	subject.Types[TYPE_NAMED] = one
	return states_own_type(subject, tree)
}

// Reports whether the declaration the named slot holds is a function that states the invariants
// of a type the file itself declares. A function that answers for a type of another file names no
// type to stand under, thus it stands where the author wrote it.
func states_own_type(subject *Printer, tree *ast.Parse_State) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "states_own_type.yes") }()
	Printer_Invariants(subject, "states_own_type.subject")
	ast.Parse_State_Invariants(tree, "states_own_type.tree")
	one := subject.Types[TYPE_NAMED]
	if one == ast.INDEX_ABSENT {
		return false
	}
	node := ast.Node_At(tree, one)
	if node.Kind != ast.NODE_FUNCTION {
		return false
	}
	child := ast.Index(ast.Node_At(tree, ast.Index(node.Parent)).First_Child)
	for range ast.NODE_COUNT_MAXIMUM {
		if child == ast.INDEX_ABSENT {
			return false
		}
		held := ast.Node_At(tree, child)
		if held.Kind == ast.NODE_TYPE {
			subject.Types[TYPE_LEFT] = child
			subject.Types[TYPE_RIGHT] = one
			if bool(names_pair(subject, tree)) {
				return true
			}
		}
		child = ast.Index(held.Next)
	}
	return false
}

// Reads the run of declarations that states the invariants of the type the named slot holds: the
// function of that name and the notes standing above it. A type that states none names no run at
// all, thus the found slot holds no node.
func take_invariant_run(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "take_invariant_run.subject")
	ast.Parse_State_Invariants(tree, "take_invariant_run.tree")
	named := subject.Types[TYPE_NAMED]
	subject.Types[TYPE_FOUND] = ast.INDEX_ABSENT
	if ast.Node_At(tree, named).Kind != ast.NODE_TYPE {
		return
	}
	head := ast.INDEX_ABSENT
	parent := ast.Index(ast.Node_At(tree, named).Parent)
	child := ast.Index(ast.Node_At(tree, parent).First_Child)
	for range ast.NODE_COUNT_MAXIMUM {
		if child == ast.INDEX_ABSENT {
			return
		}
		held := ast.Node_At(tree, child)
		if held.Kind == ast.NODE_COMMENT {
			if head == ast.INDEX_ABSENT {
				head = child
			}
			child = ast.Index(held.Next)
			continue
		}
		subject.Types[TYPE_LEFT] = named
		subject.Types[TYPE_RIGHT] = child
		paired := Boolean(held.Kind == ast.NODE_FUNCTION)
		if bool(paired) {
			paired = names_pair(subject, tree)
		}
		if bool(paired) {
			subject.Types[TYPE_FOUND] = child
			if head != ast.INDEX_ABSENT {
				subject.Types[TYPE_FOUND] = head
			}
			return
		}
		head = ast.INDEX_ABSENT
		child = ast.Index(held.Next)
	}
}

// Reports whether the function the right slot names states the invariants of the type the left
// slot names, which the name of that function says: the name of the type and the tail every such
// function closes with.
func names_pair(subject *Printer, tree *ast.Parse_State) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "names_pair.yes") }()
	Printer_Invariants(subject, "names_pair.subject")
	ast.Parse_State_Invariants(tree, "names_pair.tree")
	subject.Types[TYPE_NAMED] = subject.Types[TYPE_LEFT]
	take_name(subject, tree)
	named := subject.Sources[SOURCE_NAME]
	subject.Types[TYPE_NAMED] = subject.Types[TYPE_RIGHT]
	take_name(subject, tree)
	held := subject.Sources[SOURCE_NAME]
	if len(named) == 0 {
		return false
	}
	if len(held) != len(named)+len(INVARIANTS_TAIL) {
		return false
	}
	for index := range len(named) {
		if held[index] != named[index] {
			return false
		}
	}
	return Boolean(string(held[len(named):]) == INVARIANTS_TAIL)
}

// Reads the text of the name the declaration the named slot holds binds, which stands as the
// first child of that declaration. A declaration that binds no name states no text at all.
func take_name(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "take_name.subject")
	ast.Parse_State_Invariants(tree, "take_name.tree")
	subject.Sources[SOURCE_NAME] = nil
	one := subject.Types[TYPE_NAMED]
	if one == ast.INDEX_ABSENT {
		return
	}
	named := ast.Index(ast.Node_At(tree, one).First_Child)
	if named == ast.INDEX_ABSENT {
		return
	}
	held := ast.Node_At(tree, named)
	if held.Kind != ast.NODE_IDENTIFIER {
		return
	}
	source := subject.Sources[SOURCE_SLOT]
	subject.Sources[SOURCE_NAME] = token.Text(source, ast.Token_At(tree, held.Token))
}

// Writes one node that stands ahead of the declarations: the doc comment of the file, or the
// package clause that closes the opening. It reports whether the clause now stands written.
func print_opening(subject *Printer, tree *ast.Parse_State) (named Boolean) {
	defer func() { Boolean_Invariants(named, "print_opening.named") }()
	Printer_Invariants(subject, "print_opening.subject")
	ast.Parse_State_Invariants(tree, "print_opening.tree")
	kind := node_kind(subject, tree)
	// A file opens at its first line however many empty lines stand ahead of it, thus a blank
	// the print has written nothing before is no line of the form.
	if kind == ast.NODE_BLANK {
		if subject.Counts[COUNT_WRITTEN] == 0 {
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
	emit_token(subject, tree)
	close_line(subject, tree)
	return true
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
func emit_word(subject *Printer, word Word) {
	Printer_Invariants(subject, "emit_word.subject")
	Word_Invariants(word, "emit_word.word")
	text := "package"
	switch word {
	case WORD_IMPORT:
		text = "import"
	case WORD_CONSTANT:
		text = "const"
	case WORD_VARIABLE:
		text = "var"
	case WORD_TYPE:
		text = "type"
	case WORD_FUNCTION:
		text = "func"
	case WORD_STRUCTURE:
		text = "struct"
	case WORD_INTERFACE:
		text = "interface"
	case WORD_MAP:
		text = "map"
	case WORD_CHANNEL:
		text = "chan"
	case WORD_RANGE:
		text = "range"
	case WORD_ELSE:
		text = "else"
	case WORD_DEFAULT:
		text = "default"
	}
	for index := range len(text) {
		emit_byte(subject, Symbol(text[index]))
	}
}

// Reports whether the token before the node the walk stands on opens another name.
func opens_name(subject *Printer, tree *ast.Parse_State) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "opens_name.yes") }()
	Printer_Invariants(subject, "opens_name.subject")
	ast.Parse_State_Invariants(tree, "opens_name.tree")
	// A form names the token it opens with and never the token its own text begins at, thus
	// the test reads the token ahead of the leftmost one the form spans.
	leftmost_position(subject, tree)
	position := subject.Positions[POSITION_FOUND]
	if position == 0 {
		return false
	}
	kind := ast.Token_At(tree, ast.Token_Index(position-1)).Kind
	if kind == token.KIND_COMMA {
		return true
	}
	if kind == token.KIND_CONSTANT {
		return true
	}
	return kind == token.KIND_VARIABLE
}

// Reports whether one statement states a variable the short sign binds: a run of names, the
// values behind them, and no type at all. A declaration that wears a type states a form the short
// sign cannot hold, and a constant binds no short sign of any kind.
func states_short(subject *Printer, tree *ast.Parse_State) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "states_short.yes") }()
	Printer_Invariants(subject, "states_short.subject")
	ast.Parse_State_Invariants(tree, "states_short.tree")
	if node_kind(subject, tree) != ast.NODE_VARIABLE {
		return false
	}
	held := subject.Nodes[subject.Counts[COUNT_DEPTH]]
	depth := subject.Counts[COUNT_DEPTH]
	short := Boolean(false)
	open := descend(subject, tree)
	for bool(open) {
		// A note or an empty line the author left behind the values states no type,
		// thus the read steps over the trivia the declaration carries.
		if bool(trivia_kind(subject, tree)) {
			open = advance(subject, tree)
			continue
		}
		if bool(opens_value(subject, tree)) {
			short = true
		} else if !bool(opens_name(subject, tree)) {
			short = false
			break
		}
		open = advance(subject, tree)
	}
	subject.Counts[COUNT_DEPTH] = depth
	subject.Nodes[depth] = held
	return short
}

// Reports whether the token before the node the walk stands on opens the value of a declaration.
func opens_value(subject *Printer, tree *ast.Parse_State) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "opens_value.yes") }()
	Printer_Invariants(subject, "opens_value.subject")
	ast.Parse_State_Invariants(tree, "opens_value.tree")
	leftmost_position(subject, tree)
	position := subject.Positions[POSITION_FOUND]
	if position == 0 {
		return false
	}
	return ast.Token_At(tree, ast.Token_Index(position-1)).Kind == token.KIND_ASSIGN
}

// Writes the one declaration the walk stands on.
func print_declaration(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "print_declaration.subject")
	ast.Parse_State_Invariants(tree, "print_declaration.tree")
	// A declaration the author broke across lines indents its continuations, and the tabs it
	// owed stand again at the declaration behind it.
	held := subject.Counts[COUNT_INDENT]
	subject.Flags[FLAG_BROKEN] = false
	defer func() {
		subject.Counts[COUNT_INDENT] = held
		subject.Flags[FLAG_BROKEN] = false
	}()
	switch node_kind(subject, tree) {
	case ast.NODE_BLANK:
		close_line(subject, tree)
	case ast.NODE_COMMENT:
		emit_indent(subject)
		emit_token(subject, tree)
		close_line(subject, tree)
	case ast.NODE_IMPORT:
		print_import(subject, tree)
	case ast.NODE_CONSTANT:
		subject.Words[WORD_SLOT] = WORD_CONSTANT
		print_values(subject, tree)
	case ast.NODE_VARIABLE:
		subject.Words[WORD_SLOT] = WORD_VARIABLE
		print_values(subject, tree)
	case ast.NODE_TYPE:
		print_type_declaration(subject, tree)
	case ast.NODE_FUNCTION:
		print_function(subject, tree)
	default:
		print_statement(subject, tree)
	}
}

// Reports whether the name the walk stands on is one the print dropped from an import, and reads
// the name that import's path states into the name slot where it is. A name of any other form
// stands as the author wrote it, thus only the head of a selector answers here.
func names_alias(subject *Printer, tree *ast.Parse_State) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "names_alias.yes") }()
	Printer_Invariants(subject, "names_alias.subject")
	ast.Parse_State_Invariants(tree, "names_alias.tree")
	if node_kind(subject, tree) != ast.NODE_IDENTIFIER {
		return false
	}
	one := ast.Node_At(tree, subject.Nodes[subject.Counts[COUNT_DEPTH]]).Token
	source := subject.Sources[SOURCE_SLOT]
	text := token.Text(source, ast.Token_At(tree, one))
	for slot := range int(subject.Counts[COUNT_ALIAS]) {
		held := token.Text(source, ast.Token_At(tree, subject.Aliases[slot]))
		if string(held) != string(text) {
			continue
		}
		subject.Marks[MARK_TEXT] = subject.Paths[slot]
		take_package_name(subject, tree)
		return true
	}
	return false
}

// Writes the name one import path states, which the name slot holds.
func emit_package_name(subject *Printer) {
	Printer_Invariants(subject, "emit_package_name.subject")
	text := subject.Sources[SOURCE_NAME]
	for index := range len(text) {
		emit_byte(subject, Symbol(text[index]))
	}
}

// Reports whether the print drops the import name the walk stands on, which the run of names it
// read from the imports of the file states.
func drops_alias(subject *Printer, tree *ast.Parse_State) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "drops_alias.yes") }()
	Printer_Invariants(subject, "drops_alias.subject")
	ast.Parse_State_Invariants(tree, "drops_alias.tree")
	one := ast.Node_At(tree, subject.Nodes[subject.Counts[COUNT_DEPTH]]).Token
	for slot := range int(subject.Counts[COUNT_ALIAS]) {
		if subject.Aliases[slot] == one {
			return true
		}
	}
	return false
}

// Reads the name the path the text mark names states, which is the element that path closes with.
// A path closing at the default or the internal element names the element ahead of that one,
// because such a directory holds what its parent names rather than a package of its own.
func take_package_name(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "take_package_name.subject")
	ast.Parse_State_Invariants(tree, "take_package_name.tree")
	one := ast.Token_At(tree, subject.Marks[MARK_TEXT])
	text := token.Text(subject.Sources[SOURCE_SLOT], one)
	subject.Sources[SOURCE_NAME] = nil
	if len(text) < 3 {
		return
	}
	text = text[1 : len(text)-1]
	for range DEPTH_MAXIMUM {
		opening_count := len(text)
		for opening_count > 0 {
			if text[opening_count-1] == '/' {
				break
			}
			opening_count = opening_count - 1
		}
		held := text[opening_count:]
		if string(held) != DEFAULT_TAIL {
			if string(held) != INTERNAL_TAIL {
				subject.Sources[SOURCE_NAME] = held
				return
			}
		}
		if opening_count == 0 {
			break
		}
		text = text[:opening_count-1]
	}
	subject.Sources[SOURCE_NAME] = text
}

// Reads the imports of the file whose name the print drops. An import states a name the path
// already names, or a name the forms that read it can name by the path instead, thus the print
// writes the path alone. A name another import of the file already binds stands as the author
// wrote it, because two packages of one name name neither.
func take_aliases(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "take_aliases.subject")
	ast.Parse_State_Invariants(tree, "take_aliases.tree")
	subject.Counts[COUNT_ALIAS] = 0
	file := subject.Nodes[subject.Counts[COUNT_DEPTH]]
	child := ast.Index(ast.Node_At(tree, file).First_Child)
	for range ast.NODE_COUNT_MAXIMUM {
		if child == ast.INDEX_ABSENT {
			return
		}
		held := ast.Node_At(tree, child)
		if held.Kind == ast.NODE_IMPORT {
			subject.Types[TYPE_NAMED] = child
			hold_alias(subject, tree)
		}
		child = ast.Index(held.Next)
	}
}

// Holds the name of the import the named slot states where the print drops that name.
func hold_alias(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "hold_alias.subject")
	ast.Parse_State_Invariants(tree, "hold_alias.tree")
	one := subject.Types[TYPE_NAMED]
	name := ast.Index(ast.Node_At(tree, one).First_Child)
	if name == ast.INDEX_ABSENT {
		return
	}
	if ast.Node_At(tree, name).Kind != ast.NODE_IMPORT_NAME {
		return
	}
	path := ast.Index(ast.Node_At(tree, name).Next)
	if path == ast.INDEX_ABSENT {
		return
	}
	subject.Marks[MARK_TEXT] = ast.Node_At(tree, path).Token
	take_package_name(subject, tree)
	if len(subject.Sources[SOURCE_NAME]) == 0 {
		return
	}
	if bool(clobbers_name(subject, tree)) {
		return
	}
	count := int(subject.Counts[COUNT_ALIAS])
	if count >= IMPORT_COUNT_MAXIMUM {
		return
	}
	subject.Aliases[count] = ast.Node_At(tree, name).Token
	subject.Paths[count] = ast.Node_At(tree, path).Token
	subject.Counts[COUNT_ALIAS] = Count(count + 1)
}

// Reports whether another import of the file already binds the name the named slot's import would
// take once the print drops the name the author wrote for it.
func clobbers_name(subject *Printer, tree *ast.Parse_State) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "clobbers_name.yes") }()
	Printer_Invariants(subject, "clobbers_name.subject")
	ast.Parse_State_Invariants(tree, "clobbers_name.tree")
	named := subject.Sources[SOURCE_NAME]
	one := subject.Types[TYPE_NAMED]
	file := ast.Index(ast.Node_At(tree, one).Parent)
	child := ast.Index(ast.Node_At(tree, file).First_Child)
	for range ast.NODE_COUNT_MAXIMUM {
		if child == ast.INDEX_ABSENT {
			return false
		}
		held := ast.Node_At(tree, child)
		if held.Kind == ast.NODE_IMPORT {
			if child != one {
				// An import states its name where the author wrote one and
				// states the name of its path where the author wrote none.
				first := ast.Index(held.First_Child)
				if first == ast.INDEX_ABSENT {
					return false
				}
				node := ast.Node_At(tree, first)
				source := subject.Sources[SOURCE_SLOT]
				text := token.Text(source, ast.Token_At(tree, node.Token))
				if node.Kind != ast.NODE_IMPORT_NAME {
					subject.Marks[MARK_TEXT] = node.Token
					take_package_name(subject, tree)
					text = subject.Sources[SOURCE_NAME]
				}
				if string(text) == string(named) {
					return true
				}
			}
		}
		child = ast.Index(held.Next)
	}
	return false
}

// Writes one import: the word, the name the file binds where it states one, and the path.
func print_import(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "print_import.subject")
	ast.Parse_State_Invariants(tree, "print_import.tree")
	emit_indent(subject)
	emit_word(subject, WORD_IMPORT)
	emit_space(subject)
	if !bool(descend_part(subject, tree)) {
		close_line(subject, tree)
		return
	}
	if node_kind(subject, tree) == ast.NODE_IMPORT_NAME {
		// A name the path itself states, or one the forms that read it now name by the
		// path, states nothing the import needs, thus the print writes the path alone.
		if !bool(drops_alias(subject, tree)) {
			emit_token(subject, tree)
			emit_space(subject)
		}
		advance_part(subject, tree)
	}
	emit_token(subject, tree)
	take_trivia(subject, tree, advance(subject, tree))
	close_line(subject, tree)
	ascend(subject)
}

// Reads the trivia the walk stands ahead of into the line the print stands on. A note the author
// left on that line keeps it, thus the line closes holding everything the author wrote on it and
// a note of a line of its own still opens one.
func take_trivia(subject *Printer, tree *ast.Parse_State, open Boolean) {
	Printer_Invariants(subject, "take_trivia.subject")
	ast.Parse_State_Invariants(tree, "take_trivia.tree")
	Boolean_Invariants(open, "take_trivia.open")
	stand := open
	for bool(stand) {
		print_trivia(subject, tree)
		stand = advance(subject, tree)
	}
}

// Writes one constant or variable declaration: its names, the type they wear, and the values
// behind them.
func print_values(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "print_values.subject")
	ast.Parse_State_Invariants(tree, "print_values.tree")
	emit_indent(subject)
	if !bool(subject.Flags[FLAG_SHORT]) {
		emit_word(subject, subject.Words[WORD_SLOT])
		emit_space(subject)
	}
	open := descend(subject, tree)
	named := Boolean(true)
	valued := Boolean(false)
	// The names of a declaration stand in one run and its values stand in another, thus each
	// run states of its own whether the child the print writes stands behind a child of it.
	names := Boolean(false)
	values := Boolean(false)
	for bool(open) {
		if bool(trivia_kind(subject, tree)) {
			break
		}
		valued = valued || opens_value(subject, tree)
		behind := names
		if bool(valued) {
			behind = values
		}
		print_value_child(subject, tree, named, valued, behind)
		values = values || valued
		// The reports stand on their own lines rather than behind a conditional sign,
		// because a sign that stops short of a call leaves the print unread.
		one := names_one(named, valued)
		names = names || one
		open = advance(subject, tree)
		ahead := opens_name(subject, tree)
		named = named && open
		named = named && ahead
	}
	take_trivia(subject, tree, open)
	close_line(subject, tree)
	ascend(subject)
}

// Reports whether the node the walk stands on is a comment or an empty line.
func trivia_kind(subject *Printer, tree *ast.Parse_State) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "trivia_kind.yes") }()
	Printer_Invariants(subject, "trivia_kind.subject")
	ast.Parse_State_Invariants(tree, "trivia_kind.tree")
	kind := node_kind(subject, tree)
	if kind == ast.NODE_BLANK {
		return true
	}
	return kind == ast.NODE_COMMENT
}

// Writes one child of a value declaration: a name, the type behind the names, or a value.
func print_value_child(
	subject *Printer, tree *ast.Parse_State, named Boolean, valued Boolean, behind Boolean,
) {
	Printer_Invariants(subject, "print_value_child.subject")
	ast.Parse_State_Invariants(tree, "print_value_child.tree")
	Boolean_Invariants(named, "print_value_child.named")
	Boolean_Invariants(valued, "print_value_child.valued")
	Boolean_Invariants(behind, "print_value_child.behind")
	if bool(valued) {
		emit_separator(subject, !behind)
		subject.Counts[COUNT_NEST] = 1
		print_expression(subject, tree)
		return
	}
	if bool(named) {
		if bool(behind) {
			emit_byte(subject, ',')
			emit_space(subject)
		}
		emit_token(subject, tree)
		return
	}
	emit_space(subject)
	print_expression(subject, tree)
}

// Writes what stands between one value and the value or the name before it: the assignment sign
// opens the run and a comma holds the rest of it.
func emit_separator(subject *Printer, first Boolean) {
	Printer_Invariants(subject, "emit_separator.subject")
	Boolean_Invariants(first, "emit_separator.first")
	if bool(first) {
		emit_space(subject)
		if bool(subject.Flags[FLAG_SHORT]) {
			emit_byte(subject, ':')
		}
		emit_byte(subject, '=')
		emit_space(subject)
		return
	}
	emit_byte(subject, ',')
	emit_space(subject)
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

// Writes one type declaration: the word, the name, the parameters it states, and the type behind
// them.
func print_type_declaration(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "print_type_declaration.subject")
	ast.Parse_State_Invariants(tree, "print_type_declaration.tree")
	emit_indent(subject)
	emit_word(subject, WORD_TYPE)
	emit_space(subject)
	if !bool(descend_part(subject, tree)) {
		close_line(subject, tree)
		return
	}
	emit_token(subject, tree)
	open := advance_part(subject, tree)
	parameters := 0
	stated := false
	for bool(open) {
		kind := node_kind(subject, tree)
		if kind == ast.NODE_TYPE_PARAMETER {
			if parameters == 0 {
				emit_byte(subject, '[')
			}
			if parameters > 0 {
				emit_byte(subject, ',')
				emit_space(subject)
			}
			print_type_parameter(subject, tree)
			parameters = parameters + 1
			open = advance_part(subject, tree)
			continue
		}
		if parameters > 0 {
			emit_byte(subject, ']')
			parameters = 0
		}
		if !stated {
			emit_space(subject)
			print_expression(subject, tree)
			take_trivia(subject, tree, advance(subject, tree))
			close_line(subject, tree)
			stated = true
			open = Boolean(false)
			continue
		}
		print_behind(subject, tree)
		open = advance(subject, tree)
	}
	ascend(subject)
}

// Writes one type parameter and the bracket that opens the list where it stands first.
func print_type_parameter(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "print_type_parameter.subject")
	ast.Parse_State_Invariants(tree, "print_type_parameter.tree")
	// The tree names every part of one type parameter the same way, thus the parts ahead of
	// the last one state the names and the last one states the constraint they bind against.
	part_count(subject, tree)
	names := int(subject.Counts[COUNT_PART]) - 1
	if !bool(descend_part(subject, tree)) {
		return
	}
	written := 0
	open := Boolean(true)
	for bool(open) {
		if written >= names {
			break
		}
		if written > 0 {
			emit_byte(subject, ',')
			emit_space(subject)
		}
		emit_token(subject, tree)
		written = written + 1
		open = advance_part(subject, tree)
	}
	if bool(open) {
		if written > 0 {
			emit_space(subject)
		}
		print_expression(subject, tree)
	}
	finish_parts(subject, tree)
	ascend(subject)
}

// Writes one function declaration: the word, the receiver where it states one, the name, the
// parameters, the results, and the block behind them.
func print_function(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "print_function.subject")
	ast.Parse_State_Invariants(tree, "print_function.tree")
	emit_indent(subject)
	emit_word(subject, WORD_FUNCTION)
	emit_space(subject)
	if !bool(descend_part(subject, tree)) {
		close_line(subject, tree)
		return
	}
	if node_kind(subject, tree) == ast.NODE_RECEIVER {
		print_receiver(subject, tree)
		advance_part(subject, tree)
	}
	emit_token(subject, tree)
	open := advance_part(subject, tree)
	print_signature(subject, tree, open)
	close_line(subject, tree)
	ascend(subject)
}

// Writes the receiver of a method, which stands between the word and the name.
func print_receiver(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "print_receiver.subject")
	ast.Parse_State_Invariants(tree, "print_receiver.tree")
	emit_byte(subject, '(')
	if bool(descend_part(subject, tree)) {
		print_parameter(subject, tree)
		ascend(subject)
	}
	emit_byte(subject, ')')
	emit_space(subject)
}

// Writes the parameters, the results, and the block of one signature.
func print_signature(subject *Printer, tree *ast.Parse_State, open Boolean) {
	Printer_Invariants(subject, "print_signature.subject")
	ast.Parse_State_Invariants(tree, "print_signature.tree")
	Boolean_Invariants(open, "print_signature.open")
	// One signature binds the names a return of its body writes, and a signature inside that
	// body binds its own, thus each print holds the names it found and hands back the ones
	// the signature around it bound.
	held := subject.Results
	names := subject.Counts[COUNT_RESULT]
	subject.Counts[COUNT_RESULT] = 0
	defer func() {
		subject.Results = held
		subject.Counts[COUNT_RESULT] = names
	}()
	// A signature that runs the line past the widest one states one parameter to a line. The
	// results answer to the parameters, thus the print reads the whole signature once and
	// breaks the run that holds it.
	over := runs_over_signature(subject, tree, open)
	stand := open
	stand = print_signature_parameters(subject, tree, stand, over)
	stand = print_signature_results(subject, tree, stand)
	for bool(stand) {
		if node_kind(subject, tree) != ast.NODE_BLOCK {
			break
		}
		emit_space(subject)
		// The body one function states closes up against its braces however many
		// statements it holds.
		subject.Flags[FLAG_BODY] = true
		print_block(subject, tree)
		stand = advance(subject, tree)
	}
	if bool(stand) {
		close_line(subject, tree)
		drain_trivia(subject, tree, stand)
		subject.Flags[FLAG_LINE] = false
	}
}

// Writes the type parameters and the parameters of one signature, and reports where the walk
// stopped.
func print_signature_parameters(
	subject *Printer, tree *ast.Parse_State, open Boolean, over Boolean,
) (stand Boolean) {
	defer func() { Boolean_Invariants(stand, "print_signature_parameters.stand") }()
	Printer_Invariants(subject, "print_signature_parameters.subject")
	ast.Parse_State_Invariants(tree, "print_signature_parameters.tree")
	Boolean_Invariants(open, "print_signature_parameters.open")
	Boolean_Invariants(over, "print_signature_parameters.over")
	stand = print_type_parameters(subject, tree, open)
	emit_byte(subject, '(')
	arguments := 0
	broken := Boolean(false)
	opened := Boolean(false)
	tail := Boolean(false)
	carries := Boolean(false)
	indent := subject.Counts[COUNT_INDENT]
	for bool(stand) {
		if node_kind(subject, tree) != ast.NODE_PARAMETER {
			break
		}
		here := breaks_before(subject, tree) || over
		if arguments == 0 {
			carries = closes_signature(subject, tree) || over
			tail = closes_list(subject, tree)
			print_first_break(subject, tree, here)
			opened = here
			broken = here || tail
		}
		if arguments > 0 {
			open_break(subject, here, opened)
			opened = opened || here
			broken = broken || here
			print_element_break(subject, tree, here)
		}
		print_parameter(subject, tree)
		arguments = arguments + 1
		stand = advance_part(subject, tree)
	}
	if bool(broken) {
		subject.Counts[COUNT_COLUMN_INDENT] = indent
		// A signature the author broke across lines closes on a line of its own, thus the
		// brace of the body stands where a reader looks for it rather than behind the last
		// parameter.
		close_broken(subject, tree, tail || carries)
	}
	emit_byte(subject, ')')
	return stand
}

// Reports whether the bracket that closes the list the walk stands in the first part of stands on
// a line of its own. The list itself holds no node, thus the print counts brackets from the part
// it stands on rather than from a bracket a node names.
func closes_list(subject *Printer, tree *ast.Parse_State) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "closes_list.yes") }()
	Printer_Invariants(subject, "closes_list.subject")
	ast.Parse_State_Invariants(tree, "closes_list.tree")
	if bool(subject.Flags[FLAG_FLAT]) {
		return false
	}
	depth := 1
	leftmost_position(subject, tree)
	opening := int(subject.Positions[POSITION_FOUND])
	for position := opening; position <= ast.TOKEN_INDEX_MAXIMUM; position++ {
		kind := ast.Token_At(tree, ast.Token_Index(position)).Kind
		if kind == token.KIND_END_OF_FILE {
			return false
		}
		subject.Signs[SIGN_HELD] = kind
		if bool(opens_bracket(subject)) {
			depth = depth + 1
		}
		subject.Signs[SIGN_HELD] = kind
		if !bool(closes_bracket(subject)) {
			continue
		}
		depth = depth - 1
		if depth > 0 {
			continue
		}
		behind := ast.Token_At(tree, ast.Token_Index(position-1))
		ahead := ast.Token_At(tree, ast.Token_Index(position))
		subject.Offsets[OFFSET_FROM] = token.Offset(int(behind.Offset) + int(behind.Size))
		subject.Offsets[OFFSET_TO] = ahead.Offset
		return feeds(subject)
	}
	return false
}

// Reports whether the brace of a body stands on the line the bracket of the list the walk stands
// in the first part of closes. A signature the author broke across lines closes on a line of its
// own, thus a list that carries the brace of the body opens one line for that brace.
func closes_signature(subject *Printer, tree *ast.Parse_State) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "closes_signature.yes") }()
	Printer_Invariants(subject, "closes_signature.subject")
	ast.Parse_State_Invariants(tree, "closes_signature.tree")
	count := int(tree.Token_Cursors[ast.CURSOR_TOKEN_COUNT])
	depth := 1
	leftmost_position(subject, tree)
	closer := 0
	for position := int(subject.Positions[POSITION_FOUND]); position < count; position++ {
		kind := ast.Token_At(tree, ast.Token_Index(position)).Kind
		subject.Signs[SIGN_HELD] = kind
		if bool(opens_bracket(subject)) {
			depth = depth + 1
		}
		subject.Signs[SIGN_HELD] = kind
		if bool(closes_bracket(subject)) {
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
	// Every bracket the signature states behind the one that closed the list closes again
	// before the body opens, thus the read counts them and stops at the first brace that
	// stands outside them all.
	behind := ast.Token_At(tree, ast.Token_Index(closer))
	depth = 0
	for position := closer + 1; position < count; position++ {
		one := ast.Token_At(tree, ast.Token_Index(position))
		if one.Kind == token.KIND_BRACE_LEFT {
			if depth == 0 {
				from := int(behind.Offset) + int(behind.Size)
				subject.Offsets[OFFSET_FROM] = token.Offset(from)
				subject.Offsets[OFFSET_TO] = one.Offset
				return !feeds(subject)
			}
		}
		subject.Signs[SIGN_HELD] = one.Kind
		if bool(opens_bracket(subject)) {
			depth = depth + 1
		}
		subject.Signs[SIGN_HELD] = one.Kind
		if bool(closes_bracket(subject)) {
			depth = depth - 1
		}
	}
	return false
}

// Writes the results of one signature and reports where the walk stopped.
func print_signature_results(
	subject *Printer, tree *ast.Parse_State, open Boolean,
) (stand Boolean) {
	defer func() { Boolean_Invariants(stand, "print_signature_results.stand") }()
	Printer_Invariants(subject, "print_signature_results.subject")
	ast.Parse_State_Invariants(tree, "print_signature_results.tree")
	Boolean_Invariants(open, "print_signature_results.open")
	stand = open
	// One bare result needs no parentheses to read, and the canonical form writes none.
	// The parameters stand written by now, thus the results read the line they close and
	// break only where they themselves run past the widest one.
	over := runs_over_results(subject, tree, open)
	bare := bare_result(subject, tree, open)
	results := 0
	broken := Boolean(false)
	opened := Boolean(false)
	tail := Boolean(false)
	carries := Boolean(false)
	indent := subject.Counts[COUNT_INDENT]
	for bool(stand) {
		if node_kind(subject, tree) != ast.NODE_RESULT {
			break
		}
		here := breaks_before(subject, tree) || over
		if results == 0 {
			emit_space(subject)
			if !bool(bare) {
				emit_byte(subject, '(')
			}
			indent = subject.Counts[COUNT_INDENT]
			carries = closes_signature(subject, tree) || over
			tail = closes_list(subject, tree)
			print_first_break(subject, tree, here)
			opened = here
			broken = here || tail
		}
		if results > 0 {
			open_break(subject, here, opened)
			opened = opened || here
			broken = broken || here
			print_element_break(subject, tree, here)
		}
		take_result_names(subject, tree)
		print_parameter(subject, tree)
		results = results + 1
		stand = advance_part(subject, tree)
	}
	if bool(broken) {
		subject.Counts[COUNT_COLUMN_INDENT] = indent
		close_broken(subject, tree, tail || carries)
	}
	if results > 0 {
		if !bool(bare) {
			emit_byte(subject, ')')
		}
	}
	return stand
}

// Writes the result names the signature bound behind a return that names no value of its own. A
// statement of any other word names nothing to write, and a signature that binds no name states
// nothing either, thus both write the word alone.
func emit_result_names(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "emit_result_names.subject")
	ast.Parse_State_Invariants(tree, "emit_result_names.tree")
	if node_kind(subject, tree) != ast.NODE_RETURN {
		return
	}
	for slot := range int(subject.Counts[COUNT_RESULT]) {
		if slot > 0 {
			emit_byte(subject, ',')
		}
		emit_space(subject)
		subject.Marks[MARK_TEXT] = subject.Results[slot]
		emit_text(subject, tree)
	}
}

// Reads the names one result of a signature binds into the run a return writes for it. A name
// spelled as the blank names nothing a return can read, and a signature that binds more names
// than the run holds states a form no return writes, thus either one empties the run.
func take_result_names(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "take_result_names.subject")
	ast.Parse_State_Invariants(tree, "take_result_names.tree")
	if node_kind(subject, tree) != ast.NODE_RESULT {
		return
	}
	held := subject.Nodes[subject.Counts[COUNT_DEPTH]]
	depth := subject.Counts[COUNT_DEPTH]
	open := descend_quiet(subject, tree)
	for bool(open) {
		if node_kind(subject, tree) != ast.NODE_PARAMETER_NAME {
			break
		}
		if !bool(holds_result_name(subject, tree)) {
			subject.Counts[COUNT_RESULT] = 0
			break
		}
		open = advance_quiet(subject, tree)
	}
	subject.Counts[COUNT_DEPTH] = depth
	subject.Nodes[depth] = held
}

// Writes the name the walk stands on into the run of result names, and reports whether the run
// still holds every name the signature bound.
func holds_result_name(subject *Printer, tree *ast.Parse_State) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "holds_result_name.yes") }()
	Printer_Invariants(subject, "holds_result_name.subject")
	ast.Parse_State_Invariants(tree, "holds_result_name.tree")
	names := int(subject.Counts[COUNT_RESULT])
	if names >= RESULT_COUNT_MAXIMUM {
		return false
	}
	one := ast.Node_At(tree, subject.Nodes[subject.Counts[COUNT_DEPTH]]).Token
	source := subject.Sources[SOURCE_SLOT]
	if string(token.Text(source, ast.Token_At(tree, one))) == "_" {
		return false
	}
	subject.Results[names] = one
	subject.Counts[COUNT_RESULT] = Count(names + 1)
	return true
}

// Reports whether one signature states a single result that binds no name. Parentheses around
// such a result state nothing the reader needs, thus the canonical form drops them.
func bare_result(subject *Printer, tree *ast.Parse_State, open Boolean) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "bare_result.yes") }()
	Printer_Invariants(subject, "bare_result.subject")
	ast.Parse_State_Invariants(tree, "bare_result.tree")
	Boolean_Invariants(open, "bare_result.open")
	if !bool(open) {
		return false
	}
	if node_kind(subject, tree) != ast.NODE_RESULT {
		return false
	}
	if bool(named_result(subject, tree)) {
		return false
	}
	held := subject.Nodes[subject.Counts[COUNT_DEPTH]]
	alone := Boolean(true)
	for bool(advance_quiet(subject, tree)) {
		if node_kind(subject, tree) == ast.NODE_RESULT {
			alone = false
			break
		}
	}
	subject.Nodes[subject.Counts[COUNT_DEPTH]] = held
	return alone
}

// Reports whether the result the walk stands on binds a name.
func named_result(subject *Printer, tree *ast.Parse_State) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "named_result.yes") }()
	Printer_Invariants(subject, "named_result.subject")
	ast.Parse_State_Invariants(tree, "named_result.tree")
	if !bool(descend_quiet(subject, tree)) {
		return false
	}
	held := Boolean(node_kind(subject, tree) == ast.NODE_PARAMETER_NAME)
	ascend(subject)
	return held
}

// Writes one parameter or result: the name it states and the type behind it.
func print_parameter(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "print_parameter.subject")
	ast.Parse_State_Invariants(tree, "print_parameter.tree")
	if !bool(descend_part(subject, tree)) {
		return
	}
	// One parameter binds a run of names against one type, thus the print writes every name
	// the run holds and the type once behind them.
	names := 0
	open := Boolean(true)
	for bool(open) {
		if node_kind(subject, tree) != ast.NODE_PARAMETER_NAME {
			break
		}
		if names > 0 {
			emit_byte(subject, ',')
			emit_space(subject)
		}
		emit_token(subject, tree)
		names = names + 1
		open = advance_part(subject, tree)
	}
	if names > 0 {
		if bool(open) {
			emit_space(subject)
		}
	}
	// A signature that binds no name states a run of bare types where a named one states a
	// run of names, thus the parts behind the names read as a run rather than as one type.
	types := 0
	for bool(open) {
		if types > 0 {
			emit_byte(subject, ',')
			emit_space(subject)
		}
		print_expression(subject, tree)
		types = types + 1
		open = advance_part(subject, tree)
	}
	finish_parts(subject, tree)
	ascend(subject)
}

// Writes one block: the brace that opens it, its statements one tab deeper, and the brace that
// closes it on a line of its own.
func print_block(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "print_block.subject")
	ast.Parse_State_Invariants(tree, "print_block.tree")
	if bool(spans_one_line(subject, tree)) {
		print_inline_block(subject, tree)
		return
	}
	closer_position(subject, tree)
	subject.Positions[POSITION_FROM] = subject.Positions[POSITION_FOUND]
	// A body of one statement alone, and the body one function states, each close up against
	// their braces; every other block keeps the lines the author wrote.
	tight := subject.Flags[FLAG_BODY] || holds_one_part(subject, tree)
	subject.Flags[FLAG_BODY] = false
	defer restore_tight(subject, hold_tight(subject, tight))
	emit_byte(subject, '{')
	open := descend(subject, tree)
	entered := open
	// A note the author left on the line the brace opens keeps that line, thus the print
	// writes it before it closes the line rather than on a line of its own.
	for bool(open) && bool(trails_brace(subject, tree)) {
		print_trivia(subject, tree)
		open = advance(subject, tree)
	}
	close_line(subject, tree)
	subject.Counts[COUNT_INDENT] = subject.Counts[COUNT_INDENT] + 1
	trail := Boolean(false)
	for bool(open) {
		// An empty line the scanner counted at a token the brace stands ahead of falls
		// inside the block, and one it counted at a token behind the brace falls
		// behind it. The token each empty line names states which side it fell on.
		if bool(closes_block(subject, tree)) {
			if bool(stands_behind(subject, tree)) {
				trail = true
				open = advance(subject, tree)
				continue
			}
		}
		// A body opens at its first statement, thus an empty line standing against the
		// brace it opens with states nothing and no line is written.
		if bool(stands_against(subject, tree)) {
			open = advance(subject, tree)
			continue
		}
		// The statements that each carry a note behind them stand in one run, and every
		// note of that run opens at one column.
		if subject.Counts[COUNT_COLUMN] == 0 {
			take_statement_column(subject, tree)
		}
		trails := trails_comment(subject, tree) && !spans_form(subject, tree)
		print_statement(subject, tree)
		if !bool(trails) {
			subject.Counts[COUNT_COLUMN] = 0
		}
		open = advance(subject, tree)
	}
	subject.Counts[COUNT_COLUMN] = 0
	if bool(entered) {
		ascend(subject)
	}
	subject.Counts[COUNT_INDENT] = subject.Counts[COUNT_INDENT] - 1
	// The empty line a statement carried behind it stands against the closing brace, thus the
	// body closes at its last statement and that line is never written.
	subject.Flags[FLAG_BLANK] = false
	emit_indent(subject)
	emit_byte(subject, '}')
	if bool(trail) {
		subject.Flags[FLAG_BLANK] = true
	}
}

// Reads the token the simple error check behind the statement the walk stands on opens at, and
// names the first token of the file where no such check stands. The author writes the check of
// an error against the call that states it, thus the empty line between the two states nothing.
func take_check(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "take_check.subject")
	ast.Parse_State_Invariants(tree, "take_check.tree")
	subject.Positions[POSITION_CHECK] = POSITION_MINIMUM
	if !bool(assigns_error(subject, tree)) {
		return
	}
	held := subject.Nodes[subject.Counts[COUNT_DEPTH]]
	depth := subject.Counts[COUNT_DEPTH]
	// A note between the two forms names the check rather than the value it stands behind,
	// thus the walk steps over the trivia to the statement the author wrote next.
	open := advance(subject, tree)
	for bool(open) {
		if !bool(trivia_kind(subject, tree)) {
			break
		}
		open = advance(subject, tree)
	}
	if bool(open) {
		if bool(states_check(subject, tree)) {
			node := ast.Node_At(tree, subject.Nodes[depth])
			subject.Positions[POSITION_CHECK] = Position(node.Token)
		}
	}
	subject.Counts[COUNT_DEPTH] = depth
	subject.Nodes[depth] = held
}

// Reports whether the statement the walk stands on binds one error: an assignment of any run of
// names whose last name spells the error, which is the run the check behind it reads.
func assigns_error(subject *Printer, tree *ast.Parse_State) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "assigns_error.yes") }()
	Printer_Invariants(subject, "assigns_error.subject")
	ast.Parse_State_Invariants(tree, "assigns_error.tree")
	kind := node_kind(subject, tree)
	if kind != ast.NODE_ASSIGN {
		if kind != ast.NODE_DEFINE {
			return false
		}
	}
	first := int(ast.Node_At(tree, subject.Nodes[subject.Counts[COUNT_DEPTH]]).Token)
	held := subject.Nodes[subject.Counts[COUNT_DEPTH]]
	depth := subject.Counts[COUNT_DEPTH]
	named := Boolean(false)
	open := descend(subject, tree)
	for bool(open) {
		if bool(opens_bound(subject, tree)) {
			// The sign stands behind the last name of the run, thus the name the
			// check reads stands two tokens ahead of the value it binds.
			position := int(subject.Positions[POSITION_FOUND]) - 2
			if position < first {
				break
			}
			name := ast.Token_At(tree, ast.Token_Index(position))
			if string(token.Text(subject.Sources[SOURCE_SLOT], name)) != "err" {
				break
			}
			named = Boolean(position == first)
			if !bool(named) {
				mark := ast.Token_At(tree, ast.Token_Index(position-1))
				named = Boolean(mark.Kind == token.KIND_COMMA)
			}
			break
		}
		open = advance(subject, tree)
	}
	subject.Counts[COUNT_DEPTH] = depth
	subject.Nodes[depth] = held
	return named
}

// Reads how many names of the assignment the walk stands on the print writes. A range clause
// binds the names the author wrote to values the body may read, thus a name that reads nothing
// states nothing and the canonical form drops it along with the sign it stood behind.
func take_names(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "take_names.subject")
	ast.Parse_State_Invariants(tree, "take_names.tree")
	held := subject.Nodes[subject.Counts[COUNT_DEPTH]]
	depth := subject.Counts[COUNT_DEPTH]
	names := 0
	ranged := Boolean(false)
	first := ast.Token_Index(0)
	last := ast.Token_Index(0)
	open := descend(subject, tree)
	for bool(open) {
		if bool(opens_bound(subject, tree)) {
			ranged = Boolean(node_kind(subject, tree) == ast.NODE_RANGE)
			break
		}
		if !bool(trivia_kind(subject, tree)) {
			one := subject.Nodes[subject.Counts[COUNT_DEPTH]]
			last = ast.Node_At(tree, one).Token
			if names == 0 {
				first = last
			}
			names = names + 1
		}
		open = advance(subject, tree)
	}
	subject.Counts[COUNT_DEPTH] = depth
	subject.Nodes[depth] = held
	if bool(ranged) {
		source := subject.Sources[SOURCE_SLOT]
		if names == 2 {
			if string(token.Text(source, ast.Token_At(tree, last))) == "_" {
				names = 1
			}
		}
		if names == 1 {
			if string(token.Text(source, ast.Token_At(tree, first))) == "_" {
				names = 0
			}
		}
	}
	subject.Counts[COUNT_NAME] = Count(names)
}

// Reads the token the values of the assignment the walk stands on open at, and names the first
// token of the file where the statement binds no values at all.
func take_bound(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "take_bound.subject")
	ast.Parse_State_Invariants(tree, "take_bound.tree")
	subject.Positions[POSITION_BOUND] = POSITION_MINIMUM
	held := subject.Nodes[subject.Counts[COUNT_DEPTH]]
	depth := subject.Counts[COUNT_DEPTH]
	open := descend(subject, tree)
	for bool(open) {
		if bool(opens_bound(subject, tree)) {
			subject.Positions[POSITION_BOUND] = subject.Positions[POSITION_FOUND]
			break
		}
		open = advance(subject, tree)
	}
	subject.Counts[COUNT_DEPTH] = depth
	subject.Nodes[depth] = held
}

// Reports whether the empty line the walk stands on stands between the sign of an assignment and
// the value that sign binds, which is a line that states nothing: the value stands on the line
// the sign closes.
func stands_behind_sign(subject *Printer, tree *ast.Parse_State) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "stands_behind_sign.yes") }()
	Printer_Invariants(subject, "stands_behind_sign.subject")
	ast.Parse_State_Invariants(tree, "stands_behind_sign.tree")
	if node_kind(subject, tree) != ast.NODE_BLANK {
		return false
	}
	if subject.Positions[POSITION_BOUND] == POSITION_MINIMUM {
		return false
	}
	node := ast.Node_At(tree, subject.Nodes[subject.Counts[COUNT_DEPTH]])
	return Boolean(Position(node.Token) == subject.Positions[POSITION_BOUND])
}

// Reports whether the token before the node the walk stands on binds the names of a run to their
// values, by either sign the source states for it.
func opens_bound(subject *Printer, tree *ast.Parse_State) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "opens_bound.yes") }()
	Printer_Invariants(subject, "opens_bound.subject")
	ast.Parse_State_Invariants(tree, "opens_bound.tree")
	leftmost_position(subject, tree)
	position := subject.Positions[POSITION_FOUND]
	if position == POSITION_MINIMUM {
		return false
	}
	kind := ast.Token_At(tree, ast.Token_Index(position-1)).Kind
	if kind == token.KIND_ASSIGN {
		return true
	}
	return kind == token.KIND_DEFINE
}

// Reports whether the statement the walk stands on states the simple check of one error: the
// words that read the error against nothing and the block behind them. An initializer, another
// branch, or a condition of any other spelling each state a check the empty line still opens.
func states_check(subject *Printer, tree *ast.Parse_State) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "states_check.yes") }()
	Printer_Invariants(subject, "states_check.subject")
	ast.Parse_State_Invariants(tree, "states_check.tree")
	if node_kind(subject, tree) != ast.NODE_IF {
		return false
	}
	position := int(ast.Node_At(tree, subject.Nodes[subject.Counts[COUNT_DEPTH]]).Token)
	if position+4 >= int(tree.Token_Cursors[ast.CURSOR_TOKEN_COUNT]) {
		return false
	}
	source := subject.Sources[SOURCE_SLOT]
	if string(token.Text(source, ast.Token_At(tree, ast.Token_Index(position+1)))) != "err" {
		return false
	}
	if ast.Token_At(tree, ast.Token_Index(position+2)).Kind != token.KIND_NOT_EQUAL {
		return false
	}
	if string(token.Text(source, ast.Token_At(tree, ast.Token_Index(position+3)))) != "nil" {
		return false
	}
	if ast.Token_At(tree, ast.Token_Index(position+4)).Kind != token.KIND_BRACE_LEFT {
		return false
	}
	return holds_two_parts(subject, tree)
}

// Reports whether the form the walk stands on holds two parts alone, counting neither the empty
// lines nor the notes standing between them. A check of two parts states a condition and a block
// and no branch behind them.
func holds_two_parts(subject *Printer, tree *ast.Parse_State) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "holds_two_parts.yes") }()
	Printer_Invariants(subject, "holds_two_parts.subject")
	ast.Parse_State_Invariants(tree, "holds_two_parts.tree")
	held := subject.Nodes[subject.Counts[COUNT_DEPTH]]
	depth := subject.Counts[COUNT_DEPTH]
	parts := 0
	open := descend(subject, tree)
	for bool(open) {
		if !bool(trivia_kind(subject, tree)) {
			parts = parts + 1
		}
		open = advance(subject, tree)
	}
	subject.Counts[COUNT_DEPTH] = depth
	subject.Nodes[depth] = held
	return Boolean(parts == 2)
}

// Reports whether the empty line the walk stands on stands ahead of the simple error check the
// statement that holds it opens.
func stands_before_check(subject *Printer, tree *ast.Parse_State) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "stands_before_check.yes") }()
	Printer_Invariants(subject, "stands_before_check.subject")
	ast.Parse_State_Invariants(tree, "stands_before_check.tree")
	if node_kind(subject, tree) != ast.NODE_BLANK {
		return false
	}
	if subject.Positions[POSITION_CHECK] == POSITION_MINIMUM {
		return false
	}
	// The empty line names the token ahead of it. A semicolon a line feed stands for spans no
	// byte, and a note between the two forms stands with the check rather than with the value
	// the check reads, thus the read steps over both to the word the check opens with.
	node := ast.Node_At(tree, subject.Nodes[subject.Counts[COUNT_DEPTH]])
	position := int(node.Token)
	count := int(tree.Token_Cursors[ast.CURSOR_TOKEN_COUNT])
	for position < count {
		one := ast.Token_At(tree, ast.Token_Index(position))
		held := one.Kind == token.KIND_COMMENT
		if one.Size == 0 {
			held = one.Kind == token.KIND_SEMICOLON
		}
		if !held {
			break
		}
		position = position + 1
	}
	return Boolean(Position(position) == subject.Positions[POSITION_CHECK])
}

// Reports whether the node the walk stands on is an empty line the scanner counted behind the
// brace that closes the block rather than inside it.
func closes_block(subject *Printer, tree *ast.Parse_State) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "closes_block.yes") }()
	Printer_Invariants(subject, "closes_block.subject")
	ast.Parse_State_Invariants(tree, "closes_block.tree")
	if node_kind(subject, tree) != ast.NODE_BLANK {
		return false
	}
	node := ast.Node_At(tree, subject.Nodes[subject.Counts[COUNT_DEPTH]])
	// An empty line the scanner counted at the last brace stands inside the block, and one
	// it counted at a token behind that brace stands behind the block.
	return Boolean(ast.Token_At(tree, node.Token).Kind != token.KIND_BRACE_RIGHT)
}

// Writes one statement of a block, on a line of its own.
func print_statement(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "print_statement.subject")
	ast.Parse_State_Invariants(tree, "print_statement.tree")
	take_check(subject, tree)
	switch node_kind(subject, tree) {
	case ast.NODE_BLANK:
		if bool(stands_against(subject, tree)) {
			return
		}
		if bool(stands_before_check(subject, tree)) {
			return
		}
		close_line(subject, tree)
		return
	case ast.NODE_COMMENT:
		emit_indent(subject)
		emit_token(subject, tree)
		close_line(subject, tree)
		return
	case ast.NODE_DECLARATION_STATEMENT:
		print_declaration_statement(subject, tree)
		return
	case ast.NODE_BLOCK:
		emit_indent(subject)
		print_block(subject, tree)
		close_line(subject, tree)
		return
	}
	// A statement that the author broke across lines indents its continuations, and the tabs
	// it owed stand again at the statement behind it.
	held := subject.Counts[COUNT_INDENT]
	subject.Flags[FLAG_BROKEN] = false
	// A label names the statement behind it rather than stand inside it, thus it stands one
	// tab shallower than the statements around it and closes no line of its own.
	labelled := node_kind(subject, tree) == ast.NODE_LABEL
	if labelled {
		if subject.Counts[COUNT_INDENT] > 0 {
			subject.Counts[COUNT_INDENT] = subject.Counts[COUNT_INDENT] - 1
		}
	}
	// A clause closes its own line, because the comments behind its body take lines of their
	// own rather than the end of the line its brace closes.
	closed := labelled || bool(closes_own_line(subject, tree))
	emit_indent(subject)
	print_statement_form(subject, tree)
	if !closed {
		close_line(subject, tree)
	}
	subject.Counts[COUNT_INDENT] = held
	subject.Flags[FLAG_BROKEN] = false
}

// Writes the form of one statement, which every kind states its own way.
func print_statement_form(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "print_statement_form.subject")
	ast.Parse_State_Invariants(tree, "print_statement_form.tree")
	// A statement opens a form of its own, thus every sign it states spaces the way a
	// statement spaces however deep the form that holds the statement stands.
	subject.Counts[COUNT_NEST] = 1
	switch node_kind(subject, tree) {
	case ast.NODE_ASSIGN, ast.NODE_DEFINE, ast.NODE_OPERATION_ASSIGN, ast.NODE_SEND:
		print_assignment(subject, tree)
	case ast.NODE_INCREMENT, ast.NODE_DECREMENT:
		print_step(subject, tree)
	case ast.NODE_RETURN, ast.NODE_GO, ast.NODE_DEFER, ast.NODE_BREAK,
		ast.NODE_CONTINUE, ast.NODE_GOTO, ast.NODE_FALLTHROUGH:
		print_word_statement(subject, tree)
	case ast.NODE_IF, ast.NODE_FOR, ast.NODE_SWITCH, ast.NODE_TYPE_SWITCH,
		ast.NODE_SELECT:
		print_clause(subject, tree)
	case ast.NODE_CASE, ast.NODE_DEFAULT:
		print_case(subject, tree)
	case ast.NODE_LABEL:
		print_label(subject, tree)
	case ast.NODE_EXPRESSION_STATEMENT:
		print_inner(subject, tree)
	default:
		print_expression(subject, tree)
	}
}

// Writes the expression one statement stands for, which is the one child it holds.
func print_inner(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "print_inner.subject")
	ast.Parse_State_Invariants(tree, "print_inner.tree")
	if !bool(descend_part(subject, tree)) {
		return
	}
	print_expression(subject, tree)
	finish_parts(subject, tree)
	ascend(subject)
}

// Writes one local declaration, which states the same form a file-level one does.
func print_declaration_statement(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "print_declaration_statement.subject")
	ast.Parse_State_Invariants(tree, "print_declaration_statement.tree")
	if !bool(descend_part(subject, tree)) {
		return
	}
	held_short := subject.Flags[FLAG_SHORT]
	subject.Flags[FLAG_SHORT] = states_short(subject, tree)
	defer func() { subject.Flags[FLAG_SHORT] = held_short }()
	print_declaration(subject, tree)
	finish_parts(subject, tree)
	// The declaration closed its own line, thus the notes the walk stepped over behind it
	// take lines of their own rather than the end of the line the statement behind it writes.
	close_debts(subject, tree)
	ascend(subject)
}

// Writes one assignment: the values on its left, the sign the source states, and the values on
// its right.
func print_assignment(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "print_assignment.subject")
	ast.Parse_State_Invariants(tree, "print_assignment.tree")
	// An assignment of a run of values against a run of names states each value as one part of
	// a wider form, thus its signs close up the way the parts of any other form do.
	nest := subject.Counts[COUNT_NEST]
	if bool(assigns_runs(subject, tree)) {
		nest = nest + 1
	}
	held_bound := subject.Positions[POSITION_BOUND]
	take_bound(subject, tree)
	defer func() { subject.Positions[POSITION_BOUND] = held_bound }()
	take_names(subject, tree)
	names := int(subject.Counts[COUNT_NAME])
	if !bool(descend_part(subject, tree)) {
		return
	}
	// The tree holds no node for the sign of an assignment, thus the print reads it from the
	// token that stands ahead of the first value on the right.
	values := 0
	open := Boolean(true)
	// The names on the left indent their own continuations, and the values on the right answer
	// to the statement rather than to the line the names ended on.
	indent := subject.Counts[COUNT_INDENT]
	marked := subject.Flags[FLAG_BROKEN]
	for bool(open) {
		right := opens_right(subject, tree)
		// A name a range clause binds to nothing states nothing, thus the print steps
		// over it and over the sign that stood behind the run it closed.
		if !bool(right) {
			if values >= names {
				open = advance_part(subject, tree)
				continue
			}
		}
		if bool(right) {
			if names > 0 {
				emit_space(subject)
				emit_sign(subject, tree)
			}
			subject.Counts[COUNT_INDENT] = indent
			subject.Flags[FLAG_BROKEN] = marked
			values = 0
		}
		if values > 0 {
			emit_byte(subject, ',')
		}
		// A value the author wrote on a line of its own keeps that line, and it stands
		// one tab deeper than the statement that holds it.
		// The first name on the left opens the statement, thus it takes neither the break
		// the author wrote nor the space a value behind a sign takes.
		behind := Boolean(values > 0)
		if bool(right) {
			behind = Boolean(names > 0)
		}
		open_value(subject, tree, behind, right)
		subject.Counts[COUNT_NEST] = nest
		print_expression(subject, tree)
		values = values + 1
		open = advance_part(subject, tree)
	}
	finish_parts(subject, tree)
	ascend(subject)
}

// Opens the line one value of an assignment stands on: the break the author wrote ahead of it, or
// the space that holds it against the value before it. The value a sign binds stands on the line
// the sign closes, thus a break the author wrote behind the sign itself states nothing at all.
func open_value(subject *Printer, tree *ast.Parse_State, behind Boolean, right Boolean) {
	Printer_Invariants(subject, "open_value.subject")
	ast.Parse_State_Invariants(tree, "open_value.tree")
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
}

// Names the run position of the first token the node the walk stands on spans. A node names the
// token its own form opens with, thus an operation names its sign and never its left value.
func leftmost_position(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "leftmost_position.subject")
	ast.Parse_State_Invariants(tree, "leftmost_position.tree")
	index := subject.Nodes[subject.Counts[COUNT_DEPTH]]
	smallest := Position(ast.Node_At(tree, index).Token)
	for range DEPTH_MAXIMUM {
		child := ast.Node_At(tree, index).First_Child
		if ast.Index(child) == 0 {
			break
		}
		index = ast.Index(child)
		if Position(ast.Node_At(tree, index).Token) < smallest {
			smallest = Position(ast.Node_At(tree, index).Token)
		}
	}
	subject.Positions[POSITION_FOUND] = smallest
}

// Reports whether the node the walk stands on opens the right side of an assignment, which the
// sign that stands ahead of it states.
func opens_right(subject *Printer, tree *ast.Parse_State) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "opens_right.yes") }()
	Printer_Invariants(subject, "opens_right.subject")
	ast.Parse_State_Invariants(tree, "opens_right.tree")
	leftmost_position(subject, tree)
	position := subject.Positions[POSITION_FOUND]
	if position == 0 {
		return false
	}
	subject.Signs[SIGN_HELD] = ast.Token_At(tree, ast.Token_Index(position-1)).Kind
	return signs(subject)
}

// Reports whether one class names a sign an assignment or a send states.
func signs(subject *Printer) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "signs.yes") }()
	Printer_Invariants(subject, "signs.subject")
	switch subject.Signs[SIGN_HELD] {
	case token.KIND_ASSIGN, token.KIND_DEFINE, token.KIND_ARROW, token.KIND_PLUS_ASSIGN,
		token.KIND_MINUS_ASSIGN, token.KIND_STAR_ASSIGN, token.KIND_SLASH_ASSIGN,
		token.KIND_PERCENT_ASSIGN, token.KIND_AND_ASSIGN, token.KIND_OR_ASSIGN,
		token.KIND_EXCLUSIVE_OR_ASSIGN, token.KIND_SHIFT_LEFT_ASSIGN,
		token.KIND_SHIFT_RIGHT_ASSIGN, token.KIND_AND_NOT_ASSIGN:
		return true
	}
	return false
}

// Writes the sign that stands ahead of the node the walk stands on.
func emit_sign(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "emit_sign.subject")
	ast.Parse_State_Invariants(tree, "emit_sign.tree")
	leftmost_position(subject, tree)
	position := subject.Positions[POSITION_FOUND]
	text := token.Text(
		subject.Sources[SOURCE_SLOT], ast.Token_At(tree, ast.Token_Index(position-1)))
	for index := range len(text) {
		emit_byte(subject, Symbol(text[index]))
	}
}

// Position is one run position of the token run, which is how a print names a token the tree
// holds without holding the token itself.
//
// POSITION_MINIMUM is the first token of the run and POSITION_MAXIMUM is the last one it admits.
type Position int32

// Position_Invariants states the complete token run.
func Position_Invariants(value Position, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), POSITION_MINIMUM, POSITION_MAXIMUM).
		Ensure()
}

// Writes one increment or decrement: the value and the two signs behind it. The tree holds one
// node for the whole step, thus its class states which pair of signs the print owes.
func print_step(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "print_step.subject")
	ast.Parse_State_Invariants(tree, "print_step.tree")
	value := byte('+')
	if node_kind(subject, tree) == ast.NODE_DECREMENT {
		value = '-'
	}
	if bool(descend_part(subject, tree)) {
		print_expression(subject, tree)
		// The empty line behind a step hangs inside the step, thus the walk steps over
		// it rather than closing the statement without it.
		finish_parts(subject, tree)
		ascend(subject)
	}
	emit_byte(subject, Symbol(value))
	emit_byte(subject, Symbol(value))
}

// Writes one statement that opens with a word: the word the source states and the values behind it.
func print_word_statement(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "print_word_statement.subject")
	ast.Parse_State_Invariants(tree, "print_word_statement.tree")
	emit_token(subject, tree)
	open := descend_part(subject, tree)
	entered := open
	// A return that names no value hands back the values its signature names, thus the print
	// writes those names where the author wrote none: a reader sees what the function hands
	// back at the line it hands it back on. The names the print writes are the ones the
	// signature bound, and no inner name stands for them, because this dialect shadows none.
	if !bool(open) {
		emit_result_names(subject, tree)
	}
	values := 0
	for bool(open) {
		if values > 0 {
			emit_byte(subject, ',')
		}
		// A value the author wrote on a line of its own keeps that line, one tab deeper
		// than the statement that states it.
		here := Boolean(false)
		if values > 0 {
			here = breaks_before(subject, tree)
		}
		open_continuation(subject, tree, here)
		if !bool(here) {
			emit_space(subject)
		}
		print_expression(subject, tree)
		values = values + 1
		open = advance_part(subject, tree)
	}
	if bool(entered) {
		finish_parts(subject, tree)
		ascend(subject)
	}
}

// Writes one clause statement: the word it opens with, the header it states, and the block behind
// it.
func print_clause(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "print_clause.subject")
	ast.Parse_State_Invariants(tree, "print_clause.tree")
	// The tree holds no node for the word an else branch opens with, thus the print writes it
	// wherever a second branch stands behind the first.
	kind := node_kind(subject, tree)
	subject.Kinds[KIND_CLAUSE] = kind
	// A body of one case alone closes up against the braces that hold it, the way a body of
	// one statement does; a body of more cases keeps the lines the author wrote.
	defer restore_tight(subject, hold_tight(subject, holds_one_case(subject, tree)))
	// The tree drops the semicolons a clause states between its headers, and a loop that
	// leaves a header out states a semicolon the tree names nothing at all for, thus the print
	// reads them from the token run, opening at the word the clause states.
	cursor := Position(ast.Node_At(tree, subject.Nodes[subject.Counts[COUNT_DEPTH]]).Token)
	emit_token(subject, tree)
	open := descend_part(subject, tree)
	entered := open
	bodies := 0
	cases := 0
	// A header the author broke across lines indents its continuations, and the body it opens
	// stands at the tabs the clause itself opened at.
	indent := subject.Counts[COUNT_INDENT]
	headers := 0
	closed := false
	for bool(open) {
		// A clause inside this one names itself in the slot while it prints, thus the
		// slot states this clause again before the branch of this clause is read.
		subject.Kinds[KIND_CLAUSE] = kind
		if bool(opens_branch(subject, tree, Boolean(bodies > 0))) {
			emit_space(subject)
			emit_word(subject, WORD_ELSE)
		}
		// A branch that states a further clause closes its own line, thus the clause
		// around it writes no line of its own behind it.
		if bool(opens_else_clause(subject, tree, Boolean(bodies > 0))) {
			emit_space(subject)
			print_statement_form(subject, tree)
			closed = true
			open = advance_part(subject, tree)
			continue
		}
		if bool(opens_case(subject, tree)) {
			subject.Positions[POSITION_FROM] = cursor
			subject.Counts[COUNT_COLUMN_INDENT] = indent
			print_switch_case(subject, tree, Boolean(cases == 0))
			cases = cases + 1
			open = advance_part(subject, tree)
			continue
		}
		if node_kind(subject, tree) == ast.NODE_BLOCK {
			subject.Positions[POSITION_FROM] = cursor
			subject.Counts[COUNT_COLUMN_INDENT] = indent
			open_body(subject, tree)
			print_block(subject, tree)
			bodies = bodies + 1
			open = advance_part(subject, tree)
			continue
		}
		subject.Positions[POSITION_FROM] = cursor
		print_header(subject, tree, Boolean(headers == 0))
		cursor = subject.Positions[POSITION_FROM]
		headers = headers + 1
		open = advance_part(subject, tree)
	}
	close_clause(subject, tree, Boolean(cases > 0), Boolean(closed))
	if bool(entered) {
		drain_trivia(subject, tree, advance(subject, tree))
		ascend(subject)
	}
}

// Closes one clause: the brace its cases stand between, the line it wrote, and the comments and
// the empty line a branch that closed its own line left standing.
func close_clause(subject *Printer, tree *ast.Parse_State, cased Boolean, closed Boolean) {
	Printer_Invariants(subject, "close_clause.subject")
	ast.Parse_State_Invariants(tree, "close_clause.tree")
	Boolean_Invariants(cased, "close_clause.cased")
	Boolean_Invariants(closed, "close_clause.closed")
	if bool(cased) {
		emit_indent(subject)
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
func close_debts(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "close_debts.subject")
	ast.Parse_State_Invariants(tree, "close_debts.tree")
	if subject.Counts[COUNT_COMMENT] > 0 {
		flush_comments(subject, tree)
		emit_line(subject)
	}
	if bool(subject.Flags[FLAG_BLANK]) {
		emit_line(subject)
		subject.Flags[FLAG_BLANK] = false
		subject.Counts[COUNT_WIDTH] = 0
	}
}

// Reports whether one statement closes the line it stands on of its own. A clause writes a body
// and the comments behind that body, thus the statement that holds it closes no line for it.
func closes_own_line(subject *Printer, tree *ast.Parse_State) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "closes_own_line.yes") }()
	Printer_Invariants(subject, "closes_own_line.subject")
	ast.Parse_State_Invariants(tree, "closes_own_line.tree")
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

// Writes one case of a switch or a select: the word, the values it names, and the statements it
// holds one tab deeper.
func print_case(subject *Printer, tree *ast.Parse_State) (closed Boolean) {
	defer func() { Boolean_Invariants(closed, "print_case.closed") }()
	Printer_Invariants(subject, "print_case.subject")
	ast.Parse_State_Invariants(tree, "print_case.tree")
	body := false
	printed := false
	// The tree names a default case by the colon it stands on and holds no node for the word
	// it opens with, thus the print writes that word of its own.
	if node_kind(subject, tree) == ast.NODE_DEFAULT {
		emit_word(subject, WORD_DEFAULT)
	}
	if node_kind(subject, tree) != ast.NODE_DEFAULT {
		emit_token(subject, tree)
	}
	colon_position(subject, tree)
	colon := subject.Positions[POSITION_FOUND]
	subject.Positions[POSITION_FROM] = colon
	// A case of few values reads on one line however the author wrote it, thus the print
	// closes up a head that fits and keeps the lines of a head that does not.
	flat := fits_case(subject, tree)
	open := descend(subject, tree)
	entered := open
	for bool(open) {
		// The colon closes the values a case stands for, thus every child behind it
		// states a statement of the body, a note the body opens with counted.
		subject.Positions[POSITION_FROM] = colon
		if !body {
			body = bool(opens_body(subject, tree))
		}
		if body {
			print_case_body(subject, tree, Boolean(printed))
			printed = true
			open = advance(subject, tree)
			continue
		}
		if bool(trivia_kind(subject, tree)) {
			print_trivia(subject, tree)
			open = advance(subject, tree)
			continue
		}
		print_case_values(subject, tree, flat)
		open = advance(subject, tree)
	}
	if !printed {
		emit_byte(subject, ':')
	}
	if bool(entered) {
		ascend(subject)
	}
	if printed {
		subject.Counts[COUNT_INDENT] = subject.Counts[COUNT_INDENT] - 1
	}
	return Boolean(printed)
}

// Writes one case of a switch on the line it opens. A case that holds statements closes its own
// line, thus the print owes a line only to a case that holds none.
func print_case_line(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "print_case_line.subject")
	ast.Parse_State_Invariants(tree, "print_case_line.tree")
	held := subject.Counts[COUNT_INDENT]
	subject.Flags[FLAG_BROKEN] = false
	emit_indent(subject)
	closed := print_case(subject, tree)
	if !bool(closed) {
		close_line(subject, tree)
	}
	subject.Counts[COUNT_INDENT] = held
	subject.Flags[FLAG_BROKEN] = false
}

// Writes one statement of a case body and opens the body where it stands first.
func print_case_body(subject *Printer, tree *ast.Parse_State, open Boolean) {
	Printer_Invariants(subject, "print_case_body.subject")
	ast.Parse_State_Invariants(tree, "print_case_body.tree")
	Boolean_Invariants(open, "print_case_body.open")
	if !bool(open) {
		emit_byte(subject, ':')
		emit_line(subject)
		subject.Counts[COUNT_INDENT] = subject.Counts[COUNT_INDENT] + 1
	}
	print_statement(subject, tree)
}

// Reports whether the node the walk stands on opens the body of a case rather than one more value
// the case names. A colon stands between the values and the body, thus the token behind one
// states which side of it a node stands on.
func opens_body(subject *Printer, tree *ast.Parse_State) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "opens_body.yes") }()
	Printer_Invariants(subject, "opens_body.subject")
	ast.Parse_State_Invariants(tree, "opens_body.tree")
	colon := subject.Positions[POSITION_FROM]
	leftmost_position(subject, tree)
	return Boolean(subject.Positions[POSITION_FOUND] > colon)
}

// Names the run position of the colon that closes the values one case states. A note may stand
// between that colon and the first statement of the body, thus the print reads the colon from the
// token run rather than from the token ahead of a statement.
func colon_position(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "colon_position.subject")
	ast.Parse_State_Invariants(tree, "colon_position.tree")
	depth := 0
	opening := ast.Node_At(tree, subject.Nodes[subject.Counts[COUNT_DEPTH]]).Token
	subject.Positions[POSITION_FOUND] = 0
	for offset := int(opening); offset <= ast.TOKEN_INDEX_MAXIMUM; offset++ {
		kind := ast.Token_At(tree, ast.Token_Index(offset)).Kind
		if kind == token.KIND_END_OF_FILE {
			return
		}
		subject.Signs[SIGN_HELD] = kind
		if bool(opens_bracket(subject)) {
			depth = depth + 1
		}
		subject.Signs[SIGN_HELD] = kind
		if bool(closes_bracket(subject)) {
			depth = depth - 1
		}
		if kind != token.KIND_COLON {
			continue
		}
		if depth == 0 {
			subject.Positions[POSITION_FOUND] = Position(offset)
			return
		}
	}
}

// Writes one label and the statement it names.
func print_label(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "print_label.subject")
	ast.Parse_State_Invariants(tree, "print_label.tree")
	emit_token(subject, tree)
	emit_byte(subject, ':')
	if !bool(descend_part(subject, tree)) {
		close_line(subject, tree)
		return
	}
	// The tree names the label twice: once on the node and once as the name it binds, thus
	// the statement the label names stands behind that name.
	stand := advance_part(subject, tree)
	close_line(subject, tree)
	if bool(stand) {
		subject.Counts[COUNT_INDENT] = subject.Counts[COUNT_INDENT] + 1
		print_statement(subject, tree)
		// The statement closed its line, thus every comment behind it takes a line of
		// its own rather than the end of a line already written.
		drain_trivia(subject, tree, advance(subject, tree))
	}
	if !bool(stand) {
		finish_parts(subject, tree)
	}
	ascend(subject)
}

// Writes one expression, which every kind states its own way.
func print_expression(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "print_expression.subject")
	ast.Parse_State_Invariants(tree, "print_expression.tree")
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
func print_wrapped(subject *Printer, tree *ast.Parse_State, wrap Wrap) {
	Printer_Invariants(subject, "print_wrapped.subject")
	ast.Parse_State_Invariants(tree, "print_wrapped.tree")
	Wrap_Invariants(wrap, "print_wrapped.wrap")
	// A call and an index state parts of an expression, not a body, thus an empty line the
	// author left against their brackets stands where the author put it.
	defer restore_tight(subject, hold_tight(subject, false))
	// A form that runs the line past the widest one states one part to a line, thus the print
	// reads the width the whole form writes before it writes the bracket it opens with.
	over := runs_over(subject, tree)
	// The values inside these brackets stand on lines of their own, thus a run of one
	// operation outside them states nothing about the runs they hold.
	defer restore_split(subject, hold_split(subject, false))
	tail := closes_line(subject, tree)
	indent := subject.Counts[COUNT_INDENT]
	marked := subject.Flags[FLAG_BROKEN]
	held := subject.Counts[COUNT_NEST]
	inner_nest(subject, tree, wrap)
	head_nest(subject, wrap)
	inner := subject.Counts[COUNT_INNER]
	head := subject.Counts[COUNT_HEAD]
	open := descend_part(subject, tree)
	entered := open
	parts := 0
	broken := Boolean(false)
	opened := Boolean(false)
	// A call the author broke behind its parenthesis closes on a line of its own, thus the
	// print reads that break at the first argument and holds it past the arguments behind it.
	heads := Boolean(false)
	for bool(open) {
		// The spread of a call stands as a node of its own behind the value it spreads,
		// thus it takes no comma and no space of its own.
		if node_kind(subject, tree) == ast.NODE_ELLIPSIS {
			emit_spread(subject)
			open = advance_part(subject, tree)
			continue
		}
		if parts == 1 {
			emit_open(subject, wrap)
			// The head of the form may have indented what stands behind it, thus
			// the parts answer to the indent the bracket opened at. A bracket the
			// author left on a line of its own closes a broken form however the
			// parts before it stand.
			indent = subject.Counts[COUNT_INDENT]
			marked = subject.Flags[FLAG_BROKEN]
			opened = breaks_before(subject, tree) || over
			heads = opened && Boolean(wrap == WRAP_CALL)
			print_first_break(subject, tree, opened)
			broken = opened || tail
		}
		if parts > 1 {
			subject.Counts[COUNT_COLUMN_INDENT] = indent
			opened, broken = open_argument(subject, tree, opened, broken, marked, over)
		}
		subject.Counts[COUNT_NEST] = inner
		if parts == 0 {
			subject.Counts[COUNT_NEST] = head
		}
		print_part(subject, tree, opened, Boolean(parts > 0))
		subject.Counts[COUNT_NEST] = held
		parts = parts + 1
		open = advance_part(subject, tree)
	}
	if bool(broken) {
		subject.Counts[COUNT_COLUMN_INDENT] = indent
		close_broken(subject, tree, tail || heads)
	}
	close_wrapped(subject, tree, wrap, entered, Boolean(parts < 2))
}

// Closes one wrapped form: the parts the walk still stands ahead of, the bracket a form of one
// part alone still owes, and the bracket every form closes with.
func close_wrapped(
	subject *Printer, tree *ast.Parse_State, wrap Wrap, entered Boolean, alone Boolean,
) {
	Printer_Invariants(subject, "close_wrapped.subject")
	ast.Parse_State_Invariants(tree, "close_wrapped.tree")
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
	emit_close(subject, wrap)
}

// Opens the line the first part of a broken form stands on and indents its parts one level.
func print_first_break(subject *Printer, tree *ast.Parse_State, broken Boolean) {
	Printer_Invariants(subject, "print_first_break.subject")
	ast.Parse_State_Invariants(tree, "print_first_break.tree")
	Boolean_Invariants(broken, "print_first_break.broken")
	if !bool(broken) {
		return
	}
	subject.Counts[COUNT_INDENT] = subject.Counts[COUNT_INDENT] + 1
	close_line(subject, tree)
	emit_indent(subject)
}

// Writes what stands between two parts of a form: a comma and a space, or a comma and the line
// the author broke.
func print_element_break(subject *Printer, tree *ast.Parse_State, broken Boolean) {
	Printer_Invariants(subject, "print_element_break.subject")
	ast.Parse_State_Invariants(tree, "print_element_break.tree")
	Boolean_Invariants(broken, "print_element_break.broken")
	if !bool(broken) {
		emit_byte(subject, ',')
		emit_space(subject)
		return
	}
	emit_break(subject, tree)
}

// Closes a broken form. A bracket the author left on the line of the final part stays there and
// takes no comma; a bracket the author gave a line of its own keeps it, and the part before it
// takes the comma that a line feed there demands.
func close_broken(subject *Printer, tree *ast.Parse_State, tail Boolean) {
	Printer_Invariants(subject, "close_broken.subject")
	ast.Parse_State_Invariants(tree, "close_broken.tree")
	Boolean_Invariants(tail, "close_broken.tail")
	// A part the author broke across lines of its own indented what stood behind it, thus the
	// bracket that closes the form stands where the form opened and not where a part left it.
	subject.Counts[COUNT_INDENT] = subject.Counts[COUNT_COLUMN_INDENT]
	if !bool(tail) {
		return
	}
	emit_byte(subject, ',')
	// An empty line the author left behind the final part belongs to the statement that
	// holds the form and not to the line the bracket closes, thus the print still owes it.
	flush_comments(subject, tree)
	emit_line(subject)
	emit_indent(subject)
}

// Writes the comments the line the print stands on carries behind it. A comment the author wrote
// on a line of its own keeps that line, thus a note about the statement behind it never lands at
// the end of the statement ahead of it.
func flush_comments(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "flush_comments.subject")
	ast.Parse_State_Invariants(tree, "flush_comments.tree")
	for slot := range int(subject.Counts[COUNT_COMMENT]) {
		subject.Positions[POSITION_FOUND] = subject.Comments[slot]
		if bool(stands_alone(subject, tree)) {
			if !bool(subject.Flags[FLAG_LINE]) {
				emit_line(subject)
			}
		}
		if bool(opens_run(subject, tree)) {
			emit_line(subject)
		}
		emit_comment(subject, tree)
	}
	subject.Counts[COUNT_COMMENT] = 0
}

// Reports whether the comment at one run position opens the line it stands on.
func stands_alone(subject *Printer, tree *ast.Parse_State) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "stands_alone.yes") }()
	Printer_Invariants(subject, "stands_alone.subject")
	ast.Parse_State_Invariants(tree, "stands_alone.tree")
	position := subject.Positions[POSITION_FOUND]
	if position == 0 {
		return false
	}
	behind := ast.Token_At(tree, ast.Token_Index(position-1))
	ahead := ast.Token_At(tree, ast.Token_Index(position))
	subject.Offsets[OFFSET_FROM] = token.Offset(int(behind.Offset) + int(behind.Size))
	subject.Offsets[OFFSET_TO] = ahead.Offset
	return feeds(subject)
}

// Writes the bracket that opens one wrapped form.
func emit_open(subject *Printer, wrap Wrap) {
	Printer_Invariants(subject, "emit_open.subject")
	Wrap_Invariants(wrap, "emit_open.wrap")
	if wrap == WRAP_INDEX {
		emit_byte(subject, '[')
		return
	}
	emit_byte(subject, '(')
}

// Writes the bracket that closes one wrapped form.
func emit_close(subject *Printer, wrap Wrap) {
	Printer_Invariants(subject, "emit_close.subject")
	Wrap_Invariants(wrap, "emit_close.wrap")
	if wrap == WRAP_INDEX {
		emit_byte(subject, ']')
		return
	}
	emit_byte(subject, ')')
}

// Writes one operation over two values. A sign that binds loosely stands between spaces and one
// that binds tightly stands against its values, thus the reader sees which value binds to which
// sign without counting precedence.
func print_binary(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "print_binary.subject")
	ast.Parse_State_Invariants(tree, "print_binary.tree")
	spaced := binary_spaced(subject, tree)
	binary_kind(subject, tree)
	level := precedence(subject)
	// A run of one operation breaks at every sign it states or at none of them, thus the
	// outermost operation of the run reads the width and the operations below it hold that
	// answer. An operation that binds tighter states a run of its own and reads its own width,
	// because a value the run holds reads as one value however the run around it stands.
	split := Boolean(false)
	if bool(subject.Flags[FLAG_SPLIT]) {
		split = Boolean(int(subject.Counts[COUNT_SPLIT]) == int(level))
	}
	if !bool(split) {
		split = runs_over(subject, tree)
	}
	held_level := subject.Counts[COUNT_SPLIT]
	subject.Counts[COUNT_SPLIT] = Count(level)
	defer func() { subject.Counts[COUNT_SPLIT] = held_level }()
	defer restore_split(subject, hold_split(subject, split))
	sign := ast.Node_At(tree, subject.Nodes[subject.Counts[COUNT_DEPTH]]).Token
	held := subject.Counts[COUNT_NEST]
	if !bool(descend_part(subject, tree)) {
		return
	}
	// A left value that states the same sign holds the same nest, thus a run of one sign
	// spaces the same way however long it runs.
	subject.Counts[COUNT_NEST] = held + 1
	if bool(left_holds(subject, tree, level)) {
		subject.Counts[COUNT_NEST] = held
	}
	print_expression(subject, tree)
	subject.Counts[COUNT_NEST] = held
	if spaced {
		emit_space(subject)
	}
	// A value the operation holds states a sign of its own while it prints, thus the slot
	// states this sign again before the print writes it.
	subject.Marks[MARK_SIGN] = sign
	emit_token_at(subject, tree)
	if bool(advance_part(subject, tree)) {
		// An operation the author broke behind its sign keeps that break, and its right
		// value stands one tab deeper than the operation that holds it. The tab closes
		// with the operation, thus an operation inside another one steps deeper again
		// while a run of one sign holds one column.
		indent := subject.Counts[COUNT_INDENT]
		here := breaks_before(subject, tree) || split
		if bool(here) {
			subject.Counts[COUNT_INDENT] = indent + 1
			close_line(subject, tree)
			emit_indent(subject)
		}
		if spaced {
			if !bool(here) {
				emit_space(subject)
			}
		}
		subject.Counts[COUNT_NEST] = held + 1
		print_expression(subject, tree)
		subject.Counts[COUNT_NEST] = held
		subject.Counts[COUNT_INDENT] = indent
	}
	finish_parts(subject, tree)
	ascend(subject)
}

// Writes one token of the run the caller names, which is the token of a node the walk already
// stepped past.
func emit_token_at(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "emit_token_at.subject")
	ast.Parse_State_Invariants(tree, "emit_token_at.tree")
	subject.Marks[MARK_TEXT] = subject.Marks[MARK_SIGN]
	emit_text(subject, tree)
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

// LEVEL_MAXIMUM is the tightest binding a sign of an operation states.
const LEVEL_MAXIMUM = LEVEL_SIGN_MAXIMUM

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
func left_holds(subject *Printer, tree *ast.Parse_State, level Level) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "left_holds.yes") }()
	Printer_Invariants(subject, "left_holds.subject")
	ast.Parse_State_Invariants(tree, "left_holds.tree")
	Level_Invariants(level, "left_holds.level")
	if node_kind(subject, tree) != ast.NODE_BINARY {
		return false
	}
	binary_kind(subject, tree)
	return Boolean(precedence(subject) == level)
}

// Names how tightly one sign binds its values. The minimum names a token that is no operation
// sign.
func precedence(subject *Printer) (level Level) {
	defer func() { Level_Invariants(level, "precedence.level") }()
	Printer_Invariants(subject, "precedence.subject")
	switch subject.Signs[SIGN_HELD] {
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

// Reads the sign of the operation the walk stands on.
func binary_kind(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "binary_kind.subject")
	ast.Parse_State_Invariants(tree, "binary_kind.tree")
	node := ast.Node_At(tree, subject.Nodes[subject.Counts[COUNT_DEPTH]])
	subject.Signs[SIGN_HELD] = ast.Token_At(tree, node.Token).Kind
}

// Reports whether the sign of the operation the walk stands on takes spaces around it. The
// canonical form spaces every sign of an operation a statement states on its own, and closes up
// the tighter signs of an operation that stands inside another form or mixes two levels.
func binary_spaced(subject *Printer, tree *ast.Parse_State) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "binary_spaced.yes") }()
	Printer_Invariants(subject, "binary_spaced.subject")
	ast.Parse_State_Invariants(tree, "binary_spaced.tree")
	adds, multiplies, clash := binary_levels(subject, tree)
	// A sign that would read as another sign against the value behind it takes spaces
	// however deep it stands, thus the clash sets the cut on its own.
	cut := Cut(CUT_MINIMUM)
	if clash > 0 {
		cut = Cut(clash) + 1
	}
	if clash == 0 {
		cut = plain_cut(subject, adds, multiplies)
	}
	binary_kind(subject, tree)
	return Boolean(Cut(precedence(subject)) < cut)
}

// Names the level at and above which the signs of one operation close up. An operation that mixes
// the two levels the canonical form spaces by states the tighter of them, and one that holds a
// single level spaces every sign a statement states on its own.
func plain_cut(subject *Printer, adds Boolean, multiplies Boolean) (cut Cut) {
	defer func() { Cut_Invariants(cut, "plain_cut.cut") }()
	Printer_Invariants(subject, "plain_cut.subject")
	Boolean_Invariants(adds, "plain_cut.adds")
	Boolean_Invariants(multiplies, "plain_cut.multiplies")
	plain := subject.Counts[COUNT_NEST] == 1
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

// Names the level a sign must space at to stand apart from the sign of the value behind it. Zero
// names an operation no reading of which runs the two signs together.
func clash_level(subject *Printer) (level Clash) {
	defer func() { Clash_Invariants(level, "clash_level.level") }()
	Printer_Invariants(subject, "clash_level.subject")
	operation := subject.Signs[SIGN_BEHIND]
	unary := subject.Signs[SIGN_HELD]
	if operation == token.KIND_SLASH {
		if unary == token.KIND_STAR {
			return CLASH_TIGHT
		}
	}
	if operation == token.KIND_AND {
		if unary == token.KIND_AND {
			return CLASH_TIGHT
		}
		if unary == token.KIND_EXCLUSIVE_OR {
			return CLASH_TIGHT
		}
	}
	if operation == token.KIND_PLUS {
		if unary == token.KIND_PLUS {
			return CLASH_LOOSE
		}
	}
	if operation == token.KIND_MINUS {
		if unary == token.KIND_MINUS {
			return CLASH_LOOSE
		}
	}
	return CLASH_NONE
}

// Reports which of the two levels the canonical form spaces by stand inside the operation the
// walk stands on. A value the source parenthesised states its own form, thus the walk stops at
// it and never reads the levels it holds.
func binary_levels(
	subject *Printer, tree *ast.Parse_State,
) (adds Boolean, multiplies Boolean, clash Clash) {
	defer func() {
		Boolean_Invariants(adds, "binary_levels.adds")
		Boolean_Invariants(multiplies, "binary_levels.multiplies")
		Clash_Invariants(clash, "binary_levels.clash")
	}()
	Printer_Invariants(subject, "binary_levels.subject")
	ast.Parse_State_Invariants(tree, "binary_levels.tree")
	binary_kind(subject, tree)
	operation := subject.Signs[SIGN_HELD]
	level := precedence(subject)
	adds = Boolean(level == LEVEL_ADDITION)
	multiplies = Boolean(level == LEVEL_SIGN_MAXIMUM)
	if !bool(descend_quiet(subject, tree)) {
		return adds, multiplies, clash
	}
	if bool(reads_levels(subject, tree, level, false)) {
		left, right, held := binary_levels(subject, tree)
		adds = adds || left
		multiplies = multiplies || right
		clash = max(clash, held)
	}
	if bool(advance_quiet(subject, tree)) {
		if bool(reads_levels(subject, tree, level, true)) {
			left, right, held := binary_levels(subject, tree)
			adds = adds || left
			multiplies = multiplies || right
			clash = max(clash, held)
		}
		if node_kind(subject, tree) == ast.NODE_UNARY {
			binary_kind(subject, tree)
			subject.Signs[SIGN_BEHIND] = operation
			clash = max(clash, clash_level(subject))
		}
	}
	ascend(subject)
	return adds, multiplies, clash
}

// Reports whether the value the walk stands on states an operation the form above it spaces as
// one. A value the printer will parenthesise stands as its own form, thus the levels it holds
// say nothing about the form above it.
func reads_levels(
	subject *Printer, tree *ast.Parse_State, level Level, loose Boolean,
) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "reads_levels.yes") }()
	Printer_Invariants(subject, "reads_levels.subject")
	ast.Parse_State_Invariants(tree, "reads_levels.tree")
	Level_Invariants(level, "reads_levels.level")
	Boolean_Invariants(loose, "reads_levels.loose")
	if node_kind(subject, tree) != ast.NODE_BINARY {
		return false
	}
	// The right value of an operation takes parentheses at the level of that operation,
	// thus it reads the levels of the operation above it only when it binds tighter.
	binary_kind(subject, tree)
	held := precedence(subject)
	if bool(loose) {
		return Boolean(held >= level)
	}
	return Boolean(held > level)
}

// Moves the walk to the first part of the node it stands on, over trivia and writing nothing,
// thus a measure the print takes leaves the form it wrote untouched.
func descend_quiet(subject *Printer, tree *ast.Parse_State) (ok Boolean) {
	defer func() { Boolean_Invariants(ok, "descend_quiet.ok") }()
	Printer_Invariants(subject, "descend_quiet.subject")
	ast.Parse_State_Invariants(tree, "descend_quiet.tree")
	if !bool(descend(subject, tree)) {
		return false
	}
	if !bool(trivia_kind(subject, tree)) {
		return true
	}
	if bool(advance_quiet(subject, tree)) {
		return true
	}
	// A form of trivia alone holds no part, and a walk that reports no part must stand where
	// it stood, thus the step down closes here rather than in a caller that never took it.
	ascend(subject)
	return false
}

// Moves the walk to the next part of the form it stands in, over trivia and writing nothing.
func advance_quiet(subject *Printer, tree *ast.Parse_State) (ok Boolean) {
	defer func() { Boolean_Invariants(ok, "advance_quiet.ok") }()
	Printer_Invariants(subject, "advance_quiet.subject")
	ast.Parse_State_Invariants(tree, "advance_quiet.tree")
	for bool(advance(subject, tree)) {
		if !bool(trivia_kind(subject, tree)) {
			return true
		}
	}
	return false
}

// Writes one operation over a single value, which stands against the value it reads.
func print_unary(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "print_unary.subject")
	ast.Parse_State_Invariants(tree, "print_unary.tree")
	// A literal that states the address of the type the run it stands in names repeats that
	// type twice over, thus the canonical form writes neither the sign nor the type.
	held := subject.Types[TYPE_ELEMENT]
	defer func() { subject.Types[TYPE_ELEMENT] = held }()
	if bool(elides_address(subject, tree)) {
		subject.Types[TYPE_ELEMENT] = ast.Index(ast.Node_At(tree, held).First_Child)
		if bool(descend_part(subject, tree)) {
			print_expression(subject, tree)
			finish_parts(subject, tree)
			ascend(subject)
		}
		return
	}
	emit_token(subject, tree)
	if !bool(descend_part(subject, tree)) {
		return
	}
	print_expression(subject, tree)
	finish_parts(subject, tree)
	ascend(subject)
}

// Writes one slice suffix: the value it reads and the bounds it states between colons.
func print_slice_expression(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "print_slice_expression.subject")
	ast.Parse_State_Invariants(tree, "print_slice_expression.tree")
	// The tree holds no node for a bound the source left out, thus the print reads the colons
	// from the token run and writes as many as the author wrote.
	take_bounds(subject, tree)
	written := int(subject.Counts[COUNT_BOUND])
	closer_position(subject, tree)
	closer := subject.Positions[POSITION_FOUND]
	opener := Position(ast.Node_At(tree, subject.Nodes[subject.Counts[COUNT_DEPTH]]).Token)
	blanks := slice_blanks(subject, tree)
	held := subject.Counts[COUNT_NEST]
	// The count of brackets opens behind the bracket the slice itself stands on, thus a colon
	// the slice holds counts at the depth the count opens at.
	cursor := opener + 1
	open := descend_part(subject, tree)
	entered := open
	parts := 0
	for bool(open) {
		if parts > 0 {
			leftmost_position(subject, tree)
			start := subject.Positions[POSITION_FOUND]
			subject.Positions[POSITION_FROM] = cursor
			subject.Positions[POSITION_TO] = start
			emit_colons(subject, tree, blanks)
			cursor = start
		}
		if parts >= written {
			open = advance_part(subject, tree)
			continue
		}
		subject.Counts[COUNT_NEST] = held + 1
		if parts == 0 {
			subject.Counts[COUNT_NEST] = 1
		}
		print_expression(subject, tree)
		subject.Counts[COUNT_NEST] = held
		if parts == 0 {
			emit_byte(subject, '[')
		}
		parts = parts + 1
		open = advance_part(subject, tree)
	}
	if bool(entered) {
		finish_parts(subject, tree)
		ascend(subject)
	}
	if parts == 0 {
		emit_byte(subject, '[')
	}
	subject.Positions[POSITION_FROM] = cursor
	subject.Positions[POSITION_TO] = closer
	emit_colons(subject, tree, blanks)
	emit_byte(subject, ']')
}

// Reads how many bounds of the slice the walk stands on the print writes. A slice that closes at
// the length of the value it reads closes where the brackets already close it, thus that bound
// states nothing and the canonical form drops it.
func take_bounds(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "take_bounds.subject")
	ast.Parse_State_Invariants(tree, "take_bounds.tree")
	elide := elides_bound(subject, tree)
	held := subject.Nodes[subject.Counts[COUNT_DEPTH]]
	depth := subject.Counts[COUNT_DEPTH]
	written := 0
	open := descend(subject, tree)
	for bool(open) {
		if !bool(trivia_kind(subject, tree)) {
			written = written + 1
		}
		open = advance(subject, tree)
	}
	subject.Counts[COUNT_DEPTH] = depth
	subject.Nodes[depth] = held
	if bool(elide) {
		written = written - 1
	}
	subject.Counts[COUNT_BOUND] = Count(written)
}

// Reports whether the bound that closes one slice states the length of the value that slice
// reads. A slice of three bounds names a capacity of its own, thus its closing bound stands
// however it reads, and a slice of one colon closes at its own end with no bound at all.
func elides_bound(subject *Printer, tree *ast.Parse_State) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "elides_bound.yes") }()
	Printer_Invariants(subject, "elides_bound.subject")
	ast.Parse_State_Invariants(tree, "elides_bound.tree")
	closer_position(subject, tree)
	closer := int(subject.Positions[POSITION_FOUND])
	opener := int(ast.Node_At(tree, subject.Nodes[subject.Counts[COUNT_DEPTH]]).Token)
	if closer < opener+4 {
		return false
	}
	if !bool(holds_one_colon(subject, tree)) {
		return false
	}
	// The length reads one value inside brackets of its own, thus the call closes at the
	// bracket ahead of the one that closes the slice.
	if ast.Token_At(tree, ast.Token_Index(closer-1)).Kind != token.KIND_PARENTHESIS_RIGHT {
		return false
	}
	depth := 0
	place := closer - 1
	for place > opener {
		kind := ast.Token_At(tree, ast.Token_Index(place)).Kind
		if kind == token.KIND_PARENTHESIS_RIGHT {
			depth = depth + 1
		}
		if kind == token.KIND_PARENTHESIS_LEFT {
			depth = depth - 1
			if depth == 0 {
				break
			}
		}
		place = place - 1
	}
	if place <= opener+1 {
		return false
	}
	source := subject.Sources[SOURCE_SLOT]
	if string(token.Text(source, ast.Token_At(tree, ast.Token_Index(place-1)))) != "len" {
		return false
	}
	if ast.Token_At(tree, ast.Token_Index(place-2)).Kind != token.KIND_COLON {
		return false
	}
	// A value of one name reads the same twice however the package binds it, and a value of
	// any wider form may read otherwise, thus only a name closes a slice at its own length.
	leftmost_position(subject, tree)
	if int(subject.Positions[POSITION_FOUND])+1 != opener {
		return false
	}
	subject.Positions[POSITION_FROM] = Position(place + 1)
	subject.Positions[POSITION_TO] = Position(closer - 1)
	return reads_same(subject, tree)
}

// Reports whether the value the length of one slice reads is the value the slice itself reads,
// token for token. Two forms of one spelling state one value, thus a slice that closes at the
// length of its own value closes at its end.
func reads_same(subject *Printer, tree *ast.Parse_State) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "reads_same.yes") }()
	Printer_Invariants(subject, "reads_same.subject")
	ast.Parse_State_Invariants(tree, "reads_same.tree")
	left := int(subject.Positions[POSITION_FOUND])
	opener := int(ast.Node_At(tree, subject.Nodes[subject.Counts[COUNT_DEPTH]]).Token)
	right := int(subject.Positions[POSITION_FROM])
	closer := int(subject.Positions[POSITION_TO])
	source := subject.Sources[SOURCE_SLOT]
	for left < opener {
		if right >= closer {
			return false
		}
		one := ast.Token_At(tree, ast.Token_Index(left))
		two := ast.Token_At(tree, ast.Token_Index(right))
		if one.Kind != two.Kind {
			return false
		}
		if string(token.Text(source, one)) != string(token.Text(source, two)) {
			return false
		}
		left = left + 1
		right = right + 1
	}
	return Boolean(right == closer)
}

// Reports whether the slice the walk stands on holds one colon alone, counting no colon a bracket
// of its own holds.
func holds_one_colon(subject *Printer, tree *ast.Parse_State) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "holds_one_colon.yes") }()
	Printer_Invariants(subject, "holds_one_colon.subject")
	ast.Parse_State_Invariants(tree, "holds_one_colon.tree")
	closer := int(subject.Positions[POSITION_FOUND])
	opener := int(ast.Node_At(tree, subject.Nodes[subject.Counts[COUNT_DEPTH]]).Token)
	depth := 0
	colons := 0
	for place := opener + 1; place < closer; place = place + 1 {
		kind := ast.Token_At(tree, ast.Token_Index(place)).Kind
		subject.Signs[SIGN_HELD] = kind
		if bool(opens_bracket(subject)) {
			depth = depth + 1
		}
		subject.Signs[SIGN_HELD] = kind
		if bool(closes_bracket(subject)) {
			depth = depth - 1
		}
		if depth != 0 {
			continue
		}
		if kind == token.KIND_COLON {
			colons = colons + 1
		}
	}
	return Boolean(colons == 1)
}

// Writes the colons the source states between two run positions. A colon inside a bracket of its
// own belongs to the form that bracket opens, thus only a colon the slice itself holds counts.
func emit_colons(subject *Printer, tree *ast.Parse_State, blanks Boolean) {
	Printer_Invariants(subject, "emit_colons.subject")
	ast.Parse_State_Invariants(tree, "emit_colons.tree")
	Boolean_Invariants(blanks, "emit_colons.blanks")
	from := subject.Positions[POSITION_FROM]
	to := subject.Positions[POSITION_TO]
	depth := 0
	for position := int(from); position < int(to); position++ {
		kind := ast.Token_At(tree, ast.Token_Index(position)).Kind
		subject.Signs[SIGN_HELD] = kind
		if bool(opens_bracket(subject)) {
			depth = depth + 1
		}
		subject.Signs[SIGN_HELD] = kind
		if bool(closes_bracket(subject)) {
			depth = depth - 1
		}
		if kind != token.KIND_COLON {
			continue
		}
		if depth != 0 {
			continue
		}
		subject.Positions[POSITION_FOUND] = Position(position)
		emit_colon(subject, tree, blanks)
	}
}

// Writes one colon of a slice and the spaces the canonical form states around it. A space stands
// only against a bound the source states, thus an absent bound takes none.
func emit_colon(subject *Printer, tree *ast.Parse_State, blanks Boolean) {
	Printer_Invariants(subject, "emit_colon.subject")
	ast.Parse_State_Invariants(tree, "emit_colon.tree")
	Boolean_Invariants(blanks, "emit_colon.blanks")
	position := subject.Positions[POSITION_FOUND]
	behind := ast.Token_At(tree, ast.Token_Index(position-1)).Kind
	ahead := ast.Token_At(tree, ast.Token_Index(position+1)).Kind
	subject.Signs[SIGN_HELD] = behind
	subject.Signs[SIGN_BEHIND] = token.KIND_BRACKET_LEFT
	if bool(bounds(subject, blanks)) {
		emit_space(subject)
	}
	emit_byte(subject, ':')
	subject.Signs[SIGN_HELD] = ahead
	subject.Signs[SIGN_BEHIND] = token.KIND_BRACKET_RIGHT
	if bool(bounds(subject, blanks)) {
		emit_space(subject)
	}
}

// Reports whether a space stands on one side of the colon of a slice. A space stands only against
// a bound the source states, thus a bracket or a second colon on that side takes none.
func bounds(subject *Printer, blanks Boolean) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "bounds.yes") }()
	Printer_Invariants(subject, "bounds.subject")
	Boolean_Invariants(blanks, "bounds.blanks")
	kind := subject.Signs[SIGN_HELD]
	if !bool(blanks) {
		return false
	}
	if kind == subject.Signs[SIGN_BEHIND] {
		return false
	}
	return Boolean(kind != token.KIND_COLON)
}

// Reports whether the colons of one slice stand between spaces. Two bounds one of which states
// an operation read harder closed up than spaced, thus the canonical form spaces them.
func slice_blanks(subject *Printer, tree *ast.Parse_State) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "slice_blanks.yes") }()
	Printer_Invariants(subject, "slice_blanks.subject")
	ast.Parse_State_Invariants(tree, "slice_blanks.tree")
	if subject.Counts[COUNT_NEST] > 1 {
		return false
	}
	held := subject.Nodes[subject.Counts[COUNT_DEPTH]]
	depth := subject.Counts[COUNT_DEPTH]
	held_bounds := 0
	operations := 0
	open := descend_quiet(subject, tree)
	for bool(open) {
		held_bounds = held_bounds + 1
		if node_kind(subject, tree) == ast.NODE_BINARY {
			operations = operations + 1
		}
		open = advance_quiet(subject, tree)
	}
	subject.Counts[COUNT_DEPTH] = depth
	subject.Nodes[depth] = held
	if held_bounds <= 2 {
		return false
	}
	return Boolean(operations > 0)
}

// Writes one selector: the value it reads, the period, and the name behind it.
func print_selector(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "print_selector.subject")
	ast.Parse_State_Invariants(tree, "print_selector.tree")
	if !bool(descend_part(subject, tree)) {
		return
	}
	// A form that read an import by the name the author wrote for it names the package the
	// path states, because the import states that name no longer.
	if bool(names_alias(subject, tree)) {
		emit_package_name(subject)
	}
	if !bool(names_alias(subject, tree)) {
		print_expression(subject, tree)
	}
	emit_byte(subject, '.')
	if bool(advance_part(subject, tree)) {
		open_continuation(subject, tree, breaks_before(subject, tree))
		emit_token(subject, tree)
	}
	finish_parts(subject, tree)
	ascend(subject)
}

// Opens the line a continuation of the statement stands on. One statement takes one tab however
// many lines the author broke it into, which is what the canonical form states for a chain.
func open_continuation(subject *Printer, tree *ast.Parse_State, broken Boolean) {
	Printer_Invariants(subject, "open_continuation.subject")
	ast.Parse_State_Invariants(tree, "open_continuation.tree")
	Boolean_Invariants(broken, "open_continuation.broken")
	if !bool(broken) {
		return
	}
	if !bool(subject.Flags[FLAG_BROKEN]) {
		subject.Counts[COUNT_INDENT] = subject.Counts[COUNT_INDENT] + 1
		subject.Flags[FLAG_BROKEN] = true
	}
	close_line(subject, tree)
	emit_indent(subject)
}

// Writes one assertion: the value it reads and the type it names between brackets.
func print_assertion(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "print_assertion.subject")
	ast.Parse_State_Invariants(tree, "print_assertion.tree")
	if !bool(descend_part(subject, tree)) {
		return
	}
	print_expression(subject, tree)
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
}

// Writes one composite literal: the type it names and the elements between its braces.
func print_composite(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "print_composite.subject")
	ast.Parse_State_Invariants(tree, "print_composite.tree")
	// A literal opens a form of its own, thus its parts space their signs the way a
	// statement does however deep the literal stands.
	nest := subject.Counts[COUNT_NEST]
	width := subject.Counts[COUNT_WIDTH]
	subject.Counts[COUNT_WIDTH] = 0
	defer restore_tight(subject, hold_tight(subject, true))
	// A literal that runs the line past the widest one states one element to a line, the way
	// a literal the author broke does.
	over := runs_over(subject, tree)
	defer restore_split(subject, hold_split(subject, false))
	held_apart := subject.Flags[FLAG_APART]
	subject.Flags[FLAG_APART] = over
	defer func() { subject.Flags[FLAG_APART] = held_apart }()
	force, between := take_spread(subject, tree, closes_line(subject, tree))
	force, between = force || over, between || over
	tail := force
	indent := subject.Counts[COUNT_INDENT]
	marked := subject.Flags[FLAG_BROKEN]
	// A literal that stands as an element of another one states no type of its own, thus the
	// print reads whether a type stands ahead of the brace rather than assuming one does.
	brace := Position(ast.Node_At(tree, subject.Nodes[subject.Counts[COUNT_DEPTH]]).Token)
	open := descend_part(subject, tree)
	entered := open
	head := 0
	subject.Positions[POSITION_FROM] = brace
	held_element := subject.Types[TYPE_ELEMENT]
	defer func() { subject.Types[TYPE_ELEMENT] = held_element }()
	open = opens_type(subject, tree, open)
	head = int(subject.Counts[COUNT_HEAD])
	own := subject.Types[TYPE_OWN]
	parts := 0
	broken := Boolean(false)
	opened := Boolean(false)
	after := Boolean(false)
	for bool(open) {
		subject.Types[TYPE_ELEMENT] = own
		take_element_type(subject, tree)
		if parts == head {
			emit_byte(subject, '{')
			indent = subject.Counts[COUNT_INDENT]
			marked = subject.Flags[FLAG_BROKEN]
			opened = open_literal(subject, tree, force)
			broken = opened || tail
		}
		subject.Counts[COUNT_COLUMN_INDENT] = indent
		if parts > head {
			opened, broken = open_element(
				subject, tree, opened, broken, marked, between, after, over,
			)
		}
		after = Boolean(node_kind(subject, tree) == ast.NODE_COMPOSITE)
		subject.Counts[COUNT_NEST] = 1
		spans := spans_form(subject, tree)
		if bool(over) {
			spans = spans || runs_over(subject, tree)
		}
		print_part(subject, tree, opened, true)
		subject.Counts[COUNT_NEST] = nest
		close_key_run(subject, tree, spans)
		parts = parts + 1
		open = advance_part(subject, tree)
	}
	subject.Counts[COUNT_COLUMN_INDENT] = indent
	close_literal(subject, tree, broken, entered, Boolean(parts <= head), tail)
	subject.Counts[COUNT_WIDTH] = width
}

// Closes one literal: the line the parts of a broken form stand apart on, the walk the print
// opened, and the braces. A literal of no parts at all opens its brace here, because the print
// writes that brace at its first part and no part stood.
func close_literal(
	subject *Printer, tree *ast.Parse_State,
	broken Boolean, entered Boolean, empty Boolean, tail Boolean,
) {
	Printer_Invariants(subject, "close_literal.subject")
	ast.Parse_State_Invariants(tree, "close_literal.tree")
	Boolean_Invariants(broken, "close_literal.broken")
	Boolean_Invariants(entered, "close_literal.entered")
	Boolean_Invariants(empty, "close_literal.empty")
	Boolean_Invariants(tail, "close_literal.tail")
	if bool(broken) {
		close_broken(subject, tree, tail)
	}
	if bool(entered) {
		finish_parts(subject, tree)
		ascend(subject)
	}
	if bool(empty) {
		emit_byte(subject, '{')
	}
	emit_byte(subject, '}')
}

// Reports whether the sign that takes the address of one literal states the pointer the run that
// literal stands in already names. The run names the type of every element it holds, thus an
// element that names it again states nothing.
func elides_address(subject *Printer, tree *ast.Parse_State) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "elides_address.yes") }()
	Printer_Invariants(subject, "elides_address.subject")
	ast.Parse_State_Invariants(tree, "elides_address.tree")
	element := subject.Types[TYPE_ELEMENT]
	if element == ast.INDEX_ABSENT {
		return false
	}
	pointer := ast.Node_At(tree, element)
	if pointer.Kind != ast.NODE_POINTER_TYPE {
		return false
	}
	node := ast.Node_At(tree, subject.Nodes[subject.Counts[COUNT_DEPTH]])
	if ast.Token_At(tree, node.Token).Kind != token.KIND_AND {
		return false
	}
	operand := ast.Index(node.First_Child)
	if operand == ast.INDEX_ABSENT {
		return false
	}
	one := ast.Node_At(tree, operand)
	if one.Kind != ast.NODE_COMPOSITE {
		return false
	}
	head := ast.Index(one.First_Child)
	if head == ast.INDEX_ABSENT {
		return false
	}
	// A literal that states no type of its own opens at its brace, thus a head that stands
	// behind that brace states elements and never a type.
	if ast.Node_At(tree, head).Token >= one.Token {
		return false
	}
	subject.Types[TYPE_LEFT] = head
	subject.Types[TYPE_RIGHT] = ast.Index(pointer.First_Child)
	return same_form(subject, tree)
}

// Opens the body of one literal: the break the author left behind the brace, and the widths the
// run of pairs behind it aligns on. It reports whether that break stands.
func open_literal(subject *Printer, tree *ast.Parse_State, force Boolean) (opened Boolean) {
	defer func() { Boolean_Invariants(opened, "open_literal.opened") }()
	Printer_Invariants(subject, "open_literal.subject")
	ast.Parse_State_Invariants(tree, "open_literal.tree")
	Boolean_Invariants(force, "open_literal.force")
	opened = breaks_before(subject, tree) || force
	print_first_break(subject, tree, opened)
	if bool(opened) {
		take_key_width(subject, tree)
	}
	return opened
}

// Opens the line one element of a literal stands on and reports what the run states behind it:
// whether a break now stands ahead of the elements, and whether the literal stands broken.
func open_element(
	subject *Printer, tree *ast.Parse_State, opened Boolean, broken Boolean,
	marked Boolean, between Boolean, after Boolean, over Boolean,
) (stand Boolean, spread Boolean) {
	defer func() {
		Boolean_Invariants(stand, "open_element.stand")
		Boolean_Invariants(spread, "open_element.spread")
	}()
	Printer_Invariants(subject, "open_element.subject")
	ast.Parse_State_Invariants(tree, "open_element.tree")
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
		subject.Counts[COUNT_INDENT] = subject.Counts[COUNT_COLUMN_INDENT] + 1
		subject.Flags[FLAG_BROKEN] = marked
	}
	print_element_break(subject, tree, here)
	open_key_run(subject, tree, here)
	return opened || here, broken || here
}

// Reads how the elements of one literal stand: whether the braces or the elements open lines of
// their own, and whether a break stands between two elements. A literal the author broke states
// one element to a line, thus the print reads the whole run before it writes the first element.
func take_spread(
	subject *Printer, tree *ast.Parse_State, tail Boolean,
) (force Boolean, between Boolean) {
	defer func() {
		Boolean_Invariants(force, "take_spread.force")
		Boolean_Invariants(between, "take_spread.between")
	}()
	Printer_Invariants(subject, "take_spread.subject")
	ast.Parse_State_Invariants(tree, "take_spread.tree")
	Boolean_Invariants(tail, "take_spread.tail")
	if bool(subject.Flags[FLAG_FLAT]) {
		return false, false
	}
	held := subject.Nodes[subject.Counts[COUNT_DEPTH]]
	depth := subject.Counts[COUNT_DEPTH]
	brace := Position(ast.Node_At(tree, held).Token)
	force = tail
	parts := 0
	open := descend_quiet(subject, tree)
	for bool(open) {
		leftmost_position(subject, tree)
		// The type one literal names stands ahead of its brace, thus a part that opens
		// behind the brace is an element and every other one names the type.
		if subject.Positions[POSITION_FOUND] > brace {
			here := breaks_before(subject, tree)
			if parts > 0 {
				between = between || here
			}
			if parts == 0 {
				force = force || here
			}
			parts = parts + 1
		}
		open = advance_quiet(subject, tree)
	}
	subject.Counts[COUNT_DEPTH] = depth
	subject.Nodes[depth] = held
	return force || between, between
}

// Reports whether the type one literal names is the type the literal that holds it already named,
// which the canonical form writes once. It reads the types the parts of this literal wear as
// well, because both answers stand in the type this literal names.
func repeats_type(subject *Printer, tree *ast.Parse_State, typed Boolean) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "repeats_type.yes") }()
	Printer_Invariants(subject, "repeats_type.subject")
	ast.Parse_State_Invariants(tree, "repeats_type.tree")
	Boolean_Invariants(typed, "repeats_type.typed")
	own := ast.Index(ast.INDEX_ABSENT)
	repeats := Boolean(false)
	if bool(typed) {
		here := subject.Nodes[subject.Counts[COUNT_DEPTH]]
		subject.Types[TYPE_LEFT] = here
		subject.Types[TYPE_RIGHT] = subject.Types[TYPE_ELEMENT]
		repeats = same_form(subject, tree)
		own = here
	}
	subject.Types[TYPE_ELEMENT] = own
	take_element_type(subject, tree)
	return repeats
}

// Opens the line one part of a call or an index stands on and reports what the form now states:
// whether a break stands ahead of the parts, and whether the form stands broken. Each part stands
// where the author put it, on the line the part before it holds or on a line of its own, and a
// part that indented lines of its own leaves them behind at the column the form opened at.
func open_argument(
	subject *Printer, tree *ast.Parse_State,
	opened Boolean, broken Boolean, marked Boolean, over Boolean,
) (stand Boolean, spread Boolean) {
	defer func() {
		Boolean_Invariants(stand, "open_argument.stand")
		Boolean_Invariants(spread, "open_argument.spread")
	}()
	Printer_Invariants(subject, "open_argument.subject")
	ast.Parse_State_Invariants(tree, "open_argument.tree")
	Boolean_Invariants(opened, "open_argument.opened")
	Boolean_Invariants(broken, "open_argument.broken")
	Boolean_Invariants(marked, "open_argument.marked")
	Boolean_Invariants(over, "open_argument.over")
	here := breaks_before(subject, tree) || over
	if bool(open_column(here, opened)) {
		subject.Counts[COUNT_INDENT] = subject.Counts[COUNT_COLUMN_INDENT] + 1
		subject.Flags[FLAG_BROKEN] = marked
	}
	print_element_break(subject, tree, here)
	return opened || here, broken || here
}

// Reports whether the two forms the type slots name state one type, token for token. A literal
// whose element repeats the type the literal itself names states that type twice, thus the
// canonical form writes it once.
func same_form(subject *Printer, tree *ast.Parse_State) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "same_form.yes") }()
	Printer_Invariants(subject, "same_form.subject")
	ast.Parse_State_Invariants(tree, "same_form.tree")
	left := subject.Types[TYPE_LEFT]
	right := subject.Types[TYPE_RIGHT]
	if left == ast.INDEX_ABSENT {
		return false
	}
	if right == ast.INDEX_ABSENT {
		return false
	}
	one := ast.Node_At(tree, left)
	two := ast.Node_At(tree, right)
	if one.Kind != two.Kind {
		return false
	}
	source := subject.Sources[SOURCE_SLOT]
	head := token.Text(source, ast.Token_At(tree, one.Token))
	tail := token.Text(source, ast.Token_At(tree, two.Token))
	if string(head) != string(tail) {
		return false
	}
	first := ast.Index(one.First_Child)
	second := ast.Index(two.First_Child)
	for first != ast.INDEX_ABSENT {
		if second == ast.INDEX_ABSENT {
			return false
		}
		subject.Types[TYPE_LEFT] = first
		subject.Types[TYPE_RIGHT] = second
		if !bool(same_form(subject, tree)) {
			return false
		}
		first = ast.Index(ast.Node_At(tree, first).Next)
		second = ast.Index(ast.Node_At(tree, second).Next)
	}
	return Boolean(second == ast.INDEX_ABSENT)
}

// Reads the types the elements and the keys of one literal wear out of the type that literal
// names. A literal of any other type states no type its parts repeat, thus the read names none.
func take_element_type(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "take_element_type.subject")
	ast.Parse_State_Invariants(tree, "take_element_type.tree")
	held := subject.Types[TYPE_ELEMENT]
	subject.Types[TYPE_ELEMENT] = ast.INDEX_ABSENT
	subject.Types[TYPE_KEY] = ast.INDEX_ABSENT
	if held == ast.INDEX_ABSENT {
		return
	}
	node := ast.Node_At(tree, held)
	keyed := node.Kind == ast.NODE_MAP_TYPE
	if node.Kind != ast.NODE_ARRAY_TYPE {
		if node.Kind != ast.NODE_SLICE_TYPE {
			if !keyed {
				return
			}
		}
	}
	first := ast.Index(node.First_Child)
	if first == ast.INDEX_ABSENT {
		return
	}
	// The element of an array stands behind the length it states, and the value of a map
	// stands behind the key it reads, thus the last child names the type of an element and
	// the first one names the type of a key.
	last := first
	for ast.Index(ast.Node_At(tree, last).Next) != ast.INDEX_ABSENT {
		last = ast.Index(ast.Node_At(tree, last).Next)
	}
	subject.Types[TYPE_ELEMENT] = last
	if keyed {
		subject.Types[TYPE_KEY] = first
	}
}

// Writes nothing and reads the type one literal names: whether the print writes that type, which
// stands in the head count, and the type its elements wear, which stands in the own slot. The
// elements of a literal wear the type that literal itself names, and a literal that names none
// states elements of a type no form of the source spells, thus the print reads no type for them
// however the literal that holds it stands. A literal that names the type the literal above it
// already named writes it once, thus the walk steps over the type it wrote twice.
func opens_type(subject *Printer, tree *ast.Parse_State, open Boolean) (stand Boolean) {
	defer func() { Boolean_Invariants(stand, "opens_type.stand") }()
	Printer_Invariants(subject, "opens_type.subject")
	ast.Parse_State_Invariants(tree, "opens_type.tree")
	Boolean_Invariants(open, "opens_type.open")
	subject.Types[TYPE_OWN] = ast.INDEX_ABSENT
	subject.Counts[COUNT_HEAD] = 0
	typed := typed_head(subject, tree, open)
	if bool(typed) {
		subject.Counts[COUNT_HEAD] = 1
		subject.Types[TYPE_OWN] = subject.Nodes[subject.Counts[COUNT_DEPTH]]
	}
	own := subject.Types[TYPE_OWN]
	if bool(repeats_type(subject, tree, typed)) {
		subject.Counts[COUNT_HEAD] = 0
		open = advance_part(subject, tree)
	}
	subject.Types[TYPE_OWN] = own
	return open
}

// Reads the widest key of the run of pairs the walk stands at the opening of. Every value of one
// run stands in the same column, thus the print owes the widest key before it writes the first
// one. A run stops where a pair stops, because a literal that holds anything else aligns nothing.
func take_key_width(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "take_key_width.subject")
	ast.Parse_State_Invariants(tree, "take_key_width.tree")
	held := subject.Nodes[subject.Counts[COUNT_DEPTH]]
	widest := Width(0)
	pairs := 0
	open := Boolean(true)
	for bool(open) {
		if node_kind(subject, tree) != ast.NODE_KEY_VALUE {
			break
		}
		key_width(subject, tree)
		width := subject.Spans[SPAN_FOUND]
		// A pair that runs over lines of its own stands in no column with the pairs
		// around it, thus it opens and closes a run holding itself alone.
		if bool(spans_form(subject, tree)) {
			if pairs == 0 {
				widest = width
			}
			break
		}
		closes := breaks_run(subject, tree)
		if width > widest {
			widest = width
		}
		pairs = pairs + 1
		if bool(closes) {
			break
		}
		open = advance(subject, tree)
		if bool(open) {
			if bool(trails_note(subject, tree)) {
				open = advance(subject, tree)
			}
		}
		// Two pairs that share a line stand in no column, thus the run closes where
		// the author stopped opening a line for each pair. A form the print writes one
		// part to a line opens that line itself, thus no pair of it shares one.
		if bool(open) {
			if !bool(subject.Flags[FLAG_APART]) {
				if !bool(breaks_before(subject, tree)) {
					break
				}
			}
		}
	}
	subject.Nodes[subject.Counts[COUNT_DEPTH]] = held
	subject.Counts[COUNT_WIDTH] = Count(widest)
}

// Reports whether the form the walk stands on writes over more than one line.
func spans_form(subject *Printer, tree *ast.Parse_State) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "spans_form.yes") }()
	Printer_Invariants(subject, "spans_form.subject")
	ast.Parse_State_Invariants(tree, "spans_form.tree")
	leftmost_position(subject, tree)
	opening := ast.Token_At(tree, ast.Token_Index(subject.Positions[POSITION_FOUND]))
	rightmost_position(subject, tree)
	last := ast.Token_At(tree, ast.Token_Index(subject.Positions[POSITION_FOUND]))
	// A raw literal spans the lines it holds, thus the span closes at the end of the final
	// token rather than at the byte that token opens.
	subject.Offsets[OFFSET_FROM] = opening.Offset
	subject.Offsets[OFFSET_TO] = token.Offset(int(last.Offset) + int(last.Size))
	return feeds(subject)
}

// Reads how wide the key of the pair the walk stands on writes.
func key_width(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "key_width.subject")
	ast.Parse_State_Invariants(tree, "key_width.tree")
	subject.Spans[SPAN_FOUND] = 0
	if !bool(descend_quiet(subject, tree)) {
		return
	}
	// A key states a whole expression rather than one word, thus the column it opens the
	// value at answers for the form the print writes and not for the token the key names.
	held := subject.Counts[COUNT_NEST]
	subject.Counts[COUNT_NEST] = 1
	measure_expression(subject, tree)
	subject.Counts[COUNT_NEST] = held
	ascend(subject)
}

// Reports whether the pair the walk stands on closes the run it stands in. A comment or an empty
// line behind a pair closes the run, thus the pairs behind that line align on a width of their
// own.
func breaks_run(subject *Printer, tree *ast.Parse_State) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "breaks_run.yes") }()
	Printer_Invariants(subject, "breaks_run.subject")
	ast.Parse_State_Invariants(tree, "breaks_run.tree")
	held := subject.Nodes[subject.Counts[COUNT_DEPTH]]
	stand := advance(subject, tree)
	// A note the pair carries behind it stands in a column of its own rather than last
	// the run, thus the walk steps over it to read what stands behind the pair.
	if bool(stand) {
		if bool(trails_note(subject, tree)) {
			stand = advance(subject, tree)
		}
	}
	closes := closes_run(subject, tree, stand)
	subject.Nodes[subject.Counts[COUNT_DEPTH]] = held
	return closes
}

// Reports whether the run of pairs closes where the walk stands. Nothing behind the pair, trivia
// behind it, or a pair that shares its line each close the run the pair stood in.
func closes_run(subject *Printer, tree *ast.Parse_State, stand Boolean) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "closes_run.yes") }()
	Printer_Invariants(subject, "closes_run.subject")
	ast.Parse_State_Invariants(tree, "closes_run.tree")
	Boolean_Invariants(stand, "closes_run.stand")
	if !bool(stand) {
		return true
	}
	if bool(trivia_kind(subject, tree)) {
		return true
	}
	// A form the print writes one part to a line opens a line for every pair it holds, thus
	// the run closes at nothing the author wrote on one line.
	if bool(subject.Flags[FLAG_APART]) {
		return false
	}
	return !breaks_before(subject, tree)
}

// Reports whether the node the walk stands on states a note that closes the line before it.
func trails_note(subject *Printer, tree *ast.Parse_State) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "trails_note.yes") }()
	Printer_Invariants(subject, "trails_note.subject")
	ast.Parse_State_Invariants(tree, "trails_note.tree")
	if node_kind(subject, tree) != ast.NODE_COMMENT {
		return false
	}
	subject.Positions[POSITION_FOUND] = Position(ast.Node_At(tree,
		subject.Nodes[subject.Counts[COUNT_DEPTH]]).Token)
	return !stands_alone(subject, tree)
}

// Writes one keyed element: the key, the colon, and the value behind it.
func print_key_value(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "print_key_value.subject")
	ast.Parse_State_Invariants(tree, "print_key_value.tree")
	// The key of a pair wears the type the map states for its keys, and a measure of that key
	// prints it, thus the read stands ahead of every measure the print takes.
	keyed := subject.Types[TYPE_KEY]
	key_width(subject, tree)
	width := subject.Spans[SPAN_FOUND]
	held := subject.Counts[COUNT_WIDTH]
	if !bool(descend_part(subject, tree)) {
		return
	}
	element := subject.Types[TYPE_ELEMENT]
	subject.Types[TYPE_ELEMENT] = keyed
	print_expression(subject, tree)
	subject.Types[TYPE_ELEMENT] = element
	emit_byte(subject, ':')
	if held == 0 {
		emit_space(subject)
	}
	if held > 0 {
		subject.Spans[SPAN_FOUND] = width
		emit_padding(subject)
	}
	if bool(advance_part(subject, tree)) {
		subject.Counts[COUNT_WIDTH] = 0
		print_expression(subject, tree)
		subject.Counts[COUNT_WIDTH] = held
	}
	finish_parts(subject, tree)
	ascend(subject)
}

// Writes one function literal: the word, its signature, and the block behind it.
func print_function_literal(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "print_function_literal.subject")
	ast.Parse_State_Invariants(tree, "print_function_literal.tree")
	emit_word(subject, WORD_FUNCTION)
	open := descend_part(subject, tree)
	entered := open
	print_signature(subject, tree, open)
	if bool(entered) {
		ascend(subject)
	}
}

// Writes one range clause: the word and the value it steps over.
func print_range(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "print_range.subject")
	ast.Parse_State_Invariants(tree, "print_range.tree")
	emit_word(subject, WORD_RANGE)
	if !bool(descend_part(subject, tree)) {
		return
	}
	emit_space(subject)
	print_expression(subject, tree)
	finish_parts(subject, tree)
	ascend(subject)
}

// Writes one type form, which is what an expression that names a type states.
func print_type_form(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "print_type_form.subject")
	ast.Parse_State_Invariants(tree, "print_type_form.tree")
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
func print_prefixed(subject *Printer, tree *ast.Parse_State, prefix Prefix) {
	Printer_Invariants(subject, "print_prefixed.subject")
	ast.Parse_State_Invariants(tree, "print_prefixed.tree")
	Prefix_Invariants(prefix, "print_prefixed.prefix")
	emit_prefix(subject, prefix)
	if !bool(descend_part(subject, tree)) {
		return
	}
	print_expression(subject, tree)
	finish_parts(subject, tree)
	ascend(subject)
}

// Writes the bytes one prefix spells.
func emit_prefix(subject *Printer, prefix Prefix) {
	Printer_Invariants(subject, "emit_prefix.subject")
	Prefix_Invariants(prefix, "emit_prefix.prefix")
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
}

// Writes one array type: the count between brackets and the element behind them.
func print_array(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "print_array.subject")
	ast.Parse_State_Invariants(tree, "print_array.tree")
	emit_byte(subject, '[')
	if !bool(descend_part(subject, tree)) {
		emit_byte(subject, ']')
		return
	}
	print_expression(subject, tree)
	emit_byte(subject, ']')
	if bool(advance_part(subject, tree)) {
		print_expression(subject, tree)
	}
	finish_parts(subject, tree)
	ascend(subject)
}

// Writes one map type: the key between brackets and the value behind them.
func print_map(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "print_map.subject")
	ast.Parse_State_Invariants(tree, "print_map.tree")
	emit_word(subject, WORD_MAP)
	emit_byte(subject, '[')
	if !bool(descend_part(subject, tree)) {
		emit_byte(subject, ']')
		return
	}
	print_expression(subject, tree)
	emit_byte(subject, ']')
	if bool(advance_part(subject, tree)) {
		print_expression(subject, tree)
	}
	finish_parts(subject, tree)
	ascend(subject)
}

// Writes one union of terms, which stand between signs.
func print_union(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "print_union.subject")
	ast.Parse_State_Invariants(tree, "print_union.tree")
	open := descend_part(subject, tree)
	entered := open
	terms := 0
	for bool(open) {
		if terms > 0 {
			emit_space(subject)
			emit_byte(subject, '|')
			emit_space(subject)
		}
		print_expression(subject, tree)
		terms = terms + 1
		open = advance_part(subject, tree)
	}
	if bool(entered) {
		finish_parts(subject, tree)
		ascend(subject)
	}
}

// Writes one term of a constraint, which the tilde opens where it admits a type of its own shape.
func print_term(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "print_term.subject")
	ast.Parse_State_Invariants(tree, "print_term.tree")
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
func approximate(subject *Printer, tree *ast.Parse_State) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "approximate.yes") }()
	Printer_Invariants(subject, "approximate.subject")
	ast.Parse_State_Invariants(tree, "approximate.tree")
	position := ast.Node_At(tree, subject.Nodes[subject.Counts[COUNT_DEPTH]]).Token
	if position == 0 {
		return false
	}
	return ast.Token_At(tree, position-1).Kind == token.KIND_TILDE
}

// Writes one struct type: the word, the fields one tab deeper, and the brace that closes it. A
// struct that states no field stands on one line, which is the form the canonical printer writes.
func print_structure(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "print_structure.subject")
	ast.Parse_State_Invariants(tree, "print_structure.tree")
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
	descend(subject, tree)
	emit_space(subject)
	emit_byte(subject, '{')
	close_line(subject, tree)
	subject.Counts[COUNT_INDENT] = subject.Counts[COUNT_INDENT] + 1
	subject.Counts[COUNT_WIDTH] = 0
	// A run of fields closes up against its braces, thus the struct opens at its first field
	// and closes at its last one.
	held_tight := subject.Flags[FLAG_TIGHT]
	subject.Flags[FLAG_TIGHT] = true
	open := Boolean(true)
	for bool(open) {
		print_field(subject, tree)
		open = advance(subject, tree)
	}
	subject.Flags[FLAG_TIGHT] = held_tight
	ascend(subject)
	subject.Counts[COUNT_INDENT] = subject.Counts[COUNT_INDENT] - 1
	emit_indent(subject)
	emit_byte(subject, '}')
}

// Writes one member of a struct: a field, a comment, or an empty line. A comment and an empty
// line each close the run of fields that stood before them, thus the run behind one aligns on a
// width of its own.
func print_field(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "print_field.subject")
	ast.Parse_State_Invariants(tree, "print_field.tree")
	kind := node_kind(subject, tree)
	if kind == ast.NODE_BLANK {
		subject.Counts[COUNT_WIDTH] = 0
		if bool(stands_against(subject, tree)) {
			return
		}
		// An empty line the scanner counted behind the brace stands behind the struct, thus
		// the print owes it to the line the declaration closes and not to this one.
		if bool(closes_block(subject, tree)) {
			subject.Flags[FLAG_BLANK] = true
			return
		}
		close_line(subject, tree)
		return
	}
	if kind == ast.NODE_COMMENT {
		subject.Counts[COUNT_WIDTH] = 0
		emit_indent(subject)
		emit_token(subject, tree)
		close_line(subject, tree)
		return
	}
	if kind != ast.NODE_FIELD {
		return
	}
	if subject.Counts[COUNT_WIDTH] == 0 {
		take_run_width(subject, tree)
	}
	print_field_parts(subject, tree)
}

// Puts the widest name of the run of fields the walk opens in the width slot. A field that states
// no name of its own closes the run, thus it aligns nothing and nothing aligns on it.
func take_run_width(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "take_run_width.subject")
	ast.Parse_State_Invariants(tree, "take_run_width.tree")
	held := subject.Nodes[subject.Counts[COUNT_DEPTH]]
	widest := Width(0)
	types := Width(0)
	tagged := false
	open := Boolean(true)
	for bool(open) {
		if node_kind(subject, tree) != ast.NODE_FIELD {
			break
		}
		named := field_width(subject, tree)
		width := subject.Spans[SPAN_FOUND]
		if !bool(named) {
			break
		}
		if width > widest {
			widest = width
		}
		// A tag stands in a column of its own behind the types, thus the run answers for
		// the widest type as well as for the widest run of names.
		states := type_width(subject, tree)
		one := subject.Spans[SPAN_FOUND]
		if one > types {
			types = one
		}
		tagged = tagged || bool(states)
		// A field that carries a comment or an empty line behind it closes the run, thus
		// the run behind that line aligns on a width of its own.
		if bool(field_breaks(subject, tree)) {
			break
		}
		open = advance(subject, tree)
	}
	subject.Nodes[subject.Counts[COUNT_DEPTH]] = held
	subject.Counts[COUNT_WIDTH] = Count(widest)
	subject.Counts[COUNT_TYPE] = 0
	if tagged {
		subject.Counts[COUNT_TYPE] = Count(types)
	}
}

// Reads how wide the type of one field writes and whether the field carries a tag behind it.
func type_width(subject *Printer, tree *ast.Parse_State) (tagged Boolean) {
	defer func() { Boolean_Invariants(tagged, "type_width.tagged") }()
	Printer_Invariants(subject, "type_width.subject")
	ast.Parse_State_Invariants(tree, "type_width.tree")
	subject.Spans[SPAN_FOUND] = 0
	if !bool(descend_quiet(subject, tree)) {
		return false
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
		return false
	}
	measure_expression(subject, tree)
	held := subject.Spans[SPAN_FOUND]
	tagged = advance_quiet(subject, tree)
	ascend(subject)
	subject.Spans[SPAN_FOUND] = held
	return tagged
}

// Reports whether the field the walk stands on carries a comment or an empty line behind it.
func field_breaks(subject *Printer, tree *ast.Parse_State) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "field_breaks.yes") }()
	Printer_Invariants(subject, "field_breaks.subject")
	ast.Parse_State_Invariants(tree, "field_breaks.tree")
	if !bool(descend(subject, tree)) {
		return false
	}
	broken := Boolean(false)
	open := Boolean(true)
	for bool(open) {
		kind := node_kind(subject, tree)
		if kind == ast.NODE_BLANK {
			broken = true
		}
		if kind == ast.NODE_COMMENT {
			broken = true
		}
		open = advance(subject, tree)
	}
	ascend(subject)
	return broken
}

// Reads the width of the name one field states, and reports whether it states one at all.
func field_width(subject *Printer, tree *ast.Parse_State) (named Boolean) {
	defer func() {
		Boolean_Invariants(named, "field_width.named")
	}()
	Printer_Invariants(subject, "field_width.subject")
	ast.Parse_State_Invariants(tree, "field_width.tree")
	subject.Spans[SPAN_FOUND] = 0
	if !bool(descend(subject, tree)) {
		return false
	}
	held := Boolean(node_kind(subject, tree) == ast.NODE_FIELD_NAME)
	names_width(subject, tree)
	ascend(subject)
	if !bool(held) {
		subject.Spans[SPAN_FOUND] = 0
		return false
	}
	return true
}

// Reads how wide the names one field binds write, the commas between them counted. Every name of
// one field stands ahead of the same type, thus the column the type opens at answers for the run
// of names and not for one name of it.
func names_width(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "names_width.subject")
	ast.Parse_State_Invariants(tree, "names_width.tree")
	held := subject.Nodes[subject.Counts[COUNT_DEPTH]]
	width := Width(0)
	names := 0
	open := Boolean(true)
	for bool(open) {
		if node_kind(subject, tree) != ast.NODE_FIELD_NAME {
			break
		}
		if names > 0 {
			width = width + 2
		}
		node_width(subject, tree)
		width = width + subject.Spans[SPAN_FOUND]
		names = names + 1
		open = advance(subject, tree)
	}
	subject.Nodes[subject.Counts[COUNT_DEPTH]] = held
	subject.Spans[SPAN_FOUND] = width
}

// Writes the name of one field, the spaces that align the type behind it, and that type.
func print_field_parts(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "print_field_parts.subject")
	ast.Parse_State_Invariants(tree, "print_field_parts.tree")
	emit_indent(subject)
	if !bool(descend_part(subject, tree)) {
		close_line(subject, tree)
		return
	}
	if node_kind(subject, tree) != ast.NODE_FIELD_NAME {
		// A field that binds no name still carries the notes and the empty lines the
		// author wrote behind it, thus the walk steps over them rather than dropping
		// them.
		print_expression(subject, tree)
		stand := advance(subject, tree)
		close_line(subject, tree)
		drain_trivia(subject, tree, stand)
		ascend(subject)
		subject.Counts[COUNT_WIDTH] = 0
		return
	}
	names_width(subject, tree)
	width := subject.Spans[SPAN_FOUND]
	names := 0
	open := Boolean(true)
	for bool(open) {
		if node_kind(subject, tree) != ast.NODE_FIELD_NAME {
			break
		}
		if names > 0 {
			emit_byte(subject, ',')
			emit_space(subject)
		}
		emit_token(subject, tree)
		names = names + 1
		open = advance(subject, tree)
	}
	subject.Spans[SPAN_FOUND] = width
	emit_padding(subject)
	if bool(open) {
		measure_expression(subject, tree)
		span := subject.Spans[SPAN_FOUND]
		print_expression(subject, tree)
		open = advance(subject, tree)
		// A tag stands in the column the widest type of the run opens, thus every tag of
		// one run reads as one column rather than as a ragged edge.
		if bool(open) {
			if subject.Counts[COUNT_TYPE] > 0 {
				subject.Spans[SPAN_FOUND] = span
				subject.Spans[SPAN_COLUMN] = Width(subject.Counts[COUNT_TYPE])
				emit_column(subject)
				emit_token(subject, tree)
				open = advance(subject, tree)
			}
		}
	}
	close_line(subject, tree)
	broken := open
	drain_trivia(subject, tree, open)
	ascend(subject)
	if bool(broken) {
		subject.Counts[COUNT_WIDTH] = 0
	}
}

// Writes the spaces that stand between one field name and the type behind it, which is one space
// past the widest name the run holds.
func emit_padding(subject *Printer) {
	Printer_Invariants(subject, "emit_padding.subject")
	subject.Spans[SPAN_COLUMN] = Width(subject.Counts[COUNT_WIDTH])
	emit_column(subject)
}

// Writes one constraint: the word, the terms one tab deeper, and the brace that closes it.
func print_constraint(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "print_constraint.subject")
	ast.Parse_State_Invariants(tree, "print_constraint.tree")
	emit_word(subject, WORD_INTERFACE)
	if bool(holds_no_part(subject, tree)) {
		emit_byte(subject, '{')
		emit_byte(subject, '}')
		close_empty_list(subject, tree)
		return
	}
	descend(subject, tree)
	emit_space(subject)
	emit_byte(subject, '{')
	close_line(subject, tree)
	subject.Counts[COUNT_INDENT] = subject.Counts[COUNT_INDENT] + 1
	held_tight := subject.Flags[FLAG_TIGHT]
	subject.Flags[FLAG_TIGHT] = true
	open := Boolean(true)
	for bool(open) {
		print_element(subject, tree)
		open = advance(subject, tree)
	}
	subject.Flags[FLAG_TIGHT] = held_tight
	ascend(subject)
	subject.Counts[COUNT_INDENT] = subject.Counts[COUNT_INDENT] - 1
	emit_indent(subject)
	emit_byte(subject, '}')
}

// Writes one element of a constraint on a line of its own.
func print_element(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "print_element.subject")
	ast.Parse_State_Invariants(tree, "print_element.tree")
	kind := node_kind(subject, tree)
	if kind == ast.NODE_BLANK {
		if bool(closes_block(subject, tree)) {
			subject.Flags[FLAG_BLANK] = true
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
	print_expression(subject, tree)
	close_line(subject, tree)
}

// Reports whether the source states a line feed ahead of the node the walk stands on. The
// canonical form keeps the line the author broke, thus a print reads the break out of the source
// rather than inventing one of its own.
func breaks_before(subject *Printer, tree *ast.Parse_State) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "breaks_before.yes") }()
	Printer_Invariants(subject, "breaks_before.subject")
	ast.Parse_State_Invariants(tree, "breaks_before.tree")
	// A measure asks how wide the form reads on one line, thus it reads no break at all.
	if bool(subject.Flags[FLAG_FLAT]) {
		return false
	}
	leftmost_position(subject, tree)
	position := subject.Positions[POSITION_FOUND]
	if position == 0 {
		return false
	}
	behind := ast.Token_At(tree, ast.Token_Index(position-1))
	ahead := ast.Token_At(tree, ast.Token_Index(position))
	subject.Offsets[OFFSET_FROM] = token.Offset(int(behind.Offset) + int(behind.Size))
	subject.Offsets[OFFSET_TO] = ahead.Offset
	return feeds(subject)
}

// Reports whether the source holds a line feed between two byte offsets.
func feeds(subject *Printer) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "feeds.yes") }()
	Printer_Invariants(subject, "feeds.subject")
	from := subject.Offsets[OFFSET_FROM]
	to := subject.Offsets[OFFSET_TO]
	source := subject.Sources[SOURCE_SLOT]
	if int(to) > len(source) {
		return false
	}
	for offset := int(from); offset < int(to); offset++ {
		if source[offset] == '\n' {
			return true
		}
	}
	return false
}

// Names the run position of the final token the node the walk stands on spans.
func rightmost_position(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "rightmost_position.subject")
	ast.Parse_State_Invariants(tree, "rightmost_position.tree")
	held := subject.Nodes[subject.Counts[COUNT_DEPTH]]
	depth := subject.Counts[COUNT_DEPTH]
	largest := Position(ast.Node_At(tree, held).Token)
	open := descend(subject, tree)
	for bool(open) {
		one := Position(ast.Node_At(tree, subject.Nodes[subject.Counts[COUNT_DEPTH]]).Token)
		if one > largest {
			largest = one
		}
		if bool(descend(subject, tree)) {
			continue
		}
		open = advance(subject, tree)
		for !bool(open) {
			if subject.Counts[COUNT_DEPTH] <= depth+1 {
				open = false
				break
			}
			ascend(subject)
			open = advance(subject, tree)
		}
	}
	subject.Counts[COUNT_DEPTH] = depth
	subject.Nodes[depth] = held
	subject.Positions[POSITION_FOUND] = largest
}

// Reports whether the bracket that closes the form the walk stands on stands on a line of its
// own. A bracket the author left behind the final part stays there, thus the print states the
// break the author stated and never one of its own.
func closes_line(subject *Printer, tree *ast.Parse_State) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "closes_line.yes") }()
	Printer_Invariants(subject, "closes_line.subject")
	ast.Parse_State_Invariants(tree, "closes_line.tree")
	if bool(subject.Flags[FLAG_FLAT]) {
		return false
	}
	closer_position(subject, tree)
	position := subject.Positions[POSITION_FOUND]
	if position == 0 {
		return false
	}
	behind := ast.Token_At(tree, ast.Token_Index(position-1))
	ahead := ast.Token_At(tree, ast.Token_Index(position))
	subject.Offsets[OFFSET_FROM] = token.Offset(int(behind.Offset) + int(behind.Size))
	subject.Offsets[OFFSET_TO] = ahead.Offset
	return feeds(subject)
}

// Names the run position of the bracket that closes the form the walk stands on. The node names
// the bracket its form opens with, thus a count over the brackets behind that one finds the
// bracket that closes it. Zero names a form the run holds no last bracket for.
func closer_position(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "closer_position.subject")
	ast.Parse_State_Invariants(tree, "closer_position.tree")
	node := ast.Node_At(tree, subject.Nodes[subject.Counts[COUNT_DEPTH]])
	subject.Positions[POSITION_FOUND] = 0
	depth := 0
	run := ast.Parse_State_Token_Run(tree)
	for offset := int(node.Token); offset < len(run); offset++ {
		kind := run[offset].Kind
		if kind == token.KIND_END_OF_FILE {
			return
		}
		subject.Signs[SIGN_HELD] = kind
		if bool(opens_bracket(subject)) {
			depth = depth + 1
		}
		subject.Signs[SIGN_HELD] = kind
		if bool(closes_bracket(subject)) {
			depth = depth - 1
			if depth <= 0 {
				subject.Positions[POSITION_FOUND] = Position(offset)
				return
			}
		}
	}
}

// Opens the line one part of a broken form stands on: the comma that closes the part before it,
// the line feed, and the tabs one level deeper than the form.
func emit_break(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "emit_break.subject")
	ast.Parse_State_Invariants(tree, "emit_break.tree")
	emit_byte(subject, ',')
	close_line(subject, tree)
	emit_indent(subject)
}

// Writes one block whose statements stand on the line its brace opens.
func print_inline_block(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "print_inline_block.subject")
	ast.Parse_State_Invariants(tree, "print_inline_block.tree")
	emit_byte(subject, '{')
	open := descend(subject, tree)
	entered := open
	statements := 0
	for bool(open) {
		if bool(trivia_kind(subject, tree)) {
			print_trivia(subject, tree)
			open = advance(subject, tree)
			continue
		}
		if statements > 0 {
			emit_byte(subject, ';')
		}
		emit_space(subject)
		print_statement_form(subject, tree)
		statements = statements + 1
		open = advance(subject, tree)
	}
	if bool(entered) {
		ascend(subject)
	}
	if statements > 0 {
		emit_space(subject)
	}
	emit_byte(subject, '}')
}

// Indents the parts of a form that the author broke behind its first part. A form broken at its
// head takes its indent when it opens, thus only a form broken later owes one here.
func open_break(subject *Printer, here Boolean, broken Boolean) {
	Printer_Invariants(subject, "open_break.subject")
	Boolean_Invariants(here, "open_break.here")
	Boolean_Invariants(broken, "open_break.broken")
	if !bool(here) {
		return
	}
	if bool(broken) {
		return
	}
	subject.Counts[COUNT_INDENT] = subject.Counts[COUNT_INDENT] + 1
}

// Names the nest the parts inside the brackets of one form stand at. A call of one argument
// states that argument as plainly as a statement does, thus only a call of two or more closes up
// the signs its arguments hold.
func inner_nest(subject *Printer, tree *ast.Parse_State, wrap Wrap) {
	Printer_Invariants(subject, "inner_nest.subject")
	ast.Parse_State_Invariants(tree, "inner_nest.tree")
	Wrap_Invariants(wrap, "inner_nest.wrap")
	held := subject.Counts[COUNT_NEST]
	subject.Counts[COUNT_INNER] = held
	if wrap == WRAP_INDEX {
		subject.Counts[COUNT_INNER] = held + 1
		return
	}
	part_count(subject, tree)
	if subject.Counts[COUNT_PART] > 2 {
		subject.Counts[COUNT_INNER] = held + 1
	}
}

// Names the nest the head of one form stands at, which is the value the brackets stand behind.
func head_nest(subject *Printer, wrap Wrap) {
	Printer_Invariants(subject, "head_nest.subject")
	Wrap_Invariants(wrap, "head_nest.wrap")
	subject.Counts[COUNT_HEAD] = subject.Counts[COUNT_INNER]
	if wrap == WRAP_INDEX {
		subject.Counts[COUNT_HEAD] = 1
	}
}

// Reads how many parts the node the walk stands on holds, over the trivia between them.
func part_count(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "part_count.subject")
	ast.Parse_State_Invariants(tree, "part_count.tree")
	held := subject.Nodes[subject.Counts[COUNT_DEPTH]]
	depth := subject.Counts[COUNT_DEPTH]
	count := Count(0)
	open := descend_quiet(subject, tree)
	for bool(open) {
		count = count + 1
		open = advance_quiet(subject, tree)
	}
	subject.Counts[COUNT_DEPTH] = depth
	subject.Nodes[depth] = held
	subject.Counts[COUNT_PART] = count
}

// Reports whether one child of a clause opens the branch its condition fails into. Only an if
// statement holds a second branch, and only a block or a further if statement states one.
func opens_branch(subject *Printer, tree *ast.Parse_State, held Boolean) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "opens_branch.yes") }()
	Printer_Invariants(subject, "opens_branch.subject")
	ast.Parse_State_Invariants(tree, "opens_branch.tree")
	Boolean_Invariants(held, "opens_branch.held")
	if subject.Kinds[KIND_CLAUSE] != ast.NODE_IF {
		return false
	}
	if !bool(held) {
		return false
	}
	child := node_kind(subject, tree)
	return Boolean(child == ast.NODE_BLOCK || child == ast.NODE_IF)
}

// Reports whether the node the walk stands on states one case of a switch or a select.
func opens_case(subject *Printer, tree *ast.Parse_State) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "opens_case.yes") }()
	Printer_Invariants(subject, "opens_case.subject")
	ast.Parse_State_Invariants(tree, "opens_case.tree")
	kind := node_kind(subject, tree)
	return Boolean(kind == ast.NODE_CASE || kind == ast.NODE_DEFAULT)
}

// Reports whether the clause the walk stands on holds one case alone, which is the body that
// closes up against the braces that hold it.
func holds_one_case(subject *Printer, tree *ast.Parse_State) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "holds_one_case.yes") }()
	Printer_Invariants(subject, "holds_one_case.subject")
	ast.Parse_State_Invariants(tree, "holds_one_case.tree")
	held := subject.Nodes[subject.Counts[COUNT_DEPTH]]
	depth := subject.Counts[COUNT_DEPTH]
	cases := 0
	open := descend_quiet(subject, tree)
	for bool(open) {
		if bool(opens_case(subject, tree)) {
			cases = cases + 1
		}
		open = advance_quiet(subject, tree)
	}
	subject.Counts[COUNT_DEPTH] = depth
	subject.Nodes[depth] = held
	return Boolean(cases == 1)
}

// Reports whether the head of the case the walk stands on reads on one line: the word, the values
// behind it, and the colon that closes them, inside the width a short line states. A note the
// author left inside that head names the line it stands on, thus a head that holds one keeps the
// lines the author wrote.
func fits_case(subject *Printer, tree *ast.Parse_State) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "fits_case.yes") }()
	Printer_Invariants(subject, "fits_case.subject")
	ast.Parse_State_Invariants(tree, "fits_case.tree")
	colon := int(subject.Positions[POSITION_FROM])
	opening := int(ast.Node_At(tree, subject.Nodes[subject.Counts[COUNT_DEPTH]]).Token)
	for place := opening; place < colon; place = place + 1 {
		if ast.Token_At(tree, ast.Token_Index(place)).Kind == token.KIND_COMMENT {
			return false
		}
	}
	held := subject.Nodes[subject.Counts[COUNT_DEPTH]]
	depth := subject.Counts[COUNT_DEPTH]
	// The word one case opens with and the space behind it stand ahead of every value, and
	// the colon closes them, thus the head writes four bytes no value states.
	width := 6
	values := 0
	open := descend_quiet(subject, tree)
	if bool(open) {
		if node_kind(subject, tree) == ast.NODE_EXPRESSION_STATEMENT {
			open = descend_quiet(subject, tree)
			for bool(open) {
				measure_expression(subject, tree)
				width = width + int(subject.Spans[SPAN_FOUND]) + 2
				values = values + 1
				open = advance_quiet(subject, tree)
			}
		}
	}
	subject.Counts[COUNT_DEPTH] = depth
	subject.Nodes[depth] = held
	if values == 0 {
		return false
	}
	return Boolean(width-2 <= CASE_WIDTH_MAXIMUM)
}

// Writes the values one case stands for. The tree hangs the whole run off one node, thus the
// print walks that node rather than the case itself.
func print_case_values(subject *Printer, tree *ast.Parse_State, flat Boolean) {
	Printer_Invariants(subject, "print_case_values.subject")
	ast.Parse_State_Invariants(tree, "print_case_values.tree")
	Boolean_Invariants(flat, "print_case_values.flat")
	// A select states a whole statement where a switch states a value, thus the print writes
	// the form of the child rather than an expression the child may not be.
	if node_kind(subject, tree) != ast.NODE_EXPRESSION_STATEMENT {
		emit_space(subject)
		print_statement_form(subject, tree)
		return
	}
	open := descend_part(subject, tree)
	entered := open
	values := 0
	broken := Boolean(false)
	for bool(open) {
		if values > 0 {
			emit_byte(subject, ',')
		}
		here := Boolean(false)
		if values > 0 {
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
		values = values + 1
		open = advance_part(subject, tree)
	}
	if bool(broken) {
		subject.Counts[COUNT_INDENT] = subject.Counts[COUNT_INDENT] - 1
	}
	if bool(entered) {
		finish_parts(subject, tree)
		ascend(subject)
	}
}

// Reports whether the form the walk stands on writes every part of itself on the line it opens
// on. A struct the author wrote on one line stands on one line, thus a table of cases reads as a
// table rather than as a page of fields.
func spans_one_line(subject *Printer, tree *ast.Parse_State) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "spans_one_line.yes") }()
	Printer_Invariants(subject, "spans_one_line.subject")
	ast.Parse_State_Invariants(tree, "spans_one_line.tree")
	closer_position(subject, tree)
	closer := subject.Positions[POSITION_FOUND]
	if closer == 0 {
		return false
	}
	node := ast.Node_At(tree, subject.Nodes[subject.Counts[COUNT_DEPTH]])
	opening := ast.Token_At(tree, node.Token)
	ahead := ast.Token_At(tree, ast.Token_Index(closer))
	subject.Offsets[OFFSET_FROM] = token.Offset(int(opening.Offset) + int(opening.Size))
	subject.Offsets[OFFSET_TO] = ahead.Offset
	return !feeds(subject)
}

// Writes one struct type the author stated on one line, the fields between semicolons.
func print_inline_structure(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "print_inline_structure.subject")
	ast.Parse_State_Invariants(tree, "print_inline_structure.tree")
	emit_byte(subject, '{')
	open := descend(subject, tree)
	entered := open
	fields := 0
	for bool(open) {
		if node_kind(subject, tree) != ast.NODE_FIELD {
			// The empty line behind the brace stands behind the declaration the
			// struct closes, thus the print still owes it.
			print_trivia(subject, tree)
			open = advance(subject, tree)
			continue
		}
		if fields > 0 {
			emit_byte(subject, ';')
		}
		emit_space(subject)
		print_inline_field(subject, tree)
		fields = fields + 1
		open = advance(subject, tree)
	}
	if bool(entered) {
		ascend(subject)
	}
	if fields > 0 {
		emit_space(subject)
	}
	emit_byte(subject, '}')
}

// Writes one field of a struct the author stated on one line: the names it binds and the type
// behind them.
func print_inline_field(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "print_inline_field.subject")
	ast.Parse_State_Invariants(tree, "print_inline_field.tree")
	open := descend_quiet(subject, tree)
	entered := open
	names := 0
	for bool(open) {
		if node_kind(subject, tree) != ast.NODE_FIELD_NAME {
			break
		}
		if names > 0 {
			emit_byte(subject, ',')
			emit_space(subject)
		}
		emit_token(subject, tree)
		names = names + 1
		open = advance_quiet(subject, tree)
	}
	if bool(open) {
		if names > 0 {
			emit_space(subject)
		}
		print_expression(subject, tree)
	}
	if bool(entered) {
		ascend(subject)
	}
}

// Writes one grouping: the parentheses the author wrote and the value between them. Parentheses
// state the binding the signs inside them would otherwise state, thus the value inside stands one
// level shallower than the form around it.
func print_group(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "print_group.subject")
	ast.Parse_State_Invariants(tree, "print_group.tree")
	held := subject.Counts[COUNT_NEST]
	inner := Count(1)
	if held > 1 {
		inner = held - 1
	}
	emit_byte(subject, '(')
	open := descend_part(subject, tree)
	entered := open
	if bool(open) {
		subject.Counts[COUNT_NEST] = inner
		print_expression(subject, tree)
		subject.Counts[COUNT_NEST] = held
	}
	if bool(entered) {
		finish_parts(subject, tree)
		ascend(subject)
	}
	emit_byte(subject, ')')
}

// Writes the semicolons the source states between two run positions. A semicolon a line feed
// stands for spans no byte, thus only a semicolon the author wrote counts.
func emit_marks(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "emit_marks.subject")
	ast.Parse_State_Invariants(tree, "emit_marks.tree")
	from := subject.Positions[POSITION_FROM]
	to := subject.Positions[POSITION_TO]
	depth := 0
	for position := int(from); position < int(to); position++ {
		one := ast.Token_At(tree, ast.Token_Index(position))
		subject.Signs[SIGN_HELD] = one.Kind
		if bool(opens_bracket(subject)) {
			depth = depth + 1
		}
		subject.Signs[SIGN_HELD] = one.Kind
		if bool(closes_bracket(subject)) {
			depth = depth - 1
		}
		if depth != 0 {
			continue
		}
		subject.Positions[POSITION_FOUND] = Position(position)
		if !bool(states_mark(subject, tree)) {
			continue
		}
		emit_byte(subject, ';')
		emit_space(subject)
	}
}

// Reads how many semicolons the source states between two run positions.
func count_marks(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "count_marks.subject")
	ast.Parse_State_Invariants(tree, "count_marks.tree")
	from := subject.Positions[POSITION_FROM]
	to := subject.Positions[POSITION_TO]
	count := Count(0)
	depth := 0
	for position := int(from); position < int(to); position++ {
		one := ast.Token_At(tree, ast.Token_Index(position))
		subject.Signs[SIGN_HELD] = one.Kind
		if bool(opens_bracket(subject)) {
			depth = depth + 1
		}
		subject.Signs[SIGN_HELD] = one.Kind
		if bool(closes_bracket(subject)) {
			depth = depth - 1
		}
		if depth != 0 {
			continue
		}
		subject.Positions[POSITION_FOUND] = Position(position)
		if !bool(states_mark(subject, tree)) {
			continue
		}
		count = count + 1
	}
	subject.Counts[COUNT_MARK] = count
}

// Reports whether one assignment states more than one name and more than one value. The values of
// such an assignment stand as parts of a run rather than as statements of their own.
func assigns_runs(subject *Printer, tree *ast.Parse_State) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "assigns_runs.yes") }()
	Printer_Invariants(subject, "assigns_runs.subject")
	ast.Parse_State_Invariants(tree, "assigns_runs.tree")
	held := subject.Nodes[subject.Counts[COUNT_DEPTH]]
	depth := subject.Counts[COUNT_DEPTH]
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
	subject.Counts[COUNT_DEPTH] = depth
	subject.Nodes[depth] = held
	return Boolean(names > 1 && values > 1)
}

// Reads how wide one form writes and leaves the print stand where it stood. The storage the
// caller owns holds the bytes the measure wrote, and the count of written bytes returns, thus the
// print writes over them again when it writes the form for real.
// Reads how many columns the form the walk stands on writes when it stands on one line, which is
// what says whether the line it opens runs past the widest one. The measure writes the form past
// the bytes the print already wrote and reads them back, thus it costs no storage of its own.
func measure_flat(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "measure_flat.subject")
	ast.Parse_State_Invariants(tree, "measure_flat.tree")
	counts := subject.Counts
	flags := subject.Flags
	spans := subject.Spans
	node := subject.Nodes[subject.Counts[COUNT_DEPTH]]
	start := subject.Counts[COUNT_WRITTEN]
	subject.Flags[FLAG_FLAT] = true
	print_expression(subject, tree)
	subject.Offsets[OFFSET_FROM] = token.Offset(start)
	subject.Offsets[OFFSET_TO] = token.Offset(subject.Counts[COUNT_WRITTEN])
	take_columns(subject)
	columns := subject.Counts[COUNT_COLUMNS]
	subject.Counts = counts
	subject.Flags = flags
	subject.Spans = spans
	subject.Nodes[subject.Counts[COUNT_DEPTH]] = node
	subject.Counts[COUNT_SPAN] = columns
	take_span(subject)
}

// Reports whether the signature the walk stands at the opening of runs the line past the widest
// line. The signature holds no node of its own, thus the measure writes the parameters and the
// results the walk stands ahead of and reads the columns they wrote.
func runs_over_signature(
	subject *Printer, tree *ast.Parse_State, open Boolean,
) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "runs_over_signature.yes") }()
	Printer_Invariants(subject, "runs_over_signature.subject")
	ast.Parse_State_Invariants(tree, "runs_over_signature.tree")
	Boolean_Invariants(open, "runs_over_signature.open")
	if bool(subject.Flags[FLAG_FLAT]) {
		return false
	}
	take_line_width(subject)
	line := subject.Spans[SPAN_LINE]
	counts := subject.Counts
	flags := subject.Flags
	spans := subject.Spans
	node := subject.Nodes[subject.Counts[COUNT_DEPTH]]
	start := subject.Counts[COUNT_WRITTEN]
	subject.Flags[FLAG_FLAT] = true
	stand := print_signature_parameters(subject, tree, open, false)
	print_signature_results(subject, tree, stand)
	subject.Offsets[OFFSET_FROM] = token.Offset(start)
	subject.Offsets[OFFSET_TO] = token.Offset(subject.Counts[COUNT_WRITTEN])
	take_columns(subject)
	columns := int(subject.Counts[COUNT_COLUMNS])
	subject.Counts = counts
	subject.Flags = flags
	subject.Spans = spans
	subject.Nodes[subject.Counts[COUNT_DEPTH]] = node
	// The brace of the body stands on the line the signature closes, thus it counts against
	// the widest line the same way every other byte of that line does.
	return Boolean(int(line)+columns+len(BODY_HEAD) > COLUMN_WIDTH_MAXIMUM)
}

// Reports whether the results the walk stands at the opening of run the line past the widest one.
// The measure writes the results flat and reads the columns they wrote, and a measure inside a
// measure answers false, thus the writing it runs states no measure of its own.
func runs_over_results(
	subject *Printer, tree *ast.Parse_State, open Boolean,
) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "runs_over_results.yes") }()
	Printer_Invariants(subject, "runs_over_results.subject")
	ast.Parse_State_Invariants(tree, "runs_over_results.tree")
	Boolean_Invariants(open, "runs_over_results.open")
	if bool(subject.Flags[FLAG_FLAT]) {
		return false
	}
	take_line_width(subject)
	line := subject.Spans[SPAN_LINE]
	counts := subject.Counts
	flags := subject.Flags
	spans := subject.Spans
	node := subject.Nodes[subject.Counts[COUNT_DEPTH]]
	start := subject.Counts[COUNT_WRITTEN]
	subject.Flags[FLAG_FLAT] = true
	print_signature_results(subject, tree, open)
	subject.Offsets[OFFSET_FROM] = token.Offset(start)
	subject.Offsets[OFFSET_TO] = token.Offset(subject.Counts[COUNT_WRITTEN])
	take_columns(subject)
	columns := int(subject.Counts[COUNT_COLUMNS])
	subject.Counts = counts
	subject.Flags = flags
	subject.Spans = spans
	subject.Nodes[subject.Counts[COUNT_DEPTH]] = node
	return Boolean(int(line)+columns+len(BODY_HEAD) > COLUMN_WIDTH_MAXIMUM)
}

// BODY_HEAD is the text that stands between one signature and the body it opens.
const BODY_HEAD = " {"

// Reports whether the form the walk stands on runs the line it opens past the widest line, which
// is what states one part of that form to a line.
func runs_over(subject *Printer, tree *ast.Parse_State) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "runs_over.yes") }()
	Printer_Invariants(subject, "runs_over.subject")
	ast.Parse_State_Invariants(tree, "runs_over.tree")
	// A measure inside a measure answers nothing the outer one asks, because the outer print
	// already stands flat and holds every break it would take.
	if bool(subject.Flags[FLAG_FLAT]) {
		return false
	}
	take_line_width(subject)
	line := subject.Spans[SPAN_LINE]
	measure_flat(subject, tree)
	return Boolean(int(line)+int(subject.Spans[SPAN_FOUND]) > COLUMN_WIDTH_MAXIMUM)
}

func measure_expression(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "measure_expression.subject")
	ast.Parse_State_Invariants(tree, "measure_expression.tree")
	counts := subject.Counts
	flags := subject.Flags
	node := subject.Nodes[subject.Counts[COUNT_DEPTH]]
	start := subject.Counts[COUNT_WRITTEN]
	print_expression(subject, tree)
	written := subject.Counts[COUNT_WRITTEN] - start
	subject.Counts = counts
	subject.Flags = flags
	subject.Nodes[subject.Counts[COUNT_DEPTH]] = node
	subject.Counts[COUNT_SPAN] = written
	take_span(subject)
}

// Reads the columns the line the print stands on already holds. One character stands in one
// column however many bytes it spans, and a tab stands in the columns a reader sees, thus the read
// counts the bytes that open a character and counts no byte that continues one.
func take_line_width(subject *Printer) {
	Printer_Invariants(subject, "take_line_width.subject")
	form := subject.Forms[FORM_SLOT]
	written := int(subject.Counts[COUNT_WRITTEN])
	if written > len(form) {
		written = len(form)
	}
	place := written
	for place > 0 {
		if form[place-1] == '\n' {
			break
		}
		place = place - 1
	}
	subject.Offsets[OFFSET_FROM] = token.Offset(place)
	subject.Offsets[OFFSET_TO] = token.Offset(written)
	take_columns(subject)
	subject.Spans[SPAN_LINE] = Width(subject.Counts[COUNT_COLUMNS])
}

// Counts the columns the form holds between two bytes of it, which is what a reader of the line
// sees rather than what the storage spends.
func take_columns(subject *Printer) {
	Printer_Invariants(subject, "take_columns.subject")
	form := subject.Forms[FORM_SLOT]
	columns := 0
	to := int(subject.Offsets[OFFSET_TO])
	for place := int(subject.Offsets[OFFSET_FROM]); place < to; place++ {
		if form[place] == '\t' {
			columns = columns + TAB_WIDTH
			continue
		}
		if form[place]&CHARACTER_TAIL_MASK == CHARACTER_TAIL_MARK {
			continue
		}
		columns = columns + 1
	}
	subject.Counts[COUNT_COLUMNS] = Count(columns)
}

// Writes the width one count of written bytes states into the slot a measure answers through. A
// count that runs past the widest width a run answers for states that width instead.
func take_span(subject *Printer) {
	Printer_Invariants(subject, "take_span.subject")
	written := subject.Counts[COUNT_SPAN]
	subject.Spans[SPAN_FOUND] = Width(written)
	if written > WIDTH_MAXIMUM {
		subject.Spans[SPAN_FOUND] = WIDTH_MAXIMUM
	}
}

// Writes the spaces that carry the print to one column.
func emit_column(subject *Printer) {
	Printer_Invariants(subject, "emit_column.subject")
	width := subject.Spans[SPAN_FOUND]
	held := subject.Spans[SPAN_COLUMN]
	if held < width {
		held = width
	}
	for range int(held-width) + 1 {
		emit_space(subject)
	}
}

// Reads how many bytes stand written on the line the print stands on.
func line_width(subject *Printer) {
	Printer_Invariants(subject, "line_width.subject")
	subject.Counts[COUNT_SPAN] = subject.Counts[COUNT_WRITTEN] -
		subject.Counts[COUNT_OPENING]
	take_span(subject)
}

// Reads the column the notes of the run that opens at the statement the walk stands on write at.
// A statement that carries no note closes the run, thus a run holds the lines a reader reads as
// one table.
func take_statement_column(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "take_statement_column.subject")
	ast.Parse_State_Invariants(tree, "take_statement_column.tree")
	held := subject.Nodes[subject.Counts[COUNT_DEPTH]]
	widest := Width(0)
	open := Boolean(true)
	for bool(open) {
		if bool(trivia_kind(subject, tree)) {
			break
		}
		if !bool(trails_comment(subject, tree)) {
			break
		}
		// A statement that runs over lines of its own carries its note wherever that
		// note fell, thus it stands in no column with the statements around it.
		if bool(spans_form(subject, tree)) {
			break
		}
		measure_statement(subject, tree)
		if subject.Spans[SPAN_FOUND] > widest {
			widest = subject.Spans[SPAN_FOUND]
		}
		open = advance(subject, tree)
		if bool(open) {
			open = advance(subject, tree)
		}
	}
	subject.Nodes[subject.Counts[COUNT_DEPTH]] = held
	subject.Counts[COUNT_COLUMN] = Count(widest)
}

// Reports whether a note stands behind the node the walk stands on, on the line that node closes.
func trails_comment(subject *Printer, tree *ast.Parse_State) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "trails_comment.yes") }()
	Printer_Invariants(subject, "trails_comment.subject")
	ast.Parse_State_Invariants(tree, "trails_comment.tree")
	// The tree hangs a note wherever the walk reached it, thus the run of tokens states which
	// line the note stands on rather than the child chain that holds it.
	held := subject.Nodes[subject.Counts[COUNT_DEPTH]]
	leftmost_position(subject, tree)
	start := subject.Positions[POSITION_FOUND]
	limit := Position(ast.TOKEN_INDEX_MAXIMUM)
	noted := Boolean(false)
	last := !bool(advance(subject, tree))
	if !last {
		leftmost_position(subject, tree)
		limit = subject.Positions[POSITION_FOUND]
		// A list hangs a note beside the element it follows, thus the note stands as
		// the sibling of that element rather than inside it.
		noted = Boolean(node_kind(subject, tree) == ast.NODE_COMMENT)
	}
	subject.Nodes[subject.Counts[COUNT_DEPTH]] = held
	if bool(noted) {
		subject.Positions[POSITION_FOUND] = limit
		return !stands_alone(subject, tree)
	}
	for position := int(start) + 1; position < int(limit); position++ {
		one := ast.Token_At(tree, ast.Token_Index(position))
		if one.Kind == token.KIND_END_OF_FILE {
			return false
		}
		subject.Positions[POSITION_FOUND] = Position(position)
		if bool(stands_alone(subject, tree)) {
			return false
		}
		if one.Kind != token.KIND_COMMENT {
			continue
		}
		// A note the statement behind it opens right after closes the line, and one the
		// statement runs on past stands inside the statement rather than behind it.
		return Boolean(last || limit <= Position(position)+2)
	}
	return false
}

// Reads how wide one statement writes and leaves the print stand where it stood.
func measure_statement(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "measure_statement.subject")
	ast.Parse_State_Invariants(tree, "measure_statement.tree")
	counts := subject.Counts
	flags := subject.Flags
	node := subject.Nodes[subject.Counts[COUNT_DEPTH]]
	start := subject.Counts[COUNT_WRITTEN]
	emit_indent(subject)
	print_statement_form(subject, tree)
	written := subject.Counts[COUNT_WRITTEN] - start
	subject.Counts = counts
	subject.Flags = flags
	subject.Nodes[subject.Counts[COUNT_DEPTH]] = node
	subject.Counts[COUNT_SPAN] = written
	take_span(subject)
}

// Reports whether an empty line stands ahead of the comment at one run position.
func opens_run(subject *Printer, tree *ast.Parse_State) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "opens_run.yes") }()
	Printer_Invariants(subject, "opens_run.subject")
	ast.Parse_State_Invariants(tree, "opens_run.tree")
	position := subject.Positions[POSITION_FOUND]
	if position == 0 {
		return false
	}
	return Boolean(ast.Token_At(tree, ast.Token_Index(position)).Blanks > 0)
}

// Reports whether the node the walk stands on names a token behind the bracket at one position.
func stands_behind(subject *Printer, tree *ast.Parse_State) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "stands_behind.yes") }()
	Printer_Invariants(subject, "stands_behind.subject")
	ast.Parse_State_Invariants(tree, "stands_behind.tree")
	node := ast.Node_At(tree, subject.Nodes[subject.Counts[COUNT_DEPTH]])
	return Boolean(Position(node.Token) > subject.Positions[POSITION_FROM])
}

// Reads the column the notes of the run of elements that opens where the walk stands write at.
// An element that carries no note closes the run, thus the notes of one broken list read as one
// column rather than as a ragged edge.
func take_element_column(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "take_element_column.subject")
	ast.Parse_State_Invariants(tree, "take_element_column.tree")
	held := subject.Nodes[subject.Counts[COUNT_DEPTH]]
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
		// The line one element writes holds the tabs the list opens at, the element,
		// and the comma that closes it.
		measure_expression(subject, tree)
		width := Width(subject.Counts[COUNT_INDENT]) + subject.Spans[SPAN_FOUND] + 1
		if width > widest {
			widest = width
		}
		open = advance(subject, tree)
		if bool(open) {
			if bool(trivia_kind(subject, tree)) {
				open = advance(subject, tree)
			}
		}
	}
	subject.Nodes[subject.Counts[COUNT_DEPTH]] = held
	subject.Counts[COUNT_COLUMN] = Count(widest)
}

// Reports whether the node the walk stands on states a note on the line the brace of the block
// opens.
func trails_brace(subject *Printer, tree *ast.Parse_State) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "trails_brace.yes") }()
	Printer_Invariants(subject, "trails_brace.subject")
	ast.Parse_State_Invariants(tree, "trails_brace.tree")
	if node_kind(subject, tree) != ast.NODE_COMMENT {
		return false
	}
	subject.Positions[POSITION_FOUND] = Position(ast.Node_At(tree,
		subject.Nodes[subject.Counts[COUNT_DEPTH]]).Token)
	return !stands_alone(subject, tree)
}

// Reports whether one token opens a bracketed form.
func opens_bracket(subject *Printer) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "opens_bracket.yes") }()
	Printer_Invariants(subject, "opens_bracket.subject")
	kind := subject.Signs[SIGN_HELD]
	if kind == token.KIND_PARENTHESIS_LEFT {
		return true
	}
	if kind == token.KIND_BRACKET_LEFT {
		return true
	}
	return Boolean(kind == token.KIND_BRACE_LEFT)
}

// Reports whether one token closes a bracketed form.
func closes_bracket(subject *Printer) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "closes_bracket.yes") }()
	Printer_Invariants(subject, "closes_bracket.subject")
	kind := subject.Signs[SIGN_HELD]
	if kind == token.KIND_PARENTHESIS_RIGHT {
		return true
	}
	if kind == token.KIND_BRACKET_RIGHT {
		return true
	}
	return Boolean(kind == token.KIND_BRACE_RIGHT)
}

// Reports whether one token states a semicolon the author wrote at the depth one scan opened at.
// A semicolon a line feed stands for spans no byte, thus it is no mark of the form.
func states_mark(subject *Printer, tree *ast.Parse_State) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "states_mark.yes") }()
	Printer_Invariants(subject, "states_mark.subject")
	ast.Parse_State_Invariants(tree, "states_mark.tree")
	one := ast.Token_At(tree, ast.Token_Index(subject.Positions[POSITION_FOUND]))
	if one.Kind != token.KIND_SEMICOLON {
		return false
	}
	return Boolean(one.Size != 0)
}

// Reports whether the child the walk stands on states the further clause an else branch opens
// with, which closes its own line and leaves the clause around it none to close.
func opens_else_clause(subject *Printer, tree *ast.Parse_State, held Boolean) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "opens_else_clause.yes") }()
	Printer_Invariants(subject, "opens_else_clause.subject")
	ast.Parse_State_Invariants(tree, "opens_else_clause.tree")
	Boolean_Invariants(held, "opens_else_clause.held")
	if !bool(held) {
		return false
	}
	return Boolean(node_kind(subject, tree) == ast.NODE_IF)
}

// Reports whether the part the print stands at opens the column of a broken form, which every
// part of that form opens at however deep the part before it left the print.
func open_column(here Boolean, opened Boolean) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "open_column.yes") }()
	Boolean_Invariants(here, "open_column.here")
	Boolean_Invariants(opened, "open_column.opened")
	if bool(here) {
		return true
	}
	return opened
}

// Reads the column the notes of a broken form open at, which only a broken form holds.
func take_notes(subject *Printer, tree *ast.Parse_State, opened Boolean) {
	Printer_Invariants(subject, "take_notes.subject")
	ast.Parse_State_Invariants(tree, "take_notes.tree")
	Boolean_Invariants(opened, "take_notes.opened")
	if !bool(opened) {
		return
	}
	if subject.Counts[COUNT_COLUMN] != 0 {
		return
	}
	take_element_column(subject, tree)
}

// Closes the column of notes wherever the part the print wrote carries none behind it.
func close_notes(subject *Printer, opened Boolean, trails Boolean) {
	Printer_Invariants(subject, "close_notes.subject")
	Boolean_Invariants(opened, "close_notes.opened")
	Boolean_Invariants(trails, "close_notes.trails")
	if !bool(opened) {
		return
	}
	if bool(trails) {
		return
	}
	subject.Counts[COUNT_COLUMN] = 0
}

// Writes the type parameters of one signature and reports where the walk stopped. A signature
// that binds no type parameter writes no brackets at all.
func print_type_parameters(
	subject *Printer, tree *ast.Parse_State, open Boolean,
) (stand Boolean) {
	defer func() { Boolean_Invariants(stand, "print_type_parameters.stand") }()
	Printer_Invariants(subject, "print_type_parameters.subject")
	ast.Parse_State_Invariants(tree, "print_type_parameters.tree")
	Boolean_Invariants(open, "print_type_parameters.open")
	stand = open
	parameters := 0
	broken := Boolean(false)
	opened := Boolean(false)
	tail := Boolean(false)
	column := subject.Counts[COUNT_INDENT]
	for bool(stand) {
		if node_kind(subject, tree) != ast.NODE_TYPE_PARAMETER {
			break
		}
		here := breaks_before(subject, tree)
		if parameters == 0 {
			emit_byte(subject, '[')
			column = subject.Counts[COUNT_INDENT]
			tail = closes_list(subject, tree)
			print_first_break(subject, tree, here)
			opened = here
			broken = here || tail
		}
		if parameters > 0 {
			open_break(subject, here, opened)
			opened = opened || here
			broken = broken || here
			print_element_break(subject, tree, here)
		}
		print_type_parameter(subject, tree)
		parameters = parameters + 1
		stand = advance_part(subject, tree)
	}
	if bool(broken) {
		subject.Counts[COUNT_COLUMN_INDENT] = column
		close_broken(subject, tree, tail)
	}
	if parameters > 0 {
		emit_byte(subject, ']')
	}
	return stand
}

// Writes the semicolons and the space that stand between the headers of one clause and the body
// it opens, and returns the tabs and the break state the body stands at.
func open_body(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "open_body.subject")
	ast.Parse_State_Invariants(tree, "open_body.tree")
	cursor := subject.Positions[POSITION_FROM]
	indent := subject.Counts[COUNT_COLUMN_INDENT]
	leftmost_position(subject, tree)
	subject.Positions[POSITION_FROM] = cursor
	subject.Positions[POSITION_TO] = subject.Positions[POSITION_FOUND]
	count_marks(subject, tree)
	marks := subject.Counts[COUNT_MARK]
	emit_marks(subject, tree)
	subject.Counts[COUNT_INDENT] = indent
	// The body of a clause opens a line of its own, thus no continuation of the header stands
	// open where it opens.
	subject.Flags[FLAG_BROKEN] = false
	if marks == 0 {
		emit_space(subject)
	}
}

// Writes one header of a clause and the semicolons that stand ahead of it, and returns the run
// position the header opened at, which the header behind it counts semicolons from.
func print_header(subject *Printer, tree *ast.Parse_State, first Boolean) {
	Printer_Invariants(subject, "print_header.subject")
	ast.Parse_State_Invariants(tree, "print_header.tree")
	Boolean_Invariants(first, "print_header.first")
	cursor := subject.Positions[POSITION_FROM]
	leftmost_position(subject, tree)
	start := subject.Positions[POSITION_FOUND]
	// A loop that leaves a header out still states the semicolon that header stands between,
	// thus a space stands on each side of that semicolon.
	subject.Positions[POSITION_FROM] = cursor
	subject.Positions[POSITION_TO] = start
	count_marks(subject, tree)
	marks := subject.Counts[COUNT_MARK]
	if bool(first) {
		if marks > 0 {
			emit_space(subject)
		}
	}
	emit_marks(subject, tree)
	if marks == 0 {
		emit_space(subject)
	}
	print_statement_form(subject, tree)
	// The header behind this one counts the semicolons that stand from here, thus the slot
	// states where this header opened rather than where the clause did.
	subject.Positions[POSITION_FROM] = start
}

// Writes the three points one call states behind the value it spreads. The tree holds a node
// behind that value rather than on the points, thus the print writes them of its own.
func emit_spread(subject *Printer) {
	Printer_Invariants(subject, "emit_spread.subject")
	emit_byte(subject, '.')
	emit_byte(subject, '.')
	emit_byte(subject, '.')
}

// Writes one part of a broken form and holds the column its note stands in.
func print_part(subject *Printer, tree *ast.Parse_State, opened Boolean, behind Boolean) {
	Printer_Invariants(subject, "print_part.subject")
	ast.Parse_State_Invariants(tree, "print_part.tree")
	Boolean_Invariants(opened, "print_part.opened")
	Boolean_Invariants(behind, "print_part.behind")
	if bool(behind) {
		take_notes(subject, tree, opened)
	}
	trails := trails_comment(subject, tree)
	print_expression(subject, tree)
	close_notes(subject, opened, trails)
}

// Opens the column the values of a run of pairs stand in. A run holds the pairs that each open a
// line of their own, thus the pairs behind a closed run answer for a width of their own, and a
// pair that runs over lines of its own opens a run rather than joining one.
func open_key_run(subject *Printer, tree *ast.Parse_State, here Boolean) {
	Printer_Invariants(subject, "open_key_run.subject")
	ast.Parse_State_Invariants(tree, "open_key_run.tree")
	Boolean_Invariants(here, "open_key_run.here")
	if !bool(here) {
		return
	}
	if bool(spans_form(subject, tree)) {
		subject.Counts[COUNT_WIDTH] = 0
	}
	if subject.Counts[COUNT_WIDTH] != 0 {
		return
	}
	take_key_width(subject, tree)
}

// Closes the column of a run of pairs wherever the element the print wrote closed the run: an
// element that carries trivia behind it, and one that ran over lines of its own.
func close_key_run(subject *Printer, tree *ast.Parse_State, spans Boolean) {
	Printer_Invariants(subject, "close_key_run.subject")
	ast.Parse_State_Invariants(tree, "close_key_run.tree")
	Boolean_Invariants(spans, "close_key_run.spans")
	if bool(breaks_run(subject, tree)) {
		subject.Counts[COUNT_WIDTH] = 0
	}
	if bool(spans) {
		subject.Counts[COUNT_WIDTH] = 0
	}
}

// Reports whether one literal states a type ahead of its brace. A literal that stands as an
// element of another one states none.
func typed_head(subject *Printer, tree *ast.Parse_State, open Boolean) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "typed_head.yes") }()
	Printer_Invariants(subject, "typed_head.subject")
	ast.Parse_State_Invariants(tree, "typed_head.tree")
	Boolean_Invariants(open, "typed_head.open")
	brace := subject.Positions[POSITION_FROM]
	if !bool(open) {
		return false
	}
	leftmost_position(subject, tree)
	return Boolean(subject.Positions[POSITION_FOUND] < brace)
}

// Writes one case of a switch, and the brace its run of cases opens at. The tree hangs the cases
// of a switch off the switch itself, thus the print writes the braces they stand between.
func print_switch_case(subject *Printer, tree *ast.Parse_State, first Boolean) {
	Printer_Invariants(subject, "print_switch_case.subject")
	ast.Parse_State_Invariants(tree, "print_switch_case.tree")
	Boolean_Invariants(first, "print_switch_case.first")
	if bool(first) {
		open_body(subject, tree)
		emit_byte(subject, '{')
		close_line(subject, tree)
	}
	print_case_line(subject, tree)
}

// Writes one number literal in the form the canonical source states: a base prefix and an
// exponent mark in lower case, and no leading zeros on an imaginary whole number. The digits
// themselves stand as the author wrote them, because case carries meaning in no base.
func emit_number(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "emit_number.subject")
	ast.Parse_State_Invariants(tree, "emit_number.tree")
	text := subject.Sources[SOURCE_TEXT]
	subject.Counts[COUNT_DIGIT] = 0
	// A literal of one byte states neither a base prefix nor an exponent.
	if len(text) < 2 {
		emit_digits(subject, tree, EXPONENT_NONE)
		return
	}
	base := number_base(subject, tree)
	if base != BASE_NONE {
		emit_byte(subject, '0')
		emit_byte(subject, Symbol(base))
		subject.Counts[COUNT_DIGIT] = 2
		// Only a hexadecimal literal states a power, and no other base states an
		// exponent at all.
		exponent := EXPONENT_NONE
		if base == BASE_HEXADECIMAL {
			exponent = EXPONENT_POWER
		}
		emit_digits(subject, tree, exponent)
		return
	}
	// An exponent stands where a whole number cannot, thus a literal that holds one keeps every
	// zero it opens with.
	if bool(holds_exponent(subject, tree)) {
		emit_digits(subject, tree, EXPONENT_DECIMAL)
		return
	}
	if !bool(whole_imaginary(subject, tree)) {
		// A whole number the source opens with a zero states the eight base, and the
		// canonical form names that base rather than leaving a reader to count digits.
		if bool(states_octal(subject, tree)) {
			emit_byte(subject, '0')
			emit_byte(subject, Symbol(BASE_OCTAL))
			subject.Counts[COUNT_DIGIT] = 1
		}
		emit_digits(subject, tree, EXPONENT_NONE)
		return
	}
	number_head(subject, tree)
	emit_digits(subject, tree, EXPONENT_NONE)
}

// Names the base one number literal states, in the case the canonical form writes it. Zero names
// a literal that states no base prefix at all.
func number_base(subject *Printer, tree *ast.Parse_State) (base Base) {
	defer func() { Base_Invariants(base, "number_base.base") }()
	Printer_Invariants(subject, "number_base.subject")
	ast.Parse_State_Invariants(tree, "number_base.tree")
	text := subject.Sources[SOURCE_TEXT]
	if text[0] != '0' {
		return BASE_NONE
	}
	if text[1] == 'X' {
		return BASE_HEXADECIMAL
	}
	if text[1] == 'x' {
		return BASE_HEXADECIMAL
	}
	if text[1] == 'O' {
		return BASE_OCTAL
	}
	if text[1] == 'o' {
		return BASE_OCTAL
	}
	if text[1] == 'B' {
		return BASE_BINARY
	}
	if text[1] == 'b' {
		return BASE_BINARY
	}
	return BASE_NONE
}

// Reports whether one number literal states an exponent or a fraction, which a whole number
// states neither of.
func holds_exponent(subject *Printer, tree *ast.Parse_State) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "holds_exponent.yes") }()
	Printer_Invariants(subject, "holds_exponent.subject")
	ast.Parse_State_Invariants(tree, "holds_exponent.tree")
	text := subject.Sources[SOURCE_TEXT]
	for index := range len(text) {
		if text[index] == 'E' {
			return true
		}
	}
	return false
}

// Reports whether one number literal states an imaginary whole number, which is the one literal
// the canonical form strips leading zeros from.
func whole_imaginary(subject *Printer, tree *ast.Parse_State) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "whole_imaginary.yes") }()
	Printer_Invariants(subject, "whole_imaginary.subject")
	ast.Parse_State_Invariants(tree, "whole_imaginary.tree")
	text := subject.Sources[SOURCE_TEXT]
	if text[len(text)-1] != 'i' {
		return false
	}
	for index := range len(text) {
		if text[index] == '.' {
			return false
		}
		if text[index] == 'e' {
			return false
		}
	}
	return true
}

// Names the byte one imaginary whole number opens its digits at, which stands behind every zero
// and every underscore the author wrote ahead of them.
func number_head(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "number_head.subject")
	ast.Parse_State_Invariants(tree, "number_head.tree")
	text := subject.Sources[SOURCE_TEXT]
	head := 0
	for head < len(text)-1 {
		if text[head] != '0' {
			if text[head] != '_' {
				break
			}
		}
		head = head + 1
	}
	// A literal of zeros alone states the number zero, which is one digit and never none.
	if head == len(text)-1 {
		emit_byte(subject, '0')
	}
	subject.Counts[COUNT_DIGIT] = Count(head)
}

// Writes the digits of one number literal from the byte the caller names, with each mark it holds
// written in the case the canonical form states. A mark of zero names a literal that holds none.
func emit_digits(subject *Printer, tree *ast.Parse_State, exponent Exponent) {
	Printer_Invariants(subject, "emit_digits.subject")
	ast.Parse_State_Invariants(tree, "emit_digits.tree")
	Exponent_Invariants(exponent, "emit_digits.exponent")
	text := subject.Sources[SOURCE_TEXT]
	for index := int(subject.Counts[COUNT_DIGIT]); index < len(text); index++ {
		value := Symbol(text[index])
		if exponent != EXPONENT_NONE {
			if value == Symbol(exponent) {
				// The canonical form states an exponent mark in lower case, and the
				// letters stand thirty two apart in the alphabet the source spells.
				value = value + 32
			}
		}
		emit_byte(subject, value)
	}
}

// Writes the empty line one declaration owes the declarations ahead of it. A declaration of
// another kind, and every note that opens one, stand behind an empty line, thus a reader sees
// where one run of declarations closes and the next one opens.
func open_declaration(subject *Printer, tree *ast.Parse_State, commented Boolean) {
	Printer_Invariants(subject, "open_declaration.subject")
	ast.Parse_State_Invariants(tree, "open_declaration.tree")
	Boolean_Invariants(commented, "open_declaration.commented")
	kind := node_kind(subject, tree)
	if kind == ast.NODE_BLANK {
		return
	}
	// A note opens the run its declaration stands in, thus the note stands behind the empty
	// line and everything behind that note stands inside the run it opened.
	if bool(commented) {
		return
	}
	if kind != ast.NODE_COMMENT {
		if kind == subject.Kinds[KIND_DECLARATION] {
			return
		}
	}
	if bool(stands_after_blank(subject)) {
		return
	}
	emit_line(subject)
}

// Reads whether the run of declarations the print just wrote spans more than one line, and opens
// the empty line that stands between it and the run behind it where both do. A declaration states
// the lines it spans only once the print wrote it, thus the line stands where the run opened
// rather than where the print now stands.
func open_spread(subject *Printer, held Boolean) (spread Boolean) {
	defer func() { Boolean_Invariants(spread, "open_spread.spread") }()
	Printer_Invariants(subject, "open_spread.subject")
	Boolean_Invariants(held, "open_spread.held")
	place := int(subject.Counts[COUNT_DECLARED])
	written := int(subject.Counts[COUNT_WRITTEN])
	form := subject.Forms[FORM_SLOT]
	if written > len(form) {
		written = len(form)
	}
	lines := 0
	for step := place; step < written; step = step + 1 {
		if form[step] == '\n' {
			lines = lines + 1
		}
	}
	if lines < 2 {
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
	return true
}

// Opens one empty line at the byte the run of declarations the print wrote stands at, moving
// everything behind it one byte on.
func open_line_at(subject *Printer) {
	Printer_Invariants(subject, "open_line_at.subject")
	place := int(subject.Counts[COUNT_DECLARED])
	written := int(subject.Counts[COUNT_WRITTEN])
	form := subject.Forms[FORM_SLOT]
	if written >= len(form) {
		subject.Flags[FLAG_FULL] = true
		return
	}
	copy(form[place+1:written+1], form[place:written])
	form[place] = '\n'
	subject.Counts[COUNT_WRITTEN] = Count(written + 1)
}

// Reports whether the print stands behind an empty line, which is the line a run of declarations
// opens at.
func stands_after_blank(subject *Printer) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "stands_after_blank.yes") }()
	Printer_Invariants(subject, "stands_after_blank.subject")
	written := int(subject.Counts[COUNT_WRITTEN])
	if written < 2 {
		return false
	}
	form := subject.Forms[FORM_SLOT]
	if form[written-1] != '\n' {
		return false
	}
	return Boolean(form[written-2] == '\n')
}

// Reports whether one number literal states the eight base by the zero it opens with, which the
// canonical form writes as a base of its own. The number zero states no base, thus a literal of
// one byte states none either.
func states_octal(subject *Printer, tree *ast.Parse_State) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "states_octal.yes") }()
	Printer_Invariants(subject, "states_octal.subject")
	ast.Parse_State_Invariants(tree, "states_octal.tree")
	text := subject.Sources[SOURCE_TEXT]
	if len(text) < 2 {
		return false
	}
	if text[0] != '0' {
		return false
	}
	if text[len(text)-1] == 'i' {
		return false
	}
	for index := range len(text) {
		if text[index] == '.' {
			return false
		}
	}
	return true
}

// Writes one note in the canonical form, which opens a space behind the slashes so the words read
// as words. A directive states a name a tool reads and no space stands inside it, thus a
// directive stands as the author wrote it.
func emit_note(subject *Printer) {
	Printer_Invariants(subject, "emit_note.subject")
	if bool(wraps_note(subject)) {
		emit_wrapped_note(subject)
		return
	}
	// A note the print writes as it stands states a form of its own: a directive names a
	// tool, a block comment holds text between its own marks, and a note that opens on spaces
	// lays out lines the author counted.
	text := subject.Sources[SOURCE_TEXT]
	for index := range len(text) {
		emit_byte(subject, Symbol(text[index]))
	}
}

// Reports whether the note the print stands on wraps at the widest line. A note stating prose on a
// line of its own wraps, because the words it holds read the same on any line. A directive, a note
// standing behind code, a block comment, and a note whose text opens on spaces of its own each
// state a form the author laid out, thus the print writes them as they stand.
func wraps_note(subject *Printer) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "wraps_note.yes") }()
	Printer_Invariants(subject, "wraps_note.subject")
	text := subject.Sources[SOURCE_TEXT]
	if len(text) < 3 {
		return false
	}
	if text[1] != '/' {
		return false
	}
	if bool(states_directive(subject)) {
		return false
	}
	if text[2] == '\t' {
		return false
	}
	// A note whose text opens on a run of spaces states a layout of its own, thus the print
	// keeps every line of it as the author wrote it.
	if len(text) > 3 {
		if text[2] == ' ' {
			if text[3] == ' ' {
				return false
			}
		}
	}
	take_line_width(subject)
	held := int(subject.Counts[COUNT_INDENT]) * TAB_WIDTH
	return Boolean(int(subject.Spans[SPAN_LINE]) == held)
}

// Writes one note, opening a line of its own wherever the words it holds would run past the widest
// line. The print opens every line with the tabs the note opened at and the slashes a note states,
// thus a note of any length reads as one note.
func emit_wrapped_note(subject *Printer) {
	Printer_Invariants(subject, "emit_wrapped_note.subject")
	text := subject.Sources[SOURCE_TEXT]
	head := int(subject.Spans[SPAN_LINE]) + len(NOTE_HEAD)
	emit_note_head(subject)
	column := head
	place := 2
	for place < len(text) {
		if text[place] == ' ' {
			place = place + 1
			continue
		}
		end := place
		width := 0
		for end < len(text) {
			if text[end] == ' ' {
				break
			}
			if text[end]&CHARACTER_TAIL_MASK != CHARACTER_TAIL_MARK {
				width = width + 1
			}
			end = end + 1
		}
		if column > head {
			if column+width+1 > COLUMN_WIDTH_MAXIMUM {
				emit_line(subject)
				emit_indent(subject)
				emit_note_head(subject)
				column = head
			}
		}
		if column > head {
			emit_byte(subject, ' ')
			column = column + 1
		}
		for index := place; index < end; index++ {
			emit_byte(subject, Symbol(text[index]))
		}
		column = column + width
		place = end
	}
}

// NOTE_HEAD is the text one note opens with, the slashes and the space behind them.
const NOTE_HEAD = "// "

// Writes the slashes one note opens with and the space behind them.
func emit_note_head(subject *Printer) {
	Printer_Invariants(subject, "emit_note_head.subject")
	for index := range len(NOTE_HEAD) {
		emit_byte(subject, Symbol(NOTE_HEAD[index]))
	}
}

// Reports whether one note states a directive a tool reads rather than a sentence a reader reads.
// A directive names a tool and a word behind a colon, or one of the names the toolchain reserved
// long before that shape settled.
func states_directive(subject *Printer) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "states_directive.yes") }()
	Printer_Invariants(subject, "states_directive.subject")
	if bool(names_tool(subject)) {
		return true
	}
	return reserves_name(subject)
}

// Reports whether one note names a tool and a word behind a colon, as go:generate does.
func names_tool(subject *Printer) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "names_tool.yes") }()
	Printer_Invariants(subject, "names_tool.subject")
	text := subject.Sources[SOURCE_TEXT]
	place := 2
	for place < len(text) {
		// A tool names itself in lower case and holds a hyphen between its words, thus
		// the name closes at the first byte that states neither.
		letter := text[place] >= 'a'
		if text[place] > 'z' {
			letter = false
		}
		if text[place] == '-' {
			letter = true
		}
		if !letter {
			break
		}
		place = place + 1
	}
	if place == 2 {
		return false
	}
	if place >= len(text) {
		return false
	}
	if text[place] != ':' {
		return false
	}
	if place+1 >= len(text) {
		return false
	}
	if text[place+1] < 'a' {
		return false
	}
	return Boolean(text[place+1] <= 'z')
}

// Reports whether one note opens with a name the toolchain reserved for a directive of its own.
func reserves_name(subject *Printer) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "reserves_name.yes") }()
	Printer_Invariants(subject, "reserves_name.subject")
	subject.Directives[WORD_DIRECTIVE] = "line "
	if bool(opens_word(subject)) {
		return true
	}
	subject.Directives[WORD_DIRECTIVE] = "export "
	if bool(opens_word(subject)) {
		return true
	}
	subject.Directives[WORD_DIRECTIVE] = "extern "
	if bool(opens_word(subject)) {
		return true
	}
	subject.Directives[WORD_DIRECTIVE] = "sysnb "
	if bool(opens_word(subject)) {
		return true
	}
	subject.Directives[WORD_DIRECTIVE] = "sys "
	if bool(opens_word(subject)) {
		return true
	}
	subject.Directives[WORD_DIRECTIVE] = "nolint"
	return opens_word(subject)
}

// Reports whether one note opens with the word the slot names, behind the slashes.
func opens_word(subject *Printer) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "opens_word.yes") }()
	Printer_Invariants(subject, "opens_word.subject")
	text := subject.Sources[SOURCE_TEXT]
	word := subject.Directives[WORD_DIRECTIVE]
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
func hold_tight(subject *Printer, tight Boolean) (held Boolean) {
	defer func() { Boolean_Invariants(held, "hold_tight.held") }()
	Printer_Invariants(subject, "hold_tight.subject")
	Boolean_Invariants(tight, "hold_tight.tight")
	held = subject.Flags[FLAG_TIGHT]
	subject.Flags[FLAG_TIGHT] = tight
	return held
}

// Marks the run of one operation the print breaks at every sign, and hands back the mark that
// stood before it, which the caller stands back.
func hold_split(subject *Printer, split Boolean) (held Boolean) {
	defer func() { Boolean_Invariants(held, "hold_split.held") }()
	Printer_Invariants(subject, "hold_split.subject")
	Boolean_Invariants(split, "hold_split.split")
	held = subject.Flags[FLAG_SPLIT]
	subject.Flags[FLAG_SPLIT] = split
	return held
}

// Stands the split mark back where a form found it.
func restore_split(subject *Printer, held Boolean) {
	Printer_Invariants(subject, "restore_split.subject")
	Boolean_Invariants(held, "restore_split.held")
	subject.Flags[FLAG_SPLIT] = held
}

// Stands the tight flag back where a form found it.
func restore_tight(subject *Printer, held Boolean) {
	Printer_Invariants(subject, "restore_tight.subject")
	Boolean_Invariants(held, "restore_tight.held")
	subject.Flags[FLAG_TIGHT] = held
}

// Reports whether the node the walk stands on states an empty line standing against a brace,
// which is a line the canonical form drops: the body opens at its first statement and closes at
// its last one.
func stands_against(subject *Printer, tree *ast.Parse_State) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "stands_against.yes") }()
	Printer_Invariants(subject, "stands_against.subject")
	ast.Parse_State_Invariants(tree, "stands_against.tree")
	if node_kind(subject, tree) != ast.NODE_BLANK {
		return false
	}
	// Only a body that closes up against its brackets drops the line: every other body keeps
	// the empty lines the author wrote wherever the author wrote them.
	if !bool(subject.Flags[FLAG_TIGHT]) {
		return false
	}
	position := Position(ast.Node_At(tree, subject.Nodes[subject.Counts[COUNT_DEPTH]]).Token)
	// The empty line names the token behind it, thus a closing bracket there states a line
	// against the close and an opening bracket ahead of it states a line against the open. A
	// semicolon the line feed itself stands for spans no byte and states no bracket either
	// way, thus the read steps over it.
	subject.Positions[POSITION_FOUND] = position
	ahead_token(subject, tree)
	found := ast.Token_Index(subject.Positions[POSITION_FOUND])
	subject.Signs[SIGN_HELD] = ast.Token_At(tree, found).Kind
	if bool(closes_bracket(subject)) {
		return true
	}
	if position == POSITION_MINIMUM {
		return false
	}
	subject.Positions[POSITION_FOUND] = position
	behind_token(subject, tree)
	found = ast.Token_Index(subject.Positions[POSITION_FOUND])
	subject.Signs[SIGN_HELD] = ast.Token_At(tree, found).Kind
	return opens_bracket(subject)
}

// Names the first token at or behind one position that the source itself wrote, stepping over
// every semicolon a line feed stands for.
func ahead_token(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "ahead_token.subject")
	ast.Parse_State_Invariants(tree, "ahead_token.tree")
	for subject.Positions[POSITION_FOUND] < POSITION_MAXIMUM {
		one := ast.Token_At(tree, ast.Token_Index(subject.Positions[POSITION_FOUND]))
		if one.Size != 0 {
			return
		}
		if one.Kind != token.KIND_SEMICOLON {
			return
		}
		subject.Positions[POSITION_FOUND] = subject.Positions[POSITION_FOUND] + 1
	}
}

// Names the first token ahead of one position that the source itself wrote, stepping back over
// every semicolon a line feed stands for.
func behind_token(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "behind_token.subject")
	ast.Parse_State_Invariants(tree, "behind_token.tree")
	subject.Positions[POSITION_FOUND] = subject.Positions[POSITION_FOUND] - 1
	for subject.Positions[POSITION_FOUND] > POSITION_MINIMUM {
		one := ast.Token_At(tree, ast.Token_Index(subject.Positions[POSITION_FOUND]))
		if one.Size != 0 {
			return
		}
		if one.Kind != token.KIND_SEMICOLON {
			return
		}
		subject.Positions[POSITION_FOUND] = subject.Positions[POSITION_FOUND] - 1
	}
}

// Marks the empty line a field list of no fields closes on. The line the author left ahead of the
// next declaration stands inside the list the parser read, thus a list that writes no body of its
// own hands that line to the print that follows it.
func close_empty_list(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "close_empty_list.subject")
	ast.Parse_State_Invariants(tree, "close_empty_list.tree")
	if !bool(descend(subject, tree)) {
		return
	}
	open := Boolean(true)
	for bool(open) {
		if bool(closes_block(subject, tree)) {
			subject.Flags[FLAG_BLANK] = true
		}
		open = advance(subject, tree)
	}
	ascend(subject)
}

// Reports whether the body the walk stands on holds no part at all, counting the empty lines the
// author left inside it as no part. A field list of no fields states one form the canonical form
// closes on the line it opened on.
func holds_no_part(subject *Printer, tree *ast.Parse_State) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "holds_no_part.yes") }()
	Printer_Invariants(subject, "holds_no_part.subject")
	ast.Parse_State_Invariants(tree, "holds_no_part.tree")
	held := subject.Nodes[subject.Counts[COUNT_DEPTH]]
	depth := subject.Counts[COUNT_DEPTH]
	empty := Boolean(true)
	open := descend(subject, tree)
	for bool(open) {
		if node_kind(subject, tree) != ast.NODE_BLANK {
			empty = false
		}
		open = advance(subject, tree)
	}
	subject.Counts[COUNT_DEPTH] = depth
	subject.Nodes[depth] = held
	return empty
}

// Reports whether the body the walk stands on holds one part alone, counting neither the empty
// lines nor the notes standing between the parts.
func holds_one_part(subject *Printer, tree *ast.Parse_State) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "holds_one_part.yes") }()
	Printer_Invariants(subject, "holds_one_part.subject")
	ast.Parse_State_Invariants(tree, "holds_one_part.tree")
	held := subject.Nodes[subject.Counts[COUNT_DEPTH]]
	depth := subject.Counts[COUNT_DEPTH]
	parts := 0
	open := descend(subject, tree)
	for bool(open) {
		if node_kind(subject, tree) != ast.NODE_BLANK {
			parts = parts + 1
		}
		open = advance(subject, tree)
	}
	subject.Counts[COUNT_DEPTH] = depth
	subject.Nodes[depth] = held
	return Boolean(parts == 1)
}
