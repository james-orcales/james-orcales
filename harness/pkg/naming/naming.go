// Package naming checks the names one parsed Go file declares against the rules lint states of
// them: the word table of harness.json, the quantity, arithmetic, participle, and extremum rules,
// and the import aliases. It reads no file: the caller parses the file and hands over the tree, the
// source, the path, and the word table, and Check states the first name that breaks.
package naming

import "local/james-orcales/harness/pkg/configuration"
import "local/james-orcales/harness/pkg/requirements"
import "local/james-orcales/shared/filepath"
import "local/james-orcales/shared/go/ast"
import "local/james-orcales/shared/go/token"
import "local/james-orcales/shared/sim/aver/default"

// SOURCE_STORAGE_SIZE is the store a source stands in: one byte past the widest source.
const SOURCE_STORAGE_SIZE = requirements.SOURCE_SIZE_MAXIMUM + 1

// PATH_SIZE_MAXIMUM is the store the path of the file stands in.
const PATH_SIZE_MAXIMUM = filepath.PATH_SIZE_MAXIMUM

// MESSAGE_SIZE_MAXIMUM is the store a message stands in, closed by a zero byte.
const MESSAGE_SIZE_MAXIMUM = requirements.MESSAGE_SIZE_MAXIMUM

// WORD_SLOT_COUNT is the slots of the word table: twice the values one configuration holds, a
// power of two, thus a probe finds an empty slot.
const WORD_SLOT_COUNT = 2 * configuration.VALUE_COUNT_MAXIMUM

// WORD_SLOT_MASK folds a hash into the word table.
const WORD_SLOT_MASK = WORD_SLOT_COUNT - 1

// HASH_BASIS opens the FNV-1a hash of one word.
const HASH_BASIS = 2166136261

// HASH_PRIME steps the FNV-1a hash of one word.
const HASH_PRIME = 16777619

// LETTER_COUNT is the letters of one case.
const LETTER_COUNT = 26

// DIGIT_COUNT is the decimal digits.
const DIGIT_COUNT = 10

// CASE_DISTANCE is how far a lowercase letter stands past its uppercase one.
const CASE_DISTANCE = 'a' - 'A'

// QUOTED_BYTE_ABSENT stands for a byte below a space in a message.
const QUOTED_BYTE_ABSENT = '?'

// WORD_PROBE_SIZE is the widest word a class names: the widest listed noun fits.
const WORD_PROBE_SIZE = 16

// SELECTOR_PROBE_SIZE is the widest receiver and method a stdlib quantity call names.
const SELECTOR_PROBE_SIZE = 32

// TERM_NONE is a name no quantity evidence names.
const TERM_NONE Term = 0

// TERM_INDEX is a position among elements.
const TERM_INDEX Term = 1

// TERM_COUNT is a number of elements.
const TERM_COUNT Term = 2

// TERM_SIZE is a number of bytes.
const TERM_SIZE Term = 4

// TERM_OFFSET is a position among bytes.
const TERM_OFFSET Term = 8

// TERM_ALL is every quantity at once.
const TERM_ALL Term = 15

// Term is the set of quantities evidence asks of one name, one bit each.
type Term uint8

// Term_Invariants bounds the set by the quantities.
func Term_Invariants(value Term, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint8(uint8(value), uint8(TERM_NONE), uint8(TERM_ALL)).
    Ensure()
}

// CLASS_MEASURE is a quantity word lint bans rather than names: length or len.
const CLASS_MEASURE = 16

// CLASS_MAXIMUM is the word max.
const CLASS_MAXIMUM = 32

// CLASS_MINIMUM is the word min.
const CLASS_MINIMUM = 33

// CLASS_PARTICIPLE is a word ending in ing that no listed noun is.
const CLASS_PARTICIPLE = 34

// TEXT_SOURCE is a span of the source.
const TEXT_SOURCE = 0

// TEXT_PATH is a span of the path.
const TEXT_PATH = 1

// TEXT_CONFIGURATION is a span of the decoded configuration.
const TEXT_CONFIGURATION = 2

// STYLE_SNAKE writes each word lowercase.
const STYLE_SNAKE = 0

// STYLE_ADA writes each word capitalized, an uppercase word whole.
const STYLE_ADA = 1

// PIECE_NONE replaces no word of a rename.
const PIECE_NONE = 0

// PIECE_CONFIGURATION replaces one word with a replacement of the word table.
const PIECE_CONFIGURATION = 1

// PIECE_TERM replaces one word with a quantity.
const PIECE_TERM = 2

// PIECE_MOVE moves one word to the end.
const PIECE_MOVE = 3

// NAME_DECLARED is a declared name, which every rule reads.
const NAME_DECLARED = 0

// NAME_PACKAGE is the package name, which only the word table reads.
const NAME_PACKAGE = 1

// CELL_FOUND is 1 once the check states a finding.
const CELL_FOUND = 0

// CELL_OFFSET is the byte the finding stands at.
const CELL_OFFSET = 1

// CELL_PATH_SIZE is how many bytes of the path store the caller filled.
const CELL_PATH_SIZE = 2

// CELL_NODE is the node the walk stands on.
const CELL_NODE = 3

// CELL_ROOT is the node whose subtree the walk stays inside.
const CELL_ROOT = 4

// CELL_WRITTEN is how many bytes of the message stand written.
const CELL_WRITTEN = 5

// CELL_LOW is the first node the terms store marked.
const CELL_LOW = 6

// CELL_HIGH is one past the last node the terms store marked, or 0 for none.
const CELL_HIGH = 7

// CELL_NAME_START is the first byte of the name the rules read.
const CELL_NAME_START = 8

// CELL_NAME_END is one past the last byte of the name.
const CELL_NAME_END = 9

// CELL_NAME_TEXT is the text the name stands in.
const CELL_NAME_TEXT = 10

// CELL_NAME_KIND is which rules read the name.
const CELL_NAME_KIND = 11

// CELL_STYLE is the case a rename of the name takes.
const CELL_STYLE = 12

// CELL_TEXT is the text the word cursor reads.
const CELL_TEXT = 13

// CELL_CURSOR is the byte the word cursor stands on.
const CELL_CURSOR = 14

// CELL_LIMIT is one past the last byte the word cursor reads.
const CELL_LIMIT = 15

// CELL_WORD_START is the first byte of the word the cursor read.
const CELL_WORD_START = 16

// CELL_WORD_END is one past the last byte of the word, equal to the start past the last word.
const CELL_WORD_END = 17

// CELL_SLOT is the value slot of the table word the word is, or configuration.WORD_NONE.
const CELL_SLOT = 18

// CELL_CLASS is the class of the word.
const CELL_CLASS = 19

// CELL_SUFFIX is the quantity the last word of the name is, or TERM_NONE.
const CELL_SUFFIX = 20

// CELL_REPLACED is the word a rename replaces or moves, or -1.
const CELL_REPLACED = 21

// CELL_PIECE is what stands in the replaced word's place.
const CELL_PIECE = 22

// CELL_PIECE_START is the first byte of a replacement.
const CELL_PIECE_START = 23

// CELL_PIECE_END is one past the last byte of a replacement.
const CELL_PIECE_END = 24

// CELL_TERM is the quantity a rename writes or a mark sets.
const CELL_TERM = 25

// CELL_APPENDED is the quantity a rename appends, or TERM_NONE.
const CELL_APPENDED = 26

// CELL_JOINED is how many words a rename wrote.
const CELL_JOINED = 27

// CELL_MARKED is the node a mark sets its quantity on.
const CELL_MARKED = 28

// CELL_LEFT is the quantity the left operand of a sum ends in.
const CELL_LEFT = 29

// CELL_RIGHT is the quantity the right operand of a sum ends in.
const CELL_RIGHT = 30

// CELL_RESULT is the quantity lint's table gives a sum, or TERM_NONE.
const CELL_RESULT = 31

// CELL_SIGN is the token kind of the sum's operator.
const CELL_SIGN = 32

// CELL_BINARY is the sum the operands are read from.
const CELL_BINARY = 33

// CELL_PACKAGE_START is the first byte of the package an import names.
const CELL_PACKAGE_START = 34

// CELL_PACKAGE_END is one past the last byte of the package.
const CELL_PACKAGE_END = 35

// CELL_REPLACEMENTS is how many replacements the table word the word is holds.
const CELL_REPLACEMENTS = 36

// CELL_RECORD_COUNT is how many names the record table holds.
const CELL_RECORD_COUNT = 37

// CELL_LOCAL_COUNT is how many of the recorded names a function body declares, standing first.
const CELL_LOCAL_COUNT = 38

// CELL_FRAME is the scope the shadow walk stands in.
const CELL_FRAME = 39

// CELL_NAME_COUNT is how many names the scopes of the shadow walk hold.
const CELL_NAME_COUNT = 40

// CELL_CONTAINER is the node whose names the shadow walk takes.
const CELL_CONTAINER = 41

// CELL_JUDGED is 1 while the names the walk takes are judged and recorded, 0 while they are only
// held for the scopes inside them.
const CELL_JUDGED = 42

// CELL_DECLARED is the name the shadow walk takes.
const CELL_DECLARED = 43

// CELL_SHADOW is 1 when the finding is a shadow.
const CELL_SHADOW = 44

// CELL_GRAPH_START is the first recorded name of the call graph, past the top-level names.
const CELL_GRAPH_START = 45

// CELL_IMPORT_START is the first recorded import, past the call graph.
const CELL_IMPORT_START = 46

// CELL_SOFT_CALL is the name of the first selector of a clock, stdout, or network API, or none.
const CELL_SOFT_CALL = 47

// CELL_SOFT_IMPORT is the path of the import that selector reads from.
const CELL_SOFT_IMPORT = 48

// CELL_TRANSITIVE_CALL is the name of the first selector of an API that reaches one, or none.
const CELL_TRANSITIVE_CALL = 49

// CELL_TRANSITIVE_IMPORT is the path of the import that selector reads from.
const CELL_TRANSITIVE_IMPORT = 50

// CELL_MAIN_COUNT is how many free functions named Main the file declares.
const CELL_MAIN_COUNT = 51

// CELL_COUNT is how many slots the state holds.
const CELL_COUNT = 52

// IMPURITY_PROBE_SIZE is the widest import path and name the impurity lists name, joined.
const IMPURITY_PROBE_SIZE = 32

// RECORD_COUNT_MAXIMUM bounds the names Record lists: each node once, and each function name a
// second time in the call graph.
const RECORD_COUNT_MAXIMUM = 2 * ast.NODE_COUNT_MAXIMUM

// FRAME_COUNT is the scopes one walk holds at once: the file, the function, and one for each depth
// the parse holds.
const FRAME_COUNT = ast.DEPTH_MAXIMUM + 2

// HASH_WIDE_BASIS opens the 64-bit FNV-1a hash of one name.
const HASH_WIDE_BASIS = 14695981039346656037

// HASH_WIDE_PRIME steps the 64-bit FNV-1a hash of one name.
const HASH_WIDE_PRIME = 1099511628211

// NAME_HASH_MINIMUM is the least hash of a name.
const NAME_HASH_MINIMUM Name_Hash = 0

// NAME_HASH_MAXIMUM is the most hash of a name.
const NAME_HASH_MAXIMUM Name_Hash = 1<<64 - 1

// Name_Hash is the 64-bit FNV-1a hash of one name, which a scope compares before its bytes.
type Name_Hash uint64

// Name_Hash_Invariants bounds the hash by its width.
func Name_Hash_Invariants(value Name_Hash, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint64(
    uint64(value),
    uint64(NAME_HASH_MINIMUM),
    uint64(NAME_HASH_MAXIMUM),
  ).
    Ensure()
}

// CELL_MINIMUM is the least a slot holds: a word slot that names no word.
const CELL_MINIMUM = -1

// CELL_MAXIMUM is the most a slot holds: an offset of the source store.
const CELL_MAXIMUM = SOURCE_STORAGE_SIZE

// State_Cell is one slot of the check's state.
type State_Cell int

// State_Cell_Invariants bounds the slot by what one slot holds.
func State_Cell_Invariants(value State_Cell, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(int(value), CELL_MINIMUM, CELL_MAXIMUM).
    Ensure()
}

// State_Cells holds the check's state, one slot each.
type State_Cells []State_Cell

// State_Cells_Invariants fixes the store.
func State_Cells_Invariants(value State_Cells, _ aver.Namespace) {
  aver.Always(len(value) == CELL_COUNT, "A naming number store has complete length.")
  aver.Always(cap(value) == CELL_COUNT, "A naming number store cannot grow.")
}

// Source_Store is the store the file stands in.
type Source_Store []byte

// Source_Store_Invariants fixes the store.
func Source_Store_Invariants(value Source_Store, _ aver.Namespace) {
  aver.Always(len(value) == SOURCE_STORAGE_SIZE, "A naming source store has complete length.")
  aver.Always(cap(value) == SOURCE_STORAGE_SIZE, "A naming source store cannot grow.")
}

// Path_Store is the store the path of the file stands in.
type Path_Store []byte

// Path_Store_Invariants fixes the store.
func Path_Store_Invariants(value Path_Store, _ aver.Namespace) {
  aver.Always(len(value) == PATH_SIZE_MAXIMUM, "A naming path store has complete length.")
  aver.Always(cap(value) == PATH_SIZE_MAXIMUM, "A naming path store cannot grow.")
}

// Word_Entry is one slot of the word table: the value slot of one word of the configuration, or
// configuration.WORD_NONE, and how many replacements stand behind it.
type Word_Entry struct {
  // Slot is the value slot of the word.
  Slot configuration.Word_Slot
  // Count is how many replacements stand behind the word, in the value slots past it.
  Count State_Cell
}

// Word_Entry_Invariants composes one slot.
func Word_Entry_Invariants(value *Word_Entry, namespace aver.Namespace) {
  configuration.Word_Slot_Invariants(value.Slot, namespace)
  State_Cell_Invariants(value.Count, namespace)
}

// Word_Table is the word table.
type Word_Table []Word_Entry

// Word_Table_Invariants fixes the store.
func Word_Table_Invariants(value Word_Table, _ aver.Namespace) {
  aver.Always(len(value) == WORD_SLOT_COUNT, "A word table has complete length.")
  aver.Always(cap(value) == WORD_SLOT_COUNT, "A word table cannot grow.")
}

// Terms holds the quantities evidence asks of each node.
type Terms []Term

// Terms_Invariants fixes the store.
func Terms_Invariants(value Terms, _ aver.Namespace) {
  aver.Always(len(value) == ast.NODE_COUNT_MAXIMUM, "A term store has complete length.")
  aver.Always(cap(value) == ast.NODE_COUNT_MAXIMUM, "A term store cannot grow.")
}

// Message_Store is the store the message of a finding stands in, closed by a zero byte.
type Message_Store []byte

// Message_Store_Invariants fixes the store.
func Message_Store_Invariants(value Message_Store, _ aver.Namespace) {
  aver.Always(len(value) == MESSAGE_SIZE_MAXIMUM, "A naming message store has complete length.")
  aver.Always(cap(value) == MESSAGE_SIZE_MAXIMUM, "A naming message store cannot grow.")
}

// Node_View views the node arena of the parse of the file.
type Node_View []ast.Node

// Node_View_Invariants fixes the store.
func Node_View_Invariants(value Node_View, _ aver.Namespace) {
  aver.Always(len(value) == ast.NODE_COUNT_MAXIMUM, "A naming node view has complete length.")
  aver.Always(cap(value) == ast.NODE_COUNT_MAXIMUM, "A naming node view cannot grow.")
}

// Token_View views the token run of the parse of the file.
type Token_View []token.Token

// Token_View_Invariants fixes the store.
func Token_View_Invariants(value Token_View, _ aver.Namespace) {
  aver.Always(len(value) == ast.TOKEN_COUNT_MAXIMUM, "A naming token view has complete length.")
  aver.Always(cap(value) == ast.TOKEN_COUNT_MAXIMUM, "A naming token view cannot grow.")
}

// Decoded_Values views the value table of the decoded configuration.
type Decoded_Values []configuration.Value

// Decoded_Values_Invariants fixes the store.
func Decoded_Values_Invariants(value Decoded_Values, _ aver.Namespace) {
  aver.Always(
    len(value) == configuration.VALUE_COUNT_MAXIMUM,
    "A naming value view has complete length.",
  )
  aver.Always(cap(value) == configuration.VALUE_COUNT_MAXIMUM, "A naming value view cannot grow.")
}

// Decoded_Text views the text store of the decoded configuration.
type Decoded_Text []byte

// Decoded_Text_Invariants fixes the store.
func Decoded_Text_Invariants(value Decoded_Text, _ aver.Namespace) {
  aver.Always(
    len(value) == configuration.SOURCE_SIZE_MAXIMUM,
    "A naming text view has complete length.",
  )
  aver.Always(cap(value) == configuration.SOURCE_SIZE_MAXIMUM, "A naming text view cannot grow.")
}

// Record_Table holds the names of one file Record lists, one node each.
type Record_Table []ast.Index

// Record_Table_Invariants fixes the store.
func Record_Table_Invariants(value Record_Table, _ aver.Namespace) {
  aver.Always(len(value) == RECORD_COUNT_MAXIMUM, "A naming record table has complete length.")
  aver.Always(cap(value) == RECORD_COUNT_MAXIMUM, "A naming record table cannot grow.")
}

// Scope_Name is one name a scope of the shadow walk declares.
type Scope_Name struct {
  // Node is the name.
  Node ast.Index
  // Hash is the hash of its bytes.
  Hash Name_Hash
}

// Scope_Name_Invariants composes one name.
func Scope_Name_Invariants(value *Scope_Name, namespace aver.Namespace) {
  ast.Index_Invariants(value.Node, namespace)
  Name_Hash_Invariants(value.Hash, namespace)
}

// Scope_Names holds the names the scopes of the shadow walk declare.
type Scope_Names []Scope_Name

// Scope_Names_Invariants fixes the store.
func Scope_Names_Invariants(value Scope_Names, _ aver.Namespace) {
  aver.Always(len(value) == ast.NODE_COUNT_MAXIMUM, "A naming scope store has complete length.")
  aver.Always(cap(value) == ast.NODE_COUNT_MAXIMUM, "A naming scope store cannot grow.")
}

// Scope_Frame is one scope of the shadow walk: the statement it reads next, and the first of the
// names it declares.
type Scope_Frame struct {
  // Cursor is the statement the scope reads next, or ast.INDEX_ABSENT past its last.
  Cursor ast.Index
  // Start is the first name the scope declares.
  Start State_Cell
}

// Scope_Frame_Invariants composes one scope.
func Scope_Frame_Invariants(value *Scope_Frame, namespace aver.Namespace) {
  ast.Index_Invariants(value.Cursor, namespace)
  State_Cell_Invariants(value.Start, namespace)
}

// Frame_Stack holds the scopes of the shadow walk.
type Frame_Stack []Scope_Frame

// Frame_Stack_Invariants fixes the store.
func Frame_Stack_Invariants(value Frame_Stack, _ aver.Namespace) {
  aver.Always(len(value) == FRAME_COUNT, "A naming frame stack has complete length.")
  aver.Always(cap(value) == FRAME_COUNT, "A naming frame stack cannot grow.")
}

// Checker holds the stores one check reads and writes, each the caller's own. The tree and the
// configuration stand as views of their stores, which the caller's own chains state.
type Checker struct {
  // Nodes views the node arena of the parse of the file.
  Nodes Node_View
  // Tokens views the token run of the parse of the file.
  Tokens Token_View
  // Source holds the file the tree was parsed from.
  Source Source_Store
  // Path holds the path of the file, CELL_PATH_SIZE bytes of it.
  Path Path_Store
  // Values views the value table of the decoded configuration.
  Values Decoded_Values
  // Text views the text store of the decoded configuration.
  Text Decoded_Text
  // Words is the word table Words_Build builds from the configuration.
  Words Word_Table
  // Terms holds the quantities evidence asks of each node.
  Terms Terms
  // Message holds the message of the finding.
  Message Message_Store
  // Records holds the names Record lists.
  Records Record_Table
  // Names holds the names the scopes of the shadow walk declare, the outer scope first.
  Names Scope_Names
  // Frames holds the scopes of the shadow walk, the file first.
  Frames Frame_Stack
  // Cells holds the check's state.
  Cells State_Cells
}

// Checker_Invariants composes the stores.
func Checker_Invariants(value *Checker, namespace aver.Namespace) {
  Node_View_Invariants(value.Nodes, namespace)
  Token_View_Invariants(value.Tokens, namespace)
  Source_Store_Invariants(value.Source, namespace)
  Path_Store_Invariants(value.Path, namespace)
  Decoded_Values_Invariants(value.Values, namespace)
  Decoded_Text_Invariants(value.Text, namespace)
  Word_Table_Invariants(value.Words, namespace)
  Terms_Invariants(value.Terms, namespace)
  Message_Store_Invariants(value.Message, namespace)
  Record_Table_Invariants(value.Records, namespace)
  Scope_Names_Invariants(value.Names, namespace)
  Frame_Stack_Invariants(value.Frames, namespace)
  State_Cells_Invariants(value.Cells, namespace)
}

// Checker_Handle is a caller-owned check.
type Checker_Handle *Checker

// Checker_Handle_Invariants composes a present check.
func Checker_Handle_Invariants(value Checker_Handle, namespace aver.Namespace) {
  if value == nil {
    return
  }
  Checker_Invariants(value, namespace)
}

// Found records that the check states a finding.
type Found bool

// Found_Invariants proves both answers occur.
func Found_Invariants(value Found, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Sometimes(bool(value), "A file breaks a naming rule.").
    Ensure()
}

// Name_Declared records that a node declares a name the rules read.
type Name_Declared bool

// Name_Declared_Invariants proves both answers occur.
func Name_Declared_Invariants(value Name_Declared, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Sometimes(bool(value), "A node declares a name.").
    Ensure()
}

// Builds the word table from the words of the configuration. A word the table holds twice keeps
// its last entry, as a decoded map keeps its last key, and the replacements of a word stand in the
// value slots past it, as the decoder writes them.
func Words_Build(words Word_Table, decoded configuration.Configuration_Handle) {
  Word_Table_Invariants(words, "words_build.words")
  configuration.Configuration_Handle_Invariants(decoded, "words_build.decoded")
  for slot := range words {
    words[slot] = Word_Entry{Slot: configuration.WORD_NONE}
  }
  values := decoded.Values[:decoded.Cells[configuration.CELL_VALUE]]
  for held, value := range values {
    if value.Key != configuration.KEY_WORD_REPLACEMENTS {
      continue
    }
    if value.Word != configuration.WORD_NONE {
      continue
    }
    word := decoded.Text[value.Start:value.End]
    hash := uint32(HASH_BASIS)
    for _, symbol := range word {
      hash = (hash ^ uint32(symbol)) * HASH_PRIME
    }
    // The table holds twice the values, thus the probe meets an empty slot or the word.
    slot := hash & WORD_SLOT_MASK
    for range WORD_SLOT_COUNT {
      entry := words[slot]
      if entry.Slot == configuration.WORD_NONE {
        break
      }
      other := decoded.Values[entry.Slot]
      if string(decoded.Text[other.Start:other.End]) == string(word) {
        break
      }
      slot = (slot + 1) & WORD_SLOT_MASK
    }
    count := 0
    for _, replacement := range values[held+1:] {
      if replacement.Word != configuration.Word_Slot(held) {
        break
      }
      count++
    }
    words[slot] = Word_Entry{Slot: configuration.Word_Slot(held), Count: State_Cell(count)}
  }
}

// Checks the file the stores hold and states whether it breaks a rule. A finding leaves the byte it
// stands at in CELL_OFFSET and its message in the message store.
func Check(checker Checker_Handle) (found Found) {
  defer func() { Found_Invariants(found, "check.found") }()
  Checker_Handle_Invariants(checker, "check.checker")
  cells := checker.Cells
  clear(checker.Terms[cells[CELL_LOW]:cells[CELL_HIGH]])
  path_size := cells[CELL_PATH_SIZE]
  clear(cells)
  cells[CELL_PATH_SIZE] = path_size
  stem_check(checker)
  cells[CELL_NODE], cells[CELL_ROOT] = State_Cell(ast.INDEX_FIRST), State_Cell(ast.INDEX_FIRST)
  for range ast.NODE_COUNT_MAXIMUM {
    if cells[CELL_FOUND] == 1 {
      break
    }
    if cells[CELL_NODE] == State_Cell(ast.INDEX_ABSENT) {
      break
    }
    node_check(checker)
    step(checker)
  }
  shadow_walk(checker)
  if cells[CELL_FOUND] == 0 {
    return false
  }
  message_close(checker)
  return true
}

// Moves the walk to the node past the one it stands on in pre-order, staying inside the subtree of
// CELL_ROOT, or to no node past it.
func step(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "step.checker")
  cells := checker.Cells
  nodes := checker.Nodes
  index := ast.Index(cells[CELL_NODE])
  if nodes[index].First_Child != ast.Head(ast.INDEX_ABSENT) {
    cells[CELL_NODE] = State_Cell(nodes[index].First_Child)
    return
  }
  for range ast.DEPTH_MAXIMUM + 1 {
    if index == ast.Index(cells[CELL_ROOT]) {
      break
    }
    if nodes[index].Next != ast.Successor(ast.INDEX_ABSENT) {
      cells[CELL_NODE] = State_Cell(nodes[index].Next)
      return
    }
    index = ast.Index(nodes[index].Parent)
  }
  cells[CELL_NODE] = State_Cell(ast.INDEX_ABSENT)
}

// Reads the file name stem, cut of .go and then _test, against the word table at byte zero.
func stem_check(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "stem_check.checker")
  cells := checker.Cells
  path := checker.Path[:cells[CELL_PATH_SIZE]]
  start := 0
  for index, symbol := range path {
    if symbol == filepath.SEPARATOR {
      start = index + 1
    }
  }
  end := int(cells[CELL_PATH_SIZE])
  for _, suffix := range [...]string{".go", "_test"} {
    if end-start < len(suffix) {
      continue
    }
    if string(path[end-len(suffix):end]) == suffix {
      end -= len(suffix)
    }
  }
  if string(path[start:end]) == "_" {
    return
  }
  cells[CELL_NAME_START], cells[CELL_NAME_END] = State_Cell(start), State_Cell(end)
  cells[CELL_NAME_TEXT], cells[CELL_OFFSET] = TEXT_PATH, 0
  style_find(checker)
  vocabulary_check(checker)
}

// Applies the rules of the node the walk stands on.
func node_check(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "node_check.checker")
  cells := checker.Cells
  switch checker.Nodes[cells[CELL_NODE]].Kind {
  case ast.NODE_IMPORT:
    alias_check(checker)
  case ast.NODE_FUNCTION:
    terms_collect(checker)
  case ast.NODE_BINARY:
    cells[CELL_BINARY] = cells[CELL_NODE]
    sum_check(checker)
  case ast.NODE_IDENTIFIER, ast.NODE_PARAMETER_NAME, ast.NODE_FIELD_NAME, ast.NODE_CONSTANT_NAME:
    if bool(declared(checker)) {
      name_check(checker)
    }
  }
}

// Reports whether the node the walk stands on declares a name lint walks, and which rules read it.
func declared(checker Checker_Handle) (held Name_Declared) {
  defer func() { Name_Declared_Invariants(held, "declared.held") }()
  Checker_Handle_Invariants(checker, "declared.checker")
  cells := checker.Cells
  nodes := checker.Nodes
  index := ast.Index(cells[CELL_NODE])
  node := nodes[index]
  parent := nodes[node.Parent]
  spelled := checker.Tokens[node.Token]
  if string(checker.Source[spelled.Offset:spelled.Offset+token.Offset(spelled.Size)]) == "_" {
    return false
  }
  cells[CELL_NAME_KIND] = NAME_DECLARED
  switch node.Kind {
  case ast.NODE_CONSTANT_NAME:
    return true
  case ast.NODE_FIELD_NAME:
    structure := nodes[parent.Parent]
    if structure.Kind != ast.NODE_STRUCTURE_TYPE {
      return false
    }
    if nodes[structure.Parent].Kind != ast.NODE_TYPE {
      return false
    }
    return ast.Index(nodes[structure.Parent].Parent) == ast.INDEX_FIRST
  case ast.NODE_PARAMETER_NAME:
    switch nodes[parent.Parent].Kind {
    case ast.NODE_FUNCTION, ast.NODE_RECEIVER:
      return true
    }
    return false
  }
  switch parent.Kind {
  case ast.NODE_FILE:
    // The package clause is the one identifier the file holds as its own, past any note.
    cells[CELL_NAME_KIND] = NAME_PACKAGE
    return true
  case ast.NODE_FUNCTION:
    return true
  case ast.NODE_TYPE:
    if ast.Index(parent.Parent) != ast.INDEX_FIRST {
      return false
    }
    return ast.Index(parent.First_Child) == index
  case ast.NODE_VARIABLE, ast.NODE_DEFINE:
    return listed(checker)
  }
  return false
}

// Reports whether the identifier the walk stands on is one of the names a var or a := lists: the
// leading identifiers, each past the first behind a comma.
func listed(checker Checker_Handle) (held Name_Declared) {
  defer func() { Name_Declared_Invariants(held, "listed.held") }()
  Checker_Handle_Invariants(checker, "listed.checker")
  nodes := checker.Nodes
  tokens := checker.Tokens
  index := ast.Index(checker.Cells[CELL_NODE])
  child := ast.Index(nodes[nodes[index].Parent].First_Child)
  for range ast.NODE_COUNT_MAXIMUM {
    node := nodes[child]
    if node.Kind == ast.NODE_COMMENT {
      child = ast.Index(node.Next)
      continue
    }
    if node.Kind != ast.NODE_IDENTIFIER {
      return false
    }
    if child == index {
      return true
    }
    // A note may stand between a name and the comma past it.
    past := node.Token + 1
    for tokens[past].Kind == token.KIND_COMMENT {
      past++
    }
    if tokens[past].Kind != token.KIND_COMMA {
      return false
    }
    child = ast.Index(node.Next)
  }
  return false
}

// Applies the rules of one declared name, in the order a finding at one name takes.
func name_check(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "name_check.checker")
  cells := checker.Cells
  index := cells[CELL_NODE]
  nodes := checker.Nodes
  spelled := checker.Tokens[nodes[index].Token]
  cells[CELL_NAME_START], cells[CELL_NAME_TEXT] = State_Cell(spelled.Offset), TEXT_SOURCE
  cells[CELL_NAME_END] = State_Cell(spelled.Offset) + State_Cell(spelled.Size)
  cells[CELL_OFFSET] = State_Cell(spelled.Offset)
  style_find(checker)
  vocabulary_check(checker)
  if cells[CELL_FOUND] == 1 {
    return
  }
  if cells[CELL_NAME_KIND] == NAME_PACKAGE {
    return
  }
  if checker.Terms[index] != TERM_NONE {
    quantity_check(checker)
  }
  if cells[CELL_FOUND] == 1 {
    return
  }
  if nodes[nodes[index].Parent].Kind == ast.NODE_DEFINE {
    result_check(checker)
  }
  if cells[CELL_FOUND] == 1 {
    return
  }
  participle_check(checker)
  if cells[CELL_FOUND] == 1 {
    return
  }
  extremum_check(checker)
}

// Sets the case a rename of the name takes: Ada_Case when it opens uppercase, else snake_case.
func style_find(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "style_find.checker")
  cells := checker.Cells
  cells[CELL_STYLE] = STYLE_SNAKE
  if cells[CELL_NAME_START] == cells[CELL_NAME_END] {
    return
  }
  text := []byte(checker.Source)
  if cells[CELL_NAME_TEXT] == TEXT_PATH {
    text = checker.Path
  }
  if text[cells[CELL_NAME_START]]-'A' < LETTER_COUNT {
    cells[CELL_STYLE] = STYLE_ADA
  }
}

// Points the word cursor at the name.
func name_words(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "name_words.checker")
  cells := checker.Cells
  cells[CELL_TEXT] = cells[CELL_NAME_TEXT]
  cells[CELL_CURSOR], cells[CELL_LIMIT] = cells[CELL_NAME_START], cells[CELL_NAME_END]
  cells[CELL_WORD_START], cells[CELL_WORD_END] = cells[CELL_NAME_START], cells[CELL_NAME_START]
}

// Reads the next word past the cursor as lint splits one: an underscore parts two words, and so
// does an uppercase letter after a lowercase letter or a digit, or before a lowercase letter
// after an uppercase one. Past the last word, the word is empty.
func word_next(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "word_next.checker")
  cells := checker.Cells
  text := []byte(checker.Source)
  switch cells[CELL_TEXT] {
  case TEXT_PATH:
    text = checker.Path
  case TEXT_CONFIGURATION:
    text = checker.Text
  }
  cursor, limit := int(cells[CELL_CURSOR]), int(cells[CELL_LIMIT])
  for cursor < limit {
    if text[cursor] != '_' {
      break
    }
    cursor++
  }
  start := cursor
  if cursor < limit {
    cursor++
  }
  for cursor < limit {
    symbol, previous := text[cursor], text[cursor-1]
    if symbol == '_' {
      break
    }
    if symbol-'A' < LETTER_COUNT {
      if previous-'a' < LETTER_COUNT {
        break
      }
      if previous-'0' < DIGIT_COUNT {
        break
      }
      if previous-'A' < LETTER_COUNT {
        if cursor+1 < limit {
          if text[cursor+1]-'a' < LETTER_COUNT {
            break
          }
        }
      }
    }
    cursor++
  }
  cells[CELL_WORD_START], cells[CELL_WORD_END] = State_Cell(start), State_Cell(cursor)
  cells[CELL_CURSOR] = State_Cell(cursor)
}

// Finds the word the cursor read in the word table, lowercase, as lint looks one up.
func word_find(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "word_find.checker")
  cells := checker.Cells
  text := []byte(checker.Source)
  if cells[CELL_TEXT] == TEXT_PATH {
    text = checker.Path
  }
  word := text[cells[CELL_WORD_START]:cells[CELL_WORD_END]]
  hash := uint32(HASH_BASIS)
  for _, symbol := range word {
    if symbol-'A' < LETTER_COUNT {
      symbol += CASE_DISTANCE
    }
    hash = (hash ^ uint32(symbol)) * HASH_PRIME
  }
  cells[CELL_SLOT] = State_Cell(configuration.WORD_NONE)
  slot := hash & WORD_SLOT_MASK
  for range WORD_SLOT_COUNT {
    entry := checker.Words[slot]
    if entry.Slot == configuration.WORD_NONE {
      return
    }
    value := checker.Values[entry.Slot]
    key := checker.Text[value.Start:value.End]
    same := len(key) == len(word)
    for offset := range min(len(key), len(word)) {
      symbol := word[offset]
      if symbol-'A' < LETTER_COUNT {
        symbol += CASE_DISTANCE
      }
      if key[offset] != symbol {
        same = false
      }
    }
    if same {
      cells[CELL_SLOT], cells[CELL_REPLACEMENTS] = State_Cell(entry.Slot), entry.Count
      return
    }
    slot = (slot + 1) & WORD_SLOT_MASK
  }
}

// Classes the word the cursor read: a quantity, a quantity word lint bans, an extremum, or a
// participle, which is a word ending in ing that no noun lint lists is.
func word_class(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "word_class.checker")
  cells := checker.Cells
  text := []byte(checker.Source)
  if cells[CELL_TEXT] == TEXT_PATH {
    text = checker.Path
  }
  word := text[cells[CELL_WORD_START]:cells[CELL_WORD_END]]
  cells[CELL_CLASS] = State_Cell(TERM_NONE)
  probe := [WORD_PROBE_SIZE]byte{}
  size := copy(probe[:], word)
  for offset, symbol := range probe[:size] {
    if symbol-'A' < LETTER_COUNT {
      probe[offset] = symbol + CASE_DISTANCE
    }
  }
  participle := false
  if len(word) > len("ing")-1 {
    participle = true
    for offset, symbol := range word[len(word)-len("ing"):] {
      if symbol-'A' < LETTER_COUNT {
        symbol += CASE_DISTANCE
      }
      if symbol != "ing"[offset] {
        participle = false
      }
    }
  }
  if len(word) > WORD_PROBE_SIZE {
    size = 0
  }
  switch string(probe[:size]) {
  case "index":
    cells[CELL_CLASS] = State_Cell(TERM_INDEX)
  case "count":
    cells[CELL_CLASS] = State_Cell(TERM_COUNT)
  case "size":
    cells[CELL_CLASS] = State_Cell(TERM_SIZE)
  case "offset":
    cells[CELL_CLASS] = State_Cell(TERM_OFFSET)
  case "length", "len":
    cells[CELL_CLASS] = CLASS_MEASURE
  case "max":
    cells[CELL_CLASS] = CLASS_MAXIMUM
  case "min":
    cells[CELL_CLASS] = CLASS_MINIMUM
  case "string", "ring", "thing", "king", "wing", "sibling", "darling", "herring", "awning",
    "morning", "evening", "ceiling", "lightning", "building", "dwelling", "housing", "opening",
    "landing", "crossing", "parking", "lining", "casing", "padding", "packaging", "clothing",
    "bedding", "plumbing", "wiring", "tubing", "lighting", "icing", "dressing", "stuffing",
    "frosting", "topping", "filling", "coating", "seasoning", "helping", "serving", "offering",
    "meaning", "warning", "greeting", "blessing", "heading", "ending", "beginning", "finding",
    "reading", "saying", "feeling", "hearing", "meeting", "gathering", "briefing", "screening",
    "sighting", "posting", "listing", "mapping", "binding", "encoding", "setting", "grouping",
    "ordering", "pairing", "spacing", "timing", "sizing", "drawing", "painting", "carving",
    "etching", "engraving", "recording":
    return
  }
  if participle {
    cells[CELL_CLASS] = CLASS_PARTICIPLE
  }
}

// Moves the word cursor onto the last word of the name and classes it, and states the quantity it
// is in CELL_SUFFIX, or TERM_NONE.
func last_word(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "last_word.checker")
  cells := checker.Cells
  name_words(checker)
  start, end := cells[CELL_WORD_START], cells[CELL_WORD_END]
  for range int(cells[CELL_NAME_END]-cells[CELL_NAME_START]) + 1 {
    word_next(checker)
    if cells[CELL_WORD_START] == cells[CELL_WORD_END] {
      break
    }
    start, end = cells[CELL_WORD_START], cells[CELL_WORD_END]
  }
  cells[CELL_WORD_START], cells[CELL_WORD_END] = start, end
  word_class(checker)
  cells[CELL_SUFFIX] = State_Cell(TERM_NONE)
  if cells[CELL_CLASS] < CLASS_MEASURE {
    cells[CELL_SUFFIX] = cells[CELL_CLASS]
  }
}

// States the first word of the name the word table holds: its replacements, or its ban.
func vocabulary_check(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "vocabulary_check.checker")
  cells := checker.Cells
  name_words(checker)
  for ordinal := range int(cells[CELL_NAME_END]-cells[CELL_NAME_START]) + 1 {
    word_next(checker)
    if cells[CELL_WORD_START] == cells[CELL_WORD_END] {
      return
    }
    word_find(checker)
    if cells[CELL_SLOT] == State_Cell(configuration.WORD_NONE) {
      continue
    }
    cells[CELL_REPLACED], cells[CELL_PIECE] = State_Cell(ordinal), PIECE_CONFIGURATION
    vocabulary_message(checker)
    return
  }
}

// Writes the message of the table word the cursor stands on: its ban when the table holds no
// replacement for it, else a rename for each replacement.
func vocabulary_message(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "vocabulary_message.checker")
  cells := checker.Cells
  count := int(cells[CELL_REPLACEMENTS])
  values := checker.Values[cells[CELL_SLOT]+1 : int(cells[CELL_SLOT])+1+count]
  cells[CELL_FOUND] = 1
  message := checker.Message
  if count == 0 {
    cells[CELL_WRITTEN] = State_Cell(copy(message, "The identifier \""))
    name_append(checker)
    size := int(cells[CELL_WRITTEN])
    cells[CELL_WRITTEN] = State_Cell(
      size + copy(message[size:], "\" contains the banned substring \""),
    )
    lower_write(checker)
    size = int(cells[CELL_WRITTEN])
    cells[CELL_WRITTEN] = State_Cell(size + copy(message[size:], "\". Write a different name."))
    return
  }
  cells[CELL_WRITTEN] = State_Cell(copy(message, "Rename "))
  name_append(checker)
  size := int(cells[CELL_WRITTEN])
  size += copy(message[size:], " -> ")
  if count > 1 {
    size += copy(message[size:], "[")
  }
  written := 0
  for _, value := range values {
    if written > 0 {
      size += copy(message[size:], ", ")
    }
    written++
    cells[CELL_WRITTEN] = State_Cell(size)
    cells[CELL_PIECE_START], cells[CELL_PIECE_END] = State_Cell(value.Start), State_Cell(value.End)
    rename_write(checker)
    size = int(cells[CELL_WRITTEN])
  }
  if count > 1 {
    size += copy(message[size:], "]")
  }
  cells[CELL_WRITTEN] = State_Cell(size + copy(message[size:], "."))
}

// States a name whose quantity evidence asks a suffix it does not end in.
func quantity_check(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "quantity_check.checker")
  cells := checker.Cells
  terms := checker.Terms[cells[CELL_NODE]]
  last_word(checker)
  if Term(cells[CELL_SUFFIX])&terms != TERM_NONE {
    return
  }
  preferred := TERM_NONE
  for _, term := range [...]Term{TERM_OFFSET, TERM_SIZE, TERM_COUNT, TERM_INDEX} {
    if terms&term != TERM_NONE {
      preferred = term
    }
  }
  // The rename takes the preferred quantity in place of the first quantity word, or appends it.
  cells[CELL_REPLACED], cells[CELL_PIECE], cells[CELL_APPENDED] = -1, PIECE_TERM, State_Cell(
    preferred,
  )
  name_words(checker)
  for ordinal := range int(cells[CELL_NAME_END]-cells[CELL_NAME_START]) + 1 {
    word_next(checker)
    if cells[CELL_WORD_START] == cells[CELL_WORD_END] {
      break
    }
    word_class(checker)
    if cells[CELL_CLASS] == State_Cell(TERM_NONE) {
      continue
    }
    if cells[CELL_CLASS] > CLASS_MEASURE {
      continue
    }
    cells[CELL_REPLACED], cells[CELL_APPENDED] = State_Cell(ordinal), State_Cell(TERM_NONE)
    break
  }
  cells[CELL_FOUND] = 1
  message := checker.Message
  cells[CELL_WRITTEN] = State_Cell(copy(message, "The name "))
  name_append(checker)
  size := int(cells[CELL_WRITTEN])
  size += copy(message[size:], " has the role ")
  written := 0
  for _, term := range [...]Term{TERM_INDEX, TERM_COUNT, TERM_SIZE, TERM_OFFSET} {
    if terms&term == TERM_NONE {
      continue
    }
    if written > 0 {
      size += copy(message[size:], " or ")
    }
    written++
    cells[CELL_WRITTEN], cells[CELL_TERM] = State_Cell(size), State_Cell(term)
    term_name_write(checker)
    size = int(cells[CELL_WRITTEN])
  }
  cells[CELL_WRITTEN] = State_Cell(size + copy(message[size:], ". Rename "))
  name_append(checker)
  size = int(cells[CELL_WRITTEN])
  cells[CELL_WRITTEN], cells[CELL_TERM] = State_Cell(size+copy(message[size:], " -> ")), State_Cell(
    preferred,
  )
  rename_write(checker)
  size = int(cells[CELL_WRITTEN])
  cells[CELL_WRITTEN] = State_Cell(size + copy(message[size:], "."))
}

// Reads the operands of the sum CELL_BINARY names: the quantity each ends in, and the quantity
// lint's table gives the sum, or TERM_NONE when the table admits none.
func operands_find(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "operands_find.checker")
  cells := checker.Cells
  nodes := checker.Nodes
  tokens := checker.Tokens
  sum := nodes[cells[CELL_BINARY]]
  cells[CELL_LEFT], cells[CELL_RIGHT] = State_Cell(TERM_NONE), State_Cell(TERM_NONE)
  cells[CELL_RESULT], cells[CELL_SIGN] = State_Cell(TERM_NONE), State_Cell(tokens[sum.Token].Kind)
  switch tokens[sum.Token].Kind {
  case token.KIND_PLUS, token.KIND_MINUS:
  default:
    return
  }
  left := nodes[sum.First_Child]
  right := nodes[left.Next]
  if left.Kind != ast.NODE_IDENTIFIER {
    return
  }
  if right.Kind != ast.NODE_IDENTIFIER {
    return
  }
  for _, operand := range [...]ast.Node{left, right} {
    spelled := tokens[operand.Token]
    cells[CELL_NAME_START], cells[CELL_NAME_TEXT] = State_Cell(spelled.Offset), TEXT_SOURCE
    cells[CELL_NAME_END] = State_Cell(spelled.Offset) + State_Cell(spelled.Size)
    last_word(checker)
    cells[CELL_LEFT], cells[CELL_RIGHT] = cells[CELL_RIGHT], cells[CELL_SUFFIX]
  }
  first, second := Term(cells[CELL_LEFT]), Term(cells[CELL_RIGHT])
  if first == TERM_NONE {
    return
  }
  if second == TERM_NONE {
    return
  }
  if tokens[sum.Token].Kind == token.KIND_MINUS {
    if first == second {
      cells[CELL_RESULT] = State_Cell(first)
      switch first {
      case TERM_INDEX:
        cells[CELL_RESULT] = State_Cell(TERM_COUNT)
      case TERM_OFFSET:
        cells[CELL_RESULT] = State_Cell(TERM_SIZE)
      }
    }
    return
  }
  switch first | second {
  case TERM_COUNT, TERM_SIZE:
    cells[CELL_RESULT] = State_Cell(first)
  case TERM_COUNT | TERM_INDEX:
    cells[CELL_RESULT] = State_Cell(TERM_INDEX)
  case TERM_OFFSET | TERM_SIZE:
    cells[CELL_RESULT] = State_Cell(TERM_OFFSET)
  }
}

// States a sum of two quantities lint's table admits none for, at its left operand.
func sum_check(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "sum_check.checker")
  cells := checker.Cells
  operands_find(checker)
  if cells[CELL_LEFT] == State_Cell(TERM_NONE) {
    return
  }
  if cells[CELL_RIGHT] == State_Cell(TERM_NONE) {
    return
  }
  if cells[CELL_RESULT] != State_Cell(TERM_NONE) {
    return
  }
  nodes := checker.Nodes
  left := nodes[nodes[cells[CELL_BINARY]].First_Child]
  cells[CELL_FOUND] = 1
  cells[CELL_OFFSET] = State_Cell(checker.Tokens[left.Token].Offset)
  cells[CELL_WRITTEN] = State_Cell(copy(checker.Message, "The arithmetic "))
  operands_write(checker)
  size := int(cells[CELL_WRITTEN])
  tail := " is not coherent. Use operands with compatible suffixes."
  cells[CELL_WRITTEN] = State_Cell(size + copy(checker.Message[size:], tail))
}

// States a name a := takes from an admitted sum that does not end in the quantity it gives.
func result_check(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "result_check.checker")
  cells := checker.Cells
  nodes := checker.Nodes
  index := ast.Index(cells[CELL_NODE])
  sum := ast.Index(nodes[index].Next)
  if ast.Index(nodes[nodes[index].Parent].First_Child) != index {
    return
  }
  if nodes[sum].Kind != ast.NODE_BINARY {
    return
  }
  if nodes[sum].Next != ast.Successor(ast.INDEX_ABSENT) {
    return
  }
  start, end := cells[CELL_NAME_START], cells[CELL_NAME_END]
  cells[CELL_BINARY] = State_Cell(sum)
  operands_find(checker)
  cells[CELL_NAME_START], cells[CELL_NAME_END] = start, end
  if cells[CELL_RESULT] == State_Cell(TERM_NONE) {
    return
  }
  last_word(checker)
  if cells[CELL_SUFFIX] == cells[CELL_RESULT] {
    return
  }
  cells[CELL_FOUND] = 1
  message := checker.Message
  cells[CELL_WRITTEN] = State_Cell(copy(message, "The arithmetic "))
  name_append(checker)
  size := int(cells[CELL_WRITTEN])
  cells[CELL_WRITTEN] = State_Cell(size + copy(message[size:], " = "))
  operands_write(checker)
  size = int(cells[CELL_WRITTEN])
  size += copy(message[size:], " gives the wrong suffix. End the name with _")
  cells[CELL_WRITTEN], cells[CELL_TERM] = State_Cell(size), cells[CELL_RESULT]
  term_name_write(checker)
  size = int(cells[CELL_WRITTEN])
  cells[CELL_WRITTEN] = State_Cell(size + copy(message[size:], "."))
}

// Writes the operands of a sum as lint states them: _<left> <sign> _<right>.
func operands_write(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "operands_write.checker")
  cells := checker.Cells
  message := checker.Message
  size := int(cells[CELL_WRITTEN])
  cells[CELL_WRITTEN], cells[CELL_TERM] = State_Cell(size+copy(message[size:], "_")), cells[
    CELL_LEFT]
  term_name_write(checker)
  size = int(cells[CELL_WRITTEN])
  sign := " + _"
  if token.Kind(cells[CELL_SIGN]) == token.KIND_MINUS {
    sign = " - _"
  }
  cells[CELL_WRITTEN], cells[CELL_TERM] = State_Cell(size+copy(message[size:], sign)), cells[
    CELL_RIGHT]
  term_name_write(checker)
}

// States a name whose last word is a participle.
func participle_check(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "participle_check.checker")
  cells := checker.Cells
  last_word(checker)
  if cells[CELL_CLASS] != CLASS_PARTICIPLE {
    return
  }
  cells[CELL_FOUND] = 1
  message := checker.Message
  cells[CELL_WRITTEN] = State_Cell(copy(message, "The name \""))
  lower_write(checker)
  size := int(cells[CELL_WRITTEN])
  tail := "\" is a present participle. Rename it to a noun form."
  cells[CELL_WRITTEN] = State_Cell(size + copy(message[size:], tail))
}

// States a name holding max or min before its last word, with the word moved to the end.
func extremum_check(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "extremum_check.checker")
  cells := checker.Cells
  bound := int(cells[CELL_NAME_END]-cells[CELL_NAME_START]) + 1
  name_words(checker)
  count := 0
  for range bound {
    word_next(checker)
    if cells[CELL_WORD_START] == cells[CELL_WORD_END] {
      break
    }
    count++
  }
  name_words(checker)
  for ordinal := range count - 1 {
    word_next(checker)
    word_class(checker)
    switch cells[CELL_CLASS] {
    case CLASS_MAXIMUM, CLASS_MINIMUM:
    default:
      continue
    }
    cells[CELL_REPLACED], cells[CELL_PIECE] = State_Cell(ordinal), PIECE_MOVE
    cells[CELL_APPENDED], cells[CELL_FOUND] = State_Cell(TERM_NONE), 1
    message := checker.Message
    cells[CELL_WRITTEN] = State_Cell(copy(message, "Rename "))
    name_append(checker)
    size := int(cells[CELL_WRITTEN])
    cells[CELL_WRITTEN] = State_Cell(size + copy(message[size:], " -> "))
    rename_write(checker)
    size = int(cells[CELL_WRITTEN])
    cells[CELL_WRITTEN] = State_Cell(size + copy(message[size:], "."))
    return
  }
}

// States an import alias holding default, or one no other import of the same package asks for.
func alias_check(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "alias_check.checker")
  cells := checker.Cells
  nodes := checker.Nodes
  named := nodes[nodes[cells[CELL_NODE]].First_Child]
  if named.Kind != ast.NODE_IMPORT_NAME {
    return
  }
  spelled := checker.Tokens[named.Token]
  cells[CELL_NAME_START], cells[CELL_NAME_TEXT] = State_Cell(spelled.Offset), TEXT_SOURCE
  cells[CELL_NAME_END] = State_Cell(spelled.Offset) + State_Cell(spelled.Size)
  cells[CELL_OFFSET] = State_Cell(spelled.Offset)
  alias := checker.Source[cells[CELL_NAME_START]:cells[CELL_NAME_END]]
  for start := range len(alias) - len("default") + 1 {
    held := true
    for offset, symbol := range alias[start : start+len("default")] {
      if symbol-'A' < LETTER_COUNT {
        symbol += CASE_DISTANCE
      }
      if symbol != "default"[offset] {
        held = false
      }
    }
    if held {
      cells[CELL_PACKAGE_START], cells[CELL_PACKAGE_END] = 0, 0
      alias_message(checker)
      return
    }
  }
  import_index := cells[CELL_NODE]
  package_find(checker)
  start, end := cells[CELL_PACKAGE_START], cells[CELL_PACKAGE_END]
  own := checker.Source[start:end]
  count := 0
  child := nodes[ast.INDEX_FIRST].First_Child
  for range ast.NODE_COUNT_MAXIMUM {
    if child == ast.Head(ast.INDEX_ABSENT) {
      break
    }
    if nodes[child].Kind == ast.NODE_IMPORT {
      cells[CELL_NODE] = State_Cell(child)
      package_find(checker)
      if string(checker.Source[cells[CELL_PACKAGE_START]:cells[CELL_PACKAGE_END]]) == string(own) {
        count++
      }
    }
    child = ast.Head(nodes[child].Next)
  }
  cells[CELL_NODE], cells[CELL_PACKAGE_START], cells[CELL_PACKAGE_END] = import_index, start, end
  if count > 1 {
    return
  }
  alias_message(checker)
}

// Writes the message of the alias the name cells hold: its default, or, with no default in it,
// the package no other import names.
func alias_message(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "alias_message.checker")
  cells := checker.Cells
  message := checker.Message
  cells[CELL_FOUND] = 1
  if cells[CELL_PACKAGE_END] == 0 {
    cells[CELL_WRITTEN] = State_Cell(copy(message, "The import alias \""))
    name_append(checker)
    size := int(cells[CELL_WRITTEN])
    tail := "\" holds \"default\". Name the alias for the package. " +
      "A default directory declares the parent as that package."
    cells[CELL_WRITTEN] = State_Cell(size + copy(message[size:], tail))
    return
  }
  cells[CELL_WRITTEN] = State_Cell(copy(message, "Package import alias \""))
  name_append(checker)
  size := int(cells[CELL_WRITTEN])
  tail := "\" is unnecessary. Remove alias because no other imported package is named \""
  size += copy(message[size:], tail)
  for _, symbol := range checker.Source[cells[CELL_PACKAGE_START]:cells[CELL_PACKAGE_END]] {
    if symbol < ' ' {
      symbol = QUOTED_BYTE_ABSENT
    }
    if size < len(message) {
      message[size] = symbol
      size++
    }
  }
  cells[CELL_WRITTEN] = State_Cell(size + copy(message[size:], "\"."))
}

// Finds the package the import CELL_NODE names: the last segment of its path, or the segment
// before a last default, since a default directory declares its parent's package.
func package_find(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "package_find.checker")
  cells := checker.Cells
  nodes := checker.Nodes
  child := nodes[cells[CELL_NODE]].First_Child
  for range ast.NODE_COUNT_MAXIMUM {
    if nodes[child].Kind == ast.NODE_STRING {
      break
    }
    child = ast.Head(nodes[child].Next)
  }
  spelled := checker.Tokens[nodes[child].Token]
  start := int(spelled.Offset) + 1
  end := max(int(spelled.Offset)+int(spelled.Size)-1, start)
  segment, parent := start, start
  for offset := start; offset < end; offset++ {
    if checker.Source[offset] == filepath.SEPARATOR {
      parent, segment = segment, offset+1
    }
  }
  cells[CELL_PACKAGE_START], cells[CELL_PACKAGE_END] = State_Cell(segment), State_Cell(end)
  if string(checker.Source[segment:end]) != "default" {
    return
  }
  if segment > start {
    cells[CELL_PACKAGE_START], cells[CELL_PACKAGE_END] = State_Cell(parent), State_Cell(segment-1)
  }
}

// Writes the name the name cells hold into the message, each byte below a space as ?.
func name_append(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "name_append.checker")
  cells := checker.Cells
  text := []byte(checker.Source)
  if cells[CELL_NAME_TEXT] == TEXT_PATH {
    text = checker.Path
  }
  message := checker.Message
  size := int(cells[CELL_WRITTEN])
  for _, symbol := range text[cells[CELL_NAME_START]:cells[CELL_NAME_END]] {
    if symbol < ' ' {
      symbol = QUOTED_BYTE_ABSENT
    }
    if size < len(message) {
      message[size] = symbol
      size++
    }
  }
  cells[CELL_WRITTEN] = State_Cell(size)
}

// Writes the word the cursor read into the message, lowercase, each byte below a space as ?.
func lower_write(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "lower_write.checker")
  cells := checker.Cells
  text := []byte(checker.Source)
  if cells[CELL_TEXT] == TEXT_PATH {
    text = checker.Path
  }
  message := checker.Message
  size := int(cells[CELL_WRITTEN])
  for _, symbol := range text[cells[CELL_WORD_START]:cells[CELL_WORD_END]] {
    if symbol-'A' < LETTER_COUNT {
      symbol += CASE_DISTANCE
    }
    if symbol < ' ' {
      symbol = QUOTED_BYTE_ABSENT
    }
    if size < len(message) {
      message[size] = symbol
      size++
    }
  }
  cells[CELL_WRITTEN] = State_Cell(size)
}

// Writes the rename of the name: each word in the case of the name, the replaced word giving way
// to its piece, then the appended quantity, then the moved word.
func rename_write(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "rename_write.checker")
  cells := checker.Cells
  cells[CELL_JOINED] = 0
  name_words(checker)
  moved_start, moved_end := cells[CELL_NAME_START], cells[CELL_NAME_START]
  for ordinal := range int(cells[CELL_NAME_END]-cells[CELL_NAME_START]) + 1 {
    word_next(checker)
    if cells[CELL_WORD_START] == cells[CELL_WORD_END] {
      break
    }
    if State_Cell(ordinal) != cells[CELL_REPLACED] {
      word_write(checker)
      continue
    }
    switch cells[CELL_PIECE] {
    case PIECE_MOVE:
      moved_start, moved_end = cells[CELL_WORD_START], cells[CELL_WORD_END]
    case PIECE_TERM:
      term_write(checker)
    case PIECE_CONFIGURATION:
      piece_write(checker)
    }
  }
  if cells[CELL_APPENDED] != State_Cell(TERM_NONE) {
    cells[CELL_TERM] = cells[CELL_APPENDED]
    term_write(checker)
  }
  if cells[CELL_PIECE] == PIECE_MOVE {
    cells[CELL_WORD_START], cells[CELL_WORD_END] = moved_start, moved_end
    word_write(checker)
  }
}

// Writes each word of the replacement the piece cells hold, the cursor kept where it stood.
func piece_write(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "piece_write.checker")
  cells := checker.Cells
  text, cursor, limit := cells[CELL_TEXT], cells[CELL_CURSOR], cells[CELL_LIMIT]
  cells[CELL_TEXT] = TEXT_CONFIGURATION
  cells[CELL_CURSOR], cells[CELL_LIMIT] = cells[CELL_PIECE_START], cells[CELL_PIECE_END]
  for range int(cells[CELL_PIECE_END]-cells[CELL_PIECE_START]) + 1 {
    word_next(checker)
    if cells[CELL_WORD_START] == cells[CELL_WORD_END] {
      break
    }
    word_write(checker)
  }
  cells[CELL_TEXT], cells[CELL_CURSOR], cells[CELL_LIMIT] = text, cursor, limit
}

// Writes the word the cursor read in the case of the name, behind an underscore past the first
// word: lowercase for snake_case, and capitalized for Ada_Case unless it stands uppercase whole.
func word_write(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "word_write.checker")
  cells := checker.Cells
  text := []byte(checker.Source)
  switch cells[CELL_TEXT] {
  case TEXT_PATH:
    text = checker.Path
  case TEXT_CONFIGURATION:
    text = checker.Text
  }
  word := text[cells[CELL_WORD_START]:cells[CELL_WORD_END]]
  message := checker.Message
  size := int(cells[CELL_WRITTEN])
  if cells[CELL_JOINED] > 0 {
    size += copy(message[size:], "_")
  }
  cells[CELL_JOINED]++
  // A word stands uppercase whole when it holds a letter and no letter of it is lowercase.
  whole := false
  if cells[CELL_STYLE] == STYLE_ADA {
    for _, symbol := range word {
      if symbol-'A' < LETTER_COUNT {
        whole = true
      }
    }
    for _, symbol := range word {
      if symbol-'A' > LETTER_COUNT-1 {
        if symbol-'0' > DIGIT_COUNT-1 {
          whole = false
        }
      }
    }
  }
  for offset, symbol := range word {
    if !whole {
      if symbol-'A' < LETTER_COUNT {
        symbol += CASE_DISTANCE
      }
      if offset == 0 {
        if cells[CELL_STYLE] == STYLE_ADA {
          if symbol-'a' < LETTER_COUNT {
            symbol -= CASE_DISTANCE
          }
        }
      }
    }
    if symbol < ' ' {
      symbol = QUOTED_BYTE_ABSENT
    }
    if size < len(message) {
      message[size] = symbol
      size++
    }
  }
  cells[CELL_WRITTEN] = State_Cell(size)
}

// Writes the quantity CELL_TERM names as one word of a rename.
func term_write(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "term_write.checker")
  cells := checker.Cells
  message := checker.Message
  size := int(cells[CELL_WRITTEN])
  if cells[CELL_JOINED] > 0 {
    size += copy(message[size:], "_")
  }
  cells[CELL_JOINED]++
  opening := size
  cells[CELL_WRITTEN] = State_Cell(size)
  term_name_write(checker)
  if cells[CELL_STYLE] == STYLE_ADA {
    if opening < len(message) {
      message[opening] -= CASE_DISTANCE
    }
  }
}

// Writes the name of the quantity CELL_TERM names, lowercase.
func term_name_write(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "term_name_write.checker")
  cells := checker.Cells
  name := "offset"
  switch Term(cells[CELL_TERM]) {
  case TERM_INDEX:
    name = "index"
  case TERM_COUNT:
    name = "count"
  case TERM_SIZE:
    name = "size"
  }
  size := int(cells[CELL_WRITTEN])
  cells[CELL_WRITTEN] = State_Cell(size + copy(checker.Message[size:], name))
}

// Closes the message with a zero byte, cut at the store.
func message_close(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "message_close.checker")
  size := min(int(checker.Cells[CELL_WRITTEN]), MESSAGE_SIZE_MAXIMUM-1)
  checker.Message[size] = 0
}

// Marks the quantities the evidence of one declared function asks of the names it declares: the
// induction of a counted loop, a name := takes from a quantity call, and a name make sizes.
func terms_collect(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "terms_collect.checker")
  cells := checker.Cells
  function := cells[CELL_NODE]
  cells[CELL_ROOT] = function
  for range ast.NODE_COUNT_MAXIMUM {
    step(checker)
    if cells[CELL_NODE] == State_Cell(ast.INDEX_ABSENT) {
      break
    }
    switch checker.Nodes[cells[CELL_NODE]].Kind {
    case ast.NODE_FOR:
      induction_mark(checker)
    case ast.NODE_DEFINE:
      call_mark(checker)
    case ast.NODE_CALL:
      make_mark(checker)
    }
  }
  cells[CELL_NODE], cells[CELL_ROOT] = function, State_Cell(ast.INDEX_FIRST)
}

// Marks the induction of the loop the walk stands on as an index when the loop reads
// `for i := <literal>; i < n; i++`, or steps it down with --.
func induction_mark(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "induction_mark.checker")
  cells := checker.Cells
  nodes := checker.Nodes
  tokens := checker.Tokens
  start := nodes[nodes[cells[CELL_NODE]].First_Child]
  condition := nodes[start.Next]
  post := nodes[condition.Next]
  name := nodes[start.First_Child]
  value := nodes[name.Next]
  compare := nodes[condition.First_Child]
  if start.Kind != ast.NODE_DEFINE {
    return
  }
  if condition.Kind != ast.NODE_EXPRESSION_STATEMENT {
    return
  }
  switch post.Kind {
  case ast.NODE_INCREMENT, ast.NODE_DECREMENT:
  default:
    return
  }
  if name.Kind != ast.NODE_IDENTIFIER {
    return
  }
  switch value.Kind {
  case ast.NODE_INTEGER, ast.NODE_FLOAT, ast.NODE_IMAGINARY, ast.NODE_CHARACTER, ast.NODE_STRING:
  default:
    return
  }
  if value.Next != ast.Successor(ast.INDEX_ABSENT) {
    return
  }
  if compare.Kind != ast.NODE_BINARY {
    return
  }
  if tokens[compare.Token].Kind != token.KIND_LESS {
    return
  }
  spelled := tokens[name.Token]
  induction := checker.Source[spelled.Offset : spelled.Offset+token.Offset(spelled.Size)]
  if string(induction) == "_" {
    return
  }
  for _, operand := range [...]ast.Node{nodes[compare.First_Child], nodes[post.First_Child]} {
    if operand.Kind != ast.NODE_IDENTIFIER {
      return
    }
    spelled = tokens[operand.Token]
    if string(checker.Source[spelled.Offset:spelled.Offset+token.Offset(spelled.Size)]) !=
      string(induction) {
      return
    }
  }
  cells[CELL_MARKED], cells[CELL_TERM] = State_Cell(start.First_Child), State_Cell(TERM_INDEX)
  mark(checker)
}

// Marks the one name a := the walk stands on takes from a quantity call: len or cap asks a count
// or a size, a stdlib call lint lists its quantity, and a Len or Cap method a size.
func call_mark(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "call_mark.checker")
  cells := checker.Cells
  nodes := checker.Nodes
  tokens := checker.Tokens
  source := checker.Source
  define := nodes[cells[CELL_NODE]]
  name := nodes[define.First_Child]
  call := nodes[name.Next]
  if name.Kind != ast.NODE_IDENTIFIER {
    return
  }
  if call.Kind != ast.NODE_CALL {
    return
  }
  if call.Next != ast.Successor(ast.INDEX_ABSENT) {
    return
  }
  spelled := tokens[name.Token]
  if string(source[spelled.Offset:spelled.Offset+token.Offset(spelled.Size)]) == "_" {
    return
  }
  callee := nodes[call.First_Child]
  spelled = tokens[callee.Token]
  term := TERM_NONE
  if callee.Kind == ast.NODE_IDENTIFIER {
    switch string(source[spelled.Offset : spelled.Offset+token.Offset(spelled.Size)]) {
    case "len", "cap":
      term = TERM_COUNT | TERM_SIZE
    }
  }
  if callee.Kind == ast.NODE_SELECTOR {
    receiver := tokens[nodes[callee.First_Child].Token]
    selected := tokens[nodes[nodes[callee.First_Child].Next].Token]
    method := source[selected.Offset : selected.Offset+token.Offset(selected.Size)]
    // The probe holds receiver.method, which each listed call fits.
    probe := [SELECTOR_PROBE_SIZE]byte{}
    size := 0
    if nodes[callee.First_Child].Kind == ast.NODE_IDENTIFIER {
      size = copy(probe[:], source[receiver.Offset:receiver.Offset+token.Offset(receiver.Size)])
      size += copy(probe[size:], ".")
      size += copy(probe[size:], method)
    }
    if size == SELECTOR_PROBE_SIZE {
      size = 0
    }
    switch string(probe[:size]) {
    case "strings.Index", "strings.IndexByte", "strings.LastIndex", "bytes.Index",
      "bytes.IndexByte", "bytes.LastIndex":
      term = TERM_OFFSET
    case "strings.Count", "bytes.Count", "utf8.RuneCount", "utf8.RuneCountInString":
      term = TERM_COUNT
    case "binary.Size", "unsafe.Sizeof":
      term = TERM_SIZE
    }
    if term == TERM_NONE {
      switch string(method) {
      case "Len", "Cap":
        term = TERM_SIZE
      }
    }
  }
  if term == TERM_NONE {
    return
  }
  cells[CELL_MARKED], cells[CELL_TERM] = State_Cell(define.First_Child), State_Cell(term)
  mark(checker)
}

// Marks each name a make call the walk stands on sizes with, at the name's first declaration in
// the function: a count for a slice, a map, or a channel, and a size for a slice of bytes.
func make_mark(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "make_mark.checker")
  cells := checker.Cells
  nodes := checker.Nodes
  tokens := checker.Tokens
  source := checker.Source
  callee := nodes[nodes[cells[CELL_NODE]].First_Child]
  kind := nodes[callee.Next]
  spelled := tokens[callee.Token]
  if callee.Kind != ast.NODE_IDENTIFIER {
    return
  }
  if string(source[spelled.Offset:spelled.Offset+token.Offset(spelled.Size)]) != "make" {
    return
  }
  term := TERM_NONE
  switch kind.Kind {
  case ast.NODE_SLICE_TYPE:
    term = TERM_COUNT
    element := nodes[kind.First_Child]
    spelled = tokens[element.Token]
    if element.Kind == ast.NODE_IDENTIFIER {
      if string(source[spelled.Offset:spelled.Offset+token.Offset(spelled.Size)]) == "byte" {
        term = TERM_SIZE
      }
    }
  case ast.NODE_MAP_TYPE, ast.NODE_CHANNEL_TYPE:
    term = TERM_COUNT
  }
  if term == TERM_NONE {
    return
  }
  argument := ast.Index(kind.Next)
  for range ast.NODE_COUNT_MAXIMUM {
    if argument == ast.INDEX_ABSENT {
      break
    }
    cells[CELL_MARKED], cells[CELL_TERM] = State_Cell(argument), State_Cell(term)
    if nodes[argument].Kind == ast.NODE_IDENTIFIER {
      declaration_find(checker)
    }
    argument = ast.Index(nodes[argument].Next)
  }
}

// Marks the first declaration in the function of the name CELL_MARKED spells: a parameter, a
// result, a constant, a variable, or a name := defines. A name the function does not declare
// stays unmarked.
func declaration_find(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "declaration_find.checker")
  cells := checker.Cells
  nodes := checker.Nodes
  tokens := checker.Tokens
  spelled := tokens[nodes[cells[CELL_MARKED]].Token]
  wanted := checker.Source[spelled.Offset : spelled.Offset+token.Offset(spelled.Size)]
  origin := cells[CELL_NODE]
  cells[CELL_NODE] = cells[CELL_ROOT]
  for range ast.NODE_COUNT_MAXIMUM {
    step(checker)
    index := cells[CELL_NODE]
    if index == State_Cell(ast.INDEX_ABSENT) {
      break
    }
    switch nodes[index].Kind {
    case ast.NODE_IDENTIFIER, ast.NODE_PARAMETER_NAME, ast.NODE_CONSTANT_NAME:
    default:
      continue
    }
    parent := nodes[nodes[index].Parent]
    if parent.Kind == ast.NODE_FUNCTION {
      continue
    }
    if nodes[parent.Parent].Kind == ast.NODE_RECEIVER {
      continue
    }
    spelled = tokens[nodes[index].Token]
    if string(checker.Source[spelled.Offset:spelled.Offset+token.Offset(spelled.Size)]) !=
      string(wanted) {
      continue
    }
    if !bool(declared(checker)) {
      continue
    }
    cells[CELL_MARKED] = index
    mark(checker)
    break
  }
  cells[CELL_NODE] = origin
}

// Sets the quantity CELL_TERM names on the node CELL_MARKED names, and widens the span of marked
// nodes the next check clears.
func mark(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "mark.checker")
  cells := checker.Cells
  index := cells[CELL_MARKED]
  checker.Terms[index] |= Term(cells[CELL_TERM])
  if cells[CELL_HIGH] == 0 {
    cells[CELL_LOW] = index
  }
  cells[CELL_LOW], cells[CELL_HIGH] = min(cells[CELL_LOW], index), max(cells[CELL_HIGH], index+1)
}

// Walks the scopes of each declared function as lint does, and states the first name a scope
// declares that a scope around it declares, unless the check found another rule broken first. It
// records each name it judges, whatever it finds.
func shadow_walk(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "shadow_walk.checker")
  cells, nodes, frames := checker.Cells, checker.Nodes, checker.Frames
  cells[CELL_NAME_COUNT], cells[CELL_FRAME] = 0, 0
  frames[0] = Scope_Frame{Cursor: ast.INDEX_ABSENT, Start: 0}
  shadow_globals(checker)
  function := ast.Index(nodes[ast.INDEX_FIRST].First_Child)
  for range ast.NODE_COUNT_MAXIMUM {
    if function == ast.INDEX_ABSENT {
      break
    }
    if nodes[function].Kind == ast.NODE_FUNCTION {
      cells[CELL_CONTAINER] = State_Cell(function)
      shadow_function(checker)
    }
    function = ast.Index(nodes[function].Next)
  }
  cells[CELL_LOCAL_COUNT] = cells[CELL_RECORD_COUNT]
}

// Holds the names the file declares at the top level as the outermost scope: each function and
// method, variable, constant, and type.
func shadow_globals(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "shadow_globals.checker")
  cells, nodes := checker.Cells, checker.Nodes
  cells[CELL_JUDGED] = 0
  child := ast.Index(nodes[ast.INDEX_FIRST].First_Child)
  for range ast.NODE_COUNT_MAXIMUM {
    if child == ast.INDEX_ABSENT {
      break
    }
    node := nodes[child]
    switch node.Kind {
    case ast.NODE_VARIABLE, ast.NODE_CONSTANT:
      cells[CELL_CONTAINER] = State_Cell(child)
      shadow_names(checker)
    case ast.NODE_FUNCTION, ast.NODE_TYPE:
      name := ast.Index(node.First_Child)
      for nodes[name].Kind != ast.NODE_IDENTIFIER {
        name = ast.Index(nodes[name].Next)
      }
      cells[CELL_DECLARED] = State_Cell(name)
      shadow_declare(checker)
    }
    child = ast.Index(node.Next)
  }
}

// Walks the body of the function CELL_CONTAINER names: its parameters and results stand in the
// scope its body declares in, and each block, if, for, and range opens one inside it.
func shadow_function(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "shadow_function.checker")
  cells, nodes, frames := checker.Cells, checker.Nodes, checker.Frames
  function := ast.Index(cells[CELL_CONTAINER])
  cells[CELL_FRAME], cells[CELL_JUDGED] = 1, 0
  frames[1] = Scope_Frame{Cursor: ast.INDEX_ABSENT, Start: cells[CELL_NAME_COUNT]}
  child := ast.Index(nodes[function].First_Child)
  for range ast.NODE_COUNT_MAXIMUM {
    if child == ast.INDEX_ABSENT {
      break
    }
    switch nodes[child].Kind {
    case ast.NODE_PARAMETER, ast.NODE_RESULT:
      cells[CELL_CONTAINER] = State_Cell(child)
      shadow_names(checker)
    case ast.NODE_BLOCK:
      frames[1].Cursor = ast.Index(nodes[child].First_Child)
    }
    child = ast.Index(nodes[child].Next)
  }
  cells[CELL_JUDGED] = 1
  for range ast.NODE_COUNT_MAXIMUM {
    top := cells[CELL_FRAME]
    if top == 0 {
      break
    }
    statement := frames[top].Cursor
    if statement == ast.INDEX_ABSENT {
      cells[CELL_NAME_COUNT], cells[CELL_FRAME] = frames[top].Start, top-1
      continue
    }
    cells[CELL_CONTAINER] = State_Cell(statement)
    shadow_next(checker)
    shadow_statement(checker)
  }
  cells[CELL_NAME_COUNT] = frames[1].Start
}

// Moves the scope the walk stands in past the statement CELL_CONTAINER names: to the statement
// after it, or past an else branch to the statement after the if that holds it.
func shadow_next(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "shadow_next.checker")
  cells, nodes := checker.Cells, checker.Nodes
  current := ast.Index(cells[CELL_CONTAINER])
  for range ast.DEPTH_MAXIMUM + 1 {
    if nodes[current].Next != ast.Successor(ast.INDEX_ABSENT) {
      break
    }
    if nodes[nodes[current].Parent].Kind != ast.NODE_IF {
      break
    }
    current = ast.Index(nodes[current].Parent)
  }
  checker.Frames[cells[CELL_FRAME]].Cursor = ast.Index(nodes[current].Next)
}

// Applies the statement CELL_CONTAINER names to the scopes: a block, an if, and a for open one
// the walk enters, and a :=, a var, and a const declare their names in the scope it stands in.
func shadow_statement(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "shadow_statement.checker")
  cells, nodes, frames := checker.Cells, checker.Nodes, checker.Frames
  statement := ast.Index(cells[CELL_CONTAINER])
  node := nodes[statement]
  switch node.Kind {
  case ast.NODE_DEFINE:
    shadow_names(checker)
    return
  case ast.NODE_DECLARATION_STATEMENT:
    cells[CELL_CONTAINER] = State_Cell(node.First_Child)
    shadow_names(checker)
    return
  case ast.NODE_BLOCK, ast.NODE_IF, ast.NODE_FOR:
  default:
    return
  }
  top := cells[CELL_FRAME] + 1
  cells[CELL_FRAME] = top
  frames[top] = Scope_Frame{Cursor: ast.Index(node.First_Child), Start: cells[CELL_NAME_COUNT]}
  if node.Kind == ast.NODE_BLOCK {
    return
  }
  // An init is the first child an if or a for holds, and the body is its first block past it.
  if nodes[node.First_Child].Kind == ast.NODE_DEFINE {
    cells[CELL_CONTAINER] = State_Cell(node.First_Child)
    shadow_names(checker)
  }
  body := ast.Index(node.First_Child)
  for nodes[body].Kind != ast.NODE_BLOCK {
    body = ast.Index(nodes[body].Next)
  }
  frames[top].Cursor = ast.Index(nodes[body].First_Child)
  // An else branch opens its scope beside the if, thus the scope around the if reads it next.
  if node.Kind == ast.NODE_IF {
    branch := ast.Index(nodes[body].Next)
    for range ast.NODE_COUNT_MAXIMUM {
      if branch == ast.INDEX_ABSENT {
        break
      }
      if nodes[branch].Kind == ast.NODE_IF {
        frames[top-1].Cursor = branch
        break
      }
      if nodes[branch].Kind == ast.NODE_BLOCK {
        frames[top-1].Cursor = branch
        break
      }
      branch = ast.Index(nodes[branch].Next)
    }
  }
}

// Takes each name the node CELL_CONTAINER declares: the leading names a var or a := lists, the
// names of a const, and those of a parameter or a result. The type and the values a const or a
// parameter spells name nothing it declares.
func shadow_names(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "shadow_names.checker")
  cells, nodes, tokens := checker.Cells, checker.Nodes, checker.Tokens
  identifier_list := false
  switch nodes[cells[CELL_CONTAINER]].Kind {
  case ast.NODE_VARIABLE, ast.NODE_DEFINE:
    identifier_list = true
  }
  child := ast.Index(nodes[cells[CELL_CONTAINER]].First_Child)
  for range ast.NODE_COUNT_MAXIMUM {
    if child == ast.INDEX_ABSENT {
      return
    }
    node := nodes[child]
    switch node.Kind {
    case ast.NODE_CONSTANT_NAME, ast.NODE_PARAMETER_NAME:
      cells[CELL_DECLARED] = State_Cell(child)
      shadow_declare(checker)
    case ast.NODE_IDENTIFIER:
      if !identifier_list {
        return
      }
      cells[CELL_DECLARED] = State_Cell(child)
      shadow_declare(checker)
      // A note may stand between a name and the comma past it, and no comma ends the list.
      past := node.Token + 1
      for tokens[past].Kind == token.KIND_COMMENT {
        past++
      }
      if tokens[past].Kind != token.KIND_COMMA {
        return
      }
    }
    child = ast.Index(node.Next)
  }
}

// Declares the name CELL_DECLARED names in the scope the walk stands in. A judged name that a
// scope around it declares states the shadow, when the check found nothing before it, and is
// recorded.
func shadow_declare(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "shadow_declare.checker")
  cells, names := checker.Cells, checker.Names
  index := ast.Index(cells[CELL_DECLARED])
  spelled := checker.Tokens[checker.Nodes[index].Token]
  name := checker.Source[spelled.Offset : spelled.Offset+token.Offset(spelled.Size)]
  if string(name) == "_" {
    return
  }
  hash := Name_Hash(HASH_WIDE_BASIS)
  for _, symbol := range name {
    hash = (hash ^ Name_Hash(symbol)) * HASH_WIDE_PRIME
  }
  if cells[CELL_JUDGED] == 1 {
    checker.Records[cells[CELL_RECORD_COUNT]] = index
    cells[CELL_RECORD_COUNT]++
    for _, outer := range names[:checker.Frames[cells[CELL_FRAME]].Start] {
      if cells[CELL_FOUND] == 1 {
        break
      }
      if outer.Hash != hash {
        continue
      }
      held := checker.Tokens[checker.Nodes[outer.Node].Token]
      if string(checker.Source[held.Offset:held.Offset+token.Offset(held.Size)]) != string(name) {
        continue
      }
      shadow_message(checker)
    }
  }
  names[cells[CELL_NAME_COUNT]] = Scope_Name{Node: index, Hash: hash}
  cells[CELL_NAME_COUNT]++
}

// Writes the message of the shadow CELL_DECLARED names.
func shadow_message(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "shadow_message.checker")
  cells := checker.Cells
  spelled := checker.Tokens[checker.Nodes[cells[CELL_DECLARED]].Token]
  cells[CELL_FOUND], cells[CELL_OFFSET] = 1, State_Cell(spelled.Offset)
  cells[CELL_SHADOW] = 1
  cells[CELL_NAME_START], cells[CELL_NAME_TEXT] = State_Cell(spelled.Offset), TEXT_SOURCE
  cells[CELL_NAME_END] = State_Cell(spelled.Offset) + State_Cell(spelled.Size)
  message := checker.Message
  cells[CELL_WRITTEN] = State_Cell(copy(message, "The variable \""))
  name_append(checker)
  size := int(cells[CELL_WRITTEN])
  tail := "\" shadows a variable in an outer scope. Write a different name."
  cells[CELL_WRITTEN] = State_Cell(size + copy(message[size:], tail))
}

// Lists past the names Check judged each function but a method, type, and constant the file
// declares at the top level, for a caller judging names against the sibling files of a package,
// then the call graph of the file.
func Record(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "record.checker")
  cells, nodes := checker.Cells, checker.Nodes
  child := ast.Index(nodes[ast.INDEX_FIRST].First_Child)
  for range ast.NODE_COUNT_MAXIMUM {
    if child == ast.INDEX_ABSENT {
      break
    }
    node := nodes[child]
    name := ast.Index(node.First_Child)
    switch node.Kind {
    case ast.NODE_FUNCTION, ast.NODE_TYPE:
      // A method opens on its receiver, which a function never holds.
      if nodes[name].Kind == ast.NODE_IDENTIFIER {
        checker.Records[cells[CELL_RECORD_COUNT]] = name
        cells[CELL_RECORD_COUNT]++
      }
    case ast.NODE_CONSTANT:
      for range ast.NODE_COUNT_MAXIMUM {
        if name == ast.INDEX_ABSENT {
          break
        }
        if nodes[name].Kind == ast.NODE_CONSTANT_NAME {
          checker.Records[cells[CELL_RECORD_COUNT]] = name
          cells[CELL_RECORD_COUNT]++
        }
        name = ast.Index(nodes[name].Next)
      }
    }
    child = ast.Index(node.Next)
  }
  calls_record(checker)
  imports_record(checker)
  impurity_find(checker)
}

// Lists each import the file makes, and counts the free functions named Main it declares.
func imports_record(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "imports_record.checker")
  cells, nodes, tokens := checker.Cells, checker.Nodes, checker.Tokens
  cells[CELL_IMPORT_START], cells[CELL_MAIN_COUNT] = cells[CELL_RECORD_COUNT], 0
  child := ast.Index(nodes[ast.INDEX_FIRST].First_Child)
  for range ast.NODE_COUNT_MAXIMUM {
    if child == ast.INDEX_ABSENT {
      break
    }
    node := nodes[child]
    if node.Kind == ast.NODE_IMPORT {
      checker.Records[cells[CELL_RECORD_COUNT]] = child
      cells[CELL_RECORD_COUNT]++
    }
    // A method opens on its receiver, thus a free function holds its name first.
    if node.Kind == ast.NODE_FUNCTION {
      name := nodes[node.First_Child]
      if name.Kind == ast.NODE_IDENTIFIER {
        spelled := tokens[name.Token]
        text := checker.Source[spelled.Offset : spelled.Offset+token.Offset(spelled.Size)]
        if string(text) == "Main" {
          cells[CELL_MAIN_COUNT]++
        }
      }
    }
    child = ast.Index(node.Next)
  }
}

// States the first selector of a clock, stdout, or network API lint lists, and the first of an
// API lint lists as reaching one, each with the import whose name it selects from: the name the
// import binds, or else the last segment of its path.
func impurity_find(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "impurity_find.checker")
  cells, nodes, tokens, source := checker.Cells, checker.Nodes, checker.Tokens, checker.Source
  cells[CELL_SOFT_CALL], cells[CELL_TRANSITIVE_CALL] = 0, 0
  imports := checker.Records[cells[CELL_IMPORT_START]:cells[CELL_RECORD_COUNT]]
  cells[CELL_NODE], cells[CELL_ROOT] = State_Cell(ast.INDEX_FIRST), State_Cell(ast.INDEX_FIRST)
  for range ast.NODE_COUNT_MAXIMUM {
    step(checker)
    selector := ast.Index(cells[CELL_NODE])
    if selector == ast.INDEX_ABSENT {
      break
    }
    if nodes[selector].Kind != ast.NODE_SELECTOR {
      continue
    }
    receiver := nodes[nodes[selector].First_Child]
    selected := nodes[receiver.Next]
    if receiver.Kind != ast.NODE_IDENTIFIER {
      continue
    }
    spelled := tokens[receiver.Token]
    local := source[spelled.Offset : spelled.Offset+token.Offset(spelled.Size)]
    for _, bound := range imports {
      path := nodes[nodes[bound].First_Child]
      name := path
      for path.Kind != ast.NODE_STRING {
        path = nodes[path.Next]
      }
      held, quoted := tokens[name.Token], tokens[path.Token]
      text := source[quoted.Offset+1 : quoted.Offset+token.Offset(quoted.Size)-1]
      // An import that binds no name binds the last segment of its path.
      if name.Kind != ast.NODE_IMPORT_NAME {
        held = quoted
        held.Offset, held.Size = held.Offset+1, held.Size-2
        for offset := range text {
          if text[offset] == '/' {
            held.Offset, held.Size = quoted.Offset+1+token.Offset(offset)+1, quoted.Size-
              3-
              token.Size(offset)
          }
        }
      }
      if string(source[held.Offset:held.Offset+token.Offset(held.Size)]) != string(local) {
        continue
      }
      cells[CELL_DECLARED] = State_Cell(receiver.Next)
      cells[CELL_CONTAINER] = State_Cell(nodes[bound].First_Child)
      for nodes[cells[CELL_CONTAINER]].Kind != ast.NODE_STRING {
        cells[CELL_CONTAINER] = State_Cell(nodes[cells[CELL_CONTAINER]].Next)
      }
      if selected.Kind == ast.NODE_IDENTIFIER {
        impurity_class(checker)
      }
      break
    }
  }
  cells[CELL_NODE], cells[CELL_ROOT] = State_Cell(ast.INDEX_ABSENT), State_Cell(ast.INDEX_FIRST)
}

// Classes the selector CELL_DECLARED names under the import CELL_CONTAINER names, and records it
// as the first of its class.
func impurity_class(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "impurity_class.checker")
  cells, nodes, tokens, source := checker.Cells, checker.Nodes, checker.Tokens, checker.Source
  quoted := tokens[nodes[cells[CELL_CONTAINER]].Token]
  named := tokens[nodes[cells[CELL_DECLARED]].Token]
  probe := [IMPURITY_PROBE_SIZE]byte{}
  size := copy(probe[:], source[quoted.Offset+1:quoted.Offset+token.Offset(quoted.Size)-1])
  size += copy(probe[size:], ".")
  size += copy(probe[size:], source[named.Offset:named.Offset+token.Offset(named.Size)])
  if size == IMPURITY_PROBE_SIZE {
    return
  }
  soft, transitive := false, false
  switch string(probe[:size]) {
  case "time.Now", "time.Since", "time.Until", "time.Sleep", "time.After", "time.Tick",
    "time.NewTimer", "time.NewTicker", "fmt.Print", "fmt.Println", "fmt.Printf",
    "net/http.Get", "net/http.Post", "net/http.PostForm", "net/http.Head",
    "net/http.DefaultClient", "net/http.DefaultTransport", "net/http.DefaultServeMux",
    "net/http.Handle", "net/http.HandleFunc", "net/http.ListenAndServe",
    "net/http.ListenAndServeTLS", "net.Dial", "net.DialTimeout", "net.LookupHost",
    "net.LookupIP", "net.LookupAddr", "net.LookupCNAME", "net.LookupMX", "net.LookupNS",
    "net.LookupTXT", "net.LookupSRV", "net.LookupPort", "net.DefaultResolver":
    soft = true
  case "log.Fatal", "log.Fatalf", "log.Fatalln", "context.WithTimeout", "context.WithDeadline",
    "path/filepath.Abs", "path/filepath.Walk", "path/filepath.WalkDir", "path/filepath.Glob",
    "path/filepath.EvalSymlinks", "crypto/x509.SystemCertPool", "net/smtp.SendMail":
    transitive = true
  }
  if soft {
    if cells[CELL_SOFT_CALL] == 0 {
      cells[CELL_SOFT_CALL], cells[CELL_SOFT_IMPORT] = cells[CELL_DECLARED], cells[CELL_CONTAINER]
    }
  }
  if transitive {
    if cells[CELL_TRANSITIVE_CALL] == 0 {
      cells[CELL_TRANSITIVE_CALL] = cells[CELL_DECLARED]
      cells[CELL_TRANSITIVE_IMPORT] = cells[CELL_CONTAINER]
    }
  }
}

// Lists each function and method the file declares, each followed by the bare names its body
// calls, closures and go statements included.
func calls_record(checker Checker_Handle) {
  Checker_Handle_Invariants(checker, "calls_record.checker")
  cells, nodes := checker.Cells, checker.Nodes
  cells[CELL_GRAPH_START] = cells[CELL_RECORD_COUNT]
  function := ast.Index(nodes[ast.INDEX_FIRST].First_Child)
  for range ast.NODE_COUNT_MAXIMUM {
    if function == ast.INDEX_ABSENT {
      break
    }
    if nodes[function].Kind == ast.NODE_FUNCTION {
      // A method opens on its receiver, and its name is the identifier past it.
      name := ast.Index(nodes[function].First_Child)
      for nodes[name].Kind != ast.NODE_IDENTIFIER {
        name = ast.Index(nodes[name].Next)
      }
      checker.Records[cells[CELL_RECORD_COUNT]] = name
      cells[CELL_RECORD_COUNT]++
      cells[CELL_NODE], cells[CELL_ROOT] = State_Cell(function), State_Cell(function)
      for range ast.NODE_COUNT_MAXIMUM {
        step(checker)
        call := ast.Index(cells[CELL_NODE])
        if call == ast.INDEX_ABSENT {
          break
        }
        if nodes[call].Kind != ast.NODE_CALL {
          continue
        }
        if nodes[nodes[call].First_Child].Kind != ast.NODE_IDENTIFIER {
          continue
        }
        checker.Records[cells[CELL_RECORD_COUNT]] = ast.Index(nodes[call].First_Child)
        cells[CELL_RECORD_COUNT]++
      }
    }
    function = ast.Index(nodes[function].Next)
  }
  cells[CELL_NODE], cells[CELL_ROOT] = State_Cell(ast.INDEX_ABSENT), State_Cell(ast.INDEX_FIRST)
}
