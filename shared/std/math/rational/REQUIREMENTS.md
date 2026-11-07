
# Representation

A Rational is a numerator and a denominator, each a fixed-width integer. The denominator is
never zero and never negative, thus the sign of a value is the sign of its numerator alone.

# Normalisation

Every value a caller receives is in lowest terms: the numerator and the denominator share no
divisor but one. One shape for one value, thus equality is a comparison and never a search.

# Conversion

From_Integer lifts a whole value. From_Ratio takes a numerator and a denominator and refuses a
zero denominator. Whole truncates toward zero. Is_Whole reports a denominator of one.

# Arithmetic

Add, Subtract, Multiply, and Divide cross-multiply and normalise. Each reports whether every step
held the width. Divide refuses a zero divisor.

# Comparison

Compare cross-multiplies and reads the order of the products. The denominators are positive, thus
the comparison needs no sign case of its own.

# Text

Into_Text writes the numerator, and a slash and the denominator when the value is no whole one.
The form reads back through From_Ratio without loss.

# Bounds

A numerator and a denominator each span 512 bits. An operation that leaves the width says so, and
a normalisation that cannot complete says so too.

# Allocation

Every operation performs zero heap allocation. A value is two fixed arrays the caller holds.
