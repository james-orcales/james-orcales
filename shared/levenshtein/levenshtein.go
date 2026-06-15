// Package levenshtein measures how many single-character edits separate two strings
// and, on top of that, picks the candidate nearest a mistyped one — the "did you
// mean" suggestion behind a command-line parser or any other name lookup.
package levenshtein

// Distance_Input is the input for Distance.
type Distance_Input struct {
	// From is one of the two strings under comparison.
	From string
	// To is the other; the distance is symmetric, so argument order does not matter.
	To string
}

// Distance returns the Levenshtein edit distance: the fewest single-rune insertions,
// deletions, or substitutions that turn From into To.
func Distance(input Distance_Input) (distance int) {
	from := []rune(input.From)
	to := []rune(input.To)
	// Roll two rows to bound memory at O(len(To)): previous_row[j] holds the distance
	// from the processed prefix of From to to[:j], and current_row the next one.
	previous_row := make([]int, len(to)+1)
	for j := range previous_row {
		previous_row[j] = j
	}
	for i := 1; i <= len(from); i++ {
		current_row := make([]int, len(to)+1)
		current_row[0] = i
		for j := 1; j <= len(to); j++ {
			substitution_cost := 1
			if from[i-1] == to[j-1] {
				substitution_cost = 0
			}
			current_row[j] = min(
				previous_row[j]+1,                   // delete a rune from From
				current_row[j-1]+1,                  // insert a rune from To
				previous_row[j-1]+substitution_cost, // keep or substitute
			)
		}
		previous_row = current_row
	}
	return previous_row[len(to)]
}

// Closest_Input is the input for Closest.
type Closest_Input struct {
	// Target is the (possibly mistyped) string to match.
	Target string
	// Candidates are the valid strings to match against.
	Candidates []string
}

// Closest returns the candidate nearest Target by edit distance; found is false when
// none is close enough to be a likely typo. The threshold scales with length, so a
// short string demands a near-exact match while a longer one tolerates proportionally
// more. On a tie the earliest candidate wins.
func Closest(input Closest_Input) (match string, found bool) {
	best_distance := 0
	for _, candidate := range input.Candidates {
		distance := Distance(Distance_Input{From: input.Target, To: candidate})
		threshold := max(len(input.Target), len(candidate)) / 3
		if distance > threshold {
			continue
		}
		// A later candidate replaces the match only when strictly nearer, so a tie
		// keeps the earliest.
		if found {
			if distance >= best_distance {
				continue
			}
		}
		best_distance = distance
		match = candidate
		found = true
	}
	return match, found
}
