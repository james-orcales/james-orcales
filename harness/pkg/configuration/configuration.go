// Package configuration decodes harness.json, the policy file that holds the shape of lint.json,
// into stores the caller owns. It reads no file: the caller reads the bytes and hands them over,
// and the decoder states the first fault it meets at the byte it stands on.
package configuration

import "local/james-orcales/shared/sim/aver/default"
import "local/james-orcales/shared/unicode/ucd"
import "local/james-orcales/shared/unicode/utf8"

// SOURCE_SIZE_MAXIMUM bounds the file Parse decodes. The file holds a handful of lists and the
// word table, thus the bound only stops a file no author wrote.
const SOURCE_SIZE_MAXIMUM = 64 * 1024

// SOURCE_STORAGE_SIZE is one byte past the bound: a source that fills it proves itself past the
// bound.
const SOURCE_STORAGE_SIZE = SOURCE_SIZE_MAXIMUM + 1

// SOURCE_SIZE_MINIMUM is the empty source, which states an unfinished object.
const SOURCE_SIZE_MINIMUM = 0

// VALUE_COUNT_MAXIMUM bounds the strings one file holds, keys aside.
const VALUE_COUNT_MAXIMUM = 4096

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

// Text_Offset is where one value stands in the text store.
type Text_Offset int

// Text_Offset_Invariants bounds the offset by the text store.
func Text_Offset_Invariants(value Text_Offset, namespace aver.Namespace) {
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
  Start Text_Offset
  // End is one past the last byte of the text.
  End Text_Offset
  // Opening is the quote the string opens on in the source.
  Opening Source_Offset
}

// Value_Invariants composes one value.
func Value_Invariants(value Value, namespace aver.Namespace) {
  Key_Invariants(value.Key, namespace)
  Word_Slot_Invariants(value.Word, namespace)
  Text_Offset_Invariants(value.Start, namespace)
  Text_Offset_Invariants(value.End, namespace)
  Source_Offset_Invariants(value.Opening, namespace)
}

// Source_Offset is one byte of the source.
type Source_Offset int

// Source_Offset_Invariants bounds the offset by the source.
func Source_Offset_Invariants(value Source_Offset, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(int(value), OFFSET_MINIMUM, SOURCE_SIZE_MAXIMUM).
    Ensure()
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

// Source_Store holds the source while Parse reads it.
type Source_Store []byte

// Source_Store_Invariants fixes the store.
func Source_Store_Invariants(value Source_Store, _ aver.Namespace) {
  aver.Always(
    len(value) == SOURCE_STORAGE_SIZE,
    "A configuration source store has complete length.",
  )
  aver.Always(cap(value) == SOURCE_STORAGE_SIZE, "A configuration source store cannot grow.")
}

// CELL_MINIMUM is the least a slot holds: a word slot that names no word.
const CELL_MINIMUM = -1

// CELL_MAXIMUM is the most a slot holds: the widest character an escape states.
const CELL_MAXIMUM = 0x10ffff

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

// CELL_OFFSET is the byte of the source the decoder stands on.
const CELL_OFFSET = 0

// CELL_SIZE is how many bytes the source holds.
const CELL_SIZE = 1

// CELL_TEXT is how many bytes of the text store the values hold.
const CELL_TEXT = 2

// CELL_END is one past the last byte of the string being decoded.
const CELL_END = 3

// CELL_VALUE is how many values the table holds.
const CELL_VALUE = 4

// CELL_PRESENT holds one bit for each key the object stated.
const CELL_PRESENT = 5

// CELL_KEY is the key whose value the decoder reads.
const CELL_KEY = 6

// CELL_WORD is the word the replacements being read replace, or WORD_NONE.
const CELL_WORD = 7

// CELL_ESCAPE is the backslash the escape being read opens on.
const CELL_ESCAPE = 8

// CELL_CHARACTER is the character one Unicode escape states.
const CELL_CHARACTER = 9

// CELL_CLOSE is the brace that closes the object.
const CELL_CLOSE = 10

// CELL_FAULT is the first fault.
const CELL_FAULT = 11

// CELL_FAULT_OFFSET is the byte the first fault stands on.
const CELL_FAULT_OFFSET = 12

// CELL_FAULT_KEY is the key an absent-key fault names.
const CELL_FAULT_KEY = 13

// CELL_HIGH is the high half of the surrogate pair being read.
const CELL_HIGH = 14

// CELL_ENTRY is the slot of the glob entry being checked.
const CELL_ENTRY = 15

// CELL_SPAN_START is the first byte of the text a blank check reads.
const CELL_SPAN_START = 16

// CELL_SPAN_END is one past the last byte of the text a blank check reads.
const CELL_SPAN_END = 17

// CELL_BLANK is 1 when the text a blank check read holds spaces alone.
const CELL_BLANK = 18

// CELL_LIST is the list of the order lint checks them in that Entry_Next stands on.
const CELL_LIST = 19

// CELL_SLOT is the next value slot Entry_Next reads.
const CELL_SLOT = 20

// CELL_COUNT is how many slots the state holds.
const CELL_COUNT = 21

// Configuration is the decoded file and the stores Parse reads it through.
type Configuration struct {
  // Source holds the file while Parse reads it.
  Source Source_Store
  // Text holds the decoded strings.
  Text Text
  // Values names each decoded string.
  Values Values
  // Cells holds the decoder's state.
  Cells Cells
}

// Configuration_Invariants composes the stores.
func Configuration_Invariants(value Configuration, namespace aver.Namespace) {
  Source_Store_Invariants(value.Source, namespace)
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
  Configuration_Invariants(*value, namespace)
}

// FAULT_NONE is a file that decodes whole.
const FAULT_NONE Fault_Kind = 0

// FAULT_UNFINISHED is a source that ends inside a value.
const FAULT_UNFINISHED Fault_Kind = 1

// FAULT_OBJECT is a byte where an object opens.
const FAULT_OBJECT Fault_Kind = 2

// FAULT_ARRAY is a byte where an array opens.
const FAULT_ARRAY Fault_Kind = 3

// FAULT_STRING is a byte where a string opens.
const FAULT_STRING Fault_Kind = 4

// FAULT_COLON is a byte where a colon stands.
const FAULT_COLON Fault_Kind = 5

// FAULT_COMMA is a byte where a comma or a closing bracket stands.
const FAULT_COMMA Fault_Kind = 6

// FAULT_SURPLUS is a byte past the object.
const FAULT_SURPLUS Fault_Kind = 7

// FAULT_CONTROL is a byte below a space inside a string.
const FAULT_CONTROL Fault_Kind = 8

// FAULT_ESCAPE is an escape JSON does not state, or a surrogate without its pair.
const FAULT_ESCAPE Fault_Kind = 9

// FAULT_ENCODING is a byte that opens no UTF-8 character.
const FAULT_ENCODING Fault_Kind = 10

// FAULT_KEY_UNKNOWN is a key the shape does not hold.
const FAULT_KEY_UNKNOWN Fault_Kind = 11

// FAULT_KEY_REPEATED is a key the object already stated.
const FAULT_KEY_REPEATED Fault_Kind = 12

// FAULT_KEY_ABSENT is a key the object never stated.
const FAULT_KEY_ABSENT Fault_Kind = 13

// FAULT_COMPONENT_EMPTY is a shared_component of no byte.
const FAULT_COMPONENT_EMPTY Fault_Kind = 14

// FAULT_WORDS_EMPTY is a word_replacements of no word.
const FAULT_WORDS_EMPTY Fault_Kind = 15

// FAULT_OVERSIZED is a source past SOURCE_SIZE_MAXIMUM.
const FAULT_OVERSIZED Fault_Kind = 16

// FAULT_FULL is a string past the value table.
const FAULT_FULL Fault_Kind = 17

// FAULT_GLOB_EMPTY is a glob entry of spaces alone.
const FAULT_GLOB_EMPTY Fault_Kind = 18

// FAULT_GLOB_NEGATION is a glob entry whose ! negates nothing.
const FAULT_GLOB_NEGATION Fault_Kind = 19

// FAULT_GLOB_TOKEN is a glob entry holding ? or [, which the matcher does not support.
const FAULT_GLOB_TOKEN Fault_Kind = 20

// FAULT_GLOB_ESCAPE is a glob entry with a segment that ends in a lone backslash.
const FAULT_GLOB_ESCAPE Fault_Kind = 21

// FAULT_GLOB_NEGATED is a glob list of negations alone, which no entry of it can reach.
const FAULT_GLOB_NEGATED Fault_Kind = 22

// FAULT_ASSERTION_REPEATED is an assertion opt-out equal to an instrumentation entry, which the
// assertion mandate already exempts.
const FAULT_ASSERTION_REPEATED Fault_Kind = 23

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
    int(FAULT_UNFINISHED),
    int(FAULT_ASSERTION_REPEATED),
  ).
    Ensure()
}

// Fault_Offset is the byte one fault stands on.
type Fault_Offset int

// Fault_Offset_Invariants bounds the offset by the source.
func Fault_Offset_Invariants(value Fault_Offset, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(int(value), OFFSET_MINIMUM, SOURCE_SIZE_MAXIMUM).
    Ensure()
}

// Fault is the first fault of a file: what it is, the byte it stands on, and the key an absent
// key names.
type Fault struct {
  // Kind is what the fault is.
  Kind Fault_Kind
  // Offset is the byte the fault stands on.
  Offset Fault_Offset
  // Key is the key an absent key names.
  Key Key
}

// Fault_Invariants composes one fault.
func Fault_Invariants(value Fault, namespace aver.Namespace) {
  Fault_Kind_Invariants(value.Kind, namespace)
  Fault_Offset_Invariants(value.Offset, namespace)
  Key_Invariants(value.Key, namespace)
}

// FAULT_MESSAGE_SIZE_MINIMUM is the shortest fault message.
const FAULT_MESSAGE_SIZE_MINIMUM = len(MESSAGE_KEY_ABSENT)

// FAULT_MESSAGE_SIZE_MAXIMUM is the longest fault message.
const FAULT_MESSAGE_SIZE_MAXIMUM = len(MESSAGE_ASSERTION_REPEATED)

// MESSAGE_KEY_ABSENT states a key the object never stated, past the key.
const MESSAGE_KEY_ABSENT = "is absent"

// MESSAGE_ASSERTION_REPEATED states an assertion opt-out that repeats an instrumentation entry.
const MESSAGE_ASSERTION_REPEATED = "entry repeats an instrumentation_packages entry"

// MESSAGE_GLOB_ESCAPE states a glob entry with a segment that ends in a lone backslash.
const MESSAGE_GLOB_ESCAPE = "entry ends a segment in a lone backslash"

// Keyed is whether a fault names the key it stands under.
type Keyed bool

// Keyed_Invariants proves both kinds of fault occur.
func Keyed_Invariants(value Keyed, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Sometimes(bool(value), "A configuration fault names its key.").
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

// TOKEN_COLON separates a key from its value.
const TOKEN_COLON Token = ':'

// TOKEN_ARRAY opens an array.
const TOKEN_ARRAY Token = '['

// TOKEN_OBJECT opens an object.
const TOKEN_OBJECT Token = '{'

// Token is one byte the shape requires where the decoder stands.
type Token int

// Token_Invariants admits each token.
func Token_Invariants(value Token, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_3_Int(
    int(value),
    int(TOKEN_COLON),
    int(TOKEN_ARRAY),
    int(TOKEN_OBJECT),
  ).
    Ensure()
}

// BACKSLASH opens an escape.
const BACKSLASH = 0x5c

// QUOTE opens and closes a string.
const QUOTE = '"'

// SPACE is the first byte a string holds unescaped.
const SPACE = 0x20

// SURROGATE_HIGH_MINIMUM opens the high half of a surrogate pair.
const SURROGATE_HIGH_MINIMUM = 0xd800

// SURROGATE_LOW_MINIMUM opens the low half of a surrogate pair.
const SURROGATE_LOW_MINIMUM = 0xdc00

// SURROGATE_LOW_MAXIMUM closes the low half of a surrogate pair.
const SURROGATE_LOW_MAXIMUM = 0xdfff

// SURROGATE_BASE is the first character a surrogate pair states.
const SURROGATE_BASE = 0x10000

// SURROGATE_SPAN is how many characters one high surrogate spans.
const SURROGATE_SPAN = 0x400

// HEXADECIMAL_DIGIT_COUNT is the digits of one Unicode escape.
const HEXADECIMAL_DIGIT_COUNT = 4

// Parse decodes the source into the configuration's stores and states the first fault, or
// FAULT_NONE. A value stands in the table only up to the fault.
func Parse(configuration Configuration_Handle, source Source) (fault Fault) {
  defer func() { Fault_Invariants(fault, "parse.fault") }()
  Configuration_Handle_Invariants(configuration, "parse.configuration")
  Source_Invariants(source, "parse.source")
  numbers := configuration.Cells
  for index := range numbers {
    numbers[index] = 0
  }
  numbers[CELL_WORD] = Cell(WORD_NONE)
  numbers[CELL_SIZE] = Cell(copy(configuration.Source, source))
  if len(source) > SOURCE_SIZE_MAXIMUM {
    numbers[CELL_OFFSET] = SOURCE_SIZE_MAXIMUM
    fault_record(configuration, Recorded_Fault(FAULT_OVERSIZED))
  }
  if numbers[CELL_FAULT] == Cell(FAULT_NONE) {
    object_parse(configuration)
  }
  if numbers[CELL_FAULT] == Cell(FAULT_NONE) {
    object_close(configuration)
  }
  return Fault{
    Kind:   Fault_Kind(numbers[CELL_FAULT]),
    Offset: Fault_Offset(numbers[CELL_FAULT_OFFSET]),
    Key:    Key(numbers[CELL_FAULT_KEY]),
  }
}

// Records the fault at the byte the decoder stands on.
func fault_record(configuration Configuration_Handle, kind Recorded_Fault) {
  Configuration_Handle_Invariants(configuration, "fault_record.configuration")
  Recorded_Fault_Invariants(kind, "fault_record.kind")
  numbers := configuration.Cells
  numbers[CELL_FAULT] = Cell(kind)
  numbers[CELL_FAULT_OFFSET] = numbers[CELL_OFFSET]
}

// Reads the object and each member it holds, up to the brace that closes it. Past the seventh
// member a key is unknown or repeated, thus the members the loop reads are bounded.
func object_parse(configuration Configuration_Handle) {
  Configuration_Handle_Invariants(configuration, "object_parse.configuration")
  numbers, source := configuration.Cells, configuration.Source
  token_expect(configuration, TOKEN_OBJECT)
  if numbers[CELL_FAULT] != Cell(FAULT_NONE) {
    return
  }
  space_skip(configuration)
  closed := numbers[CELL_OFFSET] < numbers[CELL_SIZE]
  if closed {
    closed = source[numbers[CELL_OFFSET]] == '}'
  }
  for range KEY_COUNT + 1 {
    if closed {
      numbers[CELL_CLOSE] = numbers[CELL_OFFSET]
      numbers[CELL_OFFSET]++
      return
    }
    member_parse(configuration)
    if numbers[CELL_FAULT] != Cell(FAULT_NONE) {
      return
    }
    closed = bool(member_close(configuration, TERMINATOR_OBJECT))
    if numbers[CELL_FAULT] != Cell(FAULT_NONE) {
      return
    }
  }
}

// Steps past the comma after one member, or reports the closing byte. A source that ends there,
// or holds another byte, is a fault.
func member_close(configuration Configuration_Handle, terminator Terminator) (closed Closed) {
  defer func() { Closed_Invariants(closed, "member_close.closed") }()
  Configuration_Handle_Invariants(configuration, "member_close.configuration")
  Terminator_Invariants(terminator, "member_close.terminator")
  numbers, source := configuration.Cells, configuration.Source
  space_skip(configuration)
  if numbers[CELL_OFFSET] == numbers[CELL_SIZE] {
    fault_record(configuration, Recorded_Fault(FAULT_UNFINISHED))
    return false
  }
  switch source[numbers[CELL_OFFSET]] {
  case byte(terminator):
    return true
  case ',':
    numbers[CELL_OFFSET]++
    space_skip(configuration)
    return false
  }
  fault_record(configuration, Recorded_Fault(FAULT_COMMA))
  return false
}

// TERMINATOR_ARRAY closes an array.
const TERMINATOR_ARRAY Terminator = ']'

// TERMINATOR_OBJECT closes an object.
const TERMINATOR_OBJECT Terminator = '}'

// Terminator is the byte that closes the container the decoder reads.
type Terminator int

// Terminator_Invariants admits each closing byte.
func Terminator_Invariants(value Terminator, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_Int(int(value), int(TERMINATOR_ARRAY), int(TERMINATOR_OBJECT)).
    Ensure()
}

// Closed is whether the container the decoder reads is closed.
type Closed bool

// Closed_Invariants proves both answers occur.
func Closed_Invariants(value Closed, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Sometimes(bool(value), "A container closes past a member.").
    Ensure()
}

// Reads one member: its key, the colon, and the value the key names.
func member_parse(configuration Configuration_Handle) {
  Configuration_Handle_Invariants(configuration, "member_parse.configuration")
  numbers := configuration.Cells
  numbers[CELL_WORD] = Cell(WORD_NONE)
  key_parse(configuration)
  if numbers[CELL_FAULT] != Cell(FAULT_NONE) {
    return
  }
  token_expect(configuration, TOKEN_COLON)
  if numbers[CELL_FAULT] != Cell(FAULT_NONE) {
    return
  }
  switch Key(numbers[CELL_KEY]) {
  case KEY_SHARED_COMPONENT:
    value_parse(configuration)
  case KEY_WORD_REPLACEMENTS:
    words_parse(configuration)
  default:
    list_parse(configuration)
  }
}

// Reads one key, which names a key of the shape the object has not stated yet.
func key_parse(configuration Configuration_Handle) {
  Configuration_Handle_Invariants(configuration, "key_parse.configuration")
  numbers, text := configuration.Cells, configuration.Text
  space_skip(configuration)
  opening := numbers[CELL_OFFSET]
  string_decode(configuration)
  if numbers[CELL_FAULT] != Cell(FAULT_NONE) {
    return
  }
  key := Cell(-1)
  switch string(text[numbers[CELL_TEXT]:numbers[CELL_END]]) {
  case "shared_component":
    key = Cell(KEY_SHARED_COMPONENT)
  case "instrumentation_packages":
    key = Cell(KEY_INSTRUMENTATION_PACKAGES)
  case "word_replacements":
    key = Cell(KEY_WORD_REPLACEMENTS)
  case "ignore":
    key = Cell(KEY_IGNORE)
  case "opt_out_assertion_mandate_packages":
    key = Cell(KEY_ASSERTION_EXEMPT)
  case "opt_out_recursion_ban":
    key = Cell(KEY_RECURSION_EXEMPT)
  case "opt_out_interface_ban":
    key = Cell(KEY_INTERFACE_EXEMPT)
  }
  if key < 0 {
    numbers[CELL_OFFSET] = opening
    fault_record(configuration, Recorded_Fault(FAULT_KEY_UNKNOWN))
    return
  }
  if numbers[CELL_PRESENT]&(1<<key) != 0 {
    numbers[CELL_OFFSET] = opening
    fault_record(configuration, Recorded_Fault(FAULT_KEY_REPEATED))
    return
  }
  numbers[CELL_PRESENT] |= 1 << key
  numbers[CELL_KEY] = key
}

// Reads one array of strings under the key the decoder stands on.
func list_parse(configuration Configuration_Handle) {
  Configuration_Handle_Invariants(configuration, "list_parse.configuration")
  numbers, source := configuration.Cells, configuration.Source
  token_expect(configuration, TOKEN_ARRAY)
  if numbers[CELL_FAULT] != Cell(FAULT_NONE) {
    return
  }
  space_skip(configuration)
  closed := numbers[CELL_OFFSET] < numbers[CELL_SIZE]
  if closed {
    closed = source[numbers[CELL_OFFSET]] == ']'
  }
  // Past the last slot a string is a fault, thus the strings the loop reads are bounded.
  for range VALUE_COUNT_MAXIMUM + 1 {
    if closed {
      numbers[CELL_OFFSET]++
      return
    }
    value_parse(configuration)
    if numbers[CELL_FAULT] != Cell(FAULT_NONE) {
      return
    }
    closed = bool(member_close(configuration, TERMINATOR_ARRAY))
    if numbers[CELL_FAULT] != Cell(FAULT_NONE) {
      return
    }
  }
}

// Reads the word table: each word, then the array of its replacements.
func words_parse(configuration Configuration_Handle) {
  Configuration_Handle_Invariants(configuration, "words_parse.configuration")
  numbers, source := configuration.Cells, configuration.Source
  token_expect(configuration, TOKEN_OBJECT)
  if numbers[CELL_FAULT] != Cell(FAULT_NONE) {
    return
  }
  space_skip(configuration)
  closed := numbers[CELL_OFFSET] < numbers[CELL_SIZE]
  if closed {
    closed = source[numbers[CELL_OFFSET]] == '}'
  }
  for range VALUE_COUNT_MAXIMUM + 1 {
    if closed {
      numbers[CELL_OFFSET]++
      return
    }
    numbers[CELL_WORD] = Cell(WORD_NONE)
    value_parse(configuration)
    if numbers[CELL_FAULT] != Cell(FAULT_NONE) {
      return
    }
    numbers[CELL_WORD] = numbers[CELL_VALUE] - 1
    token_expect(configuration, TOKEN_COLON)
    if numbers[CELL_FAULT] != Cell(FAULT_NONE) {
      return
    }
    list_parse(configuration)
    if numbers[CELL_FAULT] != Cell(FAULT_NONE) {
      return
    }
    closed = bool(member_close(configuration, TERMINATOR_OBJECT))
    if numbers[CELL_FAULT] != Cell(FAULT_NONE) {
      return
    }
  }
}

// Reads one string into the next slot of the value table.
func value_parse(configuration Configuration_Handle) {
  Configuration_Handle_Invariants(configuration, "value_parse.configuration")
  numbers := configuration.Cells
  space_skip(configuration)
  if numbers[CELL_VALUE] == VALUE_COUNT_MAXIMUM {
    fault_record(configuration, Recorded_Fault(FAULT_FULL))
    return
  }
  opening := numbers[CELL_OFFSET]
  string_decode(configuration)
  if numbers[CELL_FAULT] != Cell(FAULT_NONE) {
    return
  }
  configuration.Values[numbers[CELL_VALUE]] = Value{
    Key:     Key(numbers[CELL_KEY]),
    Word:    Word_Slot(numbers[CELL_WORD]),
    Start:   Text_Offset(numbers[CELL_TEXT]),
    End:     Text_Offset(numbers[CELL_END]),
    Opening: Source_Offset(opening),
  }
  numbers[CELL_TEXT] = numbers[CELL_END]
  numbers[CELL_VALUE]++
}

// Decodes one string into the text store past the values it holds. A string decodes to no more
// bytes than it spends, thus the store never runs out ahead of the source.
func string_decode(configuration Configuration_Handle) {
  Configuration_Handle_Invariants(configuration, "string_decode.configuration")
  numbers, source, text := configuration.Cells, configuration.Source, configuration.Text
  if numbers[CELL_OFFSET] == numbers[CELL_SIZE] {
    fault_record(configuration, Recorded_Fault(FAULT_UNFINISHED))
    return
  }
  if source[numbers[CELL_OFFSET]] != QUOTE {
    fault_record(configuration, Recorded_Fault(FAULT_STRING))
    return
  }
  numbers[CELL_OFFSET]++
  numbers[CELL_END] = numbers[CELL_TEXT]
  for range SOURCE_STORAGE_SIZE {
    if numbers[CELL_OFFSET] == numbers[CELL_SIZE] {
      fault_record(configuration, Recorded_Fault(FAULT_UNFINISHED))
      return
    }
    value := source[numbers[CELL_OFFSET]]
    if value == QUOTE {
      numbers[CELL_OFFSET]++
      return
    }
    switch {
    case value == BACKSLASH:
      escape_decode(configuration)
    case value < SPACE:
      fault_record(configuration, Recorded_Fault(FAULT_CONTROL))
    case value < byte(utf8.CHARACTER_SELF):
      text[numbers[CELL_END]] = value
      numbers[CELL_END]++
      numbers[CELL_OFFSET]++
    default:
      character_copy(configuration)
    }
    if numbers[CELL_FAULT] != Cell(FAULT_NONE) {
      return
    }
  }
}

// Copies one UTF-8 character past ASCII into the text store. An invalid sequence is a fault at
// its first byte.
func character_copy(configuration Configuration_Handle) {
  Configuration_Handle_Invariants(configuration, "character_copy.configuration")
  numbers, source, text := configuration.Cells, configuration.Source, configuration.Text
  offset := numbers[CELL_OFFSET]
  end := min(offset+utf8.UTF_MAXIMUM, numbers[CELL_SIZE])
  character, size := utf8.Decode_Character(utf8.Bytes(source[offset:end]))
  if size == utf8.CHARACTER_SIZE_MINIMUM {
    if character == utf8.REPLACEMENT_CHARACTER {
      fault_record(configuration, Recorded_Fault(FAULT_ENCODING))
      return
    }
  }
  written := copy(text[numbers[CELL_END]:], source[offset:offset+Cell(size)])
  numbers[CELL_END] += Cell(written)
  numbers[CELL_OFFSET] += Cell(size)
}

// Decodes one escape. A fault in it stands at its backslash.
func escape_decode(configuration Configuration_Handle) {
  Configuration_Handle_Invariants(configuration, "escape_decode.configuration")
  numbers, source, text := configuration.Cells, configuration.Source, configuration.Text
  numbers[CELL_ESCAPE] = numbers[CELL_OFFSET]
  if numbers[CELL_OFFSET]+1 == numbers[CELL_SIZE] {
    fault_record(configuration, Recorded_Fault(FAULT_ESCAPE))
    return
  }
  value := source[numbers[CELL_OFFSET]+1]
  switch value {
  case QUOTE, BACKSLASH, '/':
  case 'b':
    value = '\b'
  case 'f':
    value = '\f'
  case 'n':
    value = '\n'
  case 'r':
    value = '\r'
  case 't':
    value = '\t'
  case 'u':
    numbers[CELL_OFFSET] += 2
    unicode_decode(configuration)
    return
  default:
    fault_record(configuration, Recorded_Fault(FAULT_ESCAPE))
    return
  }
  text[numbers[CELL_END]] = value
  numbers[CELL_END]++
  numbers[CELL_OFFSET] += 2
}

// Decodes the digits of one Unicode escape, and the low half of a pair a high surrogate opens.
func unicode_decode(configuration Configuration_Handle) {
  Configuration_Handle_Invariants(configuration, "unicode_decode.configuration")
  numbers, source := configuration.Cells, configuration.Source
  hexadecimal_read(configuration)
  if numbers[CELL_FAULT] != Cell(FAULT_NONE) {
    return
  }
  high := numbers[CELL_CHARACTER]
  paired := high > SURROGATE_HIGH_MINIMUM-1
  if paired {
    paired = high < SURROGATE_LOW_MINIMUM
  }
  lone := high > SURROGATE_LOW_MINIMUM-1
  if lone {
    lone = high < SURROGATE_LOW_MAXIMUM+1
  }
  if paired {
    // The low half opens with its own backslash and u.
    lone = numbers[CELL_OFFSET]+2 > numbers[CELL_SIZE]
    if !lone {
      lone = source[numbers[CELL_OFFSET]] != BACKSLASH
    }
    if !lone {
      lone = source[numbers[CELL_OFFSET]+1] != 'u'
    }
  }
  if lone {
    escape_fault(configuration)
    return
  }
  if paired {
    numbers[CELL_OFFSET] += 2
    hexadecimal_read(configuration)
    numbers[CELL_HIGH] = high
    surrogate_join(configuration)
  }
  if numbers[CELL_FAULT] != Cell(FAULT_NONE) {
    return
  }
  character_write(configuration)
}

// States a fault at the backslash the escape opens on.
func escape_fault(configuration Configuration_Handle) {
  Configuration_Handle_Invariants(configuration, "escape_fault.configuration")
  numbers := configuration.Cells
  numbers[CELL_OFFSET] = numbers[CELL_ESCAPE]
  fault_record(configuration, Recorded_Fault(FAULT_ESCAPE))
}

// Joins a high surrogate and the low half the decoder read into the character they state.
func surrogate_join(configuration Configuration_Handle) {
  Configuration_Handle_Invariants(configuration, "surrogate_join.configuration")
  numbers := configuration.Cells
  if numbers[CELL_FAULT] != Cell(FAULT_NONE) {
    return
  }
  low := numbers[CELL_CHARACTER]
  if low < SURROGATE_LOW_MINIMUM {
    escape_fault(configuration)
    return
  }
  if low > SURROGATE_LOW_MAXIMUM {
    escape_fault(configuration)
    return
  }
  joined := SURROGATE_BASE + (numbers[CELL_HIGH]-SURROGATE_HIGH_MINIMUM)*SURROGATE_SPAN
  numbers[CELL_CHARACTER] = joined + low - SURROGATE_LOW_MINIMUM
}

// Reads the four hexadecimal digits of one Unicode escape.
func hexadecimal_read(configuration Configuration_Handle) {
  Configuration_Handle_Invariants(configuration, "hexadecimal_read.configuration")
  numbers, source := configuration.Cells, configuration.Source
  if numbers[CELL_OFFSET]+HEXADECIMAL_DIGIT_COUNT > numbers[CELL_SIZE] {
    escape_fault(configuration)
    return
  }
  character := Cell(0)
  for index := range HEXADECIMAL_DIGIT_COUNT {
    value := Cell(source[numbers[CELL_OFFSET]+Cell(index)])
    digit := Cell(-1)
    switch {
    case value < '0':
    case value < '9'+1:
      digit = value - '0'
    case value < 'A':
    case value < 'F'+1:
      digit = value - 'A' + 10
    case value < 'a':
    case value < 'f'+1:
      digit = value - 'a' + 10
    }
    if digit < 0 {
      escape_fault(configuration)
      return
    }
    character = character*16 + digit
  }
  numbers[CELL_CHARACTER] = character
  numbers[CELL_OFFSET] += HEXADECIMAL_DIGIT_COUNT
}

// Writes the character one escape states into the text store as UTF-8.
func character_write(configuration Configuration_Handle) {
  Configuration_Handle_Invariants(configuration, "character_write.configuration")
  numbers, text := configuration.Cells, configuration.Text
  end := numbers[CELL_END]
  character := utf8.Character(numbers[CELL_CHARACTER])
  size := utf8.Encode_Character(utf8.Bytes(text[end:end+utf8.UTF_MAXIMUM]), character)
  numbers[CELL_END] += Cell(size)
}

// Steps past the token the shape requires where the decoder stands.
func token_expect(configuration Configuration_Handle, token Token) {
  Configuration_Handle_Invariants(configuration, "token_expect.configuration")
  Token_Invariants(token, "token_expect.token")
  numbers, source := configuration.Cells, configuration.Source
  space_skip(configuration)
  if numbers[CELL_OFFSET] == numbers[CELL_SIZE] {
    fault_record(configuration, Recorded_Fault(FAULT_UNFINISHED))
    return
  }
  if source[numbers[CELL_OFFSET]] == byte(token) {
    numbers[CELL_OFFSET]++
    return
  }
  switch token {
  case TOKEN_OBJECT:
    fault_record(configuration, Recorded_Fault(FAULT_OBJECT))
  case TOKEN_ARRAY:
    fault_record(configuration, Recorded_Fault(FAULT_ARRAY))
  case TOKEN_COLON:
    fault_record(configuration, Recorded_Fault(FAULT_COLON))
  }
}

// Steps past the whitespace JSON admits between tokens.
func space_skip(configuration Configuration_Handle) {
  Configuration_Handle_Invariants(configuration, "space_skip.configuration")
  numbers, source := configuration.Cells, configuration.Source
  for numbers[CELL_OFFSET] < numbers[CELL_SIZE] {
    switch source[numbers[CELL_OFFSET]] {
    case ' ', '\t', '\n', '\r':
      numbers[CELL_OFFSET]++
    default:
      return
    }
  }
}

// Checks what the object states as a whole: nothing past it, each key, a shared component of at
// least one byte, and at least one word.
func object_close(configuration Configuration_Handle) {
  Configuration_Handle_Invariants(configuration, "object_close.configuration")
  numbers := configuration.Cells
  space_skip(configuration)
  if numbers[CELL_OFFSET] < numbers[CELL_SIZE] {
    fault_record(configuration, Recorded_Fault(FAULT_SURPLUS))
    return
  }
  numbers[CELL_OFFSET] = numbers[CELL_CLOSE]
  for key := range Cell(KEY_COUNT) {
    if numbers[CELL_PRESENT]&(1<<key) == 0 {
      numbers[CELL_FAULT_KEY] = key
      fault_record(configuration, Recorded_Fault(FAULT_KEY_ABSENT))
      return
    }
  }
  words, component := 0, 0
  for _, value := range configuration.Values[:numbers[CELL_VALUE]] {
    if value.Key == KEY_SHARED_COMPONENT {
      component = int(value.End - value.Start)
    }
    if value.Key == KEY_WORD_REPLACEMENTS {
      if value.Word == WORD_NONE {
        words++
      }
    }
  }
  if component == 0 {
    fault_record(configuration, Recorded_Fault(FAULT_COMPONENT_EMPTY))
    return
  }
  if words == 0 {
    fault_record(configuration, Recorded_Fault(FAULT_WORDS_EMPTY))
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
    numbers[CELL_KEY] = Cell(key)
    list_check(configuration)
    if numbers[CELL_FAULT] != Cell(FAULT_NONE) {
      return
    }
  }
  exemptions_check(configuration)
}

// Checks that no assertion opt-out repeats an instrumentation entry exactly: an instrumentation
// package is exempt already, and a repeat hides which list owns it. A broader glob that covers one
// is no repeat.
func exemptions_check(configuration Configuration_Handle) {
  Configuration_Handle_Invariants(configuration, "exemptions_check.configuration")
  cells, text := configuration.Cells, configuration.Text
  values := configuration.Values[:cells[CELL_VALUE]]
  for _, exemption := range values {
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
      cells[CELL_OFFSET], cells[CELL_FAULT_KEY] = Cell(exemption.Opening), Cell(
        KEY_ASSERTION_EXEMPT,
      )
      fault_record(configuration, Recorded_Fault(FAULT_ASSERTION_REPEATED))
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
  entries, negated, first := 0, 0, Cell(0)
  for slot, value := range configuration.Values[:cells[CELL_VALUE]] {
    if value.Key != Key(cells[CELL_KEY]) {
      continue
    }
    if entries == 0 {
      first = Cell(value.Opening)
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
    cells[CELL_OFFSET], cells[CELL_FAULT_KEY] = first, cells[CELL_KEY]
    fault_record(configuration, Recorded_Fault(FAULT_GLOB_NEGATED))
  }
}

// Checks one glob entry as lint does: a blank entry, a ! that negates nothing, the ? and [ tokens
// the matcher does not support, and a segment that ends in a lone backslash, which path.Match
// refuses as a malformed pattern.
func entry_check(configuration Configuration_Handle) {
  Configuration_Handle_Invariants(configuration, "entry_check.configuration")
  cells, text := configuration.Cells, configuration.Text
  value := configuration.Values[cells[CELL_ENTRY]]
  cells[CELL_OFFSET], cells[CELL_FAULT_KEY] = Cell(value.Opening), cells[CELL_KEY]
  start, end := Cell(value.Start), Cell(value.End)
  cells[CELL_SPAN_START], cells[CELL_SPAN_END] = start, end
  blank_check(configuration)
  if cells[CELL_BLANK] == 1 {
    fault_record(configuration, Recorded_Fault(FAULT_GLOB_EMPTY))
    return
  }
  if text[start] == '!' {
    start++
    cells[CELL_SPAN_START] = start
    blank_check(configuration)
  }
  if cells[CELL_BLANK] == 1 {
    fault_record(configuration, Recorded_Fault(FAULT_GLOB_NEGATION))
    return
  }
  for _, symbol := range text[start:end] {
    switch symbol {
    case '?', '[':
      fault_record(configuration, Recorded_Fault(FAULT_GLOB_TOKEN))
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
      fault_record(configuration, Recorded_Fault(FAULT_GLOB_ESCAPE))
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
  case FAULT_UNFINISHED:
    return "unfinished value"
  case FAULT_OBJECT:
    return "expected '{'"
  case FAULT_ARRAY:
    return "expected '['"
  case FAULT_STRING:
    return "expected a string"
  case FAULT_COLON:
    return "expected ':'"
  case FAULT_COMMA:
    return "expected ',' or a closing bracket"
  case FAULT_SURPLUS:
    return "trailing data"
  case FAULT_CONTROL:
    return "control byte in a string"
  case FAULT_ESCAPE:
    return "invalid escape"
  case FAULT_ENCODING:
    return "invalid UTF-8"
  case FAULT_KEY_UNKNOWN:
    return "unknown key"
  case FAULT_KEY_REPEATED:
    return "repeated key"
  case FAULT_KEY_ABSENT:
    return MESSAGE_KEY_ABSENT
  case FAULT_COMPONENT_EMPTY:
    return "shared_component is empty"
  case FAULT_WORDS_EMPTY:
    return "word_replacements is empty"
  case FAULT_OVERSIZED:
    return "the file is larger than 65536 bytes"
  case FAULT_FULL:
    return "more than 4096 strings"
  case FAULT_GLOB_EMPTY:
    return "entry is empty"
  case FAULT_GLOB_NEGATION:
    return "entry negates nothing"
  case FAULT_GLOB_TOKEN:
    return "entry holds ? or [; write * or **"
  case FAULT_GLOB_ESCAPE:
    return MESSAGE_GLOB_ESCAPE
  case FAULT_GLOB_NEGATED:
    return "holds negation entries alone"
  }
  return MESSAGE_ASSERTION_REPEATED
}

// Fault_Keyed reports whether a fault names the key it stands under: an absent key, and a glob
// entry or list.
func Fault_Keyed(kind Recorded_Fault) (keyed Keyed) {
  defer func() { Keyed_Invariants(keyed, "fault_keyed.keyed") }()
  Recorded_Fault_Invariants(kind, "fault_keyed.kind")
  if kind == Recorded_Fault(FAULT_KEY_ABSENT) {
    return true
  }
  return Keyed(kind > Recorded_Fault(FAULT_FULL))
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
