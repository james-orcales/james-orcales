
# Specification

These rules govern this SPECIFICATION.md file.

### File Name

The specification lives in a file named exactly `SPECIFICATION.md`.

### Coverage

Only a pure package carries this file; a scopeless run requires it of every pure package.
Impure packages — `package main` and `default` — are exempt, as are vendored and `example`
packages.

### Preamble

The specification opens directly with a heading; no content precedes the first `#`.

### Leaf

A `#` is a leaf when it has no `###` children, and a branch when it does — in which case its `###`
children are the leaves. Only leaves require a test, mirroring Go's namespacing depth.

### Heading Blank Lines

Every heading is preceded and followed by a blank line.

### Heading Level

The specification uses heading levels `#` and `###`, and no others. A purely subjective judgement
call on the "readable" size difference of the two heading styles.

### Heading Characters

A heading's words are made of letters and digits only.

### Heading Uniqueness

No two `#` share a name; `###` are unique within their parent `#`. Otherwise, one test could satisfy
multiple sections.

### Section Not Empty

Every leaf is followed by at least one body line; a branch's intro body is optional.

### Section Size

A section holds at most three lines, excluding the blank lines around its heading.

### Section Contiguity

A section's body lines are contiguous; no blank line falls between them.

### Test File Name

Tests live in a file named exactly `specification_test.go`.

### Test Per Heading

Each leaf has a test named from its headings: a top-level `#` leaf is Test_<Heading>, a `###` leaf
is Test_<Parent>_<Heading>.

### Test Name Normalization

The heading is normalized to Ada_Case to form the test name.

### Test Order

The leaf tests are the file's first declarations, in leaf order, none preceding.

# Diagnostics

A per-file diagnostic is tier one or tier two. Tier one always prints; any tier-one anywhere in
scope suppresses tier two, which prints only when no tier-one fired and so may rely on tier-one
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

A symlink resolves to an existing target; a dangling symlink is banned.

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

# Commits

These rules govern the commit history on a branch.

### Subject Size

A commit subject runs to at most 100 characters.

### Conventional Subjects

A commit subject is a lowercase type, an optional (scope), an optional ! breaking marker, then a
colon, a space, and a non-empty description — the conventional-commits form.

### Fixup Commits

A subject is a fixup when it starts with fixup! or squash!, pairs review with address or apply and
with comment, feedback, or nit, or reads cr comment, code review comment, review fix, or review nit;
autosquash such commits into their target.

### Merge Commits

A branch other than `main/master` carries no merge commit; a git-subtree merge is the sole
exception; required when vendoring another repo's history.

# Module Layout

The following layout centralizes cross-module imports and splits pure and impure implementations.
Shared module:  `shared/foo (pure) -> shared/foo/default (impure)`
Binary modules: `main (impure) -> internal (pure)`

### Module Definition

A module is a directory containing `go.mod` and is registered in `go.work`.

### Module Location

All modules are located at the repo root.

### Shared Module

The shared module, `shared`, has no `internal` directory at any depth and declares no
`package main`. It is fully importable, never executed. Its workspace-root-relative directory
is lint.json's `shared_module`.

### Binary Module

In a binary module, every non-main package lives under a top-level internal directory; code at the
module root or any non-internal directory is banned.

### Main Package

A binary module holds exactly one main package, and it sits at the module root; a `cmd` directory is
banned.

### Internal Entry Point

A binary module declares exactly one free func Main in its top-level internal directory; it is
the entry point its thin package main calls, and zero or several Main functions are banned. A shared
module may also expose a func Main, representing an embeddable entry point.

### Tier Depth

A non-main package has at most one non-main package above it, not counting `v{N}` dirs or, in a
binary, the top-level internal dir. So the deepest a package nests is `shared/foo/[v{N}]/default` or
`binary/internal/foo/[v{N}]/default`: a pure library package, then its impure default tier.

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

### Grouped Declarations

Each const, var, and type stands alone; parenthesized groups are banned. Simple reduction of the
language surface area. It also helps grepping for constants with the regex `^const foo =`

### Iota

A constant spells out its value; iota is banned. Make the value explicit instead of relying on
declaration order.

### Interface Declarations

A package declares no method-set interface; a type-constraint interface for generics is allowed.

### Variable Shadows

A name declared in an inner scope must not shadow one from an outer scope.

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

A method is banned unless its exact signature satisfies a stdlib interface — error, fmt.Stringer,
io.Reader/Writer/Closer/Seeker, sort.Interface, json/text/binary marshalers, fs.FS and kin;
otherwise make it a free function taking the receiver as its first parameter.

### Self Recursion

A function never calls itself by bare name within its own file; method and package-qualified calls
do not count.

### Mutual Recursion

A function never reaches itself through a cycle of bare-name same-file calls; method and
package-qualified calls do not count.

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

### Banned Function Words

A function name's words include no helper, ignoring case; other identifiers may. The narrow ban
keeps helper from hiding what the function does.

### Whitebox Tests

A test file declares an external test package as `foo_test`; whitebox test packages are banned.

# Source And Test Requirements

These forms are required in source and test files alike.

### Goimports

Imports are grouped and ordered as goimports writes them.

### Default Package Name

A package in a directory named `default` declares the package clause of its parent directory,
shadowing the library it re-exports.

### Entry Point First

A file's main, Main, or TestMain is its first function declaration.

### Function Size

A function spans at most seventy lines.

### Input Structs

A function whose parameters or whose returns repeat a type takes a single input struct pointer or
returns one output struct, named for the function and declared just above it; a variadic may remain
a separate parameter.

### Named Returns

A function with results names every one of them.

### Keyed Struct Literals

A struct literal names each field, as in Coordinate{X: 0, Y: 1}.

### Exported Struct Fields

An exported struct exposes no field of an unexported type.

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

Exported identifiers use Ada_Case, unexported use snake_case, TestMain aside.

### Method Prefix

A free function over a locally (same package) declared type carries that type as its name prefix, as
`type_verb/Type_Verb`; its own input struct is the exception.

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

Test files carry an independent total_sloc count from source files.

### File Count Specification Test

specification_test.go carries its own independent total_sloc count from other test files.

### File Count Build Tags

Files sharing a build-tag constraint form an independent group with its own total_sloc count.

# Deterministic

A package covered by lint.json's deterministic_packages is held, atop purity, to
bans making it reproducible; opt-in, and the bans bind its _test.go files too.

### Entry Format

A deterministic_packages entry names a module's top-level directory and the tier
auto-applies to the pure packages at or under it; the shared module's libraries
are named `shared/*` for every library or `shared/<lib>` for one.

### Goroutines

A deterministic package starts no goroutine; the kernel decides goroutine
interleaving, the root of nondeterminism.

### Channels

A deterministic package uses no channel — no channel type, send, or receive —
since cross-goroutine handoff order is not the program's to decide.

### Select

A deterministic package uses no select; the runtime randomizes the choice among
the ready cases.

### Banned Imports

A deterministic package imports none of time, context, sync, or sync/atomic; it
injects the clock and holds no concurrency to guard.

### Import Induction

A deterministic package's first-party imports are themselves deterministic, save
those listed in lint.json's `instrumentation_packages` — write-only side channels
the induction exempts.

### Impurity

Determinism is stricter than purity, so only pure packages join the tier; an
impure package in a covered subtree (the main package, a default tier) is dropped
from the expansion, not held to the bans.

### Coverage

A deterministic_packages entry that covers no pure package is reported; a typo, a
stale path, or a directory holding nothing pure must not silently check nothing.

# Stdlib Time

Stdlib time may be imported only by the shared module's time/default gateway;
every other shared-module package injects the Clock instead.
