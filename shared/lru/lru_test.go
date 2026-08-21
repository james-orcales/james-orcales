// Regression tests ported from github.com/hashicorp/golang-lru, adapted to the deterministic
// house port: randomness comes from the seeded prng rather than crypto/rand, expiry rides a
// simulated io loop rather than wall time and a reaper goroutine, and the upstream Cache
// tests fold onto Simple because the thread-safe wrapper collapsed into it.
//
// TestLRUConcurrency is ported as Test_Expirable_Concurrent_Adds: determinism forbids goroutines,
// not concurrency, so its racing overlapping adds become a deterministic interleaving of the same
// operations that must converge to the same state. Three upstream tests have no analog and are
// not ported: TestLRUInterface (the LRUCache interface was removed), and the goroutine-lifecycle
// pair TestCache_CloseGoRoutine and TestCache_RestartGoRoutine (they toggle a background reaper
// this port does not have — expiry is lazy on read, and reaping rides a repeating io.Timeout the
// harness's loop fires). The ARC cache was not ported at all, so its tests are absent too.
package lru

import (
	"fmt"
	"reflect"
	"testing"
	"unsafe"

	"local/james-orcales/shared/simulation/nbio"
	"local/james-orcales/shared/simulation/prng"
	"local/james-orcales/shared/simulation/time"
)

// Builds a fresh simulated io loop for the Expirable ports: the submit surface to inject, the
// driver the test pumps, and the clock expiry is stamped against.
func new_expirable_loop() (loop nbio.Timeline, driver nbio.Driver, host time.Clock) {
	state := new(nbio.Sim)
	surface, driver := nbio.New_Simulated_IO(state, 0, time.NANOSECOND, nbio.Sim_Memory{
		Nodes:       make([]nbio.Sim_Node, EXPIRABLE_LOOP_SLOT_CAPACITY),
		Descriptors: make([]nbio.Sim_Descriptor, EXPIRABLE_LOOP_SLOT_CAPACITY),
		Operations:  make([]nbio.Sim_Operation, EXPIRABLE_LOOP_SLOT_CAPACITY),
		Queue:       make([]*nbio.Completion, EXPIRABLE_LOOP_QUEUE_CAPACITY),
		Events:      make([]nbio.Virtual_Event, EXPIRABLE_LOOP_SLOT_CAPACITY),
		Clocks:      make([]nbio.Sim_Clock, EXPIRABLE_LOOP_SLOT_CAPACITY),
	})
	return surface.Timeline, driver, nbio.Sim_Clock_To_Clock(&state.Clocks[0])
}

// Expirable arms one reaper timer per cache; the queue leaves room for the tests that arm
// many caches on one loop.
const EXPIRABLE_LOOP_QUEUE_CAPACITY = 64

// Every other simulator resource is one slot: expiry touches no endpoint and reads one clock.
const EXPIRABLE_LOOP_SLOT_CAPACITY = 1

// Allocates test ownership outside production initialization.
func new_simple_cache[K Key_Kind, V Value_Kind](
	capacity Capacity, on_evict Evict_Callback[K, V],
) (cache *Simple[K, V]) {
	cache = new(Simple[K, V])
	nodes := new(Nodes[K, V])
	New_Simple(cache, nodes, capacity, on_evict)
	return cache
}

// Allocates test ownership outside production initialization.
func new_two_queue_cache[K Key_Kind, V Value_Kind](input Two_Queue_Input) (cache *Two_Queue[K, V]) {
	cache = new(Two_Queue[K, V])
	nodes := new(Two_Queue_Nodes[K, V])
	New_Two_Queue(cache, nodes, input)
	return cache
}

// Allocates test ownership outside production initialization.
func new_expirable_cache[K Key_Kind, V Value_Kind](
	input Expirable_Input[K, V],
) (cache *Expirable[K, V]) {
	cache = new(Expirable[K, V])
	nodes := new(Nodes[K, V])
	New_Expirable(cache, nodes, input)
	return cache
}

// Collects a Simple's keys into a fresh slice through the caller-buffer API, for comparison.
func simple_keys_of[K Key_Kind, V Value_Kind](c *Simple[K, V]) (keys []K) {
	storage := new([COUNT_MAXIMUM]K)
	buffer := Keys[K]{Storage: storage, Count: Key_Count(Simple_Cap(c))}
	return storage[:Simple_Keys(c, buffer)]
}

// Collects a Simple's values into a fresh slice through the caller-buffer API.
func simple_values_of[K Key_Kind, V Value_Kind](c *Simple[K, V]) (values []V) {
	storage := new([COUNT_MAXIMUM]V)
	buffer := Values[V]{Storage: storage, Count: Value_Count(Simple_Cap(c))}
	return storage[:Simple_Values(c, buffer)]
}

// Collects a Two_Queue's keys into a fresh slice through the caller-buffer API.
func two_queue_keys_of[K Key_Kind, V Value_Kind](c *Two_Queue[K, V]) (keys []K) {
	storage := new([COUNT_MAXIMUM]K)
	buffer := Keys[K]{Storage: storage, Count: Key_Count(Two_Queue_Cap(c))}
	return storage[:Two_Queue_Keys(c, buffer)]
}

// Collects an Expirable's unexpired keys into a fresh slice through the caller-buffer API.
func expirable_keys_of[K Key_Kind, V Value_Kind](c *Expirable[K, V]) (keys []K) {
	storage := new([COUNT_MAXIMUM]K)
	buffer := Keys[K]{Storage: storage, Count: Key_Count(Expirable_Cap(c))}
	return storage[:Expirable_Keys(c, buffer)]
}

// Collects an Expirable's unexpired values into a fresh slice through the caller-buffer API.
func expirable_values_of[K Key_Kind, V Value_Kind](c *Expirable[K, V]) (values []V) {
	storage := new([COUNT_MAXIMUM]V)
	buffer := Values[V]{Storage: storage, Count: Value_Count(Expirable_Cap(c))}
	return storage[:Expirable_Values(c, buffer)]
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
	c := new_simple_cache[int, int](128, on_evict)
	for i_index := 0; i_index < 256; i_index++ {
		Simple_Add(c, i_index, i_index)
	}
	if Simple_Count(c) != 128 {
		t.Fatalf("bad count: %v", Simple_Count(c))
	}
	if Simple_Cap(c) != 128 {
		t.Fatalf("bad cap: %v", Simple_Cap(c))
	}
	if evict_count != 128 {
		t.Fatalf("bad evict count: %v", evict_count)
	}
	for i_index, key := range simple_keys_of(c) {
		value, ok := Simple_Get(c, key)
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
	c := new_simple_cache[int, int](128, nil)
	for i_index := 0; i_index < 256; i_index++ {
		Simple_Add(c, i_index, i_index)
	}
	for i_index := 0; i_index < 128; i_index++ {
		_, ok := Simple_Get(c, i_index)
		if ok {
			t.Fatalf("%d should be evicted", i_index)
		}
	}
	for i_index := 128; i_index < 192; i_index++ {
		if !Simple_Remove(c, i_index) {
			t.Fatalf("%d should be contained", i_index)
		}
		if Simple_Remove(c, i_index) {
			t.Fatalf("%d should not be contained", i_index)
		}
	}
	Simple_Get(c, 192) // 192 becomes most-recently-used, so it is last in Keys.
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
	Simple_Purge(c)
	if Simple_Count(c) != 0 {
		t.Fatalf("bad count: %v", Simple_Count(c))
	}
	_, ok := Simple_Get(c, 200)
	if ok {
		t.Fatalf("should contain nothing")
	}
}

// Test_Simple_Get_Oldest_Remove_Oldest ports TestLRU_GetOldest_RemoveOldest.
func Test_Simple_Get_Oldest_Remove_Oldest(t *testing.T) {
	c := new_simple_cache[int, int](128, nil)
	for i_index := 0; i_index < 256; i_index++ {
		Simple_Add(c, i_index, i_index)
	}
	key, _, ok := Simple_Get_Oldest(c)
	if !ok {
		t.Fatalf("missing")
	}
	if key != 128 {
		t.Fatalf("bad: %v", key)
	}
	key, _, ok = Simple_Remove_Oldest(c)
	if !ok {
		t.Fatalf("missing")
	}
	if key != 128 {
		t.Fatalf("bad: %v", key)
	}
	key, _, ok = Simple_Remove_Oldest(c)
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
	c := new_simple_cache[int, int](1, on_evict)
	if Simple_Add(c, 1, 1) {
		t.Errorf("should not have an eviction")
	}
	if evict_count != 0 {
		t.Errorf("should not have an eviction")
	}
	if !Simple_Add(c, 2, 2) {
		t.Errorf("should have an eviction")
	}
	if evict_count != 1 {
		t.Errorf("should have an eviction")
	}
}

// Test_Simple_Contains_Leaves_Recency ports TestLRU_Contains / TestLRUContains.
func Test_Simple_Contains_Leaves_Recency(t *testing.T) {
	c := new_simple_cache[int, int](2, nil)
	Simple_Add(c, 1, 1)
	Simple_Add(c, 2, 2)
	if !Simple_Contains(c, 1) {
		t.Errorf("1 should be contained")
	}
	Simple_Add(c, 3, 3)
	if Simple_Contains(c, 1) {
		t.Errorf("contains should not have renewed recency of 1")
	}
}

// Test_Simple_Peek_Leaves_Recency ports TestLRU_Peek / TestLRUPeek.
func Test_Simple_Peek_Leaves_Recency(t *testing.T) {
	c := new_simple_cache[int, int](2, nil)
	Simple_Add(c, 1, 1)
	Simple_Add(c, 2, 2)
	value, ok := Simple_Peek(c, 1)
	if !ok {
		t.Errorf("1 should be set")
	}
	if value != 1 {
		t.Errorf("1 should be 1: %v", value)
	}
	Simple_Add(c, 3, 3)
	if Simple_Contains(c, 1) {
		t.Errorf("peek should not have renewed recency of 1")
	}
}

// Test_Simple_Contains_Or_Add_Leaves_Recency ports TestLRUContainsOrAdd.
func Test_Simple_Contains_Or_Add_Leaves_Recency(t *testing.T) {
	c := new_simple_cache[int, int](2, nil)
	Simple_Add(c, 1, 1)
	Simple_Add(c, 2, 2)
	present, evicted := Simple_Contains_Or_Add(c, 1, 1)
	if !present {
		t.Errorf("1 should be contained")
	}
	if evicted {
		t.Errorf("nothing should be evicted")
	}
	Simple_Add(c, 3, 3)
	present, evicted = Simple_Contains_Or_Add(c, 1, 1)
	if present {
		t.Errorf("1 should not be contained")
	}
	if !evicted {
		t.Errorf("an eviction should have occurred")
	}
	if !Simple_Contains(c, 1) {
		t.Errorf("now 1 should be contained")
	}
}

// Test_Simple_Peek_Or_Add_Leaves_Recency ports TestLRUPeekOrAdd.
func Test_Simple_Peek_Or_Add_Leaves_Recency(t *testing.T) {
	c := new_simple_cache[int, int](2, nil)
	Simple_Add(c, 1, 1)
	Simple_Add(c, 2, 2)
	previous, present, evicted := Simple_Peek_Or_Add(c, 1, 1)
	if !present {
		t.Errorf("1 should be contained")
	}
	if evicted {
		t.Errorf("nothing should be evicted")
	}
	if previous != 1 {
		t.Errorf("previous should be 1")
	}
	Simple_Add(c, 3, 3)
	present, evicted = Simple_Contains_Or_Add(c, 1, 1)
	if present {
		t.Errorf("1 should not be contained")
	}
	if !evicted {
		t.Errorf("an eviction should have occurred")
	}
	if !Simple_Contains(c, 1) {
		t.Errorf("now 1 should be contained")
	}
}

// Test_Simple_Eviction_Same_Key_Add ports the Add case of TestCache_EvictionSameKey: re-adding a
// present key renews recency without a spurious eviction.
func Test_Simple_Eviction_Same_Key_Add(t *testing.T) {
	var evicted_keys []int
	c := new_simple_cache[int, struct{}](2, func(key int, value struct{}) {
		evicted_keys = append(evicted_keys, key)
	})
	if Simple_Add(c, 1, struct{}{}) {
		t.Error("first 1: unexpected eviction")
	}
	if !reflect.DeepEqual(simple_keys_of(c), []int{1}) {
		t.Errorf("keys: %v", simple_keys_of(c))
	}
	if Simple_Add(c, 2, struct{}{}) {
		t.Error("2: unexpected eviction")
	}
	if !reflect.DeepEqual(simple_keys_of(c), []int{1, 2}) {
		t.Errorf("keys: %v", simple_keys_of(c))
	}
	if Simple_Add(c, 1, struct{}{}) {
		t.Error("second 1: unexpected eviction")
	}
	if !reflect.DeepEqual(simple_keys_of(c), []int{2, 1}) {
		t.Errorf("keys: %v", simple_keys_of(c))
	}
	if !Simple_Add(c, 3, struct{}{}) {
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
	c := new_simple_cache[int, struct{}](2, func(key int, value struct{}) {
		evicted_keys = append(evicted_keys, key)
	})
	present, evicted := Simple_Contains_Or_Add(c, 1, struct{}{})
	if present {
		t.Error("first 1: unexpected contained")
	}
	if evicted {
		t.Error("first 1: unexpected eviction")
	}
	present, evicted = Simple_Contains_Or_Add(c, 2, struct{}{})
	if present {
		t.Error("2: unexpected contained")
	}
	if evicted {
		t.Error("2: unexpected eviction")
	}
	present, evicted = Simple_Contains_Or_Add(c, 1, struct{}{})
	if !present {
		t.Error("second 1: expected contained")
	}
	if evicted {
		t.Error("second 1: unexpected eviction")
	}
	if !reflect.DeepEqual(simple_keys_of(c), []int{1, 2}) {
		t.Errorf("keys: %v", simple_keys_of(c))
	}
	present, evicted = Simple_Contains_Or_Add(c, 3, struct{}{})
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
	c := new_simple_cache[int, struct{}](2, func(key int, value struct{}) {
		evicted_keys = append(evicted_keys, key)
	})
	_, present, evicted := Simple_Peek_Or_Add(c, 1, struct{}{})
	if present {
		t.Error("first 1: unexpected contained")
	}
	if evicted {
		t.Error("first 1: unexpected eviction")
	}
	_, present, evicted = Simple_Peek_Or_Add(c, 2, struct{}{})
	if present {
		t.Error("2: unexpected contained")
	}
	if evicted {
		t.Error("2: unexpected eviction")
	}
	_, present, evicted = Simple_Peek_Or_Add(c, 1, struct{}{})
	if !present {
		t.Error("second 1: expected contained")
	}
	if evicted {
		t.Error("second 1: unexpected eviction")
	}
	if !reflect.DeepEqual(simple_keys_of(c), []int{1, 2}) {
		t.Errorf("keys: %v", simple_keys_of(c))
	}
	_, present, evicted = Simple_Peek_Or_Add(c, 3, struct{}{})
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
	c := new_two_queue_cache[int, int](Two_Queue_Input{Capacity: Capacity(size)})
	for op_index := 0; op_index < 200000; op_index++ {
		key := int(prng.Xoshiro_Below(&generator, 512))
		switch prng.Xoshiro_Below(&generator, 3) {
		case 0:
			Two_Queue_Add(c, key, key)
		case 1:
			Two_Queue_Get(c, key)
		case 2:
			Two_Queue_Remove(c, key)
		}
		live := Two_Queue_Count(c)
		if live > Count(size) {
			t.Fatalf("live %d exceeds size %d", live, size)
		}
	}
}

// Test_Two_Queue_Get_Recent_To_Frequent ports Test2Q_Get_RecentToFrequent.
func Test_Two_Queue_Get_Recent_To_Frequent(t *testing.T) {
	c := new_two_queue_cache[int, int](Two_Queue_Input{Capacity: 128})
	for i_index := 0; i_index < 128; i_index++ {
		Two_Queue_Add(c, i_index, i_index)
	}
	if Simple_Count(&c.Live[0]) != 128 {
		t.Fatalf("bad recent: %d", Simple_Count(&c.Live[0]))
	}
	if Simple_Count(&c.Live[1]) != 0 {
		t.Fatalf("bad frequent: %d", Simple_Count(&c.Live[1]))
	}
	for i_index := 0; i_index < 128; i_index++ {
		_, ok := Two_Queue_Get(c, i_index)
		if !ok {
			t.Fatalf("missing: %d", i_index)
		}
	}
	if Simple_Count(&c.Live[0]) != 0 {
		t.Fatalf("bad recent: %d", Simple_Count(&c.Live[0]))
	}
	if Simple_Count(&c.Live[1]) != 128 {
		t.Fatalf("bad frequent: %d", Simple_Count(&c.Live[1]))
	}
}

// Test_Two_Queue_Add_Recent_To_Frequent ports Test2Q_Add_RecentToFrequent.
func Test_Two_Queue_Add_Recent_To_Frequent(t *testing.T) {
	c := new_two_queue_cache[int, int](Two_Queue_Input{Capacity: 128})
	Two_Queue_Add(c, 1, 1)
	if Simple_Count(&c.Live[0]) != 1 {
		t.Fatalf("bad recent: %d", Simple_Count(&c.Live[0]))
	}
	if Simple_Count(&c.Live[1]) != 0 {
		t.Fatalf("bad frequent: %d", Simple_Count(&c.Live[1]))
	}
	Two_Queue_Add(c, 1, 1)
	if Simple_Count(&c.Live[0]) != 0 {
		t.Fatalf("bad recent: %d", Simple_Count(&c.Live[0]))
	}
	if Simple_Count(&c.Live[1]) != 1 {
		t.Fatalf("bad frequent: %d", Simple_Count(&c.Live[1]))
	}
	Two_Queue_Add(c, 1, 1)
	if Simple_Count(&c.Live[1]) != 1 {
		t.Fatalf("bad frequent: %d", Simple_Count(&c.Live[1]))
	}
}

// Test_Two_Queue_Add_Recent_Evict ports Test2Q_Add_RecentEvict.
func Test_Two_Queue_Add_Recent_Evict(t *testing.T) {
	c := new_two_queue_cache[int, int](Two_Queue_Input{Capacity: 4})
	Two_Queue_Add(c, 1, 1)
	Two_Queue_Add(c, 2, 2)
	Two_Queue_Add(c, 3, 3)
	Two_Queue_Add(c, 4, 4)
	Two_Queue_Add(c, 5, 5)
	if Simple_Count(&c.Live[0]) != 4 {
		t.Fatalf("bad recent: %d", Simple_Count(&c.Live[0]))
	}
	if Simple_Count(&c.Ghost[0]) != 1 {
		t.Fatalf("bad ghost: %d", Simple_Count(&c.Ghost[0]))
	}
	Two_Queue_Add(c, 1, 1)
	if Simple_Count(&c.Live[0]) != 3 {
		t.Fatalf("bad recent: %d", Simple_Count(&c.Live[0]))
	}
	if Simple_Count(&c.Live[1]) != 1 {
		t.Fatalf("bad frequent: %d", Simple_Count(&c.Live[1]))
	}
	Two_Queue_Add(c, 6, 6)
	if Simple_Count(&c.Live[0]) != 3 {
		t.Fatalf("bad recent: %d", Simple_Count(&c.Live[0]))
	}
	if Simple_Count(&c.Ghost[0]) != 2 {
		t.Fatalf("bad ghost: %d", Simple_Count(&c.Ghost[0]))
	}
}

// Test_Two_Queue_Full ports Test2Q: a full lifecycle of eviction, ordering, removal, and purge.
func Test_Two_Queue_Full(t *testing.T) {
	c := new_two_queue_cache[int, int](Two_Queue_Input{Capacity: 128})
	for i_index := 0; i_index < 256; i_index++ {
		Two_Queue_Add(c, i_index, i_index)
	}
	if Two_Queue_Count(c) != 128 {
		t.Fatalf("bad count: %v", Two_Queue_Count(c))
	}
	if Two_Queue_Cap(c) != 128 {
		t.Fatalf("bad cap: %v", Two_Queue_Cap(c))
	}
	for i_index, key := range two_queue_keys_of(c) {
		value, ok := Two_Queue_Get(c, key)
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
		_, ok := Two_Queue_Get(c, i_index)
		if ok {
			t.Fatalf("%d should be evicted", i_index)
		}
	}
	Two_Queue_Purge(c)
	if Two_Queue_Count(c) != 0 {
		t.Fatalf("bad count: %v", Two_Queue_Count(c))
	}
}

// Test_Two_Queue_Contains ports Test2Q_Contains.
func Test_Two_Queue_Contains(t *testing.T) {
	c := new_two_queue_cache[int, int](Two_Queue_Input{Capacity: 2})
	Two_Queue_Add(c, 1, 1)
	Two_Queue_Add(c, 2, 2)
	if !Two_Queue_Contains(c, 1) {
		t.Errorf("1 should be contained")
	}
	Two_Queue_Add(c, 3, 3)
	if Two_Queue_Contains(c, 1) {
		t.Errorf("contains should not have renewed recency of 1")
	}
}

// Test_Two_Queue_Peek ports Test2Q_Peek.
func Test_Two_Queue_Peek(t *testing.T) {
	c := new_two_queue_cache[int, int](Two_Queue_Input{Capacity: 2})
	Two_Queue_Add(c, 1, 1)
	Two_Queue_Add(c, 2, 2)
	value, ok := Two_Queue_Peek(c, 1)
	if !ok {
		t.Errorf("1 should be set")
	}
	if value != 1 {
		t.Errorf("1 should be 1: %v", value)
	}
	Two_Queue_Add(c, 3, 3)
	if Two_Queue_Contains(c, 1) {
		t.Errorf("peek should not have renewed recency of 1")
	}
}

// Test_Expirable_No_Purge ports TestLRUNoPurge: a long-TTL cache serves an entry without expiring
// it and reports membership and keys correctly. The upstream ttl=0 and Resize modes were removed.
func Test_Expirable_No_Purge(t *testing.T) {
	loop, _, host := new_expirable_loop()
	c := new_expirable_cache[string, string](Expirable_Input[string, string]{
		Capacity: 10,
		TTL:      TTL(time.HOUR),
		Clock:    host,
		Timeline: loop,
	})
	Expirable_Add(c, "key1", "val1")
	if Expirable_Count(c) != 1 {
		t.Fatalf("count differs from expected")
	}
	value, ok := Expirable_Peek(c, "key1")
	if value != "val1" {
		t.Fatalf("value differs from expected")
	}
	if !ok {
		t.Fatalf("should be true")
	}
	if !Expirable_Contains(c, "key1") {
		t.Fatalf("should contain key1")
	}
	if Expirable_Contains(c, "key2") {
		t.Fatalf("should not contain key2")
	}
	if !reflect.DeepEqual(expirable_keys_of(c), []string{"key1"}) {
		t.Fatalf("keys differ from expected")
	}
}

// Test_Expirable_Edge_Cases ports TestLRUEdgeCases: a nil value stores and reads back, and an
// overwrite replaces it.
func Test_Expirable_Edge_Cases(t *testing.T) {
	loop, _, host := new_expirable_loop()
	c := new_expirable_cache[string, *string](Expirable_Input[string, *string]{
		Capacity: 2,
		TTL:      TTL(time.HOUR),
		Clock:    host,
		Timeline: loop,
	})
	Expirable_Add(c, "key1", nil)
	value, exists := Expirable_Get(c, "key1")
	if value != nil {
		t.Fatalf("unexpected value for key1: %v", value)
	}
	if !exists {
		t.Fatalf("key1 should exist")
	}
	new_value := "val1"
	Expirable_Add(c, "key1", &new_value)
	value, exists = Expirable_Get(c, "key1")
	if value != &new_value {
		t.Fatalf("unexpected value for key1: %v", value)
	}
	if !exists {
		t.Fatalf("key1 should exist")
	}
}

// Test_Expirable_Values ports TestLRU_Values.
func Test_Expirable_Values(t *testing.T) {
	loop, _, host := new_expirable_loop()
	c := new_expirable_cache[string, string](Expirable_Input[string, string]{
		Capacity: 3,
		TTL:      TTL(time.HOUR),
		Clock:    host,
		Timeline: loop,
	})
	Expirable_Add(c, "key1", "val1")
	Expirable_Add(c, "key2", "val2")
	Expirable_Add(c, "key3", "val3")
	values := expirable_values_of(c)
	if !reflect.DeepEqual(values, []string{"val1", "val2", "val3"}) {
		t.Fatalf("values differ from expected: %v", values)
	}
}

// Test_Expirable_With_Purge_Expiry ports the expiry half of TestLRUWithPurge, driving the io loop
// so the repeating timer reaps the expired entry and fires the callback.
func Test_Expirable_With_Purge_Expiry(t *testing.T) {
	loop, driver, host := new_expirable_loop()
	var evicted []string
	c := new_expirable_cache[string, string](Expirable_Input[string, string]{
		Capacity: 10,
		TTL:      TTL(time.MICROSECOND),
		Clock:    host,
		Timeline: loop,
		On_Evict: func(key string, value string) { evicted = append(evicted, key, value) },
	})
	Expirable_Add(c, "key1", "val1")
	// A short drive stays under the TTL, so the entry is still live.
	nbio.Driver_Run_For(driver, 200*time.NANOSECOND)
	value, ok := Expirable_Get(c, "key1")
	if value != "val1" {
		t.Fatalf("value differs from expected")
	}
	if !ok {
		t.Fatalf("should be true")
	}
	// Drive until the timer reaps the expired entry.
	empty := func() (finished bool) {
		return Expirable_Count(c) == 0
	}
	completed, drive_err := nbio.Driver_Run_Until(driver, time.MILLISECOND, empty)
	if drive_err != nil {
		t.Fatalf("drive until the timer reaps the expired entry: %v", drive_err)
	}
	if !completed {
		t.Fatalf("timer did not reap the expired entry")
	}
	if !reflect.DeepEqual(evicted, []string{"key1", "val1"}) {
		t.Fatalf("evicted differs from expected: %v", evicted)
	}
}

// Test_Expirable_Purge_Fires_Callback ports the Purge tail of TestLRUWithPurge: an undriven timer
// leaves live entries in place, and Purge clears the cache and fires the callback for each.
func Test_Expirable_Purge_Fires_Callback(t *testing.T) {
	loop, _, host := new_expirable_loop()
	evicted := make(map[string]string)
	eviction_count := 0
	c := new_expirable_cache[string, string](Expirable_Input[string, string]{
		Capacity: 10,
		TTL:      TTL(time.HOUR),
		Clock:    host,
		Timeline: loop,
		On_Evict: func(key string, value string) {
			eviction_count++
			evicted[key] = value
		},
	})
	Expirable_Add(c, "key1", "val1")
	Expirable_Add(c, "key2", "val2")
	if Expirable_Count(c) != 2 {
		t.Fatalf("count differs from expected")
	}
	Expirable_Purge(c)
	if Expirable_Count(c) != 0 {
		t.Fatalf("count differs from expected")
	}
	if eviction_count != 2 {
		t.Fatalf("eviction count differs from expected: %d", eviction_count)
	}
	if !reflect.DeepEqual(evicted, map[string]string{"key1": "val1", "key2": "val2"}) {
		t.Fatalf("evicted differs from expected: %v", evicted)
	}
}

// Test_Expirable_Purge_Enforced_By_Size ports TestLRUWithPurgeEnforcedBySize.
func Test_Expirable_Purge_Enforced_By_Size(t *testing.T) {
	loop, _, host := new_expirable_loop()
	c := new_expirable_cache[string, string](Expirable_Input[string, string]{
		Capacity: 10,
		TTL:      TTL(time.HOUR),
		Clock:    host,
		Timeline: loop,
	})
	for i_index := 0; i_index < 100; i_index++ {
		key := fmt.Sprintf("key%d", i_index)
		value := fmt.Sprintf("val%d", i_index)
		Expirable_Add(c, key, value)
		got, ok := Expirable_Get(c, key)
		if got != value {
			t.Fatalf("value differs from expected")
		}
		if !ok {
			t.Fatalf("should be true")
		}
		if Expirable_Count(c) > 20 {
			t.Fatalf("count should be less than 20")
		}
	}
	if Expirable_Count(c) != 10 {
		t.Fatalf("count differs from expected")
	}
}

// Test_Expirable_Invalidate_And_Evict ports TestLRUInvalidateAndEvict: a remove fires the eviction
// callback. The upstream size=-1 unlimited mode was removed, so a bounded size stands in.
func Test_Expirable_Invalidate_And_Evict(t *testing.T) {
	loop, _, host := new_expirable_loop()
	evicted := 0
	c := new_expirable_cache[string, string](Expirable_Input[string, string]{
		Capacity: 10,
		TTL:      TTL(time.HOUR),
		Clock:    host,
		Timeline: loop,
		On_Evict: func(key string, value string) { evicted++ },
	})
	Expirable_Add(c, "key1", "val1")
	Expirable_Add(c, "key2", "val2")
	value, ok := Expirable_Get(c, "key1")
	if !ok {
		t.Fatalf("should be true")
	}
	if value != "val1" {
		t.Fatalf("value differs from expected")
	}
	if evicted != 0 {
		t.Fatalf("nothing should be evicted yet")
	}
	Expirable_Remove(c, "key1")
	if evicted != 1 {
		t.Fatalf("remove should have evicted one")
	}
	_, ok = Expirable_Get(c, "key1")
	if ok {
		t.Fatalf("removed key1 should read as a miss")
	}
}

// Test_Expirable_Loading_Expired ports TestLoadingExpired: reads reject an entry once its TTL has
// passed on the virtual clock.
func Test_Expirable_Loading_Expired(t *testing.T) {
	loop, driver, host := new_expirable_loop()
	c := new_expirable_cache[string, string](Expirable_Input[string, string]{
		Capacity: 8,
		TTL:      TTL(time.MICROSECOND),
		Clock:    host,
		Timeline: loop,
	})
	Expirable_Add(c, "key1", "val1")
	value, ok := Expirable_Peek(c, "key1")
	if value != "val1" {
		t.Fatalf("value differs from expected")
	}
	if !ok {
		t.Fatalf("should be true")
	}
	value, ok = Expirable_Get(c, "key1")
	if value != "val1" {
		t.Fatalf("value differs from expected")
	}
	if !ok {
		t.Fatalf("should be true")
	}
	nbio.Driver_Run_For(driver, 4*time.MICROSECOND) // Past TTL.
	_, ok = Expirable_Peek(c, "key1")
	if ok {
		t.Fatalf("expired key1 should peek as a miss")
	}
	_, ok = Expirable_Get(c, "key1")
	if ok {
		t.Fatalf("expired key1 should get as a miss")
	}
}

// Test_Expirable_Remove_Oldest ports TestLRURemoveOldest.
func Test_Expirable_Remove_Oldest(t *testing.T) {
	loop, _, host := new_expirable_loop()
	c := new_expirable_cache[string, string](Expirable_Input[string, string]{
		Capacity: 2,
		TTL:      TTL(time.HOUR),
		Clock:    host,
		Timeline: loop,
	})
	if Expirable_Cap(c) != 2 {
		t.Fatalf("expected cap 2")
	}
	key, _, ok := Expirable_Remove_Oldest(c)
	if key != "" {
		t.Fatalf("should be empty")
	}
	if ok {
		t.Fatalf("should be false")
	}
	if Expirable_Remove(c, "non_existent") {
		t.Fatalf("should be false")
	}
	Expirable_Add(c, "key1", "val1")
	Expirable_Add(c, "key2", "val2")
	if !reflect.DeepEqual(expirable_keys_of(c), []string{"key1", "key2"}) {
		t.Fatalf("keys differ from expected")
	}
	key, value, ok := Expirable_Remove_Oldest(c)
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
	loop, _, host := new_expirable_loop()
	var evicted_keys []int
	record := func(key int, value struct{}) { evicted_keys = append(evicted_keys, key) }
	c := new_expirable_cache[int, struct{}](Expirable_Input[int, struct{}]{
		Capacity: 2,
		TTL:      TTL(time.HOUR),
		Clock:    host,
		Timeline: loop,
		On_Evict: record,
	})
	if Expirable_Add(c, 1, struct{}{}) {
		t.Error("first 1: unexpected eviction")
	}
	if Expirable_Add(c, 2, struct{}{}) {
		t.Error("2: unexpected eviction")
	}
	if !reflect.DeepEqual(expirable_keys_of(c), []int{1, 2}) {
		t.Errorf("keys: %v", expirable_keys_of(c))
	}
	if Expirable_Add(c, 1, struct{}{}) {
		t.Error("second 1: unexpected eviction")
	}
	if !reflect.DeepEqual(expirable_keys_of(c), []int{2, 1}) {
		t.Errorf("keys: %v", expirable_keys_of(c))
	}
	if !Expirable_Add(c, 3, struct{}{}) {
		t.Error("3: expected eviction")
	}
	if !reflect.DeepEqual(evicted_keys, []int{2}) {
		t.Errorf("evicted keys: %v", evicted_keys)
	}
}

// Test_Expirable_Lifecycle ports ExampleLRU as a test: a hit before expiry, a miss after, and a
// count of one once the expired entry is reaped and a fresh key is added.
func Test_Expirable_Lifecycle(t *testing.T) {
	loop, driver, host := new_expirable_loop()
	c := new_expirable_cache[string, string](Expirable_Input[string, string]{
		Capacity: 5,
		TTL:      TTL(time.MICROSECOND),
		Clock:    host,
		Timeline: loop,
	})
	Expirable_Add(c, "key1", "val1")
	value, ok := Expirable_Get(c, "key1")
	if !ok {
		t.Fatalf("key1 should be found before expiration")
	}
	if value != "val1" {
		t.Fatalf("value differs from expected")
	}
	// Drive until the timer reaps the expired key1.
	gone := func() (finished bool) {
		return Expirable_Count(c) == 0
	}
	completed, drive_err := nbio.Driver_Run_Until(driver, time.MILLISECOND, gone)
	if drive_err != nil {
		t.Fatalf("drive until the timer reaps key1: %v", drive_err)
	}
	if !completed {
		t.Fatalf("timer did not reap key1")
	}
	Expirable_Add(c, "key2", "val2")
	if Expirable_Count(c) != 1 {
		t.Fatalf("count = %d, want 1", Expirable_Count(c))
	}
}

// Test_Expirable_Concurrent_Adds ports TestLRUConcurrency. Determinism forbids goroutines, not
// concurrency: the upstream's 1000 racing adds of overlapping keys become a deterministic
// interleaving of the same operations, which must still converge to one entry per distinct key.
func Test_Expirable_Concurrent_Adds(t *testing.T) {
	loop, _, host := new_expirable_loop()
	c := new_expirable_cache[string, string](Expirable_Input[string, string]{
		Capacity: 100,
		TTL:      TTL(time.HOUR),
		Clock:    host,
		Timeline: loop,
	})
	for i_index := 0; i_index < 1000; i_index++ {
		key := fmt.Sprintf("key-%d", i_index/10)
		value := fmt.Sprintf("val-%d", i_index/10)
		Expirable_Add(c, key, value)
	}
	if Expirable_Count(c) != 100 {
		t.Fatalf("count = %d, want 100", Expirable_Count(c))
	}
}

// Test_Simple_Operations_Allocation_Free checks every Simple operation is zero-allocation after
// construction: with the key/value buffers reused, a churn of every read and write reports zero.
func Test_Simple_Operations_Allocation_Free(t *testing.T) {
	c := new_simple_cache[int, int](64, nil)
	for i_index := 0; i_index < 64; i_index++ {
		Simple_Add(c, i_index, i_index)
	}
	keys := Keys[int]{Storage: new([COUNT_MAXIMUM]int), Count: Key_Count(Simple_Cap(c))}
	values := Values[int]{
		Storage: new([COUNT_MAXIMUM]int), Count: Value_Count(Simple_Cap(c)),
	}
	generator := prng.New(1)
	allocations := testing.AllocsPerRun(4000, func() {
		key := int(prng.Xoshiro_Below(&generator, 256))
		Simple_Add(c, key, key)
		Simple_Get(c, key)
		Simple_Peek(c, key)
		Simple_Contains(c, key)
		Simple_Contains_Or_Add(c, key, key)
		Simple_Peek_Or_Add(c, key, key)
		Simple_Get_Oldest(c)
		Simple_Keys(c, keys)
		Simple_Values(c, values)
		Simple_Remove(c, key)
		Simple_Remove_Oldest(c)
	})
	if allocations != 0 {
		t.Fatalf("Simple operations allocated %v, want 0", allocations)
	}
}

// Test_Construction_Allocation_Free proves caller-owned initialization allocates no heap storage.
func Test_Construction_Allocation_Free(t *testing.T) {
	var simple Simple[int, int]
	var simple_nodes Nodes[int, int]
	simple_allocations := testing.AllocsPerRun(100, func() {
		New_Simple(&simple, &simple_nodes, 64, nil)
	})
	if simple_allocations != 0 {
		t.Fatalf("Simple construction allocated %v, want 0", simple_allocations)
	}

	var two_queue Two_Queue[int, int]
	var two_queue_nodes Two_Queue_Nodes[int, int]
	two_queue_allocations := testing.AllocsPerRun(100, func() {
		New_Two_Queue(
			&two_queue,
			&two_queue_nodes,
			Two_Queue_Input{Capacity: 64},
		)
	})
	if two_queue_allocations != 0 {
		t.Fatalf("Two_Queue construction allocated %v, want 0", two_queue_allocations)
	}

	var expirable Expirable[int, int]
	var expirable_nodes Nodes[int, int]
	expirable_input := Expirable_Input[int, int]{
		Capacity: 64,
		TTL:      TTL(time.HOUR),
		Clock:    invariant_clock(),
		Timeline: invariant_timeline(),
	}
	expirable_allocations := testing.AllocsPerRun(100, func() {
		New_Expirable(&expirable, &expirable_nodes, expirable_input)
	})
	if expirable_allocations != 0 {
		t.Fatalf("Expirable construction allocated %v, want 0", expirable_allocations)
	}
}

// Test_Two_Queue_Operations_Allocation_Free checks every Two_Queue operation is zero-allocation
// after construction, including the recent-to-frequent promotion and ghost-list churn.
func Test_Two_Queue_Operations_Allocation_Free(t *testing.T) {
	c := new_two_queue_cache[int, int](Two_Queue_Input{Capacity: 64})
	for i_index := 0; i_index < 64; i_index++ {
		Two_Queue_Add(c, i_index, i_index)
	}
	keys := Keys[int]{Storage: new([COUNT_MAXIMUM]int), Count: Key_Count(Two_Queue_Cap(c))}
	values := Values[int]{
		Storage: new([COUNT_MAXIMUM]int), Count: Value_Count(Two_Queue_Cap(c)),
	}
	generator := prng.New(1)
	allocations := testing.AllocsPerRun(4000, func() {
		key := int(prng.Xoshiro_Below(&generator, 256))
		Two_Queue_Add(c, key, key)
		Two_Queue_Get(c, key)
		Two_Queue_Peek(c, key)
		Two_Queue_Contains(c, key)
		Two_Queue_Keys(c, keys)
		Two_Queue_Values(c, values)
		Two_Queue_Remove(c, key)
	})
	if allocations != 0 {
		t.Fatalf("Two_Queue operations allocated %v, want 0", allocations)
	}
}

// Test_Expirable_Operations_Allocation_Free checks every Expirable operation is zero-allocation
// after construction, including the intrusive expiry-bucket bookkeeping.
func Test_Expirable_Operations_Allocation_Free(t *testing.T) {
	c := new_expirable_cache[int, int](Expirable_Input[int, int]{
		Capacity: 64,
		TTL:      TTL(time.HOUR),
		Clock:    invariant_clock(),
		Timeline: invariant_timeline(),
	})
	for i_index := 0; i_index < 64; i_index++ {
		Expirable_Add(c, i_index, i_index)
	}
	keys := Keys[int]{Storage: new([COUNT_MAXIMUM]int), Count: Key_Count(Expirable_Cap(c))}
	values := Values[int]{
		Storage: new([COUNT_MAXIMUM]int), Count: Value_Count(Expirable_Cap(c)),
	}
	generator := prng.New(1)
	allocations := testing.AllocsPerRun(4000, func() {
		key := int(prng.Xoshiro_Below(&generator, 256))
		Expirable_Add(c, key, key)
		Expirable_Get(c, key)
		Expirable_Peek(c, key)
		Expirable_Contains(c, key)
		Expirable_Get_Oldest(c)
		Expirable_Keys(c, keys)
		Expirable_Values(c, values)
		Expirable_Remove(c, key)
		Expirable_Remove_Oldest(c)
		expirable_cleanup_callback(&c.Cleanup.Completion)
		Expirable_Count(c)
	})
	if allocations != 0 {
		t.Fatalf("Expirable operations allocated %v, want 0", allocations)
	}
}

// Test_Invariant_Domains drives every operation through each declared scalar witness.
func Test_Invariant_Domains(t *testing.T) {
	t.Helper()
	for _, state := range probe_list_states() {
		drive_list_operations(state)
	}
	for _, state := range probe_simple_states() {
		drive_simple_operations(state)
	}
	for _, state := range probe_two_queue_states() {
		drive_two_queue_operations(state)
	}
	for _, state := range probe_expirable_states() {
		drive_expirable_operations(state)
	}
	drive_constructor_inputs()
	drive_ratio_inputs()
}

// List factories give fresh state to each mutating operation.
type list_factory func() (list *List[int, int])

// Probe list states name empty, small, and boundary list states.
func probe_list_states() (states []list_factory) {
	return []list_factory{
		func() (list *List[int, int]) { return new(List[int, int]) },
		list_state(1, 0),
		list_state(1, 1),
		list_state(2, 0),
		list_state(2, 1),
		list_state(2, 2),
		list_state(3, 1),
		list_state(COUNT_MAXIMUM, 0),
		list_state(COUNT_MAXIMUM, 1),
		list_state(COUNT_MAXIMUM, COUNT_MAXIMUM-3),
		list_state(COUNT_MAXIMUM, COUNT_MAXIMUM-2),
		list_state(COUNT_MAXIMUM, COUNT_MAXIMUM-1),
		list_state(COUNT_MAXIMUM, COUNT_MAXIMUM),
	}
}

// List state makes one structurally live list without lookup cost.
func list_state(capacity_count Capacity, entry_count Count) (factory list_factory) {
	return func() (list *List[int, int]) {
		list = new(List[int, int])
		nodes := new(Nodes[int, int])
		list_initialize(list, nodes, Count(capacity_count))
		for key_index := 0; key_index < int(entry_count); key_index++ {
			list_push_front(list, key_index, key_index)
		}
		return list
	}
}

// Drive list operations gives each list assertion root every state.
func drive_list_operations(state list_factory) {
	for _, capacity_count := range probe_capacities() {
		attempt(func() {
			list_initialize(state(), new(Nodes[int, int]), Count(capacity_count))
		})
	}
	attempt(func() { list_reset(state()) })
	attempt(func() { list_allocate(state()) })
	attempt(func() { list_back(state()) })
	attempt(func() { list_push_front(state(), 1, 1) })
	for _, moment := range probe_moments() {
		attempt(func() { list_push_front_expirable(state(), 1, 1, moment) })
	}
	for _, position := range probe_positions() {
		attempt(func() { list_free(state(), position) })
		attempt(func() { list_splice_front(state(), position) })
		attempt(func() { list_remove(state(), position) })
		attempt(func() { list_move_to_front(state(), position) })
		attempt(func() { entry_previous(state(), position) })
	}
	attempt(func() { list_find(state(), 0) })
	attempt(func() { list_find(state(), -1) })
	attempt(func() {
		list := list_state(COUNT_MAXIMUM, 2)()
		list_move_to_front(list, Position(POSITION_MAXIMUM))
		entry_previous(list, Position(POSITION_MAXIMUM-1))
	})
}

// Simple factories give fresh cache state to each mutating operation.
type simple_factory func() (cache *Simple[int, int])

// Probe simple states mirrors every list witness through Simple.
func probe_simple_states() (states []simple_factory) {
	states = append(states, func() (cache *Simple[int, int]) { return new(Simple[int, int]) })
	for _, state := range probe_list_states()[1:] {
		list_source := state
		states = append(states, func() (cache *Simple[int, int]) {
			list := list_source()
			return &Simple[int, int]{
				Capacity:   Capacity(list.Capacity),
				Evict_List: *list,
			}
		})
	}
	return states
}

// Drive simple operations gives each Simple root every cache state.
func drive_simple_operations(state simple_factory) {
	for _, key := range []int{-1, 0} {
		attempt(func() { Simple_Add(state(), key, key) })
		attempt(func() { Simple_Get(state(), key) })
		attempt(func() { Simple_Contains(state(), key) })
		attempt(func() { Simple_Peek(state(), key) })
		attempt(func() { Simple_Contains_Or_Add(state(), key, key) })
		attempt(func() { Simple_Peek_Or_Add(state(), key, key) })
		attempt(func() { Simple_Remove(state(), key) })
	}
	attempt(func() { Simple_Remove_Oldest(state()) })
	attempt(func() { Simple_Get_Oldest(state()) })
	attempt(func() { Simple_Count(state()) })
	attempt(func() { Simple_Cap(state()) })
	attempt(func() { Simple_Purge(state()) })
	attempt(func() { simple_remove_oldest(state()) })
	for _, position := range probe_positions() {
		attempt(func() { simple_remove_element(state(), position) })
	}
	for _, count := range probe_counts() {
		attempt(func() {
			Simple_Keys(state(), Keys[int]{
				Storage: new([COUNT_MAXIMUM]int),
				Count:   Key_Count(count),
			})
		})
		attempt(func() {
			Simple_Values(state(), Values[int]{
				Storage: new([COUNT_MAXIMUM]int),
				Count:   Value_Count(count),
			})
		})
	}
}

// Two queue factories give fresh aggregate state to each operation.
type two_queue_factory func() (cache *Two_Queue[int, int])

// Probe two queue states names each scalar witness without multiplying axes.
func probe_two_queue_states() (states []two_queue_factory) {
	for _, capacity_count := range probe_capacities() {
		capacity_probe := capacity_count
		states = append(states, two_queue_state(
			capacity_probe, 0, DEFAULT_RECENT_RATIO, DEFAULT_GHOST_RATIO,
		))
	}
	// Full ratios keep the one-entry ghost non-empty after fixed-point truncation.
	states = append(states, func() (cache *Two_Queue[int, int]) {
		return new_two_queue_cache[int, int](Two_Queue_Input{
			Capacity:     CAPACITY_INITIALIZED_MINIMUM,
			Recent_Ratio: Recent_Ratio_Input(RATIO_MAXIMUM),
			Ghost_Ratio:  Ghost_Ratio_Input(RATIO_MAXIMUM),
		})
	})
	for _, count := range probe_counts() {
		count_probe := count
		states = append(states, two_queue_state(
			2, count_probe, DEFAULT_RECENT_RATIO, DEFAULT_GHOST_RATIO,
		))
	}
	for _, ratio := range probe_ratios() {
		ratio_probe := ratio
		states = append(states, two_queue_state(
			2, 1, Recent_Ratio(ratio_probe), Ghost_Ratio(ratio_probe),
		))
	}
	for _, count := range probe_counts() {
		states = append(states, two_queue_entries(count))
	}
	return states
}

// Two queue entries fill recent storage directly so output counts reach every witness.
func two_queue_entries(entry_count Count) (factory two_queue_factory) {
	return func() (cache *Two_Queue[int, int]) {
		cache = new_two_queue_cache[int, int](Two_Queue_Input{Capacity: COUNT_MAXIMUM})
		cache.Recent_Size = COUNT_MAXIMUM
		for key_index := 0; key_index < int(entry_count); key_index++ {
			list_push_front(&cache.Live[0].Evict_List, key_index, key_index)
		}
		return cache
	}
}

// Two queue state keeps owned sub-caches initialized while probing aggregate fields.
func two_queue_state(
	capacity_count Capacity,
	recent_size Count,
	recent_ratio Recent_Ratio,
	ghost_ratio Ghost_Ratio,
) (factory two_queue_factory) {
	return func() (cache *Two_Queue[int, int]) {
		cache = new_two_queue_cache[int, int](Two_Queue_Input{Capacity: 2})
		cache.Capacity = capacity_count
		cache.Recent_Size = recent_size
		cache.Recent_Ratio = recent_ratio
		cache.Ghost_Ratio = ghost_ratio
		return cache
	}
}

// Drive two queue operations gives each Two_Queue root every state.
func drive_two_queue_operations(state two_queue_factory) {
	for _, key := range []int{-1, 0} {
		attempt(func() { Two_Queue_Add(state(), key, key) })
		attempt(func() { Two_Queue_Get(state(), key) })
		attempt(func() { Two_Queue_Contains(state(), key) })
		attempt(func() { Two_Queue_Peek(state(), key) })
		attempt(func() { Two_Queue_Remove(state(), key) })
	}
	attempt(func() { Two_Queue_Count(state()) })
	attempt(func() { Two_Queue_Cap(state()) })
	attempt(func() { Two_Queue_Purge(state()) })
	attempt(func() { two_queue_ensure_space(state(), false) })
	attempt(func() { two_queue_ensure_space(state(), true) })
	for _, count := range probe_counts() {
		attempt(func() {
			Two_Queue_Keys(state(), Keys[int]{
				Storage: new([COUNT_MAXIMUM]int),
				Count:   Key_Count(count),
			})
		})
		attempt(func() {
			Two_Queue_Values(state(), Values[int]{
				Storage: new([COUNT_MAXIMUM]int),
				Count:   Value_Count(count),
			})
		})
	}
}

// Expirable factories give fresh cache state to each operation.
type expirable_factory func() (cache *Expirable[int, int])

// Probe expirable states names cache, list, TTL, and bucket witnesses.
func probe_expirable_states() (states []expirable_factory) {
	for _, capacity_count := range probe_capacities() {
		capacity_probe := capacity_count
		states = append(states, expirable_state(capacity_probe, TTL_MINIMUM, nil, 0))
	}
	for _, ttl := range probe_ttls() {
		ttl_probe := ttl
		states = append(states, expirable_state(1, ttl_probe, nil, 0))
	}
	states = append(states, expirable_state(
		1,
		TTL(2*BUCKET_COUNT),
		nil,
		0,
	))
	for _, bucket := range probe_buckets() {
		bucket_probe := bucket
		states = append(states, expirable_state(
			1, TTL_INITIALIZED_MINIMUM, nil, bucket_probe,
		))
	}
	for _, state := range probe_list_states() {
		list_source := state
		states = append(states, expirable_state(1, TTL_INITIALIZED_MINIMUM, list_source, 0))
	}
	states = append(states, expirable_due_state())
	return states
}

// Expirable due state names fired cleanup status.
func expirable_due_state() (factory expirable_factory) {
	return func() (cache *Expirable[int, int]) {
		cache = expirable_state(1, TTL_INITIALIZED_MINIMUM, nil, 0)()
		cache.Cleanup.Due = true
		return cache
	}
}

// Expirable state keeps injected dependencies complete while probing mutable fields.
func expirable_state(
	capacity_count Capacity,
	ttl TTL,
	list_source list_factory,
	bucket Bucket_Index,
) (factory expirable_factory) {
	return func() (cache *Expirable[int, int]) {
		cache = new_expirable_cache[int, int](Expirable_Input[int, int]{
			Capacity: 1,
			TTL:      TTL_INITIALIZED_MINIMUM,
			Clock:    invariant_clock(),
			Timeline: invariant_timeline(),
		})
		cache.Capacity = capacity_count
		cache.TTL = ttl
		cache.Next_Cleanup_Bucket = bucket
		if list_source != nil {
			cache.Evict_List = *list_source()
			cache.Capacity = Capacity(cache.Evict_List.Capacity)
		}
		return cache
	}
}

// Drive expirable operations gives each Expirable root every state.
func drive_expirable_operations(state expirable_factory) {
	for _, key := range []int{-1, 0} {
		attempt(func() { Expirable_Add(state(), key, key) })
		attempt(func() { Expirable_Get(state(), key) })
		attempt(func() { Expirable_Contains(state(), key) })
		attempt(func() { Expirable_Peek(state(), key) })
		attempt(func() { Expirable_Remove(state(), key) })
	}
	attempt(func() { Expirable_Remove_Oldest(state()) })
	attempt(func() { Expirable_Get_Oldest(state()) })
	attempt(func() { Expirable_Count(state()) })
	attempt(func() { Expirable_Cap(state()) })
	attempt(func() { Expirable_Purge(state()) })
	attempt(func() { expirable_arm_cleanup(state()) })
	attempt(func() { expirable_reap_due(state()) })
	attempt(func() { expirable_cleanup_interval(state()) })
	attempt(func() { expirable_reap_ripe_bucket(state()) })
	attempt(func() { expirable_remove_oldest(state()) })
	for _, position := range probe_positions() {
		attempt(func() { expirable_add_to_bucket(state(), position) })
		attempt(func() { expirable_remove_from_bucket(state(), position) })
		attempt(func() { expirable_remove_element(state(), position) })
	}
	for _, count := range probe_counts() {
		attempt(func() {
			Expirable_Keys(state(), Keys[int]{
				Storage: new([COUNT_MAXIMUM]int),
				Count:   Key_Count(count),
			})
		})
		attempt(func() {
			Expirable_Values(state(), Values[int]{
				Storage: new([COUNT_MAXIMUM]int),
				Count:   Value_Count(count),
			})
		})
	}
}

// Drive constructor inputs covers pre-state and each constructor input domain.
func drive_constructor_inputs() {
	for _, state := range probe_simple_states() {
		for _, capacity_count := range probe_capacities() {
			attempt(func() {
				New_Simple(state(), new(Nodes[int, int]), capacity_count, nil)
			})
		}
	}
	for _, state := range probe_two_queue_states() {
		for _, capacity_count := range probe_capacities() {
			attempt(func() {
				New_Two_Queue(
					state(),
					new(Two_Queue_Nodes[int, int]),
					Two_Queue_Input{Capacity: capacity_count},
				)
			})
		}
		for _, ratio := range probe_ratios() {
			attempt(func() {
				New_Two_Queue(
					state(),
					new(Two_Queue_Nodes[int, int]),
					Two_Queue_Input{
						Capacity:     COUNT_MAXIMUM,
						Recent_Ratio: Recent_Ratio_Input(ratio),
						Ghost_Ratio:  Ghost_Ratio_Input(ratio),
					},
				)
			})
		}
	}
	for _, state := range probe_expirable_states() {
		for _, capacity_count := range probe_capacities() {
			attempt(func() {
				New_Expirable(
					state(),
					new(Nodes[int, int]),
					Expirable_Input[int, int]{
						Capacity: capacity_count,
						TTL:      TTL_INITIALIZED_MINIMUM,
						Clock:    invariant_clock(),
						Timeline: invariant_timeline(),
					},
				)
			})
		}
		for _, ttl := range probe_ttls() {
			attempt(func() {
				New_Expirable(
					state(),
					new(Nodes[int, int]),
					Expirable_Input[int, int]{
						Capacity: 1,
						TTL:      ttl,
						Clock:    invariant_clock(),
						Timeline: invariant_timeline(),
					},
				)
			})
		}
	}
}

// Drive ratio inputs covers both scale operands and their maximum output.
func drive_ratio_inputs() {
	for _, count := range probe_counts() {
		for _, ratio := range probe_ratios() {
			attempt(func() { ratio_of(count, ratio) })
		}
	}
}

// Probe capacities names each bounded capacity witness.
func probe_capacities() (capacities []Capacity) {
	return []Capacity{CAPACITY_MINIMUM, CAPACITY_INITIALIZED_MINIMUM, 2, CAPACITY_MAXIMUM}
}

// Probe counts names each bounded count witness.
func probe_counts() (counts []Count) {
	return []Count{COUNT_MINIMUM, 1, 2, COUNT_MAXIMUM}
}

// Probe positions names absence, first interior handles, and final handle.
func probe_positions() (positions []Position) {
	return []Position{POSITION_NONE, 0, 1, 2, POSITION_MAXIMUM}
}

// Probe ratios names each fixed-point share witness.
func probe_ratios() (ratios []Ratio) {
	return []Ratio{Ratio(RATIO_MINIMUM), 1, 2, Ratio(RATIO_MAXIMUM)}
}

// Probe TTLs names each cache-lifetime witness.
func probe_ttls() (ttls []TTL) {
	return []TTL{TTL_MINIMUM, TTL_INITIALIZED_MINIMUM, 2, TTL_MAXIMUM}
}

// Probe buckets names each expiry-ring position witness.
func probe_buckets() (buckets []Bucket_Index) {
	return []Bucket_Index{BUCKET_INDEX_MINIMUM, 1, 2, BUCKET_INDEX_MAXIMUM}
}

// Probe moments names each monotonic-clock witness.
func probe_moments() (moments []time.Monotonic_Moment) {
	return []time.Monotonic_Moment{
		time.MONOTONIC_MOMENT_MINIMUM,
		1,
		2,
		time.MONOTONIC_MOMENT_MAXIMUM,
	}
}

// Attempt absorbs one domain panic after assertions observe the rejected input.
func attempt(action func()) {
	defer func() { recover() }()
	action()
}

// Invariant clock supplies allocation-free injected time for domain probes.
func invariant_clock() (host time.Clock) {
	return time.Clock{
		Now_Monotonic: invariant_now_monotonic,
		Now_Realtime:  invariant_now_realtime,
	}
}

// Invariant timeline supplies allocation-free timer submission for domain probes.
func invariant_timeline() (timeline nbio.Timeline) {
	return nbio.Timeline{
		Submit:        invariant_timeout,
		Timeout:       invariant_timeout,
		Open_Event:    invariant_open_event,
		Event_Listen:  invariant_event_listen,
		Event_Trigger: invariant_event_trigger,
		Close_Event:   invariant_close_event,
	}
}

// Invariant monotonic clock returns boot moment.
func invariant_now_monotonic(_ unsafe.Pointer) (moment time.Monotonic_Moment) {
	return time.MONOTONIC_MOMENT_MINIMUM
}

// Invariant realtime clock returns epoch moment.
func invariant_now_realtime(_ unsafe.Pointer) (moment time.Moment) {
	return 0
}

// Invariant timeout accepts one timer without retaining it.
func invariant_timeout(
	_ unsafe.Pointer, _ *nbio.Completion, _ time.Duration, _ nbio.Callback,
) {
	return
}

// Invariant event opener returns one inert event.
func invariant_open_event(_ unsafe.Pointer) (event nbio.Event, err error) {
	return 0, nil
}

// Invariant event listener accepts one inert listener.
func invariant_event_listen(
	_ unsafe.Pointer, _ nbio.Event, _ *nbio.Completion, _ nbio.Callback,
) {
	return
}

// Invariant event trigger accepts one inert trigger.
func invariant_event_trigger(_ unsafe.Pointer, _ nbio.Event, _ *nbio.Completion) {
	return
}

// Invariant event closer accepts one inert event close.
func invariant_close_event(_ unsafe.Pointer, _ nbio.Event) {
	return
}

// Benchmark_Simple_Random ports BenchmarkLRU_Rand with deterministic keys.
func Benchmark_Simple_Random(b *testing.B) {
	generator := prng.New(1)
	c := new_simple_cache[int, int](COUNT_MAXIMUM, nil)
	b.ResetTimer()
	for op_index := 0; op_index < b.N; op_index++ {
		key := int(prng.Xoshiro_Below(&generator, 32768))
		if op_index%2 == 0 {
			Simple_Add(c, key, key)
		} else {
			Simple_Get(c, key)
		}
	}
}

// Benchmark_Simple_Frequent ports BenchmarkLRU_Freq with deterministic keys.
func Benchmark_Simple_Frequent(b *testing.B) {
	generator := prng.New(1)
	c := new_simple_cache[int, int](COUNT_MAXIMUM, nil)
	b.ResetTimer()
	for op_index := 0; op_index < b.N; op_index++ {
		if op_index%2 == 0 {
			Simple_Add(c, int(prng.Xoshiro_Below(&generator, 16384)), op_index)
		} else {
			Simple_Get(c, int(prng.Xoshiro_Below(&generator, 32768)))
		}
	}
}

// Benchmark_Two_Queue_Random ports Benchmark2Q_Rand with deterministic keys.
func Benchmark_Two_Queue_Random(b *testing.B) {
	generator := prng.New(1)
	c := new_two_queue_cache[int, int](Two_Queue_Input{Capacity: COUNT_MAXIMUM})
	b.ResetTimer()
	for op_index := 0; op_index < b.N; op_index++ {
		key := int(prng.Xoshiro_Below(&generator, 32768))
		if op_index%2 == 0 {
			Two_Queue_Add(c, key, key)
		} else {
			Two_Queue_Get(c, key)
		}
	}
}

// Benchmark_Two_Queue_Frequent ports Benchmark2Q_Freq with deterministic keys.
func Benchmark_Two_Queue_Frequent(b *testing.B) {
	generator := prng.New(1)
	c := new_two_queue_cache[int, int](Two_Queue_Input{Capacity: COUNT_MAXIMUM})
	b.ResetTimer()
	for op_index := 0; op_index < b.N; op_index++ {
		if op_index%2 == 0 {
			Two_Queue_Add(c, int(prng.Xoshiro_Below(&generator, 16384)), op_index)
		} else {
			Two_Queue_Get(c, int(prng.Xoshiro_Below(&generator, 32768)))
		}
	}
}

// Benchmark_Expirable_Random ports BenchmarkLRU_Rand_WithExpire; the loop is not driven, so the
// run measures the bucket bookkeeping rather than reaping.
func Benchmark_Expirable_Random(b *testing.B) {
	loop, _, host := new_expirable_loop()
	generator := prng.New(1)
	c := new_expirable_cache[int, int](Expirable_Input[int, int]{
		Capacity: COUNT_MAXIMUM,
		TTL:      TTL(10 * time.MICROSECOND),
		Clock:    host,
		Timeline: loop,
	})
	b.ResetTimer()
	for op_index := 0; op_index < b.N; op_index++ {
		key := int(prng.Xoshiro_Below(&generator, 32768))
		if op_index%2 == 0 {
			Expirable_Add(c, key, key)
		} else {
			Expirable_Get(c, key)
		}
	}
}
