
# Witness

The simulation drives internal.Main end to end so every sloc invariant is exercised through
the one public entry point. These leaves anchor the blackbox suite; the fuzz target then
explores the same driver, and the coverage recorder fails the run on any unwitnessed boundary.

### Tree

A directory of recognized files is walked, classified, and rendered through one Main call, so
the walk, the per-line scanner, the grouping, and both renderers are driven together rather
than reached one helper at a time.

### Languages

One file per recognized extension and per recognized bare name is counted in a single run, so
every language constructor executes and the table's own field bounds — comment tokens, string
delimiters, test markers — are witnessed across the whole seeded set.

### Boundaries

A line at the scan window, a tree at the file bound, and a table whose every column reaches
its width are counted, so the widths and tallies the library states its invariants against are
reached rather than assumed.

### Failure

A stat failure, a read failure, a binary file, an oversized file, and an unrecognized path
each take a branch a clean run never reaches, so the drop and error paths are driven through
Main rather than left dark.
