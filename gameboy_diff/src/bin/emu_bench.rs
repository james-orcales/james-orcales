// emu_bench: a whole-process benchmark target for maddox. Runs ONE emulator engine —
// the original mutable `rboy`, or the immutable `gameboy_rs` — on a ROM for a fixed
// tick budget and prints the framebuffer hash. The two engines do identical work (run
// N cycles with audio + serial captured), so maddox compares them apples-to-apples and
// the delta is the immutability tax. Not linted: this is a measuring instrument, so it
// uses the mutable rboy API freely. The timed path is pure emulation — no file I/O.

use gameboy_rs::device;
use gameboy_rs::gbmode;
use gameboy_rs::hash;
use rboy::device::Device;
use std::env;
use std::fs;
use std::process;

// A lock-free audio sink: rboy synthesizes samples into an owned Vec, mirroring
// gameboy_rs's audio accumulation without the Arc/Mutex the differential harness needs
// for cross-thread capture (that mutex would unfairly tax rboy in this benchmark).
struct Sink_Audio {
    samples: Vec<f32>,
}

impl rboy::AudioPlayer for Sink_Audio {
    fn play(&mut self, left: &[f32], right: &[f32]) {
        for i in 0..left.len() {
            self.samples.push(left[i]);
            self.samples.push(right[i]);
        }
    }
    fn samples_rate(&self) -> u32 {
        44100
    }
    fn underflowed(&self) -> bool {
        false
    }
}

// A lock-free serial sink: stores each transmitted byte, mirroring gameboy_rs's
// serial.output accumulation.
struct Sink_Serial {
    bytes: Vec<u8>,
}

impl rboy::SerialCallback for Sink_Serial {
    fn call(&mut self, value: u8) -> Option<u8> {
        self.bytes.push(value);
        None
    }
}

fn main() {
    let args: Vec<String> = env::args().collect();
    if args.get(1).map(String::as_str) == Some("sizes") {
        print_sizes();
        return;
    }
    if args.len() < 4 {
        eprintln!("usage: emu_bench <rboy|mine> <rom> <cycles>");
        process::exit(2);
    }
    let engine = args[1].as_str();
    let rom = match args[2].strip_prefix("synth:") {
        Some(kind) => synth_rom(kind),
        None => fs::read(&args[2]).expect("read rom"),
    };
    let budget: u64 = args[3].parse().expect("cycles must be an integer");
    let cgb = rom.get(0x143).is_some_and(|&b| b & 0x80 != 0);
    let fb = match engine {
        "rboy" => run_rboy(rom, budget, cgb),
        "mine" => run_mine(rom, budget, cgb),
        other => {
            eprintln!("unknown engine: {other} (expected rboy|mine)");
            process::exit(2);
        }
    };
    // Printing the hash keeps the run observable (no dead-code elimination) and lets a
    // rboy/mine comparison confirm the two engines still produce identical output.
    println!("{fb:016x}");
}

// Prints the in-memory size of the value-threaded machine structs — the bytes moved
// each `do_cycle` if the compiler does not elide the reconstruction.
fn print_sizes() {
    println!("Cpu   {}", std::mem::size_of::<gameboy_rs::cpu::Cpu>());
    println!("Mmu   {}", std::mem::size_of::<gameboy_rs::mmu::Mmu>());
    println!("Gpu   {}", std::mem::size_of::<gameboy_rs::gpu::Gpu>());
    println!("Sound {}", std::mem::size_of::<gameboy_rs::sound::Sound>());
    println!("Mbc   {}", std::mem::size_of::<gameboy_rs::mbc::Mbc>());
}

// Builds an in-memory MBC0 ROM that loops one operation forever, isolating a single
// cost so the mine/rboy wall-time ratio attributes that slice of the tax. The entry
// jumps to a per-kind setup (run once) followed by a tight body of the op ending in a
// JR back to the body start.
fn synth_rom(kind: &str) -> Vec<u8> {
    let mut rom = vec![0u8; 0x8000];
    rom[0x100] = 0xC3; // JP 0x0150
    rom[0x101] = 0x50;
    rom[0x102] = 0x01;
    rom[0x147] = 0x00; // MBC0, DMG (0x143 stays 0 too)
    let setup: Vec<u8> = match kind {
        "write" => vec![0x21, 0x00, 0xC0, 0x3E, 0x42], // LD HL,0xC000 ; LD A,0x42
        "read" => vec![0x21, 0x00, 0xC0],              // LD HL,0xC000
        "stack" => vec![0x31, 0xFF, 0xDF],             // LD SP,0xDFFF
        _ => Vec::new(),
    };
    let body: Vec<u8> = match kind {
        "nop" => vec![0x00; 120],           // NOP
        "alu" => vec![0x3C; 120],           // INC A
        "write" => vec![0x77; 120],         // LD (HL),A
        "read" => vec![0x7E; 120],          // LD A,(HL)
        "stack" => [0xC5, 0xC1].repeat(60), // PUSH BC ; POP BC
        other => panic!("unknown synth kind: {other}"),
    };
    // The JR loops back to the body start (setup is skipped on repeats).
    let jr_back = -(body.len() as i32 + 2);
    let program: Vec<u8> = [setup, body, vec![0x18, jr_back as u8]].concat();
    for (i, byte) in program.iter().enumerate() {
        rom[0x150 + i] = *byte;
    }
    // Header checksum over 0x134..=0x14C (both engines reject an invalid one).
    rom[0x14D] = (0x134..=0x014Cusize).fold(0u8, |acc, i| acc.wrapping_sub(rom[i]).wrapping_sub(1));
    rom
}

// Drives the original rboy for the tick budget with audio + serial captured.
fn run_rboy(rom: Vec<u8>, budget: u64, cgb: bool) -> u64 {
    let built = match cgb {
        true => Device::new_cgb_from_buffer(rom, false, None),
        false => Device::new_from_buffer(rom, false, None),
    };
    let mut dev = built.expect("rboy load");
    dev.enable_audio(Box::new(Sink_Audio { samples: Vec::new() }), false);
    dev.set_serial_callback(Box::new(Sink_Serial { bytes: Vec::new() }));
    let mut ticks: u64 = 0;
    while ticks < budget {
        ticks += dev.do_cycle() as u64;
    }
    hash::fnv1a(dev.get_gpu_data())
}

// Drives gameboy_rs for the tick budget (a fixed injected clock keeps it deterministic).
fn run_mine(rom: Vec<u8>, budget: u64, cgb: bool) -> u64 {
    let built = match cgb {
        true => device::run_headless_cgb(rom, budget, 0),
        false => device::run_headless(rom, gbmode::Gb_Mode::Classic, budget, 0),
    };
    hash::fnv1a(&built.expect("gameboy_rs run").framebuffer)
}
