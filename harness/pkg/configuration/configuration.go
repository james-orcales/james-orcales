// Package configuration decodes harness.json, the policy file that holds the shape of lint.json,
// into stores the caller owns. It reads no file: the caller reads the bytes and hands them over.
// shared/encoding/json decodes them into tokens, and Parse states the first fault the shape meets.
package configuration

import "local/james-orcales/shared/encoding/json"
import "local/james-orcales/shared/sim/aver/default"
import "local/james-orcales/shared/unicode/ucd"
import "local/james-orcales/shared/unicode/utf8"

// SOURCE_SIZE_MAXIMUM bounds the file Parse decodes. The file holds a handful of lists and the
// word table, thus the bound only stops a file no author wrote.
const SOURCE_SIZE_MAXIMUM = 64 * 1024

// SOURCE_STORAGE_SIZE is one byte past the bound: a source that fills it proves itself past the
// bound.
const SOURCE_STORAGE_SIZE = SOURCE_SIZE_MAXIMUM + 1

// SOURCE_SIZE_MINIMUM is the empty source, which is no JSON value.
const SOURCE_SIZE_MINIMUM = 0

// VALUE_COUNT_MAXIMUM bounds the strings one file holds, keys aside.
const VALUE_COUNT_MAXIMUM = 4096

// TOKEN_COUNT_MAXIMUM bounds the tokens of a file whose strings fill the value table: the root
// object opens and closes, each key spends its name and at most two brackets, and each value at
// most three tokens, a word and the brackets of its replacements.
const TOKEN_COUNT_MAXIMUM = 2 + 3*KEY_COUNT + 3*VALUE_COUNT_MAXIMUM

// COUNT_MINIMUM is a table that holds nothing.
const COUNT_MINIMUM = 0

// OFFSET_MINIMUM is the first byte of the source.
const OFFSET_MINIMUM = 0

// KEY_SHARED_COMPONENT names the shared library module, a string.
const KEY_SHARED_COMPONENT Key = 0

// KEY_INSTRUMENTATION_PACKAGES names the write-only instrumentation packages.
const KEY_INSTRUMENTATION_PACKAGES Key = 1

// KEY_WORD_REPLACEMENTS names the word table, an object of string arrays.
const KEY_WORD_REPLACEMENTS Key = 2

// KEY_IGNORE names the paths no check reads.
const KEY_IGNORE Key = 3

// KEY_ASSERTION_EXEMPT names the packages out of the assertion mandate.
const KEY_ASSERTION_EXEMPT Key = 4

// KEY_RECURSION_EXEMPT names the paths out of the recursion ban.
const KEY_RECURSION_EXEMPT Key = 5

// KEY_INTERFACE_EXEMPT names the paths out of the interface ban.
const KEY_INTERFACE_EXEMPT Key = 6

// KEY_COUNT is how many keys one file holds, each once.
const KEY_COUNT = 7

// Key names one key of the file, in the order an absent key is stated.
type Key int

// Key_Invariants admits each key.
func Key_Invariants(value Key, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(
    int(value),
    int(KEY_SHARED_COMPONENT),
    int(KEY_INTERFACE_EXEMPT),
  ).
    Ensure()
}

// KEY_NAME_SIZE_MINIMUM is the shortest key.
const KEY_NAME_SIZE_MINIMUM = len("ignore")

// KEY_NAME_SIZE_MAXIMUM is the longest key.
const KEY_NAME_SIZE_MAXIMUM = len("opt_out_assertion_mandate_packages")

// Key_Name is the text of one key.
type Key_Name string

// Key_Name_Invariants bounds the name by the keys.
func Key_Name_Invariants(value Key_Name, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(len(value), KEY_NAME_SIZE_MINIMUM, KEY_NAME_SIZE_MAXIMUM).
    Ensure()
}

// WORD_NONE is the word of a value that replaces no word.
const WORD_NONE Word_Slot = -1

// WORD_MAXIMUM is the last slot of the value table.
const WORD_MAXIMUM = VALUE_COUNT_MAXIMUM - 1

// Word_Slot is the slot of the word a replacement replaces, or WORD_NONE.
type Word_Slot int

// Word_Slot_Invariants bounds the slot by the value table.
func Word_Slot_Invariants(value Word_Slot, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(int(value), int(WORD_NONE), WORD_MAXIMUM).
    Ensure()
}

// Text_Start is the first byte of one text in the text store.
type Text_Start int

// Text_Start_Invariants bounds the start by the text store.
func Text_Start_Invariants(value Text_Start, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(int(value), OFFSET_MINIMUM, SOURCE_SIZE_MAXIMUM).
    Ensure()
}

// Text_End is one past the last byte of one text in the text store.
type Text_End int

// Text_End_Invariants bounds the end by the text store.
func Text_End_Invariants(value Text_End, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(int(value), OFFSET_MINIMUM, SOURCE_SIZE_MAXIMUM).
    Ensure()
}

// Value is one decoded string: the key it stands under, the word it replaces, and where its text
// stands in the text store.
type Value struct {
  // Key is the key the string stands under.
  Key Key
  // Word_Slot is the word a replacement replaces, or WORD_NONE.
  Word Word_Slot
  // Start is the first byte of the text.
  Start Text_Start
  // End is one past the last byte of the text.
  End Text_End
}

// Value_Invariants composes one value.
func Value_Invariants(value *Value, namespace aver.Namespace) {
  Key_Invariants(value.Key, namespace)
  Word_Slot_Invariants(value.Word, namespace)
  Text_Start_Invariants(value.Start, namespace)
  Text_End_Invariants(value.End, namespace)
}

// Values is the value table, fixed by the caller at its whole size.
type Values []Value

// Values_Invariants fixes the store.
func Values_Invariants(value Values, _ aver.Namespace) {
  aver.Always(len(value) == VALUE_COUNT_MAXIMUM, "A configuration value table has complete length.")
  aver.Always(cap(value) == VALUE_COUNT_MAXIMUM, "A configuration value table cannot grow.")
}

// Text holds the decoded strings. A string decodes to no more bytes than it spends in the source,
// thus a store the size of the widest source holds every one.
type Text []byte

// Text_Invariants fixes the store.
func Text_Invariants(value Text, _ aver.Namespace) {
  aver.Always(len(value) == SOURCE_SIZE_MAXIMUM, "A configuration text store has complete length.")
  aver.Always(cap(value) == SOURCE_SIZE_MAXIMUM, "A configuration text store cannot grow.")
}

// Source is the file as the caller read it.
type Source []byte

// Source_Invariants bounds the source by the byte that proves it past the bound.
func Source_Invariants(value Source, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(len(value), SOURCE_SIZE_MINIMUM, SOURCE_STORAGE_SIZE).
    Ensure()
}

// Tokens is the token store the decoder writes.
type Tokens json.Tokens

// Tokens_Invariants fixes the store.
func Tokens_Invariants(value Tokens, _ aver.Namespace) {
  aver.Always(len(value) == TOKEN_COUNT_MAXIMUM, "A configuration token store has complete length.")
  aver.Always(cap(value) == TOKEN_COUNT_MAXIMUM, "A configuration token store cannot grow.")
}

// CELL_MINIMUM is the least a slot holds: a word slot that names no word.
const CELL_MINIMUM = -1

// CELL_MAXIMUM is the most a slot holds: an offset of the text store.
const CELL_MAXIMUM = SOURCE_SIZE_MAXIMUM

// Cell is one slot of the decoder's state.
type Cell int

// Cell_Invariants bounds the slot by what one slot holds.
func Cell_Invariants(value Cell, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(int(value), CELL_MINIMUM, CELL_MAXIMUM).
    Ensure()
}

// Cells holds the decoder's state, one slot each.
type Cells []Cell

// Cells_Invariants fixes the store.
func Cells_Invariants(value Cells, _ aver.Namespace) {
  aver.Always(len(value) == CELL_COUNT, "A configuration number store has complete length.")
  aver.Always(cap(value) == CELL_COUNT, "A configuration number store cannot grow.")
}

// CELL_TOKEN is the token the decoder stands on.
const CELL_TOKEN = 0

// CELL_TOKEN_COUNT is how many tokens the source decoded into.
const CELL_TOKEN_COUNT = 1

// CELL_VALUE is how many values the table holds.
const CELL_VALUE = 2

// CELL_PRESENT holds one bit for each key the object stated.
const CELL_PRESENT = 3

// CELL_KEY is the key whose value the decoder reads.
const CELL_KEY = 4

// CELL_WORD is the word the replacements being read replace, or WORD_NONE.
const CELL_WORD = 5

// CELL_FAULT is the first fault.
const CELL_FAULT = 6

// CELL_FAULT_KEY is the key a keyed fault names.
const CELL_FAULT_KEY = 7

// CELL_FAULT_START is the first byte of the text a quoted fault names.
const CELL_FAULT_START = 8

// CELL_FAULT_END is one past the last byte of the text a quoted fault names.
const CELL_FAULT_END = 9

// CELL_ENTRY is the slot of the glob entry being checked.
const CELL_ENTRY = 10

// CELL_SPAN_START is the first byte of the text a blank check reads.
const CELL_SPAN_START = 11

// CELL_SPAN_END is one past the last byte of the text a blank check reads.
const CELL_SPAN_END = 12

// CELL_BLANK is 1 when the text a blank check read holds spaces alone.
const CELL_BLANK = 13

// CELL_LIST is the list of the order lint checks them in that Entry_Next stands on.
const CELL_LIST = 14

// CELL_SLOT is the next value slot Entry_Next reads.
const CELL_SLOT = 15

// CELL_COUNT is how many slots the state holds.
const CELL_COUNT = 16

// Configuration is the decoded file and the stores Parse decodes it through.
type Configuration struct {
  // Tokens holds the tokens the source decodes into.
  Tokens Tokens
  // Text holds the decoded strings.
  Text Text
  // Values names each decoded string.
  Values Values
  // Cells holds the decoder's state.
  Cells Cells
}

// Configuration_Invariants composes the stores.
func Configuration_Invariants(value *Configuration, namespace aver.Namespace) {
  Tokens_Invariants(value.Tokens, namespace)
  Text_Invariants(value.Text, namespace)
  Values_Invariants(value.Values, namespace)
  Cells_Invariants(value.Cells, namespace)
}

// Configuration_Handle is caller-owned configuration.
type Configuration_Handle *Configuration

// Configuration_Handle_Invariants composes present configuration.
func Configuration_Handle_Invariants(value Configuration_Handle, namespace aver.Namespace) {
  if value == nil {
    return
  }
  Configuration_Invariants(value, namespace)
}

// FAULT_NONE is a file that decodes whole.
const FAULT_NONE Fault_Kind = 0

// FAULT_SYNTAX is a source that is no JSON value.
const FAULT_SYNTAX Fault_Kind = 1

// FAULT_OVERSIZED is a source past SOURCE_SIZE_MAXIMUM.
const FAULT_OVERSIZED Fault_Kind = 2

// FAULT_FULL is a string past the value table, or a token past the token table.
const FAULT_FULL Fault_Kind = 3

// FAULT_OBJECT is a root other than an object.
const FAULT_OBJECT Fault_Kind = 4

// FAULT_KEY_UNKNOWN is a key the shape does not hold.
const FAULT_KEY_UNKNOWN Fault_Kind = 5

// FAULT_KEY_REPEATED is a key the object already stated.
const FAULT_KEY_REPEATED Fault_Kind = 6

// FAULT_TYPE is a key whose value is of another type than the shape states.
const FAULT_TYPE Fault_Kind = 7

// FAULT_KEY_ABSENT is a key the object never stated.
const FAULT_KEY_ABSENT Fault_Kind = 8

// FAULT_EMPTY is a shared_component of no byte, or a word_replacements of no word.
const FAULT_EMPTY Fault_Kind = 9

// FAULT_GLOB_EMPTY is a glob entry of spaces alone.
const FAULT_GLOB_EMPTY Fault_Kind = 10

// FAULT_GLOB_NEGATION is a glob entry whose ! negates nothing.
const FAULT_GLOB_NEGATION Fault_Kind = 11

// FAULT_GLOB_TOKEN is a glob entry holding ? or [, which the matcher does not support.
const FAULT_GLOB_TOKEN Fault_Kind = 12

// FAULT_GLOB_ESCAPE is a glob entry with a segment that ends in a lone backslash.
const FAULT_GLOB_ESCAPE Fault_Kind = 13

// FAULT_GLOB_NEGATED is a glob list of negations alone, which no entry of it can reach.
const FAULT_GLOB_NEGATED Fault_Kind = 14

// FAULT_ASSERTION_REPEATED is an assertion opt-out equal to an instrumentation entry, which the
// assertion mandate already exempts.
const FAULT_ASSERTION_REPEATED Fault_Kind = 15

// Fault_Kind names the first fault of a file, or FAULT_NONE.
type Fault_Kind int

// Fault_Kind_Invariants admits each fault.
func Fault_Kind_Invariants(value Fault_Kind, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(int(value), int(FAULT_NONE), int(FAULT_ASSERTION_REPEATED)).
    Ensure()
}

// Recorded_Fault is a fault the decoder met, which is never FAULT_NONE.
type Recorded_Fault Fault_Kind

// Recorded_Fault_Invariants admits each fault but none.
func Recorded_Fault_Invariants(value Recorded_Fault, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(
    int(value),
    int(FAULT_SYNTAX),
    int(FAULT_ASSERTION_REPEATED),
  ).
    Ensure()
}

// Fault_Offset is the byte a syntax fault stands on, counted as shared/encoding/json counts the
// bytes it read: one past the last byte of a source that ends inside a value.
type Fault_Offset int

// Fault_Offset_Invariants bounds the offset by the source and the byte past it.
func Fault_Offset_Invariants(value Fault_Offset, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(int(value), OFFSET_MINIMUM, SOURCE_STORAGE_SIZE).
    Ensure()
}

// Fault is the first fault of a file: what it is, the byte a syntax fault stands on, and the key
// a keyed fault names. The text a quoted fault names stands in the text store, at the cells
// CELL_FAULT_START and CELL_FAULT_END.
type Fault struct {
  // Kind is what the fault is.
  Kind Fault_Kind
  // Offset is the byte a syntax fault stands on.
  Offset Fault_Offset
  // Key is the key a keyed fault names.
  Key Key
}

// Fault_Invariants composes one fault.
func Fault_Invariants(value *Fault, namespace aver.Namespace) {
  Fault_Kind_Invariants(value.Kind, namespace)
  Fault_Offset_Invariants(value.Offset, namespace)
  Key_Invariants(value.Key, namespace)
}

// FAULT_MESSAGE_SIZE_MINIMUM is the shortest fault message.
const FAULT_MESSAGE_SIZE_MINIMUM = len(MESSAGE_EMPTY)

// FAULT_MESSAGE_SIZE_MAXIMUM is the longest fault message.
const FAULT_MESSAGE_SIZE_MAXIMUM = len(MESSAGE_ASSERTION_REPEATED)

// MESSAGE_EMPTY states a value of no byte or no word, past the key.
const MESSAGE_EMPTY = "is empty"

// MESSAGE_KEY_ABSENT states a key the object never stated, past the key.
const MESSAGE_KEY_ABSENT = "is absent"

// MESSAGE_ASSERTION_REPEATED states an assertion opt-out that repeats an instrumentation entry.
const MESSAGE_ASSERTION_REPEATED = "entry repeats an instrumentation_packages entry"

// Keyed is whether a fault names the key it stands under.
type Keyed bool

// Keyed_Invariants proves both kinds of fault occur.
func Keyed_Invariants(value Keyed, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Sometimes(bool(value), "A configuration fault names its key.").
    Ensure()
}

// Quoted is whether a fault names the text it stands on.
type Quoted bool

// Quoted_Invariants proves both kinds of fault occur.
func Quoted_Invariants(value Quoted, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Sometimes(bool(value), "A configuration fault names its text.").
    Ensure()
}

// Fault_Message is what one fault states.
type Fault_Message string

// Fault_Message_Invariants bounds the message by the messages.
func Fault_Message_Invariants(value Fault_Message, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(
    len(value),
    FAULT_MESSAGE_SIZE_MINIMUM,
    FAULT_MESSAGE_SIZE_MAXIMUM,
  ).
    Ensure()
}

// BACKSLASH escapes the byte past it in a glob.
const BACKSLASH = 0x5c

// Parse decodes the source into the configuration's stores and states the first fault, or
// FAULT_NONE. A value stands in the table only up to the fault.
func Parse(configuration Configuration_Handle, source Source) (fault Fault) {
  defer func() { Fault_Invariants(&fault, "parse.fault") }()
  Configuration_Handle_Invariants(configuration, "parse.configuration")
  Source_Invariants(source, "parse.source")
  cells := configuration.Cells
  clear(cells)
  cells[CELL_WORD] = Cell(WORD_NONE)
  if len(source) > SOURCE_SIZE_MAXIMUM {
    return Fault{Kind: FAULT_OVERSIZED}
  }
  count, _, position, status := json.Decode_Into(
    json.Tokens(configuration.Tokens),
    json.Output(configuration.Text),
    json.Encoded(source),
  )
  switch status {
  case json.STATUS_INPUT_INVALID:
    return Fault{Kind: FAULT_SYNTAX, Offset: Fault_Offset(position)}
  case json.STATUS_OUTPUT_TOO_SMALL:
    return Fault{Kind: FAULT_FULL}
  }
  // The stores are the caller's own and the text store holds the widest source, thus the decoder
  // refuses no store and no decoded size.
  aver.Always(status == json.STATUS_OK, "A configuration decodes into its stores.")
  cells[CELL_TOKEN_COUNT] = Cell(count)
  object_decode(configuration)
  if cells[CELL_FAULT] == Cell(FAULT_NONE) {
    object_close(configuration)
  }
  return Fault{
    Kind:   Fault_Kind(cells[CELL_FAULT]),
    Key:    Key(cells[CELL_FAULT_KEY]),
    Offset: OFFSET_MINIMUM,
  }
}

// Shape_Fault is a fault the decoded tokens meet: a string past the value table, or a shape the
// file does not hold.
type Shape_Fault Fault_Kind

// Shape_Fault_Invariants admits each fault past the decode.
func Shape_Fault_Invariants(value Shape_Fault, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(int(value), int(FAULT_FULL), int(FAULT_ASSERTION_REPEATED)).
    Ensure()
}

// Records the first fault, naming the key the decoder stands under.
func fault_record(configuration Configuration_Handle, kind Shape_Fault) {
  Configuration_Handle_Invariants(configuration, "fault_record.configuration")
  Shape_Fault_Invariants(kind, "fault_record.kind")
  cells := configuration.Cells
  cells[CELL_FAULT] = Cell(kind)
  cells[CELL_FAULT_KEY] = cells[CELL_KEY]
}

// Reports whether the token the decoder stands on is of one kind.
func token_is(configuration Configuration_Handle, kind Shape_Kind) (held Held) {
  defer func() { Held_Invariants(held, "token_is.held") }()
  Configuration_Handle_Invariants(configuration, "token_is.configuration")
  Shape_Kind_Invariants(kind, "token_is.kind")
  return configuration.Tokens[configuration.Cells[CELL_TOKEN]].Kind == json.Kind(kind)
}

// Shape_Kind is a token kind the shape states: a string, or a bracket of an array or an object.
type Shape_Kind json.Kind

// Shape_Kind_Invariants admits each kind the shape states.
func Shape_Kind_Invariants(value Shape_Kind, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint8(
    uint8(value),
    uint8(json.KIND_STRING),
    uint8(json.KIND_OBJECT_END),
  ).
    Ensure()
}

// Held is whether the token the decoder stands on is of the kind it asked for.
type Held bool

// Held_Invariants proves both answers occur.
func Held_Invariants(value Held, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Sometimes(
    bool(value),
    "A configuration token is of the kind the shape states.",
  ).
    Ensure()
}

// Reads the root object and each member it holds, up to the brace that closes it. The decoder
// states one complete value, thus the root's closing brace is the last token. Past the seventh
// member a key is unknown or repeated, thus the members the loop reads are bounded.
func object_decode(configuration Configuration_Handle) {
  Configuration_Handle_Invariants(configuration, "object_decode.configuration")
  cells := configuration.Cells
  if !bool(token_is(configuration, Shape_Kind(json.KIND_OBJECT))) {
    fault_record(configuration, Shape_Fault(FAULT_OBJECT))
    return
  }
  cells[CELL_TOKEN]++
  for range KEY_COUNT + 1 {
    if cells[CELL_TOKEN] == cells[CELL_TOKEN_COUNT]-1 {
      return
    }
    key_decode(configuration)
    if cells[CELL_FAULT] != Cell(FAULT_NONE) {
      return
    }
    cells[CELL_TOKEN]++
    member_decode(configuration)
    if cells[CELL_FAULT] != Cell(FAULT_NONE) {
      return
    }
  }
}

// Reads one member name, which names a key of the shape the object has not stated yet.
func key_decode(configuration Configuration_Handle) {
  Configuration_Handle_Invariants(configuration, "key_decode.configuration")
  cells := configuration.Cells
  named := configuration.Tokens[cells[CELL_TOKEN]]
  name := configuration.Text[named.Start:named.End]
  // The conversion stands inside the comparison, where it copies nothing.
  for key := range Key(KEY_COUNT) {
    if string(name) != string(Key_Text(key)) {
      continue
    }
    cells[CELL_KEY] = Cell(key)
    if cells[CELL_PRESENT]&(1<<key) != 0 {
      fault_record(configuration, Shape_Fault(FAULT_KEY_REPEATED))
      return
    }
    cells[CELL_PRESENT] |= 1 << key
    return
  }
  cells[CELL_FAULT_START], cells[CELL_FAULT_END] = Cell(named.Start), Cell(named.End)
  fault_record(configuration, Shape_Fault(FAULT_KEY_UNKNOWN))
}

// Reads the value of the key the decoder stands under, as the shape states its type.
func member_decode(configuration Configuration_Handle) {
  Configuration_Handle_Invariants(configuration, "member_decode.configuration")
  cells := configuration.Cells
  cells[CELL_WORD] = Cell(WORD_NONE)
  switch Key(cells[CELL_KEY]) {
  case KEY_SHARED_COMPONENT:
    value_decode(configuration)
  case KEY_WORD_REPLACEMENTS:
    words_decode(configuration)
  default:
    list_decode(configuration)
  }
}

// Reads one array of strings under the key the decoder stands on, and steps past its bracket.
func list_decode(configuration Configuration_Handle) {
  Configuration_Handle_Invariants(configuration, "list_decode.configuration")
  cells := configuration.Cells
  if !bool(token_is(configuration, Shape_Kind(json.KIND_ARRAY))) {
    fault_record(configuration, Shape_Fault(FAULT_TYPE))
    return
  }
  cells[CELL_TOKEN]++
  // Past the last slot a string is a fault, thus the strings the loop reads are bounded.
  for range VALUE_COUNT_MAXIMUM + 1 {
    if bool(token_is(configuration, Shape_Kind(json.KIND_ARRAY_END))) {
      cells[CELL_TOKEN]++
      return
    }
    value_decode(configuration)
    if cells[CELL_FAULT] != Cell(FAULT_NONE) {
      return
    }
  }
}

// Reads the word table: each word, then the array of its replacements, and steps past its brace.
func words_decode(configuration Configuration_Handle) {
  Configuration_Handle_Invariants(configuration, "words_decode.configuration")
  cells := configuration.Cells
  if !bool(token_is(configuration, Shape_Kind(json.KIND_OBJECT))) {
    fault_record(configuration, Shape_Fault(FAULT_TYPE))
    return
  }
  cells[CELL_TOKEN]++
  for range VALUE_COUNT_MAXIMUM + 1 {
    if bool(token_is(configuration, Shape_Kind(json.KIND_OBJECT_END))) {
      cells[CELL_TOKEN]++
      return
    }
    cells[CELL_WORD] = Cell(WORD_NONE)
    value_decode(configuration)
    if cells[CELL_FAULT] != Cell(FAULT_NONE) {
      return
    }
    cells[CELL_WORD] = cells[CELL_VALUE] - 1
    list_decode(configuration)
    if cells[CELL_FAULT] != Cell(FAULT_NONE) {
      return
    }
  }
}

// Reads one string into the next slot of the value table, and steps past it.
func value_decode(configuration Configuration_Handle) {
  Configuration_Handle_Invariants(configuration, "value_decode.configuration")
  cells := configuration.Cells
  if !bool(token_is(configuration, Shape_Kind(json.KIND_STRING))) {
    fault_record(configuration, Shape_Fault(FAULT_TYPE))
    return
  }
  if cells[CELL_VALUE] == VALUE_COUNT_MAXIMUM {
    fault_record(configuration, Shape_Fault(FAULT_FULL))
    return
  }
  decoded := configuration.Tokens[cells[CELL_TOKEN]]
  configuration.Values[cells[CELL_VALUE]] = Value{
    Key:   Key(cells[CELL_KEY]),
    Word:  Word_Slot(cells[CELL_WORD]),
    Start: Text_Start(decoded.Start),
    End:   Text_End(decoded.End),
  }
  cells[CELL_VALUE]++
  cells[CELL_TOKEN]++
}

// Checks what the object states as a whole: each key, a shared component of at least one byte, at
// least one word, then each glob list in the order lint checks them.
func object_close(configuration Configuration_Handle) {
  Configuration_Handle_Invariants(configuration, "object_close.configuration")
  cells := configuration.Cells
  for key := range Cell(KEY_COUNT) {
    if cells[CELL_PRESENT]&(1<<key) == 0 {
      cells[CELL_KEY] = key
      fault_record(configuration, Shape_Fault(FAULT_KEY_ABSENT))
      return
    }
  }
  words, component := 0, 0
  for _, value := range configuration.Values[:cells[CELL_VALUE]] {
    if value.Key == KEY_SHARED_COMPONENT {
      component = int(value.End) - int(value.Start)
    }
    if value.Key == KEY_WORD_REPLACEMENTS {
      if value.Word == WORD_NONE {
        words++
      }
    }
  }
  if component == 0 {
    cells[CELL_KEY] = Cell(KEY_SHARED_COMPONENT)
    fault_record(configuration, Shape_Fault(FAULT_EMPTY))
    return
  }
  if words == 0 {
    cells[CELL_KEY] = Cell(KEY_WORD_REPLACEMENTS)
    fault_record(configuration, Shape_Fault(FAULT_EMPTY))
    return
  }
  // Lint checks the lists in this order, thus the fault it states first is this one's too.
  lists := [...]Key{
    KEY_IGNORE,
    KEY_INSTRUMENTATION_PACKAGES,
    KEY_ASSERTION_EXEMPT,
    KEY_RECURSION_EXEMPT,
    KEY_INTERFACE_EXEMPT,
  }
  for _, key := range lists {
    cells[CELL_KEY] = Cell(key)
    list_check(configuration)
    if cells[CELL_FAULT] != Cell(FAULT_NONE) {
      return
    }
  }
  exemptions_check(configuration)
}

// Entry_Fault is a fault one glob entry meets.
type Entry_Fault Fault_Kind

// Entry_Fault_Invariants admits each fault of one entry.
func Entry_Fault_Invariants(value Entry_Fault, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(
    int(value),
    int(FAULT_GLOB_EMPTY),
    int(FAULT_ASSERTION_REPEATED),
  ).
    Ensure()
}

// Records the first fault of one entry, naming its text.
func entry_fault(configuration Configuration_Handle, kind Entry_Fault) {
  Configuration_Handle_Invariants(configuration, "entry_fault.configuration")
  Entry_Fault_Invariants(kind, "entry_fault.kind")
  cells := configuration.Cells
  value := configuration.Values[cells[CELL_ENTRY]]
  cells[CELL_FAULT_START], cells[CELL_FAULT_END] = Cell(value.Start), Cell(value.End)
  fault_record(configuration, Shape_Fault(kind))
}

// Checks that no assertion opt-out repeats an instrumentation entry exactly: an instrumentation
// package is exempt already, and a repeat hides which list owns it. A broader glob that covers one
// is no repeat.
func exemptions_check(configuration Configuration_Handle) {
  Configuration_Handle_Invariants(configuration, "exemptions_check.configuration")
  cells, text := configuration.Cells, configuration.Text
  values := configuration.Values[:cells[CELL_VALUE]]
  for slot, exemption := range values {
    if exemption.Key != KEY_ASSERTION_EXEMPT {
      continue
    }
    for _, entry := range values {
      if entry.Key != KEY_INSTRUMENTATION_PACKAGES {
        continue
      }
      if string(text[exemption.Start:exemption.End]) != string(text[entry.Start:entry.End]) {
        continue
      }
      cells[CELL_ENTRY], cells[CELL_KEY] = Cell(slot), Cell(KEY_ASSERTION_EXEMPT)
      entry_fault(configuration, Entry_Fault(FAULT_ASSERTION_REPEATED))
      return
    }
  }
}

// Checks each glob entry of the list under the key the decoder stands on, then that the list
// holds an entry past its negations: a negation revokes what an earlier entry states, thus a list
// of negations alone reaches no path.
func list_check(configuration Configuration_Handle) {
  Configuration_Handle_Invariants(configuration, "list_check.configuration")
  cells, text := configuration.Cells, configuration.Text
  entries, negated := 0, 0
  for slot, value := range configuration.Values[:cells[CELL_VALUE]] {
    if value.Key != Key(cells[CELL_KEY]) {
      continue
    }
    entries++
    cells[CELL_ENTRY] = Cell(slot)
    entry_check(configuration)
    if cells[CELL_FAULT] != Cell(FAULT_NONE) {
      return
    }
    // An entry past the check holds a byte, since one of none is blank.
    if text[value.Start] == '!' {
      negated++
    }
  }
  if entries == 0 {
    return
  }
  if negated == entries {
    fault_record(configuration, Shape_Fault(FAULT_GLOB_NEGATED))
  }
}

// Checks one glob entry as lint does: a blank entry, a ! that negates nothing, the ? and [ tokens
// the matcher does not support, and a segment that ends in a lone backslash, which path.Match
// refuses as a malformed pattern.
func entry_check(configuration Configuration_Handle) {
  Configuration_Handle_Invariants(configuration, "entry_check.configuration")
  cells, text := configuration.Cells, configuration.Text
  value := configuration.Values[cells[CELL_ENTRY]]
  start, end := Cell(value.Start), Cell(value.End)
  cells[CELL_SPAN_START], cells[CELL_SPAN_END] = start, end
  blank_check(configuration)
  if cells[CELL_BLANK] == 1 {
    entry_fault(configuration, Entry_Fault(FAULT_GLOB_EMPTY))
    return
  }
  if text[start] == '!' {
    start++
    cells[CELL_SPAN_START] = start
    blank_check(configuration)
  }
  if cells[CELL_BLANK] == 1 {
    entry_fault(configuration, Entry_Fault(FAULT_GLOB_NEGATION))
    return
  }
  for _, symbol := range text[start:end] {
    switch symbol {
    case '?', '[':
      entry_fault(configuration, Entry_Fault(FAULT_GLOB_TOKEN))
      return
    }
  }
  // The matcher drops one trailing and one leading separator before it splits the segments.
  if text[end-1] == '/' {
    end--
  }
  if start < end {
    if text[start] == '/' {
      start++
    }
  }
  for index := start; index < end; index++ {
    if text[index] != BACKSLASH {
      continue
    }
    lone := index+1 == end
    if !lone {
      lone = text[index+1] == '/'
    }
    if lone {
      entry_fault(configuration, Entry_Fault(FAULT_GLOB_ESCAPE))
      return
    }
    // The backslash escapes the byte past it, which the scan steps over.
    index++
  }
}

// Reports in CELL_BLANK whether the text between CELL_SPAN_START and CELL_SPAN_END holds spaces
// alone, as Unicode states a space. The decoder wrote the text, thus it is valid UTF-8.
func blank_check(configuration Configuration_Handle) {
  Configuration_Handle_Invariants(configuration, "blank_check.configuration")
  cells, text := configuration.Cells, configuration.Text
  cells[CELL_BLANK] = 1
  end := cells[CELL_SPAN_END]
  for offset := cells[CELL_SPAN_START]; offset < end; {
    stop := min(offset+utf8.UTF_MAXIMUM, end)
    character, size := utf8.Decode_Character(utf8.Bytes(text[offset:stop]))
    space := false
    switch character {
    case ' ', '\t', '\n', '\v', '\f', '\r':
      space = true
    }
    if character > utf8.Decoded_Character(utf8.CHARACTER_ONE_MAXIMUM) {
      space = bool(ucd.Is_Space(ucd.Character(character)))
    }
    if !space {
      cells[CELL_BLANK] = 0
      return
    }
    offset += Cell(size)
  }
}

// Key_Text states the text of one key.
func Key_Text(key Key) (name Key_Name) {
  defer func() { Key_Name_Invariants(name, "key_text.name") }()
  Key_Invariants(key, "key_text.key")
  switch key {
  case KEY_SHARED_COMPONENT:
    return "shared_component"
  case KEY_INSTRUMENTATION_PACKAGES:
    return "instrumentation_packages"
  case KEY_WORD_REPLACEMENTS:
    return "word_replacements"
  case KEY_IGNORE:
    return "ignore"
  case KEY_ASSERTION_EXEMPT:
    return "opt_out_assertion_mandate_packages"
  case KEY_RECURSION_EXEMPT:
    return "opt_out_recursion_ban"
  }
  return "opt_out_interface_ban"
}

// Fault_Text states what one fault is. A keyed fault states it past the key it names.
func Fault_Text(kind Recorded_Fault) (message Fault_Message) {
  defer func() { Fault_Message_Invariants(message, "fault_text.message") }()
  Recorded_Fault_Invariants(kind, "fault_text.kind")
  switch Fault_Kind(kind) {
  case FAULT_SYNTAX:
    return "invalid JSON"
  case FAULT_OVERSIZED:
    return "the file is larger than 65536 bytes"
  case FAULT_FULL:
    return "more than 4096 strings"
  case FAULT_OBJECT:
    return "expected an object"
  case FAULT_KEY_UNKNOWN:
    return "unknown key"
  case FAULT_KEY_REPEATED:
    return "is repeated"
  case FAULT_TYPE:
    return "holds a value of another type"
  case FAULT_KEY_ABSENT:
    return MESSAGE_KEY_ABSENT
  case FAULT_EMPTY:
    return MESSAGE_EMPTY
  case FAULT_GLOB_EMPTY:
    return "entry is empty"
  case FAULT_GLOB_NEGATION:
    return "entry negates nothing"
  case FAULT_GLOB_TOKEN:
    return "entry holds ? or [; write * or **"
  case FAULT_GLOB_ESCAPE:
    return "entry ends a segment in a lone backslash"
  case FAULT_GLOB_NEGATED:
    return "holds negation entries alone"
  }
  return MESSAGE_ASSERTION_REPEATED
}

// Fault_Keyed reports whether a fault names the key it stands under: every fault past an unknown
// key, since a known key is where each of them stands.
func Fault_Keyed(kind Recorded_Fault) (keyed Keyed) {
  defer func() { Keyed_Invariants(keyed, "fault_keyed.keyed") }()
  Recorded_Fault_Invariants(kind, "fault_keyed.kind")
  return Keyed(kind > Recorded_Fault(FAULT_KEY_UNKNOWN))
}

// Fault_Quoted reports whether a fault names the text it stands on: an unknown key, and a glob
// entry.
func Fault_Quoted(kind Recorded_Fault) (quoted Quoted) {
  defer func() { Quoted_Invariants(quoted, "fault_quoted.quoted") }()
  Recorded_Fault_Invariants(kind, "fault_quoted.kind")
  switch Fault_Kind(kind) {
  case FAULT_KEY_UNKNOWN, FAULT_GLOB_EMPTY, FAULT_GLOB_NEGATION, FAULT_GLOB_TOKEN:
    return true
  case FAULT_GLOB_ESCAPE, FAULT_ASSERTION_REPEATED:
    return true
  }
  return false
}

// LIST_COUNT is how many path lists the file holds.
const LIST_COUNT = 5

// Listed is whether Entry_Next found an entry.
type Listed bool

// Listed_Invariants proves both answers occur.
func Listed_Invariants(value Listed, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Sometimes(bool(value), "A configuration list holds a next entry.").
    Ensure()
}

// Entry_Next moves CELL_ENTRY to the next entry of the path lists in the order lint checks them,
// and reports whether one stands. Parse opens the walk at the first list.
func Entry_Next(configuration Configuration_Handle) (found Listed) {
  defer func() { Listed_Invariants(found, "entry_next.found") }()
  Configuration_Handle_Invariants(configuration, "entry_next.configuration")
  cells := configuration.Cells
  lists := [LIST_COUNT]Key{
    KEY_IGNORE,
    KEY_INSTRUMENTATION_PACKAGES,
    KEY_ASSERTION_EXEMPT,
    KEY_RECURSION_EXEMPT,
    KEY_INTERFACE_EXEMPT,
  }
  // Each step reads one slot or turns past one list, thus the steps are bounded.
  for range (LIST_COUNT + 1) * (VALUE_COUNT_MAXIMUM + 1) {
    if cells[CELL_LIST] == LIST_COUNT {
      return false
    }
    if cells[CELL_SLOT] == cells[CELL_VALUE] {
      cells[CELL_SLOT] = 0
      cells[CELL_LIST]++
      continue
    }
    slot := cells[CELL_SLOT]
    cells[CELL_SLOT]++
    if configuration.Values[slot].Key == lists[cells[CELL_LIST]] {
      cells[CELL_ENTRY] = slot
      return true
    }
  }
  return false
}
