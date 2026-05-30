
# Sim

The simulated IO backend schedules every operation to complete at now plus a modeled
latency on its injected clock, then fires each one when the clock reaches its
Ready_At — TigerBeetle's in-memory Storage and PacketSimulator, fully reproducible.

### Timeout

A timeout fires exactly when the virtual clock reaches its deadline, off the same
Ready_At queue the IO completions use, so every wait rides one timeline.

### Read

A read completes after the modeled latency and reports the buffer length, with no
real syscall and no waiting.
