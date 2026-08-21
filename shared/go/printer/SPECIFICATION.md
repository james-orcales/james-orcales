
# Printer

A Printer is the caller-owned state one file prints through. Print walks a parse tree and writes
the canonical form of the source it stands for into caller storage, thus printing allocates
nothing and a file of any admitted size costs one buffer.

# Layout

One declaration stands on one line and a block indents its statements by one tab for each depth it
nests. A list, a chain, and an operation break where the author broke them, and a bracket left on
a line of its own keeps that line and takes the comma the line demands.

# Elision

A form that repeats what another already states writes it once: the type of an element of a run,
the length a slice closes at, a name a range binds to nothing, and the type of a variable its own
values name. A field list of no fields closes on the line it opened on.

# Spread

A signature the author broke, and a call whose parenthesis closes its line, each close on a line
of their own. A literal the author broke states one element to a line, a case head inside sixty
bytes reads on one line, and two declarations that each span lines stand an empty line apart.

# Width

No line the form states runs past one hundred columns, where one character stands in one column
and a tab stands in eight. A call, a literal, a signature, and a run of one operation each state
one part to a line rather than run past it; a line holding one token that wide stands as it is.

# Order

A function that states the invariants of a type the file declares stands under that type, and the
notes above it travel with it. A function that answers for a type of another file names no type to
stand under, thus it stands where the author wrote it, as does every other declaration.

# Returns

A return that names no value writes the names its signature bound, thus a reader sees what a
function hands back at the line it hands it back on. A signature that binds no name, or spells one
as the blank, states no name to write, thus the return there stands as the author wrote it.

# Spacing

A sign that binds loosely stands between spaces and one that binds tightly stands against its
values, thus one*2 + 3 reads as it binds. A form inside another closes its signs up, and a unary
sign, a comma, a call, an index, and a selector each stand against what they follow.

# Alignment

A run of struct fields aligns the types on the widest name and the tags on the widest type, a run
of keyed elements aligns the values on the widest key, and a run of noted lines aligns the notes
on the widest line. An empty line, a comment, and a part that runs over lines close a run.

# Literals

A number states its base prefix and its exponent mark in lower case, and an imaginary whole number
states no leading zero. The digits themselves stand as the author wrote them, because the case of a
digit carries meaning in no base.

# Trivia

A comment and a run of empty lines print where the author wrote them. A comment left on the line a
form closes keeps that line and one given a line of its own keeps that line, and a run of empty
lines prints as one empty line.

# Lines

A body closes up against its brackets: a function body, a body of one statement or one case, a
literal, and a run of fields each drop the empty lines against them. A line behind the sign of an
assignment, and one ahead of a simple error check, each state nothing at all.

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
