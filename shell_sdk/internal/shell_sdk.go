// Package shell_sdk is a walled-garden structured-data CLI: one multi-call
// binary, symlinked to bare verb names, passes typed values through pipes as a
// custom binary format and renders a table at a terminal. This is the pure
// library; package main binds the real IO.
package shell_sdk

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"strconv"
	"strings"

	"local/james-orcales/shared/cli"
	invariant "local/james-orcales/shared/invariant/default"
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
	// Read_Stdin returns the bounded contents of standard input, or an error when
	// the input exceeds the bound, so a truncated value never parses silently.
	Read_Stdin func() (data []byte, err error)
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
	program := verb_program()
	if cli.Handle_Completion(program, input.Arguments, input.Output) {
		return EXIT_SUCCESS
	}
	command, parse_err := cli.Program_Parse(&program, input.Arguments)
	if errors.Is(parse_err, cli.Help_Requested) {
		// -help short-circuits parsing in cli, so per-verb usage works even when a
		// required argument is absent — the guarantee shell_sdk used to hand-roll.
		cli.Print_Requested_Help(input.Output, program, command)
		return EXIT_SUCCESS
	}
	if parse_err != nil {
		fmt.Fprintf(input.Error_Output, "shell_sdk: %v\n\n", parse_err)
		cli.Print_Help(input.Error_Output, program)
		return EXIT_USAGE
	}
	// The install verb is the bootstrap: it is not a data transform, so it runs on the
	// bare binary (via cli's busybox self-dispatch) instead of the wire pipeline.
	if command.Label == "install" {
		return run_install(input, cli.Get_Option(command.Arguments, "dir").Value.(string))
	}
	mode := resolve_output_mode(&program, input.Stdout_Is_Terminal)
	kind, _ := verb_kind_of(command.Label)
	return run_and_write(input, kind, command, mode)
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

// The install verb symlinks the data verbs into a directory. It is the bootstrap verb,
// run on the bare binary before the links exist, not a data transform.
const VERB_KIND_INSTALL Verb_Kind = 18

// Output_Mode selects how a verb presents its result.
type Output_Mode int

// The wire mode emits the binary format for the next sibling.
const OUTPUT_MODE_WIRE Output_Mode = 0

// The JSON mode emits text, leaving the garden.
const OUTPUT_MODE_JSON Output_Mode = 1

// The table mode renders an aligned table for a terminal.
const OUTPUT_MODE_TABLE Output_Mode = 2

// Runs one verb and writes its output, or reports a verb-prefixed runtime error and
// fails. A parse error is caught earlier, in Main, as a usage error.
func run_and_write(
	input *Main_Input, kind Verb_Kind, command cli.Command, mode Output_Mode,
) (status_code int) {
	output, run_err := run_verb(input, kind, command, mode)
	if run_err != nil {
		fmt.Fprintf(input.Error_Output, "shell_sdk: %s: %v\n", command.Label, run_err)
		return EXIT_FAILURE
	}
	_, write_err := input.Output.Write(output)
	if write_err != nil {
		return EXIT_FAILURE
	}
	return EXIT_SUCCESS
}

// A verb's definition: its link name, help summary, the cli shape that parses its
// command line, and the transform kind it dispatches to. The verb table built from
// these is the single source of truth for dispatch, the install fan-out, and help.
type Verb_Descriptor struct {
	// Kind is the transform this verb dispatches to.
	Kind Verb_Kind
	// Label is the verb's link name.
	Label string
	// Summary is the one-line help description.
	Summary string
	// Arguments is the cli positional shape that parses the command line.
	Arguments []cli.Option
	// Flags is the cli flag shape this verb accepts.
	Flags []cli.Option
}

// The verb table: the one place a verb's name, shape, and transform are declared.
func verb_table() (verbs []Verb_Descriptor) {
	return []Verb_Descriptor{
		{Kind: VERB_KIND_INSTALL, Label: "install",
			Summary:   "symlink the verbs into a directory",
			Arguments: string_arguments("dir")},
		{Kind: VERB_KIND_LOAD, Label: "load", Summary: "read a file into the garden",
			Arguments: string_arguments("file")},
		{Kind: VERB_KIND_FROM, Label: "from", Summary: "parse stdin as json or csv",
			Arguments: format_argument()},
		{Kind: VERB_KIND_TO, Label: "to", Summary: "serialize to json or csv",
			Arguments: format_argument()},
		{Kind: VERB_KIND_GET, Label: "get", Summary: "navigate a dotted cell path",
			Arguments: string_arguments("path")},
		{Kind: VERB_KIND_PICK, Label: "pick", Summary: "keep only the named fields",
			Arguments: variadic_argument("field")},
		{Kind: VERB_KIND_FILTER, Label: "filter", Summary: "keep matching rows",
			Arguments: filter_arguments()},
		{Kind: VERB_KIND_FIRST, Label: "first", Summary: "first element, or -count",
			Flags: count_flag()},
		{Kind: VERB_KIND_SORT_BY, Label: "sort-by", Summary: "order records by a field",
			Arguments: string_arguments("field")},
		{Kind: VERB_KIND_GROUP_BY, Label: "group-by", Summary: "bucket records by a field",
			Arguments: string_arguments("field")},
		{Kind: VERB_KIND_DISTINCT, Label: "distinct", Summary: "drop duplicate elements"},
		{Kind: VERB_KIND_REVERSE, Label: "reverse", Summary: "reverse a list"},
		{Kind: VERB_KIND_FINAL, Label: "final", Summary: "last element, or -count",
			Flags: count_flag()},
		{Kind: VERB_KIND_COLUMNS, Label: "columns", Summary: "the record keys as a list"},
		{Kind: VERB_KIND_REJECT, Label: "reject", Summary: "drop the named fields",
			Arguments: variadic_argument("field")},
		{Kind: VERB_KIND_RELABEL, Label: "relabel", Summary: "rename a field",
			Arguments: string_arguments("old", "new")},
		{Kind: VERB_KIND_COUNT, Label: "count", Summary: "the number of items or fields"},
		{Kind: VERB_KIND_WRAP, Label: "wrap", Summary: "wrap the value in a record",
			Arguments: string_arguments("name")},
		{Kind: VERB_KIND_FLATTEN, Label: "flatten", Summary: "flatten one level of lists"},
	}
}

// Builds the required string positional arguments with the given labels.
func string_arguments(names ...string) (arguments []cli.Option) {
	for name_index := 0; name_index < len(names); name_index++ {
		arguments = append(arguments, cli.New_Argument[string](
			cli.New_Argument_Input{Label: names[name_index]}))
	}
	return arguments
}

// Builds a single variadic string argument that collects the trailing positionals.
func variadic_argument(name string) (arguments []cli.Option) {
	return []cli.Option{cli.New_Variadic[string](cli.New_Variadic_Input{Label: name})}
}

// Builds the foreign-format argument shared by the from and to verbs, constrained to the
// formats shell_sdk can parse and emit. cli rejects anything else at parse time, so
// parse_format and emit_format never see an unknown format.
func format_argument() (arguments []cli.Option) {
	return []cli.Option{cli.New_Enum_Argument(cli.New_Enum_Argument_Input[string]{
		Label: "format", Enum: []string{"json", "csv"}, Description: "the foreign format",
	})}
}

// Builds the filter verb's arguments: a field path, a word operator constrained to the
// supported comparisons, and a target value. cli rejects an unknown operator at parse
// time, so verb_filter is reached only with a known one.
func filter_arguments() (arguments []cli.Option) {
	return []cli.Option{
		cli.New_Argument[string](cli.New_Argument_Input{Label: "field"}),
		cli.New_Enum_Argument(cli.New_Enum_Argument_Input[string]{
			Label: "operator", Enum: []string{"eq", "ne", "lt", "le", "gt", "ge"},
			Description: "comparison",
		}),
		cli.New_Argument[string](cli.New_Argument_Input{Label: "value"}),
	}
}

// Builds the -count flag that first and final read for their element count.
func count_flag() (flags []cli.Option) {
	return []cli.Option{cli.New_Flag(cli.New_Flag_Input[int]{
		Label: "count", Description: "how many elements",
	})}
}

// Builds the cli program: every verb is a multicall command selected by its link
// name, with -json and -table as global output-mode flags.
func verb_program() (program cli.Program) {
	table := verb_table()
	commands := []cli.Command{}
	for verb_index := 0; verb_index < len(table); verb_index++ {
		descriptor := table[verb_index]
		commands = append(commands, cli.Command{
			Label: descriptor.Label, Description: descriptor.Summary,
			Arguments: descriptor.Arguments, Flags: descriptor.Flags,
		})
	}
	return cli.New_Multicall(cli.New_Multicall_Input{
		Label: "shell_sdk", Description: "a structured-data toolkit",
		Global_Flags: output_flags(), Commands: commands,
	})
}

// The global output-mode flags: -json and -table force a rendering; the default is a
// table at a terminal, else the binary wire format between sibling verbs.
func output_flags() (flags []cli.Option) {
	return []cli.Option{
		cli.New_Flag(cli.New_Flag_Input[bool]{Label: "json", Description: "emit JSON"}),
		cli.New_Flag(cli.New_Flag_Input[bool]{
			Label: "table", Description: "render a table",
		}),
	}
}

// Resolves a verb link name to its transform kind, from the verb table.
func verb_kind_of(label string) (kind Verb_Kind, known bool) {
	table := verb_table()
	for verb_index := 0; verb_index < len(table); verb_index++ {
		if table[verb_index].Label == label {
			return table[verb_index].Kind, true
		}
	}
	return VERB_KIND_LOAD, false
}

// The verb link names install fans the binary out into, from the verb table. install
// itself is excluded: it is the bootstrap verb, run on the bare binary, and a link named
// "install" on PATH would shadow the system install(1).
func verb_names() (names []string) {
	table := verb_table()
	for verb_index := 0; verb_index < len(table); verb_index++ {
		if table[verb_index].Label == "install" {
			continue
		}
		names = append(names, table[verb_index].Label)
	}
	return names
}

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

// Resolves the output mode from the parsed global flags: a forcing flag wins, then a
// table at a terminal, else the binary wire format between sibling verbs.
func resolve_output_mode(
	program *cli.Program, stdout_is_terminal bool,
) (mode Output_Mode) {
	if cli.Get_Option(program.Global_Flags, "json").Value.(bool) {
		return OUTPUT_MODE_JSON
	}
	if cli.Get_Option(program.Global_Flags, "table").Value.(bool) {
		return OUTPUT_MODE_TABLE
	}
	if stdout_is_terminal {
		return OUTPUT_MODE_TABLE
	}
	return OUTPUT_MODE_WIRE
}

// Null_Value builds the null value.
func Null_Value() (value Value) {
	return Value{Kind: VALUE_KIND_NULL}
}

// The lowercase name of a value kind, for a wrong-shape error message.
func kind_name(kind Value_Kind) (name string) {
	switch kind {
	case VALUE_KIND_NULL:
		return "null"
	case VALUE_KIND_BOOLEAN:
		return "boolean"
	case VALUE_KIND_NUMBER:
		return "number"
	case VALUE_KIND_STRING:
		return "string"
	case VALUE_KIND_LIST:
		return "list"
	case VALUE_KIND_RECORD:
		return "record"
	}
	return "value"
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
type Wire_Work struct {
	// Is_Value selects Value over Bytes as this item's payload.
	Is_Value bool
	// Value is the value to emit, when Is_Value.
	Value Value
	// Bytes is a literal run to append, such as an encoded field name.
	Bytes []byte
}

// One container the decoder is filling in from the byte stream.
type Wire_Frame struct {
	// Kind is the container kind being filled: list or record.
	Kind Value_Kind
	// Remaining_Count is how many elements are still to read before it closes.
	Remaining_Count int
	// Items holds the list elements gathered so far.
	Items []Value
	// Fields holds the record fields gathered so far.
	Fields []Field
	// Pending_Name is the field name awaiting its value.
	Pending_Name string
	// Has_Name reports whether Pending_Name is set.
	Has_Name bool
}

// One step read from the stream: a finished scalar or empty container, or a
// non-empty container that opens a new frame.
type Wire_Read struct {
	// Is_Scalar reports that the step finished a scalar or empty container.
	Is_Scalar bool
	// Value is the finished value, when Is_Scalar.
	Value Value
	// Is_Open reports that the step opens a new container frame.
	Is_Open bool
	// Kind is the opened container's kind, when Is_Open.
	Kind Value_Kind
	// Count is the opened container's element count, when Is_Open.
	Count int
}

// Wire_Encode serializes a value to a complete wire stream. The walk is
// iterative over an explicit stack, since recursion is banned.
func Wire_Encode(value Value) (encoded []byte) {
	output := []byte(WIRE_MAGIC)
	output = append(output, WIRE_VERSION)
	stack := []Wire_Work{{Is_Value: true, Value: value}}
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
func wire_emit(value Value) (header []byte, children []Wire_Work) {
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
func wire_items_work(items []Value) (work []Wire_Work) {
	for item_index := len(items) - 1; item_index >= 0; item_index-- {
		work = append(work, Wire_Work{Is_Value: true, Value: items[item_index]})
	}
	return work
}

// Builds the child work for a record: a literal name run then the field value,
// reversed for LIFO order.
func wire_fields_work(fields []Field) (work []Wire_Work) {
	for field_index := len(fields) - 1; field_index >= 0; field_index-- {
		field := fields[field_index]
		work = append(work, Wire_Work{Is_Value: true, Value: field.Value})
		name := wire_append_bytes(nil, field.Name)
		work = append(work, Wire_Work{Is_Value: false, Bytes: name})
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
	frames := []Wire_Frame{}
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
func wire_needs_name(frames []Wire_Frame) (needs bool) {
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
func wire_set_name(frames []Wire_Frame, name string) (updated []Wire_Frame) {
	top := frames[len(frames)-1]
	top.Pending_Name = name
	top.Has_Name = true
	frames[len(frames)-1] = top
	return frames
}

// Reads one value head: a finished scalar, or an opening container.
func wire_read_value(data []byte, at int) (read Wire_Read, next int, err error) {
	if at >= len(data) {
		return Wire_Read{}, at, fmt.Errorf("wire: truncated value")
	}
	kind := Value_Kind(data[at])
	body := at + 1
	switch kind {
	case VALUE_KIND_NULL:
		return Wire_Read{Is_Scalar: true, Value: Null_Value()}, body, nil
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
	return Wire_Read{}, at, fmt.Errorf("wire: unknown tag %d", data[at])
}

// Reads a one-byte boolean.
func wire_read_boolean(data []byte, at int) (read Wire_Read, next int, err error) {
	if at >= len(data) {
		return Wire_Read{}, at, fmt.Errorf("wire: truncated boolean")
	}
	built := Boolean_Value(data[at] != 0)
	return Wire_Read{Is_Scalar: true, Value: built}, at + 1, nil
}

// Reads a length-prefixed number or string.
func wire_read_text(data []byte, at int, kind Value_Kind) (read Wire_Read, next int, err error) {
	text, after, blob_err := wire_read_blob(data, at)
	if blob_err != nil {
		return Wire_Read{}, at, blob_err
	}
	built := Number_Value(text)
	if kind == VALUE_KIND_STRING {
		built = String_Value(text)
	}
	return Wire_Read{Is_Scalar: true, Value: built}, after, nil
}

// Reads a container head: an empty one finishes as a scalar, a non-empty one
// opens a frame.
func wire_read_container(
	data []byte, at int, kind Value_Kind,
) (read Wire_Read, next int, err error) {
	count, after, count_err := wire_read_count(data, at)
	if count_err != nil {
		return Wire_Read{}, at, count_err
	}
	if count == 0 {
		return Wire_Read{Is_Scalar: true, Value: wire_empty(kind)}, after, nil
	}
	return Wire_Read{Is_Open: true, Kind: kind, Count: count}, after, nil
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
func wire_place(frames []Wire_Frame, read Wire_Read) (updated []Wire_Frame, root Value, done bool) {
	if read.Is_Open {
		opened := Wire_Frame{Kind: read.Kind, Remaining_Count: read.Count}
		return append(frames, opened), Value{}, false
	}
	return wire_deliver(frames, read.Value)
}

// Attaches a finished value to the open frame, popping and cascading any frames
// that complete until one stays open or the root is reached.
func wire_deliver(frames []Wire_Frame, value Value) (updated []Wire_Frame, root Value, done bool) {
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
func wire_attach(frame Wire_Frame, value Value) (updated Wire_Frame) {
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
func wire_frame_value(frame Wire_Frame) (value Value) {
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
type Json_Frame struct {
	// Is_Array marks an array frame; otherwise it is an object frame.
	Is_Array bool
	// Items holds the array elements gathered so far.
	Items []Value
	// Fields holds the object fields gathered so far.
	Fields []Field
	// Pending_Key is the object key awaiting its value.
	Pending_Key string
	// Has_Key reports whether Pending_Key is set.
	Has_Key bool
}

// Json_Parse parses a JSON document into a value, preserving object key order
// and keeping numbers as their exact text. The walk is iterative over a frame
// stack, and numbers never become float64, so the package stays deterministic.
func Json_Parse(data []byte) (value Value, err error) {
	frames := []Json_Frame{}
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
	frames []Json_Frame, kind int, text string,
) (updated []Json_Frame, root Value, done bool, err error) {
	switch kind {
	case JSON_TOKEN_COLON, JSON_TOKEN_COMMA:
		return frames, Value{}, false, nil
	case JSON_TOKEN_OBJECT_OPEN:
		return append(frames, Json_Frame{Is_Array: false}), Value{}, false, nil
	case JSON_TOKEN_ARRAY_OPEN:
		return append(frames, Json_Frame{Is_Array: true}), Value{}, false, nil
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
func json_is_key(frames []Json_Frame, kind int) (is_key bool) {
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
func json_set_key(frames []Json_Frame, key string) (updated []Json_Frame) {
	top := frames[len(frames)-1]
	top.Pending_Key = key
	top.Has_Key = true
	frames[len(frames)-1] = top
	return frames
}

// Places a finished value into the open frame, or returns it as the root.
func json_place(frames []Json_Frame, value Value) (updated []Json_Frame, root Value, done bool) {
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
func json_close(frames []Json_Frame) (updated []Json_Frame, root Value, done bool) {
	top := frames[len(frames)-1]
	frames = frames[:len(frames)-1]
	return json_place(frames, json_frame_value(top))
}

// Builds the finished container value from a completed frame.
func json_frame_value(frame Json_Frame) (value Value) {
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
		return json_read_literal(&Json_Read_Literal_Input{
			Data: data, At: position, Word: "true", Kind: JSON_TOKEN_TRUE,
		})
	case 'f':
		return json_read_literal(&Json_Read_Literal_Input{
			Data: data, At: position, Word: "false", Kind: JSON_TOKEN_FALSE,
		})
	case 'n':
		return json_read_literal(&Json_Read_Literal_Input{
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
type Json_Read_Literal_Input struct {
	// Data is the input being scanned.
	Data []byte
	// At is the offset the literal starts at.
	At int
	// Word is the exact keyword expected: true, false, or null.
	Word string
	// Kind is the token kind to return on a match.
	Kind int
}

// Reads a fixed keyword literal such as true, false, or null.
func json_read_literal(
	input *Json_Read_Literal_Input,
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
type Json_Emit_Work struct {
	// Is_Value selects Value over Text as this item's payload.
	Is_Value bool
	// Value is the value to emit, when Is_Value.
	Value Value
	// Text is a literal run of output such as a brace, comma, or quoted key.
	Text string
}

// Json_Emit serializes a value to compact JSON, preserving record key order. The
// walk is iterative over an explicit stack, since recursion is banned.
func Json_Emit(value Value) (text string) {
	output := []byte{}
	stack := []Json_Emit_Work{{Is_Value: true, Value: value}}
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
func json_emit_pieces(value Value) (pieces []Json_Emit_Work) {
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
func json_literal_piece(text string) (pieces []Json_Emit_Work) {
	return []Json_Emit_Work{{Is_Value: false, Text: text}}
}

// Renders true or false.
func json_boolean_text(flag bool) (text string) {
	if flag {
		return "true"
	}
	return "false"
}

// Breaks a list into bracketed, comma-separated element pieces.
func json_list_pieces(items []Value) (pieces []Json_Emit_Work) {
	pieces = append(pieces, Json_Emit_Work{Is_Value: false, Text: "["})
	for item_index := 0; item_index < len(items); item_index++ {
		if item_index > 0 {
			pieces = append(pieces, Json_Emit_Work{Is_Value: false, Text: ","})
		}
		pieces = append(pieces, Json_Emit_Work{Is_Value: true, Value: items[item_index]})
	}
	pieces = append(pieces, Json_Emit_Work{Is_Value: false, Text: "]"})
	return pieces
}

// Breaks a record into braced, comma-separated "key":value pieces.
func json_record_pieces(fields []Field) (pieces []Json_Emit_Work) {
	pieces = append(pieces, Json_Emit_Work{Is_Value: false, Text: "{"})
	for field_index := 0; field_index < len(fields); field_index++ {
		if field_index > 0 {
			pieces = append(pieces, Json_Emit_Work{Is_Value: false, Text: ","})
		}
		key := json_quote(fields[field_index].Name) + ":"
		pieces = append(pieces, Json_Emit_Work{Is_Value: false, Text: key})
		value := fields[field_index].Value
		pieces = append(pieces, Json_Emit_Work{Is_Value: true, Value: value})
	}
	pieces = append(pieces, Json_Emit_Work{Is_Value: false, Text: "}"})
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
	input *Main_Input, kind Verb_Kind, command cli.Command, mode Output_Mode,
) (output []byte, err error) {
	positionals := command_positionals(command)
	switch kind {
	case VERB_KIND_LOAD:
		return run_load(input, positionals, mode)
	case VERB_KIND_FROM:
		return run_from(input, positionals, mode)
	case VERB_KIND_TO:
		return run_to(input, positionals)
	}
	data, read_err := input.Read_Stdin()
	if read_err != nil {
		return nil, read_err
	}
	value, decode_err := decode_input(data)
	if decode_err != nil {
		return nil, decode_err
	}
	transformed, transform_err := apply_transform(&Transform_Input{
		Kind:        kind,
		Positionals: positionals,
		Count:       verb_count_flag(command, kind),
		Value:       value,
	})
	if transform_err != nil {
		return nil, transform_err
	}
	return encode_output(transformed, mode), nil
}

// The inputs a transforming verb needs: the positional arguments, the -count flag
// (for first and final), and the decoded value to transform.
type Transform_Input struct {
	// Kind is the transforming verb to apply.
	Kind Verb_Kind
	// Positionals is the verb's positional arguments.
	Positionals []string
	// Count is the -count flag, read by first and final.
	Count int
	// Value is the decoded value to transform.
	Value Value
}

// The positional argument values of a parsed command, flattened in declaration
// order: each scalar string, then each element of a trailing variadic.
func command_positionals(command cli.Command) (positionals []string) {
	positionals = []string{}
	for argument_index := 0; argument_index < len(command.Arguments); argument_index++ {
		values := option_strings(command.Arguments[argument_index])
		positionals = append(positionals, values...)
	}
	return positionals
}

// The string values an option carries: one for a scalar, each element for a slice.
func option_strings(option cli.Option) (values []string) {
	text, is_string := option.Value.(string)
	if is_string {
		return []string{text}
	}
	slice, is_slice := option.Value.([]string)
	if is_slice {
		return slice
	}
	return []string{}
}

// The -count flag for first and final; zero for every other verb, which has none.
func verb_count_flag(command cli.Command, kind Verb_Kind) (count int) {
	switch kind {
	case VERB_KIND_FIRST, VERB_KIND_FINAL:
		return cli.Get_Option(command.Flags, "count").Value.(int)
	}
	return 0
}

// Runs the load verb: read a file, decode by sniffing, re-encode for the mode.
func run_load(
	input *Main_Input, positionals []string, mode Output_Mode,
) (output []byte, err error) {
	name := positionals[0]
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

// Runs the from verb: parse stdin as the explicit foreign format.
func run_from(
	input *Main_Input, positionals []string, mode Output_Mode,
) (output []byte, err error) {
	data, read_err := input.Read_Stdin()
	if read_err != nil {
		return nil, read_err
	}
	value, parse_err := parse_format(positionals[0], data)
	if parse_err != nil {
		return nil, parse_err
	}
	return encode_output(value, mode), nil
}

// Runs the to verb: decode stdin and serialize out of the garden.
func run_to(input *Main_Input, positionals []string) (output []byte, err error) {
	data, read_err := input.Read_Stdin()
	if read_err != nil {
		return nil, read_err
	}
	value, decode_err := decode_input(data)
	if decode_err != nil {
		return nil, decode_err
	}
	text, emit_err := emit_format(positionals[0], value)
	if emit_err != nil {
		return nil, emit_err
	}
	return []byte(text), nil
}

// Parses a named foreign format into a value. The format enum guarantees a known name,
// so the switch is total.
func parse_format(format string, data []byte) (value Value, err error) {
	switch format {
	case "json":
		return Json_Parse(data)
	case "csv":
		return Csv_Parse(data)
	}
	invariant.Always(false, "the format enum guarantees json or csv")
	return Value{}, nil
}

// Serializes a value to a named foreign format, ending in a newline. The format enum
// guarantees a known name, so the switch is total.
func emit_format(format string, value Value) (text string, err error) {
	switch format {
	case "json":
		return Json_Emit(value) + "\n", nil
	case "csv":
		return Csv_Emit(value)
	}
	invariant.Always(false, "the format enum guarantees json or csv")
	return "", nil
}

// Applies a transforming verb to a decoded value.
func apply_transform(input *Transform_Input) (result Value, err error) {
	switch input.Kind {
	case VERB_KIND_GET:
		return verb_get(input.Positionals, input.Value)
	case VERB_KIND_PICK:
		return verb_pick(input.Positionals, input.Value)
	case VERB_KIND_FILTER:
		return verb_filter(input.Positionals, input.Value)
	case VERB_KIND_FIRST:
		return verb_first(input.Count, input.Value)
	}
	return apply_transform_more(input)
}

// Applies the second half of the transforming verbs.
func apply_transform_more(input *Transform_Input) (result Value, err error) {
	switch input.Kind {
	case VERB_KIND_SORT_BY:
		return verb_sort_by(input.Positionals, input.Value)
	case VERB_KIND_GROUP_BY:
		return verb_group_by(input.Positionals, input.Value)
	case VERB_KIND_DISTINCT:
		return verb_distinct(input.Positionals, input.Value)
	case VERB_KIND_REVERSE:
		return verb_reverse(input.Positionals, input.Value)
	case VERB_KIND_FINAL:
		return verb_final(input.Count, input.Value)
	case VERB_KIND_COLUMNS:
		return verb_columns(input.Positionals, input.Value)
	case VERB_KIND_REJECT:
		return verb_reject(input.Positionals, input.Value)
	case VERB_KIND_RELABEL:
		return verb_relabel(input.Positionals, input.Value)
	case VERB_KIND_COUNT:
		return verb_count(input.Positionals, input.Value)
	case VERB_KIND_WRAP:
		return verb_wrap(input.Positionals, input.Value)
	case VERB_KIND_FLATTEN:
		return verb_flatten(input.Positionals, input.Value)
	}
	return input.Value, nil
}

// Decodes stdin by sniffing: empty input is a null value, then a sibling's wire,
// foreign JSON, or raw text.
func decode_input(data []byte) (value Value, err error) {
	if len(data) == 0 {
		return Null_Value(), nil
	}
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

// Runs the get verb: navigate a dotted cell path. cli guarantees the path argument.
func verb_get(arguments []string, value Value) (result Value, err error) {
	return get_path(value, arguments[0]), nil
}

// Runs the pick verb: keep only the named fields of each record. The field list is
// variadic, so cli permits zero of them and the verb rejects that itself.
func verb_pick(arguments []string, value Value) (result Value, err error) {
	if len(arguments) == 0 {
		return Value{}, fmt.Errorf("needs at least one field")
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
type Filter_Test struct {
	// Item is the row being tested.
	Item Value
	// Field is the cell path to compare.
	Field string
	// Operator is the word comparison to apply.
	Operator string
	// Target is the value to compare against.
	Target Value
}

// Runs the filter verb: keep list rows whose cell satisfies the word operator. cli
// guarantees the field, operator, and value arguments.
func verb_filter(arguments []string, value Value) (result Value, err error) {
	operator := arguments[1]
	if value.Kind != VALUE_KIND_LIST {
		return value, nil
	}
	field := arguments[0]
	target := parse_argument_value(arguments[2])
	kept := []Value{}
	for item_index := 0; item_index < len(value.Items); item_index++ {
		item := value.Items[item_index]
		test := Filter_Test{Item: item, Field: field, Operator: operator, Target: target}
		if filter_keeps(test) {
			kept = append(kept, item)
		}
	}
	return List_Value(kept), nil
}

// Reports whether a row's field satisfies the operator against the target.
func filter_keeps(test Filter_Test) (keep bool) {
	cell := get_path(test.Item, test.Field)
	order, comparable := compare_values(Value_Pair{Left: cell, Right: test.Target})
	if !comparable {
		return false
	}
	return operator_satisfied(test.Operator, order)
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

// Runs the first verb: the first element, or the first count as a list.
func verb_first(count int, value Value) (result Value, err error) {
	if value.Kind != VALUE_KIND_LIST {
		return value, nil
	}
	if count == 0 {
		return first_element(value.Items), nil
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
type Value_Pair struct {
	// Left is the left operand of the comparison.
	Left Value
	// Right is the right operand of the comparison.
	Right Value
}

// Orders two same-kind values, reporting whether they are comparable. Numbers
// compare as exact rationals, so the result is deterministic without float64.
func compare_values(pair Value_Pair) (order int, comparable bool) {
	left := pair.Left
	right := pair.Right
	if left.Kind != right.Kind {
		return 0, false
	}
	if left.Kind == VALUE_KIND_NUMBER {
		return compare_numbers(Number_Pair{Left: left.Number, Right: right.Number})
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
type Number_Pair struct {
	// Left is the left number's exact source text.
	Left string
	// Right is the right number's exact source text.
	Right string
}

// Orders two numbers as exact rationals.
func compare_numbers(pair Number_Pair) (order int, comparable bool) {
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
	if value.Kind != VALUE_KIND_LIST {
		return Value{}, fmt.Errorf("expected a list, got %s", kind_name(value.Kind))
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
		order, comparable := compare_values(Value_Pair{Left: item_key, Right: existing_key})
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
	if value.Kind != VALUE_KIND_LIST {
		return Value{}, fmt.Errorf("expected a list, got %s", kind_name(value.Kind))
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
		return Value{}, fmt.Errorf("expected a list, got %s", kind_name(value.Kind))
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
		return Value{}, fmt.Errorf("expected a list, got %s", kind_name(value.Kind))
	}
	reversed := []Value{}
	for item_index := len(value.Items) - 1; item_index >= 0; item_index-- {
		reversed = append(reversed, value.Items[item_index])
	}
	return List_Value(reversed), nil
}

// Runs the final verb: the last element, or the last count as a list.
func verb_final(count int, value Value) (result Value, err error) {
	if value.Kind != VALUE_KIND_LIST {
		return value, nil
	}
	if count == 0 {
		return last_element(value.Items), nil
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

// Runs the reject verb: drop the named fields of each record. The field list is
// variadic, so cli permits zero of them and the verb rejects that itself.
func verb_reject(arguments []string, value Value) (result Value, err error) {
	if len(arguments) == 0 {
		return Value{}, fmt.Errorf("needs at least one field")
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
type Relabel_Input struct {
	// Old is the field name to replace.
	Old string
	// New is the field name to use instead.
	New string
}

// Runs the relabel verb: rename a field across records. cli guarantees the old and
// new arguments.
func verb_relabel(arguments []string, value Value) (result Value, err error) {
	return relabel_value(value, Relabel_Input{Old: arguments[0], New: arguments[1]}), nil
}

// Renames a field, mapping across a list of records.
func relabel_value(value Value, names Relabel_Input) (result Value) {
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
func relabel_one(value Value, names Relabel_Input) (result Value) {
	if value.Kind == VALUE_KIND_RECORD {
		return Record_Value(relabel_fields(value.Fields, names))
	}
	return value
}

// Renames the matching field of a record's fields.
func relabel_fields(fields []Field, names Relabel_Input) (renamed []Field) {
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

// Runs the wrap verb: wrap the value in a single-field record. cli guarantees the
// name argument.
func verb_wrap(arguments []string, value Value) (result Value, err error) {
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
		record := csv_record(&Csv_Record_Input{Header: header, Row: rows[row_index]})
		records = append(records, record)
	}
	return List_Value(records), nil
}

// The header and one row a CSV record is built from, bundled because the types
// repeat.
type Csv_Record_Input struct {
	// Header is the column names from the first CSV row.
	Header []string
	// Row is the cell values for the record being built.
	Row []string
}

// Builds one record from a header and a row, inferring each cell's type.
func csv_record(input *Csv_Record_Input) (value Value) {
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
		return "", fmt.Errorf("needs a list of records, got %s", kind_name(value.Kind))
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
