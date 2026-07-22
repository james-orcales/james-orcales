package invariant_test

import (
	"strings"
	"testing"
	"testing/fstest"

	core "local/james-orcales/shared/invariant"
	invariant "local/james-orcales/shared/invariant/default"
)

// The package sugar is bound to the OS-wired Default, so observing what it records means
// swapping Default for a fresh recorder for the test's duration.
func swap_default(t *testing.T) (recorder *core.Recorder) {
	t.Helper()
	recorder = &core.Recorder{Is_Test: true}
	previous := invariant.Default
	invariant.Default = recorder
	t.Cleanup(func() { invariant.Default = previous })
	return recorder
}

func seed_sometimes(recorder *core.Recorder, identifier string, messages ...string) {
	for _, message := range messages {
		key := identifier + core.ELEMENT_MESSAGE_SEPARATOR + message
		recorder.Events.Store(key, &core.Assertion_Metadata{
			Kind:    core.ASSERTION_KIND_SOMETIMES,
			Message: key,
		})
	}
}

func recorded(
	t *testing.T, recorder *core.Recorder, identifier string, message string,
) (metadata *core.Assertion_Metadata) {
	t.Helper()
	key := identifier + core.ELEMENT_MESSAGE_SEPARATOR + message
	value, exists := recorder.Events.Load(key)
	if !exists {
		t.Fatalf("missing event %q", key)
	}
	return value.(*core.Assertion_Metadata)
}

// The sugar records through Default with the identifier-scoped key.
func Test_Sometimes_Forwards_To_Default(t *testing.T) {
	recorder := swap_default(t)
	seed_sometimes(recorder, "sugar", "observed")
	invariant.Sometimes("sugar", true, "observed")
	invariant.Sometimes("sugar", false, "observed")
	metadata := recorded(t, recorder, "sugar", "observed")
	if metadata.Frequency.Load() != 1 {
		t.Fatal("the true branch must be credited once")
	}
	if metadata.False_Frequency.Load() != 1 {
		t.Fatal("the false branch must be credited once")
	}
}

// The sugar enforces through Default, naming the message.
func Test_Always_Forwards_To_Default(t *testing.T) {
	swap_default(t)
	message := recovered_panic(func() { invariant.Always(false, "sugar holds") })
	if !strings.Contains(message, "sugar holds") {
		t.Fatalf("panic = %q, want the message", message)
	}
}

// The bounds guard fires through Default.
func Test_Range_Forwards_To_Default(t *testing.T) {
	swap_default(t)
	if recovered_panic(func() { invariant.Range("sugar", 5, 0, 3) }) == "" {
		t.Fatal("an out-of-range value must panic through the sugar")
	}
}

// The membership guard fires through Default.
func Test_Enum_Forwards_To_Default(t *testing.T) {
	swap_default(t)
	if recovered_panic(func() { invariant.Enum("sugar", 7, 1, 2) }) == "" {
		t.Fatal("a non-member must panic through the sugar")
	}
}

// The signed preset's bounds are width-exact per instantiation, not int64's.
func Test_Signed_Invariants_Witnesses_Boundaries(t *testing.T) {
	recorder := swap_default(t)
	seed_sometimes(recorder, "n",
		"The value is one.",
		"The value is negative one.",
		"The value is the type's minimum.",
		"The value is the type's maximum.",
	)
	invariant.Signed_Invariants("n", int8(-128))
	if recorded(t, recorder, "n", "The value is the type's minimum.").Frequency.Load() != 1 {
		t.Fatal("the int8 minimum must witness the minimum axis")
	}
	if recorded(t, recorder, "n", "The value is one.").False_Frequency.Load() != 1 {
		t.Fatal("a non-one value must witness the false branch")
	}
	invariant.Signed_Invariants("n", int8(127))
	if recorded(t, recorder, "n", "The value is the type's maximum.").Frequency.Load() != 1 {
		t.Fatal("the int8 maximum must witness the maximum axis")
	}
}

// The unsigned preset's maximum is the instantiated width's all-ones.
func Test_Unsigned_Invariants_Witnesses_Boundaries(t *testing.T) {
	recorder := swap_default(t)
	seed_sometimes(recorder, "n",
		"The value is zero.",
		"The value is one.",
		"The value is the type's maximum.",
	)
	invariant.Unsigned_Invariants("n", uint16(65535))
	if recorded(t, recorder, "n", "The value is the type's maximum.").Frequency.Load() != 1 {
		t.Fatal("the uint16 maximum must witness the maximum axis")
	}
	invariant.Unsigned_Invariants("n", uint16(0))
	if recorded(t, recorder, "n", "The value is zero.").Frequency.Load() != 1 {
		t.Fatal("zero must witness the zero axis")
	}
}

// The float preset witnesses the values ordinary arithmetic never produces.
func Test_Float_Invariants_Witnesses_Special_Values(t *testing.T) {
	recorder := swap_default(t)
	seed_sometimes(recorder, "f",
		"The value is NaN.",
		"The value is negative infinity.",
		"The value is positive infinity.",
	)
	not_a_number := float32(0)
	not_a_number /= not_a_number
	invariant.Float_Invariants("f", not_a_number)
	if recorded(t, recorder, "f", "The value is NaN.").Frequency.Load() != 1 {
		t.Fatal("NaN must witness the NaN axis")
	}
	infinity := recorded(t, recorder, "f", "The value is positive infinity.")
	if infinity.False_Frequency.Load() != 1 {
		t.Fatal("NaN must witness the false branch of the infinity axes")
	}
}

// One Sometimes carries a bool's whole truth table.
func Test_Boolean_Invariants_Witnesses_Both_Branches(t *testing.T) {
	recorder := swap_default(t)
	seed_sometimes(recorder, "b", "The value is true.")
	invariant.Boolean_Invariants("b", true)
	invariant.Boolean_Invariants("b", false)
	metadata := recorded(t, recorder, "b", "The value is true.")
	if metadata.Frequency.Load() != 1 {
		t.Fatal("true must credit the true branch")
	}
	if metadata.False_Frequency.Load() != 1 {
		t.Fatal("false must credit the false branch")
	}
}

// The presets' declared shape — generic, identifier-first, unqualified sugar calls — must
// register when rooted from a consumer package, or the sugar tier's bundles are decorative.
func Test_Preset_Shape_Registers(t *testing.T) {
	const SUGAR_SOURCE = `package sugar
func Boolean_Invariants[T ~bool](identifier string, b T) {
	Sometimes(identifier, b, "The value is true.")
}
`
	const CONSUMER_SOURCE = `package consumer
import "fixture/sugar"
func check(ok bool) { sugar.Boolean_Invariants("flag", ok) }
`
	output := &strings.Builder{}
	code := -1
	recorder := &core.Recorder{
		File_System: fstest.MapFS{
			"go.mod":            &fstest.MapFile{Data: []byte("module fixture\n")},
			"sugar/sugar.go":    &fstest.MapFile{Data: []byte(SUGAR_SOURCE)},
			"consumer/check.go": &fstest.MapFile{Data: []byte(CONSUMER_SOURCE)},
		},
		Packages_To_Analyze: []string{"/consumer"},
		Output:              output,
		Exit:                func(status int) { code = status },
		Is_Test:             true,
		Sugar_Package:       "fixture/sugar",
	}
	core.Recorder_Register_Packages_For_Analysis(recorder)
	if code != -1 {
		t.Fatalf("exit = %d, want no exit; output=%q", code, output.String())
	}
	key := "flag" + core.ELEMENT_MESSAGE_SEPARATOR + "The value is true."
	if _, exists := recorder.Events.Load(key); !exists {
		t.Fatalf("missing event %q", key)
	}
}

func recovered_panic(action func()) (message string) {
	defer func() {
		if recovered := recover(); recovered != nil {
			message = recovered.(string)
		}
	}()
	action()
	return ""
}
