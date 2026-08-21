
# Compile

Compile validates one bounded pattern and emits one immutable NFA into caller workspace.

# Wildcards

`*` consumes any run outside separators, `**` consumes any run, and `?` consumes one decoded
character outside separators.

# Separators

Caller separators bound `*` and `?` without changing literal or `**` matching.

# Classes

Character classes contain decoded members and inclusive ranges. Leading `!` negates membership.

# Alternatives

Brace groups select one branch. Explicit compiler and matcher stacks bound nesting without
recursion.

# Escape

Backslash quotes one metacharacter. Quote_Meta_Into stages complete escaped output in caller
destination.

# Bounds

Pattern, text, separators, syntax nodes, NFA instructions, active states, and nesting have
formula-derived limits. Hostile input returns scalar status before unsafe access.

# Allocation

Compile, Match, and Quote_Meta_Into allocate zero heap bytes. Caller owns compile and match
workspaces.
