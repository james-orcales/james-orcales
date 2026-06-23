(* Black-box test suite for sloc.ml. It links the Sloc module and drives only the
   public surface — classify_file, the language tables, count, render, render_json,
   main — with in-memory trees. It asserts on observable output, touching nothing
   real, and compiles to its own binary (sloc_test) whose name keeps sloc.ml's guarded
   entry from running main.

   Build & run:
     ocamlopt -I +unix -I . unix.cmxa sloc.ml sloc_test.ml -o sloc_test && ./sloc_test *)

open Sloc

let total = ref 0
let failures = ref 0

let check name condition =
        incr total;
        if not condition then begin
                incr failures;
                Printf.printf "FAIL: %s\n" name
        end

(* Checks one source's exact code/comment/blank tally under a language. *)
let classify name language source code comment blank =
        let c = classify_file { source; language } in
        check
                (Printf.sprintf "classify: %s" name)
                (c.code = code && c.comment = comment && c.blank = blank)

let test_classify_comments () =
        classify "go line comment" language_go "// hi\n" 0 1 0;
        classify "go code" language_go "package main\n" 1 0 0;
        classify "go trailing comment is code" language_go "total++ // bump\n" 1 0 0;
        classify "go doc comment" language_go "/// doc\n" 0 1 0;
        classify "go code, blank, comment" language_go "a()\n\n// c\n" 1 1 1;
        classify "python hash comment" language_python "# hi\n" 0 1 0;
        classify "python code" language_python "x = 1\n" 1 0 0;
        classify "python trailing comment is code" language_python "x = 1  # set\n" 1 0 0;
        classify "javascript line comment" language_java_script "// hi\n" 0 1 0;
        classify "typescript line comment" language_type_script "// t\n" 0 1 0;
        classify "typescript type annotation is code" language_type_script
                "let x: number = 1;\n" 1 0 0;
        classify "lua line comment" language_lua "-- hi\n" 0 1 0;
        classify "lua code" language_lua "local x = 1\n" 1 0 0;
        classify "odin line comment" language_odin "// hi\n" 0 1 0;
        classify "zig line comment" language_zig "// hi\n" 0 1 0;
        classify "zig doc comment" language_zig "/// doc\n" 0 1 0;
        classify "c line comment" language_c "// hi\n" 0 1 0;
        classify "c block comment" language_c "/* c */\n" 0 1 0;
        classify "shell comment" language_shell "# hi\n" 0 1 0;
        classify "shell hash in string is code" language_shell "echo \"a # b\"\n" 1 0 0;
        classify "sql line comment" language_sql "-- hi\n" 0 1 0;
        classify "sql block comment" language_sql "/* c */\n" 0 1 0;
        classify "php slash comment" language_php "// hi\n" 0 1 0;
        classify "php hash comment" language_php "# hi\n" 0 1 0;
        classify "erlang comment" language_erlang "% hi\n" 0 1 0;
        classify "lisp line comment" language_common_lisp "; hi\n" 0 1 0;
        classify "cmake comment" language_cmake "# hi\n" 0 1 0;
        classify "fish comment" language_fish "# hi\n" 0 1 0;
        classify "fish hash in string is code" language_fish "echo \"a # b\"\n" 1 0 0;
        classify "nushell comment" language_nushell "# hi\n" 0 1 0

let test_classify_strings () =
        classify "line-comment token in a string" language_go "x := \"http://foo\"\n" 1 0 0;
        classify "block-comment token in a string" language_go
                "s := \"/* not a comment */\"\n" 1 0 0;
        classify "escaped quote in a string" language_go "s := \"she said \\\"hi\\\"\"\n" 1 0 0;
        classify "hash in a string is code" language_python "s = \"a # b\"\n" 1 0 0;
        classify "slashes in a string" language_java_script "let u = \"http://x\";\n" 1 0 0

let test_classify_block_comments () =
        classify "single line block" language_go "/* c */\n" 0 1 0;
        classify "block spanning lines" language_go "/* a\nb\n*/\n" 0 3 0;
        classify "code after close" language_go "/* a */ x()\n" 1 0 0;
        classify "code before open, spanning" language_go "x() /* open\nstill\n*/ y()\n" 2 1 0;
        classify "go does not nest" language_go "/* a /* b */ c */\n" 1 0 0;
        classify "blank line inside block is blank" language_go "/*\n   \n*/\n" 0 2 1;
        classify "nested on one line" language_rust "/* a /* b */ c */\n" 0 1 0;
        classify "nested across lines then code" language_rust
                "/* a /* b\nstill */ c */ x()\n" 1 1 0;
        classify "javascript block comment" language_java_script "/* c */\n" 0 1 0;
        classify "javascript block spans" language_java_script "/* a\nb\n*/\n" 0 3 0;
        classify "odin nests" language_odin "/* a /* b */ c */\n" 0 1 0;
        classify "odin nested across lines then code" language_odin
                "/* a /* b\nx */ y */ z()\n" 1 1 0;
        classify "lua block on one line" language_lua "--[[ c ]]\n" 0 1 0;
        classify "lua block spans" language_lua "--[[ a\nb\n]]\n" 0 3 0;
        classify "lua leveled block ignores short close" language_lua "--[==[ a ]] b ]==]\n" 0 1 0;
        classify "lua code after block" language_lua "--[[x]] y()\n" 1 0 0;
        classify "swift nests" language_swift "/* a /* b */ c */\n" 0 1 0;
        classify "html comment" language_html "<!-- c -->\n" 0 1 0;
        classify "html comment spans" language_html "<!-- a\nb\n-->\n" 0 3 0;
        classify "html code" language_html "<p>hi</p>\n" 1 0 0;
        classify "css block comment" language_css "/* c */\n" 0 1 0;
        classify "css rule is code" language_css "a { color: red; }\n" 1 0 0;
        classify "haskell nests" language_haskell "{- a {- b -} c -}\n" 0 1 0;
        classify "haskell line comment" language_haskell "-- x\n" 0 1 0;
        classify "ocaml nests" language_ocaml "(* a (* b *) c *)\n" 0 1 0;
        classify "julia nests" language_julia "#= a #= b =# c =#\n" 0 1 0;
        classify "julia line comment" language_julia "# x\n" 0 1 0;
        classify "nim nests" language_nim "#[ a #[ b ]# c ]#\n" 0 1 0;
        classify "lisp block nests" language_common_lisp "#| a #| b |# c |#\n" 0 1 0;
        classify "cmake bracket comment spans" language_cmake "#[[ a\nb ]]\n" 0 2 0;
        classify "markdown html comment" language_markdown "<!-- c -->\n" 0 1 0;
        classify "markdown text is code" language_markdown "# Heading\n" 1 0 0

let test_classify_raw_strings () =
        classify "single line backtick with tokens" language_go "s := `/* x */`\n" 1 0 0;
        classify "multi-line backtick hides close token" language_go
                "a := `start\n*/ text\nend`\n" 3 0 0;
        classify "blank line inside backtick is blank" language_go "a := `x\n\ny`\n" 2 0 1;
        classify "line comment after backtick close" language_go "s := `x` // c\n" 1 0 0;
        classify "r# raw string" language_rust "let s = r#\"a\"#;\n" 1 0 0;
        classify "hash mismatch does not close" language_rust
                "let s = r##\"has \"# inside\"##;\n" 1 0 0;
        classify "multi-line raw hides tokens" language_rust
                "let s = r#\"line1\n*/ // not\nend\"#;\n" 3 0 0;
        classify "raw identifier is not a raw string" language_rust "let r#match = 1;\n" 1 0 0;
        classify "byte raw string" language_rust "let b = br#\"x\"#;\n" 1 0 0;
        classify "single-line docstring" language_python "\"\"\"doc\"\"\"\n" 1 0 0;
        classify "docstring spans as code" language_python
                "def f():\n \"\"\"\n d # x\n \"\"\"\n p\n" 5 0 0;
        classify "blank line inside docstring is blank" language_python
                "\"\"\"\n\ndoc\n\"\"\"\n" 3 0 1;
        classify "single-quote docstring" language_python "'''\ndoc\n'''\n" 3 0 0;
        classify "template hides tokens" language_java_script
                "let s = `start\n// no\nend`;\n" 3 0 0;
        classify "blank line inside template is blank" language_java_script
                "let s = `a\n\nb`;\n" 2 0 1;
        classify "template literal spans" language_type_script
                "const s = `x\nnot a comment\nend`;\n" 3 0 0;
        classify "odin backtick raw string" language_odin "s := `/* x */`\n" 1 0 0;
        classify "odin multi-line backtick" language_odin "s := `a\n// b\nc`\n" 3 0 0;
        classify "lua long string" language_lua "local s = [[ a ]]\n" 1 0 0;
        classify "lua long string hides line comment" language_lua
                "local s = [[a\n-- no\nb]]\n" 3 0 0;
        classify "lua leveled long string" language_lua "local s = [==[ ]] ]==]\n" 1 0 0;
        classify "zig string with slashes is code" language_zig "const u = \"http://x\";\n" 1 0 0;
        classify "zig multiline string is code" language_zig "const s =\n    \\\\ text\n;\n" 3 0 0;
        classify "java text block spans" language_java "var s = \"\"\"\n// not\n\"\"\";\n" 3 0 0;
        classify "toml triple string spans" language_toml "s = \"\"\"\n# not\n\"\"\"\n" 3 0 0;
        classify "fsharp triple string spans" language_f_sharp
                "let s = \"\"\"\n// no\n\"\"\"\n" 3 0 0;
        classify "elixir heredoc spans" language_elixir "x = \"\"\"\n# no\n\"\"\"\n" 3 0 0

let test_classify_heredocs () =
        classify "heredoc body is code" language_shell "cat <<EOF\n# not a comment\nEOF\n" 3 0 0;
        classify "heredoc with dash" language_shell "cat <<-END\nbody\nEND\n" 3 0 0;
        classify "quoted heredoc" language_shell "cat <<'EOF'\n# x\nEOF\n" 3 0 0;
        classify "shift is not a heredoc" language_shell "x=$((a << b))\n# c\n" 1 1 0;
        classify "ruby squiggly heredoc" language_ruby "sql = <<~SQL\n# not a comment\nSQL\n" 3 0 0

let test_classify_characters_and_lifetimes () =
        classify "lifetime is not a char" language_rust "fn f<'a>(x: &'a i32) {}\n" 1 0 0;
        classify "char literal slash" language_rust "let c = '/';\n" 1 0 0;
        classify "escaped quote char" language_rust "let c = '\\'';\n" 1 0 0;
        classify "char then line comment is code" language_rust "let c = 'x'; // c\n" 1 0 0;
        classify "lifetime then string with token" language_rust "let s: &'a str = \"//x\";\n" 1 0 0

let test_classify_newlines () =
        classify "empty file" language_go "" 0 0 0;
        classify "no trailing newline" language_go "foo()" 1 0 0;
        classify "trailing newline no phantom" language_go "foo()\n" 1 0 0;
        classify "comment without newline" language_go "// c" 0 1 0;
        classify "only blank lines" language_go "\n\n\n" 0 0 3;
        classify "whitespace-only line is blank" language_go "   \t\n" 0 0 1

let test_limitations () =
        classify "regex opens false block comment" language_java_script "x = /*/\ny = 2\n" 1 1 0

(* ----- Detection ----- *)

let check_extension extension want =
        match language_for_extension extension with
        | Some l -> check (Printf.sprintf "ext %s" extension) (l.name = want)
        | None -> check (Printf.sprintf "ext %s recognized" extension) false

let check_filename name want =
        match language_for_filename name with
        | Some l -> check (Printf.sprintf "file %s" name) (l.name = want)
        | None -> check (Printf.sprintf "file %s recognized" name) false

let test_languages_detection () =
        List.iter (fun (e, w) -> check_extension e w)
                [
                        (".go", "Go"); (".rs", "Rust"); (".py", "Python"); (".js", "JavaScript");
                        (".jsx", "JavaScript"); (".mjs", "JavaScript"); (".cjs", "JavaScript");
                        (".ts", "TypeScript"); (".tsx", "TypeScript"); (".lua", "Lua");
                        (".odin", "Odin"); (".zig", "Zig"); (".c", "C"); (".h", "C");
                        (".cpp", "C++");
                        (".hpp", "C++"); (".cs", "C#"); (".java", "Java"); (".swift", "Swift");
                        (".kt", "Kotlin"); (".scala", "Scala"); (".sh", "Shell");
                        (".bash", "Shell");
                        (".rb", "Ruby"); (".yaml", "YAML"); (".yml", "YAML"); (".toml", "TOML");
                        (".sql", "SQL"); (".mk", "Makefile"); (".html", "HTML"); (".htm", "HTML");
                        (".xml", "XML"); (".svg", "XML"); (".css", "CSS"); (".scss", "SCSS");
                        (".less", "LESS"); (".m", "Objective-C"); (".mm", "Objective-C");
                        (".dart", "Dart"); (".php", "PHP"); (".sol", "Solidity");
                        (".groovy", "Groovy");
                        (".gradle", "Groovy"); (".v", "Verilog"); (".sv", "Verilog");
                        (".glsl", "GLSL");
                        (".frag", "GLSL"); (".hlsl", "HLSL"); (".ino", "Arduino");
                        (".proto", "Protobuf");
                        (".thrift", "Thrift"); (".jsonc", "JSONC"); (".json5", "JSONC");
                        (".tf", "HCL");
                        (".hcl", "HCL"); (".nix", "Nix"); (".md", "Markdown");
                        (".markdown", "Markdown");
                        (".vue", "Vue"); (".svelte", "Svelte"); (".astro", "Astro");
                        (".xaml", "XAML");
                        (".xsl", "XSLT"); (".xslt", "XSLT"); (".hs", "Haskell"); (".ml", "OCaml");
                        (".mli", "OCaml"); (".fs", "F#"); (".jl", "Julia"); (".nim", "Nim");
                        (".lisp", "Common Lisp"); (".cl", "Common Lisp"); (".scm", "Scheme");
                        (".rkt", "Racket"); (".clj", "Clojure"); (".edn", "Clojure");
                        (".el", "Emacs Lisp"); (".erl", "Erlang"); (".f90", "Fortran");
                        (".f", "Fortran"); (".adb", "Ada"); (".ads", "Ada"); (".d", "D");
                        (".pas", "Pascal"); (".pp", "Pascal"); (".r", "R"); (".ex", "Elixir");
                        (".exs", "Elixir"); (".cr", "Crystal"); (".ps1", "PowerShell");
                        (".cmake", "CMake"); (".tcl", "Tcl"); (".pl", "Perl"); (".pm", "Perl");
                        (".tex", "TeX"); (".vb", "Visual Basic"); (".fish", "Fish");
                        (".nu", "Nushell");
                ];
        List.iter (fun (n, w) -> check_filename n w)
                [
                        ("Makefile", "Makefile"); ("Dockerfile", "Dockerfile");
                        ("CMakeLists.txt", "CMake");
                ];
        check ".txt unrecognized" (language_for_extension ".txt" = None)

(* ----- Count ----- *)

(* An in-memory read-only tree built from a (path, content) list. *)
let memory_tree files =
        let read_file path =
                match List.assoc_opt path files with Some c -> Ok c | None -> Error "absent"
        in
        let read_dir dir =
                let prefix = if dir = "." then "" else dir ^ "/" in
                let plen = String.length prefix in
                let names = Hashtbl.create 16 in
                List.iter
                        (fun (path, _) ->
                                if String.length path > plen && String.sub path 0 plen = prefix
                                then begin
                                        let rest =
                                                String.sub path plen (String.length path - plen)
                                        in
                                        match String.index_opt rest '/' with
                                        | Some i -> Hashtbl.replace names (String.sub rest 0 i) true
                                        | None -> Hashtbl.replace names rest false
                                end)
                        files;
                Hashtbl.fold (fun name is_dir acc -> { de_name = name; de_is_dir = is_dir } :: acc)
                        names []
        in
        { read_dir; read_file }

let count_tree files ?(ignored = None) ?(hidden = false) () =
        count
                { ci_tree = memory_tree files; ci_ignored = ignored; ci_hidden = hidden;
                  ci_concurrency = 1 }

let by_path (report : report) =
        List.map (fun (f : file_count) -> (f.fc_path, f)) report.files

let test_count_extensions () =
        let report =
                count_tree
                        [ ("main.go", "package main\n// c\n\n"); ("sub/lib.rs", "fn main() {}\n");
                          ("readme.txt", "hello\n"); ("data.json", "{}\n") ]
                        ()
        in
        let map = by_path report in
        check "count extensions: two counted" (List.length map = 2);
        (match List.assoc_opt "main.go" map with
        | Some f ->
                check "count extensions: main.go is Go" (f.fc_language = "Go");
                check "count extensions: main.go counts"
                        (f.fc_counts = { code = 1; comment = 1; blank = 1 })
        | None -> check "count extensions: main.go present" false);
        match List.assoc_opt "sub/lib.rs" map with
        | Some f ->
                check "count extensions: sub/lib.rs counts"
                        (f.fc_counts = { code = 1; comment = 0; blank = 0 })
        | None -> check "count extensions: sub/lib.rs present" false

let test_count_hidden () =
        let files =
                [ ("main.go", "package main\n"); (".env.go", "package secret\n");
                  (".hidden/buried.go", "package buried\n") ]
        in
        let report = count_tree files () in
        check "count hidden: default counts only main.go" (List.length (by_path report) = 1);
        let with_hidden = count_tree files ~hidden:true () in
        check "count hidden: include_hidden counts all 3" (List.length (by_path with_hidden) = 3)

let test_count_exclusion () =
        let report =
                count_tree
                        [ ("keep.go", "package keep\n"); ("ignore_me.go", "package skip\n");
                          ("vendor/dep.go", "package vendored\n") ]
                        ~ignored:(Some (fun path _ -> path = "vendor" || path = "ignore_me.go"))
                        ()
        in
        let map = by_path report in
        check "count exclusion: only keep.go" (List.length map = 1);
        check "count exclusion: keep.go present" (List.mem_assoc "keep.go" map)

let test_count_binary () =
        let report =
                count_tree [ ("real.go", "package main\n"); ("blob.go", "package\x00main\n") ] ()
        in
        let map = by_path report in
        check "count binary: only real.go" (List.length map = 1);
        check "count binary: real.go present" (List.mem_assoc "real.go" map)

let test_count_tests () =
        let report =
                count_tree
                        [ ("main.go", "package main\n"); ("main_test.go", "package main\n");
                          ("tests/integ.rs", "fn t() {}\n"); ("__tests__/x.ts", "test()\n");
                          ("app.spec.ts", "test()\n") ]
                        ()
        in
        let map = by_path report in
        let is_test path = match List.assoc_opt path map with Some f -> f.fc_test | None -> false in
        check "count tests: main.go is source" (not (is_test "main.go"));
        check "count tests: main_test.go is test" (is_test "main_test.go");
        check "count tests: tests/integ.rs is test" (is_test "tests/integ.rs");
        check "count tests: __tests__/x.ts is test" (is_test "__tests__/x.ts");
        check "count tests: app.spec.ts is test" (is_test "app.spec.ts")

(* ----- Render ----- *)

let render_to_string report show_files =
        let buffer = Buffer.create 256 in
        render (Buffer.add_string buffer) { report; show_files };
        Buffer.contents buffer

let json_to_string report =
        let buffer = Buffer.create 256 in
        render_json (Buffer.add_string buffer) report;
        Buffer.contents buffer

let mk path language counts test =
        { fc_path = path; fc_language = language; fc_counts = counts; fc_test = test }

let rule n = String.concat "" (List.init n (fun _ -> "─"))

let test_render_table () =
        let files =
                [ mk "a.go" "Go" { code = 10; comment = 2; blank = 3 } false;
                  mk "b.go" "Go" { code = 5; comment = 0; blank = 1 } false;
                  mk "c.rs" "Rust" { code = 7; comment = 1; blank = 0 } false ]
        in
        let want =
                String.concat "\n"
                        [
                                rule 54;
                                " Language  Files  Lines  Code  Comments  Blanks  %Code";
                                rule 54;
                                " Systems";
                                "   Rust        1      8     7         1       0";
                                " Managed";
                                "   Go          2     21    15         2       4";
                                rule 54;
                                " Total         3     29    22         3       4";
                                rule 54;
                                "";
                        ]
        in
        check "render table" (render_to_string { files } false = want)

let test_render_files () =
        let files =
                [ mk "a.go" "Go" { code = 3; comment = 1; blank = 0 } false;
                  mk "sub/b.rs" "Rust" { code = 2; comment = 0; blank = 1 } false ]
        in
        let want =
                String.concat "\n"
                        [
                                rule 58;
                                " Language      Files  Lines  Code  Comments  Blanks  %Code";
                                rule 58;
                                " Systems";
                                "   Rust            1      3     2         0       1";
                                "     sub/b.rs             3     2         0       1";
                                " Managed";
                                "   Go              1      4     3         1       0";
                                "     a.go                 4     3         1       0";
                                rule 58;
                                " Total             2      7     5         1       1";
                                rule 58;
                                "";
                        ]
        in
        check "render files" (render_to_string { files } true = want)

let test_render_thousands () =
        let files =
                [ mk "g.go" "Go" { code = 12000; comment = 3456; blank = 789 } false;
                  mk "g.rs" "Rust" { code = 1000000; comment = 0; blank = 0 } false ]
        in
        let want =
                String.concat "\n"
                        [
                                rule 63;
                                " Language  Files      Lines       Code  Comments  Blanks  %Code";
                                rule 63;
                                " Systems";
                                "   Rust        1  1,000,000  1,000,000         0       0";
                                " Managed";
                                "   Go          1     16,245     12,000     3,456     789";
                                rule 63;
                                " Total         2  1,016,245  1,012,000     3,456     789";
                                rule 63;
                                "";
                        ]
        in
        check "render thousands" (render_to_string { files } false = want)

let test_render_tests () =
        let files =
                [ mk "a.go" "Go" { code = 10; comment = 2; blank = 3 } false;
                  mk "a_test.go" "Go" { code = 4; comment = 1; blank = 1 } true;
                  mk "b.rs" "Rust" { code = 7; comment = 1; blank = 0 } false ]
        in
        let want =
                String.concat "\n"
                        [
                                rule 56;
                                " Language    Files  Lines  Code  Comments  Blanks  %Code";
                                rule 56;
                                " Systems";
                                "   Rust          1      8     7         1       0";
                                " Managed";
                                "   Go            2     21    14         3       4";
                                "     source      1     15    10         2       3  71.4%";
                                "     tests       1      6     4         1       1  28.6%";
                                rule 56;
                                " Total           3     29    21         4       4";
                                "   source        2     23    17         3       3  81.0%";
                                "   tests         1      6     4         1       1  19.0%";
                                rule 56;
                                "";
                        ]
        in
        check "render tests" (render_to_string { files } false = want)

let test_render_json () =
        let files =
                [ mk "a.go" "Go" { code = 10; comment = 2; blank = 3 } false;
                  mk "a_test.go" "Go" { code = 4; comment = 1; blank = 1 } true ]
        in
        let want =
                "{\n\
                \  \"languages\": [\n\
                \    {\n\
                \      \"name\": \"Go\",\n\
                \      \"category\": \"Managed\",\n\
                \      \"files\": 2,\n\
                \      \"code\": 14,\n\
                \      \"comments\": 3,\n\
                \      \"blanks\": 4,\n\
                \      \"source\": {\n\
                \        \"files\": 1,\n\
                \        \"code\": 10,\n\
                \        \"comments\": 2,\n\
                \        \"blanks\": 3\n\
                \      },\n\
                \      \"tests\": {\n\
                \        \"files\": 1,\n\
                \        \"code\": 4,\n\
                \        \"comments\": 1,\n\
                \        \"blanks\": 1\n\
                \      }\n\
                \    }\n\
                \  ],\n\
                \  \"total\": {\n\
                \    \"files\": 2,\n\
                \    \"code\": 14,\n\
                \    \"comments\": 3,\n\
                \    \"blanks\": 4,\n\
                \    \"source\": {\n\
                \      \"files\": 1,\n\
                \      \"code\": 10,\n\
                \      \"comments\": 2,\n\
                \      \"blanks\": 3\n\
                \    },\n\
                \    \"tests\": {\n\
                \      \"files\": 1,\n\
                \      \"code\": 4,\n\
                \      \"comments\": 1,\n\
                \      \"blanks\": 1\n\
                \    }\n\
                \  }\n\
                 }\n"
        in
        check "render json" (json_to_string { files } = want)

(* ----- Main ----- *)

let test_main_paths () =
        let disk = memory_tree [ ("main.go", "package main\n") ] in
        let run arguments =
                let out = Buffer.create 256 in
                let err = Buffer.create 64 in
                let code =
                        main
                                {
                                        mi_arguments = arguments;
                                        mi_output = { write_string = Buffer.add_string out };
                                        mi_error = { write_string = Buffer.add_string err };
                                        mi_open = (fun _ -> disk);
                                        mi_is_dir = (fun _ -> Ok true);
                                        mi_read_file = (fun _ -> Ok "");
                                        mi_ignore_for = (fun _ -> None);
                                        mi_concurrency = 1;
                                }
                in
                check
                        (Printf.sprintf "main %s exit 0" (String.concat " " arguments))
                        (code = 0);
                Buffer.contents out
        in
        check "main positional path counts Go" (string_contains (run [ "sloc"; "src" ]) "Go");
        check "main default path counts Go" (string_contains (run [ "sloc" ]) "Go")

let () =
        test_classify_comments ();
        test_classify_strings ();
        test_classify_block_comments ();
        test_classify_raw_strings ();
        test_classify_heredocs ();
        test_classify_characters_and_lifetimes ();
        test_classify_newlines ();
        test_limitations ();
        test_languages_detection ();
        test_count_extensions ();
        test_count_hidden ();
        test_count_exclusion ();
        test_count_binary ();
        test_count_tests ();
        test_render_table ();
        test_render_files ();
        test_render_thousands ();
        test_render_tests ();
        test_render_json ();
        test_main_paths ();
        if !failures > 0 then begin
                Printf.printf "\n%d of %d checks failed\n" !failures !total;
                exit 1
        end
        else begin
                Printf.printf "all %d checks passed\n" !total;
                exit 0
        end
