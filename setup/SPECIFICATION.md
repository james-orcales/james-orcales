
# Main

### Owns Driver

Only the setup composition root advances the IO loop. Internal policy and the simulation receive
one shared `io.IO` value and cannot call `Run_Until`. The root drains each asynchronous file or
process submission before the shared operation returns.

### Drains Shared IO

The root returns each shared file or process submission only after its callback retires.

### Calls Internal Main

The composition root calls internal.Main once with the shared IO value and host facts. Internal Main
owns the complete bootstrap policy and returns the process status.
