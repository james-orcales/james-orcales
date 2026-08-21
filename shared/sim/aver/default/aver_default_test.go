//go:build !invariant_disable_coverage && !prd && !prod && !production && !invariant_noop

package aver_test

import (
	"bytes"
	"runtime"
	"strings"
	"testing"
	"testing/fstest"

	core "local/james-orcales/shared/sim/aver"
	aver "local/james-orcales/shared/sim/aver/default"
)

// Compiled target must select same source when registration starts.
func Test_Default_Build_Target(t *testing.T) {
	test_default_build(t, runtime.GOOS+" && "+runtime.GOARCH+" && gc && go1.26")
	if len(aver.Default.Build_Context.BuildTags) != 0 {
		test_default_build(t, strings.Join(aver.Default.Build_Context.BuildTags, " && "))
	}
}

func test_default_build(t *testing.T, constraint string) {
	t.Helper()
	recorder := aver.Init_Default_Recorder()
	var output bytes.Buffer
	recorder.Output = &output
	recorder.Exit = func(int) {}
	recorder.Is_Test = true
	recorder.File_System = fstest.MapFS{
		"fixture/active.go": {Data: []byte("//go:build " + constraint +
			"\n\npackage fixture\n" +
			"func check(ok bool) { aver.Always(ok, \"active\") }\n")},
		"fixture/excluded.go": {Data: []byte("//go:build !(" + constraint +
			")\n\npackage fixture\n" +
			"func check() { aver.Always(true, \"excluded\") }\n")},
	}
	core.Recorder_Register_Packages_For_Analysis(recorder, "/fixture")
	_, exists := recorder.Events.Load("active")
	if !exists {
		t.Fatalf("active assertion missing: %q", output.String())
	}
	if output.Len() != 0 {
		t.Fatalf("output=%q", output.String())
	}
}

// Fixture_Subject stands in for a bundle subject where the test drives the builder directly.
type Fixture_Subject int

// Test_Default_Assertions_Uses_Injected_Recorder protects composition-tier dependency injection.
func Test_Default_Assertions_Uses_Injected_Recorder(t *testing.T) {
	recorder := &core.Recorder{}
	previous := aver.Default
	aver.Default = recorder
	defer func() { aver.Default = previous }()
	message := panic_text_default(func() {
		aver.Tree(Fixture_Subject(0), "range").Range_Int(3, 0, 2).Ensure()
	})
	if message == "" {
		t.Fatal("default builder did not enforce through the injected recorder")
	}
}

func panic_text_default(action func()) (message string) {
	defer func() {
		if recovered := recover(); recovered != nil {
			message = recovered.(string)
		}
	}()
	action()
	return ""
}
