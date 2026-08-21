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

// PLACE_SLOT_COUNT is the byte slot count one formatter holds.
const PLACE_SLOT_COUNT = 11

// FORM_SLOT holds the storage the format wrote its form into.
const FORM_SLOT = 0

// FORM_SLOT_COUNT is the storage slot count one formatter holds.
const FORM_SLOT_COUNT = 1

// HEAD_SLOT holds the text one read of a line asks the line to open with.
const HEAD_SLOT = 0

// HEAD_SLOT_COUNT is the text slot count one formatter holds.
const HEAD_SLOT_COUNT = 1

// Place names one byte of the form a format writes.
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

// Formatter_Fields states the concrete storage layout independently of its validated view.
type Formatter_Fields struct {
	// Places holds the bytes one sort of imports reads and writes.
	Places Place_Storage
	// Forms holds the storage the sort reads the form from.
	Forms Forms
	// Heads holds the text one read of a line asks the line to open with.
	Heads Heads
}

// Formatter_Fields_Invariants composes every concrete field once.
func Formatter_Fields_Invariants(value Formatter_Fields, namespace aver.Namespace) {
	Place_Storage_Invariants(value.Places, namespace)
	Forms_Invariants(value.Forms, namespace)
	Heads_Invariants(value.Heads, namespace)
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

// Formatter is one validated view over caller-owned format storage.
type Formatter Formatter_Fields

// Formatter_Invariants fixes sort storage without proving stale mutable views.
func Formatter_Invariants(value Formatter, namespace aver.Namespace) {
	Place_Storage_Invariants(value.Places, namespace)
	Forms_Stored_Invariants(Forms_Stored(value.Forms), namespace)
	Heads_Stored_Invariants(Heads_Stored(value.Heads), namespace)
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
// whole form fit. A tree the parser refused writes what it holds, because a caller reads more
// from a partial form than from nothing.
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
	return result
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
