package setup_test

import (
	"bytes"
	"errors"
	"io"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/james-orcales/james-orcales/setup/internal"
	"github.com/james-orcales/james-orcales/shared/sh"
)

// Each test drives Plan and Main with in-memory filesystems and a recording or
// failing writer, asserting only on observable output — the planned writes, what
// the writer received, the exit code — never on internals, so the suite stays a
// black box over Plan and Main and never touches a real home directory.

// Test_Order_Of_Operations verifies Bootstrap announces each step by name before
// it runs, executes them in order, and stops at the first failure — returning
// that step's status and never announcing the steps it skipped.
func Test_Order_Of_Operations(t *testing.T) {
	t.Parallel()
	order := []int{}
	log := &bytes.Buffer{}
	status := setup.Bootstrap(&setup.Bootstrap_Input{
		Stdout: log,
		Steps: []setup.Step{
			{Name: "first", Run: func() (status_code int) {
				order = append(order, 1)
				return 0
			}},
			{Name: "second", Run: func() (status_code int) {
				order = append(order, 2)
				return 1
			}},
			{Name: "third", Run: func() (status_code int) {
				order = append(order, 3)
				return 0
			}},
		},
	})
	if status != 1 {
		t.Fatalf("expected the failing step's status, got %d", status)
	}
	if !slices.Equal(order, []int{1, 2}) {
		t.Fatalf("expected steps to run in order and stop at the failure, ran %v", order)
	}
	if log.String() != "setup: first\nsetup: second\n" {
		t.Fatalf("expected each run step announced by name in order, got %q", log.String())
	}
}

// Test_Idempotency_Accepts_A_Matching_Version verifies Installed reports true when
// the binary's --version output starts with the wanted version.
func Test_Idempotency_Accepts_A_Matching_Version(t *testing.T) {
	t.Parallel()
	commands := []sh.Command{}
	yes := setup.Installed(&setup.Installed_Input{
		Shell: recording_shell(&commands, map[string]string{
			"/bin/tool": "tool 1.2.3 (abc 2026-01-01)\n",
		}, 0),
		Executable: "/bin/tool",
		Version:    "tool 1.2.3",
	})
	if !yes {
		t.Fatal("expected Installed true when the version prefix matches")
	}
}

// Test_Idempotency_Rejects_A_Missing_Or_Stale_Binary verifies Installed reports
// false when the binary is absent or reports a different version.
func Test_Idempotency_Rejects_A_Missing_Or_Stale_Binary(t *testing.T) {
	t.Parallel()
	commands := []sh.Command{}
	yes := setup.Installed(&setup.Installed_Input{
		Shell: recording_shell(&commands, map[string]string{
			"/bin/tool": "tool 9.9.9\n",
		}, 0),
		Executable: "/bin/tool",
		Version:    "tool 1.2.3",
	})
	if yes {
		t.Fatal("expected Installed false when the version differs")
	}
}

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

// Test_Plan_Ignored_Path_Is_Skipped verifies an ignored file is not synced and an
// ignored directory is pruned: the directory's file is reachable only if the walk
// descends into it, so marking just the directory proves the prune, not a per-file
// skip.
func Test_Plan_Ignored_Path_Is_Skipped(t *testing.T) {
	t.Parallel()
	writes := run_plan(t, &setup.Plan_Input{
		Source: fstest.MapFS{
			".bashrc":         &fstest.MapFile{Data: []byte("keep")},
			".netrc":          &fstest.MapFile{Data: []byte("secret")},
			".local/bin/tool": &fstest.MapFile{Data: []byte("built")},
		},
		Destination:           fstest.MapFS{},
		Destination_Directory: test_home,
		Is_Ignored: func(relative_path string) (ignored bool) {
			return slices.Contains([]string{".netrc", ".local"}, relative_path)
		},
	})
	by_path := writes_by_path(writes)
	if by_path[filepath.Join(test_home, ".bashrc")] != "keep" {
		t.Fatalf("expected .bashrc synced, got %v", by_path)
	}
	if _, planned := by_path[filepath.Join(test_home, ".netrc")]; planned {
		t.Fatal("expected the ignored file skipped")
	}
	if _, planned := by_path[filepath.Join(test_home, ".local/bin/tool")]; planned {
		t.Fatal("expected the ignored directory pruned")
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
// defaults commands through the injected runner after the sync, without printing
// a line per command.
func Test_Main_Applies_Macos_Defaults(t *testing.T) {
	t.Parallel()
	ran := [][]string{}
	log := &bytes.Buffer{}
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
		Stdout: log,
		Stderr: io.Discard,
	})
	if status != 0 {
		t.Fatalf("expected success, got status %d", status)
	}
	if !ran_contains(ran, []string{"killall", "Dock"}) {
		t.Fatal("expected killall Dock to run")
	}
	if log.String() != "" {
		t.Fatalf("expected no per-command logging, got %q", log.String())
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

// Test_Install_Neovim_Skips_Build_When_Installed_From_Checkout verifies that when
// nvim resolves to a path inside the checkout and reports the wanted release, no
// make runs.
func Test_Install_Neovim_Skips_Build_When_Installed_From_Checkout(t *testing.T) {
	t.Parallel()
	commands := []sh.Command{}
	executable := test_repository + "/home/.local/bin/nvim"
	status := setup.Install_Neovim(&setup.Install_Neovim_Input{
		Repository_Directory: test_repository,
		Shell: recording_shell(&commands, map[string]string{
			"which":    executable + "\n",
			executable: "NVIM v0.12.3\n",
		}, 0),
	})
	if status != 0 {
		t.Fatalf("expected success, got status %d", status)
	}
	if builds := make_commands(commands); len(builds) != 0 {
		t.Fatalf("expected no make when installed from the checkout, ran %v", builds)
	}
}

// Test_Install_Neovim_Builds_When_The_Match_Is_Outside_Repository verifies that an
// nvim resolving outside the checkout does not count, so the build runs and the
// foreign binary's version is never consulted.
func Test_Install_Neovim_Builds_When_The_Match_Is_Outside_Repository(t *testing.T) {
	t.Parallel()
	commands := []sh.Command{}
	status := setup.Install_Neovim(&setup.Install_Neovim_Input{
		Repository_Directory: test_repository,
		Shell: recording_shell(&commands, map[string]string{
			"which": "/opt/homebrew/bin/nvim\n",
		}, 0),
	})
	if status != 0 {
		t.Fatalf("expected success, got status %d", status)
	}
	if builds := make_commands(commands); len(builds) != 2 {
		t.Fatalf("expected the build to run, got %d make commands", len(builds))
	}
}

// Test_Install_Neovim_Configures_Prefix_Then_Installs verifies make runs twice
// against the vendored source — configure-and-build, then install — both
// carrying the install prefix, with only the second adding the install goal, and
// each phase announced in order before its make.
func Test_Install_Neovim_Configures_Prefix_Then_Installs(t *testing.T) {
	t.Parallel()
	commands := []sh.Command{}
	shell := recording_shell(&commands, nil, 0)
	progress := &bytes.Buffer{}
	shell.Stdout = progress
	status := setup.Install_Neovim(&setup.Install_Neovim_Input{
		Repository_Directory: test_repository,
		Shell:                shell,
	})
	if status != 0 {
		t.Fatalf("expected success, got status %d", status)
	}
	want_progress := "setup: neovim: configuring prefix...\nsetup: neovim: installing...\n"
	if progress.String() != want_progress {
		t.Fatalf("expected each build phase announced in order, got %q", progress.String())
	}
	builds := make_commands(commands)
	if len(builds) != 2 {
		t.Fatalf("expected two make commands, got %d: %v", len(builds), builds)
	}
	source := filepath.Join(test_repository, "third_party/neovim")
	prefix := "CMAKE_INSTALL_PREFIX=" + filepath.Join(test_repository, "home/.local")
	for index, command := range builds {
		for _, want := range []string{"-C", source, prefix} {
			if !slices.Contains(command.Arguments, want) {
				t.Fatalf("command %d missing %q", index, want)
			}
		}
	}
	if slices.Contains(builds[0].Arguments, "install") {
		t.Fatalf("first command should not install: %v", builds[0].Arguments)
	}
	if !slices.Contains(builds[1].Arguments, "install") {
		t.Fatalf("second command should install, got %v", builds[1].Arguments)
	}
}

// Test_Install_Neovim_Reports_Build_Failure verifies a failing make stops the
// bootstrap before installing and reports a non-zero exit code.
func Test_Install_Neovim_Reports_Build_Failure(t *testing.T) {
	t.Parallel()
	commands := []sh.Command{}
	status := setup.Install_Neovim(&setup.Install_Neovim_Input{
		Repository_Directory: test_repository,
		Shell:                recording_shell(&commands, nil, 1),
	})
	if status == 0 {
		t.Fatal("expected a non-zero status on build failure")
	}
	if builds := make_commands(commands); len(builds) != 1 {
		t.Fatalf("expected to stop after the first failing make, ran %d", len(builds))
	}
}

// Test_Install_Fonts_Skips_Without_A_Font_Directory verifies an empty font
// directory — an OS with no known user font location — does nothing.
func Test_Install_Fonts_Skips_Without_A_Font_Directory(t *testing.T) {
	t.Parallel()
	copies := 0
	status := setup.Install_Fonts(&setup.Install_Fonts_Input{
		Font_Directory: "",
		Font_Present:   func(file string) (present bool) { return false },
		Copy_Font: func(file string) (err error) {
			copies++
			return nil
		},
		Refresh: nil,
		Stdout:  io.Discard,
		Stderr:  io.Discard,
	})
	if status != 0 {
		t.Fatalf("expected success, got status %d", status)
	}
	if copies != 0 {
		t.Fatalf("expected no copies without a font directory, ran %d", copies)
	}
}

// Test_Install_Fonts_Copies_Only_Missing_Fonts verifies each absent face is
// copied while a present one is left alone, and each copy and skip is logged.
func Test_Install_Fonts_Copies_Only_Missing_Fonts(t *testing.T) {
	t.Parallel()
	copied := []string{}
	log := &bytes.Buffer{}
	status := setup.Install_Fonts(&setup.Install_Fonts_Input{
		Font_Directory: test_font_directory,
		Font_Present: func(file string) (present bool) {
			return file == "IosevkaNerdFontMono-Regular.ttf"
		},
		Copy_Font: func(file string) (err error) {
			copied = append(copied, file)
			return nil
		},
		Refresh: nil,
		Stdout:  log,
		Stderr:  io.Discard,
	})
	if status != 0 {
		t.Fatalf("expected success, got status %d", status)
	}
	if len(copied) != 3 {
		t.Fatalf("expected the three missing faces copied, got %v", copied)
	}
	if slices.Contains(copied, "IosevkaNerdFontMono-Regular.ttf") {
		t.Fatalf("expected the present face skipped, got %v", copied)
	}
	want_log := "setup: fonts: skipped IosevkaNerdFontMono-Regular.ttf (present)\n" +
		"setup: fonts: copied IosevkaNerdFontMono-Bold.ttf\n" +
		"setup: fonts: copied IosevkaNerdFontMono-Oblique.ttf\n" +
		"setup: fonts: copied IosevkaNerdFontMono-BoldOblique.ttf\n"
	if log.String() != want_log {
		t.Fatalf("expected per-face copy and skip logging, got %q", log.String())
	}
}

// Test_Install_Fonts_Skips_When_All_Present verifies that when every face is in
// the destination, nothing is copied and the cache is not refreshed.
func Test_Install_Fonts_Skips_When_All_Present(t *testing.T) {
	t.Parallel()
	copies := 0
	refreshed := false
	status := setup.Install_Fonts(&setup.Install_Fonts_Input{
		Font_Directory: test_font_directory,
		Font_Present:   func(file string) (present bool) { return true },
		Copy_Font: func(file string) (err error) {
			copies++
			return nil
		},
		Refresh: func() (err error) {
			refreshed = true
			return nil
		},
		Stdout: io.Discard,
		Stderr: io.Discard,
	})
	if status != 0 {
		t.Fatalf("expected success, got status %d", status)
	}
	if copies != 0 {
		t.Fatal("expected no copies when every face is present")
	}
	if refreshed {
		t.Fatal("expected no cache refresh when nothing was copied")
	}
}

// Test_Install_Fonts_Refreshes_Cache_After_Copies verifies a provided refresh
// runs once at least one face is copied.
func Test_Install_Fonts_Refreshes_Cache_After_Copies(t *testing.T) {
	t.Parallel()
	refreshed := false
	status := setup.Install_Fonts(&setup.Install_Fonts_Input{
		Font_Directory: test_font_directory,
		Font_Present:   func(file string) (present bool) { return false },
		Copy_Font:      func(file string) (err error) { return nil },
		Refresh: func() (err error) {
			refreshed = true
			return nil
		},
		Stdout: io.Discard,
		Stderr: io.Discard,
	})
	if status != 0 {
		t.Fatalf("expected success, got status %d", status)
	}
	if !refreshed {
		t.Fatal("expected the cache refresh after copying a face")
	}
}

// Test_Install_Fonts_Reports_A_Copy_Failure verifies a failed font copy reports a
// non-zero exit code.
func Test_Install_Fonts_Reports_A_Copy_Failure(t *testing.T) {
	t.Parallel()
	status := setup.Install_Fonts(&setup.Install_Fonts_Input{
		Font_Directory: test_font_directory,
		Font_Present:   func(file string) (present bool) { return false },
		Copy_Font:      func(file string) (err error) { return errors.New("disk full") },
		Refresh:        nil,
		Stdout:         io.Discard,
		Stderr:         io.Discard,
	})
	if status == 0 {
		t.Fatal("expected a non-zero status on copy failure")
	}
}

// Test_Install_Direnv_Skips_Build_When_Already_Built verifies that when the direnv
// binary in the bin directory already reports the wanted version, the build is
// skipped — proving the gate probes the built binary.
func Test_Install_Direnv_Skips_Build_When_Already_Built(t *testing.T) {
	t.Parallel()
	commands := []sh.Command{}
	status := setup.Install_Direnv(&setup.Install_Direnv_Input{
		Direnv_Directory: test_direnv_directory,
		Binary_Directory: test_link_directory,
		Shell: recording_shell(&commands, map[string]string{
			test_link_directory + "/direnv": "2.37.1\n",
		}, 0),
	})
	if status != 0 {
		t.Fatalf("expected success, got status %d", status)
	}
	if builds := commands_named(commands, "sh"); len(builds) != 0 {
		t.Fatalf("expected no build when already built, ran %v", builds)
	}
}

// Test_Install_Direnv_Builds_When_Absent verifies that when no direnv at the
// wanted version is present, the Go toolchain builds it.
func Test_Install_Direnv_Builds_When_Absent(t *testing.T) {
	t.Parallel()
	commands := []sh.Command{}
	status := setup.Install_Direnv(&setup.Install_Direnv_Input{
		Direnv_Directory: test_direnv_directory,
		Binary_Directory: test_link_directory,
		Shell:            recording_shell(&commands, nil, 0),
	})
	if status != 0 {
		t.Fatalf("expected success, got status %d", status)
	}
	if builds := commands_named(commands, "sh"); len(builds) != 1 {
		t.Fatalf("expected one build command, ran %d", len(builds))
	}
}

// Test_Install_Direnv_Reports_A_Build_Failure verifies a failing build reports a
// non-zero exit code.
func Test_Install_Direnv_Reports_A_Build_Failure(t *testing.T) {
	t.Parallel()
	commands := []sh.Command{}
	status := setup.Install_Direnv(&setup.Install_Direnv_Input{
		Direnv_Directory: test_direnv_directory,
		Binary_Directory: test_link_directory,
		Shell:            recording_shell(&commands, nil, 1),
	})
	if status == 0 {
		t.Fatal("expected a non-zero status on build failure")
	}
}

// Test_Install_Rust_Skips_Install_When_Already_Installed verifies that when rustc
// at CARGO_HOME reports the wanted version, the install is skipped and the
// toolchain is relinked — proving the gate probes the build path, not a symlink.
func Test_Install_Rust_Skips_Install_When_Already_Installed(t *testing.T) {
	t.Parallel()
	commands := []sh.Command{}
	status := setup.Install_Rust(&setup.Install_Rust_Input{
		Cargo_Directory: test_cargo_directory,
		Link_Directory:  test_link_directory,
		Shell: recording_shell(&commands, map[string]string{
			test_cargo_directory + "/bin/rustc": "rustc 1.96.0 (abc 2026-05-25)\n",
		}, 0),
	})
	if status != 0 {
		t.Fatalf("expected success, got status %d", status)
	}
	if installs := commands_named(commands, "sh"); len(installs) != 0 {
		t.Fatalf("expected no install when already installed, ran %v", installs)
	}
	if links := commands_named(commands, "ln"); len(links) != 3 {
		t.Fatalf("expected the existing toolchain relinked, ran %d", len(links))
	}
}

// Test_Install_Rust_Installs_Then_Links_When_Absent verifies that when the
// toolchain is not linked, rustup installs it and then the binaries are symlinked.
func Test_Install_Rust_Installs_Then_Links_When_Absent(t *testing.T) {
	t.Parallel()
	commands := []sh.Command{}
	status := setup.Install_Rust(&setup.Install_Rust_Input{
		Cargo_Directory: test_cargo_directory,
		Link_Directory:  test_link_directory,
		Shell:           recording_shell(&commands, nil, 0),
	})
	if status != 0 {
		t.Fatalf("expected success, got status %d", status)
	}
	if installs := commands_named(commands, "sh"); len(installs) != 1 {
		t.Fatalf("expected one install command, ran %d", len(installs))
	}
	if links := commands_named(commands, "ln"); len(links) != 3 {
		t.Fatalf("expected cargo, rustup, and rustc symlinked, ran %d", len(links))
	}
}

// Test_Install_Rust_Reports_An_Install_Failure verifies a failing install reports
// a non-zero exit code.
func Test_Install_Rust_Reports_An_Install_Failure(t *testing.T) {
	t.Parallel()
	commands := []sh.Command{}
	status := setup.Install_Rust(&setup.Install_Rust_Input{
		Cargo_Directory: test_cargo_directory,
		Link_Directory:  test_link_directory,
		Shell:           recording_shell(&commands, nil, 1),
	})
	if status == 0 {
		t.Fatal("expected a non-zero status on install failure")
	}
}

// Test_Install_Fish_Skips_Build_When_Already_Built verifies that when the fish
// binary under CARGO_HOME already reports the wanted version, the build is skipped
// and the binary is relinked — proving the gate probes the build path, not PATH,
// so a missing symlink does not force a recompile.
func Test_Install_Fish_Skips_Build_When_Already_Built(t *testing.T) {
	t.Parallel()
	commands := []sh.Command{}
	status := setup.Install_Fish(&setup.Install_Fish_Input{
		Fish_Directory:  test_fish_directory,
		Cargo_Directory: test_cargo_directory,
		Link_Directory:  test_link_directory,
		Shell: recording_shell(&commands, map[string]string{
			test_cargo_directory + "/bin/fish": "fish, version 4.7.1\n",
		}, 0),
	})
	if status != 0 {
		t.Fatalf("expected success, got status %d", status)
	}
	if builds := commands_named(commands, "sh"); len(builds) != 0 {
		t.Fatalf("expected no build when already built, ran %v", builds)
	}
	if links := commands_named(commands, "ln"); len(links) != 3 {
		t.Fatalf("expected the existing build relinked, ran %d", len(links))
	}
}

// Test_Install_Fish_Builds_Then_Links_When_Absent verifies that when no fish at
// the wanted version is present, cargo builds it and then the binary is linked.
func Test_Install_Fish_Builds_Then_Links_When_Absent(t *testing.T) {
	t.Parallel()
	commands := []sh.Command{}
	status := setup.Install_Fish(&setup.Install_Fish_Input{
		Fish_Directory:  test_fish_directory,
		Cargo_Directory: test_cargo_directory,
		Link_Directory:  test_link_directory,
		Shell:           recording_shell(&commands, nil, 0),
	})
	if status != 0 {
		t.Fatalf("expected success, got status %d", status)
	}
	if builds := commands_named(commands, "sh"); len(builds) != 1 {
		t.Fatalf("expected one build command, ran %d", len(builds))
	}
	if links := commands_named(commands, "ln"); len(links) != 3 {
		t.Fatalf("expected three binaries symlinked, ran %d", len(links))
	}
}

// Test_Install_Fish_Reports_A_Build_Failure verifies a failing build stops before
// linking and reports a non-zero exit code.
func Test_Install_Fish_Reports_A_Build_Failure(t *testing.T) {
	t.Parallel()
	commands := []sh.Command{}
	status := setup.Install_Fish(&setup.Install_Fish_Input{
		Fish_Directory:  test_fish_directory,
		Cargo_Directory: test_cargo_directory,
		Link_Directory:  test_link_directory,
		Shell:           recording_shell(&commands, nil, 1),
	})
	if status == 0 {
		t.Fatal("expected a non-zero status on build failure")
	}
	if links := commands_named(commands, "ln"); len(links) != 0 {
		t.Fatalf("expected no link after a failed build, ran %v", links)
	}
}

// Test_Install_Fzf_Skips_Build_When_Already_Built verifies that when the fzf
// binary in the bin directory already reports the wanted version, the build is
// skipped — proving the gate probes the built binary.
func Test_Install_Fzf_Skips_Build_When_Already_Built(t *testing.T) {
	t.Parallel()
	commands := []sh.Command{}
	status := setup.Install_Fzf(&setup.Install_Fzf_Input{
		Fzf_Directory:    test_fzf_directory,
		Binary_Directory: test_link_directory,
		Shell: recording_shell(&commands, map[string]string{
			test_link_directory + "/fzf": "0.73.1\n",
		}, 0),
	})
	if status != 0 {
		t.Fatalf("expected success, got status %d", status)
	}
	if builds := commands_named(commands, "sh"); len(builds) != 0 {
		t.Fatalf("expected no build when already built, ran %v", builds)
	}
}

// Test_Install_Fzf_Builds_When_Absent verifies that when no fzf at the wanted
// version is present, the Go toolchain builds it.
func Test_Install_Fzf_Builds_When_Absent(t *testing.T) {
	t.Parallel()
	commands := []sh.Command{}
	status := setup.Install_Fzf(&setup.Install_Fzf_Input{
		Fzf_Directory:    test_fzf_directory,
		Binary_Directory: test_link_directory,
		Shell:            recording_shell(&commands, nil, 0),
	})
	if status != 0 {
		t.Fatalf("expected success, got status %d", status)
	}
	if builds := commands_named(commands, "sh"); len(builds) != 1 {
		t.Fatalf("expected one build command, ran %d", len(builds))
	}
}

// Test_Install_Fzf_Reports_A_Build_Failure verifies a failing build reports a
// non-zero exit code.
func Test_Install_Fzf_Reports_A_Build_Failure(t *testing.T) {
	t.Parallel()
	commands := []sh.Command{}
	status := setup.Install_Fzf(&setup.Install_Fzf_Input{
		Fzf_Directory:    test_fzf_directory,
		Binary_Directory: test_link_directory,
		Shell:            recording_shell(&commands, nil, 1),
	})
	if status == 0 {
		t.Fatal("expected a non-zero status on build failure")
	}
}

// Test_Install_Command_Skips_Build_When_Already_On_Path verifies that when the
// command name already resolves on PATH, no build runs — proving the only
// idempotency gate for this repo's own tools is PATH presence, not a version.
func Test_Install_Command_Skips_Build_When_Already_On_Path(t *testing.T) {
	t.Parallel()
	commands := []sh.Command{}
	status := setup.Install_Command(&setup.Install_Command_Input{
		Package_Directory: test_command_directory,
		Binary_Directory:  test_link_directory,
		Binary_Name:       "maddox",
		Shell: recording_shell(&commands, map[string]string{
			"which": test_link_directory + "/maddox\n",
		}, 0),
	})
	if status != 0 {
		t.Fatalf("expected success, got status %d", status)
	}
	if builds := commands_named(commands, "sh"); len(builds) != 0 {
		t.Fatalf("expected no build when already on PATH, ran %v", builds)
	}
}

// Test_Install_Command_Builds_When_Absent verifies that when the command name does
// not resolve on PATH, the Go toolchain builds it into the bin directory.
func Test_Install_Command_Builds_When_Absent(t *testing.T) {
	t.Parallel()
	commands := []sh.Command{}
	status := setup.Install_Command(&setup.Install_Command_Input{
		Package_Directory: test_command_directory,
		Binary_Directory:  test_link_directory,
		Binary_Name:       "m2p",
		Shell:             recording_shell(&commands, nil, 0),
	})
	if status != 0 {
		t.Fatalf("expected success, got status %d", status)
	}
	if builds := commands_named(commands, "sh"); len(builds) != 1 {
		t.Fatalf("expected one build command, ran %d", len(builds))
	}
}

// Test_Install_Command_Reports_A_Build_Failure verifies a failing build reports a
// non-zero exit code.
func Test_Install_Command_Reports_A_Build_Failure(t *testing.T) {
	t.Parallel()
	commands := []sh.Command{}
	status := setup.Install_Command(&setup.Install_Command_Input{
		Package_Directory: test_command_directory,
		Binary_Directory:  test_link_directory,
		Binary_Name:       "sloc",
		Shell:             recording_shell(&commands, nil, 1),
	})
	if status == 0 {
		t.Fatal("expected a non-zero status on build failure")
	}
}

// Test_Install_Jj_Skips_Build_When_Already_Built verifies that when the jj binary
// under CARGO_HOME already reports the wanted version, the build is skipped and
// the binary is relinked — the gate prefix-matches the commit suffix jj appends.
func Test_Install_Jj_Skips_Build_When_Already_Built(t *testing.T) {
	t.Parallel()
	commands := []sh.Command{}
	status := setup.Install_Jj(&setup.Install_Jj_Input{
		Jj_Directory:    test_jj_directory,
		Cargo_Directory: test_cargo_directory,
		Link_Directory:  test_link_directory,
		Shell: recording_shell(&commands, map[string]string{
			test_cargo_directory + "/bin/jj": "jj 0.42.0-abc123\n",
		}, 0),
	})
	if status != 0 {
		t.Fatalf("expected success, got status %d", status)
	}
	if builds := commands_named(commands, "sh"); len(builds) != 0 {
		t.Fatalf("expected no build when already built, ran %v", builds)
	}
	if links := commands_named(commands, "ln"); len(links) != 1 {
		t.Fatalf("expected the existing build relinked once, ran %d", len(links))
	}
}

// Test_Install_Jj_Builds_Then_Links_When_Absent verifies that when no jj at the
// wanted version is present, cargo builds it and then the binary is linked.
func Test_Install_Jj_Builds_Then_Links_When_Absent(t *testing.T) {
	t.Parallel()
	commands := []sh.Command{}
	status := setup.Install_Jj(&setup.Install_Jj_Input{
		Jj_Directory:    test_jj_directory,
		Cargo_Directory: test_cargo_directory,
		Link_Directory:  test_link_directory,
		Shell:           recording_shell(&commands, nil, 0),
	})
	if status != 0 {
		t.Fatalf("expected success, got status %d", status)
	}
	if builds := commands_named(commands, "sh"); len(builds) != 1 {
		t.Fatalf("expected one build command, ran %d", len(builds))
	}
	if links := commands_named(commands, "ln"); len(links) != 1 {
		t.Fatalf("expected jj symlinked once, ran %d", len(links))
	}
}

// Test_Install_Jj_Reports_A_Build_Failure verifies a failing build stops before
// linking and reports a non-zero exit code.
func Test_Install_Jj_Reports_A_Build_Failure(t *testing.T) {
	t.Parallel()
	commands := []sh.Command{}
	status := setup.Install_Jj(&setup.Install_Jj_Input{
		Jj_Directory:    test_jj_directory,
		Cargo_Directory: test_cargo_directory,
		Link_Directory:  test_link_directory,
		Shell:           recording_shell(&commands, nil, 1),
	})
	if status == 0 {
		t.Fatal("expected a non-zero status on build failure")
	}
	if links := commands_named(commands, "ln"); len(links) != 0 {
		t.Fatalf("expected no link after a failed build, ran %v", links)
	}
}

// Test_Install_Ripgrep_Skips_Build_When_Already_Built verifies that when the rg
// binary under CARGO_HOME already reports the wanted version, the build is skipped
// and the binary is relinked — the gate prefix-matches the rev suffix rg appends.
func Test_Install_Ripgrep_Skips_Build_When_Already_Built(t *testing.T) {
	t.Parallel()
	commands := []sh.Command{}
	status := setup.Install_Ripgrep(&setup.Install_Ripgrep_Input{
		Ripgrep_Directory: test_ripgrep_directory,
		Cargo_Directory:   test_cargo_directory,
		Link_Directory:    test_link_directory,
		Shell: recording_shell(&commands, map[string]string{
			test_cargo_directory + "/bin/rg": "ripgrep 15.1.0 (rev abc123)\n",
		}, 0),
	})
	if status != 0 {
		t.Fatalf("expected success, got status %d", status)
	}
	if builds := commands_named(commands, "sh"); len(builds) != 0 {
		t.Fatalf("expected no build when already built, ran %v", builds)
	}
	if links := commands_named(commands, "ln"); len(links) != 1 {
		t.Fatalf("expected the existing build relinked once, ran %d", len(links))
	}
}

// Test_Install_Ripgrep_Builds_Then_Links_When_Absent verifies that when no rg at
// the wanted version is present, cargo builds it and then the binary is linked.
func Test_Install_Ripgrep_Builds_Then_Links_When_Absent(t *testing.T) {
	t.Parallel()
	commands := []sh.Command{}
	status := setup.Install_Ripgrep(&setup.Install_Ripgrep_Input{
		Ripgrep_Directory: test_ripgrep_directory,
		Cargo_Directory:   test_cargo_directory,
		Link_Directory:    test_link_directory,
		Shell:             recording_shell(&commands, nil, 0),
	})
	if status != 0 {
		t.Fatalf("expected success, got status %d", status)
	}
	if builds := commands_named(commands, "sh"); len(builds) != 1 {
		t.Fatalf("expected one build command, ran %d", len(builds))
	}
	if links := commands_named(commands, "ln"); len(links) != 1 {
		t.Fatalf("expected rg symlinked once, ran %d", len(links))
	}
}

// Test_Install_Ripgrep_Reports_A_Build_Failure verifies a failing build stops
// before linking and reports a non-zero exit code.
func Test_Install_Ripgrep_Reports_A_Build_Failure(t *testing.T) {
	t.Parallel()
	commands := []sh.Command{}
	status := setup.Install_Ripgrep(&setup.Install_Ripgrep_Input{
		Ripgrep_Directory: test_ripgrep_directory,
		Cargo_Directory:   test_cargo_directory,
		Link_Directory:    test_link_directory,
		Shell:             recording_shell(&commands, nil, 1),
	})
	if status == 0 {
		t.Fatal("expected a non-zero status on build failure")
	}
	if links := commands_named(commands, "ln"); len(links) != 0 {
		t.Fatalf("expected no link after a failed build, ran %v", links)
	}
}

// Test_Install_Fdcli_Skips_Build_When_Already_Built verifies that when the fd binary
// under CARGO_HOME already reports the wanted version, the build is skipped and
// the binary is relinked rather than recompiled.
func Test_Install_Fdcli_Skips_Build_When_Already_Built(t *testing.T) {
	t.Parallel()
	commands := []sh.Command{}
	status := setup.Install_Fdcli(&setup.Install_Fdcli_Input{
		Fdcli_Directory: test_fdcli_directory,
		Cargo_Directory: test_cargo_directory,
		Link_Directory:  test_link_directory,
		Shell: recording_shell(&commands, map[string]string{
			test_cargo_directory + "/bin/fd": "fd 10.4.2\n",
		}, 0),
	})
	if status != 0 {
		t.Fatalf("expected success, got status %d", status)
	}
	if builds := commands_named(commands, "sh"); len(builds) != 0 {
		t.Fatalf("expected no build when already built, ran %v", builds)
	}
	if links := commands_named(commands, "ln"); len(links) != 1 {
		t.Fatalf("expected the existing build relinked once, ran %d", len(links))
	}
}

// Test_Install_Fdcli_Builds_Then_Links_When_Absent verifies that when no fd at the
// wanted version is present, cargo builds it and then the binary is linked.
func Test_Install_Fdcli_Builds_Then_Links_When_Absent(t *testing.T) {
	t.Parallel()
	commands := []sh.Command{}
	status := setup.Install_Fdcli(&setup.Install_Fdcli_Input{
		Fdcli_Directory: test_fdcli_directory,
		Cargo_Directory: test_cargo_directory,
		Link_Directory:  test_link_directory,
		Shell:           recording_shell(&commands, nil, 0),
	})
	if status != 0 {
		t.Fatalf("expected success, got status %d", status)
	}
	if builds := commands_named(commands, "sh"); len(builds) != 1 {
		t.Fatalf("expected one build command, ran %d", len(builds))
	}
	if links := commands_named(commands, "ln"); len(links) != 1 {
		t.Fatalf("expected fd symlinked once, ran %d", len(links))
	}
}

// Test_Install_Fdcli_Reports_A_Build_Failure verifies a failing build stops before
// linking and reports a non-zero exit code.
func Test_Install_Fdcli_Reports_A_Build_Failure(t *testing.T) {
	t.Parallel()
	commands := []sh.Command{}
	status := setup.Install_Fdcli(&setup.Install_Fdcli_Input{
		Fdcli_Directory: test_fdcli_directory,
		Cargo_Directory: test_cargo_directory,
		Link_Directory:  test_link_directory,
		Shell:           recording_shell(&commands, nil, 1),
	})
	if status == 0 {
		t.Fatal("expected a non-zero status on build failure")
	}
	if links := commands_named(commands, "ln"); len(links) != 0 {
		t.Fatalf("expected no link after a failed build, ran %v", links)
	}
}

// Test_Install_Ghostty_Skips_Install_When_Already_Installed verifies that when the
// installed Ghostty app already reports the wanted version, the DMG is not
// downloaded and the app's CLI is relinked — the gate probes the app, not PATH.
func Test_Install_Ghostty_Skips_Install_When_Already_Installed(t *testing.T) {
	t.Parallel()
	commands := []sh.Command{}
	binary := test_applications_directory + "/Ghostty.app/Contents/MacOS/ghostty"
	status := setup.Install_Ghostty(&setup.Install_Ghostty_Input{
		Applications_Directory: test_applications_directory,
		Link_Directory:         test_link_directory,
		Shell: recording_shell(&commands, map[string]string{
			binary: "Ghostty 1.3.1\n",
		}, 0),
	})
	if status != 0 {
		t.Fatalf("expected success, got status %d", status)
	}
	if installs := commands_named(commands, "sh"); len(installs) != 0 {
		t.Fatalf("expected no download when already installed, ran %v", installs)
	}
	if links := commands_named(commands, "ln"); len(links) != 1 {
		t.Fatalf("expected the installed app's CLI relinked once, ran %d", len(links))
	}
}

// Test_Install_Ghostty_Reinstalls_When_Signature_Is_Invalid verifies that an
// installed app at the wanted version whose code signature no longer verifies is not
// trusted: the DMG install runs again, replacing the tampered bundle.
func Test_Install_Ghostty_Reinstalls_When_Signature_Is_Invalid(t *testing.T) {
	t.Parallel()
	commands := []sh.Command{}
	binary := test_applications_directory + "/Ghostty.app/Contents/MacOS/ghostty"
	status := setup.Install_Ghostty(&setup.Install_Ghostty_Input{
		Applications_Directory: test_applications_directory,
		Link_Directory:         test_link_directory,
		Shell: recording_shell_exits(&commands, map[string]string{
			binary: "Ghostty 1.3.1\n",
		}, map[string]int{"codesign": 1}),
	})
	if status != 0 {
		t.Fatalf("expected success, got status %d", status)
	}
	if installs := commands_named(commands, "sh"); len(installs) != 1 {
		t.Fatalf("expected a reinstall on an invalid signature, ran %d", len(installs))
	}
	if links := commands_named(commands, "ln"); len(links) != 1 {
		t.Fatalf("expected the reinstalled app's CLI linked once, ran %d", len(links))
	}
}

// Test_Install_Ghostty_Pins_The_Signing_Team verifies the signature gate pins
// Ghostty's signing Team ID, so an app validly signed by another developer is
// rejected even though its own signature verifies.
func Test_Install_Ghostty_Pins_The_Signing_Team(t *testing.T) {
	t.Parallel()
	commands := []sh.Command{}
	binary := test_applications_directory + "/Ghostty.app/Contents/MacOS/ghostty"
	setup.Install_Ghostty(&setup.Install_Ghostty_Input{
		Applications_Directory: test_applications_directory,
		Link_Directory:         test_link_directory,
		Shell: recording_shell(&commands, map[string]string{
			binary: "Ghostty 1.3.1\n",
		}, 0),
	})
	verifications := commands_named(commands, "codesign")
	if len(verifications) != 1 {
		t.Fatalf("expected one codesign verification, ran %d", len(verifications))
	}
	arguments := verifications[0].Arguments
	if !pins_team(arguments, "24VZTF6M5V") {
		t.Fatalf("expected the signature check to pin the team id, got %v", arguments)
	}
}

// Test_Install_Ghostty_Installs_Then_Links_When_Absent verifies that when no Ghostty
// at the wanted version is present, the DMG install runs and then the CLI is linked.
func Test_Install_Ghostty_Installs_Then_Links_When_Absent(t *testing.T) {
	t.Parallel()
	commands := []sh.Command{}
	status := setup.Install_Ghostty(&setup.Install_Ghostty_Input{
		Applications_Directory: test_applications_directory,
		Link_Directory:         test_link_directory,
		Shell:                  recording_shell(&commands, nil, 0),
	})
	if status != 0 {
		t.Fatalf("expected success, got status %d", status)
	}
	if installs := commands_named(commands, "sh"); len(installs) != 1 {
		t.Fatalf("expected one install command, ran %d", len(installs))
	}
	if links := commands_named(commands, "ln"); len(links) != 1 {
		t.Fatalf("expected ghostty symlinked once, ran %d", len(links))
	}
}

// Test_Install_Ghostty_Reports_An_Install_Failure verifies a failing install stops
// before linking and reports a non-zero exit code.
func Test_Install_Ghostty_Reports_An_Install_Failure(t *testing.T) {
	t.Parallel()
	commands := []sh.Command{}
	status := setup.Install_Ghostty(&setup.Install_Ghostty_Input{
		Applications_Directory: test_applications_directory,
		Link_Directory:         test_link_directory,
		Shell:                  recording_shell(&commands, nil, 1),
	})
	if status == 0 {
		t.Fatal("expected a non-zero status on install failure")
	}
	if links := commands_named(commands, "ln"); len(links) != 0 {
		t.Fatalf("expected no link after a failed install, ran %v", links)
	}
}

// The fixed absolute home directory the planned writes are built against; a
// constant keeps the expected destination paths deterministic.
const test_home = "/home/user"

// The fixed absolute checkout root the Neovim build subpaths are joined onto; a
// constant keeps the expected make and link paths deterministic.
const test_repository = "/repo"

// The fixed absolute font directory the Install_Fonts tests copy into; a constant
// keeps the expected destinations deterministic.
const test_font_directory = "/fonts"

// The fixed absolute CARGO_HOME the Install_Rust tests gate against; a constant
// keeps the expected toolchain paths deterministic.
const test_cargo_directory = "/cargo"

// The fixed absolute PATH directory the Install_Rust tests symlink into; a
// constant keeps the expected link paths deterministic.
const test_link_directory = "/link"

// The fixed absolute direnv source directory the Install_Direnv tests build from; a
// constant keeps the expected build paths deterministic.
const test_direnv_directory = "/direnv-src"

// The fixed absolute fish source directory the Install_Fish tests build from; a
// constant keeps the expected build paths deterministic.
const test_fish_directory = "/fish"

// The fixed absolute fzf source directory the Install_Fzf tests build from; a
// constant keeps the expected build paths deterministic.
const test_fzf_directory = "/fzf"

// The fixed absolute package directory the Install_Command tests build from; a
// constant keeps the expected build paths deterministic.
const test_command_directory = "/command-src"

// The fixed absolute jj source directory the Install_Jj tests build from; a
// constant keeps the expected build paths deterministic.
const test_jj_directory = "/jj"

// The fixed absolute ripgrep source directory the Install_Ripgrep tests build
// from; a constant keeps the expected build paths deterministic.
const test_ripgrep_directory = "/ripgrep"

// The fixed absolute fd source directory the Install_Fdcli tests build from; a
// constant keeps the expected build paths deterministic.
const test_fdcli_directory = "/fd-src"

// The fixed absolute macOS applications directory the Install_Ghostty tests gate
// against; a constant keeps the expected app and probe paths deterministic.
const test_applications_directory = "/apps"

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

// Returns a Shell whose Run records each command into record, replies to a probe
// with responses keyed by the command path, and reports exit for every spawn, so
// a test drives the gate and the build without spawning a real process.
func recording_shell(
	record *[]sh.Command, responses map[string]string, exit int,
) (shell *sh.Shell) {
	return &sh.Shell{
		Stdout:  io.Discard,
		Stderr:  io.Discard,
		Environ: func() (environment []string) { return nil },
		Run: func(command sh.Command) (outcome sh.Outcome) {
			*record = append(*record, command)
			io.WriteString(command.Stdout, responses[command.Path])
			return sh.Outcome{Exit: exit}
		},
	}
}

// Returns a Shell like recording_shell whose spawn exit code is chosen per command
// path from exits, defaulting to zero, so a test can fail one specific command — a
// codesign verify — while the gate's version probe and the rest still succeed.
func recording_shell_exits(
	record *[]sh.Command, responses map[string]string, exits map[string]int,
) (shell *sh.Shell) {
	return &sh.Shell{
		Stdout:  io.Discard,
		Stderr:  io.Discard,
		Environ: func() (environment []string) { return nil },
		Run: func(command sh.Command) (outcome sh.Outcome) {
			*record = append(*record, command)
			io.WriteString(command.Stdout, responses[command.Path])
			return sh.Outcome{Exit: exits[command.Path]}
		},
	}
}

// Reports whether a codesign invocation pins a signing team: it carries the -R
// requirement flag and a requirement argument naming the team id, so the check
// rejects an app signed by anyone else.
func pins_team(arguments []string, team string) (pins bool) {
	if !slices.Contains(arguments, "-R") {
		return false
	}
	for _, argument := range arguments {
		if strings.Contains(argument, team) {
			return true
		}
	}
	return false
}

// Returns only the make invocations from commands, dropping the nvim version
// probe so a test asserts on the build without counting the gate.
func make_commands(commands []sh.Command) (builds []sh.Command) {
	return commands_named(commands, "make")
}

// Returns the commands whose executable is name, so a test asserts on one kind of
// invocation — make, cp, fc-cache — without counting the rest.
func commands_named(commands []sh.Command, name string) (named []sh.Command) {
	for _, command := range commands {
		if command.Path == name {
			named = append(named, command)
		}
	}
	return named
}
