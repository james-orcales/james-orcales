# invariant

I once tried to create a programming language. It was essentially the Go runtime with Zig syntax.
The distinguishing feature being first-class property testing. This was derived from my usage
pattern of assertions. Every type had its own function asserting its properties which I then
littered across every function boundary. Ideally, this is enforced by the type system like so:

```
const legal_age = 18

type Employee struct {
    Name string
    Age int
} where {
    Name != ""
    Age >= legal_age
}
```

There's a bunch of useful things we can do with this information beyond reducing boilerplate, such
as autogeneration of property testing but we'll leave it at that.

This framework exposes composable properties: eager guards that must hold on every call, and
observations the suite must witness both ways. A type's properties travel with it and compose
across types.

## Atoms

`Always(condition, message)` is a hard assertion: the condition must hold on every call, and a
single false observation panics at the callsite. Its coverage obligation is only that it be
*reached*: an `Always` the suite never exercises is reported as a gap.

`Sometimes(identifier, condition, message)` is a bare observation. It is a claim about the *run*:
across the whole suite the condition must be observed both true and false. It never panics — a
condition seen only true means the suite never drove it false, and that missing branch is a
coverage gap. The identifier names the boundary observing the property, so one property observed
at two boundaries earns two entries. `Always` catches a value that should never occur, while
`Sometimes` catches a case the tests forgot to cover.

`Range(identifier, value, minimum, maximum, excluded...)` and `Enum(identifier, value,
members...)` are the typed eager guards, generic over every defined integer width except
`uintptr`. They panic like `Always`, naming the identifier, the value, and the violated bound or
the member set; trailing `Range` exclusions are holes inside the interval the value must also
avoid.

The sugar tier adds the `*_Invariants` presets purely to cut boilerplate — each expanding into
`Sometimes` witnesses over a primitive value's boundary cases.

## Composition across types

A type's properties live in a `_Invariants` function named for the type, taking the value and a
trailing identifier naming the boundary that demands them. The type owns its properties; a
boundary demands them with one call.

```go
// A Token is the lexer's atom: never empty, never edge-padded with whitespace,
// and underscores show up only sometimes.
type Token string

func Token_Invariants(token Token, identifier string) {
    invariant.Always(token != "", "A token is never empty.")
    invariant.Always(strings.TrimSpace(string(token)) == string(token),
        "A token has no edge whitespace.")
    invariant.Sometimes(identifier, strings.Contains(string(token), "_"),
        "A token has an underscore.")
}

// A Span is a half-open byte range into the source. A zero-width span (Lo == Hi)
// is the EOF marker, so it must show up sometimes but not always.
type Span struct{ Lo, Hi int }

func Span_Invariants(span Span, identifier string) {
    invariant.Always(span.Lo <= span.Hi, "A span is ordered.")
    invariant.Sometimes(identifier, span.Lo == span.Hi, "A span is zero-width.")
}
```

A composite composes by **calling** its parts' `_Invariants`, plus its own assertions for the
cross-field properties no part can state alone:

```go
type Lexeme struct {
    Token Token
    Span  Span
}

func Lexeme_Invariants(lexeme Lexeme, identifier string) {
    invariant.Always(lexeme.Span.Hi-lexeme.Span.Lo == len(lexeme.Token),
        "A lexeme's span matches its token.")
    invariant.Sometimes(identifier, lexeme.Span.Lo == lexeme.Span.Hi,
        "A lexeme is the EOF marker.")
    Token_Invariants(lexeme.Token, "Lexeme.Token")
    Span_Invariants(lexeme.Span, "Lexeme.Span")
}
```

Declare your own `_Invariants` only for a custom, defined type. The presets are the framework's
bundles for the primitive types; user code never re-declares one. To cover a primitive, call a
preset, state its assertions inline, or wrap it in a custom type. A `_Invariants` body must be
straight-line — a branching or looping statement fails registration, since it would make the
properties the body emits depend on runtime values the static scan cannot read.

## Static registration

Before the suite runs, a source scan discovers every bare `Always` and seeds its literal message;
an unreached `Always` fails the run at the end. Registration for the bare `Sometimes`, `Range`,
and `Enum` writers is a later iteration — until then they enforce (and `Sometimes` records into
already-seeded entries) without seeding obligations of their own.

## NOTES

I'm stashing my notes here to be cleaned for the final spec. In invariant v2, i tried to only allow
inline assertions at callsites. Combine this with capped function lengths and line lengths, it
indirectly caps the cardinality of a given callsite, steering AI to simplify the design. However,
as you go up the stack (e.g. func Main(...)) combined with a sufficiently large system, you must
have some way to compose the invariants of large subsystems. Otherwise, it's impossible to represent
the system properties. Self-emitting `_Invariants` are that composition mechanism: each type's grid
is authored once and travels by a one-line call, so a boundary touching many types stays within the
line cap while still demanding every type's coverage.
