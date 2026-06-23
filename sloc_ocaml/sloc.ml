(* sloc_ocaml — a faithful, standard-distribution-only OCaml port of the Go ./sloc
   source-line counter. A generic per-line scanner, configured by a language,
   partitions every physical line into code / comment / blank; a tree walk classifies
   recognized files across Domains; a renderer prints an aligned table (or JSON).

   The whole program lives in this one file; its black-box suite lives beside it in
   sloc_test.ml, which links this file and runs as its own binary.

   Build:  ocamlopt -I +unix unix.cmxa sloc.ml -o sloc_ocaml
   Test:   ocamlopt -I +unix -I . unix.cmxa sloc.ml sloc_test.ml -o sloc_test && ./sloc_test
   Run:    ./sloc_ocaml [paths] [--files --no-ignore --hidden --json]

   Errors are values, never exceptions: the only exception handling is the OS boundary
   (os_result) in the composition root. *)

(* ===================================================================== *)
(* Counts                                                                 *)
(* ===================================================================== *)

(* The line partition of a file or group of files. *)
type counts = { code : int; comment : int; blank : int }

let zero_counts = { code = 0; comment = 0; blank = 0 }

(* The total physical line count: the three partitions sum to it because every line
   is counted exactly once. *)
let counts_lines (c : counts) = c.code + c.comment + c.blank

(* Adds one partition into another. *)
let counts_add (a : counts) (b : counts) =
        { code = a.code + b.code; comment = a.comment + b.comment; blank = a.blank + b.blank }

(* The partition a single physical line falls into. *)
type line_kind = Blank | Code | Comment

(* ===================================================================== *)
(* Languages                                                              *)
(* ===================================================================== *)

(* A string whose body is taken verbatim and may span lines, so a comment token
   inside it is inert. [hashable] marks a Rust-style raw string (lead, N hashes,
   quote), which closes only on a quote followed by exactly N hashes. *)
type verbatim_delimiter = { vopen : string; vclose : string; hashable : bool }

(* A single-line string or character literal. [escape] is the byte escaping the next
   character (None for none). [character_like] tells a Rust lifetime tick apart from
   a character literal. *)
type quote_delimiter = {
        qopen : string;
        qclose : string;
        escape : char option;
        character_like : bool;
}

(* How one language's lines are read: the tokens that begin a comment or string, and
   whether its block comments nest. The scanner is generic over this value. *)
type language = {
        name : string;
        line_comment : string list;
        block_comment_open : string;
        block_comment_close : string;
        block_comment_nests : bool;
        verbatim_strings : verbatim_delimiter list;
        quote_strings : quote_delimiter list;
        long_bracket : bool;
        test_prefixes : string list;
        test_infixes : string list;
        heredoc : bool;
}

let base_language =
        {
                name = "";
                line_comment = [];
                block_comment_open = "";
                block_comment_close = "";
                block_comment_nests = false;
                verbatim_strings = [];
                quote_strings = [];
                long_bracket = false;
                test_prefixes = [];
                test_infixes = [];
                heredoc = false;
        }

let dquote = { qopen = "\""; qclose = "\""; escape = Some '\\'; character_like = false }
let squote = { qopen = "'"; qclose = "'"; escape = Some '\\'; character_like = false }
let squote_char = { qopen = "'"; qclose = "'"; escape = Some '\\'; character_like = true }
let squote_literal = { qopen = "'"; qclose = "'"; escape = None; character_like = false }
let backtick_string = { vopen = "`"; vclose = "`"; hashable = false }
let triple_double = { vopen = "\"\"\""; vclose = "\"\"\""; hashable = false }
let triple_single = { vopen = "'''"; vclose = "'''"; hashable = false }

(* Shared quote sets. *)
let c_family_quotes = [ dquote; squote_char ]
let plain_quotes = [ dquote; squote ]
let double_quote = [ dquote ]

(* A language with only line comments and string quotes (no block comments). *)
let commented name comments quotes =
        { base_language with name; line_comment = comments; quote_strings = quotes }

(* A markup language whose only comment is <!-- -->. *)
let markup name =
        { base_language with name; block_comment_open = "<!--"; block_comment_close = "-->" }

let language_go =
        {
                base_language with
                name = "Go";
                test_infixes = [ "_test." ];
                line_comment = [ "//" ];
                block_comment_open = "/*";
                block_comment_close = "*/";
                verbatim_strings = [ backtick_string ];
                quote_strings = [ dquote; squote_char ];
        }

let language_rust =
        {
                base_language with
                name = "Rust";
                line_comment = [ "//" ];
                block_comment_open = "/*";
                block_comment_close = "*/";
                block_comment_nests = true;
                verbatim_strings =
                        [ { vopen = "br"; vclose = ""; hashable = true };
                          { vopen = "r"; vclose = ""; hashable = true } ];
                quote_strings = [ dquote; squote_char ];
        }

let language_python =
        {
                base_language with
                name = "Python";
                test_prefixes = [ "test_" ];
                test_infixes = [ "_test." ];
                line_comment = [ "#" ];
                (* Triple quotes precede the single quotes so the scanner takes them whole. *)
                verbatim_strings = [ triple_double; triple_single ];
                quote_strings = [ dquote; squote ];
        }

let language_java_script =
        {
                base_language with
                name = "JavaScript";
                test_infixes = [ ".test."; ".spec." ];
                line_comment = [ "//" ];
                block_comment_open = "/*";
                block_comment_close = "*/";
                verbatim_strings = [ backtick_string ];
                quote_strings = [ dquote; squote ];
        }

let language_type_script =
        {
                base_language with
                name = "TypeScript";
                test_infixes = [ ".test."; ".spec." ];
                line_comment = [ "//" ];
                block_comment_open = "/*";
                block_comment_close = "*/";
                verbatim_strings = [ backtick_string ];
                quote_strings = [ dquote; squote ];
        }

let language_c =
        {
                base_language with
                name = "C";
                line_comment = [ "//" ];
                block_comment_open = "/*";
                block_comment_close = "*/";
                quote_strings = c_family_quotes;
        }

let language_cpp =
        {
                base_language with
                name = "C++";
                line_comment = [ "//" ];
                block_comment_open = "/*";
                block_comment_close = "*/";
                quote_strings = c_family_quotes;
        }

let language_c_sharp =
        {
                base_language with
                name = "C#";
                test_infixes = [ "Test."; "Tests." ];
                line_comment = [ "//" ];
                block_comment_open = "/*";
                block_comment_close = "*/";
                verbatim_strings = [ triple_double ];
                quote_strings = c_family_quotes;
        }

let language_java =
        {
                base_language with
                name = "Java";
                test_infixes = [ "Test."; "Tests." ];
                line_comment = [ "//" ];
                block_comment_open = "/*";
                block_comment_close = "*/";
                verbatim_strings = [ triple_double ];
                quote_strings = c_family_quotes;
        }

let language_swift =
        {
                base_language with
                name = "Swift";
                test_infixes = [ "Tests."; "Test." ];
                line_comment = [ "//" ];
                block_comment_open = "/*";
                block_comment_close = "*/";
                block_comment_nests = true;
                verbatim_strings = [ triple_double ];
                quote_strings = double_quote;
        }

let language_kotlin =
        {
                base_language with
                name = "Kotlin";
                test_infixes = [ "Test."; "Tests." ];
                line_comment = [ "//" ];
                block_comment_open = "/*";
                block_comment_close = "*/";
                block_comment_nests = true;
                verbatim_strings = [ triple_double ];
                quote_strings = c_family_quotes;
        }

let language_scala =
        {
                base_language with
                name = "Scala";
                test_infixes = [ "Test."; "Tests."; "Spec." ];
                line_comment = [ "//" ];
                block_comment_open = "/*";
                block_comment_close = "*/";
                block_comment_nests = true;
                verbatim_strings = [ triple_double ];
                quote_strings = c_family_quotes;
        }

let language_shell =
        {
                base_language with
                name = "Shell";
                line_comment = [ "#" ];
                heredoc = true;
                quote_strings = [ dquote; squote_literal ];
        }

let language_ruby =
        {
                base_language with
                name = "Ruby";
                test_infixes = [ "_spec."; "_test." ];
                line_comment = [ "#" ];
                heredoc = true;
                quote_strings = [ dquote; squote ];
        }

let language_yaml = commented "YAML" [ "#" ] [ dquote; squote ]

let language_toml =
        {
                base_language with
                name = "TOML";
                line_comment = [ "#" ];
                verbatim_strings = [ triple_double; triple_single ];
                quote_strings = [ dquote; squote ];
        }

let language_sql =
        {
                base_language with
                name = "SQL";
                line_comment = [ "--" ];
                block_comment_open = "/*";
                block_comment_close = "*/";
                quote_strings = [ dquote; squote ];
        }

let language_makefile = commented "Makefile" [ "#" ] []
let language_dockerfile = commented "Dockerfile" [ "#" ] []

let language_html = markup "HTML"
let language_xml = markup "XML"

let language_css =
        {
                base_language with
                name = "CSS";
                block_comment_open = "/*";
                block_comment_close = "*/";
                quote_strings = [ dquote; squote ];
        }

let language_scss =
        {
                base_language with
                name = "SCSS";
                line_comment = [ "//" ];
                block_comment_open = "/*";
                block_comment_close = "*/";
                quote_strings = [ dquote; squote ];
        }

let language_less =
        {
                base_language with
                name = "LESS";
                line_comment = [ "//" ];
                block_comment_open = "/*";
                block_comment_close = "*/";
                quote_strings = [ dquote; squote ];
        }

let language_lua =
        {
                base_language with
                name = "Lua";
                line_comment = [ "--" ];
                long_bracket = true;
                quote_strings = [ dquote; squote ];
        }

let language_odin =
        {
                base_language with
                name = "Odin";
                line_comment = [ "//" ];
                block_comment_open = "/*";
                block_comment_close = "*/";
                block_comment_nests = true;
                verbatim_strings = [ backtick_string ];
                quote_strings = [ dquote; squote_char ];
        }

let language_zig = commented "Zig" [ "//" ] c_family_quotes

let language_objective_c =
        {
                base_language with
                name = "Objective-C";
                line_comment = [ "//" ];
                block_comment_open = "/*";
                block_comment_close = "*/";
                quote_strings = c_family_quotes;
        }

let language_dart =
        {
                base_language with
                name = "Dart";
                line_comment = [ "//" ];
                block_comment_open = "/*";
                block_comment_close = "*/";
                block_comment_nests = true;
                verbatim_strings = [ triple_double; triple_single ];
                quote_strings = plain_quotes;
        }

let language_php =
        {
                base_language with
                name = "PHP";
                test_infixes = [ "Test." ];
                line_comment = [ "//"; "#" ];
                block_comment_open = "/*";
                block_comment_close = "*/";
                quote_strings = plain_quotes;
        }

let language_solidity =
        {
                base_language with
                name = "Solidity";
                line_comment = [ "//" ];
                block_comment_open = "/*";
                block_comment_close = "*/";
                quote_strings = plain_quotes;
        }

let language_groovy =
        {
                base_language with
                name = "Groovy";
                line_comment = [ "//" ];
                block_comment_open = "/*";
                block_comment_close = "*/";
                verbatim_strings = [ triple_double; triple_single ];
                quote_strings = plain_quotes;
        }

let language_verilog =
        {
                base_language with
                name = "Verilog";
                line_comment = [ "//" ];
                block_comment_open = "/*";
                block_comment_close = "*/";
                quote_strings = double_quote;
        }

let language_glsl =
        {
                base_language with
                name = "GLSL";
                line_comment = [ "//" ];
                block_comment_open = "/*";
                block_comment_close = "*/";
                quote_strings = double_quote;
        }

let language_hlsl =
        {
                base_language with
                name = "HLSL";
                line_comment = [ "//" ];
                block_comment_open = "/*";
                block_comment_close = "*/";
                quote_strings = double_quote;
        }

let language_arduino =
        {
                base_language with
                name = "Arduino";
                line_comment = [ "//" ];
                block_comment_open = "/*";
                block_comment_close = "*/";
                quote_strings = c_family_quotes;
        }

let language_protobuf =
        {
                base_language with
                name = "Protobuf";
                line_comment = [ "//" ];
                block_comment_open = "/*";
                block_comment_close = "*/";
                quote_strings = plain_quotes;
        }

let language_thrift =
        {
                base_language with
                name = "Thrift";
                line_comment = [ "//"; "#" ];
                block_comment_open = "/*";
                block_comment_close = "*/";
                quote_strings = plain_quotes;
        }

let language_jsonc =
        {
                base_language with
                name = "JSONC";
                line_comment = [ "//" ];
                block_comment_open = "/*";
                block_comment_close = "*/";
                quote_strings = double_quote;
        }

let language_hcl =
        {
                base_language with
                name = "HCL";
                line_comment = [ "#"; "//" ];
                block_comment_open = "/*";
                block_comment_close = "*/";
                quote_strings = double_quote;
        }

let language_nix =
        {
                base_language with
                name = "Nix";
                line_comment = [ "#" ];
                block_comment_open = "/*";
                block_comment_close = "*/";
                verbatim_strings = [ { vopen = "''"; vclose = "''"; hashable = false } ];
                quote_strings = double_quote;
        }

let language_markdown = markup "Markdown"
let language_vue = markup "Vue"
let language_svelte = markup "Svelte"
let language_astro = markup "Astro"
let language_xaml = markup "XAML"
let language_xslt = markup "XSLT"

let language_haskell =
        {
                base_language with
                name = "Haskell";
                line_comment = [ "--" ];
                block_comment_open = "{-";
                block_comment_close = "-}";
                block_comment_nests = true;
                quote_strings = double_quote;
        }

let language_ocaml =
        {
                base_language with
                name = "OCaml";
                block_comment_open = "(*";
                block_comment_close = "*)";
                block_comment_nests = true;
                quote_strings = double_quote;
        }

let language_f_sharp =
        {
                base_language with
                name = "F#";
                line_comment = [ "//" ];
                block_comment_open = "(*";
                block_comment_close = "*)";
                block_comment_nests = true;
                verbatim_strings = [ triple_double ];
                quote_strings = double_quote;
        }

let language_julia =
        {
                base_language with
                name = "Julia";
                line_comment = [ "#" ];
                block_comment_open = "#=";
                block_comment_close = "=#";
                block_comment_nests = true;
                verbatim_strings = [ triple_double ];
                quote_strings = double_quote;
        }

let language_nim =
        {
                base_language with
                name = "Nim";
                line_comment = [ "#" ];
                block_comment_open = "#[";
                block_comment_close = "]#";
                block_comment_nests = true;
                verbatim_strings = [ triple_double ];
                quote_strings = double_quote;
        }

let language_common_lisp =
        {
                base_language with
                name = "Common Lisp";
                line_comment = [ ";" ];
                block_comment_open = "#|";
                block_comment_close = "|#";
                block_comment_nests = true;
                quote_strings = double_quote;
        }

let language_scheme =
        {
                base_language with
                name = "Scheme";
                line_comment = [ ";" ];
                block_comment_open = "#|";
                block_comment_close = "|#";
                block_comment_nests = true;
                quote_strings = double_quote;
        }

let language_racket =
        {
                base_language with
                name = "Racket";
                line_comment = [ ";" ];
                block_comment_open = "#|";
                block_comment_close = "|#";
                block_comment_nests = true;
                quote_strings = double_quote;
        }

let language_clojure = commented "Clojure" [ ";" ] double_quote
let language_emacs_lisp = commented "Emacs Lisp" [ ";" ] double_quote
let language_erlang = commented "Erlang" [ "%" ] double_quote
let language_fortran = commented "Fortran" [ "!" ] plain_quotes
let language_ada = commented "Ada" [ "--" ] c_family_quotes

let language_d =
        {
                base_language with
                name = "D";
                line_comment = [ "//" ];
                block_comment_open = "/*";
                block_comment_close = "*/";
                verbatim_strings = [ backtick_string ];
                quote_strings = c_family_quotes;
        }

let language_pascal =
        {
                base_language with
                name = "Pascal";
                line_comment = [ "//" ];
                block_comment_open = "{";
                block_comment_close = "}";
                quote_strings = [ squote_literal ];
        }

let language_r = commented "R" [ "#" ] plain_quotes

let language_elixir =
        {
                base_language with
                name = "Elixir";
                test_infixes = [ "_test." ];
                line_comment = [ "#" ];
                verbatim_strings = [ triple_double; triple_single ];
                quote_strings = plain_quotes;
        }

let language_crystal = commented "Crystal" [ "#" ] double_quote

let language_power_shell =
        {
                base_language with
                name = "PowerShell";
                line_comment = [ "#" ];
                block_comment_open = "<#";
                block_comment_close = "#>";
                quote_strings = plain_quotes;
        }

let language_fish = commented "Fish" [ "#" ] plain_quotes
let language_nushell = commented "Nushell" [ "#" ] plain_quotes

let language_cmake =
        {
                base_language with
                name = "CMake";
                line_comment = [ "#" ];
                long_bracket = true;
                quote_strings = double_quote;
        }

let language_tcl = commented "Tcl" [ "#" ] double_quote

let language_perl =
        {
                base_language with
                name = "Perl";
                line_comment = [ "#" ];
                heredoc = true;
                quote_strings = plain_quotes;
        }

let language_tex = commented "TeX" [ "%" ] []
let language_visual_basic = commented "Visual Basic" [ "'" ] double_quote

(* Resolves the seeded language for a file extension (with the leading dot). *)
let language_for_extension extension =
        match extension with
        | ".go" -> Some language_go
        | ".rs" -> Some language_rust
        | ".py" -> Some language_python
        | ".js" | ".jsx" | ".mjs" | ".cjs" -> Some language_java_script
        | ".ts" | ".tsx" -> Some language_type_script
        | ".lua" -> Some language_lua
        | ".odin" -> Some language_odin
        | ".zig" -> Some language_zig
        | ".c" | ".h" -> Some language_c
        | ".cpp" | ".cc" | ".cxx" | ".hpp" | ".hh" | ".hxx" -> Some language_cpp
        | ".cs" -> Some language_c_sharp
        | ".java" -> Some language_java
        | ".swift" -> Some language_swift
        | ".kt" | ".kts" -> Some language_kotlin
        | ".scala" | ".sc" -> Some language_scala
        | ".sh" | ".bash" | ".zsh" -> Some language_shell
        | ".rb" -> Some language_ruby
        | ".yaml" | ".yml" -> Some language_yaml
        | ".toml" -> Some language_toml
        | ".sql" -> Some language_sql
        | ".mk" -> Some language_makefile
        | ".dockerfile" -> Some language_dockerfile
        | ".html" | ".htm" -> Some language_html
        | ".xml" | ".svg" -> Some language_xml
        | ".css" -> Some language_css
        | ".scss" -> Some language_scss
        | ".less" -> Some language_less
        | ".m" | ".mm" -> Some language_objective_c
        | ".dart" -> Some language_dart
        | ".php" | ".phtml" -> Some language_php
        | ".sol" -> Some language_solidity
        | ".groovy" | ".gradle" -> Some language_groovy
        | ".v" | ".sv" | ".svh" -> Some language_verilog
        | ".glsl" | ".vert" | ".frag" | ".comp" | ".geom" -> Some language_glsl
        | ".hlsl" -> Some language_hlsl
        | ".ino" -> Some language_arduino
        | ".proto" -> Some language_protobuf
        | ".thrift" -> Some language_thrift
        | ".jsonc" | ".json5" -> Some language_jsonc
        | ".tf" | ".hcl" | ".tfvars" -> Some language_hcl
        | ".nix" -> Some language_nix
        | ".md" | ".markdown" -> Some language_markdown
        | ".vue" -> Some language_vue
        | ".svelte" -> Some language_svelte
        | ".astro" -> Some language_astro
        | ".xaml" -> Some language_xaml
        | ".xsl" | ".xslt" -> Some language_xslt
        | ".hs" | ".lhs" -> Some language_haskell
        | ".ml" | ".mli" -> Some language_ocaml
        | ".fs" | ".fsx" | ".fsi" -> Some language_f_sharp
        | ".jl" -> Some language_julia
        | ".nim" | ".nims" -> Some language_nim
        | ".lisp" | ".lsp" | ".cl" -> Some language_common_lisp
        | ".scm" | ".ss" -> Some language_scheme
        | ".rkt" -> Some language_racket
        | ".clj" | ".cljs" | ".cljc" | ".edn" -> Some language_clojure
        | ".el" -> Some language_emacs_lisp
        | ".erl" | ".hrl" -> Some language_erlang
        | ".f90" | ".f95" | ".f03" | ".f08" | ".f" | ".for" -> Some language_fortran
        | ".adb" | ".ads" | ".ada" -> Some language_ada
        | ".d" -> Some language_d
        | ".pas" | ".pp" | ".dpr" -> Some language_pascal
        | ".r" | ".R" -> Some language_r
        | ".ex" | ".exs" -> Some language_elixir
        | ".cr" -> Some language_crystal
        | ".ps1" | ".psm1" | ".psd1" -> Some language_power_shell
        | ".fish" -> Some language_fish
        | ".nu" -> Some language_nushell
        | ".cmake" -> Some language_cmake
        | ".tcl" -> Some language_tcl
        | ".pl" | ".pm" | ".t" | ".pod" -> Some language_perl
        | ".tex" | ".sty" | ".cls" | ".ltx" -> Some language_tex
        | ".vb" -> Some language_visual_basic
        | _ -> None

(* Resolves the language for an extensionless file recognized by its name. *)
let language_for_filename name =
        match name with
        | "Makefile" | "makefile" | "GNUmakefile" -> Some language_makefile
        | "Dockerfile" -> Some language_dockerfile
        | "CMakeLists.txt" -> Some language_cmake
        | _ -> None

(* The extension of a path's final element, from the last dot, or "" — Go's path.Ext.
   Unlike Filename.extension it keeps a leading-dot name's suffix, so "..fish" yields
   ".fish" (a real file in fish-shell's completions). *)
let path_ext file_path =
        let base = Filename.basename file_path in
        match String.rindex_opt base '.' with
        | Some i -> String.sub base i (String.length base - i)
        | None -> ""

(* Resolves the language for a path by its extension, or by its filename. *)
let language_for_path file_path =
        match language_for_extension (path_ext file_path) with
        | Some language -> Some language
        | None -> language_for_filename (Filename.basename file_path)

(* ===================================================================== *)
(* Byte helpers                                                           *)
(* ===================================================================== *)

(* ASCII whitespace; newline is excluded because the source is split on it. *)
let byte_is_space c = c = ' ' || c = '\t' || c = '\r' || c = '\012' || c = '\011'

(* A byte that may appear in an identifier, used to keep a raw-string lead from being
   recognized in the middle of a name. *)
let byte_is_identifier c =
        c = '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')

(* Reports whether prefix occurs in source at pos, within the line end hi. A manual
   loop, not a recursive closure, since this is the scanner's hottest check. *)
let has_prefix_at source pos hi prefix =
        let n = String.length prefix in
        if pos + n > hi then false
        else begin
                let i = ref 0 in
                while !i < n && source.[pos + !i] = prefix.[!i] do
                        incr i
                done;
                !i = n
        end

(* Reports whether any line-comment token occurs in source at pos. *)
let rec line_comment_at source pos hi = function
        | [] -> false
        | prefix :: rest -> has_prefix_at source pos hi prefix || line_comment_at source pos hi rest

(* Reports whether source[lo, hi) is empty or only ASCII whitespace. *)
let line_is_blank source lo hi =
        let i = ref lo in
        while !i < hi && byte_is_space source.[!i] do
                incr i
        done;
        !i = hi

(* ===================================================================== *)
(* Scanner                                                                *)
(* ===================================================================== *)

(* The scanner state, mutated in place as the scan walks. One per file: [depth],
   [raw_close], [comment_close], and [heredoc_close] cross line boundaries ("" / 0 means
   "not in that state"); [verdict] is reset at the start of each line. In-place
   mutation, not functional updates, keeps the per-byte hot path allocation-free. *)
type scan_state = {
        mutable depth : int;
        mutable raw_close : string;
        mutable comment_close : string;
        mutable heredoc_close : string;
        mutable verdict : line_kind;
}

let fresh_state () =
        { depth = 0; raw_close = ""; comment_close = ""; heredoc_close = ""; verdict = Blank }

(* The line's verdict is the highest partition any byte reached: code outranks comment,
   comment outranks blank. So code is absorbing, and a comment only lifts a blank line. *)
let mark_code st = st.verdict <- Code
let mark_comment st = match st.verdict with Blank -> st.verdict <- Comment | _ -> ()

(* A short alias for Option.is_none, used by the opener chain below to stay within the
   column limit; an opener returns [Some next] (the cursor past what it opened) or [None]. *)
let is_none = Option.is_none

(* A language prepared for scanning: its configuration plus a table of the bytes that
   can begin something the scan must inspect, so a run of ordinary code bytes is
   skipped in bulk. *)
type scanner = { sc_language : language; trigger : bool array }

(* Fills the trigger table from the first byte of every opener the language defines,
   taken from its fields alone so no language is special-cased. *)
let language_scanner language =
        let trigger = Array.make 256 false in
        let mark s = if String.length s > 0 then trigger.(Char.code s.[0]) <- true in
        List.iter mark language.line_comment;
        mark language.block_comment_open;
        List.iter (fun (d : verbatim_delimiter) -> mark d.vopen) language.verbatim_strings;
        List.iter (fun (d : quote_delimiter) -> mark d.qopen) language.quote_strings;
        if language.heredoc then trigger.(Char.code '<') <- true;
        if language.long_bracket then trigger.(Char.code '[') <- true;
        { sc_language = language; trigger }

(* The long bracket — '[' then a run of '=' then '[' — opening at the cursor, as its
   matching closer ']' run-of-'=' ']' and the opener length, or None. *)
let long_bracket_open source cursor hi =
        if cursor >= hi || source.[cursor] <> '[' then None
        else begin
                let scan = ref (cursor + 1) in
                let equals = ref 0 in
                while !scan < hi && source.[!scan] = '=' do
                        incr equals;
                        incr scan
                done;
                if !scan >= hi || source.[!scan] <> '[' then None
                else Some ("]" ^ String.make !equals '=' ^ "]", !scan - cursor + 1)
        end

(* Advances inside a verbatim string, where every byte is code and only the matching
   close ends it. *)
let line_scan_raw st source cursor hi =
        mark_code st;
        let close = st.raw_close in
        if has_prefix_at source cursor hi close then begin
                st.raw_close <- "";
                cursor + String.length close
        end
        else cursor + 1

(* Advances inside a block comment, where every byte is comment and only an open (when
   nesting) or a close moves the depth. *)
let line_scan_block st source cursor hi language =
        mark_comment st;
        if
                language.block_comment_nests
                && has_prefix_at source cursor hi language.block_comment_open
        then begin
                st.depth <- st.depth + 1;
                cursor + String.length language.block_comment_open
        end
        else if has_prefix_at source cursor hi language.block_comment_close then begin
                st.depth <- st.depth - 1;
                cursor + String.length language.block_comment_close
        end
        else cursor + 1

(* Advances inside a long-bracket comment, where every byte is comment and only the
   matching leveled closer ends it. *)
let line_scan_long_comment_body st source cursor hi =
        mark_comment st;
        let close = st.comment_close in
        if has_prefix_at source cursor hi close then begin
                st.comment_close <- "";
                cursor + String.length close
        end
        else cursor + 1

(* The long-bracket comment — a line-comment token then a long bracket — opening at the
   cursor, recording its closer, as the cursor past the opener, or None. *)
let rec long_comment_open st source cursor hi = function
        | [] -> None
        | token :: rest ->
                if not (has_prefix_at source cursor hi token) then
                        long_comment_open st source cursor hi rest
                else
                        let after = cursor + String.length token in
                        (match long_bracket_open source after hi with
                        | Some (closer, opener_size) ->
                                st.comment_close <- closer;
                                mark_comment st;
                                Some (after + opener_size)
                        | None -> long_comment_open st source cursor hi rest)

let line_scan_long_comment st source cursor hi language =
        if not language.long_bracket then None
        else long_comment_open st source cursor hi language.line_comment

(* The long-bracket string — like [[ or [=[ — opening at the cursor, recording its
   closer, as the cursor past the opener, or None. *)
let line_scan_long_string st source cursor hi language =
        if not language.long_bracket then None
        else
                match long_bracket_open source cursor hi with
                | None -> None
                | Some (closer, opener_size) ->
                        st.raw_close <- closer;
                        mark_code st;
                        Some (cursor + opener_size)

(* Skips an optional <<- or <<~ heredoc sigil. *)
let heredoc_skip_sigil source cursor hi =
        if cursor >= hi then cursor
        else if source.[cursor] = '-' || source.[cursor] = '~' then cursor + 1
        else cursor

(* Skips spaces and tabs between the heredoc operator and its delimiter. *)
let heredoc_skip_spaces source cursor hi =
        let c = ref cursor in
        while !c < hi && byte_is_space source.[!c] do
                incr c
        done;
        !c

(* Skips a run of identifier bytes. *)
let heredoc_skip_identifier source cursor hi =
        let c = ref cursor in
        while !c < hi && byte_is_identifier source.[!c] do
                incr c
        done;
        !c

(* Reports whether a byte opens a quoted heredoc delimiter. *)
let heredoc_is_quote c = c = '\'' || c = '"' || c = '`'

(* Reports whether a byte may begin an unquoted heredoc delimiter: uppercase or
   underscore, the convention that keeps a << b from looking like a heredoc. *)
let heredoc_word_start c = c = '_' || (c >= 'A' && c <= 'Z')

(* The heredoc opener beginning at the cursor — << then an optional - or ~, optional
   space, then a quoted word or an uppercase/underscore word — as its terminator word
   and the opener's byte length, or None. *)
let heredoc_open source cursor hi =
        if not (has_prefix_at source cursor hi "<<") then None
        else begin
                let after_sigil = heredoc_skip_sigil source (cursor + 2) hi in
                let scan = heredoc_skip_spaces source after_sigil hi in
                let quoted = scan < hi && heredoc_is_quote source.[scan] in
                let scan = if quoted then scan + 1 else scan in
                let start = scan in
                let scan = heredoc_skip_identifier source scan hi in
                if scan = start then None
                else if (not quoted) && not (heredoc_word_start source.[start]) then None
                else
                        let terminator = String.sub source start (scan - start) in
                        let scan = if quoted && scan < hi then scan + 1 else scan in
                        Some (terminator, scan - cursor)
        end

(* The heredoc opening at the cursor, recording its terminator, as the cursor past the
   opener, or None. *)
let line_scan_heredoc st source cursor hi language =
        if not language.heredoc then None
        else
                match heredoc_open source cursor hi with
                | None -> None
                | Some (terminator, opener_size) ->
                        st.heredoc_close <- terminator;
                        mark_code st;
                        Some (cursor + opener_size)

(* The block comment opening at the cursor, recording the comment, as the cursor past
   the opener, or None. *)
let line_scan_block_open st source cursor hi language =
        if language.block_comment_open = "" then None
        else if not (has_prefix_at source cursor hi language.block_comment_open) then None
        else begin
                st.depth <- 1;
                mark_comment st;
                Some (cursor + String.length language.block_comment_open)
        end

(* Reports whether a letter-led opener sits in the middle of an identifier. At a line
   start the preceding byte is the newline (or cursor 0), neither an identifier byte. *)
let verbatim_lead_middle_identifier source cursor (delimiter : verbatim_delimiter) =
        if not (byte_is_identifier delimiter.vopen.[0]) then false
        else if cursor = 0 then false
        else byte_is_identifier source.[cursor - 1]

(* The Rust-style raw-string opener — the lead, then hashes, then a quote — as its
   closer and the opener's byte length, or None. *)
let verbatim_hashable source cursor hi (delimiter : verbatim_delimiter) =
        let scan = ref (cursor + String.length delimiter.vopen) in
        let hashes = ref 0 in
        while !scan < hi && source.[!scan] = '#' do
                incr hashes;
                incr scan
        done;
        if !scan >= hi || source.[!scan] <> '"' then None
        else Some ("\"" ^ String.make !hashes '#', String.length delimiter.vopen + !hashes + 1)

(* The verbatim string opening at the cursor, recording its closer, as the cursor past
   the opener, or None. *)
let rec verbatim_open st source cursor hi = function
        | [] -> None
        | (d : verbatim_delimiter) :: rest ->
                if
                        (not (has_prefix_at source cursor hi d.vopen))
                        || verbatim_lead_middle_identifier source cursor d
                then verbatim_open st source cursor hi rest
                else if not d.hashable then begin
                        st.raw_close <- d.vclose;
                        mark_code st;
                        Some (cursor + String.length d.vopen)
                end
                else
                        (match verbatim_hashable source cursor hi d with
                        | Some (close, opener_size) ->
                                st.raw_close <- close;
                                mark_code st;
                                Some (cursor + opener_size)
                        | None -> verbatim_open st source cursor hi rest)

(* Reports whether an escape sequence begins at the scan position. *)
let quote_escapes_here source scan (delimiter : quote_delimiter) =
        match delimiter.escape with None -> false | Some e -> source.[scan] = e

(* The byte length of a single-line quoted string starting at its opening delimiter,
   stopping at the first unescaped close or end of line. *)
let scan_quoted source cursor hi (delimiter : quote_delimiter) =
        let scan = ref (cursor + String.length delimiter.qopen) in
        let closed = ref None in
        while is_none !closed && !scan < hi do
                if quote_escapes_here source !scan delimiter then scan := !scan + 2
                else if has_prefix_at source !scan hi delimiter.qclose then
                        closed := Some (!scan + String.length delimiter.qclose - cursor)
                else incr scan
        done;
        match !closed with Some length -> length | None -> hi - cursor

(* Reports whether a backslash escape follows the apostrophe. *)
let character_is_escaped source cursor hi =
        cursor + 1 < hi && source.[cursor + 1] = '\\'

(* The length of an escaped character literal, bounded so a stray apostrophe cannot
   scan the whole line. *)
let scan_escaped_character source cursor hi =
        let limit = cursor + 12 in
        let scan = ref (cursor + 3) in
        let closed = ref None in
        while is_none !closed && !scan < hi && !scan <= limit do
                if source.[!scan] = '\'' then closed := Some (!scan - cursor + 1)
                else incr scan
        done;
        match !closed with Some length -> length | None -> 1

(* The byte length of a UTF-8 sequence from its lead byte (1–4); a continuation or
   invalid lead byte counts as 1, matching utf8.DecodeRune. *)
let utf8_lead_length c =
        let b = Char.code c in
        if b < 0x80 then 1
        else if b < 0xC0 then 1
        else if b < 0xE0 then 2
        else if b < 0xF0 then 3
        else if b < 0xF8 then 4
        else 1

(* The length of a single-rune character literal at the cursor, or None when the
   apostrophe does not open one. *)
let simple_character source cursor hi =
        if cursor + 1 >= hi then None
        else if source.[cursor + 1] = '\'' then None
        else
                let size = utf8_lead_length source.[cursor + 1] in
                if cursor + 1 + size >= hi then None
                else if source.[cursor + 1 + size] <> '\'' then None
                else Some (1 + size + 1)

(* The byte length of a character or rune literal, or 1 when the apostrophe is a Rust
   lifetime tick rather than a literal. *)
let scan_character_or_lifetime source cursor hi =
        if character_is_escaped source cursor hi then scan_escaped_character source cursor hi
        else
                match simple_character source cursor hi with
                | Some size -> size
                | None -> 1

(* The single-line string or character literal opening at the cursor, as the cursor
   past it, or None. *)
let rec quote_open st source cursor hi = function
        | [] -> None
        | (d : quote_delimiter) :: rest ->
                if not (has_prefix_at source cursor hi d.qopen) then
                        quote_open st source cursor hi rest
                else begin
                        mark_code st;
                        if d.character_like then
                                Some (cursor + scan_character_or_lifetime source cursor hi)
                        else Some (cursor + scan_quoted source cursor hi d)
                end

(* Dispatches the token at the cursor when not inside a comment or string. A run of
   ordinary non-trigger bytes is skipped in bulk; otherwise each opener is tried in
   order — the first to return [Some next] wins — over one mutable [int option], so the
   per-byte hot path allocates only when something actually opens. *)
let line_scan_fresh st source cursor hi scanner =
        if not scanner.trigger.(Char.code source.[cursor]) then begin
                if byte_is_space source.[cursor] then cursor + 1
                else begin
                        mark_code st;
                        let c = ref (cursor + 1) in
                        while !c < hi && not scanner.trigger.(Char.code source.[!c]) do
                                incr c
                        done;
                        !c
                end
        end
        else begin
                let language = scanner.sc_language in
                let comments = language.line_comment in
                let verbatims = language.verbatim_strings in
                let quotes = language.quote_strings in
                let next = ref (line_scan_block_open st source cursor hi language) in
                if is_none !next then next := line_scan_long_comment st source cursor hi language;
                if is_none !next && line_comment_at source cursor hi comments then begin
                        (* A line comment runs to end of line. *)
                        mark_comment st;
                        next := Some hi
                end;
                if is_none !next then next := line_scan_long_string st source cursor hi language;
                if is_none !next then next := line_scan_heredoc st source cursor hi language;
                if is_none !next then next := verbatim_open st source cursor hi verbatims;
                if is_none !next then next := quote_open st source cursor hi quotes;
                match !next with
                | Some n -> n
                | None ->
                        (* A trigger byte that opened nothing — a lone '/', a shift '<<' — is
                           just code; advance one byte. *)
                        mark_code st;
                        cursor + 1
        end

(* Consumes the token at the cursor, updating the state, returning the next cursor. The
   carried-state branches are checked before the fresh dispatch. *)
let line_scan_step st source cursor hi scanner =
        if st.comment_close <> "" then line_scan_long_comment_body st source cursor hi
        else if st.raw_close <> "" then line_scan_raw st source cursor hi
        else if st.depth > 0 then line_scan_block st source cursor hi scanner.sc_language
        else line_scan_fresh st source cursor hi scanner

(* Reads a line inside a heredoc body: the line is code, and a line equal to the
   terminator ends the heredoc. The only place a line is materialized, and only while
   inside a heredoc body. *)
let classify_heredoc_line st source lo hi =
        let trimmed = String.trim (String.sub source lo (hi - lo)) in
        if trimmed = st.heredoc_close then st.heredoc_close <- "";
        Code

(* Returns the partition of source[lo, hi), mutating st in place — the carried fields
   persist for the next line, the verdict is reset here. *)
let classify_line source lo hi st scanner =
        st.verdict <- Blank;
        if line_is_blank source lo hi then Blank
        else if st.heredoc_close <> "" then classify_heredoc_line st source lo hi
        else begin
                let cursor = ref lo in
                while !cursor < hi do
                        cursor := line_scan_step st source !cursor hi scanner
                done;
                st.verdict
        end

type classify_file_input = { source : string; language : language }

(* Partitions every physical line of the source into code, comment, and blank counts.
   Each line is counted once, so the three sum to the line count. One mutable state and
   three int accumulators; lines are scanned in place as offsets, never copied. *)
let classify_file (input : classify_file_input) =
        let scanner = language_scanner input.language in
        let source = input.source in
        let len = String.length source in
        let st = fresh_state () in
        let code = ref 0 and comment = ref 0 and blank = ref 0 in
        let start = ref 0 in
        for index = 0 to len - 1 do
                if source.[index] = '\n' then begin
                        (match classify_line source !start index st scanner with
                        | Code -> incr code
                        | Comment -> incr comment
                        | Blank -> incr blank);
                        start := index + 1
                end
        done;
        (* Bytes after the last newline are a final line only when non-empty. *)
        if !start < len then
                (match classify_line source !start len st scanner with
                | Code -> incr code
                | Comment -> incr comment
                | Blank -> incr blank);
        { code = !code; comment = !comment; blank = !blank }

(* ===================================================================== *)
(* Count                                                                  *)
(* ===================================================================== *)

(* One counted file: its path, the language read as, its partition, and whether it is
   test code. *)
type file_count = { fc_path : string; fc_language : string; fc_counts : counts; fc_test : bool }

(* The result of counting a tree: one file_count per counted file, in walk order. *)
type report = { files : file_count list }

(* One entry from a directory listing. *)
type dir_entry = { de_name : string; de_is_dir : bool }

(* A read-only filesystem, the stand-in for Go's fs.FS. read_dir lists a directory's
   immediate children ("." is the root); read_file returns a file's bytes. *)
type tree = {
        read_dir : string -> dir_entry list;
        read_file : string -> (string, string) result;
}

(* Reports whether a path, relative to its tree root, is ignored. *)
type ignore_predicate = string -> bool -> bool

type count_input = {
        ci_tree : tree;
        ci_ignored : ignore_predicate option;
        ci_hidden : bool;
        ci_concurrency : int;
}

(* A file the walk selected, and the language to read it as. *)
type candidate = { ca_path : string; ca_language : language }

(* Reports whether a path's final element begins with a dot. *)
let path_is_hidden file_path =
        let base = Filename.basename file_path in
        String.length base > 0 && base.[0] = '.'

(* Reports whether a path is a hidden entry to skip; the root "." is not one. *)
let count_skip_hidden file_path include_hidden =
        if include_hidden then false
        else if file_path = "." then false
        else path_is_hidden file_path

(* Reports whether the injected predicate ignores a path. *)
let count_skip_ignored file_path is_directory is_ignored =
        match is_ignored with None -> false | Some predicate -> predicate file_path is_directory

(* Reports whether content holds a NUL byte in its first chunk. *)
let content_is_binary content =
        let limit = min (String.length content) 8192 in
        let rec go i = i < limit && (content.[i] = '\000' || go (i + 1)) in
        go 0

(* Reports whether needle occurs in haystack; stdlib has no substring search. *)
let string_contains haystack needle =
        let hl = String.length haystack and nl = String.length needle in
        if nl = 0 then true
        else if nl > hl then false
        else
                let rec go i = i + nl <= hl && (String.sub haystack i nl = needle || go (i + 1)) in
                go 0

(* Reports whether any component of a path is a conventional test directory. *)
let path_has_test_directory file_path =
        List.exists
                (fun c -> c = "test" || c = "tests" || c = "spec" || c = "__tests__")
                (String.split_on_char '/' file_path)

(* Reports whether a path is test code: under a test directory, or its base name
   carries the language's test prefix or infix. *)
let path_is_test file_path language =
        if path_has_test_directory file_path then true
        else
                let base = Filename.basename file_path in
                List.exists (fun prefix -> String.starts_with ~prefix base) language.test_prefixes
                || List.exists (fun infix -> string_contains base infix) language.test_infixes

(* Reads and classifies a single candidate, returning None to drop a file that is
   unreadable or binary. *)
let count_one tree (one : candidate) =
        match tree.read_file one.ca_path with
        | Error _ -> None
        | Ok content ->
                if content_is_binary content then None
                else
                        let counts =
                                classify_file { source = content; language = one.ca_language }
                        in
                        Some
                                {
                                        fc_path = one.ca_path;
                                        fc_language = one.ca_language.name;
                                        fc_counts = counts;
                                        fc_test = path_is_test one.ca_path one.ca_language;
                                }

(* Walks the tree and returns the recognized files to count, pruning hidden and
   ignored directories so their contents are never read. Children are visited in
   lexical order, matching Go's fs.WalkDir. *)
let count_candidates tree is_ignored include_hidden =
        let candidates = ref [] in
        let rec walk path =
                let entries = tree.read_dir path in
                let entries =
                        List.sort
                                (fun (a : dir_entry) b -> String.compare a.de_name b.de_name)
                                entries
                in
                List.iter
                        (fun (entry : dir_entry) ->
                                let child =
                                        if path = "." then entry.de_name
                                        else path ^ "/" ^ entry.de_name
                                in
                                if count_skip_hidden child include_hidden then ()
                                else if count_skip_ignored child entry.de_is_dir is_ignored then ()
                                else if entry.de_is_dir then walk child
                                else
                                        match language_for_path child with
                                        | Some language ->
                                                candidates :=
                                                        { ca_path = child; ca_language = language }
                                                        :: !candidates
                                        | None -> ())
                        entries
        in
        walk ".";
        List.rev !candidates

(* Reads and classifies each candidate across Domains, dropping any unreadable or
   binary file, and returns the results in candidate order. Each Domain writes its own
   contiguous chunk of disjoint result slots, so no lock is needed and the order is
   preserved. *)
let count_classify tree candidates concurrency =
        let array = Array.of_list candidates in
        let total = Array.length array in
        if total = 0 then []
        else begin
                let results = Array.make total None in
                let workers = max 1 (min concurrency total) in
                let chunk = (total + workers - 1) / workers in
                let run start =
                        let stop = min total (start + chunk) in
                        for i = start to stop - 1 do
                                results.(i) <- count_one tree array.(i)
                        done
                in
                if workers <= 1 then run 0
                else begin
                        let domains = ref [] in
                        let start = ref 0 in
                        while !start < total do
                                let s = !start in
                                domains := Domain.spawn (fun () -> run s) :: !domains;
                                start := s + chunk
                        done;
                        List.iter Domain.join !domains
                end;
                List.filter_map (fun x -> x) (Array.to_list results)
        end

(* Walks the tree, classifies every recognized file, and returns one file_count per
   file in walk order. *)
let count (input : count_input) =
        let candidates = count_candidates input.ci_tree input.ci_ignored input.ci_hidden in
        { files = count_classify input.ci_tree candidates input.ci_concurrency }

(* ===================================================================== *)
(* Render                                                                 *)
(* ===================================================================== *)

(* Renders a non-negative count with a comma between each group of three digits. *)
let with_thousands_separators value =
        let digits = string_of_int value in
        if String.length digits <= 3 then digits
        else begin
                let lead = match String.length digits mod 3 with 0 -> 3 | n -> n in
                let buffer = Buffer.create 16 in
                Buffer.add_string buffer (String.sub digits 0 lead);
                let index = ref lead in
                while !index < String.length digits do
                        Buffer.add_char buffer ',';
                        Buffer.add_string buffer (String.sub digits !index 3);
                        index := !index + 3
                done;
                Buffer.contents buffer
        end

(* A language's taxonomy category by display name. *)
let language_category name =
        match name with
        | "C" | "C++" | "Rust" | "Zig" | "Odin" | "Ada" | "Fortran" | "Pascal" | "Arduino"
        | "Solidity" ->
                "Systems"
        | "Go" | "Java" | "C#" | "Kotlin" | "Scala" | "Dart" | "Crystal" | "Nim" | "D" | "Swift"
        | "Objective-C" | "Haskell" | "OCaml" | "F#" | "Visual Basic" ->
                "Managed"
        | "Python" | "Ruby" | "JavaScript" | "TypeScript" | "Lua" | "Perl" | "PHP" | "R" | "Tcl"
        | "Julia" | "Groovy" | "Elixir" | "Erlang" | "Clojure" | "Scheme" | "Common Lisp" | "Racket"
        | "Emacs Lisp" ->
                "Dynamically Typed"
        | "Shell" | "PowerShell" | "Fish" | "Nushell" -> "Shell"
        | "HTML" | "XML" | "CSS" | "SCSS" | "LESS" | "Markdown" | "YAML" | "TOML" | "JSONC" | "XAML"
        | "XSLT" | "Vue" | "Svelte" | "Astro" | "Protobuf" | "Thrift" | "TeX" ->
                "Markup & Data"
        | "Makefile" | "Dockerfile" | "CMake" | "HCL" | "Nix" -> "Build & Config"
        | "SQL" -> "Query"
        | "Verilog" | "GLSL" | "HLSL" -> "Hardware"
        | _ -> "Other"

(* The fixed display order of the categories. *)
let category_order =
        [ "Systems"; "Managed"; "Dynamically Typed"; "Shell"; "Markup & Data"; "Build & Config";
          "Query"; "Hardware"; "Other" ]

(* One printable table row; every cell is already a string. *)
type render_row = {
        r_name : string;
        r_files : string;
        r_lines : string;
        r_code : string;
        r_comments : string;
        r_blanks : string;
        r_percent : string;
}

let empty_row =
        { r_name = ""; r_files = ""; r_lines = ""; r_code = ""; r_comments = ""; r_blanks = "";
          r_percent = "" }

let render_header_row =
        { r_name = "Language"; r_files = "Files"; r_lines = "Lines"; r_code = "Code";
          r_comments = "Comments"; r_blanks = "Blanks"; r_percent = "%Code" }

(* One language's files and their summed partition. *)
type language_group = {
        g_name : string;
        g_category : string;
        g_files : int;
        g_counts : counts;
        g_source_files : int;
        g_source : counts;
        g_test_files : int;
        g_test : counts;
        g_members : file_count list;
}

(* A category and the language groups it holds, in name order. *)
type category_group = { cat_name : string; cat_languages : language_group list }

(* Folds a report's files into per-language groups sorted by name, splitting each into
   source and test and preserving its files in report order. *)
let report_groups (report : report) =
        let table = Hashtbl.create 16 in
        let order = ref [] in
        List.iter
                (fun (file : file_count) ->
                        let group =
                                match Hashtbl.find_opt table file.fc_language with
                                | Some g -> g
                                | None ->
                                        let g =
                                                {
                                                        g_name = file.fc_language;
                                                        g_category =
                                                                language_category file.fc_language;
                                                        g_files = 0;
                                                        g_counts = zero_counts;
                                                        g_source_files = 0;
                                                        g_source = zero_counts;
                                                        g_test_files = 0;
                                                        g_test = zero_counts;
                                                        g_members = [];
                                                }
                                        in
                                        order := file.fc_language :: !order;
                                        g
                        in
                        let group = { group with g_files = group.g_files + 1 } in
                        let group =
                                { group with g_counts = counts_add group.g_counts file.fc_counts }
                        in
                        let group =
                                if file.fc_test then
                                        {
                                                group with
                                                g_test_files = group.g_test_files + 1;
                                                g_test = counts_add group.g_test file.fc_counts;
                                        }
                                else
                                        {
                                                group with
                                                g_source_files = group.g_source_files + 1;
                                                g_source = counts_add group.g_source file.fc_counts;
                                        }
                        in
                        let group = { group with g_members = file :: group.g_members } in
                        Hashtbl.replace table file.fc_language group)
                report.files;
        let groups =
                List.rev_map
                        (fun name ->
                                let g = Hashtbl.find table name in
                                { g with g_members = List.rev g.g_members })
                        !order
        in
        List.sort (fun (a : language_group) b -> String.compare a.g_name b.g_name) groups

(* Buckets the name-sorted language groups into categories in the fixed order. *)
let report_categories groups =
        List.map
                (fun name ->
                        {
                                cat_name = name;
                                cat_languages =
                                        List.filter
                                                (fun (g : language_group) -> g.g_category = name)
                                                groups;
                        })
                category_order

type counts_row_input = {
        cri_name : string;
        cri_files : int;
        cri_counts : counts;
        cri_total_code : int;
}

(* Builds an aggregate row: a name, a file count, the partition, and the code's share
   of the given code total — blank when that total is zero. *)
let counts_row (input : counts_row_input) =
        let percent =
                if input.cri_total_code > 0 then
                        let share =
                                float_of_int input.cri_counts.code
                                /. float_of_int input.cri_total_code *. 100.
                        in
                        Printf.sprintf "%.1f%%" share
                else ""
        in
        {
                r_name = input.cri_name;
                r_files = with_thousands_separators input.cri_files;
                r_lines = with_thousands_separators (counts_lines input.cri_counts);
                r_code = with_thousands_separators input.cri_counts.code;
                r_comments = with_thousands_separators input.cri_counts.comment;
                r_blanks = with_thousands_separators input.cri_counts.blank;
                r_percent = percent;
        }

(* Builds a per-file row: like an aggregate row but without a file count or a
   percentage, both per-language facts. *)
let file_row name counts =
        let row =
                counts_row
                        { cri_name = name; cri_files = 0; cri_counts = counts; cri_total_code = 0 }
        in
        { row with r_files = "" }

type split_rows_input = {
        sr_indent : string;
        sr_rows : render_row list;
        sr_source_files : int;
        sr_source : counts;
        sr_test_files : int;
        sr_test : counts;
}

(* Appends indented source and test sub-rows, but only when test files are present.
   The %Code on a sub-row is its share of the group's own code. *)
let split_rows (input : split_rows_input) =
        if input.sr_test_files = 0 then input.sr_rows
        else
                let own_code = input.sr_source.code + input.sr_test.code in
                input.sr_rows
                @ [
                          counts_row
                                  {
                                          cri_name = input.sr_indent ^ "source";
                                          cri_files = input.sr_source_files;
                                          cri_counts = input.sr_source;
                                          cri_total_code = own_code;
                                  };
                          counts_row
                                  {
                                          cri_name = input.sr_indent ^ "tests";
                                          cri_files = input.sr_test_files;
                                          cri_counts = input.sr_test;
                                          cri_total_code = own_code;
                                  };
                  ]

(* Appends a language's indented row, then its files (show_files) or its source and
   test sub-rows. *)
let report_language_rows rows (group : language_group) show_files =
        let rows =
                rows
                @ [
                          counts_row
                                  {
                                          cri_name = "  " ^ group.g_name;
                                          cri_files = group.g_files;
                                          cri_counts = group.g_counts;
                                          cri_total_code = 0;
                                  };
                  ]
        in
        if show_files then
                rows
                @ List.map
                          (fun (member : file_count) ->
                                  file_row ("    " ^ member.fc_path) member.fc_counts)
                          group.g_members
        else
                split_rows
                        {
                                sr_indent = "    ";
                                sr_rows = rows;
                                sr_source_files = group.g_source_files;
                                sr_source = group.g_source;
                                sr_test_files = group.g_test_files;
                                sr_test = group.g_test;
                        }

(* Builds the header, then each non-empty category: a label row followed by its
   languages. *)
let report_rows categories show_files =
        let rows = ref [ render_header_row ] in
        List.iter
                (fun (category : category_group) ->
                        if category.cat_languages <> [] then begin
                                rows := !rows @ [ { empty_row with r_name = category.cat_name } ];
                                List.iter
                                        (fun group ->
                                                rows := report_language_rows !rows group show_files)
                                        category.cat_languages
                        end)
                categories;
        !rows

(* Sums every group into the Total row and its source and test sub-rows. *)
let report_total groups =
        let combined = ref zero_counts in
        let source = ref zero_counts in
        let test = ref zero_counts in
        let files = ref 0 in
        let source_files = ref 0 in
        let test_files = ref 0 in
        List.iter
                (fun (group : language_group) ->
                        combined := counts_add !combined group.g_counts;
                        source := counts_add !source group.g_source;
                        test := counts_add !test group.g_test;
                        files := !files + group.g_files;
                        source_files := !source_files + group.g_source_files;
                        test_files := !test_files + group.g_test_files)
                groups;
        let totals =
                [
                        counts_row
                                {
                                        cri_name = "Total";
                                        cri_files = !files;
                                        cri_counts = !combined;
                                        cri_total_code = 0;
                                };
                ]
        in
        split_rows
                {
                        sr_indent = "  ";
                        sr_rows = totals;
                        sr_source_files = !source_files;
                        sr_source = !source;
                        sr_test_files = !test_files;
                        sr_test = !test;
                }

(* The printed width of each table column. *)
type render_column_widths = {
        w_name : int;
        w_files : int;
        w_lines : int;
        w_code : int;
        w_comments : int;
        w_blanks : int;
        w_percent : int;
}

(* Sizes each column to its widest cell across all rows. *)
let render_widths rows =
        List.fold_left
                (fun w (row : render_row) ->
                        {
                                w_name = max w.w_name (String.length row.r_name);
                                w_files = max w.w_files (String.length row.r_files);
                                w_lines = max w.w_lines (String.length row.r_lines);
                                w_code = max w.w_code (String.length row.r_code);
                                w_comments = max w.w_comments (String.length row.r_comments);
                                w_blanks = max w.w_blanks (String.length row.r_blanks);
                                w_percent = max w.w_percent (String.length row.r_percent);
                        })
                { w_name = 0; w_files = 0; w_lines = 0; w_code = 0; w_comments = 0; w_blanks = 0;
                  w_percent = 0 }
                rows

(* The printed character width of any row, which the rules match. *)
let render_column_widths_total (w : render_column_widths) =
        1 + w.w_name + 2 + w.w_files + 2 + w.w_lines + 2 + w.w_code + 2 + w.w_comments
        + 2 + w.w_blanks + 2 + w.w_percent

(* Counts UTF-8 code points (lead and ASCII bytes), ignoring continuation bytes. *)
let rune_count s =
        let count = ref 0 in
        String.iter (fun c -> if Char.code c < 0x80 || Char.code c >= 0xC0 then incr count) s;
        !count

(* Pads in code points, not bytes, matching Go's fmt %*s width (rune-counted) while
   the column widths are byte-counted — so a non-ASCII path aligns exactly as Go's own
   output does (the spec's documented non-ASCII quirk). For ASCII cells the two agree. *)
let pad_left text width = String.make (max 0 (width - rune_count text)) ' ' ^ text
let pad_right text width = text ^ String.make (max 0 (width - rune_count text)) ' '

(* Lays out one row: a leading space, the left-justified name, then each
   right-justified numeric column behind a two-space gap. *)
let render_row_format (row : render_row) (w : render_column_widths) =
        " " ^ pad_right row.r_name w.w_name
        ^ "  " ^ pad_left row.r_files w.w_files
        ^ "  " ^ pad_left row.r_lines w.w_lines
        ^ "  " ^ pad_left row.r_code w.w_code
        ^ "  " ^ pad_left row.r_comments w.w_comments
        ^ "  " ^ pad_left row.r_blanks w.w_blanks
        ^ "  " ^ pad_left row.r_percent w.w_percent

(* Trims a string's trailing spaces, as a label row leaves padding. *)
let trim_right_spaces line =
        let n = ref (String.length line) in
        while !n > 0 && line.[!n - 1] = ' ' do
                decr n
        done;
        String.sub line 0 !n

type render_input = { report : report; show_files : bool }

(* Writes the report as an aligned table: one row per language grouped by category, a
   Total row, and — with show_files — each file indented under its language. *)
let render (output : string -> unit) (input : render_input) =
        let groups = report_groups input.report in
        let categories = report_categories groups in
        let rows = report_rows categories input.show_files in
        let totals = report_total groups in
        let widths = render_widths (rows @ totals) in
        let rule =
                String.concat "" (List.init (render_column_widths_total widths) (fun _ -> "─"))
        in
        let write line = output (trim_right_spaces line ^ "\n") in
        write rule;
        write (render_row_format (List.hd rows) widths);
        write rule;
        List.iter (fun row -> write (render_row_format row widths)) (List.tl rows);
        write rule;
        List.iter (fun row -> write (render_row_format row widths)) totals;
        write rule

(* ===================================================================== *)
(* JSON                                                                   *)
(* ===================================================================== *)

(* Encodes a JSON string literal the way Go's encoder does with HTML escaping on:
   quotes and backslashes escaped, control bytes as \uXXXX, and <, >, & escaped too
   (so "Markup & Data" becomes "Markup & Data"). *)
let json_escape s =
        let buffer = Buffer.create (String.length s + 2) in
        Buffer.add_char buffer '"';
        String.iter
                (fun c ->
                        match c with
                        | '"' -> Buffer.add_string buffer "\\\""
                        | '\\' -> Buffer.add_string buffer "\\\\"
                        | '\n' -> Buffer.add_string buffer "\\n"
                        | '\r' -> Buffer.add_string buffer "\\r"
                        | '\t' -> Buffer.add_string buffer "\\t"
                        | '<' -> Buffer.add_string buffer "\\u003c"
                        | '>' -> Buffer.add_string buffer "\\u003e"
                        | '&' -> Buffer.add_string buffer "\\u0026"
                        | c when Char.code c < 0x20 ->
                                Buffer.add_string buffer (Printf.sprintf "\\u%04x" (Char.code c))
                        | c -> Buffer.add_char buffer c)
                s;
        Buffer.add_char buffer '"';
        Buffer.contents buffer

(* Emits the report as indented JSON matching Go's encoder: a name-sorted languages
   array carrying each language's category and source/test split, and a total. *)
let render_json (output : string -> unit) (report : report) =
        let buffer = Buffer.create 1024 in
        let add = Buffer.add_string buffer in
        let partition indent files (c : counts) =
                add (Printf.sprintf "%s\"files\": %d,\n" indent files);
                add (Printf.sprintf "%s\"code\": %d,\n" indent c.code);
                add (Printf.sprintf "%s\"comments\": %d,\n" indent c.comment);
                add (Printf.sprintf "%s\"blanks\": %d\n" indent c.blank)
        in
        let groups = report_groups report in
        let total_files = ref 0 in
        let total = ref zero_counts in
        let total_source_files = ref 0 in
        let total_source = ref zero_counts in
        let total_test_files = ref 0 in
        let total_test = ref zero_counts in
        add "{\n";
        add "  \"languages\": [";
        List.iteri
                (fun position (group : language_group) ->
                        total_files := !total_files + group.g_files;
                        total := counts_add !total group.g_counts;
                        total_source_files := !total_source_files + group.g_source_files;
                        total_source := counts_add !total_source group.g_source;
                        total_test_files := !total_test_files + group.g_test_files;
                        total_test := counts_add !total_test group.g_test;
                        if position > 0 then add ",";
                        add "\n    {\n";
                        add (Printf.sprintf "      \"name\": %s,\n" (json_escape group.g_name));
                        add
                                (Printf.sprintf "      \"category\": %s,\n"
                                        (json_escape group.g_category));
                        add (Printf.sprintf "      \"files\": %d,\n" group.g_files);
                        add (Printf.sprintf "      \"code\": %d,\n" group.g_counts.code);
                        add (Printf.sprintf "      \"comments\": %d,\n" group.g_counts.comment);
                        add (Printf.sprintf "      \"blanks\": %d,\n" group.g_counts.blank);
                        add "      \"source\": {\n";
                        partition "        " group.g_source_files group.g_source;
                        add "      },\n";
                        add "      \"tests\": {\n";
                        partition "        " group.g_test_files group.g_test;
                        add "      }\n";
                        add "    }")
                groups;
        if groups <> [] then add "\n  ";
        add "],\n";
        add "  \"total\": {\n";
        add (Printf.sprintf "    \"files\": %d,\n" !total_files);
        add (Printf.sprintf "    \"code\": %d,\n" (!total).code);
        add (Printf.sprintf "    \"comments\": %d,\n" (!total).comment);
        add (Printf.sprintf "    \"blanks\": %d,\n" (!total).blank);
        add "    \"source\": {\n";
        partition "      " !total_source_files !total_source;
        add "    },\n";
        add "    \"tests\": {\n";
        partition "      " !total_test_files !total_test;
        add "    }\n";
        add "  }\n";
        add "}\n";
        output (Buffer.contents buffer)

(* ===================================================================== *)
(* Main                                                                   *)
(* ===================================================================== *)

type sink = { write_string : string -> unit }

type main_input = {
        mi_arguments : string list;
        mi_output : sink;
        mi_error : sink;
        mi_open : string -> tree;
        mi_is_dir : string -> (bool, string) result;
        mi_read_file : string -> (string, string) result;
        mi_ignore_for : string -> ignore_predicate option;
        mi_concurrency : int;
}

type main_scope = { ms_paths : string list; ms_no_ignore : bool; ms_hidden : bool }

(* The resolved flags and positional paths, or an error message on an unknown flag. *)
type parsed_command = {
        pc_paths : string list;
        pc_files : bool;
        pc_no_ignore : bool;
        pc_hidden : bool;
        pc_json : bool;
}

(* Minimal command-line parse: positional paths plus the four boolean flags, each
   given as -flag or --flag. An unknown flag is an error. *)
let parse_command arguments =
        let base =
                { pc_paths = []; pc_files = false; pc_no_ignore = false; pc_hidden = false;
                  pc_json = false }
        in
        let rec go command = function
                | [] -> Ok { command with pc_paths = List.rev command.pc_paths }
                | argument :: rest ->
                        let flag =
                                if String.starts_with ~prefix:"--" argument then
                                        Some (String.sub argument 2 (String.length argument - 2))
                                else if String.starts_with ~prefix:"-" argument
                                        && String.length argument > 1
                                then Some (String.sub argument 1 (String.length argument - 1))
                                else None
                        in
                        (match flag with
                        | None -> go { command with pc_paths = argument :: command.pc_paths } rest
                        | Some "files" -> go { command with pc_files = true } rest
                        | Some "no-ignore" -> go { command with pc_no_ignore = true } rest
                        | Some "hidden" -> go { command with pc_hidden = true } rest
                        | Some "json" -> go { command with pc_json = true } rest
                        | Some other -> Error other)
        in
        go base arguments

(* Classifies one explicitly named file, reporting whether its extension is recognized. *)
let main_file (input : main_input) name =
        match language_for_path name with
        | None -> Ok None
        | Some language -> (
                match input.mi_read_file name with
                | Error e -> Error e
                | Ok content ->
                        let counts = classify_file { source = content; language } in
                        Ok
                                (Some
                                        {
                                                fc_path = name;
                                                fc_language = language.name;
                                                fc_counts = counts;
                                                fc_test = false;
                                        }))

(* Builds the ignore filter for a root, or None when ignoring is off. *)
let main_ignore (input : main_input) root no_ignore =
        if no_ignore then None else input.mi_ignore_for root

(* Prefixes a root onto a file's relative path the way Go's path.Join does: a "."
   root yields the path unchanged (no "./" prefix), and a leading "./" or trailing
   "/" on the root is dropped. *)
let main_join root rel =
        let root =
                if String.length root > 1 && root.[String.length root - 1] = '/' then
                        String.sub root 0 (String.length root - 1)
                else root
        in
        let root =
                if String.starts_with ~prefix:"./" root then
                        String.sub root 2 (String.length root - 2)
                else root
        in
        if root = "" || root = "." then rel else root ^ "/" ^ rel

(* Walks one directory, prefixing each file path with the root. *)
let main_directory (input : main_input) root (scope : main_scope) =
        let report =
                count
                        {
                                ci_tree = input.mi_open root;
                                ci_ignored = main_ignore input root scope.ms_no_ignore;
                                ci_hidden = scope.ms_hidden;
                                ci_concurrency = input.mi_concurrency;
                        }
        in
        Ok
                (List.map
                        (fun (file : file_count) ->
                                { file with fc_path = main_join root file.fc_path })
                        report.files)

(* Counts a single path: a directory is walked, a file is classified directly. *)
let main_one (input : main_input) root scope =
        match input.mi_is_dir root with
        | Error e -> Error e
        | Ok true -> main_directory input root scope
        | Ok false -> (
                match main_file input root with
                | Error e -> Error e
                | Ok None -> Ok []
                | Ok (Some file) -> Ok [ file ])

(* Counts every path and merges the results into one file list. *)
let main_collect (input : main_input) (scope : main_scope) =
        let rec go accumulator = function
                | [] -> Ok (List.concat (List.rev accumulator))
                | root :: rest ->
                        let trimmed = String.trim root in
                        if trimmed = "" then go accumulator rest
                        else (
                                match main_one input trimmed scope with
                                | Error e -> Error e
                                | Ok files -> go (files :: accumulator) rest)
        in
        go [] scope.ms_paths

let usage_text =
        "usage: sloc [paths...] [--files] [--no-ignore] [--hidden] [--json]\n"
        ^ "count lines of code, comments, and blanks\n"

(* Parses the command line, counts every path, and renders the table or JSON,
   returning a process exit code. *)
let main (input : main_input) =
        (* Arguments include the program name (os.Args); drop it before parsing. *)
        let args = match input.mi_arguments with _ :: rest -> rest | [] -> [] in
        match parse_command args with
        | Error flag ->
                input.mi_error.write_string (Printf.sprintf "sloc: unknown flag: -%s\n\n" flag);
                input.mi_error.write_string usage_text;
                2
        | Ok command ->
                let paths = if command.pc_paths = [] then [ "." ] else command.pc_paths in
                let scope =
                        { ms_paths = paths; ms_no_ignore = command.pc_no_ignore;
                          ms_hidden = command.pc_hidden }
                in
                (match main_collect input scope with
                | Error e ->
                        input.mi_error.write_string (Printf.sprintf "sloc: %s\n" e);
                        1
                | Ok files ->
                        let report = { files } in
                        if command.pc_json then render_json input.mi_output.write_string report
                        else
                                render input.mi_output.write_string
                                        { report; show_files = command.pc_files };
                        0)

(* ===================================================================== *)
(* Composition root — the one place that binds the real OS.               *)
(* ===================================================================== *)

let exit_usage = 2
let file_bytes_max = 64 * 1024 * 1024

(* The OS boundary: OCaml's stdlib and Unix report a failed file or directory
   operation only by raising, so each binding runs inside this wrapper, which turns
   that raise into the Error value the program consumes. *)
let os_result thunk =
        match thunk () with
        | value -> Ok value
        | exception Unix.Unix_error (code, _, _) -> Error (Unix.error_message code)
        | exception Sys_error message -> Error message
        | exception End_of_file -> Error "unexpected end of file"

(* Reads a file's bytes, bounded so a pathological file cannot be slurped whole. *)
let read_file_bounded path =
        let opened =
                os_result (fun () ->
                        let channel = open_in_bin path in
                        (channel, in_channel_length channel))
        in
        match opened with
        | Error _ as failed -> failed
        | Ok (channel, length) ->
                let length = min length file_bytes_max in
                os_result (fun () ->
                        let data = really_input_string channel length in
                        close_in channel;
                        data)

(* The real read-only tree over an on-disk directory rooted at [base]. read_file is
   bounded so a pathological file cannot be slurped whole. *)
let tree_of_directory base =
        let resolve relative = if relative = "." then base else Filename.concat base relative in
        let read_dir relative =
                match os_result (fun () -> Sys.readdir (resolve relative)) with
                | Error _ -> []
                | Ok entries ->
                        let to_entry name =
                                (* lstat, not stat: a symlink is a non-directory entry, so a
                                   symlinked directory is never descended, matching fs.WalkDir. *)
                                let full = Filename.concat (resolve relative) name in
                                let is_dir =
                                        match
                                                os_result (fun () -> (Unix.lstat full).Unix.st_kind)
                                        with
                                        | Ok Unix.S_DIR -> true
                                        | _ -> false
                                in
                                { de_name = name; de_is_dir = is_dir }
                        in
                        List.map to_entry (Array.to_list entries)
        in
        let read_file relative = read_file_bounded (resolve relative) in
        { read_dir; read_file }

(* Reads a command's stdout, or None when it exits non-zero. *)
let command_output argv =
        match
                os_result (fun () ->
                        let input = Unix.open_process_args_in argv.(0) argv in
                        let output = In_channel.input_all input in
                        (output, Unix.close_process_in input))
        with
        | Ok (output, Unix.WEXITED 0) -> Some output
        | _ -> None

(* Builds an ignore predicate for a git work tree: a path is ignored when git would
   not list it. None when the root is not a git tree or git is unavailable. *)
let git_ignore root =
        let argv =
                [|
                        "git"; "-C"; root; "ls-files"; "-z"; "--cached"; "--others";
                        "--exclude-standard"; "--"; "."
                |]
        in
        match command_output argv with
        | None -> None
        | Some output ->
                let kept_files = Hashtbl.create 256 in
                let kept_dirs = Hashtbl.create 256 in
                Hashtbl.replace kept_dirs "." true;
                let add_ancestors name =
                        let rec go parent =
                                if parent <> "." && parent <> "/" && parent <> "." then begin
                                        Hashtbl.replace kept_dirs parent true;
                                        go (Filename.dirname parent)
                                end
                        in
                        go (Filename.dirname name)
                in
                List.iter
                        (fun name ->
                                if name <> "" then begin
                                        Hashtbl.replace kept_files name true;
                                        add_ancestors name
                                end)
                        (String.split_on_char '\000' output);
                Some
                        (fun relative_path is_directory ->
                                if is_directory then not (Hashtbl.mem kept_dirs relative_path)
                                else not (Hashtbl.mem kept_files relative_path))

(* The composition root run: bind the real filesystem and git, then call main. *)
let run () =
        let arguments = Array.to_list Sys.argv in
        let sink_of channel = { write_string = (fun s -> output_string channel s) } in
        main
                {
                        mi_arguments = arguments;
                        mi_output = sink_of stdout;
                        mi_error = sink_of stderr;
                        mi_open = tree_of_directory;
                        mi_is_dir =
                                (fun name -> os_result (fun () -> Sys.is_directory name));
                        mi_read_file = read_file_bounded;
                        mi_ignore_for = git_ignore;
                        mi_concurrency = Domain.recommended_domain_count ();
                }

(* Run the program only as the sloc_ocaml binary; sloc_test links these definitions
   and executes under its own name, so it never runs main at startup. *)
let () =
        ignore exit_usage;
        if Filename.basename Sys.argv.(0) <> "sloc_test" then exit (run ())
