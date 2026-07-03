// Whole-process maddox harness for shared_rs::cli's parsing hot path. Not
// library code (examples/ isn't scanned by lint_rs), so it's plain,
// unremarkable Rust rather than the dialect's mutation-free style.
use shared_rs::cli;

const FLAG_COUNT: usize = 40;
const ITERATIONS: usize = 20_000;

fn synthetic_program() -> cli::Program {
    let flags: Vec<cli::Parameter> = (0..FLAG_COUNT)
        .map(|i| cli::new_string_flag(&format!("flag{i}"), "a synthetic flag", "default"))
        .collect();
    cli::new(
        "bench",
        "synthetic benchmark program",
        vec![cli::Command {
            label: "run".to_string(),
            description: String::new(),
            arguments: vec![cli::new_string_argument("task", "a synthetic argument")],
            flags,
        }],
        vec![],
    )
}

fn synthetic_arguments() -> Vec<String> {
    let mut tokens = vec!["bench".to_string(), "run".to_string(), "task-value".to_string()];
    for i in 0..FLAG_COUNT {
        tokens.push(format!("-flag{i}=value{i}"));
    }
    tokens
}

fn main() {
    let program = synthetic_program();
    let arguments = synthetic_arguments();
    let mut checksum: usize = 0;
    for _ in 0..ITERATIONS {
        let outcome = cli::program_parse(&program, &arguments).unwrap();
        checksum += outcome.command.flags.len();
    }
    println!("{checksum}");
}
