
# Printer

A Printer is the caller-owned state one file prints through. Print walks a parse tree and writes
the canonical form of the source it stands for into caller storage, thus printing allocates
nothing and a file of any admitted size costs one buffer.

# Layout

One declaration stands on one line and a block indents its statements by two spaces for each depth
it nests. A list, a chain, and an operation break where the author broke them, and a bracket left on
a line of its own keeps that line and takes the comma the line demands.

# Elision

A form that repeats what another already states writes it once: the type of an element of a run,
the length a slice closes at, a name a range binds to nothing, and the type of a variable its own
values name. A field list of no fields closes on the line it opened on.

# Names

An import writes the name the author bound only where dropping it would leave two packages of one
name, and each form that read that name names the package instead. A path closing at default or
internal names the element ahead of that one, and one file drops at most 64 such names.

# Spread

A signature the author broke, and a call whose parenthesis closes its line, each close on a line
of their own. A literal the author broke states one element to a line, a case head inside sixty
bytes reads on one line, and two declarations that each span lines stand an empty line apart.

# Bytes

No line of the form closes on a space or a tab, no two empty lines stand together, and every note
that follows a byte of its line stands behind a space. The form opens on a byte and closes on one
line feed.

# Width

A call, a literal, a signature, and a run of one operation each state one part to a line rather
than run past one hundred columns, where one character stands in one column and a tab a string
holds stands in eight. A line no break of the print shortens stands as it is, however wide it reads.

# Order

A function that states the invariants of a type the file declares stands under that type, with the
notes above it, and one file pairs at most 1,024 of them. Every other declaration, and every such
function past that count, stands where the author wrote it.

# Entry

A file that declares main or Main states that function ahead of every other function it writes,
with the notes above it. A file that declares none, and one that states it there already, each
stand as the author wrote them.

# Returns

A return that names no value writes the names its signature bound, up to 16 of them, thus a reader
sees what a function hands back at the line it hands it back on. A signature that binds no name,
spells one as the blank, or binds more, states none to write and the return writes the word alone.

# Spacing

A sign that binds loosely stands between spaces and one that binds tightly stands against its
values, thus one*2 + 3 reads as it binds. A form inside another closes its signs up, and a unary
sign, a comma, a call, an index, and a selector each stand against what they follow.

# Alignment

A run of fields aligns the types on the widest name and the tags on the widest type, a run of keyed
elements the values on the widest key, and a run of noted lines the notes on the widest line, up to
65536 columns. An empty line, a comment, and a part that runs over lines each close a run.

# Literals

A number states its base prefix and its exponent mark in lower case, and an imaginary whole number
states no leading zero, and no string states a carriage return. The digits themselves stand as the
author wrote them, because the case of a digit carries meaning in no base.

# Trivia

A comment and a run of empty lines print where the author wrote them. A comment left on the line a
form closes keeps that line and one given a line of its own keeps that line, a run of empty lines
prints as one empty line, and no comment states a carriage return.

# Crowded Lines

A line holds up to 16 comments that stand at its end. A line that carries more is refused rather
than printed without the comments the storage could not hold.

# Lines

A body closes up against its brackets, dropping the empty lines against them. An empty line inside
the head of a declaration, inside an expression, ahead of a brace, behind the sign of an
assignment, or ahead of a simple error check states nothing, and no two stand together.

# Refusals

A print that fills the storage the caller supplied stops and reports it. A tree the parser refused
prints what it holds, because a caller reads more from a partial print than from nothing, and a
tree standing for anything but a file writes nothing at all.

# Bounds

One print writes at most 2,097,152 bytes, walks a tree at most 4,096 nodes deep, opens a line at
most 128 depths deep, and stands an expression at most 64 forms deep, refusing a tree past any of
them. It costs time in the count of the nodes one tree holds and never in the square of it.

# Allocation

Print performs zero heap allocation. The caller owns the tree, the source, and the storage the
print writes into.
