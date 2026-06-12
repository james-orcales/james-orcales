//go:build ignore

// Command build runs the repo's go toolchain commands scoped to first-party
// code. Collapsing the workspace into one root module pulls two non-source
// trees into the module: third_party/ (vendored projects, including a
// deliberately-broken Go test corpus) and tmp/ (the TMPDIR build sandboxes set
// in .envrc). Both make a bare `go test ./...` fail, so the doctrine's
// shell-script and Makefile bans leave a Go program as the build entry point.
// Run via `go run build.go <test|vet|build> [extra go args]`.
package main

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
)

// Component_dirs_max caps the discovered top-level component directories. A
// handful exist today; the bound is a safety valve against a pathological tree,
// not a tuned size.
const component_dirs_max = 1024

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: go run build.go <test|vet|build> [extra go args]")
		os.Exit(2)
	}
	verb := os.Args[1]
	// The race detector and -count=1 mirror the pre-consolidation CI command;
	// vet and build take no such flags.
	flags := []string(nil)
	if verb == "test" {
		flags = []string{"-count=1", "-race"}
	}
	patterns, err := build_component_patterns()
	if err != nil {
		fmt.Fprintf(os.Stderr, "build: %v\n", err)
		os.Exit(1)
	}
	if len(patterns) == 0 {
		fmt.Fprintln(os.Stderr, "build: no first-party component packages found")
		os.Exit(1)
	}
	os.Exit(build_run_go(&build_run_go_input{
		Verb: verb, Flags: flags, Extra: os.Args[2:], Patterns: patterns,
	}))
}

type build_run_go_input struct {
	Verb     string
	Flags    []string
	Extra    []string
	Patterns []string
}

// Runs `go <verb> <flags> <extra> <patterns>` with stdio inherited and returns
// its exit code. Go demands flags precede package patterns, so any extra args
// the caller passes (e.g. -run) are slotted before the patterns too.
func build_run_go(input *build_run_go_input) (exit_code int) {
	args := []string{input.Verb}
	args = append(args, input.Flags...)
	args = append(args, input.Extra...)
	args = append(args, input.Patterns...)
	command := exec.Command("go", args...)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	command.Stdin = os.Stdin
	if run_err := command.Run(); run_err != nil {
		var exit_error *exec.ExitError
		if as_exit_error(run_err, &exit_error) {
			return exit_error.ExitCode()
		}
		fmt.Fprintf(os.Stderr, "build: go %s: %v\n", input.Verb, run_err)
		return 1
	}
	return 0
}

// errors.As without the import, kept tiny so build.go has no dependency beyond
// os/exec for the one error type it unwraps.
func as_exit_error(err error, target **exec.ExitError) (matched bool) {
	exit_error, ok := err.(*exec.ExitError)
	if !ok {
		return false
	}
	*target = exit_error
	return true
}

// Returns one `./<dir>/...` pattern per top-level directory that holds Go
// source, excluding the non-source trees a single root module would otherwise
// sweep in. Mirrors the linter's component notion: a component is a top-level
// directory of first-party Go.
func build_component_patterns() (patterns []string, err error) {
	// third_party and tmp carry Go that must never compile under this module;
	// vendor and dot-directories never hold first-party packages. home holds
	// XDG-rooted runtime state, not source.
	excluded := map[string]bool{
		"third_party": true, "tmp": true, "vendor": true, "home": true,
	}
	entries, read_err := os.ReadDir(".")
	if read_err != nil {
		return nil, read_err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if excluded[name] {
			continue
		}
		if name[0] == '.' {
			continue
		}
		has_go, walk_err := build_directory_has_go(name)
		if walk_err != nil {
			return nil, walk_err
		}
		if !has_go {
			continue
		}
		patterns = append(patterns, "./"+name+"/...")
		if len(patterns) > component_dirs_max {
			return nil, fmt.Errorf("more than %d component directories", component_dirs_max)
		}
	}
	return patterns, nil
}

var build_directory_has_go_found = fmt.Errorf("go file found")

// Reports whether directory holds a .go file at any depth. Stops at the first
// hit via a sentinel error so a deep component is not walked in full.
func build_directory_has_go(directory string) (found bool, err error) {
	walk_err := filepath.WalkDir(directory,
		func(path string, entry fs.DirEntry, entry_err error) (output error) {
			if entry_err != nil {
				return entry_err
			}
			if entry.IsDir() {
				return nil
			}
			if filepath.Ext(path) == ".go" {
				return build_directory_has_go_found
			}
			return nil
		})
	if walk_err == build_directory_has_go_found {
		return true, nil
	}
	if walk_err != nil {
		return false, walk_err
	}
	return false, nil
}
