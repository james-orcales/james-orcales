//! Black-box: first-char-driven casing (uppercase→Ada_Case, lowercase→
//! snake_case), with generics forced to Ada_Case and lifetimes to snake_case.
//! Test code is checked too.

use lint_rs::scan;
use std::path::Path;

#[test]
fn flags_miscased_identifiers_across_all_categories() {
    let violations = scan(Path::new("tests/fixtures/casing_dirty"));
    assert_eq!(violations.len(), 7, "got: {violations:?}");
    assert!(violations.iter().any(|m| m.contains("Ada_Case")), "got: {violations:?}");
    assert!(violations.iter().any(|m| m.contains("snake_case")), "got: {violations:?}");
}

#[test]
fn allows_correctly_cased_identifiers_including_tests_module() {
    let violations = scan(Path::new("tests/fixtures/casing_clean"));
    assert!(violations.is_empty(), "got: {violations:?}");
}
