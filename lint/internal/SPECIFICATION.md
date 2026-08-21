
# Diagnostics

A diagnostic is tier one or tier two. Tier one always prints; any tier-one anywhere in scope
suppresses tier two, which prints only when no tier-one fired and so may rely on tier-one
contracts.

### Tier One

Every per-file rule not listed under Tier Two.

### Tier Two

Unbounded Read, Unbounded Decode, Unbounded Decompression, Unbounded Allocation, Unbounded Http,
Deprecated Ioutil, Self Recursion, Mutual Recursion, Init Functions, Mutable Globals, Methods,
Struct Tags.

# Repository

These rules govern every tracked file, whatever its language.

### Ignored Directories

The linter never scans the top-level `third_party/` drop-zone, any `vendor/`, `.jj`, `.git`, or a
gitignored path. A nested `third_party/` below the repository root is first-party code, scanned
like anything else.

### Path Casing

Every tracked path segment, split on its dots, is snake_case, Ada_Case, or SCREAMING_SNAKE_CASE;
underscores join words, never hyphens. A vendored tree — any path with a `third_party` or `vendor`
segment — is exempt.

### Symlinks

A tracked symlink resolves to a tracked target: a tracked file, or a directory that holds
tracked files. A target outside the tracked set, or the symlink's own path, is banned. An
untracked symlink is not checked.

### Banned Script Extensions

A file is banned by a scripting extension: .py .sh .bash .zsh .fish .ksh .csh .pl .pm .rb .lua .tcl
.awk .ps1 .psm1 .bat .cmd .vbs .groovy .r .jl.

### Banned Build Files

A file is banned by base name: makefile, gnumakefile, rakefile, gemfile, pipfile, justfile, or
taskfile.

### Banned Archives

An `.xz` file is banned: the stdlib cannot decompress it, forcing external tar, whose GNU and BSD
builds diverge. Use gzip, read directly by compress/gzip and archive/tar.

### Conflict Markers

No line begins with a merge conflict marker: seven repeats of <, >, %, +, or backslash, the
sentinels a git/jujutsu writes into a file when it cannot auto-merge.

### Github Actions

In a .github/workflows YAML file, a `uses:` line is banned; rewrite the step as an inline run.
Third-party actions are considered a dependency.

# Markdown

These rules govern every tracked Markdown file; the Line Width cap also binds source lines.

### Line Width

A line spans at most 100 display columns, a wide rune counting two, a tab expanding to the next
eight-column stop, a combining or zero-width mark none. Outside fenced code a sole URL or a table
row is exempt (neither can wrap), as is a source line in a backtick raw string literal.

### Whitespace

A Markdown prose line carries no trailing whitespace; fenced code is exempt.

### Agent Documentation Size

A CLAUDE.md, AGENTS.md, or SKILL.md spans at most 100 lines.

### Agent Documentation Pairing

AGENTS.md and CLAUDE.md exist as a byte-identical pair in one directory, at the repository root or
one level below.

# Component Layout

The repo is one Go module; each top-level directory of Go code is a component. The layout splits
pure from impure. Shared component: `shared/foo (pure) -> shared/foo/default (impure)`. Binary
component: `main (impure) -> internal (pure)`.

### Single Module

The repo is one Go module: exactly one `go.mod`, at the root, with no `go.work` and no nested
`go.mod`. A component is a top-level directory of Go source, not a module of its own.

### Shared Component

The shared component, `shared`, has no `internal` directory at any depth and declares no
`package main`. It is fully importable, never executed. Its root-relative directory is
lint.json's `shared_component`.

### Binary Component

In a binary component, every non-main package lives under a top-level internal directory; code at
the component root or any non-internal directory is banned.

### Main Package

A binary component holds exactly one main package, and it sits at the component root; a `cmd`
directory is banned.

### Internal Entry Point

A binary component declares exactly one free func Main in its top-level internal directory; it is
the entry point its thin package main calls, and zero or several Main functions are banned. A shared
component may also expose a func Main, representing an embeddable entry point.

### Tier Depth

A non-main package has at most one non-main package above it, not counting `v{N}` dirs or, in a
binary, the top-level internal dir. So the deepest nest is `shared/foo/[v{N}]/default`, a library
package then its impure default tier, or `binary/internal/foo/[v{N}]/bar`, both pure.

### Impure Imports

A pure package may not import os, os/exec, os/user, os/signal, flag, runtime, math/rand, or
crypto/rand; inject these instead.

### Impure Calls

A pure package may not call the clock, stdout, or network via time, fmt, net, or net/http
(time.Now, fmt.Print, http.Get, net.Dial); inject these instead.

### Transitive Purity

A pure package imports only pure packages, never an impure one; purity then holds across the whole
import closure by induction, with no graph walk. It may import a package listed in lint.json's
`instrumentation_packages` — a write-only side channel the ban exempts.

### Transitive Stdlib

A pure package calls no curated stdlib API that transitively reaches the impure set; this binds its
test files too, so an impure reach is caught even in a _test.go file.

### Binary Purity

A binary module keeps its impurity in package main and its internal packages pure. Tests may be
impure.

### Binary Default Tier

A binary component declares no default package. Only its package main is impure, so a package
under internal gets no release from the purity bans, whatever its depth.

### Library Purity

A shared library keeps its packages pure; each package is wholly pure or wholly impure, never
mixed.

### Default Package Impurity

A shared library may add one `default` package — a `default/` directory declaring its parent's
package clause — as the home for any impure global API; it is the only impure package allowed.

# Source And Test Bans

These constructs are forbidden in source and test files alike.

### Dot Imports

An import binds a package name; dot imports are banned. Dot imports pollute the global namespace
and complicates type resolution.

### Blank Imports

An import names a package a caller uses; blank imports are banned. Blank imports indicate that
a library is being imported solely for it's global side-effects, an implicit control flow.

### Banned Imports

Outside `shared/sim/aver/**`, banned stdlib families are archive, bytes, compress, container,
encoding, flag, io, math, os, rand, slices, strconv, strings, unicode, and uuid. `os/**` and
sim/nbio/default import os; only the latter imports os/signal. Invariant gets no exemption.

### Import Aliases

Import alias is permitted only when imported package's declared name collides with another import.
It holds no `default`, in any case: default directory declares its parent's name, so tier label
there fights package's own clause.

### Grouped Declarations

Each const, var, and type stands alone; parenthesized groups are banned. Simple reduction of the
language surface area. It also helps grepping for constants with the regex `^const foo =`

### Iota

A constant spells out its value; iota is banned. Make the value explicit instead of relying on
declaration order.

### Generics

Type and func declarations have no type parameters. Each declaration names one concrete program.

### Interface Declarations

A package declares no method-set interface. Method-free interfaces remain allowed.

### Variable Shadows

A name declared in an inner scope must not shadow one from an outer scope. The outer scope is the
whole package, sibling files included; an external test package is a separate name space.

### Mutable Globals

A global mutable var is permitted only via `regexp.MustCompile` or `errors.New`, as a
`var _ Interface = (*Type)(nil) assertion`, or as a single `Default` var in a package at or under
a directory listed in lint.json's `instrumentation_packages`.

### Discards

A bare `_ = value` discard is banned; a multi-value assignment may discard when at least one
variable is named, as in `_, err := foo()`.

### Init Functions

A package declares no func init; wire setup explicitly instead. Libraries should expose a
`func Init()` instead.

### Empty Bodies

A function body holds at least one statement; an empty body is banned.

### Methods

All methods banned except `Assertion_Builder` methods in `shared/sim/aver` and its `default` tier.
Make every other method free function with receiver as first parameter.

### Self Recursion

A function never calls itself by bare name, including inside a closure or a go statement — the call
graph must be acyclic regardless of stack. A method call does not count; a file in
opt_out_recursion_ban contributes nothing to its package's graph.

### Mutual Recursion

A function never reaches itself through a cycle of bare-name calls anywhere in its package,
closures and go statements included — the call graph is a DAG. A cycle across packages cannot
compile, so the graph stops at the package boundary.

### Compound Conditions

An if condition holds a single term; split && and || into nested ifs.

### Naked Returns

A return statement names its values; a bare return is banned.

### Fallthrough

A switch case does not fall through to the next.

### Bare Loops

A for loop carries a condition, range, or post clause; a bare for that loops forever is banned.

### Struct Tags

A struct tag uses only the stdlib keys json, xml, and asn1; any other key is banned.

### Blank Mutexes

A sync.Mutex or sync.RWMutex struct field is named so it can be locked; a blank-named one is banned.

### Unbounded Read

An unbounded read is banned: io.ReadAll, io.Copy, os.ReadFile, os.ReadDir, bufio.NewScanner, or
bufio.NewReader; read into a fixed buffer instead. Generated files are exempt.

### Unbounded Decode

An unbounded decode is banned: json, xml, or gob NewDecoder, or csv.NewReader; unmarshal a bounded
[]byte instead. Generated files are exempt.

### Unbounded Decompression

An unbounded decompression is banned: gzip, flate, zlib, bzip2, lzw, zip, or tar readers; cap the
output against a literal before reading. Generated files are exempt.

### Unbounded Allocation

An unbounded allocation is banned: bytes.NewBuffer or bytes.NewBufferString; use a fixed []byte
with explicit length tracking instead. Generated files are exempt.

### Unbounded Http

An unbounded net/http call is banned: http.Get, http.Post, http.ListenAndServe, or a Default
client, mux, or transport; set explicit timeouts instead. Generated files are exempt.

### Deprecated Ioutil

A deprecated ioutil call is banned: ioutil.ReadAll, ReadFile, WriteFile, TempFile, NopCloser, and
kin; use the os or io replacement instead. Generated files are exempt.

### Banned Words

A package, file, or declared identifier splits into words; no word, ignoring case, equals a word
listed in lint.json's `word_replacements` with an empty candidate list (util, utils, utility,
utilities, len, length). Use sites go unchecked, so the len and cap builtins stay legal.

# Source And Test Requirements

These forms are required in source and test files alike.

### Goimports

Imports are grouped and ordered as goimports writes them.

### Type Resolution

Every bare type in a package declaration or function signature resolves to a predeclared type or a
type declared by that package. Qualified types outside parsed scope remain unjudged.

### Default Package Name

A package in a directory named `default` declares the package clause of its parent directory,
shadowing the library it re-exports.

### Entry Point First

A file's main, Main, or TestMain is its first function declaration.

### Function Size

A function spans at most seventy lines.

### File Size

A source or test file spans at most 10000 lines.

### Main Package Size

A package main's source spans at most 200 lines, counted per build-tag group as the file count
is. Its test files carry no count.

### Named Returns

A function with results names every one of them.

### Array Capacity

An array type's capacity is a named constant, never a literal, as in [BUFFER_SIZE_MAX]byte. The
`[...]T{…}` form declares no capacity and is exempt.

### Keyed Struct Literals

A struct literal names each field, as in Coordinate{X: 0, Y: 1}. A type the run never parsed is
exempt, since nothing says whether it is a struct.

### Exported Struct Fields

An exported struct exposes no field of an unexported type, through any struct its own package
declares.

### Exported Types

A package-level type declaration or alias is exported. Function-local types and _test.go files
are exempt.

### Struct Field Public Identifier

Every struct field name begins with a capital letter; an unexported field is banned.

### Struct Field Type Encapsulation

A struct field's type must be exported if the struct is exported.

### Comment Opening

A comment opens with a space and a capital letter.

### Comment Ending

A comment ends in a period, colon, question mark, or exclamation mark.

### Package Documentation Comments

A package carries a doc comment on its clause: `// Package x ...` on at least one of its files.

### Exported Documentation Comments

Every exported package-level declaration carries a doc comment; a lone Go directive does not count.

### Struct Field Documentation Comments

Every struct field of an exported package-level struct carries a doc comment.

### Name Style

Exported identifiers use Ada_Case, unexported use snake_case, TestMain aside; every const uses
SCREAMING_SNAKE_CASE instead, whatever its scope or export status. A numeric capacity may occupy
one underscore-delimited Ada_Case segment, as in Enum_3_Int.

### Full Words

A declared name splits into words; each word, lowercased, is looked up in lint.json's required
`word_replacements` table (from the abbreviations-in-code project): a non-empty candidate list is an
abbreviation reported with its full-word expansion (id -> identifier), an empty list is a ban.

### Noun Names

A declared name splits into words; its last word, lowercased, must not end in ing unless it is in
the curated noun allowlist (string, ring, heading, encoding, …), so String and Encoding pass but
Parsing does not.

### Quantity Suffixes

Within a function, an AST pattern binds a required suffix to a declared name: len or cap to _count
or _size, a C-style induction to _index, a string Index call to _offset. The name's trailing word
must equal a required suffix.

### Arithmetic Suffixes

Operands of + and - share one quantity suffix, and the assigned result carries it too.

### Extremum Suffixes

A declared name's words include max or min only as the final word; a leading or interior max or min
is banned. Write line_max, not max_line.

### Test Documentation

Every Test function carries a doc comment, TestMain aside.

### Snap Literals

A snap.Init or snap.Edit first argument is a backticked raw string literal.

### File Count Source

A package holds exactly ceil(total_sloc/10000) source files.

### File Count Tests

Blackbox (foo_test) and whitebox (foo) test files each carry an independent total_sloc count,
from source files and from each other.

### File Count Specification Test

specification_test.go carries its own independent total_sloc count from other test files.

### File Count Build Tags

Files sharing a build-tag constraint form an independent group with its own total_sloc count.

# Deterministic

Every pure package is held, atop purity, to bans making it reproducible; a package
in lint.json's pure_but_indeterministic_packages opts out, and the bans bind _test.go too.

### Entry Format

A pure_but_indeterministic_packages entry is an exact-path glob naming a pure package
to opt out: `shared/io` releases that package, `shared/io/**` its subtree, `*` and
`**` spanning one path segment or many; a `!`-prefixed entry negates, always winning.

### Goroutines

A deterministic package starts no goroutine; the kernel decides goroutine
interleaving, the root of nondeterminism.

### Channels

A deterministic package uses no channel — no channel type, send, or receive —
since cross-goroutine handoff order is not the program's to decide.

### Select

A deterministic package uses no select; the runtime randomizes the choice among
the ready cases.

### Floats

A deterministic package uses no float32 or float64; IEEE-754 results diverge across
platforms through fused-multiply-add contraction and extended-precision intermediates.

### Banned Imports

A deterministic package imports none of time, context, sync, or sync/atomic; it
injects the clock and holds no concurrency to guard.

### Import Induction

A deterministic package's first-party imports are themselves deterministic, save
those listed in lint.json's `instrumentation_packages` — write-only side channels
the induction exempts.

### Impurity

Determinism is stricter than purity, so only pure packages are held; an impure
package (the main package, a default tier) is never deterministic and needs no
pure_but_indeterministic_packages entry to be excused.

### Coverage

A pure_but_indeterministic_packages entry that names a concrete path matching no pure
package is reported — a typo or stale path releasing nothing; a root-anchored
wildcard, naming no path, is exempt.

### Coverage Scope

A scoped run judges coverage only for entries within the scope it parsed; an entry
for a module outside the scope is left to the whole-workspace run, which alone
parses every module.

# Stdlib Time

Stdlib time may be imported only by the shared module's sim/time/default gateway;
every other shared-module package injects the Clock instead.

# Event Loop

Blocking and non-blocking IO runs on one event loop; pure code submits to it and never drives or
mints it.

### Driver

The Driver advances time and drives the loop. Only package main or a test can create it through
New_Virtual_Timeline or New_Operating_System_IO. Only those packages can name time.Driver.
Internal code takes time.Timeline, and the harness drives it.

### Gateway

Raw IO stdlib lives only in sim/nbio/default and sim/time/default. Instrumentation,
tests, generated files, and package main are exempt. os/default may import syscall alone. Route
other raw IO through shared/sim/nbio.

### Seed

A simulated backend takes only a seed. New_Sim has one integer seed parameter, and no Sim_* helper
lets a caller script outcomes. Thus, a run is a pure function of its seed.

# Configuration

lint.json's path lists share one glob matcher, so one rule keeps every entry unambiguous about
whether it names a directory or a file.

### Directory Slash

A wildcard-free entry that resolves to a directory ends in a slash and a file entry does not, so
the slash alone tells them apart. A wildcard entry is exempt, already expressing its shape.
