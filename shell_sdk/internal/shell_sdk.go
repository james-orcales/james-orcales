// Package shell_sdk is a walled-garden structured-data CLI: one multi-call
// binary, symlinked to bare verb names, passes typed values through pipes as a
// custom binary format and renders a table at a terminal. This is the pure
// library; package main binds the real IO.
package shell_sdk

import (
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"path"
	"strconv"
	"strings"
)

// The successful process exit code.
const EXIT_SUCCESS = 0

// The usage-error exit code, for a bad invocation.
const EXIT_USAGE = 2

// The runtime-failure exit code.
const EXIT_FAILURE = 1

// Value_Kind tags which variant a Value holds.
type Value_Kind int

// The null kind marks the absence of a value.
const VALUE_KIND_NULL Value_Kind = 0

// The boolean kind holds a true or false value.
const VALUE_KIND_BOOLEAN Value_Kind = 1

// The number kind holds a number as its exact source text, so no float64 is
// needed and the package stays deterministic.
const VALUE_KIND_NUMBER Value_Kind = 2

// The string kind holds a text value.
const VALUE_KIND_STRING Value_Kind = 3

// The list kind holds an ordered sequence of values.
const VALUE_KIND_LIST Value_Kind = 4

// The record kind holds an ordered set of named fields; a list of records is a
// table.
const VALUE_KIND_RECORD Value_Kind = 5

// Value is one typed cell flowing through the pipeline. Every stage reads and
// writes it; a table is simply a list of records with aligned keys.
type Value struct {
	// Kind selects which of the fields below carries the payload.
	Kind Value_Kind
	// Boolean carries a VALUE_KIND_BOOLEAN.
	Boolean bool
	// Number carries a VALUE_KIND_NUMBER as its exact source text.
	Number string
	// Text carries a VALUE_KIND_STRING.
	Text string
	// Items carries a VALUE_KIND_LIST.
	Items []Value
	// Fields carries a VALUE_KIND_RECORD in insertion order.
	Fields []Field
}

// Field is one named cell of a record.
type Field struct {
	// Name is the field's key.
	Name string
	// Value is the field's payload.
	Value Value
}

// Main_Input carries the command line and the host bindings Main needs. The
// library does no ambient IO, so stdin, the filesystem, the terminal check, and
// symlinking are injected here.
type Main_Input struct {
	// Arguments is the command line; index zero is the invoking link's name.
	Arguments []string
	// Output is where a verb's result is written.
	Output io.Writer
	// Error_Output is where errors are written.
	Error_Output io.Writer
	// Read_Stdin returns the bounded contents of standard input.
	Read_Stdin func() (data []byte)
	// Read_File returns a file's bounded contents, or an error.
	Read_File func(name string) (data []byte, err error)
	// Stdout_Is_Terminal reports whether output goes to a terminal.
	Stdout_Is_Terminal bool
	// Link points a destination path at the running binary, for install.
	Link func(destination string) (err error)
}

// Main dispatches on the invoking link's name, runs one verb, and returns a
// process exit code. The -install flag is handled before verb dispatch.
func Main(input *Main_Input) (status_code int) {
	if len(input.Arguments) == 0 {
		fmt.Fprintf(input.Error_Output, "shell_sdk: no verb name\n")
		return EXIT_USAGE
	}
	destination, is_install := install_target(input.Arguments)
	if is_install {
		return run_install(input, destination)
	}
	if wants_help(input.Arguments) {
		print_help(input.Output)
		return EXIT_SUCCESS
	}
	verb := path.Base(input.Arguments[0])
	kind, known := verb_kind(verb)
	if !known {
		print_help(input.Error_Output)
		return EXIT_USAGE
	}
	positional, mode := resolve_output_mode(input.Arguments[1:], input.Stdout_Is_Terminal)
	return run_and_write(input, kind, positional, mode)
}

// Verb_Kind identifies one of the walled-garden verbs.
type Verb_Kind int

// The load verb reads a file into the garden.
const VERB_KIND_LOAD Verb_Kind = 0

// The get verb navigates a cell path.
const VERB_KIND_GET Verb_Kind = 1

// The pick verb projects named fields.
const VERB_KIND_PICK Verb_Kind = 2

// The filter verb keeps matching rows.
const VERB_KIND_FILTER Verb_Kind = 3

// The first verb takes leading elements.
const VERB_KIND_FIRST Verb_Kind = 4

// The from verb parses a foreign format from stdin.
const VERB_KIND_FROM Verb_Kind = 5

// The to verb serializes out of the garden.
const VERB_KIND_TO Verb_Kind = 6

// The sort-by verb orders a list of records by a field.
const VERB_KIND_SORT_BY Verb_Kind = 7

// The group-by verb buckets records by a field value.
const VERB_KIND_GROUP_BY Verb_Kind = 8

// The distinct verb drops duplicate elements. Named distinct, not uniq, to
// avoid shadowing the system uniq binary once symlinked into a path.
const VERB_KIND_DISTINCT Verb_Kind = 9

// The reverse verb reverses a list.
const VERB_KIND_REVERSE Verb_Kind = 10

// The final verb takes trailing elements. Named final, not last, to avoid
// shadowing the system last binary once symlinked into a path.
const VERB_KIND_FINAL Verb_Kind = 11

// The columns verb lists a table's keys.
const VERB_KIND_COLUMNS Verb_Kind = 12

// The reject verb drops named fields.
const VERB_KIND_REJECT Verb_Kind = 13

// The relabel verb renames a field. Named relabel, not rename, to avoid
// shadowing the system rename binary once symlinked into a path.
const VERB_KIND_RELABEL Verb_Kind = 14

// The count verb counts elements or fields.
const VERB_KIND_COUNT Verb_Kind = 15

// The wrap verb wraps a value in a record.
const VERB_KIND_WRAP Verb_Kind = 16

// The flatten verb flattens one level of nested lists.
const VERB_KIND_FLATTEN Verb_Kind = 17

// Output_Mode selects how a verb presents its result.
type Output_Mode int

// The wire mode emits the binary format for the next sibling.
const OUTPUT_MODE_WIRE Output_Mode = 0

// The JSON mode emits text, leaving the garden.
const OUTPUT_MODE_JSON Output_Mode = 1

// The table mode renders an aligned table for a terminal.
const OUTPUT_MODE_TABLE Output_Mode = 2

// Runs one verb and writes its output, or reports the error and fails.
func run_and_write(
	input *Main_Input, kind Verb_Kind, arguments []string, mode Output_Mode,
) (status_code int) {
	output, run_err := run_verb(input, kind, arguments, mode)
	if run_err != nil {
		fmt.Fprintf(input.Error_Output, "shell_sdk: %v\n", run_err)
		return EXIT_FAILURE
	}
	_, write_err := input.Output.Write(output)
	if write_err != nil {
		return EXIT_FAILURE
	}
	return EXIT_SUCCESS
}

// Resolves a verb name to its kind. The names avoid shell keywords and commands.
func verb_kind(name string) (kind Verb_Kind, known bool) {
	switch name {
	case "load":
		return VERB_KIND_LOAD, true
	case "get":
		return VERB_KIND_GET, true
	case "pick":
		return VERB_KIND_PICK, true
	case "filter":
		return VERB_KIND_FILTER, true
	case "first":
		return VERB_KIND_FIRST, true
	case "from":
		return VERB_KIND_FROM, true
	case "to":
		return VERB_KIND_TO, true
	}
	return verb_kind_more(name)
}

// Resolves the second half of the verb names, keeping verb_kind within the
// per-function line budget.
func verb_kind_more(name string) (kind Verb_Kind, known bool) {
	switch name {
	case "sort-by":
		return VERB_KIND_SORT_BY, true
	case "group-by":
		return VERB_KIND_GROUP_BY, true
	case "distinct":
		return VERB_KIND_DISTINCT, true
	case "reverse":
		return VERB_KIND_REVERSE, true
	case "final":
		return VERB_KIND_FINAL, true
	case "columns":
		return VERB_KIND_COLUMNS, true
	case "reject":
		return VERB_KIND_REJECT, true
	case "relabel":
		return VERB_KIND_RELABEL, true
	case "count":
		return VERB_KIND_COUNT, true
	case "wrap":
		return VERB_KIND_WRAP, true
	case "flatten":
		return VERB_KIND_FLATTEN, true
	}
	return VERB_KIND_LOAD, false
}

// The verb names, for -install to fan the binary out; kept in step with
// verb_kind above.
func verb_names() (names []string) {
	return []string{
		"load", "get", "pick", "filter", "first", "from", "to",
		"sort-by", "group-by", "distinct", "reverse", "final", "columns",
		"reject", "relabel", "count", "wrap", "flatten",
	}
}

// Pulls the -install=<path> destination out of the arguments, if present.
func install_target(arguments []string) (destination string, is_install bool) {
	for argument_index := 0; argument_index < len(arguments); argument_index++ {
		suffix, found := strings.CutPrefix(arguments[argument_index], "-install=")
		if found {
			return suffix, true
		}
	}
	return "", false
}

// Reports whether the arguments ask for help.
func wants_help(arguments []string) (wants bool) {
	for argument_index := 0; argument_index < len(arguments); argument_index++ {
		if arguments[argument_index] == "-h" {
			return true
		}
		if arguments[argument_index] == "--help" {
			return true
		}
	}
	return false
}

// Writes the usage help: the verbs, filter operators, output flags, and the
// install syntax.
func print_help(output io.Writer) {
	fmt.Fprint(output, HELP_TEXT)
}

// The usage help text.
const HELP_TEXT = `shell_sdk: a structured-data toolkit, one binary symlinked to verb names,
passing typed values through pipes.

Invoke via a verb-named link, for example:
  load data.json | get users | filter age gt 30 | to json

Verbs:
  load <file>                  read a file into the garden
  from <format>                parse stdin as a foreign format (json)
  to <format>                  serialize out of the garden (json)
  get <path>                   navigate a dotted cell path
  pick <field>...              keep only the named fields
  filter <field> <op> <value>  keep rows where the cell compares
  first [count]                take the first element, or the first count

Filter operators: eq ne lt le gt ge
Output flags:     --json  --table   (default: table at a terminal, else binary)

Install: shell_sdk -install=<dir>   (symlinks every verb into <dir>; add to PATH)
`

// Fans the binary out into one verb-named link per verb under the destination.
func run_install(input *Main_Input, destination string) (status_code int) {
	names := verb_names()
	for name_index := 0; name_index < len(names); name_index++ {
		link_err := input.Link(destination + "/" + names[name_index])
		if link_err != nil {
			fmt.Fprintf(input.Error_Output, "shell_sdk: install: %v\n", link_err)
			return EXIT_FAILURE
		}
	}
	fmt.Fprintf(input.Output, "linked %d verbs into %s\n", len(names), destination)
	return EXIT_SUCCESS
}

// Splits the output-mode flags out of the arguments and resolves the mode: a
// forcing flag wins, then a table at a terminal, else the binary wire format.
func resolve_output_mode(
	arguments []string, stdout_is_terminal bool,
) (positional []string, mode Output_Mode) {
	forced := OUTPUT_MODE_WIRE
	forced_set := false
	positional = []string{}
	for argument_index := 0; argument_index < len(arguments); argument_index++ {
		argument := arguments[argument_index]
		if argument == "--json" {
			forced = OUTPUT_MODE_JSON
			forced_set = true
			continue
		}
		if argument == "--table" {
			forced = OUTPUT_MODE_TABLE
			forced_set = true
			continue
		}
		positional = append(positional, argument)
	}
	if forced_set {
		return positional, forced
	}
	if stdout_is_terminal {
		return positional, OUTPUT_MODE_TABLE
	}
	return positional, OUTPUT_MODE_WIRE
}

// Null_Value builds the null value.
func Null_Value() (value Value) {
	return Value{Kind: VALUE_KIND_NULL}
}

// Boolean_Value builds a boolean value.
func Boolean_Value(flag bool) (value Value) {
	return Value{Kind: VALUE_KIND_BOOLEAN, Boolean: flag}
}

// Number_Value builds a number from its exact source text.
func Number_Value(text string) (value Value) {
	return Value{Kind: VALUE_KIND_NUMBER, Number: text}
}

// String_Value builds a text value.
func String_Value(text string) (value Value) {
	return Value{Kind: VALUE_KIND_STRING, Text: text}
}

// List_Value builds a list from its items.
func List_Value(items []Value) (value Value) {
	return Value{Kind: VALUE_KIND_LIST, Items: items}
}

// Record_Value builds a record from its fields.
func Record_Value(fields []Field) (value Value) {
	return Value{Kind: VALUE_KIND_RECORD, Fields: fields}
}

// The magic that marks a stream as shell_sdk's own binary format.
const WIRE_MAGIC = "SSDK"

// The framing version, bumped on any incompatible change.
const WIRE_VERSION = 1

// One item on the encoder's stack: a value to emit, or a literal run of bytes
// such as an already-encoded field name.
type wire_work struct {
	Is_Value bool
	Value    Value
	Bytes    []byte
}

// One container the decoder is filling in from the byte stream.
type wire_frame struct {
	Kind            Value_Kind
	Remaining_Count int
	Items           []Value
	Fields          []Field
	Pending_Name    string
	Has_Name        bool
}

// One step read from the stream: a finished scalar or empty container, or a
// non-empty container that opens a new frame.
type wire_read struct {
	Is_Scalar bool
	Value     Value
	Is_Open   bool
	Kind      Value_Kind
	Count     int
}

// Wire_Encode serializes a value to a complete wire stream. The walk is
// iterative over an explicit stack, since recursion is banned.
func Wire_Encode(value Value) (encoded []byte) {
	output := []byte(WIRE_MAGIC)
	output = append(output, WIRE_VERSION)
	stack := []wire_work{{Is_Value: true, Value: value}}
	for len(stack) > 0 {
		top := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if !top.Is_Value {
			output = append(output, top.Bytes...)
			continue
		}
		header, children := wire_emit(top.Value)
		output = append(output, header...)
		stack = append(stack, children...)
	}
	return output
}

// Returns one value's header bytes and the child work to push, reversed so a
// LIFO pop yields the children in order.
func wire_emit(value Value) (header []byte, children []wire_work) {
	header = append(header, byte(value.Kind))
	switch value.Kind {
	case VALUE_KIND_BOOLEAN:
		header = append(header, wire_boolean_byte(value.Boolean))
	case VALUE_KIND_NUMBER:
		header = wire_append_bytes(header, value.Number)
	case VALUE_KIND_STRING:
		header = wire_append_bytes(header, value.Text)
	case VALUE_KIND_LIST:
		header = wire_append_count(header, len(value.Items))
		children = wire_items_work(value.Items)
	case VALUE_KIND_RECORD:
		header = wire_append_count(header, len(value.Fields))
		children = wire_fields_work(value.Fields)
	}
	return header, children
}

// Builds the child work for a list, reversed for LIFO order.
func wire_items_work(items []Value) (work []wire_work) {
	for item_index := len(items) - 1; item_index >= 0; item_index-- {
		work = append(work, wire_work{Is_Value: true, Value: items[item_index]})
	}
	return work
}

// Builds the child work for a record: a literal name run then the field value,
// reversed for LIFO order.
func wire_fields_work(fields []Field) (work []wire_work) {
	for field_index := len(fields) - 1; field_index >= 0; field_index-- {
		field := fields[field_index]
		work = append(work, wire_work{Is_Value: true, Value: field.Value})
		name := wire_append_bytes(nil, field.Name)
		work = append(work, wire_work{Is_Value: false, Bytes: name})
	}
	return work
}

// Encodes a boolean as one byte.
func wire_boolean_byte(flag bool) (encoded byte) {
	if flag {
		return 1
	}
	return 0
}

// Appends a length-prefixed run of text.
func wire_append_bytes(output []byte, content string) (result []byte) {
	result = wire_append_count(output, len(content))
	result = append(result, content...)
	return result
}

// Appends a little-endian unsigned 32-bit length.
func wire_append_count(output []byte, count int) (result []byte) {
	return append(output, byte(count), byte(count>>8), byte(count>>16), byte(count>>24))
}

// Wire_Decode rebuilds a value from a wire stream using an explicit frame stack
// rather than recursion.
func Wire_Decode(data []byte) (value Value, err error) {
	at, header_err := wire_header(data)
	if header_err != nil {
		return Value{}, header_err
	}
	frames := []wire_frame{}
	for at < len(data) {
		if wire_needs_name(frames) {
			name, after, name_err := wire_read_blob(data, at)
			if name_err != nil {
				return Value{}, name_err
			}
			at = after
			frames = wire_set_name(frames, name)
			continue
		}
		read, after, read_err := wire_read_value(data, at)
		if read_err != nil {
			return Value{}, read_err
		}
		at = after
		updated, root, done := wire_place(frames, read)
		frames = updated
		if done {
			return root, nil
		}
	}
	return Value{}, fmt.Errorf("wire: stream did not terminate")
}

// Validates the magic and version, returning the payload offset.
func wire_header(data []byte) (at int, err error) {
	if len(data) < len(WIRE_MAGIC)+1 {
		return 0, fmt.Errorf("wire: stream too short")
	}
	if string(data[:len(WIRE_MAGIC)]) != WIRE_MAGIC {
		return 0, fmt.Errorf("wire: not a shell_sdk stream")
	}
	if data[len(WIRE_MAGIC)] != WIRE_VERSION {
		return 0, fmt.Errorf("wire: unsupported version")
	}
	return len(WIRE_MAGIC) + 1, nil
}

// Reports whether the open record wants a field name next.
func wire_needs_name(frames []wire_frame) (needs bool) {
	if len(frames) == 0 {
		return false
	}
	top := frames[len(frames)-1]
	if top.Kind != VALUE_KIND_RECORD {
		return false
	}
	if top.Remaining_Count == 0 {
		return false
	}
	return !top.Has_Name
}

// Records the pending field name on the open record frame.
func wire_set_name(frames []wire_frame, name string) (updated []wire_frame) {
	top := frames[len(frames)-1]
	top.Pending_Name = name
	top.Has_Name = true
	frames[len(frames)-1] = top
	return frames
}

// Reads one value head: a finished scalar, or an opening container.
func wire_read_value(data []byte, at int) (read wire_read, next int, err error) {
	if at >= len(data) {
		return wire_read{}, at, fmt.Errorf("wire: truncated value")
	}
	kind := Value_Kind(data[at])
	body := at + 1
	switch kind {
	case VALUE_KIND_NULL:
		return wire_read{Is_Scalar: true, Value: Null_Value()}, body, nil
	case VALUE_KIND_BOOLEAN:
		return wire_read_boolean(data, body)
	case VALUE_KIND_NUMBER:
		return wire_read_text(data, body, VALUE_KIND_NUMBER)
	case VALUE_KIND_STRING:
		return wire_read_text(data, body, VALUE_KIND_STRING)
	case VALUE_KIND_LIST:
		return wire_read_container(data, body, VALUE_KIND_LIST)
	case VALUE_KIND_RECORD:
		return wire_read_container(data, body, VALUE_KIND_RECORD)
	}
	return wire_read{}, at, fmt.Errorf("wire: unknown tag %d", data[at])
}

// Reads a one-byte boolean.
func wire_read_boolean(data []byte, at int) (read wire_read, next int, err error) {
	if at >= len(data) {
		return wire_read{}, at, fmt.Errorf("wire: truncated boolean")
	}
	built := Boolean_Value(data[at] != 0)
	return wire_read{Is_Scalar: true, Value: built}, at + 1, nil
}

// Reads a length-prefixed number or string.
func wire_read_text(data []byte, at int, kind Value_Kind) (read wire_read, next int, err error) {
	text, after, blob_err := wire_read_blob(data, at)
	if blob_err != nil {
		return wire_read{}, at, blob_err
	}
	built := Number_Value(text)
	if kind == VALUE_KIND_STRING {
		built = String_Value(text)
	}
	return wire_read{Is_Scalar: true, Value: built}, after, nil
}

// Reads a container head: an empty one finishes as a scalar, a non-empty one
// opens a frame.
func wire_read_container(
	data []byte, at int, kind Value_Kind,
) (read wire_read, next int, err error) {
	count, after, count_err := wire_read_count(data, at)
	if count_err != nil {
		return wire_read{}, at, count_err
	}
	if count == 0 {
		return wire_read{Is_Scalar: true, Value: wire_empty(kind)}, after, nil
	}
	return wire_read{Is_Open: true, Kind: kind, Count: count}, after, nil
}

// Builds an empty container value.
func wire_empty(kind Value_Kind) (value Value) {
	if kind == VALUE_KIND_LIST {
		return List_Value([]Value{})
	}
	return Record_Value([]Field{})
}

// Reads a length-prefixed run of bytes as text.
func wire_read_blob(data []byte, at int) (text string, next int, err error) {
	count, after, count_err := wire_read_count(data, at)
	if count_err != nil {
		return "", at, count_err
	}
	if after+count > len(data) {
		return "", at, fmt.Errorf("wire: truncated bytes")
	}
	return string(data[after : after+count]), after + count, nil
}

// Reads a little-endian unsigned 32-bit length.
func wire_read_count(data []byte, at int) (count int, next int, err error) {
	if at+4 > len(data) {
		return 0, at, fmt.Errorf("wire: truncated count")
	}
	total := int(data[at]) | int(data[at+1])<<8 | int(data[at+2])<<16 | int(data[at+3])<<24
	return total, at + 4, nil
}

// Opens a new container frame, or delivers a finished value.
func wire_place(frames []wire_frame, read wire_read) (updated []wire_frame, root Value, done bool) {
	if read.Is_Open {
		opened := wire_frame{Kind: read.Kind, Remaining_Count: read.Count}
		return append(frames, opened), Value{}, false
	}
	return wire_deliver(frames, read.Value)
}

// Attaches a finished value to the open frame, popping and cascading any frames
// that complete until one stays open or the root is reached.
func wire_deliver(frames []wire_frame, value Value) (updated []wire_frame, root Value, done bool) {
	current := value
	for len(frames) > 0 {
		attached := wire_attach(frames[len(frames)-1], current)
		if attached.Remaining_Count > 0 {
			frames[len(frames)-1] = attached
			return frames, Value{}, false
		}
		frames = frames[:len(frames)-1]
		current = wire_frame_value(attached)
	}
	return frames, current, true
}

// Adds a value to a list or record frame and decrements its count.
func wire_attach(frame wire_frame, value Value) (updated wire_frame) {
	if frame.Kind == VALUE_KIND_LIST {
		frame.Items = append(frame.Items, value)
		frame.Remaining_Count--
		return frame
	}
	frame.Fields = append(frame.Fields, Field{Name: frame.Pending_Name, Value: value})
	frame.Has_Name = false
	frame.Remaining_Count--
	return frame
}

// Builds the finished container value from a completed frame.
func wire_frame_value(frame wire_frame) (value Value) {
	if frame.Kind == VALUE_KIND_LIST {
		return Value{Kind: VALUE_KIND_LIST, Items: frame.Items}
	}
	return Value{Kind: VALUE_KIND_RECORD, Fields: frame.Fields}
}

// The input kinds a reader classifies stdin into.
const INPUT_KIND_WIRE = 0

// A JSON stream from a foreign producer.
const INPUT_KIND_JSON = 1

// Raw, non-structured text.
const INPUT_KIND_RAW = 2

// Sniff classifies input: our magic means a sibling wrote it, a leading brace,
// bracket, or quote means foreign JSON, and anything else is raw text.
func Sniff(data []byte) (kind int) {
	if wire_has_magic(data) {
		return INPUT_KIND_WIRE
	}
	position := json_skip_space(data, 0)
	if position >= len(data) {
		return INPUT_KIND_RAW
	}
	if json_opens_structure(data[position]) {
		return INPUT_KIND_JSON
	}
	return INPUT_KIND_RAW
}

// Reports whether the data begins with the wire magic.
func wire_has_magic(data []byte) (has bool) {
	if len(data) < len(WIRE_MAGIC) {
		return false
	}
	return string(data[:len(WIRE_MAGIC)]) == WIRE_MAGIC
}

// Reports whether a byte opens a JSON value we recognize while sniffing.
func json_opens_structure(first byte) (opens bool) {
	switch first {
	case '{', '[', '"':
		return true
	}
	return false
}

// The end-of-input marker the lexer returns past the last token.
const JSON_TOKEN_END = 0

// An opening brace.
const JSON_TOKEN_OBJECT_OPEN = 1

// A closing brace.
const JSON_TOKEN_OBJECT_CLOSE = 2

// An opening bracket.
const JSON_TOKEN_ARRAY_OPEN = 3

// A closing bracket.
const JSON_TOKEN_ARRAY_CLOSE = 4

// A value separator.
const JSON_TOKEN_COMMA = 5

// A key/value separator.
const JSON_TOKEN_COLON = 6

// A string token; its text is already unescaped.
const JSON_TOKEN_STRING = 7

// A number token; its text is the exact source.
const JSON_TOKEN_NUMBER = 8

// The true literal.
const JSON_TOKEN_TRUE = 9

// The false literal.
const JSON_TOKEN_FALSE = 10

// The null literal.
const JSON_TOKEN_NULL = 11

// One container the JSON parser is filling from the token stream.
type json_frame struct {
	Is_Array    bool
	Items       []Value
	Fields      []Field
	Pending_Key string
	Has_Key     bool
}

// Json_Parse parses a JSON document into a value, preserving object key order
// and keeping numbers as their exact text. The walk is iterative over a frame
// stack, and numbers never become float64, so the package stays deterministic.
func Json_Parse(data []byte) (value Value, err error) {
	frames := []json_frame{}
	at := 0
	for at <= len(data) {
		kind, text, next, token_err := json_next_token(data, at)
		if token_err != nil {
			return Value{}, token_err
		}
		at = next
		if kind == JSON_TOKEN_END {
			return Value{}, fmt.Errorf("json: no complete value")
		}
		updated, root, done, apply_err := json_apply(frames, kind, text)
		if apply_err != nil {
			return Value{}, apply_err
		}
		frames = updated
		if done {
			return root, nil
		}
	}
	return Value{}, fmt.Errorf("json: incomplete input")
}

// Applies one token to the parser state, returning the root when it completes.
func json_apply(
	frames []json_frame, kind int, text string,
) (updated []json_frame, root Value, done bool, err error) {
	switch kind {
	case JSON_TOKEN_COLON, JSON_TOKEN_COMMA:
		return frames, Value{}, false, nil
	case JSON_TOKEN_OBJECT_OPEN:
		return append(frames, json_frame{Is_Array: false}), Value{}, false, nil
	case JSON_TOKEN_ARRAY_OPEN:
		return append(frames, json_frame{Is_Array: true}), Value{}, false, nil
	case JSON_TOKEN_OBJECT_CLOSE, JSON_TOKEN_ARRAY_CLOSE:
		closed, closed_root, closed_done := json_close(frames)
		return closed, closed_root, closed_done, nil
	}
	if json_is_key(frames, kind) {
		return json_set_key(frames, text), Value{}, false, nil
	}
	scalar, scalar_err := json_scalar(kind, text)
	if scalar_err != nil {
		return frames, Value{}, false, scalar_err
	}
	placed, placed_root, placed_done := json_place(frames, scalar)
	return placed, placed_root, placed_done, nil
}

// Reports whether the next string token names a field key rather than a value.
func json_is_key(frames []json_frame, kind int) (is_key bool) {
	if kind != JSON_TOKEN_STRING {
		return false
	}
	if len(frames) == 0 {
		return false
	}
	top := frames[len(frames)-1]
	if top.Is_Array {
		return false
	}
	return !top.Has_Key
}

// Records the pending field key on the open object frame.
func json_set_key(frames []json_frame, key string) (updated []json_frame) {
	top := frames[len(frames)-1]
	top.Pending_Key = key
	top.Has_Key = true
	frames[len(frames)-1] = top
	return frames
}

// Places a finished value into the open frame, or returns it as the root.
func json_place(frames []json_frame, value Value) (updated []json_frame, root Value, done bool) {
	if len(frames) == 0 {
		return frames, value, true
	}
	top := frames[len(frames)-1]
	if top.Is_Array {
		top.Items = append(top.Items, value)
		frames[len(frames)-1] = top
		return frames, Value{}, false
	}
	top.Fields = append(top.Fields, Field{Name: top.Pending_Key, Value: value})
	top.Has_Key = false
	frames[len(frames)-1] = top
	return frames, Value{}, false
}

// Closes the open container, builds its value, and places it in the parent.
func json_close(frames []json_frame) (updated []json_frame, root Value, done bool) {
	top := frames[len(frames)-1]
	frames = frames[:len(frames)-1]
	return json_place(frames, json_frame_value(top))
}

// Builds the finished container value from a completed frame.
func json_frame_value(frame json_frame) (value Value) {
	if frame.Is_Array {
		return List_Value(frame.Items)
	}
	return Record_Value(frame.Fields)
}

// Builds a scalar value from a value token.
func json_scalar(kind int, text string) (value Value, err error) {
	switch kind {
	case JSON_TOKEN_STRING:
		return String_Value(text), nil
	case JSON_TOKEN_NUMBER:
		return Number_Value(text), nil
	case JSON_TOKEN_TRUE:
		return Boolean_Value(true), nil
	case JSON_TOKEN_FALSE:
		return Boolean_Value(false), nil
	case JSON_TOKEN_NULL:
		return Null_Value(), nil
	}
	return Value{}, fmt.Errorf("json: unexpected token")
}

// Reads the next token, skipping leading whitespace.
func json_next_token(data []byte, at int) (kind int, text string, next int, err error) {
	position := json_skip_space(data, at)
	if position >= len(data) {
		return JSON_TOKEN_END, "", position, nil
	}
	switch data[position] {
	case '{':
		return JSON_TOKEN_OBJECT_OPEN, "", position + 1, nil
	case '}':
		return JSON_TOKEN_OBJECT_CLOSE, "", position + 1, nil
	case '[':
		return JSON_TOKEN_ARRAY_OPEN, "", position + 1, nil
	case ']':
		return JSON_TOKEN_ARRAY_CLOSE, "", position + 1, nil
	case ',':
		return JSON_TOKEN_COMMA, "", position + 1, nil
	case ':':
		return JSON_TOKEN_COLON, "", position + 1, nil
	case '"':
		return json_read_string_token(data, position)
	case 't':
		return json_read_literal(&json_read_literal_input{
			Data: data, At: position, Word: "true", Kind: JSON_TOKEN_TRUE,
		})
	case 'f':
		return json_read_literal(&json_read_literal_input{
			Data: data, At: position, Word: "false", Kind: JSON_TOKEN_FALSE,
		})
	case 'n':
		return json_read_literal(&json_read_literal_input{
			Data: data, At: position, Word: "null", Kind: JSON_TOKEN_NULL,
		})
	}
	return json_read_number_token(data, position)
}

// Returns the offset of the first non-space byte at or after the position.
func json_skip_space(data []byte, at int) (next int) {
	position := at
	for position < len(data) {
		if !json_is_space(data[position]) {
			return position
		}
		position++
	}
	return position
}

// Reports whether a byte is JSON whitespace.
func json_is_space(character byte) (space bool) {
	switch character {
	case ' ', '\t', '\n', '\r':
		return true
	}
	return false
}

// The inputs to json_read_literal, bundled because its parameter types repeat.
type json_read_literal_input struct {
	Data []byte
	At   int
	Word string
	Kind int
}

// Reads a fixed keyword literal such as true, false, or null.
func json_read_literal(
	input *json_read_literal_input,
) (token int, text string, next int, err error) {
	end := input.At + len(input.Word)
	if end > len(input.Data) {
		return JSON_TOKEN_END, "", input.At, fmt.Errorf("json: truncated literal")
	}
	if string(input.Data[input.At:end]) != input.Word {
		return JSON_TOKEN_END, "", input.At, fmt.Errorf("json: invalid literal")
	}
	return input.Kind, "", end, nil
}

// Reads a string token, unescaping it through the standard library.
func json_read_string_token(data []byte, at int) (kind int, text string, next int, err error) {
	end, end_err := json_string_end(data, at)
	if end_err != nil {
		return JSON_TOKEN_END, "", at, end_err
	}
	decoded := ""
	decode_err := json.Unmarshal(data[at:end], &decoded)
	if decode_err != nil {
		return JSON_TOKEN_END, "", at, decode_err
	}
	return JSON_TOKEN_STRING, decoded, end, nil
}

// Returns the offset just past a string's closing quote, skipping escapes.
func json_string_end(data []byte, at int) (end int, err error) {
	scan_index := at + 1
	for scan_index < len(data) {
		if data[scan_index] == '\\' {
			scan_index += 2
			continue
		}
		if data[scan_index] == '"' {
			return scan_index + 1, nil
		}
		scan_index++
	}
	return at, fmt.Errorf("json: unterminated string")
}

// Reads a number token, keeping its exact source text.
func json_read_number_token(data []byte, at int) (kind int, text string, next int, err error) {
	end := at
	for end < len(data) {
		if !json_is_number_byte(data[end]) {
			break
		}
		end++
	}
	if end == at {
		return JSON_TOKEN_END, "", at, fmt.Errorf("json: unexpected byte")
	}
	return JSON_TOKEN_NUMBER, string(data[at:end]), end, nil
}

// Reports whether a byte can appear in a JSON number token.
func json_is_number_byte(character byte) (numeric bool) {
	switch character {
	case '-', '+', '.', 'e', 'E', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return true
	}
	return false
}

// One item on the JSON emitter's stack: a value to emit, or a literal run of
// output text such as a brace, comma, or quoted key.
type json_emit_work struct {
	Is_Value bool
	Value    Value
	Text     string
}

// Json_Emit serializes a value to compact JSON, preserving record key order. The
// walk is iterative over an explicit stack, since recursion is banned.
func Json_Emit(value Value) (text string) {
	output := []byte{}
	stack := []json_emit_work{{Is_Value: true, Value: value}}
	for len(stack) > 0 {
		top := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if !top.Is_Value {
			output = append(output, top.Text...)
			continue
		}
		pieces := json_emit_pieces(top.Value)
		for piece_index := len(pieces) - 1; piece_index >= 0; piece_index-- {
			stack = append(stack, pieces[piece_index])
		}
	}
	return string(output)
}

// Breaks one value into the emit pieces that render it.
func json_emit_pieces(value Value) (pieces []json_emit_work) {
	switch value.Kind {
	case VALUE_KIND_BOOLEAN:
		return json_literal_piece(json_boolean_text(value.Boolean))
	case VALUE_KIND_NUMBER:
		return json_literal_piece(value.Number)
	case VALUE_KIND_STRING:
		return json_literal_piece(json_quote(value.Text))
	case VALUE_KIND_LIST:
		return json_list_pieces(value.Items)
	case VALUE_KIND_RECORD:
		return json_record_pieces(value.Fields)
	}
	return json_literal_piece("null")
}

// Wraps one literal run of text as a single emit piece.
func json_literal_piece(text string) (pieces []json_emit_work) {
	return []json_emit_work{{Is_Value: false, Text: text}}
}

// Renders true or false.
func json_boolean_text(flag bool) (text string) {
	if flag {
		return "true"
	}
	return "false"
}

// Breaks a list into bracketed, comma-separated element pieces.
func json_list_pieces(items []Value) (pieces []json_emit_work) {
	pieces = append(pieces, json_emit_work{Is_Value: false, Text: "["})
	for item_index := 0; item_index < len(items); item_index++ {
		if item_index > 0 {
			pieces = append(pieces, json_emit_work{Is_Value: false, Text: ","})
		}
		pieces = append(pieces, json_emit_work{Is_Value: true, Value: items[item_index]})
	}
	pieces = append(pieces, json_emit_work{Is_Value: false, Text: "]"})
	return pieces
}

// Breaks a record into braced, comma-separated "key":value pieces.
func json_record_pieces(fields []Field) (pieces []json_emit_work) {
	pieces = append(pieces, json_emit_work{Is_Value: false, Text: "{"})
	for field_index := 0; field_index < len(fields); field_index++ {
		if field_index > 0 {
			pieces = append(pieces, json_emit_work{Is_Value: false, Text: ","})
		}
		key := json_quote(fields[field_index].Name) + ":"
		pieces = append(pieces, json_emit_work{Is_Value: false, Text: key})
		value := fields[field_index].Value
		pieces = append(pieces, json_emit_work{Is_Value: true, Value: value})
	}
	pieces = append(pieces, json_emit_work{Is_Value: false, Text: "}"})
	return pieces
}

// Quotes and escapes a string through the standard library.
func json_quote(text string) (quoted string) {
	encoded, marshal_err := json.Marshal(text)
	if marshal_err != nil {
		return "\"\""
	}
	return string(encoded)
}

// Runs one verb against injected IO, returning the bytes to write. The source
// verbs read a file or stdin; the rest read stdin, transform, and re-encode.
func run_verb(
	input *Main_Input, kind Verb_Kind, arguments []string, mode Output_Mode,
) (output []byte, err error) {
	switch kind {
	case VERB_KIND_LOAD:
		return run_load(input, arguments, mode)
	case VERB_KIND_FROM:
		return run_from(input, arguments, mode)
	case VERB_KIND_TO:
		return run_to(input, arguments)
	}
	value, decode_err := decode_input(input.Read_Stdin())
	if decode_err != nil {
		return nil, decode_err
	}
	transformed, transform_err := apply_transform(kind, arguments, value)
	if transform_err != nil {
		return nil, transform_err
	}
	return encode_output(transformed, mode), nil
}

// Runs the load verb: read a file, decode by sniffing, re-encode for the mode.
func run_load(input *Main_Input, arguments []string, mode Output_Mode) (output []byte, err error) {
	if len(arguments) == 0 {
		return nil, fmt.Errorf("load needs a file")
	}
	name := arguments[0]
	data, read_err := input.Read_File(name)
	if read_err != nil {
		return nil, fmt.Errorf("cannot read %s", name)
	}
	value, decode_err := load_decode(name, data)
	if decode_err != nil {
		return nil, decode_err
	}
	return encode_output(value, mode), nil
}

// Decodes a loaded file: a .csv or .json extension picks the parser, otherwise
// the content is sniffed as wire, JSON, or raw text.
func load_decode(name string, data []byte) (value Value, err error) {
	if strings.HasSuffix(name, ".csv") {
		return Csv_Parse(data)
	}
	if strings.HasSuffix(name, ".json") {
		return Json_Parse(data)
	}
	return decode_input(data)
}

// Runs the from verb: parse stdin as an explicit foreign format.
func run_from(input *Main_Input, arguments []string, mode Output_Mode) (output []byte, err error) {
	format := "json"
	if len(arguments) > 0 {
		format = arguments[0]
	}
	value, parse_err := parse_format(format, input.Read_Stdin())
	if parse_err != nil {
		return nil, parse_err
	}
	return encode_output(value, mode), nil
}

// Runs the to verb: decode stdin and serialize out of the garden.
func run_to(input *Main_Input, arguments []string) (output []byte, err error) {
	format := "json"
	if len(arguments) > 0 {
		format = arguments[0]
	}
	value, decode_err := decode_input(input.Read_Stdin())
	if decode_err != nil {
		return nil, decode_err
	}
	text, emit_err := emit_format(format, value)
	if emit_err != nil {
		return nil, emit_err
	}
	return []byte(text), nil
}

// Parses a named foreign format into a value.
func parse_format(format string, data []byte) (value Value, err error) {
	switch format {
	case "json":
		return Json_Parse(data)
	case "csv":
		return Csv_Parse(data)
	}
	return Value{}, fmt.Errorf("unknown format %s", format)
}

// Serializes a value to a named foreign format, ending in a newline.
func emit_format(format string, value Value) (text string, err error) {
	switch format {
	case "json":
		return Json_Emit(value) + "\n", nil
	case "csv":
		return Csv_Emit(value)
	}
	return "", fmt.Errorf("unknown format %s", format)
}

// Applies a transforming verb to a decoded value.
func apply_transform(kind Verb_Kind, arguments []string, value Value) (result Value, err error) {
	switch kind {
	case VERB_KIND_GET:
		return verb_get(arguments, value)
	case VERB_KIND_PICK:
		return verb_pick(arguments, value)
	case VERB_KIND_FILTER:
		return verb_filter(arguments, value)
	case VERB_KIND_FIRST:
		return verb_first(arguments, value)
	}
	return apply_transform_more(kind, arguments, value)
}

// Applies the second half of the transforming verbs.
func apply_transform_more(
	kind Verb_Kind, arguments []string, value Value,
) (result Value, err error) {
	switch kind {
	case VERB_KIND_SORT_BY:
		return verb_sort_by(arguments, value)
	case VERB_KIND_GROUP_BY:
		return verb_group_by(arguments, value)
	case VERB_KIND_DISTINCT:
		return verb_distinct(arguments, value)
	case VERB_KIND_REVERSE:
		return verb_reverse(arguments, value)
	case VERB_KIND_FINAL:
		return verb_final(arguments, value)
	case VERB_KIND_COLUMNS:
		return verb_columns(arguments, value)
	case VERB_KIND_REJECT:
		return verb_reject(arguments, value)
	case VERB_KIND_RELABEL:
		return verb_relabel(arguments, value)
	case VERB_KIND_COUNT:
		return verb_count(arguments, value)
	case VERB_KIND_WRAP:
		return verb_wrap(arguments, value)
	case VERB_KIND_FLATTEN:
		return verb_flatten(arguments, value)
	}
	return value, nil
}

// Decodes stdin by sniffing: a sibling's wire, foreign JSON, or raw text.
func decode_input(data []byte) (value Value, err error) {
	switch Sniff(data) {
	case INPUT_KIND_WIRE:
		return Wire_Decode(data)
	case INPUT_KIND_JSON:
		return Json_Parse(data)
	}
	return String_Value(string(data)), nil
}

// Encodes a result for the chosen output mode.
func encode_output(value Value, mode Output_Mode) (output []byte) {
	switch mode {
	case OUTPUT_MODE_JSON:
		return []byte(Json_Emit(value))
	case OUTPUT_MODE_TABLE:
		return []byte(render(value) + "\n")
	}
	return Wire_Encode(value)
}

// Runs the get verb: navigate a dotted cell path.
func verb_get(arguments []string, value Value) (result Value, err error) {
	if len(arguments) == 0 {
		return Value{}, fmt.Errorf("get needs a path")
	}
	return get_path(value, arguments[0]), nil
}

// Runs the pick verb: keep only the named fields of each record.
func verb_pick(arguments []string, value Value) (result Value, err error) {
	if len(arguments) == 0 {
		return Value{}, fmt.Errorf("pick needs field names")
	}
	return pick_value(value, arguments), nil
}

// Projects the named fields, mapping across a list of records.
func pick_value(value Value, names []string) (result Value) {
	if value.Kind == VALUE_KIND_RECORD {
		return Record_Value(pick_fields(value.Fields, names))
	}
	if value.Kind == VALUE_KIND_LIST {
		picked := []Value{}
		for item_index := 0; item_index < len(value.Items); item_index++ {
			picked = append(picked, pick_one(value.Items[item_index], names))
		}
		return List_Value(picked)
	}
	return value
}

// Projects the named fields of a single value, leaving non-records unchanged.
func pick_one(value Value, names []string) (result Value) {
	if value.Kind == VALUE_KIND_RECORD {
		return Record_Value(pick_fields(value.Fields, names))
	}
	return value
}

// Builds a sub-record of the named fields, Null-filling absent ones.
func pick_fields(fields []Field, names []string) (picked []Field) {
	for name_index := 0; name_index < len(names); name_index++ {
		name := names[name_index]
		picked = append(picked, Field{Name: name, Value: field_value(fields, name)})
	}
	return picked
}

// The inputs to filter_keeps, bundled because its parameter types repeat.
type filter_test struct {
	Item     Value
	Field    string
	Operator string
	Target   Value
}

// Runs the filter verb: keep list rows whose cell satisfies the word operator.
func verb_filter(arguments []string, value Value) (result Value, err error) {
	if len(arguments) < 3 {
		return Value{}, fmt.Errorf("filter needs: field op value")
	}
	operator := arguments[1]
	if !operator_is_known(operator) {
		return Value{}, fmt.Errorf("unknown operator: %s", operator)
	}
	if value.Kind != VALUE_KIND_LIST {
		return value, nil
	}
	field := arguments[0]
	target := parse_argument_value(arguments[2])
	kept := []Value{}
	for item_index := 0; item_index < len(value.Items); item_index++ {
		item := value.Items[item_index]
		test := filter_test{Item: item, Field: field, Operator: operator, Target: target}
		if filter_keeps(test) {
			kept = append(kept, item)
		}
	}
	return List_Value(kept), nil
}

// Reports whether a row's field satisfies the operator against the target.
func filter_keeps(test filter_test) (keep bool) {
	cell := get_path(test.Item, test.Field)
	order, comparable := compare_values(value_pair{Left: cell, Right: test.Target})
	if !comparable {
		return false
	}
	return operator_satisfied(test.Operator, order)
}

// Reports whether an operator is one of the supported word operators.
func operator_is_known(operator string) (known bool) {
	switch operator {
	case "eq", "ne", "lt", "le", "gt", "ge":
		return true
	}
	return false
}

// Maps a word operator and an ordering to a keep decision.
func operator_satisfied(operator string, order int) (satisfied bool) {
	switch operator {
	case "eq":
		return order == 0
	case "ne":
		return order != 0
	case "lt":
		return order < 0
	case "le":
		return order <= 0
	case "gt":
		return order > 0
	}
	return order >= 0
}

// Runs the first verb: the first element, or the first n as a list.
func verb_first(arguments []string, value Value) (result Value, err error) {
	if value.Kind != VALUE_KIND_LIST {
		return value, nil
	}
	if len(arguments) == 0 {
		return first_element(value.Items), nil
	}
	count, count_err := strconv.Atoi(arguments[0])
	if count_err != nil {
		return Value{}, fmt.Errorf("first needs a count")
	}
	return List_Value(first_n(value.Items, count)), nil
}

// Returns the first element, or null when the list is empty.
func first_element(items []Value) (result Value) {
	if len(items) == 0 {
		return Null_Value()
	}
	return items[0]
}

// Returns the first count elements, bounded by the list length.
func first_n(items []Value, count int) (taken []Value) {
	limit := count
	if limit > len(items) {
		limit = len(items)
	}
	if limit < 0 {
		limit = 0
	}
	return items[:limit]
}

// Navigates a dotted cell path (user.address.city). A record segment selects a
// field; a numeric segment indexes a list; a name over a list extracts that
// column from every element.
func get_path(value Value, route string) (result Value) {
	current := value
	segments := strings.Split(route, ".")
	for segment_index := 0; segment_index < len(segments); segment_index++ {
		current = path_step(current, segments[segment_index])
	}
	return current
}

// Takes one path step against a value.
func path_step(value Value, segment string) (result Value) {
	if value.Kind == VALUE_KIND_RECORD {
		return field_value(value.Fields, segment)
	}
	if value.Kind == VALUE_KIND_LIST {
		return list_step(value.Items, segment)
	}
	return Null_Value()
}

// Takes one path step against a list: a numeric segment indexes, a name extracts
// the column from every element.
func list_step(items []Value, segment string) (result Value) {
	index, is_index := parse_index(segment)
	if is_index {
		if index < len(items) {
			return items[index]
		}
		return Null_Value()
	}
	extracted := []Value{}
	for item_index := 0; item_index < len(items); item_index++ {
		extracted = append(extracted, path_field(items[item_index], segment))
	}
	return List_Value(extracted)
}

// Extracts one field from a value, or null when it is not a record.
func path_field(value Value, name string) (result Value) {
	if value.Kind == VALUE_KIND_RECORD {
		return field_value(value.Fields, name)
	}
	return Null_Value()
}

// Reads a field from a record's ordered fields, or null when absent.
func field_value(fields []Field, name string) (result Value) {
	for field_index := 0; field_index < len(fields); field_index++ {
		if fields[field_index].Name == name {
			return fields[field_index].Value
		}
	}
	return Null_Value()
}

// Parses a non-negative decimal index, reporting whether it is one.
func parse_index(segment string) (index int, is_index bool) {
	value, parse_err := strconv.Atoi(segment)
	if parse_err != nil {
		return 0, false
	}
	if value < 0 {
		return 0, false
	}
	return value, true
}

// Parses a bare command-line token into a scalar: a boolean, a number when it
// parses, otherwise a string. This is how a filter literal becomes comparable.
func parse_argument_value(text string) (value Value) {
	if text == "true" {
		return Boolean_Value(true)
	}
	if text == "false" {
		return Boolean_Value(false)
	}
	if is_number_text(text) {
		return Number_Value(text)
	}
	return String_Value(text)
}

// Reports whether text parses as an exact rational number.
func is_number_text(text string) (numeric bool) {
	_, ok := new(big.Rat).SetString(text)
	return ok
}

// The two values a comparison orders, bundled because the types repeat.
type value_pair struct {
	Left  Value
	Right Value
}

// Orders two same-kind values, reporting whether they are comparable. Numbers
// compare as exact rationals, so the result is deterministic without float64.
func compare_values(pair value_pair) (order int, comparable bool) {
	left := pair.Left
	right := pair.Right
	if left.Kind != right.Kind {
		return 0, false
	}
	if left.Kind == VALUE_KIND_NUMBER {
		return compare_numbers(number_pair{Left: left.Number, Right: right.Number})
	}
	if left.Kind == VALUE_KIND_STRING {
		return strings.Compare(left.Text, right.Text), true
	}
	if left.Kind == VALUE_KIND_BOOLEAN {
		if left.Boolean == right.Boolean {
			return 0, true
		}
		if left.Boolean {
			return 1, true
		}
		return -1, true
	}
	return 0, false
}

// The two number texts a comparison orders, bundled because the types repeat.
type number_pair struct {
	Left  string
	Right string
}

// Orders two numbers as exact rationals.
func compare_numbers(pair number_pair) (order int, comparable bool) {
	left_rat, left_ok := new(big.Rat).SetString(pair.Left)
	if !left_ok {
		return 0, false
	}
	right_rat, right_ok := new(big.Rat).SetString(pair.Right)
	if !right_ok {
		return 0, false
	}
	return left_rat.Cmp(right_rat), true
}

// Renders a value for a human at a terminal: a list of records becomes an
// aligned table, a lone record a key/value table, a scalar list one per line.
func render(value Value) (text string) {
	if value.Kind == VALUE_KIND_RECORD {
		return render_record(value.Fields)
	}
	if value.Kind != VALUE_KIND_LIST {
		return cell_text(value)
	}
	if all_records(value.Items) {
		return render_table(value.Items)
	}
	return render_list(value.Items)
}

// Reports whether a non-empty list is entirely records (table-shaped).
func all_records(items []Value) (all bool) {
	if len(items) == 0 {
		return false
	}
	for item_index := 0; item_index < len(items); item_index++ {
		if items[item_index].Kind != VALUE_KIND_RECORD {
			return false
		}
	}
	return true
}

// Renders a table: a header of the union of keys, then one row per record.
func render_table(items []Value) (text string) {
	columns := table_columns(items)
	return lay_output(columns, table_rows(items, columns))
}

// The union of record keys across the rows, in first-seen order.
func table_columns(items []Value) (columns []string) {
	columns = []string{}
	for item_index := 0; item_index < len(items); item_index++ {
		fields := items[item_index].Fields
		for field_index := 0; field_index < len(fields); field_index++ {
			columns = add_unique(columns, fields[field_index].Name)
		}
	}
	return columns
}

// Appends a name only when it is not already present.
func add_unique(columns []string, name string) (updated []string) {
	for column_index := 0; column_index < len(columns); column_index++ {
		if columns[column_index] == name {
			return columns
		}
	}
	return append(columns, name)
}

// Builds one string row per record, one cell per column.
func table_rows(items []Value, columns []string) (rows [][]string) {
	rows = [][]string{}
	for item_index := 0; item_index < len(items); item_index++ {
		rows = append(rows, row_cells(items[item_index], columns))
	}
	return rows
}

// The cells of one row, one per column (a missing field renders empty).
func row_cells(value Value, columns []string) (cells []string) {
	cells = []string{}
	for column_index := 0; column_index < len(columns); column_index++ {
		cells = append(cells, cell_text(field_value(value.Fields, columns[column_index])))
	}
	return cells
}

// Renders a lone record as a two-column key/value table.
func render_record(fields []Field) (text string) {
	rows := [][]string{}
	for field_index := 0; field_index < len(fields); field_index++ {
		field := fields[field_index]
		rows = append(rows, []string{field.Name, cell_text(field.Value)})
	}
	return lay_output([]string{"key", "value"}, rows)
}

// Renders a scalar list as one value per line.
func render_list(items []Value) (text string) {
	lines := []string{}
	for item_index := 0; item_index < len(items); item_index++ {
		lines = append(lines, cell_text(items[item_index]))
	}
	return strings.Join(lines, "\n")
}

// Lays a header and rows into an aligned, pipe-separated table.
func lay_output(headers []string, rows [][]string) (text string) {
	widths := column_widths(headers, rows)
	lines := []string{format_row(headers, widths), separator_row(widths)}
	for row_index := 0; row_index < len(rows); row_index++ {
		lines = append(lines, format_row(rows[row_index], widths))
	}
	return strings.Join(lines, "\n")
}

// The display width of each column: the widest of its header and cells.
func column_widths(headers []string, rows [][]string) (widths []int) {
	widths = []int{}
	for column_index := 0; column_index < len(headers); column_index++ {
		widths = append(widths, column_width(column_index, headers[column_index], rows))
	}
	return widths
}

// The widest cell of one column, across the header and every row.
func column_width(column_index int, header string, rows [][]string) (width int) {
	width = len(header)
	for row_index := 0; row_index < len(rows); row_index++ {
		row := rows[row_index]
		if column_index >= len(row) {
			continue
		}
		if len(row[column_index]) > width {
			width = len(row[column_index])
		}
	}
	return width
}

// Joins cells with a pipe separator, padding every column but the last.
func format_row(cells []string, widths []int) (line string) {
	pieces := []string{}
	for cell_index := 0; cell_index < len(cells); cell_index++ {
		is_last := cell_index == len(cells)-1
		width := 0
		if cell_index < len(widths) {
			width = widths[cell_index]
		}
		pieces = append(pieces, pad_cell(cells[cell_index], is_last, width))
	}
	return strings.Join(pieces, " | ")
}

// Right-pads a cell to its column width, unless it is the last cell.
func pad_cell(text string, is_last bool, width int) (padded string) {
	if is_last {
		return text
	}
	fill := width - len(text)
	if fill <= 0 {
		return text
	}
	return text + strings.Repeat(" ", fill)
}

// The dashed rule under the header, matching the column widths.
func separator_row(widths []int) (line string) {
	pieces := []string{}
	for width_index := 0; width_index < len(widths); width_index++ {
		pieces = append(pieces, strings.Repeat("-", widths[width_index]))
	}
	return strings.Join(pieces, "-+-")
}

// One scalar's cell text; containers show a short summary, null shows empty.
func cell_text(value Value) (text string) {
	switch value.Kind {
	case VALUE_KIND_BOOLEAN:
		return json_boolean_text(value.Boolean)
	case VALUE_KIND_NUMBER:
		return value.Number
	case VALUE_KIND_STRING:
		return value.Text
	case VALUE_KIND_LIST:
		return fmt.Sprintf("[list %d items]", len(value.Items))
	case VALUE_KIND_RECORD:
		return fmt.Sprintf("[record %d fields]", len(value.Fields))
	}
	return ""
}

// Runs the sort-by verb: stable-sort a list of records by a field. An iterative
// insertion sort keeps it stable and deterministic; sort.Interface is unusable
// here because Len holds a banned word and Less/Swap have repeating parameters.
func verb_sort_by(arguments []string, value Value) (result Value, err error) {
	if len(arguments) == 0 {
		return Value{}, fmt.Errorf("sort-by needs a field")
	}
	if value.Kind != VALUE_KIND_LIST {
		return value, nil
	}
	return List_Value(sort_records(value.Items, arguments[0])), nil
}

// Stable-sorts records by a field via insertion into a growing sorted list.
func sort_records(items []Value, field string) (sorted []Value) {
	sorted = []Value{}
	for item_index := 0; item_index < len(items); item_index++ {
		sorted = insert_sorted(sorted, items[item_index], field)
	}
	return sorted
}

// Inserts an item into the sorted list at its ordered position.
func insert_sorted(sorted []Value, item Value, field string) (updated []Value) {
	position := insert_position(sorted, item, field)
	updated = append(updated, sorted[:position]...)
	updated = append(updated, item)
	updated = append(updated, sorted[position:]...)
	return updated
}

// Finds the first position whose field is strictly greater than the item's, so
// equal keys keep their original order.
func insert_position(sorted []Value, item Value, field string) (position int) {
	item_key := get_path(item, field)
	for sorted_index := 0; sorted_index < len(sorted); sorted_index++ {
		existing_key := get_path(sorted[sorted_index], field)
		order, comparable := compare_values(value_pair{Left: item_key, Right: existing_key})
		if !comparable {
			continue
		}
		if order < 0 {
			return sorted_index
		}
	}
	return len(sorted)
}

// Runs the group-by verb: bucket records by a field value, first-seen order.
func verb_group_by(arguments []string, value Value) (result Value, err error) {
	if len(arguments) == 0 {
		return Value{}, fmt.Errorf("group-by needs a field")
	}
	if value.Kind != VALUE_KIND_LIST {
		return value, nil
	}
	field := arguments[0]
	groups := []Field{}
	for item_index := 0; item_index < len(value.Items); item_index++ {
		item := value.Items[item_index]
		groups = group_append(groups, cell_text(get_path(item, field)), item)
	}
	return Record_Value(groups), nil
}

// Appends an item to its group, creating the group in first-seen order.
func group_append(groups []Field, key string, item Value) (updated []Field) {
	for group_index := 0; group_index < len(groups); group_index++ {
		if groups[group_index].Name != key {
			continue
		}
		bucket := append(groups[group_index].Value.Items, item)
		groups[group_index].Value = List_Value(bucket)
		return groups
	}
	return append(groups, Field{Name: key, Value: List_Value([]Value{item})})
}

// Runs the distinct verb: drop duplicate list elements by value equality.
func verb_distinct(arguments []string, value Value) (result Value, err error) {
	if value.Kind != VALUE_KIND_LIST {
		return value, nil
	}
	seen := map[string]bool{}
	kept := []Value{}
	for item_index := 0; item_index < len(value.Items); item_index++ {
		item := value.Items[item_index]
		key := string(Wire_Encode(item))
		if seen[key] {
			continue
		}
		seen[key] = true
		kept = append(kept, item)
	}
	return List_Value(kept), nil
}

// Runs the reverse verb: reverse a list.
func verb_reverse(arguments []string, value Value) (result Value, err error) {
	if value.Kind != VALUE_KIND_LIST {
		return value, nil
	}
	reversed := []Value{}
	for item_index := len(value.Items) - 1; item_index >= 0; item_index-- {
		reversed = append(reversed, value.Items[item_index])
	}
	return List_Value(reversed), nil
}

// Runs the final verb: the last element, or the last n as a list.
func verb_final(arguments []string, value Value) (result Value, err error) {
	if value.Kind != VALUE_KIND_LIST {
		return value, nil
	}
	if len(arguments) == 0 {
		return last_element(value.Items), nil
	}
	count, count_err := strconv.Atoi(arguments[0])
	if count_err != nil {
		return Value{}, fmt.Errorf("final needs a count")
	}
	return List_Value(last_n(value.Items, count)), nil
}

// Returns the last element, or null when the list is empty.
func last_element(items []Value) (result Value) {
	if len(items) == 0 {
		return Null_Value()
	}
	return items[len(items)-1]
}

// Returns the last count elements, bounded by the list length.
func last_n(items []Value, count int) (taken []Value) {
	start := len(items) - count
	if start < 0 {
		start = 0
	}
	return items[start:]
}

// Runs the columns verb: the union of record keys as a list of strings.
func verb_columns(arguments []string, value Value) (result Value, err error) {
	if value.Kind != VALUE_KIND_LIST {
		return value, nil
	}
	names := table_columns(value.Items)
	items := []Value{}
	for name_index := 0; name_index < len(names); name_index++ {
		items = append(items, String_Value(names[name_index]))
	}
	return List_Value(items), nil
}

// Runs the reject verb: drop the named fields of each record.
func verb_reject(arguments []string, value Value) (result Value, err error) {
	if len(arguments) == 0 {
		return Value{}, fmt.Errorf("reject needs field names")
	}
	return reject_value(value, arguments), nil
}

// Drops the named fields, mapping across a list of records.
func reject_value(value Value, names []string) (result Value) {
	if value.Kind == VALUE_KIND_RECORD {
		return Record_Value(reject_fields(value.Fields, names))
	}
	if value.Kind == VALUE_KIND_LIST {
		kept := []Value{}
		for item_index := 0; item_index < len(value.Items); item_index++ {
			kept = append(kept, reject_one(value.Items[item_index], names))
		}
		return List_Value(kept)
	}
	return value
}

// Drops the named fields of a single value, leaving non-records unchanged.
func reject_one(value Value, names []string) (result Value) {
	if value.Kind == VALUE_KIND_RECORD {
		return Record_Value(reject_fields(value.Fields, names))
	}
	return value
}

// Keeps only the fields whose names are not in the reject set.
func reject_fields(fields []Field, names []string) (kept []Field) {
	for field_index := 0; field_index < len(fields); field_index++ {
		if name_in(names, fields[field_index].Name) {
			continue
		}
		kept = append(kept, fields[field_index])
	}
	return kept
}

// Reports whether a name is present in a list of names.
func name_in(names []string, name string) (present bool) {
	for name_index := 0; name_index < len(names); name_index++ {
		if names[name_index] == name {
			return true
		}
	}
	return false
}

// The two field names a relabel carries, bundled because the types repeat.
type relabel_input struct {
	Old string
	New string
}

// Runs the relabel verb: rename a field across records.
func verb_relabel(arguments []string, value Value) (result Value, err error) {
	if len(arguments) < 2 {
		return Value{}, fmt.Errorf("relabel needs: old new")
	}
	return relabel_value(value, relabel_input{Old: arguments[0], New: arguments[1]}), nil
}

// Renames a field, mapping across a list of records.
func relabel_value(value Value, names relabel_input) (result Value) {
	if value.Kind == VALUE_KIND_RECORD {
		return Record_Value(relabel_fields(value.Fields, names))
	}
	if value.Kind == VALUE_KIND_LIST {
		renamed := []Value{}
		for item_index := 0; item_index < len(value.Items); item_index++ {
			renamed = append(renamed, relabel_one(value.Items[item_index], names))
		}
		return List_Value(renamed)
	}
	return value
}

// Renames a field of a single value, leaving non-records unchanged.
func relabel_one(value Value, names relabel_input) (result Value) {
	if value.Kind == VALUE_KIND_RECORD {
		return Record_Value(relabel_fields(value.Fields, names))
	}
	return value
}

// Renames the matching field of a record's fields.
func relabel_fields(fields []Field, names relabel_input) (renamed []Field) {
	for field_index := 0; field_index < len(fields); field_index++ {
		field := fields[field_index]
		if field.Name == names.Old {
			field.Name = names.New
		}
		renamed = append(renamed, field)
	}
	return renamed
}

// Runs the count verb: the number of list items or record fields, as a number.
func verb_count(arguments []string, value Value) (result Value, err error) {
	if value.Kind == VALUE_KIND_LIST {
		return Number_Value(strconv.Itoa(len(value.Items))), nil
	}
	if value.Kind == VALUE_KIND_RECORD {
		return Number_Value(strconv.Itoa(len(value.Fields))), nil
	}
	return Number_Value("1"), nil
}

// Runs the wrap verb: wrap the value in a single-field record.
func verb_wrap(arguments []string, value Value) (result Value, err error) {
	if len(arguments) == 0 {
		return Value{}, fmt.Errorf("wrap needs a field name")
	}
	return Record_Value([]Field{{Name: arguments[0], Value: value}}), nil
}

// Runs the flatten verb: flatten one level of nested lists.
func verb_flatten(arguments []string, value Value) (result Value, err error) {
	if value.Kind != VALUE_KIND_LIST {
		return value, nil
	}
	flat := []Value{}
	for item_index := 0; item_index < len(value.Items); item_index++ {
		flat = flatten_append(flat, value.Items[item_index])
	}
	return List_Value(flat), nil
}

// Appends an item, splicing in a nested list's elements one level deep.
func flatten_append(flat []Value, item Value) (updated []Value) {
	if item.Kind != VALUE_KIND_LIST {
		return append(flat, item)
	}
	for inner_index := 0; inner_index < len(item.Items); inner_index++ {
		flat = append(flat, item.Items[inner_index])
	}
	return flat
}

// Csv_Parse parses CSV into a table: the first row names the columns, each later
// row is a record whose cells are inferred (numbers, booleans, else strings) so
// a numeric filter works downstream.
func Csv_Parse(data []byte) (value Value, err error) {
	rows := csv_rows(data)
	if len(rows) == 0 {
		return List_Value([]Value{}), nil
	}
	header := rows[0]
	records := []Value{}
	for row_index := 1; row_index < len(rows); row_index++ {
		record := csv_record(&csv_record_input{Header: header, Row: rows[row_index]})
		records = append(records, record)
	}
	return List_Value(records), nil
}

// The header and one row a CSV record is built from, bundled because the types
// repeat.
type csv_record_input struct {
	Header []string
	Row    []string
}

// Builds one record from a header and a row, inferring each cell's type.
func csv_record(input *csv_record_input) (value Value) {
	fields := []Field{}
	for column_index := 0; column_index < len(input.Header); column_index++ {
		cell := ""
		if column_index < len(input.Row) {
			cell = input.Row[column_index]
		}
		name := input.Header[column_index]
		fields = append(fields, Field{Name: name, Value: parse_argument_value(cell)})
	}
	return Record_Value(fields)
}

// Splits CSV bytes into rows of string fields, honoring quoted fields with
// embedded commas, newlines, and doubled-quote escapes. Iterative state machine.
func csv_rows(data []byte) (rows [][]string) {
	rows = [][]string{}
	fields := []string{}
	field := []byte{}
	in_quote := false
	position := 0
	for position < len(data) {
		character := data[position]
		if in_quote {
			if character != '"' {
				field = append(field, character)
				position++
				continue
			}
			if csv_is_escaped_quote(data, position) {
				field = append(field, '"')
				position += 2
				continue
			}
			in_quote = false
			position++
			continue
		}
		if character == '"' {
			in_quote = true
			position++
			continue
		}
		if character == ',' {
			fields = append(fields, string(field))
			field = []byte{}
			position++
			continue
		}
		if character == '\n' {
			fields = append(fields, string(field))
			rows = append(rows, fields)
			fields = []string{}
			field = []byte{}
			position++
			continue
		}
		if character == '\r' {
			position++
			continue
		}
		field = append(field, character)
		position++
	}
	return csv_flush(rows, fields, field)
}

// Reports whether a quote inside a quoted field is a doubled-quote escape.
func csv_is_escaped_quote(data []byte, position int) (escaped bool) {
	if position+1 >= len(data) {
		return false
	}
	return data[position+1] == '"'
}

// Flushes any pending field and row left when the input ends without a newline.
func csv_flush(rows [][]string, fields []string, field []byte) (flushed [][]string) {
	if len(fields) == 0 {
		if len(field) == 0 {
			return rows
		}
	}
	fields = append(fields, string(field))
	return append(rows, fields)
}

// Csv_Emit serializes a table (a list of records) to CSV, quoting fields that
// need it. A non-table value is an error.
func Csv_Emit(value Value) (text string, err error) {
	if value.Kind != VALUE_KIND_LIST {
		return "", fmt.Errorf("to csv needs a list of records")
	}
	columns := table_columns(value.Items)
	lines := []string{csv_line(columns)}
	for item_index := 0; item_index < len(value.Items); item_index++ {
		lines = append(lines, csv_line(csv_cells(value.Items[item_index], columns)))
	}
	return strings.Join(lines, "\n") + "\n", nil
}

// The CSV cells of one record, one per column (a missing field is empty).
func csv_cells(value Value, columns []string) (cells []string) {
	cells = []string{}
	for column_index := 0; column_index < len(columns); column_index++ {
		cells = append(cells, cell_text(field_value(value.Fields, columns[column_index])))
	}
	return cells
}

// Joins fields into a CSV line, quoting each as needed.
func csv_line(fields []string) (line string) {
	quoted := []string{}
	for field_index := 0; field_index < len(fields); field_index++ {
		quoted = append(quoted, csv_quote(fields[field_index]))
	}
	return strings.Join(quoted, ",")
}

// Quotes a field if it contains a comma, quote, or newline; doubles inner quotes.
func csv_quote(field string) (quoted string) {
	if !csv_needs_quote(field) {
		return field
	}
	return "\"" + strings.ReplaceAll(field, "\"", "\"\"") + "\""
}

// Reports whether a field needs CSV quoting.
func csv_needs_quote(field string) (needs bool) {
	return strings.ContainsAny(field, ",\"\n\r")
}
