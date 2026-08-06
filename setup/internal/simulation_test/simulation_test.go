// Package simulation_test calls setup.Main with a synchronous model fabricated from each
// seed and asserts the mirror's properties hold for any seed: convergence and idempotency,
// pruning of an ignored subtree, and rewriting only files that differ.
package simulation_test

import (
	"errors"
	"io"
	"path/filepath"
	"testing"

	setup "local/james-orcales/setup/internal"
	invariant "local/james-orcales/shared/invariant/default"
	sysio "local/james-orcales/shared/io"
	"local/james-orcales/shared/random/prng"
	shared_slices "local/james-orcales/shared/slices"
	shared_strings "local/james-orcales/shared/strings"
	systime "local/james-orcales/shared/time"
)

// The destination directory, relative to the source root, pruned from the source walk so the
// mirror never copies its own output back into itself.
const HARNESS_DESTINATION = "dest"

// A top-level entry marked ignored, so the harness can prove the mirror prunes it.
const HARNESS_IGNORED = "e0"

// Content written over a destination file to force a difference no random source file
// realistically matches, so a later run rewrites exactly the mutated files.
const HARNESS_POISON = "harness-poison-differs-from-any-generated-file"

// The number of seeds the corpus adds, so `go test` alone sweeps a range before `-fuzz`.
const HARNESS_CORPUS = 256

// A separate mutation stream keeps new configuration axes from moving the files that a pinned
// seed poisons.
const HARNESS_MUTATION_SALT = 0x9e3779b97f4a7c15

// Harness_Configuration is the fixed feature subset one seed uses for its complete timeline.
type Harness_Configuration struct {
	Home_Directory     setup.Home_Directory
	Operating_System   setup.Operating_System
	Cargo_Directory    setup.Cargo_Directory
	Data_Directory     setup.Data_Directory
	Color              setup.Console_Color
	Effective_User     setup.Effective_User_Identifier
	Installed          bool
	Version_Matches    bool
	Failure            string
	Dotfile_Bytes      int
	Probe_Output_Bytes int
	Ghostty_Verified   bool
	Empty_Version      bool
	Path_Boundary      int
	Path_Aliases       map[string]string
}

// TestMain registers the complete setup invariant tree before the seed sweep starts.
func TestMain(m *testing.M) {
	invariant.Run_Test_Main(m, "../**")
}

// The harness keeps setup's larger file and batch limits while each shared operation stays
// inside its more narrow boundary.
func Test_Shared_Algorithm_Adapters(t *testing.T) {
	t.Parallel()
	repeated := repeat_text("ab", shared_strings.TEXT_SIZE_MAXIMUM)
	if len(repeated) != 2*shared_strings.TEXT_SIZE_MAXIMUM {
		t.Fatalf("repeated text size = %d, want %d", len(repeated), 8192)
	}
	source := []byte(repeated)
	clone := clone_bytes(source)
	clone[0] = 'x'
	if source[0] != 'a' {
		t.Fatal("the byte clone aliases its source")
	}
	lines := []string{repeat_text("x", 4000), repeat_text("y", 4000)}
	result := split_lines(newline_delimited_text(lines))
	if !shared_slices.Equal(result, lines) {
		t.Fatalf("newline-delimited text = %v, want the input lines", result)
	}
}

// Fuzz_Main calls Main over the synchronous file model that each seed draws.
func Fuzz_Main(f *testing.F) {
	for seed := uint64(0); seed < HARNESS_CORPUS; seed++ {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, seed uint64) {
		drive(t, seed)
	})
}

// Calls setup.Main over one seeded model and asserts every mirror invariant: a second run over
// the freshly mirrored tree writes nothing (convergence, idempotency, no rewrite of an equal
// file), the ignored subtree never reaches the destination (prune), and after some
// destination files are made to differ, a run rewrites exactly those (overwrite).
func drive(t *testing.T, seed uint64) {
	configuration := harness_configuration(seed)
	clock, _ := systime.Virtual_Clock_To_Clock(
		systime.Virtual_Clock{Resolution: systime.NANOSECOND})
	system := simulation_file_system(seed, configuration)
	input := &setup.Main_Input{
		Environment: &setup.Environment_Input{
			Clock: clock, Console: io.Discard,
			Effective_User_Identifier: configuration.Effective_User,
			Color:                     configuration.Color,
			Home_Directory:            configuration.Home_Directory,
			Operating_System:          configuration.Operating_System,
			Cargo_Directory:           configuration.Cargo_Directory,
			Data_Directory:            configuration.Data_Directory,
		},
		IO: system,
	}
	// The first run must succeed for every well-formed seed — with no faults injected, Main
	// fails only on a Plan or write error, neither reachable here. If that ever changes this
	// catches it instead of skipping the seed blind. When faults land it becomes an explicit,
	// reason-carrying, mostly-false skip — never a silent return.
	status := setup.Main(input)
	if status == setup.EXIT_USAGE {
		return
	}
	if status == setup.EXIT_FAILURE {
		return
	}
	if status != setup.EXIT_SUCCESS {
		t.Fatal("the first mirror run over a fault-free tree failed")
	}
	writes := harness_count_writes(&system, configuration)
	input.IO = system
	second_status := setup.Main(input)
	if second_status != 0 {
		t.Fatalf("the second mirror run failed with status %d", second_status)
	}
	if *writes != 0 {
		t.Fatalf("a converged mirror rewrote %d files on a repeat run", *writes)
	}
	harness_assert_pruned(t, &system, configuration)
	mutated := harness_mutate(&system, configuration, seed)
	*writes = 0
	if setup.Main(input) != 0 {
		t.Fatal("the mirror run after mutation failed")
	}
	if *writes != mutated {
		t.Fatalf("rewrote %d files, want the %d made to differ", *writes, mutated)
	}
}

// Draws each configuration axis from its own stream, then holds the result for the whole run.
func harness_configuration(seed uint64) (configuration Harness_Configuration) {
	root := prng.New(seed)
	operating_system := prng.Generator_Split(&root)
	cargo := prng.Generator_Split(&root)
	data := prng.Generator_Split(&root)
	color := prng.Generator_Split(&root)
	installed := prng.Generator_Split(&root)
	identifier := prng.Generator_Split(&root)
	failure := prng.Generator_Split(&root)
	version := prng.Generator_Split(&root)
	home := prng.Generator_Split(&root)
	dotfile_bytes := prng.Generator_Split(&root)
	probe_output_bytes := prng.Generator_Split(&root)
	ghostty_verified := prng.Generator_Split(&root)
	empty_version := prng.Generator_Split(&root)
	path_boundary := prng.Generator_Split(&root)
	configuration.Home_Directory = setup.Home_Directory(harness_home(
		prng.Generator_Element(&home, []int{2, 4, 64})))
	configuration.Operating_System = prng.Generator_Element(
		&operating_system,
		[]setup.Operating_System{"aix", "linux", "darwin", "freebsd", "dragonfly"})
	configuration.Cargo_Directory = setup.Cargo_Directory(prng.Generator_Element(
		&cargo, []string{"", "x", "xx", repeat_text("x", 64)}))
	configuration.Data_Directory = setup.Data_Directory(prng.Generator_Element(
		&data, []string{"", "x", "xx", repeat_text("x", 64)}))
	configuration.Color = setup.Console_Color(prng.Generator_Boolean(&color))
	configuration.Installed = prng.Generator_Boolean(&installed)
	configuration.Version_Matches = prng.Generator_Boolean(&version)
	configuration.Effective_User = prng.Generator_Element(
		&identifier, []setup.Effective_User_Identifier{
			0, 1, 2, setup.EFFECTIVE_USER_IDENTIFIER_MAX,
		})
	configuration.Failure = prng.Generator_Element(&failure, []string{
		"", "direnv", "dotfiles", "defaults", "fonts", "nvim", "fzf", "maddox",
		"rust", "rust-link", "jj", "rg", "fd", "ghostty",
	})
	configuration.Dotfile_Bytes = prng.Generator_Element(
		&dotfile_bytes, []int{-1, 1, 2, setup.DOTFILE_PAYLOAD_BYTES_MAX})
	configuration.Probe_Output_Bytes = prng.Generator_Element(
		&probe_output_bytes, []int{0, 1, 2})
	configuration.Ghostty_Verified = prng.Generator_Boolean(&ghostty_verified)
	configuration.Empty_Version = prng.Generator_Boolean(&empty_version)
	configuration.Path_Boundary = prng.Generator_Element(
		&path_boundary, []int{0, 1, 2})
	configuration.Path_Aliases = map[string]string{}
	return harness_align(configuration)
}

// Aligns features whose rare work branch needs a specific host or prior state.
func harness_align(input Harness_Configuration) (configuration Harness_Configuration) {
	configuration = input
	if configuration.Failure != "" {
		configuration.Installed = false
	}
	if configuration.Failure == "defaults" {
		configuration.Operating_System = "darwin"
	}
	if configuration.Failure == "ghostty" {
		configuration.Operating_System = "darwin"
	}
	if configuration.Failure == "fonts" {
		configuration.Operating_System = "linux"
	}
	if configuration.Failure == "rust" {
		configuration.Cargo_Directory = "xx"
	}
	if configuration.Failure == "rust-link" {
		configuration.Cargo_Directory = "xx"
	}
	if configuration.Failure == "rust-link" {
		configuration.Installed = true
		configuration.Version_Matches = true
	}
	if configuration.Probe_Output_Bytes != 0 {
		// A mismatched installed version is the only path that returns the controlled
		// one-byte and two-byte probe outputs.
		configuration.Failure = ""
		configuration.Installed = true
		configuration.Version_Matches = false
	}
	if configuration.Ghostty_Verified {
		configuration.Failure = ""
		configuration.Operating_System = "darwin"
		configuration.Installed = true
		configuration.Version_Matches = true
	}
	if configuration.Empty_Version {
		configuration.Failure = ""
		configuration.Operating_System = "linux"
		configuration.Installed = true
		configuration.Version_Matches = false
		configuration.Probe_Output_Bytes = 0
	}
	if configuration.Path_Boundary != 0 {
		if configuration.Failure != "" {
			configuration.Path_Boundary = 0
		} else {
			configuration.Home_Directory = setup.Home_Directory(
				repeat_text("/h", 32))
			configuration.Operating_System = "linux"
			configuration.Cargo_Directory = "xx"
			configuration.Installed = true
			configuration.Version_Matches = true
		}
	}
	return configuration
}

// Setup permits text that is larger than one shared text value, so the harness repeats one
// bounded part at a time without changing the seed-derived size.
func repeat_text(text string, count int) (repeated string) {
	if count < 0 {
		return string(shared_strings.Repeat(
			shared_strings.Text(text), shared_strings.Repeat_Count(count)))
	}
	if count == 0 {
		return ""
	}
	if len(text) == 0 {
		return ""
	}
	if len(text) > shared_strings.TEXT_SIZE_MAXIMUM {
		return string(shared_strings.Repeat(
			shared_strings.Text(text), shared_strings.Repeat_Count(count)))
	}
	part_count_max := shared_strings.TEXT_SIZE_MAXIMUM / len(text)
	content := make([]byte, len(text)*count)
	position := 0
	for copies := count; copies != 0; {
		part_count := copies
		if part_count > part_count_max {
			part_count = part_count_max
		}
		part := shared_strings.Repeat(
			shared_strings.Text(text), shared_strings.Repeat_Count(part_count))
		position += copy(content[position:], string(part))
		copies -= part_count
	}
	return string(content)
}

// A simulated dotfile can exceed the shared slice limit, so each cloned part stays bounded.
func clone_bytes(content []byte) (clone []byte) {
	if len(content) == 0 {
		return content
	}
	clone = make([]byte, len(content))
	for offset := 0; offset < len(content); offset += shared_slices.SLICE_COUNT_MAXIMUM {
		end := offset + shared_slices.SLICE_COUNT_MAXIMUM
		if end > len(content) {
			end = len(content)
		}
		copy(clone[offset:end], shared_slices.Clone(content[offset:end]))
	}
	return clone
}

// Setup paths fit one shared text value, so path queries cross that boundary at one helper.
func text_contains(text string, part string) (contains bool) {
	return bool(shared_strings.Contains(
		shared_strings.Text(text), shared_strings.Text(part)))
}

// Setup paths fit one shared text value, so prefix queries cross that boundary at one helper.
func text_has_prefix(text string, prefix string) (present bool) {
	return bool(shared_strings.Has_Prefix(
		shared_strings.Text(text), shared_strings.Text(prefix)))
}

// Setup paths fit one shared text value, so prefix removal keeps the shared result bounded.
func text_trim_prefix(text string, prefix string) (trimmed string) {
	return string(shared_strings.Trim_Prefix(
		shared_strings.Text(text), shared_strings.Text(prefix)))
}

// A check-ignore batch can exceed one shared text value, so each joined group stays inside the
// shared boundary without changing the process input.
func newline_delimited_text(lines []string) (content []byte) {
	group := shared_strings.Texts{}
	group_size := 0
	flush := func() {
		if len(group) == 0 {
			return
		}
		content = append(content, string(shared_strings.Join(group, "\n"))...)
		content = append(content, '\n')
		group = group[:0]
		group_size = 0
	}
	for _, line := range lines {
		separator_size := 0
		if len(group) != 0 {
			separator_size = 1
		}
		if group_size+separator_size+len(line) > shared_strings.TEXT_SIZE_MAXIMUM {
			flush()
		}
		group = append(group, shared_strings.Text(line))
		group_size += len(line)
		if len(group) > 1 {
			group_size++
		}
	}
	flush()
	return content
}

// A generated batch can exceed one shared text value, so each newline search stays bounded.
// The harness rejects an overlong process line before it allocates a map key.
func split_lines(content []byte) (lines []string) {
	line_start := 0
	search_start := 0
	for search_start < len(content) {
		search_end := search_start + shared_strings.TEXT_SIZE_MAXIMUM
		if search_end > len(content) {
			search_end = len(content)
		}
		index := shared_strings.Index_Byte(
			shared_strings.Text(content[search_start:search_end]), '\n')
		if index == shared_strings.INDEX_ABSENT {
			search_start = search_end
			continue
		}
		line_end := search_start + int(index)
		if line_end != line_start {
			if line_end-line_start <= setup.CHECK_IGNORE_TARGET_BYTES_MAX {
				lines = append(lines, string(content[line_start:line_end]))
			}
		}
		line_start = line_end + 1
		search_start = line_start
	}
	if line_start < len(content) {
		if len(content)-line_start <= setup.CHECK_IGNORE_TARGET_BYTES_MAX {
			lines = append(lines, string(content[line_start:]))
		}
	}
	return lines
}

// The simulation injects a synchronous seeded model, so only the binary root owns an IO Driver.
func simulation_file_system(
	seed uint64, configuration Harness_Configuration,
) (system sysio.IO) {
	files := simulation_files(seed)
	model := &Simulation_File_Model{
		Files: files, Directories: simulation_directories(files),
		Configuration: configuration, Handles: map[sysio.File]string{}, Next_File: 1,
	}
	system = sysio.IO{
		Status: func(path string) (status sysio.File_Status, err error) {
			return simulation_status(model, path)
		},
		Open: func(path string) (file sysio.File, err error) {
			return simulation_open(model, path)
		},
		Create: func(path string) (file sysio.File, err error) {
			return simulation_create(model, path)
		},
		Make_Directory: func(path string) (err error) {
			return simulation_make_directory(model, path)
		},
		Read: func(
			completion *sysio.Completion, callback sysio.Callback,
			file sysio.File, buffer []byte, offset int64,
		) {
			simulation_read(model, completion, callback, file, buffer, offset)
		},
		Write: func(
			completion *sysio.Completion, callback sysio.Callback,
			file sysio.File, buffer []byte, offset int64,
		) {
			simulation_write(model, completion, callback, file, buffer, offset)
		},
		Close: func(
			completion *sysio.Completion, callback sysio.Timeout_Callback,
			file sysio.File,
		) {
			simulation_close(model, completion, callback, file)
		},
	}
	read_directory := func(path string) (entries []sysio.Directory_Entry, err error) {
		return simulation_read_directory(model.Files, model.Directories, path)
	}
	system.Read_Directory = func(path string) (entries []sysio.Directory_Entry, err error) {
		return harness_directory_entries(read_directory, configuration, path)
	}
	system.Spawn = harness_process_io(configuration)
	return system
}

// Simulation_File_Model retains descriptor state behind one synchronous shared IO value.
type Simulation_File_Model struct {
	Files             map[string][]byte
	Directories       map[string]bool
	Configuration     Harness_Configuration
	Controlled_Source string
	Handles           map[sysio.File]string
	Next_File         sysio.File
}

// Simulation_contents returns the seeded bytes for one external setup path.
func simulation_contents(model *Simulation_File_Model, path string) (
	contents []byte, found bool, err error,
) {
	if text_contains(path, "/third_party/iosevka_nerd_font_mono/") {
		if model.Configuration.Failure == "fonts" {
			return nil, false, errors.New("simulated font read failure")
		}
		return []byte("simulated-font"), true, nil
	}
	mapped := harness_path(model.Configuration, path)
	contents, found = model.Files[mapped]
	if !found {
		return nil, false, nil
	}
	if model.Configuration.Dotfile_Bytes < 0 {
		return clone_bytes(contents), true, nil
	}
	if text_has_prefix(path, harness_source(model.Configuration)+"/") {
		if model.Controlled_Source == "" {
			model.Controlled_Source = path
		}
		if path == model.Controlled_Source {
			controlled := repeat_text("x", model.Configuration.Dotfile_Bytes)
			return []byte(controlled), true, nil
		}
	}
	return clone_bytes(contents), true, nil
}

// Simulation_status reports the seeded path type.
func simulation_status(model *Simulation_File_Model, path string) (
	status sysio.File_Status, err error,
) {
	if text_contains(path, "/third_party/iosevka_nerd_font_mono/") {
		status.Exists, status.Is_Regular = true, true
		return status, nil
	}
	mapped := harness_path(model.Configuration, path)
	_, status.Is_Regular = model.Files[mapped]
	status.Is_Directory = model.Directories[mapped]
	status.Exists = status.Is_Regular
	if status.Is_Directory {
		status.Exists = true
	}
	return status, nil
}

// Simulation_open allocates one readable simulated descriptor.
func simulation_open(
	model *Simulation_File_Model, path string,
) (file sysio.File, err error) {
	_, found, read_err := simulation_contents(model, path)
	if read_err != nil {
		return 0, read_err
	}
	if !found {
		return 0, errors.New("the simulated file is absent")
	}
	return simulation_open_handle(model, path), nil
}

// Simulation_create truncates one simulated path and allocates its descriptor.
func simulation_create(
	model *Simulation_File_Model, path string,
) (file sysio.File, err error) {
	mapped := harness_path(model.Configuration, path)
	model.Files[mapped] = []byte{}
	simulation_add_directories(model.Directories, filepath.Dir(mapped))
	return simulation_open_handle(model, path), nil
}

// Simulation_open_handle retains one external path for later callback operations.
func simulation_open_handle(model *Simulation_File_Model, path string) (file sysio.File) {
	file = model.Next_File
	model.Next_File++
	model.Handles[file] = path
	return file
}

// Simulation_make_directory adds every modeled parent.
func simulation_make_directory(model *Simulation_File_Model, path string) (err error) {
	simulation_add_directories(
		model.Directories, harness_path(model.Configuration, path))
	return nil
}

// Simulation_read copies one descriptor range and retires it inline.
func simulation_read(
	model *Simulation_File_Model,
	completion *sysio.Completion, callback sysio.Callback,
	file sysio.File, buffer []byte, offset int64,
) {
	path, present := model.Handles[file]
	if !present {
		callback(completion, 0, errors.New("the simulated file handle is absent"))
		return
	}
	contents, found, read_err := simulation_contents(model, path)
	if read_err != nil {
		callback(completion, 0, read_err)
		return
	}
	if !found {
		callback(completion, 0, nil)
		return
	}
	if offset < 0 {
		callback(completion, 0, errors.New("the simulated read offset is invalid"))
		return
	}
	if offset > int64(len(contents)) {
		callback(completion, 0, errors.New("the simulated read offset is invalid"))
		return
	}
	callback(completion, copy(buffer, contents[int(offset):]), nil)
}

// Simulation_write stores one descriptor range and retires it inline.
func simulation_write(
	model *Simulation_File_Model,
	completion *sysio.Completion, callback sysio.Callback,
	file sysio.File, buffer []byte, offset int64,
) {
	path, present := model.Handles[file]
	if !present {
		callback(completion, 0, errors.New("the simulated write is invalid"))
		return
	}
	if offset < 0 {
		callback(completion, 0, errors.New("the simulated write is invalid"))
		return
	}
	mapped := harness_path(model.Configuration, path)
	end := int(offset) + len(buffer)
	if end > len(model.Files[mapped]) {
		growth := make([]byte, end-len(model.Files[mapped]))
		model.Files[mapped] = append(model.Files[mapped], growth...)
	}
	copy(model.Files[mapped][int(offset):], buffer)
	callback(completion, len(buffer), nil)
}

// Simulation_close releases one simulated descriptor and retires it inline.
func simulation_close(
	model *Simulation_File_Model,
	completion *sysio.Completion, callback sysio.Timeout_Callback, file sysio.File,
) {
	delete(model.Handles, file)
	callback(completion, nil)
}

// Each payload uses its own stream, so a new file does not move an existing pinned payload.
func simulation_files(seed uint64) (files map[string][]byte) {
	root := prng.New(seed)
	profile := prng.Generator_Split(&root)
	configuration := prng.Generator_Split(&root)
	ignored := prng.Generator_Split(&root)
	boundary := prng.Generator_Split(&root)
	nested_boundary := prng.Generator_Split(&root)
	payloads := []string{"a", "bb", "ccc", "dddd"}
	return map[string][]byte{
		"/profile":       []byte(prng.Generator_Element(&profile, payloads)),
		"/nested/config": []byte(prng.Generator_Element(&configuration, payloads)),
		"/e0/ignored":    []byte(prng.Generator_Element(&ignored, payloads)),
		"/f0":            []byte(prng.Generator_Element(&boundary, []string{"", "x"})),
		"/d0/x":          []byte(prng.Generator_Element(&nested_boundary, payloads)),
	}
}

// The directory index derives only from seeded file paths and preserves an explicit root.
func simulation_directories(files map[string][]byte) (directories map[string]bool) {
	directories = map[string]bool{"/": true}
	for path := range files {
		simulation_add_directories(directories, filepath.Dir(path))
	}
	return directories
}

// Each parent becomes visible to Read_Directory after a modeled write creates it.
func simulation_add_directories(directories map[string]bool, directory string) {
	directories[directory] = true
	for directory != "/" {
		directory = filepath.Dir(directory)
		directories[directory] = true
	}
}

// The model sorts map-derived entries, so the same seed always gives the same walk order.
func simulation_read_directory(
	files map[string][]byte, directories map[string]bool, path string,
) (entries []sysio.Directory_Entry, err error) {
	if !directories[path] {
		return nil, errors.New("the simulated directory is absent")
	}
	children := map[string]bool{}
	for file := range files {
		if filepath.Dir(file) == path {
			children[filepath.Base(file)] = false
		}
	}
	for directory := range directories {
		if directory != path {
			if filepath.Dir(directory) == path {
				children[filepath.Base(directory)] = true
			}
		}
	}
	for name, is_directory := range children {
		entries = append(entries, sysio.Directory_Entry{
			Name: name, Is_Directory: is_directory})
	}
	shared_slices.Sort_Function(entries, func(
		left, right sysio.Directory_Entry,
	) (comparison shared_slices.Comparison) {
		return shared_slices.Comparison(shared_strings.Compare(
			shared_strings.Text(left.Name), shared_strings.Text(right.Name)))
	})
	return entries, nil
}

// Reads one simulated directory and gives one existing entry a boundary path
// when the seed selected that path feature.
func harness_directory_entries(
	read_directory func(path string) (entries []sysio.Directory_Entry, err error),
	configuration Harness_Configuration, path string,
) (entries []sysio.Directory_Entry, err error) {
	if configuration.Failure == "dotfiles" {
		if path == harness_source(configuration) {
			return nil, errors.New("simulated dotfile scan failure")
		}
	}
	entries, read_err := read_directory(harness_path(configuration, path))
	if read_err != nil {
		return nil, read_err
	}
	if configuration.Path_Boundary == 0 {
		return entries, nil
	}
	if path != harness_source(configuration) {
		return entries, nil
	}
	want_directory := configuration.Path_Boundary == 2
	for index := range entries {
		if entries[index].Name == HARNESS_DESTINATION {
			continue
		}
		if entries[index].Name == HARNESS_IGNORED {
			continue
		}
		if entries[index].Is_Directory != want_directory {
			continue
		}
		alias := repeat_text("p", setup.RELATIVE_FILE_PATH_BYTES_MAX)
		alias_path := filepath.Join(path, alias)
		configuration.Path_Aliases[alias_path] = filepath.Join(path, entries[index].Name)
		entries[index].Name = alias
		break
	}
	return entries, nil
}

// Builds a bounded absolute home from two-byte path components.
func harness_home(bytes int) (home string) {
	return repeat_text("/h", bytes/2)
}

// Returns Main's logical dotfile source for one configured home.
func harness_source(configuration Harness_Configuration) (source string) {
	return filepath.Join(string(configuration.Home_Directory), "code/james-orcales/home")
}

// Maps Main's fixed source and home paths onto the simulator's source and destination roots.
func harness_path(configuration Harness_Configuration, path string) (mapped string) {
	for alias, original := range configuration.Path_Aliases {
		if path == alias {
			path = original
			break
		}
		if text_has_prefix(path, alias+"/") {
			path = original + text_trim_prefix(path, alias)
			break
		}
	}
	source := harness_source(configuration)
	home := string(configuration.Home_Directory)
	data := string(configuration.Data_Directory)
	if data != "" {
		if path == data {
			return "/dest/data"
		}
		if text_has_prefix(path, data+"/") {
			return filepath.Join("/dest/data", text_trim_prefix(path, data+"/"))
		}
	}
	if path == source {
		return "/"
	}
	if text_has_prefix(path, source+"/") {
		return "/" + text_trim_prefix(path, source+"/")
	}
	if path == home {
		return filepath.Join("/", HARNESS_DESTINATION)
	}
	if text_has_prefix(path, home+"/") {
		return filepath.Join(
			"/", HARNESS_DESTINATION, text_trim_prefix(path, home+"/"))
	}
	return path
}

// Returns one seed-fixed process fabric and retains the tool named by its latest version probe.
func harness_process_io(configuration Harness_Configuration) (
	spawn func(
		completion *sysio.Completion, callback sysio.Process_Callback,
		request sysio.Process_Request, deadline systime.Duration,
	),
) {
	tool := ""
	return func(
		completion *sysio.Completion, callback sysio.Process_Callback,
		request sysio.Process_Request, _ systime.Duration,
	) {
		if request.Path == "which" {
			tool = request.Arguments[0]
		} else if harness_version(request) != "" {
			tool = filepath.Base(request.Path)
		}
		result := harness_spawn(configuration, request)
		if harness_command_fails(configuration.Failure, tool, request) {
			result.Exit = 1
		}
		callback(completion, result, nil)
	}
}

// Reports installed tool versions and evaluates the one gitignore batch that Mirror submits.
func harness_spawn(
	configuration Harness_Configuration, request sysio.Process_Request,
) (result sysio.Process_Result) {
	if request.Path == "git" {
		return harness_git_ignore(configuration, request)
	}
	if request.Path == "which" {
		if !configuration.Installed {
			return result
		}
		name := request.Arguments[0]
		if name == "nvim" {
			executable := filepath.Join(
				harness_source(configuration), ".local/bin/nvim")
			return sysio.Process_Result{
				Output: []byte(executable + "\n"),
			}
		}
		return sysio.Process_Result{Output: []byte("/installed/" + name + "\n")}
	}
	version := harness_version(request)
	if version != "" {
		if configuration.Installed {
			if configuration.Version_Matches {
				return sysio.Process_Result{Output: []byte(version + "\n")}
			}
			return sysio.Process_Result{Output: []byte(repeat_text(
				"x", configuration.Probe_Output_Bytes))}
		}
	}
	return result
}

// Reports the expected version for a direct executable probe.
func harness_version(request sysio.Process_Request) (version string) {
	versions := map[string]string{
		"direnv": "2.37.1", "nvim": "NVIM v0.12.3", "fzf": "0.73.1",
		"rustc": "rustc 1.96.0", "jj": "jj 0.42.0", "rg": "ripgrep 15.1.0",
		"fd": "fd 10.4.2", "ghostty": "Ghostty 1.3.1",
	}
	return versions[filepath.Base(request.Path)]
}

// Selects only the configured step's work command, never its version probe.
func harness_command_fails(
	failure string, tool string, request sysio.Process_Request,
) (fails bool) {
	if failure == "defaults" {
		return request.Path == "defaults"
	}
	if failure == "rust-link" {
		return request.Path == "ln"
	}
	if failure == "nvim" {
		return request.Path == "make"
	}
	if failure == "fonts" {
		return false
	}
	if failure == "dotfiles" {
		return false
	}
	if failure == "" {
		return false
	}
	if failure != tool {
		return false
	}
	return harness_version(request) == ""
}

// Converts ignored simulated paths back to the absolute names that git prints.
func harness_git_ignore(
	configuration Harness_Configuration, request sysio.Process_Request,
) (result sysio.Process_Result) {
	ignored := []string{}
	for _, target := range split_lines(request.Input) {
		relative := text_trim_prefix(target, harness_source(configuration)+"/")
		if target != "" {
			if harness_ignores_one(relative) {
				ignored = append(ignored, target)
			}
		}
	}
	if len(ignored) != 0 {
		result.Output = newline_delimited_text(ignored)
	}
	return result
}

// Classifies a batch of source paths, returning the set that is the destination root or the
// ignored subtree — pruned so the mirror neither copies into itself nor syncs the ignored
// entry. Batched to match the Is_Ignored seam the mirror now calls once per tree level.
func harness_ignore(relatives []string) (ignored map[string]bool) {
	ignored = map[string]bool{}
	for _, relative := range relatives {
		if harness_ignores_one(relative) {
			ignored[relative] = true
		}
	}
	return ignored
}

// Reports whether one source path is the destination root or the ignored subtree, the
// per-path rule the harness's own walk shares with the batched predicate.
func harness_ignores_one(relative string) (ignored bool) {
	if relative == HARNESS_DESTINATION {
		return true
	}
	if text_has_prefix(relative, HARNESS_DESTINATION+"/") {
		return true
	}
	if relative == HARNESS_IGNORED {
		return true
	}
	return text_has_prefix(relative, HARNESS_IGNORED+"/")
}

// Wraps shared IO Create to count the files a mirror run writes, returning the live count.
func harness_count_writes(
	system *sysio.IO, configuration Harness_Configuration,
) (writes *int) {
	count := 0
	destinations := map[string]bool{}
	for _, relative := range harness_source_files(system, configuration) {
		destinations[filepath.Join(string(configuration.Home_Directory), relative)] = true
	}
	create := system.Create
	system.Create = func(path string) (file sysio.File, err error) {
		if destinations[path] {
			count++
		}
		return create(path)
	}
	return &count
}

// Asserts the ignored subtree never reached the destination.
func harness_assert_pruned(
	t *testing.T, system *sysio.IO, configuration Harness_Configuration,
) {
	status, _ := system.Status(filepath.Join(
		string(configuration.Home_Directory), HARNESS_IGNORED))
	if status.Exists {
		t.Fatalf("the ignored subtree %q was synced to the destination", HARNESS_IGNORED)
	}
}

// Overwrites a seed-chosen subset of the destination's mirrored files with poison so they
// differ from their source, returning how many were changed.
func harness_mutate(
	system *sysio.IO, configuration Harness_Configuration, seed uint64,
) (count int) {
	generator := prng.New(seed ^ HARNESS_MUTATION_SALT)
	for _, relative := range harness_source_files(system, configuration) {
		if len(relative) == setup.RELATIVE_FILE_PATH_BYTES_MAX {
			continue
		}
		if prng.Generator_Boolean(&generator) {
			continue
		}
		harness_overwrite(system, filepath.Join("/", HARNESS_DESTINATION, relative))
		count++
	}
	return count
}

// Lists the non-ignored source files under the root, walking the tree through Read_Directory.
func harness_source_files(
	system *sysio.IO, configuration Harness_Configuration,
) (relatives []string) {
	relatives = []string{}
	worklist := []string{"."}
	for len(worklist) > 0 {
		directory := worklist[len(worklist)-1]
		worklist = worklist[:len(worklist)-1]
		entries, err := system.Read_Directory(filepath.Join(
			harness_source(configuration), directory))
		if err != nil {
			continue
		}
		for _, entry := range entries {
			relative := filepath.Join(directory, entry.Name)
			if harness_ignores_one(relative) {
				continue
			}
			if entry.Is_Directory {
				worklist = append(worklist, relative)
				continue
			}
			relatives = append(relatives, relative)
		}
	}
	return relatives
}

// Overwrites path with the poison content through the loop, driving the write and the close
// to completion.
func harness_overwrite(system *sysio.IO, path string) {
	file, create_err := system.Create(path)
	if create_err != nil {
		return
	}
	completion := sysio.Completion{}
	system.Write(&completion, func(_ *sysio.Completion, _ int, _ error) {},
		file, []byte(HARNESS_POISON), 0)
	system.Close(&completion, func(_ *sysio.Completion, _ error) {}, file)
}
