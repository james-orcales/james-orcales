// Package disk_usage reports the disk utilization of a directory tree: it walks the
// tree once, charges every file's bytes to itself and to each of its ancestor
// directories, and renders the entries largest-first so the heaviest consumers surface
// at the top.
package disk_usage

import (
	"fmt"
	"io"
	"io/fs"
	"path"
	"slices"
	"strconv"
	"strings"
	"text/tabwriter"

	"local/james-orcales/shared/cli"
	"local/james-orcales/shared/flatjson"
)

// The process exit codes.
const exit_success = 0
const exit_usage = 2
const exit_failure = 1

// Depth_Max sentinel: render every entry, however deep.
const depth_unlimited = -1

// Entry is one path in the tree with its cumulative byte size: for a file, its own
// bytes; for a directory, the sum of every file beneath it.
type Entry struct {
	// Path is the slash-separated path relative to the analyzed root; the root itself
	// is ".".
	Path string
	// Bytes is the cumulative size charged to this path.
	Bytes int64
	// Is_Directory distinguishes a directory from a file.
	Is_Directory bool
	// Depth is the distance from the root: the root is zero, each path segment adds one.
	Depth int
}

// Report is the result of analyzing a tree: the root label the user passed, and every
// entry sorted largest-first.
type Report struct {
	// Root is the directory argument, used as the display label for the "." entry.
	Root string
	// Entries are every file and directory, sorted directories-first, then by Bytes
	// descending and Path ascending so the order is deterministic across runs.
	Entries []Entry
}

// A File_Identifier uniquely identifies the data a file points at, so two hardlinks to one inode
// compare equal. The zero value means the host could not determine an identity; the
// analysis then counts the file unconditionally rather than risk merging distinct files.
type File_Identifier struct {
	// Device is the storage device the inode lives on.
	Device int64
	// Inode is the file's inode number on that device.
	Inode uint64
}

// Analyze_Input carries the tree to analyze and the host's per-file disk accounting.
type Analyze_Input struct {
	// Root is the directory argument, used as the display label for the "." entry.
	Root string
	// File_System is the tree to walk, rooted at the directory.
	File_System fs.FS
	// Disk_Usage reports the physical bytes a file occupies and its identity. Physical
	// allocation — not logical size — is what "disk usage" means, so a sparse or cloned
	// file costs only its real extent; the identity lets a hardlink be charged once.
	Disk_Usage func(info fs.FileInfo) (bytes int64, identifier File_Identifier)
}

// Main_Input carries the command line and the host bindings Main needs. The library
// tier does no ambient I/O, so the directory is opened and measured through injected
// functions.
type Main_Input struct {
	// Arguments is the command line, including the program name.
	Arguments []string
	// Output is where the table is written.
	Output io.Writer
	// Error_Output is where usage and errors are written.
	Error_Output io.Writer
	// Open views a directory as a read-only file system rooted at it.
	Open func(root string) (file_system fs.FS)
	// Disk_Usage reports a file's physical bytes and identity; see Analyze_Input.
	Disk_Usage func(info fs.FileInfo) (bytes int64, identifier File_Identifier)
}

// Main parses the command line, analyzes the directory, and renders the table,
// returning a process exit code.
func Main(input *Main_Input) (status_code int) {
	program := main_program()
	command, parse_err := cli.Program_Parse(&program, input.Arguments)
	if parse_err != nil {
		fmt.Fprintf(input.Error_Output, "disk_usage: %v\n\n", parse_err)
		cli.Print_Help(input.Error_Output, program)
		return exit_usage
	}

	directory_only := cli.Get_Option(command.Flags, "dir-only").Value.(bool)
	files_only := cli.Get_Option(command.Flags, "files-only").Value.(bool)
	// The two filters name disjoint, exhaustive halves of the tree; asking for both at
	// once is a contradiction, not a request to be silently resolved one way.
	if directory_only {
		if files_only {
			fmt.Fprintf(input.Error_Output,
				"disk_usage: -dir-only and -files-only are mutually exclusive\n")
			return exit_usage
		}
	}

	minimum_text := cli.Get_Option(command.Flags, "minimum").Value.(string)
	minimum, minimum_err := parse_bytes(minimum_text)
	if minimum_err != nil {
		fmt.Fprintf(input.Error_Output, "disk_usage: %v\n", minimum_err)
		return exit_usage
	}

	root := cli.Get_Option(command.Arguments, "dir").Value.(string)
	report, analyze_err := Analyze(Analyze_Input{
		Root:        root,
		File_System: input.Open(root),
		Disk_Usage:  input.Disk_Usage,
	})
	if analyze_err != nil {
		fmt.Fprintf(input.Error_Output, "disk_usage: %v\n", analyze_err)
		return exit_failure
	}

	render_input := Render_Input{
		Report:         report,
		Depth_Max:      cli.Get_Option(command.Flags, "depth").Value.(int),
		Minimum:        minimum,
		Directory_Only: directory_only,
		Files_Only:     files_only,
	}
	if cli.Get_Option(command.Flags, "json").Value.(bool) {
		json_err := Render_Json(input.Output, render_input)
		if json_err != nil {
			fmt.Fprintf(input.Error_Output, "disk_usage: %v\n", json_err)
			return exit_failure
		}
		return exit_success
	}
	Render(input.Output, render_input)
	return exit_success
}

// Declares the disk_usage command line: a commandless program taking one directory and
// the display filters.
func main_program() (program cli.Program) {
	return cli.New_Single(cli.New_Single_Input{
		Label:       "disk_usage",
		Description: "report the disk utilization of a directory tree",
		Arguments: []cli.Option{
			cli.New_Argument[string](cli.New_Argument_Input{
				Label:       "dir",
				Description: "the directory to analyze",
			}),
		},
		Flags: []cli.Option{
			cli.New_Flag(cli.New_Flag_Input[bool]{
				Label:       "dir-only",
				Value:       false,
				Description: "list only directories",
			}),
			cli.New_Flag(cli.New_Flag_Input[bool]{
				Label:       "files-only",
				Value:       false,
				Description: "list only files",
			}),
			// Default to just the root, so a bare run answers "how big is this
			// directory?" without unrolling the whole tree; deeper levels are opt-in.
			// The full tree is always walked, so the sizes stay correct however shallow
			// the display is cut. Pass -1 to show every level.
			cli.New_Flag(cli.New_Flag_Input[int]{
				Label:       "depth",
				Value:       1,
				Description: "deepest level to display (root is 1; -1 shows all)",
			}),
			cli.New_Flag(cli.New_Flag_Input[bool]{
				Label:       "json",
				Value:       false,
				Description: "emit the entries as JSON instead of a table",
			}),
			cli.New_Flag(cli.New_Flag_Input[string]{
				Label:       "minimum",
				Value:       "1MiB",
				Description: "hide entries smaller than this (e.g. 500KiB)",
			}),
		},
	})
}

// Analyze walks the file system once and returns every file and directory with its
// cumulative disk usage. Each file's physical bytes — reported by the host's Disk_Usage —
// are charged to itself and to each ancestor directory up to the root, counting a
// hardlinked inode only once, so a directory's bytes are the disk its subtree occupies
// and the root holds the grand total.
func Analyze(input Analyze_Input) (report Report, err error) {
	bytes_by_path := map[string]int64{}
	is_directory := map[string]bool{}
	counted := map[File_Identifier]bool{}

	walk_err := fs.WalkDir(input.File_System, ".", func(
		name string, entry fs.DirEntry, walk_err error,
	) (err error) {
		if walk_err != nil {
			return walk_err
		}
		if entry.IsDir() {
			// Record the directory so a file-less one still appears, with zero bytes.
			is_directory[name] = true
			bytes_by_path[name] += 0
			return nil
		}
		information, info_err := entry.Info()
		if info_err != nil {
			return info_err
		}
		bytes, identity := input.Disk_Usage(information)
		// A hardlink points at data already charged elsewhere, so it costs no further
		// disk: count an identity once, at the first path that reaches it, and zero on
		// every later path, the way du attributes a shared inode.
		if identity != (File_Identifier{}) {
			if counted[identity] {
				bytes = 0
			}
			counted[identity] = true
		}
		// Charge the file's bytes to itself and walk up to ".", charging every ancestor
		// directory the same amount.
		for ancestor := name; ; ancestor = path.Dir(ancestor) {
			bytes_by_path[ancestor] += bytes
			if ancestor == "." {
				break
			}
		}
		return nil
	})
	if walk_err != nil {
		return Report{}, walk_err
	}
	return Report{
		Root:    input.Root,
		Entries: report_entries(bytes_by_path, is_directory),
	}, nil
}

// Turns the accumulated per-path bytes into sorted entries: directories first, then
// files, within a kind largest-first with ties broken on path. The ordering is fixed
// here — not left to map iteration — so the report is deterministic, and grouping
// folders ahead of files keeps the directory totals, the headline of disk usage,
// together at the top rather than interleaved with the files they contain.
func report_entries(
	bytes_by_path map[string]int64, is_directory map[string]bool,
) (entries []Entry) {
	entries = make([]Entry, 0, len(bytes_by_path))
	for name, byte_size := range bytes_by_path {
		entries = append(entries, Entry{
			Path:         name,
			Bytes:        byte_size,
			Is_Directory: is_directory[name],
			Depth:        path_depth(name),
		})
	}
	slices.SortFunc(entries, func(left, right Entry) (order int) {
		if left.Is_Directory != right.Is_Directory {
			if left.Is_Directory {
				return -1
			}
			return 1
		}
		if left.Bytes != right.Bytes {
			return int(right.Bytes - left.Bytes)
		}
		return strings.Compare(left.Path, right.Path)
	})
	return entries
}

// Returns the depth of a path as the number of components in its displayed label: the
// root directory itself is depth one, and each slash-separated segment beneath it adds
// one, so "a" is depth two and "a/b" is depth three. The root counts as a level because
// it is a printed row, so -depth=3 means "the root and two levels below it".
func path_depth(name string) (depth int) {
	if name == "." {
		return 1
	}
	return strings.Count(name, "/") + 2
}

// Render_Input is the input for Render.
type Render_Input struct {
	// Report is the analyzed tree.
	Report Report
	// Depth_Max is the deepest level to print; depth_unlimited (-1) prints everything.
	Depth_Max int
	// Minimum is the smallest size that prints; an entry below it is hidden, trimming the
	// long tail of small entries. Zero keeps every non-empty entry.
	Minimum int64
	// Directory_Only prints only directories.
	Directory_Only bool
	// Files_Only prints only files.
	Files_Only bool
}

// Render writes the report as an aligned two-column table of size and path, the rows
// grouped by depth under a per-depth header so each level reads as its own block.
// Filtering only hides rows; the sizes shown are the full cumulative totals, so a
// shallow display still reflects the bytes of the descendants it omits.
func Render(output io.Writer, input Render_Input) {
	writer := tabwriter.NewWriter(output, 0, 8, 2, ' ', 0)
	for index, depth := range render_depths(input) {
		// A blank line sets each block apart; the header line itself already breaks the
		// tabwriter column run, so every depth aligns its size column independently.
		if index > 0 {
			fmt.Fprintln(writer)
		}
		fmt.Fprintf(writer, "Depth %d\n", depth)
		for _, entry := range input.Report.Entries {
			if entry.Depth != depth {
				continue
			}
			if !render_keeps(input, entry) {
				continue
			}
			fmt.Fprintf(writer, "%s\t%s\n",
				format_bytes(entry.Bytes), render_label(input.Report, entry))
		}
	}
	writer.Flush()
}

// A json_entry is one rendered row in serialized form: a flat record flatjson can emit
// as one object in a top-level array. Path is the display label, not the relative path,
// so the JSON names the same full path the table prints.
type json_entry struct {
	Path         string `json:"path"`
	Bytes        int64  `json:"bytes"`
	Is_Directory bool   `json:"is_directory"`
	Depth        int    `json:"depth"`
}

// Render_Json writes the kept entries as a top-level JSON array of flat objects, one per
// row. It applies the same filters as Render, so -json reflects exactly the rows the
// table would show, only as machine-readable output instead of a grouped table.
func Render_Json(output io.Writer, input Render_Input) (err error) {
	entries := []json_entry{}
	for _, entry := range input.Report.Entries {
		if !render_keeps(input, entry) {
			continue
		}
		entries = append(entries, json_entry{
			Path:         render_label(input.Report, entry),
			Bytes:        entry.Bytes,
			Is_Directory: entry.Is_Directory,
			Depth:        entry.Depth,
		})
	}
	return flatjson.Marshal_Write(output, entries)
}

// Returns the distinct depths of the entries that survive filtering, ascending. The
// report is already sorted directories-first then by size, so iterating one depth at a
// time preserves that order within each block.
func render_depths(input Render_Input) (depths []int) {
	seen := map[int]bool{}
	for _, entry := range input.Report.Entries {
		if !render_keeps(input, entry) {
			continue
		}
		if seen[entry.Depth] {
			continue
		}
		seen[entry.Depth] = true
		depths = append(depths, entry.Depth)
	}
	slices.Sort(depths)
	return depths
}

// Reports whether an entry passes the active filters. A zero-byte entry is never
// printed: it contributes nothing to the report and only adds noise.
func render_keeps(input Render_Input, entry Entry) (keep bool) {
	if entry.Bytes == 0 {
		return false
	}
	if entry.Bytes < input.Minimum {
		return false
	}
	if input.Depth_Max != depth_unlimited {
		if entry.Depth > input.Depth_Max {
			return false
		}
	}
	if input.Directory_Only {
		if !entry.Is_Directory {
			return false
		}
	}
	if input.Files_Only {
		if entry.Is_Directory {
			return false
		}
	}
	return true
}

// Returns the display path for an entry: the root label for ".", otherwise the root
// label joined to the relative path so each row names a full path.
func render_label(report Report, entry Entry) (label string) {
	if entry.Path == "." {
		return report.Root
	}
	return path.Join(report.Root, entry.Path)
}

// Parses a human-readable size like "1MiB", "500KiB", or a bare "1048576" into bytes.
// Units are binary (powers of 1024) to match the tool's output, and the magnitude must
// be a whole number — the deterministic tier carries no floating point, and a threshold
// reads fine as a whole count of its unit.
func parse_bytes(text string) (bytes int64, err error) {
	trimmed := strings.TrimSpace(text)
	digits := 0
	for digits < len(trimmed) {
		character := trimmed[digits]
		if character < '0' {
			break
		}
		if character > '9' {
			break
		}
		digits++
	}
	if digits == 0 {
		return 0, fmt.Errorf("size %q must start with a whole number", text)
	}
	magnitude, parse_err := strconv.ParseInt(trimmed[:digits], 10, 64)
	if parse_err != nil {
		return 0, fmt.Errorf("size %q is out of range", text)
	}
	unit := strings.ToLower(strings.TrimSpace(trimmed[digits:]))
	multiplier, known := size_unit_bytes(unit)
	if !known {
		return 0, fmt.Errorf("size %q has an unknown unit (try KiB, MiB, GiB)", text)
	}
	return magnitude * multiplier, nil
}

// Maps a lowercased unit suffix to its byte count, treating the SI-style and bare-letter
// spellings as the same binary unit the tool prints. An empty suffix is plain bytes.
func size_unit_bytes(unit string) (multiplier int64, known bool) {
	const kibibyte = 1024
	switch unit {
	case "", "b":
		return 1, true
	case "k", "kb", "kib":
		return kibibyte, true
	case "m", "mb", "mib":
		return kibibyte * kibibyte, true
	case "g", "gb", "gib":
		return kibibyte * kibibyte * kibibyte, true
	case "t", "tb", "tib":
		return kibibyte * kibibyte * kibibyte * kibibyte, true
	case "p", "pb", "pib":
		return kibibyte * kibibyte * kibibyte * kibibyte * kibibyte, true
	}
	return 0, false
}

// Formats a byte count as a human-readable size with one decimal place past the byte
// unit, e.g. 1536 becomes "1.5 KiB" and 512 stays "512 B". The arithmetic is
// integer-only — the whole part and the single decimal digit are derived from the
// remainder — so the package stays free of floating point. Units ascend in powers of
// 1024 to match how the filesystem accounts for space.
func format_bytes(byte_size int64) (text string) {
	byte_units := []string{"B", "KiB", "MiB", "GiB", "TiB", "PiB"}
	if byte_size < 1024 {
		return fmt.Sprintf("%d %s", byte_size, byte_units[0])
	}
	// Climb to the largest unit whose value still divides into the count at least once;
	// divisor is the number of bytes in one of that unit.
	unit_index := 0
	divisor := int64(1)
	for unit_index < len(byte_units)-1 && byte_size >= divisor*1024 {
		divisor *= 1024
		unit_index++
	}
	whole := byte_size / divisor
	tenths := (byte_size % divisor) * 10 / divisor
	return fmt.Sprintf("%d.%d %s", whole, tenths, byte_units[unit_index])
}
