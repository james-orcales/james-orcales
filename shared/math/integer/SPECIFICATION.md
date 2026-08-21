
# Representation

An Integer is a two's complement value of 512 bits held in eight little-endian limbs. Two's
complement rather than a sign and a magnitude, because Go states its bitwise operators over
two's complement and a fixed width makes that reading exact.

# Conversion

From_Int_64 lifts a machine integer. To_Int_64 lowers one and reports whether the value fits.
Zero and One name the two values every caller needs.

# Addition

Add and Subtract carry and borrow across the limbs. Each reports whether the true result fits
the width, thus a caller learns of an overflow rather than reading a wrapped value.

# Multiplication

Multiply forms the double-width product of the magnitudes and applies the sign of the operands.
It reports whether the product fits the width.

# Division

Divide truncates its quotient toward zero and gives the remainder the sign of the dividend, which
is what Go states. A zero divisor is refused rather than trapped.

# Comparison

Compare reports whether the left value stands before, at, or after the right one. Sign and
Is_Zero read one value alone.

# Shifts

Shift_Left reports an overflow when a bit leaves the width. Shift_Right carries the sign into the
vacated bits, which is the arithmetic shift Go states for a signed value.

# Bitwise

And, Or, Exclusive_Or, And_Not, and Not run limb by limb. Two's complement makes each exact for a
negative operand without a case of its own.

# Common Divisor

Greatest_Common_Divisor runs Euclid over the magnitudes and returns a value that is never
negative. The divisor of zero and zero is zero.

# Text

From_Text reads a Go integer literal in base two, eight, ten, or sixteen, underscores included.
Into_Text writes the decimal form into caller storage and returns the byte count.

# Bounds

A value spans 512 bits and its largest magnitude spells 154 decimal digits. Every operation
that can leave the width says so, thus no result is a wrapped value a caller could mistake for
an exact one.

# Allocation

Every operation performs zero heap allocation. A value is a fixed array the caller holds, and
Into_Text writes into caller storage.
