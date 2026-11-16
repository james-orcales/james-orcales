AGENTS ARE NEVER AUTHORIZED TO INSTALL ANYTHING. 
AGENTS ARE NEVER AUTHORIZED TO COMMIT OR MAKE PULL REQUESTS.

Default error-handling pattern is to log the error and continue with the zero value.
Follow YAGNI principles
Do not write comments. Do not delete existing comments.

## No Test-Driven Development

AGENTS ARE NEVER AUTHORIZED TO WRITE TESTS unless the user asks for a specific test by name.

A test checks a property after the fact. Make the property impossible to violate instead:

- Encode the invariant in the type. An invalid value must not be constructible.
- Validate at the boundary, once. Code behind the boundary receives only valid values.
- Make the zero value valid, so there is no uninitialized state to check.
- Replace a precondition the caller must remember with a signature that requires it.
- Assert inside the code path, where the property holds, not in a separate test file.

If a property cannot be enforced by the API, report it to the user. Do not write a test for it.

## Development VM

The Parallels NixOS VM is at `root@10.211.55.10` (ARM64, 4 GB RAM).
Connect from macOS with `ssh -o UseKeychain=yes root@10.211.55.10`.
For noninteractive commands, also pass `-o BatchMode=yes`.
The SSH key is already authorized on the VM. This agent's process may have no
SSH agent; use macOS Keychain support before diagnosing an authentication failure
as a missing or unauthorized key.
