package lru_test

import (
	"reflect"
	"testing"

	"local/james-orcales/shared/io"
	"local/james-orcales/shared/lru"
	"local/james-orcales/shared/time"
)

// Test_Simple_Evicts_Oldest checks New_Simple caps at its size, evicting the LRU entry and
// reporting it, while re-adding a key overwrites in place without eviction.
func Test_Simple_Evicts_Oldest(t *testing.T) {
	c := lru.New_Simple[int, int](2, nil)
	evicted_1 := lru.Simple_Add(c, 1, 1)
	evicted_2 := lru.Simple_Add(c, 2, 2)
	if evicted_1 {
		t.Fatalf("add 1 under cap reported an eviction")
	}
	if evicted_2 {
		t.Fatalf("add 2 under cap reported an eviction")
	}
	if !lru.Simple_Add(c, 3, 3) {
		t.Fatalf("add past cap did not report eviction")
	}
	if lru.Simple_Contains(c, 1) {
		t.Fatalf("oldest key 1 was not evicted")
	}
	if lru.Simple_Add(c, 2, 22) {
		t.Fatalf("overwriting an existing key reported an eviction")
	}
	value, ok := lru.Simple_Peek(c, 2)
	if !ok {
		t.Fatalf("overwrite lost key 2")
	}
	if value != 22 {
		t.Fatalf("overwrite value = %d, want 22", value)
	}
	if lru.Simple_Count(c) != 2 {
		t.Fatalf("count = %d, want 2", lru.Simple_Count(c))
	}
	if lru.Simple_Cap(c) != 2 {
		t.Fatalf("cap = %d, want 2", lru.Simple_Cap(c))
	}
}

// Test_Simple_Get_Renews_Recency checks Simple_Get returns a hit and moves it to the MRU end
// so eviction falls elsewhere; a miss returns zero, false.
func Test_Simple_Get_Renews_Recency(t *testing.T) {
	c := lru.New_Simple[int, int](2, nil)
	lru.Simple_Add(c, 1, 1)
	lru.Simple_Add(c, 2, 2)
	value, ok := lru.Simple_Get(c, 1)
	if !ok {
		t.Fatalf("get 1 missed")
	}
	if value != 1 {
		t.Fatalf("get 1 = %d, want 1", value)
	}
	lru.Simple_Add(c, 3, 3)
	if lru.Simple_Contains(c, 2) {
		t.Fatalf("key 2 should have been evicted after 1 was renewed")
	}
	if !lru.Simple_Contains(c, 1) {
		t.Fatalf("renewed key 1 was evicted")
	}
	_, hit := lru.Simple_Get(c, 42)
	if hit {
		t.Fatalf("missing key reported present")
	}
}

// Test_Simple_Peek_And_Contains_Leave_Recency checks neither Simple_Peek nor Simple_Contains
// updates recency, so the inspected entry stays the eviction target.
func Test_Simple_Peek_And_Contains_Leave_Recency(t *testing.T) {
	c := lru.New_Simple[int, int](2, nil)
	lru.Simple_Add(c, 1, 1)
	lru.Simple_Add(c, 2, 2)
	value, ok := lru.Simple_Peek(c, 1)
	if !ok {
		t.Fatalf("peek 1 missed")
	}
	if value != 1 {
		t.Fatalf("peek 1 = %d, want 1", value)
	}
	if !lru.Simple_Contains(c, 1) {
		t.Fatalf("contains 1 = false")
	}
	lru.Simple_Add(c, 3, 3)
	if lru.Simple_Contains(c, 1) {
		t.Fatalf("peek/contains wrongly renewed key 1")
	}
}

// Test_Simple_Contains_Or_Add_Skips_Present_Keys checks Simple_Contains_Or_Add inserts only when
// absent and Simple_Peek_Or_Add returns the existing value without overwriting.
func Test_Simple_Contains_Or_Add_Skips_Present_Keys(t *testing.T) {
	c := lru.New_Simple[int, int](2, nil)
	present, _ := lru.Simple_Contains_Or_Add(c, 1, 1)
	if present {
		t.Fatalf("contains-or-add reported a fresh key as present")
	}
	present, _ = lru.Simple_Contains_Or_Add(c, 1, 111)
	if !present {
		t.Fatalf("contains-or-add reported a present key as absent")
	}
	value, _ := lru.Simple_Peek(c, 1)
	if value != 1 {
		t.Fatalf("contains-or-add overwrote a present key: got %d, want 1", value)
	}
	previous, present, _ := lru.Simple_Peek_Or_Add(c, 1, 999)
	if !present {
		t.Fatalf("peek-or-add reported a present key as absent")
	}
	if previous != 1 {
		t.Fatalf("peek-or-add previous = %d, want 1", previous)
	}
	_, present, _ = lru.Simple_Peek_Or_Add(c, 2, 2)
	if present {
		t.Fatalf("peek-or-add reported a fresh key as present")
	}
	if !lru.Simple_Contains(c, 2) {
		t.Fatalf("peek-or-add did not insert the absent key")
	}
}

// Test_Simple_Remove_Deletes checks Simple_Remove deletes and reports presence, and the oldest
// accessors return the LRU entry (Remove_Oldest also deleting it).
func Test_Simple_Remove_Deletes(t *testing.T) {
	c := lru.New_Simple[int, int](3, nil)
	lru.Simple_Add(c, 1, 1)
	lru.Simple_Add(c, 2, 2)
	lru.Simple_Add(c, 3, 3)
	if !lru.Simple_Remove(c, 2) {
		t.Fatalf("remove present key returned false")
	}
	if lru.Simple_Remove(c, 2) {
		t.Fatalf("remove absent key returned true")
	}
	oldest_key, oldest_value, ok := lru.Simple_Get_Oldest(c)
	if !ok {
		t.Fatalf("get oldest on non-empty cache reported empty")
	}
	if oldest_key != 1 {
		t.Fatalf("get oldest key = %d, want 1", oldest_key)
	}
	if oldest_value != 1 {
		t.Fatalf("get oldest value = %d, want 1", oldest_value)
	}
	removed_key, removed_value, ok := lru.Simple_Remove_Oldest(c)
	if !ok {
		t.Fatalf("remove oldest on non-empty cache reported empty")
	}
	if removed_key != 1 {
		t.Fatalf("remove oldest key = %d, want 1", removed_key)
	}
	if removed_value != 1 {
		t.Fatalf("remove oldest value = %d, want 1", removed_value)
	}
	if lru.Simple_Contains(c, 1) {
		t.Fatalf("remove oldest did not delete")
	}
}

// Test_Simple_Keys_And_Values_Fill_Buffer checks Simple_Keys and Simple_Values write entries
// oldest to newest into the caller's buffer and return the count, bounded by the buffer length.
func Test_Simple_Keys_And_Values_Fill_The_Buffer(t *testing.T) {
	c := lru.New_Simple[int, int](3, nil)
	lru.Simple_Add(c, 1, 10)
	lru.Simple_Add(c, 2, 20)
	lru.Simple_Add(c, 3, 30)
	keys := make([]int, lru.Simple_Cap(c))
	if n := lru.Simple_Keys(c, keys); !reflect.DeepEqual(keys[:n], []int{1, 2, 3}) {
		t.Fatalf("keys = %v, want [1 2 3]", keys[:n])
	}
	values := make([]int, lru.Simple_Cap(c))
	if n := lru.Simple_Values(c, values); !reflect.DeepEqual(values[:n], []int{10, 20, 30}) {
		t.Fatalf("values = %v, want [10 20 30]", values[:n])
	}
	// A short buffer bounds the read: it writes only what fits and reports that count.
	short := make([]int, 2)
	if n := lru.Simple_Keys(c, short); n != 2 {
		t.Fatalf("short read wrote %d, want 2", n)
	}
	if !reflect.DeepEqual(short, []int{1, 2}) {
		t.Fatalf("short read = %v, want [1 2]", short)
	}
}

// Test_Simple_Capacity_Is_Fixed checks capacity is a fixed, pre-allocated bound: a new key into a
// full cache evicts the oldest before inserting, Simple_Cap never changes, and the steady-state
// churn of Add and Get allocates nothing (the node pool and pre-sized map are committed up front).
func Test_Simple_Capacity_Is_Fixed(t *testing.T) {
	c := lru.New_Simple[int, int](4, nil)
	for i_index := 0; i_index < 4; i_index++ {
		lru.Simple_Add(c, i_index, i_index)
	}
	if !lru.Simple_Add(c, 4, 4) {
		t.Fatalf("add into a full cache did not evict")
	}
	if lru.Simple_Contains(c, 0) {
		t.Fatalf("oldest key 0 was not evicted")
	}
	if lru.Simple_Count(c) != 4 {
		t.Fatalf("count = %d, want 4", lru.Simple_Count(c))
	}
	if lru.Simple_Cap(c) != 4 {
		t.Fatalf("cap = %d, want 4 (capacity is immutable)", lru.Simple_Cap(c))
	}
	// A churn of new keys on the full cache must allocate nothing: nodes come from the pool and
	// the pre-sized map, holding at most Capacity entries, never grows.
	next := 0
	allocations := testing.AllocsPerRun(1000, func() {
		next++
		key := next % 64
		lru.Simple_Add(c, key, key)
		lru.Simple_Get(c, key)
	})
	if allocations != 0 {
		t.Fatalf("steady-state Add/Get allocated %v, want 0", allocations)
	}
}

// Test_Simple_Evict_Callback_Fires checks the Evict_Callback runs for every entry that an
// eviction, a removal, or Simple_Purge discards.
func Test_Simple_Evict_Callback_Fires(t *testing.T) {
	var evicted []int
	on_evict := func(key int, value int) { evicted = append(evicted, key) }
	c := lru.New_Simple[int, int](2, on_evict)
	lru.Simple_Add(c, 1, 1)
	lru.Simple_Add(c, 2, 2)
	lru.Simple_Add(c, 3, 3) // Evicts 1.
	lru.Simple_Remove(c, 2) // Removes 2.
	lru.Simple_Purge(c)     // Discards 3.
	if !reflect.DeepEqual(evicted, []int{1, 2, 3}) {
		t.Fatalf("evicted = %v, want [1 2 3]", evicted)
	}
}

// Test_Two_Queue_Promotes_Recent_To_Frequent checks a second access promotes a key from the
// recent list to the frequent list, where later one-off keys cannot evict it.
func Test_Two_Queue_Promotes_Recent_To_Frequent(t *testing.T) {
	c := lru.New_Two_Queue[int, int](lru.Two_Queue_Input{Capacity: 4})
	lru.Two_Queue_Add(c, 1, 1)
	// A get is the second access: it promotes 1 into the frequent list.
	value, ok := lru.Two_Queue_Get(c, 1)
	if !ok {
		t.Fatalf("get 1 missed")
	}
	if value != 1 {
		t.Fatalf("get 1 = %d, want 1", value)
	}
	// A run of one-off keys churns the recent list but must not evict frequent 1.
	for scan := 2; scan <= 9; scan++ {
		lru.Two_Queue_Add(c, scan, scan)
	}
	if !lru.Two_Queue_Contains(c, 1) {
		t.Fatalf("frequent key 1 was evicted by a scan")
	}
	peeked, ok := lru.Two_Queue_Peek(c, 1)
	if !ok {
		t.Fatalf("peek 1 missed after promotion")
	}
	if peeked != 1 {
		t.Fatalf("peek 1 = %d, want 1", peeked)
	}
}

// Test_Two_Queue_Resists_Scan checks keys promoted to the frequent list survive a burst of
// one-off keys that only churns the recent list.
func Test_Two_Queue_Resists_Scan(t *testing.T) {
	c := lru.New_Two_Queue[int, int](lru.Two_Queue_Input{Capacity: 4})
	// Promote 1 and 2 to the frequent list.
	lru.Two_Queue_Add(c, 1, 1)
	lru.Two_Queue_Get(c, 1)
	lru.Two_Queue_Add(c, 2, 2)
	lru.Two_Queue_Get(c, 2)
	// Scan a run of cold, one-off keys.
	for scan := 10; scan <= 20; scan++ {
		lru.Two_Queue_Add(c, scan, scan)
	}
	if !lru.Two_Queue_Contains(c, 1) {
		t.Fatalf("frequent key 1 did not survive the scan")
	}
	if !lru.Two_Queue_Contains(c, 2) {
		t.Fatalf("frequent key 2 did not survive the scan")
	}
}

// Test_Two_Queue_Ghost_Promotes_On_Re_Add checks a key evicted into the ghost list is promoted
// straight to the frequent list when re-added, so it then survives a scan.
func Test_Two_Queue_Ghost_Promotes_On_Re_Add(t *testing.T) {
	c := lru.New_Two_Queue[int, int](lru.Two_Queue_Input{Capacity: 4})
	lru.Two_Queue_Add(c, 1, 1)
	// Push 1 out of the recent list into the ghost list without ever promoting it. The ghost
	// list holds only size/2 = 2 keys, so keep the scan short enough that 1 is still a ghost.
	for scan := 2; scan <= 6; scan++ {
		lru.Two_Queue_Add(c, scan, scan)
	}
	if lru.Two_Queue_Contains(c, 1) {
		t.Fatalf("key 1 should have left the recent list for the ghost list")
	}
	// Re-adding a ghost key sends it to the frequent list.
	lru.Two_Queue_Add(c, 1, 1)
	for scan := 30; scan <= 40; scan++ {
		lru.Two_Queue_Add(c, scan, scan)
	}
	if !lru.Two_Queue_Contains(c, 1) {
		t.Fatalf("ghost-promoted key 1 did not survive the scan")
	}
}

// Test_Two_Queue_Reads_Do_Not_Mutate checks Two_Queue_Peek and Two_Queue_Contains do not promote
// a recent key, so a scan still evicts it, and that unset ratios take their defaults.
func Test_Two_Queue_Reads_Do_Not_Mutate(t *testing.T) {
	c := lru.New_Two_Queue[int, int](lru.Two_Queue_Input{Capacity: 4})
	if lru.Two_Queue_Cap(c) != 4 {
		t.Fatalf("cap = %d, want 4", lru.Two_Queue_Cap(c))
	}
	lru.Two_Queue_Add(c, 1, 1)
	value, ok := lru.Two_Queue_Peek(c, 1)
	if !ok {
		t.Fatalf("peek 1 missed")
	}
	if value != 1 {
		t.Fatalf("peek 1 = %d, want 1", value)
	}
	if !lru.Two_Queue_Contains(c, 1) {
		t.Fatalf("contains 1 = false")
	}
	// Peek and Contains must not have promoted 1, so a scan still evicts it from recent.
	for scan := 2; scan <= 9; scan++ {
		lru.Two_Queue_Add(c, scan, scan)
	}
	if lru.Two_Queue_Contains(c, 1) {
		t.Fatalf("peek/contains wrongly promoted key 1 past the scan")
	}
}

// Test_Expirable_Get_Rejects_Expired checks Expirable_Get and Expirable_Peek reject an entry once
// the injected clock has passed its TTL, advanced by driving the io loop.
func Test_Expirable_Get_Rejects_Expired(t *testing.T) {
	loop, driver, clock := io.New_Sim(1)
	c := lru.New_Expirable[int, int](lru.Expirable_Input[int, int]{
		Capacity: 4,
		TTL:      time.MICROSECOND,
		Clock:    clock,
		IO:       &loop,
	})
	lru.Expirable_Add(c, 1, 1)
	value, ok := lru.Expirable_Get(c, 1)
	if !ok {
		t.Fatalf("get 1 missed before expiry")
	}
	if value != 1 {
		t.Fatalf("get 1 = %d, want 1", value)
	}
	// Advance the loop well past the TTL; the cleanup timer rides this same timeline.
	driver.Run_For(4 * time.MICROSECOND)
	_, ok = lru.Expirable_Get(c, 1)
	if ok {
		t.Fatalf("get 1 returned an expired entry")
	}
	_, ok = lru.Expirable_Peek(c, 1)
	if ok {
		t.Fatalf("peek 1 returned an expired entry")
	}
}

// Test_Expirable_Timer_Reaps_Expired checks the repeating io.Timeout reaps expired entries as the
// harness drives the loop, walking the bucket ring with no manual sweep.
func Test_Expirable_Timer_Reaps_Expired(t *testing.T) {
	loop, driver, clock := io.New_Sim(1)
	c := lru.New_Expirable[int, int](lru.Expirable_Input[int, int]{
		Capacity: 8,
		TTL:      time.MICROSECOND,
		Clock:    clock,
		IO:       &loop,
	})
	lru.Expirable_Add(c, 1, 1)
	lru.Expirable_Add(c, 2, 2)
	if lru.Expirable_Count(c) != 2 {
		t.Fatalf("count = %d before expiry, want 2", lru.Expirable_Count(c))
	}
	// Drive until the timer has reaped both entries, capped so a bug cannot hang the test.
	reaped := func() (finished bool) {
		return lru.Expirable_Count(c) == 0
	}
	if !driver.Run_Until(reaped, time.MILLISECOND) {
		t.Fatalf("timer did not reap the expired entries")
	}
	if lru.Expirable_Contains(c, 1) {
		t.Fatalf("key 1 survived reaping")
	}
}

// Test_Expirable_Evicts_Oldest_By_Size checks Expirable still evicts the least-recently-used entry
// past its size, and that re-adding a key renews its recency.
func Test_Expirable_Evicts_Oldest_By_Size(t *testing.T) {
	loop, _, clock := io.New_Sim(1)
	c := lru.New_Expirable[int, int](lru.Expirable_Input[int, int]{
		Capacity: 2,
		TTL:      time.HOUR,
		Clock:    clock,
		IO:       &loop,
	})
	if lru.Expirable_Add(c, 1, 1) {
		t.Fatalf("add 1 under cap reported eviction")
	}
	lru.Expirable_Add(c, 2, 2)
	lru.Expirable_Add(c, 1, 11) // Re-adding renews 1's recency, so 2 becomes the oldest.
	if !lru.Expirable_Add(c, 3, 3) {
		t.Fatalf("add past cap did not evict")
	}
	if lru.Expirable_Contains(c, 2) {
		t.Fatalf("oldest key 2 was not evicted")
	}
	value, ok := lru.Expirable_Get(c, 1)
	if !ok {
		t.Fatalf("renewed key 1 missing")
	}
	if value != 11 {
		t.Fatalf("get 1 = %d, want 11", value)
	}
}

// Test_Expirable_Reproduces_Under_Virtual_Clock checks two Expirable caches driven by the same
// seeded io loop produce identical results, proving expiry reads only the injected clock.
func Test_Expirable_Reproduces_Under_Virtual_Clock(t *testing.T) {
	run := func() (results []bool) {
		loop, driver, clock := io.New_Sim(7)
		c := lru.New_Expirable[int, int](lru.Expirable_Input[int, int]{
			Capacity: 4,
			TTL:      time.MICROSECOND,
			Clock:    clock,
			IO:       &loop,
		})
		for step_index := 0; step_index < 8; step_index++ {
			lru.Expirable_Add(c, step_index, step_index)
			_, ok := lru.Expirable_Get(c, step_index/2)
			results = append(results, ok)
			driver.Run_For(200 * time.NANOSECOND)
		}
		return results
	}
	first := run()
	second := run()
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("expirable diverged across identical runs: %v vs %v", first, second)
	}
}
