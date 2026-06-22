//! Black-box tests: exercise `scan`'s public contract against committed
//! fixtures. Files under `tests/fixtures/` are not compiled by cargo, so the
//! `mut` they contain is inert sample text, not code under test.

use lint_rs::scan;
use std::path::Path;

#[test]
fn flags_every_mut_token_in_source() {
    let violations = scan(Path::new("tests/fixtures/dirty"));
    // Fixture has one `let mut` and one `&mut` — two tokens.
    assert_eq!(violations.len(), 2, "expected two `mut` tokens, got: {violations:?}");
    assert!(violations.iter().all(|v| v.contains("`mut` is banned")));
}

#[test]
fn ignores_mut_in_comments_strings_and_identifiers() {
    // `mut` in a comment, inside a string literal, and as a substring of the
    // identifier `immutable` must all be left alone — only the bare keyword counts.
    let violations = scan(Path::new("tests/fixtures/clean"));
    assert!(violations.is_empty(), "false positives: {violations:?}");
}
