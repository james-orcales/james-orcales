// Package invariant_default is the composition-tier sibling of invariant. It
// wires the pure library to the real OS — local filesystem, stderr, os.Args
// sniffing, runtime.Callers, os.Exit — and re-exports the surface. Import it
// aliased as invariant and use invariant.Always / invariant.Sometimes /
// invariant.Dot_Product as if no split between pure and OS-bound tiers existed.
package invariant_default

import (
	"io"
	"math"
	"os"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"
	"unsafe"

	invariant "github.com/james-orcales/james-orcales/shared/invariant/v3"
)

// Recorder re-exports the library type so callers importing only this package
// can refer to it without a second import.
type Recorder = invariant.Recorder

// Assertion_Metadata re-exports the library coverage-tracker entry type.
type Assertion_Metadata = invariant.Assertion_Metadata

// Dot_Element re-exports the library element type so bundles can declare their
// return type as []invariant.Dot_Element.
type Dot_Element = invariant.Dot_Element

// Dot_Element_Reference re-exports the library reference type used by Impossible.
type Dot_Element_Reference = invariant.Dot_Element_Reference

// Event re-exports the library Event alias so Event_True / Event_False read the
// same here as in the pure tier.
type Event = invariant.Event

// Numeric re-exports the library constraint for Boundary_Input's type parameter.
type Numeric = invariant.Numeric

// Boundary_Input re-exports the library type so callers can write
// invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{...}).
type Boundary_Input[I invariant.Numeric] = invariant.Boundary_Input[I]

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
// filesystem rooted at "/", os.Stderr, os.Exit, and runtime.Callers. It sniffs
// os.Args once for the test / fuzz / benchmark environment flags.
func Init_Default_Recorder() (recorder *invariant.Recorder) {
	is_test, is_fuzz, is_benchmark := running_environment_flags()
	// /dev/tty bypasses `go test`'s stdout/stderr capture so the success summary
	// shows without -v. Assigned only on success: a nil *os.File stored in an
	// io.Writer interface is a non-nil interface, defeating the consumer's nil check.
	var tty io.Writer
	opened, open_error := os.OpenFile("/dev/tty", os.O_WRONLY, 0)
	if open_error == nil {
		tty = opened
	}
	return &invariant.Recorder{
		Output:      os.Stderr,
		Tty:         tty,
		File_System: os.DirFS("/"),
		Get_Caller: func(skip int) (file string, line int) {
			// The +1 compensates for this closure's own frame: callers pass skip
			// as if Get_Caller were a transparent passthrough, but the closure
			// body is itself one frame above runtime.Callers.
			program_counters := [1]uintptr{}
			count := runtime.Callers(skip+1, program_counters[:])
			frame, _ := runtime.CallersFrames(program_counters[:count]).Next()
			return frame.File, frame.Line
		},
		Exit:                os.Exit,
		Is_Test:             is_test,
		Is_Fuzz:             is_fuzz,
		Is_Benchmark:        is_benchmark,
		Packages_To_Analyze: []string{"."},
		Sugar_Package:       reflect.TypeOf(sugar_package_marker{}).PkgPath(),
	}
}

// Sniffs os.Args for the go-test harness flags that distinguish a plain test run
// from a fuzz or benchmark run.
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

// Run_Test_Main is the canonical TestMain body: register, run the suite, report
// coverage gaps, exit with the suite's code.
func Run_Test_Main(m *testing.M, directories ...string) {
	invariant.Recorder_Run_Test_Main(Default, m, directories...)
}

// Register_Packages_For_Analysis forwards to the library function on Default.
func Register_Packages_For_Analysis(directories ...string) {
	invariant.Recorder_Register_Packages_For_Analysis(Default, directories...)
}

// Analyze_Assertion_Frequency forwards to the library function on Default.
func Analyze_Assertion_Frequency() {
	invariant.Recorder_Analyze_Assertion_Frequency(Default)
}

// Always builds an element asserting condition holds on every call. Only
// Dot_Product enforces and reports it; an Always alone observes nothing.
//
//go:noinline
func Always[T ~bool](condition T) (dot_element invariant.Dot_Element) {
	return invariant.Recorder_Always(Default, condition)
}

// Sometimes builds an element whose condition must be observed both true and
// false across the run. Only Dot_Product enforces coverage of both branches.
//
//go:noinline
func Sometimes[T ~bool](condition T) (dot_element invariant.Dot_Element) {
	return invariant.Recorder_Sometimes(Default, condition)
}

// Distinct_Boundary builds an element asserting Lo < Hi and Lo <= X <= Hi,
// tracking which endpoint X lands on. Like the other producers it never panics
// on its own — enforcement fires only when the element is consumed by Dot_Product.
//
//go:noinline
func Distinct_Boundary[I invariant.Numeric](
	input *invariant.Boundary_Input[I],
) (dot_element invariant.Dot_Element) {
	return invariant.Recorder_Distinct_Boundary(Default, input)
}

// Impossible declares that the referenced element events must never all co-occur on
// the same call, and globs over the axes it does not name (see invariant.Impossible).
// Pure packaging — no Recorder needed.
func Impossible(
	impossibles ...invariant.Dot_Element_Reference,
) (dot_element invariant.Dot_Element) {
	return invariant.Impossible(impossibles...)
}

// Event_True references event at its true outcome, for use inside Impossible.
func Event_True(event invariant.Event) (reference invariant.Dot_Element_Reference) {
	return invariant.Event_True(event)
}

// Event_False references event at its false outcome, for use inside Impossible.
func Event_False(event invariant.Event) (reference invariant.Dot_Element_Reference) {
	return invariant.Event_False(event)
}

// Dot_Product enforces the call's elements and, under test, records the observed
// element events and their combination against the pre-registered coverage grid.
//
//go:noinline
func Dot_Product(dot_elements ...invariant.Dot_Element) {
	invariant.Recorder_Dot_Product(Default, dot_elements...)
}

// Whole_Number constrains a generic to Go's integer kinds, signed and unsigned alike
// (uintptr is excluded — it is an address width, not a quantity).
type Whole_Number interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

// Reports whether the integer type I is signed. -1 is representable only by a signed
// type; in an unsigned one I(0)-I(1) wraps to the type maximum, which is not < 0. The
// expression is a compile-time-foldable constant once I is instantiated.
func whole_number_is_signed[I Whole_Number]() (signed bool) {
	return I(0)-I(1) < 0
}

// whole_number_max returns the largest value of the integer type I. A width-B unsigned
// integer maxes at 2^(8B) - 1 (all bits set); a signed one at 2^(8B-1) - 1, one bit
// narrower for the sign. The shift derives B from the type's byte size, so the bound is
// exact for every kind in Whole_Number and any ~-defined type over them.
func whole_number_max[I Whole_Number]() (hi I) {
	bits := uint(unsafe.Sizeof(hi)) * 8
	if whole_number_is_signed[I]() {
		return I(^uint64(0) >> (65 - bits))
	}
	return I(^uint64(0) >> (64 - bits))
}

// whole_number_min returns the smallest value of the integer type I: 0 for an unsigned
// type, and for a signed one the two's-complement floor -2^(8B-1), one below the negated
// max. The negation is dead for unsigned I (it returns 0 first) but still must compile,
// which it does — unary minus on an unsigned value is legal and wraps.
func whole_number_min[I Whole_Number]() (lo I) {
	if !whole_number_is_signed[I]() {
		return 0
	}
	return -whole_number_max[I]() - 1
}

// Whole_Number_Invariants is the preset coverage for an integer: the small-magnitude
// axes 0, 1, 2 (each must be observed both ways) plus a Distinct_Boundary over the
// type's full range [whole_number_min, whole_number_max]. The boundary tracks reaching
// each endpoint, so the run is covered once the value is observed at both the floor and
// the ceiling.
func Whole_Number_Invariants[I Whole_Number](n I) (dot_elements []invariant.Dot_Element) {
	return append(
		dot_elements,
		invariant.Recorder_Sometimes(Default, n == 0),
		invariant.Recorder_Sometimes(Default, n == 1),
		invariant.Recorder_Sometimes(Default, n == 2),
		invariant.Recorder_Distinct_Boundary(Default, &Boundary_Input[I]{
			X:  n,
			Lo: whole_number_min[I](),
			Hi: whole_number_max[I](),
		}),
	)
}

// Float_Invariants is the preset coverage for a float: NaN, negative, and
// positive must each be observed (zero is the shared finite cell). NaN excludes
// either sign, and the two signs exclude each other. ±Inf fold into the signs.
func Float_Invariants[F ~float32 | ~float64](f F) (dot_elements []invariant.Dot_Element) {
	value := float64(f)
	not_a_number := Sometimes(math.IsNaN(value))
	negative := Sometimes(value < 0)
	positive := Sometimes(value > 0)
	return append(dot_elements,
		not_a_number, negative, positive,
		Impossible(Event_True(not_a_number), Event_True(negative)),
		Impossible(Event_True(not_a_number), Event_True(positive)),
		Impossible(Event_True(negative), Event_True(positive)),
	)
}

// String_Invariants is the preset coverage for a string. Over the empty axis (a
// Sometimes over len(s) == 0; not a length boundary, which would demand an
// unobservably-long string at its Hi endpoint) it layers seven content axes: edge vs
// interior whitespace, invalid UTF-8, a NUL byte, a byte count that differs from the
// rune count (a multi-byte rune), a control character, and a line break. An empty
// string excludes every content axis, and a NUL byte or a line break is itself a
// control character.
func String_Invariants(s string) (dot_elements []invariant.Dot_Element) {
	empty := Sometimes(len(s) == 0)
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
		Impossible(Event_True(empty), Event_True(edge_whitespace)),
		Impossible(Event_True(empty), Event_True(interior_whitespace)),
		Impossible(Event_True(empty), Event_True(invalid_utf8)),
		Impossible(Event_True(empty), Event_True(nul)),
		Impossible(Event_True(empty), Event_True(byte_rune_mismatch)),
		Impossible(Event_True(empty), Event_True(control)),
		Impossible(Event_True(empty), Event_True(line_break)),
		Impossible(Event_True(nul), Event_False(control)),
		Impossible(Event_True(line_break), Event_False(control)),
	)
}

// Sometimes_Has_Edge_Whitespace records whether s begins or ends with whitespace.
//
//go:noinline
func Sometimes_Has_Edge_Whitespace(s string) (dot_element invariant.Dot_Element) {
	return invariant.Recorder_Sometimes(Default, string_has_edge_whitespace(s))
}

// Always_Has_Edge_Whitespace asserts s always begins or ends with whitespace.
//
//go:noinline
func Always_Has_Edge_Whitespace(s string) (dot_element invariant.Dot_Element) {
	return invariant.Recorder_Always(Default, string_has_edge_whitespace(s))
}

// Sometimes_Has_Interior_Whitespace records whether s has whitespace off its edges.
//
//go:noinline
func Sometimes_Has_Interior_Whitespace(s string) (dot_element invariant.Dot_Element) {
	return invariant.Recorder_Sometimes(Default, string_has_interior_whitespace(s))
}

// Always_Has_Interior_Whitespace asserts s always has whitespace off its edges.
//
//go:noinline
func Always_Has_Interior_Whitespace(s string) (dot_element invariant.Dot_Element) {
	return invariant.Recorder_Always(Default, string_has_interior_whitespace(s))
}

// Sometimes_Has_Invalid_UTF8 records whether s is sometimes not valid UTF-8.
//
//go:noinline
func Sometimes_Has_Invalid_UTF8(s string) (dot_element invariant.Dot_Element) {
	return invariant.Recorder_Sometimes(Default, string_has_invalid_utf8(s))
}

// Always_Has_Invalid_UTF8 asserts s is always not valid UTF-8.
//
//go:noinline
func Always_Has_Invalid_UTF8(s string) (dot_element invariant.Dot_Element) {
	return invariant.Recorder_Always(Default, string_has_invalid_utf8(s))
}

// Sometimes_Has_Nul records whether s sometimes contains a NUL (0x00) byte.
//
//go:noinline
func Sometimes_Has_Nul(s string) (dot_element invariant.Dot_Element) {
	return invariant.Recorder_Sometimes(Default, string_has_nul(s))
}

// Always_Has_Nul asserts s always contains a NUL (0x00) byte.
//
//go:noinline
func Always_Has_Nul(s string) (dot_element invariant.Dot_Element) {
	return invariant.Recorder_Always(Default, string_has_nul(s))
}

// Sometimes_Has_Multibyte_Rune records whether s's byte count sometimes differs from
// its rune count — a multi-byte rune is present.
//
//go:noinline
func Sometimes_Has_Multibyte_Rune(s string) (dot_element invariant.Dot_Element) {
	return invariant.Recorder_Sometimes(Default, string_has_multibyte_rune(s))
}

// Always_Has_Multibyte_Rune asserts s's byte count always differs from its rune count.
//
//go:noinline
func Always_Has_Multibyte_Rune(s string) (dot_element invariant.Dot_Element) {
	return invariant.Recorder_Always(Default, string_has_multibyte_rune(s))
}

// Sometimes_Has_Control records whether s sometimes contains a control character.
//
//go:noinline
func Sometimes_Has_Control(s string) (dot_element invariant.Dot_Element) {
	return invariant.Recorder_Sometimes(Default, string_has_control(s))
}

// Always_Has_Control asserts s always contains a control character.
//
//go:noinline
func Always_Has_Control(s string) (dot_element invariant.Dot_Element) {
	return invariant.Recorder_Always(Default, string_has_control(s))
}

// Sometimes_Has_Line_Break records whether s sometimes contains a carriage return or
// line feed.
//
//go:noinline
func Sometimes_Has_Line_Break(s string) (dot_element invariant.Dot_Element) {
	return invariant.Recorder_Sometimes(Default, string_has_line_break(s))
}

// Always_Has_Line_Break asserts s always contains a carriage return or line feed.
//
//go:noinline
func Always_Has_Line_Break(s string) (dot_element invariant.Dot_Element) {
	return invariant.Recorder_Always(Default, string_has_line_break(s))
}

// Sometimes_Has_Non_ASCII records whether s sometimes contains a non-ASCII byte.
//
//go:noinline
func Sometimes_Has_Non_ASCII(s string) (dot_element invariant.Dot_Element) {
	return invariant.Recorder_Sometimes(Default, string_has_non_ascii(s))
}

// Always_Has_Non_ASCII asserts s always contains a non-ASCII byte.
//
//go:noinline
func Always_Has_Non_ASCII(s string) (dot_element invariant.Dot_Element) {
	return invariant.Recorder_Always(Default, string_has_non_ascii(s))
}

// Never_Has_Edge_Whitespace asserts s never begins or ends with whitespace.
//
//go:noinline
func Never_Has_Edge_Whitespace(s string) (dot_element invariant.Dot_Element) {
	return invariant.Recorder_Always(Default, !string_has_edge_whitespace(s))
}

// Never_Has_Interior_Whitespace asserts s never has whitespace off its edges.
//
//go:noinline
func Never_Has_Interior_Whitespace(s string) (dot_element invariant.Dot_Element) {
	return invariant.Recorder_Always(Default, !string_has_interior_whitespace(s))
}

// Never_Has_Invalid_UTF8 asserts s is always valid UTF-8.
//
//go:noinline
func Never_Has_Invalid_UTF8(s string) (dot_element invariant.Dot_Element) {
	return invariant.Recorder_Always(Default, !string_has_invalid_utf8(s))
}

// Never_Has_Nul asserts s never contains a NUL (0x00) byte.
//
//go:noinline
func Never_Has_Nul(s string) (dot_element invariant.Dot_Element) {
	return invariant.Recorder_Always(Default, !string_has_nul(s))
}

// Never_Has_Multibyte_Rune asserts s's byte count always equals its rune count.
//
//go:noinline
func Never_Has_Multibyte_Rune(s string) (dot_element invariant.Dot_Element) {
	return invariant.Recorder_Always(Default, !string_has_multibyte_rune(s))
}

// Never_Has_Control asserts s never contains a control character.
//
//go:noinline
func Never_Has_Control(s string) (dot_element invariant.Dot_Element) {
	return invariant.Recorder_Always(Default, !string_has_control(s))
}

// Never_Has_Line_Break asserts s never contains a carriage return or line feed.
//
//go:noinline
func Never_Has_Line_Break(s string) (dot_element invariant.Dot_Element) {
	return invariant.Recorder_Always(Default, !string_has_line_break(s))
}

// Never_Has_Non_ASCII asserts s never contains a non-ASCII byte.
//
//go:noinline
func Never_Has_Non_ASCII(s string) (dot_element invariant.Dot_Element) {
	return invariant.Recorder_Always(Default, !string_has_non_ascii(s))
}

// Slice_Invariants is the preset coverage for a slice: nil, empty-but-non-nil,
// and non-empty must each be observed — the nil/empty distinction Go draws. A nil
// slice is necessarily empty, which the Impossible records.
func Slice_Invariants[E any](s []E) (dot_elements []invariant.Dot_Element) {
	empty := Sometimes(len(s) == 0)
	is_nil := Sometimes(s == nil)
	return append(dot_elements,
		empty, is_nil,
		Impossible(Event_True(is_nil), Event_False(empty)),
	)
}

// Map_Invariants is the preset coverage for a map: nil, empty-but-non-nil, and
// non-empty must each be observed. A nil map is necessarily empty.
func Map_Invariants[K comparable, V any](m map[K]V) (dot_elements []invariant.Dot_Element) {
	empty := Sometimes(len(m) == 0)
	is_nil := Sometimes(m == nil)
	return append(dot_elements,
		empty, is_nil,
		Impossible(Event_True(is_nil), Event_False(empty)),
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
// startup subtracted, Apple M4, go1.25.7:
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
