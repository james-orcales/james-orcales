// Package main is the harness command: the thin composition root over the library tier, and the
// one place allowed to bind the real loop, allocate, and pump. `go run ./harness <directory>`.
package main

import "local/james-orcales/harness/pkg"
import "local/james-orcales/harness/pkg/requirements"
import "local/james-orcales/shared/cli"
import "local/james-orcales/shared/filepath"
import "local/james-orcales/shared/go/ast"
import "local/james-orcales/shared/go/format"
import "local/james-orcales/shared/go/printer"
import "local/james-orcales/shared/go/token"
import "local/james-orcales/shared/jlog/default"
import shared_os "local/james-orcales/shared/os"
import "local/james-orcales/shared/os/default"
import "local/james-orcales/shared/sim/nbio"
import system_nbio "local/james-orcales/shared/sim/nbio/default"
import "local/james-orcales/shared/sim/time"
import system_time "local/james-orcales/shared/sim/time/default"
import "local/james-orcales/shared/slices"
import "local/james-orcales/shared/strings"

// STANDARD_OUTPUT is the descriptor diagnostics go to.
const STANDARD_OUTPUT nbio.File = 1

// STANDARD_ERROR is the descriptor a failed step goes to.
const STANDARD_ERROR nbio.File = 2

// CAPACITY bounds every backend store: a run keeps one operation live and spawns nothing.
const CAPACITY = 8

// PUMP_DURATION outlasts READ_TIMEOUT: equal deadlines race, and a read must retire first.
const PUMP_DURATION = harness.READ_TIMEOUT + time.MINUTE

// LOOP_FAILURE is a fixed line, because rendering the loop error would allocate.
const LOOP_FAILURE = "harness: failed to drive the loop\n"

func main() {
	system := os.New_Operating_System(&os.Host{Self_Exec: main_self_exec_workspace()})
	state := system_nbio.Operating_System{}
	clock := system_time.New_Operating_System_Clock()
	memory := main_loop_memory()
	loop, driver, err := system_nbio.New_Operating_System_IO(&state, memory, clock, CAPACITY, 0)
	// A failed loop skips Deinit: work stays in flight, and Deinit asserts all of it retired.
	if err != nil {
		os.Standard_Error_Write([]byte(LOOP_FAILURE))
		os.OS_Exit(system, shared_os.Exit_Code(harness.EXIT_FAILURE))
	}
	output := jlog.Stream_File{Storage: loop.Storage, File: STANDARD_OUTPUT}
	error_output := jlog.Stream_File{Storage: loop.Storage, File: STANDARD_ERROR}
	arguments := make(shared_os.Arguments, slices.COUNT_MAXIMUM)
	arguments = arguments[:os.OS_Arguments(system, arguments)]
	runner := harness.Runner{
		Runner_Fields: harness.Runner_Fields{
			Output:       jlog.File_To_Stream(&output),
			Error_Output: jlog.File_To_Stream(&error_output),
			Clock:        clock,
			Arguments:    arguments,
			Memory:       main_memory(),
		},
	}
	// Built once: a closure literal inside the pump would allocate on every pass.
	done := func() (finished bool) { return bool(runner.Work_Queued(&runner)) }
	harness.Main(&runner, loop)
	for !runner.Stopped(&runner) {
		for runner.Rearm(&runner) {
		}
		// A stopped run counts as queued work, thus this pass returns at once.
		_, drive_err := nbio.Driver_Run_Until(driver, PUMP_DURATION, done)
		if drive_err != nil {
			os.Standard_Error_Write([]byte(LOOP_FAILURE))
			os.OS_Exit(system, shared_os.Exit_Code(harness.EXIT_FAILURE))
		}
	}
	os.OS_Exit(system, shared_os.Exit_Code(runner.Status))
}

// The host requires exec storage; harness never replaces its image, so the pages stay untouched.
func main_self_exec_workspace() (workspace os.Self_Exec_Workspace_Pointer) {
	defer func() {
		os.Self_Exec_Workspace_Pointer_Invariants(workspace, "main_self_exec.workspace")
	}()
	path := make(os.Self_Exec_Path_Bytes, os.SELF_EXEC_PATH_STORAGE_BYTES)
	arguments := make(os.Self_Exec_Argument_Bytes, os.SELF_EXEC_VECTOR_STORAGE_BYTES)
	environment := make(os.Self_Exec_Environment_Bytes, os.SELF_EXEC_VECTOR_STORAGE_BYTES)
	count := os.SELF_EXEC_VECTOR_POINTER_COUNT
	argument_pointers := make(os.Self_Exec_Argument_Pointer_Vector, count)
	environment_pointers := make(os.Self_Exec_Environment_Pointer_Vector, count)
	workspace = &os.Self_Exec_Workspace{Path: &path, Arguments: &arguments}
	workspace.Environment = &environment
	workspace.Argument_Pointers = &argument_pointers
	workspace.Environment_Pointers = &environment_pointers
	return workspace
}

// Every store is allocated once, at its maximum, so the run never grows one.
func main_memory() (memory harness.Memory_Handle) {
	defer func() { harness.Memory_Handle_Invariants(memory, "main_memory.memory") }()
	memory = &harness.Memory{}
	fields := printer.Printer_Fields{}
	fields.Nodes.Values = make(printer.Node_Stack, printer.DEPTH_MAXIMUM)
	fields.Widths.Values = make(printer.Field_Widths, printer.FIELD_COUNT_MAXIMUM)
	fields.Results.Values = make(printer.Result_Tokens, printer.RESULT_COUNT_MAXIMUM)
	fields.Aliases.Values = make(printer.Import_Aliases, printer.IMPORT_COUNT_MAXIMUM)
	fields.Paths.Values = make(printer.Import_Paths, printer.IMPORT_COUNT_MAXIMUM)
	fields.Comments.Values = make(printer.Comment_Positions, printer.COMMENT_COUNT_MAXIMUM)
	fields.Counts.Values = make(printer.Counts, printer.COUNT_SLOT_COUNT)
	fields.Flags.Values = make(printer.Flags, printer.FLAG_COUNT)
	tree := ast.Parse_State{}
	tree.Tokens.Values = make(ast.Tokens, ast.TOKEN_COUNT_MAXIMUM)
	tree.Nodes.Values = make(ast.Nodes, ast.NODE_COUNT_MAXIMUM)
	tree.Parents.Values = make(ast.Parents, ast.DEPTH_MAXIMUM)
	tree.Last_Children.Values = make(ast.Last_Children, ast.DEPTH_MAXIMUM)
	tree.Earlier_Children.Values = make(ast.Earlier_Children, ast.DEPTH_MAXIMUM)
	formatter := format.Formatter{}
	formatter.Places.Values = make(format.Places, format.PLACE_SLOT_COUNT)
	memory.Formatter = &formatter
	memory.Printer = &printer.Printer{Printer_Fields: fields}
	memory.Parse_State = &tree
	memory.Form = make(printer.Form, printer.FORM_SIZE_MAXIMUM)
	memory.Lines = &token.Line_Index{Starts: make(token.Line_Starts, token.LINE_COUNT_MAXIMUM)}
	// Every path slot holds a whole path, thus a copy into one never grows it. The three tables
	// share the traversal bound of shared/filepath, thus one loop fills them.
	memory.Paths = make(filepath.Path_Storage, harness.FILE_COUNT_MAXIMUM)
	memory.Directories = make(harness.Directory_Paths, harness.DIRECTORY_COUNT_MAXIMUM)
	memory.Packages = make(harness.Packages, harness.PACKAGE_COUNT_MAXIMUM)
	for index := range memory.Paths {
		memory.Paths[index] = make(filepath.Slice, 0, filepath.PATH_SIZE_MAXIMUM)
		memory.Directories[index] = make(filepath.Slice, 0, filepath.PATH_SIZE_MAXIMUM)
		memory.Packages[index].Path = make(filepath.Slice, 0, filepath.PATH_SIZE_MAXIMUM)
	}
	memory.Current = make(filepath.Slice, 0, filepath.PATH_SIZE_MAXIMUM)
	memory.Entries = make(filepath.Directory_Entries, harness.ENTRY_COUNT_MAXIMUM)
	buffer_size := nbio.DIRECTORY_BUFFER_SIZE_MAXIMUM
	memory.Directory_Buffer = make(filepath.Directory_Buffer, buffer_size)
	memory.Source = make(harness.File_Bytes, harness.SOURCE_STORAGE_SIZE)
	memory.Line = make(harness.Line_Storage, 0, harness.LINE_SIZE_MAXIMUM)
	memory.Json = make(harness.Json_Storage, harness.JSON_STORAGE_SIZE)
	memory.Declared_Arguments = make(cli.Arguments, harness.ARGUMENT_DECLARATION_COUNT)
	memory.Declared_Flags = make(cli.Flags, harness.FLAG_DECLARATION_COUNT)
	argument_count := harness.ARGUMENT_DECLARATION_COUNT
	memory.Command_Arguments = make(cli.Command_Argument_Storage, argument_count)
	memory.Command_Flags = make(cli.Command_Flag_Storage, harness.FLAG_DECLARATION_COUNT)
	memory.Filled = make(cli.Filled_Options, harness.FILLED_COUNT)
	memory.Positionals = make(cli.Positionals, harness.TOKEN_COUNT_MAXIMUM)
	memory.Slice_Named = make(cli.Named_Variadic_Tokens, harness.TOKEN_COUNT_MAXIMUM)
	memory.Failure = make(cli.Parse_Failure_Storage, cli.FAILURE_SIZE_MAXIMUM)
	help := make(strings.Bytes, strings.TEXT_SIZE_MAXIMUM)
	memory.Help = &cli.Output{Builder: cli.Output_Builder{Storage: help}}
	memory.Program = &cli.Program{}
	memory.Parser = &cli.Parser{}
	memory.File_Packages = make(harness.Package_Indices, harness.FILE_COUNT_MAXIMUM)
	memory.Read_Path = make(filepath.Slice, 0, filepath.PATH_SIZE_MAXIMUM)
	memory.Requirements = main_checker(memory)
	return memory
}

// The checker shares the run's parse state and line index, since a run parses one file at a time.
func main_checker(memory harness.Memory_Handle) (checker requirements.Checker_Handle) {
	defer func() { requirements.Checker_Handle_Invariants(checker, "main_checker.checker") }()
	harness.Memory_Handle_Invariants(memory, "main_checker.memory")
	checker = &requirements.Checker{}
	checker.Markdown = make(requirements.Markdown_Bytes, requirements.MARKDOWN_STORAGE_SIZE)
	checker.Parse_State, checker.Lines = memory.Parse_State, memory.Lines
	count := requirements.HEADING_COUNT_MAXIMUM
	checker.Headings = make(requirements.Headings, count)
	checker.Leaves = make(requirements.Leaves, count)
	checker.Declarations = make(requirements.Declarations, count)
	checker.Name_Storage = make(requirements.Text_Storage, 0, requirements.NAME_SIZE_MAXIMUM)
	size := requirements.MESSAGE_SIZE_MAXIMUM
	checker.Message_Storage = make(requirements.Text_Storage, 0, size)
	return checker
}

// Sizes every bounded queue the backend retains for one operation at a time.
func main_loop_memory() (memory system_nbio.Operating_System_Memory) {
	memory.Timeouts = make([]*nbio.Completion, CAPACITY)
	memory.Completed = make([]*nbio.Completion, CAPACITY)
	memory.Operations = make([]system_nbio.Operating_System_Operation, CAPACITY)
	path_size := system_nbio.OPERATING_SYSTEM_PATH_BYTES_MAXIMUM
	memory.Operation_File_Paths = make([]byte, CAPACITY*path_size)
	memory.Operation_Socket_Addresses = make([]byte, CAPACITY*system_nbio.SOCKET_ADDRESS_BYTES)
	memory.Operation_Registry = make([]*system_nbio.Operating_System_Operation, CAPACITY)
	memory.Descriptors = make([]system_nbio.Operating_System_Descriptor, CAPACITY)
	memory.Signal_Waiters = make([]system_nbio.Signal_Waiter, CAPACITY)
	memory.Spawns = make([]system_nbio.Spawn, CAPACITY)
	memory.Spawn_Registry = make([]*system_nbio.Spawn, CAPACITY)
	pipe_size := system_nbio.PROCESS_PIPE_BUFFER_COUNT * system_nbio.PROCESS_PIPE_BYTES
	memory.Spawn_Pipe_Buffers = make([]byte, CAPACITY*pipe_size)
	memory.Spawn_Paths = make([]byte, CAPACITY*system_nbio.PROCESS_PATH_BYTES)
	memory.Spawn_Arguments = make([]string, CAPACITY*system_nbio.PROCESS_ARGUMENT_COUNT_MAXIMUM)
	child_count := system_nbio.PROCESS_CHILD_DESCRIPTOR_COUNT
	memory.Spawn_Child_Files = make([]uintptr, CAPACITY*child_count)
	memory.Platform_Operations = make([]*system_nbio.Operating_System_Operation, CAPACITY)
	memory.Platform_Changes = make([]system_nbio.Kernel_Event, system_nbio.POLL_EVENTS_MAX)
	memory.Platform_Events = make([]system_nbio.Kernel_Event, system_nbio.POLL_EVENTS_MAX)
	return memory
}
