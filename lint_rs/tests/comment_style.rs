//! Black-box: a line comment opens with a space then a capital, and ends in
//! `.`, `:`, `?`, or `!`. Doc comments are checked; trailing (inline) comments
//! are exempt from the capital/terminator rules but still need the space; in a
//! group only the first line's opening and the last line's ending are checked;
//! a `//` inside a string literal is not a comment; and lines inside a markdown
//! code fence are exempt from the opening and ending rules.

use lint_rs::scan;
use std::path::Path;

#[test]
fn flags_opening_ending_and_spacing() {
    let violations = scan(Path::new("tests/fixtures/comments_dirty"));
    assert_eq!(violations.len(), 4, "got: {violations:?}");
    assert!(violations.iter().any(|m| m.contains("capital")), "got: {violations:?}");
    assert!(violations.iter().any(|m| m.contains("end with")), "got: {violations:?}");
    assert!(violations.iter().any(|m| m.contains("space after")), "got: {violations:?}");
}

#[test]
fn allows_well_formed_doc_inline_and_grouped_comments() {
    let violations = scan(Path::new("tests/fixtures/comments_clean"));
    assert!(violations.is_empty(), "got: {violations:?}");
}
