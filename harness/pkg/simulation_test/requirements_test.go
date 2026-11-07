// Package simulation_test is the harness blackbox witness: every run goes through pkg.Main on a
// seeded simulated loop, and this test root alone holds the Driver.
package simulation_test

import "errors"
import "fmt"
import "testing"
import "unsafe"

import "local/james-orcales/harness/pkg"
import "local/james-orcales/harness/pkg/configuration"
import "local/james-orcales/harness/pkg/requirements"
import "local/james-orcales/harness/pkg/vcs"
import "local/james-orcales/shared/std/filepath"
import "local/james-orcales/shared/std/go/ast"
import "local/james-orcales/shared/std/go/token"
import "local/james-orcales/shared/std/sim/nbio"
import "local/james-orcales/shared/std/sim/prng"
import "local/james-orcales/shared/std/sim/time"
import "local/james-orcales/shared/std/strings"

// Test_Simulation sweeps seeds over every outcome a run reaches: usage, a failed walk, a clean
// package, and a package holding each diagnostic.
func Test_Simulation(t *testing.T) {
  memory, format_state, command := harness.New_Memory()
  directory := harness.Directory{
    Current: make(harness.Current_Path, 0, filepath.PATH_SIZE_MAXIMUM),
  }
  for seed := uint64(1); seed < SEED_COUNT+1; seed++ {
    subject := fixture_new(t, memory, format_state, command, directory, seed)
    // A command line naming no directory is one shared/std/cli refuses, thus the run
    // states the refusal and exits failed rather than walks nothing.
    fixture_run(t, subject, []string{"harness"})
    fixture_refused(t, subject)
    subject = fixture_new(t, memory, format_state, command, directory, seed)
    // A directory that stands nowhere names nothing to walk, thus the run fails its
    // walk rather than states a package of no files.
    fixture_run(t, subject, []string{"harness", "absent"})
    fixture_failed(t, subject, "walk")
    fixture_clean(t, memory, format_state, command, directory, seed)
    subject = fixture_new(t, memory, format_state, command, directory, seed)
    fixture_directory(t, subject, "package")
    fixture_directory(t, subject, "package/nested")
    fixture_file(t, subject, "package/a.go", INTERFACE_DECLARATION)
    fixture_file(t, subject, "package/notes.txt", REFUSED)
    fixture_file(t, subject, "package/nested/b.go", UNFORMATTED)
    fixture_file(t, subject, "package/nested/c.go", REFUSED)
    // A name that is no Go file names nothing to check, thus the run passes over it
    // and states the Go files under the directory at every depth.
    fixture_run(t, subject, []string{"harness", "package"})
    fixture_states(t, subject, "package/nested/b.go: not formatted")
    fixture_silent(t, subject, "notes.txt")
    fixture_run(t, subject, []string{"harness", "package", "-json"})
    fixture_silent(t, subject, "notes.txt")
    fixture_run(t, subject, []string{"harness", "-h"})
    // The help states what the program is and how it is called, and a run that
    // wrote nothing or named a failure would pass a state that stands anywhere.
    fixture_opened(t, subject, "harness report each Go file that breaks a rule\n")
    if subject.Runner.Status != harness.EXIT_SUCCESS {
      t.Fatalf("a help run exited %d", subject.Runner.Status)
    }
    fixture_run(t, subject, []string{"harness", "package", "-fix"})
    subject = fixture_new(t, memory, format_state, command, directory, seed)
    fixture_directory(t, subject, "contract")
    fixture_directory(t, subject, "contract/main")
    fixture_file(t, subject, "contract/a.go", CLEAN)
    fixture_file(t, subject, "contract/REQUIREMENTS.md", FAULTY_CONTRACT)
    fixture_file(t, subject, "contract/requirements_test.go", FAULTY_TEST)
    fixture_file(
      t, subject, "contract/main/main.go", "package main\n\nfunc main() {}\n",
    )
    fixture_run(t, subject, []string{"harness", "contract"})
    fixture_run(t, subject, []string{"harness", "contract", "-json"})
    subject = fixture_new(t, memory, format_state, command, directory, seed)
    fixture_directory(t, subject, "contract")
    fixture_file(t, subject, "contract/a.go", CLEAN)
    fixture_file(t, subject, "contract/REQUIREMENTS.md", "\n# Contract\n\nIt holds.\n")
    fixture_run(t, subject, []string{"harness", "contract"})
    fixture_deterministic(t, memory, format_state, command, directory, seed)
    fixture_generated(t, memory, format_state, command, directory, seed)
    fixture_shaped(t, memory, format_state, command, directory, seed)
  }
  fixture_once(t, memory, format_state, command, directory)
}

// Every fixture that one seed states as well as many: a tree of one package, or one the shape of
// which no order of retirements can change.
func fixture_once(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
) {
  t.Helper()
  fixture_sizes(t, memory, format_state, command, directory, 1)
  fixture_widths(t, memory, format_state, command, directory, 1)
  fixture_edges(t, memory, format_state, command, directory, 1)
  fixture_counts(t, memory, format_state, command, directory, 1)
  fixture_faults(t, memory, format_state, command, directory, 1)
  fixture_positions(t, memory, format_state, command, directory, 1)
  fixture_idempotent(t, memory, format_state, command, directory, 1)
  fixture_modes(t, memory, format_state, command, directory, 1)
  fixture_settled(t, memory, format_state, command, directory, 1)
  fixture_exempt(t, memory, format_state, command, directory, 1)
  fixture_untrusted(t, memory, format_state, command, directory, 1)
  fixture_order(t, memory, format_state, command, directory, 1)
  fixture_test_refused(t, memory, format_state, command, directory, 1)
  fixture_literal(t, memory, format_state, command, directory, 1)
  fixture_renamed(t, memory, format_state, command, directory, 1)
  fixture_uptime(t, memory, format_state, command, directory, 1)
  fixture_least(t, memory, format_state, command, directory, 1)
  fixture_linked(t, memory, format_state, command, directory, 1)
  fixture_orphans(t, memory, format_state, command, directory, 1)
  fixture_conflicted(t, memory, format_state, command, directory, 1)
  fixture_configured(t, memory, format_state, command, directory, 1)
  fixture_surveyed(t, memory, format_state, command, directory, 1)
  fixture_imported(t, memory, format_state, command, directory, 1)
  fixture_names(t, memory, format_state, command, directory, 1)
  fixture_shadowed(t, memory, format_state, command, directory, 1)
  fixture_recursed(t, memory, format_state, command, directory, 1)
  fixture_laid(t, memory, format_state, command, directory, 1)
  fixture_raced(t, memory, format_state, command, directory, 1)
  fixture_shape(t, memory, format_state, command, directory, 1)
  fixture_lines_json(t, memory, format_state, command, directory, 1)
  fixture_kept(t, memory, format_state, command, directory, 1)
  fixture_barred(t, memory, format_state, command, directory, 1)
  fixture_partial(t, memory, format_state, command, directory, 1)
  fixture_named(t, memory, format_state, command, directory, 1)
  fixture_escaped(t, memory, format_state, command, directory, 1)
  fixture_twice(t, memory, format_state, command, directory, 1)
  fixture_after(t, memory, format_state, command, directory, 1)
  fixture_pair(t, memory, format_state, command, directory, 1)
  fixture_deep(t, memory, format_state, command, directory, 1)
  fixture_external(t, memory, format_state, command, directory, 1)
  fixture_kin(t, memory, format_state, command, directory, 1)
  fixture_absent(t, memory, format_state, command, directory, 1)
  fixture_ended(t, memory, format_state, command, directory, 1)
  fixture_table(t, memory, format_state, command, directory, 1)
  fixture_first(t, memory, format_state, command, directory, 1)
  fixture_empty(t, memory, format_state, command, directory, 1)
  fixture_bytes(t, memory, format_state, command, directory, 1)
  fixture_fixed_json(t, memory, format_state, command, directory, 1)
  fixture_line(t, memory, format_state, command, directory, 1)
  fixture_outgrown(t, memory, format_state, command, directory, 1)
  fixture_folded(t, memory, format_state, command, directory, 1)
  fixture_unproven(t, memory, format_state, command, directory, 1)
  fixture_forgotten(t, memory, format_state, command, directory, 1)
  fixture_third(t, memory, format_state, command, directory, 1)
  fixture_shuffled(t, memory, format_state, command, directory, 1)
  fixture_lined(t, memory, format_state, command, directory, 1)
  fixture_unprintable(t, memory, format_state, command, directory, 1)
  fixture_filled(t, memory, format_state, command, directory, 1)
  fixture_exact(t, memory, format_state, command, directory, 1)
  fixture_widest_name(t, memory, format_state, command, directory, 1)
  fixture_test_lines(t, memory, format_state, command, directory, 1)
  fixture_full_line(t, memory, format_state, command, directory, 1)
  fixture_wide_command(t, memory, format_state, command, directory, 1)
  fixture_stale(t, memory, format_state, command, directory, 1)
  fixture_test_tail(t, memory, format_state, command, directory, 1)
  fixture_rewritten(t, memory, format_state, command, directory, 1)
  fixture_history(t, memory, format_state, command, directory, 1)
  fixture_shut(t, memory, format_state, command, directory, 1)
  fixture_spelled(t, memory, format_state, command, directory, 1)
  fixture_declared(t, memory, format_state, command, directory, 1)
}

// TABLE_NODE_COUNT holds a full package table of one directory and one Go file each.
const TABLE_NODE_COUNT = 2*harness.PACKAGE_COUNT_MAXIMUM + TALL_NODE_COUNT

// One package per slot of the package table, each owing a contract it holds none of, and one Go
// file per slot of the file table: the one tree witnesses both bounds. The parser refuses each
// file but the last, which costs no format, and the last stands in canonical form at the last
// slot, where its banned import states a finding.
func fixture_table(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  // One package per slot: each directory holds one Go file, so the package table fills.
  subject := fixture_sized(
    t,
    memory,
    format_state,
    command,
    directory,
    seed,
    TABLE_NODE_COUNT,
    NODE_CONTENTS_BYTES,
  )
  subject.Output = nbio.Stream_Memory{Memory: make([]byte, TABLE_OUTPUT_BYTES)}
  subject.Runner.Output = nbio.Memory_To_Stream(&subject.Output)
  fixture_directory(t, subject, "full")
  // A directory the walk holds pending while it reads the widest one: the pass states its
  // children beside that one, thus the directory table stands at its whole length where a
  // wide directory alone leaves one slot of it free.
  fixture_directory(t, subject, "full/kept")
  // A name the walk passes over, in the one directory the walk reads last: the pass states
  // it where every package the run will record is already recorded, thus the collect
  // stands at a full package table rather than at one slot short of it.
  fixture_file(t, subject, "full/kept/notes.txt", CLEAN)
  fixture_directory(t, subject, "full/many")
  fixture_file(t, subject, "full/many/a.go", REFUSED)
  for index := range harness.PACKAGE_COUNT_MAXIMUM - 1 {
    name := "full/many/" + fixture_name(index)
    fixture_directory(t, subject, name)
    source := REFUSED
    if index == harness.PACKAGE_COUNT_MAXIMUM-2 {
      source = "package a\n\nimport \"os\"\n"
    }
    fixture_file(t, subject, name+"/a.go", source)
  }
  // A branch off a clone that holds no main ref: git answers while the lanes work, and the run
  // states the missing ref past every package, where the package cursor stands at the table's
  // end.
  subject.Loop.Spawn = history_spawn(History_Script{Head: "feature\n", Refs: ""})
  fixture_run(t, subject, []string{"harness", "full"})
  // Each Go file is refused or bans an import and each package owes a contract it holds none
  // of, thus the run states one line per slot of both tables rather than stops at the first,
  // one more for the impurity of the one import, and one line for the history.
  packages := fixture_lines(fixture_written(subject))
  if packages != harness.FILE_COUNT_MAXIMUM+harness.PACKAGE_COUNT_MAXIMUM+2 {
    t.Fatalf(
      "a full package table stated %d findings, failing %q",
      packages,
      fixture_errors(subject),
    )
  }
  fixture_run(t, subject, []string{"harness", ""})
  fixture_failed(t, subject, "walk")
}

// The least an input holds past none of it: a refusal one step into a source, and a contract of
// one byte and of two.
func fixture_least(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  // One source of one byte, then a second run over it, and the same for one of two bytes: a
  // run opens on what the run before it left, thus the stores it states carry that source
  // rather than none.
  for _, source := range []string{"p", "pa"} {
    subject := fixture_new(t, memory, format_state, command, directory, seed)
    fixture_directory(t, subject, "small")
    fixture_file(t, subject, "small/a.go", source)
    fixture_run(t, subject, []string{"harness", "small"})
    fixture_run(t, subject, []string{"harness", "small"})
  }
  // A package of two Go files, then a second run over it: the counts a run opens on stand at
  // two, which one file leaves at one and a wider package leaves far past.
  subject := fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "pair")
  fixture_file(t, subject, "pair/a.go", MAIN)
  fixture_file(t, subject, "pair/b.go", CLEAN)
  fixture_run(t, subject, []string{"harness", "pair"})
  fixture_run(t, subject, []string{"harness", "pair"})
  // A refusal on the second line at the second column: a position states both its numbers
  // one step in, which a refusal at a first line or a first column leaves at one.
  subject = fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "small")
  fixture_file(t, subject, "small/a.go", "package a\n &")
  fixture_run(t, subject, []string{"harness", "small", "-json"})
  fixture_states(t, subject, `"line":2,"column":2`)
  fixture_run(t, subject, []string{"harness", "small", "-json"})
  // The same refusal as text: the line it states carries both numbers, which the json
  // object carries as leaves instead.
  fixture_run(t, subject, []string{"harness", "small"})
  fixture_states(t, subject, "a.go:2:2:")
  // A package of two diagnostics that owes a contract: the contract read opens on a tally
  // of two, which one diagnostic leaves at one and a package owing none never states.
  subject = fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "two")
  fixture_file(t, subject, "two/a.go", UNFORMATTED)
  fixture_file(t, subject, "two/b.go", UNFORMATTED)
  fixture_file(t, subject, "two/"+requirements.MARKDOWN_NAME, CONTRACT_MARKDOWN)
  fixture_file(t, subject, "two/"+requirements.TEST_NAME, CONTRACT_TEST)
  fixture_run(t, subject, []string{"harness", "two"})
  // The same package as json: a tier one diagnostic writes its own report, and the contract
  // read that follows opens on that report rather than on one of a tier two finding.
  fixture_run(t, subject, []string{"harness", "two", "-json"})
  // A contract of one byte, and one of two: the check reads a markdown store holding the
  // least an open file can hold past none of it.
  for _, markdown := range []string{"#", "#\n"} {
    subject = fixture_new(t, memory, format_state, command, directory, seed)
    fixture_directory(t, subject, "small")
    fixture_file(t, subject, "small/a.go", CLEAN)
    fixture_file(t, subject, "small/"+requirements.MARKDOWN_NAME, markdown)
    fixture_run(t, subject, []string{"harness", "small", "-json"})
  }
  // A test file of one byte, and one of two: the check reads a test store holding the least
  // an open file can hold past none of it.
  for _, test := range []string{"p", "pa"} {
    subject = fixture_new(t, memory, format_state, command, directory, seed)
    fixture_directory(t, subject, "small")
    fixture_file(t, subject, "small/a.go", CLEAN)
    fixture_file(t, subject, "small/"+requirements.MARKDOWN_NAME, CONTRACT_MARKDOWN)
    fixture_file(t, subject, "small/"+requirements.TEST_NAME, test)
    fixture_run(t, subject, []string{"harness", "small", "-json"})
  }
}

// SHORT_MOMENT_COUNT is how many first readings a sweep states past none at all.
const SHORT_MOMENT_COUNT = 2

// UPTIME_SECONDS_TEXT is the seconds of a run that opens at boot and stands at the last moment a
// monotonic reading holds, which no run under the simulation's own clock spends.
const UPTIME_SECONDS_TEXT = "31536000.000"

// Uptime_clock states boot at its first reading and the last moment a host holds at every later
// one. A monotonic reader never goes backwards, thus the span stands ahead of the start.
type uptime_clock struct {
  // First is the moment the first reading states, which the run took as its start.
  First time.Monotonic_Moment
  // Later is the moment every reading past the first states.
  Later time.Monotonic_Moment
  // Started is whether the first reading stands, which the run took as its start.
  Started bool
}

// States boot once, then the moment the wrapper carries.
func uptime_now(state time.State) (moment time.Monotonic_Moment) {
  clock := state.(*uptime_clock)
  if !clock.Started {
    clock.Started = true
    return clock.First
  }
  return clock.Later
}

// A run that opens at boot and stands at the last moment a host holds: its elapsed span is the
// whole one a monotonic reading states, which no run under the simulation's own clock reaches.
func fixture_uptime(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  subject := fixture_uptime_twice(
    t,
    memory,
    format_state,
    command,
    directory,
    seed,
    []string{"harness", "up"},
    "elapsed_sec="+UPTIME_SECONDS_TEXT,
  )
  // A run the command line refuses opens on what the run before it left, at no cost of
  // walking that tree again.
  fixture_run(t, subject, []string{"harness", ""})
  fixture_failed(t, subject, "walk")
  // The same run as json: its summary carries those seconds as a leaf, which the text line
  // states outside any object. It stands on its own clock, since a monotonic reader never
  // goes backwards and a second run under the first clock would start past its own jump.
  subject = fixture_uptime_twice(
    t,
    memory,
    format_state,
    command,
    directory,
    seed,
    []string{"harness", "up", "-json"},
    `"elapsed_sec":"`+UPTIME_SECONDS_TEXT+`"`,
  )
  // A run the command line refuses opens on the summary the run before it wrote, thus the
  // seconds that summary names stand at the whole width a text holds.
  fixture_run(t, subject, []string{"harness", ""})
  fixture_failed(t, subject, "walk")
  // A run that found something closes on the same seconds, in the widest summary it writes.
  subject = fixture_uptime_new(
    t,
    memory,
    format_state,
    command,
    directory,
    seed,
    time.MONOTONIC_MOMENT_MINIMUM,
    time.MONOTONIC_MOMENT_MAXIMUM,
  )
  fixture_file(t, subject, "up/a.go", UNFORMATTED)
  fixture_run(t, subject, []string{"harness", "up", "-json"})
  fixture_states(t, subject, `{"passed":false,"elapsed_sec":"`+UPTIME_SECONDS_TEXT+`"}`)
  fixture_spans(t, memory, format_state, command, directory, seed)
}

// The shortest spans a clock states and the least moments a run opens at, and a fault at the open
// a fix makes, which fails the fix step rather than the open step.
func fixture_spans(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  // A run that stands one grain past boot, and one that stands two: the shortest spans a
  // clock states, which the seconds of either spell as no seconds at all.
  // A run whose clock opens at the first grain of a host's uptime, and one that opens at
  // the second: the moment a run states as its start stands past none of them.
  for _, first := range [SHORT_MOMENT_COUNT]time.Monotonic_Moment{1, 2} {
    subject := fixture_uptime_new(
      t,
      memory,
      format_state,
      command,
      directory,
      seed,
      first,
      time.MONOTONIC_MOMENT_MAXIMUM,
    )
    fixture_run(t, subject, []string{"harness", "up"})
    fixture_run(t, subject, []string{"harness", ""})
    fixture_failed(t, subject, "walk")
  }
  // The open a fix makes stands past the one the declaration pass makes, thus a fault there
  // fails the fix step rather than the open step.
  subject := fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "fault")
  fixture_file(t, subject, "fault/a.go", UNFORMATTED)
  opens := Fault_Storage{
    Inner: subject.Loop.Storage,
    Loop:  subject.Loop,
    Kind:  FAULT_OPEN,
    Skip:  FIX_OPEN_SKIP,
  }
  fault_wire(subject, &opens)
  fixture_run(t, subject, []string{"harness", "fault", "-fix"})
  fixture_failed(t, subject, "fix")
  for _, later := range []time.Monotonic_Moment{1, 2} {
    subject = fixture_uptime_new(
      t,
      memory,
      format_state,
      command,
      directory,
      seed,
      time.MONOTONIC_MOMENT_MINIMUM,
      later,
    )
    fixture_run(t, subject, []string{"harness", "up"})
    fixture_states(t, subject, "elapsed_sec=0.000")
  }
}

// Builds a package of one main file under a clock that jumps a host's uptime.
// Two runs of one form under one jumped clock: the first states the widest seconds a text
// holds, and the second opens on the summary it wrote, thus every step that takes that summary
// states seconds of that width rather than the width one run under one clock reaches.
func fixture_uptime_twice(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
  arguments []string,
  held string,
) (subject *fixture) {
  t.Helper()
  subject = fixture_uptime_new(
    t,
    memory,
    format_state,
    command,
    directory,
    seed,
    time.MONOTONIC_MOMENT_MINIMUM,
    time.MONOTONIC_MOMENT_MAXIMUM,
  )
  fixture_run(t, subject, arguments)
  fixture_states(t, subject, held)
  fixture_run(t, subject, arguments)
  fixture_passed(t, subject)
  return subject
}

func fixture_uptime_new(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
  first time.Monotonic_Moment,
  later time.Monotonic_Moment,
) (subject *fixture) {
  t.Helper()
  subject = fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "up")
  fixture_file(t, subject, "up/m.go", MAIN)
  wrapper := uptime_clock{First: first, Later: later}
  subject.Runner.Clock = time.Clock{
    State:         &wrapper,
    Now_Monotonic: uptime_now,
    Now_Realtime:  subject.Runner.Clock.Now_Realtime,
  }
  return subject
}

// One source of one byte, one of two, and one that refuses at the first column of the first line:
// a count and a position each state their smallest values, which no larger fixture reaches. These
// stand outside the seed sweep: each holds one file, whose retirements the seed cannot reorder,
// and the sweep over them states the same set at fifteen times the cost.
func fixture_sizes(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  subject := fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "small")
  fixture_file(t, subject, "small/a.go", "&")
  // The package owes no contract, thus the refusal is the last report the run writes and
  // the run after it opens on the position that refusal named.
  fixture_file(t, subject, "small/m.go", MAIN)
  fixture_run(t, subject, []string{"harness", "small", "-json"})
  fixture_states(t, subject, `"line":1,"column":1`)
  fixture_run(t, subject, []string{"harness", ""})
  fixture_failed(t, subject, "walk")
  subject = fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "small")
  fixture_file(t, subject, "small/a.go", "p&")
  fixture_run(t, subject, []string{"harness", "small", "-json"})
  subject = fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "small")
  fixture_file(t, subject, "small/a.go", "package a\n\n&\n")
  fixture_run(t, subject, []string{"harness", "small", "-json"})
  subject = fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "small")
  fixture_file(t, subject, "small/a.go", "package a\n\n &\n")
  fixture_run(t, subject, []string{"harness", "small", "-json"})
  subject = fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "small")
  fixture_file(t, subject, "small/a.go", "package a\n&\n")
  fixture_run(t, subject, []string{"harness", "small", "-json"})
  // A refusal on the second token, then on the third: a parse states the first slots of its
  // token run, which a refusal further in passes over.
  subject = fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "small")
  fixture_file(t, subject, "small/a.go", "package &")
  fixture_run(t, subject, []string{"harness", "small", "-json"})
  subject = fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "small")
  fixture_file(t, subject, "small/a.go", "package a &")
  fixture_run(t, subject, []string{"harness", "small", "-json"})
  // A file whose whole name is the suffix, and a directory whose name carries it: the walk
  // names one a Go file and the other a directory, since a name states neither.
  subject = fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "small")
  fixture_file(t, subject, "small/.go", CLEAN)
  fixture_directory(t, subject, "small/b.go")
  fixture_file(t, subject, "small/b.go/c.go", UNFORMATTED)
  fixture_run(t, subject, []string{"harness", "small", "-json"})
  fixture_reports(t, memory, format_state, command, directory, seed)
  // An empty Go file and an empty contract: a read that moves no byte ends at once, and the
  // check states what an empty source and an empty contract hold.
  subject = fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "small")
  fixture_file(t, subject, "small/a.go", "")
  fixture_run(t, subject, []string{"harness", "small", "-json"})
  subject = fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "small")
  fixture_file(t, subject, "small/a.go", CLEAN)
  fixture_file(t, subject, "small/"+requirements.MARKDOWN_NAME, "")
  fixture_run(t, subject, []string{"harness", "small", "-json"})
  // Two lines exactly: the source ends on its refusal, so the index holds no third start.
  subject = fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "small")
  fixture_file(t, subject, "small/a.go", "package a\n&")
  fixture_run(t, subject, []string{"harness", "small", "-json"})
}

// One package of five reports, each at a different position: a report states the last one
// written, thus the run states the first line, the second, and both columns.
func fixture_reports(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  subject := fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "small")
  fixture_file(t, subject, "small/a.go", "&")
  fixture_file(t, subject, "small/aa.go", UNFORMATTED)
  fixture_file(t, subject, "small/b.go", "package a\n&")
  fixture_file(t, subject, "small/c.go", "package a\n &")
  fixture_file(t, subject, "small/d.go", "&")
  fixture_run(t, subject, []string{"harness", "small", "-json"})
}

// INTERFACE_DECLARATION declares an interface holding a method, which the parser refuses, and
// stands first in its package so the files after it state the refusal it left behind.
const INTERFACE_DECLARATION = "package a\n\ntype T interface {\n\tM()\n}\n"

// FAULTY_CONTRACT breaks every format rule once: a preamble, a wrong level, a heading without
// fencing, a bad character, a repeat, an orphan, an empty leaf, a gap, and a long section.
const FAULTY_CONTRACT = "Intro.\n\n### Orphan\n\nBody.\n\n# A-B\n\n## Two\n\n# A-B\nBody.\n\n" +
  "# Empty\n\n# Tall\n\n1\n\n2\n3\n4\n\n# Two words\n\n### x\n\nBody.\n"

// FAULTY_TEST puts a declaration in the first slot and misses the rest.
const FAULTY_TEST = "package a\n\nconst X = 1\n\nfunc Test_Empty() {}\n"

// SEED_COUNT keeps the sweep wide enough to reorder retirements and fast: the suite runs in about
// half the ten seconds a package is allowed, thus a machine slower than this one still holds. The
// sweep holds the
// trees of several files, where one seed orders retirements differently from the next; a tree of
// one file states the same run at every seed, thus those fixtures stand outside it.
const SEED_COUNT = 24

// NAME_WIDTH is how many bytes one generated name spends, which holds one slot per file.
const NAME_WIDTH = 4

// FAULT_KIND_COUNT is how many operations one sweep fails, one run each.
const FAULT_KIND_COUNT = 4

// FAULT_OPEN fails the next open of a file the run reads.
const FAULT_OPEN Fault_Kind = 1

// FAULT_READ fails the next read.
const FAULT_READ Fault_Kind = 2

// FAULT_WRITE fails the next write.
const FAULT_WRITE Fault_Kind = 3

// FAULT_DIRECTORY fails the next open of a directory the walk stands on.
const FAULT_DIRECTORY Fault_Kind = 5

// FAULT_SHORT cuts the next write down to a few bytes, the way a disk that takes part of one
// does, thus the run writes the rest in a second pass.
const FAULT_SHORT Fault_Kind = 6

// FAULT_SHORT_READ cuts the next read down to a few bytes, the way a disk that hands back part
// of one does, thus the run reads the rest in a later pass.
const FAULT_SHORT_READ Fault_Kind = 7

// FAULT_EMPTY cuts the next write down to no byte at all, the way a disk with no room left
// takes none of one.
const FAULT_EMPTY Fault_Kind = 11

// FAULT_REVERSED hands the walk each directory pass in the reverse of the order the host states,
// the way a host that keeps its entries by hash states them, thus the run stands on the order it
// sorted rather than the order it read.
const FAULT_REVERSED Fault_Kind = 13

// FAULT_CLOSE states the close of one file as failed, the way a host that writes its blocks back
// at the close answers for a disk that took none of them. The simulator releases every descriptor
// it holds, thus this names the answer a host gives rather than a descriptor left open.
const FAULT_CLOSE Fault_Kind = 14

// Shut_File is the error a host states for a close it could not finish.
var Shut_File = errors.New("harness: the file would not close")

// FAULT_STATUS answers every ask of what a path holds with an error, the way a host answers for
// a name whose file stood when the pass named it and stands no longer.
const FAULT_STATUS Fault_Kind = 10

// Absent_Path is the error a host states for a path that stands nowhere.
var Absent_Path = errors.New("harness: the path stands nowhere")

// FAULT_LINK_CONTRACT states the contract files of a run as symbolic links, the way a host that
// keeps one contract for many packages writes them.
const FAULT_LINK_CONTRACT Fault_Kind = 12

// FAULT_SYMBOLIC_LINK states every path of a run as a symbolic link, the way a tree a host
// filled with links states them.
const FAULT_SYMBOLIC_LINK Fault_Kind = 9

// FAULT_SHORT_FAIL cuts the next write down to a few bytes and fails every write after it, the
// way a disk that takes part of one write and then goes away does.
const FAULT_SHORT_FAIL Fault_Kind = 8

// SHORT_WRITE_COUNT is how many such cuts one sweep makes, at one byte and then at two.
const SHORT_WRITE_COUNT = 2

// FAULT_OFFSET sits past any file the simulator holds, thus a write there fails the way a full
// disk does. A negative offset would not: that is FILE_POSITION_CURRENT, an ordinary append.
const FAULT_OFFSET int64 = 1 << 40

// FIX_OPEN_SKIP is how many opens of one Go file stand before the one a fix writes: the
// declaration pass reads the file once, and the fix writes the form that read left.
const FIX_OPEN_SKIP = 1

// FIX_WRITE_SKIP is how many fix writes stand before the one a fault fails: the first file's fix
// lands, and the second's fails.
const FIX_WRITE_SKIP = 1

// FAULT_SKIP_COUNT is how many reads of one package a sweep fails in turn: the declaration read,
// the check read, the contract markdown, and the contract test file.
const FAULT_SKIP_COUNT = 4

// FAULT_ABSENT_PATH names no file, so an open of it fails the way a vanished file does.
const FAULT_ABSENT_PATH = "fault/absent.go"

// Fault_Kind names which operation one run fails.
type Fault_Kind int

// Fault_Storage is one storage surface that fails a chosen operation once. It bends the argument
// the simulator checks rather than forging a completion, so the failure travels the simulator's
// own path and the loop keeps its discipline.
type Fault_Storage struct {
  // Inner is the surface every operation reaches once this one has had its turn.
  Inner nbio.Storage
  // Loop is the whole surface, since a close and a deinit read the storage half's state.
  Loop nbio.IO
  // Kind is the operation this surface fails.
  Kind Fault_Kind
  // Spent records that the fault already happened, thus a run meets it once.
  Spent bool
  // Skip is how many matching operations stand before the one that fails, so a fault can
  // name a later read than the first.
  Skip int
  // Short is how many bytes a cut write takes.
  Short int
  // Configuration is the descriptor harness.json or go.mod stands open on, which no fault
  // reaches: the fault names a step of the check, and both stand before every one of them.
  Configuration nbio.File
}

// Wraps one storage surface so the chosen operation fails once.
func fault_storage(state *Fault_Storage) (storage nbio.Storage) {
  return nbio.Storage{
    State:                           state,
    Read_Procedure:                  fault_read,
    Write_Procedure:                 fault_write,
    Open_At_Procedure:               fault_open_at,
    Fsync_Procedure:                 fault_fsync,
    Mkdir_At_Procedure:              fault_mkdir_at,
    Get_Directory_Entries_Procedure: fault_directory_entries,
    Status_Procedure:                fault_status,
    Read_Link_Procedure:             fault_read_link,
  }
}

// Reports whether this operation is the one the run fails, and spends the fault when it is.
func fault_due(state *Fault_Storage, kind Fault_Kind) (due bool) {
  if state.Spent {
    return false
  }
  if state.Kind != kind {
    return false
  }
  if state.Skip > 0 {
    state.Skip--
    return false
  }
  state.Spent = true
  return true
}

func fault_read(
  state nbio.State,
  completion *nbio.Completion,
  file nbio.File,
  buffer []byte,
  offset int64,
  timeout time.Duration,
  callback nbio.Callback,
) {
  faults := state.(*Fault_Storage)
  if file == faults.Configuration {
    nbio.Storage_Read(faults.Inner, completion, file, buffer, offset, timeout, callback)
    return
  }
  if fault_due(faults, FAULT_READ) {
    // A read past the end reports end of file, thus the offset states what no file
    // holds.
    offset = -1
  }
  nbio.Storage_Read(faults.Inner, completion, file, buffer, offset, timeout, callback)
}

func fault_write(
  state nbio.State,
  completion *nbio.Completion,
  file nbio.File,
  buffer []byte,
  offset int64,
  timeout time.Duration,
  callback nbio.Callback,
) {
  faults := state.(*Fault_Storage)
  if fault_due(faults, FAULT_WRITE) {
    offset = FAULT_OFFSET
  }
  if fault_due(faults, FAULT_SHORT) {
    buffer = buffer[:min(faults.Short, len(buffer))]
  }
  if fault_due(faults, FAULT_EMPTY) {
    buffer = buffer[:0]
  }
  if faults.Kind == FAULT_SHORT_FAIL {
    if faults.Spent {
      offset = FAULT_OFFSET
    } else {
      faults.Spent = true
      buffer = buffer[:min(faults.Short, len(buffer))]
    }
  }
  nbio.Storage_Write(faults.Inner, completion, file, buffer, offset, timeout, callback)
}

func fault_open_at(
  state nbio.State,
  completion *nbio.Completion,
  directory nbio.File,
  file_path string,
  options nbio.Open_At_Options,
  callback nbio.Callback,
) {
  faults := state.(*Fault_Storage)
  // The module file stands before every step as the configuration does, thus no fault reaches it.
  configured := file_path == harness.CONFIGURATION_PATH
  if file_path == harness.MODULE_PATH {
    configured = true
  }
  if configured {
    deliver := func(retired nbio.Completion_Handle) {
      fault_configured(faults, callback, retired)
    }
    nbio.Storage_Open_At(faults.Inner, completion, directory, file_path, options, deliver)
    return
  }
  if options.Create {
    // A fixture writes through this surface too, and its own files must arrive.
    nbio.Storage_Open_At(
      faults.Inner, completion, directory, file_path, options, callback,
    )
    return
  }
  // A walk opens its directory through this surface too, thus one fault names a Go file and
  // another names whatever directory the walk stands on.
  if strings.Has_Suffix(strings.Text(file_path), harness.GO_SUFFIX) {
    if fault_due(faults, FAULT_OPEN) {
      file_path = FAULT_ABSENT_PATH
    }
  }
  if !bool(strings.Has_Suffix(strings.Text(file_path), harness.GO_SUFFIX)) {
    if fault_due(faults, FAULT_DIRECTORY) {
      file_path = FAULT_ABSENT_PATH
    }
  }
  nbio.Storage_Open_At(faults.Inner, completion, directory, file_path, options, callback)
}

func fault_fsync(
  state nbio.State,
  completion *nbio.Completion,
  file nbio.File,
  timeout time.Duration,
  callback nbio.Callback,
) {
  nbio.Storage_Fsync(state.(*Fault_Storage).Inner, completion, file, timeout, callback)
}

func fault_mkdir_at(
  state nbio.State,
  completion *nbio.Completion,
  directory nbio.File,
  file_path string,
  permissions nbio.File_Permissions,
  callback nbio.Callback,
) {
  faults := state.(*Fault_Storage)
  nbio.Storage_Mkdir_At(faults.Inner, completion, directory, file_path, permissions, callback)
}

// Turns the entries of one pass around where the simulator left them, then hands on the
// completion the loop delivered.
func fault_reversed(
  entries []nbio.Directory_Entry,
  callback nbio.Callback,
  retired nbio.Completion_Handle,
) {
  held := entries[:retired.Data]
  for index := range len(held) / 2 {
    other := len(held) - 1 - index
    held[index], held[other] = held[other], held[index]
  }
  callback(retired)
}

func fault_directory_entries(
  state nbio.State,
  completion *nbio.Completion,
  directory nbio.File,
  buffer []byte,
  entries []nbio.Directory_Entry,
  callback nbio.Callback,
) {
  faults := state.(*Fault_Storage)
  deliver := callback
  if faults.Kind == FAULT_REVERSED {
    // The pass the simulator made stands, and its entries are turned around where it
    // left them: the completion is the one the loop delivered, not one a fixture made.
    deliver = func(retired nbio.Completion_Handle) {
      fault_reversed(entries, callback, retired)
    }
  }
  nbio.Storage_Get_Directory_Entries(
    faults.Inner, completion, directory, buffer, entries, deliver,
  )
}

// States the close a host could not finish on the completion the loop delivered, then hands it on.
// FAULT_FILE_NONE names no descriptor the simulator hands out, which starts past it.
const FAULT_FILE_NONE nbio.File = 0

// Records the descriptor harness.json opened on, then hands the open on.
func fault_configured(
  faults *Fault_Storage,
  callback nbio.Callback,
  retired nbio.Completion_Handle,
) {
  if retired.Error == nil {
    faults.Configuration = nbio.File(retired.Data)
  }
  callback(retired)
}

func fault_shut(callback nbio.Callback, retired nbio.Completion_Handle) {
  retired.Error = Shut_File
  callback(retired)
}

// Wraps a fixture's loop so every storage operation, close, spawn, and deinit goes through the
// fault surface, which hands each one on to the loop it wraps.
func fault_wire(subject *fixture, faults *Fault_Storage) {
  subject.Loop.Storage = fault_storage(faults)
  subject.Loop.Network = nbio.Network{}
  subject.Loop.Close_Procedure = fault_close
  subject.Loop.Spawn = fault_spawn
  subject.Loop.Deinit_Procedure = fault_deinit
}

// A spawn reaches the inner surface's state, which the wrapped storage half no longer carries.
func fault_spawn(
  state nbio.State,
  completion *nbio.Completion,
  request nbio.Process_Request,
  deadline time.Duration,
  callback nbio.Process_Callback,
) {
  faults := state.(*Fault_Storage)
  faults.Loop.Spawn(faults.Inner.State, completion, request, deadline, callback)
}

func fault_close(
  state nbio.State,
  completion *nbio.Completion,
  file nbio.File,
  callback nbio.Callback,
) {
  faults := state.(*Fault_Storage)
  deliver := callback
  if file == faults.Configuration {
    faults.Configuration = FAULT_FILE_NONE
    nbio.IO_Close(faults.Loop, completion, file, deliver)
    return
  }
  if fault_due(faults, FAULT_CLOSE) {
    // The close the simulator made stands and its descriptor is released; the error
    // is the answer a host gives, written on the completion the loop delivered.
    deliver = func(retired nbio.Completion_Handle) {
      fault_shut(callback, retired)
    }
  }
  nbio.IO_Close(faults.Loop, completion, file, deliver)
}

func fault_deinit(state nbio.State) {
  nbio.IO_Deinit(state.(*Fault_Storage).Loop)
}

func fault_status(state nbio.State, path string) (status nbio.File_Status, err error) {
  faults := state.(*Fault_Storage)
  status, err = nbio.Storage_Status(faults.Inner, path)
  if faults.Kind == FAULT_STATUS {
    // A host answers for a path that stood when the pass named it and stands no
    // longer, thus the walk meets an error where it asked what the name holds.
    return nbio.File_Status{}, Absent_Path
  }
  if faults.Kind == FAULT_LINK_CONTRACT {
    if contract_named(path) {
      status.Mode = status.Mode | nbio.FILE_MODE_SYMBOLIC_LINK
    }
    return status, err
  }
  if faults.Kind != FAULT_SYMBOLIC_LINK {
    return status, err
  }
  // A host writes what it likes into its own tree, and a link is what it writes when it
  // wants a run to read a file the walk never named.
  status.Mode = status.Mode | nbio.FILE_MODE_SYMBOLIC_LINK
  return status, err
}

func fault_read_link(
  state nbio.State, path string, destination []byte,
) (count int, err error) {
  return nbio.Storage_Read_Link(state.(*Fault_Storage).Inner, path, destination)
}

// SETUP_TRY_COUNT bounds how often one setup step is submitted again past a drawn fault.
const SETUP_TRY_COUNT = 64

// QUEUE_CHILD_COUNT is how many directories stand pending when the walk of one of them fails.
const QUEUE_CHILD_COUNT = 3

// WIDE_NAME_BYTES is how many bytes one name outside ASCII spends, two to each character.
const WIDE_NAME_BYTES = 120

// NODE_CONTENTS_BYTES is what one ordinary fixture file holds: enough for every source the
// witness writes but the widest ones, which ask for their own tree.
const NODE_CONTENTS_BYTES = 1024

// NODE_CAPACITY leaves room for the tree the seed generates beside the package each run builds,
// and for the chain that drives a path to its whole width, which spends one directory per two
// bytes of that width.
const NODE_CAPACITY = FIXTURE_NODE_COUNT + filepath.PATH_SIZE_MAXIMUM/len("/d")

// FIXTURE_NODE_COUNT holds the tree the seed generates and the package an ordinary fixture builds.
// A fixture that fills a table to its bound sizes its own tree to that table.
const FIXTURE_NODE_COUNT = 384

// DESCRIPTOR_CAPACITY covers the one directory or file a run holds open at a time, with slack.
const DESCRIPTOR_CAPACITY = 8

// OPERATION_CAPACITY covers the one operation a run keeps in flight, with slack.
const OPERATION_CAPACITY = 8

// PASS_COUNT bounds pump passes; a test package submits well under a hundred operations.
const PASS_COUNT = 16 * harness.FILE_COUNT_MAXIMUM

// PUMP_DURATION is strictly longer than every timeout a run grants, so the operation retires
// before the root's guard does.
const PUMP_DURATION = harness.READ_TIMEOUT + time.MINUTE

// OUTPUT_BYTES holds every line a test package yields.
const OUTPUT_BYTES = 4096

// TABLE_OUTPUT_BYTES holds every line a full package table yields: each package owes both its
// contract files, thus the run states two findings per slot.
const TABLE_OUTPUT_BYTES = 262144

// WIDE_OUTPUT_BYTES holds every line a package of full tables yields: one per Go file, and the
// contract findings past them. A fixture that reads such an output states its tally rather than
// searches it, since a text search stands inside a far smaller bound.
const WIDE_OUTPUT_BYTES = WIDE_LINE_BYTES *
  (harness.FILE_COUNT_MAXIMUM + harness.PACKAGE_COUNT_MAXIMUM)

// WIDE_LINE_BYTES holds one line of such an output.
const WIDE_LINE_BYTES = 128

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

// INDENTED_UNFORMATTED is INDENTED with a doubled space the canonical form drops, thus its fix
// is wider than the fix of a flat source and states that each fix carries its own count.
const INDENTED_UNFORMATTED = "package a\n\nfunc  g() (value int) {\n  return 1\n}\n"

// INDENTED stands in the canonical form with a body, thus a check of it states the form of a
// source that opens a line under a depth, which no flat source reaches.
const INDENTED = "package a\n\nfunc g() (value int) {\n  return 1\n}\n"

// CONTRACT_TEST_A is the test file one contract whose only leaf is A owes.
const CONTRACT_TEST_A = "package a\n\nfunc Test_A() {}\n"

// CONTRACT_MARKDOWN is one contract that holds: a single leaf with a body.
const CONTRACT_MARKDOWN = "\n# Contract\n\nIt holds.\n"

// CONTRACT_TEST is the test file that contract asks for.
const CONTRACT_TEST = "package a\n\nfunc Test_Contract() {}\n"

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

// One run per failing operation: the open of a file the walk found, the read of it, and the write
// of its fix. Each failure names its step and ends the run, which no clean run reaches.
func fixture_faults(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  kinds := [FAULT_KIND_COUNT]Fault_Kind{
    FAULT_OPEN, FAULT_READ, FAULT_WRITE, FAULT_DIRECTORY,
  }
  steps := [FAULT_KIND_COUNT]string{"open", "read", "fix", "walk"}
  paths := [FAULT_KIND_COUNT]string{
    "fault/a.go",
    "fault/a.go",
    "fault/a.go",
    "fault: " + harness.WALK_CAUSE_REFUSED,
  }
  for index, kind := range kinds {
    subject := fixture_new(t, memory, format_state, command, directory, seed)
    fixture_directory(t, subject, "fault")
    fixture_file(t, subject, "fault/a.go", UNFORMATTED)
    faults := Fault_Storage{Inner: subject.Loop.Storage, Loop: subject.Loop, Kind: kind}
    fault_wire(subject, &faults)
    arguments := []string{"harness", "fault"}
    if kind == FAULT_WRITE {
      arguments = []string{"harness", "fault", "-fix"}
    }
    fixture_run(t, subject, arguments)
    fixture_failed(t, subject, steps[index])
    // The line names the open directory at the walk and why, and the file being read at
    // every later step, thus a host reads which path the failed step stood on.
    held := fixture_errors(subject)
    named := "harness: failed to " + steps[index] + ": " + paths[index] + "\n"
    if held != named {
      t.Fatalf("a fault wrote %q, not %q", held, named)
    }
    // A run the command line refuses opens on the state the failed run left, thus
    // the mode that run was reading in stands rather than goes unstated.
    fixture_run(t, subject, []string{"harness", ""})
    fixture_failed(t, subject, "walk")
  }
  fixture_fix_open(t, memory, format_state, command, directory, seed)
  fixture_short_write(t, memory, format_state, command, directory, seed)
  fixture_shut_output(t, memory, format_state, command, directory, seed)
  fixture_read_passes(t, memory, format_state, command, directory, seed)
  // A failed read of the contract test file at the end of the widest chain: that path is
  // the widest a run reads, thus the failure line names a path of the whole width. The Go
  // file and the contract markdown stand before it, so the fault lets two reads pass.
  subject := fixture_new(t, memory, format_state, command, directory, seed)
  deep := fixture_chain(t, subject)
  fixture_file(t, subject, deep+"/a.go", CLEAN)
  fixture_file(t, subject, deep+"/"+requirements.MARKDOWN_NAME, CONTRACT_MARKDOWN)
  fixture_file(t, subject, deep+"/"+requirements.TEST_NAME, CONTRACT_TEST)
  faults := Fault_Storage{
    Inner: subject.Loop.Storage,
    Loop:  subject.Loop,
    Kind:  FAULT_READ,
    Skip:  CONTRACT_READ_SKIP,
  }
  fault_wire(subject, &faults)
  fixture_run(t, subject, []string{"harness", deep})
  fixture_failed(t, subject, "read")
}

// Leaf headings that stand at the second and third slots of the heading table: a leaf names the
// parent it stands under and the heading it is, thus both stand past the first slot here.
func fixture_kin(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  subject := fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "kin")
  fixture_file(t, subject, "kin/a.go", CLEAN)
  fixture_file(
    t,
    subject,
    "kin/"+requirements.MARKDOWN_NAME,
    "\n# A\n\nBody.\n\n# B\n\n### C\n\nBody.\n",
  )
  fixture_file(
    t,
    subject,
    "kin/"+requirements.TEST_NAME,
    "package a\n\nfunc Test_A() {}\n\nfunc Test_B_C() {}\n",
  )
  fixture_run(t, subject, []string{"harness", "kin"})
  fixture_passed(t, subject)
  subject = fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "kith")
  fixture_file(t, subject, "kith/a.go", CLEAN)
  fixture_file(
    t,
    subject,
    "kith/"+requirements.MARKDOWN_NAME,
    "\n# A\n\nBody.\n\n# B\n\nBody.\n\n# C\n\n### D\n\nBody.\n",
  )
  fixture_file(
    t,
    subject,
    "kith/"+requirements.TEST_NAME,
    "package a\n\nfunc Test_A() {}\n\nfunc Test_B() {}\n\nfunc Test_C_D() {}\n",
  )
  fixture_run(t, subject, []string{"harness", "kith"})
  fixture_passed(t, subject)
}

// A package clause of main_test names the external test package of a main package, thus the
// package it stands in owes no contract, the way one holding a main file owes none.
func fixture_external(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  subject := fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "outer")
  fixture_file(t, subject, "outer/m.go", MAIN)
  fixture_file(t, subject, "outer/m_test.go", "package main_test\n\nfunc Test_M() {}\n")
  fixture_run(t, subject, []string{"harness", "outer"})
  fixture_passed(t, subject)
  subject = fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "lone")
  fixture_file(t, subject, "lone/m_test.go", "package main_test\n\nfunc Test_M() {}\n")
  fixture_run(t, subject, []string{"harness", "lone"})
  fixture_passed(t, subject)
}

// A child whose joined path stands past what a path holds: the walk states the tree it can name
// rather than joins a path past its store, thus the run fails its walk.
func fixture_deep(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  subject := fixture_new(t, memory, format_state, command, directory, seed)
  deepest := fixture_chain(t, subject)
  name := make([]byte, filepath.PATH_SIZE_MAXIMUM-len(deepest))
  for index := range name {
    name[index] = 'd'
  }
  fixture_directory(t, subject, deepest+"/"+string(name))
  fixture_run(t, subject, []string{"harness", deepest})
  fixture_failed(t, subject, "walk")
  // The same chain one byte narrower, so the walk holds that directory rather than bars
  // it: a name inside it joins a path one byte past the store, thus the pass fails where
  // the widest path a run holds is the one it stands on.
  subject = fixture_new(t, memory, format_state, command, directory, seed)
  deepest = fixture_chain(t, subject)
  widest := deepest + "/" + string(name[:len(name)-1])
  fixture_directory(t, subject, widest)
  fixture_file(t, subject, widest+"/a.go", CLEAN)
  fixture_run(t, subject, []string{"harness", deepest})
  fixture_failed(t, subject, "walk")
  // One file whose own path spends every byte a path holds, fixed where it stands: the
  // fix opens the widest path a run names rather than one short of it.
  subject = fixture_new(t, memory, format_state, command, directory, seed)
  deepest = fixture_chain(t, subject)
  spent := make([]byte, filepath.PATH_SIZE_MAXIMUM-len(deepest)-len("/.go"))
  for index := range spent {
    spent[index] = 'w'
  }
  fixture_file(t, subject, deepest+"/"+string(spent)+".go", UNFORMATTED)
  fixture_file(t, subject, deepest+"/"+requirements.MARKDOWN_NAME, CONTRACT_MARKDOWN)
  // A link of the same width beside it: the widest path a link stands at.
  fixture_link(t, subject, deepest+"/"+string(spent)+".ln", "missing")
  fixture_run(t, subject, []string{"harness", deepest, "-fix"})
  // A package at the end of the widest chain, holding a contract and no test file: the
  // test name is the widest a contract carries, thus that finding names a path of the
  // whole width, and a second run opens on the report it left.
  subject = fixture_new(t, memory, format_state, command, directory, seed)
  deepest = fixture_chain(t, subject)
  fixture_file(t, subject, deepest+"/a.go", UNFORMATTED)
  fixture_file(t, subject, deepest+"/"+requirements.MARKDOWN_NAME, CONTRACT_MARKDOWN)
  fixture_run(t, subject, []string{"harness", deepest, "-json"})
  fixture_states(t, subject, "not formatted")
  fixture_run(t, subject, []string{"harness", deepest, "-json"})
  fixture_states(t, subject, "not formatted")
}

// Two files of one package fixed in one run: the fix state carries from the first to the second,
// thus each file stands in the canonical form rather than the second taking the first one's count.
func fixture_pair(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  subject := fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "twin")
  fixture_file(t, subject, "twin/m.go", MAIN)
  fixture_file(t, subject, "twin/a.go", UNFORMATTED)
  fixture_file(t, subject, "twin/b.go", INDENTED_UNFORMATTED)
  fixture_run(t, subject, []string{"harness", "twin", "-fix"})
  fixture_passed(t, subject)
  held := fixture_contents(t, subject, "twin/a.go")
  if held != CLEAN {
    t.Fatalf("a first fix of two left %q", held)
  }
  held = fixture_contents(t, subject, "twin/b.go")
  if held != INDENTED {
    t.Fatalf("a second fix of two left %q", held)
  }
}

// A refusal at the second line and second column, in a package owing a contract: the contract
// read opens on the report that refusal filled, thus both its numbers stand at two there. A
// grouped declaration prints ungrouped, thus the run states its file unformatted.
func fixture_after(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  subject := fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "after")
  fixture_file(t, subject, "after/a.go", "package a\n &")
  // The contract files stand, thus the run reads them after the refusal it stated and
  // the read opens on the report that refusal filled.
  fixture_file(t, subject, "after/"+requirements.MARKDOWN_NAME, CONTRACT_MARKDOWN)
  fixture_file(t, subject, "after/"+requirements.TEST_NAME, CONTRACT_TEST)
  fixture_run(t, subject, []string{"harness", "after", "-json"})
  fixture_states(t, subject, `"line":2,"column":2`)
  // A second run over the same package: it opens on the report the run before it left,
  // thus the column that report names stands past the first.
  fixture_run(t, subject, []string{"harness", "after", "-json"})
  subject = fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "group")
  fixture_file(t, subject, "group/m.go", "package main\n\nimport (\n\t\"x\"\n)\n")
  fixture_run(t, subject, []string{"harness", "group"})
  fixture_states(t, subject, "group/m.go: not formatted")
  fixture_run(t, subject, []string{"harness", "group"})
  // A walk that fails with directories still queued, then a second run: the count a run
  // opens on stands where the walk left it rather than at none.
  subject = fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "queue")
  for _, name := range [QUEUE_CHILD_COUNT]string{"a", "b", "c"} {
    fixture_directory(t, subject, "queue/"+name)
  }
  // The queue pops what it pushed last, thus the bad name stands in the child the walk
  // opens first and the children before it stay pending.
  fixture_file(t, subject, "queue/c/bad\nname.go", CLEAN)
  fixture_run(t, subject, []string{"harness", "queue"})
  fixture_failed(t, subject, "walk")
  fixture_run(t, subject, []string{"harness", "queue"})
  // One child fewer: the walk leaves one directory pending rather than two.
  subject = fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "pair")
  fixture_directory(t, subject, "pair/a")
  fixture_directory(t, subject, "pair/b")
  fixture_file(t, subject, "pair/b/bad\nname.go", CLEAN)
  fixture_run(t, subject, []string{"harness", "pair"})
  fixture_failed(t, subject, "walk")
  fixture_run(t, subject, []string{"harness", ""})
  fixture_failed(t, subject, "walk")
}

// A contract file is a Go file too, thus the run states its form once rather than once for the
// walk that read it and once for the check that read it again.
func fixture_twice(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  subject := fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "twice")
  fixture_file(t, subject, "twice/a.go", CLEAN)
  fixture_file(t, subject, "twice/"+requirements.MARKDOWN_NAME, CONTRACT_MARKDOWN)
  fixture_file(
    t,
    subject,
    "twice/"+requirements.TEST_NAME,
    "package a\n\nfunc  Test_Contract() {}\n",
  )
  fixture_run(t, subject, []string{"harness", "twice"})
  fixture_wrote(t, subject, "twice/requirements_test.go: not formatted\n")
  subject = fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "both")
  fixture_file(t, subject, "both/a.go", CLEAN)
  fixture_file(t, subject, "both/"+requirements.MARKDOWN_NAME, CONTRACT_MARKDOWN)
  fixture_file(
    t, subject, "both/"+requirements.TEST_NAME, "package a\n\nfunc  Test_Other() {}\n",
  )
  fixture_run(t, subject, []string{"harness", "both"})
  // A contract file that breaks a form rule and a contract rule states both: one rule
  // stands apart from the other, thus neither reading of the file bars the other.
  fixture_wrote(
    t,
    subject,
    "both/requirements_test.go: not formatted\n"+
      `both/requirements_test.go:3: The file needs Test_Contract for the `+
      `leaf "Contract". Declare the tests in leaf order at the top of the `+
      "file.\n",
  )
  // The same package fixed: the form fault is neither written nor counted, and the fix moved
  // the lines the one parse of the test file read, thus its tests wait for the run after it.
  fixture_run(t, subject, []string{"harness", "both", "-fix"})
  fixture_wrote(
    t,
    subject,
    "both/requirements_test.go: The file was rewritten by this run. Run the check again for "+
      "its tests.\n",
  )
  // The contract fault stands, since a tier two diagnostic carries no fix.
  fixture_run(t, subject, []string{"harness", "both", "-fix"})
  fixture_wrote(
    t,
    subject,
    `both/requirements_test.go:3: The file needs Test_Contract for the `+
      `leaf "Contract". Declare the tests in leaf order at the top of the `+
      "file.\n",
  )
  held := fixture_contents(t, subject, "both/"+requirements.TEST_NAME)
  if held != "package a\n\nfunc Test_Other() {}\n" {
    t.Fatalf("a fix of a contract file left %q", held)
  }
}

// A name holding a backslash stands escaped in flat JSON, thus the object a run writes parses as
// one object rather than breaks at the escape the name opened.
func fixture_escaped(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  subject := fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "back")
  fixture_file(t, subject, "back/a\\b.go", UNFORMATTED)
  fixture_run(t, subject, []string{"harness", "back", "-json"})
  fixture_states(t, subject, `"path":"back/a\\b.go"`)
  subject = fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "heading")
  fixture_file(t, subject, "heading/a.go", CLEAN)
  fixture_file(
    t,
    subject,
    "heading/"+requirements.MARKDOWN_NAME,
    "\n# A\"B\n\nBody.\n\n# A\"B\n\nBody.\n",
  )
  fixture_run(t, subject, []string{"harness", "heading", "-json"})
  // A heading a host wrote holds the quote it chose, thus the message that quotes it back
  // stands escaped and the object parses as one object.
  fixture_states(t, subject, `"message":"The heading \"A\"B\" is not spelled`)
}

// A name holding a space or a byte outside ASCII is one a host writes and a disk holds, thus the
// walk states its file rather than refuses a tree for the names its own host chose.
func fixture_named(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  subject := fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "named")
  fixture_file(t, subject, "named/m.go", MAIN)
  fixture_file(t, subject, "named/a b.go", UNFORMATTED)
  fixture_file(t, subject, "named/\u00e9.go", UNFORMATTED)
  fixture_run(t, subject, []string{"harness", "named"})
  fixture_states(t, subject, "named/a b.go: not formatted")
  fixture_states(t, subject, "named/\u00e9.go: not formatted")
}

// A step that fails states its failure and retracts nothing: the diagnostics the run wrote
// before it stand, thus a host reads what the run found as well as why it stopped.
func fixture_partial(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  subject := fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "part")
  fixture_file(t, subject, "part/m.go", MAIN)
  fixture_file(t, subject, "part/a.go", UNFORMATTED)
  fixture_file(t, subject, "part/b.go", UNFORMATTED)
  // The run states what the declaration pass left without a read, thus the write of the
  // second line is the step that fails.
  cut := short_stream{Inner: subject.Runner.Output, Short: 0, Skip: 1}
  subject.Runner.Output = nbio.Stream{State: &cut, Procedure: short_stream_write}
  fixture_run(t, subject, []string{"harness", "part"})
  fixture_wrote(t, subject, "part/a.go: not formatted\n")
  fixture_failed(t, subject, "report")
  // The same fault under a fix: the file the run rewrote stands rewritten on the disk,
  // since a write that landed is not taken back by a step that failed after it.
  subject = fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "part")
  fixture_file(t, subject, "part/m.go", MAIN)
  fixture_file(t, subject, "part/a.go", UNFORMATTED)
  fixture_file(t, subject, "part/b.go", UNFORMATTED)
  fixes := Fault_Storage{
    Inner: subject.Loop.Storage,
    Loop:  subject.Loop,
    Kind:  FAULT_WRITE,
    Skip:  FIX_WRITE_SKIP,
  }
  fault_wire(subject, &fixes)
  fixture_run(t, subject, []string{"harness", "part", "-fix"})
  fixture_failed(t, subject, "fix")
  held := fixture_contents(t, subject, "part/a.go")
  if held != CLEAN {
    t.Fatalf("a fix before a failed step left %q", held)
  }
}

// A run holding a file the parser refuses bars every rename it would make: the refused file may
// spell the name, and a rename that reaches every other file breaks the build.
func fixture_barred(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  subject := fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "barred")
  fixture_file(t, subject, "barred/a.go", DECLARATION)
  fixture_file(t, subject, "barred/b.go", REFERENCE)
  fixture_file(t, subject, "barred/c.go", REFUSED)
  fixture_file(t, subject, "barred/"+requirements.MARKDOWN_NAME, CONTRACT_MARKDOWN)
  fixture_file(t, subject, "barred/"+requirements.TEST_NAME, CONTRACT_TEST)
  fixture_run(t, subject, []string{"harness", "barred", "-fix"})
  // The run reaches the check, thus the name stands where it stands because the run
  // barred the rename rather than because it never got that far.
  fixture_states(t, subject, "barred/c.go:")
  held := fixture_contents(t, subject, "barred/a.go")
  if held != DECLARATION {
    t.Fatalf("a run holding a refused file renamed, leaving %q", held)
  }
  held = fixture_contents(t, subject, "barred/b.go")
  if held != REFERENCE {
    t.Fatalf("a run holding a refused file renamed a reference, leaving %q", held)
  }
  // A bar on renaming is a bar on renaming alone: a file that owes a form still stands in
  // the canonical form once the run has fixed it.
  subject = fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "barred")
  fixture_file(t, subject, "barred/a.go", DECLARATION)
  fixture_file(t, subject, "barred/c.go", REFUSED)
  fixture_file(t, subject, "barred/u.go", UNFORMATTED)
  fixture_file(t, subject, "barred/"+requirements.MARKDOWN_NAME, CONTRACT_MARKDOWN)
  fixture_file(t, subject, "barred/"+requirements.TEST_NAME, CONTRACT_TEST)
  fixture_run(t, subject, []string{"harness", "barred", "-fix"})
  held = fixture_contents(t, subject, "barred/u.go")
  if held != CLEAN {
    t.Fatalf("a run that barred a rename left a form unfixed as %q", held)
  }
  held = fixture_contents(t, subject, "barred/a.go")
  if held != DECLARATION {
    t.Fatalf("a run holding a refused file renamed, leaving %q", held)
  }
  // A contract test file is a Go file too, thus a directory holding it alone is a package
  // and owes the contract that test stands for.
  subject = fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "onlytest")
  fixture_file(t, subject, "onlytest/"+requirements.TEST_NAME, CONTRACT_TEST)
  fixture_run(t, subject, []string{"harness", "onlytest"})
  fixture_wrote(
    t,
    subject,
    "onlytest/"+
      requirements.MARKDOWN_NAME+
      ": The package has no REQUIREMENTS.md. Add REQUIREMENTS.md.\n",
  )
  // A file the parser refuses states its own diagnostic and bars nothing else: the
  // contract of its package is checked after it, thus both stand and in that order.
  subject = fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "mixed")
  fixture_file(t, subject, "mixed/a.go", REFUSED)
  fixture_run(t, subject, []string{"harness", "mixed"})
  fixture_wrote(
    t,
    subject,
    "mixed/a.go:3:9: Fix the syntax at this token.\n"+
      "mixed/"+
      requirements.MARKDOWN_NAME+
      ": The package has no REQUIREMENTS.md. Add REQUIREMENTS.md.\n",
  )
}

// A tier two diagnostic carries no fix, thus a run that fixed it would open its file cut to
// nothing and write none of it back. The file stands as the host wrote it.
func fixture_kept(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  subject := fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "kept")
  fixture_file(t, subject, "kept/m.go", MAIN)
  fixture_file(t, subject, "kept/a.go", REFUSED)
  fixture_run(t, subject, []string{"harness", "kept", "-fix"})
  held := fixture_contents(t, subject, "kept/a.go")
  if held != REFUSED {
    t.Fatalf("a fix rewrote a refused file as %q", held)
  }
}

// Two diagnostics as json: each stands on a line of its own, thus a host reads them one object
// to a line rather than as one object of two.
func fixture_lines_json(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  subject := fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "pairjson")
  fixture_file(t, subject, "pairjson/m.go", MAIN)
  fixture_file(t, subject, "pairjson/a.go", UNFORMATTED)
  fixture_file(t, subject, "pairjson/b.go", UNFORMATTED)
  fixture_run(t, subject, []string{"harness", "pairjson", "-json"})
  fixture_wrote(
    t,
    subject,
    `{"path":"pairjson/a.go","line":0,"column":0,"tier":1,`+
      `"rule":"unformatted","message":"not formatted"}`+
      "\n"+
      `{"path":"pairjson/b.go","line":0,"column":0,"tier":1,`+
      `"rule":"unformatted","message":"not formatted"}`+
      "\n",
  )
}

// One diagnostic as json, stated whole: a host reads the object by its shape, thus the braces,
// the key order, and every value stand as the contract names them.
func fixture_shape(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  subject := fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "shape")
  fixture_file(t, subject, "shape/m.go", MAIN)
  fixture_file(t, subject, "shape/a.go", UNFORMATTED)
  fixture_run(t, subject, []string{"harness", "shape"})
  fixture_wrote(t, subject, "shape/a.go: not formatted\n")
  fixture_run(t, subject, []string{"harness", "shape", "-json"})
  fixture_states(
    t,
    subject,
    `{"path":"shape/a.go","line":0,"column":0,"tier":1,`+
      `"rule":"unformatted","message":"not formatted"}`,
  )
  // A run the command line refuses opens on the report the run before it wrote.
  fixture_run(t, subject, []string{"harness", ""})
  fixture_failed(t, subject, "walk")
  // A package fixed: the fix replaces the file, thus the diagnostic it carried is neither
  // written nor counted and the pass line opens the output.
  subject = fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "shape")
  fixture_file(t, subject, "shape/m.go", MAIN)
  fixture_file(t, subject, "shape/a.go", UNFORMATTED)
  fixture_run(t, subject, []string{"harness", "shape", "-fix"})
  fixture_opened(t, subject, "\u2713 all checks passed\n")
  fixture_passed(t, subject)
  // A fix replaces the file its diagnostic named and no other, thus a file the run found
  // clean stands byte for byte as the host wrote it.
  held := fixture_contents(t, subject, "shape/m.go")
  if held != MAIN {
    t.Fatalf("a fix rewrote a clean file as %q", held)
  }
  subject = fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "shape")
  fixture_file(t, subject, "shape/m.go", MAIN)
  fixture_file(t, subject, "shape/a.go", REFUSED)
  fixture_run(t, subject, []string{"harness", "shape"})
  fixture_wrote(t, subject, "shape/a.go:3:9: Fix the syntax at this token.\n")
  fixture_run(t, subject, []string{"harness", "shape", "-json"})
  fixture_states(
    t,
    subject,
    `{"path":"shape/a.go","line":3,"column":9,"tier":2,`+
      `"rule":"refused","message":"Fix the syntax at this token."}`,
  )
  subject = fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "shape")
  fixture_file(t, subject, "shape/a.go", CLEAN)
  fixture_run(t, subject, []string{"harness", "shape"})
  fixture_wrote(
    t,
    subject,
    "shape/REQUIREMENTS.md: The package has no REQUIREMENTS.md. "+
      "Add REQUIREMENTS.md.\n",
  )
  fixture_run(t, subject, []string{"harness", "shape", "-json"})
  fixture_states(
    t,
    subject,
    `{"path":"shape/REQUIREMENTS.md","line":0,"column":0,"tier":2,`+
      `"rule":"requirements","message":"The package has no `+
      `REQUIREMENTS.md. Add REQUIREMENTS.md."}`,
  )
}

// A contract whose last section carries no body line, and one a host ended on a carriage return
// alone: a section closes where the file ends as well as where the next heading stands, thus the
// run states both rather than reads past the end of what it holds.
func fixture_ended(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  subject := fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "ended")
  fixture_file(t, subject, "ended/a.go", CLEAN)
  fixture_file(t, subject, "ended/"+requirements.MARKDOWN_NAME, "\n# A\n\nBody.\n\n# B\n")
  fixture_file(
    t,
    subject,
    "ended/"+requirements.TEST_NAME,
    "package a\n\nfunc Test_A() {}\n\nfunc Test_B() {}\n",
  )
  fixture_run(t, subject, []string{"harness", "ended"})
  fixture_states(t, subject, `The section "B" has no body line.`)
  subject = fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "cut")
  fixture_file(t, subject, "cut/a.go", CLEAN)
  fixture_file(t, subject, "cut/"+requirements.MARKDOWN_NAME, "\r\n# A\r\n\r")
  fixture_file(t, subject, "cut/"+requirements.TEST_NAME, CONTRACT_TEST_A)
  fixture_run(t, subject, []string{"harness", "cut"})
  fixture_states(t, subject, `The section "A" has no body line.`)
}

// A host that answers no ask of what a path holds: the walk states the tree it can name rather
// than records a file it never learned the kind of, thus the run fails its walk.
func fixture_absent(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  subject := fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "gone")
  fixture_file(t, subject, "gone/a.go", CLEAN)
  faults := Fault_Storage{
    Inner: subject.Loop.Storage,
    Loop:  subject.Loop,
    Kind:  FAULT_STATUS,
  }
  fault_wire(subject, &faults)
  fixture_run(t, subject, []string{"harness", "gone"})
  fixture_failed(t, subject, "walk")
  // A path of bytes outside ASCII stands in the object as the host wrote it: flat JSON
  // escapes the quote and the backslash alone, thus a name of such bytes spends the same
  // bytes in the object as on the disk.
  subject = fixture_new(t, memory, format_state, command, directory, seed)
  wide := make([]byte, WIDE_NAME_BYTES)
  for index := range wide {
    wide[index] = 0xc3
    if index%2 == 1 {
      wide[index] = 0xa9
    }
  }
  name := string(wide)
  fixture_directory(t, subject, name)
  fixture_file(t, subject, name+"/a.go", UNFORMATTED)
  fixture_run(t, subject, []string{"harness", name, "-json"})
  fixture_states(t, subject, `"path":"`+name+`/a.go"`)
}

// A Go file a host states as a symbolic link and then reads no target of: the walk passes over it
// and no run reads or rewrites whatever it points at, and a link with no target resolves to
// nothing, thus the run names it.
func fixture_linked(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  subject := fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "linked")
  fixture_file(t, subject, "linked/a.go", UNFORMATTED)
  fixture_file(t, subject, "linked/m.go", MAIN)
  links := Fault_Storage{
    Inner: subject.Loop.Storage,
    Loop:  subject.Loop,
    Kind:  FAULT_SYMBOLIC_LINK,
  }
  fault_wire(subject, &links)
  fixture_run(t, subject, []string{"harness", "linked", "-fix"})
  fixture_states(t, subject, "linked/a.go: "+harness.MESSAGE_ORPHAN)
  held := fixture_contents(t, subject, "linked/a.go")
  if held != UNFORMATTED {
    t.Fatalf("a fix rewrote a linked file as %q", held)
  }
  // A contract file a host wrote as a link names no contract either: the run would read
  // whatever it stands for, thus the package is told it holds none rather than answered
  // against a file the walk never named. The Go file stands as itself, so the directory
  // is a package that owes a contract it holds none of.
  subject = fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "carried")
  fixture_file(t, subject, "carried/a.go", CLEAN)
  fixture_file(t, subject, "carried/"+requirements.MARKDOWN_NAME, CONTRACT_MARKDOWN)
  fixture_file(t, subject, "carried/"+requirements.TEST_NAME, CONTRACT_TEST)
  carried := Fault_Storage{
    Inner: subject.Loop.Storage,
    Loop:  subject.Loop,
    Kind:  FAULT_LINK_CONTRACT,
  }
  fault_wire(subject, &carried)
  fixture_run(t, subject, []string{"harness", "carried"})
  fixture_states(t, subject, "carried/REQUIREMENTS.md: The package has no REQUIREMENTS.md.")
  // A directory a host named for a contract file names no contract: the walk opens it the
  // way it opens any child, thus the package that holds it is told it holds none.
  subject = fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "shaped")
  fixture_file(t, subject, "shaped/a.go", CLEAN)
  fixture_directory(t, subject, "shaped/"+requirements.MARKDOWN_NAME)
  fixture_directory(t, subject, "shaped/"+requirements.TEST_NAME)
  fixture_run(t, subject, []string{"harness", "shaped"})
  fixture_states(t, subject, "shaped/REQUIREMENTS.md: The package has no REQUIREMENTS.md.")
}

// Symbolic links whose target resolves to nothing, in each way one does, beside links that
// resolve: each orphan fails the run with its own line, and a tree past the orphan table fails the
// walk.
func fixture_orphans(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  subject := fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "orphans")
  fixture_file(t, subject, "orphans/m.go", MAIN)
  fixture_link(t, subject, "orphans/gone", "missing")
  fixture_link(t, subject, "orphans/loop", "loop")
  fixture_link(t, subject, "orphans/far", "/missing")
  fixture_link(t, subject, "orphans/empty", "")
  fixture_link(t, subject, "orphans/kept", "m.go")
  fixture_link(t, subject, "orphans/hop", "kept")
  fixture_run(t, subject, []string{"harness", "orphans", "-json"})
  fixture_states(
    t,
    subject,
    `{"path":"orphans/loop","line":0,"column":0,"tier":2,"rule":"symlinks"`,
  )
  // A link at a path of bytes flat JSON escapes six-fold: its object outgrows the store, thus
  // the run names the failed report.
  subject = fixture_new(t, memory, format_state, command, directory, seed)
  component := make([]byte, ESCAPED_COMPONENT_BYTES)
  for index := range component {
    component[index] = '<'
  }
  path := string(component)
  fixture_directory(t, subject, path)
  for range ESCAPED_COMPONENT_COUNT - 1 {
    path += "/" + string(component)
    fixture_directory(t, subject, path)
  }
  fixture_link(t, subject, path+"/o", "missing")
  fixture_run(t, subject, []string{"harness", path, "-json"})
  fixture_failed(t, subject, "report")
  for _, count := range []int{harness.ORPHAN_COUNT_MAXIMUM, harness.ORPHAN_COUNT_MAXIMUM + 1} {
    subject = fixture_new(t, memory, format_state, command, directory, seed)
    fixture_directory(t, subject, "full")
    for index := range count {
      fixture_link(t, subject, fmt.Sprintf("full/o%d", index), "missing")
    }
    fixture_run(t, subject, []string{"harness", "full"})
  }
  if fixture_errors(subject) == "" {
    t.Fatalf("a tree past the orphan table wrote %q", fixture_written(subject))
  }
  // The shortest path a link stands at, alone in its tree. The report it leaves is the one the
  // next run opens on, thus each step that takes a report opens on the shortest path too.
  subject = fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "o")
  fixture_link(t, subject, "o/x", "missing")
  fixture_run(t, subject, []string{"harness", "o", "-json"})
  fixture_states(t, subject, `"path":"o/x"`)
  fixture_file(t, subject, "o/a.go", UNFORMATTED)
  fixture_run(t, subject, []string{"harness", "o"})
  fixture_states(t, subject, "o/REQUIREMENTS.md: The package has no REQUIREMENTS.md.")
}

// CONFLICTED_LINE_BYTES is a source whose conflict marker stands on the last line a check
// indexes.
const CONFLICTED_LINE_BYTES = token.LINE_MAXIMUM + len("%%%%%%%")

// Go files and a contract with a line that opens on a conflict marker: on the first line, the
// second, and the last line a check indexes. Each names that line.
func fixture_conflicted(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  subject := fixture_sized(
    t,
    memory,
    format_state,
    command,
    directory,
    seed,
    TALL_NODE_COUNT,
    CONFLICTED_LINE_BYTES,
  )
  fixture_directory(t, subject, "conflicted")
  fixture_file(t, subject, "conflicted/a.go", ">>>>>>> side\n")
  fixture_file(t, subject, "conflicted/b.go", "package a\n<<<<<<< side\n")
  last := make([]byte, token.LINE_MAXIMUM-1)
  for index := range last {
    last[index] = '\n'
  }
  fixture_file(t, subject, "conflicted/c.go", string(last)+"%%%%%%%")
  fixture_file(t, subject, "conflicted/"+requirements.MARKDOWN_NAME, "\n# A\n\nBody.\n+++++++\n")
  fixture_run(t, subject, []string{"harness", "conflicted", "-json"})
  fixture_states(
    t,
    subject,
    `{"path":"conflicted/b.go","line":2,"column":1,"tier":2,"rule":"conflicted"`,
  )
  fixture_states(t, subject, fmt.Sprintf(`"path":"conflicted/c.go","line":%d,`, token.LINE_MAXIMUM))
  fixture_states(t, subject, requirements.MESSAGE_TEXT_CONFLICT_MARKER)
}

// BACKSLASH opens a JSON escape, spelled apart so no escape stands in this source.
const BACKSLASH = "\\"

// A harness.json of each shape the decoder meets: one that decodes, holding lists, words, and
// escapes, then one of each fault, each absent key in turn, and the edges of the source size.
// Each fault fails the run before the walk and names itself.
func fixture_configured(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  subject := fixture_sized(
    t,
    memory,
    format_state,
    command,
    directory,
    seed,
    TALL_NODE_COUNT,
    configuration.SOURCE_STORAGE_SIZE+1,
  )
  fixture_directory(t, subject, "c")
  // The backslash escapes the byte past it, which a glob admits.
  escaped := `"b\"\/\\\b\f\n\r\t` +
    BACKSLASH +
    "u00e9" +
    BACKSLASH +
    "ud83d" +
    BACKSLASH +
    "ude00\xc3\xa9\xe2\x82\xac\xf0\x9f\x98\x80\""
  decoded := strings_replace(
    CONFIGURATION,
    `"ignore":[]`,
    `"ignore":["a","!/b/","c\\*",`+escaped+"]",
  )
  decoded = strings_replace(decoded, `{"util":[]}`, `{"a":["b","c"],"d":[]}`)
  fixture_file(t, subject, harness.CONFIGURATION_PATH, decoded)
  fixture_run(t, subject, []string{"harness", "c"})
  // The entries decode whole, and each names nothing the host holds.
  fixture_states(
    t,
    subject,
    "harness.json: The ignore entry \"a\" matches no file and no directory.",
  )
  for _, source := range fixture_configurations() {
    fixture_file(t, subject, harness.CONFIGURATION_PATH, source)
    fixture_run(t, subject, []string{"harness", "c"})
    fixture_failed(t, subject, "harness: harness.json: ")
  }
}

// IMPORT_PATH_BYTES is the width of the import path whose message fills the notice store.
const IMPORT_PATH_BYTES = 700

// Go files importing each edge of the families lint bans, from each gateway: a path of no byte,
// one byte, two, a family of each end, a path past the quote a message holds, and the imports each
// gateway may make and may not.
func fixture_imported(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  subject := fixture_new(t, memory, format_state, command, directory, seed)
  wide := make([]byte, IMPORT_PATH_BYTES)
  for index := range wide {
    wide[index] = 'x'
  }
  sources := []string{
    "a/a.go",
    "package a\n\nimport \"\"\nimport \"a\"\nimport \"ab\"\nimport \"fmt\"\nimport \"uuid\"\n",
    "b/b.go",
    "package b\n\nimport \"os\"\n",
    "c/c.go",
    "package c\n\nimport \"math/rand\"\n",
    "d/d.go",
    "package d\n\nimport \"go/ast\"\n",
    "e/e.go",
    "package e\n\nimport \"container/" + string(wide) + "\"\n",
    "std/os/default/o.go",
    "package os\n\nimport \"os\"\nimport \"os/exec\"\nimport \"strings\"\n",
    "std/sim/nbio/default/n.go",
    "package nbio\n\nimport \"os/signal\"\nimport \"syscall\"\n",
    "std/sim/time/default/t.go",
    "package time\n\nimport \"os/exec\"\nimport \"syscall\"\n",
    "std/sim/aver/v.go",
    "package aver\n\nimport \"strings\"\n",
    // Each port stands above its default tier, which the composition tier then binds.
    "std/os/p.go",
    "package os\n",
    "std/sim/nbio/p.go",
    "package nbio\n",
    "std/sim/time/p.go",
    "package time\n",
  }
  for _, path := range []string{
    "shared",
    "shared/a",
    "shared/b",
    "shared/c",
    "shared/d",
    "shared/e",
    "shared/std",
    "shared/std/os",
    "shared/std/os/default",
    "shared/std/sim",
    "shared/std/sim/nbio",
    "shared/std/sim/nbio/default",
    "shared/std/sim/time",
    "shared/std/sim/time/default",
    "shared/std/sim/aver",
  } {
    fixture_directory(t, subject, path)
  }
  for index := 0; index < len(sources); index += 2 {
    fixture_file(t, subject, "shared/"+sources[index], sources[index+1])
  }
  fixture_run(t, subject, []string{"harness", "shared"})
  for _, found := range []string{
    "shared/a/a.go:7:8: Banned import \"uuid\" belongs to stdlib family \"uuid\".",
    "shared/std/os/default/o.go:5:8: Banned import \"strings\"",
    "shared/std/sim/time/default/t.go:3:8: Banned import \"os/exec\" belongs to stdlib family \"os\".",
  } {
    if !fixture_holds(fixture_written(subject), found) {
      t.Fatalf("an import run wrote no %q", found)
    }
  }
  for _, absent := range []string{"shared/std/sim/nbio/default/n.go", "shared/std/sim/aver/v.go"} {
    if fixture_holds(fixture_written(subject), absent+":") {
      t.Fatalf("an import run faulted the gateway file %s", absent)
    }
  }
}

// Go files each declaring a name that breaks a naming rule, behind one that imports a banned
// package, stated as text and as flat JSON: the last finding a report holds names a name.
func fixture_names(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  sources := []string{
    "a/a.go",
    "package a\n\nimport \"strings\"\n\nvar util = 1\n",
    "b/b.go",
    "package b\n\nvar max_line = 1\n",
    "c/c.go",
    "package c\n\nfunc F() {\n  for i := 0; i < 3; i++ {\n  }\n}\n",
    "d/d.go",
    "package d\n\nvar util_table = 1\n",
  }
  for _, flags := range [][]string{{}, {"-json"}} {
    subject := fixture_new(t, memory, format_state, command, directory, seed)
    for _, path := range []string{"shared", "shared/a", "shared/b", "shared/c", "shared/d"} {
      fixture_directory(t, subject, path)
    }
    for index := 0; index < len(sources); index += 2 {
      fixture_file(t, subject, "shared/"+sources[index], sources[index+1])
    }
    fixture_run(t, subject, append([]string{"harness", "shared"}, flags...))
    for _, found := range []string{"util_table", "line_max", "i_index", "strings"} {
      if !fixture_holds(fixture_written(subject), found) {
        t.Fatalf("a naming run wrote no %q", found)
      }
    }
  }
}

// A package whose file hides a sibling's top-level name, one whose file hides a name of its own
// scopes, an external test package beside its library, and a package holding a file the parser
// refuses, whose siblings go unjudged.
func fixture_shadowed(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  sources := []string{
    "s/a.go",
    "package s\n\nfunc value() {}\n",
    "s/b.go",
    "package s\n\nfunc F() {\n  value := 1\n}\n",
    "s/e_test.go",
    "package s_test\n\nfunc G() {\n  value := 1\n}\n",
    "c/c.go",
    "package c\n\nfunc F(value int) {\n  {\n    value := 1\n  }\n}\n",
    "r/a.go",
    "package r\n\nfunc value() {}\n\nfunc (\n",
    "r/b.go",
    "package r\n\nfunc F() {\n  value := 1\n}\n",
  }
  subject := fixture_new(t, memory, format_state, command, directory, seed)
  for _, path := range []string{"shared", "shared/s", "shared/c", "shared/r"} {
    fixture_directory(t, subject, path)
  }
  for index := 0; index < len(sources); index += 2 {
    fixture_file(t, subject, "shared/"+sources[index], sources[index+1])
  }
  fixture_run(t, subject, []string{"harness", "shared"})
  for _, found := range []string{
    "shared/s/b.go:4:3: The variable",
    "shared/c/c.go:5:5: The variable",
  } {
    if !fixture_holds(fixture_written(subject), found) {
      t.Fatalf("a shadow run wrote no %q", found)
    }
  }
  for _, absent := range []string{"shared/s/e_test.go:", "shared/r/b.go:"} {
    if fixture_holds(fixture_written(subject), absent) {
      t.Fatalf("a shadow run judged %s", absent)
    }
  }
}

// A function calling itself, a cycle across two files, a method call, and a package the
// configuration opts out of the recursion ban, all but one file of it opted back in.
func fixture_recursed(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  sources := []string{
    "a/a.go",
    "package a\n\nfunc F() {\n  F()\n}\n",
    "b/b.go",
    "package b\n\nfunc F() {\n  G()\n}\n",
    "b/c.go",
    "package b\n\nfunc G() {\n  go F()\n}\n",
    "m/m.go",
    "package m\n\ntype T int\n\nfunc (value T) F() {\n  value.F()\n}\n",
    "x/x.go",
    "package x\n\nfunc F() {\n  F()\n}\n",
    "x/y.go",
    "package x\n\nfunc G() {\n  G()\n}\n",
  }
  subject := fixture_new(t, memory, format_state, command, directory, seed)
  for _, path := range []string{"shared", "shared/a", "shared/b", "shared/m", "shared/x"} {
    fixture_directory(t, subject, path)
  }
  for index := 0; index < len(sources); index += 2 {
    fixture_file(t, subject, "shared/"+sources[index], sources[index+1])
  }
  shared := strings_replace(CONFIGURATION, `"shared_component":"s"`, `"shared_component":"shared"`)
  exempt := strings_replace(
    shared,
    `"opt_out_recursion_ban":[]`,
    `"opt_out_recursion_ban":["/shared/x/**/","!shared/x/y.go","shared/*/z\\*.go"]`,
  )
  fixture_file(t, subject, harness.CONFIGURATION_PATH, exempt)
  fixture_run(t, subject, []string{"harness", "shared"})
  for _, found := range []string{
    "shared/a/a.go:4:3: The function F recurses.",
    "shared/b/c.go:4:6: The functions recurse in a cycle. Write a loop instead. F → G → F",
    "shared/x/y.go:4:3: The function G recurses.",
  } {
    if !fixture_holds(fixture_written(subject), found) {
      t.Fatalf("a recursion run wrote no %q", found)
    }
  }
  for _, absent := range []string{"shared/m/m.go:", "shared/x/x.go:"} {
    if fixture_holds(fixture_written(subject), absent) {
      t.Fatalf("a recursion run judged %s", absent)
    }
  }
}

// A tree breaking the component layout once each way: a default directory of another package, an
// impure dependency, an instrumentation one, an impure stdlib import, a binary importing another
// and declaring two entry points, a nested module and a workspace, then a tree of no go.mod.
func fixture_laid(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  sources := []string{
    "shared/port/o.go",
    "package port\n",
    "shared/port/default/d.go",
    "package other\n",
    "shared/probe/p.go",
    "package probe\n",
    "shared/probe/default/d.go",
    "package probe\n",
    "shared/user/u.go",
    "package user\n\nimport \"local/fixture/shared/port/default\"\n",
    "shared/watch/w.go",
    "package watch\n\nimport \"local/fixture/shared/probe/default\"\n",
    "shared/lib/l.go",
    "package lib\n\nimport \"runtime\"\n",
    "shared/deep/go.mod",
    "module deep\n",
    "shared/deep/go.work",
    "go 1.26\n",
    "tool/main.go",
    "package main\n\nfunc main() {}\n",
    "tool/pkg/p.go",
    "package pkg\n\nimport \"local/fixture/other/pkg\"\n\nfunc Main() {}\n",
    "tool/pkg/q.go",
    "package pkg\n\nfunc Main() {}\n",
    "tool/lib/l.go",
    "package lib\n",
    "bare/main.go",
    "package main\n\nfunc main() {}\n",
  }
  subject := fixture_new(t, memory, format_state, command, directory, seed)
  made := map[string]bool{}
  for index := 0; index < len(sources); index += 2 {
    path := sources[index]
    for cut := range len(path) {
      if path[cut] != '/' {
        continue
      }
      if made[path[:cut]] {
        continue
      }
      made[path[:cut]] = true
      fixture_directory(t, subject, path[:cut])
    }
    fixture_file(t, subject, path, sources[index+1])
  }
  shared := strings_replace(CONFIGURATION, `"shared_component":"s"`, `"shared_component":"shared"`)
  instrumented := strings_replace(
    shared,
    `"instrumentation_packages":[]`,
    `"instrumentation_packages":["shared/probe/**"]`,
  )
  fixture_file(t, subject, harness.CONFIGURATION_PATH, instrumented)
  fixture_run(t, subject, []string{"harness", "."})
  for _, found := range []string{
    "shared/port/default/d.go:1:9: The default package declares",
    "shared/user/u.go:3:8: The dependency",
    "shared/lib/l.go:3:8: The stdlib import \"runtime\" is impure.",
    "tool/pkg/p.go:3:8: The binary component",
    "shared/deep/go.mod: A nested go.mod",
    "shared/deep/go.work: A go.work",
    "tool: The binary component \"local/fixture/tool\" declares more than one func Main",
    "tool/lib/l.go:1:1: Move lib -> tool/pkg/lib.",
    "bare: The binary component \"local/fixture/bare\" declares no func Main",
  } {
    if !fixture_holds(fixture_written(subject), found) {
      t.Fatalf("a layout run wrote no %q", found)
    }
  }
  if fixture_holds(fixture_written(subject), "shared/watch/w.go:") {
    t.Fatalf("a layout run judged the instrumentation import")
  }
  subject = fixture_new(t, memory, format_state, command, directory, seed)
  fixture_remove(t, subject, harness.MODULE_PATH)
  fixture_directory(t, subject, "lone")
  fixture_file(t, subject, "lone/a.go", CLEAN)
  fixture_run(t, subject, []string{"harness", "lone"})
  fixture_states(t, subject, "go.mod: There is no go.mod at the repository root.")
}

// Takes the file at the path out of the tree, the way a host that never wrote it would hold none.
func fixture_remove(t *testing.T, subject *fixture, path string) {
  t.Helper()
  for index := range subject.Simulation.Nodes {
    node := &subject.Simulation.Nodes[index]
    if !node.Used {
      continue
    }
    if string(node.Name[:node.Name_Count]) != path {
      continue
    }
    node.Used = false
    return
  }
  t.Fatalf("no node holds %s", path)
}

// SURVEY_NODE_COUNT holds the surveyed tree: four directories, two Go files, harness.json, go.mod,
// and the root the simulator keeps at the first node.
const SURVEY_NODE_COUNT = 10

// SURVEY_CONTENTS_BYTES holds a harness.json of an entry as wide as a path and one wider than a
// quote, with room to spare.
const SURVEY_CONTENTS_BYTES = 4 * filepath.PATH_SIZE_MAXIMUM

// SURVEY_NAME_BYTES is one directory name of the file whose path runs past the quote a finding
// holds.
const SURVEY_NAME_BYTES = 250

// A tree the survey judges each harness.json path entry of: entries of each kind the slash rule
// states, one of no literal, one of two bytes, one as wide as a path, and a file in the longest
// list whose path runs past the quote a finding holds.
func fixture_surveyed(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  subject := fixture_sized(
    t,
    memory,
    format_state,
    command,
    directory,
    seed,
    SURVEY_NODE_COUNT,
    SURVEY_CONTENTS_BYTES,
  )
  // An entry as wide as a path states a line past what an ordinary stream holds.
  subject.Output = nbio.Stream_Memory{Memory: make([]byte, WIDE_OUTPUT_BYTES)}
  subject.Runner.Output = nbio.Memory_To_Stream(&subject.Output)
  name := make([]byte, SURVEY_NAME_BYTES)
  for index := range name {
    name[index] = 'n'
  }
  deep := "lib/" + string(name) + "/" + string(name)
  for _, path := range []string{"lib", "lib/a", "lib/" + string(name), deep} {
    fixture_directory(t, subject, path)
  }
  fixture_file(t, subject, "lib/a/a.go", MAIN)
  fixture_file(t, subject, deep+"/"+string(name)+".go", MAIN)
  fixture_file(t, subject, harness.CONFIGURATION_PATH, survey_configuration(deep+"/"+string(name)))
  fixture_run(t, subject, []string{"harness", "lib"})
  for _, found := range []string{
    "The ignore entry \"lib\" names a directory. Add a trailing slash.",
    "The ignore entry \"lib/a/a.go/\" names a file. Remove the trailing slash.",
    "The instrumentation_packages entry \"lib/a/a.go\" names a file. Only ignore",
  } {
    if !fixture_holds(fixture_written(subject), found) {
      t.Fatalf("a survey wrote no %q", found)
    }
  }
  fixture_run(t, subject, []string{"harness", "lib", "-json"})
  if !fixture_holds(fixture_written(subject), `"rule":"slashes"`) {
    t.Fatal("a survey wrote no finding as flat JSON")
  }
}

// A harness.json whose path lists hold each entry the survey judges, the long file among them.
func survey_configuration(long string) (source string) {
  wide := make([]byte, filepath.PATH_SIZE_MAXIMUM)
  for index := range wide {
    wide[index] = 'z'
  }
  source = strings_replace(
    CONFIGURATION,
    `"ignore":[]`,
    `"ignore":["lib","lib/a/a.go/","lib/nowhere","lib/a/a.go","lib/a/","/","ab","lib/*","`+
      string(wide)+
      `"]`,
  )
  source = strings_replace(
    source,
    `"instrumentation_packages":[]`,
    `"instrumentation_packages":["lib/a/a.go"]`,
  )
  return strings_replace(
    source,
    `"opt_out_assertion_mandate_packages":[]`,
    `"opt_out_assertion_mandate_packages":["`+long+`.go"]`,
  )
}

// A harness.json whose assertion opt-out repeats an instrumentation entry.
func configuration_repeated() (source string) {
  source = strings_replace(
    CONFIGURATION,
    `"instrumentation_packages":[]`,
    `"instrumentation_packages":["a/"]`,
  )
  source = strings_replace(
    source,
    `"opt_out_assertion_mandate_packages":[]`,
    `"opt_out_assertion_mandate_packages":["a/"]`,
  )
  return source
}

// One harness.json of each fault the decoder states.
func fixture_configurations() (sources []string) {
  opening := `{"shared_component":"`
  // One string past the value table, in a document that decodes whole.
  full := `{"ignore":[""`
  for range configuration.VALUE_COUNT_MAXIMUM {
    full += `,""`
  }
  full += `]}`
  oversized := make([]byte, configuration.SOURCE_STORAGE_SIZE)
  for index := range oversized {
    oversized[index] = ' '
  }
  words := strings_replace(CONFIGURATION, `{"util":[]}`, "{}")
  // A string the widest source leaves open: the decoder stops one past its last byte.
  widest := opening + string(oversized[len(opening)+1:])
  sources = append(sources, configuration_repeated())
  globs := []string{
    `[" "]`,
    `["` + BACKSLASH + `u2003"]`,
    `["! "]`,
    `["a?"]`,
    `["a["]`,
    `["a\\"]`,
    `["a\\/b"]`,
    `["!a","!b"]`,
  }
  for _, entries := range globs {
    sources = append(sources, strings_replace(CONFIGURATION, `"ignore":[]`, `"ignore":`+entries))
  }
  return append(
    sources,
    []string{
      "",
      "{",
      "[",
      `{"`,
      `{}`,
      `{"x":[]}`,
      `{"ignore":"x"}`,
      `{"ignore"]`,
      `{"ignore":[]]`,
      `{"ignore":[],"ignore":[]}`,
      `{"shared_component":1}`,
      CONFIGURATION + "x",
      opening + "\x01",
      opening + "\xff",
      opening + BACKSLASH,
      opening + BACKSLASH + "q",
      opening + BACKSLASH + "u00",
      opening + BACKSLASH + "u00g0",
      opening + BACKSLASH + "ud800\"",
      opening + BACKSLASH + "udc00\"",
      opening + BACKSLASH + "ud800" + BACKSLASH + "u0041\"",
      `{"shared_component":"s"}`,
      `{"shared_component":"s","instrumentation_packages":[]}`,
      strings_replace(CONFIGURATION, `"ignore":[],`, ""),
      strings_replace(CONFIGURATION, `"opt_out_assertion_mandate_packages":[],`, ""),
      strings_replace(CONFIGURATION, `"opt_out_interface_ban":[],`, ""),
      strings_replace(CONFIGURATION, `"shared_component":"s"`, `"shared_component":""`),
      words,
      string(oversized),
      widest,
      full,
    }...,
  )
}

// Replaces the first text in the source with another.
func strings_replace(source string, old string, new string) (replaced string) {
  for index := 0; index+len(old) < len(source)+1; index++ {
    if source[index:index+len(old)] == old {
      return source[:index] + new + source[index+len(old):]
    }
  }
  return source
}

// Makes a symbolic link at the path, pointing at the target: a file holding the target, turned
// into the link the host would make of it. The node is the one of the child's name whose parent
// holds the name of the directory above it.
func fixture_link(t *testing.T, subject *fixture, path string, target string) {
  t.Helper()
  fixture_file(t, subject, path, target)
  node := fixture_node(t, subject, path)
  node.Mode = node.Mode&^nbio.FILE_MODE_TYPE | nbio.FILE_MODE_SYMBOLIC_LINK
}

// The node of the child's name whose parent holds the name of the directory above it.
func fixture_node(t *testing.T, subject *fixture, path string) (found *nbio.Sim_Node) {
  t.Helper()
  split, opening := 0, 0
  for index := range path {
    if path[index] == '/' {
      opening, split = split, index
    }
  }
  directory := path[opening:split]
  if opening > 0 {
    directory = path[opening+1 : split]
  }
  nodes := subject.Simulation.Nodes
  for index := range nodes {
    node := &nodes[index]
    if !node.Used {
      continue
    }
    if string(node.Name[:node.Name_Count]) != path[split+1:] {
      continue
    }
    parent := nodes[node.Parent]
    if string(parent.Name[:parent.Name_Count]) != directory {
      continue
    }
    return node
  }
  t.Fatalf("no node holds %s", path)
  return nil
}

// A file another writer fixes while the run checks it: the read the fix step makes states it
// clean, thus the run writes nothing over it. A contract test of one line parses whole, thus the
// run's own line index states one line.
func fixture_raced(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  subject := fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "raced")
  fixture_file(t, subject, "raced/a.go", UNFORMATTED)
  fixture_file(t, subject, "raced/"+requirements.MARKDOWN_NAME, CONTRACT_MARKDOWN)
  fixture_file(t, subject, "raced/"+requirements.TEST_NAME, "package a")
  // A fix would end the test in a newline, thus the run that reads it whole fixes nothing.
  fixture_run(t, subject, []string{"harness", "raced"})
  node := fixture_node(t, subject, "raced/a.go")
  subject.Runner.Spread.Submit = func(lane harness.Lane_Handle) { raced_submit(node, lane) }
  fixture_run(t, subject, []string{"harness", "raced", "-fix"})
  fixture_silent(t, subject, "raced/a.go")
  held := string(node.Contents[:node.Contents_Count])
  if held != CLEAN {
    t.Fatalf("a run wrote over a file another writer fixed, leaving %q", held)
  }
}

// Checks one lane, then fixes every file the lane may hold, the way another writer would.
func raced_submit(node *nbio.Sim_Node, lane harness.Lane_Handle) {
  harness.Check_Lane(lane)
  node.Contents_Count = copy(node.Contents, CLEAN)
}

// A host that states the entries of a directory in no order of its own: the walk sorts what it
// collected, thus a run states its lines in path order whichever order the pass read them in.
func fixture_shuffled(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  subject := fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "mixed")
  fixture_file(t, subject, "mixed/a.go", UNFORMATTED)
  fixture_file(t, subject, "mixed/b.go", UNFORMATTED)
  fixture_file(t, subject, "mixed/c.go", UNFORMATTED)
  fixture_file(t, subject, "mixed/m.go", MAIN)
  fixture_directory(t, subject, "mixed/sub")
  fixture_file(t, subject, "mixed/sub/a.go", UNFORMATTED)
  fixture_file(t, subject, "mixed/sub/m.go", MAIN)
  reversed := Fault_Storage{
    Inner: subject.Loop.Storage,
    Loop:  subject.Loop,
    Kind:  FAULT_REVERSED,
  }
  fault_wire(subject, &reversed)
  fixture_run(t, subject, []string{"harness", "mixed"})
  fixture_wrote(
    t,
    subject,
    "mixed/a.go: not formatted\n"+
      "mixed/b.go: not formatted\n"+
      "mixed/c.go: not formatted\n"+
      "mixed/sub/a.go: not formatted\n",
  )
}

// A close a host answers as failed: the descriptor is gone either way, thus the run states the
// step that failed rather than stands on a file it never finished with.
func fixture_shut(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  for skip := range CLOSE_SKIP_COUNT {
    subject := fixture_new(t, memory, format_state, command, directory, seed)
    fixture_directory(t, subject, "shut")
    fixture_file(t, subject, "shut/a.go", UNFORMATTED)
    shuts := Fault_Storage{
      Inner: subject.Loop.Storage,
      Loop:  subject.Loop,
      Kind:  FAULT_CLOSE,
      Skip:  skip,
    }
    fault_wire(subject, &shuts)
    fixture_run(t, subject, []string{"harness", "shut"})
    fixture_failed(t, subject, close_step(skip))
  }
  // The close a fix makes stands past the two a check makes, thus a host that answers that one
  // as failed states the fix step rather than the close a check stood on.
  subject := fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "shut")
  fixture_file(t, subject, "shut/a.go", UNFORMATTED)
  fixed := Fault_Storage{
    Inner: subject.Loop.Storage,
    Loop:  subject.Loop,
    Kind:  FAULT_CLOSE,
    Skip:  CLOSE_SKIP_COUNT,
  }
  fault_wire(subject, &fixed)
  fixture_run(t, subject, []string{"harness", "shut", "-fix"})
  fixture_failed(t, subject, "fix")
}

// CLOSE_SKIP_COUNT is how many closes of one package a sweep answers as failed in turn: the
// directory the walk opened and the file the declaration pass read, which the check recalls.
const CLOSE_SKIP_COUNT = 2

// Names the step one close stands in: the walk holds the directory it opened, and every close
// past it holds a file the run read.
func close_step(skip int) (step string) {
  if skip == 0 {
    return "walk"
  }
  return "close"
}

// The endings and the letters a host writes past what this toolchain spells: a heading line the
// file ends on with a carriage return and no newline, and a heading word of a letter outside
// ASCII, whose upper case mapping is the one a test name would carry.
func fixture_spelled(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  subject := fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "ended")
  fixture_file(t, subject, "ended/a.go", CLEAN)
  fixture_file(t, subject, "ended/"+requirements.MARKDOWN_NAME, "\n# A\r")
  fixture_run(t, subject, []string{"harness", "ended"})
  fixture_states(t, subject, "There is no blank line after the heading \"A\"")
  subject = fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "spelled")
  fixture_file(t, subject, "spelled/a.go", CLEAN)
  fixture_file(t, subject, "spelled/"+requirements.MARKDOWN_NAME, "\n# ä\n\nBody.\n")
  fixture_run(t, subject, []string{"harness", "spelled"})
  fixture_states(t, subject, "is not spelled with ASCII letters and digits")
}

// The declarations a test file states past what one leaf needs: a method, whose receiver stands
// before the name, and one declaration per slot of the table and one more.
func fixture_declared(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  subject := fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "method")
  fixture_file(t, subject, "method/a.go", CLEAN)
  fixture_file(t, subject, "method/"+requirements.MARKDOWN_NAME, CONTRACT_MARKDOWN)
  fixture_file(
    t,
    subject,
    "method/"+requirements.TEST_NAME,
    "package a\n\nfunc Test_Contract() {}\n\ntype T int\n\nfunc (t T) M() {}\n",
  )
  fixture_run(t, subject, []string{"harness", "method"})
  fixture_passed(t, subject)
  // One declaration per slot and one more: the pass stops at the slot the table holds
  // last, thus the file states its leaves rather than a slot the store never held.
  crowded := fixture_declarations()
  subject = fixture_sized(
    t,
    memory,
    format_state,
    command,
    directory,
    seed,
    TALL_NODE_COUNT,
    len(crowded)+1,
  )
  fixture_directory(t, subject, "crowd")
  fixture_file(t, subject, "crowd/a.go", CLEAN)
  fixture_file(t, subject, "crowd/"+requirements.MARKDOWN_NAME, CONTRACT_MARKDOWN)
  fixture_file(t, subject, "crowd/"+requirements.TEST_NAME, crowded)
  fixture_run(t, subject, []string{"harness", "crowd"})
  fixture_passed(t, subject)
}

// One test file holding one declaration per slot of the declaration table and one more. The
// declarations are values, since a value states its name in the node the pass reads rather than
// in a child of it, and the pass stops at the slot either way.
func fixture_declarations() (source string) {
  body := make([]byte, 0, DECLARATION_FILE_BYTES)
  body = append(body, "package a\n\nfunc Test_Contract() {}\n"...)
  // The names stand alike, since the pass reads what each slot declares rather than what
  // a compiler would make of two declarations of one name.
  for range requirements.HEADING_COUNT_MAXIMUM {
    body = append(body, "\nvar v int\n"...)
  }
  return string(body)
}

// DECLARATION_FILE_BYTES holds that file: one line per slot, and the package clause.
const DECLARATION_FILE_BYTES = (requirements.HEADING_COUNT_MAXIMUM + 1) * 32

// A stream that takes part of one diagnostic line: shared/std/sim/nbio states a short write as an
// error of its own, thus the run fails step report rather than leaves a host reading a line cut
// at the byte the stream stopped on.
func fixture_short_line(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
  taken int,
) {
  t.Helper()
  subject := fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "short")
  fixture_file(t, subject, "short/a.go", UNFORMATTED)
  fixture_file(t, subject, "short/m.go", MAIN)
  cut := short_stream{Inner: subject.Runner.Output, Short: taken}
  subject.Runner.Output = nbio.Stream{State: &cut, Procedure: short_stream_write}
  fixture_run(t, subject, []string{"harness", "short"})
  fixture_failed(t, subject, "report")
}

// Short_stream takes part of the next write and the whole of every write after it, the way a
// pipe whose buffer is nearly full does.
type short_stream struct {
  // Inner is the stream the cut write reaches.
  Inner nbio.Stream
  // Short is how many bytes the cut write takes.
  Short int
  // Skip is how many writes stand whole before the one the stream cuts.
  Skip int
  // Spent records that the cut already happened, thus one run meets it once.
  Spent bool
}

// Hands the writes it skips on whole, cuts the next one down, then hands every write after it
// straight on.
func short_stream_write(
  state nbio.State,
  completion *nbio.Completion,
  mode nbio.Stream_Mode,
  buffer []byte,
  offset int64,
  whence nbio.Seek_From,
  callback nbio.Stream_Callback,
) {
  stream := state.(*short_stream)
  if stream.Skip > 0 {
    stream.Skip--
    stream.Inner.Procedure(
      stream.Inner.State, completion, mode, buffer, offset, whence, callback,
    )
    return
  }
  if !stream.Spent {
    stream.Spent = true
    buffer = buffer[:min(stream.Short, len(buffer))]
  }
  stream.Inner.Procedure(
    stream.Inner.State, completion, mode, buffer, offset, whence, callback,
  )
}

// A package that breaks nothing and owes a contract it holds: the run states that it passed, as
// a line and as an object, and names no diagnostic of its own.
func fixture_clean(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  subject := fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "clean")
  fixture_file(t, subject, "clean/a.go", CLEAN)
  fixture_file(t, subject, "clean/REQUIREMENTS.md", "\n# Contract\n\nIt holds.\n")
  fixture_file(
    t, subject, "clean/requirements_test.go", "package a\n\nfunc Test_Contract() {}\n",
  )
  fixture_file(t, subject, "clean/b.go", INDENTED)
  // A child directory in the first tree a run walks: the walk stands before any read,
  // thus the read path it states is the one the root opened with.
  fixture_directory(t, subject, "clean/held")
  fixture_file(t, subject, "clean/held/c.go", MAIN)
  fixture_run(t, subject, []string{"harness", "clean"})
  fixture_passed(t, subject)
  // The same package as json: a summary is one object naming the pass and the
  // seconds, thus a host reads it by its shape rather than by its words.
  fixture_run(t, subject, []string{"harness", "clean", "-json"})
  fixture_opened(t, subject, `{"passed":true,"elapsed_sec":"`)
  fixture_passed(t, subject)
}

// One run states nothing of the given text: a file the walk passes over names no diagnostic, and
// a state that it does would stand unnoticed beside the diagnostics the run does write.
func fixture_silent(t *testing.T, subject *fixture, held string) {
  t.Helper()
  written := fixture_written(subject)
  if bool(strings.Contains(strings.Text(written), strings.Text(held))) {
    t.Fatalf("a run wrote %q, which states %q", written, held)
  }
}

// A command line shared/std/cli refuses: the run names the program, states the error, and exits
// failed, thus a host reads which command line was wrong rather than a walk that never ran.
func fixture_refused(t *testing.T, subject *fixture) {
  t.Helper()
  if subject.Runner.Status != harness.EXIT_FAILURE {
    t.Fatalf("a refused command line exited %d", subject.Runner.Status)
  }
  written := fixture_errors(subject)
  opening := min(len(written), len("harness: "))
  if string(written[:opening]) != "harness: " {
    t.Fatalf("a refused command line wrote %q", written)
  }
}

// One run opens its output with the given text: a state that stands anywhere would hold for an
// output a diagnostic line opened, thus the opening itself is stated.
func fixture_opened(t *testing.T, subject *fixture, held string) {
  t.Helper()
  written := fixture_written(subject)
  if len(written) < len(held) {
    t.Fatalf("a run wrote %q, which opens no %q", written, held)
  }
  if written[:len(held)] != held {
    t.Fatalf("a run wrote %q, which opens no %q", written, held)
  }
}

// One run states the given text and nothing else: a substring stands inside a line that grew a
// field or lost one, thus a whole output is stated where the shape itself is the property.
func fixture_wrote(t *testing.T, subject *fixture, held string) {
  t.Helper()
  written := fixture_findings(fixture_written(subject))
  if written != held {
    t.Fatalf("a run wrote %q, not %q", written, held)
  }
}

// One run states the given text among its diagnostics.
func fixture_states(t *testing.T, subject *fixture, held string) {
  t.Helper()
  written := fixture_written(subject)
  if !bool(strings.Contains(strings.Text(written), strings.Text(held))) {
    t.Fatalf("a run wrote %q, which states no %q", written, held)
  }
}

// A clean run states no diagnostic and no failure: its status passes and its error stream holds
// nothing, since a step that failed would name itself there.
func fixture_passed(t *testing.T, subject *fixture) {
  t.Helper()
  if subject.Runner.Status != harness.EXIT_SUCCESS {
    t.Fatalf("a clean package wrote %q", fixture_written(subject))
  }
  failures := fixture_errors(subject)
  if failures != "" {
    t.Fatalf("a clean package failed a step, writing %q", failures)
  }
  // A clean run states that it passed, in whichever form it writes, rather than stating
  // nothing at all.
  fixture_states(t, subject, "passed")
}

// The text one run wrote to its diagnostic stream.
func fixture_written(subject *fixture) (written string) {
  return string(subject.Output.Memory[:subject.Output.Cursor])
}

// The text one run wrote to its failure stream.
func fixture_errors(subject *fixture) (written string) {
  return string(subject.Errors.Memory[:subject.Errors.Cursor])
}

// A write that takes part of one fix: the run writes the rest in a second pass, thus the fixed
// file stands whole and the count of written bytes states its smallest values.
func fixture_short_write(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  for short := range SHORT_WRITE_COUNT {
    subject := fixture_new(t, memory, format_state, command, directory, seed)
    fixture_directory(t, subject, "short")
    fixture_file(t, subject, "short/a.go", UNFORMATTED)
    faults := Fault_Storage{
      Inner: subject.Loop.Storage,
      Loop:  subject.Loop,
      Kind:  FAULT_SHORT,
      Short: short + 1,
    }
    fault_wire(subject, &faults)
    fixture_run(t, subject, []string{"harness", "short", "-fix"})
    held := fixture_contents(t, subject, "short/a.go")
    if held != CLEAN {
      t.Fatalf("a cut write left %q", held)
    }
    // A disk that takes part of one write and then goes away: the count of bytes the
    // fix wrote rests at what it took, which a fix that finishes leaves at its whole
    // size and a fix that never starts leaves at none.
    subject = fixture_new(t, memory, format_state, command, directory, seed)
    fixture_directory(t, subject, "short")
    fixture_file(t, subject, "short/a.go", UNFORMATTED)
    gone := Fault_Storage{
      Inner: subject.Loop.Storage,
      Loop:  subject.Loop,
      Kind:  FAULT_SHORT_FAIL,
      Short: short + 1,
    }
    fault_wire(subject, &gone)
    fixture_run(t, subject, []string{"harness", "short", "-fix"})
    // The fix opens its file cut to nothing and writes the form over it, thus a disk
    // that goes away part way leaves the file holding what it wrote and no more.
    held = fixture_contents(t, subject, "short/a.go")
    if held != CLEAN[:short+1] {
      t.Fatalf("a fix that stopped part way left %q", held)
    }
    fixture_run(t, subject, []string{"harness", "short"})
    fixture_short_line(
      t, memory, format_state, command, directory, seed, short+1,
    )
    // A read that hands back part of one file: the run reads the rest before it
    // checks, thus the check states the whole source rather than its first bytes.
    subject = fixture_new(t, memory, format_state, command, directory, seed)
    fixture_directory(t, subject, "short")
    fixture_file(t, subject, "short/a.go", UNFORMATTED)
    reads := Fault_Storage{
      Inner: subject.Loop.Storage,
      Loop:  subject.Loop,
      Kind:  FAULT_SHORT_READ,
      Short: short + 1,
    }
    fault_wire(subject, &reads)
    fixture_run(t, subject, []string{"harness", "short", "-fix"})
    held = fixture_contents(t, subject, "short/a.go")
    if held != CLEAN {
      t.Fatalf("a cut read left %q", held)
    }
  }
}

// A fix that stands in the canonical form: a second run over the fixed tree states nothing, thus
// the form one run writes is the form the next one reads back.
func fixture_idempotent(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  subject := fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "twice")
  fixture_directory(t, subject, "twice/nested")
  fixture_file(t, subject, "twice/a.go", UNFORMATTED)
  fixture_file(t, subject, "twice/b.go", INDENTED)
  fixture_file(t, subject, "twice/nested/c.go", UNFORMATTED)
  fixture_file(t, subject, "twice/"+requirements.MARKDOWN_NAME, CONTRACT_MARKDOWN)
  fixture_file(t, subject, "twice/"+requirements.TEST_NAME, CONTRACT_TEST)
  fixture_file(t, subject, "twice/nested/"+requirements.MARKDOWN_NAME, CONTRACT_MARKDOWN)
  fixture_file(t, subject, "twice/nested/"+requirements.TEST_NAME, CONTRACT_TEST)
  fixture_run(t, subject, []string{"harness", "twice", "-fix"})
  fixture_run(t, subject, []string{"harness", "twice"})
  fixture_passed(t, subject)
  fixed := fixture_contents(t, subject, "twice/nested/c.go")
  if fixed != CLEAN {
    t.Fatalf("a fix of a nested file left %q", fixed)
  }
  // A file already in the canonical form is not written again, thus its bytes stand where
  // they stood rather than pass through a rewrite that states the same form.
  held := fixture_contents(t, subject, "twice/b.go")
  if held != INDENTED {
    t.Fatalf("a run rewrote a file that held its form, leaving %q", held)
  }
}

// TABBED holds the canonical form with tabs where it opens a line with two spaces.
const TABBED = "package a\n\nfunc g() (value int) {\n\treturn 1\n}\n"

// MESSAGE_UNFORMATTED is what a line says of a file that does not stand in the canonical form.
const MESSAGE_UNFORMATTED = "not formatted"

// SHAPE_NODE_COUNT is how many entries one drawn tree holds: enough for a shape a fixture would
// not think to write, and few enough that every seed can afford two runs over one. Twice this
// count states the same gaps and spends half a second more of the ten a package is allowed.
const SHAPE_NODE_COUNT = 12

// SHAPE_SOURCE_COUNT is how many sources one entry draws from: a clean file, an unformatted one,
// and one the parser refuses.
const SHAPE_SOURCE_COUNT = 3

// SHAPE_DIRECTORY_CHANCE is how often a drawn entry is a directory rather than a Go file.
const SHAPE_DIRECTORY_CHANCE = 3

// A tree drawn from the seed: its shape, its depth, and what each file holds all come from the
// same draw, thus every seed checks a tree no fixture wrote and the run over it still ends,
// closes what it opened, and leaves nothing in flight.
func fixture_shaped(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  subject := fixture_new(t, memory, format_state, command, directory, seed)
  generator := prng.New(prng.Seed(seed))
  sources := [SHAPE_SOURCE_COUNT]string{CLEAN, UNFORMATTED, REFUSED}
  path := "shape"
  fixture_directory(t, subject, path)
  held := path
  for index := range SHAPE_NODE_COUNT {
    name := fixture_name(index)
    if prng.Xoshiro_Below(&generator, SHAPE_DIRECTORY_CHANCE) == 0 {
      held += "/" + name
      fixture_directory(t, subject, held)
      continue
    }
    drawn := prng.Xoshiro_Below(&generator, SHAPE_SOURCE_COUNT)
    fixture_file(t, subject, held+"/"+name+GO_SUFFIX_TEXT, sources[drawn])
  }
  fixture_run(t, subject, []string{"harness", path, "-fix"})
  // The run after it states no file out of form: a fix settles whatever shape the draw
  // made, thus what stays is what no fix can take.
  fixture_run(t, subject, []string{"harness", path})
  written := fixture_written(subject)
  if bool(strings.Contains(strings.Text(written), strings.Text(MESSAGE_UNFORMATTED))) {
    t.Fatalf("a fix over a drawn tree left a file out of form: %q", written)
  }
}

// The simulator fabricates a small tree of its own from every seed, of a handful of nodes: files,
// directories, and now and then a symbolic link. The walk stands over that tree as it stands over
// one a fixture wrote, thus every seed states a shape no fixture thought to write.
func fixture_generated(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  subject := fixture_new(t, memory, format_state, command, directory, seed)
  // One Go file of its own stands in that tree, thus the run checks and states a path the
  // walk built from the directory the command line named, which is a name of one byte.
  fixture_file(t, subject, "a.go", UNFORMATTED)
  // One seed in three states flat JSON, one fixes what it finds, and one states text, thus
  // every form a run takes stands over a tree no fixture wrote.
  if seed%GENERATED_MODE_COUNT == 0 {
    fixture_run(t, subject, []string{"harness", ".", "-json"})
    fixture_states(t, subject, `"path":"./a.go"`)
    return
  }
  if seed%GENERATED_MODE_COUNT == 1 {
    // The contract of that root stands too, thus a fix leaves a tree the run after it
    // passes rather than one that still owes a contract.
    fixture_file(t, subject, requirements.MARKDOWN_NAME, CONTRACT_MARKDOWN)
    fixture_file(t, subject, requirements.TEST_NAME, CONTRACT_TEST)
    fixture_run(t, subject, []string{"harness", ".", "-fix"})
    fixture_passed(t, subject)
    held := fixture_contents(t, subject, "a.go")
    if held != CLEAN {
      t.Fatalf("a fix inside a generated tree left %q", held)
    }
    return
  }
  fixture_run(t, subject, []string{"harness", "."})
  fixture_states(t, subject, "./a.go: not formatted")
}

// GENERATED_MODE_COUNT is how many forms a run takes over the tree the simulator fabricates.
const GENERATED_MODE_COUNT = 3

// Two runs over one tree, at seeds that order its retirements differently: both state the same
// lines in the same order, since a walk sorts its paths before any line is written.
func fixture_deterministic(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  first := fixture_ordered(t, memory, format_state, command, directory, seed)
  second := fixture_ordered(t, memory, format_state, command, directory, seed+1)
  // The tree holds findings, thus two seeds agree on them rather than on nothing.
  if fixture_lines(first) != ORDER_FINDING_COUNT {
    t.Fatalf(
      "a tree of %d findings stated %d lines: %q",
      ORDER_FINDING_COUNT,
      fixture_lines(first),
      first,
    )
  }
  if first != second {
    t.Fatalf("one tree stated %q at one seed and %q at the next", first, second)
  }
}

// One file past the source bound bars every rename of the run, since a rename it cannot
// finish breaks the build.
func fixture_rename_barred(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  wide := fixture_wide()
  subject := fixture_sized(
    t,
    memory,
    format_state,
    command,
    directory,
    seed,
    TALL_NODE_COUNT,
    len(wide)+1,
  )
  fixture_directory(t, subject, "names")
  fixture_file(t, subject, "names/a.go", DECLARATION)
  fixture_file(t, subject, "names/b.go", REFERENCE)
  fixture_file(t, subject, "names/c.go", wide)
  fixture_file(t, subject, "names/"+requirements.MARKDOWN_NAME, CONTRACT_MARKDOWN)
  fixture_file(t, subject, "names/"+requirements.TEST_NAME, CONTRACT_TEST)
  fixture_run(t, subject, []string{"harness", "names", "-fix"})
  // The run reaches the check, thus the name stands where it stands because the run barred
  // the rename rather than because it never got that far.
  fixture_states(t, subject, "exceeds source size")
  held := fixture_contents(t, subject, "names/a.go")
  if held != DECLARATION {
    t.Fatalf("a run holding an oversized file renamed, leaving %q", held)
  }
}

// One name declared in one file and spelled in another: a run fixes both, since a rename that
// misses one file breaks every file that spells it.
func fixture_renamed(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  subject := fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "names")
  fixture_file(t, subject, "names/a.go", DECLARATION)
  fixture_file(t, subject, "names/b.go", REFERENCE)
  fixture_file(t, subject, "names/"+requirements.MARKDOWN_NAME, CONTRACT_MARKDOWN)
  fixture_file(t, subject, "names/"+requirements.TEST_NAME, CONTRACT_TEST)
  fixture_run(t, subject, []string{"harness", "names", "-fix"})
  fixture_passed(t, subject)
  held := fixture_contents(t, subject, "names/a.go")
  if held != DECLARATION_RENAMED {
    t.Fatalf("a rename of a declaration left %q", held)
  }
  held = fixture_contents(t, subject, "names/b.go")
  if held != REFERENCE_RENAMED {
    t.Fatalf("a rename of a reference left %q", held)
  }
  // A name one package declares and another spells: the run declares every Go file under
  // the directory before it checks any, thus the rename reaches both packages.
  subject = fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "pair")
  fixture_directory(t, subject, "pair/one")
  fixture_directory(t, subject, "pair/two")
  fixture_file(t, subject, "pair/one/a.go", DECLARATION)
  fixture_file(t, subject, "pair/two/b.go", REFERENCE)
  fixture_run(t, subject, []string{"harness", "pair", "-fix"})
  held = fixture_contents(t, subject, "pair/two/b.go")
  if held != REFERENCE_RENAMED {
    t.Fatalf("a rename stopped at the package it was declared in, leaving %q", held)
  }
  // One file that owes both a rename and a form: its fix carries the name the run spells
  // everywhere and the form the printer states, rather than one of the two.
  subject = fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "both")
  fixture_file(t, subject, "both/a.go", DECLARATION)
  fixture_file(t, subject, "both/b.go", REFERENCE_UNFORMATTED)
  fixture_file(t, subject, "both/"+requirements.MARKDOWN_NAME, CONTRACT_MARKDOWN)
  fixture_file(t, subject, "both/"+requirements.TEST_NAME, CONTRACT_TEST)
  fixture_run(t, subject, []string{"harness", "both", "-fix"})
  fixture_passed(t, subject)
  held = fixture_contents(t, subject, "both/b.go")
  if held != REFERENCE_RENAMED {
    t.Fatalf("a file owing a rename and a form kept %q", held)
  }
  fixture_rename_barred(t, memory, format_state, command, directory, seed)
}

// DECLARATION holds one name a rename spells differently.
const DECLARATION = "package a\n\nvar someValue = 1\n"

// DECLARATION_RENAMED is that file after the rename.
const DECLARATION_RENAMED = "package a\n\nvar some_value = 1\n"

// REFERENCE spells that same name, thus one rename spans both files.
const REFERENCE = "package a\n\nvar held = someValue\n"

// LITERAL spells that same name inside a string, which no rename reaches.
const LITERAL = "package a\n\nvar text = \"someValue\"\n"

// UNSORTED holds two imports out of order, which the canonical form sorts.
const UNSORTED = "package a\n\nimport \"unsafe\"\nimport \"testing\"\n\nvar held testing.T\n"

// SORTED is that file with its imports in order.
const SORTED = "package a\n\nimport \"testing\"\nimport \"unsafe\"\n\nvar held testing.T\n"

// JOINED holds two statements on one line, joined by a semicolon.
const JOINED = "package a\n\nfunc f() {}\n\nfunc g() {\n  f(); f()\n}\n"

// SPLIT is that file in the canonical form, each statement on a line of its own.
const SPLIT = "package a\n\nfunc f() {}\n\nfunc g() {\n  f()\n  f()\n}\n"

// RAW holds a raw string whose own lines carry the space they were written with.
const RAW = "package a\n\nfunc e() (text string) {\n  return `line one\n  line two`\n}\n"

// TAGGED opens with a build tag, which states which hosts compile the file.
const TAGGED = "//go:build linux\n\npackage a\n\nfunc d() {}\n"

// COMMENTED spells that same name inside a comment, which no rename reaches either.
const COMMENTED = "package a\n\n// someValue is held.\nvar other = 2\n"

// REFERENCE_UNFORMATTED spells that name and stands outside the canonical form, thus its fix
// owes both a rename and a form.
const REFERENCE_UNFORMATTED = "package a\n\nvar  held = someValue\n"

// REFERENCE_RENAMED is that file after the rename.
const REFERENCE_RENAMED = "package a\n\nvar held = some_value\n"

// A rename spells identifiers, thus the same letters inside a string literal or a comment stand
// as the file wrote them while the declaration itself changes.
func fixture_literal(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  subject := fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "quotes")
  fixture_file(t, subject, "quotes/a.go", DECLARATION)
  fixture_file(t, subject, "quotes/b.go", LITERAL)
  fixture_file(t, subject, "quotes/c.go", COMMENTED)
  fixture_file(t, subject, "quotes/d.go", TAGGED)
  fixture_file(t, subject, "quotes/e.go", RAW)
  fixture_file(t, subject, "quotes/f.go", JOINED)
  fixture_file(t, subject, "quotes/g.go", UNSORTED)
  fixture_run(t, subject, []string{"harness", "quotes", "-fix"})
  held := fixture_contents(t, subject, "quotes/a.go")
  if held != DECLARATION_RENAMED {
    t.Fatalf("a rename left the declaration as %q", held)
  }
  // A build tag states which hosts compile the file, thus the form keeps it where it
  // stands rather than moves it under the package clause.
  // A raw string carries its own lines, thus the form leaves the space inside it where the
  // file wrote it rather than indents it with the code around it.
  // Imports stand in order in the canonical form, thus a file that spells them otherwise
  // is fixed to the same set in the order a reader can scan.
  held = fixture_contents(t, subject, "quotes/g.go")
  if held != SORTED {
    t.Fatalf("a form left imports out of order, leaving %q", held)
  }
  // Statements joined by a semicolon stand on lines of their own in the canonical form.
  held = fixture_contents(t, subject, "quotes/f.go")
  if held != SPLIT {
    t.Fatalf("a form kept statements on one line, leaving %q", held)
  }
  // The same program written with tabs: its fix is that file written with two spaces, thus
  // one form comes out of either way a host writes the same code.
  tabbed := fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, tabbed, "tabs")
  fixture_file(t, tabbed, "tabs/a.go", TABBED)
  fixture_run(t, tabbed, []string{"harness", "tabs", "-fix"})
  held = fixture_contents(t, tabbed, "tabs/a.go")
  if held != INDENTED {
    t.Fatalf("a fix of a tabbed file wrote %q", held)
  }
  held = fixture_contents(t, subject, "quotes/e.go")
  if held != RAW {
    t.Fatalf("a form reached inside a raw string, leaving %q", held)
  }
  held = fixture_contents(t, subject, "quotes/d.go")
  if held != TAGGED {
    t.Fatalf("a form moved a build tag, leaving %q", held)
  }
  held = fixture_contents(t, subject, "quotes/c.go")
  if held != COMMENTED {
    t.Fatalf("a rename reached inside a comment, leaving %q", held)
  }
  held = fixture_contents(t, subject, "quotes/b.go")
  if held != LITERAL {
    t.Fatalf("a rename reached inside a string literal, leaving %q", held)
  }
}

// One run over a package whose contract the caller states, giving back what it wrote.
func fixture_contract_findings(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
  markdown string,
) (written string) {
  t.Helper()
  subject := fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "endings")
  fixture_file(t, subject, "endings/a.go", CLEAN)
  fixture_file(t, subject, "endings/"+requirements.MARKDOWN_NAME, markdown)
  fixture_file(t, subject, "endings/"+requirements.TEST_NAME, CONTRACT_TEST)
  fixture_run(t, subject, []string{"harness", "endings"})
  return fixture_written(subject)
}

// The same text with a carriage return before every newline, as one host writes it.
func fixture_returned(text string) (returned string) {
  held := make([]byte, 0, len(text)*2)
  for index := range len(text) {
    if text[index] == '\n' {
      held = append(held, '\r')
    }
    held = append(held, text[index])
  }
  return string(held)
}

// One contract breaking every rule, written with each ending in turn, and without the newline
// that ends its last line: each states the same findings, thus no rule reads the ending.
func fixture_endings(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  // One contract breaking every rule, written with each ending in turn: both state the
  // same findings, thus no rule reads the ending rather than the line.
  plain := fixture_contract_findings(
    t,
    memory,
    format_state,
    command,
    directory,
    seed,
    FAULTY_CONTRACT,
  )
  returned := fixture_contract_findings(
    t,
    memory,
    format_state,
    command,
    directory,
    seed,
    fixture_returned(FAULTY_CONTRACT),
  )
  if plain != returned {
    t.Fatalf("one contract stated %q with one ending and %q with the other",
      plain, returned)
  }
  // The same contract without the newline that ends its last line: the text after the
  // final ending is a line of its own, thus both state the same findings.
  bare := fixture_contract_findings(
    t,
    memory,
    format_state,
    command,
    directory,
    seed,
    FAULTY_CONTRACT[:len(FAULTY_CONTRACT)-1],
  )
  if plain != bare {
    t.Fatalf("one contract stated %q ended and %q bare", plain, bare)
  }
}

// A contract test file the parser refuses: the check reads no declaration out of it, thus the
// run states the refusal alone and the leaves go unchecked.
func fixture_test_refused(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  subject := fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "broke")
  fixture_file(t, subject, "broke/a.go", CLEAN)
  fixture_file(t, subject, "broke/"+requirements.MARKDOWN_NAME, CONTRACT_MARKDOWN)
  fixture_file(t, subject, "broke/"+requirements.TEST_NAME, REFUSED)
  subject_crlf := fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject_crlf, "crlfmd")
  fixture_file(t, subject_crlf, "crlfmd/a.go", CLEAN)
  fixture_file(
    t,
    subject_crlf,
    "crlfmd/"+requirements.MARKDOWN_NAME,
    "\r\n# Contract\r\n\r\nIt holds.\r\n",
  )
  fixture_file(t, subject_crlf, "crlfmd/"+requirements.TEST_NAME, CONTRACT_TEST)
  // A contract one host wrote with two bytes for each ending states what it states on
  // every other, thus the lines it holds are the lines a reader sees.
  fixture_run(t, subject_crlf, []string{"harness", "crlfmd"})
  fixture_passed(t, subject_crlf)
  // The same file missing the blank line after its heading states that finding, thus the
  // ending it was written with hides no fault of its own.
  subject_crlf = fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject_crlf, "crlfmd")
  fixture_file(t, subject_crlf, "crlfmd/a.go", CLEAN)
  fixture_file(
    t,
    subject_crlf,
    "crlfmd/"+requirements.MARKDOWN_NAME,
    "\r\n# Contract\r\nIt holds.\r\n",
  )
  fixture_file(t, subject_crlf, "crlfmd/"+requirements.TEST_NAME, CONTRACT_TEST)
  fixture_run(t, subject_crlf, []string{"harness", "crlfmd"})
  fixture_states(t, subject_crlf, "There is no blank line after the heading")
  fixture_endings(t, memory, format_state, command, directory, seed)
  fixture_run(t, subject, []string{"harness", "broke"})
  fixture_states(t, subject, "broke/"+requirements.TEST_NAME+":3:9")
  // The leaves go unchecked, thus the refusal stands alone rather than beside a test the
  // check could not have matched against a file it never read declarations out of.
  written := fixture_written(subject)
  if fixture_lines(written) != 1 {
    t.Fatalf("a package whose test file is refused wrote %q", written)
  }
}

// A contract of two leaves whose test file declares them the other way round: the check names
// the test the first leaf owes, since the tests stand in leaf order at the top of the file.
func fixture_order(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  subject := fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "order")
  fixture_file(t, subject, "order/a.go", CLEAN)
  fixture_file(
    t,
    subject,
    "order/"+requirements.MARKDOWN_NAME,
    "\n# One\n\nBody.\n\n# Two\n\nBody.\n",
  )
  fixture_file(
    t,
    subject,
    "order/"+requirements.TEST_NAME,
    "package a\n\nfunc Test_Two() {}\n\nfunc Test_One() {}\n",
  )
  fixture_run(t, subject, []string{"harness", "order"})
  fixture_states(t, subject, "The file needs Test_One")
}

// A package under a third_party segment and one under an examples segment: each owes no
// contract, thus a walk that names the segment states nothing about either.
func fixture_exempt(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  subject := fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "vendor")
  fixture_directory(t, subject, "vendor/third_party")
  fixture_directory(t, subject, "vendor/third_party/held")
  fixture_file(t, subject, "vendor/third_party/held/a.go", CLEAN)
  fixture_directory(t, subject, "vendor/examples")
  fixture_directory(t, subject, "vendor/examples/shown")
  fixture_file(t, subject, "vendor/examples/shown/b.go", CLEAN)
  fixture_directory(t, subject, "vendor/root")
  fixture_file(t, subject, "vendor/root/main.go", MAIN)
  // A directory of no Go file is no package, thus the contract it carries states nothing.
  fixture_directory(t, subject, "vendor/notes")
  fixture_file(t, subject, "vendor/notes/"+requirements.MARKDOWN_NAME, CONTRACT_MARKDOWN)
  fixture_run(t, subject, []string{"harness", "vendor"})
  fixture_passed(t, subject)
  // The contract file is named exactly, since a disk that resolves a differently cased
  // name would hand back a file the run never asked for.
  subject = fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "cased")
  fixture_file(t, subject, "cased/a.go", CLEAN)
  fixture_file(t, subject, "cased/requirements.md", CONTRACT_MARKDOWN)
  fixture_run(t, subject, []string{"harness", "cased"})
  fixture_states(t, subject, "The package has no REQUIREMENTS.md.")
  // A file where the run asks for a directory: the pass over its entries states that it
  // holds none, thus the walk fails rather than checks the file the argument names.
  fixture_run(t, subject, []string{"harness", "cased/a.go"})
  fixture_failed(t, subject, "walk")
  // An argument of no bytes names nothing to walk, thus the run fails rather than joins
  // every path it states under a name it never read.
  fixture_run(t, subject, []string{"harness", ""})
  fixture_failed(t, subject, "walk")
  // A file of one byte where the run asks for a directory, and one of two: the line that
  // names the failed walk carries the shortest path a run states.
  fixture_file(t, subject, "x", CLEAN)
  fixture_run(t, subject, []string{"harness", "x"})
  fixture_failed(t, subject, "walk")
  fixture_file(t, subject, "xy", CLEAN)
  fixture_run(t, subject, []string{"harness", "xy"})
  fixture_failed(t, subject, "walk")
  // A trailing separator states the same directory, thus the paths it names carry one
  // separator rather than two.
  fixture_run(t, subject, []string{"harness", "cased/"})
  fixture_states(t, subject, "cased/REQUIREMENTS.md")
  // A name holding a byte below a space states half a path on a line of its own, thus the
  // walk fails rather than writes a line that breaks in two.
  subject = fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "weird")
  fixture_file(t, subject, "weird/a\nb.go", UNFORMATTED)
  fixture_run(t, subject, []string{"harness", "weird"})
  fixture_failed(t, subject, "walk")
  fixture_run(t, subject, []string{"harness", "we\nird"})
  fixture_failed(t, subject, "walk")
  // The argument is untrusted, thus the line that names the failed walk carries none of
  // its bytes rather than hands a terminal whatever a host typed.
  written := fixture_errors(subject)
  if written != "harness: failed to walk: the argument names no directory\n" {
    t.Fatalf("a refused argument wrote %q", written)
  }
}

// A file a check reads is untrusted like the names a walk reads: a byte no line can state stands
// as a question mark, a quote stands escaped in flat JSON, and the widest heading states a quote
// of the whole width one message carries.
func fixture_untrusted(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  var subject *fixture
  // Two headings of the whole width a heading holds, spelled the same: the message quotes
  // one of them, thus a quote states the widest span a message carries.
  subject = fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "widest")
  fixture_file(t, subject, "widest/a.go", CLEAN)
  fixture_file(t, subject, "widest/"+requirements.MARKDOWN_NAME, fixture_contract_twice())
  fixture_run(t, subject, []string{"harness", "widest"})
  fixture_states(t, subject, "is not unique")
  // The same message as json: it lands in a report, thus a report carries the widest
  // message a run writes rather than the short ones a Go file yields.
  fixture_run(t, subject, []string{"harness", "widest", "-json"})
  fixture_states(t, subject, "is not unique")
  // Headings of no bytes, of one, and of two, each spelled twice: a quote states the
  // smallest spans a message carries as well as the widest.
  for _, heading := range [SHORT_HEADING_COUNT]string{"", "A", "AB"} {
    subject = fixture_new(t, memory, format_state, command, directory, seed)
    fixture_directory(t, subject, "short")
    fixture_file(t, subject, "short/a.go", CLEAN)
    section := "\n# " + heading + "\n\nBody.\n"
    fixture_file(t, subject, "short/"+requirements.MARKDOWN_NAME, section+section)
    fixture_run(t, subject, []string{"harness", "short"})
  }
  subject = fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "hosts")
  fixture_file(t, subject, "hosts/bom.go", "\xef\xbb\xbf"+CLEAN)
  fixture_file(t, subject, "hosts/crlf.go", "package a\r\n\r\nfunc f() {}\r\n")
  fixture_file(t, subject, "hosts/tabbed.go", TABBED)
  fixture_file(t, subject, "hosts/bare.go", "package a\n\nfunc h() {}")
  fixture_file(t, subject, "hosts/wide.go", "package a\n\nvar caf\u00e9 = 1\n")
  // A byte order mark opens no package clause, a carriage return stands in no canonical
  // form, and a name outside ASCII is one the parser refuses: what a host writes is
  // untrusted like everything else a run reads.
  fixture_run(t, subject, []string{"harness", "hosts", "-json"})
  fixture_states(t, subject, `"path":"hosts/bom.go","line":1,"column":1,"tier":2`)
  fixture_states(t, subject, `"path":"hosts/crlf.go","line":0,"column":0,"tier":1`)
  fixture_states(t, subject, `"path":"hosts/tabbed.go","line":0,"column":0,"tier":1`)
  fixture_states(t, subject, `"path":"hosts/bare.go","line":0,"column":0,"tier":1`)
  fixture_states(t, subject, "Spell the name with ASCII letters, digits, and underscores.")
  // A file a check reads is untrusted too: a byte no line can state stands as a question
  // mark rather than reaching whatever reads the line.
  subject = fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "quoted")
  fixture_file(t, subject, "quoted/a.go", CLEAN)
  fixture_file(
    t,
    subject,
    "quoted/"+requirements.MARKDOWN_NAME,
    "\n# A\vB\n\nBody.\n\n# A\vB\n\nBody.\n",
  )
  fixture_run(t, subject, []string{"harness", "quoted"})
  fixture_states(t, subject, `The heading "A?B" is not unique.`)
  subject = fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "quote")
  fixture_file(t, subject, `quote/a".go`, UNFORMATTED)
  // A name holding a quote stands escaped in flat JSON, thus the object a run writes parses
  // as one object rather than breaks at the name.
  fixture_run(t, subject, []string{"harness", "quote", "-json"})
  fixture_states(t, subject, `"path":"quote/a\".go"`)
}

// GO_SUFFIX_TEXT names the suffix a Go file carries, which a fixture spells to size a name.
const GO_SUFFIX_TEXT = ".go"

// SHORT_HEADING_COUNT is how many short headings one sweep states: none, one byte, and two.
const SHORT_HEADING_COUNT = 3

// MAIN holds a package main test file, whose package owes no contract as a package main does,
// and which a shared component holds, as it holds no package main.
const MAIN = "package main_test\n\nfunc main() {}\n"

// A run that fixes what it can states what stays, and the run after it states the same: the fix
// settles the tree, thus nothing it wrote comes back as a finding.
func fixture_settled(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  subject := fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "settle")
  fixture_file(t, subject, "settle/a.go", UNFORMATTED)
  fixture_file(t, subject, "settle/b.go", REFUSED)
  fixture_file(t, subject, "settle/"+requirements.MARKDOWN_NAME, CONTRACT_MARKDOWN)
  fixture_file(t, subject, "settle/"+requirements.TEST_NAME, CONTRACT_TEST)
  fixture_run(t, subject, []string{"harness", "settle", "-fix"})
  // The refusal it cannot fix stands in what it wrote, thus the two runs state a finding
  // rather than both stating nothing.
  fixture_states(t, subject, "settle/b.go")
  fixed := fixture_written(subject)
  fixture_run(t, subject, []string{"harness", "settle"})
  stayed := fixture_written(subject)
  // Both flags at once: the fix takes what it can and the report states the rest as flat
  // JSON, which is how a run inside a pipeline takes both.
  fixture_run(t, subject, []string{"harness", "settle", "-fix", "-json"})
  fixture_states(t, subject, `"path":"settle/b.go"`)
  if fixed != stayed {
    t.Fatalf("a fix stated %q and the run after it stated %q", fixed, stayed)
  }
}

// Two runs over one tree, one stating text and one flat JSON: both state one line per
// diagnostic, thus neither mode drops a finding the other holds. One seed states this, since the
// order a walk reads a tree in cannot change how many lines it writes.
func fixture_modes(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  // One subject per run, since a memory stream holds what the run before it wrote.
  plain := fixture_moded(t, memory, format_state, command, directory, seed)
  flat := fixture_moded(t, memory, format_state, command, directory, seed, "-json")
  // The tree holds findings, thus both modes state them rather than agreeing on nothing.
  if plain != MODES_FINDING_COUNT {
    t.Fatalf("a tree of %d findings stated %d text lines", MODES_FINDING_COUNT, plain)
  }
  if plain != flat {
    t.Fatalf("a tree stated %d text lines and %d flat JSON lines", plain, flat)
  }
}

// One run over one tree, stating how many lines it wrote.
func fixture_moded(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
  arguments ...string,
) (count int) {
  t.Helper()
  subject := fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "modes")
  fixture_directory(t, subject, "modes/nested")
  fixture_file(t, subject, "modes/a.go", UNFORMATTED)
  fixture_file(t, subject, "modes/b.go", REFUSED)
  fixture_file(t, subject, "modes/nested/c.go", INTERFACE_DECLARATION)
  fixture_run(t, subject, append([]string{"harness", "modes"}, arguments...))
  return fixture_lines(fixture_written(subject))
}

// MODES_FINDING_COUNT is how many findings the tree both modes read holds: one per Go file, and
// one per package for the contract neither of its two packages carries.
const MODES_FINDING_COUNT = 5

// How many lines one run wrote.
func fixture_lines(written string) (count int) {
  opening := 0
  for index := range len(written) {
    if written[index] != '\n' {
      continue
    }
    if !fixture_elapsed(written[opening:index]) {
      count++
    }
    opening = index + 1
  }
  return count
}

// Cuts the elapsed seconds a run with a finding closes on off what it wrote.
func fixture_findings(written string) (findings string) {
  opening := len(written) - 1
  for opening > 0 {
    if written[opening-1] == '\n' {
      break
    }
    opening--
  }
  opening = max(opening, 0)
  if fixture_elapsed(written[opening:]) {
    return written[:opening]
  }
  return written
}

// Reports whether a line is the elapsed seconds a run with a finding closes on, which states no
// finding.
func fixture_elapsed(line string) (elapsed bool) {
  for _, opening := range []string{harness.ELAPSED_OPENING, `{"passed":false,`} {
    if len(line) < len(opening) {
      continue
    }
    if line[:len(opening)] == opening {
      return true
    }
  }
  return false
}

// ORDER_FINDING_COUNT is how many findings that tree holds: one per Go file, and one per package
// for the contract neither of its two packages carries.
const ORDER_FINDING_COUNT = 7

// One run over a tree of several packages, whose diagnostics the caller compares.
func fixture_ordered(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) (written string) {
  t.Helper()
  subject := fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "order")
  fixture_directory(t, subject, "order/nested")
  fixture_file(t, subject, "order/a.go", UNFORMATTED)
  fixture_file(t, subject, "order/b.go", REFUSED)
  fixture_file(t, subject, "order/c.go", INTERFACE_DECLARATION)
  fixture_file(t, subject, "order/nested/d.go", UNFORMATTED)
  fixture_file(t, subject, "order/nested/e.go", REFUSED)
  fixture_run(t, subject, []string{"harness", "order", "-json"})
  return fixture_written(subject)
}

// A diagnostic stream that takes no byte: the write of a report fails, and a failed step ends the
// run rather than passing it off as written.
func fixture_shut_output(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  subject := fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "shut")
  fixture_file(t, subject, "shut/a.go", UNFORMATTED)
  subject.Output.Closed = true
  fixture_run(t, subject, []string{"harness", "shut"})
  fixture_failed(t, subject, "report")
  // A clean package whose pass line takes no byte: the run states nothing, thus it cannot
  // state that it passed either.
  subject = fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "shut")
  fixture_file(t, subject, "shut/a.go", CLEAN)
  fixture_file(t, subject, "shut/"+requirements.MARKDOWN_NAME, CONTRACT_MARKDOWN)
  fixture_file(t, subject, "shut/"+requirements.TEST_NAME, CONTRACT_TEST)
  subject.Output.Closed = true
  fixture_run(t, subject, []string{"harness", "shut"})
  if subject.Runner.Status != harness.EXIT_FAILURE {
    t.Fatal("a pass line that took no byte left the run passing")
  }
  // A failure line that takes no byte: the run holds its failing status with nothing to
  // name it, since no second line can state the first one's own failure.
  subject = fixture_new(t, memory, format_state, command, directory, seed)
  subject.Errors.Closed = true
  fixture_run(t, subject, []string{"harness", "absent"})
  if subject.Runner.Status != harness.EXIT_FAILURE {
    t.Fatal("a failure line that took no byte left the run passing")
  }
}

// A failed open of the file a fix writes: the declaration pass opens it first, thus the fault
// lets one open pass and names the second.
func fixture_fix_open(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  subject := fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "fixopen")
  fixture_file(t, subject, "fixopen/a.go", UNFORMATTED)
  faults := Fault_Storage{
    Inner: subject.Loop.Storage,
    Loop:  subject.Loop,
    Kind:  FAULT_OPEN,
    Skip:  FIX_OPEN_SKIP,
  }
  fault_wire(subject, &faults)
  fixture_run(t, subject, []string{"harness", "fixopen", "-fix"})
  fixture_failed(t, subject, "fix")
}

// One failed read per pass over the same package: the declaration pass reads each Go file before
// the check pass reads it again, and the contract files follow both.
func fixture_read_passes(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  for skip := range FAULT_SKIP_COUNT {
    subject := fixture_new(t, memory, format_state, command, directory, seed)
    fixture_directory(t, subject, "passes")
    fixture_file(t, subject, "passes/a.go", UNFORMATTED)
    fixture_file(t, subject, "passes/"+requirements.MARKDOWN_NAME, CONTRACT_MARKDOWN)
    fixture_file(t, subject, "passes/"+requirements.TEST_NAME, CONTRACT_TEST)
    faults := Fault_Storage{
      Inner: subject.Loop.Storage,
      Loop:  subject.Loop,
      Kind:  FAULT_READ,
      Skip:  skip,
    }
    fault_wire(subject, &faults)
    fixture_run(t, subject, []string{"harness", "passes"})
    fixture_failed(t, subject, "read")
    // One fault stands per run, thus a second run meets none and opens on the
    // stores the failed one left: the pass that stands before any read of its own
    // states the kind that run stopped on rather than the kind this one will set.
    fixture_run(t, subject, []string{"harness", "passes"})
    fixture_states(t, subject, "passes/a.go: not formatted")
  }
}

// Every run closes what it opened: a descriptor the simulator still holds names a file the run
// left open, which a later run would stand on.
func fixture_drained(t *testing.T, subject *fixture) {
  t.Helper()
  for index := range subject.Simulation.Descriptors {
    if subject.Simulation.Descriptors[index].Used {
      t.Fatalf("a run left descriptor %d open", index)
    }
  }
  // An operation slot the simulator still holds names work the run never waited for, which
  // would retire into a runner that already stopped.
  for index := range subject.Simulation.Operations {
    if subject.Simulation.Operations[index].Kind != nbio.SIM_OPERATION_KIND_FREE {
      t.Fatalf("a run left operation %d in flight", index)
    }
  }
}

// Every run that exits passing names no failed step. The other way round states nothing: a run
// exits failing for a diagnostic it reported as well as for a step it could not take.
func fixture_status(t *testing.T, subject *fixture) {
  t.Helper()
  if subject.Runner.Status != harness.EXIT_SUCCESS {
    return
  }
  failures := fixture_errors(subject)
  if failures != "" {
    t.Fatalf("a run exited passing after writing %q", failures)
  }
}

// A drawn fault ends its run: the status names a failure, and the error stream names the step.
func fixture_failed(t *testing.T, subject *fixture, step string) {
  t.Helper()
  if subject.Runner.Status != harness.EXIT_FAILURE {
    t.Fatalf("fault at %s left status %d", step, subject.Runner.Status)
  }
  written := fixture_errors(subject)
  if !bool(strings.Contains(strings.Text(written), strings.Text(step))) {
    t.Fatalf("fault at %s wrote %q", step, written)
  }
  // One failed step ends the run, thus it names itself once and the run stops rather than
  // walks on to fail a second time.
  if fixture_lines(written) != 1 {
    t.Fatalf("fault at %s wrote %d lines: %q", step, fixture_lines(written), written)
  }
}

// One package holding every file a walk collects: a count and a slot each state the largest value
// its table admits, which no smaller tree reaches.
func fixture_counts(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  // One pending directory per slot: a subdirectory without a Go file joins the queue and
  // declares no package, thus the queue fills where the package table does not.
  subject := fixture_sized(
    t,
    memory,
    format_state,
    command,
    directory,
    seed,
    harness.DIRECTORY_COUNT_MAXIMUM+TALL_NODE_COUNT,
    NODE_CONTENTS_BYTES,
  )
  fixture_directory(t, subject, "wide")
  for index := range harness.DIRECTORY_COUNT_MAXIMUM {
    fixture_directory(t, subject, "wide/"+fixture_name(index))
  }
  fixture_run(t, subject, []string{"harness", "wide"})
  fixture_passed(t, subject)
  // One child past what one directory pass holds: the pass states no entry at all, thus
  // the walk fails rather than states a tree it never read.
  subject = fixture_sized(
    t,
    memory,
    format_state,
    command,
    directory,
    seed,
    harness.ENTRY_COUNT_MAXIMUM+TALL_NODE_COUNT,
    NODE_CONTENTS_BYTES,
  )
  fixture_directory(t, subject, "crowd")
  for index := range harness.ENTRY_COUNT_MAXIMUM + 1 {
    fixture_directory(t, subject, "crowd/"+fixture_name(index))
  }
  fixture_run(t, subject, []string{"harness", "crowd"})
  fixture_failed(t, subject, "walk")
  // One Go file past what the path table holds, spread over two directories so that no one
  // pass over a directory outgrows its entries: the walk fails rather than drops a file.
  subject = fixture_sized(
    t,
    memory,
    format_state,
    command,
    directory,
    seed,
    SPREAD_PART_COUNT*SPREAD_FILE_COUNT+TALL_NODE_COUNT,
    NODE_CONTENTS_BYTES,
  )
  fixture_directory(t, subject, "spread")
  for part := range SPREAD_PART_COUNT {
    branch := "spread/" + fixture_name(part)
    fixture_directory(t, subject, branch)
    for index := range SPREAD_FILE_COUNT {
      fixture_file(t, subject, branch+"/"+fixture_name(index)+".go", CLEAN)
    }
  }
  fixture_run(t, subject, []string{"harness", "spread"})
  fixture_failed(t, subject, "walk")
  fixture_queue(t, memory, format_state, command, directory, seed)
}

// One directory past what the pending queue holds: a walk stands on the last of two branches while
// the other waits, thus its children fill the queue one past its end.
func fixture_queue(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  subject := fixture_sized(
    t,
    memory,
    format_state,
    command,
    directory,
    seed,
    harness.DIRECTORY_COUNT_MAXIMUM+TALL_NODE_COUNT,
    NODE_CONTENTS_BYTES,
  )
  fixture_directory(t, subject, "queue")
  fixture_directory(t, subject, "queue/a")
  fixture_directory(t, subject, "queue/b")
  for index := range harness.DIRECTORY_COUNT_MAXIMUM {
    fixture_directory(t, subject, "queue/b/"+fixture_name(index))
  }
  fixture_run(t, subject, []string{"harness", "queue"})
  fixture_failed(t, subject, "walk")
}

// SPREAD_PART_COUNT is how many directories hold the files that outgrow the path table.
const SPREAD_PART_COUNT = 2

// SPREAD_FILE_COUNT fills those directories one file past what the path table holds, while each
// one stays inside what a single pass over its entries states.
const SPREAD_FILE_COUNT = harness.FILE_COUNT_MAXIMUM/SPREAD_PART_COUNT + 1

// One contract standing on every line its store holds: a heading on the last line of a full
// store stands on the widest line number a scan states, which no shorter file reaches.
func fixture_lined(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  subject := fixture_sized(
    t,
    memory,
    format_state,
    command,
    directory,
    seed,
    TALL_NODE_COUNT,
    requirements.MARKDOWN_STORAGE_SIZE,
  )
  fixture_directory(t, subject, "tall")
  fixture_file(t, subject, "tall/a.go", CLEAN)
  fixture_file(t, subject, "tall/"+requirements.MARKDOWN_NAME, fixture_markdown_tall())
  fixture_run(t, subject, []string{"harness", "tall"})
}

// Builds one contract of blank lines with a heading on the last line its store holds. The
// heading carries its marker, thus it stands one line short of the line index's own end.
func fixture_markdown_tall() (markdown string) {
  written := make([]byte, requirements.LINE_NUMBER_MAXIMUM-requirements.HEADING_MARKER_SIZE)
  for index := range written {
    written[index] = '\n'
  }
  return string(written) + "# "
}

// NESTED_DEPTH_COUNT is how deep one unprintable file nests. A nest spends two of the depth one
// parse admits, thus the file stands inside that bound with room for the function holding it.
const NESTED_DEPTH_COUNT = ast.DEPTH_MAXIMUM/2 - TALL_NODE_COUNT

// NESTED_NAME_SIZE is how wide the one name each line of an unprintable file spells. A wide
// name spends the source on bytes the form keeps rather than on lines the form indents, thus
// the file holds fewer lines for the same form. The form spends the indentation beside each
// name and the source does not, thus a name short of that indentation keeps the source under
// half the form, which is under its own bound.
const NESTED_NAME_SIZE = NESTED_DEPTH_COUNT * 5 / 3

// NESTED_INDENT_SIZE is how wide the canonical form indents one line of the innermost nest: two
// spaces for the function body and two for each nest inside it.
const NESTED_INDENT_SIZE = 2 * (NESTED_DEPTH_COUNT + 1)

// NESTED_LINE_COUNT is how many lines one unprintable file holds: enough that the indentation
// and the names they carry spend more than the store the printer writes into.
const NESTED_LINE_COUNT = harness.FORM_STORAGE_SIZE/(NESTED_INDENT_SIZE+NESTED_NAME_SIZE) + 1

// One file inside every bound a parse admits whose canonical form outgrows the store the printer
// writes into: indentation spends bytes the source never held, thus the run names the form it
// never wrote rather than stands on a part of one.
func fixture_unprintable(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  nested := fixture_nested()
  subject := fixture_sized(
    t,
    memory,
    format_state,
    command,
    directory,
    seed,
    TALL_NODE_COUNT,
    len(nested)+1,
  )
  fixture_directory(t, subject, "nested")
  fixture_file(t, subject, "nested/a.go", nested)
  fixture_run(t, subject, []string{"harness", "nested"})
  fixture_states(t, subject, "canonical form exceeds its storage")
  // A small tree next: the run opens on the diagnostic the run before it left, and so does
  // the close of its first file, thus both stand on the rule no smaller file breaks.
  fixture_directory(t, subject, "after")
  fixture_file(t, subject, "after/a.go", MAIN)
  fixture_run(t, subject, []string{"harness", "after"})
}

// Builds one file that nests as deep as a parse admits and holds lines enough for the canonical
// form of them to outgrow the printer's store.
func fixture_nested() (source string) {
  name := make([]byte, NESTED_NAME_SIZE)
  for index := range name {
    name[index] = 'x'
  }
  written := make([]byte, 0, NESTED_LINE_COUNT*(NESTED_NAME_SIZE+len("++\n")))
  written = append(written, "package a\n\nfunc f() {\n"...)
  for range NESTED_DEPTH_COUNT {
    written = append(written, "if x {\n"...)
  }
  for range NESTED_LINE_COUNT {
    written = append(append(written, name...), "++\n"...)
  }
  for range NESTED_DEPTH_COUNT {
    written = append(written, "}\n"...)
  }
  written = append(written, "}\n"...)
  return string(written)
}

// EXACT_FIXED_SIZE is what the canonical form of one nested file spends outside the lines of its
// innermost nest: the package clause and the function that opens under a blank line, each nest
// opening at its own depth, each nest closing at its own depth, and the brace that closes the
// function.
const EXACT_FIXED_SIZE = len("package a\n\nfunc f() {\n") +
  NESTED_DEPTH_COUNT*(NESTED_DEPTH_COUNT+1) +
  NESTED_DEPTH_COUNT*len("if x {\n") +
  NESTED_DEPTH_COUNT*(NESTED_DEPTH_COUNT+1) +
  NESTED_DEPTH_COUNT*len("}\n") +
  len("}\n")

// EXACT_LINE_SIZE is what the canonical form spends on one line of the innermost nest.
const EXACT_LINE_SIZE = NESTED_INDENT_SIZE + NESTED_NAME_SIZE + len("++\n")

// EXACT_LINE_COUNT is how many lines of the innermost nest one exact file holds: as many as fit
// under the store, since the last of them carries the bytes that close the rest of the gap.
const EXACT_LINE_COUNT = (harness.FORM_STORAGE_SIZE - EXACT_FIXED_SIZE) / EXACT_LINE_SIZE

// EXACT_LAST_SIZE is how wide the name on the last line of one exact file stands: the one name
// that spends what the whole lines leave short of the store.
const EXACT_LAST_SIZE = NESTED_NAME_SIZE +
  harness.FORM_STORAGE_SIZE -
  EXACT_FIXED_SIZE -
  EXACT_LINE_COUNT*EXACT_LINE_SIZE

// One file whose canonical form fills the store the printer writes into, fixed in place: the fix
// the run writes spends every byte one form holds, which a smaller file never reaches. The main
// file beside it exempts the package from its contract, thus the fix is all the run writes.
func fixture_exact(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  subject := fixture_sized(
    t,
    memory,
    format_state,
    command,
    directory,
    seed,
    TALL_NODE_COUNT,
    harness.FORM_STORAGE_SIZE+1,
  )
  fixture_directory(t, subject, "exact")
  fixture_file(t, subject, "exact/m.go", MAIN)
  // The fixed file stands last in path order, thus its fix is the one the run leaves.
  fixture_file(t, subject, "exact/z.go", fixture_form_exact())
  fixture_run(t, subject, []string{"harness", "exact", "-fix"})
  fixture_passed(t, subject)
  // The same tree again: a run opens on the fix and the count the run before it left, thus
  // it stands on a fix of the whole store before it reads a byte.
  fixture_run(t, subject, []string{"harness", "exact"})
}

// Builds one nested file whose canonical form spends exactly the store the printer writes into.
func fixture_form_exact() (source string) {
  written := make([]byte, 0, harness.FORM_STORAGE_SIZE/2)
  written = append(written, "package a\n\nfunc f() {\n"...)
  for range NESTED_DEPTH_COUNT {
    written = append(written, "if x {\n"...)
  }
  for index := range EXACT_LINE_COUNT {
    width := NESTED_NAME_SIZE
    if index == EXACT_LINE_COUNT-1 {
      width = EXACT_LAST_SIZE
    }
    for range width {
      written = append(written, 'x')
    }
    written = append(written, "++\n"...)
  }
  for range NESTED_DEPTH_COUNT {
    written = append(written, "}\n"...)
  }
  written = append(written, "}\n"...)
  return string(written)
}

// WIDENING_CHARACTER is one character whose upper case spends one byte more than it does.
const WIDENING_CHARACTER = '\u023f'

// WIDENING_WORD_COUNT is how many one-character words of it one heading holds, each closed by a
// space but the last.
const WIDENING_WORD_COUNT = (requirements.HEADING_SIZE_MAXIMUM + len(" ")) /
  requirements.GROWN_WORD_SIZE_MINIMUM

// One contract whose # and ### each spend the whole heading width on words that widen in upper
// case, answered by a test file that declares no test for them: the name the leaf builds is the
// widest a name stands, thus the finding that names it states the widest message a check writes.
func fixture_widest_name(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  heading := make([]byte, 0, requirements.HEADING_SIZE_MAXIMUM)
  for index := range WIDENING_WORD_COUNT {
    if index > 0 {
      heading = append(heading, ' ')
    }
    heading = append(heading, string(WIDENING_CHARACTER)...)
  }
  markdown := "\n# " + string(heading) + "\n\nBody.\n\n### " + string(heading) + "\n\nBody.\n"
  subject := fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "widest")
  fixture_file(t, subject, "widest/a.go", UNFORMATTED)
  fixture_file(t, subject, "widest/"+requirements.MARKDOWN_NAME, markdown)
  fixture_file(
    t, subject, "widest/"+requirements.TEST_NAME, "package a\n\nfunc Test_Other() {}\n",
  )
  fixture_run(t, subject, []string{"harness", "widest"})
  fixture_run(t, subject, []string{"harness", "widest", "-json"})
  // The same tree again: a run opens on the message the run before it left, and so does the
  // report of its Go file, which the run writes before any contract finding.
  fixture_run(t, subject, []string{"harness", "widest", "-json"})
  // The same contract beside a Go file in its canonical form: the first report the run writes
  // is a contract finding, thus it opens on the widest message the run before it left.
  fixture_directory(t, subject, "clean")
  fixture_file(t, subject, "clean/a.go", CLEAN)
  fixture_file(t, subject, "clean/"+requirements.MARKDOWN_NAME, markdown)
  fixture_file(
    t, subject, "clean/"+requirements.TEST_NAME, "package a\n\nfunc Test_Other() {}\n",
  )
  fixture_run(t, subject, []string{"harness", "clean", "-json"})
}

// A contract test file whose only declaration stands on the last line the line index holds, then
// one line past it: the first names the widest line a finding states, and the second names none,
// since a source of more lines than the index holds states no position at all.
func fixture_test_lines(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  // The last line closes on no newline, thus the file holds as many lines as the index.
  widest := fixture_test_tall(token.LINE_COUNT_MAXIMUM, "")
  past := fixture_test_tall(token.LINE_COUNT_MAXIMUM, "\n")
  subject := fixture_sized(
    t,
    memory,
    format_state,
    command,
    directory,
    seed,
    TALL_NODE_COUNT,
    len(past)+1,
  )
  fixture_directory(t, subject, "lines")
  fixture_file(t, subject, "lines/a.go", CLEAN)
  fixture_file(t, subject, "lines/"+requirements.MARKDOWN_NAME, CONTRACT_MARKDOWN)
  fixture_file(t, subject, "lines/"+requirements.TEST_NAME, widest)
  fixture_run(t, subject, []string{"harness", "lines"})
  fixture_states(t, subject, fmt.Sprintf(":%d: The file needs", token.LINE_COUNT_MAXIMUM))
  fixture_file(t, subject, "lines/"+requirements.TEST_NAME, past)
  fixture_run(t, subject, []string{"harness", "lines"})
}

// Builds one test file whose only declaration stands on the given line, closed by the given end.
func fixture_test_tall(line int, end string) (source string) {
  head := "package a\n"
  gap := make([]byte, line-2)
  for index := range gap {
    gap[index] = '\n'
  }
  return head + string(gap) + "func Test_B() {}" + end
}

// A command line naming an option as wide as the error one line states and more: the error shared
// cli renders names that option whole, thus the line the run writes fills the line store. Each
// run after it opens on that line, and a step that writes a line of its own stands on it first.
func fixture_full_line(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  subject := fixture_new(t, memory, format_state, command, directory, seed)
  subject.Output = nbio.Stream_Memory{Memory: make([]byte, WIDE_OUTPUT_BYTES)}
  subject.Runner.Output = nbio.Memory_To_Stream(&subject.Output)
  subject.Errors = nbio.Stream_Memory{Memory: make([]byte, WIDE_OUTPUT_BYTES)}
  subject.Runner.Error_Output = nbio.Memory_To_Stream(&subject.Errors)
  fixture_directory(t, subject, "fine")
  fixture_file(t, subject, "fine/a.go", CLEAN)
  fixture_file(t, subject, "fine/"+requirements.MARKDOWN_NAME, CONTRACT_MARKDOWN)
  fixture_file(t, subject, "fine/"+requirements.TEST_NAME, CONTRACT_TEST)
  fixture_directory(t, subject, "bent")
  fixture_file(t, subject, "bent/a.go", UNFORMATTED)
  fixture_directory(t, subject, "owed")
  fixture_file(t, subject, "owed/a.go", CLEAN)
  option := make([]byte, harness.USAGE_ERROR_SIZE_MAXIMUM)
  for index := range option {
    option[index] = 'o'
  }
  refused := []string{"harness", "fine", "-" + string(option)}
  for _, after := range [][]string{
    refused,
    {"harness", "bent"},
    {"harness", "absent"},
    {"harness", "fine"},
    {"harness", "owed", "-json"},
  } {
    fixture_run(t, subject, refused)
    fixture_refused(t, subject)
    if len(fixture_errors(subject)) != harness.LINE_SIZE_MAXIMUM {
      t.Fatalf("a refused wide option wrote %q", fixture_errors(subject))
    }
    fixture_run(t, subject, after)
  }
}

// A command line as wide as a host hands one: a vector of as many arguments as the root reads,
// and a directory argument of as many bytes as one text holds. Neither names a walk, thus each run
// fails rather than walks.
func fixture_wide_command(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  subject := fixture_new(t, memory, format_state, command, directory, seed)
  arguments := make([]string, harness.TOKEN_COUNT_MAXIMUM)
  arguments[0] = "harness"
  for index := 1; index < len(arguments); index++ {
    arguments[index] = "a"
  }
  fixture_run(t, subject, arguments)
  fixture_refused(t, subject)
  root := make([]byte, strings.TEXT_SIZE_MAXIMUM)
  for index := range root {
    root[index] = 'a'
  }
  fixture_run(t, subject, []string{"harness", string(root)})
  fixture_failed(t, subject, "walk")
}

// Each parse opens on the one the parse before it left: a Go file the parser refuses at its first
// tokens, for a grouped declaration, and for its nesting, each the last file its package checks,
// thus the contract parse after it and the package after that open on the refusal.
func fixture_stale(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  nested := "package a\n\nvar a = " +
    fixture_repeat("(", ast.DEPTH_MAXIMUM) +
    "1" +
    fixture_repeat(")", ast.DEPTH_MAXIMUM) +
    "\n"
  for _, refused := range []string{
    "package 1\n",
    "package a b\n",
    "package a\n\nvar (\n\ta = 1\n)\n",
    nested,
  } {
    subject := fixture_new(t, memory, format_state, command, directory, seed)
    fixture_directory(t, subject, "stale")
    fixture_directory(t, subject, "stale/p")
    fixture_file(t, subject, "stale/p/"+requirements.MARKDOWN_NAME, CONTRACT_MARKDOWN)
    fixture_file(t, subject, "stale/p/"+requirements.TEST_NAME, CONTRACT_TEST)
    fixture_file(t, subject, "stale/p/z.go", refused)
    fixture_directory(t, subject, "stale/q")
    fixture_file(t, subject, "stale/q/a.go", CLEAN)
    fixture_run(t, subject, []string{"harness", "stale"})
    fixture_states(t, subject, "stale/p/z.go:")
  }
}

// TAIL_NODE_COUNT holds the tail tree: the root, its directory, three files, and go.mod.
const TAIL_NODE_COUNT = 9

// TAIL_HEAD opens the widest test source on a string that fills it.
const TAIL_HEAD = "package a\n\nvar b = \""

// TAIL_DECLARATION closes the string, and the source on the function whose name stands closest
// to its end: the parameters are all that follow it.
const TAIL_DECLARATION = "\"\n\nfunc a()"

// TAIL_CONTRACT opens its second heading on the line after a body.
const TAIL_CONTRACT = "\n# Contract\n\nIt holds.\n# Next\n\nIt holds.\n"

// A test file as wide as a source holds, whose last declaration closes it: the span of that name
// stands at the widest offsets one slot states, and the read that loads it stands one byte short
// of the store before the read that finds its end. The contract above it opens a heading on the
// line after a body, which draws a finding of its own.
func fixture_test_tail(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  subject := fixture_sized(
    t,
    memory,
    format_state,
    command,
    directory,
    seed,
    TAIL_NODE_COUNT,
    requirements.SOURCE_SIZE_MAXIMUM+1,
  )
  test := make([]byte, 0, requirements.SOURCE_SIZE_MAXIMUM)
  test = append(test, TAIL_HEAD...)
  for len(test) < requirements.SOURCE_SIZE_MAXIMUM-len(TAIL_DECLARATION) {
    test = append(test, 'x')
  }
  test = append(test, TAIL_DECLARATION...)
  fixture_directory(t, subject, "tail")
  fixture_file(t, subject, "tail/a.go", CLEAN)
  fixture_file(t, subject, "tail/"+requirements.MARKDOWN_NAME, TAIL_CONTRACT)
  fixture_file(t, subject, "tail/"+requirements.TEST_NAME, string(test))
  fixture_run(t, subject, []string{"harness", "tail"})
  fixture_states(t, subject, "tail/"+requirements.MARKDOWN_NAME+":")
}

// REWRITE_LEAF is a contract whose leaf heading stands on the line after its branch, and whose
// body holds a blank line between its two lines.
const REWRITE_LEAF = "\n# A\n\n### B\nBody.\n\nMore.\n"

// REWRITE_LEAF_FIXED is what the rewrite makes of it.
const REWRITE_LEAF_FIXED = "\n# A\n\n### B\n\nBody.\nMore.\n"

// Contracts a fix rewrites: one of a lone empty heading, which is the shortest fix a file takes,
// one whose leaf the rewrite gives its blank lines, and one whose heading is past its bound, which
// the rewrite leaves as the check will state it.
func fixture_rewritten(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  subject := fixture_new(t, memory, format_state, command, directory, seed)
  long := "# " + fixture_repeat("A", requirements.HEADING_SIZE_MAXIMUM+1) + "\nBody.\n"
  contracts := [...]string{"# ", REWRITE_LEAF, long}
  fixture_directory(t, subject, "loose")
  for index, markdown := range contracts {
    name := "loose/" + fixture_name(index)
    fixture_directory(t, subject, name)
    fixture_file(t, subject, name+"/a.go", CLEAN)
    fixture_file(t, subject, name+"/"+requirements.MARKDOWN_NAME, markdown)
    fixture_file(t, subject, name+"/"+requirements.TEST_NAME, CONTRACT_TEST)
  }
  fixture_run(t, subject, []string{"harness", "loose", "-fix"})
  fixed := [...]string{"\n# \n", REWRITE_LEAF_FIXED, long}
  for index, want := range fixed {
    path := "loose/" + fixture_name(index) + "/" + requirements.MARKDOWN_NAME
    held := fixture_contents(t, subject, path)
    if held != want {
      t.Fatalf("a fix of %s left %q", path, held)
    }
  }
}

// Repeats one text a count of times.
func fixture_repeat(text string, count int) (repeated string) {
  built := make([]byte, 0, len(text)*count)
  for range count {
    built = append(built, text...)
  }
  return string(built)
}

// TALL_NODE_COUNT holds the widest tree one tall source stands in: the root, its directory, five
// files, harness.json, and go.mod.
const TALL_NODE_COUNT = 10

// A source refused on the last line the index holds: a position states the largest line a
// diagnostic names, which no shorter source reaches.
func fixture_positions(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  subject := fixture_reported(
    t,
    memory,
    format_state,
    command,
    directory,
    seed,
    "tall",
    fixture_tall(),
    `"line":131072`,
  )
  // A run the command line refuses opens on what the run before it left, at no cost of
  // walking that tree again.
  fixture_run(t, subject, []string{"harness", ""})
  fixture_failed(t, subject, "walk")
  // One line past what the index holds: the refusal stands at no position the index can
  // name, thus the line states the path alone.
  over := fixture_tall() + "\n&"
  subject = fixture_sized(
    t,
    memory,
    format_state,
    command,
    directory,
    seed,
    TALL_NODE_COUNT,
    len(over)+1,
  )
  fixture_directory(t, subject, "past")
  fixture_file(t, subject, "past/a.go", over)
  fixture_run(t, subject, []string{"harness", "past", "-json"})
  fixture_states(t, subject, `"line":0,"column":0,"tier":2,"rule":"refused"`)
  // The same refusal as text: a line states the path alone, since no position stands for
  // the index to name.
  fixture_run(t, subject, []string{"harness", "past"})
  fixture_states(t, subject, "past/a.go: Fix the syntax at this token.\n")
  fixture_huge(t, memory, format_state, command, directory, seed)
  fixture_spare(t, memory, format_state, command, directory, seed)
}

// A source that fills the whole store: the read stops at the store's end, which states the largest
// count a read reaches and the oversized rule it proves. One source of one line, refused at its
// last byte: a position states the widest column a token stands at, which only a source without a
// newline reaches.
func fixture_huge(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  wide := fixture_wide()
  subject := fixture_sized(
    t,
    memory,
    format_state,
    command,
    directory,
    seed,
    TALL_NODE_COUNT,
    len(wide)+1,
  )
  fixture_directory(t, subject, "huge")
  fixture_file(t, subject, "huge/a.go", wide)
  fixture_run(t, subject, []string{"harness", "huge", "-json"})
  fixture_states(t, subject, `"rule":"oversized"`)
  fixture_reported(
    t,
    memory,
    format_state,
    command,
    directory,
    seed,
    "broad",
    fixture_broad(),
    `"column":`+fmt.Sprint(requirements.SOURCE_SIZE_MAXIMUM),
  )
  fixture_fat(t, memory, format_state, command, directory, seed)
  fixture_headings(t, memory, format_state, command, directory, seed)
}

// The widest source a fix rewrites whole, then the same source as a contract test file.
func fixture_spare(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  spacious := fixture_spacious()
  subject := fixture_sized(
    t,
    memory,
    format_state,
    command,
    directory,
    seed,
    TALL_NODE_COUNT,
    len(spacious)+1,
  )
  fixture_directory(t, subject, "spare")
  fixture_file(t, subject, "spare/a.go", spacious)
  fixture_run(t, subject, []string{"harness", "spare", "-fix"})
  held := fixture_contents(t, subject, "spare/a.go")
  if held != FIXED {
    t.Fatalf("a fix of the widest source wrote %q", held)
  }
  // The same source as a contract test file: the check loads every byte of it, which is the
  // widest test file a check takes whole.
  subject = fixture_sized(
    t,
    memory,
    format_state,
    command,
    directory,
    seed,
    TALL_NODE_COUNT,
    len(spacious)+1,
  )
  fixture_directory(t, subject, "spare")
  fixture_file(t, subject, "spare/a.go", CLEAN)
  fixture_file(t, subject, "spare/"+requirements.MARKDOWN_NAME, CONTRACT_MARKDOWN)
  fixture_file(t, subject, "spare/"+requirements.TEST_NAME, spacious)
  fixture_run(t, subject, []string{"harness", "spare"})
}

// FIXED is the canonical form of a source that holds a package clause and spare bytes.
const FIXED = "package a\n"

// Reads one file back, at the widest a fix of it writes.
func fixture_contents(t *testing.T, subject *fixture, path string) (contents string) {
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
  buffer := make([]byte, FIXED_SIZE_MAXIMUM)
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
  fixture_close(t, subject, file)
  // A read that fills its buffer states a file wider than one a fixture writes back, thus
  // the comparison that follows would stand on bytes the file never ended at.
  if count == len(buffer) {
    t.Fatalf("a read of %s filled its buffer", path)
  }
  return string(buffer[:count])
}

// FIXED_SIZE_MAXIMUM bounds what one read of a fixed file takes back, which holds every source a
// fixture writes and reads again.
const FIXED_SIZE_MAXIMUM = 256

// One source of the widest length a check admits that still parses: the whole of it reaches the
// formatter, which no refused source does, and its canonical form drops every spare byte.
func fixture_spacious() (source string) {
  head := "package a\n"
  gap := make([]byte, requirements.SOURCE_SIZE_MAXIMUM-len(head))
  for index := range gap {
    gap[index] = ' '
  }
  return head + string(gap)
}

// A contract that fills its whole store: the check stops at once, since a cut file would state
// faults it does not hold, thus the diagnostic it writes stands alone.
func fixture_fat(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  contract := fixture_contract_full()
  subject := fixture_sized(
    t,
    memory,
    format_state,
    command,
    directory,
    seed,
    TALL_NODE_COUNT,
    len(contract)+1,
  )
  fixture_directory(t, subject, "fat")
  fixture_file(t, subject, "fat/a.go", CLEAN)
  fixture_file(t, subject, "fat/"+requirements.MARKDOWN_NAME, contract)
  fixture_run(t, subject, []string{"harness", "fat"})
  fixture_states(t, subject, "The file is larger than its storage.")
  // The leaves go unchecked, thus the diagnostic stands alone rather than beside faults
  // stated of a contract the run holds only the first bytes of.
  written := fixture_written(subject)
  if fixture_lines(written) != 1 {
    t.Fatalf("a package whose contract fills its store wrote %q", written)
  }
}

// One contract of the whole width its store holds, thus a read of it fills that store.
func fixture_contract_full() (markdown string) {
  body := make([]byte, requirements.MARKDOWN_STORAGE_SIZE-len("\n# Contract\n\n"))
  for index := range body {
    body[index] = 'x'
  }
  return "\n# Contract\n\n" + string(body)
}

// One source holding the widest line a check admits, refused at its last column. It carries no
// newline, since a second line would put the refusal at a column the first line already passed.
func fixture_broad() (source string) {
  head := "package a"
  tail := "&"
  gap := make([]byte, requirements.SOURCE_SIZE_MAXIMUM-len(head)-len(tail))
  for index := range gap {
    gap[index] = ' '
  }
  return head + string(gap) + tail
}

// One source one byte past the widest a check admits, which fills the whole source store.
func fixture_wide() (source string) {
  head := "package a\n"
  filler := make([]byte, harness.SOURCE_STORAGE_SIZE-len(head))
  for index := range filler {
    filler[index] = '\n'
  }
  return head + string(filler)
}

// One package run twice as json: the second run opens on the report the first one wrote, thus
// every step that takes that report states the position the first run put there rather than the
// one this run will. The package holds a main file the parser takes, so it owes no contract and
// no contract finding stands past the Go diagnostic to leave its own position in the store.
func fixture_reported(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
  name string,
  source string,
  held string,
) (subject *fixture) {
  t.Helper()
  subject = fixture_sized(
    t,
    memory,
    format_state,
    command,
    directory,
    seed,
    TALL_NODE_COUNT,
    len(source)+1,
  )
  fixture_directory(t, subject, name)
  // The main file stands first in path order, thus the file this states is the last the
  // run checks and the diagnostic it leaves is the one a later run opens on.
  fixture_file(t, subject, name+"/a.go", MAIN)
  fixture_file(t, subject, name+"/z.go", source)
  fixture_run(t, subject, []string{"harness", name, "-json"})
  fixture_states(t, subject, held)
  fixture_run(t, subject, []string{"harness", name, "-json"})
  fixture_states(t, subject, held)
  // A package that owes a contract next: the finding it states opens on the report the Go
  // file left, since a contract finding writes into the same store.
  fixture_directory(t, subject, name+"_owed")
  fixture_file(t, subject, name+"_owed/a.go", CLEAN)
  fixture_run(t, subject, []string{"harness", name + "_owed", "-json"})
  return subject
}

// One source holding every line the index admits, refused on the last of them.
func fixture_tall() (source string) {
  head := "package a\n"
  // No newline after the refusal: one more would start a line past the index, and the index
  // would hold no position for it.
  tail := "&"
  // One line for the package clause and one for the refusal; the rest stand between them.
  blanks := make([]byte, token.LINE_COUNT_MAXIMUM-2)
  for index := range blanks {
    blanks[index] = '\n'
  }
  return head + string(blanks) + tail
}

// One name per slot, in a fixed width so the walk sorts them the same way each run.
func fixture_name(index int) (name string) {
  digits := [NAME_WIDTH]byte{}
  for place := range digits {
    digits[len(digits)-1-place] = byte('a' + index%26)
    index /= 26
  }
  return string(digits[:])
}

// A directory of one byte, one of two, and a chain that carries a path to its whole width: a path
// store states its smallest and its largest length. One package per tree, thus these stand
// outside the seed sweep.
func fixture_widths(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  subject := fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "a")
  fixture_file(t, subject, "a/b.go", CLEAN)
  fixture_run(t, subject, []string{"harness", "a"})
  subject = fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "ab")
  fixture_file(t, subject, "ab/b.go", CLEAN)
  // A child under a directory of two bytes: the walk states that parent's path as it
  // takes the child, which a wider parent never states.
  fixture_directory(t, subject, "ab/c")
  fixture_file(t, subject, "ab/c/d.go", CLEAN)
  fixture_run(t, subject, []string{"harness", "ab"})
  // A refusal at the shortest path a run reads: the line it states opens with that path
  // and nothing else, which no longer path reaches.
  subject = fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "a")
  fixture_file(t, subject, "a/b.go", REFUSED)
  fixture_run(t, subject, []string{"harness", "a"})
  fixture_states(t, subject, "a/b.go:3:9")
  subject = fixture_new(t, memory, format_state, command, directory, seed)
  deep := fixture_chain(t, subject)
  fixture_file(t, subject, deep+"/b.go", REFUSED)
  fixture_run(t, subject, []string{"harness", deep})
  fixture_run(t, subject, []string{"harness", deep, "-json"})
  // One step past that width: the walk admits the directory and its Go file, and the check
  // would name a contract file past what a read path holds, thus the run fails the walk.
  subject = fixture_new(t, memory, format_state, command, directory, seed)
  over := fixture_chain(t, subject) + "/d"
  fixture_directory(t, subject, over)
  fixture_file(t, subject, over+"/a.go", CLEAN)
  fixture_run(t, subject, []string{"harness", over})
  fixture_failed(t, subject, "walk")
  // A package of one main file at the shortest path a run reads: it owes no contract, thus
  // the run ends with that path rather than one the contract check named after it.
  subject = fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "a")
  fixture_file(t, subject, "a/m.go", MAIN)
  fixture_run(t, subject, []string{"harness", "a"})
  fixture_passed(t, subject)
  // A Go file whose path spends the whole width: the check reads it, thus every step of a
  // check states a read path of that width rather than one the chain left room under.
  subject = fixture_new(t, memory, format_state, command, directory, seed)
  deepest := fixture_chain(t, subject)
  spare_size := filepath.PATH_SIZE_MAXIMUM - len(deepest) - len("/") - len(GO_SUFFIX_TEXT)
  name := make([]byte, spare_size)
  for index := range name {
    name[index] = 'w'
  }
  fixture_file(t, subject, deepest+"/"+string(name)+GO_SUFFIX_TEXT, REFUSED)
  fixture_run(t, subject, []string{"harness", deepest})
  fixture_states(t, subject, string(name)+GO_SUFFIX_TEXT+":3:9")
  // The same file reported as json: its path field spends the whole width a path holds,
  // which the text line states as part of a longer line rather than on its own.
  fixture_run(t, subject, []string{"harness", deepest, "-json"})
  fixture_states(t, subject, string(name)+GO_SUFFIX_TEXT)
  // A package at the whole width a path holds, owing both contract files: the finding that
  // names its test file states the widest path text a report writes.
  subject = fixture_new(t, memory, format_state, command, directory, seed)
  widest := fixture_chain(t, subject)
  fixture_file(t, subject, widest+"/a.go", REFUSED)
  fixture_file(t, subject, widest+"/"+requirements.MARKDOWN_NAME, CONTRACT_MARKDOWN)
  fixture_run(t, subject, []string{"harness", widest, "-json"})
  fixture_states(t, subject, widest+"/"+requirements.TEST_NAME)
  // A run the command line refuses opens on what the run before it left, at no cost of
  // walking that tree again.
  fixture_run(t, subject, []string{"harness", ""})
  fixture_failed(t, subject, "walk")
}

// The widest source one parse reads, the last line one index holds, and the last column one line
// holds: each states the largest value its type admits. One tree of one package each, thus these
// stand outside the seed sweep, which orders the retirements of several files.
func fixture_edges(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  var subject *fixture
  // An argument one byte past what a path slot holds: the walk fails before it opens
  // anything, and its failure line names no path at all.
  subject = fixture_new(t, memory, format_state, command, directory, seed)
  over := make([]byte, filepath.PATH_SIZE_MAXIMUM+1)
  for index := range over {
    over[index] = 'o'
  }
  fixture_run(t, subject, []string{"harness", string(over)})
  fixture_failed(t, subject, "walk")
  // A directory at the whole width a path holds: a walk opens it, and it owes no contract,
  // so no read names a path past what a read path holds. A second walk opens on its parent,
  // thus it records that directory from the widest parent a child fits under.
  subject = fixture_new(t, memory, format_state, command, directory, seed)
  widest := fixture_chain_full(t, subject)
  fixture_run(t, subject, []string{"harness", widest})
  fixture_run(t, subject, []string{"harness", widest[:len(widest)-len("/w")]})
  subject = fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "codes")
  fixture_file(t, subject, "codes/a.go", "package a\n\nimport (\n\t\"os\"\n)\n")
  fixture_run(t, subject, []string{"harness", "codes", "-json"})
  subject = fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "codes")
  fixture_file(t, subject, "codes/a.go", fixture_deep_nest())
  fixture_run(t, subject, []string{"harness", "codes", "-json"})
  fixture_states(t, subject, `"rule":"refused"`)
  // A second run over the same source: it opens on the deepest refusal one parse names,
  // which no shallower fixture leaves behind.
  fixture_run(t, subject, []string{"harness", "codes", "-json"})
  subject = fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "wide")
  fixture_file(t, subject, "wide/a.go", CLEAN)
  fixture_file(t, subject, "wide/REQUIREMENTS.md", fixture_contract())
  fixture_run(t, subject, []string{"harness", "wide"})
  // A test file that names another test: the finding names the test the widest leaf owes,
  // and that name stands twice in the message, thus a report carries the widest message a
  // run writes rather than the short ones a Go file yields.
  fixture_file(t, subject, "wide/"+requirements.TEST_NAME, "package a\n\nfunc Test_Z() {}\n")
  fixture_run(t, subject, []string{"harness", "wide", "-json"})
  fixture_states(t, subject, "Declare the tests in leaf order at the top of the file.")
}

// One source nested deeper than one parse admits.
func fixture_deep_nest() (source string) {
  opened := make([]byte, ast.DEPTH_MAXIMUM+1)
  closed := make([]byte, ast.DEPTH_MAXIMUM+1)
  for index := range opened {
    opened[index] = '{'
    closed[index] = '}'
  }
  return "package a\n\nfunc f() " + string(opened) + string(closed) + "\n"
}

// A heading one byte past its width, and a contract of more headings than the table holds: each
// states the bound it stands past rather than the sections it names.
func fixture_headings(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  subject := fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "heads")
  fixture_file(t, subject, "heads/a.go", CLEAN)
  fixture_file(t, subject, "heads/"+requirements.MARKDOWN_NAME, fixture_heading_wide())
  fixture_run(t, subject, []string{"harness", "heads"})
  fixture_states(t, subject, "The heading is longer than 128 bytes.")
  crowded := fixture_headings_full()
  subject = fixture_sized(
    t,
    memory,
    format_state,
    command,
    directory,
    seed,
    TALL_NODE_COUNT,
    len(crowded)+1,
  )
  fixture_directory(t, subject, "heads")
  fixture_file(t, subject, "heads/a.go", CLEAN)
  fixture_file(t, subject, "heads/"+requirements.MARKDOWN_NAME, crowded)
  fixture_run(t, subject, []string{"harness", "heads"})
  fixture_states(t, subject, "more headings than its heading table holds")
  // A contract test file past the source bound: the check of it states the bound, and the
  // contract states the leaf it cannot match rather than a test it never read.
  wide := fixture_wide()
  subject = fixture_sized(
    t,
    memory,
    format_state,
    command,
    directory,
    seed,
    TALL_NODE_COUNT,
    len(wide)+1,
  )
  fixture_directory(t, subject, "tests")
  fixture_file(t, subject, "tests/a.go", CLEAN)
  fixture_file(t, subject, "tests/"+requirements.MARKDOWN_NAME, CONTRACT_MARKDOWN)
  fixture_file(t, subject, "tests/"+requirements.TEST_NAME, wide)
  fixture_run(t, subject, []string{"harness", "tests"})
  fixture_states(t, subject, "exceeds source size")
  // The leaves go unchecked, thus the run states that one diagnostic and no missing test
  // for a file it never read whole.
  written := fixture_written(subject)
  if fixture_lines(written) != 1 {
    t.Fatalf("a package whose test file is oversized wrote %q", written)
  }
}

// One contract whose heading stands one byte past the width a heading holds.
func fixture_heading_wide() (markdown string) {
  branch := make([]byte, requirements.HEADING_SIZE_MAXIMUM+1)
  for index := range branch {
    branch[index] = 'A'
  }
  return "\n# " + string(branch) + "\n\nBody.\n"
}

// One contract of one heading past what the heading table holds, each with a body of its own.
func fixture_headings_full() (markdown string) {
  body := make([]byte, 0, HEADINGS_FULL_BYTES)
  for index := range requirements.HEADING_COUNT_MAXIMUM + 1 {
    body = append(body, "\n# "...)
    body = append(body, fixture_name(index)...)
    body = append(body, "\n\nBody.\n"...)
  }
  return string(body)
}

// One contract filling the heading table exactly, each ### under a # of its own: every pass past
// the heading scan stands on every slot the table holds, which a contract of one heading fewer
// never reaches.
func fixture_filled(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  subject := fixture_sized(
    t,
    memory,
    format_state,
    command,
    directory,
    seed,
    TALL_NODE_COUNT,
    HEADINGS_FULL_BYTES,
  )
  subject.Output = nbio.Stream_Memory{Memory: make([]byte, WIDE_OUTPUT_BYTES)}
  subject.Runner.Output = nbio.Memory_To_Stream(&subject.Output)
  fixture_directory(t, subject, "filled")
  fixture_file(t, subject, "filled/a.go", CLEAN)
  fixture_file(
    t, subject, "filled/"+requirements.MARKDOWN_NAME, fixture_headings_filled(),
  )
  fixture_run(t, subject, []string{"harness", "filled"})
  // The table filled with lone # sections: the last one closes on the last slot, which a
  // table ending in a ### never leaves to a lone section.
  subject = fixture_sized(
    t,
    memory,
    format_state,
    command,
    directory,
    seed,
    TALL_NODE_COUNT,
    HEADINGS_FULL_BYTES,
  )
  subject.Output = nbio.Stream_Memory{Memory: make([]byte, WIDE_OUTPUT_BYTES)}
  subject.Runner.Output = nbio.Memory_To_Stream(&subject.Output)
  fixture_directory(t, subject, "lone")
  fixture_file(t, subject, "lone/a.go", CLEAN)
  fixture_file(t, subject, "lone/"+requirements.MARKDOWN_NAME, fixture_headings_lone())
  fixture_run(t, subject, []string{"harness", "lone"})
}

// Builds one contract of exactly as many lone # sections as the heading table holds.
func fixture_headings_lone() (markdown string) {
  body := make([]byte, 0, HEADINGS_FULL_BYTES)
  for index := range requirements.HEADING_COUNT_MAXIMUM {
    body = append(body, "\n# B"...)
    body = append(body, fixture_name(index)...)
    body = append(body, "\n\nBody.\n"...)
  }
  return string(body)
}

// Builds one contract of exactly as many headings as the table holds, each ### under a # of its
// own.
func fixture_headings_filled() (markdown string) {
  body := make([]byte, 0, HEADINGS_FULL_BYTES)
  for index := range requirements.HEADING_COUNT_MAXIMUM / HEADING_PAIR_COUNT {
    body = append(body, "\n# B"...)
    body = append(body, fixture_name(index)...)
    body = append(body, "\n\nBody.\n\n### L"...)
    body = append(body, fixture_name(index)...)
    body = append(body, "\n\nBody.\n"...)
  }
  return string(body)
}

// HEADING_PAIR_COUNT is how many headings one pair of them spends: the # and the ### under it.
const HEADING_PAIR_COUNT = 2

// HEADINGS_FULL_BYTES holds that contract: one heading line and one body line apiece.
const HEADINGS_FULL_BYTES = (requirements.HEADING_COUNT_MAXIMUM + 1) * 32

// One contract whose two headings each spend the whole heading width and spell the same name,
// thus the message that names one quotes a span of that width.
func fixture_contract_twice() (markdown string) {
  branch := make([]byte, requirements.HEADING_SIZE_MAXIMUM)
  for index := range branch {
    branch[index] = 'A'
  }
  section := "\n# " + string(branch) + "\n\nBody.\n"
  return section + section
}

// One contract whose two headings each spend the whole heading width.
func fixture_contract() (markdown string) {
  branch := make([]byte, requirements.HEADING_SIZE_MAXIMUM)
  leaf := make([]byte, requirements.HEADING_SIZE_MAXIMUM)
  for index := range branch {
    branch[index] = 'A'
    leaf[index] = 'B'
  }
  return "\n# " + string(branch) + "\n\n### " + string(leaf) + "\n\nBody.\n"
}

// Makes a chain that spends the whole width a path holds, which no contract file fits under.
func fixture_chain_full(t *testing.T, subject *fixture) (path string) {
  t.Helper()
  path = "w"
  fixture_directory(t, subject, path)
  for len(path)+len("/w") < PATH_SIZE_LIMIT {
    path += "/w"
    fixture_directory(t, subject, path)
  }
  return path
}

// PATH_SIZE_LIMIT is one byte past the widest path, thus a path inside the bound stands
// strictly below it.
const PATH_SIZE_LIMIT = filepath.PATH_SIZE_MAXIMUM + 1

// CONTRACT_READ_SKIP is how many reads stand before the contract test file: the Go file the walk
// found, and the contract markdown the check reads first.
const CONTRACT_READ_SKIP = 2

// Makes the deepest chain a package stands in: each step spends a separator and one byte, and the
// last one leaves room for the contract test file, which is the widest name the package owes. The
// root spends two bytes so that path lands on the last byte rather than one past it.
func fixture_chain(t *testing.T, subject *fixture) (path string) {
  t.Helper()
  path = "dd"
  fixture_directory(t, subject, path)
  spare := len("/") + len(requirements.TEST_NAME)
  for len(path)+len("/d")+spare < PATH_SIZE_LIMIT {
    path += "/d"
    fixture_directory(t, subject, path)
  }
  return path
}

// Every seed gets its own tree and loop, while the run stores stay shared: the test, like package
// main, allocates them once.
func fixture_new(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) (subject *fixture) {
  t.Helper()
  return fixture_sized(
    t,
    memory,
    format_state,
    command,
    directory,
    seed,
    NODE_CAPACITY,
    NODE_CONTENTS_BYTES,
  )
}

// A tree of the given size. One node's contents is its file capacity, so a fixture that needs a
// wide source asks for a small tree of wide nodes rather than a wide tree the host cannot hold.
func fixture_sized(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
  node_count int,
  contents_size int,
) (subject *fixture) {
  t.Helper()
  subject = &fixture{}
  nodes := make([]nbio.Sim_Node, node_count)
  for index := range nodes {
    nodes[index].Name = make([]byte, nbio.SIM_PATH_COMPONENT_BYTES_MAXIMUM)
    nodes[index].Contents = make([]byte, contents_size)
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
      Read:         harness.Read{Path: read_path_new()},
      Work_Queued:  harness.Runner_Work_Queued,
      Spread:       harness.Spread{Submit: harness.Check_Lane, Reap: harness.Lane_Oldest},
      Stopped:      harness.Runner_Stopped,
      Output:       nbio.Memory_To_Stream(&subject.Output),
      Error_Output: nbio.Memory_To_Stream(&subject.Errors),
      Clock:        nbio.Sim_Clock_To_Clock(&clocks[0]),
      Memory:       memory,
      Format:       format_state,
      Command:      command,
      Directory:    directory,
      Reports: harness.Reports{
        Report: harness.Report{
          Rule:    harness.RULE_TEXT_REFUSED,
          Tier:    harness.Report_Tier(harness.TIER_TWO),
          Path:    harness.PATH_TEXT_NONE,
          Message: harness.MESSAGE_OVERSIZED,
        },
        Summary: harness.Summary{
          Elapsed_Seconds: harness.ELAPSED_TEXT_NONE,
        },
      },
      Flags: harness.Run_Flag_Storage{
        Values: make(harness.Run_Flags, harness.RUN_FLAG_COUNT),
      },
    },
  }
  // Built once: a closure literal inside the pump would allocate on every pass.
  subject.Done = func() (finished bool) {
    return fixture_queued(subject)
  }
  fixture_file(t, subject, harness.CONFIGURATION_PATH, CONFIGURATION)
  fixture_file(t, subject, harness.MODULE_PATH, MODULE)
  return subject
}

// MODULE is the go.mod a fixture tree holds at its root.
const MODULE = "module local/fixture\n\ngo 1.26\n"

// CONFIGURATION is the smallest harness.json a run decodes.
const CONFIGURATION = `{"shared_component":"s","instrumentation_packages":[],` +
  `"opt_out_assertion_mandate_packages":[],` +
  `"opt_out_recursion_ban":[],"opt_out_interface_ban":[],"ignore":[],"word_replacements":{"util":[]}}`

func fixture_directory(t *testing.T, subject *fixture, path string) {
  t.Helper()
  for range SETUP_TRY_COUNT {
    nbio.Storage_Mkdir_At(
      subject.Loop.Storage,
      fixture_submit(subject),
      nbio.DIRECTORY_CURRENT,
      path,
      DIRECTORY_PERMISSIONS,
      fixture_retired,
    )
    if !fixture_wait(t, subject) {
      return
    }
  }
  t.Fatalf("setup never made the directory %s", path)
}

func fixture_file(t *testing.T, subject *fixture, path string, contents string) {
  t.Helper()
  file := fixture_open(t, subject, path)
  for range SETUP_TRY_COUNT {
    nbio.Storage_Write(
      subject.Loop.Storage,
      fixture_submit(subject),
      file,
      []byte(contents),
      0,
      harness.READ_TIMEOUT,
      fixture_retired,
    )
    if fixture_wait(t, subject) {
      continue
    }
    if subject.Setup.Completion.Data != len(contents) {
      t.Fatalf(
        "wrote %d bytes of %s, want %d", subject.Setup.Completion.Data,
        path, len(contents),
      )
    }
    fixture_close(t, subject, file)
    return
  }
  t.Fatalf("setup never wrote %s", path)
}

// Opens one fixture file, past whatever faults the seed draws.
func fixture_open(t *testing.T, subject *fixture, path string) (file nbio.File) {
  t.Helper()
  for range SETUP_TRY_COUNT {
    nbio.Storage_Open_At(
      subject.Loop.Storage,
      fixture_submit(subject),
      nbio.DIRECTORY_CURRENT,
      path,
      nbio.Open_At_Options{
        Access:      nbio.OPEN_WRITE_ONLY,
        Create:      true,
        Truncate:    true,
        Permissions: FILE_PERMISSIONS,
      },
      fixture_retired,
    )
    if !fixture_wait(t, subject) {
      return nbio.File(subject.Setup.Completion.Data)
    }
  }
  t.Fatalf("setup never opened %s", path)
  return 0
}

func fixture_close(t *testing.T, subject *fixture, file nbio.File) {
  t.Helper()
  nbio.IO_Close(subject.Loop, fixture_submit(subject), file, fixture_retired)
  fixture_wait(t, subject)
}

func fixture_submit(subject *fixture) (completion *nbio.Completion) {
  subject.Setup.Retired = false
  return &subject.Setup.Completion
}

// Reports whether one drive pass is over, which the run states of the phase it stands in.
func fixture_queued(subject *fixture) (finished bool) {
  phase := harness.Driven_Phase(subject.Runner.Progress.Phase)
  return bool(subject.Runner.Work_Queued(phase))
}

func fixture_retired(completion nbio.Completion_Handle) {
  (*setup)(unsafe.Pointer(completion)).Retired = true
}

func fixture_retired_read(subject *fixture) (retired bool) {
  return subject.Setup.Retired
}

func fixture_wait(t *testing.T, subject *fixture) (failed bool) {
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
  return subject.Setup.Completion.Error != nil
}

// Names the component a run walks the shared one, while the tree holds the harness.json every
// fixture starts from, thus a tree under any top-level directory stands in the shared layout. The
// node is written in place, since a write through the storage would spend a fault a fixture set.
func fixture_shared(subject *fixture, arguments []string) {
  if len(arguments) < 2 {
    return
  }
  top := arguments[1]
  for len(top) > 1 {
    if top[:2] != "./" {
      break
    }
    top = top[2:]
  }
  for index := range len(top) {
    if top[index] == '/' {
      top = top[:index]
      break
    }
  }
  // A top the JSON would have to escape, or none, names no component the run could state.
  if len(top) == 0 {
    return
  }
  for index := range len(top) {
    symbol := top[index]
    if symbol < ' ' {
      return
    }
    if symbol == '"' {
      return
    }
    if symbol == '\\' {
      return
    }
  }
  shaped := strings_replace(CONFIGURATION, `"shared_component":"s"`, `"shared_component":"`+top+`"`)
  for index := range subject.Simulation.Nodes {
    node := &subject.Simulation.Nodes[index]
    if !node.Used {
      continue
    }
    if string(node.Name[:node.Name_Count]) != harness.CONFIGURATION_PATH {
      continue
    }
    // The file the fixture started from names some component and holds the rest unchanged.
    held := string(node.Contents[:node.Contents_Count])
    opening := `{"shared_component":"`
    rest := CONFIGURATION[len(opening)+len("s"):]
    // A component of no byte is a fault the fixture wrote to be stated.
    if len(held) < len(opening)+len(rest)+1 {
      continue
    }
    if held[:len(opening)] != opening {
      continue
    }
    if held[len(held)-len(rest):] != rest {
      continue
    }
    if len(shaped) > len(node.Contents) {
      continue
    }
    node.Contents_Count = copy(node.Contents, shaped)
  }
}

// The root alternation of package main: run each recorded continuation, then one pump pass.
func fixture_run(t *testing.T, subject *fixture, arguments []string) {
  t.Helper()
  fixture_shared(subject, arguments)
  // Each run states its own lines: a memory stream holds what the run before it wrote, and
  // a fixture that read both would state a count no run reached.
  subject.Output.Cursor = 0
  subject.Errors.Cursor = 0
  subject.Runner.Arguments = arguments
  // A fixture wraps the loop after it builds the runner, thus each run binds the loop anew.
  subject.Runner.IO = subject.Loop
  harness.Main(
    harness.Main_Input{
      Phase: (*harness.Opening_Phase)(&subject.Runner.Progress.Phase),
      Start: &subject.Runner.Start,
      Clock: subject.Runner.Clock,
    },
  )
  for range PASS_COUNT {
    // The pump of package main stands its stop check here, thus a stopped run makes
    // one drive pass of its own before this one reads what it left.
    if subject.Runner.Stopped(harness.Pass_Phase(subject.Runner.Progress.Phase)) {
      fixture_status(t, subject)
      fixture_drained(t, subject)
      return
    }
    state := fixture_streams(subject)
    now := time.Clock_Now_Monotonic(subject.Runner.Clock)
    elapsed := harness.Elapsed(now - subject.Runner.Start)
    for harness.Rearm(
      harness.Rearm_Input{
        Memory:    subject.Runner.Memory,
        State:     state,
        Progress:  &subject.Runner.Progress,
        Elapsed:   elapsed,
        Arguments: subject.Runner.Arguments,
        Command:   subject.Runner.Command,
      },
    ) {
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

// The first byte of each store: a contract whose heading opens the file, and a test file whose
// leading declaration opens the byte past the package clause.
func fixture_first(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  subject := fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "first")
  fixture_file(t, subject, "first/a.go", CLEAN)
  // A heading at the first byte carries no blank line before it and no text after its
  // marker, thus its own offsets stand at the marker the file opens with.
  fixture_file(t, subject, "first/"+requirements.MARKDOWN_NAME, "# ")
  // A declaration at the byte past the package clause opens the leading slot at the first
  // offset any declaration stands at.
  fixture_file(t, subject, "first/"+requirements.TEST_NAME, "package a\nvar b=0\n")
  // A name of two bytes: the walk reads the shortest names a host spells beside the
  // widest ones, thus the name check stands on both ends of what one holds.
  fixture_directory(t, subject, "first/ab")
  fixture_run(t, subject, []string{"harness", "first"})
  // A heading of a level the doctrine names neither: the message states the fault and
  // both levels a host writes instead, which is the widest text one message opens with.
  subject = fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "level")
  fixture_file(t, subject, "level/a.go", CLEAN)
  fixture_file(t, subject, "level/"+requirements.MARKDOWN_NAME, "\n## Bad\n\nBody.\n")
  fixture_run(t, subject, []string{"harness", "level"})
  fixture_states(t, subject, `The heading level is not "#" or "###".`)
  fixture_run(t, subject, []string{"harness", "first"})
  // A declaration on the line the package clause opens: a test file states its leading
  // declarations wherever they stand, thus one sharing that line stands on line one.
  subject = fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "one")
  fixture_file(t, subject, "one/a.go", CLEAN)
  fixture_file(t, subject, "one/"+requirements.MARKDOWN_NAME, CONTRACT_MARKDOWN)
  fixture_file(t, subject, "one/"+requirements.TEST_NAME, "package a;var b=0\n")
  fixture_run(t, subject, []string{"harness", "one"})
  // A directory holding no Go file is no package, thus the run stands on a count of none
  // where one Go file would leave it at one.
  subject = fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "bare")
  fixture_file(t, subject, "bare/notes.md", "# Notes\n")
  fixture_run(t, subject, []string{"harness", "bare"})
  fixture_states(t, subject, "✓ all checks passed")
}

// A fix whose write takes no byte at all.
func fixture_empty(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  // A write that takes no byte of a fix leaves the file where it stood, thus the run fails
  // the fix step rather than writes the same bytes again with no end.
  subject := fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "fault")
  fixture_file(t, subject, "fault/a.go", UNFORMATTED)
  empty := Fault_Storage{Inner: subject.Loop.Storage, Loop: subject.Loop, Kind: FAULT_EMPTY}
  fault_wire(subject, &empty)
  fixture_run(t, subject, []string{"harness", "fault", "-fix"})
  fixture_failed(t, subject, "fix")
}

// The bytes a name spends: one no text spells, and a path of separators alone.
func fixture_bytes(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  // A name holding a byte no text spells stands, thus the line that names it carries that
  // byte rather than a name the run bent into text.
  subject := fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "bytes")
  fixture_file(t, subject, "bytes/\xff.go", UNFORMATTED)
  fixture_run(t, subject, []string{"harness", "bytes"})
  fixture_states(t, subject, "bytes/\xff.go: not formatted")
  // A path of separators alone names no directory once the run drops them, thus it fails
  // the walk the way an argument of no bytes does.
  fixture_run(t, subject, []string{"harness", "/"})
  fixture_failed(t, subject, "walk")
  fixture_run(t, subject, []string{"harness", "//"})
  fixture_failed(t, subject, "walk")
  // A host hands the run its own argument vector, thus a vector of no argument at all is
  // one the run states refused rather than reads a program name out of.
  fixture_run(t, subject, []string{})
  fixture_refused(t, subject)
  // A run opens on the status the run before it left, thus a clean package after a refused
  // command line states the pass rather than carries that refusal over.
  fixture_directory(t, subject, "whole")
  fixture_file(t, subject, "whole/a.go", CLEAN)
  fixture_file(t, subject, "whole/"+requirements.MARKDOWN_NAME, CONTRACT_MARKDOWN)
  fixture_file(t, subject, "whole/"+requirements.TEST_NAME, CONTRACT_TEST)
  fixture_run(t, subject, []string{"harness", "whole"})
  fixture_passed(t, subject)
}

// A fix the run wrote states no object: with json the line holds the tier two finding alone.
func fixture_fixed_json(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  subject := fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "both")
  fixture_file(t, subject, "both/a.go", CLEAN)
  fixture_file(t, subject, "both/"+requirements.MARKDOWN_NAME, CONTRACT_MARKDOWN)
  fixture_file(
    t, subject, "both/"+requirements.TEST_NAME, "package a\n\nfunc  Test_Other() {}\n",
  )
  fixture_run(t, subject, []string{"harness", "both", "-fix", "-json"})
  fixture_states(t, subject, `"rule":"requirements"`)
  fixture_silent(t, subject, `"rule":"unformatted"`)
}

// The command lines shared/std/cli refuses past a missing directory: an unknown flag, and a second
// positional the program declares no slot for.
func fixture_line(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  subject := fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "line")
  fixture_file(t, subject, "line/m.go", MAIN)
  fixture_run(t, subject, []string{"harness", "line", "-absent"})
  fixture_refused(t, subject)
  fixture_run(t, subject, []string{"harness", "line", "line"})
  fixture_refused(t, subject)
  // The error names what a host typed, thus a byte that steers a terminal stands as one
  // that does not rather than reaches whatever reads the line.
  fixture_run(t, subject, []string{"harness", "line", "-\x1b[31m"})
  fixture_refused(t, subject)
  written := fixture_errors(subject)
  for index := range len(written) {
    if written[index] < harness.NAME_BYTE_MINIMUM {
      if written[index] == '\n' {
        continue
      }
      t.Fatalf("a refused command line wrote %q", written)
    }
  }
  if !bool(strings.Contains(strings.Text(written), "?[31m")) {
    t.Fatalf("a refused command line wrote %q", written)
  }
}

// ESCAPED_COMPONENT_BYTES is one path component of bytes flat JSON escapes six-fold, at the
// widest one component holds.
const ESCAPED_COMPONENT_BYTES = 250

// ESCAPED_COMPONENT_COUNT carries the path far enough that its escaped form outgrows the object.
const ESCAPED_COMPONENT_COUNT = 3

// A path of bytes flat JSON escapes six-fold: the object outgrows its store, thus the run names
// the failed report rather than writes a line cut short of the object it states.
func fixture_outgrown(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  subject := fixture_new(t, memory, format_state, command, directory, seed)
  component := make([]byte, ESCAPED_COMPONENT_BYTES)
  for index := range component {
    component[index] = '<'
  }
  path := string(component)
  fixture_directory(t, subject, path)
  for range ESCAPED_COMPONENT_COUNT - 1 {
    path += "/" + string(component)
    fixture_directory(t, subject, path)
  }
  fixture_file(t, subject, path+"/m.go", MAIN)
  fixture_file(t, subject, path+"/a.go", UNFORMATTED)
  fixture_run(t, subject, []string{"harness", path, "-json"})
  fixture_failed(t, subject, "report")
  // The same path under a package that owes a contract: the contract finding names that
  // path too, thus the object the contract check states outgrows its store rather than
  // the one a Go diagnostic states.
  subject = fixture_new(t, memory, format_state, command, directory, seed)
  path = string(component)
  fixture_directory(t, subject, path)
  for range ESCAPED_COMPONENT_COUNT - 1 {
    path += "/" + string(component)
    fixture_directory(t, subject, path)
  }
  fixture_file(t, subject, path+"/a.go", CLEAN)
  fixture_file(t, subject, path+"/"+requirements.MARKDOWN_NAME, FAULTY_CONTRACT)
  fixture_run(t, subject, []string{"harness", path, "-json"})
  fixture_failed(t, subject, "report")
}

// The contract rules that guard the name a leaf builds: a heading of no word, one carrying a
// capital inside a word, two headings folding to one name, and a file stating no heading. Each
// states a tier two finding of its own, which the check alone states for none of them.
func fixture_folded(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  subject := fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "fold")
  fixture_file(t, subject, "fold/a.go", CLEAN)
  fixture_file(
    t,
    subject,
    "fold/"+requirements.MARKDOWN_NAME,
    "\n# A\n\n### B\n\nBody.\n\n# A B\n\nBody.\n",
  )
  fixture_file(t, subject, "fold/"+requirements.TEST_NAME, "package a\n")
  fixture_run(t, subject, []string{"harness", "fold", "-json"})
  fixture_states(t, subject, `"path":"fold/REQUIREMENTS.md","line":8,"column":0,"tier":2`)
  fixture_states(t, subject, `states the test name an earlier heading states`)
  subject = fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "cased")
  fixture_file(t, subject, "cased/a.go", CLEAN)
  fixture_file(t, subject, "cased/"+requirements.MARKDOWN_NAME, "\n# abC\n\nBody.\n")
  fixture_file(t, subject, "cased/"+requirements.TEST_NAME, "package a\n")
  fixture_run(t, subject, []string{"harness", "cased", "-json"})
  fixture_states(t, subject, `carries a capital after a lower case letter`)
  subject = fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "bare")
  fixture_file(t, subject, "bare/a.go", CLEAN)
  fixture_file(t, subject, "bare/"+requirements.MARKDOWN_NAME, "\n")
  fixture_file(t, subject, "bare/"+requirements.TEST_NAME, "package a\n")
  fixture_run(t, subject, []string{"harness", "bare", "-json"})
  fixture_states(t, subject, `"path":"bare/REQUIREMENTS.md","line":0,"column":0,"tier":2`)
  fixture_states(t, subject, `The file states no heading`)
}

// MAIN_REFUSED spells the package main clause in a file the parser refuses, thus the clause
// stands in the token run and the file proves nothing.
const MAIN_REFUSED = "package main\n\nfunc f( {}\n"

// A package main file the parser refuses exempts nothing: the run read a clause from a file it
// would not take, thus the package owes a contract until a host fixes the file.
func fixture_unproven(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  subject := fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "unproven")
  fixture_file(t, subject, "unproven/m.go", MAIN_REFUSED)
  fixture_run(t, subject, []string{"harness", "unproven"})
  fixture_states(t, subject, "unproven/REQUIREMENTS.md: The package has no REQUIREMENTS.md.")
  // The same file the parser takes: the clause stands proven, thus the package owes none.
  subject = fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "proven")
  fixture_file(t, subject, "proven/m.go", MAIN)
  fixture_run(t, subject, []string{"harness", "proven"})
  fixture_passed(t, subject)
  // One file the parser takes proves the clause for the package, thus a refused file
  // beside it bars nothing rather than carries its own fault onto the exemption.
  subject = fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "beside")
  fixture_file(t, subject, "beside/a.go", MAIN_REFUSED)
  fixture_file(t, subject, "beside/m.go", MAIN)
  fixture_run(t, subject, []string{"harness", "beside"})
  fixture_silent(t, subject, "has no REQUIREMENTS.md")
}

// Whether one path names a contract file, whichever directory it stands in.
func contract_named(path string) (named bool) {
  if bool(strings.Has_Suffix(strings.Text(path), requirements.MARKDOWN_NAME)) {
    return true
  }
  return bool(strings.Has_Suffix(strings.Text(path), requirements.TEST_NAME))
}

// A rename comes from the run that reads the declaration: a second run over another tree states
// its own names, thus a name the run before it declared renames nothing here.
func fixture_forgotten(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  subject := fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "declared")
  fixture_file(t, subject, "declared/a.go", DECLARATION)
  fixture_file(t, subject, "declared/b.go", REFERENCE)
  fixture_file(t, subject, "declared/"+requirements.MARKDOWN_NAME, CONTRACT_MARKDOWN)
  fixture_file(t, subject, "declared/"+requirements.TEST_NAME, CONTRACT_TEST)
  fixture_run(t, subject, []string{"harness", "declared", "-fix"})
  held := fixture_contents(t, subject, "declared/b.go")
  if held != REFERENCE_RENAMED {
    t.Fatalf("a run that declares the name left %q", held)
  }
  // A second tree that spells the name and declares it nowhere: the run states its own
  // names, thus the spelling stands as its host wrote it.
  subject = fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "spelled")
  fixture_file(t, subject, "spelled/b.go", REFERENCE)
  fixture_file(t, subject, "spelled/m.go", MAIN)
  fixture_run(t, subject, []string{"harness", "spelled", "-fix"})
  held = fixture_contents(t, subject, "spelled/b.go")
  if held != REFERENCE {
    t.Fatalf("a run renamed by a name the run before it declared, leaving %q", held)
  }
}

// CONTRACTED_PACKAGE_COUNT carries the package cursor past its first two slots, which a tree of
// one contracted package and a tree of two both leave short of.
const CONTRACTED_PACKAGE_COUNT = 3

// A tree of three packages that each hold a contract: the cursor the contract pass stands on
// walks every slot rather than the first two, thus each read states the package it belongs to.
func fixture_third(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  subject := fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "three")
  for index := range CONTRACTED_PACKAGE_COUNT {
    held := "three/" + fixture_name(index)
    fixture_directory(t, subject, held)
    fixture_file(t, subject, held+"/a.go", CLEAN)
    fixture_file(t, subject, held+"/"+requirements.MARKDOWN_NAME, CONTRACT_MARKDOWN)
    fixture_file(t, subject, held+"/"+requirements.TEST_NAME, CONTRACT_TEST)
  }
  fixture_run(t, subject, []string{"harness", "three"})
  fixture_passed(t, subject)
}

// The path of the file a run reads stands in a store the root owns, opening on a path every
// invariant of one admits.
func read_path_new() (path harness.Read_Path_Slice) {
  path = make(harness.Read_Path_Slice, 0, filepath.PATH_SIZE_MAXIMUM)
  return append(path, harness.READ_PATH_NONE...)
}

// HISTORY_FAILED is the answer of a git command that failed.
const HISTORY_FAILED = "!"

// HISTORY_REFS_LOCAL is a repository holding main alone.
const HISTORY_REFS_LOCAL = "refs/heads/main\n"

// HISTORY_REFS_REMOTE is a clone holding main and origin/main.
const HISTORY_REFS_REMOTE = "refs/heads/main\nrefs/remotes/origin/main\n"

// HISTORY_HASH_COUNT is the hex digits of one object name.
const HISTORY_HASH_COUNT = 40

// History_Script answers the git commands a run spawns: the branch HEAD names, the main refs that
// exist, and the log between them.
type History_Script struct {
  Head string
  Refs string
  Log  string
}

// Spawn_Procedure is the loop's spawn operation.
type Spawn_Procedure func(
  nbio.State,
  *nbio.Completion,
  nbio.Process_Request,
  time.Duration,
  nbio.Process_Callback,
)

// Answers each spawned git command from the script, as the git of a clone would.
func history_spawn(script History_Script) (spawn Spawn_Procedure) {
  return func(
    _ nbio.State,
    completion *nbio.Completion,
    request nbio.Process_Request,
    _ time.Duration,
    callback nbio.Process_Callback,
  ) {
    history_answer(script, completion, request, callback)
  }
}

// Answers one spawned git command from the script. An answer past the capture storage reports
// what the loop reports for one, with the storage full.
func history_answer(
  script History_Script,
  completion *nbio.Completion,
  request nbio.Process_Request,
  callback nbio.Process_Callback,
) {
  answer := script.Log
  switch request.Arguments[0] {
  case "rev-parse":
    answer = script.Head
  case "for-each-ref":
    answer = script.Refs
  }
  if answer == HISTORY_FAILED {
    callback(completion, nbio.Process_Result{Exit: 128}, nil)
    return
  }
  count := copy(request.Output_Storage, answer)
  var err error
  if count < len(answer) {
    err = nbio.Process_Output_Too_Large
  }
  callback(completion, nbio.Process_Result{Output: request.Output_Storage[:count]}, err)
}

// One log line: a commit named by one repeated hex digit, its parent or two, and its subject.
func history_line(name byte, merged bool, subject string) (line string) {
  parents := fixture_repeat("1", HISTORY_HASH_COUNT)
  if merged {
    parents += " " + fixture_repeat("2", HISTORY_HASH_COUNT)
  }
  return fixture_repeat(string(name), HISTORY_HASH_COUNT) + " " + parents + "\t" + subject + "\n"
}

// A log of every shape a subject takes to each rule, at each edge of the bounds a rule reads, and
// of every line git never writes in the form the run asked for.
func history_log() (log string) {
  widest := vcs.SUBJECT_TEXT_SIZE_MAXIMUM
  for _, subject := range []string{
    "a",
    "ab",
    fixture_repeat("x", widest),
    fixture_repeat("x", widest+1),
    "feat: a",
    "Bad",
    "feat: " + fixture_repeat("x", vcs.SUBJECT_SIZE_MAXIMUM-len("feat: ")),
    "fixup! a",
    "fix: address review comment",
    "fix: review nit",
    vcs.PULL_MERGE_OPENING,
    "Merge pull request #1 from o/b",
    "Merge pull request #1 frox",
    "Merge pull request #1 from \x01",
    vcs.PULL_MERGE_OPENING + fixture_repeat("1", widest-len(vcs.PULL_MERGE_OPENING)),
    vcs.SHA_MERGE_OPENING,
    "Merge abc1234 into def5678",
    "Merge \x01",
    "Merge \x02",
    "Merge \xff",
    vcs.SHA_MERGE_OPENING + fixture_repeat("x", widest-len(vcs.SHA_MERGE_OPENING)),
    "Revert \"a\"",
    "Revert \"a",
    "feat: \x02",
    "feat: \xff",
    "feat:  a",
  } {
    log += history_line('a', false, subject)
  }
  for _, subject := range []string{
    "a",
    "ab",
    fixture_repeat("x", widest),
    fixture_repeat("x", vcs.SUBJECT_SIZE_MAXIMUM),
    "Merge branch 'x'",
    "Add 'a/' from commit 'b'",
    "Merge commit 'a' as 'b'",
    "Add '",
    "Add '" + fixture_repeat("x", vcs.SUBJECT_SIZE_MAXIMUM-len("Add '")),
    "Merge pull request #1 from o/b",
  } {
    log += history_line('b', true, subject)
  }
  malformed := "\na\nab\nzzzzzzzzzz 1\ta\n0123456789\n0123456789abc 1\n012345678 1\ta\n"
  return log + malformed + "0123456789 1\t\n"
}

// The history a run judges past its packages: each answer git gives, in text and in flat JSON, a
// log of every subject shape, one past its storage, one that fills it, and one whose last newline
// closes it.
func fixture_history(
  t *testing.T,
  memory harness.Memory_Handle,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  subject := fixture_new(t, memory, format_state, command, directory, seed)
  subject.Output = nbio.Stream_Memory{Memory: make([]byte, WIDE_OUTPUT_BYTES)}
  subject.Runner.Output = nbio.Memory_To_Stream(&subject.Output)
  fixture_directory(t, subject, "hist")
  fixture_file(t, subject, "hist/m.go", MAIN)
  size := harness.HISTORY_STORAGE_SIZE
  head := history_line('c', false, "")
  head = head[:len(head)-1]
  filled := head + fixture_repeat("x", size-len(head))
  closed := head + fixture_repeat("x", size-len(head)-1) + "\n"
  log := history_log()
  // A package that owes a contract, thus each run states a contract finding before the history
  // it walks, on the continuation the history of the run before it left.
  fixture_directory(t, subject, "hist/lib")
  fixture_file(t, subject, "hist/lib/a.go", CLEAN)
  for _, script := range []History_Script{
    {Head: HISTORY_FAILED},
    {Head: "main\n"},
    {Head: "feature\n", Refs: HISTORY_FAILED},
    {Head: "feature\n", Refs: ""},
    {Head: "feature\n", Refs: "refs/remotes/origin/main\n", Log: HISTORY_FAILED},
    {Head: "feature\n", Refs: HISTORY_REFS_REMOTE, Log: filled + "x"},
    {Head: "feature\n", Refs: HISTORY_REFS_REMOTE, Log: filled},
    {Head: "feature\n", Refs: HISTORY_REFS_REMOTE, Log: closed},
    {Head: "feature\n", Refs: HISTORY_REFS_LOCAL, Log: log},
  } {
    subject.Loop.Spawn = history_spawn(script)
    fixture_run(t, subject, []string{"harness", "hist"})
  }
  // The same log as flat JSON, which each finding writes as one object.
  fixture_run(t, subject, []string{"harness", "hist", "-json"})
  found := `"path":"\u003cgit:aaaaaaaaaa\u003e"`
  if !fixture_holds(fixture_written(subject), found) {
    t.Fatalf("a history run wrote no finding at %s", found)
  }
  // A tree whose last finding names a name, thus the history walks on the report it left.
  subject = fixture_new(t, memory, format_state, command, directory, seed)
  fixture_directory(t, subject, "named")
  fixture_file(t, subject, "named/m.go", "package main\n\nvar max_line = 1\n\nfunc main() {}\n")
  subject.Loop.Spawn = history_spawn(
    History_Script{Head: "feature\n", Refs: HISTORY_REFS_LOCAL, Log: log},
  )
  fixture_run(t, subject, []string{"harness", "named", "-json"})
  if !fixture_holds(fixture_written(subject), found) {
    t.Fatalf("a named history run wrote no finding at %s", found)
  }
}

// Reports whether a text of any length holds a fragment.
func fixture_holds(text string, fragment string) (held bool) {
  for start := 0; start+len(fragment) < len(text)+1; start++ {
    if text[start:start+len(fragment)] == fragment {
      return true
    }
  }
  return false
}

// The parts of the runner one pass of the pump works on.
func fixture_streams(subject *fixture) (state harness.Streams) {
  return harness.Streams{
    Loop:           &subject.Runner.Loop,
    Stage:          &subject.Runner.Stage,
    Walk:           &subject.Runner.Walk,
    Package_Cursor: &subject.Runner.Package_Cursor,
    Orphans:        &subject.Runner.Orphan_Cursor,
    Findings:       &subject.Runner.Findings,
    Flags:          subject.Runner.Flags,
    Files:          &subject.Runner.Files,
    File_Cursor:    &subject.Runner.File_Cursor,
    Read:           &subject.Runner.Read,
    Format:         subject.Runner.Format,
    Directory:      &subject.Runner.Directory,
    Outcome:        &subject.Runner.Outcome,
    Report:         &subject.Runner.Report,
    Summary:        &subject.Runner.Summary,
    Finding:        &subject.Runner.Finding,
    Fix_State:      &subject.Runner.Fix_State,
    Tally:          &subject.Runner.Tally,
    History:        &subject.Runner.History,
    Output:         subject.Runner.Output,
    Errors:         subject.Runner.Error_Output,
    Spread:         subject.Runner.Spread,
  }
}
