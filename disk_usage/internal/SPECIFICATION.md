
# Analyze

Analyze walks a file system once and returns every file and directory with its
cumulative disk usage, charging each file's physical bytes up to the root.

### Cumulative Sizes

A file's bytes are its own physical allocation; a directory's bytes are the sum beneath
it, recursively, and the root "." holds the whole tree's total.

### Empty Directories

A directory holding no file is still recorded, with zero bytes, so the model stays
complete even though Render later hides it.

### Depth

The root counts as depth one and depth grows by one per slash-separated segment, so a
path's depth is the number of components in its displayed label.

### Order

Entries group directories ahead of files, each group by size descending then path
ascending, so folders headline the listing and the order is deterministic.

### Hardlinks

A file's bytes are charged once per identity, at the first path that reaches it; a later
hardlink to the same inode adds nothing, matching how du counts shared data.

# Render

Render writes the report as an aligned table of size and path, one row per entry that
passes the filters, sizes human-readable in powers of 1024.

### Depth Limit

Depth_Max hides entries deeper than the limit; -1 shows the whole tree, and a zero-byte
entry is always hidden so an empty directory never prints.

### Minimum

An entry smaller than Minimum is hidden, trimming the long tail of small files and
directories so only the space worth noticing shows.

### Kind

Directory_Only prints only directories and Files_Only prints only files; the root "."
is a directory, dropped under Files_Only.

### Label

A row names the root label for the "." entry, otherwise the root label joined to the
relative path.

### Grouping

Rows are grouped by depth beneath a per-depth header, the depths ascending, so each
level of the tree reads as its own block.

### Json

With -json the kept entries marshal through flatjson to a top-level array of flat
objects, one per row, each carrying its path, bytes, kind, and depth.

# Main

Main parses the command line, analyzes the directory argument, and renders the table.

### Filters

The -dir-only and -files-only flags name disjoint halves of the tree, so requesting
both is a usage error rather than a silently-resolved preference.
