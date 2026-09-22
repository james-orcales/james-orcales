// Package requirements checks one package's contract: that it carries REQUIREMENTS.md, that the
// markdown obeys the doctrine, and that requirements_test.go mirrors the leaf headings. It reads
// no file. The caller loads each file into caller-owned storage and resumes the check, which stops
// at every finding so the caller can report it before the next.
//
// Every store the check holds is one fixed length, and every cursor it keeps is one slot of the
// number or flag table. A step thus proves the storage it was handed, never the state of a pass it
// never ran.
package requirements

import "local/james-orcales/shared/go/ast"
import "local/james-orcales/shared/go/token"
import "local/james-orcales/shared/math/bits"
import "local/james-orcales/shared/path"
import "local/james-orcales/shared/sim/aver/default"
import "local/james-orcales/shared/unicode/ucd"
import "local/james-orcales/shared/unicode/utf8"

// MARKDOWN_NAME is the exact-cased contract file; a differently cased file on a case-insensitive
// disk is no contract.
const MARKDOWN_NAME = "REQUIREMENTS.md"

// TEST_NAME is the exact-cased test file that mirrors the contract.
const TEST_NAME = "requirements_test.go"

// HEADING_SIZE_MAXIMUM bounds one heading's text, so every message quoting it stays bounded.
const HEADING_SIZE_MAXIMUM = 128

// HEADING_COUNT_MAXIMUM bounds the headings one REQUIREMENTS.md holds. The widest contract this
// repository keeps states 116, thus the table holds twice that with room to grow.
const HEADING_COUNT_MAXIMUM = 256

// MARKDOWN_STORAGE_SIZE holds one byte past the largest REQUIREMENTS.md, so a full read proves a
// file oversized without a second read. The widest contract this repository keeps spends 20
// kibibytes, thus the store holds past that with room to grow.
const MARKDOWN_STORAGE_SIZE = 32*bits.KIBIBYTE_BYTES + 1

// SOURCE_SIZE_MAXIMUM is the widest Go source one check reads. The widest source this repository
// keeps spends 423 kibibytes, thus the bound stands past that with room to grow, and inside what
// one parse admits.
const SOURCE_SIZE_MAXIMUM = 512 * bits.KIBIBYTE_BYTES

// TEST_STORAGE_SIZE holds one byte past the largest Go source, as the caller's own source store
// does; the caller hands this check that same store.
const TEST_STORAGE_SIZE = SOURCE_SIZE_MAXIMUM + 1

// PATH_STORAGE_SIZE holds the widest package directory, which is one whole filesystem path.
const PATH_STORAGE_SIZE = path.PATH_SIZE_MAXIMUM

// UPPER_GROWTH_MAXIMUM is the most bytes an upper case mapping adds to one character: no mapping
// widens a character by more than one byte, and no ASCII character widens at all.
const UPPER_GROWTH_MAXIMUM = 1

// GROWN_WORD_SIZE_MINIMUM is the fewest bytes one heading spends on a word whose upper case is
// wider: a character that widens is no ASCII one, thus two bytes, and one space closes the word.
const GROWN_WORD_SIZE_MINIMUM = 2 + len(" ")

// ADA_SIZE_MAXIMUM is the widest part of a name one heading builds: the heading, and the one byte
// each widening word adds, where the last word spends no closing space.
const ADA_SIZE_MAXIMUM = HEADING_SIZE_MAXIMUM +
  UPPER_GROWTH_MAXIMUM*(HEADING_SIZE_MAXIMUM+len(" "))/GROWN_WORD_SIZE_MINIMUM

// NAME_SIZE_MAXIMUM holds one test name: its prefix and two headings joined by an underscore.
const NAME_SIZE_MAXIMUM = len(TEST_PREFIX) + 2*ADA_SIZE_MAXIMUM + len("_")

// MESSAGE_TEXT_TEST_ORDER opens the message of a test declared out of leaf order.
const MESSAGE_TEXT_TEST_ORDER = "The file needs "

// MESSAGE_TEXT_LEAF stands between the test name and the leaf it answers.
const MESSAGE_TEXT_LEAF = " for the leaf \""

// MESSAGE_TEXT_TEST_ORDER_TAIL closes the message of a test declared out of leaf order.
const MESSAGE_TEXT_TEST_ORDER_TAIL = "\". Declare the tests in leaf order at the top of the file."

// MESSAGE_TEXT_BLANK_BEFORE opens the message of a heading with no blank line before it, which is
// the widest opening a message that quotes carries.
const MESSAGE_TEXT_BLANK_BEFORE = `There is no blank line before the heading "`

// MESSAGE_SIZE_MAXIMUM holds the longest message: the test order finding names a test whole and
// then its leaf, and every other message quotes one heading, which spends less.
const MESSAGE_SIZE_MAXIMUM = len(MESSAGE_TEXT_TEST_ORDER) +
  NAME_SIZE_MAXIMUM +
  len(MESSAGE_TEXT_LEAF) +
  NAME_SIZE_MAXIMUM -
  len(TEST_PREFIX) +
  len(MESSAGE_TEXT_TEST_ORDER_TAIL)

// STEP_COUNT_MAXIMUM bounds the steps one Next takes: each line of the file costs a few steps in
// each pass, and each heading and leaf one step.
const STEP_COUNT_MAXIMUM = 16 * MARKDOWN_STORAGE_SIZE

// TEST_PREFIX opens the name of every leaf's test.
const TEST_PREFIX = "Test_"

// MARKDOWN_OFFSET_MAXIMUM is the furthest offset a recorded heading names. A file that fills the
// store is oversized and the check stops before it records a heading, thus the widest file a
// heading stands in is one byte short of the store.
const MARKDOWN_OFFSET_MAXIMUM = MARKDOWN_STORAGE_SIZE - 1

// HEADING_MARKER_SIZE is the smallest marker a heading carries before its text: one # and the
// space that closes it.
const HEADING_MARKER_SIZE = len("# ")

// PACKAGE_CLAUSE_SIZE_MINIMUM is the smallest package clause a Go file opens with: the keyword,
// the space after it, one letter of name, and the ending that closes the line.
const PACKAGE_CLAUSE_SIZE_MINIMUM = len("package a\n")

// DECLARATION_START_MINIMUM is the first offset a declaration's span stands at, since a package
// clause stands before every declaration of a file.
const DECLARATION_START_MINIMUM = PACKAGE_CLAUSE_SIZE_MINIMUM

// DECLARATION_END_MINIMUM is the first offset a declaration's span closes at. A span at the
// first offset opens on a keyword rather than a name, since a name needs the keyword before it,
// thus the shortest keyword sets how far that span reaches.
const DECLARATION_END_MINIMUM = DECLARATION_START_MINIMUM + len("var")

// LINE_ABSENT names no line: a finding the check places in a file rather than on a line of it.
const LINE_ABSENT = 0

// LINE_NUMBER_MAXIMUM_TEST is the last line a position states, since a source of more lines than
// the index holds states no position at all.
const LINE_NUMBER_MAXIMUM_TEST = token.LINE_MAXIMUM

// LINE_NUMBER_MINIMUM is the first line of a file, since a scan counts lines from one.
const LINE_NUMBER_MINIMUM = 1

// LINE_NUMBER_MAXIMUM is the most lines a markdown store holds, since one line spends one byte
// on the ending that closes it.
const LINE_NUMBER_MAXIMUM = MARKDOWN_STORAGE_SIZE - 1

// ABSENT names no heading.
const ABSENT Number = -1

// NUMBER_MINIMUM is ABSENT, the smallest number one slot holds.
const NUMBER_MINIMUM = int(ABSENT)

// NUMBER_MAXIMUM is one past the widest bounded input, a Go source.
const NUMBER_MAXIMUM = SOURCE_SIZE_MAXIMUM + 1

// LEVEL_BRANCH is a # heading.
const LEVEL_BRANCH Number = 1

// LEVEL_LEAF is a ### heading.
const LEVEL_LEAF Number = 3

// File names which of a package's two contract files a finding stands in.
type File Number

// File_Invariants admits each contract file.
func File_Invariants(value File, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_Int(int(value), int(FILE_MARKDOWN), int(FILE_TEST)).
    Ensure()
}

// FILE_MARKDOWN is a finding in REQUIREMENTS.md.
const FILE_MARKDOWN File = 0

// FILE_TEST is a finding in requirements_test.go.
const FILE_TEST File = 1

// STATE_STEP is a check between findings.
const STATE_STEP Number = 0

// STATE_FINDING is a finding in the message store, the file slot, and the line slot.
const STATE_FINDING Number = 1

// STATE_TEST asks the caller for requirements_test.go in the test store before Next.
const STATE_TEST Number = 2

// STATE_DONE is a finished check.
const STATE_DONE Number = 3

// PASS_PRESENCE decides whether the package owes a REQUIREMENTS.md.
const PASS_PRESENCE Number = 0

// PASS_HEADINGS collects headings and flags line-local faults.
const PASS_HEADINGS Number = 1

// PASS_TREE decides leaves and flags orphan and repeated headings.
const PASS_TREE Number = 2

// PASS_NAMES flags a leaf whose test name an earlier leaf already states.
const PASS_NAMES Number = 3

// PASS_BODIES flags empty, oversized, and gapped sections.
const PASS_BODIES Number = 4

// PASS_TEST_FILE asks for requirements_test.go, or flags its absence.
const PASS_TEST_FILE Number = 5

// PASS_TEST_PARSE collects the leading declarations of requirements_test.go.
const PASS_TEST_PARSE Number = 6

// PASS_TESTS flags each leading declaration that is not its leaf's test.
const PASS_TESTS Number = 7

// PASS_DONE ends the check.
const PASS_DONE Number = 8

// Heading_Message names a finding text that quotes the heading it stands on.
type Heading_Message Message

// Heading_Message_Invariants admits each message that quotes a heading.
func Heading_Message_Invariants(value Heading_Message, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(
    int(value),
    int(MESSAGE_HEADING_CHARACTERS),
    int(MESSAGE_HEADING_CASE),
  ).
    Ensure()
}

// Message names which finding text a run writes.
type Message Number

// Message_Invariants admits each message a finding carries.
func Message_Invariants(value Message, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(
    int(value),
    int(MESSAGE_MARKDOWN_ABSENT),
    int(MESSAGE_CONFLICT_MARKER),
  ).
    Ensure()
}

// MESSAGE_MARKDOWN_ABSENT flags a package that owes a contract.
const MESSAGE_MARKDOWN_ABSENT Message = 0

// MESSAGE_MARKDOWN_OVERSIZED flags a contract past its storage.
const MESSAGE_MARKDOWN_OVERSIZED Message = 1

// MESSAGE_HEADING_OVERSIZED flags a heading past HEADING_SIZE_MAXIMUM.
const MESSAGE_HEADING_OVERSIZED Message = 2

// MESSAGE_HEADINGS_FULL flags a contract past its heading table.
const MESSAGE_HEADINGS_FULL Message = 3

// MESSAGE_HEADING_LEVEL flags a heading neither # nor ###.
const MESSAGE_HEADING_LEVEL Message = 4

// MESSAGE_PREAMBLE flags content before the first heading.
const MESSAGE_PREAMBLE Message = 5

// MESSAGE_TEST_ABSENT flags a contract without its test file.
const MESSAGE_TEST_ABSENT Message = 6

// MESSAGE_HEADING_CHARACTERS flags a heading word that is no letter or digit.
const MESSAGE_HEADING_CHARACTERS Message = 7

// MESSAGE_BLANK_BEFORE flags a heading without a blank line above it.
const MESSAGE_BLANK_BEFORE Message = 8

// MESSAGE_BLANK_AFTER flags a heading without a blank line below it.
const MESSAGE_BLANK_AFTER Message = 9

// MESSAGE_ORPHAN flags a ### heading without a # above it.
const MESSAGE_ORPHAN Message = 10

// MESSAGE_DUPLICATE flags a repeated heading.
const MESSAGE_DUPLICATE Message = 11

// MESSAGE_SECTION_EMPTY flags a leaf section without a body line.
const MESSAGE_SECTION_EMPTY Message = 12

// MESSAGE_SECTION_SIZE flags a section past three body lines.
const MESSAGE_SECTION_SIZE Message = 13

// MESSAGE_SECTION_GAP flags a blank line inside a section body.
const MESSAGE_SECTION_GAP Message = 14

// MESSAGE_LEAF_NAME flags a leaf whose test name an earlier leaf already states.
const MESSAGE_LEAF_NAME Message = 15

// MESSAGE_HEADING_CASE flags a heading word carrying a capital after a lower case letter.
const MESSAGE_HEADING_CASE Message = 16

// MESSAGE_HEADINGS_ABSENT flags a present contract that states no heading.
const MESSAGE_HEADINGS_ABSENT Message = 17

// MESSAGE_TEST_ORDER flags a declaration slot that is not its leaf's test.
const MESSAGE_TEST_ORDER Message = 18

// MESSAGE_CONFLICT_MARKER flags a line that opens on a merge conflict marker.
const MESSAGE_CONFLICT_MARKER Message = 19

// NUMBER_STATE holds what Next left for the caller.
const NUMBER_STATE = 0

// NUMBER_PASS holds the check the package stands in.
const NUMBER_PASS = 1

// NUMBER_FILE holds the file a finding stands in.
const NUMBER_FILE = 2

// NUMBER_LINE holds the line a finding stands on, or 0 for the whole file.
const NUMBER_LINE = 3

// NUMBER_MESSAGE holds which message a finding states.
const NUMBER_MESSAGE = 4

// NUMBER_MARKDOWN_COUNT holds the loaded bytes of the markdown store.
const NUMBER_MARKDOWN_COUNT = 5

// NUMBER_TEST_COUNT holds the loaded bytes of the test store.
const NUMBER_TEST_COUNT = 6

// NUMBER_PATH_COUNT holds the bytes of the package directory in the path store.
const NUMBER_PATH_COUNT = 7

// NUMBER_NAME_COUNT holds the bytes of the test name in the name store.
const NUMBER_NAME_COUNT = 8

// NUMBER_MESSAGE_COUNT holds the bytes of the message in the message store.
const NUMBER_MESSAGE_COUNT = 9

// NUMBER_LINE_OFFSET holds where the scanned line opens.
const NUMBER_LINE_OFFSET = 10

// NUMBER_LINE_END holds where the scanned line closes.
const NUMBER_LINE_END = 11

// NUMBER_LINE_NUMBER holds the scanned line's number.
const NUMBER_LINE_NUMBER = 12

// NUMBER_LINE_CHECK holds the next check of the scanned line.
const NUMBER_LINE_CHECK = 13

// NUMBER_HEADING_COUNT holds the headings collected.
const NUMBER_HEADING_COUNT = 14

// NUMBER_HEADING_CURSOR holds the next heading a pass visits.
const NUMBER_HEADING_CURSOR = 15

// NUMBER_PARENT holds the open # heading, or ABSENT.
const NUMBER_PARENT = 16

// NUMBER_LEAF_COUNT holds the leaves collected.
const NUMBER_LEAF_COUNT = 17

// NUMBER_LEAF_CURSOR holds the next leaf whose test the check compares.
const NUMBER_LEAF_CURSOR = 18

// NUMBER_OPEN holds the section the scanned line belongs to, or ABSENT.
const NUMBER_OPEN = 19

// NUMBER_NEXT_OPEN holds the section a closing section hands over to.
const NUMBER_NEXT_OPEN = 20

// NUMBER_BODY_COUNT holds the open section's body lines.
const NUMBER_BODY_COUNT = 21

// NUMBER_DECLARATION_COUNT holds the leading declarations collected.
const NUMBER_DECLARATION_COUNT = 22

// NUMBER_SCAN_START holds where the text a scan reads opens.
const NUMBER_SCAN_START = 23

// NUMBER_SCAN_END holds where it closes.
const NUMBER_SCAN_END = 24

// NUMBER_SCAN_POSITION holds the next rune a scan decodes.
const NUMBER_SCAN_POSITION = 25

// NUMBER_FINDING_HEADING holds the heading a finding quotes.
const NUMBER_FINDING_HEADING = 26

// NUMBER_CHARACTER holds the rune a scan decoded.
const NUMBER_CHARACTER = 27

// NUMBER_UPPER holds its upper case.
const NUMBER_UPPER = 28

// NUMBER_NAME_TOKEN holds the token that names a declaration, or ast.INDEX_ABSENT.
const NUMBER_NAME_TOKEN = 29

// NUMBER_NAME_LEAF holds the leaf whose test name the name store carries.
const NUMBER_NAME_LEAF = 30

// NUMBER_REWRITE_COUNT counts the bytes the rewrite wrote.
const NUMBER_REWRITE_COUNT = 31

// NUMBER_REWRITE_HELD is where the blank lines after a body line open, or ABSENT while none wait.
const NUMBER_REWRITE_HELD = 32

// NUMBER_REWRITE_NEXT is where the line after the scanned one opens.
const NUMBER_REWRITE_NEXT = 33

// NUMBER_SLOT_COUNT is how many numbers one check keeps.
const NUMBER_SLOT_COUNT = 34

// FLAG_HAS_MARKDOWN records an exact-cased REQUIREMENTS.md.
const FLAG_HAS_MARKDOWN = 0

// FLAG_HAS_TEST records an exact-cased requirements_test.go.
const FLAG_HAS_TEST = 1

// FLAG_IMPURE records a package main file, which exempts the package.
const FLAG_IMPURE = 2

// FLAG_SEEN_HEADING records a heading above the scanned line.
const FLAG_SEEN_HEADING = 3

// FLAG_HAS_CHILD records a ### under the open #.
const FLAG_HAS_CHILD = 4

// FLAG_BLANK records a blank line inside the open section's body.
const FLAG_BLANK = 5

// FLAG_SCAN_RESULT records what a scan found.
const FLAG_SCAN_RESULT = 6

// FLAG_CHARACTER_SPACE records white space.
const FLAG_CHARACTER_SPACE = 7

// FLAG_CHARACTER_WORD records a letter or a digit.
const FLAG_CHARACTER_WORD = 8

// FLAG_HEADING_CASE records a heading word carrying a capital after a lower case letter.
const FLAG_HEADING_CASE = 9

// FLAG_REWRITE_OPEN records that the rewrite stands in the section of a heading the check records.
const FLAG_REWRITE_OPEN = 10

// FLAG_REWRITE_BODY records that the open section holds a body line.
const FLAG_REWRITE_BODY = 11

// FLAG_REWRITE_FIT records that every write fit the rewrite storage.
const FLAG_REWRITE_FIT = 12

// FLAG_SLOT_COUNT is how many answers one check keeps.
const FLAG_SLOT_COUNT = 13

// SECTION_BODY_MAXIMUM is the most body lines one section holds.
const SECTION_BODY_MAXIMUM = 3

// Number is one slot of the number table: a count, a line, an offset, a rune, or one member of a
// state, pass, file, or message set.
type Number int

// Number_Invariants bounds one number by the widest bounded input, a Go source.
func Number_Invariants(value Number, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(int(value), NUMBER_MINIMUM, NUMBER_MAXIMUM).
    Ensure()
}

// Numbers holds every number one check keeps, and cannot grow.
type Numbers []Number

// Numbers_Invariants fixes the table to one slot per number.
func Numbers_Invariants(value Numbers, _ aver.Namespace) {
  aver.Always(len(value) == NUMBER_SLOT_COUNT, "Check numbers have complete length.")
  aver.Always(cap(value) == NUMBER_SLOT_COUNT, "Check numbers cannot grow.")
  // A cursor past its count names a heading the parse never recorded, thus each cursor
  // stands at its count at most, where it states that every one is served.
  aver.Always(
    value[NUMBER_HEADING_CURSOR] < value[NUMBER_HEADING_COUNT]+1,
    "A heading cursor stands inside its count.",
  )
  aver.Always(
    value[NUMBER_LEAF_CURSOR] < value[NUMBER_LEAF_COUNT]+1,
    "A leaf cursor stands inside its count.",
  )
}

// Number_Storage keeps the number table caller-owned.
type Number_Storage struct {
  // Values makes ownership explicit at the package boundary.
  Values Numbers
}

// Number_Storage_Invariants composes the number table.
func Number_Storage_Invariants(value Number_Storage, namespace aver.Namespace) {
  Numbers_Invariants(value.Values, namespace)
}

// Flag is one slot of the flag table.
type Flag bool

// Flag_Invariants proves both answers occur.
func Flag_Invariants(value Flag, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Sometimes(bool(value), "The requirements flag is set.").
    Ensure()
}

// Flags holds every flag one check keeps, and cannot grow.
type Flags []Flag

// Flags_Invariants fixes the table to one slot per flag.
func Flags_Invariants(value Flags, _ aver.Namespace) {
  aver.Always(len(value) == FLAG_SLOT_COUNT, "Check flags have complete length.")
  aver.Always(cap(value) == FLAG_SLOT_COUNT, "Check flags cannot grow.")
}

// Flag_Storage keeps the flag table caller-owned.
type Flag_Storage struct {
  // Values makes ownership explicit at the package boundary.
  Values Flags
}

// Flag_Storage_Invariants composes the flag table.
func Flag_Storage_Invariants(value Flag_Storage, namespace aver.Namespace) {
  Flags_Invariants(value.Values, namespace)
}

// Markdown_Bytes holds one REQUIREMENTS.md, one byte past the largest.
type Markdown_Bytes []byte

// Markdown_Bytes_Invariants fixes the markdown store.
func Markdown_Bytes_Invariants(value Markdown_Bytes, _ aver.Namespace) {
  aver.Always(len(value) == MARKDOWN_STORAGE_SIZE, "A markdown store has complete length.")
  aver.Always(cap(value) == MARKDOWN_STORAGE_SIZE, "A markdown store cannot grow.")
}

// Rewritten_Bytes holds one rewritten REQUIREMENTS.md, as wide as the store it loads back into.
type Rewritten_Bytes []byte

// Rewritten_Bytes_Invariants fixes the store.
func Rewritten_Bytes_Invariants(value Rewritten_Bytes, _ aver.Namespace) {
  aver.Always(len(value) == MARKDOWN_STORAGE_SIZE, "A rewrite store has complete length.")
  aver.Always(cap(value) == MARKDOWN_STORAGE_SIZE, "A rewrite store cannot grow.")
}

// Test_Bytes holds one requirements_test.go, the caller's own source store.
type Test_Bytes []byte

// Test_Bytes_Invariants fixes the test store.
func Test_Bytes_Invariants(value Test_Bytes, _ aver.Namespace) {
  aver.Always(len(value) == TEST_STORAGE_SIZE, "A test store has complete length.")
  aver.Always(cap(value) == TEST_STORAGE_SIZE, "A test store cannot grow.")
}

// Path_Bytes holds the package directory.
type Path_Bytes []byte

// Path_Bytes_Invariants fixes the path store.
func Path_Bytes_Invariants(value Path_Bytes, _ aver.Namespace) {
  aver.Always(len(value) == PATH_STORAGE_SIZE, "A path store has complete length.")
  aver.Always(cap(value) == PATH_STORAGE_SIZE, "A path store cannot grow.")
}

// Name_Bytes holds the test name one leaf needs.
type Name_Bytes []byte

// Name_Bytes_Invariants fixes the name store.
func Name_Bytes_Invariants(value Name_Bytes, _ aver.Namespace) {
  aver.Always(len(value) == NAME_SIZE_MAXIMUM, "A name store has complete length.")
  aver.Always(cap(value) == NAME_SIZE_MAXIMUM, "A name store cannot grow.")
}

// Prior_Bytes holds the test name one scan compares the earlier leaves against. It stands apart
// from the name store, because the scan holds two names at once.
type Prior_Bytes []byte

// Prior_Bytes_Invariants fixes the prior name store.
func Prior_Bytes_Invariants(value Prior_Bytes, _ aver.Namespace) {
  aver.Always(len(value) == NAME_SIZE_MAXIMUM, "A prior name store has complete length.")
  aver.Always(cap(value) == NAME_SIZE_MAXIMUM, "A prior name store cannot grow.")
}

// Message_Bytes holds the message of one finding.
type Message_Bytes []byte

// Message_Bytes_Invariants fixes the message store.
func Message_Bytes_Invariants(value Message_Bytes, _ aver.Namespace) {
  aver.Always(len(value) == MESSAGE_SIZE_MAXIMUM, "A message store has complete length.")
  aver.Always(cap(value) == MESSAGE_SIZE_MAXIMUM, "A message store cannot grow.")
}

// Heading_Level is 1 for a # heading and 3 for a ###.
type Heading_Level int

// Heading_Level_Invariants admits both levels.
func Heading_Level_Invariants(value Heading_Level, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_Int(int(value), int(LEVEL_BRANCH), int(LEVEL_LEAF)).
    Ensure()
}

// HEADING_LINE_MAXIMUM is the furthest line a heading stands on. A heading carries its marker on
// the line it stands on, thus the last line of a full store holds no heading.
const HEADING_LINE_MAXIMUM = LINE_NUMBER_MAXIMUM - 1

// Heading_Line is the 1-based line one heading stands on.
type Heading_Line int

// Heading_Line_Invariants bounds a heading's line by the markdown store.
func Heading_Line_Invariants(value Heading_Line, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(int(value), LINE_NUMBER_MINIMUM, HEADING_LINE_MAXIMUM).
    Ensure()
}

// POSITION_MINIMUM is the first line and the first offset alike. A heading is recorded before
// the rules that flag it, thus one a contract wrote at its first byte stands here.
const POSITION_MINIMUM = 0

// Heading_Start is the offset of one heading's text.
type Heading_Start int

// Heading_Start_Invariants bounds it by the markdown store. A heading's text stands past its
// marker, thus no heading opens at the first byte of the file.
func Heading_Start_Invariants(value Heading_Start, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(int(value), HEADING_MARKER_SIZE, MARKDOWN_OFFSET_MAXIMUM).
    Ensure()
}

// Heading_End is the offset past one heading's text.
type Heading_End int

// Heading_End_Invariants bounds it by the markdown store. A heading's text closes no earlier
// than its marker ends, thus the smallest end is that marker's own size.
func Heading_End_Invariants(value Heading_End, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(int(value), HEADING_MARKER_SIZE, MARKDOWN_OFFSET_MAXIMUM).
    Ensure()
}

// Heading_Leaf records a heading whose section needs a body and a test.
type Heading_Leaf bool

// Heading_Leaf_Invariants proves both kinds occur.
func Heading_Leaf_Invariants(value Heading_Leaf, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Sometimes(bool(value), "A heading opens a leaf section.").
    Ensure()
}

// Heading is one # or ### heading of a REQUIREMENTS.md.
type Heading struct {
  // Level is LEVEL_BRANCH or LEVEL_LEAF.
  Level Heading_Level
  // Line is the line the heading stands on.
  Line Heading_Line
  // Start is the offset of the heading text after its marker.
  Start Heading_Start
  // End is the offset past the heading text.
  End Heading_End
  // Leaf records a heading whose section needs a body and a test.
  Leaf Heading_Leaf
}

// Heading_Invariants composes one heading.
func Heading_Invariants(value Heading, namespace aver.Namespace) {
  Heading_Level_Invariants(value.Level, namespace)
  Heading_Line_Invariants(value.Line, namespace)
  Heading_Start_Invariants(value.Start, namespace)
  Heading_End_Invariants(value.End, namespace)
  Heading_Leaf_Invariants(value.Leaf, namespace)
}

// Opened_Heading is one heading a scan just read. The tree pass marks the sections that need a
// body and a test, thus the mark of one stands nowhere here.
type Opened_Heading struct {
  // Level is LEVEL_BRANCH or LEVEL_LEAF.
  Level Heading_Level
  // Line is the line the heading stands on.
  Line Heading_Line
  // Start is the offset of the heading text after its marker.
  Start Heading_Start
  // End is the offset past the heading text.
  End Heading_End
}

// Opened_Heading_Invariants composes one heading a scan read.
func Opened_Heading_Invariants(value Opened_Heading, namespace aver.Namespace) {
  Heading_Level_Invariants(value.Level, namespace)
  Heading_Line_Invariants(value.Line, namespace)
  Heading_Start_Invariants(value.Start, namespace)
  Heading_End_Invariants(value.End, namespace)
}

// Headings holds the headings of the file in order, and cannot grow.
type Headings []Heading

// Headings_Invariants fixes the heading table.
func Headings_Invariants(value Headings, _ aver.Namespace) {
  aver.Always(len(value) == HEADING_COUNT_MAXIMUM, "A heading table has complete length.")
  aver.Always(cap(value) == HEADING_COUNT_MAXIMUM, "A heading table cannot grow.")
}

// Heading_Slot is the heading a section closed on, or ABSENT where none stands open.
type Heading_Slot Number

// Heading_Slot_Invariants bounds the slot by the heading table, and admits the absent one.
func Heading_Slot_Invariants(value Heading_Slot, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(int(value), int(ABSENT), HEADING_INDEX_MAXIMUM).
    Ensure()
}

// Leaf_Index is a slot of the leaf table.
type Leaf_Index int

// Leaf_Index_Invariants bounds it by the leaf table, which holds one slot per heading.
func Leaf_Index_Invariants(value Leaf_Index, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(int(value), HEADING_INDEX_MINIMUM, HEADING_INDEX_MAXIMUM).
    Ensure()
}

// Leaf_Parent is the # above a ### leaf, or ABSENT for a # leaf.
type Leaf_Parent int

// Leaf_Parent_Invariants bounds it by the heading table.
func Leaf_Parent_Invariants(value Leaf_Parent, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(int(value), int(ABSENT), HEADING_INDEX_MAXIMUM).
    Ensure()
}

// HEADING_INDEX_MINIMUM is the first slot of the heading table.
const HEADING_INDEX_MINIMUM = 0

// HEADING_INDEX_MAXIMUM is the last slot of the heading table, since a count of slots names one
// past the last that holds a heading.
const HEADING_INDEX_MAXIMUM = HEADING_COUNT_MAXIMUM - 1

// Leaf_Heading is the leaf's own heading.
type Leaf_Heading int

// Leaf_Heading_Invariants bounds it by the heading table. A leaf stands on a heading the check
// already recorded, thus it names a slot rather than ABSENT.
func Leaf_Heading_Invariants(value Leaf_Heading, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(int(value), HEADING_INDEX_MINIMUM, HEADING_INDEX_MAXIMUM).
    Ensure()
}

// Leaf is one leaf section: a # alone, or a ### under its #.
type Leaf struct {
  // Parent is the # above a ### leaf, or ABSENT for a # leaf.
  Parent Leaf_Parent
  // Heading is the leaf's own heading.
  Heading Leaf_Heading
}

// Leaf_Invariants composes one leaf.
func Leaf_Invariants(value Leaf, namespace aver.Namespace) {
  Leaf_Parent_Invariants(value.Parent, namespace)
  Leaf_Heading_Invariants(value.Heading, namespace)
}

// CHILD_PARENT_INDEX_MINIMUM is the first slot a parent stands in, since a ### leaf stands
// under a # the check already recorded.
const CHILD_PARENT_INDEX_MINIMUM = HEADING_INDEX_MINIMUM

// CHILD_PARENT_INDEX_MAXIMUM is the last slot a parent stands in, since the ### under it takes
// the slot after it at the latest.
const CHILD_PARENT_INDEX_MAXIMUM = HEADING_INDEX_MAXIMUM - 1

// Child_Parent is the slot of the # a ### leaf stands under. A child states its parent, thus
// the value naming no parent stands nowhere here.
type Child_Parent int

// Child_Parent_Invariants bounds the parent by the heading table.
func Child_Parent_Invariants(value Child_Parent, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(
    int(value),
    CHILD_PARENT_INDEX_MINIMUM,
    CHILD_PARENT_INDEX_MAXIMUM,
  ).
    Ensure()
}

// CHILD_HEADING_INDEX_MINIMUM is the first slot a ### leaf's own heading stands in, since the
// # it stands under holds the slot before it at the earliest.
const CHILD_HEADING_INDEX_MINIMUM = HEADING_INDEX_MINIMUM + 1

// Child_Heading is the slot of a ### leaf's own heading. A child stands below the # it names,
// thus the first slot of the table stands nowhere here.
type Child_Heading int

// Child_Heading_Invariants bounds the heading by the heading table.
func Child_Heading_Invariants(value Child_Heading, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(
    int(value),
    CHILD_HEADING_INDEX_MINIMUM,
    HEADING_INDEX_MAXIMUM,
  ).
    Ensure()
}

// Child_Leaf is one ### leaf standing under its #. A # leaf stands under no heading, thus it
// stands nowhere here.
type Child_Leaf struct {
  // Parent is the # the leaf stands under.
  Parent Child_Parent
  // Heading is the leaf's own heading.
  Heading Child_Heading
}

// Child_Leaf_Invariants composes one leaf standing under a #.
func Child_Leaf_Invariants(value Child_Leaf, namespace aver.Namespace) {
  Child_Parent_Invariants(value.Parent, namespace)
  Child_Heading_Invariants(value.Heading, namespace)
}

// Leaves holds the leaves of the file in order; each heading is a leaf at most once, thus the
// table holds as many slots as the heading table.
type Leaves []Leaf

// Leaves_Invariants fixes the leaf table.
func Leaves_Invariants(value Leaves, _ aver.Namespace) {
  aver.Always(len(value) == HEADING_COUNT_MAXIMUM, "A leaf table has complete length.")
  aver.Always(cap(value) == HEADING_COUNT_MAXIMUM, "A leaf table cannot grow.")
}

// DECLARATION_END_MAXIMUM is the last offset a span closes at. A loaded test source holds the
// bound at most; a keyword stands before the name it opens, and a function's name before its
// parameters, thus two bytes at least follow every span.
const DECLARATION_END_MAXIMUM = SOURCE_SIZE_MAXIMUM - len("()")

// DECLARATION_START_MAXIMUM is the last offset a span opens at: a function's name holds one byte,
// and a keyword opens further back.
const DECLARATION_START_MAXIMUM = DECLARATION_END_MAXIMUM - len("a")

// Declaration_Start is the offset of the span one slot states: a function's name, or the keyword
// that opens any other declaration.
type Declaration_Start int

// Declaration_Start_Invariants bounds it by the widest test source.
func Declaration_Start_Invariants(value Declaration_Start, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(
    int(value),
    DECLARATION_START_MINIMUM,
    DECLARATION_START_MAXIMUM,
  ).
    Ensure()
}

// Declaration_End is the offset past it.
type Declaration_End int

// Declaration_End_Invariants bounds it by the widest test source.
func Declaration_End_Invariants(value Declaration_End, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(
    int(value),
    DECLARATION_END_MINIMUM,
    DECLARATION_END_MAXIMUM,
  ).
    Ensure()
}

// Declaration_Line is the line it stands on, or 0 past the line index.
type Declaration_Line int

// Declaration_Line_Invariants bounds it by what a position states. A source of more lines than
// the index holds states none, thus the absent line stands here too.
func Declaration_Line_Invariants(value Declaration_Line, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(int(value), LINE_ABSENT, LINE_NUMBER_MAXIMUM_TEST).
    Ensure()
}

// Declaration is one leading declaration of requirements_test.go: a function by its name, or a
// const, var, or type by its keyword.
type Declaration struct {
  // Start is the offset of the name.
  Start Declaration_Start
  // End is the offset past the name.
  End Declaration_End
  // Line is the line the name stands on.
  Line Declaration_Line
}

// Declaration_Invariants composes one declaration.
func Declaration_Invariants(value Declaration, namespace aver.Namespace) {
  Declaration_Start_Invariants(value.Start, namespace)
  Declaration_End_Invariants(value.End, namespace)
  Declaration_Line_Invariants(value.Line, namespace)
}

// Declarations holds the leading declarations, one slot per leaf at most.
type Declarations []Declaration

// Declarations_Invariants fixes the declaration table.
func Declarations_Invariants(value Declarations, _ aver.Namespace) {
  aver.Always(
    len(value) == HEADING_COUNT_MAXIMUM,
    "A declaration table has complete length.",
  )
  aver.Always(cap(value) == HEADING_COUNT_MAXIMUM, "A declaration table cannot grow.")
}

// Syntax is the parse state and the line index one check borrows for the test file: the run owns
// them, and only the pass that parses takes them.
type Syntax struct {
  // Syntax_Tree holds the parsed file. Each parse reads its own refusal from its result,
  // thus the store holds the tree alone rather than the refusal the parse before it left.
  Syntax_Tree ast.Syntax_Tree_Handle
  // Lines indexes it, so a finding names a declaration's line.
  Lines token.Line_Index_Handle
}

// Syntax_Invariants composes what one parse borrows.
func Syntax_Invariants(value Syntax, namespace aver.Namespace) {
  ast.Syntax_Tree_Handle_Invariants(value.Syntax_Tree, namespace)
  token.Line_Index_Handle_Invariants(value.Lines, namespace)
}

// Checker_Fields is one package's check: the stores the caller owns, and the numbers and flags
// one pass leaves for the next.
type Checker_Fields struct {
  // Path holds the package directory; a third_party or examples segment exempts it.
  Path Path_Bytes
  // Markdown holds REQUIREMENTS.md; the caller loads it before Begin.
  Markdown Markdown_Bytes
  // Rewritten holds what Rewrite writes of the contract before it loads it back.
  Rewritten Rewritten_Bytes
  // Test holds requirements_test.go, which the caller loads when the state is STATE_TEST.
  Test Test_Bytes
  // Headings holds the headings.
  Headings Headings
  // Leaves holds the leaves.
  Leaves Leaves
  // Declarations holds the leading declarations of the test file.
  Declarations Declarations
  // Name holds the test name a leaf needs.
  Name Name_Bytes
  // Prior holds the test name a scan compares the earlier leaves against.
  Prior Prior_Bytes
  // Message holds the message of a finding.
  Message Message_Bytes
  // Numbers holds every count, cursor, and member the check keeps.
  Numbers Number_Storage
  // Flags holds every yes or no the check keeps.
  Flags Flag_Storage
}

// Checker_Fields_Invariants composes every store the caller owns. The parse state and the line
// index belong to the run that lends them, thus the run asserts them and this chain holds each
// type once.
func Checker_Fields_Invariants(value Checker_Fields, namespace aver.Namespace) {
  Path_Bytes_Invariants(value.Path, namespace)
  Markdown_Bytes_Invariants(value.Markdown, namespace)
  Rewritten_Bytes_Invariants(value.Rewritten, namespace)
  Test_Bytes_Invariants(value.Test, namespace)
  Headings_Invariants(value.Headings, namespace)
  Leaves_Invariants(value.Leaves, namespace)
  Declarations_Invariants(value.Declarations, namespace)
  Name_Bytes_Invariants(value.Name, namespace)
  Prior_Bytes_Invariants(value.Prior, namespace)
  Message_Bytes_Invariants(value.Message, namespace)
  Number_Storage_Invariants(value.Numbers, namespace)
  Flag_Storage_Invariants(value.Flags, namespace)
  // A count past its store names bytes the store never held, thus each count stands at
  // the store it counts rather than at a slot past the end of it.
  numbers := value.Numbers.Values
  aver.Always(
    int(numbers[NUMBER_MARKDOWN_COUNT]) < len(value.Markdown)+1,
    "A markdown count stands inside its store.",
  )
  aver.Always(
    int(numbers[NUMBER_TEST_COUNT]) < len(value.Test)+1,
    "A test count stands inside its store.",
  )
  aver.Always(
    int(numbers[NUMBER_PATH_COUNT]) < len(value.Path)+1,
    "A path count stands inside its store.",
  )
  aver.Always(
    int(numbers[NUMBER_NAME_COUNT]) < len(value.Name)+1,
    "A name count stands inside its store.",
  )
  aver.Always(
    int(numbers[NUMBER_MESSAGE_COUNT]) < len(value.Message)+1,
    "A message count stands inside its store.",
  )
}

// Checker is one package's check.
type Checker struct {
  Checker_Fields
}

// Checker_Invariants composes every check field.
func Checker_Invariants(value Checker, namespace aver.Namespace) {
  Checker_Fields_Invariants(value.Checker_Fields, namespace)
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

// Begin starts the check of the package the stores and flags describe.
func Begin(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "begin.checker")
  numbers := checker.Numbers.Values
  numbers[NUMBER_PASS] = PASS_PRESENCE
  numbers[NUMBER_STATE] = STATE_STEP
  numbers[NUMBER_HEADING_COUNT] = 0
  numbers[NUMBER_LEAF_COUNT] = 0
  // A cursor stands where the package before it left off, thus each opens at the first
  // slot rather than past a table this check has yet to fill.
  numbers[NUMBER_HEADING_CURSOR] = 0
  numbers[NUMBER_LEAF_CURSOR] = 0

  numbers[NUMBER_DECLARATION_COUNT] = 0
  numbers[NUMBER_MESSAGE_COUNT] = 0
  numbers[NUMBER_NAME_COUNT] = 0
}

// Next advances to the next finding, to a request for requirements_test.go, or to the end.
func Next(checker Checker_Handle, syntax Syntax) {
  Checker_Handle_Invariants(checker, "next.checker")
  Syntax_Invariants(syntax, "next.syntax")
  numbers := checker.Numbers.Values
  numbers[NUMBER_STATE] = STATE_STEP
  for range STEP_COUNT_MAXIMUM {
    if numbers[NUMBER_STATE] != STATE_STEP {
      return
    }
    switch numbers[NUMBER_PASS] {
    case PASS_PRESENCE:
      step_presence(checker)
    case PASS_HEADINGS:
      step_heading(checker)
    case PASS_NAMES:
      step_names(checker)
    case PASS_TREE:
      step_tree(checker)
    case PASS_BODIES:
      step_body(checker)
    case PASS_TEST_FILE:
      step_test_file(checker)
    case PASS_TEST_PARSE:
      step_test_parse(checker, syntax)
    case PASS_TESTS:
      step_test(checker)
    case PASS_DONE:
      numbers[NUMBER_STATE] = STATE_DONE
    }
  }
}

// A package without a contract owes one, unless it is package main or vendored or example code.
// A contract past its storage stops the check, because a cut file would report faults it lacks.
func step_presence(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "step_presence.checker")
  numbers := checker.Numbers.Values
  if checker.Flags.Values[FLAG_HAS_MARKDOWN] {
    if int(numbers[NUMBER_MARKDOWN_COUNT]) == len(checker.Markdown) {
      numbers[NUMBER_PASS] = PASS_DONE
      finding_file(checker, FILE_MARKDOWN, 0, MESSAGE_MARKDOWN_OVERSIZED)
      return
    }
    numbers[NUMBER_PASS] = PASS_HEADINGS
    numbers[NUMBER_LINE_OFFSET] = 0
    numbers[NUMBER_LINE_NUMBER] = 1
    numbers[NUMBER_LINE_CHECK] = 0
    checker.Flags.Values[FLAG_SEEN_HEADING] = false
    return
  }
  numbers[NUMBER_PASS] = PASS_DONE
  if checker.Flags.Values[FLAG_IMPURE] {
    return
  }
  scan_exempt(checker)
  if checker.Flags.Values[FLAG_SCAN_RESULT] {
    return
  }
  finding_file(checker, FILE_MARKDOWN, 0, MESSAGE_MARKDOWN_ABSENT)
}

// Records whether any segment of the package directory is third_party or examples.
func scan_exempt(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "scan_exempt.checker")
  checker.Flags.Values[FLAG_SCAN_RESULT] = false
  path := checker.Path[:checker.Numbers.Values[NUMBER_PATH_COUNT]]
  start := 0
  for position_index := 0; position_index < len(path)+1; position_index++ {
    if position_index < len(path) {
      if path[position_index] != '/' {
        continue
      }
    }
    segment := string(path[start:position_index])
    if segment == "third_party" {
      checker.Flags.Values[FLAG_SCAN_RESULT] = true
    }
    if segment == "examples" {
      checker.Flags.Values[FLAG_SCAN_RESULT] = true
    }
    start = position_index + 1
  }
}

// Scans one line of the file in up to four checks, stopping at each finding.
func step_heading(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "step_heading.checker")
  numbers := checker.Numbers.Values
  numbers[NUMBER_FINDING_HEADING] = numbers[NUMBER_HEADING_COUNT] - 1
  switch numbers[NUMBER_LINE_CHECK] {
  case 0:
    line_measure(checker)
    heading_collect(checker)
  case 1:
    numbers[NUMBER_LINE_CHECK] = 2
    heading := checker.Headings[numbers[NUMBER_FINDING_HEADING]]
    numbers[NUMBER_SCAN_START] = Number(heading.Start)
    numbers[NUMBER_SCAN_END] = Number(heading.End)
    scan_words_invalid(checker)
    if checker.Flags.Values[FLAG_HEADING_CASE] {
      if !bool(checker.Flags.Values[FLAG_SCAN_RESULT]) {
        finding_quote(checker, Heading_Message(MESSAGE_HEADING_CASE))
        return
      }
    }
    if checker.Flags.Values[FLAG_SCAN_RESULT] {
      finding_quote(checker, Heading_Message(MESSAGE_HEADING_CHARACTERS))
    }
  case 2:
    numbers[NUMBER_LINE_CHECK] = 3
    preceded := numbers[NUMBER_LINE_NUMBER] > 1
    if preceded {
      preceded = bool(blank_before(checker))
    }
    if !preceded {
      finding_quote(checker, Heading_Message(MESSAGE_BLANK_BEFORE))
    }
  case 3:
    followed := bool(blank_after(checker))
    line_advance(checker)
    if !followed {
      finding_quote(checker, Heading_Message(MESSAGE_BLANK_AFTER))
    }
  }
}

// Reads the scanned line's heading, or flags what stands on a line that is none.
func heading_collect(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "heading_collect.checker")
  numbers := checker.Numbers.Values
  markdown := checker.Markdown[:numbers[NUMBER_MARKDOWN_COUNT]]
  line := string(markdown[numbers[NUMBER_LINE_OFFSET]:numbers[NUMBER_LINE_END]])
  level := Number(0)
  start := numbers[NUMBER_LINE_OFFSET]
  if len(line) > len("### ")-1 {
    if line[:len("### ")] == "### " {
      level = LEVEL_LEAF
      start += Number(len("### "))
    }
  }
  if level == 0 {
    if len(line) > len("# ")-1 {
      if line[:len("# ")] == "# " {
        level = LEVEL_BRANCH
        start += Number(len("# "))
      }
    }
  }
  if level == 0 {
    heading_absent(checker)
    return
  }
  if int(numbers[NUMBER_LINE_END]-start) > HEADING_SIZE_MAXIMUM {
    numbers[NUMBER_PASS] = PASS_DONE
    finding_file(
      checker,
      FILE_MARKDOWN,
      Finding_Line(numbers[NUMBER_LINE_NUMBER]),
      MESSAGE_HEADING_OVERSIZED,
    )
    return
  }
  if int(numbers[NUMBER_HEADING_COUNT]) == len(checker.Headings) {
    numbers[NUMBER_PASS] = PASS_DONE
    finding_file(
      checker,
      FILE_MARKDOWN,
      Finding_Line(numbers[NUMBER_LINE_NUMBER]),
      MESSAGE_HEADINGS_FULL,
    )
    return
  }
  // A table states its length alone, thus each heading states its own parts where it is
  // recorded rather than nowhere at all.
  recorded := Opened_Heading{
    Level: Heading_Level(level),
    Line:  Heading_Line(numbers[NUMBER_LINE_NUMBER]),
    Start: Heading_Start(start),
    End:   Heading_End(numbers[NUMBER_LINE_END]),
  }
  Opened_Heading_Invariants(recorded, "heading_open.recorded")
  checker.Headings[numbers[NUMBER_HEADING_COUNT]] = Heading{
    Level: recorded.Level,
    Line:  recorded.Line,
    Start: recorded.Start,
    End:   recorded.End,
  }
  numbers[NUMBER_HEADING_COUNT]++
  checker.Flags.Values[FLAG_SEEN_HEADING] = true
  numbers[NUMBER_LINE_CHECK] = 1
}

// A line that opens with # but no heading marker is a wrong level; any other content before the
// first heading is a preamble.
func heading_absent(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "heading_absent.checker")
  numbers := checker.Numbers.Values
  markdown := checker.Markdown[:numbers[NUMBER_MARKDOWN_COUNT]]
  marked := numbers[NUMBER_LINE_OFFSET] < numbers[NUMBER_LINE_END]
  if marked {
    marked = markdown[numbers[NUMBER_LINE_OFFSET]] == '#'
  }
  number := numbers[NUMBER_LINE_NUMBER]
  opening := markdown[numbers[NUMBER_LINE_OFFSET]:numbers[NUMBER_LINE_END]]
  opening = opening[:min(len(opening), CONFLICT_MARKER_SIZE)]
  numbers[NUMBER_SCAN_START] = numbers[NUMBER_LINE_OFFSET]
  numbers[NUMBER_SCAN_END] = numbers[NUMBER_LINE_END]
  line_advance(checker)
  if Conflict_Marker(Line_Opening(opening)) {
    finding_file(checker, FILE_MARKDOWN, Finding_Line(number), MESSAGE_CONFLICT_MARKER)
    return
  }
  if marked {
    finding_file(checker, FILE_MARKDOWN, Finding_Line(number), MESSAGE_HEADING_LEVEL)
    return
  }
  scan_blank(checker)
  if checker.Flags.Values[FLAG_SCAN_RESULT] {
    return
  }
  if checker.Flags.Values[FLAG_SEEN_HEADING] {
    return
  }
  finding_file(checker, FILE_MARKDOWN, Finding_Line(number), MESSAGE_PREAMBLE)
}

// Walks the headings into leaves: a # with no ### child is a leaf, and so is each ###.
func step_tree(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "step_tree.checker")
  numbers := checker.Numbers.Values
  if numbers[NUMBER_HEADING_CURSOR] == numbers[NUMBER_HEADING_COUNT] {
    tree_close(checker)
    numbers[NUMBER_PASS] = PASS_NAMES
    numbers[NUMBER_LEAF_CURSOR] = 0
    return
  }
  cursor := numbers[NUMBER_HEADING_CURSOR]
  heading := checker.Headings[cursor]
  numbers[NUMBER_HEADING_CURSOR]++
  numbers[NUMBER_FINDING_HEADING] = cursor
  if Number(heading.Level) == LEVEL_LEAF {
    if numbers[NUMBER_PARENT] == ABSENT {
      finding_quote(checker, Heading_Message(MESSAGE_ORPHAN))
      return
    }
    checker.Flags.Values[FLAG_HAS_CHILD] = true
    child := Child_Leaf{
      Parent:  Child_Parent(numbers[NUMBER_PARENT]),
      Heading: Child_Heading(cursor),
    }
    Child_Leaf_Invariants(child, "tree_step.child")
    checker.Leaves[numbers[NUMBER_LEAF_COUNT]] = Leaf{
      Parent:  Leaf_Parent(child.Parent),
      Heading: Leaf_Heading(child.Heading),
    }
    numbers[NUMBER_LEAF_COUNT]++
    checker.Headings[cursor].Leaf = true
    numbers[NUMBER_SCAN_START] = numbers[NUMBER_PARENT] + 1
  } else {
    tree_close(checker)
    numbers[NUMBER_PARENT] = cursor
    checker.Flags.Values[FLAG_HAS_CHILD] = false
    numbers[NUMBER_SCAN_START] = 0
  }
  numbers[NUMBER_SCAN_END] = cursor
  scan_duplicate(checker)
  if checker.Flags.Values[FLAG_SCAN_RESULT] {
    finding_quote(checker, Heading_Message(MESSAGE_DUPLICATE))
  }
}

// Records the open # as a leaf when it gained no ### child.
func tree_close(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "tree_close.checker")
  numbers := checker.Numbers.Values
  if numbers[NUMBER_PARENT] == ABSENT {
    return
  }
  if checker.Flags.Values[FLAG_HAS_CHILD] {
    return
  }
  lone := Leaf{
    Parent: Leaf_Parent(ABSENT), Heading: Leaf_Heading(numbers[NUMBER_PARENT]),
  }
  // A # with no ### child stands under no other heading, thus its parent states the one
  // value rather than a slot the table could hold.
  Leaf_Heading_Invariants(lone.Heading, "tree_close.lone")
  aver.Always(lone.Parent == Leaf_Parent(ABSENT), "A lone section stands under no #.")
  checker.Leaves[numbers[NUMBER_LEAF_COUNT]] = lone
  numbers[NUMBER_LEAF_COUNT]++
  checker.Headings[numbers[NUMBER_PARENT]].Leaf = true
}

// Flags the leaf the cursor stands on where an earlier leaf states its test name. Two headings
// a host wrote apart fold to one name, and the file that answers one answers both, thus the run
// names the second rather than asks for a declaration Go states once.
func step_names(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "step_names.checker")
  numbers := checker.Numbers.Values
  if numbers[NUMBER_LEAF_CURSOR] == numbers[NUMBER_LEAF_COUNT] {
    numbers[NUMBER_PASS] = PASS_BODIES
    numbers[NUMBER_LINE_OFFSET] = 0
    numbers[NUMBER_LINE_NUMBER] = 1
    numbers[NUMBER_LINE_CHECK] = 0
    numbers[NUMBER_OPEN] = ABSENT
    numbers[NUMBER_BODY_COUNT] = 0
    checker.Flags.Values[FLAG_BLANK] = false
    numbers[NUMBER_HEADING_CURSOR] = 0
    numbers[NUMBER_LEAF_CURSOR] = 0
    return
  }
  subject := numbers[NUMBER_LEAF_CURSOR]
  numbers[NUMBER_LEAF_CURSOR]++
  numbers[NUMBER_NAME_LEAF] = Number(subject)
  name_write(checker)
  prior := append(checker.Prior[:0], checker.Name[:numbers[NUMBER_NAME_COUNT]]...)
  checker.Flags.Values[FLAG_SCAN_RESULT] = false
  markdown := checker.Markdown[:numbers[NUMBER_MARKDOWN_COUNT]]
  second := checker.Leaves[subject]
  tail := checker.Headings[second.Heading]
  for index := Leaf_Index(0); index < Leaf_Index(subject); index++ {
    // Two leaves under one # standing on one text state one name, and the
    // uniqueness rule already flags that pair, thus this states it once.
    first := checker.Leaves[index]
    head := checker.Headings[first.Heading]
    repeated := first.Parent == second.Parent
    if repeated {
      repeated = string(markdown[head.Start:head.End]) ==
        string(markdown[tail.Start:tail.End])
    }
    if repeated {
      continue
    }
    numbers[NUMBER_NAME_LEAF] = Number(index)
    name_write(checker)
    if string(checker.Name[:numbers[NUMBER_NAME_COUNT]]) == string(prior) {
      checker.Flags.Values[FLAG_SCAN_RESULT] = true
    }
  }
  if !bool(checker.Flags.Values[FLAG_SCAN_RESULT]) {
    return
  }
  numbers[NUMBER_FINDING_HEADING] = Number(checker.Leaves[subject].Heading)
  finding_quote(checker, Heading_Message(MESSAGE_LEAF_NAME))
}

// LATER_LEAF_INDEX_MINIMUM is the first slot a leaf standing after another stands in, since the
// leaf of the first slot stands after none.
const LATER_LEAF_INDEX_MINIMUM = HEADING_INDEX_MINIMUM + 1

// Later_Leaf is the slot of one leaf that stands after another. The first leaf of a file stands
// after none, thus its slot stands nowhere here.
type Later_Leaf int

// Later_Leaf_Invariants bounds the slot by the leaf table.
func Later_Leaf_Invariants(value Later_Leaf, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(int(value), LATER_LEAF_INDEX_MINIMUM, HEADING_INDEX_MAXIMUM).
    Ensure()
}

// Records whether a heading in the scanned span of the same level shares the text of the heading
// the scan ends at.
func scan_duplicate(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "scan_duplicate.checker")
  numbers := checker.Numbers.Values
  markdown := checker.Markdown[:numbers[NUMBER_MARKDOWN_COUNT]]
  subject := checker.Headings[numbers[NUMBER_SCAN_END]]
  text := string(markdown[subject.Start:subject.End])
  checker.Flags.Values[FLAG_SCAN_RESULT] = false
  for index := numbers[NUMBER_SCAN_START]; index < numbers[NUMBER_SCAN_END]; index++ {
    other := checker.Headings[index]
    if other.Level != subject.Level {
      continue
    }
    if string(markdown[other.Start:other.End]) == text {
      checker.Flags.Values[FLAG_SCAN_RESULT] = true
    }
  }
}

// Scans one line of the file for its section's body, stopping at each finding.
func step_body(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "step_body.checker")
  numbers := checker.Numbers.Values
  if numbers[NUMBER_LINE_CHECK] == 1 {
    body_line(checker)
    return
  }
  line_measure(checker)
  markdown := checker.Markdown[:numbers[NUMBER_MARKDOWN_COUNT]]
  cursor := numbers[NUMBER_HEADING_CURSOR]
  if cursor < numbers[NUMBER_HEADING_COUNT] {
    if Number(checker.Headings[cursor].Line) == numbers[NUMBER_LINE_NUMBER] {
      numbers[NUMBER_HEADING_CURSOR]++
      numbers[NUMBER_NEXT_OPEN] = cursor
      body_close(checker)
      return
    }
  }
  marked := numbers[NUMBER_LINE_OFFSET] < numbers[NUMBER_LINE_END]
  if marked {
    marked = markdown[numbers[NUMBER_LINE_OFFSET]] == '#'
  }
  if marked {
    numbers[NUMBER_NEXT_OPEN] = ABSENT
    body_close(checker)
    return
  }
  numbers[NUMBER_SCAN_START] = numbers[NUMBER_LINE_OFFSET]
  numbers[NUMBER_SCAN_END] = numbers[NUMBER_LINE_END]
  scan_blank(checker)
  if checker.Flags.Values[FLAG_SCAN_RESULT] {
    if numbers[NUMBER_BODY_COUNT] > 0 {
      checker.Flags.Values[FLAG_BLANK] = true
    }
    line_advance(checker)
    return
  }
  if numbers[NUMBER_OPEN] == ABSENT {
    line_advance(checker)
    return
  }
  numbers[NUMBER_LINE_CHECK] = 1
  if checker.Flags.Values[FLAG_BLANK] {
    checker.Flags.Values[FLAG_BLANK] = false
    numbers[NUMBER_FINDING_HEADING] = numbers[NUMBER_OPEN]
    number := numbers[NUMBER_LINE_NUMBER]
    finding_quote(checker, Heading_Message(MESSAGE_SECTION_GAP))
    numbers[NUMBER_LINE] = number
  }
}

// Counts one body line of the open section, flagging the line past its bound.
func body_line(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "body_line.checker")
  numbers := checker.Numbers.Values
  numbers[NUMBER_BODY_COUNT]++
  number := numbers[NUMBER_LINE_NUMBER]
  line_advance(checker)
  if numbers[NUMBER_BODY_COUNT] != SECTION_BODY_MAXIMUM+1 {
    return
  }
  numbers[NUMBER_FINDING_HEADING] = numbers[NUMBER_OPEN]
  finding_quote(checker, Heading_Message(MESSAGE_SECTION_SIZE))
  numbers[NUMBER_LINE] = number
}

// Closes the open section, flagging a leaf with no body, then opens the next one.
func body_close(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "body_close.checker")
  numbers := checker.Numbers.Values
  closed := numbers[NUMBER_OPEN]
  empty := Flag(numbers[NUMBER_BODY_COUNT] == 0)
  // A table states its length alone, and a bundle body is straight-line, thus the range
  // one slot of it stands in is stated where a slot is written rather than nowhere.
  opened := numbers[NUMBER_NEXT_OPEN]
  Heading_Slot_Invariants(Heading_Slot(opened), "body_close.opened")
  numbers[NUMBER_OPEN] = opened
  numbers[NUMBER_BODY_COUNT] = 0
  checker.Flags.Values[FLAG_BLANK] = false
  line_advance(checker)
  body_stated(checker, Heading_Slot(closed), empty)
}

// States a leaf section that closed carrying no body line. A section closes where the next
// heading stands and where the file ends alike, thus both ways of closing state it.
func body_stated(checker Checker_Handle, closed Heading_Slot, empty Flag) {
  Checker_Handle_Invariants(checker, "body_stated.checker")
  Heading_Slot_Invariants(closed, "body_stated.closed")
  Flag_Invariants(empty, "body_stated.empty")
  if closed == Heading_Slot(ABSENT) {
    return
  }
  if !checker.Headings[closed].Leaf {
    return
  }
  if !bool(empty) {
    return
  }
  checker.Numbers.Values[NUMBER_FINDING_HEADING] = Number(closed)
  finding_quote(checker, Heading_Message(MESSAGE_SECTION_EMPTY))
}

// Asks for requirements_test.go, or flags its absence.
func step_test_file(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "step_test_file.checker")
  numbers := checker.Numbers.Values
  if !checker.Flags.Values[FLAG_HAS_TEST] {
    numbers[NUMBER_PASS] = PASS_DONE
    finding_file(checker, FILE_TEST, 0, MESSAGE_TEST_ABSENT)
    return
  }
  numbers[NUMBER_PASS] = PASS_TEST_PARSE
  numbers[NUMBER_STATE] = STATE_TEST
}

// Collects the leading declarations of the test file. A source the parser refuses has its own
// diagnostic, thus its tests go unchecked rather than flagged twice.
func step_test_parse(checker Checker_Handle, syntax Syntax) {
  Checker_Handle_Invariants(checker, "step_test_parse.checker")
  Syntax_Invariants(syntax, "step_test_parse.syntax")
  numbers := checker.Numbers.Values
  numbers[NUMBER_PASS] = PASS_DONE
  source := token.Source(checker.Test[:numbers[NUMBER_TEST_COUNT]])
  view := ast.Parse_State{Syntax_Tree: *syntax.Syntax_Tree}
  state := &view
  parsed := ast.Parse(state, source)
  if !bool(parsed.OK) {
    return
  }
  indexed := token.Index_Lines(syntax.Lines, source)
  root := ast.Node_At(state, ast.Index(parsed.Root))
  child := ast.Index(root.First_Child)
  for range ast.NODE_COUNT_MAXIMUM {
    if child == ast.INDEX_ABSENT {
      break
    }
    if int(numbers[NUMBER_DECLARATION_COUNT]) == len(checker.Declarations) {
      break
    }
    node := ast.Node_At(state, child)
    child = ast.Index(node.Next)
    // The token that names one top-level declaration, or none for what is no
    // declaration: the package name, an import, a comment, or a blank line.
    numbers[NUMBER_NAME_TOKEN] = Number(ast.INDEX_ABSENT)
    switch node.Kind {
    case ast.NODE_CONSTANT, ast.NODE_VARIABLE, ast.NODE_TYPE:
      numbers[NUMBER_NAME_TOKEN] = Number(node.Token)
    case ast.NODE_FUNCTION:
      // A method's receiver comes first, thus the name is the first identifier,
      // not the first child.
      part := ast.Index(node.First_Child)
      for range ast.NODE_COUNT_MAXIMUM {
        if part == ast.INDEX_ABSENT {
          break
        }
        held := ast.Node_At(state, part)
        if held.Kind == ast.NODE_IDENTIFIER {
          numbers[NUMBER_NAME_TOKEN] = Number(held.Token)
          break
        }
        part = ast.Index(held.Next)
      }
    }
    if numbers[NUMBER_NAME_TOKEN] == Number(ast.INDEX_ABSENT) {
      continue
    }
    token_index := ast.Token_Index(numbers[NUMBER_NAME_TOKEN])
    named := ast.Token_At(state, token_index)
    line := Number(0)
    if indexed {
      index := syntax.Lines
      line = Number(token.Position_Of(index, named.Offset).Line)
    }
    named_slot := Declaration{
      Start: Declaration_Start(named.Offset),
      End:   Declaration_End(int(named.Offset) + int(named.Size)),
      Line:  Declaration_Line(line),
    }
    Declaration_Invariants(named_slot, "step_test_parse.named_slot")
    checker.Declarations[numbers[NUMBER_DECLARATION_COUNT]] = named_slot
    numbers[NUMBER_DECLARATION_COUNT]++
  }
  numbers[NUMBER_PASS] = PASS_TESTS
  numbers[NUMBER_LEAF_CURSOR] = 0
}

// Compares one leaf's test name with the declaration in its slot.
func step_test(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "step_test.checker")
  numbers := checker.Numbers.Values
  if numbers[NUMBER_LEAF_CURSOR] == numbers[NUMBER_LEAF_COUNT] {
    numbers[NUMBER_PASS] = PASS_DONE
    return
  }
  numbers[NUMBER_NAME_LEAF] = numbers[NUMBER_LEAF_CURSOR]
  name_write(checker)
  slot := numbers[NUMBER_LEAF_CURSOR]
  numbers[NUMBER_LEAF_CURSOR]++
  line := Number(0)
  if slot < numbers[NUMBER_DECLARATION_COUNT] {
    declaration := checker.Declarations[slot]
    line = Number(declaration.Line)
    name := string(checker.Test[declaration.Start:declaration.End])
    if name == string(checker.Name[:numbers[NUMBER_NAME_COUNT]]) {
      return
    }
  }
  finding_file(checker, FILE_TEST, Finding_Line(line), MESSAGE_TEST_ORDER)
}

// Writes the test name of the leaf the cursor stands on into the name store.
func name_write(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "name_write.checker")
  numbers := checker.Numbers.Values
  leaf := checker.Leaves[numbers[NUMBER_NAME_LEAF]]
  name := append(checker.Name[:0], TEST_PREFIX...)
  numbers[NUMBER_NAME_COUNT] = Number(len(name))
  if Number(leaf.Parent) != ABSENT {
    heading := checker.Headings[leaf.Parent]
    numbers[NUMBER_SCAN_START] = Number(heading.Start)
    numbers[NUMBER_SCAN_END] = Number(heading.End)
    name_ada(checker)
    name = append(checker.Name[:numbers[NUMBER_NAME_COUNT]], '_')
    numbers[NUMBER_NAME_COUNT] = Number(len(name))
  }
  heading := checker.Headings[leaf.Heading]
  numbers[NUMBER_SCAN_START] = Number(heading.Start)
  numbers[NUMBER_SCAN_END] = Number(heading.End)
  name_ada(checker)
}

// Appends the Ada_Case form of the scanned span to the name store: each word's first rune upper
// cased, and the words joined by underscores.
func name_ada(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "name_ada.checker")
  numbers := checker.Numbers.Values
  markdown := checker.Markdown[:numbers[NUMBER_MARKDOWN_COUNT]]
  name := checker.Name[:numbers[NUMBER_NAME_COUNT]]
  word_open := false
  written := false
  numbers[NUMBER_SCAN_POSITION] = numbers[NUMBER_SCAN_START]
  for range HEADING_SIZE_MAXIMUM {
    if numbers[NUMBER_SCAN_POSITION] == numbers[NUMBER_SCAN_END] {
      break
    }
    start := numbers[NUMBER_SCAN_POSITION]
    character_read(checker)
    if checker.Flags.Values[FLAG_CHARACTER_SPACE] {
      word_open = false
      continue
    }
    if word_open {
      name = append(name, markdown[start:numbers[NUMBER_SCAN_POSITION]]...)
      continue
    }
    if written {
      name = append(name, '_')
    }
    word_open = true
    written = true
    character_upper(checker)
    var encoded [utf8.CHARACTER_SIZE_MAXIMUM]byte
    upper := utf8.Character(numbers[NUMBER_UPPER])
    count := utf8.Encode_Character(utf8.Bytes(encoded[:]), upper)
    name = append(name, encoded[:count]...)
  }
  numbers[NUMBER_NAME_COUNT] = Number(len(name))
}

// Records whether the scanned span holds a word rune that is neither a letter nor a digit, which
// survives into the test name and makes it no identifier.
func scan_words_invalid(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "scan_words_invalid.checker")
  numbers := checker.Numbers.Values
  invalid := false
  // A heading of no word builds the name Test_ alone, which the canonical form spells
  // Test, thus a file that answers the leaf and a file that stands in that form are two
  // different files. Such a heading is as invalid as one holding a byte no word holds.
  worded := false
  // The canonical form splits a word where a capital follows a lower case letter, thus a
  // heading spelled that way builds a name the form rewrites and the leaf never matches.
  cased := false
  lower := false
  numbers[NUMBER_SCAN_POSITION] = numbers[NUMBER_SCAN_START]
  for range HEADING_SIZE_MAXIMUM {
    if numbers[NUMBER_SCAN_POSITION] == numbers[NUMBER_SCAN_END] {
      break
    }
    character_read(checker)
    if checker.Flags.Values[FLAG_CHARACTER_SPACE] {
      lower = false
      continue
    }
    if !checker.Flags.Values[FLAG_CHARACTER_WORD] {
      invalid = true
    }
    worded = true
    character := numbers[NUMBER_CHARACTER]
    if lower {
      if uint32(character-'A') < 26 {
        cased = true
      }
    }
    lower = uint32(character-'a') < 26
  }
  checker.Flags.Values[FLAG_HEADING_CASE] = Flag(cased)
  if !worded {
    invalid = true
  }
  checker.Flags.Values[FLAG_SCAN_RESULT] = Flag(invalid)
}

// Records whether the scanned span holds white space alone.
func scan_blank(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "scan_blank.checker")
  numbers := checker.Numbers.Values
  blank := true
  numbers[NUMBER_SCAN_POSITION] = numbers[NUMBER_SCAN_START]
  for range MARKDOWN_STORAGE_SIZE {
    if numbers[NUMBER_SCAN_POSITION] == numbers[NUMBER_SCAN_END] {
      break
    }
    character_read(checker)
    if !checker.Flags.Values[FLAG_CHARACTER_SPACE] {
      blank = false
    }
  }
  checker.Flags.Values[FLAG_SCAN_RESULT] = Flag(blank)
}

// Decodes the rune the scan stands on and classifies it. Headings use ASCII almost always, thus
// the Unicode tables stay the fallback.
func character_read(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "character_read.checker")
  numbers := checker.Numbers.Values
  span := checker.Markdown[numbers[NUMBER_SCAN_POSITION]:numbers[NUMBER_SCAN_END]]
  character, size := utf8.Decode_Character(utf8.Bytes(span))
  numbers[NUMBER_CHARACTER] = Number(character)
  numbers[NUMBER_SCAN_POSITION] += Number(size)
  checker.Flags.Values[FLAG_CHARACTER_SPACE] = false
  checker.Flags.Values[FLAG_CHARACTER_WORD] = false
  // A rune past ASCII is no word character, whichever class Unicode gives it: the name a
  // word builds is one this toolchain has to spell, and it spells ASCII alone.
  if character > 0x7f {
    space := ucd.Is_Space(ucd.Character(character))
    checker.Flags.Values[FLAG_CHARACTER_SPACE] = Flag(space)
    return
  }
  switch character {
  case ' ', '\t', '\n', '\v', '\f', '\r':
    checker.Flags.Values[FLAG_CHARACTER_SPACE] = true
  }
  // Unsigned, so a rune below the range wraps past it rather than passing as negative.
  if uint32(character-'a') < 26 {
    checker.Flags.Values[FLAG_CHARACTER_WORD] = true
  }
  if uint32(character-'A') < 26 {
    checker.Flags.Values[FLAG_CHARACTER_WORD] = true
  }
  if uint32(character-'0') < 10 {
    checker.Flags.Values[FLAG_CHARACTER_WORD] = true
  }
}

// Records the upper case of the rune the scan read.
func character_upper(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "character_upper.checker")
  numbers := checker.Numbers.Values
  character := numbers[NUMBER_CHARACTER]
  numbers[NUMBER_UPPER] = character
  if uint32(character-'a') < 26 {
    numbers[NUMBER_UPPER] = character - ('a' - 'A')
    return
  }
  if character > 0x7f {
    numbers[NUMBER_UPPER] = Number(ucd.To_Upper(ucd.Character(character)))
  }
}

// Flags a contract that states no heading and no content either. Such a file states no section,
// thus the package it stands in owes the nothing a package with no contract at all would owe.
// Content without a heading is the preamble fault, and one finding for it is enough.
func headings_stated(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "headings_stated.checker")
  numbers := checker.Numbers.Values
  if numbers[NUMBER_HEADING_COUNT] != 0 {
    return
  }
  markdown := checker.Markdown[:numbers[NUMBER_MARKDOWN_COUNT]]
  for index := range markdown {
    switch markdown[index] {
    case ' ', '\t', LINE_RETURN, '\n':
      continue
    }
    return
  }
  finding_file(checker, FILE_MARKDOWN, LINE_ABSENT, MESSAGE_HEADINGS_ABSENT)
}

// LINE_RETURN stands before the newline of a file a host wrote with two bytes for one ending.
const LINE_RETURN = '\r'

// Reports whether the line after the open one holds nothing, whichever ending its host wrote.
func blank_after(checker Checker_Handle) (blank Flag) {
  defer func() { Flag_Invariants(blank, "blank_after.blank") }()
  Checker_Handle_Invariants(checker, "blank_after.checker")
  numbers := checker.Numbers.Values
  markdown := checker.Markdown[:numbers[NUMBER_MARKDOWN_COUNT]]
  position := int(numbers[NUMBER_LINE_END])
  if position == len(markdown) {
    return false
  }
  if markdown[position] == LINE_RETURN {
    position++
  }
  // A carriage return the file ends on carries no newline after it, thus the step past
  // that newline stands only where the newline does.
  if position == len(markdown) {
    return false
  }
  position++
  if position == len(markdown) {
    return true
  }
  if markdown[position] == LINE_RETURN {
    position++
  }
  if position == len(markdown) {
    return true
  }
  return Flag(markdown[position] == '\n')
}

// Reports whether the line before the open one holds nothing, whichever ending its host wrote.
func blank_before(checker Checker_Handle) (blank Flag) {
  defer func() { Flag_Invariants(blank, "blank_before.blank") }()
  Checker_Handle_Invariants(checker, "blank_before.checker")
  numbers := checker.Numbers.Values
  markdown := checker.Markdown[:numbers[NUMBER_MARKDOWN_COUNT]]
  position := int(numbers[NUMBER_LINE_OFFSET]) - 2
  if position < 0 {
    return true
  }
  if markdown[position] == LINE_RETURN {
    position--
  }
  if position < 0 {
    return true
  }
  return Flag(markdown[position] == '\n')
}

// Finds where the scanned line closes. A carriage return before the newline belongs to the
// ending rather than the line, thus a file one host wrote states what it states on every other.
func line_measure(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "line_measure.checker")
  numbers := checker.Numbers.Values
  markdown := checker.Markdown[:numbers[NUMBER_MARKDOWN_COUNT]]
  numbers[NUMBER_LINE_END] = Number(len(markdown))
  for position := int(numbers[NUMBER_LINE_OFFSET]); position < len(markdown); position++ {
    if markdown[position] != '\n' {
      continue
    }
    numbers[NUMBER_LINE_END] = Number(position)
    break
  }
  end := int(numbers[NUMBER_LINE_END])
  if end == int(numbers[NUMBER_LINE_OFFSET]) {
    return
  }
  if markdown[end-1] != LINE_RETURN {
    return
  }
  numbers[NUMBER_LINE_END] = Number(end - 1)
}

// Moves to the next line, or to the next pass past the last. The text after the final newline
// counts as a line, empty or not.
func line_advance(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "line_advance.checker")
  numbers := checker.Numbers.Values
  numbers[NUMBER_LINE_CHECK] = 0
  if numbers[NUMBER_LINE_END] == numbers[NUMBER_MARKDOWN_COUNT] {
    switch numbers[NUMBER_PASS] {
    case PASS_HEADINGS:
      numbers[NUMBER_PASS] = PASS_TREE
      numbers[NUMBER_HEADING_CURSOR] = 0
      numbers[NUMBER_PARENT] = ABSENT
      checker.Flags.Values[FLAG_HAS_CHILD] = false
      headings_stated(checker)
    case PASS_BODIES:
      numbers[NUMBER_PASS] = PASS_TEST_FILE
      closed := numbers[NUMBER_OPEN]
      empty := Flag(numbers[NUMBER_BODY_COUNT] == 0)
      numbers[NUMBER_OPEN] = ABSENT
      numbers[NUMBER_BODY_COUNT] = 0
      body_stated(checker, Heading_Slot(closed), empty)
    }
    return
  }
  // The ending is one line whichever bytes a host spends on it, thus a carriage return
  // before the newline is stepped over rather than counted as a line of its own.
  markdown := checker.Markdown[:numbers[NUMBER_MARKDOWN_COUNT]]
  ending := int(numbers[NUMBER_LINE_END])
  if markdown[ending] == LINE_RETURN {
    // A carriage return belongs to the ending only where a newline follows it, thus
    // one a host wrote last stands as a byte of its line rather than half an ending.
    if ending+1 < len(markdown) {
      ending++
    }
  }
  numbers[NUMBER_LINE_OFFSET] = Number(ending + 1)
  numbers[NUMBER_LINE_NUMBER]++
}

// Opens a finding at one line of one file, whose message quotes nothing.
func finding_file(checker Checker_Handle, file File, line Finding_Line, message Message) {
  Checker_Handle_Invariants(checker, "finding_file.checker")
  File_Invariants(file, "finding_file.file")
  Finding_Line_Invariants(line, "finding_file.line")
  Message_Invariants(message, "finding_file.message")
  numbers := checker.Numbers.Values
  numbers[NUMBER_STATE] = STATE_FINDING
  numbers[NUMBER_FILE] = Number(file)
  numbers[NUMBER_LINE] = Number(line)
  numbers[NUMBER_MESSAGE] = Number(message)
  message_write(checker)
}

// Finding_Line is the line one finding names, or LINE_ABSENT where it names a file alone. The
// test store is the widest file a check reads, thus its lines bound this.
type Finding_Line int

// Finding_Line_Invariants bounds the line one finding names.
func Finding_Line_Invariants(value Finding_Line, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(int(value), LINE_ABSENT, LINE_NUMBER_MAXIMUM_TEST).
    Ensure()
}

// Opens a finding on the line of the heading a message quotes.
func finding_quote(checker Checker_Handle, message Heading_Message) {
  Checker_Handle_Invariants(checker, "finding_quote.checker")
  Heading_Message_Invariants(message, "finding_quote.message")
  numbers := checker.Numbers.Values
  heading := checker.Headings[numbers[NUMBER_FINDING_HEADING]]
  finding_file(checker, FILE_MARKDOWN, Finding_Line(heading.Line), Message(message))
}

// QUOTED_BYTE_MINIMUM is the smallest byte a message quotes: one below it steers a terminal or
// breaks the line that holds it rather than names anything.
const QUOTED_BYTE_MINIMUM = ' '

// QUOTED_BYTE_ABSENT stands where a message would have quoted a byte below that.
const QUOTED_BYTE_ABSENT = '?'

// SPAN_SIZE_MINIMUM is no bytes: a heading of no bytes is one the file wrote as a heading mark
// and nothing after it.
const SPAN_SIZE_MINIMUM = 0

// QUOTE_OPENING_SIZE_MINIMUM is the shortest opening a message that quotes carries, thus a
// message holds at least that much before the quote stands in it.
const QUOTE_OPENING_SIZE_MINIMUM = len(`The heading "`)

// QUOTE_OPENING_SIZE_MAXIMUM is the widest opening a message that quotes carries.
const QUOTE_OPENING_SIZE_MAXIMUM = len(MESSAGE_TEXT_BLANK_BEFORE)

// Quote_Opening is the text one quote lands after: the opening of the message that quotes.
type Quote_Opening []byte

// Quote_Opening_Invariants bounds the opening by the openings a message that quotes carries.
func Quote_Opening_Invariants(value Quote_Opening, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(
    len(value),
    QUOTE_OPENING_SIZE_MINIMUM,
    QUOTE_OPENING_SIZE_MAXIMUM,
  ).
    Ensure()
}

// QUOTED_MESSAGE_SIZE_MAXIMUM is the widest message one quote leaves: the test order opening and
// the test name it quotes, which outruns any opening that quotes a heading.
const QUOTED_MESSAGE_SIZE_MAXIMUM = len(MESSAGE_TEXT_TEST_ORDER) + NAME_SIZE_MAXIMUM

// Quoted_Message is the message one quote is written into, which views the message store.
type Quoted_Message []byte

// Quoted_Message_Invariants bounds the message by its opening and the quote it holds.
func Quoted_Message_Invariants(value Quoted_Message, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(
    len(value),
    QUOTE_OPENING_SIZE_MINIMUM,
    QUOTED_MESSAGE_SIZE_MAXIMUM,
  ).
    Ensure()
}

// Markdown_Span is one heading of the file a check read, which views the markdown store.
type Untrusted_Span []byte

// Untrusted_Span_Invariants bounds the span by the widest one a message quotes, which is the
// test name a leaf builds rather than the heading it builds it from.
func Untrusted_Span_Invariants(value Untrusted_Span, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(len(value), SPAN_SIZE_MINIMUM, NAME_SIZE_MAXIMUM).
    Ensure()
}

// Quotes one span of the file a check read. The file is untrusted, thus a byte no line can state
// stands as a question mark rather than reaching whatever reads the line.
func quoted(opening Quote_Opening, span Untrusted_Span) (written Quoted_Message) {
  defer func() { Quoted_Message_Invariants(written, "quoted.written") }()
  Quote_Opening_Invariants(opening, "quoted.opening")
  Untrusted_Span_Invariants(span, "quoted.span")
  written = Quoted_Message(opening)
  for index := range len(span) {
    if span[index] < QUOTED_BYTE_MINIMUM {
      written = append(written, QUOTED_BYTE_ABSENT)
      continue
    }
    written = append(written, span[index])
  }
  // The span is untrusted, and so is the message it lands in, thus the whole of it
  // stands stated here rather than the part this quote wrote.
  for index := range len(written) {
    aver.Always(
      written[index] > QUOTED_BYTE_MINIMUM-1,
      "A quoted message holds no byte below a space.",
    )
  }
  return written
}

// Writes the message of the open finding, quoting the heading or the test name it names.
func message_write(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "message_write.checker")
  numbers := checker.Numbers.Values
  message := checker.Message[:0]
  stated := Message(numbers[NUMBER_MESSAGE])
  message = append(message, message_opening(stated)...)
  switch stated {
  case MESSAGE_TEST_ORDER:
    // The name is built from the heading a host wrote, thus it is as untrusted as the
    // heading and stands quoted the same way.
    name := checker.Name[:numbers[NUMBER_NAME_COUNT]]
    opened_count := len(message)
    message = Message_Bytes(quoted(Quote_Opening(message), Untrusted_Span(name)))
    // The leaf is the name past its prefix, and the name already stands quoted, thus
    // the second quote reads what the first wrote rather than scans the file again.
    leaf := message[opened_count+len(TEST_PREFIX):]
    message = append(message, MESSAGE_TEXT_LEAF...)
    message = append(message, leaf...)
  case MESSAGE_HEADING_CHARACTERS, MESSAGE_BLANK_BEFORE, MESSAGE_BLANK_AFTER,
    MESSAGE_ORPHAN, MESSAGE_DUPLICATE, MESSAGE_SECTION_EMPTY, MESSAGE_SECTION_SIZE,
    MESSAGE_SECTION_GAP, MESSAGE_LEAF_NAME, MESSAGE_HEADING_CASE:
    heading := checker.Headings[numbers[NUMBER_FINDING_HEADING]]
    markdown := checker.Markdown[:numbers[NUMBER_MARKDOWN_COUNT]]
    span := Untrusted_Span(markdown[heading.Start:heading.End])
    message = Message_Bytes(quoted(Quote_Opening(message), span))
  }
  message = append(message, message_tail(stated)...)
  numbers[NUMBER_MESSAGE_COUNT] = Number(len(message))
}

// MESSAGE_OPENING_SIZE_MINIMUM is the shortest text one message opens with, which names the
// thing it states before the heading it quotes.
const MESSAGE_OPENING_SIZE_MINIMUM = len(`The heading "`)

// MESSAGE_TEXT_HEADINGS_FULL is what a message states of a contract past its heading table.
const MESSAGE_TEXT_HEADINGS_FULL = "The file has more headings than its heading table holds. " +
  "Write fewer headings."

// MESSAGE_TEXT_HEADING_LEVEL is what a message states of a heading of no admitted level.
const MESSAGE_TEXT_HEADING_LEVEL = `The heading level is not "#" or "###". ` +
  `Write a "#" heading or a "###" heading.`

// MESSAGE_OPENING_SIZE_MAXIMUM is the longest text one message opens with. A message that
// quotes nothing stands whole in its opening, and the heading level states both the fault and
// the two levels a host writes instead, which no other opening outruns.
const MESSAGE_OPENING_SIZE_MAXIMUM = len(MESSAGE_TEXT_HEADING_LEVEL)

// MESSAGE_TAIL_SIZE_MAXIMUM is the longest text one message closes with, which states the fault
// of a heading and the two ways a host writes it differently.
const MESSAGE_TAIL_SIZE_MAXIMUM = len(MESSAGE_TEXT_HEADING_CASE)

// MESSAGE_TEXT_HEADING_CASE is what a message states past the heading that carries a capital
// inside one of its words.
const MESSAGE_TEXT_HEADING_CASE = `" carries a capital after a lower case letter. ` +
  "Start the word with the capital, or spell the word in capitals."

// Message_Opening is the text one message opens with. Every message names what it states, thus
// the text of no bytes that closes one stands nowhere here.
type Message_Opening string

// Message_Opening_Invariants bounds the opening by the message that holds it.
func Message_Opening_Invariants(value Message_Opening, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(
    len(value),
    MESSAGE_OPENING_SIZE_MINIMUM,
    MESSAGE_OPENING_SIZE_MAXIMUM,
  ).
    Ensure()
}

// Message_Part is one fixed piece of a finding's message: the text it opens with, or the text it
// closes with past what it quotes.
type Message_Part string

// Message_Part_Invariants bounds one piece by the message that holds it.
func Message_Part_Invariants(value Message_Part, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Holed_Int(
    len(value),
    MESSAGE_PART_SIZE_MINIMUM,
    MESSAGE_TAIL_SIZE_MAXIMUM,
    MESSAGE_PART_HOLE_ONE,
    MESSAGE_PART_HOLE_TWO,
    MESSAGE_PART_HOLE_TWO,
    MESSAGE_PART_HOLE_TWO,
  ).
    Ensure()
}

// MESSAGE_PART_HOLE_ONE is the one byte no part of a message spends: a part is the whole of a
// sentence, or none of one.
const MESSAGE_PART_HOLE_ONE = 1

// MESSAGE_PART_HOLE_TWO is the two bytes no part of a message spends.
const MESSAGE_PART_HOLE_TWO = 2

// MESSAGE_PART_SIZE_MINIMUM is the empty tail of a message that quotes nothing.
const MESSAGE_PART_SIZE_MINIMUM = 0

// The text one message opens with.
func message_opening(message Message) (text Message_Opening) {
  defer func() { Message_Opening_Invariants(text, "message_opening.text") }()
  Message_Invariants(message, "message_opening.message")
  switch message {
  case MESSAGE_MARKDOWN_ABSENT:
    return "The package has no REQUIREMENTS.md. Add REQUIREMENTS.md."
  case MESSAGE_MARKDOWN_OVERSIZED:
    return "The file is larger than its storage. Write a shorter file."
  case MESSAGE_HEADINGS_ABSENT:
    return "The file states no heading. Write a heading."
  case MESSAGE_CONFLICT_MARKER:
    return MESSAGE_TEXT_CONFLICT_MARKER
  case MESSAGE_HEADING_OVERSIZED:
    return "The heading is longer than 128 bytes. Write a shorter heading."
  case MESSAGE_HEADINGS_FULL:
    return MESSAGE_TEXT_HEADINGS_FULL
  case MESSAGE_HEADING_LEVEL:
    return MESSAGE_TEXT_HEADING_LEVEL
  case MESSAGE_PREAMBLE:
    return "The content is before the first heading. Start the file with a heading."
  case MESSAGE_TEST_ABSENT:
    return "The package has no requirements_test.go. Add requirements_test.go."
  case MESSAGE_TEST_ORDER:
    return MESSAGE_TEXT_TEST_ORDER
  case MESSAGE_ORPHAN:
    return `The "###" heading "`
  case MESSAGE_BLANK_BEFORE:
    return MESSAGE_TEXT_BLANK_BEFORE
  case MESSAGE_BLANK_AFTER:
    return `There is no blank line after the heading "`
  case MESSAGE_HEADING_CHARACTERS, MESSAGE_DUPLICATE, MESSAGE_LEAF_NAME,
    MESSAGE_HEADING_CASE:
    return `The heading "`
  }
  return `The section "`
}

// The text one message ends with, past what it quotes.
func message_tail(message Message) (text Message_Part) {
  defer func() { Message_Part_Invariants(text, "message_tail.text") }()
  Message_Invariants(message, "message_tail.message")
  switch message {
  case MESSAGE_HEADING_CHARACTERS:
    return `" is not spelled with ASCII letters and digits. ` +
      "Write a heading of ASCII letters and digits."
  case MESSAGE_BLANK_BEFORE, MESSAGE_BLANK_AFTER:
    return `". Write a blank line.`
  case MESSAGE_ORPHAN:
    return `" has no parent "#" heading. Put the heading below a "#" heading.`
  case MESSAGE_DUPLICATE:
    return `" is not unique. Write a different heading.`
  case MESSAGE_HEADING_CASE:
    return MESSAGE_TEXT_HEADING_CASE
  case MESSAGE_LEAF_NAME:
    return `" states the test name an earlier heading states. ` +
      "Write a different heading."
  case MESSAGE_SECTION_EMPTY:
    return `" has no body line. Write a body line.`
  case MESSAGE_SECTION_SIZE:
    return `" has more than three lines. Write a maximum of three lines.`
  case MESSAGE_SECTION_GAP:
    return `" has a blank line between two body lines. Remove the blank line.`
  case MESSAGE_TEST_ORDER:
    return MESSAGE_TEXT_TEST_ORDER_TAIL
  }
  return ""
}

// COUNT_MINIMUM is no bytes: a store the caller filled with nothing holds none.
const COUNT_MINIMUM = 0

// Markdown_Count is how many bytes of REQUIREMENTS.md the markdown store holds.
type Markdown_Count Number

// Markdown_Count_Invariants bounds the count by the store it names.
func Markdown_Count_Invariants(value Markdown_Count, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(int(value), COUNT_MINIMUM, MARKDOWN_STORAGE_SIZE).
    Ensure()
}

// Load records how many bytes of REQUIREMENTS.md the caller put in the markdown store.
func Load(checker Checker_Handle, count Markdown_Count) {
  Checker_Handle_Invariants(checker, "load.checker")
  Markdown_Count_Invariants(count, "load.count")
  checker.Numbers.Values[NUMBER_MARKDOWN_COUNT] = Number(count)
}

// Test_Count is how many bytes of requirements_test.go the test store holds.
type Test_Count Number

// Test_Count_Invariants bounds the count by the widest source one check reads. A test file that
// fills the store is past that source, thus the caller loads none of it rather than a part.
func Test_Count_Invariants(value Test_Count, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(int(value), COUNT_MINIMUM, SOURCE_SIZE_MAXIMUM).
    Ensure()
}

// Load_Test records how many bytes of requirements_test.go the caller put in the test store. A
// source past the store has its own diagnostic, thus the caller records none of it and the tests
// go unchecked rather than flagged twice.
func Load_Test(checker Checker_Handle, count Test_Count) {
  Checker_Handle_Invariants(checker, "load_test.checker")
  Test_Count_Invariants(count, "load_test.count")
  checker.Numbers.Values[NUMBER_TEST_COUNT] = Number(count)
}

// PATH_COUNT_MINIMUM is one byte, since the directory a package stands in names something.
const PATH_COUNT_MINIMUM = 1

// PATH_COUNT_MAXIMUM is the widest package directory the check stands on. A package owes a
// contract, thus its directory leaves room for the contract name beside the path it holds.
const PATH_COUNT_MAXIMUM = PATH_STORAGE_SIZE - len("/") - len(TEST_NAME)

// Path_Count is how many bytes of the package directory the path store holds.
type Path_Count Number

// Path_Count_Invariants bounds the count by the store it names.
func Path_Count_Invariants(value Path_Count, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(int(value), PATH_COUNT_MINIMUM, PATH_COUNT_MAXIMUM).
    Ensure()
}

// Load_Path records how many bytes of the package directory the caller put in the path store.
func Load_Path(checker Checker_Handle, count Path_Count) {
  Checker_Handle_Invariants(checker, "load_path.checker")
  Path_Count_Invariants(count, "load_path.count")
  checker.Numbers.Values[NUMBER_PATH_COUNT] = Number(count)
}

// HEADING_PREFIX_NONE opens a line that is no heading the check records.
const HEADING_PREFIX_NONE = 0

// HEADING_PREFIX_BRANCH opens a # heading.
const HEADING_PREFIX_BRANCH = len("# ")

// HEADING_PREFIX_LEAF opens a ### heading.
const HEADING_PREFIX_LEAF = len("### ")

// Heading_Prefix is how many bytes open a line as a heading the check records.
type Heading_Prefix int

// Heading_Prefix_Invariants admits no heading, a # heading, and a ### heading.
func Heading_Prefix_Invariants(value Heading_Prefix, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_3_Int(
    int(value),
    HEADING_PREFIX_NONE,
    HEADING_PREFIX_BRANCH,
    HEADING_PREFIX_LEAF,
  ).
    Ensure()
}

// Rewrite puts a blank line on each side of every heading and takes the blank lines out from
// between two body lines, then loads what it wrote in place of the contract, and reports whether
// the contract changed. It leaves alone a contract whose check stops at a bound, and one whose
// rewrite would outgrow its storage.
func Rewrite(checker Checker_Handle) (rewritten Flag) {
  defer func() { Flag_Invariants(rewritten, "rewrite.rewritten") }()
  Checker_Handle_Invariants(checker, "rewrite.checker")
  numbers, flags := checker.Numbers.Values, checker.Flags.Values
  markdown := checker.Markdown[:numbers[NUMBER_MARKDOWN_COUNT]]
  if len(markdown) == len(checker.Markdown) {
    return false
  }
  if !bool(rewrite_bounded(checker)) {
    return false
  }
  numbers[NUMBER_REWRITE_COUNT], numbers[NUMBER_REWRITE_HELD] = 0, ABSENT
  numbers[NUMBER_LINE_OFFSET] = 0
  flags[FLAG_REWRITE_OPEN], flags[FLAG_REWRITE_BODY], flags[FLAG_REWRITE_FIT] = false, false, true
  // The text after the final newline is a line of its own, thus the last line is the one no
  // newline closes.
  for range MARKDOWN_STORAGE_SIZE {
    line_measure(checker)
    rewrite_line(checker)
    if !bool(rewrite_closed(checker)) {
      break
    }
    numbers[NUMBER_LINE_OFFSET] = numbers[NUMBER_REWRITE_NEXT]
  }
  numbers[NUMBER_LINE_OFFSET] = numbers[NUMBER_MARKDOWN_COUNT]
  rewrite_flush(checker)
  written := checker.Rewritten[:numbers[NUMBER_REWRITE_COUNT]]
  if !flags[FLAG_REWRITE_FIT] {
    return false
  }
  if string(written) == string(markdown) {
    return false
  }
  numbers[NUMBER_MARKDOWN_COUNT] = Number(copy(checker.Markdown, written))
  return true
}

// Reports whether a check of the contract reads every line: a heading past its bound, or one
// past the heading table, stops it there, and the rewrite answers nothing past that.
func rewrite_bounded(checker Checker_Handle) (bounded Flag) {
  defer func() { Flag_Invariants(bounded, "rewrite_bounded.bounded") }()
  Checker_Handle_Invariants(checker, "rewrite_bounded.checker")
  numbers := checker.Numbers.Values
  headings := 0
  numbers[NUMBER_LINE_OFFSET] = 0
  for range MARKDOWN_STORAGE_SIZE {
    line_measure(checker)
    prefix := rewrite_prefix(checker)
    if prefix > HEADING_PREFIX_NONE {
      headings++
      size := int(numbers[NUMBER_LINE_END]-numbers[NUMBER_LINE_OFFSET]) - int(prefix)
      if size > HEADING_SIZE_MAXIMUM {
        return false
      }
    }
    if headings > HEADING_COUNT_MAXIMUM {
      return false
    }
    rewrite_next(checker)
    if !bool(rewrite_closed(checker)) {
      return true
    }
    numbers[NUMBER_LINE_OFFSET] = numbers[NUMBER_REWRITE_NEXT]
  }
  return true
}

// Records where the line after the measured one opens: past its ending, whichever bytes a host
// spent on it.
func rewrite_next(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "rewrite_next.checker")
  numbers := checker.Numbers.Values
  markdown := checker.Markdown[:numbers[NUMBER_MARKDOWN_COUNT]]
  next := int(numbers[NUMBER_LINE_END])
  if next < len(markdown) {
    if markdown[next] == LINE_RETURN {
      next++
    }
  }
  if next < len(markdown) {
    next++
  }
  numbers[NUMBER_REWRITE_NEXT] = Number(next)
}

// Reports whether a newline closes the measured line, thus another line follows it.
func rewrite_closed(checker Checker_Handle) (closed Flag) {
  defer func() { Flag_Invariants(closed, "rewrite_closed.closed") }()
  Checker_Handle_Invariants(checker, "rewrite_closed.checker")
  numbers := checker.Numbers.Values
  next := numbers[NUMBER_REWRITE_NEXT]
  if next == numbers[NUMBER_LINE_END] {
    return false
  }
  return Flag(checker.Markdown[next-1] == '\n')
}

// Reports how many bytes open the measured line as a heading the check records.
func rewrite_prefix(checker Checker_Handle) (prefix Heading_Prefix) {
  defer func() { Heading_Prefix_Invariants(prefix, "rewrite_prefix.prefix") }()
  Checker_Handle_Invariants(checker, "rewrite_prefix.checker")
  numbers := checker.Numbers.Values
  line := string(checker.Markdown[numbers[NUMBER_LINE_OFFSET]:numbers[NUMBER_LINE_END]])
  if len(line) > HEADING_PREFIX_LEAF-1 {
    if line[:HEADING_PREFIX_LEAF] == "### " {
      return Heading_Prefix(HEADING_PREFIX_LEAF)
    }
  }
  if len(line) > HEADING_PREFIX_BRANCH-1 {
    if line[:HEADING_PREFIX_BRANCH] == "# " {
      return Heading_Prefix(HEADING_PREFIX_BRANCH)
    }
  }
  return Heading_Prefix(HEADING_PREFIX_NONE)
}

// Writes one measured line: a heading with its blank lines, a blank line after a body line held
// until the line after it shows whether it falls between two body lines, and any other line as
// it stands. A line opening with a mark closes the open section, and only a heading the check
// records opens the next one.
func rewrite_line(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "rewrite_line.checker")
  numbers, flags := checker.Numbers.Values, checker.Flags.Values
  start, end := numbers[NUMBER_LINE_OFFSET], numbers[NUMBER_LINE_END]
  rewrite_next(checker)
  heading := rewrite_prefix(checker) > HEADING_PREFIX_NONE
  marked := start < end
  if marked {
    marked = checker.Markdown[start] == '#'
  }
  if marked {
    rewrite_flush(checker)
    flags[FLAG_REWRITE_OPEN], flags[FLAG_REWRITE_BODY] = Flag(heading), false
  }
  if heading {
    rewrite_heading(checker)
    return
  }
  numbers[NUMBER_SCAN_START], numbers[NUMBER_SCAN_END] = start, end
  scan_blank(checker)
  blank := flags[FLAG_SCAN_RESULT]
  if flags[FLAG_REWRITE_OPEN] {
    if blank {
      if flags[FLAG_REWRITE_BODY] {
        if numbers[NUMBER_REWRITE_HELD] == ABSENT {
          numbers[NUMBER_REWRITE_HELD] = start
        }
        return
      }
    }
    if !blank {
      numbers[NUMBER_REWRITE_HELD] = ABSENT
      flags[FLAG_REWRITE_BODY] = true
    }
  }
  numbers[NUMBER_SCAN_START], numbers[NUMBER_SCAN_END] = start, numbers[NUMBER_REWRITE_NEXT]
  rewrite_copy(checker)
}

// Writes the blank lines held after a body line: a section closes on them, thus they fall
// between no two body lines.
func rewrite_flush(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "rewrite_flush.checker")
  numbers := checker.Numbers.Values
  if numbers[NUMBER_REWRITE_HELD] == ABSENT {
    return
  }
  numbers[NUMBER_SCAN_START] = numbers[NUMBER_REWRITE_HELD]
  numbers[NUMBER_SCAN_END] = numbers[NUMBER_LINE_OFFSET]
  numbers[NUMBER_REWRITE_HELD] = ABSENT
  rewrite_copy(checker)
}

// Writes one heading, with the blank line before it that the text written so far lacks, and the
// blank line after it that the line after it lacks.
func rewrite_heading(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "rewrite_heading.checker")
  numbers := checker.Numbers.Values
  markdown := checker.Markdown[:numbers[NUMBER_MARKDOWN_COUNT]]
  if !bool(rewrite_preceded(checker)) {
    rewrite_blank(checker)
  }
  numbers[NUMBER_SCAN_START] = numbers[NUMBER_LINE_OFFSET]
  numbers[NUMBER_SCAN_END] = numbers[NUMBER_REWRITE_NEXT]
  rewrite_copy(checker)
  // A heading the file ends on carries no newline, thus the newline it owes closes it.
  if !bool(rewrite_closed(checker)) {
    rewrite_blank(checker)
    return
  }
  after := int(numbers[NUMBER_REWRITE_NEXT])
  if after < len(markdown) {
    if markdown[after] == LINE_RETURN {
      after++
    }
  }
  if after < len(markdown) {
    if markdown[after] != '\n' {
      rewrite_blank(checker)
    }
  }
}

// Reports whether the text written so far ends on a blank line, which a heading written next
// stands after: a heading on the first line stands after none.
func rewrite_preceded(checker Checker_Handle) (preceded Flag) {
  defer func() { Flag_Invariants(preceded, "rewrite_preceded.preceded") }()
  Checker_Handle_Invariants(checker, "rewrite_preceded.checker")
  written := checker.Rewritten[:checker.Numbers.Values[NUMBER_REWRITE_COUNT]]
  if len(written) == 0 {
    return false
  }
  position := len(written) - 2
  if position < 0 {
    return true
  }
  if written[position] == LINE_RETURN {
    position--
  }
  if position < 0 {
    return true
  }
  return Flag(written[position] == '\n')
}

// Writes one blank line, ending the way the measured line ends, or with a newline where the line
// ends the file.
func rewrite_blank(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "rewrite_blank.checker")
  numbers := checker.Numbers.Values
  if !bool(rewrite_closed(checker)) {
    rewrite_newline(checker)
    return
  }
  numbers[NUMBER_SCAN_START] = numbers[NUMBER_LINE_END]
  numbers[NUMBER_SCAN_END] = numbers[NUMBER_REWRITE_NEXT]
  rewrite_copy(checker)
}

// Writes the span the scan names, or records that the rewrite storage holds no room for it. The
// storage keeps its last byte free, since a contract that fills its store is past its bound.
func rewrite_copy(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "rewrite_copy.checker")
  numbers := checker.Numbers.Values
  span := checker.Markdown[numbers[NUMBER_SCAN_START]:numbers[NUMBER_SCAN_END]]
  count := int(numbers[NUMBER_REWRITE_COUNT])
  if count+len(span) > len(checker.Rewritten)-1 {
    checker.Flags.Values[FLAG_REWRITE_FIT] = false
    return
  }
  numbers[NUMBER_REWRITE_COUNT] += Number(copy(checker.Rewritten[count:], span))
}

// Writes one newline, for a heading the file ends on.
func rewrite_newline(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "rewrite_newline.checker")
  numbers := checker.Numbers.Values
  count := int(numbers[NUMBER_REWRITE_COUNT])
  if count+len("\n") > len(checker.Rewritten)-1 {
    checker.Flags.Values[FLAG_REWRITE_FIT] = false
    return
  }
  checker.Rewritten[count] = '\n'
  numbers[NUMBER_REWRITE_COUNT]++
}

// Fixable reports whether the open finding is one Rewrite answers: a blank line missing around a
// heading, or one standing between two body lines.
func Fixable(checker Checker_Handle) (fixable Flag) {
  defer func() { Flag_Invariants(fixable, "fixable.fixable") }()
  Checker_Handle_Invariants(checker, "fixable.checker")
  switch Message(checker.Numbers.Values[NUMBER_MESSAGE]) {
  case MESSAGE_BLANK_BEFORE, MESSAGE_BLANK_AFTER, MESSAGE_SECTION_GAP:
    return true
  }
  return false
}

// MESSAGE_TEXT_CONFLICT_MARKER is the message of a line that opens on a merge conflict marker.
const MESSAGE_TEXT_CONFLICT_MARKER = "The file has a conflict marker. " +
  "Resolve the conflict and remove the marker."

// CONFLICT_MARKER_SIZE is the bytes of one merge conflict marker: one sentinel, seven times.
const CONFLICT_MARKER_SIZE = len("<<<<<<<")

// LINE_OPENING_SIZE_MINIMUM is the opening of an empty line.
const LINE_OPENING_SIZE_MINIMUM = 0

// Line_Opening is the first bytes of one line, as many as a conflict marker spans.
type Line_Opening string

// Line_Opening_Invariants bounds the opening by the marker it is read for.
func Line_Opening_Invariants(value Line_Opening, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(
    len(value),
    LINE_OPENING_SIZE_MINIMUM,
    CONFLICT_MARKER_SIZE,
  ).
    Ensure()
}

// Conflict_Marker reports whether a line opens on a merge conflict marker: seven repeats of <, >,
// %, +, or backslash, the sentinels a VCS writes into a file it cannot merge.
func Conflict_Marker(opening Line_Opening) (marked Flag) {
  defer func() { Flag_Invariants(marked, "conflict_marker.marked") }()
  Line_Opening_Invariants(opening, "conflict_marker.opening")
  if len(opening) < CONFLICT_MARKER_SIZE {
    return false
  }
  switch opening {
  case "<<<<<<<", ">>>>>>>", "%%%%%%%", "+++++++", "\\\\\\\\\\\\\\":
    return true
  }
  return false
}
