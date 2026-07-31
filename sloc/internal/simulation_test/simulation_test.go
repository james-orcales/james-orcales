// Package simulation_test is sloc's blackbox witness: a single fuzz target drives
// internal.Main end to end — command-line parsing, the tree walk, per-line
// classification, and both renderers — so every production invariant is exercised
// through the one public entry point, never by reaching into a helper. Under a plain
// `go test` the seed corpus is replayed and the invariant recorder judges coverage;
// under -fuzz the same driver explores.
package simulation_test

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"strings"
	"testing"
	"testing/fstest"

	invariant "local/james-orcales/shared/invariant/default"
	sloc "local/james-orcales/sloc/internal"
)

// TestMain wires the coverage recorder over the internal tree ("../**"): every Always
// reached must hold and every Sometimes both fire, or the run fails.
func TestMain(m *testing.M) {
	invariant.Run_Test_Main(m, "../**")
}

// Fuzz_Main is the sole witness. It decodes the fuzz bytes into a counting scenario and
// drives internal.Main; the seed corpus is a battery of honest end-to-end runs, each
// reaching a real boundary of the pipeline.
func Fuzz_Main(f *testing.F) {
	for _, seed := range seed_corpus() {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		drive(decode_scenario(data))
	})
}

// A scenario is one whole run: the tree to count, the command line to count it with,
// and the host failures to inject. Every field is decoded from the fuzz bytes, so any
// input names a complete run.
type scenario struct {
	// Recipe selects which synthetic tree is built.
	Recipe uint8
	// Show_Files lists each file under its language.
	Show_Files bool
	// No_Ignore counts the paths the predicate would drop.
	No_Ignore bool
	// Hidden counts dot-prefixed entries.
	Hidden bool
	// Json renders the machine-readable form instead of the table.
	Json bool
	// Ignore_Some installs a predicate that drops paths holding "ignored".
	Ignore_Some bool
	// Explicit makes the root stat as a file rather than a directory, taking the
	// explicitly-named-file path instead of the walk.
	Explicit bool
	// Stat_Error injects a failure from the directory check.
	Stat_Error bool
	// Read_Error injects a failure from the file read.
	Read_Error bool
	// Bad_Argument appends a flag the parser rejects, so the usage exit is reached.
	Bad_Argument bool
	// Root_Bytes is the width of a synthetic root argument, so the root bound is
	// witnessed without needing a tree that deep.
	Root_Bytes int
	// Argument_Pad is how many inert trailing arguments to append.
	Argument_Pad int
	// Line_Bytes is the width of the generated wide line.
	Line_Bytes int
	// Line_Count is how many lines the generated files carry.
	Line_Count int
	// File_Count is how many files the many-files recipe builds.
	File_Count int
	// Concurrency is the worker bound handed to the library.
	Concurrency int
}

// A disk_fixture keeps modeled classifications beside the paths they describe so a
// simulation cannot substitute a result without also manufacturing that file.
type disk_fixture struct {
	// Disk is the tree Main walks and reads.
	Disk fstest.MapFS
	// Roots supplies distinct trees only when a multi-root bound requires them.
	Roots map[sloc.Root]fstest.MapFS
	// Classifications replaces only derivations the fixture explicitly establishes.
	Classifications map[sloc.Classified_Path]sloc.File_Partition
}

// A cursor reads the fuzz bytes; every read past the end yields zero, so any input
// decodes to a complete scenario and the fuzzer never wastes a mutation on length.
type cursor struct {
	// Data is the fuzz input being read.
	Data []byte
	// Index is how far into it the read has reached.
	Index int
}

// Returns the next byte, or zero past the end.
func take_byte(read *cursor) (value uint8) {
	if read.Index >= len(read.Data) {
		return 0
	}
	value = read.Data[read.Index]
	read.Index++
	return value
}

// Returns the next big-endian uint32, or zero past the end.
func take_uint32(read *cursor) (value uint32) {
	for index := 0; index < 4; index++ {
		value = value<<8 | uint32(take_byte(read))
	}
	return value
}

// Returns the next byte's low bit as a flag.
func take_flag(read *cursor) (value bool) {
	return take_byte(read)&1 == 1
}

// Returns the next value folded into [0, bound].
func take_bounded(read *cursor, bound int) (value int) {
	if bound <= 0 {
		return 0
	}
	return int(take_uint32(read)) % (bound + 1)
}

// Decodes the fuzz bytes into a scenario. Counts are folded into sloc's own exported
// ceilings, so the driver cannot drift from the bounds it is meant to exercise.
func decode_scenario(data []byte) (one scenario) {
	read := &cursor{Data: data, Index: 0}
	one.Recipe = take_byte(read)
	one.Show_Files = take_flag(read)
	one.No_Ignore = take_flag(read)
	one.Hidden = take_flag(read)
	one.Json = take_flag(read)
	one.Ignore_Some = take_flag(read)
	one.Explicit = take_flag(read)
	one.Stat_Error = take_flag(read)
	one.Read_Error = take_flag(read)
	one.Bad_Argument = take_flag(read)
	one.Root_Bytes = take_bounded(read, sloc.ROOT_BYTES_MAX)
	one.Argument_Pad = take_bounded(read, sloc.ARGUMENTS_COUNT_MAX)
	one.Line_Bytes = take_bounded(read, sloc.LINE_BYTES_MAX)
	one.Line_Count = take_bounded(read, LINE_COUNT_CHOICE_MAX)
	one.File_Count = take_bounded(read, 512)
	one.Concurrency = concurrency_of(take_bounded(read, CONCURRENCY_CHOICE_MAX))
	return one
}

// LINE_COUNT_CHOICE_MAX is the widest generated file, in lines. It is large enough that
// a tally prints with two thousands separators, which is the width between a column's
// header label and its own bound.
const LINE_COUNT_CHOICE_MAX = 2048

// CONCURRENCY_CHOICE_MAX is the highest worker-bound choice the decoder names.
const CONCURRENCY_CHOICE_MAX = 5

// Returns the worker bound a choice names. The library states its worker bound with
// the unbounded integer preset, whose boundaries are the whole int64 range, so the
// choices name those explicitly rather than leaving them to a 32-bit decode.
func concurrency_of(choice int) (workers int) {
	switch choice {
	case 0:
		return math.MinInt64
	case 1:
		return -1
	case 2:
		return 0
	case 3:
		return 1
	case 4:
		return math.MaxInt64
	}
	return 2
}

// Drives one whole run of internal.Main against a synthetic tree.
func drive(one scenario) {
	fixture := build_fixture(one)
	drive_main(&drive_main_input{
		Scenario: one,
		Fixture:  fixture,
		Output:   io.Discard,
	})
}

type drive_main_input struct {
	Scenario scenario
	Fixture  disk_fixture
	Output   io.Writer
}

// The input keeps alternate model checks on the same Main host as the fuzz driver, so
// their comparisons cannot bypass a production stage.
func drive_main(input *drive_main_input) {
	one := input.Scenario
	fixture := input.Fixture
	disk := fixture.Disk
	stderr := strings.Builder{}
	sloc.Main(sloc.Main_Input{
		Arguments:    build_arguments(one),
		Output:       input.Output,
		Error_Output: &stderr,
		File_System: func(root sloc.Root) (file_system fs.FS) {
			return broken_disk{MapFS: fixture_disk_for(fixture, root)}
		},
		Path_Information: func(
			name sloc.File_Path,
		) (information fs.FileInfo, err error) {
			if one.Stat_Error {
				return nil, errors.New("stat failed")
			}
			return simulation_information(!one.Explicit), nil
		},
		File: func(name sloc.File_Path) (file fs.File, err error) {
			if one.Read_Error {
				return &read_error_file{File: simulation_file([]byte{'a'})}, nil
			}
			entry, found := disk[string(name)]
			if !found {
				return simulation_file(nil), nil
			}
			return simulation_file(entry.Data), nil
		},
		Command: func(
			name string, arguments []string,
		) (output []byte, err error) {
			if !one.Ignore_Some {
				return nil, errors.New("command unavailable")
			}
			return simulation_git_output(disk), nil
		},
		Classifier:  classifier_for(fixture.Classifications),
		Concurrency: sloc.Concurrency(one.Concurrency),
	})
}

// Root selection is explicit data so a multi-root simulation remains deterministic
// without depending on how many times Open has already been called.
func fixture_disk_for(fixture disk_fixture, root sloc.Root) (disk fstest.MapFS) {
	disk, found := fixture.Roots[root]
	if found {
		return disk
	}
	return fixture.Disk
}

// Only fixtures built to witness large derived tallies use a model; keeping that
// decision here prevents ordinary fuzz recipes from silently replacing the scanner.
func build_fixture(one scenario) (fixture disk_fixture) {
	recipe := uint8(int(one.Recipe) % RECIPE_COUNT)
	kind, wide := wide_kind_for_recipe(recipe)
	if wide {
		if recipe == 15 {
			if one.Json {
				return disk_wide_source_table()
			}
		}
		return disk_wide_table(kind)
	}
	if recipe == 12 {
		return disk_oversized()
	}
	if recipe == 26 {
		return disk_all_dropped()
	}
	if recipe >= 27 {
		if recipe <= 29 {
			return disk_dropped(dropped_fixture_count(recipe, one.File_Count))
		}
	}
	if recipe == 31 {
		if line_bound_past_max(one) {
			return disk_past_lines_max()
		}
		return disk_line_bound(one)
	}
	if recipe == 33 {
		return disk_test_partitions()
	}
	return disk_fixture{Disk: build_disk(one), Classifications: nil}
}

// Nil selects the byte classifier for ordinary recipes; a bounded model is total for
// nonempty inputs so no expensive derivation can enter by accident.
func classifier_for(
	classifications map[sloc.Classified_Path]sloc.File_Partition,
) (classifier sloc.File_Classifier) {
	if classifications == nil {
		return sloc.File_Classifier{Kind: sloc.FILE_CLASSIFIER_KIND_BYTES}
	}
	return sloc.File_Classifier{
		Kind:            sloc.FILE_CLASSIFIER_KIND_MODEL,
		Classifications: sloc.File_Classifications(classifications),
	}
}

// Returns stable information for the directory decision without host filesystem state.
func simulation_information(directory bool) (information fs.FileInfo) {
	mode := fs.FileMode(0)
	if directory {
		mode = fs.ModeDir
	}
	disk := fstest.MapFS{"entry": &fstest.MapFile{Mode: mode}}
	information, information_err := fs.Stat(disk, "entry")
	if information_err != nil {
		panic(information_err)
	}
	return information
}

// Returns a new synthetic file because Main closes each injected file handle.
func simulation_file(content []byte) (file fs.File) {
	disk := fstest.MapFS{"file": &fstest.MapFile{Data: content}}
	opened, open_err := disk.Open("file")
	if open_err != nil {
		panic(open_err)
	}
	return opened
}

// Returns the Git listing of files whose paths do not contain the simulated ignore marker.
func simulation_git_output(disk fstest.MapFS) (output []byte) {
	listing := strings.Builder{}
	walk_err := fs.WalkDir(disk, ".", func(
		file_path string, entry fs.DirEntry, step_err error,
	) (err error) {
		if step_err != nil {
			return step_err
		}
		if entry.IsDir() {
			return nil
		}
		if strings.Contains(file_path, "ignored") {
			return nil
		}
		listing.WriteString(file_path)
		listing.WriteByte(0)
		return nil
	})
	if walk_err != nil {
		panic(walk_err)
	}
	return []byte(listing.String())
}

// Read_error_file preserves valid file status while it injects the read failure.
type read_error_file struct {
	fs.File
}

// Read injects a deterministic file-read failure after Main reads valid file status.
func (file *read_error_file) Read(buffer []byte) (read_count int, err error) {
	return 0, errors.New("read failed")
}

// Builds the command line: the program name, the flags the scenario sets, the root to
// count, and any inert padding.
func build_arguments(one scenario) (arguments sloc.Arguments) {
	arguments = sloc.Arguments{"sloc"}
	// The parser takes single-dash flags and rejects the double-dash form outright.
	if one.Show_Files {
		arguments = append(arguments, "-files")
	}
	if one.No_Ignore {
		arguments = append(arguments, "-no-ignore")
	}
	if one.Hidden {
		arguments = append(arguments, "-hidden")
	}
	if one.Json {
		arguments = append(arguments, "-json")
	}
	if one.Bad_Argument {
		arguments = append(arguments, "-nosuchflag")
	}
	if line_bound_past_max(one) {
		return append(arguments, string(LINE_BOUND_ROOT_GO), string(LINE_BOUND_ROOT_PYTHON))
	}
	// A synthetic root of a chosen width witnesses the root bound without a tree that
	// deep; the walk itself always starts from the file system's own ".". A root wide
	// enough to carry an extension gets one, so the explicitly-named-file route
	// reaches a path the language table recognizes.
	if one.Root_Bytes > 0 {
		root := strings.Repeat("r", one.Root_Bytes)
		// A modest root gets an extension so the explicitly-named-file route reaches
		// a recognized path; the widest is left bare so the same route also reaches
		// the by-name lookup with a base name at its own bound.
		if one.Root_Bytes > 3 {
			if one.Root_Bytes < 64 {
				root = strings.Repeat("r", one.Root_Bytes-3) + ".go"
			}
		}
		arguments = append(arguments, root)
	}
	for index := 0; index < one.Argument_Pad; index++ {
		if len(arguments) >= sloc.ARGUMENTS_COUNT_MAX {
			return arguments
		}
		arguments = append(arguments, ".")
	}
	return arguments
}

// RECIPE_COUNT is how many synthetic trees the driver knows how to build; the decoded
// recipe byte is folded into it, so every fuzz input names a real tree.
const RECIPE_COUNT = 34

// DEEP_COMMENT_OPENERS is how many nested block comments disk_deep_comment opens. It
// runs past the depth bound so the carried depth saturates rather than merely rising.
const DEEP_COMMENT_OPENERS = 300

// DROPPED_LINE_MARGIN is how far past the scan window disk_dropped's lines run. It is
// small on purpose: the tally counts lines, so width past the window is wasted bytes.
const DROPPED_LINE_MARGIN = 1

// DROPPED_SATURATING_LINES is how many lines read short it takes to put the tally on
// its own bound. The tally saturates, so overshooting lands on the bound exactly.
const DROPPED_SATURATING_LINES = sloc.DROPPED_COUNT_MAX + 1

// BROKEN_PREFIX names the files broken_disk refuses to open or read. It is a name rather
// than a count so a tree's breakage survives the walk's ordering.
const BROKEN_PREFIX = "broken"

// Builds the synthetic tree the scenario names.
func build_disk(one scenario) (disk fstest.MapFS) {
	switch int(one.Recipe) % RECIPE_COUNT {
	case 0:
		return fstest.MapFS{}
	case 1:
		return disk_simple()
	case 2:
		return disk_languages()
	case 3:
		return disk_wide_line(one)
	case 4:
		return disk_many_lines(one)
	case 5:
		return disk_byte_extremes()
	case 6:
		return disk_many_files(one)
	case 7:
		return disk_binary()
	case 8:
		return disk_hidden_and_ignored()
	case 9:
		return disk_scanner_corners()
	case 10:
		return disk_wide_table(WIDE_KIND_SPREAD).Disk
	case 11:
		return disk_unrecognized()
	case 12:
		return disk_oversized().Disk
	case 13:
		return disk_wide_table(WIDE_KIND_COMMENT).Disk
	case 14:
		return disk_wide_table(WIDE_KIND_BLANK).Disk
	case 15:
		return disk_wide_table(WIDE_KIND_SINGLE).Disk
	case 16:
		return disk_wide_table(WIDE_KIND_TESTS).Disk
	case 17:
		return disk_wide_table(WIDE_KIND_CODE).Disk
	}
	return build_disk_skipped(one)
}

// The trees whose point is what a run leaves out rather than what it counts. They are
// split from build_disk only because one switch over every recipe runs past the
// function-length bound.
func build_disk_skipped(one scenario) (disk fstest.MapFS) {
	switch int(one.Recipe) % RECIPE_COUNT {
	case 18:
		return disk_broken(1)
	case 19:
		return disk_broken(2)
	case 20:
		return disk_broken(sloc.FILES_COUNT_MAX)
	case 21:
		return disk_binary_many(2)
	case 22:
		return disk_binary_many(sloc.FILES_COUNT_MAX)
	case 23:
		return disk_oversized_many(oversized_fixture_count(one.File_Count))
	case 24:
		return disk_overflow(1)
	case 25:
		return disk_overflow(2)
	case 26:
		return disk_all_dropped().Disk
	case 27:
		return disk_dropped(1).Disk
	case 28:
		return disk_dropped(2).Disk
	case 29:
		return disk_dropped(dropped_fixture_count(29, one.File_Count)).Disk
	case 30:
		return disk_deep_comment()
	case 31:
		return disk_line_bound(one).Disk
	case 33:
		return disk_test_partitions().Disk
	}
	return disk_pair()
}

// LINE_BOUND_KIND_BLANK fills one language's tally with blank lines, which cost one
// byte each and so are the cheapest way to reach an eight-figure count.
const LINE_BOUND_KIND_BLANK = 0

// LINE_BOUND_KIND_CODE fills it with code lines.
const LINE_BOUND_KIND_CODE = 1

// LINE_BOUND_KIND_COMMENT fills it with comment lines.
const LINE_BOUND_KIND_COMMENT = 2

// LINE_BOUND_FILE_WIDTH_COUNT is the first five-digit file tally, whose separators
// widen the Files column without spending the whole file bound.
const LINE_BOUND_FILE_WIDTH_COUNT = 10000

// LINE_BOUND_ROOT_GO selects the Go half of the merged past-lines maximum witness.
const LINE_BOUND_ROOT_GO sloc.Root = "go-root"

// LINE_BOUND_ROOT_PYTHON selects the Python half of the merged maximum witness.
const LINE_BOUND_ROOT_PYTHON sloc.Root = "python-root"

// Returns the line one kind repeats and the extension that reads it that way. Python
// is what makes a one-character line a comment rather than code.
func line_bound_line(kind int) (text string, suffix string) {
	switch kind {
	case LINE_BOUND_KIND_CODE:
		return "a\n", ".go"
	case LINE_BOUND_KIND_COMMENT:
		return "#\n", ".py"
	}
	return "\n", ".rs"
}

// Three languages each summing to exactly the line bound, one filling its tally with
// code, one with comments, one with blanks. A language's partitions sum to its own
// total, so no single language can put more than one of them on the bound; three are
// what it takes, and counting them in one run is also what puts every column of the
// table at its widest at the same time.
func disk_line_bound(one scenario) (fixture disk_fixture) {
	fixture = disk_fixture{
		Disk:            fstest.MapFS{},
		Classifications: map[sloc.Classified_Path]sloc.File_Partition{},
	}
	fill_line_bound(&fixture, LINE_BOUND_KIND_CODE)
	fill_line_bound(&fixture, LINE_BOUND_KIND_COMMENT)
	fill_line_bound(&fixture, LINE_BOUND_KIND_BLANK)
	// A zero-code test file splits the Go group without changing its bound, so the
	// source row's percentage and denominator can reach their maxima together.
	fixture.Disk["Ybound_test.go"] = file("")
	fixture.Classifications["Ybound_test.go"] = sloc.File_Partition{}
	if one.Show_Files {
		fill_line_bound_width(&fixture)
	}
	fill_line_bound_past(&fixture, line_bound_past_count(one.File_Count))
	return fixture
}

// Distinct languages keep each aggregate value separate, so the test partition sees
// the same minimum, one, two, and maximum properties as the source partition.
func disk_test_partitions() (fixture disk_fixture) {
	fixture = disk_fixture{
		Disk:            fstest.MapFS{},
		Classifications: map[sloc.Classified_Path]sloc.File_Partition{},
	}
	test_line_partitions_add(&fixture)
	test_dropped_partitions_add(&fixture)
	return fixture
}

type test_partition_add_input struct {
	Fixture *disk_fixture
	Label   string
	Suffix  string
	Kind    int
	Count   int
	Dropped bool
}

// Each modeled file stays in the production source bound. The group sum supplies the
// larger shared line property.
func test_partition_add(input *test_partition_add_input) {
	if input.Dropped {
		for left, index := input.Count, 0; left > 0; index++ {
			line_count := min(left, sloc.FILE_DROPPED_COUNT_MAX)
			name := fmt.Sprintf(
				"test/%s%02d%s", input.Label, index, input.Suffix)
			input.Fixture.Disk[name] = file("a")
			classified_path := sloc.Classified_Path(name)
			input.Fixture.Classifications[classified_path] = sloc.File_Partition{
				Code:    sloc.Code_Count(line_count),
				Dropped: sloc.Dropped_Count(line_count),
			}
			left -= line_count
		}
		return
	}
	text, _ := line_bound_line(input.Kind)
	per_file := sloc.SOURCE_BYTES_MAX / len(text)
	for left, index := input.Count, 0; left > 0; index++ {
		line_count := min(left, per_file)
		name := fmt.Sprintf("test/%s%02d%s", input.Label, index, input.Suffix)
		input.Fixture.Disk[name] = file("a")
		input.Fixture.Classifications[sloc.Classified_Path(name)] = line_bound_counts(
			&line_bound_counts_input{Kind: input.Kind, Line_Count: line_count})
		left -= line_count
	}
}

// Separate languages stop the aggregate values from combining before the harness sees
// their one, two, and maximum properties.
func test_line_partitions_add(fixture *disk_fixture) {
	test_partition_add(&test_partition_add_input{
		Fixture: fixture, Label: "code_max", Suffix: ".go",
		Kind: LINE_BOUND_KIND_CODE, Count: sloc.LINE_COUNT_MAX,
	})
	test_partition_add(&test_partition_add_input{
		Fixture: fixture, Label: "code_one", Suffix: ".zig",
		Kind: LINE_BOUND_KIND_CODE, Count: 1,
	})
	test_partition_add(&test_partition_add_input{
		Fixture: fixture, Label: "code_two_a", Suffix: ".odin",
		Kind: LINE_BOUND_KIND_CODE, Count: 1,
	})
	test_partition_add(&test_partition_add_input{
		Fixture: fixture, Label: "code_two_b", Suffix: ".odin",
		Kind: LINE_BOUND_KIND_CODE, Count: 1,
	})
	test_partition_add(&test_partition_add_input{
		Fixture: fixture, Label: "comment_max", Suffix: ".py",
		Kind: LINE_BOUND_KIND_COMMENT, Count: sloc.LINE_COUNT_MAX,
	})
	test_partition_add(&test_partition_add_input{
		Fixture: fixture, Label: "comment_one", Suffix: ".rb",
		Kind: LINE_BOUND_KIND_COMMENT, Count: 1,
	})
	test_partition_add(&test_partition_add_input{
		Fixture: fixture, Label: "comment_two", Suffix: ".sh",
		Kind: LINE_BOUND_KIND_COMMENT, Count: 2,
	})
	test_partition_add(&test_partition_add_input{
		Fixture: fixture, Label: "blank_max", Suffix: ".rs",
		Kind: LINE_BOUND_KIND_BLANK, Count: sloc.LINE_COUNT_MAX,
	})
	test_partition_add(&test_partition_add_input{
		Fixture: fixture, Label: "blank_one", Suffix: ".cpp",
		Kind: LINE_BOUND_KIND_BLANK, Count: 1,
	})
	test_partition_add(&test_partition_add_input{
		Fixture: fixture, Label: "blank_two", Suffix: ".java",
		Kind: LINE_BOUND_KIND_BLANK, Count: 2,
	})
}

// A dropped-line value also needs a counted partition. Code is the cheapest valid
// partition because one modeled byte can establish the repeated classification.
func test_dropped_partitions_add(fixture *disk_fixture) {
	for _, input := range []*test_partition_add_input{
		{Fixture: fixture, Label: "dropped_max", Suffix: ".c",
			Count: sloc.DROPPED_COUNT_MAX, Dropped: true},
		{Fixture: fixture, Label: "dropped_one", Suffix: ".ts", Count: 1, Dropped: true},
		{Fixture: fixture, Label: "dropped_two", Suffix: ".js", Count: 2, Dropped: true},
	} {
		test_partition_add(input)
	}
}

// A modeled empty population widens the file tally while the longest member widens
// the label; no extra lines compete with the language already at its line bound.
func fill_line_bound_width(fixture *disk_fixture) {
	for index := 0; index < LINE_BOUND_FILE_WIDTH_COUNT; index++ {
		name := fmt.Sprintf("Wwidth%05d.go", index)
		fixture.Disk[name] = file("")
		fixture.Classifications[sloc.Classified_Path(name)] = sloc.File_Partition{}
	}
	name := "X" + strings.Repeat("p", sloc.FILE_PATH_BYTES_MAX-4) + ".go"
	fixture.Disk[name] = file("")
	fixture.Classifications[sloc.Classified_Path(name)] = sloc.File_Partition{}
}

// The ordinary recipe witnesses two; dedicated seeds select one and a three-root
// partition whose merged report reaches the file tally bound exactly.
func line_bound_past_count(choice int) (count int) {
	if choice == 1 {
		return 1
	}
	return 2
}

// File_Count choice two names the special two-root witness without adding another
// decoded scenario field solely for one boundary.
func line_bound_past_max(one scenario) (maximum bool) {
	if int(one.Recipe)%RECIPE_COUNT != 31 {
		return false
	}
	return one.File_Count == 2
}

// Two roots keep their own omitted-file bounds valid while their tallies sum to the
// report maximum. Blank lines let the first root reach its ceiling with fewest files.
func disk_past_lines_max() (fixture disk_fixture) {
	go_disk := disk_fixture{
		Disk:            fstest.MapFS{},
		Classifications: map[sloc.Classified_Path]sloc.File_Partition{},
	}
	python_disk := disk_fixture{
		Disk:            fstest.MapFS{},
		Classifications: map[sloc.Classified_Path]sloc.File_Partition{},
	}
	fill_line_bound(&go_disk, LINE_BOUND_KIND_BLANK)
	fill_line_bound(&python_disk, LINE_BOUND_KIND_COMMENT)
	fill_past_language(&fill_past_language_input{
		Fixture: &go_disk,
		Count:   sloc.ROOT_PAST_LINES_MAX,
		Prefix:  "Zgo",
		Suffix:  ".rs",
	})
	fill_past_language(&fill_past_language_input{
		Fixture: &python_disk,
		Count:   sloc.FILES_COUNT_MAX - sloc.ROOT_PAST_LINES_MAX,
		Prefix:  "Zpython",
		Suffix:  ".py",
	})
	classifications := map[sloc.Classified_Path]sloc.File_Partition{}
	classifications_add(&classifications_add_input{
		Into: classifications,
		More: go_disk.Classifications,
	})
	classifications_add(&classifications_add_input{
		Into: classifications,
		More: python_disk.Classifications,
	})
	return disk_fixture{
		Disk: fstest.MapFS{},
		Roots: map[sloc.Root]fstest.MapFS{
			LINE_BOUND_ROOT_GO:     go_disk.Disk,
			LINE_BOUND_ROOT_PYTHON: python_disk.Disk,
		},
		Classifications: classifications,
	}
}

// The path prefixes and extensions keep both roots' classifier keys disjoint.
type fill_past_language_input struct {
	Fixture *disk_fixture
	Count   int
	Prefix  string
	Suffix  string
}

func fill_past_language(input *fill_past_language_input) {
	for index := 0; index < input.Count; index++ {
		name := fmt.Sprintf("%s%05d%s", input.Prefix, index, input.Suffix)
		input.Fixture.Disk[name] = file("a\n")
		input.Fixture.Classifications[sloc.Classified_Path(name)] =
			sloc.File_Partition{Code: 1}
	}
}

// A fresh destination makes the combined callback immutable before workers receive it.
type classifications_add_input struct {
	Into map[sloc.Classified_Path]sloc.File_Partition
	More map[sloc.Classified_Path]sloc.File_Partition
}

func classifications_add(input *classifications_add_input) {
	for name, counts := range input.More {
		input.Into[name] = counts
	}
}

// Files past the language bound remain exact modeled inputs so the expensive fixture
// cannot fall back to byte classification implicitly.
func fill_line_bound_past(fixture *disk_fixture, count int) {
	for index := 0; index < count; index++ {
		name := fmt.Sprintf("Zpast%05d.go", index)
		fixture.Disk[name] = file("a\n")
		fixture.Classifications[sloc.Classified_Path(name)] = sloc.File_Partition{Code: 1}
	}
}

// Adds one language's files, sized to land on the bound exactly. Exact rather than
// generous: the walk drops a file that would carry its language past the bound, so
// overshooting would leave the tally short and witness nothing.
func fill_line_bound(fixture *disk_fixture, kind int) {
	text, suffix := line_bound_line(kind)
	per_file := sloc.SOURCE_BYTES_MAX / len(text)
	for left, index := sloc.LINE_COUNT_MAX, 0; left > 0; index++ {
		lines := min(left, per_file)
		name := fmt.Sprintf("L%d%04d%s", kind, index, suffix)
		fixture.Disk[name] = file(text)
		fixture.Classifications[sloc.Classified_Path(name)] = line_bound_counts(
			&line_bound_counts_input{Kind: kind, Line_Count: lines})
		left -= lines
	}
}

// The model states only the partition implied by the representative source kind; the
// scaled equivalence specification keeps this mapping tied to the real scanner.
type line_bound_counts_input struct {
	Kind       int
	Line_Count int
}

func line_bound_counts(input *line_bound_counts_input) (counts sloc.File_Partition) {
	switch input.Kind {
	case LINE_BOUND_KIND_CODE:
		return sloc.File_Partition{Code: sloc.Code_Count(input.Line_Count)}
	case LINE_BOUND_KIND_COMMENT:
		return sloc.File_Partition{Comment: sloc.Comment_Count(input.Line_Count)}
	}
	return sloc.File_Partition{Blank: sloc.Blank_Count(input.Line_Count)}
}

// A Rust file whose block comments nest past the depth bound, so the carried depth
// saturates. One opener per line: a single line of them would run past the scan window
// and the openers beyond it would never be seen.
func disk_deep_comment() (disk fstest.MapFS) {
	openers := strings.Repeat("/*\n", DEEP_COMMENT_OPENERS)
	return fstest.MapFS{"deep.rs": file(openers + "still inside\n")}
}

// A tree holding more recognized files than the walk may take. The excess is what the
// overflow tally counts, and the tally saturates, so its own bound needs the excess to
// reach DROPPED_COUNT_MAX rather than merely to exist.
func disk_overflow(excess int) (disk fstest.MapFS) {
	disk = fstest.MapFS{}
	for index := range sloc.FILES_COUNT_MAX + excess {
		disk[fmt.Sprintf("v%06d.go", index)] = file("a\n")
	}
	return disk
}

// A tree whose lines are wider than the scan window, which the reader counts but reads
// short. One file cannot hold enough of them to saturate the tally — the source bound
// stops it — so they are spread over as many files as the count needs.
func disk_dropped(count int) (fixture disk_fixture) {
	fixture = disk_fixture{
		Disk:            fstest.MapFS{},
		Classifications: map[sloc.Classified_Path]sloc.File_Partition{},
	}
	wide := strings.Repeat("a", sloc.LINE_BYTES_MAX+DROPPED_LINE_MARGIN) + "\n"
	per_file := sloc.FILE_DROPPED_COUNT_MAX
	// Splitting the two-line fixture witnesses a two-entry model without making any
	// production tally or the large saturation fixture more expensive.
	if count == 2 {
		per_file = 1
	}
	for left, index := count, 0; left > 0; index++ {
		lines := min(left, per_file)
		name := fmt.Sprintf("w%06d.go", index)
		fixture.Disk[name] = file(wide)
		fixture.Classifications[sloc.Classified_Path(name)] = sloc.File_Partition{
			Code:    sloc.Code_Count(lines),
			Dropped: sloc.Dropped_Count(lines),
		}
		left -= lines
	}
	return fixture
}

// Recipe 29 uses its ordinary seed for saturation and two dedicated choices for the
// decimal widths below it; recipes 27 and 28 retain the one and two value witnesses.
func dropped_fixture_count(recipe uint8, choice int) (count int) {
	if recipe == 27 {
		return 1
	}
	if recipe == 28 {
		return 2
	}
	if choice == 1 {
		return 10
	}
	if choice == 2 {
		return sloc.DROPPED_COUNT_MAX - 1
	}
	return DROPPED_SATURATING_LINES
}

// One production-reachable tree combines every omission so the maximum dropped-section
// shape is witnessed without constructing another independent file-bound fixture.
func disk_all_dropped() (fixture disk_fixture) {
	fixture = disk_fixture{
		Disk:            fstest.MapFS{},
		Classifications: map[sloc.Classified_Path]sloc.File_Partition{},
	}
	fill_line_bound(&fixture, LINE_BOUND_KIND_CODE)
	fill_line_bound(&fixture, LINE_BOUND_KIND_COMMENT)
	fill_line_bound(&fixture, LINE_BOUND_KIND_BLANK)
	fixture.Disk["Ybound_test.go"] = file("")
	fixture.Classifications["Ybound_test.go"] = sloc.File_Partition{}
	fixture.Disk["Zpast.go"] = file("a\n")
	fixture.Classifications["Zpast.go"] = sloc.File_Partition{Code: 1}
	fixture.Disk["a_broken.go"] = file("a\n")
	fixture.Disk["a_binary.go"] = file("a\x00\n")
	fixture.Disk["a_oversized.go"] = file(strings.Repeat("a", sloc.SOURCE_BYTES_MAX+1))
	fixture.Disk["a_wide.c"] = file(
		strings.Repeat("a", sloc.LINE_BYTES_MAX+DROPPED_LINE_MARGIN) + "\n")
	fixture.Classifications["a_wide.c"] = sloc.File_Partition{Code: 1, Dropped: 1}
	for index := 0; index < sloc.FILES_COUNT_MAX; index++ {
		fixture.Disk[fmt.Sprintf("m%06d.go", index)] = file("")
	}
	for index := 0; index <= sloc.DROPPED_COUNT_MAX; index++ {
		fixture.Disk[fmt.Sprintf("z%06d.go", index)] = file("")
	}
	return fixture
}

// Files the walk can see but the reader cannot take. Nothing in fstest fails a read, so
// the failure is injected by broken_disk around the tree rather than inside it.
func disk_broken(count int) (disk fstest.MapFS) {
	disk = fstest.MapFS{"fine.go": file("a\n")}
	for index := range count {
		disk[fmt.Sprintf("%s%06d.go", BROKEN_PREFIX, index)] = file("a\n")
	}
	return disk
}

// A tree whose files all hold a zero byte in their opening chunk, so the binary tally
// rather than the counted one is what fills.
func disk_binary_many(count int) (disk fstest.MapFS) {
	disk = fstest.MapFS{"fine.go": file("a\n")}
	for index := range count {
		disk[fmt.Sprintf("b%06d.go", index)] = file("package\x00main\n")
	}
	return disk
}

// A tree of files past the source bound. Every path shares one immutable entry because
// file identity is irrelevant once the production size check rejects its contents.
func disk_oversized_many(count int) (disk fstest.MapFS) {
	disk = fstest.MapFS{}
	if count < sloc.FILES_COUNT_MAX {
		disk["fine.go"] = file("a\n")
	}
	over := file(strings.Repeat("a", sloc.SOURCE_BYTES_MAX+1))
	for index := range count {
		disk[fmt.Sprintf("o%06d.go", index)] = over
	}
	return disk
}

// The decoded file count normally leaves recipe 23 on its two-value witness; its
// widest ordinary choice selects the production file bound for the maximum tally.
func oversized_fixture_count(choice int) (count int) {
	if choice == 512 {
		return sloc.FILES_COUNT_MAX
	}
	return 2
}

// One source file and one test file, a single code line each and nothing else in the
// tree, so the group's own code total — the denominator a sub-row's share is taken
// against — is exactly two.
func disk_pair() (disk fstest.MapFS) {
	return fstest.MapFS{
		"p/a.go":      file("a\n"),
		"p/a_test.go": file("b\n"),
	}
}

// Returns a file entry holding the given text.
func file(text string) (entry *fstest.MapFile) {
	return &fstest.MapFile{Data: []byte(text)}
}

// A tree whose BROKEN_PREFIX files fail to open and to read. ReadFile is overridden as
// well as Open because fs.ReadFile prefers the embedded MapFS's own implementation and
// would otherwise never reach the failure.
type broken_disk struct {
	fstest.MapFS
}

func (disk broken_disk) Open(name string) (file fs.File, err error) {
	if strings.Contains(name, BROKEN_PREFIX) {
		return nil, errors.New("open failed")
	}
	return disk.MapFS.Open(name)
}

func (disk broken_disk) ReadFile(name string) (content []byte, err error) {
	if strings.Contains(name, BROKEN_PREFIX) {
		return nil, errors.New("read failed")
	}
	entry, found := disk.MapFS[name]
	if found {
		// Simulation fixtures are immutable after construction, so returning their
		// bytes directly preserves ReadFile semantics without copying shared bound data.
		return entry.Data, nil
	}
	return disk.MapFS.ReadFile(name)
}

// One small Go file: the ordinary case every other recipe varies from.
func disk_simple() (disk fstest.MapFS) {
	return fstest.MapFS{
		"main.go": file("package main\n\n// A comment.\nfunc main() {}\n"),
	}
}

// One file per recognized extension, so every language constructor runs and the
// language table's own field bounds are witnessed across the whole seeded set.
func disk_languages() (disk fstest.MapFS) {
	disk = fstest.MapFS{}
	for _, extension := range recognized_extensions() {
		disk["a"+extension] = file("x\n# c\n\n")
	}
	// The extensionless names resolve by base name rather than extension.
	disk["Makefile"] = file("all:\n\t# c\n")
	disk["makefile"] = file("all:\n")
	disk["GNUmakefile"] = file("all:\n")
	disk["Dockerfile"] = file("FROM x\n# c\n")
	disk["CMakeLists.txt"] = file("# c\nproject(x)\n")
	// A name ending in a dot yields the one-byte "." extension.
	disk["trailing."] = file("x\n")
	return disk
}

// A file whose single line reaches the chosen width, so the scan window and the cursor
// that walks it are witnessed at their bounds.
func disk_wide_line(one scenario) (disk fstest.MapFS) {
	width := one.Line_Bytes
	if width < 1 {
		width = 1
	}
	return fstest.MapFS{
		"wide.go":  file(strings.Repeat("a", width) + "\n"),
		"exact.go": file(strings.Repeat("b", sloc.LINE_BYTES_MAX) + "\n"),
		"over.go":  file(strings.Repeat("c", sloc.LINE_BYTES_MAX+64) + "\n"),
		"tiny.go":  file("a\nbc\n\n \n"),
	}
}

// A file with many lines, so the line tallies climb toward their bound.
func disk_many_lines(one scenario) (disk fstest.MapFS) {
	count := one.Line_Count
	if count < 1 {
		count = 1
	}
	return fstest.MapFS{
		"many.go":  file(strings.Repeat("a\n", count)),
		"blank.go": file(strings.Repeat("\n", count)),
		"note.go":  file(strings.Repeat("// c\n", count)),
	}
}

// A file carrying the byte values a source byte's bound names. The NUL sits past the
// binary sniff window so the file is still read as text.
func disk_byte_extremes() (disk fstest.MapFS) {
	// The blank check stops at the first byte that is not a space, so an extreme byte
	// is only ever handed to it when it leads its own line.
	lead := strings.Repeat("a\n", sloc.BINARY_SNIFF_BYTES/2+8)
	tail := "\x00a\n\x01a\n\x02a\n\xffa\n"
	// A heredoc operator followed by an extreme byte is what hands that byte to the
	// quote check and the identifier check.
	// The NUL must sit past the binary sniff window or the whole file is dropped
	// before a single line of it is ever scanned.
	heredoc := strings.Repeat("x\n", sloc.BINARY_SNIFF_BYTES/2+8) +
		"cat <<\x00\ncat <<\x01\ncat <<\x02\ncat <<\xff\ncat <<0\ncat <<z\n"
	return fstest.MapFS{
		"bytes.go":    file(lead + tail),
		"bytes.sh":    file(heredoc),
		"spaces.go":   file(" \t\r\f\v\n"),
		"identify.sh": file("cat <<_A\nbody\n_A\ncat <<Z9\nbody\nZ9\n"),
	}
}

// Many small files, so the file tallies climb and the worker pool actually fans out.
func disk_many_files(one scenario) (disk fstest.MapFS) {
	count := one.File_Count
	if count < 1 {
		count = 1
	}
	disk = fstest.MapFS{}
	for index := 0; index < count; index++ {
		disk["d/f"+decimal(index)+".go"] = file("a\n")
	}
	return disk
}

// A file holding a NUL inside the sniff window, which the reader drops as binary.
func disk_binary() (disk fstest.MapFS) {
	return fstest.MapFS{
		"blob.go": file("package\x00main\n"),
		"good.go": file("package main\n"),
	}
}

// Hidden entries, entries the injected predicate ignores, and the test-file shapes.
func disk_hidden_and_ignored() (disk fstest.MapFS) {
	return fstest.MapFS{
		".c":             file("int x;\n"),
		".hidden/a.go":   file("a\n"),
		"visible.go":     file("a\n"),
		"ignored/b.go":   file("b\n"),
		"ignored.go":     file("c\n"),
		"test/c.go":      file("c\n"),
		"spec/d.go":      file("d\n"),
		"__tests__/e.js": file("e\n"),
		"f_test.go":      file("f\n"),
		"test_g.py":      file("g\n"),
	}
}

// The scanner's corners: heredocs, long brackets, hashable raw strings, nesting block
// comments, character literals and lifetimes, and the delimiters that carry the widest
// computed closers.
func disk_scanner_corners() (disk fstest.MapFS) {
	hashes := strings.Repeat("#", sloc.HASH_COUNT_MAX)
	equals := strings.Repeat("=", sloc.HASH_COUNT_MAX)
	nested := strings.Repeat("/*", sloc.NESTING_DEPTH_MAX+5)
	disk = fstest.MapFS{
		"heredoc.sh": file(
			"cat <<EOF\n# not a comment\nEOF\n" +
				"cat <<-'END'\nbody\nEND\n" +
				"cat << ~X\nbody\nX\n" +
				"a << b\n" +
				// The widest terminator a line can carry: the operator at
				// the line's very start and an identifier filling the rest.
				"<<" + strings.Repeat("W", sloc.LINE_BYTES_MAX-2) + "\n" +
				strings.Repeat("b", sloc.LINE_BYTES_MAX) + "\n"),
		"bracket.lua": file(
			"--[[ long comment ]]\n" +
				"--[[\ncomment\n]]\n" +
				// A single-level bracket, whose computed closer is the width
				// between the bare pair and the run at its bound.
				"--[=[ level one ]=]\n" +
				"local m = [=[ body ]=]\n" +
				"local s = [[ long string ]]\n" +
				"local u = [[\nbody\n]]\n" +
				"local t = [" + equals + "[\nbody\n]" + equals + "]\n" +
				"--[" + equals + "[\ncomment\n]" + equals + "]\n" +
				"-- plain\n"),
		"raw.rs": file(
			"let a = r\"x\";\n" +
				"let b = br#\"x\"#;\n" +
				// Two hashes witness an interior computed-closer width.
				"let b2 = br##\"x\"##;\n" +
				"let c = r" + hashes + "\"\nbody\n\"" + hashes + ";\n" +
				"let d = 'a';\n" +
				"let e = '\\n';\n" +
				"let f: &'static str = \"x\";\n" +
				// A lead with no quote after it is a raw identifier, not a string,
				// so the hashable match reports its failure rather than opening.
				"let g = r + 1;\n" +
				"/* nesting /* inner */ still */\n" +
				nested + "\n"),
		"verbatim.py": file(
			"x = '''\nbody\n'''\n" +
				"y = \"\"\"\nbody\n\"\"\"\n" +
				"z = 'plain'\n"),
		// Nix's indented string opens and closes on a two-byte pair, the only
		// verbatim form of that width any seeded language declares.
		"indented.nix": file("x = ''\n  body\n'';\ny = \"plain\";\n"),
		// A quoted string carrying a real escape, which the character-literal fixtures
		// above never reach because a character-like form takes its own route.
		"escape.go": file(
			"s := \"a\\\\nb\"\nt := \"plain\"\n" +
				// A backtick raw string, whose closer is a single byte.
				"u := `\nraw\n`\n"),
		// Two test files in one language and one source file beside them, sized so the
		// group's own code total is exactly two.
		"pair/a_test.go": file("a\n"),
		"pair/b_test.go": file("\n"),
		"pair/c.go":      file("c\n"),
		// An extension no language claims, wide enough to fall through every lookup.
		"fallthrough.zzzzzzzzzz": file("x\n"),
		// A one-byte heredoc word, and a one-byte body line inside it.
		"short.sh":    file("cat <<A\nx\nA\n"),
		"markup.html": file("<!-- c -->\n<p>x</p>\n"),
		"lisp.el":     file("; c\n(x)\n"),
	}
	for name, entry := range disk_scanner_positions() {
		disk[name] = entry
	}
	return disk
}

// Scanner fragments place syntax triggers at every shared position boundary.
func disk_scanner_positions() (disk fstest.MapFS) {
	return fstest.MapFS{
		// Invalid fragments are intentional. They place each Lua trigger at the shared
		// active-position boundaries without bypassing the production scanner.
		"positions.lua": file(
			"[\n[x\nx[\nxx[\n" +
				strings.Repeat("x", sloc.LINE_BYTES_MAX-1) + "[\n" +
				strings.Repeat("x", sloc.LINE_BYTES_MAX-2) + "[x\n" +
				strings.Repeat("x", sloc.LINE_BYTES_MAX-2) + "--\n"),
		// The carried body makes the long-comment reader visit both line-width bounds
		// and the last active cursor.
		"comment_positions.lua": file(
			"--[[\nx\n" + strings.Repeat("x", sloc.LINE_BYTES_MAX) + "\n]]\n"),
		// A partial operator still reaches heredoc recognition and keeps malformed
		// forms from opening carried state.
		"positions.sh": file(
			"<\n<<\nx<\n" + strings.Repeat("x", sloc.LINE_BYTES_MAX-1) + "<\n"),
		// Each unmatched quote ends with its line, so every following line starts fresh.
		"quote_positions.go": file(
			"\"\nx\"\nxx\"\n" +
				strings.Repeat("x", sloc.LINE_BYTES_MAX-1) + "\"\n"),
		// The closing-only lines reset the raw carry before the next opener position.
		"verbatim_positions.go": file(
			"`\n`\nx`\n`\nxx`\n`\n" +
				strings.Repeat("x", sloc.LINE_BYTES_MAX-1) + "`\n`\n" +
				"`\n" + strings.Repeat("x", sloc.LINE_BYTES_MAX) + "\n`\n"),
		// Rust's apostrophe can start a character or a lifetime. These malformed forms
		// keep that decision at each position boundary.
		"character_positions.rs": file(
			"'\nx'\nxx'\n" +
				strings.Repeat("x", sloc.LINE_BYTES_MAX-1) + "'\n"),
		// Pascal contributes the one-byte block pair and the no-escape string form.
		"positions.pas": file(
			"{\nx\n" + strings.Repeat("x", sloc.LINE_BYTES_MAX) + "\n}\n'x'\n"),
	}
}

// A tree sized so every printed column reaches its own width bound at once: the file
// tally fills its cell, the line tallies fill theirs, a per-file row carries a path at
// the path bound, and one language's tests hold no code so its source row prints a
// full hundred percent.
func disk_wide_table(kind int) (fixture disk_fixture) {
	return disk_wide_table_kind(kind, false)
}

// The JSON boundary needs the maximum source-file property. The table variant keeps
// one test file because its per-file row bound also needs a source/test split.
func disk_wide_source_table() (fixture disk_fixture) {
	return disk_wide_table_kind(WIDE_KIND_SINGLE, true)
}

// Builds one wide table and selects whether its closing file stays in the source role.
func disk_wide_table_kind(kind int, source_only bool) (fixture disk_fixture) {
	fixture = disk_fixture{
		Disk:            fstest.MapFS{},
		Classifications: map[sloc.Classified_Path]sloc.File_Partition{},
	}
	// Only the spread table needs many files and many lines at once — it is the one
	// fixture whose table reaches every column bound together. The rest need one or
	// the other, so they are built tall or wide but never both, which is what keeps
	// the suite inside its timeout.
	file_count := wide_files(kind)
	per_file := wide_lines(kind)
	body := wide_line(kind)
	counts := wide_counts(&wide_counts_input{Kind: kind, Line_Count: per_file})
	shape := &wide_table_shape{
		Fixture:     &fixture,
		Kind:        kind,
		File_Count:  file_count,
		Line_Count:  per_file,
		Source:      body,
		Counts:      counts,
		Suffix:      wide_suffix(kind),
		Source_Only: source_only,
		Extensions:  recognized_extensions(),
	}
	wide_table_series_add(shape)
	wide_table_boundaries_add(shape)
	return fixture
}

type wide_table_shape struct {
	Fixture     *disk_fixture
	Kind        int
	File_Count  int
	Line_Count  int
	Source      string
	Counts      sloc.File_Partition
	Suffix      string
	Source_Only bool
	Extensions  []string
}

// The walk stops selecting at the file bound, so the series is exactly two files short
// of it before the maximum path and closing partition are added.
func wide_table_series_add(shape *wide_table_shape) {
	for index := 0; index < shape.File_Count-2; index++ {
		// Spread over every extension, or concentrated in one: the row count and the
		// taxonomy reach their bounds only when spread, and a single language group's
		// own tallies reach theirs only when concentrated.
		name := shape.Suffix
		if shape.Kind == WIDE_KIND_SPREAD {
			name = shape.Extensions[index%len(shape.Extensions)]
		}
		prefix := "w/f"
		if shape.Kind == WIDE_KIND_SPREAD {
			prefix = "test/f"
		}
		wide_file_add(&wide_file_add_input{
			Fixture: shape.Fixture,
			Name:    prefix + decimal(index) + name,
			Source:  shape.Source,
			Counts:  shape.Counts,
		})
	}
}

// The non-series files carry the independent maxima and the partition that cannot be
// represented by the repeated body.
func wide_table_boundaries_add(shape *wide_table_shape) {
	// Exactly the path bound, so the label column and the counted-path bound are both
	// witnessed at their maximum rather than one byte short of it.
	long := "w/" +
		strings.Repeat("p", sloc.FILE_PATH_BYTES_MAX-2-len(shape.Suffix)) + shape.Suffix
	wide_file_add(&wide_file_add_input{
		Fixture: shape.Fixture,
		Name:    long,
		Source:  shape.Source,
		Counts:  shape.Counts,
	})
	// The spread table's test file holds no code at all, so its language's source row
	// is the whole of that language's code and prints a full hundred percent. Every
	// other kind fills it like the rest, so one group's own totals reach their bound.
	final_body := shape.Source
	final_counts := shape.Counts
	final_name := "w/z" + shape.Suffix
	if shape.Kind == WIDE_KIND_SPREAD {
		// The spread table's closing file is a test with no code at all, so its
		// language's source row is the whole of that language's code.
		final_body = "\n"
		final_counts = sloc.File_Partition{Blank: sloc.Blank_Count(shape.Line_Count)}
		final_name = "w/z_test.go"
	}
	if shape.Kind == WIDE_KIND_CODE {
		// The tall code table's closing file is a test carrying code, so the group
		// splits and the denominator its shares are taken against is the line bound.
		final_name = "w/z_test.go"
	}
	if shape.Kind == WIDE_KIND_SINGLE {
		if !shape.Source_Only {
			final_name = "w/z_test.go"
		}
	}
	wide_file_add(&wide_file_add_input{
		Fixture: shape.Fixture,
		Name:    final_name,
		Source:  final_body,
		Counts:  final_counts,
	})
	// One file past the bound, so the walk actually reaches the point where it stops
	// selecting rather than merely ending with the last one it took. Only the kinds
	// built at the file bound get it: elsewhere the walk would take it and the tree's
	// lines would pass the total bound, which Count refuses outright.
	if shape.File_Count == sloc.FILES_COUNT_MAX {
		wide_file_add(&wide_file_add_input{
			Fixture: shape.Fixture,
			Name:    "w/zz_overflow" + shape.Suffix,
			Source:  shape.Source,
			Counts:  shape.Counts,
		})
	}
}

// Both maps are filled together so the classifier model cannot name a file that the
// filesystem does not expose, or omit one whose repeated source it replaces.
type wide_file_add_input struct {
	Fixture *disk_fixture
	Name    string
	Source  string
	Counts  sloc.File_Partition
}

func wide_file_add(input *wide_file_add_input) {
	input.Fixture.Disk[input.Name] = file(input.Source)
	input.Fixture.Classifications[sloc.Classified_Path(input.Name)] = input.Counts
}

// WIDE_KIND_SPREAD is the wide-table kind that spreads its files across every
// recognized extension rather than concentrating them in one language.
const WIDE_KIND_SPREAD = 0

// WIDE_KIND_COMMENT is the kind whose every line is a comment, in a language that
// takes "#" as one.
const WIDE_KIND_COMMENT = 1

// WIDE_KIND_BLANK is the kind whose every line is blank.
const WIDE_KIND_BLANK = 2

// WIDE_KIND_SINGLE is the kind that concentrates every file into one language.
const WIDE_KIND_SINGLE = 3

// WIDE_KIND_TESTS is the kind whose every file is a test file.
const WIDE_KIND_TESTS = 4

// WIDE_KIND_CODE is the tall kind whose every line is code, in one language, so that
// language's own code tally reaches the line bound.
const WIDE_KIND_CODE = 5

// Returns the wide-table shape assigned to a recipe. Keeping this mapping beside the
// shape constants prevents the driver and its model from choosing differently.
func wide_kind_for_recipe(recipe uint8) (kind int, wide bool) {
	switch recipe {
	case 10:
		return WIDE_KIND_SPREAD, true
	case 13:
		return WIDE_KIND_COMMENT, true
	case 14:
		return WIDE_KIND_BLANK, true
	case 15:
		return WIDE_KIND_SINGLE, true
	case 16:
		return WIDE_KIND_TESTS, true
	case 17:
		return WIDE_KIND_CODE, true
	}
	return 0, false
}

// WIDE_TALL_FILES is how many files the tally kinds spread their lines over. They
// exist to put a line column on its bound, not a file column, so a handful of tall
// files costs a fraction of a tree at the file bound.
const WIDE_TALL_FILES = 16

// WIDE_SPREAD_LINES is what each file carries when the tree spreads over the file
// bound: 16 lines across 65,535 files is a seven-figure total from tiny files.
const WIDE_SPREAD_LINES = 16

// WIDE_TALL_LINES is what each file carries when the tree is a few tall files instead,
// chosen so the total lands in the same seven figures the spread shape reaches.
//
// It is deliberately NOT sized to witness a ten-byte tally cell. Doing that needs a
// ten-million-line tree, and the fuzzer replays every recipe many times over: measured
// at 8m24s of suite for that one fixture. Numeric width axes have to be witnessed by
// something other than a big number.
const WIDE_TALL_LINES = 65536

// Returns how many files the wide table builds. The kinds that widen a file column or
// a row count need a tree at the file bound; the kinds that widen a line column reach
// the same total over far fewer, taller files.
func wide_files(kind int) (count int) {
	switch kind {
	case WIDE_KIND_COMMENT:
		return WIDE_TALL_FILES
	case WIDE_KIND_BLANK:
		return WIDE_TALL_FILES
	case WIDE_KIND_CODE:
		return WIDE_TALL_FILES
	}
	return sloc.FILES_COUNT_MAX
}

// Returns how many lines each of the wide table's files carries. These sized off the
// line bound until the bound became eight figures: a hundred million lines is two
// hundred megabytes of fixture and minutes of scanning, so the tree can no longer be
// driven to it. They now size for table *shape* — a seven-figure total, wide enough to
// fill every column, spread either over the file bound or over a few tall files.
func wide_lines(kind int) (count int) {
	if wide_files(kind) == sloc.FILES_COUNT_MAX {
		return WIDE_SPREAD_LINES
	}
	return WIDE_TALL_LINES
}

// Returns the extension the wide table's files carry. The comment kind needs a
// language that reads "#" as a comment, or its lines would count as code.
func wide_suffix(kind int) (suffix string) {
	switch kind {
	case WIDE_KIND_COMMENT:
		return ".py"
	case WIDE_KIND_TESTS:
		return "_test.go"
	}
	return ".go"
}

// Returns the line the wide table is filled with: code, comment, or blank, so each
// tally column in turn reaches the width its own bound names.
func wide_line(kind int) (text string) {
	switch kind {
	case WIDE_KIND_COMMENT:
		return "# c\n"
	case WIDE_KIND_BLANK:
		return "\n"
	}
	return "a\n"
}

// The repeated source establishes one line kind; the scaled equivalence specification
// anchors this arithmetic to the scanner before the large fixture uses it.
type wide_counts_input struct {
	Kind       int
	Line_Count int
}

func wide_counts(input *wide_counts_input) (counts sloc.File_Partition) {
	switch input.Kind {
	case WIDE_KIND_COMMENT:
		return sloc.File_Partition{Comment: sloc.Comment_Count(input.Line_Count)}
	case WIDE_KIND_BLANK:
		return sloc.File_Partition{Blank: sloc.Blank_Count(input.Line_Count)}
	}
	return sloc.File_Partition{Code: sloc.Code_Count(input.Line_Count)}
}

// Paths that resolve to no language at all, and a bare name with no extension.
func disk_unrecognized() (disk fstest.MapFS) {
	return fstest.MapFS{
		"notes.txt": file("x\n"),
		"README":    file("x\n"),
		"a":         file("x\n"),
		"good.go":   file("a\n"),
		// A name that is one long dotted suffix, so the extension the lookups are
		// handed is a whole path. It leads with a dot, so only a run counting hidden
		// entries reaches it.
		"." + strings.Repeat("z", sloc.FILE_PATH_BYTES_MAX-1): file("x\n"),
	}
}

// A file past the source bound, which the reader drops the way it drops a binary one.
func disk_oversized() (fixture disk_fixture) {
	fixture = disk_fixture{
		Disk: fstest.MapFS{
			"huge.go":  file(strings.Repeat("a", sloc.SOURCE_BYTES_MAX+1)),
			"fine.go":  file("a\n"),
			"empty.go": file(""),
			"one.go":   file("a"),
			"two.go":   file("a\n"),
			// Exactly the source bound, so the widest file the reader accepts is read
			// rather than only the one past it that is dropped.
			"exact.go": file(strings.Repeat("a\n", sloc.SOURCE_BYTES_MAX/2)),
		},
		Classifications: map[sloc.Classified_Path]sloc.File_Partition{
			"fine.go":  {Code: 1},
			"empty.go": {},
			"one.go":   {Code: 1},
			"two.go":   {Code: 1},
			"exact.go": {Code: sloc.SOURCE_BYTES_MAX / 2},
		},
	}
	return fixture
}

// Renders a non-negative integer without pulling in a formatter.
func decimal(value int) (text string) {
	if value == 0 {
		return "0"
	}
	digits := []byte{}
	for value > 0 {
		digits = append([]byte{byte('0' + value%10)}, digits...)
		value /= 10
	}
	return string(digits)
}

// Encodes a scenario back into the bytes decode_scenario reads, so a seed can be
// written as a struct rather than as a hand-packed buffer. The order here mirrors
// decode_scenario exactly; a field added there must be added here.
func build(one scenario) (data []byte) {
	data = append(data, one.Recipe)
	data = append(data, flag_byte(one.Show_Files))
	data = append(data, flag_byte(one.No_Ignore))
	data = append(data, flag_byte(one.Hidden))
	data = append(data, flag_byte(one.Json))
	data = append(data, flag_byte(one.Ignore_Some))
	data = append(data, flag_byte(one.Explicit))
	data = append(data, flag_byte(one.Stat_Error))
	data = append(data, flag_byte(one.Read_Error))
	data = append(data, flag_byte(one.Bad_Argument))
	data = append(data, uint32_bytes(uint32(one.Root_Bytes))...)
	data = append(data, uint32_bytes(uint32(one.Argument_Pad))...)
	data = append(data, uint32_bytes(uint32(one.Line_Bytes))...)
	data = append(data, uint32_bytes(uint32(one.Line_Count))...)
	data = append(data, uint32_bytes(uint32(one.File_Count))...)
	data = append(data, uint32_bytes(uint32(one.Concurrency))...)
	return data
}

// Returns the byte a flag decodes from.
func flag_byte(value bool) (encoded uint8) {
	if value {
		return 1
	}
	return 0
}

// Returns a uint32 in the big-endian order the cursor reads.
func uint32_bytes(value uint32) (encoded []byte) {
	return []byte{
		byte(value >> 24), byte(value >> 16), byte(value >> 8), byte(value),
	}
}

// Returns the ordinary scenario every other seed varies from.
func base_scenario() (one scenario) {
	return scenario{
		Recipe:       1,
		Show_Files:   false,
		No_Ignore:    false,
		Hidden:       false,
		Json:         false,
		Ignore_Some:  false,
		Explicit:     false,
		Stat_Error:   false,
		Read_Error:   false,
		Bad_Argument: false,
		Root_Bytes:   0,
		Argument_Pad: 0,
		Line_Bytes:   0,
		Line_Count:   0,
		File_Count:   0,
		Concurrency:  5,
	}
}

// Returns the base scenario with the given changes applied.
func with(change func(one *scenario)) (data []byte) {
	one := base_scenario()
	change(&one)
	return build(one)
}

// The seed corpus: one honest end-to-end run per boundary the pipeline can reach. Go's
// fuzzer guides on edges, not values, so these are seeded, not discovered.
func seed_corpus() (seeds [][]byte) {
	seeds = append(seeds, build(base_scenario()))
	seeds = append(seeds, seeds_recipes()...)
	seeds = append(seeds, seeds_flags()...)
	seeds = append(seeds, seeds_widths()...)
	seeds = append(seeds, seeds_counts()...)
	seeds = append(seeds, seeds_failure()...)
	seeds = append(seeds, seeds_concurrency()...)
	return seeds
}

// One seed per synthetic tree, so every recipe is reached at least once.
func seeds_recipes() (seeds [][]byte) {
	for index := 0; index < RECIPE_COUNT; index++ {
		recipe := uint8(index)
		// The widest tables are seeded on their own, in exactly the modes that reach
		// their bounds: each costs a tree at the file bound, so driving them once per
		// mode rather than once per knob is what keeps the run inside its timeout.
		if is_wide_recipe(recipe) {
			continue
		}
		seeds = append(seeds, with(func(one *scenario) {
			one.Recipe = recipe
			one.Line_Bytes = sloc.LINE_BYTES_MAX
			one.Line_Count = 3
			one.File_Count = 3
		}))
	}
	return seeds
}

// Reports whether a recipe builds a tree at the file bound, which is expensive enough
// that it is driven only where a bound needs it.
func is_wide_recipe(recipe uint8) (wide bool) {
	_, wide = wide_kind_for_recipe(recipe)
	return wide
}

// Every flag, alone and together, over a tree that has something to say about each.
func seeds_flags() (seeds [][]byte) {
	seeds = append(seeds, with(func(one *scenario) { one.Show_Files = true }))
	seeds = append(seeds, with(func(one *scenario) { one.Json = true }))
	seeds = append(seeds, with(func(one *scenario) {
		one.Hidden = true
		one.Recipe = 8
	}))
	seeds = append(seeds, with(func(one *scenario) {
		one.Recipe = 8
		one.Ignore_Some = true
	}))
	seeds = append(seeds, with(func(one *scenario) {
		one.Recipe = 8
		one.Ignore_Some = true
		one.No_Ignore = true
	}))
	seeds = append(seeds, with(func(one *scenario) {
		one.Recipe = 8
		one.Show_Files = true
		one.Hidden = true
		one.Json = true
	}))
	seeds = append(seeds, with(func(one *scenario) {
		one.Recipe = 2
		one.Show_Files = true
	}))
	seeds = append(seeds, with(func(one *scenario) {
		one.Recipe = 2
		one.Json = true
	}))
	return seeds
}

// The width boundaries: the root argument, the scan window, and the widest table.
func seeds_widths() (seeds [][]byte) {
	for _, width := range []int{1, 2, 3, sloc.ROOT_BYTES_MAX} {
		root := width
		seeds = append(seeds, with(func(one *scenario) { one.Root_Bytes = root }))
	}
	for _, width := range []int{1, 2, 3, sloc.LINE_BYTES_MAX - 1, sloc.LINE_BYTES_MAX} {
		line := width
		seeds = append(seeds, with(func(one *scenario) {
			one.Recipe = 3
			one.Line_Bytes = line
		}))
	}
	// The spread table is the only one whose label column and serialized rows need
	// their own modes; the rest widen a tally column, which a plain run reaches.
	seeds = append(seeds, with(func(one *scenario) {
		one.Recipe = 10
		one.Show_Files = true
	}))
	seeds = append(seeds, with(func(one *scenario) {
		one.Recipe = 10
		one.Json = true
	}))
	for _, recipe := range []uint8{13, 14, 15, 16, 17} {
		wide := recipe
		if wide != 15 {
			seeds = append(seeds, with(func(one *scenario) { one.Recipe = wide }))
		}
		seeds = append(seeds, with(func(one *scenario) {
			one.Recipe = wide
			one.Json = true
		}))
	}
	// One language holding every file, broken down per file: the widest a single
	// language's own contribution to the table can be.
	seeds = append(seeds, with(func(one *scenario) {
		one.Recipe = 15
		one.Show_Files = true
	}))
	// The tree at the line bound is the most expensive one the suite builds, so it is
	// driven once per renderer and never per knob. The generic recipe loop already
	// seeds its table run; this is the serialized one.
	seeds = append(seeds, with(func(one *scenario) {
		one.Recipe = 31
		one.Json = true
	}))
	return seeds
}

// The count boundaries: lines per file, files per tree, and arguments per command line.
func seeds_counts() (seeds [][]byte) {
	for _, recipe := range []uint8{0, 32, 33} {
		selected := recipe
		seeds = append(seeds, with(func(one *scenario) {
			one.Recipe = selected
			one.Json = true
		}))
	}
	for _, count := range []int{0, 1, 2, 3, 4096, LINE_COUNT_CHOICE_MAX} {
		lines := count
		seeds = append(seeds, with(func(one *scenario) {
			one.Recipe = 4
			one.Line_Count = lines
		}))
	}
	// The per-file breakdown of a tree holding a two-byte path, which only a run
	// counting hidden entries reaches.
	seeds = append(seeds, with(func(one *scenario) {
		one.Recipe = 8
		one.Hidden = true
		one.Show_Files = true
	}))
	for _, count := range []int{1, 2, 3, 512} {
		files := count
		seeds = append(seeds, with(func(one *scenario) {
			one.Recipe = 6
			one.File_Count = files
		}))
	}
	for _, count := range []int{1, 2, 32, sloc.ARGUMENTS_COUNT_MAX} {
		pad := count
		seeds = append(seeds, with(func(one *scenario) { one.Argument_Pad = pad }))
	}
	for _, choice := range []int{1, 2} {
		picked := choice
		seeds = append(seeds, with(func(one *scenario) {
			one.Recipe = 29
			one.File_Count = picked
		}))
	}
	seeds = append(seeds, with(func(one *scenario) {
		one.Recipe = 23
		one.File_Count = 512
	}))
	seeds = append(seeds, with(func(one *scenario) {
		one.Recipe = 31
		one.File_Count = 2
	}))
	return seeds
}

// The host failures and the explicitly-named-file path.
func seeds_failure() (seeds [][]byte) {
	seeds = append(seeds, with(func(one *scenario) { one.Stat_Error = true }))
	seeds = append(seeds, with(func(one *scenario) { one.Bad_Argument = true }))
	// The unrecognized tree holds a name that is one long dotted suffix, which only a
	// run counting hidden entries reaches.
	seeds = append(seeds, with(func(one *scenario) {
		one.Recipe = 11
		one.Hidden = true
	}))
	seeds = append(seeds, with(func(one *scenario) {
		one.Bad_Argument = true
		one.Json = true
	}))
	// A path count at its own bound, which is the command line less the program name.
	seeds = append(seeds, with(func(one *scenario) {
		one.Argument_Pad = sloc.PATHS_COUNT_MAX
	}))
	// The explicitly-named path is the only site that sees a one-byte file path, so
	// its width boundaries are seeded here rather than left to the walk.
	for _, width := range []int{1, 2, 3, sloc.FILE_PATH_BYTES_MAX} {
		named := width
		seeds = append(seeds, with(func(one *scenario) {
			one.Explicit = true
			one.Root_Bytes = named
		}))
	}
	seeds = append(seeds, with(func(one *scenario) {
		one.Explicit = true
		one.Root_Bytes = 7
	}))
	seeds = append(seeds, with(func(one *scenario) {
		one.Explicit = true
		one.Read_Error = true
		one.Root_Bytes = 7
	}))
	seeds = append(seeds, with(func(one *scenario) {
		one.Explicit = true
		one.Json = true
		one.Root_Bytes = 7
	}))
	return seeds
}

// The worker bound, which the library clamps at both ends.
func seeds_concurrency() (seeds [][]byte) {
	// The field carries the choice index here, not the worker count itself; drive
	// resolves it so the int64 boundaries are named rather than decoded.
	for choice := 0; choice <= CONCURRENCY_CHOICE_MAX; choice++ {
		picked := choice
		seeds = append(seeds, with(func(one *scenario) {
			one.Recipe = 6
			one.File_Count = 8
			one.Concurrency = picked
		}))
	}
	return seeds
}

// Every extension the language table recognizes, so the sweep reaches each constructor.
func recognized_extensions() (extensions []string) {
	return []string{
		".ada", ".adb", ".ads", ".astro", ".bash", ".c", ".cc", ".cjs", ".cl", ".clj",
		".cljc", ".cljs", ".cls", ".cmake", ".comp", ".cpp", ".cr", ".cs", ".css",
		".cxx", ".d", ".dart", ".dockerfile", ".dpr", ".edn", ".el", ".erl", ".ex",
		".exs", ".f", ".f03", ".f08", ".f90", ".f95", ".fish", ".for", ".frag", ".fs",
		".fsi", ".fsx", ".geom", ".glsl", ".go", ".gradle", ".groovy", ".h", ".hcl",
		".hh", ".hlsl", ".hpp", ".hrl", ".hs", ".htm", ".html", ".hxx", ".ino",
		".java", ".jl", ".js", ".json5", ".jsonc", ".jsx", ".kt", ".kts", ".less",
		".lhs", ".lisp", ".lsp", ".ltx", ".lua", ".m", ".markdown", ".md", ".mjs",
		".mk", ".ml", ".mli", ".mm", ".nim", ".nims", ".nix", ".nu", ".odin", ".pas",
		".php", ".phtml", ".pl", ".pm", ".pod", ".pp", ".proto", ".ps1", ".psd1",
		".psm1", ".py", ".r", ".R", ".rb", ".rkt", ".rs", ".sc", ".scala", ".scm",
		".scss", ".sh", ".sol", ".sql", ".ss", ".sty", ".sv", ".svelte", ".svg",
		".svh", ".swift", ".t", ".tcl", ".tex", ".tf", ".tfvars", ".thrift", ".toml",
		".ts", ".tsx", ".v", ".vb", ".vert", ".vue", ".xaml", ".xml", ".xsl", ".xslt",
		".yaml", ".yml", ".zig", ".zsh",
	}
}
