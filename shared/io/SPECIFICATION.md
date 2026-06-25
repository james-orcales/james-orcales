
# Sim

The simulated IO backend schedules every operation at now plus a seed-drawn latency and
fires it when the clock reaches its Ready_At. New_Sim(seed) builds it and never hands it
out, drawing every outcome from that seed, so a run reproduces and nothing is scriptable.

### Timeout

A timeout fires exactly when the virtual clock reaches its deadline, off the same
Ready_At queue the IO completions use, so every wait rides one timeline.

### Read

A read completes after the modeled latency and reports the buffer length, with no
real syscall and no waiting.

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
