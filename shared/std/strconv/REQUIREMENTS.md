
# Allocation

Every exported operation owns no heap storage. A writer fills caller storage and
returns written byte count. Too-small storage causes panic before any partial result
escapes.

# Boolean

Parse_Boolean reads twelve accepted spellings. Format_Boolean_Into writes `true` or
`false` into caller Buffer and returns Boolean_Count.

# Text To Integer

Parse_Unsigned_Integer and Parse_Integer read a Text in an Implied_Base and a Bit_Size.
An Implied_Base of zero takes the base from the prefix and permits an underscore
between digits. Parse_Decimal reads base ten at the machine integer width.

# Integer To Text

Format_Unsigned_Integer_Into and Format_Integer_Into write value in Base from two to
thirty-six, with lowercase letters for digits above nine. Format_Decimal_Into writes
base ten. Each form fills caller Buffer and returns its exact text count.

# Fixed Point

Parse_Fixed_Point reads decimal text into fixedpoint Number, which deterministic tier
holds in place of float. Format_Fixed_Point_Into writes Number with zero to six
fraction digits into caller Buffer and returns Fixed_Point_Text_Count.

# Domain Errors

Out-of-domain Base, Bit_Size, or size causes panic. Too-small caller Buffer causes panic
before partial output escapes. Error_Syntax reports invalid text. Error_Range reports
number that target width cannot hold.

# Quote Forms

Quote_Into writes double-quoted Go string literal. Quote_Rune_Into writes single-quoted
Go character literal. ASCII forms escape each character above ASCII range. Graphic
forms keep each graphic character. Each form fills caller Buffer and returns count.

# Backquote Form

Can_Backquote reports whether a Text stays unchanged inside backquotes. A control
character other than a tab, a backquote, a delete, a byte order mark, or invalid text
prevents this form.

# Unquote Forms

Unquote_Into reads single-quoted, double-quoted, or backquoted Go literal into caller
Buffer and returns decoded count. Quoted_Prefix returns literal at Text start.
Unquote_Character decodes one literal-body character and returns remainder.

# Printability

Is_Print reports whether a Character is printable, and Is_Graphic reports whether it is
graphic. The unicode package supplies both facts, thus a negative Character is neither.

# Size Limits

A Text holds at most TEXT_SIZE_MAXIMUM bytes, and a Buffer holds at most
BUFFER_SIZE_MAXIMUM bytes. Each written form has its own limit, which the input limit
and the longest escape give.
