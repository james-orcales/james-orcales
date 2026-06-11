// Package setup mirrors a tree of dotfiles into the home directory in one
// direction, repo to home. It is the pure library tier of the setup binary:
// Plan diffs two injected filesystems and returns the writes a sync would make,
// and Main performs them through an injected writer, so the logic here binds to
// no real filesystem and stays a black box under test.
package setup

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"strings"
)

// Dotfile_bytes_max bounds a single dotfile read into one fixed buffer. 1 MiB
// dwarfs any real configuration file yet caps memory against a pathological
// input, satisfying the linter's unbounded-read ban.
const dotfile_bytes_max = 1048576

// Exit_failure is the process exit code for a planning or write failure during
// an otherwise well-formed run.
const exit_failure = 1

// Main_Input carries the injected dependencies Main needs to sync dotfiles.
type Main_Input struct {
	// Source is the dotfiles tree to mirror from.
	Source fs.FS
	// Destination is a read-only view of the home directory, used for diffing.
	Destination fs.FS
	// Destination_Directory is the absolute home directory writes land under.
	Destination_Directory string
	// Write_File persists contents at an absolute path, creating parent
	// directories. It is injected so this library tier never binds to os
	// directly; package main supplies the filesystem-backed implementation.
	Write_File func(path string, contents []byte) (err error)
	// Operating_System gates the macos defaults step, which runs only on
	// "darwin" (a runtime.GOOS value).
	Operating_System string
	// Run_Command runs an external program by name with arguments. It is injected
	// so this library tier never executes anything; package main supplies the
	// exec-backed runner used for the macos defaults.
	Run_Command func(name string, arguments []string) (err error)
	// Stdout receives one line naming each file written.
	Stdout io.Writer
	// Stderr receives a diagnostic line when planning or a write fails.
	Stderr io.Writer
}

// Main syncs the source dotfiles into the home directory and, on darwin, applies
// the macos defaults, returning a process exit code. It is the binary's single
// entry point, kept here so package main stays a thin, untested shell.
func Main(input *Main_Input) (status_code int) {
	writes, plan_err := Plan(&Plan_Input{
		Source:                input.Source,
		Destination:           input.Destination,
		Destination_Directory: input.Destination_Directory,
	})
	if plan_err != nil {
		fmt.Fprintf(input.Stderr, "setup: %v\n", plan_err)
		return exit_failure
	}
	for _, write := range writes {
		write_err := input.Write_File(write.Destination_Path, write.Contents)
		if write_err != nil {
			fmt.Fprintf(input.Stderr, "setup: %v\n", write_err)
			return exit_failure
		}
		fmt.Fprintf(input.Stdout, "setup: wrote %s\n", write.Destination_Path)
	}
	// The macos defaults touch macOS-only preference domains, so they run there
	// and nowhere else.
	if input.Operating_System != "darwin" {
		return 0
	}
	return apply_macos_defaults(input.Run_Command, input.Stderr)
}

// Takes just the runner and the error sink it uses, not the whole Main_Input: a
// function whose parameter is *Main_Input would be forced to be named Main. Runs
// every macos defaults command, stopping at the first that fails.
func apply_macos_defaults(
	run func(name string, arguments []string) (err error), stderr io.Writer,
) (status_code int) {
	for _, command := range macos_commands() {
		run_err := run(command.Name, command.Arguments)
		if run_err != nil {
			fmt.Fprintf(stderr, "setup: %v\n", run_err)
			return exit_failure
		}
	}
	return 0
}

// Macos_defaults_script lists the `defaults write` and `killall` commands that
// configure macOS, one per line, parsed into commands at run time. The clock
// date format is appended separately by macos_commands because its value holds
// spaces that whitespace-splitting a line would break apart.
const macos_defaults_script = `
defaults write com.apple.dock autohide -bool true
defaults write com.apple.dock autohide-delay -float 0
defaults write com.apple.dock autohide-time-modifier -int 0
defaults write com.apple.dock orientation -string left
defaults write com.apple.dock show-recents -bool false
killall Dock
defaults write com.apple.finder AppleShowAllExtensions -bool true
defaults write com.apple.finder AppleShowAllFiles -bool true
defaults write com.apple.finder AppleShowScrollBars -bool true
defaults write com.apple.finder ShowPathbar -bool true
defaults write com.apple.finder ShowStatusBar -bool true
defaults write com.apple.finder NewWindowTarget -string Home
defaults write com.apple.finder FXPreferredViewStyle -string Nlsv
defaults write com.apple.finder FXDefaultSearchScope -string SCcf
defaults write com.apple.finder _FXSortFoldersFirst -bool true
defaults write com.apple.finder _FXShowPosixPathInTitle -bool true
killall Finder
defaults write com.apple.screensaver askForPassword -int 1
defaults write com.apple.screensaver askForPasswordDelay -int 0
defaults write com.apple.AdLib allowApplePersonalizedAdvertising -bool false
defaults write com.apple.desktopservices DSDontWriteNetworkStores -bool true
defaults write com.apple.desktopservices DSDontWriteUSBStores -bool true
defaults write com.apple.SoftwareUpdate AutomaticCheckEnabled -bool true
defaults write com.apple.SoftwareUpdate ScheduleFrequency -int 1
defaults write com.apple.SoftwareUpdate AutomaticDownload -int 0
defaults write com.apple.SoftwareUpdate CriticalUpdateInstall -int 1
defaults write NSGlobalDomain com.apple.mouse.linear -bool true
defaults write NSGlobalDomain WebKitDeveloperExtras -bool true
defaults write NSGlobalDomain AppleShowScrollBars -string always
defaults write NSGlobalDomain NSAutomaticCapitalizationEnabled -bool false
defaults write NSGlobalDomain NSAutomaticDashSubstitutionEnabled -bool false
defaults write NSGlobalDomain NSAutomaticInlinePredictionEnabled -bool false
defaults write NSGlobalDomain NSAutomaticPeriodSubstitutionEnabled -bool false
defaults write NSGlobalDomain NSAutomaticQuoteSubstitutionEnabled -bool false
defaults write NSGlobalDomain NSAutomaticSpellingCorrectionEnabled -bool false
`

// One external program invocation parsed from macos_defaults_script.
type macos_command struct {
	Name      string
	Arguments []string
}

// Parses macos_defaults_script into one command per non-blank line and appends
// the clock date format command, whose spaced value cannot share the line format.
func macos_commands() (commands []macos_command) {
	commands = []macos_command{}
	for line := range strings.Lines(macos_defaults_script) {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		commands = append(commands, macos_command{
			Name:      fields[0],
			Arguments: fields[1:],
		})
	}
	return append(commands, macos_command{
		Name: "defaults",
		Arguments: []string{
			"write", "com.apple.menuextra.clock", "DateFormat",
			"-string", "EEE MMM d mm:HH",
		},
	})
}

// File_Write is a single pending write the sync would perform.
type File_Write struct {
	// Destination_Path is the absolute path the contents must be written to: the
	// source file's path relative to the source root, joined onto the home dir.
	Destination_Path string
	// Contents is the exact source bytes to write. Comparing them against the
	// destination is what decided this write was needed at all.
	Contents []byte
}

// Plan_Input carries the injected dependencies Plan needs to decide what to sync.
type Plan_Input struct {
	// Source is the dotfiles tree, walked in full from its root.
	Source fs.FS
	// Destination is a read-only view of the home directory, used only to read
	// existing files so Plan can skip writes that would change nothing.
	Destination fs.FS
	// Destination_Directory is the absolute home directory the relative source
	// paths are mirrored under to form each write's Destination_Path.
	Destination_Directory string
}

// Plan returns the writes that would bring the home directory in line with the
// source dotfiles: every regular file under the source tree, each emitted only
// when the destination is missing or its contents differ.
func Plan(input *Plan_Input) (writes []File_Write, err error) {
	writes = []File_Write{}
	walk_err := fs.WalkDir(input.Source, ".",
		func(source_path string, entry fs.DirEntry, step_err error) (result error) {
			if step_err != nil {
				return step_err
			}
			if entry.IsDir() {
				return nil
			}
			write, planned, plan_err := plan_file(&plan_file_input{
				Input:       input,
				Source_Path: source_path,
			})
			if plan_err != nil {
				return plan_err
			}
			if planned {
				writes = append(writes, write)
			}
			return nil
		})
	if walk_err != nil {
		return nil, walk_err
	}
	return writes, nil
}

// Carries the arguments for planning one source file.
type plan_file_input struct {
	Input       *Plan_Input
	Source_Path string
}

// Decides whether one source file needs syncing. planned is false when the
// destination already holds identical bytes; otherwise it returns the write
// mirroring the source path under the home directory.
func plan_file(input *plan_file_input) (write File_Write, planned bool, err error) {
	source_contents, read_err := read_source(input.Input.Source, input.Source_Path)
	if read_err != nil {
		return File_Write{}, false, read_err
	}
	if destination_matches(input.Input.Destination, input.Source_Path, source_contents) {
		return File_Write{}, false, nil
	}
	destination_path := filepath.Join(input.Input.Destination_Directory, input.Source_Path)
	return File_Write{
		Destination_Path: destination_path,
		Contents:         source_contents,
	}, true, nil
}

// Reads a source file's full contents, bounded by dotfile_bytes_max.
func read_source(source fs.FS, source_path string) (contents []byte, err error) {
	file, open_err := source.Open(source_path)
	if open_err != nil {
		return nil, open_err
	}
	defer file.Close()
	return read_bounded(file)
}

// Reports whether the home directory already holds exactly source_contents at
// relative_path. A destination that is absent or unreadable counts as a
// mismatch, so the file is written — the rule the original installer used.
func destination_matches(
	destination fs.FS, relative_path string, source_contents []byte) (matches bool) {
	file, open_err := destination.Open(relative_path)
	if open_err != nil {
		return false
	}
	defer file.Close()
	destination_contents, read_err := read_bounded(file)
	if read_err != nil {
		return false
	}
	return bytes.Equal(source_contents, destination_contents)
}

// Reads up to dotfile_bytes_max bytes from file into one fixed buffer, erroring
// if the file overflows the cap or a read fails mid-stream, so a silently
// truncated dotfile never passes as the whole file.
func read_bounded(file fs.File) (contents []byte, err error) {
	buffer := make([]byte, dotfile_bytes_max)
	read_total := 0
	for read_total < len(buffer) {
		n, read_err := file.Read(buffer[read_total:])
		read_total += n
		if read_err == io.EOF {
			return buffer[:read_total], nil
		}
		if read_err != nil {
			return nil, read_err
		}
	}
	return nil, errors.New("dotfile exceeds the maximum size")
}
