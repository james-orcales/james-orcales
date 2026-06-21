package prng_test

import (
	"reflect"
	"testing"

	"github.com/james-orcales/james-orcales/shared/prng"
)

// Test_Seed_Expands_To_State checks New is deterministic and seed-sensitive.
func Test_Seed_Expands_To_State(t *testing.T) {
	first := prng.New(42)
	again := prng.New(42)
	if prng.Generator_Next(&first) != prng.Generator_Next(&again) {
		t.Fatalf("same seed produced different streams")
	}
	other := prng.New(43)
	repeat := prng.New(42)
	if prng.Generator_Next(&other) == prng.Generator_Next(&repeat) {
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
		value := prng.Generator_Next(&generator)
		if value != want[index] {
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
			value := prng.Generator_Below(&generator, bound)
			if value < 0 {
				t.Fatalf("Below(%d) returned negative %d", bound, value)
			}
			if value >= bound {
				t.Fatalf("Below(%d) returned %d, out of range", bound, value)
			}
		}
	}
}

// Test_Element_Comes_From_Slice checks Element returns a member and rejects an empty slice.
func Test_Element_Comes_From_Slice(t *testing.T) {
	generator := prng.New(2)
	items := []string{"a", "b", "c"}
	for draw_index := 0; draw_index < 1000; draw_index++ {
		item := prng.Generator_Element(&generator, items)
		found := false
		for _, candidate := range items {
			if item == candidate {
				found = true
			}
		}
		if !found {
			t.Fatalf("Element returned %q, not in the slice", item)
		}
	}
	panicked := did_panic(func() {
		empty := []string{}
		prng.Generator_Element(&generator, empty)
	})
	if !panicked {
		t.Fatalf("Element on an empty slice did not panic")
	}
}

// Test_Boolean_Is_Even checks Boolean is roughly balanced over a large sample.
func Test_Boolean_Is_Even(t *testing.T) {
	generator := prng.New(3)
	sample_count := 100000
	true_count := 0
	for draw_index := 0; draw_index < sample_count; draw_index++ {
		if prng.Generator_Boolean(&generator) {
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
		if prng.Generator_Chance(&generator, never) {
			t.Fatalf("Chance with a zero numerator returned true")
		}
		if !prng.Generator_Chance(&generator, always) {
			t.Fatalf("Chance with a full numerator returned false")
		}
	}
	true_count := 0
	for draw_index := 0; draw_index < sample_count; draw_index++ {
		if prng.Generator_Chance(&generator, quarter) {
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
	outcomes := []string{"rare", "common", "never"}
	weights := []uint64{10, 90, 0}
	distribution := prng.New_Distribution(outcomes, weights)
	sample_count := 100000
	rare_count := 0
	common_count := 0
	for draw_index := 0; draw_index < sample_count; draw_index++ {
		item := prng.Generator_Sample(&generator, distribution)
		if item == "never" {
			t.Fatalf("Sample returned a zero-weight outcome")
		}
		if item == "rare" {
			rare_count++
		}
		if item == "common" {
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
	original := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	items := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	prng.Generator_Shuffle(&generator, items)
	seen := make([]bool, len(original))
	for _, value := range items {
		seen[value] = true
	}
	for _, present := range seen {
		if !present {
			t.Fatalf("Shuffle dropped or duplicated an element")
		}
	}
	reordered := false
	for attempt_index := 0; attempt_index < 10; attempt_index++ {
		prng.Generator_Shuffle(&generator, items)
		for index := 0; index < len(items); index++ {
			if items[index] != original[index] {
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
	child := prng.Generator_Split(&parent)
	differs := false
	for draw_index := 0; draw_index < 16; draw_index++ {
		if prng.Generator_Next(&child) != prng.Generator_Next(&parent) {
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
		reflect.TypeOf(prng.Generator{}),
		reflect.TypeOf(prng.Ratio{}),
		reflect.TypeOf(prng.Distribution[int]{}),
	}
	for _, candidate := range types {
		if type_has_float(candidate) {
			t.Fatalf("type %s holds a floating-point field", candidate.Name())
		}
	}
}

// Test_Hot_Path_Is_Zero_Allocation checks a steady-state Next draw does not allocate.
func Test_Hot_Path_Is_Zero_Allocation(t *testing.T) {
	generator := prng.New(8)
	allocations := testing.AllocsPerRun(1000, func() {
		prng.Generator_Next(&generator)
	})
	if allocations != 0 {
		t.Fatalf("Next allocated %.1f times per call, want zero", allocations)
	}
}

// Test_Bimodal_Distribution_Has_Two_Modes checks a fast and a slow cluster with an empty valley.
func Test_Bimodal_Distribution_Has_Two_Modes(t *testing.T) {
	generator := prng.New(12)
	distribution := prng.Bimodal_Distribution(&prng.Bimodal_Distribution_Input{
		Fast:        1000,
		Slow:        8000,
		Slow_Chance: prng.Ratio{Numerator: 10, Denominator: 100},
	})
	sample_count := 200000
	fast_count := 0
	slow_count := 0
	for draw_index := 0; draw_index < sample_count; draw_index++ {
		value := prng.Generator_Sample(&generator, distribution)
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
	distribution := prng.Percentile_Distribution(&prng.Percentile_Distribution_Input{
		P25:  100,
		P50:  200,
		P75:  300,
		P95:  400,
		P99:  500,
		P100: 600,
	})
	sample_count := 200000
	below_p50 := 0
	below_p95 := 0
	below_p99 := 0
	for draw_index := 0; draw_index < sample_count; draw_index++ {
		value := prng.Generator_Sample(&generator, distribution)
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

// Runs action and reports whether it panicked, used to assert preconditions.
func did_panic(action func()) (panicked bool) {
	defer func() {
		if recover() != nil {
			panicked = true
		}
	}()
	action()
	return panicked
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
