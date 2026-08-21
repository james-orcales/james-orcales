
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

Caller captures allocating ambient values in Host before construction. Kernel reads stay live.
Host and Self_Exec_Workspace remain caller-owned. Nothing owns a queue or takes a loop.
Every OS operation, including construction and returned failures, allocates zero heap bytes.

### Self Exec

Self_Exec replaces process image and returns only on failure. Nil environment uses Host values.
Non-nil environment replaces all values; empty inherits nothing. Workspace encodes raw execve input.
Embedded NUL returns EINVAL. Input exceeding caller-owned workspace returns E2BIG.
