// Package format states the canonical form of one Go source in this dialect. It exists because
// the printer answers what a tree writes and a caller asks something else: whether the file on
// disk already stands in the form this repository keeps. The printer is the engine and this is
// the policy above it, thus one place states the form and every caller reads the same answer.
package format

import "local/james-orcales/shared/go/ast"
import "local/james-orcales/shared/go/printer"
import "local/james-orcales/shared/go/token"
import "local/james-orcales/shared/sim/aver/default"

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

// SCAN_SIZE_MINIMUM is the shortest form a sort reads. Every file states a package clause, a name
// of one letter at least, and the line that closes the clause, thus storage shorter than that
// holds no run of imports and the sort never reads it.
const SCAN_SIZE_MINIMUM = HEAD_SIZE_MAXIMUM + PACKAGE_NAME_SIZE_MINIMUM + LINE_HEAD_SIZE

// PACKAGE_NAME_SIZE_MINIMUM is the byte count of the shortest package name, one letter.
const PACKAGE_NAME_SIZE_MINIMUM = 1

// Scan is the storage one sort of imports reads. It is the destination the sort was handed, and
// the sort reads it only where the storage holds a whole file, thus it is never shorter than the
// shortest form.
type Scan printer.Form

// Scan_Invariants states every storage a sort reads.
func Scan_Invariants(value Scan, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(len(value), SCAN_SIZE_MINIMUM, printer.FORM_SIZE_MAXIMUM).
    Ensure()
}

// PATH_SIZE_MINIMUM is the byte count of the shortest import path: a quotation mark, one letter,
// and the quotation mark that closes it.
const PATH_SIZE_MINIMUM = 3

// RUN_SCAN_SIZE_MINIMUM is the shortest form a run of imports stands in: the shortest file, the
// line that opens the run, and one import of the shortest path. A sort reads a run only after it
// took a record of one, thus storage shorter than this holds no run and no step of one reads it.
const RUN_SCAN_SIZE_MINIMUM = SCAN_SIZE_MINIMUM + LINE_HEAD_SIZE + IMPORT_LINE_SIZE_MINIMUM

// IMPORT_LINE_SIZE_MINIMUM is the byte count of the shortest import line: the text it opens with,
// the shortest path, and the line that closes it.
const IMPORT_LINE_SIZE_MINIMUM = IMPORT_HEAD_SIZE + PATH_SIZE_MINIMUM + LINE_HEAD_SIZE

// Run_Scan is the storage one step of a run of imports reads. It is the storage the sort was
// handed, and a step reads it only where a record of one import already stands in it.
type Run_Scan printer.Form

// Run_Scan_Invariants states every storage a step of a run reads.
func Run_Scan_Invariants(value Run_Scan, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(
    len(value),
    RUN_SCAN_SIZE_MINIMUM,
    printer.FORM_SIZE_MAXIMUM,
  ).
    Ensure()
}

// COMPARE_SCAN_SIZE_MINIMUM is the shortest form two imports stand in. A compare orders one
// record against another, thus storage that holds one import alone no compare reads.
const COMPARE_SCAN_SIZE_MINIMUM = RUN_SCAN_SIZE_MINIMUM + IMPORT_LINE_SIZE_MINIMUM

// Compare_Scan is the storage one compare of two imports reads. It is the storage the run was
// handed, and a compare reads it only where a second record already stands past the first.
type Compare_Scan printer.Form

// Compare_Scan_Invariants states every storage a compare reads.
func Compare_Scan_Invariants(value Compare_Scan, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(
    len(value),
    COMPARE_SCAN_SIZE_MINIMUM,
    printer.FORM_SIZE_MAXIMUM,
  ).
    Ensure()
}

// DECLARATION_SIZE_MINIMUM is the byte count of the shortest declaration the rename table takes:
// the word that opens it, a name of two letters, the equals sign the spaces around it, the value,
// and the line that closes the declaration. A name of one letter stands in its case already and
// the table never takes it.
const DECLARATION_SIZE_MINIMUM = 11

// NAME_SCAN_SIZE_MINIMUM is the shortest form a rename reads: the shortest file, the line that
// opens the declaration, and the shortest declaration the table takes. A rename reads the storage
// only where the table holds a name, thus storage shorter than this no rename reads.
const NAME_SCAN_SIZE_MINIMUM = SCAN_SIZE_MINIMUM + LINE_HEAD_SIZE + DECLARATION_SIZE_MINIMUM

// Name_Scan is the storage one rename reads. It is the storage the format was handed, and a
// rename reads it only where a declaration the table took already stands in it.
type Name_Scan printer.Form

// Name_Scan_Invariants states every storage a rename reads.
func Name_Scan_Invariants(value Name_Scan, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(
    len(value),
    NAME_SCAN_SIZE_MINIMUM,
    printer.FORM_SIZE_MAXIMUM,
  ).
    Ensure()
}

// LINE_HEAD_SIZE is the byte count of the text one empty line states.
const LINE_HEAD_SIZE = len(LINE_HEAD)

// NOTE_HEAD_SIZE is the byte count of the text one note opens with.
const NOTE_HEAD_SIZE = len(NOTE_HEAD)

// IMPORT_HEAD_SIZE is the byte count of the text one import declaration opens with.
const IMPORT_HEAD_SIZE = len(IMPORT_HEAD)

// HEAD_SIZE_MAXIMUM is the widest line prefix one formatter searches for.
const HEAD_SIZE_MAXIMUM = len(PACKAGE_HEAD)

// Head is one line prefix a formatter searches for.
type Head string

// Head_Invariants states every line prefix a formatter searches for. Four prefixes exist and no
// other stands, thus the width of one is one of four and never a range between them.
func Head_Invariants(value Head, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_4_Int(
    len(value),
    LINE_HEAD_SIZE,
    NOTE_HEAD_SIZE,
    IMPORT_HEAD_SIZE,
    HEAD_SIZE_MAXIMUM,
  ).
    Ensure()
}

// Place_Storage keeps caller-owned sort positions distinct from their view.
type Place_Storage struct {
  // Values makes ownership explicit at the package boundary.
  Values Places
}

// Place_Storage_Invariants composes sort-position storage.
func Place_Storage_Invariants(value *Place_Storage, namespace aver.Namespace) {
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
    uint8(value),
    uint8(CASE_SNAKE),
    uint8(CASE_ADA),
    uint8(CASE_CONSTANT),
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
func Name_Invariants(value *Name, namespace aver.Namespace) {
  Span_Invariants(&value.Text, namespace)
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
func Span_Invariants(value *Span, namespace aver.Namespace) {
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
func Name_Storage_Invariants(value *Name_Storage, namespace aver.Namespace) {
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
func Text_Storage_Invariants(value *Text_Storage, namespace aver.Namespace) {
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
func Slot_Storage_Invariants(value *Slot_Storage, namespace aver.Namespace) {
  Slots_Invariants(value.Values, namespace)
}

// Sources keeps the source one declaration reads names from in named storage.
type Sources struct {
  // Value survives every step of a declaration, each of which reads a name from it.
  Value token.Source
}

// Sources_Invariants composes the source domain.
func Sources_Invariants(value *Sources, namespace aver.Namespace) {
  token.Source_Invariants(value.Value, namespace)
}

// DECLARING_SOURCE_SIZE_MINIMUM is the shortest source that declares a name the table takes: the
// shortest file, the line that opens the declaration, and the shortest declaration. A source
// shorter than this declares nothing and no step of a declaration reads it.
const DECLARING_SOURCE_SIZE_MINIMUM = SCAN_SIZE_MINIMUM + LINE_HEAD_SIZE +
  DECLARATION_SIZE_MINIMUM

// Declaring_Source is the source one step of a declaration reads. A step reads it only where the
// tree already named a declaration standing in it.
type Declaring_Source token.Source

// Declaring_Source_Invariants states every source a step of a declaration reads.
func Declaring_Source_Invariants(value Declaring_Source, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(
    len(value),
    DECLARING_SOURCE_SIZE_MINIMUM,
    token.SOURCE_SIZE_MAXIMUM,
  ).
    Ensure()
}

// WORD_SIZE_MINIMUM is the byte count of the shortest word a search looks for. A word of no byte
// names nothing, thus a search is handed one letter at least and never an empty view.
const WORD_SIZE_MINIMUM = 1

// DECLARING_WRAPPER_SIZE is the byte count of the shortest text that stands around a declared
// name: the shortest source that declares one, less the name itself.
const DECLARING_WRAPPER_SIZE = DECLARING_SOURCE_SIZE_MINIMUM - RENAMED_WORD_SIZE_MINIMUM

// WORD_SIZE_MAXIMUM is the byte count of the widest word a search looks for. Every word the table
// holds is a name one source spelled, thus the widest stands in the widest source with nothing
// around it but the shortest text a declaration needs.
const WORD_SIZE_MAXIMUM = token.SOURCE_SIZE_MAXIMUM - DECLARING_WRAPPER_SIZE

// Probe is the word one search of the table looks for. A rename reads it out of the form and a
// spell reads it out of the table text. It is its own type and never the scanned storage, because
// Scan already stands in the formatter chain and one type stands there one time. A search takes it
// as an argument and the table never holds it, for the reason Names_Invariants states: a step that
// carried it would owe the width of every word the table could hold.
type Probe []byte

// Probe_Invariants states every bounded word a search looks for.
func Probe_Invariants(value Probe, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(len(value), WORD_SIZE_MINIMUM, WORD_SIZE_MAXIMUM).
    Ensure()
}

// RENAMED_WORD_SIZE_MINIMUM is the byte count of the shortest word a spell walks. A name of one
// letter stands in its case already, thus the table never takes it and no spell reads it.
const RENAMED_WORD_SIZE_MINIMUM = 2

// Spelled is the word one spell walks segment by segment. It is a name the table took, and the
// table takes a name only where the case it stands in is not the case the linter asks.
type Spelled []byte

// Spelled_Invariants states every bounded word a spell walks.
func Spelled_Invariants(value Spelled, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(len(value), RENAMED_WORD_SIZE_MINIMUM, WORD_SIZE_MAXIMUM).
    Ensure()
}

// Table_Fields is the storage one rename reads and writes: the names it holds, the text of those
// names, the index over them, and the word one search looks for. A rename step reads no line
// prefix and no source, thus it takes this and owes those domains nothing.
type Table_Fields struct {
  // Names holds the declared names one rename spells anew.
  Names Name_Storage
  // Texts holds the text of every declared name.
  Texts Text_Storage
  // Slots holds the hash index over the declared names.
  Slots Slot_Storage
}

// Table_Fields_Invariants composes table storage.
func Table_Fields_Invariants(value *Table_Fields, namespace aver.Namespace) {
  Name_Storage_Invariants(&value.Names, namespace)
  Text_Storage_Invariants(&value.Texts, namespace)
  Slot_Storage_Invariants(&value.Slots, namespace)
}

// Table_Handle gives caller-owned table storage one identity.
type Table_Handle *Table_Fields

// Table_Handle_Invariants composes present table storage.
func Table_Handle_Invariants(value Table_Handle, namespace aver.Namespace) {
  if value == nil {
    return
  }
  Table_Fields_Invariants(value, namespace)
}

// Formatter is the caller-owned format storage.
type Formatter struct {
  // Places holds the bytes one sort of imports reads and writes.
  Places Place_Storage
  // Table_Fields stays embedded so a rename step takes the table alone.
  Table_Fields
  // Sources holds the source one declaration reads names from.
  Sources Sources
}

// Formatter_Invariants composes caller-owned format storage.
func Formatter_Invariants(value *Formatter, namespace aver.Namespace) {
  Place_Storage_Invariants(&value.Places, namespace)
  Table_Fields_Invariants(&value.Table_Fields, namespace)
  Sources_Invariants(&value.Sources, namespace)
}

// Formatter_Handle gives caller-owned format state one identity.
type Formatter_Handle *Formatter

// Formatter_Handle_Invariants composes present format storage.
func Formatter_Handle_Invariants(value Formatter_Handle, namespace aver.Namespace) {
  if value == nil {
    return
  }
  Formatter_Invariants(value, namespace)
}

// Format writes the canonical form of one parsed tree into caller storage and reports whether the
// whole form fit. It renames by the table Declare fills and never by the tree alone, because a
// name the tree declares is spelled in files the format never sees. A tree the parser refused
// writes what it holds, because a caller reads more from a partial form than from nothing.
func Format(
  subject Formatter_Handle,
  writer printer.Printer_Handle,
  destination printer.Form,
  tree ast.Parse_State_Handle,
  source token.Source,
) (result printer.Print_Result) {
  defer func() { printer.Print_Result_Invariants(&result, "format.result") }()
  Formatter_Handle_Invariants(subject, "format.subject")
  printer.Printer_Handle_Invariants(writer, "format.writer")
  printer.Form_Invariants(destination, "format.destination")
  ast.Parse_State_Handle_Invariants(tree, "format.tree")
  token.Source_Invariants(source, "format.source")
  result = printer.Print(writer, destination, tree, source)
  // Storage too short for one package clause holds no run of imports and no name the
  // rename reaches, thus the sort reads nothing and the form stands as the printer wrote it.
  if len(destination) < SCAN_SIZE_MINIMUM {
    return result
  }
  subject.Places.Values[PLACE_LIMIT] = Place(result.Count)
  sort_imports(subject.Places, Scan(destination))
  // The sort takes the empty lines the runs state nothing for out of the form and opens
  // one where the runs part, thus the count reads what the form holds once it sorted.
  result.Count = printer.Form_Count(subject.Places.Values[PLACE_LIMIT])
  // A table that holds no name reads no byte of the form again, thus the rename costs a
  // program that declares every name in its case nothing.
  if subject.Places.Values[PLACE_NAMES] == 0 {
    return result
  }
  // The table holds a name, thus the storage holds the declaration that spelled it. The
  // guard states what the table already proved.
  if len(destination) < NAME_SCAN_SIZE_MINIMUM {
    return result
  }
  held := Rename(subject, destination, result.Count)
  if !bool(held.OK) {
    result.OK = false
  }
  result.Count = held.Count
  return result
}

// Rename spells every name of one form the way the table asks, and reports the form it wrote and
// whether the storage held it. A caller that holds a form of its own renames it the way one
// format renames the form it wrote, thus a print and a rename stand apart.
func Rename(
  subject Formatter_Handle,
  destination printer.Form,
  count printer.Form_Count,
) (result printer.Print_Result) {
  defer func() { printer.Print_Result_Invariants(&result, "rename_names.result") }()
  Formatter_Handle_Invariants(subject, "rename_names.subject")
  printer.Form_Invariants(destination, "rename_names.destination")
  printer.Form_Count_Invariants(count, "rename_names.count")
  result = printer.Print_Result{Count: count, OK: true}
  // A table that holds no name reads no byte of the form, thus the rename costs a program
  // that declares every name in its case nothing.
  if subject.Places.Values[PLACE_NAMES] == 0 {
    return result
  }
  // The table holds a name, thus the storage holds the declaration that spelled it. The
  // guard states what the table already proved.
  if len(destination) < NAME_SCAN_SIZE_MINIMUM {
    return result
  }
  subject.Places.Values[PLACE_LIMIT] = Place(count)
  fit := rename(subject.Places, Name_Scan(destination), &subject.Table_Fields)
  result.Count = printer.Form_Count(subject.Places.Values[PLACE_LIMIT])
  result.OK = printer.Boolean(fit)
  return result
}

// Declare takes each name one file declares in the wrong case into the table every later format
// renames by, and reports whether the table held room for all of them. A name one file declares
// is spelled in other files too, thus a caller declares every file of a program before it formats
// any, and the rename reaches every file that spells the name.
func Declare(
  subject Formatter_Handle,
  tree ast.Parse_State_Handle,
  source token.Source,
) (fit printer.Boolean) {
  defer func() { printer.Boolean_Invariants(fit, "declare.fit") }()
  Formatter_Handle_Invariants(subject, "declare.subject")
  ast.Parse_State_Handle_Invariants(tree, "declare.tree")
  token.Source_Invariants(source, "declare.source")
  subject.Sources.Value = source
  return take_names(subject.Places, &subject.Table_Fields, subject.Sources, &tree.Syntax_Tree)
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
func Clean_Result_Invariants(value *Clean_Result, namespace aver.Namespace) {
  printer.Boolean_Invariants(value.Clean, namespace)
  Fit_Invariants(value.OK, namespace)
}

// Clean reports whether one source already stands in the canonical form, and whether the form it
// compared against fit the storage. A form that ran past the storage answers no, because the form
// the source would take was never written.
func Clean(
  subject Formatter_Handle,
  writer printer.Printer_Handle,
  destination printer.Form,
  tree ast.Parse_State_Handle,
  source token.Source,
) (result Clean_Result) {
  defer func() { Clean_Result_Invariants(&result, "clean.result") }()
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
func sort_imports(places Place_Storage, scanned Scan) {
  Place_Storage_Invariants(&places, "sort_imports.places")
  Scan_Invariants(scanned, "sort_imports.scanned")
  places.Values[PLACE_SCAN] = PLACE_MINIMUM
  for places.Values[PLACE_SCAN] < places.Values[PLACE_LIMIT] {
    if bool(take_record(places, scanned)) {
      break
    }
    places.Values[PLACE_CURSOR] = places.Values[PLACE_SCAN]
    // Every import of one file stands ahead of every other declaration, thus the
    // scan closes at the first line that opens something else.
    if bool(closes_imports(places, scanned)) {
      return
    }
    close_place(places, scanned)
    places.Values[PLACE_SCAN] = places.Values[PLACE_CURSOR]
  }
  if !(places.Values[PLACE_SCAN] < places.Values[PLACE_LIMIT]) {
    return
  }
  // A record of one import already stands in the storage, thus the storage holds a whole
  // run. The guard states what the record proved.
  if len(scanned) < RUN_SCAN_SIZE_MINIMUM {
    return
  }
  take_section(places, Run_Scan(scanned))
  sort_run(places, Run_Scan(scanned))
  part_runs(places, Run_Scan(scanned))
}

// Reads the whole run of imports the scan stands at the opening of, taking out every empty line
// the author left between two of them. The rule states which run each path stands in, thus a line
// the author wrote between two imports states nothing the form keeps.
func take_section(places Place_Storage, scanned Run_Scan) {
  Place_Storage_Invariants(&places, "take_section.places")
  Run_Scan_Invariants(scanned, "take_section.scanned")
  form := scanned
  places.Values[PLACE_HEAD] = places.Values[PLACE_SCAN]
  for places.Values[PLACE_SCAN] < places.Values[PLACE_LIMIT] {
    if bool(take_record(places, Scan(scanned))) {
      places.Values[PLACE_SCAN] = places.Values[PLACE_SCAN_END]
      continue
    }
    head := int(places.Values[PLACE_SCAN])
    if form[head] != '\n' {
      break
    }
    // An empty line an import stands behind parts no run of its own, thus the
    // form drops it and the rule parts the runs instead.
    places.Values[PLACE_SCAN] = Place(head + 1)
    found := take_record(places, Scan(scanned))
    places.Values[PLACE_SCAN] = Place(head)
    if !bool(found) {
      break
    }
    limit := int(places.Values[PLACE_LIMIT])
    copy(form[head:], form[head+1:limit])
    places.Values[PLACE_LIMIT] = Place(limit - 1)
  }
  places.Values[PLACE_TAIL] = places.Values[PLACE_SCAN]
}

// Moves one read of the form past the line it stands on. A line the form never closed runs to the
// end of the form, because a refused print writes the bytes it reached and no line feed behind
// them.
func close_place(places Place_Storage, scanned Scan) {
  Place_Storage_Invariants(&places, "close_place.places")
  Scan_Invariants(scanned, "close_place.scanned")
  form := scanned
  limit := places.Values[PLACE_LIMIT]
  for places.Values[PLACE_CURSOR] < limit {
    held := places.Values[PLACE_CURSOR]
    places.Values[PLACE_CURSOR] = held + 1
    if form[held] == '\n' {
      return
    }
  }
}

// Reports whether the line one read stands on closes the imports of the file. The package clause,
// an empty line, and a note each stand where an import may stand behind them; every other line
// states a declaration no import stands behind.
func closes_imports(places Place_Storage, scanned Scan) (yes printer.Boolean) {
  defer func() { printer.Boolean_Invariants(yes, "closes_imports.yes") }()
  Place_Storage_Invariants(&places, "closes_imports.places")
  Scan_Invariants(scanned, "closes_imports.scanned")
  if bool(states_head(places, scanned, PACKAGE_HEAD)) {
    return false
  }
  if bool(opens_note(places, scanned)) {
    return false
  }
  return !states_head(places, scanned, LINE_HEAD)
}

// PACKAGE_HEAD is the text one package clause opens with.
const PACKAGE_HEAD = "package "

// LINE_HEAD is the text one empty line states.
const LINE_HEAD = "\n"

// Reports whether the line one read stands on opens one import declaration.
func opens_import(places Place_Storage, scanned Scan) (yes printer.Boolean) {
  defer func() { printer.Boolean_Invariants(yes, "opens_import.yes") }()
  Place_Storage_Invariants(&places, "opens_import.places")
  Scan_Invariants(scanned, "opens_import.scanned")
  return states_head(places, scanned, IMPORT_HEAD)
}

// Reports whether the line one read stands on opens one note.
func opens_note(places Place_Storage, scanned Scan) (yes printer.Boolean) {
  defer func() { printer.Boolean_Invariants(yes, "opens_note.yes") }()
  Place_Storage_Invariants(&places, "opens_note.places")
  Scan_Invariants(scanned, "opens_note.scanned")
  return states_head(places, scanned, NOTE_HEAD)
}

// Reports whether the line one read stands on opens with the text stated.
func states_head(places Place_Storage, scanned Scan, head Head) (yes printer.Boolean) {
  defer func() { printer.Boolean_Invariants(yes, "states_head.yes") }()
  Place_Storage_Invariants(&places, "states_head.places")
  Scan_Invariants(scanned, "states_head.scanned")
  Head_Invariants(head, "states_head.head")
  form := scanned
  place := int(places.Values[PLACE_CURSOR])
  if place+len(head) > int(places.Values[PLACE_LIMIT]) {
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
func take_record(places Place_Storage, scanned Scan) (yes printer.Boolean) {
  defer func() { printer.Boolean_Invariants(yes, "take_record.yes") }()
  Place_Storage_Invariants(&places, "take_record.places")
  Scan_Invariants(scanned, "take_record.scanned")
  places.Values[PLACE_CURSOR] = places.Values[PLACE_SCAN]
  for places.Values[PLACE_CURSOR] < places.Values[PLACE_LIMIT] {
    if bool(opens_import(places, scanned)) {
      close_place(places, scanned)
      places.Values[PLACE_SCAN_END] = places.Values[PLACE_CURSOR]
      return true
    }
    if !bool(opens_note(places, scanned)) {
      return false
    }
    close_place(places, scanned)
  }
  return false
}

// Sorts one run of records by the path each one names. The sort carries the smallest record of
// what remains to the opening of what remains, thus every record stands where its path states and
// no record leaves the run it stood in.
func sort_run(places Place_Storage, scanned Run_Scan) {
  Place_Storage_Invariants(&places, "sort_run.places")
  Run_Scan_Invariants(scanned, "sort_run.scanned")
  place := places.Values[PLACE_HEAD]
  for place < places.Values[PLACE_TAIL] {
    places.Values[PLACE_SCAN] = place
    take_record(places, Scan(scanned))
    places.Values[PLACE_BEST] = place
    places.Values[PLACE_BEST_END] = places.Values[PLACE_SCAN_END]
    take_smallest(places, scanned)
    places.Values[PLACE_FROM] = place
    places.Values[PLACE_MIDDLE] = places.Values[PLACE_BEST]
    places.Values[PLACE_TO] = places.Values[PLACE_BEST_END]
    turn(places, scanned)
    size := places.Values[PLACE_BEST_END] - places.Values[PLACE_BEST]
    place = place + size
  }
  places.Values[PLACE_SCAN] = places.Values[PLACE_TAIL]
}

// Reads the record of the run that names the smallest path, from the record behind the one the
// sort stands on to the end of the run.
func take_smallest(places Place_Storage, scanned Run_Scan) {
  Place_Storage_Invariants(&places, "take_smallest.places")
  Run_Scan_Invariants(scanned, "take_smallest.scanned")
  best := places.Values[PLACE_BEST]
  best_end := places.Values[PLACE_BEST_END]
  places.Values[PLACE_SCAN] = best_end
  for places.Values[PLACE_SCAN] < places.Values[PLACE_TAIL] {
    if !bool(take_record(places, Scan(scanned))) {
      break
    }
    places.Values[PLACE_BEST] = best
    places.Values[PLACE_BEST_END] = best_end
    // A second record stands past the first, thus the storage holds two imports.
    // The guard states what the record proved.
    if len(scanned) < COMPARE_SCAN_SIZE_MINIMUM {
      break
    }
    if bool(names_ahead(places, Compare_Scan(scanned))) {
      best = places.Values[PLACE_SCAN]
      best_end = places.Values[PLACE_SCAN_END]
    }
    places.Values[PLACE_SCAN] = places.Values[PLACE_SCAN_END]
  }
  places.Values[PLACE_BEST] = best
  places.Values[PLACE_BEST_END] = best_end
}

// Reports whether the record one scan stands on names a path ahead of the path the smallest
// record names. A record states its path on the line it closes with, which is the import itself.
func names_ahead(places Place_Storage, compared Compare_Scan) (yes printer.Boolean) {
  defer func() { printer.Boolean_Invariants(yes, "names_ahead.yes") }()
  Place_Storage_Invariants(&places, "names_ahead.places")
  Compare_Scan_Invariants(compared, "names_ahead.compared")
  form := compared
  places.Values[PLACE_CURSOR] = places.Values[PLACE_SCAN_END]
  take_path(places, compared)
  scan := places.Values[PLACE_CURSOR]
  scan_kind := path_kind(places, compared)
  places.Values[PLACE_CURSOR] = places.Values[PLACE_BEST_END]
  take_path(places, compared)
  best := places.Values[PLACE_CURSOR]
  best_kind := path_kind(places, compared)
  // The standard library stands ahead of the module itself, and the module stands ahead of
  // every other one, thus the run a path stands in orders it before the path does.
  if scan_kind != best_kind {
    yes = printer.Boolean(scan_kind < best_kind)
    return yes
  }
  for scan < places.Values[PLACE_SCAN_END] {
    if !(best < places.Values[PLACE_BEST_END]) {
      return false
    }
    if form[scan] != form[best] {
      yes = printer.Boolean(form[scan] < form[best])
      return yes
    }
    scan = scan + 1
    best = best + 1
  }
  yes = printer.Boolean(best < places.Values[PLACE_BEST_END])
  return yes
}

// Moves one read from the byte a record closes at to the byte its path opens at, which stands
// behind the quotation mark the import line holds. A line that states no quotation mark opens its
// path at its own opening, so that a form the print refused compares by the bytes it holds.
func take_path(places Place_Storage, compared Compare_Scan) {
  Place_Storage_Invariants(&places, "take_path.places")
  Compare_Scan_Invariants(compared, "take_path.compared")
  form := compared
  end := places.Values[PLACE_CURSOR]
  open_line(places, compared)
  head := places.Values[PLACE_CURSOR]
  for places.Values[PLACE_CURSOR] < end {
    if form[places.Values[PLACE_CURSOR]] == '"' {
      return
    }
    places.Values[PLACE_CURSOR] = places.Values[PLACE_CURSOR] + 1
  }
  places.Values[PLACE_CURSOR] = head
}

// Moves one read from the byte a line closes at to the byte that line opens at.
func open_line(places Place_Storage, compared Compare_Scan) {
  Place_Storage_Invariants(&places, "open_line.places")
  Compare_Scan_Invariants(compared, "open_line.compared")
  form := compared
  if places.Values[PLACE_CURSOR] > PLACE_MINIMUM {
    places.Values[PLACE_CURSOR] = places.Values[PLACE_CURSOR] - 1
  }
  for places.Values[PLACE_CURSOR] > PLACE_MINIMUM {
    if form[places.Values[PLACE_CURSOR]-1] == '\n' {
      return
    }
    places.Values[PLACE_CURSOR] = places.Values[PLACE_CURSOR] - 1
  }
}

// Turns the bytes of one span so the record that opened in the middle of it opens the span. The
// turn stands on three reversals, thus the form holds every byte it held and no storage stands
// behind the sort.
func turn(places Place_Storage, scanned Run_Scan) {
  Place_Storage_Invariants(&places, "turn.places")
  Run_Scan_Invariants(scanned, "turn.scanned")
  from := places.Values[PLACE_FROM]
  middle := places.Values[PLACE_MIDDLE]
  to := places.Values[PLACE_TO]
  places.Values[PLACE_TO] = middle
  reverse(places, scanned)
  places.Values[PLACE_FROM] = middle
  places.Values[PLACE_TO] = to
  reverse(places, scanned)
  places.Values[PLACE_FROM] = from
  reverse(places, scanned)
}

// Reverses the bytes of one span of the form.
func reverse(places Place_Storage, scanned Run_Scan) {
  Place_Storage_Invariants(&places, "reverse.places")
  Run_Scan_Invariants(scanned, "reverse.scanned")
  form := scanned
  head := places.Values[PLACE_FROM]
  tail := places.Values[PLACE_TO]
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
// that name states the rest. A struct field takes the exported case whatever the author wrote,
// because every package that reads the type spells that field.
func take_names(
  places Place_Storage,
  table Table_Handle,
  sources Sources,
  tree ast.Syntax_Tree_Handle,
) (fit printer.Boolean) {
  defer func() { printer.Boolean_Invariants(fit, "take_names.fit") }()
  Place_Storage_Invariants(&places, "take_names.places")
  Table_Handle_Invariants(table, "take_names.table")
  Sources_Invariants(&sources, "take_names.sources")
  ast.Syntax_Tree_Handle_Invariants(tree, "take_names.tree")
  places.Values[PLACE_DROPPED] = 0
  // A walk by the links meets every node a clean parse wrote in the order the source states
  // the declarations, because only a wrap, which declares no name, precedes what it adopted.
  slot := ast.INDEX_FIRST
  for range ast.NODE_COUNT_MAXIMUM {
    if slot == ast.INDEX_ABSENT {
      break
    }
    node := ast.Syntax_Node_At(tree, slot)
    next := ast.Index(node.First_Child)
    climb := slot
    for range ast.DEPTH_MAXIMUM + 1 {
      if next != ast.INDEX_ABSENT {
        break
      }
      if climb == ast.INDEX_ABSENT {
        break
      }
      next = ast.Index(ast.Syntax_Node_At(tree, climb).Next)
      climb = ast.Index(ast.Syntax_Node_At(tree, climb).Parent)
    }
    slot = next
    place := node.Token
    want := CASE_SNAKE
    switch node.Kind {
    case ast.NODE_CONSTANT_NAME:
      want = CASE_CONSTANT
    case ast.NODE_FIELD_NAME:
      want = CASE_ADA
    case ast.NODE_PARAMETER_NAME, ast.NODE_DEFINE, ast.NODE_TYPE_PARAMETER:
      // A var or a type node stands on its keyword, and its name stands one token behind.
    case ast.NODE_VARIABLE, ast.NODE_TYPE:
      place = node.Token + 1
      // A function states its name in its first identifier child, because its receiver
      // stands in a node of its own.
    case ast.NODE_FUNCTION:
      child := ast.Index(node.First_Child)
      for child != ast.INDEX_ABSENT {
        if ast.Syntax_Node_At(tree, child).Kind == ast.NODE_IDENTIFIER {
          break
        }
        child = ast.Index(ast.Syntax_Node_At(tree, child).Next)
      }
      if child == ast.INDEX_ABSENT {
        continue
      }
      place = ast.Syntax_Node_At(tree, child).Token
      one := ast.Syntax_Token_At(tree, place)
      if string(token.Text(sources.Value, one)) == TEST_MAIN {
        continue
      }
    default:
      continue
    }
    places.Values[PLACE_TOKEN] = Place(place)
    table.Names.Values[places.Values[PLACE_NAMES]].Case = want
    // The tree named a declaration, thus the source holds one.
    if len(sources.Value) < DECLARING_SOURCE_SIZE_MINIMUM {
      return false
    }
    take_name_list(places, table, Declaring_Source(sources.Value), tree)
  }
  return printer.Boolean(places.Values[PLACE_DROPPED] == 0)
}

// Reads the list of names the candidate opens, joined by commas, which is how a var, a define, a
// field, a parameter, and a type parameter each declare more than one.
func take_name_list(
  places Place_Storage,
  table Table_Handle,
  sources Declaring_Source,
  tree ast.Syntax_Tree_Handle,
) {
  Place_Storage_Invariants(&places, "take_name_list.places")
  Table_Handle_Invariants(table, "take_name_list.table")
  Declaring_Source_Invariants(sources, "take_name_list.sources")
  ast.Syntax_Tree_Handle_Invariants(tree, "take_name_list.tree")
  place := ast.Token_Index(places.Values[PLACE_TOKEN])
  want := table.Names.Values[places.Values[PLACE_NAMES]].Case
  // A name is no end token, thus the token behind it stands inside the run, and the end
  // token is no comma, thus the list stops there at the latest.
  for range ast.TOKEN_COUNT_MAXIMUM {
    places.Values[PLACE_TOKEN] = Place(place)
    table.Names.Values[places.Values[PLACE_NAMES]].Case = want
    take_name(places, table, sources, tree)
    comma := place + 1
    if ast.Syntax_Token_At(tree, comma).Kind != token.KIND_COMMA {
      return
    }
    place = comma + 1
  }
}

// Takes the candidate name, whose case stands in the slot just past the table, into the table. A
// name that is no identifier, opens with no letter, stands in its case already, or stands in the
// table already takes no entry. A name that is no constant takes its case from its first letter.
func take_name(
  places Place_Storage,
  table Table_Handle,
  sources Declaring_Source,
  tree ast.Syntax_Tree_Handle,
) {
  Place_Storage_Invariants(&places, "take_name.places")
  Table_Handle_Invariants(table, "take_name.table")
  Declaring_Source_Invariants(sources, "take_name.sources")
  ast.Syntax_Tree_Handle_Invariants(tree, "take_name.tree")
  count := places.Values[PLACE_NAMES]
  one := ast.Syntax_Token_At(tree, ast.Token_Index(places.Values[PLACE_TOKEN]))
  if one.Kind != token.KIND_IDENTIFIER {
    return
  }
  text := token.Text(token.Source(sources), one)
  // A blank identifier discards a value; it has no name to normalize.
  if string(text) == "_" {
    return
  }
  want := table.Names.Values[count].Case
  // A constant and a struct field each take the case their declaration states, thus the
  // first letter of the name states the case of every other name alone.
  if want == CASE_SNAKE {
    if text[0]-'a' < LETTER_COUNT {
      want = CASE_SNAKE
    } else if text[0]-'A' < LETTER_COUNT {
      want = CASE_ADA
    } else {
      return
    }
  }
  table.Names.Values[count].Case = want
  probe := Probe(text)
  if bool(in_case(places, table, probe)) {
    return
  }
  find_slot(places, table, probe)
  if places.Values[PLACE_FOUND] != count {
    return
  }
  start := places.Values[PLACE_TEXT]
  // A full table drops the name and counts it, because a name the table never holds is a name
  // no format renames, and the caller must learn the form it reads is no canonical form.
  if int(count) == NAME_COUNT_MAXIMUM {
    places.Values[PLACE_DROPPED] = places.Values[PLACE_DROPPED] + 1
    return
  }
  if int(start)+len(text) > NAME_TEXT_SIZE_MAXIMUM {
    places.Values[PLACE_DROPPED] = places.Values[PLACE_DROPPED] + 1
    return
  }
  copy(table.Texts.Values[start:], text)
  table.Names.Values[count] = Name{
    Text: Span{Start: start, Size: Text_Size(len(text))}, Case: want,
  }
  table.Slots.Values[places.Values[PLACE_SLOT]] = Slot(count + 1)
  places.Values[PLACE_NAMES] = count + 1
  places.Values[PLACE_TEXT] = start + Place(len(text))
}

// Reads the entry of the table whose text the probe holds into the found place, or the count of
// names when none holds it, and the slot the probe stops at into the slot place. The index holds
// at most half its slots, thus a probe meets an empty slot before it walks the whole index.
func find_slot(places Place_Storage, table Table_Handle, probe Probe) {
  Place_Storage_Invariants(&places, "find_slot.places")
  Table_Handle_Invariants(table, "find_slot.table")
  Probe_Invariants(probe, "find_slot.probe")
  hash := uint32(HASH_OFFSET)
  for _, value := range probe {
    hash = (hash ^ uint32(value)) * HASH_PRIME
  }
  slot := hash & (SLOT_COUNT - 1)
  for range SLOT_COUNT {
    held := table.Slots.Values[slot]
    if held == SLOT_EMPTY {
      break
    }
    entry := table.Names.Values[held-1]
    end := entry.Text.Start + Place(entry.Text.Size)
    text := table.Texts.Values[entry.Text.Start:end]
    if string(text) == string(probe) {
      places.Values[PLACE_FOUND] = Place(held - 1)
      return
    }
    slot = (slot + 1) & (SLOT_COUNT - 1)
  }
  aver.Always(table.Slots.Values[slot] == SLOT_EMPTY, "A probe meets an empty slot.")
  places.Values[PLACE_FOUND] = places.Values[PLACE_NAMES]
  places.Values[PLACE_SLOT] = Place(slot)
}

// Reports whether the probe stands in the case of the candidate, which is the pattern the linter
// matches. A doubled or closing underscore states an empty segment, which no case admits.
func in_case(places Place_Storage, table Table_Handle, probe Probe) (yes printer.Boolean) {
  defer func() { printer.Boolean_Invariants(yes, "in_case.yes") }()
  Place_Storage_Invariants(&places, "in_case.places")
  Table_Handle_Invariants(table, "in_case.table")
  Probe_Invariants(probe, "in_case.probe")
  text := probe
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
    places.Values[PLACE_START] = Place(start)
    places.Values[PLACE_END] = Place(index)
    if !bool(in_segment(places, table, probe)) {
      return false
    }
    start = index + 1
  }
  return true
}

// Reports whether one segment of the candidate name between underscores stands in its case. An
// Ada_Case segment is a capitalized word, an acronym, an acronym plural, or a number that follows
// the first segment, because a capacity names its own segment and Go opens a name on a letter.
func in_segment(places Place_Storage, table Table_Handle, probe Probe) (yes printer.Boolean) {
  defer func() { printer.Boolean_Invariants(yes, "in_segment.yes") }()
  Place_Storage_Invariants(&places, "in_segment.places")
  Table_Handle_Invariants(table, "in_segment.table")
  Probe_Invariants(probe, "in_segment.probe")
  entry := table.Names.Values[places.Values[PLACE_NAMES]]
  text := probe
  start := int(places.Values[PLACE_START])
  segment := text[start:places.Values[PLACE_END]]
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
func rename(places Place_Storage, scanned Name_Scan, table Table_Handle) (fit printer.Boolean) {
  defer func() { printer.Boolean_Invariants(fit, "rename.fit") }()
  Place_Storage_Invariants(&places, "rename.places")
  Name_Scan_Invariants(scanned, "rename.scanned")
  Table_Handle_Invariants(table, "rename.table")
  form := scanned
  fit = true
  places.Values[PLACE_CURSOR] = PLACE_MINIMUM
  // The clause that names the package names no declaration the table holds, thus a field
  // or a value of that spelling renames the declaration it names and never the clause.
  for places.Values[PLACE_CURSOR] < places.Values[PLACE_LIMIT] {
    stood := bool(states_head(places, Scan(scanned), PACKAGE_HEAD))
    close_place(places, Scan(scanned))
    if stood {
      break
    }
  }
  for places.Values[PLACE_CURSOR] < places.Values[PLACE_LIMIT] {
    take_word(places, scanned)
    place := places.Values[PLACE_WORD]
    end := places.Values[PLACE_WORD_END]
    if end == place {
      skip_inert(places, scanned)
      continue
    }
    places.Values[PLACE_CURSOR] = end
    // A word wider than the widest name a source spells names no entry of the
    // table, thus the search never reads one.
    if int(end-place) > WORD_SIZE_MAXIMUM {
      continue
    }
    probe := Probe(form[place:end])
    find_slot(places, table, probe)
    if places.Values[PLACE_FOUND] == places.Values[PLACE_NAMES] {
      continue
    }
    // A spell at the end of the storage writes no byte, thus the first spell measures
    // the name and the second writes it behind the bytes the measure made room for.
    places.Values[PLACE_WORD] = Place(len(form))
    spell(places, scanned, table)
    size := Place(places.Values[PLACE_SIZE])
    limit := places.Values[PLACE_LIMIT]
    if int(place+size) < len(form) {
      copy(form[place+size:], form[end:limit])
    }
    places.Values[PLACE_WORD] = place
    spell(places, scanned, table)
    limit = limit + size - (end - place)
    if int(limit) > len(form) {
      fit = false
      limit = Place(len(form))
    }
    places.Values[PLACE_LIMIT] = limit
    places.Values[PLACE_CURSOR] = place + size
  }
  return fit
}

// Reads the identifier the cursor stands on. A cursor on a byte that opens no identifier reads an
// empty one, which the caller tells by the end standing on the opening.
func take_word(places Place_Storage, scanned Name_Scan) {
  Place_Storage_Invariants(&places, "take_word.places")
  Name_Scan_Invariants(scanned, "take_word.scanned")
  form := scanned
  cursor := places.Values[PLACE_CURSOR]
  limit := places.Values[PLACE_LIMIT]
  places.Values[PLACE_WORD] = cursor
  places.Values[PLACE_WORD_END] = cursor
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
  places.Values[PLACE_WORD_END] = end
}

// Moves the cursor past the unit of the form that names nothing: a note, a string, a character, a
// number, or one byte of any other kind. A spelling inside one of them names nothing, and a unit
// the form never closed runs to the limit, because a refused print writes no closing mark.
func skip_inert(places Place_Storage, scanned Name_Scan) {
  Place_Storage_Invariants(&places, "skip_inert.places")
  Name_Scan_Invariants(scanned, "skip_inert.scanned")
  form := scanned[:places.Values[PLACE_LIMIT]]
  place := int(places.Values[PLACE_CURSOR])
  value := form[place]
  end := place + 1
  places.Values[PLACE_CURSOR] = Place(end)
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
    skip_number(places, scanned)
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
  places.Values[PLACE_CURSOR] = Place(min(end, len(form)))
}

// Moves the cursor past the number it stands just behind the opening of. A number opens on a
// digit, or on a point a digit follows, and runs over every letter, digit, underscore, and point
// behind it, thus no identifier opens inside it.
func skip_number(places Place_Storage, scanned Name_Scan) {
  Place_Storage_Invariants(&places, "skip_number.places")
  Name_Scan_Invariants(scanned, "skip_number.scanned")
  form := scanned[:places.Values[PLACE_LIMIT]]
  end := int(places.Values[PLACE_CURSOR])
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
  places.Values[PLACE_CURSOR] = Place(end)
}

// Writes the found name in the case its entry takes at the word place, and reads the size of the
// spelling into the size place. A byte the storage holds no room for takes no write. An Ada_Case
// word that holds no lowercase letter stands as written, because an acronym keeps its capitals.
func spell(places Place_Storage, scanned Name_Scan, table Table_Handle) {
  Place_Storage_Invariants(&places, "spell.places")
  Name_Scan_Invariants(scanned, "spell.scanned")
  Table_Handle_Invariants(table, "spell.table")
  form := scanned
  place := int(places.Values[PLACE_WORD])
  entry := table.Names.Values[places.Values[PLACE_FOUND]]
  text := table.Texts.Values[entry.Text.Start : entry.Text.Start+Place(entry.Text.Size)]
  // The table took the name, thus it holds two letters at least. The guard states
  // what the table already proved.
  if len(text) < RENAMED_WORD_SIZE_MINIMUM {
    return
  }
  word := Spelled(text)
  size := 0
  for start := 0; start < len(text); start = int(places.Values[PLACE_END]) {
    places.Values[PLACE_START] = Place(start)
    word_end(places, table, word)
    end := int(places.Values[PLACE_END])
    if end == start {
      places.Values[PLACE_END] = Place(start + 1)
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
  places.Values[PLACE_SIZE] = Place(size)
}

// Reads the byte just past the word of the probe that opens at the start place into the end place.
// Words stand where the linter splits them: at an underscore, at an uppercase letter behind a
// lowercase letter or a digit, and at the last capital of a run a lowercase letter follows. An
// underscore at the start states an empty word.
func word_end(places Place_Storage, table Table_Handle, word Spelled) {
  Place_Storage_Invariants(&places, "word_end.places")
  Table_Handle_Invariants(table, "word_end.table")
  Spelled_Invariants(word, "word_end.word")
  text := word
  start := int(places.Values[PLACE_START])
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
  places.Values[PLACE_END] = Place(end)
}

// MODULE_HEAD is the head every path of this module opens with. A path that opens with it names
// a package the module itself states, thus it stands in the run between the standard library and
// every other module.
const MODULE_HEAD = "local/james-orcales/"

// PATH_KIND_STANDARD names a path of the standard library, which opens no element on a dot.
const PATH_KIND_STANDARD = 0

// PATH_KIND_MODULE names a path this module itself states.
const PATH_KIND_MODULE = 1

// PATH_KIND_OTHER names a path of another module, which opens on a host name and so holds a dot.
const PATH_KIND_OTHER = 2

// Path_Kind names which of the three runs one import stands in.
type Path_Kind uint8

// Path_Kind_Invariants states every run one import stands in.
func Path_Kind_Invariants(value Path_Kind, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_3_Uint8(
    uint8(value),
    PATH_KIND_STANDARD,
    PATH_KIND_MODULE,
    PATH_KIND_OTHER,
  ).
    Ensure()
}

// Reads which run the path standing at the cursor belongs to. The cursor stands on the mark the
// path opens with, thus the read steps past that mark and reads the path itself.
func path_kind(places Place_Storage, compared Compare_Scan) (kind Path_Kind) {
  defer func() { Path_Kind_Invariants(kind, "path_kind.kind") }()
  Place_Storage_Invariants(&places, "path_kind.places")
  Compare_Scan_Invariants(compared, "path_kind.compared")
  form := compared
  head := int(places.Values[PLACE_CURSOR]) + 1
  limit := int(places.Values[PLACE_LIMIT])
  if !(head+len(MODULE_HEAD) > limit) {
    if string(form[head:head+len(MODULE_HEAD)]) == MODULE_HEAD {
      return PATH_KIND_MODULE
    }
  }
  // A path of another module opens on the host that states it, thus a dot stands ahead of
  // the first slash it spells.
  for place := head; place < limit; place++ {
    if form[place] == '"' {
      break
    }
    if form[place] == '/' {
      break
    }
    if form[place] == '.' {
      return PATH_KIND_OTHER
    }
  }
  return PATH_KIND_STANDARD
}

// Parts the runs of the sorted imports, opening one empty line where the run the paths stand in
// changes. The rule states the runs, thus the form holds one line between them however the author
// wrote the file.
func part_runs(places Place_Storage, scanned Run_Scan) {
  Place_Storage_Invariants(&places, "part_runs.places")
  Run_Scan_Invariants(scanned, "part_runs.scanned")
  // A form too short for two imports holds one run and parts nothing. The guard states
  // what the width of the storage already proved.
  if len(scanned) < COMPARE_SCAN_SIZE_MINIMUM {
    return
  }
  form := scanned
  places.Values[PLACE_SCAN] = places.Values[PLACE_HEAD]
  held := Path_Kind(PATH_KIND_STANDARD)
  first := printer.Boolean(true)
  for places.Values[PLACE_SCAN] < places.Values[PLACE_TAIL] {
    start := int(places.Values[PLACE_SCAN])
    if !bool(take_record(places, Scan(scanned))) {
      return
    }
    places.Values[PLACE_CURSOR] = places.Values[PLACE_SCAN_END]
    take_path(places, Compare_Scan(scanned))
    kind := path_kind(places, Compare_Scan(scanned))
    places.Values[PLACE_SCAN] = places.Values[PLACE_SCAN_END]
    parts := printer.Boolean(kind != held)
    held = kind
    if bool(first) {
      parts = false
    }
    first = false
    if !bool(parts) {
      continue
    }
    // The run this record opens stands one empty line behind the run ahead of it.
    // A form that holds no room for the line stands as it is, because a form past
    // the storage is no canonical form and the caller hears of it.
    limit := int(places.Values[PLACE_LIMIT])
    if limit+1 > len(form) {
      return
    }
    copy(form[start+1:limit+1], form[start:limit])
    form[start] = '\n'
    places.Values[PLACE_LIMIT] = Place(limit + 1)
    places.Values[PLACE_TAIL] = places.Values[PLACE_TAIL] + 1
    places.Values[PLACE_SCAN] = places.Values[PLACE_SCAN] + 1
  }
}
