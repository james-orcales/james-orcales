//! Shared Rust libraries for the dialect — one crate, each library a module
//! under `src/` (no per-library manifest). Two arena libraries: `arena`
//! (append-only, bare handle) and `gen_arena` (generational: removal + slot
//! reuse with stale-handle protection).

pub mod arena;
pub mod gen_arena;
