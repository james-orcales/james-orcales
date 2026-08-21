
# Values

A Value is a kind and the storage every kind needs, and Kind_Of reads the kind. Unknown names a
value the grammar could not fold, and every operation on an unknown yields an unknown rather than
a wrong answer.

# Kinds

A value is unknown, a truth, a text, a whole number, a number, or a pair of numbers. A whole
number and a number share their storage, because a whole number is a ratio over one.

# Construction

Make_Unknown, Make_Boolean, Make_Text, Make_Int_64, Make_Ratio, and Make_Complex build a value,
and a truth rides in the real part as one or zero. Make_From_Literal reads a Go integer literal of
any base, and one that ends in the imaginary mark folds to a pair whose real part is zero.

# Reading

Kind_Of names what a value holds. Boolean_Value, Text_Value, and Int_64_Value read a value back
and report whether the kind and the width allowed it. Into_Text writes any value into caller
storage, and a pair writes both parts.

# Unary Operations

Plus, Minus, Complement, and Not each apply to the kinds Go admits and yield an unknown for the
rest. Minus reverses both parts of a pair, and Complement takes a whole number alone.

# Binary Operations

Add, Subtract, Multiply, Quotient, Remainder, and the four bit operations follow Go: a quotient
of two whole numbers is whole, and a quotient of numbers is exact. A pair adds part by part, and
it multiplies and divides as Go states.

# Comparison

Compare reads two values of one kind and reports the order Go states. A pair orders by its real
part and then by its imaginary part, thus one order names the one comparison Go allows a pair.

# Shifts

Shift_Left and Shift_Right move a whole number by a count. A shift of anything else, or past the
width, yields an unknown.

# Bounds

A whole number and a number each span 512 bits of numerator and denominator. Text spans the
source it views, and a written form spans no more than that. An operation past a bound yields an
unknown.

# Allocation

Every operation performs zero heap allocation. A value is fixed storage the caller holds, and
Into_Text writes into caller storage.
