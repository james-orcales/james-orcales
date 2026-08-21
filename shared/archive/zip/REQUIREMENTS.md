
# Bounded Archive

Reader and Writer use injected nbio Stream and caller storage. Header and File_System expose
classic ZIP metadata, raw copy, traversal, and assembly. Every count, path, record, timestamp,
mode, and buffer stays bounded. Hostile, ZIP64, multi-disk, corrupt, or unsupported input fails.

# Allocation

Every exported operation performs zero heap allocations. Reader, Writer, filesystem, header,
stored, and DEFLATE paths retain only caller-owned memory. Tests prove each public operation.
