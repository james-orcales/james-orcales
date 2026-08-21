
# Parse

Parse scans a source into the token run, then descends it into the node arena. The caller owns
one Parse_State and the package holds no storage of its own, thus a parse allocates nothing.

# Nodes

A node carries its kind, the token that names it, its parent, and its child chain. Slot zero is
no node, thus zero names an absent child and a walk needs no separate empty test.

# Package Clause

A file opens with the package keyword and one identifier.

# Imports

An import declaration carries one path literal and an optional name. The name is a node of its
own, thus a caller tells the name an import binds from the path it binds it to.

# Constants And Variables

A const or var declaration carries one or more names, an optional type, and an optional
value. A constant name is a node of its own, because it answers to a case rule a variable name
does not.

# Types

A type declaration carries one name and one type. Every Go type form parses: name, pointer,
array, slice, map, channel, function, struct, constraint interface, and generic instance.

# Functions

A function declaration carries an optional receiver, one name, its signature, and its body. A
result is a node of its own, thus a caller reads whether a function sends anything back. A method
declaration parses, because the linter bans the interface and never the method.

# Type Parameters

A function or a type declaration may carry a type parameter list. A constraint is a type, a
union of terms, or an approximation term under the tilde.

# Statements

Every Go statement form parses: block, declaration, assignment, increment, send, return, if, for,
range, switch, type switch, select, go, defer, break, continue, goto, fallthrough, and label.

# Expressions

Every Go expression form parses with Go's own precedence: identifier, literal, unary, binary,
call, index, slice, selector, type assertion, composite literal, function literal, and grouping.

# Trivia

A comment and a run of empty lines are nodes of the tree at the place they stand, not lists
beside it. A caller reads a doc comment by walking to the sibling before a declaration and needs
no separate map to guess which comment belongs to what.

### Fidelity

Every comment the scanner reads appears exactly once in the tree, and so does every run of empty
lines. Nothing the file holds is dropped on the way in, thus a caller can write the source back
out from the tree alone.

### Order

A pre-order walk meets the trivia in source order, thus a caller that prints the tree prints each
comment and each empty line where the author wrote it.

### Attachment

A comment stands under the node whose span holds it and before whatever follows it. A doc comment
is therefore the sibling ahead of its declaration and a body comment is a child of its block.

### Blank Lines

A run of empty lines is one NODE_BLANK whose token is the token behind the run, and that token
counts the lines. A run is taken one time however often a step asks for it.

# Refused Syntax

Valid Go this grammar drops must fail, and must fail with the code named below. A caller
therefore never reads a dropped form as a bare syntax error. A form the linter merely rejects is
not dropped: it parses, so the linter can judge the tree and name the rule itself.

### Declaration Groups

A parenthesized const, var, type, or import group fails with FAILURE_DECLARATION_GROUP. One
declaration stands for each name, which is what a reader greps for.

### Interface Methods

An interface holding a method fails with FAILURE_INTERFACE_METHOD, whether it stands as a
declaration, a parameter type, or a field type. Only a constraint element parses, and a method
declaration parses because the linter bans the interface and never the method.

### Wide Identifiers

A name holding a byte at or above 128 fails with FAILURE_WIDE_IDENTIFIER. A comment and a string
literal still carry any byte.

### Iota

The iota identifier fails with FAILURE_IOTA wherever it stands, because it ties a constant value
to the order its declaration happens to sit in.

### Private Fields

A struct field name that opens with a lowercase letter fails with FAILURE_PRIVATE_FIELD, and an
embedded type names the field it embeds. A field name is a node of its own, thus a caller tells
it from the type behind it.

### Compound Predicates

An if condition joined by a conditional and or a conditional or fails with
FAILURE_COMPOUND_PREDICATE, and a grouping around it changes nothing. A for, a switch, and an
assignment each read a joined term as they always did.

### Dot Imports

An import that binds no name fails with FAILURE_DOT_IMPORT, because it spills a whole package
into the file and leaves no way to read where a name came from.

### Blank Imports

An import bound to the blank name fails with FAILURE_BLANK_IMPORT, because it takes a package
for its effects alone and no call states that it ran.

### Constant Case

A constant name that is no uppercase word, or run of uppercase words joined by single
underscores, fails with FAILURE_CONSTANT_CASE. The rule binds a constant at any scope, and a
variable name answers to none of it.

### Naked Returns

A return that names no value fails with FAILURE_NAKED_RETURN when its function declares a result.
A function that declares none owes nothing, thus a bare return there parses, and a function
literal answers for its own signature and never for the one around it.

### Bare Loops

A for that no clause constrains fails with FAILURE_BARE_LOOP: a bare for, a three-clause for
with every clause empty, and a for whose only condition is the true literal. A range says the
same thing in a form of its own and reads as the assertion it is.

### Unnamed Results

A signature result that carries no name fails with FAILURE_UNNAMED_RESULT. A result names itself
and its type, thus a result standing as a type alone names nothing.

### Type Aliases

A type declaration bound by the assignment sign fails with FAILURE_TYPE_ALIAS. An alias gives one
type a second name, and a reader who meets the second name learns nothing about the first.

### Missing Package Clause

A file that opens with anything but the package keyword fails with FAILURE_PACKAGE_CLAUSE.

### Token Count

A source spending more tokens than one run holds fails with FAILURE_TOKEN_COUNT.

### Node Count

A source spending more nodes than the arena holds fails with FAILURE_NODE_COUNT.

### Nesting Depth

A source nesting deeper than the parent stack holds fails with FAILURE_NESTING_DEPTH.

### Broken Syntax

A token that fits no rule fails with FAILURE_SYNTAX. This code names no dropped Go form, thus a
source it refuses is a source Go refuses too.

# Failure Reports

How a caller reads a refusal back.

### Codes

Failure reads FAILURE_NONE after a clean parse and the code of the refused form otherwise. Each
code is distinct, thus a caller switches on it rather than on a message.

### Messages

Failure_Message reads one code as an imperative sentence that says what to do about it. Each
sentence is a compile-time constant, thus naming a failure allocates nothing.

### Position

Failure_Token names the run position of the token the parse stopped at, and it stays inside the
run the scan wrote, thus reading it back through Token_At is always safe.

### Tree

A refused source still returns the tree that was built, because a linter reads more from a
partial tree than from nothing. A refused token carries an error node, and a source that overran
the token count returns no tree at all.

# Approximate Forms

A form that no parser settles without a type checker still parses, and its tree states the shape
the grammar could see rather than the one Go resolves.

### Bare Parameters

A run of names with no type behind it stays a run of identifiers under one parameter node,
because only a type checker separates two parameter types from two parameter names.

### Bracket Suffixes

A bracket suffix holding one expression is an index node whether it reads an element or supplies
one type argument. A comma inside makes it generic and a colon makes it a slice.

### Clause Literals

A composite literal in a control clause parses only when its type is no bare name. A bare name
there reads as a value and its braces as a block, where Go refuses the source outright.

### Open Comments

A general comment that no delimiter closes runs to the end of the source and stays a comment,
where Go reports it unterminated.

# Bounds

A parse holds at most 131,072 tokens and 131,064 nodes, and one Parse_State spans 4,197,248
bytes: twenty for each node and twelve for each token. The empty line count rides in a byte the
token already padded out, thus the spacing of a file costs a parse nothing.

# Allocation

Parse and every accessor perform zero heap allocation. The caller makes one Parse_State and the
parse writes only inside it.
