package harness_test

import "runtime"
import "testing"
import "unsafe"

import "local/james-orcales/harness/pkg"
import "local/james-orcales/harness/pkg/configuration"
import "local/james-orcales/harness/pkg/naming"
import "local/james-orcales/harness/pkg/requirements"
import "local/james-orcales/shared/std/cli"
import "local/james-orcales/shared/std/filepath"
import "local/james-orcales/shared/std/go/ast"
import "local/james-orcales/shared/std/go/format"
import "local/james-orcales/shared/std/go/printer"
import "local/james-orcales/shared/std/go/token"
import "local/james-orcales/shared/std/sim/nbio"
import "local/james-orcales/shared/std/sim/prng"
import "local/james-orcales/shared/std/sim/time"
import "local/james-orcales/shared/std/strings"
import "local/james-orcales/shared/std/testify"

// Test_Diagnostics_Tier_One verifies an auto-fixable diagnostic carries its fix.
func Test_Diagnostics_Tier_One(t *testing.T) {
  runner := checked_new(UNFORMATTED)
  testify.Equal(t, harness.TIER_ONE, runner.Diagnostic.Tier)
  testify.Equal(t, CLEAN, string(runner.Diagnostic.Fix))
}

// Test_Diagnostics_Tier_Two verifies a diagnostic that is not auto-fixable carries no fix.
func Test_Diagnostics_Tier_Two(t *testing.T) {
  runner := checked_new(REFUSED)
  testify.Equal(t, harness.TIER_TWO, runner.Diagnostic.Tier)
  testify.Equal(t, 0, len(runner.Diagnostic.Fix))
}

// Test_Diagnostics_Every_Finding verifies a file whose form the check accepts states each finding
// of each rule, in line and column order, and a refused file states each refusal its parse logged.
func Test_Diagnostics_Every_Finding(t *testing.T) {
  memory, format_state, command, directory := memory_new()
  subject := fixture_new(t, memory, format_state, command, directory, 1)
  fixture_file(t, subject, harness.CONFIGURATION_PATH, shared_configuration())
  fixture_directory(t, subject, "shared")
  fixture_package_directory(t, subject, "shared/a")
  source := "package a\n\nimport \"strings\"\n\nvar util = 1\n\nfunc F() {\n  F()\n}\n\n" +
    "var util_table = 1\n"
  fixture_file(t, subject, "shared/a/a.go", source)
  testify.True(t, fixture_run(subject, []string{"harness", "shared"}))
  testify.Equal(
    t,
    "shared/a/a.go:3:8: Banned import \"strings\" belongs to stdlib family \"strings\". "+
      "Use shared package.\n"+
      "shared/a/a.go:5:5: The identifier \"util\" contains the banned substring \"util\". "+
      "Write a different name.\n"+
      "shared/a/a.go:8:3: The function F recurses. Write a loop instead.\n"+
      "shared/a/a.go:11:5: The identifier \"util_table\" contains the banned substring "+
      "\"util\". Write a different name.\n",
    fixture_output(subject),
  )
  // A file the parser refuses states each refusal its parse logged, in line and column order.
  refused := fixture_new(t, memory, format_state, command, directory, 1)
  fixture_file(t, refused, harness.CONFIGURATION_PATH, shared_configuration())
  fixture_directory(t, refused, "shared")
  fixture_package_directory(t, refused, "shared/a")
  fixture_file(t, refused, "shared/a/a.go",
    "package a\n\nconst b = iota\n\nvar c = d && e\n")
  testify.True(t, fixture_run(refused, []string{"harness", "shared"}))
  testify.Equal(
    t,
    "shared/a/a.go:3:7: "+
      string(ast.Failure_Message(ast.FAILURE_CONSTANT_CASE))+
      "\n"+
      "shared/a/a.go:3:11: "+
      string(ast.Failure_Message(ast.FAILURE_IOTA))+
      "\n"+
      "shared/a/a.go:5:11: "+
      string(ast.Failure_Message(ast.FAILURE_REFUSED_SIGN))+
      "\n",
    fixture_output(refused),
  )
}

// Test_Command_Usage verifies a command line shared/std/cli refuses fails with its error, and -h
// writes the help.
func Test_Command_Usage(t *testing.T) {
  memory, format_state, command, directory := memory_new()
  cases := [][]string{{"harness"}, {"harness", "a", "b"}, {"harness", "a", "-jsn"}}
  for _, arguments := range cases {
    subject := fixture_new(t, memory, format_state, command, directory, 1)
    testify.True(t, fixture_run(subject, arguments))
    testify.Equal(t, harness.EXIT_FAILURE, subject.Runner.Status)
    testify.Equal(t, "", fixture_output(subject))
    errors := fixture_errors(subject)
    testify.True(t, bool(strings.Has_Prefix(strings.Text(errors), "harness: ")), errors)
    testify.True(t, bool(strings.Has_Suffix(strings.Text(errors), "\n")), errors)
    testify.True(t, len(errors) > len("harness: \n"), errors)
  }
  subject := fixture_new(t, memory, format_state, command, directory, 1)
  testify.True(t, fixture_run(subject, []string{"harness", "-h"}))
  testify.Equal(t, harness.EXIT_SUCCESS, subject.Runner.Status)
  help := fixture_output(subject)
  testify.True(t, bool(strings.Contains(strings.Text(help), "<directory: string>")), help)
  testify.True(t, bool(strings.Contains(strings.Text(help), "json")), help)
  testify.Equal(t, "", fixture_errors(subject))
}

// Test_Command_Vector verifies an argument vector of no argument at all is refused, rather than
// handed to a parse that states a program name as a precondition.
func Test_Command_Vector(t *testing.T) {
  memory, format_state, command, directory := memory_new()
  subject := fixture_new(t, memory, format_state, command, directory, 1)
  testify.True(t, fixture_run(subject, []string{}))
  testify.Equal(t, harness.EXIT_FAILURE, subject.Runner.Status)
  testify.Equal(t, "", fixture_output(subject))
  testify.Equal(
    t, "harness: the argument vector names no program\n", fixture_errors(subject),
  )
}

// Test_Command_Configuration verifies a run reads harness.json before the walk: a file that does
// not decode states its fault and the byte it stands on, and an absent file fails step open.
func Test_Command_Configuration(t *testing.T) {
  memory, format_state, command, directory := memory_new()
  subject := fixture_new(t, memory, format_state, command, directory, 1)
  fixture_package_directory(t, subject, "package")
  fixture_file(t, subject, "package/a.go", UNFORMATTED)
  fixture_file(t, subject, harness.CONFIGURATION_PATH, "{}")
  testify.True(t, fixture_run(subject, []string{"harness", "package"}))
  testify.Equal(t, harness.EXIT_FAILURE, subject.Runner.Status)
  testify.Equal(t, "", fixture_output(subject))
  testify.Equal(
    t,
    "harness: harness.json: shared_component is absent\n",
    fixture_errors(subject),
  )
  subject = fixture_new(t, memory, format_state, command, directory, 1)
  fixture_package_directory(t, subject, "package")
  globbed := CONFIGURATION[:len(CONFIGURATION)-len(`"ignore":[],"word_replacements":{"util":[]}}`)]
  globbed += `"ignore":["a?"],"word_replacements":{"util":[]}}`
  fixture_file(t, subject, harness.CONFIGURATION_PATH, globbed)
  testify.True(t, fixture_run(subject, []string{"harness", "package"}))
  testify.Equal(
    t,
    "harness: harness.json: ignore entry holds ? or [; write * or **: \"a?\"\n",
    fixture_errors(subject),
  )
  // A source that is no JSON value names the byte the decoder stopped on, counted from one,
  // and an unknown key quotes itself.
  sources := [...]string{`{"shared_component":`, `{"a\u0001":1}`}
  lines := [...]string{
    "harness: harness.json: invalid JSON at byte 21\n",
    "harness: harness.json: unknown key: \"a?\"\n",
  }
  for index, source := range sources {
    subject = fixture_new(t, memory, format_state, command, directory, 1)
    fixture_package_directory(t, subject, "package")
    fixture_file(t, subject, harness.CONFIGURATION_PATH, source)
    testify.True(t, fixture_run(subject, []string{"harness", "package"}))
    testify.Equal(t, lines[index], fixture_errors(subject))
  }
  subject = fixture_new(t, memory, format_state, command, directory, 1)
  fixture_package_directory(t, subject, "package")
  fixture_remove(t, subject, harness.CONFIGURATION_PATH)
  testify.True(t, fixture_run(subject, []string{"harness", "package"}))
  testify.Equal(t, harness.EXIT_FAILURE, subject.Runner.Status)
  testify.Equal(t, "harness: failed to open: harness.json\n", fixture_errors(subject))
}

// Test_Command_Configuration_Paths verifies each wildcard-free entry states a directory with a
// trailing slash and a file without, a package list names no file, and an entry naming nothing
// is a finding.
func Test_Command_Configuration_Paths(t *testing.T) {
  memory, format_state, command, directory := memory_new()
  subject := fixture_new(t, memory, format_state, command, directory, 1)
  fixture_directory(t, subject, "package")
  fixture_file(t, subject, "package/a.go", "package main\n\nfunc main() {}\n")
  source := configuration_with(
    CONFIGURATION,
    "ignore",
    `["package","package/a.go/","nowhere","package/a.go","package/","*"]`,
  )
  source = configuration_with(source, "instrumentation_packages", `["package/a.go"]`)
  source = configuration_with(source, "opt_out_recursion_ban", `["!/package/","package/a.go"]`)
  source = configuration_with(source, "opt_out_interface_ban", `["/package"]`)
  fixture_file(t, subject, harness.CONFIGURATION_PATH, source)
  testify.True(t, fixture_run(subject, []string{"harness", "package"}))
  testify.Equal(t, harness.EXIT_FAILURE, subject.Runner.Status)
  testify.Equal(
    t,
    "harness.json: The ignore entry \"package\" names a directory. Add a trailing slash.\n"+
      "harness.json: The ignore entry \"package/a.go/\" names a file. "+
      "Remove the trailing slash.\n"+
      "harness.json: The ignore entry \"nowhere\" matches no file and no directory.\n"+
      "harness.json: The instrumentation_packages entry \"package/a.go\" names a file. "+
      "Only ignore and opt_out_recursion_ban can name a file.\n"+
      "harness.json: The opt_out_interface_ban entry \"/package\" names a directory. "+
      "Add a trailing slash.\n",
    fixture_output(subject),
  )
  testify.True(t, fixture_run(subject, []string{"harness", "package", "-json"}))
  testify.True(
    t,
    bool(
      strings.Contains(
        strings.Text(fixture_output(subject)),
        `{"path":"harness.json","line":0,"column":0,"tier":2,"rule":"slashes",`,
      ),
    ),
  )
}

// Test_Command_Ignore verifies an ignored directory and an ignored file state nothing, and the
// files beside them still do.
func Test_Command_Ignore(t *testing.T) {
  memory, format_state, command, directory := memory_new()
  subject := fixture_new(t, memory, format_state, command, directory, 1)
  fixture_file(t, subject, harness.CONFIGURATION_PATH, shared_configuration())
  fixture_directory(t, subject, "shared")
  fixture_package_directory(t, subject, "shared/a")
  fixture_directory(t, subject, "shared/b")
  ignored := strings_replace(
    shared_configuration(),
    `"ignore":[]`,
    `"ignore":["shared/b/**","shared/a/c.go"]`,
  )
  fixture_file(t, subject, harness.CONFIGURATION_PATH, ignored)
  fixture_file(t, subject, "shared/a/a.go", "package a\n\nfunc F() {\n  F()\n}\n")
  fixture_file(t, subject, "shared/a/c.go", "package a\n\nfunc G() {\n  G()\n}\n")
  fixture_file(t, subject, "shared/b/b.go", UNFORMATTED)
  testify.True(t, fixture_run(subject, []string{"harness", "shared"}))
  testify.Equal(t, harness.EXIT_FAILURE, subject.Runner.Status)
  testify.Equal(
    t,
    "shared/a/a.go:4:3: The function F recurses. Write a loop instead.\n",
    fixture_output(subject),
  )
}

// Test_Command_Argument verifies a trailing separator names the same directory as none, and an
// argument of no bytes fails the walk.
func Test_Command_Argument(t *testing.T) {
  memory, format_state, command, directory := memory_new()
  subject := fixture_new(t, memory, format_state, command, directory, 1)
  fixture_package_directory(t, subject, "package")
  fixture_file(t, subject, "package/a.go", UNFORMATTED)
  testify.True(t, fixture_run(subject, []string{"harness", "package/"}))
  testify.Equal(t, "package/a.go: not formatted\n", fixture_output(subject))
  testify.Equal(t, "", fixture_errors(subject))

  subject = fixture_new(t, memory, format_state, command, directory, 1)
  testify.True(t, fixture_run(subject, []string{"harness", ""}))
  testify.Equal(t, harness.EXIT_FAILURE, subject.Runner.Status)
  testify.Equal(
    t,
    "harness: failed to walk: the argument names no directory\n",
    fixture_errors(subject),
  )

  subject = fixture_new(t, memory, format_state, command, directory, 1)
  testify.True(t, fixture_run(subject, []string{"harness", "we\nird"}))
  testify.Equal(t, harness.EXIT_FAILURE, subject.Runner.Status)
  testify.Equal(
    t,
    "harness: failed to walk: the argument names no directory\n",
    fixture_errors(subject),
  )
}

// Test_Command_Entries verifies an entry whose name holds a byte below a space fails the walk.
func Test_Command_Entries(t *testing.T) {
  memory, format_state, command, directory := memory_new()
  subject := fixture_new(t, memory, format_state, command, directory, 1)
  fixture_package_directory(t, subject, "package")
  fixture_file(t, subject, "package/a\nb.go", CLEAN)
  testify.True(t, fixture_run(subject, []string{"harness", "package"}))
  testify.Equal(t, harness.EXIT_FAILURE, subject.Runner.Status)
  testify.Equal(
    t,
    "harness: failed to walk: package: a name holds a byte below a space\n",
    fixture_errors(subject),
  )
}

// Test_Command_Symlinks verifies a link whose target resolves to nothing is flagged, whether the
// target is absent or the links loop, and a link that resolves is not.
func Test_Command_Symlinks(t *testing.T) {
  memory, format_state, command, directory := memory_new()
  subject := fixture_new(t, memory, format_state, command, directory, 1)
  fixture_directory(t, subject, "package")
  fixture_file(t, subject, "package/m.go", "package main\n\nfunc main() {}\n")
  fixture_link(t, subject, "package/gone.go", "missing.go")
  fixture_link(t, subject, "package/loop", "loop")
  fixture_link(t, subject, "package/kept", "m.go")
  testify.True(t, fixture_run(subject, []string{"harness", "package"}))
  testify.Equal(t, harness.EXIT_FAILURE, subject.Runner.Status)
  output := fixture_output(subject)
  for _, line := range []string{
    "package/gone.go: The symlink's target is absent. Remove the symlink or restore its target.\n",
    "package/loop: The symlink's target is absent. Remove the symlink or restore its target.\n",
  } {
    testify.True(t, bool(strings.Contains(strings.Text(output), strings.Text(line))), output)
  }
  testify.False(t, bool(strings.Contains(strings.Text(output), "package/kept")), output)
}

// Test_Command_Recursive verifies every Go file at any depth is checked, and nothing else.
func Test_Command_Recursive(t *testing.T) {
  memory, format_state, command, directory := memory_new()
  for seed := uint64(1); seed < SEED_COUNT+1; seed++ {
    subject := fixture_new(t, memory, format_state, command, directory, seed)
    fixture_package(t, subject)
    testify.True(t, fixture_run(subject, []string{"harness", "package"}))
    testify.Equal(t, harness.EXIT_FAILURE, subject.Runner.Status)
    testify.Equal(
      t,
      "package/nested/b.go: not formatted\n"+
        "package/nested/deep/c.go:3:9: "+
        string(ast.Failure_Message(ast.FAILURE_SYNTAX))+
        "\n",
      fixture_output(subject),
    )
    testify.Equal(t, "", fixture_errors(subject))
  }
}

// Test_Command_Json verifies -json writes each diagnostic as one flat JSON line.
func Test_Command_Json(t *testing.T) {
  memory, format_state, command, directory := memory_new()
  for seed := uint64(1); seed < SEED_COUNT+1; seed++ {
    subject := fixture_new(t, memory, format_state, command, directory, seed)
    fixture_package(t, subject)
    testify.True(t, fixture_run(subject, []string{"harness", "package", "-json"}))
    testify.Equal(t, harness.EXIT_FAILURE, subject.Runner.Status)
    testify.Equal(
      t,
      `{"path":"package/nested/b.go","line":0,"column":0,"tier":1,`+
        `"rule":"unformatted","message":"not formatted"}`+
        "\n"+
        `{"path":"package/nested/deep/c.go","line":3,"column":9,"tier":2,`+
        `"rule":"refused","message":"`+
        string(ast.Failure_Message(ast.FAILURE_SYNTAX))+
        `"}`+
        "\n",
      fixture_output(subject),
    )
    testify.Equal(t, "", fixture_errors(subject))
  }
}

// Test_Command_Fix verifies -fix replaces each auto-fixable file with its fix and reports only what
// stays.
func Test_Command_Fix(t *testing.T) {
  memory, format_state, command, directory := memory_new()
  for seed := uint64(1); seed < SEED_COUNT+1; seed++ {
    subject := fixture_new(t, memory, format_state, command, directory, seed)
    fixture_package(t, subject)
    testify.True(t, fixture_run(subject, []string{"harness", "package", "-fix"}))
    testify.Equal(t, harness.EXIT_FAILURE, subject.Runner.Status)
    refused := string(ast.Failure_Message(ast.FAILURE_SYNTAX))
    output := fixture_output(subject)
    testify.Equal(t, "package/nested/deep/c.go:3:9: "+refused+"\n", output)
    testify.Equal(t, "", fixture_errors(subject))
    testify.Equal(t, CLEAN, fixture_read(t, subject, "package/nested/b.go"))
    testify.Equal(t, REFUSED, fixture_read(t, subject, "package/nested/deep/c.go"))

    subject = fixture_new(t, memory, format_state, command, directory, seed)
    fixture_package_directory(t, subject, "package")
    fixture_file(t, subject, "package/b.go", UNFORMATTED)
    testify.True(t, fixture_run(subject, []string{"harness", "package", "-fix"}))
    testify.Equal(t, harness.EXIT_SUCCESS, subject.Runner.Status)
    testify.Equal(t, "", fixture_errors(subject))
    testify.Equal(t, CLEAN, fixture_read(t, subject, "package/b.go"))
  }
}

// Test_Command_Requirements verifies each package's contract findings follow the Go file
// diagnostics, at the contract file of their package, in text and in flat JSON.
func Test_Command_Requirements(t *testing.T) {
  memory, format_state, command, directory := memory_new()
  for seed := uint64(1); seed < SEED_COUNT+1; seed++ {
    output := requirements_run(
      t,
      memory,
      format_state,
      command,
      directory,
      seed,
      "package",
      UNFORMATTED,
      REQUIREMENTS_ABSENT,
      REQUIREMENTS_ABSENT,
      "package",
    )
    testify.Equal(
      t,
      "package/a.go: not formatted\n"+
        MARKDOWN_PATH+
        ": The package has no REQUIREMENTS.md. Add REQUIREMENTS.md.\n",
      output,
    )
    main_file := "package main\n\nfunc main() {}\n"
    output = requirements_run(
      t,
      memory,
      format_state,
      command,
      directory,
      seed,
      "package",
      main_file,
      REQUIREMENTS_ABSENT,
      REQUIREMENTS_ABSENT,
      "package",
    )
    requirements_passed(t, output)
    requirements_contracts(t, memory, format_state, command, directory, seed)
  }
  requirements_crowded(t, memory, format_state, command, directory)
  requirements_rewritten(t, memory, format_state, command, directory)
}

// Test_Command_History verifies the run judges each commit between the main ref and HEAD past
// the packages, judges none on main or where git fails, and names a missing main ref.
func Test_Command_History(t *testing.T) {
  memory, format_state, command, directory := memory_new()
  log := history_line("a", "b", "feat: fine") +
    history_line("c", "d", "did stuff") +
    history_line("e", "f 1111111111111111111111111111111111111111", "Merge branch 'x'") +
    history_line("2", "3", "fixup! feat: fine") +
    history_line("4", "5", "feat: "+HISTORY_WIDE_DESCRIPTION)
  scripts := [...]History_Script{
    {Head: "feature\n", Refs: HISTORY_REFS, Log: log},
    {Head: "main\n", Refs: HISTORY_REFS, Log: log},
    {Head: HISTORY_FAILED, Refs: HISTORY_REFS, Log: log},
    {Head: "HEAD\n", Refs: "", Log: log},
  }
  outputs := [...]string{
    "<git:cccccccccc>: The commit subject is not conventional. " +
      "Write \"type(scope): description\".\n" +
      "<git:eeeeeeeeee>: The commit is a merge commit. Rebase the branch onto main.\n" +
      "<git:2222222222>: The commit is a fixup commit. Squash it into its target commit.\n" +
      "<git:4444444444>: The commit subject is longer than 100 characters. " +
      "Write a shorter subject.\n",
    "",
    "",
    "<git:main>: The main ref is absent. Fetch main.\n",
  }
  for index, script := range scripts {
    subject := fixture_new(t, memory, format_state, command, directory, 1)
    subject.Loop.Spawn = history_spawn(script)
    fixture_directory(t, subject, "package")
    fixture_binary(t, subject, "package")
    fixture_file(t, subject, "package/m.go", "package main\n\nfunc main() {}\n")
    testify.True(t, fixture_run(subject, []string{"harness", "package"}))
    if outputs[index] == "" {
      requirements_passed(t, fixture_output(subject))
      continue
    }
    testify.Equal(t, outputs[index], fixture_output(subject))
    testify.Equal(t, harness.EXIT_FAILURE, subject.Runner.Status)
  }
}

// Test_Command_Quotes verifies a message quoting a heading writes each byte below a space as a
// question mark rather than passing it on to whatever reads the line.
func Test_Command_Quotes(t *testing.T) {
  memory, format_state, command, directory := memory_new()
  output := requirements_run(
    t,
    memory,
    format_state,
    command,
    directory,
    1,
    "package",
    CLEAN,
    "\n# A\vB\n\nBody.\n\n# A\vB\n\nBody.\n",
    CONTRACT_TEST,
    "package",
  )
  testify.True(
    t, bool(strings.Contains(strings.Text(output), strings.Text(`"A?B"`))), output,
  )
}

// Test_Command_Status verifies a package without diagnostics exits 0 and one with any exits 1.
func Test_Command_Status(t *testing.T) {
  memory, format_state, command, directory := memory_new()
  for seed := uint64(1); seed < SEED_COUNT+1; seed++ {
    subject := fixture_new(t, memory, format_state, command, directory, seed)
    fixture_package_directory(t, subject, "package")
    fixture_file(t, subject, "package/a.go", CLEAN)
    testify.True(t, fixture_run(subject, []string{"harness", "package"}))
    testify.Equal(t, harness.EXIT_SUCCESS, subject.Runner.Status)
    testify.Equal(t, "", fixture_errors(subject))

    subject = fixture_new(t, memory, format_state, command, directory, seed)
    testify.True(t, fixture_run(subject, []string{"harness", "absent"}))
    testify.Equal(t, harness.EXIT_FAILURE, subject.Runner.Status)
    testify.Equal(
      t,
      "harness: failed to walk: absent: the host refused the directory\n",
      fixture_errors(subject),
    )

    subject = fixture_new(t, memory, format_state, command, directory, seed)
    over := make([]byte, filepath.PATH_SIZE_MAXIMUM+1)
    for index := range over {
      over[index] = 'o'
    }
    testify.True(t, fixture_run(subject, []string{"harness", string(over)}))
    testify.Equal(t, harness.EXIT_FAILURE, subject.Runner.Status)
    testify.Equal(
      t,
      "harness: failed to walk: the argument names no directory\n",
      fixture_errors(subject),
    )
  }
}

// Test_Command_Partial verifies a failed step retracts nothing the run already wrote.
func Test_Command_Partial(t *testing.T) {
  memory, format_state, command, directory := memory_new()
  subject := fixture_new(t, memory, format_state, command, directory, 1)
  fixture_package_directory(t, subject, "package")
  wide := make([]byte, PARTIAL_NAME_BYTES)
  for index := range wide {
    wide[index] = 'w'
  }
  for index := range PARTIAL_FILE_COUNT {
    wide[0] = byte('a' + index%26)
    wide[1] = byte('a' + index/26)
    fixture_file(t, subject, "package/"+string(wide)+".go", UNFORMATTED)
  }
  testify.True(t, fixture_run(subject, []string{"harness", "package"}))
  testify.Equal(t, harness.EXIT_FAILURE, subject.Runner.Status)
  // The store the run writes into holds fewer lines than the package yields, thus the
  // write of one line fails and the lines before it stand.
  written := string(subject.Output.Memory[:subject.Output.Cursor])
  testify.True(t, bool(strings.Has_Prefix(strings.Text(written), "package/")), written)
  failed := fixture_errors(subject)
  testify.True(
    t, bool(strings.Has_Prefix(strings.Text(failed), "harness: failed to")), failed,
  )
}

// Test_Command_Passed verifies a clean run writes the pass line and its elapsed seconds, with or
// without -json.
func Test_Command_Passed(t *testing.T) {
  memory, format_state, command, directory := memory_new()
  cases := []struct {
    Arguments []string
    Prefix    string
    Suffix    string
  }{
    {[]string{"harness", "package"}, "✓ all checks passed\nelapsed_sec=", "\n"},
    {[]string{"harness", "package", "-json"}, `{"passed":true,"elapsed_sec":"`, "\"}\n"},
  }
  for seed := uint64(1); seed < SEED_COUNT+1; seed++ {
    for _, one := range cases {
      subject := fixture_new(t, memory, format_state, command, directory, seed)
      fixture_package_directory(t, subject, "package")
      fixture_file(t, subject, "package/a.go", CLEAN)
      testify.True(t, fixture_run(subject, one.Arguments))
      output := fixture_written(subject)
      opened := strings.Has_Prefix(strings.Text(output), strings.Text(one.Prefix))
      testify.True(t, bool(opened), output)
      closed := strings.Has_Suffix(strings.Text(output), strings.Text(one.Suffix))
      testify.True(t, bool(closed), output)
      elapsed := output[len(one.Prefix) : len(output)-len(one.Suffix)]
      point := len(elapsed) - len(".000")
      testify.True(t, point > 0, output)
      testify.Equal(t, byte('.'), elapsed[point], output)
      for index, character := range []byte(elapsed) {
        if index == point {
          continue
        }
        testify.True(t, character-'0' < 10, output)
      }
    }
  }
}

// Test_Command_Elapsed verifies a run with a diagnostic closes on the elapsed seconds, as text
// and as flat JSON.
func Test_Command_Elapsed(t *testing.T) {
  memory, format_state, command, directory := memory_new()
  cases := []struct {
    Arguments []string
    Prefix    string
    Suffix    string
  }{
    {[]string{"harness", "package"}, "package/a.go: not formatted\nelapsed_sec=", "\n"},
    {
      []string{"harness", "package", "-json"},
      `"message":"not formatted"}` + "\n" + `{"passed":false,"elapsed_sec":"`,
      "\"}\n",
    },
  }
  for _, one := range cases {
    subject := fixture_new(t, memory, format_state, command, directory, 1)
    fixture_package_directory(t, subject, "package")
    fixture_file(t, subject, "package/a.go", UNFORMATTED)
    testify.True(t, fixture_run(subject, one.Arguments))
    testify.Equal(t, harness.EXIT_FAILURE, subject.Runner.Status)
    output := fixture_written(subject)
    opening := int(strings.Index(strings.Text(output), strings.Text(one.Prefix)))
    testify.True(t, opening > -1, output)
    closed := strings.Has_Suffix(strings.Text(output), strings.Text(one.Suffix))
    testify.True(t, bool(closed), output)
    elapsed := output[opening+len(one.Prefix) : len(output)-len(one.Suffix)]
    point := len(elapsed) - len(".000")
    testify.True(t, point > 0, output)
    testify.Equal(t, byte('.'), elapsed[point], output)
  }
}

// Test_Format_Clean verifies a canonical file yields no diagnostic.
func Test_Format_Clean(t *testing.T) {
  runner := checked_new(CLEAN)
  testify.Equal(t, harness.TIER_NONE, runner.Diagnostic.Tier)
}

// Test_Format_Unformatted verifies a file off the canonical form yields a tier one diagnostic.
func Test_Format_Unformatted(t *testing.T) {
  runner := checked_new(UNFORMATTED)
  testify.Equal(t, harness.TIER_ONE, runner.Diagnostic.Tier)
  testify.Equal(t, harness.RULE_UNFORMATTED, runner.Diagnostic.Rule)
  testify.Equal(t, CLEAN, string(runner.Diagnostic.Fix))
}

// Test_Format_Refused verifies a refused file yields a tier two diagnostic naming the parser's
// refusal and where it stands.
func Test_Format_Refused(t *testing.T) {
  memory, format_state, command, directory := memory_new()
  runner := checked(memory, format_state, command, directory, REFUSED)
  testify.Equal(t, harness.TIER_TWO, runner.Diagnostic.Tier)
  testify.Equal(t, harness.RULE_REFUSED, runner.Diagnostic.Rule)
  testify.Equal(t, harness.Refusal(ast.FAILURE_SYNTAX), runner.Diagnostic.Failure)
  testify.Equal(t, harness.Diagnostic_Line(3), runner.Diagnostic.Line)
  testify.Equal(t, harness.Diagnostic_Column(9), runner.Diagnostic.Column)
  runner = checked(memory, format_state, command, directory, GROUPED)
  testify.Equal(t, harness.TIER_ONE, runner.Diagnostic.Tier)
  testify.Equal(t, "package a\n\nimport \"x\"\n", string(runner.Diagnostic.Fix))
  // A source of more lines than one index holds places no token, thus the refusal names
  // the file alone rather than a position the index never took.
  tall := make([]byte, 0, token.LINE_COUNT_MAXIMUM+len(REFUSED))
  tall = append(tall, "package a\n"...)
  for range token.LINE_COUNT_MAXIMUM {
    tall = append(tall, '\n')
  }
  tall = append(tall, "func f() int {\n\treturn 1\n}\n"...)
  runner = checked(memory, format_state, command, directory, string(tall))
  testify.Equal(t, harness.RULE_REFUSED, runner.Diagnostic.Rule)
  testify.Equal(t, harness.Diagnostic_Line(harness.POSITION_ABSENT), runner.Diagnostic.Line)
  testify.Equal(
    t, harness.Diagnostic_Column(harness.POSITION_ABSENT), runner.Diagnostic.Column,
  )
}

// Test_Format_Test_Closures verifies a closure holding logic refuses a file but a test file, and a
// test file's refusal past its closures stands where its token does.
func Test_Format_Test_Closures(t *testing.T) {
  memory, format_state, command, directory := memory_new()
  runner := checked(memory, format_state, command, directory, CLOSURE_LOGIC)
  testify.Equal(t, harness.Refusal(ast.FAILURE_CLOSURE_BODY), runner.Diagnostic.Failure)
  tested := harness.Check(runner.Memory, runner.Syntax, runner.Source_Count, runner.Format, true)
  testify.Equal(t, harness.TIER_NONE, tested.Tier)
  runner = checked(memory, format_state, command, directory, CLOSURE_LOGIC+UNNAMED_RESULT)
  tested = harness.Check(runner.Memory, runner.Syntax, runner.Source_Count, runner.Format, true)
  testify.Equal(t, harness.Refusal(ast.FAILURE_UNNAMED_RESULT), tested.Failure)
  testify.Equal(t, harness.Diagnostic_Line(10), tested.Line)
}

// Test_Format_Indentation verifies a file indented with tabs, and one ending without a newline,
// each stand outside the canonical form.
func Test_Format_Indentation(t *testing.T) {
  tabbed := checked_new("package a\n\nfunc g() (value int) {\n\treturn 1\n}\n")
  testify.Equal(t, harness.TIER_ONE, tabbed.Diagnostic.Tier)
  bare := checked_new("package a\n\nfunc h() {}")
  testify.Equal(t, harness.TIER_ONE, bare.Diagnostic.Tier)
}

// Test_Format_Names verifies a run renames a declared name in every file that spells it, and that a
// run holding a refused file renames nothing.
func Test_Format_Names(t *testing.T) {
  memory, format_state, command, directory := memory_new()
  for seed := uint64(1); seed < SEED_COUNT+1; seed++ {
    subject := fixture_new(t, memory, format_state, command, directory, seed)
    fixture_package_directory(t, subject, "package")
    fixture_file(t, subject, "package/a.go", DECLARATION)
    fixture_file(t, subject, "package/b.go", REFERENCE)
    testify.True(t, fixture_run(subject, []string{"harness", "package", "-fix"}))
    testify.Equal(t, harness.EXIT_SUCCESS, subject.Runner.Status)
    testify.Equal(t, "", fixture_errors(subject))
    testify.Equal(t, DECLARATION_RENAMED, fixture_read(t, subject, "package/a.go"))
    testify.Equal(t, REFERENCE_RENAMED, fixture_read(t, subject, "package/b.go"))

    subject = fixture_new(t, memory, format_state, command, directory, seed)
    fixture_package_directory(t, subject, "package")
    fixture_file(t, subject, "package/a.go", DECLARATION)
    fixture_file(t, subject, "package/b.go", REFERENCE)
    fixture_file(t, subject, "package/c.go", REFUSED)
    testify.True(t, fixture_run(subject, []string{"harness", "package", "-fix"}))
    testify.Equal(t, harness.EXIT_FAILURE, subject.Runner.Status)
    testify.Equal(t, DECLARATION, fixture_read(t, subject, "package/a.go"))
    testify.Equal(t, REFERENCE, fixture_read(t, subject, "package/b.go"))
  }
}

// Test_Format_Oversized verifies a file past the source bound yields a tier two diagnostic.
func Test_Format_Oversized(t *testing.T) {
  runner := checked_new("")
  runner.Source_Count = requirements.SOURCE_SIZE_MAXIMUM + 1
  runner.Diagnostic = harness.Check(
    runner.Memory,
    runner.Syntax,
    runner.Source_Count,
    runner.Format,
    false,
  )
  testify.Equal(t, harness.TIER_TWO, runner.Diagnostic.Tier)
  testify.Equal(t, harness.RULE_OVERSIZED, runner.Diagnostic.Rule)
}

// Test_Format_Conflicted verifies a Go file with a line that opens on a conflict marker yields a
// tier two diagnostic at that line, and a marker past a line's first byte does not.
func Test_Format_Conflicted(t *testing.T) {
  runner := checked_new("package a\n\n>>>>>>> side\n")
  testify.Equal(t, harness.TIER_TWO, runner.Diagnostic.Tier)
  testify.Equal(t, harness.RULE_CONFLICTED, runner.Diagnostic.Rule)
  testify.Equal(t, harness.Diagnostic_Line(3), runner.Diagnostic.Line)
  testify.Equal(t, harness.Diagnostic_Column(1), runner.Diagnostic.Column)
  runner = checked_new("package a\n\n// <<<<<<< side\n")
  testify.Equal(t, harness.TIER_NONE, runner.Diagnostic.Tier)
}

// Test_Format_Unprintable verifies a source whose canonical form outgrows the printer's storage
// yields a tier two diagnostic rather than standing on a form the printer never wrote. It stands
// here rather than in the simulation, since the printer spends a second on the two mebibytes it
// writes before it stops, which the whole simulation is allowed for every run it makes.
func Test_Format_Unprintable(t *testing.T) {
  source := make([]byte, 0, NEST_STATEMENT_COUNT*len("f()\n"))
  source = append(source, "package a\n\nfunc f() {\n"...)
  for range NEST_DEPTH {
    source = append(source, "if true {\n"...)
  }
  for range NEST_STATEMENT_COUNT {
    source = append(source, "f()\n"...)
  }
  for range NEST_DEPTH {
    source = append(source, "}\n"...)
  }
  source = append(source, "}\n"...)
  // The contract names the text this rule writes, and the printer spends a second on the
  // two mebibytes it writes, thus one tree states both forms rather than three runs of it.
  testify.Equal(t, "canonical form exceeds its storage", harness.MESSAGE_UNPRINTABLE)
  testify.Equal(t, harness.Rule_Text("unprintable"), harness.RULE_TEXT_UNPRINTABLE)
  memory, format_state, command, directory := memory_new()
  subject := fixture_new(t, memory, format_state, command, directory, 1)
  // A source this wide outgrows what a fixture file holds, thus this one tree carries
  // wider nodes than the fixtures that state a file of a few lines.
  fixture_widen(subject, len(source)+1)
  fixture_package_directory(t, subject, "package")
  fixture_file(t, subject, "package/wide.go", string(source))
  fixture_run(subject, []string{"harness", "package"})
  testify.Equal(
    t, "package/wide.go: canonical form exceeds its storage\n", fixture_output(subject),
  )
  fixture_run(subject, []string{"harness", "package", "-json"})
  testify.Equal(
    t,
    "package/wide.go: canonical form exceeds its storage\n"+
      "elapsed_sec=0.000\n"+
      `{"path":"package/wide.go","line":0,"column":0,"tier":2,`+
      `"rule":"unprintable",`+
      `"message":"canonical form exceeds its storage"}`+
      "\n",
    fixture_output(subject),
  )
}

// Test_Format_Imports verifies a clean file importing a banned stdlib package names its first such
// import, and a gateway or aver file importing what lint lets it does not.
func Test_Format_Imports(t *testing.T) {
  gateway_relative_path(t)
  memory, format_state, command, directory := memory_new()
  subject := fixture_new(t, memory, format_state, command, directory, 1)
  fixture_file(t, subject, harness.CONFIGURATION_PATH, shared_configuration())
  for _, path := range []string{
    "shared",
    "shared/a",
    "shared/b",
    "shared/c",
    "shared/std",
    "shared/std/os",
    "shared/std/os/default",
    "shared/std/sim",
    "shared/std/sim/aver",
  } {
    fixture_package_directory(t, subject, path)
  }
  // The os port stands above its default tier, which the composition tier then binds.
  fixture_file(t, subject, "shared/std/os/o.go", "package os\n")
  fixture_default_contract(t, subject, "shared/std/os/default", "os")
  fixture_file(t, subject, "shared/a/a.go", "package a\n\nimport \"fmt\"\nimport \"strings\"\n")
  fixture_file(
    t,
    subject,
    "shared/b/b.go",
    "package b\n\nimport \"crypto/sha256\"\nimport \"os\"\n",
  )
  fixture_file(t, subject, "shared/c/c.go", "package c\n\nimport \"math/rand/v2\"\n")
  fixture_file(t, subject, "shared/std/os/default/d.go", "package os\n\nimport \"os\"\n")
  fixture_file(t, subject, "shared/std/sim/aver/e.go", "package aver\n\nimport \"strings\"\n")
  testify.True(t, fixture_run(subject, []string{"harness", "shared"}))
  testify.Equal(t, harness.EXIT_FAILURE, subject.Runner.Status)
  testify.Equal(
    t,
    "shared/a/a.go:4:8: Banned import \"strings\" belongs to stdlib family \"strings\". "+
      "Use shared package.\n"+
      "shared/b/b.go:3:8: Banned import \"crypto/sha256\" belongs to stdlib family \"crypto\". "+
      "Use shared package.\n"+
      "shared/b/b.go:4:8: Banned import \"os\" belongs to stdlib family \"os\". "+
      "Use shared package.\n"+
      "shared/b/b.go:4:8: The stdlib import \"os\" is impure. "+
      "See lint/README.md for the resolutions.\n"+
      "shared/c/c.go:3:8: Banned import \"math/rand/v2\" belongs to stdlib family \"rand\". "+
      "Use shared package.\n",
    fixture_output(subject),
  )
}

// Test_Format_Name_Rules verifies a clean file breaking a naming rule states it at the name, in
// text
// and flat JSON, and a file importing a banned package states the import alone.
func Test_Format_Name_Rules(t *testing.T) {
  memory, format_state, command, directory := memory_new()
  banned := "The identifier \\\"util_table\\\" contains the banned substring \\\"util\\\". " +
    "Write a different name."
  imported := "Banned import \\\"strings\\\" belongs to stdlib family \\\"strings\\\". " +
    "Use shared package."
  for _, flags := range [][]string{{}, {"-json"}} {
    subject := fixture_new(t, memory, format_state, command, directory, 1)
    fixture_file(t, subject, harness.CONFIGURATION_PATH, shared_configuration())
    for _, path := range []string{"shared", "shared/a", "shared/b", "shared/c"} {
      fixture_package_directory(t, subject, path)
    }
    fixture_file(t, subject, "shared/a/a.go", "package a\n\nvar util_table = 1\n")
    fixture_file(t, subject, "shared/b/b.go", "package b\n\nimport \"strings\"\n\nvar util = 1\n")
    fixture_file(t, subject, "shared/c/c.go", "package c\n\nvar table = 1\n")
    testify.True(t, fixture_run(subject, append([]string{"harness", "shared"}, flags...)))
    testify.Equal(t, harness.EXIT_FAILURE, subject.Runner.Status)
    word := "The identifier \\\"util\\\" contains the banned substring \\\"util\\\". " +
      "Write a different name."
    want := "shared/a/a.go:3:5: " +
      unescaped(banned) +
      "\n" +
      "shared/b/b.go:3:8: " +
      unescaped(imported) +
      "\n" +
      "shared/b/b.go:5:5: " +
      unescaped(word) +
      "\n"
    if len(flags) > 0 {
      want = `{"path":"shared/a/a.go","line":3,"column":5,"tier":2,"rule":"names","message":"` +
        banned +
        "\"}\n" +
        `{"path":"shared/b/b.go","line":3,"column":8,"tier":2,"rule":"imports","message":"` +
        imported +
        "\"}\n" +
        `{"path":"shared/b/b.go","line":5,"column":5,"tier":2,"rule":"names","message":"` +
        word +
        "\"}\n"
    }
    testify.Equal(t, want, fixture_output(subject))
  }
}

// Test_Format_Shadows verifies a name hiding one of its own scopes, or a top-level name a sibling
// file declares, states the shadow, an external test package shares no name with its library, and
// a builtin a sibling calls is no top-level name.
func Test_Format_Shadows(t *testing.T) {
  memory, format_state, command, directory := memory_new()
  message := "The variable \\\"value\\\" shadows a variable in an outer scope. Write a different name."
  for _, flags := range [][]string{{}, {"-json"}} {
    subject := fixture_new(t, memory, format_state, command, directory, 1)
    fixture_file(t, subject, harness.CONFIGURATION_PATH, shared_configuration())
    for _, path := range []string{"shared", "shared/a", "shared/c"} {
      fixture_package_directory(t, subject, path)
    }
    fixture_file(t, subject, "shared/a/a.go", "package a\n\nfunc value() {}\n")
    fixture_file(t, subject, "shared/a/b.go", "package a\n\nfunc F() {\n  value := 1\n}\n")
    fixture_file(
      t,
      subject,
      "shared/a/e_test.go",
      "package a_test\n\nfunc G() {\n  value := 1\n}\n",
    )
    // A call names no declaration, thus a local that spells a builtin a sibling calls stands.
    fixture_file(t, subject, "shared/a/f.go", "package a\n\nfunc H() {\n  copy(a, b)\n}\n")
    fixture_file(t, subject, "shared/a/g.go", "package a\n\nfunc K() {\n  copy := 1\n}\n")
    fixture_file(
      t,
      subject,
      "shared/c/c.go",
      "package c\n\nfunc F(value int) {\n  {\n    value := 1\n  }\n}\n",
    )
    testify.True(t, fixture_run(subject, append([]string{"harness", "shared"}, flags...)))
    testify.Equal(t, harness.EXIT_FAILURE, subject.Runner.Status)
    want := "shared/a/b.go:4:3: " +
      unescaped(message) +
      "\n" +
      "shared/c/c.go:5:5: " +
      unescaped(message) +
      "\n"
    if len(flags) > 0 {
      want = `{"path":"shared/a/b.go","line":4,"column":3,"tier":2,"rule":"shadows","message":"` +
        message +
        "\"}\n" +
        `{"path":"shared/c/c.go","line":5,"column":5,"tier":2,"rule":"shadows","message":"` +
        message +
        "\"}\n"
    }
    testify.Equal(t, want, fixture_output(subject))
  }
}

// Test_Format_Recursion verifies a function calling itself, and a cycle across two files, each
// state the call that closes it, while a method call and a file opted out of the ban state nothing.
func Test_Format_Recursion(t *testing.T) {
  memory, format_state, command, directory := memory_new()
  subject := fixture_new(t, memory, format_state, command, directory, 1)
  fixture_file(t, subject, harness.CONFIGURATION_PATH, shared_configuration())
  for _, path := range []string{"shared", "shared/a", "shared/b", "shared/m", "shared/x"} {
    fixture_package_directory(t, subject, path)
  }
  exempt := strings_replace(
    shared_configuration(),
    `"opt_out_recursion_ban":[]`,
    `"opt_out_recursion_ban":["shared/x/**"]`,
  )
  fixture_file(t, subject, harness.CONFIGURATION_PATH, exempt)
  fixture_file(t, subject, "shared/a/a.go", "package a\n\nfunc F() {\n  F()\n}\n")
  fixture_file(t, subject, "shared/b/b.go", "package b\n\nfunc F() {\n  G()\n}\n")
  fixture_file(t, subject, "shared/b/c.go", "package b\n\nfunc G() {\n  go F()\n}\n")
  fixture_file(
    t,
    subject,
    "shared/m/m.go",
    "package m\n\ntype T int\n\nfunc (value T) F() {\n  value.F()\n}\n",
  )
  fixture_file(t, subject, "shared/x/x.go", "package x\n\nfunc F() {\n  F()\n}\n")
  testify.True(t, fixture_run(subject, []string{"harness", "shared"}))
  testify.Equal(t, harness.EXIT_FAILURE, subject.Runner.Status)
  testify.Equal(
    t,
    "shared/a/a.go:4:3: The function F recurses. Write a loop instead.\n"+
      "shared/b/c.go:4:6: The functions recurse in a cycle. Write a loop instead. F → G → F\n",
    fixture_output(subject),
  )
}

// Test_Format_Lanes verifies a run holds up to LANE_COUNT Go files at the root's spread at once,
// states the same whatever order the spread checks them in, and renames a name a later file
// declares in a file checked before it.
func Test_Format_Lanes(t *testing.T) {
  memory, format_state, command, directory := memory_new()
  written := [ORDER_COUNT]string{}
  held := [ORDER_COUNT]Held_Lanes{{Checked: true}, {Checked: false}}
  for order := range written {
    subject := lanes_fixture(t, memory, format_state, command, directory)
    spread := &held[order]
    subject.Runner.Spread.Submit = func(lane harness.Lane_Handle) { held_submit(spread, lane) }
    subject.Runner.Spread.Reap = func(memory harness.Memory_Handle) (lane harness.Lane_Index) {
      return held_reap(spread, memory)
    }
    testify.True(t, fixture_run(subject, []string{"harness", "package"}))
    written[order] = fixture_output(subject)
  }
  // Each Go file is parsed once, the contract test beside the files among them, even in a run
  // whose table took a name.
  testify.Equal(t, harness.LANE_COUNT+2, held[0].Submitted)
  testify.Equal(t, harness.LANE_COUNT+2, held[1].Submitted)
  testify.Equal(t, 1, held[0].Deepest)
  testify.Equal(t, harness.LANE_COUNT, held[1].Deepest)
  testify.Equal(t, written[0], written[1])
  testify.True(t, bool(strings.Contains(strings.Text(written[0]), "package/a.go: not formatted")))
  subject := lanes_fixture(t, memory, format_state, command, directory)
  testify.True(t, fixture_run(subject, []string{"harness", "package", "-fix"}))
  testify.Equal(t, REFERENCE_RENAMED, fixture_read(t, subject, "package/a.go"))
  last := string(rune('a'+harness.LANE_COUNT)) + ".go"
  testify.Equal(t, DECLARATION_RENAMED, fixture_read(t, subject, "package/"+last))
}

// Test_Layout_Single_Module verifies a tree holding no go.mod states it at go.mod, and a nested
// go.mod and a go.work each state themselves.
func Test_Layout_Single_Module(t *testing.T) {
  memory, format_state, command, directory := memory_new()
  subject := fixture_new(t, memory, format_state, command, directory, 1)
  fixture_package_directory(t, subject, "package")
  fixture_file(t, subject, "package/a.go", CLEAN)
  fixture_remove(t, subject, "go.mod")
  testify.True(t, fixture_run(subject, []string{"harness", "package"}))
  testify.Equal(t, harness.EXIT_FAILURE, subject.Runner.Status)
  testify.Equal(
    t,
    "go.mod: There is no go.mod at the repository root. The linter needs the module anchor.\n",
    fixture_output(subject),
  )
  subject = fixture_new(t, memory, format_state, command, directory, 1)
  fixture_package_directory(t, subject, "package")
  fixture_file(t, subject, "package/a.go", CLEAN)
  fixture_file(t, subject, "package/go.mod", "module other\n")
  fixture_file(t, subject, "package/go.work", "go 1.26\n")
  testify.True(t, fixture_run(subject, []string{"harness", "package"}))
  testify.Equal(
    t,
    "package/go.mod: A nested go.mod divides the repository into more than one module. "+
      "Remove \"package/go.mod\".\n"+
      "package/go.work: A go.work makes a workspace with more than one module. "+
      "Remove \"package/go.work\".\n",
    fixture_output(subject),
  )
}

// Test_Layout_Shared_Component verifies the shared component states an internal directory and a
// package main, and a library package stands.
func Test_Layout_Shared_Component(t *testing.T) {
  layout_expect(
    t,
    "shared",
    []string{
      "shared/internal/a.go",
      "package internal\n",
      "shared/m/m.go",
      "package main\n\nfunc main() {}\n",
      "shared/fine/f.go",
      "package fine\n",
    },
    []string{
      "shared/internal/a.go:1:1: A shared library permits no internal/ directory. " +
        "Remove \"shared/internal\".",
      "shared/m/m.go:1:1: The shared library \"local/fixture/shared\" permits no package main. " +
        "Move the entry point to a binary component.",
    },
    []string{"shared/fine/f.go:"},
  )
}

// Test_Layout_Binary_Component verifies a binary component states a package outside pkg, at its
// root or below it, and its pkg packages and package main stand.
func Test_Layout_Binary_Component(t *testing.T) {
  layout_expect(
    t,
    "tool",
    []string{
      "tool/main.go",
      "package main\n\nfunc main() {}\n",
      "tool/pkg/p.go",
      "package pkg\n\nfunc Main() {}\n",
      "tool/lib/l.go",
      "package lib\n",
      "tool/r.go",
      "package tool\n",
    },
    []string{
      "tool/lib/l.go:1:1: Move lib -> tool/pkg/lib.",
      "tool/r.go:1:1: Move . -> tool/pkg.",
    },
    []string{"tool/pkg/p.go:", "tool/main.go:"},
  )
}

// Test_Layout_Binary_Isolation verifies a binary component importing another states the import,
// and one importing the shared component stands.
func Test_Layout_Binary_Isolation(t *testing.T) {
  layout_expect(
    t,
    "tool",
    []string{
      "tool/main.go",
      "package main\n\nfunc main() {}\n",
      "tool/pkg/p.go",
      "package pkg\n\nimport \"local/fixture/other/pkg\"\n\nfunc Main() {}\n",
      "tool/pkg/q/q.go",
      "package q\n\nimport \"local/fixture/shared/fine\"\n",
    },
    []string{
      "tool/pkg/p.go:3:8: The binary component \"local/fixture/tool\" imports the binary " +
        "component \"local/fixture/other\". Move the shared code to the shared component.",
    },
    []string{"tool/pkg/q/q.go:"},
  )
}

// Test_Layout_Main_Package verifies a package main below the component root states it.
func Test_Layout_Main_Package(t *testing.T) {
  layout_expect(
    t,
    "tool",
    []string{
      "tool/cmd/main.go",
      "package main\n\nfunc main() {}\n",
      "tool/pkg/p.go",
      "package pkg\n\nfunc Main() {}\n",
    },
    []string{
      "tool/cmd/main.go:1:1: The binary component \"local/fixture/tool\" has its main package at " +
        "\"cmd\". Move the main package to the component root.",
    },
    []string{"tool/pkg/p.go:"},
  )
}

// Test_Layout_Entry_Point verifies a binary component the walk covers states no func Main and
// several at the component, a test's func Main counts for nothing, and a narrower walk states none.
func Test_Layout_Entry_Point(t *testing.T) {
  absent := "tool: The binary component \"local/fixture/tool\" declares no func Main in pkg/. " +
    "Declare a func Main in pkg/."
  layout_expect(
    t,
    "tool",
    []string{
      "tool/main.go",
      "package main\n\nfunc main() {}\n",
      "tool/pkg/p.go",
      "package pkg\n",
      "tool/pkg/p_test.go",
      "package pkg\n\nfunc Main() {}\n",
    },
    []string{absent},
    nil,
  )
  layout_expect(
    t,
    "tool/pkg",
    []string{
      "tool/main.go", "package main\n\nfunc main() {}\n",
      "tool/pkg/p.go", "package pkg\n",
    },
    nil,
    []string{"tool: "},
  )
  layout_expect(
    t,
    "tool",
    []string{
      "tool/main.go",
      "package main\n\nfunc main() {}\n",
      "tool/pkg/p.go",
      "package pkg\n\nfunc Main() {}\n",
      "tool/pkg/q.go",
      "package pkg\n\nfunc Main() {}\n",
    },
    []string{
      "tool: The binary component \"local/fixture/tool\" declares more than one func Main in " +
        "pkg/. Declare only one func Main.",
    },
    nil,
  )
}

// Test_Layout_Tier_Depth verifies a package under two library ancestors states them, while a
// version segment and a binary's pkg count for none.
func Test_Layout_Tier_Depth(t *testing.T) {
  layout_expect(
    t,
    ".",
    []string{
      "shared/a/a.go",
      "package a\n",
      "shared/a/b/b.go",
      "package b\n",
      "shared/a/b/c/c.go",
      "package c\n",
      "shared/a/v2/d/d.go",
      "package d\n",
      "tool/main.go",
      "package main\n\nfunc main() {}\n",
      "tool/pkg/p.go",
      "package pkg\n\nfunc Main() {}\n",
      "tool/pkg/x/x.go",
      "package x\n",
      "tool/pkg/x/y/y.go",
      "package y\n",
    },
    []string{
      "shared/a/b/c/c.go:1:1: The package \"c\" at \"a/b/c\" is below the library tier. " +
        "It has 2 non-main ancestors: [a/b a].",
    },
    []string{"shared/a/b/b.go:", "shared/a/v2/d/d.go:", "tool/pkg/x/y/y.go:"},
  )
}

// Test_Layout_Impure_Imports verifies a pure file importing an impure stdlib package states it,
// while a test and the composition tier stand.
func Test_Layout_Impure_Imports(t *testing.T) {
  layout_expect(
    t,
    "shared",
    []string{
      "shared/p/p.go",
      "package p\n\nimport \"runtime\"\n",
      "shared/p/p_test.go",
      "package p\n\nimport \"runtime\"\n",
      "shared/p/q/q.go",
      "package q\n\nimport \"runtime\"\n",
    },
    []string{
      "shared/p/p.go:3:8: The stdlib import \"runtime\" is impure. " +
        "See lint/README.md for the resolutions.",
    },
    []string{"shared/p/p_test.go:", "shared/p/q/q.go:"},
  )
}

// Test_Layout_Impure_Calls verifies a pure file selecting the clock states it, and a test stands.
func Test_Layout_Impure_Calls(t *testing.T) {
  clock := "package p\n\nimport \"time\"\n\nfunc F() {\n  time.Now()\n}\n"
  layout_expect(
    t,
    "shared",
    []string{
      "shared/p/p.go", clock,
      "shared/p/p_test.go", clock,
    },
    []string{
      "shared/p/p.go:6:3: The stdlib call time.Now is impure. " +
        "See lint/README.md for the resolutions.",
    },
    []string{"shared/p/p_test.go:"},
  )
}

// Test_Layout_Transitive_Purity verifies a pure package importing a default directory states it,
// and one importing an instrumentation package stands.
func Test_Layout_Transitive_Purity(t *testing.T) {
  layout_expect(
    t,
    "shared",
    []string{
      "shared/port/o.go",
      "package port\n",
      "shared/port/default/d.go",
      "package port\n",
      "shared/probe/p.go",
      "package probe\n",
      "shared/probe/default/d.go",
      "package probe\n",
      "shared/user/u.go",
      "package user\n\nimport \"local/fixture/shared/port/default\"\n",
      "shared/watch/w.go",
      "package watch\n\nimport \"local/fixture/shared/probe/default\"\n",
    },
    []string{
      "shared/user/u.go:3:8: The dependency \"local/fixture/shared/port/default\" is impure. " +
        "A pure package imports only pure packages.",
    },
    []string{"shared/watch/w.go:"},
  )
}

// Test_Layout_Transitive_Stdlib verifies a pure package selecting an API that reaches the clock
// states it, its tests included.
func Test_Layout_Transitive_Stdlib(t *testing.T) {
  deadline := "package w\n\nimport \"context\"\n\nfunc F() {\n  context.WithTimeout()\n}\n"
  layout_expect(
    t,
    "shared",
    []string{
      "shared/w/w.go", deadline,
      "shared/w/w_test.go", deadline,
    },
    []string{
      "shared/w/w.go:6:3: The transitive call context.WithTimeout is impure. " +
        "A pure package calls only pure APIs.",
      "shared/w/w_test.go:6:3: The transitive call context.WithTimeout is impure. " +
        "A pure package calls only pure APIs.",
    },
    nil,
  )
}

// Test_Layout_Binary_Purity verifies a binary's pkg package importing an impure stdlib package
// states it, and its package main stands.
func Test_Layout_Binary_Purity(t *testing.T) {
  layout_expect(
    t,
    "tool",
    []string{
      "tool/main.go",
      "package main\n\nimport \"runtime\"\n\nfunc main() {}\n",
      "tool/pkg/p.go",
      "package pkg\n\nfunc Main() {}\n",
      "tool/pkg/x/y/y.go",
      "package y\n\nimport \"runtime\"\n",
    },
    []string{
      "tool/pkg/x/y/y.go:3:8: The stdlib import \"runtime\" is impure. " +
        "See lint/README.md for the resolutions.",
    },
    []string{"tool/main.go:"},
  )
}

// Test_Layout_Binary_Default_Tier verifies a binary's default directory states it.
func Test_Layout_Binary_Default_Tier(t *testing.T) {
  layout_expect(
    t,
    "tool",
    []string{
      "tool/main.go",
      "package main\n\nfunc main() {}\n",
      "tool/pkg/p.go",
      "package pkg\n\nfunc Main() {}\n",
      "tool/pkg/default/d.go",
      "package pkg\n",
    },
    []string{
      "tool/pkg/default/d.go:1:1: A binary component permits no default tier. " +
        "Bind the real world in package main. Remove \"tool/pkg/default\".",
    },
    nil,
  )
}

// Test_Layout_Library_Purity verifies a shared library package importing an impure stdlib package
// states it, and its default directory, the composition tier, stands.
func Test_Layout_Library_Purity(t *testing.T) {
  layout_expect(
    t,
    "shared",
    []string{
      "shared/lib/l.go",
      "package lib\n\nimport \"runtime\"\n",
      "shared/lib/default/d.go",
      "package lib\n\nimport \"runtime\"\n",
    },
    []string{
      "shared/lib/l.go:3:8: The stdlib import \"runtime\" is impure. " +
        "See lint/README.md for the resolutions.",
    },
    []string{"shared/lib/default/d.go:"},
  )
}

// Test_Layout_Default_Package_Impurity verifies a default directory declaring another package
// states it at the clause, and one declaring its parent's test package stands.
func Test_Layout_Default_Package_Impurity(t *testing.T) {
  layout_expect(
    t,
    "shared",
    []string{
      "shared/lib/l.go",
      "package lib\n",
      "shared/lib/default/d.go",
      "package other\n",
      "shared/lib/default/d_test.go",
      "package lib_test\n",
    },
    []string{
      "shared/lib/default/d.go:1:9: The default package declares \"package other\", " +
        "not \"package lib\". The default package shadows the library it re-exports.",
    },
    []string{"shared/lib/default/d_test.go:"},
  )
}

// Test_Memory_Budget verifies New_Memory allocates at most MEMORY_SIZE_MAXIMUM bytes.
func Test_Memory_Budget(t *testing.T) {
  before, after := runtime.MemStats{}, runtime.MemStats{}
  runtime.ReadMemStats(&before)
  harness.New_Memory()
  runtime.ReadMemStats(&after)
  spent := after.TotalAlloc - before.TotalAlloc
  testify.True(t, spent < harness.MEMORY_SIZE_MAXIMUM+1, "New_Memory spent %d bytes", spent)
}

// Test_Memory_Packages verifies two packages whose names together outgrow the sibling table each
// judge their siblings, since the next package reuses the stores.
func Test_Memory_Packages(t *testing.T) {
  memory, format_state, command, directory := memory_new()
  size := Fixture_Size{Nodes: NODE_CAPACITY, Contents: WIDE_CONTENTS_BYTES, Output: OUTPUT_BYTES}
  subject := fixture_sized(t, memory, format_state, command, directory, 1, size)
  fixture_file(t, subject, harness.CONFIGURATION_PATH, shared_configuration())
  fixture_package_directory(t, subject, "shared")
  filler := locals_source(harness.SIBLING_TEXT_SIZE_MAXIMUM / 2)
  for _, name := range []string{"a", "b"} {
    path := "shared/" + name
    fixture_package_directory(t, subject, path)
    fixture_file(t, subject, path+"/a.go", "package a\n\nfunc value() {}\n")
    fixture_file(t, subject, path+"/b.go", "package a\n\nfunc F() {\n  value := 1\n}\n")
    fixture_file(t, subject, path+"/c.go", "package a\n\n"+filler)
  }
  testify.True(t, fixture_run(subject, []string{"harness", "shared"}))
  message := "The variable \"value\" shadows a variable in an outer scope. Write a different name."
  want := "shared/a/b.go:4:3: " + message + "\nshared/b/b.go:4:3: " + message + "\n"
  testify.Equal(t, want, fixture_output(subject))
}

// Test_Memory_Capacity verifies a file past the sibling table, the naming finding table, the
// report table, or the declaration table states the store it outgrew.
func Test_Memory_Capacity(t *testing.T) {
  memory, format_state, command, directory := memory_new()
  size := Fixture_Size{Nodes: NODE_CAPACITY, Contents: WIDE_CONTENTS_BYTES, Output: OUTPUT_BYTES}
  subject := fixture_sized(t, memory, format_state, command, directory, 1, size)
  fixture_file(t, subject, harness.CONFIGURATION_PATH, shared_configuration())
  fixture_package_directory(t, subject, "shared")
  fixture_package_directory(t, subject, "shared/a")
  filler := locals_source(harness.SIBLING_TEXT_SIZE_MAXIMUM / 2)
  fixture_file(t, subject, "shared/a/a.go", "package a\n\n"+filler)
  fixture_file(t, subject, "shared/a/b.go", "package a\n\n"+filler)
  testify.True(t, fixture_run(subject, []string{"harness", "shared"}))
  testify.Equal(t, harness.EXIT_FAILURE, subject.Runner.Status)
  testify.Equal(
    t,
    "shared/a/b.go: The file outgrows the sibling table. Its shadows across files, recursion, "+
      "and layout stand unjudged.\n",
    fixture_output(subject),
  )
  size.Output = REPORT_OUTPUT_BYTES
  subject = fixture_sized(t, memory, format_state, command, directory, 1, size)
  fixture_file(t, subject, harness.CONFIGURATION_PATH, shared_configuration())
  fixture_package_directory(t, subject, "shared")
  fixture_package_directory(t, subject, "shared/a")
  fixture_file(
    t,
    subject,
    "shared/a/a.go",
    "package a\n\n"+variables_source("util", naming.FINDING_COUNT_MAXIMUM),
  )
  testify.True(t, fixture_run(subject, []string{"harness", "shared"}))
  output := fixture_output(subject)
  naming_full := "shared/a/a.go: The file outgrows the naming finding table. Its findings past it " +
    "stand unstated.\n"
  testify.Equal(t, 1, strings_count(output, naming_full))
  testify.Equal(t, naming.FINDING_COUNT_MAXIMUM, strings_count(output, "\n"))
  subject = fixture_sized(t, memory, format_state, command, directory, 1, size)
  fixture_file(t, subject, harness.CONFIGURATION_PATH, shared_configuration())
  fixture_package_directory(t, subject, "shared")
  fixture_package_directory(t, subject, "shared/a")
  scoped := variables_source("table", naming.SCOPE_NAME_COUNT_MAXIMUM+1)
  fixture_file(t, subject, "shared/a/a.go", "package a\n\n"+scoped)
  testify.True(t, fixture_run(subject, []string{"harness", "shared"}))
  testify.Equal(
    t,
    "shared/a/a.go: The file outgrows the scope table. Its shadows past it stand unjudged.\n",
    fixture_output(subject),
  )
  subject = fixture_sized(t, memory, format_state, command, directory, 1, size)
  fixture_file(t, subject, harness.CONFIGURATION_PATH, shared_configuration())
  fixture_package_directory(t, subject, "shared")
  fixture_package_directory(t, subject, "shared/a")
  banned := variables_source("util", naming.FINDING_COUNT_MAXIMUM-naming.FINDING_FIRST)
  files := harness.REPORT_COUNT_MAXIMUM/(naming.FINDING_COUNT_MAXIMUM-naming.FINDING_FIRST) + 1
  for index := range files {
    fixture_file(t, subject, "shared/a/"+string(rune('a'+index))+".go", "package a\n\n"+banned)
  }
  testify.True(t, fixture_run(subject, []string{"harness", "shared"}))
  output = fixture_output(subject)
  reports_full := ": The run outgrows the report table. Its findings past this one stand unstated.\n"
  testify.Equal(t, 1, strings_count(output, reports_full))
  testify.Equal(t, harness.REPORT_COUNT_MAXIMUM-harness.REPORT_FIRST, strings_count(output, "\n"))
  size = Fixture_Size{
    Nodes:    DECLARED_NODE_COUNT,
    Contents: DECLARED_CONTENTS_BYTES,
    Output:   REPORT_OUTPUT_BYTES,
  }
  subject = fixture_sized(t, memory, format_state, command, directory, 1, size)
  fixture_file(t, subject, harness.CONFIGURATION_PATH, shared_configuration())
  fixture_package_directory(t, subject, "shared")
  declared := declared_source(requirements.HEADING_COUNT_MAXIMUM)
  packages := harness.DECLARATION_COUNT_MAXIMUM/requirements.HEADING_COUNT_MAXIMUM + 1
  for index := range packages {
    path := "shared/p" + decimal(index)
    fixture_directory(t, subject, path)
    fixture_file(t, subject, path+"/REQUIREMENTS.md", CONTRACT_MARKDOWN)
    fixture_file(t, subject, path+"/requirements_test.go", declared)
  }
  testify.True(t, fixture_run(subject, []string{"harness", "shared"}))
  // The walk decides which packages read last, thus which ones outgrow the table.
  declarations_full := "/requirements_test.go: The file outgrows the declaration table. Its " +
    "declarations past it stand unchecked.\n"
  testify.True(
    t,
    strings_count(fixture_output(subject), declarations_full) > 0,
    fixture_errors(subject),
  )
  // A tree of long paths outgrows the walk text store before any check reads a file.
  spelled := make([]byte, WIDE_COMPONENT_BYTES)
  for index := range spelled {
    spelled[index] = 'd'
  }
  long := string(spelled)
  deep := "shared/" + long + "/" + long + "/" + long
  size = Fixture_Size{
    Nodes:    WALK_NODE_COUNT,
    Contents: NODE_CONTENTS_BYTES,
    Output:   OUTPUT_BYTES,
  }
  subject = fixture_sized(t, memory, format_state, command, directory, 1, size)
  fixture_file(t, subject, harness.CONFIGURATION_PATH, shared_configuration())
  fixture_package_directory(t, subject, "shared")
  fixture_directory(t, subject, "shared/"+long)
  fixture_directory(t, subject, "shared/"+long+"/"+long)
  fixture_directory(t, subject, deep)
  for index := range WALK_FILE_COUNT {
    fixture_file(t, subject, deep+"/"+decimal(index)+long[:WIDE_COMPONENT_BYTES-8]+".go", CLEAN)
  }
  testify.True(t, fixture_run(subject, []string{"harness", "shared"}))
  testify.Equal(t, harness.EXIT_FAILURE, subject.Runner.Status)
  testify.True(
    t,
    bool(
      strings.Contains(
        strings.Text(fixture_errors(subject)),
        ": more than 131072 path bytes held\n",
      ),
    ),
    fixture_errors(subject),
  )
}

// Test_Allocation verifies a whole run, walk and checks and report, allocates nothing.
func Test_Allocation(t *testing.T) {
  memory, format_state, command, directory := memory_new()
  cases := []struct {
    Arguments []string
    Setup     func(t *testing.T, subject *fixture)
  }{
    {[]string{"harness", "package"}, fixture_package},
    {[]string{"harness", "package", "-json"}, fixture_package},
    {[]string{"harness", "package", "-fix"}, fixture_package},
    {[]string{"harness"}, fixture_empty},
    {[]string{"harness", "-h"}, fixture_empty},
    {[]string{"harness", "absent"}, fixture_empty},
    {[]string{"harness", "package"}, fixture_clean},
    {[]string{"harness", "package", "-json"}, fixture_clean},
    {[]string{"harness", "package"}, fixture_faulty_contract},
    {[]string{"harness", "package", "-json"}, fixture_faulty_contract},
    {[]string{"harness", "package"}, fixture_uncontracted},
  }
  for seed := uint64(1); seed < SEED_COUNT+1; seed++ {
    for _, one := range cases {
      subject := fixture_new(t, memory, format_state, command, directory, seed)
      one.Setup(t, subject)
      testify.Equal(
        t,
        ALLOCATION_NONE,
        allocation_least(t, func() { fixture_run(subject, one.Arguments) }),
        "seed %d, %v",
        seed,
        one.Arguments,
      )
      phase := harness.Pass_Phase(subject.Runner.Progress.Phase)
      testify.True(t, bool(subject.Runner.Stopped(phase)))
    }
  }
}

// SEED_COUNT keeps the sweep wide enough to reorder retirements and fast.
const SEED_COUNT = 16

// NODE_CAPACITY leaves room for the tree the seed generates beside the package each test builds.
const NODE_CAPACITY = 64

// DESCRIPTOR_CAPACITY covers the one directory or file a run holds open at a time, with slack.
const DESCRIPTOR_CAPACITY = 8

// OPERATION_CAPACITY covers the one operation a run keeps in flight, with slack.
const OPERATION_CAPACITY = 8

// PASS_COUNT bounds pump passes; a test package submits well under a hundred operations.
const PASS_COUNT = 1024

// PUMP_DURATION is strictly longer than every timeout a run grants, so the operation retires
// before the root's guard does.
const PUMP_DURATION = harness.READ_TIMEOUT + time.MINUTE

// PARTIAL_FILE_COUNT is how many files one package needs for its lines to outgrow the store the
// run writes into, so that a write fails after the lines before it landed.
const PARTIAL_FILE_COUNT = 40

// PARTIAL_NAME_BYTES is how wide each of those names stands, so that forty lines outgrow it.
const PARTIAL_NAME_BYTES = 100

// OUTPUT_BYTES holds every line a test package yields.
const OUTPUT_BYTES = 4096

// NODE_CONTENTS_BYTES is what one simulated file holds: the run under test reads its own storage,
// so a fixture file needs no more than the widest one a check writes.
const NODE_CONTENTS_BYTES = 1024

// FILE_PERMISSIONS is what each fixture file carries; the run only reads.
const FILE_PERMISSIONS nbio.File_Permissions = 0o644

// DIRECTORY_PERMISSIONS is what each fixture directory carries.
const DIRECTORY_PERMISSIONS nbio.File_Permissions = 0o755

// CLEAN is a file in canonical form.
const CLEAN = "package a\n\nfunc f() {}\n"

// UNFORMATTED is CLEAN with a doubled space the canonical form drops.
const UNFORMATTED = "package a\n\nfunc  f() {}\n"

// NEST_DEPTH stands inside what one parse admits, thus the source parses and reaches the printer.
const NEST_DEPTH = 120

// NEST_STATEMENT_COUNT holds enough statements that their indentation alone outgrows the form
// store: each one spends two bytes per step of the depth it stands under.
const NEST_STATEMENT_COUNT = 9000

// DECLARATION holds a name in the wrong case.
const DECLARATION = "package a\n\nvar someValue = 1\n"

// DECLARATION_RENAMED is DECLARATION with its name in snake_case.
const DECLARATION_RENAMED = "package a\n\nvar some_value = 1\n"

// REFERENCE holds the name of DECLARATION, in a file of its own.
const REFERENCE = "package a\n\nvar held = someValue\n"

// REFERENCE_RENAMED is REFERENCE with the name in snake_case.
const REFERENCE_RENAMED = "package a\n\nvar held = some_value\n"

// REFUSED is a file the parser refuses as bad syntax at the brace.
const REFUSED = "package a\n\nfunc f( {}\n"

// GROUPED is valid Go the formatter expands into separate declarations.
const GROUPED = "package a\n\nimport (\n\t\"x\"\n)\n"

// CONTRACT_MARKDOWN is the smallest REQUIREMENTS.md that breaks no rule.
const CONTRACT_MARKDOWN = "\n# Contract\n\nIt holds.\n"

// CONTRACT_TEST mirrors CONTRACT_MARKDOWN.
const CONTRACT_TEST = "package a\n\nfunc Test_Contract(t *T) {}\n"

// REQUIREMENTS_ABSENT marks a case whose package carries no such file.
const REQUIREMENTS_ABSENT = "\x00"

// MARKDOWN_PATH is where each requirements case writes its REQUIREMENTS.md.
const MARKDOWN_PATH = "package/REQUIREMENTS.md"

type setup struct {
  Completion nbio.Completion
  Retired    bool
}

type fixture struct {
  Simulation nbio.Sim
  Loop       nbio.IO
  Driver     nbio.Driver
  Output     nbio.Stream_Memory
  Errors     nbio.Stream_Memory
  Runner     harness.Runner
  Setup      setup
  Done       func() (finished bool)
}

// Builds one checked runner over fresh stores.
func checked_new(source string) (runner *Checked) {
  memory, format_state, command, directory := memory_new()
  return checked(memory, format_state, command, directory, source)
}

// Checked is one source a runner checked, and the diagnostic the check states.
type Checked struct {
  *harness.Runner
  // Diagnostic is what the check states.
  Diagnostic harness.Diagnostic
  // Syntax is the parse stores the check stands on.
  Syntax requirements.Syntax
}

// Checks one source through a runner that owns memory, as a run would after reading it.
func checked(
  memory harness.Memory,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  source string,
) (runner *Checked) {
  // A check reads no clock, and a run field is asserted whole, thus the test binds one.
  simulation := nbio.Sim{}
  clocks := make([]nbio.Sim_Clock, 1)
  nodes := make([]nbio.Sim_Node, 1)
  nodes[0].Name = make([]byte, nbio.SIM_PATH_COMPONENT_BYTES_MAXIMUM)
  nodes[0].Contents = make([]byte, NODE_CONTENTS_BYTES)
  descriptors := make([]nbio.Sim_Descriptor, 1)
  descriptors[0].Address_IP = make([]byte, nbio.IPV6_ADDRESS_BYTES)
  descriptors[0].Peer_IP = make([]byte, nbio.IPV6_ADDRESS_BYTES)
  operations := make([]nbio.Sim_Operation, 1)
  operations[0].Address_IP = make([]byte, nbio.IPV6_ADDRESS_BYTES)
  nbio.New_Simulated_IO(
    &simulation,
    0,
    time.NANOSECOND,
    nbio.Sim_Memory{
      Nodes:       nodes,
      Descriptors: descriptors,
      Operations:  operations,
      Queue:       make([]*nbio.Completion, 1),
      Events:      make([]nbio.Virtual_Event, 1),
      Clocks:      clocks,
    },
  )
  runner = &Checked{Runner: &harness.Runner{}, Syntax: syntax_new()}
  *runner.Runner = harness.Runner{
    Runner_Fields: harness.Runner_Fields{
      Read:      harness.Read{Path: read_path_new()},
      Memory:    &memory,
      Format:    format_state,
      Command:   command,
      Directory: directory,
      Clock:     nbio.Sim_Clock_To_Clock(&clocks[0]),
      Reports: harness.Reports{
        Report: harness.Report{
          Rule:    harness.RULE_TEXT_REFUSED,
          Tier:    harness.Report_Tier(harness.TIER_TWO),
          Path:    harness.PATH_TEXT_NONE,
          Message: harness.MESSAGE_OVERSIZED,
        },
        Summary: harness.Summary{
          Elapsed_Seconds: harness.ELAPSED_TEXT_NONE,
        },
      },
      Flags: harness.Run_Flag_Storage{
        Values: make(harness.Run_Flags, harness.RUN_FLAG_COUNT),
      },
    },
  }
  runner.Source_Count = harness.Source_Count(copy(memory.Source, source))
  runner.Diagnostic = harness.Check(
    runner.Memory,
    runner.Syntax,
    runner.Source_Count,
    runner.Format,
    false,
  )
  return runner
}

// Every seed gets its own tree and loop, while the run stores stay shared: the test, like package
// main, allocates them once, and the run allocates nothing.
func fixture_new(
  t *testing.T,
  memory harness.Memory,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) (subject *fixture) {
  t.Helper()
  size := Fixture_Size{Nodes: NODE_CAPACITY, Contents: NODE_CONTENTS_BYTES, Output: OUTPUT_BYTES}
  return fixture_sized(t, memory, format_state, command, directory, seed, size)
}

// Fixture_Size is how many files a fixture tree holds, the bytes of each, and of its output.
type Fixture_Size struct {
  // Nodes is how many files and directories the tree holds.
  Nodes int
  // Contents is the bytes one file holds.
  Contents int
  // Output is the bytes the output holds.
  Output int
}

// Builds one fixture over a tree of the given size.
func fixture_sized(
  t *testing.T,
  memory harness.Memory,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
  size Fixture_Size,
) (subject *fixture) {
  t.Helper()
  subject = &fixture{}
  nodes := make([]nbio.Sim_Node, size.Nodes)
  for index := range nodes {
    nodes[index].Name = make([]byte, nbio.SIM_PATH_COMPONENT_BYTES_MAXIMUM)
    nodes[index].Contents = make([]byte, size.Contents)
  }
  descriptors := make([]nbio.Sim_Descriptor, DESCRIPTOR_CAPACITY)
  for index := range descriptors {
    descriptors[index].Address_IP = make([]byte, nbio.IPV6_ADDRESS_BYTES)
    descriptors[index].Peer_IP = make([]byte, nbio.IPV6_ADDRESS_BYTES)
  }
  clocks := make([]nbio.Sim_Clock, 1)
  operations := make([]nbio.Sim_Operation, OPERATION_CAPACITY)
  for index := range operations {
    operations[index].Address_IP = make([]byte, nbio.IPV6_ADDRESS_BYTES)
  }
  subject.Loop, subject.Driver = nbio.New_Simulated_IO(
    &subject.Simulation,
    seed,
    time.NANOSECOND,
    nbio.Sim_Memory{
      Nodes:       nodes,
      Descriptors: descriptors,
      Operations:  operations,
      Queue:       make([]*nbio.Completion, OPERATION_CAPACITY*2),
      Events:      make([]nbio.Virtual_Event, 1),
      Clocks:      clocks,
    },
  )
  subject.Output = nbio.Stream_Memory{Memory: make([]byte, size.Output)}
  subject.Errors = nbio.Stream_Memory{Memory: make([]byte, OUTPUT_BYTES)}
  subject.Runner = harness.Runner{
    Runner_Fields: harness.Runner_Fields{
      Read:         harness.Read{Path: read_path_new()},
      Work_Queued:  harness.Runner_Work_Queued,
      Spread:       harness.Spread{Submit: harness.Check_Lane, Reap: harness.Lane_Oldest},
      Stopped:      harness.Runner_Stopped,
      Output:       nbio.Memory_To_Stream(&subject.Output),
      Error_Output: nbio.Memory_To_Stream(&subject.Errors),
      Clock:        nbio.Sim_Clock_To_Clock(&clocks[0]),
      Memory:       &memory,
      Format:       format_state,
      Command:      command,
      Directory:    directory,
      Reports: harness.Reports{
        Report: harness.Report{
          Rule:    harness.RULE_TEXT_REFUSED,
          Tier:    harness.Report_Tier(harness.TIER_TWO),
          Path:    harness.PATH_TEXT_NONE,
          Message: harness.MESSAGE_OVERSIZED,
        },
        Summary: harness.Summary{
          Elapsed_Seconds: harness.ELAPSED_TEXT_NONE,
        },
      },
      Flags: harness.Run_Flag_Storage{
        Values: make(harness.Run_Flags, harness.RUN_FLAG_COUNT),
      },
    },
  }
  // Built once: a closure literal inside the pump would allocate on every pass.
  subject.Done = func() (finished bool) {
    return fixture_queued(subject)
  }
  fixture_file(t, subject, harness.CONFIGURATION_PATH, CONFIGURATION)
  fixture_file(t, subject, harness.MODULE_PATH, MODULE)
  return subject
}

// The run owns the parsed file and its line index; each pass that parses takes them.
func syntax_new() (syntax requirements.Syntax) {
  tree := ast.Syntax_Tree{}
  tree.Tokens.Values = make(ast.Tokens, ast.TOKEN_COUNT_MAXIMUM)
  tree.Nodes.Values = make(ast.Nodes, ast.NODE_COUNT_MAXIMUM)
  tree.Parents.Values = make(ast.Parents, ast.DEPTH_MAXIMUM)
  tree.Last_Children.Values = make(ast.Last_Children, ast.DEPTH_MAXIMUM)
  tree.Earlier_Children.Values = make(ast.Earlier_Children, ast.DEPTH_MAXIMUM)
  tree.Errors.Values = make(ast.Errors, ast.ERROR_COUNT_MAXIMUM)
  starts := make(token.Line_Starts, token.LINE_COUNT_MAXIMUM)
  syntax.Syntax_Tree = &tree
  syntax.Lines = &token.Line_Index{Starts: starts}
  return syntax
}

func memory_new() (
  memory harness.Memory,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
) {
  print := printer.Printer{}
  print.Nodes.Values = make(printer.Node_Stack, printer.DEPTH_MAXIMUM)
  print.Pairs.Functions = make(printer.Pair_Functions, printer.PAIR_COUNT_MAXIMUM)
  print.Pairs.Heads = make(printer.Pair_Heads, printer.PAIR_COUNT_MAXIMUM)
  print.Aliases.Values = make(printer.Import_Aliases, printer.IMPORT_COUNT_MAXIMUM)
  print.Paths.Values = make(printer.Import_Paths, printer.IMPORT_COUNT_MAXIMUM)
  print.Comments.Values = make(printer.Comment_Positions, printer.COMMENT_COUNT_MAXIMUM)
  print.Counts.Values = make(printer.Counts, printer.COUNT_SLOT_COUNT)
  print.Flags.Values = make(printer.Flags, printer.FLAG_COUNT)
  source := make(printer.Source_Bytes, token.SOURCE_SIZE_MAXIMUM)
  print.Source = &printer.Source_Storage{Values: source}
  print.Form = &printer.Form_Storage{
    Values: make(printer.Form_Bytes, printer.FORM_SIZE_MAXIMUM),
  }
  formatter := format.Formatter{}
  formatter.Places.Values = make(format.Places, format.PLACE_SLOT_COUNT)
  formatter.Names.Values = make(format.Names, format.NAME_SLOT_COUNT)
  formatter.Texts.Values = make(format.Text, format.NAME_TEXT_SIZE_MAXIMUM)
  formatter.Slots.Values = make(format.Slots, format.SLOT_COUNT)
  format_state.Formatter = harness.Formatter_Stores{
    Places: formatter.Places,
    Table:  formatter.Table_Fields,
  }
  format_state.Printer = &print
  memory.Form = make(harness.Form_Bytes, harness.FORM_STORAGE_SIZE)
  memory.Paths = make(harness.Path_Table, harness.FILE_COUNT_MAXIMUM)
  memory.Directories = make(harness.Directory_Paths, harness.DIRECTORY_COUNT_MAXIMUM)
  memory.Path_Text = make(harness.Path_Text_Store, harness.WALK_TEXT_SIZE)
  memory.Directory_Text = make(harness.Directory_Text, harness.WALK_TEXT_SIZE)
  directory.Current = make(harness.Current_Path, 0, filepath.PATH_SIZE_MAXIMUM)
  memory.Entries = make(harness.Entry_Table, harness.ENTRY_COUNT_MAXIMUM)
  memory.Directory_Buffer = make(
    harness.Entry_Buffer, nbio.DIRECTORY_BUFFER_SIZE_MAXIMUM,
  )
  memory.Source = make(harness.File_Bytes, harness.SOURCE_STORAGE_SIZE)
  memory.Line = make(harness.Line_Storage, harness.LINE_SIZE_MAXIMUM)
  memory.Json = make(harness.Json_Storage, harness.JSON_STORAGE_SIZE)
  declarations := harness.ARGUMENT_DECLARATION_COUNT
  command.Declared_Arguments = make(harness.Argument_Declarations, declarations)
  command.Declared_Flags = make(harness.Flag_Declarations, harness.FLAG_DECLARATION_COUNT)
  argument_count := harness.ARGUMENT_DECLARATION_COUNT
  command.Command_Arguments = make(harness.Parsed_Arguments, argument_count)
  command.Command_Flags = make(harness.Parsed_Flags, harness.FLAG_DECLARATION_COUNT)
  command.Filled = make(harness.Filled_Marks, harness.FILLED_COUNT)
  command.Positionals = make(harness.Positional_Tokens, harness.TOKEN_COUNT_MAXIMUM)
  command.Slice_Named = make(harness.Named_Tokens, harness.TOKEN_COUNT_MAXIMUM)
  command.Failure = make(harness.Failure_Bytes, cli.FAILURE_SIZE_MAXIMUM)
  command.Help = make(harness.Help_Bytes, strings.TEXT_SIZE_MAXIMUM)
  memory.Packages = make(harness.Packages, harness.PACKAGE_COUNT_MAXIMUM)
  memory.Declaration_Pool = make(harness.Declaration_Pool, harness.DECLARATION_POOL_SIZE)
  for index := range memory.Packages {
    end := harness.DECLARATION_FIRST + requirements.HEADING_COUNT_MAXIMUM
    window := memory.Declaration_Pool[harness.DECLARATION_FIRST:end:end]
    memory.Packages[index].Declarations = requirements.Declarations(window)
  }
  memory.File_Packages = make(harness.Package_Indices, harness.FILE_COUNT_MAXIMUM)
  memory.History_Output = make(harness.History_Output, harness.HISTORY_OUTPUT_SIZE)
  memory.History_Errors = make(harness.History_Errors, harness.HISTORY_ERROR_SIZE)
  memory.History_Name = make(harness.History_Name, harness.HISTORY_NAME_SIZE)
  memory_link(&memory)
  memory_configuration(&memory)
  memory_diagnostics(&memory)
  memory.Siblings = harness.New_Siblings()
  memory.Layout = harness.New_Layout()
  memory.Reports = harness.New_Reports()
  memory.Graph = make(harness.Graph_Store, harness.GRAPH_STORE_SIZE)
  memory.Glob = make(harness.Glob_Storage, harness.GLOB_STORAGE_SIZE)
  memory.Requirements = checker_new(memory)
  memory_lanes(&memory, format_state)
  return memory, format_state, command, directory
}

// The checker shares the run's parse state and line index, since a run parses one file at a time.
func checker_new(memory harness.Memory) (checker *requirements.Checker) {
  checker = &requirements.Checker{}
  checker.Markdown = make(requirements.Markdown_Bytes, requirements.MARKDOWN_STORAGE_SIZE)
  checker.Rewritten = make(requirements.Rewritten_Bytes, requirements.MARKDOWN_STORAGE_SIZE)
  checker.Headings = make(requirements.Headings, requirements.HEADING_COUNT_MAXIMUM)
  checker.Leaves = make(requirements.Leaves, requirements.HEADING_COUNT_MAXIMUM)
  checker.Declarations = make(requirements.Declarations, requirements.HEADING_COUNT_MAXIMUM)
  checker.Name = make(requirements.Name_Bytes, requirements.NAME_SIZE_MAXIMUM)
  checker.Prior = make(requirements.Prior_Bytes, requirements.NAME_SIZE_MAXIMUM)
  checker.Message = make(requirements.Message_Bytes, requirements.MESSAGE_SIZE_MAXIMUM)
  checker.Events = make(requirements.Events, requirements.EVENT_COUNT_MAXIMUM)
  checker.Lines = make(requirements.Rewritten_Lines, requirements.LINE_TABLE_SIZE)
  checker.Path = make(requirements.Path_Bytes, requirements.PATH_STORAGE_SIZE)
  checker.Test = requirements.Test_Bytes(memory.Source)
  checker.Numbers.Values = make(requirements.Numbers, requirements.NUMBER_SLOT_COUNT)
  checker.Flags.Values = make(requirements.Flags, requirements.FLAG_SLOT_COUNT)
  return checker
}

// Runs one package holding a Go file and the given contract files, and returns what it wrote.
func requirements_run(
  t *testing.T,
  memory harness.Memory,
  format_state harness.Format,
  command harness.Command,
  walk_directory harness.Directory,
  seed uint64,
  directory string,
  source string,
  markdown string,
  test string,
  arguments ...string,
) (output string) {
  t.Helper()
  subject := fixture_new(t, memory, format_state, command, walk_directory, seed)
  fixture_directory(t, subject, directory)
  // A package main stands at the root of a binary component, which holds its func Main in pkg.
  if strings.Has_Prefix(strings.Text(source), "package main\n") {
    fixture_binary(t, subject, directory)
  }
  fixture_file(t, subject, directory+"/a.go", source)
  if markdown != REQUIREMENTS_ABSENT {
    fixture_file(t, subject, directory+"/REQUIREMENTS.md", markdown)
  }
  if test != REQUIREMENTS_ABSENT {
    fixture_file(t, subject, directory+"/requirements_test.go", test)
  }
  testify.True(t, fixture_run(subject, append([]string{"harness"}, arguments...)))
  testify.Equal(t, "", fixture_errors(subject))
  return fixture_output(subject)
}

// A run without diagnostics writes the pass line first.
func requirements_passed(t *testing.T, output string) {
  t.Helper()
  opened := strings.Has_Prefix(strings.Text(output), "✓ all checks passed\n")
  testify.True(t, bool(opened), output)
}

// Builds a package whose files sort to every outcome: a clean file at the top, a file off the
// canonical form one level down, a refused file two levels down, and a non-Go file the run must
// skip though its bytes would refuse.
func fixture_package(t *testing.T, subject *fixture) {
  t.Helper()
  fixture_package_directory(t, subject, "package")
  fixture_package_directory(t, subject, "package/nested")
  fixture_package_directory(t, subject, "package/nested/deep")
  fixture_file(t, subject, "package/a.go", CLEAN)
  fixture_file(t, subject, "package/notes.txt", REFUSED)
  fixture_file(t, subject, "package/nested/b.go", UNFORMATTED)
  fixture_file(t, subject, "package/nested/deep/c.go", REFUSED)
}

func fixture_empty(t *testing.T, subject *fixture) {
  t.Helper()
}

// A contracted package without a diagnostic, which ends in the pass line.
func fixture_clean(t *testing.T, subject *fixture) {
  t.Helper()
  fixture_package_directory(t, subject, "package")
  fixture_file(t, subject, "package/a.go", CLEAN)
}

// A package whose contract breaks format and test order rules, and whose subpackage has none.
func fixture_faulty_contract(t *testing.T, subject *fixture) {
  t.Helper()
  fixture_directory(t, subject, "package")
  fixture_directory(t, subject, "package/inner")
  fixture_file(t, subject, "package/a.go", CLEAN)
  fixture_file(t, subject, "package/inner/b.go", CLEAN)
  markdown := "Intro.\n\n# A\n\n1\n\n2\n3\n4\n\n### x y\n"
  fixture_file(t, subject, "package/REQUIREMENTS.md", markdown)
  fixture_file(t, subject, "package/requirements_test.go", "package a\n\nconst X = 1\n")
}

// A package whose contract lacks its test file.
func fixture_uncontracted(t *testing.T, subject *fixture) {
  t.Helper()
  fixture_directory(t, subject, "package")
  fixture_file(t, subject, "package/a.go", CLEAN)
  fixture_file(t, subject, "package/REQUIREMENTS.md", CONTRACT_MARKDOWN)
}

// Makes a package directory that already carries a contract, so a test of anything else reads
// only its own diagnostics.
func fixture_package_directory(t *testing.T, subject *fixture, path string) {
  t.Helper()
  fixture_directory(t, subject, path)
  fixture_file(t, subject, path+"/REQUIREMENTS.md", CONTRACT_MARKDOWN)
  fixture_file(t, subject, path+"/requirements_test.go", CONTRACT_TEST)
}

func fixture_directory(t *testing.T, subject *fixture, path string) {
  t.Helper()
  nbio.Storage_Mkdir_At(
    subject.Loop.Storage,
    fixture_submit(subject),
    nbio.DIRECTORY_CURRENT,
    path,
    DIRECTORY_PERMISSIONS,
    fixture_retired,
  )
  fixture_wait(t, subject)
}

func fixture_file(t *testing.T, subject *fixture, path string, contents string) {
  t.Helper()
  nbio.Storage_Open_At(
    subject.Loop.Storage,
    fixture_submit(subject),
    nbio.DIRECTORY_CURRENT,
    path,
    nbio.Open_At_Options{
      Access:      nbio.OPEN_WRITE_ONLY,
      Create:      true,
      Truncate:    true,
      Permissions: FILE_PERMISSIONS,
    },
    fixture_retired,
  )
  fixture_wait(t, subject)
  file := nbio.File(subject.Setup.Completion.Data)
  nbio.Storage_Write(
    subject.Loop.Storage,
    fixture_submit(subject),
    file,
    []byte(contents),
    0,
    harness.READ_TIMEOUT,
    fixture_retired,
  )
  fixture_wait(t, subject)
  testify.Equal(t, len(contents), subject.Setup.Completion.Data)
  nbio.IO_Close(subject.Loop, fixture_submit(subject), file, fixture_retired)
  fixture_wait(t, subject)
}

// Reads a whole fixture file back; a fixture file stays inside one simulated file's bytes.
func fixture_read(t *testing.T, subject *fixture, path string) (contents string) {
  t.Helper()
  nbio.Storage_Open_At(
    subject.Loop.Storage,
    fixture_submit(subject),
    nbio.DIRECTORY_CURRENT,
    path,
    nbio.Open_At_Options{Access: nbio.OPEN_READ_ONLY},
    fixture_retired,
  )
  fixture_wait(t, subject)
  file := nbio.File(subject.Setup.Completion.Data)
  buffer := make([]byte, NODE_CONTENTS_BYTES)
  nbio.Storage_Read(
    subject.Loop.Storage,
    fixture_submit(subject),
    file,
    buffer,
    0,
    harness.READ_TIMEOUT,
    fixture_retired,
  )
  fixture_wait(t, subject)
  count := subject.Setup.Completion.Data
  nbio.IO_Close(subject.Loop, fixture_submit(subject), file, fixture_retired)
  fixture_wait(t, subject)
  // A read that fills its buffer states a file wider than one a fixture writes back, thus
  // the comparison that follows would stand on bytes the file never ended at.
  if count == len(buffer) {
    t.Fatalf("a read of %s filled its buffer", path)
  }
  return string(buffer[:count])
}

func fixture_submit(subject *fixture) (completion *nbio.Completion) {
  subject.Setup.Retired = false
  return &subject.Setup.Completion
}

func fixture_retired(completion nbio.Completion_Handle) {
  (*setup)(unsafe.Pointer(completion)).Retired = true
}

func fixture_retired_read(subject *fixture) (retired bool) {
  return subject.Setup.Retired
}

func fixture_wait(t *testing.T, subject *fixture) {
  t.Helper()
  completed, err := nbio.Driver_Run_Until(
    subject.Driver,
    PUMP_DURATION,
    func() (finished bool) { return fixture_retired_read(subject) },
  )
  testify.No_Error(t, err)
  testify.True(t, completed)
  testify.No_Error(t, subject.Setup.Completion.Error)
}

// Pumps one run to its stop. It reports rather than fails, so a zero-allocation check can wrap
// it whole.
func fixture_run(subject *fixture, arguments []string) (stopped bool) {
  subject.Runner.Arguments = arguments
  // A fixture wraps the loop after it builds the runner, thus each run binds the loop anew.
  subject.Runner.IO = subject.Loop
  harness.Main(
    harness.Main_Input{
      Phase: (*harness.Opening_Phase)(&subject.Runner.Progress.Phase),
      Start: &subject.Runner.Start,
      Clock: subject.Runner.Clock,
    },
  )
  for range PASS_COUNT {
    // The pump of package main stands its stop check here, thus a stopped run
    // makes one drive pass of its own before this one reads what it left.
    if subject.Runner.Stopped(harness.Pass_Phase(subject.Runner.Progress.Phase)) {
      return true
    }
    state := harness.Streams{
      Loop:           &subject.Runner.Loop,
      Stage:          &subject.Runner.Stage,
      Walk:           &subject.Runner.Walk,
      Package_Cursor: &subject.Runner.Package_Cursor,
      Orphans:        &subject.Runner.Orphan_Cursor,
      Findings:       &subject.Runner.Findings,
      Flags:          subject.Runner.Flags,
      Files:          &subject.Runner.Files,
      File_Cursor:    &subject.Runner.File_Cursor,
      Read:           &subject.Runner.Read,
      Format:         subject.Runner.Format,
      Directory:      &subject.Runner.Directory,
      Outcome:        &subject.Runner.Outcome,
      Report:         &subject.Runner.Report,
      Summary:        &subject.Runner.Summary,
      Finding:        &subject.Runner.Finding,
      Fix_State:      &subject.Runner.Fix_State,
      Tally:          &subject.Runner.Tally,
      History:        &subject.Runner.History,
      Output:         subject.Runner.Output,
      Errors:         subject.Runner.Error_Output,
      Spread:         subject.Runner.Spread,
    }
    now := time.Clock_Now_Monotonic(subject.Runner.Clock)
    elapsed := harness.Elapsed(now - subject.Runner.Start)
    for harness.Rearm(
      harness.Rearm_Input{
        Memory:    subject.Runner.Memory,
        State:     state,
        Progress:  &subject.Runner.Progress,
        Elapsed:   elapsed,
        Arguments: subject.Runner.Arguments,
        Command:   subject.Runner.Command,
      },
    ) {
    }
    completed, err := nbio.Driver_Run_Until(subject.Driver, PUMP_DURATION, subject.Done)
    if err != nil {
      return false
    }
    if !completed {
      return false
    }
  }
  return false
}

// Reports whether one drive pass is over, which the run states of the phase it stands in.
func fixture_queued(subject *fixture) (finished bool) {
  phase := harness.Driven_Phase(subject.Runner.Progress.Phase)
  return bool(subject.Runner.Work_Queued(phase))
}

// States what a run wrote to its output, past the elapsed seconds a run with a diagnostic closes
// on, which Test_Command_Elapsed reads whole.
func fixture_output(subject *fixture) (output string) {
  output = fixture_written(subject)
  line_start := len(output) - 1
  for line_start > 0 {
    if output[line_start-1] == '\n' {
      break
    }
    line_start--
  }
  line_start = max(line_start, 0)
  for _, opening := range []string{harness.ELAPSED_OPENING, `{"passed":false,`} {
    if len(output)-line_start < len(opening) {
      continue
    }
    if output[line_start:line_start+len(opening)] == opening {
      return output[:line_start]
    }
  }
  return output
}

// States what a run wrote to its output, whole.
func fixture_written(subject *fixture) (output string) {
  return string(subject.Output.Memory[:subject.Output.Cursor])
}

func fixture_errors(subject *fixture) (errors string) {
  return string(subject.Errors.Memory[:subject.Errors.Cursor])
}

// Allocation_Count is how many heap allocations one measured run makes.
type Allocation_Count int

// ALLOCATION_NONE is the allocation count a run states: none at all.
const ALLOCATION_NONE Allocation_Count = 0

// ALLOCATION_SAMPLE_COUNT is how many samples one measure takes before it states the count. A
// run that allocates lands on every sample, thus one clean sample settles it.
const ALLOCATION_SAMPLE_COUNT = 4

// The least allocation count of a few samples. testing.AllocsPerRun reads the malloc counter of
// the whole process, thus a goroutine the test never started can land its own allocation on one
// sample; a run that allocates lands on all of them.
func allocation_least(t *testing.T, callback func()) (least Allocation_Count) {
  t.Helper()
  least = Allocation_Count(testing.AllocsPerRun(1, callback))
  for range ALLOCATION_SAMPLE_COUNT - 1 {
    if least == ALLOCATION_NONE {
      return least
    }
    least = min(least, Allocation_Count(testing.AllocsPerRun(1, callback)))
  }
  return least
}

// CROWDED_LINE_COUNT is one line past the heading table, thus a contract of that many bad lines
// states more findings than any count of headings admits.
const CROWDED_LINE_COUNT = requirements.HEADING_COUNT_MAXIMUM + 1

// CROWDED_OUTPUT_BYTES holds a line for each of those findings.
const CROWDED_OUTPUT_BYTES = CROWDED_LINE_COUNT * 256

// A contract states one finding per line it holds, thus one package states more findings than
// its heading table holds slots and the run counts every one of them.
func requirements_crowded(
  t *testing.T,
  memory harness.Memory,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
) {
  t.Helper()
  crowded := make([]byte, 0, CROWDED_LINE_COUNT*len("##\n"))
  for range CROWDED_LINE_COUNT {
    crowded = append(crowded, "##\n"...)
  }
  subject := fixture_new(t, memory, format_state, command, directory, 1)
  fixture_widen(subject, len(crowded)+1)
  subject.Output = nbio.Stream_Memory{Memory: make([]byte, CROWDED_OUTPUT_BYTES)}
  fixture_directory(t, subject, "package")
  fixture_file(t, subject, "package/a.go", CLEAN)
  fixture_file(t, subject, "package/REQUIREMENTS.md", string(crowded))
  fixture_file(t, subject, "package/requirements_test.go", CONTRACT_TEST)
  fixture_run(subject, []string{"harness", "package"})
  output := fixture_output(subject)
  found := 0
  for index := range len(output) {
    if output[index] == '\n' {
      found++
    }
  }
  testify.Equal(t, CROWDED_LINE_COUNT, found)
}

// PROPERTY_SEED_COUNT is how many contracts a sweep draws: enough that a draw states each byte
// class beside each other one, and few enough that the suite stands inside its own budget.
const PROPERTY_SEED_COUNT = 200

// PROPERTY_SIZE_MAXIMUM is the widest contract a draw writes.
const PROPERTY_SIZE_MAXIMUM = 192

// Test_Requirements_Drawn verifies no contract a draw writes leaves the run unstopped or fails a
// step: a host writes whatever bytes it likes, thus every draw states findings rather than a
// failure of the run that read it.
func Test_Requirements_Drawn(t *testing.T) {
  memory, format_state, command, directory := memory_new()
  for seed := uint64(1); seed < PROPERTY_SEED_COUNT+1; seed++ {
    generator := prng.New(prng.Seed(seed))
    markdown := drawn_markdown(&generator)
    subject := fixture_new(t, memory, format_state, command, directory, seed)
    fixture_directory(t, subject, "drawn")
    fixture_file(t, subject, "drawn/a.go", CLEAN)
    fixture_file(t, subject, "drawn/REQUIREMENTS.md", markdown)
    fixture_file(t, subject, "drawn/requirements_test.go", CONTRACT_TEST)
    testify.True(t, fixture_run(subject, []string{"harness", "drawn"}), markdown)
    testify.Equal(t, "", fixture_errors(subject), markdown)
    stated := fixture_output(subject)
    // The stores span every run, thus the same contract read on a tree of its own
    // states the same findings rather than findings the run before it left behind.
    second := fixture_new(t, memory, format_state, command, directory, seed)
    fixture_directory(t, second, "drawn")
    fixture_file(t, second, "drawn/a.go", CLEAN)
    fixture_file(t, second, "drawn/REQUIREMENTS.md", markdown)
    fixture_file(t, second, "drawn/requirements_test.go", CONTRACT_TEST)
    testify.True(t, fixture_run(second, []string{"harness", "drawn"}), markdown)
    testify.Equal(t, stated, fixture_output(second), markdown)
  }
}

// One contract of drawn bytes: the classes a heading, a body, and an ending are spelled with,
// beside the bytes a host writes that none of them admit.
func drawn_markdown(generator *prng.Xoshiro) (markdown string) {
  classes := [DRAWN_CLASS_COUNT]string{
    "#",
    " ",
    "\n",
    "\r",
    "a",
    "B",
    "_",
    "\x01",
    "ä",
    "\"",
    "9",
    "\t",
  }
  size := prng.Xoshiro_Below(generator, PROPERTY_SIZE_MAXIMUM)
  body := make([]byte, 0, PROPERTY_SIZE_MAXIMUM)
  for range size {
    drawn := prng.Xoshiro_Below(generator, DRAWN_CLASS_COUNT)
    body = append(body, classes[drawn]...)
  }
  return string(body)
}

// DRAWN_CLASS_COUNT is how many byte classes one draw picks between.
const DRAWN_CLASS_COUNT = 12

// The path of the file a run reads stands in a store the root owns, opening on a path every
// invariant of one admits.
func read_path_new() (path harness.Read_Path_Slice) {
  path = make(harness.Read_Path_Slice, 0, filepath.PATH_SIZE_MAXIMUM)
  return append(path, harness.READ_PATH_NONE...)
}

// A contract holding a gap between two body lines, then one owing its test file as flat JSON.
func requirements_contracts(
  t *testing.T,
  memory harness.Memory,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
  seed uint64,
) {
  t.Helper()
  output := requirements_run(
    t,
    memory,
    format_state,
    command,
    directory,
    seed,
    "package",
    CLEAN,
    "\n# A\n\n1\n\n2\n",
    "package a\n\nfunc Test_A() {}\n",
    "package",
  )
  testify.Equal(
    t,
    MARKDOWN_PATH+
      `:6: The section "A" has a blank line between two body lines. `+
      "Remove the blank line.\n",
    output,
  )
  output = requirements_run(
    t,
    memory,
    format_state,
    command,
    directory,
    seed,
    "package",
    CLEAN,
    CONTRACT_MARKDOWN,
    REQUIREMENTS_ABSENT,
    "package",
    "-json",
  )
  testify.Equal(
    t,
    `{"path":"package/requirements_test.go","line":0,"column":0,"tier":2,`+
      `"rule":"requirements",`+
      `"message":"The package has no requirements_test.go. Add requirements_test.go."}`+
      "\n",
    output,
  )
}

// A contract missing a blank line after its heading and holding one between its body lines: the
// finding is tier one, and -fix rewrites the file without either, thus the run then passes.
func requirements_rewritten(
  t *testing.T,
  memory harness.Memory,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
) {
  t.Helper()
  broken, test := "\n# A\nBody.\n\nMore.\n", "package a\n\nfunc Test_A() {}\n"
  output := requirements_run(
    t,
    memory,
    format_state,
    command,
    directory,
    1,
    "package",
    CLEAN,
    broken,
    test,
    "package",
    "-json",
  )
  testify.Equal(
    t,
    `{"path":"package/REQUIREMENTS.md","line":2,"column":0,"tier":1,"rule":"requirements",`+
      `"message":"There is no blank line after the heading \"A\". Write a blank line."}`+
      "\n"+
      `{"path":"package/REQUIREMENTS.md","line":5,"column":0,"tier":1,"rule":"requirements",`+
      `"message":"The section \"A\" has a blank line between two body lines. `+
      `Remove the blank line."}`+
      "\n",
    output,
  )
  subject := fixture_new(t, memory, format_state, command, directory, 1)
  fixture_directory(t, subject, "package")
  fixture_file(t, subject, "package/a.go", CLEAN)
  fixture_file(t, subject, MARKDOWN_PATH, broken)
  fixture_file(t, subject, "package/requirements_test.go", test)
  testify.True(t, fixture_run(subject, []string{"harness", "package", "-fix"}))
  requirements_passed(t, fixture_output(subject))
  testify.Equal(t, "\n# A\n\nBody.\nMore.\n", fixture_read(t, subject, MARKDOWN_PATH))
}

// HISTORY_WIDE_DESCRIPTION puts a subject that opens with "feat: " past the size cap.
const HISTORY_WIDE_DESCRIPTION = "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx" +
  "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"

// HISTORY_FAILED is the answer of a git command that failed.
const HISTORY_FAILED = "!"

// HISTORY_REFS is the main refs a clone holds.
const HISTORY_REFS = "refs/heads/main\nrefs/remotes/origin/main\n"

// History_Script answers the git commands a run spawns: the branch HEAD names, the main refs that
// exist, and the log between them.
type History_Script struct {
  Head string
  Refs string
  Log  string
}

// One log line: a commit named by one repeated hex digit, its parents, and its subject.
func history_line(name string, parents string, subject string) (line string) {
  hash := ""
  for range 40 {
    hash += name
  }
  parent := ""
  for range 40 {
    parent += parents[:1]
  }
  return hash + " " + parent + parents[1:] + "\t" + subject + "\n"
}

// Spawn_Procedure is the loop's spawn operation.
type Spawn_Procedure func(
  nbio.State,
  *nbio.Completion,
  nbio.Process_Request,
  time.Duration,
  nbio.Process_Callback,
)

// Answers each spawned git command from the script, as the git of a clone would.
func history_spawn(script History_Script) (spawn Spawn_Procedure) {
  return func(
    _ nbio.State,
    completion *nbio.Completion,
    request nbio.Process_Request,
    _ time.Duration,
    callback nbio.Process_Callback,
  ) {
    history_answer(script, completion, request, callback)
  }
}

// Answers one spawned git command from the script.
func history_answer(
  script History_Script,
  completion *nbio.Completion,
  request nbio.Process_Request,
  callback nbio.Process_Callback,
) {
  answer := script.Log
  switch request.Arguments[0] {
  case "rev-parse":
    answer = script.Head
  case "for-each-ref":
    answer = script.Refs
  }
  if answer == HISTORY_FAILED {
    callback(completion, nbio.Process_Result{Exit: 128}, nil)
    return
  }
  count := copy(request.Output_Storage, answer)
  callback(completion, nbio.Process_Result{Output: request.Output_Storage[:count]}, nil)
}

// Makes a symbolic link at the path, pointing at the target: a file holding the target, turned
// into the link the host would make of it.
func fixture_link(t *testing.T, subject *fixture, path string, target string) {
  t.Helper()
  fixture_file(t, subject, path, target)
  name := path[len(path)-len(path[fixture_last_separator(path)+1:]):]
  for index := range subject.Simulation.Nodes {
    node := &subject.Simulation.Nodes[index]
    if !node.Used {
      continue
    }
    if string(node.Name[:node.Name_Count]) != name {
      continue
    }
    node.Mode = node.Mode&^nbio.FILE_MODE_TYPE | nbio.FILE_MODE_SYMBOLIC_LINK
    return
  }
  t.Fatalf("no node holds %s", path)
}

// Reports where the last separator of a path stands.
func fixture_last_separator(path string) (index int) {
  for index = len(path) - 1; index > 0; index-- {
    if path[index] == '/' {
      return index
    }
  }
  return -1
}

// Builds the stores the walk resolves symbolic links in.
func memory_link(memory harness.Memory_Handle) {
  memory.Orphans = make(harness.Orphan_Paths, harness.ORPHAN_COUNT_MAXIMUM)
  for index := range memory.Orphans {
    memory.Orphans[index] = make(filepath.Slice, 0, filepath.PATH_SIZE_MAXIMUM)
  }
  memory.Link = make(harness.Link_Path, filepath.PATH_SIZE_MAXIMUM)
  memory.Resolved = make(harness.Resolved_Path, filepath.PATH_SIZE_MAXIMUM)
  memory.Target = make(harness.Link_Target, harness.LINK_TARGET_SIZE)
}

// MODULE is the go.mod a fixture tree holds at its root.
const MODULE = "module local/fixture\n\ngo 1.26\n"

// CONFIGURATION is the smallest harness.json a run decodes.
const CONFIGURATION = `{"shared_component":"package","instrumentation_packages":[],` +
  `"opt_out_assertion_mandate_packages":[],` +
  `"opt_out_recursion_ban":[],"opt_out_interface_ban":[],"ignore":[],"word_replacements":{"util":[]}}`

// States the smallest harness.json whose shared component is the shared directory, which the
// gateways of the stdlib ports stand under.
func shared_configuration() (source string) {
  return strings_replace(
    CONFIGURATION,
    `"shared_component":"package"`,
    `"shared_component":"shared"`,
  )
}

// Makes the component at the root a binary one: the shared component is the shared directory, and
// the component holds its func Main in a pkg package of its own contract.
func fixture_binary(t *testing.T, subject *fixture, root string) {
  t.Helper()
  fixture_file(t, subject, harness.CONFIGURATION_PATH, shared_configuration())
  fixture_package_directory(t, subject, root+"/pkg")
  fixture_file(t, subject, root+"/pkg/p.go", "package pkg\n\nfunc Main() {}\n")
}

// Writes the contract test of a default directory under the package clause its parent names.
func fixture_default_contract(t *testing.T, subject *fixture, path string, clause string) {
  t.Helper()
  fixture_file(
    t,
    subject,
    path+"/requirements_test.go",
    "package "+clause+"\n\nfunc Test_Contract(t *T) {}\n",
  )
}

// Runs the harness over a tree of the given files, the shared component the shared directory, and
// expects each given line in what it wrote and none of the absent fragments.
func layout_expect(t *testing.T, root string, sources []string, lines []string, absent []string) {
  t.Helper()
  memory, format_state, command, directory := memory_new()
  subject := fixture_new(t, memory, format_state, command, directory, 1)
  fixture_file(t, subject, harness.CONFIGURATION_PATH, layout_configuration())
  made := map[string]bool{}
  for index := 0; index < len(sources); index += 2 {
    path := sources[index]
    for cut := range len(path) {
      if path[cut] != '/' {
        continue
      }
      if made[path[:cut]] {
        continue
      }
      made[path[:cut]] = true
      fixture_directory(t, subject, path[:cut])
    }
    fixture_file(t, subject, path, sources[index+1])
  }
  testify.True(t, fixture_run(subject, []string{"harness", root}))
  written := fixture_written(subject)
  for _, line := range lines {
    testify.True(t, bool(strings.Contains(strings.Text(written), strings.Text(line+"\n"))), written)
  }
  for _, fragment := range absent {
    testify.False(t, bool(strings.Contains(strings.Text(written), strings.Text(fragment))), written)
  }
}

// States the harness.json of a layout tree: the shared directory is the shared component, and
// the probe tree an instrumentation package.
func layout_configuration() (source string) {
  return strings_replace(
    shared_configuration(),
    `"instrumentation_packages":[]`,
    `"instrumentation_packages":["shared/probe/**"]`,
  )
}

// Takes the file at the path out of the tree, the way a host that never wrote it would hold none.
func fixture_remove(t *testing.T, subject *fixture, path string) {
  t.Helper()
  for index := range subject.Simulation.Nodes {
    node := &subject.Simulation.Nodes[index]
    if !node.Used {
      continue
    }
    if string(node.Name[:node.Name_Count]) != path {
      continue
    }
    node.Used = false
    return
  }
  t.Fatalf("no node holds %s", path)
}

// Widens every node of the tree to the size, keeping the bytes each one holds.
func fixture_widen(subject *fixture, size int) {
  for index := range subject.Simulation.Nodes {
    node := &subject.Simulation.Nodes[index]
    widened := make([]byte, size)
    copy(widened, node.Contents)
    node.Contents = widened
  }
}

// Builds the stores of the diagnostics the declaration pass leaves, and of the notice each states.
func memory_diagnostics(memory harness.Memory_Handle) {
  memory.Notice = make(harness.Notice_Store, harness.MESSAGE_SIZE_MAXIMUM)
  memory.Diagnostics = make(harness.File_Diagnostics, harness.FILE_COUNT_MAXIMUM)
  memory.Sizes = make(harness.File_Sizes, harness.SIZE_TABLE_SIZE)
  memory.Order = make(harness.File_Order, harness.FILE_COUNT_MAXIMUM)
}

// Builds the lane table, which views every other store, thus past them.
func memory_lanes(memory harness.Memory_Handle, format_state harness.Format) {
  memory.Checked = make(harness.Lane_Checks, harness.LANE_COUNT)
  memory.Lanes, memory.Held = make(harness.Lane_Table, harness.LANE_COUNT), make(
    harness.Lane_Marks,
    harness.LANE_COUNT,
  )
  harness.New_Lanes(memory, format_state)
}

// Builds the stores harness.json decodes into.
func memory_configuration(memory harness.Memory_Handle) {
  memory.Configuration = &configuration.Configuration{
    Tokens: make(configuration.Tokens, configuration.TOKEN_COUNT_MAXIMUM),
    Text:   make(configuration.Text, configuration.SOURCE_SIZE_MAXIMUM),
    Values: make(configuration.Values, configuration.VALUE_COUNT_MAXIMUM),
    Cells:  make(configuration.Cells, configuration.CELL_COUNT),
  }
}

// The configuration with the empty list under the key replaced by the entries.
func configuration_with(source string, key string, entries string) (replaced string) {
  return strings_replace(source, "\""+key+"\":[]", "\""+key+"\":"+entries)
}

// Replaces the first text in the source with another.
func strings_replace(source string, old string, new string) (replaced string) {
  for index := 0; index+len(old) < len(source)+1; index++ {
    if source[index:index+len(old)] == old {
      return source[:index] + new + source[index+len(old):]
    }
  }
  return source
}

// WIDE_COMPONENT_BYTES is one wide name of a path, inside what one simulated name holds.
const WIDE_COMPONENT_BYTES = 200

// WALK_FILE_COUNT is how many Go files of wide paths outgrow the walk text store.
const WALK_FILE_COUNT = harness.WALK_TEXT_SIZE/(4*WIDE_COMPONENT_BYTES) + 1

// WALK_NODE_COUNT holds those files, their directories, and the fixture's own files.
const WALK_NODE_COUNT = WALK_FILE_COUNT + NODE_CAPACITY

// WIDE_CONTENTS_BYTES holds a Go file as wide as a run reads.
const WIDE_CONTENTS_BYTES = requirements.SOURCE_SIZE_MAXIMUM

// REPORT_OUTPUT_BYTES holds a line for each slot of the report table.
const REPORT_OUTPUT_BYTES = harness.REPORT_COUNT_MAXIMUM * 256

// DECLARED_NODE_COUNT holds the tree of the declaration case: a directory, a contract, and a test
// file for each package past the declaration table.
const DECLARED_NODE_COUNT = 3*
  (harness.DECLARATION_COUNT_MAXIMUM/requirements.HEADING_COUNT_MAXIMUM+1) +
  NODE_CAPACITY

// DECLARED_CONTENTS_BYTES holds one test file of HEADING_COUNT_MAXIMUM declarations.
const DECLARED_CONTENTS_BYTES = 64 * requirements.HEADING_COUNT_MAXIMUM

// Builds one function declaring, in one statement, clean locals of WIDE_NAME_WORDS words each
// until their bytes pass the given size.
func locals_source(size int) (source string) {
  built := []byte("func G() {\n  ")
  spelled := 0
  for index := 0; spelled < size+1; index++ {
    if index > 0 {
      built = append(built, ", "...)
    }
    opening_size := len(built)
    for range WIDE_NAME_WORDS {
      built = append(built, "table_"...)
    }
    built = append(built, decimal(index)...)
    spelled += len(built) - opening_size
  }
  return string(append(built, " := 0\n}\n"...))
}

// WIDE_NAME_WORDS is how many words one wide name spells, thus few names fill the sibling text.
const WIDE_NAME_WORDS = 160

// Builds top-level variables of the given count of names, each the word and a number, a few
// names to a declaration.
func variables_source(word string, count int) (source string) {
  built := []byte{}
  for index := range count {
    separator := ", "
    if index%VARIABLE_NAME_COUNT == 0 {
      separator = "var "
    }
    built = append(built, separator...)
    built = append(built, word...)
    built = append(built, "_"...)
    built = append(built, decimal(index)...)
    if index%VARIABLE_NAME_COUNT == VARIABLE_NAME_COUNT-1 {
      built = append(built, " int\n"...)
    }
  }
  if count%VARIABLE_NAME_COUNT != 0 {
    built = append(built, " int\n"...)
  }
  return string(built)
}

// VARIABLE_NAME_COUNT is how many names one declaration of variables_source holds.
const VARIABLE_NAME_COUNT = 32

// Builds a test file of the given count of declarations, each a type, the fewest tokens one
// declaration spends.
func declared_source(count int) (source string) {
  built := []byte("package a\n\n")
  for index := range count {
    built = append(built, "type T_"...)
    built = append(built, decimal(index)...)
    built = append(built, " int\n"...)
  }
  return string(built)
}

// Spells a count in decimal digits.
func decimal(value int) (text string) {
  digits := []byte{byte('0' + value%10)}
  for value > 9 {
    value /= 10
    digits = append([]byte{byte('0' + value%10)}, digits...)
  }
  return string(digits)
}

// Counts the times one text stands in another.
func strings_count(text string, part string) (count int) {
  for index := 0; index+len(part) < len(text)+1; index++ {
    if text[index:index+len(part)] == part {
      count++
    }
  }
  return count
}

// CLOSURE_LOGIC is a file whose closure declares a local before it calls.
const CLOSURE_LOGIC = "package a\n\nfunc f() {\n  g(func() {\n    x := 1\n    h(x)\n  })\n}\n"

// UNNAMED_RESULT is a function past CLOSURE_LOGIC whose result is unnamed.
const UNNAMED_RESULT = "\nfunc k() int {\n  return 1\n}\n"

// A gateway named by a relative path, with its leading current directory or none, stays exempt.
func gateway_relative_path(t *testing.T) {
  t.Helper()
  memory, format_state, command, directory := memory_new()
  subject := fixture_new(t, memory, format_state, command, directory, 1)
  fixture_file(t, subject, harness.CONFIGURATION_PATH, shared_configuration())
  fixture_directory(t, subject, "shared")
  fixture_directory(t, subject, "shared/std")
  fixture_directory(t, subject, "shared/std/os")
  fixture_package_directory(t, subject, "shared/std/os/default")
  fixture_default_contract(t, subject, "shared/std/os/default", "os")
  fixture_file(t, subject, "shared/std/os/default/host.go", "package os\n\nimport \"syscall\"\n")
  for _, path := range []string{"shared/std/os/default", "./shared/std/os/default"} {
    testify.True(t, fixture_run(subject, []string{"harness", path, "-fix"}))
    output := fixture_output(subject)
    testify.False(t, bool(strings.Contains(strings.Text(output), "Banned import")), path, output)
    testify.Equal(t, harness.EXIT_SUCCESS, subject.Runner.Status, path, output)
  }
}

// Takes the escapes of a flat JSON message back out of it.
func unescaped(message string) (text string) {
  built := make([]byte, 0, len(message))
  for index := 0; index < len(message); index++ {
    if message[index] == '\\' {
      index++
    }
    built = append(built, message[index])
  }
  return string(built)
}

// Held_Lanes is a spread that holds the lanes it is handed: the lanes it holds unchecked, the most
// it held at once, and whether it checks each lane as it is handed one. One that does not checks
// nothing until the run reaps a lane, then checks every lane it holds, the last handed first, and
// hands back the lane the run read last, thus the run closes its lanes out of file order.
type Held_Lanes struct {
  // Lanes holds the lanes handed over and not yet checked.
  Lanes []harness.Lane_Handle
  // Deepest is the most lanes held at once.
  Deepest int
  // Checked is whether each lane is checked as it is handed over.
  Checked bool
  // Submitted is how many lanes the run handed over.
  Submitted int
}

// Takes one lane into the spread.
func held_submit(held *Held_Lanes, lane harness.Lane_Handle) {
  held.Lanes = append(held.Lanes, lane)
  held.Submitted++
  held.Deepest = max(held.Deepest, len(held.Lanes))
  if held.Checked {
    harness.Check_Lane(lane)
    held.Lanes = held.Lanes[:0]
  }
}

// Checks every lane the spread holds, the last handed over first, and hands back a lane the run
// holds and has not handed back: the one it read first when the spread checks each lane as it
// comes, the one it read last when it does not.
func held_reap(held *Held_Lanes, memory harness.Memory_Handle) (lane harness.Lane_Index) {
  for index := len(held.Lanes) - 1; index > -1; index-- {
    harness.Check_Lane(held.Lanes[index])
  }
  held.Lanes = held.Lanes[:0]
  if held.Checked {
    return harness.Lane_Oldest(memory)
  }
  newest := -1
  for index := range memory.Lanes {
    if !bool(memory.Held[index]) {
      continue
    }
    if bool(memory.Checked[index]) {
      continue
    }
    if newest < 0 {
      newest = index
    }
    if memory.Lanes[index].Slot > memory.Lanes[newest].Slot {
      newest = index
    }
  }
  return harness.Lane_Index(newest)
}

// ORDER_COUNT is how many orders the lanes test checks a run's lanes in: submitted, and reversed.
const ORDER_COUNT = 2

// One package of a file past the lane table: the first spells the name the last declares, and
// every file between is clean.
func lanes_fixture(
  t *testing.T,
  memory harness.Memory,
  format_state harness.Format,
  command harness.Command,
  directory harness.Directory,
) (subject *fixture) {
  subject = fixture_new(t, memory, format_state, command, directory, 1)
  fixture_package_directory(t, subject, "package")
  fixture_file(t, subject, "package/a.go", REFERENCE)
  for index := 1; index < harness.LANE_COUNT; index++ {
    fixture_file(t, subject, "package/"+string(rune('a'+index))+".go", CLEAN)
  }
  last := string(rune('a'+harness.LANE_COUNT)) + ".go"
  fixture_file(t, subject, "package/"+last, DECLARATION)
  return subject
}
