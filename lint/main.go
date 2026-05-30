// Usage: lint [dir]   (defaults to current directory).
package main

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/james-orcales/james-orcales/lint/internal"
)

// Max_filesystem_path_chars caps any filesystem path or path fragment the
// resolver handles: POSIX PATH_MAX is 4096 on Linux. Mirrors the bound used
// by the internal library tier.
const max_filesystem_path_chars = 4096

// Min_non_empty anchors the Lo bucket of "string is non-empty" axes.
// Distinct_Boundary requires Lo < Hi, so a Lo of 1 captures the smallest
// observable non-empty length.
const min_non_empty = 1

// Max_git_output_chars caps the stdout of any git invocation we shell out
// to. Keeps memory bounded against pathological repositories without
// truncating realistic outputs.
const max_git_output_chars = 16777216

// Max_git_args caps the variadic args slice passed to a `git` subcommand
// invocation — long subcommand lines are bounded by a reasonable budget.
const max_git_args = 64

// Exit_code_hard_error is the os.Exit value used when the resolver hits a
// non-recoverable filesystem or git failure that the rest of the linter
// cannot proceed past.
const exit_code_hard_error = 2

// Max_tracked_paths caps the per-repository tracked-files set returned by
// git ls-files. Sized to bound memory against pathological monorepos.
const max_tracked_paths = 1048576

// Max_commit_list_chars caps the per-commit-list buffer accumulated from
// git log output; one commit per line, sized for git-log budget.
const max_commit_list = 1048576

// Tabs_per_thousand renders the comma separator for thousands-formatted
// numbers; the small string value is hoisted so the magic three-digit
// grouping value lives at the file top.
const thousands_group_size = 3

// Max_non_negative_int64 caps the input to the non-negative-int64 axis. The
// value is math.MaxInt64 / 2 to leave headroom for downstream arithmetic
// without overflow when callers add multipliers.
const max_non_negative_int64 = 4_611_686_018_427_387_904

func main() {
	request := "."
	if len(os.Args) > 1 {
		request = os.Args[1]
	}
	// Re-root Fsys at the workspace/module anchor so module discovery
	// always has a go.work or go.mod above every file. Without this,
	// running on a subpackage (e.g. ./golang_snacks/snap/v2) would walk
	// a Fsys with no go.mod in reach, and the doctrine checks would
	// degrade to per-file mode — flagging composition-tier packages that
	// should be exempt. Scope_Prefix narrows output back to what the user asked for.
	root, scope_prefix := main_resolve_workspace(request)
	start := time.Now()
	code := lint.Main(&lint.Main_Input{
		Fsys:           os.DirFS(root),
		Stdout:         os.Stdout,
		Stderr:         os.Stderr,
		Root_Directory: root,
		Tracked:        main_load_tracked(root),
		Git:            main_load_git(root),
		CPU_Count:      runtime.NumCPU(),
		Readlink:       os.Readlink,
		Stat: func(
			name string) (info fs.FileInfo, err error) {
			return os.Stat(name)
		},
		Scope_Prefix: scope_prefix,
	})
	main_print_rss_and_elapsed(start)
	os.Exit(code)
}

// Walks up from request to find the monorepo's go.work file. The
// directory containing it becomes the lint root; the relative path
// from there to the request is the scope filter passed to the linter
// so that output stays focused on what the user actually asked for.
// go.work is required — this linter is built for the monorepo's
// workspace shape, and operating without that anchor would silently
// degrade the doctrine checks (module discovery, composition-tier
// exemption, cross-module boundary). Aborts with a clear error rather
// than running in a partially-blind mode.
func main_resolve_workspace(request string) (root string, scope_prefix string) {
	defer func() {
	}()

	absolute_request, err := filepath.Abs(request)
	if err != nil {
		fmt.Fprintf(os.Stderr, "lint: cannot resolve %q: %v\n", request, err)
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
		if _, status_err := os.Stat(filepath.Join(current, "go.work")); status_err == nil {
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
			"lint: no go.work found above %q; this linter expects "+
				"a monorepo with a workspace file\n",
			request,
		)
		os.Exit(2)
	}
	relative, err := filepath.Rel(anchor, absolute_request)
	if err != nil {
		fmt.Fprintf(os.Stderr, "lint: cannot compute relative path: %v\n", err)
		os.Exit(2)
	}
	if relative == "." {
		return anchor, ""
	}
	return anchor, filepath.ToSlash(relative)
}

// Reports peak RSS and wall-clock elapsed since start. Lives in main.go
// because RSS/timing reads ambient process state, which the library tier
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
	peak_rss_mb := main_format_thousands(peak_rss_bytes / (1024 * 1024))
	elapsed_seconds := time.Since(start).Seconds()
	fmt.Fprintf(os.Stderr, "peak_rss=%s MiB elapsed_sec=%.3f\n", peak_rss_mb, elapsed_seconds)
}

// Formats a non-negative int64 with comma thousands separators.
// E.g., 1234567 → "1,234,567"; 42 → "42".
func main_format_thousands(n int64) (output string) {
	defer func() {
	}()

	digits := strconv.FormatInt(n, 10)
	digit_count := len(digits)
	if digit_count <= 3 {
		return digits
	}
	var b strings.Builder
	head := digit_count % 3
	if head > 0 {
		b.WriteString(digits[:head])
		b.WriteByte(',')
	}
	for i_index := head; i_index < digit_count; i_index += 3 {
		b.WriteString(digits[i_index : i_index+3])
		if i_index+3 < digit_count {
			b.WriteByte(',')
		}
	}
	return b.String()
}

// Runs `git <args>` with cmd.Dir = root and returns trimmed stdout. ok is
// false when git exits non-zero or isn't installed — callers degrade rather
// than abort, matching main_load_tracked's behavior on non-git trees.
func main_git(root string, args ...string) (output string, ok bool) {
	defer func() {
	}()
	command := exec.Command("git", args...)
	command.Dir = root
	stdout, err := command.Output()
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(string(stdout)), true
}

// Gathers the data the git-history tier needs. Skips entirely when HEAD is
// on main (nothing to check against itself), when the tree isn't a git repo,
// or when no main ref resolves locally. The Main_Ref_Missing signal lets the
// tier emit a specific failure on shallow CI checkouts rather than silently
// passing.
func main_load_git(root string) (input lint.Git_Input) {
	defer func() {
	}()
	head, ok := main_git(root, "rev-parse", "--abbrev-ref", "HEAD")
	if !ok {
		return lint.Git_Input{}
	}
	if head == "main" {
		return lint.Git_Input{}
	}
	// Detached HEAD at main's tip should also skip — happens when the
	// linter runs on a freshly-checked-out main without a tracking branch
	// (CI tag builds, `git checkout <sha>` for bisects).
	if head_sha, head_ok := main_git(root, "rev-parse", "HEAD"); head_ok {
		if main_sha, main_ok := main_git(root, "rev-parse", "main"); main_ok {
			if head_sha == main_sha {
				return lint.Git_Input{}
			}
		}
	}
	main_reference := main_load_git_find_main_reference(root)
	if main_reference == "" {
		return lint.Git_Input{Enabled: true, Main_Reference_Absent: true}
	}
	tip := main_load_git_resolve_pr_tip(root)
	return lint.Git_Input{
		Enabled: true,
		Merge_Commits: main_load_git_read_commits(&main_load_git_read_commits_input{
			Root: root, Flag: "--merges", Range: main_reference + ".." + tip,
		}),
		Non_Merge_Commits: main_load_git_read_commits(&main_load_git_read_commits_input{
			Root: root, Flag: "--no-merges", Range: main_reference + "..HEAD",
		}),
	}
}

func main_load_git_find_main_reference(root string) (reference string) {
	defer func() {
	}()

	for _, r := range []string{"origin/main", "main"} {
		if _, ok := main_git(root, "rev-parse", "--verify", "--quiet", r); ok {
			return r
		}
	}
	return ""
}

// Returns HEAD^2 when HEAD is a GitHub-style merge commit (two parents) so
// the merge-commits check inspects the PR's actual commits rather than the
// synthetic merge. Otherwise HEAD.
func main_load_git_resolve_pr_tip(root string) (tip string) {
	defer func() {
	}()

	output, ok := main_git(root, "rev-list", "--parents", "-n", "1", "HEAD")
	if !ok {
		return "HEAD"
	}
	if len(strings.Fields(output)) != 3 {
		return "HEAD"
	}
	return "HEAD^2"
}

type main_load_git_read_commits_input struct {
	Root  string
	Flag  string
	Range string
}

// --first-parent restricts the walk to the mainline: at each merge commit the
// traversal follows only the first parent, so commits brought in by a merge
// (PR branches, subtree imports) aren't enumerated — only the branch's own
// new commits. Without this, `git subtree add` floods the range with the
// imported repo's entire history.
func main_load_git_read_commits(
	input *main_load_git_read_commits_input) (output []lint.Git_Commit) {
	defer func() {
	}()
	stdout, ok := main_git(
		input.Root, "log", "--first-parent", input.Flag, "--format=%H|%s", input.Range)
	if !ok {
		return nil
	}
	for _, line := range strings.Split(stdout, "\n") {
		if line == "" {
			continue
		}
		pipe_offset := strings.IndexByte(line, '|')
		if pipe_offset < 0 {
			continue
		}
		output = append(
			output,
			lint.Git_Commit{Hash: line[:pipe_offset], Subject: line[pipe_offset+1:]})
	}
	return output
}

// Enumerates every path git considers in-scope: committed files plus
// untracked-but-not-ignored. The combined set is what "everything that
// isn't in .gitignore" actually means under git's rules. Returns nil on
// any failure (no .git, git missing, etc.) so the linter falls back to
// walking the full tree instead of silently linting nothing.
// NUL-separated output (-z) survives paths with embedded whitespace.
func main_load_tracked(root string) (output map[string]bool) {
	defer func() {
	}()
	command := exec.Command(
		"git", "ls-files", "--cached", "--others", "--exclude-standard", "-z")
	command.Dir = root
	stdout, err := command.Output()
	if err != nil {
		fmt.Fprintf(
			os.Stderr,
			"lint: git ls-files failed: %v; falling back to full-tree walk\n",
			err)
		return nil
	}
	output = make(map[string]bool)
	for _, f := range strings.Split(string(stdout), "\x00") {
		if f != "" {
			output[f] = true
		}
	}
	return output
}
