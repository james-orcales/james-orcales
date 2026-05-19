// Package invariant_default is the composition-tier sibling of invariant. It
// wires the pure library to the real OS (filesystem, stderr, os.Args sniffing,
// runtime.Callers, os.Exit) and re-exports the surface so callers can write:
//
//	import invariant "github.com/james-orcales/james-orcales/golang_snacks/invariant/v2/invariant_default"
//
// and use invariant.Ensure / invariant.Always / invariant.Sometimes / … as if
// no split had happened.
package invariant_default

import (
	"fmt"
	"io"
	"iter"
	"os"
	"runtime"
	"strconv"
	"strings"
	"testing"

	invariant "github.com/james-orcales/james-orcales/golang_snacks/invariant/v2"
)

// Type aliases re-export the library's types so callers importing only this
// package don't need a second import to refer to Recorder, Frame_Information, etc.
type (
	Recorder           = invariant.Recorder
	Frame_Information  = invariant.Frame_Information
	Assertion_Metadata = invariant.Assertion_Metadata
	Regex_Match_Input  = invariant.Regex_Match_Input
	Cross_Axis         = invariant.Cross_Axis
	Bucket_Reference   = invariant.Bucket_Reference
)

// Boundary_Input is the generic alias for the pure-tier type. Re-exported
// so callers writing invariant.Boundary(&invariant.Boundary_Input[int]{...})
// need only one import.
type Boundary_Input[I invariant.Numeric] = invariant.Boundary_Input[I]

// Default is the OS-bound Recorder used by the package-level Ensure / Always / …
// convenience functions. Tests that need to redirect I/O construct their own
// Recorder via the pure invariant package; Default exists for the common case.
var Default = Init_Default_Recorder()

// Init_Default_Recorder builds a Recorder wired to the host OS: the local
// filesystem, os.Stderr, os.Exit, and runtime.Callers. Sniffs os.Args once to
// determine the test/fuzz/benchmark environment flags.
func Init_Default_Recorder() (recorder *invariant.Recorder) {

	working_directory, _ := os.Getwd()
	is_test, is_fuzz, is_benchmark := running_environment_flags()
	// /dev/tty bypasses go test's stdout/stderr capture so the success-summary
	// line shows without -v. Assigned only when the open succeeds — assigning
	// a nil *os.File to an io.Writer interface produces a non-nil interface,
	// which would defeat the nil-check on the consumer side.
	var tty io.Writer
	if opened, err := os.OpenFile("/dev/tty", os.O_WRONLY, 0); err == nil {
		tty = opened
	}
	return &invariant.Recorder{
		Output:           os.Stderr,
		Tty:              tty,
		File_System:      os.DirFS("/"),
		Axis_Handle_Pool: invariant.New_Axis_Handle_Pool(),
		Get_Caller: func(skip int) (frame_information invariant.Frame_Information, err error) {
			// +1 compensates for this closure's own stack frame: callers
			// pass skip counted as if Get_Caller were a transparent
			// passthrough, but the closure body is itself a frame above
			// runtime.Callers. Composition-tier wrappers below are marked
			// //go:noinline so the rest of the framework's skip math (which
			// assumes each wrapper IS a real frame) holds.
			program_counters := [1]uintptr{}
			count := runtime.Callers(skip+1, program_counters[:])
			frame, _ := runtime.CallersFrames(program_counters[:count]).Next()
			return invariant.Frame_Information{File: frame.File, Line: frame.Line}, nil
		},
		Exit:                os.Exit,
		Framework_Exit:      os.Exit,
		Fatal_Failures:      true,
		Full_Location:       os.Getenv("INVARIANT_FULL_LOCATION") == "1",
		Stacktrace_Depth:    stacktrace_depth_from_environment(),
		Working_Directory:   working_directory,
		Is_Test:             is_test,
		Is_Fuzz:             is_fuzz,
		Is_Benchmark:        is_benchmark,
		Packages_To_Analyze: []string{"."},
	}
}

func stacktrace_depth_from_environment() (depth int) {

	value := os.Getenv("INVARIANT_STACKTRACE_DEPTH")
	result, err := strconv.Atoi(value)
	if err != nil {
		return 10
	}
	return result
}

func running_environment_flags() (is_test bool, is_fuzz bool, is_benchmark bool) {

	for _, argument := range os.Args {
		if strings.HasPrefix(argument, "-test.fuzz") {
			is_fuzz = true
		}
		if strings.HasPrefix(argument, "-test.bench") {
			is_benchmark = true
		}
		if strings.HasPrefix(argument, "-test.") {
			is_test = true
		}
	}
	return is_test, is_fuzz, is_benchmark
}

// Run_Test_Main is the canonical TestMain body.
func Run_Test_Main(m *testing.M, dirs ...string) {
	invariant.Recorder_Run_Test_Main(Default, m, dirs...)
}

// Register_Packages_For_Analysis forwards to the library function on Default.
func Register_Packages_For_Analysis(dirs ...string) {
	invariant.Recorder_Register_Packages_For_Analysis(Default, dirs...)
}

// Analyze_Assertion_Frequency forwards to the library function on Default.
func Analyze_Assertion_Frequency() {
	invariant.Recorder_Analyze_Assertion_Frequency(Default)
}

// Always has two properties, nothing more and nothing less:
//
//  1. It evaluates to true at least once in the test suite.
//  2. It panics or exits on violation.
//
// Only Cross_Product enforces coverage of these properties. Always alone
// is a bug — use Is_Always.
//
//go:noinline
func Always[B ~bool](predicate B, message string) (record invariant.Cross_Axis) {

	return invariant.Recorder_Always(Default, predicate, message)
}

// Sometimes classifies predicate as a Cross_Axis with {false, true} buckets;
// records both branches across the test run. Message describes what the
// axis is tracking so coverage reports name each axis.
//
// Only Cross_Product enforces coverage. Sometimes alone is a bug — use
// Is_Sometimes.
//
//go:noinline
func Sometimes[B ~bool](predicate B, message string) (record invariant.Cross_Axis) {

	return invariant.Recorder_Sometimes(Default, predicate, message)
}

// Distinct_Boundary classifies X as a Cross_Axis with two endpoint buckets
// — "Lo" at index 0 and "Hi" at index 1 — while enforcing Lo < Hi
// (distinct endpoints) and Lo <= X <= Hi. Both Lo>=Hi and out-of-range X
// fail-fatal as user-domain assertion failures. Interior X values
// produce a -1 sentinel that the Cross_Product key lookup ignores: only
// endpoints contribute coverage.
//
//go:noinline
func Distinct_Boundary[I invariant.Numeric](input *invariant.Boundary_Input[I]) (record invariant.Cross_Axis) {

	return invariant.Recorder_Distinct_Boundary(Default, input)
}

// Is_Distinct_Boundary is the single-axis sugar — same Lo < Hi + bound
// enforcement, but the coverage entry is registered via the of-path so a
// bare site doesn't need a surrounding Cross_Product.
//
//go:noinline
func Is_Distinct_Boundary[I invariant.Numeric](input *invariant.Boundary_Input[I]) {
	invariant.Recorder_Is_Distinct_Boundary(Default, input)
}

// Is_Always panics with message on false. Registers a tracker entry at the
// call site so a never-firing site surfaces in the coverage report.
//
//go:noinline
func Is_Always(condition bool, message string) {
	invariant.Recorder_Is_Always(Default, condition, message)
}

// Is_Always_Nil_Error is Is_Always for the err == nil property.
//
//go:noinline
func Is_Always_Nil_Error(err error, message string) {
	invariant.Recorder_Is_Always_Nil_Error(Default, err, message)
}

// Is_Sometimes records that condition was true at least once during the test run.
//
//go:noinline
func Is_Sometimes(condition bool, message string) (firing_true bool) {

	return invariant.Recorder_Is_Sometimes(Default, condition, message)
}

// Unimplemented panics with an assertion-failure-prefixed message.
//
//go:noinline
func Unimplemented(message string) {
	invariant.Recorder_Unimplemented(Default, message)
}

// Unreachable panics with an assertion-failure-prefixed message.
//
//go:noinline
func Unreachable(message string) {
	invariant.Recorder_Unreachable(Default, message)
}

// Format_Default_Recorder is a debug helper that returns a compact summary of
// the Default recorder's configuration.
func Format_Default_Recorder() (summary string) {

	return fmt.Sprintf("invariant.Default{is_test=%t is_fuzz=%t is_benchmark=%t}",
		Default.Is_Test, Default.Is_Fuzz, Default.Is_Benchmark)
}

// Cross_Product fires one composite Sometimes per bucket-tuple in the
// cross-product of records.
//
//go:noinline
func Cross_Product(records ...invariant.Cross_Axis) {
	invariant.Recorder_Cross_Product(Default, records...)
}

// Bucket_False names the false bucket of a Sometimes axis for use inside an
// Excluding clause. Pure packaging — no Recorder needed.
func Bucket_False(axis invariant.Cross_Axis) (reference invariant.Bucket_Reference) {
	return invariant.Bucket_False(axis)
}

// Bucket_True names the true bucket of a Sometimes axis. See Bucket_False.
func Bucket_True(axis invariant.Cross_Axis) (reference invariant.Bucket_Reference) {
	return invariant.Bucket_True(axis)
}

// Bucket_Lo names the lower endpoint of a Boundary axis. See Bucket_False.
func Bucket_Lo(axis invariant.Cross_Axis) (reference invariant.Bucket_Reference) {
	return invariant.Bucket_Lo(axis)
}

// Bucket_Hi names the upper endpoint of a Boundary axis. See Bucket_False.
func Bucket_Hi(axis invariant.Cross_Axis) (reference invariant.Bucket_Reference) {
	return invariant.Bucket_Hi(axis)
}

// Excluding declares the named cell impossible inside a Cross_Product. See
// the pure-tier doc on invariant.Excluding.
func Excluding(message string, references ...invariant.Bucket_Reference) (record invariant.Cross_Axis) {
	return invariant.Excluding(message, references...)
}

// Game_Loop yields an infinite sequence of increasing integers. Re-exported
// from the pure tier so callers importing only this package can use the
// `for range invariant.Game_Loop()` pattern without a second import.
func Game_Loop() (sequence iter.Seq[int]) {
	return invariant.Game_Loop()
}

// Until returns a bounded sequence useful for safely capping infinite loops.
// Re-exported from the pure tier.
func Until[T invariant.Integer_Like](limit T) (sequence iter.Seq[int]) {
	return invariant.Until(limit)
}

// Has_Whitespace re-exports the pure-tier predicate.
func Has_Whitespace(s string) (yes bool) { return invariant.Has_Whitespace(s) }

// Has_Special_Control re-exports the pure-tier predicate.
func Has_Special_Control(s string) (yes bool) { return invariant.Has_Special_Control(s) }

// Has_Null_Byte re-exports the pure-tier predicate.
func Has_Null_Byte(s string) (yes bool) { return invariant.Has_Null_Byte(s) }
