
# Conversion

From_Integer lifts a Whole_Integer to an Integer_Number. From_Ratio lifts a numerator
and denominator. Whole returns a Whole_Integer. Is_Integer reports whether a Number has
no fractional units.

# Arithmetic

Multiply accepts a Multiplicand and a Multiplier. Divide accepts a Dividend and a
Divisor. Each operation uses a 128-bit intermediate, so the scale cancels without an
int64 product overflow. Native operators perform addition and subtraction.

# Ratio

Apply scales a value by a Ratio, a dimensionless fixed-point multiplier kept a
distinct type so the call needs no input struct and a ratio reads as a plain constant.

# Square Root

Square_Root returns a Number_Root, and Square_Root_Scaled returns a Scaled_Root.
Integer_Root accepts separate High_Word and Low_Word values. It returns the floored
Root_Integer of the 128-bit radicand.

# Sine

Sine_Turns returns a Sine in the closed interval from negative one to one. It reduces
the angle to one period and uses Bhaskara's rational approximation.

# Format

Into_Text writes a value into caller storage with zero to six fractional digits. It rounds
dropped remainder half away from zero. Short storage stays unchanged.

# Serialization

Into_JSON writes one Number as a bare decimal into caller storage. From_JSON reads one bounded
bare decimal. Both return scalar results, so serialization owns no hidden storage.

# Allocation

Every operation performs zero heap allocation. Text operations write caller storage.
