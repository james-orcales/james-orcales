use shared_rs::cli;

fn args(tokens: &[&str]) -> Vec<String> {
    tokens.iter().map(|s| s.to_string()).collect()
}

fn todoctl_add_command() -> cli::Command {
    cli::Command {
        label: "add".to_string(),
        description: String::new(),
        arguments: vec![cli::new_string_argument("task", "describes what you need to do")],
        flags: vec![
            cli::new_string_flag("deadline", "deadline date", "Jan 02 Mon"),
            cli::new_string_flag("priority", "set priority", "low"),
            cli::new_bool_flag("noop", "random flag that does nothing", false),
        ],
    }
}

fn todoctl_list_command() -> cli::Command {
    cli::Command {
        label: "list".to_string(),
        description: String::new(),
        arguments: vec![],
        flags: vec![
            cli::new_string_flag("columns", "comma-separated", "all"),
            cli::new_int_flag("count", "how many tasks to display", 0),
        ],
    }
}

fn todoctl_delete_command() -> cli::Command {
    cli::Command {
        label: "delete".to_string(),
        description: "remove a task by ID".to_string(),
        arguments: vec![cli::new_int_argument("id", "the integer index of the task to remove")],
        flags: vec![],
    }
}

fn todoctl_program() -> cli::Program {
    cli::new(
        "todoctl",
        "is a todo list manager",
        vec![
            cli::Command {
                label: "help".to_string(),
                description: "print help message".to_string(),
                arguments: vec![],
                flags: vec![],
            },
            todoctl_add_command(),
            todoctl_list_command(),
            todoctl_delete_command(),
        ],
        vec![],
    )
}

// Parse > Commands:

#[test]
fn parse_named_command_resolves() {
    let program = todoctl_program();
    let outcome = cli::program_parse(&program, &args(&["todoctl", "list"])).unwrap();
    assert_eq!(outcome.command.label, "list");
}

#[test]
fn parse_absent_command_defaults_to_first() {
    let program = todoctl_program();
    let outcome = cli::program_parse(&program, &args(&["todoctl"])).unwrap();
    assert_eq!(outcome.command.label, "help");
}

#[test]
fn parse_unknown_command_errors() {
    let program = todoctl_program();
    assert!(cli::program_parse(&program, &args(&["todoctl", "bogus"])).is_err());
}

#[test]
fn parse_unknown_command_suggests_near_miss() {
    let program = todoctl_program();
    let error = cli::program_parse(&program, &args(&["todoctl", "lst"])).unwrap_err();
    assert!(cli::parse_error_message(&error).contains("did you mean \"list\""));
}

#[test]
fn parse_unknown_command_no_suggestion_for_wild_miss() {
    let program = todoctl_program();
    let error = cli::program_parse(&program, &args(&["todoctl", "zzzzzzzz"])).unwrap_err();
    assert!(!cli::parse_error_message(&error).contains("did you mean"));
}

// Parse > Single Command:

fn sloc_program() -> cli::Program {
    cli::new_single(
        "sloc",
        "count lines of code",
        vec![cli::new_string_argument("path", "directory to scan")],
        vec![cli::new_bool_flag("hidden", "include hidden dot-files", false)],
    )
}

#[test]
fn parse_single_command_reads_first_token_as_positional() {
    let program = sloc_program();
    let outcome = cli::program_parse(&program, &args(&["sloc", "./src"])).unwrap();
    let value = cli::get_option(&outcome.command.arguments, "path", |p| p.value.clone());
    assert_eq!(value, cli::Parameter_Value::Str("./src".to_string()));
}

#[test]
fn parse_single_command_reads_flag() {
    let program = sloc_program();
    let outcome = cli::program_parse(&program, &args(&["sloc", "./src", "-hidden"])).unwrap();
    let value = cli::get_option(&outcome.command.flags, "hidden", |p| p.value.clone());
    assert_eq!(value, cli::Parameter_Value::Bool(true));
}

#[test]
fn parse_single_command_token_matching_a_sibling_name_is_positional() {
    let program = sloc_program();
    let outcome = cli::program_parse(&program, &args(&["sloc", "help"])).unwrap();
    let value = cli::get_option(&outcome.command.arguments, "path", |p| p.value.clone());
    assert_eq!(value, cli::Parameter_Value::Str("help".to_string()));
}

#[test]
fn parse_single_command_missing_positional_errors() {
    let program = sloc_program();
    assert!(cli::program_parse(&program, &args(&["sloc"])).is_err());
}

// Parse > Arguments:

#[test]
fn parse_missing_argument_errors() {
    let program = todoctl_program();
    assert!(cli::program_parse(&program, &args(&["todoctl", "add"])).is_err());
}

#[test]
fn parse_int_argument_converts() {
    let program = todoctl_program();
    let outcome = cli::program_parse(&program, &args(&["todoctl", "delete", "3"])).unwrap();
    let value = cli::get_option(&outcome.command.arguments, "id", |p| p.value.clone());
    assert_eq!(value, cli::Parameter_Value::Int(3));
}

#[test]
fn parse_non_numeric_int_argument_errors() {
    let program = todoctl_program();
    assert!(cli::program_parse(&program, &args(&["todoctl", "delete", "abc"])).is_err());
}

// Parse > Named:

#[test]
fn parse_argument_settable_by_name() {
    let program = todoctl_program();
    let outcome = cli::program_parse(&program, &args(&["todoctl", "add", "-task=hello"])).unwrap();
    let value = cli::get_option(&outcome.command.arguments, "task", |p| p.value.clone());
    assert_eq!(value, cli::Parameter_Value::Str("hello".to_string()));
}

#[test]
fn parse_named_and_positional_interleave() {
    let program = todoctl_program();
    let outcome =
        cli::program_parse(&program, &args(&["todoctl", "add", "-priority=high", "world"])).unwrap();
    let task = cli::get_option(&outcome.command.arguments, "task", |p| p.value.clone());
    assert_eq!(task, cli::Parameter_Value::Str("world".to_string()));
    let priority = cli::get_option(&outcome.command.flags, "priority", |p| p.value.clone());
    assert_eq!(priority, cli::Parameter_Value::Str("high".to_string()));
}

#[test]
fn parse_positional_skips_argument_already_named() {
    let program = cli::new_single(
        "pair",
        "two values",
        vec![cli::new_string_argument("first", ""), cli::new_string_argument("second", "")],
        vec![],
    );
    let outcome = cli::program_parse(&program, &args(&["pair", "-first=x", "y"])).unwrap();
    let first = cli::get_option(&outcome.command.arguments, "first", |p| p.value.clone());
    assert_eq!(first, cli::Parameter_Value::Str("x".to_string()));
    let second = cli::get_option(&outcome.command.arguments, "second", |p| p.value.clone());
    assert_eq!(second, cli::Parameter_Value::Str("y".to_string()));
}

#[test]
fn parse_scalar_set_twice_errors() {
    let program = todoctl_program();
    let result = cli::program_parse(
        &program,
        &args(&["todoctl", "add", "task", "-priority=high", "-priority=low"]),
    );
    assert!(result.is_err());
}

#[test]
fn parse_unknown_option_errors() {
    let program = todoctl_program();
    assert!(cli::program_parse(&program, &args(&["todoctl", "add", "task", "-zzz=1"])).is_err());
}

#[test]
fn parse_unknown_option_suggests_near_miss() {
    let program = todoctl_program();
    let error =
        cli::program_parse(&program, &args(&["todoctl", "add", "task", "-priorty=high"])).unwrap_err();
    assert!(cli::parse_error_message(&error).contains("did you mean -priority"));
}

// Parse > Variadic:

fn variadic_sloc_program() -> cli::Program {
    cli::new_single(
        "sloc",
        "count lines of code",
        vec![cli::new_string_variadic("path", "")],
        vec![cli::new_bool_flag("hidden", "", false)],
    )
}

fn assert_variadic_strs(program: &cli::Program, tokens: &[&str], label: &str, want: &[&str]) {
    let outcome = cli::program_parse(program, &args(tokens)).unwrap();
    let got = cli::get_option(&outcome.command.arguments, label, |p| p.value.clone());
    let want_value = cli::Parameter_Value::Str_Variadic(want.iter().map(|s| s.to_string()).collect());
    assert_eq!(got, want_value);
}

#[test]
fn parse_variadic_collects_trailing_positionals() {
    let program = variadic_sloc_program();
    assert_variadic_strs(&program, &["sloc", "a", "b", "c"], "path", &["a", "b", "c"]);
}

#[test]
fn parse_variadic_zero_positionals_is_empty() {
    let program = variadic_sloc_program();
    assert_variadic_strs(&program, &["sloc"], "path", &[]);
}

#[test]
fn parse_variadic_stops_at_a_flag_it_does_not_own() {
    let program = variadic_sloc_program();
    assert_variadic_strs(&program, &["sloc", "a", "b", "-hidden"], "path", &["a", "b"]);
}

#[test]
fn parse_variadic_repeated_named_append() {
    let program = variadic_sloc_program();
    assert_variadic_strs(&program, &["sloc", "-path=a", "-path=b"], "path", &["a", "b"]);
}

#[test]
fn parse_variadic_merges_positional_and_named_in_token_order() {
    let program = variadic_sloc_program();
    assert_variadic_strs(&program, &["sloc", "x", "-path=a"], "path", &["x", "a"]);
}

#[test]
fn parse_variadic_scalar_may_precede_the_slice() {
    let program = cli::new(
        "fileutil",
        "file utilities",
        vec![cli::Command {
            label: "cp".to_string(),
            description: "copy files".to_string(),
            arguments: vec![cli::new_string_argument("dest", ""), cli::new_string_variadic("source", "")],
            flags: vec![],
        }],
        vec![],
    );
    let outcome = cli::program_parse(&program, &args(&["fileutil", "cp", "d", "s1", "s2"])).unwrap();
    let source = cli::get_option(&outcome.command.arguments, "source", |p| p.value.clone());
    assert_eq!(source, cli::Parameter_Value::Str_Variadic(vec!["s1".to_string(), "s2".to_string()]));

    let outcome = cli::program_parse(&program, &args(&["fileutil", "cp", "a", "b", "-dest=/tmp"])).unwrap();
    let source = cli::get_option(&outcome.command.arguments, "source", |p| p.value.clone());
    assert_eq!(source, cli::Parameter_Value::Str_Variadic(vec!["a".to_string(), "b".to_string()]));

    assert!(cli::program_parse(&program, &args(&["fileutil", "cp"])).is_err());
}

#[test]
fn parse_variadic_int_slice_converts_and_errors_on_bad_element() {
    let program = cli::new_single("sum", "add numbers", vec![cli::new_int_variadic("n", "")], vec![]);
    let outcome = cli::program_parse(&program, &args(&["sum", "1", "2", "3"])).unwrap();
    let value = cli::get_option(&outcome.command.arguments, "n", |p| p.value.clone());
    assert_eq!(value, cli::Parameter_Value::Int_Variadic(vec![1, 2, 3]));
    assert!(cli::program_parse(&program, &args(&["sum", "1", "x"])).is_err());
}

// Parse > Flags:

#[test]
fn parse_flag_assignment_by_type() {
    let program = todoctl_program();
    let outcome =
        cli::program_parse(&program, &args(&["todoctl", "add", "task", "-priority=high"])).unwrap();
    let value = cli::get_option(&outcome.command.flags, "priority", |p| p.value.clone());
    assert_eq!(value, cli::Parameter_Value::Str("high".to_string()));
}

#[test]
fn parse_unknown_flag_errors() {
    let program = todoctl_program();
    assert!(cli::program_parse(&program, &args(&["todoctl", "add", "task", "-bogus=1"])).is_err());
}

#[test]
fn parse_double_dash_flag_errors() {
    let program = todoctl_program();
    assert!(cli::program_parse(&program, &args(&["todoctl", "add", "task", "--priority=high"])).is_err());
}

#[test]
fn parse_non_boolean_flag_without_value_errors() {
    let program = todoctl_program();
    assert!(cli::program_parse(&program, &args(&["todoctl", "add", "task", "-priority"])).is_err());
}

// Trim Quotes:

fn quote_test_program() -> cli::Program {
    cli::new(
        "prog",
        "test program",
        vec![cli::Command {
            label: "add".to_string(),
            description: String::new(),
            arguments: vec![cli::new_string_argument("task", "")],
            flags: vec![cli::new_string_flag("flag", "", "")],
        }],
        vec![],
    )
}

fn assert_trim_quotes(raw: &str, want: &str) {
    let program = quote_test_program();
    let outcome = cli::program_parse(&program, &args(&["prog", "add", "task", raw])).unwrap();
    let value = cli::get_option(&outcome.command.flags, "flag", |p| p.value.clone());
    assert_eq!(value, cli::Parameter_Value::Str(want.to_string()));
}

#[test]
fn trim_quotes_double() {
    assert_trim_quotes(r#"-flag="value""#, "value");
}

#[test]
fn trim_quotes_single() {
    assert_trim_quotes("-flag='value'", "value");
}

#[test]
fn trim_quotes_none() {
    assert_trim_quotes("-flag=value", "value");
}

#[test]
fn trim_quotes_empty_double() {
    assert_trim_quotes(r#"-flag="""#, "");
}

#[test]
fn trim_quotes_mismatched() {
    assert_trim_quotes(r#"-flag="value'"#, "\"value'");
}

#[test]
fn trim_quotes_spaces() {
    assert_trim_quotes(r#"-flag="hello world""#, "hello world");
}

// Get Option:

#[test]
fn get_option_present_label_found() {
    let options = vec![cli::new_string_flag("a", "", "x"), cli::new_string_flag("b", "", "y")];
    let value = cli::get_option(&options, "b", |p| p.value.clone());
    assert_eq!(value, cli::Parameter_Value::Str("y".to_string()));
}

#[test]
#[should_panic]
fn get_option_absent_label_panics() {
    let options = vec![cli::new_string_flag("a", "", "x")];
    cli::get_option(&options, "absent", |p| p.value.clone());
}

// Help:

#[test]
fn print_help_shows_program_header_and_commands() {
    let program = todoctl_program();
    let help = cli::print_help(&program);
    assert!(help.starts_with("todoctl is a todo list manager\n\n"));
    assert!(help.contains("Usage:"));
    assert!(help.contains("todoctl <command> <arguments> [-flags[=value]]"));
    assert!(help.contains("Available Commands:"));
    assert!(help.contains("\x1b[34madd\x1b[0m"));
    assert!(help.contains("<task: string>"));
    assert!(help.contains("-deadline=string"));
    assert!(help.contains("(default: Jan 02 Mon)"));
    assert!(help.contains("deadline date"));
    assert!(help.contains("-noop"));
    assert!(help.contains("<id: int>"));
}

#[test]
fn print_help_single_drops_command_selector() {
    let program = sloc_program();
    let help = cli::print_help(&program);
    assert!(!help.contains("<command>"));
    assert!(help.contains("sloc <path: string>"));
}

#[test]
fn print_help_variadic_shows_ellipsis_not_raw_slice_type() {
    let program = variadic_sloc_program();
    let help = cli::print_help(&program);
    assert!(help.contains("<path: string...>"));
    assert!(!help.contains("[]string"));
    assert!(!help.contains("Str_Variadic"));
}

// Demo (end-to-end, mirroring Go's todoctl fixture).

struct Todoctl_State {
    database: Vec<String>,
}

fn run_command(command: &cli::Command, state: &mut Todoctl_State) -> Vec<String> {
    match command.label.as_str() {
        "add" => run_add(command, state),
        "delete" => run_delete(command, state),
        "list" => run_list(command, state),
        _ => Vec::new(),
    }
}

fn run_add(command: &cli::Command, state: &mut Todoctl_State) -> Vec<String> {
    let task = string_value(&command.arguments, "task");
    let deadline = string_value(&command.flags, "deadline");
    let priority = string_value(&command.flags, "priority");
    state.database.push(format!("{deadline} | {priority} | {task}"));
    Vec::new()
}

fn string_value(parameters: &[cli::Parameter], label: &str) -> String {
    match cli::get_option(parameters, label, |p| p.value.clone()) {
        cli::Parameter_Value::Str(text) => text,
        _ => panic!("expected a string value"),
    }
}

fn run_delete(command: &cli::Command, state: &mut Todoctl_State) -> Vec<String> {
    let id = match cli::get_option(&command.arguments, "id", |p| p.value.clone()) {
        cli::Parameter_Value::Int(id) => id as usize,
        _ => panic!("expected an int value"),
    };
    let removed = state.database.remove(id);
    vec![format!("Deleted task {id}: {removed}")]
}

fn run_list(command: &cli::Command, state: &Todoctl_State) -> Vec<String> {
    let count = match cli::get_option(&command.flags, "count", |p| p.value.clone()) {
        cli::Parameter_Value::Int(count) => count as usize,
        _ => panic!("expected an int value"),
    };
    state.database.iter().take(count).cloned().collect()
}

#[test]
fn demo_add_list_delete_end_to_end() {
    let program = todoctl_program();
    let mut state = Todoctl_State { database: Vec::new() };

    let commands: &[&[&str]] = &[
        &["todoctl", "add", "commit to github", "-deadline=Nov 21 Fri"],
        &["todoctl", "add", "something important", "-noop"],
        &["todoctl", "add", "foo bar baz"],
    ];
    for tokens in commands {
        let outcome = cli::program_parse(&program, &args(tokens)).unwrap();
        run_command(&outcome.command, &mut state);
    }
    assert_eq!(
        state.database,
        vec![
            "Nov 21 Fri | low | commit to github".to_string(),
            "Jan 02 Mon | low | something important".to_string(),
            "Jan 02 Mon | low | foo bar baz".to_string(),
        ]
    );

    let outcome = cli::program_parse(&program, &args(&["todoctl", "list", "-count=2"])).unwrap();
    let listed = run_command(&outcome.command, &mut state);
    assert_eq!(
        listed,
        vec![
            "Nov 21 Fri | low | commit to github".to_string(),
            "Jan 02 Mon | low | something important".to_string(),
        ]
    );

    let outcome = cli::program_parse(&program, &args(&["todoctl", "delete", "1"])).unwrap();
    let deleted = run_command(&outcome.command, &mut state);
    assert_eq!(deleted, vec!["Deleted task 1: Jan 02 Mon | low | something important".to_string()]);
    assert_eq!(
        state.database,
        vec!["Nov 21 Fri | low | commit to github".to_string(), "Jan 02 Mon | low | foo bar baz".to_string()]
    );
}

#[test]
fn new_builds_a_program_with_given_commands() {
    let program = cli::new(
        "prog",
        "test program",
        vec![cli::Command {
            label: "add".to_string(),
            description: String::new(),
            arguments: vec![],
            flags: vec![],
        }],
        vec![],
    );
    assert_eq!(program.label, "prog");
    assert_eq!(program.commands.len(), 1);
    assert!(!program.single);
}

#[test]
fn constructors_set_expected_value_variants() {
    let argument = cli::new_string_argument("task", "desc");
    assert!(matches!(argument.value, cli::Parameter_Value::Str(ref s) if s.is_empty()));
    assert!(!argument.is_flag);

    let flag = cli::new_bool_flag("hidden", "desc", false);
    assert!(matches!(flag.value, cli::Parameter_Value::Bool(false)));
    assert!(flag.is_flag);

    let variadic = cli::new_string_variadic("path", "desc");
    assert!(matches!(variadic.value, cli::Parameter_Value::Str_Variadic(ref v) if v.is_empty()));
}

#[test]
#[should_panic]
fn new_command_without_a_label_panics() {
    cli::new(
        "prog",
        "test",
        vec![cli::Command {
            label: String::new(),
            description: String::new(),
            arguments: vec![],
            flags: vec![],
        }],
        vec![],
    );
}

#[test]
#[should_panic]
fn new_single_flag_with_underscore_panics() {
    cli::new_single("sloc", "count lines of code", vec![], vec![cli::new_bool_flag("no_ignore", "", false)]);
}

#[test]
#[should_panic]
fn new_single_argument_with_underscore_panics() {
    cli::new_single("sloc", "count lines of code", vec![cli::new_string_argument("bad_label", "")], vec![]);
}

#[test]
#[should_panic]
fn new_single_argument_colliding_with_flag_panics() {
    cli::new_single(
        "sloc",
        "count lines of code",
        vec![cli::new_string_argument("dup", "")],
        vec![cli::new_bool_flag("dup", "", false)],
    );
}

#[test]
#[should_panic]
fn new_single_non_terminal_slice_argument_panics() {
    cli::new_single(
        "x",
        "test",
        vec![cli::new_string_variadic("a", ""), cli::new_string_argument("b", "")],
        vec![],
    );
}
