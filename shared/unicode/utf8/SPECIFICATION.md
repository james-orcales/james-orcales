
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

# Allocation

Every public operation performs zero heap allocation.

# Domain Errors

Bytes and Text contain at most SEQUENCE_SIZE_MAXIMUM bytes. A result above this limit causes a
panic. Encode_Character and Append_Character also cause a panic when caller storage is too short.
