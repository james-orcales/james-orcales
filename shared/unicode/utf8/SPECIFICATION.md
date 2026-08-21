
# Complete Sequences

Full_Character reports whether the first bytes contain a complete UTF-8 character. An invalid
first encoding is complete because the decoder consumes one byte for its replacement character.

# Decode

Decode_Character and Decode_Final_Character return the first or final decoded character. The Text
forms have the same behavior. Empty input consumes zero bytes, and invalid input consumes one byte.

# Encoding

Character_Size gives the UTF-8 size of a character. Encode_Character writes one encoding into
caller storage. Append_Character adds one encoding into caller capacity and substitutes
REPLACEMENT_CHARACTER for an invalid character.

# Character Count

Character_Count and Character_Count_Text count valid and invalid encodings. Each invalid byte
counts as one replacement character.

# Validation

Character_Start identifies a possible first byte. Valid and Valid_Text require complete, shortest
UTF-8 encodings. Valid_Character rejects negative values, surrogates, and values above RUNE_MAX.

# Raw Storage Extensions

Buffer and Reader character operations decode or encode UTF-8 while borrowing shared/bytes cursor
storage. Character unread restores the complete encoded boundary. Raw byte ownership remains in
shared/bytes.

# Search

Contains, Index, and Last_Index forms decode source before matching a character, character set, or
predicate. Returned indexes identify encoded byte positions. Equal_Fold compares decoded
characters through Unicode simple folding.

# Fields And Iteration

Fields operations split Unicode-space or predicate-delimited character runs into clipped source
views. Sequence forms synchronously yield those views and stop when callback rejects continuation.

# Transform

Map, case conversion, legacy title conversion, invalid UTF-8 repair, and rune decoding write
caller-owned storage. Case operations use shared/unicode/ucd. Special variants accept
ucd.Special_Case directly. Byte destinations never overlap source.

# Trim

Trim forms remove decoded cut-set characters, Unicode whitespace, or predicate matches. Returned
Bytes aliases source storage.

# Allocation

Every public operation performs zero heap allocation. Transforms and collection operations use
caller-owned storage.

# Domain Errors

Bytes and Text contain at most SEQUENCE_SIZE_MAXIMUM bytes. Fields and decoded character storage
use derived bounded counts. A result above these limits, short destination, overlapping transform
storage, or invalid unread request causes panic. Encode_Character and Append_Character also cause
panic when caller storage is too short.
