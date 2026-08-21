
# Allocation

Marshal_Into and Marshal_Write perform zero heap allocation with assertion tracking enabled.
Callers provide both exact reflection state and complete bounded output storage. Values expose
only primitive leaves because standard marshaler methods return allocation-owning byte slices.

# Nested Struct Flattens To Prefixed Keys

A nested struct field is flattened into the parent object, each leaf keyed by the field
path joined with an underscore, so addr.city becomes addr_city.

# Scalar Fields Marshal To Their JSON Forms

String, integer, and boolean fields marshal to their JSON scalar forms. Float fields are rejected
because the deterministic shared tier admits no floating-point representation.

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

# Marshaler Leaf Is Rejected

A method-bearing struct is not a primitive leaf. Marshal_Into rejects it instead of invoking an
allocation-owning MarshalJSON or MarshalText method.

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

Marshal_Write encodes the value and writes the flat JSON to a Writer.
