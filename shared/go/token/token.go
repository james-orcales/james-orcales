// Package token scans Go source text into tokens. It holds the lexical layer only: the caller
// owns the source bytes and the cursor, thus a scan allocates nothing. The grammar it accepts is
// the subset this repository's linter permits, so an identifier is ASCII and nothing else.
package token

import (
	"local/james-orcales/shared/sim/aver/default"
)

// SOURCE_SIZE_MINIMUM admits the empty source, which scans to KIND_END_OF_FILE at once.
const SOURCE_SIZE_MINIMUM = 0

// SOURCE_SIZE_MAXIMUM is one mebibyte, three times the largest Go file in this repository, so a
// hostile source is refused before the scanner reads one byte.
const SOURCE_SIZE_MAXIMUM = 1048576

// TAIL_SIZE_MINIMUM excludes the exhausted source, because a tail always opens one token.
const TAIL_SIZE_MINIMUM = 1

// FRACTION_TAIL_SIZE_MINIMUM counts the period and the digit that make a leading fraction.
const FRACTION_TAIL_SIZE_MINIMUM = 2

// COMMENT_TAIL_SIZE_MINIMUM counts both bytes of the shortest comment opener.
const COMMENT_TAIL_SIZE_MINIMUM = 2

// LINE_COUNT_MAXIMUM caps the lines one index holds. A source of this many lines spends fewer
// than eight bytes on each of them, thus every file a reader writes fits and a generated wall of
// line feeds is refused rather than read past the array.
const LINE_COUNT_MAXIMUM = 131072

// LINE_COUNT_MINIMUM is the line count of no source at all.
const LINE_COUNT_MINIMUM = 0

// LINE_MINIMUM is the first line, because a diagnostic counts lines from one.
const LINE_MINIMUM = 1

// LINE_MAXIMUM is the final line one index holds.
const LINE_MAXIMUM = LINE_COUNT_MAXIMUM

// COLUMN_MINIMUM is the first column, because a diagnostic counts columns from one.
const COLUMN_MINIMUM = 1

// COLUMN_MAXIMUM is the column one byte past the widest line a source holds.
const COLUMN_MAXIMUM = SOURCE_SIZE_MAXIMUM + 1

// LINE_SLOT holds how many lines the index read.
const LINE_SLOT = 0

// LINE_SLOT_COUNT is the counter count one index holds.
const LINE_SLOT_COUNT = 1

// OFFSET_MINIMUM is the first byte of a source.
const OFFSET_MINIMUM = 0

// OFFSET_MAXIMUM is one past the final byte, the cursor position of an exhausted source.
const OFFSET_MAXIMUM = SOURCE_SIZE_MAXIMUM

// SIZE_MINIMUM is the byte count of an inserted semicolon, which no source byte spells.
const SIZE_MINIMUM = 0

// SIZE_MAXIMUM admits a source that is one token, such as one general comment.
const SIZE_MAXIMUM = SOURCE_SIZE_MAXIMUM

// TOKEN_SIZE_MINIMUM is the byte count of the shortest token that source bytes spell.
const TOKEN_SIZE_MINIMUM = 1

// TOKEN_SIZE_MAXIMUM admits a source that is one token.
const TOKEN_SIZE_MAXIMUM = SOURCE_SIZE_MAXIMUM

// COMMENT_SIZE_MINIMUM counts both bytes of the shortest comment, an empty line comment.
const COMMENT_SIZE_MINIMUM = 2

// COMMENT_SIZE_MAXIMUM admits a source that is one general comment.
const COMMENT_SIZE_MAXIMUM = SOURCE_SIZE_MAXIMUM

// BLANK_COUNT_MINIMUM is the count of a token that follows the line before it.
const BLANK_COUNT_MINIMUM = 0

// BLANK_COUNT_MAXIMUM saturates the count. A run past it is a run no reader distinguishes,
// and the count rides in a byte the token already pads out.
const BLANK_COUNT_MAXIMUM = 255

// EXPONENT_TEXT_SIZE is the one byte an exponent letter spans.
const EXPONENT_TEXT_SIZE = 1

// OPERATOR_SIZE_ONE is the byte count of a one-byte operator such as the plus sign.
const OPERATOR_SIZE_ONE = 1

// OPERATOR_SIZE_TWO is the byte count of a two-byte operator such as the define sign.
const OPERATOR_SIZE_TWO = 2

// OPERATOR_SIZE_THREE is the byte count of the longest Go operators.
const OPERATOR_SIZE_THREE = 3

// KIND_END_OF_FILE is the exhausted source. It is the zero kind, thus a fresh Scanner starts
// with no statement open and inserts no semicolon before its first token.
const KIND_END_OF_FILE Kind = 0

// KIND_ILLEGAL is a byte that opens no Go token.
const KIND_ILLEGAL Kind = 1

// KIND_IDENTIFIER is a name that is no keyword.
const KIND_IDENTIFIER Kind = 2

// KIND_BREAK is the break keyword.
const KIND_BREAK Kind = 3

// KIND_CASE is the case keyword.
const KIND_CASE Kind = 4

// KIND_CHANNEL is the chan keyword.
const KIND_CHANNEL Kind = 5

// KIND_CONSTANT is the const keyword.
const KIND_CONSTANT Kind = 6

// KIND_CONTINUE is the continue keyword.
const KIND_CONTINUE Kind = 7

// KIND_DEFAULT is the default keyword.
const KIND_DEFAULT Kind = 8

// KIND_DEFER is the defer keyword.
const KIND_DEFER Kind = 9

// KIND_ELSE is the else keyword.
const KIND_ELSE Kind = 10

// KIND_FALLTHROUGH is the fallthrough keyword.
const KIND_FALLTHROUGH Kind = 11

// KIND_FOR is the for keyword.
const KIND_FOR Kind = 12

// KIND_FUNCTION is the func keyword.
const KIND_FUNCTION Kind = 13

// KIND_GO is the go keyword.
const KIND_GO Kind = 14

// KIND_GOTO is the goto keyword.
const KIND_GOTO Kind = 15

// KIND_IF is the if keyword.
const KIND_IF Kind = 16

// KIND_IMPORT is the import keyword.
const KIND_IMPORT Kind = 17

// KIND_INTERFACE is the interface keyword.
const KIND_INTERFACE Kind = 18

// KIND_MAP is the map keyword.
const KIND_MAP Kind = 19

// KIND_PACKAGE is the package keyword.
const KIND_PACKAGE Kind = 20

// KIND_RANGE is the range keyword.
const KIND_RANGE Kind = 21

// KIND_RETURN is the return keyword.
const KIND_RETURN Kind = 22

// KIND_SELECT is the select keyword.
const KIND_SELECT Kind = 23

// KIND_STRUCTURE is the struct keyword.
const KIND_STRUCTURE Kind = 24

// KIND_SWITCH is the switch keyword.
const KIND_SWITCH Kind = 25

// KIND_TYPE is the type keyword.
const KIND_TYPE Kind = 26

// KIND_VARIABLE is the var keyword.
const KIND_VARIABLE Kind = 27

// KIND_COMMENT is a line comment or a general comment.
const KIND_COMMENT Kind = 28

// KIND_INTEGER is an integer literal.
const KIND_INTEGER Kind = 29

// KIND_FLOAT is a floating-point literal.
const KIND_FLOAT Kind = 30

// KIND_IMAGINARY is an imaginary literal.
const KIND_IMAGINARY Kind = 31

// KIND_CHARACTER is a character literal between apostrophes.
const KIND_CHARACTER Kind = 32

// KIND_STRING is an interpreted or raw string literal.
const KIND_STRING Kind = 33

// KIND_SHIFT_LEFT_ASSIGN is the left-shift assignment operator.
const KIND_SHIFT_LEFT_ASSIGN Kind = 34

// KIND_SHIFT_RIGHT_ASSIGN is the right-shift assignment operator.
const KIND_SHIFT_RIGHT_ASSIGN Kind = 35

// KIND_AND_NOT_ASSIGN is the and-not assignment operator.
const KIND_AND_NOT_ASSIGN Kind = 36

// KIND_ELLIPSIS is the ellipsis of a variadic parameter or an array capacity.
const KIND_ELLIPSIS Kind = 37

// KIND_PLUS_ASSIGN is the addition assignment operator.
const KIND_PLUS_ASSIGN Kind = 38

// KIND_MINUS_ASSIGN is the subtraction assignment operator.
const KIND_MINUS_ASSIGN Kind = 39

// KIND_STAR_ASSIGN is the multiplication assignment operator.
const KIND_STAR_ASSIGN Kind = 40

// KIND_SLASH_ASSIGN is the division assignment operator.
const KIND_SLASH_ASSIGN Kind = 41

// KIND_PERCENT_ASSIGN is the remainder assignment operator.
const KIND_PERCENT_ASSIGN Kind = 42

// KIND_AND_ASSIGN is the bitwise-and assignment operator.
const KIND_AND_ASSIGN Kind = 43

// KIND_OR_ASSIGN is the bitwise-or assignment operator.
const KIND_OR_ASSIGN Kind = 44

// KIND_EXCLUSIVE_OR_ASSIGN is the exclusive-or assignment operator.
const KIND_EXCLUSIVE_OR_ASSIGN Kind = 45

// KIND_SHIFT_LEFT is the left-shift operator.
const KIND_SHIFT_LEFT Kind = 46

// KIND_SHIFT_RIGHT is the right-shift operator.
const KIND_SHIFT_RIGHT Kind = 47

// KIND_AND_NOT is the and-not operator.
const KIND_AND_NOT Kind = 48

// KIND_LOGICAL_AND is the conditional-and operator.
const KIND_LOGICAL_AND Kind = 49

// KIND_LOGICAL_OR is the conditional-or operator.
const KIND_LOGICAL_OR Kind = 50

// KIND_ARROW is the channel direction and send-receive operator.
const KIND_ARROW Kind = 51

// KIND_INCREMENT is the increment statement operator.
const KIND_INCREMENT Kind = 52

// KIND_DECREMENT is the decrement statement operator.
const KIND_DECREMENT Kind = 53

// KIND_EQUAL is the equality operator.
const KIND_EQUAL Kind = 54

// KIND_NOT_EQUAL is the inequality operator.
const KIND_NOT_EQUAL Kind = 55

// KIND_LESS_EQUAL is the less-or-equal operator.
const KIND_LESS_EQUAL Kind = 56

// KIND_GREATER_EQUAL is the greater-or-equal operator.
const KIND_GREATER_EQUAL Kind = 57

// KIND_DEFINE is the short variable declaration operator.
const KIND_DEFINE Kind = 58

// KIND_PLUS is the addition operator.
const KIND_PLUS Kind = 59

// KIND_MINUS is the subtraction operator.
const KIND_MINUS Kind = 60

// KIND_STAR is the multiplication, pointer, and dereference operator.
const KIND_STAR Kind = 61

// KIND_SLASH is the division operator.
const KIND_SLASH Kind = 62

// KIND_PERCENT is the remainder operator.
const KIND_PERCENT Kind = 63

// KIND_AND is the bitwise-and and address operator.
const KIND_AND Kind = 64

// KIND_OR is the bitwise-or operator.
const KIND_OR Kind = 65

// KIND_EXCLUSIVE_OR is the exclusive-or and complement operator.
const KIND_EXCLUSIVE_OR Kind = 66

// KIND_LESS is the less-than operator.
const KIND_LESS Kind = 67

// KIND_GREATER is the greater-than operator.
const KIND_GREATER Kind = 68

// KIND_ASSIGN is the assignment operator.
const KIND_ASSIGN Kind = 69

// KIND_NOT is the negation operator.
const KIND_NOT Kind = 70

// KIND_PARENTHESIS_LEFT is the opening parenthesis.
const KIND_PARENTHESIS_LEFT Kind = 71

// KIND_PARENTHESIS_RIGHT is the closing parenthesis.
const KIND_PARENTHESIS_RIGHT Kind = 72

// KIND_BRACKET_LEFT is the opening square bracket.
const KIND_BRACKET_LEFT Kind = 73

// KIND_BRACKET_RIGHT is the closing square bracket.
const KIND_BRACKET_RIGHT Kind = 74

// KIND_BRACE_LEFT is the opening curly brace.
const KIND_BRACE_LEFT Kind = 75

// KIND_BRACE_RIGHT is the closing curly brace.
const KIND_BRACE_RIGHT Kind = 76

// KIND_COMMA is the comma separator.
const KIND_COMMA Kind = 77

// KIND_PERIOD is the selector and qualifier period.
const KIND_PERIOD Kind = 78

// KIND_SEMICOLON is a written or an inserted semicolon.
const KIND_SEMICOLON Kind = 79

// KIND_COLON is the label, case, and slice colon.
const KIND_COLON Kind = 80

// KIND_TILDE is the type-constraint approximation operator.
const KIND_TILDE Kind = 81

// KIND_MINIMUM is the exhausted source, the smallest kind.
const KIND_MINIMUM = uint8(KIND_END_OF_FILE)

// KIND_MAXIMUM is the final one-byte operator, the largest kind.
const KIND_MAXIMUM = uint8(KIND_TILDE)

// SCANNED_KIND_MINIMUM excludes KIND_END_OF_FILE, because source bytes spell a scanned kind.
const SCANNED_KIND_MINIMUM = int(KIND_ILLEGAL)

// SCANNED_KIND_MAXIMUM is the final one-byte operator.
const SCANNED_KIND_MAXIMUM = int(KIND_TILDE)

// WORD_KIND_MINIMUM is KIND_IDENTIFIER, which sits beside the keyword block so one range holds
// every kind that a letter run can produce.
const WORD_KIND_MINIMUM = int(KIND_IDENTIFIER)

// WORD_KIND_MAXIMUM is the final keyword.
const WORD_KIND_MAXIMUM = int(KIND_VARIABLE)

// KEYWORD_KIND_MINIMUM is the first keyword.
const KEYWORD_KIND_MINIMUM = int(KIND_BREAK)

// KEYWORD_KIND_MAXIMUM is the final keyword.
const KEYWORD_KIND_MAXIMUM = int(KIND_VARIABLE)

// OPERATOR_KIND_MINIMUM is the first three-byte operator.
const OPERATOR_KIND_MINIMUM = int(KIND_SHIFT_LEFT_ASSIGN)

// OPERATOR_KIND_MAXIMUM is the final one-byte operator.
const OPERATOR_KIND_MAXIMUM = int(KIND_TILDE)

// OPERATOR_KIND_2_MINIMUM is the first two-byte operator.
const OPERATOR_KIND_2_MINIMUM = int(KIND_PLUS_ASSIGN)

// OPERATOR_KIND_2_MAXIMUM is the final two-byte operator.
const OPERATOR_KIND_2_MAXIMUM = int(KIND_DEFINE)

// OPERATOR_KIND_1_MINIMUM is the first one-byte operator.
const OPERATOR_KIND_1_MINIMUM = int(KIND_PLUS)

// OPERATOR_KIND_1_MAXIMUM is the final one-byte operator.
const OPERATOR_KIND_1_MAXIMUM = int(KIND_TILDE)

// Kind is the lexical class of one token. It stands over a byte because the grammar holds
// eighty-two classes, and a token run of a whole file pays for this width once for each token.
type Kind uint8

// Kind_Invariants states the complete lexical class domain.
func Kind_Invariants(value Kind, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), KIND_MINIMUM, KIND_MAXIMUM).
		Ensure()
}

// Scanned_Kind is the class of a token that source bytes spell, thus never KIND_END_OF_FILE.
type Scanned_Kind int

// Scanned_Kind_Invariants excludes the kind that no source byte spells.
func Scanned_Kind_Invariants(value Scanned_Kind, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), SCANNED_KIND_MINIMUM, SCANNED_KIND_MAXIMUM).
		Ensure()
}

// Word_Kind is the class of one letter run: a keyword or KIND_IDENTIFIER.
type Word_Kind int

// Word_Kind_Invariants states the identifier and keyword block.
func Word_Kind_Invariants(value Word_Kind, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), WORD_KIND_MINIMUM, WORD_KIND_MAXIMUM).
		Ensure()
}

// Keyword_Kind is the class of one Go keyword.
type Keyword_Kind int

// Keyword_Kind_Invariants states the keyword block alone.
func Keyword_Kind_Invariants(value Keyword_Kind, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), KEYWORD_KIND_MINIMUM, KEYWORD_KIND_MAXIMUM).
		Ensure()
}

// Number_Kind is the class of one numeric literal.
type Number_Kind int

// Number_Kind_Invariants states the three numeric classes.
func Number_Kind_Invariants(value Number_Kind, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Int(int(value), int(KIND_INTEGER), int(KIND_FLOAT), int(KIND_IMAGINARY)).
		Ensure()
}

// Quoted_Kind is the class of one quoted literal, or KIND_ILLEGAL when no quote closed it.
type Quoted_Kind int

// Quoted_Kind_Invariants states the two closed literal classes and the unclosed one.
func Quoted_Kind_Invariants(value Quoted_Kind, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Int(int(value), int(KIND_ILLEGAL), int(KIND_CHARACTER), int(KIND_STRING)).
		Ensure()
}

// Operator_Kind is the class of one operator or delimiter.
type Operator_Kind int

// Operator_Kind_Invariants states the whole operator block.
func Operator_Kind_Invariants(value Operator_Kind, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), OPERATOR_KIND_MINIMUM, OPERATOR_KIND_MAXIMUM).
		Ensure()
}

// Operator_Kind_1 is the class of one one-byte operator.
type Operator_Kind_1 int

// Operator_Kind_1_Invariants states the one-byte operator block.
func Operator_Kind_1_Invariants(value Operator_Kind_1, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), OPERATOR_KIND_1_MINIMUM, OPERATOR_KIND_1_MAXIMUM).
		Ensure()
}

// Operator_Kind_2 is the class of one two-byte operator.
type Operator_Kind_2 int

// Operator_Kind_2_Invariants states the two-byte operator block.
func Operator_Kind_2_Invariants(value Operator_Kind_2, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), OPERATOR_KIND_2_MINIMUM, OPERATOR_KIND_2_MAXIMUM).
		Ensure()
}

// Operator_Kind_3 is the class of one three-byte operator.
type Operator_Kind_3 int

// Operator_Kind_3_Invariants states the four three-byte operators.
func Operator_Kind_3_Invariants(value Operator_Kind_3, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_4_Int(
			int(value), int(KIND_SHIFT_LEFT_ASSIGN), int(KIND_SHIFT_RIGHT_ASSIGN),
			int(KIND_AND_NOT_ASSIGN), int(KIND_ELLIPSIS),
		).
		Ensure()
}

// Operator_Size is the byte count of one operator.
type Operator_Size int

// Operator_Size_Invariants states the three legal operator widths.
func Operator_Size_Invariants(value Operator_Size, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Int(int(value), OPERATOR_SIZE_ONE, OPERATOR_SIZE_TWO, OPERATOR_SIZE_THREE).
		Ensure()
}

// Blank_Count is the count of empty lines before one token. It stands over a byte because a
// token already pads out to twelve, thus this count costs a source nothing.
type Blank_Count uint8

// Blank_Count_Invariants states the saturating empty line count.
func Blank_Count_Invariants(value Blank_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), BLANK_COUNT_MINIMUM, BLANK_COUNT_MAXIMUM).
		Ensure()
}

// Boolean is a true or false report about one scan.
type Boolean bool

// Boolean_Invariants states both scan reports as obligations.
func Boolean_Invariants(value Boolean, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The token scan report is true.").
		Ensure()
}

// Offset is a byte position in a source. It stands over a four-byte integer because a source
// spans at most one mebibyte, and a token run pays for this width twice for each token.
type Offset int32

// Offset_Invariants admits the position one past the final byte, which ends every scan.
func Offset_Invariants(value Offset, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), OFFSET_MINIMUM, OFFSET_MAXIMUM).
		Ensure()
}

// Size is the byte count of one token, zero for an inserted semicolon. It stands over a
// four-byte integer for the same reason Offset does.
type Size int32

// Size_Invariants admits the zero width of an inserted semicolon.
func Size_Invariants(value Size, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), SIZE_MINIMUM, SIZE_MAXIMUM).
		Ensure()
}

// Token_Size is the byte count of one token that source bytes spell, thus never zero.
type Token_Size int

// Token_Size_Invariants excludes the zero width that no source byte can produce.
func Token_Size_Invariants(value Token_Size, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), TOKEN_SIZE_MINIMUM, TOKEN_SIZE_MAXIMUM).
		Ensure()
}

// Comment_Size is the byte count of one comment, whose opener alone spans two bytes.
type Comment_Size int

// Comment_Size_Invariants counts the two bytes that every comment opener holds.
func Comment_Size_Invariants(value Comment_Size, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), COMMENT_SIZE_MINIMUM, COMMENT_SIZE_MAXIMUM).
		Ensure()
}

// Source is the Go source text of one file. The caller owns the bytes.
type Source []byte

// Source_Invariants caps the source so a hostile file is refused before the first read.
func Source_Invariants(value Source, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), SOURCE_SIZE_MINIMUM, SOURCE_SIZE_MAXIMUM).
		Ensure()
}

// Tail is the source that the cursor has not read, at a position that opens one token.
type Tail []byte

// Tail_Invariants excludes the exhausted source, which opens no token.
func Tail_Invariants(value Tail, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), TAIL_SIZE_MINIMUM, SOURCE_SIZE_MAXIMUM).
		Ensure()
}

// Comment_Tail is a tail whose first two bytes open a comment.
type Comment_Tail []byte

// Comment_Tail_Invariants states that both opener bytes are present.
func Comment_Tail_Invariants(value Comment_Tail, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), COMMENT_TAIL_SIZE_MINIMUM, SOURCE_SIZE_MAXIMUM).
		Ensure()
}

// Operator_Tail_2 is a tail long enough to hold a two-byte operator.
type Operator_Tail_2 []byte

// Operator_Tail_2_Invariants states that both operator bytes are present.
func Operator_Tail_2_Invariants(value Operator_Tail_2, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), OPERATOR_SIZE_TWO, SOURCE_SIZE_MAXIMUM).
		Ensure()
}

// Operator_Tail_3 is a tail long enough to hold a three-byte operator.
type Operator_Tail_3 []byte

// Operator_Tail_3_Invariants states that all three operator bytes are present.
func Operator_Tail_3_Invariants(value Operator_Tail_3, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), OPERATOR_SIZE_THREE, SOURCE_SIZE_MAXIMUM).
		Ensure()
}

// Exponent_Text is the one byte that stands before a sign inside a number.
type Exponent_Text []byte

// Exponent_Text_Invariants states that a sign is judged against one byte alone.
func Exponent_Text_Invariants(value Exponent_Text, namespace aver.Namespace) {
	aver.Always(
		len(value) == EXPONENT_TEXT_SIZE,
		"An exponent test reads the one byte before the sign.",
	)
}

// Token_Text is the exact byte run of one token that source bytes spell.
type Token_Text []byte

// Token_Text_Invariants excludes the empty run, which spells no token.
func Token_Text_Invariants(value Token_Text, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), TAIL_SIZE_MINIMUM, SOURCE_SIZE_MAXIMUM).
		Ensure()
}

// Token is one lexical unit of a source.
type Token struct {
	// Offset is the position of the first byte of this token in its source. An inserted
	// semicolon carries the position of the byte that follows the line it closes.
	Offset Offset
	// Size is the byte count of this token, zero for an inserted semicolon.
	Size Size
	// Kind is the lexical class this token belongs to. It stands last so the two four-byte
	// fields pack ahead of it and one token spans twelve bytes rather than sixteen.
	Kind Kind
	// Blanks is the count of empty lines between the token before this one and this one, so a
	// caller can write the source back out with the spacing its author gave it.
	Blanks Blank_Count
}

// Token_Invariants composes the class, the position, and the width of one token.
func Token_Invariants(value Token, namespace aver.Namespace) {
	Offset_Invariants(value.Offset, namespace)
	Size_Invariants(value.Size, namespace)
	Kind_Invariants(value.Kind, namespace)
	Blank_Count_Invariants(value.Blanks, namespace)
}

// Line is the line one offset stands on, counted from one.
type Line int32

// Line_Invariants states every line one index names.
func Line_Invariants(value Line, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), LINE_MINIMUM, LINE_MAXIMUM).
		Ensure()
}

// Column is the byte one offset stands at inside its line, counted from one. A tab counts as one
// byte, because a diagnostic names the byte a reader's editor counts to.
type Column int32

// Column_Invariants states every column one line holds.
func Column_Invariants(value Column, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), COLUMN_MINIMUM, COLUMN_MAXIMUM).
		Ensure()
}

// Line_Count is how many lines one index read.
type Line_Count int32

// Line_Count_Invariants states every count one index holds.
func Line_Count_Invariants(value Line_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), LINE_COUNT_MINIMUM, LINE_COUNT_MAXIMUM).
		Ensure()
}

// Line_Index holds where each line of one source opens. The caller owns it and reuses it for one
// file at a time, thus naming a position allocates nothing. The count lives in an array slot for
// the reason the scanner keeps its cursor in one: a field would owe its whole domain at every
// step that reads the index.
type Line_Index struct {
	// Starts holds the offset each line opens at, in the order the source states them.
	Starts [LINE_COUNT_MAXIMUM]Offset
	// Counts holds how many lines the index read.
	Counts [LINE_SLOT_COUNT]Line_Count
}

// Line_Index_Invariants states the storage the caller supplies.
func Line_Index_Invariants(subject *Line_Index, namespace aver.Namespace) {
	aver.Always(
		len(subject.Starts) == LINE_COUNT_MAXIMUM,
		"A line index holds one slot for every admitted line.",
	)
}

// Index_Lines reads where every line of one source opens. A source of more lines than the index
// holds is refused, and the index then names the lines it did read.
func Index_Lines(subject *Line_Index, source Source) (ok Boolean) {
	defer func() { Boolean_Invariants(ok, "index_lines.ok") }()
	Line_Index_Invariants(subject, "index_lines.subject")
	Source_Invariants(source, "index_lines.source")
	subject.Starts[0] = OFFSET_MINIMUM
	count := 1
	for offset := range len(source) {
		if source[offset] != '\n' {
			continue
		}
		if count == LINE_COUNT_MAXIMUM {
			subject.Counts[LINE_SLOT] = Line_Count(count)
			return false
		}
		subject.Starts[count] = Offset(offset + 1)
		count++
	}
	subject.Counts[LINE_SLOT] = Line_Count(count)
	return true
}

// Position_Of names the line and the column one offset stands at. The offset one past the final
// byte stands at the end of the final line, which is where a scan that ran out reports.
func Position_Of(subject *Line_Index, offset Offset) (line Line, column Column) {
	defer func() {
		Line_Invariants(line, "position_of.line")
		Column_Invariants(column, "position_of.column")
	}()
	Line_Index_Invariants(subject, "position_of.subject")
	Offset_Invariants(offset, "position_of.offset")
	// The starts rise with the lines, thus halving the run finds the line a byte stands on in
	// the steps a whole file's worth of lines needs and never in a walk of them.
	low := 0
	high := int(subject.Counts[LINE_SLOT]) - 1
	for low < high {
		middle := low + (high-low+1)/2
		if subject.Starts[middle] > offset {
			high = middle - 1
			continue
		}
		low = middle
	}
	return Line(low + 1), Column(int(offset) - int(subject.Starts[low]) + 1)
}

// Scanner is the cursor over one source. Its zero value with a Source set is ready to scan.
type Scanner struct {
	// Source is the text this cursor reads. The caller owns the bytes and the scanner never
	// writes them.
	Source Source
	// Offset is the position of the byte that the next scan reads.
	Offset Offset
	// Previous is the class of the last token that was no comment. Semicolon insertion reads
	// it, so a comment must not replace it or a comment at the end of a line would suppress
	// the semicolon that line owes.
	Previous Kind
	// Newline reports whether a line feed stands between Previous and Offset. A general
	// comment holding a line feed sets it, because such a comment closes its line.
	Newline Boolean
	// Blanks is the count of line feeds the last space skip crossed. Scan reads it, drops the
	// one that ended the line before, and hands the rest to the token it emits.
	Blanks Blank_Count
}

// Scanner_Invariants composes the text, the position, and the insertion state of one cursor.
func Scanner_Invariants(subject *Scanner, namespace aver.Namespace) {
	Source_Invariants(subject.Source, namespace)
	Offset_Invariants(subject.Offset, namespace)
	Kind_Invariants(subject.Previous, namespace)
	Boolean_Invariants(subject.Newline, namespace)
	Blank_Count_Invariants(subject.Blanks, namespace)
}

// Scan reads the token at the cursor and advances the cursor past it. An exhausted source scans
// to KIND_END_OF_FILE forever, thus a caller loop needs no stop test beyond that kind.
func Scan(subject *Scanner) (token Token) {
	defer func() { Token_Invariants(token, "scan.token") }()
	Scanner_Invariants(subject, "scan.subject")
	// A comment leaves the previous kind standing, thus the opening of the source is what the
	// count reads rather than the kind the end of a source carries.
	opening := subject.Offset == OFFSET_MINIMUM
	skip_space(subject)
	// The empty line count and the statement test stand here rather than in bodies of their
	// own, because a body of a few lines costs one call for every token a file spends.
	blanks := subject.Blanks
	if !opening {
		if blanks != BLANK_COUNT_MINIMUM {
			blanks = blanks - 1
		}
	}
	closes := false
	switch subject.Previous {
	case KIND_IDENTIFIER, KIND_INTEGER, KIND_FLOAT, KIND_IMAGINARY, KIND_CHARACTER,
		KIND_STRING, KIND_BREAK, KIND_CONTINUE, KIND_FALLTHROUGH, KIND_RETURN,
		KIND_INCREMENT, KIND_DECREMENT, KIND_PARENTHESIS_RIGHT, KIND_BRACKET_RIGHT,
		KIND_BRACE_RIGHT:
		closes = true
	}
	if bool(subject.Newline) {
		if closes {
			subject.Previous = KIND_SEMICOLON
			subject.Newline = false
			return Token{
				Kind:   KIND_SEMICOLON,
				Offset: subject.Offset,
				Size:   SIZE_MINIMUM,
				Blanks: blanks,
			}
		}
	}
	if int(subject.Offset) == len(subject.Source) {
		subject.Previous = KIND_END_OF_FILE
		return Token{
			Kind:   KIND_END_OF_FILE,
			Offset: subject.Offset,
			Size:   SIZE_MINIMUM,
			Blanks: blanks,
		}
	}
	start := subject.Offset
	kind, size, newline := scan_token(Tail(subject.Source[start:]))
	subject.Offset = start + Offset(size)
	if kind != Scanned_Kind(KIND_COMMENT) {
		subject.Previous = Kind(kind)
		subject.Newline = false
	}
	if bool(newline) {
		subject.Newline = true
	}
	return Token{
		Kind:   Kind(kind),
		Offset: start,
		Size:   Size(size),
		Blanks: blanks,
	}
}

// Text returns the bytes one token spans. An inserted semicolon spans none, thus it returns an
// empty view rather than a semicolon the source never held.
func Text(source Source, one Token) (text Source) {
	defer func() { Source_Invariants(text, "text.text") }()
	Source_Invariants(source, "text.source")
	Token_Invariants(one, "text.one")
	return source[one.Offset : int(one.Offset)+int(one.Size)]
}

// Advances the cursor past every separator byte, recording whether a line feed was among them.
// The exhausted source counts as a line feed, because the end of a source closes its final line.
func skip_space(subject *Scanner) {
	Scanner_Invariants(subject, "skip_space.subject")
	subject.Blanks = BLANK_COUNT_MINIMUM
	for int(subject.Offset) < len(subject.Source) {
		value := subject.Source[subject.Offset]
		switch value {
		case ' ', '\t', '\r':
			subject.Offset++
		case '\n':
			subject.Newline = true
			if subject.Blanks < BLANK_COUNT_MAXIMUM {
				subject.Blanks++
			}
			subject.Offset++
		default:
			return
		}
	}
	subject.Newline = true
}

// Scans the one token that opens a tail. Every byte that opens no other token opens KIND_ILLEGAL
// of width one, thus a hostile source advances the cursor and never stalls a scan.
func scan_token(tail Tail) (kind Scanned_Kind, size Token_Size, newline Boolean) {
	defer func() {
		Scanned_Kind_Invariants(kind, "scan_token.kind")
		Token_Size_Invariants(size, "scan_token.size")
		Boolean_Invariants(newline, "scan_token.newline")
	}()
	Tail_Invariants(tail, "scan_token.tail")
	first := tail[0]
	switch {
	case first == '_', first >= 'a' && first <= 'z', first >= 'A' && first <= 'Z':
		word, word_size := scan_identifier(tail)
		return Scanned_Kind(word), word_size, false
	case first >= '0' && first <= '9':
		number, number_size := scan_number(tail)
		return Scanned_Kind(number), number_size, false
	case first == '.':
		if bool(starts_fraction(tail)) {
			number, number_size := scan_number(tail)
			return Scanned_Kind(number), number_size, false
		}
	case first == '\'', first == '"', first == '`':
		quoted, quoted_size := scan_quoted_token(tail)
		return Scanned_Kind(quoted), quoted_size, false
	case first == '/':
		if bool(starts_comment(tail)) {
			comment_newline, comment_size := scan_comment(Comment_Tail(tail))
			return Scanned_Kind(KIND_COMMENT), Token_Size(comment_size), comment_newline
		}
	}
	operator, operator_size, found := scan_operator(tail)
	if bool(found) {
		return Scanned_Kind(operator), Token_Size(operator_size), false
	}
	return Scanned_Kind(KIND_ILLEGAL), TOKEN_SIZE_MINIMUM, false
}

// Scans one quoted literal and names its class. An unclosed literal is KIND_ILLEGAL, because a
// source that never closes a quote gives the parser no literal to read.
func scan_quoted_token(tail Tail) (kind Quoted_Kind, size Token_Size) {
	defer func() {
		Quoted_Kind_Invariants(kind, "scan_quoted_token.kind")
		Token_Size_Invariants(size, "scan_quoted_token.size")
	}()
	Tail_Invariants(tail, "scan_quoted_token.tail")
	closed, quoted_size := scan_quoted(tail)
	if !bool(closed) {
		return Quoted_Kind(KIND_ILLEGAL), quoted_size
	}
	if tail[0] == '\'' {
		return Quoted_Kind(KIND_CHARACTER), quoted_size
	}
	return Quoted_Kind(KIND_STRING), quoted_size
}

// Reports whether a period opens a fraction, the one form that starts a number with no digit.
func starts_fraction(tail Tail) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "starts_fraction.yes") }()
	Tail_Invariants(tail, "starts_fraction.tail")
	if len(tail) < FRACTION_TAIL_SIZE_MINIMUM {
		return false
	}
	switch tail[1] {
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return true
	}
	return false
}

// Reports whether a slash opens a comment rather than a division operator.
func starts_comment(tail Tail) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "starts_comment.yes") }()
	Tail_Invariants(tail, "starts_comment.tail")
	if len(tail) < COMMENT_TAIL_SIZE_MINIMUM {
		return false
	}
	switch tail[1] {
	case '/', '*':
		return true
	}
	return false
}

// Scans one letter run and names it, so a keyword never reaches the parser as an identifier.
func scan_identifier(tail Tail) (kind Word_Kind, size Token_Size) {
	defer func() {
		Word_Kind_Invariants(kind, "scan_identifier.kind")
		Token_Size_Invariants(size, "scan_identifier.size")
	}()
	Tail_Invariants(tail, "scan_identifier.tail")
	// The run width stands in a local, because a defined width behind a conversion is a value
	// the compiler reloads and rechecks at every step of the run.
	width := 0
Word:
	for width < len(tail) {
		value := tail[width]
		switch {
		case value == '_', value >= 'a' && value <= 'z',
			value >= 'A' && value <= 'Z', value >= '0' && value <= '9':
			width++
		default:
			break Word
		}
	}
	// The keyword table stands here rather than behind a body of its own, because every word a
	// file spells reaches it and a body of three lines costs one call for each of them.
	keyword, found := keyword_lookup(Token_Text(tail[:width]))
	kind = Word_Kind(KIND_IDENTIFIER)
	if bool(found) {
		kind = Word_Kind(keyword)
	}
	return kind, Token_Size(width)
}

// Looks one letter run up in the twenty-five Go keywords.
func keyword_lookup(text Token_Text) (kind Keyword_Kind, found Boolean) {
	defer func() {
		Keyword_Kind_Invariants(kind, "keyword_lookup.kind")
		Boolean_Invariants(found, "keyword_lookup.found")
	}()
	Token_Text_Invariants(text, "keyword_lookup.text")
	// The lookup states one exit, because a body of twenty-five exits puts its defer on the
	// stack and pays the runtime for each one.
	kind = Keyword_Kind(KEYWORD_KIND_MINIMUM)
	switch string(text) {
	case "break":
		kind, found = Keyword_Kind(KIND_BREAK), true
	case "case":
		kind, found = Keyword_Kind(KIND_CASE), true
	case "chan":
		kind, found = Keyword_Kind(KIND_CHANNEL), true
	case "const":
		kind, found = Keyword_Kind(KIND_CONSTANT), true
	case "continue":
		kind, found = Keyword_Kind(KIND_CONTINUE), true
	case "default":
		kind, found = Keyword_Kind(KIND_DEFAULT), true
	case "defer":
		kind, found = Keyword_Kind(KIND_DEFER), true
	case "else":
		kind, found = Keyword_Kind(KIND_ELSE), true
	case "fallthrough":
		kind, found = Keyword_Kind(KIND_FALLTHROUGH), true
	case "for":
		kind, found = Keyword_Kind(KIND_FOR), true
	case "func":
		kind, found = Keyword_Kind(KIND_FUNCTION), true
	case "go":
		kind, found = Keyword_Kind(KIND_GO), true
	case "goto":
		kind, found = Keyword_Kind(KIND_GOTO), true
	case "if":
		kind, found = Keyword_Kind(KIND_IF), true
	case "import":
		kind, found = Keyword_Kind(KIND_IMPORT), true
	case "interface":
		kind, found = Keyword_Kind(KIND_INTERFACE), true
	case "map":
		kind, found = Keyword_Kind(KIND_MAP), true
	case "package":
		kind, found = Keyword_Kind(KIND_PACKAGE), true
	case "range":
		kind, found = Keyword_Kind(KIND_RANGE), true
	case "return":
		kind, found = Keyword_Kind(KIND_RETURN), true
	case "select":
		kind, found = Keyword_Kind(KIND_SELECT), true
	case "struct":
		kind, found = Keyword_Kind(KIND_STRUCTURE), true
	case "switch":
		kind, found = Keyword_Kind(KIND_SWITCH), true
	case "type":
		kind, found = Keyword_Kind(KIND_TYPE), true
	case "var":
		kind, found = Keyword_Kind(KIND_VARIABLE), true
	}
	return kind, found
}

// Scans one numeric literal whole. A malformed literal still scans whole, because the parser
// reports a bad literal with more context than a scanner holds.
func scan_number(tail Tail) (kind Number_Kind, size Token_Size) {
	defer func() {
		Number_Kind_Invariants(kind, "scan_number.kind")
		Token_Size_Invariants(size, "scan_number.size")
	}()
	Tail_Invariants(tail, "scan_number.tail")
	hexadecimal := false
	if tail[0] == '0' {
		if len(tail) > TOKEN_SIZE_MINIMUM {
			switch tail[TOKEN_SIZE_MINIMUM] {
			case 'x', 'X':
				hexadecimal = true
			}
		}
	}
	size = TOKEN_SIZE_MINIMUM
Digits:
	for int(size) < len(tail) {
		value := tail[size]
		switch {
		case value == '_', value == '.', value >= '0' && value <= '9',
			value >= 'a' && value <= 'z', value >= 'A' && value <= 'Z':
			size++
		case value == '+', value == '-':
			letter := Exponent_Text(tail[size-1 : size])
			if !bool(after_exponent(letter, Boolean(hexadecimal))) {
				break Digits
			}
			size++
		default:
			break Digits
		}
	}
	return number_kind(Token_Text(tail[:size]), Boolean(hexadecimal)), size
}

// Reports whether a sign follows an exponent letter, the one place a sign joins a number.
func after_exponent(text Exponent_Text, hexadecimal Boolean) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "after_exponent.yes") }()
	Exponent_Text_Invariants(text, "after_exponent.text")
	Boolean_Invariants(hexadecimal, "after_exponent.hexadecimal")
	switch text[0] {
	case 'e', 'E':
		return !hexadecimal
	case 'p', 'P':
		return hexadecimal
	}
	return false
}

// Names one complete numeric literal. A period or an exponent makes it a float and a trailing i
// makes it imaginary, and the imaginary suffix wins because it may follow either form.
func number_kind(text Token_Text, hexadecimal Boolean) (kind Number_Kind) {
	defer func() { Number_Kind_Invariants(kind, "number_kind.kind") }()
	Token_Text_Invariants(text, "number_kind.text")
	Boolean_Invariants(hexadecimal, "number_kind.hexadecimal")
	kind = Number_Kind(KIND_INTEGER)
	for index := range text {
		switch text[index] {
		case '.':
			kind = Number_Kind(KIND_FLOAT)
		case 'e', 'E':
			if !bool(hexadecimal) {
				kind = Number_Kind(KIND_FLOAT)
			}
		case 'p', 'P':
			if bool(hexadecimal) {
				kind = Number_Kind(KIND_FLOAT)
			}
		}
	}
	if text[len(text)-1] == 'i' {
		return Number_Kind(KIND_IMAGINARY)
	}
	return kind
}

// Scans one quoted literal from its opening quote through the quote that closes it. A raw
// literal takes no escape and may hold line feeds; the other two forms end at the line feed that
// no quote closed, because Go admits no literal that spans a line.
func scan_quoted(tail Tail) (closed Boolean, size Token_Size) {
	defer func() {
		Boolean_Invariants(closed, "scan_quoted.closed")
		Token_Size_Invariants(size, "scan_quoted.size")
	}()
	Tail_Invariants(tail, "scan_quoted.tail")
	quote := tail[0]
	raw := false
	if quote == '`' {
		raw = true
	}
	size = TOKEN_SIZE_MINIMUM
	for int(size) < len(tail) {
		value := tail[size]
		size++
		if value == quote {
			return true, size
		}
		if raw {
			continue
		}
		if value == '\n' {
			// The line feed opens the next line, thus the literal stops before it.
			size--
			return false, size
		}
		if value == '\\' {
			if int(size) < len(tail) {
				size++
			}
		}
	}
	return false, size
}

// Scans one comment. A line comment stops before the line feed that ends it, so the cursor still
// sees that line feed and the line it closes still gets its inserted semicolon.
func scan_comment(tail Comment_Tail) (newline Boolean, size Comment_Size) {
	defer func() {
		Boolean_Invariants(newline, "scan_comment.newline")
		Comment_Size_Invariants(size, "scan_comment.size")
	}()
	Comment_Tail_Invariants(tail, "scan_comment.tail")
	size = COMMENT_SIZE_MINIMUM
	if tail[1] == '/' {
		for int(size) < len(tail) {
			if tail[size] == '\n' {
				return false, size
			}
			size++
		}
		return false, size
	}
	for int(size) < len(tail) {
		value := tail[size]
		size++
		if value == '\n' {
			newline = true
		}
		if value == '*' {
			if int(size) < len(tail) {
				if tail[size] == '/' {
					size++
					return newline, size
				}
			}
		}
	}
	return newline, size
}

// Scans one operator, longest match first, so an and-not assignment never scans as an and.
func scan_operator(tail Tail) (kind Operator_Kind, size Operator_Size, found Boolean) {
	defer func() {
		Operator_Kind_Invariants(kind, "scan_operator.kind")
		Operator_Size_Invariants(size, "scan_operator.size")
		Boolean_Invariants(found, "scan_operator.found")
	}()
	Tail_Invariants(tail, "scan_operator.tail")
	// A byte that opens no wider operator skips both wider tables. A brace, a bracket, a
	// parenthesis, a comma, and a semicolon are the tokens source spends most, thus the common
	// byte reaches its own table at once.
	wider := false
	switch tail[0] {
	case '<', '>', '&', '|', '^', '+', '-', '*', '/', '%', '=', '!', ':', '.':
		wider = true
	}
	if wider {
		if len(tail) >= OPERATOR_SIZE_THREE {
			three, three_found := operator_lookup_3(Operator_Tail_3(tail))
			if bool(three_found) {
				return Operator_Kind(three), OPERATOR_SIZE_THREE, true
			}
		}
		if len(tail) >= OPERATOR_SIZE_TWO {
			two, two_found := operator_lookup_2(Operator_Tail_2(tail))
			if bool(two_found) {
				return Operator_Kind(two), OPERATOR_SIZE_TWO, true
			}
		}
	}
	one, one_found := operator_lookup_1(tail)
	if bool(one_found) {
		return Operator_Kind(one), OPERATOR_SIZE_ONE, true
	}
	return Operator_Kind(OPERATOR_KIND_MINIMUM), OPERATOR_SIZE_ONE, false
}

// Looks the first three bytes up in the four three-byte Go operators.
func operator_lookup_3(tail Operator_Tail_3) (kind Operator_Kind_3, found Boolean) {
	defer func() {
		Operator_Kind_3_Invariants(kind, "operator_lookup_3.kind")
		Boolean_Invariants(found, "operator_lookup_3.found")
	}()
	Operator_Tail_3_Invariants(tail, "operator_lookup_3.tail")
	kind = Operator_Kind_3(KIND_SHIFT_LEFT_ASSIGN)
	switch string(tail[:OPERATOR_SIZE_THREE]) {
	case "<<=":
		kind, found = Operator_Kind_3(KIND_SHIFT_LEFT_ASSIGN), true
	case ">>=":
		kind, found = Operator_Kind_3(KIND_SHIFT_RIGHT_ASSIGN), true
	case "&^=":
		kind, found = Operator_Kind_3(KIND_AND_NOT_ASSIGN), true
	case "...":
		kind, found = Operator_Kind_3(KIND_ELLIPSIS), true
	}
	return kind, found
}

// Looks the first two bytes up in the twenty-one two-byte Go operators.
func operator_lookup_2(tail Operator_Tail_2) (kind Operator_Kind_2, found Boolean) {
	defer func() {
		Operator_Kind_2_Invariants(kind, "operator_lookup_2.kind")
		Boolean_Invariants(found, "operator_lookup_2.found")
	}()
	Operator_Tail_2_Invariants(tail, "operator_lookup_2.tail")
	kind = Operator_Kind_2(OPERATOR_KIND_2_MINIMUM)
	switch string(tail[:OPERATOR_SIZE_TWO]) {
	case "+=":
		kind, found = Operator_Kind_2(KIND_PLUS_ASSIGN), true
	case "-=":
		kind, found = Operator_Kind_2(KIND_MINUS_ASSIGN), true
	case "*=":
		kind, found = Operator_Kind_2(KIND_STAR_ASSIGN), true
	case "/=":
		kind, found = Operator_Kind_2(KIND_SLASH_ASSIGN), true
	case "%=":
		kind, found = Operator_Kind_2(KIND_PERCENT_ASSIGN), true
	case "&=":
		kind, found = Operator_Kind_2(KIND_AND_ASSIGN), true
	case "|=":
		kind, found = Operator_Kind_2(KIND_OR_ASSIGN), true
	case "^=":
		kind, found = Operator_Kind_2(KIND_EXCLUSIVE_OR_ASSIGN), true
	case "<<":
		kind, found = Operator_Kind_2(KIND_SHIFT_LEFT), true
	case ">>":
		kind, found = Operator_Kind_2(KIND_SHIFT_RIGHT), true
	case "&^":
		kind, found = Operator_Kind_2(KIND_AND_NOT), true
	case "&&":
		kind, found = Operator_Kind_2(KIND_LOGICAL_AND), true
	case "||":
		kind, found = Operator_Kind_2(KIND_LOGICAL_OR), true
	case "<-":
		kind, found = Operator_Kind_2(KIND_ARROW), true
	case "++":
		kind, found = Operator_Kind_2(KIND_INCREMENT), true
	case "--":
		kind, found = Operator_Kind_2(KIND_DECREMENT), true
	case "==":
		kind, found = Operator_Kind_2(KIND_EQUAL), true
	case "!=":
		kind, found = Operator_Kind_2(KIND_NOT_EQUAL), true
	case "<=":
		kind, found = Operator_Kind_2(KIND_LESS_EQUAL), true
	case ">=":
		kind, found = Operator_Kind_2(KIND_GREATER_EQUAL), true
	case ":=":
		kind, found = Operator_Kind_2(KIND_DEFINE), true
	}
	return kind, found
}

// Looks the first byte up in the twenty-three one-byte Go operators and delimiters.
func operator_lookup_1(tail Tail) (kind Operator_Kind_1, found Boolean) {
	defer func() {
		Operator_Kind_1_Invariants(kind, "operator_lookup_1.kind")
		Boolean_Invariants(found, "operator_lookup_1.found")
	}()
	Tail_Invariants(tail, "operator_lookup_1.tail")
	kind = Operator_Kind_1(OPERATOR_KIND_1_MINIMUM)
	switch tail[0] {
	case '+':
		kind, found = Operator_Kind_1(KIND_PLUS), true
	case '-':
		kind, found = Operator_Kind_1(KIND_MINUS), true
	case '*':
		kind, found = Operator_Kind_1(KIND_STAR), true
	case '/':
		kind, found = Operator_Kind_1(KIND_SLASH), true
	case '%':
		kind, found = Operator_Kind_1(KIND_PERCENT), true
	case '&':
		kind, found = Operator_Kind_1(KIND_AND), true
	case '|':
		kind, found = Operator_Kind_1(KIND_OR), true
	case '^':
		kind, found = Operator_Kind_1(KIND_EXCLUSIVE_OR), true
	case '<':
		kind, found = Operator_Kind_1(KIND_LESS), true
	case '>':
		kind, found = Operator_Kind_1(KIND_GREATER), true
	case '=':
		kind, found = Operator_Kind_1(KIND_ASSIGN), true
	case '!':
		kind, found = Operator_Kind_1(KIND_NOT), true
	case '(':
		kind, found = Operator_Kind_1(KIND_PARENTHESIS_LEFT), true
	case ')':
		kind, found = Operator_Kind_1(KIND_PARENTHESIS_RIGHT), true
	case '[':
		kind, found = Operator_Kind_1(KIND_BRACKET_LEFT), true
	case ']':
		kind, found = Operator_Kind_1(KIND_BRACKET_RIGHT), true
	case '{':
		kind, found = Operator_Kind_1(KIND_BRACE_LEFT), true
	case '}':
		kind, found = Operator_Kind_1(KIND_BRACE_RIGHT), true
	case ',':
		kind, found = Operator_Kind_1(KIND_COMMA), true
	case '.':
		kind, found = Operator_Kind_1(KIND_PERIOD), true
	case ';':
		kind, found = Operator_Kind_1(KIND_SEMICOLON), true
	case ':':
		kind, found = Operator_Kind_1(KIND_COLON), true
	case '~':
		kind, found = Operator_Kind_1(KIND_TILDE), true
	}
	return kind, found
}
