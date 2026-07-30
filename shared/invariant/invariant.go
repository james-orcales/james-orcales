// Package invariant exposes eager guards and bare observations: Always, Range, and Enum panic
// at their own callsite when violated, while Sometimes records which branch of a condition the
// suite witnessed. The _Invariants composition bundles a type's assertions so they travel with
// the type.
package invariant

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io"
	"io/fs"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

// ASSERTION_FAILURE_MESSAGE_PREFIX opens every assertion-failure message.
const ASSERTION_FAILURE_MESSAGE_PREFIX = "🚨 Assertion Failure 🚨: "

// ELEMENT_MESSAGE_SEPARATOR makes structural coverage keys unambiguous because registration
// rejects it in every message.
const ELEMENT_MESSAGE_SEPARATOR = "\x00"

// Bounds registration's backwards walks over fluent receivers. A valid chain stops at its root
// far below this because the independently enforced ordinal space has only 255 links; the larger
// bound keeps malformed syntax from making analysis depend unboundedly on source depth.
const BUNDLE_EXPANSION_STEPS_MAX = 4096

// Bounds the walk up the directory tree searching for a go.mod, so module
// discovery can't loop unboundedly on a pathological path.
const MODULE_SEARCH_DEPTH_MAX = 256

// ASSERTION_KIND_ALWAYS classifies a per-element tracker entry for an Always.
const ASSERTION_KIND_ALWAYS Assertion_Kind = 0

// ASSERTION_KIND_SOMETIMES classifies a per-element tracker entry for a Sometimes.
const ASSERTION_KIND_SOMETIMES Assertion_Kind = 1

// Recorder accumulates assertion observations for one run and identifies each
// element by its caller Site.
type Recorder struct {
	// File_System reads Go source files during AST analysis. Paths are absolute OS paths;
	// lookups strip the leading "/" before calling fs.ReadFile.
	File_System fs.FS

	// Events is the coverage tracker: one entry per registered element bucket,
	// keyed by message and credited as observations arrive.
	Events sync.Map

	// Output receives the coverage-gap report and the orphan/bundle diagnostics.
	Output io.Writer
	// Exit ends the process with a status code; the composition tier wires it to os.Exit.
	Exit func(code int)
	// Tty receives the clean-run success summary so it shows even without `go test -v`.
	Tty io.Writer

	// Is_Test reports a `go test` run (plain, a `-fuzz` coordinator, or a fuzz worker) — every
	// mode that records coverage. Only a benchmark opts out of recording.
	Is_Test bool
	// Is_Fuzz reports a fuzzing run (coordinator or worker).
	Is_Fuzz bool
	// Is_Fuzz_Worker reports a `-test.fuzzworker` subprocess: it runs the fuzzed body (so it
	// records, and persists each newly-covered key via Coverage_Sink for the coordinator to
	// merge), but it does not analyze — its view of coverage is partial. It always enforces.
	Is_Fuzz_Worker bool
	// Is_Benchmark reports a benchmark run, which records and checks nothing.
	Is_Benchmark bool

	// Packages_To_Analyze are the directories whose source is parsed to seed the
	// expected-coverage space.
	Packages_To_Analyze []string

	// Working_Directory resolves the relative entries of Packages_To_Analyze to
	// absolute paths. The composition tier sets it to the process working directory;
	// empty leaves a relative entry relative.
	Working_Directory string

	// Sugar_Package is the import path of the recorder-less sugar tier. When a template
	// resolved from that package is descended, its unqualified invariant calls are recognized;
	// empty keeps identically named calls elsewhere from becoming assertions.
	Sugar_Package string

	// Package_Label is the package's path relative to the module root, derived by
	// Recorder_Register_Packages_For_Analysis and printed in the clean-run summary so
	// the line is identifiable when many packages print to the same terminal. Empty
	// when no go.mod is found or when the registered directory is the module root.
	Package_Label string

	// Coverage_Sink, when set, is called the first time each coverage key+branch is observed.
	// A fuzz worker subprocess wires it to persist its exploration to a shared file so the
	// coordinator can merge it (the coordinator never runs the fuzzed body itself). Nil
	// everywhere else — recording then just bumps the in-process counters.
	Coverage_Sink func(key string, fired_true bool)

	// Merge_Fuzz_Coverage, when set, is called by Recorder_Run_Test_Main after the suite runs
	// and before the analysis. A fuzz coordinator wires it to read every worker's persisted
	// coverage and credit it into the registered grid so the analysis sees what workers found.
	Merge_Fuzz_Coverage func()
}

// Signed spans the primitive signed integer widths.
type Signed interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64
}

// Unsigned spans the primitive unsigned integer widths. uintptr is excluded — it carries an
// address, not a quantity, so numeric bounds on it assert nothing about the domain.
type Unsigned interface {
	~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

// Float spans both floating-point widths.
type Float interface {
	~float32 | ~float64
}

// Integer spans the primitive integer widths the bare guards accept.
type Integer interface {
	Signed | Unsigned
}

// Assertion_Kind discriminates a coverage tracker entry: a per-element Always or Sometimes.
type Assertion_Kind uint8

// Assertion_Metadata is one coverage tracker entry: how often an element's event was observed
// across the run. Seeded at registration, incremented at runtime, scanned by the never-fired
// report.
type Assertion_Metadata struct {
	// Frequency counts true-event observations: an Always or a Sometimes true.
	Frequency atomic.Int64
	// False_Frequency counts false-event observations: a Sometimes false.
	False_Frequency atomic.Int64
	// Kind discriminates the entry: Always or Sometimes.
	Kind Assertion_Kind
	// Message is the identity the entry is keyed by — the element's own message.
	Message string
	// Condition is the source text of the asserted expression, for the gap report.
	Condition string
}

// A Handle_Entry is a resolved tracker slot: the seeded metadata and the tracker key, cached so
// Coverage_Sink can persist it without rebuilding the string.
type Handle_Entry struct {
	// Metadata is the seeded tracker entry, nil when registration seeded none.
	Metadata *Assertion_Metadata
	// Key is the tracker key, cached so Coverage_Sink can persist it without rebuilding it.
	Key string
}

// Bumps entry's metadata: Frequency on a true event, False_Frequency on false. A nil-metadata
// entry (registration seeded none, like a carved cell) is skipped. On the 0→1 transition of a
// branch (its first coverage) it fires Coverage_Sink with the entry's cached key, so a fuzz
// worker persists the cell; atomic Add returns the post-increment value, so the sink fires once
// per branch.
func recorder_increment_entry(recorder *Recorder, entry Handle_Entry, fired_true bool) {
	if entry.Metadata == nil {
		return
	}
	// Seen-ness is the entire signal — the gap report reads zero versus nonzero — so after
	// the first hit an add would only ping-pong the entry's cache line across workers. The
	// plain load may race another first hit; the add below still elects exactly one sink call.
	if fired_true {
		if entry.Metadata.Frequency.Load() != 0 {
			return
		}
		if entry.Metadata.Frequency.Add(1) == 1 {
			if recorder.Coverage_Sink != nil {
				recorder.Coverage_Sink(entry.Key, true)
			}
		}
		return
	}
	if entry.Metadata.False_Frequency.Load() != 0 {
		return
	}
	if entry.Metadata.False_Frequency.Add(1) == 1 {
		if recorder.Coverage_Sink != nil {
			recorder.Coverage_Sink(entry.Key, false)
		}
	}
}

// Bumps the seeded entry at key — the coordinator's merge path (Recorder_Merge_Fuzz_Coverage_From),
// which holds only string keys, not the runtime's cached handles. A missing entry is skipped.
func recorder_increment(recorder *Recorder, key string, fired_true bool) {
	value, ok := recorder.Events.Load(key)
	if !ok {
		return
	}
	recorder_increment_entry(
		recorder, Handle_Entry{Metadata: value.(*Assertion_Metadata), Key: key}, fired_true)
}

// Fuzz_Coverage_Line encodes one covered (key, branch) as the line a fuzz worker appends to
// the shared coverage file: base64(key) + "\t" + "T"/"F" + "\n". base64 because a key carries
// the NUL element-separator and otherwise-arbitrary bytes; the trailing newline makes the file
// line-oriented for the coordinator's merge. It is one string, so the worker writes it with a
// single Write under O_APPEND (the lock-free-append requirement).
func Fuzz_Coverage_Line(key string, fired_true bool) (line string) {
	branch := "F"
	if fired_true {
		branch = "T"
	}
	return base64.StdEncoding.EncodeToString([]byte(key)) + "\t" + branch + "\n"
}

func recorder_merge_process_line(recorder *Recorder, line string) {
	tab_offset := strings.IndexByte(line, '\t')
	if tab_offset < 0 {
		return
	}
	key, decode_error := base64.StdEncoding.DecodeString(line[:tab_offset])
	if decode_error != nil {
		return
	}
	recorder_merge_increment(recorder, string(key), line[tab_offset+1:] == "T")
}

// Merge resolves the emitted string key directly; a key with no seeded entry is skipped like any
// runtime increment.
func recorder_merge_increment(recorder *Recorder, key string, fired_true bool) {
	recorder_increment(recorder, key, fired_true)
}

// Recorder_Merge_Fuzz_Coverage_From unions the coverage a fuzz coordinator reads from r
// (the workers' shared file, one Fuzz_Coverage_Line per line) into the registered grid: each
// covered (key, branch) marks that branch non-zero. Binary — per-process counts are not summed
// across workers. A malformed or partial trailing line is skipped (a worker killed mid-write
// costs at most its last line); a key with no seeded entry is skipped, like any runtime increment.
func Recorder_Merge_Fuzz_Coverage_From(recorder *Recorder, r io.Reader) {
	var buffer [4096]byte
	var partial string
	var read_error error
	for read_error == nil {
		var n int
		n, read_error = r.Read(buffer[:])
		if n > 0 {
			chunk := partial + string(buffer[:n])
			newline_offset := strings.IndexByte(chunk, '\n')
			for newline_offset >= 0 {
				recorder_merge_process_line(recorder, chunk[:newline_offset])
				chunk = chunk[newline_offset+1:]
				newline_offset = strings.IndexByte(chunk, '\n')
			}
			partial = chunk
		}
	}
}

// Recorder_Register_Packages_For_Analysis parses every non-test .go file under
// the given directories and seeds recorder.Events with one entry per element
// bucket and one per non-carved tuple of each invariant.Dot_Product call. That
// seeded set is the expected-coverage space the never-fired report scans after
// the run; literal invariant.X selectors and *_Invariants bundles are recognised.
//
// Directories default to recorder.Packages_To_Analyze when none are passed; a
// directory may glob, a `*` segment matching one path element and `**` any depth,
// expanded against File_System. Each assertion is keyed by its message; a duplicate
// message, or one that is not a string literal, fails registration (see
// recorder_check_duplicate_messages / recorder_check_non_literal_messages).
func Recorder_Register_Packages_For_Analysis(recorder *Recorder, directories ...string) {
	if len(directories) > 0 {
		recorder.Packages_To_Analyze = directories
	}
	file_set := token.NewFileSet()
	var files []*ast.File
	module_path := ""
	module_root := ""
	for _, directory := range recorder.Packages_To_Analyze {
		// A filepath.Abs here would reach the OS for the working directory, which a pure
		// package must not do; Working_Directory is injected so this stays pure.
		absolute := directory
		if !filepath.IsAbs(absolute) {
			absolute = filepath.Join(recorder.Working_Directory, absolute)
		}
		absolute = filepath.Clean(absolute)
		expanded_directories := recorder_expand_directories(recorder.File_System, absolute)
		for _, expanded := range expanded_directories {
			if module_path == "" {
				module_path, module_root = recorder_module(recorder, expanded)
			}
			if recorder.Package_Label == "" {
				if module_root != "" {
					relative := strings.TrimPrefix(expanded, module_root)
					relative = strings.TrimPrefix(relative, "/")
					if relative != "" {
						recorder.Package_Label = relative
					}
				}
			}
			parsed := recorder_parse_directory(recorder.File_System, file_set, expanded)
			files = append(files, parsed...)
		}
	}
	index := &Bundle_Index{
		File_System:   recorder.File_System,
		File_Set:      file_set,
		Module_Path:   module_path,
		Module_Root:   module_root,
		Sugar_Package: recorder.Sugar_Package,
		Same_Set:      ast_index_functions(files),
		Loaded:        map[string]map[string]Indexed_Function{},
	}
	reg := &Registration{Seen_Prefix: map[string]bool{}}
	for _, file := range files {
		recorder_register_file(recorder, file_set, file, index, reg)
	}
	recorder_check_bundle_control_flow(recorder, file_set, files)
	recorder_check_primitive_bundles(recorder, file_set, files, index)
	recorder_check_unresolved(recorder, reg.Unresolved)
	recorder_check_non_literal_messages(recorder, reg.Non_Literal)
	recorder_check_duplicate_messages(recorder, reg.Collision)
	recorder_check_unresolved_bounds(recorder, reg.Unresolved_Bound)
	recorder_check_invalid_chains(recorder, reg.Invalid_Chain)
}

// Reports whether name exists in recorder.File_System.
func recorder_has_entry(recorder *Recorder, name string) (exists bool) {
	_, stat_error := fs.Stat(recorder.File_System, name)
	return stat_error == nil
}

// Walks up from start_directory for a go.mod, returning the module path it
// declares and the absolute directory containing it. Both are "" when none is
// found within MODULE_SEARCH_DEPTH_MAX — cross-package resolution then degrades
// to same-package bundles only.
func recorder_module(
	recorder *Recorder, start_directory string,
) (module_path string, module_root string) {
	directory := start_directory
	for range MODULE_SEARCH_DEPTH_MAX {
		relative := path.Join(strings.TrimPrefix(directory, "/"), "go.mod")
		SOURCE, read_error := fs.ReadFile(recorder.File_System, relative)
		if read_error == nil {
			return parse_module_path(SOURCE), directory
		}
		PARENT := path.Dir(directory)
		if PARENT == directory {
			break
		}
		directory = PARENT
	}
	return "", ""
}

// Returns the module path declared by a go.mod's `module` directive, or "" when
// absent — a line scan, no golang.org/x/mod dependency.
func parse_module_path(SOURCE []byte) (module_path string) {
	for _, line := range strings.Split(string(SOURCE), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		if fields[0] != "module" {
			continue
		}
		return fields[1]
	}
	return ""
}

// Parses the non-test .go files directly under the absolute Directory into AST
// files. File_System is rooted at "/", so the leading "/" is stripped to address
// it; the parsed file's name is the absolute path, used only for diagnostics now
// (identity is the message, not the position). Subdirectories are skipped — one
// directory is one package.
func recorder_parse_directory(
	file_system fs.FS, file_set *token.FileSet, directory string,
) (files []*ast.File) {
	root := strings.TrimPrefix(directory, "/")
	fs.WalkDir(file_system, root, func(
		file_path string, entry fs.DirEntry, walk_error error,
	) (err error) {
		if walk_error != nil {
			return walk_error
		}
		if entry.IsDir() {
			if file_path == root {
				return nil
			}
			return fs.SkipDir
		}
		if !strings.HasSuffix(file_path, ".go") {
			return nil
		}
		if strings.HasSuffix(file_path, "_test.go") {
			return nil
		}
		SOURCE, read_error := fs.ReadFile(file_system, file_path)
		if read_error != nil {
			return nil
		}
		name := "/" + file_path
		file, parse_error := parser.ParseFile(
			file_set, name, SOURCE, parser.SkipObjectResolution,
		)
		if parse_error == nil {
			files = append(files, file)
		}
		return nil
	})
	return files
}

// One frontier entry of the directory-glob walk: a directory reached so far and the
// index of the next pattern segment to match against its children.
type Recorder_Glob_State struct {
	// Directory is a directory reached so far in the glob walk.
	Directory string
	// Index is the next pattern segment to match against this directory's children.
	Index int
}

// Expands a directory pattern against the file system into concrete directories. A
// `*` segment matches one path element; a `**` segment matches zero or more, so
// `a/**` covers a and every directory beneath it. A pattern holding no `*` is
// returned unchanged. Results are unique and lexically sorted for a stable seed.
func recorder_expand_directories(file_system fs.FS, pattern string) (directories []string) {
	if !strings.Contains(pattern, "*") {
		return []string{pattern}
	}
	rooted := strings.HasPrefix(pattern, "/")
	segments := strings.Split(strings.TrimPrefix(pattern, "/"), "/")
	frontier := []Recorder_Glob_State{{Directory: ".", Index: 0}}
	matched := map[string]bool{}
	for len(frontier) > 0 {
		current := frontier[len(frontier)-1]
		frontier = frontier[:len(frontier)-1]
		if current.Index == len(segments) {
			matched[current.Directory] = true
			continue
		}
		frontier = append(frontier,
			recorder_glob_step(file_system, current, segments[current.Index])...)
	}
	for directory := range matched {
		result := directory
		if rooted {
			result = "/" + directory
		}
		directories = append(directories, result)
	}
	sort.Strings(directories)
	return directories
}

// The frontier entries reached by matching one pattern segment against current's
// children. A `**` also matches in place (zero elements) and stays in play as it
// descends, so it spans any depth.
func recorder_glob_step(
	file_system fs.FS, current Recorder_Glob_State, segment string,
) (next []Recorder_Glob_State) {
	children := recorder_child_directories(file_system, current.Directory)
	if segment == "**" {
		next = append(next, Recorder_Glob_State{
			Directory: current.Directory, Index: current.Index + 1})
		for _, CHILD := range children {
			next = append(next,
				Recorder_Glob_State{Directory: CHILD, Index: current.Index})
		}
		return next
	}
	for _, CHILD := range children {
		matched, _ := path.Match(segment, path.Base(CHILD))
		if !matched {
			continue
		}
		next = append(next, Recorder_Glob_State{Directory: CHILD, Index: current.Index + 1})
	}
	return next
}

// The immediate subdirectories of directory in the file system, as fs paths.
func recorder_child_directories(file_system fs.FS, directory string) (children []string) {
	entries, read_error := fs.ReadDir(file_system, directory)
	if read_error != nil {
		return nil
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		CHILD := entry.Name()
		if directory != "." {
			CHILD = directory + "/" + entry.Name()
		}
		children = append(children, CHILD)
	}
	return children
}

// An Indexed_Function is a discovered FuncDecl paired with the local-name →
// import-path map of the file it lives in (so the bundle's own qualified
// sub-calls resolve) and whether it lives in the sugar package (so the descent
// recognises its unqualified primitive calls).
type Indexed_Function struct {
	// Declaration is the discovered function declaration.
	Declaration *ast.FuncDecl
	// Imports maps the file's local names to import paths, so qualified sub-calls resolve.
	Imports map[string]string
	// Is_Sugar reports whether the function lives in the sugar package.
	Is_Sugar bool
}

// A Bundle_Index resolves a *_Invariants bundle call to its declaration. Same_Set
// holds the analyzed packages' functions by bare name (same-package bundles); a
// qualified call resolves cross-package within the module via Module_Path /
// Module_Root, lazily parsing and caching each package in Loaded. A bundle outside
// this module is unresolvable.
type Bundle_Index struct {
	// File_System is the filesystem the module's packages are parsed from.
	File_System fs.FS
	// File_Set is the token file set cross-package parses are recorded in.
	File_Set *token.FileSet
	// Module_Path is the module's import-path prefix, used to detect in-module qualified calls.
	Module_Path string
	// Module_Root is the module's absolute root directory on File_System.
	Module_Root string
	// Sugar_Package is the import path of the sugar package.
	Sugar_Package string
	// Same_Set holds the analyzed packages' functions by bare name (same-package bundles).
	Same_Set map[string]Indexed_Function
	// Loaded caches lazily parsed cross-package functions, keyed by import path then bare name.
	Loaded map[string]map[string]Indexed_Function
	// Constants maps package constants to their value expressions. A flat bare-name index, like
	// Same_Set, is sufficient because one analysis covers one package tree.
	Constants map[string]ast.Expr
}

// Maps each function name to its declaration and its file's imports, for
// descending *_Invariants bundles. A later definition wins on a name collision.
func ast_index_functions(files []*ast.File) (functions map[string]Indexed_Function) {
	functions = map[string]Indexed_Function{}
	for _, file := range files {
		imports := ast_file_imports(file)
		for _, declaration := range file.Decls {
			function, is_function := declaration.(*ast.FuncDecl)
			if !is_function {
				continue
			}
			functions[function.Name.Name] = Indexed_Function{
				Declaration: function,
				Imports:     imports,
			}
		}
	}
	return functions
}

// Registers every invariant.Dot_Product call in one parsed file. The file's
// import map is threaded down so a qualified cross-package bundle resolves.
func recorder_register_file(
	recorder *Recorder, file_set *token.FileSet, file *ast.File, index *Bundle_Index,
	reg *Registration,
) {
	imports := ast_file_imports(file)
	allow_unqualified := false
	if index.Sugar_Package != "" {
		file_package := recorder_file_package(file_set, file, index)
		allow_unqualified = file_package == index.Sugar_Package
	}
	for _, declaration := range file.Decls {
		function, is_function := declaration.(*ast.FuncDecl)
		if !is_function {
			continue
		}
		if function.Body == nil {
			continue
		}
		recorder_register_function(
			recorder, file_set, function, imports, index, reg, allow_unqualified)
	}
}

// Registers each invariant call in the function: any call may be a bare eager Always.
func recorder_register_function(
	recorder *Recorder, file_set *token.FileSet, function *ast.FuncDecl,
	imports map[string]string, index *Bundle_Index, reg *Registration, allow_unqualified bool,
) {
	ast.Inspect(function.Body, func(node ast.Node) (descend bool) {
		call, is_call := node.(*ast.CallExpr)
		if !is_call {
			return true
		}
		// This walk is the only registration that sees a bare eager Always — keyed by its
		// own message like any other.
		recorder_register_eager_always(recorder, file_set, call, reg)
		return true
	})
}

// Returns the name of a _Invariants function's trailing namespace parameter — the grid identity
// its self-emitted Dot_Product is prefixed by, typed string or Namespace. "" when there is no
// such parameter.
func ast_namespace_parameter(function *ast.FuncDecl) (name string) {
	if function.Type.Params == nil {
		return ""
	}
	fields := function.Type.Params.List
	if len(fields) == 0 {
		return ""
	}
	last := fields[len(fields)-1]
	if !ast_is_namespace_type(last.Type) {
		return ""
	}
	if len(last.Names) == 0 {
		return ""
	}
	return last.Names[len(last.Names)-1].Name
}

// Reports whether expression is a namespace parameter's type: the bare `string`, the `Namespace`
// defined type (bare in this package), or a qualified `pkg.Namespace` (the type re-exported).
func ast_is_namespace_type(expression ast.Expr) (is_namespace bool) {
	if identifier, ok := expression.(*ast.Ident); ok {
		return identifier.Name == "string" || identifier.Name == "Namespace"
	}
	if selector, ok := expression.(*ast.SelectorExpr); ok {
		return selector.Sel.Name == "Namespace"
	}
	return false
}

// Seeds a reachability entry for a bare eager Always call — invariant.Always or
// Recorder_Always — keyed by its own message. Without this a
// never-reached Always could not be reported, since it never flows through a Dot_Product.
// Calls of any other kind (a Sometimes, a plain function) seed
// nothing: a bare element records nothing and is the caller's responsibility to consume.
// A duplicate Always message is a fatal collision.
func recorder_register_eager_always(
	recorder *Recorder, file_set *token.FileSet, call *ast.CallExpr, reg *Registration,
) {
	axis, is_axis := recorder_axis_of(file_set, call, false, reg)
	if !is_axis {
		return
	}
	if axis.Kind != ASSERTION_KIND_ALWAYS {
		return
	}
	_, loaded := recorder.Events.LoadOrStore(axis.Message, &Assertion_Metadata{
		Kind:      ASSERTION_KIND_ALWAYS,
		Message:   axis.Message,
		Condition: axis.Condition,
	})
	if loaded {
		reg.Collision = append(reg.Collision,
			recorder_position(file_set, call)+
				"  duplicate message: "+strconv.Quote(axis.Message))
	}
}

// Reports every bundle the analyzer recognised by name but could not resolve to a
// declaration, then exits. A recognised-but-unresolvable bundle would seed none of
// its elements while the runtime still enforces them, so its coverage obligations
// would vanish unnoticed; failing keeps coverage from being silently dropped — the
// analyzer descends a bundle or refuses it.
func recorder_check_unresolved(recorder *Recorder, unresolved []string) {
	if len(unresolved) == 0 {
		return
	}
	banner := "🚨 " + strconv.Itoa(len(unresolved)) + " unresolved bundles 🚨"
	fmt.Fprintln(recorder.Output, banner)
	for _, line := range unresolved {
		fmt.Fprintln(recorder.Output, line)
	}
	fmt.Fprintln(recorder.Output, banner)
	recorder.Exit(1)
}

// Reports every typed preset whose bound the evaluator could not resolve, then exits.
// A recognised-but-unevaluable bound leaves the grid unseeded while the runtime still enforces the
// range, so its coverage obligations would vanish unnoticed; failing keeps coverage from being
// silently dropped — the analyzer seeds a Range grid or refuses it.
func recorder_check_unresolved_bounds(recorder *Recorder, unresolved []string) {
	if len(unresolved) == 0 {
		return
	}
	banner := "🚨 " + strconv.Itoa(len(unresolved)) + " unresolved preset bounds 🚨"
	fmt.Fprintln(recorder.Output, banner)
	for _, line := range unresolved {
		fmt.Fprintln(recorder.Output, line)
	}
	fmt.Fprintln(recorder.Output, banner)
	recorder.Exit(1)
}

// Reports every assertion whose message is not a string literal, then exits. The runtime
// stamps whatever the message expression evaluates to, but the static side cannot key a
// non-literal — so its coverage would never be credited and its gap would vanish. Refuse
// it: a message is a compile-time literal or registration fails.
func recorder_check_non_literal_messages(recorder *Recorder, non_literal []string) {
	if len(non_literal) == 0 {
		return
	}
	banner := "🚨 " + strconv.Itoa(len(non_literal)) + " non-literal messages 🚨"
	fmt.Fprintln(recorder.Output, banner)
	for _, line := range non_literal {
		fmt.Fprintln(recorder.Output, line)
	}
	fmt.Fprintln(recorder.Output, banner)
	recorder.Exit(1)
}

// Reports every message collision, then exits. Two distinct assertions claiming one
// message — two Dot_Products sharing a prefix, a repeated axis message within one
// Dot_Product, or two Always sharing a message — would silently merge into one entry and
// mask a gap. A duplicate is fatal, never merged.
func recorder_check_duplicate_messages(recorder *Recorder, collisions []string) {
	if len(collisions) == 0 {
		return
	}
	banner := "🚨 " + strconv.Itoa(len(collisions)) + " duplicate messages 🚨"
	fmt.Fprintln(recorder.Output, banner)
	for _, line := range collisions {
		fmt.Fprintln(recorder.Output, line)
	}
	fmt.Fprintln(recorder.Output, banner)
	recorder.Exit(1)
}

// A malformed chain cannot be partially registered because every omitted axis or carve would
// weaken the demanded product; registration reports every structural refusal before exiting.
func recorder_check_invalid_chains(recorder *Recorder, invalid []string) {
	if len(invalid) == 0 {
		return
	}
	banner := "🚨 " + strconv.Itoa(len(invalid)) + " invalid Dot_Product chains 🚨"
	fmt.Fprintln(recorder.Output, banner)
	for _, line := range invalid {
		fmt.Fprintln(recorder.Output, line)
	}
	fmt.Fprintln(recorder.Output, banner)
	recorder.Exit(1)
}

// Checks the analyzed files for a *_Invariants / *_invariants bundle whose body contains a
// branching or looping statement (if, switch, type-switch, for, range, select) — banned,
// because it would make the axes the bundle self-emits depend on runtime values the static scan
// cannot read, silently under-registering coverage. A bundle body must be straight-line.
// Reports every violation under one banner and exits 1.
func recorder_check_bundle_control_flow(
	recorder *Recorder, file_set *token.FileSet, files []*ast.File,
) {
	var violations []string
	for _, file := range files {
		for _, declaration := range file.Decls {
			function, is_function := declaration.(*ast.FuncDecl)
			if !is_function {
				continue
			}
			if function.Body == nil {
				continue
			}
			if !ast_is_invariants_name(function.Name.Name) {
				continue
			}
			name := function.Name.Name
			ast.Inspect(function.Body, func(node ast.Node) (descend bool) {
				if !ast_is_control_flow(node) {
					return true
				}
				violations = append(violations, recorder_position(file_set, node)+
					"  banned: control flow inside bundle "+name)
				return true
			})
		}
	}
	if len(violations) == 0 {
		return
	}
	banner := "🚨 " + strconv.Itoa(len(violations)) + " bundle control-flow statements 🚨"
	fmt.Fprintln(recorder.Output, banner)
	for _, violation := range violations {
		fmt.Fprintln(recorder.Output, violation)
	}
	fmt.Fprintln(recorder.Output, banner)
	recorder.Exit(1)
}

// Fails registration when a bundle outside the framework's own package takes a
// primitive subject — a builtin, an unnamed slice/map, or any unnamed composite.
// Bundles for primitive types are the framework's presets; user code states a
// primitive inline or wraps it in a custom type. The Sugar_Package, which owns the
// presets, is exempt.
func recorder_check_primitive_bundles(
	recorder *Recorder, file_set *token.FileSet, files []*ast.File, index *Bundle_Index,
) {
	var offenders []string
	for _, file := range files {
		if index.Sugar_Package != "" {
			if recorder_file_package(file_set, file, index) == index.Sugar_Package {
				continue
			}
		}
		offenders = append(offenders,
			recorder_file_primitive_bundles(file_set, file)...)
	}
	if len(offenders) == 0 {
		return
	}
	banner := "🚨 " + strconv.Itoa(len(offenders)) + " primitive bundles 🚨"
	fmt.Fprintln(recorder.Output, banner)
	for _, line := range offenders {
		fmt.Fprintln(recorder.Output, line)
	}
	fmt.Fprintln(recorder.Output, banner)
	recorder.Exit(1)
}

// Collects every primitive-subject bundle declared in one file.
func recorder_file_primitive_bundles(
	file_set *token.FileSet, file *ast.File,
) (offenders []string) {
	for _, declaration := range file.Decls {
		function, is_function := declaration.(*ast.FuncDecl)
		if !is_function {
			continue
		}
		if !ast_is_invariants_name(function.Name.Name) {
			continue
		}
		if ast_namespace_parameter(function) == "" {
			continue
		}
		subject := recorder_bundle_subject(function)
		if subject == nil {
			continue
		}
		if !recorder_type_is_primitive(subject, recorder_bundle_type_parameters(function)) {
			continue
		}
		offenders = append(offenders, recorder_position(file_set, function)+
			"  primitive bundle subject: "+function.Name.Name)
	}
	return offenders
}

// Returns a bundle's subject type — its first parameter's type — or nil when the
// function declares no parameters.
func recorder_bundle_subject(function *ast.FuncDecl) (subject ast.Expr) {
	if function.Type.Params == nil {
		return nil
	}
	if len(function.Type.Params.List) == 0 {
		return nil
	}
	return function.Type.Params.List[0].Type
}

// Returns the bundle's own type-parameter names; a subject naming one of them is a
// generic custom subject, not a primitive.
func recorder_bundle_type_parameters(function *ast.FuncDecl) (names map[string]bool) {
	names = map[string]bool{}
	if function.Type.TypeParams == nil {
		return names
	}
	for _, field := range function.Type.TypeParams.List {
		for _, name := range field.Names {
			names[name.Name] = true
		}
	}
	return names
}

// Reports whether a bundle subject is a primitive: a predeclared builtin, or an
// unnamed composite (slice, map, channel, anonymous struct/interface/func). A bare
// defined-type name, an imported pkg.Type, or a type parameter is a custom subject.
func recorder_type_is_primitive(
	expression ast.Expr, type_parameters map[string]bool,
) (yes bool) {
	core := expression
	star, is_star := core.(*ast.StarExpr)
	if is_star {
		core = star.X
	}
	index, is_index := core.(*ast.IndexExpr)
	if is_index {
		core = index.X
	}
	index_list, is_index_list := core.(*ast.IndexListExpr)
	if is_index_list {
		core = index_list.X
	}
	identifier, is_identifier := core.(*ast.Ident)
	if is_identifier {
		if type_parameters[identifier.Name] {
			return false
		}
		return recorder_is_builtin_type_name(identifier.Name)
	}
	_, is_selector := core.(*ast.SelectorExpr)
	if is_selector {
		return false
	}
	return true
}

// Reports whether name is a Go predeclared type name.
func recorder_is_builtin_type_name(name string) (yes bool) {
	switch name {
	case "string", "bool", "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64", "uintptr",
		"byte", "rune", "float32", "float64", "complex64", "complex128",
		"error", "any", "comparable":
		return true
	default:
		return false
	}
}

// Returns the import path of file's package, derived from its absolute path against
// the index's module root and path. "" when no module was found.
func recorder_file_package(
	file_set *token.FileSet, file *ast.File, index *Bundle_Index,
) (import_path string) {
	if index.Module_Path == "" {
		return ""
	}
	absolute := file_set.Position(file.Pos()).Filename
	relative := strings.TrimPrefix(path.Dir(absolute), index.Module_Root)
	relative = strings.TrimPrefix(relative, "/")
	if relative == "" {
		return index.Module_Path
	}
	return path.Join(index.Module_Path, relative)
}

// Reports whether node is a branching or looping statement banned in a bundle body.
func ast_is_control_flow(node ast.Node) (is_control_flow bool) {
	switch node.(type) {
	case *ast.IfStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt,
		*ast.ForStmt, *ast.RangeStmt, *ast.SelectStmt:
		return true
	}
	return false
}

// Registration_Axis is the analyzer's complete description of one tuple coordinate. Keeping the
// source condition beside registration-owned identity lets analysis explain a missing coordinate
// without making runtime retain source text.
type Registration_Axis struct {
	// Ordinal keeps repeated chain messages distinct.
	Ordinal uint8
	// Tuple_Position is registration's authoritative packed-mask position.
	Tuple_Position uint8
	// Message is the element's own literal; the Dot_Product prefix forms the coverage key.
	Message string
	// Condition is the source text of the asserted condition.
	Condition string
	// Kind is whether the element is an Always or a Sometimes.
	Kind Assertion_Kind
	// Bucket_Count is how many buckets the axis adds to the tuple grid (Always=1, Sometimes=2).
	Bucket_Count int
}

// Registration accumulates the diagnostics a registration pass gathers before deciding
// whether to fail: bundles recognised by name but unresolvable, messages that are not
// string literals, and message collisions. Each is fatal on its own (see the
// recorder_check_* reporters). Seen_Prefix tracks Dot_Product messages so two grids
// cannot share one — the global-uniqueness guarantee for prefixes.
type Registration struct {
	// Unresolved holds bundles recognised by name but not resolvable to a declaration.
	Unresolved []string
	// Non_Literal holds messages that are not string literals, which cannot be keyed.
	Non_Literal []string
	// Collision holds Dot_Product messages that collided with an already-seen prefix.
	Collision []string
	// Unresolved_Bound holds typed preset links whose bounds the constant evaluator could not
	// resolve, so their grid could not be seeded.
	Unresolved_Bound []string
	// Invalid_Chain holds structural chain errors that would otherwise drop demanded coverage.
	Invalid_Chain []string
	// Seen_Prefix tracks Dot_Product messages so two grids cannot share one prefix.
	Seen_Prefix map[string]bool
}

// Returns the unquoted Go string value of the argument at index when it is a string
// literal, mirroring the message the runtime stamps. ok is false when the argument is
// absent, not a *ast.BasicLit, not a STRING, or unquotable — i.e. a variable or a
// concatenation the static side cannot resolve to a key.
func ast_string_literal(call *ast.CallExpr, index int) (value string, ok bool) {
	if len(call.Args) <= index {
		return "", false
	}
	literal, is_literal := call.Args[index].(*ast.BasicLit)
	if !is_literal {
		return "", false
	}
	if literal.Kind != token.STRING {
		return "", false
	}
	unquoted, unquote_error := strconv.Unquote(literal.Value)
	if unquote_error != nil {
		return "", false
	}
	return unquoted, true
}

// Renders one unresolvable bundle as
// "<site>  unresolved bundle: <name> cannot be analyzed".
func recorder_unresolved_line(file_set *token.FileSet, call *ast.CallExpr) (line string) {
	return recorder_position(file_set, call) + "  unresolved bundle: " +
		ast_callee_name(call) + " cannot be analyzed"
}

// Resolves a bundle call to its declaration using the calling file's imports: a
// bare call hits Same_Set (same-package); a qualified pkg.Foo_Invariants resolves
// pkg to an import path and, if it is inside the module, loads that package.
// found is false for an unresolvable bundle (cross-module, missing go.mod, an
// unknown qualifier, or an absent declaration).
func bundle_index_lookup(
	index *Bundle_Index, imports map[string]string, bundle *ast.CallExpr,
) (function Indexed_Function, found bool) {
	qualifier, name := ast_bundle_qualifier(bundle)
	if qualifier == "" {
		same_package, present := index.Same_Set[name]
		return same_package, present
	}
	import_path, imported := imports[qualifier]
	if !imported {
		return Indexed_Function{}, false
	}
	cross_package, present := bundle_index_load(index, import_path)[name]
	return cross_package, present
}

// Resolves import_path to its absolute source directory, treating every import path as relative to
// the module path declared in go.mod: import_path must be the module path itself or sit under it,
// and the remainder names the directory under Module_Root. resolved is false for a path outside
// this module (an external dependency). This is pure string-prefix matching — the module path need
// not be a URL, so `module local` resolves `local/shared/foo` to <root>/shared/foo all the same.
func bundle_index_module_root(
	index *Bundle_Index, import_path string,
) (directory string, resolved bool) {
	if index.Module_Path == "" {
		return "", false
	}
	if import_path == index.Module_Path {
		return index.Module_Root, true
	}
	if strings.HasPrefix(import_path, index.Module_Path+"/") {
		return index.Module_Root + strings.TrimPrefix(import_path, index.Module_Path), true
	}
	return "", false
}

// Lazily parses the package at import_path and returns its functions by name,
// caching the result. The cache is seeded before parsing so a cyclic import
// resolves to the empty map rather than looping. Returns an empty map for a path
// outside this module (see bundle_index_module_root).
func bundle_index_load(
	index *Bundle_Index, import_path string,
) (functions map[string]Indexed_Function) {
	if cached, done := index.Loaded[import_path]; done {
		return cached
	}
	functions = map[string]Indexed_Function{}
	index.Loaded[import_path] = functions
	directory, resolved := bundle_index_module_root(index, import_path)
	if !resolved {
		return functions
	}
	files := recorder_parse_directory(index.File_System, index.File_Set, directory)
	is_sugar := import_path == index.Sugar_Package
	for name, function := range ast_index_functions(files) {
		function.Is_Sugar = is_sugar
		functions[name] = function
	}
	return functions
}

// Returns the called function's name: the Ident name for a bare call or the Sel
// name for a qualified call; "" otherwise.
func ast_callee_name(call *ast.CallExpr) (name string) {
	if identifier, is_identifier := call.Fun.(*ast.Ident); is_identifier {
		return identifier.Name
	}
	if selector, is_selector := call.Fun.(*ast.SelectorExpr); is_selector {
		return selector.Sel.Name
	}
	return ""
}

// Splits a bundle call into its package qualifier and name: ("", name) for a bare
// Foo_Invariants(), (pkg, name) for a qualified pkg.Foo_Invariants(). Both "" for
// any other call shape.
func ast_bundle_qualifier(call *ast.CallExpr) (qualifier string, name string) {
	if identifier, is_identifier := call.Fun.(*ast.Ident); is_identifier {
		return "", identifier.Name
	}
	selector, is_selector := call.Fun.(*ast.SelectorExpr)
	if !is_selector {
		return "", ""
	}
	package_identifier, is_package := selector.X.(*ast.Ident)
	if !is_package {
		return "", ""
	}
	return package_identifier.Name, selector.Sel.Name
}

// Maps each of a file's imports to its local name: the explicit alias when
// present, else the import path's last segment. The latter is a heuristic —
// correct when the package's clause name matches its directory basename, which
// holds for the common case but not for, e.g., a package "invariant" in dir "v3".
func ast_file_imports(file *ast.File) (imports map[string]string) {
	imports = map[string]string{}
	for _, specification := range file.Imports {
		import_path := strings.Trim(specification.Path.Value, `"`)
		local := path.Base(import_path)
		if specification.Name != nil {
			local = specification.Name.Name
		}
		imports[local] = import_path
	}
	return imports
}

// Maps an axis constructor selector to its assertion kind and the index of its
// condition-bearing argument. The bare sugar forms (Always / Sometimes) carry the condition
// first; the explicit Recorder_* forms lead with the recorder, so the condition rides the
// second argument. is_axis is false for any other selector.
func ast_axis_signature(
	selector string,
) (kind Assertion_Kind, condition_index int, is_axis bool) {
	switch selector {
	case "Always":
		return ASSERTION_KIND_ALWAYS, 0, true
	case "Sometimes":
		return ASSERTION_KIND_SOMETIMES, 0, true
	case "Recorder_Always":
		return ASSERTION_KIND_ALWAYS, 1, true
	case "Recorder_Sometimes":
		return ASSERTION_KIND_SOMETIMES, 1, true
	}
	return 0, 0, false
}

// Returns the axis for an Always/Sometimes constructor call, in either the bare sugar form or
// the explicit Recorder_* form; is_axis is false for any other call (Impossible, a bundle, a
// non-invariant call).
func recorder_axis_of(
	file_set *token.FileSet, call *ast.CallExpr, allow_unqualified bool, reg *Registration,
) (axis Registration_Axis, is_axis bool) {
	selector := ast_selector(call, allow_unqualified)
	if kind, condition_index, ok := ast_axis_signature(selector); ok {
		condition := ast_condition_text(file_set, call, condition_index)
		bucket_count := 2
		if kind == ASSERTION_KIND_ALWAYS {
			bucket_count = 1
		}
		// The message is the argument past the condition; the runtime stamps the same
		// literal. A non-literal cannot be keyed, so it is reported and fails registration.
		message, literal := ast_string_literal(call, condition_index+1)
		if !literal {
			reg.Non_Literal = append(reg.Non_Literal,
				recorder_position(file_set, call)+
					"  "+selector+" message is not a string literal")
		}
		return Registration_Axis{
			Message:      message,
			Condition:    condition,
			Kind:         kind,
			Bucket_Count: bucket_count,
		}, true
	}
	return Registration_Axis{}, false
}

// Returns the X in a literal `invariant.X(...)` selector call, or "" otherwise.
func ast_invariant_selector(call *ast.CallExpr) (name string) {
	return ast_selector(call, false)
}

// Returns the invariant primitive a call names: the X of a qualified
// `invariant.X(...)`, or — when allow_unqualified (the call is inside a bundle in
// the sugar package) — a bare `X(...)` whose X is a known primitive. "" otherwise.
func ast_selector(call *ast.CallExpr, allow_unqualified bool) (name string) {
	if selector, is_selector := call.Fun.(*ast.SelectorExpr); is_selector {
		package_identifier, is_identifier := selector.X.(*ast.Ident)
		if !is_identifier {
			return ""
		}
		if package_identifier.Name != "invariant" {
			return ""
		}
		return selector.Sel.Name
	}
	if !allow_unqualified {
		return ""
	}
	identifier, is_identifier := call.Fun.(*ast.Ident)
	if !is_identifier {
		return ""
	}
	if !ast_is_invariant_primitive(identifier.Name) {
		return ""
	}
	return identifier.Name
}

// Reports whether name is an invariant element/reference primitive, the set the
// sugar tier exposes as bare functions and that may appear unqualified inside a
// sugar-package bundle.
func ast_is_invariant_primitive(name string) (is_primitive bool) {
	switch name {
	case "Always", "Sometimes", "Range", "Enum",
		"Recorder_Always", "Recorder_Sometimes", "Recorder_Range", "Recorder_Enum":
		return true
	}
	return false
}

// Reports whether name is a bundle function name: a *_Invariants (exported) or
// *_invariants (unexported) suffix. Both casings exist because the bundle's name
// follows its type's casing — the free-function-over-a-type rule the linter
// enforces — so the analyzer must accept either or silently drop an unexported
// type's bundle coverage.
func ast_is_invariants_name(name string) (is_bundle bool) {
	return strings.HasSuffix(name, "_Invariants") || strings.HasSuffix(name, "_invariants")
}

// Returns "file:line" for the node's start position.
func recorder_position(file_set *token.FileSet, node ast.Node) (site string) {
	position := file_set.Position(node.Pos())
	return position.Filename + ":" + strconv.Itoa(position.Line)
}

// Returns the source text of the constructor's condition argument — at condition_index,
// past any leading recorder — for the never-fired report; "" when the call lacks it.
func ast_condition_text(
	file_set *token.FileSet, call *ast.CallExpr, condition_index int,
) (text string) {
	if len(call.Args) <= condition_index {
		return ""
	}
	return ast_expression_text(file_set, call.Args[condition_index])
}

// Returns the source text of expression, or "" when it can't be printed.
func ast_expression_text(file_set *token.FileSet, expression ast.Expr) (text string) {
	var buffer bytes.Buffer
	if printer.Fprint(&buffer, file_set, expression) != nil {
		return ""
	}
	return buffer.String()
}

// A Coverage_Gap is one seeded assertion that the run failed to exercise, paired
// with the reason it counts as a gap (which branch or combination went unseen).
type Coverage_Gap struct {
	// Metadata is the seeded assertion that went unexercised.
	Metadata *Assertion_Metadata
	// Reason names why it counts as a gap: which branch or combination went unseen.
	Reason string
}

// Recorder_Analyze_Assertion_Frequency reports every pre-registered assertion
// whose true branch never fired and every Sometimes whose false branch never
// fired — naming each by its message and condition source — then calls Exit(1) when
// any gap exists. It is a no-op in a benchmark or a fuzz worker subprocess; a plain test
// run and the fuzz coordinator both analyze.
func Recorder_Analyze_Assertion_Frequency(recorder *Recorder) {
	if !recorder.Is_Test {
		return
	}
	if recorder.Is_Benchmark {
		return
	}
	if recorder.Is_Fuzz_Worker {
		return
	}
	gaps := recorder_collect_gaps(recorder)
	if len(gaps) == 0 {
		return
	}
	recorder_report_gaps(recorder, gaps)
	recorder.Exit(1)
}

// Walks the tracker and returns every coverage gap across all seeded assertions.
func recorder_collect_gaps(recorder *Recorder) (gaps []Coverage_Gap) {
	recorder.Events.Range(func(key, value any) (continue_iteration bool) {
		metadata := value.(*Assertion_Metadata)
		gaps = append(gaps, assertion_metadata_gaps(metadata)...)
		return true
	})
	return gaps
}

// Returns the coverage gaps one assertion exhibits. A Sometimes contributes a gap
// per branch it never observed (true and/or false); an Always that never
// fired is a single gap; a fully exercised assertion contributes none.
func assertion_metadata_gaps(metadata *Assertion_Metadata) (gaps []Coverage_Gap) {
	if metadata.Kind == ASSERTION_KIND_SOMETIMES {
		if metadata.Frequency.Load() == 0 {
			gaps = append(gaps, Coverage_Gap{
				Metadata: metadata, Reason: "true branch never observed",
			})
		}
		if metadata.False_Frequency.Load() == 0 {
			gaps = append(gaps, Coverage_Gap{
				Metadata: metadata, Reason: "false branch never observed",
			})
		}
		return gaps
	}
	if metadata.Frequency.Load() != 0 {
		return gaps
	}
	return append(gaps, Coverage_Gap{Metadata: metadata, Reason: "never reached"})
}

// Prints the gaps to recorder.Output in two sections — branch, reachability —
// each sorted by site. A banner carrying the gap count brackets the report so
// the verdict survives a top-down or bottom-up skim.
func recorder_report_gaps(recorder *Recorder, gaps []Coverage_Gap) {
	banner := "🚨 " + strconv.Itoa(len(gaps)) + " coverage gaps 🚨"
	fmt.Fprintln(recorder.Output, banner)
	recorder_report_section(
		recorder.Output, "Branch gaps", gaps, ASSERTION_KIND_SOMETIMES)
	recorder_report_section(
		recorder.Output, "Reachability gaps", gaps, ASSERTION_KIND_ALWAYS)
	fmt.Fprintln(recorder.Output, banner)
}

// Prints, under a markdown heading, the gaps whose assertion is of the given
// kind, sorted by message. Emits nothing when no gap matches, so empty sections
// stay silent.
func recorder_report_section(
	output io.Writer, title string, gaps []Coverage_Gap, kind Assertion_Kind,
) {
	selected := make([]Coverage_Gap, 0, len(gaps))
	for _, gap := range gaps {
		if gap.Metadata.Kind == kind {
			selected = append(selected, gap)
		}
	}
	if len(selected) == 0 {
		return
	}
	// Two gaps can share a message — a Sometimes missing both branches — so the Reason breaks
	// the tie. Without it the order rides on the tracker's unordered iteration and the report
	// is non-deterministic.
	sort.Slice(selected, func(i, j int) (less bool) {
		if selected[i].Metadata.Message != selected[j].Metadata.Message {
			return selected[i].Metadata.Message < selected[j].Metadata.Message
		}
		return selected[i].Reason < selected[j].Reason
	})
	fmt.Fprintln(output)
	fmt.Fprintln(output, "# "+title)
	for _, gap := range selected {
		fmt.Fprintln(output, coverage_gap_line(gap))
	}
}

// Decodes a bucket index for an axis of the given kind into the event it stands for: a
// Sometimes 0/1 into false/true, an Always into held (its one bucket means the condition held,
// the only outcome an Always records).
func assertion_kind_bucket_text(kind Assertion_Kind, index int) (text string) {
	if kind == ASSERTION_KIND_ALWAYS {
		return "held"
	}
	if index == 1 {
		return "true"
	}
	return "false"
}

// Renders one branch or reachability gap as a report line, naming its kind,
// reason, and condition source.
func coverage_gap_line(gap Coverage_Gap) (line string) {
	metadata := gap.Metadata
	return message_display(metadata.Message) + "  " + assertion_kind_name(metadata.Kind) +
		" — " + gap.Reason + ": " + strconv.Quote(metadata.Condition)
}

// Renders a coverage key for the report: the element separator (the NUL joining a
// Dot_Product prefix to an axis message) shows as " · " so "signup.username␀empty" reads
// as "signup.username · empty". A bare message (an Always, or a grid prefix) is unchanged.
func message_display(message string) (display string) {
	return strings.ReplaceAll(message, ELEMENT_MESSAGE_SEPARATOR, " · ")
}

// Returns the report label for a kind: the same word the static pass keys on.
func assertion_kind_name(kind Assertion_Kind) (name string) {
	if kind == ASSERTION_KIND_SOMETIMES {
		return "Sometimes"
	}
	return "Always"
}

// Recorder_Assertion_Summary renders the clean-run banner naming how many
// properties the run tested: an Always is one individual property, a Sometimes is
// two (its true and its false branch are separate obligations); the Always family
// is the panic-able subset whose violation fails fatally at runtime.
func Recorder_Assertion_Summary(recorder *Recorder) (summary string) {
	individual := 0
	combinations := 0
	panic_able := 0
	recorder.Events.Range(func(key, value any) (continue_iteration bool) {
		metadata := value.(*Assertion_Metadata)
		switch metadata.Kind {
		case ASSERTION_KIND_ALWAYS:
			individual++
			panic_able++
		default:
			// A Sometimes must witness both its true and its false branch, so it
			// counts twice.
			individual += 2
		}
		return true
	})
	if recorder.Package_Label != "" {
		return fmt.Sprintf(
			"✓ %s: tested %d properties (%d individual + %d combinations, "+
				"of which %d are panic-able)",
			recorder.Package_Label,
			individual+combinations, individual, combinations, panic_able,
		)
	}
	return fmt.Sprintf(
		"✓ tested %d properties (%d individual + %d combinations, "+
			"of which %d are panic-able)",
		individual+combinations, individual, combinations, panic_able,
	)
}

// Recorder_Run_Test_Main is the canonical TestMain body: it registers the
// analyzed directories, runs the suite, reports any unexercised assertions, then
// exits with the suite's code. On a clean run — the suite passed and the analysis
// found no gaps — it prints the tested-property summary to Tty (falling back to
// Output) so the line shows even without `go test -v`.
func Recorder_Run_Test_Main(recorder *Recorder, m *testing.M, directories ...string) {
	Recorder_Register_Packages_For_Analysis(recorder, directories...)
	code := m.Run()
	// A fuzz coordinator merges the workers' persisted coverage before analyzing — it never ran
	// the fuzzed body itself, so without this its grid would be empty (see Coverage / Modes).
	if recorder.Merge_Fuzz_Coverage != nil {
		recorder.Merge_Fuzz_Coverage()
	}
	Recorder_Analyze_Assertion_Frequency(recorder)
	if code != 0 {
		recorder.Exit(code)
		return
	}
	summary_output := recorder.Tty
	if summary_output == nil {
		summary_output = recorder.Output
	}
	fmt.Fprintln(summary_output, Recorder_Assertion_Summary(recorder))
	recorder.Exit(code)
}
