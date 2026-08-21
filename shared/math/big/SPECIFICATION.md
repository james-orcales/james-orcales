
# Representation

Int keeps a sign and a normalized little-endian magnitude. Its zero value is zero. Each Int owns
WORD_COUNT_MAXIMUM words inline, so caller placement owns all storage. WORD_COUNT_MAXIMUM derives
from repository byte bound and machine word size.

# Conversion

Machine, byte, and word setters copy bounded magnitudes into caller Int storage.
Byte and word outputs use caller storage and preserve short destinations.
Narrow machine conversions report STATUS_VALUE_OVERFLOW instead of truncating.

# Magnitude

Int_Bytes_Into writes minimal unsigned big-endian magnitude. Int_Fill_Bytes zero-pads complete
validated storage. Short destinations remain unchanged. Trailing-zero count ignores sign.

# Sign

Int_Sign separates negative, zero, and positive values. Int_Absolute and Int_Negate write caller
destinations. Zero never carries a negative sign.

# Comparison

Int_Compare orders signed values. Int_Compare_Absolute orders magnitudes. Both accept destination
aliasing because they write nothing.

# Addition

Int_Add and Int_Subtract write caller destinations and permit the destination to be either input.
A mathematical result beyond BIT_COUNT_MAXIMUM returns STATUS_VALUE_OVERFLOW and leaves the
destination unchanged.

# Multiplication

Int_Multiply writes one exact product through caller-owned workspace and permits destination
aliasing. Overflow leaves destination unchanged.

# Division

Int_Quotient, Int_Remainder, and Int_Quotient_Remainder implement truncated division.
Int_Divide_Modulus implements Euclidean division. Caller workspace preserves aliases; invalid
division leaves destinations unchanged.

# Shift

Shift_Count_Validate bounds work before shifting. Int_Shift_Left reports overflow transactionally;
Int_Shift_Right preserves arithmetic right-shift semantics for negative values.

# Bitwise

Int_And, Int_And_Not, Int_Or, Int_Xor, and Int_Not preserve infinite two's-complement semantics.
Caller workspace makes aliases safe. Unrepresentable negative boundary leaves destination unchanged.
Bit_Index_Validate bounds Int_Bit and Int_Set_Bit before any bit-addressed work begins.

# Number Theory

Integer combinatorics preserve stdlib edge results and atomic overflow.
GCD, roots, and modular operations preserve failures; modular roots take a caller nonresidue.
Random and probable-prime work is bounded; Jacobi rejects bad denominators and Lucas exhaustion.

# Rational

Rat stores one reduced numerator and positive denominator within derived component bounds.
Operations, conversions, text, analysis, and complete stdlib parsing use caller storage.
Zero denominator and component overflow leave destination unchanged.

# Float

Float stores bounded precision, rounding policy, accuracy, sign, mantissa, and normalized exponent.
Binary32, binary64, bounded parsing, and caller-workspace arithmetic preserve policy, signed zero,
and infinity. Bound failures are atomic; overflow maps to zero or infinity where specified.

# Text

Base validation precedes bounded Int and Float conversion.
Int text covers bases 2 through 62. Float text covers b, p, x, e, E, f, g, and G.
Parsers consume complete stdlib syntax transactionally. Float exact factors are explicitly bounded.

# Serialization

Int gob encoding matches stdlib version-one wire format through bounded caller storage.
Rat and Float gob encoding preserve implicit denominator, policy, and aligned mantissa wire fields.
Gob decode rejects unsupported versions and oversized magnitudes without destination mutation.

# Bounds

Every exported operation has fixed work bounded by WORD_COUNT_MAXIMUM. Bytes_Validate rejects
oversized unvalidated input. Failures leave mutation destinations unchanged.

# Allocation

Every exported operation performs zero heap allocation. Int stores its complete bound inline, and
each operation returns only scalar values and typed status.
