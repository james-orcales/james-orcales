// Package build answers whether one file stands in one build. It reads the name of the file and
// the constraint the file states, and it reads no file system and no environment: the caller owns
// the name, the text, and the target, thus one read allocates nothing and states no default.
package build

import (
	"unsafe"

	"local/james-orcales/shared/go/token"
	"local/james-orcales/shared/sim/aver/default"
	"local/james-orcales/shared/sim/nbio"
	"local/james-orcales/shared/sim/time"
)

// TEXT_SIZE_MAXIMUM is the widest constraint one read admits, which is wider than any constraint
// the toolchain itself writes.
const TEXT_SIZE_MAXIMUM = 4096

// DEPTH_MAXIMUM caps the brackets one constraint nests, which is deeper than any constraint a
// reader follows.
const DEPTH_MAXIMUM = 32

// TAG_COUNT_MAXIMUM caps the tags one target names.
const TAG_COUNT_MAXIMUM = 32

// WORD_SYSTEM holds the operating system one target names.
const WORD_SYSTEM = 0

// WORD_ARCHITECTURE holds the architecture one target names.
const WORD_ARCHITECTURE = 1

// WORD_SLOT_COUNT is the word slot count one target holds.
const WORD_SLOT_COUNT = 2

// COUNT_TAG holds how many tags one target names.
const COUNT_TAG = 0

// COUNT_PLACE holds the byte of the text one read stands on.
const COUNT_PLACE = 1

// COUNT_DEPTH holds the brackets the read stands inside of.
const COUNT_DEPTH = 2

// COUNT_SLOT_COUNT is the counter count one state holds.
const COUNT_SLOT_COUNT = 3

// COUNT_MINIMUM is the smallest count one read holds.
const COUNT_MINIMUM = 0

// COUNT_MAXIMUM is the largest count one read holds, which is the widest text it reads.
const COUNT_MAXIMUM = TEXT_SIZE_MAXIMUM

// PLACE_MINIMUM is the first byte of one name.
const PLACE_MINIMUM = 0

// PLACE_MAXIMUM is the byte past the widest name one read admits, which is the widest source one
// scan of Go text reads.
const PLACE_MAXIMUM = token.SOURCE_SIZE_MAXIMUM

// SOURCE_TEXT holds the text one read runs over.
const SOURCE_TEXT = 0

// SOURCE_TAG holds the tag one read stands on.
const SOURCE_TAG = 1

// SOURCE_WORD holds the word one read compares the tag against.
const SOURCE_WORD = 2

// SOURCE_SLOT_COUNT is the text slot count one state holds.
const SOURCE_SLOT_COUNT = 3

// SYMBOL_HELD holds the byte one read stands on.
const SYMBOL_HELD = 0

// SYMBOL_SLOT_COUNT is the byte slot count one state holds.
const SYMBOL_SLOT_COUNT = 1

// PLACE_OPENING holds the byte one field of a name opens at.
const PLACE_OPENING = 0

// PLACE_FIELD holds the byte the final field of a name opens at.
const PLACE_FIELD = 1

// PLACE_SLOT_COUNT is the name place slot count one state holds.
const PLACE_SLOT_COUNT = 2

// HEAD_SLOT holds the text one read asks a line to open with.
const HEAD_SLOT = 0

// HEAD_SLOT_COUNT is the head slot count one state holds.
const HEAD_SLOT_COUNT = 1

// FLAG_FAILED marks a read that met a text it cannot read.
const FLAG_FAILED = 0

// FLAG_SLOT_COUNT is the flag count one state holds.
const FLAG_SLOT_COUNT = 1

// UNIX_WORD is the tag that holds on every operating system Go calls one.
const UNIX_WORD = "unix"

// TEST_WORD is the word a name of a test file closes with, which names no system.
const TEST_WORD = "test"

// Boolean is a true or false report about one build.
type Boolean bool

// Boolean_Invariants states both reports as obligations.
func Boolean_Invariants(value Boolean, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The build report is true.").
		Ensure()
}

// Count names a byte of one text, or a count of the tags or the brackets one read holds.
type Count int32

// Count_Invariants states every count one read holds.
func Count_Invariants(value Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), COUNT_MINIMUM, COUNT_MAXIMUM).
		Ensure()
}

// Place names one byte of the name of a file.
type Place int32

// Place_Invariants states every byte one name holds.
func Place_Invariants(value Place, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), PLACE_MINIMUM, PLACE_MAXIMUM).
		Ensure()
}

// System is the operating-system word one target names.
type System token.Source

// System_Invariants states every bounded operating-system word.
func System_Invariants(value System, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), token.SOURCE_SIZE_MINIMUM, token.SOURCE_SIZE_MAXIMUM).
		Ensure()
}

// Architecture is the machine word one target names.
type Architecture token.Source

// Architecture_Invariants states every bounded machine word.
func Architecture_Invariants(value Architecture, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), token.SOURCE_SIZE_MINIMUM, token.SOURCE_SIZE_MAXIMUM).
		Ensure()
}

// Target_Words keeps unlike system words in named storage.
type Target_Words struct {
	// System stays independent because a file may constrain only the operating system.
	System System
	// Architecture stays independent because a file may constrain only the machine.
	Architecture Architecture
}

// Target_Words_Invariants composes each system-word domain once.
func Target_Words_Invariants(value Target_Words, namespace aver.Namespace) {
	System_Invariants(value.System, namespace)
	Architecture_Invariants(value.Architecture, namespace)
}

// Target_Tags keeps bounded build tags in caller storage.
type Target_Tags []token.Source

// Target_Tags_Invariants fixes tag storage and prevents growth.
func Target_Tags_Invariants(value Target_Tags, _ aver.Namespace) {
	aver.Always(len(value) == TAG_COUNT_MAXIMUM, "Target tags have complete length.")
	aver.Always(cap(value) == TAG_COUNT_MAXIMUM, "Target tags cannot grow.")
}

// Target_Tag_Storage keeps caller-owned tags distinct from their view.
type Target_Tag_Storage struct {
	// Values makes ownership explicit at the package boundary.
	Values Target_Tags
}

// Target_Tag_Storage_Invariants composes optional-tag storage.
func Target_Tag_Storage_Invariants(value Target_Tag_Storage, namespace aver.Namespace) {
	Target_Tags_Invariants(value.Values, namespace)
}

// Tag_Count is how many optional tags one target names.
type Tag_Count Count

// Tag_Count_Invariants states the complete optional-tag count.
func Tag_Count_Invariants(value Tag_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), COUNT_MINIMUM, TAG_COUNT_MAXIMUM).
		Ensure()
}

// Target_Counts keeps only the count a target owns.
type Target_Counts struct {
	// Tag excludes parser cursors that belong to Build instead.
	Tag Tag_Count
}

// Target_Counts_Invariants composes the optional-tag count.
func Target_Counts_Invariants(value Target_Counts, namespace aver.Namespace) {
	Tag_Count_Invariants(value.Tag, namespace)
}

// Text_Place is the byte of a constraint one read stands on.
type Text_Place Count

// Text_Place_Invariants states every byte of a bounded constraint.
func Text_Place_Invariants(value Text_Place, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), COUNT_MINIMUM, TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Constraint_Depth is the bracket depth one read stands inside.
type Constraint_Depth Count

// Constraint_Depth_Invariants states every admitted bracket depth.
func Constraint_Depth_Invariants(value Constraint_Depth, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), COUNT_MINIMUM, DEPTH_MAXIMUM).
		Ensure()
}

// Build_Counts keeps unlike reader cursors in named storage.
type Build_Counts struct {
	// Place survives recursive reads that temporarily advance the text.
	Place Text_Place
	// Depth survives recursive reads that temporarily enter another group.
	Depth Constraint_Depth
}

// Build_Counts_Invariants composes each reader-cursor domain once.
func Build_Counts_Invariants(value Build_Counts, namespace aver.Namespace) {
	Text_Place_Invariants(value.Place, namespace)
	Constraint_Depth_Invariants(value.Depth, namespace)
}

// Constraint_Text is the complete constraint one reader scans.
type Constraint_Text token.Source

// Constraint_Text_Invariants states every bounded source view.
func Constraint_Text_Invariants(value Constraint_Text, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), token.SOURCE_SIZE_MINIMUM, token.SOURCE_SIZE_MAXIMUM).
		Ensure()
}

// Tag is the constraint word one reader currently holds.
type Tag token.Source

// Tag_Invariants states every bounded tag view.
func Tag_Invariants(value Tag, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), token.SOURCE_SIZE_MINIMUM, token.SOURCE_SIZE_MAXIMUM).
		Ensure()
}

// Word is the target word one reader compares against.
type Word token.Source

// Word_Invariants states every bounded target-word view.
func Word_Invariants(value Word, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), token.SOURCE_SIZE_MINIMUM, token.SOURCE_SIZE_MAXIMUM).
		Ensure()
}

// Build_Sources keeps unlike reader views in named storage.
type Build_Sources struct {
	// Text survives the views taken from it.
	Text Constraint_Text
	// Tag separates the current term from the complete constraint.
	Tag Tag
	// Word separates target input from the term being compared.
	Word Word
}

// Build_Sources_Invariants composes each reader-view domain once.
func Build_Sources_Invariants(value Build_Sources, namespace aver.Namespace) {
	Constraint_Text_Invariants(value.Text, namespace)
	Tag_Invariants(value.Tag, namespace)
	Word_Invariants(value.Word, namespace)
}

// SYMBOL_MINIMUM admits the zero value before one read begins.
const SYMBOL_MINIMUM = 0

// SYMBOL_MAXIMUM is the largest byte one constraint can state.
const SYMBOL_MAXIMUM = 255

// Held_Symbol is the byte one reader currently holds.
type Held_Symbol byte

// Held_Symbol_Invariants states every byte.
func Held_Symbol_Invariants(value Held_Symbol, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), SYMBOL_MINIMUM, SYMBOL_MAXIMUM).
		Ensure()
}

// Build_Symbols keeps the active byte in named storage.
type Build_Symbols struct {
	// Held survives the lookahead that compares the second sign.
	Held Held_Symbol
}

// Build_Symbols_Invariants composes the active-byte domain.
func Build_Symbols_Invariants(value Build_Symbols, namespace aver.Namespace) {
	Held_Symbol_Invariants(value.Held, namespace)
}

// HEAD_SIZE_MAXIMUM is the widest line prefix one build reader searches for.
const HEAD_SIZE_MAXIMUM = len(BUILD_HEAD)

// Head is one line prefix a build reader searches for.
type Head string

// Head_Invariants admits absence and every bounded line prefix.
func Head_Invariants(value Head, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), PLACE_MINIMUM, HEAD_SIZE_MAXIMUM).
		Ensure()
}

// Build_Heads keeps the active line prefix in named storage.
type Build_Heads struct {
	// Value survives the comparison of the line behind it.
	Value Head
}

// Build_Heads_Invariants composes the line-prefix domain.
func Build_Heads_Invariants(value Build_Heads, namespace aver.Namespace) {
	Head_Invariants(value.Value, namespace)
}

// Opening_Place is the byte at which the first name field opens.
type Opening_Place Place

// Opening_Place_Invariants states every byte of a bounded name.
func Opening_Place_Invariants(value Opening_Place, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), PLACE_MINIMUM, PLACE_MAXIMUM).
		Ensure()
}

// Field_Place is the byte at which the final name field opens.
type Field_Place Place

// Field_Place_Invariants states every byte of a bounded name.
func Field_Place_Invariants(value Field_Place, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), PLACE_MINIMUM, PLACE_MAXIMUM).
		Ensure()
}

// Build_Places keeps unlike name positions in named storage.
type Build_Places struct {
	// Opening survives the backwards scan for the final field.
	Opening Opening_Place
	// Field separates the scan answer from its lower bound.
	Field Field_Place
}

// Build_Places_Invariants composes each name-position domain once.
func Build_Places_Invariants(value Build_Places, namespace aver.Namespace) {
	Opening_Place_Invariants(value.Opening, namespace)
	Field_Place_Invariants(value.Field, namespace)
}

// Failed is whether one reader met malformed text.
type Failed Boolean

// Failed_Invariants states both reader outcomes.
func Failed_Invariants(value Failed, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The build reader fails malformed text.").
		Ensure()
}

// Build_Flags keeps failure separate from returned truth values.
type Build_Flags struct {
	// Failed survives nested reads whose truth value differs from validity.
	Failed Failed
}

// Build_Flags_Invariants composes the reader-failure domain.
func Build_Flags_Invariants(value Build_Flags, namespace aver.Namespace) {
	Failed_Invariants(value.Failed, namespace)
}

// Target_Fields states the concrete target layout independently of its validated view.
type Target_Fields struct {
	// Words holds the operating system and the architecture the build names.
	Words Target_Words
	// Tags holds the tags the build states beyond the two words.
	Tags Target_Tag_Storage
	// Counts holds how many of those tags the build states.
	Counts Target_Counts
}

// Target_Fields_Invariants composes every concrete target field once.
func Target_Fields_Invariants(value Target_Fields, namespace aver.Namespace) {
	Target_Words_Invariants(value.Words, namespace)
	Target_Tag_Storage_Invariants(value.Tags, namespace)
	Target_Counts_Invariants(value.Counts, namespace)
}

// Target_Words_Stored prevents one read from proving every caller word length.
type Target_Words_Stored interface{}

// Target_Words_Stored_Invariants fixes target-word representation.
func Target_Words_Stored_Invariants(value Target_Words_Stored, _ aver.Namespace) {
	_, valid := value.(Target_Words)
	aver.Always(valid == (value != nil), "Target words have expected storage type.")
}

// Target_Counts_Stored prevents one read from proving every optional-tag count.
type Target_Counts_Stored interface{}

// Target_Counts_Stored_Invariants fixes target-count representation.
func Target_Counts_Stored_Invariants(value Target_Counts_Stored, _ aver.Namespace) {
	_, valid := value.(Target_Counts)
	aver.Always(valid == (value != nil), "Target counts have expected storage type.")
}

// Target is one validated view over caller-owned target storage.
type Target Target_Fields

// Target_Invariants fixes tag storage without proving stale caller values.
func Target_Invariants(value Target, namespace aver.Namespace) {
	Target_Words_Stored_Invariants(Target_Words_Stored(value.Words), namespace)
	Target_Tag_Storage_Invariants(value.Tags, namespace)
	Target_Counts_Stored_Invariants(Target_Counts_Stored(value.Counts), namespace)
}

// Target_Handle gives caller-owned target storage one identity.
type Target_Handle *Target

// Target_Handle_Invariants composes a present target.
func Target_Handle_Invariants(value Target_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Target_Invariants(*value, namespace)
}

// Build_Fields states the concrete reader layout independently of its validated view.
type Build_Fields struct {
	// Counts holds the byte the read stands on and the brackets it stands inside of.
	Counts Build_Counts
	// Sources holds the text one read runs over and the tag it stands on.
	Sources Build_Sources
	// Symbols holds the byte one read stands on.
	Symbols Build_Symbols
	// Heads holds the text one read asks a line to open with.
	Heads Build_Heads
	// Places holds the bytes one read of a name stands between.
	Places Build_Places
	// Flags holds what the read met.
	Flags Build_Flags
}

// Build_Fields_Invariants composes every concrete reader field once.
func Build_Fields_Invariants(value Build_Fields, namespace aver.Namespace) {
	Build_Counts_Invariants(value.Counts, namespace)
	Build_Sources_Invariants(value.Sources, namespace)
	Build_Symbols_Invariants(value.Symbols, namespace)
	Build_Heads_Invariants(value.Heads, namespace)
	Build_Places_Invariants(value.Places, namespace)
	Build_Flags_Invariants(value.Flags, namespace)
}

// Build_Counts_Stored keeps one call from proving every cursor value.
type Build_Counts_Stored interface{}

// Build_Counts_Stored_Invariants fixes reader-count representation.
func Build_Counts_Stored_Invariants(value Build_Counts_Stored, _ aver.Namespace) {
	_, valid := value.(Build_Counts)
	aver.Always(valid == (value != nil), "Build counts have expected storage type.")
}

// Build_Sources_Stored keeps one call from proving every source length.
type Build_Sources_Stored interface{}

// Build_Sources_Stored_Invariants fixes reader-source representation.
func Build_Sources_Stored_Invariants(value Build_Sources_Stored, _ aver.Namespace) {
	_, valid := value.(Build_Sources)
	aver.Always(valid == (value != nil), "Build sources have expected storage type.")
}

// Build_Symbols_Stored keeps one call from proving every held byte.
type Build_Symbols_Stored interface{}

// Build_Symbols_Stored_Invariants fixes reader-symbol representation.
func Build_Symbols_Stored_Invariants(value Build_Symbols_Stored, _ aver.Namespace) {
	_, valid := value.(Build_Symbols)
	aver.Always(valid == (value != nil), "Build symbols have expected storage type.")
}

// Build_Heads_Stored keeps one call from proving every line-prefix length.
type Build_Heads_Stored interface{}

// Build_Heads_Stored_Invariants fixes reader-head representation.
func Build_Heads_Stored_Invariants(value Build_Heads_Stored, _ aver.Namespace) {
	_, valid := value.(Build_Heads)
	aver.Always(valid == (value != nil), "Build heads have expected storage type.")
}

// Build_Places_Stored keeps one call from proving every name place.
type Build_Places_Stored interface{}

// Build_Places_Stored_Invariants fixes reader-place representation.
func Build_Places_Stored_Invariants(value Build_Places_Stored, _ aver.Namespace) {
	_, valid := value.(Build_Places)
	aver.Always(valid == (value != nil), "Build places have expected storage type.")
}

// Build_Flags_Stored keeps one call from proving both mutable outcomes.
type Build_Flags_Stored interface{}

// Build_Flags_Stored_Invariants fixes reader-flag representation.
func Build_Flags_Stored_Invariants(value Build_Flags_Stored, _ aver.Namespace) {
	_, valid := value.(Build_Flags)
	aver.Always(valid == (value != nil), "Build flags have expected storage type.")
}

// Build is the caller-owned state one read of a constraint runs through. The text and the target
// stay with the caller, thus the state holds the place the read stands at and nothing else.
type Build Build_Fields

// Build_Invariants fixes reader representation without proving stale mutable values.
func Build_Invariants(value Build, namespace aver.Namespace) {
	Build_Counts_Stored_Invariants(Build_Counts_Stored(value.Counts), namespace)
	Build_Sources_Stored_Invariants(Build_Sources_Stored(value.Sources), namespace)
	Build_Symbols_Stored_Invariants(Build_Symbols_Stored(value.Symbols), namespace)
	Build_Heads_Stored_Invariants(Build_Heads_Stored(value.Heads), namespace)
	Build_Places_Stored_Invariants(Build_Places_Stored(value.Places), namespace)
	Build_Flags_Stored_Invariants(Build_Flags_Stored(value.Flags), namespace)
}

// Read distinguishes accepted input from one reader refused.
type Read Boolean

// Read_Invariants states both reader outcomes.
func Read_Invariants(value Read, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Reader accepts input.").
		Ensure()
}

// Read_Result keeps membership and accepted input at one output boundary.
type Read_Result struct {
	// Held remains separate because valid input may stand outside target build.
	Held Boolean
	// OK distinguishes rejected input from valid input that does not hold.
	OK Read
}

// Read_Result_Invariants composes one bounded reader outcome.
func Read_Result_Invariants(value Read_Result, namespace aver.Namespace) {
	Boolean_Invariants(value.Held, namespace)
	Read_Invariants(value.OK, namespace)
}

// Build_Handle gives caller-owned reader storage one identity.
type Build_Handle *Build

// Build_Handle_Invariants composes present reader storage.
func Build_Handle_Invariants(value Build_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Build_Invariants(*value, namespace)
}

// Holds_Constraint reports whether the constraint one file states holds in one target, and whether
// the reader read it. A text the reader refuses holds nothing, thus a caller that meets one keeps
// the file out of every build.
func Holds_Constraint(
	subject Build_Handle, one Target_Handle, text token.Source,
) (result Read_Result) {
	defer func() { Read_Result_Invariants(result, "holds_constraint.result") }()
	Build_Handle_Invariants(subject, "holds_constraint.subject")
	Target_Handle_Invariants(one, "holds_constraint.one")
	token.Source_Invariants(text, "holds_constraint.text")
	if len(text) > TEXT_SIZE_MAXIMUM {
		return Read_Result{Held: false, OK: false}
	}
	subject.Sources.Text = Constraint_Text(text)
	subject.Counts.Place = COUNT_MINIMUM
	subject.Counts.Depth = COUNT_MINIMUM
	subject.Flags.Failed = false
	result.Held = read_join(subject, one)
	skip_spaces(subject)
	if int(subject.Counts.Place) != len(text) {
		return Read_Result{Held: false, OK: false}
	}
	if bool(subject.Flags.Failed) {
		return Read_Result{Held: false, OK: false}
	}
	result.OK = true
	return result
}

// Reads the tags one join of constraints holds, which is every tag either side of the two bars
// holds.
func read_join(subject Build_Handle, one Target_Handle) (held Boolean) {
	defer func() { Boolean_Invariants(held, "read_join.held") }()
	Build_Handle_Invariants(subject, "read_join.subject")
	Target_Handle_Invariants(one, "read_join.one")
	held = read_meet(subject, one)
	for range TEXT_SIZE_MAXIMUM {
		subject.Symbols.Held = '|'
		if !bool(states_sign(subject)) {
			return held
		}
		if bool(read_meet(subject, one)) {
			held = true
		}
	}
	return held
}

// Reads the tags one meeting of constraints holds, which is every tag both sides of the two signs
// hold.
func read_meet(subject Build_Handle, one Target_Handle) (held Boolean) {
	defer func() { Boolean_Invariants(held, "read_meet.held") }()
	Build_Handle_Invariants(subject, "read_meet.subject")
	Target_Handle_Invariants(one, "read_meet.one")
	held = read_term(subject, one)
	for range TEXT_SIZE_MAXIMUM {
		subject.Symbols.Held = '&'
		if !bool(states_sign(subject)) {
			return held
		}
		if !bool(read_term(subject, one)) {
			held = false
		}
	}
	return held
}

// Reads one term of a constraint: a tag, a term the exclamation reverses, or a group of terms
// between brackets.
func read_term(subject Build_Handle, one Target_Handle) (held Boolean) {
	defer func() { Boolean_Invariants(held, "read_term.held") }()
	Build_Handle_Invariants(subject, "read_term.subject")
	Target_Handle_Invariants(one, "read_term.one")
	skip_spaces(subject)
	if bool(subject.Flags.Failed) {
		return false
	}
	text := subject.Sources.Text
	place := int(subject.Counts.Place)
	if place == len(text) {
		subject.Flags.Failed = true
		return false
	}
	if text[place] == '!' {
		subject.Counts.Place = Text_Place(place + 1)
		return !read_term(subject, one)
	}
	if text[place] == '(' {
		return read_group(subject, one)
	}
	take_tag(subject)
	if bool(subject.Flags.Failed) {
		return false
	}
	return holds_tag(subject, one)
}

// Reads one group of terms and the bracket that closes it.
func read_group(subject Build_Handle, one Target_Handle) (held Boolean) {
	defer func() { Boolean_Invariants(held, "read_group.held") }()
	Build_Handle_Invariants(subject, "read_group.subject")
	Target_Handle_Invariants(one, "read_group.one")
	depth := subject.Counts.Depth
	if int(depth) >= DEPTH_MAXIMUM {
		subject.Flags.Failed = true
		return false
	}
	subject.Counts.Depth = depth + 1
	subject.Counts.Place = subject.Counts.Place + 1
	held = read_join(subject, one)
	skip_spaces(subject)
	text := subject.Sources.Text
	place := int(subject.Counts.Place)
	if place == len(text) {
		subject.Flags.Failed = true
		return false
	}
	if text[place] != ')' {
		subject.Flags.Failed = true
		return false
	}
	subject.Counts.Place = Text_Place(place + 1)
	subject.Counts.Depth = depth
	return held
}

// Reads the tag the text stands on into the tag slot. A text that states no tag where one belongs
// fails the read.
func take_tag(subject Build_Handle) {
	Build_Handle_Invariants(subject, "take_tag.subject")
	text := subject.Sources.Text
	opening := int(subject.Counts.Place)
	place := opening
	for place < len(text) {
		subject.Symbols.Held = Held_Symbol(text[place])
		if !bool(names_tag(subject)) {
			break
		}
		place = place + 1
	}
	if place == opening {
		subject.Flags.Failed = true
		subject.Sources.Tag = nil
		return
	}
	subject.Sources.Tag = Tag(text[opening:place])
	subject.Counts.Place = Text_Place(place)
}

// Reports whether the byte the tag slot holds stands inside a tag, which is a letter, a digit, an
// underscore, or a period.
func names_tag(subject Build_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "names_tag.yes") }()
	Build_Handle_Invariants(subject, "names_tag.subject")
	value := subject.Symbols.Held
	if value == '_' {
		return true
	}
	if value == '.' {
		return true
	}
	if value >= '0' {
		if value < ':' {
			return true
		}
	}
	if value >= 'a' {
		if value < '{' {
			return true
		}
	}
	if value >= 'A' {
		return Boolean(value < '[')
	}
	return false
}

// Reports whether the sign the read stands on is the one stated twice, and steps past it where it
// stands. Every sign this constraint language states writes its byte twice.
func states_sign(subject Build_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "states_sign.yes") }()
	Build_Handle_Invariants(subject, "states_sign.subject")
	skip_spaces(subject)
	value := subject.Symbols.Held
	text := subject.Sources.Text
	place := int(subject.Counts.Place)
	if place+2 > len(text) {
		return false
	}
	if text[place] != byte(value) {
		return false
	}
	if text[place+1] != byte(value) {
		return false
	}
	subject.Counts.Place = Text_Place(place + 2)
	return true
}

// Steps the read past the spaces and the tabs the author wrote between terms.
func skip_spaces(subject Build_Handle) {
	Build_Handle_Invariants(subject, "skip_spaces.subject")
	text := subject.Sources.Text
	place := int(subject.Counts.Place)
	for place < len(text) {
		if text[place] == ' ' {
			place = place + 1
			continue
		}
		if text[place] != '\t' {
			break
		}
		place = place + 1
	}
	subject.Counts.Place = Text_Place(place)
}

// Reports whether the tag the read stands on holds in one target: the operating system it names,
// the architecture it names, one of the tags it states, or the word every Unix system answers to.
func holds_tag(subject Build_Handle, one Target_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "holds_tag.yes") }()
	Build_Handle_Invariants(subject, "holds_tag.subject")
	Target_Handle_Invariants(one, "holds_tag.one")
	subject.Sources.Word = Word(one.Words.System)
	if string(subject.Sources.Tag) == UNIX_WORD {
		return names_unix(subject)
	}
	if bool(same_word(subject)) {
		return true
	}
	subject.Sources.Word = Word(one.Words.Architecture)
	if bool(same_word(subject)) {
		return true
	}
	for slot := range int(one.Counts.Tag) {
		subject.Sources.Word = Word(one.Tags.Values[slot])
		if bool(same_word(subject)) {
			return true
		}
	}
	return false
}

// Reports whether the tag one read stands on states the word it compares against. A word of no
// bytes names nothing, thus a target that names no system holds no tag at all.
func same_word(subject Build_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "same_word.yes") }()
	Build_Handle_Invariants(subject, "same_word.subject")
	one := subject.Sources.Tag
	if len(one) == 0 {
		return false
	}
	return Boolean(string(one) == string(subject.Sources.Word))
}

// Names_System reports whether the name of one file states the build one target names. A name
// closing at a known system, a known architecture, or both stands in that build alone, and every
// other name stands in every build.
func Names_System(subject Build_Handle, one Target_Handle, name token.Source) (held Boolean) {
	defer func() { Boolean_Invariants(held, "names_system.held") }()
	Build_Handle_Invariants(subject, "names_system.subject")
	Target_Handle_Invariants(one, "names_system.one")
	token.Source_Invariants(name, "names_system.name")
	text := take_stem(name)
	subject.Sources.Text = Constraint_Text(text)
	take_opening(subject)
	opening := int(subject.Places.Opening)
	if opening == len(text) {
		return true
	}
	take_field(subject)
	tail := int(subject.Places.Field)
	// A name closing at the word of a test file answers for the name ahead of that word, thus
	// the read cuts the word and the underscore that opened it.
	if string(text[tail:]) == TEST_WORD {
		text = text[:tail-1]
		if len(text) < opening+1 {
			return true
		}
		subject.Sources.Text = Constraint_Text(text)
		take_field(subject)
		tail = int(subject.Places.Field)
	}
	// A name states a system and an architecture only where a second field stands behind the
	// underscore that opened the run.
	if tail-1 > opening {
		subject.Sources.Text = Constraint_Text(text[:tail-1])
		take_field(subject)
		head := int(subject.Places.Field)
		subject.Sources.Tag = Tag(text[head : tail-1])
		if bool(known_system(subject)) {
			subject.Sources.Tag = Tag(text[tail:])
			if bool(known_architecture(subject)) {
				subject.Sources.Word = Word(one.Words.Architecture)
				if !bool(same_word(subject)) {
					return false
				}
				subject.Sources.Tag = Tag(text[head : tail-1])
				subject.Sources.Word = Word(one.Words.System)
				return same_word(subject)
			}
		}
	}
	subject.Sources.Tag = Tag(text[tail:])
	return holds_word(subject, one)
}

// Holds_Imports reports whether every import one file states names this module or the standard
// library, and whether the reader read them. A path outside both names a third party, which
// stands in no build this package answers for.
func Holds_Imports(subject Build_Handle, text token.Source) (result Read_Result) {
	defer func() { Read_Result_Invariants(result, "holds_imports.result") }()
	Build_Handle_Invariants(subject, "holds_imports.subject")
	token.Source_Invariants(text, "holds_imports.text")
	subject.Sources.Text = Constraint_Text(text)
	subject.Flags.Failed = false
	place := 0
	count := 0
	opened := false
	stated := false
	for place < len(text) {
		end := place
		for end < len(text) {
			if text[end] == '\n' {
				break
			}
			end = end + 1
		}
		subject.Sources.Tag = Tag(text[place:end])
		place = end + 1
		if !stated {
			subject.Heads.Value = PACKAGE_HEAD
			stated = bool(opens_line(subject))
			continue
		}
		if bool(states_no_import(subject)) {
			continue
		}
		if opened {
			if string(subject.Sources.Tag) == IMPORT_TAIL {
				opened = false
				continue
			}
		} else {
			subject.Heads.Value = IMPORT_OPENING
			if bool(opens_line(subject)) {
				opened = true
				continue
			}
			subject.Heads.Value = IMPORT_HEAD
			if !bool(opens_line(subject)) {
				// The imports of a file stand ahead of every other declaration,
				// thus the first one that follows closes the run of them.
				return Read_Result{Held: true, OK: true}
			}
		}
		count = count + 1
		if count > IMPORT_COUNT_MAXIMUM {
			return Read_Result{Held: false, OK: false}
		}
		line := holds_import_line(subject)
		if !bool(line.OK) {
			return Read_Result{Held: false, OK: false}
		}
		if !bool(line.Held) {
			return Read_Result{Held: false, OK: true}
		}
	}
	// A block the text never closes states an import the read never saw, thus the read states
	// no truth about the file at all.
	if opened {
		return Read_Result{Held: false, OK: false}
	}
	return Read_Result{Held: true, OK: true}
}

// Reads the path one line states and reports whether it stands in the build, and whether the line
// stated a path at all.
func holds_import_line(subject Build_Handle) (result Read_Result) {
	defer func() { Read_Result_Invariants(result, "holds_import_line.result") }()
	Build_Handle_Invariants(subject, "holds_import_line.subject")
	if !bool(take_import_path(subject)) {
		return Read_Result{Held: false, OK: false}
	}
	return Read_Result{Held: Boolean(!bool(names_third_party(subject))), OK: true}
}

// Reports whether the line the tag slot holds states no import at all, which a blank line and a
// comment both do. A block of imports holds either between the paths it states.
func states_no_import(subject Build_Handle) (skipped Boolean) {
	defer func() { Boolean_Invariants(skipped, "states_no_import.skipped") }()
	Build_Handle_Invariants(subject, "states_no_import.subject")
	line := subject.Sources.Tag
	if len(line) == 0 {
		return true
	}
	if line[0] == '/' {
		return true
	}
	if line[0] != '\t' {
		return false
	}
	if len(line) == 1 {
		return true
	}
	return Boolean(line[1] == '/')
}

// Reads the path the line the tag slot holds states into the word slot, and reports whether it
// states one. A line whose quotes never close states no path the read can stand on.
func take_import_path(subject Build_Handle) (found Boolean) {
	defer func() { Boolean_Invariants(found, "take_import_path.found") }()
	Build_Handle_Invariants(subject, "take_import_path.subject")
	line := subject.Sources.Tag
	opening := 0
	for opening < len(line) {
		if line[opening] == '"' {
			break
		}
		opening = opening + 1
	}
	if opening == len(line) {
		return false
	}
	tail := opening + 1
	for tail < len(line) {
		if line[tail] == '"' {
			break
		}
		tail = tail + 1
	}
	if tail == len(line) {
		return false
	}
	subject.Sources.Word = Word(line[opening+1 : tail])
	return true
}

// Reports whether the path the word slot holds names a third party. The first element of a path
// the standard library states holds no period, and neither does one this module states, thus a
// period there names a host, which is how every module outside this one is named.
func names_third_party(subject Build_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "names_third_party.yes") }()
	Build_Handle_Invariants(subject, "names_third_party.subject")
	path := subject.Sources.Word
	place := 0
	for place < len(path) {
		if path[place] == '/' {
			return false
		}
		if path[place] == '.' {
			return true
		}
		place = place + 1
	}
	return false
}

// Reads the name of one file up to the period that closes it, which is the text the systems it
// names stand in.
func take_stem(name token.Source) (text token.Source) {
	defer func() { token.Source_Invariants(text, "take_stem.text") }()
	token.Source_Invariants(name, "take_stem.name")
	for place := range len(name) {
		if name[place] == '.' {
			return name[:place]
		}
	}
	return name
}

// Names the byte the first underscore of one name stands at, or the byte past its end where it
// holds none. Everything ahead of that underscore names no system.
func take_opening(subject Build_Handle) {
	Build_Handle_Invariants(subject, "take_opening.subject")
	text := subject.Sources.Text
	for index := range len(text) {
		if text[index] == '_' {
			subject.Places.Opening = Opening_Place(index)
			return
		}
	}
	subject.Places.Opening = Opening_Place(len(text))
}

// Names the byte the final field of one name opens at, which is the byte behind the last
// underscore standing at or behind the opening one.
func take_field(subject Build_Handle) {
	Build_Handle_Invariants(subject, "take_field.subject")
	text := subject.Sources.Text
	opening := int(subject.Places.Opening)
	held := opening + 1
	for index := int(opening); index < len(text); index++ {
		if text[index] == '_' {
			held = index + 1
		}
	}
	subject.Places.Field = Field_Place(held)
}

// Reports whether one word of a name states the build the target names. A word that names neither
// a system nor an architecture says nothing, thus the file stands in every build.
func holds_word(subject Build_Handle, one Target_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "holds_word.yes") }()
	Build_Handle_Invariants(subject, "holds_word.subject")
	Target_Handle_Invariants(one, "holds_word.one")
	if bool(known_system(subject)) {
		subject.Sources.Word = Word(one.Words.System)
		return same_word(subject)
	}
	if bool(known_architecture(subject)) {
		subject.Sources.Word = Word(one.Words.Architecture)
		return same_word(subject)
	}
	return true
}

// Reports whether the tag one read stands on names an operating system the toolchain builds for.
func known_system(subject Build_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "known_system.yes") }()
	Build_Handle_Invariants(subject, "known_system.subject")
	switch string(subject.Sources.Tag) {
	case "aix", "android", "darwin", "dragonfly", "freebsd", "hurd", "illumos", "ios", "js",
		"linux", "nacl", "netbsd", "openbsd", "plan9", "solaris", "wasip1", "windows",
		"zos":
		return true
	}
	return false
}

// Reports whether the tag one read stands on names an architecture the toolchain builds for.
func known_architecture(subject Build_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "known_architecture.yes") }()
	Build_Handle_Invariants(subject, "known_architecture.subject")
	switch string(subject.Sources.Tag) {
	case "386", "amd64", "amd64p32", "arm", "armbe", "arm64", "arm64be", "loong64", "mips",
		"mipsle", "mips64", "mips64le", "mips64p32", "mips64p32le", "ppc", "ppc64",
		"ppc64le", "riscv", "riscv64", "s390", "s390x", "sparc", "sparc64", "wasm":
		return true
	}
	return false
}

// Reports whether the word one read compares against names an operating system Go calls a Unix.
func names_unix(subject Build_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "names_unix.yes") }()
	Build_Handle_Invariants(subject, "names_unix.subject")
	switch string(subject.Sources.Word) {
	case "aix", "android", "darwin", "dragonfly", "freebsd", "hurd", "illumos", "ios",
		"linux", "netbsd", "openbsd", "solaris":
		return true
	}
	return false
}

// PHASE_IDLE names a runner standing between two reads, which owes a step of its own.
const PHASE_IDLE = 0

// PHASE_OPEN_DIRECTORY names the open of the directory the runner reads.
const PHASE_OPEN_DIRECTORY = 1

// PHASE_READ_ENTRIES names one pass over the children of that directory.
const PHASE_READ_ENTRIES = 2

// PHASE_OPEN_FILE names the open of one file the runner reads the constraint of.
const PHASE_OPEN_FILE = 3

// PHASE_READ_HEADER names the read of the bytes one file opens with.
const PHASE_READ_HEADER = 4

// PHASE_CLOSE_FILE names the close of that file.
const PHASE_CLOSE_FILE = 5

// PHASE_CLOSE_DIRECTORY names the close that ends the read.
const PHASE_CLOSE_DIRECTORY = 6

// PHASE_MINIMUM is the phase of a runner that owes a step of its own.
const PHASE_MINIMUM = PHASE_IDLE

// PHASE_MAXIMUM is the phase of the close that ends the read.
const PHASE_MAXIMUM = PHASE_CLOSE_DIRECTORY

// NAME_COUNT_MINIMUM is the count of no name at all.
const NAME_COUNT_MINIMUM = 0

// STORAGE_SIZE_MINIMUM keeps a runner the caller has not opened yet valid, which holds no
// storage at all.
const STORAGE_SIZE_MINIMUM = 0

// ENTRY_COUNT_MAXIMUM caps the children one pass over a directory reads, which is one slot for
// every record the block budget of a pass admits.
const ENTRY_COUNT_MAXIMUM = 8192

// STORAGE_SIZE_MAXIMUM caps the bytes one read of a directory writes into, which is one mebibyte:
// the widest source this dialect reads.
const STORAGE_SIZE_MAXIMUM = 1048576

// PATH_SIZE_MINIMUM is the path of no bytes, which names the directory the process stands in.
const PATH_SIZE_MINIMUM = 0

// PATH_SIZE_MAXIMUM is the widest path one read of a directory admits.
const PATH_SIZE_MAXIMUM = 4096

// BUILD_HEAD is the text one build constraint opens with.
const BUILD_HEAD = "//go:build "

// PACKAGE_HEAD is the text the package clause opens with, which closes the run of lines a
// constraint stands in.
const PACKAGE_HEAD = "package "

// IMPORT_HEAD is the text one import declaration opens with.
const IMPORT_HEAD = "import "

// IMPORT_OPENING is the text the block of imports one file states opens with.
const IMPORT_OPENING = "import ("

// IMPORT_TAIL is the text that closes that block.
const IMPORT_TAIL = ")"

// IMPORT_COUNT_MAXIMUM caps the imports one head states, which is wider than any file the
// toolchain itself writes.
const IMPORT_COUNT_MAXIMUM = 256

// SOURCE_TAIL is the text the name of one Go file closes with.
const SOURCE_TAIL = ".go"

// DIRECTORY_COUNT_NAME holds how many names one read kept.
const DIRECTORY_COUNT_NAME = 0

// DIRECTORY_COUNT_WRITTEN holds how many bytes of the storage those names view.
const DIRECTORY_COUNT_WRITTEN = 1

// DIRECTORY_COUNT_ENTRY holds the child of the pass the read stands on.
const DIRECTORY_COUNT_ENTRY = 2

// DIRECTORY_COUNT_ENTRY_TOTAL holds how many children the pass holds.
const DIRECTORY_COUNT_ENTRY_TOTAL = 3

// DIRECTORY_COUNT_PHASE holds the read the runner submitted.
const DIRECTORY_COUNT_PHASE = 4

// DIRECTORY_COUNT_SLOT_COUNT is the counter count one runner holds.
const DIRECTORY_COUNT_SLOT_COUNT = 5

// DIRECTORY_FLAG_QUEUED marks a runner whose callback stood one continuation.
const DIRECTORY_FLAG_QUEUED = 0

// DIRECTORY_FLAG_STOPPED marks a read that stopped.
const DIRECTORY_FLAG_STOPPED = 1

// DIRECTORY_FLAG_SLOT_COUNT is the flag count one runner holds.
const DIRECTORY_FLAG_SLOT_COUNT = 2

// Name_Count is how many source names one directory read retained.
type Name_Count Count

// Name_Count_Invariants states every retained-name count.
func Name_Count_Invariants(value Name_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), NAME_COUNT_MINIMUM, ENTRY_COUNT_MAXIMUM).
		Ensure()
}

// Written_Count is how many bytes retained names view.
type Written_Count Count

// Written_Count_Invariants states every retained-name byte count.
func Written_Count_Invariants(value Written_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), STORAGE_SIZE_MINIMUM, STORAGE_SIZE_MAXIMUM).
		Ensure()
}

// Entry_Index is the child of one directory pass a reader stands on.
type Entry_Index Count

// Entry_Index_Invariants states every bounded child cursor.
func Entry_Index_Invariants(value Entry_Index, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), NAME_COUNT_MINIMUM, ENTRY_COUNT_MAXIMUM).
		Ensure()
}

// Entry_Total is how many children one directory pass returned.
type Entry_Total Count

// Entry_Total_Invariants states every bounded pass size.
func Entry_Total_Invariants(value Entry_Total, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), NAME_COUNT_MINIMUM, ENTRY_COUNT_MAXIMUM).
		Ensure()
}

// Phase is the read one directory runner submitted.
type Phase Count

// Phase_Invariants states every runner phase.
func Phase_Invariants(value Phase, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), PHASE_MINIMUM, PHASE_MAXIMUM).
		Ensure()
}

// Directory_Counts keeps unlike runner cursors in named storage.
type Directory_Counts struct {
	// Name survives passes that overwrite entry storage.
	Name Name_Count
	// Written survives passes that overwrite record storage.
	Written Written_Count
	// Entry separates the current child from the pass size.
	Entry Entry_Index
	// Entry_Total remains fixed while the current child advances.
	Entry_Total Entry_Total
	// Phase survives the callback boundary between submission and application.
	Phase Phase
}

// Directory_Counts_Invariants composes each runner-cursor domain once.
func Directory_Counts_Invariants(value Directory_Counts, namespace aver.Namespace) {
	Name_Count_Invariants(value.Name, namespace)
	Written_Count_Invariants(value.Written, namespace)
	Entry_Index_Invariants(value.Entry, namespace)
	Entry_Total_Invariants(value.Entry_Total, namespace)
	Phase_Invariants(value.Phase, namespace)
}

// Queued is whether a callback left work for the caller.
type Queued Boolean

// Queued_Invariants states both callback outcomes.
func Queued_Invariants(value Queued, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The directory callback queues work.").
		Ensure()
}

// Stopped is whether a directory runner reached a terminal state.
type Stopped Boolean

// Stopped_Invariants states both terminal outcomes.
func Stopped_Invariants(value Stopped, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The directory runner stops.").
		Ensure()
}

// Directory_Flags keeps unlike runner reports in named storage.
type Directory_Flags struct {
	// Queued survives until the caller applies the callback result.
	Queued Queued
	// Stopped survives every later poll of the terminal runner.
	Stopped Stopped
}

// Directory_Flags_Invariants composes each runner-report domain once.
func Directory_Flags_Invariants(value Directory_Flags, namespace aver.Namespace) {
	Queued_Invariants(value.Queued, namespace)
	Stopped_Invariants(value.Stopped, namespace)
}

// Path names the directory one read stands on.
type Path string

// Path_Invariants states every path one read of a directory admits.
func Path_Invariants(value Path, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), PATH_SIZE_MINIMUM, PATH_SIZE_MAXIMUM).
		Ensure()
}

// Loop names the IO one read of a directory submits through. A runner the caller has not opened
// yet holds none, thus the domain holds the loop that states a backend beside the loop that
// states none, which nbio.IO alone does not.
type Loop nbio.IO

// Loop_Invariants states what holds of one loop across the whole life of a runner. The backend
// stands outside it: a runner the caller has not opened yet holds none, and the open states the
// one it submits through.
func Loop_Invariants(value Loop, namespace aver.Namespace) {
	nbio.IO_Invariants(nbio.IO(value), namespace)
	aver.Always(
		value.Network.State == value.Storage.State,
		"A directory read carries one backend across both halves.",
	)
}

// Entry_Storage receives one pass over the children of a directory, which the caller owns.
type Entry_Storage []nbio.Directory_Entry

// Entry_Storage_Invariants states every run of slots one pass writes into.
func Entry_Storage_Invariants(value Entry_Storage, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), STORAGE_SIZE_MINIMUM, ENTRY_COUNT_MAXIMUM).
		Ensure()
}

// Record_Storage receives the records the backend writes for one pass, which the caller owns.
type Record_Storage []byte

// Record_Storage_Invariants states every run of bytes one pass writes into. The widest one is the
// block budget the backend states, which refuses a pass wider than one.
func Record_Storage_Invariants(value Record_Storage, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), STORAGE_SIZE_MINIMUM, nbio.DIRECTORY_BUFFER_SIZE_MAXIMUM,
		).
		Ensure()
}

// Name_Bytes holds the bytes the names one read keeps view, which the caller owns.
type Name_Bytes []byte

// Name_Bytes_Invariants states every run of bytes the names of one read view.
func Name_Bytes_Invariants(value Name_Bytes, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), STORAGE_SIZE_MINIMUM, STORAGE_SIZE_MAXIMUM).
		Ensure()
}

// Header_Storage holds the bytes one file opens with, which the caller owns.
type Header_Storage []byte

// Header_Storage_Invariants states every run of bytes one header read writes into.
func Header_Storage_Invariants(value Header_Storage, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), STORAGE_SIZE_MINIMUM, STORAGE_SIZE_MAXIMUM).
		Ensure()
}

// Name_Storage holds the names one read of a directory keeps, which view the bytes the caller
// owns.
type Name_Storage []token.Source

// Name_Storage_Invariants states every run of names one read hands back.
func Name_Storage_Invariants(value Name_Storage, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), STORAGE_SIZE_MINIMUM, ENTRY_COUNT_MAXIMUM).
		Ensure()
}

// Directory_Memory states the storage one read of a directory writes into. The caller owns every
// run of it, thus the runner allocates nothing and the composition root alone states the widths.
type Directory_Memory struct {
	// Entries receives one pass over the children of the directory.
	Entries Entry_Storage
	// Records receives the records the backend writes for that pass.
	Records Record_Storage
	// Names holds the names the read keeps.
	Names Name_Storage
	// Bytes holds the bytes those names view, because the pass behind one overwrites the
	// entries it wrote.
	Bytes Name_Bytes
	// Header holds the bytes one file opens with.
	Header Header_Storage
	// Reader holds the state one read of a constraint runs through.
	Reader Build_Handle
}

// Directory_Memory_Invariants states every run of storage one read writes into.
func Directory_Memory_Invariants(value Directory_Memory, namespace aver.Namespace) {
	Entry_Storage_Invariants(value.Entries, namespace)
	Record_Storage_Invariants(value.Records, namespace)
	Name_Storage_Invariants(value.Names, namespace)
	Name_Bytes_Invariants(value.Bytes, namespace)
	Header_Storage_Invariants(value.Header, namespace)
	Build_Handle_Invariants(value.Reader, namespace)
}

// Directory_Runner_Fields states the concrete runner layout independently of its validated view.
type Directory_Runner_Fields struct {
	// Completion stands first so the static callback recovers the runner with no closure.
	Completion nbio.Completion
	// Loop submits the storage work the caller drives.
	Loop Loop
	// Target names the build the files stand in or stand outside of.
	Target Target_Handle
	// Reader holds the state one read of a constraint runs through, which the caller owns.
	Reader Build_Handle
	// Entries receives one pass over the children of the directory.
	Entries Entry_Storage
	// Records receives the records the backend writes for one pass.
	Records Record_Storage
	// Names holds the names the read keeps, which view the bytes behind them.
	Names Name_Storage
	// Bytes holds the bytes those names view, because the pass behind one overwrites the
	// entries it wrote.
	Bytes Name_Bytes
	// Header holds the bytes one file opens with, which the constraint stands in.
	Header Header_Storage
	// Counts holds the names the read kept, the bytes they view, the child of the pass it
	// stands on, the children that pass holds, and the read it submitted.
	Counts Directory_Counts
	// Flags holds what the runner met: a continuation one callback stood, and the stop.
	Flags Directory_Flags
	// Directory is the descriptor the read holds between its open and its close.
	Directory nbio.File
	// File is the descriptor one header read holds.
	File nbio.File
	// Result retains the terminal error.
	Result error
}

// Directory_Runner_Fields_Invariants composes every invariant-bearing runner field once.
func Directory_Runner_Fields_Invariants(value Directory_Runner_Fields, namespace aver.Namespace) {
	Entry_Storage_Invariants(value.Entries, namespace)
	Record_Storage_Invariants(value.Records, namespace)
	Name_Storage_Invariants(value.Names, namespace)
	Name_Bytes_Invariants(value.Bytes, namespace)
	Header_Storage_Invariants(value.Header, namespace)
	Build_Handle_Invariants(value.Reader, namespace)
	Target_Handle_Invariants(value.Target, namespace)
	Directory_Counts_Invariants(value.Counts, namespace)
	Directory_Flags_Invariants(value.Flags, namespace)
	// The loop stands last because it states a backend: a runner the open has not reached
	// yet holds none, and the storage ahead of this line states its widths either way.
	Loop_Invariants(value.Loop, namespace)
}

// Directory_Runner_Fields_Stored keeps one poll from proving stale runner state.
type Directory_Runner_Fields_Stored interface{}

// Directory_Runner_Fields_Stored_Invariants fixes runner representation.
func Directory_Runner_Fields_Stored_Invariants(
	value Directory_Runner_Fields_Stored, _ aver.Namespace,
) {
	_, valid := value.(Directory_Runner_Fields)
	aver.Always(valid == (value != nil), "Directory runner has expected storage type.")
}

// Directory_Runner_Envelope keeps the callback-compatible layout behind one boundary.
type Directory_Runner_Envelope struct {
	// Directory_Runner_Fields stays embedded so callers retain concrete field access.
	Directory_Runner_Fields
}

// Directory_Runner_Envelope_Invariants composes concrete runner storage.
func Directory_Runner_Envelope_Invariants(
	value Directory_Runner_Envelope, namespace aver.Namespace,
) {
	Directory_Runner_Fields_Invariants(value.Directory_Runner_Fields, namespace)
}

// Directory_Runner reads one directory and keeps the Go files that stand in one build. It composes
// the injected storage the caller drives, thus this package opens nothing of its own and the
// composition root alone owns the loop.
type Directory_Runner Directory_Runner_Envelope

// Directory_Runner_Invariants states the storage shape one directory read holds.
func Directory_Runner_Invariants(value Directory_Runner, namespace aver.Namespace) {
	Directory_Runner_Fields_Stored_Invariants(
		Directory_Runner_Fields_Stored(value.Directory_Runner_Fields), namespace,
	)
}

// Directory_Runner_Handle gives caller-owned directory state one identity.
type Directory_Runner_Handle *Directory_Runner

// Directory_Runner_Handle_Invariants composes present directory state.
func Directory_Runner_Handle_Invariants(
	value Directory_Runner_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Directory_Runner_Invariants(*value, namespace)
}

// Directory_Runner_Init opens one directory and states the read that follows. The caller owns the
// loop, the target, and every buffer the read writes into, thus the runner allocates nothing.
func Directory_Runner_Init(
	runner Directory_Runner_Handle,
	loop nbio.IO,
	one Target_Handle,
	path Path,
	memory Directory_Memory,
) {
	Directory_Runner_Handle_Invariants(runner, "directory_runner_init.runner")
	nbio.IO_Invariants(loop, "directory_runner_init.loop")
	Loop_Invariants(Loop(loop), "directory_runner_init.runner_loop")
	Target_Handle_Invariants(one, "directory_runner_init.one")
	Path_Invariants(path, "directory_runner_init.path")
	Directory_Memory_Invariants(memory, "directory_runner_init.memory")
	// The open states the whole runner ahead of the assertion that reads it, because a runner
	// the caller has not opened yet holds no loop, and no loop states no backend.
	counts, flags := runner.Counts, runner.Flags
	*runner = Directory_Runner{Directory_Runner_Fields: Directory_Runner_Fields{
		Loop: Loop(loop), Target: one, Reader: memory.Reader, Entries: memory.Entries,
		Records: memory.Records, Names: memory.Names, Bytes: memory.Bytes,
		Header: memory.Header, Counts: counts, Flags: flags,
	},
	}
	Directory_Runner_Handle_Invariants(runner, "directory_runner_init.opened")
	runner.Counts.Phase = PHASE_OPEN_DIRECTORY
	nbio.Storage_Open_At(
		runner.Loop.Storage, &runner.Completion, nbio.DIRECTORY_CURRENT, string(path),
		nbio.Open_At_Options{Access: nbio.OPEN_READ_ONLY}, directory_completion,
	)
}

// Holds the continuation one completion queued, which the caller reads through the rearm.
func directory_completion(completion nbio.Completion_Handle) {
	// First-field ownership avoids a closure allocation and a self-pointer escape.
	runner := (Directory_Runner_Handle)(unsafe.Pointer(completion))
	runner.Flags.Queued = true
}

// Directory_Runner_Rearm runs the continuation the callback queued, or the step the runner owes of
// its own. False means the caller drives the pending read, or the runner stopped.
func Directory_Runner_Rearm(runner Directory_Runner_Handle) (rearm Boolean) {
	defer func() { Boolean_Invariants(rearm, "directory_runner_rearm.rearm") }()
	Directory_Runner_Handle_Invariants(runner, "directory_runner_rearm.runner")
	if bool(runner.Flags.Stopped) {
		return false
	}
	// A read the caller gave no room for one child reads nothing, thus it stops where it
	// stands rather than ask the backend for a pass it cannot hold.
	if len(runner.Entries) == STORAGE_SIZE_MINIMUM {
		runner.Flags.Stopped = true
		return false
	}
	// A read the caller gave no storage reads nothing, thus it stops where it stands rather
	// than ask the backend for a pass it has nowhere to write.
	if bool(runner.Flags.Queued) {
		runner.Flags.Queued = false
		return directory_completion_apply(runner)
	}
	if runner.Counts.Phase != PHASE_IDLE {
		return false
	}
	open_next_file(runner)
	return false
}

// Directory_Runner_Work_Queued reports the callback stored one continuation for the caller.
func Directory_Runner_Work_Queued(runner Directory_Runner_Handle) (queued Boolean) {
	defer func() { Boolean_Invariants(queued, "directory_runner_work_queued.queued") }()
	Directory_Runner_Handle_Invariants(runner, "directory_runner_work_queued.runner")
	return Boolean(bool(runner.Flags.Queued))
}

// Directory_Runner_Stopped reports the terminal state.
func Directory_Runner_Stopped(runner Directory_Runner_Handle) (stopped Boolean) {
	defer func() { Boolean_Invariants(stopped, "directory_runner_stopped.stopped") }()
	Directory_Runner_Handle_Invariants(runner, "directory_runner_stopped.runner")
	return Boolean(bool(runner.Flags.Stopped))
}

// Directory_Runner_Status reports the error that stopped the runner.
func Directory_Runner_Status(runner Directory_Runner_Handle) (err error) {
	Directory_Runner_Handle_Invariants(runner, "directory_runner_status.runner")
	return runner.Result
}

// Directory_Runner_Names hands back the names of the files that stand in the build, which view the
// storage the caller owns.
func Directory_Runner_Names(runner Directory_Runner_Handle) (names Name_Storage) {
	defer func() { Name_Storage_Invariants(names, "directory_runner_names.names") }()
	Directory_Runner_Handle_Invariants(runner, "directory_runner_names.runner")
	aver.Always(
		bool(runner.Flags.Stopped),
		"A directory read is read after the runner stops.",
	)
	return Name_Storage(runner.Names[:runner.Counts.Name])
}

// Runs the continuation the read that completed states.
func directory_completion_apply(runner Directory_Runner_Handle) (rearm Boolean) {
	defer func() { Boolean_Invariants(rearm, "directory_completion_apply.rearm") }()
	Directory_Runner_Handle_Invariants(runner, "directory_completion_apply.runner")
	switch runner.Counts.Phase {
	case PHASE_OPEN_DIRECTORY:
		open_directory_apply(runner)
		return false
	case PHASE_READ_ENTRIES:
		return read_entries_apply(runner)
	case PHASE_OPEN_FILE:
		return open_file_apply(runner)
	case PHASE_READ_HEADER:
		read_header_apply(runner)
		return false
	case PHASE_CLOSE_FILE:
		runner.Counts.Entry = runner.Counts.Entry + 1
		runner.Counts.Phase = PHASE_IDLE
		return true
	}
	runner.Flags.Stopped = true
	return false
}

// Holds the descriptor the open of the directory answered with, and reads its first pass.
func open_directory_apply(runner Directory_Runner_Handle) {
	Directory_Runner_Handle_Invariants(runner, "open_directory_apply.runner")
	if runner.Completion.Error != nil {
		runner.Result = runner.Completion.Error
		runner.Flags.Stopped = true
		return
	}
	runner.Directory = nbio.File(runner.Completion.Data)
	read_entries(runner)
}

// Stands the read at the first child of one pass. A pass of no child closes the read.
func read_entries_apply(runner Directory_Runner_Handle) (rearm Boolean) {
	defer func() { Boolean_Invariants(rearm, "read_entries_apply.rearm") }()
	Directory_Runner_Handle_Invariants(runner, "read_entries_apply.runner")
	held := Count(runner.Completion.Data)
	if held > Count(len(runner.Entries)) {
		held = Count(len(runner.Entries))
	}
	if runner.Completion.Error != nil {
		runner.Result = runner.Completion.Error
		held = NAME_COUNT_MINIMUM
	}
	// A pass of no child closes the read, and so does the pass one error stopped.
	if held == NAME_COUNT_MINIMUM {
		runner.Counts.Phase = PHASE_CLOSE_DIRECTORY
		nbio.IO_Close(
			nbio.IO(runner.Loop), &runner.Completion, runner.Directory,
			directory_completion,
		)
		return false
	}
	runner.Counts.Entry_Total = Entry_Total(held)
	runner.Counts.Entry = NAME_COUNT_MINIMUM
	runner.Counts.Phase = PHASE_IDLE
	return true
}

// Holds the descriptor the open of one file answered with, and reads the bytes it opens with. A
// file the open refused stands outside every build the runner answers for.
func open_file_apply(runner Directory_Runner_Handle) (rearm Boolean) {
	defer func() { Boolean_Invariants(rearm, "open_file_apply.rearm") }()
	Directory_Runner_Handle_Invariants(runner, "open_file_apply.runner")
	if runner.Completion.Error != nil {
		runner.Counts.Entry = runner.Counts.Entry + 1
		runner.Counts.Phase = PHASE_IDLE
		return true
	}
	runner.File = nbio.File(runner.Completion.Data)
	runner.Counts.Phase = PHASE_READ_HEADER
	nbio.Storage_Read(
		runner.Loop.Storage, &runner.Completion, runner.File, runner.Header, 0,
		time.Duration(0), directory_completion,
	)
	return false
}

// Reads the constraint the header states and keeps the name where it holds.
func read_header_apply(runner Directory_Runner_Handle) {
	Directory_Runner_Handle_Invariants(runner, "read_header_apply.runner")
	count := 0
	if runner.Completion.Error == nil {
		count = int(runner.Completion.Data)
	}
	if count > len(runner.Header) {
		count = len(runner.Header)
	}
	runner.Reader.Sources.Text = Constraint_Text(runner.Header[:count])
	// A header that states no constraint states no bar to the build, and one that states a
	// constraint the reader refused keeps the file out of every build.
	held := Boolean(true)
	if bool(take_build_line(runner.Reader)) {
		result := Holds_Constraint(
			runner.Reader, runner.Target, token.Source(runner.Reader.Sources.Tag),
		)
		held = result.Held
		if !bool(result.OK) {
			held = false
		}
	}
	// A file naming a third party stands in no build, thus the read drops it the same way it
	// drops one whose constraint fails.
	if bool(held) {
		result := Holds_Imports(runner.Reader, token.Source(runner.Header[:count]))
		held = result.Held
		if !bool(result.OK) {
			held = false
		}
	}
	if bool(held) {
		hold_name(runner)
	}
	runner.Counts.Phase = PHASE_CLOSE_FILE
	nbio.IO_Close(nbio.IO(runner.Loop), &runner.Completion, runner.File, directory_completion)
}

// Reads the constraint line one header states into the tag slot, and reports whether it states
// one. That line stands ahead of the package clause, thus the read stops where the clause opens.
func take_build_line(subject Build_Handle) (found Boolean) {
	defer func() { Boolean_Invariants(found, "take_build_line.found") }()
	Build_Handle_Invariants(subject, "take_build_line.subject")
	text := subject.Sources.Text
	place := 0
	for place < len(text) {
		end := place
		for end < len(text) {
			if text[end] == '\n' {
				break
			}
			end = end + 1
		}
		subject.Sources.Tag = Tag(text[place:end])
		subject.Heads.Value = PACKAGE_HEAD
		if bool(opens_line(subject)) {
			return false
		}
		subject.Heads.Value = BUILD_HEAD
		if bool(opens_line(subject)) {
			subject.Sources.Tag = Tag(text[place+len(BUILD_HEAD) : end])
			return true
		}
		place = end + 1
	}
	return false
}

// Reports whether the line the tag slot holds opens with the text stated.
func opens_line(subject Build_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "opens_line.yes") }()
	Build_Handle_Invariants(subject, "opens_line.subject")
	line := subject.Sources.Tag
	head := subject.Heads.Value
	if len(line) < len(head) {
		return false
	}
	return Boolean(string(line[:len(head)]) == string(head))
}

// Writes the name of the file the read stands on into the storage the caller owns, because the
// pass behind this one overwrites the entries it stands in. A name the storage cannot hold stops
// no read: the file stands unread rather than read wrongly.
func hold_name(runner Directory_Runner_Handle) {
	Directory_Runner_Handle_Invariants(runner, "hold_name.runner")
	name := runner.Entries[runner.Counts.Entry].Name
	if int(runner.Counts.Name) >= len(runner.Names) {
		return
	}
	written := int(runner.Counts.Written)
	if written+len(name) > len(runner.Bytes) {
		return
	}
	copy(runner.Bytes[written:], name)
	held := runner.Bytes[written : written+len(name)]
	runner.Names[runner.Counts.Name] = token.Source(held)
	runner.Counts.Name = runner.Counts.Name + 1
	runner.Counts.Written = Written_Count(written + len(name))
}

// Reads one pass over the children of the directory.
func read_entries(runner Directory_Runner_Handle) {
	Directory_Runner_Handle_Invariants(runner, "read_entries.runner")
	runner.Counts.Phase = PHASE_READ_ENTRIES
	nbio.Storage_Get_Directory_Entries(
		runner.Loop.Storage, &runner.Completion, runner.Directory,
		runner.Records, runner.Entries, directory_completion,
	)
}

// Opens the file the read stands on, over every child that names no Go file of this build. A pass
// the read stands at the end of opens the pass behind it.
func open_next_file(runner Directory_Runner_Handle) {
	Directory_Runner_Handle_Invariants(runner, "open_next_file.runner")
	for int(runner.Counts.Entry) < int(runner.Counts.Entry_Total) {
		if bool(names_source(runner)) {
			break
		}
		runner.Counts.Entry = runner.Counts.Entry + 1
	}
	if int(runner.Counts.Entry) >= int(runner.Counts.Entry_Total) {
		read_entries(runner)
		return
	}
	runner.Counts.Phase = PHASE_OPEN_FILE
	nbio.Storage_Open_At(
		runner.Loop.Storage, &runner.Completion, runner.Directory,
		runner.Entries[runner.Counts.Entry].Name,
		nbio.Open_At_Options{Access: nbio.OPEN_READ_ONLY}, directory_completion,
	)
}

// Reports whether the child the read stands on names a Go file standing in the build the target
// names.
func names_source(runner Directory_Runner_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "names_source.yes") }()
	Directory_Runner_Handle_Invariants(runner, "names_source.runner")
	entry := runner.Entries[runner.Counts.Entry]
	if entry.Is_Directory {
		return false
	}
	name := entry.Name
	if len(name) <= len(SOURCE_TAIL) {
		return false
	}
	if name[len(name)-len(SOURCE_TAIL):] != SOURCE_TAIL {
		return false
	}
	// Backend names are strings. Stage one in caller bytes before byte-oriented build checks;
	// converting it directly would allocate storage hidden from caller.
	written := int(runner.Counts.Written)
	if written+len(name) > len(runner.Bytes) {
		return false
	}
	copy(runner.Bytes[written:], name)
	held := token.Source(runner.Bytes[written : written+len(name)])
	return Names_System(runner.Reader, runner.Target, held)
}
