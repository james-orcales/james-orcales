package invariant_test

import (
	"math"
	"testing"

	core "local/james-orcales/shared/invariant"
	"local/james-orcales/shared/invariant/default"
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
