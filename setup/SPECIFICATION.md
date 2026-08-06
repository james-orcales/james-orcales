
# Main

### Owns Driver

Only the setup composition root advances the IO loop. Internal policy and the simulation receive
synchronous capabilities and cannot call Run_Until.

### Calls Internal Main

The composition root calls internal.Main once with the constructed capabilities. Internal Main
owns the complete bootstrap policy and returns the process status.
