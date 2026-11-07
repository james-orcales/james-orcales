
# Parse

Parse scans a source into the token run and runs the grammar program over it into the node arena.
Its Parse_Result names the root, the run extents, the arena, the error log, and the earliest
refusal. The caller owns every store and the package holds none, thus a parse allocates nothing.

# Nodes

A node carries its kind, the token that names it, its parent, and its child chain. Slot zero is
no node, thus zero names an absent child and a walk needs no separate empty test.

# Token Run

Parse_State_Token_Run borrows the exact token prefix the parse its Parse_Result names wrote. It
contains at least the end token and at most 131,072 tokens. Caller owns the Parse_State and the
borrowed run allocates nothing.

# Parsed Cursors

A parse writes the end token and the absent slot whatever it reads, thus the Parse_Result counts
one token and one slot at least.

# Package Clause

A file opens with the package keyword and one identifier.

# Imports

An import declaration carries one path literal and an optional name. The name is a node of its
own, thus a caller tells the name an import binds from the path it binds it to. A parenthesized
group states one import for each path it holds.

# Constants And Variables

A const or var declaration carries one or more names, an optional type, and an optional value. A
constant name is a node of its own, because it answers to a case rule a variable name does not. A
parenthesized group states one declaration for each name it holds.

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
range, switch, type switch, select, go, defer, break, continue, goto, and label. A fallthrough
parses and then fails, thus the caller reads the rule it broke.

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

Every form of valid Go parses. A form the linter refuses parses whole, logs its code below at the
token that broke the rule, and the parse reads on, thus one parse names every refusal a source
states. Only a source Go refuses too, or one that overruns a bound, stops the parse.

### Interface Methods

An interface holding a method logs FAILURE_INTERFACE_METHOD at the method name, in a declaration,
a parameter type, or a field type. The method parses as a field holding its signature as a
function type. A method declaration parses clean, because the linter bans only the interface.

### Wide Identifiers

A name holding a byte at or above 128 logs FAILURE_WIDE_IDENTIFIER. The scanner reads a wide
letter or digit into the name it stands in, as Go does, thus the tree holds the name whole. A
comment and a string literal still carry any byte.

### Iota

The iota identifier logs FAILURE_IOTA wherever it stands, because it ties a constant value
to the order its declaration happens to sit in.

### Refused Signs

The conditional and, the conditional or, the less-or-equal, and the greater-or-equal each log
FAILURE_REFUSED_SIGN and bind at Go's own level, thus the tree reads as Go reads it. Nested ifs
state a joined condition, and a strict comparison states an order sign with equality.

### Dot Imports

An import bound to the period logs FAILURE_DOT_IMPORT, and the period is its name node, because it spills a whole package
into the file and leaves no way to read where a name came from.

### Blank Imports

An import bound to the blank name logs FAILURE_BLANK_IMPORT, because it takes a package
for its effects alone and no call states that it ran.

### Constant Case

A constant name that is no uppercase word, or run of uppercase words joined by single
underscores, logs FAILURE_CONSTANT_CASE. The rule binds a constant at any scope, and a
variable name answers to none of it.

### Bare Loops

A for that no clause constrains logs FAILURE_BARE_LOOP: a bare for, a three-clause for
with every clause empty, and a for whose only condition is the true literal. A range says the
same thing in a form of its own and reads as the assertion it is.

### Unnamed Results

A signature result that carries no name logs FAILURE_UNNAMED_RESULT. A result names itself
and its type, thus a result standing as a type alone names nothing.

### Type Aliases

A type declaration bound by the assignment sign logs FAILURE_TYPE_ALIAS at that sign. An alias gives one
type a second name, and a reader who meets the second name learns nothing about the first.

### Closure Bodies

A function literal whose body holds anything but calls logs FAILURE_CLOSURE_BODY, one time for
each statement that breaks the rule. Only the last may return one call, and a grouped call is a
call. Logic a closure hides stays out of reach of a test, and a named function lays it open.

### Missing Package Clause

A file that opens with anything but the package keyword fails with FAILURE_PACKAGE_CLAUSE.

### Token Count

A source spending more tokens than one run holds fails with FAILURE_TOKEN_COUNT.

### Node Count

A source spending more nodes than the arena holds fails with FAILURE_NODE_COUNT.

### Nesting Depth

A source nesting deeper than the parent stack holds fails with FAILURE_NESTING_DEPTH, and so does
a run of groupings deeper than the grammar program may call itself.

### Fallthrough

A case that runs on into the case behind it logs FAILURE_FALLTHROUGH. A case closes where
its statements close, thus a reader of one case reads the whole of what it runs.

### Broken Syntax

A token that fits no rule fails with FAILURE_SYNTAX. This code names no dropped Go form, thus a
source it refuses is a source Go refuses too.

# Failure Reports

How a caller reads a refusal back.

### Codes

The Failure a Parse_Result carries reads FAILURE_NONE after a clean parse and the code of the
refusal at the earliest token otherwise. Each code is distinct, thus a caller switches on it rather than on a
message.

### Messages

Failure_Message reads one code as an imperative sentence that says what to do about it. Each
sentence is a compile-time constant, thus naming a failure allocates nothing.

### Position

The Failure_Token a Parse_Result carries names the run position of the token the earliest
refusal stands at, and it stays inside the run the scan wrote, thus reading it back through Token_At is always
safe.

### Error Log

The caller supplies a log of 256 entries. Each refusal lands in the log in parse order, Error_Count
bounds what it wrote, and past 256 the log keeps the first 256. A stop joins the log behind every
refusal met before it, and a parse empties the log a parse before it wrote.

### Tree

A source a rule refuses returns the whole tree. A stopped source returns the tree built so far,
because a linter reads more from a partial tree than from nothing. A stopped token carries an
error node, and a source that overran the token count returns no tree at all.

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

A parse holds at most 131,072 tokens and 131,064 nodes. One Parse_State descriptor spans 152 bytes.
Caller supplies exact token and node stores, three stacks of 256 indexes, and a 256-refusal log.
An empty line count rides in a padded byte token, thus file spacing costs a parse nothing.

# Tail

A parse empties every token and every slot past what it wrote, thus a wider parse before it leaves
nothing behind, and a read past the tree fails rather than reading a stale node.

# Allocation

Parse and every accessor perform zero heap allocation. Caller supplies Parse_State plus every
backing store; parse writes only inside them.
