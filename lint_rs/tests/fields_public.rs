//! Black-box: every struct/union field must be `pub` — no exceptions.

use lint_rs::scan;
use std::path::Path;

#[test]
fn flags_non_pub_named_and_tuple_fields() {
    let violations = scan(Path::new("tests/fixtures/fields_dirty"));
    assert_eq!(violations.len(), 2, "got: {violations:?}");
    assert!(violations.iter().all(|m| m.contains("must be `pub`")), "got: {violations:?}");
}

#[test]
fn allows_all_pub_fields_and_unit_structs() {
    let violations = scan(Path::new("tests/fixtures/fields_clean"));
    assert!(violations.is_empty(), "got: {violations:?}");
}
