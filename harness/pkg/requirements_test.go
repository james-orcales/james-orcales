package harness_test

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
import "local/james-orcales/shared/testify"

// Test_Diagnostics_Tier_One verifies an auto-fixable diagnostic carries its fix.
func Test_Diagnostics_Tier_One(t *testing.T) {
	runner := checked(memory_new(), UNFORMATTED)
	testify.Equal(t, harness.TIER_ONE, runner.Diagnostic.Tier)
	testify.Equal(t, CLEAN, string(runner.Diagnostic.Fix))
}

// Test_Diagnostics_Tier_Two verifies a diagnostic that is not auto-fixable carries no fix.
func Test_Diagnostics_Tier_Two(t *testing.T) {
	runner := checked(memory_new(), REFUSED)
	testify.Equal(t, harness.TIER_TWO, runner.Diagnostic.Tier)
	testify.Equal(t, 0, len(runner.Diagnostic.Fix))
}

// Test_Command_Usage verifies a command line shared/cli refuses fails with its error, and -h
// writes the help.
func Test_Command_Usage(t *testing.T) {
	memory := memory_new()
	cases := [][]string{{"harness"}, {"harness", "a", "b"}, {"harness", "a", "-jsn"}}
	for _, arguments := range cases {
		subject := fixture_new(t, memory, 1)
		testify.True(t, fixture_run(subject, arguments))
		testify.Equal(t, harness.EXIT_FAILURE, subject.Runner.Status)
		testify.Equal(t, "", fixture_output(subject))
		errors := fixture_errors(subject)
		testify.True(t, bool(strings.Has_Prefix(strings.Text(errors), "harness: ")), errors)
		testify.True(t, bool(strings.Has_Suffix(strings.Text(errors), "\n")), errors)
		testify.True(t, len(errors) > len("harness: \n"), errors)
	}
	subject := fixture_new(t, memory, 1)
	testify.True(t, fixture_run(subject, []string{"harness", "-h"}))
	testify.Equal(t, harness.EXIT_SUCCESS, subject.Runner.Status)
	help := fixture_output(subject)
	testify.True(t, bool(strings.Contains(strings.Text(help), "<directory: string>")), help)
	testify.True(t, bool(strings.Contains(strings.Text(help), "json")), help)
	testify.Equal(t, "", fixture_errors(subject))
}

// Test_Command_Recursive verifies every Go file at any depth is checked, and nothing else.
func Test_Command_Recursive(t *testing.T) {
	memory := memory_new()
	for seed := uint64(1); seed < SEED_COUNT+1; seed++ {
		subject := fixture_new(t, memory, seed)
		fixture_package(t, subject)
		testify.True(t, fixture_run(subject, []string{"harness", "package"}))
		testify.Equal(t, harness.EXIT_FAILURE, subject.Runner.Status)
		testify.Equal(
			t,
			"package/nested/b.go: not formatted\n"+
				"package/nested/deep/c.go:3:9: "+
				string(ast.Failure_Message(ast.FAILURE_SYNTAX))+
				"\n",
			fixture_output(subject),
		)
		testify.Equal(t, "", fixture_errors(subject))
	}
}

// Test_Command_Json verifies -json writes each diagnostic as one flat JSON line.
func Test_Command_Json(t *testing.T) {
	memory := memory_new()
	for seed := uint64(1); seed < SEED_COUNT+1; seed++ {
		subject := fixture_new(t, memory, seed)
		fixture_package(t, subject)
		testify.True(t, fixture_run(subject, []string{"harness", "package", "-json"}))
		testify.Equal(t, harness.EXIT_FAILURE, subject.Runner.Status)
		testify.Equal(
			t,
			`{"path":"package/nested/b.go","line":0,"column":0,"tier":1,`+
				`"rule":"unformatted","message":"not formatted"}`+
				"\n"+
				`{"path":"package/nested/deep/c.go","line":3,"column":9,"tier":2,`+
				`"rule":"refused","message":"`+
				string(ast.Failure_Message(ast.FAILURE_SYNTAX))+
				`"}`+
				"\n",
			fixture_output(subject),
		)
		testify.Equal(t, "", fixture_errors(subject))
	}
}

// Test_Command_Fix verifies -fix replaces each auto-fixable file with its fix and reports only what
// stays.
func Test_Command_Fix(t *testing.T) {
	memory := memory_new()
	for seed := uint64(1); seed < SEED_COUNT+1; seed++ {
		subject := fixture_new(t, memory, seed)
		fixture_package(t, subject)
		testify.True(t, fixture_run(subject, []string{"harness", "package", "-fix"}))
		testify.Equal(t, harness.EXIT_FAILURE, subject.Runner.Status)
		refused := string(ast.Failure_Message(ast.FAILURE_SYNTAX))
		output := fixture_output(subject)
		testify.Equal(t, "package/nested/deep/c.go:3:9: "+refused+"\n", output)
		testify.Equal(t, "", fixture_errors(subject))
		testify.Equal(t, CLEAN, fixture_read(t, subject, "package/nested/b.go"))
		testify.Equal(t, REFUSED, fixture_read(t, subject, "package/nested/deep/c.go"))

		subject = fixture_new(t, memory, seed)
		fixture_package_directory(t, subject, "package")
		fixture_file(t, subject, "package/b.go", UNFORMATTED)
		testify.True(t, fixture_run(subject, []string{"harness", "package", "-fix"}))
		testify.Equal(t, harness.EXIT_SUCCESS, subject.Runner.Status)
		testify.Equal(t, "", fixture_errors(subject))
		testify.Equal(t, CLEAN, fixture_read(t, subject, "package/b.go"))
	}
}

// Test_Command_Requirements verifies each package's contract findings follow the Go file
// diagnostics, at the contract file of their package, in text and in flat JSON.
func Test_Command_Requirements(t *testing.T) {
	memory := memory_new()
	for seed := uint64(1); seed < SEED_COUNT+1; seed++ {
		output := requirements_run(
			t,
			memory,
			seed,
			"package",
			UNFORMATTED,
			REQUIREMENTS_ABSENT,
			REQUIREMENTS_ABSENT,
			"package",
		)
		testify.Equal(
			t,
			"package/a.go: not formatted\n"+
				MARKDOWN_PATH+
				": The package has no REQUIREMENTS.md. Add REQUIREMENTS.md.\n",
			output,
		)
		main_file := "package main\n\nfunc main() {}\n"
		output = requirements_run(
			t,
			memory,
			seed,
			"package",
			main_file,
			REQUIREMENTS_ABSENT,
			REQUIREMENTS_ABSENT,
			"package",
		)
		requirements_passed(t, output)
		output = requirements_run(
			t,
			memory,
			seed,
			"package",
			CLEAN,
			"\n# A\n\n1\n\n2\n",
			"package a\n\nfunc Test_A() {}\n",
			"package",
		)
		testify.Equal(
			t,
			MARKDOWN_PATH+
				`:6: The section "A" has a blank line between two body lines. `+
				"Remove the blank line.\n",
			output,
		)
		output = requirements_run(
			t,
			memory,
			seed,
			"package",
			CLEAN,
			CONTRACT_MARKDOWN,
			REQUIREMENTS_ABSENT,
			"package",
			"-json",
		)
		testify.Equal(
			t,
			`{"path":"package/requirements_test.go","line":0,"column":0,"tier":2,`+
				`"rule":"requirements",`+
				`"message":"The package has no requirements_test.go. Add requirements_test.go."}`+
				"\n",
			output,
		)
	}
}

// Test_Command_Status verifies a package without diagnostics exits 0 and one with any exits 1.
func Test_Command_Status(t *testing.T) {
	memory := memory_new()
	for seed := uint64(1); seed < SEED_COUNT+1; seed++ {
		subject := fixture_new(t, memory, seed)
		fixture_package_directory(t, subject, "package")
		fixture_file(t, subject, "package/a.go", CLEAN)
		testify.True(t, fixture_run(subject, []string{"harness", "package"}))
		testify.Equal(t, harness.EXIT_SUCCESS, subject.Runner.Status)
		testify.Equal(t, "", fixture_errors(subject))

		subject = fixture_new(t, memory, seed)
		testify.True(t, fixture_run(subject, []string{"harness", "absent"}))
		testify.Equal(t, harness.EXIT_FAILURE, subject.Runner.Status)
		testify.Equal(t, "harness: failed to walk: absent\n", fixture_errors(subject))
	}
}

// Test_Command_Passed verifies a clean run writes the pass line and its elapsed seconds, with or
// without -json.
func Test_Command_Passed(t *testing.T) {
	memory := memory_new()
	cases := []struct {
		Arguments []string
		Prefix    string
		Suffix    string
	}{
		{[]string{"harness", "package"}, "✓ all checks passed\nelapsed_sec=", "\n"},
		{[]string{"harness", "package", "-json"}, `{"passed":true,"elapsed_sec":"`, "\"}\n"},
	}
	for seed := uint64(1); seed < SEED_COUNT+1; seed++ {
		for _, one := range cases {
			subject := fixture_new(t, memory, seed)
			fixture_package_directory(t, subject, "package")
			fixture_file(t, subject, "package/a.go", CLEAN)
			testify.True(t, fixture_run(subject, one.Arguments))
			output := fixture_output(subject)
			opened := strings.Has_Prefix(strings.Text(output), strings.Text(one.Prefix))
			testify.True(t, bool(opened), output)
			closed := strings.Has_Suffix(strings.Text(output), strings.Text(one.Suffix))
			testify.True(t, bool(closed), output)
			elapsed := output[len(one.Prefix) : len(output)-len(one.Suffix)]
			point := len(elapsed) - len(".000")
			testify.True(t, point > 0, output)
			testify.Equal(t, byte('.'), elapsed[point], output)
			for index, character := range []byte(elapsed) {
				if index == point {
					continue
				}
				testify.True(t, character-'0' < 10, output)
			}
		}
	}
}

// Test_Format_Clean verifies a canonical file yields no diagnostic.
func Test_Format_Clean(t *testing.T) {
	runner := checked(memory_new(), CLEAN)
	testify.Equal(t, harness.TIER_NONE, runner.Diagnostic.Tier)
}

// Test_Format_Unformatted verifies a file off the canonical form yields a tier one diagnostic.
func Test_Format_Unformatted(t *testing.T) {
	runner := checked(memory_new(), UNFORMATTED)
	testify.Equal(t, harness.TIER_ONE, runner.Diagnostic.Tier)
	testify.Equal(t, harness.RULE_UNFORMATTED, runner.Diagnostic.Rule)
	testify.Equal(t, CLEAN, string(runner.Diagnostic.Fix))
}

// Test_Format_Refused verifies a refused file yields a tier two diagnostic naming the parser's
// refusal and where it stands.
func Test_Format_Refused(t *testing.T) {
	memory := memory_new()
	runner := checked(memory, REFUSED)
	testify.Equal(t, harness.TIER_TWO, runner.Diagnostic.Tier)
	testify.Equal(t, harness.RULE_REFUSED, runner.Diagnostic.Rule)
	testify.Equal(t, ast.FAILURE_SYNTAX, runner.Diagnostic.Failure)
	testify.Equal(t, token.Line(3), runner.Diagnostic.Position.Line)
	testify.Equal(t, token.Column(9), runner.Diagnostic.Position.Column)
	runner = checked(memory, GROUPED)
	testify.Equal(t, harness.RULE_REFUSED, runner.Diagnostic.Rule)
	testify.Equal(t, ast.FAILURE_DECLARATION_GROUP, runner.Diagnostic.Failure)
	testify.Equal(t, token.Line(3), runner.Diagnostic.Position.Line)
	testify.Equal(t, token.Column(8), runner.Diagnostic.Position.Column)
}

// Test_Format_Oversized verifies a file past the source bound yields a tier two diagnostic.
func Test_Format_Oversized(t *testing.T) {
	runner := checked(memory_new(), "")
	runner.Source_Count = token.SOURCE_SIZE_MAXIMUM + 1
	harness.Check(runner)
	testify.Equal(t, harness.TIER_TWO, runner.Diagnostic.Tier)
	testify.Equal(t, harness.RULE_OVERSIZED, runner.Diagnostic.Rule)
}

// Test_Allocation verifies a whole run, walk and checks and report, allocates nothing.
func Test_Allocation(t *testing.T) {
	memory := memory_new()
	cases := []struct {
		Arguments []string
		Setup     func(t *testing.T, subject *fixture)
	}{
		{[]string{"harness", "package"}, fixture_package},
		{[]string{"harness", "package", "-json"}, fixture_package},
		{[]string{"harness", "package", "-fix"}, fixture_package},
		{[]string{"harness"}, fixture_empty},
		{[]string{"harness", "-h"}, fixture_empty},
		{[]string{"harness", "absent"}, fixture_empty},
		{[]string{"harness", "package"}, fixture_clean},
		{[]string{"harness", "package", "-json"}, fixture_clean},
		{[]string{"harness", "package"}, fixture_faulty_contract},
		{[]string{"harness", "package", "-json"}, fixture_faulty_contract},
		{[]string{"harness", "package"}, fixture_uncontracted},
	}
	for seed := uint64(1); seed < SEED_COUNT+1; seed++ {
		for _, one := range cases {
			subject := fixture_new(t, memory, seed)
			one.Setup(t, subject)
			testify.Zero_Allocation(t, func() { fixture_run(subject, one.Arguments) })
			testify.True(t, bool(subject.Runner.Stopped(&subject.Runner)))
		}
	}
}

// SEED_COUNT keeps the sweep wide enough to reorder retirements and fast.
const SEED_COUNT = 16

// NODE_CAPACITY leaves room for the tree the seed generates beside the package each test builds.
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

// REFUSED is a file the parser refuses as bad syntax at the brace.
const REFUSED = "package a\n\nfunc f( {}\n"

// GROUPED is valid Go the parser refuses as a declaration group at the parenthesis.
const GROUPED = "package a\n\nimport (\n\t\"x\"\n)\n"

// CONTRACT_MARKDOWN is the smallest REQUIREMENTS.md that breaks no rule.
const CONTRACT_MARKDOWN = "\n# Contract\n\nIt holds.\n"

// CONTRACT_TEST mirrors CONTRACT_MARKDOWN.
const CONTRACT_TEST = "package a\n\nfunc Test_Contract(t *T) {}\n"

// REQUIREMENTS_ABSENT marks a case whose package carries no such file.
const REQUIREMENTS_ABSENT = "\x00"

// MARKDOWN_PATH is where each requirements case writes its REQUIREMENTS.md.
const MARKDOWN_PATH = "package/REQUIREMENTS.md"

// TEST_PATH is where each requirements case writes its requirements_test.go.
const TEST_PATH = "package/requirements_test.go"

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

// Checks one source through a runner that owns memory, as a run would after reading it.
func checked(memory harness.Memory, source string) (runner *harness.Runner) {
	runner = &harness.Runner{Runner_Fields: harness.Runner_Fields{Memory: &memory}}
	runner.Source_Count = harness.Source_Count(copy(memory.Source, source))
	harness.Check(runner)
	return runner
}

// Every seed gets its own tree and loop, while the run stores stay shared: the test, like package
// main, allocates them once, and the run allocates nothing.
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
	checker.Message_Storage = make(
		requirements.Text_Storage,
		0,
		requirements.MESSAGE_SIZE_MAXIMUM,
	)
	return checker
}

// Runs one package holding a Go file and the given contract files, and returns what it wrote.
func requirements_run(
	t *testing.T,
	memory harness.Memory,
	seed uint64,
	directory string,
	source string,
	markdown string,
	test string,
	arguments ...string,
) (output string) {
	t.Helper()
	subject := fixture_new(t, memory, seed)
	fixture_directory(t, subject, directory)
	fixture_file(t, subject, directory+"/a.go", source)
	if markdown != REQUIREMENTS_ABSENT {
		fixture_file(t, subject, directory+"/REQUIREMENTS.md", markdown)
	}
	if test != REQUIREMENTS_ABSENT {
		fixture_file(t, subject, directory+"/requirements_test.go", test)
	}
	testify.True(t, fixture_run(subject, append([]string{"harness"}, arguments...)))
	testify.Equal(t, "", fixture_errors(subject))
	return fixture_output(subject)
}

// A run without diagnostics writes the pass line first.
func requirements_passed(t *testing.T, output string) {
	t.Helper()
	opened := strings.Has_Prefix(strings.Text(output), "✓ all checks passed\n")
	testify.True(t, bool(opened), output)
}

// Builds a package whose files sort to every outcome: a clean file at the top, a file off the
// canonical form one level down, a refused file two levels down, and a non-Go file the run must
// skip though its bytes would refuse.
func fixture_package(t *testing.T, subject *fixture) {
	t.Helper()
	fixture_package_directory(t, subject, "package")
	fixture_package_directory(t, subject, "package/nested")
	fixture_package_directory(t, subject, "package/nested/deep")
	fixture_file(t, subject, "package/a.go", CLEAN)
	fixture_file(t, subject, "package/notes.txt", REFUSED)
	fixture_file(t, subject, "package/nested/b.go", UNFORMATTED)
	fixture_file(t, subject, "package/nested/deep/c.go", REFUSED)
}

func fixture_empty(t *testing.T, subject *fixture) {
	t.Helper()
}

// A contracted package without a diagnostic, which ends in the pass line.
func fixture_clean(t *testing.T, subject *fixture) {
	t.Helper()
	fixture_package_directory(t, subject, "package")
	fixture_file(t, subject, "package/a.go", CLEAN)
}

// A package whose contract breaks format and test order rules, and whose subpackage has none.
func fixture_faulty_contract(t *testing.T, subject *fixture) {
	t.Helper()
	fixture_directory(t, subject, "package")
	fixture_directory(t, subject, "package/inner")
	fixture_file(t, subject, "package/a.go", CLEAN)
	fixture_file(t, subject, "package/inner/b.go", CLEAN)
	fixture_file(t, subject, "package/REQUIREMENTS.md", "Intro.\n\n# A\n\n1\n\n2\n3\n4\n\n### x y\n")
	fixture_file(t, subject, "package/requirements_test.go", "package a\n\nconst X = 1\n")
}

// A package whose contract lacks its test file.
func fixture_uncontracted(t *testing.T, subject *fixture) {
	t.Helper()
	fixture_directory(t, subject, "package")
	fixture_file(t, subject, "package/a.go", CLEAN)
	fixture_file(t, subject, "package/REQUIREMENTS.md", CONTRACT_MARKDOWN)
}

// Makes a package directory that already carries a contract, so a test of anything else reads
// only its own diagnostics.
func fixture_package_directory(t *testing.T, subject *fixture, path string) {
	t.Helper()
	fixture_directory(t, subject, path)
	fixture_file(t, subject, path+"/REQUIREMENTS.md", CONTRACT_MARKDOWN)
	fixture_file(t, subject, path+"/requirements_test.go", CONTRACT_TEST)
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
	testify.Equal(t, len(contents), subject.Setup.Completion.Data)
	nbio.IO_Close(subject.Loop, fixture_submit(subject), file, fixture_retired)
	fixture_wait(t, subject)
}

// Reads a whole fixture file back; a fixture file stays inside one simulated file's bytes.
func fixture_read(t *testing.T, subject *fixture, path string) (contents string) {
	t.Helper()
	nbio.Storage_Open_At(
		subject.Loop.Storage,
		fixture_submit(subject),
		nbio.DIRECTORY_CURRENT,
		path,
		nbio.Open_At_Options{Access: nbio.OPEN_READ_ONLY},
		fixture_retired,
	)
	fixture_wait(t, subject)
	file := nbio.File(subject.Setup.Completion.Data)
	buffer := make([]byte, nbio.SIM_FILE_BYTES_MAXIMUM)
	nbio.Storage_Read(
		subject.Loop.Storage,
		fixture_submit(subject),
		file,
		buffer,
		0,
		harness.READ_TIMEOUT,
		fixture_retired,
	)
	fixture_wait(t, subject)
	count := subject.Setup.Completion.Data
	nbio.IO_Close(subject.Loop, fixture_submit(subject), file, fixture_retired)
	fixture_wait(t, subject)
	return string(buffer[:count])
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
	testify.No_Error(t, err)
	testify.True(t, completed)
	testify.No_Error(t, subject.Setup.Completion.Error)
}

// Pumps one run to its stop. It reports rather than fails, so a zero-allocation check can wrap
// it whole.
func fixture_run(subject *fixture, arguments []string) (stopped bool) {
	subject.Runner.Arguments = arguments
	harness.Main(&subject.Runner, subject.Loop)
	for range PASS_COUNT {
		for subject.Runner.Rearm(&subject.Runner) {
		}
		if subject.Runner.Stopped(&subject.Runner) {
			return true
		}
		completed, err := nbio.Driver_Run_Until(subject.Driver, PUMP_DURATION, subject.Done)
		if err != nil {
			return false
		}
		if !completed {
			return false
		}
	}
	return bool(subject.Runner.Stopped(&subject.Runner))
}

func fixture_output(subject *fixture) (output string) {
	return string(subject.Output.Memory[:subject.Output.Cursor])
}

func fixture_errors(subject *fixture) (errors string) {
	return string(subject.Errors.Memory[:subject.Errors.Cursor])
}
