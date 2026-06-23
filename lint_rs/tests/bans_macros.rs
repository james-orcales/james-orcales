//! Black-box: authoring macros is banned (`macro_rules!`, decl-macro 2.0,
//! proc-macro fns). Invoking macros (`println!`, `#[derive]`) is allowed.

use lint_rs::scan;
use std::path::Path;

#[test]
fn flags_macro_definitions() {
    let violations = scan(Path::new("tests/fixtures/macros_dirty"));
    assert_eq!(violations.len(), 2, "got: {violations:?}");
    assert!(violations.iter().all(|m| m.contains("macro")), "got: {violations:?}");
}

#[test]
fn allows_macro_invocations_and_derives() {
    let violations = scan(Path::new("tests/fixtures/macros_clean"));
    assert!(violations.is_empty(), "got: {violations:?}");
}
