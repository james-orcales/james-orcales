
# Boolean

Parse_Boolean reads the twelve accepted spellings, and Format_Boolean writes `true` or
`false`. Append_Boolean writes the same text into a Buffer.

# Text To Integer

Parse_Unsigned_Integer and Parse_Integer read a Text in an Implied_Base and a Bit_Size.
An Implied_Base of zero takes the base from the prefix and permits an underscore
between digits. Parse_Decimal reads base ten at the machine integer width.

# Integer To Text

Format_Unsigned_Integer and Format_Integer write a value in a Base from two to
thirty-six, with the lowercase letters for the digits above nine. Format_Decimal writes
base ten. Each Append form writes the same text into a Buffer.

# Fixed Point

Parse_Fixed_Point reads decimal text into a fixedpoint Number, which is what the
deterministic tier holds in place of a float. Format_Fixed_Point and Append_Fixed_Point
write a Number with zero to six fraction digits.

# Domain Errors

An out-of-domain Base, Bit_Size, or size causes a panic, not an error value. The two
remaining errors are Error_Syntax for text that is not a number and Error_Range for a
number that the width cannot hold.

# Quote Forms

Quote returns a double-quoted Go string literal, and Quote_Rune returns a single-quoted
Go character literal. The ASCII form escapes each character above the ASCII range, and
the Graphic form keeps each character that is graphic.

# Backquote Form

Can_Backquote reports whether a Text stays unchanged inside backquotes. A control
character other than a tab, a backquote, a delete, a byte order mark, or invalid text
prevents this form.

# Unquote Forms

Unquote reads a single-quoted, double-quoted, or backquoted Go literal. Quoted_Prefix
returns the literal at the start of a Text. Unquote_Character decodes one character of
a literal body and returns the remainder.

# Printability

Is_Print reports whether a Character is printable, and Is_Graphic reports whether it is
graphic. The unicode package supplies both facts, thus a negative Character is neither.

# Size Limits

A Text holds at most TEXT_SIZE_MAXIMUM bytes, and a Buffer holds at most
BUFFER_SIZE_MAXIMUM bytes. Each written form has its own limit, which the input limit
and the longest escape give.
