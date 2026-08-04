
# Sim

The simulated IO backend schedules every operation at now plus a seed-drawn latency and
fires it when the clock reaches its Ready_At. New_Sim(seed) builds it and never hands it
out, drawing every outcome from that seed, so a run reproduces and nothing is scriptable.

### Timeout

A timeout fires exactly when the virtual clock reaches its deadline, off the same
Ready_At queue the IO completions use, so every wait rides one timeline. The duration
must be positive; a caller that needs a deferred callback uses Next_Tick.

### Next Tick

Next_Tick appends a deferred callback to the completed queue without submitting kernel IO.
Reset_Next_Tick removes every queued next-tick completion for the selected source and returns
those completions to idle without invoking their callbacks.

### Read

A read on an opened file returns its stored bytes from the offset after the modeled latency,
so a mirror reads back what an earlier write stored; a read on any other descriptor reports
the buffer length.

### Fsync

Fsync completes after the simulator has accepted every prior write to the file. It does not
create, close, or transfer descriptor ownership.

### Open At

Open_At asynchronously opens or creates a file relative to DIRECTORY_CURRENT, forces
close-on-exec ownership, and returns the new caller-owned descriptor through its callback.
OPEN_AT_NO_FOLLOW rejects a symbolic link in the final path part. An unknown flag panics.

### Event

Open_Event creates the TigerBeetle cross-thread event primitive. Event_Listen arms one
completion, Event_Trigger makes that completion ready, and Close_Event releases the event only
after its listener has drained.

### Listen

Listen consumes a caller-owned TCP socket and returns the resolved address synchronously.
When the requested port is zero, the simulator returns a non-zero synthetic port.

### Accept

Accept requires a positive finite deadline and yields a descriptor distinct from its listener when
inbound latency wins. The deadline wins ties, retires once with Deadline_Exceeded, and yields no
descriptor. This finite lifetime deliberately diverges from TigerBeetle's unbounded accept.

### Open Socket

Open_Socket_TCP and Open_Socket_UDP return fresh caller-owned descriptors synchronously and
record them in Raw_Open. Only an explicit caller Close or Close_Socket releases them.

### Connect

A connect requires a positive finite deadline and borrows the caller's socket. Latency reports
success or Connection_Refused; the deadline wins ties with Deadline_Exceeded. Every outcome leaves
the descriptor in Raw_Open for caller teardown, extending TigerBeetle's unbounded Connect.

### Receive

A receive completes after the modeled latency and reports the buffer length, with
no real syscall and no waiting.

### Send

A send completes after the modeled latency and reports the buffer length, with no
real syscall and no waiting.

### Send Now

Send_Now attempts a synchronous datagram send. The simulator reports the whole buffer sent for
an open socket and reports not-sent for a closed or send-shutdown socket.

### Shutdown

Shutdown changes the selected receive/send halves synchronously without taking ownership of the
socket. Armed and later receives on a receive-shutdown socket resolve with EOF; armed and later
sends on a send-shutdown socket resolve with Broken_Pipe. The descriptor remains open.

### Close

A close is an asynchronous operation and removes the descriptor from Raw_Open when its completion
retires. It neither cancels nor purges other operations: the descriptor owner must join every
submitted operation before submitting Close.

### Close Socket

Close_Socket synchronously releases a caller-owned socket during setup failure or final deinit.
It is distinct from asynchronous Close and does not deliver a completion.

### Run Until

Run_Until drives the loop until its predicate reports true, delivering completions each
step, so a straight-line caller can wait for its own operation inline.

### Reuse

Submitting a completion that is still armed panics as an illegal lifecycle transition:
only an idle completion may be armed. Delivery returns it to idle before the callback
runs, so reuse after — or from within — the callback is legal.

### Copy

Submitting a by-value copy of a completion panics: the loop tracks a completion by its own
address, so a copy carries the original's identity and fails loudly rather than splitting
the loop's view from the caller's.

### Reentrancy

Driving the loop from within a completion callback panics: Run, Run_For, and Run_Until are
top-level only, so a re-entrant drive fails loudly rather than corrupting the queue
mid-drain.

### Open

Open returns a descriptor for an existing file synchronously; an absent path or a directory
errors instead. A later Read of the descriptor delivers the file's bytes on the loop.

### Create

Create returns a fresh writable descriptor synchronously, distinct from every other; a
later Write persists bytes to it on the loop.

### Peer Address

Peer_Address reports a connected descriptor's remote address synchronously — a
getpeername has no completion; an unknown descriptor yields the empty address.

### Watch Signal

Watch_Signal requires a positive finite deadline. A signal arriving first fires once; the deadline
wins ties and retires once with Deadline_Exceeded and no signal. This finite lifetime deliberately
diverges from TigerBeetle, which has no signal-watch operation.

### Compute

Offloaded work runs and its completion fires on a later drain, on the loop's own
timeline; the simulator runs it inline so the result stays reproducible.

### Spawn

A spawn requires a positive finite deadline; natural completion returns the seed-drawn exit code,
while the deadline wins ties with Deadline_Exceeded. The real backend kills its subprocess group,
bounds pipe cleanup to one second, and returns partial output. TigerBeetle has no Spawn counterpart.

### Self Exec

Self_Exec replaces the image and returns only on failure; close-on-exec sockets make it rebind.
A nil environment preserves ambient values; a non-nil slice is the complete replacement
environment, so an empty slice inherits nothing. The simulator cannot replace its test process.

### Read Directory

Read_Directory lists a directory's immediate children synchronously, sorted by name so the
run reproduces, each entry naming a child and whether it is itself a directory.

### Status

Status reports synchronously whether a path exists and, if so, whether it is a directory and
its size in bytes. It also reports if the path is a regular file. An absent path is not-exists
with a nil error, so a caller branches on the status, not an error.

### Make Directory

Make_Directory creates a path and any missing parents synchronously against the tree; an
existing directory converges, so a repeated mkdir is not an error.

### Introspect

Introspect reports every queued simulator operation by class, the synthetic poll, wake, and
compute lifecycle flags derived from those queues, and every descriptor tracked in Raw_Open.
