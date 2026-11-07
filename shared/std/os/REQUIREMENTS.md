
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

### Memory Map

Memory_Map carves next region of exactly size bytes from caller-owned arena and zeroes it.
Capacity equals length, so append cannot spill into next region. Regions never overlap. Arena
too small for region returns Memory_Exhausted.

### Memory Unmap

Memory_Unmap returns no error and releases nothing.

### Exit

Exit records the status and marks the simulation exited, then returns. A simulation cannot end
its own test process, so the caller stops on its own after the call.

# OS

Caller captures allocating ambient values in Host before construction. Arguments and Environment
read the process live. Host and Self_Exec_Workspace remain caller-owned. Nothing takes a loop.
Every OS operation allocates zero heap bytes, except Environment and a nil-environment Self_Exec.

### Self Exec

Self_Exec replaces process image and returns only on failure. Nil environment uses live values.
Non-nil environment replaces all values; empty inherits nothing. Workspace encodes raw execve input.
Embedded NUL returns EINVAL. Input exceeding caller-owned workspace returns E2BIG.
