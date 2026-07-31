
# Main

### Stays Thin

The setup composition root contains fewer than 90 lines. It constructs operating-system
capabilities, advances the IO loop, injects dependencies, and owns no setup policy.

### Calls Internal Main

The composition root calls internal.Main once with the constructed capabilities. Internal Main
owns the complete bootstrap policy and returns the process status.

# File System

### Reads And Writes

The setup composition root converts asynchronous file operations into synchronous operations for
the setup library. The root owns the Driver and reports file contents only after Close retires.

### Reports Driver Error

The file-system adapter reports a Driver error and does not report a partial operation as success.

# Spawn Command

### Uses Finite Duration

Each setup process has a six-hour maximum duration. Expiry stops its process group through
shared/io, while the root drives the loop until the process callback retires.

### Reports Driver Error

The command adapter returns a nonzero process result when the Driver reports an error before the
process callback retires.
