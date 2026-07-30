package invariant_test

import (
	"strings"
	"testing"

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

func recovered_panic(action func()) (message string) {
	defer func() {
		if recovered := recover(); recovered != nil {
			message = recovered.(string)
		}
	}()
	action()
	return ""
}
