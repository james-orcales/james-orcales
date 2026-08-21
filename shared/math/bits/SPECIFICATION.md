
# Word Size

WORD_SIZE gives the bit width of the machine word, either 32 or 64. The width forms the
upper bound of each count that a machine-word operation returns.

# Integer Limits

The signed limits INTEGER_8_MINIMUM through INTEGER_64_MAXIMUM and the unsigned limits
WORD_8_MINIMUM through WORD_64_MAXIMUM give the bounds of each machine width. Every
package above names these rather than repeating them.

# Decimal Digit Bound

DECIMAL_DIGIT_BINARY_LOGARITHM_SCALE and DECIMAL_DIGIT_BINARY_LOGARITHM_CEILING provide one
fixed-point upper bound for log10(2). Binary-width packages reuse it to derive decimal widths.

# Byte Units

KILOBYTE_BYTES through EXABYTE_BYTES give the SI decimal byte units as powers of 1000.
KIBIBYTE_BYTES through EXBIBYTE_BYTES give the IEC binary byte units as powers of 1024.
The constants are untyped, thus an integer type can use each value that fits its range.

# Leading Zeros

Leading_Zeros_8 through Leading_Zeros_64 count the zero bits above the highest set bit.
Leading_Zeros counts the same bits in a machine word. A zero operand has no set bit,
thus each function returns the full width.

# Trailing Zeros

Trailing_Zeros_8 through Trailing_Zeros_64 count the zero bits below the lowest set bit.
Trailing_Zeros counts the same bits in a machine word. A zero operand returns the full
width, the same result that Leading_Zeros gives.

# Ones Count

Ones_Count_8 through Ones_Count_64 count the set bits. Ones_Count counts the set bits of
a machine word. The count and the leading-zero count of one operand never both reach the
full width.

# Bit Size

Bit_Size_8 through Bit_Size_64 give the position above the highest set bit, which is the
smallest number of bits that holds the operand. Bit_Size gives the same position in a
machine word. The width less the leading-zero count gives each result.

# Rotation

Rotate_Left_8 through Rotate_Left_64 turn the bits left by a Rotation and return the
bits that leave the top to the bottom. Rotate_Left turns a machine word. A negative
Rotation turns the bits right by that magnitude.

# Reversal

Reverse_8 through Reverse_64 exchange the bit at each position with the bit at the
mirror position. Reverse reverses a machine word. Two applications of one function
return the operand.

# Byte Reversal

Reverse_Bytes_16 through Reverse_Bytes_64 exchange the byte at each position with the
byte at the mirror position and leave the bits in each byte unchanged. Reverse_Bytes
reverses the bytes of a machine word.

# Addition

Add_32, Add_64, and Add_Word add two operands and a Carry_In of zero or one. Each
returns the low word of the sum and a Carry_Out of zero or one. The pair holds the full
sum, thus a chain of calls adds an integer of any width.

# Subtraction

Subtract_32, Subtract_64, and Subtract_Word subtract a subtrahend and a Borrow_In of
zero or one from a minuend. Each returns the low word of the difference and a Borrow_Out
of zero or one.

# Multiplication

Multiply_32, Multiply_64, and Multiply_Word multiply two operands and return the high
word and the low word of the double-width product. The product of two full-width
operands does not fit one word, thus both words are necessary.

# Division

Divide_32, Divide_64, and Divide_Word divide a double-width dividend by a divisor and
return the quotient and the remainder. Remainder_32, Remainder_64, and Remainder_Word
return the remainder alone and accept a quotient that overflows.

# Domain Errors

A zero divisor causes a panic, not an error value. A quotient that a Divide function
cannot hold in one word causes a panic. A Carry_In or a Borrow_In above one causes a
panic.
