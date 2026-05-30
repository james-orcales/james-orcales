package myers

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/james-orcales/james-orcales/golang_snacks/invariant"
)

const (
	EditRetain uint8 = 10
	EditDelete       = 20
	EditInsert       = 30
)

type Edit struct {
	Kind uint8
	Data []rune
}

type Differ struct {
	Edits          []Edit
	Old, New       []rune
	OldStr, NewStr string
}

type NewInput struct {
	Old string
	New string
}

func New(in NewInput) (d *Differ) {
	return &Differ{
		Old:    []rune(in.Old),
		New:    []rune(in.New),
		OldStr: in.Old,
		NewStr: in.New,
	}
}

func (d *Differ) Reset() {
	d.Edits = d.Edits[:0]
	d.Old, d.New = d.Old[:0], d.New[:0]
	d.OldStr, d.NewStr = "", ""
}

// TODO: Concise diffs -> Configurable surrounding line count for each edit.
func (dfr *Differ) LineDiff() (diff string) {
	{
		before := *dfr
		defer func() {
			invariant.Always(before.OldStr == dfr.OldStr, "LineDiff only mutates Differ.Edits")
			invariant.Always(before.NewStr == dfr.NewStr, "LineDiff only mutates Differ.Edits")
			invariant.XAlways(func() bool { return slices.Equal(before.Old, dfr.Old) }, "LineDiff only mutates Differ.Edits")
			invariant.XAlways(func() bool { return slices.Equal(before.New, dfr.New) }, "LineDiff only mutates Differ.Edits")
		}()
	}
	if dfr.OldStr == dfr.NewStr {
		if dfr.OldStr == "" {
			return ""
		} else {
			return fmt.Sprintf(" %s", dfr.OldStr)
		}
	}
	if dfr.OldStr == "" {
		return "+" + strings.ReplaceAll(dfr.NewStr, "\n", "\n+")
	}
	if dfr.NewStr == "" {
		return "-" + strings.ReplaceAll(dfr.OldStr, "\n", "\n-")
	}

	invariant.Sometimes(strings.LastIndexByte(dfr.OldStr, '\n') != len(dfr.OldStr)-1, "Old text doesn't have a trailing newline")
	invariant.Sometimes(strings.LastIndexByte(dfr.NewStr, '\n') != len(dfr.NewStr)-1, "New text doesn't have a trailing newline")

	var old strings.Builder
	var new strings.Builder

	n := strings.Count(dfr.OldStr, "\n")
	old.Grow(n)
	new.Grow(strings.Count(dfr.NewStr, "\n"))

	var ch rune
	lineToRune := make(map[string]rune, n)
	runeToLine := make(map[rune]string, n)

	for line := range strings.SplitSeq(dfr.OldStr, "\n") {
		if _, ok := lineToRune[line]; !ok {
			lineToRune[line] = ch
			runeToLine[ch] = line
			ch++
		}
		old.WriteRune(lineToRune[line])
	}
	for line := range strings.SplitSeq(dfr.NewStr, "\n") {
		if _, ok := lineToRune[line]; !ok {
			lineToRune[line] = ch
			runeToLine[ch] = line
			ch++
		}
		new.WriteRune(lineToRune[line])
	}

	d := New(NewInput{Old: old.String(), New: new.String()})
	defer func() { dfr.Edits = d.Edits }()

	d.OptimizedDiff()
	d.MergeShiftDiffCleanup()

	result := make([]string, 0, len(d.Edits))
	for _, edit := range d.Edits {
		if len(edit.Data) == 0 {
			continue
		}
		var indicator string
		switch edit.Kind {
		case EditRetain:
			indicator = " "
		case EditInsert:
			indicator = "+"
		case EditDelete:
			indicator = "-"
		}
		for _, char := range edit.Data {
			result = append(result, fmt.Sprintf("%s%s", indicator, runeToLine[char]))
		}
	}
	return strings.Join(result, "\n")
}

func (d *Differ) Diff() (diff string) {
	before := d
	d.OptimizedDiff()
	d.MergeShiftDiffCleanup()
	invariant.XAlways(func() bool {
		old, new := d.rebuildStringFromEdits()
		return (before.OldStr == old) == (before.NewStr == new)
	}, "Edits add up to original text")
	return d.String()
}

func (d Differ) String() (s string) {
	var sb strings.Builder
	for _, edit := range d.Edits {
		if len(edit.Data) == 0 {
			continue
		}
		kind := ""
		switch edit.Kind {
		case EditRetain:
			kind = " "
		case EditInsert:
			kind = "+"
		case EditDelete:
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

func (d *Differ) MergeShiftDiffCleanup() {
	{
		before := *d
		defer func() {
			invariant.Always(before.OldStr == d.OldStr, "MergeShiftDiffCleanup only mutates Differ.Edits")
			invariant.Always(before.NewStr == d.NewStr, "MergeShiftDiffCleanup only mutates Differ.Edits")
			invariant.XAlways(
				func() bool { return slices.Equal(before.Old, d.Old) },
				"MergeShiftDiffCleanup only mutates Differ.Edits",
			)
			invariant.XAlways(
				func() bool { return slices.Equal(before.New, d.New) },
				"MergeShiftDiffCleanup only mutates Differ.Edits",
			)
			invariant.XAlways(func() bool {
				old, new := d.rebuildStringFromEdits()
				return (before.OldStr == old) == (before.NewStr == new)
			}, "Edits add up to original text")
		}()
	}
	if len(d.Edits) < 3 {
		return
	}

	invariant.Always(len(d.OldStr) > 0, "Old is not empty")
	invariant.Always(len(d.NewStr) > 0, "New is not empty")
	invariant.Always(len(d.Old) > 0, "Old is not empty")
	invariant.Always(len(d.New) > 0, "New is not empty")

	if d.Edits[0].Kind != EditRetain {
		d.Edits = slices.Insert(d.Edits, 0, Edit{Kind: EditRetain, Data: nil})
	}
	if d.Edits[len(d.Edits)-1].Kind != EditRetain {
		d.Edits = append(d.Edits, Edit{Kind: EditRetain, Data: nil})
	}

	// === MERGE ===================================================================================================
	func() {
		result := make([]Edit, 0, len(d.Edits))
		defer func() { d.Edits = result }()

		old, new := d.Old, d.New
		var toDelete, toInsert []rune
		for _, edit := range d.Edits {
			switch edit.Kind {
			case EditDelete:
				toDelete = old[:len(toDelete)+len(edit.Data)]
			case EditInsert:
				toInsert = new[:len(toInsert)+len(edit.Data)]
			case EditRetain:
				currentEdit := edit
				hasDelete := len(toDelete) > 0
				hasInsert := len(toInsert) > 0
				if hasDelete && hasInsert {
					if prefix := FindCommonPrefix(FindCommonPrefixInput{A: toInsert, B: toDelete}); len(prefix) > 0 {
						prevRetain := &result[len(result)-1]
						prevRetain.Data = slices.Concat(prevRetain.Data, prefix)
						toDelete = toDelete[len(prefix):]
						toInsert = toInsert[len(prefix):]
					}
					if suffix := FindCommonSuffix(FindCommonSuffixInput{A: toInsert, B: toDelete}); len(suffix) > 0 {
						currentEdit.Data = slices.Concat(currentEdit.Data, suffix)
						toDelete = toDelete[:len(toDelete)-len(suffix)]
						toInsert = toInsert[:len(toInsert)-len(suffix)]
					}
				}
				if hasDelete {
					result = append(result, Edit{Kind: EditDelete, Data: toDelete})
					old = old[len(toDelete):]
				}
				if hasInsert {
					result = append(result, Edit{Kind: EditInsert, Data: toInsert})
					new = new[len(toInsert):]
				}
				result = append(result, currentEdit)
				old = old[len(currentEdit.Data):]
				new = new[len(currentEdit.Data):]
				toDelete = nil
				toInsert = nil
			}
		}
	}()

	invariant.Always(len(d.Edits) >= 3, "Actual edits are sandwiched by empty edits")
	if len(d.Edits[0].Data) == 0 {
		d.Edits = d.Edits[1:]
	}
	if len(d.Edits[len(d.Edits)-1].Data) == 0 {
		d.Edits = d.Edits[:len(d.Edits)-1]
	}

	invariant.XAlways(func() bool {
		for _, edit := range d.Edits {
			if len(edit.Data) == 0 {
				return false
			}
		}
		return true
	}, "All edits have non-empty data")

	// === SHIFT ===================================================================================================
	isShifted := false
	func() {
		result := []Edit{d.Edits[0]}
		defer func() {
			result = append(result, d.Edits[len(d.Edits)-1])
			d.Edits = result
		}()
		for offset, edit := range d.Edits[1 : len(d.Edits)-1] {
			offset++
			prev := &result[len(result)-1]
			next := &d.Edits[offset+1]
			if prev.Kind == EditRetain && next.Kind == EditRetain {
				invariant.Always(edit.Kind != EditRetain, "Edit kinds are alternated")
				if RunesHaveSuffix(RunesHaveSuffixInput{Str: edit.Data, Expect: prev.Data}) {
					isShifted = true
					next.Data = slices.Concat(prev.Data, next.Data)
					prev.Data = slices.Concat(prev.Data, edit.Data[:len(edit.Data)-len(prev.Data)])
					prev.Kind = edit.Kind
					continue
				} else if RunesHavePrefix(RunesHavePrefixInput{Str: edit.Data, Expect: next.Data}) {
					isShifted = true
					prev.Data = slices.Concat(prev.Data, next.Data)
					next.Data = slices.Concat(edit.Data[len(next.Data):], next.Data)
					next.Kind = edit.Kind
					continue
				}
			}
			result = append(result, edit)
		}
	}()
	if isShifted {
		d.MergeShiftDiffCleanup()
	}
}

func (d *Differ) OptimizedDiff() {
	{
		before := *d
		defer func() {
			invariant.Always(before.OldStr == d.OldStr, "OptimizedDiff only mutates Differ.Edits")
			invariant.Always(before.NewStr == d.NewStr, "OptimizedDiff only mutates Differ.Edits")
			invariant.XAlways(func() bool { return slices.Equal(before.Old, d.Old) }, "OptimizedDiff only mutates Differ.Edits")
			invariant.XAlways(func() bool { return slices.Equal(before.New, d.New) }, "OptimizedDiff only mutates Differ.Edits")
			invariant.XAlways(func() bool {
				old, new := d.rebuildStringFromEdits()
				return (before.OldStr == old) == (before.NewStr == new)
			}, "Edits add up to original text")
		}()
	}

	old, new := d.Old, d.New

	if d.OldStr == d.NewStr {
		d.Edits = append(d.Edits, Edit{Kind: EditRetain, Data: old})
		return
	}
	if d.NewStr == "" {
		d.Edits = append(d.Edits, Edit{Kind: EditDelete, Data: old})
		return
	}
	if d.OldStr == "" {
		d.Edits = append(d.Edits, Edit{Kind: EditInsert, Data: new})
		return
	}

	prefix := FindCommonPrefix(FindCommonPrefixInput{A: old, B: new})
	if len(prefix) > 0 {
		d.Edits = append(d.Edits, Edit{Kind: EditRetain, Data: prefix})
	}
	old = old[len(prefix):]
	new = new[len(prefix):]

	suffix := FindCommonSuffix(FindCommonSuffixInput{A: old, B: new})
	defer func() {
		if len(suffix) > 0 {
			d.Edits = append(d.Edits, Edit{Kind: EditRetain, Data: suffix})
		}
	}()
	old = old[:len(old)-len(suffix)]
	new = new[:len(new)-len(suffix)]

	isSimpleDelete := len(old) > 0 && len(new) == 0
	isSimpleInsert := len(old) == 0 && len(new) > 0

	if isSimpleDelete {
		d.Edits = append(d.Edits, Edit{Kind: EditDelete, Data: old})
		return
	} else if isSimpleInsert {
		d.Edits = append(d.Edits, Edit{Kind: EditInsert, Data: new})
		return
	} else {
		x := runesIndex(runesIndexInput{Haystack: old, Needle: new})
		y := runesIndex(runesIndexInput{Haystack: new, Needle: old})
		if isDeleteSandwich := x > 0; isDeleteSandwich {
			d.Edits = append(d.Edits, Edit{Kind: EditDelete, Data: old[:x]})
			old = old[x:]

			d.Edits = append(d.Edits, Edit{Kind: EditRetain, Data: old[:len(new)]})
			old = old[len(new):]

			d.Edits = append(d.Edits, Edit{Kind: EditDelete, Data: old})
			return
		} else if isInsertSandwich := y > 0; isInsertSandwich {
			d.Edits = append(d.Edits, Edit{Kind: EditInsert, Data: new[:y]})
			new = new[y:]

			d.Edits = append(d.Edits, Edit{Kind: EditRetain, Data: new[:len(old)]})
			new = new[len(old):]

			d.Edits = append(d.Edits, Edit{Kind: EditInsert, Data: new})
			return
		} else {
			var recurse func(*Differ)
			recurse = func(diff *Differ) {
				oldRunes, newRunes := diff.Old, diff.New
				substr := FindCommonSubstring(FindCommonSubstringInput{A: oldRunes, B: newRunes})
				if len(substr) == 0 {
					diff.AlgorithmDiff()
					return
				}
				newSubstrStart := runesIndex(runesIndexInput{Haystack: newRunes, Needle: substr})
				oldSubstrStart := runesIndex(runesIndexInput{Haystack: oldRunes, Needle: substr})
				{
					dClone := Differ{
						Edits:  diff.Edits,
						Old:    oldRunes[:oldSubstrStart],
						New:    newRunes[:newSubstrStart],
						OldStr: diff.OldStr,
						NewStr: diff.NewStr,
					}
					recurse(&dClone)
					diff.Edits = dClone.Edits
				}
				diff.Edits = append(diff.Edits, Edit{Kind: EditRetain, Data: substr})
				{
					dClone := Differ{
						Edits:  diff.Edits,
						Old:    oldRunes[oldSubstrStart+len(substr):],
						New:    newRunes[newSubstrStart+len(substr):],
						OldStr: diff.OldStr,
						NewStr: diff.NewStr,
					}
					recurse(&dClone)
					diff.Edits = dClone.Edits
				}
			}
			dfr := *d
			dfr.Old = old
			dfr.New = new
			recurse(&dfr)
			d.Edits = dfr.Edits
			return
		}
	}
}

func (d *Differ) AlgorithmDiff() {
	{
		before := *d
		defer func() {
			invariant.Always(before.OldStr == d.OldStr, "AlgorithmDiff only mutates Differ.Edits")
			invariant.Always(before.NewStr == d.NewStr, "AlgorithmDiff only mutates Differ.Edits")
			invariant.XAlways(func() bool { return slices.Equal(before.Old, d.Old) }, "AlgorithmDiff only mutates Differ.Edits")
			invariant.XAlways(func() bool { return slices.Equal(before.New, d.New) }, "AlgorithmDiff only mutates Differ.Edits")

			cond := (string(before.Old) == before.OldStr) == (string(before.New) == before.NewStr)
			invariant.Sometimes(cond, "Runes contain the actual text")
			// TODO: Create a unit test that triggers this:
			// invariant.Sometimes(!cond, "Runes contain lineToRuneMapping under Differ.LineDiff")
			if cond {
				invariant.XAlways(func() bool {
					old, new := d.rebuildStringFromEdits()
					return (before.OldStr == old) == (before.NewStr == new)
				}, "Edits add up to original text")
			}
		}()
	}

	if len(d.Old) == 0 && len(d.New) == 0 {
		invariant.Always(d.OldStr == "" && d.NewStr == "", "Empty slices only from direct call with empty strings")
		return
	}
	if d.OldStr == d.NewStr && d.OldStr != "" {
		d.Edits = append(d.Edits, Edit{Kind: EditRetain, Data: d.Old})
		return
	}
	if d.NewStr == "" {
		invariant.Always(d.OldStr != "", "Simple delete")
		d.Edits = append(d.Edits, Edit{Kind: EditDelete, Data: d.Old})
		return
	}
	if d.OldStr == "" {
		invariant.Always(d.NewStr != "", "Simple insert")
		d.Edits = append(d.Edits, Edit{Kind: EditInsert, Data: d.New})
		return
	}

	old, new := d.Old, d.New
	before := len(d.Edits)
	defer func() {
		slices.Reverse(d.Edits[before:])
	}()

	maxEdits := len(old) + len(new)
	trace := make([][]int, 0, maxEdits+1)
	tracker := make([]int, maxEdits*2+1)

	for depth := range maxEdits + 1 {
		prevTracker := slices.Clone(tracker)
		shouldContinue := func() bool {
			for k := -depth; k <= depth; k += 2 {
				kOffset := maxEdits + k
				var x, y, prevX int
				isInsert := k == -depth || (k != depth && tracker[kOffset+1] > tracker[kOffset-1])
				if isInsert {
					prevX = tracker[kOffset+1]
					x = prevX
				} else {
					prevX = tracker[kOffset-1]
					x = prevX + 1
				}
				y = x - k

				invariant.Always(x >= k, "")
				invariant.Sometimes(x == k, "This node has never been reached before")
				invariant.Sometimes(x > k, "The node prior to insertion created a snake")

				invariant.Always(
					tracker[kOffset] >= prevTracker[kOffset],
					"Furthest-reaching X increases across depths on same diagonal",
				)
				if k < depth {
					invariant.Always(
						x >= prevTracker[kOffset+1],
						"Furthest-reaching X increases across depths from above diagonal",
					)
				}
				if k > -depth {
					invariant.Always(
						x >= prevTracker[kOffset-1],
						"Furthest-reaching X increases across depths on below diagonal",
					)
					if isInsert {
						prevK := k + 1
						prevY := prevX - prevK
						invariant.Always(x == prevX, "Insert only increments y")
						invariant.Always(y == prevY+1, "Insert only increments y")
					} else {
						prevK := k - 1
						prevY := prevX - prevK
						invariant.Always(x == prevX+1, "Delete only increments x")
						invariant.Always(y == prevY, "Delete only increments x")
					}
				}

				for x < len(old) && y < len(new) && old[x] == new[y] {
					x, y = x+1, y+1
				}

				tracker[kOffset] = x
				if fullyConverted := x >= len(old) && y >= len(new); fullyConverted {
					return false
				}
			}
			return true
		}()
		trace = append(trace, slices.Clone(tracker))
		if !shouldContinue {
			break
		}
	}

	x, y := len(old), len(new)
	for depth := len(trace) - 1; depth >= 0; depth -= 1 {
		traceEntry := trace[depth]
		k := x - y
		kOffset := maxEdits + k

		var edit Edit
		var prevK int
		if k == -depth || (k != depth && traceEntry[kOffset+1] > traceEntry[kOffset-1]) {
			edit.Kind = EditInsert
			prevK = k + 1
		} else {
			edit.Kind = EditDelete
			prevK = k - 1
		}

		prevX := traceEntry[maxEdits+prevK]
		prevY := prevX - prevK

		right := x
		for x > prevX && y > prevY {
			x, y = x-1, y-1
		}
		left := x
		if left < right {
			d.Edits = append(d.Edits, Edit{Kind: EditRetain, Data: old[left:right]})
		}

		if depth > 0 && edit.Kind == EditDelete {
			edit.Data = old[prevX:][:1]
		}
		if depth > 0 && edit.Kind == EditInsert {
			edit.Data = new[prevY:][:1]
		}
		if edit.Data != nil {
			d.Edits = append(d.Edits, edit)
		}

		x, y = prevX, prevY
	}
}

// === HELPERS =========================================================================================================

type FindCommonPrefixInput struct {
	A []rune
	B []rune
}

func FindCommonPrefix(in FindCommonPrefixInput) (result []rune) {
	a, b := in.A, in.B
	defer func() {
		if len(result) > 0 {
			invariant.Always(RunesHavePrefix(RunesHavePrefixInput{
				Str: a, Expect: result,
			}), "Both slices have the common prefix")
			invariant.Always(RunesHavePrefix(RunesHavePrefixInput{
				Str: b, Expect: result,
			}), "Both slices have the common prefix")
		}
	}()
	if len(a) > 0 && len(b) > 0 {
		l := min(len(a), len(b))
		for i := 0; i < l; i++ {
			if a[i] != b[i] {
				return a[:i]
			}
		}
		return a[:l]
	}
	return nil
}

type FindCommonSuffixInput struct {
	A []rune
	B []rune
}

func FindCommonSuffix(in FindCommonSuffixInput) (result []rune) {
	a, b := in.A, in.B
	defer func() {
		if len(result) > 0 {
			invariant.Always(RunesHaveSuffix(RunesHaveSuffixInput{
				Str: a, Expect: result,
			}), "Both slices have the common suffix")
			invariant.Always(RunesHaveSuffix(RunesHaveSuffixInput{
				Str: b, Expect: result,
			}), "Both slices have the common suffix")
		}
	}()
	la, lb := len(a), len(b)
	l := min(la, lb)
	for i := 0; i < l; i++ {
		if a[la-1-i] != b[lb-1-i] {
			return a[la-i:]
		}
	}
	return a[la-l:]
}

type FindCommonSubstringInput struct {
	A []rune
	B []rune
}

func FindCommonSubstring(in FindCommonSubstringInput) (substr []rune) {
	a, b := in.A, in.B
	if len(a) < len(b) {
		a, b = b, a
	}

	al, bl := len(a), len(b)
	minLength := (al + 1) / 2
	if bl >= minLength {
		for length := bl; length >= minLength; length-- {
			for i := 0; i <= al-length; i++ {
				for j := 0; j <= bl-length; j++ {
					aSlice := a[i:][:length:length]
					bSlice := b[j:][:length:length]
					if slices.Equal(aSlice, bSlice) {
						return aSlice
					}
				}
			}
		}
	}
	return nil
}

type RunesHavePrefixInput struct {
	Str    []rune
	Expect []rune
}

func RunesHavePrefix(in RunesHavePrefixInput) (ok bool) {
	if len(in.Str) == 0 || len(in.Expect) == 0 || len(in.Expect) > len(in.Str) {
		return false
	}
	actual := in.Str[:len(in.Expect)]
	return slices.Equal(actual, in.Expect)
}

type RunesHaveSuffixInput struct {
	Str    []rune
	Expect []rune
}

func RunesHaveSuffix(in RunesHaveSuffixInput) (ok bool) {
	if len(in.Str) == 0 || len(in.Expect) == 0 || len(in.Expect) > len(in.Str) {
		return false
	}
	actual := in.Str[len(in.Str)-len(in.Expect):]
	return slices.Equal(actual, in.Expect)
}

type runesIndexInput struct {
	Haystack []rune
	Needle   []rune
}

func runesIndex(in runesIndexInput) (idx int) {
	if len(in.Needle) > 0 && len(in.Needle) <= len(in.Haystack) {
		for start := 0; start <= len(in.Haystack)-len(in.Needle); start++ {
			if slices.Equal(in.Haystack[start:][:len(in.Needle)], in.Needle) {
				return start
			}
		}
	}
	return -1
}

func printJSON(obj interface{}) {
	text, _ := json.MarshalIndent(obj, "", "\t")
	fmt.Println(string(text))
}

func (d *Differ) rebuildStringFromEdits() (oldStr string, newStr string) {
	var old strings.Builder
	var new strings.Builder

	for _, edit := range d.Edits {
		if edit.Kind == EditRetain {
			for _, r := range edit.Data {
				old.WriteRune(r)
				new.WriteRune(r)
			}
		} else if edit.Kind == EditDelete {
			for _, r := range edit.Data {
				old.WriteRune(r)
			}
		} else if edit.Kind == EditInsert {
			for _, r := range edit.Data {
				new.WriteRune(r)
			}
		}
	}
	return old.String(), new.String()
}
