//! The headless composition root: the one place that binds real effects. `run` reads the ROM
//! and its battery save from the filesystem, injects the wall clock, emulates the pure core,
//! and writes the battery save and save-state back; `fetch` shells out to curl/shasum/unzip to
//! pull the license-restricted test-ROM suites. Every effect is bound here; the emulator core
//! reads no clock and touches no file. `fs::write`, `SystemTime`, and `process::Command` are
//! dialect-legal (free functions / method chains, no `&mut` tokens), so the boundary stays in
//! the linted crate; argument parsing uses the repo's `shared_rs::cli`.

use gameboy_emulator::cpu;
use gameboy_emulator::device;
use gameboy_emulator::gbmode;
use gameboy_emulator::hash;
use gameboy_emulator::mbc;
use gameboy_emulator::state;
use shared_rs::cli;
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
    let program = program();
    let arguments: Vec<String> = env::args().collect();
    // With no subcommand, cli would default to `run` and fail on the absent ROM; show help.
    if arguments.len() < 2 {
        eprint!("{}", cli::print_help(&program));
        process::exit(2);
    }
    match cli::program_parse(&program, &arguments) {
        Ok(outcome) => process::exit(dispatch(&outcome)),
        Err(error) => {
            eprintln!("{}", cli::parse_error_message(&error));
            eprint!("{}", cli::print_help(&program));
            process::exit(2);
        }
    }
}

// The two subcommands: `run` emulates a ROM; `fetch` downloads the license-restricted suites.
fn program() -> cli::Program {
    cli::new(
        "gameboy_emulator",
        "a headless Game Boy emulator with a test-ROM fetcher",
        vec![
            cli::Command {
                label: "run".to_string(),
                description: "run a ROM headless and print its serial, framebuffer, and audio fingerprints"
                    .to_string(),
                arguments: vec![cli::new_string_argument("rom", "path to the ROM image")],
                flags: vec![cli::new_int_flag(
                    "cycles",
                    "T-cycles to run before reporting",
                    MAIN_DEFAULT_CYCLES as i64,
                )],
            },
            cli::Command {
                label: "fetch".to_string(),
                description: "download the license-restricted test-ROM suites into test_roms/".to_string(),
                arguments: vec![],
                flags: vec![],
            },
        ],
        vec![],
    )
}

// Runs the parsed subcommand and returns its process exit code.
fn dispatch(outcome: &cli::Parse_Outcome) -> i32 {
    match outcome.command.label.as_str() {
        "run" => run(
            &cli::get_option(&outcome.command.arguments, "rom", string_value),
            cli::get_option(&outcome.command.flags, "cycles", int_value) as u64,
        ),
        "fetch" => fetch_test_roms(),
        other => unreachable!("program_parse yielded an undeclared command {other:?}"),
    }
}

// Reads a string parameter's value out of `cli::get_option`'s visited parameter.
fn string_value(parameter: &cli::Parameter) -> String {
    match &parameter.value {
        cli::Parameter_Value::Str(text) => text.clone(),
        _ => String::new(),
    }
}

// Reads an integer parameter's value out of `cli::get_option`'s visited parameter.
fn int_value(parameter: &cli::Parameter) -> i64 {
    match parameter.value {
        cli::Parameter_Value::Int(number) => number,
        _ => 0,
    }
}

// The pinned c-sp game-boy-test-roms v7.0 release and its zip's SHA-256. Fail-closed: if the
// asset is ever re-cut, the checksum stops matching and `fetch` aborts until this is bumped.
const FETCH_URL: &str = "https://github.com/c-sp/gameboy-test-roms/releases/download/v7.0/game-boy-test-roms-v7.0.zip";
const FETCH_SHA256: &str = "b9a9d7a1075aa35a3d07c07c34974048672d8520dca9e07a50178f5860c3832c";

// The six suites kept out of version control — blargg/mbc3-tester/turtle-tests/little-things-gb
// carry no license, gambatte/mooneye-test-suite-wilbertpol are GPL — extracted into this dir.
const FETCH_SUITES: &str = "blargg mbc3-tester turtle-tests little-things-gb gambatte mooneye-test-suite-wilbertpol";
const FETCH_DEST: &str = concat!(env!("CARGO_MANIFEST_DIR"), "/test_roms");

// Download, verify, and extract the six suites. `$1..$4` arrive as argv (not interpolated) so no
// shell quoting is needed; `set -e` plus the `shasum -c` gate abort before anything reaches
// test_roms/, and the scratch dir is removed on every exit.
const FETCH_SCRIPT: &str = "\
set -e
url=\"$1\"; sha=\"$2\"; dest=\"$3\"; suites=\"$4\"
work=\"$(mktemp -d)\"; trap 'rm -rf \"$work\"' EXIT
curl --proto '=https' --tlsv1.2 -fsSL -o \"$work/roms.zip\" \"$url\"
echo \"$sha  $work/roms.zip\" | shasum -a 256 -c -
unzip -q \"$work/roms.zip\" -d \"$work/extract\"
for suite in $suites; do rm -rf \"$dest/$suite\"; cp -R \"$work/extract/$suite\" \"$dest/$suite\"; done
";

// Fetches the license-restricted test-ROM suites. Shells out because no HTTP/zip/hash crate is
// vendored; the method-chained `Command` introduces no `mut` token (as linted `sloc_rs` does).
fn fetch_test_roms() -> i32 {
    let outcome = process::Command::new("sh")
        .arg("-c")
        .arg(FETCH_SCRIPT)
        .arg("gameboy_emulator-fetch")
        .arg(FETCH_URL)
        .arg(FETCH_SHA256)
        .arg(FETCH_DEST)
        .arg(FETCH_SUITES)
        .status();
    match outcome {
        Ok(status) if status.success() => {
            println!("fetched the restricted suites into {FETCH_DEST}");
            0
        }
        Ok(status) => {
            eprintln!("fetch failed (checksum mismatch, or missing curl/unzip/shasum)");
            status.code().unwrap_or(1)
        }
        Err(error) => {
            eprintln!("could not launch the fetch shell: {error}");
            1
        }
    }
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
