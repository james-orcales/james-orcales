package invariant_test

import (
	"strings"
	"testing"
	"testing/fstest"

	"local/james-orcales/shared/invariant"
)

// A bare eager Always is keyed by its message alone, seeded unreached.
func Test_Register_Eager_Always_Seeds_By_Message(t *testing.T) {
	recorder, _, code := registered_fixture(`package fixture
func check(a bool, b bool) {
	invariant.Always(a, "first holds")
	invariant.Always(b, "second holds")
}
`)
	if code != -1 {
		t.Fatalf("exit = %d, want no exit", code)
	}
	for _, message := range []string{"first holds", "second holds"} {
		metadata := recorder_event(t, recorder, message)
		if metadata.Kind != invariant.ASSERTION_KIND_ALWAYS {
			t.Fatalf("kind = %d, want Always", metadata.Kind)
		}
		if metadata.Frequency.Load() != 0 {
			t.Fatal("registration must seed unreached entries")
		}
	}
}

// Two Always sharing one message would merge into one entry and mask a gap.
func Test_Register_Fatal_On_Duplicate_Always_Message(t *testing.T) {
	_, output, code := registered_fixture(`package fixture
func check(a bool, b bool) {
	invariant.Always(a, "same")
	invariant.Always(b, "same")
}
`)
	if code != 1 {
		t.Fatalf("exit = %d, want 1; output=%q", code, output.String())
	}
	if !strings.Contains(output.String(), `duplicate message: "same"`) {
		t.Fatalf("output = %q, want the duplicate line", output.String())
	}
}

// A run whose every seeded entry was observed prints nothing and exits nowhere.
func Test_Analyze_Clean_Run_Is_Silent(t *testing.T) {
	recorder, output, _ := registered_fixture(`package fixture
func check(ok bool) { invariant.Always(ok, "reachable") }
`)
	invariant.Recorder_Always(recorder, true, "reachable")
	invariant.Recorder_Analyze_Assertion_Frequency(recorder)
	if output.Len() != 0 {
		t.Fatalf("clean output = %q, want empty", output.String())
	}
}

// An unreached Always is a reachability gap: reported and fatal.
func Test_Analyze_Reports_Never_Fired_And_Exits(t *testing.T) {
	recorder, output, _ := registered_fixture(`package fixture
func check(ok bool) { invariant.Always(ok, "reachable") }
`)
	analyze_code := -1
	recorder.Exit = func(status int) { analyze_code = status }
	invariant.Recorder_Analyze_Assertion_Frequency(recorder)
	if analyze_code != 1 {
		t.Fatalf("exit = %d, want 1", analyze_code)
	}
	if !strings.Contains(output.String(), "never reached") {
		t.Fatalf("gap report = %q, want never reached", output.String())
	}
}

// The success summary tallies the registered property space, labeled by package.
func Test_Assertion_Summary_Counts_Properties(t *testing.T) {
	recorder, _, _ := registered_fixture(`package fixture
func check(a bool, b bool) {
	invariant.Always(a, "first holds")
	invariant.Always(b, "second holds")
}
`)
	summary := invariant.Recorder_Assertion_Summary(recorder)
	want := "✓ fixture: tested 2 properties" +
		" (2 individual + 0 combinations, of which 2 are panic-able)"
	if summary != want {
		t.Fatalf("summary = %q, want %q", summary, want)
	}
}

// Directory globs bound the analyzed tree; `**` spans any depth while `*` spans one element.
func glob_fixture(pattern string) (recorder *invariant.Recorder) {
	const SHALLOW = `package fixture
func check(ok bool) { invariant.Always(ok, "shallow holds") }
`
	const DEEP = `package fixture
func check(ok bool) { invariant.Always(ok, "deep holds") }
`
	recorder = &invariant.Recorder{
		File_System: fstest.MapFS{
			"go.mod": &fstest.MapFile{
				Data: []byte("module fixture\n"),
			},
			"root/child/check.go": &fstest.MapFile{
				Data: []byte(SHALLOW),
			},
			"root/child/sub/check.go": &fstest.MapFile{
				Data: []byte(DEEP),
			},
		},
		Packages_To_Analyze: []string{pattern},
		Output:              &strings.Builder{},
		Exit:                func(int) {},
		Is_Test:             true,
	}
	invariant.Recorder_Register_Packages_For_Analysis(recorder)
	return recorder
}

// `**` bounds the analyzed tree at any depth.
func Test_Register_Double_Star_Seeds_Whole_Subtree(t *testing.T) {
	recorder := glob_fixture("/root/**")
	recorder_event(t, recorder, "shallow holds")
	recorder_event(t, recorder, "deep holds")
}

// `*` spans exactly one path element.
func Test_Register_Single_Star_Seeds_Only_Immediate_Children(t *testing.T) {
	recorder := glob_fixture("/root/*")
	recorder_event(t, recorder, "shallow holds")
	if _, exists := recorder.Events.Load("deep holds"); exists {
		t.Fatal("a single star must not descend past one path element")
	}
}

// A worker's persisted coverage line credits the coordinator's seeded entry.
func Test_Merge_Fuzz_Coverage_Credits_Seeded_Entries(t *testing.T) {
	recorder, _, _ := registered_fixture(`package fixture
func check(ok bool) { invariant.Always(ok, "reachable") }
`)
	line := invariant.Fuzz_Coverage_Line("reachable", true)
	invariant.Recorder_Merge_Fuzz_Coverage_From(recorder, strings.NewReader(line))
	if recorder_event(t, recorder, "reachable").Frequency.Load() != 1 {
		t.Fatal("a persisted key must credit its seeded entry")
	}
}
