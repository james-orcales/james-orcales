// Package requirements checks one package's contract: that it carries REQUIREMENTS.md, that the
// markdown obeys the doctrine, and that requirements_test.go mirrors the leaf headings. It reads
// no file. The caller loads each file into caller-owned storage and resumes the check, which stops
// at every finding so the caller can report it before the next.
package requirements

import "local/james-orcales/shared/bytes"
import "local/james-orcales/shared/filepath"
import "local/james-orcales/shared/go/ast"
import "local/james-orcales/shared/go/token"
import "local/james-orcales/shared/math/bits"
import "local/james-orcales/shared/sim/aver/default"
import "local/james-orcales/shared/slices"
import "local/james-orcales/shared/unicode/ucd"
import "local/james-orcales/shared/unicode/utf8"

// MARKDOWN_NAME is the exact-cased contract file; a differently cased file on a case-insensitive
// disk is no contract.
const MARKDOWN_NAME = "REQUIREMENTS.md"

// TEST_NAME is the exact-cased test file that mirrors the contract.
const TEST_NAME = "requirements_test.go"

// HEADING_SIZE_MAXIMUM bounds one heading's text, so every message quoting it stays bounded.
const HEADING_SIZE_MAXIMUM = 128

// HEADING_COUNT_MAXIMUM bounds the headings one REQUIREMENTS.md holds.
const HEADING_COUNT_MAXIMUM = 1024

// MARKDOWN_STORAGE_SIZE holds one byte past the largest REQUIREMENTS.md, so a full read proves a
// file oversized without a second read.
const MARKDOWN_STORAGE_SIZE = 64*bits.KIBIBYTE_BYTES + 1

// UPPER_CASE_SIZE_MAXIMUM is how wide an upper case mapping can make one byte of a heading.
const UPPER_CASE_SIZE_MAXIMUM = utf8.CHARACTER_SIZE_MAXIMUM

// NAME_SIZE_MAXIMUM holds one test name: its prefix and two headings joined by an underscore.
const NAME_SIZE_MAXIMUM = len(TEST_PREFIX) + 2*UPPER_CASE_SIZE_MAXIMUM*HEADING_SIZE_MAXIMUM + 1

// MESSAGE_SIZE_MAXIMUM holds the longest message: a test name, its leaf, and fixed text.
const MESSAGE_SIZE_MAXIMUM = 2*NAME_SIZE_MAXIMUM + 128

// STEP_COUNT_MAXIMUM bounds the steps one Next takes: each line of the file costs a few steps in
// each pass, and each heading and leaf one step.
const STEP_COUNT_MAXIMUM = 16 * MARKDOWN_STORAGE_SIZE

// TEST_PREFIX opens the name of every leaf's test.
const TEST_PREFIX = "Test_"

// ABSENT names no heading.
const ABSENT Index = -1

// INDEX_MAXIMUM is one past the widest bounded input, a Go source.
const INDEX_MAXIMUM = token.SOURCE_SIZE_MAXIMUM + 1

// LEVEL_BRANCH is a # heading.
const LEVEL_BRANCH Index = 1

// LEVEL_LEAF is a ### heading.
const LEVEL_LEAF Index = 3

// FILE_MARKDOWN is a finding in REQUIREMENTS.md.
const FILE_MARKDOWN File = 0

// FILE_TEST is a finding in requirements_test.go.
const FILE_TEST File = 1

// STATE_STEP is a check between findings.
const STATE_STEP State = 0

// STATE_FINDING is a finding in Message, File, and Line, waiting for Next.
const STATE_FINDING State = 1

// STATE_TEST asks the caller for requirements_test.go in Test_Source before Next.
const STATE_TEST State = 2

// STATE_DONE is a finished check.
const STATE_DONE State = 3

// PASS_PRESENCE decides whether the package owes a REQUIREMENTS.md.
const PASS_PRESENCE Pass = 0

// PASS_HEADINGS collects headings and flags line-local faults.
const PASS_HEADINGS Pass = 1

// PASS_TREE decides leaves and flags orphan and repeated headings.
const PASS_TREE Pass = 2

// PASS_BODIES flags empty, oversized, and gapped sections.
const PASS_BODIES Pass = 3

// PASS_TEST_FILE asks for requirements_test.go, or flags its absence.
const PASS_TEST_FILE Pass = 4

// PASS_TEST_PARSE collects the leading declarations of requirements_test.go.
const PASS_TEST_PARSE Pass = 5

// PASS_TESTS flags each leading declaration that is not its leaf's test.
const PASS_TESTS Pass = 6

// PASS_DONE ends the check.
const PASS_DONE Pass = 7

// MESSAGE_MARKDOWN_ABSENT is the finding of a package that owes a contract.
const MESSAGE_MARKDOWN_ABSENT Text = "The package has no REQUIREMENTS.md. Add REQUIREMENTS.md."

// MESSAGE_MARKDOWN_OVERSIZED is the finding of a contract past its storage.
const MESSAGE_MARKDOWN_OVERSIZED Text = "The file is larger than its storage. Write a shorter file."

// MESSAGE_HEADING_OVERSIZED is the finding of a heading past HEADING_SIZE_MAXIMUM.
const MESSAGE_HEADING_OVERSIZED Text = "The heading is longer than 128 bytes. " +
	"Write a shorter heading."

// MESSAGE_HEADINGS_FULL is the finding of a contract past its heading table.
const MESSAGE_HEADINGS_FULL Text = "The file has more headings than its heading table holds. " +
	"Write fewer headings."

// MESSAGE_HEADING_LEVEL is the finding of a heading neither # nor ###.
const MESSAGE_HEADING_LEVEL Text = `The heading level is not "#" or "###". ` +
	`Write a "#" heading or a "###" heading.`

// MESSAGE_PREAMBLE is the finding of content before the first heading.
const MESSAGE_PREAMBLE Text = "The content is before the first heading. " +
	"Start the file with a heading."

// MESSAGE_TEST_ABSENT is the finding of a contract without its test file.
const MESSAGE_TEST_ABSENT Text = "The package has no requirements_test.go. " +
	"Add requirements_test.go."

// SECTION_BODY_MAXIMUM is the most body lines one section holds.
const SECTION_BODY_MAXIMUM = 3

// Index is a count, a line number, an offset, or a slot of a bounded table.
type Index int

// Index_Invariants bounds an index by the widest bounded input, a Go source.
func Index_Invariants(value Index, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), int(ABSENT), INDEX_MAXIMUM).
		Ensure()
}

// Flag is one yes or no the check records.
type Flag bool

// Flag_Invariants proves both answers occur.
func Flag_Invariants(value Flag, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The requirements flag is set.").
		Ensure()
}

// File is the file a finding stands in.
type File int

// File_Invariants admits both files.
func File_Invariants(value File, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Int(int(value), int(FILE_MARKDOWN), int(FILE_TEST)).
		Ensure()
}

// State is what Next leaves for the caller.
type State int

// State_Invariants admits each state.
func State_Invariants(value State, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_4_Int(
			int(value),
			int(STATE_STEP),
			int(STATE_FINDING),
			int(STATE_TEST),
			int(STATE_DONE),
		).
		Ensure()
}

// Pass is the check the package stands in.
type Pass int

// Pass_Invariants admits each pass.
func Pass_Invariants(value Pass, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), int(PASS_PRESENCE), int(PASS_DONE)).
		Ensure()
}

// Markdown_Bytes is a REQUIREMENTS.md as read.
type Markdown_Bytes []byte

// Markdown_Bytes_Invariants bounds a read contract by its storage.
func Markdown_Bytes_Invariants(value Markdown_Bytes, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value),
			bytes.SLICE_SIZE_MINIMUM,
			MARKDOWN_STORAGE_SIZE,
		).
		Ensure()
}

// Text_Storage holds a message or a test name being written.
type Text_Storage []byte

// Text_Storage_Invariants bounds text storage by the longest message.
func Text_Storage_Invariants(value Text_Storage, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value),
			bytes.SLICE_SIZE_MINIMUM,
			MESSAGE_SIZE_MAXIMUM,
		).
		Ensure()
}

// Text is a fixed piece of a message.
type Text string

// Text_Invariants bounds a piece by the longest message.
func Text_Invariants(value Text, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value),
			bytes.TEXT_SIZE_MINIMUM,
			MESSAGE_SIZE_MAXIMUM,
		).
		Ensure()
}

// Heading is one # or ### heading of a REQUIREMENTS.md.
type Heading struct {
	// Level is LEVEL_BRANCH or LEVEL_LEAF.
	Level Index
	// Line is the 1-based line the heading stands on.
	Line Index
	// Start is the offset of the heading text after its marker.
	Start Index
	// End is the offset past the heading text.
	End Index
	// Leaf records a heading whose section needs a body and a test.
	Leaf Flag
}

// Heading_Invariants composes one heading.
func Heading_Invariants(value Heading, namespace aver.Namespace) {
	Index_Invariants(value.Level, namespace)
	Index_Invariants(value.Line, namespace)
	Index_Invariants(value.Start, namespace)
	Index_Invariants(value.End, namespace)
	Flag_Invariants(value.Leaf, namespace)
}

// Headings holds the headings of the file in order.
type Headings []Heading

// Headings_Invariants bounds the heading table.
func Headings_Invariants(value Headings, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value),
			slices.COUNT_MINIMUM,
			HEADING_COUNT_MAXIMUM,
		).
		Ensure()
}

// Leaf is one leaf section: a # alone, or a ### under its #.
type Leaf struct {
	// Parent is the # above a ### leaf, or ABSENT for a # leaf.
	Parent Index
	// Heading is the leaf's own heading.
	Heading Index
}

// Leaf_Invariants composes one leaf.
func Leaf_Invariants(value Leaf, namespace aver.Namespace) {
	Index_Invariants(value.Parent, namespace)
	Index_Invariants(value.Heading, namespace)
}

// Leaves holds the leaves of the file in order; each heading is a leaf at most once, thus the
// table holds as many slots as the heading table.
type Leaves []Leaf

// Leaves_Invariants bounds the leaf table by the heading table.
func Leaves_Invariants(value Leaves, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value),
			slices.COUNT_MINIMUM,
			HEADING_COUNT_MAXIMUM,
		).
		Ensure()
}

// Declaration is one leading declaration of requirements_test.go: a function by its name, or a
// const, var, or type by its keyword.
type Declaration struct {
	// Start is the offset of the name.
	Start Index
	// End is the offset past the name.
	End Index
	// Line is the line the name stands on, or 0 past the line index.
	Line Index
}

// Declaration_Invariants composes one declaration.
func Declaration_Invariants(value Declaration, namespace aver.Namespace) {
	Index_Invariants(value.Start, namespace)
	Index_Invariants(value.End, namespace)
	Index_Invariants(value.Line, namespace)
}

// Declarations holds the leading declarations, one slot per leaf at most.
type Declarations []Declaration

// Declarations_Invariants bounds the table by the leaf table.
func Declarations_Invariants(value Declarations, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value),
			slices.COUNT_MINIMUM,
			HEADING_COUNT_MAXIMUM,
		).
		Ensure()
}

// Checker_Fields is one package's check. The caller fills the inputs and storage, calls Begin,
// then calls Next until State is STATE_DONE, reporting each STATE_FINDING and answering each
// STATE_TEST.
type Checker_Fields struct {
	// Path is the package directory; a third_party or examples segment exempts it.
	Path filepath.Slice
	// Has_Markdown records an exact-cased REQUIREMENTS.md.
	Has_Markdown Flag
	// Has_Test records an exact-cased requirements_test.go.
	Has_Test Flag
	// Impure records a package main file, which exempts the package.
	Impure Flag
	// Markdown holds REQUIREMENTS.md; the caller loads it before Begin.
	Markdown Markdown_Bytes
	// Markdown_Count counts the loaded bytes of Markdown.
	Markdown_Count Index
	// Test_Source is requirements_test.go, which the caller sets when State is STATE_TEST.
	Test_Source token.Source
	// Parse_State parses Test_Source.
	Parse_State ast.Parse_State_Handle
	// Lines indexes Test_Source, so a finding names a declaration's line.
	Lines token.Line_Index_Handle
	// Headings holds the headings.
	Headings Headings
	// Leaves holds the leaves.
	Leaves Leaves
	// Declarations holds the leading declarations of Test_Source.
	Declarations Declarations
	// Name_Storage holds the test name a leaf needs.
	Name_Storage Text_Storage
	// Message_Storage holds the message of a finding.
	Message_Storage Text_Storage
	// Message views the message of a finding.
	Message Text_Storage
	// File is the file a finding stands in.
	File File
	// Line is the line a finding stands on, or 0 for the whole file.
	Line Index
	// State is what Next left for the caller.
	State State
	// Pass is the check the package stands in.
	Pass Pass
	// Line_Offset is where the scanned line opens.
	Line_Offset Index
	// Line_End is where the scanned line closes.
	Line_End Index
	// Line_Number is the scanned line's number.
	Line_Number Index
	// Line_Check is the next check of the scanned line.
	Line_Check Index
	// Seen_Heading records a heading above the scanned line.
	Seen_Heading Flag
	// Heading_Count counts the headings.
	Heading_Count Index
	// Heading_Cursor is the next heading a pass visits.
	Heading_Cursor Index
	// Parent is the open # heading, or ABSENT.
	Parent Index
	// Has_Child records a ### under the open #.
	Has_Child Flag
	// Leaf_Count counts the leaves.
	Leaf_Count Index
	// Open is the heading whose section the scanned line belongs to, or ABSENT.
	Open Index
	// Body_Count counts the open section's body lines.
	Body_Count Index
	// Blank records a blank line inside the open section's body.
	Blank Flag
	// Leaf_Cursor is the next leaf whose test the check compares.
	Leaf_Cursor Index
	// Declaration_Count counts the declarations.
	Declaration_Count Index
	// Scan_Start is where the text a scan reads opens.
	Scan_Start Index
	// Scan_End is where it closes.
	Scan_End Index
	// Scan_Result is what the scan found.
	Scan_Result Flag
	// Scan_Position is the next rune a scan decodes.
	Scan_Position Index
	// Character is the rune the scan decoded.
	Character utf8.Decoded_Character
	// Character_Space records white space.
	Character_Space Flag
	// Character_Word records a letter or a digit.
	Character_Word Flag
	// Upper is the upper case of Character.
	Upper ucd.Character
	// Node is the top-level declaration whose name the check reads.
	Node ast.Node
	// Name_Token names Node, or ast.INDEX_ABSENT.
	Name_Token ast.Index
	// Next_Open is the section a closing section hands over to.
	Next_Open Index
	// Finding_Heading is the heading a finding quotes.
	Finding_Heading Index
	// Text_Before opens a finding's message.
	Text_Before Text
	// Text_After closes a quoting finding's message.
	Text_After Text
}

// Checker_Fields_Invariants composes every check field.
func Checker_Fields_Invariants(value Checker_Fields, namespace aver.Namespace) {
	filepath.Slice_Invariants(value.Path, namespace)
	Flag_Invariants(value.Has_Markdown, namespace)
	Flag_Invariants(value.Has_Test, namespace)
	Flag_Invariants(value.Impure, namespace)
	Markdown_Bytes_Invariants(value.Markdown, namespace)
	Index_Invariants(value.Markdown_Count, namespace)
	token.Source_Invariants(value.Test_Source, namespace)
	ast.Parse_State_Handle_Invariants(value.Parse_State, namespace)
	token.Line_Index_Handle_Invariants(value.Lines, namespace)
	Headings_Invariants(value.Headings, namespace)
	Leaves_Invariants(value.Leaves, namespace)
	Declarations_Invariants(value.Declarations, namespace)
	Text_Storage_Invariants(value.Name_Storage, namespace)
	Text_Storage_Invariants(value.Message_Storage, namespace)
	Text_Storage_Invariants(value.Message, namespace)
	File_Invariants(value.File, namespace)
	Index_Invariants(value.Line, namespace)
	State_Invariants(value.State, namespace)
	Pass_Invariants(value.Pass, namespace)
	Index_Invariants(value.Line_Offset, namespace)
	Index_Invariants(value.Line_End, namespace)
	Index_Invariants(value.Line_Number, namespace)
	Index_Invariants(value.Line_Check, namespace)
	Flag_Invariants(value.Seen_Heading, namespace)
	Index_Invariants(value.Heading_Count, namespace)
	Index_Invariants(value.Heading_Cursor, namespace)
	Index_Invariants(value.Parent, namespace)
	Flag_Invariants(value.Has_Child, namespace)
	Index_Invariants(value.Leaf_Count, namespace)
	Index_Invariants(value.Open, namespace)
	Index_Invariants(value.Body_Count, namespace)
	Flag_Invariants(value.Blank, namespace)
	Index_Invariants(value.Leaf_Cursor, namespace)
	Index_Invariants(value.Declaration_Count, namespace)
	Index_Invariants(value.Scan_Start, namespace)
	Index_Invariants(value.Scan_End, namespace)
	Flag_Invariants(value.Scan_Result, namespace)
	Index_Invariants(value.Scan_Position, namespace)
	utf8.Decoded_Character_Invariants(value.Character, namespace)
	Flag_Invariants(value.Character_Space, namespace)
	Flag_Invariants(value.Character_Word, namespace)
	ucd.Character_Invariants(value.Upper, namespace)
	ast.Node_Invariants(value.Node, namespace)
	ast.Index_Invariants(value.Name_Token, namespace)
	Index_Invariants(value.Next_Open, namespace)
	Index_Invariants(value.Finding_Heading, namespace)
	Text_Invariants(value.Text_Before, namespace)
	Text_Invariants(value.Text_After, namespace)
}

// Checker_Fields_Stored keeps one check step from proving every table length it never touched.
type Checker_Fields_Stored interface{}

// Checker_Fields_Stored_Invariants fixes check state representation.
func Checker_Fields_Stored_Invariants(value Checker_Fields_Stored, _ aver.Namespace) {
	_, valid := value.(Checker_Fields)
	aver.Always(valid == (value != nil), "Check fields have expected storage type.")
}

// Checker_Envelope gives check fields one embedded owner.
type Checker_Envelope struct {
	Checker_Fields
}

// Checker_Envelope_Invariants composes check fields.
func Checker_Envelope_Invariants(value Checker_Envelope, namespace aver.Namespace) {
	Checker_Fields_Invariants(value.Checker_Fields, namespace)
}

// Checker is one package's check.
type Checker Checker_Envelope

// Checker_Invariants proves check field representation.
func Checker_Invariants(value Checker, namespace aver.Namespace) {
	Checker_Fields_Stored_Invariants(Checker_Fields_Stored(value.Checker_Fields), namespace)
}

// Checker_Handle is caller-owned check state.
type Checker_Handle *Checker

// Checker_Handle_Invariants composes present check state.
func Checker_Handle_Invariants(value Checker_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Checker_Invariants(*value, namespace)
}

// Begin starts the check of the package the inputs describe.
func Begin(checker Checker_Handle) {
	Checker_Handle_Invariants(checker, "begin.checker")
	checker.Pass = PASS_PRESENCE
	checker.State = STATE_STEP
	checker.Heading_Count = 0
	checker.Leaf_Count = 0
	checker.Declaration_Count = 0
}

// Next advances to the next finding, to a request for requirements_test.go, or to the end.
func Next(checker Checker_Handle) {
	Checker_Handle_Invariants(checker, "next.checker")
	checker.State = STATE_STEP
	for range STEP_COUNT_MAXIMUM {
		if checker.State != STATE_STEP {
			return
		}
		switch checker.Pass {
		case PASS_PRESENCE:
			step_presence(checker)
		case PASS_HEADINGS:
			step_heading(checker)
		case PASS_TREE:
			step_tree(checker)
		case PASS_BODIES:
			step_body(checker)
		case PASS_TEST_FILE:
			step_test_file(checker)
		case PASS_TEST_PARSE:
			step_test_parse(checker)
		case PASS_TESTS:
			step_test(checker)
		case PASS_DONE:
			checker.State = STATE_DONE
		}
	}
}

// A package without a contract owes one, unless it is package main or vendored or example code.
// A contract past its storage stops the check, because a cut file would report faults it lacks.
func step_presence(checker Checker_Handle) {
	Checker_Handle_Invariants(checker, "step_presence.checker")
	if checker.Has_Markdown {
		if int(checker.Markdown_Count) == len(checker.Markdown) {
			checker.Pass = PASS_DONE
			checker.File = FILE_MARKDOWN
			checker.Line = 0
			checker.Text_Before = MESSAGE_MARKDOWN_OVERSIZED
			finding_text(checker)
			return
		}
		checker.Pass = PASS_HEADINGS
		checker.Line_Offset = 0
		checker.Line_Number = 1
		checker.Line_Check = 0
		checker.Seen_Heading = false
		return
	}
	checker.Pass = PASS_DONE
	if checker.Impure {
		return
	}
	scan_exempt(checker)
	if checker.Scan_Result {
		return
	}
	checker.File = FILE_MARKDOWN
	checker.Line = 0
	checker.Text_Before = MESSAGE_MARKDOWN_ABSENT
	finding_text(checker)
}

// Records whether any segment of Path is third_party or examples.
func scan_exempt(checker Checker_Handle) {
	Checker_Handle_Invariants(checker, "scan_exempt.checker")
	checker.Scan_Result = false
	path := checker.Path
	start := 0
	for position_index := 0; position_index < len(path)+1; position_index++ {
		if position_index < len(path) {
			if path[position_index] != '/' {
				continue
			}
		}
		segment := string(path[start:position_index])
		if segment == "third_party" {
			checker.Scan_Result = true
		}
		if segment == "examples" {
			checker.Scan_Result = true
		}
		start = position_index + 1
	}
}

// Scans one line of the file in up to four checks, stopping at each finding.
func step_heading(checker Checker_Handle) {
	Checker_Handle_Invariants(checker, "step_heading.checker")
	markdown := checker.Markdown[:checker.Markdown_Count]
	checker.Finding_Heading = checker.Heading_Count - 1
	switch checker.Line_Check {
	case 0:
		line_measure(checker)
		heading_collect(checker)
	case 1:
		checker.Line_Check = 2
		checker.Scan_Start = checker.Headings[checker.Finding_Heading].Start
		checker.Scan_End = checker.Headings[checker.Finding_Heading].End
		scan_words_invalid(checker)
		if checker.Scan_Result {
			checker.Text_Before = `The heading "`
			checker.Text_After = `" has a character that is not a letter or a digit. ` +
				"Write only letters and digits."
			finding_quote(checker)
		}
	case 2:
		checker.Line_Check = 3
		preceded := checker.Line_Number > 1
		if preceded {
			preceded = checker.Line_Offset == 1
			if !preceded {
				preceded = markdown[checker.Line_Offset-2] == '\n'
			}
		}
		if !preceded {
			checker.Text_Before = `There is no blank line before the heading "`
			checker.Text_After = `". Write a blank line.`
			finding_quote(checker)
		}
	case 3:
		followed := int(checker.Line_End) < len(markdown)
		if followed {
			followed = int(checker.Line_End)+1 == len(markdown)
			if !followed {
				followed = markdown[checker.Line_End+1] == '\n'
			}
		}
		line_advance(checker)
		if !followed {
			checker.Text_Before = `There is no blank line after the heading "`
			checker.Text_After = `". Write a blank line.`
			finding_quote(checker)
		}
	}
}

// Reads the scanned line's heading, or flags what stands on a line that is none.
func heading_collect(checker Checker_Handle) {
	Checker_Handle_Invariants(checker, "heading_collect.checker")
	markdown := checker.Markdown[:checker.Markdown_Count]
	line := string(markdown[checker.Line_Offset:checker.Line_End])
	level := Index(0)
	start := checker.Line_Offset
	if len(line) > len("### ")-1 {
		if line[:len("### ")] == "### " {
			level = LEVEL_LEAF
			start += Index(len("### "))
		}
	}
	if level == 0 {
		if len(line) > len("# ")-1 {
			if line[:len("# ")] == "# " {
				level = LEVEL_BRANCH
				start += Index(len("# "))
			}
		}
	}
	if level == 0 {
		heading_absent(checker)
		return
	}
	checker.File = FILE_MARKDOWN
	checker.Line = checker.Line_Number
	if int(checker.Line_End-start) > HEADING_SIZE_MAXIMUM {
		checker.Pass = PASS_DONE
		checker.Text_Before = MESSAGE_HEADING_OVERSIZED
		finding_text(checker)
		return
	}
	if int(checker.Heading_Count) == len(checker.Headings) {
		checker.Pass = PASS_DONE
		checker.Text_Before = MESSAGE_HEADINGS_FULL
		finding_text(checker)
		return
	}
	checker.Headings[checker.Heading_Count] = Heading{
		Level: level,
		Line:  checker.Line_Number,
		Start: start,
		End:   checker.Line_End,
	}
	checker.Heading_Count++
	checker.Seen_Heading = true
	checker.Line_Check = 1
}

// A line that opens with # but no heading marker is a wrong level; any other content before the
// first heading is a preamble.
func heading_absent(checker Checker_Handle) {
	Checker_Handle_Invariants(checker, "heading_absent.checker")
	markdown := checker.Markdown[:checker.Markdown_Count]
	marked := checker.Line_Offset < checker.Line_End
	if marked {
		marked = markdown[checker.Line_Offset] == '#'
	}
	checker.File = FILE_MARKDOWN
	checker.Line = checker.Line_Number
	checker.Scan_Start = checker.Line_Offset
	checker.Scan_End = checker.Line_End
	line_advance(checker)
	if marked {
		checker.Text_Before = MESSAGE_HEADING_LEVEL
		finding_text(checker)
		return
	}
	scan_blank(checker)
	if checker.Scan_Result {
		return
	}
	if checker.Seen_Heading {
		return
	}
	checker.Text_Before = MESSAGE_PREAMBLE
	finding_text(checker)
}

// Walks the headings into leaves: a # with no ### child is a leaf, and so is each ###.
func step_tree(checker Checker_Handle) {
	Checker_Handle_Invariants(checker, "step_tree.checker")
	if checker.Heading_Cursor == checker.Heading_Count {
		tree_close(checker)
		checker.Pass = PASS_BODIES
		checker.Line_Offset = 0
		checker.Line_Number = 1
		checker.Line_Check = 0
		checker.Open = ABSENT
		checker.Body_Count = 0
		checker.Blank = false
		checker.Heading_Cursor = 0
		return
	}
	cursor := checker.Heading_Cursor
	heading := checker.Headings[cursor]
	checker.Heading_Cursor++
	checker.Finding_Heading = cursor
	if heading.Level == LEVEL_LEAF {
		if checker.Parent == ABSENT {
			checker.Text_Before = `The "###" heading "`
			checker.Text_After = `" has no parent "#" heading. ` +
				`Put the heading below a "#" heading.`
			finding_quote(checker)
			return
		}
		checker.Has_Child = true
		checker.Leaves[checker.Leaf_Count] = Leaf{Parent: checker.Parent, Heading: cursor}
		checker.Leaf_Count++
		checker.Headings[cursor].Leaf = true
		checker.Scan_Start = checker.Parent + 1
	} else {
		tree_close(checker)
		checker.Parent = cursor
		checker.Has_Child = false
		checker.Scan_Start = 0
	}
	checker.Scan_End = cursor
	scan_duplicate(checker)
	if checker.Scan_Result {
		checker.Text_Before = `The heading "`
		checker.Text_After = `" is not unique. Write a different heading.`
		finding_quote(checker)
	}
}

// Records the open # as a leaf when it gained no ### child.
func tree_close(checker Checker_Handle) {
	Checker_Handle_Invariants(checker, "tree_close.checker")
	if checker.Parent == ABSENT {
		return
	}
	if checker.Has_Child {
		return
	}
	checker.Leaves[checker.Leaf_Count] = Leaf{Parent: ABSENT, Heading: checker.Parent}
	checker.Leaf_Count++
	checker.Headings[checker.Parent].Leaf = true
}

// Records whether a heading in [Scan_Start, Scan_End) of the same level shares the text of the
// heading at Scan_End.
func scan_duplicate(checker Checker_Handle) {
	Checker_Handle_Invariants(checker, "scan_duplicate.checker")
	markdown := checker.Markdown[:checker.Markdown_Count]
	subject := checker.Headings[checker.Scan_End]
	text := string(markdown[subject.Start:subject.End])
	checker.Scan_Result = false
	for index := checker.Scan_Start; index < checker.Scan_End; index++ {
		other := checker.Headings[index]
		if other.Level != subject.Level {
			continue
		}
		if string(markdown[other.Start:other.End]) == text {
			checker.Scan_Result = true
		}
	}
}

// Scans one line of the file for its section's body, stopping at each finding.
func step_body(checker Checker_Handle) {
	Checker_Handle_Invariants(checker, "step_body.checker")
	if checker.Line_Check == 1 {
		body_line(checker)
		return
	}
	line_measure(checker)
	markdown := checker.Markdown[:checker.Markdown_Count]
	cursor := checker.Heading_Cursor
	if cursor < checker.Heading_Count {
		if checker.Headings[cursor].Line == checker.Line_Number {
			checker.Heading_Cursor++
			checker.Next_Open = cursor
			body_close(checker)
			return
		}
	}
	marked := checker.Line_Offset < checker.Line_End
	if marked {
		marked = markdown[checker.Line_Offset] == '#'
	}
	if marked {
		checker.Next_Open = ABSENT
		body_close(checker)
		return
	}
	checker.Scan_Start = checker.Line_Offset
	checker.Scan_End = checker.Line_End
	scan_blank(checker)
	if checker.Scan_Result {
		if checker.Body_Count > 0 {
			checker.Blank = true
		}
		line_advance(checker)
		return
	}
	if checker.Open == ABSENT {
		line_advance(checker)
		return
	}
	checker.Line_Check = 1
	if checker.Blank {
		checker.Blank = false
		checker.Finding_Heading = checker.Open
		checker.Text_Before = `The section "`
		checker.Text_After = `" has a blank line between two body lines. ` +
			"Remove the blank line."
		finding_quote(checker)
		checker.Line = checker.Line_Number
	}
}

// Counts one body line of the open section, flagging the line past its bound.
func body_line(checker Checker_Handle) {
	Checker_Handle_Invariants(checker, "body_line.checker")
	checker.Body_Count++
	number := checker.Line_Number
	line_advance(checker)
	if checker.Body_Count != SECTION_BODY_MAXIMUM+1 {
		return
	}
	checker.Finding_Heading = checker.Open
	checker.Text_Before = `The section "`
	checker.Text_After = `" has more than three lines. Write a maximum of three lines.`
	finding_quote(checker)
	checker.Line = number
}

// Closes the open section, flagging a leaf with no body, then opens Next_Open.
func body_close(checker Checker_Handle) {
	Checker_Handle_Invariants(checker, "body_close.checker")
	closed := checker.Open
	empty := checker.Body_Count == 0
	checker.Open = checker.Next_Open
	checker.Body_Count = 0
	checker.Blank = false
	line_advance(checker)
	if closed == ABSENT {
		return
	}
	if !checker.Headings[closed].Leaf {
		return
	}
	if !empty {
		return
	}
	checker.Finding_Heading = closed
	checker.Text_Before = `The section "`
	checker.Text_After = `" has no body line. Write a body line.`
	finding_quote(checker)
}

// Asks for requirements_test.go, or flags its absence.
func step_test_file(checker Checker_Handle) {
	Checker_Handle_Invariants(checker, "step_test_file.checker")
	if !checker.Has_Test {
		checker.Pass = PASS_DONE
		checker.File = FILE_TEST
		checker.Line = 0
		checker.Text_Before = MESSAGE_TEST_ABSENT
		finding_text(checker)
		return
	}
	checker.Pass = PASS_TEST_PARSE
	checker.State = STATE_TEST
}

// Collects the leading declarations of Test_Source. A source the parser refuses has its own
// diagnostic, thus its tests go unchecked rather than flagged twice.
func step_test_parse(checker Checker_Handle) {
	Checker_Handle_Invariants(checker, "step_test_parse.checker")
	checker.Pass = PASS_DONE
	parsed := ast.Parse(checker.Parse_State, checker.Test_Source)
	if !bool(parsed.OK) {
		return
	}
	indexed := token.Index_Lines(checker.Lines, checker.Test_Source)
	root := ast.Node_At(checker.Parse_State, ast.Index(parsed.Root))
	child := ast.Index(root.First_Child)
	for range ast.NODE_COUNT_MAXIMUM {
		if child == ast.INDEX_ABSENT {
			break
		}
		if int(checker.Declaration_Count) == len(checker.Declarations) {
			break
		}
		checker.Node = ast.Node_At(checker.Parse_State, child)
		child = ast.Index(checker.Node.Next)
		declaration_name(checker)
		if checker.Name_Token == ast.INDEX_ABSENT {
			continue
		}
		named := ast.Token_At(checker.Parse_State, ast.Token_Index(checker.Name_Token))
		line := Index(0)
		if indexed {
			line = Index(token.Position_Of(checker.Lines, named.Offset).Line)
		}
		checker.Declarations[checker.Declaration_Count] = Declaration{
			Start: Index(named.Offset),
			End:   Index(int(named.Offset) + int(named.Size)),
			Line:  line,
		}
		checker.Declaration_Count++
	}
	checker.Pass = PASS_TESTS
	checker.Leaf_Cursor = 0
}

// Records in Name_Token the token that names the top-level declaration Node, or ast.INDEX_ABSENT
// for what is no declaration: the package name, an import, a comment, or a blank line.
func declaration_name(checker Checker_Handle) {
	Checker_Handle_Invariants(checker, "declaration_name.checker")
	checker.Name_Token = ast.INDEX_ABSENT
	switch checker.Node.Kind {
	case ast.NODE_CONSTANT, ast.NODE_VARIABLE, ast.NODE_TYPE:
		checker.Name_Token = ast.Index(checker.Node.Token)
	case ast.NODE_FUNCTION:
		// A method's receiver comes first, thus the name is the first identifier, not the
		// first child.
		child := ast.Index(checker.Node.First_Child)
		for range ast.NODE_COUNT_MAXIMUM {
			if child == ast.INDEX_ABSENT {
				return
			}
			part := ast.Node_At(checker.Parse_State, child)
			if part.Kind == ast.NODE_IDENTIFIER {
				checker.Name_Token = ast.Index(part.Token)
				return
			}
			child = ast.Index(part.Next)
		}
	}
}

// Compares one leaf's test name with the declaration in its slot.
func step_test(checker Checker_Handle) {
	Checker_Handle_Invariants(checker, "step_test.checker")
	if checker.Leaf_Cursor == checker.Leaf_Count {
		checker.Pass = PASS_DONE
		return
	}
	name_write(checker)
	slot := checker.Leaf_Cursor
	checker.Leaf_Cursor++
	checker.File = FILE_TEST
	checker.Line = 0
	if slot < checker.Declaration_Count {
		declaration := checker.Declarations[slot]
		checker.Line = declaration.Line
		name := string(checker.Test_Source[declaration.Start:declaration.End])
		if name == string(checker.Name_Storage) {
			return
		}
	}
	checker.Text_Before = "The file needs "
	finding_text(checker)
	checker.Message = append(checker.Message, checker.Name_Storage...)
	checker.Message = append(checker.Message, " for the leaf \""...)
	checker.Message = append(checker.Message, checker.Name_Storage[len(TEST_PREFIX):]...)
	checker.Message = append(
		checker.Message,
		"\". Declare the tests in leaf order at the top of the file."...,
	)
}

// Writes the test name of the leaf at Leaf_Cursor into Name_Storage.
func name_write(checker Checker_Handle) {
	Checker_Handle_Invariants(checker, "name_write.checker")
	leaf := checker.Leaves[checker.Leaf_Cursor]
	checker.Name_Storage = append(checker.Name_Storage[:0], TEST_PREFIX...)
	if leaf.Parent != ABSENT {
		checker.Scan_Start = checker.Headings[leaf.Parent].Start
		checker.Scan_End = checker.Headings[leaf.Parent].End
		name_ada(checker)
		checker.Name_Storage = append(checker.Name_Storage, '_')
	}
	checker.Scan_Start = checker.Headings[leaf.Heading].Start
	checker.Scan_End = checker.Headings[leaf.Heading].End
	name_ada(checker)
}

// Appends the Ada_Case form of [Scan_Start, Scan_End): each word's first rune upper cased, and the
// words joined by underscores.
func name_ada(checker Checker_Handle) {
	Checker_Handle_Invariants(checker, "name_ada.checker")
	markdown := checker.Markdown[:checker.Markdown_Count]
	word_open := false
	written := false
	checker.Scan_Position = checker.Scan_Start
	for range HEADING_SIZE_MAXIMUM {
		if checker.Scan_Position == checker.Scan_End {
			return
		}
		start := checker.Scan_Position
		character_read(checker)
		if checker.Character_Space {
			word_open = false
			continue
		}
		if word_open {
			rest := markdown[start:checker.Scan_Position]
			checker.Name_Storage = append(checker.Name_Storage, rest...)
			continue
		}
		if written {
			checker.Name_Storage = append(checker.Name_Storage, '_')
		}
		word_open = true
		written = true
		character_upper(checker)
		var encoded [utf8.CHARACTER_SIZE_MAXIMUM]byte
		upper := utf8.Character(checker.Upper)
		count := utf8.Encode_Character(utf8.Bytes(encoded[:]), upper)
		checker.Name_Storage = append(checker.Name_Storage, encoded[:count]...)
	}
}

// Records whether [Scan_Start, Scan_End) holds a word rune that is neither a letter nor a digit,
// which survives into the test name and makes it no identifier.
func scan_words_invalid(checker Checker_Handle) {
	Checker_Handle_Invariants(checker, "scan_words_invalid.checker")
	invalid := false
	checker.Scan_Position = checker.Scan_Start
	for range HEADING_SIZE_MAXIMUM {
		if checker.Scan_Position == checker.Scan_End {
			break
		}
		character_read(checker)
		if checker.Character_Space {
			continue
		}
		if !checker.Character_Word {
			invalid = true
		}
	}
	checker.Scan_Result = Flag(invalid)
}

// Records whether [Scan_Start, Scan_End) holds white space alone.
func scan_blank(checker Checker_Handle) {
	Checker_Handle_Invariants(checker, "scan_blank.checker")
	blank := true
	checker.Scan_Position = checker.Scan_Start
	for range MARKDOWN_STORAGE_SIZE {
		if checker.Scan_Position == checker.Scan_End {
			break
		}
		character_read(checker)
		if !checker.Character_Space {
			blank = false
		}
	}
	checker.Scan_Result = Flag(blank)
}

// Decodes the rune at Scan_Position and classifies it. Headings use ASCII almost always, thus the
// Unicode tables stay the fallback.
func character_read(checker Checker_Handle) {
	Checker_Handle_Invariants(checker, "character_read.checker")
	markdown := checker.Markdown[checker.Scan_Position:checker.Scan_End]
	character, size := utf8.Decode_Character(utf8.Bytes(markdown))
	checker.Character = character
	checker.Scan_Position += Index(size)
	checker.Character_Space = false
	checker.Character_Word = false
	if character > 0x7f {
		checker.Character_Space = Flag(ucd.Is_Space(ucd.Character(character)))
		checker.Character_Word = Flag(ucd.Is_Letter(ucd.Character(character)))
		if !checker.Character_Word {
			checker.Character_Word = Flag(ucd.Is_Digit(ucd.Character(character)))
		}
		return
	}
	switch character {
	case ' ', '\t', '\n', '\v', '\f', '\r':
		checker.Character_Space = true
	}
	// Unsigned, so a rune below the range wraps past it rather than passing as negative.
	if uint32(character-'a') < 26 {
		checker.Character_Word = true
	}
	if uint32(character-'A') < 26 {
		checker.Character_Word = true
	}
	if uint32(character-'0') < 10 {
		checker.Character_Word = true
	}
}

// Records in Upper the upper case of Character.
func character_upper(checker Checker_Handle) {
	Checker_Handle_Invariants(checker, "character_upper.checker")
	character := checker.Character
	checker.Upper = ucd.Character(character)
	if uint32(character-'a') < 26 {
		checker.Upper = ucd.Character(character - ('a' - 'A'))
		return
	}
	if character > 0x7f {
		checker.Upper = ucd.To_Upper(ucd.Character(character))
	}
}

// Finds where the scanned line closes.
func line_measure(checker Checker_Handle) {
	Checker_Handle_Invariants(checker, "line_measure.checker")
	markdown := checker.Markdown[:checker.Markdown_Count]
	checker.Line_End = Index(len(markdown))
	for position := int(checker.Line_Offset); position < len(markdown); position++ {
		if markdown[position] == '\n' {
			checker.Line_End = Index(position)
			break
		}
	}
}

// Moves to the next line, or to the next pass past the last. The text after the final newline
// counts as a line, empty or not.
func line_advance(checker Checker_Handle) {
	Checker_Handle_Invariants(checker, "line_advance.checker")
	checker.Line_Check = 0
	if int(checker.Line_End) == int(checker.Markdown_Count) {
		switch checker.Pass {
		case PASS_HEADINGS:
			checker.Pass = PASS_TREE
			checker.Heading_Cursor = 0
			checker.Parent = ABSENT
			checker.Has_Child = false
		case PASS_BODIES:
			checker.Pass = PASS_TEST_FILE
		}
		return
	}
	checker.Line_Offset = checker.Line_End + 1
	checker.Line_Number++
}

// Opens a finding at File and Line whose message is Text_Before.
func finding_text(checker Checker_Handle) {
	Checker_Handle_Invariants(checker, "finding_text.checker")
	checker.State = STATE_FINDING
	checker.Message = append(checker.Message_Storage[:0], checker.Text_Before...)
}

// Opens a finding on the line of Finding_Heading whose message quotes its text between
// Text_Before and Text_After.
func finding_quote(checker Checker_Handle) {
	Checker_Handle_Invariants(checker, "finding_quote.checker")
	subject := checker.Headings[checker.Finding_Heading]
	markdown := checker.Markdown[:checker.Markdown_Count]
	checker.File = FILE_MARKDOWN
	checker.Line = subject.Line
	finding_text(checker)
	checker.Message = append(checker.Message, markdown[subject.Start:subject.End]...)
	checker.Message = append(checker.Message, checker.Text_After...)
}
