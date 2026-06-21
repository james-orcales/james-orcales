
# Nested Struct Flattens To Prefixed Keys

A nested struct field is flattened into the parent object, each leaf keyed by the field
path joined with an underscore, so addr.city becomes addr_city.

# Scalar Fields Marshal To Their JSON Forms

String, integer, boolean, and float fields marshal to their JSON scalar forms.

# Scalar Slices Pass Through

A slice of scalars is emitted as a flat JSON array under its own key, not exploded into
indexed keys.

# Json Tag Names The Leaf

A field's json tag supplies its key segment when present; otherwise the Go field name is
used verbatim.

# Embedded Struct Adds No Segment

An anonymous embedded struct's fields are promoted to the parent object with no path
segment of their own, matching encoding/json.

# Nil Pointer Emits Null

A nil pointer is never dropped: a scalar pointer emits null, and a nested-struct pointer
emits null for each of its leaf keys, so the key set is the same whether or not it is nil.

# Marshaler Leaf Is Delegated

A type implementing json.Marshaler, such as time.Time, is treated as a leaf and encoded
by encoding/json rather than recursed into.

# Top Level Array Marshals

A slice or array marshals to a top-level JSON array of flat objects — the array-of-objects
shape the resource permits.

# Non Flat Field Is Rejected

A field that cannot stay flat, such as a map or a slice of structs, is a marshal error
rather than silent nested output.

# Colliding Keys Are Rejected

Two fields that produce the same flat key are a marshal error rather than duplicate keys,
so flat output never silently drops a field.

# Stream Write Encodes To Writer

Marshal_Write encodes the value and writes the flat JSON to an io.Writer.
