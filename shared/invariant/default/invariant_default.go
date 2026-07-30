// Package invariant_default is the composition-tier sibling of invariant. It
// wires the pure library to the real OS — local filesystem, stderr, os.Args
// sniffing, os.Exit — and re-exports the surface. Import it aliased as invariant
// and use invariant.Always / invariant.Assertions as if no split between pure
// and OS-bound tiers existed.
package invariant

import (
	"io"
	"math"
	"os"
	"reflect"
	"strings"
	"testing"

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
	return &invariant.Recorder{
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

// Run_Test_Main is the canonical TestMain body: register, run the suite, report coverage gaps,
// exit with the suite's code. Under -fuzz it first wires cross-process coverage (see
// fuzz_coverage_setup) so worker subprocesses' exploration reaches the coordinator's analysis.
func Run_Test_Main(m *testing.M, directories ...string) {
	fuzz_coverage_setup(Default)
	invariant.Recorder_Run_Test_Main(Default, m, directories...)
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

// Assertion_Builder re-exports the fluent value for explicit APIs without adding an adapter.
type Assertion_Builder = invariant.Assertion_Builder

// Namespace re-exports the chain identity used by every _Invariants helper.
type Namespace = invariant.Namespace

// Assertions starts one deferred builder on Default.
func Assertions(namespace Namespace) (builder Assertion_Builder) {
	return invariant.Recorder_Assertions(Default, namespace)
}

// A compile-time failure on narrow-int platforms prevents the int/uint helpers from claiming the
// 64-bit boundaries their messages name.
const INT_IS_64_BITS_WIDE = uint(math.MaxInt - math.MaxInt64)

// Int_Invariants keeps the platform integer's conventional edge witnesses mandatory.
func Int_Invariants(n int, namespace Namespace) {
	Assertions(namespace).
		Sometimes(n == 1, "The value is one.").
		Sometimes(n == -1, "The value is negative one.").
		Sometimes(n == math.MinInt64, "The value is the minimum int.").
		Sometimes(n == math.MaxInt64, "The value is the maximum int.").
		Ensure()
}

// Int8_Invariants keeps the signed eight-bit edge witnesses mandatory.
func Int8_Invariants(n int8, namespace Namespace) {
	Assertions(namespace).
		Sometimes(n == 1, "The value is one.").
		Sometimes(n == -1, "The value is negative one.").
		Sometimes(n == math.MinInt8, "The value is the minimum int8.").
		Sometimes(n == math.MaxInt8, "The value is the maximum int8.").
		Ensure()
}

// Int16_Invariants keeps the signed sixteen-bit edge witnesses mandatory.
func Int16_Invariants(n int16, namespace Namespace) {
	Assertions(namespace).
		Sometimes(n == 1, "The value is one.").
		Sometimes(n == -1, "The value is negative one.").
		Sometimes(n == math.MinInt16, "The value is the minimum int16.").
		Sometimes(n == math.MaxInt16, "The value is the maximum int16.").
		Ensure()
}

// Int32_Invariants keeps the signed thirty-two-bit edge witnesses mandatory.
func Int32_Invariants(n int32, namespace Namespace) {
	Assertions(namespace).
		Sometimes(n == 1, "The value is one.").
		Sometimes(n == -1, "The value is negative one.").
		Sometimes(n == math.MinInt32, "The value is the minimum int32.").
		Sometimes(n == math.MaxInt32, "The value is the maximum int32.").
		Ensure()
}

// Int64_Invariants keeps the signed sixty-four-bit edge witnesses mandatory.
func Int64_Invariants(n int64, namespace Namespace) {
	Assertions(namespace).
		Sometimes(n == 1, "The value is one.").
		Sometimes(n == -1, "The value is negative one.").
		Sometimes(n == math.MinInt64, "The value is the minimum int64.").
		Sometimes(n == math.MaxInt64, "The value is the maximum int64.").
		Ensure()
}

// Uint_Invariants keeps the platform unsigned integer's edge witnesses mandatory.
func Uint_Invariants(n uint, namespace Namespace) {
	Assertions(namespace).
		Sometimes(n == 0, "The value is zero.").
		Sometimes(n == 1, "The value is one.").
		Sometimes(n == math.MaxUint64, "The value is the maximum uint.").
		Ensure()
}

// Uint8_Invariants keeps the unsigned eight-bit edge witnesses mandatory.
func Uint8_Invariants(n uint8, namespace Namespace) {
	Assertions(namespace).
		Sometimes(n == 0, "The value is zero.").
		Sometimes(n == 1, "The value is one.").
		Sometimes(n == math.MaxUint8, "The value is the maximum uint8.").
		Ensure()
}

// Uint16_Invariants keeps the unsigned sixteen-bit edge witnesses mandatory.
func Uint16_Invariants(n uint16, namespace Namespace) {
	Assertions(namespace).
		Sometimes(n == 0, "The value is zero.").
		Sometimes(n == 1, "The value is one.").
		Sometimes(n == math.MaxUint16, "The value is the maximum uint16.").
		Ensure()
}

// Uint32_Invariants keeps the unsigned thirty-two-bit edge witnesses mandatory.
func Uint32_Invariants(n uint32, namespace Namespace) {
	Assertions(namespace).
		Sometimes(n == 0, "The value is zero.").
		Sometimes(n == 1, "The value is one.").
		Sometimes(n == math.MaxUint32, "The value is the maximum uint32.").
		Ensure()
}

// Uint64_Invariants keeps the unsigned sixty-four-bit edge witnesses mandatory.
func Uint64_Invariants(n uint64, namespace Namespace) {
	Assertions(namespace).
		Sometimes(n == 0, "The value is zero.").
		Sometimes(n == 1, "The value is one.").
		Sometimes(n == math.MaxUint64, "The value is the maximum uint64.").
		Ensure()
}

// Float64_Invariants keeps non-finite values visible in generated coverage.
func Float64_Invariants(f float64, namespace Namespace) {
	Assertions(namespace).
		Sometimes(math.IsNaN(f), "The value is NaN.").
		Sometimes(f == math.Inf(-1), "The value is negative infinity.").
		Sometimes(f == math.Inf(1), "The value is positive infinity.").
		Ensure()
}

// Float32_Invariants keeps non-finite values visible in generated coverage.
func Float32_Invariants(f float32, namespace Namespace) {
	Assertions(namespace).
		Sometimes(math.IsNaN(float64(f)), "The value is NaN.").
		Sometimes(float64(f) == math.Inf(-1), "The value is negative infinity.").
		Sometimes(float64(f) == math.Inf(1), "The value is positive infinity.").
		Ensure()
}

// Boolean_Invariants requires both values without a manually namespaced message.
func Boolean_Invariants(b bool, namespace Namespace) {
	Assertions(namespace).Sometimes(b, "The value is true.").Ensure()
}
