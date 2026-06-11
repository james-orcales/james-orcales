package setup_test

import (
	"errors"
	"io"
	"path/filepath"
	"slices"
	"testing"
	"testing/fstest"

	"github.com/james-orcales/james-orcales/setup/internal"
)

// Each test drives Plan and Main with in-memory filesystems and a recording or
// failing writer, asserting only on observable output — the planned writes, what
// the writer received, the exit code — never on internals, so the suite stays a
// black box over Plan and Main and never touches a real home directory.

// Test_Plan_Empty_Source_Yields_No_Writes verifies a source tree with no files
// produces nothing to sync.
func Test_Plan_Empty_Source_Yields_No_Writes(t *testing.T) {
	t.Parallel()
	writes := run_plan(t, &setup.Plan_Input{
		Source:                fstest.MapFS{},
		Destination:           fstest.MapFS{},
		Destination_Directory: test_home,
	})
	if len(writes) != 0 {
		t.Fatalf("expected no writes, got %d", len(writes))
	}
}

// Test_Plan_Missing_Destination_Is_Created verifies a source file absent from the
// home directory becomes a write of its bytes to the mirrored path.
func Test_Plan_Missing_Destination_Is_Created(t *testing.T) {
	t.Parallel()
	writes := run_plan(t, &setup.Plan_Input{
		Source: fstest.MapFS{
			".bashrc": &fstest.MapFile{Data: []byte("hello")},
		},
		Destination:           fstest.MapFS{},
		Destination_Directory: test_home,
	})
	by_path := writes_by_path(writes)
	want_path := filepath.Join(test_home, ".bashrc")
	if by_path[want_path] != "hello" {
		t.Fatalf("expected %q written with \"hello\", got %v", want_path, by_path)
	}
	if len(by_path) != 1 {
		t.Fatalf("expected exactly one write, got %d", len(by_path))
	}
}

// Test_Plan_Identical_Destination_Is_Skipped verifies a source file whose home
// copy already holds the same bytes produces no write, so sync is idempotent.
func Test_Plan_Identical_Destination_Is_Skipped(t *testing.T) {
	t.Parallel()
	writes := run_plan(t, &setup.Plan_Input{
		Source: fstest.MapFS{
			".gitconfig": &fstest.MapFile{Data: []byte("identical")},
		},
		Destination: fstest.MapFS{
			".gitconfig": &fstest.MapFile{Data: []byte("identical")},
		},
		Destination_Directory: test_home,
	})
	if len(writes) != 0 {
		t.Fatalf("expected no writes for identical contents, got %d", len(writes))
	}
}

// Test_Plan_Differing_Destination_Is_Overwritten verifies a source file whose
// home copy differs is rewritten with the source bytes.
func Test_Plan_Differing_Destination_Is_Overwritten(t *testing.T) {
	t.Parallel()
	writes := run_plan(t, &setup.Plan_Input{
		Source: fstest.MapFS{
			".gitconfig": &fstest.MapFile{Data: []byte("new")},
		},
		Destination: fstest.MapFS{
			".gitconfig": &fstest.MapFile{Data: []byte("old")},
		},
		Destination_Directory: test_home,
	})
	by_path := writes_by_path(writes)
	want_path := filepath.Join(test_home, ".gitconfig")
	if by_path[want_path] != "new" {
		t.Fatalf("expected %q overwritten with \"new\", got %v", want_path, by_path)
	}
}

// Test_Plan_Nested_Path_Mirrors_Under_Home verifies a deeply nested source file
// maps to the same relative path joined under the destination directory.
func Test_Plan_Nested_Path_Mirrors_Under_Home(t *testing.T) {
	t.Parallel()
	writes := run_plan(t, &setup.Plan_Input{
		Source: fstest.MapFS{
			".config/nvim/init.lua": &fstest.MapFile{Data: []byte("lua")},
		},
		Destination:           fstest.MapFS{},
		Destination_Directory: test_home,
	})
	by_path := writes_by_path(writes)
	want_path := filepath.Join(test_home, ".config/nvim/init.lua")
	if by_path[want_path] != "lua" {
		t.Fatalf("expected a nested write at %q, got %v", want_path, by_path)
	}
}

// Test_Main_Writes_Planned_Files verifies Main hands each planned file's contents
// to the injected writer at its mirrored path and reports success.
func Test_Main_Writes_Planned_Files(t *testing.T) {
	t.Parallel()
	written := map[string]string{}
	status := setup.Main(&setup.Main_Input{
		Source: fstest.MapFS{
			".bashrc": &fstest.MapFile{Data: []byte("hello")},
		},
		Destination:           fstest.MapFS{},
		Destination_Directory: test_home,
		Write_File: func(path string, contents []byte) (err error) {
			written[path] = string(contents)
			return nil
		},
		Stdout: io.Discard,
		Stderr: io.Discard,
	})
	if status != 0 {
		t.Fatalf("expected success, got status %d", status)
	}
	want_path := filepath.Join(test_home, ".bashrc")
	if written[want_path] != "hello" {
		t.Fatalf("expected %q written with \"hello\", got %v", want_path, written)
	}
}

// Test_Main_Skips_Identical_Destination verifies Main writes nothing when the
// home directory already matches the source, so a repeat run is a no-op.
func Test_Main_Skips_Identical_Destination(t *testing.T) {
	t.Parallel()
	write_count := 0
	status := setup.Main(&setup.Main_Input{
		Source: fstest.MapFS{
			".bashrc": &fstest.MapFile{Data: []byte("same")},
		},
		Destination: fstest.MapFS{
			".bashrc": &fstest.MapFile{Data: []byte("same")},
		},
		Destination_Directory: test_home,
		Write_File: func(path string, contents []byte) (err error) {
			write_count++
			return nil
		},
		Stdout: io.Discard,
		Stderr: io.Discard,
	})
	if status != 0 {
		t.Fatalf("expected success, got status %d", status)
	}
	if write_count != 0 {
		t.Fatalf("expected no writes for identical contents, got %d", write_count)
	}
}

// Test_Main_Reports_Write_Failure verifies a writer error makes Main report a
// non-zero exit code rather than swallowing the failure.
func Test_Main_Reports_Write_Failure(t *testing.T) {
	t.Parallel()
	status := setup.Main(&setup.Main_Input{
		Source: fstest.MapFS{
			".bashrc": &fstest.MapFile{Data: []byte("x")},
		},
		Destination:           fstest.MapFS{},
		Destination_Directory: test_home,
		Write_File: func(path string, contents []byte) (err error) {
			return errors.New("disk full")
		},
		Stdout: io.Discard,
		Stderr: io.Discard,
	})
	if status == 0 {
		t.Fatal("expected a non-zero status on write failure")
	}
}

// Test_Main_Applies_Macos_Defaults verifies that on darwin Main runs the macos
// defaults commands through the injected runner after the sync.
func Test_Main_Applies_Macos_Defaults(t *testing.T) {
	t.Parallel()
	ran := [][]string{}
	status := setup.Main(&setup.Main_Input{
		Source:                fstest.MapFS{},
		Destination:           fstest.MapFS{},
		Destination_Directory: test_home,
		Operating_System:      "darwin",
		Write_File: func(path string, contents []byte) (err error) {
			return nil
		},
		Run_Command: func(name string, arguments []string) (err error) {
			ran = append(ran, append([]string{name}, arguments...))
			return nil
		},
		Stdout: io.Discard,
		Stderr: io.Discard,
	})
	if status != 0 {
		t.Fatalf("expected success, got status %d", status)
	}
	if !ran_contains(ran, []string{"killall", "Dock"}) {
		t.Fatal("expected killall Dock to run")
	}
	clock := []string{
		"defaults", "write", "com.apple.menuextra.clock",
		"DateFormat", "-string", "EEE MMM d mm:HH",
	}
	if !ran_contains(ran, clock) {
		t.Fatal("expected the clock date format default to be set")
	}
}

// Test_Main_Skips_Macos_Defaults_Off_Darwin verifies no defaults commands run on
// any operating system other than darwin.
func Test_Main_Skips_Macos_Defaults_Off_Darwin(t *testing.T) {
	t.Parallel()
	run_count := 0
	status := setup.Main(&setup.Main_Input{
		Source:                fstest.MapFS{},
		Destination:           fstest.MapFS{},
		Destination_Directory: test_home,
		Operating_System:      "linux",
		Write_File: func(path string, contents []byte) (err error) {
			return nil
		},
		Run_Command: func(name string, arguments []string) (err error) {
			run_count++
			return nil
		},
		Stdout: io.Discard,
		Stderr: io.Discard,
	})
	if status != 0 {
		t.Fatalf("expected success, got status %d", status)
	}
	if run_count != 0 {
		t.Fatalf("expected no commands off darwin, got %d", run_count)
	}
}

// The fixed absolute home directory the planned writes are built against; a
// constant keeps the expected destination paths deterministic.
const test_home = "/home/user"

// Invokes Plan for input and fails the test on an unexpected error, so each case
// asserts on the writes without repeating the error plumbing.
func run_plan(t *testing.T, input *setup.Plan_Input) (writes []setup.File_Write) {
	t.Helper()
	result, plan_err := setup.Plan(input)
	if plan_err != nil {
		t.Fatalf("Plan returned an unexpected error: %v", plan_err)
	}
	return result
}

// Indexes writes by destination path, mapping each to its contents as a string
// so a test can assert on the writes regardless of their order.
func writes_by_path(writes []setup.File_Write) (by_path map[string]string) {
	by_path = make(map[string]string, len(writes))
	for _, write := range writes {
		by_path[write.Destination_Path] = string(write.Contents)
	}
	return by_path
}

// Reports whether ran holds a command exactly equal to want — name and arguments
// together — so a test can assert one specific invocation happened.
func ran_contains(ran [][]string, want []string) (found bool) {
	for _, command := range ran {
		if slices.Equal(command, want) {
			return true
		}
	}
	return false
}
