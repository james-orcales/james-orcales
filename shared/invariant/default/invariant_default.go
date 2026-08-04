// Package invariant_default is the composition-tier sibling of invariant. It
// wires the pure library to the real OS — local filesystem, stderr, os.Args
// sniffing, os.Exit — and re-exports the surface. Import it aliased as invariant
// and use invariant.Always / invariant.Assertions as if no split between pure
// and OS-bound tiers existed.
package invariant

import (
	"io"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"

	encoding_flatjson "local/james-orcales/shared/encoding/flatjson"
	invariant "local/james-orcales/shared/invariant"
)

// Recorder re-exports the library type so callers importing only this package
// can refer to it without a second import.
type Recorder = invariant.Recorder

// Assertion_Metadata re-exports the library coverage-tracker entry type.
type Assertion_Metadata = invariant.Assertion_Metadata

// Sugar_Package_Marker gives registration an import identity without a hard-coded module path.
type Sugar_Package_Marker struct{}

// Default is the OS-bound Recorder backing the package-level sugar. Tests that
// need to redirect I/O construct their own Recorder via the pure invariant
// package; Default serves the common case.
var Default = Init_Default_Recorder()

// Init_Default_Recorder builds a Recorder wired to the host OS: the local
// filesystem rooted at "/", os.Stderr, and os.Exit. It sniffs os.Args once for
// the test / fuzz / benchmark environment flags. No caller seam is wired — an
// assertion is identified by its message, not its source location.
func Init_Default_Recorder() (recorder *invariant.Recorder) {
	is_test, is_fuzz, is_fuzz_worker, is_benchmark := running_environment_flags()
	// /dev/tty bypasses `go test`'s stdout/stderr capture so the success summary
	// shows without -v. Assigned only on success: a nil *os.File stored in an
	// io.Writer interface is a non-nil interface, defeating the consumer's nil check.
	var tty io.Writer
	opened, open_error := os.OpenFile("/dev/tty", os.O_WRONLY, 0)
	if open_error == nil {
		tty = opened
	}
	working_directory, _ := os.Getwd()
	recorder = &invariant.Recorder{
		Output:              os.Stderr,
		Tty:                 tty,
		File_System:         os.DirFS("/"),
		Exit:                os.Exit,
		Is_Test:             is_test,
		Is_Fuzz:             is_fuzz,
		Is_Fuzz_Worker:      is_fuzz_worker,
		Is_Benchmark:        is_benchmark,
		Packages_To_Analyze: []string{"."},
		Working_Directory:   working_directory,
		Sugar_Package:       reflect.TypeOf(Sugar_Package_Marker{}).PkgPath(),
	}
	recorder.Report_Overflow = coverage_gap_overflow_write
	recorder_output_configure(recorder, os.Getenv(OUTPUT_ENVIRONMENT))
	return recorder
}

// Persists a report too large for a terminal and returns where it put it. The file holds JSON in
// each output mode, because a spilled report is read by a later pass and never by eye. It stays on
// disk after the run: it is the artifact the terminal line points at.
func coverage_gap_overflow_write(
	gaps []invariant.Coverage_Gap,
) (path string, err error) {
	file, create_error := os.CreateTemp("", "invariant-coverage-gaps-*.json")
	if create_error != nil {
		return "", create_error
	}
	write_error := Coverage_Gap_Json_Write(file, gaps)
	close_error := file.Close()
	if write_error != nil {
		return "", write_error
	}
	if close_error != nil {
		return "", close_error
	}
	return file.Name(), nil
}

// Sniffs os.Args for the go-test harness flags that distinguish a plain test run from a fuzz
// coordinator, a fuzz worker subprocess, or a benchmark. "-test.fuzzworker" is a strict prefix
// extension of the "-test.fuzz" the is_fuzz check also matches, so a worker reports both is_fuzz
// and is_fuzz_worker; the recorder gates coverage on is_fuzz_worker.
func running_environment_flags() (
	is_test bool, is_fuzz bool, is_fuzz_worker bool, is_benchmark bool,
) {
	for _, argument := range os.Args {
		if strings.HasPrefix(argument, "-test.fuzzworker") {
			is_fuzz_worker = true
		}
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
	return is_test, is_fuzz, is_fuzz_worker, is_benchmark
}

// Run_Test_Main adds OS-backed fuzz coverage and the selected report format to the core runner.
func Run_Test_Main(m *testing.M, directories ...string) {
	fuzz_coverage_setup(Default)
	invariant.Recorder_Run_Test_Main(Default, m, directories...)
}

// OUTPUT_ENVIRONMENT selects the complete coverage-gap representation for a test process.
const OUTPUT_ENVIRONMENT = "INVARIANT_OUTPUT"

// Coverage_Gap_Json_Write emits one compact flat array and its terminating newline.
func Coverage_Gap_Json_Write(output io.Writer, gaps []invariant.Coverage_Gap) (err error) {
	if marshal_error := encoding_flatjson.Marshal_Write(output, gaps); marshal_error != nil {
		return marshal_error
	}
	written, write_error := io.WriteString(output, "\n")
	if write_error != nil {
		return write_error
	}
	if written != 1 {
		return io.ErrShortWrite
	}
	return nil
}

// Applies one already-read environment value before the suite can emit a different format.
func recorder_output_configure(recorder *invariant.Recorder, output string) {
	recorder.Output_Configuration_Diagnostic = ""
	switch output {
	case "", "table":
		recorder.Report_Coverage_Gaps = invariant.Coverage_Gap_Table_Write
	case "json":
		recorder.Report_Coverage_Gaps = Coverage_Gap_Json_Write
	default:
		recorder.Report_Coverage_Gaps = nil
		recorder.Output_Configuration_Diagnostic =
			OUTPUT_ENVIRONMENT + " has unknown value " + strconv.Quote(output) +
				"; expected \"table\" or \"json\""
	}
}

// FUZZ_COVERAGE_FILE_ENVIRONMENT names the env var a fuzz coordinator sets to the shared
// coverage file path. The -test.fuzzworker subprocesses it spawns inherit the env (Go captures
// os.Environ() when starting workers), so they find the same file.
const FUZZ_COVERAGE_FILE_ENVIRONMENT = "INVARIANT_FUZZ_COVERAGE_FILE"

// Fuzz_coverage_setup wires the cross-process coverage seams for a fuzzing run. Under -fuzz
// the coordinator never executes the fuzzed body — that happens in worker subprocesses — so
// each worker appends every newly-covered branch to one shared append-only file, and the
// coordinator unions that file into its registered entries before analyzing. A plain test or
// benchmark wires nothing.
func fuzz_coverage_setup(recorder *invariant.Recorder) {
	if recorder.Is_Fuzz_Worker {
		path := os.Getenv(FUZZ_COVERAGE_FILE_ENVIRONMENT)
		if path == "" {
			return
		}
		file, open_error := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
		if open_error != nil {
			return
		}
		// One Write per line under O_APPEND: the kernel serializes appends across worker
		// processes, so no mutex (useless across processes) and no flock are needed.
		// The line format lives in Fuzz_Coverage_Line so it round-trips with the merge.
		recorder.Coverage_Sink = func(key string, fired_true bool) {
			io.WriteString(file, invariant.Fuzz_Coverage_Line(key, fired_true))
		}
		return
	}
	if !recorder.Is_Fuzz {
		return
	}
	// Coordinator: create the shared file now (before m.Run spawns workers) and hand its
	// path to them via the inherited env; union and remove it after the run.
	file, create_error := os.CreateTemp("", "invariant-fuzz-coverage-*.tsv")
	if create_error != nil {
		return
	}
	path := file.Name()
	file.Close()
	os.Setenv(FUZZ_COVERAGE_FILE_ENVIRONMENT, path)
	recorder.Merge_Fuzz_Coverage = func() {
		opened, open_error := os.Open(path)
		if open_error == nil {
			invariant.Recorder_Merge_Fuzz_Coverage_From(recorder, opened)
			opened.Close()
		}
		os.Remove(path)
	}
}

// Register_Packages_For_Analysis forwards to the library function on Default.
func Register_Packages_For_Analysis(directories ...string) {
	invariant.Recorder_Register_Packages_For_Analysis(Default, directories...)
}

// Analyze_Assertion_Frequency forwards to the library function on Default.
func Analyze_Assertion_Frequency() {
	invariant.Recorder_Analyze_Assertion_Frequency(Default)
}

// Always is an eager guard, so its enforcement and reachability stay independent of any chain.
func Always[T ~bool](condition T, message string) {
	invariant.Recorder_Always(Default, condition, message)
}

// Sometimes states one inline two-branch axis on Default, for a body that owns no bundle.
func Sometimes[T ~bool](condition T, message string) {
	invariant.Recorder_Sometimes(Default, condition, message)
}

// Range bounds one inline value to a contiguous interval on Default.
func Range[Value invariant.Integer](
	value Value, minimum Value, maximum Value, message string,
) {
	invariant.Recorder_Range(Default, value, minimum, maximum, message)
}

// Enum holds one inline value to two members on Default.
func Enum[Value invariant.Integer](
	value Value, first Value, second Value, message string,
) {
	invariant.Recorder_Enum(Default, value, first, second, message)
}

// Range_Holed removes four fixed exclusions from an inline interval on Default. A duplicate final
// hole stands for an unused slot, so one arity serves every width.
func Range_Holed[Value invariant.Integer](
	value Value, minimum Value, maximum Value,
	hole_1 Value, hole_2 Value, hole_3 Value, hole_4 Value, message string,
) {
	invariant.Recorder_Range_Holed(
		Default, value, minimum, maximum, hole_1, hole_2, hole_3, hole_4, message)
}

// Assertion_Builder re-exports the fluent value for explicit APIs without adding an adapter.
type Assertion_Builder = invariant.Assertion_Builder

// Namespace re-exports the chain identity used by every _Invariants helper.
type Namespace = invariant.Namespace

// Tree starts one deferred builder on Default. The subject supplies the chain type.
func Tree[Subject any](
	subject Subject, namespace Namespace,
) (builder Assertion_Builder) {
	return invariant.Recorder_Tree(Default, subject, namespace)
}
