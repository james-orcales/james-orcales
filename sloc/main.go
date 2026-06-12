// Package main is the sloc command: the thin composition root over the internal
// library tier, and the one place allowed to bind the real filesystem and shell out
// to git.
package main

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"

	sloc "github.com/james-orcales/james-orcales/sloc/internal"
)

// Caps a single explicitly-named file read, bounding memory on a pathologically large
// file.
const main_file_bytes_max = 64 << 20

func main() {
	os.Exit(sloc.Main(&sloc.Main_Input{
		Arguments:         os.Args,
		Output:            os.Stdout,
		Error_Output:      os.Stderr,
		Open:              func(root string) (file_system fs.FS) { return os.DirFS(root) },
		Path_Is_Directory: main_is_directory,
		Read_File:         main_read_file,
		Ignore_For:        main_git_ignore,
		Concurrency:       runtime.GOMAXPROCS(0),
	}))
}

// Reports whether a path names a directory.
func main_is_directory(name string) (is_directory bool, err error) {
	information, stat_err := os.Stat(name)
	if stat_err != nil {
		return false, stat_err
	}
	return information.IsDir(), nil
}

// Reads up to main_file_bytes_max bytes of a file — the bounded read the directory
// walk performs through the file system — capping memory on a huge file.
func main_read_file(name string) (content []byte, err error) {
	file, open_err := os.Open(name)
	if open_err != nil {
		return nil, open_err
	}
	defer file.Close()
	information, stat_err := file.Stat()
	if stat_err != nil {
		return nil, stat_err
	}
	byte_size := information.Size()
	if byte_size > main_file_bytes_max {
		byte_size = main_file_bytes_max
	}
	buffer := make([]byte, byte_size)
	_, read_err := io.ReadFull(io.LimitReader(file, byte_size), buffer)
	if main_read_failed(read_err) {
		return nil, read_err
	}
	return buffer, nil
}

// Reports whether a bounded read ended in a real error rather than a short or empty
// file.
func main_read_failed(read_err error) (failed bool) {
	if read_err == nil {
		return false
	}
	if errors.Is(read_err, io.EOF) {
		return false
	}
	return !errors.Is(read_err, io.ErrUnexpectedEOF)
}

// Builds an ignore predicate for a directory inside a git work tree: a path is
// ignored when git would not list it. One `git ls-files` lists every kept file under
// the root; a directory holding no kept file is pruned. When the root is not a git
// work tree, or git is unavailable, it returns nil so nothing is ignored.
func main_git_ignore(root string) (is_ignored sloc.Ignore_Predicate) {
	// --cached lists tracked files, --others untracked ones, --exclude-standard drops
	// the gitignored untracked files; together they are exactly the files git does not
	// ignore. -z keeps paths literal, and "." scopes the listing to this root.
	command := exec.Command(
		"git", "-C", root,
		"ls-files", "-z", "--cached", "--others", "--exclude-standard", "--", ".")
	output, run_err := command.Output()
	if run_err != nil {
		return nil
	}

	kept_files := map[string]bool{}
	kept_directories := map[string]bool{".": true}
	for _, name := range strings.Split(string(output), "\x00") {
		if name == "" {
			continue
		}
		kept_files[name] = true
		for parent := path.Dir(name); main_inside_root(parent); parent = path.Dir(parent) {
			kept_directories[parent] = true
		}
	}

	return func(relative_path string, is_directory bool) (ignored bool) {
		if is_directory {
			return !kept_directories[relative_path]
		}
		return !kept_files[relative_path]
	}
}

// Reports whether a path is a directory below the root, the stop condition for
// walking a file's ancestors.
func main_inside_root(parent string) (inside bool) {
	if parent == "." {
		return false
	}
	return parent != "/"
}
