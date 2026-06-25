
# Sim

The simulated IO backend schedules every operation at now plus a seed-drawn latency and
fires it when the clock reaches its Ready_At. New_Sim(seed) builds it and never hands it
out, drawing every outcome from that seed, so a run reproduces and nothing is scriptable.

### Timeout

A timeout fires exactly when the virtual clock reaches its deadline, off the same
Ready_At queue the IO completions use, so every wait rides one timeline.

### Read

A read on an opened file returns its stored bytes from the offset after the modeled latency,
so a mirror reads back what an earlier write stored; a read on any other descriptor reports
the buffer length.

### Listen

Listen returns a fresh synthetic descriptor synchronously; the simulated backend
binds nothing, so the call never fails.

### Accept

An accept completes after the modeled latency and yields a new descriptor distinct
from its listener, modeling one inbound connection.

### Connect

A connect completes after the modeled latency and yields a fresh connected
descriptor, with no real handshake.

### Receive

A receive completes after the modeled latency and reports the buffer length, with
no real syscall and no waiting.

### Send

A send completes after the modeled latency and reports the buffer length, with no
real syscall and no waiting.

### Close

A close completes after the modeled latency and reports no error.

### Run Until

Run_Until drives the loop until its predicate reports true, delivering completions each
step, so a straight-line caller can wait for its own operation inline.

### Cancel

Cancelling an in-flight operation still fires its callback exactly once, with the
Cancelled error rather than a result, so every submission resolves and nothing leaks.

### Reuse

Submitting a completion that is still in flight panics: one Completion backs at most one
operation at a time, so reusing it before its callback fires fails loudly rather than
corrupting the queue.

### Open

Open returns a descriptor for an existing file synchronously; an absent path or a directory
errors instead. A later Read of the descriptor delivers the file's bytes on the loop.

### Create

Create returns a fresh writable descriptor synchronously, distinct from every other; a
later Write persists bytes to it on the loop.

### Peer Address

Peer_Address reports a connected descriptor's remote address synchronously — a
getpeername has no completion; an unknown descriptor yields the empty address.

### Accept Secure

A secure accept completes after the drawn latency and yields a new descriptor, modeling
one inbound TLS connection; the simulator has no TLS, so it drives the plaintext socket.

### Connect Secure

A secure connect completes after the drawn latency and yields a fresh connected
descriptor; the simulator models it like Connect, since it has no TLS.

### Connect Insecure

An insecure connect behaves identically to a secure one in the simulator, which has no
TLS: it yields a fresh connected descriptor after the drawn latency.

### Watch Signal

A watched signal arrives at a seed-drawn grain and fires its callback exactly once with
that signal — the operating-system event modeled as a seed outcome, not scripted.

### Compute

Offloaded work runs and its completion fires on a later drain, on the loop's own
timeline; the simulator runs it inline so the result stays reproducible.

### Spawn

A spawn completes after the drawn latency with a seed-drawn exit code and no captured
output — the seed decides success or failure, since scripted output is disallowed. A live
Stdout or Stderr sink instead streams that output on the real backend, uncaptured.

### Read Directory

Read_Directory lists a directory's immediate children synchronously, sorted by name so the
run reproduces, each entry naming a child and whether it is itself a directory.

### Status

Status reports synchronously whether a path exists and, if so, whether it is a directory; an
absent path is not-exists with a nil error, so a caller branches on the status, not an error.

### Make Directory

Make_Directory creates a path and any missing parents synchronously against the tree; an
existing directory converges, so a repeated mkdir is not an error.
