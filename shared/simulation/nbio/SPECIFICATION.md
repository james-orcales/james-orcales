
# Sim

Simulated IO backend schedule each operation at now plus seed-drawn latency. New_Simulated_IO
initializes caller-owned bounded state. Independent streams draw ordinary outcomes and timeout
order from one seed. Thus, runs reproduce and nothing is scriptable.

### Read

Storage.Read requires a positive timeout. Linux and the simulator enforce it. A timeout reports
zero with Deadline_Exceeded and leaves the buffer unchanged. Darwin ignores the timeout because
its file path has no kernel timeout. A zero-length read uses the first grain without a draw.

### Write

Storage.Write requires a positive timeout. Linux and the simulator enforce it. A timeout reports
zero with Deadline_Exceeded and leaves the file unchanged. Darwin ignores the timeout because
its file path has no kernel timeout. A zero-length write uses the first grain without a draw.

### Fsync

Fsync requires a positive timeout. Linux and the simulator enforce it. A timeout reports zero and
Deadline_Exceeded after it retires the operation. Darwin accepts the timeout but does not apply
it because its current file path has no kernel timeout. Fsync does not transfer file ownership.

### Open At

Open_At asynchronously open or make file relative to DIRECTORY_CURRENT, force close-on-exec
ownership, and return new caller-owned descriptor through its callback. OPEN_AT_NO_FOLLOW reject
symbolic link in final path part. Unknown flag panic.

### Socket

Socket_TCP and Socket_UDP make non-blocking close-on-exec descriptors. Each limit is positive.
Socket creation rejects a disabled limit. Every TCP socket enables keepalive and finite linger.
The pure TCP profile enables TCP_NODELAY. A backend can reduce a limit for its platform.

### Bind

Bind enable SO_REUSEADDR before it gives the caller-owned socket its local address. The caller
cannot omit address reuse or apply it after bind.

### Accept

Accept require positive finite timeout. It yields a descriptor distinct from its listener when
inbound completion wins, or Deadline_Exceeded and no descriptor when timeout wins. A separate
seed stream selects equal-time order because supported kernels can report either terminal event.

### Connect

Connect require positive finite timeout and borrow caller socket. Latency reports success or
Connection_Refused. Timeout reports Deadline_Exceeded only when it retires the pending operation.
Every outcome leaves descriptor in Raw_Open for caller teardown.

### Receive

Receive require positive finite timeout. Modeled latency reports buffer length when the operation
wins. Timeout reports zero and Deadline_Exceeded. Both outcomes leave descriptor caller-owned.

### Send

Send require positive finite timeout. Modeled latency reports buffer length when the operation
wins. Timeout reports zero and Deadline_Exceeded. Both outcomes leave descriptor caller-owned.

### Shutdown

Shutdown change selected receive/send halves synchronously, without take of socket ownership.
Armed and later receive on receive-shutdown socket resolve with EOF. Armed and later send on
send-shutdown socket resolve with Broken_Pipe. Descriptor stay open.

### Close

Close is asynchronous operation, and remove descriptor from Raw_Open when its completion retire.
It neither cancel nor purge other operations: descriptor owner must join every submitted
operation before submit of Close. It is one release for both file and socket.

### Open

Open return descriptor of existing file, synchronously. Absent path, or directory, error instead.
Later Read of descriptor deliver file bytes on loop.

### Create

Create return fresh writable descriptor, synchronously, distinct from every other. Later Write
persist bytes to it on loop.

### Peer Address

Peer_Address report remote address of connected descriptor, synchronously — getpeername has no
completion. Unknown descriptor yield empty address.

### Status

Status reports path kind and size synchronously without following a final symbolic link.
Absent path reports not-exists with nil error. Read_Link writes target into caller storage.
Read_Link rejects other kinds with Not_Symbolic_Link.

### Watch Signal

Watch_Signal requires positive finite deadline; signal arriving first fires once, and deadline wins
ties with Deadline_Exceeded and SIGNAL_EXPIRED. Caller retains armed signal because expired value
identifies no watch. Simulator draws arrival grain from its own seed stream.

### Spawn

A spawn requires a positive finite deadline; natural completion returns the seed-drawn exit code,
while the deadline wins ties with Deadline_Exceeded. Real backend kills its subprocess group, bounds
pipe cleanup to one second, and returns partial output; simulator captures none.

### Deinit

Deinit assert every descriptor run open is closed. Run that still hold one panic. Surface own
leak check, thus caller need no census.

# Address

Address is IP address, port, and explicit family, as plain data. It take no part of net tree.

### Parse

Address_Parse read dotted-quad as IPv4, and colon form as IPv6, with one "::" run and trailing
embedded IPv4 both accepted. Leading zero in octet, zone identifier, and port outside uint16 all
error. No DNS, thus parse never block.

# Stream

A Stream pairs caller-owned state with one static procedure for a bounded byte transport. Every
operation uses a caller-owned Completion and a last callback. Completion.Data holds the
operation result.

### Callback

The concrete Stream owns callback time. An immediate Stream can call back inline. A simulated or
kernel-backed Stream can submit work and call back when that work retires.

### Read

Read moves bytes from the cursor and advances it. End of memory reports Stream_EOF.

### Write

Write stores bytes at the cursor and advances it. Bounded truncation reports Stream_Short_Write.

### Read At

Read_At uses an explicit offset and does not move the cursor.

### Write At

Write_At uses an explicit offset and does not move the cursor.

### Seek

Seek accepts start, current, or end origin. An unknown origin or out-of-range result errors.

### Size

Size reports the complete storage size, independent of the cursor.

### Query

Query sends STREAM_MODE_QUERY to the Stream. Its callback reports the supported mode set in
Completion.Data.

### Flush

Flush submits buffered work. Memory and discard have no pending buffer and succeed.

### Close

Close is idempotent. Later data operations report Stream_Empty.

### Destroy

Destroy releases owned state where applicable. Memory treats it as Close.

### Errors

Empty, closed, and unsupported modes report Stream_Empty. Invalid counts and offsets report their
specific Stream error instead of changing or fabricating bytes.

### Memory

Memory uses the caller-owned slice as its complete budget and never grows it.

### Discard

Discard accepts writes, stores no bytes, and has no read or seek capability.

### Limit

Limit forwards only its remaining byte budget and reports the withheld part.

### Count

Count forwards supported modes and tallies bytes after the inner operation retires.

### Tee

Tee writes to both destinations in order and reports the smaller count.

### Composition

Composed streams preserve each inner procedure callback policy and sequence dependent callbacks.

# Allocation

IO, Network, Storage, Address, and Stream allocate no heap memory. Caller supplies retained state
and bounded queues; exhaustion fails instead of growing. Directory names borrow caller memory.
