// Package harness is the pure library tier of the harness linter: it walks one Go package and
// reports each file that breaks a rule. Every effect is submitted to the injected IO surface —
// package main binds the real loop, allocates every store, and pumps.
package harness

import "reflect"
import "unsafe"

import "local/james-orcales/harness/pkg/requirements"
import "local/james-orcales/shared/bytes"
import "local/james-orcales/shared/cli"
import "local/james-orcales/shared/encoding/flatjson"
import "local/james-orcales/shared/filepath"
import "local/james-orcales/shared/go/ast"
import "local/james-orcales/shared/go/format"
import "local/james-orcales/shared/go/printer"
import "local/james-orcales/shared/go/token"
import "local/james-orcales/shared/os"
import "local/james-orcales/shared/sim/aver/default"
import "local/james-orcales/shared/sim/nbio"
import "local/james-orcales/shared/sim/time"
import "local/james-orcales/shared/slices"
import "local/james-orcales/shared/strconv"
import "local/james-orcales/shared/strings"

// TIER_NONE is the tier of a check that found nothing.
const TIER_NONE Tier = 0

// TIER_ONE is an auto-fixable diagnostic.
const TIER_ONE Tier = 1

// TIER_TWO is a diagnostic that is not auto-fixable.
const TIER_TWO Tier = 2

// RULE_NONE is the rule of a check that found nothing.
const RULE_NONE Rule = 0

// RULE_UNFORMATTED is a file off the canonical form.
const RULE_UNFORMATTED Rule = 1

// RULE_REFUSED is a file the parser refuses.
const RULE_REFUSED Rule = 2

// RULE_OVERSIZED is a file past token.SOURCE_SIZE_MAXIMUM.
const RULE_OVERSIZED Rule = 3

// MESSAGE_UNFORMATTED is what a line says of RULE_UNFORMATTED.
const MESSAGE_UNFORMATTED = "not formatted"

// MESSAGE_OVERSIZED is what a line says of RULE_OVERSIZED.
const MESSAGE_OVERSIZED = "exceeds source size"

// STEP_WALK is a failed directory open, read, or close, or a tree past its storage.
const STEP_WALK Step = 0

// STEP_OPEN is a failed file open.
const STEP_OPEN Step = 1

// STEP_READ is a failed file read.
const STEP_READ Step = 2

// STEP_CLOSE is a failed file close.
const STEP_CLOSE Step = 3

// STEP_REPORT is a diagnostic whose flat JSON outgrows flatjson.OUTPUT_SIZE_MAXIMUM, which only
// a path of escaped bytes reaches.
const STEP_REPORT Step = 4

// STEP_FIX is a failed open, write, or close of a fix over its file.
const STEP_FIX Step = 5

// RULE_REQUIREMENTS is a package contract that breaks the REQUIREMENTS.md doctrine.
const RULE_REQUIREMENTS Rule = 4

// RULE_TEXT_REQUIREMENTS names RULE_REQUIREMENTS in flat JSON.
const RULE_TEXT_REQUIREMENTS Report_Text = "requirements"

// PACKAGE_COUNT_MAXIMUM is one package per Go file at most.
const PACKAGE_COUNT_MAXIMUM = FILE_COUNT_MAXIMUM

// RESUME_FILE continues with the next Go file after a report.
const RESUME_FILE Resume = 0

// RESUME_PACKAGE continues with the next package after a report.
const RESUME_PACKAGE Resume = 1

// RESUME_REQUIREMENTS continues the open package's contract checks after a report.
const RESUME_REQUIREMENTS Resume = 2

// READING_SOURCE reads a Go file to check it.
const READING_SOURCE Reading = 0

// READING_MARKDOWN reads a REQUIREMENTS.md.
const READING_MARKDOWN Reading = 1

// READING_TEST reads a requirements_test.go.
const READING_TEST Reading = 2

// FLAG_FIX labels the flag that writes each fix over its file.
const FLAG_FIX = "fix"

// WRITE_TIMEOUT bounds one write of a fix to a local file.
const WRITE_TIMEOUT = READ_TIMEOUT

// RULE_TEXT_UNFORMATTED names RULE_UNFORMATTED in flat JSON.
const RULE_TEXT_UNFORMATTED Report_Text = "unformatted"

// RULE_TEXT_REFUSED names RULE_REFUSED in flat JSON.
const RULE_TEXT_REFUSED Report_Text = "refused"

// RULE_TEXT_OVERSIZED names RULE_OVERSIZED in flat JSON.
const RULE_TEXT_OVERSIZED Report_Text = "oversized"

// PROGRAM_LABEL names the program in help and in every diagnostic prefix.
const PROGRAM_LABEL = "harness"

// ARGUMENT_DIRECTORY labels the one positional argument.
const ARGUMENT_DIRECTORY = "directory"

// FLAG_JSON labels the flag that selects flat JSON diagnostics.
const FLAG_JSON = "json"

// ARGUMENT_DECLARATION_COUNT is the directory.
const ARGUMENT_DECLARATION_COUNT = 1

// FLAG_DECLARATION_COUNT is -json and -fix.
const FLAG_DECLARATION_COUNT = 2

// FILLED_COUNT holds one mark per declared argument and flag.
const FILLED_COUNT = ARGUMENT_DECLARATION_COUNT + FLAG_DECLARATION_COUNT

// TOKEN_COUNT_MAXIMUM bounds the command line tokens the parser holds; the root reads at most
// slices.COUNT_MAXIMUM arguments.
const TOKEN_COUNT_MAXIMUM = slices.COUNT_MAXIMUM

// PROGRAM_PREFIX starts the diagnostic of a refused command line.
const PROGRAM_PREFIX = PROGRAM_LABEL + ": "

// FAILURE_PREFIX starts the diagnostic of a failed step.
const FAILURE_PREFIX = PROGRAM_PREFIX + "failed to "

// JSON_STORAGE_SIZE holds one flat JSON diagnostic and its newline.
const JSON_STORAGE_SIZE = flatjson.OUTPUT_SIZE_MAXIMUM + len("\n")

// SEPARATOR stands between a path and what the line says of it.
const SEPARATOR = ": "

// PATH_SEPARATOR joins a directory and the name of one child.
const PATH_SEPARATOR = '/'

// GO_SUFFIX ends the name of every file the run checks.
const GO_SUFFIX = ".go"

// POSITION_NUMBER_COUNT is a line and a column.
const POSITION_NUMBER_COUNT = 2

// POSITION_SIZE_MAXIMUM holds a line and a column, each behind its colon.
const POSITION_SIZE_MAXIMUM = POSITION_NUMBER_COUNT * (len(":") + strconv.DIGIT_TEXT_SIZE_MAXIMUM)

// LINE_SIZE_MAXIMUM holds any text line one run writes: a prefix, a path, a position, and the
// longest message, whether the parser or shared/cli states it.
const LINE_SIZE_MAXIMUM = len(FAILURE_PREFIX) +
	filepath.PATH_SIZE_MAXIMUM +
	POSITION_SIZE_MAXIMUM +
	len(SEPARATOR) +
	ast.MESSAGE_SIZE_MAXIMUM +
	cli.FAILURE_SIZE_MAXIMUM +
	requirements.MESSAGE_SIZE_MAXIMUM +
	len("\n")

// WRITE_SIZE_MAXIMUM holds any one write: a text line, a flat JSON line, or the help.
const WRITE_SIZE_MAXIMUM = LINE_SIZE_MAXIMUM + JSON_STORAGE_SIZE + strings.TEXT_SIZE_MAXIMUM

// FILE_COUNT_MAXIMUM shares the traversal bound of shared/filepath, far past any one package.
const FILE_COUNT_MAXIMUM = filepath.FILESYSTEM_PATH_COUNT_MAXIMUM

// DIRECTORY_COUNT_MAXIMUM bounds the directories a walk holds pending at once.
const DIRECTORY_COUNT_MAXIMUM = filepath.FILESYSTEM_PATH_COUNT_MAXIMUM

// ENTRY_COUNT_MAXIMUM bounds the children one directory pass reports.
const ENTRY_COUNT_MAXIMUM = filepath.FILESYSTEM_PATH_COUNT_MAXIMUM

// SOURCE_STORAGE_SIZE holds one byte past the largest source, so a full read proves a file
// oversized without a second read.
const SOURCE_STORAGE_SIZE = token.SOURCE_SIZE_MAXIMUM + 1

// READ_TIMEOUT bounds one read of a local file.
const READ_TIMEOUT = 1 * time.MINUTE

// EXIT_SUCCESS reports a package without diagnostics.
const EXIT_SUCCESS Status_Code = 0

// EXIT_FAILURE reports any diagnostic or failed step.
const EXIT_FAILURE Status_Code = 1

// PHASE_READY is a stage Rearm has yet to submit.
const PHASE_READY Phase = 0

// PHASE_SUBMITTED is a stage whose operation is in flight on the loop.
const PHASE_SUBMITTED Phase = 1

// PHASE_RETIRED is a stage whose result the callback recorded and Rearm has yet to judge.
const PHASE_RETIRED Phase = 2

// PHASE_STOPPED is a finished run whose Status is final.
const PHASE_STOPPED Phase = 3

// STAGE_DIRECTORY_OPEN opens the next pending directory.
const STAGE_DIRECTORY_OPEN Stage = 0

// STAGE_DIRECTORY_READ reads one pass of the open directory's children.
const STAGE_DIRECTORY_READ Stage = 1

// STAGE_DIRECTORY_CLOSE closes the open directory.
const STAGE_DIRECTORY_CLOSE Stage = 2

// STAGE_OPEN opens the current file.
const STAGE_OPEN Stage = 3

// STAGE_READ reads the current file.
const STAGE_READ Stage = 4

// STAGE_CLOSE closes the current file and checks it.
const STAGE_CLOSE Stage = 5

// STAGE_REPORT writes the diagnostic of the current file.
const STAGE_REPORT Stage = 6

// STAGE_FAIL writes the diagnostic of a refused command line or a failed step.
const STAGE_FAIL Stage = 7

// STAGE_HELP writes the help -h asked for.
const STAGE_HELP Stage = 8

// STAGE_PASSED writes the pass line of a run without diagnostics.
const STAGE_PASSED Stage = 9

// STAGE_FIX_OPEN opens the current file to replace it with its fix.
const STAGE_FIX_OPEN Stage = 10

// STAGE_FIX_WRITE writes the rest of the fix.
const STAGE_FIX_WRITE Stage = 11

// STAGE_FIX_CLOSE closes the fixed file.
const STAGE_FIX_CLOSE Stage = 12

// STAGE_PACKAGE starts the contract check of the next package, without IO. A stage of its own
// keeps the package walk a loop through Rearm rather than a recursion.
const STAGE_PACKAGE Stage = 13

// STAGE_REQUIREMENTS resumes the open package's contract check, without IO.
const STAGE_REQUIREMENTS Stage = 14

// PASSED opens the pass line; the elapsed seconds follow it.
const PASSED = "✓ all checks passed\nelapsed_sec="

// MILLISECONDS_PER_SECOND splits elapsed milliseconds into seconds and their fraction.
const MILLISECONDS_PER_SECOND = 1000

// Main records the command line and readies the walk. It returns without waiting: the root
// repeats Rearm and pumps.
func Main(runner Runner_Handle, loop nbio.IO) {
	Runner_Handle_Invariants(runner, "main.runner")
	nbio.IO_Invariants(loop, "main.loop")
	runner.IO = loop
	runner.Start = time.Clock_Now_Monotonic(runner.Clock)
	runner.Rearm = runner_rearm
	runner.Work_Queued = runner_work_queued
	runner.Stopped = runner_stopped
	runner.File_Count = 0
	runner.File_Index = 0
	runner.Directory_Count = 0
	runner.Package_Count = 0
	runner.Diagnostic_Count = 0
	runner.Walk_Failed = false
	runner.Read_Failed = false
	runner.Memory.Current = runner.Memory.Current[:0]
	runner.Stage = STAGE_DIRECTORY_OPEN
	runner.Phase = PHASE_READY
	runner_parse(runner)
	// A program without secrets parses synchronously, thus the parse is already done.
	result, complete := cli.Parser_Done(runner.Memory.Parser)
	aver.Always(bool(complete), "A program without secrets parses synchronously.")
	failure := cli.Parse_Error_Bytes(result.Error)
	if cli.Parse_Error_Equals(result.Error, cli.HELP_REQUESTED) {
		help := cli.Output_Reference_Of(runner.Memory.Help)
		cli.Output_Reset(help)
		cli.Print_Help(help, *runner.Memory.Program)
		runner.Line = Line(cli.Output_Bytes(help))
		runner.Status = EXIT_SUCCESS
		runner.Stage = STAGE_HELP
		return
	}
	if len(failure) > 0 {
		runner.Line = append(Line(runner.Memory.Line[:0]), PROGRAM_PREFIX...)
		runner.Line = append(append(runner.Line, failure...), '\n')
		runner.Status = EXIT_FAILURE
		runner.Stage = STAGE_FAIL
		return
	}
	arguments := cli.Resolved_Options(result.Command.Arguments)
	flags := cli.Resolved_Options(result.Command.Flags)
	runner.Json = Json(cli.Option_Boolean(flags, FLAG_JSON))
	runner.Autofix = Autofix(cli.Option_Boolean(flags, FLAG_FIX))
	directory := cli.Option_String(arguments, ARGUMENT_DIRECTORY)
	// The argument is untrusted: a path past the host bound fits no path slot.
	if len(directory) > filepath.PATH_SIZE_MAXIMUM {
		runner.Step = STEP_WALK
		runner_fail(runner)
		return
	}
	slot := runner.Memory.Directories[0]
	runner.Memory.Directories[0] = append(slot[:0], directory...)
	runner.Directory_Count = 1
}

// Declares the command line into root storage, since a declaration literal would allocate, then
// parses it.
func runner_parse(runner Runner_Handle) {
	Runner_Handle_Invariants(runner, "runner_parse.runner")
	runner.Memory.Declared_Arguments[0] = cli.New_Option(
		cli.New_Option_Input{
			Label:       ARGUMENT_DIRECTORY,
			Description: "Go package to check, at any depth",
		},
	)
	runner.Memory.Declared_Flags[0] = cli.New_Option(
		cli.New_Option_Input{
			Label:       FLAG_JSON,
			Description: "write each diagnostic as one line of flat JSON",
			Type:        cli.OPTION_TYPE_BOOLEAN,
			Is_Flag:     true,
		},
	)
	runner.Memory.Declared_Flags[1] = cli.New_Option(
		cli.New_Option_Input{
			Label:       FLAG_FIX,
			Description: "replace each auto-fixable file with its fix",
			Type:        cli.OPTION_TYPE_BOOLEAN,
			Is_Flag:     true,
		},
	)
	*runner.Memory.Program = cli.New(
		cli.New_Input{
			Mode:        cli.PROGRAM_MODE_SINGLE,
			Label:       PROGRAM_LABEL,
			Description: "report each Go file that breaks a rule",
			Arguments:   runner.Memory.Declared_Arguments,
			Flags:       runner.Memory.Declared_Flags,
		},
	)
	cli.Program_Parse(
		*runner.Memory.Program,
		runner.Memory.Parser,
		cli.Program_Parse_Input{
			Arguments:         cli.Process_Arguments(runner.Arguments),
			Command_Arguments: runner.Memory.Command_Arguments,
			Command_Flags:     runner.Memory.Command_Flags,
			Filled:            runner.Memory.Filled,
			Positionals:       runner.Memory.Positionals,
			Slice_Named:       runner.Memory.Slice_Named,
			Failure_Storage:   runner.Memory.Failure,
		},
	)
}

// Check records in runner.Diagnostic the rule that Memory.Source[:Source_Count] breaks, or
// TIER_NONE.
func Check(runner Runner_Handle) {
	Runner_Handle_Invariants(runner, "check.runner")
	file := runner.Memory.Source[:runner.Source_Count]
	runner.Diagnostic = Diagnostic{Tier: TIER_NONE, Rule: RULE_NONE}
	if len(file) > token.SOURCE_SIZE_MAXIMUM {
		runner.Diagnostic = Diagnostic{Tier: TIER_TWO, Rule: RULE_OVERSIZED}
		return
	}
	source := token.Source(file)
	parsed := ast.Parse(runner.Memory.Parse_State, source)
	if !bool(parsed.OK) {
		runner_refused(runner)
		return
	}
	result := format.Format(
		runner.Memory.Formatter,
		runner.Memory.Printer,
		runner.Memory.Form,
		runner.Memory.Parse_State,
		source,
	)
	aver.Always(bool(result.OK), "The form of an accepted source fits FORM_SIZE_MAXIMUM.")
	form := runner.Memory.Form[:result.Count]
	if string(form) == string(source) {
		return
	}
	runner.Diagnostic = Diagnostic{Tier: TIER_ONE, Rule: RULE_UNFORMATTED, Fix: Fix(form)}
}

// Records the parser's own refusal: its code, which names the fix, and where the refused token
// stands. A source with more lines than the index holds keeps position zero.
func runner_refused(runner Runner_Handle) {
	Runner_Handle_Invariants(runner, "runner_refused.runner")
	source := token.Source(runner.Memory.Source[:runner.Source_Count])
	state := runner.Memory.Parse_State
	runner.Diagnostic = Diagnostic{
		Tier:    TIER_TWO,
		Rule:    RULE_REFUSED,
		Failure: ast.Failure(state),
	}
	if !bool(token.Index_Lines(runner.Memory.Lines, source)) {
		return
	}
	refused := ast.Token_At(state, ast.Failure_Token(state))
	runner.Diagnostic.Position = token.Position_Of(runner.Memory.Lines, refused.Offset)
}

// Tier is whether a diagnostic is auto-fixable.
type Tier int

// Tier_Invariants admits no diagnostic and both tiers.
func Tier_Invariants(value Tier, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Int(
			int(value),
			int(TIER_NONE),
			int(TIER_ONE),
			int(TIER_TWO),
		).
		Ensure()
}

// Rule is the rule a file breaks.
type Rule int

// Rule_Invariants admits no rule and each rule.
func Rule_Invariants(value Rule, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), int(RULE_NONE), int(RULE_REQUIREMENTS)).
		Ensure()
}

// Fix is the whole file as it should stand.
type Fix []byte

// Fix_Invariants bounds a fix by the widest form one print writes.
func Fix_Invariants(value Fix, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value),
			bytes.SLICE_SIZE_MINIMUM,
			printer.FORM_SIZE_MAXIMUM,
		).
		Ensure()
}

// Step is the step a failed run names.
type Step int

// Step_Invariants admits each step.
func Step_Invariants(value Step, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), int(STEP_WALK), int(STEP_FIX)).
		Ensure()
}

// Index is a count, or a slot of a bounded table.
type Index int

// Index_Invariants bounds an index by the widest bounded input, a source.
func Index_Invariants(value Index, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), slices.COUNT_MINIMUM, SOURCE_STORAGE_SIZE).
		Ensure()
}

// Flag is one yes or no a run records.
type Flag bool

// Flag_Invariants proves both answers occur.
func Flag_Invariants(value Flag, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The harness flag is set.").
		Ensure()
}

// Resume is where a run continues after it writes a report.
type Resume int

// Resume_Invariants admits each continuation.
func Resume_Invariants(value Resume, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Int(
			int(value),
			int(RESUME_FILE),
			int(RESUME_PACKAGE),
			int(RESUME_REQUIREMENTS),
		).
		Ensure()
}

// Reading is what the open file read serves.
type Reading int

// Reading_Invariants admits each purpose.
func Reading_Invariants(value Reading, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Int(
			int(value),
			int(READING_SOURCE),
			int(READING_MARKDOWN),
			int(READING_TEST),
		).
		Ensure()
}

// Package is one directory holding a Go file, as the walk found it.
type Package struct {
	// Path is the directory.
	Path filepath.Slice
	// Markdown records an exact-cased REQUIREMENTS.md.
	Markdown Flag
	// Test records an exact-cased requirements_test.go.
	Test Flag
	// Impure records a package main file, which exempts the directory from the contract.
	Impure Flag
}

// Package_Invariants composes one package.
func Package_Invariants(value Package, namespace aver.Namespace) {
	filepath.Slice_Invariants(value.Path, namespace)
	Flag_Invariants(value.Markdown, namespace)
	Flag_Invariants(value.Test, namespace)
	Flag_Invariants(value.Impure, namespace)
}

// Packages holds every package the walk found.
type Packages []Package

// Packages_Invariants bounds the package table.
func Packages_Invariants(value Packages, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value),
			slices.COUNT_MINIMUM,
			PACKAGE_COUNT_MAXIMUM,
		).
		Ensure()
}

// Package_Indices names each collected Go file's package, slot for slot with Memory.Paths.
type Package_Indices []Index

// Package_Indices_Invariants bounds the table by the file table.
func Package_Indices_Invariants(value Package_Indices, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, FILE_COUNT_MAXIMUM).
		Ensure()
}

// Autofix records whether each fix replaces its file.
type Autofix bool

// Autofix_Invariants proves both modes occur.
func Autofix_Invariants(value Autofix, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Fixes replace their files.").
		Ensure()
}

// Fix_Failed records a failed fix write, which reports only once the file is closed.
type Fix_Failed bool

// Fix_Failed_Invariants proves both write outcomes occur.
func Fix_Failed_Invariants(value Fix_Failed, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "A fix write failed.").
		Ensure()
}

// Fix_Count counts the bytes of the fix written so far.
type Fix_Count int

// Fix_Count_Invariants bounds written bytes by the widest fix.
func Fix_Count_Invariants(value Fix_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value),
			bytes.SLICE_SIZE_MINIMUM,
			printer.FORM_SIZE_MAXIMUM,
		).
		Ensure()
}

// Json records whether diagnostics are written as flat JSON.
type Json bool

// Json_Invariants proves both output forms occur.
func Json_Invariants(value Json, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Diagnostics are written as flat JSON.").
		Ensure()
}

// Report_Text is one text leaf of a flat JSON diagnostic. It views stored bytes or a constant.
type Report_Text string

// Report_Text_Invariants bounds a leaf by the widest flat JSON object.
func Report_Text_Invariants(value Report_Text, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value),
			bytes.TEXT_SIZE_MINIMUM,
			flatjson.OUTPUT_SIZE_MAXIMUM,
		).
		Ensure()
}

// Report is one diagnostic as flat JSON states it. Its order is the key order.
type Report struct {
	// Path names the file.
	Path Report_Text `json:"path"`
	// Line is where a refused token stands; 0 without a position.
	Line token.Line `json:"line"`
	// Column is where a refused token stands; 0 without a position.
	Column token.Column `json:"column"`
	// Tier is whether the diagnostic is auto-fixable.
	Tier Tier `json:"tier"`
	// Rule names the rule the file breaks.
	Rule Report_Text `json:"rule"`
	// Message says what to do about it.
	Message Report_Text `json:"message"`
}

// Report_Invariants composes one flat JSON diagnostic.
func Report_Invariants(value Report, namespace aver.Namespace) {
	Report_Text_Invariants(value.Path, namespace)
	token.Line_Invariants(value.Line, namespace)
	token.Column_Invariants(value.Column, namespace)
	Tier_Invariants(value.Tier, namespace)
	Report_Text_Invariants(value.Rule, namespace)
	Report_Text_Invariants(value.Message, namespace)
}

// Passed records that a run found no diagnostic.
type Passed bool

// Passed_Invariants proves both outcomes occur.
func Passed_Invariants(value Passed, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "A run passed.").
		Ensure()
}

// Summary is a passed run as flat JSON states it. Its order is the key order.
type Summary struct {
	// Passed is true; a run with a diagnostic writes no summary.
	Passed Passed `json:"passed"`
	// Elapsed_Seconds is text, since shared/encoding/flatjson holds no float.
	Elapsed_Seconds Report_Text `json:"elapsed_sec"`
}

// Summary_Invariants composes one flat JSON summary.
func Summary_Invariants(value Summary, namespace aver.Namespace) {
	Passed_Invariants(value.Passed, namespace)
	Report_Text_Invariants(value.Elapsed_Seconds, namespace)
}

// Json_Storage holds one flat JSON line and its newline.
type Json_Storage []byte

// Json_Storage_Invariants bounds flat JSON storage.
func Json_Storage_Invariants(value Json_Storage, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value),
			bytes.SLICE_SIZE_MINIMUM,
			JSON_STORAGE_SIZE,
		).
		Ensure()
}

// Line is the part of Line_Storage one write sends.
type Line []byte

// Line_Invariants keeps one write inside the storage it views.
func Line_Invariants(value Line, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value),
			bytes.SLICE_SIZE_MINIMUM,
			WRITE_SIZE_MAXIMUM,
		).
		Ensure()
}

// Line_Storage holds the line being written.
type Line_Storage []byte

// Line_Storage_Invariants bounds line storage.
func Line_Storage_Invariants(value Line_Storage, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value),
			bytes.SLICE_SIZE_MINIMUM,
			LINE_SIZE_MAXIMUM,
		).
		Ensure()
}

// Directory_Paths holds the directories the walk has yet to open, one slot each.
type Directory_Paths []filepath.Slice

// Directory_Paths_Invariants bounds pending directories.
func Directory_Paths_Invariants(value Directory_Paths, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value),
			slices.COUNT_MINIMUM,
			DIRECTORY_COUNT_MAXIMUM,
		).
		Ensure()
}

// File_Bytes is a file as read, which may run one byte past token.SOURCE_SIZE_MAXIMUM: that byte
// is how a read proves the file oversized, thus it is no token.Source until Check says so.
type File_Bytes []byte

// File_Bytes_Invariants bounds a read file by its storage.
func File_Bytes_Invariants(value File_Bytes, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value),
			bytes.SLICE_SIZE_MINIMUM,
			SOURCE_STORAGE_SIZE,
		).
		Ensure()
}

// File_Count counts the Go file paths the walk collected.
type File_Count int

// File_Count_Invariants bounds collected paths.
func File_Count_Invariants(value File_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), slices.COUNT_MINIMUM, FILE_COUNT_MAXIMUM).
		Ensure()
}

// File_Index is the file the run stands on; File_Count means past the last.
type File_Index int

// File_Index_Invariants bounds the current file.
func File_Index_Invariants(value File_Index, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), slices.COUNT_MINIMUM, FILE_COUNT_MAXIMUM).
		Ensure()
}

// Directory_Count counts the directories the walk has yet to open.
type Directory_Count int

// Directory_Count_Invariants bounds pending directories.
func Directory_Count_Invariants(value Directory_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value),
			slices.COUNT_MINIMUM,
			DIRECTORY_COUNT_MAXIMUM,
		).
		Ensure()
}

// Diagnostic_Count counts the diagnostics a run reported, at most one per file.
type Diagnostic_Count int

// Diagnostic_Count_Invariants bounds reported diagnostics.
func Diagnostic_Count_Invariants(value Diagnostic_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), slices.COUNT_MINIMUM, FILE_COUNT_MAXIMUM).
		Ensure()
}

// Source_Count counts the bytes of the current file read so far.
type Source_Count int

// Source_Count_Invariants bounds read bytes by the source storage.
func Source_Count_Invariants(value Source_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value),
			bytes.SLICE_SIZE_MINIMUM,
			SOURCE_STORAGE_SIZE,
		).
		Ensure()
}

// Boolean is one yes or no the root reads.
type Boolean bool

// Boolean_Invariants proves both answers occur.
func Boolean_Invariants(value Boolean, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The harness answer is yes.").
		Ensure()
}

// Walk_Failed records a failed directory pass, which reports only once the directory is closed.
type Walk_Failed bool

// Walk_Failed_Invariants proves both walk outcomes occur.
func Walk_Failed_Invariants(value Walk_Failed, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "A directory pass failed.").
		Ensure()
}

// Read_Failed records a failed file read, which reports only once the file is closed.
type Read_Failed bool

// Read_Failed_Invariants proves both read outcomes occur.
func Read_Failed_Invariants(value Read_Failed, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "A file read failed.").
		Ensure()
}

// Status_Code is the exit code of the run.
type Status_Code int

// Status_Code_Invariants admits success and failure.
func Status_Code_Invariants(value Status_Code, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Int(int(value), int(EXIT_SUCCESS), int(EXIT_FAILURE)).
		Ensure()
}

// Phase is where the current stage stands; callbacks only move it forward, Rearm acts on it.
type Phase int

// Phase_Invariants admits each phase.
func Phase_Invariants(value Phase, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_4_Int(
			int(value),
			int(PHASE_READY),
			int(PHASE_SUBMITTED),
			int(PHASE_RETIRED),
			int(PHASE_STOPPED),
		).
		Ensure()
}

// Stage is the step of the run.
type Stage int

// Stage_Invariants admits each stage.
func Stage_Invariants(value Stage, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value),
			int(STAGE_DIRECTORY_OPEN),
			int(STAGE_REQUIREMENTS),
		).
		Ensure()
}

// Diagnostic is the rule one file breaks.
type Diagnostic struct {
	// Tier is whether the diagnostic is auto-fixable; TIER_NONE when the file breaks nothing.
	Tier Tier
	// Rule is the rule the file breaks.
	Rule Rule
	// Fix is the whole file as it should stand; empty unless TIER_ONE. It views Memory.Form,
	// thus the next check overwrites it.
	Fix Fix
	// Failure is the parser's refusal; FAILURE_NONE unless RULE_REFUSED.
	Failure ast.Failure_Code
	// Position is where the refused token stands; zero unless RULE_REFUSED.
	Position token.Position_Result
}

// Diagnostic_Invariants composes one diagnostic.
func Diagnostic_Invariants(value Diagnostic, namespace aver.Namespace) {
	Tier_Invariants(value.Tier, namespace)
	Rule_Invariants(value.Rule, namespace)
	Fix_Invariants(value.Fix, namespace)
	ast.Failure_Code_Invariants(value.Failure, namespace)
	token.Position_Result_Invariants(value.Position, namespace)
}

// Memory is every store one run uses, allocated once by the root before the run starts.
type Memory struct {
	// Formatter holds the import sort cursors.
	Formatter format.Formatter_Handle
	// Printer holds the printer stacks.
	Printer printer.Printer_Handle
	// Parse_State holds the parsed file.
	Parse_State ast.Parse_State_Handle
	// Form holds the canonical form.
	Form printer.Form
	// Lines holds where each line of a refused source opens.
	Lines token.Line_Index_Handle
	// Paths holds every Go file path the walk collects, one slot each.
	Paths filepath.Path_Storage
	// Directories holds the directories the walk has yet to open.
	Directories Directory_Paths
	// Current holds the path of the directory the walk has open; a pending slot would be
	// overwritten by the children the pass pushes.
	Current filepath.Slice
	// Entries receives one directory pass.
	Entries filepath.Directory_Entries
	// Directory_Buffer receives backend records for one pass.
	Directory_Buffer filepath.Directory_Buffer
	// Source holds the file being read.
	Source File_Bytes
	// Line holds the text line being written.
	Line Line_Storage
	// Json holds the flat JSON line being written.
	Json Json_Storage
	// Declared_Arguments holds the one positional declaration.
	Declared_Arguments cli.Arguments
	// Declared_Flags holds the -json declaration.
	Declared_Flags cli.Flags
	// Command_Arguments receives the parsed positional.
	Command_Arguments cli.Command_Argument_Storage
	// Command_Flags receives the parsed flag.
	Command_Flags cli.Command_Flag_Storage
	// Filled marks each option the command line set.
	Filled cli.Filled_Options
	// Positionals receives the command line's positional tokens.
	Positionals cli.Positionals
	// Slice_Named receives named variadic tokens; the program declares none, but the parser
	// sizes its token pass by it.
	Slice_Named cli.Named_Variadic_Tokens
	// Failure receives the parser's rendered error.
	Failure cli.Parse_Failure_Storage
	// Help holds the rendered help.
	Help cli.Output_Handle
	// Packages holds every package the walk found.
	Packages Packages
	// File_Packages names each Go file's package, slot for slot with Paths.
	File_Packages Package_Indices
	// Read_Path holds the path of the file being read, fixed, or reported.
	Read_Path filepath.Slice
	// Program holds the command line declaration.
	Program Program_Handle
	// Requirements checks each package's contract.
	Requirements requirements.Checker_Handle
	// Parser parses the command line.
	Parser cli.Parser_Handle
}

// Memory_Invariants composes run storage.
func Memory_Invariants(value Memory, namespace aver.Namespace) {
	format.Formatter_Handle_Invariants(value.Formatter, namespace)
	printer.Printer_Handle_Invariants(value.Printer, namespace)
	ast.Parse_State_Handle_Invariants(value.Parse_State, namespace)
	printer.Form_Invariants(value.Form, namespace)
	token.Line_Index_Handle_Invariants(value.Lines, namespace)
	filepath.Path_Storage_Invariants(value.Paths, namespace)
	Directory_Paths_Invariants(value.Directories, namespace)
	filepath.Slice_Invariants(value.Current, namespace)
	filepath.Directory_Entries_Invariants(value.Entries, namespace)
	filepath.Directory_Buffer_Invariants(value.Directory_Buffer, namespace)
	File_Bytes_Invariants(value.Source, namespace)
	Line_Storage_Invariants(value.Line, namespace)
	Json_Storage_Invariants(value.Json, namespace)
	cli.Arguments_Invariants(value.Declared_Arguments, namespace)
	cli.Flags_Invariants(value.Declared_Flags, namespace)
	cli.Command_Argument_Storage_Invariants(value.Command_Arguments, namespace)
	cli.Command_Flag_Storage_Invariants(value.Command_Flags, namespace)
	cli.Filled_Options_Invariants(value.Filled, namespace)
	cli.Positionals_Invariants(value.Positionals, namespace)
	cli.Named_Variadic_Tokens_Invariants(value.Slice_Named, namespace)
	cli.Parse_Failure_Storage_Invariants(value.Failure, namespace)
	cli.Output_Handle_Invariants(value.Help, namespace)
	Packages_Invariants(value.Packages, namespace)
	Package_Indices_Invariants(value.File_Packages, namespace)
	filepath.Slice_Invariants(value.Read_Path, namespace)
	Program_Handle_Invariants(value.Program, namespace)
	cli.Parser_Handle_Invariants(value.Parser, namespace)
	requirements.Checker_Handle_Invariants(value.Requirements, namespace)
}

// Memory_Handle is caller-owned run storage.
type Memory_Handle *Memory

// Memory_Handle_Invariants composes present run storage.
func Memory_Handle_Invariants(value Memory_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Memory_Invariants(*value, namespace)
}

// Program_Handle is caller-owned command line declaration.
type Program_Handle *cli.Program

// Program_Handle_Invariants composes a present declaration.
func Program_Handle_Invariants(value Program_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	cli.Program_Invariants(*value, namespace)
}

// Rearm_Procedure turns what a callback recorded into the next submission and reports whether
// it did anything. A static function holds no state, so binding it allocates nothing.
type Rearm_Procedure func(runner Runner_Handle) (progressed Boolean)

// Work_Queued_Procedure reports whether a callback recorded work only Rearm can act on, or the run
// stopped: either way a pump pass is over.
type Work_Queued_Procedure func(runner Runner_Handle) (queued Boolean)

// Stopped_Procedure reports whether the run is over and Status is final.
type Stopped_Procedure func(runner Runner_Handle) (stopped Boolean)

// Runner_Fields is one whole run's caller-owned state. Callbacks record into it; the root calls
// Rearm to turn what they recorded into the next submission, then pumps.
type Runner_Fields struct {
	// Completion stays first so the static callback recovers the runner from it alone.
	Completion nbio.Completion
	// IO is the injected loop surface every operation goes through.
	IO nbio.IO
	// Output receives one line per diagnostic.
	Output nbio.Stream
	// Error_Output receives the diagnostic of a malformed command line or failed step.
	Error_Output nbio.Stream
	// Clock times the run.
	Clock time.Clock
	// Start is when Main ran.
	Start time.Monotonic_Moment
	// Arguments is the command line the root read.
	Arguments os.Arguments
	// Rearm is bound by Main.
	Rearm Rearm_Procedure
	// Work_Queued is bound by Main.
	Work_Queued Work_Queued_Procedure
	// Stopped is bound by Main.
	Stopped Stopped_Procedure
	// Memory is every store the run uses; the root allocates it once. It stays behind a handle,
	// because a run's fields past the stack bound of an interface conversion would allocate on
	// every assertion.
	Memory Memory_Handle
	// Json selects flat JSON diagnostics.
	Json Json
	// Package_Count counts the packages in Memory.Packages.
	Package_Count Index
	// Package_Index is the package whose contract the run checks.
	Package_Index Index
	// Pending_Go records a Go file in the open directory.
	Pending_Go Flag
	// Pending_Markdown records a REQUIREMENTS.md in the open directory.
	Pending_Markdown Flag
	// Pending_Test records a requirements_test.go in the open directory.
	Pending_Test Flag
	// Reading is what the open file read serves.
	Reading Reading
	// Resume is where the run continues after its report.
	Resume Resume
	// Read_File is the contract file of the open package that Read_Path names.
	Read_File requirements.File
	// Autofix selects replacing each file with its fix.
	Autofix Autofix
	// Fix_Count counts the fix bytes written to the current file.
	Fix_Count Fix_Count
	// Fix_Failed records a failed fix write.
	Fix_Failed Fix_Failed
	// Summary is the flat JSON form of a passed run.
	Summary Summary
	// Report is the flat JSON form of the last diagnostic.
	Report Report
	// Directory_Count counts pending directories in Memory.Directories.
	Directory_Count Directory_Count
	// Directory is the open descriptor of Memory.Current.
	Directory nbio.File
	// Walk_Failed records a failed directory pass.
	Walk_Failed Walk_Failed
	// File_Count counts the paths in Memory.Paths.
	File_Count File_Count
	// File_Index is the file the run stands on.
	File_Index File_Index
	// File is the open descriptor of the current file.
	File nbio.File
	// Source_Count counts the bytes of the current file in Memory.Source.
	Source_Count Source_Count
	// Read_Failed records a failed read.
	Read_Failed Read_Failed
	// Diagnostic is what the last check found.
	Diagnostic Diagnostic
	// Diagnostic_Count counts the diagnostics the run reported.
	Diagnostic_Count Diagnostic_Count
	// Step is the step a failed run names.
	Step Step
	// Line views the part of Memory.Line to write.
	Line Line
	// Stage is the step of the run.
	Stage Stage
	// Phase is where the stage stands.
	Phase Phase
	// Status is the exit code, final once Phase is PHASE_STOPPED.
	Status Status_Code
}

// Runner_Fields_Invariants composes every run field.
func Runner_Fields_Invariants(value Runner_Fields, namespace aver.Namespace) {
	nbio.IO_Invariants(value.IO, namespace)
	time.Clock_Invariants(value.Clock, namespace)
	time.Monotonic_Moment_Invariants(value.Start, namespace)
	os.Arguments_Invariants(value.Arguments, namespace)
	Memory_Handle_Invariants(value.Memory, namespace)
	Json_Invariants(value.Json, namespace)
	Index_Invariants(value.Package_Count, namespace)
	Index_Invariants(value.Package_Index, namespace)
	Flag_Invariants(value.Pending_Go, namespace)
	Flag_Invariants(value.Pending_Markdown, namespace)
	Flag_Invariants(value.Pending_Test, namespace)
	Reading_Invariants(value.Reading, namespace)
	Resume_Invariants(value.Resume, namespace)
	requirements.File_Invariants(value.Read_File, namespace)
	Autofix_Invariants(value.Autofix, namespace)
	Fix_Count_Invariants(value.Fix_Count, namespace)
	Fix_Failed_Invariants(value.Fix_Failed, namespace)
	Summary_Invariants(value.Summary, namespace)
	Report_Invariants(value.Report, namespace)
	Directory_Count_Invariants(value.Directory_Count, namespace)
	Walk_Failed_Invariants(value.Walk_Failed, namespace)
	File_Count_Invariants(value.File_Count, namespace)
	File_Index_Invariants(value.File_Index, namespace)
	Source_Count_Invariants(value.Source_Count, namespace)
	Read_Failed_Invariants(value.Read_Failed, namespace)
	Diagnostic_Invariants(value.Diagnostic, namespace)
	Diagnostic_Count_Invariants(value.Diagnostic_Count, namespace)
	Step_Invariants(value.Step, namespace)
	Line_Invariants(value.Line, namespace)
	Stage_Invariants(value.Stage, namespace)
	Phase_Invariants(value.Phase, namespace)
	Status_Code_Invariants(value.Status, namespace)
}

// Runner_Fields_Stored keeps one run step from proving every store length it never touched.
type Runner_Fields_Stored interface{}

// Runner_Fields_Stored_Invariants fixes run state representation.
func Runner_Fields_Stored_Invariants(value Runner_Fields_Stored, _ aver.Namespace) {
	_, valid := value.(Runner_Fields)
	aver.Always(valid == (value != nil), "Run fields have expected storage type.")
}

// Runner_Envelope gives run fields one embedded owner.
type Runner_Envelope struct {
	Runner_Fields
}

// Runner_Envelope_Invariants composes run fields.
func Runner_Envelope_Invariants(value Runner_Envelope, namespace aver.Namespace) {
	Runner_Fields_Invariants(value.Runner_Fields, namespace)
}

// Runner is one whole run.
type Runner Runner_Envelope

// Runner_Invariants proves run field representation.
func Runner_Invariants(value Runner, namespace aver.Namespace) {
	Runner_Fields_Stored_Invariants(Runner_Fields_Stored(value.Runner_Fields), namespace)
}

// Runner_Handle is caller-owned run state.
type Runner_Handle *Runner

// Runner_Handle_Invariants composes present run state.
func Runner_Handle_Invariants(value Runner_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Runner_Invariants(*value, namespace)
}

func runner_rearm(runner Runner_Handle) (progressed Boolean) {
	defer func() { Boolean_Invariants(progressed, "runner_rearm.progressed") }()
	Runner_Handle_Invariants(runner, "runner_rearm.runner")
	switch runner.Phase {
	case PHASE_READY:
		runner_submit(runner)
		return true
	case PHASE_RETIRED:
		runner.Phase = PHASE_READY
		runner_judge(runner)
		return true
	}
	return false
}

func runner_work_queued(runner Runner_Handle) (queued Boolean) {
	defer func() { Boolean_Invariants(queued, "runner_work_queued.queued") }()
	Runner_Handle_Invariants(runner, "runner_work_queued.runner")
	switch runner.Phase {
	case PHASE_READY, PHASE_RETIRED, PHASE_STOPPED:
		return true
	}
	return false
}

func runner_stopped(runner Runner_Handle) (stopped Boolean) {
	defer func() { Boolean_Invariants(stopped, "runner_stopped.stopped") }()
	Runner_Handle_Invariants(runner, "runner_stopped.runner")
	return runner.Phase == PHASE_STOPPED
}

// A stream can retire inside the submit, thus the phase moves first or it would erase that
// retirement.
func runner_submit(runner Runner_Handle) {
	Runner_Handle_Invariants(runner, "runner_submit.runner")
	switch runner.Stage {
	case STAGE_PACKAGE:
		runner_package_next(runner)
		return
	case STAGE_REQUIREMENTS:
		runner_requirements_next(runner)
		return
	}
	runner.Phase = PHASE_SUBMITTED
	completion := &runner.Completion
	storage := runner.IO.Storage
	switch runner.Stage {
	case STAGE_DIRECTORY_OPEN:
		runner.Directory_Count--
		next := runner.Memory.Directories[runner.Directory_Count]
		current := append(runner.Memory.Current[:0], next...)
		runner.Memory.Current = current
		// A view, since a conversion would allocate; the slot holds still until the retire.
		path := unsafe.String(unsafe.SliceData(current), len(current))
		nbio.Storage_Open_At(
			storage,
			completion,
			nbio.DIRECTORY_CURRENT,
			path,
			nbio.Open_At_Options{Access: nbio.OPEN_READ_ONLY},
			runner_retired,
		)
	case STAGE_DIRECTORY_READ:
		nbio.Storage_Get_Directory_Entries(
			storage,
			completion,
			runner.Directory,
			runner.Memory.Directory_Buffer,
			runner.Memory.Entries,
			runner_retired,
		)
	case STAGE_DIRECTORY_CLOSE:
		nbio.IO_Close(runner.IO, completion, runner.Directory, runner_retired)
	case STAGE_OPEN:
		stored := runner.Memory.Read_Path
		path := unsafe.String(unsafe.SliceData(stored), len(stored))
		nbio.Storage_Open_At(
			storage,
			completion,
			nbio.DIRECTORY_CURRENT,
			path,
			nbio.Open_At_Options{Access: nbio.OPEN_READ_ONLY},
			runner_retired,
		)
	case STAGE_READ:
		runner_submit_read(runner)
	case STAGE_CLOSE:
		nbio.IO_Close(runner.IO, completion, runner.File, runner_retired)
	case STAGE_REPORT:
		nbio.Write(runner.Output, completion, runner.Line, runner_retired)
	case STAGE_FAIL:
		nbio.Write(runner.Error_Output, completion, runner.Line, runner_retired)
	case STAGE_HELP, STAGE_PASSED:
		nbio.Write(runner.Output, completion, runner.Line, runner_retired)
	case STAGE_FIX_OPEN, STAGE_FIX_WRITE, STAGE_FIX_CLOSE:
		runner_submit_fix(runner)
	}
}

// Reads the rest of the open file into the storage of what the read serves.
func runner_submit_read(runner Runner_Handle) {
	Runner_Handle_Invariants(runner, "runner_submit_read.runner")
	buffer := runner.Memory.Source[runner.Source_Count:]
	offset := int64(runner.Source_Count)
	if runner.Reading == READING_MARKDOWN {
		checker := runner.Memory.Requirements
		buffer = File_Bytes(checker.Markdown[checker.Markdown_Count:])
		offset = int64(checker.Markdown_Count)
	}
	nbio.Storage_Read(
		runner.IO.Storage,
		&runner.Completion,
		runner.File,
		buffer,
		offset,
		READ_TIMEOUT,
		runner_retired,
	)
}

// Truncate then write, since the storage surface names no rename to swap a whole file in.
func runner_submit_fix(runner Runner_Handle) {
	Runner_Handle_Invariants(runner, "runner_submit_fix.runner")
	completion := &runner.Completion
	storage := runner.IO.Storage
	switch runner.Stage {
	case STAGE_FIX_OPEN:
		stored := runner.Memory.Read_Path
		path := unsafe.String(unsafe.SliceData(stored), len(stored))
		options := nbio.Open_At_Options{Access: nbio.OPEN_WRITE_ONLY, Truncate: true}
		nbio.Storage_Open_At(
			storage,
			completion,
			nbio.DIRECTORY_CURRENT,
			path,
			options,
			runner_retired,
		)
	case STAGE_FIX_WRITE:
		nbio.Storage_Write(
			storage,
			completion,
			runner.File,
			runner.Diagnostic.Fix[runner.Fix_Count:],
			int64(runner.Fix_Count),
			WRITE_TIMEOUT,
			runner_retired,
		)
	case STAGE_FIX_CLOSE:
		nbio.IO_Close(runner.IO, completion, runner.File, runner_retired)
	}
}

func runner_judge(runner Runner_Handle) {
	Runner_Handle_Invariants(runner, "runner_judge.runner")
	completion := &runner.Completion
	switch runner.Stage {
	case STAGE_DIRECTORY_OPEN:
		runner.Step = STEP_WALK
		if completion.Error != nil {
			runner_fail(runner)
			return
		}
		runner.Directory = nbio.File(completion.Data)
		runner.Walk_Failed = false
		runner.Pending_Go = false
		runner.Pending_Markdown = false
		runner.Pending_Test = false
		runner.Stage = STAGE_DIRECTORY_READ
	case STAGE_DIRECTORY_READ:
		runner_judge_directory_read(runner)
	case STAGE_DIRECTORY_CLOSE:
		runner_judge_directory_close(runner)
	case STAGE_OPEN:
		runner.Step = STEP_OPEN
		if completion.Error != nil {
			runner_fail(runner)
			return
		}
		runner.File = nbio.File(completion.Data)
		runner.Source_Count = 0
		runner.Memory.Requirements.Markdown_Count = 0
		runner.Read_Failed = false
		runner.Stage = STAGE_READ
	case STAGE_READ:
		runner_judge_read(runner)
	case STAGE_CLOSE:
		runner_judge_close(runner)
	case STAGE_REPORT:
		if runner.Resume == RESUME_REQUIREMENTS {
			runner_requirements_next(runner)
			return
		}
		runner.File_Index++
		runner_file_next(runner)
	case STAGE_FAIL, STAGE_HELP, STAGE_PASSED:
		runner.Phase = PHASE_STOPPED
	case STAGE_FIX_OPEN, STAGE_FIX_WRITE, STAGE_FIX_CLOSE:
		runner_judge_fix(runner)
	}
}

// A descriptor outlives a failed write, thus the failure waits for the close. A write that moves
// no byte fails too, or a full disk would spin the run.
func runner_judge_fix(runner Runner_Handle) {
	Runner_Handle_Invariants(runner, "runner_judge_fix.runner")
	runner.Step = STEP_FIX
	completion := &runner.Completion
	switch runner.Stage {
	case STAGE_FIX_OPEN:
		if completion.Error != nil {
			runner_fail(runner)
			return
		}
		runner.File = nbio.File(completion.Data)
		runner.Stage = STAGE_FIX_WRITE
		if len(runner.Diagnostic.Fix) == 0 {
			runner.Stage = STAGE_FIX_CLOSE
		}
	case STAGE_FIX_WRITE:
		if completion.Error != nil {
			runner.Fix_Failed = true
			runner.Stage = STAGE_FIX_CLOSE
			return
		}
		if completion.Data == 0 {
			runner.Fix_Failed = true
			runner.Stage = STAGE_FIX_CLOSE
			return
		}
		runner.Fix_Count += Fix_Count(completion.Data)
		if int(runner.Fix_Count) == len(runner.Diagnostic.Fix) {
			runner.Stage = STAGE_FIX_CLOSE
		}
	case STAGE_FIX_CLOSE:
		if completion.Error != nil {
			runner_fail(runner)
			return
		}
		if runner.Fix_Failed {
			runner_fail(runner)
			return
		}
		runner.File_Index++
		runner_file_next(runner)
	}
}

// A descriptor outlives a failed pass, thus the failure waits for the close.
func runner_judge_directory_read(runner Runner_Handle) {
	Runner_Handle_Invariants(runner, "runner_judge_directory_read.runner")
	if runner.Completion.Error != nil {
		runner.Walk_Failed = true
		runner.Stage = STAGE_DIRECTORY_CLOSE
		return
	}
	if runner.Completion.Data == 0 {
		runner.Stage = STAGE_DIRECTORY_CLOSE
		return
	}
	for index := range runner.Completion.Data {
		runner_collect(runner, runner.Memory.Entries[index])
		if runner.Walk_Failed {
			runner.Stage = STAGE_DIRECTORY_CLOSE
			return
		}
	}
}

// Records one child: a directory waits its turn, a regular Go file joins the check. A symbolic
// link is neither, so the walk never leaves the tree it was given.
func runner_collect(runner Runner_Handle, entry nbio.Directory_Entry) {
	Runner_Handle_Invariants(runner, "runner_collect.runner")
	current := runner.Memory.Current
	if len(current)+len("/")+len(entry.Name) > filepath.PATH_SIZE_MAXIMUM {
		runner.Walk_Failed = true
		return
	}
	if entry.Is_Directory {
		if int(runner.Directory_Count) == len(runner.Memory.Directories) {
			runner.Walk_Failed = true
			return
		}
		child := runner.Memory.Directories[runner.Directory_Count][:0]
		child = append(append(child, current...), PATH_SEPARATOR)
		runner.Memory.Directories[runner.Directory_Count] = append(child, entry.Name...)
		runner.Directory_Count++
		return
	}
	// Exact names, since a case-insensitive disk would resolve a differently cased file.
	if entry.Name == requirements.MARKDOWN_NAME {
		runner.Pending_Markdown = true
	}
	if entry.Name == requirements.TEST_NAME {
		runner.Pending_Test = true
	}
	if len(entry.Name) < len(GO_SUFFIX) {
		return
	}
	if entry.Name[len(entry.Name)-len(GO_SUFFIX):] != GO_SUFFIX {
		return
	}
	if int(runner.File_Count) == len(runner.Memory.Paths) {
		runner.Walk_Failed = true
		return
	}
	path := runner.Memory.Paths[runner.File_Count][:0]
	path = append(append(path, current...), PATH_SEPARATOR)
	path = append(path, entry.Name...)
	status, status_err := nbio.Storage_Status(
		runner.IO.Storage,
		unsafe.String(unsafe.SliceData(path), len(path)),
	)
	if status_err != nil {
		runner.Walk_Failed = true
		return
	}
	if !nbio.File_Mode_Is_Regular(status.Mode) {
		return
	}
	runner.Memory.Paths[runner.File_Count] = path
	runner.Memory.File_Packages[runner.File_Count] = runner.Package_Count
	runner.File_Count++
	runner.Pending_Go = true
}

func runner_judge_directory_close(runner Runner_Handle) {
	Runner_Handle_Invariants(runner, "runner_judge_directory_close.runner")
	if runner.Completion.Error != nil {
		runner_fail(runner)
		return
	}
	if runner.Walk_Failed {
		runner_fail(runner)
		return
	}
	// A directory joins the package table once it proves to hold a Go file; its files already
	// name the slot it takes.
	if runner.Pending_Go {
		if int(runner.Package_Count) == len(runner.Memory.Packages) {
			runner_fail(runner)
			return
		}
		slot := &runner.Memory.Packages[runner.Package_Count]
		slot.Path = append(slot.Path[:0], runner.Memory.Current...)
		slot.Markdown = runner.Pending_Markdown
		slot.Test = runner.Pending_Test
		slot.Impure = false
		runner.Package_Count++
	}
	if runner.Directory_Count > 0 {
		runner.Stage = STAGE_DIRECTORY_OPEN
		return
	}
	runner_sort(runner)
	runner.File_Index = 0
	runner_file_next(runner)
}

// A directory pass reports children in whatever order the kernel keeps, thus the paths sort
// before any line is written, or two runs over one tree would disagree.
func runner_sort(runner Runner_Handle) {
	Runner_Handle_Invariants(runner, "runner_sort.runner")
	paths := runner.Memory.Paths[:runner.File_Count]
	owners := runner.Memory.File_Packages[:runner.File_Count]
	for index := 1; index < len(paths); index++ {
		held := paths[index]
		owner := owners[index]
		position := index
		for position > 0 {
			if string(held) < string(paths[position-1]) {
				paths[position] = paths[position-1]
				owners[position] = owners[position-1]
				position--
				continue
			}
			break
		}
		paths[position] = held
		owners[position] = owner
	}
}

// A descriptor outlives a failed read, thus the failure waits for the close.
func runner_judge_read(runner Runner_Handle) {
	Runner_Handle_Invariants(runner, "runner_judge_read.runner")
	if runner.Completion.Error != nil {
		runner.Read_Failed = true
		runner.Stage = STAGE_CLOSE
		return
	}
	if runner.Completion.Data == 0 {
		runner.Stage = STAGE_CLOSE
		return
	}
	if runner.Reading == READING_MARKDOWN {
		checker := runner.Memory.Requirements
		checker.Markdown_Count += requirements.Index(runner.Completion.Data)
		if int(checker.Markdown_Count) == len(checker.Markdown) {
			runner.Stage = STAGE_CLOSE
		}
		return
	}
	runner.Source_Count += Source_Count(runner.Completion.Data)
	if int(runner.Source_Count) == len(runner.Memory.Source) {
		runner.Stage = STAGE_CLOSE
	}
}

func runner_judge_close(runner Runner_Handle) {
	Runner_Handle_Invariants(runner, "runner_judge_close.runner")
	if runner.Completion.Error != nil {
		runner.Step = STEP_CLOSE
		runner_fail(runner)
		return
	}
	if runner.Read_Failed {
		runner.Step = STEP_READ
		runner_fail(runner)
		return
	}
	checker := runner.Memory.Requirements
	switch runner.Reading {
	case READING_MARKDOWN:
		requirements.Begin(checker)
		runner_requirements_next(runner)
		return
	case READING_TEST:
		// A test file past the source bound has its own diagnostic; an empty source refuses
		// to parse, thus its tests go unchecked rather than flagged twice.
		checker.Test_Source = nil
		if int(runner.Source_Count) < len(runner.Memory.Source) {
			test := runner.Memory.Source[:runner.Source_Count]
			checker.Test_Source = token.Source(test)
		}
		runner_requirements_next(runner)
		return
	}
	Check(runner)
	runner_impure(runner)
	runner.Resume = RESUME_FILE
	if runner.Diagnostic.Tier == TIER_NONE {
		runner.File_Index++
		runner_file_next(runner)
		return
	}
	if runner.Diagnostic.Tier == TIER_ONE {
		if runner.Autofix {
			runner.Fix_Count = 0
			runner.Fix_Failed = false
			runner.Stage = STAGE_FIX_OPEN
			return
		}
	}
	runner.Diagnostic_Count++
	if runner.Json {
		runner_report_json(runner)
		return
	}
	path := runner.Memory.Paths[runner.File_Index]
	runner.Line = append(Line(runner.Memory.Line[:0]), path...)
	switch runner.Diagnostic.Rule {
	case RULE_UNFORMATTED:
		runner.Line = append(append(runner.Line, SEPARATOR...), MESSAGE_UNFORMATTED...)
	case RULE_REFUSED:
		runner_line_position(runner)
		message := ast.Failure_Message(runner.Diagnostic.Failure)
		runner.Line = append(append(runner.Line, SEPARATOR...), message...)
	case RULE_OVERSIZED:
		runner.Line = append(append(runner.Line, SEPARATOR...), MESSAGE_OVERSIZED...)
	}
	runner.Line = append(runner.Line, '\n')
	runner.Stage = STAGE_REPORT
}

// Writes the diagnostic as one flat JSON line. The leaves view stored bytes and constants, since a
// conversion would allocate.
func runner_report_json(runner Runner_Handle) {
	Runner_Handle_Invariants(runner, "runner_report_json.runner")
	path := runner.Memory.Paths[runner.File_Index]
	diagnostic := runner.Diagnostic
	runner.Report = Report{
		Path:   Report_Text(unsafe.String(unsafe.SliceData(path), len(path))),
		Line:   diagnostic.Position.Line,
		Column: diagnostic.Position.Column,
		Tier:   diagnostic.Tier,
	}
	switch diagnostic.Rule {
	case RULE_UNFORMATTED:
		runner.Report.Rule = RULE_TEXT_UNFORMATTED
		runner.Report.Message = MESSAGE_UNFORMATTED
	case RULE_REFUSED:
		runner.Report.Rule = RULE_TEXT_REFUSED
		runner.Report.Message = Report_Text(ast.Failure_Message(diagnostic.Failure))
	case RULE_OVERSIZED:
		runner.Report.Rule = RULE_TEXT_OVERSIZED
		runner.Report.Message = MESSAGE_OVERSIZED
	}
	runner_report_write(runner)
}

// Marshals runner.Report as one flat JSON line.
func runner_report_write(runner Runner_Handle) {
	Runner_Handle_Invariants(runner, "runner_report_write.runner")
	storage := runner.Memory.Json
	data, err := flatjson.Marshal_Into(
		flatjson.Output(storage[:flatjson.OUTPUT_SIZE_MAXIMUM]),
		flatjson.Value(reflect.ValueOf(&runner.Report)),
	)
	// Only a path of bytes JSON escapes six-fold outgrows the object, and the run names it
	// rather than cut the line short.
	if err != nil {
		runner.Step = STEP_REPORT
		runner_fail(runner)
		return
	}
	line := storage[:len(data)+len("\n")]
	line[len(data)] = '\n'
	runner.Line = Line(line)
	runner.Stage = STAGE_REPORT
}

// Appends `:<line>:<column>` of the refused token. A source past the line index has no position
// the index can name, thus its line names the path alone rather than a wrong place.
func runner_line_position(runner Runner_Handle) {
	Runner_Handle_Invariants(runner, "runner_line_position.runner")
	position := runner.Diagnostic.Position
	if position.Line < token.LINE_MINIMUM {
		return
	}
	// The digits land in the line's spare capacity, since a conversion would allocate.
	numbers := [POSITION_NUMBER_COUNT]strconv.Unsigned_Integer{
		strconv.Unsigned_Integer(position.Line),
		strconv.Unsigned_Integer(position.Column),
	}
	for _, number := range numbers {
		runner.Line = append(runner.Line, ':')
		size := len(runner.Line)
		spare := strconv.Buffer(runner.Line[size : size+strconv.DIGIT_TEXT_SIZE_MAXIMUM])
		count := strconv.Format_Unsigned_Integer_Into(spare, number, strconv.DECIMAL_BASE)
		runner.Line = runner.Line[:size+int(count)]
	}
}

func runner_file_next(runner Runner_Handle) {
	Runner_Handle_Invariants(runner, "runner_file_next.runner")
	if runner.File_Index < File_Index(runner.File_Count) {
		runner.Memory.Read_Path = append(
			runner.Memory.Read_Path[:0],
			runner.Memory.Paths[runner.File_Index]...,
		)
		runner.Reading = READING_SOURCE
		runner.Stage = STAGE_OPEN
		return
	}
	runner_package_sort(runner)
	runner.Package_Index = 0
	runner.Stage = STAGE_PACKAGE
}

// Records a package main file, whose package owes no contract. A refused file still carries the
// token run, thus its package clause still reads.
func runner_impure(runner Runner_Handle) {
	Runner_Handle_Invariants(runner, "runner_impure.runner")
	if runner.Diagnostic.Rule == RULE_OVERSIZED {
		return
	}
	run := ast.Parse_State_Token_Run(runner.Memory.Parse_State)
	source := runner.Memory.Source[:runner.Source_Count]
	for index := 0; index+1 < len(run); index++ {
		if run[index].Kind != token.KIND_PACKAGE {
			continue
		}
		name := run[index+1]
		text := string(source[name.Offset : int(name.Offset)+int(name.Size)])
		owner := &runner.Memory.Packages[runner.Memory.File_Packages[runner.File_Index]]
		if text == "main" {
			owner.Impure = true
		}
		if text == "main_test" {
			owner.Impure = true
		}
		return
	}
}

// Packages sort by path, so their findings come out in one order whatever the kernel's.
func runner_package_sort(runner Runner_Handle) {
	Runner_Handle_Invariants(runner, "runner_package_sort.runner")
	packages := runner.Memory.Packages[:runner.Package_Count]
	for index := 1; index < len(packages); index++ {
		held := packages[index]
		position := index
		for position > 0 {
			if string(held.Path) < string(packages[position-1].Path) {
				packages[position] = packages[position-1]
				position--
				continue
			}
			break
		}
		packages[position] = held
	}
}

// Starts the contract check of the next package, reading its REQUIREMENTS.md first when it has
// one; past the last package, the run ends.
func runner_package_next(runner Runner_Handle) {
	Runner_Handle_Invariants(runner, "runner_package_next.runner")
	if runner.Package_Index == runner.Package_Count {
		runner_finish(runner)
		return
	}
	found := runner.Memory.Packages[runner.Package_Index]
	checker := runner.Memory.Requirements
	checker.Path = found.Path
	checker.Has_Markdown = requirements.Flag(found.Markdown)
	checker.Has_Test = requirements.Flag(found.Test)
	checker.Impure = requirements.Flag(found.Impure)
	checker.Markdown_Count = 0
	if found.Markdown {
		runner.Read_File = requirements.FILE_MARKDOWN
		runner_read_path(runner)
		runner.Reading = READING_MARKDOWN
		runner.Stage = STAGE_OPEN
		return
	}
	requirements.Begin(checker)
	runner.Stage = STAGE_REQUIREMENTS
}

// Resumes the open package's check: a finding is reported, a request for requirements_test.go
// reads it, and the end moves to the next package.
func runner_requirements_next(runner Runner_Handle) {
	Runner_Handle_Invariants(runner, "runner_requirements_next.runner")
	checker := runner.Memory.Requirements
	requirements.Next(checker)
	switch checker.State {
	case requirements.STATE_FINDING:
		runner_report_requirement(runner)
	case requirements.STATE_TEST:
		runner.Read_File = requirements.FILE_TEST
		runner_read_path(runner)
		runner.Reading = READING_TEST
		runner.Stage = STAGE_OPEN
	case requirements.STATE_DONE:
		runner.Package_Index++
		runner.Stage = STAGE_PACKAGE
	}
}

// Points Read_Path at the Read_File of the open package.
func runner_read_path(runner Runner_Handle) {
	Runner_Handle_Invariants(runner, "runner_read_path.runner")
	path := append(runner.Memory.Read_Path[:0], runner.Memory.Requirements.Path...)
	path = append(path, PATH_SEPARATOR)
	if runner.Read_File == requirements.FILE_TEST {
		runner.Memory.Read_Path = append(path, requirements.TEST_NAME...)
		return
	}
	runner.Memory.Read_Path = append(path, requirements.MARKDOWN_NAME...)
}

// Writes one contract finding at its file, as a text line or a flat JSON line.
func runner_report_requirement(runner Runner_Handle) {
	Runner_Handle_Invariants(runner, "runner_report_requirement.runner")
	checker := runner.Memory.Requirements
	runner.Read_File = checker.File
	runner_read_path(runner)
	runner.Diagnostic_Count++
	runner.Resume = RESUME_REQUIREMENTS
	path := runner.Memory.Read_Path
	message := checker.Message
	if runner.Json {
		message_text := unsafe.String(unsafe.SliceData(message), len(message))
		runner.Report = Report{
			Path:    Report_Text(unsafe.String(unsafe.SliceData(path), len(path))),
			Line:    token.Line(checker.Line),
			Tier:    TIER_TWO,
			Rule:    RULE_TEXT_REQUIREMENTS,
			Message: Report_Text(message_text),
		}
		runner_report_write(runner)
		return
	}
	runner.Line = append(Line(runner.Memory.Line[:0]), path...)
	if checker.Line > 0 {
		runner.Line = append(runner.Line, ':')
		size := len(runner.Line)
		spare := strconv.Buffer(runner.Line[size : size+strconv.DIGIT_TEXT_SIZE_MAXIMUM])
		number := strconv.Unsigned_Integer(checker.Line)
		count := strconv.Format_Unsigned_Integer_Into(spare, number, strconv.DECIMAL_BASE)
		runner.Line = runner.Line[:size+int(count)]
	}
	runner.Line = append(append(runner.Line, SEPARATOR...), message...)
	runner.Line = append(runner.Line, '\n')
	runner.Stage = STAGE_REPORT
}

// Ends the run: any diagnostic fails it, and a clean run writes the pass line.
func runner_finish(runner Runner_Handle) {
	Runner_Handle_Invariants(runner, "runner_finish.runner")
	runner.Status = EXIT_SUCCESS
	if runner.Diagnostic_Count > 0 {
		runner.Status = EXIT_FAILURE
		runner.Phase = PHASE_STOPPED
		return
	}
	runner_passed(runner)
}

// Writes the pass line and the elapsed seconds to the millisecond. The digits land in the line's
// spare capacity, since a conversion would allocate.
func runner_passed(runner Runner_Handle) {
	Runner_Handle_Invariants(runner, "runner_passed.runner")
	now := time.Clock_Now_Monotonic(runner.Clock)
	milliseconds := int64(now-runner.Start) / int64(time.MILLISECOND)
	runner.Line = append(Line(runner.Memory.Line[:0]), PASSED...)
	size := len(runner.Line)
	spare := strconv.Buffer(runner.Line[size : size+strconv.DIGIT_TEXT_SIZE_MAXIMUM])
	seconds := strconv.Unsigned_Integer(milliseconds / MILLISECONDS_PER_SECOND)
	count := strconv.Format_Unsigned_Integer_Into(spare, seconds, strconv.DECIMAL_BASE)
	runner.Line = append(runner.Line[:size+int(count)], '.')
	fraction := milliseconds % MILLISECONDS_PER_SECOND
	runner.Line = append(
		runner.Line,
		byte('0'+fraction/100),
		byte('0'+fraction/10%10),
		byte('0'+fraction%10),
		'\n',
	)
	runner.Stage = STAGE_PASSED
	if !runner.Json {
		return
	}
	// The elapsed text already stands in the line, thus the summary views it rather than write
	// it twice.
	elapsed := runner.Line[len(PASSED) : len(runner.Line)-len("\n")]
	seconds_text := unsafe.String(unsafe.SliceData(elapsed), len(elapsed))
	runner.Summary = Summary{Passed: true, Elapsed_Seconds: Report_Text(seconds_text)}
	storage := runner.Memory.Json
	data, err := flatjson.Marshal_Into(
		flatjson.Output(storage[:flatjson.OUTPUT_SIZE_MAXIMUM]),
		flatjson.Value(reflect.ValueOf(&runner.Summary)),
	)
	aver.Always(err == nil, "A pass summary fits flat JSON output.")
	line := storage[:len(data)+len("\n")]
	line[len(data)] = '\n'
	runner.Line = Line(line)
}

// Writes the failure line of runner.Step. A walk names the directory it had open, and any other
// step names the current file.
func runner_fail(runner Runner_Handle) {
	Runner_Handle_Invariants(runner, "runner_fail.runner")
	line := append(Line(runner.Memory.Line[:0]), FAILURE_PREFIX...)
	switch runner.Step {
	case STEP_WALK:
		line = append(line, "walk"...)
	case STEP_OPEN:
		line = append(line, "open"...)
	case STEP_READ:
		line = append(line, "read"...)
	case STEP_CLOSE:
		line = append(line, "close"...)
	case STEP_FIX:
		line = append(line, "fix"...)
	case STEP_REPORT:
		line = append(line, "report"...)
	}
	line = append(line, SEPARATOR...)
	if runner.Step == STEP_WALK {
		line = append(line, runner.Memory.Current...)
	} else {
		line = append(line, runner.Memory.Read_Path...)
	}
	runner.Line = append(line, '\n')
	runner.Status = EXIT_FAILURE
	runner.Stage = STAGE_FAIL
	runner.Phase = PHASE_READY
}

// Records only; Rearm judges, so no continuation runs inside the loop's own stack.
func runner_retired(completion nbio.Completion_Handle) {
	Runner_Handle(unsafe.Pointer(completion)).Phase = PHASE_RETIRED
}
