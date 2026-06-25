
# Coverage

Which packages must carry a SPECIFICATION.md, and which are exempt.

### Presence

A pure in-scope package whose SPECIFICATION.md is absent is flagged. The caller
decides presence by the exact-cased name and passes a nil spec when it is gone.

### Exemptions

A module-less, vendored, example, or impure package is never required to carry
the file; its absence draws no diagnostic.

# Format

The structural rules the markdown of a present SPECIFICATION.md must obey.

### Preamble

No content precedes the first heading; a leading blank line is allowed.

### Leaf

A # with no ### child is a leaf; a # with ### children is a branch whose leaves
are those children. Only leaves require a test.

### Heading Blank Lines

Every heading is preceded and followed by a blank line.

### Heading Level

Only # and ### headings are used; any other level is flagged.

### Heading Characters

A heading's words are made of letters and digits only, so its Ada_Case test
name is a legal identifier.

### Heading Uniqueness

No two # share a name; ### names are unique within their parent #.

### Section Not Empty

Every leaf section carries at least one body line; a branch intro may be empty.

### Section Size

A section holds at most three body lines, excluding the blank lines fencing its
heading.

### Section Contiguity

A section's body lines are contiguous; no blank line falls between them.

# Tests

How specification_test.go must mirror the spec's leaf headings.

### Presence

A package whose specification_test.go is absent is flagged; the caller passes a
nil test AST when the file is gone.

### Per Heading

Each leaf heading has a Test_<Heading> function, its Ada_Case name derived from
the heading words.

### Name Normalization

A leaf test not named for its heading's Ada_Case form is flagged at its slot.

### Order

The leaf tests are the file's leading declarations, in leaf order; a var, const,
type, or misordered test before them shifts the match and is flagged.
