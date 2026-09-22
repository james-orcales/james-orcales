
# Source

Source_Validate changes hostile borrowed text into Source only when byte count stays inside
SOURCE_SIZE_MAXIMUM. Scanner holds Source view; no reader, file, stderr, or process loop hides
behind scan call.

# Tokens

Scanner_Scan recognizes standard identifiers, numeric literals, character literals, string
literals, raw strings, and comments under caller Mode. Unselected forms remain individual
characters. GO_TOKENS skips comments.

# Characters

Scanner_Peek returns next decoded character without consuming public cursor. Scanner_Next consumes
it. Initial byte-order mark is ignored. Invalid UTF-8 and NUL produce typed Error reports.

# Diagnostics

Caller Error_Function owns diagnostic policy. Error carries Error_Code, Character, and Position;
package writes no stderr and creates no owned message. Error_Count remains bounded by source size.

# Positions

Position uses zero-based byte Offset and one-based Line and character Column. Scanner_Position
names boundary after prior token or character. Position_Append_Into writes standard filename,
line, and column form into caller storage.

# Text

Scanner_Token_Text returns borrowed Source substring. Token_Append_Into writes predefined token
label or quoted character into caller storage. Too-small storage reports exact required count and
does not mutate destination.

# Bounds

Source and Filename hold at most SOURCE_SIZE_MAXIMUM and FILENAME_SIZE_MAXIMUM bytes. Every
coordinate, mode, whitespace mask, error count, output, and written count has formula-derived
package boundary. Hostile oversized source is refused before scanner mutation.

# Allocation

Every public runtime operation performs zero heap allocation with assertion tracking enabled.
Caller owns source, scanner state, callbacks, and formatted output.
