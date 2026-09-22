
# Parse

Parse decodes `harness.json`, which holds the shape of `lint.json`, through
`shared/encoding/json` into caller stores.

### Shape

One object of seven keys, each once: `shared_component` a string, `word_replacements` an object of
string arrays, and string arrays `instrumentation_packages`, `ignore`, `opt_out_recursion_ban`,
`opt_out_assertion_mandate_packages`, `opt_out_interface_ban`; another type is a keyed fault.

### Values

Each string decodes into one value that names its key, as `shared/encoding/json` decodes it; a
replacement also names its word. `shared_component` and `word_replacements` hold at least one byte
and one word.

### Globs

Each entry of the five path lists is a glob of `*` and `**`, as lint states one. An entry of spaces
alone, a `!` that negates nothing, a `?` or `[`, a segment ending in a lone backslash, or a list of
negations alone is a fault naming its key and the entry, in the order lint checks the lists.

### Exemptions

An instrumentation package is exempt from the assertion mandate already, thus an
`opt_out_assertion_mandate_packages` entry equal to an `instrumentation_packages` entry is a fault.

### Order

Past a Parse, Entry_Next yields each entry of the five path lists once, in the order lint checks
them: ignore, instrumentation, assertion opt-outs, recursion opt-outs, interface opt-outs.

### Faults

Parse states the first fault: a source that is no JSON value, at the byte `shared/encoding/json`
states; a root other than an object; an unknown key, naming it; a repeated key; then an absent key
or an empty value.

### Bounds

A source past `SOURCE_SIZE_MAXIMUM`, or a string past the value table, is a fault. Parse writes no
byte past its stores, since the file is untrusted.
