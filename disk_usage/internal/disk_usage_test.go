package disk_usage_test

import (
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	disk_usage "local/james-orcales/disk_usage/internal"
)

// Runs Main over an in-memory tree and returns its stdout, stderr, and exit code.
func run_main(arguments []string, disk fstest.MapFS) (stdout string, stderr string, code int) {
	output := strings.Builder{}
	errors := strings.Builder{}
	code = disk_usage.Main(&disk_usage.Main_Input{
		Arguments:    arguments,
		Output:       &output,
		Error_Output: &errors,
		Open:         func(root string) (file_system fs.FS) { return disk },
		Disk_Usage:   logical_usage,
	})
	return output.String(), errors.String(), code
}

// Test_Main_Reports_Directory verifies the happy path: a directory argument is walked
// and its total surfaces in the output. -minimum=0 keeps the tiny test tree in view past
// the 1 MiB default.
func Test_Main_Reports_Directory(t *testing.T) {
	stdout, stderr, code := run_main(
		[]string{"disk_usage", "project", "-minimum=0"},
		tree(map[string]string{"a/file.txt": "12345"}),
	)
	if code != 0 {
		t.Fatalf("exit %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "project") {
		t.Errorf("root label missing from output:\n%s", stdout)
	}
}

// Test_Main_Depth_Flag verifies -depth caps the displayed depth.
func Test_Main_Depth_Flag(t *testing.T) {
	stdout, stderr, code := run_main(
		[]string{"disk_usage", "project", "-depth=1", "-minimum=0"},
		tree(map[string]string{"a/deep/file.txt": "12345"}),
	)
	if code != 0 {
		t.Fatalf("exit %d, stderr: %s", code, stderr)
	}
	if strings.Contains(stdout, "file.txt") {
		t.Errorf("depth=1 should hide the depth-3 file:\n%s", stdout)
	}
}

// Test_Main_Hides_Empty verifies a zero-byte entry is never printed, so an empty
// directory does not clutter the output.
func Test_Main_Hides_Empty(t *testing.T) {
	disk := tree(map[string]string{"full/file.txt": "12345"})
	disk["empty/.gitkeep"] = &fstest.MapFile{Data: nil}
	// -depth=-1 so the empty directory would be in range and -minimum=0 so size does not
	// hide it; only the zero-byte filter keeps it out, which is what this test pins down.
	stdout, stderr, code := run_main(
		[]string{"disk_usage", "project", "-depth=-1", "-minimum=0"}, disk)
	if code != 0 {
		t.Fatalf("exit %d, stderr: %s", code, stderr)
	}
	if strings.Contains(stdout, "empty") {
		t.Errorf("zero-byte directory should be hidden:\n%s", stdout)
	}
}

// Test_Main_Minimum verifies the -minimum flag: its 1 MiB default hides a sub-megabyte
// tree, a small threshold brings it back, and an unparseable value is a usage error.
func Test_Main_Minimum(t *testing.T) {
	disk := tree(map[string]string{"a/file.txt": "12345"})

	stdout, stderr, code := run_main([]string{"disk_usage", "project"}, disk)
	if code != 0 {
		t.Fatalf("default minimum: exit %d, stderr: %s", code, stderr)
	}
	if strings.Contains(stdout, "project") {
		t.Errorf("default 1 MiB minimum should hide a tiny tree:\n%s", stdout)
	}

	stdout, stderr, code = run_main([]string{"disk_usage", "project", "-minimum=1B"}, disk)
	if code != 0 {
		t.Fatalf("small minimum: exit %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "project") {
		t.Errorf("-minimum=1B should show the tree:\n%s", stdout)
	}

	_, stderr, code = run_main([]string{"disk_usage", "project", "-minimum=huge"}, disk)
	if code != 2 {
		t.Fatalf("unparseable minimum: exit = %d, want 2", code)
	}
	if stderr == "" {
		t.Errorf("expected a usage message for a bad minimum")
	}
}
