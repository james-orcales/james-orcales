//! Black-box: the unbounded-API blacklist — flat-named calls whose bounded twin
//! is a different name. Path calls `fs::read`, `fs::read_to_string`,
//! `iter::repeat`, `TcpStream::connect`, and the low-collision methods
//! `recv`/`accept` are banned; the bounded twins (`repeat_n`, `connect_timeout`,
//! `try_recv`) pass, since the match is on the exact name.

use lint_rs::scan;
use std::path::Path;

#[test]
fn flags_unbounded_calls() {
    let violations = scan(Path::new("tests/fixtures/unbounded_dirty"));
    assert_eq!(violations.len(), 6, "got: {violations:?}");
    assert!(violations.iter().all(|m| m.contains("unbounded")), "got: {violations:?}");
}

#[test]
fn allows_bounded_twins() {
    let violations = scan(Path::new("tests/fixtures/unbounded_clean"));
    assert!(violations.is_empty(), "got: {violations:?}");
}
