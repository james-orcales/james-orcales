
# Module

A Module is caller-owned state one module of Go source folds into. Caller supplies every arena and
cursor store, then feeds every file to Declare, every file to Resolve, and every body to Check.
One parse tree at a time therefore serves a module of any size.

# Files

Add_File binds one file to the package path it belongs to and to the source the caller owns. The
caller keeps those bytes alive, because every name a symbol wears is a view of them, and it feeds
the files one build states, because this module reads no build tag.

# Universe

The universe holds the predeclared types and functions. A name that no file declares resolves
there, thus int, string, and len mean what Go says without a file that states them.

# Declarations

Declare records one package-level symbol for each constant, variable, type, and function a file
states, and one import binding for each import. It reads no type, because a name a later file
declares is a name this pass cannot know.

# Type Expressions

Resolve folds the type expression of each symbol into the type arena: pointer, array, slice, map,
channel, function, struct, constraint, generic instance, and type parameter. Two equal types may
be two entries, thus a caller compares kinds and parts and never indexes.

# Named Types

A type declaration builds a named type that carries its symbol and its underlying type. A cycle
through a name that no pointer, slice, or map breaks is a refusal, because such a type has no size.

# Bodies

Check walks the bodies of one file and records the type of every expression it folds. The caller
owns one Body, and an expression this pass cannot fold records the invalid type rather than a
guess.

# Statements

A short declaration, a declaration statement, a range clause, and a type switch each bind names,
and a block scope holds them until it ends. A name a block binds stands ahead of the package name
it shadows.

# Expressions

An identifier, a literal, a unary or binary operation, a call, a conversion, an index, a slice, a
selector, an assertion, a composite literal, a function literal, and a grouping each fold to the
type Go states for them.

# Answers

One reader answers for each part a symbol, a type, and a member hold, and Lookup names one package
member. Underlying_Of walks a named type down to the type its declaration states, and Type_At
reads the type one tree slot folded to.

# Refusals

A name that resolves nowhere, a name one package states twice, an import no file declares, and a
count past a bound each fail with a code of their own. Failure_Message reads one code as an
imperative sentence.

# Bounds

A module holds at most 1,024 files, 256 packages, 65,536 symbols, 65,536 types, and 32,768
members. A name spans at most 512 bytes, a type expression nests at most 64 deep, and one body
binds at most 8,192 names.

# Allocation

Every call performs zero heap allocation. Caller supplies Module, Body, Parse_State, and every
backing store; every pass writes only inside them.
