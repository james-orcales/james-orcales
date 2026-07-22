package invariant_test

import (
	"fmt"
	"io"
	"math"
	"strings"
	"testing"
	"testing/fstest"

	core "local/james-orcales/shared/invariant"
	"local/james-orcales/shared/invariant/default"
)

// Seeds Default's tracker with the named axes under namespace, so a subsequent self-emitting
// preset call records into resolvable entries (the static scan seeds these in a real run).
func seed_preset_axes(namespace string, messages ...string) {
	var source strings.Builder
	source.WriteString("package fixture\nfunc check() { invariant.Dot_Product(")
	fmt.Fprintf(&source, "%q)", namespace)
	for _, message := range messages {
		fmt.Fprintf(&source, ".Sometimes(false, %q)", message)
	}
	rule := 0
	for first := range messages {
		for second := first + 1; second < len(messages); second++ {
			rule++
			fmt.Fprintf(&source, ".Impossible(%q, invariant.Event_True(%q), "+
				"invariant.Event_True(%q))",
				fmt.Sprintf("Boundary events are mutually exclusive (%d).", rule),
				messages[first], messages[second])
		}
	}
	source.WriteString(".Ensure() }\n")
	recorder := &core.Recorder{
		File_System: fstest.MapFS{
			"fixture/check.go": &fstest.MapFile{Data: []byte(source.String())},
		},
		Output:  io.Discard,
		Exit:    func(code int) { panic(fmt.Sprintf("registration exit %d", code)) },
		Is_Test: true,
	}
	core.Recorder_Register_Packages_For_Analysis(recorder, "/fixture")
	invariant.Default = recorder
	for _, message := range messages {
		key := namespace + core.ELEMENT_MESSAGE_SEPARATOR + message
		invariant.Default.Events.Store(key, &core.Assertion_Metadata{
			Kind: core.ASSERTION_KIND_SOMETIMES, Message: key,
		})
	}
}

// Renders, as a compact "T"/"F" signature, which of the named axes recorded a true event under
// namespace — read from Default's tracker after a single self-emitting preset call.
func recorded_signature(namespace string, messages ...string) (signature string) {
	for ordinal, message := range messages {
		key := namespace + core.ELEMENT_MESSAGE_SEPARATOR + fmt.Sprint(ordinal) +
			core.ELEMENT_MESSAGE_SEPARATOR + message
		value, loaded := invariant.Default.Events.Load(key)
		fired := false
		if loaded {
			fired = value.(*core.Assertion_Metadata).Frequency.Load() > 0
		}
		key = namespace + core.ELEMENT_MESSAGE_SEPARATOR + message
		value, loaded = invariant.Default.Events.Load(key)
		if loaded {
			fired = fired || value.(*core.Assertion_Metadata).Frequency.Load() > 0
		}
		if fired {
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

// Range_Invariants witnesses both interval edges and every sentinel strictly inside; the signed
// range [-100, 100] admits the minimum, the maximum, and all of 0/1/2/-1 as its interior.
func Test_Range_Invariants_Witnesses_Edges_And_Interior(t *testing.T) {
	axes := []string{
		core.RANGE_MESSAGE_MINIMUM,
		core.RANGE_MESSAGE_MAXIMUM,
		core.RANGE_MESSAGE_ZERO,
		core.RANGE_MESSAGE_ONE,
		core.RANGE_MESSAGE_TWO,
		core.RANGE_MESSAGE_NEGATIVE_ONE,
	}
	cases := []struct {
		Name string
		V    int
		Want string
	}{
		{"minimum", -100, "TFFFFF"},
		{"maximum", 100, "FTFFFF"},
		{"zero", 0, "FFTFFF"},
		{"one", 1, "FFFTFF"},
		{"two", 2, "FFFFTF"},
		{"negative one", -1, "FFFFFT"},
		{"ordinary", 42, "FFFFFF"},
	}
	for _, c := range cases {
		namespace := "test.range.signed." + c.Name
		seed_preset_axes(namespace, axes...)
		invariant.Range_Invariants(c.V, -100, 100, invariant.Namespace(namespace))
		if got := recorded_signature(namespace, axes...); got != c.Want {
			t.Errorf("%s: %q, want %q", c.Name, got, c.Want)
		}
	}
}

// A sentinel outside the open interval is not witnessed: [3, 100] admits its edges but none of
// 0/1/2/-1, so at its minimum only the minimum axis fires.
func Test_Range_Invariants_Excludes_Exterior_Sentinels(t *testing.T) {
	axes := []string{
		core.RANGE_MESSAGE_MINIMUM,
		core.RANGE_MESSAGE_ZERO,
		core.RANGE_MESSAGE_ONE,
		core.RANGE_MESSAGE_TWO,
		core.RANGE_MESSAGE_NEGATIVE_ONE,
	}
	namespace := "test.range.exterior"
	seed_preset_axes(namespace, axes...)
	invariant.Range_Invariants(3, 3, 100, invariant.Namespace(namespace))
	if got := recorded_signature(namespace, axes...); got != "TFFFF" {
		t.Errorf("exterior sentinels: %q, want %q", got, "TFFFF")
	}
}

// An unsigned value never witnesses negative one: -1 is below its zero floor, so the neg axis is
// dropped. The minimum edge covers zero, with 1 and 2 as its interior sentinels.
func Test_Range_Invariants_Unsigned_Drops_Negative_One(t *testing.T) {
	axes := []string{
		core.RANGE_MESSAGE_MINIMUM,
		core.RANGE_MESSAGE_ONE,
		core.RANGE_MESSAGE_NEGATIVE_ONE,
	}
	namespace := "test.range.unsigned"
	seed_preset_axes(namespace, axes...)
	invariant.Range_Invariants(uint(0), uint(0), uint(100), invariant.Namespace(namespace))
	if got := recorded_signature(namespace, axes...); got != "TFF" {
		t.Errorf("unsigned minimum: %q, want %q", got, "TFF")
	}
}

// The bounds are hard guards: a value outside [min,max] panics in every mode, like an Always.
func Test_Range_Invariants_Panics_Beyond_The_Interval(t *testing.T) {
	cases := []struct {
		Name     string
		V        int
		Min, Max int
	}{
		{"below min", 5, 10, 100},
		{"above max", 200, 10, 100},
	}
	for _, c := range cases {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("%s: expected panic, got none", c.Name)
				}
			}()
			invariant.Range_Invariants(c.V, c.Min, c.Max,
				invariant.Namespace("test.range.panic."+c.Name))
		}()
	}
}

// Each bound guard is a registered reachability obligation keyed by the callsite namespace — not a
// shared literal that would collide across every caller — so reaching the preset credits it.
func Test_Range_Invariants_Credits_Namespaced_Bound_Guards(t *testing.T) {
	namespace := "test.range.guards"
	for _, label := range []string{core.RANGE_GUARD_UPPER, core.RANGE_GUARD_LOWER} {
		key := namespace + core.ELEMENT_MESSAGE_SEPARATOR + label
		invariant.Default.Events.Store(key, &core.Assertion_Metadata{
			Kind: core.ASSERTION_KIND_ALWAYS, Message: key,
		})
	}
	invariant.Range_Invariants(5, 0, 100, invariant.Namespace(namespace))
	for _, label := range []string{core.RANGE_GUARD_UPPER, core.RANGE_GUARD_LOWER} {
		key := namespace + core.ELEMENT_MESSAGE_SEPARATOR + label
		value, loaded := invariant.Default.Events.Load(key)
		if !loaded {
			t.Errorf("bound guard %q was not seeded", label)
			continue
		}
		if value.(*core.Assertion_Metadata).Frequency.Load() == 0 {
			t.Errorf("bound guard %q was not credited", label)
		}
	}
}

// Boolean_Invariants returns one axis; the suite must witness the value true and false.
func Test_Boolean_Invariants_Tracks_Values(t *testing.T) {
	cases := []struct {
		Name string
		B    bool
		Want string
	}{
		{"true", true, "T"},
		{"false", false, "F"},
	}
	for _, c := range cases {
		namespace := "test.bool." + c.Name
		seed_preset_axes(namespace, "The value is true.")
		invariant.Boolean_Invariants(c.B, invariant.Namespace(namespace))
		if got := recorded_signature(namespace, "The value is true."); got != c.Want {
			t.Errorf("%s: [is_true] = %q, want %q", c.Name, got, c.Want)
		}
	}
}
