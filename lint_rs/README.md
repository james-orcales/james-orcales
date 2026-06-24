# lint_rs

`lint_rs` enforces an **immutable Rust dialect built as a hard divergence from mutable-by-default
Go**. It is the linting half of a second implementation: one spec, implemented once in procedural
Go and once in this dialect, with the two cross-checked by differential / integration fuzzing. A
bug survives that cross-check only where *both* implementations agree — and they agree when they
share a shape. Go and a faithful Rust port share the shape that breeds the hardest bugs: mutable
carried state and aliasing. So the dialect is pinned to the opposite pole of that one axis. Where
Go mutates, it forbids mutation; where Go aliases through slices, maps, interfaces, and pointers,
it threads owned values and indexes arenas by handle. The two implementations fail *differently*,
and a disagreement surfaces the bug instead of burying it in agreement.

*Hard* is the operative word: the divergence is compiler-enforced, not a matter of style or
discipline — the linter and the borrow checker together make the mutable shape unwritable. Its
one organizing idea is **OCaml's value-oriented model with the mutability escape hatches nailed
shut** — not a port of TigerStyle or NASA's Power of Ten, though it shares their taste for a
reduced surface.

Run it on a directory; it prints `file:line:col: message` per violation and exits non-zero.

```
cargo run -p lint_rs -- shared_rs/src
```

## Diverge in the core, conform at the edge

The divergence is load-bearing only in the *core*: the algorithm, the data representation, the
carried state — where a shared bug would otherwise hide in agreement. At the *boundary* the
requirement inverts. Differential testing compares outputs, so the twin must *reproduce* the Go
program's output byte-for-byte, quirks and all, not improve on it. An immutable core that diverges
from Go plus an output that conforms to Go is exactly the pair the fuzzer needs: divergent enough
to disagree on a bug, identical enough that every disagreement *is* a bug.

## It forces a real port, not a 1:1 transliteration

The second implementation is, in practice, written by an AI agent translating the Go code — and
an agent's default is the nearest-idiom port. Go makes that port procedural by default: every
variable is mutable (there is no `let` / `let mut` split), and its core types — slices, maps,
interfaces — are reference-semantic and freely aliased. So a 1:1 translation maps each Go idiom
straight onto its mutable Rust twin:

- `x := …; x = …` → `let mut x`
- `append(s, v)` → `s.push(v)` (a `&mut` call)
- `m[k] = v` / `delete(m, k)` → `map.insert` / `map.remove` (a `&mut` call)
- a shared `*Struct` → `&mut` params, or `Rc<RefCell<…>>`

The output is a Rust program with the *same* mutable carried state, the same aliasing, and the
same control flow as the Go original — a transliteration. For differential testing that is
worthless: it re-converges with the Go implementation and reproduces its bugs identically, so the
cross-check has nothing to catch.

`lint_rs` makes every one of those translations a hard error. The agent cannot emit `let mut`,
cannot reach a `&mut` to call `push` or `insert`, cannot store a reference or build a pointer
graph. So it is forced off the transliteration path and made to *re-derive* the computation in
the dialect's shape — a fold instead of an in-place accumulator, an arena handle instead of a
pointer, consume-and-return instead of mutate. The divergence stops depending on the agent
choosing to diverge (it won't) and becomes something the compiler refuses to let it skip.

## Why Rust, not OCaml or Haskell

The linter's whole value is that the divergence is *guaranteed*, not left to discipline — and
only Rust can make that guarantee, because of the borrow checker.

OCaml and Haskell can *host* this style; you can write value-threaded, arena-handle, immutable
code in either. What they cannot do is *enforce* it. Neither tracks borrows or ownership, so
"references flow in, never out" and "no aliasing" are not even checkable properties — there is
nothing in the language to forbid a stored reference or a shared mutable alias. Mutation can
only be chased down by enumerating every mutable API (`ref`, mutable fields, `Array.set`, …), a
denylist that is leaky by construction and that nothing backstops.

In Rust the ban is *sound*. Forbid only the `mut` keyword and the borrow checker makes every
derived `&mut`, every stored or returned borrow, every aliasing escape structurally
uncompilable — so `lint_rs` and the borrow checker together *prove* the subset holds rather
than hoping it does. The "Lifetimes, eliminated" guarantee below is exactly this: borrow-checker
machinery constrained until it has nothing left to reject.

This is what the differential purpose needs. An *unenforced* functional twin can quietly drift
back toward the procedural shape — a `ref` here, a shared mutable cache there — and re-converge
with the Go implementation, erasing the divergence that makes the cross-check find bugs. Only an
enforced subset stays divergent.

The no-GC throughput is real but secondary: the borrow checker is what delivers memory safety
*without* a GC, so the twin runs at native speed with no pauses to cap fuzzing iterations. That
is a consequence of the enforcement mechanism, not the reason to choose it — OCaml is native and
fast too; what it lacks is the checker that makes the guarantee.

## How it forces the divergence

Each ban deletes a whole axis of Rust's complexity and pushes the implementation away from the
procedural Go shape:

- **No `mut`** (one hardcoded path+name whitelist, for the arena primitive). With no `mut`
  binding you cannot form a `&mut`, so every `&mut self` stdlib API is transitively uncallable.
  The line-by-line state Go mutates in place becomes a fold over values threaded through
  returns — so a state-carry bug manifests differently here than in the mutating Go version.
- **No interior mutability** — `Cell`, `RefCell`, `Mutex`, `RwLock`, atomics,
  `Once`/`LazyLock`. The escape hatch that lets `&self` mutate is closed.
- **No `Rc`/`Arc`/`Weak`** — no refcounted shared pointers. Sharing is by arena handle,
  not pointer.
- **No references in fields or returns, and no lifetime parameters.** Borrows flow *in* as
  params and never *out* — which deletes the lifetime system; see *Lifetimes, eliminated*.
- **No user-authored polymorphism** — no inherent methods, no method-bearing `trait`
  declarations. Behavior lives in free functions over data, not methods or trait
  hierarchies. (Trait *impls* of std traits and `#[derive]` remain.)
- **No macro authoring** — `macro_rules!` and proc-macros. Invoking blessed
  macros/derives is fine.
- **Go-style modules** — one module per file, `use` binds a module (never an item), access
  is qualified, no inline `mod` (except `tests`).
- **Casing** — first-char-driven `snake_case` / `Ada_Case`; all struct fields `pub`.

What remains is values, free functions, enums + `match`, and modules: the OCaml core, minus
OCaml's `ref` / mutable-field / `Array.set` escape hatches, which here are statically
forbidden everywhere except one audited primitive.

## Lifetimes, eliminated

Lifetimes and the borrow checker are the part of Rust people actually *fight* — the source of
its learning curve, its hardest compile errors, and most of the contortions (`Rc<RefCell>`,
arenas, `unsafe`) that real codebases adopt to escape them. This dialect removes them almost
entirely, and the reason is structural, not cosmetic.

A reference may appear only as a `&T` **parameter** (or a local); it can never be stored in a
field or returned. So a borrow's lifetime never has to be *named* or *related* to anything —
it is created at a call, used, and dropped. That means **elision always succeeds**: you reach
for `<'a>` only when an output lifetime must be tied to an input, and there are no output
references here. Add no `mut` — which makes the shared-XOR-mutable rule vacuous, since nothing
is mutable to begin with — and the borrow checker has essentially nothing left to reject.

What that deletes from the language you write:

- **`<'a>`, `where T: 'a`, HRTB (`for<'a>`), lifetime variance** — none can be expressed, and
  none is needed.
- **The classic errors** — "borrowed value does not live long enough", "returns a value
  referencing data owned by the current function", "cannot borrow as mutable" — become
  structurally impossible.
- **Self-referential structs and `Pin`** — they exist only to manage stored/returned borrows,
  which are gone.

The only lifetime that survives is `'static` (a fixed bound, not a parameter) and the elided
lifetimes on `&T` parameters, which you never see. So the single hardest thing to learn in
Rust — and the single thing code generators, humans and LLMs alike, most reliably get wrong —
stops being something you interact with at all.

## The architecture it forces

The bans are not stylistic; they push the implementation into a structurally different shape
from a procedural one, which is where the differential value comes from:

- **Data lives in arenas; references are integer handles, not pointers.** See
  `shared_rs::arena` (append-only) and `shared_rs::gen_arena` (generational, with
  stale-handle protection). Graphs and shared structure become `Vec<Node>` + handles — where
  the Go implementation links pointers, this one indexes, so the two diverge exactly where
  pointer-aliasing bugs hide (and the data stays cache-friendly and trivially serializable).
- **Logic is free functions transforming owned, immutable values.** State evolves by
  threading values through returns (consume-and-return), the way an OCaml fold or an Erlang
  receive loop does, not by mutation.
- **Reads borrow inward via a visitor**, e.g. `with(&arena, handle, |v| ...)`, since a
  function may not return `&T`.
- **The one mutable core** is the arena's `insert`/`remove` — the only functions whitelisted
  for `mut` (a `Vec` push has no mut-free O(1) equivalent). Everything above it is pure.
- **Concurrency is share-nothing.** No shared mutable state to guard; threads communicate by
  moving owned values through channels (`mpsc` is blessed) or by fork-join over immutable
  data with `thread::scope`. Where Go shares a buffer across workers, this partitions work,
  spawns workers with their chunk moved in, and collects results at `join` — no shared queue,
  no work-stealing, fully reproducible.

## What it is *not*

It is not TigerStyle / NASA enforcement. It does not chase bounded loops, assertion density,
or zero-allocation; it permits recursion and heap growth. The single idea is **paradigm
divergence from procedural code — immutability with the escape hatches nailed shut** — closer
to OCaml than to a C-flavored safety standard.

## Rules enforced

AST (`lint_rs`): `mut` ban + whitelist; no inherent methods / method traits; module-only
imports; no inlined crate-root paths (`std`/`core`/`alloc`/`crate`/`super` and every
dependency crate must be `use`d and referenced by the bound short name — Go's import-per-
package rule; so `syn::Item` needs `use syn;`, and `std::iter::once` becomes `use std::iter;`
then `iter::once`); no inline modules; casing; `pub` fields; no macro authoring; no reference
fields; no reference returns; no lifetime parameters; function size (a body spans at most 70
lines, brace to brace); entry point first (`fn main` leads its file); comment style (a line
comment opens with a space then a capital and ends in `.`, `:`, `?`, or `!` — trailing
comments and markdown code fences are exempt, and `//` inside a string literal is not a
comment); unbounded-API blacklist (`fs::read`, `fs::read_to_string`, `iter::repeat`,
`TcpStream::connect`, and the methods `recv`/`accept` — each has a bounded twin such as
`repeat_n` / `connect_timeout` / `try_recv`; the take and literal-arg buckets need dataflow
the linter lacks and are not covered). Files that fail to parse are reported, not skipped.

Types (workspace `clippy.toml`, denied by default): no interior mutability
(`Cell`/`RefCell`/`Mutex`/`RwLock`/atomics/`Once`/`LazyLock`); no `Rc`/`Arc`/`Weak`; no
`*::leak` / `UnsafeCell::get`. `mpsc` channels are deliberately allowed — the message-passing
primitive.

`lint_rs` self-hosts: it obeys every rule above, so `cargo run -p lint_rs -- lint_rs/src` is
clean.
