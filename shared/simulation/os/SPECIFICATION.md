
# Virtual OS

A Virtual_OS is plain data, so a simulation reads back exactly the ambient state it stated.
It takes no seed and no clock, and every read is repeatable within one run.

### Arguments

Arguments copies stated argv into caller-owned storage. Caller must provide enough entries for
whole answer. Returned count identifies populated prefix. Editing destination does not change what
later call reads.

### Environment

Environment copies stated variables into caller-owned storage, each one a "NAME=VALUE" string.
Caller must provide enough entries for whole answer. Returned count identifies populated prefix.

### Variable

Variable returns the value of one name and found true, or the empty value and found false
when the name is unset. A name set to the empty value is found. A later duplicate wins.

### Executable

Executable returns the stated image path and no error.

### Working Directory

Working_Directory returns the stated directory and no error.

### Hostname

Hostname returns the stated machine name and no error.

### Process Identifier

Process_Identifier returns the stated process id, which is always positive.

### Effective User Identifier

Effective_User_Identifier returns the stated effective user id. Zero is root, so the field
carries no positive bound and a simulation states it deliberately.

### Self Exec

Self_Exec always fails with Self_Exec_Unsupported. A simulation cannot replace its own test
process, so it reports the failure rather than destroying the run.

# OS

Seeded backend draws signal grain and exit code from seed, then retires both on injected loop.
Caller owns backend state and bounded operation storage; constructor rejects empty or oversized
storage, and callback frees entry before running so it may submit immediately.

### Self Exec

Self_Exec replaces process image and returns only on failure. Nil environment preserves ambient
values. Non-nil slice is complete replacement environment, thus empty slice inherits nothing.

### Watch Signal

Watch_Signal requires positive finite deadline; signal arriving first fires once, and deadline wins
ties with Deadline_Exceeded and SIGNAL_EXPIRED. Caller retains armed signal because expired value
identifies no watch.

### Spawn

A spawn requires a positive finite deadline; natural completion returns the seed-drawn exit code,
while the deadline wins ties with Deadline_Exceeded. The real backend kills its subprocess group,
bounds pipe cleanup to one second, and returns partial output.
