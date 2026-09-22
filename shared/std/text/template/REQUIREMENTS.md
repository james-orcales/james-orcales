
# Compile

Compile validates bounded borrowed source and stores complete text/template syntax in caller
Syntax_Workspace. Source validation, delimiter policy, complete grammar, and scalar diagnostics
remain pure inside the template library; no composition package owns parser state.

# Values

Value is a reflection-free closed union. Nil, Boolean, signed integer, unsigned integer, binary64,
complex binary64, text, sequence, object, map, and function values borrow caller storage. Object
fields and map entries stay caller-owned and bounded.

# Execute

Execute_Into applies standard text/template control, pipelines, variables, functions, builtins,
field selection, indexing, slicing, comparison, and template invocation. Caller workspace owns
frames, variables, arguments, literal decoding, ordering, and staged output.

# Bounds

Source, delimiters, syntax nodes, templates, nesting, output, values, functions, variables,
arguments, traversal, iteration, and template depth have formula-derived limits. Malicious or
mutated input returns status before unsafe access or capacity writes. Explicit budget stops work.

# Output

Execution stages one complete bounded result before copying to destination. Failure leaves caller
output unchanged. Diagnostics contain status and source position, never allocated error text.

# Allocation

Every public operation performs zero heap allocation with assertion tracking enabled. Source,
configuration, syntax, values, functions, execution workspace, diagnostics, and output remain
caller-owned.
