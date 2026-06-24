//! Black-box: a path may not inline a crate root at the call site. `std`/`core`/
//! `alloc`/`crate`/`super` and every dependency crate (from Cargo.toml) must be
//! bound with a `use` and referenced by the short name — Go's one-import-per-
//! package rule. `syn` is allowed only after `use syn;`; `std::iter::once` must
//! become `use std::iter;` then `iter::once`.

use lint_rs::scan;
use std::path::Path;

#[test]
fn flags_inlined_crate_root_paths() {
    let violations = scan(Path::new("tests/fixtures/imports_inline_dirty"));
    assert_eq!(violations.len(), 3, "got: {violations:?}");
    assert!(violations.iter().all(|m| m.contains("inlined import")), "got: {violations:?}");
}

#[test]
fn allows_paths_qualified_through_a_use() {
    let violations = scan(Path::new("tests/fixtures/imports_inline_clean"));
    assert!(violations.is_empty(), "got: {violations:?}");
}
