
# Allocation

Every exported operation performs zero heap allocations, measured separately by Zero_Allocation.
Exported invariant helpers run inside each measurement. Tests never call assertion writers directly.
API returns only scalars or input Text views. Storage-owning operations stay absent.

# Comparison

Compare returns one normalized lexical order. Equal_Fold compares decoded characters through
Unicode simple folding. Neither operation creates normalized text.

# Search

Contains, Count, and Index forms find substrings, bytes, characters, sets, or predicate matches.
Index_Byte_Or_Non_ASCII finds a selected ASCII byte or first non-ASCII byte. Empty separator
occurs at each UTF-8 character boundary.

# Split And Join

Split operations fill caller-owned Text slots and return populated slot count. Field and line
operations fill caller-owned Text slots with input views. Join writes into caller-owned byte
storage. Insufficient destination storage causes invariant panic.

# Transform

Clone, map, repeat, replace, case, title, and UTF-8 repair operations write into caller-owned byte
storage. Returned Bytes always view that storage and never own backing memory.

# Trim And Cut

Trim forms remove cut-set characters, predicate matches, Unicode space, prefixes, or suffixes.
Cut forms divide Text and report presence. Every returned Text aliases input storage.

# Builder

Builder zero value owns one fixed 4,096-byte array. Writes never grow storage. Builder_Bytes returns
a view into that array, and reset retains storage.

# Reader

Reader zero value holds one Text view and byte cursor. Reads copy only into caller storage. Unread
restores most recent character boundary.

# Replacer

Replacer borrows caller-owned ordered rules. Replacement output uses caller-owned byte storage.
Rules and output never receive package-owned copies.

# Size Limit

Text holds at most TEXT_SIZE_MAXIMUM bytes. Every public operation validates Text before scan.
Oversized, malicious input causes invariant panic.
