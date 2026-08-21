// Package levenshtein measures bounded rune edit distance without allocation.
package levenshtein

import (
	"local/james-orcales/shared/simulation/aver/default"
	"local/james-orcales/shared/slices"
	"local/james-orcales/shared/strings"
)

// TEXT_SIZE_MAXIMUM reuses repository text boundary.
const TEXT_SIZE_MAXIMUM = strings.TEXT_SIZE_MAXIMUM

// TEXT_SIZE_UNVALIDATED_MAXIMUM admits first rejected byte.
const TEXT_SIZE_UNVALIDATED_MAXIMUM = TEXT_SIZE_MAXIMUM + 1

// RUNE_COUNT_MAXIMUM follows byte bound because each rune consumes at least one byte.
const RUNE_COUNT_MAXIMUM = TEXT_SIZE_MAXIMUM

// ROW_COUNT includes empty-prefix boundary.
const ROW_COUNT = RUNE_COUNT_MAXIMUM + 1

// CANDIDATE_COUNT_MAXIMUM reuses repository slice boundary.
const CANDIDATE_COUNT_MAXIMUM = slices.SLICE_COUNT_MAXIMUM

// STATUS_OK reports accepted input.
const STATUS_OK Status = 0

// STATUS_INPUT_INVALID reports oversized text.
const STATUS_INPUT_INVALID Status = 1

// Status reports whether bounded operation ran.
type Status uint8

// Status_Invariants lists both operation outcomes.
func Status_Invariants(value Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value), uint8(STATUS_OK), uint8(STATUS_INPUT_INVALID)).
		Ensure()
}

// Found reports whether Closest selected candidate.
type Found bool

// Found_Invariants requires match and miss coverage.
func Found_Invariants(value Found, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Closest finds one candidate.").
		Ensure()
}

// Distance_Value is bounded rune edit count.
type Distance_Value int

// Distance_Value_Invariants follows maximum decoded rune count.
func Distance_Value_Invariants(value Distance_Value, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), strings.TEXT_SIZE_MINIMUM, RUNE_COUNT_MAXIMUM).
		Ensure()
}

// Match is borrowed selected candidate.
type Match string

// Match_Invariants keeps result inside accepted text bound.
func Match_Invariants(value Match, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), strings.TEXT_SIZE_MINIMUM, TEXT_SIZE_MAXIMUM).
		Ensure()
}

// From_Text_Unvalidated is hostile source text before size validation.
type From_Text_Unvalidated string

// From_Text_Unvalidated_Invariants admits first rejected source byte.
func From_Text_Unvalidated_Invariants(
	value From_Text_Unvalidated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), strings.TEXT_SIZE_MINIMUM, TEXT_SIZE_UNVALIDATED_MAXIMUM).
		Ensure()
}

// To_Text_Unvalidated is hostile destination text before size validation.
type To_Text_Unvalidated string

// To_Text_Unvalidated_Invariants admits first rejected destination byte.
func To_Text_Unvalidated_Invariants(
	value To_Text_Unvalidated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), strings.TEXT_SIZE_MINIMUM, TEXT_SIZE_UNVALIDATED_MAXIMUM).
		Ensure()
}

// Target_Text_Unvalidated is hostile typo target before size validation.
type Target_Text_Unvalidated string

// Target_Text_Unvalidated_Invariants admits first rejected target byte.
func Target_Text_Unvalidated_Invariants(
	value Target_Text_Unvalidated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), strings.TEXT_SIZE_MINIMUM, TEXT_SIZE_UNVALIDATED_MAXIMUM).
		Ensure()
}

// Candidates_Unvalidated is bounded borrowed candidate collection.
type Candidates_Unvalidated []string

// Candidates_Unvalidated_Invariants bounds candidate search count.
func Candidates_Unvalidated_Invariants(
	value Candidates_Unvalidated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, CANDIDATE_COUNT_MAXIMUM).
		Ensure()
}

// From_Runes owns decoded source characters.
type From_Runes [RUNE_COUNT_MAXIMUM]rune

// From_Runes_Invariants fixes source workspace capacity.
func From_Runes_Invariants(value From_Runes, _ aver.Namespace) {
	aver.Always(
		len(value) == RUNE_COUNT_MAXIMUM,
		"From rune workspace has fixed text capacity.",
	)
}

// To_Runes owns decoded destination characters.
type To_Runes [RUNE_COUNT_MAXIMUM]rune

// To_Runes_Invariants fixes destination workspace capacity.
func To_Runes_Invariants(value To_Runes, _ aver.Namespace) {
	aver.Always(
		len(value) == RUNE_COUNT_MAXIMUM,
		"To rune workspace has fixed text capacity.",
	)
}

// Previous_Row owns prior dynamic-programming row.
type Previous_Row [ROW_COUNT]int

// Previous_Row_Invariants fixes one slot per destination boundary.
func Previous_Row_Invariants(value Previous_Row, _ aver.Namespace) {
	aver.Always(len(value) == ROW_COUNT, "Previous row has every rune boundary.")
}

// Current_Row owns current dynamic-programming row.
type Current_Row [ROW_COUNT]int

// Current_Row_Invariants fixes one slot per destination boundary.
func Current_Row_Invariants(value Current_Row, _ aver.Namespace) {
	aver.Always(len(value) == ROW_COUNT, "Current row has every rune boundary.")
}

// Workspace owns every rune and matrix row used by Distance and Closest.
type Workspace struct {
	// From avoids allocating source rune conversion.
	From From_Runes
	// To avoids allocating destination rune conversion.
	To To_Runes
	// Previous carries prior distances.
	Previous Previous_Row
	// Current carries next distances.
	Current Current_Row
}

// Workspace_Invariants composes fixed caller-owned storage.
func Workspace_Invariants(value Workspace, namespace aver.Namespace) {
	From_Runes_Invariants(value.From, namespace)
	To_Runes_Invariants(value.To, namespace)
	Previous_Row_Invariants(value.Previous, namespace)
	Current_Row_Invariants(value.Current, namespace)
}

// Distance_Input carries caller storage and hostile texts.
type Distance_Input struct {
	// Workspace owns all scratch state.
	Workspace *Workspace
	// From is source text.
	From From_Text_Unvalidated
	// To is destination text.
	To To_Text_Unvalidated
}

// Distance_Input_Invariants composes storage and both text boundaries.
func Distance_Input_Invariants(value Distance_Input, namespace aver.Namespace) {
	Workspace_Invariants(*value.Workspace, namespace)
	From_Text_Unvalidated_Invariants(value.From, namespace)
	To_Text_Unvalidated_Invariants(value.To, namespace)
}

// Distance returns rune edit count without allocating.
func Distance(input Distance_Input) (distance Distance_Value, status Status) {
	defer func() {
		Distance_Value_Invariants(distance, "distance.distance")
		Status_Invariants(status, "distance.status")
	}()
	Distance_Input_Invariants(input, "distance.input")
	if len(input.From) > TEXT_SIZE_MAXIMUM {
		return 0, STATUS_INPUT_INVALID
	}
	if len(input.To) > TEXT_SIZE_MAXIMUM {
		return 0, STATUS_INPUT_INVALID
	}
	from_count := 0
	for _, character := range input.From {
		input.Workspace.From[from_count] = character
		from_count++
	}
	to_count := 0
	for _, character := range input.To {
		input.Workspace.To[to_count] = character
		to_count++
	}
	for boundary := 0; boundary <= to_count; boundary++ {
		input.Workspace.Previous[boundary] = boundary
	}
	for from_index := 1; from_index <= from_count; from_index++ {
		input.Workspace.Current[0] = from_index
		for to_index := 1; to_index <= to_count; to_index++ {
			substitution_cost := 1
			if input.Workspace.From[from_index-1] == input.Workspace.To[to_index-1] {
				substitution_cost = 0
			}
			input.Workspace.Current[to_index] = min(
				input.Workspace.Previous[to_index]+1,
				input.Workspace.Current[to_index-1]+1,
				input.Workspace.Previous[to_index-1]+substitution_cost,
			)
		}
		copy(input.Workspace.Previous[:to_count+1], input.Workspace.Current[:to_count+1])
	}
	return Distance_Value(input.Workspace.Previous[to_count]), STATUS_OK
}

// Closest_Input carries caller storage and bounded search values.
type Closest_Input struct {
	// Workspace is reused for every candidate distance.
	Workspace *Workspace
	// Target is possibly mistyped text.
	Target Target_Text_Unvalidated
	// Candidates are accepted spellings.
	Candidates Candidates_Unvalidated
}

// Closest_Input_Invariants composes search storage and values.
func Closest_Input_Invariants(value Closest_Input, namespace aver.Namespace) {
	Workspace_Invariants(*value.Workspace, namespace)
	Target_Text_Unvalidated_Invariants(value.Target, namespace)
	Candidates_Unvalidated_Invariants(value.Candidates, namespace)
}

// Closest returns earliest nearest candidate without allocating.
func Closest(input Closest_Input) (match Match, found Found, status Status) {
	defer func() {
		Match_Invariants(match, "closest.match")
		Found_Invariants(found, "closest.found")
		Status_Invariants(status, "closest.status")
	}()
	Closest_Input_Invariants(input, "closest.input")
	if len(input.Target) > TEXT_SIZE_MAXIMUM {
		return "", false, STATUS_INPUT_INVALID
	}
	best_distance := Distance_Value(0)
	for _, candidate := range input.Candidates {
		distance, distance_status := Distance(Distance_Input{
			Workspace: input.Workspace,
			From:      From_Text_Unvalidated(input.Target),
			To:        To_Text_Unvalidated(candidate),
		})
		if distance_status != STATUS_OK {
			return "", false, STATUS_INPUT_INVALID
		}
		threshold := Distance_Value(max(len(input.Target), len(candidate)) / 3)
		if distance > threshold {
			continue
		}
		if found {
			if distance >= best_distance {
				continue
			}
		}
		best_distance = distance
		match = Match(candidate)
		found = true
	}
	return match, found, STATUS_OK
}
