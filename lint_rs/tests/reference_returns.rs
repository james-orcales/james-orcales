//! Black-box: a function may not return a reference. Borrows flow in, never out
//! — return an owned value, or read via a visitor closure.

use lint_rs::scan;
use std::path::Path;

#[test]
fn flags_reference_return_types() {
    let violations = scan(Path::new("tests/fixtures/ref_returns_dirty"));
    assert_eq!(violations.len(), 2, "got: {violations:?}");
    assert!(violations.iter().all(|m| m.contains("reference return banned")), "got: {violations:?}");
}

#[test]
fn allows_owned_returns_and_visitor_reads() {
    let violations = scan(Path::new("tests/fixtures/ref_returns_clean"));
    assert!(violations.is_empty(), "got: {violations:?}");
}
