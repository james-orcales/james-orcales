// Package harness is the pure library tier of the harness linter: it walks one Go package and
// reports each file that breaks a rule. Every effect is submitted to the injected IO surface —
// package main binds the real loop, allocates every store, and pumps.
package harness

import "errors"
import "reflect"
import "unsafe"

import "local/james-orcales/harness/pkg/configuration"
import "local/james-orcales/harness/pkg/requirements"
import "local/james-orcales/harness/pkg/vcs"
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

// RULE_OVERSIZED is a file past requirements.SOURCE_SIZE_MAXIMUM.
const RULE_OVERSIZED Rule = 3

// RULE_UNPRINTABLE is a file whose canonical form outgrows the store the printer writes into. A
// source stands inside its own bound and still spends more than that store, since indentation
// adds bytes the source never held.
const RULE_UNPRINTABLE Rule = 4

// RULE_CONFLICTED is a file with a line that opens on a merge conflict marker.
const RULE_CONFLICTED Rule = 6

// RULE_IMPORTS is a Go file importing a stdlib package lint bans.
const RULE_IMPORTS Rule = 5

// MESSAGE_UNPRINTABLE is what a line says of RULE_UNPRINTABLE.
const MESSAGE_UNPRINTABLE = "canonical form exceeds its storage"

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

// RULE_TEXT_REQUIREMENTS names a package contract that breaks the REQUIREMENTS.md doctrine. A
// contract finding carries its text alone, since the checker states the rule and the run writes
// what it states rather than a rule of its own.
const RULE_TEXT_REQUIREMENTS Rule_Text = "requirements"

// PACKAGE_COUNT_MAXIMUM is one package per Go file at most.
const PACKAGE_COUNT_MAXIMUM = FILE_COUNT_MAXIMUM

// RESUME_FILE continues with the next Go file after a report.
const RESUME_FILE Resume = 0

// RESUME_REQUIREMENTS continues the open package's contract checks after a report.
const RESUME_REQUIREMENTS Resume = 1

// RESUME_HISTORY continues the history walk after a report.
const RESUME_HISTORY Resume = 2

// READING_SOURCE reads a Go file to check it.
const READING_SOURCE Reading = 0

// READING_MARKDOWN reads a REQUIREMENTS.md.
const READING_MARKDOWN Reading = 1

// READING_TEST reads a requirements_test.go.
const READING_TEST Reading = 2

// READING_DECLARATION reads a Go file for the names it declares.
const READING_DECLARATION Reading = 3

// CONFIGURATION_PATH is the policy file a run reads from its working directory before the walk.
const CONFIGURATION_PATH = "harness.json"

// CONFIGURATION_PREFIX opens the line of a configuration that does not decode.
const CONFIGURATION_PREFIX = PROGRAM_PREFIX + CONFIGURATION_PATH + SEPARATOR

// BYTE_INFIX stands between a fault and the byte it stands on.
const BYTE_INFIX = " at byte "

// FLAG_FIX labels the flag that writes each fix over its file.
const FLAG_FIX = "fix"

// WRITE_TIMEOUT bounds one write of a fix to a local file.
const WRITE_TIMEOUT = READ_TIMEOUT

// RULE_TEXT_UNFORMATTED names RULE_UNFORMATTED in flat JSON.
const RULE_TEXT_UNFORMATTED Rule_Text = "unformatted"

// RULE_TEXT_REFUSED names RULE_REFUSED in flat JSON.
const RULE_TEXT_REFUSED Rule_Text = "refused"

// RULE_TEXT_OVERSIZED names RULE_OVERSIZED in flat JSON.
const RULE_TEXT_OVERSIZED Rule_Text = "oversized"

// RULE_TEXT_UNPRINTABLE names RULE_UNPRINTABLE in flat JSON.
const RULE_TEXT_UNPRINTABLE Rule_Text = "unprintable"

// RULE_TEXT_CONFLICTED names RULE_CONFLICTED in flat JSON.
const RULE_TEXT_CONFLICTED Rule_Text = "conflicted"

// RULE_TEXT_IMPORTS names RULE_IMPORTS in flat JSON.
const RULE_TEXT_IMPORTS Rule_Text = "imports"

// RULE_TEXT_SYMLINKS names a symbolic link whose target resolves to nothing.
const RULE_TEXT_SYMLINKS Rule_Text = "symlinks"

// MESSAGE_ORPHAN is what a line says of a symbolic link whose target resolves to nothing.
const MESSAGE_ORPHAN = "The symlink's target is absent. Remove the symlink or restore its target."

// ORPHAN_COUNT_MAXIMUM bounds the symbolic links one walk records whose target resolves to
// nothing. A tree past it fails the walk, as a tree past any other table does.
const ORPHAN_COUNT_MAXIMUM = 16

// LINK_COUNT_MAXIMUM is how many links one resolution follows before it names the target absent.
const LINK_COUNT_MAXIMUM = 8

// LINK_TARGET_SIZE is one byte past the widest path: a target that fills the store is one no path
// holds.
const LINK_TARGET_SIZE = filepath.PATH_SIZE_MAXIMUM + 1

// PROGRAM_LABEL names the program in help and in every diagnostic prefix.
const PROGRAM_LABEL = "harness"

// PROGRAM_DESCRIPTION states what the program does in help.
const PROGRAM_DESCRIPTION = "report each Go file that breaks a rule"

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

// ARGUMENT_COUNT_MINIMUM is the program name, which every vector a host execs holds.
const ARGUMENT_COUNT_MINIMUM = 1

// USAGE_ARGUMENTS_ABSENT names a vector holding no program name.
const USAGE_ARGUMENTS_ABSENT = "the argument vector names no program"

// USAGE_ABSENT_LINE is the line a vector holding no program name writes.
const USAGE_ABSENT_LINE = PROGRAM_PREFIX + USAGE_ARGUMENTS_ABSENT + "\n"

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

// LINE_SIZE_MAXIMUM holds any text line one run writes: a refused command line, a failed step, a
// file diagnostic, or a contract finding. A flat JSON diagnostic is the widest line a run states
// of its own, thus the text store holds as much, and a refused command line states only as much
// of the error shared/cli renders as that store holds, since the error echoes what a host typed.
const LINE_SIZE_MAXIMUM = JSON_STORAGE_SIZE

// USAGE_ERROR_SIZE_MAXIMUM is how much of the error shared/cli renders one line states: the store
// past the prefix and the newline that closes the line.
const USAGE_ERROR_SIZE_MAXIMUM = LINE_SIZE_MAXIMUM - len(PROGRAM_PREFIX) - len("\n")

// WRITE_SIZE_MAXIMUM holds any one write: a text line, a flat JSON line, or the help. One write
// views one store, the text and flat JSON stores hold one width, and the help states less.
const WRITE_SIZE_MAXIMUM = LINE_SIZE_MAXIMUM

// FILE_COUNT_MAXIMUM bounds the Go files one run collects. A run checks any tree of this
// repository, the shared component among them, thus the table holds each of its Go files with
// room to grow.
const FILE_COUNT_MAXIMUM = 1024

// DIRECTORY_COUNT_MAXIMUM bounds the directories a walk holds pending at once, which one package
// tree holds as few of as it holds files.
const DIRECTORY_COUNT_MAXIMUM = FILE_COUNT_MAXIMUM

// ENTRY_COUNT_MAXIMUM bounds the children one directory pass reports.
const ENTRY_COUNT_MAXIMUM = filepath.FILESYSTEM_PATH_COUNT_MAXIMUM

// SOURCE_STORAGE_SIZE holds one byte past the largest source, so a full read proves a file
// oversized without a second read.
const SOURCE_STORAGE_SIZE = requirements.SOURCE_SIZE_MAXIMUM + 1

// FORM_STORAGE_SIZE holds the widest canonical form one source states. Indentation spends at most
// one byte per byte of the source it indents, thus the form stands at twice the source.
const FORM_STORAGE_SIZE = 2 * requirements.SOURCE_SIZE_MAXIMUM

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

// PHASE_OPENED is a run Main opened and Rearm has yet to parse.
const PHASE_OPENED Phase = 4

// ADVANCE_READY is a stage that named the next without IO.
const ADVANCE_READY Advance_Pass = 0

// ADVANCE_SUBMIT is a stage whose operation goes to the loop.
const ADVANCE_SUBMIT Advance_Pass = 1

// ADVANCE_STOPPED is a run the stage finished.
const ADVANCE_STOPPED Advance_Pass = 2

// STAGE_DIRECTORY_OPEN opens the next pending directory.
const STAGE_DIRECTORY_OPEN Stage = 0

// STAGE_DIRECTORY_READ reads one pass of the open directory's children.
const STAGE_DIRECTORY_READ Stage = 1

// STAGE_DIRECTORY_CLOSE closes the open directory.
const STAGE_DIRECTORY_CLOSE Stage = 2

// STAGE_OPEN opens the current file.
const STAGE_OPEN Stage = 10

// STAGE_READ reads the current file.
const STAGE_READ Stage = 3

// STAGE_CLOSE closes the current file and checks it.
const STAGE_CLOSE Stage = 4

// STAGE_REPORT writes the diagnostic of the current file.
const STAGE_REPORT Stage = 8

// STAGE_FAIL writes the diagnostic of a refused command line or a failed step.
const STAGE_FAIL Stage = 11

// STAGE_HELP writes the help -h asked for.
const STAGE_HELP Stage = 5

// STAGE_PASSED writes the pass line of a run without diagnostics.
const STAGE_PASSED Stage = 6

// STAGE_FIX_OPEN opens the current file to replace it with its fix.
const STAGE_FIX_OPEN Stage = 9

// STAGE_FIX_WRITE writes the rest of the fix.
const STAGE_FIX_WRITE Stage = 13

// STAGE_FIX_CLOSE closes the fixed file.
const STAGE_FIX_CLOSE Stage = 19

// STAGE_DECLARE reads the next Go file for the names it declares, without IO of its own. A stage
// of its own keeps the declaration pass a loop through Rearm rather than a recursion.
const STAGE_DECLARE Stage = 15

// STAGE_SERVE turns the run to the file after the one it served, without IO of its own. A stage
// of its own keeps the file walk a loop through Rearm rather than a recursion.
const STAGE_SERVE Stage = 16

// STAGE_DECLARED turns the declaration pass to the file after the one it declared, without IO of
// its own.
const STAGE_DECLARED Stage = 17

// STAGE_FINDING writes the line of the contract finding the check stands on, without IO of its
// own; the write of that line stands at the report stage after it.
const STAGE_FINDING Stage = 18

// STAGE_PACKAGED turns the contract check to the package after the one it finished, without IO
// of its own.
const STAGE_PACKAGED Stage = 14

// STAGE_HISTORY_NEXT judges the next commit of the log git read, without IO.
const STAGE_HISTORY_NEXT Stage = 20

// STAGE_HISTORY_HEAD spawns the git that names the branch HEAD stands on.
const STAGE_HISTORY_HEAD Stage = 21

// STAGE_HISTORY_REFS spawns the git that lists the main refs the repository holds.
const STAGE_HISTORY_REFS Stage = 22

// STAGE_HISTORY_LOG spawns the git that reads the log between the main ref and HEAD.
const STAGE_HISTORY_LOG Stage = 23

// STAGE_PACKAGE starts the contract check of the next package, without IO. A stage of its own
// keeps the package walk a loop through Rearm rather than a recursion.
const STAGE_PACKAGE Stage = 12

// STAGE_REQUIREMENTS resumes the open package's contract check, without IO.
const STAGE_REQUIREMENTS Stage = 7

// PASSED opens the pass line; the elapsed seconds follow it.
const PASSED = "✓ all checks passed\nelapsed_sec="

// MILLISECONDS_PER_SECOND splits elapsed milliseconds into seconds and their fraction.
const MILLISECONDS_PER_SECOND = 1000

// Main opens a run: it stamps the start and stands the run where Rearm parses the command line.
// It returns without waiting: the root repeats Rearm and pumps.
func Main(input Main_Input) {
  Main_Input_Invariants(input, "main.input")
  *input.Start = time.Clock_Now_Monotonic(input.Clock)
  *input.Phase = Opening_Phase(PHASE_OPENED)
}

// Declares the command line into run storage, since a declaration literal would allocate.
func options_declare(arguments Argument_Declarations, flags Flag_Declarations) {
  Argument_Declarations_Invariants(arguments, "options_declare.arguments")
  Flag_Declarations_Invariants(flags, "options_declare.flags")
  arguments[0] = cli.New_Option(
    cli.New_Option_Input{
      Label:       ARGUMENT_DIRECTORY,
      Description: "Go package to check, at any depth",
    },
  )
  flags[0] = cli.New_Option(
    cli.New_Option_Input{
      Label:       FLAG_JSON,
      Description: "write each diagnostic as one line of flat JSON",
      Type:        cli.OPTION_TYPE_BOOLEAN,
      Is_Flag:     true,
    },
  )
  flags[1] = cli.New_Option(
    cli.New_Option_Input{
      Label:       FLAG_FIX,
      Description: "replace each auto-fixable file with its fix",
      Type:        cli.OPTION_TYPE_BOOLEAN,
      Is_Flag:     true,
    },
  )
}

// Parses the command line and records what it holds. A cli.Parser composes its own
// declaration types twice over, thus a run that held one could state no chain, and the program
// stands declared here alone, since each run declares the same one.
func parse(
  memory Memory_Handle,
  findings Findings_Handle,
  arguments Program_Arguments,
  run_flags Run_Flag_Storage,
  command Command,
  outcome Outcome_Handle,
) (next Command_Pass) {
  defer func() { Command_Pass_Invariants(next, "parse.next") }()
  Command_Invariants(command, "parse.command")
  Run_Flag_Storage_Invariants(run_flags, "parse.run_flags")
  Memory_Handle_Invariants(memory, "parse.memory")
  Findings_Handle_Invariants(findings, "parse.findings")
  Program_Arguments_Invariants(arguments, "parse.arguments")
  Outcome_Handle_Invariants(outcome, "parse.outcome")
  options_declare(command.Declared_Arguments, command.Declared_Flags)
  program := cli.New(
    cli.New_Input{
      Mode:        cli.PROGRAM_MODE_SINGLE,
      Description: PROGRAM_DESCRIPTION,
      Label:       PROGRAM_LABEL,
      Arguments:   cli.Arguments(command.Declared_Arguments),
      Flags:       cli.Flags(command.Declared_Flags),
    },
  )
  // The parse stores span the run: cli.Program_Parse leaks its input, thus a store this
  // frame owned would land on the heap.
  parser := cli.Parser{}
  cli.Program_Parse(
    program,
    &parser,
    cli.Program_Parse_Input{
      Arguments:         cli.Process_Arguments(arguments),
      Command_Arguments: cli.Command_Argument_Storage(command.Command_Arguments),
      Command_Flags:     cli.Command_Flag_Storage(command.Command_Flags),
      Filled:            cli.Filled_Options(command.Filled),
      Positionals:       cli.Positionals(command.Positionals),
      Slice_Named:       cli.Named_Variadic_Tokens(command.Slice_Named),
      Failure_Storage:   cli.Parse_Failure_Storage(command.Failure),
    },
  )
  // A program without secrets parses synchronously, thus the parse is already done.
  done, complete := cli.Parser_Done(&parser)
  aver.Always(bool(complete), "A program without secrets parses synchronously.")
  failure := cli.Parse_Error_Bytes(done.Error)
  if cli.Parse_Error_Equals(done.Error, cli.HELP_REQUESTED) {
    storage := strings.Bytes(command.Help)
    output := cli.Output{Builder: cli.Output_Builder{Storage: storage}}
    help := cli.Output_Reference_Of(&output)
    cli.Print_Help(help, program)
    findings.Line, outcome.Status = Line(cli.Output_Bytes(help)), EXIT_SUCCESS
    return Command_Pass(STAGE_HELP)
  }
  if len(failure) > 0 {
    findings.Line = append(Line(memory.Line[:0]), PROGRAM_PREFIX...)
    // The error names a command line a host typed, thus a byte that steers a
    // terminal stands as one that does not rather than reaches whatever reads it, and
    // the line states as much of it as the store holds rather than grows past it.
    for index := range min(len(failure), USAGE_ERROR_SIZE_MAXIMUM) {
      written := failure[index]
      if written < NAME_BYTE_MINIMUM {
        written = requirements.QUOTED_BYTE_ABSENT
      }
      findings.Line = append(findings.Line, written)
    }
    findings.Line, outcome.Status = append(findings.Line, '\n'), EXIT_FAILURE
    return Command_Pass(STAGE_FAIL)
  }
  flags := cli.Resolved_Options(done.Command.Flags)
  run_flags.Values[RUN_JSON] = Mark(cli.Option_Boolean(flags, FLAG_JSON))
  run_flags.Values[RUN_AUTOFIX] = Mark(cli.Option_Boolean(flags, FLAG_FIX))
  root := cli.Option_String(cli.Resolved_Options(done.Command.Arguments), ARGUMENT_DIRECTORY)
  return Command_Pass(record(memory, findings, root, run_flags, outcome))
}

// States the directory one parse named and stands the walk on it. The argument is untrusted, thus
// a root the walk cannot stand on names the step that failed here rather than a directory.
func record(
  memory Memory_Handle,
  findings Findings_Handle,
  root cli.Value_Text,
  run_flags Run_Flag_Storage,
  outcome Outcome_Handle,
) (next Record_Pass) {
  defer func() { Record_Pass_Invariants(next, "record.next") }()
  Run_Flag_Storage_Invariants(run_flags, "record.run_flags")
  Memory_Handle_Invariants(memory, "record.memory")
  Findings_Handle_Invariants(findings, "record.findings")
  cli.Value_Text_Invariants(root, "record.root")
  Outcome_Handle_Invariants(outcome, "record.outcome")
  // A trailing separator stands in every path the walk joins under it, thus the run drops
  // it here rather than names a file twice over.
  for len(root) > 0 {
    if root[len(root)-1] != PATH_SEPARATOR {
      break
    }
    root = root[:len(root)-1]
  }
  // A path past the host bound fits no path slot, and one of no bytes names nothing to walk.
  if len(root) > filepath.PATH_SIZE_MAXIMUM {
    root = ""
  }
  unwalkable := Boolean(len(root) < PATH_SIZE_MINIMUM)
  // A byte below a space breaks the line that names it in two the same way a child's name
  // would.
  if !bool(unwalkable) {
    unwalkable = name_refused(Spelled_Name(root))
  }
  if bool(unwalkable) {
    run_flags.Values[RUN_WALK_ARGUMENT] = true
    fail(memory, findings, Failure_Path(nil), STEP_WALK, run_flags, outcome)
    return Record_Pass(STAGE_FAIL)
  }
  // The walk opens on the one directory a command line names once the run read its
  // configuration, which the caller names.
  memory.Directories[0] = append(memory.Directories[0][:0], root...)
  return Record_Pass(STAGE_OPEN)
}

// Check records in runner.Diagnostic the rule that Memory.Source[:Source_Count] breaks, or
// TIER_NONE.
func Check(
  memory Memory_Handle,
  syntax requirements.Syntax,
  count Source_Count,
  format_state Format,
) (diagnostic Diagnostic) {
  defer func() { Diagnostic_Invariants(diagnostic, "check.diagnostic") }()
  Memory_Handle_Invariants(memory, "check.memory")
  requirements.Syntax_Invariants(syntax, "check.syntax")
  Source_Count_Invariants(count, "check.count")
  Format_Invariants(format_state, "Check.format_state")
  file := memory.Source[:count]
  diagnostic = Diagnostic{Tier: TIER_NONE, Rule: RULE_NONE}
  if len(file) > requirements.SOURCE_SIZE_MAXIMUM {
    return Diagnostic{Tier: TIER_TWO, Rule: RULE_OVERSIZED}
  }
  source := token.Source(file)
  // A marker is no Go, thus the parser would refuse it in words that name no conflict.
  conflicted := conflict_line(Checked_Source(file))
  if conflicted != POSITION_ABSENT {
    return Diagnostic{Tier: TIER_TWO, Rule: RULE_CONFLICTED, Line: conflicted, Column: 1}
  }
  state := ast.Parse_State{Syntax_Tree: *syntax.Syntax_Tree}
  parsed := ast.Parse(&state, source)
  if !bool(parsed.OK) {
    // The parser's own refusal: its code, which names the fix, and where the refused
    // token stands. A source with more lines than the index holds keeps position zero.
    failure := Refusal(parsed.Failure)
    diagnostic = Diagnostic{Tier: TIER_TWO, Rule: RULE_REFUSED, Failure: failure}
    if !bool(token.Index_Lines(syntax.Lines, source)) {
      return diagnostic
    }
    token_at := ast.Token_At(&state, parsed.Failure_Token)
    position := token.Position_Of(syntax.Lines, token_at.Offset)
    diagnostic.Line = Diagnostic_Line(position.Line)
    diagnostic.Column = Diagnostic_Column(position.Column)
    return diagnostic
  }
  view := format.Formatter{
    Places:       format_state.Formatter.Places,
    Table_Fields: format_state.Formatter.Table,
  }
  result := format.Format(
    &view, format_state.Printer, printer.Form(memory.Form),
    &state, source,
  )
  // Indentation spends bytes the source never held, thus a source inside its own bound can
  // still state a form past the store the printer writes into. The run names that rather
  // than stands on a form it never wrote.
  if !bool(result.OK) {
    return Diagnostic{Tier: TIER_TWO, Rule: RULE_UNPRINTABLE}
  }
  form := memory.Form[:result.Count]
  if string(form) == string(source) {
    return diagnostic
  }
  return Diagnostic{Tier: TIER_ONE, Rule: RULE_UNFORMATTED, Fix: Fix(form)}
}

// POSITION_ABSENT names no position: a source no line index could place.
const POSITION_ABSENT = 0

// Checked_Source is a Go file Check takes, inside requirements.SOURCE_SIZE_MAXIMUM.
type Checked_Source []byte

// Checked_Source_Invariants bounds the source by the widest file a check takes.
func Checked_Source_Invariants(value Checked_Source, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(
    len(value),
    bytes.SLICE_SIZE_MINIMUM,
    requirements.SOURCE_SIZE_MAXIMUM,
  ).
    Ensure()
}

// Reports the first line of the source that opens on a merge conflict marker, or
// POSITION_ABSENT. A line past token.LINE_MAXIMUM goes unread, and the parser refuses it.
func conflict_line(source Checked_Source) (line Diagnostic_Line) {
  defer func() { Diagnostic_Line_Invariants(line, "conflict_line.line") }()
  Checked_Source_Invariants(source, "conflict_line.source")
  number := 1
  for offset := range source {
    if number > token.LINE_MAXIMUM {
      return POSITION_ABSENT
    }
    opened := offset == 0
    if offset > 0 {
      opened = source[offset-1] == '\n'
    }
    if !opened {
      continue
    }
    opening := source[offset:min(len(source), offset+requirements.CONFLICT_MARKER_SIZE)]
    if requirements.Conflict_Marker(requirements.Line_Opening(opening)) {
      return Diagnostic_Line(number)
    }
    number++
  }
  return POSITION_ABSENT
}

// Diagnostic_Line is where a refused token stands, or POSITION_ABSENT.
type Diagnostic_Line int32

// Diagnostic_Line_Invariants admits every line, and the absent one.
func Diagnostic_Line_Invariants(value Diagnostic_Line, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int32(int32(value), POSITION_ABSENT, token.LINE_MAXIMUM).
    Ensure()
}

// Diagnostic_Column is where a refused token stands, or POSITION_ABSENT.
type Diagnostic_Column int32

// Diagnostic_Column_Invariants admits every column, and the absent one.
func Diagnostic_Column_Invariants(value Diagnostic_Column, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int32(int32(value), POSITION_ABSENT, COLUMN_WIDEST).
    Ensure()
}

// Refusal is the parser's refusal as one diagnostic records it. The parse state holds the live
// one, which the next parse overwrites, thus a diagnostic keeps its own.
type Refusal uint8

// Refusal_Invariants states every refusal the parser names, and the clean parse.
func Refusal_Invariants(value Refusal, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint8(
    uint8(value),
    ast.FAILURE_CODE_MINIMUM,
    ast.FAILURE_CODE_MAXIMUM,
  ).
    Ensure()
}

// COLUMN_WIDEST is the widest column a token stands at: one past it is the sentinel the token
// package names, which no source reaches.
const COLUMN_WIDEST = requirements.SOURCE_SIZE_MAXIMUM

// Tier is whether a diagnostic is auto-fixable.
type Tier int

// Tier_Invariants admits no diagnostic and both tiers.
func Tier_Invariants(value Tier, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_3_Int(
    int(value), int(TIER_NONE), int(TIER_ONE), int(TIER_TWO),
  ).
    Ensure()
}

// Report_Tier is the tier a flat JSON diagnostic states.
type Report_Tier int

// Report_Tier_Invariants admits the two tiers one written report states. A report not yet
// written states the tier of a diagnostic carrying no fix, since a run writes no report of a
// file that broke nothing.
func Report_Tier_Invariants(value Report_Tier, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_Int(int(value), int(TIER_ONE), int(TIER_TWO)).
    Ensure()
}

// Report_Line is where a refused token stands, or POSITION_ABSENT.
type Report_Line int32

// Report_Line_Invariants admits every line, and the absent one.
func Report_Line_Invariants(value Report_Line, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int32(int32(value), POSITION_ABSENT, token.LINE_MAXIMUM).
    Ensure()
}

// Report_Column is where a refused token stands, or POSITION_ABSENT.
type Report_Column int32

// Report_Column_Invariants admits every column, and the absent one.
func Report_Column_Invariants(value Report_Column, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int32(int32(value), POSITION_ABSENT, COLUMN_WIDEST).
    Ensure()
}

// Rule is the rule a file breaks.
type Rule int

// Rule_Invariants admits no rule and each rule.
func Rule_Invariants(value Rule, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(int(value), int(RULE_NONE), int(RULE_CONFLICTED)).
    Ensure()
}

// Source_Path is the path of a Go file the run read.
type Source_Path []byte

// Source_Path_Invariants bounds a source path by the shortest a run can read: one directory byte,
// the separator, one name byte, and the suffix that makes it a Go file.
func Source_Path_Invariants(value Source_Path, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(
    len(value),
    READ_PATH_SIZE_MINIMUM,
    filepath.PATH_SIZE_MAXIMUM,
  ).
    Ensure()
}

// Fix is the whole file as it should stand.
type Fix []byte

// Fix_Invariants bounds a fix by the widest form one print writes. A tier two diagnostic carries
// no fix, thus no bytes stands; a fix that stands opens a package clause, thus one byte and two
// each name a form no print writes.
func Fix_Invariants(value Fix, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Holed_Int(
    len(value),
    bytes.SLICE_SIZE_MINIMUM,
    FORM_STORAGE_SIZE,
    FIX_HOLE_ONE,
    FIX_HOLE_TWO,
    FIX_HOLE_TWO,
    FIX_HOLE_TWO,
  ).
    Ensure()
}

// FIX_HOLE_ONE is the one byte no canonical form spends.
const FIX_HOLE_ONE = 1

// FIX_HOLE_TWO is the two bytes no canonical form spends.
const FIX_HOLE_TWO = 2

// Step is the step a failed run names.
type Step int

// Step_Invariants admits each step.
func Step_Invariants(value Step, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(int(value), int(STEP_WALK), int(STEP_FIX)).
    Ensure()
}

// PACKAGE_SLOT_MAXIMUM is the last slot of the package table, since a count of slots names one
// past the last that holds a package.
const PACKAGE_SLOT_MAXIMUM = PACKAGE_COUNT_MAXIMUM - 1

// Package_Slot is the package one file belongs to: a slot of the package table.
type Package_Slot int

// Package_Slot_Invariants bounds the slot by the package table it names.
func Package_Slot_Invariants(value Package_Slot, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(int(value), slices.COUNT_MINIMUM, PACKAGE_SLOT_MAXIMUM).
    Ensure()
}

// Package_Count counts the packages one walk found.
type Package_Count int

// Package_Count_Invariants bounds the count by the package table.
func Package_Count_Invariants(value Package_Count, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(int(value), slices.COUNT_MINIMUM, PACKAGE_COUNT_MAXIMUM).
    Ensure()
}

// Package_Index is the package whose contract the run checks.
type Package_Index int

// Package_Index_Invariants bounds the slot by the package table.
func Package_Index_Invariants(value Package_Index, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(int(value), slices.COUNT_MINIMUM, PACKAGE_COUNT_MAXIMUM).
    Ensure()
}

// CONTRACT_PATH_SIZE_MINIMUM is the shortest path one contract file stands at: the shortest path
// a package stands at, the separator under it, and the shorter of the two names it owes.
const CONTRACT_PATH_SIZE_MINIMUM = requirements.PATH_COUNT_MINIMUM +
  len("/") +
  len(requirements.MARKDOWN_NAME)

// Contract_Path is the path of one contract file the run reads. A contract file stands under a
// package directory by a name of its own, thus a path of no bytes stands nowhere here.
type Contract_Path []byte

// Contract_Path_Invariants bounds one contract path by the store that holds it.
func Contract_Path_Invariants(value Contract_Path, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(
    len(value),
    CONTRACT_PATH_SIZE_MINIMUM,
    filepath.PATH_SIZE_MAXIMUM,
  ).
    Ensure()
}

// Read_Path_Slice holds the path of the contract file the run reads.
type Read_Path_Slice []byte

// Read_Path_Slice_Invariants fixes the read path store.
func Read_Path_Slice_Invariants(value Read_Path_Slice, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(
    len(value),
    READ_PATH_SIZE_MINIMUM,
    filepath.PATH_SIZE_MAXIMUM,
  ).
    Ensure()
}

// Read_Path_Handle is the caller-owned path of the file the run reads.
type Read_Path_Handle *Read_Path_Slice

// Read_Path_Handle_Invariants composes a present read path.
func Read_Path_Handle_Invariants(value Read_Path_Handle, namespace aver.Namespace) {
  if value == nil {
    return
  }
  Read_Path_Slice_Invariants(*value, namespace)
}

// Program_Arguments is a command line that names its program: the open refuses a vector holding
// none before any parse.
type Program_Arguments os.Arguments

// Program_Arguments_Invariants bounds the vector by the program name and the host bound.
func Program_Arguments_Invariants(value Program_Arguments, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(len(value), ARGUMENT_COUNT_MINIMUM, slices.COUNT_MAXIMUM).
    Ensure()
}

// Mark is one yes or no a run records.
type Mark bool

// Mark_Invariants proves both answers occur.
func Mark_Invariants(value Mark, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Sometimes(bool(value), "The harness mark is set.").
    Ensure()
}

// RUN_JSON records whether diagnostics are written as flat JSON.
const RUN_JSON = 0

// RUN_AUTOFIX records whether each fix replaces its file.
const RUN_AUTOFIX = 1

// RUN_PENDING_GO records a Go file in the open directory.
const RUN_PENDING_GO = 2

// RUN_PENDING_MARKDOWN records a REQUIREMENTS.md in the open directory.
const RUN_PENDING_MARKDOWN = 3

// RUN_PENDING_TEST records a requirements_test.go in the open directory.
const RUN_PENDING_TEST = 4

// RUN_UNDECLARED records a file whose names the table never took, which bars every rename.
const RUN_UNDECLARED = 5

// RUN_WALK_FAILED records a failed directory pass, which reports once the directory is closed.
const RUN_WALK_FAILED = 6

// RUN_READ_FAILED records a failed file read, which reports once the file is closed.
const RUN_READ_FAILED = 7

// RUN_FIX_FAILED records a failed fix write, which reports once the file is closed.
const RUN_FIX_FAILED = 8

// RUN_HISTORY_DONE records that the run judged the history it stands on.
const RUN_HISTORY_DONE = 9

// RUN_CONFIGURED records that the run decoded harness.json, the first file it reads.
const RUN_CONFIGURED = 10

// RUN_WALK_ARGUMENT records a command line argument that names no directory to walk.
const RUN_WALK_ARGUMENT = 11

// RUN_WALK_FILES records a tree past the Go file table.
const RUN_WALK_FILES = 12

// RUN_WALK_DIRECTORIES records a tree past the table of directories pending.
const RUN_WALK_DIRECTORIES = 13

// RUN_WALK_ORPHANS records a tree past the table of broken symlinks.
const RUN_WALK_ORPHANS = 14

// RUN_WALK_NAME records a name holding a byte below a space.
const RUN_WALK_NAME = 15

// RUN_WALK_WIDE records a path past the path bound.
const RUN_WALK_WIDE = 16

// RUN_FLAG_COUNT is how many flags one run keeps.
const RUN_FLAG_COUNT = 17

// Run_Flags holds every yes or no one run keeps, and cannot grow.
type Run_Flags []Mark

// Run_Flags_Invariants fixes the table to one slot per flag.
func Run_Flags_Invariants(value Run_Flags, _ aver.Namespace) {
  aver.Always(len(value) == RUN_FLAG_COUNT, "Run flags have complete length.")
  aver.Always(cap(value) == RUN_FLAG_COUNT, "Run flags cannot grow.")
}

// Run_Flag_Storage keeps the flag table caller-owned.
type Run_Flag_Storage struct {
  // Values makes ownership explicit at the package boundary.
  Values Run_Flags
}

// Run_Flag_Storage_Invariants composes the flag table.
func Run_Flag_Storage_Invariants(value Run_Flag_Storage, namespace aver.Namespace) {
  Run_Flags_Invariants(value.Values, namespace)
}

// Resume is where a run continues after it writes a report.
type Resume int

// Resume_Invariants admits each continuation.
func Resume_Invariants(value Resume, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_3_Int(
    int(value),
    int(RESUME_FILE),
    int(RESUME_REQUIREMENTS),
    int(RESUME_HISTORY),
  ).
    Ensure()
}

// READ_FILE_MARKDOWN names the REQUIREMENTS.md of the open package.
const READ_FILE_MARKDOWN = Read_File(requirements.FILE_MARKDOWN)

// READ_FILE_TEST names the requirements_test.go of the open package.
const READ_FILE_TEST = Read_File(requirements.FILE_TEST)

// Read_File is the contract file of the open package.
type Read_File int

// Read_File_Invariants admits each contract file.
func Read_File_Invariants(value Read_File, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_Int(int(value), int(READ_FILE_MARKDOWN), int(READ_FILE_TEST)).
    Ensure()
}

// Reading is what the open file read serves.
type Reading int

// Reading_Invariants admits each purpose.
func Reading_Invariants(value Reading, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_4_Int(
    int(value),
    int(READING_SOURCE),
    int(READING_MARKDOWN),
    int(READING_TEST),
    int(READING_DECLARATION),
  ).
    Ensure()
}

// Package is one directory holding a Go file, as the walk found it.
type Package struct {
  // Path is the directory.
  Path Package_Path
  // Markdown records an exact-cased REQUIREMENTS.md.
  Markdown Markdown_Mark
  // Test records an exact-cased requirements_test.go.
  Test Test_Mark
  // Impure records a package main file, which exempts the directory from the contract.
  Impure Impure_Mark
}

// Package_Invariants composes one package. Each mark states its own kind, since one type
// standing for three of them would name a package whose parts the compiler lets a run swap.
func Package_Invariants(value Package, namespace aver.Namespace) {
  Package_Path_Invariants(value.Path, namespace)
  Markdown_Mark_Invariants(value.Markdown, namespace)
  Test_Mark_Invariants(value.Test, namespace)
  Impure_Mark_Invariants(value.Impure, namespace)
}

// Recorded_Package is one package the walk just took: its directory and the contract files it
// holds. A main file exempts a package from its contract, and the check that finds one stands
// past the walk, thus that mark stands nowhere here.
type Recorded_Package struct {
  // Path is the directory.
  Path Package_Path
  // Markdown records an exact-cased REQUIREMENTS.md.
  Markdown Markdown_Mark
  // Test records an exact-cased requirements_test.go.
  Test Test_Mark
}

// Recorded_Package_Invariants composes one package the walk took.
func Recorded_Package_Invariants(value Recorded_Package, namespace aver.Namespace) {
  Package_Path_Invariants(value.Path, namespace)
  Markdown_Mark_Invariants(value.Markdown, namespace)
  Test_Mark_Invariants(value.Test, namespace)
}

// Markdown_Mark records whether one package holds an exact-cased REQUIREMENTS.md.
type Markdown_Mark bool

// Markdown_Mark_Invariants proves both answers occur.
func Markdown_Mark_Invariants(value Markdown_Mark, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Sometimes(bool(value), "The package holds a contract markdown.").
    Ensure()
}

// Test_Mark records whether one package holds an exact-cased requirements_test.go.
type Test_Mark bool

// Test_Mark_Invariants proves both answers occur.
func Test_Mark_Invariants(value Test_Mark, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Sometimes(bool(value), "The package holds a contract test.").
    Ensure()
}

// Impure_Mark records whether one package holds a main file, which owes no contract.
type Impure_Mark bool

// Impure_Mark_Invariants proves both answers occur.
func Impure_Mark_Invariants(value Impure_Mark, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Sometimes(bool(value), "The package holds a main file.").
    Ensure()
}

// Packages holds every package the walk found.
type Packages []Package

// Packages_Invariants bounds the package table.
func Packages_Invariants(value Packages, _ aver.Namespace) {
  aver.Always(len(value) == PACKAGE_COUNT_MAXIMUM, "A package table has complete length.")
  aver.Always(cap(value) == PACKAGE_COUNT_MAXIMUM, "A package table cannot grow.")
}

// Package_Indices names each collected Go file's package, slot for slot with Memory.Paths.
type Package_Indices []Package_Slot

// Package_Indices_Invariants bounds the table by the file table.
func Package_Indices_Invariants(value Package_Indices, _ aver.Namespace) {
  aver.Always(len(value) == FILE_COUNT_MAXIMUM, "A file package table has complete length.")
  aver.Always(cap(value) == FILE_COUNT_MAXIMUM, "A file package table cannot grow.")
}

// Fix_Count counts the bytes of the fix written so far.
type Fix_Count int

// Fix_Count_Invariants bounds written bytes by the widest fix.
func Fix_Count_Invariants(value Fix_Count, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(
    int(value), slices.COUNT_MINIMUM, FORM_STORAGE_SIZE,
  ).
    Ensure()
}

// Path_Text is the path leaf of a flat JSON diagnostic. It views stored bytes.
type Path_Text string

// Path_Text_Invariants bounds the leaf by the widest flat JSON object.
func Path_Text_Invariants(value Path_Text, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(
    len(value),
    PATH_TEXT_SIZE_MINIMUM,
    filepath.PATH_SIZE_MAXIMUM,
  ).
    Ensure()
}

// RULE_TEXT_SIZE_MINIMUM is the shortest rule the harness names.
const RULE_TEXT_SIZE_MINIMUM = len(RULE_TEXT_REFUSED)

// RULE_TEXT_SIZE_MAXIMUM is the longest.
const RULE_TEXT_SIZE_MAXIMUM = len(RULE_TEXT_REQUIREMENTS)

// MESSAGE_SIZE_MINIMUM is the shortest message the harness writes.
const MESSAGE_SIZE_MINIMUM = len(MESSAGE_UNFORMATTED)

// MESSAGE_SIZE_MAXIMUM is the longest, which a contract finding states.
const MESSAGE_SIZE_MAXIMUM = requirements.MESSAGE_SIZE_MAXIMUM

// Rule_Text is the rule leaf of a flat JSON diagnostic. It views a constant.
type Rule_Text string

// Rule_Text_Invariants bounds the leaf by the widest flat JSON object.
func Rule_Text_Invariants(value Rule_Text, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(len(value), RULE_TEXT_SIZE_MINIMUM, RULE_TEXT_SIZE_MAXIMUM).
    Ensure()
}

// Message_Text is the message leaf of a flat JSON diagnostic. It views stored bytes or a constant.
type Message_Text string

// Message_Text_Invariants bounds the leaf by the widest flat JSON object.
func Message_Text_Invariants(value Message_Text, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(len(value), MESSAGE_SIZE_MINIMUM, MESSAGE_SIZE_MAXIMUM).
    Ensure()
}

// PATH_SIZE_MINIMUM is one byte, since every path names something.
const PATH_SIZE_MINIMUM = 1

// CONTRACT_NAME_SIZE_MAXIMUM is the widest contract file name a package directory holds. The
// test file outgrows the markdown one, thus a directory with room for it has room for both.
const CONTRACT_NAME_SIZE_MAXIMUM = len(requirements.TEST_NAME)

// READ_PATH_NONE is the path of a read not yet made, at the shortest a run reads.
const READ_PATH_NONE = "a/a" + GO_SUFFIX

// READ_PATH_SIZE_MINIMUM is the shortest path a run reads: one directory byte, the separator, and
// the shortest name a Go file carries.
const READ_PATH_SIZE_MINIMUM = PATH_SIZE_MINIMUM + len("/") + len("a") + len(GO_SUFFIX)

// PATH_TEXT_SIZE_MINIMUM is the shortest path one report names, which is a link the walk found.
const PATH_TEXT_SIZE_MINIMUM = ENTRY_PATH_SIZE_MINIMUM

// PATH_TEXT_NONE is the path of a report not yet written, at the shortest a report names.
const PATH_TEXT_NONE Path_Text = "a/a.go"

// ELAPSED_TEXT_NONE is the elapsed seconds of a summary not yet written.
const ELAPSED_TEXT_NONE Elapsed_Text = "0.000"

// ELAPSED_SIZE_MINIMUM is one second digit, the point, and the three milliseconds digits.
const ELAPSED_SIZE_MINIMUM = 1 + len(".") + 3

// Elapsed is how long one run has stood, the span between two readings of the injected clock.
type Elapsed time.Duration

// Elapsed_Invariants bounds a run by the uptime a monotonic moment holds: a run opens at one
// reading and stands at a later one, thus it spans no more than the host has been up.
func Elapsed_Invariants(value Elapsed, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int64(
    int64(value),
    int64(ELAPSED_MINIMUM),
    int64(time.MONOTONIC_MOMENT_MAXIMUM),
  ).
    Ensure()
}

// ELAPSED_MINIMUM is no time at all: two readings of one clock can state the same moment.
const ELAPSED_MINIMUM Elapsed = 0

// SECONDS_TEXT_SIZE_MAXIMUM is every decimal digit the seconds of one run spend. A monotonic
// moment states the host's uptime, which shared/sim/time bounds at a year, thus no run spans
// more seconds than a year holds however long the host has been up.
const SECONDS_TEXT_SIZE_MAXIMUM = len("31536000")

// ELAPSED_SIZE_MAXIMUM is every second digit one run can spend, and that same fraction.
const ELAPSED_SIZE_MAXIMUM = SECONDS_TEXT_SIZE_MAXIMUM + len(".") + 3

// Elapsed_Text is the elapsed seconds leaf of a flat JSON summary. It views stored bytes.
type Elapsed_Text string

// Elapsed_Text_Invariants bounds the leaf by the widest flat JSON object.
func Elapsed_Text_Invariants(value Elapsed_Text, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(len(value), ELAPSED_SIZE_MINIMUM, ELAPSED_SIZE_MAXIMUM).
    Ensure()
}

// Report is one diagnostic as flat JSON states it. Its order is the key order.
type Report struct {
  // Path names the file.
  Path Path_Text `json:"path"`
  // Line is where a refused token stands; absent without a position.
  Line Report_Line `json:"line"`
  // Column is where a refused token stands; absent without a position.
  Column Report_Column `json:"column"`
  // Tier is whether the diagnostic is auto-fixable.
  Tier Report_Tier `json:"tier"`
  // Rule names the rule the file breaks.
  Rule Rule_Text `json:"rule"`
  // Message says what to do about it.
  Message Message_Text `json:"message"`
}

// Report_Invariants composes one flat JSON diagnostic.
func Report_Invariants(value Report, namespace aver.Namespace) {
  Path_Text_Invariants(value.Path, namespace)
  Report_Line_Invariants(value.Line, namespace)
  Report_Column_Invariants(value.Column, namespace)
  Report_Tier_Invariants(value.Tier, namespace)
  Rule_Text_Invariants(value.Rule, namespace)
  Message_Text_Invariants(value.Message, namespace)
}

// Passed records that the summary store holds a run that found no diagnostic, rather than none.
type Passed bool

// Passed_Invariants proves both outcomes occur.
func Passed_Invariants(value Passed, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Sometimes(bool(value), "A run passed.").
    Ensure()
}

// Summary is a passed run as flat JSON states it. Its order is the key order.
type Summary struct {
  // Passed is true once a passed run wrote the summary; a run with a diagnostic writes none.
  Passed Passed `json:"passed"`
  // Elapsed_Seconds is text, since shared/encoding/flatjson holds no float.
  Elapsed_Seconds Elapsed_Text `json:"elapsed_sec"`
}

// Summary_Invariants composes one flat JSON summary.
func Summary_Invariants(value Summary, namespace aver.Namespace) {
  Passed_Invariants(value.Passed, namespace)
  Elapsed_Text_Invariants(value.Elapsed_Seconds, namespace)
}

// Json_Storage holds one flat JSON line and its newline.
type Json_Storage []byte

// Json_Storage_Invariants bounds flat JSON storage.
func Json_Storage_Invariants(value Json_Storage, _ aver.Namespace) {
  aver.Always(len(value) == JSON_STORAGE_SIZE, "A flat JSON store has complete length.")
  aver.Always(cap(value) == JSON_STORAGE_SIZE, "A flat JSON store cannot grow.")
}

// Line is the part of Line_Storage one write sends.
type Line []byte

// Line_Invariants keeps one write inside the storage it views. A write of no bytes stands where
// the run has nothing to say; a write that stands names a path or opens a program prefix, thus
// one byte and two each name a line no run writes.
func Line_Invariants(value Line, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Holed_Int(
    len(value),
    bytes.SLICE_SIZE_MINIMUM,
    WRITE_SIZE_MAXIMUM,
    LINE_HOLE_ONE,
    LINE_HOLE_TWO,
    LINE_HOLE_TWO,
    LINE_HOLE_TWO,
  ).
    Ensure()
}

// LINE_HOLE_ONE is the one byte no line a run writes spends.
const LINE_HOLE_ONE = 1

// LINE_HOLE_TWO is the two bytes no line a run writes spends.
const LINE_HOLE_TWO = 2

// Line_Storage holds the line being written.
type Line_Storage []byte

// Line_Storage_Invariants bounds line storage.
func Line_Storage_Invariants(value Line_Storage, _ aver.Namespace) {
  aver.Always(len(value) == LINE_SIZE_MAXIMUM, "A line store has complete length.")
  aver.Always(cap(value) == LINE_SIZE_MAXIMUM, "A line store cannot grow.")
}

// Directory_Paths holds the directories the walk has yet to open, one slot each.
type Directory_Paths []filepath.Slice

// Directory_Paths_Invariants bounds pending directories.
func Directory_Paths_Invariants(value Directory_Paths, _ aver.Namespace) {
  aver.Always(len(value) == DIRECTORY_COUNT_MAXIMUM, "A directory table has complete length.")
  aver.Always(cap(value) == DIRECTORY_COUNT_MAXIMUM, "A directory table cannot grow.")
}

// File_Bytes is a file as read, which may run one byte past requirements.SOURCE_SIZE_MAXIMUM:
// that byte is how a read proves the file oversized, thus it is no token.Source until Check says
// so.
type File_Bytes []byte

// File_Bytes_Invariants bounds a read file by its storage.
func File_Bytes_Invariants(value File_Bytes, _ aver.Namespace) {
  aver.Always(len(value) == SOURCE_STORAGE_SIZE, "A source store has complete length.")
  aver.Always(cap(value) == SOURCE_STORAGE_SIZE, "A source store cannot grow.")
}

// File_Count counts the Go file paths the walk collected.
type File_Count int

// File_Count_Invariants bounds collected paths.
func File_Count_Invariants(value File_Count, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(int(value), slices.COUNT_MINIMUM, FILE_COUNT_MAXIMUM).
    Ensure()
}

// SERVED_FILE_COUNT_MINIMUM is the fewest files a run serves one of, since a run that serves a
// file collected it first.
const SERVED_FILE_COUNT_MINIMUM = 1

// Served_File_Count is how many files a run serving one of them collected. A walk that collected
// none serves none, thus a count of none stands nowhere here.
type Served_File_Count int

// Served_File_Count_Invariants bounds the count by the path table.
func Served_File_Count_Invariants(value Served_File_Count, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(int(value), SERVED_FILE_COUNT_MINIMUM, FILE_COUNT_MAXIMUM).
    Ensure()
}

// SERVED_PACKAGE_COUNT_MINIMUM is the fewest packages a run serving one file holds, since the
// file it serves stands in one of them.
const SERVED_PACKAGE_COUNT_MINIMUM = 1

// Served_Package_Count is how many packages a run serving a file recorded. A file the run serves
// stands in a package, thus a count of none stands nowhere here.
type Served_Package_Count int

// Served_Package_Count_Invariants bounds the count by the package table.
func Served_Package_Count_Invariants(value Served_Package_Count, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(
    int(value),
    SERVED_PACKAGE_COUNT_MINIMUM,
    PACKAGE_COUNT_MAXIMUM,
  ).
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

// FILE_SLOT_MAXIMUM is the last slot of the path table, since a count of slots names one past
// the last of them.
const FILE_SLOT_MAXIMUM = FILE_COUNT_MAXIMUM - 1

// File_Slot is the slot of one file the run stands on. A file the run serves stands in the
// table, thus the count that names one past the last of them stands nowhere here.
type File_Slot int

// File_Slot_Invariants bounds the slot by the path table.
func File_Slot_Invariants(value File_Slot, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(int(value), slices.COUNT_MINIMUM, FILE_SLOT_MAXIMUM).
    Ensure()
}

// Directory_Count counts the directories the walk has yet to open.
type Directory_Count int

// Directory_Count_Invariants bounds pending directories.
func Directory_Count_Invariants(value Directory_Count, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(
    int(value), slices.COUNT_MINIMUM, DIRECTORY_COUNT_MAXIMUM,
  ).
    Ensure()
}

// Source_Count counts the bytes of the current file read so far.
type Source_Count int

// Source_Count_Invariants bounds read bytes by the source storage.
func Source_Count_Invariants(value Source_Count, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(
    int(value), slices.COUNT_MINIMUM, SOURCE_STORAGE_SIZE,
  ).
    Ensure()
}

// Unfilled_Count is the count of one source whose store holds a byte past it. A read stands on
// it only while the store has room, since a full store closes the file, thus the whole store
// stands nowhere here.
type Unfilled_Count Source_Count

// Unfilled_Count_Invariants bounds the count one byte short of the store.
func Unfilled_Count_Invariants(value Unfilled_Count, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(
    int(value),
    slices.COUNT_MINIMUM,
    requirements.SOURCE_SIZE_MAXIMUM,
  ).
    Ensure()
}

// Unfilled_Count_Handle is the caller-owned count one read adds to.
type Unfilled_Count_Handle *Unfilled_Count

// Unfilled_Count_Handle_Invariants composes a present count.
func Unfilled_Count_Handle_Invariants(value Unfilled_Count_Handle, namespace aver.Namespace) {
  if value == nil {
    return
  }
  Unfilled_Count_Invariants(*value, namespace)
}

// Boolean is one yes or no the root reads.
type Boolean bool

// Boolean_Invariants proves both answers occur.
func Boolean_Invariants(value Boolean, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Sometimes(bool(value), "The harness answer is yes.").
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
    Range_Int(int(value), int(PHASE_READY), int(PHASE_OPENED)).
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
    int(STAGE_HISTORY_LOG),
  ).
    Ensure()
}

// Directory_Pass is the stage one directory pass leaves the walk on: another pass over the same
// directory, or the close of it.
type Directory_Pass Stage

// Directory_Pass_Invariants admits the two stages one directory pass leaves the walk on.
func Directory_Pass_Invariants(value Directory_Pass, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_Int(
    int(value),
    int(STAGE_DIRECTORY_READ),
    int(STAGE_DIRECTORY_CLOSE),
  ).
    Ensure()
}

// Read_Pass is the stage one read of a file leaves the run on: another read of the same file, or
// the close of it.
type Read_Pass Stage

// Read_Pass_Invariants admits the two stages one read leaves the run on.
func Read_Pass_Invariants(value Read_Pass, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_Int(int(value), int(STAGE_READ), int(STAGE_CLOSE)).
    Ensure()
}

// File_Pass is the stage one pass over the file table leaves the run on: the open of the next
// file, or the first package once every file is served.
type File_Pass Stage

// File_Pass_Invariants admits the two stages one pass over the file table leaves the run on.
func File_Pass_Invariants(value File_Pass, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_Int(int(value), int(STAGE_OPEN), int(STAGE_PACKAGE)).
    Ensure()
}

// Walk_Pass is the stage one directory close leaves the run on: the open of the directory after
// it, the failure, or the pass that declares the names of the files it collected.
type Walk_Pass Stage

// Walk_Pass_Invariants admits the three stages one directory close leaves the run on.
func Walk_Pass_Invariants(value Walk_Pass, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_3_Int(
    int(value),
    int(STAGE_DIRECTORY_OPEN),
    int(STAGE_FAIL),
    int(STAGE_DECLARE),
  ).
    Ensure()
}

// Served_Pass is the stage one read the check pass did not open leaves the run on: the contract
// check that takes the file it read, or the declaration pass that reads the file after it.
type Served_Pass Stage

// Served_Pass_Invariants admits the two stages one such read leaves the run on.
func Served_Pass_Invariants(value Served_Pass, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_3_Int(
    int(value),
    int(STAGE_REQUIREMENTS),
    int(STAGE_FIX_OPEN),
    int(STAGE_DECLARED),
  ).
    Ensure()
}

// Report_Pass is the stage one written diagnostic leaves the run on: the contract check that
// owed it, the failure a write that took no byte states, or the file after the one it named.
type Report_Pass Stage

// Report_Pass_Invariants admits the three stages one written diagnostic leaves.
func Report_Pass_Invariants(value Report_Pass, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_4_Int(
    int(value),
    int(STAGE_REQUIREMENTS),
    int(STAGE_FAIL),
    int(STAGE_SERVE),
    int(STAGE_HISTORY_NEXT),
  ).
    Ensure()
}

// Contract_Pass is the stage one step of a contract check leaves the run on: the open of the
// file it asks for, the line of a finding, or the package after the one it finished.
type Contract_Pass Stage

// Contract_Pass_Invariants admits the stages one step of a contract check leaves.
func Contract_Pass_Invariants(value Contract_Pass, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_3_Int(
    int(value),
    int(STAGE_OPEN),
    int(STAGE_PACKAGED),
    int(STAGE_FINDING),
  ).
    Ensure()
}

// Package_Pass is the stage the start of one package check leaves the run on: the open of the
// contract it holds, the check of a contract already loaded, or the package stage itself where
// no package is left to check.
type Package_Pass Stage

// Package_Pass_Invariants admits the stages the start of one package check leaves.
func Package_Pass_Invariants(value Package_Pass, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_3_Int(
    int(value),
    int(STAGE_REQUIREMENTS),
    int(STAGE_OPEN),
    int(STAGE_PACKAGE),
  ).
    Ensure()
}

// Step_Pass is the stage one package step leaves the run on: the stages a package check opens,
// the pass line of a clean run, or the package stage a stopped run rests at.
type Step_Pass Stage

// Step_Pass_Invariants admits the stages one package step leaves the run on.
func Step_Pass_Invariants(value Step_Pass, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(int(value), int(STAGE_PASSED), int(STAGE_HISTORY_HEAD)).
    Ensure()
}

// Command_Pass is the stage one command line parse leaves the run on: the open of harness.json
// before the walk it named, the help it asked for, or the refusal of what a host typed.
type Command_Pass Stage

// Command_Pass_Invariants admits the stages one command line parse leaves the run on.
func Command_Pass_Invariants(value Command_Pass, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_3_Int(
    int(value),
    int(STAGE_HELP),
    int(STAGE_OPEN),
    int(STAGE_FAIL),
  ).
    Ensure()
}

// Record_Pass is the stage one recorded command line leaves the run on: the open of harness.json
// before the walk of the directory it names, or the failed walk of one no walk can stand on.
type Record_Pass Stage

// Record_Pass_Invariants admits the stages one record leaves the run on.
func Record_Pass_Invariants(value Record_Pass, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_Int(int(value), int(STAGE_OPEN), int(STAGE_FAIL)).
    Ensure()
}

// CLOSE_PASS_CONFIGURED is the close of a configuration that decodes, which the walk follows.
const CLOSE_PASS_CONFIGURED Close_Pass = 5

// CLOSE_PASS_FAILED is the close a step of the run failed at, which the run cannot finish.
const CLOSE_PASS_FAILED Close_Pass = 0

// CLOSE_PASS_SERVED is the close of a file the run is done with, whatever it found in it.
const CLOSE_PASS_SERVED Close_Pass = 1

// CLOSE_PASS_FIX is the close of a file whose canonical form the run writes back.
const CLOSE_PASS_FIX Close_Pass = 2

// CLOSE_PASS_REPORT is the close of a file whose diagnostic the run wrote.
const CLOSE_PASS_REPORT Close_Pass = 3

// CLOSE_PASS_CONTRACT is the close of a file the run read for a package contract rather than
// for its own bytes, which the contract check takes from here.
const CLOSE_PASS_CONTRACT Close_Pass = 4

// Close_Pass is what one close leaves the run to do. The close names the five, and the caller
// alone turns one of them into the stage the run stands on next.
type Close_Pass int

// Close_Pass_Invariants admits the five one close leaves.
func Close_Pass_Invariants(value Close_Pass, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(
    int(value),
    int(CLOSE_PASS_FAILED),
    int(CLOSE_PASS_CONFIGURED),
  ).
    Ensure()
}

// FIX_SIZE_MINIMUM is the shortest fix one file takes: a contract of one empty heading, with the
// blank line before it and the newline that closes it. A Go file's canonical form is longer, since
// it holds a package clause.
const FIX_SIZE_MINIMUM = len("\n# \n")

// Written_Fix is the fix one file takes: the whole canonical form of it. A fix of no bytes would
// leave the file empty, thus the shortest form one print writes stands here rather than none.
type Written_Fix Fix

// Written_Fix_Invariants bounds the fix one file takes by the forms one print writes.
func Written_Fix_Invariants(value Written_Fix, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(len(value), FIX_SIZE_MINIMUM, FORM_STORAGE_SIZE).
    Ensure()
}

// Fix_Pass is the stage one step of a fix leaves the run on: the rest of its write, its close,
// its failure, the file after the Go file it fixed, or the check of the contract it fixed.
type Fix_Pass Stage

// Fix_Pass_Invariants bounds the stages a fix step leaves.
func Fix_Pass_Invariants(value Fix_Pass, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(int(value), int(STAGE_REQUIREMENTS), int(STAGE_FIX_CLOSE)).
    Ensure()
}

// Fixed_Reading is what a fixed file serves: a Go file, or a contract.
type Fixed_Reading Reading

// Fixed_Reading_Invariants admits the two files a fix writes.
func Fixed_Reading_Invariants(value Fixed_Reading, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_Int(int(value), int(READING_SOURCE), int(READING_MARKDOWN)).
    Ensure()
}

// Open_Pass is the stage one opened file leaves the run on: the read of its bytes, or the
// failure a file the run cannot open states.
type Open_Pass Stage

// Open_Pass_Invariants admits the two stages one opened file leaves.
func Open_Pass_Invariants(value Open_Pass, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_Int(int(value), int(STAGE_READ), int(STAGE_FAIL)).
    Ensure()
}

// Directory_Open_Pass is the stage one opened directory leaves the run on: the read of the
// entries it holds, or the failure a directory the walk cannot open states.
type Directory_Open_Pass Stage

// Directory_Open_Pass_Invariants admits the two stages one opened directory leaves.
func Directory_Open_Pass_Invariants(value Directory_Open_Pass, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_Int(int(value), int(STAGE_DIRECTORY_READ), int(STAGE_FAIL)).
    Ensure()
}

// Reported_Tier is the tier one written diagnostic stands in. A run writes the diagnostic of a
// file that broke a rule, thus the tier naming no break stands nowhere here.
type Reported_Tier Tier

// Reported_Tier_Invariants admits the two tiers one written diagnostic stands in.
func Reported_Tier_Invariants(value Reported_Tier, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_Int(int(value), int(TIER_ONE), int(TIER_TWO)).
    Ensure()
}

// Reported_Rule is the rule one written diagnostic names. A run writes no line of a file that
// broke nothing, thus the rule naming no break stands nowhere here.
type Reported_Rule Rule

// Reported_Rule_Invariants admits each rule one written diagnostic names.
func Reported_Rule_Invariants(value Reported_Rule, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(int(value), int(RULE_UNFORMATTED), int(RULE_CONFLICTED)).
    Ensure()
}

// Reported is one diagnostic a run writes: the tier it stands in, the rule it names, and where
// the refused token stands. It carries no fix, because a line states none of one.
type Reported struct {
  // Tier is the tier the written diagnostic stands in.
  Tier Reported_Tier
  // Rule is the rule the written diagnostic names.
  Rule Reported_Rule
  // Failure is the parser's refusal; FAILURE_NONE unless RULE_REFUSED.
  Failure Refusal
  // Line is where the refused token stands; absent unless RULE_REFUSED.
  Line Diagnostic_Line
  // Column is where the refused token stands; absent unless RULE_REFUSED.
  Column Diagnostic_Column
}

// Reported_Invariants composes one written diagnostic.
func Reported_Invariants(value Reported, namespace aver.Namespace) {
  Reported_Tier_Invariants(value.Tier, namespace)
  Reported_Rule_Invariants(value.Rule, namespace)
  Refusal_Invariants(value.Failure, namespace)
  Diagnostic_Line_Invariants(value.Line, namespace)
  Diagnostic_Column_Invariants(value.Column, namespace)
}

// Fix_Stage is the stage one fix submission stands at. A fix opens the file, writes the rest of
// the form, and closes it, thus three stages stand here and no other.
type Fix_Stage Stage

// Fix_Stage_Invariants admits the three stages one fix stands at.
func Fix_Stage_Invariants(value Fix_Stage, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_3_Int(
    int(value),
    int(STAGE_FIX_OPEN),
    int(STAGE_FIX_WRITE),
    int(STAGE_FIX_CLOSE),
  ).
    Ensure()
}

// Submitted_Stage is the stage one submission stands at. A stage that queues no operation of
// its own stands nowhere here, thus the stages that do bound this from the first of them to the
// last.
type Submitted_Stage Stage

// Submitted_Stage_Invariants bounds the stage one submission stands at.
func Submitted_Stage_Invariants(value Submitted_Stage, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(
    int(value),
    int(STAGE_DIRECTORY_OPEN),
    int(STAGE_HISTORY_LOG),
  ).
    Ensure()
}

// Stage_Handle is the caller-owned stage the run stands on. It stands apart from the loop,
// because a helper that reads no stage states none of the stages it never stands at.
type Stage_Handle *Stage

// Stage_Handle_Invariants states the stage a present handle names.
func Stage_Handle_Invariants(value Stage_Handle, namespace aver.Namespace) {
  if value == nil {
    return
  }
  Stage_Invariants(*value, namespace)
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
  Failure Refusal
  // Line is where the refused token stands; absent unless RULE_REFUSED.
  Line Diagnostic_Line
  // Column is where the refused token stands; absent unless RULE_REFUSED.
  Column Diagnostic_Column
}

// Diagnostic_Invariants composes one diagnostic.
func Diagnostic_Invariants(value Diagnostic, namespace aver.Namespace) {
  Tier_Invariants(value.Tier, namespace)
  Rule_Invariants(value.Rule, namespace)
  Fix_Invariants(value.Fix, namespace)
  Refusal_Invariants(value.Failure, namespace)
  Diagnostic_Line_Invariants(value.Line, namespace)
  Diagnostic_Column_Invariants(value.Column, namespace)
  // A tier one diagnostic carries the fix and a tier two carries none, thus the tier and
  // the fix state the same thing and one standing without the other names a diagnostic no
  // run can write.
  aver.Always(
    (value.Tier == TIER_ONE) == (len(value.Fix) > bytes.SLICE_SIZE_MINIMUM),
    "A fix stands with tier one alone.",
  )
  // A diagnostic that breaks no rule stands at no tier, thus neither one states a finding
  // the other leaves out.
  aver.Always(
    (value.Rule == RULE_NONE) == (value.Tier == TIER_NONE),
    "A diagnostic breaking no rule stands at no tier.",
  )
}

// Argument_Declarations is the run's own argument declaration table, fixed at its whole size.
type Argument_Declarations cli.Arguments

// Argument_Declarations_Invariants fixes the store.
func Argument_Declarations_Invariants(value Argument_Declarations, _ aver.Namespace) {
  aver.Always(
    len(value) == ARGUMENT_DECLARATION_COUNT,
    "An argument declaration table has complete length.",
  )
  aver.Always(
    cap(value) == ARGUMENT_DECLARATION_COUNT,
    "An argument declaration table cannot grow.",
  )
}

// Flag_Declarations is the run's own flag declaration table, fixed by the root at its whole size.
type Flag_Declarations cli.Flags

// Flag_Declarations_Invariants fixes the store.
func Flag_Declarations_Invariants(value Flag_Declarations, _ aver.Namespace) {
  aver.Always(
    len(value) == FLAG_DECLARATION_COUNT,
    "A flag declaration table has complete length.",
  )
  aver.Always(cap(value) == FLAG_DECLARATION_COUNT, "A flag declaration table cannot grow.")
}

// Parsed_Arguments is the run's own parsed argument table, fixed by the root at its whole size.
type Parsed_Arguments cli.Command_Argument_Storage

// Parsed_Arguments_Invariants fixes the store.
func Parsed_Arguments_Invariants(value Parsed_Arguments, _ aver.Namespace) {
  aver.Always(
    len(value) == ARGUMENT_DECLARATION_COUNT,
    "A parsed argument table has complete length.",
  )
  aver.Always(
    cap(value) == ARGUMENT_DECLARATION_COUNT, "A parsed argument table cannot grow.",
  )
}

// Parsed_Flags is the run's own parsed flag table, fixed by the root at its whole size.
type Parsed_Flags cli.Command_Flag_Storage

// Parsed_Flags_Invariants fixes the store.
func Parsed_Flags_Invariants(value Parsed_Flags, _ aver.Namespace) {
  aver.Always(
    len(value) == FLAG_DECLARATION_COUNT, "A parsed flag table has complete length.",
  )
  aver.Always(cap(value) == FLAG_DECLARATION_COUNT, "A parsed flag table cannot grow.")
}

// Filled_Marks is the run's own filled option table, fixed by the root at its whole size.
type Filled_Marks cli.Filled_Options

// Filled_Marks_Invariants fixes the store.
func Filled_Marks_Invariants(value Filled_Marks, _ aver.Namespace) {
  aver.Always(len(value) == FILLED_COUNT, "A filled option table has complete length.")
  aver.Always(cap(value) == FILLED_COUNT, "A filled option table cannot grow.")
}

// Positional_Tokens is the run's own positional table, fixed by the root at its whole size.
type Positional_Tokens cli.Positionals

// Positional_Tokens_Invariants fixes the store.
func Positional_Tokens_Invariants(value Positional_Tokens, _ aver.Namespace) {
  aver.Always(len(value) == TOKEN_COUNT_MAXIMUM, "A positional table has complete length.")
  aver.Always(cap(value) == TOKEN_COUNT_MAXIMUM, "A positional table cannot grow.")
}

// Named_Tokens is the run's own named variadic token table, fixed by the root at its whole size.
type Named_Tokens cli.Named_Variadic_Tokens

// Named_Tokens_Invariants fixes the store.
func Named_Tokens_Invariants(value Named_Tokens, _ aver.Namespace) {
  aver.Always(
    len(value) == TOKEN_COUNT_MAXIMUM,
    "A named variadic token table has complete length.",
  )
  aver.Always(cap(value) == TOKEN_COUNT_MAXIMUM, "A named variadic token table cannot grow.")
}

// Help_Bytes is the store the help renders into, fixed by the root at its whole size.
type Help_Bytes strings.Bytes

// Help_Bytes_Invariants fixes the store.
func Help_Bytes_Invariants(value Help_Bytes, _ aver.Namespace) {
  aver.Always(len(value) == strings.TEXT_SIZE_MAXIMUM, "A help store has complete length.")
  aver.Always(cap(value) == strings.TEXT_SIZE_MAXIMUM, "A help store cannot grow.")
}

// Failure_Bytes is the run's own parse failure store, fixed by the root at its whole size.
type Failure_Bytes cli.Parse_Failure_Storage

// Failure_Bytes_Invariants fixes the store.
func Failure_Bytes_Invariants(value Failure_Bytes, _ aver.Namespace) {
  aver.Always(
    len(value) == cli.FAILURE_SIZE_MAXIMUM,
    "A parse failure store has complete length.",
  )
  aver.Always(cap(value) == cli.FAILURE_SIZE_MAXIMUM, "A parse failure store cannot grow.")
}

// Path_Table is the run's own path table, fixed by the root at its whole size.
type Path_Table filepath.Path_Storage

// Path_Table_Invariants fixes the store.
func Path_Table_Invariants(value Path_Table, _ aver.Namespace) {
  aver.Always(len(value) == FILE_COUNT_MAXIMUM, "A path table has complete length.")
  aver.Always(cap(value) == FILE_COUNT_MAXIMUM, "A path table cannot grow.")
}

// Entry_Table is the run's own entry table, fixed by the root at its whole size.
type Entry_Table filepath.Directory_Entries

// Entry_Table_Invariants fixes the store.
func Entry_Table_Invariants(value Entry_Table, _ aver.Namespace) {
  aver.Always(len(value) == ENTRY_COUNT_MAXIMUM, "An entry table has complete length.")
  aver.Always(cap(value) == ENTRY_COUNT_MAXIMUM, "An entry table cannot grow.")
}

// Entry_Buffer is the run's own directory buffer, fixed by the root at its whole size.
type Entry_Buffer filepath.Directory_Buffer

// Entry_Buffer_Invariants fixes the store.
func Entry_Buffer_Invariants(value Entry_Buffer, _ aver.Namespace) {
  aver.Always(
    len(value) == nbio.DIRECTORY_BUFFER_SIZE_MAXIMUM,
    "A directory buffer has complete length.",
  )
  aver.Always(
    cap(value) == nbio.DIRECTORY_BUFFER_SIZE_MAXIMUM, "A directory buffer cannot grow.",
  )
}

// Form_Bytes is the run's own form store, fixed by the root at its whole size.
type Form_Bytes printer.Form

// Form_Bytes_Invariants fixes the store.
func Form_Bytes_Invariants(value Form_Bytes, _ aver.Namespace) {
  aver.Always(len(value) == FORM_STORAGE_SIZE, "A form store has complete length.")
  aver.Always(cap(value) == FORM_STORAGE_SIZE, "A form store cannot grow.")
}

// Report_Handle is the caller-owned report one write states.
type Report_Handle *Report

// Report_Handle_Invariants composes a present report.
func Report_Handle_Invariants(value Report_Handle, namespace aver.Namespace) {
  if value == nil {
    return
  }
  Report_Invariants(*value, namespace)
}

// Summary_Handle is the caller-owned summary one write states.
type Summary_Handle *Summary

// Summary_Handle_Invariants composes a present summary.
func Summary_Handle_Invariants(value Summary_Handle, namespace aver.Namespace) {
  if value == nil {
    return
  }
  Summary_Invariants(*value, namespace)
}

// Current_Path is the directory the walk has open. The run fixes it at one whole path.
type Current_Path filepath.Slice

// Current_Path_Invariants fixes the store.
func Current_Path_Invariants(value Current_Path, namespace aver.Namespace) {
  aver.Always(cap(value) == filepath.PATH_SIZE_MAXIMUM, "A current path cannot grow.")
  aver.Tree(value, namespace).
    Range_Int(
    len(value),
    bytes.SLICE_SIZE_MINIMUM,
    filepath.PATH_SIZE_MAXIMUM,
  ).
    Ensure()
}

// PACKAGE_PATH_SIZE_MAXIMUM is the widest directory a package stands in. A package owes a
// contract, thus its directory leaves room for the contract name beside the path it holds, and
// a walk fails on a directory that leaves none.
const PACKAGE_PATH_SIZE_MAXIMUM = filepath.PATH_SIZE_MAXIMUM -
  len("/") - CONTRACT_NAME_SIZE_MAXIMUM

// Package_Path is the directory of one package, fixed at one whole path.
type Package_Path filepath.Slice

// Package_Path_Invariants fixes the store. A path names a directory the walk opened, thus it
// holds a byte at least.
func Package_Path_Invariants(value Package_Path, namespace aver.Namespace) {
  aver.Always(cap(value) == filepath.PATH_SIZE_MAXIMUM, "A package path cannot grow.")
  aver.Tree(value, namespace).
    Range_Int(len(value), PATH_SIZE_MINIMUM, PACKAGE_PATH_SIZE_MAXIMUM).
    Ensure()
}

// Reports is what one write states as flat JSON: the diagnostic of the last finding and the
// summary of a passed run. Only the writes that state them take it, so a helper that never writes
// one owes no leaf of it.
type Reports struct {
  // Report is the flat JSON form of the last diagnostic.
  Report Report
  // Summary is the flat JSON form of a passed run.
  Summary Summary
}

// Reports_Invariants composes what one write states.
func Reports_Invariants(value Reports, namespace aver.Namespace) {
  Report_Invariants(value.Report, namespace)
  Summary_Invariants(value.Summary, namespace)
}

// Failure_Path is the path one failed step names. The caller states which path failed, since a
// walk names the directory it had open while every later step names the file it was reading.
type Failure_Path filepath.Slice

// Failure_Path_Invariants bounds the named path.
func Failure_Path_Invariants(value Failure_Path, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(
    len(value),
    bytes.SLICE_SIZE_MINIMUM,
    filepath.PATH_SIZE_MAXIMUM,
  ).
    Ensure()
}

// Directory is the directory the walk has open. It stands apart from the run's stores, because a
// pending slot would be overwritten by the children one pass pushes.
type Directory struct {
  // Current holds the path of the open directory.
  Current Current_Path
  // Directory_Count counts pending directories in Memory.Directories.
  Directory_Count Directory_Count
}

// Directory_Invariants composes the open directory.
func Directory_Invariants(value Directory, namespace aver.Namespace) {
  Current_Path_Invariants(value.Current, namespace)
  Directory_Count_Invariants(value.Directory_Count, namespace)
}

// WALKED_PATH_SIZE_MINIMUM is the shortest path the walk stands on, since a directory names
// itself with one byte at least.
const WALKED_PATH_SIZE_MINIMUM = filepath.NONEMPTY_SIZE_MINIMUM

// Walked_Path is the path of the directory the walk stands on. A pass stands on a directory a
// command line or a parent named, thus a path of no bytes stands nowhere here.
type Walked_Path []byte

// Walked_Path_Invariants bounds the path the walk stands on by the store that holds it.
func Walked_Path_Invariants(value Walked_Path, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(
    len(value),
    WALKED_PATH_SIZE_MINIMUM,
    filepath.PATH_SIZE_MAXIMUM,
  ).
    Ensure()
}

// PARENT_PATH_SIZE_MAXIMUM is the widest path a child fits under: a separator and one name byte.
const PARENT_PATH_SIZE_MAXIMUM = filepath.PATH_SIZE_MAXIMUM - len("/a")

// Parent_Path is the path of a directory the walk records a child of. The walk bars a child
// whose path fits no path slot, thus the parent leaves room for a separator and one name byte.
type Parent_Path []byte

// Parent_Path_Invariants bounds the parent by the widest path less the child it holds.
func Parent_Path_Invariants(value Parent_Path, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(
    len(value),
    WALKED_PATH_SIZE_MINIMUM,
    PARENT_PATH_SIZE_MAXIMUM,
  ).
    Ensure()
}

// Directory_Count_Handle is the caller-owned count of directories the walk has yet to open. It
// stands apart from the directory, because a helper that queues one states no path of its own.
type Directory_Count_Handle *Directory_Count

// Directory_Count_Handle_Invariants bounds the count behind its handle.
func Directory_Count_Handle_Invariants(
  value Directory_Count_Handle,
  namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Directory_Count_Invariants(*value, namespace)
}

// Directory_Handle is a caller-owned open directory.
type Directory_Handle *Directory

// Directory_Handle_Invariants composes the open directory behind its handle.
func Directory_Handle_Invariants(value Directory_Handle, namespace aver.Namespace) {
  if value == nil {
    return
  }
  Directory_Invariants(*value, namespace)
}

// Command is every store one command line parse fills. It stands apart from the run's stores,
// because a step past the parse states nothing of the tokens it read.
type Command struct {
  // Declared_Arguments holds the one positional declaration.
  Declared_Arguments Argument_Declarations
  // Declared_Flags holds the -json and -fix declarations.
  Declared_Flags Flag_Declarations
  // Command_Arguments receives the parsed positional.
  Command_Arguments Parsed_Arguments
  // Command_Flags receives the parsed flags.
  Command_Flags Parsed_Flags
  // Filled marks each option the command line set.
  Filled Filled_Marks
  // Positionals receives the command line's positional tokens.
  Positionals Positional_Tokens
  // Slice_Named receives named variadic tokens; the program declares none, but the parser
  // sizes its token pass by it.
  Slice_Named Named_Tokens
  // Failure receives the parser's rendered error.
  Failure Failure_Bytes
  // Help holds the rendered help.
  Help Help_Bytes
}

// Command_Invariants composes what one parse stands on.
func Command_Invariants(value Command, namespace aver.Namespace) {
  Argument_Declarations_Invariants(value.Declared_Arguments, namespace)
  Flag_Declarations_Invariants(value.Declared_Flags, namespace)
  Parsed_Arguments_Invariants(value.Command_Arguments, namespace)
  Parsed_Flags_Invariants(value.Command_Flags, namespace)
  Filled_Marks_Invariants(value.Filled, namespace)
  Positional_Tokens_Invariants(value.Positionals, namespace)
  Named_Tokens_Invariants(value.Slice_Named, namespace)
  Failure_Bytes_Invariants(value.Failure, namespace)
  Help_Bytes_Invariants(value.Help, namespace)
}

// Formatter_Stores is what one formatter keeps from one call to the next: the places and the name
// table. The source one declaration reads names from stands only inside that call, thus the run
// holds none and hands each call a formatter over these stores.
type Formatter_Stores struct {
  // Places holds the import sort cursors and the table counts.
  Places format.Place_Storage
  // Table holds the names every format renames by.
  Table format.Table_Fields
}

// Formatter_Stores_Invariants composes the kept stores.
func Formatter_Stores_Invariants(value Formatter_Stores, namespace aver.Namespace) {
  format.Place_Storage_Invariants(value.Places, namespace)
  format.Table_Fields_Invariants(value.Table, namespace)
}

// Format is what rewrites one source into its canonical form. It stands apart from the run's
// stores, because a step that names a path states nothing of the printer's stacks.
type Format struct {
  // Formatter holds the import sort cursors.
  Formatter Formatter_Stores
  // Printer holds the printer stacks.
  Printer printer.Printer_Handle
}

// Format_Invariants composes what one rewrite stands on.
func Format_Invariants(value Format, namespace aver.Namespace) {
  Formatter_Stores_Invariants(value.Formatter, namespace)
  printer.Printer_Handle_Invariants(value.Printer, namespace)
}

// Memory is the store of the tree one run walks and of what it writes about it: the paths it
// collected, the packages they stand in, the source it reads, and the lines it writes. The stores
// one pass alone fills stand in the part that pass takes, thus a step that never parses a command
// line or rewrites a source states neither.
type Memory struct {
  // Form holds the canonical form.
  Form Form_Bytes
  // Paths holds every Go file path the walk collects, one slot each.
  Paths Path_Table
  // Directories holds the directories the walk has yet to open.
  Directories Directory_Paths
  // Entries receives one directory pass.
  Entries Entry_Table
  // Directory_Buffer receives backend records for one pass.
  Directory_Buffer Entry_Buffer
  // Source holds the file being read.
  Source File_Bytes
  // Line holds the text line being written.
  Line Line_Storage
  // Json holds the flat JSON line being written.
  Json Json_Storage
  // Packages holds every package the walk found.
  Packages Packages
  // File_Packages names each Go file's package, slot for slot with Paths.
  File_Packages Package_Indices
  // Requirements checks each package's contract.
  Requirements requirements.Checker_Handle
  // History_Output holds what one git command wrote.
  History_Output History_Output
  // History_Errors holds what one git command wrote to its standard error.
  History_Errors History_Errors
  // History_Name holds the name one commit finding stands at.
  History_Name History_Name
  // Configuration holds the decoded harness.json.
  Configuration configuration.Configuration_Handle
  // Notice holds the message of one finding of harness.json.
  Notice Notice_Store
  // Orphans holds each symbolic link the walk found whose target resolves to nothing.
  Orphans Orphan_Paths
  // Link holds the path of one child the walk asks the host about.
  Link Link_Path
  // Resolved holds the path one link resolution stands on.
  Resolved Resolved_Path
  // Target holds the target one symbolic link names.
  Target Link_Target
}

// Memory_Invariants states that every store the root allocated stands at its whole size. A store
// another package declares belongs to that package's chain, which its own simulation covers, thus
// this chain states the one property the run depends on: the store is there and cannot grow.
func Memory_Invariants(value Memory, namespace aver.Namespace) {
  Form_Bytes_Invariants(value.Form, namespace)
  Path_Table_Invariants(value.Paths, namespace)
  Directory_Paths_Invariants(value.Directories, namespace)
  Entry_Table_Invariants(value.Entries, namespace)
  Entry_Buffer_Invariants(value.Directory_Buffer, namespace)
  File_Bytes_Invariants(value.Source, namespace)
  Line_Storage_Invariants(value.Line, namespace)
  Json_Storage_Invariants(value.Json, namespace)
  Packages_Invariants(value.Packages, namespace)
  Package_Indices_Invariants(value.File_Packages, namespace)
  requirements.Checker_Handle_Invariants(value.Requirements, namespace)
  History_Output_Invariants(value.History_Output, namespace)
  History_Errors_Invariants(value.History_Errors, namespace)
  History_Name_Invariants(value.History_Name, namespace)
  configuration.Configuration_Handle_Invariants(value.Configuration, namespace)
  Notice_Store_Invariants(value.Notice, namespace)
  Orphan_Paths_Invariants(value.Orphans, namespace)
  Link_Path_Invariants(value.Link, namespace)
  Resolved_Path_Invariants(value.Resolved, namespace)
  Link_Target_Invariants(value.Target, namespace)
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

// Driven_Phase is the phase one drive pass reads. A rearm submits every stage it is ready to
// submit before the drive opens, thus the stage it has yet to submit stands nowhere here.
type Driven_Phase Phase

// Driven_Phase_Invariants admits the three phases one drive pass reads.
func Driven_Phase_Invariants(value Driven_Phase, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_3_Int(
    int(value),
    int(PHASE_SUBMITTED),
    int(PHASE_RETIRED),
    int(PHASE_STOPPED),
  ).
    Ensure()
}

// Pass_Phase is the phase one pump pass opens on. A drive pass stands until no operation is in
// flight, thus the phase of one that is stands nowhere here, and a rearm leaves no stage ready.
type Pass_Phase Phase

// Pass_Phase_Invariants admits the three phases one pump pass opens on.
func Pass_Phase_Invariants(value Pass_Phase, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_3_Int(
    int(value),
    int(PHASE_RETIRED),
    int(PHASE_STOPPED),
    int(PHASE_OPENED),
  ).
    Ensure()
}

// Work_Queued_Procedure reports whether a callback recorded work only Rearm can act on, or the run
// stopped: either way a pump pass is over.
type Work_Queued_Procedure func(phase Driven_Phase) (queued Boolean)

// Stopped_Procedure reports whether the run is over and Status is final.
type Stopped_Procedure func(phase Pass_Phase) (stopped Boolean)

// Loop is where one run stands in the operating system loop: the operation in flight, the
// surface it goes through, the descriptors it holds, and the step it is on.
type Loop struct {
  // Completion stays first so the static callback recovers the runner from it alone.
  Completion nbio.Completion
  // IO is the injected loop surface every operation goes through.
  IO nbio.IO
  // Directory is the open descriptor of Memory.Current.
  Directory nbio.File
  // File is the open descriptor of the current file.
  File nbio.File
}

// Loop_Invariants composes where the run stands.
func Loop_Invariants(value Loop, namespace aver.Namespace) {
  nbio.IO_Invariants(value.IO, namespace)
}

// Progress is where one stage stands, the step a failure names, and the code the run exits with.
// Only the pump and the steps that end a run take it, so a helper that never moves it states no
// phase it cannot reach.
type Progress struct {
  // Phase is where the stage stands.
  Phase Phase
}

// Progress_Invariants composes where one stage stands.
func Progress_Invariants(value Progress, namespace aver.Namespace) {
  Phase_Invariants(value.Phase, namespace)
}

// Outcome is what the run will exit as, and the step it stands on while it gets there. It stands
// apart from the phase, because a step that names a failure states nothing of the loop's turn.
type Outcome struct {
  // Status is the exit code, final once Phase is PHASE_STOPPED.
  Status Status_Code
}

// Outcome_Invariants composes what the run will exit as.
func Outcome_Invariants(value Outcome, namespace aver.Namespace) {
  Status_Code_Invariants(value.Status, namespace)
}

// Outcome_Handle is a caller-owned outcome.
type Outcome_Handle *Outcome

// Outcome_Handle_Invariants composes a present outcome.
func Outcome_Handle_Invariants(value Outcome_Handle, namespace aver.Namespace) {
  if value == nil {
    return
  }
  Outcome_Invariants(*value, namespace)
}

// Progress_Handle is caller-owned run progress.
type Progress_Handle *Progress

// Progress_Handle_Invariants composes present run progress.
func Progress_Handle_Invariants(value Progress_Handle, namespace aver.Namespace) {
  if value == nil {
    return
  }
  Progress_Invariants(*value, namespace)
}

// Loop_Handle is caller-owned loop position.
type Loop_Handle *Loop

// Loop_Handle_Invariants composes a present loop position.
func Loop_Handle_Invariants(value Loop_Handle, namespace aver.Namespace) {
  if value == nil {
    return
  }
  Loop_Invariants(*value, namespace)
}

// Walk is the pass over the tree: what it collected.
type Walk struct {
  // Package_Count counts the packages in Memory.Packages.
  Package_Count Package_Count
  // Orphan_Count counts the links in Memory.Orphans.
  Orphan_Count Orphan_Count
}

// Walk_Invariants composes the tree pass.
func Walk_Invariants(value Walk, namespace aver.Namespace) {
  Package_Count_Invariants(value.Package_Count, namespace)
  Orphan_Count_Invariants(value.Orphan_Count, namespace)
}

// Orphan_Count counts the symbolic links the walk found whose target resolves to nothing.
type Orphan_Count int

// Orphan_Count_Invariants bounds the count by the orphan table.
func Orphan_Count_Invariants(value Orphan_Count, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(int(value), slices.COUNT_MINIMUM, ORPHAN_COUNT_MAXIMUM).
    Ensure()
}

// Orphan_Index is the next orphan the run states, or the count once it stated each.
type Orphan_Index int

// Orphan_Index_Invariants bounds the index by the orphan table.
func Orphan_Index_Invariants(value Orphan_Index, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(int(value), slices.COUNT_MINIMUM, ORPHAN_COUNT_MAXIMUM).
    Ensure()
}

// Orphan_Cursor is the orphan the run states next. It stands apart from the pass, because the
// walk that collects orphans states none of them, thus a step of that walk states no cursor it
// could move.
type Orphan_Cursor struct {
  // Orphan_Index is the next link of Memory.Orphans the run states.
  Orphan_Index Orphan_Index
}

// Orphan_Cursor_Invariants composes the orphan cursor.
func Orphan_Cursor_Invariants(value Orphan_Cursor, namespace aver.Namespace) {
  Orphan_Index_Invariants(value.Orphan_Index, namespace)
}

// Orphan_Cursor_Handle is a caller-owned orphan cursor.
type Orphan_Cursor_Handle *Orphan_Cursor

// Orphan_Cursor_Handle_Invariants composes the orphan cursor behind its handle.
func Orphan_Cursor_Handle_Invariants(value Orphan_Cursor_Handle, namespace aver.Namespace) {
  if value == nil {
    return
  }
  Orphan_Cursor_Invariants(*value, namespace)
}

// Orphan_Count_Handle is the count of a caller-owned pass.
type Orphan_Count_Handle *Orphan_Count

// Orphan_Count_Handle_Invariants composes the count behind its handle.
func Orphan_Count_Handle_Invariants(value Orphan_Count_Handle, namespace aver.Namespace) {
  if value == nil {
    return
  }
  Orphan_Count_Invariants(*value, namespace)
}

// ENTRY_PATH_SIZE_MINIMUM is the shortest path of a child: one directory byte, the separator,
// and a one-byte name.
const ENTRY_PATH_SIZE_MINIMUM = WALKED_PATH_SIZE_MINIMUM + len("/") + len("a")

// Entry_Path is the path of one child the walk found.
type Entry_Path []byte

// Entry_Path_Invariants bounds the path by the widest one a path holds.
func Entry_Path_Invariants(value Entry_Path, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(
    len(value),
    ENTRY_PATH_SIZE_MINIMUM,
    filepath.PATH_SIZE_MAXIMUM,
  ).
    Ensure()
}

// Orphan_Paths holds the path of each symbolic link the walk found whose target resolves to
// nothing, one slot each.
type Orphan_Paths []filepath.Slice

// Orphan_Paths_Invariants fixes the store.
func Orphan_Paths_Invariants(value Orphan_Paths, _ aver.Namespace) {
  aver.Always(len(value) == ORPHAN_COUNT_MAXIMUM, "An orphan table has complete length.")
  aver.Always(cap(value) == ORPHAN_COUNT_MAXIMUM, "An orphan table cannot grow.")
}

// Link_Path holds the path of one child the walk asks the host about.
type Link_Path []byte

// Link_Path_Invariants fixes the store.
func Link_Path_Invariants(value Link_Path, _ aver.Namespace) {
  aver.Always(len(value) == filepath.PATH_SIZE_MAXIMUM, "A link path store has complete length.")
  aver.Always(cap(value) == filepath.PATH_SIZE_MAXIMUM, "A link path store cannot grow.")
}

// Resolved_Path holds the path one link resolution stands on.
type Resolved_Path []byte

// Resolved_Path_Invariants fixes the store.
func Resolved_Path_Invariants(value Resolved_Path, _ aver.Namespace) {
  aver.Always(
    len(value) == filepath.PATH_SIZE_MAXIMUM,
    "A resolved path store has complete length.",
  )
  aver.Always(cap(value) == filepath.PATH_SIZE_MAXIMUM, "A resolved path store cannot grow.")
}

// Link_Target holds the target one symbolic link names.
type Link_Target []byte

// Link_Target_Invariants fixes the store.
func Link_Target_Invariants(value Link_Target, _ aver.Namespace) {
  aver.Always(len(value) == LINK_TARGET_SIZE, "A link target store has complete length.")
  aver.Always(cap(value) == LINK_TARGET_SIZE, "A link target store cannot grow.")
}

// Walk_Handle is a caller-owned tree pass.
type Walk_Handle *Walk

// Walk_Handle_Invariants composes the tree pass behind its handle.
func Walk_Handle_Invariants(value Walk_Handle, namespace aver.Namespace) {
  if value == nil {
    return
  }
  Walk_Invariants(*value, namespace)
}

// Package_Cursor is the package whose contract the run checks. It stands apart from the pass,
// because the walk that collects packages checks none of them, thus a step of that walk states
// no cursor it could move.
type Package_Cursor struct {
  // Package_Index is the package whose contract the run checks.
  Package_Index Package_Index
}

// Package_Cursor_Invariants composes the package cursor.
func Package_Cursor_Invariants(value Package_Cursor, namespace aver.Namespace) {
  Package_Index_Invariants(value.Package_Index, namespace)
}

// Package_Cursor_Handle is a caller-owned package cursor.
type Package_Cursor_Handle *Package_Cursor

// Package_Cursor_Handle_Invariants composes the package cursor behind its handle.
func Package_Cursor_Handle_Invariants(value Package_Cursor_Handle, namespace aver.Namespace) {
  if value == nil {
    return
  }
  Package_Cursor_Invariants(*value, namespace)
}

// Served_Reading is the purpose one file pass stands on: the bytes of a Go file, or the names
// that file declares. A contract file stands in the pass that checks packages, thus neither
// purpose of one stands here.
type Served_Reading Reading

// Served_Reading_Invariants admits the two purposes one file pass stands on.
func Served_Reading_Invariants(value Served_Reading, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_Int(int(value), int(READING_SOURCE), int(READING_DECLARATION)).
    Ensure()
}

// Served_Reading_Handle is the caller-owned purpose of the next file pass read.
type Served_Reading_Handle *Served_Reading

// Served_Reading_Handle_Invariants composes the purpose behind its handle.
func Served_Reading_Handle_Invariants(
  value Served_Reading_Handle,
  namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Served_Reading_Invariants(*value, namespace)
}

// Checked_Reading is the purpose one package check stands on: the bytes of a Go file, its
// contract markdown, or its contract test. A file read for the names it declares stands in the
// pass before this one, thus that purpose stands nowhere here.
type Checked_Reading Reading

// Checked_Reading_Invariants admits the three purposes one package check stands on.
func Checked_Reading_Invariants(value Checked_Reading, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_3_Int(
    int(value),
    int(READING_SOURCE),
    int(READING_MARKDOWN),
    int(READING_TEST),
  ).
    Ensure()
}

// Checked_Reading_Handle is the caller-owned purpose of the next package check read.
type Checked_Reading_Handle *Checked_Reading

// Checked_Reading_Handle_Invariants composes the purpose behind its handle.
func Checked_Reading_Handle_Invariants(
  value Checked_Reading_Handle,
  namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Checked_Reading_Invariants(*value, namespace)
}

// Contract_Reading is the purpose one read the check pass did not open stands on: a package's
// contract markdown, its contract test, or a Go file read for the names it declares. A Go file
// read for its own bytes stands at the close that judges it, thus that purpose stands nowhere
// here.
type Contract_Reading Reading

// Contract_Reading_Invariants admits the three purposes one such read stands on.
func Contract_Reading_Invariants(value Contract_Reading, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_3_Int(
    int(value),
    int(READING_MARKDOWN),
    int(READING_TEST),
    int(READING_DECLARATION),
  ).
    Ensure()
}

// Contract_Reading_Handle is the caller-owned purpose of one such read.
type Contract_Reading_Handle *Contract_Reading

// Contract_Reading_Handle_Invariants composes the purpose behind its handle.
func Contract_Reading_Handle_Invariants(
  value Contract_Reading_Handle,
  namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Contract_Reading_Invariants(*value, namespace)
}

// Reading_Handle is the caller-owned purpose of the next read. It stands apart from the read,
// because a helper that names the next file counts no byte of the file standing open.
type Reading_Handle *Reading

// Reading_Handle_Invariants composes the purpose behind its handle.
func Reading_Handle_Invariants(value Reading_Handle, namespace aver.Namespace) {
  if value == nil {
    return
  }
  Reading_Invariants(*value, namespace)
}

// Read is the file the run has open: what its bytes serve, which contract file they are, and how
// many of them landed. It stands apart from the pass, because a read states no count of the tree.
type Read struct {
  // Reading is what the open file read serves.
  Reading Reading
  // Source_Count counts the bytes of the current file in Memory.Source.
  Source_Count Source_Count
  // Path holds the path of the file being read, fixed, or reported.
  Path Read_Path_Slice
}

// Read_Invariants composes the open read.
func Read_Invariants(value Read, namespace aver.Namespace) {
  Reading_Invariants(value.Reading, namespace)
  Source_Count_Invariants(value.Source_Count, namespace)
  Read_Path_Slice_Invariants(value.Path, namespace)
}

// Read_Handle is a caller-owned open read.
type Read_Handle *Read

// Read_Handle_Invariants composes the open read behind its handle.
func Read_Handle_Invariants(value Read_Handle, namespace aver.Namespace) {
  if value == nil {
    return
  }
  Read_Invariants(*value, namespace)
}

// Files is where the run stands in the collected paths. It stands apart from the pass, because a
// step that reports one file states no count of packages or directories.
type Files struct {
  // File_Count counts the paths in Memory.Paths.
  File_Count File_Count
}

// Files_Invariants composes the collected paths.
func Files_Invariants(value Files, namespace aver.Namespace) {
  File_Count_Invariants(value.File_Count, namespace)
}

// Files_Handle is a caller-owned count of collected paths.
type Files_Handle *Files

// Files_Handle_Invariants composes the collected paths behind their handle.
func Files_Handle_Invariants(value Files_Handle, namespace aver.Namespace) {
  if value == nil {
    return
  }
  Files_Invariants(*value, namespace)
}

// File_Cursor is the file the run stands on. It stands apart from the count, because the walk
// that collects paths stands on none of them, thus a step of that walk states no cursor it
// could move.
type File_Cursor struct {
  // File_Index is the file the run stands on.
  File_Index File_Index
}

// File_Cursor_Invariants composes the file cursor.
func File_Cursor_Invariants(value File_Cursor, namespace aver.Namespace) {
  File_Index_Invariants(value.File_Index, namespace)
}

// File_Cursor_Handle is a caller-owned file cursor.
type File_Cursor_Handle *File_Cursor

// File_Cursor_Handle_Invariants composes the file cursor behind its handle.
func File_Cursor_Handle_Invariants(value File_Cursor_Handle, namespace aver.Namespace) {
  if value == nil {
    return
  }
  File_Cursor_Invariants(*value, namespace)
}

// Finding is what the last check found. Only the passes that check, fix or report one take it,
// so a helper that never reads it owes no rule and no position.
type Finding struct {
  // Diagnostic is what the last check found.
  Diagnostic Diagnostic
}

// Finding_Invariants composes what the last check found.
func Finding_Invariants(value Finding, namespace aver.Namespace) {
  Diagnostic_Invariants(value.Diagnostic, namespace)
}

// Finding_Handle is caller-owned finding state.
type Finding_Handle *Finding

// Finding_Handle_Invariants composes a present finding.
func Finding_Handle_Invariants(value Finding_Handle, namespace aver.Namespace) {
  if value == nil {
    return
  }
  Finding_Invariants(*value, namespace)
}

// Fix_State is how far one fix write got. Only the passes that write a fix take it, so a helper
// that never writes one owes no count of it.
type Fix_State struct {
  // Fix_Count counts the fix bytes written to the current file.
  Fix_Count Fix_Count
}

// Fix_State_Invariants composes how far one fix write got.
func Fix_State_Invariants(value Fix_State, namespace aver.Namespace) {
  Fix_Count_Invariants(value.Fix_Count, namespace)
}

// Fix_State_Handle is caller-owned fix state.
type Fix_State_Handle *Fix_State

// Fix_State_Handle_Invariants composes present fix state.
func Fix_State_Handle_Invariants(value Fix_State_Handle, namespace aver.Namespace) {
  if value == nil {
    return
  }
  Fix_State_Invariants(*value, namespace)
}

// Tally is whether the run reported a finding and where it continues after one. Only the passes
// that report take it, so a helper that never reports owes no mark and no continuation.
type Tally struct {
  // Found is whether the run reported a diagnostic, which fails its exit.
  Found Boolean
  // Resume is where the run continues after its report.
  Resume Resume
}

// Tally_Invariants composes what the run reported.
func Tally_Invariants(value Tally, namespace aver.Namespace) {
  Boolean_Invariants(value.Found, namespace)
  Resume_Invariants(value.Resume, namespace)
}

// Tally_Handle is caller-owned report state.
type Tally_Handle *Tally

// Tally_Handle_Invariants composes present report state.
func Tally_Handle_Invariants(value Tally_Handle, namespace aver.Namespace) {
  if value == nil {
    return
  }
  Tally_Invariants(*value, namespace)
}

// Findings is the line the run writes about what it found.
type Findings struct {
  // Line views the part of Memory.Line to write.
  Line Line
}

// Findings_Invariants composes what the run found.
func Findings_Invariants(value Findings, namespace aver.Namespace) {
  Line_Invariants(value.Line, namespace)
}

// Findings_Handle is caller-owned finding state.
type Findings_Handle *Findings

// Findings_Handle_Invariants composes present finding state.
func Findings_Handle_Invariants(value Findings_Handle, namespace aver.Namespace) {
  if value == nil {
    return
  }
  Findings_Invariants(*value, namespace)
}

// Streams is where one submission writes and what it stands on: the parts the pump hands the
// submit step, so the step states them rather than the whole run.
type Streams struct {
  // Loop is where the run stands.
  Loop Loop_Handle
  // Stage is the step of the run.
  Stage Stage_Handle
  // Walk is the pass over the tree.
  Walk Walk_Handle
  // Package_Cursor is the package whose contract the run checks.
  Package_Cursor Package_Cursor_Handle
  // Orphans is the orphan the run states next.
  Orphans Orphan_Cursor_Handle
  // Files is what the walk collected.
  Files Files_Handle
  // File_Cursor is the file the run stands on.
  File_Cursor File_Cursor_Handle
  // Read is the file the run has open.
  Read Read_Handle
  // Format is what rewrites one source into its canonical form.
  Format Format
  // Directory is the directory the walk has open.
  Directory Directory_Handle
  // Flags is every yes or no the run keeps.
  Flags Run_Flag_Storage
  // Findings is what the run found.
  Findings Findings_Handle
  // Outcome is what the run will exit as.
  Outcome Outcome_Handle
  // Report is the flat JSON form of the last diagnostic.
  Report Report_Handle
  // Summary is the flat JSON form of a passed run.
  Summary Summary_Handle
  // Finding is what the last check found.
  Finding Finding_Handle
  // Fix_State is how far one fix write got.
  Fix_State Fix_State_Handle
  // Tally is how many findings the run reported.
  Tally Tally_Handle
  // History is where the history check stands.
  History History_Handle
  // Output receives one line per diagnostic.
  Output nbio.Stream
  // Errors receives the diagnostic of a failed step.
  Errors nbio.Stream
}

// Streams_Invariants composes what one submission stands on.
func Streams_Invariants(value Streams, namespace aver.Namespace) {
  Loop_Handle_Invariants(value.Loop, namespace)
  Stage_Handle_Invariants(value.Stage, namespace)
  Walk_Handle_Invariants(value.Walk, namespace)
  Package_Cursor_Handle_Invariants(value.Package_Cursor, namespace)
  Orphan_Cursor_Handle_Invariants(value.Orphans, namespace)
  Files_Handle_Invariants(value.Files, namespace)
  File_Cursor_Handle_Invariants(value.File_Cursor, namespace)
  Read_Handle_Invariants(value.Read, namespace)
  Format_Invariants(value.Format, namespace)
  Directory_Handle_Invariants(value.Directory, namespace)
  Run_Flag_Storage_Invariants(value.Flags, namespace)
  Findings_Handle_Invariants(value.Findings, namespace)
  Outcome_Handle_Invariants(value.Outcome, namespace)
  Report_Handle_Invariants(value.Report, namespace)
  Summary_Handle_Invariants(value.Summary, namespace)
  Finding_Handle_Invariants(value.Finding, namespace)
  Fix_State_Handle_Invariants(value.Fix_State, namespace)
  Tally_Handle_Invariants(value.Tally, namespace)
  History_Handle_Invariants(value.History, namespace)
  // A cursor past the count names a package the walk never recorded, thus the cursor
  // stands at the count at most, where it states that every package is checked.
  aver.Always(
    int(value.Package_Cursor.Package_Index) < int(value.Walk.Package_Count)+1,
    "A package cursor stands inside its count.",
  )
  // A cursor past the count names a path the walk never collected, thus the cursor stands
  // at the count at most, where it states that every file is served.
  aver.Always(
    int(value.File_Cursor.File_Index) < int(value.Files.File_Count)+1,
    "A file cursor stands inside its count.",
  )
}

// Runner_Fields is one whole run's caller-owned state. Callbacks record into it; the root calls
// Rearm to turn what they recorded into the next submission, then pumps. It stands in parts, so a
// helper takes the part it works on and its namespace states that part alone.
type Runner_Fields struct {
  // Loop stays first so the static callback recovers the runner from the completion alone.
  Loop
  // Stage is the step of the run.
  Stage Stage
  // Progress is where the stage stands and how the run ends.
  Progress
  // Outcome is what the run will exit as.
  Outcome
  // Reports is what one write states as flat JSON.
  Reports
  // Output receives one line per diagnostic.
  Output nbio.Stream
  // Error_Output receives the diagnostic of a malformed command line or failed step.
  Error_Output nbio.Stream
  // Clock times the run.
  Clock time.Clock
  // Start is when the root opened the run.
  Start time.Monotonic_Moment
  // Arguments is the command line the root read.
  Arguments os.Arguments
  // Work_Queued is bound by the root.
  Work_Queued Work_Queued_Procedure
  // Stopped is bound by the root.
  Stopped Stopped_Procedure
  // Syntax holds the parsed file and its line index.
  Syntax requirements.Syntax
  // Memory is every store the run uses; the root allocates it once. It stays behind a handle,
  // because a run's fields past the stack bound of an interface conversion would allocate on
  // every assertion.
  Memory Memory_Handle
  // Walk is the pass over the tree.
  Walk
  // Package_Cursor is the package whose contract the run checks.
  Package_Cursor
  // Orphan_Cursor is the orphan the run states next.
  Orphan_Cursor
  // Files is what the walk collected.
  Files
  // File_Cursor is the file the run stands on.
  File_Cursor
  // Read is the file the run has open.
  Read
  // Format is what rewrites one source into its canonical form.
  Format
  // Command is every store one command line parse fills.
  Command
  // Directory is the directory the walk has open.
  Directory
  // Flags is every yes or no the run keeps. It stands apart from the pass, because a step
  // that reads one flag states no count of the tree.
  Flags Run_Flag_Storage
  // Findings is what the run found.
  Findings
  // Finding is what the last check found.
  Finding
  // Fix_State is how far one fix write got.
  Fix_State
  // Tally is how many findings the run reported.
  Tally
  // History is where the history check stands.
  History History
}

// Runner_Fields_Invariants composes every run field.
func Runner_Fields_Invariants(value Runner_Fields, namespace aver.Namespace) {
  Loop_Invariants(value.Loop, namespace)
  Stage_Invariants(value.Stage, namespace)
  Progress_Invariants(value.Progress, namespace)
  Outcome_Invariants(value.Outcome, namespace)
  Reports_Invariants(value.Reports, namespace)
  time.Clock_Invariants(value.Clock, namespace)
  time.Monotonic_Moment_Invariants(value.Start, namespace)
  os.Arguments_Invariants(value.Arguments, namespace)
  Memory_Handle_Invariants(value.Memory, namespace)
  requirements.Syntax_Invariants(value.Syntax, namespace)
  Walk_Invariants(value.Walk, namespace)
  Package_Cursor_Invariants(value.Package_Cursor, namespace)
  Orphan_Cursor_Invariants(value.Orphan_Cursor, namespace)
  Files_Invariants(value.Files, namespace)
  File_Cursor_Invariants(value.File_Cursor, namespace)
  Read_Invariants(value.Read, namespace)
  Format_Invariants(value.Format, namespace)
  Command_Invariants(value.Command, namespace)
  Directory_Invariants(value.Directory, namespace)
  Run_Flag_Storage_Invariants(value.Flags, namespace)
  Findings_Invariants(value.Findings, namespace)
  Finding_Invariants(value.Finding, namespace)
  Fix_State_Invariants(value.Fix_State, namespace)
  Tally_Invariants(value.Tally, namespace)
  History_Invariants(value.History, namespace)
}

// Runner is one whole run.
type Runner struct {
  Runner_Fields
}

// Runner_Invariants composes every run field.
func Runner_Invariants(value Runner, namespace aver.Namespace) {
  Runner_Fields_Invariants(value.Runner_Fields, namespace)
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

// Rearm_Input is what one rearm stands on: the run's stores, the parse it holds, the parts one
// pass works on, where the pump stands, and how long the run has taken.
type Rearm_Input struct {
  // Memory is the run's stores.
  Memory Memory_Handle
  // Syntax is the parse state and its line index.
  Syntax requirements.Syntax
  // State is the parts one pass works on.
  State Streams
  // Progress is where the pump stands.
  Progress Progress_Handle
  // Elapsed is how long the run has taken.
  Elapsed Elapsed
  // Arguments is the command line the root read.
  Arguments os.Arguments
  // Command is the parse stores.
  Command Command
}

// Rearm_Input_Invariants composes what one rearm stands on.
func Rearm_Input_Invariants(value Rearm_Input, namespace aver.Namespace) {
  Memory_Handle_Invariants(value.Memory, namespace)
  requirements.Syntax_Invariants(value.Syntax, namespace)
  Streams_Invariants(value.State, namespace)
  Progress_Handle_Invariants(value.Progress, namespace)
  Elapsed_Invariants(value.Elapsed, namespace)
  os.Arguments_Invariants(value.Arguments, namespace)
  Command_Invariants(value.Command, namespace)
  // A count past its store names a slot the store never held, thus each count stands at
  // the store it counts rather than at a slot past the end of it.
  memory, state := value.Memory, value.State
  aver.Always(
    int(state.Files.File_Count) < len(memory.Paths)+1,
    "A file count stands inside its table.",
  )
  aver.Always(
    int(state.Walk.Package_Count) < len(memory.Packages)+1,
    "A package count stands inside its table.",
  )
  aver.Always(
    int(state.Read.Source_Count) < len(memory.Source)+1,
    "A source count stands inside its store.",
  )
  aver.Always(
    int(state.Directory.Directory_Count) < len(memory.Directories)+1,
    "A directory count stands inside its table.",
  )
}

// Opening_Phase is the phase a run opens on: a runner no run has stood on, or the run before it
// stopped.
type Opening_Phase Phase

// Opening_Phase_Invariants admits a fresh runner and a stopped run.
func Opening_Phase_Invariants(value Opening_Phase, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_Int(int(value), int(PHASE_READY), int(PHASE_STOPPED)).
    Ensure()
}

// Opening_Phase_Handle is the caller-owned phase a run opens on.
type Opening_Phase_Handle *Opening_Phase

// Opening_Phase_Handle_Invariants composes a present opening phase.
func Opening_Phase_Handle_Invariants(value Opening_Phase_Handle, namespace aver.Namespace) {
  if value == nil {
    return
  }
  Opening_Phase_Invariants(*value, namespace)
}

// Moment_Handle is a caller-owned moment.
type Moment_Handle *time.Monotonic_Moment

// Moment_Handle_Invariants composes a present moment.
func Moment_Handle_Invariants(value Moment_Handle, namespace aver.Namespace) {
  if value == nil {
    return
  }
  time.Monotonic_Moment_Invariants(*value, namespace)
}

// Main_Input is what a run opens on: the phase the runner stands in, its start, and the clock.
type Main_Input struct {
  // Phase is where the runner stands.
  Phase Opening_Phase_Handle
  // Start is when the run opens.
  Start Moment_Handle
  // Clock times the run.
  Clock time.Clock
}

// Main_Input_Invariants composes what a run opens on.
func Main_Input_Invariants(value Main_Input, namespace aver.Namespace) {
  Opening_Phase_Handle_Invariants(value.Phase, namespace)
  Moment_Handle_Invariants(value.Start, namespace)
  time.Clock_Invariants(value.Clock, namespace)
}

// Advance_Pass is what one ready stage leaves the run on: another stage ready, an operation to
// submit, or a stopped run.
type Advance_Pass int

// Advance_Pass_Invariants admits each pass.
func Advance_Pass_Invariants(value Advance_Pass, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_3_Int(
    int(value),
    int(ADVANCE_READY),
    int(ADVANCE_SUBMIT),
    int(ADVANCE_STOPPED),
  ).
    Ensure()
}

// Advance_Input is what one ready stage stands on: the run's stores, the parse it holds, the
// parts one pass works on, and how long the run has taken.
type Advance_Input struct {
  // Memory is the run's stores.
  Memory Memory_Handle
  // Syntax is the parse state and its line index.
  Syntax requirements.Syntax
  // State is the parts one pass works on.
  State Streams
  // Elapsed is how long the run has taken.
  Elapsed Elapsed
}

// Advance_Input_Invariants composes what one ready stage stands on.
func Advance_Input_Invariants(value Advance_Input, namespace aver.Namespace) {
  Memory_Handle_Invariants(value.Memory, namespace)
  requirements.Syntax_Invariants(value.Syntax, namespace)
  Streams_Invariants(value.State, namespace)
  Elapsed_Invariants(value.Elapsed, namespace)
}

// Steps the stage a run is ready on: a stage that queues no operation of its own retires
// nothing, thus it leaves the run ready for the stage it names.
func advance(input Advance_Input) (next Advance_Pass) {
  defer func() { Advance_Pass_Invariants(next, "advance.next") }()
  Advance_Input_Invariants(input, "advance.input")
  memory, syntax, state, elapsed := input.Memory, input.Syntax, input.State, input.Elapsed
  checked := Checked_Reading_Handle((*Checked_Reading)(&state.Read.Reading))
  path := Read_Path_Handle(&state.Read.Path)
  packages, files := state.Walk.Package_Count, state.Files.File_Count
  switch *state.Stage {
  case STAGE_PACKAGE:
    // A run that found something rests at the package stage, thus the pump stops it
    // where the step left it.
    stepped := package_step(
      memory,
      state.Walk,
      state.Package_Cursor,
      state.Findings,
      elapsed,
      state.Summary,
      state.Tally.Found,
      state.Flags,
      checked,
      state.Outcome,
      path,
    )
    *state.Stage = Stage(stepped)
    if stepped == Step_Pass(STAGE_PACKAGE) {
      return ADVANCE_STOPPED
    }
  case STAGE_SERVE:
    state.File_Cursor.File_Index++
    served := Served_Reading_Handle((*Served_Reading)(&state.Read.Reading))
    next := file_next(memory, packages, files, state.File_Cursor, served, path)
    *state.Stage = Stage(next)
  case STAGE_DECLARED:
    state.File_Cursor.File_Index++
    *state.Stage = STAGE_DECLARE
  case STAGE_DECLARE:
    reading := Served_Reading_Handle((*Served_Reading)(&state.Read.Reading))
    formatter, cursor := state.Format.Formatter,
      state.File_Cursor
    next := declare_next(memory, packages, state.Flags, files, cursor, reading, formatter, path)
    *state.Stage = Stage(next)
  case STAGE_REQUIREMENTS:
    *state.Stage = Stage(requirements_next(memory, syntax, checked, path))
  case STAGE_FINDING:
    // A contract finding fails the run, and the run takes the contract up again once its line
    // is written.
    *state.Stage, state.Tally.Found, state.Tally.Resume = STAGE_REPORT, true, RESUME_REQUIREMENTS
    if !bool(report_requirement(memory, state.Findings, state.Report, state.Flags, path)) {
      failed := Failure_Path(state.Read.Path)
      fail(memory, state.Findings, failed, STEP_REPORT, state.Flags, state.Outcome)
      *state.Stage = STAGE_FAIL
    }
  case STAGE_PACKAGED:
    state.Package_Cursor.Package_Index++
    *state.Stage = STAGE_PACKAGE
  case STAGE_HISTORY_NEXT:
    findings, report, tally := state.Findings, state.Report, state.Tally
    past := Past{Orphans: state.Orphans, Walk: state.Walk}
    past.Loop, past.Outcome = state.Loop, state.Outcome
    next := history_next(memory, findings, report, tally, state.Flags, &state.History.Walk, past)
    *state.Stage = Stage(next)
  default:
    return ADVANCE_SUBMIT
  }
  return ADVANCE_READY
}

// Rearm turns what a callback recorded into the next submission and reports whether it did
// anything. The root calls it with the parts one pass works on, so no namespace here states the
// whole run.
func Rearm(input Rearm_Input) (progressed Boolean) {
  defer func() { Boolean_Invariants(progressed, "rearm.progressed") }()
  Rearm_Input_Invariants(input, "rearm.input")
  memory, syntax, state := input.Memory, input.Syntax, input.State
  progress := input.Progress
  switch progress.Phase {
  case PHASE_OPENED:
    // A run opens on the stores the run before it left, thus each count, cursor,
    // and flag stands at its start rather than carries one run into the next.
    *state.Walk, *state.Package_Cursor, *state.Files = Walk{}, Package_Cursor{}, Files{}
    *state.File_Cursor, *state.Fix_State = File_Cursor{}, Fix_State{}
    state.Tally.Found, state.Read.Source_Count, state.History.Walk = false, 0, History_Walk{}
    // A run reads harness.json, then walks the one directory a command line names.
    state.Read.Path = append(state.Read.Path[:0], CONFIGURATION_PATH...)
    state.Directory.Current, state.Read.Reading = state.Directory.Current[:0], READING_SOURCE
    *state.Orphans, state.Directory.Directory_Count = Orphan_Cursor{}, 1
    clear(state.Flags.Values)
    progress.Phase, *state.Stage = PHASE_READY, STAGE_FAIL
    // The vector is a host's, and cli.Program_Parse states a program name as a
    // precondition rather than an error, thus a vector holding none is refused here.
    if len(input.Arguments) < ARGUMENT_COUNT_MINIMUM {
      state.Findings.Line = append(Line(memory.Line[:0]), USAGE_ABSENT_LINE...)
      state.Outcome.Status = EXIT_FAILURE
      return true
    }
    named := Program_Arguments(input.Arguments)
    *state.Stage = Stage(
      parse(memory, state.Findings, named, state.Flags,
        input.Command, state.Outcome),
    )
    return true
  case PHASE_READY:
    switch advance(
      Advance_Input{
        Memory: memory, Syntax: syntax, State: state, Elapsed: input.Elapsed,
      },
    ) {
    case ADVANCE_STOPPED:
      progress.Phase = PHASE_STOPPED
    case ADVANCE_SUBMIT:
      // A stream can retire inside the submit, thus the phase moves first or
      // the retirement it records would be erased.
      progress.Phase = PHASE_SUBMITTED
      submit(
        memory,
        state.Loop,
        Submitted_Stage(*state.Stage),
        state.Findings,
        state.Output,
        state.Errors,
        state.Finding.Diagnostic.Fix,
        state.Fix_State,
        state.Read,
        state.Directory,
        state.History.Walk.Source,
        &state.History.Command,
      )
    }
    return true
  case PHASE_RETIRED:
    progress.Phase = PHASE_READY
    // The last line of a run retires at the stage that wrote it, thus the judge
    // states that the run is done and the pump stops it here.
    if bool(judge(memory, syntax, state)) {
      progress.Phase = PHASE_STOPPED
    }
    return true
  }
  return false
}

// Runner_Work_Queued reports whether a callback recorded work only Rearm can act on, or the run
// stopped: either way a pump pass is over.
func Runner_Work_Queued(phase Driven_Phase) (queued Boolean) {
  defer func() { Boolean_Invariants(queued, "runner_work_queued.queued") }()
  Driven_Phase_Invariants(phase, "runner_work_queued.phase")
  return phase != Driven_Phase(PHASE_SUBMITTED)
}

// Runner_Stopped reports whether the run is over and Status is final.
func Runner_Stopped(phase Pass_Phase) (stopped Boolean) {
  defer func() { Boolean_Invariants(stopped, "runner_stopped.stopped") }()
  Pass_Phase_Invariants(phase, "runner_stopped.phase")
  return phase == Pass_Phase(PHASE_STOPPED)
}

// Submits the operation the stage stands on.
func submit(
  memory Memory_Handle,
  loop Loop_Handle,
  stage Submitted_Stage,
  findings Findings_Handle,
  output nbio.Stream,
  errors nbio.Stream,
  fix Fix,
  fix_state Fix_State_Handle,
  read Read_Handle,
  directory Directory_Handle,
  source History_Source,
  command Git_Command_Handle,
) {
  History_Source_Invariants(source, "submit.source")
  Git_Command_Handle_Invariants(command, "submit.command")
  Read_Handle_Invariants(read, "submit.read")
  Fix_State_Handle_Invariants(fix_state, "submit.fix_state")
  Fix_Invariants(fix, "submit.fix")
  Memory_Handle_Invariants(memory, "submit.memory")
  Loop_Handle_Invariants(loop, "submit.loop")
  Findings_Handle_Invariants(findings, "submit.findings")
  Directory_Handle_Invariants(directory, "submit.directory")
  Submitted_Stage_Invariants(stage, "submit.stage")
  completion := &loop.Completion
  storage := loop.IO.Storage
  switch Stage(stage) {
  case STAGE_DIRECTORY_OPEN:
    directory.Directory_Count--
    next := memory.Directories[directory.Directory_Count]
    current := append(directory.Current[:0], next...)
    directory.Current = current
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
      loop.Directory,
      filepath.Directory_Buffer(memory.Directory_Buffer),
      filepath.Directory_Entries(memory.Entries),
      runner_retired,
    )
  case STAGE_DIRECTORY_CLOSE:
    nbio.IO_Close(loop.IO, completion, loop.Directory, runner_retired)
  case STAGE_OPEN:
    stored := read.Path
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
    submit_read(memory, loop, read.Reading, Unfilled_Count(read.Source_Count))
  case STAGE_CLOSE:
    nbio.IO_Close(loop.IO, completion, loop.File, runner_retired)
  case STAGE_REPORT:
    nbio.Write(output, completion, findings.Line, runner_retired)
  case STAGE_HISTORY_HEAD, STAGE_HISTORY_REFS, STAGE_HISTORY_LOG:
    submit_history(memory, loop, History_Stage(stage), Submitted_Source(source), command)
  case STAGE_FAIL:
    nbio.Write(errors, completion, findings.Line, runner_retired)
  case STAGE_HELP, STAGE_PASSED:
    nbio.Write(output, completion, findings.Line, runner_retired)
  case STAGE_FIX_OPEN, STAGE_FIX_WRITE, STAGE_FIX_CLOSE:
    submit_fix(
      read.Path, loop, Fix_Stage(stage), Written_Fix(fix), fix_state,
    )
  }
}

// Reads the rest of the open file into the storage of what the read serves.
func submit_read(
  memory Memory_Handle,
  loop Loop_Handle,
  reading Reading,
  count Unfilled_Count,
) {
  Memory_Handle_Invariants(memory, "submit_read.memory")
  Reading_Invariants(reading, "submit_read.reading")
  Unfilled_Count_Invariants(count, "submit_read.count")
  Loop_Handle_Invariants(loop, "submit_read.loop")
  buffer := memory.Source[count:]
  offset := int64(count)
  if reading == READING_MARKDOWN {
    checker := memory.Requirements
    loaded := checker.Numbers.Values[requirements.NUMBER_MARKDOWN_COUNT]
    buffer = File_Bytes(checker.Markdown[loaded:])
    offset = int64(loaded)
  }
  nbio.Storage_Read(
    loop.IO.Storage,
    &loop.Completion,
    loop.File,
    buffer,
    offset,
    READ_TIMEOUT,
    runner_retired,
  )
}

// Truncate then write, since the storage surface names no rename to swap a whole file in.
func submit_fix(
  path_store Read_Path_Slice,
  loop Loop_Handle,
  stage Fix_Stage,
  fix Written_Fix,
  fix_state Fix_State_Handle,
) {
  Fix_State_Handle_Invariants(fix_state, "submit_fix.fix_state")
  Written_Fix_Invariants(fix, "submit_fix.fix")
  Read_Path_Slice_Invariants(path_store, "submit_fix.path_store")
  Loop_Handle_Invariants(loop, "submit_fix.loop")
  Fix_Stage_Invariants(stage, "submit_fix.stage")
  completion := &loop.Completion
  storage := loop.IO.Storage
  switch Stage(stage) {
  case STAGE_FIX_OPEN:
    stored := path_store
    path := unsafe.String(unsafe.SliceData(stored), len(stored))
    options := nbio.Open_At_Options{Access: nbio.OPEN_WRITE_ONLY, Truncate: true}
    nbio.Storage_Open_At(
      storage, completion, nbio.DIRECTORY_CURRENT, path, options, runner_retired,
    )
  case STAGE_FIX_WRITE:
    nbio.Storage_Write(
      storage,
      completion,
      loop.File,
      fix[fix_state.Fix_Count:],
      int64(fix_state.Fix_Count),
      WRITE_TIMEOUT,
      runner_retired,
    )
  case STAGE_FIX_CLOSE:
    nbio.IO_Close(loop.IO, completion, loop.File, runner_retired)
  }
}

// Walk_Stage is a stage of the walk over one directory: its open, a read of its entries, or its
// close.
type Walk_Stage Stage

// Walk_Stage_Invariants admits the three stages of the walk over one directory.
func Walk_Stage_Invariants(value Walk_Stage, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_3_Int(
    int(value),
    int(STAGE_DIRECTORY_OPEN),
    int(STAGE_DIRECTORY_READ),
    int(STAGE_DIRECTORY_CLOSE),
  ).
    Ensure()
}

// Walk_Stage_Handle is the caller-owned stage of the walk, which the judge moves past.
type Walk_Stage_Handle *Walk_Stage

// Walk_Stage_Handle_Invariants composes the stage behind its handle.
func Walk_Stage_Handle_Invariants(value Walk_Stage_Handle, namespace aver.Namespace) {
  if value == nil {
    return
  }
  Walk_Stage_Invariants(*value, namespace)
}

// Judges one step of the walk over a directory and moves the run to the stage it names. It reports
// whether a failure there owes its line: an open and a close name their own, a read defers to the
// close that follows it.
func judge_walk(
  memory Memory_Handle,
  loop Loop_Handle,
  walk Walk_Handle,
  run_flags Run_Flag_Storage,
  files Files_Handle,
  stores Formatter_Stores,
  current Walked_Path,
  queue Directory_Count_Handle,
  stage Walk_Stage_Handle,
) (owed Boolean) {
  defer func() { Boolean_Invariants(owed, "judge_walk.owed") }()
  Memory_Handle_Invariants(memory, "judge_walk.memory")
  Loop_Handle_Invariants(loop, "judge_walk.loop")
  Walk_Handle_Invariants(walk, "judge_walk.walk")
  Run_Flag_Storage_Invariants(run_flags, "judge_walk.run_flags")
  Files_Handle_Invariants(files, "judge_walk.files")
  Formatter_Stores_Invariants(stores, "judge_walk.stores")
  Walked_Path_Invariants(current, "judge_walk.current")
  Directory_Count_Handle_Invariants(queue, "judge_walk.queue")
  Walk_Stage_Handle_Invariants(stage, "judge_walk.stage")
  next := (*Stage)((*Walk_Stage)(stage))
  switch Stage(*stage) {
  case STAGE_DIRECTORY_OPEN:
    *next = Stage(judge_directory_open(loop, run_flags))
    return true
  case STAGE_DIRECTORY_READ:
    *next = Stage(judge_directory_read(memory, loop, walk, run_flags, files, current, queue))
    return false
  }
  closed := judge_directory_close(memory, loop, walk, run_flags, files, stores, current, queue)
  *next = Stage(closed)
  return true
}

func judge(
  memory Memory_Handle,
  syntax requirements.Syntax,
  state Streams,
) (stopped Boolean) {
  defer func() { Boolean_Invariants(stopped, "judge.stopped") }()
  requirements.Syntax_Invariants(syntax, "judge.syntax")
  Memory_Handle_Invariants(memory, "judge.memory")
  Streams_Invariants(state, "judge.state")
  loop, stage, read := state.Loop, state.Stage, state.Read
  findings, run_flags, outcome := state.Findings, state.Flags, state.Outcome
  tally, current, opened := state.Tally, Walked_Path(state.Directory.Current), STAGE_DIRECTORY_OPEN
  queue := Directory_Count_Handle(&state.Directory.Directory_Count)
  walk, files, stores := state.Walk, state.Files, state.Format.Formatter
  // A step the run cannot finish names itself once, past the stage it failed at. A close
  // names its own, thus it owes none here.
  owed, step, failed := Boolean(false), STEP_WALK, Failure_Path(current)
  switch *stage {
  case STAGE_DIRECTORY_OPEN, STAGE_DIRECTORY_READ, STAGE_DIRECTORY_CLOSE:
    walked := Walk_Stage_Handle((*Walk_Stage)((*Stage)(stage)))
    owed = judge_walk(memory, loop, walk, run_flags, files, stores, current, queue, walked)
  case STAGE_OPEN:
    *stage = Stage(judge_open(memory, loop, run_flags, read))
    owed, step, failed = true, STEP_OPEN, Failure_Path(read.Path)
  case STAGE_READ:
    unfilled := Unfilled_Count_Handle((*Unfilled_Count)(&read.Source_Count))
    *stage = Stage(judge_read(memory, loop, run_flags, read.Reading, unfilled))
  case STAGE_CLOSE:
    closed := judge_close(
      memory,
      syntax,
      loop,
      findings,
      state.Report,
      state.Finding,
      tally,
      run_flags,
      File_Slot(state.File_Cursor.File_Index),
      read,
      state.Format,
      outcome,
    )
    // A contract close takes the stage its check names, thus its slot of the table stands unread.
    stages := [...]Stage{STAGE_FAIL, STAGE_SERVE, STAGE_FIX_OPEN, STAGE_REPORT, opened, opened}
    *stage = stages[closed]
    if closed == CLOSE_PASS_CONTRACT {
      contract := Contract_Reading_Handle((*Contract_Reading)(&read.Reading))
      count := read.Source_Count
      next := judge_served(memory, syntax, run_flags, contract, count, stores, state.Finding)
      *stage = Stage(next)
    }
  case STAGE_REPORT:
    *stage = Stage(judge_report(loop, tally.Resume))
    owed, step, failed = true, STEP_REPORT, Failure_Path(read.Path)
  case STAGE_FAIL, STAGE_HELP, STAGE_PASSED:
    judge_last(loop, outcome)
    return true
  case STAGE_HISTORY_HEAD, STAGE_HISTORY_REFS, STAGE_HISTORY_LOG:
    source := Judged_Source_Handle((*Judged_Source)(&state.History.Walk.Source))
    *stage = Stage(judge_history(memory, loop, History_Stage(*stage), source))
  case STAGE_FIX_OPEN, STAGE_FIX_WRITE, STAGE_FIX_CLOSE:
    fix, reading := Written_Fix(state.Finding.Diagnostic.Fix), Fixed_Reading(read.Reading)
    *stage = Stage(judge_fix(loop, Fix_Stage(*stage), fix, state.Fix_State, run_flags, reading))
    owed, step, failed = true, STEP_FIX, Failure_Path(read.Path)
  }
  if bool(owed) {
    if *stage == STAGE_FAIL {
      fail(memory, findings, failed, step, run_flags, outcome)
    }
  }
  return false
}

// States what one last line leaves the run at. The last line has no second line to name its own
// failed write, thus a failed write of it stands in the status alone.
func judge_last(loop Loop_Handle, outcome Outcome_Handle) {
  Loop_Handle_Invariants(loop, "judge_last.loop")
  Outcome_Handle_Invariants(outcome, "judge_last.outcome")
  if loop.Completion.Error != nil {
    outcome.Status = EXIT_FAILURE
  }
}

// Takes the descriptor one open landed on, and opens the read of that file on a clean store.
func judge_open(
  memory Memory_Handle,
  loop Loop_Handle,
  run_flags Run_Flag_Storage,
  read Read_Handle,
) (next Open_Pass) {
  defer func() { Open_Pass_Invariants(next, "judge_open.next") }()
  Memory_Handle_Invariants(memory, "judge_open.memory")
  Loop_Handle_Invariants(loop, "judge_open.loop")
  Run_Flag_Storage_Invariants(run_flags, "judge_open.run_flags")
  Read_Handle_Invariants(read, "judge_open.read")
  // A file the run cannot open names the step that failed, which the caller states.
  if loop.Completion.Error != nil {
    return Open_Pass(STAGE_FAIL)
  }
  loop.File = nbio.File(loop.Completion.Data)
  read.Source_Count = 0
  requirements.Load(memory.Requirements, 0)
  run_flags.Values[RUN_READ_FAILED] = false
  return Open_Pass(STAGE_READ)
}

// Judges the write of one diagnostic: a write that took no byte names its own step, and one that
// landed moves to whatever the run stands on next.
func judge_report(loop Loop_Handle, resume Resume) (next Report_Pass) {
  defer func() { Report_Pass_Invariants(next, "judge_report.next") }()
  Resume_Invariants(resume, "judge_report.resume")
  Loop_Handle_Invariants(loop, "judge_report.loop")
  // A write that took no byte names the step that failed, which the caller states.
  if loop.Completion.Error != nil {
    return Report_Pass(STAGE_FAIL)
  }
  // A run that owes its open package more of its contract takes it up again; one that
  // owes none turns to the file after the one the diagnostic named.
  if resume == RESUME_REQUIREMENTS {
    return Report_Pass(STAGE_REQUIREMENTS)
  }
  if resume == RESUME_HISTORY {
    return Report_Pass(STAGE_HISTORY_NEXT)
  }
  return Report_Pass(STAGE_SERVE)
}

// A descriptor outlives a failed write, thus the failure waits for the close. A write that moves
// no byte fails too, or a full disk would spin the run.
func judge_fix(
  loop Loop_Handle,
  stage Fix_Stage,
  fix Written_Fix,
  fix_state Fix_State_Handle,
  run_flags Run_Flag_Storage,
  reading Fixed_Reading,
) (next Fix_Pass) {
  defer func() { Fix_Pass_Invariants(next, "judge_fix.next") }()
  Fixed_Reading_Invariants(reading, "judge_fix.reading")
  Fix_State_Handle_Invariants(fix_state, "judge_fix.fix_state")
  Written_Fix_Invariants(fix, "judge_fix.fix")
  Loop_Handle_Invariants(loop, "judge_fix.loop")
  Run_Flag_Storage_Invariants(run_flags, "judge_fix.run_flags")
  Fix_Stage_Invariants(stage, "judge_fix.stage")
  completion := &loop.Completion
  switch stage {
  case Fix_Stage(STAGE_FIX_OPEN):
    if completion.Error != nil {
      return Fix_Pass(STAGE_FAIL)
    }
    loop.File = nbio.File(completion.Data)
    // The fix of one file counts the bytes of that file alone, thus the count the
    // fix before it left stands at nothing here rather than carries into this one.
    fix_state.Fix_Count = 0
    run_flags.Values[RUN_FIX_FAILED] = false
    // A fix opens its file cut to nothing, thus a fix of no bytes would leave the
    // file empty. Only a tier one diagnostic reaches here, and it carries the whole
    // canonical form, thus the fix that opened one stands.
    aver.Always(
      len(fix) > bytes.SLICE_SIZE_MINIMUM,
      "A fix that opened a file carries bytes.",
    )
    return Fix_Pass(STAGE_FIX_WRITE)
  case Fix_Stage(STAGE_FIX_WRITE):
    if completion.Error != nil {
      run_flags.Values[RUN_FIX_FAILED] = true
      return Fix_Pass(STAGE_FIX_CLOSE)
    }
    if completion.Data == 0 {
      run_flags.Values[RUN_FIX_FAILED] = true
      return Fix_Pass(STAGE_FIX_CLOSE)
    }
    fix_state.Fix_Count += Fix_Count(completion.Data)
    if int(fix_state.Fix_Count) == len(fix) {
      return Fix_Pass(STAGE_FIX_CLOSE)
    }
    return Fix_Pass(STAGE_FIX_WRITE)
  }
  // A fix that neither opened nor wrote stands at its close, since a fix stands at
  // one of three stages and the switch names the other two.
  aver.Always(
    stage == Fix_Stage(STAGE_FIX_CLOSE),
    "A fix past its open and its write stands at its close.",
  )
  // A fix the run cannot finish names the step that failed, which the caller states.
  if completion.Error != nil {
    return Fix_Pass(STAGE_FAIL)
  }
  if run_flags.Values[RUN_FIX_FAILED] {
    return Fix_Pass(STAGE_FAIL)
  }
  // A fixed contract stands before its own check, thus the run takes the check up again.
  if reading == Fixed_Reading(READING_MARKDOWN) {
    return Fix_Pass(STAGE_REQUIREMENTS)
  }
  return Fix_Pass(STAGE_SERVE)
}

// A descriptor outlives a failed pass, thus the failure waits for the close.
func judge_directory_read(
  memory Memory_Handle,
  loop Loop_Handle,
  walk Walk_Handle,
  run_flags Run_Flag_Storage,
  files Files_Handle,
  current Walked_Path,
  queue Directory_Count_Handle,
) (next Directory_Pass) {
  defer func() { Directory_Pass_Invariants(next, "judge_directory_read.next") }()
  Walked_Path_Invariants(current, "judge_directory_read.current")
  Directory_Count_Handle_Invariants(queue, "judge_directory_read.queue")
  Files_Handle_Invariants(files, "judge_directory_read.files")
  Memory_Handle_Invariants(memory, "judge_directory_read.memory")
  Walk_Handle_Invariants(walk, "judge_directory_read.walk")
  Run_Flag_Storage_Invariants(run_flags, "judge_directory_read.run_flags")
  Loop_Handle_Invariants(loop, "judge_directory_read.loop")
  if loop.Completion.Error != nil {
    run_flags.Values[RUN_WALK_FAILED] = true
    return Directory_Pass(STAGE_DIRECTORY_CLOSE)
  }
  if loop.Completion.Data == 0 {
    return Directory_Pass(STAGE_DIRECTORY_CLOSE)
  }
  for index := range loop.Completion.Data {
    collect(
      memory, walk, loop.IO.Storage, memory.Entries[index], run_flags, files,
      current, queue,
    )
    if run_flags.Values[RUN_WALK_FAILED] {
      return Directory_Pass(STAGE_DIRECTORY_CLOSE)
    }
  }
  // The pass took entries, thus another pass over the same directory stands before its
  // close: a host states its children in as many passes as its store holds.
  return Directory_Pass(STAGE_DIRECTORY_READ)
}

// Opens the walk of one directory: the descriptor stands, and what the pass finds starts clean.
func judge_directory_open(
  loop Loop_Handle,
  run_flags Run_Flag_Storage,
) (next Directory_Open_Pass) {
  defer func() {
    Directory_Open_Pass_Invariants(next, "judge_directory_open.next")
  }()
  Run_Flag_Storage_Invariants(run_flags, "judge_directory_open.run_flags")
  Loop_Handle_Invariants(loop, "judge_directory_open.loop")
  // A directory the walk cannot open names the step that failed, which the caller states.
  if loop.Completion.Error != nil {
    return Directory_Open_Pass(STAGE_FAIL)
  }
  loop.Directory = nbio.File(loop.Completion.Data)
  run_flags.Values[RUN_WALK_FAILED] = false
  run_flags.Values[RUN_PENDING_GO] = false
  run_flags.Values[RUN_PENDING_MARKDOWN] = false
  run_flags.Values[RUN_PENDING_TEST] = false
  return Directory_Open_Pass(STAGE_DIRECTORY_READ)
}

// Records one child directory for the walk to open, or states that the store holds no slot for
// it. The walk stands on the parent, thus the child carries the parent's path before its name.
func collect_directory(
  directories Directory_Paths,
  entry nbio.Directory_Entry,
  run_flags Run_Flag_Storage,
  current Parent_Path,
  queue Directory_Count_Handle,
) {
  Directory_Paths_Invariants(directories, "collect_directory.directories")
  Run_Flag_Storage_Invariants(run_flags, "collect_directory.run_flags")
  Parent_Path_Invariants(current, "collect_directory.current")
  Directory_Count_Handle_Invariants(queue, "collect_directory.queue")
  if int(*queue) == len(directories) {
    run_flags.Values[RUN_WALK_FAILED], run_flags.Values[RUN_WALK_DIRECTORIES] = true, true
    return
  }
  child := directories[*queue][:0]
  child = append(append(child, current...), PATH_SEPARATOR)
  directories[*queue] = append(child, entry.Name...)
  *queue++
}

// Records one child: a directory waits its turn, a regular Go file joins the check. A symbolic
// link is neither, so the walk never leaves the tree it was given, and states only whether its
// target resolves.
func collect(
  memory Memory_Handle,
  walk Walk_Handle,
  storage nbio.Storage,
  entry nbio.Directory_Entry,
  run_flags Run_Flag_Storage,
  files Files_Handle,
  current Walked_Path,
  queue Directory_Count_Handle,
) {
  Memory_Handle_Invariants(memory, "collect.memory")
  Walk_Handle_Invariants(walk, "collect.walk")
  Run_Flag_Storage_Invariants(run_flags, "collect.run_flags")
  Files_Handle_Invariants(files, "collect.files")
  Walked_Path_Invariants(current, "collect.current")
  Directory_Count_Handle_Invariants(queue, "collect.queue")
  // A name is untrusted: one holding a byte below a space breaks the line that states it
  // in two, and one too wide states no path, thus the walk fails on either.
  run_flags.Values[RUN_WALK_NAME] = Mark(name_refused(Spelled_Name(entry.Name)))
  run_flags.Values[RUN_WALK_WIDE] = len(current)+len("/")+len(entry.Name) >
    filepath.PATH_SIZE_MAXIMUM
  barred := run_flags.Values[RUN_WALK_NAME]
  if run_flags.Values[RUN_WALK_WIDE] {
    barred = true
  }
  if barred {
    run_flags.Values[RUN_WALK_FAILED] = true
    return
  }
  if entry.Is_Directory {
    parent := Parent_Path(current)
    collect_directory(memory.Directories, entry, run_flags, parent, queue)
    return
  }
  // An entry that is no regular file names neither source nor contract: the run would read
  // or rewrite what it stands for, thus the walk asks the host before the name decides.
  link := append(append(memory.Link[:0], current...), PATH_SEPARATOR)
  link = append(link, entry.Name...)
  orphans := Orphan_Count_Handle(&walk.Orphan_Count)
  if !bool(collect_status(memory, orphans, storage, run_flags, Entry_Path(link))) {
    return
  }
  // Exact names, since a case-insensitive disk would resolve a differently cased file.
  source := false
  if len(entry.Name) > len(GO_SUFFIX)-1 {
    source = entry.Name[len(entry.Name)-len(GO_SUFFIX):] == GO_SUFFIX
  }
  named := source
  if entry.Name == requirements.MARKDOWN_NAME {
    named = true
  }
  if entry.Name == requirements.TEST_NAME {
    named = true
  }
  if !named {
    return
  }
  if int(files.File_Count) == len(memory.Paths) {
    run_flags.Values[RUN_WALK_FAILED], run_flags.Values[RUN_WALK_FILES] = true, true
    return
  }
  path := memory.Paths[files.File_Count][:0]
  path = append(append(path, current...), PATH_SEPARATOR)
  path = append(path, entry.Name...)
  if entry.Name == requirements.MARKDOWN_NAME {
    run_flags.Values[RUN_PENDING_MARKDOWN] = true
  }
  if entry.Name == requirements.TEST_NAME {
    run_flags.Values[RUN_PENDING_TEST] = true
  }
  if !source {
    return
  }
  memory.Paths[files.File_Count] = path
  owner, slot := Package_Slot(walk.Package_Count), File_Slot(files.File_Count)
  collect_source(memory.File_Packages, owner, slot, run_flags)
  files.File_Count++
}

// Reports whether one child is a regular file, as the host states it without following a link. A
// link whose target resolves to nothing joins the orphan table.
func collect_status(
  memory Memory_Handle,
  orphans Orphan_Count_Handle,
  storage nbio.Storage,
  run_flags Run_Flag_Storage,
  entry Entry_Path,
) (regular Boolean) {
  defer func() { Boolean_Invariants(regular, "collect_status.regular") }()
  Memory_Handle_Invariants(memory, "collect_status.memory")
  Orphan_Count_Handle_Invariants(orphans, "collect_status.orphans")
  Run_Flag_Storage_Invariants(run_flags, "collect_status.run_flags")
  Entry_Path_Invariants(entry, "collect_status.entry")
  stated := unsafe.String(unsafe.SliceData(entry), len(entry))
  status, status_err := nbio.Storage_Status(storage, stated)
  if status_err != nil {
    run_flags.Values[RUN_WALK_FAILED] = true
    return false
  }
  if !nbio.File_Mode_Is_Symbolic_Link(status.Mode) {
    return Boolean(nbio.File_Mode_Is_Regular(status.Mode))
  }
  if !bool(link_orphaned(memory, storage, entry)) {
    return false
  }
  if int(*orphans) == len(memory.Orphans) {
    run_flags.Values[RUN_WALK_FAILED], run_flags.Values[RUN_WALK_ORPHANS] = true, true
    return false
  }
  memory.Orphans[*orphans] = append(memory.Orphans[*orphans][:0], entry...)
  *orphans++
  return false
}

// Follows one link and reports whether its target resolves to nothing: a target that names no
// entry, one no path holds, or a chain past LINK_COUNT_MAXIMUM links.
func link_orphaned(
  memory Memory_Handle,
  storage nbio.Storage,
  link Entry_Path,
) (orphaned Boolean) {
  defer func() { Boolean_Invariants(orphaned, "link_orphaned.orphaned") }()
  Memory_Handle_Invariants(memory, "link_orphaned.memory")
  Entry_Path_Invariants(link, "link_orphaned.link")
  path := append(memory.Resolved[:0], link...)
  for range LINK_COUNT_MAXIMUM {
    stated := unsafe.String(unsafe.SliceData(path), len(path))
    count, read_err := nbio.Storage_Read_Link(storage, stated, memory.Target)
    if read_err != nil {
      return true
    }
    if count == 0 {
      return true
    }
    // A relative target stands in the directory of the link that names it.
    base := 0
    if memory.Target[0] != PATH_SEPARATOR {
      base = int(bytes.Last_Index_Byte(bytes.Slice(path), PATH_SEPARATOR)) + 1
    }
    if base+count > filepath.PATH_SIZE_MAXIMUM {
      return true
    }
    path = append(path[:base], memory.Target[:count]...)
    stated = unsafe.String(unsafe.SliceData(path), len(path))
    status, status_err := nbio.Storage_Status(storage, stated)
    if status_err != nil {
      return true
    }
    if !status.Exists {
      return true
    }
    if !nbio.File_Mode_Is_Symbolic_Link(status.Mode) {
      return false
    }
  }
  return true
}

// Records the Go file the walk stands on: its slot names the package the walk is closing.
func collect_source(
  owners Package_Indices,
  owner Package_Slot,
  slot File_Slot,
  run_flags Run_Flag_Storage,
) {
  Package_Indices_Invariants(owners, "collect_source.owners")
  Package_Slot_Invariants(owner, "collect_source.owner")
  File_Slot_Invariants(slot, "collect_source.slot")
  Run_Flag_Storage_Invariants(run_flags, "collect_source.run_flags")
  owners[slot] = owner
  run_flags.Values[RUN_PENDING_GO] = true
}

func judge_directory_close(
  memory Memory_Handle,
  loop Loop_Handle,
  walk Walk_Handle,
  run_flags Run_Flag_Storage,
  files Files_Handle,
  formatter Formatter_Stores,
  current Walked_Path,
  queue Directory_Count_Handle,
) (next Walk_Pass) {
  defer func() { Walk_Pass_Invariants(next, "judge_directory_close.next") }()
  Memory_Handle_Invariants(memory, "judge_directory_close.memory")
  Walk_Handle_Invariants(walk, "judge_directory_close.walk")
  Run_Flag_Storage_Invariants(run_flags, "judge_directory_close.run_flags")
  Files_Handle_Invariants(files, "judge_directory_close.files")
  Formatter_Stores_Invariants(formatter, "judge_directory_close.formatter")
  Walked_Path_Invariants(current, "judge_directory_close.current")
  Directory_Count_Handle_Invariants(queue, "judge_directory_close.queue")
  Loop_Handle_Invariants(loop, "judge_directory_close.loop")
  // A close the host refused and a pass the walk gave up on each leave the walk failed,
  // thus one line names the directory either way.
  stopped := loop.Completion.Error != nil
  if run_flags.Values[RUN_WALK_FAILED] {
    stopped = true
  }
  if stopped {
    return Walk_Pass(STAGE_FAIL)
  }
  // A directory joins the package table once it proves to hold a Go file; its files already
  // name the slot it takes.
  if run_flags.Values[RUN_PENDING_GO] {
    // A package owns at least one Go file and the two tables hold the same count,
    // thus the path table fills first and no tree reaches the full one. A package
    // owes a contract too, thus its directory holds the contract name beside the Go
    // file that named it, and one too deep for that name states so rather than
    // builds a path past its store.
    contract := len(current) + len("/") + CONTRACT_NAME_SIZE_MAXIMUM
    aver.Always(
      int(walk.Package_Count) < len(memory.Packages),
      "A package table outlasts the file table, which fills first.",
    )
    run_flags.Values[RUN_WALK_WIDE] = contract > filepath.PATH_SIZE_MAXIMUM
    if run_flags.Values[RUN_WALK_WIDE] {
      return Walk_Pass(STAGE_FAIL)
    }
    slot := &memory.Packages[walk.Package_Count]
    slot.Path = append(slot.Path[:0], current...)
    slot.Markdown = Markdown_Mark(run_flags.Values[RUN_PENDING_MARKDOWN])
    slot.Test = Test_Mark(run_flags.Values[RUN_PENDING_TEST])
    slot.Impure = false
    recorded := Recorded_Package{
      Path: slot.Path, Markdown: slot.Markdown, Test: slot.Test,
    }
    // A table states its length alone, thus each package states its own parts
    // where it is recorded rather than nowhere at all.
    Recorded_Package_Invariants(recorded, "judge_directory_close.recorded")
    walk.Package_Count++
  }
  if *queue > 0 {
    return Walk_Pass(STAGE_DIRECTORY_OPEN)
  }
  paths_sort(memory.Paths, memory.File_Packages, files.File_Count)
  format.Forget(&format.Formatter{Places: formatter.Places, Table_Fields: formatter.Table})
  // A run opens its cursor at the first file and the walk stands on none, thus the cursor
  // still stands there and needs no reset that would carry it into this pass.
  run_flags.Values[RUN_UNDECLARED] = false
  return Walk_Pass(STAGE_DECLARE)
}

// Reads the next Go file for the names it declares. Past the last file the check pass starts,
// because a name one file declares is spelled in files the run checks before it.
func declare_next(
  memory Memory_Handle,
  packages Package_Count,
  run_flags Run_Flag_Storage,
  served File_Count,
  file_cursor File_Cursor_Handle,
  reading Served_Reading_Handle,
  formatter Formatter_Stores,
  path_store Read_Path_Handle,
) (next File_Pass) {
  defer func() { File_Pass_Invariants(next, "declare_next.next") }()
  Read_Path_Handle_Invariants(path_store, "declare_next.path_store")
  Memory_Handle_Invariants(memory, "declare_next.memory")
  Package_Count_Invariants(packages, "declare_next.packages")
  Run_Flag_Storage_Invariants(run_flags, "declare_next.run_flags")
  File_Count_Invariants(served, "declare_next.served")
  File_Cursor_Handle_Invariants(file_cursor, "declare_next.file_cursor")
  Served_Reading_Handle_Invariants(reading, "declare_next.reading")
  Formatter_Stores_Invariants(formatter, "declare_next.formatter")
  if file_cursor.File_Index < File_Index(served) {
    *path_store = append(
      (*path_store)[:0], memory.Paths[file_cursor.File_Index]...,
    )
    *reading = Served_Reading(READING_DECLARATION)
    return File_Pass(STAGE_OPEN)
  }
  // A rename that misses one file breaks every file that spells the name, thus a file the
  // table never took bars every rename of the run.
  if run_flags.Values[RUN_UNDECLARED] {
    view := format.Formatter{Places: formatter.Places, Table_Fields: formatter.Table}
    format.Forget(&view)
  }
  // A name one file declares is spelled in files the run checks before it, thus every file
  // stands declared here rather than one of them reaching the check undeclared.
  aver.Always(
    int(file_cursor.File_Index) == int(served),
    "A run declares every file before it checks any.",
  )
  file_cursor.File_Index = 0
  return file_next(
    memory,
    packages,
    served,
    file_cursor,
    reading,
    path_store,
  )
}

// Takes the names the read file declares into the table, or records that the table never took
// them: a file past the source bound or refused by the parser holds no tree to read them from.
func declare(
  formatter Formatter_Stores,
  tree ast.Syntax_Tree_Handle,
  file File_Bytes,
  count Source_Count,
) (declared Mark) {
  defer func() { Mark_Invariants(declared, "declare.declared") }()
  Formatter_Stores_Invariants(formatter, "declare.formatter")
  ast.Syntax_Tree_Handle_Invariants(tree, "declare.tree")
  File_Bytes_Invariants(file, "declare.file")
  Source_Count_Invariants(count, "declare.count")
  text := file[:count]
  if len(text) > requirements.SOURCE_SIZE_MAXIMUM {
    return false
  }
  source := token.Source(text)
  state := ast.Parse_State{Syntax_Tree: *tree}
  if !bool(ast.Parse(&state, source).OK) {
    return false
  }
  view := format.Formatter{Places: formatter.Places, Table_Fields: formatter.Table}
  return Mark(format.Declare(&view, &state, source))
}

// Untrusted_Name is a name the run was handed rather than wrote: a child one directory pass
// states, or the directory one command line names.
type Untrusted_Name string

// Untrusted_Name_Invariants bounds the name by what one path holds.
func Untrusted_Name_Invariants(value Untrusted_Name, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(
    len(value),
    bytes.SLICE_SIZE_MINIMUM,
    filepath.PATH_SIZE_MAXIMUM,
  ).
    Ensure()
}

// Reports whether one entry name holds a byte no line can state.
// States that no byte of a path a run writes steers a terminal or breaks the line that holds it.
// Every path is joined from names the walk read, thus the guard those names passed stands here
// too rather than only where each one was read.
func path_steady(path Untrusted_Name) {
  Untrusted_Name_Invariants(path, "path_steady.path")
  for index := range len(path) {
    aver.Always(
      path[index] > NAME_BYTE_MINIMUM-1,
      "A path a run writes holds no byte below a space.",
    )
  }
}

func name_refused(name Spelled_Name) (refused Boolean) {
  defer func() { Boolean_Invariants(refused, "name_refused.refused") }()
  Spelled_Name_Invariants(name, "name_refused.name")
  for index := range len(name) {
    if name[index] < NAME_BYTE_MINIMUM {
      return true
    }
  }
  return false
}

// NAME_BYTE_MINIMUM is the smallest byte a name holds: one below it steers a terminal or breaks
// a line rather than names anything.
const NAME_BYTE_MINIMUM = ' '

// SPELLED_NAME_SIZE_MINIMUM is the shortest name a host spells, since a name of no bytes names
// nothing at all.
const SPELLED_NAME_SIZE_MINIMUM = filepath.NONEMPTY_SIZE_MINIMUM

// Spelled_Name is one name a host spelled: a command line argument, or an entry of a directory
// the walk read. A name of no bytes names nothing, thus its callers refuse it before it stands
// here.
type Spelled_Name string

// Spelled_Name_Invariants bounds the name by what one path holds.
func Spelled_Name_Invariants(value Spelled_Name, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(
    len(value),
    SPELLED_NAME_SIZE_MINIMUM,
    filepath.PATH_SIZE_MAXIMUM,
  ).
    Ensure()
}

// A directory pass reports children in whatever order the kernel keeps, thus the paths sort
// before any line is written, or two runs over one tree would disagree.
func paths_sort(table Path_Table, owners Package_Indices, count File_Count) {
  Path_Table_Invariants(table, "paths_sort.table")
  Package_Indices_Invariants(owners, "paths_sort.owners")
  File_Count_Invariants(count, "paths_sort.count")
  paths := table[:count]
  owners = owners[:count]
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

// Judges a read that served the contract check or the name table rather than one file's rules.
func judge_served(
  memory Memory_Handle,
  syntax requirements.Syntax,
  run_flags Run_Flag_Storage,
  reading Contract_Reading_Handle,
  source_count Source_Count,
  formatter Formatter_Stores,
  finding Finding_Handle,
) (next Served_Pass) {
  defer func() { Served_Pass_Invariants(next, "judge_served.next") }()
  Finding_Handle_Invariants(finding, "judge_served.finding")
  Memory_Handle_Invariants(memory, "judge_served.memory")
  requirements.Syntax_Invariants(syntax, "judge_served.syntax")
  Run_Flag_Storage_Invariants(run_flags, "judge_served.run_flags")
  Contract_Reading_Handle_Invariants(reading, "judge_served.reading")
  Source_Count_Invariants(source_count, "judge_served.source_count")
  Formatter_Stores_Invariants(formatter, "judge_served.formatter")
  checker := memory.Requirements
  switch Reading(*reading) {
  case READING_MARKDOWN:
    // A contract off the form the rewrite answers is fixed before its check, thus the
    // check states only what stays, at the lines the fixed file holds.
    rewritten := requirements.Flag(false)
    if run_flags.Values[RUN_AUTOFIX] {
      rewritten = requirements.Rewrite(checker)
    }
    requirements.Begin(checker)
    if rewritten {
      count := checker.Numbers.Values[requirements.NUMBER_MARKDOWN_COUNT]
      // The rewritten contract is the form it should stand in, which the fix carries the
      // way a Go file's canonical form does.
      fix := Fix(checker.Markdown[:count])
      finding.Diagnostic = Diagnostic{Tier: TIER_ONE, Rule: RULE_UNFORMATTED, Fix: fix}
      return Served_Pass(STAGE_FIX_OPEN)
    }
  case READING_DECLARATION:
    declared := declare(
      formatter, syntax.Syntax_Tree, memory.Source, source_count,
    )
    if !declared {
      run_flags.Values[RUN_UNDECLARED] = true
    }
    return Served_Pass(STAGE_DECLARED)
  case READING_TEST:
    // A test file past the source bound has its own diagnostic; an empty source refuses
    // to parse, thus its tests go unchecked rather than flagged twice.
    count := requirements.Number(0)
    if int(source_count) < len(memory.Source) {
      count = requirements.Number(source_count)
    }
    requirements.Load_Test(checker, requirements.Test_Count(count))
  }
  return Served_Pass(STAGE_REQUIREMENTS)
}

// A descriptor outlives a failed read, thus the failure waits for the close.
func judge_read(
  memory Memory_Handle,
  loop Loop_Handle,
  run_flags Run_Flag_Storage,
  reading Reading,
  count Unfilled_Count_Handle,
) (next Read_Pass) {
  defer func() { Read_Pass_Invariants(next, "judge_read.next") }()
  Memory_Handle_Invariants(memory, "judge_read.memory")
  Run_Flag_Storage_Invariants(run_flags, "judge_read.run_flags")
  Reading_Invariants(reading, "judge_read.reading")
  Unfilled_Count_Handle_Invariants(count, "judge_read.count")
  Loop_Handle_Invariants(loop, "judge_read.loop")
  if loop.Completion.Error != nil {
    run_flags.Values[RUN_READ_FAILED] = true
    return Read_Pass(STAGE_CLOSE)
  }
  if loop.Completion.Data == 0 {
    return Read_Pass(STAGE_CLOSE)
  }
  if reading == READING_MARKDOWN {
    checker := memory.Requirements
    loaded := checker.Numbers.Values[requirements.NUMBER_MARKDOWN_COUNT]
    loaded += requirements.Number(loop.Completion.Data)
    requirements.Load(checker, requirements.Markdown_Count(loaded))
    if int(loaded) == len(checker.Markdown) {
      return Read_Pass(STAGE_CLOSE)
    }
    return Read_Pass(STAGE_READ)
  }
  *count += Unfilled_Count(loop.Completion.Data)
  if int(*count) == len(memory.Source) {
    return Read_Pass(STAGE_CLOSE)
  }
  // The read took bytes and the store holds more, thus another read of the same file
  // stands before its close.
  return Read_Pass(STAGE_READ)
}

func judge_close(
  memory Memory_Handle,
  syntax requirements.Syntax,
  loop Loop_Handle,
  findings Findings_Handle,
  report Report_Handle,
  finding Finding_Handle,
  tally Tally_Handle,
  run_flags Run_Flag_Storage,
  slot File_Slot,
  read Read_Handle,
  format_state Format,
  outcome Outcome_Handle,
) (next Close_Pass) {
  defer func() { Close_Pass_Invariants(next, "judge_close.next") }()
  Format_Invariants(format_state, "judge_close.format_state")
  Tally_Handle_Invariants(tally, "judge_close.tally")
  Finding_Handle_Invariants(finding, "judge_close.finding")
  requirements.Syntax_Invariants(syntax, "judge_close.syntax")
  Memory_Handle_Invariants(memory, "judge_close.memory")
  Findings_Handle_Invariants(findings, "judge_close.findings")
  Run_Flag_Storage_Invariants(run_flags, "judge_close.run_flags")
  File_Slot_Invariants(slot, "judge_close.slot")
  Read_Handle_Invariants(read, "judge_close.read")
  Report_Handle_Invariants(report, "judge_close.report")
  Loop_Handle_Invariants(loop, "judge_close.loop")
  Outcome_Handle_Invariants(outcome, "judge_close.outcome")
  if loop.Completion.Error != nil {
    fail(memory, findings, Failure_Path(read.Path), STEP_CLOSE, run_flags, outcome)
    return CLOSE_PASS_FAILED
  }
  if run_flags.Values[RUN_READ_FAILED] {
    fail(memory, findings, Failure_Path(read.Path), STEP_READ, run_flags, outcome)
    return CLOSE_PASS_FAILED
  }
  // The first file a run reads is its configuration.
  if !run_flags.Values[RUN_CONFIGURED] {
    count := Configured_Count(min(int(read.Source_Count), configuration.SOURCE_STORAGE_SIZE))
    return Close_Pass(configure(memory, findings, run_flags, count, outcome))
  }
  if read.Reading != READING_SOURCE {
    return CLOSE_PASS_CONTRACT
  }
  finding.Diagnostic = Check(memory, syntax, read.Source_Count, format_state)
  impure(memory, syntax, finding.Diagnostic.Rule, slot, read.Source_Count)
  finding.Diagnostic = imports_check(memory, syntax, slot, read.Source_Count, finding.Diagnostic)
  tally.Resume = RESUME_FILE
  if finding.Diagnostic.Tier == TIER_NONE {
    return CLOSE_PASS_SERVED
  }
  if finding.Diagnostic.Tier == TIER_ONE {
    if run_flags.Values[RUN_AUTOFIX] {
      return CLOSE_PASS_FIX
    }
  }
  written := Reported{
    Tier:    Reported_Tier(finding.Diagnostic.Tier),
    Rule:    Reported_Rule(finding.Diagnostic.Rule),
    Failure: finding.Diagnostic.Failure,
    Line:    finding.Diagnostic.Line,
    Column:  finding.Diagnostic.Column,
  }
  tally.Found = true
  if !bool(
    report_file(
      Report_File_Input{
        Memory:   memory,
        Findings: findings,
        Report:   report,
        Written:  written,
        Flags:    run_flags,
        Slot:     slot,
      },
    ),
  ) {
    fail(memory, findings, Failure_Path(read.Path), STEP_REPORT, run_flags, outcome)
    return CLOSE_PASS_FAILED
  }
  return CLOSE_PASS_REPORT
}

// Decodes the configuration the run read. A file that does not decode states its fault and the
// byte it stands on, and the run fails before the walk.
func configure(
  memory Memory_Handle,
  findings Findings_Handle,
  run_flags Run_Flag_Storage,
  count Configured_Count,
  outcome Outcome_Handle,
) (next Configured_Pass) {
  defer func() { Configured_Pass_Invariants(next, "configure.next") }()
  Memory_Handle_Invariants(memory, "configure.memory")
  Findings_Handle_Invariants(findings, "configure.findings")
  Run_Flag_Storage_Invariants(run_flags, "configure.run_flags")
  Configured_Count_Invariants(count, "configure.count")
  Outcome_Handle_Invariants(outcome, "configure.outcome")
  run_flags.Values[RUN_CONFIGURED] = true
  source := configuration.Source(memory.Source[:count])
  fault := configuration.Parse(memory.Configuration, source)
  if fault.Kind == configuration.FAULT_NONE {
    return Configured_Pass(CLOSE_PASS_CONFIGURED)
  }
  kind := configuration.Recorded_Fault(fault.Kind)
  line := append(Line(memory.Line[:0]), CONFIGURATION_PREFIX...)
  if bool(configuration.Fault_Keyed(kind)) {
    line = append(append(line, configuration.Key_Text(fault.Key)...), ' ')
  }
  line = append(append(line, configuration.Fault_Text(kind)...), BYTE_INFIX...)
  // The digits land in the line's spare capacity, since a conversion would allocate.
  size := len(line)
  spare := strconv.Buffer(line[size : size+strconv.DIGIT_TEXT_SIZE_MAXIMUM])
  offset := strconv.Unsigned_Integer(fault.Offset)
  written := strconv.Format_Unsigned_Integer_Into(spare, offset, strconv.DECIMAL_BASE)
  findings.Line, outcome.Status = append(line[:size+int(written)], '\n'), EXIT_FAILURE
  return Configured_Pass(CLOSE_PASS_FAILED)
}

// Configured_Count is how many bytes of harness.json the run hands the decoder: the whole file,
// or one byte past the bound the decoder takes, which proves the file past it.
type Configured_Count int

// Configured_Count_Invariants bounds the count by what the decoder takes.
func Configured_Count_Invariants(value Configured_Count, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(
    int(value),
    slices.COUNT_MINIMUM,
    configuration.SOURCE_STORAGE_SIZE,
  ).
    Ensure()
}

// Configured_Pass is what the decode of harness.json leaves the run to do: the walk, or the
// failure it states.
type Configured_Pass Close_Pass

// Configured_Pass_Invariants admits the two a decode leaves.
func Configured_Pass_Invariants(value Configured_Pass, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_Int(
    int(value),
    int(CLOSE_PASS_FAILED),
    int(CLOSE_PASS_CONFIGURED),
  ).
    Ensure()
}

// Report_File_Input is what the write of one file diagnostic stands on: the stores, the line and
// the report it fills, the diagnostic, the mode, and the file it names.
type Report_File_Input struct {
  // Memory is the run's stores.
  Memory Memory_Handle
  // Findings holds the line the write fills.
  Findings Findings_Handle
  // Report holds the flat JSON object the write fills.
  Report Report_Handle
  // Written is the diagnostic the line states.
  Written Reported
  // Flags holds the mode the line stands in.
  Flags Run_Flag_Storage
  // Slot is the file the diagnostic names.
  Slot File_Slot
}

// Report_File_Input_Invariants composes what the write of one file diagnostic stands on.
func Report_File_Input_Invariants(value Report_File_Input, namespace aver.Namespace) {
  Memory_Handle_Invariants(value.Memory, namespace)
  Findings_Handle_Invariants(value.Findings, namespace)
  Report_Handle_Invariants(value.Report, namespace)
  Reported_Invariants(value.Written, namespace)
  Run_Flag_Storage_Invariants(value.Flags, namespace)
  File_Slot_Invariants(value.Slot, namespace)
}

// Writes one file diagnostic, as a text line or a flat JSON line.
func report_file(input Report_File_Input) (wrote Boolean) {
  defer func() { Boolean_Invariants(wrote, "report_file.wrote") }()
  Report_File_Input_Invariants(input, "report_file.input")
  memory, findings, report := input.Memory, input.Findings, input.Report
  written := input.Written
  path := memory.Paths[input.Slot]
  // One rule names one text and one message, which a flat JSON line and a text line state
  // alike.
  rule_text, message := RULE_TEXT_UNFORMATTED, Message_Text(MESSAGE_UNFORMATTED)
  switch Rule(written.Rule) {
  case RULE_REFUSED:
    rule_text = RULE_TEXT_REFUSED
    message = Message_Text(ast.Failure_Message(ast.Failure_Code(written.Failure)))
  case RULE_OVERSIZED:
    rule_text, message = RULE_TEXT_OVERSIZED, MESSAGE_OVERSIZED
  case RULE_UNPRINTABLE:
    rule_text, message = RULE_TEXT_UNPRINTABLE, MESSAGE_UNPRINTABLE
  case RULE_CONFLICTED:
    rule_text = RULE_TEXT_CONFLICTED
    message = requirements.MESSAGE_TEXT_CONFLICT_MARKER
  case RULE_IMPORTS:
    rule_text, message = RULE_TEXT_IMPORTS, Message_Text(import_message(memory))
  }
  if input.Flags.Values[RUN_JSON] {
    // The report lands in run storage: a local would escape into the reflected value
    // and allocate.
    *report = Report{
      Path:    Path_Text(unsafe.String(unsafe.SliceData(path), len(path))),
      Line:    Report_Line(written.Line),
      Column:  Report_Column(written.Column),
      Tier:    Report_Tier(written.Tier),
      Rule:    rule_text,
      Message: message,
    }
    return report_write(memory, findings, report)
  }
  path_steady(Untrusted_Name(path))
  findings.Line = append(Line(memory.Line[:0]), path...)
  // Only a refusal names a position, and a source past the line index has none the index
  // can name, thus its line names the path alone rather than a wrong place.
  if written.Line != POSITION_ABSENT {
    // The digits land in the line's spare capacity, since a conversion would allocate.
    places := [POSITION_NUMBER_COUNT]strconv.Unsigned_Integer{
      strconv.Unsigned_Integer(written.Line),
      strconv.Unsigned_Integer(written.Column),
    }
    for _, place := range places {
      findings.Line = append(findings.Line, ':')
      size := len(findings.Line)
      room := size + strconv.DIGIT_TEXT_SIZE_MAXIMUM
      spare := strconv.Buffer(findings.Line[size:room])
      count := strconv.Format_Unsigned_Integer_Into(
        spare, place, strconv.DECIMAL_BASE,
      )
      findings.Line = findings.Line[:size+int(count)]
    }
  }
  findings.Line = append(append(findings.Line, SEPARATOR...), message...)
  findings.Line = append(findings.Line, '\n')
  return true
}

// Marshals one diagnostic as one flat JSON line.
func report_write(
  memory Memory_Handle,
  findings Findings_Handle,
  report Report_Handle,
) (wrote Boolean) {
  defer func() { Boolean_Invariants(wrote, "report_write.wrote") }()
  Memory_Handle_Invariants(memory, "report_write.memory")
  Findings_Handle_Invariants(findings, "report_write.findings")
  Report_Handle_Invariants(report, "report_write.report")
  storage := memory.Json
  data, err := flatjson.Marshal_Into(
    flatjson.Output(storage[:flatjson.OUTPUT_SIZE_MAXIMUM]),
    flatjson.Value(reflect.ValueOf(report)),
  )
  // Only a path of bytes JSON escapes six-fold outgrows the object, thus the caller names
  // the step that failed rather than this one cutting the line short.
  if err != nil {
    return false
  }
  line := storage[:len(data)+len("\n")]
  line[len(data)] = '\n'
  findings.Line = Line(line)
  return true
}

// Diagnostic_Text is one diagnostic line as it stands while a report builds it. It holds the
// path the report names before anything else, thus it is never shorter than one.
type Diagnostic_Text []byte

// Diagnostic_Text_Invariants bounds the line being built by the path it opens with.
func Diagnostic_Text_Invariants(value Diagnostic_Text, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(len(value), PATH_TEXT_SIZE_MINIMUM, LINE_SIZE_MAXIMUM).
    Ensure()
}

func file_next(
  memory Memory_Handle,
  packages Package_Count,
  served File_Count,
  file_cursor File_Cursor_Handle,
  reading Served_Reading_Handle,
  path_store Read_Path_Handle,
) (next File_Pass) {
  defer func() { File_Pass_Invariants(next, "file_next.next") }()
  Read_Path_Handle_Invariants(path_store, "file_next.path_store")
  Served_Reading_Handle_Invariants(reading, "file_next.reading")
  Memory_Handle_Invariants(memory, "file_next.memory")
  Package_Count_Invariants(packages, "file_next.packages")
  File_Count_Invariants(served, "file_next.served")
  File_Cursor_Handle_Invariants(file_cursor, "file_next.file_cursor")
  if file_cursor.File_Index < File_Index(served) {
    *path_store = append(
      (*path_store)[:0], memory.Paths[file_cursor.File_Index]...,
    )
    *reading = Served_Reading(READING_SOURCE)
    return File_Pass(STAGE_OPEN)
  }
  // A run that turns to its packages with a file left states no diagnostic of it, thus the
  // cursor stands at the count here rather than short of a file nobody checked.
  aver.Always(
    int(file_cursor.File_Index) == int(served),
    "A run serves every file before its packages.",
  )
  // A run opens its cursor at the first package and the walk checks none, thus the cursor
  // still stands there and needs no reset that would carry it into this pass.
  packages_sort(memory.Packages, packages)
  return File_Pass(STAGE_PACKAGE)
}

// Records a package main file, whose package owes no contract. A file the run could not read
// whole, or the parser would not take, proves no package clause: its token run states whatever
// the source held up to the fault. Such a file exempts nothing, thus its package owes a contract
// until a host fixes the file.
func impure(
  memory Memory_Handle,
  syntax requirements.Syntax,
  rule Rule,
  slot File_Slot,
  source_count Source_Count,
) {
  Memory_Handle_Invariants(memory, "impure.memory")
  requirements.Syntax_Invariants(syntax, "impure.syntax")
  Rule_Invariants(rule, "impure.rule")
  File_Slot_Invariants(slot, "impure.slot")
  Source_Count_Invariants(source_count, "impure.source_count")
  if rule == RULE_OVERSIZED {
    return
  }
  if rule == RULE_REFUSED {
    return
  }
  if rule == RULE_CONFLICTED {
    return
  }
  view := ast.Parse_State{Syntax_Tree: *syntax.Syntax_Tree}
  run := ast.Parse_State_Token_Run(&view)
  source := File_Bytes(memory.Source[:source_count])
  for index := 0; index+1 < len(run); index++ {
    if run[index].Kind != token.KIND_PACKAGE {
      continue
    }
    name := run[index+1]
    text := string(source[name.Offset : int(name.Offset)+int(name.Size)])
    owner := &memory.Packages[memory.File_Packages[slot]]
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
func packages_sort(table Packages, count Package_Count) {
  Packages_Invariants(table, "packages_sort.table")
  Package_Count_Invariants(count, "packages_sort.count")
  packages := table[:count]
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
func package_step(
  memory Memory_Handle,
  walk Walk_Handle,
  cursor Package_Cursor_Handle,
  findings Findings_Handle,
  elapsed Elapsed,
  summary Summary_Handle,
  found Boolean,
  run_flags Run_Flag_Storage,
  reading Checked_Reading_Handle,
  outcome Outcome_Handle,
  path_store Read_Path_Handle,
) (next Step_Pass) {
  defer func() { Step_Pass_Invariants(next, "package_step.next") }()
  Read_Path_Handle_Invariants(path_store, "package_step.path_store")
  Outcome_Handle_Invariants(outcome, "package_step.outcome")
  Checked_Reading_Handle_Invariants(reading, "package_step.reading")
  Run_Flag_Storage_Invariants(run_flags, "package_step.run_flags")
  Boolean_Invariants(found, "package_step.found")
  Memory_Handle_Invariants(memory, "package_step.memory")
  Walk_Handle_Invariants(walk, "package_step.walk")
  Package_Cursor_Handle_Invariants(cursor, "package_step.cursor")
  Findings_Handle_Invariants(findings, "package_step.findings")
  Elapsed_Invariants(elapsed, "package_step.elapsed")
  Summary_Handle_Invariants(summary, "package_step.summary")
  // A package left to check leaves the run on the stage that reads it; none left leaves it
  // where it stands, thus the finish of the whole run follows here.
  started := package_next(
    memory, walk, cursor, reading, path_store,
  )
  if started != Package_Pass(STAGE_PACKAGE) {
    return Step_Pass(started)
  }
  // A run that finishes with a package left states no finding of it, thus the cursor
  // stands at the count here rather than short of a package nobody checked.
  aver.Always(
    int(cursor.Package_Index) == int(walk.Package_Count),
    "A run checks every package before it finishes.",
  )
  // Past the packages the run judges the history it stands on, then finishes here again.
  if !run_flags.Values[RUN_HISTORY_DONE] {
    return Step_Pass(STAGE_HISTORY_HEAD)
  }
  // A run that found something states its findings and rests at the package stage; one
  // that found none writes the pass line and stands at the stage that writes it.
  outcome.Status = EXIT_SUCCESS
  if bool(found) {
    outcome.Status = EXIT_FAILURE
    return Step_Pass(STAGE_PACKAGE)
  }
  passed(memory, findings, elapsed, summary, run_flags)
  return Step_Pass(STAGE_PASSED)
}

// Starts the contract check of the next package, reading its REQUIREMENTS.md first when it has
// one; past the last package, it reports that the run is done.
func package_next(
  memory Memory_Handle,
  walk Walk_Handle,
  cursor Package_Cursor_Handle,
  reading Checked_Reading_Handle,
  path_store Read_Path_Handle,
) (next Package_Pass) {
  defer func() { Package_Pass_Invariants(next, "package_next.next") }()
  Read_Path_Handle_Invariants(path_store, "package_next.path_store")
  Memory_Handle_Invariants(memory, "package_next.memory")
  Walk_Handle_Invariants(walk, "package_next.walk")
  Package_Cursor_Handle_Invariants(cursor, "package_next.cursor")
  Checked_Reading_Handle_Invariants(reading, "package_next.reading")
  // No package left to check leaves the run where it stands, thus the caller finishes it.
  if int(cursor.Package_Index) == int(walk.Package_Count) {
    return Package_Pass(STAGE_PACKAGE)
  }
  found := memory.Packages[cursor.Package_Index]
  checker := memory.Requirements
  requirements.Load_Path(checker, requirements.Path_Count(copy(checker.Path, found.Path)))
  checker.Flags.Values[requirements.FLAG_HAS_MARKDOWN] = requirements.Flag(found.Markdown)
  checker.Flags.Values[requirements.FLAG_HAS_TEST] = requirements.Flag(found.Test)
  checker.Flags.Values[requirements.FLAG_IMPURE] = requirements.Flag(found.Impure)
  requirements.Load(checker, 0)
  if found.Markdown {
    *path_store = Read_Path_Slice(read_path(
      *path_store, checker, READ_FILE_MARKDOWN,
    ))
    *reading = Checked_Reading(READING_MARKDOWN)
    return Package_Pass(STAGE_OPEN)
  }
  requirements.Begin(checker)
  return Package_Pass(STAGE_REQUIREMENTS)
}

// Resumes the open package's check: a finding is reported, a request for requirements_test.go
// reads it, and the end moves to the next package.
func requirements_next(
  memory Memory_Handle,
  syntax requirements.Syntax,
  reading Checked_Reading_Handle,
  path_store Read_Path_Handle,
) (next Contract_Pass) {
  defer func() { Contract_Pass_Invariants(next, "requirements_next.next") }()
  Read_Path_Handle_Invariants(path_store, "requirements_next.path_store")
  Checked_Reading_Handle_Invariants(reading, "requirements_next.reading")
  requirements.Syntax_Invariants(syntax, "requirements_next.syntax")
  Memory_Handle_Invariants(memory, "requirements_next.memory")
  checker := memory.Requirements
  requirements.Next(checker, syntax)
  switch checker.Numbers.Values[requirements.NUMBER_STATE] {
  case requirements.STATE_FINDING:
    return Contract_Pass(STAGE_FINDING)
  case requirements.STATE_TEST:
    *path_store = Read_Path_Slice(read_path(
      *path_store, checker, READ_FILE_TEST,
    ))
    *reading = Checked_Reading(READING_TEST)
    return Contract_Pass(STAGE_OPEN)
  }
  // The check of one package is done, thus the run turns to the package after it.
  return Contract_Pass(STAGE_PACKAGED)
}

// Points Read_Path at the Read_File of the open package.
func read_path(
  store Read_Path_Slice,
  checker requirements.Checker_Handle,
  file Read_File,
) (written Contract_Path) {
  defer func() { Contract_Path_Invariants(written, "read_path.written") }()
  Read_Path_Slice_Invariants(store, "read_path.store")
  requirements.Checker_Handle_Invariants(checker, "read_path.checker")
  Read_File_Invariants(file, "read_path.file")
  count := checker.Numbers.Values[requirements.NUMBER_PATH_COUNT]
  written = append(Contract_Path(store[:0]), checker.Path[:count]...)
  written = append(written, PATH_SEPARATOR)
  if file == READ_FILE_TEST {
    return append(written, requirements.TEST_NAME...)
  }
  return append(written, requirements.MARKDOWN_NAME...)
}

// Writes one contract finding at its file, as a text line or a flat JSON line.
func report_requirement(
  memory Memory_Handle,
  findings Findings_Handle,
  report Report_Handle,
  run_flags Run_Flag_Storage,
  path_store Read_Path_Handle,
) (wrote Boolean) {
  defer func() { Boolean_Invariants(wrote, "report_requirement.wrote") }()
  Read_Path_Handle_Invariants(path_store, "report_requirement.path_store")
  Memory_Handle_Invariants(memory, "report_requirement.memory")
  Findings_Handle_Invariants(findings, "report_requirement.findings")
  Run_Flag_Storage_Invariants(run_flags, "report_requirement.run_flags")
  Report_Handle_Invariants(report, "report_requirement.report")
  checker := memory.Requirements
  stated := Read_File(checker.Numbers.Values[requirements.NUMBER_FILE])
  *path_store = Read_Path_Slice(read_path(*path_store, checker, stated))
  line := Diagnostic_Line(checker.Numbers.Values[requirements.NUMBER_LINE])
  path := *path_store
  message := checker.Message[:checker.Numbers.Values[requirements.NUMBER_MESSAGE_COUNT]]
  if run_flags.Values[RUN_JSON] {
    message_text := unsafe.String(unsafe.SliceData(message), len(message))
    // A finding the rewrite answers carries its fix, as the rewritten contract.
    tier := Report_Tier(TIER_TWO)
    if requirements.Fixable(checker) {
      tier = Report_Tier(TIER_ONE)
    }
    *report = Report{
      Path:    Path_Text(unsafe.String(unsafe.SliceData(path), len(path))),
      Line:    Report_Line(line),
      Tier:    tier,
      Rule:    RULE_TEXT_REQUIREMENTS,
      Message: Message_Text(message_text),
    }
    return report_write(memory, findings, report)
  }
  findings.Line = append(Line(memory.Line[:0]), path...)
  if line > 0 {
    findings.Line = append(findings.Line, ':')
    size := len(findings.Line)
    spare := strconv.Buffer(findings.Line[size : size+strconv.DIGIT_TEXT_SIZE_MAXIMUM])
    number := strconv.Unsigned_Integer(line)
    count := strconv.Format_Unsigned_Integer_Into(spare, number, strconv.DECIMAL_BASE)
    findings.Line = findings.Line[:size+int(count)]
  }
  findings.Line = append(append(findings.Line, SEPARATOR...), message...)
  findings.Line = append(findings.Line, '\n')
  return true
}

// Writes the pass line and the elapsed seconds to the millisecond. The digits land in the line's
// spare capacity, since a conversion would allocate.
func passed(
  memory Memory_Handle,
  findings Findings_Handle,
  elapsed Elapsed,
  summary Summary_Handle,
  run_flags Run_Flag_Storage,
) {
  Memory_Handle_Invariants(memory, "passed.memory")
  Findings_Handle_Invariants(findings, "passed.findings")
  Elapsed_Invariants(elapsed, "passed.elapsed")
  Run_Flag_Storage_Invariants(run_flags, "passed.run_flags")
  Summary_Handle_Invariants(summary, "passed.summary")
  milliseconds := int64(elapsed) / int64(time.MILLISECOND)
  findings.Line = append(Line(memory.Line[:0]), PASSED...)
  size := len(findings.Line)
  spare := strconv.Buffer(findings.Line[size : size+strconv.DIGIT_TEXT_SIZE_MAXIMUM])
  seconds := strconv.Unsigned_Integer(milliseconds / MILLISECONDS_PER_SECOND)
  count := strconv.Format_Unsigned_Integer_Into(spare, seconds, strconv.DECIMAL_BASE)
  findings.Line = append(findings.Line[:size+int(count)], '.')
  fraction := milliseconds % MILLISECONDS_PER_SECOND
  findings.Line = append(
    findings.Line,
    byte('0'+fraction/100),
    byte('0'+fraction/10%10),
    byte('0'+fraction%10),
    '\n',
  )
  if !run_flags.Values[RUN_JSON] {
    return
  }
  findings.Line = Line(summary_write(memory.Json, summary, Passed_Line(findings.Line)))
}

// The elapsed text already stands in the line, thus the summary views it rather than write it
// twice. The summary lands in run storage for the same reason the report does.
// PASSED_SIZE_MINIMUM is the shortest pass line: the text, one second digit and its fraction,
// and the newline that ends it.
const PASSED_SIZE_MINIMUM = len(PASSED) + ELAPSED_SIZE_MINIMUM + len("\n")

// PASSED_SIZE_MAXIMUM is the widest: every second digit one run can spend.
const PASSED_SIZE_MAXIMUM = len(PASSED) + ELAPSED_SIZE_MAXIMUM + len("\n")

// Passed_Line is the line a clean run wrote, which the summary reads its seconds back out of.
type Passed_Line []byte

// Passed_Line_Invariants bounds the pass line by the text and the seconds it carries.
func Passed_Line_Invariants(value Passed_Line, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(len(value), PASSED_SIZE_MINIMUM, PASSED_SIZE_MAXIMUM).
    Ensure()
}

// SUMMARY_FIXED_SIZE is every byte of a flat JSON summary but the seconds it carries.
const SUMMARY_FIXED_SIZE = len(`{"passed":true,"elapsed_sec":""}`)

// SUMMARY_SIZE_MINIMUM holds that summary at one second digit, and the newline that ends it.
const SUMMARY_SIZE_MINIMUM = SUMMARY_FIXED_SIZE + ELAPSED_SIZE_MINIMUM + len("\n")

// SUMMARY_SIZE_MAXIMUM holds it at every second digit one run can spend.
const SUMMARY_SIZE_MAXIMUM = SUMMARY_FIXED_SIZE + ELAPSED_SIZE_MAXIMUM + len("\n")

// Summary_Line is the flat JSON line a clean run writes, which views the JSON store.
type Summary_Line []byte

// Summary_Line_Invariants bounds the summary by the seconds it carries.
func Summary_Line_Invariants(value Summary_Line, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(len(value), SUMMARY_SIZE_MINIMUM, SUMMARY_SIZE_MAXIMUM).
    Ensure()
}

func summary_write(
  storage Json_Storage,
  summary Summary_Handle,
  passed Passed_Line,
) (written Summary_Line) {
  defer func() { Summary_Line_Invariants(written, "summary_write.written") }()
  Json_Storage_Invariants(storage, "summary_write.storage")
  Summary_Handle_Invariants(summary, "summary_write.summary")
  Passed_Line_Invariants(passed, "summary_write.passed")
  elapsed := passed[len(PASSED) : len(passed)-len("\n")]
  seconds_text := unsafe.String(unsafe.SliceData(elapsed), len(elapsed))
  *summary = Summary{Passed: true, Elapsed_Seconds: Elapsed_Text(seconds_text)}
  data, err := flatjson.Marshal_Into(
    flatjson.Output(storage[:flatjson.OUTPUT_SIZE_MAXIMUM]),
    flatjson.Value(reflect.ValueOf(summary)),
  )
  aver.Always(err == nil, "A pass summary fits flat JSON output.")
  line := storage[:len(data)+len("\n")]
  line[len(data)] = '\n'
  return Summary_Line(line)
}

// Writes the failure line of the step it takes. A step names a failure and nothing else, thus
// the caller states it here rather than carries it through every pass that never fails. A walk
// names the directory it had open, and any other step names the current file. The phase stays
// where the caller put it: every caller stands in a pass that already made the run ready, thus
// this states no phase and takes none.
func fail(
  memory Memory_Handle,
  findings Findings_Handle,
  path Failure_Path,
  step Step,
  run_flags Run_Flag_Storage,
  outcome Outcome_Handle,
) {
  Memory_Handle_Invariants(memory, "fail.memory")
  Findings_Handle_Invariants(findings, "fail.findings")
  Failure_Path_Invariants(path, "fail.path")
  Step_Invariants(step, "fail.step")
  Run_Flag_Storage_Invariants(run_flags, "fail.run_flags")
  Outcome_Handle_Invariants(outcome, "fail.outcome")
  path_steady(Untrusted_Name(path))
  line := append(Line(memory.Line[:0]), FAILURE_PREFIX...)
  switch step {
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
  // An argument past the path bound names no path, thus its line states the cause alone.
  if len(path) > 0 {
    line = append(append(line, SEPARATOR...), path...)
  }
  if step == STEP_WALK {
    cause := walk_cause(run_flags)
    openings := [...]string{
      WALK_CAUSE_REFUSED,
      "the argument names no directory",
      "more than ",
      "more than ",
      "more than ",
      "a name holds a byte below a space",
      "a path past ",
    }
    bounds := [...]int{
      0,
      0,
      FILE_COUNT_MAXIMUM,
      DIRECTORY_COUNT_MAXIMUM,
      ORPHAN_COUNT_MAXIMUM,
      0,
      filepath.PATH_SIZE_MAXIMUM,
    }
    tails := [...]string{
      "",
      "",
      " Go files",
      " directories pending",
      " broken symlinks",
      "",
      " bytes",
    }
    line = append(append(line, SEPARATOR...), openings[cause]...)
    if bounds[cause] > 0 {
      // The digits land in the line's spare capacity, since a conversion would allocate.
      size := len(line)
      spare := strconv.Buffer(line[size : size+strconv.DIGIT_TEXT_SIZE_MAXIMUM])
      bound := strconv.Unsigned_Integer(bounds[cause])
      line = line[:size+
        int(strconv.Format_Unsigned_Integer_Into(spare, bound, strconv.DECIMAL_BASE))]
    }
    line = append(line, tails[cause]...)
  }
  findings.Line = append(line, '\n')
  outcome.Status = EXIT_FAILURE
}

// Records only; Rearm judges, so no continuation runs inside the loop's own stack.
func runner_retired(completion nbio.Completion_Handle) {
  Runner_Handle(unsafe.Pointer(completion)).Phase = PHASE_RETIRED
}

// New_Memory allocates every store once, at its maximum, so the run never grows one. The parse
// state and
// its line index come back beside the stores, since the run holds them rather than its storage.
func New_Memory() (
  memory Memory_Handle,
  format_state Format,
  command Command,
) {
  defer func() {
    Memory_Handle_Invariants(memory, "new_memory.memory")
    Format_Invariants(format_state, "new_memory.format_state")
    Command_Invariants(command, "new_memory.command")
  }()
  memory = &Memory{}
  format_state.Printer, format_state.Formatter = new_print(), new_formatter()
  memory.Form = make(Form_Bytes, FORM_STORAGE_SIZE)
  // Every slot holds a whole path, and the three tables share one bound, thus one loop fills.
  memory.Paths = make(Path_Table, FILE_COUNT_MAXIMUM)
  memory.Directories = make(Directory_Paths, DIRECTORY_COUNT_MAXIMUM)
  memory.Packages = make(Packages, PACKAGE_COUNT_MAXIMUM)
  for index := range memory.Paths {
    memory.Paths[index] = make(filepath.Slice, 0, filepath.PATH_SIZE_MAXIMUM)
    memory.Directories[index] = make(filepath.Slice, 0, filepath.PATH_SIZE_MAXIMUM)
    memory.Packages[index].Path = make(Package_Path, 0, filepath.PATH_SIZE_MAXIMUM)
  }
  memory.File_Packages = make(Package_Indices, FILE_COUNT_MAXIMUM)
  memory.Entries = make(Entry_Table, ENTRY_COUNT_MAXIMUM)
  memory.Directory_Buffer = make(Entry_Buffer, nbio.DIRECTORY_BUFFER_SIZE_MAXIMUM)
  memory.Source = make(File_Bytes, SOURCE_STORAGE_SIZE)
  memory.Json = make(Json_Storage, JSON_STORAGE_SIZE)
  memory.Line = make(Line_Storage, LINE_SIZE_MAXIMUM)
  // A run states its first report before it has one to state, thus the leaves start at a
  // value every invariant of a report admits.
  arguments, flags := ARGUMENT_DECLARATION_COUNT, FLAG_DECLARATION_COUNT
  command.Declared_Arguments = make(Argument_Declarations, arguments)
  command.Declared_Flags = make(Flag_Declarations, flags)
  command.Command_Arguments = make(Parsed_Arguments, arguments)
  command.Command_Flags = make(Parsed_Flags, flags)
  command.Filled = make(Filled_Marks, FILLED_COUNT)
  tokens := TOKEN_COUNT_MAXIMUM
  command.Positionals = make(Positional_Tokens, tokens)
  command.Slice_Named = make(Named_Tokens, tokens)
  command.Failure = make(Failure_Bytes, cli.FAILURE_SIZE_MAXIMUM)
  command.Help = make(Help_Bytes, strings.TEXT_SIZE_MAXIMUM)
  memory.Requirements = new_checker(memory.Source)
  memory.History_Output = make(History_Output, HISTORY_OUTPUT_SIZE)
  memory.History_Errors = make(History_Errors, HISTORY_ERROR_SIZE)
  memory.History_Name = make(History_Name, HISTORY_NAME_SIZE)
  memory.Configuration = &configuration.Configuration{
    Source: make(configuration.Source_Store, configuration.SOURCE_STORAGE_SIZE),
    Text:   make(configuration.Text, configuration.SOURCE_SIZE_MAXIMUM),
    Values: make(configuration.Values, configuration.VALUE_COUNT_MAXIMUM),
    Cells:  make(configuration.Cells, configuration.CELL_COUNT),
  }
  memory.Notice = make(Notice_Store, MESSAGE_SIZE_MAXIMUM)
  memory.Orphans = make(Orphan_Paths, ORPHAN_COUNT_MAXIMUM)
  for index := range memory.Orphans {
    memory.Orphans[index] = make(filepath.Slice, 0, filepath.PATH_SIZE_MAXIMUM)
  }
  memory.Link = make(Link_Path, filepath.PATH_SIZE_MAXIMUM)
  memory.Resolved = make(Resolved_Path, filepath.PATH_SIZE_MAXIMUM)
  memory.Target = make(Link_Target, LINK_TARGET_SIZE)
  return memory, format_state, command
}

func new_print() (print printer.Printer_Handle) {
  defer func() { printer.Printer_Handle_Invariants(print, "new_print.print") }()
  pairs, imports := printer.PAIR_COUNT_MAXIMUM, printer.IMPORT_COUNT_MAXIMUM
  state := printer.Printer{}
  print = &state
  print.Nodes.Values = make(printer.Node_Stack, printer.DEPTH_MAXIMUM)
  print.Pairs.Functions = make(printer.Pair_Functions, pairs)
  print.Pairs.Heads = make(printer.Pair_Heads, pairs)
  print.Aliases.Values = make(printer.Import_Aliases, imports)
  print.Paths.Values = make(printer.Import_Paths, imports)
  print.Comments.Values = make(printer.Comment_Positions, printer.COMMENT_COUNT_MAXIMUM)
  print.Counts.Values = make(printer.Counts, printer.COUNT_SLOT_COUNT)
  print.Flags.Values = make(printer.Flags, printer.FLAG_COUNT)
  print.Source = &printer.Source_Storage{
    Values: make(printer.Source_Bytes, token.SOURCE_SIZE_MAXIMUM),
  }
  print.Form = &printer.Form_Storage{
    Values: make(printer.Form_Bytes, printer.FORM_SIZE_MAXIMUM),
  }
  return print
}

func new_formatter() (formatter Formatter_Stores) {
  defer func() { Formatter_Stores_Invariants(formatter, "new_formatter.formatter") }()
  formatter.Places.Values = make(format.Places, format.PLACE_SLOT_COUNT)
  formatter.Table.Slots.Values = make(format.Slots, format.SLOT_COUNT)
  formatter.Table.Names.Values = make(format.Names, format.NAME_SLOT_COUNT)
  formatter.Table.Texts.Values = make(format.Text, format.NAME_TEXT_SIZE_MAXIMUM)
  return formatter
}

func new_checker(source File_Bytes) (handle requirements.Checker_Handle) {
  defer func() {
    requirements.Checker_Handle_Invariants(handle, "new_checker.handle")
  }()
  File_Bytes_Invariants(source, "new_checker.source")
  checker := requirements.Checker{}
  handle = &checker
  count := requirements.HEADING_COUNT_MAXIMUM
  checker.Markdown = make(requirements.Markdown_Bytes, requirements.MARKDOWN_STORAGE_SIZE)
  checker.Rewritten = make(requirements.Rewritten_Bytes, requirements.MARKDOWN_STORAGE_SIZE)
  checker.Headings = make(requirements.Headings, count)
  checker.Leaves = make(requirements.Leaves, count)
  checker.Declarations = make(requirements.Declarations, count)
  checker.Name = make(requirements.Name_Bytes, requirements.NAME_SIZE_MAXIMUM)
  checker.Prior = make(requirements.Prior_Bytes, requirements.NAME_SIZE_MAXIMUM)
  checker.Message = make(requirements.Message_Bytes, requirements.MESSAGE_SIZE_MAXIMUM)
  checker.Path = make(requirements.Path_Bytes, requirements.PATH_STORAGE_SIZE)
  checker.Test = requirements.Test_Bytes(source)
  checker.Numbers.Values = make(requirements.Numbers, requirements.NUMBER_SLOT_COUNT)
  checker.Flags.Values = make(requirements.Flags, requirements.FLAG_SLOT_COUNT)
  return handle
}

// GIT_PATH is the executable the history check runs, which the host's PATH names.
const GIT_PATH = "git"

// HISTORY_STORAGE_SIZE bounds the log one run reads: a line per commit, thus some two thousand
// commits at a hundred bytes each.
const HISTORY_STORAGE_SIZE = 1 << 18

// HISTORY_ERROR_SIZE bounds what one git command writes to its standard error, which the run reads
// only as a failure.
const HISTORY_ERROR_SIZE = 4096

// HISTORY_TIMEOUT bounds one git command.
const HISTORY_TIMEOUT = 30 * time.SECOND

// HISTORY_HASH_SIZE is the digits of an object name one finding states, Git's own short width.
const HISTORY_HASH_SIZE = 10

// HISTORY_NAME_OPENING opens the name a commit finding stands at.
const HISTORY_NAME_OPENING = "<git:"

// HISTORY_NAME_SIZE is the name one commit finding stands at: the opening, the short object name,
// and the closing bracket.
const HISTORY_NAME_SIZE = len(HISTORY_NAME_OPENING) + HISTORY_HASH_SIZE + len(">")

// HISTORY_NAME_MAIN is the name a missing main ref stands at.
const HISTORY_NAME_MAIN = "<git:main>"

// HISTORY_NAME_LOG is the name a log past its storage stands at.
const HISTORY_NAME_LOG = "<git:log>"

// MESSAGE_HISTORY_SIZE states a subject past the size cap.
const MESSAGE_HISTORY_SIZE = "The commit subject is longer than 100 characters. " +
  "Write a shorter subject."

// MESSAGE_HISTORY_CONVENTIONAL states a subject off the conventional shape.
const MESSAGE_HISTORY_CONVENTIONAL = "The commit subject is not conventional. " +
  "Write \"type(scope): description\"."

// MESSAGE_HISTORY_FIXUP states a fixup commit.
const MESSAGE_HISTORY_FIXUP = "The commit is a fixup commit. Squash it into its target commit."

// MESSAGE_HISTORY_MERGE states a merge commit.
const MESSAGE_HISTORY_MERGE = "The commit is a merge commit. Rebase the branch onto main."

// MESSAGE_HISTORY_ABSENT states a repository that holds no main ref.
const MESSAGE_HISTORY_ABSENT = "The main ref is absent. Fetch main."

// MESSAGE_HISTORY_FULL states a log past its storage.
const MESSAGE_HISTORY_FULL = "The log is larger than its storage. Land the branch in parts."

// RULE_TEXT_COMMITS names a commit finding in flat JSON.
const RULE_TEXT_COMMITS Rule_Text = "commits"

// PROCESS_FAILED names a git command that exited failing, which the loop reports as no error.
var PROCESS_FAILED = errors.New("harness: git exited failing")

// History_Output is the store one git command writes its standard output into.
type History_Output []byte

// History_Output_Invariants fixes the store.
func History_Output_Invariants(value History_Output, _ aver.Namespace) {
  aver.Always(len(value) == HISTORY_OUTPUT_SIZE, "A history store has complete length.")
  aver.Always(cap(value) == HISTORY_OUTPUT_SIZE, "A history store cannot grow.")
}

// History_Errors is the store one git command writes its standard error into.
type History_Errors []byte

// History_Errors_Invariants fixes the store.
func History_Errors_Invariants(value History_Errors, _ aver.Namespace) {
  aver.Always(len(value) == HISTORY_ERROR_SIZE, "A history error store has complete length.")
  aver.Always(cap(value) == HISTORY_ERROR_SIZE, "A history error store cannot grow.")
}

// History_Name is the store the name of one commit finding stands in.
type History_Name []byte

// History_Name_Invariants fixes the store.
func History_Name_Invariants(value History_Name, _ aver.Namespace) {
  aver.Always(len(value) == HISTORY_NAME_SIZE, "A history name store has complete length.")
  aver.Always(cap(value) == HISTORY_NAME_SIZE, "A history name store cannot grow.")
}

// HISTORY_SKIP is a history the run judges nothing of: no repository, a branch main or master
// names, or a git that failed.
const HISTORY_SKIP History_Source = 0

// HISTORY_REMOTE is a history the run reads from origin/main.
const HISTORY_REMOTE History_Source = 1

// HISTORY_LOCAL is a history the run reads from main, since the clone holds no origin/main.
const HISTORY_LOCAL History_Source = 2

// HISTORY_ABSENT is a repository that holds no main ref.
const HISTORY_ABSENT History_Source = 3

// HISTORY_FULL is a log past its storage.
const HISTORY_FULL History_Source = 4

// History_Source is where the history check reads from, or why it reads nothing.
type History_Source int

// History_Source_Invariants admits each source.
func History_Source_Invariants(value History_Source, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(int(value), int(HISTORY_SKIP), int(HISTORY_FULL)).
    Ensure()
}

// History_Offset is the byte of the log the history check stands at: the start, or past a whole
// line that stated a finding, which spends its object name and more and closes on a newline git
// wrote inside its storage.
type History_Offset int

// History_Offset_Invariants bounds the offset by the log's store.
func History_Offset_Invariants(value History_Offset, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Holed_Int(
    int(value),
    slices.COUNT_MINIMUM,
    HISTORY_STORAGE_SIZE,
    HISTORY_OFFSET_HOLE_ONE,
    HISTORY_OFFSET_HOLE_TWO,
    HISTORY_OFFSET_HOLE_TWO,
    HISTORY_OFFSET_HOLE_TWO,
  ).
    Ensure()
}

// HISTORY_OFFSET_HOLE_ONE is one byte past the start, which no finding line spends.
const HISTORY_OFFSET_HOLE_ONE = 1

// HISTORY_OFFSET_HOLE_TWO is two bytes past it.
const HISTORY_OFFSET_HOLE_TWO = 2

// HISTORY_OUTPUT_SIZE is the log store: what git may write, and the byte that marks its end.
const HISTORY_OUTPUT_SIZE = HISTORY_STORAGE_SIZE + len("\x00")

// LOG_FORMAT asks git for one line per commit: its object name, its parents, a tab, the subject.
const LOG_FORMAT = "--format=%H %P%x09%s"

// Submitted_Source is where the history check reads from as the run submits a command: nowhere
// yet, or the ref the log reads from. A missing ref or a log past its storage submits no git.
type Submitted_Source History_Source

// Submitted_Source_Invariants admits the sources a submission stands on.
func Submitted_Source_Invariants(value Submitted_Source, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_3_Int(
    int(value),
    int(HISTORY_SKIP),
    int(HISTORY_REMOTE),
    int(HISTORY_LOCAL),
  ).
    Ensure()
}

// Judged_Source is where the history check reads from as the run judges a command: nowhere yet,
// or the ref the log reads from.
type Judged_Source History_Source

// Judged_Source_Invariants admits the sources a judgement stands on.
func Judged_Source_Invariants(value Judged_Source, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_3_Int(
    int(value),
    int(HISTORY_SKIP),
    int(HISTORY_REMOTE),
    int(HISTORY_LOCAL),
  ).
    Ensure()
}

// Judged_Source_Handle is the caller-owned source a judgement moves.
type Judged_Source_Handle *Judged_Source

// Judged_Source_Handle_Invariants composes a present source.
func Judged_Source_Handle_Invariants(value Judged_Source_Handle, namespace aver.Namespace) {
  if value == nil {
    return
  }
  Judged_Source_Invariants(*value, namespace)
}

// GIT_ARGUMENT_COUNT is the arguments the widest git command the run spawns takes.
const GIT_ARGUMENT_COUNT = 4

// GIT_HEAD_ARGUMENT_COUNT is the arguments the command that names HEAD's branch takes.
const GIT_HEAD_ARGUMENT_COUNT = 3

// GIT_SUBCOMMAND_SIZE_MAXIMUM is the widest subcommand the run spawns git with.
const GIT_SUBCOMMAND_SIZE_MAXIMUM = len("for-each-ref")

// GIT_OPTION_SIZE_MAXIMUM is the widest option the run spawns git with.
const GIT_OPTION_SIZE_MAXIMUM = len("--format=%(refname)")

// GIT_REFERENCE_SIZE_MAXIMUM is the widest third argument the run spawns git with.
const GIT_REFERENCE_SIZE_MAXIMUM = len(LOG_FORMAT)

// GIT_SPAN_SIZE_MAXIMUM is the widest fourth argument the run spawns git with.
const GIT_SPAN_SIZE_MAXIMUM = len("refs/remotes/origin/main")

// GIT_ARGUMENT_HOLE_ONE is one byte, which no argument the run spawns git with spends.
const GIT_ARGUMENT_HOLE_ONE = 1

// GIT_ARGUMENT_HOLE_TWO is two bytes, which none spends either.
const GIT_ARGUMENT_HOLE_TWO = 2

// Git_Subcommand is the subcommand of the git command the run spawned last, or none.
type Git_Subcommand string

// Git_Subcommand_Invariants bounds the subcommand by the widest the run spawns.
func Git_Subcommand_Invariants(value Git_Subcommand, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Holed_Int(
    len(value),
    slices.COUNT_MINIMUM,
    GIT_SUBCOMMAND_SIZE_MAXIMUM,
    GIT_ARGUMENT_HOLE_ONE,
    GIT_ARGUMENT_HOLE_TWO,
    GIT_ARGUMENT_HOLE_TWO,
    GIT_ARGUMENT_HOLE_TWO,
  ).
    Ensure()
}

// Git_Option is the option of the git command the run spawned last, or none.
type Git_Option string

// Git_Option_Invariants bounds the option by the widest the run spawns.
func Git_Option_Invariants(value Git_Option, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Holed_Int(
    len(value),
    slices.COUNT_MINIMUM,
    GIT_OPTION_SIZE_MAXIMUM,
    GIT_ARGUMENT_HOLE_ONE,
    GIT_ARGUMENT_HOLE_TWO,
    GIT_ARGUMENT_HOLE_TWO,
    GIT_ARGUMENT_HOLE_TWO,
  ).
    Ensure()
}

// Git_Reference is the third argument of the git command the run spawned last, or none.
type Git_Reference string

// Git_Reference_Invariants bounds the argument by the widest the run spawns.
func Git_Reference_Invariants(value Git_Reference, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Holed_Int(
    len(value),
    slices.COUNT_MINIMUM,
    GIT_REFERENCE_SIZE_MAXIMUM,
    GIT_ARGUMENT_HOLE_ONE,
    GIT_ARGUMENT_HOLE_TWO,
    GIT_ARGUMENT_HOLE_TWO,
    GIT_ARGUMENT_HOLE_TWO,
  ).
    Ensure()
}

// Git_Span is the fourth argument of the git command the run spawned last, or none. A command
// of three arguments leaves the one before it standing, which it does not pass.
type Git_Span string

// Git_Span_Invariants bounds the argument by the widest the run spawns.
func Git_Span_Invariants(value Git_Span, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Holed_Int(
    len(value),
    slices.COUNT_MINIMUM,
    GIT_SPAN_SIZE_MAXIMUM,
    GIT_ARGUMENT_HOLE_ONE,
    GIT_ARGUMENT_HOLE_TWO,
    GIT_ARGUMENT_HOLE_TWO,
    GIT_ARGUMENT_HOLE_TWO,
  ).
    Ensure()
}

// Git_Command is the argument vector of the git command the run spawned last, one field per
// argument in the order git reads them. A runner no command stood on holds none.
type Git_Command struct {
  // Subcommand is the first argument.
  Subcommand Git_Subcommand
  // Option is the second.
  Option Git_Option
  // Reference is the third.
  Reference Git_Reference
  // Span is the fourth.
  Span Git_Span
}

// Git_Command_Invariants composes one argument vector.
func Git_Command_Invariants(value Git_Command, namespace aver.Namespace) {
  Git_Subcommand_Invariants(value.Subcommand, namespace)
  Git_Option_Invariants(value.Option, namespace)
  Git_Reference_Invariants(value.Reference, namespace)
  Git_Span_Invariants(value.Span, namespace)
}

// Git_Command_Handle is the caller-owned argument vector a spawn writes.
type Git_Command_Handle *Git_Command

// Git_Command_Handle_Invariants composes a present argument vector.
func Git_Command_Handle_Invariants(value Git_Command_Handle, namespace aver.Namespace) {
  if value == nil {
    return
  }
  Git_Command_Invariants(*value, namespace)
}

// History_Walk is where the history check stands: where it reads from, and how far through the
// log it read.
type History_Walk struct {
  // Source is where the check reads from, or why it reads nothing.
  Source History_Source
  // Offset is the line of the log the check judges next.
  Offset History_Offset
}

// History_Walk_Invariants composes where the history check stands.
func History_Walk_Invariants(value History_Walk, namespace aver.Namespace) {
  History_Source_Invariants(value.Source, namespace)
  History_Offset_Invariants(value.Offset, namespace)
}

// History_Walk_Handle is caller-owned walk state.
type History_Walk_Handle *History_Walk

// History_Walk_Handle_Invariants composes present walk state.
func History_Walk_Handle_Invariants(value History_Walk_Handle, namespace aver.Namespace) {
  if value == nil {
    return
  }
  History_Walk_Invariants(*value, namespace)
}

// History is the history check: the walk, and the argument vector of the git it spawned last.
type History struct {
  // Walk is where the check stands.
  Walk History_Walk
  // Command is the argument vector of the git the check spawned last.
  Command Git_Command
}

// History_Invariants composes the history check.
func History_Invariants(value History, namespace aver.Namespace) {
  History_Walk_Invariants(value.Walk, namespace)
  Git_Command_Invariants(value.Command, namespace)
}

// History_Handle is caller-owned history state.
type History_Handle *History

// History_Handle_Invariants composes present history state.
func History_Handle_Invariants(value History_Handle, namespace aver.Namespace) {
  if value == nil {
    return
  }
  History_Invariants(*value, namespace)
}

// History_Stage is a stage that spawns git.
type History_Stage Stage

// History_Stage_Invariants admits the three git commands.
func History_Stage_Invariants(value History_Stage, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_3_Int(
    int(value),
    int(STAGE_HISTORY_HEAD),
    int(STAGE_HISTORY_REFS),
    int(STAGE_HISTORY_LOG),
  ).
    Ensure()
}

// History_Pass is the stage one git command leaves the run on: the next command, or the walk of
// what the commands read.
type History_Pass Stage

// History_Pass_Invariants admits the stages one git command leaves.
func History_Pass_Invariants(value History_Pass, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_3_Int(
    int(value),
    int(STAGE_HISTORY_NEXT),
    int(STAGE_HISTORY_REFS),
    int(STAGE_HISTORY_LOG),
  ).
    Ensure()
}

// Next_Pass is the stage one step of the history walk leaves the run on: the write of a finding,
// or the finish of the run.
type Next_Pass Stage

// Next_Pass_Invariants admits the stages one step of the walk leaves.
func Next_Pass_Invariants(value Next_Pass, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_3_Int(
    int(value),
    int(STAGE_REPORT),
    int(STAGE_FAIL),
    int(STAGE_PACKAGE),
  ).
    Ensure()
}

// Records what one git command did, and leaves the run to judge it. A command that exited failing
// reads as a failure, which the loop reports as none.
func runner_spawned(completion nbio.Completion_Handle, result nbio.Process_Result, err error) {
  completion.Data, completion.Error = len(result.Output), err
  if err == nil {
    if result.Exit != 0 {
      completion.Error = PROCESS_FAILED
    }
  }
  Runner_Handle(unsafe.Pointer(completion)).Phase = PHASE_RETIRED
}

// Spawns the git command the stage names, into the run's own stores.
func submit_history(
  memory Memory_Handle,
  loop Loop_Handle,
  stage History_Stage,
  source Submitted_Source,
  command Git_Command_Handle,
) {
  Memory_Handle_Invariants(memory, "submit_history.memory")
  Loop_Handle_Invariants(loop, "submit_history.loop")
  History_Stage_Invariants(stage, "submit_history.stage")
  Submitted_Source_Invariants(source, "submit_history.source")
  Git_Command_Handle_Invariants(command, "submit_history.command")
  count := GIT_ARGUMENT_COUNT
  switch Stage(stage) {
  case STAGE_HISTORY_HEAD:
    command.Subcommand, command.Option, command.Reference = "rev-parse", "--abbrev-ref", "HEAD"
    count = GIT_HEAD_ARGUMENT_COUNT
  case STAGE_HISTORY_REFS:
    command.Subcommand, command.Option = "for-each-ref", "--format=%(refname)"
    command.Reference, command.Span = "refs/heads/main", "refs/remotes/origin/main"
  case STAGE_HISTORY_LOG:
    command.Subcommand, command.Option, command.Reference = "log", "--first-parent", LOG_FORMAT
    command.Span = "main..HEAD"
    if source == Submitted_Source(HISTORY_REMOTE) {
      command.Span = "origin/main..HEAD"
    }
  }
  // The fields stand one per argument, in order, each a string, thus they lie one after another
  // as the vector the loop takes, which a view of them states without an allocation.
  arguments := unsafe.Slice((*string)(unsafe.Pointer(&command.Subcommand)), count)
  // The store keeps one byte past what git may write, thus the end of the log stands marked.
  request := nbio.Process_Request{
    Path:                 GIT_PATH,
    Arguments:            arguments,
    Error_Output_Storage: memory.History_Errors,
    Output_Storage:       memory.History_Output[:HISTORY_STORAGE_SIZE],
  }
  // The loop names the callback's own types, thus it stands as a literal that hands them on
  // and captures nothing, so allocates nothing.
  nbio.IO_Spawn(
    loop.IO,
    &loop.Completion,
    request,
    HISTORY_TIMEOUT,
    func(completion *nbio.Completion, result nbio.Process_Result, err error) {
      runner_spawned(nbio.Completion_Handle(completion), result, err)
    },
  )
}

// Judges what one git command read: the branch HEAD stands on, the main refs the repository
// holds, or the log between the main ref and HEAD. A git that failed reads as no history at all,
// since a run off a repository has none to judge.
func judge_history(
  memory Memory_Handle,
  loop Loop_Handle,
  stage History_Stage,
  source Judged_Source_Handle,
) (next History_Pass) {
  defer func() { History_Pass_Invariants(next, "judge_history.next") }()
  Memory_Handle_Invariants(memory, "judge_history.memory")
  Loop_Handle_Invariants(loop, "judge_history.loop")
  History_Stage_Invariants(stage, "judge_history.stage")
  Judged_Source_Handle_Invariants(source, "judge_history.source")
  output := string(memory.History_Output[:loop.Completion.Data])
  failed := loop.Completion.Error != nil
  chosen := *source
  *source = Judged_Source(HISTORY_SKIP)
  if Stage(stage) == STAGE_HISTORY_LOG {
    if loop.Completion.Error == nbio.Process_Output_Too_Large {
      *source = Judged_Source(HISTORY_FULL)
      return History_Pass(STAGE_HISTORY_NEXT)
    }
    // Git writes no zero byte into the lines the run asks for, thus one marks the end.
    if !failed {
      *source = chosen
      memory.History_Output[len(output)] = 0
    }
    return History_Pass(STAGE_HISTORY_NEXT)
  }
  if failed {
    return History_Pass(STAGE_HISTORY_NEXT)
  }
  if Stage(stage) == STAGE_HISTORY_HEAD {
    // A branch main or master names is the history every other branch lands on.
    switch output {
    case "", "\n", "main\n", "master\n":
      return History_Pass(STAGE_HISTORY_NEXT)
    }
    return History_Pass(STAGE_HISTORY_REFS)
  }
  // Git lists the refs it holds by name, thus the answer stands in one of four forms. The
  // remote ref comes first: a clone's own main can lag it, and the branch lands on it.
  switch output {
  case "refs/heads/main\n":
    *source = Judged_Source(HISTORY_LOCAL)
  case "refs/remotes/origin/main\n", "refs/heads/main\nrefs/remotes/origin/main\n":
    *source = Judged_Source(HISTORY_REMOTE)
  default:
    *source = Judged_Source(HISTORY_ABSENT)
    return History_Pass(STAGE_HISTORY_NEXT)
  }
  return History_Pass(STAGE_HISTORY_LOG)
}

// HISTORY_KIND_NONE is a walk that found nothing left to state.
const HISTORY_KIND_NONE History_Kind = 0

// HISTORY_KIND_ABSENT is a repository that holds no main ref. The kinds below it are the verdicts
// of harness/pkg/vcs, which convert.
const HISTORY_KIND_ABSENT History_Kind = 5

// HISTORY_KIND_FULL is a log past its storage.
const HISTORY_KIND_FULL History_Kind = 6

// History_Kind is what the next finding of the history states, or that none is left.
type History_Kind int

// History_Kind_Invariants admits each kind.
func History_Kind_Invariants(value History_Kind, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(int(value), int(HISTORY_KIND_NONE), int(HISTORY_KIND_FULL)).
    Ensure()
}

// Past is what the run states past its packages besides the history: the findings of
// harness.json, the links the walk found whose target resolves to nothing, and the outcome a line
// past flat JSON fails.
type Past struct {
  // Orphans is the orphan the run states next.
  Orphans Orphan_Cursor_Handle
  // Walk is what the walk collected.
  Walk Walk_Handle
  // Loop is the surface the survey asks the host through.
  Loop Loop_Handle
  // Outcome is what the run exits as.
  Outcome Outcome_Handle
}

// Past_Invariants composes what the run states past its packages.
func Past_Invariants(value Past, namespace aver.Namespace) {
  Orphan_Cursor_Handle_Invariants(value.Orphans, namespace)
  Walk_Handle_Invariants(value.Walk, namespace)
  Loop_Handle_Invariants(value.Loop, namespace)
  Outcome_Handle_Invariants(value.Outcome, namespace)
}

// States the next finding past the packages, as a text line or a flat JSON line, at the name it
// stands at: each finding of harness.json, each link whose target resolves to nothing, then each
// finding of the history. A run with nothing left to state finishes.
func history_next(
  memory Memory_Handle,
  findings Findings_Handle,
  report Report_Handle,
  tally Tally_Handle,
  run_flags Run_Flag_Storage,
  history History_Walk_Handle,
  past Past,
) (next Next_Pass) {
  defer func() { Next_Pass_Invariants(next, "history_next.next") }()
  Memory_Handle_Invariants(memory, "history_next.memory")
  Findings_Handle_Invariants(findings, "history_next.findings")
  Report_Handle_Invariants(report, "history_next.report")
  Tally_Handle_Invariants(tally, "history_next.tally")
  Run_Flag_Storage_Invariants(run_flags, "history_next.run_flags")
  History_Walk_Handle_Invariants(history, "history_next.history")
  Past_Invariants(past, "history_next.past")
  if !bool(past_pick(memory, report, run_flags, history, past)) {
    return Next_Pass(STAGE_PACKAGE)
  }
  tally.Found, tally.Resume = true, RESUME_HISTORY
  if run_flags.Values[RUN_JSON] {
    // Only a path or an entry of bytes JSON escapes six-fold outgrows the object.
    if !bool(report_write(memory, findings, report)) {
      fail(memory, findings, Failure_Path(report.Path), STEP_REPORT, run_flags, past.Outcome)
      return Next_Pass(STAGE_FAIL)
    }
    return Next_Pass(STAGE_REPORT)
  }
  findings.Line = append(append(Line(memory.Line[:0]), report.Path...), SEPARATOR...)
  findings.Line = append(append(findings.Line, report.Message...), '\n')
  return Next_Pass(STAGE_REPORT)
}

// Picks the next finding past the packages into the report, and reports whether one stands.
func past_pick(
  memory Memory_Handle,
  report Report_Handle,
  run_flags Run_Flag_Storage,
  history History_Walk_Handle,
  past Past,
) (picked Boolean) {
  defer func() { Boolean_Invariants(picked, "past_pick.picked") }()
  Memory_Handle_Invariants(memory, "past_pick.memory")
  Report_Handle_Invariants(report, "past_pick.report")
  Run_Flag_Storage_Invariants(run_flags, "past_pick.run_flags")
  History_Walk_Handle_Invariants(history, "past_pick.history")
  Past_Invariants(past, "past_pick.past")
  // A run that states nothing past its packages leaves the report the last finding wrote, which
  // the run after it opens on.
  fix := survey_next(memory, past.Loop)
  if fix != FIX_NONE {
    notice := survey_message(memory, Stated_Fix(fix))
    *report = Report{Path: CONFIGURATION_PATH, Tier: Report_Tier(TIER_TWO), Rule: RULE_TEXT_SLASHES}
    report.Message = Message_Text(notice)
    return true
  }
  if int(past.Orphans.Orphan_Index) < int(past.Walk.Orphan_Count) {
    path := memory.Orphans[past.Orphans.Orphan_Index]
    *report = Report{Tier: Report_Tier(TIER_TWO), Rule: RULE_TEXT_SYMLINKS, Message: MESSAGE_ORPHAN}
    report.Path = Path_Text(unsafe.String(unsafe.SliceData(path), len(path)))
    past.Orphans.Orphan_Index++
    return true
  }
  kind := history_finding(memory, run_flags, history)
  if kind == HISTORY_KIND_NONE {
    return false
  }
  stored := memory.History_Name
  *report = Report{Tier: Report_Tier(TIER_TWO)}
  report.Path = Path_Text(unsafe.String(unsafe.SliceData(stored), len(stored)))
  switch kind {
  case HISTORY_KIND_ABSENT:
    report.Path = HISTORY_NAME_MAIN
  case HISTORY_KIND_FULL:
    report.Path = HISTORY_NAME_LOG
  }
  messages := [...]Message_Text{
    "",
    MESSAGE_HISTORY_SIZE,
    MESSAGE_HISTORY_CONVENTIONAL,
    MESSAGE_HISTORY_FIXUP,
    MESSAGE_HISTORY_MERGE,
    MESSAGE_HISTORY_ABSENT,
    MESSAGE_HISTORY_FULL,
  }
  report.Rule, report.Message = RULE_TEXT_COMMITS, messages[kind]
  return true
}

// Walks the history to its next finding. A history with nothing to judge, or none left, states
// none, and a missing ref or a log past its storage states one and ends the walk, thus the source
// stands until the next run opens.
func history_finding(
  memory Memory_Handle,
  run_flags Run_Flag_Storage,
  history History_Walk_Handle,
) (kind History_Kind) {
  defer func() { History_Kind_Invariants(kind, "history_finding.kind") }()
  Memory_Handle_Invariants(memory, "history_finding.memory")
  Run_Flag_Storage_Invariants(run_flags, "history_finding.run_flags")
  History_Walk_Handle_Invariants(history, "history_finding.history")
  if run_flags.Values[RUN_HISTORY_DONE] {
    return HISTORY_KIND_NONE
  }
  kind = history_kind(memory, history)
  switch kind {
  case HISTORY_KIND_NONE, HISTORY_KIND_ABSENT, HISTORY_KIND_FULL:
    run_flags.Values[RUN_HISTORY_DONE] = true
  }
  return kind
}

// Finds the next finding of the history: the missing main ref, the log past its storage, or the
// next commit of the log that breaks a rule, whose name the name store then holds.
func history_kind(memory Memory_Handle, history History_Walk_Handle) (kind History_Kind) {
  defer func() { History_Kind_Invariants(kind, "history_kind.kind") }()
  Memory_Handle_Invariants(memory, "history_kind.memory")
  History_Walk_Handle_Invariants(history, "history_kind.history")
  switch history.Source {
  case HISTORY_SKIP:
    return HISTORY_KIND_NONE
  case HISTORY_ABSENT:
    return HISTORY_KIND_ABSENT
  case HISTORY_FULL:
    return HISTORY_KIND_FULL
  }
  log := memory.History_Output
  for range HISTORY_STORAGE_SIZE {
    line := log[history.Offset:]
    for index := range len(line) {
      if line[index] == '\n' {
        line = line[:index]
        break
      }
      if line[index] == 0 {
        line = line[:index]
        break
      }
    }
    ended := log[int(history.Offset)+len(line)] == 0
    history.Offset += History_Offset(len(line) + 1)
    verdict := history_verdict(memory, History_Line(line))
    if ended {
      // A walk that reached the end stands at the start again, and judges nothing more.
      history.Source, history.Offset = HISTORY_SKIP, 0
    }
    if verdict != vcs.VERDICT_NONE {
      return History_Kind(verdict)
    }
    if ended {
      break
    }
  }
  return HISTORY_KIND_NONE
}

// History_Line is one line of the log: an object name, its parents, a tab, and the subject.
type History_Line []byte

// History_Line_Invariants bounds the line by the log's store.
func History_Line_Invariants(value History_Line, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(len(value), slices.COUNT_MINIMUM, HISTORY_STORAGE_SIZE).
    Ensure()
}

// Judges one line of the log, and names its commit in the name store where it breaks a rule. A
// line git did not write in the form the run asked for names no commit, thus it states nothing.
func history_verdict(memory Memory_Handle, line History_Line) (verdict vcs.Verdict) {
  defer func() { vcs.Verdict_Invariants(verdict, "history_verdict.verdict") }()
  Memory_Handle_Invariants(memory, "history_verdict.memory")
  History_Line_Invariants(line, "history_verdict.line")
  space, tab := -1, -1
  for index := range len(line) {
    if line[index] == '\t' {
      tab = index
      break
    }
    if line[index] == ' ' {
      if space < 0 {
        space = index
      }
    }
  }
  if space < HISTORY_HASH_SIZE {
    return vcs.VERDICT_NONE
  }
  if tab < space {
    return vcs.VERDICT_NONE
  }
  for index := range HISTORY_HASH_SIZE {
    digit := line[index] > '0'-1
    if digit {
      digit = line[index] < '9'+1
    }
    letter := line[index] > 'a'-1
    if letter {
      letter = line[index] < 'f'+1
    }
    if !digit {
      if !letter {
        return vcs.VERDICT_NONE
      }
    }
  }
  subject := line[tab+1:]
  if len(subject) == 0 {
    return vcs.VERDICT_NONE
  }
  // A parent past the first stands after a space of its own, thus the commit merged.
  merged := false
  for index := space + 1; index < tab; index++ {
    if line[index] == ' ' {
      merged = true
    }
  }
  name := append(memory.History_Name[:0], HISTORY_NAME_OPENING...)
  name = append(append(name, line[:HISTORY_HASH_SIZE]...), '>')
  // A subject past the judge's bound is past the size cap whatever it holds.
  if len(subject) > vcs.SUBJECT_TEXT_SIZE_MAXIMUM {
    return vcs.VERDICT_SIZE
  }
  view := unsafe.String(unsafe.SliceData(subject), len(subject))
  return vcs.Judge(vcs.Subject(view), vcs.Merged(merged))
}

// RULE_TEXT_SLASHES names a harness.json path entry whose trailing slash states the other kind of
// path than the host holds, or one the host holds nothing at.
const RULE_TEXT_SLASHES Rule_Text = "slashes"

// FIX_NONE is an entry the survey states nothing of.
const FIX_NONE Entry_Fix = 0

// FIX_ABSENT is a wildcard-free entry the host holds nothing at.
const FIX_ABSENT Entry_Fix = 1

// FIX_DIRECTORY is a wildcard-free entry naming a directory without a trailing slash.
const FIX_DIRECTORY Entry_Fix = 2

// FIX_FILE is a wildcard-free entry naming a file with a trailing slash.
const FIX_FILE Entry_Fix = 3

// FIX_LISTED is a wildcard-free entry naming a file in a list that names packages.
const FIX_LISTED Entry_Fix = 4

// Entry_Fix is what the survey states of one harness.json entry.
type Entry_Fix int

// Entry_Fix_Invariants admits each finding the slash rule states, and none.
func Entry_Fix_Invariants(value Entry_Fix, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(int(value), int(FIX_NONE), int(FIX_LISTED)).
    Ensure()
}

// Stated_Fix is a finding the survey states.
type Stated_Fix Entry_Fix

// Stated_Fix_Invariants admits each finding but none.
func Stated_Fix_Invariants(value Stated_Fix, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_4_Int(
    int(value),
    int(FIX_ABSENT),
    int(FIX_DIRECTORY),
    int(FIX_FILE),
    int(FIX_LISTED),
  ).
    Ensure()
}

// NOTICE_OPENING opens each finding of harness.json.
const NOTICE_OPENING = "The "

// NOTICE_ENTRY stands between the key and the entry it quotes.
const NOTICE_ENTRY = " entry \""

// NOTICE_CLOSE closes the entry a finding quotes.
const NOTICE_CLOSE = "\" "

// NOTICE_ABSENT closes the finding of an entry the host holds nothing at.
const NOTICE_ABSENT = "matches no file and no directory."

// NOTICE_LISTED closes the finding of a file in a list that names packages.
const NOTICE_LISTED = "names a file. Only ignore and opt_out_recursion_ban can name a file."

// NOTICE_OPENING_SIZE_MAXIMUM is the longest text a finding opens with, before its entry.
const NOTICE_OPENING_SIZE_MAXIMUM = len(NOTICE_OPENING) +
  configuration.KEY_NAME_SIZE_MAXIMUM +
  len(NOTICE_ENTRY)

// QUOTE_SIZE_MAXIMUM is how much of an entry a finding quotes: the rest of the message store,
// since an entry is untrusted and may run past any line a host reads.
const QUOTE_SIZE_MAXIMUM = MESSAGE_SIZE_MAXIMUM -
  NOTICE_OPENING_SIZE_MAXIMUM -
  len(NOTICE_CLOSE) -
  len(NOTICE_LISTED)

// NOTICE_SIZE_MINIMUM is the shortest finding: an entry of one byte in the shortest list, which
// names nothing.
const NOTICE_SIZE_MINIMUM = len(NOTICE_OPENING) +
  configuration.KEY_NAME_SIZE_MINIMUM +
  len(NOTICE_ENTRY) +
  len("/") +
  len(NOTICE_CLOSE) +
  len(NOTICE_ABSENT)

// Notice_Store holds the message of one finding of harness.json.
type Notice_Store []byte

// Notice_Store_Invariants fixes the store.
func Notice_Store_Invariants(value Notice_Store, _ aver.Namespace) {
  aver.Always(len(value) == MESSAGE_SIZE_MAXIMUM, "A notice store has complete length.")
  aver.Always(cap(value) == MESSAGE_SIZE_MAXIMUM, "A notice store cannot grow.")
}

// Notice_Text is the message of one finding of harness.json, as the notice store holds it.
type Notice_Text string

// Notice_Text_Invariants bounds the message by the shortest finding and the store.
func Notice_Text_Invariants(value Notice_Text, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(len(value), NOTICE_SIZE_MINIMUM, MESSAGE_SIZE_MAXIMUM).
    Ensure()
}

// Finds the next finding of the slash rule over each harness.json path list, in the order lint
// checks them, and states it, or FIX_NONE past the last entry.
func survey_next(memory Memory_Handle, loop Loop_Handle) (fix Entry_Fix) {
  defer func() { Entry_Fix_Invariants(fix, "survey_next.fix") }()
  Memory_Handle_Invariants(memory, "survey_next.memory")
  Loop_Handle_Invariants(loop, "survey_next.loop")
  for range configuration.VALUE_COUNT_MAXIMUM {
    if !bool(configuration.Entry_Next(memory.Configuration)) {
      return FIX_NONE
    }
    fix = entry_slash(memory, loop)
    if fix != FIX_NONE {
      return fix
    }
  }
  return FIX_NONE
}

// Applies the slash rule to the entry configuration.Entry_Next stands on: a wildcard-free entry
// names a
// directory with a trailing slash and a file without, as the host states the path, and a list
// that names packages names no file. The host stands in for lint's tracked tree.
func entry_slash(memory Memory_Handle, loop Loop_Handle) (fix Entry_Fix) {
  defer func() { Entry_Fix_Invariants(fix, "entry_slash.fix") }()
  Memory_Handle_Invariants(memory, "entry_slash.memory")
  Loop_Handle_Invariants(loop, "entry_slash.loop")
  decoded := memory.Configuration
  value := decoded.Values[decoded.Cells[configuration.CELL_ENTRY]]
  text := decoded.Text[value.Start:value.End]
  // A wildcard entry states a shape rather than one path.
  for _, symbol := range text {
    if symbol == '*' {
      return FIX_NONE
    }
  }
  start, end, slashed := 0, len(text), false
  if start < end {
    if text[start] == '!' {
      start++
    }
  }
  if start < end {
    if text[start] == '/' {
      start++
    }
  }
  if start < end {
    slashed = text[end-1] == '/'
  }
  if slashed {
    end--
  }
  directory, file := Boolean(false), Boolean(false)
  if end-start < filepath.PATH_SIZE_MAXIMUM+1 {
    directory, file = entry_status(loop, Entry_Literal(text[start:end]))
  }
  listed := value.Key != configuration.KEY_IGNORE
  if value.Key == configuration.KEY_RECURSION_EXEMPT {
    listed = false
  }
  fix = FIX_NONE
  if !bool(directory) {
    if !bool(file) {
      fix = FIX_ABSENT
    }
  }
  if bool(directory) {
    if !slashed {
      fix = FIX_DIRECTORY
    }
  }
  if bool(file) {
    if slashed {
      fix = FIX_FILE
    }
    if listed {
      fix = FIX_LISTED
    }
  }
  return fix
}

// Entry_Literal is the path an entry names, without its negation and slashes. One wider than a
// path names nothing the host holds, thus the survey judges it before it takes this form.
type Entry_Literal []byte

// Entry_Literal_Invariants bounds the literal by the text it stands in.
func Entry_Literal_Invariants(value Entry_Literal, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(
    len(value),
    bytes.SLICE_SIZE_MINIMUM,
    filepath.PATH_SIZE_MAXIMUM,
  ).
    Ensure()
}

// Reports whether the host holds a directory or a file at the literal, without following a link:
// a link is a file of its own, as git tracks one. A literal of no byte, or one wider than a path,
// names neither, and so does one the host refuses.
func entry_status(loop Loop_Handle, literal Entry_Literal) (directory Boolean, file Boolean) {
  defer func() {
    Boolean_Invariants(directory, "entry_status.directory")
    Boolean_Invariants(file, "entry_status.file")
  }()
  Loop_Handle_Invariants(loop, "entry_status.loop")
  Entry_Literal_Invariants(literal, "entry_status.literal")
  if len(literal) == 0 {
    return false, false
  }
  stated := unsafe.String(unsafe.SliceData(literal), len(literal))
  status, status_err := nbio.Storage_Status(loop.IO.Storage, stated)
  if status_err != nil {
    return false, false
  }
  if !status.Exists {
    return false, false
  }
  directory = Boolean(nbio.File_Mode_Is_Directory(status.Mode))
  return directory, !directory
}

// Writes the message of a finding into the notice store: the key, the entry it quotes, and what
// the finding states, in lint's words. The entry is untrusted, thus each byte below a space
// stands as ?, and the quote stops where the store does.
func survey_message(memory Memory_Handle, fix Stated_Fix) (notice Notice_Text) {
  defer func() { Notice_Text_Invariants(notice, "survey_message.notice") }()
  Memory_Handle_Invariants(memory, "survey_message.memory")
  Stated_Fix_Invariants(fix, "survey_message.fix")
  decoded := memory.Configuration
  value := decoded.Values[decoded.Cells[configuration.CELL_ENTRY]]
  written := append(memory.Notice[:0], NOTICE_OPENING...)
  written = append(written, configuration.Key_Text(value.Key)...)
  written = append(written, NOTICE_ENTRY...)
  entry := decoded.Text[value.Start:value.End]
  for _, symbol := range entry[:min(len(entry), QUOTE_SIZE_MAXIMUM)] {
    if symbol < NAME_BYTE_MINIMUM {
      symbol = requirements.QUOTED_BYTE_ABSENT
    }
    written = append(written, symbol)
  }
  written = append(written, NOTICE_CLOSE...)
  tails := [...]string{
    "",
    NOTICE_ABSENT,
    "names a directory. Add a trailing slash.",
    "names a file. Remove the trailing slash.",
    NOTICE_LISTED,
  }
  written = append(written, tails[fix]...)
  return Notice_Text(unsafe.String(unsafe.SliceData(written), len(written)))
}

// IMPORT_OPENING opens the message of a banned import.
const IMPORT_OPENING = "Banned import \""

// IMPORT_FAMILY stands between the import and the family it belongs to.
const IMPORT_FAMILY = "\" belongs to stdlib family \""

// IMPORT_TAIL closes the message of a banned import.
const IMPORT_TAIL = "\". Use shared package."

// IMPORT_MESSAGE_SIZE_MINIMUM is the shortest message: os, of the os family.
const IMPORT_MESSAGE_SIZE_MINIMUM = len(IMPORT_OPENING) +
  len("os") +
  len(IMPORT_FAMILY) +
  len("os") +
  len(IMPORT_TAIL)

// IMPORT_MESSAGE_SIZE_MAXIMUM is the longest message: the notice store less the zero byte that
// closes the message.
const IMPORT_MESSAGE_SIZE_MAXIMUM = MESSAGE_SIZE_MAXIMUM - 1

// IMPORT_QUOTE_SIZE_MAXIMUM is how much of an import a message quotes: the rest of the message
// past the longest family, since the path is untrusted.
const IMPORT_QUOTE_SIZE_MAXIMUM = IMPORT_MESSAGE_SIZE_MAXIMUM -
  len(IMPORT_OPENING) -
  len(IMPORT_FAMILY) -
  FAMILY_NAME_SIZE_MAXIMUM -
  len(IMPORT_TAIL)

// Import_Message is the message of a banned import, as the notice store holds it.
type Import_Message string

// Import_Message_Invariants bounds the message by the shortest import and the store.
func Import_Message_Invariants(value Import_Message, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(
    len(value),
    IMPORT_MESSAGE_SIZE_MINIMUM,
    IMPORT_MESSAGE_SIZE_MAXIMUM,
  ).
    Ensure()
}

// FAMILY_NONE is an import lint lets a file make.
const FAMILY_NONE Import_Family = 0

// FAMILY_OS is the os family, which three gateways bind.
const FAMILY_OS Import_Family = 1

// FAMILY_RANDOM is every home of rand.
const FAMILY_RANDOM Import_Family = 2

// FAMILY_CRYPTO is the crypto paths shared packages stand in for.
const FAMILY_CRYPTO Import_Family = 3

// FAMILY_DATABASE is the database paths shared packages stand in for.
const FAMILY_DATABASE Import_Family = 4

// FAMILY_GO is the go paths shared packages stand in for.
const FAMILY_GO Import_Family = 5

// FAMILY_HASH is the hash paths shared packages stand in for.
const FAMILY_HASH Import_Family = 6

// FAMILY_NETWORK is the net package, which a shared package stands in for.
const FAMILY_NETWORK Import_Family = 7

// FAMILY_PATH is the path packages shared packages stand in for.
const FAMILY_PATH Import_Family = 8

// FAMILY_SORT is the sort package, which a shared package stands in for.
const FAMILY_SORT Import_Family = 9

// FAMILY_TEXT is the text paths shared packages stand in for.
const FAMILY_TEXT Import_Family = 10

// FAMILY_ARCHIVE opens the families lint bans whole, each path under them included.
const FAMILY_ARCHIVE Import_Family = 11

// FAMILY_UUID is the last family lint bans whole.
const FAMILY_UUID Import_Family = 24

// Import_Family is the stdlib family a banned import belongs to.
type Import_Family int

// Import_Family_Invariants admits each family, and none.
func Import_Family_Invariants(value Import_Family, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(int(value), int(FAMILY_NONE), int(FAMILY_UUID)).
    Ensure()
}

// Stated_Family is the family of an import the check states.
type Stated_Family Import_Family

// Stated_Family_Invariants admits each family but none.
func Stated_Family_Invariants(value Stated_Family, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(int(value), int(FAMILY_OS), int(FAMILY_UUID)).
    Ensure()
}

// FAMILY_NAME_SIZE_MINIMUM is the shortest family name.
const FAMILY_NAME_SIZE_MINIMUM = len("go")

// FAMILY_NAME_SIZE_MAXIMUM is the longest family name.
const FAMILY_NAME_SIZE_MAXIMUM = len("container")

// Family_Name is how lint spells one family.
type Family_Name string

// Family_Name_Invariants bounds the name by the families.
func Family_Name_Invariants(value Family_Name, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(
    len(value),
    FAMILY_NAME_SIZE_MINIMUM,
    FAMILY_NAME_SIZE_MAXIMUM,
  ).
    Ensure()
}

// Checks the imports of the Go file the run read as lint does, and states the first banned one:
// a family lint bans whole, or an exact path a shared package stands in for. The aver subtree
// imports what it likes, and a gateway binds the process. Lint gates a tier two rule behind tier
// one, thus a file the check already faulted keeps that diagnostic alone.
func imports_check(
  memory Memory_Handle,
  syntax requirements.Syntax,
  slot File_Slot,
  count Source_Count,
  checked Diagnostic,
) (diagnostic Diagnostic) {
  defer func() { Diagnostic_Invariants(diagnostic, "imports_check.diagnostic") }()
  Memory_Handle_Invariants(memory, "imports_check.memory")
  requirements.Syntax_Invariants(syntax, "imports_check.syntax")
  File_Slot_Invariants(slot, "imports_check.slot")
  Source_Count_Invariants(count, "imports_check.count")
  Diagnostic_Invariants(checked, "imports_check.checked")
  diagnostic = checked
  if checked.Tier != TIER_NONE {
    return diagnostic
  }
  gateway := file_gateway(memory, slot)
  view := ast.Parse_State{Syntax_Tree: *syntax.Syntax_Tree}
  run := ast.Parse_State_Token_Run(&view)
  source := memory.Source[:count]
  within := false
  for index, lexeme := range run {
    switch lexeme.Kind {
    case token.KIND_IMPORT:
      within = true
    case token.KIND_FUNCTION, token.KIND_TYPE, token.KIND_VARIABLE, token.KIND_CONSTANT:
      return diagnostic
    }
    if !within {
      continue
    }
    if lexeme.Kind != token.KIND_STRING {
      continue
    }
    // An import path holds no escape, thus the bytes between its quotes are the path, which
    // the notice store holds up to its first zero byte.
    clear(memory.Notice)
    path := source[lexeme.Offset+1 : int(lexeme.Offset)+int(lexeme.Size)-1]
    copy(memory.Notice[:IMPORT_QUOTE_SIZE_MAXIMUM], path)
    family := import_family(memory)
    if family == FAMILY_NONE {
      continue
    }
    if bool(import_exempt(memory, gateway, Stated_Family(family))) {
      continue
    }
    // An aliased import stands at its name, as lint states where the specification opens.
    opening := lexeme.Offset
    if run[index-1].Kind == token.KIND_IDENTIFIER {
      opening = run[index-1].Offset
    }
    import_notice(memory, Stated_Family(family))
    diagnostic = Diagnostic{Tier: TIER_TWO, Rule: RULE_IMPORTS}
    if !bool(token.Index_Lines(syntax.Lines, token.Source(source))) {
      return diagnostic
    }
    position := token.Position_Of(syntax.Lines, opening)
    diagnostic.Line = Diagnostic_Line(position.Line)
    diagnostic.Column = Diagnostic_Column(position.Column)
    return diagnostic
  }
  return diagnostic
}

// States the family of the import path the notice store holds, as lint's families name them:
// rand in any of its homes, the exact paths a shared package stands in for, then the families
// lint bans whole with every path under them.
func import_family(memory Memory_Handle) (family Import_Family) {
  defer func() { Import_Family_Invariants(family, "import_family.family") }()
  Memory_Handle_Invariants(memory, "import_family.memory")
  held := string(memory.Notice[:bytes.Index_Byte(bytes.Slice(memory.Notice), 0)])
  switch held {
  case "crypto/ecdsa", "crypto/ed25519", "crypto/elliptic", "crypto/hkdf", "crypto/hmac",
    "crypto/md5", "crypto/pbkdf2", "crypto/rsa", "crypto/sha1", "crypto/sha256",
    "crypto/sha512", "crypto/subtle", "crypto/x509", "crypto/x509/pkix":
    return FAMILY_CRYPTO
  case "database/sql", "database/sql/driver":
    return FAMILY_DATABASE
  case "go/ast", "go/build", "go/constant", "go/format", "go/printer", "go/token", "go/types":
    return FAMILY_GO
  case "hash/adler32", "hash/crc32", "hash/crc64", "hash/fnv", "hash/maphash":
    return FAMILY_HASH
  case "net":
    return FAMILY_NETWORK
  case "path", "path/filepath":
    return FAMILY_PATH
  case "sort":
    return FAMILY_SORT
  case "text/scanner", "text/tabwriter", "text/template":
    return FAMILY_TEXT
  }
  for _, home := range [...]string{"crypto/rand", "math/rand", "rand"} {
    if bool(path_family(Held_Path(held), Family_Root(home))) {
      return FAMILY_RANDOM
    }
  }
  for candidate := FAMILY_OS; candidate < FAMILY_UUID+1; candidate++ {
    // The families past os and before archive are exact paths, which the switch named.
    if candidate > FAMILY_OS {
      if candidate < FAMILY_ARCHIVE {
        continue
      }
    }
    if bool(path_family(Held_Path(held), Family_Root(family_name(Stated_Family(candidate))))) {
      return candidate
    }
  }
  return FAMILY_NONE
}

// Held_Path is an import path as the notice store holds it, cut at the quote a message holds.
type Held_Path string

// Held_Path_Invariants bounds the path by the quote.
func Held_Path_Invariants(value Held_Path, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(
    len(value),
    bytes.SLICE_SIZE_MINIMUM,
    IMPORT_QUOTE_SIZE_MAXIMUM,
  ).
    Ensure()
}

// FAMILY_ROOT_SIZE_MAXIMUM is the longest path a family stands at, a home of rand.
const FAMILY_ROOT_SIZE_MAXIMUM = len("crypto/rand")

// Family_Root is the path a family lint bans whole stands at.
type Family_Root string

// Family_Root_Invariants bounds the root by the families.
func Family_Root_Invariants(value Family_Root, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(
    len(value),
    FAMILY_NAME_SIZE_MINIMUM,
    FAMILY_ROOT_SIZE_MAXIMUM,
  ).
    Ensure()
}

// Reports whether an import path is the family's root or stands under it.
func path_family(path Held_Path, family Family_Root) (held Boolean) {
  defer func() { Boolean_Invariants(held, "path_family.held") }()
  Held_Path_Invariants(path, "path_family.path")
  Family_Root_Invariants(family, "path_family.family")
  if len(path) < len(family) {
    return false
  }
  if string(path[:len(family)]) != string(family) {
    return false
  }
  if len(path) == len(family) {
    return true
  }
  return path[len(family)] == PATH_SEPARATOR
}

// GATEWAY_NONE is a file in no directory lint exempts from an import ban.
const GATEWAY_NONE Gateway = 0

// GATEWAY_AVER is a file of the aver subtree, which imports any stdlib package.
const GATEWAY_AVER Gateway = 1

// GATEWAY_OS is a file of the os backend.
const GATEWAY_OS Gateway = 2

// GATEWAY_LOOP is a file of the loop backend.
const GATEWAY_LOOP Gateway = 3

// GATEWAY_CLOCK is a file of the clock backend.
const GATEWAY_CLOCK Gateway = 4

// Gateway is the directory lint exempts a file from import bans by, or none.
type Gateway int

// Gateway_Invariants admits each gateway, and none.
func Gateway_Invariants(value Gateway, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(int(value), int(GATEWAY_NONE), int(GATEWAY_CLOCK)).
    Ensure()
}

// States the gateway the Go file the run read stands in: its directory is the gateway's root, or
// stands under it past a separator.
func file_gateway(memory Memory_Handle, slot File_Slot) (gateway Gateway) {
  defer func() { Gateway_Invariants(gateway, "file_gateway.gateway") }()
  Memory_Handle_Invariants(memory, "file_gateway.memory")
  File_Slot_Invariants(slot, "file_gateway.slot")
  path := memory.Paths[slot]
  cut := int(bytes.Last_Index_Byte(bytes.Slice(path), PATH_SEPARATOR))
  directory := string(path[:max(cut, 0)])
  roots := [...]string{"", AVER_DIRECTORY, OS_GATEWAY, LOOP_GATEWAY, CLOCK_GATEWAY}
  for index := GATEWAY_AVER; index < GATEWAY_CLOCK+1; index++ {
    root := roots[index]
    if directory == root {
      return index
    }
    if len(directory) < len(root)+len("/a") {
      continue
    }
    if directory[len(root)] != PATH_SEPARATOR {
      continue
    }
    if directory[:len(root)] == root {
      return index
    }
  }
  return GATEWAY_NONE
}

// Reports whether a file in the gateway may make the import the notice store holds: a file of
// the aver subtree makes any, a process gateway binds os and syscall, the loop backend alone
// registers signals, and the os backend alone takes the rest of the os family.
func import_exempt(memory Memory_Handle, gateway Gateway, family Stated_Family) (exempt Boolean) {
  defer func() { Boolean_Invariants(exempt, "import_exempt.exempt") }()
  Memory_Handle_Invariants(memory, "import_exempt.memory")
  Gateway_Invariants(gateway, "import_exempt.gateway")
  Stated_Family_Invariants(family, "import_exempt.family")
  held := string(memory.Notice[:bytes.Index_Byte(bytes.Slice(memory.Notice), 0)])
  switch {
  case gateway == GATEWAY_AVER:
    return true
  case held == "os/signal":
    return Boolean(gateway == GATEWAY_LOOP)
  case held == "os", held == "syscall":
    return Boolean(gateway > GATEWAY_AVER)
  case Import_Family(family) == FAMILY_OS:
    return Boolean(gateway == GATEWAY_OS)
  }
  return false
}

// AVER_DIRECTORY roots the assertion library, which imports the stdlib directly since every
// shared port depends on it.
const AVER_DIRECTORY = "shared/sim/aver"

// OS_GATEWAY binds the process for the shared os package.
const OS_GATEWAY = "shared/os/default"

// LOOP_GATEWAY binds the process for the loop backend, which alone registers signals.
const LOOP_GATEWAY = "shared/sim/nbio/default"

// CLOCK_GATEWAY binds the process for the clock backend.
const CLOCK_GATEWAY = "shared/sim/time/default"

// Rewrites the import path the notice store holds into the message of its ban, closed by a zero
// byte. Each byte below a space stands as ?, since the path is untrusted.
func import_notice(memory Memory_Handle, family Stated_Family) {
  Memory_Handle_Invariants(memory, "import_notice.memory")
  Stated_Family_Invariants(family, "import_notice.family")
  notice := memory.Notice
  held := int(bytes.Index_Byte(bytes.Slice(notice), 0))
  copy(notice[len(IMPORT_OPENING):], notice[:held])
  copy(notice, IMPORT_OPENING)
  quoted := notice[len(IMPORT_OPENING) : len(IMPORT_OPENING)+held]
  for index, symbol := range quoted {
    if symbol < NAME_BYTE_MINIMUM {
      quoted[index] = requirements.QUOTED_BYTE_ABSENT
    }
  }
  written := append(notice[:len(IMPORT_OPENING)+held], IMPORT_FAMILY...)
  written = append(written, family_name(family)...)
  written = append(written, IMPORT_TAIL...)
  clear(notice[len(written):])
}

// States the message of the banned import the notice store holds, up to the zero byte that
// closes it.
func import_message(memory Memory_Handle) (message Import_Message) {
  defer func() { Import_Message_Invariants(message, "import_message.message") }()
  Memory_Handle_Invariants(memory, "import_message.memory")
  notice := memory.Notice[:bytes.Index_Byte(bytes.Slice(memory.Notice), 0)]
  return Import_Message(unsafe.String(unsafe.SliceData(notice), len(notice)))
}

// Spells one family as lint does.
func family_name(family Stated_Family) (name Family_Name) {
  defer func() { Family_Name_Invariants(name, "family_name.name") }()
  Stated_Family_Invariants(family, "family_name.family")
  names := [...]Family_Name{
    "",
    "os",
    "rand",
    "crypto",
    "database",
    "go",
    "hash",
    "net",
    "path",
    "sort",
    "text",
    "archive",
    "bytes",
    "compress",
    "container",
    "encoding",
    "flag",
    "io",
    "math",
    "slices",
    "strconv",
    "strings",
    "syscall",
    "unicode",
    "uuid",
  }
  return names[family]
}

// WALK_CAUSE_REFUSED is why a walk fails when the host refused a directory it opened or read.
const WALK_CAUSE_REFUSED = "the host refused the directory"

// CAUSE_REFUSED is a directory the host refused to open or read.
const CAUSE_REFUSED Walk_Cause = 0

// CAUSE_WIDE is a path past the path bound, the last cause.
const CAUSE_WIDE Walk_Cause = 6

// Walk_Cause is why a walk failed, in the order fail spells them: the host's refusal, the
// argument, the Go file table, the directory queue, the orphan table, a name, and a path.
type Walk_Cause int

// Walk_Cause_Invariants admits each cause.
func Walk_Cause_Invariants(value Walk_Cause, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(int(value), int(CAUSE_REFUSED), int(CAUSE_WIDE)).
    Ensure()
}

// States why a walk failed, as the run recorded it where the walk stopped.
func walk_cause(run_flags Run_Flag_Storage) (cause Walk_Cause) {
  defer func() { Walk_Cause_Invariants(cause, "walk_cause.cause") }()
  Run_Flag_Storage_Invariants(run_flags, "walk_cause.run_flags")
  for offset, flag := range [...]int{
    RUN_WALK_ARGUMENT,
    RUN_WALK_FILES,
    RUN_WALK_DIRECTORIES,
    RUN_WALK_ORPHANS,
    RUN_WALK_NAME,
    RUN_WALK_WIDE,
  } {
    if run_flags.Values[flag] {
      return Walk_Cause(offset + 1)
    }
  }
  return CAUSE_REFUSED
}
