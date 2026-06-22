//! Black-box: every `use` binds a module; type/trait item imports, globs,
//! grouped braces, and denylisted std free functions are banned. Allowlisted
//! std traits and `pub use` re-exports are allowed.

use lint_rs::scan;
use std::path::Path;

#[test]
fn flags_item_glob_grouped_and_denylisted_imports() {
    let violations = scan(Path::new("tests/fixtures/import_dirty"));
    assert_eq!(violations.len(), 4, "got: {violations:?}");
}

#[test]
fn allows_module_imports_allowlisted_traits_and_pub_use() {
    let violations = scan(Path::new("tests/fixtures/import_clean"));
    assert!(violations.is_empty(), "got: {violations:?}");
}
