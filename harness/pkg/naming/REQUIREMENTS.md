
# Naming

Check states the first name one parsed Go file breaks, as lint states it: the byte it stands at and
the message. The caller owns the tree, the source, the word table, and every store.

### Name Style

Check splits a name into words as lint does: at an underscore, at an uppercase letter past a
lowercase letter or a digit, and before one past an uppercase run. A rename keeps the case: Ada_Case
when the name opens uppercase, an uppercase word whole, else snake_case.

### Full Words

A name lint walks, the package name, or the file stem cut of `.go` then `_test`, holding a word the
table replaces, states `Rename <name> -> <rename>.`, or `[<rename>, ...]` for several. A function,
its receiver, parameters, and results, a top-level type and its struct, const, var, and := declare.

### Noun Names

A walked name whose last word ends in `ing` states `The name "<word>" is a present participle.
Rename it to a noun form.`, unless the word is a noun lint lists, such as `string` or `heading`.

### Quantity Suffixes

In a function, `:=` of `len` or `cap` ends in count or size, of a stdlib Index in offset, of a Len
or Cap method in size; a counted loop steps an index; `make` sizes a count, or a size of bytes.
Else it states `The name <name> has the role <roles>. Rename <name> -> <rename>.`

### Arithmetic Suffixes

A `+` or `-` of two names ending in index, count, offset, or size that lint's table admits none for
states `The arithmetic _<left> <sign> _<right> is not coherent. Use operands with compatible
suffixes.` at its left operand.

### Arithmetic Results

A name `:=` takes from an admitted sum, ending off its result, states `The arithmetic <name> =
_<left> <sign> _<right> gives the wrong suffix. End the name with _<result>.`

### Extremum Suffixes

A walked name holding `max` or `min` as a word before its last states `Rename <name> -> <rename>.`,
the word moved to the end.

### Banned Words

A name of Full Words holding a word the table replaces with nothing states `The identifier "<name>"
contains the banned substring "<word>". Write a different name.` A used name stands, such as `len`.

### Import Aliases

An import alias holding `default` in any case states `The import alias "<alias>" holds "default".
Name the alias for the package. A default directory declares the parent as that package.`

### Unnecessary Aliases

Any other alias states `Package import alias "<alias>" is unnecessary. Remove alias because no other
imported package is named "<package>".`, unless another import names it: its last path segment, or
the segment before a last `default`.

### Shadows

A name a function body declares by `:=`, var, const, or an if, for, or range init, that a scope
around it declares, states `The variable "<name>" shadows a variable in an outer scope. Write a
different name.`

### Scopes

Scopes nest as lint's do: the file's top-level names, a function's parameters, results, and body,
then each block, if, for, and range. A switch, a select, and a closure open none Check reads.

### Records

Record lists each name Shadows judged, then each function but a method, type, and constant the file
declares at the top level, for a caller judging names against the sibling files of a package.

### Calls

Record then lists each function and method, each followed by the bare names its body calls,
closures and go statements included, for a caller judging the call graph of a package.

### Impurity

Record then lists each import, states the first selector of a clock, stdout, or network API lint
lists and the first of one lint lists as reaching one, with their imports, and counts func Main.

### Order

Check states the first finding in source order: the file stem at byte zero, then each import and
name. At one name, a table word comes first, then quantity, arithmetic, participle, and extremum.
A shadow comes past every other finding.

### Bounds

A message past the store is cut at it, and each byte of a table replacement or an import path below
a space stands as `?`, since `harness.json` and the file are untrusted.
