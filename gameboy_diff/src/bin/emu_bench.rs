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
    if args.len() < 4 {
        eprintln!("usage: emu_bench <rboy|mine> <rom> <cycles>");
        process::exit(2);
    }
    let engine = args[1].as_str();
    let rom = fs::read(&args[2]).expect("read rom");
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
