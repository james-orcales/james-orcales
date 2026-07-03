//! The headless composition root: the one place that binds real effects — reading
//! the ROM and its battery save from the filesystem, injecting the wall clock, and
//! writing the battery save back — around the pure, deterministic core. Every effect
//! is injected here; the emulator itself reads no clock and touches no file. `fs::write`
//! and `SystemTime` are dialect-legal (free functions, no `&mut`, not blacklisted), so
//! the boundary stays in the linted crate with no external dependency.

use gameboy_rs::cpu;
use gameboy_rs::device;
use gameboy_rs::gbmode;
use gameboy_rs::hash;
use gameboy_rs::mbc;
use gameboy_rs::state;
use std::env;
use std::fs;
use std::io;
use std::io::Read;
use std::process;
use std::time;

// The largest Game Boy cartridge is 8 MiB; cap the ROM read there.
const MAIN_ROM_BYTES_MAX: u64 = 8 << 20;

// A default budget long enough for the combined blargg cpu_instrs suite to finish.
const MAIN_DEFAULT_CYCLES: u64 = 250_000_000;

fn main() {
    let arguments: Vec<String> = env::args().collect();
    match arguments.get(1) {
        None => {
            eprintln!("usage: gameboy_rs <rom> [cycles]");
            process::exit(2);
        }
        Some(path) => process::exit(run(path, cycles_argument(&arguments))),
    }
}

/// The cycle budget from the optional second argument, or the default.
fn cycles_argument(arguments: &[String]) -> u64 {
    arguments.get(2).and_then(|text| text.parse().ok()).unwrap_or(MAIN_DEFAULT_CYCLES)
}

// The current wall-clock time in Unix seconds — the single impure read, done at the
// boundary and injected into the pure core.
fn wall_clock() -> u64 {
    time::SystemTime::now().duration_since(time::UNIX_EPOCH).map(|elapsed| elapsed.as_secs()).unwrap_or(0)
}

/// Loads the ROM and any battery save, injects the clock, runs, prints the output,
/// and writes the battery save back; returns the process exit code.
fn run(path: &str, cycles: u64) -> i32 {
    let rom = match read_file(path, MAIN_ROM_BYTES_MAX) {
        Some(bytes) => bytes,
        None => {
            eprintln!("could not read ROM: {path}");
            return 1;
        }
    };
    match assemble(rom, path) {
        Ok(machine) => finish(cpu::set_clock(machine, wall_clock()), cycles, path),
        Err(message) => {
            eprintln!("{message}");
            1
        }
    }
}

// Builds the machine for the ROM, choosing CGB from the header and loading any
// existing battery save into the cartridge.
fn assemble(rom: Vec<u8>, path: &str) -> Result<cpu::Cpu, String> {
    let cgb = rom.get(0x143).is_some_and(|&byte| byte & 0x80 != 0);
    let save = read_file(&save_path(path), MAIN_ROM_BYTES_MAX).unwrap_or_default();
    let cart = mbc::load_ram(mbc::get_mbc(rom, false)?, &save);
    match cgb {
        true => cpu::new_cgb(cart),
        false => cpu::new(cart, gbmode::Gb_Mode::Classic),
    }
}

// Runs the machine, prints its output, snapshots a save state, and writes the battery
// save back to disk. Both files are produced by the pure core (`dump_ram`, `save`)
// and only written here, at the boundary.
fn finish(machine: cpu::Cpu, cycles: u64, path: &str) -> i32 {
    let done = device::run(machine, cycles);
    let battery = mbc::dump_ram(&done.mmu.mbc);
    let backed = mbc::is_battery_backed(&done.mmu.mbc);
    let snapshot = state::save(&done);
    let code = report(&device::output_of(done));
    if backed {
        let _ = fs::write(save_path(path), &battery);
    }
    let _ = fs::write(state_path(path), &snapshot);
    code
}

/// Prints the serial text, the framebuffer fingerprint, and the audio fingerprint.
fn report(output: &device::Run_Output) -> i32 {
    print!("{}", String::from_utf8_lossy(&output.serial));
    println!();
    println!("framebuffer: {:016x}", hash::fnv1a(&output.framebuffer));
    let audio_bytes: Vec<u8> = output.audio.iter().flat_map(|sample| sample.to_le_bytes()).collect();
    println!("audio: {} samples, {:016x}", output.audio.len(), hash::fnv1a(&audio_bytes));
    0
}

// The battery-save path for a ROM: the ROM path with a `.gbsave` suffix.
fn save_path(rom_path: &str) -> String {
    format!("{rom_path}.gbsave")
}

// The save-state path for a ROM: the ROM path with a `.savestate` suffix.
fn state_path(rom_path: &str) -> String {
    format!("{rom_path}.savestate")
}

/// Reads a file through the bounded streaming path (bulk `fs::read` is blacklisted).
fn read_file(path: &str, cap: u64) -> Option<Vec<u8>> {
    let file = fs::File::open(path).ok()?;
    io::BufReader::new(file).take(cap).bytes().collect::<io::Result<Vec<u8>>>().ok()
}
