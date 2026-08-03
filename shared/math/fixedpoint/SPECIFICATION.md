
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

Format renders a value as Text with zero to six fractional digits. It rounds the
dropped remainder half away from zero.

# Serialization

A Number marshals to and from JSON as a bare decimal number, so a struct of Numbers
serializes the way the float it replaced once did.
