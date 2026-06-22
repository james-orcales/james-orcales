//! Black-box: a file syn cannot parse must surface as a diagnostic, not be
//! silently skipped (the old token scanner dropped unparseable files).

use lint_rs::scan;
use std::path::Path;

#[test]
fn reports_unparseable_file_instead_of_skipping() {
    let violations = scan(Path::new("tests/fixtures/parse_error_dirty"));
    assert_eq!(violations.len(), 1, "expected one parse-error diagnostic, got: {violations:?}");
    assert!(violations[0].contains("parse error"), "got: {violations:?}");
}
