// Package simulation_test calls setup.Main with a deferred model fabricated from each seed
// and asserts the mirror's properties hold for any seed: convergence and idempotency, pruning
// of an ignored subtree, and rewriting only files that differ.
package simulation_test

import (
	"container/list"
	"errors"
	"path/filepath"
	"testing"

	setup "local/james-orcales/setup/internal"
	shared_bytes "local/james-orcales/shared/bytes"
	invariant "local/james-orcales/shared/invariant/default"
	sysio "local/james-orcales/shared/io"
	"local/james-orcales/shared/random/prng"
	shared_slices "local/james-orcales/shared/slices"
	shared_strings "local/james-orcales/shared/strings"
	systime "local/james-orcales/shared/time"
)

// The destination directory, relative to the source root, pruned from the source walk so the
// mirror never copies its own output back into itself.
const HARNESS_DESTINATION = "dest"

// A top-level entry marked ignored, so the harness can prove the mirror prunes it.
const HARNESS_IGNORED = "e0"

// An empty directory can reach the directory-path limit without forcing a child past that limit.
const HARNESS_EMPTY_DIRECTORY = "g0"

// Content written over a destination file to force a difference no random source file
// realistically matches, so a later run rewrites exactly the mutated files.
const HARNESS_POISON = "harness-poison-differs-from-any-generated-file"

// The normal test owns the complete sweep so fuzz registration does not run it a second time.
const HARNESS_CORPUS = 576

// The callback adapter uses next-tick delivery, so 64 grains expose an intentional stall quickly.
const HARNESS_PUMP_DURATION_MAX = 64 * systime.NANOSECOND

// One seed gives the fuzz engine its input shape before it generates values beyond the sweep.
const HARNESS_FUZZ_SEED uint64 = 0

// The fixed seeds that freeze each stream topology without coupling the streams together.
const HARNESS_STREAM_SNAPSHOT_SEEDS = 3

// A separate mutation stream keeps new configuration axes from moving the files that a pinned
// seed poisons.
const HARNESS_MUTATION_SALT = 0x9e3779b97f4a7c15

// The mirror target lands before font installation in the fixed bootstrap order.
const HARNESS_IO_TARGET_MIRROR = "mirror"

// The font target crosses the separate large-file read and write loops.
const HARNESS_IO_TARGET_FONT = "font"

// A required site must stop setup after its selected IO fault.
const HARNESS_IO_SITE_REQUIRED = "required"

// A probe site must reject its result and continue with the required work.
const HARNESS_IO_SITE_PROBE = "probe"

// A gitignore site must stop the mirror before an unknown path can be copied.
const HARNESS_IO_SITE_GITIGNORE = "gitignore"

// Vendored font sources identify the separate large-file copy path.
const HARNESS_FONT_SOURCE_PART = "/third_party/iosevka_nerd_font_mono/"

// Harness_Timeline submits model callbacks through one seeded shared IO loop.
type Harness_Timeline struct {
	// IO schedules each model callback without giving the model the Driver.
	IO sysio.IO
	// Callbacks catches a root that stops before every scheduled model callback retires.
	Callbacks *list.List
	// Submission identifies the one callback that the test root armed on the seeded loop.
	Submission *list.Element
}

// Harness_Root is the test composition root that alone owns timeline advancement.
type Harness_Root struct {
	// Timeline is the submit-only surface that the model callback adapters use.
	Timeline *Harness_Timeline
	// Driver advances the seeded loop only from the test root.
	Driver sysio.Driver
}

// Harness_Configuration is the fixed feature subset one seed uses for its complete timeline.
type Harness_Configuration struct {
	Home_Directory     setup.Home_Directory
	Operating_System   setup.Operating_System
	Cargo_Directory    setup.Cargo_Directory
	Data_Directory     setup.Data_Directory
	Color              setup.Console_Color
	Effective_User     setup.Effective_User_Identifier
	Installed          bool
	Version_Matches    bool
	Failure            string
	Dotfile_Bytes      int
	Probe_Output_Bytes int
	Ghostty_Verified   bool
	Empty_Version      bool
	Path_Boundary      int
	Path_Aliases       map[string]string
	Clock_Epoch        systime.Moment
	IO_Chunk_Bytes     int
	IO_Fault           string
	IO_Fault_Target    string
	IO_Fault_Site      string
	Dotfile_Byte       byte
	Probe_Output_Byte  byte
	Profile_File       bool
	Configuration_File bool
	Ignored_File       bool
	Boundary_File      bool
	Boundary_Directory bool
	IO_Witness         bool
	Nested_File        bool
}

// Harness_Configuration_Streams gives each configuration axis an independent sequence.
type Harness_Configuration_Streams struct {
	Operating_System   prng.Generator
	Cargo              prng.Generator
	Data               prng.Generator
	Color              prng.Generator
	Installed          prng.Generator
	Identifier         prng.Generator
	Failure            prng.Generator
	Version            prng.Generator
	Home               prng.Generator
	Dotfile_Bytes      prng.Generator
	Probe_Output_Bytes prng.Generator
	Ghostty_Verified   prng.Generator
	Empty_Version      prng.Generator
	Path_Boundary      prng.Generator
	Clock_Epoch        prng.Generator
	IO_Chunk_Bytes     prng.Generator
	IO_Fault           prng.Generator
	IO_Fault_Target    prng.Generator
	Dotfile_Content    prng.Generator
	Probe_Output       prng.Generator
	IO_Fault_Site      prng.Generator
	Profile_File       prng.Generator
	Configuration_File prng.Generator
	Ignored_File       prng.Generator
	Boundary_File      prng.Generator
	Boundary_Directory prng.Generator
	IO_Witness         prng.Generator
	Nested_File        prng.Generator
}

// Harness_File_Streams gives each modeled payload an independent sequence.
type Harness_File_Streams struct {
	Profile         prng.Generator
	Configuration   prng.Generator
	Ignored         prng.Generator
	Boundary        prng.Generator
	Nested_Boundary prng.Generator
	IO_Witness      prng.Generator
	Font            prng.Generator
}

// TestMain registers the complete setup invariant tree before the seed sweep starts.
func TestMain(m *testing.M) {
	invariant.Run_Test_Main(m, "../**")
}

// New_harness_root derives callback delivery and advancement from the same seed.
func new_harness_root(seed uint64) (root Harness_Root) {
	timeline, driver, _ := sysio.New_Sim(seed)
	root.Timeline = &Harness_Timeline{IO: timeline, Callbacks: list.New()}
	root.Driver = driver
	return root
}

// Harness_defer keeps callback delivery outside the submission call stack.
func harness_defer(timeline *Harness_Timeline, action func()) {
	timeline.Callbacks.PushBack(action)
}

// Harness_defer_late forces a second retirement behind one root rearm boundary.
func harness_defer_late(timeline *Harness_Timeline, action func()) {
	harness_defer(timeline, action)
}

// Harness_arm_callback gives only the queue head to the seeded loop for this pump pass.
func harness_arm_callback(timeline *Harness_Timeline) {
	if timeline.Submission != nil {
		return
	}
	element := timeline.Callbacks.Front()
	if element == nil {
		return
	}
	completion := &sysio.Completion{}
	timeline.Submission = element
	timeline.IO.Next_Tick(completion, func(_ *sysio.Completion) {
		action := timeline.Callbacks.Remove(timeline.Submission).(func())
		timeline.Submission = nil
		action()
	}, sysio.NEXT_TICK_VSR)
}

// Harness_defer_retirement gives only a duplicate the later simulated grain.
func harness_defer_retirement(
	timeline *Harness_Timeline, duplicate bool, action func(),
) {
	if duplicate {
		harness_defer_late(timeline, action)
		return
	}
	harness_defer(timeline, action)
}

// The harness keeps setup's larger file and batch limits while each shared operation stays
// inside its more narrow boundary.
func Test_Shared_Algorithm_Adapters(t *testing.T) {
	t.Parallel()
	repeated := repeat_text("ab", shared_strings.TEXT_SIZE_MAXIMUM)
	if len(repeated) != 2*shared_strings.TEXT_SIZE_MAXIMUM {
		t.Fatalf("repeated text size = %d, want %d", len(repeated), 8192)
	}
	source := []byte(repeated)
	clone := clone_bytes(source)
	clone[0] = 'x'
	if source[0] != 'a' {
		t.Fatal("the byte clone aliases its source")
	}
	lines := []string{repeat_text("x", 4000), repeat_text("y", 4000)}
	result := split_lines(newline_delimited_text(lines))
	if !shared_slices.Equal(result, lines) {
		t.Fatalf("newline-delimited text = %v, want the input lines", result)
	}
}

// Fuzz_Main calls Main over the deferred file model that each seed draws.
func Fuzz_Main(f *testing.F) {
	f.Add(HARNESS_FUZZ_SEED)
	f.Fuzz(func(t *testing.T, seed uint64) {
		drive(t, seed)
	})
}

// Calls setup.Main over one seeded model and asserts every mirror invariant: a second run over
// the freshly mirrored tree writes nothing (convergence, idempotency, no rewrite of an equal
// file), the ignored subtree never reaches the destination (prune), and after some
// destination files are made to differ, a run rewrites exactly those (overwrite).
func drive(t *testing.T, seed uint64) {
	configuration := harness_configuration(seed)
	diagnostics := &Harness_Diagnostic_Witness{
		Expected: harness_io_fault_diagnostic(
			configuration.IO_Fault, configuration.IO_Fault_Target,
			configuration.IO_Fault_Site),
	}
	console := sysio.Stream{}
	if configuration.IO_Fault != "" {
		console = harness_diagnostic_stream(diagnostics)
	}
	clock, _ := systime.Virtual_Clock_To_Clock(
		systime.Virtual_Clock{
			Resolution: systime.NANOSECOND, Epoch: configuration.Clock_Epoch})
	system, root, observation := simulation_file_system(seed, configuration)
	input := &setup.Main_Input{
		Environment: &setup.Environment_Input{
			Clock: clock, Console: console,
			Effective_User_Identifier: configuration.Effective_User,
			Color:                     configuration.Color,
			Home_Directory:            configuration.Home_Directory,
			Operating_System:          configuration.Operating_System,
			Cargo_Directory:           configuration.Cargo_Directory,
			Data_Directory:            configuration.Data_Directory,
		},
		IO: system,
	}
	// The first run must succeed for every well-formed seed — with no faults injected, Main
	// fails only on a Plan or write error, neither reachable here. If that ever changes this
	// catches it instead of skipping the seed blind. When faults land it becomes an explicit,
	// reason-carrying, mostly-false skip — never a silent return.
	status := harness_main_status(t, input, root)
	if status == setup.EXIT_USAGE {
		return
	}
	if status == setup.EXIT_FAILURE {
		if configuration.IO_Fault_Site == HARNESS_IO_SITE_PROBE {
			t.Fatalf(
				"the recoverable %q IO fault stopped setup",
				configuration.IO_Fault)
		}
		harness_assert_failure(t, configuration, observation, diagnostics.Observed)
		return
	}
	if configuration.Failure != "" {
		t.Fatalf("the setup ignored the selected %q fault", configuration.Failure)
	}
	if configuration.IO_Fault != "" {
		if configuration.IO_Fault_Site == HARNESS_IO_SITE_PROBE {
			harness_assert_recovery(t, configuration, observation)
		} else {
			t.Fatalf(
				"the setup ignored the selected %q "+
					"IO fault", configuration.IO_Fault)
		}
	}
	if status != setup.EXIT_SUCCESS {
		t.Fatal("the first mirror run over a fault-free tree failed")
	}
	harness_verify_success(t, seed, configuration, observation, system, root, input)
}

// A successful first run must converge, prune ignored input, and rewrite each mutation.
func harness_verify_success(
	t *testing.T, seed uint64, configuration Harness_Configuration,
	observation *Harness_Observation, system sysio.IO, root Harness_Root,
	input *setup.Main_Input,
) {
	t.Helper()
	harness_assert_short_io(t, configuration, observation)
	writes := harness_count_writes(&system, configuration)
	input.IO = system
	second_status := harness_main_status(t, input, root)
	if second_status != 0 {
		t.Fatalf("the second mirror run failed with status %d", second_status)
	}
	if *writes != 0 {
		t.Fatalf("a converged mirror rewrote %d files on a repeat run", *writes)
	}
	harness_assert_pruned(t, &system, configuration)
	mutated := harness_mutate(&system, configuration, seed)
	*writes = 0
	if harness_main_status(t, input, root) != 0 {
		t.Fatal("the mirror run after mutation failed")
	}
	if *writes != mutated {
		t.Fatalf("rewrote %d files, want the %d made to differ", *writes, mutated)
	}
}

// The simulation root owns the same continuation and Driver alternation as the binary root.
func harness_main_status(
	t *testing.T, input *setup.Main_Input, root Harness_Root,
) (status setup.Exit_Code) {
	t.Helper()
	runner := setup.Main(input)
	for !runner.Stopped() {
		for runner.Rearm() {
		}
		if runner.Stopped() {
			break
		}
		harness_arm_callback(root.Timeline)
		completed, drive_err := root.Driver.Run_Until(func() (finished bool) {
			if runner.Stopped() {
				return true
			}
			if runner.Work_Queued() {
				return true
			}
			if root.Timeline.Submission == nil {
				if root.Timeline.Callbacks.Len() != 0 {
					return true
				}
			}
			return false
		}, HARNESS_PUMP_DURATION_MAX)
		if drive_err != nil {
			return harness_drive_failure(t, input)
		}
		if !completed {
			return harness_drive_failure(t, input)
		}
	}
	if root.Timeline.Callbacks.Len() != 0 {
		t.Fatalf(
			"the simulation stopped with %d pending callbacks",
			root.Timeline.Callbacks.Len(),
		)
	}
	root.Driver.Deinit()
	return runner.Status()
}

// Harness_drive_failure keeps the simulation root's diagnostic equal to the binary root.
func harness_drive_failure(
	t *testing.T, input *setup.Main_Input,
) (status setup.Exit_Code) {
	t.Helper()
	message := []byte(setup.IO_OPERATION_INCOMPLETE + "\n")
	written, write_err := sysio.Write(input.Environment.Console, message)
	if write_err != nil {
		t.Fatal("the simulation root could not write the driver diagnostic")
	}
	if written != int64(len(message)) {
		t.Fatal("the simulation root wrote a partial driver diagnostic")
	}
	return setup.EXIT_FAILURE
}

// A probe fault is useful only when setup rejects the probe and does its required work.
func harness_assert_recovery(
	t *testing.T, configuration Harness_Configuration, observation *Harness_Observation,
) {
	t.Helper()
	if !observation.IO_Fault_Observed {
		t.Fatalf(
			"the setup did not reach the recoverable %q IO fault",
			configuration.IO_Fault)
	}
	if !observation.IO_Fault_Recovery {
		t.Fatalf(
			"the setup did not recover from the %q IO fault at %q",
			configuration.IO_Fault, configuration.IO_Fault_Target)
	}
}

// A selected fault is useful only when it lands and stops each later IO submission.
func harness_assert_failure(
	t *testing.T, configuration Harness_Configuration, observation *Harness_Observation,
	diagnostic_observed bool,
) {
	t.Helper()
	if configuration.IO_Fault != "" {
		if !observation.IO_Fault_Observed {
			t.Fatalf(
				"the setup did not reach the selected %q IO fault",
				configuration.IO_Fault)
		}
		if observation.IO_Operation_After_Fault {
			t.Fatalf(
				"the setup submitted IO after the selected %q IO fault",
				configuration.IO_Fault)
		}
		if observation.IO_Fault_Cleanup {
			if !text_contains(configuration.IO_Fault, "did-not-retire") {
				t.Fatalf("setup did not close after the selected %q IO fault",
					configuration.IO_Fault)
			}
		}
		if !diagnostic_observed {
			t.Fatalf(
				"the %q IO fault did not produce diagnostic %q",
				configuration.IO_Fault,
				harness_io_fault_diagnostic(
					configuration.IO_Fault, configuration.IO_Fault_Target,
					configuration.IO_Fault_Site))
		}
		return
	}
	if configuration.Failure == "" {
		t.Fatalf("the setup failed without a selected fault: %+v", configuration)
	}
	if !observation.Failure_Observed {
		t.Fatalf(
			"the setup did not reach the selected %q fault", configuration.Failure)
	}
	if observation.Operation_After_Failure {
		t.Fatalf(
			"the setup submitted IO after the selected %q fault", configuration.Failure)
	}
}

// Harness_Diagnostic_Witness recognizes one bounded guard result without retaining setup logs.
type Harness_Diagnostic_Witness struct {
	Expected string
	Observed bool
}

// Write records whether one rendered log line contains the selected guard result.
func (witness *Harness_Diagnostic_Witness) Write(content []byte) (count int, err error) {
	if !witness.Observed {
		witness.Observed = bounded_bytes_contains(content, []byte(witness.Expected))
	}
	return len(content), nil
}

// The stream keeps diagnostic observation on the same boundary as production output.
func harness_diagnostic_stream(witness *Harness_Diagnostic_Witness) (stream sysio.Stream) {
	return sysio.Stream{
		Data: witness,
		Procedure: func(
			data any, mode sysio.Stream_Mode, content []byte,
			_ int64, _ sysio.Seek_From,
		) (count int64, err error) {
			if mode == sysio.STREAM_MODE_QUERY {
				return int64(sysio.Mode_Set_Add(0, sysio.STREAM_MODE_WRITE)), nil
			}
			// Other operations cannot contribute a rendered diagnostic.
			if mode != sysio.STREAM_MODE_WRITE {
				return 0, sysio.Stream_Empty
			}
			diagnostic := data.(*Harness_Diagnostic_Witness)
			written, write_err := diagnostic.Write(content)
			return int64(written), write_err
		},
	}
}

// Each search part stays inside the shared byte domain while overlap retains boundary matches.
func bounded_bytes_contains(source []byte, target []byte) (contained bool) {
	if len(target) == 0 {
		return true
	}
	if len(target) > shared_bytes.SLICE_SIZE_MAXIMUM {
		return false
	}
	for offset := 0; offset < len(source); {
		end := offset + shared_bytes.SLICE_SIZE_MAXIMUM
		if end > len(source) {
			end = len(source)
		}
		if shared_bytes.Contains(
			shared_bytes.Slice(source[offset:end]), shared_bytes.Slice(target),
		) {
			return true
		}
		if end == len(source) {
			return false
		}
		offset = end - len(target) + 1
	}
	return false
}

// The diagnostic proves the exact guard rejected a malicious completion before a later limit did.
func harness_io_fault_diagnostic(fault string, target string, site string) (diagnostic string) {
	if text_contains(fault, "did-not-retire") {
		return setup.IO_OPERATION_INCOMPLETE
	}
	if site == HARNESS_IO_SITE_GITIGNORE {
		return "plan failed"
	}
	if text_has_prefix(fault, "spawn-") {
		return "direnv build failed"
	}
	switch fault {
	case "status-error":
		return "simulated status failure"
	case "open-error":
		return "simulated open failure"
	case "create-error":
		return "simulated create failure"
	case "make-directory-error":
		return "simulated make-directory failure"
	case "read-directory-error":
		return "simulated read-directory failure"
	case "read-error":
		return "simulated read failure"
	case "read-did-not-retire", "read-retired-twice",
		"read-negative-count", "read-over-count":
		return harness_read_io_fault_diagnostic(fault, target)
	case "write-error":
		return "simulated write failure"
	case "write-did-not-retire":
		return setup.IO_OPERATION_INCOMPLETE
	case "write-retired-twice":
		if target == HARNESS_IO_TARGET_FONT {
			return "the font write retired more than once"
		}
		return "the file write retired more than once"
	case "write-zero-count", "write-negative-count":
		if target == HARNESS_IO_TARGET_FONT {
			return "the font write made no progress"
		}
		return "the file write made no progress"
	case "write-over-count":
		if target == HARNESS_IO_TARGET_FONT {
			return "the font write returned an invalid byte count"
		}
		return "the file write returned an invalid byte count"
	case "close-error":
		return "simulated close failure"
	case "close-did-not-retire":
		return setup.IO_OPERATION_INCOMPLETE
	case "close-retired-twice":
		return "the file close retired more than once"
	}
	return ""
}

// The font copy and dotfile reader name their separate completion guards.
func harness_read_io_fault_diagnostic(fault string, target string) (diagnostic string) {
	prefix := "the file read "
	if target == HARNESS_IO_TARGET_FONT {
		prefix = "the font read "
	}
	switch fault {
	case "read-did-not-retire":
		return setup.IO_OPERATION_INCOMPLETE
	case "read-retired-twice":
		return prefix + "retired more than once"
	case "read-negative-count":
		return prefix + "returned a negative byte count"
	case "read-over-count":
		return prefix + "returned an invalid byte count"
	}
	return ""
}

// Each short-completion configuration must make both file loops submit more than one operation.
func harness_assert_short_io(
	t *testing.T, configuration Harness_Configuration, observation *Harness_Observation,
) {
	t.Helper()
	if configuration.IO_Chunk_Bytes == setup.DOTFILE_PAYLOAD_BYTES_MAX {
		return
	}
	if !observation.Short_Read_Observed {
		t.Fatal("the selected short IO limit produced no short read")
	}
	if !observation.Short_Write_Observed {
		t.Fatal("the selected short IO limit produced no short write")
	}
}

// Draws each configuration axis from its own stream, then holds the result for the whole run.
func harness_configuration(seed uint64) (configuration Harness_Configuration) {
	streams := harness_configuration_streams(seed)
	configuration.Home_Directory = setup.Home_Directory(harness_home(
		prng.Generator_Element(&streams.Home, []int{2, 4, 64})))
	configuration.Operating_System = prng.Generator_Element(
		&streams.Operating_System,
		[]setup.Operating_System{"aix", "linux", "darwin", "freebsd", "dragonfly"})
	configuration.Cargo_Directory = setup.Cargo_Directory(prng.Generator_Element(
		&streams.Cargo, []string{"", "x", "xx", repeat_text("x", 64)}))
	configuration.Data_Directory = setup.Data_Directory(prng.Generator_Element(
		&streams.Data, []string{"", "x", "xx", repeat_text("x", 64)}))
	configuration.Color = setup.Console_Color(prng.Generator_Boolean(&streams.Color))
	configuration.Installed = prng.Generator_Boolean(&streams.Installed)
	configuration.Version_Matches = prng.Generator_Boolean(&streams.Version)
	configuration.Effective_User = prng.Generator_Element(
		&streams.Identifier, []setup.Effective_User_Identifier{
			0, 1, 2, setup.EFFECTIVE_USER_IDENTIFIER_MAX,
		})
	configuration.Failure = prng.Generator_Sample(
		&streams.Failure, prng.New_Distribution(
			[]string{
				"", "direnv", "dotfiles", "defaults", "fonts", "nvim", "fzf",
				"maddox",
				"rust", "rust-link", "jj", "rg", "fd", "ghostty",
			},
			[]uint64{8, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
		),
	)
	configuration.Dotfile_Bytes = prng.Generator_Sample(
		&streams.Dotfile_Bytes, prng.New_Distribution(
			[]int{-1, 1, 2, setup.DOTFILE_PAYLOAD_BYTES_MAX},
			[]uint64{16, 16, 16, 1},
		),
	)
	configuration.Probe_Output_Bytes = prng.Generator_Element(
		&streams.Probe_Output_Bytes, []int{0, 1, 2})
	configuration.Ghostty_Verified = prng.Generator_Boolean(&streams.Ghostty_Verified)
	configuration.Empty_Version = prng.Generator_Boolean(&streams.Empty_Version)
	configuration.Path_Boundary = prng.Generator_Element(
		&streams.Path_Boundary, []int{0, 1, 2})
	configuration.Clock_Epoch = systime.Moment(int64(
		prng.Generator_Next(&streams.Clock_Epoch)))
	configuration.IO_Chunk_Bytes = prng.Generator_Sample(
		&streams.IO_Chunk_Bytes, prng.New_Distribution(
			[]int{1, 2, setup.DOTFILE_PAYLOAD_BYTES_MAX},
			[]uint64{1, 1, 16},
		),
	)
	configuration.IO_Fault = harness_io_fault(&streams.IO_Fault)
	configuration.IO_Fault_Target = prng.Generator_Element(
		&streams.IO_Fault_Target,
		[]string{HARNESS_IO_TARGET_MIRROR, HARNESS_IO_TARGET_FONT})
	configuration.Dotfile_Byte = prng.Generator_Element(
		&streams.Dotfile_Content, []byte{'a', 'b', 'x', 'y'})
	configuration.Probe_Output_Byte = prng.Generator_Element(
		&streams.Probe_Output, []byte{'a', 'b', 'x', 'y'})
	configuration.IO_Fault_Site = prng.Generator_Sample(
		&streams.IO_Fault_Site, prng.New_Distribution(
			[]string{
				HARNESS_IO_SITE_REQUIRED, HARNESS_IO_SITE_PROBE,
				HARNESS_IO_SITE_GITIGNORE,
			},
			[]uint64{1, 3, 2},
		))
	harness_fixture_configuration(&streams, &configuration)
	configuration.Path_Aliases = map[string]string{}
	return harness_align(configuration)
}

// Each fixture feature draws once and stays fixed for the complete run.
func harness_fixture_configuration(
	streams *Harness_Configuration_Streams, configuration *Harness_Configuration,
) {
	configuration.Profile_File = prng.Generator_Boolean(&streams.Profile_File)
	configuration.Configuration_File = prng.Generator_Boolean(
		&streams.Configuration_File)
	configuration.Ignored_File = prng.Generator_Boolean(&streams.Ignored_File)
	configuration.Boundary_File = prng.Generator_Boolean(&streams.Boundary_File)
	configuration.Boundary_Directory = prng.Generator_Boolean(
		&streams.Boundary_Directory)
	configuration.IO_Witness = prng.Generator_Boolean(&streams.IO_Witness)
	configuration.Nested_File = prng.Generator_Boolean(&streams.Nested_File)
}

// Fault-free runs retain enough weight for convergence while each fault remains common.
func harness_io_fault(generator *prng.Generator) (fault string) {
	return prng.Generator_Sample(generator, prng.New_Distribution(
		[]string{
			"", "status-error", "open-error", "create-error",
			"make-directory-error", "read-directory-error", "read-error",
			"read-did-not-retire", "read-retired-twice", "read-negative-count",
			"read-over-count", "write-error", "write-did-not-retire",
			"write-retired-twice", "write-zero-count", "write-negative-count",
			"write-over-count", "close-error", "close-did-not-retire",
			"close-retired-twice", "spawn-error", "spawn-did-not-retire",
			"spawn-retired-twice", "spawn-nonzero-exit",
		},
		[]uint64{
			32, 4, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2,
			2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2,
		},
	))
}

// Splits the configuration root once at construction, before any axis draws a value.
func harness_configuration_streams(seed uint64) (streams Harness_Configuration_Streams) {
	root := prng.New(seed)
	streams.Operating_System = prng.Generator_Split(&root)
	streams.Cargo = prng.Generator_Split(&root)
	streams.Data = prng.Generator_Split(&root)
	streams.Color = prng.Generator_Split(&root)
	streams.Installed = prng.Generator_Split(&root)
	streams.Identifier = prng.Generator_Split(&root)
	streams.Failure = prng.Generator_Split(&root)
	streams.Version = prng.Generator_Split(&root)
	streams.Home = prng.Generator_Split(&root)
	streams.Dotfile_Bytes = prng.Generator_Split(&root)
	streams.Probe_Output_Bytes = prng.Generator_Split(&root)
	streams.Ghostty_Verified = prng.Generator_Split(&root)
	streams.Empty_Version = prng.Generator_Split(&root)
	streams.Path_Boundary = prng.Generator_Split(&root)
	streams.Clock_Epoch = prng.Generator_Split(&root)
	streams.IO_Chunk_Bytes = prng.Generator_Split(&root)
	streams.IO_Fault = prng.Generator_Split(&root)
	streams.IO_Fault_Target = prng.Generator_Split(&root)
	streams.Dotfile_Content = prng.Generator_Split(&root)
	streams.Probe_Output = prng.Generator_Split(&root)
	streams.IO_Fault_Site = prng.Generator_Split(&root)
	streams.Profile_File = prng.Generator_Split(&root)
	streams.Configuration_File = prng.Generator_Split(&root)
	streams.Ignored_File = prng.Generator_Split(&root)
	streams.Boundary_File = prng.Generator_Split(&root)
	streams.Boundary_Directory = prng.Generator_Split(&root)
	streams.IO_Witness = prng.Generator_Split(&root)
	streams.Nested_File = prng.Generator_Split(&root)
	return streams
}

// Aligns features whose rare work branch needs a specific host or prior state.
func harness_align(input Harness_Configuration) (configuration Harness_Configuration) {
	configuration = input
	harness_align_boundaries(&configuration)
	harness_align_io_fault(&configuration)
	harness_align_command_failure(&configuration)
	harness_align_success(&configuration)
	harness_align_path_boundary(&configuration)
	harness_align_fixture(&configuration)
	return configuration
}

// Production boundary runs take precedence over faults that would stop them first.
func harness_align_boundaries(configuration *Harness_Configuration) {
	if configuration.Dotfile_Bytes == setup.DOTFILE_PAYLOAD_BYTES_MAX {
		configuration.Failure = ""
		configuration.IO_Fault = ""
	}
	if configuration.Probe_Output_Bytes == 2 {
		configuration.Failure = ""
		configuration.IO_Fault = ""
		configuration.Ghostty_Verified = false
		configuration.Empty_Version = false
	}
}

// An IO fault receives a small Linux tree and reaches its selected shared operation.
func harness_align_io_fault(configuration *Harness_Configuration) {
	if configuration.IO_Fault == "" {
		configuration.IO_Fault_Target = ""
		configuration.IO_Fault_Site = ""
		return
	}
	switch configuration.IO_Fault {
	case "read-directory-error", "spawn-error", "spawn-did-not-retire",
		"spawn-retired-twice", "spawn-nonzero-exit":
		configuration.IO_Fault_Target = HARNESS_IO_TARGET_MIRROR
	}
	if !harness_io_fault_has_probe(configuration.IO_Fault) {
		configuration.IO_Fault_Site = HARNESS_IO_SITE_REQUIRED
	}
	if configuration.IO_Fault_Site == HARNESS_IO_SITE_GITIGNORE {
		if !text_has_prefix(configuration.IO_Fault, "spawn-") {
			configuration.IO_Fault_Site = HARNESS_IO_SITE_REQUIRED
		}
	}
	if configuration.IO_Fault_Target == HARNESS_IO_TARGET_FONT {
		if configuration.IO_Fault != "status-error" {
			configuration.IO_Fault_Site = HARNESS_IO_SITE_REQUIRED
		}
	}
	if configuration.IO_Fault != "" {
		configuration.Failure = ""
		configuration.Operating_System = "linux"
		configuration.Installed = false
		configuration.Dotfile_Bytes = -1
		configuration.Probe_Output_Bytes = 0
		configuration.Ghostty_Verified = false
		configuration.Empty_Version = false
		configuration.Path_Boundary = 0
		configuration.IO_Chunk_Bytes = setup.DOTFILE_PAYLOAD_BYTES_MAX
	}
	if configuration.IO_Fault_Site == HARNESS_IO_SITE_PROBE {
		if text_has_prefix(configuration.IO_Fault, "spawn-") {
			configuration.Installed = true
			configuration.Version_Matches = true
		}
	}
}

// Status and spawn have optional probes whose failure can permit required work to continue.
func harness_io_fault_has_probe(fault string) (present bool) {
	if fault == "status-error" {
		return true
	}
	if fault == "spawn-did-not-retire" {
		return false
	}
	return text_has_prefix(fault, "spawn-")
}

// A command fault removes unrelated rare features and selects the host that owns its step.
func harness_align_command_failure(configuration *Harness_Configuration) {
	if configuration.Failure != "" {
		configuration.Installed = false
		configuration.Dotfile_Bytes = -1
		configuration.Probe_Output_Bytes = 0
		configuration.Ghostty_Verified = false
		configuration.Empty_Version = false
		configuration.Path_Boundary = 0
	}
	if configuration.Dotfile_Bytes == setup.DOTFILE_PAYLOAD_BYTES_MAX {
		configuration.IO_Chunk_Bytes = setup.DOTFILE_PAYLOAD_BYTES_MAX
	}
	if configuration.Failure == "defaults" {
		configuration.Operating_System = "darwin"
	}
	if configuration.Failure == "ghostty" {
		configuration.Operating_System = "darwin"
	}
	if configuration.Failure == "fonts" {
		configuration.Operating_System = "linux"
	}
	if configuration.Failure == "rust" {
		configuration.Cargo_Directory = "xx"
	}
	if configuration.Failure == "rust-link" {
		configuration.Cargo_Directory = "xx"
	}
	if configuration.Failure == "rust-link" {
		configuration.Installed = true
		configuration.Version_Matches = true
	}
}

// Fault-free features receive the prior state that makes each selected branch reachable.
func harness_align_success(configuration *Harness_Configuration) {
	if configuration.Failure == "" {
		if configuration.Probe_Output_Bytes != 0 {
			// A mismatched version is the only path that returns the controlled output.
			configuration.Installed = true
			configuration.Version_Matches = false
		}
	}
	if configuration.Failure == "" {
		if configuration.Ghostty_Verified {
			configuration.Operating_System = "darwin"
			configuration.Installed = true
			configuration.Version_Matches = true
		}
	}
	if configuration.Failure == "" {
		if configuration.Empty_Version {
			configuration.Operating_System = "linux"
			configuration.Installed = true
			configuration.Version_Matches = false
			configuration.Probe_Output_Bytes = 0
		}
	}
}

// A long relative path uses the longest valid home without making installation work dominate it.
func harness_align_path_boundary(configuration *Harness_Configuration) {
	if configuration.Path_Boundary != 0 {
		configuration.Home_Directory = setup.Home_Directory(
			repeat_text("/h", 32))
		configuration.Operating_System = "linux"
		configuration.Cargo_Directory = "xx"
		configuration.Installed = true
		configuration.Version_Matches = true
	}
}

// A rare path activates only the fixture feature that lets the selected operation reach it.
func harness_align_fixture(configuration *Harness_Configuration) {
	if configuration.Dotfile_Bytes >= 0 {
		configuration.Profile_File = true
	}
	if configuration.Path_Boundary == 1 {
		configuration.Boundary_File = true
	}
	if configuration.Path_Boundary == 2 {
		configuration.Boundary_Directory = true
	}
	if configuration.IO_Chunk_Bytes < setup.DOTFILE_PAYLOAD_BYTES_MAX {
		configuration.IO_Witness = true
	}
	if configuration.IO_Fault_Site == HARNESS_IO_SITE_GITIGNORE {
		configuration.Profile_File = true
	}
	if configuration.IO_Fault_Target != HARNESS_IO_TARGET_MIRROR {
		return
	}
	if text_has_prefix(configuration.IO_Fault, "spawn-") {
		return
	}
	if configuration.IO_Fault == "read-directory-error" {
		return
	}
	configuration.Profile_File = true
}

// Setup permits text that is larger than one shared text value, so the harness repeats one
// bounded part at a time without changing the seed-derived size.
func repeat_text(text string, count int) (repeated string) {
	if count < 0 {
		return string(shared_strings.Repeat(
			shared_strings.Text(text), shared_strings.Repeat_Count(count)))
	}
	if count == 0 {
		return ""
	}
	if len(text) == 0 {
		return ""
	}
	if len(text) > shared_strings.TEXT_SIZE_MAXIMUM {
		return string(shared_strings.Repeat(
			shared_strings.Text(text), shared_strings.Repeat_Count(count)))
	}
	part_count_max := shared_strings.TEXT_SIZE_MAXIMUM / len(text)
	content := make([]byte, len(text)*count)
	position := 0
	for copies := count; copies != 0; {
		part_count := copies
		if part_count > part_count_max {
			part_count = part_count_max
		}
		part := shared_strings.Repeat(
			shared_strings.Text(text), shared_strings.Repeat_Count(part_count))
		position += copy(content[position:], string(part))
		copies -= part_count
	}
	return string(content)
}

// A simulated dotfile can exceed the shared slice limit, so each cloned part stays bounded.
func clone_bytes(content []byte) (clone []byte) {
	if len(content) == 0 {
		return content
	}
	clone = make([]byte, len(content))
	for offset := 0; offset < len(content); offset += shared_slices.SLICE_COUNT_MAXIMUM {
		end := offset + shared_slices.SLICE_COUNT_MAXIMUM
		if end > len(content) {
			end = len(content)
		}
		copy(clone[offset:end], shared_slices.Clone(content[offset:end]))
	}
	return clone
}

// Setup paths fit one shared text value, so path queries cross that boundary at one helper.
func text_contains(text string, part string) (contains bool) {
	return bool(shared_strings.Contains(
		shared_strings.Text(text), shared_strings.Text(part)))
}

// Setup paths fit one shared text value, so prefix queries cross that boundary at one helper.
func text_has_prefix(text string, prefix string) (present bool) {
	return bool(shared_strings.Has_Prefix(
		shared_strings.Text(text), shared_strings.Text(prefix)))
}

// Setup paths fit one shared text value, so prefix removal keeps the shared result bounded.
func text_trim_prefix(text string, prefix string) (trimmed string) {
	return string(shared_strings.Trim_Prefix(
		shared_strings.Text(text), shared_strings.Text(prefix)))
}

// A check-ignore batch can exceed one shared text value, so each joined group stays inside the
// shared boundary without changing the process input.
func newline_delimited_text(lines []string) (content []byte) {
	group := shared_strings.Texts{}
	group_size := 0
	flush := func() {
		if len(group) == 0 {
			return
		}
		content = append(content, string(shared_strings.Join(group, "\n"))...)
		content = append(content, '\n')
		group = group[:0]
		group_size = 0
	}
	for _, line := range lines {
		separator_size := 0
		if len(group) != 0 {
			separator_size = 1
		}
		if group_size+separator_size+len(line) > shared_strings.TEXT_SIZE_MAXIMUM {
			flush()
		}
		group = append(group, shared_strings.Text(line))
		group_size += len(line)
		if len(group) > 1 {
			group_size++
		}
	}
	flush()
	return content
}

// A generated batch can exceed one shared text value, so each newline search stays bounded.
// The harness rejects an overlong process line before it allocates a map key.
func split_lines(content []byte) (lines []string) {
	line_start := 0
	search_start := 0
	for search_start < len(content) {
		search_end := search_start + shared_strings.TEXT_SIZE_MAXIMUM
		if search_end > len(content) {
			search_end = len(content)
		}
		index := shared_strings.Index_Byte(
			shared_strings.Text(content[search_start:search_end]), '\n')
		if index == shared_strings.INDEX_ABSENT {
			search_start = search_end
			continue
		}
		line_end := search_start + int(index)
		if line_end != line_start {
			if line_end-line_start <= setup.CHECK_IGNORE_TARGET_BYTES_MAX {
				lines = append(lines, string(content[line_start:line_end]))
			}
		}
		line_start = line_end + 1
		search_start = line_start
	}
	if line_start < len(content) {
		if len(content)-line_start <= setup.CHECK_IGNORE_TARGET_BYTES_MAX {
			lines = append(lines, string(content[line_start:]))
		}
	}
	return lines
}

// The simulation owns its Driver and applies one deferred rule to every callback IO member.
func simulation_file_system(
	seed uint64, configuration Harness_Configuration,
) (system sysio.IO, root Harness_Root, observation *Harness_Observation) {
	root = new_harness_root(seed)
	files, font_contents := simulation_files(seed, configuration)
	observation = &Harness_Observation{}
	model := &Simulation_File_Model{
		Files: files, Directories: simulation_directories(files, configuration),
		Configuration: configuration, Handles: map[sysio.File]string{}, Next_File: 1,
		Observation: observation, Font_Contents: font_contents,
	}
	system = simulation_io(model)
	simulation_callback_io(&system, root.Timeline, model)
	read_directory := func(path string) (entries []sysio.Directory_Entry, err error) {
		return simulation_read_directory(model.Files, model.Directories, path)
	}
	system.Read_Directory = func(path string) (entries []sysio.Directory_Entry, err error) {
		harness_observe_operation(observation, "read-directory")
		if harness_inject_io_fault(
			observation, configuration, "read-directory-error",
			HARNESS_IO_TARGET_MIRROR, HARNESS_IO_SITE_REQUIRED, false,
		) {
			return nil, errors.New("simulated read-directory failure")
		}
		return harness_directory_entries(
			read_directory, configuration, observation, path)
	}
	return system, root, observation
}

// One adapter keeps every callback member on the same deferred simulation timeline.
func simulation_callback_io(
	system *sysio.IO, timeline *Harness_Timeline, model *Simulation_File_Model,
) {
	observation := model.Observation
	configuration := model.Configuration
	system.Read = func(
		completion *sysio.Completion, callback sysio.Callback,
		file sysio.File, buffer []byte, offset int64,
	) {
		harness_observe_operation(observation, "read")
		simulation_read(
			model, completion,
			harness_deferred_callback(
				timeline, observation, configuration, callback),
			file, buffer, offset,
		)
	}
	system.Write = func(
		completion *sysio.Completion, callback sysio.Callback,
		file sysio.File, buffer []byte, offset int64,
	) {
		harness_observe_operation(observation, "write")
		simulation_write(
			model, completion,
			harness_deferred_callback(
				timeline, observation, configuration, callback),
			file, buffer, offset,
		)
	}
	system.Close = func(
		completion *sysio.Completion, callback sysio.Timeout_Callback,
		file sysio.File,
	) {
		harness_observe_operation(observation, "close")
		simulation_close(
			model, completion,
			harness_deferred_timeout_callback(
				timeline, observation, configuration, callback),
			file,
		)
	}
	spawn := harness_process_io(configuration, observation)
	system.Spawn = func(
		completion *sysio.Completion, callback sysio.Process_Callback,
		request sysio.Process_Request, deadline systime.Duration,
	) {
		spawn(
			completion,
			harness_deferred_process_callback(
				timeline, observation, configuration, callback),
			request, deadline,
		)
	}
}

// The immediate members reject their selected fault before they mutate the file model.
func simulation_io(model *Simulation_File_Model) (system sysio.IO) {
	observation := model.Observation
	configuration := model.Configuration
	return sysio.IO{
		Status: func(path string) (status sysio.File_Status, err error) {
			harness_observe_operation(observation, "status")
			target, site := simulation_status_fault(model, path)
			if harness_inject_io_fault(
				observation, configuration, "status-error", target, site, false,
			) {
				return sysio.File_Status{}, errors.New("simulated status failure")
			}
			return simulation_status(model, path)
		},
		Open: func(path string) (file sysio.File, err error) {
			harness_observe_operation(observation, "open")
			if harness_inject_io_fault(
				observation, configuration, "open-error",
				simulation_path_target(path), HARNESS_IO_SITE_REQUIRED, false,
			) {
				return 0, errors.New("simulated open failure")
			}
			return simulation_open(model, path)
		},
		Create: func(path string) (file sysio.File, err error) {
			harness_observe_operation(observation, "create")
			harness_observe_recovery(observation, configuration, "create", "")
			if harness_inject_io_fault(
				observation, configuration, "create-error",
				simulation_path_target(path), HARNESS_IO_SITE_REQUIRED, false,
			) {
				return 0, errors.New("simulated create failure")
			}
			return simulation_create(model, path)
		},
		Make_Directory: func(path string) (err error) {
			harness_observe_operation(observation, "make-directory")
			if harness_inject_io_fault(
				observation, configuration, "make-directory-error",
				simulation_directory_target(model), HARNESS_IO_SITE_REQUIRED, false,
			) {
				return errors.New("simulated make-directory failure")
			}
			return simulation_make_directory(model, path)
		},
	}
}

// Harness_deferred_callback gives each file callback its own later Driver delivery.
func harness_deferred_callback(
	timeline *Harness_Timeline, observation *Harness_Observation,
	configuration Harness_Configuration, callback sysio.Callback,
) (deferred sysio.Callback) {
	retirement_count := 0
	return func(
		completion *sysio.Completion, count int, operation_err error,
	) {
		retirement_count++
		duplicate := retirement_count > 1
		harness_defer_retirement(timeline, duplicate, func() {
			harness_observe_duplicate_retirement(
				observation, configuration, duplicate)
			callback(completion, count, operation_err)
		})
	}
}

// Harness_deferred_timeout_callback gives each close callback a later Driver delivery.
func harness_deferred_timeout_callback(
	timeline *Harness_Timeline, observation *Harness_Observation,
	configuration Harness_Configuration, callback sysio.Timeout_Callback,
) (deferred sysio.Timeout_Callback) {
	retirement_count := 0
	return func(completion *sysio.Completion, operation_err error) {
		retirement_count++
		duplicate := retirement_count > 1
		harness_defer_retirement(timeline, duplicate, func() {
			harness_observe_duplicate_retirement(
				observation, configuration, duplicate)
			callback(completion, operation_err)
		})
	}
}

// Harness_deferred_process_callback gives each process callback a later Driver delivery.
func harness_deferred_process_callback(
	timeline *Harness_Timeline, observation *Harness_Observation,
	configuration Harness_Configuration, callback sysio.Process_Callback,
) (deferred sysio.Process_Callback) {
	retirement_count := 0
	return func(
		completion *sysio.Completion, result sysio.Process_Result,
		operation_err error,
	) {
		retirement_count++
		duplicate := retirement_count > 1
		harness_defer_retirement(timeline, duplicate, func() {
			harness_observe_duplicate_retirement(
				observation, configuration, duplicate)
			callback(completion, result, operation_err)
		})
	}
}

// Harness_Observation proves that a selected fault lands and stops later IO submissions.
type Harness_Observation struct {
	Failure_Observed         bool
	Operation_After_Failure  bool
	IO_Fault_Injected        bool
	IO_Fault_Observed        bool
	IO_Operation_After_Fault bool
	IO_Fault_Cleanup         bool
	IO_Fault_Recovery        bool
	IO_Fault_Subject         string
	Short_Read_Observed      bool
	Short_Write_Observed     bool
}

// A duplicate becomes observable only when the Driver delivers its second callback.
func harness_observe_duplicate_retirement(
	observation *Harness_Observation, configuration Harness_Configuration,
	duplicate bool,
) {
	if !duplicate {
		return
	}
	if !text_contains(configuration.IO_Fault, "retired-twice") {
		return
	}
	if observation.IO_Fault_Injected {
		observation.IO_Fault_Observed = true
	}
}

// A recoverable status fault must cause a destination create, not only unrelated later IO.
func harness_observe_recovery(
	observation *Harness_Observation, configuration Harness_Configuration,
	operation string, subject string,
) {
	if !observation.IO_Fault_Observed {
		return
	}
	if configuration.IO_Fault_Site != HARNESS_IO_SITE_PROBE {
		return
	}
	if configuration.IO_Fault == "status-error" {
		if operation == "create" {
			observation.IO_Fault_Recovery = true
		}
		return
	}
	if text_has_prefix(configuration.IO_Fault, "spawn-") {
		if operation == "spawn-work" {
			if subject == observation.IO_Fault_Subject {
				observation.IO_Fault_Recovery = true
			}
		}
	}
}

// Records one submission after the selected failure returns control to setup.
func harness_observe_operation(observation *Harness_Observation, operation string) {
	if observation.Failure_Observed {
		observation.Operation_After_Failure = true
	}
	if observation.IO_Fault_Injected {
		if observation.IO_Fault_Cleanup {
			if operation == "close" {
				observation.IO_Fault_Cleanup = false
				return
			}
		}
	}
	if !observation.IO_Fault_Observed {
		return
	}
	observation.IO_Operation_After_Fault = true
}

// A selected callback fault allows only the close that releases its borrowed descriptor.
func harness_inject_io_fault(
	observation *Harness_Observation, configuration Harness_Configuration,
	fault string, target string, site string, cleanup bool,
) (injected bool) {
	if configuration.IO_Fault != fault {
		return false
	}
	if observation.IO_Fault_Injected {
		return false
	}
	if configuration.IO_Fault_Target != target {
		return false
	}
	if configuration.IO_Fault_Site != site {
		return false
	}
	observation.IO_Fault_Injected = true
	if !text_contains(fault, "retired-twice") {
		observation.IO_Fault_Observed = true
	}
	observation.IO_Fault_Cleanup = cleanup
	return true
}

// Records the exact selected failure when its simulated boundary rejects one operation.
func harness_observe_failure(
	observation *Harness_Observation, configuration Harness_Configuration, failure string,
) {
	if configuration.Failure == failure {
		observation.Failure_Observed = true
	}
}

// Simulation_File_Model retains descriptor state behind one shared IO value.
type Simulation_File_Model struct {
	Files             map[string][]byte
	Directories       map[string]bool
	Configuration     Harness_Configuration
	Controlled_Source string
	Font_Contents     []byte
	Font_Copy         bool
	Handles           map[sysio.File]string
	Next_File         sysio.File
	Observation       *Harness_Observation
}

// Simulation_contents returns the seeded bytes for one external setup path.
func simulation_contents(model *Simulation_File_Model, path string) (
	contents []byte, found bool, err error,
) {
	if text_contains(path, HARNESS_FONT_SOURCE_PART) {
		if model.Configuration.Failure == "fonts" {
			harness_observe_failure(
				model.Observation, model.Configuration, "fonts")
			return nil, false, errors.New("simulated font read failure")
		}
		return clone_bytes(model.Font_Contents), true, nil
	}
	mapped := harness_path(model.Configuration, path)
	contents, found = model.Files[mapped]
	if !found {
		return nil, false, nil
	}
	if model.Configuration.Dotfile_Bytes < 0 {
		return clone_bytes(contents), true, nil
	}
	if text_has_prefix(path, harness_source(model.Configuration)+"/") {
		if model.Controlled_Source == "" {
			model.Controlled_Source = path
		}
		if path == model.Controlled_Source {
			controlled := repeat_text(
				string(model.Configuration.Dotfile_Byte),
				model.Configuration.Dotfile_Bytes)
			return []byte(controlled), true, nil
		}
	}
	return clone_bytes(contents), true, nil
}

// Simulation_status reports the seeded path type.
func simulation_status(model *Simulation_File_Model, path string) (
	status sysio.File_Status, err error,
) {
	if text_contains(path, HARNESS_FONT_SOURCE_PART) {
		model.Font_Copy = true
		status.Exists, status.Is_Regular = true, true
		return status, nil
	}
	mapped := harness_path(model.Configuration, path)
	_, status.Is_Regular = model.Files[mapped]
	status.Is_Directory = model.Directories[mapped]
	status.Exists = status.Is_Regular
	if status.Is_Directory {
		status.Exists = true
	}
	return status, nil
}

// Simulation_open allocates one readable simulated descriptor.
func simulation_open(
	model *Simulation_File_Model, path string,
) (file sysio.File, err error) {
	_, found, read_err := simulation_contents(model, path)
	if read_err != nil {
		return 0, read_err
	}
	if !found {
		return 0, errors.New("the simulated file is absent")
	}
	return simulation_open_handle(model, path), nil
}

// Simulation_create truncates one simulated path and allocates its descriptor.
func simulation_create(
	model *Simulation_File_Model, path string,
) (file sysio.File, err error) {
	mapped := harness_path(model.Configuration, path)
	model.Files[mapped] = []byte{}
	simulation_add_directories(model.Directories, filepath.Dir(mapped))
	return simulation_open_handle(model, path), nil
}

// Simulation_open_handle retains one external path for later callback operations.
func simulation_open_handle(model *Simulation_File_Model, path string) (file sysio.File) {
	file = model.Next_File
	model.Next_File++
	model.Handles[file] = path
	return file
}

// Simulation_make_directory adds every modeled parent.
func simulation_make_directory(model *Simulation_File_Model, path string) (err error) {
	simulation_add_directories(
		model.Directories, harness_path(model.Configuration, path))
	return nil
}

// A status site separates required source metadata from recoverable destination probes.
func simulation_status_fault(
	model *Simulation_File_Model, path string,
) (target string, site string) {
	if text_contains(path, HARNESS_FONT_SOURCE_PART) {
		return HARNESS_IO_TARGET_FONT, HARNESS_IO_SITE_REQUIRED
	}
	if filepath.Ext(path) == ".ttf" {
		return HARNESS_IO_TARGET_FONT, HARNESS_IO_SITE_PROBE
	}
	mapped := harness_path(model.Configuration, path)
	if mapped == "/"+HARNESS_DESTINATION {
		return HARNESS_IO_TARGET_MIRROR, HARNESS_IO_SITE_PROBE
	}
	if text_has_prefix(mapped, "/"+HARNESS_DESTINATION+"/") {
		return HARNESS_IO_TARGET_MIRROR, HARNESS_IO_SITE_PROBE
	}
	return HARNESS_IO_TARGET_MIRROR, HARNESS_IO_SITE_REQUIRED
}

// Source and destination font descriptors both belong to the large-file copy protocol.
func simulation_path_target(path string) (target string) {
	if text_contains(path, HARNESS_FONT_SOURCE_PART) {
		return HARNESS_IO_TARGET_FONT
	}
	if filepath.Ext(path) == ".ttf" {
		return HARNESS_IO_TARGET_FONT
	}
	return HARNESS_IO_TARGET_MIRROR
}

// Font copy creates its destination directory only after it has read the vendored source.
func simulation_directory_target(model *Simulation_File_Model) (target string) {
	if model.Font_Copy {
		return HARNESS_IO_TARGET_FONT
	}
	return HARNESS_IO_TARGET_MIRROR
}

// An absent handle cannot name a font, and the normal simulator reports that error separately.
func simulation_handle_target(
	model *Simulation_File_Model, file sysio.File,
) (target string) {
	return simulation_path_target(model.Handles[file])
}

// Injects one selected read result before the normal model reads descriptor state.
func simulation_read_fault(
	model *Simulation_File_Model,
	completion *sysio.Completion, callback sysio.Callback,
	file sysio.File, buffer []byte,
) (injected bool) {
	target := simulation_handle_target(model, file)
	if harness_inject_io_fault(
		model.Observation, model.Configuration, "read-error", target,
		HARNESS_IO_SITE_REQUIRED, true,
	) {
		callback(completion, 0, errors.New("simulated read failure"))
		return true
	}
	if harness_inject_io_fault(
		model.Observation, model.Configuration, "read-did-not-retire", target,
		HARNESS_IO_SITE_REQUIRED, true,
	) {
		return true
	}
	if harness_inject_io_fault(
		model.Observation, model.Configuration, "read-retired-twice", target,
		HARNESS_IO_SITE_REQUIRED, true,
	) {
		callback(completion, 0, nil)
		callback(completion, 0, nil)
		return true
	}
	if harness_inject_io_fault(
		model.Observation, model.Configuration, "read-negative-count", target,
		HARNESS_IO_SITE_REQUIRED, true,
	) {
		callback(completion, -1, nil)
		return true
	}
	if harness_inject_io_fault(
		model.Observation, model.Configuration, "read-over-count", target,
		HARNESS_IO_SITE_REQUIRED, true,
	) {
		callback(completion, len(buffer)+1, nil)
		return true
	}
	return false
}

// Simulation_read mutates the model before its boundary adapter queues callback delivery.
func simulation_read(
	model *Simulation_File_Model,
	completion *sysio.Completion, callback sysio.Callback,
	file sysio.File, buffer []byte, offset int64,
) {
	if simulation_read_fault(model, completion, callback, file, buffer) {
		return
	}
	path, present := model.Handles[file]
	if !present {
		callback(completion, 0, errors.New("the simulated file handle is absent"))
		return
	}
	contents, found, read_err := simulation_contents(model, path)
	if read_err != nil {
		callback(completion, 0, read_err)
		return
	}
	if !found {
		callback(completion, 0, nil)
		return
	}
	if offset < 0 {
		callback(completion, 0, errors.New("the simulated read offset is invalid"))
		return
	}
	if offset > int64(len(contents)) {
		callback(completion, 0, errors.New("the simulated read offset is invalid"))
		return
	}
	count := len(buffer)
	if count > model.Configuration.IO_Chunk_Bytes {
		count = model.Configuration.IO_Chunk_Bytes
	}
	available := len(contents) - int(offset)
	completed := copy(buffer[:count], contents[int(offset):])
	if completed == model.Configuration.IO_Chunk_Bytes {
		if completed < available {
			if completed < len(buffer) {
				model.Observation.Short_Read_Observed = true
			}
		}
	}
	callback(completion, completed, nil)
}

// Simulation_write mutates the model before its boundary adapter queues callback delivery.
func simulation_write(
	model *Simulation_File_Model,
	completion *sysio.Completion, callback sysio.Callback,
	file sysio.File, buffer []byte, offset int64,
) {
	target := simulation_handle_target(model, file)
	if harness_inject_io_fault(
		model.Observation, model.Configuration, "write-error", target,
		HARNESS_IO_SITE_REQUIRED, true,
	) {
		callback(completion, 0, errors.New("simulated write failure"))
		return
	}
	if harness_inject_io_fault(
		model.Observation, model.Configuration, "write-did-not-retire", target,
		HARNESS_IO_SITE_REQUIRED, true,
	) {
		return
	}
	if harness_inject_io_fault(
		model.Observation, model.Configuration, "write-retired-twice", target,
		HARNESS_IO_SITE_REQUIRED, true,
	) {
		callback(completion, len(buffer), nil)
		callback(completion, len(buffer), nil)
		return
	}
	if harness_inject_io_fault(
		model.Observation, model.Configuration, "write-zero-count", target,
		HARNESS_IO_SITE_REQUIRED, true,
	) {
		callback(completion, 0, nil)
		return
	}
	if harness_inject_io_fault(
		model.Observation, model.Configuration, "write-negative-count", target,
		HARNESS_IO_SITE_REQUIRED, true,
	) {
		callback(completion, -1, nil)
		return
	}
	if harness_inject_io_fault(
		model.Observation, model.Configuration, "write-over-count", target,
		HARNESS_IO_SITE_REQUIRED, true,
	) {
		callback(completion, len(buffer)+1, nil)
		return
	}
	path, present := model.Handles[file]
	if !present {
		callback(completion, 0, errors.New("the simulated write is invalid"))
		return
	}
	if offset < 0 {
		callback(completion, 0, errors.New("the simulated write is invalid"))
		return
	}
	mapped := harness_path(model.Configuration, path)
	count := len(buffer)
	if count > model.Configuration.IO_Chunk_Bytes {
		count = model.Configuration.IO_Chunk_Bytes
	}
	if count < len(buffer) {
		model.Observation.Short_Write_Observed = true
	}
	end := int(offset) + count
	if end > len(model.Files[mapped]) {
		growth := make([]byte, end-len(model.Files[mapped]))
		model.Files[mapped] = append(model.Files[mapped], growth...)
	}
	copy(model.Files[mapped][int(offset):], buffer[:count])
	callback(completion, count, nil)
}

// Simulation_close mutates the model before its boundary adapter queues callback delivery.
func simulation_close(
	model *Simulation_File_Model,
	completion *sysio.Completion, callback sysio.Timeout_Callback, file sysio.File,
) {
	target := simulation_handle_target(model, file)
	if harness_inject_io_fault(
		model.Observation, model.Configuration, "close-error", target,
		HARNESS_IO_SITE_REQUIRED, false,
	) {
		callback(completion, errors.New("simulated close failure"))
		return
	}
	if harness_inject_io_fault(
		model.Observation, model.Configuration, "close-did-not-retire", target,
		HARNESS_IO_SITE_REQUIRED, false,
	) {
		return
	}
	if harness_inject_io_fault(
		model.Observation, model.Configuration, "close-retired-twice", target,
		HARNESS_IO_SITE_REQUIRED, false,
	) {
		callback(completion, nil)
		callback(completion, nil)
		return
	}
	delete(model.Handles, file)
	callback(completion, nil)
}

// Each payload uses its own stream, so a new file does not move an existing pinned payload.
func simulation_files(
	seed uint64, configuration Harness_Configuration,
) (files map[string][]byte, font_contents []byte) {
	streams := harness_file_streams(seed)
	payloads := []string{"a", "bb", "ccc", "dddd"}
	profile := []byte(prng.Generator_Element(&streams.Profile, payloads))
	configured := []byte(prng.Generator_Element(&streams.Configuration, payloads))
	ignored := []byte(prng.Generator_Element(&streams.Ignored, payloads))
	boundary := []byte(prng.Generator_Element(
		&streams.Boundary, []string{"", "x"}))
	nested_boundary := []byte(prng.Generator_Element(&streams.Nested_Boundary, payloads))
	io_witness := []byte(prng.Generator_Element(
		&streams.IO_Witness, []string{"xxx", "yyyy"}))
	files = map[string][]byte{}
	if configuration.Profile_File {
		files["/profile"] = profile
	}
	if configuration.Configuration_File {
		files["/nested/config"] = configured
	}
	if configuration.Ignored_File {
		files["/e0/ignored"] = ignored
	}
	if configuration.Boundary_File {
		files["/f0"] = boundary
	}
	if configuration.Nested_File {
		files["/d0/x"] = nested_boundary
	}
	if configuration.IO_Witness {
		files["/i0"] = io_witness
	}
	font_contents = []byte(prng.Generator_Element(
		&streams.Font, []string{"font-a", "font-bb", "font-ccc", "font-dddd"}))
	return files, font_contents
}

// Splits the payload root once at construction, before any file draws its bytes.
func harness_file_streams(seed uint64) (streams Harness_File_Streams) {
	root := prng.New(seed)
	streams.Profile = prng.Generator_Split(&root)
	streams.Configuration = prng.Generator_Split(&root)
	streams.Ignored = prng.Generator_Split(&root)
	streams.Boundary = prng.Generator_Split(&root)
	streams.Nested_Boundary = prng.Generator_Split(&root)
	streams.IO_Witness = prng.Generator_Split(&root)
	streams.Font = prng.Generator_Split(&root)
	return streams
}

// Returns fixed-width content from the three payload endpoints that the harness controls.
func harness_seeded_payloads(seed uint64) (
	dotfile string, probe string, font string,
) {
	configuration := harness_configuration(seed)
	configuration.Failure = ""
	configuration.IO_Fault = ""
	configuration.Dotfile_Bytes = 1
	configuration.Probe_Output_Bytes = 1
	configuration.Installed = true
	configuration.Version_Matches = false
	configuration.Profile_File = true
	files, font_contents := simulation_files(seed, configuration)
	model := &Simulation_File_Model{
		Files: files, Directories: simulation_directories(files, configuration),
		Configuration: configuration, Handles: map[sysio.File]string{},
		Next_File: 1, Observation: &Harness_Observation{}, Font_Contents: font_contents,
	}
	dotfile_contents, _, _ := simulation_contents(
		model, filepath.Join(harness_source(configuration), "profile"))
	probe_result := harness_spawn(
		configuration, sysio.Process_Request{Path: "direnv"})
	font_contents, _, _ = simulation_contents(
		model, filepath.Join(HARNESS_FONT_SOURCE_PART, "font.ttf"))
	return string(dotfile_contents), string(probe_result.Output), string(font_contents)
}

// Returns the first draw of every named stream before the harness consumes that stream.
func harness_stream_first_values(seed uint64) (values map[string]uint64) {
	configuration := harness_configuration_streams(seed)
	files := harness_file_streams(seed)
	mutation := harness_mutation_stream(seed)
	return map[string]uint64{
		"configuration_home": prng.Generator_Next(&configuration.Home),
		"configuration_operating_system": prng.Generator_Next(
			&configuration.Operating_System),
		"configuration_cargo":      prng.Generator_Next(&configuration.Cargo),
		"configuration_data":       prng.Generator_Next(&configuration.Data),
		"configuration_color":      prng.Generator_Next(&configuration.Color),
		"configuration_installed":  prng.Generator_Next(&configuration.Installed),
		"configuration_identifier": prng.Generator_Next(&configuration.Identifier),
		"configuration_failure":    prng.Generator_Next(&configuration.Failure),
		"configuration_version":    prng.Generator_Next(&configuration.Version),
		"configuration_dotfile_bytes": prng.Generator_Next(
			&configuration.Dotfile_Bytes),
		"configuration_probe_output_bytes": prng.Generator_Next(
			&configuration.Probe_Output_Bytes),
		"configuration_ghostty_verified": prng.Generator_Next(
			&configuration.Ghostty_Verified),
		"configuration_empty_version": prng.Generator_Next(
			&configuration.Empty_Version),
		"configuration_path_boundary": prng.Generator_Next(
			&configuration.Path_Boundary),
		"configuration_clock_epoch": prng.Generator_Next(
			&configuration.Clock_Epoch),
		"configuration_io_chunk_bytes": prng.Generator_Next(
			&configuration.IO_Chunk_Bytes),
		"configuration_io_fault": prng.Generator_Next(
			&configuration.IO_Fault),
		"configuration_io_fault_target": prng.Generator_Next(
			&configuration.IO_Fault_Target),
		"configuration_dotfile_content": prng.Generator_Next(
			&configuration.Dotfile_Content),
		"configuration_probe_output": prng.Generator_Next(
			&configuration.Probe_Output),
		"configuration_io_fault_site": prng.Generator_Next(
			&configuration.IO_Fault_Site),
		"configuration_profile_file": prng.Generator_Next(
			&configuration.Profile_File),
		"configuration_configuration_file": prng.Generator_Next(
			&configuration.Configuration_File),
		"configuration_ignored_file": prng.Generator_Next(
			&configuration.Ignored_File),
		"configuration_boundary_file": prng.Generator_Next(
			&configuration.Boundary_File),
		"configuration_boundary_directory": prng.Generator_Next(
			&configuration.Boundary_Directory),
		"configuration_io_witness": prng.Generator_Next(
			&configuration.IO_Witness),
		"configuration_nested_file": prng.Generator_Next(
			&configuration.Nested_File),
		"file_profile":         prng.Generator_Next(&files.Profile),
		"file_configuration":   prng.Generator_Next(&files.Configuration),
		"file_ignored":         prng.Generator_Next(&files.Ignored),
		"file_boundary":        prng.Generator_Next(&files.Boundary),
		"file_nested_boundary": prng.Generator_Next(&files.Nested_Boundary),
		"file_io_witness":      prng.Generator_Next(&files.IO_Witness),
		"file_font":            prng.Generator_Next(&files.Font),
		"mutation":             prng.Generator_Next(&mutation),
	}
}

// The salt holds the mutation stream stable when the configuration root gains a new axis.
func harness_mutation_stream(seed uint64) (generator prng.Generator) {
	return prng.New(seed ^ HARNESS_MUTATION_SALT)
}

// The directory index derives only from seeded file paths and preserves an explicit root.
func simulation_directories(
	files map[string][]byte, configuration Harness_Configuration,
) (directories map[string]bool) {
	directories = map[string]bool{"/": true}
	if configuration.Boundary_Directory {
		directories["/"+HARNESS_EMPTY_DIRECTORY] = true
	}
	for path := range files {
		simulation_add_directories(directories, filepath.Dir(path))
	}
	return directories
}

// Each parent becomes visible to Read_Directory after a modeled write creates it.
func simulation_add_directories(directories map[string]bool, directory string) {
	directories[directory] = true
	for directory != "/" {
		directory = filepath.Dir(directory)
		directories[directory] = true
	}
}

// The model sorts map-derived entries, so the same seed always gives the same walk order.
func simulation_read_directory(
	files map[string][]byte, directories map[string]bool, path string,
) (entries []sysio.Directory_Entry, err error) {
	if !directories[path] {
		return nil, errors.New("the simulated directory is absent")
	}
	children := map[string]bool{}
	for file := range files {
		if filepath.Dir(file) == path {
			children[filepath.Base(file)] = false
		}
	}
	for directory := range directories {
		if directory != path {
			if filepath.Dir(directory) == path {
				children[filepath.Base(directory)] = true
			}
		}
	}
	for name, is_directory := range children {
		entries = append(entries, sysio.Directory_Entry{
			Name: name, Is_Directory: is_directory})
	}
	shared_slices.Sort_Function(entries, func(
		left, right sysio.Directory_Entry,
	) (comparison shared_slices.Comparison) {
		return shared_slices.Comparison(shared_strings.Compare(
			shared_strings.Text(left.Name), shared_strings.Text(right.Name)))
	})
	return entries, nil
}

// Reads one simulated directory and gives one existing entry a boundary path
// when the seed selected that path feature.
func harness_directory_entries(
	read_directory func(path string) (entries []sysio.Directory_Entry, err error),
	configuration Harness_Configuration, observation *Harness_Observation, path string,
) (entries []sysio.Directory_Entry, err error) {
	if configuration.Failure == "dotfiles" {
		if path == harness_source(configuration) {
			harness_observe_failure(observation, configuration, "dotfiles")
			return nil, errors.New("simulated dotfile scan failure")
		}
	}
	entries, read_err := read_directory(harness_path(configuration, path))
	if read_err != nil {
		return nil, read_err
	}
	if configuration.Path_Boundary == 0 {
		return entries, nil
	}
	if path != harness_source(configuration) {
		return entries, nil
	}
	want_directory := configuration.Path_Boundary == 2
	for index := range entries {
		if entries[index].Name == HARNESS_DESTINATION {
			continue
		}
		if entries[index].Name == HARNESS_IGNORED {
			continue
		}
		if entries[index].Is_Directory != want_directory {
			continue
		}
		if want_directory {
			if entries[index].Name != HARNESS_EMPTY_DIRECTORY {
				continue
			}
		}
		alias := repeat_text("p", setup.RELATIVE_FILE_PATH_BYTES_MAX)
		alias_path := filepath.Join(path, alias)
		configuration.Path_Aliases[alias_path] = filepath.Join(path, entries[index].Name)
		entries[index].Name = alias
		break
	}
	return entries, nil
}

// Builds a bounded absolute home from two-byte path components.
func harness_home(bytes int) (home string) {
	return repeat_text("/h", bytes/2)
}

// Returns Main's logical dotfile source for one configured home.
func harness_source(configuration Harness_Configuration) (source string) {
	return filepath.Join(string(configuration.Home_Directory), "code/james-orcales/home")
}

// Maps Main's fixed source and home paths onto the simulator's source and destination roots.
func harness_path(configuration Harness_Configuration, path string) (mapped string) {
	for alias, original := range configuration.Path_Aliases {
		if path == alias {
			path = original
			break
		}
		if text_has_prefix(path, alias+"/") {
			path = original + text_trim_prefix(path, alias)
			break
		}
	}
	source := harness_source(configuration)
	home := string(configuration.Home_Directory)
	data := string(configuration.Data_Directory)
	if data != "" {
		if path == data {
			return "/dest/data"
		}
		if text_has_prefix(path, data+"/") {
			return filepath.Join("/dest/data", text_trim_prefix(path, data+"/"))
		}
	}
	if path == source {
		return "/"
	}
	if text_has_prefix(path, source+"/") {
		return "/" + text_trim_prefix(path, source+"/")
	}
	if path == home {
		return filepath.Join("/", HARNESS_DESTINATION)
	}
	if text_has_prefix(path, home+"/") {
		return filepath.Join(
			"/", HARNESS_DESTINATION, text_trim_prefix(path, home+"/"))
	}
	return path
}

// Returns one seed-fixed process fabric and retains the tool named by its latest version probe.
func harness_process_io(
	configuration Harness_Configuration, observation *Harness_Observation,
) (
	spawn func(
		completion *sysio.Completion, callback sysio.Process_Callback,
		request sysio.Process_Request, deadline systime.Duration,
	),
) {
	tool := ""
	return func(
		completion *sysio.Completion, callback sysio.Process_Callback,
		request sysio.Process_Request, _ systime.Duration,
	) {
		harness_observe_operation(observation, "spawn")
		if request.Path == "which" {
			tool = request.Arguments[0]
		} else if harness_version(request) != "" {
			tool = filepath.Base(request.Path)
		}
		result := harness_spawn(configuration, request)
		if request.Path == "git" {
			if harness_process_fault(
				configuration, observation, HARNESS_IO_SITE_GITIGNORE,
				completion, callback, result,
			) {
				return
			}
		}
		if harness_process_probe(request) {
			if harness_process_fault(
				configuration, observation, HARNESS_IO_SITE_PROBE,
				completion, callback, result,
			) {
				observation.IO_Fault_Subject = tool
				return
			}
		}
		if harness_command_fails(tool, tool, request) {
			harness_observe_recovery(
				observation, configuration, "spawn-work", tool)
			if harness_process_fault(
				configuration, observation, HARNESS_IO_SITE_REQUIRED,
				completion, callback, result,
			) {
				return
			}
		}
		if harness_command_fails(configuration.Failure, tool, request) {
			harness_observe_failure(
				observation, configuration, configuration.Failure)
			result.Exit = 1
		}
		callback(completion, result, nil)
	}
}

// A version or path probe can fail before setup submits the required work command.
func harness_process_probe(request sysio.Process_Request) (probe bool) {
	if request.Path == "which" {
		return true
	}
	return harness_version(request) != ""
}

// Retires one selected process fault at its required or recoverable site.
func harness_process_fault(
	configuration Harness_Configuration, observation *Harness_Observation, site string,
	completion *sysio.Completion, callback sysio.Process_Callback,
	result sysio.Process_Result,
) (injected bool) {
	if harness_inject_io_fault(
		observation, configuration, "spawn-error",
		HARNESS_IO_TARGET_MIRROR, site, false,
	) {
		callback(completion, sysio.Process_Result{}, errors.New("simulated spawn failure"))
		return true
	}
	if harness_inject_io_fault(
		observation, configuration, "spawn-did-not-retire",
		HARNESS_IO_TARGET_MIRROR, site, false,
	) {
		return true
	}
	if harness_inject_io_fault(
		observation, configuration, "spawn-retired-twice",
		HARNESS_IO_TARGET_MIRROR, site, false,
	) {
		callback(completion, result, nil)
		callback(completion, result, nil)
		return true
	}
	if harness_inject_io_fault(
		observation, configuration, "spawn-nonzero-exit",
		HARNESS_IO_TARGET_MIRROR, site, false,
	) {
		result.Exit = 1
		callback(completion, result, nil)
		return true
	}
	return false
}

// Reports installed tool versions and evaluates the one gitignore batch that Mirror submits.
func harness_spawn(
	configuration Harness_Configuration, request sysio.Process_Request,
) (result sysio.Process_Result) {
	if request.Path == "git" {
		return harness_git_ignore(configuration, request)
	}
	if request.Path == "which" {
		if !configuration.Installed {
			return result
		}
		name := request.Arguments[0]
		if name == "nvim" {
			executable := filepath.Join(
				harness_source(configuration), ".local/bin/nvim")
			return sysio.Process_Result{
				Output: []byte(executable + "\n"),
			}
		}
		return sysio.Process_Result{Output: []byte("/installed/" + name + "\n")}
	}
	version := harness_version(request)
	if version != "" {
		if configuration.Installed {
			if configuration.Version_Matches {
				return sysio.Process_Result{Output: []byte(version + "\n")}
			}
			return sysio.Process_Result{Output: []byte(repeat_text(
				string(configuration.Probe_Output_Byte),
				configuration.Probe_Output_Bytes))}
		}
	}
	return result
}

// Reports the expected version for a direct executable probe.
func harness_version(request sysio.Process_Request) (version string) {
	versions := map[string]string{
		"direnv": "2.37.1", "nvim": "NVIM v0.12.3", "fzf": "0.73.1",
		"rustc": "rustc 1.96.0", "jj": "jj 0.42.0", "rg": "ripgrep 15.1.0",
		"fd": "fd 10.4.2", "ghostty": "Ghostty 1.3.1",
	}
	return versions[filepath.Base(request.Path)]
}

// Selects only the configured step's work command, never its version probe.
func harness_command_fails(
	failure string, tool string, request sysio.Process_Request,
) (fails bool) {
	if request.Path == "which" {
		return false
	}
	if request.Path == "git" {
		return false
	}
	if request.Path == "fc-cache" {
		return false
	}
	if failure == "defaults" {
		return request.Path == "defaults"
	}
	if failure == "rust-link" {
		return request.Path == "ln"
	}
	if failure == "nvim" {
		return request.Path == "make"
	}
	if failure == "fonts" {
		return false
	}
	if failure == "dotfiles" {
		return false
	}
	if failure == "rust" {
		return tool == "rustc" && request.Path == "sh"
	}
	if failure == "" {
		return false
	}
	if failure != tool {
		return false
	}
	return harness_version(request) == ""
}

// Converts ignored simulated paths back to the absolute names that git prints.
func harness_git_ignore(
	configuration Harness_Configuration, request sysio.Process_Request,
) (result sysio.Process_Result) {
	ignored := []string{}
	for _, target := range split_lines(request.Input) {
		relative := text_trim_prefix(target, harness_source(configuration)+"/")
		if target != "" {
			if harness_ignores_one(relative) {
				ignored = append(ignored, target)
			}
		}
	}
	if len(ignored) != 0 {
		result.Output = newline_delimited_text(ignored)
	}
	return result
}

// Classifies a batch of source paths, returning the set that is the destination root or the
// ignored subtree — pruned so the mirror neither copies into itself nor syncs the ignored
// entry. Batched to match the Is_Ignored seam the mirror now calls once per tree level.
func harness_ignore(relatives []string) (ignored map[string]bool) {
	ignored = map[string]bool{}
	for _, relative := range relatives {
		if harness_ignores_one(relative) {
			ignored[relative] = true
		}
	}
	return ignored
}

// Reports whether one source path is the destination root or the ignored subtree, the
// per-path rule the harness's own walk shares with the batched predicate.
func harness_ignores_one(relative string) (ignored bool) {
	if relative == HARNESS_DESTINATION {
		return true
	}
	if text_has_prefix(relative, HARNESS_DESTINATION+"/") {
		return true
	}
	if relative == HARNESS_IGNORED {
		return true
	}
	return text_has_prefix(relative, HARNESS_IGNORED+"/")
}

// Wraps shared IO Create to count the files a mirror run writes, returning the live count.
func harness_count_writes(
	system *sysio.IO, configuration Harness_Configuration,
) (writes *int) {
	count := 0
	destinations := map[string]bool{}
	for _, relative := range harness_source_files(system, configuration) {
		destinations[filepath.Join(string(configuration.Home_Directory), relative)] = true
	}
	create := system.Create
	system.Create = func(path string) (file sysio.File, err error) {
		if destinations[path] {
			count++
		}
		return create(path)
	}
	return &count
}

// Asserts the ignored subtree never reached the destination.
func harness_assert_pruned(
	t *testing.T, system *sysio.IO, configuration Harness_Configuration,
) {
	status, _ := system.Status(filepath.Join(
		string(configuration.Home_Directory), HARNESS_IGNORED))
	if status.Exists {
		t.Fatalf("the ignored subtree %q was synced to the destination", HARNESS_IGNORED)
	}
}

// Overwrites a seed-chosen subset of the destination's mirrored files with poison so they
// differ from their source, returning how many were changed.
func harness_mutate(
	system *sysio.IO, configuration Harness_Configuration, seed uint64,
) (count int) {
	generator := harness_mutation_stream(seed)
	for _, relative := range harness_source_files(system, configuration) {
		if len(relative) == setup.RELATIVE_FILE_PATH_BYTES_MAX {
			continue
		}
		if prng.Generator_Boolean(&generator) {
			continue
		}
		harness_overwrite(system, filepath.Join("/", HARNESS_DESTINATION, relative))
		count++
	}
	return count
}

// Lists the non-ignored source files under the root, walking the tree through Read_Directory.
func harness_source_files(
	system *sysio.IO, configuration Harness_Configuration,
) (relatives []string) {
	relatives = []string{}
	worklist := []string{"."}
	for len(worklist) > 0 {
		directory := worklist[len(worklist)-1]
		worklist = worklist[:len(worklist)-1]
		entries, err := system.Read_Directory(filepath.Join(
			harness_source(configuration), directory))
		if err != nil {
			continue
		}
		for _, entry := range entries {
			relative := filepath.Join(directory, entry.Name)
			if harness_ignores_one(relative) {
				continue
			}
			if entry.Is_Directory {
				worklist = append(worklist, relative)
				continue
			}
			relatives = append(relatives, relative)
		}
	}
	return relatives
}

// Overwrites path with the poison content through the loop, driving the write and the close
// to completion.
func harness_overwrite(system *sysio.IO, path string) {
	file, create_err := system.Create(path)
	if create_err != nil {
		return
	}
	completion := sysio.Completion{}
	system.Write(&completion, func(_ *sysio.Completion, _ int, _ error) {},
		file, []byte(HARNESS_POISON), 0)
	system.Close(&completion, func(_ *sysio.Completion, _ error) {}, file)
}
