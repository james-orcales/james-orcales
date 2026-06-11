
# Plan

Plan returns the writes that bring the home directory in line with the source dotfiles.

### Empty Source Yields No Writes

A source tree with no files produces nothing to sync.

### Missing Destination Is Created

A source file absent from the home directory is written to its mirrored path.

### Identical Destination Is Skipped

A source file whose home copy already holds the same bytes is left untouched.

### Differing Destination Is Overwritten

A source file whose home copy differs is rewritten with the source bytes.

### Nested Path Mirrors Under Home

A nested source file maps to the same relative path joined under the home directory.

# Main

Main plans the sync and writes each pending file through the injected writer.

### Writes Planned Files

Each planned file's contents reach the writer at its mirrored path, reporting success.

### Skips Identical Destination

A home directory already matching the source yields no writes on a repeat run.

### Reports Write Failure

A writer error makes the run report a non-zero exit code.

### Applies Macos Defaults

On darwin the macos defaults commands run through the injected runner after the sync.

### Skips Macos Defaults Off Darwin

On any operating system other than darwin no defaults commands run.
