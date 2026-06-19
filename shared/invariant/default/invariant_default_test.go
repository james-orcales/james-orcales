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

// Float_Invariants returns [NaN, negative, positive, 3 Impossibles]; a NaN lands
// only on the NaN axis, ±Inf fold into the signs.
func Test_Float_Invariants_Tracks_Special_Values(t *testing.T) {
	cases := []struct {
		Name string
		F    float64
		Want string
	}{
		{"NaN", math.NaN(), "TFF"},
		{"negative", -2, "FTF"},
		{"positive infinity", math.Inf(1), "FFT"},
		{"zero", 0, "FFF"},
	}
	for _, c := range cases {
		namespace := "test.float." + c.Name
		seed_preset_axes(namespace, "nan", "negative", "positive")
		invariant.Float_Invariants(c.F, invariant.Namespace(namespace))
		got := recorded_signature(namespace, "nan", "negative", "positive")
		if got != c.Want {
			t.Errorf("%s: [nan negative positive] = %q, want %q", c.Name, got, c.Want)
		}
	}
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
		"empty", "Sometimes_Has_Edge_Whitespace", "Sometimes_Has_Interior_Whitespace",
		"Sometimes_Has_Invalid_UTF8", "Sometimes_Has_Nul", "Sometimes_Has_Multibyte_Rune",
		"Sometimes_Has_Control", "Sometimes_Has_Line_Break",
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

// Test_Never_Has_Panics_When_Present verifies a Never_Has_X helper is an eager Always: it
// panics when the string carries the forbidden property and is a no-op when it does not.
func Test_Never_Has_Panics_When_Present(t *testing.T) {
	if did_panic(func() { invariant.Never_Has_Control("abc") }) {
		t.Error(`Never_Has_Control("abc") must not panic — no control char`)
	}
	if !did_panic(func() { invariant.Never_Has_Control("a\x00b") }) {
		t.Error(`Never_Has_Control("a\x00b") must panic — NUL is a control char`)
	}
}

// Reports whether calling action panics, recovering so the test can assert on it.
func did_panic(action func()) (panicked bool) {
	defer func() {
		if recover() != nil {
			panicked = true
		}
	}()
	action()
	return false
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
