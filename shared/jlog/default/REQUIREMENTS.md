
# Default Floor Is Info

New_Default_Logger builds with an Info floor, so trace and debug noise stays out of the
stderr stream unless a caller raises verbosity explicitly.

# Default Writes Stderr Through Storage

Default submits each line to the root's descriptor at its current position with
nbio.Storage_Write on the injected Storage; the emit returns before the loop retires it, and
the retirement frees the line.

# Default Writes To A Pipe

A line reaches a pipe descriptor intact through the real operating-system IO, the shape stderr
takes when the process is piped.
