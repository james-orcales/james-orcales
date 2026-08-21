
# Scan

Scan reads the token at the cursor and advances the cursor past it. The cursor holds the source,
the byte offset, and the previous kind, thus a scan allocates nothing and the caller owns storage.
An exhausted source scans to KIND_END_OF_FILE forever, so a caller loop needs no other stop test.

# Whitespace

Space, horizontal tab, carriage return, and line feed separate tokens and scan as no token. A
source of only whitespace scans to KIND_END_OF_FILE.

# Comments

A line comment runs to the line feed and a general comment runs to its closing delimiter. Both
scan as KIND_COMMENT and neither replaces the previous kind that semicolon insertion reads. A
general comment holding a line feed closes its line.

# Identifiers

An identifier opens with an ASCII letter or an underscore and continues with letters, digits, and
underscores. A byte at or above 128 opens no identifier, because this repository writes every
declared name in ASCII.

# Keywords

Each of the twenty-five Go keywords scans as its own kind, never as KIND_IDENTIFIER.

# Number Literals

Decimal, binary, octal, and hexadecimal integers scan as KIND_INTEGER. A fraction or an exponent
makes KIND_FLOAT and a trailing i makes KIND_IMAGINARY. An underscore separates digits.

# Character Literals

A character literal holds one character or one escape between apostrophes and scans as
KIND_CHARACTER. A backslash quotes the byte after it. An unclosed literal scans as KIND_ILLEGAL.

# String Literals

An interpreted literal stands between quotation marks and holds escapes; a raw literal stands
between back quotes, holds no escape, and may hold line feeds. Both scan as KIND_STRING, and an
unclosed literal scans as KIND_ILLEGAL.

# Operators

Every Go operator and delimiter scans as its own kind. The scan takes the longest match, thus the
three bytes of an and-not assignment scan as one token and never as an and then an exclusive-or
assignment.

# Semicolon Insertion

A line feed after an identifier, a literal, break, continue, fallthrough, return, an increment, a
decrement, or a closing bracket scans as KIND_SEMICOLON of size zero. The end of the source closes
the final line.

# Illegal Input

A byte that opens no Go token scans as KIND_ILLEGAL of size one and the cursor still advances,
thus a hostile source never stalls a scan.

# Bounds

A source holds at most 1,048,576 bytes and every offset and size stays inside that bound. An
oversized source panics before the scanner reads one byte.

# Allocation

Scan and Token_Text perform zero heap allocation. The caller owns the source bytes and the
scanner, thus this package holds no storage of its own.
