//! How many single-character edits separate two strings, and, on top of that,
//! which candidate sits nearest a mistyped one — the "did you mean"
//! suggestion behind a command-line parser or any other name lookup.
//!
//! ```
//! use shared_rs::levenshtein;
//! assert_eq!(levenshtein::distance("kitten", "sitting"), 3);
//! let commands = vec!["help".to_string(), "add".to_string(), "list".to_string()];
//! assert_eq!(levenshtein::closest("lst", &commands), Some("list".to_string()));
//! ```

use std::iter;

/// The Levenshtein edit distance: the fewest single-character insertions,
/// deletions, or substitutions that turn `from` into `to`. Symmetric in its
/// two arguments.
///
/// Rolls two rows rather than a full `len(from) x len(to)` table, so memory
/// stays `O(len(to))`. Each row is built with `fold` instead of index
/// assignment: the dialect bans `mut`, so a row is grown one cell at a time
/// via `chain(once(cell))` rather than written in place.
pub fn distance(from: &str, to: &str) -> usize {
    let from_chars: Vec<char> = from.chars().collect();
    let to_chars: Vec<char> = to.chars().collect();
    let first_row: Vec<usize> = (0..=to_chars.len()).collect();
    let final_row = from_chars.iter().enumerate().fold(first_row, |previous_row, (index, &from_char)| {
        distance_row(&previous_row, &to_chars, from_char, index + 1)
    });
    final_row[to_chars.len()]
}

/// One rolled row of the edit-distance table: `row_start` is the row's first
/// cell (the cost of deleting the whole `from` prefix seen so far), and each
/// following cell is folded from the previous row and the cells already built
/// in this one.
fn distance_row(
    previous_row: &[usize], to_chars: &[char], from_char: char, row_start: usize,
) -> Vec<usize> {
    to_chars.iter().enumerate().fold(vec![row_start], |current_row, (index, &to_char)| {
        let substitution_cost = usize::from(from_char != to_char);
        let delete_cost = previous_row[index + 1] + 1;
        let insert_cost = current_row[index] + 1;
        let substitute_cost = previous_row[index] + substitution_cost;
        let cell = delete_cost.min(insert_cost).min(substitute_cost);
        current_row.into_iter().chain(iter::once(cell)).collect()
    })
}

/// The candidate in `candidates` nearest `target` by edit distance, or `None`
/// when none is close enough to be a likely typo. The threshold scales with
/// length, so a short string demands a near-exact match while a longer one
/// tolerates proportionally more. On a tie the earliest candidate wins.
pub fn closest(target: &str, candidates: &[String]) -> Option<String> {
    let best = candidates.iter().fold(None, |best, candidate| closest_step(target, candidate, best));
    best.map(|(candidate, _distance)| candidate)
}

/// Folds one candidate into the running best match: replaces it only when
/// `candidate` is within threshold AND strictly nearer than the current best,
/// so an equal-distance later candidate leaves the earlier one in place.
fn closest_step(target: &str, candidate: &str, best: Option<(String, usize)>) -> Option<(String, usize)> {
    let candidate_distance = distance(target, candidate);
    let threshold = target.chars().count().max(candidate.chars().count()) / 3;
    match (candidate_distance > threshold, &best) {
        (true, _) => best,
        (false, Some((_, best_distance))) if candidate_distance >= *best_distance => best,
        (false, _) => Some((candidate.to_string(), candidate_distance)),
    }
}
