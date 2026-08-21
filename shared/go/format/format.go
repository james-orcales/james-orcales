// Package format states the canonical form of one Go source in this dialect. It exists because
// the printer answers what a tree writes and a caller asks something else: whether the file on
// disk already stands in the form this repository keeps. The printer is the engine and this is
// the policy above it, thus one place states the form and every caller reads the same answer.
package format

import (
	"local/james-orcales/shared/go/ast"
	"local/james-orcales/shared/go/printer"
	"local/james-orcales/shared/go/token"
	"local/james-orcales/shared/invariant/default"
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
func Place_Invariants(value Place, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int32(int32(value), PLACE_MINIMUM, PLACE_MAXIMUM).
		Ensure()
}

// Formatter is the caller-owned state one format runs through. A sort of imports reads the form
// the print wrote and turns the records inside it, thus the formatter holds that form and the
// bytes the sort stands between, and no storage of its own.
type Formatter struct {
	// Places holds the bytes one sort of imports reads and writes.
	Places [PLACE_SLOT_COUNT]Place
	// Forms holds the storage the sort reads the form from.
	Forms [FORM_SLOT_COUNT]printer.Form
	// Heads holds the text one read of a line asks the line to open with.
	Heads [HEAD_SLOT_COUNT]string
}

// Formatter_Invariants states one formatter holds a place for every byte the sort names.
func Formatter_Invariants(subject *Formatter, namespace invariant.Namespace) {
	invariant.Always(
		len(subject.Places) == PLACE_SLOT_COUNT,
		"A formatter holds one place for every byte the sort names.",
	)
}

// Format writes the canonical form of one parsed tree into caller storage and reports whether the
// whole form fit. A tree the parser refused writes what it holds, because a caller reads more
// from a partial form than from nothing.
func Format(
	subject *Formatter, writer *printer.Printer, destination printer.Form,
	tree *ast.Parse_State, source token.Source,
) (count printer.Form_Count, ok printer.Boolean) {
	defer func() {
		printer.Form_Count_Invariants(count, "format.count")
		printer.Boolean_Invariants(ok, "format.ok")
	}()
	Formatter_Invariants(subject, "format.subject")
	printer.Printer_Invariants(writer, "format.writer")
	printer.Form_Invariants(destination, "format.destination")
	ast.Parse_State_Invariants(tree, "format.tree")
	token.Source_Invariants(source, "format.source")
	count, ok = printer.Print(writer, destination, tree, source)
	subject.Forms[FORM_SLOT] = destination
	subject.Places[PLACE_LIMIT] = Place(count)
	sort_imports(subject)
	return count, ok
}

// Clean reports whether one source already stands in the canonical form, and whether the form it
// compared against fit the storage. A form that ran past the storage answers no, because the form
// the source would take was never written.
func Clean(
	subject *Formatter, writer *printer.Printer, destination printer.Form,
	tree *ast.Parse_State, source token.Source,
) (clean printer.Boolean, ok printer.Boolean) {
	defer func() {
		printer.Boolean_Invariants(clean, "clean.clean")
		printer.Boolean_Invariants(ok, "clean.ok")
	}()
	Formatter_Invariants(subject, "clean.subject")
	printer.Printer_Invariants(writer, "clean.writer")
	printer.Form_Invariants(destination, "clean.destination")
	ast.Parse_State_Invariants(tree, "clean.tree")
	token.Source_Invariants(source, "clean.source")
	count, held := Format(subject, writer, destination, tree, source)
	if !bool(held) {
		return false, false
	}
	if int(count) != len(source) {
		return false, true
	}
	for index := range int(count) {
		if destination[index] != source[index] {
			return false, true
		}
	}
	return true, true
}

// Sorts each run of imports the form holds by the path each one names. The tree states the order
// the author wrote and the canonical form states the order a reader searches, thus the sort
// stands here and never in the printer, which answers for the tree alone.
func sort_imports(subject *Formatter) {
	Formatter_Invariants(subject, "sort_imports.subject")
	subject.Places[PLACE_SCAN] = PLACE_MINIMUM
	for subject.Places[PLACE_SCAN] < subject.Places[PLACE_LIMIT] {
		if !bool(take_record(subject)) {
			subject.Places[PLACE_CURSOR] = subject.Places[PLACE_SCAN]
			// Every import of one file stands ahead of every other declaration,
			// thus the scan closes at the first line that opens something else and
			// reads no further than the imports it sorts.
			if bool(closes_imports(subject)) {
				return
			}
			close_place(subject)
			subject.Places[PLACE_SCAN] = subject.Places[PLACE_CURSOR]
			continue
		}
		take_run(subject)
		sort_run(subject)
	}
}

// Moves one read of the form past the line it stands on. A line the form never closed runs to the
// end of the form, because a refused print writes the bytes it reached and no line feed behind
// them.
func close_place(subject *Formatter) {
	Formatter_Invariants(subject, "close_place.subject")
	form := subject.Forms[FORM_SLOT]
	limit := subject.Places[PLACE_LIMIT]
	for subject.Places[PLACE_CURSOR] < limit {
		held := subject.Places[PLACE_CURSOR]
		subject.Places[PLACE_CURSOR] = held + 1
		if form[held] == '\n' {
			return
		}
	}
}

// Reports whether the line one read stands on closes the imports of the file. The package clause,
// an empty line, and a note each stand where an import may stand behind them; every other line
// states a declaration no import stands behind.
func closes_imports(subject *Formatter) (yes printer.Boolean) {
	defer func() { printer.Boolean_Invariants(yes, "closes_imports.yes") }()
	Formatter_Invariants(subject, "closes_imports.subject")
	subject.Heads[HEAD_SLOT] = PACKAGE_HEAD
	if bool(states_head(subject)) {
		return false
	}
	if bool(opens_note(subject)) {
		return false
	}
	subject.Heads[HEAD_SLOT] = LINE_HEAD
	return !states_head(subject)
}

// PACKAGE_HEAD is the text one package clause opens with.
const PACKAGE_HEAD = "package "

// LINE_HEAD is the text one empty line states.
const LINE_HEAD = "\n"

// Reports whether the line one read stands on opens one import declaration.
func opens_import(subject *Formatter) (yes printer.Boolean) {
	defer func() { printer.Boolean_Invariants(yes, "opens_import.yes") }()
	Formatter_Invariants(subject, "opens_import.subject")
	subject.Heads[HEAD_SLOT] = IMPORT_HEAD
	return states_head(subject)
}

// Reports whether the line one read stands on opens one note.
func opens_note(subject *Formatter) (yes printer.Boolean) {
	defer func() { printer.Boolean_Invariants(yes, "opens_note.yes") }()
	Formatter_Invariants(subject, "opens_note.subject")
	subject.Heads[HEAD_SLOT] = NOTE_HEAD
	return states_head(subject)
}

// Reports whether the line one read stands on opens with the text stated.
func states_head(subject *Formatter) (yes printer.Boolean) {
	defer func() { printer.Boolean_Invariants(yes, "states_head.yes") }()
	Formatter_Invariants(subject, "states_head.subject")
	form := subject.Forms[FORM_SLOT]
	head := subject.Heads[HEAD_SLOT]
	place := int(subject.Places[PLACE_CURSOR])
	if place+len(head) > int(subject.Places[PLACE_LIMIT]) {
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
func take_record(subject *Formatter) (yes printer.Boolean) {
	defer func() { printer.Boolean_Invariants(yes, "take_record.yes") }()
	Formatter_Invariants(subject, "take_record.subject")
	subject.Places[PLACE_CURSOR] = subject.Places[PLACE_SCAN]
	for subject.Places[PLACE_CURSOR] < subject.Places[PLACE_LIMIT] {
		if bool(opens_import(subject)) {
			close_place(subject)
			subject.Places[PLACE_SCAN_END] = subject.Places[PLACE_CURSOR]
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
func take_run(subject *Formatter) {
	Formatter_Invariants(subject, "take_run.subject")
	subject.Places[PLACE_HEAD] = subject.Places[PLACE_SCAN]
	for bool(take_record(subject)) {
		subject.Places[PLACE_SCAN] = subject.Places[PLACE_SCAN_END]
	}
	subject.Places[PLACE_TAIL] = subject.Places[PLACE_SCAN]
}

// Sorts one run of records by the path each one names. The sort carries the smallest record of
// what remains to the opening of what remains, thus every record stands where its path states and
// no record leaves the run it stood in.
func sort_run(subject *Formatter) {
	Formatter_Invariants(subject, "sort_run.subject")
	place := subject.Places[PLACE_HEAD]
	for place < subject.Places[PLACE_TAIL] {
		subject.Places[PLACE_SCAN] = place
		take_record(subject)
		subject.Places[PLACE_BEST] = place
		subject.Places[PLACE_BEST_END] = subject.Places[PLACE_SCAN_END]
		take_smallest(subject)
		subject.Places[PLACE_FROM] = place
		subject.Places[PLACE_MIDDLE] = subject.Places[PLACE_BEST]
		subject.Places[PLACE_TO] = subject.Places[PLACE_BEST_END]
		turn(subject)
		place = place + subject.Places[PLACE_BEST_END] - subject.Places[PLACE_BEST]
	}
	subject.Places[PLACE_SCAN] = subject.Places[PLACE_TAIL]
}

// Reads the record of the run that names the smallest path, from the record behind the one the
// sort stands on to the end of the run.
func take_smallest(subject *Formatter) {
	Formatter_Invariants(subject, "take_smallest.subject")
	best := subject.Places[PLACE_BEST]
	best_end := subject.Places[PLACE_BEST_END]
	subject.Places[PLACE_SCAN] = best_end
	for subject.Places[PLACE_SCAN] < subject.Places[PLACE_TAIL] {
		if !bool(take_record(subject)) {
			break
		}
		subject.Places[PLACE_BEST] = best
		subject.Places[PLACE_BEST_END] = best_end
		if bool(names_ahead(subject)) {
			best = subject.Places[PLACE_SCAN]
			best_end = subject.Places[PLACE_SCAN_END]
		}
		subject.Places[PLACE_SCAN] = subject.Places[PLACE_SCAN_END]
	}
	subject.Places[PLACE_BEST] = best
	subject.Places[PLACE_BEST_END] = best_end
}

// Reports whether the record one scan stands on names a path ahead of the path the smallest
// record names. A record states its path on the line it closes with, which is the import itself.
func names_ahead(subject *Formatter) (yes printer.Boolean) {
	defer func() { printer.Boolean_Invariants(yes, "names_ahead.yes") }()
	Formatter_Invariants(subject, "names_ahead.subject")
	form := subject.Forms[FORM_SLOT]
	subject.Places[PLACE_CURSOR] = subject.Places[PLACE_SCAN_END]
	take_path(subject)
	scan := subject.Places[PLACE_CURSOR]
	subject.Places[PLACE_CURSOR] = subject.Places[PLACE_BEST_END]
	take_path(subject)
	best := subject.Places[PLACE_CURSOR]
	for scan < subject.Places[PLACE_SCAN_END] {
		if best >= subject.Places[PLACE_BEST_END] {
			return false
		}
		if form[scan] != form[best] {
			yes = printer.Boolean(form[scan] < form[best])
			return yes
		}
		scan = scan + 1
		best = best + 1
	}
	yes = printer.Boolean(best < subject.Places[PLACE_BEST_END])
	return yes
}

// Moves one read from the byte a record closes at to the byte its path opens at, which stands
// behind the quotation mark the import line holds. A line that states no quotation mark opens its
// path at its own opening, so that a form the print refused compares by the bytes it holds.
func take_path(subject *Formatter) {
	Formatter_Invariants(subject, "take_path.subject")
	form := subject.Forms[FORM_SLOT]
	end := subject.Places[PLACE_CURSOR]
	open_line(subject)
	head := subject.Places[PLACE_CURSOR]
	for subject.Places[PLACE_CURSOR] < end {
		if form[subject.Places[PLACE_CURSOR]] == '"' {
			return
		}
		subject.Places[PLACE_CURSOR] = subject.Places[PLACE_CURSOR] + 1
	}
	subject.Places[PLACE_CURSOR] = head
}

// Moves one read from the byte a line closes at to the byte that line opens at.
func open_line(subject *Formatter) {
	Formatter_Invariants(subject, "open_line.subject")
	form := subject.Forms[FORM_SLOT]
	if subject.Places[PLACE_CURSOR] > PLACE_MINIMUM {
		subject.Places[PLACE_CURSOR] = subject.Places[PLACE_CURSOR] - 1
	}
	for subject.Places[PLACE_CURSOR] > PLACE_MINIMUM {
		if form[subject.Places[PLACE_CURSOR]-1] == '\n' {
			return
		}
		subject.Places[PLACE_CURSOR] = subject.Places[PLACE_CURSOR] - 1
	}
}

// Turns the bytes of one span so the record that opened in the middle of it opens the span. The
// turn stands on three reversals, thus the form holds every byte it held and no storage stands
// behind the sort.
func turn(subject *Formatter) {
	Formatter_Invariants(subject, "turn.subject")
	from := subject.Places[PLACE_FROM]
	middle := subject.Places[PLACE_MIDDLE]
	to := subject.Places[PLACE_TO]
	subject.Places[PLACE_TO] = middle
	reverse(subject)
	subject.Places[PLACE_FROM] = middle
	subject.Places[PLACE_TO] = to
	reverse(subject)
	subject.Places[PLACE_FROM] = from
	reverse(subject)
}

// Reverses the bytes of one span of the form.
func reverse(subject *Formatter) {
	Formatter_Invariants(subject, "reverse.subject")
	form := subject.Forms[FORM_SLOT]
	head := subject.Places[PLACE_FROM]
	tail := subject.Places[PLACE_TO]
	for head < tail-1 {
		held := form[head]
		form[head] = form[tail-1]
		form[tail-1] = held
		head = head + 1
		tail = tail - 1
	}
}
