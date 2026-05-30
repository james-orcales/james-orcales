
# Render

Render turns Markdown source into the laid out, styled text of a PDF body.

### Headings

A line opening with one to six number signs renders as a heading sized by its
level, with a blank line above and, for the top two levels, a rule beneath.

### Paragraphs

Consecutive prose lines join into one paragraph wrapped to the text column width.

### Emphasis

A bold or italic span switches its run to the matching weight and slant.

### Code

An inline span or a fenced block renders white on a dark gray panel in a fixed
width font, sized so a fenced line of a hundred characters fits the column; an
inline span too wide for its column breaks across lines at character boundaries.

### Lists

Ordered and unordered items render with a marker and a hanging indent.

### Quote

A block quote sits in a padded light gray panel with a soft gray left bar and
gray text.

### Link

A link renders its label in blue underlined text and carries its target in a
clickable annotation.

### Rule

A thematic break renders as a horizontal line across the column.

### Tables

A pipe table renders GitHub style — a gray cell grid, a bold header, shaded
alternate rows — with cells that wrap and render inline markdown.

# Pages

Pages collects the laid out body, opening another page when the cursor overflows.

### Single

A body shorter than one page produces exactly one page.

### Overflow

A body taller than one page continues onto further pages.

# Document

Document frames the rendered pages as one complete PDF file.

### Header

The output opens with a PDF version header and a binary marker.

### Trailer

The output carries a cross reference table and ends with the end of file marker.

# Main

Main reads the injected Markdown and writes the PDF to the injected output.

### Output

A run writes a valid PDF to the output sink and returns a zero status code.
