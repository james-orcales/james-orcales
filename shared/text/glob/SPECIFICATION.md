
# Compile Builds A Matcher

Compile turns a pattern into a Pattern whose Match reports whether a string
satisfies the pattern; a plain pattern matches only itself.

# Wildcards Match Runs

With no separators `*`, `**`, and `?` all range over any characters — `*` and
`**` over any run, `?` over exactly one.

# Separators Bound Wildcards

Given a separator rune, `*` and `?` refuse to cross it, while `**` still spans it,
so path-like patterns segment on the separator.

# Character Classes Match Sets

A `[...]` class matches one member or one character in a `lo-hi` range, and a
leading `!` negates the class.

# Alternatives Match Any Branch

A `{a,b,c}` group matches when the surrounding pattern matches with any one branch
substituted in.

# Escaping Matches Literally

A backslash before a metacharacter matches that character literally, so `\*`
matches a single asterisk.

# Must Compile Panics On Bad Pattern

Must_Compile returns the Pattern for a valid pattern and panics for one Compile
would reject.

# Quote Meta Escapes Metacharacters

Quote_Meta backslash-escapes every metacharacter, so compiling its result matches
the original text literally.

# Deep Nesting Is Rejected

A pattern whose `{...}` nesting exceeds PATTERN_DEPTH_MAX is rejected with an
error rather than overflowing the stack.
