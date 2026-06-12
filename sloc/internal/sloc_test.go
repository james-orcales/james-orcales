package sloc_test

import (
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	sloc "github.com/james-orcales/james-orcales/sloc/internal"
)

// Test_Main_Paths verifies the commandless command line: a directory given as a
// positional argument is counted, and no argument defaults to the current directory.
func Test_Main_Paths(t *testing.T) {
	disk := fstest.MapFS{
		"main.go": &fstest.MapFile{Data: []byte("package main\n")},
	}
	run := func(arguments []string) (output string) {
		buffer := strings.Builder{}
		stderr := strings.Builder{}
		code := sloc.Main(&sloc.Main_Input{
			Arguments:    arguments,
			Output:       &buffer,
			Error_Output: &stderr,
			Open:         func(root string) (file_system fs.FS) { return disk },
			Path_Is_Directory: func(name string) (is_directory bool, err error) {
				return true, nil
			},
			Read_File: func(name string) (content []byte, err error) {
				return nil, nil
			},
			Ignore_For: func(root string) (is_ignored sloc.Ignore_Predicate) {
				return nil
			},
			Concurrency: 1,
		})
		if code != 0 {
			t.Fatalf("%v: exit %d, stderr: %s", arguments, code, stderr.String())
		}
		return buffer.String()
	}

	if !strings.Contains(run([]string{"sloc", "src"}), "Go") {
		t.Error("expected Go counted for a positional path")
	}
	if !strings.Contains(run([]string{"sloc"}), "Go") {
		t.Error("expected Go counted for the default path")
	}
}
