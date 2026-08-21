package build_test

import (
	"errors"
	"testing"
	"unsafe"

	"local/james-orcales/shared/go/build"
	"local/james-orcales/shared/go/token"
	"local/james-orcales/shared/simulation/nbio"
	"local/james-orcales/shared/simulation/time"
	"local/james-orcales/shared/strconv"
	"local/james-orcales/shared/testify"
)

// Test_Build binds the Build specification leaf before fixture declarations.
func Test_Build(t *testing.T) {
	test_build(t)
}

// Test_Target binds the Target specification leaf before fixture declarations.
func Test_Target(t *testing.T) {
	test_target(t)
}

// Test_Names binds the Names specification leaf before fixture declarations.
func Test_Names(t *testing.T) {
	test_names(t)
}

// Test_Constraints binds the Constraints specification leaf before fixture declarations.
func Test_Constraints(t *testing.T) {
	test_constraints(t)
}

// Test_Systems binds the Systems specification leaf before fixture declarations.
func Test_Systems(t *testing.T) {
	test_systems(t)
}

// Test_Directories binds the Directories specification leaf before fixture declarations.
func Test_Directories(t *testing.T) {
	test_directories(t)
}

// Test_Refusals binds the Refusals specification leaf before fixture declarations.
func Test_Refusals(t *testing.T) {
	test_refusals(t)
}

// Test_Bounds binds the Bounds specification leaf before fixture declarations.
func Test_Bounds(t *testing.T) {
	test_bounds(t)
}

// Test_Allocation binds the Allocation specification leaf before fixture declarations.
func Test_Allocation(t *testing.T) {
	test_allocation(t)
}

type allocation_fixture struct {
	Held build.Boolean
	Ok   build.Boolean
}

// Builds one target of the system, the architecture, and the tags a case names.
func target(system string, architecture string, tags []string) (held build.Target) {
	held.Words[build.WORD_SYSTEM] = token.Source(system)
	held.Words[build.WORD_ARCHITECTURE] = token.Source(architecture)
	for index := range len(tags) {
		held.Tags[index] = token.Source(tags[index])
	}
	held.Counts[build.COUNT_TAG] = build.Count(len(tags))
	return held
}

// Reports whether one file name stands in the target a case names.
func named(name string, system string, architecture string) (held build.Boolean) {
	subject := new(build.Build)
	one := target(system, architecture, nil)
	return build.Names_System(subject, &one, token.Source(name))
}

// Reports whether one constraint holds in the target a case names.
func holds(t *testing.T, text string, one build.Target) (held build.Boolean) {
	t.Helper()
	subject := new(build.Build)
	held, ok := build.Holds_Constraint(subject, &one, token.Source(text))
	testify.True(t, bool(ok), "the reader reads the constraint %q", text)
	return held
}

func test_build(t *testing.T) {
	subject := new(build.Build)
	one := target("linux", "amd64", []string{"race"})
	held, ok := build.Holds_Constraint(subject, &one, token.Source("linux"))
	testify.True(t, bool(ok), "the reader reads one constraint")
	testify.True(t, bool(held), "the file stands in the build its system names")
	held, ok = build.Holds_Constraint(subject, &one, token.Source("windows"))
	testify.True(t, bool(ok), "one state reads one constraint after another")
	testify.False(t, bool(held), "the file stands outside the build of another system")
}

func test_target(t *testing.T) {
	one := target("linux", "amd64", []string{"race", "purego"})
	testify.True(t, bool(holds(t, "race", one)), "a tag the target names holds")
	testify.True(t, bool(holds(t, "purego", one)), "every tag the target names holds")
	testify.False(t, bool(holds(t, "cgo", one)), "a tag the target never names holds nothing")
	testify.True(t, bool(holds(t, "amd64", one)), "the architecture the target names holds")
	bare := target("", "", nil)
	testify.False(t, bool(holds(t, "linux", bare)), "a target that names nothing holds nothing")
}

func test_names(t *testing.T) {
	for _, one := range []struct {
		Name         string
		System       string
		Architecture string
		Held         bool
	}{
		{"build.go", "linux", "amd64", true},
		{"linux.go", "windows", "amd64", true},
		{"file_linux.go", "linux", "amd64", true},
		{"file_linux.go", "windows", "amd64", false},
		{"file_amd64.go", "linux", "amd64", true},
		{"file_amd64.go", "linux", "arm64", false},
		{"file_linux_amd64.go", "linux", "amd64", true},
		{"file_linux_amd64.go", "linux", "arm64", false},
		{"file_linux_amd64.go", "windows", "amd64", false},
		{"file_linux_test.go", "linux", "amd64", true},
		{"file_linux_test.go", "windows", "amd64", false},
		{"file_linux_amd64_test.go", "linux", "amd64", true},
		{"file_linux_amd64_test.go", "linux", "arm64", false},
		{"file_held.go", "linux", "amd64", true},
		{"file_test.go", "linux", "amd64", true},
	} {
		testify.Equal(t, one.Held, bool(named(one.Name, one.System, one.Architecture)),
			"%q stands in the build of %s and %s", one.Name, one.System,
			one.Architecture)
	}
}

func test_constraints(t *testing.T) {
	one := target("linux", "amd64", []string{"race"})
	for _, held := range []struct {
		Text string
		Held bool
	}{
		{"linux", true},
		{"windows", false},
		{"!windows", true},
		{"!linux", false},
		{"linux && amd64", true},
		{"linux && arm64", false},
		{"windows || linux", true},
		{"windows || plan9", false},
		{"(windows || linux) && amd64", true},
		{"(windows || plan9) && amd64", false},
		{"!(windows || plan9)", true},
		{"unix", true},
		{"linux && race", true},
		{"linux && !race", false},
		{"  linux  &&  amd64  ", true},
	} {
		testify.Equal(t, held.Held, bool(holds(t, held.Text, one)),
			"the constraint %q holds", held.Text)
	}
	windows := target("windows", "amd64", nil)
	testify.False(t, bool(holds(t, "unix", windows)),
		"the word unix holds on no system Go calls otherwise")
}

func test_systems(t *testing.T) {
	testify.True(t, bool(named("file_darwin.go", "darwin", "arm64")),
		"a known operating system names the build it stands in")
	testify.True(t, bool(named("file_riscv64.go", "linux", "riscv64")),
		"a known architecture names the build it stands in")
	// A word the list never names is a tag like any other, thus the name says nothing about
	// the system the file stands in.
	testify.True(t, bool(named("file_held.go", "linux", "amd64")),
		"a word neither list names says nothing")
	testify.True(t, bool(named("file_unix.go", "linux", "amd64")),
		"the word unix names no system a file name states")
}

// TEST_ENTRY_COUNT bounds the children one fixture directory hands one pass.
const TEST_ENTRY_COUNT = 512

// TEST_RECORD_COUNT bounds the records one fixture pass writes.
const TEST_RECORD_COUNT = 4096

// TEST_BYTE_COUNT bounds the bytes the names of one fixture read view.
const TEST_BYTE_COUNT = 8192

// TEST_HEADER_COUNT bounds the bytes of one fixture file the read of a constraint reads.
const TEST_HEADER_COUNT = 128

// Failure the stub storage answers an open of a refused file with.
var directory_error = errors.New("the open of one file is refused")

// States one directory the stub storage hands the runner: the children of one pass and the bytes
// each of them opens with.
type directory_fixture struct {
	Names   []string
	Folders []bool
	Headers []string
	Refused string
	Passes  int
	Opened  int
}

// Opens the directory the runner names, or one child of it. The descriptor of a child names the
// slot of that child, thus a read of it finds the bytes the fixture states.
func directory_open(
	state unsafe.Pointer, completion *time.Completion, directory nbio.File, file_path string,
	options nbio.Open_At_Options, callback time.Callback,
) {
	fixture := (*directory_fixture)(state)
	completion.Error = nil
	completion.Data = 1
	if directory != nbio.DIRECTORY_CURRENT {
		completion.Data = 0
		if file_path == fixture.Refused {
			completion.Error = directory_error
			fixture.Opened = fixture.Opened + 1
			callback(completion)
			return
		}
		for index := range len(fixture.Names) {
			if fixture.Names[index] == file_path {
				completion.Data = 2 + index
			}
		}
		fixture.Opened = fixture.Opened + 1
	}
	callback(completion)
}

// Hands the runner one pass over the children of the directory, and no child on the pass behind
// it.
func directory_entries(
	state unsafe.Pointer, completion *time.Completion, directory nbio.File, buffer []byte,
	entries []nbio.Directory_Entry, callback time.Callback,
) {
	fixture := (*directory_fixture)(state)
	completion.Error = nil
	completion.Data = 0
	if fixture.Passes == 0 {
		for index := range len(fixture.Names) {
			if index >= len(entries) {
				break
			}
			entries[index] = nbio.Directory_Entry{
				Name: fixture.Names[index], Is_Directory: fixture.Folders[index],
			}
			completion.Data = index + 1
		}
	}
	fixture.Passes = fixture.Passes + 1
	callback(completion)
}

// Reads the bytes the child of the fixture opens with into the buffer the runner owns.
func directory_read(
	state unsafe.Pointer, completion *time.Completion, file nbio.File, buffer []byte,
	offset int64, timeout time.Duration, callback time.Callback,
) {
	fixture := (*directory_fixture)(state)
	completion.Error = nil
	completion.Data = 0
	slot := int(file) - 2
	if slot >= 0 {
		if slot < len(fixture.Headers) {
			completion.Data = copy(buffer, fixture.Headers[slot])
		}
	}
	callback(completion)
}

// Closes one descriptor the read held.
func directory_close(
	state unsafe.Pointer, completion *time.Completion, file nbio.File, callback time.Callback,
) {
	completion.Error = nil
	completion.Data = 0
	callback(completion)
}

// Drives one directory read to its terminal state, which a composition root does with its loop.
func directory_drive(runner *build.Directory_Runner) {
	for range 4096 {
		if bool(build.Directory_Runner_Stopped(runner)) {
			return
		}
		if bool(build.Directory_Runner_Rearm(runner)) {
			continue
		}
		if bool(build.Directory_Runner_Work_Queued(runner)) {
			continue
		}
		return
	}
}

// Reads one directory the fixture states and hands back the names the read kept.
func read_directory(
	t *testing.T, fixture *directory_fixture, path build.Path, names int,
) (held build.Name_Storage) {
	t.Helper()
	return read_sized(t, fixture, path, TEST_ENTRY_COUNT, TEST_RECORD_COUNT, TEST_BYTE_COUNT)
}

// Reads one directory through storage of the widths stated, which is how a read reaches every
// width the package admits.
func read_sized(
	t *testing.T, fixture *directory_fixture, path build.Path, slots int, records int,
	bytes int,
) (held build.Name_Storage) {
	t.Helper()
	one := target("linux", "amd64", nil)
	loop := nbio.IO{Storage: nbio.Storage{
		State:                           unsafe.Pointer(fixture),
		Open_At_Procedure:               directory_open,
		Get_Directory_Entries_Procedure: directory_entries,
		Read_Procedure:                  directory_read,
	}, Close_Procedure: directory_close}
	runner := build.Directory_Runner{}
	build.Directory_Runner_Init(&runner, loop, &one, path, build.Directory_Memory{
		Entries: make(build.Entry_Storage, slots),
		Records: make(build.Record_Storage, records),
		Names:   make(build.Name_Storage, slots),
		Bytes:   make(build.Name_Bytes, bytes),
		Header:  make(build.Header_Storage, bytes),
		Reader:  new(build.Build),
	})
	directory_drive(&runner)
	testify.True(t, bool(build.Directory_Runner_Stopped(&runner)),
		"the read of one directory stops")
	testify.No_Error(t, build.Directory_Runner_Status(&runner), "the read states no error")
	return build.Directory_Runner_Names(&runner)
}

// TEST_DIGIT_COUNT holds the digits of the widest count of files one fixture states.
const TEST_DIGIT_COUNT = 8

// Names the Go file of the index stated.
func file_name(index int) (name string) {
	digits := make(strconv.Buffer, TEST_DIGIT_COUNT)
	count := strconv.Format_Decimal_Into(digits, strconv.Machine_Integer(index))
	return "file" + string(digits[:count]) + ".go"
}

// States one directory of the count of Go files stated, each standing in every build.
func many_files(count int) (fixture directory_fixture) {
	for index := range count {
		fixture.Names = append(fixture.Names, file_name(index))
		fixture.Folders = append(fixture.Folders, false)
		fixture.Headers = append(fixture.Headers, "package one\n")
	}
	return fixture
}

func test_directories(t *testing.T) {
	fixture := directory_fixture{
		Names: []string{
			"build.go", "file_windows.go", "reader_linux.go", "writer_linux.go",
			"notes.txt", "sub", "closed_linux.go",
		},
		Folders: []bool{false, false, false, false, false, true, false},
		Refused: "closed_linux.go",
		Headers: []string{
			"package one\n",
			"package one\n",
			"//go:build amd64\n\npackage one\n",
			"//go:build arm64\n\npackage one\n",
			"",
			"",
			"package one\n",
		},
	}
	names := read_directory(t, &fixture, "one", 2)
	testify.Equal(t, 2, len(names), "the read keeps the files that stand in the build")
	testify.Equal(t, "build.go", string(names[0]),
		"a file of no constraint stands in the build")
	testify.Equal(t, "reader_linux.go", string(names[1]),
		"a file whose constraint holds stands in the build")
	testify.Equal(t, 4, fixture.Opened,
		"the read opens the files its names admit and no other")
	// A directory of no Go file hands back no name, one of a single file hands back one, and
	// one of every name the read admits hands back that count and no more.
	empty := directory_fixture{}
	testify.Equal(t, 0, len(read_directory(t, &empty, "", 0)),
		"a directory of no child hands back no name")
	single := many_files(1)
	testify.Equal(t, 1, len(read_directory(t, &single, "a", 1)),
		"a directory of one file hands back one name")
	pair := many_files(2)
	testify.Equal(t, 2, len(read_directory(t, &pair, "ab", 2)),
		"a directory of two files hands back two names")
	widest := many_files(TEST_ENTRY_COUNT)
	path := ""
	for range build.PATH_SIZE_MAXIMUM {
		path = path + "x"
	}
	// One read through storage of every width the package admits states those widths, thus
	// the widths a caller may state stand read as well as the names they hold.
	// One read of every width the package admits reaches the whole domain of its storage,
	// which is what a caller may hand it.
	for _, slots := range []int{0, 1, 2, build.ENTRY_COUNT_MAXIMUM} {
		for _, bytes := range []int{0, 1, 2, build.STORAGE_SIZE_MAXIMUM} {
			sized := many_files(2)
			read_sized(t, &sized, "one", slots, 1, bytes)
		}
	}
	testify.Equal(t, TEST_ENTRY_COUNT,
		len(read_directory(t, &widest, build.Path(path), TEST_ENTRY_COUNT)),
		"a directory of every name the read admits hands back that count")
}

func test_refusals(t *testing.T) {
	subject := new(build.Build)
	one := target("linux", "amd64", nil)
	for _, text := range []string{
		"", "   ", "&&", "linux &&", "(linux", "linux)", "!", "linux amd64",
		"(", ")", "||linux",
	} {
		held, ok := build.Holds_Constraint(subject, &one, token.Source(text))
		testify.False(t, bool(ok), "the reader refuses %q", text)
		testify.False(t, bool(held), "a constraint the reader refused holds nothing")
	}
}

func test_bounds(t *testing.T) {
	subject := new(build.Build)
	one := target("linux", "amd64", nil)
	wide := ""
	for range build.TEXT_SIZE_MAXIMUM {
		wide = wide + "x"
	}
	held, ok := build.Holds_Constraint(subject, &one, token.Source(wide+"x"))
	testify.False(t, bool(ok), "a constraint past the widest text is refused")
	testify.False(t, bool(held), "a constraint the reader refused holds nothing")
	deep := ""
	for range build.DEPTH_MAXIMUM + 1 {
		deep = deep + "("
	}
	deep = deep + "linux"
	for range build.DEPTH_MAXIMUM + 1 {
		deep = deep + ")"
	}
	held, ok = build.Holds_Constraint(subject, &one, token.Source(deep))
	testify.False(t, bool(ok), "a constraint past the deepest nest is refused")
	testify.False(t, bool(held), "a constraint the reader refused holds nothing")
	// A text of the widest size one source admits stands past the widest constraint, thus the
	// reader refuses it and every text of a few bytes states no constraint at all.
	widest := make([]byte, token.SOURCE_SIZE_MAXIMUM)
	for index := range widest {
		widest[index] = 'x'
	}
	held, ok = build.Holds_Constraint(subject, &one, token.Source(widest))
	testify.False(t, bool(ok), "a constraint of the widest source is refused")
	testify.False(t, bool(held), "a constraint the reader refused holds nothing")
	testify.True(t, bool(build.Names_System(subject, &one, token.Source(widest))),
		"a name of the widest source names no system")
	testify.True(t, bool(build.Names_System(subject, &one, token.Source(""))),
		"a name of no bytes names no system")
	held, ok = build.Holds_Constraint(subject, &one, token.Source(""))
	testify.False(t, bool(ok), "a constraint of no bytes states none")
	testify.False(t, bool(held), "a constraint the reader refused holds nothing")
	for _, text := range []string{"x", "xy"} {
		held, ok = build.Holds_Constraint(subject, &one, token.Source(text))
		testify.True(t, bool(ok), "a constraint of a few bytes states one tag")
		testify.False(t, bool(held), "a tag the target never names holds nothing")
		testify.True(t, bool(build.Names_System(subject, &one, token.Source(text))),
			"a name of a few bytes names no system")
	}
}

func test_allocation(t *testing.T) {
	fixture := allocation_fixture{}
	subject := new(build.Build)
	one := target("linux", "amd64", []string{"race"})
	text := token.Source("(linux || darwin) && amd64 && !race")
	name := token.Source("file_linux_amd64_test.go")
	checks := []struct {
		Name string
		Call func()
	}{
		{Name: "Holds_Constraint", Call: func() {
			fixture.Held, fixture.Ok = build.Holds_Constraint(subject, &one, text)
		}},
		{Name: "Names_System", Call: func() {
			fixture.Held = build.Names_System(subject, &one, name)
		}},
	}
	for _, check := range checks {
		t.Run(check.Name, func(t *testing.T) { testify.Zero_Allocation(t, check.Call) })
	}
	testify.True(t, bool(fixture.Held), "the allocation fixture names its build")
}
