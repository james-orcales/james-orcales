//! Black-box: no `<'a>` may be declared. With references barred from fields and
//! returns, elision covers the rest, so a named lifetime is never needed.

use lint_rs::scan;
use std::path::Path;

#[test]
fn flags_lifetime_parameters() {
    let violations = scan(Path::new("tests/fixtures/lifetime_dirty"));
    assert_eq!(violations.len(), 2, "got: {violations:?}");
    assert!(violations.iter().all(|m| m.contains("lifetime parameter banned")), "got: {violations:?}");
}

#[test]
fn allows_no_lifetime_params() {
    let violations = scan(Path::new("tests/fixtures/lifetime_clean"));
    assert!(violations.is_empty(), "got: {violations:?}");
}
