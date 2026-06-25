package disk_usage_test

import (
	"encoding/json"
	"io/fs"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	disk_usage "github.com/james-orcales/james-orcales/disk_usage/internal"
)

// Test_Analyze_Cumulative_Sizes verifies a directory's bytes are the sum of every file
// beneath it and a file's bytes are its own, with the root holding the grand total.
func Test_Analyze_Cumulative_Sizes(t *testing.T) {
	report := analyze(t, "root", tree(map[string]string{
		"a/one.txt":     "12345",   // 5 bytes
		"a/b/two.txt":   "123",     // 3 bytes
		"a/b/three.txt": "1234567", // 7 bytes
		"c/four.txt":    "12",      // 2 bytes
	}))

	cases := []struct {
		Path  string
		Bytes int64
	}{
		{".", 17},   // every file
		{"a", 15},   // 5 + 3 + 7
		{"a/b", 10}, // 3 + 7
		{"c", 2},    // 2
		{"a/one.txt", 5},
	}
	for _, want := range cases {
		entry := entry_for(t, report, want.Path)
		if entry.Bytes != want.Bytes {
			t.Errorf("%s: bytes = %d, want %d", want.Path, entry.Bytes, want.Bytes)
		}
	}
}

// Test_Analyze_Empty_Directories verifies a directory holding no file is still recorded,
// with zero bytes, so the model stays complete even though Render later hides it.
func Test_Analyze_Empty_Directories(t *testing.T) {
	disk := tree(map[string]string{"keep/file.txt": "x"})
	// MapFS materializes a directory only when it holds an entry, so seed a zero-byte
	// child to make "empty" a real directory that contributes no bytes.
	disk["empty/.gitkeep"] = &fstest.MapFile{Data: nil}

	report := analyze(t, "root", disk)
	entry := entry_for(t, report, "empty")
	if !entry.Is_Directory {
		t.Errorf("empty: Is_Directory = false, want true")
	}
	if entry.Bytes != 0 {
		t.Errorf("empty: bytes = %d, want 0", entry.Bytes)
	}
}

// Test_Analyze_Depth verifies the root counts as depth one and depth grows by one per
// path segment, so a path's depth is the number of components in its displayed label.
func Test_Analyze_Depth(t *testing.T) {
	report := analyze(t, "root", tree(map[string]string{
		"a/b/c.txt": "x",
	}))
	cases := map[string]int{".": 1, "a": 2, "a/b": 3, "a/b/c.txt": 4}
	for path, depth := range cases {
		entry := entry_for(t, report, path)
		if entry.Depth != depth {
			t.Errorf("%s: depth = %d, want %d", path, entry.Depth, depth)
		}
	}
}

// Test_Analyze_Order verifies directories sort ahead of every file, and within each
// kind entries are largest-first, so folder totals headline the listing.
func Test_Analyze_Order(t *testing.T) {
	report := analyze(t, "root", tree(map[string]string{
		"big-dir/a.bin": "1234567890", // 10 bytes
		"big-dir/b.bin": "1234567890", // 10 bytes
		"small.txt":     "12345",      // 5 bytes
		"tiny.txt":      "1",          // 1 byte
	}))

	// No directory may appear after a file has been seen.
	seen_file := false
	for _, entry := range report.Entries {
		if !entry.Is_Directory {
			seen_file = true
			continue
		}
		if seen_file {
			t.Errorf("directory %q sorted after a file", entry.Path)
		}
	}

	// Files descend by size among themselves.
	files := []disk_usage.Entry{}
	for _, entry := range report.Entries {
		if !entry.Is_Directory {
			files = append(files, entry)
		}
	}
	for index := 1; index < len(files); index++ {
		if files[index-1].Bytes < files[index].Bytes {
			t.Errorf("files not largest-first at %q", files[index].Path)
		}
	}
}

// Test_Analyze_Hardlinks verifies a file's bytes are charged once per identity: a second
// hardlink to the same inode adds nothing to the totals.
func Test_Analyze_Hardlinks(t *testing.T) {
	disk := tree(map[string]string{
		"a/original.bin": "12345", // 5 bytes
		"b/link.bin":     "12345", // a hardlink to the same inode
	})
	report, err := disk_usage.Analyze(disk_usage.Analyze_Input{
		Root:        "root",
		File_System: disk,
		Disk_Usage:  hardlink_usage,
	})
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	// The walk reaches a/original.bin first, so it carries the 5 bytes; b/link.bin and
	// its directory add nothing, and the root counts the inode a single time.
	if got := entry_for(t, report, ".").Bytes; got != 5 {
		t.Errorf("root total = %d, want 5 (inode counted once)", got)
	}
	if got := entry_for(t, report, "a").Bytes; got != 5 {
		t.Errorf("a = %d, want 5 (first path charged)", got)
	}
	if got := entry_for(t, report, "b").Bytes; got != 0 {
		t.Errorf("b = %d, want 0 (hardlink adds nothing)", got)
	}
}

// Test_Render_Depth_Limit verifies Depth_Max hides entries deeper than the limit while
// the sizes shown still account for the hidden descendants.
func Test_Render_Depth_Limit(t *testing.T) {
	report := analyze(t, "root", tree(map[string]string{
		"a/deep/file.txt": "12345",
	}))
	output := strings.Builder{}
	// "a" is depth two and "a/deep/file.txt" is depth four, so a limit of two keeps the
	// directory and hides the file beneath it.
	disk_usage.Render(&output, disk_usage.Render_Input{Report: report, Depth_Max: 2})
	text := output.String()

	if !strings.Contains(text, "a") {
		t.Errorf("depth-2 entry missing:\n%s", text)
	}
	if strings.Contains(text, "file.txt") {
		t.Errorf("depth-4 entry should be hidden:\n%s", text)
	}
}

// Test_Render_Minimum verifies an entry smaller than Minimum is hidden while one at or
// above it still prints — and that a hidden entry is still counted in its parent's total,
// since Minimum only filters the display, not the sums.
func Test_Render_Minimum(t *testing.T) {
	report := analyze(t, "root", tree(map[string]string{
		"big.txt":   strings.Repeat("x", 200),
		"small.txt": strings.Repeat("x", 50),
	}))
	output := strings.Builder{}
	disk_usage.Render(&output, disk_usage.Render_Input{
		Report: report, Depth_Max: -1, Minimum: 100})
	text := output.String()

	if !strings.Contains(text, "big.txt") {
		t.Errorf("entry at or above the minimum was hidden:\n%s", text)
	}
	if strings.Contains(text, "small.txt") {
		t.Errorf("entry below the minimum was shown:\n%s", text)
	}
	// The root still totals 250 B: the hidden 50-byte file is summed, only not shown.
	if !strings.Contains(text, "250 B") {
		t.Errorf("hidden entry was dropped from the parent total:\n%s", text)
	}
}

// Test_Render_Kind verifies Directory_Only prints only directories and Files_Only prints
// only files, dropping the root directory along with the rest.
func Test_Render_Kind(t *testing.T) {
	report := analyze(t, "root", tree(map[string]string{
		"a/file.txt": "12345",
	}))

	directories := strings.Builder{}
	disk_usage.Render(&directories, disk_usage.Render_Input{
		Report: report, Depth_Max: -1, Directory_Only: true})
	if strings.Contains(directories.String(), "file.txt") {
		t.Errorf("dir-only printed a file:\n%s", directories.String())
	}
	if !strings.Contains(directories.String(), "a") {
		t.Errorf("dir-only omitted a directory:\n%s", directories.String())
	}

	files := strings.Builder{}
	disk_usage.Render(&files, disk_usage.Render_Input{
		Report: report, Depth_Max: -1, Files_Only: true})
	if !strings.Contains(files.String(), "file.txt") {
		t.Errorf("files-only omitted a file:\n%s", files.String())
	}
	for _, label := range render_labels(files.String()) {
		if label == "root" {
			t.Errorf("files-only printed the root directory:\n%s", files.String())
		}
	}
}

// Test_Render_Label verifies the root entry shows the root label while a nested entry
// shows the root label joined to its relative path.
func Test_Render_Label(t *testing.T) {
	report := analyze(t, "project", tree(map[string]string{
		"a/file.txt": "12345",
	}))
	output := strings.Builder{}
	disk_usage.Render(&output, disk_usage.Render_Input{Report: report, Depth_Max: -1})
	labels := render_labels(output.String())

	if !slices.Contains(labels, "project") {
		t.Errorf("root label missing: %v", labels)
	}
	if !slices.Contains(labels, "project/a/file.txt") {
		t.Errorf("nested label not joined to root: %v", labels)
	}
}

// Test_Render_Grouping verifies rows are grouped by depth under an ascending per-depth
// header, with each entry sitting beneath the header for its level.
func Test_Render_Grouping(t *testing.T) {
	report := analyze(t, "root", tree(map[string]string{
		"a/file.txt": "12345",
	}))
	output := strings.Builder{}
	disk_usage.Render(&output, disk_usage.Render_Input{Report: report, Depth_Max: -1})
	text := output.String()

	first_offset := strings.Index(text, "Depth 1")
	second_offset := strings.Index(text, "Depth 2")
	third_offset := strings.Index(text, "Depth 3")
	if first_offset < 0 {
		t.Fatalf("missing Depth 1 header:\n%s", text)
	}
	if second_offset < 0 {
		t.Fatalf("missing Depth 2 header:\n%s", text)
	}
	if third_offset < 0 {
		t.Fatalf("missing Depth 3 header:\n%s", text)
	}
	if first_offset >= second_offset {
		t.Errorf("Depth 1 header not before Depth 2:\n%s", text)
	}
	if second_offset >= third_offset {
		t.Errorf("Depth 2 header not before Depth 3:\n%s", text)
	}

	// The root row is the only depth-1 entry, so it must sit between its header and the
	// next one.
	root_offset := strings.Index(text, "root")
	if root_offset <= first_offset {
		t.Errorf("root row not under its Depth 1 header:\n%s", text)
	}
	if root_offset >= second_offset {
		t.Errorf("root row leaked past the Depth 2 header:\n%s", text)
	}
}

// Test_Render_Json verifies the JSON form is a top-level array of flat objects keyed by
// path, bytes, kind, and depth, and that the display filters still apply.
func Test_Render_Json(t *testing.T) {
	report := analyze(t, "root", tree(map[string]string{
		"a/file.txt": "12345",
	}))

	output := strings.Builder{}
	json_err := disk_usage.Render_Json(&output,
		disk_usage.Render_Input{Report: report, Depth_Max: -1})
	if json_err != nil {
		t.Fatalf("Render_Json: %v", json_err)
	}

	entries := decode_entries(t, output.String())
	// The root, "a", and the file all survive the unfiltered render.
	if len(entries) != 3 {
		t.Fatalf("got %d entries, want 3:\n%s", len(entries), output.String())
	}
	for _, key := range []string{"path", "bytes", "is_directory", "depth"} {
		if _, ok := entries[0][key]; !ok {
			t.Errorf("missing %q key: %v", key, entries[0])
		}
	}

	// Files_Only must drop the directory entries from the JSON too.
	files := strings.Builder{}
	files_err := disk_usage.Render_Json(&files,
		disk_usage.Render_Input{Report: report, Depth_Max: -1, Files_Only: true})
	if files_err != nil {
		t.Fatalf("Render_Json files-only: %v", files_err)
	}
	for _, entry := range decode_entries(t, files.String()) {
		if entry["is_directory"] == true {
			t.Errorf("files-only JSON included a directory: %v", entry)
		}
	}
}

// Test_Main_Filters verifies -dir-only and -files-only are contradictory and fail as a
// usage error rather than silently preferring one.
func Test_Main_Filters(t *testing.T) {
	_, stderr, code := run_main(
		[]string{"disk_usage", "project", "-dir-only", "-files-only"},
		tree(map[string]string{"a/file.txt": "12345"}),
	)
	if code != 2 {
		t.Fatalf("exit = %d, want 2 (usage)", code)
	}
	if stderr == "" {
		t.Errorf("expected a usage message on stderr")
	}
}

// Analyzes a tree with a logical, identity-free disk-usage function: each file's size is
// its content length and no two files share an identity, so the dedup path stays inert
// and the tests exercise the accounting rather than the host stat.
func analyze(t *testing.T, root string, disk fstest.MapFS) (report disk_usage.Report) {
	t.Helper()
	result, err := disk_usage.Analyze(disk_usage.Analyze_Input{
		Root:        root,
		File_System: disk,
		Disk_Usage:  logical_usage,
	})
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	return result
}

// Reports a MapFS file's logical size as its disk usage, with the zero File_Identifier
// so the library never merges entries — the host stat that would supply a real identity
// is absent in tests.
func logical_usage(info fs.FileInfo) (bytes int64, identifier disk_usage.File_Identifier) {
	return info.Size(), disk_usage.File_Identifier{}
}

// Reports every file at one fixed identity, modelling separate paths as hardlinks to a
// single inode so the dedup path is exercised.
func hardlink_usage(info fs.FileInfo) (bytes int64, identifier disk_usage.File_Identifier) {
	return info.Size(), disk_usage.File_Identifier{Device: 1, Inode: 99}
}

// Builds an in-memory tree whose file contents stand in for byte sizes, so a file's
// size is exactly the length of the string given here.
func tree(files map[string]string) (file_system fstest.MapFS) {
	file_system = fstest.MapFS{}
	for name, content := range files {
		file_system[name] = &fstest.MapFile{Data: []byte(content)}
	}
	return file_system
}

// Returns the analyzed entry for a path, failing the test when it is absent.
func entry_for(t *testing.T, report disk_usage.Report, name string) (entry disk_usage.Entry) {
	t.Helper()
	for _, candidate := range report.Entries {
		if candidate.Path == name {
			return candidate
		}
	}
	t.Fatalf("no entry for %q in %v", name, report.Entries)
	return disk_usage.Entry{}
}

// Parses JSON output into a slice of objects, failing the test when it is not valid
// JSON.
func decode_entries(t *testing.T, text string) (entries []map[string]any) {
	t.Helper()
	if err := json.Unmarshal([]byte(text), &entries); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, text)
	}
	return entries
}

// Extracts the path label from each rendered row: it is the last whitespace-separated
// field, the size column being the columns before it.
func render_labels(text string) (labels []string) {
	for _, line := range strings.Split(strings.TrimSpace(text), "\n") {
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		labels = append(labels, fields[len(fields)-1])
	}
	return labels
}
