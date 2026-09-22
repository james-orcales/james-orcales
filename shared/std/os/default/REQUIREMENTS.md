
# Host

Host composition supplies process identity, executable path, working directory, hostname,
arguments, and environment. Arguments and environment reflect live process state; copied
arguments cannot mutate runtime storage, and self exec distinguishes nil from empty environment.

# Exit

Exit terminates the process with the requested status, verified in a child process.

# Bounds

Readers reject insufficient destination storage and preserve the shared OS value bounds.
Mapped memory is zeroed, writable through its final byte, and can be unmapped.

# Allocation

Host composition and operations avoid Go heap allocation, including memory map and unmap.
Reading the runtime environment block is exempt because the runtime returns an allocated copy.

# Files

File writes preserve content, reads fill caller storage, and absent paths return errors.
Standard error writes succeed, and a pipe descriptor is not reported as a terminal.

# Machine

Machine inspection reports architecture, core counts, physical memory, storage, and OS identity.
Supported host-specific fields are present, and physical core count does not exceed logical count.
