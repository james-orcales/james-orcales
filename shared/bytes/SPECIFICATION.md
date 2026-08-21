
# Allocation

Every exported operation performs zero heap allocations, measured separately by Zero_Allocation.
API returns scalars, input views, or Buffer views. Output writes caller storage and returns count.
Storage-owning returns, iterator closures, append growth, and hidden scratch allocation stay absent.

# Constant Facts

Constants with same domain fact use one definition. Primitive-width bounds come from
shared/math/bits. Equal numbers with different meanings remain separate definitions.

# Comparison

Equal reports byte equality. Compare gives normalized lexical order. Equal_Fold compares decoded
characters. Has_Text_Suffix compares byte storage with string without conversion storage.
Nil and empty Slice are equal. Overlap reports whether two nonempty views share byte storage.

# Search

Count, Contains, and Index forms find Slices, bytes, characters, sets, or predicate matches. Empty
separator occurs at each UTF-8 character boundary.

# Split And Join

Split and field operations fill caller-owned Slice slots with clipped input views and return
populated slot count. N forms apply Limit. Join_Into writes parts and separators into separate
caller byte storage and returns byte count.

# Transform

Map, Repeat, case, title, UTF-8 repair, and rune decoding operations write caller storage and
return populated count. Map, case, title, and UTF-8 repair destinations do not overlap source.

# Trim

Trim forms remove prefixes, suffixes, Unicode space, cut-set characters, or predicate matches.
Every returned Slice aliases input Slice.

# Replace

Replace_Into substitutes at most Replacement_Count non-overlapping matches. Negative count and
Replace_All_Into substitute every match, including empty matches between characters. Destination
storage remains separate from source, old value, and replacement.

# Cut And Clone

Cut forms divide Slice without allocation. Clone_Into copies into caller storage and returns byte
count. Clone destination may overlap source because copy order preserves bytes.

# Iteration

Lines, split sequences, and field sequences synchronously yield clipped input views and return
yielded count. Callback rejection stops traversal. No operation returns closure.

# Buffer

Buffer_Init and Buffer_Init_Text bind explicit caller storage. Buffer never grows beyond that
storage. Grow validates or compacts existing storage. Writes return byte count. Reads return count,
presence, or aliased views. Reset retains caller storage.

# Reader

Reader_Reset binds caller Slice and resets cursor. Reads copy only into caller storage or return
scalars. Unread restores eligible byte or character boundary. Seek accepts three bounded origins.

# Size Limits

Slice and Text hold at most SLICE_SIZE_MAXIMUM bytes. Split slot count holds at most
SLICES_COUNT_MAXIMUM. Buffer capacity uses same byte limit. Oversized malicious input causes panic.

# Domain Errors

Invalid Limit, Repeat_Count, Replacement_Count, Reader offset, seek origin, unread request, short
destination, harmful overlap, or Buffer growth causes panic.

# Invariant Domains

Tests reach both Boolean results, normalized orders, count boundaries, search sentinels, byte and
character boundaries, Buffer states, Reader states, and each seek origin through public operations.
