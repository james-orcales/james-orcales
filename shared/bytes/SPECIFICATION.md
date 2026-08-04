
# Constant Facts

Constants with the same domain fact use one definition. Primitive-width bounds come from
shared/math/bits. Equal numbers with different meanings remain separate definitions.

# Comparison

Equal reports byte equality. Compare gives lexical order, and Equal_Fold compares UTF-8
text through Unicode simple folding. A nil Slice is equal to an empty Slice.

# Search

Count, Contains, and the Index forms find bytes, slices, characters, sets, or predicate
matches. An empty separator occurs before and after each UTF-8 sequence.

# Split And Join

Split and Split_After divide a Slice at non-overlapping separators. The N forms apply a
Limit. Fields divides at Unicode space, Fields_Function uses a predicate, and Join joins.

# Transform

Map changes each character. Repeat copies a Slice, the case forms apply Unicode mappings,
To_Valid_UTF8 replaces invalid runs, and Runes decodes the Slice.

# Trim

The Trim forms remove prefixes, suffixes, Unicode space, cut-set characters, or predicate
matches. A returned Slice aliases the input Slice.

# Replace

Replace substitutes at most a Replacement_Count of non-overlapping matches. A negative
count and Replace_All substitute every match, including empty matches between characters.

# Cut And Clone

Cut and its prefix and suffix forms divide a Slice without allocation. Clone copies a
non-nil Slice and keeps a nil Slice nil.

# Iteration

Lines yields each newline-terminated line and a final unterminated line. The sequence split
and field forms yield the same Slices as their collection forms without a collection.

# Buffer

Buffer stores readable bytes. Buffer_To_Stream provides sequential Stream reads and writes.
Free functions read and write bytes, characters, and text. Reads move the position. Reset removes
content; Truncate keeps a readable prefix; Grow reserves capacity. Views can alias content.

# Reader

Reader reads a fixed Slice. Reader_To_Stream provides Stream reads, positioned reads, seeks, and
size queries. Free functions provide byte, character, and unread operations. Reset replaces the
Slice and resets the position. Reader_Unread_Size reports the unread byte count.

# Size Limits

A Slice and Text hold at most SLICE_SIZE_MAXIMUM bytes. An operation that would make a
larger owned result causes an Error_Too_Large panic. Buffer content and Reader source Slices use
the same limit.

# Domain Errors

An out-of-domain Limit, Repeat_Count, or Replacement_Count causes a panic. Join, Map,
Repeat, case mapping, UTF-8 repair, and replacement reject results above the size limit.
