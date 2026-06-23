//! Black-box: a struct/enum/union field may not hold a reference (directly or
//! nested). Own the value or store an integer handle.

use lint_rs::scan;
use std::path::Path;

#[test]
fn flags_reference_typed_fields() {
    let violations = scan(Path::new("tests/fixtures/ref_fields_dirty"));
    assert_eq!(violations.len(), 2, "got: {violations:?}");
    assert!(violations.iter().all(|m| m.contains("reference field banned")), "got: {violations:?}");
}

#[test]
fn allows_owned_and_handle_fields() {
    let violations = scan(Path::new("tests/fixtures/ref_fields_clean"));
    assert!(violations.is_empty(), "got: {violations:?}");
}
