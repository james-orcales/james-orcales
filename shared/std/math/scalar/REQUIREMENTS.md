
# Integer Arithmetic

Absolute_Integer returns the magnitude of a signed integer. Minimum_Integer and
Maximum_Integer return the smaller and the larger of two integers. The most negative
integer has no positive magnitude, thus Absolute_Integer rejects it.

# Constants

PI, E, and the logarithm constants hold their value as a fixedpoint Number. The grid
holds about six decimal digits, thus each constant is correct to that precision only.
The deterministic tier has no float, thus a Number replaces every float64.

# Whole Numbers

Floor returns the largest whole Number that is not larger than the value, and Ceiling
returns the smallest that is not smaller. Truncate discards the fraction toward zero,
and Round moves a half away from zero. Modulo keeps the sign of the dividend.

# Roots

Square_Root returns the root of a value that is not negative. Cube_Root returns the root
of a value of either sign. Hypotenuse returns the root of the sum of two squares. It
divides the smaller leg by the larger first, thus a large leg does not overflow.

# Exponential

Exponential_2 raises two to a power, and Exponential raises E. The fraction of the
exponent selects from a table of the roots of two, and the whole part is a shift. An
exponent above the maximum has no result in the Number range, thus it is rejected.

# Logarithm

Logarithm_2 returns the base-two logarithm of a positive value. It takes the whole part
from the position of the highest set bit, then finds one fraction bit for each squaring
of the mantissa. Logarithm and Logarithm_10 scale that result by a constant.

# Power

Power raises a positive Base to any Exponent. It takes the base-two logarithm of the
base, multiplies by the exponent, and raises two to that product. The two errors
combine, thus Power is less exact than either operation alone.

# Trigonometry

Sine, Cosine, and Tangent accept an Angle in radians. Each converts the angle to turns
and uses the fixedpoint sine. That sine is Bhaskara's approximation, thus the error
reaches about 0.002 and is much larger than the grid.

# Inverse Trigonometry

Arcsine, Arccosine, and Arctangent return an Angle in radians. Arctangent uses a
rational approximation with an error of about 0.0015. Arctangent_Quotient accepts an
Opposite and an Adjacent leg and returns the angle in the correct quadrant.

# Hyperbolic

Hyperbolic_Sine, Hyperbolic_Cosine, and Hyperbolic_Tangent build on Exponential. The
exponential bound applies to each, thus an angle of large magnitude is rejected.

# Domain Errors

An out-of-domain value causes a panic, not an error value. A negative radicand, a
logarithm argument that is not positive, an exponent above the range, and an arcsine
argument outside the closed unit interval each panic.
