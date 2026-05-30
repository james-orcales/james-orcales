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

This framework exposes composable properties. They compose along two axes: individual
properties combine into a coverage grid, and whole bundles of properties combine across types.

## Atoms

Every property is built from two atoms, `Always` and `Sometimes`. They look alike — each takes a
bool — but they assert opposite kinds of thing, and the difference is the whole design.

`Always(condition)` is a hard assertion: the condition must hold on every call, and a single false
observation panics.  Its coverage obligation is only that it be *reached*: an `Always` the suite
never exercises is reported as a gap.

`Sometimes(condition)` asserts nothing about any single call — it never panics. It is a claim about
the *run*: across the whole suite the condition must be observed both true and false. A `Sometimes`
seen only true means the suite never drove it false; that missing branch is a coverage gap reported
at the end. So `Always` catches a value that should never occur, and `Sometimes` catches a case the
tests forgot to cover — a panic versus a silent blind spot.

`Always` and `Sometimes` are the only true atoms; everything else is sugar over them. The runtime
special-cases a handful of derived forms purely to cut boilerplate such as `Distinct_Boundary` which 
generates two `Always` and two `Sometimes` for the price of one.

Every atom is inert alone. Constructing one records nothing and enforces nothing; it is only when
the atom is handed to a `Dot_Product` (directly, or by flowing through a bundle into one) that it is
enforced and its coverage tracked. An atom that never reaches a `Dot_Product` is an orphan — it
asserts nothing despite looking like it does — and is reported as such.

### Axes compose into a grid

A `Sometimes` is a two-outcome axis: the suite must witness it both true and false. An `Always`
is a guard, not an axis — it has one legal outcome, so it never widens the grid; it is simply
checked in every cell and panics the moment it fails. `Dot_Product` takes the cartesian product of
the axes and demands every cell be witnessed.

```go
invariant.Dot_Product(
    invariant.Always(p),    // guard: must hold in every cell below, never a grid coordinate
    invariant.Sometimes(q), // axis
    invariant.Sometimes(r), // axis
)
```

The two `Sometimes` axes generate a 2×2 grid. Every cell must be reached by the suite — and in
each, `Always(p)` must hold or the run panics:

```
(q=0, r=0)
(q=0, r=1)
(q=1, r=0)
(q=1, r=1)
```

Two axes → 2² cells, three → 2³, n → 2ⁿ. The grid is the whole point and also the whole danger:
it grows exponentially. `Impossible` is the pressure valve — it deletes cells that cannot occur, so
the suite is never asked to witness the impossible:

```go
invariant.Impossible(invariant.Event_True(q), invariant.Event_True(r)) // carves out cell (q=1, r=1)
```

## Bundles compose across types

A type's properties live in a `*_Invariants` bundle — a `[]invariant.Dot_Element` returned by a
function named for the type. A composite type's bundle is built from its components' bundles laid
end to end, plus the cross-field properties no component can state on its own.

```go
// A Token is the lexer's atom: never empty, never edge-padded with whitespace,
// and underscores show up only sometimes.
type Token string

func Token_Invariants(token Token) []invariant.Dot_Element {
    return []invariant.Dot_Element{
        invariant.Always(token != ""),
        invariant.Never_Has_Edge_Whitespace(string(token)),
        invariant.Sometimes(strings.Contains(string(token), "_")),
    }
}

// A Span is a half-open byte range into the source. A zero-width span (Lo == Hi)
// is the EOF marker, so it must show up sometimes but not always.
type Span struct{ Lo, Hi int }

func Span_Invariants(span Span) []invariant.Dot_Element {
    return []invariant.Dot_Element{
        invariant.Always(span.Lo <= span.Hi),
        invariant.Sometimes(span.Lo == span.Hi),
    }
}

// A Lexeme is a Token plus where it was cut from. Its bundle is its parts'
// bundles concatenated — the reuse that lets invariants compose with the types —
// plus the one property that relates the parts and so belongs to neither.
type Lexeme struct {
    Token Token
    Span  Span
}

func Lexeme_Invariants(lexeme Lexeme) (dot_elements []invariant.Dot_Element) {
    dot_elements = append(dot_elements, Token_Invariants(lexeme.Token)...)
    dot_elements = append(dot_elements, Span_Invariants(lexeme.Span)...)
    return append(dot_elements,
        invariant.Always(lexeme.Span.Hi-lexeme.Span.Lo == len(lexeme.Token)))
}
```

At a callsite touching more than one value, `Cross_Product` composes their bundles. It is exactly
`Dot_Product` over whole bundles instead of bare axes — the same full joint grid, so the suite must
witness every interaction between the lexeme's axes and the source's:

```go
// At the lexer boundary, the emitted lexeme and the source it was cut from must
// jointly hold. String_Invariants is the framework's own bundle for a string.
func emit(lexeme Lexeme, source string) {
    invariant.Cross_Product(
        Lexeme_Invariants(lexeme),
        invariant.String_Invariants(source),
    )
}
```

Because the grid is joint, `Cross_Product` inherits the same exponential blow-up as `Dot_Product`,
and the same valve: `Impossible` carves the cells the composition makes unreachable.

## NOTES

I'm stashing my notes here to be cleaned for the final spec. In invariant v2, i tried to only allow
inline assertions at callsites. Combine this with capped function lengths and line lengths, it
indirectly caps the cardinality of a given callsite, steering AI to simplify the design. However,
as you go up the stack (e.g. func Main(...)) combined with a sufficiently large system, you must
have some way to compose the invariants of large subsystems. Otherwise, it's impossible to represent
the system properties.
