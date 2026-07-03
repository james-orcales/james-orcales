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
representation recovers ~10% of the wall-time tax and, more usefully, proves the
remaining ~18× is structural value-threading, not `memcpy`.

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
