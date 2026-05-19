# Assertion Enforcement

This spec describes the assertion-enforcement analyzer in `lint/`. It targets
`golang_snacks/invariant/v2`. The keywords MUST, MUST NOT, SHOULD, MAY are
used as defined in RFC 2119.

The analyzer is a sub-analyzer of the `lint/` binary. Diagnostics are errors;
CI fails on any violation. There is no opt-out. There is no auto-fix.

---

# 1. Vocabulary

- **Predicate** — the first argument to `Always` / `Sometimes` / `Boundary`.
- **Prologue** — the contiguous run of statements at the top of a function
  body before any non-assertion side effect.
- **Requirement** — one assertion call the developer MUST write. `Boundary`
  counts as one requirement; `Always(x != 0)` counts as another.
- **Unit** — a function prologue, a single declaration statement, or a
  return-assertion defer body. Cross_Product is scoped per unit.
- **Axis** / **Bucket** — defined in `golang_snacks/invariant/v2`
  (`Cross_Axis`, `Cross_Bucket`).
- **Implication** — the body of an `if X != nil { ... }` block, where the
  condition is exactly an `X != nil` comparison and `X` is an ident-chain.
  Assertions inside an Implication are credited under the guarantee that `X`
  is non-nil.
- **Cross_Product** — `invariant.Cross_Product(...)`. The container that
  carries axis-calls when a unit has ≥2 requirements.
- **Nillable ancestry** — the property of a field path that walks through
  at least one nillable segment (`*T`, slice, map, channel, interface). The
  property gates whether the path's leaf assertions may live inside an
  Implication.

---

# 2. When

## 2.1 What gets enforced

The analyzer enforces assertions on:

- All function parameters, including receivers.
- All return values.
- All variable declarations and re-assignments whose right-hand side is a
  function call. `:=` and `=` are equivalent for this purpose.

A "function call" means an `ast.CallExpr` of a user-defined function or
method. The following look like calls in Go syntax but DO NOT trigger the
declaration rule:

```go
v, ok := x.(T)        // type assertion — out of scope
v, ok := <-ch         // channel receive — out of scope
v, ok := m[k]         // map index — out of scope
p := new(T)           // builtin — out of scope
s := make([]T, n)     // builtin — out of scope
p := &x               // address-of — out of scope (no call)
```

Method calls DO count:

```go
out := s.Stamp()                 // in scope; out gets per-type assertion
```

## 2.2 Exemptions

- Files matching `*_test.go`.
- Files whose header matches the generated-code marker (`// Code generated
  ... DO NOT EDIT`).
- Blank-identifier parameters (`_`). They are skipped entirely and MUST NOT
  contribute to the requirement count.
- Calls in statement position whose return value is discarded (e.g.
  `f()` where `f` returns a value but the result is not bound).
- Stdlib types and other foreign-package types: no field recursion, no
  per-type rules. (See §4.4 for the foreign-field boundary.)

## 2.3 Banned constructs

- `init()` functions.
- Unnamed returns. A function with returns MUST name them so the
  return-assertion defer body can reference them.

```go
// VIOLATION — unnamed returns
func parse(input []byte) (Result, error) { ... }

// CORRECT
func parse(input []byte) (result Result, err error) { ... }
```

---

# 3. Where

## 3.1 Function-body layout

Every function body MUST be laid out as:

1. The return-assertion `defer`, on line one. It MUST precede every other
   statement, including any cleanup defer.
2. The parameter assertions (bare calls when one requirement, one
   `Cross_Product` when ≥2 requirements).
3. The function body proper.

```go
func handle(input Request_Input) (out Response, err error) {
    defer Cross_Product(                                    // §3.1.1
        Always(out.Status != 0, "Out status is set"),
        Sometimes(err == nil, "Returned error is sometimes nil"),
    )
    Cross_Product(                                          // §3.1.2
        Boundary(&Boundary_Input[int]{
            X: input.Retry, Lo: 0, Hi: 8,
            Message: "Retry stays within budget",
        }),
        Always(input.Retry != 0, "Retry is set"),
        Always(input.Host != "", "Host is non-empty"),
    )

    // §3.1.3 — function body
    conn, err := dial(input.Host)
    Always(err == nil, "Dial returned a nil error")
    // ...
}
```

## 3.2 Multi-defer ordering

The return-assertion defer MUST be the first statement of the function. Any
cleanup defers register after the assertion defer registers, which means
cleanup runs FIRST at unwind time (LIFO). This is intentional: the
assertion defer reads named returns after cleanup has finalized them.

```go
func read_config(input File_Input) (cfg Config, err error) {
    defer Cross_Product(
        Always(err == nil, "Returned error is nil"),
        Sometimes(cfg.Strict, "Strict mode is exercised both ways"),
    )
    Cross_Product(
        Always(input.Path != "", "Path is non-empty"),
        Always(input.FS != nil, "Filesystem is injected"),
    )

    f, err := input.FS.Open(input.Path)
    Always(err == nil, "Open returned a nil error")
    Always(f != nil, "Open returned a non-nil handle")
    defer f.Close()
    // ...
}
```

## 3.3 Declaration ordering

For declarations from a function call — in any block scope (top level,
conditional body, loop body, closure body) — the very next statement MUST
be the assertion. Multiple declarations MUST NOT be batched; each
declaration gets its own immediate assertion.

```go
// CORRECT
conn, err := dial(host)
Always(err == nil, "Dial returned a nil error")
session, err := negotiate(conn)
Always(err == nil, "Negotiate returned a nil error")

// VIOLATION — batched assertion between two declarations
conn, err := dial(host)
session, err := negotiate(conn)
Cross_Product(
    Always(err == nil, "Negotiate returned a nil error"),
    Always(conn != nil, "Connection is non-nil"),
)
```

## 3.4 Re-assignment

Plain re-assignment from a function call (`x = f()`) is subject to the same
rule as `:=`:

```go
conn, err := dial(primary)
Always(err == nil, "Primary dial returned a nil error")
if conn == nil {
    conn, err = dial(fallback)
    Always(err == nil, "Fallback dial returned a nil error")
}
```

## 3.5 Closures

A closure is its own function. Its body has its own prologue, its own
return-assertion defer, and its own declaration-following-call rule. The
analyzer descends into closure bodies independently of the enclosing
function.

```go
input.Each_Item(func(item Item) (kept bool) {
    defer Always(kept || !kept, "Kept is a defined bool")  // trivial defer
    Always(item.ID != "", "Item ID is non-empty")
    // ...
    return true
})
```

## 3.6 Conditionals and loops

The same declaration-following-call rule applies in any block scope.

```go
for _, host := range hosts {
    conn, err := dial(host)
    Always(err == nil, "Dial returned a nil error")
    // ...
}

if input.Use_Cache {
    cached, err := load(input.Cache_Path)
    Always(err == nil, "Load returned a nil error")
    // ...
}
```

---

# 4. What

The required predicates are listed in §7. This section defines which Go
types each rule applies to.

## 4.1 Types in scope

- **Integers** — every signed and unsigned width. `int`, `int8`, …,
  `uint64`.
- **`uintptr`** — zero-check only; Boundary does not apply.
- **`rune`** — out of scope for now.
- **Floats** — `float32`, `float64`. Integer rules plus NaN/Inf checks.
- **Booleans**.
- **Strings**.
- **Slices** — nil-check, `len`, `cap`.
- **Maps** — nil-check, `len` only. Go has no `cap` on maps.
- **Channels** — nil-check, `len`, `cap`.
- **Structs** — recurse all fields.
- **Pointers** — nillable rules plus pointee recursion.
- **Interfaces** (including `error`) — nil-check only. The method set and
  dynamic type MUST NOT be recursed.
- **Function-typed values** — nil-check only.

## 4.2 Wrappers, aliases, variadic

- Wrapper types (`type User_ID int`) take the underlying type's rules.
- Type aliases (`type Status = pkg.Status`) resolve to the underlying type,
  then apply the rules.
- Variadic `...T` is asserted as `[]T`.
- `[]byte` takes slice rules ONLY; the string-text predicates do NOT apply.

## 4.3 Generics

The type-set constraint of a type parameter determines its rules:

- `[T Integer_Like]` — integer rules.
- `[T Numeric]` — float rules (the union covers floats).
- `[T any]` — opaque; no required predicates.

```go
func clamp[T Integer_Like](x T, lo T, hi T) (clamped T) {
    defer Boundary(&Boundary_Input[T]{
        X: clamped, Lo: 0, Hi: 0xFFFF,
        Message: "Clamped stays within range",
    })
    Cross_Product(
        Boundary(&Boundary_Input[T]{X: x, Lo: 0, Hi: 0xFFFF,
            Message: "X is in range"}),
        Always(x != 0, "X is set"),
        Boundary(&Boundary_Input[T]{X: lo, Lo: 0, Hi: 0xFFFF,
            Message: "Lo is in range"}),
        Boundary(&Boundary_Input[T]{X: hi, Lo: 0, Hi: 0xFFFF,
            Message: "Hi is in range"}),
    )
    // ...
}
```

## 4.4 Foreign-field boundary

Stdlib and external-module types are foreign. Foreign types appearing as
parameters, returns, or call results:

- Receive NO per-type assertion of their own.
- Are NOT recursed.

```go
// stdlib type as a parameter — no assertions on `now` or its fields.
func stamp(now time.Time) (encoded string) {
    defer Always(encoded != "", "Encoded stamp is non-empty")
    // ...
}
```

Our own code that returns a foreign type IS asserted at the declaration
site, AND the foreign type's public fields are recursed. The skip applies
to private fields (which we never own) but not to public ones (which the
foreign type's API surfaces and our caller may inspect).

```go
// our function returns *url.URL — recurse the public fields.
u := our_resolver.Resolve(input.Raw)
Cross_Product(
    Always(u != nil, "Resolved URL is non-nil"),
)
if u != nil {
    Cross_Product(
        Boundary(&Boundary_Input[int]{X: len(u.Scheme), Lo: 1, Hi: 16,
            Message: "Scheme length is reasonable"}),
        Always(u.Scheme != "", "Scheme is non-empty"),
        Always(!Has_Control(u.Scheme), "Scheme contains no control char"),
        Always(!Has_Null_Byte(u.Scheme), "Scheme contains no NUL byte"),
        Sometimes(Has_Whitespace(u.Scheme), "Schemes with whitespace exercised"),
        Boundary(&Boundary_Input[int]{X: len(u.Host), Lo: 1, Hi: 253,
            Message: "Host length within DNS limit"}),
        // ...remaining public fields elided
    )
}
```

## 4.5 Struct field recursion

Recurse all fields in the struct's declaration order:

- **Private fields** are skipped only when the struct is defined in another
  package. In our own packages, private fields are recursed.
- **Embedded fields** recurse transparently — treat them as if their fields
  were spelled out on the outer struct.
- **Recursive types** (e.g. `type Node struct { Next *Node }`) unroll at
  most twice. The walker carries a per-type visit counter and stops
  pushing children once the counter reaches 2; this lets a self-referential
  struct's own fields be asserted at least once on the back-edge instead
  of bailing on the first cycle.
- **Foreign-type fields** (`*ast.SelectorExpr` whose package qualifier
  resolves to a stdlib import — `time.Time`, `context.Context`,
  `sync.WaitGroup`, `sync.Once`, `sync.Map`, `sync/atomic.*`) are
  opaque: no requirement is emitted and the walker does not recurse.
- **Opaque-on-mutex**: a struct whose declaration includes a `sync.Mutex`
  or `sync.RWMutex` field is itself opaque — its sibling fields produce
  no requirements at this layer because the lock's contract says those
  fields' invariants are meaningful only inside a `Lock()`/`Unlock()`
  critical section, not at function entry. The containing pointer (if
  any) still requires its own nil-check.
- **Blank-named sync mutex (`_ sync.Mutex` / `_ sync.RWMutex`)** is
  forbidden: the field has no usable Lock receiver, so its only effect
  would be to suppress opaque-on-mutex recursion (an escape hatch dressed
  as a mutex). Use the `noCopy` idiom for non-copy semantics; name the
  field if you actually want to lock it.

Documented follow-up (out of scope for this section): two complementary
analyzers turn opaque-on-mutex from a blanket skip into a real contract:
(a) a dead-lock check that requires every `sync.Mutex` field to have at
least one `receiver.field.Lock()` (or `RLock()`) call somewhere in the
same package, and (b) a critical-section assertion analyzer that requires
the assertions on opaque-on-mutex-skipped sibling fields to appear inside
the matching `Lock()`/`Unlock()` block.

```go
type Server_Input struct {
    Listener Listener_Input    // recurse transparently
    Logger   *Logger           // pointer — nil-check + pointee recurse
    private  int               // recursed (same package)
}

type Listener_Input struct {
    Port int
    Host string
}

func serve(input Server_Input) (err error) {
    defer Sometimes(err == nil, "Returned error is sometimes nil")
    Cross_Product(
        Boundary(&Boundary_Input[int]{X: input.Listener.Port, Lo: 1, Hi: 65535,
            Message: "Port in TCP range"}),
        Always(input.Listener.Port != 0, "Port is set"),
        Boundary(&Boundary_Input[int]{X: len(input.Listener.Host), Lo: 1, Hi: 253,
            Message: "Host length is reasonable"}),
        Always(input.Listener.Host != "", "Host is non-empty"),
        Sometimes(Has_Whitespace(input.Listener.Host), "Host with whitespace exercised"),
        Always(!Has_Control(input.Listener.Host), "Host contains no control char"),
        Always(!Has_Null_Byte(input.Listener.Host), "Host contains no NUL byte"),
        Sometimes(input.Logger == nil, "Logger is sometimes nil"),
        Boundary(&Boundary_Input[int]{X: input.private, Lo: 0, Hi: 100,
            Message: "Private counter in range"}),
        Sometimes(input.private == 0, "Private counter is sometimes zero"),
    )
    if input.Logger != nil {
        // Logger fields recursed here under the Implication.
    }
    // ...
}
```

## 4.6 Pointer-to-pointer

`**T` is recursed: nil-check the outer, then nil-check the inner inside the
outer's Implication, then apply T's rules inside the inner's Implication.

```go
func bind(pp **Config) (err error) {
    defer Sometimes(err == nil, "Returned error is sometimes nil")
    Sometimes(pp == nil, "PP is sometimes nil")
    if pp != nil {
        Sometimes(*pp == nil, "Inner pointer is sometimes nil")
        if *pp != nil {
            cfg := *pp
            Cross_Product(
                Boundary(&Boundary_Input[int]{X: cfg.Retry, Lo: 0, Hi: 8,
                    Message: "Retry stays within budget"}),
                Always(cfg.Retry != 0, "Retry is set"),
            )
        }
    }
    // ...
}
```

---

# 5. Grammar

## 5.1 Allowed predicate shapes

The grammar applies to every predicate-bearing call: `Always`,
`Sometimes`, `Is_Always`, `Is_Sometimes`. A top-level predicate MUST
match exactly one of these shapes:

- Bare ident-chain: `x`, `x.y`, `x.y.z`, `!x`, `!x.y`.
- `len(x)` or `cap(x)` in the operand position of the rules below.
- `X ==/!= nil`.
- `X ==/!= 0`.
- `X ==/!= ""`.
- `X ==/!= <literal-or-untyped-const>`.

Inside `Cross_Product`, axis arguments are call expressions
(`Always(...)`, `Sometimes(...)`, `Boundary(...)`) whose individual
predicates obey the same grammar.

## 5.2 Canonical zero/nil/empty form for Sometimes

The Always family (`Always`, `Is_Always`) credits both `X == zero` and
`X != zero` for the sentinels `nil`, `0`, `""`. The Sometimes family
(`Sometimes`, `Is_Sometimes`) credits ONLY the `X == zero` form. The
`!=` form is not a valid requirement-satisfying shape; if it is the
only assertion present for a nillable / int / string requirement, the
linter emits `invariant_assertion_missing` because the requirement is
unmet.

The rationale: `Sometimes` asserts that the SENTINEL state is reached.
`Sometimes(X == nil)` is the coverage observation that "this code path
was exercised with a nil X." `Sometimes(X != nil)` is trivially true the
moment any non-nil flows through; it carries no coverage information.

```go
// CORRECT — canonical form.
Sometimes(input.Optional == nil, "Optional is sometimes nil")

// VIOLATION — Sometimes(X != zero) does not satisfy the requirement;
// the linter reports a missing assertion for input.Optional.
Sometimes(input.Optional != nil, "Optional is sometimes non-nil")
```

## 5.3 Forbidden at top level

- Parentheses around the whole predicate.
- `&&` or `||`.

```go
// VIOLATION — parens
Always((x != nil), "X is non-nil")

// VIOLATION — top-level &&
Always(x != nil && x.Ready, "X is ready")
// CORRECT — nest the guard
if x != nil {
    Always(x.Ready, "X is ready")
}
```

Compound boolean operators nested INSIDE a function-call argument are
fine; only the top of the predicate is inspected:

```go
Always(In_Range(x && y, ...), "Range covers conjunction")  // OK
```

## 5.4 Message text

Message-text rules — literal-only, starts uppercase, ≥3 words, ≥10 runes,
no negative words — are enforced by `golang_snacks/invariant/v2` at
registration time and are NOT restated here.

---

# 6. Implication

## 6.1 Definition

`if X != nil { ... }` credits assertions inside the body, where `X` is an
ident-chain. The walker descends into the body of every Implication, in
both function prologues and defer bodies.

## 6.2 Init clauses

An init clause counts as the declaration's required assertion for the
introduced identifier:

```go
if x := load(input.Path); x != nil {
    // The if x != nil here is BOTH the decl assertion AND an Implication
    // for x. No second assertion line is required.
    Always(x.Ready, "Loaded x is ready")
}
```

## 6.3 Compound conditions are forbidden

```go
// VIOLATION
if x != nil && y != nil {
    // ...
}

// CORRECT — nest
if x != nil {
    if y != nil {
        // ...
    }
}
```

## 6.4 `else`, type switches, other guards

- `else` branches receive NO credit. They mean `X` is nil — there is
  nothing in scope to assert against.
- Type switches do NOT create Implications. `switch v := x.(type)` is out
  of scope.
- Other condition shapes (`if flag`, `if x == nil`, `if X > 0`, …) do NOT
  create Implications.

## 6.5 Nesting

Implications nest. The walker descends through each level:

```go
if outer != nil {
    if outer.Inner != nil {
        Always(outer.Inner.Value != 0, "Inner value is set")
    }
}
```

FuncLit boundaries are not crossed by the Implication descent — a closure
starts its own walk.

## 6.6 Where assertions MUST live

A required assertion whose field path walks only through value types
(receivers, value-struct fields, plain ints / bools / strings) MUST appear
OUTSIDE any Implication. Hiding it inside `if X != nil { ... }` is a
violation, because the assertion would silently disappear when `X` is nil.

When the path includes a nillable segment (`*T`, slice, map, channel,
interface), the assertion MAY appear inside the corresponding Implication —
that is the only safe place to deref the pointer.

```go
// input.N has no nillable ancestor — MUST sit at prologue top level.
Always(input.N == 0, "Input N starts at zero")

// VIOLATION — input.N hidden inside an Implication on a sibling pointer.
if input.Logger != nil {
    Always(input.N == 0, "Input N starts at zero")
}
```

```go
// diag is a pointer — recurse its fields INSIDE the guard.
defer func() {
    Sometimes(diag == nil, "Returned diag is sometimes nil")
    if diag != nil {
        Always(diag.Tier == 0, "Diag tier defaults to zero")
    }
}()
```

---

# 7. How

For each type, ONE form on each bullet MUST appear at the call site. When
the bullet lists `Always(...) OR Sometimes(...)`, the author picks exactly
one.

## 7.1 Nillables (pointers, interfaces, channels, maps, slices, funcs)

- `Always(x ==/!= nil)` OR `Sometimes(x == nil)`.

```go
Always(input.Pointer != nil, "Pointer is non-nil")
Sometimes(input.Optional == nil, "Optional is sometimes nil")
```

## 7.2 Integers

- `Boundary(&Boundary_Input[T]{X: x, Lo: ..., Hi: ..., Message: "..."})`.
  The `Lo` and `Hi` values MUST be decimal, negative, hex, octal, or binary
  integer literals; constant expressions (`1 << 8`, `1024 * 1024`); or
  package-level untyped consts. Boundary is required for every integer; use
  a workspace cap when no tighter fit exists.
- `Always(x ==/!= 0)` OR `Sometimes(x == 0)`. (See §5.2 — Sometimes does
  not credit the `!=` form.)

```go
Boundary(&Boundary_Input[int]{
    X: input.Retry, Lo: 0, Hi: 8,
    Message: "Retry stays within budget",
})
Always(input.Retry != 0, "Retry is set")

Boundary(&Boundary_Input[int]{
    X: input.Offset, Lo: -1, Hi: 1 << 32,
    Message: "Offset stays in 32-bit range",
})
Sometimes(input.Offset == 0, "Offset is sometimes zero")
```

## 7.3 `uintptr`

- `Always(x ==/!= 0)` OR `Sometimes(x == 0)`. Boundary does NOT apply —
  uintptr's range is opaque.

```go
Always(input.Handle != 0, "Handle is set")
```

## 7.4 Float

- The integer rules (`Boundary` + zero-check).
- `Always(!math.IsNaN(x), ...)`.
- `Always(!math.IsInf(x, 0), ...)`.

```go
Boundary(&Boundary_Input[float64]{X: score, Lo: 0, Hi: 100,
    Message: "Score stays within percent range"})
Always(score != 0, "Score is set")
Always(!math.IsNaN(score), "Score is a real number")
Always(!math.IsInf(score, 0), "Score is finite")
```

## 7.5 Booleans

- `Always(x)` OR `Sometimes(x)`.

```go
Sometimes(input.Enabled, "Enabled flag is exercised both ways")
Always(input.Initialized, "Subject is initialized")
```

## 7.6 String

- The integer rules applied to `len(x)`. `Hi` MUST be a real bound;
  unbounded strings are forbidden.
- `Always(Has_Whitespace(x))` OR `Sometimes(Has_Whitespace(x))`.
- `Always(Has_Control(x))` OR `Sometimes(Has_Control(x))`.
- `Always(Has_Null_Byte(x))` OR `Sometimes(Has_Null_Byte(x))`.

The text predicates live in `invariant` itself. The rules apply to every
string regardless of provenance.

```go
Boundary(&Boundary_Input[int]{X: len(input.Name), Lo: 1, Hi: 64,
    Message: "Name length stays within limit"})
Always(len(input.Name) != 0, "Name is non-empty")
Sometimes(Has_Whitespace(input.Name), "Names with whitespace exercised")
Always(!Has_Control(input.Name), "Name contains no control character")
Always(!Has_Null_Byte(input.Name), "Name contains no NUL byte")
```

## 7.7 Slices

- The integer rules applied to `len(x)` AND `cap(x)`.
- Nil and empty are distinct concerns:
  - `Sometimes(x == nil, ...)` or `Always(x != nil, ...)` covers nil.
  - `len(x) == 0` is asserted INSIDE the non-nil Implication.

```go
Sometimes(input.Items == nil, "Items is sometimes nil")
if input.Items != nil {
    Boundary(&Boundary_Input[int]{X: len(input.Items), Lo: 0, Hi: 1024,
        Message: "Item count within cap"})
    Boundary(&Boundary_Input[int]{X: cap(input.Items), Lo: 0, Hi: 1024,
        Message: "Item capacity within cap"})
    Sometimes(len(input.Items) == 0, "Items is sometimes empty when non-nil")
}
```

## 7.8 Maps

- The integer rules applied to `len(x)` only. `cap` is undefined for maps
  and MUST NOT be asserted.

```go
Sometimes(input.Tags == nil, "Tags is sometimes nil")
if input.Tags != nil {
    Boundary(&Boundary_Input[int]{X: len(input.Tags), Lo: 0, Hi: 64,
        Message: "Tag count within cap"})
}
```

## 7.9 Channels

- The nillable rules apply.
- The integer rules apply to `cap(x)` only. `len(x)` is excluded because the
  observation is racy under concurrent producers/consumers (the value can
  change between the read at the assertion site and the next instruction)
  and tautological-or-instantaneous against any literal bound where `Hi ≥
  cap`. `cap(x)` is stable for the channel's lifetime — it carries the
  buffered-vs-unbuffered distinction and the buffer-size contract, which is
  what assertions can usefully pin.

## 7.10 Structs

- Recurse all fields per §4.5. Cross-package private fields are skipped.

## 7.11 Pointers

- The nillable rules apply.
- The pointee's type is recursed inside the corresponding Implication.
- Every body dereference `*p` of a pointer asserted as `Sometimes(p == nil)`
  MUST appear inside an Implication for `p`. Pointers asserted as
  `Always(p != nil)` MAY be dereferenced freely.

```go
Always(input.Cfg != nil, "Cfg is non-nil")
// Safe to deref input.Cfg freely below this line.
use(*input.Cfg)

Sometimes(input.Tracer == nil, "Tracer is sometimes nil")
// VIOLATION — bare deref of a maybe-nil pointer.
input.Tracer.Begin()
// CORRECT — deref inside the Implication.
if input.Tracer != nil {
    input.Tracer.Begin()
}
```

## 7.12 Interfaces and functions

- The nillable rules apply. Nothing else.

```go
Always(input.Logger != nil, "Logger is non-nil")    // interface
Always(input.On_Tick != nil, "On_Tick is non-nil")  // func-typed
```

## 7.13 Receivers

Receivers obey the same rules as parameters of the same type. A pointer
receiver gets the nil-check; a value receiver gets the field recursion.

```go
func (s *Server) Serve(input Listener_Input) (err error) {
    defer Sometimes(err == nil, "Returned error is sometimes nil")
    Cross_Product(
        Always(s != nil, "Receiver is non-nil"),
        Boundary(&Boundary_Input[int]{X: input.Port, Lo: 1, Hi: 65535,
            Message: "Port in TCP range"}),
        Always(input.Port != 0, "Port is set"),
        // ...
    )
    // ...
}

func (s Stamper) Stamp() (out string) {
    defer Always(out != "", "Stamp is non-empty")
    Cross_Product(
        Always(s.Now != nil, "Now is injected"),
        Always(s.Format != "", "Format is non-empty"),
        // ...string-rules elided for brevity
    )
    // ...
}
```

---

# 8. Cross_Product

## 8.1 When

`Cross_Product` is REQUIRED when a unit has ≥2 requirements. Below the
threshold, the bare call stands alone.

- Unit = function prologue, single declaration statement, or
  return-assertion defer body.
- Exactly ONE `Cross_Product` is credited per unit.

## 8.2 Subsumption

`Cross_Product` subsumes the standalone calls. Axis-calls are inlined as
arguments; the same predicate MUST NOT appear as a sibling standalone
call.

```go
// VIOLATION — duplicate predicate.
Cross_Product(
    Boundary(&Boundary_Input[int]{X: x, Lo: 0, Hi: 100, Message: "X in range"}),
    Always(x != 0, "X is set"),
)
Always(x != 0, "X is set")  // ← duplicate; remove

// CORRECT — single Cross_Product, no siblings.
Cross_Product(
    Boundary(&Boundary_Input[int]{X: x, Lo: 0, Hi: 100, Message: "X in range"}),
    Always(x != 0, "X is set"),
)
```

## 8.3 Axis shape

The argument shape (axis registration, `Excluding` / `Solely` cells,
`Bucket_*` references) is defined in `golang_snacks/invariant/v2`. The
spec does not pin it.

---

# 9. Defer body

The return-assertion defer body MUST contain exactly:

- One `Cross_Product` (or one bare call when there is a single requirement)
  covering the named returns.
- Implications wrapping any nillable returns whose pointee is recursed.

Nothing else. Cleanup defers are separate top-of-body defers; they do not
nest inside the assertion defer.

```go
defer func() {
    Cross_Product(
        Always(err == nil, "Returned error is nil"),
        Sometimes(out == nil, "Out is sometimes nil"),
    )
    if out != nil {
        Cross_Product(
            Boundary(&Boundary_Input[int]{X: out.Tier, Lo: 0, Hi: 5,
                Message: "Out tier within taxonomy"}),
            Always(out.Tier != 0, "Out tier is set"),
        )
    }
}()
```

---

# 10. Diagnostics

- Every violation is an error. CI fails.
- One diagnostic per missing assertion.
- The diagnostic NAMES the missing assertion (e.g. "int field `input.N` is
  missing its zero-check"). It does NOT emit the exact code to insert.
- No auto-fix.
- No opt-out directives — fix the violation.

The `Recorder_` variants in `golang_snacks/invariant/v2` are never
required by this spec. Use the bare names (`Always`, `Sometimes`,
`Boundary`, `Cross_Product`).
