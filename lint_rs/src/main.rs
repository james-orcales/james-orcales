//! `lint_rs` CLI. Scans a directory (default `src`, or the first argument) and
//! exits non-zero if any rule fires, so it drops into CI or a pre-commit hook.

use std::path;
use std::process;

fn main() -> process::ExitCode {
    let root = std::env::args()
        .nth(1)
        .map(path::PathBuf::from)
        .unwrap_or_else(|| path::PathBuf::from("src"));

    let violations = lint_rs::scan(&root);
    violations.iter().for_each(|violation| println!("{violation}"));

    match violations.is_empty() {
        true => process::ExitCode::SUCCESS,
        false => process::ExitCode::FAILURE,
    }
}
