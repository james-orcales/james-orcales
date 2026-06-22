//! Black-box: references hidden in non-angle-bracket type positions must still be
//! caught — inside a parenthesized `Fn(&T)`, an associated-type binding
//! `Item = &T`, and a bare `fn(&A) -> &B` type.

use lint_rs::scan;
use std::path::Path;

#[test]
fn finds_references_hidden_in_type_positions() {
    let violations = scan(Path::new("tests/fixtures/ref_hidden_dirty"));
    // callback field (Fn(&u8)), func field (fn(&u8)), make return (Item = &u8).
    assert_eq!(violations.len(), 3, "got: {violations:?}");
}
