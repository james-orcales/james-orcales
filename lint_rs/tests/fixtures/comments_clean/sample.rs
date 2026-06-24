//! Module doc comment.

/// Doc comment on a function.
pub fn documented() {
    let url = "http://example.com/a//b";
    let _ = url;
}

// First line of a group
// middle line, unchecked
// last line of the group.
pub fn grouped() {
    let value = 0; // inline note, exempt
    let _ = value;
}

/// Example:
/// ```
/// let value = documented();
/// ```
pub fn fenced() {
    let value = 0;
    let _ = value;
}
