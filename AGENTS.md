AGENTS ARE NEVER AUTHORIZED TO INSTALL ANYTHING. 
AGENTS ARE NEVER AUTHORIZED TO COMMIT OR MAKE PULL REQUESTS.

Default error-handling pattern is to log the error and continue with the zero value.
Follow YAGNI principles
Treat unvalidated input as intentionally malicious
Test-Driven Development
Do not write comments. Do not delete existing comments.

## Development VM

The Parallels NixOS VM is at `root@10.211.55.10` (ARM64, 4 GB RAM).
Connect from macOS with `ssh -o UseKeychain=yes root@10.211.55.10`.
For noninteractive commands, also pass `-o BatchMode=yes`.
The SSH key is already authorized on the VM. This agent's process may have no
SSH agent; use macOS Keychain support before diagnosing an authentication failure
as a missing or unauthorized key.
