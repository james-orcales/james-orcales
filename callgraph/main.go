// Usage: callgraph [dir] [-filter=substring]   (defaults to current directory).
package main

import (
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"local/james-orcales/callgraph/internal"
)

func main() {
	os.Exit(callgraph.Main(&callgraph.Main_Input{
		Arguments: os.Args, Output: os.Stdout, Error_Output: os.Stderr,
		Load: load,
	}))
}

// Binds Load to bounded operating-system file operations and the Go toolchain.
func load() (packages []*callgraph.Package, module string) {
	directory, directory_err := os.Getwd()
	if directory_err != nil {
		return nil, ""
	}
	return callgraph.Load(&callgraph.Load_Input{
		Working_Directory: directory,
		Read_File:         read_file_bounded,
		Read_Directory:    read_directory_bounded,
		Walk_Directory:    filepath.WalkDir,
		Export_Data:       export_data,
		Open_Export: func(path string) (reader io.ReadCloser, err error) {
			return os.Open(path)
		},
	})
}

// Maps each dependency to the export file that the Go toolchain compiled.
func export_data(root string) (exports map[string]string) {
	exports = map[string]string{}
	// One invalid package must not hide valid export data for its dependencies.
	command := exec.Command(
		"go", "list", "-e", "-deps", "-export",
		"-f", "{{.ImportPath}}\t{{.Export}}", "./...")
	command.Dir = root
	output, list_err := command.Output()
	if list_err != nil {
		return exports
	}
	for _, line := range strings.Split(string(output), "\n") {
		path, export, split := strings.Cut(line, "\t")
		if split {
			if export != "" {
				exports[path] = export
			}
		}
	}
	return exports
}

// Reads at most limit_size bytes from path.
func read_file_bounded(path string, limit_size int64) (content []byte) {
	file, open_err := os.Open(path)
	if open_err != nil {
		return nil
	}
	defer file.Close()
	buffer := make([]byte, limit_size)
	count, _ := io.ReadFull(io.LimitReader(file, limit_size), buffer)
	return buffer[:count]
}

// Reads at most limit directory entries from directory.
func read_directory_bounded(directory string, limit int) (entries []fs.DirEntry) {
	handle, open_err := os.Open(directory)
	if open_err != nil {
		return nil
	}
	defer handle.Close()
	entries, _ = handle.ReadDir(limit)
	return entries
}
