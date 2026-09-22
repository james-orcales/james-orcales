
# Parse

Parse decodes `harness.json`, which holds the shape of `lint.json`, into caller stores.

### Shape

One JSON object of seven keys, each required once: `shared_component` a string, `word_replacements`
an object of string arrays, and `instrumentation_packages`,
`ignore`, `opt_out_assertion_mandate_packages`, `opt_out_recursion_ban`, `opt_out_interface_ban`.

### Values

Each string decodes into one value that names its key; a replacement also names its word. An
escape decodes as JSON states it, a surrogate pair included. `shared_component` and
`word_replacements` hold at least one byte and one word.

### Globs

Each entry of the five path lists is a glob of `*` and `**`, as lint states one. An entry of spaces
alone, a `!` that negates nothing, a `?` or `[`, a segment ending in a lone backslash, or a list of
negations alone is a fault at the entry, naming its key, in the order lint checks the lists.

### Exemptions

An instrumentation package is exempt from the assertion mandate already, thus an
`opt_out_assertion_mandate_packages` entry equal to an `instrumentation_packages` entry is a fault.

### Order

Past a Parse, Entry_Next yields each entry of the five path lists once, in the order lint checks
them: ignore, instrumentation, assertion opt-outs, recursion opt-outs, interface opt-outs.

### Faults

Parse states the first fault at the byte it stands on: a byte the shape does not admit, an
unfinished value, trailing data, a control byte, a bad escape, invalid UTF-8, an unknown or
repeated key, or an absent key or empty value, at the end of the object.

### Bounds

A source past `SOURCE_SIZE_MAXIMUM`, or a string past the value table, is a fault. Parse writes no
byte past its stores, since the file is untrusted.
