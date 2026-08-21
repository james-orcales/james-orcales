// Package simulation_test is the harness blackbox witness: every run goes through pkg.Main on a
// seeded simulated loop, and this test root alone holds the Driver.
package simulation_test

import "testing"
import "unsafe"

import "local/james-orcales/harness/pkg"
import "local/james-orcales/harness/pkg/requirements"
import "local/james-orcales/shared/cli"
import "local/james-orcales/shared/filepath"
import "local/james-orcales/shared/go/ast"
import "local/james-orcales/shared/go/format"
import "local/james-orcales/shared/go/printer"
import "local/james-orcales/shared/go/token"
import "local/james-orcales/shared/sim/nbio"
import "local/james-orcales/shared/sim/time"
import "local/james-orcales/shared/strings"

// Test_Simulation sweeps seeds over every outcome a run reaches: usage, a failed walk, a clean
// package, and a package holding each diagnostic.
func Test_Simulation(t *testing.T) {
	memory := memory_new()
	for seed := uint64(1); seed < SEED_COUNT+1; seed++ {
		subject := fixture_new(t, memory, seed)
		fixture_run(t, subject, []string{"harness"})
		subject = fixture_new(t, memory, seed)
		fixture_run(t, subject, []string{"harness", "absent"})
		subject = fixture_new(t, memory, seed)
		fixture_directory(t, subject, "clean")
		fixture_file(t, subject, "clean/a.go", CLEAN)
		fixture_file(t, subject, "clean/REQUIREMENTS.md", "\n# Contract\n\nIt holds.\n")
		fixture_file(
			t,
			subject,
			"clean/requirements_test.go",
			"package a\n\nfunc Test_Contract() {}\n",
		)
		fixture_run(t, subject, []string{"harness", "clean"})
		fixture_run(t, subject, []string{"harness", "clean", "-json"})
		subject = fixture_new(t, memory, seed)
		fixture_directory(t, subject, "package")
		fixture_directory(t, subject, "package/nested")
		fixture_file(t, subject, "package/a.go", CLEAN)
		fixture_file(t, subject, "package/notes.txt", REFUSED)
		fixture_file(t, subject, "package/nested/b.go", UNFORMATTED)
		fixture_file(t, subject, "package/nested/c.go", REFUSED)
		fixture_run(t, subject, []string{"harness", "package"})
		fixture_run(t, subject, []string{"harness", "package", "-json"})
		fixture_run(t, subject, []string{"harness", "-h"})
		fixture_run(t, subject, []string{"harness", "package", "-fix"})
		subject = fixture_new(t, memory, seed)
		fixture_directory(t, subject, "contract")
		fixture_directory(t, subject, "contract/main")
		fixture_file(t, subject, "contract/a.go", CLEAN)
		fixture_file(t, subject, "contract/REQUIREMENTS.md", FAULTY_CONTRACT)
		fixture_file(t, subject, "contract/requirements_test.go", FAULTY_TEST)
		fixture_file(
			t,
			subject,
			"contract/main/main.go",
			"package main\n\nfunc main() {}\n",
		)
		fixture_run(t, subject, []string{"harness", "contract"})
		fixture_run(t, subject, []string{"harness", "contract", "-json"})
		subject = fixture_new(t, memory, seed)
		fixture_directory(t, subject, "contract")
		fixture_file(t, subject, "contract/a.go", CLEAN)
		fixture_file(t, subject, "contract/REQUIREMENTS.md", "\n# Contract\n\nIt holds.\n")
		fixture_run(t, subject, []string{"harness", "contract"})
	}
}

// FAULTY_CONTRACT breaks every format rule once: a preamble, a wrong level, a heading without
// fencing, a bad character, a repeat, an orphan, an empty leaf, a gap, and a long section.
const FAULTY_CONTRACT = "Intro.\n\n### Orphan\n\nBody.\n\n# A-B\n\n## Two\n\n# A-B\nBody.\n\n" +
	"# Empty\n\n# Tall\n\n1\n\n2\n3\n4\n\n# Two words\n\n### x\n\nBody.\n"

// FAULTY_TEST puts a declaration in the first slot and misses the rest.
const FAULTY_TEST = "package a\n\nconst X = 1\n\nfunc Test_Empty() {}\n"

// SEED_COUNT keeps the sweep wide enough to reorder retirements and fast.
const SEED_COUNT = 16

// NODE_CAPACITY leaves room for the tree the seed generates beside the package each run builds.
const NODE_CAPACITY = 64

// DESCRIPTOR_CAPACITY covers the one directory or file a run holds open at a time, with slack.
const DESCRIPTOR_CAPACITY = 8

// OPERATION_CAPACITY covers the one operation a run keeps in flight, with slack.
const OPERATION_CAPACITY = 8

// PASS_COUNT bounds pump passes; a test package submits well under a hundred operations.
const PASS_COUNT = 1024

// PUMP_DURATION is strictly longer than every timeout a run grants, so the operation retires
// before the root's guard does.
const PUMP_DURATION = harness.READ_TIMEOUT + time.MINUTE

// OUTPUT_BYTES holds every line a test package yields.
const OUTPUT_BYTES = 4096

// FILE_PERMISSIONS is what each fixture file carries; the run only reads.
const FILE_PERMISSIONS nbio.File_Permissions = 0o644

// DIRECTORY_PERMISSIONS is what each fixture directory carries.
const DIRECTORY_PERMISSIONS nbio.File_Permissions = 0o755

// CLEAN is a file in canonical form.
const CLEAN = "package a\n\nfunc f() {}\n"

// UNFORMATTED is CLEAN with a doubled space the canonical form drops.
const UNFORMATTED = "package a\n\nfunc  f() {}\n"

// REFUSED is a file the parser refuses.
const REFUSED = "package a\n\nfunc f( {}\n"

type setup struct {
	Completion nbio.Completion
	Retired    bool
}

type fixture struct {
	Simulation nbio.Sim
	Loop       nbio.IO
	Driver     nbio.Driver
	Output     nbio.Stream_Memory
	Errors     nbio.Stream_Memory
	Runner     harness.Runner
	Setup      setup
	Done       func() (finished bool)
}

// Every seed gets its own tree and loop, while the run stores stay shared: the test, like package
// main, allocates them once.
func fixture_new(t *testing.T, memory harness.Memory, seed uint64) (subject *fixture) {
	t.Helper()
	subject = &fixture{}
	nodes := make([]nbio.Sim_Node, NODE_CAPACITY)
	for index := range nodes {
		nodes[index].Name = make([]byte, nbio.SIM_PATH_COMPONENT_BYTES_MAXIMUM)
		nodes[index].Contents = make([]byte, nbio.SIM_FILE_BYTES_MAXIMUM)
	}
	descriptors := make([]nbio.Sim_Descriptor, DESCRIPTOR_CAPACITY)
	for index := range descriptors {
		descriptors[index].Address_IP = make([]byte, nbio.IPV6_ADDRESS_BYTES)
		descriptors[index].Peer_IP = make([]byte, nbio.IPV6_ADDRESS_BYTES)
	}
	clocks := make([]nbio.Sim_Clock, 1)
	operations := make([]nbio.Sim_Operation, OPERATION_CAPACITY)
	for index := range operations {
		operations[index].Address_IP = make([]byte, nbio.IPV6_ADDRESS_BYTES)
	}
	subject.Loop, subject.Driver = nbio.New_Simulated_IO(
		&subject.Simulation,
		seed,
		time.NANOSECOND,
		nbio.Sim_Memory{
			Nodes:       nodes,
			Descriptors: descriptors,
			Operations:  operations,
			Queue:       make([]*nbio.Completion, OPERATION_CAPACITY*2),
			Events:      make([]nbio.Virtual_Event, 1),
			Clocks:      clocks,
		},
	)
	subject.Output = nbio.Stream_Memory{Memory: make([]byte, OUTPUT_BYTES)}
	subject.Errors = nbio.Stream_Memory{Memory: make([]byte, OUTPUT_BYTES)}
	subject.Runner = harness.Runner{
		Runner_Fields: harness.Runner_Fields{
			Output:       nbio.Memory_To_Stream(&subject.Output),
			Error_Output: nbio.Memory_To_Stream(&subject.Errors),
			Clock:        nbio.Sim_Clock_To_Clock(&clocks[0]),
			Memory:       &memory,
		},
	}
	// Built once: a closure literal inside the pump would allocate on every pass.
	subject.Done = func() (finished bool) {
		return bool(subject.Runner.Work_Queued(&subject.Runner))
	}
	return subject
}

func fixture_directory(t *testing.T, subject *fixture, path string) {
	t.Helper()
	nbio.Storage_Mkdir_At(
		subject.Loop.Storage,
		fixture_submit(subject),
		nbio.DIRECTORY_CURRENT,
		path,
		DIRECTORY_PERMISSIONS,
		fixture_retired,
	)
	fixture_wait(t, subject)
}

func fixture_file(t *testing.T, subject *fixture, path string, contents string) {
	t.Helper()
	nbio.Storage_Open_At(
		subject.Loop.Storage,
		fixture_submit(subject),
		nbio.DIRECTORY_CURRENT,
		path,
		nbio.Open_At_Options{
			Access:      nbio.OPEN_WRITE_ONLY,
			Create:      true,
			Permissions: FILE_PERMISSIONS,
		},
		fixture_retired,
	)
	fixture_wait(t, subject)
	file := nbio.File(subject.Setup.Completion.Data)
	nbio.Storage_Write(
		subject.Loop.Storage,
		fixture_submit(subject),
		file,
		[]byte(contents),
		0,
		harness.READ_TIMEOUT,
		fixture_retired,
	)
	fixture_wait(t, subject)
	if subject.Setup.Completion.Data != len(contents) {
		t.Fatalf(
			"wrote %d bytes of %s, want %d",
			subject.Setup.Completion.Data,
			path,
			len(contents),
		)
	}
	nbio.IO_Close(subject.Loop, fixture_submit(subject), file, fixture_retired)
	fixture_wait(t, subject)
}

func fixture_submit(subject *fixture) (completion *nbio.Completion) {
	subject.Setup.Retired = false
	return &subject.Setup.Completion
}

func fixture_retired(completion nbio.Completion_Handle) {
	(*setup)(unsafe.Pointer(completion)).Retired = true
}

func fixture_retired_read(subject *fixture) (retired bool) {
	return subject.Setup.Retired
}

func fixture_wait(t *testing.T, subject *fixture) {
	t.Helper()
	completed, err := nbio.Driver_Run_Until(
		subject.Driver,
		PUMP_DURATION,
		func() (finished bool) { return fixture_retired_read(subject) },
	)
	if err != nil {
		t.Fatalf("drive setup: %v", err)
	}
	if !completed {
		t.Fatal("setup stalled past its deadline")
	}
	if subject.Setup.Completion.Error != nil {
		t.Fatalf("setup: %v", subject.Setup.Completion.Error)
	}
}

// The root alternation of package main: run each recorded continuation, then one pump pass.
func fixture_run(t *testing.T, subject *fixture, arguments []string) {
	t.Helper()
	subject.Runner.Arguments = arguments
	harness.Main(&subject.Runner, subject.Loop)
	for range PASS_COUNT {
		for subject.Runner.Rearm(&subject.Runner) {
		}
		if subject.Runner.Stopped(&subject.Runner) {
			return
		}
		completed, err := nbio.Driver_Run_Until(subject.Driver, PUMP_DURATION, subject.Done)
		if err != nil {
			t.Fatalf("drive: %v", err)
		}
		if !completed {
			t.Fatal("the run stalled past its deadline")
		}
	}
	t.Fatal("the run never stopped")
}

func memory_new() (memory harness.Memory) {
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
	memory.Paths = make(filepath.Path_Storage, harness.FILE_COUNT_MAXIMUM)
	for index := range memory.Paths {
		memory.Paths[index] = make(filepath.Slice, 0, filepath.PATH_SIZE_MAXIMUM)
	}
	memory.Directories = make(harness.Directory_Paths, harness.DIRECTORY_COUNT_MAXIMUM)
	for index := range memory.Directories {
		memory.Directories[index] = make(filepath.Slice, 0, filepath.PATH_SIZE_MAXIMUM)
	}
	memory.Current = make(filepath.Slice, 0, filepath.PATH_SIZE_MAXIMUM)
	memory.Entries = make(filepath.Directory_Entries, harness.ENTRY_COUNT_MAXIMUM)
	memory.Directory_Buffer = make(
		filepath.Directory_Buffer,
		nbio.DIRECTORY_BUFFER_SIZE_MAXIMUM,
	)
	memory.Source = make(harness.File_Bytes, harness.SOURCE_STORAGE_SIZE)
	memory.Line = make(harness.Line_Storage, 0, harness.LINE_SIZE_MAXIMUM)
	memory.Json = make(harness.Json_Storage, harness.JSON_STORAGE_SIZE)
	memory.Declared_Arguments = make(cli.Arguments, harness.ARGUMENT_DECLARATION_COUNT)
	memory.Declared_Flags = make(cli.Flags, harness.FLAG_DECLARATION_COUNT)
	memory.Command_Arguments = make(
		cli.Command_Argument_Storage,
		harness.ARGUMENT_DECLARATION_COUNT,
	)
	memory.Command_Flags = make(cli.Command_Flag_Storage, harness.FLAG_DECLARATION_COUNT)
	memory.Filled = make(cli.Filled_Options, harness.FILLED_COUNT)
	memory.Positionals = make(cli.Positionals, harness.TOKEN_COUNT_MAXIMUM)
	memory.Slice_Named = make(cli.Named_Variadic_Tokens, harness.TOKEN_COUNT_MAXIMUM)
	memory.Failure = make(cli.Parse_Failure_Storage, cli.FAILURE_SIZE_MAXIMUM)
	help := make(strings.Bytes, strings.TEXT_SIZE_MAXIMUM)
	memory.Help = &cli.Output{Builder: cli.Output_Builder{Storage: help}}
	memory.Program = &cli.Program{}
	memory.Parser = &cli.Parser{}
	memory.Packages = make(harness.Packages, harness.PACKAGE_COUNT_MAXIMUM)
	for index := range memory.Packages {
		memory.Packages[index].Path = make(filepath.Slice, 0, filepath.PATH_SIZE_MAXIMUM)
	}
	memory.File_Packages = make(harness.Package_Indices, harness.FILE_COUNT_MAXIMUM)
	memory.Read_Path = make(filepath.Slice, 0, filepath.PATH_SIZE_MAXIMUM)
	memory.Requirements = checker_new(memory)
	return memory
}

// The checker shares the run's parse state and line index, since a run parses one file at a time.
func checker_new(memory harness.Memory) (checker *requirements.Checker) {
	checker = &requirements.Checker{}
	checker.Markdown = make(requirements.Markdown_Bytes, requirements.MARKDOWN_STORAGE_SIZE)
	checker.Parse_State = memory.Parse_State
	checker.Lines = memory.Lines
	checker.Headings = make(requirements.Headings, requirements.HEADING_COUNT_MAXIMUM)
	checker.Leaves = make(requirements.Leaves, requirements.HEADING_COUNT_MAXIMUM)
	checker.Declarations = make(requirements.Declarations, requirements.HEADING_COUNT_MAXIMUM)
	checker.Name_Storage = make(requirements.Text_Storage, 0, requirements.NAME_SIZE_MAXIMUM)
	message_size := requirements.MESSAGE_SIZE_MAXIMUM
	checker.Message_Storage = make(requirements.Text_Storage, 0, message_size)
	return checker
}
