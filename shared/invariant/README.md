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

This framework exposes composable properties. A property is a coverage grid of axes; a type's
properties travel with it and compose across types — each type's grid kept self-contained.

## Atoms

Every property is built from two atoms, `Always` and `Sometimes`. They look alike — each takes a
bool and a message — but they assert opposite kinds of thing, and the difference is the whole
design.

`Always(condition, message)` is a hard assertion: the condition must hold on every call, and a
single false observation panics. Its coverage obligation is only that it be *reached*: an `Always`
the suite never exercises is reported as a gap.

`Sometimes(condition, message)` asserts nothing about any single call — it never panics. It is a
claim about the *run*: across the whole suite the condition must be observed both true and false. A
`Sometimes` seen only true means the suite never drove it false; that missing branch is a coverage
gap reported at the end. So `Always` catches a value that should never occur, and `Sometimes`
catches a case the tests forgot to cover — a panic versus a silent blind spot.

`Always` and `Sometimes` are the only true atoms; everything else is sugar over them. The sugar tier
adds the `*_Invariants` presets purely to cut boilerplate — each expanding into `Always` /
`Sometimes` checks over a value.

A `Sometimes` is inert alone: constructing one records nothing and enforces nothing; it is only when
it is handed to a `Dot_Product` that it is enforced and its coverage tracked. `Always` is the
exception — it is eager, enforcing the moment it is called, and is never handed to a `Dot_Product`.

### Axes compose into a grid

A `Sometimes` is a two-outcome axis: the suite must witness it both true and false. An `Always`
is a guard, not an axis — it has one legal outcome, so it never widens the grid; it is eager and
enforced at its own call, outside the `Dot_Product`. `Dot_Product` takes the cartesian product of
the axes and demands every cell be witnessed. Its first argument is a `Namespace` — a literal that
identifies the grid and prefixes every axis's message into a coverage key:

```go
invariant.Always(p, "p holds") // eager guard: enforced right here, on every call
invariant.Dot_Product("widget",
    invariant.Sometimes(q, "q"), // axis
    invariant.Sometimes(r, "r"), // axis
)
```

The two `Sometimes` axes generate a 2×2 grid. Every cell must be reached by the suite, while
`Always(p)` is enforced eagerly on every call, independent of the grid:

```
(q=0, r=0)
(q=0, r=1)
(q=1, r=0)
(q=1, r=1)
```

Two axes → 2² cells, three → 2³, n → 2ⁿ. The grid is the whole point and also the whole danger:
it grows exponentially. `Impossible` is the pressure valve — it deletes cells that cannot occur, so
the suite is never asked to witness the impossible. It names sibling axes by their message and globs
over the axes it does not name:

```go
invariant.Impossible(invariant.Event_True("q"), invariant.Event_True("r")) // carves cell (q=1, r=1)
```

### Imply gates an axis

Some axes are meaningful only under a precondition — a field is checked only when a pointer is
non-nil. `Imply(prerequisite, condition, message)` is a gated `Sometimes`: it is recorded only on a
call where the prerequisite holds, don't-care otherwise, and it joins no tuple of the grid (the
message-less prerequisite is no axis to cross). The condition is evaluated eagerly, so one safe only
under the prerequisite must still self-guard:

```go
invariant.Imply(p != nil, p != nil && p.Ready, "ready when present")
```

## Composition across types

A type's properties live in a `_Invariants` function named for the type, taking the value and a
`Namespace`. It **self-emits** its own `Dot_Product` under that namespace — the type owns its grid,
and the caller supplies the per-callsite identity as an inline literal.

```go
// A Token is the lexer's atom: never empty, never edge-padded with whitespace,
// and underscores show up only sometimes.
type Token string

func Token_Invariants(token Token, namespace invariant.Namespace) {
    // Eager guards fire right here; only the Sometimes axes form the grid.
    invariant.Always(token != "", "non-empty")
    invariant.Always(strings.TrimSpace(string(token)) == string(token), "no edge whitespace")
    invariant.Dot_Product(namespace,
        invariant.Sometimes(strings.Contains(string(token), "_"), "has underscore"),
    )
}

// A Span is a half-open byte range into the source. A zero-width span (Lo == Hi)
// is the EOF marker, so it must show up sometimes but not always.
type Span struct{ Lo, Hi int }

func Span_Invariants(span Span, namespace invariant.Namespace) {
    invariant.Always(span.Lo <= span.Hi, "ordered")
    invariant.Dot_Product(namespace,
        invariant.Sometimes(span.Lo == span.Hi, "zero width"),
    )
}
```

A composite composes by **calling** its parts' `_Invariants` with literal sub-namespaces, plus its
own axes for the cross-field properties no part can state alone. Each part registers its **own,
self-contained grid** — there is no joint cross-product across the parts:

```go
type Lexeme struct {
    Token Token
    Span  Span
}

func Lexeme_Invariants(lexeme Lexeme, namespace invariant.Namespace) {
    // The cross-field property relates the parts — an eager guard, not an axis.
    invariant.Always(lexeme.Span.Hi-lexeme.Span.Lo == len(lexeme.Token), "span matches token")
    // The composite's own axis: a coverage case only it can state.
    invariant.Dot_Product(namespace,
        invariant.Sometimes(lexeme.Span.Lo == lexeme.Span.Hi, "eof lexeme"),
    )
    // Composition: each part self-emits its grid under its own literal namespace.
    Token_Invariants(lexeme.Token, "Lexeme.Token")
    Span_Invariants(lexeme.Span, "Lexeme.Span")
}
```

At a boundary touching more than one value, call each one's `_Invariants` under its own namespace.
There is no `Cross_Product` and no joint grid: keeping the grids marginal (the same model
Antithesis uses) is what removes the combinatorial blow-up — `n` types contribute `n` separate
grids, not one of size `2^(sum of their axes)`:

```go
// At the lexer boundary, the emitted lexeme and the source it was cut from each
// register their own grid. String_Invariants is the framework's own preset for a string.
func emit(lexeme Lexeme, source string) {
    Lexeme_Invariants(lexeme, "lex.emit.lexeme")
    invariant.String_Invariants(source, "lex.emit.source")
}
```

Each callsite namespace is the grid's identity: two callsites with distinct namespaces register
independent grids that never mask each other's gaps, and reusing one namespace is a duplicate that
fails registration. A `_Invariants` body must be straight-line — a branching or looping statement
fails registration, since it would make the axes the body self-emits depend on runtime values the
static scan cannot read.

## Static registration

Before the suite runs, a source scan walks every `_Invariants(v, "literal")` callsite, resolves the
function, and registers the grid its body self-emits — keyed by the literal namespace. A
never-witnessed cell, an unobserved `Sometimes` branch, or an unreached `Always` is reported at the
end and fails the run. The runtime and the static side rendezvous on the same key (namespace
prefixed onto each axis's own message), so what the scan demands is exactly what the run credits.

## NOTES

I'm stashing my notes here to be cleaned for the final spec. In invariant v2, i tried to only allow
inline assertions at callsites. Combine this with capped function lengths and line lengths, it
indirectly caps the cardinality of a given callsite, steering AI to simplify the design. However,
as you go up the stack (e.g. func Main(...)) combined with a sufficiently large system, you must
have some way to compose the invariants of large subsystems. Otherwise, it's impossible to represent
the system properties. Self-emitting `_Invariants` are that composition mechanism: each type's grid
is authored once and travels by a one-line call, so a boundary touching many types stays within the
line cap while still demanding every type's coverage.
