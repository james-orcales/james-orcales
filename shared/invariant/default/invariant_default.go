// Package invariant_default is the composition-tier sibling of invariant. It
// wires the pure library to the real OS — local filesystem, stderr, os.Args
// sniffing, os.Exit — and re-exports the surface. Import it aliased as invariant
// and use invariant.Always / invariant.Sometimes / invariant.Dot_Product as if no
// split between pure and OS-bound tiers existed.
package invariant

import (
	"io"
	"math"
	"os"
	"reflect"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

	invariant "github.com/james-orcales/james-orcales/shared/invariant"
)

// Recorder re-exports the library type so callers importing only this package
// can refer to it without a second import.
type Recorder = invariant.Recorder

// Assertion_Metadata re-exports the library coverage-tracker entry type.
type Assertion_Metadata = invariant.Assertion_Metadata

// Dot_Element re-exports the library element type so bundles can name it in Bundle.
type Dot_Element = invariant.Dot_Element

// Namespace re-exports the library grid-identity type a _Invariants takes and self-emits under.
type Namespace = invariant.Namespace

// Bundle re-exports the library Bundle alias ([]Dot_Element) — the return type of a
// _Invariants function, spread into a Dot_Product.
type Bundle = invariant.Bundle

// Dot_Element_Reference re-exports the library reference type used by Impossible.
type Dot_Element_Reference = invariant.Dot_Element_Reference

// Marker type whose reflect-reported import path is this package's own, so
// Init_Default_Recorder can hand the static registration this package's path
// (derived, not hardcoded) to recognise the unqualified primitive calls inside
// the presets defined here.
type sugar_package_marker struct{}

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
		Sugar_Package:       reflect.TypeOf(sugar_package_marker{}).PkgPath(),
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

// Fuzz_coverage_file_environment names the env var a fuzz coordinator sets to the shared
// coverage file path. The -test.fuzzworker subprocesses it spawns inherit the env (Go captures
// os.Environ() when starting workers), so they find the same file.
const fuzz_coverage_file_environment = "INVARIANT_FUZZ_COVERAGE_FILE"

// Fuzz_coverage_setup wires the cross-process coverage seams for a fuzzing run. Under -fuzz
// the coordinator never executes the fuzzed body — that happens in worker subprocesses — so
// each worker appends every newly-covered cell to one shared append-only file, and the
// coordinator unions that file into its registered grid before analyzing. A plain test or
// benchmark wires nothing.
func fuzz_coverage_setup(recorder *invariant.Recorder) {
	if recorder.Is_Fuzz_Worker {
		path := os.Getenv(fuzz_coverage_file_environment)
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
	os.Setenv(fuzz_coverage_file_environment, path)
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

// Always is an eager guard: it panics immediately when condition is false, naming itself by
// message. It is not an element and is never passed to Dot_Product. Under a plain test run a
// never-reached Always surfaces as a reachability gap.
func Always[T ~bool](condition T, message string) {
	invariant.Recorder_Always(Default, condition, message)
}

// Sometimes builds an element whose condition must be observed both true and false across
// the run; message is its own identity, prefixed by the consuming Dot_Product. Only
// Dot_Product enforces coverage of both branches.
func Sometimes[T ~bool](condition T, message string) (dot_element invariant.Dot_Element) {
	return invariant.Recorder_Sometimes(Default, condition, message)
}

// Imply builds a gated Sometimes: an axis recorded only on a call where prerequisite holds,
// don't-care otherwise, and excluded from the grid. condition is evaluated eagerly, so a
// condition safe only under the prerequisite must self-guard. AND prerequisites to nest.
func Imply[P ~bool, C ~bool](
	prerequisite P, condition C, message string,
) (dot_element invariant.Dot_Element) {
	return invariant.Recorder_Imply(Default, prerequisite, condition, message)
}

// Impossible declares that the referenced element events must never all co-occur on
// the same call, and globs over the axes it does not name (see invariant.Impossible).
// Pure packaging — no Recorder needed.
func Impossible(
	impossibles ...invariant.Dot_Element_Reference,
) (dot_element invariant.Dot_Element) {
	return invariant.Impossible(impossibles...)
}

// Event_True references the axis with message at its true outcome, for use inside Impossible.
func Event_True(message string) (reference invariant.Dot_Element_Reference) {
	return invariant.Event_True(message)
}

// Event_False references the axis with message at its false outcome, for use inside Impossible.
func Event_False(message string) (reference invariant.Dot_Element_Reference) {
	return invariant.Event_False(message)
}

// Dot_Product enforces the call's elements and, under test, records the observed element
// events and their combination against the pre-registered coverage grid. namespace identifies
// the grid and is prefixed onto each held axis's own message to form its coverage key.
func Dot_Product(namespace Namespace, dot_elements ...invariant.Dot_Element) {
	invariant.Recorder_Dot_Product(Default, namespace, dot_elements...)
}

// An int is assumed to be 64 bits wide, so that the minimum and maximum axes below are exactly an
// int's bounds. This conversion fails to compile on a platform where int is narrower, because there
// math.MaxInt is smaller than math.MaxInt64 and their difference is negative, which a uint cannot
// represent. An int and a uint always share a width, so this guards the unsigned presets that name
// math.MaxUint as well.
const _ = uint(math.MaxInt - math.MaxInt64)

// Int_Invariants is the preset coverage for an int. The suite must witness the value one, negative
// one, the type's minimum, and the type's maximum, plus an ordinary value that is none of these.
// The four are mutually exclusive, which the carves enforce.
func Int_Invariants(n int, namespace Namespace) {
	Dot_Product(namespace,
		Sometimes(n == 1, "The value is one."),
		Sometimes(n == -1, "The value is negative one."),
		Sometimes(n == math.MinInt64, "The value is the minimum int."),
		Sometimes(n == math.MaxInt64, "The value is the maximum int."),
		Impossible(
			Event_True("The value is one."),
			Event_True("The value is negative one."),
		),
		Impossible(
			Event_True("The value is one."),
			Event_True("The value is the minimum int."),
		),
		Impossible(
			Event_True("The value is one."),
			Event_True("The value is the maximum int."),
		),
		Impossible(
			Event_True("The value is negative one."),
			Event_True("The value is the minimum int."),
		),
		Impossible(
			Event_True("The value is negative one."),
			Event_True("The value is the maximum int."),
		),
		Impossible(
			Event_True("The value is the minimum int."),
			Event_True("The value is the maximum int."),
		),
	)
}

// Int8_Invariants is Int_Invariants for an int8, whose minimum and maximum are MinInt8 and MaxInt8.
func Int8_Invariants(n int8, namespace Namespace) {
	Dot_Product(namespace,
		Sometimes(n == 1, "The value is one."),
		Sometimes(n == -1, "The value is negative one."),
		Sometimes(n == math.MinInt8, "The value is the minimum int8."),
		Sometimes(n == math.MaxInt8, "The value is the maximum int8."),
		Impossible(
			Event_True("The value is one."),
			Event_True("The value is negative one."),
		),
		Impossible(
			Event_True("The value is one."),
			Event_True("The value is the minimum int8."),
		),
		Impossible(
			Event_True("The value is one."),
			Event_True("The value is the maximum int8."),
		),
		Impossible(
			Event_True("The value is negative one."),
			Event_True("The value is the minimum int8."),
		),
		Impossible(
			Event_True("The value is negative one."),
			Event_True("The value is the maximum int8."),
		),
		Impossible(
			Event_True("The value is the minimum int8."),
			Event_True("The value is the maximum int8."),
		),
	)
}

// Int16_Invariants is Int_Invariants for an int16, whose bounds are MinInt16 and MaxInt16.
func Int16_Invariants(n int16, namespace Namespace) {
	Dot_Product(namespace,
		Sometimes(n == 1, "The value is one."),
		Sometimes(n == -1, "The value is negative one."),
		Sometimes(n == math.MinInt16, "The value is the minimum int16."),
		Sometimes(n == math.MaxInt16, "The value is the maximum int16."),
		Impossible(
			Event_True("The value is one."),
			Event_True("The value is negative one."),
		),
		Impossible(
			Event_True("The value is one."),
			Event_True("The value is the minimum int16."),
		),
		Impossible(
			Event_True("The value is one."),
			Event_True("The value is the maximum int16."),
		),
		Impossible(
			Event_True("The value is negative one."),
			Event_True("The value is the minimum int16."),
		),
		Impossible(
			Event_True("The value is negative one."),
			Event_True("The value is the maximum int16."),
		),
		Impossible(
			Event_True("The value is the minimum int16."),
			Event_True("The value is the maximum int16."),
		),
	)
}

// Int32_Invariants is Int_Invariants for an int32, whose bounds are MinInt32 and MaxInt32.
func Int32_Invariants(n int32, namespace Namespace) {
	Dot_Product(namespace,
		Sometimes(n == 1, "The value is one."),
		Sometimes(n == -1, "The value is negative one."),
		Sometimes(n == math.MinInt32, "The value is the minimum int32."),
		Sometimes(n == math.MaxInt32, "The value is the maximum int32."),
		Impossible(
			Event_True("The value is one."),
			Event_True("The value is negative one."),
		),
		Impossible(
			Event_True("The value is one."),
			Event_True("The value is the minimum int32."),
		),
		Impossible(
			Event_True("The value is one."),
			Event_True("The value is the maximum int32."),
		),
		Impossible(
			Event_True("The value is negative one."),
			Event_True("The value is the minimum int32."),
		),
		Impossible(
			Event_True("The value is negative one."),
			Event_True("The value is the maximum int32."),
		),
		Impossible(
			Event_True("The value is the minimum int32."),
			Event_True("The value is the maximum int32."),
		),
	)
}

// Int64_Invariants is Int_Invariants for an int64, whose bounds are MinInt64 and MaxInt64.
func Int64_Invariants(n int64, namespace Namespace) {
	Dot_Product(namespace,
		Sometimes(n == 1, "The value is one."),
		Sometimes(n == -1, "The value is negative one."),
		Sometimes(n == math.MinInt64, "The value is the minimum int64."),
		Sometimes(n == math.MaxInt64, "The value is the maximum int64."),
		Impossible(
			Event_True("The value is one."),
			Event_True("The value is negative one."),
		),
		Impossible(
			Event_True("The value is one."),
			Event_True("The value is the minimum int64."),
		),
		Impossible(
			Event_True("The value is one."),
			Event_True("The value is the maximum int64."),
		),
		Impossible(
			Event_True("The value is negative one."),
			Event_True("The value is the minimum int64."),
		),
		Impossible(
			Event_True("The value is negative one."),
			Event_True("The value is the maximum int64."),
		),
		Impossible(
			Event_True("The value is the minimum int64."),
			Event_True("The value is the maximum int64."),
		),
	)
}

// Uint_Invariants is the preset coverage for a uint. The suite must witness the value zero, one,
// and the type's maximum, plus the ordinary non-zero case that holds none of those axes. An
// unsigned value has no sign, so zero stands in for the sign axis and the minimum is zero itself.
// The carves keep zero, one, and the maximum mutually exclusive.
func Uint_Invariants(n uint, namespace Namespace) {
	Dot_Product(namespace,
		Sometimes(n == 0, "The value is zero."),
		Sometimes(n == 1, "The value is one."),
		Sometimes(n == math.MaxUint64, "The value is the maximum uint."),
		Impossible(
			Event_True("The value is zero."),
			Event_True("The value is one."),
		),
		Impossible(
			Event_True("The value is zero."),
			Event_True("The value is the maximum uint."),
		),
		Impossible(
			Event_True("The value is one."),
			Event_True("The value is the maximum uint."),
		),
	)
}

// Uint8_Invariants is Uint_Invariants for a uint8, whose maximum is MaxUint8.
func Uint8_Invariants(n uint8, namespace Namespace) {
	Dot_Product(namespace,
		Sometimes(n == 0, "The value is zero."),
		Sometimes(n == 1, "The value is one."),
		Sometimes(n == math.MaxUint8, "The value is the maximum uint8."),
		Impossible(
			Event_True("The value is zero."),
			Event_True("The value is one."),
		),
		Impossible(
			Event_True("The value is zero."),
			Event_True("The value is the maximum uint8."),
		),
		Impossible(
			Event_True("The value is one."),
			Event_True("The value is the maximum uint8."),
		),
	)
}

// Uint16_Invariants is Uint_Invariants for a uint16, whose maximum is MaxUint16.
func Uint16_Invariants(n uint16, namespace Namespace) {
	Dot_Product(namespace,
		Sometimes(n == 0, "The value is zero."),
		Sometimes(n == 1, "The value is one."),
		Sometimes(n == math.MaxUint16, "The value is the maximum uint16."),
		Impossible(
			Event_True("The value is zero."),
			Event_True("The value is one."),
		),
		Impossible(
			Event_True("The value is zero."),
			Event_True("The value is the maximum uint16."),
		),
		Impossible(
			Event_True("The value is one."),
			Event_True("The value is the maximum uint16."),
		),
	)
}

// Uint32_Invariants is Uint_Invariants for a uint32, whose maximum is MaxUint32.
func Uint32_Invariants(n uint32, namespace Namespace) {
	Dot_Product(namespace,
		Sometimes(n == 0, "The value is zero."),
		Sometimes(n == 1, "The value is one."),
		Sometimes(n == math.MaxUint32, "The value is the maximum uint32."),
		Impossible(
			Event_True("The value is zero."),
			Event_True("The value is one."),
		),
		Impossible(
			Event_True("The value is zero."),
			Event_True("The value is the maximum uint32."),
		),
		Impossible(
			Event_True("The value is one."),
			Event_True("The value is the maximum uint32."),
		),
	)
}

// Uint64_Invariants is Uint_Invariants for a uint64, whose maximum is MaxUint64.
func Uint64_Invariants(n uint64, namespace Namespace) {
	Dot_Product(namespace,
		Sometimes(n == 0, "The value is zero."),
		Sometimes(n == 1, "The value is one."),
		Sometimes(n == math.MaxUint64, "The value is the maximum uint64."),
		Impossible(
			Event_True("The value is zero."),
			Event_True("The value is one."),
		),
		Impossible(
			Event_True("The value is zero."),
			Event_True("The value is the maximum uint64."),
		),
		Impossible(
			Event_True("The value is one."),
			Event_True("The value is the maximum uint64."),
		),
	)
}

// Float64_Invariants is the preset coverage for a float64. The suite must witness NaN, negative
// infinity, and positive infinity, plus an ordinary value that is none of these. The three are
// mutually exclusive, which the carves enforce.
func Float64_Invariants(f float64, namespace Namespace) {
	Dot_Product(namespace,
		Sometimes(math.IsNaN(f), "The value is NaN."),
		Sometimes(f == math.Inf(-1), "The value is negative infinity."),
		Sometimes(f == math.Inf(1), "The value is positive infinity."),
		Impossible(
			Event_True("The value is NaN."),
			Event_True("The value is negative infinity."),
		),
		Impossible(
			Event_True("The value is NaN."),
			Event_True("The value is positive infinity."),
		),
		Impossible(
			Event_True("The value is negative infinity."),
			Event_True("The value is positive infinity."),
		),
	)
}

// Float32_Invariants is Float64_Invariants for a float32, widened to float64 for the comparisons.
func Float32_Invariants(f float32, namespace Namespace) {
	value := float64(f)
	Dot_Product(namespace,
		Sometimes(math.IsNaN(value), "The value is NaN."),
		Sometimes(value == math.Inf(-1), "The value is negative infinity."),
		Sometimes(value == math.Inf(1), "The value is positive infinity."),
		Impossible(
			Event_True("The value is NaN."),
			Event_True("The value is negative infinity."),
		),
		Impossible(
			Event_True("The value is NaN."),
			Event_True("The value is positive infinity."),
		),
		Impossible(
			Event_True("The value is negative infinity."),
			Event_True("The value is positive infinity."),
		),
	)
}

// String_Invariants is the preset coverage for a string. Over the empty axis (a
// Sometimes over len(s) == 0; not a length boundary, which would demand an
// unobservably-long string at its Hi endpoint) it layers seven content axes: edge vs
// interior whitespace, invalid UTF-8, a NUL byte, a byte count that differs from the
// rune count (a multi-byte rune), a control character, and a line break. An empty
// string excludes every content axis, and a NUL byte or a line break is itself a
// control character.
func String_Invariants(s string, namespace Namespace) {
	Dot_Product(namespace,
		Sometimes(len(s) == 0, "The value is empty."),
		Sometimes(string_has_edge_whitespace(s), "The value has edge whitespace."),
		Sometimes(string_has_interior_whitespace(s), "The value has interior whitespace."),
		Sometimes(string_has_invalid_utf8(s), "The value has invalid UTF-8."),
		Sometimes(string_has_nul(s), "The value has a NUL byte."),
		Sometimes(string_has_multibyte_rune(s), "The value has a multi-byte rune."),
		Sometimes(string_has_control(s), "The value has a control character."),
		Sometimes(string_has_line_break(s), "The value has a line break."),
		Impossible(
			Event_True("The value is empty."),
			Event_True("The value has edge whitespace."),
		),
		Impossible(
			Event_True("The value is empty."),
			Event_True("The value has interior whitespace."),
		),
		Impossible(
			Event_True("The value is empty."),
			Event_True("The value has invalid UTF-8."),
		),
		Impossible(
			Event_True("The value is empty."),
			Event_True("The value has a NUL byte."),
		),
		Impossible(
			Event_True("The value is empty."),
			Event_True("The value has a multi-byte rune."),
		),
		Impossible(
			Event_True("The value is empty."),
			Event_True("The value has a control character."),
		),
		Impossible(
			Event_True("The value is empty."),
			Event_True("The value has a line break."),
		),
		Impossible(
			Event_True("The value has a NUL byte."),
			Event_False("The value has a control character."),
		),
		Impossible(
			Event_True("The value has a line break."),
			Event_False("The value has a control character."),
		),
	)
}

// Slice_Invariants is the preset coverage for a slice: nil, empty-but-non-nil,
// and non-empty must each be observed — the nil/empty distinction Go draws. A nil
// slice is necessarily empty, which the Impossible records.
func Slice_Invariants[E any](s []E, namespace Namespace) {
	Dot_Product(namespace,
		Sometimes(len(s) == 0, "empty"),
		Sometimes(s == nil, "nil"),
		Impossible(Event_True("nil"), Event_False("empty")),
	)
}

// Map_Invariants is the preset coverage for a map: nil, empty-but-non-nil, and
// non-empty must each be observed. A nil map is necessarily empty.
func Map_Invariants[K comparable, V any](m map[K]V, namespace Namespace) {
	Dot_Product(namespace,
		Sometimes(len(m) == 0, "empty"),
		Sometimes(m == nil, "nil"),
		Impossible(Event_True("nil"), Event_False("empty")),
	)
}

// Reports whether the first or last rune of s is a Unicode whitespace rune.
func string_has_edge_whitespace(s string) (has bool) {
	if s == "" {
		return false
	}
	first, _ := utf8.DecodeRuneInString(s)
	if unicode.IsSpace(first) {
		return true
	}
	last, _ := utf8.DecodeLastRuneInString(s)
	return unicode.IsSpace(last)
}

// Reports whether s has a Unicode whitespace rune at a position that is neither the
// first nor the last rune — whitespace the edge check does not already account for.
func string_has_interior_whitespace(s string) (has bool) {
	runes := []rune(s)
	for index := 1; index < len(runes)-1; index++ {
		if unicode.IsSpace(runes[index]) {
			return true
		}
	}
	return false
}

// Reports whether s contains a Unicode control character.
func string_has_control(s string) (has bool) {
	for _, character := range s {
		if unicode.IsControl(character) {
			return true
		}
	}
	return false
}

// Reports whether s is not valid UTF-8.
func string_has_invalid_utf8(s string) (has bool) {
	return !utf8.ValidString(s)
}

// Reports whether s contains a NUL (0x00) byte.
func string_has_nul(s string) (has bool) {
	return strings.IndexByte(s, 0) >= 0
}

// Reports whether s's byte count differs from its rune count — it contains a
// multi-byte rune.
func string_has_multibyte_rune(s string) (has bool) {
	return len(s) != utf8.RuneCountInString(s)
}

// Reports whether s contains a carriage return or line feed.
func string_has_line_break(s string) (has bool) {
	return strings.ContainsAny(s, "\r\n")
}
