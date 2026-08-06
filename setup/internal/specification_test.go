package setup_test

import (
	"errors"
	"io"
	"path/filepath"
	"testing"

	"local/james-orcales/setup/internal"
	shared_bytes "local/james-orcales/shared/bytes"
	sysio "local/james-orcales/shared/io"
	"local/james-orcales/shared/jlog"
	shared_slices "local/james-orcales/shared/slices"
	shared_strings "local/james-orcales/shared/strings"
	systime "local/james-orcales/shared/time"
)

// Each test drives Plan and Mirror with in-memory filesystems and a recording or
// failing writer, asserting only on observable output — the planned writes, what
// the writer received, the exit code — never on internals, so the suite stays a
// black box over Plan and Mirror and never touches a real home directory.

// Test_Main_Runs_Complete_Bootstrap verifies Main constructs the ordered bootstrap and returns
// its first failing status instead of making the composition root construct the steps.
func Test_Main_Runs_Complete_Bootstrap(t *testing.T) {
	t.Parallel()
	commands := []sysio.Process_Request{}
	loop, _, clock := sysio.New_Sim(1)
	status := setup.Main(&setup.Main_Input{
		Environment: &setup.Environment_Input{
			Clock: clock, Console: io.Discard, Effective_User_Identifier: 1,
			Home_Directory: "/home/person", Operating_System: "freebsd",
		},
		IO: recording_io(directory_file_system(t, loop), &commands, nil, 1),
	})
	if status != 1 {
		t.Fatalf("status = %d, want the direnv failure status 1", status)
	}
	if len(commands) != 2 {
		t.Fatalf("commands = %v, want the direnv probe and build", commands)
	}
	want_probe := "/home/person/code/james-orcales/home/.local/bin/direnv"
	if commands[0].Path != want_probe {
		t.Fatalf("first command = %q, want %q", commands[0].Path, want_probe)
	}
	if commands[1].Path != "sh" {
		t.Fatalf("second command = %q, want direnv build through sh", commands[1].Path)
	}
}

// Test_Bootstrap_Steps verifies Bootstrap_Steps owns the complete bootstrap
// policy in the order that the setup command requires.
func Test_Bootstrap_Steps(t *testing.T) {
	t.Parallel()
	commands := []sysio.Process_Request{}
	loop, _, _ := sysio.New_Sim(1)
	steps := setup.Bootstrap_Steps(&setup.Bootstrap_Steps_Input{
		Home_Directory:   "/home/person",
		Operating_System: "freebsd",
		IO:               recording_io(directory_file_system(t, loop), &commands, nil, 0),
	})
	names := []string{}
	for _, step := range steps {
		names = append(names, string(step.Name))
	}
	want := []string{
		"direnv", "dotfiles", "fonts", "neovim", "fzf", "maddox", "m2p", "sloc",
		"timeout", "rust", "jj", "ripgrep", "fd", "ghostty",
	}
	if !shared_slices.Equal(names, want) {
		t.Fatalf("steps = %v, want %v", names, want)
	}
}

// Test_Order_Of_Operations verifies Bootstrap announces each step by name before
// it runs, executes them in order, and stops at the first failure — returning
// that step's status and never announcing the steps it skipped.
func Test_Order_Of_Operations(t *testing.T) {
	t.Parallel()
	order := []int{}
	log := shared_bytes.New_Buffer(nil)
	steps := setup.Steps{
		{Name: "first", Run: func() (succeeded setup.Step_Success) {
			order = append(order, 1)
			return true
		}},
		{Name: "second", Run: func() (succeeded setup.Step_Success) {
			order = append(order, 2)
			return false
		}},
		{Name: "third", Run: func() (succeeded setup.Step_Success) {
			order = append(order, 3)
			return true
		}},
	}
	for len(steps) < 14 {
		steps = append(steps, setup.Step{
			Name: "rest", Run: func() (succeeded setup.Step_Success) { return true },
		})
	}
	status := setup.Bootstrap(&setup.Bootstrap_Input{Logger: buffer_logger(log), Steps: steps})
	if status {
		t.Fatalf("expected the failing step's status, got %t", status)
	}
	if !shared_slices.Equal(order, []int{1, 2}) {
		t.Fatalf("expected steps to run in order and stop at the failure, ran %v", order)
	}
	want := "{\"level\":\"info\",\"name\":\"first\",\"message\":\"step\"}\n" +
		"{\"level\":\"info\",\"name\":\"second\",\"message\":\"step\"}\n"
	if buffer_text(log) != shared_strings.Text(want) {
		t.Fatalf(
			"expected each run step announced by name in order, got %q",
			buffer_text(log),
		)
	}
}

// Test_Idempotency_Accepts_A_Matching_Version verifies Installed reports true when
// the binary's --version output starts with the wanted version.
func Test_Idempotency_Accepts_A_Matching_Version(t *testing.T) {
	t.Parallel()
	commands := []sysio.Process_Request{}
	yes := setup.Installed(&setup.Installed_Input{
		IO: recording_process_io(&commands, map[string]string{
			"/bin/tool-x": "tool 1.2.3 (abc 2026-01-01)\n",
		}, 0),
		Executable: "/bin/tool-x",
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
	commands := []sysio.Process_Request{}
	yes := setup.Installed(&setup.Installed_Input{
		IO: recording_process_io(&commands, map[string]string{
			"/bin/tool-x": "tool 9.9.9\n",
		}, 0),
		Executable: "/bin/tool-x",
		Version:    "tool 1.2.3",
	})
	if yes {
		t.Fatal("expected Installed false when the version differs")
	}
}

// Test_Mirror_Applies_Macos_Defaults verifies that on darwin Mirror runs the macos
// defaults commands through the injected runner after the sync, without printing
// a line per command.
func Test_Mirror_Applies_Macos_Defaults(t *testing.T) {
	t.Parallel()
	commands := []sysio.Process_Request{}
	log := shared_bytes.New_Buffer(nil)
	loop, _, _ := sysio.New_Sim(0)
	if make_err := loop.Make_Directory(TEST_SOURCE); make_err != nil {
		t.Fatalf("make source: %v", make_err)
	}
	system := recording_io(directory_file_system(t, loop), &commands, nil, 0)
	status := setup.Mirror(&setup.Mirror_Input{
		IO:                    system,
		Source_Directory:      TEST_SOURCE,
		Destination_Directory: TEST_HOME,
		Operating_System:      "darwin",
		Logger:                buffer_logger(log),
	})
	if !status {
		t.Fatalf("expected success, got status %t", status)
	}
	if !request_contains(commands, []string{"killall", "Dock"}) {
		t.Fatal("expected killall Dock to run")
	}
	// The 35 defaults commands must not each print a line; the dotfiles scan narrates
	// itself, so assert the absence of per-command noise rather than an empty log.
	if shared_strings.Contains(buffer_text(log), "defaults") {
		t.Fatalf("expected no per-command logging, got %q", buffer_text(log))
	}
	clock := []string{
		"defaults", "write", "com.apple.menuextra.clock",
		"DateFormat", "-string", "EEE MMM d mm:HH",
	}
	if !request_contains(commands, clock) {
		t.Fatal("expected the clock date format default to be set")
	}
}

// Test_Mirror_Skips_Macos_Defaults_Off_Darwin verifies no defaults commands run on
// any operating system other than darwin.
func Test_Mirror_Skips_Macos_Defaults_Off_Darwin(t *testing.T) {
	t.Parallel()
	commands := []sysio.Process_Request{}
	loop, _, _ := sysio.New_Sim(0)
	if make_err := loop.Make_Directory(TEST_SOURCE); make_err != nil {
		t.Fatalf("make source: %v", make_err)
	}
	system := recording_io(directory_file_system(t, loop), &commands, nil, 0)
	status := setup.Mirror(&setup.Mirror_Input{
		IO:                    system,
		Source_Directory:      TEST_SOURCE,
		Destination_Directory: TEST_HOME,
		Operating_System:      "linux",
	})
	if !status {
		t.Fatalf("expected success, got status %t", status)
	}
	if len(commands) != 0 {
		t.Fatalf("expected no commands off darwin, got %d", len(commands))
	}
}

// Test_Mirror_Narrates_The_Scan verifies Mirror announces each source directory as the
// walk reads it, so a long silent scan of a large tree — one gitignore probe per
// entry — shows progress instead of looking hung, and reports an up-to-date tree
// when it writes nothing.
func Test_Mirror_Narrates_The_Scan(t *testing.T) {
	t.Parallel()
	loop, _, _ := sysio.New_Sim(0)
	// A nested directory, so the walk reads past the root and narrates more than one line.
	if make_err := loop.Make_Directory(TEST_SOURCE); make_err != nil {
		t.Fatalf("make source: %v", make_err)
	}
	if make_err := loop.Make_Directory(TEST_SOURCE + "/nested"); make_err != nil {
		t.Fatalf("make nested: %v", make_err)
	}
	log := shared_bytes.New_Buffer(nil)
	commands := []sysio.Process_Request{}
	system := recording_io(directory_file_system(t, loop), &commands, nil, 0)
	status := setup.Mirror(&setup.Mirror_Input{
		IO:                    system,
		Source_Directory:      TEST_SOURCE,
		Destination_Directory: TEST_HOME,
		Operating_System:      "linux",
		Logger:                buffer_logger(log),
	})
	if !status {
		t.Fatalf("expected success, got status %t", status)
	}
	for _, want := range []string{
		"{\"level\":\"debug\",\"dir\":\".\",\"message\":\"scanning\"}\n",
		"{\"level\":\"debug\",\"dir\":\"nested\",\"message\":\"scanning\"}\n",
		"{\"level\":\"info\",\"message\":\"dotfiles up to date\"}\n",
	} {
		if !shared_strings.Contains(buffer_text(log), shared_strings.Text(want)) {
			t.Fatalf("expected the scan to narrate %q, got %q", want, buffer_text(log))
		}
	}
}

// Test_Mirror_Probes_Each_Directory_In_One_Batch verifies the walk classifies a directory's
// entries with a single Is_Ignored call carrying them all, not one call per entry —
// the property that turns many sequential git spawns into one probe per directory.
func Test_Mirror_Probes_Each_Directory_In_One_Batch(t *testing.T) {
	t.Parallel()
	loop, _, _ := sysio.New_Sim(0)
	// Three sibling directories under the root, so a batched probe of the root sees
	// all three at once while a per-entry probe would see one at a time.
	for _, directory := range []string{
		TEST_SOURCE, TEST_SOURCE + "/a", TEST_SOURCE + "/b", TEST_SOURCE + "/c",
	} {
		if make_err := loop.Make_Directory(directory); make_err != nil {
			t.Fatalf("make %s: %v", directory, make_err)
		}
	}
	commands := []sysio.Process_Request{}
	system := recording_io(directory_file_system(t, loop), &commands, nil, 0)
	status := setup.Mirror(&setup.Mirror_Input{
		IO:                    system,
		Source_Directory:      TEST_SOURCE,
		Destination_Directory: TEST_HOME,
		Operating_System:      "linux",
	})
	if !status {
		t.Fatalf("expected success, got status %t", status)
	}
	probes := commands_named(commands, "git")
	if len(probes) == 0 {
		t.Fatal("expected the walk to probe ignores at least once")
	}
	if shared_strings.Count(shared_strings.Text(probes[0].Input), "\n") != 3 {
		t.Fatalf(
			"expected the root's three entries probed in one batch, got %q",
			probes[0].Input)
	}
}

// Test_Install_Neovim_Skips_Build_When_Installed_From_Checkout verifies that when
// nvim resolves to a path inside the checkout and reports the wanted release, no
// make runs.
func Test_Install_Neovim_Skips_Build_When_Installed_From_Checkout(t *testing.T) {
	t.Parallel()
	commands := []sysio.Process_Request{}
	executable := TEST_REPOSITORY + "/home/.local/bin/nvim"
	status := setup.Install_Neovim(&setup.Install_Neovim_Input{
		Repository_Directory: TEST_REPOSITORY,
		IO: recording_process_io(&commands, map[string]string{
			"which":    executable + "\n",
			executable: "NVIM v0.12.3\n",
		}, 0),
	})
	if !status {
		t.Fatalf("expected success, got status %t", status)
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
	commands := []sysio.Process_Request{}
	status := setup.Install_Neovim(&setup.Install_Neovim_Input{
		Repository_Directory: TEST_REPOSITORY,
		IO: recording_process_io(&commands, map[string]string{
			"which": "/opt/homebrew/bin/nvim\n",
		}, 0),
	})
	if !status {
		t.Fatalf("expected success, got status %t", status)
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
	commands := []sysio.Process_Request{}
	system := recording_process_io(&commands, nil, 0)
	progress := shared_bytes.New_Buffer(nil)
	status := setup.Install_Neovim(&setup.Install_Neovim_Input{
		Repository_Directory: TEST_REPOSITORY,
		IO:                   system,
		Logger:               buffer_logger(progress),
	})
	if !status {
		t.Fatalf("expected success, got status %t", status)
	}
	want_progress := "{\"level\":\"info\",\"message\":\"configuring neovim prefix\"}\n" +
		"{\"level\":\"info\",\"message\":\"installing neovim\"}\n"
	if buffer_text(progress) != shared_strings.Text(want_progress) {
		t.Fatalf(
			"expected each build phase announced in order, got %q",
			buffer_text(progress),
		)
	}
	builds := make_commands(commands)
	if len(builds) != 2 {
		t.Fatalf("expected two make commands, got %d: %v", len(builds), builds)
	}
	source := filepath.Join(TEST_REPOSITORY, "third_party/neovim")
	prefix := "CMAKE_INSTALL_PREFIX=" + filepath.Join(TEST_REPOSITORY, "home/.local")
	for index, command := range builds {
		for _, want := range []string{"-C", source, prefix} {
			if !shared_slices.Contains(command.Arguments, want) {
				t.Fatalf("command %d missing %q", index, want)
			}
		}
	}
	if shared_slices.Contains(builds[0].Arguments, "install") {
		t.Fatalf("first command should not install: %v", builds[0].Arguments)
	}
	if !shared_slices.Contains(builds[1].Arguments, "install") {
		t.Fatalf("second command should install, got %v", builds[1].Arguments)
	}
}

// Test_Install_Neovim_Reports_Build_Failure verifies a failing make stops the
// bootstrap before installing and reports a non-zero exit code.
func Test_Install_Neovim_Reports_Build_Failure(t *testing.T) {
	t.Parallel()
	commands := []sysio.Process_Request{}
	status := setup.Install_Neovim(&setup.Install_Neovim_Input{
		Repository_Directory: TEST_REPOSITORY,
		IO:                   recording_process_io(&commands, nil, 1),
	})
	if status {
		t.Fatal("expected a non-zero status on build failure")
	}
	if builds := make_commands(commands); len(builds) != 1 {
		t.Fatalf("expected to stop after the first failing make, ran %d", len(builds))
	}
}

// Test_Install_Fonts_Copies_Only_Missing_Fonts verifies each absent face is
// copied while a present one is left alone, and each copy and skip is logged.
func Test_Install_Fonts_Copies_Only_Missing_Fonts(t *testing.T) {
	t.Parallel()
	copied := []string{}
	log := shared_bytes.New_Buffer(nil)
	status := setup.Install_Fonts(&setup.Install_Fonts_Input{
		IO: font_test_io(func(file string) (present bool) {
			return file == "IosevkaNerdFontMono-Regular.ttf"
		}, &copied, nil, nil),
		Home_Directory: TEST_HOME,
		Font_Directory: TEST_FONT_DIRECTORY,
		Refresh_Cache:  false,
		Logger:         buffer_logger(log),
	})
	if !status {
		t.Fatalf("expected success, got status %t", status)
	}
	if len(copied) != 3 {
		t.Fatalf("expected the three missing faces copied, got %v", copied)
	}
	if shared_slices.Contains(copied, "IosevkaNerdFontMono-Regular.ttf") {
		t.Fatalf("expected the present face skipped, got %v", copied)
	}
	want_log := "{\"level\":\"info\",\"file\":\"IosevkaNerdFontMono-Regular.ttf\"," +
		"\"message\":\"font present, skipped\"}\n" +
		"{\"level\":\"info\",\"file\":\"IosevkaNerdFontMono-Bold.ttf\"," +
		"\"message\":\"copied font\"}\n" +
		"{\"level\":\"info\",\"file\":\"IosevkaNerdFontMono-Oblique.ttf\"," +
		"\"message\":\"copied font\"}\n" +
		"{\"level\":\"info\",\"file\":\"IosevkaNerdFontMono-BoldOblique.ttf\"," +
		"\"message\":\"copied font\"}\n"
	if buffer_text(log) != shared_strings.Text(want_log) {
		t.Fatalf("expected per-face copy and skip logging, got %q", buffer_text(log))
	}
}

// Test_Install_Fonts_Skips_When_All_Present verifies that when every face is in
// the destination, nothing is copied and the cache is not refreshed.
func Test_Install_Fonts_Skips_When_All_Present(t *testing.T) {
	t.Parallel()
	copied := []string{}
	commands := []sysio.Process_Request{}
	status := setup.Install_Fonts(&setup.Install_Fonts_Input{
		IO: font_test_io(
			func(string) (present bool) { return true }, &copied, nil, &commands),
		Home_Directory: TEST_HOME,
		Font_Directory: TEST_FONT_DIRECTORY, Refresh_Cache: true,
	})
	if !status {
		t.Fatalf("expected success, got status %t", status)
	}
	if len(copied) != 0 {
		t.Fatal("expected no copies when every face is present")
	}
	if len(commands) != 0 {
		t.Fatal("expected no cache refresh when nothing was copied")
	}
}

// Test_Install_Fonts_Refreshes_Cache_After_Copies verifies a provided refresh
// runs once at least one face is copied.
func Test_Install_Fonts_Refreshes_Cache_After_Copies(t *testing.T) {
	t.Parallel()
	copied := []string{}
	commands := []sysio.Process_Request{}
	status := setup.Install_Fonts(&setup.Install_Fonts_Input{
		IO: font_test_io(
			func(string) (present bool) { return false }, &copied, nil, &commands),
		Home_Directory: TEST_HOME,
		Font_Directory: TEST_FONT_DIRECTORY, Refresh_Cache: true,
	})
	if !status {
		t.Fatalf("expected success, got status %t", status)
	}
	if len(commands_named(commands, "fc-cache")) != 1 {
		t.Fatal("expected the cache refresh after copying a face")
	}
}

// Test_Install_Fonts_Reports_A_Copy_Failure verifies a failed font copy reports a
// non-zero exit code.
func Test_Install_Fonts_Reports_A_Copy_Failure(t *testing.T) {
	t.Parallel()
	copied := []string{}
	status := setup.Install_Fonts(&setup.Install_Fonts_Input{
		IO: font_test_io(
			func(string) (present bool) { return false },
			&copied, errors.New("disk full"), nil),
		Home_Directory: TEST_HOME,
		Font_Directory: TEST_FONT_DIRECTORY, Refresh_Cache: false,
	})
	if status {
		t.Fatal("expected a non-zero status on copy failure")
	}
}

// Test_Install_Direnv_Skips_Build_When_Already_Built verifies that when the direnv
// binary in the bin directory already reports the wanted version, the build is
// skipped — proving the gate probes the built binary.
func Test_Install_Direnv_Skips_Build_When_Already_Built(t *testing.T) {
	t.Parallel()
	commands := []sysio.Process_Request{}
	status := setup.Install_Direnv(&setup.Install_Direnv_Input{
		Direnv_Directory: TEST_DIRENV_DIRECTORY,
		Binary_Directory: TEST_LINK_DIRECTORY,
		IO: recording_process_io(&commands, map[string]string{
			TEST_LINK_DIRECTORY + "/direnv": "2.37.1\n",
		}, 0),
	})
	if !status {
		t.Fatalf("expected success, got status %t", status)
	}
	if builds := commands_named(commands, "sh"); len(builds) != 0 {
		t.Fatalf("expected no build when already built, ran %v", builds)
	}
}

// Test_Install_Direnv_Builds_When_Absent verifies that when no direnv at the
// wanted version is present, the Go toolchain builds it.
func Test_Install_Direnv_Builds_When_Absent(t *testing.T) {
	t.Parallel()
	commands := []sysio.Process_Request{}
	status := setup.Install_Direnv(&setup.Install_Direnv_Input{
		Direnv_Directory: TEST_DIRENV_DIRECTORY,
		Binary_Directory: TEST_LINK_DIRECTORY,
		IO:               recording_process_io(&commands, nil, 0),
	})
	if !status {
		t.Fatalf("expected success, got status %t", status)
	}
	if builds := commands_named(commands, "sh"); len(builds) != 1 {
		t.Fatalf("expected one build command, ran %d", len(builds))
	}
}

// Test_Install_Direnv_Reports_A_Build_Failure verifies a failing build reports a
// non-zero exit code.
func Test_Install_Direnv_Reports_A_Build_Failure(t *testing.T) {
	t.Parallel()
	commands := []sysio.Process_Request{}
	status := setup.Install_Direnv(&setup.Install_Direnv_Input{
		Direnv_Directory: TEST_DIRENV_DIRECTORY,
		Binary_Directory: TEST_LINK_DIRECTORY,
		IO:               recording_process_io(&commands, nil, 1),
	})
	if status {
		t.Fatal("expected a non-zero status on build failure")
	}
}

// Test_Install_Rust_Skips_Install_When_Already_Installed verifies that when rustc
// at CARGO_HOME reports the wanted version, the install is skipped and the
// toolchain is relinked — proving the gate probes the build path, not a symlink.
func Test_Install_Rust_Skips_Install_When_Already_Installed(t *testing.T) {
	t.Parallel()
	commands := []sysio.Process_Request{}
	status := setup.Install_Rust(&setup.Install_Rust_Input{
		Cargo_Directory: TEST_CARGO_DIRECTORY,
		Link_Directory:  TEST_LINK_DIRECTORY,
		IO: recording_process_io(&commands, map[string]string{
			TEST_CARGO_DIRECTORY + "/bin/rustc": "rustc 1.96.0 (abc 2026-05-25)\n",
		}, 0),
	})
	if !status {
		t.Fatalf("expected success, got status %t", status)
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
	commands := []sysio.Process_Request{}
	status := setup.Install_Rust(&setup.Install_Rust_Input{
		Cargo_Directory: TEST_CARGO_DIRECTORY,
		Link_Directory:  TEST_LINK_DIRECTORY,
		IO:              recording_process_io(&commands, nil, 0),
	})
	if !status {
		t.Fatalf("expected success, got status %t", status)
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
	commands := []sysio.Process_Request{}
	status := setup.Install_Rust(&setup.Install_Rust_Input{
		Cargo_Directory: TEST_CARGO_DIRECTORY,
		Link_Directory:  TEST_LINK_DIRECTORY,
		IO:              recording_process_io(&commands, nil, 1),
	})
	if status {
		t.Fatal("expected a non-zero status on install failure")
	}
}

// Test_Install_Fzf_Skips_Build_When_Already_Built verifies that when the fzf
// binary in the bin directory already reports the wanted version, the build is
// skipped — proving the gate probes the built binary.
func Test_Install_Fzf_Skips_Build_When_Already_Built(t *testing.T) {
	t.Parallel()
	commands := []sysio.Process_Request{}
	status := setup.Install_Fzf(&setup.Install_Fzf_Input{
		Fzf_Directory:    TEST_FZF_DIRECTORY,
		Binary_Directory: TEST_LINK_DIRECTORY,
		IO: recording_process_io(&commands, map[string]string{
			TEST_LINK_DIRECTORY + "/fzf": "0.73.1\n",
		}, 0),
	})
	if !status {
		t.Fatalf("expected success, got status %t", status)
	}
	if builds := commands_named(commands, "sh"); len(builds) != 0 {
		t.Fatalf("expected no build when already built, ran %v", builds)
	}
}

// Test_Install_Fzf_Builds_When_Absent verifies that when no fzf at the wanted
// version is present, the Go toolchain builds it.
func Test_Install_Fzf_Builds_When_Absent(t *testing.T) {
	t.Parallel()
	commands := []sysio.Process_Request{}
	status := setup.Install_Fzf(&setup.Install_Fzf_Input{
		Fzf_Directory:    TEST_FZF_DIRECTORY,
		Binary_Directory: TEST_LINK_DIRECTORY,
		IO:               recording_process_io(&commands, nil, 0),
	})
	if !status {
		t.Fatalf("expected success, got status %t", status)
	}
	if builds := commands_named(commands, "sh"); len(builds) != 1 {
		t.Fatalf("expected one build command, ran %d", len(builds))
	}
}

// Test_Install_Fzf_Reports_A_Build_Failure verifies a failing build reports a
// non-zero exit code.
func Test_Install_Fzf_Reports_A_Build_Failure(t *testing.T) {
	t.Parallel()
	commands := []sysio.Process_Request{}
	status := setup.Install_Fzf(&setup.Install_Fzf_Input{
		Fzf_Directory:    TEST_FZF_DIRECTORY,
		Binary_Directory: TEST_LINK_DIRECTORY,
		IO:               recording_process_io(&commands, nil, 1),
	})
	if status {
		t.Fatal("expected a non-zero status on build failure")
	}
}

// Test_Install_Command_Skips_Build_When_Already_On_Path verifies that when the
// command name already resolves on PATH, no build runs — proving the only
// idempotency gate for this repo's own tools is PATH presence, not a version.
func Test_Install_Command_Skips_Build_When_Already_On_Path(t *testing.T) {
	t.Parallel()
	commands := []sysio.Process_Request{}
	status := setup.Install_Command(&setup.Install_Command_Input{
		Package_Directory: TEST_COMMAND_DIRECTORY,
		Binary_Directory:  TEST_LINK_DIRECTORY,
		Binary_Name:       "maddox",
		IO: recording_process_io(&commands, map[string]string{
			"which": TEST_LINK_DIRECTORY + "/maddox\n",
		}, 0),
	})
	if !status {
		t.Fatalf("expected success, got status %t", status)
	}
	if builds := commands_named(commands, "sh"); len(builds) != 0 {
		t.Fatalf("expected no build when already on PATH, ran %v", builds)
	}
}

// Test_Install_Command_Builds_When_Absent verifies that when the command name does
// not resolve on PATH, the Go toolchain builds it into the bin directory.
func Test_Install_Command_Builds_When_Absent(t *testing.T) {
	t.Parallel()
	commands := []sysio.Process_Request{}
	status := setup.Install_Command(&setup.Install_Command_Input{
		Package_Directory: TEST_COMMAND_DIRECTORY,
		Binary_Directory:  TEST_LINK_DIRECTORY,
		Binary_Name:       "m2p",
		IO:                recording_process_io(&commands, nil, 0),
	})
	if !status {
		t.Fatalf("expected success, got status %t", status)
	}
	if builds := commands_named(commands, "sh"); len(builds) != 1 {
		t.Fatalf("expected one build command, ran %d", len(builds))
	}
}

// Test_Install_Command_Reports_A_Build_Failure verifies a failing build reports a
// non-zero exit code.
func Test_Install_Command_Reports_A_Build_Failure(t *testing.T) {
	t.Parallel()
	commands := []sysio.Process_Request{}
	status := setup.Install_Command(&setup.Install_Command_Input{
		Package_Directory: TEST_COMMAND_DIRECTORY,
		Binary_Directory:  TEST_LINK_DIRECTORY,
		Binary_Name:       "sloc",
		IO:                recording_process_io(&commands, nil, 1),
	})
	if status {
		t.Fatal("expected a non-zero status on build failure")
	}
}

// Test_Install_Jj_Skips_Build_When_Already_Built verifies that when the jj binary
// in the bin directory already reports the wanted version, the build is skipped —
// the gate prefix-matches the commit suffix jj appends.
func Test_Install_Jj_Skips_Build_When_Already_Built(t *testing.T) {
	t.Parallel()
	commands := []sysio.Process_Request{}
	status := setup.Install_Jj(&setup.Install_Jj_Input{
		Jj_Directory:     TEST_JJ_DIRECTORY,
		Binary_Directory: TEST_LINK_DIRECTORY,
		IO: recording_process_io(&commands, map[string]string{
			TEST_LINK_DIRECTORY + "/jj": "jj 0.42.0-abc123\n",
		}, 0),
	})
	if !status {
		t.Fatalf("expected success, got status %t", status)
	}
	if builds := commands_named(commands, "sh"); len(builds) != 0 {
		t.Fatalf("expected no build when already built, ran %v", builds)
	}
	if links := commands_named(commands, "ln"); len(links) != 0 {
		t.Fatalf("expected no symlink; jj installs in place, ran %v", links)
	}
}

// Test_Install_Jj_Builds_When_Absent verifies that when no jj at the wanted
// version is present, cargo builds and installs it into the bin directory with
// --root, and no symlink is created.
func Test_Install_Jj_Builds_When_Absent(t *testing.T) {
	t.Parallel()
	commands := []sysio.Process_Request{}
	status := setup.Install_Jj(&setup.Install_Jj_Input{
		Jj_Directory:     TEST_JJ_DIRECTORY,
		Binary_Directory: TEST_LINK_DIRECTORY,
		IO:               recording_process_io(&commands, nil, 0),
	})
	if !status {
		t.Fatalf("expected success, got status %t", status)
	}
	builds := commands_named(commands, "sh")
	if len(builds) != 1 {
		t.Fatalf("expected one build command, ran %d", len(builds))
	}
	if !shared_strings.Contains(joined_text(builds[0].Arguments, " "), "--root") {
		t.Fatalf("expected cargo install --root, ran %v", builds[0].Arguments)
	}
	if links := commands_named(commands, "ln"); len(links) != 0 {
		t.Fatalf("expected no symlink; jj installs in place, ran %v", links)
	}
}

// Test_Install_Jj_Reports_A_Build_Failure verifies a failing build reports a
// non-zero exit code.
func Test_Install_Jj_Reports_A_Build_Failure(t *testing.T) {
	t.Parallel()
	commands := []sysio.Process_Request{}
	status := setup.Install_Jj(&setup.Install_Jj_Input{
		Jj_Directory:     TEST_JJ_DIRECTORY,
		Binary_Directory: TEST_LINK_DIRECTORY,
		IO:               recording_process_io(&commands, nil, 1),
	})
	if status {
		t.Fatal("expected a non-zero status on build failure")
	}
}

// Test_Install_Ripgrep_Skips_Build_When_Already_Built verifies that when the rg
// binary in the bin directory already reports the wanted version, the build is
// skipped — the gate prefix-matches the rev suffix rg appends.
func Test_Install_Ripgrep_Skips_Build_When_Already_Built(t *testing.T) {
	t.Parallel()
	commands := []sysio.Process_Request{}
	status := setup.Install_Ripgrep(&setup.Install_Ripgrep_Input{
		Ripgrep_Directory: TEST_RIPGREP_DIRECTORY,
		Binary_Directory:  TEST_LINK_DIRECTORY,
		IO: recording_process_io(&commands, map[string]string{
			TEST_LINK_DIRECTORY + "/rg": "ripgrep 15.1.0 (rev abc123)\n",
		}, 0),
	})
	if !status {
		t.Fatalf("expected success, got status %t", status)
	}
	if builds := commands_named(commands, "sh"); len(builds) != 0 {
		t.Fatalf("expected no build when already built, ran %v", builds)
	}
	if links := commands_named(commands, "ln"); len(links) != 0 {
		t.Fatalf("expected no symlink; rg installs in place, ran %v", links)
	}
}

// Test_Install_Ripgrep_Builds_When_Absent verifies that when no rg at the wanted
// version is present, cargo builds and installs it into the bin directory with
// --root, and no symlink is created.
func Test_Install_Ripgrep_Builds_When_Absent(t *testing.T) {
	t.Parallel()
	commands := []sysio.Process_Request{}
	status := setup.Install_Ripgrep(&setup.Install_Ripgrep_Input{
		Ripgrep_Directory: TEST_RIPGREP_DIRECTORY,
		Binary_Directory:  TEST_LINK_DIRECTORY,
		IO:                recording_process_io(&commands, nil, 0),
	})
	if !status {
		t.Fatalf("expected success, got status %t", status)
	}
	builds := commands_named(commands, "sh")
	if len(builds) != 1 {
		t.Fatalf("expected one build command, ran %d", len(builds))
	}
	if !shared_strings.Contains(joined_text(builds[0].Arguments, " "), "--root") {
		t.Fatalf("expected cargo install --root, ran %v", builds[0].Arguments)
	}
	if links := commands_named(commands, "ln"); len(links) != 0 {
		t.Fatalf("expected no symlink; rg installs in place, ran %v", links)
	}
}

// Test_Install_Ripgrep_Reports_A_Build_Failure verifies a failing build reports a
// non-zero exit code.
func Test_Install_Ripgrep_Reports_A_Build_Failure(t *testing.T) {
	t.Parallel()
	commands := []sysio.Process_Request{}
	status := setup.Install_Ripgrep(&setup.Install_Ripgrep_Input{
		Ripgrep_Directory: TEST_RIPGREP_DIRECTORY,
		Binary_Directory:  TEST_LINK_DIRECTORY,
		IO:                recording_process_io(&commands, nil, 1),
	})
	if status {
		t.Fatal("expected a non-zero status on build failure")
	}
}

// Test_Install_Fdcli_Skips_Build_When_Already_Built verifies that when the fd binary
// in the bin directory already reports the wanted version, the build is skipped
// rather than recompiled.
func Test_Install_Fdcli_Skips_Build_When_Already_Built(t *testing.T) {
	t.Parallel()
	commands := []sysio.Process_Request{}
	status := setup.Install_Fdcli(&setup.Install_Fdcli_Input{
		Fdcli_Directory:  TEST_FDCLI_DIRECTORY,
		Binary_Directory: TEST_LINK_DIRECTORY,
		IO: recording_process_io(&commands, map[string]string{
			TEST_LINK_DIRECTORY + "/fd": "fd 10.4.2\n",
		}, 0),
	})
	if !status {
		t.Fatalf("expected success, got status %t", status)
	}
	if builds := commands_named(commands, "sh"); len(builds) != 0 {
		t.Fatalf("expected no build when already built, ran %v", builds)
	}
	if links := commands_named(commands, "ln"); len(links) != 0 {
		t.Fatalf("expected no symlink; fd installs in place, ran %v", links)
	}
}

// Test_Install_Fdcli_Builds_When_Absent verifies that when no fd at the wanted
// version is present, cargo builds and installs it into the bin directory with
// --root, and no symlink is created.
func Test_Install_Fdcli_Builds_When_Absent(t *testing.T) {
	t.Parallel()
	commands := []sysio.Process_Request{}
	status := setup.Install_Fdcli(&setup.Install_Fdcli_Input{
		Fdcli_Directory:  TEST_FDCLI_DIRECTORY,
		Binary_Directory: TEST_LINK_DIRECTORY,
		IO:               recording_process_io(&commands, nil, 0),
	})
	if !status {
		t.Fatalf("expected success, got status %t", status)
	}
	builds := commands_named(commands, "sh")
	if len(builds) != 1 {
		t.Fatalf("expected one build command, ran %d", len(builds))
	}
	if !shared_strings.Contains(joined_text(builds[0].Arguments, " "), "--root") {
		t.Fatalf("expected cargo install --root, ran %v", builds[0].Arguments)
	}
	if links := commands_named(commands, "ln"); len(links) != 0 {
		t.Fatalf("expected no symlink; fd installs in place, ran %v", links)
	}
}

// Test_Install_Fdcli_Reports_A_Build_Failure verifies a failing build reports a
// non-zero exit code.
func Test_Install_Fdcli_Reports_A_Build_Failure(t *testing.T) {
	t.Parallel()
	commands := []sysio.Process_Request{}
	status := setup.Install_Fdcli(&setup.Install_Fdcli_Input{
		Fdcli_Directory:  TEST_FDCLI_DIRECTORY,
		Binary_Directory: TEST_LINK_DIRECTORY,
		IO:               recording_process_io(&commands, nil, 1),
	})
	if status {
		t.Fatal("expected a non-zero status on build failure")
	}
}

// Test_Install_Ghostty_Skips_Install_When_Already_Installed verifies that when the
// installed Ghostty app already reports the wanted version, the DMG is not
// downloaded and the app's CLI is relinked — the gate probes the app, not PATH.
func Test_Install_Ghostty_Skips_Install_When_Already_Installed(t *testing.T) {
	t.Parallel()
	commands := []sysio.Process_Request{}
	binary := TEST_APPLICATIONS_DIRECTORY + "/Ghostty.app/Contents/MacOS/ghostty"
	status := setup.Install_Ghostty(&setup.Install_Ghostty_Input{
		Applications_Directory: TEST_APPLICATIONS_DIRECTORY,
		Link_Directory:         TEST_LINK_DIRECTORY,
		IO: recording_process_io(&commands, map[string]string{
			binary: "Ghostty 1.3.1\n",
		}, 0),
	})
	if !status {
		t.Fatalf("expected success, got status %t", status)
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
	commands := []sysio.Process_Request{}
	binary := TEST_APPLICATIONS_DIRECTORY + "/Ghostty.app/Contents/MacOS/ghostty"
	status := setup.Install_Ghostty(&setup.Install_Ghostty_Input{
		Applications_Directory: TEST_APPLICATIONS_DIRECTORY,
		Link_Directory:         TEST_LINK_DIRECTORY,
		IO: recording_process_io_exits(&commands, map[string]string{
			binary: "Ghostty 1.3.1\n",
		}, map[string]int{"codesign": 1}),
	})
	if !status {
		t.Fatalf("expected success, got status %t", status)
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
	commands := []sysio.Process_Request{}
	binary := TEST_APPLICATIONS_DIRECTORY + "/Ghostty.app/Contents/MacOS/ghostty"
	setup.Install_Ghostty(&setup.Install_Ghostty_Input{
		Applications_Directory: TEST_APPLICATIONS_DIRECTORY,
		Link_Directory:         TEST_LINK_DIRECTORY,
		IO: recording_process_io(&commands, map[string]string{
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
	commands := []sysio.Process_Request{}
	status := setup.Install_Ghostty(&setup.Install_Ghostty_Input{
		Applications_Directory: TEST_APPLICATIONS_DIRECTORY,
		Link_Directory:         TEST_LINK_DIRECTORY,
		IO:                     recording_process_io(&commands, nil, 0),
	})
	if !status {
		t.Fatalf("expected success, got status %t", status)
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
	commands := []sysio.Process_Request{}
	status := setup.Install_Ghostty(&setup.Install_Ghostty_Input{
		Applications_Directory: TEST_APPLICATIONS_DIRECTORY,
		Link_Directory:         TEST_LINK_DIRECTORY,
		IO:                     recording_process_io(&commands, nil, 1),
	})
	if status {
		t.Fatal("expected a non-zero status on install failure")
	}
	if links := commands_named(commands, "ln"); len(links) != 0 {
		t.Fatalf("expected no link after a failed install, ran %v", links)
	}
}

// These Main examples use directory-only trees because they specify command and narration
// behavior. Fail the test if a change adds an unrelated file operation to one of them.
func directory_file_system(t *testing.T, loop sysio.IO) (system sysio.IO) {
	t.Helper()
	return loop
}

// The fixed absolute home directory the planned writes are built against; a
// constant keeps the expected destination paths deterministic.
const TEST_HOME = "/home/user"

// The fixed absolute source directory the macos-defaults tests walk — created empty so the
// sync writes nothing and only the injected runner's behavior is under test.
const TEST_SOURCE = "/home/user/code/setup/home"

// The fixed absolute checkout root the Neovim build subpaths are joined onto; a
// constant keeps the expected make and link paths deterministic.
const TEST_REPOSITORY = "/home/user/code/repository"

// The fixed absolute font directory the Install_Fonts tests copy into; a constant
// keeps the expected destinations deterministic.
const TEST_FONT_DIRECTORY = "/home/user/.local/share/fonts"

// The fixed absolute CARGO_HOME the Install_Rust tests gate against; a constant
// keeps the expected toolchain paths deterministic.
const TEST_CARGO_DIRECTORY = "/cargo"

// The fixed absolute PATH directory the Install_Rust tests symlink into; a
// constant keeps the expected link paths deterministic.
const TEST_LINK_DIRECTORY = "/home/user/code/james-orcales/home/.local/bin"

// The fixed absolute direnv source directory the Install_Direnv tests build from; a
// constant keeps the expected build paths deterministic.
const TEST_DIRENV_DIRECTORY = "/home/user/code/james-orcales/third_party/direnv"

// The fixed absolute fzf source directory the Install_Fzf tests build from; a
// constant keeps the expected build paths deterministic.
const TEST_FZF_DIRECTORY = "/home/user/code/james-orcales/third_party/fzf"

// The fixed absolute package directory the Install_Command tests build from; a
// constant keeps the expected build paths deterministic.
const TEST_COMMAND_DIRECTORY = "/home/user/code/james-orcales/maddox"

// The fixed absolute jj source directory the Install_Jj tests build from; a
// constant keeps the expected build paths deterministic.
const TEST_JJ_DIRECTORY = "/home/user/code/james-orcales/third_party/jj"

// The fixed absolute ripgrep source directory the Install_Ripgrep tests build
// from; a constant keeps the expected build paths deterministic.
const TEST_RIPGREP_DIRECTORY = "/home/user/code/james-orcales/third_party/ripgrep"

// The fixed absolute fd source directory the Install_Fdcli tests build from; a
// constant keeps the expected build paths deterministic.
const TEST_FDCLI_DIRECTORY = "/home/user/code/james-orcales/third_party/fd"

// The fixed absolute macOS applications directory the Install_Ghostty tests gate
// against; a constant keeps the expected app and probe paths deterministic.
const TEST_APPLICATIONS_DIRECTORY = "/Applications"

// Reports whether ran holds a command exactly equal to want — name and arguments
// together — so a test can assert one specific invocation happened.
func request_contains(requests []sysio.Process_Request, want []string) (found bool) {
	for _, request := range requests {
		command := append([]string{request.Path}, request.Arguments...)
		if shared_slices.Equal(command, want) {
			return true
		}
	}
	return false
}

// The jlog package keeps its standard writer boundary, while the test stores output in a
// bounded shared buffer. This adapter contains that required standard-library interface.
type buffer_writer struct {
	Buffer *shared_bytes.Buffer
}

// Write is the standard-library writer operation that jlog requires.
func (writer buffer_writer) Write(content []byte) (count int, err error) {
	written, write_err := shared_bytes.Buffer_Write(
		writer.Buffer, shared_bytes.Slice(content))
	return int(written), write_err
}

// The shared text type lets every log assertion use the bounded string algorithms directly.
func buffer_text(buffer *shared_bytes.Buffer) (content shared_strings.Text) {
	return shared_strings.Text(shared_bytes.Buffer_String(buffer))
}

// The test command lists fit one shared text value, so this conversion keeps the shared join
// boundary visible at the assertion site.
func joined_text(values []string, separator string) (joined shared_strings.Text) {
	texts := make(shared_strings.Texts, len(values))
	for index, value := range values {
		texts[index] = shared_strings.Text(value)
	}
	return shared_strings.Join(texts, shared_strings.Text(separator))
}

// Returns a logger writing flat JSON to buffer, floored at debug so the scan's debug lines
// show, with no clock so its output is timestamp-free and byte-for-byte assertable.
func buffer_logger(buffer *shared_bytes.Buffer) (logger jlog.Logger) {
	return jlog.New(jlog.New_Input{
		Writer: buffer_writer{Buffer: buffer}, Floor: jlog.LEVEL_DEBUG,
	})
}

// Shared IO makes the font specifications observe status, copy, and refresh effects directly.
func font_test_io(
	present func(file string) (present bool), copied *[]string, copy_err error,
	commands *[]sysio.Process_Request,
) (system sysio.IO) {
	handles := map[sysio.File]string{}
	next_file := sysio.File(1)
	open := func(path string) (file sysio.File) {
		file = next_file
		next_file++
		handles[file] = path
		return file
	}
	system.Read_Directory = func(string) (
		entries []sysio.Directory_Entry, err error,
	) {
		return nil, nil
	}
	system.Status = func(path string) (status sysio.File_Status, err error) {
		status.Is_Regular = filepath.Dir(path) != TEST_FONT_DIRECTORY
		if !status.Is_Regular {
			status.Is_Regular = present(filepath.Base(path))
		}
		status.Exists = status.Is_Regular
		return status, nil
	}
	system.Open = func(path string) (file sysio.File, err error) { return open(path), nil }
	system.Create = func(path string) (file sysio.File, err error) {
		if copy_err != nil {
			return 0, copy_err
		}
		*copied = append(*copied, filepath.Base(path))
		return open(path), nil
	}
	system.Make_Directory = func(string) (err error) { return nil }
	system.Read = func(
		completion *sysio.Completion, callback sysio.Callback,
		_ sysio.File, buffer []byte, offset int64,
	) {
		content := []byte("font")
		if offset >= int64(len(content)) {
			callback(completion, 0, nil)
			return
		}
		callback(completion, copy(buffer, content[int(offset):]), nil)
	}
	system.Write = func(
		completion *sysio.Completion, callback sysio.Callback,
		_ sysio.File, buffer []byte, _ int64,
	) {
		callback(completion, len(buffer), nil)
	}
	system.Close = func(
		completion *sysio.Completion, callback sysio.Timeout_Callback, file sysio.File,
	) {
		delete(handles, file)
		callback(completion, nil)
	}
	system.Spawn = func(
		completion *sysio.Completion, callback sysio.Process_Callback,
		request sysio.Process_Request, _ systime.Duration,
	) {
		if commands != nil {
			*commands = append(*commands, request)
		}
		callback(completion, sysio.Process_Result{}, nil)
	}
	return system
}

// Returns shared IO whose Spawn records each request into record, replies to a probe
// with responses keyed by the command path, and reports exit for every spawn, so
// a test drives the gate and the build without spawning a real process or a loop.
func recording_process_io(
	record *[]sysio.Process_Request, responses map[string]string, exit int,
) (system sysio.IO) {
	return recording_io(sysio.IO{}, record, responses, exit)
}

// Returns shared IO like recording_process_io whose spawn exit code is chosen per command
// path from exits, defaulting to zero, so a test can fail one specific command — a
// codesign verify — while the gate's version probe and the rest still succeed.
func recording_process_io_exits(
	record *[]sysio.Process_Request, responses map[string]string, exits map[string]int,
) (system sysio.IO) {
	system.Spawn = func(
		completion *sysio.Completion, callback sysio.Process_Callback,
		request sysio.Process_Request, _ systime.Duration,
	) {
		*record = append(*record, request)
		callback(completion, sysio.Process_Result{
			Output: []byte(responses[request.Path]), Exit: exits[request.Path],
		}, nil)
	}
	return system
}

// A copied IO value lets one test combine simulated files with recorded processes.
func recording_io(
	system sysio.IO, record *[]sysio.Process_Request,
	responses map[string]string, exit int,
) (recording sysio.IO) {
	system.Spawn = func(
		completion *sysio.Completion, callback sysio.Process_Callback,
		request sysio.Process_Request, _ systime.Duration,
	) {
		*record = append(*record, request)
		callback(completion, sysio.Process_Result{
			Output: []byte(responses[request.Path]), Exit: exit,
		}, nil)
	}
	return system
}

// Reports whether a codesign invocation pins a signing team: it carries the -R
// requirement flag and a requirement argument naming the team id, so the check
// rejects an app signed by anyone else.
func pins_team(arguments []string, team string) (pins bool) {
	if !shared_slices.Contains(arguments, "-R") {
		return false
	}
	for _, argument := range arguments {
		if shared_strings.Contains(
			shared_strings.Text(argument), shared_strings.Text(team),
		) {
			return true
		}
	}
	return false
}

// Returns only the make invocations from commands, dropping the nvim version
// probe so a test asserts on the build without counting the gate.
func make_commands(commands []sysio.Process_Request) (builds []sysio.Process_Request) {
	return commands_named(commands, "make")
}

// Returns the commands whose executable is name, so a test asserts on one kind of
// invocation — make, cp, fc-cache — without counting the rest.
func commands_named(commands []sysio.Process_Request, name string) (named []sysio.Process_Request) {
	for _, command := range commands {
		if command.Path == name {
			named = append(named, command)
		}
	}
	return named
}
