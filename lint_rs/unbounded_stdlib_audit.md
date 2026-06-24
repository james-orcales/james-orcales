# Unbounded stdlib audit

An audit of the Rust standard library for **unbounded APIs** — any call whose work or memory grows
with attacker-controlled input and has no explicit cap — to define `lint_rs`'s unbounded-API bans.

Grounded in the actual `rust-src`, not from memory: `rustc 1.96.0 (2026-05-25)`, toolchain
`stable-aarch64-apple-darwin`. Every entry cites `library/<crate>/…:line`, shortened below to
`std/…`, `core/…`, `alloc/…`.

## Headline: most unbounded APIs are already dead under the `mut` ban

Every `&mut self` reader, every `reserve`/`resize`, every socket `recv`, `io::copy`, `Child::wait`,
and `Condvar::wait` is uncallable — you cannot form the `&mut` to reach it. So the *live* surface
the dialect must actually ban is far smaller than the raw list. Each entry below is marked **LIVE**
(callable mut-free → must be handled) or *dead* (already blocked, listed for completeness).

The bounded form determines the bucket:

- **blacklist** — bounded form is a *different* function, so a flat call-name ban has no false
  positives.
- **take** — same function; bounded only when a `take(N)` (`io::Read` or `Iterator`) caps it.
- **literal-arg** — bounded iff the size argument is a literal, not input-derived.

## Reads — `std/src/io`, `std/src/fs.rs`

| API | Location | Status | Bounded form | Bucket |
|---|---|---|---|---|
| `fs::read` | `std/src/fs.rs:341` | LIVE | `File::open` + `take` + drain | blacklist |
| `fs::read_to_string` | `std/src/fs.rs:383` | LIVE | same | blacklist |
| `fs::read_dir` | `std/src/fs.rs:3313` | LIVE (lazy) | hazard is `.collect()`, not the call | — |
| `io::read_to_string` | `std/src/io/mod.rs:1328` | LIVE | `io::read_to_string(r.take(N))` | take |
| `io::repeat` (infinite reader) | `std/src/io/util.rs:263` | LIVE | `.take(N)` before draining | take |
| `Read::bytes` | `std/src/io/mod.rs:1144` | LIVE | on `r.take(N)` | take (clashes with `str::bytes`) |
| `BufRead::lines` | `std/src/io/mod.rs:2684` | LIVE | on `r.take(N)` | take (clashes with `str::lines`) |
| `BufRead::split` | `std/src/io/mod.rs:2647` | LIVE | on `r.take(N)` | take (clashes with `split`) |
| `Stdin::lines` | `std/src/io/stdio.rs:432` | LIVE | `.take(N)` | take |
| `Read::take` | `std/src/io/mod.rs:1221` | LIVE | — (this *is* the cap) | allowed |
| `Read::read_exact` | `std/src/io/mod.rs:1026` | dead (`&mut`) | — (bounded anyway) | — |
| `io::copy` | `std/src/io/copy.rs:62` | dead (`&mut` args) | — | — |
| `Read::read_to_end` / `read_to_string` | `std/src/io/mod.rs:917` / `:973` | dead (`&mut`) | — | — |
| `BufRead::read_until` / `read_line` | `std/src/io/mod.rs:2476` / `:2609` | dead (`&mut`) | — | — |
| `Stdin::read_line` | `std/src/io/stdio.rs:411` | dead (`&mut buf`) | — | — |

## Infinite iterators — `core/src/iter`

| API | Location | Status | Bounded form | Bucket |
|---|---|---|---|---|
| `iter::repeat` | `core/src/iter/sources/repeat.rs:64` | LIVE | `iter::repeat_n` (real twin) | blacklist |
| `iter::repeat_with` | `core/src/iter/sources/repeat_with.rs:65` | LIVE | `.take(N)` | take |
| `iter::successors` | `core/src/iter/sources/successors.rs:22` | LIVE | `.take(N)` | take |
| `iter::from_fn` | `core/src/iter/sources/from_fn.rs:45` | LIVE | `.take(N)` | take |
| `Iterator::cycle` | `core/src/iter/traits/iterator.rs:3592` | LIVE | `.take(N)` | take |
| `iter::repeat_n` | `core/src/iter/sources/repeat_n.rs:59` | LIVE | — (the bounded twin) | allowed |

## Allocation — `alloc`, `std/src/collections`

Live ones are the by-value constructors; the `&mut self` growers (`reserve`/`resize`) are dead.

| API | Location | Status | Bucket |
|---|---|---|---|
| `Vec::with_capacity` | `alloc/src/vec/mod.rs:523` | LIVE | literal-arg |
| `String::with_capacity` | `alloc/src/string.rs:484` | LIVE | literal-arg |
| `HashMap::with_capacity` | `std/src/collections/hash/map.rs:290` | LIVE | literal-arg |
| `HashSet::with_capacity` | `std/src/collections/hash/set.rs:169` | LIVE | literal-arg |
| `VecDeque::with_capacity` | `alloc/src/collections/vec_deque/mod.rs:798` | LIVE | literal-arg |
| `BinaryHeap::with_capacity` | `alloc/src/collections/binary_heap/mod.rs:535` | LIVE | literal-arg |
| `<[T]>::repeat` | `alloc/src/slice.rs:509` | LIVE | literal-arg |
| `str::repeat` | `alloc/src/str.rs:529` | LIVE | literal-arg |
| `vec![x; n]` (macro) | — | LIVE | literal-arg |
| `reserve` / `reserve_exact` / `resize` / `resize_with` | `vec:1470/1500/3491/3141`, `string:1219` | dead (`&mut`) | — |

## Blocking / no timeout — `std/src/net`, `mpsc`, `thread`, `sync`

| API | Location | Status | Bounded form | Bucket |
|---|---|---|---|---|
| `TcpStream::connect` | `std/src/net/tcp.rs:168` | LIVE | `connect_timeout` (`tcp.rs:184`) | blacklist |
| `TcpListener::accept` | `std/src/net/tcp.rs:840` | LIVE | none in std | blacklist |
| `mpsc::Receiver::recv` | `std/src/sync/mpsc.rs:871` | LIVE | `recv_timeout` (`:931`) / `try_recv` (`:812`) | blacklist |
| `thread::JoinHandle::join` | `std/src/thread/mod.rs` | LIVE | none in std | blacklist (see note) |
| `UdpSocket::recv` / `recv_from` / `peek` | `std/src/net/udp.rs:736/148/776` | dead (`&mut buf`) | — | — |
| `Condvar::wait` | `std/src/sync/condvar.rs` | dead (needs banned `Mutex`) | — | — |

## Subprocess — `std/src/process.rs`

| API | Location | Status | Bucket |
|---|---|---|---|
| `Command::output` | `process.rs:1083` | LIVE | blessed |
| `Child::wait_with_output` | `process.rs:2369` | LIVE | blessed |
| `Command::status` | `process.rs:1108` | LIVE but bounded (no capture) | allowed |
| `Child::wait` | `process.rs:2293` | dead (`&mut self`) | — |

`Command::output` / `wait_with_output` are **blessed**, not banned: they are the only mut-free
subprocess captures, and bounding one needs piping plus reaping the child via `Child::wait`
(`&mut self` → banned), so no bounded mut-free form exists. Like `mpsc`, they stay allowed.

## The live ban list, by bucket

Status: the blacklist bucket is implemented in `lint_rs` as flat call-name bans — the path
calls `fs::read`, `fs::read_to_string`, `iter::repeat`, `TcpStream::connect`, plus the methods
`recv` and `accept`. `join` stays legal (it collides with `Path`/slice `join`). The take and
literal-arg buckets are not implemented; they need dataflow the linter does not have.

- **blacklist** — `fs::read`, `fs::read_to_string`, `iter::repeat`, `TcpStream::connect`,
  `TcpListener::accept`, `mpsc::Receiver::recv`, `thread::JoinHandle::join`.
- **blessed** — `Command::output`, `Child::wait_with_output`: unbounded, but the only mut-free
  subprocess captures, with no bounded form (reaping needs `&mut` `wait`). Allowed, like `mpsc`.
- **take** — `io::read_to_string`, `io::repeat`, `Read::bytes`, `BufRead::lines`,
  `BufRead::split`, `Stdin::lines`, `iter::repeat_with`, `iter::successors`, `iter::from_fn`,
  `Iterator::cycle`.
- **literal-arg** — the six `with_capacity`s, `<[T]>::repeat`, `str::repeat`, `vec![x; n]`.
- **already dead under `mut`** — `io::copy`, all `&mut self` `Read`/`BufRead` readers,
  `reserve`/`resize`, socket `recv`/`peek`, `Child::wait`, `Condvar::wait`.

## Detection in `lint_rs` (no type information)

`lint_rs` matches calls syntactically, so the blacklist splits by *call shape*:

- **Path calls** match cleanly on the `(parent, name)` tail — `fs::read`, `iter::repeat`,
  `TcpStream::connect`. Parent-qualified, so collisions are unlikely.
- **Method calls** match only on the bare method name (the receiver type is unknown), so they are
  bannable only when that name does not collide with a common bounded method:
  - banned: `recv`, `accept` — low collision. (`output` / `wait_with_output` are blessed above.)
  - **`join` is NOT banned**: it collides with `Path::join` and `<[T]>::join`, which are pervasive
    and unrelated to blocking. Catching `JoinHandle::join` needs the receiver type, which is
    unavailable. It stays legal.

The **take** and **literal-arg** buckets are not flat-bannable here for the same reason
(`str::lines` vs `BufRead::lines`, literal vs input-derived count) — they need dataflow the linter
does not have, and are tracked separately from this blacklist.
