# shell_sdk

A walled-garden structured-data CLI: one binary, symlinked to bare verb names,
that passes typed values between pipeline siblings as a compact binary format and
renders a table when it reaches a terminal.

## The walled garden

Inside a pipeline, verbs speak a private binary wire format — no reparsing, no
lossy round-trips between stages. A value only becomes text when you ask for it:

- piped to another shell_sdk verb → the binary wire format;
- at a terminal → an aligned table;
- with `-json` → JSON; with `-table` → a table, even when piped;
- `to json` / `to csv` → the value leaves the garden as text.

```
from csv < people.csv | filter age gt 30 | sort-by age | to json
```

`from`/`load` enter the garden, the middle verbs stay in it, `to` leaves it.

## Install

The binary dispatches on the name it is invoked as (argv[0]), busybox-style, and run by
its own name it takes the verb from the first token. The `install` verb fans it out into
a verb-named symlink per verb:

```
shell_sdk install ~/.local/bin       # add that directory to your PATH
```

Every verb is now its own command: `filter`, `sort-by`, `to`, and so on.

## Verbs

| verb | arguments | does |
|---|---|---|
| `load` | `<file>` | read a file into the garden (`.csv`/`.json` by extension, else sniffed) |
| `from` | `<format>` | parse stdin as `json` or `csv` |
| `to` | `<format>` | serialize to `json` or `csv`, leaving the garden |
| `get` | `<path>` | navigate a dotted cell path |
| `pick` | `<field>…` | keep only the named fields |
| `reject` | `<field>…` | drop the named fields |
| `relabel` | `<old> <new>` | rename a field across records |
| `filter` | `<field> <operator> <value>` | keep rows where the cell compares |
| `sort-by` | `<field>` | order a list of records by a field (stable) |
| `group-by` | `<field>` | bucket records by a field value, first-seen order |
| `distinct` | | drop duplicate elements |
| `reverse` | | reverse a list |
| `first` | `-count=<n>` | the first element, or the first n |
| `final` | `-count=<n>` | the last element, or the last n |
| `columns` | | the union of record keys |
| `count` | | the number of items or fields |
| `wrap` | `<name>` | wrap the value in a single-field record |
| `flatten` | | flatten one level of nested lists |

Filter operators are words, not symbols, so they never collide with the shell:
`eq ne lt le gt ge`. Verb names avoid system binaries — `distinct` (not `uniq`),
`final` (not `last`), `relabel` (not `rename`).

Every verb takes `-help` for its own usage; run the binary by its own name for the
full catalog.

## Numbers

Numbers are stored as their exact source text and compared as rationals, so the
tool is deterministic and `3.14` round-trips unchanged — no float64 anywhere.

## Examples

```
# average-free rollup: how many people per department, largest first
from csv < people.csv | group-by dept | to json

# a projection out to CSV
load people.json | pick name age | to csv

# the two oldest, as JSON
from csv < people.csv | sort-by age | final -count=2 -json
```

## Exit codes

- `0` success
- `2` usage error (unknown verb or flag, missing argument) — prints help
- `1` runtime failure (unreadable file, parse error, wrong shape, over-limit input)

## Implementation

`main.go` is the only impure file: it binds stdin, the filesystem, and the
terminal, then hands them to the pure `internal` library. Command-line parsing is
the shared `cli` package in multicall mode. See `internal/SPECIFICATION.md`.
