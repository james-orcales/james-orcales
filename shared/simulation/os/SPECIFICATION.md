
# Virtual OS

A Virtual_OS is plain data, so a simulation reads back exactly the ambient state it stated.
It takes no seed and no clock, and every read is repeatable within one run.

### Arguments

Arguments returns the stated argv. Each call returns a copy, so a caller that edits the
result does not change what a later call reads.

### Environment

Environment returns the stated variables, each one a "NAME=VALUE" string. Each call returns
a copy.

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

The seeded backend draws the signal grain and the exit code from its seed and retires both
operations on the injected loop, so a run reproduces and nothing is scriptable.

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
