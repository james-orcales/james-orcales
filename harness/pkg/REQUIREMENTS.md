
# Diagnostics

Each diagnostic names one file and stands in one of two tiers.

### Tier One

Tier one diagnostic is auto-fixable. It carries the fix: the whole file as it should stand.

### Tier Two

Tier two diagnostic is not auto-fixable. It carries no fix.

# Command

### Usage

`shared/cli` parses `harness <directory> [-json] [-fix]`; `<directory>` is one Go package. A refused
command line writes `harness: <error>` and exits 1, each byte below a space as `?` and the error cut
at the line store, since it names what a host typed. `-h` writes help and exits 0.

### Vector

A host hands the run its own argument vector, and one holding no argument at all names no
program, thus the run writes `harness: the argument vector names no program` and exits 1.

### Configuration

A run decodes `harness.json` in its working directory by `harness/pkg/configuration` before the
walk. An unopenable or unreadable file fails that step; an undecodable one writes `harness:
harness.json: [<key> ]<fault> at byte <offset>`, keyed for absent keys and globs, and exits 1.

### Configuration Paths

Past the packages, a wildcard-free path list entry names a directory with a trailing slash and a
file without, as the host states it; a package list names no file. Each other entry, and one naming
nothing, is tier two at `harness.json`, rule `slashes`, in lint's words and list order.

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

With `-json`, each diagnostic is one line of `shared/encoding/flatjson`: `path`, `line`, `column`,
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

# Format

### Clean

Go file that stands in the canonical form of `shared/go/format` yields no diagnostic.

### Unformatted

Go file that does not stand in the canonical form yields tier one diagnostic `not formatted`. Fix
is the canonical form. A file whose lines end with a carriage return stands outside that form, and
one opening with a byte order mark opens no package clause, thus the parser refuses it.

### Refused

Go file the parser refuses yields tier two diagnostic, since a refused tree has no fix. Its line
is `<path>:<line>:<column>: <message>`: where the refused token stands, then the parser's message.
A file of more lines than one index holds names no position, thus its line states the path alone.

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
"<path>" belongs to stdlib family "<family>". Use shared package.`, rule `imports`, at its first
such import, with lint's families and gateway exemptions.

# Allocation

`New_Memory` allocates every store once, before the run starts. After that a run allocates
zero heap bytes.
