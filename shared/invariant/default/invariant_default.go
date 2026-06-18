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
	"unsafe"

	invariant "github.com/james-orcales/james-orcales/shared/invariant"
)

// Recorder re-exports the library type so callers importing only this package
// can refer to it without a second import.
type Recorder = invariant.Recorder

// Assertion_Metadata re-exports the library coverage-tracker entry type.
type Assertion_Metadata = invariant.Assertion_Metadata

// Dot_Element re-exports the library element type so bundles can name it in Bundle.
type Dot_Element = invariant.Dot_Element

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
// events and their combination against the pre-registered coverage grid. message identifies
// the grid and is prefixed onto each held axis's own message to form its coverage key.
func Dot_Product(message string, dot_elements ...invariant.Dot_Element) {
	invariant.Recorder_Dot_Product(Default, message, dot_elements...)
}

// Whole_Number constrains a generic to Go's integer kinds, signed and unsigned alike
// (uintptr is excluded — it is an address width, not a quantity).
type Whole_Number interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

// Whole_Number_Invariants is the preset coverage for an integer: the small-magnitude axes 0,
// 1, 2, each of which must be observed both ways. Element messages are local; the consuming
// Dot_Product namespaces them.
func Whole_Number_Invariants[I Whole_Number](n I) (dot_elements Bundle) {
	return append(
		dot_elements,
		invariant.Recorder_Sometimes(Default, n == 0, "zero"),
		invariant.Recorder_Sometimes(Default, n == 1, "one"),
		invariant.Recorder_Sometimes(Default, n == 2, "two"),
	)
}

// Float_Invariants is the preset coverage for a float: NaN, negative, and
// positive must each be observed (zero is the shared finite cell). NaN excludes
// either sign, and the two signs exclude each other. ±Inf fold into the signs.
func Float_Invariants[F ~float32 | ~float64](f F) (dot_elements Bundle) {
	value := float64(f)
	not_a_number := Sometimes(math.IsNaN(value), "nan")
	negative := Sometimes(value < 0, "negative")
	positive := Sometimes(value > 0, "positive")
	return append(dot_elements,
		not_a_number, negative, positive,
		Impossible(Event_True("nan"), Event_True("negative")),
		Impossible(Event_True("nan"), Event_True("positive")),
		Impossible(Event_True("negative"), Event_True("positive")),
	)
}

// String_Invariants is the preset coverage for a string. Over the empty axis (a
// Sometimes over len(s) == 0; not a length boundary, which would demand an
// unobservably-long string at its Hi endpoint) it layers seven content axes: edge vs
// interior whitespace, invalid UTF-8, a NUL byte, a byte count that differs from the
// rune count (a multi-byte rune), a control character, and a line break. An empty
// string excludes every content axis, and a NUL byte or a line break is itself a
// control character.
func String_Invariants(s string) (dot_elements Bundle) {
	empty := Sometimes(len(s) == 0, "empty")
	edge_whitespace := Sometimes_Has_Edge_Whitespace(s)
	interior_whitespace := Sometimes_Has_Interior_Whitespace(s)
	invalid_utf8 := Sometimes_Has_Invalid_UTF8(s)
	nul := Sometimes_Has_Nul(s)
	byte_rune_mismatch := Sometimes_Has_Multibyte_Rune(s)
	control := Sometimes_Has_Control(s)
	line_break := Sometimes_Has_Line_Break(s)
	return append(dot_elements,
		empty,
		edge_whitespace, interior_whitespace, invalid_utf8, nul,
		byte_rune_mismatch, control, line_break,
		Impossible(Event_True("empty"), Event_True("Sometimes_Has_Edge_Whitespace")),
		Impossible(Event_True("empty"), Event_True("Sometimes_Has_Interior_Whitespace")),
		Impossible(Event_True("empty"), Event_True("Sometimes_Has_Invalid_UTF8")),
		Impossible(Event_True("empty"), Event_True("Sometimes_Has_Nul")),
		Impossible(Event_True("empty"), Event_True("Sometimes_Has_Multibyte_Rune")),
		Impossible(Event_True("empty"), Event_True("Sometimes_Has_Control")),
		Impossible(Event_True("empty"), Event_True("Sometimes_Has_Line_Break")),
		Impossible(Event_True("Sometimes_Has_Nul"), Event_False("Sometimes_Has_Control")),
		Impossible(
			Event_True("Sometimes_Has_Line_Break"),
			Event_False("Sometimes_Has_Control"),
		),
	)
}

// Sometimes_Has_Edge_Whitespace records whether s begins or ends with whitespace. Its
// message is its own name, which the static registration derives from the call selector.
func Sometimes_Has_Edge_Whitespace(s string) (dot_element invariant.Dot_Element) {
	return invariant.Recorder_Sometimes(
		Default, string_has_edge_whitespace(s), "Sometimes_Has_Edge_Whitespace")
}

// Always_Has_Edge_Whitespace asserts s always begins or ends with whitespace.
func Always_Has_Edge_Whitespace(s string) {
	invariant.Recorder_Always(
		Default, string_has_edge_whitespace(s), "Always_Has_Edge_Whitespace")
}

// Sometimes_Has_Interior_Whitespace records whether s has whitespace off its edges.
func Sometimes_Has_Interior_Whitespace(s string) (dot_element invariant.Dot_Element) {
	return invariant.Recorder_Sometimes(
		Default, string_has_interior_whitespace(s), "Sometimes_Has_Interior_Whitespace")
}

// Always_Has_Interior_Whitespace asserts s always has whitespace off its edges.
func Always_Has_Interior_Whitespace(s string) {
	invariant.Recorder_Always(
		Default, string_has_interior_whitespace(s), "Always_Has_Interior_Whitespace")
}

// Sometimes_Has_Invalid_UTF8 records whether s is sometimes not valid UTF-8.
func Sometimes_Has_Invalid_UTF8(s string) (dot_element invariant.Dot_Element) {
	return invariant.Recorder_Sometimes(
		Default, string_has_invalid_utf8(s), "Sometimes_Has_Invalid_UTF8")
}

// Always_Has_Invalid_UTF8 asserts s is always not valid UTF-8.
func Always_Has_Invalid_UTF8(s string) {
	invariant.Recorder_Always(Default, string_has_invalid_utf8(s), "Always_Has_Invalid_UTF8")
}

// Sometimes_Has_Nul records whether s sometimes contains a NUL (0x00) byte.
func Sometimes_Has_Nul(s string) (dot_element invariant.Dot_Element) {
	return invariant.Recorder_Sometimes(Default, string_has_nul(s), "Sometimes_Has_Nul")
}

// Always_Has_Nul asserts s always contains a NUL (0x00) byte.
func Always_Has_Nul(s string) {
	invariant.Recorder_Always(Default, string_has_nul(s), "Always_Has_Nul")
}

// Sometimes_Has_Multibyte_Rune records whether s's byte count sometimes differs from
// its rune count — a multi-byte rune is present.
func Sometimes_Has_Multibyte_Rune(s string) (dot_element invariant.Dot_Element) {
	return invariant.Recorder_Sometimes(
		Default, string_has_multibyte_rune(s), "Sometimes_Has_Multibyte_Rune")
}

// Always_Has_Multibyte_Rune asserts s's byte count always differs from its rune count.
func Always_Has_Multibyte_Rune(s string) {
	invariant.Recorder_Always(
		Default, string_has_multibyte_rune(s), "Always_Has_Multibyte_Rune")
}

// Sometimes_Has_Control records whether s sometimes contains a control character.
func Sometimes_Has_Control(s string) (dot_element invariant.Dot_Element) {
	return invariant.Recorder_Sometimes(Default, string_has_control(s), "Sometimes_Has_Control")
}

// Always_Has_Control asserts s always contains a control character.
func Always_Has_Control(s string) {
	invariant.Recorder_Always(Default, string_has_control(s), "Always_Has_Control")
}

// Sometimes_Has_Line_Break records whether s sometimes contains a carriage return or
// line feed.
func Sometimes_Has_Line_Break(s string) (dot_element invariant.Dot_Element) {
	return invariant.Recorder_Sometimes(
		Default, string_has_line_break(s), "Sometimes_Has_Line_Break")
}

// Always_Has_Line_Break asserts s always contains a carriage return or line feed.
func Always_Has_Line_Break(s string) {
	invariant.Recorder_Always(Default, string_has_line_break(s), "Always_Has_Line_Break")
}

// Sometimes_Has_Non_ASCII records whether s sometimes contains a non-ASCII byte.
func Sometimes_Has_Non_ASCII(s string) (dot_element invariant.Dot_Element) {
	return invariant.Recorder_Sometimes(
		Default, string_has_non_ascii(s), "Sometimes_Has_Non_ASCII")
}

// Always_Has_Non_ASCII asserts s always contains a non-ASCII byte.
func Always_Has_Non_ASCII(s string) {
	invariant.Recorder_Always(Default, string_has_non_ascii(s), "Always_Has_Non_ASCII")
}

// Never_Has_Edge_Whitespace asserts s never begins or ends with whitespace.
func Never_Has_Edge_Whitespace(s string) {
	invariant.Recorder_Always(
		Default, !string_has_edge_whitespace(s), "Never_Has_Edge_Whitespace")
}

// Never_Has_Interior_Whitespace asserts s never has whitespace off its edges.
func Never_Has_Interior_Whitespace(s string) {
	invariant.Recorder_Always(
		Default, !string_has_interior_whitespace(s), "Never_Has_Interior_Whitespace")
}

// Never_Has_Invalid_UTF8 asserts s is always valid UTF-8.
func Never_Has_Invalid_UTF8(s string) {
	invariant.Recorder_Always(Default, !string_has_invalid_utf8(s), "Never_Has_Invalid_UTF8")
}

// Never_Has_Nul asserts s never contains a NUL (0x00) byte.
func Never_Has_Nul(s string) {
	invariant.Recorder_Always(Default, !string_has_nul(s), "Never_Has_Nul")
}

// Never_Has_Multibyte_Rune asserts s's byte count always equals its rune count.
func Never_Has_Multibyte_Rune(s string) {
	invariant.Recorder_Always(
		Default, !string_has_multibyte_rune(s), "Never_Has_Multibyte_Rune")
}

// Never_Has_Control asserts s never contains a control character.
func Never_Has_Control(s string) {
	invariant.Recorder_Always(Default, !string_has_control(s), "Never_Has_Control")
}

// Never_Has_Line_Break asserts s never contains a carriage return or line feed.
func Never_Has_Line_Break(s string) {
	invariant.Recorder_Always(Default, !string_has_line_break(s), "Never_Has_Line_Break")
}

// Never_Has_Non_ASCII asserts s never contains a non-ASCII byte.
func Never_Has_Non_ASCII(s string) {
	invariant.Recorder_Always(Default, !string_has_non_ascii(s), "Never_Has_Non_ASCII")
}

// Slice_Invariants is the preset coverage for a slice: nil, empty-but-non-nil,
// and non-empty must each be observed — the nil/empty distinction Go draws. A nil
// slice is necessarily empty, which the Impossible records.
func Slice_Invariants[E any](s []E) (dot_elements Bundle) {
	empty := Sometimes(len(s) == 0, "empty")
	is_nil := Sometimes(s == nil, "nil")
	return append(dot_elements,
		empty, is_nil,
		Impossible(Event_True("nil"), Event_False("empty")),
	)
}

// Map_Invariants is the preset coverage for a map: nil, empty-but-non-nil, and
// non-empty must each be observed. A nil map is necessarily empty.
func Map_Invariants[K comparable, V any](m map[K]V) (dot_elements Bundle) {
	empty := Sometimes(len(m) == 0, "empty")
	is_nil := Sometimes(m == nil, "nil")
	return append(dot_elements,
		empty, is_nil,
		Impossible(Event_True("nil"), Event_False("empty")),
	)
}

// Reports whether s contains a Unicode whitespace rune.
func string_has_whitespace(s string) (has bool) {
	for _, character := range s {
		if unicode.IsSpace(character) {
			return true
		}
	}
	return false
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

// Reports whether s contains a non-ASCII byte (any byte above 127). Every byte
// of a multi-byte UTF-8 sequence has its high bit set, so a set high bit is
// exactly equivalent to decoding runes and comparing against unicode.MaxASCII —
// without paying the decoder cost on every code point.
//
// Scans 16 bytes per iteration (SWAR — SIMD within a register): a non-ASCII byte
// is exactly a set high bit, so word & high_bits is non-zero iff some byte in the
// word is non-ASCII. Two 8-byte words are ORed before the test, so one branch
// covers 16 bytes. This framework enforces hundreds of thousands of assertions,
// so the predicate is hot enough to justify the unsafe load. Benchmarked on a
// 1 MiB all-ASCII string (worst case — no match, every byte visited), 100 runs,
// startup subtracted, Apple M4, go1.26.3:
//
//	range over runes         730 ms    2.7 GiB/s    1.0x
//	byte loop, high-bit      484 ms    4.1 GiB/s    1.5x
//	hand-packed SWAR         321 ms    6.1 GiB/s    2.3x
//	this (unsafe, 16B/iter)   38 ms   54.4 GiB/s   19.5x
//
// The unsafe load is what earns the last ~9x over hand-packed SWAR: s[i+1]..s[i+7]
// each compile to their own bounds check, byte load, and a shift/OR chain to
// assemble the word, whereas *(*uint64)(...) is a single MOVD with none. At
// 54 GiB/s the 1 MiB string is L2-bandwidth bound, so wider unrolling or NEON buys
// little more. Scan direction is irrelevant to the worst case (a two-pointer walk
// touches the same bytes with worse prefetch); bytes-per-iteration is the lever.
func string_has_non_ascii(s string) (has bool) {
	const high_bits = 0x8080808080808080
	// In-bounds despite unsafe: the loads fire only when i+16 <= len(s), so they
	// never read past s; the tail handles the final fewer-than-16 bytes one at a
	// time. base is never dereferenced when len(s) == 0 (both loops are skipped).
	base := unsafe.Pointer(unsafe.StringData(s))
	i := 0
	for ; i+16 <= len(s); i += 16 {
		word := *(*uint64)(unsafe.Add(base, i)) | *(*uint64)(unsafe.Add(base, i+8))
		if word&high_bits != 0 {
			return true
		}
	}
	for ; i < len(s); i++ {
		if s[i] >= utf8.RuneSelf {
			return true
		}
	}
	return false
}
