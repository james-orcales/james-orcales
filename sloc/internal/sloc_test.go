package sloc_test

import (
	"errors"
	"io"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	sloc "local/james-orcales/sloc/internal"
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
		code := sloc.Main(sloc.Main_Input{
			Arguments:    arguments,
			Output:       &buffer,
			Error_Output: &stderr,
			File_System:  func(root sloc.Root) (file_system fs.FS) { return disk },
			Path_Information: func(
				name sloc.File_Path,
			) (information fs.FileInfo, err error) {
				return test_information(true), nil
			},
			File: func(name sloc.File_Path) (file fs.File, err error) {
				return test_file(nil), nil
			},
			Command: func(
				name string, arguments []string,
			) (output []byte, err error) {
				return nil, errors.New("command unavailable")
			},
			Classifier: sloc.File_Classifier{
				Kind: sloc.FILE_CLASSIFIER_KIND_BYTES,
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

// Test_Main_Classifier keeps the explicit-file route on the same dependency boundary as a
// walked tree, otherwise simulations would silently regain the production scanner there.
func Test_Main_Classifier(t *testing.T) {
	source := sloc.Source("the real scanner would count one code line\n")
	want := sloc.File_Partition{Code: 19, Comment: 23, Blank: 29, Dropped: 31}
	output := strings.Builder{}
	error_output := strings.Builder{}
	code := sloc.Main(sloc.Main_Input{
		Arguments:    sloc.Arguments{"sloc", "modeled.go"},
		Output:       &output,
		Error_Output: &error_output,
		File_System: func(root sloc.Root) (file_system fs.FS) {
			return fstest.MapFS{}
		},
		Path_Information: func(
			name sloc.File_Path,
		) (information fs.FileInfo, err error) {
			return test_information(false), nil
		},
		File: func(name sloc.File_Path) (file fs.File, err error) {
			return test_file(source), nil
		},
		Command: func(
			name string, arguments []string,
		) (output []byte, err error) {
			return nil, errors.New("command unavailable")
		},
		Classifier: sloc.File_Classifier{
			Kind: sloc.FILE_CLASSIFIER_KIND_MODEL,
			Classifications: sloc.File_Classifications{
				"modeled.go": want,
			},
		},
		Concurrency: 1,
	})
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %q", code, error_output.String())
	}
	if !strings.Contains(output.String(), "19") {
		t.Errorf("output did not carry modeled code count:\n%s", output.String())
	}
}

// Test_Main_Classifier_Failures verifies an injected model cannot silently fall back
// to scanning and an unknown concrete implementation cannot enter the component.
func Test_Main_Classifier_Failures(t *testing.T) {
	cases := []struct {
		Name       string
		Classifier sloc.File_Classifier
		Source     sloc.Source
	}{
		{
			Name: "missing model",
			Classifier: sloc.File_Classifier{
				Kind: sloc.FILE_CLASSIFIER_KIND_MODEL,
			},
			Source: sloc.Source("nonempty\n"),
		},
		{
			Name: "unknown kind",
			Classifier: sloc.File_Classifier{
				Kind: sloc.File_Classifier_Kind(0),
			},
			Source: sloc.Source("nonempty\n"),
		},
	}
	for _, one := range cases {
		t.Run(one.Name, func(subtest *testing.T) {
			defer func() {
				if recovered := recover(); recovered == nil {
					subtest.Fatal(
						"classifier must reject an incomplete capability")
				}
			}()
			sloc.Main(classifier_main_input(one.Classifier, one.Source))
		})
	}
}

// Test_Main_Classifier_Empty_Model verifies the zero classification remains derivable
// without manufacturing a redundant modeled result for an empty source.
func Test_Main_Classifier_Empty_Model(t *testing.T) {
	classifier := sloc.File_Classifier{Kind: sloc.FILE_CLASSIFIER_KIND_MODEL}
	code := sloc.Main(classifier_main_input(classifier, sloc.Source{}))
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
}

// The explicit-file route keeps classifier panics on the caller goroutine, otherwise
// a worker panic could not be asserted without weakening production concurrency.
func classifier_main_input(
	classifier sloc.File_Classifier, source sloc.Source,
) (input sloc.Main_Input) {
	return sloc.Main_Input{
		Arguments:    sloc.Arguments{"sloc", "modeled.go"},
		Output:       io.Discard,
		Error_Output: io.Discard,
		File_System: func(root sloc.Root) (file_system fs.FS) {
			return fstest.MapFS{}
		},
		Path_Information: func(
			name sloc.File_Path,
		) (information fs.FileInfo, err error) {
			return test_information(false), nil
		},
		File: func(name sloc.File_Path) (file fs.File, err error) {
			return test_file(source), nil
		},
		Command: func(
			name string, arguments []string,
		) (output []byte, err error) {
			return nil, errors.New("command unavailable")
		},
		Classifier:  classifier,
		Concurrency: 1,
	}
}

// Returns stable file information without coupling a Main test to the host filesystem.
func test_information(directory bool) (information fs.FileInfo) {
	mode := fs.FileMode(0)
	if directory {
		mode = fs.ModeDir
	}
	disk := fstest.MapFS{"entry": &fstest.MapFile{Mode: mode}}
	information, information_err := fs.Stat(disk, "entry")
	if information_err != nil {
		panic(information_err)
	}
	return information
}

// Returns a fresh file handle because Main closes each injected handle after one read.
func test_file(source []byte) (file fs.File) {
	disk := fstest.MapFS{"file": &fstest.MapFile{Data: source}}
	file, open_err := disk.Open("file")
	if open_err != nil {
		panic(open_err)
	}
	return file
}
