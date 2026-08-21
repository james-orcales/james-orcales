
# Coverage

Which packages owe a REQUIREMENTS.md.

### Presence

A package holding no REQUIREMENTS.md is flagged: `The package has no REQUIREMENTS.md.`

### Exemptions

A package holding a `package main` file, or under a `third_party` or `examples` segment, owes none.

# Format

The markdown structure a present REQUIREMENTS.md obeys.

### Preamble

No content precedes the first heading; a leading blank line is allowed.

### Heading Level

Only `#` and `###` headings are used; any other heading is flagged.

### Heading Blank Lines

Every heading is preceded and followed by a blank line.

### Heading Characters

A heading's words hold letters and digits only, so its Ada_Case test name is an identifier.

### Heading Uniqueness

No two `#` share a name, and `###` names are unique within their `#`. A `###` needs a `#` above.

### Section Not Empty

A leaf section carries a body line. A `#` with no `###` child is a leaf, and so is each `###`; a
`#` with children may be empty.

### Section Size

A section holds at most three body lines.

### Section Contiguity

A section's body lines are contiguous; no blank line falls between them.

# Tests

How requirements_test.go mirrors the leaf headings.

### Presence

A package with REQUIREMENTS.md and no requirements_test.go is flagged.

### Order

The leading declarations are `Test_<Leaf>` for each leaf in leaf order, a `###` leaf being
`<Parent>_<Child>` in Ada_Case. A slot holding anything else is flagged at its line.

# Resume

Next stops at each finding and at the request for requirements_test.go. The caller reports or
loads, then calls Next again, until the check is done.

# Bounds

A REQUIREMENTS.md past its storage, more headings than the heading table holds, or a heading past
128 bytes is flagged, and the check stops there.

# Allocation

A check performs zero heap allocation. The caller owns every table and storage.
