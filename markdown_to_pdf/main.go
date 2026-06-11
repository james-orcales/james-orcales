// Package main is the markdown_to_pdf command. render converts a Markdown file to
// PDF beside it or to -out; preview renders one to /tmp and opens it; golden
// does the same with a built-in showcase.
package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/james-orcales/james-orcales/markdown_to_pdf/internal"
	"github.com/james-orcales/james-orcales/shared/cli"
)

// Markdown_bytes_max caps the input the command reads into its fixed buffer.
// 16 MiB dwarfs any hand-written document yet bounds memory against an
// accidental or hostile huge file, satisfying the unbounded-read ban.
const markdown_bytes_max = 16777216

// Exit_usage marks a malformed command line, kept distinct from a run failure
// so a caller can tell "you invoked me wrong" from "the work itself failed".
const exit_usage = 2

// Exit_failure marks a read, render, or write failure during an otherwise
// well-formed invocation.
const exit_failure = 1

// Exit_exists marks the refusal to clobber: no -out was given and the path
// derived beside the input already holds a file, so nothing is written.
const exit_exists = 3

// Golden_path is the fixed destination of the golden showcase, parked in /tmp
// as a regenerate-on-demand reference rather than a file the caller names.
const golden_path = "/tmp/markdown_to_pdf_golden.pdf"

func main() {
	program := main_program()
	if len(os.Args) < 2 {
		cli.Print_Help(os.Stderr, program)
		os.Exit(exit_usage)
	}
	command, parse_err := cli.Program_Parse(&program, os.Args)
	if parse_err != nil {
		fmt.Fprintln(os.Stderr, parse_err)
		cli.Print_Help(os.Stderr, program)
		os.Exit(exit_usage)
	}
	if command.Label == "golden" {
		os.Exit(main_golden())
	}
	if command.Label == "preview" {
		os.Exit(main_preview(command))
	}
	os.Exit(main_render_command(command))
}

// Declares the markdown_to_pdf program: a render command taking the input file and
// an optional -out path, and a golden command that writes the showcase.
func main_program() (program cli.Program) {
	input := cli.New_Argument[string](cli.New_Argument_Input{
		Label:       "input",
		Description: "the Markdown file to convert",
	})
	output_flag := cli.New_Flag[string](cli.New_Flag_Input[string]{
		Label:       "out",
		Description: "output PDF path; defaults to beside the input",
	})
	render := cli.Command{
		Label:       "render",
		Description: "render a Markdown file to PDF, beside it or to -out",
		Arguments:   []cli.Option{input},
		Flags:       []cli.Option{output_flag},
	}
	preview := cli.Command{
		Label:       "preview",
		Description: "render a Markdown file to /tmp and open it",
		Arguments:   []cli.Option{input},
	}
	golden := cli.Command{
		Label:       "golden",
		Description: "write a feature showcase to /tmp and open it",
	}
	return cli.New(cli.New_Input{
		Label:       "markdown_to_pdf",
		Description: "render Markdown to PDF",
		Commands:    []cli.Command{render, preview, golden},
	})
}

// Pulls the input and -out off the render command, derives the output path, and
// renders the input into it.
func main_render_command(command cli.Command) (status_code int) {
	input_path := cli.Get_Option(command.Arguments, "input").Value.(string)
	output_path := main_output_path(&main_output_path_input{
		Input:  input_path,
		Output: cli.Get_Option(command.Flags, "out").Value.(string),
	})
	markdown, read_ok := main_read_file(input_path)
	if !read_ok {
		return exit_failure
	}
	return main_render(markdown, output_path)
}

// Renders the built-in showcase to /tmp and opens it.
func main_golden() (status_code int) {
	return main_render_then_open([]byte(golden_showcase), golden_path)
}

// Renders the command's input file to a /tmp preview path and opens it,
// overwriting any prior preview unconditionally.
func main_preview(command cli.Command) (status_code int) {
	input_path := cli.Get_Option(command.Arguments, "input").Value.(string)
	markdown, read_ok := main_read_file(input_path)
	if !read_ok {
		return exit_failure
	}
	return main_render_then_open(markdown, main_preview_path(input_path))
}

// A preview is written to /tmp, named for the input's base so previewing
// several files does not collide.
func main_preview_path(input_path string) (preview_path string) {
	base_name := filepath.Base(input_path)
	stem := strings.TrimSuffix(base_name, filepath.Ext(base_name))
	return filepath.Join("/tmp", stem+".pdf")
}

// Renders markdown to path, overwriting it, then opens the result in the default
// viewer; the path is reported so the caller knows where it landed.
func main_render_then_open(markdown []byte, path string) (status_code int) {
	status := main_render(markdown, path)
	if status != 0 {
		return status
	}
	fmt.Fprintf(os.Stderr, "markdown_to_pdf: wrote %s\n", path)
	main_open(path)
	return 0
}

// Hands the rendered showcase to the system opener so it surfaces in the
// default PDF viewer. A failure here is reported but does not fail the run,
// since the file is already written.
func main_open(path string) {
	open_err := exec.Command("open", path).Run()
	if open_err != nil {
		fmt.Fprintf(os.Stderr, "markdown_to_pdf: %v\n", open_err)
	}
}

// Creates output_path, renders markdown into it, and returns the process exit
// code. Shared by the file and -golden paths so both bind the output the same
// way.
func main_render(markdown []byte, output_path string) (status_code int) {
	output, create_err := os.Create(output_path)
	if create_err != nil {
		fmt.Fprintf(os.Stderr, "markdown_to_pdf: %v\n", create_err)
		return exit_failure
	}
	status := markdown_to_pdf.Main(&markdown_to_pdf.Main_Input{
		Markdown: markdown,
		Output:   output,
		Stderr:   os.Stderr,
	})
	close_err := output.Close()
	if close_err != nil {
		fmt.Fprintf(os.Stderr, "markdown_to_pdf: %v\n", close_err)
		return exit_failure
	}
	return status
}

type main_output_path_input struct {
	Input  string
	Output string
}

// Returns the explicit -out path when set; otherwise the input path with its
// extension swapped for .pdf, so a bare render writes beside its source. A
// derived path that already exists aborts the run rather than overwrite a file
// the caller never named.
func main_output_path(input *main_output_path_input) (output_path string) {
	if input.Output != "" {
		return input.Output
	}
	derived := strings.TrimSuffix(input.Input, filepath.Ext(input.Input)) + ".pdf"
	_, stat_err := os.Stat(derived)
	if stat_err == nil {
		fmt.Fprintf(os.Stderr, "markdown_to_pdf: %s already exists\n", derived)
		os.Exit(exit_exists)
	}
	return derived
}

// Reads the named file into one fixed buffer, bounded by markdown_bytes_max so
// no single input can exhaust memory. ok is false, with a stderr message, when
// the file cannot be opened, overflows the cap, or errors mid-read.
func main_read_file(name string) (contents []byte, ok bool) {
	file, open_err := os.Open(name)
	if open_err != nil {
		fmt.Fprintf(os.Stderr, "markdown_to_pdf: %v\n", open_err)
		return nil, false
	}
	defer file.Close()
	buffer := make([]byte, markdown_bytes_max)
	read_total := 0
	for read_total < len(buffer) {
		n, read_err := file.Read(buffer[read_total:])
		read_total += n
		if read_err == io.EOF {
			return buffer[:read_total], true
		}
		if read_err != nil {
			fmt.Fprintf(os.Stderr, "markdown_to_pdf: %v\n", read_err)
			return nil, false
		}
	}
	// The buffer filled before EOF, so the file is larger than the cap.
	fmt.Fprintln(os.Stderr, "markdown_to_pdf: input exceeds 16 MiB")
	return nil, false
}

// One Markdown document exercising every feature the converter renders — from
// smart punctuation and box-drawing diagrams to wrapping table cells — long
// enough to spill onto a second page so pagination shows too.
const golden_showcase = "# markdown_to_pdf showcase\n" +
	"\n" +
	"A minimal, zero-dependency Markdown to PDF converter. Every section below\n" +
	"exercises one of its features, rendered straight from Markdown with no\n" +
	"external libraries.\n" +
	"\n" +
	"## Text and emphasis\n" +
	"\n" +
	"Paragraphs wrap to the page width as you would expect. Within a line you can\n" +
	"mix **bold**, *italic*, and `inline code`, and a link such as\n" +
	"[random link that will brick your pc](https://github.com/james-orcales) renders blue,\n" +
	"underlined, and clickable.\n" +
	"\n" +
	"## Typography\n" +
	"\n" +
	"Punctuation is measured at its true width, so nothing drifts:\n" +
	"em-dashes — like this — en-dashes (pages 1–10), “curly quotes”,\n" +
	"the app’s apostrophe, and a bullet • all advance correctly, so a\n" +
	"**bold lead-in** `right next to code` keeps its panel aligned.\n" +
	"\n" +
	"## Heading levels\n" +
	"\n" +
	"### Level three heading\n" +
	"\n" +
	"#### Level four heading\n" +
	"\n" +
	"##### Level five heading\n" +
	"\n" +
	"###### Level six heading\n" +
	"\n" +
	"## Lists\n" +
	"\n" +
	"An unordered list:\n" +
	"\n" +
	"- Espresso\n" +
	"- Cortado\n" +
	"- Flat white\n" +
	"\n" +
	"An ordered list:\n" +
	"\n" +
	"1. Grind the beans\n" +
	"2. Pull the shot\n" +
	"3. Steam the milk\n" +
	"\n" +
	"## Block quote\n" +
	"\n" +
	"> A block quote is indented beside a soft gray bar, GitHub style, setting it\n" +
	"> apart from the surrounding paragraphs.\n" +
	"\n" +
	"## Code\n" +
	"\n" +
	"Inline `code` and fenced blocks both render white on a dark gray panel. A\n" +
	"fenced line of up to one hundred characters fits the column:\n" +
	"\n" +
	"```\n" +
	"func render(markdown []byte) []byte {\n" +
	"    // even a fairly long comment line stays on one line and fits the page width\n" +
	"    return assemble(layout(parse(markdown)))\n" +
	"}\n" +
	"```\n" +
	"\n" +
	"## Diagrams\n" +
	"\n" +
	"Box-drawing characters and arrows inside a code fence transliterate to\n" +
	"ASCII, so diagrams stay aligned without embedding a font:\n" +
	"\n" +
	"```\n" +
	"┌────────┐   ┌────────┐   ┌────────┐\n" +
	"│ parse  │──▶│ layout │──▶│ render │\n" +
	"└───┬────┘   └────────┘   └────────┘\n" +
	"    │\n" +
	"    ▼\n" +
	"┌────────┐\n" +
	"│ blocks │\n" +
	"└────────┘\n" +
	"```\n" +
	"\n" +
	"## Horizontal rule\n" +
	"\n" +
	"A thematic break draws a line across the column:\n" +
	"\n" +
	"---\n" +
	"\n" +
	"## Tables\n" +
	"\n" +
	"Cells parse inline markdown and wrap to their column; a row grows to fit\n" +
	"its tallest cell:\n" +
	"\n" +
	"| Drink | Ratio | Notes |\n" +
	"| --- | --- | --- |\n" +
	"| Espresso | `1:2` | the base shot, pulled in about 30 seconds |\n" +
	"| Cortado | `1:1` | equal parts espresso and milk; " +
	"see [the guide](https://github.com/james-orcales) |\n" +
	"| Flat white | `1:3` | a double ristretto under steamed microfoam, very smooth |\n" +
	"\n" +
	"The table keeps a clear margin above this paragraph rather than letting\n" +
	"prose lap its bottom border.\n" +
	"\n" +
	"## Pagination\n" +
	"\n" +
	"When content runs past the bottom margin the renderer opens a new page and\n" +
	"continues. This document is long enough that it flows onto a second page,\n" +
	"which is itself a demonstration of automatic pagination.\n" +
	"\n" +
	"Headings keep a blank line above them, paragraphs are separated by a small\n" +
	"gap, and every page carries the same media box, fonts, and margins. The cross\n" +
	"reference table and trailer are regenerated to match however many pages the\n" +
	"document needs.\n"
