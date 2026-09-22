
# Integer

Unsigned_Size and Integer_Size report exact gob scalar storage. Encode operations write caller
storage; decode operations accept stdlib scalar widths and leave trailing input to the caller.

# Boolean

Boolean values use gob unsigned zero and one. Decode accepts every canonical nonzero unsigned value
as true.

# Float

Float operations accept and return IEEE binary64 bits. Gob byte reversal happens before unsigned
integer framing, preserving every finite, infinite, and NaN representation without hidden storage.

# Complex

Complex operations frame real bits before imaginary bits as two adjacent gob floats. Each part
keeps its own scalar length while one aggregate bound covers caller input and output.

# Bytes

Bytes_Size reports the exact count prefix plus content. Encode_Bytes_Into writes caller storage;
Decode_Bytes borrows its decoded content from input.

# Bounds

Scalar encodings follow their fixed word widths. Byte input, output, count prefixes, consumption,
and error positions follow formulas from the shared slice boundary. Oversized slices panic.

# Allocation

Every exported operation performs zero heap allocation with assertion tracking enabled. Encoded
output remains caller-owned and decoded bytes remain borrowed.
