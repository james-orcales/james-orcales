//! Black-box: `mut` is exempt only inside one exact function in one exact file
//! (path suffix + name). The same function name at any other path is NOT exempt
//! — the whitelist pins a specific function in the universe, not a name.

use lint_rs::scan;
use std::path::Path;

#[test]
fn exempts_mut_only_at_the_exact_path_and_name() {
    // `insert` at `.../handle/src/plain.rs` is the whitelisted function: exempt.
    let exact = scan(Path::new("tests/fixtures/mut_whitelist_ok"));
    assert!(exact.is_empty(), "whitelisted insert should be exempt, got: {exact:?}");

    // `insert` at any other path is NOT exempt — name alone is insufficient.
    let elsewhere = scan(Path::new("tests/fixtures/mut_whitelist_bad"));
    assert_eq!(elsewhere.len(), 1, "got: {elsewhere:?}");
    assert!(elsewhere[0].contains("`mut` is banned"), "got: {elsewhere:?}");
}
