
# Validate

Validate accepts one bounded well-formed document with matched elements, quoted attributes,
comments, processing instructions, directives, CDATA, UTF-8 text, and canonical entity references.

# Escape

Escape_Size reports exact XML text storage. Escape_Into applies the standard predefined and
numeric replacements into caller-owned output, including replacement of invalid UTF-8.

# Unescape

Unescape_Size validates predefined and numeric references before output access. Unescape_Into
writes decoded XML text into caller-owned output.

# Bounds

Document, text, output, nesting, counts, and positions follow formulas from the shared byte limit
and shortest complete element. Oversized slices panic; oversized escaped output returns status.

# Allocation

Every exported operation performs zero heap allocation with assertion tracking enabled. Document,
text, and output storage remain caller-owned.
