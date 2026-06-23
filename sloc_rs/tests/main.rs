use sloc_rs::sloc;

/// The in-memory disk the run is driven against; any root maps to it, mirroring
/// a re-rooting `Open`.
fn disk() -> sloc::File_Tree {
    sloc::File_Tree {
        files: vec![sloc::Tree_File { path: "main.go".to_string(), data: b"package main\n".to_vec() }],
    }
}

/// Runs the command against the in-memory disk and returns stdout, asserting a
/// success exit.
fn run_args(arguments: &[&str], tree: &sloc::File_Tree) -> String {
    let owned: Vec<String> = arguments.iter().map(|argument| argument.to_string()).collect();
    let result = sloc::run(
        &owned,
        1,
        |_path| true,
        |_root, dir| sloc::tree_list_dir(tree, dir),
        |_root, path| sloc::tree_read_file(tree, path),
        |_path| None,
        |_root| sloc::Ignore_Set::default(),
    );
    assert_eq!(result.code, 0, "stderr: {}", result.stderr);
    result.stdout
}

#[test]
fn counts_positional_path() {
    let tree = disk();
    assert!(run_args(&["sloc", "src"], &tree).contains("Go"), "expected Go for a positional path");
}

#[test]
fn counts_default_path() {
    let tree = disk();
    assert!(run_args(&["sloc"], &tree).contains("Go"), "expected Go for the default path");
}
