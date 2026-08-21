
# Configuration

New_Configuration validates minimum width, tab width, padding, pad character, and flags.
Tab padding always forces left alignment. Configuration owns no output or process driver.

# Source

Source_Validate changes hostile borrowed bytes into Source only inside SOURCE_SIZE_MAXIMUM.

# Format

Format_Into aligns tab-terminated cells into caller output. Caller supplies Workspace. The
operation returns exact required size for short output and leaves that output unchanged.

# Escapes

Escape segments preserve embedded tabs and line breaks. Strip_Escape removes only their boundary
bytes. Filter_HTML gives tags zero width and entities one character width.

# Columns

Horizontal and vertical tabs form hard and soft cells. Newline ends a line. Form feed also ends
the current column section. Alignment, empty-column removal, tab indentation, and debug markers
match text/tabwriter.

# Bounds

Source and output hold at most SOURCE_SIZE_MAXIMUM and OUTPUT_SIZE_MAXIMUM bytes. Configuration
scalars and workspace indexes have formula-derived limits. Oversized result, overlap, invalid
configuration, and absent workspace return status before output mutation.

# Allocation

Every public runtime operation performs zero heap allocation with assertion tracking enabled.
Caller owns source, output, configuration, and workspace.
