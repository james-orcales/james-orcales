package invariant_test

import (
	"math"
	"testing"

	core "github.com/james-orcales/james-orcales/shared/invariant"
	"github.com/james-orcales/james-orcales/shared/invariant/default"
)

// Seeds Default's tracker with the named axes under namespace, so a subsequent self-emitting
// preset call records into resolvable entries (the static scan seeds these in a real run).
func seed_preset_axes(namespace string, messages ...string) {
	for _, message := range messages {
		key := namespace + core.Element_Message_Separator + message
		invariant.Default.Events.Store(key, &core.Assertion_Metadata{
			Kind: core.Assertion_Kind_Sometimes, Message: key,
		})
	}
}

// Renders, as a compact "T"/"F" signature, which of the named axes recorded a true event under
// namespace — read from Default's tracker after a single self-emitting preset call.
func recorded_signature(namespace string, messages ...string) (signature string) {
	for _, message := range messages {
		key := namespace + core.Element_Message_Separator + message
		value, loaded := invariant.Default.Events.Load(key)
		if !loaded {
			signature += "F"
			continue
		}
		if value.(*core.Assertion_Metadata).Frequency.Load() > 0 {
			signature += "T"
			continue
		}
		signature += "F"
	}
	return signature
}

// Float64_Invariants records NaN, negative infinity, and positive infinity; an ordinary value holds
// none of them, and the three are mutually exclusive.
func Test_Float64_Invariants_Tracks_Special_Values(t *testing.T) {
	axes := []string{
		"The value is NaN.",
		"The value is negative infinity.",
		"The value is positive infinity.",
	}
	cases := []struct {
		Name string
		F    float64
		Want string
	}{
		{"nan", math.NaN(), "TFF"},
		{"negative infinity", math.Inf(-1), "FTF"},
		{"positive infinity", math.Inf(1), "FFT"},
		{"ordinary", 2, "FFF"},
	}
	for _, c := range cases {
		namespace := "test.float64." + c.Name
		seed_preset_axes(namespace, axes...)
		invariant.Float64_Invariants(c.F, invariant.Namespace(namespace))
		if got := recorded_signature(namespace, axes...); got != c.Want {
			t.Errorf("%s: %q, want %q", c.Name, got, c.Want)
		}
	}
}

// Int8_Invariants records the unit values and the type bounds; an ordinary value holds none of
// them, and the four are mutually exclusive.
func Test_Int8_Invariants_Tracks_Values(t *testing.T) {
	axes := []string{
		"The value is one.",
		"The value is negative one.",
		"The value is the minimum int8.",
		"The value is the maximum int8.",
	}
	cases := []struct {
		Name string
		N    int8
		Want string
	}{
		{"one", 1, "TFFF"},
		{"negative one", -1, "FTFF"},
		{"int8 minimum", math.MinInt8, "FFTF"},
		{"int8 maximum", math.MaxInt8, "FFFT"},
		{"ordinary", 5, "FFFF"},
	}
	for _, c := range cases {
		namespace := "test.int8." + c.Name
		seed_preset_axes(namespace, axes...)
		invariant.Int8_Invariants(c.N, invariant.Namespace(namespace))
		if got := recorded_signature(namespace, axes...); got != c.Want {
			t.Errorf("%s: %q, want %q", c.Name, got, c.Want)
		}
	}
}

// Uint8_Invariants records zero, one, and the maximum; an ordinary non-zero value holds none of
// them, and an unsigned value never has a sign axis.
func Test_Uint8_Invariants_Tracks_Values(t *testing.T) {
	axes := []string{
		"The value is zero.",
		"The value is one.",
		"The value is the maximum uint8.",
	}
	cases := []struct {
		Name string
		N    uint8
		Want string
	}{
		{"zero", 0, "TFF"},
		{"one", 1, "FTF"},
		{"ordinary", 5, "FFF"},
		{"uint8 maximum", math.MaxUint8, "FFT"},
	}
	for _, c := range cases {
		namespace := "test.uint8." + c.Name
		seed_preset_axes(namespace, axes...)
		invariant.Uint8_Invariants(c.N, invariant.Namespace(namespace))
		if got := recorded_signature(namespace, axes...); got != c.Want {
			t.Errorf("%s: %q, want %q", c.Name, got, c.Want)
		}
	}
}

// Every numeric primitive has its own preset; this runs each so a wrong type fails the build,
// without re-asserting the per-axis behavior above.
func Test_Numeric_Presets_Run_For_Every_Primitive(t *testing.T) {
	invariant.Int_Invariants(0, "n.int")
	invariant.Int8_Invariants(0, "n.int8")
	invariant.Int16_Invariants(0, "n.int16")
	invariant.Int32_Invariants(0, "n.int32")
	invariant.Int64_Invariants(0, "n.int64")
	invariant.Uint_Invariants(0, "n.uint")
	invariant.Uint8_Invariants(0, "n.uint8")
	invariant.Uint16_Invariants(0, "n.uint16")
	invariant.Uint32_Invariants(0, "n.uint32")
	invariant.Uint64_Invariants(0, "n.uint64")
	invariant.Float32_Invariants(0, "n.float32")
	invariant.Float64_Invariants(0, "n.float64")
}

// String_Invariants returns [empty, edge_ws, interior_ws, invalid_utf8, nul,
// byte_rune, control, line_break, 9 Impossibles] = 17 elements; the leading 8
// Sometimes signature must track the string's content.
func Test_String_Invariants_Tracks_Content(t *testing.T) {
	cases := []struct {
		Name string
		S    string
		Want string
	}{
		// Signature order: empty, edge_ws, interior_ws, invalid_utf8, nul,
		// byte_rune, control, line_break.
		{"empty", "", "TFFFFFFF"},
		{"plain ascii", "ab", "FFFFFFFF"},
		{"edge whitespace", " ab", "FTFFFFFF"},
		{"interior whitespace", "a b", "FFTFFFFF"},
		{"multibyte rune", "café", "FFFFFTFF"},
		{"invalid utf8", "\xff", "FFFTFFFF"},
		{"nul is control", "a\x00b", "FFFFTFTF"},
		{"line break is whitespace and control", "a\nb", "FFTFFFTT"},
	}
	axes := []string{
		"The value is empty.",
		"The value has edge whitespace.",
		"The value has interior whitespace.",
		"The value has invalid UTF-8.",
		"The value has a NUL byte.",
		"The value has a multi-byte rune.",
		"The value has a control character.",
		"The value has a line break.",
	}
	for _, c := range cases {
		namespace := "test.string." + c.Name
		seed_preset_axes(namespace, axes...)
		invariant.String_Invariants(c.S, invariant.Namespace(namespace))
		if got := recorded_signature(namespace, axes...); got != c.Want {
			t.Errorf("%s: signature = %q, want %q", c.Name, got, c.Want)
		}
	}
}

// Slice_Invariants returns [empty, is_nil, Impossible]; it distinguishes nil from
// empty-but-non-nil from populated.
func Test_Slice_Invariants_Distinguishes_Nil_And_Empty(t *testing.T) {
	cases := []struct {
		Name string
		S    []int
		Want string
	}{
		{"nil", nil, "TT"},
		{"empty non-nil", []int{}, "TF"},
		{"populated", []int{1}, "FF"},
	}
	for _, c := range cases {
		namespace := "test.slice." + c.Name
		seed_preset_axes(namespace, "empty", "nil")
		invariant.Slice_Invariants(c.S, invariant.Namespace(namespace))
		if got := recorded_signature(namespace, "empty", "nil"); got != c.Want {
			t.Errorf("%s: [empty is_nil] = %q, want %q", c.Name, got, c.Want)
		}
	}
}

// Map_Invariants returns [empty, is_nil, Impossible]; like the slice preset it
// distinguishes nil from empty-but-non-nil from populated.
func Test_Map_Invariants_Distinguishes_Nil_And_Empty(t *testing.T) {
	cases := []struct {
		Name string
		M    map[string]int
		Want string
	}{
		{"nil", nil, "TT"},
		{"empty non-nil", map[string]int{}, "TF"},
		{"populated", map[string]int{"a": 1}, "FF"},
	}
	for _, c := range cases {
		namespace := "test.map." + c.Name
		seed_preset_axes(namespace, "empty", "nil")
		invariant.Map_Invariants(c.M, invariant.Namespace(namespace))
		if got := recorded_signature(namespace, "empty", "nil"); got != c.Want {
			t.Errorf("%s: [empty is_nil] = %q, want %q", c.Name, got, c.Want)
		}
	}
}
