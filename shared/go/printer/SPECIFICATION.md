
# Printer

A Printer is the caller-owned state one file prints through. Print walks a parse tree and writes
the canonical form of the source it stands for into caller storage, thus printing allocates
nothing and a file of any admitted size costs one buffer.

# Layout

One declaration stands on one line and a block indents its statements by one tab for each depth it
nests. A list, a chain, and an operation break where the author broke them, and a bracket left on
a line of its own keeps that line and takes the comma the line demands.

# Spacing

A sign that binds loosely stands between spaces and one that binds tightly stands against its
values, thus one*2 + 3 reads as it binds. A form inside another closes its signs up, and a unary
sign, a comma, a call, an index, and a selector each stand against what they follow.

# Alignment

A run of struct fields aligns the types on the widest name and the tags on the widest type, a run
of keyed elements aligns the values on the widest key, and a run of noted lines aligns the notes
on the widest line. An empty line, a comment, and a part that runs over lines close a run.

# Trivia

A comment and a run of empty lines print where the author wrote them. A comment left on the line a
form closes keeps that line and one given a line of its own keeps that line, and a run of empty
lines prints as one empty line.

# Refusals

A print that fills the storage the caller supplied stops and reports it. A tree the parser refused
prints what it holds, because a caller reads more from a partial print than from nothing, and a
tree standing for anything but a file writes nothing at all.

# Bounds

One print writes at most 2,097,152 bytes, walks a tree as deep as the 131,064 nodes one parse
holds, and aligns at most 1,024 fields in one run.

# Allocation

Print performs zero heap allocation. The caller owns the tree, the source, and the storage the
print writes into.
