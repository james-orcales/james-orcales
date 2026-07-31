// Package main is the markdown_to_pdf command. render converts Markdown to PDF
// or PDF to Markdown; preview writes the conversion to the system temp
// directory and opens it; golden keeps the built-in PDF showcase.
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"local/james-orcales/markdown_to_pdf/internal"
	"local/james-orcales/shared/cli"
)

// MARKDOWN_BYTES_MAX caps the input the command reads into its fixed buffer.
// 16 MiB dwarfs any hand-written document yet bounds memory against an
// accidental or hostile huge file, satisfying the unbounded-read ban.
const MARKDOWN_BYTES_MAX = 16777216

// EXIT_USAGE marks a malformed command line, kept distinct from a run failure
// so a caller can tell "you invoked me wrong" from "the work itself failed".
const EXIT_USAGE = 2

// EXIT_FAILURE marks a read, render, or write failure during an otherwise
// well-formed invocation.
const EXIT_FAILURE = 1

// EXIT_EXISTS marks the refusal to clobber: no -out was given and the path
// derived beside the input already holds a file, so nothing is written.
const EXIT_EXISTS = 3

func main() {
	program := main_program()
	if cli.Handle_Completion(program, os.Args, os.Stdout) {
		os.Exit(0)
	}
	if len(os.Args) < 2 {
		cli.Print_Help(os.Stderr, program)
		os.Exit(EXIT_USAGE)
	}
	command, parse_err := cli.Program_Parse(&program, os.Args)
	if errors.Is(parse_err, cli.Help_Requested) {
		cli.Print_Requested_Help(os.Stdout, program, command)
		os.Exit(0)
	}
	if parse_err != nil {
		fmt.Fprintln(os.Stderr, parse_err)
		cli.Print_Help(os.Stderr, program)
		os.Exit(EXIT_USAGE)
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
		Description: "the Markdown or PDF file to convert",
	})
	output_flag := cli.New_Flag[string](cli.New_Flag_Input[string]{
		Label:       "out",
		Description: "output path; defaults to .pdf for Markdown or .md for PDF",
	})
	render := cli.Command{
		Label:       "render",
		Description: "convert Markdown to PDF or PDF to Markdown, beside it or to -out",
		Arguments:   []cli.Option{input},
		Flags:       []cli.Option{output_flag},
	}
	preview := cli.Command{
		Label:       "preview",
		Description: "convert a Markdown or PDF file in the temp directory and open it",
		Arguments:   []cli.Option{input},
	}
	golden := cli.Command{
		Label:       "golden",
		Description: "write a feature showcase to the system temp directory and open it",
	}
	return cli.New(cli.New_Input{
		Label:       "markdown_to_pdf",
		Description: "convert Markdown to PDF or PDF to Markdown",
		Commands:    []cli.Command{render, preview, golden},
	})
}

// Pulls the input and -out off the render command, derives the output path, and
// renders the input into it.
func main_render_command(command cli.Command) (status_code int) {
	input_path := cli.Get_Option(command.Arguments, "input").Value.(string)
	explicit_output := cli.Get_Option(command.Flags, "out").Value.(string)
	output_path := main_output_path(&Main_Output_Path_Input{
		Input:  input_path,
		Output: explicit_output,
	})
	if explicit_output == "" {
		if main_path_exists(output_path) {
			fmt.Fprintf(os.Stderr, "markdown_to_pdf: %s already exists\n", output_path)
			return EXIT_EXISTS
		}
	}
	is_pdf := main_input_is_pdf(input_path)
	bytes_max := MARKDOWN_BYTES_MAX
	limit_label := "16 MiB"
	if is_pdf {
		bytes_max = markdown_to_pdf.PDF_BYTES_MAX
		limit_label = "64 MiB"
	}
	contents, read_ok := main_read_file(&Main_Read_File_Input{
		Name: input_path, Bytes_Max: bytes_max, Limit_Label: limit_label,
	})
	if !read_ok {
		return EXIT_FAILURE
	}
	return main_convert_to_path(contents, is_pdf, output_path)
}

// Renders the built-in showcase to the OS temp directory and opens it.
func main_golden() (status_code int) {
	return main_render_then_open([]byte(GOLDEN_SHOWCASE), golden_path())
}

// Evaluated at call time so the process's $TMPDIR is always respected.
func golden_path() (path string) {
	return filepath.Join(os.TempDir(), "markdown_to_pdf_golden.pdf")
}

// Renders the command's input file to the OS temp directory and opens it,
// overwriting any prior preview unconditionally.
func main_preview(command cli.Command) (status_code int) {
	input_path := cli.Get_Option(command.Arguments, "input").Value.(string)
	is_pdf := main_input_is_pdf(input_path)
	bytes_max := MARKDOWN_BYTES_MAX
	limit_label := "16 MiB"
	if is_pdf {
		bytes_max = markdown_to_pdf.PDF_BYTES_MAX
		limit_label = "64 MiB"
	}
	contents, read_ok := main_read_file(&Main_Read_File_Input{
		Name: input_path, Bytes_Max: bytes_max, Limit_Label: limit_label,
	})
	if !read_ok {
		return EXIT_FAILURE
	}
	return main_convert_then_open(contents, is_pdf, main_preview_path(input_path))
}

// A preview is written to the OS temp directory, named for the input's base so
// previewing several files does not collide.
func main_preview_path(input_path string) (preview_path string) {
	base_name := filepath.Base(input_path)
	stem := strings.TrimSuffix(base_name, filepath.Ext(base_name))
	if main_input_is_pdf(input_path) {
		return filepath.Join(os.TempDir(), stem+".md")
	}
	return filepath.Join(os.TempDir(), stem+".pdf")
}

// Renders markdown to path, overwriting it, then opens the result in the default
// viewer; the path is reported so the caller knows where it landed.
func main_render_then_open(markdown []byte, path string) (status_code int) {
	return main_convert_then_open(markdown, false, path)
}

// Converts source before it opens path, then reports and opens a successful
// preview. Parse failures therefore leave any prior preview intact.
func main_convert_then_open(source []byte, is_pdf bool, path string) (status_code int) {
	status := main_convert_to_path(source, is_pdf, path)
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
	return main_convert_to_path(markdown, false, output_path)
}

// Converts source completely before it opens output_path. This ordering is
// load-bearing for explicit output paths because a malformed PDF must not
// truncate the caller's existing file.
func main_convert_to_path(source []byte, is_pdf bool, output_path string) (status_code int) {
	document := source
	if is_pdf {
		markdown, convert_err := markdown_to_pdf.PDF_To_Markdown(source)
		if convert_err != nil {
			fmt.Fprintf(os.Stderr, "markdown_to_pdf: %v\n", convert_err)
			return EXIT_FAILURE
		}
		document = markdown
	} else {
		document = markdown_to_pdf.Render(source)
	}
	return main_write_output(document, output_path)
}

func main_write_output(document []byte, output_path string) (status_code int) {
	output, create_err := os.Create(output_path)
	if create_err != nil {
		fmt.Fprintf(os.Stderr, "markdown_to_pdf: %v\n", create_err)
		return EXIT_FAILURE
	}
	_, write_err := output.Write(document)
	if write_err != nil {
		fmt.Fprintf(os.Stderr, "markdown_to_pdf: %v\n", write_err)
		output.Close()
		return EXIT_FAILURE
	}
	close_err := output.Close()
	if close_err != nil {
		fmt.Fprintf(os.Stderr, "markdown_to_pdf: %v\n", close_err)
		return EXIT_FAILURE
	}
	return 0
}

type Main_Output_Path_Input struct {
	// Input is the source Markdown path.
	Input string
	// Output is the explicit -out path, or empty to derive from Input.
	Output string
}

// Returns the explicit -out path when set. A derived path uses .md for PDF
// input and .pdf for every other input.
func main_output_path(input *Main_Output_Path_Input) (output_path string) {
	if input.Output != "" {
		return input.Output
	}
	extension := ".pdf"
	if main_input_is_pdf(input.Input) {
		extension = ".md"
	}
	return strings.TrimSuffix(input.Input, filepath.Ext(input.Input)) + extension
}

func main_input_is_pdf(path string) (is_pdf bool) {
	return strings.EqualFold(filepath.Ext(path), ".pdf")
}

func main_path_exists(path string) (exists bool) {
	_, stat_err := os.Stat(path)
	return stat_err == nil
}

// Main_Read_File_Input describes a bounded command input read.
type Main_Read_File_Input struct {
	// Name is the input path.
	Name string
	// Bytes_Max is the direction-specific size limit.
	Bytes_Max int
	// Limit_Label is the diagnostic form of Bytes_Max.
	Limit_Label string
}

// Reads the named file into one fixed buffer. ok is false, with a stderr
// message, when the file cannot be opened, overflows the cap, or errors.
func main_read_file(input *Main_Read_File_Input) (contents []byte, ok bool) {
	file, open_err := os.Open(input.Name)
	if open_err != nil {
		fmt.Fprintf(os.Stderr, "markdown_to_pdf: %v\n", open_err)
		return nil, false
	}
	defer file.Close()
	buffer := make([]byte, input.Bytes_Max+1)
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
	// The extra byte distinguishes an input exactly at the cap from overflow.
	fmt.Fprintf(os.Stderr, "markdown_to_pdf: input exceeds %s\n", input.Limit_Label)
	return nil, false
}

// One Markdown document exercising every feature the converter renders — from
// smart punctuation and box-drawing diagrams to wrapping table cells — long
// enough to spill onto a second page so pagination shows too.
const GOLDEN_SHOWCASE = "# markdown_to_pdf showcase\n" +
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
