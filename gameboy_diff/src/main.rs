// The differential harness: run the original rboy and gameboy_rs on the same ROM for
// the same tick budget, then compare their framebuffer hash and serial output. This
// crate is the measuring instrument, not the artifact under test, so it uses the
// mutable rboy API, Arc/Mutex serial capture, and fs::read freely.

use gameboy_rs::device;
use gameboy_rs::gbmode;
use gameboy_rs::hash;
use rboy::device::Device;
use std::env;
use std::fs;
use std::process;
use std::sync::Arc;
use std::sync::Mutex;

// A serial sink that records every transmitted byte. Returning None leaves SB
// unchanged and raises no serial interrupt, matching gameboy_rs's serial output.
struct Capture {
    bytes: Arc<Mutex<Vec<u8>>>,
}

impl rboy::SerialCallback for Capture {
    fn call(&mut self, value: u8) -> Option<u8> {
        self.bytes.lock().unwrap().push(value);
        None
    }
}

// A capturing audio sink so rboy's APU runs headless and its synthesized samples are
// recorded (interleaved left/right), to be diffed against gameboy_rs's audio.
struct Capture_Audio {
    samples: Arc<Mutex<Vec<f32>>>,
}

impl rboy::AudioPlayer for Capture_Audio {
    fn play(&mut self, left: &[f32], right: &[f32]) {
        let mut buf = self.samples.lock().unwrap();
        for i in 0..left.len() {
            buf.push(left[i]);
            buf.push(right[i]);
        }
    }
    fn samples_rate(&self) -> u32 {
        44100
    }
    fn underflowed(&self) -> bool {
        false
    }
}

// The FNV-1a hash of an f32 sample stream (over its little-endian bytes).
fn audio_hash(samples: &[f32]) -> u64 {
    let bytes: Vec<u8> = samples.iter().flat_map(|s| s.to_le_bytes()).collect();
    hash::fnv1a(&bytes)
}

fn main() {
    let args: Vec<String> = env::args().collect();
    let path = match args.get(1) {
        Some(p) => p,
        None => {
            eprintln!("usage: gameboy_diff <rom> [cycles]");
            process::exit(2);
        }
    };
    let budget: u64 = args.get(2).and_then(|s| s.parse().ok()).unwrap_or(250_000_000);
    let rom = fs::read(path).expect("read rom");
    process::exit(diff(&path, rom, budget));
}

// Drives the original rboy headlessly, returning (framebuffer hash, serial bytes,
// audio hash).
fn run_rboy(rom: Vec<u8>, budget: u64, cgb: bool) -> (u64, Vec<u8>, u64) {
    let built = match cgb {
        true => Device::new_cgb_from_buffer(rom, false, None),
        false => Device::new_from_buffer(rom, false, None),
    };
    let mut dev = built.expect("rboy load");
    let audio = Arc::new(Mutex::new(Vec::new()));
    dev.enable_audio(Box::new(Capture_Audio { samples: audio.clone() }), false);
    let serial = Arc::new(Mutex::new(Vec::new()));
    dev.set_serial_callback(Box::new(Capture { bytes: serial.clone() }));
    let mut ticks: u64 = 0;
    while ticks < budget {
        ticks += dev.do_cycle() as u64;
    }
    let fb = hash::fnv1a(dev.get_gpu_data());
    let bytes = serial.lock().unwrap().clone();
    let sound = audio_hash(&audio.lock().unwrap());
    (fb, bytes, sound)
}

// Compares rboy and gameboy_rs on one ROM; returns 0 on a full match.
fn diff(name: &str, rom: Vec<u8>, budget: u64) -> i32 {
    // The CGB flag (header byte 0x143 bit 7) selects Color hardware on both sides.
    let cgb = rom.get(0x143).is_some_and(|&b| b & 0x80 != 0);
    let (rboy_fb, rboy_serial, rboy_audio) = run_rboy(rom.clone(), budget, cgb);
    // A fixed injected clock keeps gameboy_rs deterministic; RTC-reading ROMs then
    // diverge from rboy's wall clock (none are in the corpus) but the run reproduces.
    let built = match cgb {
        true => device::run_headless_cgb(rom, budget, 0),
        false => device::run_headless(rom, gbmode::Gb_Mode::Classic, budget, 0),
    };
    let mine = built.expect("gameboy_rs run");
    let my_fb = hash::fnv1a(&mine.framebuffer);
    let my_audio = audio_hash(&mine.audio);
    let fb_ok = rboy_fb == my_fb;
    let serial_ok = rboy_serial == mine.serial;
    let audio_ok = rboy_audio == my_audio;
    println!("{name}");
    println!("  framebuffer  rboy={rboy_fb:016x}  mine={my_fb:016x}  {}", verdict(fb_ok));
    println!(
        "  serial       rboy={}B  mine={}B  {}",
        rboy_serial.len(),
        mine.serial.len(),
        verdict(serial_ok)
    );
    println!("  audio        rboy={rboy_audio:016x}  mine={my_audio:016x}  {}", verdict(audio_ok));
    if !serial_ok {
        println!("    rboy: {:?}", String::from_utf8_lossy(&rboy_serial));
        println!("    mine: {:?}", String::from_utf8_lossy(&mine.serial));
    }
    match fb_ok && serial_ok && audio_ok {
        true => 0,
        false => 1,
    }
}

fn verdict(ok: bool) -> &'static str {
    match ok {
        true => "MATCH",
        false => "DIFFER",
    }
}
