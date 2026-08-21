
# Format

Format writes the canonical form of one parsed tree into caller storage and reports whether the
whole form fit. The caller owns the tree, the printer, and the storage, thus one format allocates
nothing and states no policy the caller can turn off.

# Clean

Clean reports whether one source already stands in the canonical form, which is the question a
linter asks of a file. It writes the form into the storage to answer, thus a caller that wants the
form as well reads the storage it handed over.

# Imports

The imports of one file stand in three runs parted by one empty line: the standard library, then
this module, then every other one. Each run stands in the order of the paths it names, the note
above an import travels with it, and an empty line between two imports states nothing.

# Names

A declared name takes the case the linter asks: SCREAMING_SNAKE_CASE for a constant, Ada_Case for
a struct field, else Ada_Case or snake_case by its first letter. Each identifier that spells it
takes the new spelling too, the clause that names the package aside.

# Declarations

Format renames by the table alone. Declare takes the names of one file into it, thus a caller that
declares each file of a program renames a name in every file that spells it. Forget empties the
table, and a name past the storage of the table reports that the table is full.

# Refusals

A form that fills the storage the caller supplied stops and reports it, and a tree the parser
refused formats what it holds. A source that stands past the storage is no clean source, because
the form it would take was never written.

# Bounds

One format writes at most 2,097,152 bytes, which is the widest form one print writes.

# Allocation

Format and Clean perform zero heap allocation. The caller owns the tree, the source, and the
storage the form stands in.
