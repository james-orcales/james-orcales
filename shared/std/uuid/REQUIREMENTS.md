
# Parse Round Trips String

Parse of a UUID's String returns the same 16 bytes, across the canonical, URN,
braced, and unhyphenated forms; a wrong length, a non-hex digit, or a 38-character
string that is not brace-wrapped at the two ends is an error.

# Version And Variant Are Stamped

Each generator stamps its version nibble (1, 2, 3, 4, 5, 6, 7) and the RFC 9562
variant bits, so UUID_Version and UUID_Variant read back what produced the value.

# Seed Reproduces Sequence

Two generators over the same seed and clock emit an identical stream of UUIDs, and
distinct seeds diverge — the property deterministic simulation replays from a seed.

# V7 Is Monotonic

Successive V7 draws from one generator strictly increase byte-wise, even within a
single clock millisecond, so they sort in creation order.

# Text Marshaling Round Trips

UUID_Marshal_Text then UUID_Unmarshal_Text recovers original UUID, and text is
canonical 36-character form.

# Binary Marshaling Round Trips

UUID_Marshal_Binary emits 16 raw bytes and UUID_Unmarshal_Binary recovers them; a
slice of any other length is an error.

# JSON Null Round Trips

A valid Null_UUID passes through Null_UUID_Marshal_JSON and Null_UUID_Unmarshal_JSON;
an invalid one marshals to JSON null and unmarshals from null as invalid.

# Database Scan Reads Value

UUID_Scan reads UUID from shared driver text and 16 raw bytes. UUID_Value emits
canonical text through shared closed driver union.

# Name Based Is Deterministic

V3 and V5 over the same namespace and data always return the same UUID, and a
different namespace or data changes it.

# Time Round Trips Through UUID

UUID_Time reads back the timestamp a V1, V6, or V7 embeds, and Time_Unix converts it
to the Unix seconds the clock supplied.

# DCE Embeds Domain And Identifier

A DCE Security UUID carries version 2, the requested domain, and the caller's
identifier, which UUID_Domain and UUID_Identifier read back.

# Entire Package Is Zero Allocation

Every UUID operation, including rejected input and host construction, allocates zero
heap memory. Caller owns UUID, encoded output, and host seed scratch storage.
