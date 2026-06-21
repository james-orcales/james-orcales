# flatjson

A one-way **flat-JSON encoder** for arbitrary Go structs: `encoding/json`'s output, flattened
so a nested struct field becomes a prefixed key (`Addr.City` → `addr_city`).

```go
type Place struct {
    Name string  `json:"name"`
    Addr Address `json:"addr"`
}
data, _ := flatjson.Marshal(Place{Name: "cafe", Addr: Address{City: "nyc", Zip: 10001}})
// {"name":"cafe","addr_city":"nyc","addr_zip":10001}
```

It fills the gap between `shared/jlog` (flat output, but a fixed field-list API, not arbitrary
structs) and `encoding/json` (arbitrary structs, but nested output). The shape rationale — flat
objects, or a top-level array of them, queried with `jq` and indexed by log backends — is the
kellybrazil resource (`documentation/resources/kellybrazil.*`), shared with jlog.

## Why it is encode-only

There is no `Unmarshal`. That is deliberate, and the reasons are not in the resource — it is
about *output design*, not round-tripping — so they are recorded here. Flattening **cannot be
inverted without loss**:

1. **The bytes are not self-describing.** `addr_city` could be a nested `addr.city`, or a literal
   field tagged `addr_city`. The flat JSON alone cannot tell, so decoding would need the Go type —
   and even with it:

2. **A nil sub-object cannot survive.** A nil `Addr *Address` has no flat key to carry "this whole
   sub-object is absent" — only `addr_city`, `addr_zip` exist. Any decoder re-allocates and fills,
   so `nil → &Address{}`. The boundary is exactly what flattening throws away; nothing recovers it.

3. **Three goals cannot hold at once: flat + stable schema + lossless round-trip.** Nested JSON
   gets all three — but only by *not being flat* (it writes `"addr":null`). Flat output must give
   one up.

4. **The resource's stable-schema rule is itself anti-round-trip.** It says emit `null` for a
   blank and never drop a key ("the user can always find an attribute"). But omitting a nil pointer
   was the only encoding that round-tripped it; honoring the resource means a nil pointer emits
   `null`, which decodes back to a zero value, not nil.

5. **Key collisions are silent data loss.** Two field paths can flatten to one key; on decode that
   is a vanished field. Marshal rejects it outright — a collision is a marshal error, not
   duplicate-key output.

6. **Round-trip already has a better tool.** Nested `encoding/json` is self-describing, lossless by
   construction, and less code. Flattening for round-trip is strictly worse, so flatjson does not
   compete there.

## When to use it

- **Use it for** flat output: CLI / JSON Lines, `jq` pipelines, log and observability sinks.
- **For data you read back** (config, IPC, storage), use nested `encoding/json`.

See `SPECIFICATION.md` for the per-behavior contract.
