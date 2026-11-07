
# Main

### Owns Driver

Only the setup composition root advances the IO loop. Internal policy and the simulation receive
one shared `io.IO` value and cannot call `Run_Until`. Internal policy records each continuation in
its runner, and the root advances the loop until the runner stops.

### Drive Runner Advances Original IO

The root alternates runner continuations with its Driver. A callback submitted through the original
shared `io.IO` value can queue the next continuation, and the root observes its retirement.

### Pump Guard Exceeds Process Deadline

The root pump guard is strictly longer than every setup process deadline. A bounded process retires
before the root can report an incomplete IO operation.

### Deinitializes Only Joined Runner

The root deinitializes its Driver only after the runner stops and every submitted operation retires.
A driver failure leaves backend resources to process exit instead of violating Driver lifecycle.

### Does Not Build Blocking IO

The root does not copy `io.IO` or replace its operations with blocking wrappers. It submits through
the original shared IO value and drives only from the runner state.

### Calls Internal Main

The composition root calls internal.Main once with the shared IO value and host facts. Internal Main
owns the complete bootstrap policy and returns its runner. The root drives that runner and returns
its process status.

### Imports No Standard IO

No setup Go file imports the standard-library `io` package. Setup uses the shared `io.IO` boundary
and its types.
