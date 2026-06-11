// Package myers computes character- and line-level diffs between two texts with
// Myers' O(ND) algorithm, fronted by prefix, suffix, and common-run fast paths.
package myers

import (
	"fmt"
	"slices"
	"strings"

	invariant "github.com/james-orcales/james-orcales/shared/invariant/default"
)

// Edit_Retain marks runes present unchanged in both Old and New.
const Edit_Retain uint8 = 10

// Edit_Delete marks runes present only in Old.
const Edit_Delete uint8 = 20

// Edit_Insert marks runes present only in New.
const Edit_Insert uint8 = 30

// Edit is one contiguous run of runes sharing a single kind in the diff script.
type Edit struct {
	// Kind is Edit_Retain, Edit_Delete, or Edit_Insert.
	Kind uint8
	// Data is the runes this edit covers.
	Data []rune
}

// Differ holds the two texts under comparison and the edit script built for
// them. Only Edits is mutated by the diff functions; the text fields are inputs.
type Differ struct {
	// Edits is the diff script the diff functions accumulate.
	Edits []Edit
	// Old is the source text as runes.
	Old []rune
	// New is the target text as runes.
	New []rune
	// Old_String is the source text.
	Old_String string
	// New_String is the target text.
	New_String string
}

// New_Input is the constructor input for New.
type New_Input struct {
	// Old is the source text.
	Old string
	// New is the target text.
	New string
}

// New builds a Differ over the given texts.
func New(input New_Input) (d *Differ) {
	return &Differ{
		Old:        []rune(input.Old),
		New:        []rune(input.New),
		Old_String: input.Old,
		New_String: input.New,
	}
}

// Differ_Reset clears the edits and texts while retaining slice capacity.
func Differ_Reset(d *Differ) {
	d.Edits = d.Edits[:0]
	d.Old, d.New = d.Old[:0], d.New[:0]
	d.Old_String, d.New_String = "", ""
}

// Returns a deferred check asserting that nothing but the Edits field changed
// between this call and the deferred invocation.
func differ_assert_only_edits_mutated(d *Differ) (check func()) {
	before := *d
	return func() {
		// Diffing reads the texts and writes only Edits; the inputs must survive intact.
		invariant.Dot_Product(
			invariant.Always(before.Old_String == d.Old_String),
			invariant.Always(before.New_String == d.New_String),
			invariant.Always(slices.Equal(before.Old, d.Old)),
			invariant.Always(slices.Equal(before.New, d.New)),
		)
	}
}

// Holds the Old and New text reconstructed from a Differ's edits, used by the
// reconstruction invariants.
type differ_rebuilt_text struct {
	// Old is the source text rebuilt from retain and delete edits.
	Old string
	// New is the target text rebuilt from retain and insert edits.
	New string
}

// Replays the edit script to recover the two texts it encodes.
func differ_rebuild_string_from_edits(d *Differ) (text differ_rebuilt_text) {
	var old strings.Builder
	var new strings.Builder
	for _, edit := range d.Edits {
		if edit.Kind == Edit_Retain {
			for _, r := range edit.Data {
				old.WriteRune(r)
				new.WriteRune(r)
			}
		} else if edit.Kind == Edit_Delete {
			for _, r := range edit.Data {
				old.WriteRune(r)
			}
		} else if edit.Kind == Edit_Insert {
			for _, r := range edit.Data {
				new.WriteRune(r)
			}
		}
	}
	return differ_rebuilt_text{Old: old.String(), New: new.String()}
}

// Line-to-rune encoding of a Differ's texts: each distinct line becomes a
// single rune so line diffing reduces to rune diffing.
type differ_line_codes struct {
	// Old is Old_String with each line replaced by its code rune.
	Old string
	// New is New_String with each line replaced by its code rune.
	New string
	// Rune_To_Line recovers the original line for a code rune.
	Rune_To_Line map[rune]string
}

// Maps each distinct line in the Differ's texts to a unique rune, returning the
// rune-encoded texts and the reverse mapping.
func differ_encode_lines(d *Differ) (codes differ_line_codes) {
	n_count := strings.Count(d.Old_String, "\n")
	var old strings.Builder
	var new strings.Builder
	old.Grow(n_count)
	new.Grow(strings.Count(d.New_String, "\n"))

	var ch rune
	line_to_rune := make(map[string]rune, n_count)
	rune_to_line := make(map[rune]string, n_count)
	for line := range strings.SplitSeq(d.Old_String, "\n") {
		if _, ok := line_to_rune[line]; !ok {
			line_to_rune[line] = ch
			rune_to_line[ch] = line
			ch++
		}
		old.WriteRune(line_to_rune[line])
	}
	for line := range strings.SplitSeq(d.New_String, "\n") {
		if _, ok := line_to_rune[line]; !ok {
			line_to_rune[line] = ch
			rune_to_line[ch] = line
			ch++
		}
		new.WriteRune(line_to_rune[line])
	}
	return differ_line_codes{Old: old.String(), New: new.String(), Rune_To_Line: rune_to_line}
}

// Differ_Line_Diff renders a line-granularity diff: each output line is an
// original line prefixed by a space (retained), '+' (inserted), or '-'
// (deleted).
//
// TODO: Concise diffs -> Configurable surrounding line count for each edit.
func Differ_Line_Diff(dfr *Differ) (diff string) {
	defer differ_assert_only_edits_mutated(dfr)()
	if dfr.Old_String == dfr.New_String {
		if dfr.Old_String == "" {
			return ""
		}
		return fmt.Sprintf(" %s", dfr.Old_String)
	}
	if dfr.Old_String == "" {
		return "+" + strings.ReplaceAll(dfr.New_String, "\n", "\n+")
	}
	if dfr.New_String == "" {
		return "-" + strings.ReplaceAll(dfr.Old_String, "\n", "\n-")
	}

	invariant.Dot_Product(invariant.Sometimes(
		strings.LastIndexByte(dfr.Old_String, '\n') != len(dfr.Old_String)-1))
	invariant.Dot_Product(invariant.Sometimes(
		strings.LastIndexByte(dfr.New_String, '\n') != len(dfr.New_String)-1))

	codes := differ_encode_lines(dfr)
	d := New(New_Input{Old: codes.Old, New: codes.New})
	defer func() { dfr.Edits = d.Edits }()
	Differ_Optimized_Diff(d)
	Differ_Merge_Shift_Diff_Cleanup(d)

	result := make([]string, 0, len(d.Edits))
	for _, edit := range d.Edits {
		if len(edit.Data) == 0 {
			continue
		}
		indicator := ""
		switch edit.Kind {
		case Edit_Retain:
			indicator = " "
		case Edit_Insert:
			indicator = "+"
		case Edit_Delete:
			indicator = "-"
		}
		for _, character := range edit.Data {
			result = append(result, indicator+codes.Rune_To_Line[character])
		}
	}
	return strings.Join(result, "\n")
}

// Differ_Diff returns the character-level diff string for the Differ's texts.
func Differ_Diff(d *Differ) (diff string) {
	before := *d
	Differ_Optimized_Diff(d)
	Differ_Merge_Shift_Diff_Cleanup(d)
	// The edits rebuild both texts together — except when an invalid UTF-8 byte
	// decoded to U+FFFD, so the runes can't reproduce the original bytes (the false
	// branch, witnessed by the invalid-UTF-8 input).
	rebuilt := differ_rebuild_string_from_edits(d)
	invariant.Dot_Product(invariant.Sometimes(
		(before.Old_String == rebuilt.Old) == (before.New_String == rebuilt.New)))
	return d.String()
}

// String renders the edit script as kind-prefixed, double-quoted runs.
func (d Differ) String() (s string) {
	var sb strings.Builder
	for _, edit := range d.Edits {
		if len(edit.Data) == 0 {
			continue
		}
		kind := ""
		switch edit.Kind {
		case Edit_Retain:
			kind = " "
		case Edit_Insert:
			kind = "+"
		case Edit_Delete:
			kind = "-"
		}
		sb.WriteString(kind)
		sb.WriteRune('"')
		for _, r := range edit.Data {
			if r == '"' {
				sb.WriteRune('\\')
				sb.WriteRune('"')
			} else {
				sb.WriteRune(r)
			}
		}
		sb.WriteRune('"')
	}
	return sb.String()
}

// Differ_Merge_Shift_Diff_Cleanup coalesces adjacent same-kind runs and shifts
// edit boundaries to align with retains, repeating until no boundary moves.
func Differ_Merge_Shift_Diff_Cleanup(d *Differ) {
	defer differ_assert_only_edits_mutated(d)()
	before := *d
	defer func() {
		// The edits rebuild both texts together, save the invalid-UTF-8 case where a
		// byte decoded to U+FFFD (the false branch).
		rebuilt := differ_rebuild_string_from_edits(d)
		invariant.Dot_Product(invariant.Sometimes(
			(before.Old_String == rebuilt.Old) == (before.New_String == rebuilt.New)))
	}()
	for is_shifted := true; is_shifted; {
		if len(d.Edits) < 3 {
			return
		}
		invariant.Dot_Product(
			invariant.Always(len(d.Old_String) > 0),
			invariant.Always(len(d.New_String) > 0),
			invariant.Always(len(d.Old) > 0),
			invariant.Always(len(d.New) > 0),
		)

		if d.Edits[0].Kind != Edit_Retain {
			d.Edits = slices.Insert(d.Edits, 0, Edit{Kind: Edit_Retain, Data: nil})
		}
		if d.Edits[len(d.Edits)-1].Kind != Edit_Retain {
			d.Edits = append(d.Edits, Edit{Kind: Edit_Retain, Data: nil})
		}

		differ_merge(d)

		// Both ends now carry empty retains (differ_merge padded them), so 3+ edits exist.
		invariant.Dot_Product(invariant.Always(len(d.Edits) >= 3))
		if len(d.Edits[0].Data) == 0 {
			d.Edits = d.Edits[1:]
		}
		if len(d.Edits[len(d.Edits)-1].Data) == 0 {
			d.Edits = d.Edits[:len(d.Edits)-1]
		}
		// After trimming the empty boundary retains, every remaining edit carries data.
		invariant.Dot_Product(invariant.Always(func() (ok bool) {
			for _, edit := range d.Edits {
				if len(edit.Data) == 0 {
					return false
				}
			}
			return true
		}()))
		is_shifted = differ_shift(d)
	}
}

// Rewrites d.Edits so consecutive deletes and inserts are gathered against the
// retains that bound them, lifting any shared prefix or suffix into the
// neighbouring retains.
func differ_merge(d *Differ) {
	result := make([]Edit, 0, len(d.Edits))
	defer func() { d.Edits = result }()

	old, new := d.Old, d.New
	var to_delete, to_insert []rune
	for _, edit := range d.Edits {
		if edit.Kind == Edit_Delete {
			to_delete = old[:len(to_delete)+len(edit.Data)]
			continue
		}
		if edit.Kind == Edit_Insert {
			to_insert = new[:len(to_insert)+len(edit.Data)]
			continue
		}
		current_edit := edit
		has_delete := len(to_delete) > 0
		has_insert := len(to_insert) > 0
		if has_delete {
			if has_insert {
				lifted := differ_merge_lift_affixes(result, differ_affix{
					Current_Edit: current_edit,
					To_Delete:    to_delete,
					To_Insert:    to_insert,
				})
				current_edit = lifted.Current_Edit
				to_delete = lifted.To_Delete
				to_insert = lifted.To_Insert
			}
		}
		if has_delete {
			result = append(result, Edit{Kind: Edit_Delete, Data: to_delete})
			old = old[len(to_delete):]
		}
		if has_insert {
			result = append(result, Edit{Kind: Edit_Insert, Data: to_insert})
			new = new[len(to_insert):]
		}
		result = append(result, current_edit)
		old = old[len(current_edit.Data):]
		new = new[len(current_edit.Data):]
		to_delete = nil
		to_insert = nil
	}
}

// Carries the merge state mutated when an adjacent insert and delete share an
// affix run.
type differ_affix struct {
	// Current_Edit is the bounding retain, possibly extended by a shared suffix.
	Current_Edit Edit
	// To_Delete is the pending deletion remainder after lifting.
	To_Delete []rune
	// To_Insert is the pending insertion remainder after lifting.
	To_Insert []rune
}

// Lifts the run shared at the front of the pending insert and delete into the
// previous retain, and the run shared at the back into the bounding retain.
func differ_merge_lift_affixes(result []Edit, state differ_affix) (lifted differ_affix) {
	prefix := Find_Common_Prefix(
		Find_Common_Prefix_Input{A: state.To_Insert, B: state.To_Delete},
	)
	if len(prefix) > 0 {
		previous_retain := &result[len(result)-1]
		previous_retain.Data = slices.Concat(previous_retain.Data, prefix)
		state.To_Delete = state.To_Delete[len(prefix):]
		state.To_Insert = state.To_Insert[len(prefix):]
	}
	suffix := Find_Common_Suffix(
		Find_Common_Suffix_Input{A: state.To_Insert, B: state.To_Delete},
	)
	if len(suffix) > 0 {
		state.Current_Edit.Data = slices.Concat(state.Current_Edit.Data, suffix)
		state.To_Delete = state.To_Delete[:len(state.To_Delete)-len(suffix)]
		state.To_Insert = state.To_Insert[:len(state.To_Insert)-len(suffix)]
	}
	return state
}

// Moves a delete or insert that ends with its left retain or begins with its
// right retain across that retain, aligning runs. Reports whether any boundary
// moved so the caller can repeat the cleanup.
func differ_shift(d *Differ) (is_shifted bool) {
	result := []Edit{d.Edits[0]}
	defer func() {
		result = append(result, d.Edits[len(d.Edits)-1])
		d.Edits = result
	}()
	for offset, edit := range d.Edits[1 : len(d.Edits)-1] {
		offset++
		previous := &result[len(result)-1]
		next := &d.Edits[offset+1]
		if previous.Kind != Edit_Retain {
			result = append(result, edit)
			continue
		}
		if next.Kind != Edit_Retain {
			result = append(result, edit)
			continue
		}
		// Both neighbours are retains, so an interior edit between them is never a retain.
		invariant.Dot_Product(invariant.Always(edit.Kind != Edit_Retain))
		if Runes_Have_Suffix(
			Runes_Have_Suffix_Input{String: edit.Data, Expect: previous.Data},
		) {
			is_shifted = true
			next.Data = slices.Concat(previous.Data, next.Data)
			head := edit.Data[:len(edit.Data)-len(previous.Data)]
			previous.Data = slices.Concat(previous.Data, head)
			previous.Kind = edit.Kind
			continue
		}
		if Runes_Have_Prefix(
			Runes_Have_Prefix_Input{String: edit.Data, Expect: next.Data},
		) {
			is_shifted = true
			previous.Data = slices.Concat(previous.Data, next.Data)
			next.Data = slices.Concat(edit.Data[len(next.Data):], next.Data)
			next.Kind = edit.Kind
			continue
		}
		result = append(result, edit)
	}
	return is_shifted
}

// Differ_Optimized_Diff peels common prefix and suffix retains, handles simple
// inserts, deletes, and single-sided sandwiches directly, and delegates the
// remainder to a common-run split.
func Differ_Optimized_Diff(d *Differ) {
	defer differ_assert_only_edits_mutated(d)()
	before := *d
	defer func() {
		// The edits rebuild both texts together, save the invalid-UTF-8 case where a
		// byte decoded to U+FFFD (the false branch).
		rebuilt := differ_rebuild_string_from_edits(d)
		invariant.Dot_Product(invariant.Sometimes(
			(before.Old_String == rebuilt.Old) == (before.New_String == rebuilt.New)))
	}()

	old, new := d.Old, d.New
	if d.Old_String == d.New_String {
		d.Edits = append(d.Edits, Edit{Kind: Edit_Retain, Data: old})
		return
	}
	if d.New_String == "" {
		d.Edits = append(d.Edits, Edit{Kind: Edit_Delete, Data: old})
		return
	}
	if d.Old_String == "" {
		d.Edits = append(d.Edits, Edit{Kind: Edit_Insert, Data: new})
		return
	}

	prefix := Find_Common_Prefix(Find_Common_Prefix_Input{A: old, B: new})
	if len(prefix) > 0 {
		d.Edits = append(d.Edits, Edit{Kind: Edit_Retain, Data: prefix})
	}
	old = old[len(prefix):]
	new = new[len(prefix):]

	suffix := Find_Common_Suffix(Find_Common_Suffix_Input{A: old, B: new})
	defer func() {
		if len(suffix) > 0 {
			d.Edits = append(d.Edits, Edit{Kind: Edit_Retain, Data: suffix})
		}
	}()
	old = old[:len(old)-len(suffix)]
	new = new[:len(new)-len(suffix)]

	differ_optimized_core(differ_optimized_core_input{D: d, Old: old, New: new})
}

// Carries the trimmed texts into the optimized core.
type differ_optimized_core_input struct {
	// D is the Differ whose Edits are extended.
	D *Differ
	// Old is the source remainder after affix peeling.
	Old []rune
	// New is the target remainder after affix peeling.
	New []rune
}

// Handles simple inserts, deletes, and one-sided sandwiches, delegating the
// genuinely mixed case to a common-run split.
func differ_optimized_core(input differ_optimized_core_input) {
	d, old, new := input.D, input.Old, input.New
	is_simple_delete := len(old) > 0 && len(new) == 0
	is_simple_insert := len(old) == 0 && len(new) > 0
	if is_simple_delete {
		d.Edits = append(d.Edits, Edit{Kind: Edit_Delete, Data: old})
		return
	}
	if is_simple_insert {
		d.Edits = append(d.Edits, Edit{Kind: Edit_Insert, Data: new})
		return
	}

	x := runes_index(runes_index_input{Haystack: old, Needle: new})
	y := runes_index(runes_index_input{Haystack: new, Needle: old})
	is_delete_sandwich := x > 0
	if is_delete_sandwich {
		d.Edits = append(d.Edits, Edit{Kind: Edit_Delete, Data: old[:x]})
		old = old[x:]
		d.Edits = append(d.Edits, Edit{Kind: Edit_Retain, Data: old[:len(new)]})
		old = old[len(new):]
		d.Edits = append(d.Edits, Edit{Kind: Edit_Delete, Data: old})
		return
	}
	is_insert_sandwich := y > 0
	if is_insert_sandwich {
		d.Edits = append(d.Edits, Edit{Kind: Edit_Insert, Data: new[:y]})
		new = new[y:]
		d.Edits = append(d.Edits, Edit{Kind: Edit_Retain, Data: new[:len(old)]})
		new = new[len(old):]
		d.Edits = append(d.Edits, Edit{Kind: Edit_Insert, Data: new})
		return
	}

	inner := *d
	inner.Old = old
	inner.New = new
	differ_optimized_split(&inner)
	d.Edits = inner.Edits
}

// Recursively divides the texts on their longest common run, falling back to
// the Myers algorithm where no qualifying run exists.
func differ_optimized_split(d *Differ) {
	var recurse func(diff *Differ)
	recurse = func(diff *Differ) {
		old_runes, new_runes := diff.Old, diff.New
		run := Find_Common_Run(Find_Common_Run_Input{A: old_runes, B: new_runes})
		if len(run) == 0 {
			Differ_Algorithm_Diff(diff)
			return
		}
		new_run_start := runes_index(runes_index_input{Haystack: new_runes, Needle: run})
		old_run_start := runes_index(runes_index_input{Haystack: old_runes, Needle: run})
		{
			clone := Differ{
				Edits:      diff.Edits,
				Old:        old_runes[:old_run_start],
				New:        new_runes[:new_run_start],
				Old_String: diff.Old_String,
				New_String: diff.New_String,
			}
			recurse(&clone)
			diff.Edits = clone.Edits
		}
		diff.Edits = append(diff.Edits, Edit{Kind: Edit_Retain, Data: run})
		{
			clone := Differ{
				Edits:      diff.Edits,
				Old:        old_runes[old_run_start+len(run):],
				New:        new_runes[new_run_start+len(run):],
				Old_String: diff.Old_String,
				New_String: diff.New_String,
			}
			recurse(&clone)
			diff.Edits = clone.Edits
		}
	}
	recurse(d)
}

// Differ_Algorithm_Diff produces a minimal edit script with Myers' O(ND)
// algorithm: a forward furthest-reaching trace followed by a backtrack.
func Differ_Algorithm_Diff(d *Differ) {
	defer differ_assert_only_edits_mutated(d)()
	before := *d
	defer func() {
		// The runes equal []rune of the text for valid UTF-8, but an invalid byte
		// decodes to U+FFFD, so the runes no longer reproduce the original bytes. The
		// false branch is witnessed by the invalid-UTF-8 input, the true branch by the
		// rest.
		condition := (string(before.Old) == before.Old_String) ==
			(string(before.New) == before.New_String)
		invariant.Dot_Product(invariant.Sometimes(condition))
		if condition {
			// With the runes equal to the text, the script must replay back to it.
			rebuilt := differ_rebuild_string_from_edits(d)
			invariant.Dot_Product(invariant.Always(
				(before.Old_String == rebuilt.Old) ==
					(before.New_String == rebuilt.New)))
		}
	}()

	if len(d.Old) == 0 {
		if len(d.New) == 0 {
			// Empty rune slices reach here only on a direct call with empty texts.
			invariant.Dot_Product(invariant.Always(
				d.Old_String == "" && d.New_String == ""))
			return
		}
	}
	if d.Old_String == d.New_String {
		if d.Old_String != "" {
			d.Edits = append(d.Edits, Edit{Kind: Edit_Retain, Data: d.Old})
			return
		}
	}
	if d.New_String == "" {
		// Reaching here with both texts empty is handled above, so Old is non-empty.
		invariant.Dot_Product(invariant.Always(d.Old_String != ""))
		d.Edits = append(d.Edits, Edit{Kind: Edit_Delete, Data: d.Old})
		return
	}
	if d.Old_String == "" {
		invariant.Dot_Product(invariant.Always(d.New_String != ""))
		d.Edits = append(d.Edits, Edit{Kind: Edit_Insert, Data: d.New})
		return
	}

	old, new := d.Old, d.New
	before_count := len(d.Edits)
	trace := differ_algorithm_forward_trace(
		differ_algorithm_forward_trace_input{Old: old, New: new},
	)
	d.Edits = append(d.Edits, differ_algorithm_backtrack(differ_algorithm_backtrack_input{
		Trace: trace, Old: old, New: new,
	})...)
	slices.Reverse(d.Edits[before_count:])
}

// Carries the texts into the forward trace.
type differ_algorithm_forward_trace_input struct {
	// Old is the source text as runes.
	Old []rune
	// New is the target text as runes.
	New []rune
}

// Runs Myers' forward pass, returning the furthest-reaching X snapshot recorded
// at each edit depth.
func differ_algorithm_forward_trace(input differ_algorithm_forward_trace_input) (trace [][]int) {
	old, new := input.Old, input.New
	edits_max := len(old) + len(new)
	trace = make([][]int, 0, edits_max+1)
	tracker := make([]int, edits_max*2+1)

	for depth := range edits_max + 1 {
		previous_tracker := slices.Clone(tracker)
		more := differ_forward_step(differ_forward_step_input{
			Depth:            depth,
			Tracker:          tracker,
			Previous_Tracker: previous_tracker,
			Edits_Max:        edits_max,
			Old:              old,
			New:              new,
		})
		trace = append(trace, slices.Clone(tracker))
		if !more {
			break
		}
	}
	return trace
}

// Carries one forward-pass depth into differ_forward_step.
type differ_forward_step_input struct {
	// Depth is the current edit depth.
	Depth int
	// Tracker is the furthest-reaching X per diagonal, mutated in place.
	Tracker []int
	// Previous_Tracker is the prior depth's Tracker snapshot.
	Previous_Tracker []int
	// Edits_Max is the longest possible edit script, len(Old)+len(New).
	Edits_Max int
	// Old is the source text as runes.
	Old []rune
	// New is the target text as runes.
	New []rune
}

// Advances every diagonal of one Myers forward-pass depth, updating Tracker and
// reporting whether the far corner has not yet been reached.
func differ_forward_step(input differ_forward_step_input) (more bool) {
	depth, tracker, previous_tracker := input.Depth, input.Tracker, input.Previous_Tracker
	edits_max, old, new := input.Edits_Max, input.Old, input.New
	for k := -depth; k <= depth; k += 2 {
		k_offset := edits_max + k
		var x, y, previous_x int
		is_insert := k == -depth ||
			(k != depth && tracker[k_offset+1] > tracker[k_offset-1])
		if is_insert {
			previous_x = tracker[k_offset+1]
			x = previous_x
		} else {
			previous_x = tracker[k_offset-1]
			x = previous_x + 1
		}
		y = x - k

		// Reaching x never falls below its diagonal k; x > k means a snake (matching run)
		// extended this node, x == k means the diagonal was first reached here.
		invariant.Dot_Product(
			invariant.Always(x >= k),
			invariant.Sometimes(x > k),
		)

		// Furthest-reaching X is monotonic across depths along each diagonal.
		invariant.Dot_Product(invariant.Always(
			tracker[k_offset] >= previous_tracker[k_offset]))
		if k < depth {
			invariant.Dot_Product(invariant.Always(
				x >= previous_tracker[k_offset+1]))
		}
		if k > -depth {
			invariant.Dot_Product(invariant.Always(
				x >= previous_tracker[k_offset-1]))
			if is_insert {
				previous_k := k + 1
				previous_y := previous_x - previous_k
				// An insert step advances y by one and leaves x where it was.
				invariant.Dot_Product(
					invariant.Always(x == previous_x),
					invariant.Always(y == previous_y+1),
				)
			} else {
				previous_k := k - 1
				previous_y := previous_x - previous_k
				// A delete step advances x by one and leaves y where it was.
				invariant.Dot_Product(
					invariant.Always(x == previous_x+1),
					invariant.Always(y == previous_y),
				)
			}
		}

		for x < len(old) && y < len(new) && old[x] == new[y] {
			x, y = x+1, y+1
		}

		tracker[k_offset] = x
		if fully_converted := x >= len(old) && y >= len(new); fully_converted {
			return false
		}
	}
	return true
}

// Carries the forward trace and texts into the backtrack pass.
type differ_algorithm_backtrack_input struct {
	// Trace is the per-depth furthest-reaching X snapshots from the forward pass.
	Trace [][]int
	// Old is the source text as runes.
	Old []rune
	// New is the target text as runes.
	New []rune
}

// Walks the forward trace from the end, emitting the retains, deletes, and
// inserts of the minimal script in reverse order.
func differ_algorithm_backtrack(input differ_algorithm_backtrack_input) (edits []Edit) {
	trace, old, new := input.Trace, input.Old, input.New
	edits_max := len(old) + len(new)
	x, y := len(old), len(new)
	for depth := len(trace) - 1; depth >= 0; depth-- {
		trace_entry := trace[depth]
		k := x - y
		k_offset := edits_max + k

		var edit Edit
		var previous_k int
		is_insert := k == -depth ||
			(k != depth && trace_entry[k_offset+1] > trace_entry[k_offset-1])
		if is_insert {
			edit.Kind = Edit_Insert
			previous_k = k + 1
		} else {
			edit.Kind = Edit_Delete
			previous_k = k - 1
		}

		previous_x := trace_entry[edits_max+previous_k]
		previous_y := previous_x - previous_k

		right := x
		for x > previous_x && y > previous_y {
			x, y = x-1, y-1
		}
		left := x
		if left < right {
			edits = append(edits, Edit{Kind: Edit_Retain, Data: old[left:right]})
		}

		if depth > 0 {
			if edit.Kind == Edit_Delete {
				edit.Data = old[previous_x:][:1]
			}
			if edit.Kind == Edit_Insert {
				edit.Data = new[previous_y:][:1]
			}
		}
		if edit.Data != nil {
			edits = append(edits, edit)
		}

		x, y = previous_x, previous_y
	}
	return edits
}

// Helpers.

// Find_Common_Prefix_Input is the input for Find_Common_Prefix.
type Find_Common_Prefix_Input struct {
	// A is one rune slice.
	A []rune
	// B is the other rune slice.
	B []rune
}

// Find_Common_Prefix returns the longest run of runes that begins both inputs.
func Find_Common_Prefix(input Find_Common_Prefix_Input) (result []rune) {
	a, b := input.A, input.B
	defer func() {
		if len(result) > 0 {
			invariant.Dot_Product(
				invariant.Always(Runes_Have_Prefix(Runes_Have_Prefix_Input{
					String: a, Expect: result,
				})),
				invariant.Always(Runes_Have_Prefix(Runes_Have_Prefix_Input{
					String: b, Expect: result,
				})),
			)
		}
	}()
	if len(a) == 0 {
		return nil
	}
	if len(b) == 0 {
		return nil
	}
	l := min(len(a), len(b))
	for i_index := 0; i_index < l; i_index++ {
		if a[i_index] != b[i_index] {
			return a[:i_index]
		}
	}
	return a[:l]
}

// Find_Common_Suffix_Input is the input for Find_Common_Suffix.
type Find_Common_Suffix_Input struct {
	// A is one rune slice.
	A []rune
	// B is the other rune slice.
	B []rune
}

// Find_Common_Suffix returns the longest run of runes that ends both inputs.
func Find_Common_Suffix(input Find_Common_Suffix_Input) (result []rune) {
	a, b := input.A, input.B
	defer func() {
		if len(result) > 0 {
			invariant.Dot_Product(
				invariant.Always(Runes_Have_Suffix(Runes_Have_Suffix_Input{
					String: a, Expect: result,
				})),
				invariant.Always(Runes_Have_Suffix(Runes_Have_Suffix_Input{
					String: b, Expect: result,
				})),
			)
		}
	}()
	la, lb := len(a), len(b)
	l := min(la, lb)
	for i_index := 0; i_index < l; i_index++ {
		if a[la-1-i_index] != b[lb-1-i_index] {
			return a[la-i_index:]
		}
	}
	return a[la-l:]
}

// Find_Common_Run_Input is the input for Find_Common_Run.
type Find_Common_Run_Input struct {
	// A is one rune slice.
	A []rune
	// B is the other rune slice.
	B []rune
}

// Find_Common_Run returns the longest contiguous run of runes shared by both
// inputs, provided it spans at least half the longer input; otherwise nil.
func Find_Common_Run(input Find_Common_Run_Input) (run []rune) {
	a, b := input.A, input.B
	if len(a) < len(b) {
		a, b = b, a
	}

	al, bl := len(a), len(b)
	run_size_min := (al + 1) / 2
	if bl < run_size_min {
		return nil
	}
	for run_size := bl; run_size >= run_size_min; run_size-- {
		for i_index := 0; i_index <= al-run_size; i_index++ {
			for j_index := 0; j_index <= bl-run_size; j_index++ {
				a_slice := a[i_index:][:run_size:run_size]
				b_slice := b[j_index:][:run_size:run_size]
				if slices.Equal(a_slice, b_slice) {
					return a_slice
				}
			}
		}
	}
	return nil
}

// Runes_Have_Prefix_Input is the input for Runes_Have_Prefix.
type Runes_Have_Prefix_Input struct {
	// String is the slice tested.
	String []rune
	// Expect is the prefix sought.
	Expect []rune
}

// Runes_Have_Prefix reports whether the input begins with the non-empty Expect.
func Runes_Have_Prefix(input Runes_Have_Prefix_Input) (ok bool) {
	if len(input.String) == 0 {
		return false
	}
	if len(input.Expect) == 0 {
		return false
	}
	if len(input.Expect) > len(input.String) {
		return false
	}
	actual := input.String[:len(input.Expect)]
	return slices.Equal(actual, input.Expect)
}

// Runes_Have_Suffix_Input is the input for Runes_Have_Suffix.
type Runes_Have_Suffix_Input struct {
	// String is the slice tested.
	String []rune
	// Expect is the suffix sought.
	Expect []rune
}

// Runes_Have_Suffix reports whether the input ends with the non-empty Expect.
func Runes_Have_Suffix(input Runes_Have_Suffix_Input) (ok bool) {
	if len(input.String) == 0 {
		return false
	}
	if len(input.Expect) == 0 {
		return false
	}
	if len(input.Expect) > len(input.String) {
		return false
	}
	actual := input.String[len(input.String)-len(input.Expect):]
	return slices.Equal(actual, input.Expect)
}

// Input for runes_index.
type runes_index_input struct {
	// Haystack is the slice searched.
	Haystack []rune
	// Needle is the slice sought.
	Needle []rune
}

// Returns the first index at which Needle occurs in Haystack, or -1.
func runes_index(input runes_index_input) (index int) {
	if len(input.Needle) == 0 {
		return -1
	}
	if len(input.Needle) > len(input.Haystack) {
		return -1
	}
	for start := 0; start <= len(input.Haystack)-len(input.Needle); start++ {
		if slices.Equal(input.Haystack[start:][:len(input.Needle)], input.Needle) {
			return start
		}
	}
	return -1
}
