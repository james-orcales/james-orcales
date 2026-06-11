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
		if elements[index].Event == core.Event_Kind_True {
			signature += "T"
			continue
		}
		signature += "F"
	}
	return signature
}

// Asserts the trailing boundary of a Whole_Number_Invariants bundle (index 3, after
// the three Sometimes axes) is a Distinct_Boundary firing the expected endpoint event.
func assert_whole_number_boundary(
	t *testing.T, name string, elements []core.Dot_Element, want core.Event_Kind,
) {
	t.Helper()
	if len(elements) != 4 {
		t.Fatalf("%s: returned %d elements, want 4", name, len(elements))
	}
	boundary := elements[3]
	if boundary.Kind != core.Dot_Element_Kind_Boundary {
		t.Errorf("%s: Kind = %d, want Boundary", name, boundary.Kind)
	}
	if boundary.Event != want {
		t.Errorf("%s: Event = %d, want %d", name, boundary.Event, want)
	}
}

// Whole_Number_Invariants is a Distinct_Boundary over the type's own range, so the
// floor is the type's: a signed type bottoms out at signed_min (the value lands on the
// Lo endpoint there), an unsigned type at 0. The maximum is the Hi endpoint and any
// value between is interior — which is why zero is interior for a signed type but the
// Lo endpoint for an unsigned one.
func Test_Whole_Number_Invariants_Bounds(t *testing.T) {
	signed := []struct {
		Name string
		N    int
		Want core.Event_Kind
	}{
		{"signed minimum is the Lo endpoint", math.MinInt, core.Event_Kind_False},
		{"signed maximum is the Hi endpoint", math.MaxInt, core.Event_Kind_True},
		{"a middling signed value is interior", 5, core.Event_Kind_Interior},
		{"zero is interior for a signed type", 0, core.Event_Kind_Interior},
	}
	for _, c := range signed {
		assert_whole_number_boundary(
			t, c.Name, invariant.Whole_Number_Invariants(c.N), c.Want)
	}
	unsigned := []struct {
		Name string
		N    uint
		Want core.Event_Kind
	}{
		{"zero is the Lo endpoint for an unsigned type", 0, core.Event_Kind_False},
		{"unsigned maximum is the Hi endpoint", math.MaxUint, core.Event_Kind_True},
		{"a positive unsigned value is interior", 5, core.Event_Kind_Interior},
	}
	for _, c := range unsigned {
		assert_whole_number_boundary(
			t, c.Name, invariant.Whole_Number_Invariants(c.N), c.Want)
	}
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

// Test_Never_Has_Asserts_Absence verifies a Never_Has_X helper is an Always element
// (single bucket) whose event is the negation of the property: true when the string
// lacks it, false when it carries it.
func Test_Never_Has_Asserts_Absence(t *testing.T) {
	clean := invariant.Never_Has_Control("abc")
	if clean.Kind != core.Dot_Element_Kind_Always {
		t.Errorf("Never_Has_Control Kind = %d, want Always (%d)",
			clean.Kind, core.Dot_Element_Kind_Always)
	}
	if clean.Event != core.Event_Kind_True {
		t.Error(`Never_Has_Control("abc") must fire true — no control char`)
	}
	dirty := invariant.Never_Has_Control("a\x00b")
	if dirty.Event != core.Event_Kind_False {
		t.Error(`Never_Has_Control("a\x00b") must fire false — NUL is a control char`)
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
