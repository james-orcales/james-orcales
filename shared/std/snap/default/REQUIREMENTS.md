
# Host

The default adapter reads and writes files through the operating-system gateway using caller
storage. Host errors are returned to the caller. Its caller lookup reports the calling source
file and line, and its output writer uses standard error.

# Snapshots

Init and Edit bind snapshots to Default with their call-site location and expected text.
Only Edit enables source rewriting. The forwarding functions accept the core library's types
and preserve snapshot comparison behavior.
