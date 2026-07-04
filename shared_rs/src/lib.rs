//! Shared Rust libraries for the dialect — one crate, each library a module
//! under `src/` (no per-library manifest). `arena` (append-only, bare handle)
//! and `gen_arena` (generational: removal + slot reuse with stale-handle
//! protection) store values by integer handle; `fixedpoint` is the deterministic
//! `f64` stand-in (scale-cancelling arithmetic and a Bhaskara sine); `time` is a
//! dependency-injected clock — a pure virtual clock for simulation plus OS reads;
//! `levenshtein` measures edit distance and picks a "did you mean" candidate;
//! `cli` is a command-line argument parser built on it.

pub mod arena;
pub mod cli;
pub mod fixedpoint;
pub mod gen_arena;
pub mod levenshtein;
pub mod time;
