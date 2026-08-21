
# Diagnostics

Each diagnostic names one file and stands in one of two tiers.

### Tier One

Tier one diagnostic is auto-fixable. It carries the fix: the whole file as it should stand.

### Tier Two

Tier two diagnostic is not auto-fixable. It carries no fix.

# Command

### Usage

`shared/cli` parses `harness <directory> [-json] [-fix]`; `<directory>` is one Go package. A
command line it refuses writes `harness: <error>` and exits 1. `-h` writes help and exits 0.

### Recursive

Command checks each Go file under `<directory>`, at any depth, and writes one line per diagnostic:
path, then message.

### Json

With `-json`, each diagnostic is one line of `shared/encoding/flatjson`: `path`, `line`, `column`,
`tier`, `rule`, `message`. A diagnostic without a position has line and column 0.

### Fix

With `-fix`, the fix of each tier one diagnostic replaces its file, and the diagnostic is neither
written nor counted. A tier two diagnostic stays. A failed open, write, or close fails step `fix`.

### Requirements

`harness/pkg/requirements` checks each package's contract after the Go files. Each finding is
tier two with rule `requirements`, at `REQUIREMENTS.md` or `requirements_test.go` of its package.

### Status

A run with no diagnostic exits 0. A run with any diagnostic, or any failed step, exits 1. Failed
step writes `harness: failed to <step>: <path>`.

### Passed

A run with no diagnostic writes `✓ all checks passed`, then `elapsed_sec=<s.mmm>` from the clock.
With `-json` it writes `{"passed":true,"elapsed_sec":"<s.mmm>"}`, a string since flatjson holds no
float.

# Format

### Clean

Go file that stands in the canonical form of `shared/go/format` yields no diagnostic.

### Unformatted

Go file that does not stand in the canonical form yields tier one diagnostic `not formatted`. Fix
is the canonical form.

### Refused

Go file the parser refuses yields tier two diagnostic, since a refused tree has no fix. Its line
is `<path>:<line>:<column>: <message>`: where the refused token stands, then the parser's message.

### Oversized

Go file past `token.SOURCE_SIZE_MAXIMUM` yields tier two diagnostic `exceeds source size`.

# Allocation

Package main allocates `Memory` once. After that a run allocates zero heap bytes.
