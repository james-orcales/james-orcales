# gameboy_rs — the immutability tax

`gameboy_rs` reimplements the mutation-heavy `rboy` Game Boy emulator in the
value-oriented, zero-`mut` / zero-`unsafe` dialect `lint_rs` enforces. It is
byte-for-byte identical to rboy — 866 test ROMs match on framebuffer + serial + audio.
This measures what banning in-place mutation costs at runtime.

## Method

Both emulators are driven headless by the same binary (`gameboy_diff`'s `emu_bench`):
read a ROM, run a fixed budget of **40M T-cycles** with audio + serial captured, print
the framebuffer hash. Identical work, identical output (the hash matches) — the only
difference is the emulator core, so the delta is the tax and nothing else. Measured
with `maddox`, this repo's `poop`-style whole-process comparator (Apple-Silicon PMU
counters via `proc_pid_rusage`). `rboy` is the reference (Benchmark 1); the `delta`
column is gameboy_rs's overhead. Audio is synthesized and stored on both sides; rboy's
capture is lock-free (no `Arc`/`Mutex`) so the mutex the correctness harness uses does
not tax it here. The timed path is pure emulation — no save-state / battery file I/O.

```
Machine: Apple M4 (arm64)
  cores: 4 P + 6 E = 10 logical   freq: ?   ram: 16GiB   storage: 460GiB
  L1: 128KiB   L2: 16MiB   OS: macOS 26.2   kernel: Darwin 25.2.0
```

## Results

### CPU-bound — blargg `cpu_instrs`

```
$ maddox "emu_bench rboy cpu_instrs.gb 40000000" \
         "emu_bench mine cpu_instrs.gb 40000000" -warmup=2 -runs=40 -duration=40

Benchmark 1 (40 runs, 5.33s): emu_bench rboy cpu_instrs.gb 40000000
  measurement      mean ± σ              min ... max        outliers
  wall_time       133ms ± 1.86ms       129ms ... 137ms        0 (0%)
  peak_rss      6.73MiB ± 4.27KiB    6.72MiB ... 6.73MiB      3 (8%)
  cpu_cycles       564M ± 7.31M         551M ... 572M         0 (0%)
  instructions    3.67G ± 30.6K        3.67G ... 3.67G        3 (8%)
  cpu_user       3.15ms ± 44.5us      3.06ms ... 3.24ms       0 (0%)
  cpu_system     27.9us ± 1.72us      20.8us ... 31.9us       2 (5%)

Benchmark 2 (16 runs, 42.4s): emu_bench mine cpu_instrs.gb 40000000
  measurement      mean ± σ              min ... max        outliers  delta
  wall_time       2.65s ± 15.8ms       2.62s ... 2.67s        0 (0%)  +1891.4% ±  3.8%
  peak_rss      13.0MiB ± 24.1KiB    12.9MiB ... 13.0MiB      0 (0%)  +  92.5% ±  0.1%
  cpu_cycles      11.0G ± 10.5M        11.0G ... 11.0G        1 (6%)  +1853.2% ±  0.9%
  instructions    48.1G ± 248K        48.1G ... 48.1G         0 (0%)  +1211.1% ±  0.0%
  cpu_user       63.3ms ± 377us       62.6ms ... 63.8ms       0 (0%)  +1905.8% ±  3.8%
  cpu_system      335us ± 4.53us       327us ... 341us        0 (0%)  +1100.7% ±  6.0%
```

### APU-heavy — blargg `dmg_sound`

```
$ maddox "emu_bench rboy dmg_sound.gb 40000000" \
         "emu_bench mine dmg_sound.gb 40000000" -warmup=2 -runs=40 -duration=40

Benchmark 1 (40 runs, 3.65s): emu_bench rboy dmg_sound.gb 40000000
  measurement      mean ± σ              min ... max        outliers
  wall_time      91.3ms ± 921us       88.4ms ... 92.7ms       1 (3%)
  peak_rss      6.73MiB ± 3.53KiB    6.72MiB ... 6.73MiB      2 (5%)
  cpu_cycles       382M ± 2.88M         374M ... 386M        7 (18%)
  instructions    2.57G ± 31.8K        2.57G ... 2.57G        1 (3%)

Benchmark 2 (24 runs, 40.1s): emu_bench mine dmg_sound.gb 40000000
  measurement      mean ± σ              min ... max        outliers  delta
  wall_time       1.67s ± 7.57ms       1.66s ... 1.69s        0 (0%)  +1731.4% ±  2.6%
  peak_rss      17.1MiB ± 43.5KiB    17.0MiB ... 17.2MiB      2 (8%)  + 153.3% ±  0.2%
  cpu_cycles      6.92G ± 10.3M        6.90G ... 6.94G        0 (0%)  +1711.0% ±  0.9%
  instructions    29.5G ± 519K        29.5G ... 29.5G         1 (4%)  +1047.7% ±  0.0%
```

The largest peak-RSS delta lands here (+153%): the four `blip_buf` accumulators plus
the synthesized f32 audio.

### CGB — `cgb-acid2` (double-speed + HDMA + CGB PPU)

```
$ maddox "emu_bench rboy cgb-acid2.gbc 40000000" \
         "emu_bench mine cgb-acid2.gbc 40000000" -warmup=2 -runs=40 -duration=40

Benchmark 1 (40 runs, 4.89s): emu_bench rboy cgb-acid2.gbc 40000000
  measurement      mean ± σ              min ... max        outliers
  wall_time       122ms ± 1.48ms       120ms ... 127ms        1 (3%)
  peak_rss      1.34MiB ± 0          1.34MiB ... 1.34MiB      0 (0%)
  cpu_cycles       496M ± 5.55M         490M ... 511M         0 (0%)
  instructions    3.68G ± 39.6K        3.68G ... 3.68G        1 (3%)

Benchmark 2 (15 runs, 40.2s): emu_bench mine cgb-acid2.gbc 40000000
  measurement      mean ± σ              min ... max        outliers  delta
  wall_time       2.68s ± 33.1ms       2.66s ... 2.79s        1 (7%)  +2094.2% ±  8.5%
  peak_rss      1.76MiB ± 79.4KiB    1.69MiB ... 1.89MiB      0 (0%)  +  31.2% ±  1.8%
  cpu_cycles      10.9G ± 53.5M        10.9G ... 11.1G        1 (7%)  +2108.5% ±  3.4%
  instructions    49.1G ± 1.47M        49.1G ... 49.1G        0 (0%)  +1234.0% ±  0.0%
```

## The tax

| ROM (subsystem)      | wall time | instructions | cpu cycles | peak RSS |
| -------------------- | --------- | ------------ | ---------- | -------- |
| cpu_instrs (CPU)     | **19.9×** | 13.1×        | 19.5×      | 1.9×     |
| dmg_sound (APU)      | **18.3×** | 11.5×        | 18.1×      | 2.5×     |
| cgb-acid2 (CGB/PPU)  | **22.0×** | 13.3×        | 22.0×      | 1.3×     |

Roughly **~20× slower for ~2× the memory**, uniformly across subsystems.

**Where it comes from.** The dialect bans in-place mutation, so a byte written to a
memory region cannot be poked in place — the region is rebuilt as a whole new value:
`[&region[..i], &[v], &region[i+1..]].concat()`, O(1) → O(n). A Game Boy program writes
memory incessantly (every stack push/pop, every RAM/VRAM store, every PPU scanline,
every `blip_buf::add_delta`), so this one substitution dominates the whole run. This is
the "immutability tax" the rewrite set out to measure, and it is large.

**Instructions vs cycles.** gameboy_rs retires ~11–13× more instructions (the memcpy of
each rebuild) but burns ~18–22× more *cycles* — so its IPC is also worse. The gap is the
cache cost: the whole-region copies are memory-bandwidth-bound and evict the working set
from the M4's L1/L2, making each extra instruction more expensive than rboy's in-place
store. The rebuild strategy is punished twice — once in instruction count, once in cache.

**Peak RSS is modest (1.3–2.5×).** The rebuilds are transient — the old `Vec` is freed
the instant the new one is built — so they inflate *allocation churn* (→ cycles) far more
than resident footprint. The APU's +153% is the one real memory cost (the accumulators).

**It's pervasive, not localized.** CGB tops the wall/cycle tax (~22×) because `cgb-acid2`
HALTs the CPU and leaves the PPU rendering, so there the tax is per-scanline framebuffer
splices; the CPU ROM (~20×) is dominated by RAM/stack-write rebuilds; the APU ROM (~18×)
by the blip-buffer rebuild. All three land in the same ~20× band — no subsystem escapes.

## What the tax buys

The same ~6k lines, with zero `mut`, zero `&mut`, zero `unsafe`, zero interior
mutability, and zero `Rc`/`Arc` — every state transition a pure `value -> value`
function, yet still byte-exact with the mutable original across 866 ROMs. The dialect
trades ~20× runtime and ~2× memory for total value-semantics and the absence of an
entire class of aliasing/mutation bugs.

## Raising the ceiling: arena-backed, allocate-once pages

The tax above is the naïve representation (R1): every write rebuilds the whole region.
The dialect's blessed `gen_arena` (`insert`/`update`/`with` — already whitelisted, **no
new primitive, no change to `lint_rs` or `shared_rs`**) lets a region instead be a grid
of fixed-size pages, each a slot inserted once at construction; a write rebuilds only
the one page (O(page), not O(region)). Applied to the two write-heavy hot regions — the
framebuffer (rebuilt per scanline) and WRAM — with `PAGE = 64`, this stays byte-exact
across 241 ROMs and 100% lint-clean (every `mut` remains inside `shared_rs`; gameboy_rs
has zero `mut` tokens still):

| ROM | R1 wall | paged wall | Δ wall | Δ instructions | Δ cycles | peak RSS |
| --- | ------- | ---------- | ------ | -------------- | -------- | ------------- |
| cpu_instrs (CPU) | 2.65s | **2.37s** | −10.6% | −6.4% | −10.7% | 13.0 → 16.9 MiB |
| dmg_sound (APU)  | 1.67s | **1.49s** | −10.8% | −4.4% | −11.4% | ~unchanged |
| cgb-acid2 (CGB)  | 2.68s | **2.58s** | −3.7%  |  ~0%   | −2.8%  | ~unchanged |

**~10–11% off the wall time on CPU/APU, ~4% on render-bound CGB** — still ~18× rboy (the
goal was never to beat it, only to lift the dialect's own floor).

**The tell is that instructions barely move while cycles drop ~11%.** Paging copies far
fewer bytes per write, so the win is almost entirely reduced memory traffic / cache
thrash, not fewer retired instructions — attacking exactly the "cycles outran
instructions" effect noted above. Which pins down the honest limit: **the region
rebuilds were only a minority of the tax.** The instruction-count gap to rboy (~11–13×)
hardly changes, because the bulk is *structural* — threading the whole machine value
through every step — and no arena touches that without abandoning the value-semantics
model the experiment exists to test.

Two more findings:
- **Allocate-once costs memory.** Keeping every page resident plus the per-page arena
  `Slot` overhead (generation + tag) raised CPU peak RSS ~30% (13 → 17 MiB) — a
  deliberate time-for-memory trade.
- **Page write-dominated regions; keep read-dominated ones flat.** Converting VRAM (read
  per-pixel during rendering) *measured a regression* — CGB 2.55 → 2.58s, the per-read
  visitor cost outweighing the rare tile-write savings — so it was reverted. That rule
  is the real takeaway; cart RAM (write-heavy but idle in these ROMs) is left convertible
  for save-heavy titles.

Bottom line: within the existing rules, no new primitive, byte-exact — the paged
representation recovers ~10% of the wall-time tax. Where the *rest* goes is measured
below, not assumed.

## Where the residual tax lives (measured)

The paging result implies the region rebuilds are a minority of the tax, but that is an
inference. Two direct measurements pin it down.

**Isolating microbenchmarks** — both engines run tight loop ROMs (`emu_bench synth:<kind>`:
a valid ROM that loops one operation forever), so the per-instruction ratio attributes
each slice cleanly:

| loop | operation | mine / rboy |
| ---- | --------- | ----------- |
| `nop`   | fetch + decode + peripheral tick (no data memory) | **25.0×** |
| `alu`   | `INC A` | 25.8× |
| `write` | `LD (HL),A` | 21.2× |
| `read`  | `LD A,(HL)` | 19.7× |
| `stack` | `PUSH/POP` | 14.4× |

The **core fetch/decode/tick loop is the worst at 25×**; the memory data ops are a
*lower*-ratio increment on top (isolated op-tax 5.6–12.7×). So the paged region access is
not the bottleneck — the per-`do_cycle` machinery is. Real ROMs blend these to ~18×.

**A Time Profiler run** of the `nop` loop (14.4k samples) locates the cost — and rules out
the obvious suspect:

| self-time | where |
| --------- | ----- |
| ~69% | interpreter + loop (`device::run`, `mmu::do_cycle`, `cpu::do_cycle`/`execute`) |
| ~14% | peripheral tick (`gpu::step_ticks`/`do_cycle`/render, `sound::do_cycle`) |
| ~2%  | `Vec` construction / allocator |
| **~1.2%** | **`_platform_memmove`** |

So despite `Cpu` measuring **1224 bytes** (`size_of`), bulk struct-copy is *not* the tax —
LLVM elides the move. The cost is the value-oriented control flow itself: `Struct { field:
new, ..old }` reads and writes *every* field of the ~1.2 KB `Mmu`/`Gpu`/`Sound` to change
one, and the peripheral tick reconstructs them every cycle — versus rboy's single in-place
field write. That is distributed per-field moves plus genuine interpreter work, i.e.
**compute-bound, not memory-bandwidth-bound**, which is exactly why `memmove` stays at 1%.

**Is ~18× the floor?** It is the floor *of this struct layout* — not of the dialect, and not
fixed. A spike measured the slope directly: box the two biggest cold blocks that ride the hot
loop for nothing (the `Gpu` palettes, 192 B, and `Sound`, 464 B — idle in a `nop` loop) so the
per-cycle reconstruction moves a pointer instead of copying them, and re-measure the core loop:

| threaded bytes removed | `synth:nop` | core tax |
| ---------------------- | ----------- | -------- |
| 0 (baseline)           | 2.75s | 25.0× |
| 192 (palettes)         | 2.49s | 22.6× |
| 656 (+ Sound, ~53% of `Cpu`) | 1.74s | **15.8×** |

The tax falls with the **inline bytes copied per reconstruction — not with field count.** Both
changes boxed a large *inline* field: the palettes (a `[[[u8;3];4];8]` fixed array) and `Sound`
(a 464 B inline struct), turning an inline by-value field into a heap pointer so `..old` moves 8
bytes instead of the payload. No fields were removed. Rust already moves `Vec`/`Box` fields as
fat pointers for free; what costs is specifically the *large inline* fields (fixed-size arrays,
big value-structs) that get byte-copied on every reconstruction — and only across the `gpu`/
`sound` `do_cycle` boundaries the compiler did not inline (a small `Copy` struct like the
register file is shredded into SSA values by SROA and threads for free). So the lever is
**heap-indirect the large inline fields** on the hot path — *not* reduce field count, and *not*
"pass primitives" (you cannot hand a buffer to a function as a scalar). `Box` supplies it: on a
linear timeline it shares the payload across the version transition (move the pointer, rebuild
only on the field's own events) — structural sharing without `Rc`/`Arc`.

This is a **distinct cost from the memory-write tax above.** That one is the buffer *rebuild*
itself (`concat`, O(1)→O(n)), internal to the one field and identical regardless of struct
shape — fixed by `region.rs`'s paging. The core-loop cost here is the *by-value copy of large
inline fields* during reconstruction — fixed by heap-indirecting them. Two costs, two
representations, both orthogonal to how many fields the enclosing struct has.

## Banked: the large inline fields, heap-indirected

The spike is now shipped for real. Exactly the large inline fields on the reconstruction path
were `Box`ed — the two CGB palette arrays (`[[[u8;3];4];8]`, 96 B each), `Sound` (464 B), and the
`Mbc` enum (112 B) — so `..old` moves a pointer, not the payload; each rebuilds only on its own
event (a palette/sound-register or bank-switch write), and `Sound`'s box moves through untouched
when the APU is off, reboxing only when it is actually running (a cheap fixed-size alloc/free, not
a per-cycle copy). Four `Box` fields, no other change. Byte-exact across **241 ROMs**, `lint_rs` =
0, zero `mut`/`unsafe`:

| workload | paged | + 4 boxes | tax vs rboy |
| ------------------ | ----- | --------- | ----------- |
| cpu_instrs (CPU)   | 2.43s | **1.68s** (−31%) | 18.7× → **12.9×** |
| dmg_sound (APU)    | 1.51s | **1.05s** (−31%) | → **13.1×** |
| cgb-acid2 (CGB)    | 2.62s | **1.52s** (−42%) | 21.1× → **12.6×** |
| synth:nop (core)   | 2.70s | **1.60s** (−41%) | 25.0× → **14.5×** |

Four `Box`es cut the real-ROM tax from ~18–21× down to ~13×. That is the whole lever — "large
inline field → heap-indirect," nothing about field count, call shape, or the already-pointer
`Vec`/`Region` fields (their write cost is the separate `region.rs` axis). Not `with_mut`, and not
boxing a genuinely-per-cycle field (that would allocate every cycle).

**Then tick fusion.** `mmu::do_cycle` had threaded the whole `Mmu` through five sequential
reconstructions (timer → keypad → gpu → sound → serial), each moving ~600 B of header. They touch
disjoint fields and share only the commutative interrupt-OR, so they collapse into **one**
reconstruction. Byte-exact across 253 ROMs; synth:nop 1.60→1.42s, cpu_instrs to **13.3×**.

**How low it goes.** Past this the profile is *diffuse* — `mmu` tick ≈25%, `Gpu` logic ≈20%, CPU
interpreter ≈21%, rendering ≈7% — no concentrated lever, because what remains is the genuine
value-threaded work of reconstructing the machine value every cycle plus the fetch/decode/match
dispatch. The dialect's match-over-enums + free-function + ≤70-line style makes that dispatch
inherently heavier than rboy's inlined mutating giant-match, so it is a floor a byte-exact
value-threaded interpreter can't cross. Further hot/cold splitting (`Gpu` config, the `Mmu` behind
a `Cpu`-level arena handle, stack-array scanlines) would shave a few percent each toward ~9–10×,
but ~5× is below the floor of this architecture. The high-leverage, clean wins are banked.

## Reproduce

```
cargo build --release -p gameboy_diff        # builds emu_bench
EB=.local/share/rust/target/release/emu_bench
maddox "$EB rboy <rom> 40000000" "$EB mine <rom> 40000000" -warmup=2 -runs=40 -duration=40
```

`emu_bench <rboy|mine> <rom> <cycles>` runs one engine for the budget and prints the
framebuffer hash; the two engines print the same hash (the run is identical work).

## Size

| build            | source LOC | release binary | notes                                  |
| ---------------- | ---------- | -------------- | -------------------------------------- |
| `gameboy_rs`     | 6,033      | 530 KiB        | headless; `cargo build --release`      |
| `rboy`           | 7,137      | —              | includes its SDL/winit GUI `main`      |

Comparable source size — the tax is entirely at runtime, not in code volume.
