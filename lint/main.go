// Usage: lint [dir]   (defaults to current directory).
package main

import (
	"fmt"
	"io/fs"
	"local/james-orcales/lint/internal/strings"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"local/james-orcales/lint/internal"
)

func main() {
	request := "."
	if len(os.Args) > 1 {
		request = os.Args[1]
	}
	// Re-root Fsys at the repo's single root go.mod so component discovery
	// always has the module anchor above every file. Without this,
	// running on a subpackage (e.g. ./shared/snap/v2) would walk
	// a Fsys with no go.mod in reach, and the doctrine checks would
	// degrade to per-file mode — flagging composition-tier packages that
	// should be exempt. Scope_Prefix narrows output back to what the user asked for.
	root, scope_prefix := main_resolve_root(request)
	start := time.Now()
	fsys := os.DirFS(root)
	// The lint.json config (policy plus the word-replacements table) is read by
	// lint.Main from this Fsys, not here: one config path shared with the tests.
	code := lint.Main(&lint.Main_Input{
		Fsys:           fsys,
		Stdout:         os.Stdout,
		Stderr:         os.Stderr,
		Root_Directory: root,
		Tracked:        main_load_tracked(root, fsys),
		Git:            lint.Load_Git(main_git_command(root)),
		CPU_Count:      runtime.NumCPU(),
		Readlink:       os.Readlink,
		Scope_Prefix:   scope_prefix,
	})
	main_print_rss_and_elapsed(start)
	os.Exit(code)
}

// Walks up from request to find the monorepo's single root go.mod. The
// directory containing it becomes the lint root; the relative path
// from there to the request is the scope filter passed to the linter
// so that output stays focused on what the user actually asked for.
// The root go.mod is required — this linter is built for the monorepo's
// single-module shape, and operating without that anchor would silently
// degrade the doctrine checks (component discovery, composition-tier
// exemption, cross-component boundary). Aborts with a clear error rather
// than running in a partially-blind mode.
func main_resolve_root(request string) (root string, scope_prefix string) {
	absolute_request, err := filepath.Abs(request)
	if err != nil {
		fmt.Fprintf(os.Stderr,
			"lint: The linter cannot resolve %q: %v\n", request, err)
		os.Exit(2)
	}
	current := absolute_request
	if information, status_err := os.Stat(current); status_err == nil {
		if !information.IsDir() {
			current = filepath.Dir(current)
		}
	}
	anchor := ""
	for step := 0; ; step++ {
		if _, status_err := os.Stat(filepath.Join(current, "go.mod")); status_err == nil {
			anchor = current
			break
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	if anchor == "" {
		fmt.Fprintf(os.Stderr,
			"lint: There is no go.mod above %q. "+
				"The linter expects the single root module of the monorepo.\n",
			request,
		)
		os.Exit(2)
	}
	relative, err := filepath.Rel(anchor, absolute_request)
	if err != nil {
		fmt.Fprintf(os.Stderr,
			"lint: The linter cannot compute the relative path: %v\n", err)
		os.Exit(2)
	}
	if relative == "." {
		return anchor, ""
	}
	return anchor, filepath.ToSlash(relative)
}

// Reports peak RSS and wall-clock elapsed since start. Lives in main.go
// because RSS/timing reads impure process state, which the library tier
// is forbidden from doing.
func main_print_rss_and_elapsed(start time.Time) {
	var ru syscall.Rusage
	peak_rss_bytes := int64(0)
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &ru); err == nil {
		peak_rss_bytes = int64(ru.Maxrss)
		// Darwin reports Maxrss in bytes; Linux reports in KiB. Normalize.
		if runtime.GOOS == "linux" {
			peak_rss_bytes *= 1024
		}
	}
	peak_rss_mb := lint.Format_Thousands(peak_rss_bytes / (1024 * 1024))
	elapsed_seconds := time.Since(start).Seconds()
	fmt.Fprintf(os.Stderr, "peak_rss=%s MiB elapsed_sec=%.3f\n", peak_rss_mb, elapsed_seconds)
}

// Binds lint.Git_Command to a real `git` in root. The library tier drives the
// sequence of subcommands; shelling out is the impurity that stays here.
func main_git_command(root string) (run lint.Git_Command) {
	return func(args ...string) (output string, ok bool) {
		return main_git_command_run(root, args...)
	}
}

func main_git_command_run(root string, args ...string) (output string, ok bool) {
	command := exec.Command("git", args...)
	command.Dir = root
	stdout, err := command.Output()
	if err != nil {
		return "", false
	}
	return strings.Trim_Space(string(stdout)), true
}

// Builds the tracked-file set the linter walks: git answers which paths are
// gitignored, the library tier walks the tree and prunes them. Returns nil on
// git failure so the linter falls back to walking the full tree.
func main_load_tracked(root string, fsys fs.FS) (tracked map[string]bool) {
	ignored, ok := main_git_ignored(root)
	if !ok {
		return nil
	}
	walked, walk_ok := lint.Tracked_Paths(fsys, ignored)
	if !walk_ok {
		return nil
	}
	return walked
}

// Returns the gitignored paths under root, wholly-ignored directories collapsed
// to a trailing slash (`.jj/`) so the walk can prune them whole. NUL-separated
// output (-z) survives paths with embedded whitespace.
func main_git_ignored(root string) (ignored map[string]bool, ok bool) {
	command := exec.Command(
		"git", "ls-files", "--others", "--ignored", "--exclude-standard",
		"--directory", "-z")
	command.Dir = root
	stdout, err := command.Output()
	if err != nil {
		fmt.Fprintf(os.Stderr,
			"lint: The git ignore-scan failed: %v. The linter reads the "+
				"full tree.\n", err)
		return nil, false
	}
	return lint.Parse_Ignored_Set(stdout), true
}
