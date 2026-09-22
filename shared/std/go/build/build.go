// Package build answers whether one file stands in one build. It reads the name of the file and
// the constraint the file states, and it reads no file system and no environment: the caller owns
// the name, the text, and the target, thus one read allocates nothing and states no default.
package build

import "unsafe"

import "local/james-orcales/shared/go/token"
import "local/james-orcales/shared/sim/aver/default"
import "local/james-orcales/shared/sim/nbio"
import "local/james-orcales/shared/sim/time"

// TEXT_SIZE_MAXIMUM is the widest constraint one read admits, which is wider than any constraint
// the toolchain itself writes.
const TEXT_SIZE_MAXIMUM = 4096

// DEPTH_MAXIMUM caps the brackets one constraint nests, which is deeper than any constraint a
// reader follows.
const DEPTH_MAXIMUM = 32

// TAG_COUNT_MAXIMUM caps the tags one target names.
const TAG_COUNT_MAXIMUM = 32

// COUNT_MINIMUM is the smallest count one read holds.
const COUNT_MINIMUM = 0

// COUNT_MAXIMUM is the largest count one read holds, which is the widest text it reads.
const COUNT_MAXIMUM = TEXT_SIZE_MAXIMUM

// PLACE_MINIMUM is the first byte of one name.
const PLACE_MINIMUM = 0

// PLACE_MAXIMUM is the byte past the widest name one read admits, which is the widest source one
// scan of Go text reads.
const PLACE_MAXIMUM = token.SOURCE_SIZE_MAXIMUM

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
    Range_Int(
    len(value),
    token.SOURCE_SIZE_MINIMUM,
    token.SOURCE_SIZE_MAXIMUM,
  ).
    Ensure()
}

// Architecture is the machine word one target names.
type Architecture token.Source

// Architecture_Invariants states every bounded machine word.
func Architecture_Invariants(value Architecture, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(
    len(value),
    token.SOURCE_SIZE_MINIMUM,
    token.SOURCE_SIZE_MAXIMUM,
  ).
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
func Target_Words_Invariants(value *Target_Words, namespace aver.Namespace) {
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
func Target_Tag_Storage_Invariants(value *Target_Tag_Storage, namespace aver.Namespace) {
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
func Target_Counts_Invariants(value *Target_Counts, namespace aver.Namespace) {
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
func Build_Counts_Invariants(value *Build_Counts, namespace aver.Namespace) {
  Text_Place_Invariants(value.Place, namespace)
  Constraint_Depth_Invariants(value.Depth, namespace)
}

// Constraint_Text is the complete constraint one reader scans.
type Constraint_Text token.Source

// Constraint_Text_Invariants states every bounded view of one constraint. A read refuses text
// wider than the widest constraint before it holds any, and a stem that passes that width names
// no system, thus the view never passes it.
func Constraint_Text_Invariants(value Constraint_Text, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(len(value), token.SOURCE_SIZE_MINIMUM, TEXT_SIZE_MAXIMUM).
    Ensure()
}

// TAG_SIZE_MINIMUM is the narrowest tag. A tag opens at a byte a tag holds, thus it holds one.
const TAG_SIZE_MINIMUM = 1

// Tag is one word of a constraint.
type Tag token.Source

// Tag_Invariants states every bounded tag view. A tag is one word of a constraint, thus it
// stands inside the width that constraint states.
func Tag_Invariants(value Tag, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(len(value), TAG_SIZE_MINIMUM, TEXT_SIZE_MAXIMUM).
    Ensure()
}

// Term_Text is the text one term of a constraint opens, up to the end of the constraint. The read
// stands inside the text where it reads a term, thus the term holds one byte at least.
type Term_Text token.Source

// Term_Text_Invariants states every text a term opens.
func Term_Text_Invariants(value Term_Text, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(len(value), TAG_SIZE_MINIMUM, TEXT_SIZE_MAXIMUM).
    Ensure()
}

// STEM_SIZE_MINIMUM is the narrowest stem a read of fields stands on, which is the underscore
// that opens the first field.
const STEM_SIZE_MINIMUM = 1

// Stem is the name of one file ahead of its extension, holding an underscore that opens a field.
type Stem token.Source

// Stem_Invariants states every stem the read of fields stands on.
func Stem_Invariants(value Stem, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(len(value), STEM_SIZE_MINIMUM, TEXT_SIZE_MAXIMUM).
    Ensure()
}

// NAME_FIELD_SIZE_MAXIMUM is the widest field a name states. A field stands behind the
// underscore that opened it, thus it is one byte narrower than the widest stem.
const NAME_FIELD_SIZE_MAXIMUM = TEXT_SIZE_MAXIMUM - STEM_SIZE_MINIMUM

// Name_Field is one field of the name of a file, which the underscores of that name divide.
type Name_Field token.Source

// Name_Field_Invariants states every field a name states, down to the one of no bytes.
func Name_Field_Invariants(value Name_Field, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(
    len(value),
    token.SOURCE_SIZE_MINIMUM,
    NAME_FIELD_SIZE_MAXIMUM,
  ).
    Ensure()
}

// Scanned_Text is the whole text one scan of lines reads: the imports one file states, or the
// header one read of a directory holds. It is caller storage and stands as wide as one source.
type Scanned_Text token.Source

// Scanned_Text_Invariants states every bounded text one scan of lines reads.
func Scanned_Text_Invariants(value Scanned_Text, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(
    len(value),
    token.SOURCE_SIZE_MINIMUM,
    token.SOURCE_SIZE_MAXIMUM,
  ).
    Ensure()
}

// Line is one line of the text a scan reads. A line closes at the first break, and the text holds
// no break at all where it states one line, thus a line stands as wide as that text.
type Line token.Source

// Line_Invariants states every bounded line one scan reads.
func Line_Invariants(value Line, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(
    len(value),
    token.SOURCE_SIZE_MINIMUM,
    token.SOURCE_SIZE_MAXIMUM,
  ).
    Ensure()
}

// IMPORT_LINE_SIZE_MAXIMUM is the widest line an import states. It stands behind the package
// clause, and the shortest clause is its head and the break that closes it.
const IMPORT_LINE_SIZE_MAXIMUM = token.SOURCE_SIZE_MAXIMUM - PACKAGE_HEAD_SIZE - 1

// Import_Line is one line of the run of imports a file states. The read skips a line holding
// nothing ahead of it, thus the line holds one byte at least.
type Import_Line token.Source

// Import_Line_Invariants states every line of the run of imports.
func Import_Line_Invariants(value Import_Line, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(
    len(value),
    IMPORT_LINE_SIZE_MINIMUM,
    IMPORT_LINE_SIZE_MAXIMUM,
  ).
    Ensure()
}

// IMPORT_LINE_SIZE_MINIMUM is the narrowest line the read of imports stands on.
const IMPORT_LINE_SIZE_MINIMUM = 1

// IMPORT_PATH_SIZE_MAXIMUM is the widest path an import states, which is the widest line less
// the head that opens it and the two quotes that hold it.
const IMPORT_PATH_SIZE_MAXIMUM = IMPORT_LINE_SIZE_MAXIMUM - IMPORT_HEAD_SIZE - 2

// Import_Path is the path one import states.
type Import_Path token.Source

// Import_Path_Invariants states every path an import states, down to the path of no bytes.
func Import_Path_Invariants(value Import_Path, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(
    len(value),
    token.SOURCE_SIZE_MINIMUM,
    IMPORT_PATH_SIZE_MAXIMUM,
  ).
    Ensure()
}

// BUILD_LINE_SIZE_MAXIMUM is the widest constraint one header states, which is the widest source
// less the head that opens the line.
const BUILD_LINE_SIZE_MAXIMUM = token.SOURCE_SIZE_MAXIMUM - HEAD_SIZE_MAXIMUM

// Build_Line is the constraint one header states, as wide as the author wrote it.
type Build_Line token.Source

// Build_Line_Invariants states every constraint a header states, down to the one of no bytes.
func Build_Line_Invariants(value Build_Line, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(
    len(value),
    token.SOURCE_SIZE_MINIMUM,
    BUILD_LINE_SIZE_MAXIMUM,
  ).
    Ensure()
}

// Word is the target word one reader compares against.
type Word token.Source

// Word_Invariants states every bounded target-word view. The read compares a word against a tag
// of one constraint, and no tag stands wider than the widest constraint, thus a word it holds
// never passes that width.
func Word_Invariants(value Word, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(len(value), token.SOURCE_SIZE_MINIMUM, TEXT_SIZE_MAXIMUM).
    Ensure()
}

// Build_Sources keeps unlike reader views in named storage.
type Build_Sources struct {
  // Text survives the views taken from it.
  Text Constraint_Text
}

// Build_Sources_Invariants composes each reader-view domain once.
func Build_Sources_Invariants(value *Build_Sources, namespace aver.Namespace) {
  Constraint_Text_Invariants(value.Text, namespace)
}

// SYMBOL_MINIMUM is the smallest byte a term can open at.
const SYMBOL_MINIMUM = 0

// SYMBOL_MAXIMUM is the largest byte a term can open at. A term opens at whatever byte the author
// wrote there, thus every byte stands in the domain.
const SYMBOL_MAXIMUM = 255

// Held_Symbol is the byte one read of a tag stands on.
type Held_Symbol byte

// Held_Symbol_Invariants states every byte.
func Held_Symbol_Invariants(value Held_Symbol, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint8(uint8(value), SYMBOL_MINIMUM, SYMBOL_MAXIMUM).
    Ensure()
}

// AND_SIGN is the byte a meeting of terms writes twice.
const AND_SIGN = '&'

// OR_SIGN is the byte a join of terms writes twice.
const OR_SIGN = '|'

// Sign is the byte one operator of the constraint language writes twice.
type Sign byte

// Sign_Invariants states the two operators the language writes.
func Sign_Invariants(value Sign, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_Uint8(uint8(value), AND_SIGN, OR_SIGN).
    Ensure()
}

// HEAD_SIZE_MAXIMUM is the widest line prefix one build reader searches for.
const HEAD_SIZE_MAXIMUM = len(BUILD_HEAD)

// IMPORT_HEAD_SIZE is the byte count of the text one import states.
const IMPORT_HEAD_SIZE = len(IMPORT_HEAD)

// PACKAGE_HEAD_SIZE is the byte count of the text one package clause states, which the opening of
// an import block matches.
const PACKAGE_HEAD_SIZE = len(PACKAGE_HEAD)

// Head is one line prefix a build reader searches for.
type Head string

// Head_Invariants states every line prefix the reader searches for. Three widths stand and no
// other, thus a prefix of one byte or of two names nothing the reader looks for.
func Head_Invariants(value Head, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_3_Int(
    len(value),
    IMPORT_HEAD_SIZE,
    PACKAGE_HEAD_SIZE,
    HEAD_SIZE_MAXIMUM,
  ).
    Ensure()
}

// Opening_Place is the byte at which the first name field opens.
type Opening_Place Place

// Opening_Place_Invariants states every byte of one constraint text. The place stands at an
// underscore of that text or at its end, thus it never passes the width the text states.
func Opening_Place_Invariants(value Opening_Place, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int32(int32(value), PLACE_MINIMUM, TEXT_SIZE_MAXIMUM).
    Ensure()
}

// Field_Place is the byte at which the final name field opens.
type Field_Place Place

// Field_Place_Invariants states every byte of one constraint text. A field opens behind an
// underscore the text holds, and the read names one only where an underscore stands inside the
// text, thus the place never passes the width the text states.
func Field_Place_Invariants(value Field_Place, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int32(int32(value), STEM_SIZE_MINIMUM, TEXT_SIZE_MAXIMUM).
    Ensure()
}

// Target is the caller-owned target storage.
type Target struct {
  // Words holds the operating system and the architecture the build names.
  Words Target_Words
  // Tags holds the tags the build states beyond the two words.
  Tags Target_Tag_Storage
  // Counts holds how many of those tags the build states.
  Counts Target_Counts
}

// Target_Invariants composes every concrete target field once.
func Target_Invariants(value *Target, namespace aver.Namespace) {
  Target_Words_Invariants(&value.Words, namespace)
  Target_Tag_Storage_Invariants(&value.Tags, namespace)
  Target_Counts_Invariants(&value.Counts, namespace)
}

// Target_Handle gives caller-owned target storage one identity.
type Target_Handle *Target

// Target_Handle_Invariants composes a present target.
func Target_Handle_Invariants(value Target_Handle, namespace aver.Namespace) {
  if value == nil {
    return
  }
  Target_Invariants(value, namespace)
}

// Build is the caller-owned state one read of a constraint runs through. The text and the target
// stay with the caller, thus the state holds the place the read stands at and nothing else.
type Build struct {
  // Counts holds the byte the read stands on and the brackets it stands inside of.
  Counts Build_Counts
  // Sources holds the text one read runs over and the tag it stands on.
  Sources Build_Sources
}

// Build_Invariants fixes reader representation without proving stale mutable values.
func Build_Invariants(value *Build, namespace aver.Namespace) {
  Build_Counts_Invariants(&value.Counts, namespace)
  Build_Sources_Invariants(&value.Sources, namespace)
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
func Read_Result_Invariants(value *Read_Result, namespace aver.Namespace) {
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
  Build_Invariants(value, namespace)
}

// Holds_Constraint reports whether the constraint one file states holds in one target, and whether
// the reader read it. A text the reader refuses holds nothing, thus a caller that meets one keeps
// the file out of every build.
func Holds_Constraint(
  subject Build_Handle,
  one Target_Handle,
  text token.Source,
) (result Read_Result) {
  defer func() { Read_Result_Invariants(&result, "holds_constraint.result") }()
  Build_Handle_Invariants(subject, "holds_constraint.subject")
  Target_Handle_Invariants(one, "holds_constraint.one")
  token.Source_Invariants(text, "holds_constraint.text")
  if len(text) > TEXT_SIZE_MAXIMUM {
    return Read_Result{Held: false, OK: false}
  }
  subject.Sources.Text = Constraint_Text(text)
  subject.Counts.Place = COUNT_MINIMUM
  subject.Counts.Depth = COUNT_MINIMUM
  result = read_run(subject, one, OR_SIGN)
  if !bool(result.OK) {
    return Read_Result{Held: false, OK: false}
  }
  skip_spaces(subject)
  if int(subject.Counts.Place) != len(text) {
    return Read_Result{Held: false, OK: false}
  }
  return result
}

// Reads one run of sides the sign stated twice divides. A join holds where either side holds and
// divides meetings; a meeting holds where both sides hold and divides terms. A side the reader
// refused refuses the whole run, thus the read stops there.
func read_run(subject Build_Handle, one Target_Handle, sign Sign) (result Read_Result) {
  defer func() { Read_Result_Invariants(&result, "read_run.result") }()
  Build_Handle_Invariants(subject, "read_run.subject")
  Target_Handle_Invariants(one, "read_run.one")
  Sign_Invariants(sign, "read_run.sign")
  result = read_side(subject, one, sign)
  for range TEXT_SIZE_MAXIMUM {
    if !bool(result.OK) {
      return result
    }
    skip_spaces(subject)
    text := subject.Sources.Text
    place := int(subject.Counts.Place)
    // Every sign this constraint language states writes its byte twice.
    if place+2 > len(text) {
      return result
    }
    if text[place] != byte(sign) {
      return result
    }
    if text[place+1] != byte(sign) {
      return result
    }
    subject.Counts.Place = Text_Place(place + 2)
    next := read_side(subject, one, sign)
    result.OK = next.OK
    if sign == OR_SIGN {
      if bool(next.Held) {
        result.Held = true
      }
      continue
    }
    if !bool(next.Held) {
      result.Held = false
    }
  }
  return result
}

// Reads one side of a run: a meeting where the run is a join, and a term where it is a meeting.
func read_side(subject Build_Handle, one Target_Handle, sign Sign) (result Read_Result) {
  defer func() { Read_Result_Invariants(&result, "read_side.result") }()
  Build_Handle_Invariants(subject, "read_side.subject")
  Target_Handle_Invariants(one, "read_side.one")
  Sign_Invariants(sign, "read_side.sign")
  if sign == OR_SIGN {
    return read_run(subject, one, AND_SIGN)
  }
  return read_term(subject, one)
}

// Reads one term of a constraint: a tag, a term the exclamation reverses, or a group of terms
// between brackets.
func read_term(subject Build_Handle, one Target_Handle) (result Read_Result) {
  defer func() { Read_Result_Invariants(&result, "read_term.result") }()
  Build_Handle_Invariants(subject, "read_term.subject")
  Target_Handle_Invariants(one, "read_term.one")
  skip_spaces(subject)
  text := subject.Sources.Text
  place := int(subject.Counts.Place)
  if place == len(text) {
    return Read_Result{Held: false, OK: false}
  }
  if text[place] == '!' {
    subject.Counts.Place = Text_Place(place + 1)
    result = read_term(subject, one)
    // A term the reader refused states no truth to reverse.
    if !bool(result.OK) {
      return result
    }
    return Read_Result{Held: Boolean(!bool(result.Held)), OK: true}
  }
  if text[place] == '(' {
    depth := subject.Counts.Depth
    if !(int(depth) < DEPTH_MAXIMUM) {
      return Read_Result{Held: false, OK: false}
    }
    subject.Counts.Depth = depth + 1
    subject.Counts.Place = Text_Place(place + 1)
    result = read_run(subject, one, OR_SIGN)
    if !bool(result.OK) {
      return result
    }
    skip_spaces(subject)
    bracket := int(subject.Counts.Place)
    if bracket == len(text) {
      return Read_Result{Held: false, OK: false}
    }
    if text[bracket] != ')' {
      return Read_Result{Held: false, OK: false}
    }
    subject.Counts.Place = Text_Place(bracket + 1)
    subject.Counts.Depth = depth
    return result
  }
  // A term opening at a byte no tag holds states no tag, thus the read refuses it before it
  // takes one.
  if !bool(names_tag(Held_Symbol(text[place]))) {
    return Read_Result{Held: false, OK: false}
  }
  tag := take_tag(Term_Text(text[place:]))
  subject.Counts.Place = Text_Place(place + len(tag))
  return Read_Result{Held: holds_tag(tag, one), OK: true}
}

// Reads the tag one term opens with. The term opens at a byte a tag holds, thus the tag holds one
// byte at least.
func take_tag(term Term_Text) (tag Tag) {
  defer func() { Tag_Invariants(tag, "take_tag.tag") }()
  Term_Text_Invariants(term, "take_tag.term")
  place := 0
  for place < len(term) {
    if !bool(names_tag(Held_Symbol(term[place]))) {
      break
    }
    place = place + 1
  }
  return Tag(term[:place])
}

// Reports whether one byte stands inside a tag, which is a letter, a digit, an
// underscore, or a period.
func names_tag(value Held_Symbol) (yes Boolean) {
  defer func() { Boolean_Invariants(yes, "names_tag.yes") }()
  Held_Symbol_Invariants(value, "names_tag.value")
  if value == '_' {
    return true
  }
  if value == '.' {
    return true
  }
  if !(value < '0') {
    if value < ':' {
      return true
    }
  }
  if !(value < 'a') {
    if value < '{' {
      return true
    }
  }
  if !(value < 'A') {
    return Boolean(value < '[')
  }
  return false
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
func holds_tag(tag Tag, one Target_Handle) (yes Boolean) {
  defer func() { Boolean_Invariants(yes, "holds_tag.yes") }()
  Tag_Invariants(tag, "holds_tag.tag")
  Target_Handle_Invariants(one, "holds_tag.one")
  if string(tag) == UNIX_WORD {
    // No system Go calls a Unix spells a word as wide as the widest constraint, thus a
    // wider word names none of them.
    if len(one.Words.System) > TEXT_SIZE_MAXIMUM {
      return false
    }
    return names_unix(Word(one.Words.System))
  }
  if bool(same_word(tag, token.Source(one.Words.System))) {
    return true
  }
  if bool(same_word(tag, token.Source(one.Words.Architecture))) {
    return true
  }
  for slot := range int(one.Counts.Tag) {
    if bool(same_word(tag, one.Tags.Values[slot])) {
      return true
    }
  }
  return false
}

// Reports whether the tag one read stands on states the word it compares against. A word of no
// bytes names nothing, thus a target that names no system holds no tag at all.
func same_word(tag Tag, word token.Source) (yes Boolean) {
  defer func() { Boolean_Invariants(yes, "same_word.yes") }()
  Tag_Invariants(tag, "same_word.tag")
  token.Source_Invariants(word, "same_word.word")
  return Boolean(string(tag) == string(word))
}

// Names_System reports whether the name of one file states the build one target names. A name
// closing at a known system, a known architecture, or both stands in that build alone, and every
// other name stands in every build.
func Names_System(one Target_Handle, name token.Source) (held Boolean) {
  defer func() { Boolean_Invariants(held, "names_system.held") }()
  Target_Handle_Invariants(one, "names_system.one")
  token.Source_Invariants(name, "names_system.name")
  stem := take_stem(name)
  // No system and no architecture spells a word as wide as the widest constraint, thus a
  // stem that passes that width names neither and the read never holds it.
  if len(stem) > TEXT_SIZE_MAXIMUM {
    return true
  }
  opening := int(take_opening(Constraint_Text(stem)))
  if opening == len(stem) {
    return true
  }
  text := Stem(stem)
  tail := int(take_field(text))
  // A name closing at the word of a test file answers for the name ahead of that word, thus
  // the read cuts the word and the underscore that opened it.
  if string(text[tail:]) == TEST_WORD {
    text = text[:tail-1]
    if len(text) < opening+1 {
      return true
    }
    tail = int(take_field(text))
  }
  // A name states a system and an architecture only where a second field stands behind the
  // underscore that opened the run.
  if tail-1 > opening {
    head := int(take_field(text[:tail-1]))
    system := Name_Field(text[head : tail-1])
    if bool(known_system(system)) {
      machine := Name_Field(text[tail:])
      if bool(known_architecture(machine)) {
        architecture := token.Source(one.Words.Architecture)
        if !bool(same_word(Tag(machine), architecture)) {
          return false
        }
        return same_word(Tag(system), token.Source(one.Words.System))
      }
    }
  }
  return holds_word(Name_Field(text[tail:]), one)
}

// Holds_Imports reports whether every import one file states names this module or the standard
// library, and whether the reader read them. A path outside both names a third party, which
// stands in no build this package answers for.
func Holds_Imports(text token.Source) (result Read_Result) {
  defer func() { Read_Result_Invariants(&result, "holds_imports.result") }()
  token.Source_Invariants(text, "holds_imports.text")
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
    line := Line(text[place:end])
    place = end + 1
    if !stated {
      stated = bool(opens_line(PACKAGE_HEAD, line))
      continue
    }
    if len(line) == 0 {
      continue
    }
    if bool(states_no_import(Import_Line(line))) {
      continue
    }
    if opened {
      if string(line) == IMPORT_TAIL {
        opened = false
        continue
      }
    } else {
      if bool(opens_line(IMPORT_OPENING, line)) {
        opened = true
        continue
      }
      if !bool(opens_line(IMPORT_HEAD, line)) {
        // The imports of a file stand ahead of every other declaration,
        // thus the first one that follows closes the run of them.
        return Read_Result{Held: true, OK: true}
      }
    }
    count = count + 1
    if count > IMPORT_COUNT_MAXIMUM {
      return Read_Result{Held: false, OK: false}
    }
    read := holds_import_line(Import_Line(line))
    if !bool(read.OK) {
      return Read_Result{Held: false, OK: false}
    }
    if !bool(read.Held) {
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
func holds_import_line(line Import_Line) (result Read_Result) {
  defer func() { Read_Result_Invariants(&result, "holds_import_line.result") }()
  Import_Line_Invariants(line, "holds_import_line.line")
  path, found := take_import_path(line)
  if !bool(found) {
    return Read_Result{Held: false, OK: false}
  }
  return Read_Result{Held: Boolean(!bool(names_third_party(path))), OK: true}
}

// Reports whether one line states no import at all, which a comment does. A block of imports
// holds one between the paths it states.
func states_no_import(line Import_Line) (skipped Boolean) {
  defer func() { Boolean_Invariants(skipped, "states_no_import.skipped") }()
  Import_Line_Invariants(line, "states_no_import.line")
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

// Reads the path one line states, and reports whether it states one. A line whose quotes never
// close states no path the read can stand on.
func take_import_path(line Import_Line) (path Import_Path, found Boolean) {
  defer func() {
    Import_Path_Invariants(path, "take_import_path.path")
    Boolean_Invariants(found, "take_import_path.found")
  }()
  Import_Line_Invariants(line, "take_import_path.line")
  opening := 0
  for opening < len(line) {
    if line[opening] == '"' {
      break
    }
    opening = opening + 1
  }
  if opening == len(line) {
    return Import_Path(""), false
  }
  tail := opening + 1
  for tail < len(line) {
    if line[tail] == '"' {
      break
    }
    tail = tail + 1
  }
  if tail == len(line) {
    return Import_Path(""), false
  }
  return Import_Path(line[opening+1 : tail]), true
}

// Reports whether the path the word slot holds names a third party. The first element of a path
// the standard library states holds no period, and neither does one this module states, thus a
// period there names a host, which is how every module outside this one is named.
func names_third_party(path Import_Path) (yes Boolean) {
  defer func() { Boolean_Invariants(yes, "names_third_party.yes") }()
  Import_Path_Invariants(path, "names_third_party.path")
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
func take_opening(text Constraint_Text) (opening Opening_Place) {
  defer func() { Opening_Place_Invariants(opening, "take_opening.opening") }()
  Constraint_Text_Invariants(text, "take_opening.text")
  for index := range len(text) {
    if text[index] == '_' {
      return Opening_Place(index)
    }
  }
  return Opening_Place(len(text))
}

// Names the byte the final field of one name opens at, which is the byte behind the last
// underscore the stem holds. The read cuts a stem only behind the underscore that opened it, thus
// every stem it hands here holds one.
func take_field(text Stem) (field Field_Place) {
  defer func() { Field_Place_Invariants(field, "take_field.field") }()
  Stem_Invariants(text, "take_field.text")
  held := 0
  for index := range len(text) {
    if text[index] == '_' {
      held = index + 1
    }
  }
  return Field_Place(held)
}

// Reports whether one word of a name states the build the target names. A word that names neither
// a system nor an architecture says nothing, thus the file stands in every build.
func holds_word(field Name_Field, one Target_Handle) (yes Boolean) {
  defer func() { Boolean_Invariants(yes, "holds_word.yes") }()
  Name_Field_Invariants(field, "holds_word.field")
  Target_Handle_Invariants(one, "holds_word.one")
  if bool(known_system(field)) {
    return same_word(Tag(field), token.Source(one.Words.System))
  }
  if bool(known_architecture(field)) {
    return same_word(Tag(field), token.Source(one.Words.Architecture))
  }
  return true
}

// Reports whether one field of a name names an operating system the toolchain builds for.
func known_system(field Name_Field) (yes Boolean) {
  defer func() { Boolean_Invariants(yes, "known_system.yes") }()
  Name_Field_Invariants(field, "known_system.field")
  switch string(field) {
  case "aix", "android", "darwin", "dragonfly", "freebsd", "hurd", "illumos", "ios", "js",
    "linux", "nacl", "netbsd", "openbsd", "plan9", "solaris", "wasip1", "windows",
    "zos":
    return true
  }
  return false
}

// Reports whether one field of a name names an architecture the toolchain builds for.
func known_architecture(field Name_Field) (yes Boolean) {
  defer func() { Boolean_Invariants(yes, "known_architecture.yes") }()
  Name_Field_Invariants(field, "known_architecture.field")
  switch string(field) {
  case "386", "amd64", "amd64p32", "arm", "armbe", "arm64", "arm64be", "loong64", "mips",
    "mipsle", "mips64", "mips64le", "mips64p32", "mips64p32le", "ppc", "ppc64",
    "ppc64le", "riscv", "riscv64", "s390", "s390x", "sparc", "sparc64", "wasm":
    return true
  }
  return false
}

// Reports whether the word one read compares against names an operating system Go calls a Unix.
func names_unix(word Word) (yes Boolean) {
  defer func() { Boolean_Invariants(yes, "names_unix.yes") }()
  Word_Invariants(word, "names_unix.word")
  switch string(word) {
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
func Directory_Counts_Invariants(value *Directory_Counts, namespace aver.Namespace) {
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
func Directory_Flags_Invariants(value *Directory_Flags, namespace aver.Namespace) {
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
    len(value),
    STORAGE_SIZE_MINIMUM,
    nbio.DIRECTORY_BUFFER_SIZE_MAXIMUM,
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
}

// Directory_Memory_Invariants states every run of storage one read writes into.
func Directory_Memory_Invariants(value *Directory_Memory, namespace aver.Namespace) {
  Entry_Storage_Invariants(value.Entries, namespace)
  Record_Storage_Invariants(value.Records, namespace)
  Name_Storage_Invariants(value.Names, namespace)
  Name_Bytes_Invariants(value.Bytes, namespace)
  Header_Storage_Invariants(value.Header, namespace)
}

// Pass_Fields is the storage one pass over a directory holds. A step of a pass reads no
// constraint and names no target, thus it takes this and owes those domains nothing.
type Pass_Fields struct {
  // Completion stands first so the static callback recovers the runner with no closure.
  Completion nbio.Completion
  // Loop submits the storage work the caller drives.
  Loop Loop
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

// Pass_Fields_Invariants composes every invariant-bearing field of one pass once.
func Pass_Fields_Invariants(value *Pass_Fields, namespace aver.Namespace) {
  Entry_Storage_Invariants(value.Entries, namespace)
  Record_Storage_Invariants(value.Records, namespace)
  Name_Storage_Invariants(value.Names, namespace)
  Name_Bytes_Invariants(value.Bytes, namespace)
  Header_Storage_Invariants(value.Header, namespace)
  Directory_Counts_Invariants(&value.Counts, namespace)
  Directory_Flags_Invariants(&value.Flags, namespace)
  // The loop stands last because it states a backend: a runner the open has not reached
  // yet holds none, and the storage ahead of this line states its widths either way.
  Loop_Invariants(value.Loop, namespace)
}

// Pass_Handle gives caller-owned pass storage one identity.
type Pass_Handle *Pass_Fields

// Pass_Handle_Invariants composes present pass storage.
func Pass_Handle_Invariants(value Pass_Handle, namespace aver.Namespace) {
  if value == nil {
    return
  }
  Pass_Fields_Invariants(value, namespace)
}

// Directory_Runner reads one directory and keeps the Go files that stand in one build. It composes
// the injected storage the caller drives, thus this package opens nothing of its own and the
// composition root alone owns the loop.
type Directory_Runner struct {
  // Pass_Fields stands first so the static callback recovers the runner with no closure,
  // and so a step that reads no constraint takes the pass alone.
  Pass_Fields
}

// Directory_Runner_Invariants composes every invariant-bearing runner field once.
func Directory_Runner_Invariants(value *Directory_Runner, namespace aver.Namespace) {
  Pass_Fields_Invariants(&value.Pass_Fields, namespace)
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
  Directory_Runner_Invariants(value, namespace)
}

// Directory_Runner_Init opens one directory and states the read that follows. The caller owns the
// loop and every buffer the read writes into, thus the runner allocates nothing.
func Directory_Runner_Init(
  runner Directory_Runner_Handle,
  loop nbio.IO,
  path Path,
  memory Directory_Memory,
) {
  Directory_Runner_Handle_Invariants(runner, "directory_runner_init.runner")
  nbio.IO_Invariants(loop, "directory_runner_init.loop")
  Loop_Invariants(Loop(loop), "directory_runner_init.runner_loop")
  Path_Invariants(path, "directory_runner_init.path")
  Directory_Memory_Invariants(&memory, "directory_runner_init.memory")
  // The open states the whole runner ahead of the assertion that reads it, because a runner
  // the caller has not opened yet holds no loop, and no loop states no backend.
  counts, flags := runner.Counts, runner.Flags
  *runner = Directory_Runner{
    Pass_Fields: Pass_Fields{
      Loop:    Loop(loop),
      Entries: memory.Entries,
      Records: memory.Records,
      Names:   memory.Names,
      Bytes:   memory.Bytes,
      Header:  memory.Header,
      Counts:  counts,
      Flags:   flags,
    },
  }
  Directory_Runner_Handle_Invariants(runner, "directory_runner_init.opened")
  runner.Counts.Phase = PHASE_OPEN_DIRECTORY
  nbio.Storage_Open_At(
    runner.Loop.Storage,
    &runner.Completion,
    nbio.DIRECTORY_CURRENT,
    string(path),
    nbio.Open_At_Options{Access: nbio.OPEN_READ_ONLY},
    directory_completion,
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
func Directory_Runner_Rearm(runner Directory_Runner_Handle, one Target_Handle) (rearm Boolean) {
  defer func() { Boolean_Invariants(rearm, "directory_runner_rearm.rearm") }()
  Directory_Runner_Handle_Invariants(runner, "directory_runner_rearm.runner")
  Target_Handle_Invariants(one, "directory_runner_rearm.one")
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
    return directory_completion_apply(runner, one)
  }
  if runner.Counts.Phase != PHASE_IDLE {
    return false
  }
  open_next_file(runner, one)
  return false
}

// Directory_Runner_Work_Queued reports the callback stored one continuation for the caller.
func Directory_Runner_Work_Queued(runner Pass_Handle) (queued Boolean) {
  defer func() { Boolean_Invariants(queued, "directory_runner_work_queued.queued") }()
  Pass_Handle_Invariants(runner, "directory_runner_work_queued.runner")
  return Boolean(bool(runner.Flags.Queued))
}

// Directory_Runner_Stopped reports the terminal state.
func Directory_Runner_Stopped(runner Pass_Handle) (stopped Boolean) {
  defer func() { Boolean_Invariants(stopped, "directory_runner_stopped.stopped") }()
  Pass_Handle_Invariants(runner, "directory_runner_stopped.runner")
  return Boolean(bool(runner.Flags.Stopped))
}

// Directory_Runner_Status reports the error that stopped the runner.
func Directory_Runner_Status(runner Pass_Handle) (err error) {
  Pass_Handle_Invariants(runner, "directory_runner_status.runner")
  return runner.Result
}

// Directory_Runner_Names hands back the names of the files that stand in the build, which view the
// storage the caller owns.
func Directory_Runner_Names(runner Pass_Handle) (names Name_Storage) {
  defer func() { Name_Storage_Invariants(names, "directory_runner_names.names") }()
  Pass_Handle_Invariants(runner, "directory_runner_names.runner")
  aver.Always(
    bool(runner.Flags.Stopped),
    "A directory read is read after the runner stops.",
  )
  return Name_Storage(runner.Names[:runner.Counts.Name])
}

// Runs the continuation the read that completed states.
func directory_completion_apply(
  runner Directory_Runner_Handle, one Target_Handle,
) (rearm Boolean) {
  defer func() { Boolean_Invariants(rearm, "directory_completion_apply.rearm") }()
  Directory_Runner_Handle_Invariants(runner, "directory_completion_apply.runner")
  Target_Handle_Invariants(one, "directory_completion_apply.one")
  switch runner.Counts.Phase {
  case PHASE_OPEN_DIRECTORY:
    open_directory_apply(&runner.Pass_Fields)
    return false
  case PHASE_READ_ENTRIES:
    return read_entries_apply(&runner.Pass_Fields)
  case PHASE_OPEN_FILE:
    return open_file_apply(&runner.Pass_Fields)
  case PHASE_READ_HEADER:
    read_header_apply(runner, one)
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
func open_directory_apply(runner Pass_Handle) {
  Pass_Handle_Invariants(runner, "open_directory_apply.runner")
  if runner.Completion.Error != nil {
    runner.Result = runner.Completion.Error
    runner.Flags.Stopped = true
    return
  }
  runner.Directory = nbio.File(runner.Completion.Data)
  read_entries(runner)
}

// Stands the read at the first child of one pass. A pass of no child closes the read.
func read_entries_apply(runner Pass_Handle) (rearm Boolean) {
  defer func() { Boolean_Invariants(rearm, "read_entries_apply.rearm") }()
  Pass_Handle_Invariants(runner, "read_entries_apply.runner")
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
func open_file_apply(runner Pass_Handle) (rearm Boolean) {
  defer func() { Boolean_Invariants(rearm, "open_file_apply.rearm") }()
  Pass_Handle_Invariants(runner, "open_file_apply.runner")
  if runner.Completion.Error != nil {
    runner.Counts.Entry = runner.Counts.Entry + 1
    runner.Counts.Phase = PHASE_IDLE
    return true
  }
  runner.File = nbio.File(runner.Completion.Data)
  runner.Counts.Phase = PHASE_READ_HEADER
  nbio.Storage_Read(
    runner.Loop.Storage,
    &runner.Completion,
    runner.File,
    runner.Header,
    0,
    time.Duration(0),
    directory_completion,
  )
  return false
}

// Reads the constraint the header states and keeps the name where it holds.
func read_header_apply(runner Directory_Runner_Handle, one Target_Handle) {
  Directory_Runner_Handle_Invariants(runner, "read_header_apply.runner")
  Target_Handle_Invariants(one, "read_header_apply.one")
  count := 0
  if runner.Completion.Error == nil {
    count = int(runner.Completion.Data)
  }
  if count > len(runner.Header) {
    count = len(runner.Header)
  }
  // A header that states no constraint states no bar to the build, and one that states a
  // constraint the reader refused keeps the file out of every build.
  held := Boolean(true)
  stated, found := take_build_line(Scanned_Text(runner.Header[:count]))
  if bool(found) {
    // Every read of a constraint states its reader afresh, thus the reader is this
    // read's own and stands on the stack rather than on storage the runner holds.
    reader := Build{}
    result := Holds_Constraint(&reader, one, token.Source(stated))
    held = result.Held
    if !bool(result.OK) {
      held = false
    }
  }
  // A file naming a third party stands in no build, thus the read drops it the same way it
  // drops one whose constraint fails.
  if bool(held) {
    result := Holds_Imports(token.Source(runner.Header[:count]))
    held = result.Held
    if !bool(result.OK) {
      held = false
    }
  }
  if bool(held) {
    hold_name(&runner.Pass_Fields)
  }
  runner.Counts.Phase = PHASE_CLOSE_FILE
  nbio.IO_Close(nbio.IO(runner.Loop), &runner.Completion, runner.File, directory_completion)
}

// Reads the constraint line one header states into the tag slot, and reports whether it states
// one. That line stands ahead of the package clause, thus the read stops where the clause opens.
func take_build_line(scanned Scanned_Text) (stated Build_Line, found Boolean) {
  defer func() {
    Build_Line_Invariants(stated, "take_build_line.stated")
    Boolean_Invariants(found, "take_build_line.found")
  }()
  Scanned_Text_Invariants(scanned, "take_build_line.scanned")
  text := scanned
  place := 0
  for place < len(text) {
    end := place
    for end < len(text) {
      if text[end] == '\n' {
        break
      }
      end = end + 1
    }
    line := Line(text[place:end])
    if bool(opens_line(PACKAGE_HEAD, line)) {
      return Build_Line(""), false
    }
    if bool(opens_line(BUILD_HEAD, line)) {
      return Build_Line(text[place+len(BUILD_HEAD) : end]), true
    }
    place = end + 1
  }
  return Build_Line(""), false
}

// Reports whether the line the tag slot holds opens with the text stated.
func opens_line(head Head, line Line) (yes Boolean) {
  defer func() { Boolean_Invariants(yes, "opens_line.yes") }()
  Head_Invariants(head, "opens_line.head")
  Line_Invariants(line, "opens_line.line")
  if len(line) < len(head) {
    return false
  }
  return Boolean(string(line[:len(head)]) == string(head))
}

// Writes the name of the file the read stands on into the storage the caller owns, because the
// pass behind this one overwrites the entries it stands in. A name the storage cannot hold stops
// no read: the file stands unread rather than read wrongly.
func hold_name(runner Pass_Handle) {
  Pass_Handle_Invariants(runner, "hold_name.runner")
  name := runner.Entries[runner.Counts.Entry].Name
  if !(int(runner.Counts.Name) < len(runner.Names)) {
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
func read_entries(runner Pass_Handle) {
  Pass_Handle_Invariants(runner, "read_entries.runner")
  runner.Counts.Phase = PHASE_READ_ENTRIES
  nbio.Storage_Get_Directory_Entries(
    runner.Loop.Storage,
    &runner.Completion,
    runner.Directory,
    runner.Records,
    runner.Entries,
    directory_completion,
  )
}

// Opens the file the read stands on, over every child that names no Go file of this build. A pass
// the read stands at the end of opens the pass behind it.
func open_next_file(runner Directory_Runner_Handle, one Target_Handle) {
  Directory_Runner_Handle_Invariants(runner, "open_next_file.runner")
  Target_Handle_Invariants(one, "open_next_file.one")
  for int(runner.Counts.Entry) < int(runner.Counts.Entry_Total) {
    if bool(names_source(runner, one)) {
      break
    }
    runner.Counts.Entry = runner.Counts.Entry + 1
  }
  if !(int(runner.Counts.Entry) < int(runner.Counts.Entry_Total)) {
    read_entries(&runner.Pass_Fields)
    return
  }
  runner.Counts.Phase = PHASE_OPEN_FILE
  nbio.Storage_Open_At(
    runner.Loop.Storage,
    &runner.Completion,
    runner.Directory,
    runner.Entries[runner.Counts.Entry].Name,
    nbio.Open_At_Options{Access: nbio.OPEN_READ_ONLY},
    directory_completion,
  )
}

// Reports whether the child the read stands on names a Go file standing in the build the target
// names.
func names_source(runner Directory_Runner_Handle, one Target_Handle) (yes Boolean) {
  defer func() { Boolean_Invariants(yes, "names_source.yes") }()
  Directory_Runner_Handle_Invariants(runner, "names_source.runner")
  Target_Handle_Invariants(one, "names_source.one")
  entry := runner.Entries[runner.Counts.Entry]
  if entry.Is_Directory {
    return false
  }
  name := entry.Name
  if !(len(name) > len(SOURCE_TAIL)) {
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
  return Names_System(one, held)
}
