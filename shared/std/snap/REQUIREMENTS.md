
# Equal

Snapshot_Is_Equal compares actual output against a snapshot's expected value and
reports whether they match.

### Match

A matching actual value returns true and writes no diagnostic output to the
Snapper's Output.

### Mismatch

A differing actual value returns false and prints a mismatch header and a Myers
line diff to the Snapper's Output.

### Legend

The mismatch header carries a color legend mapping minus to red expected and
plus to green actual.

# Edit

An edit-mode snapshot rewrites its own source literal with the actual output
instead of failing.

### Rewrite

Should_Edit rewrites the source `snap.Edit` literal to `snap.Init` with the
actual output and prints an UPDATED notice.

### Line Delta

A multi-line replacement records the change in line count so later edits in the
same file adjust their target line.

### Malformed

An edit-mode snapshot whose source line holds no backticked `snap.Edit` call
prints a notice naming the location and returns false.

# Panic

Expect_Panic asserts that a callback panics with a message matching the
snapshot.

### Match

A panic whose message matches the snapshot passes without failing the test.

### Mismatch

A panic whose message differs from the snapshot is reported as a failed
comparison.

# Batch

Batch_Expect drives a table of input-and-snapshot entries.

### Expect

Each entry runs in order, transforming the input and comparing the result
against the entry's snapshot. A mismatch names the entry and fails the test.

### Panic

Each entry runs in order and its panic message is compared against the entry's
snapshot.

# Run

Run captures what a callback writes to the Snapper's buffers and asserts the
combined output.

### Capture

Stdout and Stderr written by the callback are captured and asserted against the
snapshot. The returned strings borrow the Snapper's buffers until the next Run.

### Both

Stdout alone, Stderr alone, and both together each frame as their own report;
no output at all reports `snap.Run: no output`.

### Overflow

A capture past BUFFER_SIZE_MAXIMUM panics instead of growing storage.

# Storage

Every byte a Snapper writes lives in root-provided Storage bound through
Snapper_Bind_Storage; the library never allocates.

### Short

A storage slice shorter than its bound is refused at bind time.
