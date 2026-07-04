use shared_rs::levenshtein;

#[test]
fn distance_equal_strings_is_zero() {
    assert_eq!(levenshtein::distance("abc", "abc"), 0);
}

#[test]
fn distance_against_empty_is_the_other_length() {
    assert_eq!(levenshtein::distance("", "abc"), 3);
    assert_eq!(levenshtein::distance("abc", ""), 3);
}

#[test]
fn distance_single_insertion_is_one() {
    assert_eq!(levenshtein::distance("ab", "abc"), 1);
}

#[test]
fn distance_single_deletion_is_one() {
    assert_eq!(levenshtein::distance("abc", "ab"), 1);
}

#[test]
fn distance_single_substitution_is_one() {
    assert_eq!(levenshtein::distance("cat", "car"), 1);
}

#[test]
fn distance_priorty_priority_is_one() {
    assert_eq!(levenshtein::distance("priorty", "priority"), 1);
}

#[test]
fn distance_kitten_sitting_is_three() {
    assert_eq!(levenshtein::distance("kitten", "sitting"), 3);
}

#[test]
fn distance_is_symmetric() {
    assert_eq!(levenshtein::distance("sitting", "kitten"), 3);
}

#[test]
fn closest_near_miss_matches() {
    let commands: Vec<String> =
        ["help", "add", "list", "delete"].iter().map(|s| s.to_string()).collect();
    let closest = levenshtein::closest("lst", &commands);
    assert_eq!(closest, Some("list".to_string()));
}

#[test]
fn closest_wild_miss_finds_nothing() {
    let commands: Vec<String> =
        ["help", "add", "list", "delete"].iter().map(|s| s.to_string()).collect();
    assert_eq!(levenshtein::closest("zzzzzzzz", &commands), None);
}

#[test]
fn closest_empty_candidates_finds_nothing() {
    assert_eq!(levenshtein::closest("anything", &[]), None);
}

#[test]
fn closest_tie_keeps_the_earliest_candidate() {
    let candidates = vec!["abce".to_string(), "abcf".to_string()];
    assert_eq!(levenshtein::closest("abcd", &candidates), Some("abce".to_string()));
}
