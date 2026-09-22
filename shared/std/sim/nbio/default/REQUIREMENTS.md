
# Storage

Host storage opens, writes, synchronizes, reads, and closes files through caller-owned
completions. Positioned operations preserve the file cursor. Pipe reads wait for data and
observe end of file after the writer closes.

# Network

Host sockets connect, accept, send, and receive through the event loop. Expiring a receive
preserves caller ownership of the socket. Sending from a connect callback is supported.

# Timeline

Timeouts fire after their deadlines. Retired completions release their callbacks.
Rearming an already armed completion and driving the loop from a callback are rejected.

# Events

Each event trigger retires one listener, which may rearm the same completion. Triggering
from another thread remains safe while the loop rearms that completion.

# Processes

Spawn reports exit status and captured output. Deadlines terminate the process group and
deliver one terminal callback. Output beyond caller storage reports an error while
retaining the prefix that fits. A timed-out child is still reaped.

# Allocations

Construction and storage and network operations allocate no heap memory after caller
storage is prepared. Spawn and captured spawn allocate no more than their underlying
syscall baseline.
