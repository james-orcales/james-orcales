//! The sloc command: the thin composition root over the library, and the one
//! place that binds the real filesystem and shells out to git. Every host
//! binding is a free function passed into `sloc::run`.

use sloc_rs::sloc;
use std::collections;
use std::env;
use std::fs;
use std::io;
use std::io::Read;
use std::process;
use std::thread;

/// Caps a single file read, bounding memory on a pathologically large file.
const MAIN_FILE_BYTES_MAX: u64 = 64 << 20;

/// The counting workers spend most of their time blocked on reads, so the pool
/// is oversubscribed past the core count: four per core keeps the cores busy
/// classifying while others wait on a read.
const MAIN_WORKERS_PER_CORE: usize = 4;

fn main() {
    let arguments: Vec<String> = env::args().collect();
    let parallelism = thread::available_parallelism().map(|count| count.get()).unwrap_or(1);
    let result = sloc::run(
        &arguments,
        parallelism * MAIN_WORKERS_PER_CORE,
        is_directory,
        list_dir,
        read_in,
        read_file,
        ignore_for,
    );
    print!("{}", result.stdout);
    eprint!("{}", result.stderr);
    process::exit(result.code);
}

/// Reports whether a path names a directory, following symlinks like Go's
/// `os.Stat` for an explicitly-named argument.
fn is_directory(path: &str) -> bool {
    fs::metadata(path).map(|information| information.is_dir()).unwrap_or(false)
}

/// Lists a directory's immediate children, lexically sorted, using the entry's
/// own type (an lstat that leaves symlinked directories undescended).
fn list_dir(root: &str, dir: &str) -> Vec<sloc::Walk_Entry> {
    let real = if dir == "." { root.to_string() } else { join(root, dir) };
    let reader = match fs::read_dir(&real) {
        Ok(reader) => reader,
        Err(_) => return Vec::new(),
    };
    // A BTreeMap collected from the entries sorts by name without a `mut` sort.
    let sorted: collections::BTreeMap<String, bool> = reader
        .filter_map(|entry| entry.ok())
        .filter_map(|entry| {
            let name = entry.file_name().into_string().ok()?;
            let is_dir = entry.file_type().map(|kind| kind.is_dir()).unwrap_or(false);
            Some((name, is_dir))
        })
        .collect();
    sorted.into_iter().map(|(name, is_directory)| sloc::Walk_Entry { name, is_directory }).collect()
}

/// Reads a file inside the directory walk, joining the root and the walk-relative
/// path.
fn read_in(root: &str, relative: &str) -> Option<Vec<u8>> {
    read_capped(&join(root, relative))
}

/// Reads an explicitly-named file.
fn read_file(path: &str) -> Option<Vec<u8>> {
    read_capped(path)
}

/// Reads a file, never holding more than the byte cap: a streaming read stopped
/// at `MAIN_FILE_BYTES_MAX` through the allowlisted `Read` adapters (`take` and
/// `bytes`), whose iteration `mut` lives in the desugaring, not here. The bulk
/// `fs::read` is banned as an unbounded read, and it is the only `mut`-free way
/// to slurp raw bytes (`io::read_to_string` would reject non-UTF-8), so every
/// file takes the streaming path.
fn read_capped(path: &str) -> Option<Vec<u8>> {
    let file = fs::File::open(path).ok()?;
    io::BufReader::new(file)
        .take(MAIN_FILE_BYTES_MAX)
        .bytes()
        .collect::<io::Result<Vec<u8>>>()
        .ok()
}

/// Builds the ignore filter for a directory inside a git work tree: a path is
/// ignored when git would not list it. One `git ls-files` lists every kept file
/// under the root; a directory holding no kept file is pruned. Outside a git
/// work tree, or when git is unavailable, the filter is inactive so nothing is
/// ignored.
fn ignore_for(root: &str) -> sloc::Ignore_Set {
    // --cached lists tracked files, --others untracked ones, --exclude-standard
    // drops the gitignored untracked files; together they are exactly the files
    // git does not ignore. -z keeps paths literal and "." scopes the listing.
    let output = match process::Command::new("git")
        .args(["-C", root, "ls-files", "-z", "--cached", "--others", "--exclude-standard", "--", "."])
        .output()
    {
        Ok(result) if result.status.success() => result.stdout,
        _ => return sloc::Ignore_Set::default(),
    };
    let names: Vec<String> = output
        .split(|&byte| byte == 0)
        .filter(|name| !name.is_empty())
        .map(|name| String::from_utf8_lossy(name).into_owned())
        .collect();
    let kept_files: collections::HashSet<String> = names.iter().cloned().collect();
    let kept_directories: collections::HashSet<String> =
        names.iter().flat_map(|name| ancestors(name)).collect();
    sloc::Ignore_Set { active: true, kept_files, kept_directories }
}

/// The directory ancestors of a path, relative to the walked root and excluding
/// the root itself: `a/b/c.go` yields `a` and `a/b`.
fn ancestors(path: &str) -> Vec<String> {
    let parts: Vec<&str> = path.split('/').collect();
    (1..parts.len()).map(|index| parts[..index].join("/")).collect()
}

/// Joins a directory and a relative path, leaving a `.` root off the front.
fn join(left: &str, right: &str) -> String {
    if left == "." {
        right.to_string()
    } else {
        format!("{left}/{right}")
    }
}
