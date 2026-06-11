
# Spawn Raw Plan

Spawn Raw Plan is the pure kernel that partitions arguments into a Command —
environment, executable, and its arguments — without touching the OS.

### Valid

Leading "KEY=VALUE" arguments are environment; the first non-assignment is the
executable and the rest its arguments. A value may contain '=' or be empty, and
an '=' after the executable is an ordinary argument.

### Invalid

A plan with no executable — an empty argument list, or one that is all
environment — is invalid.

# Shell Run Command

Shell Run Command resolves a Shell's defaults into a Command, then hands the
fully specified Command to the injected Run.

### Environment

The base environment comes first and the per-command entries last, so a
duplicate key resolves to the per-command override.

### Streams

A Command with a nil Stdout or Stderr inherits the Shell's default streams.

# Shell Spawn

Shell Spawn runs the command described by its arguments with the Shell's default
streams.

### Exit

A zero exit status reports success; any other status reports failure.

# Shell Pipe

Shell Pipe runs the command described by its arguments and captures its stdout.

### Capture

The captured stdout is returned whitespace-trimmed.

# Shell Working Directory

A Shell's working directory is the field it runs commands in, changed by
deriving a new Shell rather than by mutating the process.

### Derivation

Deriving a sub-shell sets the new directory on the copy and leaves the original
Shell untouched.
