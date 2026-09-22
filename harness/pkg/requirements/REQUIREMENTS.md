
# Coverage

Which packages owe a REQUIREMENTS.md.

### Presence

A package is a directory holding at least one Go file. One holding no REQUIREMENTS.md is flagged:
`The package has no REQUIREMENTS.md.`

### Exemptions

A package under a `third_party` or `examples` segment owes none, and so does one holding a
`package main` or `package main_test` file the run read whole and the parser took; a file it
refused proves no clause. One holding a REQUIREMENTS.md is checked against it all the same.

# Format

The markdown structure a present REQUIREMENTS.md obeys. A line ends at a newline, and a carriage
return before it belongs to that ending rather than to the line. One the file ends on belongs to
the line it stands in, since no newline follows it.

### Preamble

No content precedes the first heading; the blank line that heading carries before it is no
content of its own. A file that states neither a heading nor content states no section at all,
and is flagged.

### Heading Level

Only `#` and `###` headings are used; any other heading is flagged.

### Heading Blank Lines

Every heading is preceded and followed by a blank line. The first heading carries that blank line
before it too, thus a contract opens with one.

### Heading Characters

A heading holds at least one word; its words hold ASCII letters and digits only and carry no
capital after a lower case letter, so its Ada_Case test name is one this toolchain spells and
keeps. A heading that builds a name no test file could carry is flagged.

### Heading Uniqueness

No two `#` share a name, and `###` names are unique within their `#`. A `###` needs a `#` above.
Two leaf sections whose names fold to one test name are flagged too, since one test file states
that name once.

### Section Not Empty

A leaf section carries a body line. A `#` with no `###` child is a leaf, and so is each `###`; a
`#` with children may be empty.

### Section Size

A section holds at most three body lines.

### Section Contiguity

A section's body lines are contiguous; no blank line falls between them.

### Conflict Markers

No line opens on a merge conflict marker: seven repeats of <, >, %, +, or backslash, the
sentinels a VCS writes into a file it cannot merge.

# Tests

How requirements_test.go mirrors the leaf headings.

### Presence

A package with REQUIREMENTS.md and no requirements_test.go is flagged.

### Order

The leading declarations are `Test_<Leaf>` for each leaf in leaf order, a `###` leaf being
`<Parent>_<Child>` in Ada_Case. A slot holding anything else is flagged at its line, and a
method carries its receiver before the name the slot states.

### Refused

A requirements_test.go the parser refuses holds no declaration the check can read, thus the order
check does not run for it and the check states nothing of its slots.

# Resume

Next stops at each finding and at the request for requirements_test.go. The caller reports or
loads, then calls Next again, until the check is done.

# Rewrite

Rewrite puts a blank line on each side of every heading and takes the blank lines out from between
two body lines, the three findings a contract states with one answer. It leaves alone a contract
whose check stops at a bound, and one whose rewrite would outgrow its storage.

# Bounds

A REQUIREMENTS.md past its storage, more headings than the heading table holds, or a heading past
128 bytes is flagged, and the check stops there. Declarations past the declaration table go
unread, since the leaves they could answer stand past the leaf table too.

# Allocation

A check performs zero heap allocation. The caller owns every table and storage.
