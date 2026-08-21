// Package printer writes the canonical form of a parse tree. It exists because a linter that
// judges whether a file is clean needs the form the formatter would write, and reading that form
// out of the tree costs nothing the caller did not already pay for. Every store is fixed storage
// the caller owns, thus a print allocates nothing.
package printer

import (
	"local/james-orcales/shared/go/ast"
	"local/james-orcales/shared/go/token"
	"local/james-orcales/shared/invariant/default"
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

// SPAN_FOUND holds the width one measure answered with.
const SPAN_FOUND = 0

// SPAN_COLUMN holds the column one run of lines aligns its notes at.
const SPAN_COLUMN = 1

// SPAN_SLOT_COUNT is the width slot count one printer holds.
const SPAN_SLOT_COUNT = 2

// POSITION_FOUND holds the run position one search answered with.
const POSITION_FOUND = 0

// POSITION_FROM holds the run position one scan opens at.
const POSITION_FROM = 1

// POSITION_TO holds the run position one scan closes at.
const POSITION_TO = 2

// POSITION_SLOT_COUNT is the run position slot count one printer holds.
const POSITION_SLOT_COUNT = 3

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

// KIND_SLOT_COUNT is the node class slot count one printer holds.
const KIND_SLOT_COUNT = 1

// MARK_SIGN holds the token of the sign one operation states between its values.
const MARK_SIGN = 0

// MARK_SLOT_COUNT is the token slot count one printer holds.
const MARK_SLOT_COUNT = 1

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

// COUNT_SLOT_COUNT is the counter count one printer holds.
const COUNT_SLOT_COUNT = 17

// FORM_SLOT holds the storage the print writes into.
const FORM_SLOT = 0

// FORM_SLOT_COUNT is the storage slot count one printer holds.
const FORM_SLOT_COUNT = 1

// SOURCE_SLOT holds the source the tree stands for.
const SOURCE_SLOT = 0

// SOURCE_SLOT_COUNT is the source slot count one printer holds.
const SOURCE_SLOT_COUNT = 1

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

// FLAG_COUNT is the flag count one printer holds.
const FLAG_COUNT = 5

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
func Symbol_Invariants(value Symbol, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint8(uint8(value), SYMBOL_MINIMUM, SYMBOL_MAXIMUM).
		Ensure()
}

// Boolean is a true or false report about one print.
type Boolean bool

// Boolean_Invariants states both print reports as obligations.
func Boolean_Invariants(value Boolean, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Sometimes(bool(value), "The print report is true.").
		Ensure()
}

// Form is caller storage one print writes itself into.
type Form []byte

// Form_Invariants states the storage the widest form needs.
func Form_Invariants(value Form, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), FORM_SIZE_MINIMUM, FORM_SIZE_MAXIMUM).
		Ensure()
}

// Form_Count is the byte count one print wrote.
type Form_Count int

// Form_Count_Invariants states the byte count of the widest form.
func Form_Count_Invariants(value Form_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), FORM_SIZE_MINIMUM, FORM_SIZE_MAXIMUM).
		Ensure()
}

// Count is one counter of the print. Every counter lives in a slot, thus no body owes the whole
// counter domain at a step that can only see one part of it.
type Count int

// Count_Invariants states the widest thing one counter counts.
func Count_Invariants(value Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), FORM_SIZE_MINIMUM, COUNT_MAXIMUM).
		Ensure()
}

// Width is the byte count one name spends, which is what an alignment run pads against.
type Width int32

// Width_Invariants states every width a name spends.
func Width_Invariants(value Width, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
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
	// Marks holds the run positions of the tokens the print writes of its own.
	Marks [MARK_SLOT_COUNT]ast.Token_Index
	// Signs holds the classes of the tokens one reader of the run stands between.
	Signs [SIGN_SLOT_COUNT]token.Kind
	// Words holds the keyword one form opens with, which the tree drops.
	Words [WORD_SLOT_COUNT]Word
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
func Printer_Invariants(subject *Printer, namespace invariant.Namespace) {
	invariant.Always(
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
	for range int(subject.Counts[COUNT_INDENT]) {
		emit_byte(subject, '\t')
	}
}

// Writes the source bytes the token of the node the walk stands on spans.
func emit_token(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "emit_token.subject")
	ast.Parse_State_Invariants(tree, "emit_token.tree")
	node := ast.Node_At(tree, subject.Nodes[subject.Counts[COUNT_DEPTH]])
	text := token.Text(subject.Sources[SOURCE_SLOT], ast.Token_At(tree, node.Token))
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
	one := ast.Token_At(tree, ast.Token_Index(position))
	text := token.Text(subject.Sources[SOURCE_SLOT], one)
	for index := range len(text) {
		emit_byte(subject, Symbol(text[index]))
	}
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
	if !bool(descend(subject, tree)) {
		return
	}
	// A comment ahead of the package clause is the doc of the file, and no empty line stands
	// between the two, thus the print holds back the blank the scanner counted there. Every
	// declaration behind the clause states its own trivia, thus this walk takes plain steps.
	open := Boolean(true)
	named := Boolean(false)
	for bool(open) {
		if !bool(named) {
			named = print_opening(subject, tree)
			open = advance(subject, tree)
			continue
		}
		print_declaration(subject, tree)
		open = advance(subject, tree)
	}
	ascend(subject)
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
func Word_Invariants(value Word, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
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
		emit_token(subject, tree)
		emit_space(subject)
		advance_part(subject, tree)
	}
	emit_token(subject, tree)
	close_line(subject, tree)
	drain_trivia(subject, tree, advance(subject, tree))
	ascend(subject)
}

// Writes one constant or variable declaration: its names, the type they wear, and the values
// behind them.
func print_values(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "print_values.subject")
	ast.Parse_State_Invariants(tree, "print_values.tree")
	emit_indent(subject)
	emit_word(subject, subject.Words[WORD_SLOT])
	emit_space(subject)
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
	close_line(subject, tree)
	drain_trivia(subject, tree, open)
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
			close_line(subject, tree)
			stated = true
			open = advance(subject, tree)
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
	stand := open
	stand = print_signature_parameters(subject, tree, stand)
	stand = print_signature_results(subject, tree, stand)
	for bool(stand) {
		if node_kind(subject, tree) != ast.NODE_BLOCK {
			break
		}
		emit_space(subject)
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
	subject *Printer, tree *ast.Parse_State, open Boolean,
) (stand Boolean) {
	defer func() { Boolean_Invariants(stand, "print_signature_parameters.stand") }()
	Printer_Invariants(subject, "print_signature_parameters.subject")
	ast.Parse_State_Invariants(tree, "print_signature_parameters.tree")
	Boolean_Invariants(open, "print_signature_parameters.open")
	stand = print_type_parameters(subject, tree, open)
	emit_byte(subject, '(')
	arguments := 0
	broken := Boolean(false)
	opened := Boolean(false)
	tail := Boolean(false)
	indent := subject.Counts[COUNT_INDENT]
	for bool(stand) {
		if node_kind(subject, tree) != ast.NODE_PARAMETER {
			break
		}
		here := breaks_before(subject, tree)
		if arguments == 0 {
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
		close_broken(subject, tree, tail)
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
	bare := bare_result(subject, tree, open)
	results := 0
	broken := Boolean(false)
	opened := Boolean(false)
	tail := Boolean(false)
	indent := subject.Counts[COUNT_INDENT]
	for bool(stand) {
		if node_kind(subject, tree) != ast.NODE_RESULT {
			break
		}
		here := breaks_before(subject, tree)
		if results == 0 {
			emit_space(subject)
			if !bool(bare) {
				emit_byte(subject, '(')
			}
			indent = subject.Counts[COUNT_INDENT]
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
		print_parameter(subject, tree)
		results = results + 1
		stand = advance_part(subject, tree)
	}
	if bool(broken) {
		subject.Counts[COUNT_COLUMN_INDENT] = indent
		close_broken(subject, tree, tail)
	}
	if results > 0 {
		if !bool(bare) {
			emit_byte(subject, ')')
		}
	}
	return stand
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
	emit_indent(subject)
	emit_byte(subject, '}')
	if bool(trail) {
		subject.Flags[FLAG_BLANK] = true
	}
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
	switch node_kind(subject, tree) {
	case ast.NODE_BLANK:
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
		if bool(right) {
			emit_space(subject)
			emit_sign(subject, tree)
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
			behind = true
		}
		here := Boolean(false)
		if bool(behind) {
			here = breaks_before(subject, tree)
		}
		open_continuation(subject, tree, here)
		if bool(behind) {
			if !bool(here) {
				emit_space(subject)
			}
		}
		subject.Counts[COUNT_NEST] = nest
		print_expression(subject, tree)
		values = values + 1
		open = advance_part(subject, tree)
	}
	finish_parts(subject, tree)
	ascend(subject)
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
func Position_Invariants(value Position, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
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
		print_case_values(subject, tree)
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
func Wrap_Invariants(value Wrap, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_Uint8(uint8(value), uint8(WRAP_CALL), uint8(WRAP_INDEX)).
		Ensure()
}

// Writes one form that wraps its parts in brackets: a call, an index, or a grouping.
func print_wrapped(subject *Printer, tree *ast.Parse_State, wrap Wrap) {
	Printer_Invariants(subject, "print_wrapped.subject")
	ast.Parse_State_Invariants(tree, "print_wrapped.tree")
	Wrap_Invariants(wrap, "print_wrapped.wrap")
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
			opened = breaks_before(subject, tree)
			print_first_break(subject, tree, opened)
			broken = opened || tail
		}
		if parts > 1 {
			// Each part stands where the author put it: on the line the part before
			// it holds, or on a line of its own. A part that indented lines of its
			// own leaves them behind, thus each part opens at the column of the form.
			here := breaks_before(subject, tree)
			if bool(open_column(here, opened)) {
				subject.Counts[COUNT_INDENT] = indent + 1
				subject.Flags[FLAG_BROKEN] = marked
			}
			opened = opened || here
			broken = broken || here
			print_element_break(subject, tree, here)
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
		close_broken(subject, tree, tail)
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
		here := breaks_before(subject, tree)
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
	index := subject.Marks[MARK_SIGN]
	text := token.Text(subject.Sources[SOURCE_SLOT], ast.Token_At(tree, index))
	for position := range len(text) {
		emit_byte(subject, Symbol(text[position]))
	}
}

// LEVEL_MINIMUM names a token that binds no values, which is a token that is no operation sign.
const LEVEL_MINIMUM = 0

// LEVEL_SIGN_MAXIMUM names the tightest binding a sign of an operation states.
const LEVEL_SIGN_MAXIMUM = 5

// LEVEL_MAXIMUM is the tightest binding a sign of an operation states.
const LEVEL_MAXIMUM = LEVEL_SIGN_MAXIMUM

// CUT_MINIMUM is the loosest binding a form closes its signs up at.
const CUT_MINIMUM = 4

// CUT_MIDDLE closes up the signs that bind tightest and spaces every looser one, which is what a
// form that mixes the two levels the canonical form spaces by states.
const CUT_MIDDLE = 5

// CUT_MAXIMUM stands above every sign, which is the cut a form that spaces every sign it holds
// states, and no sign binds that tightly.
const CUT_MAXIMUM = 6

// CLASH_NONE names an operation no reading of which runs two signs together.
const CLASH_NONE = 0

// CLASH_LOOSE names a clash the looser of the two spaced levels settles.
const CLASH_LOOSE = 4

// CLASH_TIGHT names a clash only the tighter of the two spaced levels settles.
const CLASH_TIGHT = 5

// Clash names the level a sign must space at to stand apart from the sign of the value behind it.
type Clash uint8

// Clash_Invariants states every clash two signs standing together state.
func Clash_Invariants(value Clash, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_3_Uint8(uint8(value), CLASH_NONE, CLASH_LOOSE, CLASH_TIGHT).
		Ensure()
}

// Cut names the binding at and above which one form closes up the signs it holds.
type Cut uint8

// Cut_Invariants states every cut one form closes its signs up at.
func Cut_Invariants(value Cut, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_3_Uint8(uint8(value), CUT_MINIMUM, CUT_MIDDLE, CUT_MAXIMUM).
		Ensure()
}

// Level names how tightly one sign binds the values it stands between.
type Level uint8

// Level_Invariants states every binding one sign of an operation holds.
func Level_Invariants(value Level, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint8(uint8(value), LEVEL_MINIMUM, LEVEL_MAXIMUM).
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
		return 5
	case token.KIND_PLUS, token.KIND_MINUS, token.KIND_OR, token.KIND_EXCLUSIVE_OR:
		return 4
	case token.KIND_EQUAL, token.KIND_NOT_EQUAL, token.KIND_LESS, token.KIND_LESS_EQUAL,
		token.KIND_GREATER, token.KIND_GREATER_EQUAL:
		return 3
	case token.KIND_LOGICAL_AND:
		return 2
	case token.KIND_LOGICAL_OR:
		return 1
	}
	return 0
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
	four, five, clash := binary_levels(subject, tree)
	// A sign that would read as another sign against the value behind it takes spaces
	// however deep it stands, thus the clash sets the cut on its own.
	cut := Cut(CUT_MINIMUM)
	if clash > 0 {
		cut = Cut(clash) + 1
	}
	if clash == 0 {
		cut = plain_cut(subject, four, five)
	}
	binary_kind(subject, tree)
	return Boolean(Cut(precedence(subject)) < cut)
}

// Names the level at and above which the signs of one operation close up. An operation that mixes
// the two levels the canonical form spaces by states the tighter of them, and one that holds a
// single level spaces every sign a statement states on its own.
func plain_cut(subject *Printer, four Boolean, five Boolean) (cut Cut) {
	defer func() { Cut_Invariants(cut, "plain_cut.cut") }()
	Printer_Invariants(subject, "plain_cut.subject")
	Boolean_Invariants(four, "plain_cut.four")
	Boolean_Invariants(five, "plain_cut.five")
	plain := subject.Counts[COUNT_NEST] == 1
	if bool(four) {
		if bool(five) {
			if plain {
				return 5
			}
			return 4
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
			return 5
		}
	}
	if operation == token.KIND_AND {
		if unary == token.KIND_AND {
			return 5
		}
		if unary == token.KIND_EXCLUSIVE_OR {
			return 5
		}
	}
	if operation == token.KIND_PLUS {
		if unary == token.KIND_PLUS {
			return 4
		}
	}
	if operation == token.KIND_MINUS {
		if unary == token.KIND_MINUS {
			return 4
		}
	}
	return 0
}

// Reports which of the two levels the canonical form spaces by stand inside the operation the
// walk stands on. A value the source parenthesised states its own form, thus the walk stops at
// it and never reads the levels it holds.
func binary_levels(
	subject *Printer, tree *ast.Parse_State,
) (four Boolean, five Boolean, clash Clash) {
	defer func() {
		Boolean_Invariants(four, "binary_levels.four")
		Boolean_Invariants(five, "binary_levels.five")
		Clash_Invariants(clash, "binary_levels.clash")
	}()
	Printer_Invariants(subject, "binary_levels.subject")
	ast.Parse_State_Invariants(tree, "binary_levels.tree")
	binary_kind(subject, tree)
	operation := subject.Signs[SIGN_HELD]
	level := precedence(subject)
	four = Boolean(level == 4)
	five = Boolean(level == 5)
	if !bool(descend_quiet(subject, tree)) {
		return four, five, clash
	}
	if bool(reads_levels(subject, tree, level, false)) {
		left, right, held := binary_levels(subject, tree)
		four = four || left
		five = five || right
		clash = max(clash, held)
	}
	if bool(advance_quiet(subject, tree)) {
		if bool(reads_levels(subject, tree, level, true)) {
			left, right, held := binary_levels(subject, tree)
			four = four || left
			five = five || right
			clash = max(clash, held)
		}
		if node_kind(subject, tree) == ast.NODE_UNARY {
			binary_kind(subject, tree)
			subject.Signs[SIGN_BEHIND] = operation
			clash = max(clash, clash_level(subject))
		}
	}
	ascend(subject)
	return four, five, clash
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
	print_expression(subject, tree)
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
	tail := closes_line(subject, tree)
	indent := subject.Counts[COUNT_INDENT]
	marked := subject.Flags[FLAG_BROKEN]
	// A literal that stands as an element of another one states no type of its own, thus the
	// print reads whether a type stands ahead of the brace rather than assuming one does.
	brace := Position(ast.Node_At(tree, subject.Nodes[subject.Counts[COUNT_DEPTH]]).Token)
	open := descend_part(subject, tree)
	entered := open
	head := 0
	subject.Positions[POSITION_FROM] = brace
	if bool(typed_head(subject, tree, open)) {
		head = 1
	}
	parts := 0
	broken := Boolean(false)
	opened := Boolean(false)
	for bool(open) {
		if parts == head {
			emit_byte(subject, '{')
			indent = subject.Counts[COUNT_INDENT]
			marked = subject.Flags[FLAG_BROKEN]
			opened = breaks_before(subject, tree)
			print_first_break(subject, tree, opened)
			if bool(opened) {
				take_key_width(subject, tree)
			}
			broken = opened || tail
		}
		if parts > head {
			here := breaks_before(subject, tree)
			if bool(open_column(here, opened)) {
				subject.Counts[COUNT_INDENT] = indent + 1
				subject.Flags[FLAG_BROKEN] = marked
			}
			opened = opened || here
			broken = broken || here
			print_element_break(subject, tree, here)
			open_key_run(subject, tree, here)
		}
		subject.Counts[COUNT_NEST] = 1
		spans := spans_form(subject, tree)
		print_part(subject, tree, opened, true)
		subject.Counts[COUNT_NEST] = nest
		close_key_run(subject, tree, spans)
		parts = parts + 1
		open = advance_part(subject, tree)
	}
	if bool(broken) {
		subject.Counts[COUNT_COLUMN_INDENT] = indent
		close_broken(subject, tree, tail)
	}
	if bool(entered) {
		finish_parts(subject, tree)
		ascend(subject)
	}
	if parts <= head {
		emit_byte(subject, '{')
	}
	emit_byte(subject, '}')
	subject.Counts[COUNT_WIDTH] = width
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
		// the author stopped opening a line for each pair.
		if bool(open) {
			if !bool(breaks_before(subject, tree)) {
				break
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
	key_width(subject, tree)
	width := subject.Spans[SPAN_FOUND]
	held := subject.Counts[COUNT_WIDTH]
	if !bool(descend_part(subject, tree)) {
		return
	}
	print_expression(subject, tree)
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
func Prefix_Invariants(value Prefix, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
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
	if !bool(descend(subject, tree)) {
		emit_byte(subject, '{')
		emit_byte(subject, '}')
		return
	}
	emit_space(subject)
	emit_byte(subject, '{')
	close_line(subject, tree)
	subject.Counts[COUNT_INDENT] = subject.Counts[COUNT_INDENT] + 1
	subject.Counts[COUNT_WIDTH] = 0
	open := Boolean(true)
	for bool(open) {
		print_field(subject, tree)
		open = advance(subject, tree)
	}
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
	if !bool(descend(subject, tree)) {
		emit_byte(subject, '{')
		emit_byte(subject, '}')
		return
	}
	emit_space(subject)
	emit_byte(subject, '{')
	close_line(subject, tree)
	subject.Counts[COUNT_INDENT] = subject.Counts[COUNT_INDENT] + 1
	open := Boolean(true)
	for bool(open) {
		print_element(subject, tree)
		open = advance(subject, tree)
	}
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
	for offset := int(node.Token); offset <= ast.TOKEN_INDEX_MAXIMUM; offset++ {
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

// Writes the values one case stands for. The tree hangs the whole run off one node, thus the
// print walks that node rather than the case itself.
func print_case_values(subject *Printer, tree *ast.Parse_State) {
	Printer_Invariants(subject, "print_case_values.subject")
	ast.Parse_State_Invariants(tree, "print_case_values.tree")
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
