//go:build !noassert

// This file is the enforcing half of a compile-time dual. It holds every entry point that runs in
// a shipped binary — the eager guards and the bare observation that validate, enforce, and
// credit — while enforcement_noassert.go holds a signature-identical set of no-ops selected by
// `-tags noassert`. The registration and analysis machinery stays in invariant.go untagged because
// it runs only under `go test`, where the tag is never set.
//
// The split is what makes the tag honest. A shared body guarded by a build-time constant would
// leave the real code in the binary and make elimination a question about the inliner's budget;
// here the disabled build compiles bodies that are literally empty, so "off" costs nothing by
// construction rather than by the compiler's discretion.

package invariant

import (
	"strconv"
)

// Recorder_Always stays outside Product because an eager guard has no second branch to widen a
// demanded grid. It panics immediately when condition is false in every run mode; under a plain
// test run it also credits reachability so an uncalled guard remains visible as a gap.
func Recorder_Always[T ~bool](recorder *Recorder, condition T, message string) {
	if !condition {
		panic(ASSERTION_FAILURE_MESSAGE_PREFIX + message + "  Always — condition was false")
	}
	// Enforcement (the panic above) runs in every mode; coverage is credited under a test run
	// or the fuzz coordinator (not a worker), matching Ensure's recording policy. The
	// reachability entry is seeded statically by recorder_register_eager_always;
	// recorder_increment no-ops when the bare Always was never registered (an Always in a
	// non-analyzed package).
	if !recorder.Is_Test {
		return
	}
	if recorder.Is_Benchmark {
		return
	}
	recorder_increment(recorder, message, true)
}

// Recorder_Sometimes is the bare observation: it claims its condition is witnessed both true and
// false across the suite, never that it holds now, so no polarity may panic. Recording follows
// Recorder_Always's policy; a key registration never seeded no-ops inside recorder_increment.
// The identifier scopes the key so one property observed at two boundaries earns two entries;
// registration refuses an empty identifier, so the key shape is unconditional.
func Recorder_Sometimes[T ~bool](
	recorder *Recorder, identifier string, condition T, message string,
) {
	if !recorder.Is_Test {
		return
	}
	if recorder.Is_Benchmark {
		return
	}
	recorder_increment(recorder, identifier+ELEMENT_MESSAGE_SEPARATOR+message, bool(condition))
}

// Recorder_Range is the bare bounds guard: eager like Always, panicking with the identifier, the
// value, and the violated bound. Trailing exclusions are holes inside the interval the value must
// also avoid; a hole at or outside a bound is a malformed guard and panics in every mode. The
// formatting, the hole loop, and the crediting are outlined so the passing path stays under the
// inliner's budget — a hole-less call pays one len check on a nil slice.
func Recorder_Range[T Integer](
	recorder *Recorder, identifier string, value T, minimum T, maximum T, excluded ...T,
) {
	if value < minimum {
		range_minimum_panic(value, minimum, identifier)
	}
	if value > maximum {
		range_maximum_panic(value, maximum, identifier)
	}
	if len(excluded) != 0 {
		range_excluded(value, minimum, maximum, excluded, identifier)
	}
	range_credit(recorder, identifier, value, minimum, maximum)
}

func range_excluded[T Integer](value T, minimum T, maximum T, excluded []T, identifier string) {
	for _, hole := range excluded {
		if hole <= minimum {
			range_exclusion_panic(hole, identifier)
		}
		if hole >= maximum {
			range_exclusion_panic(hole, identifier)
		}
		if value == hole {
			panic(ASSERTION_FAILURE_MESSAGE_PREFIX + identifier + "  Range — value " +
				integer_text(value) + " is excluded")
		}
	}
}

// Coverage follows Recorder_Always's policy. The two boundary witnesses are seeded statically
// per root with the same texts, so both sides rendezvous on identical keys; an unregistered
// call no-ops inside recorder_increment.
func range_credit[T Integer](
	recorder *Recorder, identifier string, value T, minimum T, maximum T,
) {
	if !recorder.Is_Test {
		return
	}
	if recorder.Is_Benchmark {
		return
	}
	recorder_increment(recorder,
		identifier+ELEMENT_MESSAGE_SEPARATOR+RANGE_MESSAGE_MINIMUM, value == minimum)
	recorder_increment(recorder,
		identifier+ELEMENT_MESSAGE_SEPARATOR+RANGE_MESSAGE_MAXIMUM, value == maximum)
}

// Recorder_Enum is the bare membership guard: the value must equal one of members. Fewer than
// two distinct members is a malformed guard — a one-member enum is an equality Always in
// disguise — and panics in every mode. The panic names the identifier and lists the members so
// a failure names the whole legal set.
func Recorder_Enum[T Integer](recorder *Recorder, identifier string, value T, members ...T) {
	if enum_distinct_count(members) < 2 {
		panic(ASSERTION_FAILURE_MESSAGE_PREFIX + identifier +
			"  Enum — an enum needs at least two distinct members")
	}
	member_found := false
	for _, member := range members {
		if value == member {
			member_found = true
		}
	}
	if !member_found {
		enum_membership_panic(identifier, value, members)
	}
	enum_credit(recorder, identifier, value, members)
}

func enum_distinct_count[T Integer](members []T) (count int) {
	for index, member := range members {
		duplicate := false
		for _, earlier := range members[:index] {
			if earlier == member {
				duplicate = true
			}
		}
		if !duplicate {
			count++
		}
	}
	return count
}

// Each distinct member owns one witness; a repeated member credits only through its first
// occurrence so the observations mirror the statically seeded set.
func enum_credit[T Integer](recorder *Recorder, identifier string, value T, members []T) {
	if !recorder.Is_Test {
		return
	}
	if recorder.Is_Benchmark {
		return
	}
	for index, member := range members {
		duplicate := false
		for _, earlier := range members[:index] {
			if earlier == member {
				duplicate = true
			}
		}
		if duplicate {
			continue
		}
		key := identifier + ELEMENT_MESSAGE_SEPARATOR +
			Enum_Member_Message(integer_text(member))
		recorder_increment(recorder, key, value == member)
	}
}

func enum_membership_panic[T Integer](identifier string, value T, members []T) {
	members_text := ""
	for _, member := range members {
		if members_text != "" {
			members_text += " "
		}
		members_text += integer_text(member)
	}
	panic(ASSERTION_FAILURE_MESSAGE_PREFIX + identifier + "  Enum — value " +
		integer_text(value) + " is not among the members " + members_text)
}

// The failure path affords the formatting these build; outlining them is what keeps the
// Recorder_Range body a pair of compares.
func range_exclusion_panic[T Integer](hole T, identifier string) {
	panic(ASSERTION_FAILURE_MESSAGE_PREFIX + identifier + "  Range — exclusion " +
		integer_text(hole) + " is not strictly inside the interval")
}

func range_minimum_panic[T Integer](value T, minimum T, identifier string) {
	panic(ASSERTION_FAILURE_MESSAGE_PREFIX + identifier + "  Range — value " +
		integer_text(value) + " is below the minimum " + integer_text(minimum))
}

func range_maximum_panic[T Integer](value T, maximum T, identifier string) {
	panic(ASSERTION_FAILURE_MESSAGE_PREFIX + identifier + "  Range — value " +
		integer_text(value) + " is above the maximum " + integer_text(maximum))
}

// A non-negative value converts to uint64 exactly for every Integer width — uint64 spans them —
// and a negative one is signed by definition, so int64 spans it. The split avoids reflection.
func integer_text[T Integer](value T) (text string) {
	if value >= 0 {
		return strconv.FormatUint(uint64(value), 10)
	}
	return strconv.FormatInt(int64(value), 10)
}
