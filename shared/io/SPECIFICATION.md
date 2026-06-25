
# Sim

The simulated IO backend schedules every operation to complete at now plus a modeled
latency on its injected clock, then fires each one when the clock reaches its
Ready_At — TigerBeetle's in-memory Storage and PacketSimulator, fully reproducible.

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
