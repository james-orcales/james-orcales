// Package format states the canonical form of one Go source in this dialect. It exists because
// the printer answers what a tree writes and a caller asks something else: whether the file on
// disk already stands in the form this repository keeps. The printer is the engine and this is
// the policy above it, thus one place states the form and every caller reads the same answer.
package format

import (
	"local/james-orcales/shared/go/ast"
	"local/james-orcales/shared/go/printer"
	"local/james-orcales/shared/go/token"
	"local/james-orcales/shared/sim/aver/default"
)

// IMPORT_HEAD is the text one import declaration opens with.
const IMPORT_HEAD = "import "

// NOTE_HEAD is the text one note opens with.
const NOTE_HEAD = "//"

// PLACE_MINIMUM is the first byte of one form.
const PLACE_MINIMUM = 0

// PLACE_MAXIMUM is the last byte one form admits.
const PLACE_MAXIMUM = printer.FORM_SIZE_MAXIMUM

// PLACE_LIMIT holds the byte the written form closes at.
const PLACE_LIMIT = 0

// PLACE_CURSOR holds the byte one read of the form stands on.
const PLACE_CURSOR = 1

// PLACE_HEAD holds the byte one run of imports opens at.
const PLACE_HEAD = 2

// PLACE_TAIL holds the byte just past the run of imports.
const PLACE_TAIL = 3

// PLACE_SCAN holds the byte the record one scan stands on opens at.
const PLACE_SCAN = 4

// PLACE_SCAN_END holds the byte just past the record one scan stands on.
const PLACE_SCAN_END = 5

// PLACE_BEST holds the byte the smallest record of the run opens at.
const PLACE_BEST = 6

// PLACE_BEST_END holds the byte just past the smallest record of the run.
const PLACE_BEST_END = 7

// PLACE_FROM holds the byte one turn of the form opens at.
const PLACE_FROM = 8

// PLACE_MIDDLE holds the byte one turn of the form carries to its opening.
const PLACE_MIDDLE = 9

// PLACE_TO holds the byte just past one turn of the form.
const PLACE_TO = 10

// PLACE_WORD holds the byte one identifier of the form opens at.
const PLACE_WORD = 11

// PLACE_WORD_END holds the byte just past one identifier of the form.
const PLACE_WORD_END = 12

// PLACE_NAMES holds the count of names the rename table holds.
const PLACE_NAMES = 13

// PLACE_FOUND holds the entry of the rename table one word names, or the count of names when no
// entry names it.
const PLACE_FOUND = 14

// PLACE_SIZE holds the byte count of the name one spell writes.
const PLACE_SIZE = 15

// PLACE_START holds the byte of a name one word of it opens at.
const PLACE_START = 16

// PLACE_END holds the byte of a name just past one word of it.
const PLACE_END = 17

// PLACE_TEXT holds the byte count of the table text the table holds.
const PLACE_TEXT = 18

// PLACE_TOKEN holds the token of the candidate name one declaration tests.
const PLACE_TOKEN = 19

// PLACE_SLOT holds the empty slot of the hash index one probe stops at.
const PLACE_SLOT = 20

// PLACE_DROPPED holds the count of names one declaration found no room for.
const PLACE_DROPPED = 21

// PLACE_SLOT_COUNT is the slot count one formatter holds.
const PLACE_SLOT_COUNT = 22

// FORM_SLOT holds the storage the format wrote its form into.
const FORM_SLOT = 0

// FORM_SLOT_COUNT is the storage slot count one formatter holds.
const FORM_SLOT_COUNT = 1

// HEAD_SLOT holds the text one read of a line asks the line to open with.
const HEAD_SLOT = 0

// HEAD_SLOT_COUNT is the text slot count one formatter holds.
const HEAD_SLOT_COUNT = 1

// Place names one position a format reads or writes: a byte of the form, a byte of a name, or an
// entry of the rename table. Each fits the widest form.
type Place int32

// Place_Invariants states every byte one form holds.
func Place_Invariants(value Place, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), PLACE_MINIMUM, PLACE_MAXIMUM).
		Ensure()
}

// Places keeps bounded sort positions in caller storage.
type Places []Place

// Places_Invariants fixes every sort position and prevents growth.
func Places_Invariants(value Places, _ aver.Namespace) {
	aver.Always(len(value) == PLACE_SLOT_COUNT, "Formatter places have complete length.")
	aver.Always(cap(value) == PLACE_SLOT_COUNT, "Formatter places cannot grow.")
}

// Destination is the output view the import sort turns.
type Destination printer.Form

// Destination_Invariants states every bounded output view.
func Destination_Invariants(value Destination, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), printer.FORM_SIZE_MINIMUM, printer.FORM_SIZE_MAXIMUM).
		Ensure()
}

// Forms keeps the destination in named storage.
type Forms struct {
	// Value survives reads that temporarily move sort cursors.
	Value Destination
}

// Forms_Invariants composes the output-view domain.
func Forms_Invariants(value Forms, namespace aver.Namespace) {
	Destination_Invariants(value.Value, namespace)
}

// HEAD_SIZE_MAXIMUM is the widest line prefix one formatter searches for.
const HEAD_SIZE_MAXIMUM = len(PACKAGE_HEAD)

// Head is one line prefix a formatter searches for.
type Head string

// Head_Invariants admits absence and every bounded line prefix.
func Head_Invariants(value Head, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), PLACE_MINIMUM, HEAD_SIZE_MAXIMUM).
		Ensure()
}

// Heads keeps the active line prefix in named storage.
type Heads struct {
	// Value survives the scan that compares the bytes behind it.
	Value Head
}

// Heads_Invariants composes the line-prefix domain.
func Heads_Invariants(value Heads, namespace aver.Namespace) {
	Head_Invariants(value.Value, namespace)
}

// Place_Storage keeps caller-owned sort positions distinct from their view.
type Place_Storage struct {
	// Values makes ownership explicit at the package boundary.
	Values Places
}

// Place_Storage_Invariants composes sort-position storage.
func Place_Storage_Invariants(value Place_Storage, namespace aver.Namespace) {
	Places_Invariants(value.Values, namespace)
}

// NAME_COUNT_MAXIMUM is the widest table of names one formatter renames by. Each declared name
// holds one node of its own at least, thus one tree declares no more.
const NAME_COUNT_MAXIMUM = ast.NODE_COUNT_MAXIMUM

// NAME_SLOT_COUNT is the entry count one name storage holds: the table and the one candidate a
// declaration tests before it takes it.
const NAME_SLOT_COUNT = NAME_COUNT_MAXIMUM + 1

// NAME_TEXT_SIZE_MAXIMUM is the byte count of every name text one table holds, which is the text
// of the widest source.
const NAME_TEXT_SIZE_MAXIMUM = token.SOURCE_SIZE_MAXIMUM

// SLOT_COUNT is the slot count of the hash index over the table. It is a power of two past twice
// the widest table, thus a probe masks rather than divides and never walks a full index.
const SLOT_COUNT = 262144

// HASH_OFFSET is the FNV-1a offset basis.
const HASH_OFFSET = 2166136261

// HASH_PRIME is the FNV-1a prime.
const HASH_PRIME = 16777619

// CASE_SNAKE is the case of a name that opens lowercase.
const CASE_SNAKE Case = 1

// CASE_ADA is the case of a name that opens uppercase.
const CASE_ADA Case = 2

// CASE_CONSTANT is the case of a constant.
const CASE_CONSTANT Case = 3

// Case names one of the three cases the linter asks of a declared name.
type Case uint8

// Case_Invariants states every case.
func Case_Invariants(value Case, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(CASE_SNAKE), uint8(CASE_ADA), uint8(CASE_CONSTANT),
		).
		Ensure()
}

// Name is one declared name some file spells in the wrong case. The text stands in the table
// rather than in the source, because the source of one file is gone when the next file formats.
type Name struct {
	// Text is the span of the name text in the table text.
	Text Span
	// Case is the case the name takes.
	Case Case
}

// Name_Invariants composes one declared name.
func Name_Invariants(value Name, namespace aver.Namespace) {
	Span_Invariants(value.Text, namespace)
	Case_Invariants(value.Case, namespace)
}

// Span is the bytes of one name text in the table text.
type Span struct {
	// Start is the byte the text opens at.
	Start Place
	// Size is the byte count of the text.
	Size Text_Size
}

// Span_Invariants composes one text span.
func Span_Invariants(value Span, namespace aver.Namespace) {
	Place_Invariants(value.Start, namespace)
	Text_Size_Invariants(value.Size, namespace)
}

// Text_Size is the byte count of one name text.
type Text_Size int32

// Text_Size_Invariants states every name text size. A name holds one byte at least.
func Text_Size_Invariants(value Text_Size, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), TEXT_SIZE_MINIMUM, NAME_TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Names keeps the rename table in caller storage.
type Names []Name

// Names_Invariants fixes the rename table and prevents growth. No entry is proved here, because
// every step of a format would read the whole table.
func Names_Invariants(value Names, _ aver.Namespace) {
	aver.Always(len(value) == NAME_SLOT_COUNT, "Formatter names have complete length.")
	aver.Always(cap(value) == NAME_SLOT_COUNT, "Formatter names cannot grow.")
}

// Name_Storage keeps caller-owned names distinct from their view.
type Name_Storage struct {
	// Values makes ownership explicit at the package boundary.
	Values Names
}

// Name_Storage_Invariants composes name storage.
func Name_Storage_Invariants(value Name_Storage, namespace aver.Namespace) {
	Names_Invariants(value.Values, namespace)
}

// Text keeps every name text of the table in caller storage.
type Text []byte

// Text_Invariants fixes the table text and prevents growth.
func Text_Invariants(value Text, _ aver.Namespace) {
	aver.Always(len(value) == NAME_TEXT_SIZE_MAXIMUM, "Formatter text has complete length.")
	aver.Always(cap(value) == NAME_TEXT_SIZE_MAXIMUM, "Formatter text cannot grow.")
}

// Text_Storage keeps caller-owned name text distinct from its view.
type Text_Storage struct {
	// Values makes ownership explicit at the package boundary.
	Values Text
}

// Text_Storage_Invariants composes text storage.
func Text_Storage_Invariants(value Text_Storage, namespace aver.Namespace) {
	Text_Invariants(value.Values, namespace)
}

// SLOT_EMPTY is the slot that names no entry.
const SLOT_EMPTY = 0

// TEXT_SIZE_MINIMUM is the byte count of the shortest name, which holds one letter.
const TEXT_SIZE_MINIMUM = 1

// Slot is one slot of the hash index: zero when empty, else one past the entry it names, thus a
// zeroed index is an empty one.
type Slot int32

// Slot_Invariants states every slot.
func Slot_Invariants(value Slot, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), SLOT_EMPTY, NAME_COUNT_MAXIMUM).
		Ensure()
}

// Slots keeps the hash index in caller storage.
type Slots []Slot

// Slots_Invariants fixes the hash index and prevents growth.
func Slots_Invariants(value Slots, _ aver.Namespace) {
	aver.Always(len(value) == SLOT_COUNT, "Formatter slots have complete length.")
	aver.Always(cap(value) == SLOT_COUNT, "Formatter slots cannot grow.")
}

// Slot_Storage keeps caller-owned slots distinct from their view.
type Slot_Storage struct {
	// Values makes ownership explicit at the package boundary.
	Values Slots
}

// Slot_Storage_Invariants composes slot storage.
func Slot_Storage_Invariants(value Slot_Storage, namespace aver.Namespace) {
	Slots_Invariants(value.Values, namespace)
}

// Sources keeps the source one declaration reads names from in named storage.
type Sources struct {
	// Value survives every step of a declaration, each of which reads a name from it.
	Value token.Source
}

// Sources_Invariants composes the source domain.
func Sources_Invariants(value Sources, namespace aver.Namespace) {
	token.Source_Invariants(value.Value, namespace)
}

// Probes keeps the text one search of the table looks for in named storage.
type Probes struct {
	// Value survives the probe, which reads it at every slot it tests.
	Value Destination
}

// Probes_Invariants composes the probe domain.
func Probes_Invariants(value Probes, namespace aver.Namespace) {
	Destination_Invariants(value.Value, namespace)
}

// Formatter_Fields states the concrete storage layout independently of its validated view.
type Formatter_Fields struct {
	// Places holds the bytes one sort of imports reads and writes.
	Places Place_Storage
	// Names holds the declared names one rename spells anew.
	Names Name_Storage
	// Texts holds the text of every declared name.
	Texts Text_Storage
	// Slots holds the hash index over the declared names.
	Slots Slot_Storage
	// Forms holds the storage the sort reads the form from.
	Forms Forms
	// Heads holds the text one read of a line asks the line to open with.
	Heads Heads
	// Sources holds the source one declaration reads names from.
	Sources Sources
	// Probes holds the text one search of the table looks for.
	Probes Probes
}

// Formatter_Fields_Invariants composes every concrete field once.
func Formatter_Fields_Invariants(value Formatter_Fields, namespace aver.Namespace) {
	Place_Storage_Invariants(value.Places, namespace)
	Name_Storage_Invariants(value.Names, namespace)
	Text_Storage_Invariants(value.Texts, namespace)
	Slot_Storage_Invariants(value.Slots, namespace)
	Forms_Invariants(value.Forms, namespace)
	Heads_Invariants(value.Heads, namespace)
	Sources_Invariants(value.Sources, namespace)
	Probes_Invariants(value.Probes, namespace)
}

// Forms_Stored prevents one sort step from proving every destination length.
type Forms_Stored interface{}

// Forms_Stored_Invariants fixes destination representation.
func Forms_Stored_Invariants(value Forms_Stored, _ aver.Namespace) {
	_, valid := value.(Forms)
	aver.Always(valid == (value != nil), "Formatter forms have expected storage type.")
}

// Heads_Stored prevents one sort step from proving every line-prefix length.
type Heads_Stored interface{}

// Heads_Stored_Invariants fixes line-prefix representation.
func Heads_Stored_Invariants(value Heads_Stored, _ aver.Namespace) {
	_, valid := value.(Heads)
	aver.Always(valid == (value != nil), "Formatter heads have expected storage type.")
}

// Sources_Stored prevents one declaration step from proving every source length.
type Sources_Stored interface{}

// Sources_Stored_Invariants fixes source representation.
func Sources_Stored_Invariants(value Sources_Stored, _ aver.Namespace) {
	_, valid := value.(Sources)
	aver.Always(valid == (value != nil), "Formatter sources have expected storage type.")
}

// Probes_Stored prevents one search step from proving every probe length.
type Probes_Stored interface{}

// Probes_Stored_Invariants fixes probe representation.
func Probes_Stored_Invariants(value Probes_Stored, _ aver.Namespace) {
	_, valid := value.(Probes)
	aver.Always(valid == (value != nil), "Formatter probes have expected storage type.")
}

// Formatter is one validated view over caller-owned format storage.
type Formatter Formatter_Fields

// Formatter_Invariants fixes sort storage without proving stale mutable views.
func Formatter_Invariants(value Formatter, namespace aver.Namespace) {
	Place_Storage_Invariants(value.Places, namespace)
	Name_Storage_Invariants(value.Names, namespace)
	Text_Storage_Invariants(value.Texts, namespace)
	Slot_Storage_Invariants(value.Slots, namespace)
	Forms_Stored_Invariants(Forms_Stored(value.Forms), namespace)
	Heads_Stored_Invariants(Heads_Stored(value.Heads), namespace)
	Sources_Stored_Invariants(Sources_Stored(value.Sources), namespace)
	Probes_Stored_Invariants(Probes_Stored(value.Probes), namespace)
}

// Formatter_Handle gives caller-owned format state one identity.
type Formatter_Handle *Formatter

// Formatter_Handle_Invariants composes present format storage.
func Formatter_Handle_Invariants(value Formatter_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Formatter_Invariants(*value, namespace)
}

// Format writes the canonical form of one parsed tree into caller storage and reports whether the
// whole form fit. It renames by the table Declare fills and never by the tree alone, because a
// name the tree declares is spelled in files the format never sees. A tree the parser refused
// writes what it holds, because a caller reads more from a partial form than from nothing.
func Format(
	subject Formatter_Handle, writer printer.Printer_Handle, destination printer.Form,
	tree ast.Parse_State_Handle, source token.Source,
) (result printer.Print_Result) {
	defer func() { printer.Print_Result_Invariants(result, "format.result") }()
	Formatter_Handle_Invariants(subject, "format.subject")
	printer.Printer_Handle_Invariants(writer, "format.writer")
	printer.Form_Invariants(destination, "format.destination")
	ast.Parse_State_Handle_Invariants(tree, "format.tree")
	token.Source_Invariants(source, "format.source")
	result = printer.Print(writer, destination, tree, source)
	subject.Forms.Value = Destination(destination)
	subject.Places.Values[PLACE_LIMIT] = Place(result.Count)
	sort_imports(subject)
	// A table that holds no name reads no byte of the form again, thus the rename costs a
	// program that declares every name in its case nothing.
	if subject.Places.Values[PLACE_NAMES] == 0 {
		return result
	}
	fit := rename(subject)
	result.Count = printer.Form_Count(subject.Places.Values[PLACE_LIMIT])
	if !bool(fit) {
		result.OK = false
	}
	return result
}

// Declare takes each name one file declares in the wrong case into the table every later format
// renames by, and reports whether the table held room for all of them. A name one file declares
// is spelled in other files too, thus a caller declares every file of a program before it formats
// any, and the rename reaches every file that spells the name.
func Declare(
	subject Formatter_Handle, tree ast.Parse_State_Handle, source token.Source,
) (fit printer.Boolean) {
	defer func() { printer.Boolean_Invariants(fit, "declare.fit") }()
	Formatter_Handle_Invariants(subject, "declare.subject")
	ast.Parse_State_Handle_Invariants(tree, "declare.tree")
	token.Source_Invariants(source, "declare.source")
	subject.Sources.Value = source
	return take_names(subject, tree)
}

// Forget empties the table, thus a caller that formats another program renames by its names
// alone.
func Forget(subject Formatter_Handle) {
	Formatter_Handle_Invariants(subject, "forget.subject")
	subject.Places.Values[PLACE_NAMES] = 0
	subject.Places.Values[PLACE_TEXT] = 0
	for slot := range subject.Slots.Values {
		subject.Slots.Values[slot] = 0
	}
}

// Fit distinguishes complete canonical forms from bounded prefixes.
type Fit printer.Boolean

// Fit_Invariants states both storage outcomes.
func Fit_Invariants(value Fit, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Canonical form fits caller storage.").
		Ensure()
}

// Clean_Result keeps canonical equality and storage outcome at one output boundary.
type Clean_Result struct {
	// Clean remains false when form differs or cannot fit.
	Clean printer.Boolean
	// OK distinguishes differing complete forms from incomplete prefixes.
	OK Fit
}

// Clean_Result_Invariants composes one cleanliness outcome.
func Clean_Result_Invariants(value Clean_Result, namespace aver.Namespace) {
	printer.Boolean_Invariants(value.Clean, namespace)
	Fit_Invariants(value.OK, namespace)
}

// Clean reports whether one source already stands in the canonical form, and whether the form it
// compared against fit the storage. A form that ran past the storage answers no, because the form
// the source would take was never written.
func Clean(
	subject Formatter_Handle, writer printer.Printer_Handle, destination printer.Form,
	tree ast.Parse_State_Handle, source token.Source,
) (result Clean_Result) {
	defer func() { Clean_Result_Invariants(result, "clean.result") }()
	Formatter_Handle_Invariants(subject, "clean.subject")
	printer.Printer_Handle_Invariants(writer, "clean.writer")
	printer.Form_Invariants(destination, "clean.destination")
	ast.Parse_State_Handle_Invariants(tree, "clean.tree")
	token.Source_Invariants(source, "clean.source")
	formatted := Format(subject, writer, destination, tree, source)
	if !bool(formatted.OK) {
		return Clean_Result{Clean: false, OK: false}
	}
	if int(formatted.Count) != len(source) {
		return Clean_Result{Clean: false, OK: true}
	}
	for index := range int(formatted.Count) {
		if destination[index] != source[index] {
			return Clean_Result{Clean: false, OK: true}
		}
	}
	return Clean_Result{Clean: true, OK: true}
}

// Sorts each run of imports the form holds by the path each one names. The tree states the order
// the author wrote and the canonical form states the order a reader searches, thus the sort
// stands here and never in the printer, which answers for the tree alone.
func sort_imports(subject Formatter_Handle) {
	Formatter_Handle_Invariants(subject, "sort_imports.subject")
	subject.Places.Values[PLACE_SCAN] = PLACE_MINIMUM
	for subject.Places.Values[PLACE_SCAN] < subject.Places.Values[PLACE_LIMIT] {
		if !bool(take_record(subject)) {
			subject.Places.Values[PLACE_CURSOR] = subject.Places.Values[PLACE_SCAN]
			// Every import of one file stands ahead of every other declaration,
			// thus the scan closes at the first line that opens something else and
			// reads no further than the imports it sorts.
			if bool(closes_imports(subject)) {
				return
			}
			close_place(subject)
			subject.Places.Values[PLACE_SCAN] = subject.Places.Values[PLACE_CURSOR]
			continue
		}
		take_run(subject)
		sort_run(subject)
	}
}

// Moves one read of the form past the line it stands on. A line the form never closed runs to the
// end of the form, because a refused print writes the bytes it reached and no line feed behind
// them.
func close_place(subject Formatter_Handle) {
	Formatter_Handle_Invariants(subject, "close_place.subject")
	form := subject.Forms.Value
	limit := subject.Places.Values[PLACE_LIMIT]
	for subject.Places.Values[PLACE_CURSOR] < limit {
		held := subject.Places.Values[PLACE_CURSOR]
		subject.Places.Values[PLACE_CURSOR] = held + 1
		if form[held] == '\n' {
			return
		}
	}
}

// Reports whether the line one read stands on closes the imports of the file. The package clause,
// an empty line, and a note each stand where an import may stand behind them; every other line
// states a declaration no import stands behind.
func closes_imports(subject Formatter_Handle) (yes printer.Boolean) {
	defer func() { printer.Boolean_Invariants(yes, "closes_imports.yes") }()
	Formatter_Handle_Invariants(subject, "closes_imports.subject")
	subject.Heads.Value = PACKAGE_HEAD
	if bool(states_head(subject)) {
		return false
	}
	if bool(opens_note(subject)) {
		return false
	}
	subject.Heads.Value = LINE_HEAD
	return !states_head(subject)
}

// PACKAGE_HEAD is the text one package clause opens with.
const PACKAGE_HEAD = "package "

// LINE_HEAD is the text one empty line states.
const LINE_HEAD = "\n"

// Reports whether the line one read stands on opens one import declaration.
func opens_import(subject Formatter_Handle) (yes printer.Boolean) {
	defer func() { printer.Boolean_Invariants(yes, "opens_import.yes") }()
	Formatter_Handle_Invariants(subject, "opens_import.subject")
	subject.Heads.Value = IMPORT_HEAD
	return states_head(subject)
}

// Reports whether the line one read stands on opens one note.
func opens_note(subject Formatter_Handle) (yes printer.Boolean) {
	defer func() { printer.Boolean_Invariants(yes, "opens_note.yes") }()
	Formatter_Handle_Invariants(subject, "opens_note.subject")
	subject.Heads.Value = NOTE_HEAD
	return states_head(subject)
}

// Reports whether the line one read stands on opens with the text stated.
func states_head(subject Formatter_Handle) (yes printer.Boolean) {
	defer func() { printer.Boolean_Invariants(yes, "states_head.yes") }()
	Formatter_Handle_Invariants(subject, "states_head.subject")
	form := subject.Forms.Value
	head := subject.Heads.Value
	place := int(subject.Places.Values[PLACE_CURSOR])
	if place+len(head) > int(subject.Places.Values[PLACE_LIMIT]) {
		return false
	}
	for index := range len(head) {
		if form[place+index] != head[index] {
			return false
		}
	}
	return true
}

// Reads the record one scan stands on: the notes the author wrote above one import and the import
// itself. A note that opens no import states no record, thus a note standing alone closes the run
// rather than joining it.
func take_record(subject Formatter_Handle) (yes printer.Boolean) {
	defer func() { printer.Boolean_Invariants(yes, "take_record.yes") }()
	Formatter_Handle_Invariants(subject, "take_record.subject")
	subject.Places.Values[PLACE_CURSOR] = subject.Places.Values[PLACE_SCAN]
	for subject.Places.Values[PLACE_CURSOR] < subject.Places.Values[PLACE_LIMIT] {
		if bool(opens_import(subject)) {
			close_place(subject)
			subject.Places.Values[PLACE_SCAN_END] = subject.Places.Values[PLACE_CURSOR]
			return true
		}
		if !bool(opens_note(subject)) {
			return false
		}
		close_place(subject)
	}
	return false
}

// Reads the run of records the scan stands at the opening of. An empty line and a declaration of
// any other kind each state a line that opens no record, thus each one closes the run and the
// groups the author wrote stand where the author put them.
func take_run(subject Formatter_Handle) {
	Formatter_Handle_Invariants(subject, "take_run.subject")
	subject.Places.Values[PLACE_HEAD] = subject.Places.Values[PLACE_SCAN]
	for bool(take_record(subject)) {
		subject.Places.Values[PLACE_SCAN] = subject.Places.Values[PLACE_SCAN_END]
	}
	subject.Places.Values[PLACE_TAIL] = subject.Places.Values[PLACE_SCAN]
}

// Sorts one run of records by the path each one names. The sort carries the smallest record of
// what remains to the opening of what remains, thus every record stands where its path states and
// no record leaves the run it stood in.
func sort_run(subject Formatter_Handle) {
	Formatter_Handle_Invariants(subject, "sort_run.subject")
	place := subject.Places.Values[PLACE_HEAD]
	for place < subject.Places.Values[PLACE_TAIL] {
		subject.Places.Values[PLACE_SCAN] = place
		take_record(subject)
		subject.Places.Values[PLACE_BEST] = place
		subject.Places.Values[PLACE_BEST_END] = subject.Places.Values[PLACE_SCAN_END]
		take_smallest(subject)
		subject.Places.Values[PLACE_FROM] = place
		subject.Places.Values[PLACE_MIDDLE] = subject.Places.Values[PLACE_BEST]
		subject.Places.Values[PLACE_TO] = subject.Places.Values[PLACE_BEST_END]
		turn(subject)
		size := subject.Places.Values[PLACE_BEST_END] - subject.Places.Values[PLACE_BEST]
		place = place + size
	}
	subject.Places.Values[PLACE_SCAN] = subject.Places.Values[PLACE_TAIL]
}

// Reads the record of the run that names the smallest path, from the record behind the one the
// sort stands on to the end of the run.
func take_smallest(subject Formatter_Handle) {
	Formatter_Handle_Invariants(subject, "take_smallest.subject")
	best := subject.Places.Values[PLACE_BEST]
	best_end := subject.Places.Values[PLACE_BEST_END]
	subject.Places.Values[PLACE_SCAN] = best_end
	for subject.Places.Values[PLACE_SCAN] < subject.Places.Values[PLACE_TAIL] {
		if !bool(take_record(subject)) {
			break
		}
		subject.Places.Values[PLACE_BEST] = best
		subject.Places.Values[PLACE_BEST_END] = best_end
		if bool(names_ahead(subject)) {
			best = subject.Places.Values[PLACE_SCAN]
			best_end = subject.Places.Values[PLACE_SCAN_END]
		}
		subject.Places.Values[PLACE_SCAN] = subject.Places.Values[PLACE_SCAN_END]
	}
	subject.Places.Values[PLACE_BEST] = best
	subject.Places.Values[PLACE_BEST_END] = best_end
}

// Reports whether the record one scan stands on names a path ahead of the path the smallest
// record names. A record states its path on the line it closes with, which is the import itself.
func names_ahead(subject Formatter_Handle) (yes printer.Boolean) {
	defer func() { printer.Boolean_Invariants(yes, "names_ahead.yes") }()
	Formatter_Handle_Invariants(subject, "names_ahead.subject")
	form := subject.Forms.Value
	subject.Places.Values[PLACE_CURSOR] = subject.Places.Values[PLACE_SCAN_END]
	take_path(subject)
	scan := subject.Places.Values[PLACE_CURSOR]
	subject.Places.Values[PLACE_CURSOR] = subject.Places.Values[PLACE_BEST_END]
	take_path(subject)
	best := subject.Places.Values[PLACE_CURSOR]
	for scan < subject.Places.Values[PLACE_SCAN_END] {
		if best >= subject.Places.Values[PLACE_BEST_END] {
			return false
		}
		if form[scan] != form[best] {
			yes = printer.Boolean(form[scan] < form[best])
			return yes
		}
		scan = scan + 1
		best = best + 1
	}
	yes = printer.Boolean(best < subject.Places.Values[PLACE_BEST_END])
	return yes
}

// Moves one read from the byte a record closes at to the byte its path opens at, which stands
// behind the quotation mark the import line holds. A line that states no quotation mark opens its
// path at its own opening, so that a form the print refused compares by the bytes it holds.
func take_path(subject Formatter_Handle) {
	Formatter_Handle_Invariants(subject, "take_path.subject")
	form := subject.Forms.Value
	end := subject.Places.Values[PLACE_CURSOR]
	open_line(subject)
	head := subject.Places.Values[PLACE_CURSOR]
	for subject.Places.Values[PLACE_CURSOR] < end {
		if form[subject.Places.Values[PLACE_CURSOR]] == '"' {
			return
		}
		subject.Places.Values[PLACE_CURSOR] = subject.Places.Values[PLACE_CURSOR] + 1
	}
	subject.Places.Values[PLACE_CURSOR] = head
}

// Moves one read from the byte a line closes at to the byte that line opens at.
func open_line(subject Formatter_Handle) {
	Formatter_Handle_Invariants(subject, "open_line.subject")
	form := subject.Forms.Value
	if subject.Places.Values[PLACE_CURSOR] > PLACE_MINIMUM {
		subject.Places.Values[PLACE_CURSOR] = subject.Places.Values[PLACE_CURSOR] - 1
	}
	for subject.Places.Values[PLACE_CURSOR] > PLACE_MINIMUM {
		if form[subject.Places.Values[PLACE_CURSOR]-1] == '\n' {
			return
		}
		subject.Places.Values[PLACE_CURSOR] = subject.Places.Values[PLACE_CURSOR] - 1
	}
}

// Turns the bytes of one span so the record that opened in the middle of it opens the span. The
// turn stands on three reversals, thus the form holds every byte it held and no storage stands
// behind the sort.
func turn(subject Formatter_Handle) {
	Formatter_Handle_Invariants(subject, "turn.subject")
	from := subject.Places.Values[PLACE_FROM]
	middle := subject.Places.Values[PLACE_MIDDLE]
	to := subject.Places.Values[PLACE_TO]
	subject.Places.Values[PLACE_TO] = middle
	reverse(subject)
	subject.Places.Values[PLACE_FROM] = middle
	subject.Places.Values[PLACE_TO] = to
	reverse(subject)
	subject.Places.Values[PLACE_FROM] = from
	reverse(subject)
}

// Reverses the bytes of one span of the form.
func reverse(subject Formatter_Handle) {
	Formatter_Handle_Invariants(subject, "reverse.subject")
	form := subject.Forms.Value
	head := subject.Places.Values[PLACE_FROM]
	tail := subject.Places.Values[PLACE_TO]
	for head < tail-1 {
		held := form[head]
		form[head] = form[tail-1]
		form[tail-1] = held
		head = head + 1
		tail = tail - 1
	}
}

// LETTER_COUNT is the count of letters one case of the alphabet holds. A byte below the first
// letter wraps past it in a subtraction, thus one comparison tests both ends of the alphabet.
const LETTER_COUNT = 26

// DIGIT_COUNT is the count of decimal digits.
const DIGIT_COUNT = 10

// LETTER_CASE_BIT is the bit that tells a lowercase letter from its uppercase letter: setting it
// lowers a letter, and clearing it raises one.
const LETTER_CASE_BIT = 0x20

// TEST_MAIN is the function name the testing package reserves.
const TEST_MAIN = "TestMain"

// Reads every declared name the tree holds into the rename table, and reports whether the table
// held room for each. A declaration node states where its first name stands, and the list behind
// that name states the rest.
func take_names(subject Formatter_Handle, tree ast.Parse_State_Handle) (fit printer.Boolean) {
	defer func() { printer.Boolean_Invariants(fit, "take_names.fit") }()
	Formatter_Handle_Invariants(subject, "take_names.subject")
	ast.Parse_State_Handle_Invariants(tree, "take_names.tree")
	subject.Places.Values[PLACE_DROPPED] = 0
	for slot := ast.INDEX_FIRST; slot < ast.Index(tree.Node_Cursors.Count); slot++ {
		node := ast.Node_At(tree, slot)
		place := node.Token
		want := CASE_SNAKE
		switch node.Kind {
		case ast.NODE_CONSTANT_NAME:
			want = CASE_CONSTANT
		case ast.NODE_FIELD_NAME, ast.NODE_PARAMETER_NAME, ast.NODE_DEFINE,
			ast.NODE_TYPE_PARAMETER:
		// A var or a type node stands on its keyword, and its name stands one token behind.
		case ast.NODE_VARIABLE, ast.NODE_TYPE:
			place = node.Token + 1
		// A function states its name in its first identifier child, because its receiver
		// stands in a node of its own.
		case ast.NODE_FUNCTION:
			child := ast.Index(node.First_Child)
			for child != ast.INDEX_ABSENT {
				if ast.Node_At(tree, child).Kind == ast.NODE_IDENTIFIER {
					break
				}
				child = ast.Index(ast.Node_At(tree, child).Next)
			}
			if child == ast.INDEX_ABSENT {
				continue
			}
			place = ast.Node_At(tree, child).Token
			one := ast.Token_At(tree, place)
			if string(token.Text(subject.Sources.Value, one)) == TEST_MAIN {
				continue
			}
		default:
			continue
		}
		subject.Places.Values[PLACE_TOKEN] = Place(place)
		subject.Names.Values[subject.Places.Values[PLACE_NAMES]].Case = want
		take_name_list(subject, tree)
	}
	return printer.Boolean(subject.Places.Values[PLACE_DROPPED] == 0)
}

// Reads the list of names the candidate opens, joined by commas, which is how a var, a define, a
// field, a parameter, and a type parameter each declare more than one.
func take_name_list(subject Formatter_Handle, tree ast.Parse_State_Handle) {
	Formatter_Handle_Invariants(subject, "take_name_list.subject")
	ast.Parse_State_Handle_Invariants(tree, "take_name_list.tree")
	place := ast.Token_Index(subject.Places.Values[PLACE_TOKEN])
	want := subject.Names.Values[subject.Places.Values[PLACE_NAMES]].Case
	token_count := ast.Token_Index(tree.Token_Cursors.Count)
	for place < token_count {
		subject.Places.Values[PLACE_TOKEN] = Place(place)
		subject.Names.Values[subject.Places.Values[PLACE_NAMES]].Case = want
		take_name(subject, tree)
		comma := place + 1
		if comma == token_count {
			return
		}
		if ast.Token_At(tree, comma).Kind != token.KIND_COMMA {
			return
		}
		place = comma + 1
	}
}

// Takes the candidate name, whose case stands in the slot just past the table, into the table. A
// name that is no identifier, opens with no letter, stands in its case already, or stands in the
// table already takes no entry. A name that is no constant takes its case from its first letter.
func take_name(subject Formatter_Handle, tree ast.Parse_State_Handle) {
	Formatter_Handle_Invariants(subject, "take_name.subject")
	ast.Parse_State_Handle_Invariants(tree, "take_name.tree")
	count := subject.Places.Values[PLACE_NAMES]
	one := ast.Token_At(tree, ast.Token_Index(subject.Places.Values[PLACE_TOKEN]))
	if one.Kind != token.KIND_IDENTIFIER {
		return
	}
	text := token.Text(subject.Sources.Value, one)
	want := subject.Names.Values[count].Case
	if want != CASE_CONSTANT {
		if text[0]-'a' < LETTER_COUNT {
			want = CASE_SNAKE
		} else if text[0]-'A' < LETTER_COUNT {
			want = CASE_ADA
		} else {
			return
		}
	}
	subject.Names.Values[count].Case = want
	subject.Probes.Value = Destination(text)
	if bool(in_case(subject)) {
		return
	}
	find_slot(subject)
	if subject.Places.Values[PLACE_FOUND] != count {
		return
	}
	start := subject.Places.Values[PLACE_TEXT]
	// A full table drops the name and counts it, because a name the table never holds is a name
	// no format renames, and the caller must learn the form it reads is no canonical form.
	if int(count) == NAME_COUNT_MAXIMUM {
		subject.Places.Values[PLACE_DROPPED] = subject.Places.Values[PLACE_DROPPED] + 1
		return
	}
	if int(start)+len(text) > NAME_TEXT_SIZE_MAXIMUM {
		subject.Places.Values[PLACE_DROPPED] = subject.Places.Values[PLACE_DROPPED] + 1
		return
	}
	copy(subject.Texts.Values[start:], text)
	subject.Names.Values[count] = Name{
		Text: Span{Start: start, Size: Text_Size(len(text))}, Case: want,
	}
	subject.Slots.Values[subject.Places.Values[PLACE_SLOT]] = Slot(count + 1)
	subject.Places.Values[PLACE_NAMES] = count + 1
	subject.Places.Values[PLACE_TEXT] = start + Place(len(text))
}

// Reads the entry of the table whose text the probe holds into the found place, or the count of
// names when none holds it, and the slot the probe stops at into the slot place. The index holds
// at most half its slots, thus a probe meets an empty slot before it walks the whole index.
func find_slot(subject Formatter_Handle) {
	Formatter_Handle_Invariants(subject, "find_slot.subject")
	probe := subject.Probes.Value
	hash := uint32(HASH_OFFSET)
	for _, value := range probe {
		hash = (hash ^ uint32(value)) * HASH_PRIME
	}
	slot := hash & (SLOT_COUNT - 1)
	for range SLOT_COUNT {
		held := subject.Slots.Values[slot]
		if held == SLOT_EMPTY {
			break
		}
		entry := subject.Names.Values[held-1]
		end := entry.Text.Start + Place(entry.Text.Size)
		text := subject.Texts.Values[entry.Text.Start:end]
		if string(text) == string(probe) {
			subject.Places.Values[PLACE_FOUND] = Place(held - 1)
			return
		}
		slot = (slot + 1) & (SLOT_COUNT - 1)
	}
	aver.Always(subject.Slots.Values[slot] == SLOT_EMPTY, "A probe meets an empty slot.")
	subject.Places.Values[PLACE_FOUND] = subject.Places.Values[PLACE_NAMES]
	subject.Places.Values[PLACE_SLOT] = Place(slot)
}

// Reports whether the probe stands in the case of the candidate, which is the pattern the linter
// matches. A doubled or closing underscore states an empty segment, which no case admits.
func in_case(subject Formatter_Handle) (yes printer.Boolean) {
	defer func() { printer.Boolean_Invariants(yes, "in_case.yes") }()
	Formatter_Handle_Invariants(subject, "in_case.subject")
	text := subject.Probes.Value
	start := 0
	for index := range len(text) + 1 {
		if index < len(text) {
			if text[index] != '_' {
				continue
			}
		}
		if index == start {
			return false
		}
		subject.Places.Values[PLACE_START] = Place(start)
		subject.Places.Values[PLACE_END] = Place(index)
		if !bool(in_segment(subject)) {
			return false
		}
		start = index + 1
	}
	return true
}

// Reports whether one segment of the candidate name between underscores stands in its case. An
// Ada_Case segment is a capitalized word, an acronym, an acronym plural, or a number that follows
// the first segment, because a capacity names its own segment and Go opens a name on a letter.
func in_segment(subject Formatter_Handle) (yes printer.Boolean) {
	defer func() { printer.Boolean_Invariants(yes, "in_segment.yes") }()
	Formatter_Handle_Invariants(subject, "in_segment.subject")
	entry := subject.Names.Values[subject.Places.Values[PLACE_NAMES]]
	text := subject.Probes.Value
	start := int(subject.Places.Values[PLACE_START])
	segment := text[start:subject.Places.Values[PLACE_END]]
	lower_count := 0
	upper_count := 0
	for _, value := range segment[1:] {
		if value-'a' < LETTER_COUNT {
			lower_count = lower_count + 1
		} else if value-'A' < LETTER_COUNT {
			upper_count = upper_count + 1
		}
	}
	head := segment[0]
	if head-'0' < DIGIT_COUNT {
		if start == 0 {
			return false
		}
		switch entry.Case {
		case CASE_SNAKE:
			return printer.Boolean(upper_count == 0)
		case CASE_CONSTANT:
			return printer.Boolean(lower_count == 0)
		}
		return printer.Boolean(lower_count+upper_count == 0)
	}
	switch entry.Case {
	case CASE_SNAKE:
		if head-'a' > LETTER_COUNT-1 {
			return false
		}
		return printer.Boolean(upper_count == 0)
	case CASE_CONSTANT:
		if head-'A' > LETTER_COUNT-1 {
			return false
		}
		return printer.Boolean(lower_count == 0)
	}
	if head-'A' > LETTER_COUNT-1 {
		return false
	}
	if upper_count == 0 {
		return true
	}
	if lower_count == 0 {
		return true
	}
	// An acronym takes a plural, as in IDs, because Go names a run of them that way.
	if lower_count == 1 {
		return printer.Boolean(segment[len(segment)-1] == 's')
	}
	return false
}

// Spells each identifier of the form that names an entry of the rename table in the case the
// entry takes, and reports whether the form still fits the storage. A spelling that grows the form
// past the storage drops the bytes that no longer fit, as a print past the storage does.
func rename(subject Formatter_Handle) (fit printer.Boolean) {
	defer func() { printer.Boolean_Invariants(fit, "rename.fit") }()
	Formatter_Handle_Invariants(subject, "rename.subject")
	form := subject.Forms.Value
	fit = true
	subject.Places.Values[PLACE_CURSOR] = PLACE_MINIMUM
	for subject.Places.Values[PLACE_CURSOR] < subject.Places.Values[PLACE_LIMIT] {
		take_word(subject)
		place := subject.Places.Values[PLACE_WORD]
		end := subject.Places.Values[PLACE_WORD_END]
		if end == place {
			skip_inert(subject)
			continue
		}
		subject.Places.Values[PLACE_CURSOR] = end
		subject.Probes.Value = form[place:end]
		find_slot(subject)
		if subject.Places.Values[PLACE_FOUND] == subject.Places.Values[PLACE_NAMES] {
			continue
		}
		// A spell at the end of the storage writes no byte, thus the first spell measures
		// the name and the second writes it behind the bytes the measure made room for.
		subject.Places.Values[PLACE_WORD] = Place(len(form))
		spell(subject)
		size := Place(subject.Places.Values[PLACE_SIZE])
		limit := subject.Places.Values[PLACE_LIMIT]
		if int(place+size) < len(form) {
			copy(form[place+size:], form[end:limit])
		}
		subject.Places.Values[PLACE_WORD] = place
		spell(subject)
		limit = limit + size - (end - place)
		if int(limit) > len(form) {
			fit = false
			limit = Place(len(form))
		}
		subject.Places.Values[PLACE_LIMIT] = limit
		subject.Places.Values[PLACE_CURSOR] = place + size
	}
	return fit
}

// Reads the identifier the cursor stands on. A cursor on a byte that opens no identifier reads an
// empty one, which the caller tells by the end standing on the opening.
func take_word(subject Formatter_Handle) {
	Formatter_Handle_Invariants(subject, "take_word.subject")
	form := subject.Forms.Value
	cursor := subject.Places.Values[PLACE_CURSOR]
	limit := subject.Places.Values[PLACE_LIMIT]
	subject.Places.Values[PLACE_WORD] = cursor
	subject.Places.Values[PLACE_WORD_END] = cursor
	opens := form[cursor] == '_'
	if (form[cursor]|LETTER_CASE_BIT)-'a' < LETTER_COUNT {
		opens = true
	}
	if !opens {
		return
	}
	end := cursor + 1
	for end < limit {
		value := form[end]
		word := value == '_'
		if (value|LETTER_CASE_BIT)-'a' < LETTER_COUNT {
			word = true
		}
		if value-'0' < DIGIT_COUNT {
			word = true
		}
		if !word {
			break
		}
		end = end + 1
	}
	subject.Places.Values[PLACE_WORD_END] = end
}

// Moves the cursor past the unit of the form that names nothing: a note, a string, a character, a
// number, or one byte of any other kind. A spelling inside one of them names nothing, and a unit
// the form never closed runs to the limit, because a refused print writes no closing mark.
func skip_inert(subject Formatter_Handle) {
	Formatter_Handle_Invariants(subject, "skip_inert.subject")
	form := subject.Forms.Value[:subject.Places.Values[PLACE_LIMIT]]
	place := int(subject.Places.Values[PLACE_CURSOR])
	value := form[place]
	end := place + 1
	subject.Places.Values[PLACE_CURSOR] = Place(end)
	mark := value
	switch value {
	case '/':
		if end == len(form) {
			return
		}
		mark = form[end]
		if mark == '/' {
			mark = '\n'
		}
		if mark != '\n' {
			if mark != '*' {
				return
			}
		}
		end = end + 1
	case '"', '\'', '`':
	default:
		skip_number(subject)
		return
	}
	for end < len(form) {
		held := form[end]
		end = end + 1
		// A string and a character escape the byte behind a backslash, and a note and a raw
		// string hold a backslash as written.
		if held == '\\' {
			if value != '`' {
				if value != '/' {
					end = end + 1
					continue
				}
			}
		}
		if held != mark {
			continue
		}
		if mark != '*' {
			break
		}
		if end < len(form) {
			if form[end] == '/' {
				end = end + 1
				break
			}
		}
	}
	subject.Places.Values[PLACE_CURSOR] = Place(min(end, len(form)))
}

// Moves the cursor past the number it stands just behind the opening of. A number opens on a
// digit, or on a point a digit follows, and runs over every letter, digit, underscore, and point
// behind it, thus no identifier opens inside it.
func skip_number(subject Formatter_Handle) {
	Formatter_Handle_Invariants(subject, "skip_number.subject")
	form := subject.Forms.Value[:subject.Places.Values[PLACE_LIMIT]]
	end := int(subject.Places.Values[PLACE_CURSOR])
	value := form[end-1]
	if value-'0' > DIGIT_COUNT-1 {
		if value != '.' {
			return
		}
		if end == len(form) {
			return
		}
		if form[end]-'0' > DIGIT_COUNT-1 {
			return
		}
	}
	for end < len(form) {
		held := form[end]
		unit := held == '_'
		if (held|LETTER_CASE_BIT)-'a' < LETTER_COUNT {
			unit = true
		}
		if held-'0' < DIGIT_COUNT {
			unit = true
		}
		if held == '.' {
			unit = true
		}
		if !unit {
			break
		}
		end = end + 1
	}
	subject.Places.Values[PLACE_CURSOR] = Place(end)
}

// Writes the found name in the case its entry takes at the word place, and reads the size of the
// spelling into the size place. A byte the storage holds no room for takes no write. An Ada_Case
// word that holds no lowercase letter stands as written, because an acronym keeps its capitals.
func spell(subject Formatter_Handle) {
	Formatter_Handle_Invariants(subject, "spell.subject")
	form := subject.Forms.Value
	place := int(subject.Places.Values[PLACE_WORD])
	entry := subject.Names.Values[subject.Places.Values[PLACE_FOUND]]
	text := subject.Texts.Values[entry.Text.Start : entry.Text.Start+Place(entry.Text.Size)]
	subject.Probes.Value = Destination(text)
	size := 0
	for start := 0; start < len(text); start = int(subject.Places.Values[PLACE_END]) {
		subject.Places.Values[PLACE_START] = Place(start)
		word_end(subject)
		end := int(subject.Places.Values[PLACE_END])
		if end == start {
			subject.Places.Values[PLACE_END] = Place(start + 1)
			continue
		}
		if size > 0 {
			if place+size < len(form) {
				form[place+size] = '_'
			}
			size = size + 1
		}
		acronym := true
		for _, value := range text[start:end] {
			if value-'a' < LETTER_COUNT {
				acronym = false
			}
		}
		for index := start; index < end; index++ {
			value := text[index]
			if value-'0' > DIGIT_COUNT-1 {
				switch entry.Case {
				case CASE_SNAKE:
					value = value | LETTER_CASE_BIT
				case CASE_CONSTANT:
					value = value &^ LETTER_CASE_BIT
				case CASE_ADA:
					if !acronym {
						value = value | LETTER_CASE_BIT
						if index == start {
							value = value &^ LETTER_CASE_BIT
						}
					}
				}
			}
			if place+size < len(form) {
				form[place+size] = value
			}
			size = size + 1
		}
	}
	subject.Places.Values[PLACE_SIZE] = Place(size)
}

// Reads the byte just past the word of the probe that opens at the start place into the end place.
// Words stand where the linter splits them: at an underscore, at an uppercase letter behind a
// lowercase letter or a digit, and at the last capital of a run a lowercase letter follows. An
// underscore at the start states an empty word.
func word_end(subject Formatter_Handle) {
	Formatter_Handle_Invariants(subject, "word_end.subject")
	text := subject.Probes.Value
	start := int(subject.Places.Values[PLACE_START])
	end := start
	for end < len(text) {
		value := text[end]
		if value == '_' {
			break
		}
		if end > start {
			if value-'A' < LETTER_COUNT {
				if text[end-1]-'a' < LETTER_COUNT {
					break
				}
				if text[end-1]-'0' < DIGIT_COUNT {
					break
				}
				if end+1 < len(text) {
					if text[end+1]-'a' < LETTER_COUNT {
						break
					}
				}
			}
		}
		end = end + 1
	}
	subject.Places.Values[PLACE_END] = Place(end)
}
