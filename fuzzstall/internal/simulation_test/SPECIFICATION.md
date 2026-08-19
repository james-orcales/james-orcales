
# Main

Main follows a seed-generated run through the injected handle. A sweep proves the judgment
holds for any run the generator writes and for any cutting of that run's output into chunks.

### Echo

Every byte the run hands over reaches this program's output, in order, whatever chunk
boundary the seed puts in the middle of a line.

### Saturation

A stretch as long as the window with no new interesting input terminates the run and reports
124. A run with no such stretch keeps its own exit code and is never terminated.

### Refusal

A window that names no span reports 2 and starts nothing. A run that cannot start at all
reports 125.
