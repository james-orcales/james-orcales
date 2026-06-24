//! Black-box: a function body spanning more than seventy lines (open brace to
//! close brace, inclusive) is banned; a body at exactly the cap passes.

use lint_rs::scan;
use std::path::Path;

#[test]
fn flags_function_over_the_line_cap() {
    let violations = scan(Path::new("tests/fixtures/function_size_dirty"));
    assert_eq!(violations.len(), 1, "got: {violations:?}");
    assert!(violations[0].contains("max 70"), "got: {violations:?}");
}

#[test]
fn allows_function_at_the_line_cap() {
    let violations = scan(Path::new("tests/fixtures/function_size_clean"));
    assert!(violations.is_empty(), "got: {violations:?}");
}
