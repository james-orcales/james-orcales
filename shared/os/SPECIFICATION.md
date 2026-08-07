
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

### Identifier

Identifier returns the stated process id, which is always positive.

### Effective User Identifier

Effective_User_Identifier returns the stated effective user id. Zero is root, so the field
carries no positive bound and a simulation states it deliberately.

### Self Exec

Self_Exec always fails with Self_Exec_Unsupported. A simulation cannot replace its own test
process, so it reports the failure rather than destroying the run.
