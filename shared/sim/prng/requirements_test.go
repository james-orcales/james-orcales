package prng_test

import (
	"reflect"
	"testing"

	"local/james-orcales/shared/sim/prng"
	"local/james-orcales/shared/testify"

	crypto_prng "local/james-orcales/shared/crypto/prng"
)

// Test_Seed_Expands_To_State checks New is deterministic and seed-sensitive.
func Test_Seed_Expands_To_State(t *testing.T) {
	first := prng.New(42)
	again := prng.New(42)
	if prng.Xoshiro_Next(&first) != prng.Xoshiro_Next(&again) {
		t.Fatalf("same seed produced different streams")
	}
	other := prng.New(43)
	repeat := prng.New(42)
	if prng.Xoshiro_Next(&other) == prng.Xoshiro_Next(&repeat) {
		t.Fatalf("distinct seeds produced the same first draw")
	}
}

// Test_Known_Sequence locks the output stream for a fixed seed.
func Test_Known_Sequence(t *testing.T) {
	generator := prng.New(0)
	// Frozen from this implementation. The constants and step match TigerBeetle's stdx.PRNG, so
	// the sequence equals its from_seed(0) stream; the contract is per-version reproducibility.
	want := []uint64{
		5987356902031041503,
		7051070477665621255,
		6633766593972829180,
		211316841551650330,
		9136120204379184874,
		379361710973160858,
		15813423377499357806,
		15596884590815070553,
	}
	for index := 0; index < len(want); index++ {
		value := prng.Xoshiro_Next(&generator)
		if uint64(value) != want[index] {
			t.Fatalf("draw %d was %d, want %d", index, value, want[index])
		}
	}
}

// Test_Below_Is_Bounded checks Below stays within zero and bound.
func Test_Below_Is_Bounded(t *testing.T) {
	generator := prng.New(1)
	bounds := []int{1, 2, 7, 1000, 1 << 40}
	for _, bound := range bounds {
		for draw_index := 0; draw_index < 10000; draw_index++ {
			value := prng.Xoshiro_Below(&generator, prng.Bound(bound))
			if value < 0 {
				t.Fatalf("Below(%d) returned negative %d", bound, value)
			}
			if int(value) >= bound {
				t.Fatalf("Below(%d) returned %d, out of range", bound, value)
			}
		}
	}
}

// Test_Boolean_Is_Even checks Boolean is roughly balanced over a large sample.
func Test_Boolean_Is_Even(t *testing.T) {
	generator := prng.New(3)
	sample_count := 100000
	true_count := 0
	for draw_index := 0; draw_index < sample_count; draw_index++ {
		if prng.Xoshiro_Boolean(&generator) {
			true_count++
		}
	}
	if true_count < sample_count*45/100 {
		t.Fatalf("Boolean true %d of %d, below band", true_count, sample_count)
	}
	if true_count > sample_count*55/100 {
		t.Fatalf("Boolean true %d of %d, above band", true_count, sample_count)
	}
}

// Test_Chance_Matches_Ratio checks Chance honors the integer Ratio, including the extremes.
func Test_Chance_Matches_Ratio(t *testing.T) {
	generator := prng.New(4)
	sample_count := 100000
	never := prng.Ratio{Numerator: 0, Denominator: 100}
	always := prng.Ratio{Numerator: 100, Denominator: 100}
	quarter := prng.Ratio{Numerator: 25, Denominator: 100}
	for draw_index := 0; draw_index < sample_count; draw_index++ {
		if prng.Xoshiro_Chance(&generator, never) {
			t.Fatalf("Chance with a zero numerator returned true")
		}
		if !prng.Xoshiro_Chance(&generator, always) {
			t.Fatalf("Chance with a full numerator returned false")
		}
	}
	true_count := 0
	for draw_index := 0; draw_index < sample_count; draw_index++ {
		if prng.Xoshiro_Chance(&generator, quarter) {
			true_count++
		}
	}
	if true_count < sample_count*20/100 {
		t.Fatalf("Chance quarter true %d of %d, below band", true_count, sample_count)
	}
	if true_count > sample_count*30/100 {
		t.Fatalf("Chance quarter true %d of %d, above band", true_count, sample_count)
	}
}

// Test_Sample_Matches_Weights checks Sample honors integer weights and skips zero-weight outcomes.
func Test_Sample_Matches_Weights(t *testing.T) {
	generator := prng.New(5)
	const RARE, COMMON, NEVER prng.Word = 1, 2, 3
	outcomes := prng.Outcomes{RARE, COMMON, NEVER}
	weights := prng.Weights{10, 90, 0}
	distribution := prng.New_Distribution(outcomes, weights)
	sample_count := 100000
	rare_count := 0
	common_count := 0
	for draw_index := 0; draw_index < sample_count; draw_index++ {
		item := prng.Xoshiro_Sample(&generator, distribution)
		if item == NEVER {
			t.Fatalf("Sample returned a zero-weight outcome")
		}
		if item == RARE {
			rare_count++
		}
		if item == COMMON {
			common_count++
		}
	}
	if rare_count < sample_count*5/100 {
		t.Fatalf("rare %d of %d, below band", rare_count, sample_count)
	}
	if rare_count > sample_count*15/100 {
		t.Fatalf("rare %d of %d, above band", rare_count, sample_count)
	}
	if common_count < sample_count*85/100 {
		t.Fatalf("common %d of %d, below band", common_count, sample_count)
	}
}

// Test_Shuffle_Permutes checks Shuffle preserves the multiset and can reorder.
func Test_Shuffle_Permutes(t *testing.T) {
	generator := prng.New(6)
	original := []prng.Index{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	permutation := prng.Permutation{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	prng.Xoshiro_Shuffle(&generator, permutation)
	seen := make([]bool, len(original))
	for _, value := range permutation {
		seen[value] = true
	}
	for _, present := range seen {
		if !present {
			t.Fatalf("Shuffle dropped or duplicated an element")
		}
	}
	reordered := false
	for attempt_index := 0; attempt_index < 10; attempt_index++ {
		prng.Xoshiro_Shuffle(&generator, permutation)
		for index := 0; index < len(original); index++ {
			if permutation[index] != original[index] {
				reordered = true
			}
		}
	}
	if !reordered {
		t.Fatalf("Shuffle never changed the order")
	}
}

// Test_Split_Is_Independent checks a Split child diverges from the parent's stream.
func Test_Split_Is_Independent(t *testing.T) {
	parent := prng.New(7)
	child := prng.Xoshiro_Split(&parent)
	differs := false
	for draw_index := 0; draw_index < 16; draw_index++ {
		if prng.Xoshiro_Next(&child) != prng.Xoshiro_Next(&parent) {
			differs = true
		}
	}
	if !differs {
		t.Fatalf("Split child tracked the parent stream")
	}
}

// Test_Types_Hold_No_Floating_Point checks the core types carry only integer fields.
func Test_Types_Hold_No_Floating_Point(t *testing.T) {
	types := []reflect.Type{
		reflect.TypeOf(prng.Xoshiro{}),
		reflect.TypeOf(prng.Ratio{}),
		reflect.TypeOf(prng.Distribution{}),
	}
	for _, candidate := range types {
		if type_has_float(candidate) {
			t.Fatalf("type %s holds a floating-point field", candidate.Name())
		}
	}
}

// Test_Every_Entry_Point_Is_Zero_Allocation checks no public operation reaches the heap. Each
// result escapes into a package-level sink through the closure, thus the compiler cannot drop
// the call, and the measurement is of the operation, not of a dead store.
func Test_Every_Entry_Point_Is_Zero_Allocation(t *testing.T) {
	generator := prng.New(8)
	sink := new_allocation_sink()
	certain := prng.Ratio{Numerator: 1, Denominator: 1}
	bimodal := prng.Bimodal_Distribution_Input{
		Fast: 1, Slow: 2, Slow_Chance: prng.Ratio{Numerator: 1, Denominator: 10},
	}
	percentile := prng.Percentile_Distribution_Input{
		P25: 1, P50: 2, P75: 3, P95: 4, P99: 5, P100: 6,
	}
	entry_points := []struct {
		Name   string
		Action func()
	}{
		{Name: "New", Action: func() { sink.Generator = prng.New(8) }},
		{Name: "Next", Action: func() { sink.Word = prng.Xoshiro_Next(&generator) }},
		{Name: "Below", Action: func() {
			sink.Index = prng.Xoshiro_Below(&generator, 100)
		}},
		{Name: "Boolean", Action: func() {
			sink.Boolean = prng.Xoshiro_Boolean(&generator)
		}},
		{Name: "Chance", Action: func() {
			sink.Boolean = prng.Xoshiro_Chance(&generator, certain)
		}},
		{Name: "New_Distribution", Action: func() {
			sink.Distribution = prng.New_Distribution(sink.Outcomes, sink.Weights)
		}},
		{Name: "Sample", Action: func() {
			sink.Word = prng.Xoshiro_Sample(&generator, sink.Distribution)
		}},
		{Name: "Bimodal_Fill", Action: func() {
			prng.Bimodal_Fill(bimodal, sink.Bimodal_Outcomes, sink.Bimodal_Weights)
		}},
		{Name: "Percentile_Fill", Action: func() {
			prng.Percentile_Fill(
				percentile, sink.Percentile_Outcomes, sink.Percentile_Weights,
			)
		}},
		{Name: "Shuffle", Action: func() {
			prng.Xoshiro_Shuffle(&generator, sink.Permutation)
		}},
		{Name: "Split", Action: func() {
			sink.Generator = prng.Xoshiro_Split(&generator)
		}},
		{Name: "To_Source", Action: func() {
			sink.Source = prng.Xoshiro_To_Source(&generator)
		}},
		{Name: "Source_Read", Action: func() {
			crypto_prng.Source_Read(sink.Source, sink.Bytes)
		}},
	}
	for _, entry_point := range entry_points {
		t.Run(entry_point.Name, func(t *testing.T) {
			// New_Distribution folds weights in place: each run starts from raw mass.
			sink.Weights[0], sink.Weights[1] = 10, 20
			sink.Weights[2], sink.Weights[3] = 30, 40
			testify.Zero_Allocation(t, entry_point.Action)
		})
	}
}

// Test_Xoshiro_Converts_To_Source checks the bound crypto/prng Source is a pure function of the
// seed and spends one Next word per eight bytes, so a pinned simulation seed replays each entropy
// draw a signer or key builder makes; a nil Xoshiro dies before binding.
func Test_Xoshiro_Converts_To_Source(t *testing.T) {
	first := prng.New(9)
	again := prng.New(9)
	other := prng.New(10)
	var got, want, differ [crypto_prng.WORD_BYTE_COUNT + 1]byte
	crypto_prng.Source_Read(prng.Xoshiro_To_Source(&first), got[:])
	crypto_prng.Source_Read(prng.Xoshiro_To_Source(&again), want[:])
	crypto_prng.Source_Read(prng.Xoshiro_To_Source(&other), differ[:])
	if got != want {
		t.Fatalf("same seed produced different bytes")
	}
	if got == differ {
		t.Fatalf("distinct seeds produced the same bytes")
	}
	reference := prng.New(9)
	word := prng.Xoshiro_Next(&reference)
	for index := 0; index < crypto_prng.WORD_BYTE_COUNT; index++ {
		if got[index] != byte(word>>(index*8)) {
			t.Fatalf("byte %d was %d, want little-endian word byte", index, got[index])
		}
	}
	if got[crypto_prng.WORD_BYTE_COUNT] != byte(prng.Xoshiro_Next(&reference)) {
		t.Fatalf("ninth byte did not spend a second word")
	}
	if prng.Xoshiro_Next(&reference) != prng.Xoshiro_Next(&first) {
		t.Fatalf("a partial word consumed more than one draw")
	}
	testify.Panics(t, func() { prng.Xoshiro_To_Source(nil) }, "nil Xoshiro did not die")
	// A constructed state reaches each word edge of the slot deterministically; chance would
	// take 2^64 draws to land on one of them.
	for _, value := range [...]prng.Word{
		prng.WORD_MINIMUM, prng.WORD_MINIMUM + 1, prng.WORD_MINIMUM + 2, prng.WORD_MAXIMUM,
	} {
		edge := generator_for_next(value)
		var octet [crypto_prng.WORD_BYTE_COUNT]byte
		crypto_prng.Source_Read(prng.Xoshiro_To_Source(&edge), octet[:])
		if octet[0] != byte(value) {
			t.Fatalf("constructed draw %d did not pass through the slot", value)
		}
	}
}

// Test_Bimodal_Distribution_Has_Two_Modes checks a fast and a slow cluster with an empty valley.
func Test_Bimodal_Distribution_Has_Two_Modes(t *testing.T) {
	generator := prng.New(12)
	outcomes := make(prng.Bimodal_Outcomes, prng.BIMODAL_OUTCOME_COUNT)
	weights := make(prng.Bimodal_Weights, prng.BIMODAL_OUTCOME_COUNT)
	prng.Bimodal_Fill(prng.Bimodal_Distribution_Input{
		Fast:        1000,
		Slow:        8000,
		Slow_Chance: prng.Ratio{Numerator: 10, Denominator: 100},
	}, outcomes, weights)
	distribution := prng.New_Distribution(prng.Outcomes(outcomes), prng.Weights(weights))
	sample_count := 200000
	fast_count := 0
	slow_count := 0
	for draw_index := 0; draw_index < sample_count; draw_index++ {
		value := prng.Xoshiro_Sample(&generator, distribution)
		if value <= 2000 {
			fast_count++
		}
		if value >= 8000 {
			slow_count++
		}
	}
	valley_count := sample_count - fast_count - slow_count
	if valley_count != 0 {
		t.Fatalf("bimodal produced %d values in the valley", valley_count)
	}
	if fast_count < sample_count*880/1000 {
		t.Fatalf("fast mode %d of %d, below band", fast_count, sample_count)
	}
	if fast_count > sample_count*920/1000 {
		t.Fatalf("fast mode %d of %d, above band", fast_count, sample_count)
	}
}

// Test_Percentile_Distribution_Hits_Percentiles checks draws reproduce the given p50, p95, and p99.
func Test_Percentile_Distribution_Hits_Percentiles(t *testing.T) {
	generator := prng.New(13)
	outcomes := make(prng.Percentile_Outcomes, prng.PERCENTILE_OUTCOME_COUNT)
	weights := make(prng.Percentile_Weights, prng.PERCENTILE_OUTCOME_COUNT)
	prng.Percentile_Fill(prng.Percentile_Distribution_Input{
		P25:  100,
		P50:  200,
		P75:  300,
		P95:  400,
		P99:  500,
		P100: 600,
	}, outcomes, weights)
	distribution := prng.New_Distribution(prng.Outcomes(outcomes), prng.Weights(weights))
	sample_count := 200000
	below_p50 := 0
	below_p95 := 0
	below_p99 := 0
	for draw_index := 0; draw_index < sample_count; draw_index++ {
		value := prng.Xoshiro_Sample(&generator, distribution)
		if value <= 200 {
			below_p50++
		}
		if value <= 400 {
			below_p95++
		}
		if value <= 500 {
			below_p99++
		}
	}
	if below_p50 < sample_count*480/1000 {
		t.Fatalf("p50 fraction %d of %d, below band", below_p50, sample_count)
	}
	if below_p50 > sample_count*520/1000 {
		t.Fatalf("p50 fraction %d of %d, above band", below_p50, sample_count)
	}
	if below_p95 < sample_count*940/1000 {
		t.Fatalf("p95 fraction %d of %d, below band", below_p95, sample_count)
	}
	if below_p95 > sample_count*960/1000 {
		t.Fatalf("p95 fraction %d of %d, above band", below_p95, sample_count)
	}
	if below_p99 < sample_count*985/1000 {
		t.Fatalf("p99 fraction %d of %d, below band", below_p99, sample_count)
	}
	if below_p99 > sample_count*995/1000 {
		t.Fatalf("p99 fraction %d of %d, above band", below_p99, sample_count)
	}
}

// Test_Invariant_Boundaries constructs deterministic draws because chance cannot prove word edges.
func Test_Invariant_Boundaries(t *testing.T) {
	verify_word_boundaries(t)
	verify_generator_boundaries()
	verify_seed_boundaries(t)
	verify_ratio_boundaries()
	verify_distribution_input_boundaries()
	verify_collection_boundaries()
	verify_distribution_boundaries()
}

// All storage the entry points read and write, allocated once, outside every measurement.
func new_allocation_sink() (sink allocation_sink) {
	sink.Outcomes = prng.Outcomes{0, 1, 2, 3}
	sink.Weights = prng.Weights{10, 20, 30, 40}
	sink.Bimodal_Outcomes = make(prng.Bimodal_Outcomes, prng.BIMODAL_OUTCOME_COUNT)
	sink.Bimodal_Weights = make(prng.Bimodal_Weights, prng.BIMODAL_OUTCOME_COUNT)
	sink.Percentile_Outcomes = make(
		prng.Percentile_Outcomes, prng.PERCENTILE_OUTCOME_COUNT,
	)
	sink.Percentile_Weights = make(
		prng.Percentile_Weights, prng.PERCENTILE_OUTCOME_COUNT,
	)
	sink.Permutation = prng.Permutation{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	sink.Bytes = make([]byte, crypto_prng.WORD_BYTE_COUNT)
	return sink
}

// Every result an entry point produces, held outside the measured closure.
type allocation_sink struct {
	Generator           prng.Xoshiro
	Word                prng.Word
	Index               prng.Index
	Boolean             prng.Boolean
	Distribution        prng.Distribution
	Source              crypto_prng.Source
	Outcomes            prng.Outcomes
	Weights             prng.Weights
	Bimodal_Outcomes    prng.Bimodal_Outcomes
	Bimodal_Weights     prng.Bimodal_Weights
	Percentile_Outcomes prng.Percentile_Outcomes
	Percentile_Weights  prng.Percentile_Weights
	Permutation         prng.Permutation
	Bytes               []byte
}

// Word values every full-width domain in this package states: both bounds and the interior
// sentinels the framework expands.
func special_words() (words []prng.Word) {
	return []prng.Word{prng.WORD_MINIMUM, 1, 2, prng.WORD_MAXIMUM}
}

// One generator per state word per special value. The other words hold a filler that keeps the
// state nonzero, thus each variant is a legal generator.
func boundary_generators() (generators []prng.Xoshiro) {
	const FILLER = 3
	filled := prng.Xoshiro{First: FILLER, Second: FILLER, Third: FILLER, Fourth: FILLER}
	for _, value := range special_words() {
		first, second, third, fourth := filled, filled, filled, filled
		first.First = prng.Xoshiro_First(value)
		second.Second = prng.Xoshiro_Second(value)
		third.Third = prng.Xoshiro_Third(value)
		fourth.Fourth = prng.Xoshiro_Fourth(value)
		generators = append(generators, first, second, third, fourth)
	}
	return generators
}

// Every draw on every boundary generator, so each entry point's state domains see each edge.
// Each draw takes its own copy: a draw advances the state it is handed.
func verify_generator_boundaries() {
	distribution := prng.New_Distribution(prng.Outcomes{0}, prng.Weights{1})
	certain := prng.Ratio{Numerator: 1, Denominator: 1}
	for _, boundary := range boundary_generators() {
		next := boundary
		prng.Xoshiro_Next(&next)
		below := boundary
		prng.Xoshiro_Below(&below, 1)
		flip := boundary
		prng.Xoshiro_Boolean(&flip)
		chance := boundary
		prng.Xoshiro_Chance(&chance, certain)
		sample := boundary
		prng.Xoshiro_Sample(&sample, distribution)
		shuffle := boundary
		prng.Xoshiro_Shuffle(&shuffle, prng.Permutation{})
		split := boundary
		prng.Xoshiro_Split(&split)
		source := boundary
		prng.Xoshiro_To_Source(&source)
	}
}

// Seeds whose New lands one state word on each special value. Word k of New(seed) is the
// splitmix64 avalanche of seed plus k+1 increments, and each avalanche step is a bijection,
// thus the seed for a target word is its inverse image minus the increments. A Split child is
// New of the parent's next draw, thus a parent constructed to draw that seed reaches the same
// edge through Split.
func verify_seed_boundaries(t *testing.T) {
	prng.New(2)
	// The first strided state is seed plus one increment, thus these seeds put that state at
	// one, two, and the maximum.
	for _, state := range []uint64{1, 2, uint64(prng.SEED_MAXIMUM)} {
		prng.New(prng.Seed(state - prng.SPLIT_MIX_INCREMENT))
	}
	for _, value := range special_words() {
		for word_index := uint64(1); word_index <= 4; word_index++ {
			state := split_mix_inverse(uint64(value))
			seed := prng.Seed(state - word_index*prng.SPLIT_MIX_INCREMENT)
			built := prng.New(seed)
			if word_at(built, word_index) != value {
				t.Fatalf("seed %d missed word %d at %d", seed, word_index, value)
			}
			parent := generator_for_next(prng.Word(seed))
			prng.Xoshiro_Split(&parent)
		}
	}
}

// State word k, one-based, of a generator.
func word_at(generator prng.Xoshiro, word_index uint64) (word prng.Word) {
	switch word_index {
	case 1:
		return prng.Word(generator.First)
	case 2:
		return prng.Word(generator.Second)
	case 3:
		return prng.Word(generator.Third)
	default:
		return prng.Word(generator.Fourth)
	}
}

// The state whose splitmix64 avalanche is word: each xorshift and each odd multiply inverts.
func split_mix_inverse(word uint64) (state uint64) {
	state = xorshift_right_inverse(word, 31)
	state *= multiplicative_inverse(prng.SPLIT_MIX_MULTIPLIER_SECOND)
	state = xorshift_right_inverse(state, 27)
	state *= multiplicative_inverse(prng.SPLIT_MIX_MULTIPLIER_FIRST)
	state = xorshift_right_inverse(state, 30)
	return state
}

// Inverts value ^= value >> shift. Each application recovers shift more high bits, thus one
// application per shift-width of the word suffices.
func xorshift_right_inverse(value uint64, shift uint) (original uint64) {
	original = value
	for step := uint(0); step <= 64/shift; step++ {
		original = value ^ (original >> shift)
	}
	return original
}

// The inverse of an odd word modulo 2^64 by Newton iteration: an odd word is its own inverse
// to three bits, and each step doubles the correct bits, thus six steps pass sixty-four.
func multiplicative_inverse(odd uint64) (inverse uint64) {
	inverse = odd
	for step_index := 0; step_index < 6; step_index++ {
		inverse *= 2 - odd*inverse
	}
	return inverse
}

func verify_word_boundaries(t *testing.T) {
	prng.New(prng.SEED_MAXIMUM)
	for _, value := range [...]prng.Word{
		prng.WORD_MINIMUM,
		prng.WORD_MINIMUM + 1,
		prng.WORD_MINIMUM + 2,
		prng.WORD_MAXIMUM,
	} {
		generator := generator_for_next(value)
		if actual := prng.Xoshiro_Next(&generator); actual != value {
			t.Fatalf("constructed draw was %d, want %d", actual, value)
		}
	}

	maximum_generator := generator_for_next(prng.WORD_MAXIMUM)
	maximum_index := prng.Xoshiro_Below(&maximum_generator, prng.BOUND_MAXIMUM)
	if maximum_index != prng.INDEX_MAXIMUM {
		t.Fatalf("maximum index was %d, want %d", maximum_index, prng.INDEX_MAXIMUM)
	}
}

// Zero, one, two, and maximum witnesses.
func boundary_ratios() (ratios []prng.Ratio) {
	return []prng.Ratio{
		{Numerator: 0, Denominator: 1},
		{Numerator: 1, Denominator: 1},
		{Numerator: 2, Denominator: 2},
		{Numerator: prng.Ratio_Numerator(prng.WEIGHT_MAXIMUM),
			Denominator: prng.Ratio_Denominator(prng.WEIGHT_MAXIMUM)},
	}
}

func verify_ratio_boundaries() {
	for _, ratio := range boundary_ratios() {
		generator := generator_for_next(prng.WORD_MAXIMUM)
		prng.Xoshiro_Chance(&generator, ratio)
	}
}

func verify_distribution_input_boundaries() {
	ratios := boundary_ratios()
	for _, value := range [...]prng.Word{0, 1, 2, prng.WORD_MAXIMUM} {
		index := int(value)
		if value == prng.WORD_MAXIMUM {
			index = len(ratios) - 1
		}
		// Storage at its floor and at the shared width, thus both ends of each domain.
		for _, count := range []int{
			prng.BIMODAL_OUTCOME_COUNT, prng.DISTRIBUTION_COUNT_MAXIMUM,
		} {
			prng.Bimodal_Fill(prng.Bimodal_Distribution_Input{
				Fast:        prng.Bimodal_Fast(value),
				Slow:        prng.Bimodal_Slow(value),
				Slow_Chance: ratios[index],
			}, make(prng.Bimodal_Outcomes, count), make(prng.Bimodal_Weights, count))
		}
		for _, count := range []int{
			prng.PERCENTILE_OUTCOME_COUNT, prng.DISTRIBUTION_COUNT_MAXIMUM,
		} {
			prng.Percentile_Fill(prng.Percentile_Distribution_Input{
				P25:  prng.Percentile_25(value),
				P50:  prng.Percentile_50(value),
				P75:  prng.Percentile_75(value),
				P95:  prng.Percentile_95(value),
				P99:  prng.Percentile_99(value),
				P100: prng.Percentile_100(value),
			},
				make(prng.Percentile_Outcomes, count),
				make(prng.Percentile_Weights, count),
			)
		}
	}
}

func verify_collection_boundaries() {
	for _, count := range []int{0, 1, 2, prng.PERMUTATION_COUNT_MAXIMUM} {
		generator := prng.New(1)
		prng.Xoshiro_Shuffle(&generator, make(prng.Permutation, count))
	}
}

func verify_distribution_boundaries() {
	for _, count := range []int{1, 2, prng.DISTRIBUTION_COUNT_MAXIMUM} {
		outcomes := make(prng.Outcomes, count)
		weights := make(prng.Weights, count)
		for index := range outcomes {
			outcomes[index] = prng.Word(index)
			weights[index] = 1
		}
		if count == 1 {
			// The one slot: the widest weight, and the widest outcome a sample returns.
			weights[0] = prng.WEIGHT_MAXIMUM
			outcomes[0] = prng.WORD_MAXIMUM
		}
		if count == 2 {
			weights[1] = 2
		}
		distribution := prng.New_Distribution(outcomes, weights)
		generator := generator_for_next(prng.WORD_MAXIMUM)
		prng.Xoshiro_Sample(&generator, distribution)
	}
}

func generator_for_next(value prng.Word) (generator prng.Xoshiro) {
	generator.First = prng.Xoshiro_First(value)
	generator.Fourth = prng.Xoshiro_Fourth(0 - value)
	if value == 0 {
		generator.Second = 1
	}
	return generator
}

func Benchmark_Next(b *testing.B) {
	generator := prng.New(1)
	for b.Loop() {
		prng.Xoshiro_Next(&generator)
	}
}

func Benchmark_Below(b *testing.B) {
	generator := prng.New(1)
	for b.Loop() {
		prng.Xoshiro_Below(&generator, 100)
	}
}

func Benchmark_Boolean(b *testing.B) {
	generator := prng.New(1)
	for b.Loop() {
		prng.Xoshiro_Boolean(&generator)
	}
}

func Benchmark_Chance(b *testing.B) {
	generator := prng.New(1)
	probability := prng.Ratio{Numerator: 8, Denominator: 100}
	for b.Loop() {
		prng.Xoshiro_Chance(&generator, probability)
	}
}

func Benchmark_Sample(b *testing.B) {
	generator := prng.New(1)
	distribution := prng.New_Distribution(
		prng.Outcomes{0, 1, 2, 3}, prng.Weights{10, 20, 30, 40},
	)
	for b.Loop() {
		prng.Xoshiro_Sample(&generator, distribution)
	}
}

func Benchmark_Shuffle(b *testing.B) {
	generator := prng.New(1)
	permutation := prng.Permutation{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	for b.Loop() {
		prng.Xoshiro_Shuffle(&generator, permutation)
	}
}

func Benchmark_Split(b *testing.B) {
	generator := prng.New(1)
	for b.Loop() {
		prng.Xoshiro_Split(&generator)
	}
}

// Reports whether a struct type, or the element of a slice or array field, is floating point.
func type_has_float(structure reflect.Type) (has bool) {
	for field_index := 0; field_index < structure.NumField(); field_index++ {
		field_type := structure.Field(field_index).Type
		kind := field_type.Kind()
		if kind == reflect.Slice {
			kind = field_type.Elem().Kind()
		}
		if kind == reflect.Array {
			kind = field_type.Elem().Kind()
		}
		if kind == reflect.Float32 {
			has = true
		}
		if kind == reflect.Float64 {
			has = true
		}
	}
	return has
}
