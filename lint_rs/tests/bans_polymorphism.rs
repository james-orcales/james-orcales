//! Black-box: inherent impls and traits-with-methods are banned; trait impls,
//! marker traits, and derives are allowed.

use lint_rs::scan;
use std::path::Path;

#[test]
fn flags_inherent_impl_and_trait_with_methods() {
    let violations = scan(Path::new("tests/fixtures/poly_dirty"));
    assert_eq!(violations.len(), 2, "got: {violations:?}");
    assert!(violations.iter().any(|m| m.contains("inherent impl")), "got: {violations:?}");
    assert!(violations.iter().any(|m| m.contains("trait with methods")), "got: {violations:?}");
}

#[test]
fn allows_trait_impls_marker_traits_and_derives() {
    let violations = scan(Path::new("tests/fixtures/poly_clean"));
    assert!(violations.is_empty(), "got: {violations:?}");
}
