// Package simulation_test calls fuzzstall.Main over a seed-generated run and asserts the
// stall judgment holds for any seed: the run's output reaches the caller byte for byte
// whatever the chunking, and the run is terminated exactly when a stretch as long as the
// window carries no new interesting input.
package simulation_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	fuzzstall "local/james-orcales/fuzzstall/internal"
	invariant "local/james-orcales/shared/invariant/default"
	"local/james-orcales/shared/random/prng"
	systime "local/james-orcales/shared/simulation/time"
)

// TestMain registers the complete fuzzstall invariant tree before the seed sweep starts.
func TestMain(m *testing.M) {
	invariant.Run_Test_Main(m, "../**")
}

// HARNESS_CORPUS is how many seeds one sweep walks. Coverage comes from the sweep and never
// from one run, because each run holds one feature subset for its whole timeline.
const HARNESS_CORPUS = 256

// HARNESS_LINE_COUNT_MAX bounds the lines one generated run writes, which keeps the sweep
// inside the ten seconds a package's tests are given.
const HARNESS_LINE_COUNT_MAX = 12

// HARNESS_BASELINE_COUNT_MAX bounds the baseline lines a run writes before it fuzzes.
const HARNESS_BASELINE_COUNT_MAX = 3

// HARNESS_INTERVAL_SECOND_MAX is the longest gap the generator leaves between two lines. It
// reaches past HARNESS_WINDOW_SECOND_MAX so a seed can stall a run by its spacing alone.
const HARNESS_INTERVAL_SECOND_MAX = 9

// HARNESS_WINDOW_SECOND_MAX is the longest window a generated command line asks for.
const HARNESS_WINDOW_SECOND_MAX = 7

// HARNESS_TAIL_SECOND_MAX is how long a run may stay silent after its last line before it
// exits, which is the stretch that saturates a run whose lines were otherwise close together.
const HARNESS_TAIL_SECOND_MAX = 9

// HARNESS_CHUNK_SIZE_MAX is the widest cut the generator makes in the run's output. It is far
// below one line, so a seed puts a chunk boundary inside a progress line and its count.
const HARNESS_CHUNK_SIZE_MAX = 24

// HARNESS_STATUS_CODE_MAX bounds the exit code a run that ended by itself reports.
const HARNESS_STATUS_CODE_MAX = 4

// HARNESS_RISE_PERCENT is how often a progress line reports a higher count than the one
// before it, for a run whose configuration lets the count rise at all.
const HARNESS_RISE_PERCENT = 50

// HARNESS_FEATURE_PERCENT is how often one optional feature is active in a run. A subset makes
// nothing new possible: it only stops the active features from taking depth from each other.
const HARNESS_FEATURE_PERCENT = 50

// HARNESS_REFUSAL_PERCENT is how often a run is refused before it starts. It is low because a
// refused run exercises no timeline, and the timeline is what this sweep is for.
const HARNESS_REFUSAL_PERCENT = 10

// HARNESS_PERCENT is the denominator every rate above is written over.
const HARNESS_PERCENT = 100

// HARNESS_MALFORMED_PERCENT is how often a run carries a rate this program refuses. It is
// above the rate a run fails to start at, because each refused shape walks a text domain to
// one of its bounds and there are nine of them to reach.
const HARNESS_MALFORMED_PERCENT = 25

// HARNESS_WIDE_CHUNK_COUNT is how many opening cuts take the run's whole read width: one that
// ends on a newline, one that carries none, and one that opens on a newline and so leaves the
// widest remainder a cut can carry.
const HARNESS_WIDE_CHUNK_COUNT = 3

// BASELINE is the line a run writes while it gathers baseline coverage, which carries no count
// and so must never start the window.
const BASELINE = "fuzz: elapsed: 0s, gathering baseline coverage: 3/38 completed\n"

// PROGRESS is the progress line a run writes, with its elapsed span and its count.
const PROGRESS = "fuzz: elapsed: %ds, execs: %d (33/sec), new interesting: %d (total: %d)\n"

// NOISE is a line a run writes that carries no count at all.
const NOISE = "--- FAIL: Fuzz_Parse (3.02s)\n"

// OVERFLOW_SIZE is the width of the one line a run may write that outgrows the assembler's
// buffer, so the sweep reaches the truncation path.
const OVERFLOW_SIZE = fuzzstall.CHUNK_SIZE_MAX + fuzzstall.LINE_SIZE_MAX

// MALFORMED is the set of windows that name no span, one of which a refused run carries.
const MALFORMED = "5h"

// Harness_Streams holds one generator for each axis the model draws from. Each axis gets its
// own stream, thus a draw this harness adds to one axis moves no other axis and every seed
// banked against the others still reproduces.
type Harness_Streams struct {
	// Configuration draws which features the run holds for its whole timeline.
	Configuration prng.Generator
	// Shape draws how many lines of each kind the run writes.
	Shape prng.Generator
	// Window draws the span the command line asks for.
	Window prng.Generator
	// Interval draws the gap between two lines.
	Interval prng.Generator
	// Rise draws whether a progress line reports a higher count.
	Rise prng.Generator
	// Noise draws where a line carrying no count lands.
	Noise prng.Generator
	// Chunk draws where the output is cut.
	Chunk prng.Generator
	// Tail draws how long the run stays silent before it exits.
	Tail prng.Generator
	// Status draws the exit code a run that ended by itself reports.
	Status prng.Generator
}

// Harness_Configuration is the feature subset one seed holds for its whole run.
type Harness_Configuration struct {
	// Baseline activates the baseline lines a run writes before it fuzzes.
	Baseline bool
	// Noise activates the lines that carry no count.
	Noise bool
	// Overflow activates one line that outgrows the assembler's buffer.
	Overflow bool
	// Rises lets the count rise at all. A run without it stalls by construction.
	Rises bool
	// Steady raises the count on every line rather than on some of them, so the record
	// walks the low counts one at a time instead of stepping over them.
	Steady bool
	// Malformed makes the command line carry a window that names no span.
	Malformed bool
	// Unstarted makes the run fail to start at all.
	Unstarted bool
	// Blank activates the lines of no, one, and two bytes, which walk a line to its
	// shortest shapes.
	Blank bool
	// Wide makes one progress line report the largest count a progress line spells.
	Wide bool
	// Window is the span the command line asks for.
	Window fuzzstall.Window
	// Quota is how many new interesting inputs that window must carry.
	Quota fuzzstall.Quota
	// Rise_Step is how far a progress line moves the count. It is drawn against the quota,
	// so a run reaches its rate, holds just under it, or passes it by a wide margin.
	Rise_Step fuzzstall.Interesting_Count
	// Rate_Text is the rate as the command line spells it, which for a malformed run is a
	// text of one of the widths an argument can carry.
	Rate_Text string
	// Epoch is where this run's clock starts. It is drawn rather than fixed because the
	// judgment reads clock differences, and a run that always starts at zero leaves the
	// far end of a reading's domain unvisited.
	Epoch fuzzstall.Reading
	// Argument_Count is how many arguments the command line carries.
	Argument_Count int
	// Chunk_Size is the widest cut this run makes in its output.
	Chunk_Size int
	// Tail is how long the run stays silent after its last line, counted in windows for the
	// same reason the gaps between lines are.
	Tail fuzzstall.Window
	// Status_Code is what a run that ended by itself reports.
	Status_Code fuzzstall.Status_Code
}

// Harness_Line is one line a generated run writes, and the moment it writes it.
type Harness_Line struct {
	// Text is the line, including its newline.
	Text string
	// Interesting_Count is the count the line reports, meaningful only when Counted.
	Interesting_Count fuzzstall.Interesting_Count
	// Counted reports whether the line carries a count at all.
	Counted bool
	// At is the moment the run writes the line.
	At fuzzstall.Reading
}

// Harness_Model is one generated run: the command line that watches it, the chunks its output
// arrives in, when each chunk and each counted line lands, and when the run ends.
type Harness_Model struct {
	// Configuration is the feature subset this run holds.
	Configuration Harness_Configuration
	// Command_Line is what the caller typed to watch this run.
	Command_Line fuzzstall.Command_Line
	// Window is the span the command line asks for.
	Window fuzzstall.Window
	// Quota is how many new interesting inputs that window must carry.
	Quota fuzzstall.Quota
	// Output is every byte the run writes, in order.
	Output string
	// Chunks are those bytes as the run hands them over.
	Chunks []fuzzstall.Chunk
	// Chunk_Moments is when each chunk lands, one for each chunk.
	Chunk_Moments []fuzzstall.Reading
	// Progress_Moments is when each counted line is observed, which is the moment of the
	// chunk that closed it rather than the moment the run wrote it.
	Progress_Moments []fuzzstall.Reading
	// Progress_Counts is the count each counted line reports, paired with the moments.
	Progress_Counts []fuzzstall.Interesting_Count
	// End_At is when the run exits.
	End_At fuzzstall.Reading
	// Epoch is the first reading this run's clock gives. The drive starts there rather than
	// at zero, because a clock that started behind the run's own first line would step one
	// window at a time across the whole gap before the run had written anything.
	Epoch fuzzstall.Reading
}

// Harness_Run is the state one drive of Main moves through.
type Harness_Run struct {
	// Model is the run being replayed.
	Model Harness_Model
	// Index is how many chunks have been handed over.
	Index int
	// Now is the clock the drive reads.
	Now fuzzstall.Reading
	// Terminated records that the drive killed the run.
	Terminated bool
	// Delivered is every byte handed over, which the echo must match.
	Delivered strings.Builder
}

// Harness_Outcome is everything one drive of Main produced.
type Harness_Outcome struct {
	// Status_Code is what Main returned.
	Status_Code fuzzstall.Status_Code
	// Output is everything Main echoed.
	Output string
	// Problems is everything Main wrote to standard error.
	Problems string
	// Terminated reports whether Main killed the run.
	Terminated bool
	// Delivered is every byte the run handed over.
	Delivered string
}

// Forks one stream for each axis, in a fixed order. A stream added before an existing one
// moves every seed behind it, thus a new axis goes at the end.
func new_harness_streams(seed uint64) (streams Harness_Streams) {
	root := prng.New(seed)
	streams.Configuration = prng.Generator_Split(&root)
	streams.Shape = prng.Generator_Split(&root)
	streams.Window = prng.Generator_Split(&root)
	streams.Interval = prng.Generator_Split(&root)
	streams.Rise = prng.Generator_Split(&root)
	streams.Noise = prng.Generator_Split(&root)
	streams.Chunk = prng.Generator_Split(&root)
	streams.Tail = prng.Generator_Split(&root)
	streams.Status = prng.Generator_Split(&root)
	return streams
}

// Draws the feature subset one run holds for its whole timeline.
func new_harness_configuration(streams *Harness_Streams) (
	configuration Harness_Configuration,
) {
	configuration.Baseline = harness_chance(&streams.Configuration, HARNESS_FEATURE_PERCENT)
	configuration.Noise = harness_chance(&streams.Configuration, HARNESS_FEATURE_PERCENT)
	configuration.Overflow = harness_chance(&streams.Configuration, HARNESS_FEATURE_PERCENT)
	configuration.Rises = harness_chance(&streams.Configuration, HARNESS_FEATURE_PERCENT)
	configuration.Malformed = harness_chance(
		&streams.Configuration, HARNESS_MALFORMED_PERCENT)
	configuration.Unstarted = harness_chance(
		&streams.Configuration, HARNESS_REFUSAL_PERCENT)
	configuration.Steady = harness_chance(&streams.Configuration, HARNESS_FEATURE_PERCENT)
	if configuration.Steady {
		// A steady run raises the count on every line, which is what walks the low counts
		// one at a time rather than stepping over them.
		configuration.Rises = true
	}
	if configuration.Wide {
		// A wide run must raise on every line, or its last line lands short of the largest
		// count a progress line spells and that bound goes unvisited.
		configuration.Rises = true
		configuration.Steady = true
	}
	configuration.Blank = harness_chance(&streams.Configuration, HARNESS_FEATURE_PERCENT)
	configuration.Wide = harness_chance(&streams.Configuration, HARNESS_FEATURE_PERCENT)
	configuration.Window = prng.Generator_Element(&streams.Window, harness_windows())
	configuration.Quota = prng.Generator_Element(&streams.Window, []fuzzstall.Quota{
		fuzzstall.QUOTA_MIN, 2, 3, 90, fuzzstall.QUOTA_MAX,
	})
	// Drawn against the quota, so a run reaches its rate exactly, holds one input under it,
	// or passes it by a wide margin. A step fixed at one would leave every quota above one
	// unreachable, and the sweep would only ever watch a run saturate.
	configuration.Rise_Step = fuzzstall.Interesting_Count(configuration.Quota) *
		prng.Generator_Element(&streams.Rise, []fuzzstall.Interesting_Count{0, 1, 2})
	if prng.Generator_Boolean(&streams.Rise) {
		configuration.Rise_Step = fuzzstall.Interesting_Count(configuration.Quota) - 1
	}
	configuration.Rate_Text = harness_rate_text(streams, configuration)
	configuration.Epoch = prng.Generator_Element(&streams.Window, harness_epochs())
	configuration.Argument_Count = prng.Generator_Element(
		&streams.Shape, harness_argument_counts(configuration))
	configuration.Chunk_Size = prng.Generator_Element(
		&streams.Chunk, []int{1, 2, 8, 64, fuzzstall.CHUNK_SIZE_MAX})
	configuration.Tail = configuration.Window * prng.Generator_Element(
		&streams.Tail, []fuzzstall.Window{0, 1, 2, 3})
	configuration.Status_Code = prng.Generator_Element(&streams.Status,
		[]fuzzstall.Status_Code{0, 1, 2, fuzzstall.STATUS_CODE_MAX})
	return configuration
}

// Returns the windows a run may ask for. Both bounds of the domain are in the set, because a
// bound no run reaches is a bound the framework reports as a gap: coverage comes from the
// sweep, never from one run.
func harness_windows() (windows []fuzzstall.Window) {
	return []fuzzstall.Window{
		fuzzstall.WINDOW_MIN,
		fuzzstall.WINDOW_MAX,
		harness_second(2), harness_second(3), harness_second(5), harness_second(7),
		// A window of each width a magnitude takes, so the sweep spells one, two, three,
		// and four digits rather than only the shortest and the longest.
		harness_second(60), harness_second(600),
	}
}

// Returns where a run's clock may start, both bounds of a reading's domain included. The far
// end is the epoch a run whose whole span still lands inside the domain would take, and
// harness_chunk is what holds the last reading there.
func harness_epochs() (epochs []fuzzstall.Reading) {
	return []fuzzstall.Reading{
		fuzzstall.READING_MIN,
		fuzzstall.READING_MIN + 1,
		fuzzstall.READING_MIN + 2,
		fuzzstall.READING_MAX,
	}
}

// Returns how many arguments a command line may carry. A refused run reaches the counts that
// carry no window and no target, and every other run reaches the bound.
func harness_argument_counts(configuration Harness_Configuration) (counts []int) {
	if configuration.Malformed {
		return []int{1, 2, 3, fuzzstall.COMMAND_LINE_COUNT_MAX}
	}
	return []int{3, fuzzstall.COMMAND_LINE_COUNT_MAX}
}

// Returns the rate as the command line spells it. A malformed run spells one of the widths an
// argument carries rather than a rate, which is what walks that text to its bounds.
func harness_rate_text(
	streams *Harness_Streams, configuration Harness_Configuration,
) (text string) {
	if configuration.Malformed {
		return harness_malformed(streams)
	}
	second := int64(configuration.Window) / int64(systime.SECOND)
	if configuration.Quota == fuzzstall.QUOTA_MIN {
		// A bare window and the same window written with its quota name one rate, thus the
		// sweep spells it both ways.
		if prng.Generator_Boolean(&streams.Window) {
			return fmt.Sprintf("%ds", second)
		}
	}
	return fmt.Sprintf("%d/%ds", int64(configuration.Quota), second)
}

// Returns a rate this program refuses. Each shape is refused whatever its width, and together
// they walk the text — and the quota, magnitude, and suffix a split takes out of it — to the
// bounds of each of those domains.
func harness_malformed(streams *Harness_Streams) (text string) {
	return prng.Generator_Element(&streams.Window, []string{
		// Nothing at all, and one byte of each kind: the front of a text's domain.
		"", "z", "9",
		// A two-letter unit, and a magnitude with one behind it.
		"zz", "5zz",
		// A whole text of each kind, which is as wide as a text and a unit both reach.
		strings.Repeat("z", fuzzstall.TEXT_BYTES_MAX),
		strings.Repeat("9", fuzzstall.TEXT_BYTES_MAX),
		// A separator with no quota before it, the front of a quota text's domain.
		"/30s",
		// Every byte but the separator given to the quota, its far end.
		strings.Repeat("9", fuzzstall.QUOTA_TEXT_BYTES_MAX) + "/",
	})
}

// Returns whether one feature is active, at a rate written over HARNESS_PERCENT.
func harness_chance(generator *prng.Generator, percent uint64) (active bool) {
	return prng.Generator_Chance(
		generator, prng.Ratio{Numerator: percent, Denominator: HARNESS_PERCENT})
}

// Returns a span of whole seconds as the moment it lands on off an epoch of zero.
func harness_second(second int) (span fuzzstall.Window) {
	return fuzzstall.Window(systime.Duration(second) * systime.SECOND)
}

// Draws the gap between two lines as a fraction or a multiple of the run's window. The
// judgment turns on the ratio between the gap and the window and never on either one alone,
// thus drawing the gap in windows both reaches every ratio and keeps a run's step count small
// whatever span the window names. A gap drawn in seconds against a one-nanosecond window
// would step the model's clock a nanosecond at a time for a thousand million turns.
func harness_interval(
	streams *Harness_Streams, configuration Harness_Configuration,
) (gap fuzzstall.Window) {
	multiple := prng.Generator_Element(&streams.Interval, []fuzzstall.Window{1, 2, 3})
	if prng.Generator_Boolean(&streams.Interval) {
		// Under one window, so a run of these gaps never saturates between two lines. The
		// two that stop a nanosecond and two nanoseconds short of the window leave a wait
		// of exactly that much behind them, which is the front of a wait's own domain.
		return prng.Generator_Element(&streams.Interval, []fuzzstall.Window{
			configuration.Window/2 + 1,
			configuration.Window - 1,
			configuration.Window - 2,
		})
	}
	return configuration.Window * multiple
}

// Builds one run off its seed: its feature subset, the lines it writes, and the chunks those
// lines arrive in.
func new_harness_model(seed uint64) (model Harness_Model) {
	streams := new_harness_streams(seed)
	configuration := new_harness_configuration(&streams)
	opening := []Harness_Line{}
	if configuration.Overflow {
		// A line exactly as wide as one read, written first, so a chunk cut at the read's
		// own bound holds this line whole and ends on its newline. It is what walks a
		// chunk and a newline offset to the far end of their domains.
		opening = append(opening, Harness_Line{
			Text: strings.Repeat("w", fuzzstall.CHUNK_SIZE_MAX-1) + "\n",
		})
		// A second line one byte wider, so the cut behind the first holds a read's worth of
		// bytes carrying no newline at all — the other far end a read reaches.
		opening = append(opening, Harness_Line{
			Text: strings.Repeat("x", fuzzstall.CHUNK_SIZE_MAX) + "\n",
		})
		// A third of that width, so the cut behind those two opens on the second line's
		// newline and leaves a whole read behind it: the widest remainder a cut can carry.
		opening = append(opening, Harness_Line{
			Text: strings.Repeat("y", fuzzstall.CHUNK_SIZE_MAX) + "\n",
		})
	}
	lines, last_at := harness_baseline(&streams, configuration, opening, 0)
	lines, last_at = harness_progress(&streams, configuration, lines, last_at)
	end_at := last_at + fuzzstall.Reading(configuration.Tail)
	if configuration.Epoch == fuzzstall.READING_MAX {
		// The run ends on its last line, so that line — and every reading taken as it
		// lands — reaches the far end of a reading's domain rather than stopping short of
		// it by the tail and by the gap that follows the line.
		end_at = lines[len(lines)-1].At
	}
	// The run's own span is built off zero and then moved onto its epoch, because an epoch
	// at the far end of a reading's domain must leave the whole span inside that domain.
	epoch := configuration.Epoch
	if epoch == fuzzstall.READING_MAX {
		epoch = fuzzstall.READING_MAX - end_at
	}
	lines = harness_shift(lines, epoch)
	model.Configuration = configuration
	model.Command_Line = harness_command_line(configuration)
	model.Window = configuration.Window
	model.Quota = configuration.Quota
	model.Epoch = epoch
	model.End_At = end_at + epoch
	harness_chunk(&model, &streams, lines)
	return model
}

// Moves every line onto the run's epoch.
func harness_shift(
	lines []Harness_Line, epoch fuzzstall.Reading,
) (moved []Harness_Line) {
	for index := range len(lines) {
		lines[index].At += epoch
	}
	return lines
}

// Returns the command line that watches one run over its window. Its width is drawn, so the
// sweep reaches a line carrying nothing but the program name and a line carrying every
// argument one can hold.
func harness_command_line(
	configuration Harness_Configuration,
) (command_line fuzzstall.Command_Line) {
	command_line = fuzzstall.Command_Line{"fuzzstall"}
	if configuration.Argument_Count < 2 {
		return command_line
	}
	command_line = append(command_line, configuration.Rate_Text)
	for range configuration.Argument_Count - 2 {
		command_line = append(command_line, "./shared/cli")
	}
	return command_line
}

// Appends the baseline lines a run writes before it fuzzes. They carry no count, thus a
// baseline longer than the window must not read as a run that gave up.
func harness_baseline(
	streams *Harness_Streams, configuration Harness_Configuration,
	lines []Harness_Line, at fuzzstall.Reading,
) (written []Harness_Line, moment fuzzstall.Reading) {
	if !configuration.Baseline {
		return lines, at
	}
	baseline_count := 1 + prng.Generator_Below(
		&streams.Shape, HARNESS_BASELINE_COUNT_MAX)
	for range baseline_count {
		// The line lands on the moment the run is at, and the gap follows it. Written the
		// other way the run's first line could never land on the run's own first reading,
		// and the front of a reading's domain would go unvisited.
		lines = append(lines, Harness_Line{Text: BASELINE, At: at})
		at += fuzzstall.Reading(harness_interval(streams, configuration))
	}
	return lines, at
}

// Appends the progress lines a run writes, each one interval behind the last, carrying a count
// that rises only when the configuration lets it. A run whose count cannot rise stalls by
// construction, which is the subset that reaches the judgment this program exists for.
func harness_progress(
	streams *Harness_Streams, configuration Harness_Configuration,
	lines []Harness_Line, at fuzzstall.Reading,
) (written []Harness_Line, moment fuzzstall.Reading) {
	progress_count := 1 + prng.Generator_Below(&streams.Shape, HARNESS_LINE_COUNT_MAX)
	count := harness_count_first(configuration, progress_count)
	for index := range progress_count {
		count += harness_rise(streams, configuration)
		lines = append(lines, Harness_Line{
			Text:              fmt.Sprintf(PROGRESS, index, index, count, count+7),
			Interesting_Count: count,
			Counted:           true,
			At:                at,
		})
		lines = harness_between(streams, configuration, lines, at, index)
		at += fuzzstall.Reading(harness_interval(streams, configuration))
	}
	return lines, at
}

// Returns the count a run's first progress line reports. A wide run starts far enough below
// the largest count a line spells that its last line lands on that bound and no line passes
// it: a count past it carries more digits than the field admits, and a line the parser reads
// as carrying no count is one the model and the program would disagree about.
func harness_count_first(
	configuration Harness_Configuration, progress_count int,
) (count fuzzstall.Interesting_Count) {
	if !configuration.Wide {
		return 0
	}
	return fuzzstall.INTERESTING_COUNT_MAX -
		configuration.Rise_Step*fuzzstall.Interesting_Count(progress_count)
}

// Returns how far one progress line moves the count.
func harness_rise(
	streams *Harness_Streams, configuration Harness_Configuration,
) (rise fuzzstall.Interesting_Count) {
	if !configuration.Rises {
		return 0
	}
	if configuration.Steady {
		return configuration.Rise_Step
	}
	if harness_chance(&streams.Rise, HARNESS_RISE_PERCENT) {
		return configuration.Rise_Step
	}
	return 0
}

// Appends the lines that carry no count beside a progress line: the noise a run writes, and
// the one line that outgrows the assembler's buffer. Both land at the moment beside them, so
// neither moves the timeline the judgment reads.
func harness_between(
	streams *Harness_Streams, configuration Harness_Configuration,
	lines []Harness_Line, at fuzzstall.Reading, index int,
) (written []Harness_Line) {
	if configuration.Noise {
		if harness_chance(&streams.Noise, HARNESS_FEATURE_PERCENT) {
			lines = append(lines, Harness_Line{Text: NOISE, At: at})
		}
	}
	if index != 0 {
		return lines
	}
	if configuration.Blank {
		lines = append(lines, harness_edges(at)...)
	}
	if !configuration.Overflow {
		return lines
	}
	return append(lines, Harness_Line{
		Text: strings.Repeat("x", OVERFLOW_SIZE) + "\n",
		At:   at,
	})
}

// Returns the lines that walk a line and a field offset to their bounds: a line of no, one,
// and two bytes, and the field sitting at the front of a line, one and two bytes into it, and
// as far into the widest line as it still fits. None of them carries a readable count, thus
// none of them moves the judgment the sweep is checking.
func harness_edges(at fuzzstall.Reading) (lines []Harness_Line) {
	field := fuzzstall.INTERESTING_FIELD
	return []Harness_Line{
		{Text: "\n", At: at},
		{Text: "a\n", At: at},
		{Text: "ab\n", At: at},
		{Text: field + "\n", At: at},
		{Text: "p" + field + "\n", At: at},
		{Text: "pp" + field + "\n", At: at},
		{Text: strings.Repeat("p", fuzzstall.FIELD_OFFSET_MAX) + field + "\n", At: at},
		// The field with a tail of one, two, and the widest run of bytes a line leaves
		// behind it. None of the tails is a number, thus each walks the read of a count to
		// a bound without any of them naming a count that would move the judgment.
		{Text: field + "x\n", At: at},
		{Text: field + "xy\n", At: at},
		{Text: field + strings.Repeat("x", fuzzstall.FIELD_OFFSET_MAX) + "\n", At: at},
	}
}

// Cuts the generated lines into seeded chunks and records when each chunk and each counted
// line arrives. A chunk carries the moment of its last byte, which is the moment the watcher
// reads when that chunk lands, thus a line is observed at the moment of the chunk that closed
// it — exactly the moment the program under test reads.
func harness_chunk(model *Harness_Model, streams *Harness_Streams, lines []Harness_Line) {
	text := []byte{}
	byte_moments := []fuzzstall.Reading{}
	newline_offsets := []int{}
	for _, line := range lines {
		for range len(line.Text) {
			byte_moments = append(byte_moments, line.At)
		}
		text = append(text, line.Text...)
		if !line.Counted {
			continue
		}
		newline_offsets = append(newline_offsets, len(text)-1)
		model.Progress_Counts = append(model.Progress_Counts, line.Interesting_Count)
	}
	model.Output = string(text)
	chunk_starts := []int{}
	if model.Configuration.Blank {
		// A read that returns no byte at all, which a pipe gives and which must move
		// nothing: it walks a chunk to the front of its domain.
		chunk_starts = append(chunk_starts, 0)
		model.Chunks = append(model.Chunks, fuzzstall.Chunk{})
		model.Chunk_Moments = append(model.Chunk_Moments, byte_moments[0])
	}
	start_offset := 0
	for start_offset < len(text) {
		widest := model.Configuration.Chunk_Size
		chunk_size := 1 + prng.Generator_Below(&streams.Chunk, widest)
		if len(chunk_starts) < HARNESS_WIDE_CHUNK_COUNT {
			// The opening cuts take the whole read width, so a read that fills its
			// buffer is one the sweep reaches and not one it only might. Three of
			// them: one ends on a newline, one holds none, one opens on one.
			chunk_size = widest
		}
		end_offset := start_offset + chunk_size
		if end_offset > len(text) {
			end_offset = len(text)
		}
		chunk_starts = append(chunk_starts, start_offset)
		model.Chunks = append(
			model.Chunks, fuzzstall.Chunk(text[start_offset:end_offset]))
		model.Chunk_Moments = append(model.Chunk_Moments, byte_moments[end_offset-1])
		start_offset = end_offset
	}
	for _, newline_offset := range newline_offsets {
		model.Progress_Moments = append(model.Progress_Moments,
			model.Chunk_Moments[harness_chunk_index(chunk_starts, newline_offset)])
	}
}

// Returns which chunk holds the byte at offset.
func harness_chunk_index(chunk_starts []int, offset int) (index int) {
	for scan_index := range len(chunk_starts) {
		if chunk_starts[scan_index] > offset {
			return scan_index - 1
		}
	}
	return len(chunk_starts) - 1
}

// Runs Main over one model and returns everything the drive produced.
func harness_drive(model Harness_Model) (outcome Harness_Outcome) {
	run := Harness_Run{Model: model, Now: model.Epoch}
	output := strings.Builder{}
	problems := strings.Builder{}
	status_code := fuzzstall.Main(&fuzzstall.Main_Input{
		Arguments:    model.Command_Line,
		Output:       &output,
		Error_Output: &problems,
		Now:          func() (reading fuzzstall.Reading) { return run.Now },
		Start:        harness_start(&run),
	})
	return Harness_Outcome{
		Status_Code: status_code,
		Output:      output.String(),
		Problems:    problems.String(),
		Terminated:  run.Terminated,
		Delivered:   run.Delivered.String(),
	}
}

// Returns the seam the model enters through. A refused run hands back the handle that is
// already over, which is what a failed start gives a caller.
func harness_start(run *Harness_Run) (start fuzzstall.Start) {
	return func(
		path fuzzstall.Toolchain, arguments fuzzstall.Arguments,
	) (child fuzzstall.Child, err error) {
		if run.Model.Configuration.Unstarted {
			// The handle on a run that never started: over, killing nothing, and
			// reporting this program's own failure.
			return fuzzstall.Child{
				Next_Output: func(
					_ fuzzstall.Deadline,
				) (chunk fuzzstall.Chunk, event fuzzstall.Event) {
					return nil, fuzzstall.EVENT_END
				},
				Terminate: func() {},
				Wait: func() (status_code fuzzstall.Status_Code) {
					return fuzzstall.EXIT_START_FAILURE
				},
			}, errors.New("executable file not found")
		}
		return fuzzstall.Child{
			Next_Output: harness_next_output(run),
			Terminate:   func() { run.Terminated = true },
			Wait: func() (status_code fuzzstall.Status_Code) {
				return run.Model.Configuration.Status_Code
			},
		}, nil
	}
}

// Returns the bounded wait the model answers: the next chunk when it lands inside the
// deadline, the deadline when it does not, and the end once every chunk is over. The deadline
// wins a tie, which is the rule a real bounded read follows.
func harness_next_output(run *Harness_Run) (next fuzzstall.Next_Output) {
	return func(deadline fuzzstall.Deadline) (chunk fuzzstall.Chunk, event fuzzstall.Event) {
		expiry := run.Now + fuzzstall.Reading(deadline)
		if run.Index == len(run.Model.Chunks) {
			if run.Model.End_At >= expiry {
				run.Now = expiry
				return nil, fuzzstall.EVENT_DEADLINE
			}
			run.Now = run.Model.End_At
			return nil, fuzzstall.EVENT_END
		}
		if run.Model.Chunk_Moments[run.Index] >= expiry {
			run.Now = expiry
			return nil, fuzzstall.EVENT_DEADLINE
		}
		chunk = run.Model.Chunks[run.Index]
		run.Now = run.Model.Chunk_Moments[run.Index]
		run.Index++
		run.Delivered.Write(chunk)
		return chunk, fuzzstall.EVENT_OUTPUT
	}
}

// Returns the longest stretch the run leaves between two new interesting inputs, counted from
// its first progress line and closed by the moment the run ends. The judgment must key on
// exactly this, thus the sweep derives it from the model and never from the program.
func harness_stall(model Harness_Model) (stall fuzzstall.Window, started bool) {
	highest := fuzzstall.Interesting_Count(0)
	anchor_count := fuzzstall.Interesting_Count(0)
	anchor_at := fuzzstall.Reading(0)
	for index := range len(model.Progress_Moments) {
		at := model.Progress_Moments[index]
		count := model.Progress_Counts[index]
		if !started {
			started = true
			highest = count
			anchor_count = count
			anchor_at = at
			continue
		}
		if count <= highest {
			continue
		}
		highest = count
		// The window starts again only where the quota is met, thus a run that finds
		// inputs steadily but below its rate leaves one long stretch and not many short
		// ones.
		if fuzzstall.Quota(count-anchor_count) < model.Quota {
			continue
		}
		stall = harness_longer(stall, fuzzstall.Window(at-anchor_at))
		anchor_count = count
		anchor_at = at
	}
	if !started {
		return 0, false
	}
	return harness_longer(stall, fuzzstall.Window(model.End_At-anchor_at)), true
}

// Returns the longer of two spans.
func harness_longer(
	left fuzzstall.Window, right fuzzstall.Window,
) (longer fuzzstall.Window) {
	if right > left {
		return right
	}
	return left
}
