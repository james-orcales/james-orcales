
# Edit

An edit is a contiguous run of runes tagged Retain, Delete, or Insert; together
the runs form a diff script.

### Stringer

String renders the script as kind-prefixed double-quoted runs: a space for
retain, plus for insert, minus for delete, with inner quotes backslash-escaped.

# Differ

A Differ carries the two texts under comparison as both runes and strings, and
the edit script the diff functions build for them.

### Construction

New copies the Old and New strings into the Differ as both rune slices and
strings, leaving the edit script empty.

### Reset

Reset empties the edit script and clears both texts while retaining the backing
array capacity of the rune slices.

# Diff

Diff returns the character-level script: the runs that retain, delete, and
insert runes to turn Old into New.

### Basics

Single-rune and empty inputs produce the minimal script: identity retains,
pure insertion, pure deletion, and one-rune substitutions.

### Examples

Real sentences and adversarial mixed edits, including invalid UTF-8, diff to a
compact cleaned-up script of merged and boundary-shifted runs.

# Line Diff

Line Diff renders a diff at line granularity: each output line is an original
line prefixed by a space, a plus, or a minus.

### Basics

Identity, insertion, deletion, and substitution of single lines render with the
correct space, plus, and minus prefixes.

### Blocks

Inserting, deleting, or replacing a line inside a block leaves the surrounding
lines retained and marks only the changed line.

### Code

Edits to source lines render line-for-line, each changed line shown as a paired
deletion and insertion.

### Records

JSON and YAML records diff line-by-line, retaining unchanged keys and blank
separators while marking only the altered lines.

### Markup

Nested HTML edits retain the unchanged tags and mark only the lines whose text
or attributes changed.

### Document

A multi-record YAML document with reorderings retains the stable keys and pairs
each changed line with its replacement.

### Source

Multi-line function bodies and try blocks diff line-for-line, retaining the
structural lines and pairing each changed statement.

### Query

A multi-clause SQL statement diffs line-by-line, retaining the stable clauses
and pairing each changed clause with its replacement.

### License

A whole-license rewrite reduces to the same block diff a line-based tool would
produce, retaining only the blank separators.

# Algorithm Diff

Algorithm Diff is Myers' O(ND) core: a forward furthest-reaching trace and a
backtrack that emit a minimal rune-level script.

### Basics

Empty and single-rune inputs short-circuit to identity, pure insert, pure
delete, or a one-rune substitution.

### Blog

The blog-post sentences diff to a minimal script of single-rune deletes and
inserts around the retained runs.

### Custom

Adversarial mixed edits diff to a minimal single-rune script, the raw form the
cleanup pass later merges and shifts.

# Find Common Prefix

Find Common Prefix returns the longest run of runes that begins both inputs, or
nil when they share no leading rune.

### Cases

The result is argument-order independent and is a genuine prefix of both inputs
across ASCII, accented, CJK, and emoji runes.

# Find Common Suffix

Find Common Suffix returns the longest run of runes that ends both inputs, or
nil when they share no trailing rune.

### Cases

The result is argument-order independent and is a genuine suffix of both inputs
across ASCII, accented, CJK, and emoji runes.

# Find Common Run

Find Common Run returns the longest contiguous shared run, but only when it
spans at least half the longer input; otherwise nil.

### Cases

Centered, prefix, and suffix overlaps are found across odd and even lengths,
and a too-short overlap yields nil.

# Runes Have Prefix

Runes Have Prefix reports whether a non-empty expected run begins the input.

### Predicate

An empty input, an empty expected run, or an expected run longer than the input
all report false.

# Runes Have Suffix

Runes Have Suffix reports whether a non-empty expected run ends the input.

### Predicate

An empty input, an empty expected run, or an expected run longer than the input
all report false.
