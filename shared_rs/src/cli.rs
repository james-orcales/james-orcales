//! A minimal command-line interface parser: commands with positional
//! arguments and optional flags. Every argument and flag is also settable by
//! `-label=value`, in any order with the positionals; a command's last
//! argument may be variadic, collecting the trailing positionals plus every
//! repeated `-label=value`. Build a [`Program`] with [`new`] (commands,
//! selected by name, default to the first) or [`new_single`] (no selector:
//! the first token is the first positional, as in `sloc ./src`), parse with
//! [`program_parse`], and render help with [`print_help`].
//!
//! ```
//! use shared_rs::cli;
//! let program = cli::new_single(
//!     "sloc",
//!     "count lines of code",
//!     vec![cli::new_string_argument("path", "directory to scan")],
//!     vec![cli::new_bool_flag("hidden", "include hidden dot-files", false)],
//! );
//! let arguments: Vec<String> = ["sloc", "./src", "-hidden"].iter().map(|s| s.to_string()).collect();
//! let outcome = cli::program_parse(&program, &arguments).unwrap();
//! let path = cli::get_option(&outcome.command.arguments, "path", |p| p.value.clone());
//! assert_eq!(path, cli::Parameter_Value::Str("./src".to_string()));
//! assert!(cli::print_help(&program).contains("sloc <path: string>"));
//! ```

use std::collections;
use std::iter;

use crate::levenshtein;

/// The value an argument or flag carries: the default before parsing, the
/// user's value after. A scalar `Str`/`Int`/`Bool`, or — for a command's
/// trailing argument only — a variadic `Str_Variadic`/`Int_Variadic`
/// collecting every positional left over plus each repeated `-label=value`.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum Parameter_Value {
    Str(String),
    Int(i64),
    Bool(bool),
    Str_Variadic(Vec<String>),
    Int_Variadic(Vec<i64>),
}

/// A single argument or flag belonging to a [`Command`]. Renamed from the Go
/// original's `Option` to avoid colliding with `std::option::Option`.
#[derive(Clone, Debug)]
pub struct Parameter {
    pub label: String,
    pub description: String,
    pub value: Parameter_Value,
    pub is_flag: bool,
}

/// A single command within a [`Program`]: a label typed on the command line,
/// ordered required arguments (except a variadic last argument, which is
/// optional), and unordered optional flags.
#[derive(Clone, Debug)]
pub struct Command {
    pub label: String,
    pub description: String,
    pub arguments: Vec<Parameter>,
    pub flags: Vec<Parameter>,
}

/// A command-line application: one or more commands, plus flags accepted by
/// every command. `single` marks a program built by [`new_single`]: it has no
/// command selector, so the first token after the program name is the first
/// positional argument, not a command name.
#[derive(Clone, Debug)]
pub struct Program {
    pub label: String,
    pub description: String,
    pub commands: Vec<Command>,
    pub global_flags: Vec<Parameter>,
    pub single: bool,
}

/// A required, ordered positional argument holding text. Arguments always
/// start at their zero value; custom defaults are not allowed.
pub fn new_string_argument(label: &str, description: &str) -> Parameter {
    Parameter {
        label: label.to_string(),
        description: description.to_string(),
        value: Parameter_Value::Str(String::new()),
        is_flag: false,
    }
}

/// A required, ordered positional argument holding a whole number.
pub fn new_int_argument(label: &str, description: &str) -> Parameter {
    Parameter {
        label: label.to_string(),
        description: description.to_string(),
        value: Parameter_Value::Int(0),
        is_flag: false,
    }
}

/// A command's trailing argument, collecting the positionals left after the
/// scalar arguments plus each repeated `-label=value`. Always optional — an
/// empty list is valid, not an error. Must be the last argument declared.
pub fn new_string_variadic(label: &str, description: &str) -> Parameter {
    Parameter {
        label: label.to_string(),
        description: description.to_string(),
        value: Parameter_Value::Str_Variadic(Vec::new()),
        is_flag: false,
    }
}

/// A variadic trailing argument whose elements convert to whole numbers.
pub fn new_int_variadic(label: &str, description: &str) -> Parameter {
    Parameter {
        label: label.to_string(),
        description: description.to_string(),
        value: Parameter_Value::Int_Variadic(Vec::new()),
        is_flag: false,
    }
}

/// An optional, unordered flag with a text default.
pub fn new_string_flag(label: &str, description: &str, default_value: &str) -> Parameter {
    Parameter {
        label: label.to_string(),
        description: description.to_string(),
        value: Parameter_Value::Str(default_value.to_string()),
        is_flag: true,
    }
}

/// An optional, unordered flag with a whole-number default.
pub fn new_int_flag(label: &str, description: &str, default_value: i64) -> Parameter {
    Parameter {
        label: label.to_string(),
        description: description.to_string(),
        value: Parameter_Value::Int(default_value),
        is_flag: true,
    }
}

/// An optional, unordered flag with a boolean default. A boolean flag needs
/// no value on the command line: its bare presence (`-name`) sets it true.
pub fn new_bool_flag(label: &str, description: &str, default_value: bool) -> Parameter {
    Parameter {
        label: label.to_string(),
        description: description.to_string(),
        value: Parameter_Value::Bool(default_value),
        is_flag: true,
    }
}

/// Builds a [`Program`] from its commands and global flags, validating that
/// every command has a label and every argument/flag is well formed. Panics
/// when validation fails — malformed static configuration is a programmer
/// error, not a runtime condition to recover from.
pub fn new(label: &str, description: &str, commands: Vec<Command>, global_flags: Vec<Parameter>) -> Program {
    assert!(!commands.is_empty(), "Program has zero commands specified.");
    global_flags.iter().enumerate().for_each(|(index, flag)| {
        assert!(flag.is_flag, "Global flag #{index} must be created with a flag constructor.");
        validate_flag(&format!("Global flag #{index}"), flag);
    });
    commands.iter().enumerate().for_each(|(index, command)| {
        assert!(!command.label.is_empty(), "Program.commands[{index}].label is unset.");
        validate_command_options(command, &global_flags);
    });
    Program { label: label.to_string(), description: description.to_string(), commands, global_flags, single: false }
}

/// Builds a no-selector [`Program`] for a binary that does one thing, like
/// `sloc ./src`: the first token after the program name is the first
/// positional argument. Validates arguments and flags the same way [`new`]
/// does, panicking when validation fails.
pub fn new_single(label: &str, description: &str, arguments: Vec<Parameter>, flags: Vec<Parameter>) -> Program {
    let command =
        Command { label: label.to_string(), description: description.to_string(), arguments, flags };
    validate_command_options(&command, &[]);
    Program {
        label: label.to_string(),
        description: description.to_string(),
        commands: vec![command],
        global_flags: Vec::new(),
        single: true,
    }
}

/// Panics when any of a command's options is malformed: a bad label, an
/// unsupported value type, a slice argument that is not last, or a label
/// shared with another option (arguments and flags occupy one
/// `-label=value` namespace, seeded here with the global flags).
fn validate_command_options(command: &Command, global_flags: &[Parameter]) {
    let seen: Vec<String> = global_flags.iter().map(|flag| flag.label.clone()).collect();
    let seen = command
        .arguments
        .iter()
        .enumerate()
        .fold(seen, |seen, (index, argument)| validate_argument(command, index, argument, &seen));
    command.flags.iter().fold(seen, |seen, flag| validate_command_flag(command, flag, &seen));
}

/// Validates one argument against the labels seen so far and returns the seen
/// set with this argument's label added.
fn validate_argument(command: &Command, index: usize, argument: &Parameter, seen: &[String]) -> Vec<String> {
    let context = format!("Argument #{index} for command {:?}", command.label);
    validate_label(&context, &argument.label);
    let is_last = index == command.arguments.len() - 1;
    assert!(
        !is_variadic(&argument.value) || is_last,
        "Slice argument {:?} must be the last argument.",
        argument.label
    );
    let supported = matches!(
        argument.value,
        Parameter_Value::Str(_) | Parameter_Value::Int(_) | Parameter_Value::Str_Variadic(_) | Parameter_Value::Int_Variadic(_)
    );
    assert!(supported, "Argument {:?} has an unsupported type.", argument.label);
    assert!(
        !seen.contains(&argument.label),
        "Command {:?} has argument {:?} that collides with another option.",
        command.label,
        argument.label
    );
    seen.iter().cloned().chain(iter::once(argument.label.clone())).collect()
}

/// Validates one flag against the labels seen so far and returns the seen
/// set with this flag's label added.
fn validate_command_flag(command: &Command, flag: &Parameter, seen: &[String]) -> Vec<String> {
    validate_flag(&format!("Flag for command {:?}", command.label), flag);
    assert!(
        !seen.contains(&flag.label),
        "Command {:?} has flag {:?} that collides with another option.",
        command.label,
        flag.label
    );
    seen.iter().cloned().chain(iter::once(flag.label.clone())).collect()
}

/// Reports whether a value is variadic (a slice) — the trait that makes an
/// argument collect positionals, and a named option append on each repeated
/// `-label=value`.
fn is_variadic(value: &Parameter_Value) -> bool {
    matches!(value, Parameter_Value::Str_Variadic(_) | Parameter_Value::Int_Variadic(_))
}

/// Panics when a flag's label is invalid or its value is not a supported
/// scalar type — a flag cannot be variadic.
fn validate_flag(context: &str, flag: &Parameter) {
    validate_label(context, &flag.label);
    let supported = matches!(flag.value, Parameter_Value::Str(_) | Parameter_Value::Bool(_) | Parameter_Value::Int(_));
    assert!(supported, "Flag {:?} has an unsupported type.", flag.label);
}

/// Panics when an option's label is empty or carries a character that cannot
/// appear in a `-label` token. Arguments and flags share this rule because
/// both are settable by name.
fn validate_label(context: &str, label: &str) {
    assert!(!label.is_empty(), "{context} has no label.");
    assert!(
        !label.contains('_'),
        "Option labels cannot contain underscores. Instead of {label:?}, use {:?}",
        label.replace('_', "-")
    );
    assert!(!label.contains(' '), "Option labels cannot contain spaces: {label:?}");
}

/// Every way [`program_parse`] can fail: an unknown command or option (each
/// possibly carrying a "did you mean" suggestion), a scalar set more than
/// once, a positional with nothing left to fill, a required argument left
/// unset, a value that will not convert to a whole number, a non-boolean flag
/// given no value, or a double-dash flag.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum Parse_Error {
    Unknown_Command { name: String, suggestion: Option<String> },
    Unknown_Option { label: String, suggestion: Option<String> },
    Option_Set_Twice { label: String },
    Unexpected_Positional { value: String },
    Missing_Argument { label: String },
    Invalid_Integer { label: String, value: String },
    Flag_Needs_Value { label: String },
    Double_Dash_Flag { given: String },
}

/// Renders a [`Parse_Error`] the way a CLI would print it. A free function,
/// not `impl Display`: `Display::fmt` takes `f: &mut Formatter`, and that
/// `&mut` would be literal source text this module writes — the `mut` ban's
/// whitelist is exact `(file, fn)` pairs already spent on `arena`/`gen_arena`,
/// and `fmt` here is not one of them.
pub fn parse_error_message(error: &Parse_Error) -> String {
    match error {
        Parse_Error::Unknown_Command { name, suggestion } => match suggestion {
            Some(candidate) => format!("unknown command {name:?}, did you mean {candidate:?}?"),
            None => format!("unknown command {name:?}"),
        },
        Parse_Error::Unknown_Option { label, suggestion } => match suggestion {
            Some(candidate) => format!("unknown option -{label}, did you mean -{candidate}?"),
            None => format!("unknown option -{label}"),
        },
        Parse_Error::Option_Set_Twice { label } => format!("-{label} may only be given once"),
        Parse_Error::Unexpected_Positional { value } => {
            format!("too many arguments: {value:?} was not expected")
        }
        Parse_Error::Missing_Argument { label } => format!(
            "missing required argument {label:?}; pass it by position or as -{label}=value"
        ),
        Parse_Error::Invalid_Integer { label, value } => {
            format!("{label} expects a whole number, but got {value:?}")
        }
        Parse_Error::Flag_Needs_Value { label } => {
            format!("-{label} needs a value, e.g. -{label}=value")
        }
        Parse_Error::Double_Dash_Flag { given } => {
            format!("use a single dash: -{given}, not --{given}")
        }
    }
}

/// Retrieves an option by label from a slice of options, handing the found
/// [`Parameter`] to `reader` and returning its owned result — a visitor, not
/// `-> &Parameter`, because the dialect bans reference returns. Panics when
/// the label is not found.
pub fn get_option<Result_Type>(
    parameters: &[Parameter], label: &str, reader: impl FnOnce(&Parameter) -> Result_Type,
) -> Result_Type {
    match parameters.iter().find(|parameter| parameter.label == label) {
        Some(parameter) => reader(parameter),
        None => panic!("{label:?} is an unknown option"),
    }
}

/// The active command and the program's global flags, resolved and populated
/// by [`program_parse`]. Go's original mutates `*Program` in place to expose
/// global-flag values after parsing while returning only the command; taking
/// `&mut Program` here would need the banned `mut` token, so both are
/// returned together instead.
#[derive(Clone, Debug)]
pub struct Parse_Outcome {
    pub command: Command,
    pub global_flags: Vec<Parameter>,
}

/// Parses `arguments` (`std::env::args()`-shaped: the program name, then the
/// rest) against `program` and returns the active command with populated
/// values. If no command is named, the first declared command is used. Every
/// option is settable by `-label=value` and arguments may also be given
/// positionally; the two kinds of token may interleave freely.
pub fn program_parse(program: &Program, arguments: &[String]) -> Result<Parse_Outcome, Parse_Error> {
    assert!(!arguments.is_empty(), "program_parse needs at least one argument (the program name).");
    let (command, arguments_start) = resolve_command(program, arguments)?;
    match arguments.len() < arguments_start {
        true => Ok(Parse_Outcome { command, global_flags: program.global_flags.clone() }),
        false => program_parse_tokens(program, command, &arguments[arguments_start..]),
    }
}

/// The tail of [`program_parse`] once the active command and its token slice
/// are known: assign every named option, fill positionals (including the
/// trailing variadic), then check every required argument landed.
fn program_parse_tokens(
    program: &Program, command: Command, tokens: &[String],
) -> Result<Parse_Outcome, Parse_Error> {
    // `command` is already this call's own owned copy (cloned once in
    // `resolve_command`), so its arguments/flags move into `assign_named`
    // directly rather than being cloned a second time from a borrow.
    let Command { label, description, arguments, flags } = command;
    let named = assign_named(program, arguments, flags, tokens)?;
    let global_flags = named.global_flags.clone();
    let (final_command, filled) = assign_positionals(label, description, named)?;
    validate_required(&final_command, &filled)?;
    Ok(Parse_Outcome { command: final_command, global_flags })
}

/// Finds the active command (defaulting to the first) and the index at which
/// its positionals and flags begin: after the program name and the command
/// name in multi-command mode, after only the program name in single-command
/// mode, where the first token is already a positional.
fn resolve_command(program: &Program, arguments: &[String]) -> Result<(Command, usize), Parse_Error> {
    match program.single {
        true => Ok((program.commands[0].clone(), 1)),
        false => resolve_named_command(program, arguments),
    }
}

fn resolve_named_command(program: &Program, arguments: &[String]) -> Result<(Command, usize), Parse_Error> {
    match arguments.get(1) {
        None => Ok((program.commands[0].clone(), 2)),
        Some(name) => match program.commands.iter().find(|command| &command.label == name) {
            Some(command) => Ok((command.clone(), 2)),
            None => Err(unknown_command_error(program, name)),
        },
    }
}

fn unknown_command_error(program: &Program, name: &str) -> Parse_Error {
    let candidates: Vec<String> = program.commands.iter().map(|command| command.label.clone()).collect();
    Parse_Error::Unknown_Command { name: name.to_string(), suggestion: levenshtein::closest(name, &candidates) }
}

/// Where an option was found: which of the three label namespaces (a
/// command's arguments, its flags, or the program's global flags) and its
/// index within that namespace.
enum Option_Site {
    Argument(usize),
    Flag(usize),
    Global(usize),
}

/// The running state of [`assign_named`]'s left-to-right token walk. The
/// command's arguments/flags and the program's global flags start as clones
/// and stay untouched throughout the walk — read for lookups and type
/// checks, but never rebuilt per token. Each scalar assignment instead
/// queues into `pending`, applied to all three in one pass per namespace
/// once the walk finishes ([`apply_pending_updates`]): rebuilding an
/// `O(n)`-sized `Vec<Parameter>` on every one of `m` tokens costs `O(n·m)`,
/// while collecting and applying once costs `O(n+m)`.
struct Named_State {
    pub arguments: Vec<Parameter>,
    pub flags: Vec<Parameter>,
    pub global_flags: Vec<Parameter>,
    pub filled: Vec<String>,
    pub positionals: Vec<(usize, String)>,
    pub slice_contributions: Vec<(usize, String)>,
    pub pending: Vec<(Option_Site, Parameter_Value)>,
}

/// Applies every `-label=value` token to its option and records the bare
/// positionals, tagged with their original position so a trailing variadic
/// argument can merge them with its own named contributions in command-line
/// order. A single `try_fold` over the tokens — no `mut` binding needed, since
/// `try_fold` is callable on the unbound temporary iterator and each step
/// returns a fresh [`Named_State`] rather than mutating one in place.
fn assign_named(
    program: &Program, arguments: Vec<Parameter>, flags: Vec<Parameter>, tokens: &[String],
) -> Result<Named_State, Parse_Error> {
    let initial = Named_State {
        arguments,
        flags,
        global_flags: program.global_flags.clone(),
        filled: Vec::new(),
        positionals: Vec::new(),
        slice_contributions: Vec::new(),
        pending: Vec::new(),
    };
    let walked = tokens.iter().enumerate().try_fold(initial, assign_named_token)?;
    Ok(apply_pending_updates(walked))
}

/// Applies every scalar assignment collected during the token walk in one
/// pass per namespace (arguments, flags, global flags) instead of the
/// `O(n)` rebuild [`assign_scalar_site`] used to do on every single token.
fn apply_pending_updates(state: Named_State) -> Named_State {
    let grouped = partition_pending(state.pending);
    Named_State {
        arguments: apply_updates(state.arguments, grouped.arguments),
        flags: apply_updates(state.flags, grouped.flags),
        global_flags: apply_updates(state.global_flags, grouped.global_flags),
        pending: Vec::new(),
        ..state
    }
}

/// Pending updates grouped by namespace — the shape [`apply_pending_updates`]
/// applies in one pass each.
struct Pending_Updates {
    pub arguments: Vec<(usize, Parameter_Value)>,
    pub flags: Vec<(usize, Parameter_Value)>,
    pub global_flags: Vec<(usize, Parameter_Value)>,
}

fn partition_pending(pending: Vec<(Option_Site, Parameter_Value)>) -> Pending_Updates {
    let empty = Pending_Updates { arguments: Vec::new(), flags: Vec::new(), global_flags: Vec::new() };
    pending.into_iter().fold(empty, partition_pending_step)
}

fn partition_pending_step(updates: Pending_Updates, (site, value): (Option_Site, Parameter_Value)) -> Pending_Updates {
    match site {
        Option_Site::Argument(index) => {
            Pending_Updates { arguments: append_update(updates.arguments, index, value), ..updates }
        }
        Option_Site::Flag(index) => Pending_Updates { flags: append_update(updates.flags, index, value), ..updates },
        Option_Site::Global(index) => {
            Pending_Updates { global_flags: append_update(updates.global_flags, index, value), ..updates }
        }
    }
}

fn append_update(
    list: Vec<(usize, Parameter_Value)>, index: usize, value: Parameter_Value,
) -> Vec<(usize, Parameter_Value)> {
    list.into_iter().chain(iter::once((index, value))).collect()
}

/// Applies every pending `(index, value)` update to `parameters` in one pass,
/// via a `HashMap` lookup per element — `O(n + u)` for `n` parameters and `u`
/// updates, rather than the `O(n)`-per-update cost of rebuilding the whole
/// vector once per pending change.
fn apply_updates(parameters: Vec<Parameter>, updates: Vec<(usize, Parameter_Value)>) -> Vec<Parameter> {
    let by_index: collections::HashMap<usize, Parameter_Value> = updates.into_iter().collect();
    parameters
        .into_iter()
        .enumerate()
        .map(|(index, parameter)| match by_index.get(&index) {
            Some(value) => Parameter { value: value.clone(), ..parameter },
            None => parameter,
        })
        .collect()
}

fn assign_named_token(state: Named_State, (index, token): (usize, &String)) -> Result<Named_State, Parse_Error> {
    match is_named_token(token) {
        false => Ok(Named_State { positionals: append_indexed(state.positionals, index, token.clone()), ..state }),
        true => assign_named_option(state, index, token),
    }
}

/// Reports whether a token sets an option by name. A lone `-` is a
/// positional value, not a named one.
fn is_named_token(token: &str) -> bool {
    token.starts_with('-') && token != "-"
}

fn append_indexed(list: Vec<(usize, String)>, index: usize, value: String) -> Vec<(usize, String)> {
    list.into_iter().chain(iter::once((index, value))).collect()
}

/// Splits a `-label=value` token into its parts and routes it: an unknown
/// label errors (with a suggestion when one looks like a typo), a variadic
/// match appends to the slice contributions, and a scalar match is assigned
/// in place.
fn assign_named_option(state: Named_State, index: usize, token: &str) -> Result<Named_State, Parse_Error> {
    let (label, value, had_value) = parse_named_token(token)?;
    match find_option_site(&state, &label) {
        None => Err(unknown_option_error(&state, &label)),
        Some(site) => assign_option_site(state, index, &label, &value, had_value, site),
    }
}

/// Splits a `-label=value` token into its parts, rejecting the double-dash
/// form. A lone `-` was already excluded by [`is_named_token`].
fn parse_named_token(token: &str) -> Result<(String, String, bool), Parse_Error> {
    if let Some(rest) = token.strip_prefix("--")
        && !rest.is_empty()
    {
        return Err(Parse_Error::Double_Dash_Flag { given: rest.to_string() });
    }
    match token[1..].split_once('=') {
        Some((label, value)) => Ok((label.to_string(), value.to_string(), true)),
        None => Ok((token[1..].to_string(), String::new(), false)),
    }
}

/// Finds the option named by a `-label` token across the command's arguments,
/// then its flags, then the program's global flags.
fn find_option_site(state: &Named_State, label: &str) -> Option<Option_Site> {
    find_index(&state.arguments, label)
        .map(Option_Site::Argument)
        .or_else(|| find_index(&state.flags, label).map(Option_Site::Flag))
        .or_else(|| find_index(&state.global_flags, label).map(Option_Site::Global))
}

fn find_index(parameters: &[Parameter], label: &str) -> Option<usize> {
    parameters.iter().position(|parameter| parameter.label == label)
}

fn unknown_option_error(state: &Named_State, label: &str) -> Parse_Error {
    let candidates: Vec<String> = state
        .arguments
        .iter()
        .chain(state.flags.iter())
        .chain(state.global_flags.iter())
        .map(|parameter| parameter.label.clone())
        .collect();
    Parse_Error::Unknown_Option { label: label.to_string(), suggestion: levenshtein::closest(label, &candidates) }
}

/// Routes a found option: a variadic match appends to the slice
/// contributions (repeatable, no double-set check); a scalar match is
/// assigned in place.
fn assign_option_site(
    state: Named_State, index: usize, label: &str, value: &str, had_value: bool, site: Option_Site,
) -> Result<Named_State, Parse_Error> {
    match is_variadic(&option_value_at(&state, &site)) {
        true => Ok(Named_State {
            slice_contributions: append_indexed(state.slice_contributions, index, value.to_string()),
            ..state
        }),
        false => assign_scalar_site(state, label, value, had_value, site),
    }
}

fn option_value_at(state: &Named_State, site: &Option_Site) -> Parameter_Value {
    match site {
        Option_Site::Argument(index) => state.arguments[*index].value.clone(),
        Option_Site::Flag(index) => state.flags[*index].value.clone(),
        Option_Site::Global(index) => state.global_flags[*index].value.clone(),
    }
}

/// Rebuilds `parameters` with the value at `index` replaced — the
/// mutation-free stand-in for assigning through a pointer into the slot.
fn replace_value(parameters: Vec<Parameter>, index: usize, value: Parameter_Value) -> Vec<Parameter> {
    parameters
        .into_iter()
        .enumerate()
        .map(|(current, parameter)| match current == index {
            true => Parameter { value: value.clone(), ..parameter },
            false => parameter,
        })
        .collect()
}

/// Validates and assigns a scalar named value: a label already in `filled`
/// errors, as does a non-boolean value given no `=value`.
fn assign_scalar_site(
    state: Named_State, label: &str, value: &str, had_value: bool, site: Option_Site,
) -> Result<Named_State, Parse_Error> {
    if state.filled.contains(&label.to_string()) {
        return Err(Parse_Error::Option_Set_Twice { label: label.to_string() });
    }
    let current = option_value_at(&state, &site);
    let is_boolean = matches!(current, Parameter_Value::Bool(_));
    let absent = !had_value || value.is_empty();
    if !is_boolean && absent {
        return Err(Parse_Error::Flag_Needs_Value { label: label.to_string() });
    }
    let new_value = apply_scalar_value(current, value, label)?;
    Ok(Named_State {
        pending: append_pending(state.pending, site, new_value),
        filled: append(state.filled, label.to_string()),
        ..state
    })
}

fn append_pending(
    pending: Vec<(Option_Site, Parameter_Value)>, site: Option_Site, value: Parameter_Value,
) -> Vec<(Option_Site, Parameter_Value)> {
    pending.into_iter().chain(iter::once((site, value))).collect()
}

fn append(list: Vec<String>, item: String) -> Vec<String> {
    list.into_iter().chain(iter::once(item)).collect()
}

/// Converts a parsed flag value to the option's declared scalar type.
fn apply_scalar_value(current: Parameter_Value, value: &str, label: &str) -> Result<Parameter_Value, Parse_Error> {
    match current {
        Parameter_Value::Bool(_) => Ok(Parameter_Value::Bool(true)),
        Parameter_Value::Str(_) => Ok(Parameter_Value::Str(trim_quotes(value))),
        Parameter_Value::Int(_) => match value.parse::<i64>() {
            Ok(number) => Ok(Parameter_Value::Int(number)),
            Err(_) => Err(Parse_Error::Invalid_Integer { label: label.to_string(), value: value.to_string() }),
        },
        Parameter_Value::Str_Variadic(_) | Parameter_Value::Int_Variadic(_) => {
            unreachable!("assign_option_site routes variadic matches to slice_contributions")
        }
    }
}

/// Removes a matching pair of surrounding double or single quotes from a
/// value, leaving other strings — including a mismatched pair — untouched.
fn trim_quotes(text: &str) -> String {
    let chars: Vec<char> = text.chars().collect();
    match chars.len() < 2 {
        true => text.to_string(),
        false => trim_quotes_chars(&chars, text),
    }
}

fn trim_quotes_chars(chars: &[char], original: &str) -> String {
    let first = chars[0];
    let last = chars[chars.len() - 1];
    match (first == '"' && last == '"') || (first == '\'' && last == '\'') {
        true => chars[1..chars.len() - 1].iter().collect(),
        false => original.to_string(),
    }
}

/// Fills the command's scalar arguments from the positional tokens in
/// declaration order, skipping any already set by name, then routes the rest
/// into the trailing variadic argument (if any) together with its named
/// contributions, merged in command-line order.
fn assign_positionals(
    label: String, description: String, named: Named_State,
) -> Result<(Command, Vec<String>), Parse_Error> {
    let slice_index = variadic_argument_index(&named.arguments);
    let fill_targets: Vec<usize> = named
        .arguments
        .iter()
        .enumerate()
        .filter(|(_, argument)| !is_variadic(&argument.value) && !named.filled.contains(&argument.label))
        .map(|(index, _)| index)
        .collect();
    let fill = fill_scalar_positionals(named.arguments, &fill_targets, &named.positionals, named.filled)?;
    let arguments = merge_slice_argument(fill.arguments, slice_index, fill.overflow, named.slice_contributions)?;
    let command = Command { label, description, arguments, flags: named.flags };
    Ok((command, fill.filled))
}

/// The index of the command's trailing variadic argument, or `None` when the
/// last argument is not variadic. Construction-time validation already
/// guarantees a variadic argument, if any, is the last one.
fn variadic_argument_index(arguments: &[Parameter]) -> Option<usize> {
    match arguments.last() {
        Some(last) if is_variadic(&last.value) => Some(arguments.len() - 1),
        _ => None,
    }
}

/// The result of [`fill_scalar_positionals`]: the updated arguments and
/// filled set, alongside whichever positionals were left over (destined for
/// the trailing variadic argument, if any).
struct Positional_Fill {
    pub arguments: Vec<Parameter>,
    pub overflow: Vec<(usize, String)>,
    pub filled: Vec<String>,
}

/// Resolves positionals against `fill_targets` in order — extending `filled`
/// with each one, exactly as a named assignment would, so the
/// required-argument check sees it — and applies them to `arguments` in one
/// pass, rather than rebuilding the whole vector once per positional.
fn fill_scalar_positionals(
    arguments: Vec<Parameter>, fill_targets: &[usize], positionals: &[(usize, String)], filled: Vec<String>,
) -> Result<Positional_Fill, Parse_Error> {
    let (updates, filled) = fill_targets.iter().zip(positionals.iter()).try_fold(
        (Vec::new(), filled),
        |(updates, filled), (&target, (_, value))| {
            let label = arguments[target].label.clone();
            let resolved = resolve_positional_value(&arguments[target].value, value, &label)?;
            Ok((append_update(updates, target, resolved), append(filled, label)))
        },
    )?;
    let overflow: Vec<(usize, String)> = positionals.iter().skip(fill_targets.len()).cloned().collect();
    Ok(Positional_Fill { arguments: apply_updates(arguments, updates), overflow, filled })
}

/// Converts a positional token to the scalar argument's type. Unlike a named
/// value, a positional is taken verbatim: no quote trimming, and empty is
/// allowed.
fn resolve_positional_value(current: &Parameter_Value, value: &str, label: &str) -> Result<Parameter_Value, Parse_Error> {
    match current {
        Parameter_Value::Str(_) => Ok(Parameter_Value::Str(value.to_string())),
        Parameter_Value::Int(_) => match value.parse::<i64>() {
            Ok(number) => Ok(Parameter_Value::Int(number)),
            Err(_) => Err(Parse_Error::Invalid_Integer { label: label.to_string(), value: value.to_string() }),
        },
        _ => unreachable!("fill_targets excludes variadic arguments"),
    }
}

/// Merges the overflow positionals and the slice argument's named
/// contributions into the trailing variadic argument, in original
/// command-line order — a `BTreeMap` keyed by each token's original index
/// orders them without an explicit sort (which would need a `mut` binding).
/// No variadic argument and leftover positionals is an error instead.
fn merge_slice_argument(
    arguments: Vec<Parameter>, slice_index: Option<usize>, overflow: Vec<(usize, String)>,
    slice_named: Vec<(usize, String)>,
) -> Result<Vec<Parameter>, Parse_Error> {
    match slice_index {
        None => match overflow.first() {
            Some((_, value)) => Err(Parse_Error::Unexpected_Positional { value: value.clone() }),
            None => Ok(arguments),
        },
        Some(index) => {
            let ordered: collections::BTreeMap<usize, String> = overflow.into_iter().chain(slice_named).collect();
            let values: Vec<String> = ordered.into_values().collect();
            let new_value = build_slice_value(&arguments[index].value, &values, &arguments[index].label)?;
            Ok(replace_value(arguments, index, new_value))
        }
    }
}

/// Builds the variadic argument's final value from its contributions,
/// already ordered, converting each element to the slice's element type.
fn build_slice_value(current: &Parameter_Value, values: &[String], label: &str) -> Result<Parameter_Value, Parse_Error> {
    match current {
        Parameter_Value::Str_Variadic(_) => Ok(Parameter_Value::Str_Variadic(values.to_vec())),
        Parameter_Value::Int_Variadic(_) => Ok(Parameter_Value::Int_Variadic(parse_int_elements(values, label)?)),
        _ => unreachable!("slice_index only points at a variadic argument"),
    }
}

fn parse_int_elements(values: &[String], label: &str) -> Result<Vec<i64>, Parse_Error> {
    values
        .iter()
        .map(|value| {
            value
                .parse::<i64>()
                .map_err(|_| Parse_Error::Invalid_Integer { label: label.to_string(), value: value.clone() })
        })
        .collect()
}

/// Returns an error naming the first scalar argument left unset by both name
/// and position. The variadic argument, if any, is never required — an empty
/// list is valid.
fn validate_required(command: &Command, filled: &[String]) -> Result<(), Parse_Error> {
    command
        .arguments
        .iter()
        .filter(|argument| !is_variadic(&argument.value))
        .find(|argument| !filled.contains(&argument.label))
        .map_or(Ok(()), |argument| Err(Parse_Error::Missing_Argument { label: argument.label.clone() }))
}

/// The note printed under the usage line when a program has at least one
/// positional argument: each may also be supplied by name, not only by
/// position.
const NAMED_ARGUMENT_LEGEND: &str = "    Positional arguments may also be supplied via -key=val syntax.";

/// Renders help for `program`: the header, usage line, global flags, and
/// every command with its arguments and flags — or, in single-command mode,
/// the program's own positionals and flags with no command selector.
/// Returns an owned `String` rather than writing to `impl io::Write`: taking
/// `&mut dyn Write` would need the banned `mut` token.
pub fn print_help(program: &Program) -> String {
    let header = format!("{} {}\n\n", program.label, program.description);
    match program.single {
        true => header + &help_single_block(program),
        false => header + &help_usage_line(program) + &help_global_flags_block(program) + &help_commands_block(program),
    }
}

/// Reports whether any of the program's commands declares a positional
/// argument — the condition under which the named-argument legend is worth
/// printing.
fn program_has_arguments(program: &Program) -> bool {
    program.commands.iter().any(|command| !command.arguments.is_empty())
}

fn help_usage_line(program: &Program) -> String {
    let legend = match program_has_arguments(program) {
        true => format!("{NAMED_ARGUMENT_LEGEND}\n"),
        false => String::new(),
    };
    format!("Usage:\n    {} <command> <arguments> [-flags[=value]]\n{legend}\n", program.label)
}

fn help_global_flags_block(program: &Program) -> String {
    match program.global_flags.is_empty() {
        true => String::new(),
        false => {
            let lines: String =
                program.global_flags.iter().map(|flag| help_flag_line(flag, "    ", true)).collect();
            format!("Global Flags:\n{lines}\n")
        }
    }
}

fn help_commands_block(program: &Program) -> String {
    let commands: String = program.commands.iter().map(help_command_block).collect();
    format!("Available Commands:\n{commands}")
}

fn help_command_block(command: &Command) -> String {
    let signature = help_command_signature(command);
    let flags = match command.flags.is_empty() {
        true => String::new(),
        false => {
            let lines: String =
                command.flags.iter().map(|flag| help_flag_line(flag, "        ", false)).collect();
            format!("\n{lines}")
        }
    };
    format!("    {signature}  {}\n{flags}\n", command.description)
}

/// The colored command label followed by its argument signatures, as it
/// appears at the start of its help row.
fn help_command_signature(command: &Command) -> String {
    let arguments: String =
        command.arguments.iter().map(|argument| format!("{} ", help_argument_signature(argument))).collect();
    format!("\x1b[34m{}\x1b[0m {arguments}", command.label)
}

/// Renders help for a single-command program: a usage line carrying the
/// program's own positional arguments — there is no command selector to
/// choose — followed by its flags.
fn help_single_block(program: &Program) -> String {
    let command = &program.commands[0];
    let signature: String =
        command.arguments.iter().map(|argument| format!("{} ", help_argument_signature(argument))).collect();
    let legend = match command.arguments.is_empty() {
        true => String::new(),
        false => format!("{NAMED_ARGUMENT_LEGEND}\n"),
    };
    let flags = match command.flags.is_empty() {
        true => String::new(),
        false => {
            let lines: String =
                command.flags.iter().map(|flag| help_flag_line(flag, "    ", true)).collect();
            format!("\nFlags:\n{lines}")
        }
    };
    format!("Usage:\n    {} {signature}[-flags[=value]]\n{legend}{flags}", program.label)
}

/// Formats a positional argument for a usage line: `<label: type>`, with a
/// trailing ellipsis for a variadic — its element type, not its slice type,
/// so a `Str_Variadic` reads as `<paths: string...>` rather than the noisier
/// `<paths: Str_Variadic>`.
fn help_argument_signature(argument: &Parameter) -> String {
    match &argument.value {
        Parameter_Value::Str_Variadic(_) => format!("<{}: string...>", argument.label),
        Parameter_Value::Int_Variadic(_) => format!("<{}: int...>", argument.label),
        Parameter_Value::Str(_) => format!("<{}: string>", argument.label),
        Parameter_Value::Int(_) => format!("<{}: int>", argument.label),
        Parameter_Value::Bool(_) => format!("<{}: bool>", argument.label),
    }
}

/// Renders one flag row: the type-annotated label, its default, and its
/// description. The label is colored when `color` is set.
fn help_flag_line(flag: &Parameter, indent: &str, color: bool) -> String {
    let label = match color {
        true => format!("\x1b[34m{}\x1b[0m", flag.label),
        false => flag.label.clone(),
    };
    match &flag.value {
        Parameter_Value::Bool(_) => format!("{indent}-{label}  {}\n", flag.description),
        other => format!(
            "{indent}-{label}{}  (default: {})  {}\n",
            help_value_type(other),
            help_default_value(other),
            flag.description
        ),
    }
}

fn help_value_type(value: &Parameter_Value) -> String {
    match value {
        Parameter_Value::Str(_) => "=string".to_string(),
        Parameter_Value::Int(_) => "=int".to_string(),
        _ => String::new(),
    }
}

/// The flag's default rendered for display; an empty string default renders
/// as the literal `""` rather than invisibly.
fn help_default_value(value: &Parameter_Value) -> String {
    match value {
        Parameter_Value::Str(text) if text.is_empty() => "\"\"".to_string(),
        Parameter_Value::Str(text) => text.clone(),
        Parameter_Value::Int(number) => number.to_string(),
        _ => String::new(),
    }
}
