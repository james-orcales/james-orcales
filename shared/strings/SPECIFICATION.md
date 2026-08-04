
# Comparison

Compare gives lexical order through three normalized values. Equal_Fold compares UTF-8 text
through Unicode simple folding. Clone returns the same value through a new text allocation.

# Search

Count, Contains, and the Index forms find substrings, bytes, characters, sets, or predicate
matches. Index_Byte_Or_Non_ASCII combines a byte-set search with the first non-ASCII byte. An
empty separator occurs at each UTF-8 character boundary.

# Split And Join

Split and Split_After divide Text at non-overlapping separators. The N forms apply a Limit.
Fields divides at Unicode space, and Fields_Function uses an injected predicate. Join joins Texts.

# Transform

Map changes each character. Repeat copies Text. The case forms use shared/unicode/ucd mappings.
The special-case forms accept ucd.Special_Case. To_Valid_UTF8 uses shared/unicode/utf8 decoding.
Replace substitutes non-overlapping matches.

# Trim And Cut

The Trim forms remove prefixes, suffixes, Unicode space, cut-set characters, or predicate matches.
Cut and its prefix and suffix forms divide Text and report if the selected value was present.

# Iteration

Lines yields each newline-terminated line and a final unterminated line. The split and field
sequences give the applicable collection results without a result collection.

# Builder

Builder stores bounded owned text. Builder_To_Stream supplies its write, flush, close, destroy,
and query operations through shared/io.Stream. Reset removes the content and reopens the stream.

# Interface Boundary

Builder, Reader, and Replacer have no exported methods. They implement no standard-library
interface. Package functions and shared/io.Stream supply all stateful operations.

# Reader

Reader reads bounded Text. Reader_To_Stream supplies shared stream operations.
A seek can set any nonnegative position. A read at or after the end reports Stream_EOF.
The unread functions reverse byte and character reads directly.

# Replacer

Replacer keeps an immutable copy of ordered old-new rules. Replacer_Write_Text writes through a
shared/io.Stream, and Replacer_Replace returns bounded Text.

# Size Limits

Text and Slice hold at most TEXT_SIZE_MAXIMUM bytes. An owned result above that limit causes an
Error_Too_Large panic. Each collection and numeric argument also has a declared bound.

# Domain Errors

An out-of-domain Limit, Repeat_Count, Replacement_Count, Growth_Count, or collection causes a
panic. A shared stream reports its shared/io stream errors without a standard-library error value.
