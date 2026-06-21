
# Conversion

From_Integer lifts a whole number and From_Ratio a quotient into fixed-point; Whole
truncates back toward zero, and Is_Integer reports a value carrying no fractional part.

# Arithmetic

Multiply and Divide combine two fixed-point values through a 128-bit intermediate, so
the scale cancels without the overflow a bare int64 product would hit. Addition and
subtraction stay the native operators, the shared scale already aligning them.

# Ratio

Apply scales a value by a Ratio, a dimensionless fixed-point multiplier kept a
distinct type so the call needs no input struct and a ratio reads as a plain constant.

# Square Root

Square_Root roots a fixed-point value and Square_Root_Scaled a plain integer;
Integer_Root floors the root of a 128-bit radicand, the primitive both build on.

# Sine

Sine_Turns returns the sine of an angle measured in whole turns, reduced to one
period, through Bhaskara's rational approximation so no irrational pi enters.

# Format

Format renders a value as decimal text with a set number of fractional digits,
rounding the dropped remainder half away from zero.

# Serialization

A Number marshals to and from JSON as a bare decimal number, so a struct of Numbers
serializes the way the float it replaced once did.
