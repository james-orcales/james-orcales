
# Diagnostics

Each diagnostic names one file and stands in one of two tiers.

### Tier One

Tier one diagnostic is auto-fixable. It carries the fix: the whole file as it should stand.

### Tier Two

Tier two diagnostic is not auto-fixable. It carries no fix.

### Every Finding

A Go file whose form the check accepts states each finding of each rule, by line and column, then
in the order found. A file the parser refuses states each refusal its parse logged, the same way.

# Command

### Usage

`shared/std/cli` parses `harness <directory> [-json] [-fix]`; `<directory>` is one Go package. A refused
command line writes `harness: <error>` and exits 1, each byte below a space as `?` and the error cut
at the line store, since it names what a host typed. `-h` writes help and exits 0.

### Vector

A host hands the run its own argument vector, and one holding no argument at all names no
program, thus the run writes `harness: the argument vector names no program` and exits 1.

### Configuration

A run decodes `harness.json` by `harness/pkg/configuration` before the walk; an unreadable file
fails that step. An undecodable one writes `harness: harness.json: [<key> ]<fault>[: "<text>"][ at
byte <offset>]`, the byte as `shared/std/encoding/json` states it, and exits 1.

### Configuration Paths

Past the packages, a wildcard-free path list entry names a directory with a trailing slash and a
file without, as the host states it; a package list names no file. Each other entry, and one naming
nothing, is tier two at `harness.json`, rule `slashes`, in lint's words and list order.

### Ignore

A path an `ignore` glob of `harness.json` matches, as lint matches one, stands outside the run: a
directory the walk never opens, and a file it never checks.

### Argument

A trailing separator names the same directory as none, and a path of separators alone names none
at all. An argument of no bytes names nothing to walk, thus the run fails step `walk`. One holding
a byte below a space fails the same way.

### Entries

A directory entry whose name holds a byte below a space names a file no line can state, thus the
walk fails rather than writes a line that breaks in two. A name of any other byte stands,
whichever bytes a host spent on it.

### Symlinks

A symbolic link under `<directory>` whose target resolves to nothing, or past eight links,
yields tier two diagnostic `The symlink's target is absent. Remove the symlink or restore its
target.`, rule `symlinks`. The walk reads a link's target and follows no link into the tree.

### Recursive

Command checks each Go file under `<directory>`, at any depth, writing one line per diagnostic in
path order: path, then message. An entry that is no regular file names neither source nor
contract, thus the walk passes it over. A directory too deep for its contract fails step `walk`.

### Json

With `-json`, each diagnostic is one line of `shared/std/encoding/flatjson`: `path`, `line`, `column`,
`tier`, `rule`, `message`. A diagnostic without a position has line and column 0.

### Fix

With `-fix`, the fix of each tier one diagnostic replaces its file, and the diagnostic is neither
written nor counted. A tier two diagnostic stays. A failed open, write, or close fails step `fix`,
and a write that stops part way leaves the file holding what it wrote and no more.

### Requirements

`harness/pkg/requirements` checks each package's contract after the Go files, rule `requirements`.
A blank line missing around a heading or standing between body lines is tier one, fixed by
`requirements.Rewrite`; every other finding is tier two. An unreadable contract file states itself.

### History

Past the packages, a run off main and master judges each first-parent commit from the main ref
to HEAD by `harness/pkg/vcs`: tier two, rule `commits`, at `<git:hash>`. A missing main ref, or a
history past its storage, states itself at `<git:main>` or `<git:log>`; a failed git states nothing.

### Quotes

A message that quotes a file the run read writes each byte below a space as `?`, since the file is
untrusted and such a byte steers a terminal or breaks the line that holds it. A quote it holds
stands escaped in flat JSON, thus the object parses as one object.

### Status

A run with no diagnostic exits 0; one with any diagnostic or failed step exits 1. A failed step
writes `harness: failed to <step>: <path>`, the file being read, or at step `walk` the open
directory and why: `: more than 2048 Go files`, the argument, a name, or the host's refusal.

### Partial

A failed step retracts nothing: every diagnostic the run wrote before it stands, and so does
every fix it wrote to a file.

### Passed

A run with no diagnostic writes `✓ all checks passed`, then `elapsed_sec=<s.mmm>` from the clock.
With `-json` it writes `{"passed":true,"elapsed_sec":"<s.mmm>"}`, a string since flatjson holds no
float.

### Elapsed

A run with a diagnostic writes `elapsed_sec=<s.mmm>` past its findings, and with `-json`
`{"passed":false,"elapsed_sec":"<s.mmm>"}`. A failed step writes none.

# Format

### Clean

Go file that stands in the canonical form of `shared/std/go/format` yields no diagnostic.

### Unformatted

Go file that does not stand in the canonical form yields tier one diagnostic `not formatted`. Fix
is the canonical form. A file whose lines end with a carriage return stands outside that form, and
one opening with a byte order mark opens no package clause, thus the parser refuses it.

### Refused

Go file the parser refuses yields tier two diagnostic, since a refused tree has no fix. Its line
is `<path>:<line>:<column>: <message>`: where the refused token stands, then the parser's message.
A file of more lines than one index holds names no position, thus its line states the path alone.

### Test Closures

A _test.go file's closure may hold logic: its refusals past its closure bodies alone refuse it.

### Indentation

The canonical form opens a line with two spaces for each depth it stands under, thus a file
indented with tabs is not formatted, and so is one that ends without a newline.

### Names

A run declares the names of every Go file before it checks any, thus a name renames in every file
that spells it. A run holding a refused or oversized file, or more names than the table holds,
renames nothing, because a rename it cannot finish breaks the build.

### Oversized

Go file past `requirements.SOURCE_SIZE_MAXIMUM` yields tier two diagnostic `exceeds source size`.

### Conflicted

Go file with a line that opens on a merge conflict marker, seven repeats of `<`, `>`, `%`, `+`,
or a backslash, yields tier two diagnostic `The file has a conflict marker. Resolve the conflict
and remove the marker.`, rule `conflicted`, at that line and column 1.

### Unprintable

Go file whose canonical form outgrows the printer's storage yields tier two diagnostic `canonical
form exceeds its storage`, since indentation spends bytes the source never held.

### Imports

A clean Go file importing a stdlib package lint bans yields tier two diagnostic `Banned import
"<path>" belongs to stdlib family "<family>". Use shared package.`, rule `imports`, at each such
import, with lint's families and gateway exemptions.

### Name Rules

A clean Go file breaking a rule of `harness/pkg/naming`, over the word table of `harness.json`,
yields tier two diagnostic with its message, rule `names`, at each name, beside each banned
import the file makes.

### Shadows

A clean Go file whose function declares a name its scopes, or a sibling file of its package clause,
declare yields tier two diagnostic `The variable "<name>" shadows a variable in an outer scope.
Write a different name.`, rule `shadows`, there. A package the run cannot record judges no sibling.

### Recursion

A Go file whose bare-name call closes a cycle among its package's functions yields tier two
diagnostic `The function <f> recurses. Write a loop instead.`, or for a longer one `The functions
recurse in a cycle. Write a loop instead. <f> → <g> → <f>`, rule `recursion`, past exempt files.

### Lanes

The root's spread checks up to `LANE_COUNT` Go files at once, in any order, by no name table;
names and purity are taken in file order. A table that took a name renames the bytes of each file
it can reach, parsing none again, thus a run states what one file at a time states.

# Layout

The repository is one Go module, and each top-level directory of Go code is a component: the
shared one `harness.json` names, and each other one a binary. A finding is tier two, rule `layout`.

### Single Module

A run reads `go.mod` past `harness.json`: none states `There is no go.mod at the repository root.
The linter needs the module anchor.`, and the walk states each nested `go.mod` and each `go.work`.

### Shared Component

The shared component holds no internal directory, stating `A shared library permits no internal/
directory. Remove "<directory>".`, and no package main, stating `The shared library "<import>"
permits no package main. Move the entry point to a binary component.`

### Binary Component

A binary component holds each package but main under its top-level pkg directory, else the file
states `Move <directory> -> <component>/pkg/<directory>.`

### Binary Isolation

A binary component imports no package of another, else the import states `The binary component
"<import>" imports the binary component "<import>". Move the shared code to the shared component.`

### Main Package

A binary component's package main stands at its root, else the file states `The binary component
"<import>" has its main package at "<directory>". Move the main package to the component root.`

### Entry Point

A binary component the walk covers whole declares one func Main in its pkg directory, else it
states `The binary component "<import>" declares no func Main in pkg/.`, or more than one, and why.

### Tier Depth

A package, no test and no main, stands under one library ancestor at most, a v<N> segment and a
binary's pkg aside: `The package "<p>" at "<dir>" is below the library tier. It has <n> ...`

### Impure Imports

A pure file, no test, main, composition tier, or aver file, imports no os, os/exec, os/user,
os/signal, flag, runtime, math/rand, or crypto/rand: `The stdlib import "<path>" is impure. ...`

### Impure Calls

Such a file selects no clock, stdout, or network API lint lists, each of time, fmt, net, and
net/http, else it states `The stdlib call <path>.<name> is impure. See lint/README.md ...`

### Transitive Purity

A pure package imports no default directory or composition tier of the shared component, but an
instrumentation package: `The dependency "<path>" is impure. A pure package imports only ...`

### Transitive Stdlib

A pure package, its tests included, selects no API lint lists as reaching an impure one, else it
states `The transitive call <path>.<name> is impure. A pure package calls only pure APIs.`

### Binary Purity

A binary component's package main alone stands impure, thus the purity rules bind each package
under its pkg directory whatever its depth, and release its package main.

### Binary Default Tier

A binary component holds no default directory, else the file states `A binary component permits
no default tier. Bind the real world in package main. Remove "<directory>".`

### Library Purity

A shared package stands pure but for a default directory and the composition tier, whose files
alone the purity rules release.

### Default Package Impurity

A default directory declares its parent's package, its _test aside, else the clause states `The
default package declares "package <c>", not "package <p>". The default package shadows ...`

# Memory

A run holds 150 MiB. A store that fills states what it could not hold, as tier two rule `capacity`
at the file, and the run fails.

### Budget

`New_Memory` allocates at most `MEMORY_SIZE_MAXIMUM` bytes: the 150 MiB less the loop's stores
and the runtime's own. The run lends shared/os no self-exec workspace, since it never replaces
its image.

### Packages

A run checks its Go files package by package, and judges a package's siblings and call graph once
its last file closes. The next package reuses the stores, thus a run holds one package's names.

### Capacity

A file past the sibling, record, scope, naming finding, report, or declaration table, or the line
index, states `The file outgrows the <table>.` or `The run outgrows the report table.` and what it
leaves undone. A tree past a walk text store fails at `walk`: `: more than 131072 path bytes held`.

# Allocation

`New_Memory` allocates every store once, before the run starts. After that a run allocates
zero heap bytes.
