package build_test

import "errors"
import "testing"

import "local/james-orcales/shared/std/go/build"
import "local/james-orcales/shared/std/go/token"
import "local/james-orcales/shared/std/sim/nbio"
import "local/james-orcales/shared/std/sim/time"
import "local/james-orcales/shared/std/strconv"
import "local/james-orcales/shared/std/testify"

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

// Test_Imports binds the Imports specification leaf before fixture declarations.
func Test_Imports(t *testing.T) {
  test_imports(t)
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
  Held    build.Boolean
  Ok      build.Read
  Rearm   build.Boolean
  Queued  build.Boolean
  Stopped build.Boolean
  Names   build.Name_Storage
  Error   error
}

// Builds one target of the system, the architecture, and the tags a case names.
func target(system string, architecture string, tags []string) (held build.Target) {
  held.Words.System = build.System(system)
  held.Words.Architecture = build.Architecture(architecture)
  held.Tags.Values = make(build.Target_Tags, build.TAG_COUNT_MAXIMUM)
  for index := range len(tags) {
    held.Tags.Values[index] = token.Source(tags[index])
  }
  held.Counts.Tag = build.Tag_Count(len(tags))
  return held
}

// Reports whether one file name stands in the target a case names.
func named(name string, system string, architecture string) (held build.Boolean) {
  one := target(system, architecture, nil)
  return build.Names_System(&one, token.Source(name))
}

// Reports whether one constraint holds in the target a case names.
func holds(t *testing.T, text string, one build.Target) (held build.Boolean) {
  t.Helper()
  subject := new(build.Build)
  result := build.Holds_Constraint(subject, &one, token.Source(text))
  held, ok := result.Held, result.OK
  testify.True(t, bool(ok), "the reader reads the constraint %q", text)
  return held
}

func test_build(t *testing.T) {
  subject := new(build.Build)
  one := target("linux", "amd64", []string{"race"})
  result := build.Holds_Constraint(subject, &one, token.Source("linux"))
  held, ok := result.Held, result.OK
  testify.True(t, bool(ok), "the reader reads one constraint")
  testify.True(t, bool(held), "the file stands in the build its system names")
  result = build.Holds_Constraint(subject, &one, token.Source("windows"))
  held, ok = result.Held, result.OK
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
    testify.Equal(
      t,
      one.Held,
      bool(named(one.Name, one.System, one.Architecture)),
      "%q stands in the build of %s and %s",
      one.Name,
      one.System,
      one.Architecture,
    )
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
    testify.Equal(
      t,
      held.Held,
      bool(holds(t, held.Text, one)),
      "the constraint %q holds",
      held.Text,
    )
  }
  windows := target("windows", "amd64", nil)
  testify.False(
    t,
    bool(holds(t, "unix", windows)),
    "the word unix holds on no system Go calls otherwise",
  )
}

func test_imports(t *testing.T) {
  // A path this module names and a path the standard library names stand in the build; a
  // path whose first element holds a dot names a host, thus a third party.
  for _, item := range []struct {
    Text string
    Held bool
    Ok   bool
  }{
    {"x", true, true},
    {"xy", true, true},
    {"package one\n", true, true},
    {"package one\n\nimport \"unsafe\"\n", true, true},
    {"package one\n\nimport \"local/james-orcales/shared/std/go/token\"\n", true, true},
    {"package one\n\nimport \"github.com/x/y\"\n", false, true},
    {"package one\n\nimport \"gopkg.in/yaml.v3\"\n", false, true},
    {"package one\n\nimport one \"github.com/x/y\"\n", false, true},
    {
      "package one\n\nimport (\n\t\"unsafe\"\n\n\t\"local/james-orcales/x\"\n)\n",
      true, true,
    },
    {"package one\n\nimport (\n\t\"unsafe\"\n\n\t\"github.com/x/y\"\n)\n", false, true},
    {"package one\n\nimport (\n\t\"unsafe\"\n", false, false},
    {"package one\n\nimport \"unsafe\n", false, false},
  } {
    result := build.Holds_Imports(token.Source(item.Text))
    held, ok := result.Held, result.OK
    testify.Equal(t, item.Held, bool(held), "the imports of %q hold", item.Text)
    testify.Equal(t, item.Ok, bool(ok), "the imports of %q read", item.Text)
  }
  // The widest text one source states reads as well as one head, which is the width a
  // caller may hand the read.
  widest := make([]byte, token.SOURCE_SIZE_MAXIMUM)
  for index := range widest {
    widest[index] = 'a'
  }
  result := build.Holds_Imports(token.Source(widest))
  held, ok := result.Held, result.OK
  testify.True(t, bool(held), "the widest text names no third party")
  testify.True(t, bool(ok), "the widest text reads")
  // A file the read of a directory meets states its imports the same way, thus a file
  // naming a third party stands outside the build the runner answers for.
  fixture := directory_fixture{
    Names:   []string{"first.go", "second.go"},
    Folders: []bool{false, false},
    Headers: []string{
      "package one\n\nimport \"local/james-orcales/x\"\n",
      "package one\n\nimport \"github.com/x/y\"\n",
    },
  }
  names := read_directory(t, &fixture, "one", 1)
  testify.Equal(t, 1, len(names), "a file naming a third party stands in no build")
  testify.Equal(
    t,
    "first.go",
    string(names[0]),
    "a file naming this module alone stands in the build",
  )
}

func test_systems(t *testing.T) {
  testify.True(
    t,
    bool(named("file_darwin.go", "darwin", "arm64")),
    "a known operating system names the build it stands in",
  )
  testify.True(
    t,
    bool(named("file_riscv64.go", "linux", "riscv64")),
    "a known architecture names the build it stands in",
  )
  // A word the list never names is a tag like any other, thus the name says nothing about
  // the system the file stands in.
  testify.True(
    t,
    bool(named("file_held.go", "linux", "amd64")),
    "a word neither list names says nothing",
  )
  testify.True(
    t,
    bool(named("file_unix.go", "linux", "amd64")),
    "the word unix names no system a file name states",
  )
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
  state nbio.State,
  completion *nbio.Completion,
  directory nbio.File,
  file_path string,
  options nbio.Open_At_Options,
  callback nbio.Callback,
) {
  fixture := state.(*directory_fixture)
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
  state nbio.State,
  completion *nbio.Completion,
  directory nbio.File,
  buffer []byte,
  entries []nbio.Directory_Entry,
  callback nbio.Callback,
) {
  fixture := state.(*directory_fixture)
  completion.Error = nil
  completion.Data = 0
  if fixture.Passes == 0 {
    for index := range len(fixture.Names) {
      if !(index < len(entries)) {
        break
      }
      entries[index] = nbio.Directory_Entry{
        Name:         fixture.Names[index],
        Is_Directory: fixture.Folders[index],
      }
      completion.Data = index + 1
    }
  }
  fixture.Passes = fixture.Passes + 1
  callback(completion)
}

// Reads the bytes the child of the fixture opens with into the buffer the runner owns.
func directory_read(
  state nbio.State,
  completion *nbio.Completion,
  file nbio.File,
  buffer []byte,
  offset int64,
  timeout time.Duration,
  callback nbio.Callback,
) {
  fixture := state.(*directory_fixture)
  completion.Error = nil
  completion.Data = 0
  slot := int(file) - 2
  if !(slot < 0) {
    if slot < len(fixture.Headers) {
      completion.Data = copy(buffer, fixture.Headers[slot])
    }
  }
  callback(completion)
}

// Closes one descriptor the read held.
func directory_close(
  state nbio.State,
  completion *nbio.Completion,
  file nbio.File,
  callback nbio.Callback,
) {
  completion.Error = nil
  completion.Data = 0
  callback(completion)
}

// Drives one directory read to its terminal state, which a composition root does with its loop.
func directory_drive(runner build.Directory_Runner_Handle, one build.Target_Handle) {
  for range 4096 {
    if bool(build.Directory_Runner_Stopped(&runner.Pass_Fields)) {
      return
    }
    if bool(build.Directory_Runner_Rearm(runner, one)) {
      continue
    }
    if bool(build.Directory_Runner_Work_Queued(&runner.Pass_Fields)) {
      continue
    }
    return
  }
}

// Reads one directory the fixture states and hands back the names the read kept.
func read_directory(
  t *testing.T,
  fixture *directory_fixture,
  path build.Path,
  names int,
) (held build.Name_Storage) {
  t.Helper()
  return read_sized(
    t,
    fixture,
    path,
    TEST_ENTRY_COUNT,
    TEST_RECORD_COUNT,
    TEST_BYTE_COUNT,
    target("linux", "amd64", nil),
  )
}

// Reads one directory through storage of the widths stated, which is how a read reaches every
// width the package admits.
func read_sized(
  t *testing.T,
  fixture *directory_fixture,
  path build.Path,
  slots int,
  records int,
  bytes int,
  one build.Target,
) (held build.Name_Storage) {
  t.Helper()
  // Both halves carry one backend, which is what one loop states.
  loop := nbio.IO{
    Storage: nbio.Storage{
      State:                           fixture,
      Open_At_Procedure:               directory_open,
      Get_Directory_Entries_Procedure: directory_entries,
      Read_Procedure:                  directory_read,
    },
    Network:         nbio.Network{State: fixture},
    Close_Procedure: directory_close,
  }
  runner := build.Directory_Runner{}
  build.Directory_Runner_Init(
    &runner,
    loop,
    path,
    build.Directory_Memory{
      Entries: make(build.Entry_Storage, slots),
      Records: make(build.Record_Storage, records),
      Names:   make(build.Name_Storage, slots),
      Bytes:   make(build.Name_Bytes, bytes),
      Header:  make(build.Header_Storage, bytes),
    },
  )
  directory_drive(&runner, &one)
  testify.True(
    t,
    bool(build.Directory_Runner_Stopped(&runner.Pass_Fields)),
    "the read of one directory stops",
  )
  testify.No_Error(
    t,
    build.Directory_Runner_Status(&runner.Pass_Fields),
    "the read states no error",
  )
  return build.Directory_Runner_Names(&runner.Pass_Fields)
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
      "build.go",
      "file_windows.go",
      "reader_linux.go",
      "writer_linux.go",
      "notes.txt",
      "sub",
      "closed_linux.go",
      "low.go",
      "next.go",
      "high.go",
    },
    Folders: []bool{
      false, false, false, false, false, true, false,
      false, false, false,
    },
    Refused: "closed_linux.go",
    Headers: []string{
      "package one\n",
      "package one\n",
      "//go:build ((amd64))\n\npackage one\n",
      "//go:build (arm64)\n\npackage one\n",
      "",
      "",
      "package one\n",
      // A constraint holding a byte no term spells states nothing the read
      // admits, thus the file it heads stands in no build at all.
      "//go:build linux \x01\n\npackage one\n",
      "//go:build linux \x02\n\npackage one\n",
      "//go:build linux \xff\n\npackage one\n",
    },
  }
  names := read_directory(t, &fixture, "one", 2)
  testify.Equal(t, 2, len(names), "the read keeps the files that stand in the build")
  testify.Equal(t, "build.go", string(names[0]),
    "a file of no constraint stands in the build")
  testify.Equal(
    t,
    "reader_linux.go",
    string(names[1]),
    "a file whose constraint holds stands in the build",
  )
  testify.Equal(t, 7, fixture.Opened,
    "the read opens the files its names admit and no other")
  // A directory of no Go file hands back no name, one of a single file hands back one, and
  // one of every name the read admits hands back that count and no more.
  empty := directory_fixture{}
  testify.Equal(
    t,
    0,
    len(read_directory(t, &empty, "", 0)),
    "a directory of no child hands back no name",
  )
  single := many_files(1)
  testify.Equal(
    t,
    1,
    len(read_directory(t, &single, "a", 1)),
    "a directory of one file hands back one name",
  )
  pair := many_files(2)
  testify.Equal(
    t,
    2,
    len(read_directory(t, &pair, "ab", 2)),
    "a directory of two files hands back two names",
  )
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
      read_sized(t, &sized, "one", slots, 1, bytes, target("linux", "amd64", nil))
    }
  }
  testify.Equal(
    t,
    TEST_ENTRY_COUNT,
    len(read_directory(t, &widest, build.Path(path), TEST_ENTRY_COUNT)),
    "a directory of every name the read admits hands back that count",
  )
}

func test_refusals(t *testing.T) {
  subject := new(build.Build)
  one := target("linux", "amd64", nil)
  for _, text := range []string{
    "",
    "   ",
    "&&",
    "linux &&",
    "(linux",
    "linux)",
    "!",
    "linux amd64",
    "(",
    ")",
    "||linux",
  } {
    result := build.Holds_Constraint(subject, &one, token.Source(text))
    held, ok := result.Held, result.OK
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
  result := build.Holds_Constraint(subject, &one, token.Source(wide+"x"))
  held, ok := result.Held, result.OK
  testify.False(t, bool(ok), "a constraint past the widest text is refused")
  testify.False(t, bool(held), "a constraint the reader refused holds nothing")
  widest_text_cases(t, subject, &one, wide)
}

// States the widest text each read admits, which is how a width a type declares stands
// observed and not merely stated.
func widest_text_cases(
  t *testing.T,
  subject build.Build_Handle,
  one build.Target_Handle,
  wide string,
) {
  // The widest constraint the read admits stands one byte inside the refusal, thus the text
  // the reader holds and every word it names reach that width and never pass it.
  result := build.Holds_Constraint(subject, one, token.Source(wide))
  testify.True(t, bool(result.OK), "a constraint of the widest text is read")
  testify.False(t, bool(result.Held),
    "a tag no target names holds the file out of the build")
  // No system and no architecture spells a word as wide as the widest constraint, thus a
  // name whose stem passes that width names neither and stands in every build.
  testify.True(
    t,
    bool(build.Names_System(one, token.Source(wide+"x.go"))),
    "a name whose stem passes the widest constraint stands in every build",
  )
  // Every place a read names stands in the constraint text, thus none passes the width that
  // text states. A stem holding no underscore opens its field one byte past the text.
  stem := wide[:build.TEXT_SIZE_MAXIMUM-3]
  testify.True(
    t,
    bool(build.Names_System(one, token.Source(stem+".go"))),
    "a stem no underscore divides names no system",
  )
  // A stem holding no underscore names no field at all, thus the read answers at the
  // opening and the field stands where the read before it left it.
  divided := wide[:build.TEXT_SIZE_MAXIMUM-6] + "_amd64"
  testify.True(
    t,
    bool(build.Names_System(one, token.Source(divided+".go"))),
    "a stem the widest constraint holds divides into fields",
  )
  deep := ""
  for range build.DEPTH_MAXIMUM + 1 {
    deep = deep + "("
  }
  deep = deep + "linux"
  for range build.DEPTH_MAXIMUM + 1 {
    deep = deep + ")"
  }
  result = build.Holds_Constraint(subject, one, token.Source(deep))
  held, ok := result.Held, result.OK
  testify.False(t, bool(ok), "a constraint past the deepest nest is refused")
  testify.False(t, bool(held), "a constraint the reader refused holds nothing")
  wide_word_case(t, subject, wide)
  nested_depth_cases(t, subject, one)
  // A text of the widest size one source admits stands past the widest constraint, thus the
  // reader refuses it and every text of a few bytes states no constraint at all.
  widest := make([]byte, token.SOURCE_SIZE_MAXIMUM)
  for index := range widest {
    widest[index] = 'x'
  }
  result = build.Holds_Constraint(subject, one, token.Source(widest))
  held, ok = result.Held, result.OK
  testify.False(t, bool(ok), "a constraint of the widest source is refused")
  testify.False(t, bool(held), "a constraint the reader refused holds nothing")
  testify.True(
    t,
    bool(build.Names_System(one, token.Source(widest))),
    "a name of the widest source names no system",
  )
  testify.True(
    t,
    bool(build.Names_System(one, token.Source(""))),
    "a name of no bytes names no system",
  )
  result = build.Holds_Constraint(subject, one, token.Source(""))
  held, ok = result.Held, result.OK
  testify.False(t, bool(ok), "a constraint of no bytes states none")
  testify.False(t, bool(held), "a constraint the reader refused holds nothing")
  for _, text := range []string{"x", "xy"} {
    result = build.Holds_Constraint(subject, one, token.Source(text))
    held, ok = result.Held, result.OK
    testify.True(t, bool(ok), "a constraint of a few bytes states one tag")
    testify.False(t, bool(held), "a tag the target never names holds nothing")
    testify.True(
      t,
      bool(build.Names_System(one, token.Source(text))),
      "a name of a few bytes names no system",
    )
  }
}

func test_allocation(t *testing.T) {
  allocation_read_checks(t)
  allocation_directory_checks(t)
}

// Measures every text reader apart from stateful directory steps.
func allocation_constraint(
  fixture *allocation_fixture,
  subject build.Build_Handle,
  one build.Target_Handle,
  text token.Source,
) {
  result := build.Holds_Constraint(subject, one, text)
  fixture.Held, fixture.Ok = result.Held, result.OK
}

func allocation_imports(fixture *allocation_fixture, imports token.Source) {
  result := build.Holds_Imports(imports)
  fixture.Held, fixture.Ok = result.Held, result.OK
}

func filler(width_size int) (text string) {
  held := make([]byte, width_size)
  for index := range held {
    held[index] = 'x'
  }
  return string(held)
}

func allocation_read_checks(t *testing.T) {
  fixture := allocation_fixture{}
  subject := new(build.Build)
  one := target("linux", "amd64", []string{"race"})
  text := token.Source("(linux || darwin) && amd64 && !race")
  name := token.Source("file_linux_amd64_test.go")
  imports := token.Source("package one\n\nimport \"local/one\"\n")
  checks := []struct {
    Name string
    Call func()
  }{
    {Name: "Holds_Constraint", Call: func() {
      allocation_constraint(&fixture, subject, &one, text)
    }},
    {Name: "Names_System", Call: func() { allocation_names_system(&fixture, &one, name) }},
    {Name: "Holds_Imports", Call: func() {
      allocation_imports(&fixture, imports)
    }},
  }
  for _, check := range checks {
    t.Run(check.Name, func(t *testing.T) { testify.Zero_Allocation(t, check.Call) })
  }
  testify.True(t, bool(fixture.Held), "the allocation fixture names its build")
}

// Measures every directory operation and one complete nonempty read.
func allocation_directory_checks(t *testing.T) {
  fixture := allocation_fixture{}
  one := target("linux", "amd64", nil)
  directory := directory_fixture{
    Names:   []string{"file.go"},
    Folders: []bool{false},
    Headers: []string{"package one\n"},
  }
  loop := nbio.IO{
    Storage: nbio.Storage{
      State:                           &directory,
      Open_At_Procedure:               directory_open,
      Get_Directory_Entries_Procedure: directory_entries,
      Read_Procedure:                  directory_read,
    },
    Network:         nbio.Network{State: &directory},
    Close_Procedure: directory_close,
  }
  memory := build.Directory_Memory{
    Entries: make(build.Entry_Storage, TEST_ENTRY_COUNT),
    Records: make(build.Record_Storage, TEST_RECORD_COUNT),
    Names:   make(build.Name_Storage, TEST_ENTRY_COUNT),
    Bytes:   make(build.Name_Bytes, TEST_BYTE_COUNT),
    Header:  make(build.Header_Storage, TEST_HEADER_COUNT),
  }
  runner := build.Directory_Runner{}
  stopped := build.Directory_Runner{}
  build.Directory_Runner_Init(&stopped, loop, "one", memory)
  stopped.Flags.Stopped = true
  checks := []struct {
    Name string
    Call func()
  }{
    {Name: "Directory_Runner_Init", Call: func() {
      allocation_runner_init(&runner, loop, memory)
    }},
    {Name: "Directory_Runner_Rearm", Call: func() {
      allocation_runner_rearm(&fixture, &runner, &one)
    }},
    {Name: "Directory_Runner_Work_Queued", Call: func() {
      allocation_runner_work_queued(&fixture, &runner)
    }},
    {Name: "Directory_Runner_Stopped", Call: func() {
      allocation_runner_stopped(&fixture, &stopped)
    }},
    {Name: "Directory_Runner_Status", Call: func() {
      allocation_runner_status(&fixture, &stopped)
    }},
    {Name: "Directory_Runner_Names", Call: func() {
      allocation_runner_names(&fixture, &stopped)
    }},
    {Name: "Directory_Runner_Drive", Call: func() {
      allocation_runner_drive(&fixture, &directory, &runner, loop, memory, &one)
    }},
  }
  for _, check := range checks {
    t.Run(check.Name, func(t *testing.T) { testify.Zero_Allocation(t, check.Call) })
  }
}

func allocation_names_system(
  fixture *allocation_fixture,
  one build.Target_Handle,
  name token.Source,
) {
  fixture.Held = build.Names_System(one, name)
}

func allocation_runner_init(
  runner build.Directory_Runner_Handle,
  loop nbio.IO,
  memory build.Directory_Memory,
) {
  *runner = build.Directory_Runner{}
  build.Directory_Runner_Init(runner, loop, "one", memory)
}

func allocation_runner_rearm(
  fixture *allocation_fixture,
  runner build.Directory_Runner_Handle,
  one build.Target_Handle,
) {
  fixture.Rearm = build.Directory_Runner_Rearm(runner, one)
}

func allocation_runner_work_queued(
  fixture *allocation_fixture,
  runner build.Directory_Runner_Handle,
) {
  fixture.Queued = build.Directory_Runner_Work_Queued(&runner.Pass_Fields)
}

func allocation_runner_stopped(
  fixture *allocation_fixture,
  runner build.Directory_Runner_Handle,
) {
  fixture.Stopped = build.Directory_Runner_Stopped(&runner.Pass_Fields)
}

func allocation_runner_status(
  fixture *allocation_fixture,
  runner build.Directory_Runner_Handle,
) {
  fixture.Error = build.Directory_Runner_Status(&runner.Pass_Fields)
}

func allocation_runner_names(
  fixture *allocation_fixture,
  runner build.Directory_Runner_Handle,
) {
  fixture.Names = build.Directory_Runner_Names(&runner.Pass_Fields)
}

func allocation_runner_drive(
  fixture *allocation_fixture,
  directory *directory_fixture,
  runner build.Directory_Runner_Handle,
  loop nbio.IO,
  memory build.Directory_Memory,
  one build.Target_Handle,
) {
  directory.Passes = 0
  directory.Opened = 0
  allocation_runner_init(runner, loop, memory)
  directory_drive(runner, one)
  fixture.Names = build.Directory_Runner_Names(&runner.Pass_Fields)
}

// States each depth the read admits, which is how every depth the type declares stands observed
// and not merely stated.
func nested_depth_cases(t *testing.T, subject build.Build_Handle, one build.Target_Handle) {
  // The deepest nest the read admits stands one bracket inside the refusal, and a nest of
  // one bracket and of two stand inside it, thus every depth the type states is read.
  for _, depth := range []int{1, 2, build.DEPTH_MAXIMUM} {
    nested := ""
    for range depth {
      nested = nested + "("
    }
    nested = nested + "linux"
    for range depth {
      nested = nested + ")"
    }
    result := build.Holds_Constraint(subject, one, token.Source(nested))
    testify.True(t, bool(result.OK), "a nest of %d brackets is read", depth)
    testify.True(t, bool(result.Held),
      "a nest of %d brackets holds the system it names", depth)
  }
}

// States a target word wider than the widest constraint, which no tag of a constraint matches.
func wide_word_case(t *testing.T, subject build.Build_Handle, wide string) {
  stretched := target(wide+"x", "amd64", nil)
  result := build.Holds_Constraint(subject, &stretched, token.Source("linux"))
  testify.True(t, bool(result.OK), "a target of a wide word reads a constraint")
  testify.False(t, bool(result.Held),
    "a target word wider than the widest constraint names no tag")
}

// States the width of each line prefix the reader searches for. Four widths stand and no other,
// thus a prefix of one byte or of two names nothing the reader looks for.
func Test_Head_Widths(t *testing.T) {
  testify.Equal(
    t,
    build.IMPORT_HEAD_SIZE,
    len(build.IMPORT_HEAD),
    "the import prefix states its width",
  )
  testify.Equal(
    t,
    build.PACKAGE_HEAD_SIZE,
    len(build.PACKAGE_HEAD),
    "the package prefix states its width",
  )
  testify.Equal(
    t,
    build.PACKAGE_HEAD_SIZE,
    len(build.IMPORT_OPENING),
    "the opening of an import block stands as wide as the package prefix",
  )
  testify.Equal(
    t,
    build.HEAD_SIZE_MAXIMUM,
    len(build.BUILD_HEAD),
    "the build prefix is the widest the reader searches for",
  )
}

// States a target word of each width the type admits, which is how a width it declares stands
// observed and not merely stated. A word of one byte and of two names a system as well as a
// longer one does, and the widest a source admits names none.
func Test_Target_Word_Widths(t *testing.T) {
  subject := new(build.Build)
  widest := make([]byte, token.SOURCE_SIZE_MAXIMUM)
  for index := range widest {
    widest[index] = 'x'
  }
  counts := []int{1, 0, 2, build.TAG_COUNT_MAXIMUM}
  for index, word := range []string{"", "a", "ab", string(widest)} {
    tags := make([]string, counts[index])
    for slot := range tags {
      tags[slot] = word
    }
    one := target(word, word, tags)
    result := build.Holds_Constraint(subject, &one, token.Source("linux"))
    testify.True(t, bool(result.OK), "a target of a word of %d bytes reads", len(word))
    testify.False(
      t,
      bool(result.Held),
      "a word of %d bytes names no system the constraint states",
      len(word),
    )
    result = build.Holds_Constraint(subject, &one, token.Source("(linux)"))
    testify.True(t, bool(result.OK), "a group of one tag reads")
    testify.False(t, bool(result.Held),
      "a group of a system the target never names holds nothing")
    testify.True(
      t,
      bool(build.Names_System(&one, token.Source("a.go"))),
      "a name stating no system stands in every build",
    )
    testify.False(
      t,
      bool(build.Names_System(&one, token.Source("a_linux.go"))),
      "a name stating a system the target never names stands outside the build",
    )
    fixture := directory_fixture{
      Names:   []string{"a.go", "b.go"},
      Folders: []bool{false, false},
      Headers: []string{"//go:build (linux)\n\npackage one\n", "package one\n"},
    }
    names := read_sized(
      t,
      &fixture,
      "one",
      TEST_ENTRY_COUNT,
      TEST_RECORD_COUNT,
      TEST_BYTE_COUNT,
      one,
    )
    testify.Equal(
      t,
      1,
      len(names),
      "a directory read keeps the file whose constraint names no system",
    )
  }
}

// Drives one name scan over each width the constraint text admits, which is how every width
// the two places it stands between stands observed.
func Test_Name_Place_Widths(t *testing.T) {
  one := target("linux", "amd64", nil)
  wide := ""
  for range build.TEXT_SIZE_MAXIMUM {
    wide = wide + "x"
  }
  // A stem holding no underscore opens at its end, thus a stem of the widest text opens at
  // that width.
  testify.True(
    t,
    bool(build.Names_System(&one, token.Source(wide+".go"))),
    "a stem of the widest text names no system",
  )
  // A field opens behind the last underscore, thus a stem closing at one opens its field at
  // the width of that stem.
  divider := wide[:build.TEXT_SIZE_MAXIMUM-1] + "_"
  testify.True(
    t,
    bool(build.Names_System(&one, token.Source(divider+".go"))),
    "a stem closing at an underscore names no system",
  )
  // One byte and two stand at the other end of the same widths.
  testify.True(
    t,
    bool(build.Names_System(&one, token.Source("a_.go"))),
    "a stem of one letter and an underscore names no system",
  )
}

// States each byte one constraint can hold, from the first to the last. A read stands on the
// byte it reached, thus every byte the type admits stands on that slot at some read.
func Test_Held_Symbol_Widths(t *testing.T) {
  subject := new(build.Build)
  one := target("linux", "amd64", nil)
  // A term opens at whatever byte the author wrote there, thus the read of a tag stands on
  // every byte a constraint can state, from none to the widest.
  for _, symbol := range []byte{0, 1, 2, 255} {
    for _, text := range []string{
      string([]byte{symbol}), "linux && " + string([]byte{symbol}),
    } {
      result := build.Holds_Constraint(subject, &one, token.Source(text))
      testify.False(t, bool(result.OK),
        "a term opening at the byte %d states no tag", symbol)
      testify.False(t, bool(result.Held),
        "a constraint the reader refused holds nothing")
    }
  }
}

// States a header whose build line and whose import path both pass the widest constraint. A
// header states whatever the author wrote, thus a read of one stands on every width a source
// holds and refuses the ones no constraint states.
func Test_Header_Widths(t *testing.T) {
  // The widest run of imports is one path filling the whole source behind the shortest
  // clause, thus the line and the path it holds reach the widest width each admits.
  for _, item := range []struct {
    Text string
    Held bool
    Ok   bool
  }{
    {"package \nimport \"\"", true, true},
    {"package \nimport \"a\"", true, true},
    {"package \nimport \"ab\"", true, true},
    {"package \nimport (\nx\n)", false, false},
    {"package \nimport (\nxy\n)", false, false},
    {"package \nimport \"" + filler(build.IMPORT_PATH_SIZE_MAXIMUM) + "\"", true, true},
    {"package \nimport \"" + filler(build.TEXT_SIZE_MAXIMUM+1) + "\"\n", true, true},
  } {
    result := build.Holds_Imports(token.Source(item.Text))
    testify.Equal(t, item.Held, bool(result.Held), "the imports of %d bytes hold",
      len(item.Text))
    testify.Equal(t, item.Ok, bool(result.OK), "the imports of %d bytes read",
      len(item.Text))
  }
  // A header states its constraint as wide as the author wrote it, thus the read of one
  // stands on a constraint of none, one, two, and every byte the source holds.
  one := target("linux", "amd64", nil)
  for _, header := range []string{
    "x",
    "xy",
    "//go:build \n",
    "//go:build a\n",
    "//go:build ab\n",
    "//go:build " + filler(build.TEXT_SIZE_MAXIMUM+1) + "\n",
    "//go:build " + filler(build.BUILD_LINE_SIZE_MAXIMUM),
  } {
    fixture := directory_fixture{
      Names:   []string{"a.go"},
      Folders: []bool{false},
      Headers: []string{header},
    }
    names := read_sized(
      t,
      &fixture,
      "one",
      TEST_ENTRY_COUNT,
      TEST_RECORD_COUNT,
      token.SOURCE_SIZE_MAXIMUM,
      one,
    )
    testify.True(t, !(len(names) > 1), "a header of %d bytes holds one file or none",
      len(header))
  }
}

// States each edge a read of a constraint and a read of a name stand on: a reader that a refused
// read left inside two groups, a constraint closing at a bracket that opens its widest place, a
// field of every width a name states, and a system word of every width the word of Unix meets.
func Test_Read_Edges(t *testing.T) {
  subject := new(build.Build)
  one := target("linux", "amd64", nil)
  testify.False(
    t,
    bool(build.Holds_Constraint(subject, &one, token.Source("((")).OK),
    "two brackets that never close state no constraint",
  )
  testify.True(
    t,
    bool(build.Holds_Constraint(subject, &one, token.Source("linux")).OK),
    "a reader a refused read left inside two groups reads the next constraint afresh",
  )
  wide := ""
  for range build.TEXT_SIZE_MAXIMUM {
    wide = wide + "x"
  }
  result := build.Holds_Constraint(subject, &one, token.Source(wide[:len(wide)-5]+" && ("))
  testify.False(t, bool(result.OK), "a bracket closing the widest constraint states no term")
  for _, name := range []string{
    "x_a.go",
    "x_ab.go",
    "_" + wide[:build.NAME_FIELD_SIZE_MAXIMUM] + ".go",
  } {
    testify.True(
      t,
      bool(build.Names_System(&one, token.Source(name))),
      "a name of %d bytes stating no system stands in every build",
      len(name),
    )
  }
  for _, word := range []string{"", "a", "ab", wide} {
    unix := target(word, "amd64", nil)
    result = build.Holds_Constraint(subject, &unix, token.Source("unix"))
    testify.True(t, bool(result.OK), "the word unix reads")
    testify.False(t, bool(result.Held),
      "a system of %d bytes Go never calls a Unix", len(word))
  }
  result = build.Holds_Imports(token.Source("package one\n// one\nimport \"unsafe\"\n"))
  testify.True(t, bool(result.OK), "a comment between the clause and the imports reads")
  testify.True(t, bool(result.Held), "a comment names no third party")
}
