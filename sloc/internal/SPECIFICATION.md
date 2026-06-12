
# Classify

Classify_File partitions every physical line into code, comment, or blank, counting
each line once so the three sum to the line count. A line bearing code is code even
with a trailing comment; a comment-only line is a comment; a whitespace line is blank.

### Comments

A line comment runs to end of line, a trailing comment leaves the line code, and a
documentation comment is an ordinary comment.

### Strings

A comment token inside a string is code, not a comment, and a backslash escapes the
closing quote.

### Block Comments

Go block comments do not nest, so the first close ends them; Rust block comments nest,
so an inner open must be matched before the comment ends. They may span lines.

### Raw Strings

A raw string spans lines verbatim with its comment tokens inert: backtick strings and
template literals, Python triple quotes, and Rust hash-counted raw strings. A docstring
is a string, so its lines are code.

### Heredocs

A heredoc body is code, with its comment tokens inert, until the terminator line: the
opener is `<<` then an optional `-` or `~` and a quoted or uppercase word (Shell, Ruby,
Perl), so the `a << b` shift operator opens nothing.

### Characters and Lifetimes

A character or rune literal is code, and a Rust lifetime tick opens nothing, so it
never swallows the rest of the line.

### Newlines

Lines split on newline: a trailing newline adds no phantom line, a final unterminated
line still counts, and the empty file is zero lines.

# Languages

The seeded languages span the C, hash-comment, and markup families (Go, Rust, C, C++,
Java, Python, Ruby, Lua, Zig, Odin, HTML, and more), each a configuration of comment
tokens and string delimiters for the one generic scanner.

### Detection

A file's language is resolved from its extension, or for an extensionless file like a
Makefile from its name; an unrecognized file resolves to nothing.

# Count

Count walks a tree, classifies every recognized file in parallel, and returns one
File_Count per file in the lexical order the walk visited them.

### Extensions

Only files whose extension is recognized are counted; the rest are skipped silently.

### Hidden

A dot-prefixed file or directory is skipped by default and counted only with
Include_Hidden; a hidden directory is pruned, not descended.

### Exclusion

The injected predicate prunes an ignored directory and skips an ignored file; a nil
predicate ignores nothing.

### Binary

A file whose first chunk holds a NUL byte is treated as binary and skipped rather than
counted as garbage lines.

### Tests

A file is marked a test by a test directory (test, tests, spec, __tests__) in its path,
or by its language's filename convention (Go's _test., a .spec. infix, a test_ prefix).

# Render

Render writes an aligned table that groups languages by category with a grand Total,
stable for a given report regardless of counting order.

### Table

Languages are grouped under category labels (Systems, Managed, Dynamically Typed, Shell,
Markup & Data, Build & Config, Query, Hardware), with a grand Total and a %Code column
giving each source or test sub-row its share of its own group's code, blank elsewhere.

### Files

With Show_Files each file is listed indented under its language, with its own counts
and no per-file file count.

### Thousands

A count of a thousand or more is grouped with commas.

### Tests

A language with test files splits into indented source and test sub-rows, and the grand
Total splits the same way; a language without tests shows only its single row.

### JSON

Render_Json emits the report as JSON instead of a table: a name-sorted languages array,
each with its category and source and test counts, and a total.

# Limitations

Rare constructs are misclassified without breaking the line sum: lowercase heredoc
delimiters and a second heredoc on one line (Shell, Perl, Ruby), regexes, deferred
raw-string and second block-comment forms (C++, D, Pascal), and non-ASCII lines.
