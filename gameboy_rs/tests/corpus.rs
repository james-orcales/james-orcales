//! End-to-end regression net: replays every committed test ROM and checks its framebuffer,
//! serial, and audio fingerprints against committed golden hashes. This preserves the coverage
//! the rboy differential gave, without the dependency — the goldens were recorded while
//! gameboy_rs was byte-verified against rboy across the whole corpus, so a mismatch here is a
//! regression from that validated behaviour. Each ROM runs only until its framebuffer and serial
//! stop changing (then a few chunks more to confirm) — a microtest that settles in a few thousand
//! cycles stops almost at once, a slow renderer runs longer — so the cost is proportional, with no
//! flat budget. Run in release — `cargo test -p gameboy_rs --release`. Set `CORPUS_BLESS=1` to
//! regenerate the golden.
//!
//! Written in the crate dialect like everything else here — no `mut`, iteration by fold, state
//! threaded as values — and on the linter's CI path (`gameboy_rs/tests`), so it is enforced, not
//! taken on trust.

use gameboy_rs::cpu;
use gameboy_rs::device;
use gameboy_rs::gbmode;
use gameboy_rs::hash;
use gameboy_rs::mbc;
use gameboy_rs::region;
use std::collections;
use std::env;
use std::fs;
use std::io;
use std::io::Read;
use std::panic;
use std::path;

// Early-stop parameters: step the emulator `CORPUS_CHUNK` T-cycles at a time and stop once the
// (framebuffer, serial) fingerprint holds steady for `CORPUS_PATIENCE` consecutive chunks — past
// that a regression could only hide in state the fingerprint doesn't read anyway. `CORPUS_CAP`
// bounds ROMs whose screen never settles (none committed do) and so also caps the recursion depth.
const CORPUS_CHUNK: u64 = 50_000;
const CORPUS_PATIENCE: u32 = 3;
const CORPUS_CAP: u64 = 8_000_000;

// The 8-MiB cartridge cap, mirroring the composition root's bounded ROM read.
const CORPUS_ROM_CAP: u64 = 8 << 20;

// The permissively-licensed suites committed to the repo (the gitignore whitelist). The fetched,
// license-restricted suites are skipped when absent, so this net runs on a clean checkout.
const CORPUS_SUITES: [&str; 12] = [
    "age-test-roms",
    "bully",
    "cgb-acid-hell",
    "cgb-acid2",
    "dmg-acid2",
    "gbmicrotest",
    "mealybug-tearoom-tests",
    "mooneye-test-suite",
    "rtc3test",
    "same-suite",
    "scribbltests",
    "strikethrough",
];

// The golden fingerprint file, beside this test.
const CORPUS_GOLDEN: &str = concat!(env!("CARGO_MANIFEST_DIR"), "/tests/corpus.golden");

// The test_roms root the suites live under.
fn corpus_root() -> path::PathBuf {
    path::PathBuf::from(concat!(env!("CARGO_MANIFEST_DIR"), "/test_roms"))
}

// Every `.gb`/`.gbc` under a directory, recursively, in a stable order.
fn roms_under(dir: &path::Path) -> collections::BTreeSet<path::PathBuf> {
    match fs::read_dir(dir) {
        Err(_) => collections::BTreeSet::new(),
        Ok(entries) => entries
            .filter_map(Result::ok)
            .flat_map(|entry| {
                let path = entry.path();
                match path.is_dir() {
                    true => roms_under(&path).into_iter().collect::<Vec<_>>(),
                    false => match path.extension().and_then(|extension| extension.to_str()) {
                        Some("gb") | Some("gbc") => vec![path],
                        _ => Vec::new(),
                    },
                }
            })
            .collect(),
    }
}

// The ROM path relative to test_roms — the portable fixture key.
fn corpus_key(rom: &path::Path, root: &path::Path) -> String {
    rom.strip_prefix(root).unwrap_or(rom).to_string_lossy().into_owned()
}

// Reads a file through the bounded streaming path (bulk `fs::read` is blacklisted in the core).
fn read_bounded(path: &path::Path) -> Option<Vec<u8>> {
    let file = fs::File::open(path).ok()?;
    io::BufReader::new(file).take(CORPUS_ROM_CAP).bytes().collect::<io::Result<Vec<u8>>>().ok()
}

// The audio channels flattened to little-endian bytes, matching the composition root's report.
fn audio_bytes(audio: &[f32]) -> Vec<u8> {
    audio.iter().flat_map(|sample| sample.to_le_bytes()).collect()
}

// Builds the machine for a ROM (CGB chosen from the header), clock pinned to 0 for determinism.
fn build_machine(rom: Vec<u8>, cgb: bool) -> Result<cpu::Cpu, String> {
    let cart = mbc::get_mbc(rom, false)?;
    let built = match cgb {
        true => cpu::new_cgb(cart),
        false => cpu::new(cart, gbmode::Gb_Mode::Classic),
    };
    built.map(|machine| cpu::set_clock(machine, 0))
}

// The stability signal: the framebuffer and serial hashes. Audio is excluded — it only ever grows,
// so it can never read "settled" — but it is still captured in the final fingerprint.
fn visual_hash(machine: &cpu::Cpu) -> (u64, u64) {
    (hash::fnv1a(&region::to_vec(&machine.mmu.gpu.data)), hash::fnv1a(&machine.mmu.serial.output))
}

// Steps the machine `CORPUS_CHUNK` cycles at a time until its visual fingerprint holds steady for
// `CORPUS_PATIENCE` chunks (settled) or `CORPUS_CAP` is reached (never settles). Threads the
// machine, the last fingerprint, the steady-chunk run, and the elapsed cycles — no mutation.
fn run_settled(machine: cpu::Cpu, last: (u64, u64), steady: u32, elapsed: u64) -> cpu::Cpu {
    match steady >= CORPUS_PATIENCE || elapsed >= CORPUS_CAP {
        true => machine,
        false => {
            let advanced = device::run(machine, CORPUS_CHUNK);
            let now = visual_hash(&advanced);
            let steady_next = match now == last {
                true => steady + 1,
                false => 0,
            };
            run_settled(advanced, now, steady_next, elapsed + CORPUS_CHUNK)
        }
    }
}

// Replays a ROM until its output settles and returns its (framebuffer, serial, audio) fingerprints,
// or None if it does not run cleanly — the MBC rejects it, or it hits an unimplemented opcode. Such
// ROMs (the illegal-opcode mooneye/gambatte tests both engines refuse) are left out of the goldens
// rather than pinned to a crash.
fn fingerprint(rom_path: &path::Path) -> Option<(u64, u64, u64)> {
    let rom = read_bounded(rom_path)?;
    let cgb = rom.get(0x143).is_some_and(|&byte| byte & 0x80 != 0);
    let run = panic::catch_unwind(move || {
        let machine = build_machine(rom, cgb).ok()?;
        let start = visual_hash(&machine);
        let output = device::output_of(run_settled(machine, start, 0, 0));
        Some((
            hash::fnv1a(&output.framebuffer),
            hash::fnv1a(&output.serial),
            hash::fnv1a(&audio_bytes(&output.audio)),
        ))
    });
    run.ok().flatten()
}

// The cleanly-running ROMs of every committed suite, keyed by portable path, fingerprinted.
fn corpus_fingerprints() -> collections::BTreeMap<String, (u64, u64, u64)> {
    let root = corpus_root();
    CORPUS_SUITES
        .iter()
        .flat_map(|suite| roms_under(&root.join(suite)))
        .filter_map(|rom| fingerprint(&rom).map(|prints| (corpus_key(&rom, &root), prints)))
        .collect()
}

// Renders the golden file: one `key\tfb\tserial\taudio` line per ROM, hex, sorted by key.
fn render_golden(prints: &collections::BTreeMap<String, (u64, u64, u64)>) -> String {
    prints
        .iter()
        .map(|(key, (fb, serial, audio))| format!("{key}\t{fb:016x}\t{serial:016x}\t{audio:016x}\n"))
        .collect()
}

// Parses the golden file back into fingerprints.
fn parse_golden(text: &str) -> collections::BTreeMap<String, (u64, u64, u64)> {
    text.lines()
        .filter_map(|line| match line.split('\t').collect::<Vec<&str>>().as_slice() {
            [key, fb, serial, audio] => {
                Some((key.to_string(), (parse_hex(fb), parse_hex(serial), parse_hex(audio))))
            }
            _ => None,
        })
        .collect()
}

fn parse_hex(text: &str) -> u64 {
    u64::from_str_radix(text, 16).unwrap_or(0)
}

// Reads the golden file through the bounded path as UTF-8.
fn load_golden() -> collections::BTreeMap<String, (u64, u64, u64)> {
    let bytes = read_bounded(path::Path::new(CORPUS_GOLDEN)).expect("golden file present (bless first)");
    parse_golden(&String::from_utf8(bytes).expect("golden is UTF-8"))
}

#[test]
fn corpus_matches_golden() {
    match env::var("CORPUS_BLESS").is_ok() {
        true => {
            // Silence the expected illegal-opcode panics as refused ROMs are probed and excluded.
            panic::set_hook(Box::new(|_| {}));
            let prints = corpus_fingerprints();
            fs::write(CORPUS_GOLDEN, render_golden(&prints)).expect("write golden");
            println!("blessed {} ROMs into {CORPUS_GOLDEN}", prints.len());
        }
        false => {
            let root = corpus_root();
            let regressions: Vec<String> = load_golden()
                .iter()
                .filter_map(|(key, expected)| match root.join(key).exists() {
                    false => None,
                    true => match fingerprint(&root.join(key)) {
                        Some(actual) if actual == *expected => None,
                        Some(actual) => Some(format!("{key}: {actual:?} != golden {expected:?}")),
                        None => Some(format!("{key}: now fails/panics (was clean)")),
                    },
                })
                .collect();
            assert!(
                regressions.is_empty(),
                "{} corpus regression(s):\n{}",
                regressions.len(),
                regressions.join("\n"),
            );
        }
    }
}
