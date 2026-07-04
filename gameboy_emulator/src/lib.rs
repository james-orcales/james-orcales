//! An immutable-dialect Game Boy (Color) emulator: a value-threaded reimplementation
//! of the mutation-heavy `rboy`, written to pass `lint_rs`. State evolves by
//! consume-and-return (`step(cpu) -> (Cpu, u32)`) rather than in-place mutation, and
//! the memory regions rboy writes in place are rebuilt as whole values (see the
//! `memory` module), since the dialect bans `mut` and structural sharing across
//! versions would need the banned `Rc`/`Arc`.

pub mod alu;
pub mod blip_buf;
pub mod cpu;
pub mod device;
pub mod gbmode;
pub mod gpu;
pub mod gpu_render;
pub mod hash;
pub mod keypad;
pub mod mbc;
pub mod memory;
pub mod mmu;
pub mod printer;
pub mod region;
pub mod register;
pub mod serial;
pub mod sound;
pub mod state;
pub mod timer;
