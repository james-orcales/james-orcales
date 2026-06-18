package invariant_test

import (
	"math"
	"testing"

	core "github.com/james-orcales/james-orcales/shared/invariant"
	"github.com/james-orcales/james-orcales/shared/invariant/default"
)

// Renders the True/False outcome of a preset's leading count Sometimes elements
// as a compact signature ("T"/"F" per element), for comparing against an expected
// pattern.
func event_signature(elements []core.Dot_Element, count int) (signature string) {
	for index := 0; index < count; index++ {
		if elements[index].Event {
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
		elements := invariant.Float_Invariants(c.F)
		if len(elements) != 6 {
			t.Fatalf("%s: returned %d elements, want 6", c.Name, len(elements))
		}
		if got := event_signature(elements, 3); got != c.Want {
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
	for _, c := range cases {
		elements := invariant.String_Invariants(c.S)
		if len(elements) != 17 {
			t.Fatalf("%s: returned %d elements, want 17", c.Name, len(elements))
		}
		if got := event_signature(elements, 8); got != c.Want {
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
		elements := invariant.Slice_Invariants(c.S)
		if len(elements) != 3 {
			t.Fatalf("%s: returned %d elements, want 3", c.Name, len(elements))
		}
		if got := event_signature(elements, 2); got != c.Want {
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
		elements := invariant.Map_Invariants(c.M)
		if len(elements) != 3 {
			t.Fatalf("%s: returned %d elements, want 3", c.Name, len(elements))
		}
		if got := event_signature(elements, 2); got != c.Want {
			t.Errorf("%s: [empty is_nil] = %q, want %q", c.Name, got, c.Want)
		}
	}
}
