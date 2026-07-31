// Package main binds the sloc command to the operating system.
package main

import (
	"io/fs"
	"os"
	"os/exec"
	"runtime"

	sloc "local/james-orcales/sloc/internal"
)

func main() {
	os.Exit(int(sloc.Main(sloc.Main_Input{
		Arguments:    os.Args,
		Output:       os.Stdout,
		Error_Output: os.Stderr,
		File_System: func(root sloc.Root) (file_system fs.FS) {
			return os.DirFS(string(root))
		},
		Path_Information: func(
			name sloc.File_Path,
		) (information fs.FileInfo, err error) {
			return os.Stat(string(name))
		},
		File: func(name sloc.File_Path) (file fs.File, err error) {
			return os.Open(string(name))
		},
		Command: func(
			name string, arguments []string,
		) (output []byte, err error) {
			return exec.Command(name, arguments...).Output()
		},
		Classifier: sloc.File_Classifier{
			Kind: sloc.FILE_CLASSIFIER_KIND_BYTES,
		},
		Concurrency: sloc.Concurrency(
			runtime.GOMAXPROCS(0) * sloc.WORKERS_PER_PROCESSOR),
	})))
}
