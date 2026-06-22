//! Black-box: items declared inside an *expression* block — an `if` body, a
//! `match` arm, a closure body — must be checked like any other item, not slip
//! through because the traversal only followed `Stmt::Item`.

use lint_rs::scan;
use std::path::Path;

#[test]
fn finds_items_nested_in_expression_blocks() {
    let violations = scan(Path::new("tests/fixtures/nested_items_dirty"));
    // One non-pub field hidden in an `if`, a `match` arm, and a closure body.
    assert_eq!(violations.len(), 3, "got: {violations:?}");
    assert!(violations.iter().all(|m| m.contains("must be `pub`")), "got: {violations:?}");
}
