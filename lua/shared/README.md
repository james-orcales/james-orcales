# shared_lua

The Lua analog of `shared` (Go) and `shared_rs` (Rust): a small first-party Lua 5.1 utility library,
ported from the `lua_core` library previously vendored inside the `ostep_odin` project.

## Modules

Requiring the package assembles everything and returns a table:

```lua
local core = require("shared_lua/init") -- from the repo root
```

- **`types`** — the runtime type system (below): the `types.*` combinators, `resolve`, `format`,
  and the `def` decorator, as a standalone module. It `return`s a table, writes no globals, has no
  load-time side effects, and touches no `os`/`io`, so it loads on Lua 5.1–5.4 and LuaJIT and is
  safe to `require` inside embedded sandboxes (OpenResty, HAProxy, `mod_lua`). Depends on nothing.
- **`base`** — a prelude loaded for its side effects: it `require`s `types`, re-publishes `types`
  and `def` as globals for convenience, and adds a replacement `assert`/`unreachable`/
  `unimplemented` that print a traceback and `os.exit(1)`, `sh` (shell exec with `"|"` piping),
  `HOST_SYSTEM_INFO` (populated from `uname` at load), and the
  `eprint`/`fatal`/`fatalf`/`DEBUG`/`INFO`/`WARN`/`ERROR` loggers. Meant for CLI use, not embedding.
- **`strings`** — extends the global `string` table, so its helpers are callable as methods:
  `("a,b,c"):split(",")`, `("  hi  "):trim()`, `("hello world"):to_snake_case()`, plus `cut`,
  `count`, `lines`, the `trim_*` and `*_justify` families, `has_prefix`/`has_suffix`, `contains`,
  `common_prefix`, and the case converters.
- **`array`** — a returned table of higher-order helpers over 1-based arrays: `take`, `drop`, `find`,
  `map`, `any`, `contains`, `each`, `filter`, `count`, `reject`, `zip`, `enumerate`. Require it
  directly: `local array = require("shared_lua/array")`.
- **`core.json`** — lazy-loaded on first access. Tyler Neylon's json.lua (`stringify`/`parse`/`null`).
- **`core.sha`** — lazy-loaded on first access. Egor Skriptunoff's pure_lua_SHA (md5, sha1, sha2,
  sha3, blake2, blake3, hmac). Slow by design; intended for bootstrap checksums.

The modules resolve one another by relative path at load time (via `debug.getinfo`), so require the
package by path from the repo root, exactly as `ostep_odin/ctl.lua` did.

## The `def` type system

`def(inputs..., "->", outputs..., fn)` wraps `fn` so its arguments and return values are checked on
every call. A "type" takes any of three forms, which mix freely:

- a **string token** — `"number"`, `"string"`, `"boolean"`, `"table"`, `"function"`, `"thread"`,
  `"userdata"`, `"any"`, and `"array"` (a *real* Lua sequence: keys `1..n`, no holes). `?` prefixes
  an optional (`"?string"`), `|` forms a union (`"number|string"`).
- a **predicate function** — called with the value; returns truthy to pass. The escape hatch for
  anything the combinators don't cover.
- a **`types.*` combinator** — `array_of(T)` (a dynamic slice `[]T`) or `array_of(T, n)` (a fixed
  array `[n]T`), `map_of(K, V)`, `tuple(T1, …)` (fixed-length, heterogeneous), `shape{k = T, …}`
  (fields required unless wrapped in `optional`), `union(…)`, `optional(T)`, `enum(…)`,
  `literal(v)`, `variadic(T)` (last input slot: zero or more trailing args). Combinators nest.

Optionality is per slot: an optional input may be absent and an optional return may be nil. A
violation raises a **catchable** error (via `error`, blaming the caller) that names the parameter,
extends a path into nested structures, and reports the expected type, the actual type, and the
offending value:

```
def: argument #1.ports[2] expected number, got string "nope"
```

Call `types.disable_signatures(true)` to make `def` return the raw function with zero checking
overhead in production. It is read once, at decoration time, so toggle it before requiring the
modules whose signatures you want raw. It is module-local state, not a global — `types.lua` writes no
globals.

### Using the types without `def` (embedding)

The whole type system — combinators and `def` alike — lives in `types.lua`; `def` is just `types.def`
(re-published as a global `def` by `base`). To validate a value you don't need `def` or `base`: every
type value has a `.check(value)` returning `ok, fault`, and `types.format(fault)` renders a located
message. This is the path for validating untrusted input on a hot path (e.g. a web handler), where
you want the check but not `def`'s per-call wrapping:

```lua
local types = require("shared_lua/types")   -- pure; no globals, no os/io, loads on 5.1–5.4 + LuaJIT
local ResponseShape = types.shape{
        decision    = types.enum("allow", "block", "redirect", "not_matched"),
        status_code = "?number",
        headers     = types.optional(types.map_of("string", "string")),
}
local ok, fault = ResponseShape.check(untrusted)
if not ok then
        log(types.format(fault))   -- e.g. "value.decision expected enum(...), got string \"nope\""
end
```

`display()` guards `tostring` with `pcall` and truncates, so a value with a hostile `__tostring` or
a huge payload can't blow up the error path.

This design was arrived at by surveying runtime type/contract systems (Racket contracts,
clojure.spec, malli, Pydantic, beartype, tableshape, checks): named combinators over a string
mini-language, plain predicates as the escape hatch, and — the biggest fix over the original — a
violation message that carries the offending value and its runtime type located by a path, not just
a bare "wrong type".

## Build / test / run

```sh
# Build the vendored Lua 5.1 interpreter once (posix has no readline dependency):
make -C third_party/lua-5.1.5 posix        # use `macosx` on a Mac

# Run the whole test suite:
third_party/lua-5.1.5/src/lua shared_lua/test.lua
```

`direnv` builds the interpreter and puts it on `PATH` as `lua`, so locally `lua shared_lua/test.lua`
also works. CI runs the same command in the `lua` job of `.github/workflows/ci.yml`.

## Notes on the port

The first-party modules (`base`, `array`, `strings`, `init`) are ported faithfully — the global
injection, the `string`-table extension, and `base`'s `os.exit`/`uname` behavior are all preserved.
The only code changes fix functions that crashed on their first call:

- `def` was originally broken in both directions (its input and return checks rejected the very
  `array`/`any` types the library declares, and a mis-scoped variable made any typed return abort
  the process). It has since been **redesigned** into the combinator type system described above,
  which supersedes the original token-only checker.
- `array.drop` was off by one; `array.filter`/`reject`/`enumerate` called a nonexistent
  `array.insert`.
- `strings.split` inserted into an undefined global; `strings.count` called a nonexistent `:substr`;
  `strings.trim` had a broken duplicate definition and a signature that made the no-argument call
  fatal.

`string.lines` was appending a spurious trailing `""` for input not ending in a newline (and
disagreeing with its own `lines_iter` companion); it now delegates to `lines_iter` so both share one
definition of a line — a trailing newline terminates the last line rather than adding an empty one,
while a genuine empty line mid-string is preserved.

`json.lua` and `sha.lua` are copied verbatim with their upstream headers intact.

## Linter

This repo's linter bans `.lua` files ("rewrite as a go script"). `shared_lua/` is a sanctioned
first-party Lua carve-out, exempted via the `shared_lua/**` entry in the repo-root `lint.json`
`ignore` list — the same mechanism that exempts `install_golang.sh`.

## Licenses

- `LICENSE.apache2.james_orcales`, `LICENSE.zlib.james_orcales` — the first-party modules
  (dual-licensed, as every sibling component is).
- `LICENSE.mit.skriptunoff` — `sha.lua`.
- `LICENSE.publicdomain.tylerneylon` — `json.lua`.
