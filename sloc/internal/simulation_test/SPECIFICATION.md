
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

### Boundary Model

The line-count bound is assembled from production-reachable per-file classifications in three
languages. Its accepted files are modeled totally; ordinary recipes still use the production
scanner, while aggregation and rendering retain the real bounds.

### Source Bound Model

The file exactly at the source-byte bound retains all of its bytes so the bounded read remains
real, but its repeated line partition is modeled. The file one byte over remains rejected by the
production size check before classification.

### Classifier Input Bounds

The largest accepted path and source use the production byte classifier. The source has one wide
line so the test reaches both input bounds without an expensive per-line scan.

### Model Path Minimum

The shortest recognized path uses the modeled classifier through Main. Hidden-file selection lets
the walk count the bare `.c` path while the empty source keeps its classification exact.

### Wide Model

The wide-table fixtures model their repeated per-file line partitions. The walk still visits
production file counts and both renderers still consume full reports; representative sources
are checked against the production scanner before their counts are scaled.

### Table Width Model

One modeled report combines the maximum path, a five-digit per-language file tally, three
languages at the code/comment/blank bounds, and a source/test split. Rendering that report with
files shown reaches the production table-width bound exactly.

### Shared Properties

The source and test partition types use the same file, line, and dropped-count limits. One modeled
report gives each test count its minimum, one, two, and maximum values through both renderers.

### Failure

A stat failure, a read failure, a binary file, an oversized file, and an unrecognized path
each take a branch a clean run never reaches, so the drop and error paths are driven through
Main rather than left dark.
