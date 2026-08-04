
# Surrogates

Is_Surrogate reports whether a character can occur in a UTF-16 surrogate pair.

# Character Conversion

Decode_Character combines one high surrogate and one low surrogate. Encode_Character divides one
supplementary character into its pair. Invalid inputs produce REPLACEMENT_CHARACTER.

# Sequence Conversion

Encode converts characters to UTF-16 words. Decode converts UTF-16 words to characters. Each
invalid character or unpaired surrogate produces one replacement character.

# Append

Append_Character adds the one-word or two-word encoding of a character. An invalid character adds
REPLACEMENT_CHARACTER.

# Domain Errors

Characters and Words contain at most SEQUENCE_SIZE_MAXIMUM items. A result above this limit causes
a panic.
