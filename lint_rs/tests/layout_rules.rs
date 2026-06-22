//! Black-box: inline modules with a body are banned, except `mod tests`.

use lint_rs::scan;
use std::path::Path;

#[test]
fn flags_inline_module_with_body() {
    let violations = scan(Path::new("tests/fixtures/layout_dirty"));
    assert_eq!(violations.len(), 1, "got: {violations:?}");
    assert!(violations[0].contains("inline module"), "got: {violations:?}");
}

#[test]
fn allows_file_modules_and_inline_tests() {
    let violations = scan(Path::new("tests/fixtures/layout_clean"));
    assert!(violations.is_empty(), "got: {violations:?}");
}
