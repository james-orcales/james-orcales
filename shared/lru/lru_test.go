// Regression tests ported from github.com/hashicorp/golang-lru, adapted to the deterministic
// house port: randomness comes from the seeded prng rather than crypto/rand, expiry rides a
// simulated io loop rather than wall time and a reaper goroutine, and the upstream lru.Cache
// tests fold onto Simple because the thread-safe wrapper collapsed into it.
//
// TestLRUConcurrency is ported as Test_Expirable_Concurrent_Adds: determinism forbids goroutines,
// not concurrency, so its racing overlapping adds become a deterministic interleaving of the same
// operations that must converge to the same state. Three upstream tests have no analog and are
// not ported: TestLRUInterface (the LRUCache interface was removed), and the goroutine-lifecycle
// pair TestCache_CloseGoRoutine and TestCache_RestartGoRoutine (they toggle a background reaper
// this port does not have — expiry is lazy on read, and reaping rides a repeating io.Timeout the
// harness's loop fires). The ARC cache was not ported at all, so its tests are absent too.
package lru_test

import (
	"fmt"
	"reflect"
	"testing"

	"local/james-orcales/shared/io"
	"local/james-orcales/shared/lru"
	"local/james-orcales/shared/random/prng"
	"local/james-orcales/shared/time"
)

// Builds a fresh simulated io loop for the Expirable ports: the submit surface to inject, the
// driver the test pumps, and the clock expiry is stamped against.
func new_expirable_loop() (loop io.IO, driver io.Driver, clock time.Clock) {
	return io.New_Sim(1)
}

// Collects a Simple's keys into a fresh slice through the caller-buffer API, for comparison.
func simple_keys_of[K comparable, V any](c *lru.Simple[K, V]) (keys []K) {
	keys = make([]K, lru.Simple_Cap(c))
	return keys[:lru.Simple_Keys(c, keys)]
}

// Collects a Simple's values into a fresh slice through the caller-buffer API.
func simple_values_of[K comparable, V any](c *lru.Simple[K, V]) (values []V) {
	values = make([]V, lru.Simple_Cap(c))
	return values[:lru.Simple_Values(c, values)]
}

// Collects a Two_Queue's keys into a fresh slice through the caller-buffer API.
func two_queue_keys_of[K comparable, V any](c *lru.Two_Queue[K, V]) (keys []K) {
	keys = make([]K, lru.Two_Queue_Cap(c))
	return keys[:lru.Two_Queue_Keys(c, keys)]
}

// Collects an Expirable's unexpired keys into a fresh slice through the caller-buffer API.
func expirable_keys_of[K comparable, V any](c *lru.Expirable[K, V]) (keys []K) {
	keys = make([]K, lru.Expirable_Cap(c))
	return keys[:lru.Expirable_Keys(c, keys)]
}

// Collects an Expirable's unexpired values into a fresh slice through the caller-buffer API.
func expirable_values_of[K comparable, V any](c *lru.Expirable[K, V]) (values []V) {
	values = make([]V, lru.Expirable_Cap(c))
	return values[:lru.Expirable_Values(c, values)]
}

// Test_Simple_LRU_Full ports the fill-and-evict half of simplelru TestLRU / lru TestLRU: eviction,
// the evict callback, and key/value ordering.
func Test_Simple_LRU_Full(t *testing.T) {
	evict_count := 0
	on_evict := func(key int, value int) {
		if key != value {
			t.Fatalf("evict values not equal (%v != %v)", key, value)
		}
		evict_count++
	}
	c := lru.New_Simple[int, int](128, on_evict)
	for i_index := 0; i_index < 256; i_index++ {
		lru.Simple_Add(c, i_index, i_index)
	}
	if lru.Simple_Count(c) != 128 {
		t.Fatalf("bad count: %v", lru.Simple_Count(c))
	}
	if lru.Simple_Cap(c) != 128 {
		t.Fatalf("bad cap: %v", lru.Simple_Cap(c))
	}
	if evict_count != 128 {
		t.Fatalf("bad evict count: %v", evict_count)
	}
	for i_index, key := range simple_keys_of(c) {
		value, ok := lru.Simple_Get(c, key)
		if !ok {
			t.Fatalf("bad key: %v", key)
		}
		if value != key {
			t.Fatalf("bad key: %v", key)
		}
		if value != i_index+128 {
			t.Fatalf("bad key: %v", key)
		}
	}
	for i_index, value := range simple_values_of(c) {
		if value != i_index+128 {
			t.Fatalf("bad value: %v", value)
		}
	}
}

// Test_Simple_LRU_Order_After_Remove ports the remove-and-purge half of TestLRU: removal, the
// resulting key ordering after a get renews recency, and purge.
func Test_Simple_LRU_Order_After_Remove(t *testing.T) {
	c := lru.New_Simple[int, int](128, nil)
	for i_index := 0; i_index < 256; i_index++ {
		lru.Simple_Add(c, i_index, i_index)
	}
	for i_index := 0; i_index < 128; i_index++ {
		_, ok := lru.Simple_Get(c, i_index)
		if ok {
			t.Fatalf("%d should be evicted", i_index)
		}
	}
	for i_index := 128; i_index < 192; i_index++ {
		if !lru.Simple_Remove(c, i_index) {
			t.Fatalf("%d should be contained", i_index)
		}
		if lru.Simple_Remove(c, i_index) {
			t.Fatalf("%d should not be contained", i_index)
		}
	}
	lru.Simple_Get(c, 192) // 192 becomes most-recently-used, so it is last in Keys.
	for i_index, key := range simple_keys_of(c) {
		if i_index < 63 {
			if key != i_index+193 {
				t.Fatalf("out of order key: %v", key)
			}
		}
		if i_index == 63 {
			if key != 192 {
				t.Fatalf("out of order key: %v", key)
			}
		}
	}
	lru.Simple_Purge(c)
	if lru.Simple_Count(c) != 0 {
		t.Fatalf("bad count: %v", lru.Simple_Count(c))
	}
	_, ok := lru.Simple_Get(c, 200)
	if ok {
		t.Fatalf("should contain nothing")
	}
}

// Test_Simple_Get_Oldest_Remove_Oldest ports TestLRU_GetOldest_RemoveOldest.
func Test_Simple_Get_Oldest_Remove_Oldest(t *testing.T) {
	c := lru.New_Simple[int, int](128, nil)
	for i_index := 0; i_index < 256; i_index++ {
		lru.Simple_Add(c, i_index, i_index)
	}
	key, _, ok := lru.Simple_Get_Oldest(c)
	if !ok {
		t.Fatalf("missing")
	}
	if key != 128 {
		t.Fatalf("bad: %v", key)
	}
	key, _, ok = lru.Simple_Remove_Oldest(c)
	if !ok {
		t.Fatalf("missing")
	}
	if key != 128 {
		t.Fatalf("bad: %v", key)
	}
	key, _, ok = lru.Simple_Remove_Oldest(c)
	if !ok {
		t.Fatalf("missing")
	}
	if key != 129 {
		t.Fatalf("bad: %v", key)
	}
}

// Test_Simple_Add_Reports_Eviction ports TestLRU_Add / TestLRUAdd.
func Test_Simple_Add_Reports_Eviction(t *testing.T) {
	evict_count := 0
	on_evict := func(key int, value int) { evict_count++ }
	c := lru.New_Simple[int, int](1, on_evict)
	if lru.Simple_Add(c, 1, 1) {
		t.Errorf("should not have an eviction")
	}
	if evict_count != 0 {
		t.Errorf("should not have an eviction")
	}
	if !lru.Simple_Add(c, 2, 2) {
		t.Errorf("should have an eviction")
	}
	if evict_count != 1 {
		t.Errorf("should have an eviction")
	}
}

// Test_Simple_Contains_Leaves_Recency ports TestLRU_Contains / TestLRUContains.
func Test_Simple_Contains_Leaves_Recency(t *testing.T) {
	c := lru.New_Simple[int, int](2, nil)
	lru.Simple_Add(c, 1, 1)
	lru.Simple_Add(c, 2, 2)
	if !lru.Simple_Contains(c, 1) {
		t.Errorf("1 should be contained")
	}
	lru.Simple_Add(c, 3, 3)
	if lru.Simple_Contains(c, 1) {
		t.Errorf("contains should not have renewed recency of 1")
	}
}

// Test_Simple_Peek_Leaves_Recency ports TestLRU_Peek / TestLRUPeek.
func Test_Simple_Peek_Leaves_Recency(t *testing.T) {
	c := lru.New_Simple[int, int](2, nil)
	lru.Simple_Add(c, 1, 1)
	lru.Simple_Add(c, 2, 2)
	value, ok := lru.Simple_Peek(c, 1)
	if !ok {
		t.Errorf("1 should be set")
	}
	if value != 1 {
		t.Errorf("1 should be 1: %v", value)
	}
	lru.Simple_Add(c, 3, 3)
	if lru.Simple_Contains(c, 1) {
		t.Errorf("peek should not have renewed recency of 1")
	}
}

// Test_Simple_Contains_Or_Add_Leaves_Recency ports TestLRUContainsOrAdd.
func Test_Simple_Contains_Or_Add_Leaves_Recency(t *testing.T) {
	c := lru.New_Simple[int, int](2, nil)
	lru.Simple_Add(c, 1, 1)
	lru.Simple_Add(c, 2, 2)
	present, evicted := lru.Simple_Contains_Or_Add(c, 1, 1)
	if !present {
		t.Errorf("1 should be contained")
	}
	if evicted {
		t.Errorf("nothing should be evicted")
	}
	lru.Simple_Add(c, 3, 3)
	present, evicted = lru.Simple_Contains_Or_Add(c, 1, 1)
	if present {
		t.Errorf("1 should not be contained")
	}
	if !evicted {
		t.Errorf("an eviction should have occurred")
	}
	if !lru.Simple_Contains(c, 1) {
		t.Errorf("now 1 should be contained")
	}
}

// Test_Simple_Peek_Or_Add_Leaves_Recency ports TestLRUPeekOrAdd.
func Test_Simple_Peek_Or_Add_Leaves_Recency(t *testing.T) {
	c := lru.New_Simple[int, int](2, nil)
	lru.Simple_Add(c, 1, 1)
	lru.Simple_Add(c, 2, 2)
	previous, present, evicted := lru.Simple_Peek_Or_Add(c, 1, 1)
	if !present {
		t.Errorf("1 should be contained")
	}
	if evicted {
		t.Errorf("nothing should be evicted")
	}
	if previous != 1 {
		t.Errorf("previous should be 1")
	}
	lru.Simple_Add(c, 3, 3)
	present, evicted = lru.Simple_Contains_Or_Add(c, 1, 1)
	if present {
		t.Errorf("1 should not be contained")
	}
	if !evicted {
		t.Errorf("an eviction should have occurred")
	}
	if !lru.Simple_Contains(c, 1) {
		t.Errorf("now 1 should be contained")
	}
}

// Test_Simple_Eviction_Same_Key_Add ports the Add case of TestCache_EvictionSameKey: re-adding a
// present key renews recency without a spurious eviction.
func Test_Simple_Eviction_Same_Key_Add(t *testing.T) {
	var evicted_keys []int
	c := lru.New_Simple[int, struct{}](2, func(key int, value struct{}) {
		evicted_keys = append(evicted_keys, key)
	})
	if lru.Simple_Add(c, 1, struct{}{}) {
		t.Error("first 1: unexpected eviction")
	}
	if !reflect.DeepEqual(simple_keys_of(c), []int{1}) {
		t.Errorf("keys: %v", simple_keys_of(c))
	}
	if lru.Simple_Add(c, 2, struct{}{}) {
		t.Error("2: unexpected eviction")
	}
	if !reflect.DeepEqual(simple_keys_of(c), []int{1, 2}) {
		t.Errorf("keys: %v", simple_keys_of(c))
	}
	if lru.Simple_Add(c, 1, struct{}{}) {
		t.Error("second 1: unexpected eviction")
	}
	if !reflect.DeepEqual(simple_keys_of(c), []int{2, 1}) {
		t.Errorf("keys: %v", simple_keys_of(c))
	}
	if !lru.Simple_Add(c, 3, struct{}{}) {
		t.Error("3: expected eviction")
	}
	if !reflect.DeepEqual(simple_keys_of(c), []int{1, 3}) {
		t.Errorf("keys: %v", simple_keys_of(c))
	}
	if !reflect.DeepEqual(evicted_keys, []int{2}) {
		t.Errorf("evicted keys: %v", evicted_keys)
	}
}

// Test_Simple_Eviction_Same_Key_Contains_Or_Add ports the ContainsOrAdd case of
// TestCache_EvictionSameKey.
func Test_Simple_Eviction_Same_Key_Contains_Or_Add(t *testing.T) {
	var evicted_keys []int
	c := lru.New_Simple[int, struct{}](2, func(key int, value struct{}) {
		evicted_keys = append(evicted_keys, key)
	})
	present, evicted := lru.Simple_Contains_Or_Add(c, 1, struct{}{})
	if present {
		t.Error("first 1: unexpected contained")
	}
	if evicted {
		t.Error("first 1: unexpected eviction")
	}
	present, evicted = lru.Simple_Contains_Or_Add(c, 2, struct{}{})
	if present {
		t.Error("2: unexpected contained")
	}
	if evicted {
		t.Error("2: unexpected eviction")
	}
	present, evicted = lru.Simple_Contains_Or_Add(c, 1, struct{}{})
	if !present {
		t.Error("second 1: expected contained")
	}
	if evicted {
		t.Error("second 1: unexpected eviction")
	}
	if !reflect.DeepEqual(simple_keys_of(c), []int{1, 2}) {
		t.Errorf("keys: %v", simple_keys_of(c))
	}
	present, evicted = lru.Simple_Contains_Or_Add(c, 3, struct{}{})
	if present {
		t.Error("3: unexpected contained")
	}
	if !evicted {
		t.Error("3: expected eviction")
	}
	if !reflect.DeepEqual(evicted_keys, []int{1}) {
		t.Errorf("evicted keys: %v", evicted_keys)
	}
}

// Test_Simple_Eviction_Same_Key_Peek_Or_Add ports the PeekOrAdd case of TestCache_EvictionSameKey.
func Test_Simple_Eviction_Same_Key_Peek_Or_Add(t *testing.T) {
	var evicted_keys []int
	c := lru.New_Simple[int, struct{}](2, func(key int, value struct{}) {
		evicted_keys = append(evicted_keys, key)
	})
	_, present, evicted := lru.Simple_Peek_Or_Add(c, 1, struct{}{})
	if present {
		t.Error("first 1: unexpected contained")
	}
	if evicted {
		t.Error("first 1: unexpected eviction")
	}
	_, present, evicted = lru.Simple_Peek_Or_Add(c, 2, struct{}{})
	if present {
		t.Error("2: unexpected contained")
	}
	if evicted {
		t.Error("2: unexpected eviction")
	}
	_, present, evicted = lru.Simple_Peek_Or_Add(c, 1, struct{}{})
	if !present {
		t.Error("second 1: expected contained")
	}
	if evicted {
		t.Error("second 1: unexpected eviction")
	}
	if !reflect.DeepEqual(simple_keys_of(c), []int{1, 2}) {
		t.Errorf("keys: %v", simple_keys_of(c))
	}
	_, present, evicted = lru.Simple_Peek_Or_Add(c, 3, struct{}{})
	if present {
		t.Error("3: unexpected contained")
	}
	if !evicted {
		t.Error("3: expected eviction")
	}
	if !reflect.DeepEqual(evicted_keys, []int{1}) {
		t.Errorf("evicted keys: %v", evicted_keys)
	}
}

// Test_Two_Queue_Random_Ops ports Test2Q_RandomOps: the live count never exceeds the size across
// a long run of adds, gets, and removes.
func Test_Two_Queue_Random_Ops(t *testing.T) {
	size := 128
	generator := prng.New(3)
	c := lru.New_Two_Queue[int, int](lru.Two_Queue_Input{Capacity: size})
	for op_index := 0; op_index < 200000; op_index++ {
		key := prng.Generator_Below(&generator, 512)
		switch prng.Generator_Below(&generator, 3) {
		case 0:
			lru.Two_Queue_Add(c, key, key)
		case 1:
			lru.Two_Queue_Get(c, key)
		case 2:
			lru.Two_Queue_Remove(c, key)
		}
		live := lru.Simple_Count(c.Recent) + lru.Simple_Count(c.Frequent)
		if live > size {
			t.Fatalf("live %d exceeds size %d", live, size)
		}
	}
}

// Test_Two_Queue_Get_Recent_To_Frequent ports Test2Q_Get_RecentToFrequent.
func Test_Two_Queue_Get_Recent_To_Frequent(t *testing.T) {
	c := lru.New_Two_Queue[int, int](lru.Two_Queue_Input{Capacity: 128})
	for i_index := 0; i_index < 128; i_index++ {
		lru.Two_Queue_Add(c, i_index, i_index)
	}
	if lru.Simple_Count(c.Recent) != 128 {
		t.Fatalf("bad recent: %d", lru.Simple_Count(c.Recent))
	}
	if lru.Simple_Count(c.Frequent) != 0 {
		t.Fatalf("bad frequent: %d", lru.Simple_Count(c.Frequent))
	}
	for i_index := 0; i_index < 128; i_index++ {
		_, ok := lru.Two_Queue_Get(c, i_index)
		if !ok {
			t.Fatalf("missing: %d", i_index)
		}
	}
	if lru.Simple_Count(c.Recent) != 0 {
		t.Fatalf("bad recent: %d", lru.Simple_Count(c.Recent))
	}
	if lru.Simple_Count(c.Frequent) != 128 {
		t.Fatalf("bad frequent: %d", lru.Simple_Count(c.Frequent))
	}
}

// Test_Two_Queue_Add_Recent_To_Frequent ports Test2Q_Add_RecentToFrequent.
func Test_Two_Queue_Add_Recent_To_Frequent(t *testing.T) {
	c := lru.New_Two_Queue[int, int](lru.Two_Queue_Input{Capacity: 128})
	lru.Two_Queue_Add(c, 1, 1)
	if lru.Simple_Count(c.Recent) != 1 {
		t.Fatalf("bad recent: %d", lru.Simple_Count(c.Recent))
	}
	if lru.Simple_Count(c.Frequent) != 0 {
		t.Fatalf("bad frequent: %d", lru.Simple_Count(c.Frequent))
	}
	lru.Two_Queue_Add(c, 1, 1)
	if lru.Simple_Count(c.Recent) != 0 {
		t.Fatalf("bad recent: %d", lru.Simple_Count(c.Recent))
	}
	if lru.Simple_Count(c.Frequent) != 1 {
		t.Fatalf("bad frequent: %d", lru.Simple_Count(c.Frequent))
	}
	lru.Two_Queue_Add(c, 1, 1)
	if lru.Simple_Count(c.Frequent) != 1 {
		t.Fatalf("bad frequent: %d", lru.Simple_Count(c.Frequent))
	}
}

// Test_Two_Queue_Add_Recent_Evict ports Test2Q_Add_RecentEvict.
func Test_Two_Queue_Add_Recent_Evict(t *testing.T) {
	c := lru.New_Two_Queue[int, int](lru.Two_Queue_Input{Capacity: 4})
	lru.Two_Queue_Add(c, 1, 1)
	lru.Two_Queue_Add(c, 2, 2)
	lru.Two_Queue_Add(c, 3, 3)
	lru.Two_Queue_Add(c, 4, 4)
	lru.Two_Queue_Add(c, 5, 5)
	if lru.Simple_Count(c.Recent) != 4 {
		t.Fatalf("bad recent: %d", lru.Simple_Count(c.Recent))
	}
	if lru.Simple_Count(c.Recent_Evict) != 1 {
		t.Fatalf("bad ghost: %d", lru.Simple_Count(c.Recent_Evict))
	}
	lru.Two_Queue_Add(c, 1, 1)
	if lru.Simple_Count(c.Recent) != 3 {
		t.Fatalf("bad recent: %d", lru.Simple_Count(c.Recent))
	}
	if lru.Simple_Count(c.Frequent) != 1 {
		t.Fatalf("bad frequent: %d", lru.Simple_Count(c.Frequent))
	}
	lru.Two_Queue_Add(c, 6, 6)
	if lru.Simple_Count(c.Recent) != 3 {
		t.Fatalf("bad recent: %d", lru.Simple_Count(c.Recent))
	}
	if lru.Simple_Count(c.Recent_Evict) != 2 {
		t.Fatalf("bad ghost: %d", lru.Simple_Count(c.Recent_Evict))
	}
}

// Test_Two_Queue_Full ports Test2Q: a full lifecycle of eviction, ordering, removal, and purge.
func Test_Two_Queue_Full(t *testing.T) {
	c := lru.New_Two_Queue[int, int](lru.Two_Queue_Input{Capacity: 128})
	for i_index := 0; i_index < 256; i_index++ {
		lru.Two_Queue_Add(c, i_index, i_index)
	}
	if lru.Two_Queue_Count(c) != 128 {
		t.Fatalf("bad count: %v", lru.Two_Queue_Count(c))
	}
	if lru.Two_Queue_Cap(c) != 128 {
		t.Fatalf("bad cap: %v", lru.Two_Queue_Cap(c))
	}
	for i_index, key := range two_queue_keys_of(c) {
		value, ok := lru.Two_Queue_Get(c, key)
		if !ok {
			t.Fatalf("bad key: %v", key)
		}
		if value != key {
			t.Fatalf("bad key: %v", key)
		}
		if value != i_index+128 {
			t.Fatalf("bad key: %v", key)
		}
	}
	for i_index := 0; i_index < 128; i_index++ {
		_, ok := lru.Two_Queue_Get(c, i_index)
		if ok {
			t.Fatalf("%d should be evicted", i_index)
		}
	}
	lru.Two_Queue_Purge(c)
	if lru.Two_Queue_Count(c) != 0 {
		t.Fatalf("bad count: %v", lru.Two_Queue_Count(c))
	}
}

// Test_Two_Queue_Contains ports Test2Q_Contains.
func Test_Two_Queue_Contains(t *testing.T) {
	c := lru.New_Two_Queue[int, int](lru.Two_Queue_Input{Capacity: 2})
	lru.Two_Queue_Add(c, 1, 1)
	lru.Two_Queue_Add(c, 2, 2)
	if !lru.Two_Queue_Contains(c, 1) {
		t.Errorf("1 should be contained")
	}
	lru.Two_Queue_Add(c, 3, 3)
	if lru.Two_Queue_Contains(c, 1) {
		t.Errorf("contains should not have renewed recency of 1")
	}
}

// Test_Two_Queue_Peek ports Test2Q_Peek.
func Test_Two_Queue_Peek(t *testing.T) {
	c := lru.New_Two_Queue[int, int](lru.Two_Queue_Input{Capacity: 2})
	lru.Two_Queue_Add(c, 1, 1)
	lru.Two_Queue_Add(c, 2, 2)
	value, ok := lru.Two_Queue_Peek(c, 1)
	if !ok {
		t.Errorf("1 should be set")
	}
	if value != 1 {
		t.Errorf("1 should be 1: %v", value)
	}
	lru.Two_Queue_Add(c, 3, 3)
	if lru.Two_Queue_Contains(c, 1) {
		t.Errorf("peek should not have renewed recency of 1")
	}
}

// Test_Expirable_No_Purge ports TestLRUNoPurge: a long-TTL cache serves an entry without expiring
// it and reports membership and keys correctly. The upstream ttl=0 and Resize modes were removed.
func Test_Expirable_No_Purge(t *testing.T) {
	loop, _, clock := new_expirable_loop()
	c := lru.New_Expirable[string, string](lru.Expirable_Input[string, string]{
		Capacity: 10,
		TTL:      time.HOUR,
		Clock:    clock,
		IO:       &loop,
	})
	lru.Expirable_Add(c, "key1", "val1")
	if lru.Expirable_Count(c) != 1 {
		t.Fatalf("count differs from expected")
	}
	value, ok := lru.Expirable_Peek(c, "key1")
	if value != "val1" {
		t.Fatalf("value differs from expected")
	}
	if !ok {
		t.Fatalf("should be true")
	}
	if !lru.Expirable_Contains(c, "key1") {
		t.Fatalf("should contain key1")
	}
	if lru.Expirable_Contains(c, "key2") {
		t.Fatalf("should not contain key2")
	}
	if !reflect.DeepEqual(expirable_keys_of(c), []string{"key1"}) {
		t.Fatalf("keys differ from expected")
	}
}

// Test_Expirable_Edge_Cases ports TestLRUEdgeCases: a nil value stores and reads back, and an
// overwrite replaces it.
func Test_Expirable_Edge_Cases(t *testing.T) {
	loop, _, clock := new_expirable_loop()
	c := lru.New_Expirable[string, *string](lru.Expirable_Input[string, *string]{
		Capacity: 2,
		TTL:      time.HOUR,
		Clock:    clock,
		IO:       &loop,
	})
	lru.Expirable_Add(c, "key1", nil)
	value, exists := lru.Expirable_Get(c, "key1")
	if value != nil {
		t.Fatalf("unexpected value for key1: %v", value)
	}
	if !exists {
		t.Fatalf("key1 should exist")
	}
	new_value := "val1"
	lru.Expirable_Add(c, "key1", &new_value)
	value, exists = lru.Expirable_Get(c, "key1")
	if value != &new_value {
		t.Fatalf("unexpected value for key1: %v", value)
	}
	if !exists {
		t.Fatalf("key1 should exist")
	}
}

// Test_Expirable_Values ports TestLRU_Values.
func Test_Expirable_Values(t *testing.T) {
	loop, _, clock := new_expirable_loop()
	c := lru.New_Expirable[string, string](lru.Expirable_Input[string, string]{
		Capacity: 3,
		TTL:      time.HOUR,
		Clock:    clock,
		IO:       &loop,
	})
	lru.Expirable_Add(c, "key1", "val1")
	lru.Expirable_Add(c, "key2", "val2")
	lru.Expirable_Add(c, "key3", "val3")
	values := expirable_values_of(c)
	if !reflect.DeepEqual(values, []string{"val1", "val2", "val3"}) {
		t.Fatalf("values differ from expected: %v", values)
	}
}

// Test_Expirable_With_Purge_Expiry ports the expiry half of TestLRUWithPurge, driving the io loop
// so the repeating timer reaps the expired entry and fires the callback.
func Test_Expirable_With_Purge_Expiry(t *testing.T) {
	loop, driver, clock := new_expirable_loop()
	var evicted []string
	c := lru.New_Expirable[string, string](lru.Expirable_Input[string, string]{
		Capacity: 10,
		TTL:      time.MICROSECOND,
		Clock:    clock,
		IO:       &loop,
		On_Evict: func(key string, value string) { evicted = append(evicted, key, value) },
	})
	lru.Expirable_Add(c, "key1", "val1")
	// A short drive stays under the TTL, so the entry is still live.
	driver.Run_For(200 * time.NANOSECOND)
	value, ok := lru.Expirable_Get(c, "key1")
	if value != "val1" {
		t.Fatalf("value differs from expected")
	}
	if !ok {
		t.Fatalf("should be true")
	}
	// Drive until the timer reaps the expired entry.
	empty := func() (finished bool) {
		return lru.Expirable_Count(c) == 0
	}
	if !driver.Run_Until(empty, time.MILLISECOND) {
		t.Fatalf("timer did not reap the expired entry")
	}
	if !reflect.DeepEqual(evicted, []string{"key1", "val1"}) {
		t.Fatalf("evicted differs from expected: %v", evicted)
	}
}

// Test_Expirable_Purge_Fires_Callback ports the Purge tail of TestLRUWithPurge: an undriven timer
// leaves live entries in place, and Purge clears the cache and fires the callback for each.
func Test_Expirable_Purge_Fires_Callback(t *testing.T) {
	loop, _, clock := new_expirable_loop()
	var evicted []string
	c := lru.New_Expirable[string, string](lru.Expirable_Input[string, string]{
		Capacity: 10,
		TTL:      time.HOUR,
		Clock:    clock,
		IO:       &loop,
		On_Evict: func(key string, value string) { evicted = append(evicted, key, value) },
	})
	lru.Expirable_Add(c, "key1", "val1")
	lru.Expirable_Add(c, "key2", "val2")
	if lru.Expirable_Count(c) != 2 {
		t.Fatalf("count differs from expected")
	}
	lru.Expirable_Purge(c)
	if lru.Expirable_Count(c) != 0 {
		t.Fatalf("count differs from expected")
	}
	if !reflect.DeepEqual(evicted, []string{"key1", "val1", "key2", "val2"}) {
		t.Fatalf("evicted differs from expected: %v", evicted)
	}
}

// Test_Expirable_Purge_Enforced_By_Size ports TestLRUWithPurgeEnforcedBySize.
func Test_Expirable_Purge_Enforced_By_Size(t *testing.T) {
	loop, _, clock := new_expirable_loop()
	c := lru.New_Expirable[string, string](lru.Expirable_Input[string, string]{
		Capacity: 10,
		TTL:      time.HOUR,
		Clock:    clock,
		IO:       &loop,
	})
	for i_index := 0; i_index < 100; i_index++ {
		key := fmt.Sprintf("key%d", i_index)
		value := fmt.Sprintf("val%d", i_index)
		lru.Expirable_Add(c, key, value)
		got, ok := lru.Expirable_Get(c, key)
		if got != value {
			t.Fatalf("value differs from expected")
		}
		if !ok {
			t.Fatalf("should be true")
		}
		if lru.Expirable_Count(c) > 20 {
			t.Fatalf("count should be less than 20")
		}
	}
	if lru.Expirable_Count(c) != 10 {
		t.Fatalf("count differs from expected")
	}
}

// Test_Expirable_Invalidate_And_Evict ports TestLRUInvalidateAndEvict: a remove fires the eviction
// callback. The upstream size=-1 unlimited mode was removed, so a bounded size stands in.
func Test_Expirable_Invalidate_And_Evict(t *testing.T) {
	loop, _, clock := new_expirable_loop()
	evicted := 0
	c := lru.New_Expirable[string, string](lru.Expirable_Input[string, string]{
		Capacity: 10,
		TTL:      time.HOUR,
		Clock:    clock,
		IO:       &loop,
		On_Evict: func(key string, value string) { evicted++ },
	})
	lru.Expirable_Add(c, "key1", "val1")
	lru.Expirable_Add(c, "key2", "val2")
	value, ok := lru.Expirable_Get(c, "key1")
	if !ok {
		t.Fatalf("should be true")
	}
	if value != "val1" {
		t.Fatalf("value differs from expected")
	}
	if evicted != 0 {
		t.Fatalf("nothing should be evicted yet")
	}
	lru.Expirable_Remove(c, "key1")
	if evicted != 1 {
		t.Fatalf("remove should have evicted one")
	}
	_, ok = lru.Expirable_Get(c, "key1")
	if ok {
		t.Fatalf("removed key1 should read as a miss")
	}
}

// Test_Expirable_Loading_Expired ports TestLoadingExpired: reads reject an entry once its TTL has
// passed on the virtual clock.
func Test_Expirable_Loading_Expired(t *testing.T) {
	loop, driver, clock := new_expirable_loop()
	c := lru.New_Expirable[string, string](lru.Expirable_Input[string, string]{
		Capacity: 8,
		TTL:      time.MICROSECOND,
		Clock:    clock,
		IO:       &loop,
	})
	lru.Expirable_Add(c, "key1", "val1")
	value, ok := lru.Expirable_Peek(c, "key1")
	if value != "val1" {
		t.Fatalf("value differs from expected")
	}
	if !ok {
		t.Fatalf("should be true")
	}
	value, ok = lru.Expirable_Get(c, "key1")
	if value != "val1" {
		t.Fatalf("value differs from expected")
	}
	if !ok {
		t.Fatalf("should be true")
	}
	driver.Run_For(4 * time.MICROSECOND) // Past the TTL.
	_, ok = lru.Expirable_Peek(c, "key1")
	if ok {
		t.Fatalf("expired key1 should peek as a miss")
	}
	_, ok = lru.Expirable_Get(c, "key1")
	if ok {
		t.Fatalf("expired key1 should get as a miss")
	}
}

// Test_Expirable_Remove_Oldest ports TestLRURemoveOldest.
func Test_Expirable_Remove_Oldest(t *testing.T) {
	loop, _, clock := new_expirable_loop()
	c := lru.New_Expirable[string, string](lru.Expirable_Input[string, string]{
		Capacity: 2,
		TTL:      time.HOUR,
		Clock:    clock,
		IO:       &loop,
	})
	if lru.Expirable_Cap(c) != 2 {
		t.Fatalf("expected cap 2")
	}
	key, _, ok := lru.Expirable_Remove_Oldest(c)
	if key != "" {
		t.Fatalf("should be empty")
	}
	if ok {
		t.Fatalf("should be false")
	}
	if lru.Expirable_Remove(c, "non_existent") {
		t.Fatalf("should be false")
	}
	lru.Expirable_Add(c, "key1", "val1")
	lru.Expirable_Add(c, "key2", "val2")
	if !reflect.DeepEqual(expirable_keys_of(c), []string{"key1", "key2"}) {
		t.Fatalf("keys differ from expected")
	}
	key, value, ok := lru.Expirable_Remove_Oldest(c)
	if key != "key1" {
		t.Fatalf("key differs from expected")
	}
	if value != "val1" {
		t.Fatalf("value differs from expected")
	}
	if !ok {
		t.Fatalf("should be true")
	}
	if !reflect.DeepEqual(expirable_keys_of(c), []string{"key2"}) {
		t.Fatalf("keys differ from expected")
	}
}

// Test_Expirable_Eviction_Same_Key ports the expirable TestCache_EvictionSameKey.
func Test_Expirable_Eviction_Same_Key(t *testing.T) {
	loop, _, clock := new_expirable_loop()
	var evicted_keys []int
	record := func(key int, value struct{}) { evicted_keys = append(evicted_keys, key) }
	c := lru.New_Expirable[int, struct{}](lru.Expirable_Input[int, struct{}]{
		Capacity: 2,
		TTL:      time.HOUR,
		Clock:    clock,
		IO:       &loop,
		On_Evict: record,
	})
	if lru.Expirable_Add(c, 1, struct{}{}) {
		t.Error("first 1: unexpected eviction")
	}
	if lru.Expirable_Add(c, 2, struct{}{}) {
		t.Error("2: unexpected eviction")
	}
	if !reflect.DeepEqual(expirable_keys_of(c), []int{1, 2}) {
		t.Errorf("keys: %v", expirable_keys_of(c))
	}
	if lru.Expirable_Add(c, 1, struct{}{}) {
		t.Error("second 1: unexpected eviction")
	}
	if !reflect.DeepEqual(expirable_keys_of(c), []int{2, 1}) {
		t.Errorf("keys: %v", expirable_keys_of(c))
	}
	if !lru.Expirable_Add(c, 3, struct{}{}) {
		t.Error("3: expected eviction")
	}
	if !reflect.DeepEqual(evicted_keys, []int{2}) {
		t.Errorf("evicted keys: %v", evicted_keys)
	}
}

// Test_Expirable_Lifecycle ports ExampleLRU as a test: a hit before expiry, a miss after, and a
// count of one once the expired entry is reaped and a fresh key is added.
func Test_Expirable_Lifecycle(t *testing.T) {
	loop, driver, clock := new_expirable_loop()
	c := lru.New_Expirable[string, string](lru.Expirable_Input[string, string]{
		Capacity: 5,
		TTL:      time.MICROSECOND,
		Clock:    clock,
		IO:       &loop,
	})
	lru.Expirable_Add(c, "key1", "val1")
	value, ok := lru.Expirable_Get(c, "key1")
	if !ok {
		t.Fatalf("key1 should be found before expiration")
	}
	if value != "val1" {
		t.Fatalf("value differs from expected")
	}
	// Drive until the timer reaps the expired key1.
	gone := func() (finished bool) {
		return lru.Expirable_Count(c) == 0
	}
	if !driver.Run_Until(gone, time.MILLISECOND) {
		t.Fatalf("timer did not reap key1")
	}
	lru.Expirable_Add(c, "key2", "val2")
	if lru.Expirable_Count(c) != 1 {
		t.Fatalf("count = %d, want 1", lru.Expirable_Count(c))
	}
}

// Test_Expirable_Concurrent_Adds ports TestLRUConcurrency. Determinism forbids goroutines, not
// concurrency: the upstream's 1000 racing adds of overlapping keys become a deterministic
// interleaving of the same operations, which must still converge to one entry per distinct key.
func Test_Expirable_Concurrent_Adds(t *testing.T) {
	loop, _, clock := new_expirable_loop()
	c := lru.New_Expirable[string, string](lru.Expirable_Input[string, string]{
		Capacity: 100,
		TTL:      time.HOUR,
		Clock:    clock,
		IO:       &loop,
	})
	for i_index := 0; i_index < 1000; i_index++ {
		key := fmt.Sprintf("key-%d", i_index/10)
		value := fmt.Sprintf("val-%d", i_index/10)
		lru.Expirable_Add(c, key, value)
	}
	if lru.Expirable_Count(c) != 100 {
		t.Fatalf("count = %d, want 100", lru.Expirable_Count(c))
	}
}

// Test_Simple_Operations_Allocation_Free checks every Simple operation is zero-allocation after
// construction: with the key/value buffers reused, a churn of every read and write reports zero.
func Test_Simple_Operations_Allocation_Free(t *testing.T) {
	c := lru.New_Simple[int, int](64, nil)
	for i_index := 0; i_index < 64; i_index++ {
		lru.Simple_Add(c, i_index, i_index)
	}
	keys := make([]int, lru.Simple_Cap(c))
	values := make([]int, lru.Simple_Cap(c))
	generator := prng.New(1)
	allocations := testing.AllocsPerRun(4000, func() {
		key := prng.Generator_Below(&generator, 256)
		lru.Simple_Add(c, key, key)
		lru.Simple_Get(c, key)
		lru.Simple_Peek(c, key)
		lru.Simple_Contains(c, key)
		lru.Simple_Contains_Or_Add(c, key, key)
		lru.Simple_Peek_Or_Add(c, key, key)
		lru.Simple_Get_Oldest(c)
		lru.Simple_Keys(c, keys)
		lru.Simple_Values(c, values)
		lru.Simple_Remove(c, key)
		lru.Simple_Remove_Oldest(c)
	})
	if allocations != 0 {
		t.Fatalf("Simple operations allocated %v, want 0", allocations)
	}
}

// Test_Two_Queue_Operations_Allocation_Free checks every Two_Queue operation is zero-allocation
// after construction, including the recent-to-frequent promotion and ghost-list churn.
func Test_Two_Queue_Operations_Allocation_Free(t *testing.T) {
	c := lru.New_Two_Queue[int, int](lru.Two_Queue_Input{Capacity: 64})
	for i_index := 0; i_index < 64; i_index++ {
		lru.Two_Queue_Add(c, i_index, i_index)
	}
	keys := make([]int, lru.Two_Queue_Cap(c))
	values := make([]int, lru.Two_Queue_Cap(c))
	generator := prng.New(1)
	allocations := testing.AllocsPerRun(4000, func() {
		key := prng.Generator_Below(&generator, 256)
		lru.Two_Queue_Add(c, key, key)
		lru.Two_Queue_Get(c, key)
		lru.Two_Queue_Peek(c, key)
		lru.Two_Queue_Contains(c, key)
		lru.Two_Queue_Keys(c, keys)
		lru.Two_Queue_Values(c, values)
		lru.Two_Queue_Remove(c, key)
	})
	if allocations != 0 {
		t.Fatalf("Two_Queue operations allocated %v, want 0", allocations)
	}
}

// Test_Expirable_Operations_Allocation_Free checks every Expirable operation is zero-allocation
// after construction, including the intrusive expiry-bucket bookkeeping.
func Test_Expirable_Operations_Allocation_Free(t *testing.T) {
	loop, _, clock := new_expirable_loop()
	c := lru.New_Expirable[int, int](lru.Expirable_Input[int, int]{
		Capacity: 64,
		TTL:      time.HOUR,
		Clock:    clock,
		IO:       &loop,
	})
	for i_index := 0; i_index < 64; i_index++ {
		lru.Expirable_Add(c, i_index, i_index)
	}
	keys := make([]int, lru.Expirable_Cap(c))
	values := make([]int, lru.Expirable_Cap(c))
	generator := prng.New(1)
	allocations := testing.AllocsPerRun(4000, func() {
		key := prng.Generator_Below(&generator, 256)
		lru.Expirable_Add(c, key, key)
		lru.Expirable_Get(c, key)
		lru.Expirable_Peek(c, key)
		lru.Expirable_Contains(c, key)
		lru.Expirable_Get_Oldest(c)
		lru.Expirable_Keys(c, keys)
		lru.Expirable_Values(c, values)
		lru.Expirable_Remove(c, key)
		lru.Expirable_Remove_Oldest(c)
	})
	if allocations != 0 {
		t.Fatalf("Expirable operations allocated %v, want 0", allocations)
	}
}

// Benchmark_Simple_Random ports BenchmarkLRU_Rand with deterministic keys.
func Benchmark_Simple_Random(b *testing.B) {
	generator := prng.New(1)
	c := lru.New_Simple[int, int](8192, nil)
	b.ResetTimer()
	for op_index := 0; op_index < b.N; op_index++ {
		key := prng.Generator_Below(&generator, 32768)
		if op_index%2 == 0 {
			lru.Simple_Add(c, key, key)
		} else {
			lru.Simple_Get(c, key)
		}
	}
}

// Benchmark_Simple_Frequent ports BenchmarkLRU_Freq with deterministic keys.
func Benchmark_Simple_Frequent(b *testing.B) {
	generator := prng.New(1)
	c := lru.New_Simple[int, int](8192, nil)
	b.ResetTimer()
	for op_index := 0; op_index < b.N; op_index++ {
		if op_index%2 == 0 {
			lru.Simple_Add(c, prng.Generator_Below(&generator, 16384), op_index)
		} else {
			lru.Simple_Get(c, prng.Generator_Below(&generator, 32768))
		}
	}
}

// Benchmark_Two_Queue_Random ports Benchmark2Q_Rand with deterministic keys.
func Benchmark_Two_Queue_Random(b *testing.B) {
	generator := prng.New(1)
	c := lru.New_Two_Queue[int, int](lru.Two_Queue_Input{Capacity: 8192})
	b.ResetTimer()
	for op_index := 0; op_index < b.N; op_index++ {
		key := prng.Generator_Below(&generator, 32768)
		if op_index%2 == 0 {
			lru.Two_Queue_Add(c, key, key)
		} else {
			lru.Two_Queue_Get(c, key)
		}
	}
}

// Benchmark_Two_Queue_Frequent ports Benchmark2Q_Freq with deterministic keys.
func Benchmark_Two_Queue_Frequent(b *testing.B) {
	generator := prng.New(1)
	c := lru.New_Two_Queue[int, int](lru.Two_Queue_Input{Capacity: 8192})
	b.ResetTimer()
	for op_index := 0; op_index < b.N; op_index++ {
		if op_index%2 == 0 {
			lru.Two_Queue_Add(c, prng.Generator_Below(&generator, 16384), op_index)
		} else {
			lru.Two_Queue_Get(c, prng.Generator_Below(&generator, 32768))
		}
	}
}

// Benchmark_Expirable_Random ports BenchmarkLRU_Rand_WithExpire; the loop is not driven, so the
// run measures the bucket bookkeeping rather than reaping.
func Benchmark_Expirable_Random(b *testing.B) {
	loop, _, clock := new_expirable_loop()
	generator := prng.New(1)
	c := lru.New_Expirable[int, int](lru.Expirable_Input[int, int]{
		Capacity: 8192,
		TTL:      10 * time.MICROSECOND,
		Clock:    clock,
		IO:       &loop,
	})
	b.ResetTimer()
	for op_index := 0; op_index < b.N; op_index++ {
		key := prng.Generator_Below(&generator, 32768)
		if op_index%2 == 0 {
			lru.Expirable_Add(c, key, key)
		} else {
			lru.Expirable_Get(c, key)
		}
	}
}
