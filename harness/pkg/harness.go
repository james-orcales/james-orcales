// Package harness is the pure library tier of the harness linter: it walks one Go package and
// reports each file that breaks a rule. Every effect is submitted to the injected IO surface —
// package main binds the real loop, allocates every store, and pumps.
package harness

import "errors"
import "reflect"
import "unsafe"

import "local/james-orcales/harness/pkg/configuration"
import "local/james-orcales/harness/pkg/naming"
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
const RULE_CONFLICTED Rule = 10

// RULE_LAYOUT is a Go file that breaks the component layout.
const RULE_LAYOUT Rule = 9

// RULE_RECURSION is a Go file holding a call that closes a cycle among its package's functions.
const RULE_RECURSION Rule = 8

// RULE_SHADOWS is a Go file declaring a name a scope around it, or a sibling file, declares.
const RULE_SHADOWS Rule = 7

// RULE_IMPORTS is a Go file importing a stdlib package lint bans.
const RULE_IMPORTS Rule = 5

// RULE_NAMES is a Go file declaring a name that breaks a rule of harness/pkg/naming.
const RULE_NAMES Rule = 6

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

// MODULE_PATH is the module file a run reads from its working directory past harness.json.
const MODULE_PATH = "go.mod"

// WORKSPACE_NAME is the file that joins modules into a workspace.
const WORKSPACE_NAME = "go.work"

// RULE_TEXT_LAYOUT names a finding of the component layout in flat JSON.
const RULE_TEXT_LAYOUT Rule_Text = "layout"

// CONFIGURATION_PREFIX opens the line of a configuration that does not decode.
const CONFIGURATION_PREFIX = PROGRAM_PREFIX + CONFIGURATION_PATH + SEPARATOR

// QUOTE_OPENING stands between a fault and the text it quotes.
const QUOTE_OPENING = ": \""

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

// RULE_TEXT_NAMES names RULE_NAMES in flat JSON.
const RULE_TEXT_NAMES Rule_Text = "names"

// RULE_TEXT_SHADOWS names RULE_SHADOWS in flat JSON.
const RULE_TEXT_SHADOWS Rule_Text = "shadows"

// RULE_TEXT_RECURSION names RULE_RECURSION in flat JSON.
const RULE_TEXT_RECURSION Rule_Text = "recursion"

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

// STAGE_PASSED writes the closing line of a run: the pass line, or the elapsed seconds alone.
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
const STAGE_HISTORY_LOG Stage = 24

// STAGE_RECALL states the diagnostic the declaration pass left for the file the run stands on,
// without IO.
const STAGE_RECALL Stage = 23

// STAGE_PACKAGE starts the contract check of the next package, without IO. A stage of its own
// keeps the package walk a loop through Rearm rather than a recursion.
const STAGE_PACKAGE Stage = 12

// STAGE_REQUIREMENTS resumes the open package's contract check, without IO.
const STAGE_REQUIREMENTS Stage = 7

// ELAPSED_OPENING opens the line of the elapsed seconds, which close every run that ends.
const ELAPSED_OPENING = "elapsed_sec="

// PASSED opens the pass line; the elapsed seconds follow it.
const PASSED = "✓ all checks passed\n" + ELAPSED_OPENING

// MILLISECONDS_PER_SECOND splits elapsed milliseconds into seconds and their fraction.
const MILLISECONDS_PER_SECOND = 1000

// Main opens a run: it stamps the start and stands the run where Rearm parses the command line.
// It returns without waiting: the root repeats Rearm and pumps.
func Main(input Main_Input) {
  Main_Input_Invariants(&input, "main.input")
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
  Command_Invariants(&command, "parse.command")
  Run_Flag_Storage_Invariants(&run_flags, "parse.run_flags")
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
  Run_Flag_Storage_Invariants(&run_flags, "record.run_flags")
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
  // The component layout reads which components the walk covers from its root.
  rooted := memory.Layout.Text[LAYOUT_ROOT : LAYOUT_ROOT+LAYOUT_REGION_SIZE]
  rooted[copy(rooted[:LAYOUT_REGION_SIZE-1], root)] = 0
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
  defer func() { Diagnostic_Invariants(&diagnostic, "check.diagnostic") }()
  Memory_Handle_Invariants(memory, "check.memory")
  requirements.Syntax_Invariants(&syntax, "check.syntax")
  Source_Count_Invariants(count, "check.count")
  Format_Invariants(&format_state, "Check.format_state")
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

// RUN_RENAME records the rename past the declaration pass, which reads again the bytes of each file
// the whole name table can reach and renames them, parsing none.
const RUN_RENAME = 17

// RUN_HISTORY_READ records that git told the run its history while the lanes worked, thus past
// the packages the run judges it without asking git again.
const RUN_HISTORY_READ = 18

// RUN_SIBLINGS records that the run judged the names of each package's files against each other.
const RUN_SIBLINGS = 19

// RUN_MODULE records that the run read go.mod, or found none, past harness.json.
const RUN_MODULE = 20

// RUN_FLAG_COUNT is how many flags one run keeps.
const RUN_FLAG_COUNT = 21

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
func Run_Flag_Storage_Invariants(value *Run_Flag_Storage, namespace aver.Namespace) {
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
  // Declarations holds the leading declarations the one parse of requirements_test.go collected.
  Declarations requirements.Declarations
  // Declared is how many of them it collected.
  Declared requirements.Declaration_Count
  // Read is what became of requirements_test.go past that parse.
  Read requirements.Test_Read
}

// Package_Invariants composes one package. Each mark states its own kind, since one type
// standing for three of them would name a package whose parts the compiler lets a run swap.
func Package_Invariants(value *Package, namespace aver.Namespace) {
  Package_Path_Invariants(value.Path, namespace)
  Markdown_Mark_Invariants(value.Markdown, namespace)
  Test_Mark_Invariants(value.Test, namespace)
  Impure_Mark_Invariants(value.Impure, namespace)
  requirements.Declarations_Invariants(value.Declarations, namespace)
  requirements.Declaration_Count_Invariants(value.Declared, namespace)
  requirements.Test_Read_Invariants(value.Read, namespace)
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
func Recorded_Package_Invariants(value *Recorded_Package, namespace aver.Namespace) {
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
const RULE_TEXT_SIZE_MINIMUM = len(RULE_TEXT_NAMES)

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
func Report_Invariants(value *Report, namespace aver.Namespace) {
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
func Summary_Invariants(value *Summary, namespace aver.Namespace) {
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

// File_Pass is the stage one pass over the file table leaves the run on: the recall of one
// diagnostic, or the first package once every file is served.
type File_Pass Stage

// File_Pass_Invariants admits each stage one pass over the file table leaves the run on.
func File_Pass_Invariants(value File_Pass, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_Int(int(value), int(STAGE_PACKAGE), int(STAGE_RECALL)).
    Ensure()
}

// Declare_Pass is the stage one step of the declaration pass leaves the run on: the open of the
// next file, the fix one closed lane owes, the pass again, or whatever the first step of the file
// table leaves.
type Declare_Pass Stage

// Declare_Pass_Invariants admits each stage one step of the declaration pass leaves the run on.
func Declare_Pass_Invariants(value Declare_Pass, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(int(value), int(STAGE_FIX_OPEN), int(STAGE_RECALL)).
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
    Enum_Int(int(value), int(STAGE_REQUIREMENTS), int(STAGE_DECLARED)).
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
    Enum_4_Int(
    int(value),
    int(STAGE_FIX_OPEN),
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

// CLOSE_PASS_FAILED is the close a step of the run failed at, which the run cannot finish.
const CLOSE_PASS_FAILED Close_Pass = 0

// CLOSE_PASS_CONTRACT is the close of a file the declaration pass or a contract read, which the
// read's own step takes from here.
const CLOSE_PASS_CONTRACT Close_Pass = 1

// CLOSE_PASS_CONFIGURED is the close of a configuration that decodes, which the walk follows.
const CLOSE_PASS_CONFIGURED Close_Pass = 2

// Close_Pass is what one close leaves the run to do. The close names the three, and the caller
// alone turns one of them into the stage the run stands on next.
type Close_Pass int

// Close_Pass_Invariants admits the three one close leaves.
func Close_Pass_Invariants(value Close_Pass, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_3_Int(
    int(value),
    int(CLOSE_PASS_FAILED),
    int(CLOSE_PASS_CONTRACT),
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

// Fixed_Reading is what a fixed file serves: a Go file the declaration pass read, or a contract.
type Fixed_Reading Reading

// Fixed_Reading_Invariants admits the two files a fix writes.
func Fixed_Reading_Invariants(value Fixed_Reading, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_Int(int(value), int(READING_MARKDOWN), int(READING_DECLARATION)).
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
func Reported_Invariants(value *Reported, namespace aver.Namespace) {
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
func Diagnostic_Invariants(value *Diagnostic, namespace aver.Namespace) {
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
  Report_Invariants(value, namespace)
}

// Summary_Handle is the caller-owned summary one write states.
type Summary_Handle *Summary

// Summary_Handle_Invariants composes a present summary.
func Summary_Handle_Invariants(value Summary_Handle, namespace aver.Namespace) {
  if value == nil {
    return
  }
  Summary_Invariants(value, namespace)
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
func Reports_Invariants(value *Reports, namespace aver.Namespace) {
  Report_Invariants(&value.Report, namespace)
  Summary_Invariants(&value.Summary, namespace)
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
func Directory_Invariants(value *Directory, namespace aver.Namespace) {
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
  Directory_Invariants(value, namespace)
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
func Command_Invariants(value *Command, namespace aver.Namespace) {
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
func Formatter_Stores_Invariants(value *Formatter_Stores, namespace aver.Namespace) {
  format.Place_Storage_Invariants(&value.Places, namespace)
  format.Table_Fields_Invariants(&value.Table, namespace)
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
func Format_Invariants(value *Format, namespace aver.Namespace) {
  Formatter_Stores_Invariants(&value.Formatter, namespace)
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
  // Sizes holds each Go file's size as the walk found it, slot for slot with Paths.
  Sizes File_Sizes
  // Order holds the slots in the order the declaration pass reads them: the largest file first.
  Order File_Order
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
  // Notice holds the message of one finding of harness.json, of one banned import, or of one name.
  Notice Notice_Store
  // Name_Checker checks the names of the Go file a lane holds, over the word table of the
  // configuration; the run's own memory holds none.
  Name_Checker naming.Checker_Handle
  // Diagnostics holds the diagnostic the declaration pass left for each Go file.
  Diagnostics File_Diagnostics
  // Notices holds the banned import message the declaration pass left for each Go file.
  Notices Notice_Table
  // Orphans holds each symbolic link the walk found whose target resolves to nothing.
  Orphans Orphan_Paths
  // Link holds the path of one child the walk asks the host about.
  Link Link_Path
  // Resolved holds the path one link resolution stands on.
  Resolved Resolved_Path
  // Target holds the target one symbolic link names.
  Target Link_Target
  // Lanes holds the stores each Go file of one batch is checked in apart.
  Lanes Lane_Table
  // Held marks each lane the declaration pass handed over and has not closed.
  Held Lane_Marks
  // Siblings holds the names each Go file declares, which the files of one package judge.
  Siblings Sibling_Table
  // Layout holds the findings of the component layout that name no Go file, and the module path.
  Layout Layout_Table
  // Graph holds the call graph of the package the recursion judgment reads.
  Graph Graph_Store
  // Glob holds the glob pattern and the path one match of opt_out_recursion_ban reads.
  Glob Glob_Storage
}

// Memory_Invariants states that every store the root allocated stands at its whole size. A store
// another package declares belongs to that package's chain, which its own simulation covers, thus
// this chain states the one property the run depends on: the store is there and cannot grow.
func Memory_Invariants(value *Memory, namespace aver.Namespace) {
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
  File_Sizes_Invariants(value.Sizes, namespace)
  File_Order_Invariants(value.Order, namespace)
  requirements.Checker_Handle_Invariants(value.Requirements, namespace)
  History_Output_Invariants(value.History_Output, namespace)
  History_Errors_Invariants(value.History_Errors, namespace)
  History_Name_Invariants(value.History_Name, namespace)
  configuration.Configuration_Handle_Invariants(value.Configuration, namespace)
  Notice_Store_Invariants(value.Notice, namespace)
  naming.Checker_Handle_Invariants(value.Name_Checker, namespace)
  File_Diagnostics_Invariants(value.Diagnostics, namespace)
  Notice_Table_Invariants(value.Notices, namespace)
  Orphan_Paths_Invariants(value.Orphans, namespace)
  Link_Path_Invariants(value.Link, namespace)
  Resolved_Path_Invariants(value.Resolved, namespace)
  Link_Target_Invariants(value.Target, namespace)
  Lane_Table_Invariants(value.Lanes, namespace)
  Lane_Marks_Invariants(value.Held, namespace)
  Sibling_Table_Invariants(&value.Siblings, namespace)
  Layout_Table_Invariants(&value.Layout, namespace)
  Graph_Store_Invariants(value.Graph, namespace)
  Glob_Storage_Invariants(value.Glob, namespace)
}

// Memory_Handle is caller-owned run storage.
type Memory_Handle *Memory

// Memory_Handle_Invariants composes present run storage.
func Memory_Handle_Invariants(value Memory_Handle, namespace aver.Namespace) {
  if value == nil {
    return
  }
  Memory_Invariants(value, namespace)
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

// Lane_Procedure is one step of the root's spread on one lane.
type Lane_Procedure func(lane Lane_Handle)

// Reap_Procedure hands back one lane the run holds whose check is done, waiting for one where none
// is.
type Reap_Procedure func(memory Memory_Handle) (lane Lane_Index)

// Spread is how the root checks lanes: Submit hands one lane over, and Reap hands back a lane whose
// check is done. No lane reads what another writes, and what a closed lane leaves the run states
// the same in any order, thus a spread checks lanes at once or in turn, and reaps them in any
// order.
type Spread struct {
  // Submit hands one loaded lane over for its check.
  Submit Lane_Procedure
  // Reap hands back one checked lane.
  Reap Reap_Procedure
}

// Spread_Invariants states both steps bound.
func Spread_Invariants(value *Spread, _ aver.Namespace) {
  aver.Always(value.Submit != nil, "A spread submits.")
  aver.Always(value.Reap != nil, "A spread reaps.")
}

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
func Loop_Invariants(value *Loop, namespace aver.Namespace) {
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
func Progress_Invariants(value *Progress, namespace aver.Namespace) {
  Phase_Invariants(value.Phase, namespace)
}

// Outcome is what the run will exit as, and the step it stands on while it gets there. It stands
// apart from the phase, because a step that names a failure states nothing of the loop's turn.
type Outcome struct {
  // Status is the exit code, final once Phase is PHASE_STOPPED.
  Status Status_Code
}

// Outcome_Invariants composes what the run will exit as.
func Outcome_Invariants(value *Outcome, namespace aver.Namespace) {
  Status_Code_Invariants(value.Status, namespace)
}

// Outcome_Handle is a caller-owned outcome.
type Outcome_Handle *Outcome

// Outcome_Handle_Invariants composes a present outcome.
func Outcome_Handle_Invariants(value Outcome_Handle, namespace aver.Namespace) {
  if value == nil {
    return
  }
  Outcome_Invariants(value, namespace)
}

// Progress_Handle is caller-owned run progress.
type Progress_Handle *Progress

// Progress_Handle_Invariants composes present run progress.
func Progress_Handle_Invariants(value Progress_Handle, namespace aver.Namespace) {
  if value == nil {
    return
  }
  Progress_Invariants(value, namespace)
}

// Loop_Handle is caller-owned loop position.
type Loop_Handle *Loop

// Loop_Handle_Invariants composes a present loop position.
func Loop_Handle_Invariants(value Loop_Handle, namespace aver.Namespace) {
  if value == nil {
    return
  }
  Loop_Invariants(value, namespace)
}

// Walk is the pass over the tree: what it collected.
type Walk struct {
  // Package_Count counts the packages in Memory.Packages.
  Package_Count Package_Count
  // Orphan_Count counts the links in Memory.Orphans.
  Orphan_Count Orphan_Count
}

// Walk_Invariants composes the tree pass.
func Walk_Invariants(value *Walk, namespace aver.Namespace) {
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
func Orphan_Cursor_Invariants(value *Orphan_Cursor, namespace aver.Namespace) {
  Orphan_Index_Invariants(value.Orphan_Index, namespace)
}

// Orphan_Cursor_Handle is a caller-owned orphan cursor.
type Orphan_Cursor_Handle *Orphan_Cursor

// Orphan_Cursor_Handle_Invariants composes the orphan cursor behind its handle.
func Orphan_Cursor_Handle_Invariants(value Orphan_Cursor_Handle, namespace aver.Namespace) {
  if value == nil {
    return
  }
  Orphan_Cursor_Invariants(value, namespace)
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
  Walk_Invariants(value, namespace)
}

// Package_Cursor is the package whose contract the run checks. It stands apart from the pass,
// because the walk that collects packages checks none of them, thus a step of that walk states
// no cursor it could move.
type Package_Cursor struct {
  // Package_Index is the package whose contract the run checks.
  Package_Index Package_Index
}

// Package_Cursor_Invariants composes the package cursor.
func Package_Cursor_Invariants(value *Package_Cursor, namespace aver.Namespace) {
  Package_Index_Invariants(value.Package_Index, namespace)
}

// Package_Cursor_Handle is a caller-owned package cursor.
type Package_Cursor_Handle *Package_Cursor

// Package_Cursor_Handle_Invariants composes the package cursor behind its handle.
func Package_Cursor_Handle_Invariants(value Package_Cursor_Handle, namespace aver.Namespace) {
  if value == nil {
    return
  }
  Package_Cursor_Invariants(value, namespace)
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
func Read_Invariants(value *Read, namespace aver.Namespace) {
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
  Read_Invariants(value, namespace)
}

// Files is where the run stands in the collected paths. It stands apart from the pass, because a
// step that reports one file states no count of packages or directories.
type Files struct {
  // File_Count counts the paths in Memory.Paths.
  File_Count File_Count
}

// Files_Invariants composes the collected paths.
func Files_Invariants(value *Files, namespace aver.Namespace) {
  File_Count_Invariants(value.File_Count, namespace)
}

// Files_Handle is a caller-owned count of collected paths.
type Files_Handle *Files

// Files_Handle_Invariants composes the collected paths behind their handle.
func Files_Handle_Invariants(value Files_Handle, namespace aver.Namespace) {
  if value == nil {
    return
  }
  Files_Invariants(value, namespace)
}

// File_Cursor is the file the run stands on. It stands apart from the count, because the walk
// that collects paths stands on none of them, thus a step of that walk states no cursor it
// could move.
type File_Cursor struct {
  // File_Index is the file the run stands on.
  File_Index File_Index
}

// File_Cursor_Invariants composes the file cursor.
func File_Cursor_Invariants(value *File_Cursor, namespace aver.Namespace) {
  File_Index_Invariants(value.File_Index, namespace)
}

// File_Cursor_Handle is a caller-owned file cursor.
type File_Cursor_Handle *File_Cursor

// File_Cursor_Handle_Invariants composes the file cursor behind its handle.
func File_Cursor_Handle_Invariants(value File_Cursor_Handle, namespace aver.Namespace) {
  if value == nil {
    return
  }
  File_Cursor_Invariants(value, namespace)
}

// Finding is what the fix step writes: the form the last check found a file should stand in. A
// diagnostic the run states rather than fixes stands in the file table, thus the finding holds
// no rule and no position.
type Finding struct {
  // Fix is the whole file as it should stand.
  Fix Fix
}

// Finding_Invariants composes what the fix step writes.
func Finding_Invariants(value *Finding, namespace aver.Namespace) {
  Fix_Invariants(value.Fix, namespace)
}

// Finding_Handle is caller-owned finding state.
type Finding_Handle *Finding

// Finding_Handle_Invariants composes a present finding.
func Finding_Handle_Invariants(value Finding_Handle, namespace aver.Namespace) {
  if value == nil {
    return
  }
  Finding_Invariants(value, namespace)
}

// Fix_State is how far one fix write got. Only the passes that write a fix take it, so a helper
// that never writes one owes no count of it.
type Fix_State struct {
  // Fix_Count counts the fix bytes written to the current file.
  Fix_Count Fix_Count
}

// Fix_State_Invariants composes how far one fix write got.
func Fix_State_Invariants(value *Fix_State, namespace aver.Namespace) {
  Fix_Count_Invariants(value.Fix_Count, namespace)
}

// Fix_State_Handle is caller-owned fix state.
type Fix_State_Handle *Fix_State

// Fix_State_Handle_Invariants composes present fix state.
func Fix_State_Handle_Invariants(value Fix_State_Handle, namespace aver.Namespace) {
  if value == nil {
    return
  }
  Fix_State_Invariants(value, namespace)
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
func Tally_Invariants(value *Tally, namespace aver.Namespace) {
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
  Tally_Invariants(value, namespace)
}

// Findings is the line the run writes about what it found.
type Findings struct {
  // Line views the part of Memory.Line to write.
  Line Line
}

// Findings_Invariants composes what the run found.
func Findings_Invariants(value *Findings, namespace aver.Namespace) {
  Line_Invariants(value.Line, namespace)
}

// Findings_Handle is caller-owned finding state.
type Findings_Handle *Findings

// Findings_Handle_Invariants composes present finding state.
func Findings_Handle_Invariants(value Findings_Handle, namespace aver.Namespace) {
  if value == nil {
    return
  }
  Findings_Invariants(value, namespace)
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
  // Spread checks the lanes.
  Spread Spread
}

// Streams_Invariants composes what one submission stands on.
func Streams_Invariants(value *Streams, namespace aver.Namespace) {
  Spread_Invariants(&value.Spread, namespace)
  Loop_Handle_Invariants(value.Loop, namespace)
  Stage_Handle_Invariants(value.Stage, namespace)
  Walk_Handle_Invariants(value.Walk, namespace)
  Package_Cursor_Handle_Invariants(value.Package_Cursor, namespace)
  Orphan_Cursor_Handle_Invariants(value.Orphans, namespace)
  Files_Handle_Invariants(value.Files, namespace)
  File_Cursor_Handle_Invariants(value.File_Cursor, namespace)
  Read_Handle_Invariants(value.Read, namespace)
  Format_Invariants(&value.Format, namespace)
  Directory_Handle_Invariants(value.Directory, namespace)
  Run_Flag_Storage_Invariants(&value.Flags, namespace)
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
  // Spread is bound by the root.
  Spread Spread
  // Stopped is bound by the root.
  Stopped Stopped_Procedure
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
func Runner_Fields_Invariants(value *Runner_Fields, namespace aver.Namespace) {
  Spread_Invariants(&value.Spread, namespace)
  Loop_Invariants(&value.Loop, namespace)
  Stage_Invariants(value.Stage, namespace)
  Progress_Invariants(&value.Progress, namespace)
  Outcome_Invariants(&value.Outcome, namespace)
  Reports_Invariants(&value.Reports, namespace)
  time.Clock_Invariants(&value.Clock, namespace)
  time.Monotonic_Moment_Invariants(value.Start, namespace)
  os.Arguments_Invariants(value.Arguments, namespace)
  Memory_Handle_Invariants(value.Memory, namespace)
  Walk_Invariants(&value.Walk, namespace)
  Package_Cursor_Invariants(&value.Package_Cursor, namespace)
  Orphan_Cursor_Invariants(&value.Orphan_Cursor, namespace)
  Files_Invariants(&value.Files, namespace)
  File_Cursor_Invariants(&value.File_Cursor, namespace)
  Read_Invariants(&value.Read, namespace)
  Format_Invariants(&value.Format, namespace)
  Command_Invariants(&value.Command, namespace)
  Directory_Invariants(&value.Directory, namespace)
  Run_Flag_Storage_Invariants(&value.Flags, namespace)
  Findings_Invariants(&value.Findings, namespace)
  Finding_Invariants(&value.Finding, namespace)
  Fix_State_Invariants(&value.Fix_State, namespace)
  Tally_Invariants(&value.Tally, namespace)
  History_Invariants(&value.History, namespace)
}

// Runner is one whole run.
type Runner struct {
  Runner_Fields
}

// Runner_Invariants composes every run field.
func Runner_Invariants(value *Runner, namespace aver.Namespace) {
  Runner_Fields_Invariants(&value.Runner_Fields, namespace)
}

// Runner_Handle is caller-owned run state.
type Runner_Handle *Runner

// Runner_Handle_Invariants composes present run state.
func Runner_Handle_Invariants(value Runner_Handle, namespace aver.Namespace) {
  if value == nil {
    return
  }
  Runner_Invariants(value, namespace)
}

// Rearm_Input is what one rearm stands on: the run's stores, the parse it holds, the parts one
// pass works on, where the pump stands, and how long the run has taken.
type Rearm_Input struct {
  // Memory is the run's stores.
  Memory Memory_Handle
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
func Rearm_Input_Invariants(value *Rearm_Input, namespace aver.Namespace) {
  Memory_Handle_Invariants(value.Memory, namespace)
  Streams_Invariants(&value.State, namespace)
  Progress_Handle_Invariants(value.Progress, namespace)
  Elapsed_Invariants(value.Elapsed, namespace)
  os.Arguments_Invariants(value.Arguments, namespace)
  Command_Invariants(&value.Command, namespace)
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
func Main_Input_Invariants(value *Main_Input, namespace aver.Namespace) {
  Opening_Phase_Handle_Invariants(value.Phase, namespace)
  Moment_Handle_Invariants(value.Start, namespace)
  time.Clock_Invariants(&value.Clock, namespace)
}

// Advance_Pass is what one ready stage leaves the run on: another stage ready, or an operation
// to submit, since every run closes on a line it writes.
type Advance_Pass int

// Advance_Pass_Invariants admits each pass.
func Advance_Pass_Invariants(value Advance_Pass, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_Int(int(value), int(ADVANCE_READY), int(ADVANCE_SUBMIT)).
    Ensure()
}

// Advance_Input is what one ready stage stands on: the run's stores, the parse it holds, the
// parts one pass works on, and how long the run has taken.
type Advance_Input struct {
  // Memory is the run's stores.
  Memory Memory_Handle
  // State is the parts one pass works on.
  State Streams
  // Elapsed is how long the run has taken.
  Elapsed Elapsed
}

// Advance_Input_Invariants composes what one ready stage stands on.
func Advance_Input_Invariants(value *Advance_Input, namespace aver.Namespace) {
  Memory_Handle_Invariants(value.Memory, namespace)
  Streams_Invariants(&value.State, namespace)
  Elapsed_Invariants(value.Elapsed, namespace)
}

// Steps the stage a run is ready on: a stage that queues no operation of its own retires
// nothing, thus it leaves the run ready for the stage it names.
func advance(input Advance_Input) (next Advance_Pass) {
  defer func() { Advance_Pass_Invariants(next, "advance.next") }()
  Advance_Input_Invariants(&input, "advance.input")
  memory, state, elapsed := input.Memory, input.State, input.Elapsed
  checked := Checked_Reading_Handle((*Checked_Reading)(&state.Read.Reading))
  path := Read_Path_Handle(&state.Read.Path)
  packages, files := state.Walk.Package_Count, state.Files.File_Count
  cursor, served := state.File_Cursor, Served_Reading_Handle((*Served_Reading)(&state.Read.Reading))
  lanes := Lane_Input{Spread: state.Spread, Finding: state.Finding, Flags: state.Flags}
  lanes.Formatter = state.Format.Formatter
  switch *state.Stage {
  case STAGE_PACKAGE:
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
  case STAGE_SERVE:
    cursor.File_Index++
    next := file_next(memory, packages, files, cursor, served, state.Flags)
    *state.Stage = Stage(next)
  case STAGE_RECALL:
    findings, report, slot := state.Findings, state.Report, File_Slot(state.File_Cursor.File_Index)
    recalled := file_recall(memory, findings, report, state.Tally, state.Flags, slot, state.Outcome)
    *state.Stage = Stage(recalled)
  case STAGE_DECLARED:
    slot := memory.Order[cursor.File_Index]
    *state.Stage = Stage(declared(memory, lanes, slot, state.Read.Source_Count))
    cursor.File_Index++
  case STAGE_DECLARE:
    next := declare_next(memory, packages, files, cursor, served, path, lanes)
    *state.Stage = Stage(next)
  case STAGE_REQUIREMENTS:
    *state.Stage = Stage(requirements_next(memory, checked, path, state.Finding))
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
    past := Past{Orphans: state.Orphans, Walk: state.Walk, Loop: state.Loop, Outcome: state.Outcome}
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
  Rearm_Input_Invariants(&input, "rearm.input")
  memory, state := input.Memory, input.State
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
    memory.Layout.Findings[LAYOUT_HEADER] = Layout_Finding{
      Path_Start: LAYOUT_FINDING_FIRST,
      Path_End:   LAYOUT_FINDING_FIRST,
    }
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
    advanced := Advance_Input{Memory: memory, State: state, Elapsed: input.Elapsed}
    switch advance(advanced) {
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
        state.Finding.Fix,
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
    if bool(judge(memory, state)) {
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
  Run_Flag_Storage_Invariants(&run_flags, "judge_walk.run_flags")
  Files_Handle_Invariants(files, "judge_walk.files")
  Formatter_Stores_Invariants(&stores, "judge_walk.stores")
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
  state Streams,
) (stopped Boolean) {
  defer func() { Boolean_Invariants(stopped, "judge.stopped") }()
  Memory_Handle_Invariants(memory, "judge.memory")
  Streams_Invariants(&state, "judge.state")
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
    // A run that finds no module file states it and walks the tree anyway.
    if loop.Completion.Error != nil {
      if run_flags.Values[RUN_CONFIGURED] {
        if !run_flags.Values[RUN_MODULE] {
          run_flags.Values[RUN_MODULE] = true
          module_absent(memory)
          *stage = opened
          return false
        }
      }
    }
    *stage = Stage(judge_open(memory, loop, run_flags, read))
    owed, step, failed = true, STEP_OPEN, Failure_Path(read.Path)
  case STAGE_READ:
    unfilled := Unfilled_Count_Handle((*Unfilled_Count)(&read.Source_Count))
    *stage = Stage(judge_read(memory, loop, run_flags, read.Reading, unfilled))
  case STAGE_CLOSE:
    closed := judge_close(memory, loop, findings, run_flags, read, outcome)
    // A contract close takes the stage its check names, thus its slot of the table stands unread.
    stages := [...]Stage{STAGE_FAIL, opened, opened}
    *stage = stages[closed]
    // The configuration opens the module file, and the module file opens the walk.
    if closed == CLOSE_PASS_CONFIGURED {
      if !run_flags.Values[RUN_MODULE] {
        read.Path = append(read.Path[:0], MODULE_PATH...)
        *stage = STAGE_OPEN
      }
    }
    if closed == CLOSE_PASS_CONTRACT {
      contract := Contract_Reading_Handle((*Contract_Reading)(&read.Reading))
      count := read.Source_Count
      next := judge_served(memory, run_flags, contract, count)
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
    *stage = Stage(
      history_resume(memory, judge_history(memory, loop, History_Stage(*stage), source)),
    )
  case STAGE_FIX_OPEN, STAGE_FIX_WRITE, STAGE_FIX_CLOSE:
    fix, reading := Written_Fix(state.Finding.Fix), Fixed_Reading(read.Reading)
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
  Run_Flag_Storage_Invariants(&run_flags, "judge_open.run_flags")
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
  Run_Flag_Storage_Invariants(&run_flags, "judge_fix.run_flags")
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
  return Fix_Pass(STAGE_DECLARE)
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
  Run_Flag_Storage_Invariants(&run_flags, "judge_directory_read.run_flags")
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
  Run_Flag_Storage_Invariants(&run_flags, "judge_directory_open.run_flags")
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
  Run_Flag_Storage_Invariants(&run_flags, "collect_directory.run_flags")
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
  Run_Flag_Storage_Invariants(&run_flags, "collect.run_flags")
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
  // A path the ignore list holds stands outside the run: the walk opens no such directory and
  // checks no such file.
  // A walked path spends at most PATH_SIZE_MAXIMUM bytes, thus its zero byte fits the part.
  glob := memory.Glob[GLOB_PATTERN_SIZE:GLOB_STORAGE_SIZE]
  size := copy(glob, current)
  size += copy(glob[size:], "/")
  glob[size+copy(glob[size:], entry.Name)] = 0
  // A module file past the root, or a workspace file anywhere, divides the one module.
  if !entry.Is_Directory {
    switch entry.Name {
    case MODULE_PATH, WORKSPACE_NAME:
      layout_module(memory, Boolean(entry.Name == WORKSPACE_NAME))
    }
  }
  if bool(glob_listed(memory, GLOB_LIST_IGNORE)) {
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
  if !bool(collect_status(memory, orphans, storage, run_flags, Entry_Path(link), files.File_Count)) {
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
  taken File_Count,
) (regular Boolean) {
  defer func() { Boolean_Invariants(regular, "collect_status.regular") }()
  File_Count_Invariants(taken, "collect_status.taken")
  Memory_Handle_Invariants(memory, "collect_status.memory")
  Orphan_Count_Handle_Invariants(orphans, "collect_status.orphans")
  Run_Flag_Storage_Invariants(&run_flags, "collect_status.run_flags")
  Entry_Path_Invariants(entry, "collect_status.entry")
  stated := unsafe.String(unsafe.SliceData(entry), len(entry))
  status, status_err := nbio.Storage_Status(storage, stated)
  if status_err != nil {
    run_flags.Values[RUN_WALK_FAILED] = true
    return false
  }
  // The slot past the last file the walk took stands for the file this entry may be, thus its
  // size stands there whether or not the walk takes it.
  memory.Sizes[taken] = File_Size(min(status.Size, int64(SOURCE_STORAGE_SIZE)))
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
  Run_Flag_Storage_Invariants(&run_flags, "collect_source.run_flags")
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
  Run_Flag_Storage_Invariants(&run_flags, "judge_directory_close.run_flags")
  Files_Handle_Invariants(files, "judge_directory_close.files")
  Formatter_Stores_Invariants(&formatter, "judge_directory_close.formatter")
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
    // A package whose test file no lane reads holds no declaration the check can read.
    slot.Impure, slot.Declared, slot.Read = false, 0, requirements.TEST_REFUSED
    recorded := Recorded_Package{
      Path: slot.Path, Markdown: slot.Markdown, Test: slot.Test,
    }
    // A table states its length alone, thus each package states its own parts
    // where it is recorded rather than nowhere at all.
    Recorded_Package_Invariants(&recorded, "judge_directory_close.recorded")
    walk.Package_Count++
  }
  if *queue > 0 {
    return Walk_Pass(STAGE_DIRECTORY_OPEN)
  }
  paths_sort(memory.Paths, memory.File_Packages, memory.Sizes, files.File_Count)
  order_sort(memory, files.File_Count)
  format.Forget(&format.Formatter{Places: formatter.Places, Table_Fields: formatter.Table})
  // A run opens its cursor at the first file and the walk stands on none, thus the cursor
  // still stands there and needs no reset that would carry it into this pass.
  run_flags.Values[RUN_UNDECLARED], run_flags.Values[RUN_RENAME] = false, false
  run_flags.Values[RUN_SIBLINGS] = false
  siblings_open(memory, files.File_Count)
  return Walk_Pass(STAGE_DECLARE)
}

// Reads the next Go file for the names it declares. Past the last file the check pass starts,
// because a name one file declares is spelled in files the run checks before it.
func declare_next(
  memory Memory_Handle,
  packages Package_Count,
  served File_Count,
  file_cursor File_Cursor_Handle,
  reading Served_Reading_Handle,
  path_store Read_Path_Handle,
  input Lane_Input,
) (next Declare_Pass) {
  defer func() { Declare_Pass_Invariants(next, "declare_next.next") }()
  Read_Path_Handle_Invariants(path_store, "declare_next.path_store")
  Memory_Handle_Invariants(memory, "declare_next.memory")
  Package_Count_Invariants(packages, "declare_next.packages")
  File_Count_Invariants(served, "declare_next.served")
  File_Cursor_Handle_Invariants(file_cursor, "declare_next.file_cursor")
  Served_Reading_Handle_Invariants(reading, "declare_next.reading")
  Lane_Input_Invariants(&input, "declare_next.input")
  run_flags, formatter := input.Flags, input.Formatter
  // The rename reads again each file whose bytes are the form its one parse printed: a clean one,
  // one whose only fault is a banned import, and one -fix rewrote to its form.
  for file_cursor.File_Index < File_Index(served) {
    if !run_flags.Values[RUN_RENAME] {
      break
    }
    rule := memory.Diagnostics[memory.Order[file_cursor.File_Index]].Rule
    owed := rule == RULE_NONE
    if rule == RULE_IMPORTS {
      owed = true
    }
    if rule == RULE_NAMES {
      owed = true
    }
    if rule == RULE_SHADOWS {
      owed = true
    }
    if rule == RULE_RECURSION {
      owed = true
    }
    if rule == RULE_LAYOUT {
      owed = true
    }
    if rule == RULE_UNFORMATTED {
      owed = bool(run_flags.Values[RUN_AUTOFIX])
    }
    if owed {
      break
    }
    file_cursor.File_Index++
  }
  if file_cursor.File_Index < File_Index(served) {
    // Each lane holds a file, thus the spread hands one back before the pass reads another.
    if int(lanes_held(memory)) == LANE_COUNT {
      reaped := input.Spread.Reap(memory)
      if bool(lane_turn(memory, input, reaped, path_store)) {
        return Declare_Pass(STAGE_FIX_OPEN)
      }
    }
    *path_store = append(
      (*path_store)[:0], memory.Paths[memory.Order[file_cursor.File_Index]]...,
    )
    *reading = Served_Reading(READING_DECLARATION)
    return Declare_Pass(STAGE_OPEN)
  }
  // Git answers for the history while the lanes work, thus the run asks it once the last file
  // stands read rather than once every package stands checked.
  if lanes_held(memory) > 0 {
    if !run_flags.Values[RUN_HISTORY_READ] {
      run_flags.Values[RUN_HISTORY_READ] = true
      return Declare_Pass(STAGE_HISTORY_HEAD)
    }
  }
  // Past the last file, each lane the pass still holds closes, in the order the spread reaps them.
  for range LANE_COUNT {
    if lanes_held(memory) == 0 {
      break
    }
    reaped := input.Spread.Reap(memory)
    if bool(lane_turn(memory, input, reaped, path_store)) {
      return Declare_Pass(STAGE_FIX_OPEN)
    }
  }
  // A name one file declares is spelled in files the run checks before it, thus every file
  // stands declared here rather than one of them reaching the check undeclared.
  aver.Always(
    int(file_cursor.File_Index) == int(served),
    "A run declares every file before it checks any.",
  )
  file_cursor.File_Index = 0
  if !run_flags.Values[RUN_SIBLINGS] {
    run_flags.Values[RUN_SIBLINGS] = true
    siblings_judge(memory)
    recursion_judge(memory)
    layout_judge(memory)
  }
  if !run_flags.Values[RUN_RENAME] {
    // A rename that misses one file breaks every file that spells the name, thus a file the
    // table never took bars every rename of the run.
    if run_flags.Values[RUN_UNDECLARED] {
      view := format.Formatter{Places: formatter.Places, Table_Fields: formatter.Table}
      format.Forget(&view)
    }
    // Lanes format by no table, thus a table that took a name renames each file it can reach
    // past the pass, and a table that took none leaves each diagnostic final.
    if formatter.Places.Values[format.PLACE_NAMES] != 0 {
      run_flags.Values[RUN_RENAME] = true
      return Declare_Pass(STAGE_DECLARE)
    }
  }
  return Declare_Pass(file_next(memory, packages, served, file_cursor, reading, run_flags))
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
func paths_sort(table Path_Table, owners Package_Indices, sizes File_Sizes, count File_Count) {
  Path_Table_Invariants(table, "paths_sort.table")
  Package_Indices_Invariants(owners, "paths_sort.owners")
  File_Sizes_Invariants(sizes, "paths_sort.sizes")
  File_Count_Invariants(count, "paths_sort.count")
  paths := table[:count]
  owners, sizes = owners[:count], sizes[:count]
  for index := 1; index < len(paths); index++ {
    held, owner, size := paths[index], owners[index], sizes[index]
    position := index
    for position > 0 {
      if string(held) < string(paths[position-1]) {
        paths[position], owners[position] = paths[position-1], owners[position-1]
        sizes[position] = sizes[position-1]
        position--
        continue
      }
      break
    }
    paths[position], owners[position], sizes[position] = held, owner, size
  }
}

// Orders the slots the declaration pass reads, the largest file first: a large file read last
// holds the whole run past every other file's check, and one read first overlaps them.
func order_sort(memory Memory_Handle, count File_Count) {
  Memory_Handle_Invariants(memory, "order_sort.memory")
  File_Count_Invariants(count, "order_sort.count")
  order := memory.Order[:count]
  for index := range order {
    slot := File_Slot(index)
    position := index
    for position > 0 {
      if memory.Sizes[slot] > memory.Sizes[order[position-1]] {
        order[position] = order[position-1]
        position--
        continue
      }
      break
    }
    order[position] = slot
  }
}

// FILE_SIZE_MINIMUM is an empty file.
const FILE_SIZE_MINIMUM = 0

// File_Size is one Go file's size as the walk found it, up to one byte past the source bound.
type File_Size int64

// File_Size_Invariants bounds the size by the source store.
func File_Size_Invariants(value File_Size, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int64(int64(value), FILE_SIZE_MINIMUM, SOURCE_STORAGE_SIZE).
    Ensure()
}

// SIZE_TABLE_SIZE holds one size for each slot of the file table, and one for the entry the walk
// asks about past a full table.
const SIZE_TABLE_SIZE = FILE_COUNT_MAXIMUM + 1

// File_Sizes holds each Go file's size, slot for slot with the file table.
type File_Sizes []File_Size

// File_Sizes_Invariants fixes the table.
func File_Sizes_Invariants(value File_Sizes, _ aver.Namespace) {
  aver.Always(len(value) == SIZE_TABLE_SIZE, "A size table has complete length.")
  aver.Always(cap(value) == SIZE_TABLE_SIZE, "A size table cannot grow.")
}

// File_Order holds the slots of the file table in the order the declaration pass reads them.
type File_Order []File_Slot

// File_Order_Invariants fixes the table.
func File_Order_Invariants(value File_Order, _ aver.Namespace) {
  aver.Always(len(value) == FILE_COUNT_MAXIMUM, "A read order has complete length.")
  aver.Always(cap(value) == FILE_COUNT_MAXIMUM, "A read order cannot grow.")
}

// Judges a read that served the contract check or the name table rather than one file's rules.
func judge_served(
  memory Memory_Handle,
  run_flags Run_Flag_Storage,
  reading Contract_Reading_Handle,
  source_count Source_Count,
) (next Served_Pass) {
  defer func() { Served_Pass_Invariants(next, "judge_served.next") }()
  Memory_Handle_Invariants(memory, "judge_served.memory")
  Run_Flag_Storage_Invariants(&run_flags, "judge_served.run_flags")
  Contract_Reading_Handle_Invariants(reading, "judge_served.reading")
  Source_Count_Invariants(source_count, "judge_served.source_count")
  checker := memory.Requirements
  switch Reading(*reading) {
  case READING_MARKDOWN:
    // Under a fix the check's one walk of the contract writes the form it should stand in as
    // well, and asks for it to be written.
    wanted := requirements.Flag(run_flags.Values[RUN_AUTOFIX])
    checker.Flags.Values[requirements.FLAG_REWRITE_WANTED] = wanted
    requirements.Begin(checker)
  case READING_DECLARATION:
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
  Run_Flag_Storage_Invariants(&run_flags, "judge_read.run_flags")
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
  loop Loop_Handle,
  findings Findings_Handle,
  run_flags Run_Flag_Storage,
  read Read_Handle,
  outcome Outcome_Handle,
) (next Close_Pass) {
  defer func() { Close_Pass_Invariants(next, "judge_close.next") }()
  Memory_Handle_Invariants(memory, "judge_close.memory")
  Findings_Handle_Invariants(findings, "judge_close.findings")
  Run_Flag_Storage_Invariants(&run_flags, "judge_close.run_flags")
  Read_Handle_Invariants(read, "judge_close.read")
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
  // The module file closes on a zero byte, which the store past the widest source holds.
  if !run_flags.Values[RUN_MODULE] {
    run_flags.Values[RUN_MODULE] = true
    memory.Source[min(int(read.Source_Count), SOURCE_STORAGE_SIZE-1)] = 0
    module_read(memory)
    return CLOSE_PASS_CONFIGURED
  }
  return CLOSE_PASS_CONTRACT
}

// Decodes the configuration the run read. A file that does not decode states its fault, and the
// run fails before the walk: the key it stands under, what it is, the text it names, and the byte a
// source that is no JSON value stops on. The text is untrusted, thus a byte that steers a terminal
// stands as a mark and the quote ends where the message store does.
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
  Run_Flag_Storage_Invariants(&run_flags, "configure.run_flags")
  Configured_Count_Invariants(count, "configure.count")
  Outcome_Handle_Invariants(outcome, "configure.outcome")
  run_flags.Values[RUN_CONFIGURED] = true
  source := configuration.Source(memory.Source[:count])
  fault := configuration.Parse(memory.Configuration, source)
  if fault.Kind == configuration.FAULT_NONE {
    naming.Words_Build(memory.Lanes[0].Memory.Name_Checker.Words, memory.Configuration)
    return Configured_Pass(CLOSE_PASS_CONFIGURED)
  }
  kind := configuration.Recorded_Fault(fault.Kind)
  outcome.Status = EXIT_FAILURE
  line := append(Line(memory.Line[:0]), CONFIGURATION_PREFIX...)
  if bool(configuration.Fault_Keyed(kind)) {
    line = append(append(line, configuration.Key_Text(fault.Key)...), ' ')
  }
  line = append(line, configuration.Fault_Text(kind)...)
  if bool(configuration.Fault_Quoted(kind)) {
    line = append(line, QUOTE_OPENING...)
    // The cells name the text the fault quotes, in the text store.
    decoded := memory.Configuration
    start := decoded.Cells[configuration.CELL_FAULT_START]
    quote := decoded.Text[start:decoded.Cells[configuration.CELL_FAULT_END]]
    for _, symbol := range quote[:min(len(quote), QUOTE_SIZE_MAXIMUM)] {
      if symbol < NAME_BYTE_MINIMUM {
        symbol = requirements.QUOTED_BYTE_ABSENT
      }
      line = append(line, symbol)
    }
    line = append(line, '"')
  }
  if fault.Kind != configuration.FAULT_SYNTAX {
    findings.Line = append(line, '\n')
    return Configured_Pass(CLOSE_PASS_FAILED)
  }
  line = append(line, BYTE_INFIX...)
  // The digits land in the line's spare capacity, since a conversion would allocate.
  size := len(line)
  spare := strconv.Buffer(line[size : size+strconv.DIGIT_TEXT_SIZE_MAXIMUM])
  offset := strconv.Unsigned_Integer(fault.Offset)
  written := strconv.Format_Unsigned_Integer_Into(spare, offset, strconv.DECIMAL_BASE)
  findings.Line = append(line[:size+int(written)], '\n')
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
func Report_File_Input_Invariants(value *Report_File_Input, namespace aver.Namespace) {
  Memory_Handle_Invariants(value.Memory, namespace)
  Findings_Handle_Invariants(value.Findings, namespace)
  Report_Handle_Invariants(value.Report, namespace)
  Reported_Invariants(&value.Written, namespace)
  Run_Flag_Storage_Invariants(&value.Flags, namespace)
  File_Slot_Invariants(value.Slot, namespace)
}

// Writes one file diagnostic, as a text line or a flat JSON line.
func report_file(input Report_File_Input) (wrote Boolean) {
  defer func() { Boolean_Invariants(wrote, "report_file.wrote") }()
  Report_File_Input_Invariants(&input, "report_file.input")
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
  case RULE_NAMES:
    notice := memory.Notice[:bytes.Index_Byte(bytes.Slice(memory.Notice), 0)]
    message = Message_Text(unsafe.String(unsafe.SliceData(notice), len(notice)))
    rule_text = RULE_TEXT_NAMES
  case RULE_SHADOWS:
    notice := memory.Notice[:bytes.Index_Byte(bytes.Slice(memory.Notice), 0)]
    message = Message_Text(unsafe.String(unsafe.SliceData(notice), len(notice)))
    rule_text = RULE_TEXT_SHADOWS
  case RULE_RECURSION:
    notice := memory.Notice[:bytes.Index_Byte(bytes.Slice(memory.Notice), 0)]
    message = Message_Text(unsafe.String(unsafe.SliceData(notice), len(notice)))
    rule_text = RULE_TEXT_RECURSION
  case RULE_LAYOUT:
    notice := memory.Notice[:bytes.Index_Byte(bytes.Slice(memory.Notice), 0)]
    message = Message_Text(unsafe.String(unsafe.SliceData(notice), len(notice)))
    rule_text = RULE_TEXT_LAYOUT
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
  // The name views the path, since a copy of one past a few words would allocate.
  path_steady(Untrusted_Name(unsafe.String(unsafe.SliceData(path), len(path))))
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
  run_flags Run_Flag_Storage,
) (next File_Pass) {
  defer func() { File_Pass_Invariants(next, "file_next.next") }()
  Run_Flag_Storage_Invariants(&run_flags, "file_next.run_flags")
  Served_Reading_Handle_Invariants(reading, "file_next.reading")
  Memory_Handle_Invariants(memory, "file_next.memory")
  Package_Count_Invariants(packages, "file_next.packages")
  File_Count_Invariants(served, "file_next.served")
  File_Cursor_Handle_Invariants(file_cursor, "file_next.file_cursor")
  // The check pass reads each file for its own bytes, whether or not it opens one.
  *reading = Served_Reading(READING_SOURCE)
  // A diagnostic the declaration pass left stands final: a clean file states nothing, a file the
  // pass fixed states nothing, and any other states itself.
  for file_cursor.File_Index < File_Index(served) {
    left := memory.Diagnostics[file_cursor.File_Index]
    stated := left.Tier != TIER_NONE
    if left.Tier == TIER_ONE {
      if run_flags.Values[RUN_AUTOFIX] {
        stated = false
      }
    }
    if stated {
      return File_Pass(STAGE_RECALL)
    }
    file_cursor.File_Index++
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
  requirements.Syntax_Invariants(&syntax, "impure.syntax")
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
  Run_Flag_Storage_Invariants(&run_flags, "package_step.run_flags")
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
    if run_flags.Values[RUN_HISTORY_READ] {
      return Step_Pass(STAGE_HISTORY_NEXT)
    }
    return Step_Pass(STAGE_HISTORY_HEAD)
  }
  // A run closes on the stage that writes its elapsed seconds, behind the pass line of a run
  // that found nothing.
  outcome.Status = EXIT_SUCCESS
  if bool(found) {
    outcome.Status = EXIT_FAILURE
  }
  passed(memory, findings, elapsed, summary, run_flags, found)
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
  copy(checker.Declarations, found.Declarations[:found.Declared])
  requirements.Load_Tests(checker, found.Declared, found.Read)
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
  reading Checked_Reading_Handle,
  path_store Read_Path_Handle,
  finding Finding_Handle,
) (next Contract_Pass) {
  defer func() { Contract_Pass_Invariants(next, "requirements_next.next") }()
  Read_Path_Handle_Invariants(path_store, "requirements_next.path_store")
  Checked_Reading_Handle_Invariants(reading, "requirements_next.reading")
  Finding_Handle_Invariants(finding, "requirements_next.finding")
  Memory_Handle_Invariants(memory, "requirements_next.memory")
  checker := memory.Requirements
  requirements.Next(checker)
  switch checker.Numbers.Values[requirements.NUMBER_STATE] {
  case requirements.STATE_FINDING:
    return Contract_Pass(STAGE_FINDING)
  case requirements.STATE_TEST:
    *path_store = Read_Path_Slice(read_path(
      *path_store, checker, READ_FILE_TEST,
    ))
    *reading = Checked_Reading(READING_TEST)
    return Contract_Pass(STAGE_OPEN)
  case requirements.STATE_REWRITE:
    // The rewritten contract is the form it should stand in, which the fix carries the way a Go
    // file's canonical form does.
    count := checker.Numbers.Values[requirements.NUMBER_REWRITE_COUNT]
    finding.Fix = Fix(checker.Rewritten[:count])
    *path_store = Read_Path_Slice(read_path(
      *path_store, checker, READ_FILE_MARKDOWN,
    ))
    *reading = Checked_Reading(READING_MARKDOWN)
    return Contract_Pass(STAGE_FIX_OPEN)
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
  Run_Flag_Storage_Invariants(&run_flags, "report_requirement.run_flags")
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

// Writes the elapsed seconds to the millisecond, behind the pass line of a run that found
// nothing. The digits land in the line's spare capacity, since a conversion would allocate.
func passed(
  memory Memory_Handle,
  findings Findings_Handle,
  elapsed Elapsed,
  summary Summary_Handle,
  run_flags Run_Flag_Storage,
  found Boolean,
) {
  Memory_Handle_Invariants(memory, "passed.memory")
  Findings_Handle_Invariants(findings, "passed.findings")
  Elapsed_Invariants(elapsed, "passed.elapsed")
  Run_Flag_Storage_Invariants(&run_flags, "passed.run_flags")
  Summary_Handle_Invariants(summary, "passed.summary")
  Boolean_Invariants(found, "passed.found")
  milliseconds := int64(elapsed) / int64(time.MILLISECOND)
  findings.Line = append(Line(memory.Line[:0]), PASSED...)
  if bool(found) {
    findings.Line = append(Line(memory.Line[:0]), ELAPSED_OPENING...)
  }
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
  findings.Line = Line(summary_write(memory.Json, summary, Passed_Line(findings.Line), found))
}

// The elapsed text already stands in the line, thus the summary views it rather than write it
// twice. The summary lands in run storage for the same reason the report does.
// PASSED_SIZE_MINIMUM is the shortest closing line: the elapsed seconds of a run that found
// something, one second digit and its fraction, and the newline that ends it.
const PASSED_SIZE_MINIMUM = len(ELAPSED_OPENING) + ELAPSED_SIZE_MINIMUM + len("\n")

// PASSED_SIZE_MAXIMUM is the widest: every second digit one run can spend.
const PASSED_SIZE_MAXIMUM = len(PASSED) + ELAPSED_SIZE_MAXIMUM + len("\n")

// Passed_Line is the closing line a run wrote, which the summary reads its seconds back out of.
type Passed_Line []byte

// Passed_Line_Invariants bounds the pass line by the text and the seconds it carries.
func Passed_Line_Invariants(value Passed_Line, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(len(value), PASSED_SIZE_MINIMUM, PASSED_SIZE_MAXIMUM).
    Ensure()
}

// SUMMARY_FIXED_SIZE is every byte of a flat JSON summary of a clean run but the seconds it
// carries.
const SUMMARY_FIXED_SIZE = len(`{"passed":true,"elapsed_sec":""}`)

// SUMMARY_FOUND_SIZE is every byte of one of a run that found something but its seconds.
const SUMMARY_FOUND_SIZE = len(`{"passed":false,"elapsed_sec":""}`)

// SUMMARY_SIZE_MINIMUM holds a clean summary at one second digit, and the newline that ends it.
const SUMMARY_SIZE_MINIMUM = SUMMARY_FIXED_SIZE + ELAPSED_SIZE_MINIMUM + len("\n")

// SUMMARY_SIZE_MAXIMUM holds one of a run that found something at every second digit.
const SUMMARY_SIZE_MAXIMUM = SUMMARY_FOUND_SIZE + ELAPSED_SIZE_MAXIMUM + len("\n")

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
  found Boolean,
) (written Summary_Line) {
  defer func() { Summary_Line_Invariants(written, "summary_write.written") }()
  Json_Storage_Invariants(storage, "summary_write.storage")
  Summary_Handle_Invariants(summary, "summary_write.summary")
  Passed_Line_Invariants(passed, "summary_write.passed")
  Boolean_Invariants(found, "summary_write.found")
  opening_size := len(PASSED)
  if bool(found) {
    opening_size = len(ELAPSED_OPENING)
  }
  elapsed := passed[opening_size : len(passed)-len("\n")]
  seconds_text := unsafe.String(unsafe.SliceData(elapsed), len(elapsed))
  *summary = Summary{Passed: Passed(!bool(found)), Elapsed_Seconds: Elapsed_Text(seconds_text)}
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
  Run_Flag_Storage_Invariants(&run_flags, "fail.run_flags")
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
    Format_Invariants(&format_state, "new_memory.format_state")
    Command_Invariants(&command, "new_memory.command")
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
    memory.Packages[index].Declarations = make(
      requirements.Declarations,
      requirements.HEADING_COUNT_MAXIMUM,
    )
  }
  memory.File_Packages = make(Package_Indices, FILE_COUNT_MAXIMUM)
  memory.Sizes, memory.Order = make(File_Sizes, SIZE_TABLE_SIZE), make(
    File_Order,
    FILE_COUNT_MAXIMUM,
  )
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
    Tokens: make(configuration.Tokens, configuration.TOKEN_COUNT_MAXIMUM),
    Text:   make(configuration.Text, configuration.SOURCE_SIZE_MAXIMUM),
    Values: make(configuration.Values, configuration.VALUE_COUNT_MAXIMUM),
    Cells:  make(configuration.Cells, configuration.CELL_COUNT),
  }
  memory.Notice = make(Notice_Store, MESSAGE_SIZE_MAXIMUM)
  memory.Diagnostics = make(File_Diagnostics, FILE_COUNT_MAXIMUM)
  memory.Notices = make(Notice_Table, FILE_COUNT_MAXIMUM)
  for slot := range memory.Notices {
    memory.Notices[slot] = make(Notice_Store, MESSAGE_SIZE_MAXIMUM)
  }
  memory.Siblings = New_Siblings()
  memory.Layout = New_Layout()
  memory.Graph, memory.Glob = make(Graph_Store, GRAPH_STORE_SIZE), make(
    Glob_Storage,
    GLOB_STORAGE_SIZE,
  )
  memory.Orphans = make(Orphan_Paths, ORPHAN_COUNT_MAXIMUM)
  for index := range memory.Orphans {
    memory.Orphans[index] = make(filepath.Slice, 0, filepath.PATH_SIZE_MAXIMUM)
  }
  memory.Link = make(Link_Path, filepath.PATH_SIZE_MAXIMUM)
  memory.Resolved = make(Resolved_Path, filepath.PATH_SIZE_MAXIMUM)
  memory.Target = make(Link_Target, LINK_TARGET_SIZE)
  // Each lane views every store above, thus the table stands past them.
  memory.Lanes, memory.Held = make(Lane_Table, LANE_COUNT), make(Lane_Marks, LANE_COUNT)
  New_Lanes(memory, format_state)
  return memory, format_state, command
}

// LANE_COUNT is how many Go files the spread holds at once. A lane holds one file's check stores
// whole, thus the count bounds what a run holds for its spread.
const LANE_COUNT = 16

// Lane is what one Go file's check stands on apart from every other: its source, its form, its
// notice, its parse, its format places, and its printer. Its memory views the run's stores for the
// rest, which the check reads, or writes at its file's own slot alone.
type Lane struct {
  // Memory views the run's stores, save the source, the form, and the notice.
  Memory Memory
  // Syntax is the lane's parse state and line index.
  Syntax requirements.Syntax
  // Format is the lane's format places and printer, over the run's name table.
  Format Format
  // Slot is the file the lane holds.
  Slot File_Slot
  // Count is how many bytes of the file the source holds.
  Count Source_Count
  // Fix views the canonical form the check wrote, which the lane's next file overwrites.
  Fix Fix
  // Index is the lane's own slot of the table.
  Index Lane_Index
}

// Lane_Invariants composes one lane.
func Lane_Invariants(value *Lane, namespace aver.Namespace) {
  Memory_Invariants(&value.Memory, namespace)
  requirements.Syntax_Invariants(&value.Syntax, namespace)
  Format_Invariants(&value.Format, namespace)
  File_Slot_Invariants(value.Slot, namespace)
  Source_Count_Invariants(value.Count, namespace)
  Fix_Invariants(value.Fix, namespace)
  Lane_Index_Invariants(value.Index, namespace)
}

// Lane_Handle is one lane of the table.
type Lane_Handle *Lane

// Lane_Handle_Invariants composes a present lane.
func Lane_Handle_Invariants(value Lane_Handle, namespace aver.Namespace) {
  if value == nil {
    return
  }
  Lane_Invariants(value, namespace)
}

// Lane_Marks holds one mark for each lane of the table.
type Lane_Marks []Mark

// Lane_Marks_Invariants fixes the table to one mark per lane.
func Lane_Marks_Invariants(value Lane_Marks, _ aver.Namespace) {
  aver.Always(len(value) == LANE_COUNT, "Lane marks have complete length.")
  aver.Always(cap(value) == LANE_COUNT, "Lane marks cannot grow.")
}

// Lane_Table holds one lane for each file the spread holds at once.
type Lane_Table []Lane

// Lane_Table_Invariants fixes the table. Each lane views the memory that holds the table, thus the
// table states its length alone rather than each lane again.
func Lane_Table_Invariants(value Lane_Table, _ aver.Namespace) {
  aver.Always(len(value) == LANE_COUNT, "A lane table has complete length.")
  aver.Always(cap(value) == LANE_COUNT, "A lane table cannot grow.")
}

// New_Lanes fills each lane of the table with the stores it holds apart, and a view of every other
// store of the memory, thus it stands past the rest of the memory.
func New_Lanes(memory Memory_Handle, format_state Format) {
  Memory_Handle_Invariants(memory, "new_lanes.memory")
  Format_Invariants(&format_state, "new_lanes.format_state")
  words := make(naming.Word_Table, naming.WORD_SLOT_COUNT)
  for index := range memory.Lanes {
    lane := &memory.Lanes[index]
    lane.Memory = *memory
    lane.Index = Lane_Index(index)
    // A lane checks no contract, and the run's checker moves with every file it opens.
    lane.Memory.Requirements = nil
    lane.Memory.Source = make(File_Bytes, SOURCE_STORAGE_SIZE)
    lane.Memory.Form = make(Form_Bytes, FORM_STORAGE_SIZE)
    lane.Memory.Notice = make(Notice_Store, MESSAGE_SIZE_MAXIMUM)
    tree := &ast.Syntax_Tree{}
    tree.Tokens.Values = make(ast.Tokens, ast.TOKEN_COUNT_MAXIMUM)
    tree.Nodes.Values = make(ast.Nodes, ast.NODE_COUNT_MAXIMUM)
    tree.Parents.Values = make(ast.Parents, ast.DEPTH_MAXIMUM)
    tree.Last_Children.Values = make(ast.Last_Children, ast.DEPTH_MAXIMUM)
    tree.Earlier_Children.Values = make(ast.Earlier_Children, ast.DEPTH_MAXIMUM)
    starts := make(token.Line_Starts, token.LINE_COUNT_MAXIMUM)
    lane.Syntax = requirements.Syntax{Syntax_Tree: tree, Lines: &token.Line_Index{Starts: starts}}
    // The lanes share one word table, which the run builds once the configuration decodes.
    lane.Memory.Name_Checker = &naming.Checker{
      Nodes:   naming.Node_View(tree.Nodes.Values),
      Tokens:  naming.Token_View(tree.Tokens.Values),
      Source:  naming.Source_Store(lane.Memory.Source),
      Path:    make(naming.Path_Store, naming.PATH_SIZE_MAXIMUM),
      Values:  naming.Decoded_Values(memory.Configuration.Values),
      Text:    naming.Decoded_Text(memory.Configuration.Text),
      Words:   words,
      Terms:   make(naming.Terms, ast.NODE_COUNT_MAXIMUM),
      Message: naming.Message_Store(lane.Memory.Notice),
      Records: make(naming.Record_Table, naming.RECORD_COUNT_MAXIMUM),
      Names:   make(naming.Scope_Names, ast.NODE_COUNT_MAXIMUM),
      Frames:  make(naming.Frame_Stack, naming.FRAME_COUNT),
      Cells:   make(naming.State_Cells, naming.CELL_COUNT),
    }
    lane.Format.Printer = new_print()
    lane.Format.Formatter.Places.Values = make(format.Places, format.PLACE_SLOT_COUNT)
    lane.Format.Formatter.Table = format_state.Formatter.Table
  }
}

// Check_Lane checks the Go file one lane holds and leaves its diagnostic, with the notice it
// states, at the file's own slot.
func Check_Lane(lane Lane_Handle) {
  Lane_Handle_Invariants(lane, "check_lane.lane")
  memory := Memory_Handle(&lane.Memory)
  diagnostic := file_check(memory, lane.Syntax, lane.Slot, lane.Count, lane.Format)
  // The fix views the lane's form, which the lane's next file overwrites, thus the table
  // holds none and the lane keeps it until the pass closes the lane.
  lane.Fix, diagnostic.Fix = diagnostic.Fix, nil
  memory.Diagnostics[lane.Slot] = diagnostic
  copy(memory.Notices[lane.Slot], memory.Notice)
  if !bool(test_file(memory, lane.Slot)) {
    return
  }
  // The contract check reads the declarations of its test file out of this one parse of it, and
  // the package's own slot holds them, which no other lane writes.
  owner := &memory.Packages[memory.File_Packages[lane.Slot]]
  owner.Declared, owner.Read = 0, requirements.TEST_REFUSED
  if !bool(tree_taken(diagnostic.Rule)) {
    return
  }
  source := requirements.Parsed_Source(lane.Memory.Source[:lane.Count])
  owner.Declared = requirements.Collect(lane.Syntax, source, owner.Declarations)
  owner.Read = requirements.TEST_PARSED
}

// Reports whether the Go file in one slot is a package's requirements_test.go.
func test_file(memory Memory_Handle, slot File_Slot) (held Boolean) {
  defer func() { Boolean_Invariants(held, "test_file.held") }()
  Memory_Handle_Invariants(memory, "test_file.memory")
  File_Slot_Invariants(slot, "test_file.slot")
  path := memory.Paths[slot]
  name := path[max(len(path)-len(requirements.TEST_NAME), 0):]
  if string(name) != requirements.TEST_NAME {
    return false
  }
  if len(path) == len(name) {
    return true
  }
  return Boolean(path[len(path)-len(name)-1] == PATH_SEPARATOR)
}

// Reports whether a check that stated the rule stood on a tree the parser took.
func tree_taken(rule Rule) (taken Boolean) {
  defer func() { Boolean_Invariants(taken, "tree_taken.taken") }()
  Rule_Invariants(rule, "tree_taken.rule")
  switch rule {
  case RULE_OVERSIZED, RULE_REFUSED, RULE_CONFLICTED:
    return false
  }
  return true
}

// Lane_Oldest is the reap of a spread that checks each lane as it is submitted: every lane the run
// holds is checked already, thus it hands back the one whose file the run read first, which keeps
// the run in file order.
func Lane_Oldest(memory Memory_Handle) (oldest Lane_Index) {
  defer func() { Lane_Index_Invariants(oldest, "lane_oldest.oldest") }()
  Memory_Handle_Invariants(memory, "lane_oldest.memory")
  found := false
  for index := range memory.Lanes {
    if !bool(memory.Held[index]) {
      continue
    }
    if !found {
      oldest, found = Lane_Index(index), true
      continue
    }
    if memory.Lanes[index].Slot < memory.Lanes[oldest].Slot {
      oldest = Lane_Index(index)
    }
  }
  aver.Always(found, "A reap stands on a lane the run holds.")
  return oldest
}

// Copies the Go file the declaration pass read into its lane, with format places that hold none of
// the table, since the pass takes names while lanes format.
func lane_load(memory Memory_Handle, lane Lane_Handle, count Source_Count) {
  Memory_Handle_Invariants(memory, "lane_load.memory")
  Lane_Handle_Invariants(lane, "lane_load.lane")
  Source_Count_Invariants(count, "lane_load.count")
  copy(lane.Memory.Source, memory.Source[:count])
  lane.Count = count
  clear(lane.Format.Formatter.Places.Values)
  // A file the check never reads leaves no name of its body to record.
  cells := lane.Memory.Name_Checker.Cells
  cells[naming.CELL_RECORD_COUNT], cells[naming.CELL_LOCAL_COUNT] = 0, 0
}

// Lane_Input is what one step of the declaration pass stands on: the spread, the finding a fix
// writes, the name table, and the run's flags.
type Lane_Input struct {
  // Spread checks the lanes.
  Spread Spread
  // Finding holds the fix one closed lane owes.
  Finding Finding_Handle
  // Formatter holds the name table.
  Formatter Formatter_Stores
  // Flags holds the run's flags.
  Flags Run_Flag_Storage
}

// Lane_Input_Invariants composes one step's input.
func Lane_Input_Invariants(value *Lane_Input, namespace aver.Namespace) {
  Spread_Invariants(&value.Spread, namespace)
  Finding_Handle_Invariants(value.Finding, namespace)
  Formatter_Stores_Invariants(&value.Formatter, namespace)
  Run_Flag_Storage_Invariants(&value.Flags, namespace)
}

// Lane_Index is one lane of the table.
type Lane_Index int

// Lane_Index_Invariants admits each lane.
func Lane_Index_Invariants(value Lane_Index, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(int(value), LANE_INDEX_MINIMUM, LANE_INDEX_MAXIMUM).
    Ensure()
}

// Held_Count is how many lanes the run holds.
type Held_Count int

// Held_Count_Invariants bounds the count by the lane table.
func Held_Count_Invariants(value Held_Count, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(int(value), LANE_INDEX_MINIMUM, LANE_COUNT).
    Ensure()
}

// Counts the lanes the run holds.
func lanes_held(memory Memory_Handle) (count Held_Count) {
  defer func() { Held_Count_Invariants(count, "lanes_held.count") }()
  Memory_Handle_Invariants(memory, "lanes_held.memory")
  for _, held := range memory.Held {
    if held {
      count++
    }
  }
  return count
}

// LANE_INDEX_MINIMUM is the first lane.
const LANE_INDEX_MINIMUM = 0

// LANE_INDEX_MAXIMUM is the last lane.
const LANE_INDEX_MAXIMUM = LANE_COUNT - 1

// Declared_Pass is the stage one declared file leaves the run on: the step past it, or the fix its
// rename owes.
type Declared_Pass Stage

// Declared_Pass_Invariants admits the two stages one declared file leaves the run on.
func Declared_Pass_Invariants(value Declared_Pass, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_Int(int(value), int(STAGE_FIX_OPEN), int(STAGE_DECLARE)).
    Ensure()
}

// Hands the file the declaration pass read to a lane the pass holds no file in. Past the pass, the
// file is read for its rename alone: its bytes are the form its one parse printed, thus
// the rename renames them by the whole table, and a file the rename changes is off its canonical
// form, which -fix writes.
func declared(
  memory Memory_Handle,
  input Lane_Input,
  slot File_Slot,
  count Source_Count,
) (next Declared_Pass) {
  defer func() { Declared_Pass_Invariants(next, "declared.next") }()
  Memory_Handle_Invariants(memory, "declared.memory")
  Lane_Input_Invariants(&input, "declared.input")
  File_Slot_Invariants(slot, "declared.slot")
  Source_Count_Invariants(count, "declared.count")
  if input.Flags.Values[RUN_RENAME] {
    source := memory.Source[:count]
    view := format.Formatter{Places: input.Formatter.Places, Table_Fields: input.Formatter.Table}
    copied := printer.Form_Count(copy(memory.Form, source))
    result := format.Rename(&view, printer.Form(memory.Form), copied)
    form := memory.Form[:result.Count]
    if !bool(result.OK) {
      memory.Diagnostics[slot] = Diagnostic{Tier: TIER_TWO, Rule: RULE_UNPRINTABLE}
      return Declared_Pass(STAGE_DECLARE)
    }
    if string(form) == string(source) {
      return Declared_Pass(STAGE_DECLARE)
    }
    memory.Diagnostics[slot] = Diagnostic{Tier: TIER_ONE, Rule: RULE_UNFORMATTED}
    if !input.Flags.Values[RUN_AUTOFIX] {
      return Declared_Pass(STAGE_DECLARE)
    }
    input.Finding.Fix = Fix(form)
    // The fix moves the lines the parse collected declarations at, as the fix of its form does.
    if bool(test_file(memory, slot)) {
      memory.Packages[memory.File_Packages[slot]].Read = requirements.TEST_REWRITTEN
    }
    return Declared_Pass(STAGE_FIX_OPEN)
  }
  // The pass read this file once a lane stood free.
  free := 0
  for free < LANE_COUNT-1 {
    if !bool(memory.Held[free]) {
      break
    }
    free++
  }
  lane, held := &memory.Lanes[free], &memory.Held[free]
  aver.Always(!bool(*held), "A lane takes a file once the pass closed the one it held.")
  lane.Slot = slot
  lane_load(memory, lane, count)
  *held = true
  input.Spread.Submit(lane)
  return Declared_Pass(STAGE_DECLARE)
}

// Closes one lane the pass holds, once the spread checked it. A lane whose file is off its
// canonical form under -fix owes that form to the file, thus the fix stands next.
func lane_turn(
  memory Memory_Handle,
  input Lane_Input,
  index Lane_Index,
  path_store Read_Path_Handle,
) (owed Boolean) {
  defer func() { Boolean_Invariants(owed, "lane_turn.owed") }()
  Memory_Handle_Invariants(memory, "lane_turn.memory")
  Lane_Input_Invariants(&input, "lane_turn.input")
  Lane_Index_Invariants(index, "lane_turn.index")
  Read_Path_Handle_Invariants(path_store, "lane_turn.path_store")
  lane := &memory.Lanes[index]
  lane_close(lane, input.Formatter, input.Flags)
  memory.Held[index] = false
  if memory.Diagnostics[lane.Slot].Tier != TIER_ONE {
    return false
  }
  if !input.Flags.Values[RUN_AUTOFIX] {
    return false
  }
  input.Finding.Fix = lane.Fix
  // The fix moves the lines the parse collected declarations at, thus the contract check of the
  // package waits for the next run.
  if bool(test_file(memory, lane.Slot)) {
    memory.Packages[memory.File_Packages[lane.Slot]].Read = requirements.TEST_REWRITTEN
  }
  *path_store = append((*path_store)[:0], memory.Paths[lane.Slot]...)
  return true
}

// Takes what one checked lane leaves the run, in file order: the purity of the package its file
// stands in, and the names the file declares, which the rename past the pass renames by. A file the
// parser never took holds no tree to read names from, thus the table bars every rename.
func lane_close(lane Lane_Handle, formatter Formatter_Stores, run_flags Run_Flag_Storage) {
  Lane_Handle_Invariants(lane, "lane_close.lane")
  Formatter_Stores_Invariants(&formatter, "lane_close.formatter")
  Run_Flag_Storage_Invariants(&run_flags, "lane_close.run_flags")
  memory := Memory_Handle(&lane.Memory)
  rule := memory.Diagnostics[lane.Slot].Rule
  impure(memory, lane.Syntax, rule, lane.Slot, lane.Count)
  siblings_record(memory, lane.Syntax, lane.Slot, lane.Count, tree_taken(rule))
  layout_imports_record(memory, lane.Slot, lane.Count, tree_taken(rule))
  layout_path_record(memory, lane.Slot, lane.Count, tree_taken(rule))
  if !bool(tree_taken(rule)) {
    run_flags.Values[RUN_UNDECLARED] = true
    return
  }
  view := format.Formatter{Places: formatter.Places, Table_Fields: formatter.Table}
  state := ast.Parse_State{Syntax_Tree: *lane.Syntax.Syntax_Tree}
  source := token.Source(lane.Memory.Source[:lane.Count])
  if !bool(format.Declare(&view, &state, source)) {
    run_flags.Values[RUN_UNDECLARED] = true
  }
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
  defer func() { Formatter_Stores_Invariants(&formatter, "new_formatter.formatter") }()
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
  checker.Events = make(requirements.Events, requirements.EVENT_COUNT_MAXIMUM)
  checker.Lines = make(requirements.Rewritten_Lines, requirements.LINE_TABLE_SIZE)
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
func Git_Command_Invariants(value *Git_Command, namespace aver.Namespace) {
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
  Git_Command_Invariants(value, namespace)
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
func History_Walk_Invariants(value *History_Walk, namespace aver.Namespace) {
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
  History_Walk_Invariants(value, namespace)
}

// History is the history check: the walk, and the argument vector of the git it spawned last.
type History struct {
  // Walk is where the check stands.
  Walk History_Walk
  // Command is the argument vector of the git the check spawned last.
  Command Git_Command
}

// History_Invariants composes the history check.
func History_Invariants(value *History, namespace aver.Namespace) {
  History_Walk_Invariants(&value.Walk, namespace)
  Git_Command_Invariants(&value.Command, namespace)
}

// History_Handle is caller-owned history state.
type History_Handle *History

// History_Handle_Invariants composes present history state.
func History_Handle_Invariants(value History_Handle, namespace aver.Namespace) {
  if value == nil {
    return
  }
  History_Invariants(value, namespace)
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

// Turns the stage one git command leaves into the stage the run stands on next: a history git told
// while the lanes still work returns to them, and the run judges it past the packages.
func history_resume(memory Memory_Handle, next History_Pass) (stage History_Resume) {
  defer func() { History_Resume_Invariants(stage, "history_resume.stage") }()
  Memory_Handle_Invariants(memory, "history_resume.memory")
  History_Pass_Invariants(next, "history_resume.next")
  if next != History_Pass(STAGE_HISTORY_NEXT) {
    return History_Resume(next)
  }
  if lanes_held(memory) > 0 {
    return History_Resume(STAGE_DECLARE)
  }
  return History_Resume(next)
}

// History_Resume is the stage one git command leaves the run on past the lanes it may return to:
// the next command, the judging of the history, or the lanes still working.
type History_Resume Stage

// History_Resume_Invariants admits each stage one git command leaves the run on.
func History_Resume_Invariants(value History_Resume, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_4_Int(
    int(value),
    int(STAGE_DECLARE),
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
func Past_Invariants(value *Past, namespace aver.Namespace) {
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
  Run_Flag_Storage_Invariants(&run_flags, "history_next.run_flags")
  History_Walk_Handle_Invariants(history, "history_next.history")
  Past_Invariants(&past, "history_next.past")
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
  Run_Flag_Storage_Invariants(&run_flags, "past_pick.run_flags")
  History_Walk_Handle_Invariants(history, "past_pick.history")
  Past_Invariants(&past, "past_pick.past")
  // A run that states nothing past its packages leaves the report the last finding wrote, which
  // the run after it opens on.
  fix := survey_next(memory, past.Loop)
  if fix != FIX_NONE {
    notice := survey_message(memory, Stated_Fix(fix))
    *report = Report{Path: CONFIGURATION_PATH, Tier: Report_Tier(TIER_TWO), Rule: RULE_TEXT_SLASHES}
    report.Message = Message_Text(notice)
    return true
  }
  if bool(layout_pick(memory, report)) {
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
  Run_Flag_Storage_Invariants(&run_flags, "history_finding.run_flags")
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
  defer func() { Diagnostic_Invariants(&diagnostic, "imports_check.diagnostic") }()
  Memory_Handle_Invariants(memory, "imports_check.memory")
  requirements.Syntax_Invariants(&syntax, "imports_check.syntax")
  File_Slot_Invariants(slot, "imports_check.slot")
  Source_Count_Invariants(count, "imports_check.count")
  Diagnostic_Invariants(&checked, "imports_check.checked")
  diagnostic = checked
  if checked.Tier != TIER_NONE {
    return diagnostic
  }
  gateway := file_gateway(memory, slot)
  view := ast.Parse_State{Syntax_Tree: *syntax.Syntax_Tree}
  source := memory.Source[:count]
  // A clean parse roots the file at the first slot, and each import is a child of that root.
  root := ast.Node_At(&view, ast.INDEX_FIRST)
  aver.Always(root.Kind == ast.NODE_FILE, "A clean parse roots the file at the first slot.")
  child := ast.Index(root.First_Child)
  for ; child != ast.INDEX_ABSENT; child = ast.Index(ast.Node_At(&view, child).Next) {
    node := ast.Node_At(&view, child)
    if node.Kind != ast.NODE_IMPORT {
      continue
    }
    // An import opens at the name it binds, when it binds one, and its path stands after it,
    // thus the node states where the specification opens, as lint states it.
    opening := ast.Token_At(&view, node.Token)
    lexeme := opening
    if opening.Kind != token.KIND_STRING {
      lexeme = ast.Token_At(&view, node.Token+1)
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
    import_notice(memory, Stated_Family(family))
    diagnostic = Diagnostic{Tier: TIER_TWO, Rule: RULE_IMPORTS}
    if !bool(token.Index_Lines(syntax.Lines, token.Source(source))) {
      return diagnostic
    }
    position := token.Position_Of(syntax.Lines, opening.Offset)
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
  // A caller may spell a repository-relative path with leading current-directory segments.
  // Strip only those segments; the gateway still has to match a whole directory below.
  for len(directory) > 1 {
    if directory[:2] != "./" {
      break
    }
    directory = directory[2:]
  }
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
  Run_Flag_Storage_Invariants(&run_flags, "walk_cause.run_flags")
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

// File_Diagnostics holds the diagnostic the declaration pass left for each Go file, one slot each.
type File_Diagnostics []Diagnostic

// File_Diagnostics_Invariants fixes the store.
func File_Diagnostics_Invariants(value File_Diagnostics, _ aver.Namespace) {
  aver.Always(len(value) == FILE_COUNT_MAXIMUM, "A file diagnostic table has complete length.")
  aver.Always(cap(value) == FILE_COUNT_MAXIMUM, "A file diagnostic table cannot grow.")
}

// Notice_Table holds the banned import message the declaration pass left for each Go file.
type Notice_Table []Notice_Store

// Notice_Table_Invariants fixes the store.
func Notice_Table_Invariants(value Notice_Table, _ aver.Namespace) {
  aver.Always(len(value) == FILE_COUNT_MAXIMUM, "A notice table has complete length.")
  aver.Always(cap(value) == FILE_COUNT_MAXIMUM, "A notice table cannot grow.")
}

// States the rule the Go file the run read breaks: the check of its form, then its imports, and
// records a package main file as it goes.
func file_check(
  memory Memory_Handle,
  syntax requirements.Syntax,
  slot File_Slot,
  count Source_Count,
  format_state Format,
) (diagnostic Diagnostic) {
  defer func() { Diagnostic_Invariants(&diagnostic, "file_check.diagnostic") }()
  Memory_Handle_Invariants(memory, "file_check.memory")
  requirements.Syntax_Invariants(&syntax, "file_check.syntax")
  File_Slot_Invariants(slot, "file_check.slot")
  Source_Count_Invariants(count, "file_check.count")
  Format_Invariants(&format_state, "file_check.format_state")
  diagnostic = Check(memory, syntax, count, format_state)
  diagnostic = imports_check(memory, syntax, slot, count, diagnostic)
  return names_check(memory, syntax, slot, count, diagnostic)
}

// Checks the names of the Go file the run read by harness/pkg/naming, and states the first that
// breaks a rule. A file the check or its imports already faulted keeps that diagnostic alone.
func names_check(
  memory Memory_Handle,
  syntax requirements.Syntax,
  slot File_Slot,
  count Source_Count,
  checked Diagnostic,
) (diagnostic Diagnostic) {
  defer func() { Diagnostic_Invariants(&diagnostic, "names_check.diagnostic") }()
  Memory_Handle_Invariants(memory, "names_check.memory")
  requirements.Syntax_Invariants(&syntax, "names_check.syntax")
  File_Slot_Invariants(slot, "names_check.slot")
  Source_Count_Invariants(count, "names_check.count")
  Diagnostic_Invariants(&checked, "names_check.checked")
  diagnostic = checked
  if checked.Tier != TIER_NONE {
    return diagnostic
  }
  checker := memory.Name_Checker
  checker.Cells[naming.CELL_PATH_SIZE] = naming.State_Cell(copy(checker.Path, memory.Paths[slot]))
  if !bool(naming.Check(checker)) {
    return diagnostic
  }
  diagnostic = Diagnostic{Tier: TIER_TWO, Rule: RULE_NAMES}
  if checker.Cells[naming.CELL_SHADOW] == 1 {
    diagnostic.Rule = RULE_SHADOWS
  }
  if !bool(token.Index_Lines(syntax.Lines, token.Source(memory.Source[:count]))) {
    return diagnostic
  }
  position := token.Position_Of(syntax.Lines, token.Offset(checker.Cells[naming.CELL_OFFSET]))
  diagnostic.Line = Diagnostic_Line(position.Line)
  diagnostic.Column = Diagnostic_Column(position.Column)
  return diagnostic
}

// Recall_Pass is the stage one recalled diagnostic leaves the run on: its write, or the failure of
// a line past flat JSON.
type Recall_Pass Stage

// Recall_Pass_Invariants admits the two stages one recalled diagnostic leaves.
func Recall_Pass_Invariants(value Recall_Pass, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_Int(int(value), int(STAGE_REPORT), int(STAGE_FAIL)).
    Ensure()
}

// States the diagnostic the declaration pass left for one Go file, with the banned import message
// it left beside it, as a text line or a flat JSON line. A line past flat JSON fails the report.
func file_recall(
  memory Memory_Handle,
  findings Findings_Handle,
  report Report_Handle,
  tally Tally_Handle,
  run_flags Run_Flag_Storage,
  slot File_Slot,
  outcome Outcome_Handle,
) (next Recall_Pass) {
  defer func() { Recall_Pass_Invariants(next, "file_recall.next") }()
  Memory_Handle_Invariants(memory, "file_recall.memory")
  Findings_Handle_Invariants(findings, "file_recall.findings")
  Report_Handle_Invariants(report, "file_recall.report")
  Tally_Handle_Invariants(tally, "file_recall.tally")
  Run_Flag_Storage_Invariants(&run_flags, "file_recall.run_flags")
  File_Slot_Invariants(slot, "file_recall.slot")
  Outcome_Handle_Invariants(outcome, "file_recall.outcome")
  left := memory.Diagnostics[slot]
  copy(memory.Notice, memory.Notices[slot])
  tally.Found, tally.Resume = true, RESUME_FILE
  written := Reported{
    Tier:    Reported_Tier(left.Tier),
    Rule:    Reported_Rule(left.Rule),
    Failure: left.Failure,
    Line:    left.Line,
    Column:  left.Column,
  }
  input := Report_File_Input{Memory: memory, Findings: findings, Report: report, Written: written}
  input.Flags, input.Slot = run_flags, slot
  if !bool(report_file(input)) {
    fail(memory, findings, Failure_Path(memory.Paths[slot]), STEP_REPORT, run_flags, outcome)
    return Recall_Pass(STAGE_FAIL)
  }
  return Recall_Pass(STAGE_REPORT)
}

// SIBLING_NAME_COUNT_MAXIMUM bounds the names the run records for the judgment across the files
// of a package: each name a function body declares, and each top-level one. The first slot holds
// no name but the fill of the stores, as the arena of a tree holds no node there.
const SIBLING_NAME_COUNT_MAXIMUM = 1 << 19

// SIBLING_NAME_FIRST is the first slot that holds a name.
const SIBLING_NAME_FIRST = 1

// SIBLING_HEADER is the slot that holds the fill: its start is the next free name, and its end the
// next free byte of the text store.
const SIBLING_HEADER = 0

// SIBLING_TEXT_SIZE_MAXIMUM bounds the bytes of those names and of each package clause.
const SIBLING_TEXT_SIZE_MAXIMUM = 1 << 23

// SIBLING_SLOT_COUNT is the slots of the hash set one judgment fills: twice the names, a power of
// two, thus a probe meets an empty slot.
const SIBLING_SLOT_COUNT = 2 * SIBLING_NAME_COUNT_MAXIMUM

// SIBLING_SLOT_MASK folds a hash into the hash set.
const SIBLING_SLOT_MASK = SIBLING_SLOT_COUNT - 1

// SIBLING_OFFSET_MINIMUM is the first slot of a store, and the empty slot of the hash set.
const SIBLING_OFFSET_MINIMUM = 0

// Sibling_Offset is one place in the stores of the sibling judgment: a name or a byte.
type Sibling_Offset int32

// Sibling_Offset_Invariants bounds the place by the widest store.
func Sibling_Offset_Invariants(value Sibling_Offset, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int32(
    int32(value),
    SIBLING_OFFSET_MINIMUM,
    SIBLING_TEXT_SIZE_MAXIMUM,
  ).
    Ensure()
}

// Sibling_Name is one recorded name: its hash, its bytes in the text store, and, for a name a
// function body declares, where it stands.
type Sibling_Name struct {
  // Hash is the hash of the name's bytes.
  Hash naming.Name_Hash
  // Start is the first byte of the name in the text store.
  Start Sibling_Offset
  // End is one past its last byte.
  End Sibling_Offset
  // Line is the line the name stands on, absent for a top-level name.
  Line Diagnostic_Line
  // Column is the column it stands at, absent for a top-level name.
  Column Diagnostic_Column
  // Call is whether the name stands at a call of the call graph, rather than names a function.
  Call Boolean
}

// Sibling_Name_Invariants composes one name.
func Sibling_Name_Invariants(value *Sibling_Name, namespace aver.Namespace) {
  naming.Name_Hash_Invariants(value.Hash, namespace)
  Sibling_Offset_Invariants(value.Start, namespace)
  Sibling_Offset_Invariants(value.End, namespace)
  Diagnostic_Line_Invariants(value.Line, namespace)
  Diagnostic_Column_Invariants(value.Column, namespace)
  Boolean_Invariants(value.Call, namespace)
}

// Sibling_Names holds every recorded name, past the fill at the first slot.
type Sibling_Names []Sibling_Name

// Sibling_Names_Invariants fixes the store.
func Sibling_Names_Invariants(value Sibling_Names, _ aver.Namespace) {
  aver.Always(len(value) == SIBLING_NAME_COUNT_MAXIMUM, "A sibling name table has complete length.")
  aver.Always(cap(value) == SIBLING_NAME_COUNT_MAXIMUM, "A sibling name table cannot grow.")
}

// Sibling_Text holds the bytes of the recorded names and package clauses.
type Sibling_Text []byte

// Sibling_Text_Invariants fixes the store.
func Sibling_Text_Invariants(value Sibling_Text, _ aver.Namespace) {
  aver.Always(len(value) == SIBLING_TEXT_SIZE_MAXIMUM, "A sibling text store has complete length.")
  aver.Always(cap(value) == SIBLING_TEXT_SIZE_MAXIMUM, "A sibling text store cannot grow.")
}

// Sibling_File is what one Go file recorded: its names, the names its bodies declare first, and
// its package clause, and whether the walk found it and the judgment reads it.
type Sibling_File struct {
  // Start is the file's first name.
  Start Sibling_Offset
  // Locals is one past the last name its bodies declare, and the first top-level one.
  Locals Sibling_Offset
  // Graph is the first name of its call graph: each function and method, each followed by the
  // calls its body makes.
  Graph Sibling_Offset
  // End is one past its last name.
  End Sibling_Offset
  // Clause_Start is the first byte of its package clause in the text store.
  Clause_Start Sibling_Offset
  // Clause_End is one past its last byte.
  Clause_End Sibling_Offset
  // Walked is whether the walk found the file.
  Walked Boolean
  // Recorded is whether the file's names stand in the table.
  Recorded Boolean
  // Judged is whether the judgment reads the file against its siblings, or read its package's
  // call graph.
  Judged Boolean
  // Member is whether the file stands in the package whose call graph the judgment reads.
  Member Boolean
  // Imports_End is one past the last import the file makes, which stand from End.
  Imports_End Sibling_Offset
  // Soft is the name standing for the first clock, stdout, or network selector, or none.
  Soft Sibling_Offset
  // Transitive is the name standing for the first selector that reaches one, or none.
  Transitive Sibling_Offset
  // Mains is how many free functions named Main the file declares.
  Mains Sibling_Offset
  // Path_Start is how many bytes of leading current directories the file's path spends.
  Path_Start Sibling_Offset
  // Top_End is one past the last byte of the file's component in its path, or Path_Start for a
  // file at the root, which stands in none.
  Top_End Sibling_Offset
  // Canonical_Start is the first byte of the file's directory in its component, each major version
  // segment aside, in the text store.
  Canonical_Start Sibling_Offset
  // Canonical_End is one past its last byte.
  Canonical_End Sibling_Offset
  // Clause_Line is the line the package clause's name stands on.
  Clause_Line Diagnostic_Line
  // Clause_Column is the column it stands at.
  Clause_Column Diagnostic_Column
  // Test is whether the file is a test file.
  Test Boolean
}

// Sibling_File_Invariants composes one file.
func Sibling_File_Invariants(value *Sibling_File, namespace aver.Namespace) {
  Sibling_Offset_Invariants(value.Start, namespace)
  Sibling_Offset_Invariants(value.Locals, namespace)
  Sibling_Offset_Invariants(value.Graph, namespace)
  Sibling_Offset_Invariants(value.End, namespace)
  Sibling_Offset_Invariants(value.Clause_Start, namespace)
  Sibling_Offset_Invariants(value.Clause_End, namespace)
  Boolean_Invariants(value.Walked, namespace)
  Boolean_Invariants(value.Recorded, namespace)
  Boolean_Invariants(value.Judged, namespace)
  Boolean_Invariants(value.Member, namespace)
  Sibling_Offset_Invariants(value.Imports_End, namespace)
  Sibling_Offset_Invariants(value.Soft, namespace)
  Sibling_Offset_Invariants(value.Transitive, namespace)
  Sibling_Offset_Invariants(value.Mains, namespace)
  Sibling_Offset_Invariants(value.Path_Start, namespace)
  Sibling_Offset_Invariants(value.Top_End, namespace)
  Sibling_Offset_Invariants(value.Canonical_Start, namespace)
  Sibling_Offset_Invariants(value.Canonical_End, namespace)
  Diagnostic_Line_Invariants(value.Clause_Line, namespace)
  Diagnostic_Column_Invariants(value.Clause_Column, namespace)
  Boolean_Invariants(value.Test, namespace)
}

// Sibling_Files holds what each Go file recorded, slot for slot with Paths.
type Sibling_Files []Sibling_File

// Sibling_Files_Invariants fixes the store.
func Sibling_Files_Invariants(value Sibling_Files, _ aver.Namespace) {
  aver.Always(len(value) == FILE_COUNT_MAXIMUM, "A sibling file table has complete length.")
  aver.Always(cap(value) == FILE_COUNT_MAXIMUM, "A sibling file table cannot grow.")
}

// Sibling_Slots is the hash set of one judgment: each slot holds a recorded name, or none.
type Sibling_Slots []Sibling_Offset

// Sibling_Slots_Invariants fixes the store.
func Sibling_Slots_Invariants(value Sibling_Slots, _ aver.Namespace) {
  aver.Always(len(value) == SIBLING_SLOT_COUNT, "A sibling hash set has complete length.")
  aver.Always(cap(value) == SIBLING_SLOT_COUNT, "A sibling hash set cannot grow.")
}

// Sibling_Table holds the stores of the judgment of each package's files against each other.
type Sibling_Table struct {
  // Names holds every recorded name.
  Names Sibling_Names
  // Text holds their bytes.
  Text Sibling_Text
  // Files holds what each Go file recorded.
  Files Sibling_Files
  // Slots is the hash set one judgment fills.
  Slots Sibling_Slots
}

// Sibling_Table_Invariants composes the stores.
func Sibling_Table_Invariants(value *Sibling_Table, namespace aver.Namespace) {
  Sibling_Names_Invariants(value.Names, namespace)
  Sibling_Text_Invariants(value.Text, namespace)
  Sibling_Files_Invariants(value.Files, namespace)
  Sibling_Slots_Invariants(value.Slots, namespace)
}

// Allocates the stores of the sibling judgment.
func New_Siblings() (table Sibling_Table) {
  defer func() { Sibling_Table_Invariants(&table, "new_siblings.table") }()
  return Sibling_Table{
    Names: make(Sibling_Names, SIBLING_NAME_COUNT_MAXIMUM),
    Text:  make(Sibling_Text, SIBLING_TEXT_SIZE_MAXIMUM),
    Files: make(Sibling_Files, FILE_COUNT_MAXIMUM),
    Slots: make(Sibling_Slots, SIBLING_SLOT_COUNT),
  }
}

// Opens the table for the Go files one walk found: each walked, none recorded yet.
func siblings_open(memory Memory_Handle, count File_Count) {
  Memory_Handle_Invariants(memory, "siblings_open.memory")
  File_Count_Invariants(count, "siblings_open.count")
  table := memory.Siblings
  table.Names[SIBLING_HEADER] = Sibling_Name{Start: SIBLING_NAME_FIRST}
  clear(table.Files)
  for slot := range table.Files[:count] {
    table.Files[slot].Walked = true
  }
}

// Records the names of the Go file one lane holds: each name its bodies declare, where it stands,
// then its top-level names, and its package clause. A tree the parser refused, or a file past the
// table or the line index, stays unrecorded, and its package judges no sibling.
func siblings_record(
  memory Memory_Handle,
  syntax requirements.Syntax,
  slot File_Slot,
  count Source_Count,
  taken Boolean,
) {
  Memory_Handle_Invariants(memory, "siblings_record.memory")
  requirements.Syntax_Invariants(&syntax, "siblings_record.syntax")
  File_Slot_Invariants(slot, "siblings_record.slot")
  Source_Count_Invariants(count, "siblings_record.count")
  Boolean_Invariants(taken, "siblings_record.taken")
  if !bool(taken) {
    return
  }
  table, checker := memory.Siblings, memory.Name_Checker
  naming.Record(checker)
  source := memory.Source[:count]
  if !bool(token.Index_Lines(syntax.Lines, token.Source(source))) {
    return
  }
  records := checker.Records[:checker.Cells[naming.CELL_IMPORT_START]]
  locals, graph := int(checker.Cells[naming.CELL_LOCAL_COUNT]), int(
    checker.Cells[naming.CELL_GRAPH_START],
  )
  // A file opted out of the recursion ban adds neither a function nor a call to the graph.
  if bool(recursion_exempt(memory, slot)) {
    records = records[:graph]
  }
  header := table.Names[SIBLING_HEADER]
  start, text := int(header.Start), int(header.End)
  if len(records) > SIBLING_NAME_COUNT_MAXIMUM-start {
    return
  }
  // The package clause is the one identifier the file holds as its own, past any note.
  named := ast.Index(checker.Nodes[ast.INDEX_FIRST].First_Child)
  for checker.Nodes[named].Kind != ast.NODE_IDENTIFIER {
    named = ast.Index(checker.Nodes[named].Next)
  }
  clause := checker.Tokens[checker.Nodes[named].Token]
  file := Sibling_File{Start: Sibling_Offset(start), Walked: true, Recorded: true}
  file.Locals = Sibling_Offset(start + locals)
  file.Graph = Sibling_Offset(start + graph)
  file.End = Sibling_Offset(start + len(records))
  clause_text := source[clause.Offset : clause.Offset+token.Offset(clause.Size)]
  file.Clause_Start = Sibling_Offset(text)
  // A copy stops at the store, thus a name that does not fit leaves the file unrecorded.
  text += copy(table.Text[text:], clause_text)
  file.Clause_End = Sibling_Offset(text)
  if int(file.Clause_End-file.Clause_Start) < len(clause_text) {
    return
  }
  for index, node := range records {
    spelled := checker.Tokens[checker.Nodes[node].Token]
    name := source[spelled.Offset : spelled.Offset+token.Offset(spelled.Size)]
    written := copy(table.Text[text:], name)
    if written < len(name) {
      return
    }
    hash := naming.Name_Hash(naming.HASH_WIDE_BASIS)
    for _, symbol := range name {
      hash = (hash ^ naming.Name_Hash(symbol)) * naming.HASH_WIDE_PRIME
    }
    entry := Sibling_Name{Hash: hash, Start: Sibling_Offset(text)}
    entry.End = Sibling_Offset(text + written)
    // A name a body declares and a call each state where they stand.
    placed := index < locals
    if index > graph-1 {
      entry.Call = Boolean(checker.Nodes[checker.Nodes[node].Parent].Kind == ast.NODE_CALL)
      placed = bool(entry.Call)
    }
    if placed {
      position := token.Position_Of(syntax.Lines, spelled.Offset)
      entry.Line, entry.Column = Diagnostic_Line(position.Line), Diagnostic_Column(position.Column)
    }
    table.Names[start+index] = entry
    text += written
  }
  table.Names[SIBLING_HEADER].Start = Sibling_Offset(start + len(records))
  table.Names[SIBLING_HEADER].End = Sibling_Offset(text)
  table.Files[slot] = file
}

// Judges each clean Go file against the top-level names its siblings declare, once every file
// stands recorded.
func siblings_judge(memory Memory_Handle) {
  Memory_Handle_Invariants(memory, "siblings_judge.memory")
  table := memory.Siblings
  for slot, own := range table.Files {
    if !bool(own.Recorded) {
      continue
    }
    if own.Locals == own.Start {
      continue
    }
    if memory.Diagnostics[slot].Tier != TIER_NONE {
      continue
    }
    table.Files[slot].Judged = true
    if bool(siblings_fill(memory, true)) {
      siblings_probe(memory)
    }
    siblings_fill(memory, false)
    table.Files[slot].Judged = false
  }
}

// Fills the hash set with the top-level names each sibling of the judged Go file declares, or
// empties it again, and reports whether every file of its package stands recorded. A sibling
// stands in the file's package directory under the file's package clause.
func siblings_fill(memory Memory_Handle, fill Boolean) (known Boolean) {
  defer func() { Boolean_Invariants(known, "siblings_fill.known") }()
  Memory_Handle_Invariants(memory, "siblings_fill.memory")
  Boolean_Invariants(fill, "siblings_fill.fill")
  table := memory.Siblings
  judged := 0
  for slot, file := range table.Files {
    if bool(file.Judged) {
      judged = slot
    }
  }
  own := table.Files[judged]
  clause := table.Text[own.Clause_Start:own.Clause_End]
  known = true
  for other, file := range table.Files {
    if other == judged {
      continue
    }
    if !bool(file.Walked) {
      continue
    }
    if memory.File_Packages[other] != memory.File_Packages[judged] {
      continue
    }
    if !bool(file.Recorded) {
      known = false
      continue
    }
    if string(table.Text[file.Clause_Start:file.Clause_End]) != string(clause) {
      continue
    }
    // The top-level names stand between the locals and the call graph.
    for entry := file.Locals; entry < file.Graph; entry++ {
      position := int(table.Names[entry].Hash & SIBLING_SLOT_MASK)
      // Emptying scans past the slots an earlier name gave back, to the one this name holds.
      for range SIBLING_SLOT_COUNT {
        held := table.Slots[position]
        if bool(fill) {
          if held == SIBLING_OFFSET_MINIMUM {
            table.Slots[position] = entry
            break
          }
        } else if held == entry {
          table.Slots[position] = SIBLING_OFFSET_MINIMUM
          break
        }
        position = (position + 1) & SIBLING_SLOT_MASK
      }
    }
  }
  return known
}

// States the first name the bodies of the judged Go file declare that the hash set holds, as the
// shadow of a sibling's top-level name.
func siblings_probe(memory Memory_Handle) {
  Memory_Handle_Invariants(memory, "siblings_probe.memory")
  table := memory.Siblings
  judged := 0
  for slot, file := range table.Files {
    if bool(file.Judged) {
      judged = slot
    }
  }
  own := table.Files[judged]
  for entry := own.Start; entry < own.Locals; entry++ {
    local := table.Names[entry]
    name := table.Text[local.Start:local.End]
    position := int(local.Hash & SIBLING_SLOT_MASK)
    for range SIBLING_SLOT_COUNT {
      held := table.Slots[position]
      if held == SIBLING_OFFSET_MINIMUM {
        break
      }
      global := table.Names[held]
      if global.Hash == local.Hash {
        if string(table.Text[global.Start:global.End]) == string(name) {
          memory.Diagnostics[judged] = Diagnostic{
            Tier:   TIER_TWO,
            Rule:   RULE_SHADOWS,
            Line:   local.Line,
            Column: local.Column,
          }
          notice := memory.Notices[judged]
          written := copy(notice, "The variable \"")
          written += copy(notice[written:], name)
          tail := "\" shadows a variable in an outer scope. Write a different name."
          written += copy(notice[written:], tail)
          notice[min(written, len(notice)-1)] = 0
          return
        }
      }
      position = (position + 1) & SIBLING_SLOT_MASK
    }
  }
}

// GRAPH_NODES is the section of the graph store listing the functions of one package, each by
// its first declaration, in file order.
const GRAPH_NODES = 0

// GRAPH_CALLERS is the section naming the calling function of each call, by its declaration.
const GRAPH_CALLERS = 1 * SIBLING_NAME_COUNT_MAXIMUM

// GRAPH_CALLEES is the section naming the called function of each call.
const GRAPH_CALLEES = 2 * SIBLING_NAME_COUNT_MAXIMUM

// GRAPH_SITES is the section naming the recorded name each call stands at.
const GRAPH_SITES = 3 * SIBLING_NAME_COUNT_MAXIMUM

// GRAPH_ORDER is the section listing the calls grouped by caller, each group in call order.
const GRAPH_ORDER = 4 * SIBLING_NAME_COUNT_MAXIMUM

// GRAPH_STARTS is the section holding, for each function by its declaration, its first call.
const GRAPH_STARTS = 5 * SIBLING_NAME_COUNT_MAXIMUM

// GRAPH_ENDS is the section holding, for each function by its declaration, one past its last call.
const GRAPH_ENDS = 6 * SIBLING_NAME_COUNT_MAXIMUM

// GRAPH_MARKS is the section holding the mark of each function by its declaration: unvisited,
// finished, or on the path at a depth past GRAPH_MARK_PATH.
const GRAPH_MARKS = 7 * SIBLING_NAME_COUNT_MAXIMUM

// GRAPH_PATH is the section holding the functions the search stands in, the first outermost.
const GRAPH_PATH = 8 * SIBLING_NAME_COUNT_MAXIMUM

// GRAPH_CURSORS is the section holding, for each depth of the path, the next call it reads.
const GRAPH_CURSORS = 9 * SIBLING_NAME_COUNT_MAXIMUM

// GRAPH_STATE is the section holding the counts and the cycle one search found, each state
// constant below naming a slot inside it.
const GRAPH_STATE = 10 * SIBLING_NAME_COUNT_MAXIMUM

// GRAPH_NODE_COUNT is how many functions the package declares.
const GRAPH_NODE_COUNT = 0

// GRAPH_EDGE_COUNT is how many calls name one of them.
const GRAPH_EDGE_COUNT = 1

// GRAPH_CYCLE_START is the depth of the path the cycle a call closes opens at.
const GRAPH_CYCLE_START = 2

// GRAPH_CYCLE_END is the depth of the path the call closing the cycle stands at.
const GRAPH_CYCLE_END = 3

// GRAPH_SITE is the recorded name the call closing the cycle stands at.
const GRAPH_SITE = 4

// GRAPH_LOOKUP is the recorded name one lookup reads.
const GRAPH_LOOKUP = 5

// GRAPH_FOUND is the function declaration the lookup found, or none.
const GRAPH_FOUND = 6

// GRAPH_EMPTY is the empty slot of the hash set a lookup that found none stopped at.
const GRAPH_EMPTY = 7

// GRAPH_LAYOUT_SLOT is the Go file the component layout judges.
const GRAPH_LAYOUT_SLOT = 8

// GRAPH_LAYOUT_ANCESTRY is the ancestry of that file's directory.
const GRAPH_LAYOUT_ANCESTRY = 9

// GRAPH_LAYOUT_SHARED is 1 when that file stands in the shared component.
const GRAPH_LAYOUT_SHARED = 10

// GRAPH_LAYOUT_IMPURE is 1 when that file stands in an impure package.
const GRAPH_LAYOUT_IMPURE = 11

// GRAPH_STATE_COUNT is how many slots the state holds.
const GRAPH_STATE_COUNT = 12

// GRAPH_STORE_SIZE holds every section and the state.
const GRAPH_STORE_SIZE = GRAPH_STATE + GRAPH_STATE_COUNT

// GRAPH_MARK_UNVISITED is a function the search has not reached.
const GRAPH_MARK_UNVISITED = 0

// GRAPH_MARK_FINISHED is a function each call of which the search followed.
const GRAPH_MARK_FINISHED = 1

// GRAPH_MARK_PATH is the mark of the outermost function on the path; each depth past it adds one.
const GRAPH_MARK_PATH = 2

// Graph_Store holds the call graph of one package in sections back to back.
type Graph_Store []Sibling_Offset

// Graph_Store_Invariants fixes the store.
func Graph_Store_Invariants(value Graph_Store, _ aver.Namespace) {
  aver.Always(len(value) == GRAPH_STORE_SIZE, "A call graph store has complete length.")
  aver.Always(cap(value) == GRAPH_STORE_SIZE, "A call graph store cannot grow.")
}

// GLOB_PATTERN_SIZE is the part of the glob store one pattern stands in, closed by a zero byte:
// the widest text a configuration holds, and the byte past it.
const GLOB_PATTERN_SIZE = configuration.SOURCE_SIZE_MAXIMUM + 1

// GLOB_STORAGE_SIZE is the pattern's part and the path's past it, each closed by a zero byte.
const GLOB_STORAGE_SIZE = GLOB_PATTERN_SIZE + filepath.PATH_SIZE_MAXIMUM + 1

// Glob_Storage holds one glob pattern and the path it is matched against, each closed by a zero
// byte, which neither a path nor a pattern the run matches holds.
type Glob_Storage []byte

// Glob_Storage_Invariants fixes the store.
func Glob_Storage_Invariants(value Glob_Storage, _ aver.Namespace) {
  aver.Always(len(value) == GLOB_STORAGE_SIZE, "A glob store has complete length.")
  aver.Always(cap(value) == GLOB_STORAGE_SIZE, "A glob store cannot grow.")
}

// Reports whether an opt_out_recursion_ban entry holds the Go file in one slot.
func recursion_exempt(memory Memory_Handle, slot File_Slot) (exempt Boolean) {
  defer func() { Boolean_Invariants(exempt, "recursion_exempt.exempt") }()
  Memory_Handle_Invariants(memory, "recursion_exempt.memory")
  File_Slot_Invariants(slot, "recursion_exempt.slot")
  glob := memory.Glob
  glob[GLOB_PATTERN_SIZE+copy(glob[GLOB_PATTERN_SIZE:GLOB_STORAGE_SIZE-1], memory.Paths[slot])] = 0
  return glob_listed(memory, GLOB_LIST_RECURSION)
}

// GLOB_LIST_IGNORE is the ignore list.
const GLOB_LIST_IGNORE Glob_List = 0

// GLOB_LIST_RECURSION is opt_out_recursion_ban.
const GLOB_LIST_RECURSION Glob_List = 1

// GLOB_LIST_INSTRUMENTATION is instrumentation_packages.
const GLOB_LIST_INSTRUMENTATION Glob_List = 2

// Glob_List is one path list of harness.json a glob match reads.
type Glob_List int

// Glob_List_Invariants admits each list.
func Glob_List_Invariants(value Glob_List, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_3_Int(
    int(value),
    int(GLOB_LIST_IGNORE),
    int(GLOB_LIST_RECURSION),
    int(GLOB_LIST_INSTRUMENTATION),
  ).
    Ensure()
}

// Reports whether an entry of one path list holds the path
// the glob store holds, as lint matches a glob against it: the last entry that matches decides, a
// negated one clearing the match. An entry states the path from the root, thus a leading and a
// closing separator name nothing, and neither does a leading current directory of the path.
func glob_listed(memory Memory_Handle, list Glob_List) (listed Boolean) {
  defer func() { Boolean_Invariants(listed, "glob_listed.listed") }()
  Memory_Handle_Invariants(memory, "glob_listed.memory")
  Glob_List_Invariants(list, "glob_listed.list")
  decoded, glob := memory.Configuration, memory.Glob
  keys := [...]configuration.Key{
    configuration.KEY_IGNORE,
    configuration.KEY_RECURSION_EXEMPT,
    configuration.KEY_INSTRUMENTATION_PACKAGES,
  }
  key := keys[list]
  path := glob[GLOB_PATTERN_SIZE:]
  for string(path[:2]) == "./" {
    copy(path, path[2:])
  }
  for _, value := range decoded.Values[:decoded.Cells[configuration.CELL_VALUE]] {
    if value.Key != key {
      continue
    }
    pattern := decoded.Text[value.Start:value.End]
    negated := false
    if len(pattern) > 0 {
      negated = pattern[0] == '!'
    }
    if negated {
      pattern = pattern[1:]
    }
    for len(pattern) > 0 {
      if pattern[0] != PATH_SEPARATOR {
        break
      }
      pattern = pattern[1:]
    }
    for len(pattern) > 0 {
      if pattern[len(pattern)-1] != PATH_SEPARATOR {
        break
      }
      pattern = pattern[:len(pattern)-1]
    }
    glob[copy(glob[:GLOB_PATTERN_SIZE-1], pattern)] = 0
    if bool(glob_matched(memory)) {
      listed = Boolean(!negated)
    }
  }
  return listed
}

// Reports whether the pattern the glob store holds matches its path: a * stands for any run of
// bytes but a separator, a ** segment for any run of whole segments, and a backslash quotes the
// byte past it. A mismatch resumes past the last * one byte further, else past the last ** one
// segment further, and a pattern whose rest is stars, or a separator and **, matches no rest.
func glob_matched(memory Memory_Handle) (matched Boolean) {
  defer func() { Boolean_Invariants(matched, "glob_matched.matched") }()
  Memory_Handle_Invariants(memory, "glob_matched.memory")
  glob := memory.Glob
  // Each part closes on a zero byte, which the part of its store always holds.
  pattern, path := glob[:GLOB_PATTERN_SIZE], glob[GLOB_PATTERN_SIZE:]
  for index, symbol := range pattern {
    if symbol == 0 {
      pattern = pattern[:index]
      break
    }
  }
  for index, symbol := range path {
    if symbol == 0 {
      path = path[:index]
      break
    }
  }
  at, cursor, star, star_cursor, double, double_cursor := 0, 0, -1, 0, -1, 0
  for range (len(path) + 1) * (len(pattern) + 2) {
    if cursor == len(path) {
      break
    }
    if at+1 < len(pattern) {
      if string(pattern[at:at+2]) == "**" {
        at = at + 2
        if at < len(pattern) {
          if pattern[at] == PATH_SEPARATOR {
            at++
          }
        }
        double, double_cursor, star = at, cursor, -1
        continue
      }
    }
    if at < len(pattern) {
      if pattern[at] == '*' {
        at, star, star_cursor = at+1, at+1, cursor
        continue
      }
      literal, width := pattern[at], 1
      if literal == '\\' {
        if at+1 < len(pattern) {
          literal, width = pattern[at+1], 2
        }
      }
      if literal == path[cursor] {
        at, cursor = at+width, cursor+1
        continue
      }
    }
    if star > -1 {
      if path[star_cursor] != PATH_SEPARATOR {
        star_cursor++
        at, cursor = star, star_cursor
        continue
      }
    }
    if double < 0 {
      return false
    }
    double_cursor++
    // A ** closed by a separator spans whole segments, thus it resumes past the next one.
    if pattern[double-1] == PATH_SEPARATOR {
      for double_cursor < len(path) {
        if path[double_cursor-1] == PATH_SEPARATOR {
          break
        }
        double_cursor++
      }
    }
    at, cursor, star = double, double_cursor, -1
  }
  for at < len(pattern) {
    if pattern[at] == '*' {
      at++
      continue
    }
    if string(pattern[at:]) != "/**" {
      break
    }
    at = len(pattern)
  }
  return Boolean(at == len(pattern))
}

// Judges the call graph of each package once every Go file stands recorded: a package is the
// files of one directory under one package clause, judged once, from its first file, which marks
// each of its files judged.
func recursion_judge(memory Memory_Handle) {
  Memory_Handle_Invariants(memory, "recursion_judge.memory")
  table := memory.Siblings
  for slot, own := range table.Files {
    if !bool(own.Recorded) {
      continue
    }
    if bool(own.Judged) {
      continue
    }
    // A member stands recorded in the file's directory under its package clause.
    clause := table.Text[own.Clause_Start:own.Clause_End]
    for other, file := range table.Files {
      if !bool(file.Recorded) {
        continue
      }
      if memory.File_Packages[other] != memory.File_Packages[slot] {
        continue
      }
      if string(table.Text[file.Clause_Start:file.Clause_End]) == string(clause) {
        table.Files[other].Member = true
      }
    }
    graph_build(memory)
    graph_order(memory)
    graph_search(memory)
    graph_clear(memory)
    for other, file := range table.Files {
      if bool(file.Member) {
        table.Files[other].Judged, table.Files[other].Member = true, false
      }
    }
  }
  for slot := range table.Files {
    table.Files[slot].Judged = false
  }
}

// Builds the call graph of the judged package: each function by its first declaration, in file
// order, then each call that names one, counted for its caller.
func graph_build(memory Memory_Handle) {
  Memory_Handle_Invariants(memory, "graph_build.memory")
  state := memory.Graph[GRAPH_STATE:]
  table, graph := memory.Siblings, memory.Graph
  state[GRAPH_NODE_COUNT], state[GRAPH_EDGE_COUNT] = 0, 0
  for _, file := range table.Files {
    if !bool(file.Member) {
      continue
    }
    for entry := file.Graph; entry < file.End; entry++ {
      if bool(table.Names[entry].Call) {
        continue
      }
      state[GRAPH_LOOKUP] = entry
      graph_find(memory)
      if state[GRAPH_FOUND] != SIBLING_OFFSET_MINIMUM {
        continue
      }
      table.Slots[state[GRAPH_EMPTY]] = entry
      graph[GRAPH_NODES+state[GRAPH_NODE_COUNT]] = entry
      state[GRAPH_NODE_COUNT]++
      graph[GRAPH_STARTS+entry], graph[GRAPH_ENDS+entry] = 0, 0
      graph[GRAPH_MARKS+entry] = GRAPH_MARK_UNVISITED
    }
  }
  for _, file := range table.Files {
    if !bool(file.Member) {
      continue
    }
    caller := Sibling_Offset(SIBLING_OFFSET_MINIMUM)
    for entry := file.Graph; entry < file.End; entry++ {
      state[GRAPH_LOOKUP] = entry
      graph_find(memory)
      found := state[GRAPH_FOUND]
      if !bool(table.Names[entry].Call) {
        caller = found
        continue
      }
      if found == SIBLING_OFFSET_MINIMUM {
        continue
      }
      edge := state[GRAPH_EDGE_COUNT]
      graph[GRAPH_CALLERS+edge], graph[GRAPH_CALLEES+edge] = caller, found
      graph[GRAPH_SITES+edge] = entry
      graph[GRAPH_ENDS+caller]++
      state[GRAPH_EDGE_COUNT]++
    }
  }
}

// Finds the function declaration whose name the recorded name GRAPH_LOOKUP spells, and leaves it
// in GRAPH_FOUND, or none with the empty slot the probe stopped at in GRAPH_EMPTY.
func graph_find(memory Memory_Handle) {
  Memory_Handle_Invariants(memory, "graph_find.memory")
  state := memory.Graph[GRAPH_STATE:]
  table := memory.Siblings
  name := table.Names[state[GRAPH_LOOKUP]]
  text := table.Text[name.Start:name.End]
  state[GRAPH_FOUND] = SIBLING_OFFSET_MINIMUM
  position := int(name.Hash & SIBLING_SLOT_MASK)
  for range SIBLING_SLOT_COUNT {
    held := table.Slots[position]
    if held == SIBLING_OFFSET_MINIMUM {
      state[GRAPH_EMPTY] = Sibling_Offset(position)
      return
    }
    candidate := table.Names[held]
    if candidate.Hash == name.Hash {
      if string(table.Text[candidate.Start:candidate.End]) == string(text) {
        state[GRAPH_FOUND] = held
        return
      }
    }
    position = (position + 1) & SIBLING_SLOT_MASK
  }
}

// Groups the calls of the graph by caller, each group in call order: the count of each caller
// becomes the span of its group.
func graph_order(memory Memory_Handle) {
  Memory_Handle_Invariants(memory, "graph_order.memory")
  state := memory.Graph[GRAPH_STATE:]
  graph := memory.Graph
  offset := Sibling_Offset(SIBLING_OFFSET_MINIMUM)
  for _, node := range graph[GRAPH_NODES : GRAPH_NODES+state[GRAPH_NODE_COUNT]] {
    graph[GRAPH_STARTS+node], offset = offset, offset+graph[GRAPH_ENDS+node]
    graph[GRAPH_ENDS+node] = graph[GRAPH_STARTS+node]
  }
  for edge := range state[GRAPH_EDGE_COUNT] {
    caller := graph[GRAPH_CALLERS+edge]
    graph[GRAPH_ORDER+graph[GRAPH_ENDS+caller]] = edge
    graph[GRAPH_ENDS+caller]++
  }
}

// Searches the graph depth first from each function in file order, and states each call that
// reaches a function on the path, which closes a cycle.
func graph_search(memory Memory_Handle) {
  Memory_Handle_Invariants(memory, "graph_search.memory")
  state := memory.Graph[GRAPH_STATE:]
  graph := memory.Graph
  for _, root := range graph[GRAPH_NODES : GRAPH_NODES+state[GRAPH_NODE_COUNT]] {
    if graph[GRAPH_MARKS+root] != GRAPH_MARK_UNVISITED {
      continue
    }
    depth := Sibling_Offset(0)
    graph[GRAPH_PATH+depth], graph[GRAPH_CURSORS+depth] = root, graph[GRAPH_STARTS+root]
    graph[GRAPH_MARKS+root] = GRAPH_MARK_PATH
    // Each step enters a function, leaves one, or follows a call, each at most once.
    for range 3*SIBLING_NAME_COUNT_MAXIMUM + 1 {
      top := graph[GRAPH_PATH+depth]
      if graph[GRAPH_CURSORS+depth] == graph[GRAPH_ENDS+top] {
        graph[GRAPH_MARKS+top] = GRAPH_MARK_FINISHED
        if depth == 0 {
          break
        }
        depth--
        continue
      }
      edge := graph[GRAPH_ORDER+graph[GRAPH_CURSORS+depth]]
      graph[GRAPH_CURSORS+depth]++
      callee := graph[GRAPH_CALLEES+edge]
      mark := graph[GRAPH_MARKS+callee]
      if mark == GRAPH_MARK_UNVISITED {
        depth++
        graph[GRAPH_PATH+depth], graph[GRAPH_CURSORS+depth] = callee, graph[GRAPH_STARTS+callee]
        graph[GRAPH_MARKS+callee] = GRAPH_MARK_PATH + depth
        continue
      }
      if mark == GRAPH_MARK_FINISHED {
        continue
      }
      state[GRAPH_CYCLE_START], state[GRAPH_CYCLE_END] = mark-GRAPH_MARK_PATH, depth
      state[GRAPH_SITE] = graph[GRAPH_SITES+edge]
      graph_report(memory)
    }
  }
}

// States the cycle the call GRAPH_SITE closes at the file it stands in, unless that file states a
// finding already: the function that calls itself, or each function of the cycle in call order.
func graph_report(memory Memory_Handle) {
  Memory_Handle_Invariants(memory, "graph_report.memory")
  state := memory.Graph[GRAPH_STATE:]
  table, graph := memory.Siblings, memory.Graph
  site := state[GRAPH_SITE]
  owner := 0
  for slot, file := range table.Files {
    if bool(file.Recorded) {
      if file.Graph < site+1 {
        if site < file.End {
          owner = slot
        }
      }
    }
  }
  if memory.Diagnostics[owner].Tier != TIER_NONE {
    return
  }
  call := table.Names[site]
  memory.Diagnostics[owner] = Diagnostic{
    Tier:   TIER_TWO,
    Rule:   RULE_RECURSION,
    Line:   call.Line,
    Column: call.Column,
  }
  notice := memory.Notices[owner]
  start, end := state[GRAPH_CYCLE_START], state[GRAPH_CYCLE_END]
  first := table.Names[graph[GRAPH_PATH+start]]
  written := 0
  if start == end {
    written += copy(notice[written:], "The function ")
    written += copy(notice[written:], table.Text[first.Start:first.End])
    written += copy(notice[written:], " recurses. Write a loop instead.")
    notice[min(written, len(notice)-1)] = 0
    return
  }
  written += copy(notice[written:], "The functions recurse in a cycle. Write a loop instead. ")
  for depth := start; depth < end+1; depth++ {
    node := table.Names[graph[GRAPH_PATH+depth]]
    written += copy(notice[written:], table.Text[node.Start:node.End])
    written += copy(notice[written:], " → ")
  }
  written += copy(notice[written:], table.Text[first.Start:first.End])
  notice[min(written, len(notice)-1)] = 0
}

// Empties the hash set of the functions the graph holds, in the order they entered it: a probe
// scans past the slots an earlier function gave back, to the one this one holds.
func graph_clear(memory Memory_Handle) {
  Memory_Handle_Invariants(memory, "graph_clear.memory")
  state := memory.Graph[GRAPH_STATE:]
  table, graph := memory.Siblings, memory.Graph
  for _, node := range graph[GRAPH_NODES : GRAPH_NODES+state[GRAPH_NODE_COUNT]] {
    position := int(table.Names[node].Hash & SIBLING_SLOT_MASK)
    for range SIBLING_SLOT_COUNT {
      if table.Slots[position] == node {
        table.Slots[position] = SIBLING_OFFSET_MINIMUM
        break
      }
      position = (position + 1) & SIBLING_SLOT_MASK
    }
  }
}

// LAYOUT_FINDING_COUNT_MAXIMUM bounds the findings of the component layout that name no Go file.
// The first slot holds none but the fill: its path start the next free slot, its path end the next
// one to state, and its text the module path.
const LAYOUT_FINDING_COUNT_MAXIMUM = 1024

// LAYOUT_HEADER is the slot that holds the fill.
const LAYOUT_HEADER = 0

// LAYOUT_FINDING_FIRST is the first slot that holds a finding.
const LAYOUT_FINDING_FIRST = 1

// LAYOUT_TEXT_SIZE_MAXIMUM bounds the bytes of the module path and of those findings.
const LAYOUT_TEXT_SIZE_MAXIMUM = 256 * 1024

// Layout_Finding is one finding of the component layout that names no Go file: the path it stands
// at and its message, each in the text store.
type Layout_Finding struct {
  // Path_Start is the first byte of the path.
  Path_Start Sibling_Offset
  // Path_End is one past its last byte.
  Path_End Sibling_Offset
  // Text_Start is the first byte of the message.
  Text_Start Sibling_Offset
  // Text_End is one past its last byte.
  Text_End Sibling_Offset
}

// Layout_Finding_Invariants composes one finding.
func Layout_Finding_Invariants(value *Layout_Finding, namespace aver.Namespace) {
  Sibling_Offset_Invariants(value.Path_Start, namespace)
  Sibling_Offset_Invariants(value.Path_End, namespace)
  Sibling_Offset_Invariants(value.Text_Start, namespace)
  Sibling_Offset_Invariants(value.Text_End, namespace)
}

// Layout_Findings holds those findings past the fill.
type Layout_Findings []Layout_Finding

// Layout_Findings_Invariants fixes the store.
func Layout_Findings_Invariants(value Layout_Findings, _ aver.Namespace) {
  aver.Always(
    len(value) == LAYOUT_FINDING_COUNT_MAXIMUM,
    "A layout finding table has complete length.",
  )
  aver.Always(cap(value) == LAYOUT_FINDING_COUNT_MAXIMUM, "A layout finding table cannot grow.")
}

// Layout_Text holds the module path and the paths and messages of the findings.
type Layout_Text []byte

// Layout_Text_Invariants fixes the store.
func Layout_Text_Invariants(value Layout_Text, _ aver.Namespace) {
  aver.Always(len(value) == LAYOUT_TEXT_SIZE_MAXIMUM, "A layout text store has complete length.")
  aver.Always(cap(value) == LAYOUT_TEXT_SIZE_MAXIMUM, "A layout text store cannot grow.")
}

// Layout_Table holds the findings of the component layout that name no Go file.
type Layout_Table struct {
  // Findings holds the findings past the fill.
  Findings Layout_Findings
  // Text holds their bytes and the module path.
  Text Layout_Text
}

// Layout_Table_Invariants composes the stores.
func Layout_Table_Invariants(value *Layout_Table, namespace aver.Namespace) {
  Layout_Findings_Invariants(value.Findings, namespace)
  Layout_Text_Invariants(value.Text, namespace)
}

// Allocates the stores of the component layout.
func New_Layout() (table Layout_Table) {
  defer func() { Layout_Table_Invariants(&table, "new_layout.table") }()
  return Layout_Table{
    Findings: make(Layout_Findings, LAYOUT_FINDING_COUNT_MAXIMUM),
    Text:     make(Layout_Text, LAYOUT_TEXT_SIZE_MAXIMUM),
  }
}

// Reads the module path out of the module file the source store holds up to its zero byte: the
// first word past a line that opens with module, as lint reads it.
func module_read(memory Memory_Handle) {
  Memory_Handle_Invariants(memory, "module_read.memory")
  source := memory.Source
  table := memory.Layout
  start := 0
  for offset := range len(source) - len("module ") {
    if source[offset] == 0 {
      break
    }
    if offset > 0 {
      if source[offset-1] != '\n' {
        continue
      }
    }
    if string(source[offset:offset+len("module")]) != "module" {
      continue
    }
    past := source[offset+len("module")]
    if past != ' ' {
      if past != '\t' {
        continue
      }
    }
    start = offset + len("module")
    break
  }
  for start > 0 {
    if source[start] != ' ' {
      if source[start] != '\t' {
        break
      }
    }
    start++
  }
  end := start
  for start > 0 {
    symbol := source[end]
    if symbol < '!' {
      break
    }
    end++
  }
  written := copy(table.Text[:filepath.PATH_SIZE_MAXIMUM], source[start:end])
  table.Findings[LAYOUT_HEADER].Text_Start = 0
  table.Findings[LAYOUT_HEADER].Text_End = Sibling_Offset(written)
}

// States that the working directory holds no module file.
func module_absent(memory Memory_Handle) {
  Memory_Handle_Invariants(memory, "module_absent.memory")
  glob := memory.Glob[GLOB_PATTERN_SIZE:GLOB_STORAGE_SIZE]
  glob[copy(glob, MODULE_PATH)] = 0
  layout_record(
    memory,
    "There is no go.mod at the repository root. The linter needs the module anchor.",
    LAYOUT_QUOTE_NONE,
  )
}

// States a module file below the root, or a workspace file, at the path the glob store holds.
func layout_module(memory Memory_Handle, workspace Boolean) {
  Memory_Handle_Invariants(memory, "layout_module.memory")
  Boolean_Invariants(workspace, "layout_module.workspace")
  path := memory.Glob[GLOB_PATTERN_SIZE:GLOB_STORAGE_SIZE]
  for index, symbol := range path {
    if symbol == 0 {
      path = path[:index]
      break
    }
  }
  // A leading current directory names the root, whose module file is the one module.
  for len(path) > 1 {
    if string(path[:2]) != "./" {
      break
    }
    path = path[2:]
  }
  if !bool(workspace) {
    if string(path) == MODULE_PATH {
      return
    }
    layout_record(
      memory,
      "A nested go.mod divides the repository into more than one module. Remove ",
      LAYOUT_QUOTE_PATH,
    )
    return
  }
  layout_record(
    memory,
    "A go.work makes a workspace with more than one module. Remove ",
    LAYOUT_QUOTE_PATH,
  )
}

// LAYOUT_QUOTE_NONE closes a message on its own text.
const LAYOUT_QUOTE_NONE Layout_Quote = 0

// LAYOUT_QUOTE_PATH closes a message on the path it stands at, quoted, and a full stop.
const LAYOUT_QUOTE_PATH Layout_Quote = 1

// LAYOUT_QUOTE_MAIN_ABSENT closes a message on the component the path names, quoted, which
// declares no func Main in pkg.
const LAYOUT_QUOTE_MAIN_ABSENT Layout_Quote = 2

// LAYOUT_QUOTE_MAIN_SEVERAL closes a message on the component, quoted, which declares several.
const LAYOUT_QUOTE_MAIN_SEVERAL Layout_Quote = 3

// Layout_Quote is what closes one message of the component layout past its text.
type Layout_Quote int

// Layout_Quote_Invariants admits each close.
func Layout_Quote_Invariants(value Layout_Quote, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_4_Int(
    int(value),
    int(LAYOUT_QUOTE_NONE),
    int(LAYOUT_QUOTE_PATH),
    int(LAYOUT_QUOTE_MAIN_ABSENT),
    int(LAYOUT_QUOTE_MAIN_SEVERAL),
  ).
    Ensure()
}

// Layout_Message is the text a message of the component layout opens with.
type Layout_Message string

// Layout_Message_Invariants bounds the text by the messages.
func Layout_Message_Invariants(value Layout_Message, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(
    len(value),
    LAYOUT_MESSAGE_SIZE_MINIMUM,
    LAYOUT_MESSAGE_SIZE_MAXIMUM,
  ).
    Ensure()
}

// LAYOUT_MESSAGE_SIZE_MINIMUM is the shortest such text.
const LAYOUT_MESSAGE_SIZE_MINIMUM = len(LAYOUT_COMPONENT_OPENING)

// LAYOUT_COMPONENT_OPENING opens the message of a component's func Main.
const LAYOUT_COMPONENT_OPENING = "The binary component "

// LAYOUT_MESSAGE_SIZE_MAXIMUM is the longest.
const LAYOUT_MESSAGE_SIZE_MAXIMUM = len(
  "There is no go.mod at the repository root. The linter needs the module anchor.",
)

// Records one finding at the path the glob store holds, its leading current directories aside:
// the message opens with the text, and the close follows it. A table or a text store past its
// bound records nothing more.
func layout_record(memory Memory_Handle, message Layout_Message, quote Layout_Quote) {
  Memory_Handle_Invariants(memory, "layout_record.memory")
  Layout_Message_Invariants(message, "layout_record.message")
  Layout_Quote_Invariants(quote, "layout_record.quote")
  table := memory.Layout
  header := table.Findings[LAYOUT_HEADER]
  if int(header.Path_Start) == LAYOUT_FINDING_COUNT_MAXIMUM {
    return
  }
  path := memory.Glob[GLOB_PATTERN_SIZE:GLOB_STORAGE_SIZE]
  for index, symbol := range path {
    if symbol == 0 {
      path = path[:index]
      break
    }
  }
  for len(path) > 1 {
    if string(path[:2]) != "./" {
      break
    }
    path = path[2:]
  }
  fill := LAYOUT_FILL
  if header.Path_Start > LAYOUT_FINDING_FIRST {
    fill = int(table.Findings[header.Path_Start-1].Text_End)
  }
  // The widest finding spends its path twice and a message of the store's width.
  if fill+2*len(path)+MESSAGE_SIZE_MAXIMUM > LAYOUT_TEXT_SIZE_MAXIMUM {
    return
  }
  finding := Layout_Finding{Path_Start: Sibling_Offset(fill)}
  fill += copy(table.Text[fill:], path)
  finding.Path_End, finding.Text_Start = Sibling_Offset(fill), Sibling_Offset(fill)
  written := copy(table.Text[fill:fill+MESSAGE_SIZE_MAXIMUM], message)
  store := table.Text[:fill+MESSAGE_SIZE_MAXIMUM]
  switch quote {
  case LAYOUT_QUOTE_PATH:
    written += copy(store[fill+written:], "\"")
    written += copy(store[fill+written:], path)
    written += copy(store[fill+written:], "\".")
  case LAYOUT_QUOTE_MAIN_ABSENT, LAYOUT_QUOTE_MAIN_SEVERAL:
    written += copy(store[fill+written:], "\"")
    written += copy(store[fill+written:], table.Text[:header.Text_End])
    written += copy(store[fill+written:], "/")
    written += copy(store[fill+written:], path)
    tail := "\" declares no func Main in pkg/. Declare a func Main in pkg/."
    if quote == LAYOUT_QUOTE_MAIN_SEVERAL {
      tail = "\" declares more than one func Main in pkg/. Declare only one func Main."
    }
    written += copy(store[fill+written:], tail)
  }
  finding.Text_End = Sibling_Offset(fill + written)
  table.Findings[header.Path_Start] = finding
  table.Findings[LAYOUT_HEADER].Path_Start++
}

// Picks the next finding of the component layout that names no Go file into the report, and
// reports whether one stands.
func layout_pick(memory Memory_Handle, report Report_Handle) (picked Boolean) {
  defer func() { Boolean_Invariants(picked, "layout_pick.picked") }()
  Memory_Handle_Invariants(memory, "layout_pick.memory")
  Report_Handle_Invariants(report, "layout_pick.report")
  table := memory.Layout
  header := table.Findings[LAYOUT_HEADER]
  if header.Path_End == header.Path_Start {
    return false
  }
  finding := table.Findings[header.Path_End]
  table.Findings[LAYOUT_HEADER].Path_End++
  path := table.Text[finding.Path_Start:finding.Path_End]
  message := table.Text[finding.Text_Start:finding.Text_End]
  *report = Report{Tier: Report_Tier(TIER_TWO), Rule: RULE_TEXT_LAYOUT}
  report.Path = Path_Text(unsafe.String(unsafe.SliceData(path), len(path)))
  report.Message = Message_Text(unsafe.String(unsafe.SliceData(message), len(message)))
  return true
}

// LAYOUT_REGION_SIZE is one region of the layout text store: a path and its zero byte.
const LAYOUT_REGION_SIZE = filepath.PATH_SIZE_MAXIMUM + 1

// LAYOUT_ROOT is the region holding the root the walk opened on.
const LAYOUT_ROOT = LAYOUT_REGION_SIZE

// LAYOUT_SUBJECT is the region holding the directory a layout judgment reads, in its component.
const LAYOUT_SUBJECT = 2 * LAYOUT_REGION_SIZE

// LAYOUT_TOP is the region holding the component of that directory.
const LAYOUT_TOP = 3 * LAYOUT_REGION_SIZE

// LAYOUT_NAMES is the region holding the library ancestors of that directory, space separated.
const LAYOUT_NAMES = 4 * LAYOUT_REGION_SIZE

// LAYOUT_FILL is the first byte of the findings, past the regions.
const LAYOUT_FILL = 6 * LAYOUT_REGION_SIZE

// ANCESTRY_NONE is a directory with no library ancestor: a library package.
const ANCESTRY_NONE Ancestry = 0

// ANCESTRY_ONE is a directory with one: the composition tier.
const ANCESTRY_ONE Ancestry = 1

// ANCESTRY_SEVERAL is a directory with more: below the library tier.
const ANCESTRY_SEVERAL Ancestry = 2

// Ancestry is how many library ancestors a directory stands under, past one counted as several.
type Ancestry int

// Ancestry_Invariants admits each count.
func Ancestry_Invariants(value Ancestry, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_3_Int(
    int(value),
    int(ANCESTRY_NONE),
    int(ANCESTRY_ONE),
    int(ANCESTRY_SEVERAL),
  ).
    Ensure()
}

// Splits the repository path the glob store holds, a directory, into its component, which the top
// region holds, and its directory in that component, each major version segment aside, which the
// subject region holds: a component root stands as ".". Each region closes on a zero byte.
func layout_subject(memory Memory_Handle) {
  Memory_Handle_Invariants(memory, "layout_subject.memory")
  text := memory.Layout.Text
  path := memory.Glob[GLOB_PATTERN_SIZE:GLOB_STORAGE_SIZE]
  for index, symbol := range path {
    if symbol == 0 {
      path = path[:index]
      break
    }
  }
  top_size := len(path)
  for index, symbol := range path {
    if symbol == PATH_SEPARATOR {
      top_size = index
      break
    }
  }
  text[LAYOUT_TOP+copy(text[LAYOUT_TOP:LAYOUT_TOP+LAYOUT_REGION_SIZE-1], path[:top_size])] = 0
  subject := text[LAYOUT_SUBJECT : LAYOUT_SUBJECT+LAYOUT_REGION_SIZE-1]
  size := 0
  for start := top_size + 1; start < len(path)+1; {
    end := start
    for end < len(path) {
      if path[end] == PATH_SEPARATOR {
        break
      }
      end++
    }
    // A major version segment is a v and digits alone.
    segment := path[start:end]
    version := len(segment) > 1
    if version {
      version = segment[0] == 'v'
    }
    for _, symbol := range segment[min(1, len(segment)):] {
      if symbol-'0' > 9 {
        version = false
      }
    }
    if !version {
      if size > 0 {
        size += copy(subject[size:], "/")
      }
      size += copy(subject[size:], segment)
    }
    start = end + 1
  }
  if size == 0 {
    size = copy(subject, ".")
  }
  text[LAYOUT_SUBJECT+size] = 0
}

// Counts the library ancestors of the subject directory in the top component: each directory above
// it that holds a Go package the walk recorded, no test and no main, a binary's top-level pkg
// aside. The names region holds them, nearest first, space separated.
func layout_ancestry(memory Memory_Handle) (ancestry Ancestry) {
  defer func() { Ancestry_Invariants(ancestry, "layout_ancestry.ancestry") }()
  Memory_Handle_Invariants(memory, "layout_ancestry.memory")
  table, text := memory.Siblings, memory.Layout.Text
  subject := text[LAYOUT_SUBJECT : LAYOUT_SUBJECT+LAYOUT_REGION_SIZE]
  subject = subject[:bytes.Index_Byte(bytes.Slice(subject), 0)]
  top := text[LAYOUT_TOP : LAYOUT_TOP+LAYOUT_REGION_SIZE]
  top = top[:bytes.Index_Byte(bytes.Slice(top), 0)]
  shared := layout_shared(memory)
  names := text[LAYOUT_NAMES : LAYOUT_NAMES+2*LAYOUT_REGION_SIZE-1]
  size, count := 0, 0
  for cut := len(subject) - 1; cut > 0; cut-- {
    if subject[cut] != PATH_SEPARATOR {
      continue
    }
    ancestor := subject[:cut]
    if !bool(shared) {
      if string(ancestor) == "pkg" {
        continue
      }
    }
    held := false
    for other, file := range table.Files {
      if !bool(file.Recorded) {
        continue
      }
      if bool(file.Test) {
        continue
      }
      if string(table.Text[file.Clause_Start:file.Clause_End]) == "main" {
        continue
      }
      if string(memory.Paths[other][file.Path_Start:file.Top_End]) != string(top) {
        continue
      }
      if string(table.Text[file.Canonical_Start:file.Canonical_End]) == string(ancestor) {
        held = true
        break
      }
    }
    if held {
      if count > 0 {
        size += copy(names[size:], " ")
      }
      size += copy(names[size:], ancestor)
      count++
    }
  }
  text[LAYOUT_NAMES+size] = 0
  return Ancestry(min(count, int(ANCESTRY_SEVERAL)))
}

// Reports whether the component the top region holds is the shared one harness.json names.
func layout_shared(memory Memory_Handle) (shared Boolean) {
  defer func() { Boolean_Invariants(shared, "layout_shared.shared") }()
  Memory_Handle_Invariants(memory, "layout_shared.memory")
  decoded, text := memory.Configuration, memory.Layout.Text
  top := text[LAYOUT_TOP : LAYOUT_TOP+LAYOUT_REGION_SIZE]
  top = top[:bytes.Index_Byte(bytes.Slice(top), 0)]
  for _, value := range decoded.Values[:decoded.Cells[configuration.CELL_VALUE]] {
    if value.Key != configuration.KEY_SHARED_COMPONENT {
      continue
    }
    named := decoded.Text[value.Start:value.End]
    for len(named) > 1 {
      if string(named[:2]) != "./" {
        break
      }
      named = named[2:]
    }
    for len(named) > 0 {
      if named[len(named)-1] != PATH_SEPARATOR {
        break
      }
      named = named[:len(named)-1]
    }
    return Boolean(string(named) == string(top))
  }
  return false
}

// Records the imports of the Go file one lane holds, each at the import, and the first clock,
// stdout, or network selector and the first selector that reaches one, each as path.name at its
// receiver. A file past the table stays unrecorded.
func layout_imports_record(
  memory Memory_Handle,
  slot File_Slot,
  count Source_Count,
  taken Boolean,
) {
  Memory_Handle_Invariants(memory, "layout_imports_record.memory")
  File_Slot_Invariants(slot, "layout_imports_record.slot")
  Source_Count_Invariants(count, "layout_imports_record.count")
  Boolean_Invariants(taken, "layout_imports_record.taken")
  table, checker := memory.Siblings, memory.Name_Checker
  if !bool(taken) {
    return
  }
  file := table.Files[slot]
  if !bool(file.Recorded) {
    return
  }
  cells, nodes, tokens := checker.Cells, checker.Nodes, checker.Tokens
  source := memory.Source[:count]
  imports := checker.Records[cells[naming.CELL_IMPORT_START]:cells[naming.CELL_RECORD_COUNT]]
  header := table.Names[SIBLING_HEADER]
  start, text := int(header.Start), int(header.End)
  if len(imports)+2 > SIBLING_NAME_COUNT_MAXIMUM-start {
    table.Files[slot].Recorded = false
    return
  }
  // Each import records its path, the text between its quotes, then each selector its own.
  sites := [...]naming.State_Cell{
    0,
    cells[naming.CELL_SOFT_CALL],
    cells[naming.CELL_TRANSITIVE_CALL],
  }
  paths := [...]naming.State_Cell{
    0,
    cells[naming.CELL_SOFT_IMPORT],
    cells[naming.CELL_TRANSITIVE_IMPORT],
  }
  entry := start
  for index := range len(imports) + 2 {
    site, quoted := ast.Index(0), ast.Index(0)
    if index < len(imports) {
      site, quoted = imports[index], ast.Index(nodes[imports[index]].First_Child)
      for nodes[quoted].Kind != ast.NODE_STRING {
        quoted = ast.Index(nodes[quoted].Next)
      }
    }
    if index > len(imports)-1 {
      site, quoted = ast.Index(sites[index-len(imports)+1]), ast.Index(paths[index-len(imports)+1])
      if site == ast.INDEX_ABSENT {
        continue
      }
      // A selector stands at its receiver, the child before the name it selects.
      site = ast.Index(nodes[nodes[site].Parent].First_Child)
    }
    held := tokens[nodes[quoted].Token]
    path := source[held.Offset+1 : held.Offset+token.Offset(held.Size)-1]
    written := copy(table.Text[text:], path)
    if index > len(imports)-1 {
      named := tokens[nodes[nodes[site].Next].Token]
      written += copy(table.Text[text+written:], ".")
      written += copy(
        table.Text[text+written:],
        source[named.Offset:named.Offset+token.Offset(named.Size)],
      )
      if index == len(imports) {
        file.Soft = Sibling_Offset(entry)
      } else {
        file.Transitive = Sibling_Offset(entry)
      }
    }
    if text+written == SIBLING_TEXT_SIZE_MAXIMUM {
      table.Files[slot].Recorded = false
      return
    }
    // The line and the column count the bytes before the site, as the line index does.
    offset := int(tokens[nodes[site].Token].Offset)
    line, opening := 1, 0
    for cursor, symbol := range source[:offset] {
      if symbol == '\n' {
        line, opening = line+1, cursor+1
      }
    }
    table.Names[entry] = Sibling_Name{
      Start:  Sibling_Offset(text),
      End:    Sibling_Offset(text + written),
      Line:   Diagnostic_Line(line),
      Column: Diagnostic_Column(offset - opening + 1),
    }
    text, entry = text+written, entry+1
    if index == len(imports)-1 {
      file.Imports_End = Sibling_Offset(entry)
    }
  }
  if len(imports) == 0 {
    file.Imports_End = file.End
  }
  file.Mains = Sibling_Offset(cells[naming.CELL_MAIN_COUNT])
  table.Names[SIBLING_HEADER].Start, table.Names[SIBLING_HEADER].End = Sibling_Offset(entry), Sibling_Offset(
    text,
  )
  table.Files[slot] = file
}

// Records where the Go file one lane holds stands: its component, its directory in that component,
// whether it is a test, and where its package clause names its package.
func layout_path_record(
  memory Memory_Handle,
  slot File_Slot,
  count Source_Count,
  taken Boolean,
) {
  Memory_Handle_Invariants(memory, "layout_path_record.memory")
  File_Slot_Invariants(slot, "layout_path_record.slot")
  Source_Count_Invariants(count, "layout_path_record.count")
  Boolean_Invariants(taken, "layout_path_record.taken")
  table := memory.Siblings
  if !bool(taken) {
    return
  }
  file := table.Files[slot]
  if !bool(file.Recorded) {
    return
  }
  path := memory.Paths[slot]
  start := 0
  for len(path)-start > 1 {
    if string(path[start:start+2]) != "./" {
      break
    }
    start += 2
  }
  file.Path_Start, file.Top_End = Sibling_Offset(start), Sibling_Offset(start)
  directory := int(bytes.Last_Index_Byte(bytes.Slice(path), PATH_SEPARATOR))
  for index := start; index < len(path); index++ {
    if path[index] == PATH_SEPARATOR {
      file.Top_End = Sibling_Offset(index)
      break
    }
  }
  file.Test = len(path) > len("_test.go")-1
  if bool(file.Test) {
    file.Test = Boolean(string(path[len(path)-len("_test.go"):]) == "_test.go")
  }
  glob := memory.Glob[GLOB_PATTERN_SIZE:GLOB_STORAGE_SIZE]
  glob[copy(glob, path[start:max(directory, start)])] = 0
  layout_subject(memory)
  subject := memory.Layout.Text[LAYOUT_SUBJECT : LAYOUT_SUBJECT+LAYOUT_REGION_SIZE]
  subject = subject[:bytes.Index_Byte(bytes.Slice(subject), 0)]
  text := int(table.Names[SIBLING_HEADER].End)
  written := copy(table.Text[text:], subject)
  if text+written == SIBLING_TEXT_SIZE_MAXIMUM {
    table.Files[slot].Recorded = false
    return
  }
  file.Canonical_Start, file.Canonical_End = Sibling_Offset(text), Sibling_Offset(text+written)
  table.Names[SIBLING_HEADER].End = Sibling_Offset(text + written)
  // The package clause is the one identifier the file holds as its own, past any note.
  nodes := memory.Name_Checker.Nodes
  named := ast.Index(nodes[ast.INDEX_FIRST].First_Child)
  for nodes[named].Kind != ast.NODE_IDENTIFIER {
    named = ast.Index(nodes[named].Next)
  }
  offset := int(memory.Name_Checker.Tokens[nodes[named].Token].Offset)
  line, opening := 1, 0
  for index, symbol := range memory.Source[:offset] {
    if symbol == '\n' {
      line, opening = line+1, index+1
    }
  }
  file.Clause_Line, file.Clause_Column = Diagnostic_Line(line), Diagnostic_Column(offset-opening+1)
  table.Files[slot] = file
}

// Judges the component layout of each clean Go file the walk recorded, in slot order, then the
// func Main of each binary component the walk covers whole.
func layout_judge(memory Memory_Handle) {
  Memory_Handle_Invariants(memory, "layout_judge.memory")
  state := memory.Graph[GRAPH_STATE:]
  for slot, file := range memory.Siblings.Files {
    // Past the walked files every slot stands unrecorded, which the judgment passes by unread.
    if !bool(file.Recorded) {
      continue
    }
    state[GRAPH_LAYOUT_SLOT] = Sibling_Offset(slot)
    if !bool(layout_open(memory)) {
      continue
    }
    layout_file(memory)
  }
  layout_entry(memory)
}

// Reports whether the Go file GRAPH_LAYOUT_SLOT names stands recorded in a component, clean so
// far, and loads its directory into the subject region and its component into the top region.
func layout_open(memory Memory_Handle) (open Boolean) {
  defer func() { Boolean_Invariants(open, "layout_open.open") }()
  Memory_Handle_Invariants(memory, "layout_open.memory")
  slot := memory.Graph[GRAPH_STATE:][GRAPH_LAYOUT_SLOT]
  file := memory.Siblings.Files[slot]
  if !bool(file.Recorded) {
    return false
  }
  if file.Top_End == file.Path_Start {
    return false
  }
  if memory.Diagnostics[slot].Tier != TIER_NONE {
    return false
  }
  path := memory.Paths[slot]
  directory := max(int(bytes.Last_Index_Byte(bytes.Slice(path), PATH_SEPARATOR)), 0)
  glob := memory.Glob[GLOB_PATTERN_SIZE:GLOB_STORAGE_SIZE]
  glob[copy(glob, path[file.Path_Start:directory])] = 0
  layout_subject(memory)
  return true
}

// Judges the Go file GRAPH_LAYOUT_SLOT names, rule by rule, until one states a finding: where it
// stands, how deep, what it imports, and what it calls.
func layout_file(memory Memory_Handle) {
  Memory_Handle_Invariants(memory, "layout_file.memory")
  state := memory.Graph[GRAPH_STATE:]
  if bool(layout_shared_place(memory)) {
    return
  }
  if bool(layout_binary_place(memory)) {
    return
  }
  if bool(layout_default_place(memory)) {
    return
  }
  // The ancestors of the file's own directory decide its tier, which the imports clobber.
  state[GRAPH_LAYOUT_ANCESTRY] = Sibling_Offset(layout_ancestry(memory))
  if bool(layout_depth(memory)) {
    return
  }
  if bool(layout_imports(memory)) {
    return
  }
  layout_calls(memory)
}

// States the finding the notice store of the run holds at the Go file GRAPH_LAYOUT_SLOT names, at
// the first column of its first line, cut at the store.
func layout_state(memory Memory_Handle) {
  Memory_Handle_Invariants(memory, "layout_state.memory")
  slot := memory.Graph[GRAPH_STATE:][GRAPH_LAYOUT_SLOT]
  memory.Diagnostics[slot] = Diagnostic{Tier: TIER_TWO, Rule: RULE_LAYOUT, Line: 1, Column: 1}
  copy(memory.Notices[slot], memory.Notice)
}

// States a shared component file standing under an internal directory, or in package main.
func layout_shared_place(memory Memory_Handle) (stated Boolean) {
  defer func() { Boolean_Invariants(stated, "layout_shared_place.stated") }()
  Memory_Handle_Invariants(memory, "layout_shared_place.memory")
  if !bool(layout_shared(memory)) {
    return false
  }
  table, notice := memory.Siblings, memory.Notice
  slot := memory.Graph[GRAPH_STATE:][GRAPH_LAYOUT_SLOT]
  file := table.Files[slot]
  path := memory.Paths[slot][file.Path_Start:]
  directory := path[:max(int(bytes.Last_Index_Byte(bytes.Slice(path), PATH_SEPARATOR)), 0)]
  written := 0
  opening := 0
  for end := range len(directory) + 1 {
    if end < len(directory) {
      if directory[end] != PATH_SEPARATOR {
        continue
      }
    }
    if string(directory[opening:end]) == "internal" {
      written = copy(notice, "A shared library permits no internal/ directory. Remove \"")
      written += copy(notice[written:], directory[:end])
      written += copy(notice[written:], "\".")
      break
    }
    opening = end + 1
  }
  if written == 0 {
    if string(table.Text[file.Clause_Start:file.Clause_End]) != "main" {
      return false
    }
    written = copy(notice, "The shared library \"")
    written += copy(
      notice[written:],
      memory.Layout.Text[:memory.Layout.Findings[LAYOUT_HEADER].Text_End],
    )
    written += copy(notice[written:], "/")
    written += copy(notice[written:], path[:file.Top_End-file.Path_Start])
    tail := "\" permits no package main. Move the entry point to a binary component."
    written += copy(notice[written:], tail)
  }
  notice[min(written, len(notice)-1)] = 0
  layout_state(memory)
  return true
}

// States a binary component file whose package is no main and stands outside pkg, whose package
// main stands below the component root, or which stands in a default directory.
func layout_binary_place(memory Memory_Handle) (stated Boolean) {
  defer func() { Boolean_Invariants(stated, "layout_binary_place.stated") }()
  Memory_Handle_Invariants(memory, "layout_binary_place.memory")
  if bool(layout_shared(memory)) {
    return false
  }
  table, notice := memory.Siblings, memory.Notice
  slot := memory.Graph[GRAPH_STATE:][GRAPH_LAYOUT_SLOT]
  file := table.Files[slot]
  path := memory.Paths[slot][file.Path_Start:]
  top := path[:file.Top_End-file.Path_Start]
  directory := path[:max(int(bytes.Last_Index_Byte(bytes.Slice(path), PATH_SEPARATOR)), 0)]
  // A file at the component root stands in no directory of it, which the messages spell ".".
  relative := directory[len(directory):]
  if len(directory) > len(top) {
    relative = directory[len(top)+1:]
  }
  first := relative
  for index, symbol := range relative {
    if symbol == PATH_SEPARATOR {
      first = relative[:index]
      break
    }
  }
  written := 0
  main := string(table.Text[file.Clause_Start:file.Clause_End]) == "main"
  if !main {
    if string(first) != "pkg" {
      written = copy(notice, "Move ")
      written += copy(notice[written:], relative)
      if len(relative) == 0 {
        written += copy(notice[written:], ".")
      }
      written += copy(notice[written:], " -> ")
      written += copy(notice[written:], top)
      written += copy(notice[written:], "/pkg")
      if len(relative) > 0 {
        written += copy(notice[written:], "/")
        written += copy(notice[written:], relative)
      }
      written += copy(notice[written:], ".")
    }
  }
  if written == 0 {
    if main {
      if len(relative) > 0 {
        written = copy(notice, "The binary component \"")
        written += copy(
          notice[written:],
          memory.Layout.Text[:memory.Layout.Findings[LAYOUT_HEADER].Text_End],
        )
        written += copy(notice[written:], "/")
        written += copy(notice[written:], top)
        written += copy(notice[written:], "\" has its main package at \"")
        written += copy(notice[written:], relative)
        written += copy(notice[written:], "\". Move the main package to the component root.")
      }
    }
  }
  if written == 0 {
    if string(
      directory[max(int(bytes.Last_Index_Byte(bytes.Slice(directory), PATH_SEPARATOR))+1, 0):],
    ) ==
      "default" {
      written = copy(notice, "A binary component permits no default tier. ")
      written += copy(notice[written:], "Bind the real world in package main. Remove \"")
      written += copy(notice[written:], directory)
      written += copy(notice[written:], "\".")
    }
  }
  if written == 0 {
    return false
  }
  notice[min(written, len(notice)-1)] = 0
  layout_state(memory)
  return true
}

// States a Go file in a default directory whose package clause, its _test aside, names no package
// of its parent directory's name, at the clause.
func layout_default_place(memory Memory_Handle) (stated Boolean) {
  defer func() { Boolean_Invariants(stated, "layout_default_place.stated") }()
  Memory_Handle_Invariants(memory, "layout_default_place.memory")
  table, notice := memory.Siblings, memory.Notice
  slot := memory.Graph[GRAPH_STATE:][GRAPH_LAYOUT_SLOT]
  file := table.Files[slot]
  path := memory.Paths[slot][file.Path_Start:]
  directory := path[:max(int(bytes.Last_Index_Byte(bytes.Slice(path), PATH_SEPARATOR)), 0)]
  cut := int(bytes.Last_Index_Byte(bytes.Slice(directory), PATH_SEPARATOR))
  if string(directory[cut+1:]) != "default" {
    return false
  }
  if cut < 0 {
    return false
  }
  above := directory[:cut]
  parent := above[int(bytes.Last_Index_Byte(bytes.Slice(above), PATH_SEPARATOR))+1:]
  clause := table.Text[file.Clause_Start:file.Clause_End]
  base := clause
  if len(base) > len("_test") {
    if string(base[len(base)-len("_test"):]) == "_test" {
      base = base[:len(base)-len("_test")]
    }
  }
  if string(base) == string(parent) {
    return false
  }
  written := copy(notice, "The default package declares \"package ")
  written += copy(notice[written:], clause)
  written += copy(notice[written:], "\", not \"package ")
  written += copy(notice[written:], parent)
  written += copy(notice[written:], "\". The default package shadows the library it re-exports.")
  notice[min(written, len(notice)-1)] = 0
  layout_state(memory)
  memory.Diagnostics[slot].Line, memory.Diagnostics[slot].Column = file.Clause_Line, file.Clause_Column
  return true
}

// States a Go file, no test and no package main, whose package stands under more than one library
// ancestor, and names them nearest first.
func layout_depth(memory Memory_Handle) (stated Boolean) {
  defer func() { Boolean_Invariants(stated, "layout_depth.stated") }()
  Memory_Handle_Invariants(memory, "layout_depth.memory")
  table, notice, text := memory.Siblings, memory.Notice, memory.Layout.Text
  state := memory.Graph[GRAPH_STATE:]
  file := table.Files[state[GRAPH_LAYOUT_SLOT]]
  clause := table.Text[file.Clause_Start:file.Clause_End]
  if bool(file.Test) {
    return false
  }
  if string(clause) == "main" {
    return false
  }
  if Ancestry(state[GRAPH_LAYOUT_ANCESTRY]) != ANCESTRY_SEVERAL {
    return false
  }
  names := text[LAYOUT_NAMES : LAYOUT_NAMES+2*LAYOUT_REGION_SIZE]
  names = names[:bytes.Index_Byte(bytes.Slice(names), 0)]
  count := 1
  for _, symbol := range names {
    if symbol == ' ' {
      count++
    }
  }
  written := copy(notice, "The package \"")
  written += copy(notice[written:], clause)
  written += copy(notice[written:], "\" at \"")
  written += copy(notice[written:], table.Text[file.Canonical_Start:file.Canonical_End])
  written += copy(notice[written:], "\" is below the library tier. It has ")
  digits := [strconv.DIGIT_TEXT_SIZE_MAXIMUM]byte{}
  number := strconv.Unsigned_Integer(count)
  size := strconv.Format_Unsigned_Integer_Into(
    strconv.Buffer(digits[:]),
    number,
    strconv.DECIMAL_BASE,
  )
  written += copy(notice[written:], digits[:size])
  written += copy(notice[written:], " non-main ancestors: [")
  written += copy(notice[written:], names)
  written += copy(notice[written:], "].")
  notice[min(written, len(notice)-1)] = 0
  layout_state(memory)
  return true
}

// Reports whether the Go file GRAPH_LAYOUT_SLOT names stands in an impure package: a package main,
// or in the shared component a default directory or the composition tier.
func layout_impure(memory Memory_Handle) (impure Boolean) {
  defer func() { Boolean_Invariants(impure, "layout_impure.impure") }()
  Memory_Handle_Invariants(memory, "layout_impure.memory")
  table := memory.Siblings
  state := memory.Graph[GRAPH_STATE:]
  file := table.Files[state[GRAPH_LAYOUT_SLOT]]
  clause := string(table.Text[file.Clause_Start:file.Clause_End])
  if clause == "main" {
    return true
  }
  if clause == "main_test" {
    return true
  }
  path := memory.Paths[state[GRAPH_LAYOUT_SLOT]][file.Path_Start:]
  top := memory.Layout.Text[LAYOUT_TOP : LAYOUT_TOP+LAYOUT_REGION_SIZE]
  top[copy(top[:LAYOUT_REGION_SIZE-1], path[:file.Top_End-file.Path_Start])] = 0
  if !bool(layout_shared(memory)) {
    return false
  }
  canonical := table.Text[file.Canonical_Start:file.Canonical_End]
  last := canonical[int(bytes.Last_Index_Byte(bytes.Slice(canonical), PATH_SEPARATOR))+1:]
  if string(last) == "default" {
    return true
  }
  return Boolean(Ancestry(state[GRAPH_LAYOUT_ANCESTRY]) == ANCESTRY_ONE)
}

// States the first import of the Go file GRAPH_LAYOUT_SLOT names that breaks the layout: a package
// of another binary component, an impure stdlib package from a pure file, or an impure first-party
// package from a pure package that no instrumentation package list holds.
func layout_imports(memory Memory_Handle) (stated Boolean) {
  defer func() { Boolean_Invariants(stated, "layout_imports.stated") }()
  Memory_Handle_Invariants(memory, "layout_imports.memory")
  table, notice, state := memory.Siblings, memory.Notice, memory.Graph[GRAPH_STATE:]
  file := table.Files[state[GRAPH_LAYOUT_SLOT]]
  path := memory.Paths[state[GRAPH_LAYOUT_SLOT]][file.Path_Start:]
  shared := layout_shared(memory)
  tainted := layout_impure(memory)
  state[GRAPH_LAYOUT_SHARED], state[GRAPH_LAYOUT_IMPURE] = 0, 0
  if bool(shared) {
    state[GRAPH_LAYOUT_SHARED] = 1
  }
  if bool(tainted) {
    state[GRAPH_LAYOUT_IMPURE] = 1
  }
  module := memory.Layout.Text[:memory.Layout.Findings[LAYOUT_HEADER].Text_End]
  composition := bool(shared)
  if composition {
    composition = Ancestry(state[GRAPH_LAYOUT_ANCESTRY]) == ANCESTRY_ONE
  }
  // The assertion library imports the stdlib its every port depends on.
  aver := len(path) > len(AVER_DIRECTORY)
  if aver {
    aver = string(path[:len(AVER_DIRECTORY)+1]) == AVER_DIRECTORY+"/"
  }
  pure := string(table.Text[file.Clause_Start:file.Clause_End]) != "main"
  if bool(file.Test) {
    pure = false
  }
  if composition {
    pure = false
  }
  for entry := file.End; entry < file.Imports_End; entry++ {
    imported := table.Text[table.Names[entry].Start:table.Names[entry].End]
    found := false
    switch string(imported) {
    case "os", "os/exec", "os/user", "os/signal", "flag", "runtime", "math/rand", "crypto/rand":
      if pure {
        found = !aver
      }
    }
    if found {
      written := copy(notice, "The stdlib import \"")
      written += copy(notice[written:], imported)
      written += copy(notice[written:], "\" is impure. See lint/README.md for the resolutions.")
      notice[min(written, len(notice)-1)] = 0
    }
    // A first-party import names a component past the module path.
    first_party := len(module) > 0
    if first_party {
      first_party = len(imported) > len(module)
    }
    if first_party {
      first_party = string(imported[:len(module)]) == string(module)
    }
    if first_party {
      first_party = imported[len(module)] == PATH_SEPARATOR
    }
    if !found {
      if first_party {
        state[GRAPH_LOOKUP] = entry
        found = bool(layout_first_party(memory))
      }
    }
    if found {
      layout_state(memory)
      memory.Diagnostics[state[GRAPH_LAYOUT_SLOT]].Line = table.Names[entry].Line
      memory.Diagnostics[state[GRAPH_LAYOUT_SLOT]].Column = table.Names[entry].Column
      return true
    }
  }
  return false
}

// States the import GRAPH_LOOKUP names, a first-party package, when it breaks the layout for the
// Go file GRAPH_LAYOUT_SLOT names: a binary component's file imports another binary component, or
// a pure package imports a default directory or the composition tier of the shared component that
// no instrumentation package list holds.
func layout_first_party(memory Memory_Handle) (stated Boolean) {
  defer func() { Boolean_Invariants(stated, "layout_first_party.stated") }()
  Memory_Handle_Invariants(memory, "layout_first_party.memory")
  table, notice, state := memory.Siblings, memory.Notice, memory.Graph[GRAPH_STATE:]
  file := table.Files[state[GRAPH_LAYOUT_SLOT]]
  path := memory.Paths[state[GRAPH_LAYOUT_SLOT]][file.Path_Start:]
  own := path[:file.Top_End-file.Path_Start]
  name := table.Names[state[GRAPH_LOOKUP]]
  imported := table.Text[name.Start:name.End]
  module := memory.Layout.Text[:memory.Layout.Findings[LAYOUT_HEADER].Text_End]
  rest := imported[len(module)+1:]
  glob := memory.Glob[GLOB_PATTERN_SIZE:GLOB_STORAGE_SIZE]
  glob[copy(glob[:len(glob)-1], rest)] = 0
  layout_subject(memory)
  other := memory.Layout.Text[LAYOUT_TOP : LAYOUT_TOP+LAYOUT_REGION_SIZE]
  other = other[:bytes.Index_Byte(bytes.Slice(other), 0)]
  shared := layout_shared(memory)
  written := 0
  if state[GRAPH_LAYOUT_SHARED] == 0 {
    if !bool(shared) {
      if string(other) != string(own) {
        written = copy(notice, "The binary component \"")
        written += copy(notice[written:], module)
        written += copy(notice[written:], "/")
        written += copy(notice[written:], own)
        written += copy(notice[written:], "\" imports the binary component \"")
        written += copy(notice[written:], module)
        written += copy(notice[written:], "/")
        written += copy(notice[written:], other)
        written += copy(notice[written:], "\". Move the shared code to the shared component.")
      }
    }
  }
  if written == 0 {
    if state[GRAPH_LAYOUT_IMPURE] == 1 {
      return false
    }
    if !bool(shared) {
      return false
    }
    subject := memory.Layout.Text[LAYOUT_SUBJECT : LAYOUT_SUBJECT+LAYOUT_REGION_SIZE]
    subject = subject[:bytes.Index_Byte(bytes.Slice(subject), 0)]
    last := subject[int(bytes.Last_Index_Byte(bytes.Slice(subject), PATH_SEPARATOR))+1:]
    tainted := string(last) == "default"
    if !tainted {
      tainted = layout_ancestry(memory) == ANCESTRY_ONE
    }
    if !tainted {
      return false
    }
    // Write-only instrumentation feeds no impurity into what a pure package evaluates.
    if bool(glob_listed(memory, GLOB_LIST_INSTRUMENTATION)) {
      return false
    }
    written = copy(notice, "The dependency \"")
    written += copy(notice[written:], imported)
    written += copy(notice[written:], "\" is impure. A pure package imports only pure packages.")
  }
  notice[min(written, len(notice)-1)] = 0
  return true
}

// States the first clock, stdout, or network selector of the Go file GRAPH_LAYOUT_SLOT names from
// a pure file, or else its first selector that reaches one from a pure package.
func layout_calls(memory Memory_Handle) {
  Memory_Handle_Invariants(memory, "layout_calls.memory")
  table, notice, state := memory.Siblings, memory.Notice, memory.Graph[GRAPH_STATE:]
  file := table.Files[state[GRAPH_LAYOUT_SLOT]]
  pure := string(table.Text[file.Clause_Start:file.Clause_End]) != "main"
  if bool(file.Test) {
    pure = false
  }
  if state[GRAPH_LAYOUT_SHARED] == 1 {
    if Ancestry(state[GRAPH_LAYOUT_ANCESTRY]) == ANCESTRY_ONE {
      pure = false
    }
  }
  site, opening, tail := Sibling_Offset(0), "", ""
  if file.Soft != 0 {
    if pure {
      site, opening = file.Soft, "The stdlib call "
      tail = " is impure. See lint/README.md for the resolutions."
    }
  }
  if site == 0 {
    if file.Transitive != 0 {
      if state[GRAPH_LAYOUT_IMPURE] == 0 {
        site, opening = file.Transitive, "The transitive call "
        tail = " is impure. A pure package calls only pure APIs."
      }
    }
  }
  if site == 0 {
    return
  }
  name := table.Names[site]
  written := copy(notice, opening)
  written += copy(notice[written:], table.Text[name.Start:name.End])
  written += copy(notice[written:], tail)
  notice[min(written, len(notice)-1)] = 0
  layout_state(memory)
  memory.Diagnostics[state[GRAPH_LAYOUT_SLOT]].Line = name.Line
  memory.Diagnostics[state[GRAPH_LAYOUT_SLOT]].Column = name.Column
}

// States each binary component the walk covers whole, from the root it opened on, that declares
// no func Main in its top-level pkg directory, or several, at the component.
func layout_entry(memory Memory_Handle) {
  Memory_Handle_Invariants(memory, "layout_entry.memory")
  table, text := memory.Siblings, memory.Layout.Text
  root := text[LAYOUT_ROOT : LAYOUT_ROOT+LAYOUT_REGION_SIZE]
  root = root[:bytes.Index_Byte(bytes.Slice(root), 0)]
  for len(root) > 1 {
    if string(root[:2]) != "./" {
      break
    }
    root = root[2:]
  }
  for slot, file := range table.Files {
    if !bool(file.Recorded) {
      continue
    }
    top := memory.Paths[slot][file.Path_Start:file.Top_End]
    if len(top) == 0 {
      continue
    }
    if string(root) != "." {
      if string(root) != string(top) {
        continue
      }
    }
    mains, first := 0, true
    for other, member := range table.Files {
      if !bool(member.Recorded) {
        continue
      }
      if string(memory.Paths[other][member.Path_Start:member.Top_End]) != string(top) {
        continue
      }
      if other < slot {
        first = false
      }
      if bool(member.Test) {
        continue
      }
      path := memory.Paths[other][member.Path_Start:]
      directory := path[:max(int(bytes.Last_Index_Byte(bytes.Slice(path), PATH_SEPARATOR)), 0)]
      if string(directory[min(len(top)+1, len(directory)):]) == "pkg" {
        mains += int(member.Mains)
      }
    }
    if !first {
      continue
    }
    text[LAYOUT_TOP+copy(text[LAYOUT_TOP:LAYOUT_TOP+LAYOUT_REGION_SIZE-1], top)] = 0
    if bool(layout_shared(memory)) {
      continue
    }
    if mains == 1 {
      continue
    }
    glob := memory.Glob[GLOB_PATTERN_SIZE:GLOB_STORAGE_SIZE]
    glob[copy(glob, top)] = 0
    quote := LAYOUT_QUOTE_MAIN_ABSENT
    if mains > 1 {
      quote = LAYOUT_QUOTE_MAIN_SEVERAL
    }
    layout_record(memory, LAYOUT_COMPONENT_OPENING, quote)
  }
}
