
# Allocation

Every exported operation performs zero heap allocations, measured separately by Zero_Allocation.
API returns scalars, input views, or Buffer views. Output writes caller storage and returns count.
Storage-owning returns, iterator closures, append growth, and hidden scratch allocation stay absent.

# Constant Facts

Constants with same domain fact use one definition. Primitive-width bounds come from
shared/math/bits. Equal numbers with different meanings remain separate definitions.

# Comparison

Equal reports byte equality. Compare gives normalized lexical order. Has_Text_Suffix compares raw
byte storage with a string without conversion storage. Nil and empty Slice are equal. Overlap
reports whether two nonempty views share byte storage.

# Search

Count, Contains, and Index forms find byte sequences or individual bytes. Empty separator occurs
at every byte boundary. No operation decodes Unicode.

# Split And Join

Split operations fill caller-owned Slice slots with clipped input views and return populated slot
count. Empty separator splits individual bytes. N forms apply Limit. Join_Into writes parts and
separators into separate caller byte storage and returns byte count.

# Repeat

Repeat_Into writes consecutive raw copies into caller storage and returns byte count.

# Trim

Trim_Prefix and Trim_Suffix remove exact byte sequences. Returned Slice aliases input Slice.

# Replace

Replace_Into substitutes at most Replacement_Count non-overlapping matches. Negative count and
Replace_All_Into substitute every match, including empty matches between bytes. Destination
storage remains separate from source, old value, and replacement.

# Cut And Clone

Cut forms divide Slice without allocation. Clone_Into copies into caller storage and returns byte
count. Clone destination may overlap source because copy order preserves bytes.

# Iteration

Lines and split sequences synchronously yield clipped input views and return yielded count. Empty
separator yields individual bytes. Callback rejection stops traversal. No operation returns a
closure.

# Buffer

Buffer_Init and Buffer_Init_Text bind explicit caller storage. Buffer never grows beyond that
storage. Grow validates or compacts existing storage. Writes and reads operate on raw bytes. Reset
retains caller storage. Read_Operation lets an encoding package record one bounded grouped read.

# Reader

Reader_Reset binds caller Slice and resets cursor. Reads copy only into caller storage or return
raw byte scalars. Unread restores one byte. Previous lets an encoding package record one grouped
read boundary. Seek accepts three bounded origins.

# Size Limits

Slice and Text hold at most SLICE_SIZE_MAXIMUM bytes. Split slot count holds at most
SLICES_COUNT_MAXIMUM. Buffer capacity uses same byte limit. Oversized malicious input causes panic.

# Domain Errors

Invalid Limit, Repeat_Count, Replacement_Count, Reader offset, seek origin, unread request, short
destination, harmful overlap, or Buffer growth causes panic.

# Invariant Domains

Tests reach both Boolean results, normalized orders, count boundaries, search sentinels, byte
boundaries, Buffer states, Reader states, and each seek origin through public operations.
