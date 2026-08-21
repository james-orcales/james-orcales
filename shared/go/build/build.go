// Package build answers whether one file stands in one build. It reads the name of the file and
// the constraint the file states, and it reads no file system and no environment: the caller owns
// the name, the text, and the target, thus one read allocates nothing and states no default.
package build

import (
	"unsafe"

	"local/james-orcales/shared/go/token"
	"local/james-orcales/shared/invariant/default"
	"local/james-orcales/shared/simulation/nbio"
	"local/james-orcales/shared/simulation/time"
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
func Boolean_Invariants(value Boolean, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Sometimes(bool(value), "The build report is true.").
		Ensure()
}

// Count names a byte of one text, or a count of the tags or the brackets one read holds.
type Count int32

// Count_Invariants states every count one read holds.
func Count_Invariants(value Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int32(int32(value), COUNT_MINIMUM, COUNT_MAXIMUM).
		Ensure()
}

// Place names one byte of the name of a file.
type Place int32

// Place_Invariants states every byte one name holds.
func Place_Invariants(value Place, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int32(int32(value), PLACE_MINIMUM, PLACE_MAXIMUM).
		Ensure()
}

// Target names the build one file stands in or stands outside of: the operating system, the
// architecture, and the tags the caller states.
type Target struct {
	// Words holds the operating system and the architecture the build names.
	Words [WORD_SLOT_COUNT]token.Source
	// Tags holds the tags the build states beyond the two words.
	Tags [TAG_COUNT_MAXIMUM]token.Source
	// Counts holds how many of those tags the build states.
	Counts [COUNT_SLOT_COUNT]Count
}

// Target_Invariants states one target holds a slot for every tag it names.
func Target_Invariants(subject *Target, namespace invariant.Namespace) {
	invariant.Always(
		len(subject.Tags) == TAG_COUNT_MAXIMUM,
		"A target holds one slot for every tag it names.",
	)
}

// Build is the caller-owned state one read of a constraint runs through. The text and the target
// stay with the caller, thus the state holds the place the read stands at and nothing else.
type Build struct {
	// Counts holds the byte the read stands on and the brackets it stands inside of.
	Counts [COUNT_SLOT_COUNT]Count
	// Sources holds the text one read runs over and the tag it stands on.
	Sources [SOURCE_SLOT_COUNT]token.Source
	// Symbols holds the byte one read stands on.
	Symbols [SYMBOL_SLOT_COUNT]byte
	// Heads holds the text one read asks a line to open with.
	Heads [HEAD_SLOT_COUNT]string
	// Places holds the bytes one read of a name stands between.
	Places [PLACE_SLOT_COUNT]Place
	// Flags holds what the read met.
	Flags [FLAG_SLOT_COUNT]Boolean
}

// Build_Invariants states one state holds a slot for every count a read keeps.
func Build_Invariants(subject *Build, namespace invariant.Namespace) {
	invariant.Always(
		len(subject.Counts) == COUNT_SLOT_COUNT,
		"A build state holds one slot for every count a read keeps.",
	)
}

// Holds_Constraint reports whether the constraint one file states holds in one target, and whether
// the reader read it. A text the reader refuses holds nothing, thus a caller that meets one keeps
// the file out of every build.
func Holds_Constraint(
	subject *Build, one *Target, text token.Source,
) (held Boolean, ok Boolean) {
	defer func() {
		Boolean_Invariants(held, "holds_constraint.held")
		Boolean_Invariants(ok, "holds_constraint.ok")
	}()
	Build_Invariants(subject, "holds_constraint.subject")
	Target_Invariants(one, "holds_constraint.one")
	token.Source_Invariants(text, "holds_constraint.text")
	if len(text) > TEXT_SIZE_MAXIMUM {
		return false, false
	}
	subject.Sources[SOURCE_TEXT] = text
	subject.Counts[COUNT_PLACE] = COUNT_MINIMUM
	subject.Counts[COUNT_DEPTH] = COUNT_MINIMUM
	subject.Flags[FLAG_FAILED] = false
	held = read_join(subject, one)
	skip_spaces(subject)
	if int(subject.Counts[COUNT_PLACE]) != len(text) {
		return false, false
	}
	if bool(subject.Flags[FLAG_FAILED]) {
		return false, false
	}
	return held, true
}

// Reads the tags one join of constraints holds, which is every tag either side of the two bars
// holds.
func read_join(subject *Build, one *Target) (held Boolean) {
	defer func() { Boolean_Invariants(held, "read_join.held") }()
	Build_Invariants(subject, "read_join.subject")
	Target_Invariants(one, "read_join.one")
	held = read_meet(subject, one)
	for range TEXT_SIZE_MAXIMUM {
		subject.Symbols[SYMBOL_HELD] = '|'
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
func read_meet(subject *Build, one *Target) (held Boolean) {
	defer func() { Boolean_Invariants(held, "read_meet.held") }()
	Build_Invariants(subject, "read_meet.subject")
	Target_Invariants(one, "read_meet.one")
	held = read_term(subject, one)
	for range TEXT_SIZE_MAXIMUM {
		subject.Symbols[SYMBOL_HELD] = '&'
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
func read_term(subject *Build, one *Target) (held Boolean) {
	defer func() { Boolean_Invariants(held, "read_term.held") }()
	Build_Invariants(subject, "read_term.subject")
	Target_Invariants(one, "read_term.one")
	skip_spaces(subject)
	if bool(subject.Flags[FLAG_FAILED]) {
		return false
	}
	text := subject.Sources[SOURCE_TEXT]
	place := int(subject.Counts[COUNT_PLACE])
	if place == len(text) {
		subject.Flags[FLAG_FAILED] = true
		return false
	}
	if text[place] == '!' {
		subject.Counts[COUNT_PLACE] = Count(place + 1)
		return !read_term(subject, one)
	}
	if text[place] == '(' {
		return read_group(subject, one)
	}
	take_tag(subject)
	if bool(subject.Flags[FLAG_FAILED]) {
		return false
	}
	return holds_tag(subject, one)
}

// Reads one group of terms and the bracket that closes it.
func read_group(subject *Build, one *Target) (held Boolean) {
	defer func() { Boolean_Invariants(held, "read_group.held") }()
	Build_Invariants(subject, "read_group.subject")
	Target_Invariants(one, "read_group.one")
	depth := subject.Counts[COUNT_DEPTH]
	if int(depth) >= DEPTH_MAXIMUM {
		subject.Flags[FLAG_FAILED] = true
		return false
	}
	subject.Counts[COUNT_DEPTH] = depth + 1
	subject.Counts[COUNT_PLACE] = subject.Counts[COUNT_PLACE] + 1
	held = read_join(subject, one)
	skip_spaces(subject)
	text := subject.Sources[SOURCE_TEXT]
	place := int(subject.Counts[COUNT_PLACE])
	if place == len(text) {
		subject.Flags[FLAG_FAILED] = true
		return false
	}
	if text[place] != ')' {
		subject.Flags[FLAG_FAILED] = true
		return false
	}
	subject.Counts[COUNT_PLACE] = Count(place + 1)
	subject.Counts[COUNT_DEPTH] = depth
	return held
}

// Reads the tag the text stands on into the tag slot. A text that states no tag where one belongs
// fails the read.
func take_tag(subject *Build) {
	Build_Invariants(subject, "take_tag.subject")
	text := subject.Sources[SOURCE_TEXT]
	opening := int(subject.Counts[COUNT_PLACE])
	place := opening
	for place < len(text) {
		subject.Symbols[SYMBOL_HELD] = text[place]
		if !bool(names_tag(subject)) {
			break
		}
		place = place + 1
	}
	if place == opening {
		subject.Flags[FLAG_FAILED] = true
		subject.Sources[SOURCE_TAG] = nil
		return
	}
	subject.Sources[SOURCE_TAG] = text[opening:place]
	subject.Counts[COUNT_PLACE] = Count(place)
}

// Reports whether the byte the tag slot holds stands inside a tag, which is a letter, a digit, an
// underscore, or a period.
func names_tag(subject *Build) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "names_tag.yes") }()
	Build_Invariants(subject, "names_tag.subject")
	value := subject.Symbols[SYMBOL_HELD]
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
func states_sign(subject *Build) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "states_sign.yes") }()
	Build_Invariants(subject, "states_sign.subject")
	skip_spaces(subject)
	value := subject.Symbols[SYMBOL_HELD]
	text := subject.Sources[SOURCE_TEXT]
	place := int(subject.Counts[COUNT_PLACE])
	if place+2 > len(text) {
		return false
	}
	if text[place] != value {
		return false
	}
	if text[place+1] != value {
		return false
	}
	subject.Counts[COUNT_PLACE] = Count(place + 2)
	return true
}

// Steps the read past the spaces and the tabs the author wrote between terms.
func skip_spaces(subject *Build) {
	Build_Invariants(subject, "skip_spaces.subject")
	text := subject.Sources[SOURCE_TEXT]
	place := int(subject.Counts[COUNT_PLACE])
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
	subject.Counts[COUNT_PLACE] = Count(place)
}

// Reports whether the tag the read stands on holds in one target: the operating system it names,
// the architecture it names, one of the tags it states, or the word every Unix system answers to.
func holds_tag(subject *Build, one *Target) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "holds_tag.yes") }()
	Build_Invariants(subject, "holds_tag.subject")
	Target_Invariants(one, "holds_tag.one")
	subject.Sources[SOURCE_WORD] = one.Words[WORD_SYSTEM]
	if string(subject.Sources[SOURCE_TAG]) == UNIX_WORD {
		return names_unix(subject)
	}
	if bool(same_word(subject)) {
		return true
	}
	subject.Sources[SOURCE_WORD] = one.Words[WORD_ARCHITECTURE]
	if bool(same_word(subject)) {
		return true
	}
	for slot := range int(one.Counts[COUNT_TAG]) {
		subject.Sources[SOURCE_WORD] = one.Tags[slot]
		if bool(same_word(subject)) {
			return true
		}
	}
	return false
}

// Reports whether the tag one read stands on states the word it compares against. A word of no
// bytes names nothing, thus a target that names no system holds no tag at all.
func same_word(subject *Build) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "same_word.yes") }()
	Build_Invariants(subject, "same_word.subject")
	one := subject.Sources[SOURCE_TAG]
	if len(one) == 0 {
		return false
	}
	return Boolean(string(one) == string(subject.Sources[SOURCE_WORD]))
}

// Names_System reports whether the name of one file states the build one target names. A name
// closing at a known system, a known architecture, or both stands in that build alone, and every
// other name stands in every build.
func Names_System(subject *Build, one *Target, name token.Source) (held Boolean) {
	defer func() { Boolean_Invariants(held, "names_system.held") }()
	Build_Invariants(subject, "names_system.subject")
	Target_Invariants(one, "names_system.one")
	token.Source_Invariants(name, "names_system.name")
	text := take_stem(name)
	subject.Sources[SOURCE_TEXT] = text
	take_opening(subject)
	opening := subject.Places[PLACE_OPENING]
	if int(opening) == len(text) {
		return true
	}
	take_field(subject)
	tail := subject.Places[PLACE_FIELD]
	// A name closing at the word of a test file answers for the name ahead of that word, thus
	// the read cuts the word and the underscore that opened it.
	if string(text[tail:]) == TEST_WORD {
		text = text[:tail-1]
		if len(text) < int(opening)+1 {
			return true
		}
		subject.Sources[SOURCE_TEXT] = text
		take_field(subject)
		tail = subject.Places[PLACE_FIELD]
	}
	// A name states a system and an architecture only where a second field stands behind the
	// underscore that opened the run.
	if tail-1 > opening {
		subject.Sources[SOURCE_TEXT] = text[:tail-1]
		take_field(subject)
		head := subject.Places[PLACE_FIELD]
		subject.Sources[SOURCE_TAG] = text[head : tail-1]
		if bool(known_system(subject)) {
			subject.Sources[SOURCE_TAG] = text[tail:]
			if bool(known_architecture(subject)) {
				subject.Sources[SOURCE_WORD] = one.Words[WORD_ARCHITECTURE]
				if !bool(same_word(subject)) {
					return false
				}
				subject.Sources[SOURCE_TAG] = text[head : tail-1]
				subject.Sources[SOURCE_WORD] = one.Words[WORD_SYSTEM]
				return same_word(subject)
			}
		}
	}
	subject.Sources[SOURCE_TAG] = text[tail:]
	return holds_word(subject, one)
}

// Holds_Imports reports whether every import one file states names this module or the standard
// library, and whether the reader read them. A path outside both names a third party, which
// stands in no build this package answers for.
func Holds_Imports(subject *Build, text token.Source) (held Boolean, ok Boolean) {
	defer func() {
		Boolean_Invariants(held, "holds_imports.held")
		Boolean_Invariants(ok, "holds_imports.ok")
	}()
	Build_Invariants(subject, "holds_imports.subject")
	token.Source_Invariants(text, "holds_imports.text")
	subject.Sources[SOURCE_TEXT] = text
	subject.Flags[FLAG_FAILED] = false
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
		subject.Sources[SOURCE_TAG] = text[place:end]
		place = end + 1
		if !stated {
			subject.Heads[HEAD_SLOT] = PACKAGE_HEAD
			stated = bool(opens_line(subject))
			continue
		}
		if bool(states_no_import(subject)) {
			continue
		}
		if opened {
			if string(subject.Sources[SOURCE_TAG]) == IMPORT_TAIL {
				opened = false
				continue
			}
		} else {
			subject.Heads[HEAD_SLOT] = IMPORT_OPENING
			if bool(opens_line(subject)) {
				opened = true
				continue
			}
			subject.Heads[HEAD_SLOT] = IMPORT_HEAD
			if !bool(opens_line(subject)) {
				// The imports of a file stand ahead of every other declaration,
				// thus the first one that follows closes the run of them.
				return true, true
			}
		}
		count = count + 1
		if count > IMPORT_COUNT_MAXIMUM {
			return false, false
		}
		one, read := holds_import_line(subject)
		if !bool(read) {
			return false, false
		}
		if !bool(one) {
			return false, true
		}
	}
	// A block the text never closes states an import the read never saw, thus the read states
	// no truth about the file at all.
	if opened {
		return false, false
	}
	return true, true
}

// Reads the path one line states and reports whether it stands in the build, and whether the line
// stated a path at all.
func holds_import_line(subject *Build) (held Boolean, ok Boolean) {
	defer func() {
		Boolean_Invariants(held, "holds_import_line.held")
		Boolean_Invariants(ok, "holds_import_line.ok")
	}()
	Build_Invariants(subject, "holds_import_line.subject")
	if !bool(take_import_path(subject)) {
		return false, false
	}
	return Boolean(!bool(names_third_party(subject))), true
}

// Reports whether the line the tag slot holds states no import at all, which a blank line and a
// comment both do. A block of imports holds either between the paths it states.
func states_no_import(subject *Build) (skipped Boolean) {
	defer func() { Boolean_Invariants(skipped, "states_no_import.skipped") }()
	Build_Invariants(subject, "states_no_import.subject")
	line := subject.Sources[SOURCE_TAG]
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
func take_import_path(subject *Build) (found Boolean) {
	defer func() { Boolean_Invariants(found, "take_import_path.found") }()
	Build_Invariants(subject, "take_import_path.subject")
	line := subject.Sources[SOURCE_TAG]
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
	subject.Sources[SOURCE_WORD] = line[opening+1 : tail]
	return true
}

// Reports whether the path the word slot holds names a third party. The first element of a path
// the standard library states holds no period, and neither does one this module states, thus a
// period there names a host, which is how every module outside this one is named.
func names_third_party(subject *Build) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "names_third_party.yes") }()
	Build_Invariants(subject, "names_third_party.subject")
	path := subject.Sources[SOURCE_WORD]
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
func take_opening(subject *Build) {
	Build_Invariants(subject, "take_opening.subject")
	text := subject.Sources[SOURCE_TEXT]
	for index := range len(text) {
		if text[index] == '_' {
			subject.Places[PLACE_OPENING] = Place(index)
			return
		}
	}
	subject.Places[PLACE_OPENING] = Place(len(text))
}

// Names the byte the final field of one name opens at, which is the byte behind the last
// underscore standing at or behind the opening one.
func take_field(subject *Build) {
	Build_Invariants(subject, "take_field.subject")
	text := subject.Sources[SOURCE_TEXT]
	opening := subject.Places[PLACE_OPENING]
	held := opening + 1
	for index := int(opening); index < len(text); index++ {
		if text[index] == '_' {
			held = Place(index + 1)
		}
	}
	subject.Places[PLACE_FIELD] = held
}

// Reports whether one word of a name states the build the target names. A word that names neither
// a system nor an architecture says nothing, thus the file stands in every build.
func holds_word(subject *Build, one *Target) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "holds_word.yes") }()
	Build_Invariants(subject, "holds_word.subject")
	Target_Invariants(one, "holds_word.one")
	if bool(known_system(subject)) {
		subject.Sources[SOURCE_WORD] = one.Words[WORD_SYSTEM]
		return same_word(subject)
	}
	if bool(known_architecture(subject)) {
		subject.Sources[SOURCE_WORD] = one.Words[WORD_ARCHITECTURE]
		return same_word(subject)
	}
	return true
}

// Reports whether the tag one read stands on names an operating system the toolchain builds for.
func known_system(subject *Build) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "known_system.yes") }()
	Build_Invariants(subject, "known_system.subject")
	switch string(subject.Sources[SOURCE_TAG]) {
	case "aix", "android", "darwin", "dragonfly", "freebsd", "hurd", "illumos", "ios", "js",
		"linux", "nacl", "netbsd", "openbsd", "plan9", "solaris", "wasip1", "windows",
		"zos":
		return true
	}
	return false
}

// Reports whether the tag one read stands on names an architecture the toolchain builds for.
func known_architecture(subject *Build) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "known_architecture.yes") }()
	Build_Invariants(subject, "known_architecture.subject")
	switch string(subject.Sources[SOURCE_TAG]) {
	case "386", "amd64", "amd64p32", "arm", "armbe", "arm64", "arm64be", "loong64", "mips",
		"mipsle", "mips64", "mips64le", "mips64p32", "mips64p32le", "ppc", "ppc64",
		"ppc64le", "riscv", "riscv64", "s390", "s390x", "sparc", "sparc64", "wasm":
		return true
	}
	return false
}

// Reports whether the word one read compares against names an operating system Go calls a Unix.
func names_unix(subject *Build) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "names_unix.yes") }()
	Build_Invariants(subject, "names_unix.subject")
	switch string(subject.Sources[SOURCE_WORD]) {
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

// Path names the directory one read stands on.
type Path string

// Path_Invariants states every path one read of a directory admits.
func Path_Invariants(value Path, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
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
func Loop_Invariants(value Loop, namespace invariant.Namespace) {
	invariant.Always(
		value.Network.State == value.Storage.State,
		"A directory read carries one backend across both halves.",
	)
}

// Entry_Storage receives one pass over the children of a directory, which the caller owns.
type Entry_Storage []nbio.Directory_Entry

// Entry_Storage_Invariants states every run of slots one pass writes into.
func Entry_Storage_Invariants(value Entry_Storage, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), STORAGE_SIZE_MINIMUM, ENTRY_COUNT_MAXIMUM).
		Ensure()
}

// Record_Storage receives the records the backend writes for one pass, which the caller owns.
type Record_Storage []byte

// Record_Storage_Invariants states every run of bytes one pass writes into. The widest one is the
// block budget the backend states, which refuses a pass wider than one.
func Record_Storage_Invariants(value Record_Storage, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(
			len(value), STORAGE_SIZE_MINIMUM, nbio.DIRECTORY_BUFFER_SIZE_MAXIMUM,
		).
		Ensure()
}

// Name_Bytes holds the bytes the names one read keeps view, which the caller owns.
type Name_Bytes []byte

// Name_Bytes_Invariants states every run of bytes the names of one read view.
func Name_Bytes_Invariants(value Name_Bytes, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), STORAGE_SIZE_MINIMUM, STORAGE_SIZE_MAXIMUM).
		Ensure()
}

// Header_Storage holds the bytes one file opens with, which the caller owns.
type Header_Storage []byte

// Header_Storage_Invariants states every run of bytes one header read writes into.
func Header_Storage_Invariants(value Header_Storage, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), STORAGE_SIZE_MINIMUM, STORAGE_SIZE_MAXIMUM).
		Ensure()
}

// Name_Storage holds the names one read of a directory keeps, which view the bytes the caller
// owns.
type Name_Storage []token.Source

// Name_Storage_Invariants states every run of names one read hands back.
func Name_Storage_Invariants(value Name_Storage, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
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
	Reader *Build
}

// Directory_Memory_Invariants states every run of storage one read writes into.
func Directory_Memory_Invariants(value Directory_Memory, namespace invariant.Namespace) {
	Entry_Storage_Invariants(value.Entries, namespace)
	Record_Storage_Invariants(value.Records, namespace)
	Name_Storage_Invariants(value.Names, namespace)
	Name_Bytes_Invariants(value.Bytes, namespace)
	Header_Storage_Invariants(value.Header, namespace)
	Build_Invariants(value.Reader, namespace)
}

// Directory_Runner reads one directory and keeps the Go files that stand in one build. It composes
// the injected storage the caller drives, thus this package opens nothing of its own and the
// composition root alone owns the loop.
type Directory_Runner struct {
	// Completion stands first so the static callback recovers the runner with no closure.
	Completion nbio.Completion
	// Loop submits the storage work the caller drives.
	Loop Loop
	// Target names the build the files stand in or stand outside of.
	Target *Target
	// Reader holds the state one read of a constraint runs through, which the caller owns.
	Reader *Build
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
	Counts [DIRECTORY_COUNT_SLOT_COUNT]Count
	// Flags holds what the runner met: a continuation one callback stood, and the stop.
	Flags [DIRECTORY_FLAG_SLOT_COUNT]Boolean
	// Directory is the descriptor the read holds between its open and its close.
	Directory nbio.File
	// File is the descriptor one header read holds.
	File nbio.File
	// Result retains the terminal error.
	Result error
}

// Directory_Runner_Invariants states the counts one read of a directory holds.
func Directory_Runner_Invariants(runner *Directory_Runner, namespace invariant.Namespace) {
	invariant.Always(runner != nil, "A directory runner has caller-owned state.")
	Entry_Storage_Invariants(runner.Entries, namespace)
	Record_Storage_Invariants(runner.Records, namespace)
	Name_Storage_Invariants(runner.Names, namespace)
	Name_Bytes_Invariants(runner.Bytes, namespace)
	Header_Storage_Invariants(runner.Header, namespace)
	Build_Invariants(runner.Reader, namespace)
	Target_Invariants(runner.Target, namespace)
	// The loop stands last because it states a backend: a runner the open has not reached
	// yet holds none, and the storage ahead of this line states its widths either way.
	Loop_Invariants(runner.Loop, namespace)
}

// Directory_Runner_Init opens one directory and states the read that follows. The caller owns the
// loop, the target, and every buffer the read writes into, thus the runner allocates nothing.
func Directory_Runner_Init(
	runner *Directory_Runner, loop nbio.IO, one *Target, path Path, memory Directory_Memory,
) {
	nbio.IO_Invariants(loop, "directory_runner_init.loop")
	Target_Invariants(one, "directory_runner_init.one")
	Path_Invariants(path, "directory_runner_init.path")
	Directory_Memory_Invariants(memory, "directory_runner_init.memory")
	// The open states the whole runner ahead of the assertion that reads it, because a runner
	// the caller has not opened yet holds no loop, and no loop states no backend.
	*runner = Directory_Runner{
		Loop: Loop(loop), Target: one, Reader: memory.Reader, Entries: memory.Entries,
		Records: memory.Records, Names: memory.Names, Bytes: memory.Bytes,
		Header: memory.Header,
	}
	Directory_Runner_Invariants(runner, "directory_runner_init.opened")
	runner.Counts[DIRECTORY_COUNT_PHASE] = PHASE_OPEN_DIRECTORY
	nbio.Storage_Open_At(
		runner.Loop.Storage, &runner.Completion, nbio.DIRECTORY_CURRENT, string(path),
		nbio.Open_At_Options{Access: nbio.OPEN_READ_ONLY}, directory_completion,
	)
}

// Holds the continuation one completion queued, which the caller reads through the rearm.
func directory_completion(completion *nbio.Completion) {
	// First-field ownership avoids a closure allocation and a self-pointer escape.
	runner := (*Directory_Runner)(unsafe.Pointer(completion))
	runner.Flags[DIRECTORY_FLAG_QUEUED] = true
}

// Directory_Runner_Rearm runs the continuation the callback queued, or the step the runner owes of
// its own. False means the caller drives the pending read, or the runner stopped.
func Directory_Runner_Rearm(runner *Directory_Runner) (rearm Boolean) {
	defer func() { Boolean_Invariants(rearm, "directory_runner_rearm.rearm") }()
	Directory_Runner_Invariants(runner, "directory_runner_rearm.runner")
	if bool(runner.Flags[DIRECTORY_FLAG_STOPPED]) {
		return false
	}
	// A read the caller gave no room for one child reads nothing, thus it stops where it
	// stands rather than ask the backend for a pass it cannot hold.
	if len(runner.Entries) == STORAGE_SIZE_MINIMUM {
		runner.Flags[DIRECTORY_FLAG_STOPPED] = true
		return false
	}
	// A read the caller gave no storage reads nothing, thus it stops where it stands rather
	// than ask the backend for a pass it has nowhere to write.
	if bool(runner.Flags[DIRECTORY_FLAG_QUEUED]) {
		runner.Flags[DIRECTORY_FLAG_QUEUED] = false
		return directory_completion_apply(runner)
	}
	if runner.Counts[DIRECTORY_COUNT_PHASE] != PHASE_IDLE {
		return false
	}
	open_next_file(runner)
	return false
}

// Directory_Runner_Work_Queued reports the callback stored one continuation for the caller.
func Directory_Runner_Work_Queued(runner *Directory_Runner) (queued Boolean) {
	defer func() { Boolean_Invariants(queued, "directory_runner_work_queued.queued") }()
	Directory_Runner_Invariants(runner, "directory_runner_work_queued.runner")
	return Boolean(bool(runner.Flags[DIRECTORY_FLAG_QUEUED]))
}

// Directory_Runner_Stopped reports the terminal state.
func Directory_Runner_Stopped(runner *Directory_Runner) (stopped Boolean) {
	defer func() { Boolean_Invariants(stopped, "directory_runner_stopped.stopped") }()
	Directory_Runner_Invariants(runner, "directory_runner_stopped.runner")
	return Boolean(bool(runner.Flags[DIRECTORY_FLAG_STOPPED]))
}

// Directory_Runner_Status reports the error that stopped the runner.
func Directory_Runner_Status(runner *Directory_Runner) (err error) {
	Directory_Runner_Invariants(runner, "directory_runner_status.runner")
	return runner.Result
}

// Directory_Runner_Names hands back the names of the files that stand in the build, which view the
// storage the caller owns.
func Directory_Runner_Names(runner *Directory_Runner) (names Name_Storage) {
	defer func() { Name_Storage_Invariants(names, "directory_runner_names.names") }()
	Directory_Runner_Invariants(runner, "directory_runner_names.runner")
	invariant.Always(
		bool(runner.Flags[DIRECTORY_FLAG_STOPPED]),
		"A directory read is read after the runner stops.",
	)
	return Name_Storage(runner.Names[:runner.Counts[DIRECTORY_COUNT_NAME]])
}

// Runs the continuation the read that completed states.
func directory_completion_apply(runner *Directory_Runner) (rearm Boolean) {
	defer func() { Boolean_Invariants(rearm, "directory_completion_apply.rearm") }()
	Directory_Runner_Invariants(runner, "directory_completion_apply.runner")
	switch runner.Counts[DIRECTORY_COUNT_PHASE] {
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
		runner.Counts[DIRECTORY_COUNT_ENTRY] = runner.Counts[DIRECTORY_COUNT_ENTRY] + 1
		runner.Counts[DIRECTORY_COUNT_PHASE] = PHASE_IDLE
		return true
	}
	runner.Flags[DIRECTORY_FLAG_STOPPED] = true
	return false
}

// Holds the descriptor the open of the directory answered with, and reads its first pass.
func open_directory_apply(runner *Directory_Runner) {
	Directory_Runner_Invariants(runner, "open_directory_apply.runner")
	if runner.Completion.Error != nil {
		runner.Result = runner.Completion.Error
		runner.Flags[DIRECTORY_FLAG_STOPPED] = true
		return
	}
	runner.Directory = nbio.File(runner.Completion.Data)
	read_entries(runner)
}

// Stands the read at the first child of one pass. A pass of no child closes the read.
func read_entries_apply(runner *Directory_Runner) (rearm Boolean) {
	defer func() { Boolean_Invariants(rearm, "read_entries_apply.rearm") }()
	Directory_Runner_Invariants(runner, "read_entries_apply.runner")
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
		runner.Counts[DIRECTORY_COUNT_PHASE] = PHASE_CLOSE_DIRECTORY
		nbio.IO_Close(
			nbio.IO(runner.Loop), &runner.Completion, runner.Directory,
			directory_completion,
		)
		return false
	}
	runner.Counts[DIRECTORY_COUNT_ENTRY_TOTAL] = held
	runner.Counts[DIRECTORY_COUNT_ENTRY] = NAME_COUNT_MINIMUM
	runner.Counts[DIRECTORY_COUNT_PHASE] = PHASE_IDLE
	return true
}

// Holds the descriptor the open of one file answered with, and reads the bytes it opens with. A
// file the open refused stands outside every build the runner answers for.
func open_file_apply(runner *Directory_Runner) (rearm Boolean) {
	defer func() { Boolean_Invariants(rearm, "open_file_apply.rearm") }()
	Directory_Runner_Invariants(runner, "open_file_apply.runner")
	if runner.Completion.Error != nil {
		runner.Counts[DIRECTORY_COUNT_ENTRY] = runner.Counts[DIRECTORY_COUNT_ENTRY] + 1
		runner.Counts[DIRECTORY_COUNT_PHASE] = PHASE_IDLE
		return true
	}
	runner.File = nbio.File(runner.Completion.Data)
	runner.Counts[DIRECTORY_COUNT_PHASE] = PHASE_READ_HEADER
	nbio.Storage_Read(
		runner.Loop.Storage, &runner.Completion, runner.File, runner.Header, 0,
		time.Duration(0), directory_completion,
	)
	return false
}

// Reads the constraint the header states and keeps the name where it holds.
func read_header_apply(runner *Directory_Runner) {
	Directory_Runner_Invariants(runner, "read_header_apply.runner")
	count := 0
	if runner.Completion.Error == nil {
		count = int(runner.Completion.Data)
	}
	if count > len(runner.Header) {
		count = len(runner.Header)
	}
	runner.Reader.Sources[SOURCE_TEXT] = token.Source(runner.Header[:count])
	// A header that states no constraint states no bar to the build, and one that states a
	// constraint the reader refused keeps the file out of every build.
	held := Boolean(true)
	if bool(take_build_line(runner.Reader)) {
		one, ok := Holds_Constraint(
			runner.Reader, runner.Target, runner.Reader.Sources[SOURCE_TAG],
		)
		held = one
		if !bool(ok) {
			held = false
		}
	}
	// A file naming a third party stands in no build, thus the read drops it the same way it
	// drops one whose constraint fails.
	if bool(held) {
		one, ok := Holds_Imports(runner.Reader, token.Source(runner.Header[:count]))
		held = one
		if !bool(ok) {
			held = false
		}
	}
	if bool(held) {
		hold_name(runner)
	}
	runner.Counts[DIRECTORY_COUNT_PHASE] = PHASE_CLOSE_FILE
	nbio.IO_Close(nbio.IO(runner.Loop), &runner.Completion, runner.File, directory_completion)
}

// Reads the constraint line one header states into the tag slot, and reports whether it states
// one. That line stands ahead of the package clause, thus the read stops where the clause opens.
func take_build_line(subject *Build) (found Boolean) {
	defer func() { Boolean_Invariants(found, "take_build_line.found") }()
	Build_Invariants(subject, "take_build_line.subject")
	text := subject.Sources[SOURCE_TEXT]
	place := 0
	for place < len(text) {
		end := place
		for end < len(text) {
			if text[end] == '\n' {
				break
			}
			end = end + 1
		}
		subject.Sources[SOURCE_TAG] = text[place:end]
		subject.Heads[HEAD_SLOT] = PACKAGE_HEAD
		if bool(opens_line(subject)) {
			return false
		}
		subject.Heads[HEAD_SLOT] = BUILD_HEAD
		if bool(opens_line(subject)) {
			subject.Sources[SOURCE_TAG] = text[place+len(BUILD_HEAD) : end]
			return true
		}
		place = end + 1
	}
	return false
}

// Reports whether the line the tag slot holds opens with the text stated.
func opens_line(subject *Build) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "opens_line.yes") }()
	Build_Invariants(subject, "opens_line.subject")
	line := subject.Sources[SOURCE_TAG]
	head := subject.Heads[HEAD_SLOT]
	if len(line) < len(head) {
		return false
	}
	return Boolean(string(line[:len(head)]) == head)
}

// Writes the name of the file the read stands on into the storage the caller owns, because the
// pass behind this one overwrites the entries it stands in. A name the storage cannot hold stops
// no read: the file stands unread rather than read wrongly.
func hold_name(runner *Directory_Runner) {
	Directory_Runner_Invariants(runner, "hold_name.runner")
	name := runner.Entries[runner.Counts[DIRECTORY_COUNT_ENTRY]].Name
	if int(runner.Counts[DIRECTORY_COUNT_NAME]) >= len(runner.Names) {
		return
	}
	written := int(runner.Counts[DIRECTORY_COUNT_WRITTEN])
	if written+len(name) > len(runner.Bytes) {
		return
	}
	copy(runner.Bytes[written:], name)
	held := runner.Bytes[written : written+len(name)]
	runner.Names[runner.Counts[DIRECTORY_COUNT_NAME]] = token.Source(held)
	runner.Counts[DIRECTORY_COUNT_NAME] = runner.Counts[DIRECTORY_COUNT_NAME] + 1
	runner.Counts[DIRECTORY_COUNT_WRITTEN] = Count(written + len(name))
}

// Reads one pass over the children of the directory.
func read_entries(runner *Directory_Runner) {
	Directory_Runner_Invariants(runner, "read_entries.runner")
	runner.Counts[DIRECTORY_COUNT_PHASE] = PHASE_READ_ENTRIES
	nbio.Storage_Get_Directory_Entries(
		runner.Loop.Storage, &runner.Completion, runner.Directory,
		runner.Records, runner.Entries, directory_completion,
	)
}

// Opens the file the read stands on, over every child that names no Go file of this build. A pass
// the read stands at the end of opens the pass behind it.
func open_next_file(runner *Directory_Runner) {
	Directory_Runner_Invariants(runner, "open_next_file.runner")
	for runner.Counts[DIRECTORY_COUNT_ENTRY] < runner.Counts[DIRECTORY_COUNT_ENTRY_TOTAL] {
		if bool(names_source(runner)) {
			break
		}
		runner.Counts[DIRECTORY_COUNT_ENTRY] = runner.Counts[DIRECTORY_COUNT_ENTRY] + 1
	}
	if runner.Counts[DIRECTORY_COUNT_ENTRY] >= runner.Counts[DIRECTORY_COUNT_ENTRY_TOTAL] {
		read_entries(runner)
		return
	}
	runner.Counts[DIRECTORY_COUNT_PHASE] = PHASE_OPEN_FILE
	nbio.Storage_Open_At(
		runner.Loop.Storage, &runner.Completion, runner.Directory,
		runner.Entries[runner.Counts[DIRECTORY_COUNT_ENTRY]].Name,
		nbio.Open_At_Options{Access: nbio.OPEN_READ_ONLY}, directory_completion,
	)
}

// Reports whether the child the read stands on names a Go file standing in the build the target
// names.
func names_source(runner *Directory_Runner) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "names_source.yes") }()
	Directory_Runner_Invariants(runner, "names_source.runner")
	entry := runner.Entries[runner.Counts[DIRECTORY_COUNT_ENTRY]]
	if entry.Is_Directory {
		return false
	}
	held := token.Source(entry.Name)
	if len(held) <= len(SOURCE_TAIL) {
		return false
	}
	if string(held[len(held)-len(SOURCE_TAIL):]) != SOURCE_TAIL {
		return false
	}
	return Names_System(runner.Reader, runner.Target, held)
}
