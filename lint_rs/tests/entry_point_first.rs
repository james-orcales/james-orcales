//! Black-box: when a file declares `fn main`, it must be the first function;
//! non-function items (a struct, a const) may precede it.

use lint_rs::scan;
use std::path::Path;

#[test]
fn flags_main_after_another_function() {
    let violations = scan(Path::new("tests/fixtures/entry_point_dirty"));
    assert_eq!(violations.len(), 1, "got: {violations:?}");
    assert!(violations[0].contains("first function"), "got: {violations:?}");
}

#[test]
fn allows_main_first_after_non_function_items() {
    let violations = scan(Path::new("tests/fixtures/entry_point_clean"));
    assert!(violations.is_empty(), "got: {violations:?}");
}
