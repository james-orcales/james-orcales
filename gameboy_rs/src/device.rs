//! The headless composition root: assemble the machine from a ROM image and drive
//! it for a fixed tick budget, returning the framebuffer and the serial output.
//! Deterministic — no window, no audio, no wall clock — so two runs of the same ROM
//! for the same budget produce identical output, which is what the differential
//! harness and the blargg serial oracle compare.

use crate::cpu;
use crate::gbmode;
use crate::mbc;
use std::ops;

/// The observable result of a headless run: the RGB framebuffer, every byte the
/// program transmitted over the serial port, and the synthesized stereo audio
/// (interleaved left/right f32 samples).
#[derive(Clone, Debug)]
pub struct Run_Output {
    pub framebuffer: Vec<u8>,
    pub serial: Vec<u8>,
    pub audio: Vec<f32>,
}

/// Runs `rom` in the given console mode until at least `budget` PPU T-cycles have
/// elapsed, returning the final framebuffer and accumulated serial output.
pub fn run_headless(rom: Vec<u8>, mode: gbmode::Gb_Mode, budget: u64, now: u64) -> Result<Run_Output, String> {
    let machine = cpu::set_clock(cpu::new(mbc::get_mbc(rom, false)?, mode)?, now);
    Ok(output_of(run(machine, budget)))
}

/// Runs `rom` on CGB hardware (the Color/compatibility mode chosen from the header)
/// for the tick budget with the injected clock, returning the observable output.
pub fn run_headless_cgb(rom: Vec<u8>, budget: u64, now: u64) -> Result<Run_Output, String> {
    let machine = cpu::set_clock(cpu::new_cgb(mbc::get_mbc(rom, false)?)?, now);
    Ok(output_of(run(machine, budget)))
}

/// Drives a built machine for the tick budget and returns the final machine, so the
/// composition root can read its outputs and battery RAM. The clock is injected by
/// the caller before this point; the run itself is a pure fold over cycles.
pub fn run(machine: cpu::Cpu, budget: u64) -> cpu::Cpu {
    run_cycles(machine, budget)
}

/// The observable output of a finished machine: framebuffer, serial, and audio.
pub fn output_of(done: cpu::Cpu) -> Run_Output {
    let audio = done.mmu.sound.audio_left.iter().zip(&done.mmu.sound.audio_right).flat_map(|(&l, &r)| [l, r]).collect();
    Run_Output { audio, framebuffer: done.mmu.gpu.data, serial: done.mmu.serial.output }
}

// Threads the machine through `do_cycle` until the tick budget is met. `try_fold`
// gives a constant-stack loop (no recursion for the tens of millions of steps a
// test ROM needs) that moves the state through by value (no clone) and breaks the
// instant the budget is reached. Each step advances >= 1 tick, so the tick budget
// bounds the step count.
fn run_cycles(machine: cpu::Cpu, budget: u64) -> cpu::Cpu {
    let outcome = (0..budget).try_fold((machine, 0u64), |(current, elapsed), _| {
        let (next, ticks) = cpu::do_cycle(current);
        let total = elapsed + ticks as u64;
        match total >= budget {
            true => ops::ControlFlow::Break(next),
            false => ops::ControlFlow::Continue((next, total)),
        }
    });
    match outcome {
        ops::ControlFlow::Break(machine) => machine,
        ops::ControlFlow::Continue((machine, _)) => machine,
    }
}
