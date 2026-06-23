use sloc_rs::sloc;
use std::collections;

// A classify case pins the exact code, comment, and blank tally a source must
// produce: (name, source, code, comment, blank).
type Classify_Case = (&'static str, &'static str, usize, usize, usize);

/// Checks each case's exact line partition.
fn run_classify_cases(language: &sloc::Language, cases: &[Classify_Case]) {
    for (name, source, code, comment, blank) in cases {
        let counts = sloc::classify_file(sloc::Classify_File_Input {
            source: source.as_bytes().to_vec(),
            language: language.clone(),
        });
        let want = sloc::Counts { code: *code, comment: *comment, blank: *blank };
        assert_eq!(counts, want, "{name}: source {source:?}");
    }
}

/// Checks that each key resolves through `lookup` to the expected language name.
fn check_detection(cases: &[(String, String)], lookup: impl Fn(&str) -> Option<sloc::Language>) {
    for (key, want) in cases {
        match lookup(key) {
            Some(language) => assert_eq!(language.name, *want, "{key}"),
            None => panic!("{key} not recognized"),
        }
    }
}

/// Owns a table of (key, expected-name) string pairs.
fn owned_pairs(pairs: &[(&str, &str)]) -> Vec<(String, String)> {
    pairs.iter().map(|(key, name)| (key.to_string(), name.to_string())).collect()
}

/// Counts an in-memory tree with nothing ignored.
fn count_tree(tree: &sloc::File_Tree, include_hidden: bool, concurrency: usize) -> sloc::Report {
    sloc::count(
        &sloc::Count_Input { root: ".".to_string(), include_hidden, concurrency },
        |dir| sloc::tree_list_dir(tree, dir),
        |path| sloc::tree_read_file(tree, path),
        |_path, _is_dir| false,
    )
}

/// Builds one in-memory tree file.
fn tree_file(path: &str, data: &str) -> sloc::Tree_File {
    sloc::Tree_File { path: path.to_string(), data: data.as_bytes().to_vec() }
}

/// Indexes a report's files by path for order-independent assertions.
fn report_by_path(report: &sloc::Report) -> collections::HashMap<String, sloc::File_Count> {
    report.files.iter().map(|file| (file.path.clone(), file.clone())).collect()
}

#[test]
fn classify_comments() {
    run_classify_cases(&sloc::language_go(), &[
        ("go line comment", "// hi\n", 0, 1, 0),
        ("go code", "package main\n", 1, 0, 0),
        ("go trailing comment is code", "total++ // bump\n", 1, 0, 0),
        ("go doc comment", "/// doc\n", 0, 1, 0),
        ("go code, blank, comment", "a()\n\n// c\n", 1, 1, 1),
    ]);
    run_classify_cases(&sloc::language_python(), &[
        ("python hash comment", "# hi\n", 0, 1, 0),
        ("python code", "x = 1\n", 1, 0, 0),
        ("python trailing comment is code", "x = 1  # set\n", 1, 0, 0),
    ]);
    run_classify_cases(&sloc::language_java_script(), &[("javascript line comment", "// hi\n", 0, 1, 0)]);
    run_classify_cases(&sloc::language_type_script(), &[
        ("typescript line comment", "// t\n", 0, 1, 0),
        ("typescript type annotation is code", "let x: number = 1;\n", 1, 0, 0),
    ]);
    run_classify_cases(&sloc::language_lua(), &[
        ("lua line comment", "-- hi\n", 0, 1, 0),
        ("lua code", "local x = 1\n", 1, 0, 0),
    ]);
    run_classify_cases(&sloc::language_odin(), &[("odin line comment", "// hi\n", 0, 1, 0)]);
    run_classify_cases(&sloc::language_zig(), &[
        ("zig line comment", "// hi\n", 0, 1, 0),
        ("zig doc comment", "/// doc\n", 0, 1, 0),
    ]);
    run_classify_cases(&sloc::language_c(), &[
        ("c line comment", "// hi\n", 0, 1, 0),
        ("c block comment", "/* c */\n", 0, 1, 0),
    ]);
    run_classify_cases(&sloc::language_shell(), &[
        ("shell comment", "# hi\n", 0, 1, 0),
        ("shell hash in string is code", "echo \"a # b\"\n", 1, 0, 0),
    ]);
    run_classify_cases(&sloc::language_sql(), &[
        ("sql line comment", "-- hi\n", 0, 1, 0),
        ("sql block comment", "/* c */\n", 0, 1, 0),
    ]);
    run_classify_cases(&sloc::language_php(), &[
        ("php slash comment", "// hi\n", 0, 1, 0),
        ("php hash comment", "# hi\n", 0, 1, 0),
    ]);
    run_classify_cases(&sloc::language_erlang(), &[("erlang comment", "% hi\n", 0, 1, 0)]);
    run_classify_cases(&sloc::language_common_lisp(), &[("lisp line comment", "; hi\n", 0, 1, 0)]);
    run_classify_cases(&sloc::language_cmake(), &[("cmake comment", "# hi\n", 0, 1, 0)]);
    run_classify_cases(&sloc::language_fish(), &[
        ("fish comment", "# hi\n", 0, 1, 0),
        ("fish hash in string is code", "echo \"a # b\"\n", 1, 0, 0),
    ]);
    run_classify_cases(&sloc::language_nushell(), &[("nushell comment", "# hi\n", 0, 1, 0)]);
}

#[test]
fn classify_strings() {
    run_classify_cases(&sloc::language_go(), &[
        ("line-comment token in a string", "x := \"http://foo\"\n", 1, 0, 0),
        ("block-comment token in a string", "s := \"/* not a comment */\"\n", 1, 0, 0),
        ("escaped quote in a string", "s := \"she said \\\"hi\\\"\"\n", 1, 0, 0),
    ]);
    run_classify_cases(&sloc::language_python(), &[("hash in a string is code", "s = \"a # b\"\n", 1, 0, 0)]);
    run_classify_cases(&sloc::language_java_script(), &[("slashes in a string", "let u = \"http://x\";\n", 1, 0, 0)]);
}

#[test]
fn classify_block_comments() {
    run_classify_cases(&sloc::language_go(), &[
        ("single line block", "/* c */\n", 0, 1, 0),
        ("block spanning lines", "/* a\nb\n*/\n", 0, 3, 0),
        ("code after close", "/* a */ x()\n", 1, 0, 0),
        ("code before open, spanning", "x() /* open\nstill\n*/ y()\n", 2, 1, 0),
        ("go does not nest", "/* a /* b */ c */\n", 1, 0, 0),
        ("blank line inside block is blank", "/*\n   \n*/\n", 0, 2, 1),
    ]);
    run_classify_cases(&sloc::language_rust(), &[
        ("nested on one line", "/* a /* b */ c */\n", 0, 1, 0),
        ("nested across lines then code", "/* a /* b\nstill */ c */ x()\n", 1, 1, 0),
    ]);
    run_classify_cases(&sloc::language_java_script(), &[
        ("javascript block comment", "/* c */\n", 0, 1, 0),
        ("javascript block spans", "/* a\nb\n*/\n", 0, 3, 0),
    ]);
    run_classify_cases(&sloc::language_odin(), &[
        ("odin nests", "/* a /* b */ c */\n", 0, 1, 0),
        ("odin nested across lines then code", "/* a /* b\nx */ y */ z()\n", 1, 1, 0),
    ]);
    run_classify_cases(&sloc::language_lua(), &[
        ("lua block on one line", "--[[ c ]]\n", 0, 1, 0),
        ("lua block spans", "--[[ a\nb\n]]\n", 0, 3, 0),
        ("lua leveled block ignores short close", "--[==[ a ]] b ]==]\n", 0, 1, 0),
        ("lua code after block", "--[[x]] y()\n", 1, 0, 0),
    ]);
    run_classify_cases(&sloc::language_swift(), &[("swift nests", "/* a /* b */ c */\n", 0, 1, 0)]);
    run_classify_cases(&sloc::language_html(), &[
        ("html comment", "<!-- c -->\n", 0, 1, 0),
        ("html comment spans", "<!-- a\nb\n-->\n", 0, 3, 0),
        ("html code", "<p>hi</p>\n", 1, 0, 0),
    ]);
    run_classify_cases(&sloc::language_css(), &[
        ("css block comment", "/* c */\n", 0, 1, 0),
        ("css rule is code", "a { color: red; }\n", 1, 0, 0),
    ]);
    run_classify_cases(&sloc::language_haskell(), &[
        ("haskell nests", "{- a {- b -} c -}\n", 0, 1, 0),
        ("haskell line comment", "-- x\n", 0, 1, 0),
    ]);
    run_classify_cases(&sloc::language_ocaml(), &[("ocaml nests", "(* a (* b *) c *)\n", 0, 1, 0)]);
    run_classify_cases(&sloc::language_julia(), &[
        ("julia nests", "#= a #= b =# c =#\n", 0, 1, 0),
        ("julia line comment", "# x\n", 0, 1, 0),
    ]);
    run_classify_cases(&sloc::language_nim(), &[("nim nests", "#[ a #[ b ]# c ]#\n", 0, 1, 0)]);
    run_classify_cases(&sloc::language_common_lisp(), &[("lisp block nests", "#| a #| b |# c |#\n", 0, 1, 0)]);
    run_classify_cases(&sloc::language_cmake(), &[("cmake bracket comment spans", "#[[ a\nb ]]\n", 0, 2, 0)]);
    run_classify_cases(&sloc::language_markdown(), &[
        ("markdown html comment", "<!-- c -->\n", 0, 1, 0),
        ("markdown text is code", "# Heading\n", 1, 0, 0),
    ]);
}

#[test]
fn classify_raw_strings() {
    run_classify_cases(&sloc::language_go(), &[
        ("single line backtick with tokens", "s := `/* x */`\n", 1, 0, 0),
        ("multi-line backtick hides close token", "a := `start\n*/ text\nend`\n", 3, 0, 0),
        ("blank line inside backtick is blank", "a := `x\n\ny`\n", 2, 0, 1),
        ("line comment after backtick close", "s := `x` // c\n", 1, 0, 0),
    ]);
    run_classify_cases(&sloc::language_rust(), &[
        ("r# raw string", "let s = r#\"a\"#;\n", 1, 0, 0),
        ("hash mismatch does not close", "let s = r##\"has \"# inside\"##;\n", 1, 0, 0),
        ("multi-line raw hides tokens", "let s = r#\"line1\n*/ // not\nend\"#;\n", 3, 0, 0),
        ("raw identifier is not a raw string", "let r#match = 1;\n", 1, 0, 0),
        ("byte raw string", "let b = br#\"x\"#;\n", 1, 0, 0),
    ]);
    run_classify_cases(&sloc::language_python(), &[
        ("single-line docstring", "\"\"\"doc\"\"\"\n", 1, 0, 0),
        ("docstring spans as code", "def f():\n \"\"\"\n d # x\n \"\"\"\n p\n", 5, 0, 0),
        ("blank line inside docstring is blank", "\"\"\"\n\ndoc\n\"\"\"\n", 3, 0, 1),
        ("single-quote docstring", "'''\ndoc\n'''\n", 3, 0, 0),
    ]);
    run_classify_cases(&sloc::language_java_script(), &[
        ("template hides tokens", "let s = `start\n// no\nend`;\n", 3, 0, 0),
        ("blank line inside template is blank", "let s = `a\n\nb`;\n", 2, 0, 1),
    ]);
    run_classify_cases(&sloc::language_type_script(), &[("template literal spans", "const s = `x\nnot a comment\nend`;\n", 3, 0, 0)]);
    run_classify_cases(&sloc::language_odin(), &[
        ("odin backtick raw string", "s := `/* x */`\n", 1, 0, 0),
        ("odin multi-line backtick", "s := `a\n// b\nc`\n", 3, 0, 0),
    ]);
    run_classify_cases(&sloc::language_lua(), &[
        ("lua long string", "local s = [[ a ]]\n", 1, 0, 0),
        ("lua long string hides line comment", "local s = [[a\n-- no\nb]]\n", 3, 0, 0),
        ("lua leveled long string", "local s = [==[ ]] ]==]\n", 1, 0, 0),
    ]);
    run_classify_cases(&sloc::language_zig(), &[
        ("zig string with slashes is code", "const u = \"http://x\";\n", 1, 0, 0),
        ("zig multiline string is code", "const s =\n    \\\\ text\n;\n", 3, 0, 0),
    ]);
    run_classify_cases(&sloc::language_java(), &[("java text block spans", "var s = \"\"\"\n// not\n\"\"\";\n", 3, 0, 0)]);
    run_classify_cases(&sloc::language_toml(), &[("toml triple string spans", "s = \"\"\"\n# not\n\"\"\"\n", 3, 0, 0)]);
    run_classify_cases(&sloc::language_f_sharp(), &[("fsharp triple string spans", "let s = \"\"\"\n// no\n\"\"\"\n", 3, 0, 0)]);
    run_classify_cases(&sloc::language_elixir(), &[("elixir heredoc spans", "x = \"\"\"\n# no\n\"\"\"\n", 3, 0, 0)]);
}

#[test]
fn classify_heredocs() {
    run_classify_cases(&sloc::language_shell(), &[
        ("heredoc body is code", "cat <<EOF\n# not a comment\nEOF\n", 3, 0, 0),
        ("heredoc with dash", "cat <<-END\nbody\nEND\n", 3, 0, 0),
        ("quoted heredoc", "cat <<'EOF'\n# x\nEOF\n", 3, 0, 0),
        ("shift is not a heredoc", "x=$((a << b))\n# c\n", 1, 1, 0),
    ]);
    run_classify_cases(&sloc::language_ruby(), &[("ruby squiggly heredoc", "sql = <<~SQL\n# not a comment\nSQL\n", 3, 0, 0)]);
}

#[test]
fn classify_characters_and_lifetimes() {
    run_classify_cases(&sloc::language_rust(), &[
        ("lifetime is not a char", "fn f<'a>(x: &'a i32) {}\n", 1, 0, 0),
        ("char literal slash", "let c = '/';\n", 1, 0, 0),
        ("escaped quote char", "let c = '\\'';\n", 1, 0, 0),
        ("char then line comment is code", "let c = 'x'; // c\n", 1, 0, 0),
        ("lifetime then string with token", "let s: &'a str = \"//x\";\n", 1, 0, 0),
    ]);
}

#[test]
fn classify_newlines() {
    run_classify_cases(&sloc::language_go(), &[
        ("empty file", "", 0, 0, 0),
        ("no trailing newline", "foo()", 1, 0, 0),
        ("trailing newline no phantom", "foo()\n", 1, 0, 0),
        ("comment without newline", "// c", 0, 1, 0),
        ("only blank lines", "\n\n\n", 0, 0, 3),
        ("whitespace-only line is blank", "   \t\n", 0, 0, 1),
    ]);
}

#[test]
fn languages_detection() {
    check_detection(&detection_extensions_core(), sloc::language_for_extension);
    check_detection(&detection_extensions_rest(), sloc::language_for_extension);
    check_detection(&detection_filenames(), sloc::language_for_filename);
    assert!(sloc::language_for_extension(".txt").is_none(), ".txt should be unrecognized");
}

#[test]
fn count_extensions() {
    let tree = sloc::File_Tree {
        files: vec![
            tree_file("main.go", "package main\n// c\n\n"),
            tree_file("sub/lib.rs", "fn main() {}\n"),
            tree_file("readme.txt", "hello\n"),
            tree_file("data.json", "{}\n"),
        ],
    };
    let report = count_tree(&tree, false, 4);
    let by_path = report_by_path(&report);
    assert_eq!(by_path.len(), 2, "expected 2 counted files, got {:?}", report.files);
    assert_eq!(by_path["main.go"].language, "Go");
    assert_eq!(by_path["main.go"].counts, sloc::Counts { code: 1, comment: 1, blank: 1 });
    assert_eq!(by_path["sub/lib.rs"].counts, sloc::Counts { code: 1, comment: 0, blank: 0 });
}

#[test]
fn count_hidden() {
    let tree = sloc::File_Tree {
        files: vec![
            tree_file("main.go", "package main\n"),
            tree_file(".env.go", "package secret\n"),
            tree_file(".hidden/buried.go", "package buried\n"),
        ],
    };
    let report = count_tree(&tree, false, 1);
    let by_path = report_by_path(&report);
    assert_eq!(by_path.len(), 1, "default should count only main.go, got {:?}", report.files);
    assert_eq!(by_path["main.go"].language, "Go");
    let with_hidden = count_tree(&tree, true, 1);
    assert_eq!(report_by_path(&with_hidden).len(), 3, "Include_Hidden should count all 3");
}

#[test]
fn count_exclusion() {
    let tree = sloc::File_Tree {
        files: vec![
            tree_file("keep.go", "package keep\n"),
            tree_file("ignore_me.go", "package skip\n"),
            tree_file("vendor/dep.go", "package vendored\n"),
        ],
    };
    let report = sloc::count(
        &sloc::Count_Input { root: ".".to_string(), include_hidden: false, concurrency: 1 },
        |dir| sloc::tree_list_dir(&tree, dir),
        |path| sloc::tree_read_file(&tree, path),
        |path, _is_dir| path == "vendor" || path == "ignore_me.go",
    );
    let by_path = report_by_path(&report);
    assert_eq!(by_path.len(), 1, "expected only keep.go, got {:?}", report.files);
    assert_eq!(by_path["keep.go"].path, "keep.go");
}

#[test]
fn count_binary() {
    let tree = sloc::File_Tree {
        files: vec![
            tree_file("real.go", "package main\n"),
            sloc::Tree_File { path: "blob.go".to_string(), data: b"package\x00main\n".to_vec() },
        ],
    };
    let report = count_tree(&tree, false, 1);
    let by_path = report_by_path(&report);
    assert_eq!(by_path.len(), 1, "expected only real.go, got {:?}", report.files);
    assert_eq!(by_path["real.go"].path, "real.go");
}

#[test]
fn count_tests() {
    let tree = sloc::File_Tree {
        files: vec![
            tree_file("main.go", "package main\n"),
            tree_file("main_test.go", "package main\n"),
            tree_file("tests/integ.rs", "fn t() {}\n"),
            tree_file("__tests__/x.ts", "test()\n"),
            tree_file("app.spec.ts", "test()\n"),
        ],
    };
    let report = count_tree(&tree, false, 4);
    let by_path = report_by_path(&report);
    let expected = [
        ("main.go", false),
        ("main_test.go", true),
        ("tests/integ.rs", true),
        ("__tests__/x.ts", true),
        ("app.spec.ts", true),
    ];
    for (path, want) in expected {
        assert_eq!(by_path[path].is_test, want, "{path} is_test");
    }
}

#[test]
fn render_table() {
    let files = vec![
        file_count("a.go", "Go", 10, 2, 3, false),
        file_count("b.go", "Go", 5, 0, 1, false),
        file_count("c.rs", "Rust", 7, 1, 0, false),
    ];
    let rule = "─".repeat(54);
    let want = [
        rule.as_str(),
        " Language  Files  Lines  Code  Comments  Blanks  %Code",
        rule.as_str(),
        " Systems",
        "   Rust        1      8     7         1       0",
        " Managed",
        "   Go          2     21    15         2       4",
        rule.as_str(),
        " Total         3     29    22         3       4",
        rule.as_str(),
    ]
    .join("\n")
        + "\n";
    let output = sloc::render(&sloc::Render_Input { report: sloc::Report { files }, show_files: false });
    assert_eq!(output, want);
}

#[test]
fn render_files() {
    let files = vec![
        file_count("a.go", "Go", 3, 1, 0, false),
        file_count("sub/b.rs", "Rust", 2, 0, 1, false),
    ];
    let rule = "─".repeat(58);
    let want = [
        rule.as_str(),
        " Language      Files  Lines  Code  Comments  Blanks  %Code",
        rule.as_str(),
        " Systems",
        "   Rust            1      3     2         0       1",
        "     sub/b.rs             3     2         0       1",
        " Managed",
        "   Go              1      4     3         1       0",
        "     a.go                 4     3         1       0",
        rule.as_str(),
        " Total             2      7     5         1       1",
        rule.as_str(),
    ]
    .join("\n")
        + "\n";
    let output = sloc::render(&sloc::Render_Input { report: sloc::Report { files }, show_files: true });
    assert_eq!(output, want);
}

#[test]
fn render_thousands() {
    let files = vec![
        file_count("g.go", "Go", 12000, 3456, 789, false),
        file_count("g.rs", "Rust", 1000000, 0, 0, false),
    ];
    let rule = "─".repeat(63);
    let want = [
        rule.as_str(),
        " Language  Files      Lines       Code  Comments  Blanks  %Code",
        rule.as_str(),
        " Systems",
        "   Rust        1  1,000,000  1,000,000         0       0",
        " Managed",
        "   Go          1     16,245     12,000     3,456     789",
        rule.as_str(),
        " Total         2  1,016,245  1,012,000     3,456     789",
        rule.as_str(),
    ]
    .join("\n")
        + "\n";
    let output = sloc::render(&sloc::Render_Input { report: sloc::Report { files }, show_files: false });
    assert_eq!(output, want);
}

#[test]
fn render_tests() {
    let files = vec![
        file_count("a.go", "Go", 10, 2, 3, false),
        file_count("a_test.go", "Go", 4, 1, 1, true),
        file_count("b.rs", "Rust", 7, 1, 0, false),
    ];
    let rule = "─".repeat(56);
    let want = [
        rule.as_str(),
        " Language    Files  Lines  Code  Comments  Blanks  %Code",
        rule.as_str(),
        " Systems",
        "   Rust          1      8     7         1       0",
        " Managed",
        "   Go            2     21    14         3       4",
        "     source      1     15    10         2       3  71.4%",
        "     tests       1      6     4         1       1  28.6%",
        rule.as_str(),
        " Total           3     29    21         4       4",
        "   source        2     23    17         3       3  81.0%",
        "   tests         1      6     4         1       1  19.0%",
        rule.as_str(),
    ]
    .join("\n")
        + "\n";
    let output = sloc::render(&sloc::Render_Input { report: sloc::Report { files }, show_files: false });
    assert_eq!(output, want);
}

#[test]
fn render_json() {
    let files = vec![
        file_count("a.go", "Go", 10, 2, 3, false),
        file_count("a_test.go", "Go", 4, 1, 1, true),
    ];
    let want = r#"{
  "languages": [
    {
      "name": "Go",
      "category": "Managed",
      "files": 2,
      "code": 14,
      "comments": 3,
      "blanks": 4,
      "source": {
        "files": 1,
        "code": 10,
        "comments": 2,
        "blanks": 3
      },
      "tests": {
        "files": 1,
        "code": 4,
        "comments": 1,
        "blanks": 1
      }
    }
  ],
  "total": {
    "files": 2,
    "code": 14,
    "comments": 3,
    "blanks": 4,
    "source": {
      "files": 1,
      "code": 10,
      "comments": 2,
      "blanks": 3
    },
    "tests": {
      "files": 1,
      "code": 4,
      "comments": 1,
      "blanks": 1
    }
  }
}
"#;
    let output = sloc::render_json(&sloc::Report { files });
    assert_eq!(output, want);
}

#[test]
fn limitations() {
    // A JavaScript regex beginning with a star opens a false block comment that
    // bleeds into the next line — a documented inaccuracy.
    run_classify_cases(&sloc::language_java_script(), &[("regex opens false block comment", "x = /*/\ny = 2\n", 1, 1, 0)]);
}

/// Builds a file count for the render tests.
fn file_count(path: &str, language: &str, code: usize, comment: usize, blank: usize, is_test: bool) -> sloc::File_Count {
    sloc::File_Count {
        path: path.to_string(),
        language: language.to_string(),
        counts: sloc::Counts { code, comment, blank },
        is_test,
    }
}

/// Detection expectations for the original and C-style/markup extensions.
fn detection_extensions_core() -> Vec<(String, String)> {
    owned_pairs(&[
        (".go", "Go"), (".rs", "Rust"), (".py", "Python"), (".js", "JavaScript"),
        (".jsx", "JavaScript"), (".mjs", "JavaScript"), (".cjs", "JavaScript"),
        (".ts", "TypeScript"), (".tsx", "TypeScript"), (".lua", "Lua"), (".odin", "Odin"),
        (".zig", "Zig"), (".c", "C"), (".h", "C"), (".cpp", "C++"), (".hpp", "C++"),
        (".cs", "C#"), (".java", "Java"), (".swift", "Swift"), (".kt", "Kotlin"),
        (".scala", "Scala"), (".sh", "Shell"), (".bash", "Shell"), (".rb", "Ruby"),
        (".yaml", "YAML"), (".yml", "YAML"), (".toml", "TOML"), (".sql", "SQL"),
        (".mk", "Makefile"), (".html", "HTML"), (".htm", "HTML"), (".xml", "XML"),
        (".svg", "XML"), (".css", "CSS"), (".scss", "SCSS"), (".less", "LESS"),
        (".m", "Objective-C"), (".mm", "Objective-C"), (".dart", "Dart"), (".php", "PHP"),
        (".sol", "Solidity"), (".groovy", "Groovy"), (".gradle", "Groovy"), (".v", "Verilog"),
        (".sv", "Verilog"), (".glsl", "GLSL"), (".frag", "GLSL"), (".hlsl", "HLSL"),
        (".ino", "Arduino"), (".proto", "Protobuf"), (".thrift", "Thrift"), (".jsonc", "JSONC"),
        (".json5", "JSONC"), (".tf", "HCL"), (".hcl", "HCL"), (".nix", "Nix"),
    ])
}

/// Detection expectations for the remaining extensions.
fn detection_extensions_rest() -> Vec<(String, String)> {
    owned_pairs(&[
        (".md", "Markdown"), (".markdown", "Markdown"), (".vue", "Vue"), (".svelte", "Svelte"),
        (".astro", "Astro"), (".xaml", "XAML"), (".xsl", "XSLT"), (".xslt", "XSLT"),
        (".hs", "Haskell"), (".ml", "OCaml"), (".mli", "OCaml"), (".fs", "F#"), (".jl", "Julia"),
        (".nim", "Nim"), (".lisp", "Common Lisp"), (".cl", "Common Lisp"), (".scm", "Scheme"),
        (".rkt", "Racket"), (".clj", "Clojure"), (".edn", "Clojure"), (".el", "Emacs Lisp"),
        (".erl", "Erlang"), (".f90", "Fortran"), (".f", "Fortran"), (".adb", "Ada"),
        (".ads", "Ada"), (".d", "D"), (".pas", "Pascal"), (".pp", "Pascal"), (".r", "R"),
        (".ex", "Elixir"), (".exs", "Elixir"), (".cr", "Crystal"), (".ps1", "PowerShell"),
        (".cmake", "CMake"), (".tcl", "Tcl"), (".pl", "Perl"), (".pm", "Perl"), (".tex", "TeX"),
        (".vb", "Visual Basic"), (".fish", "Fish"), (".nu", "Nushell"),
    ])
}

/// Detection expectations for special filenames.
fn detection_filenames() -> Vec<(String, String)> {
    owned_pairs(&[("Makefile", "Makefile"), ("Dockerfile", "Dockerfile"), ("CMakeLists.txt", "CMake")])
}
