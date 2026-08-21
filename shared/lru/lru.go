// Package lru is a dependency-injected port of github.com/hashicorp/golang-lru:
// three fixed-size caches — Simple (LRU), Two_Queue (scan-resistant 2Q), and Expirable
// (TTL) — over one shared doubly-linked list.
//
// The upstream leans on ambient state the house linter forbids and deterministic
// simulation cannot replay, all of which this port removes:
//   - Every cache is single-threaded. The upstream guarded each with a sync.Mutex, but a
//     deterministic package holds no lock; the caller serializes access through its loop,
//     so there is no shared-memory concurrency to guard. This collapses the upstream's
//     lock-free base (simplelru.LRU) and thread-safe wrapper (lru.Cache) into one Simple —
//     the wrapper existed only to add a lock and buffer evictions past it.
//   - The LRUCache interface (a user interface) is gone; Two_Queue holds concrete *Simple
//     sub-caches — the house prefers one implementation to a substitution seam.
//   - Expirable read time.Now() directly; here it reads an injected time.Clock's
//     Now_Monotonic, so expiry replays bit-for-bit under a virtual clock in tests.
//   - Expirable drove cleanup from a self-owned time.Ticker goroutine; a deterministic package
//     starts none, so it arms one injected Timeline timeout. Timeout records cleanup due. Next
//     cache access performs bounded generic cleanup and arms next timeout.
//   - Constructors returned an error on non-positive size; that is a programmer bug, so they
//     assert instead. Upstream size<=0 "unlimited" and ttl<=0 "no expiry" modes are gone.
//   - 2Q's recent/ghost split was float64, which a deterministic package bans; here it is a
//     fixedpoint.Ratio, the house stand-in for float64 (prng.Ratio is a probability, not this).
//   - Nothing is unbounded. Caller gives fixed Nodes storage. New initializes its Capacity prefix.
//     Construction, Add, Get, eviction, and cleanup allocate nothing. There is no Resize.
//     Keys and Values fill caller storage, so caller bounds reads too.
//
// The linter also bans methods that do not satisfy a stdlib interface, so every cache
// operation is a free function prefixed by its type — Simple_Get(cache, key), not
// cache.Get(key) — mirroring shared/uuid's UUID_Version(u).
package lru

import (
	"unsafe"

	"local/james-orcales/shared/math/fixedpoint"
	"local/james-orcales/shared/sim/aver/default"
	"local/james-orcales/shared/sim/nbio"
	"local/james-orcales/shared/sim/time"
	"local/james-orcales/shared/slices"
)

// Key is one comparable cache key.
type Key interface{}

// Value is one cache value.
type Value interface{}

// Count is cache entry quantity or capacity.
type Count int

// Count_Invariants bounds cache quantities to repository slice limit.
func Count_Invariants(count Count, namespace aver.Namespace) {
	aver.Tree(count, namespace).
		Range_Int(int(count), int(COUNT_MINIMUM), int(COUNT_MAXIMUM)).
		Ensure()
}

// COUNT_MINIMUM is empty cache quantity.
const COUNT_MINIMUM = 0

// COUNT_MAXIMUM matches the repository collection boundary.
const COUNT_MAXIMUM = slices.SLICE_COUNT_MAXIMUM

// POSITION_MINIMUM admits absent node handle.
const POSITION_MINIMUM = -1

// POSITION_MAXIMUM is final node in largest pool.
const POSITION_MAXIMUM = COUNT_MAXIMUM - 1

// Status reports one cache predicate.
type Status bool

// Status_Invariants requires both predicate results.
func Status_Invariants(status Status, namespace aver.Namespace) {
	aver.Tree(status, namespace).
		Sometimes(bool(status), "Cache predicate is true.").
		Ensure()
}

// Present_Status reports key presence before conditional insertion.
type Present_Status bool

// Present_Status_Invariants requires both presence results.
func Present_Status_Invariants(status Present_Status, namespace aver.Namespace) {
	aver.Tree(status, namespace).
		Sometimes(bool(status), "Key is present.").
		Ensure()
}

// Evicted_Status reports eviction during conditional insertion.
type Evicted_Status bool

// Evicted_Status_Invariants requires both eviction results.
func Evicted_Status_Invariants(status Evicted_Status, namespace aver.Namespace) {
	aver.Tree(status, namespace).
		Sometimes(bool(status), "Conditional insertion evicts one entry.").
		Ensure()
}

// Conditional_Result holds presence and eviction results from one conditional insertion.
type Conditional_Result struct {
	// Present reports whether key already existed.
	Present Present_Status
	// Evicted reports whether insertion discarded one entry.
	Evicted Evicted_Status
}

// Conditional_Result_Invariants composes conditional insertion results.
func Conditional_Result_Invariants(result Conditional_Result, namespace aver.Namespace) {
	Present_Status_Invariants(result.Present, namespace)
	Evicted_Status_Invariants(result.Evicted, namespace)
}

// Peek_Or_Add_Result holds existing value and conditional insertion results.
type Peek_Or_Add_Result struct {
	// Previous is existing value when Present is true.
	Previous Value
	// Present reports whether key already existed.
	Present Present_Status
	// Evicted reports whether insertion discarded one entry.
	Evicted Evicted_Status
}

// Peek_Or_Add_Result_Invariants composes conditional insertion results.
func Peek_Or_Add_Result_Invariants(result Peek_Or_Add_Result, namespace aver.Namespace) {
	Present_Status_Invariants(result.Present, namespace)
	Evicted_Status_Invariants(result.Evicted, namespace)
}

// Position identifies one node or POSITION_NONE.
type Position int

// Position_Invariants bounds node handles to backing pool positions.
func Position_Invariants(position Position, namespace aver.Namespace) {
	aver.Tree(position, namespace).
		Range_Int(int(position), POSITION_MINIMUM, POSITION_MAXIMUM).
		Ensure()
}

// POSITION_NONE identifies no node.
const POSITION_NONE Position = -1

// BUCKET_INDEX_MINIMUM is first expiry bucket.
const BUCKET_INDEX_MINIMUM = 0

// BUCKET_INDEX_MAXIMUM is final expiry bucket.
const BUCKET_INDEX_MAXIMUM = BUCKET_COUNT - 1

// Bucket_Index identifies one expiry-ring bucket.
type Bucket_Index uint8

// Bucket_Index_Invariants bounds bucket handles to expiry ring.
func Bucket_Index_Invariants(index Bucket_Index, namespace aver.Namespace) {
	aver.Tree(index, namespace).
		Range_Uint8(uint8(index), BUCKET_INDEX_MINIMUM, BUCKET_INDEX_MAXIMUM).
		Ensure()
}

// Key_Count is maximum key quantity one call writes.
type Key_Count int

// Key_Count_Invariants bounds caller key output.
func Key_Count_Invariants(count Key_Count, namespace aver.Namespace) {
	aver.Tree(count, namespace).
		Range_Int(int(count), COUNT_MINIMUM, COUNT_MAXIMUM).
		Ensure()
}

// Key_Storage is caller-owned key output. An assertion tree takes no generic subject, thus one
// Always states the one length every read admits.
type Key_Storage []Key

// Key_Storage_Invariants fixes key output to the largest cache.
func Key_Storage_Invariants(value Key_Storage, _ aver.Namespace) {
	aver.Always(
		len(value) == COUNT_MAXIMUM,
		"Key output holds every entry of the largest cache.",
	)
}

// Keys is caller-owned key storage plus requested write bound.
type Keys struct {
	// Storage is caller-owned output sized to largest cache, thus read never grows heap.
	Storage Key_Storage
	// Count bounds this call below full storage.
	Count Key_Count
}

// Keys_Invariants composes storage and bounds requested writes.
func Keys_Invariants(keys Keys, namespace aver.Namespace) {
	Key_Storage_Invariants(keys.Storage, namespace)
	Key_Count_Invariants(keys.Count, namespace)
}

// Value_Count is maximum value quantity one call writes.
type Value_Count int

// Value_Count_Invariants bounds caller value output.
func Value_Count_Invariants(count Value_Count, namespace aver.Namespace) {
	aver.Tree(count, namespace).
		Range_Int(int(count), COUNT_MINIMUM, COUNT_MAXIMUM).
		Ensure()
}

// Value_Storage is caller-owned value output.
type Value_Storage []Value

// Value_Storage_Invariants fixes value output to largest cache.
func Value_Storage_Invariants(value Value_Storage, _ aver.Namespace) {
	aver.Always(
		len(value) == COUNT_MAXIMUM,
		"Value output holds every entry of the largest cache.",
	)
}

// Values is caller-owned value storage plus requested write bound.
type Values struct {
	// Storage is caller-owned output sized to largest cache, thus read never grows heap.
	Storage Value_Storage
	// Count bounds this call below full storage.
	Count Value_Count
}

// Values_Invariants requires storage and bounds requested writes.
func Values_Invariants(values Values, namespace aver.Namespace) {
	Value_Storage_Invariants(values.Storage, namespace)
	Value_Count_Invariants(values.Count, namespace)
}

// Capacity is immutable cache entry limit, or zero before initialization.
type Capacity int

// Capacity_Invariants admits caller-owned zero storage before New initializes it.
func Capacity_Invariants(capacity Capacity, namespace aver.Namespace) {
	aver.Tree(capacity, namespace).
		Range_Int(int(capacity), CAPACITY_MINIMUM, CAPACITY_MAXIMUM).
		Ensure()
}

// CAPACITY_MINIMUM is uninitialized caller-owned storage.
const CAPACITY_MINIMUM = 0

// CAPACITY_INITIALIZED_MINIMUM is smallest useful cache.
const CAPACITY_INITIALIZED_MINIMUM = CAPACITY_MINIMUM + 1

// CAPACITY_MAXIMUM follows backing slice boundary.
const CAPACITY_MAXIMUM = COUNT_MAXIMUM

// Entry_Count is active node quantity in one list.
type Entry_Count int

// Entry_Count_Invariants bounds active node quantity.
func Entry_Count_Invariants(count Entry_Count, namespace aver.Namespace) {
	aver.Tree(count, namespace).
		Range_Int(int(count), int(COUNT_MINIMUM), int(COUNT_MAXIMUM)).
		Ensure()
}

// Node_Capacity is initialized part of fixed node pool.
type Node_Capacity int

// Node_Capacity_Invariants bounds initialized node storage.
func Node_Capacity_Invariants(capacity Node_Capacity, namespace aver.Namespace) {
	aver.Tree(capacity, namespace).
		Range_Int(int(capacity), CAPACITY_MINIMUM, CAPACITY_MAXIMUM).
		Ensure()
}

// Free_Position heads one free chain.
type Free_Position Position

// Free_Position_Invariants bounds free-chain head.
func Free_Position_Invariants(position Free_Position, namespace aver.Namespace) {
	aver.Tree(position, namespace).
		Range_Int(int(position), POSITION_MINIMUM, POSITION_MAXIMUM).
		Ensure()
}

// Front_Position heads one recency chain.
type Front_Position Position

// Front_Position_Invariants bounds recency-chain head.
func Front_Position_Invariants(position Front_Position, namespace aver.Namespace) {
	aver.Tree(position, namespace).
		Range_Int(int(position), POSITION_MINIMUM, POSITION_MAXIMUM).
		Ensure()
}

// Back_Position tails one recency chain.
type Back_Position Position

// Back_Position_Invariants bounds recency-chain tail.
func Back_Position_Invariants(position Back_Position, namespace aver.Namespace) {
	aver.Tree(position, namespace).
		Range_Int(int(position), POSITION_MINIMUM, POSITION_MAXIMUM).
		Ensure()
}

// Buckets owns fixed expiry-ring storage.
type Buckets []Expirable_Bucket

// Buckets_Invariants fixes expiry-ring size.
func Buckets_Invariants(buckets Buckets, namespace aver.Namespace) {
	aver.Tree(buckets, namespace).
		Enum_Int(len(buckets), COUNT_MINIMUM, BUCKET_COUNT).
		Ensure()
}

// Evict_Callback runs when a cache discards an entry — through eviction, removal, or purge.
// One type serves every cache here; the upstream declared an identical callback twice
// (simplelru and expirable), which the flat package collapses into this.
type Evict_Callback func(key Key, value Value)

// Entry is one node of the intrusive lists shared by every cache in this package. Its storage is
// owned by a List's pre-allocated pool, not the heap, so an Add reuses a node rather than
// allocating one. Every node lives in the recency ring (Next/Previous); an Expirable node lives
// simultaneously in its expiry bucket chain (Bucket_Next/Bucket_Previous) over the same node.
type Entry struct {
	// Next is newer neighbor or next free node.
	Next Position
	// Previous is older neighbor.
	Previous Position
	// Used distinguishes live nodes from free nodes.
	Used Status
	// Bucket_Next chains toward next entry in expiry bucket.
	Bucket_Next Position
	// Bucket_Previous chains toward previous entry in expiry bucket.
	Bucket_Previous Position
	// Key is the lookup key.
	Key Key
	// Value is the stored value.
	Value Value
	// Expires_At is the Moment past which Expirable treats this entry as gone; zero for the
	// non-expiring caches.
	Expires_At time.Monotonic_Moment
	// Expire_Bucket is which of Expirable's ring buckets holds this entry, so a renew or a
	// remove finds it in O(1); zero for the non-expiring caches.
	Expire_Bucket Bucket_Index
}

// Entry_Invariants composes node handles, status, time, and bucket.
func Entry_Invariants(entry Entry, namespace aver.Namespace) {
	Position_Invariants(entry.Next, namespace)
	Position_Invariants(entry.Previous, namespace)
	Status_Invariants(entry.Used, namespace)
	Position_Invariants(entry.Bucket_Next, namespace)
	Position_Invariants(entry.Bucket_Previous, namespace)
	time.Monotonic_Moment_Invariants(entry.Expires_At, namespace)
	Bucket_Index_Invariants(entry.Expire_Bucket, namespace)
}

// Nodes is caller-owned fixed entry storage.
type Nodes []Entry

// Nodes_Invariants admits nil before initialization and fixed storage afterward.
func Nodes_Invariants(nodes Nodes, namespace aver.Namespace) {
	aver.Tree(nodes, namespace).
		Enum_Int(len(nodes), COUNT_MINIMUM, COUNT_MAXIMUM).
		Ensure()
}

// List is fixed-capacity doubly-linked metadata over caller-owned Nodes. Positions replace
// pointers because pointer cycles cannot form bounded assertion trees. POSITION_NONE closes
// each end. Nodes move between active and free chains, so every operation reuses storage.
type List struct {
	// Nodes points at caller-owned fixed pool.
	Nodes Nodes
	// Capacity is initialized prefix of Nodes.
	Capacity Node_Capacity
	// Free heads free chain.
	Free Free_Position
	// Front is most-recently-used node.
	Front Front_Position
	// Back is least-recently-used node.
	Back Back_Position
	// Count is active node quantity.
	Count Entry_Count
}

// List_Invariants composes pool state and node handles.
func List_Invariants(list List, namespace aver.Namespace) {
	Nodes_Invariants(list.Nodes, namespace)
	Node_Capacity_Invariants(list.Capacity, namespace)
	Free_Position_Invariants(list.Free, namespace)
	Front_Position_Invariants(list.Front, namespace)
	Back_Position_Invariants(list.Back, namespace)
	Entry_Count_Invariants(list.Count, namespace)
	aver.Always(
		(list.Capacity == 0) == (list.Nodes == nil),
		"List capacity and node ownership share initialization state.",
	)
	aver.Always(
		list.Count <= Entry_Count(list.Capacity),
		"List live count does not exceed initialized storage.",
	)
}

// List_Handle borrows mutable list state.
type List_Handle *List

// List_Handle_Invariants composes borrowed list state.
func List_Handle_Invariants(list List_Handle, namespace aver.Namespace) {
	if list == nil {
		return
	}
	List_Invariants(*list, namespace)
}

// List initialization readies caller-owned node storage at fixed capacity.
func list_initialize(l List_Handle, nodes Nodes, capacity_count Count) {
	List_Handle_Invariants(l, "list_initialize.list")
	Nodes_Invariants(nodes, "list_initialize.nodes")
	Count_Invariants(capacity_count, "list_initialize.capacity_count")
	aver.Always(nodes != nil, "List initialization has node storage.")
	clear(nodes)
	l.Nodes = nodes
	l.Capacity = Node_Capacity(capacity_count)
	list_reset(l)
}

// Empties l: zeroes every node, re-threads them all onto Free, and closes the sentinel ring. It
// runs at construction and on Purge, so a purge reclaims every node without touching the heap.
func list_reset(l List_Handle) {
	List_Handle_Invariants(l, "list_reset.list")
	var zero_key Key
	var zero_value Value
	l.Free = Free_Position(POSITION_NONE)
	for index := 0; index < int(l.Capacity); index++ {
		node := &l.Nodes[index]
		node.Key = zero_key
		node.Value = zero_value
		node.Expires_At = 0
		node.Expire_Bucket = 0
		node.Used = false
		node.Previous = POSITION_NONE
		node.Bucket_Next = POSITION_NONE
		node.Bucket_Previous = POSITION_NONE
		node.Next = Position(l.Free)
		l.Free = Free_Position(index)
	}
	l.Front = Front_Position(POSITION_NONE)
	l.Back = Back_Position(POSITION_NONE)
	l.Count = 0
}

// Pops a node from the free-list. It asserts the pool is not empty: every cache evicts before it
// inserts, so peak node use equals capacity and the pool cannot underflow — a nil here is a bug.
func list_allocate(l List_Handle) (position Position) {
	defer func() { Position_Invariants(position, "list_allocate.position") }()
	List_Handle_Invariants(l, "list_allocate.list")
	if l.Free == Free_Position(POSITION_NONE) {
		return POSITION_NONE
	}
	position = Position(l.Free)
	l.Free = Free_Position(l.Nodes[position].Next)
	return position
}

// Returns e to the free-list, zeroing its key and value first so a pooled node pins neither the
// key nor the value it used to hold — the references are released for GC even though the node is
// reused.
func list_free(l List_Handle, position Position) {
	Position_Invariants(position, "list_free.position")
	List_Handle_Invariants(l, "list_free.list")
	var zero_key Key
	var zero_value Value
	e := &l.Nodes[position]
	e.Key = zero_key
	e.Value = zero_value
	e.Expires_At = 0
	e.Expire_Bucket = 0
	e.Used = false
	e.Previous = POSITION_NONE
	e.Bucket_Next = POSITION_NONE
	e.Bucket_Previous = POSITION_NONE
	e.Next = Position(l.Free)
	l.Free = Free_Position(position)
}

// Returns the least-recently-used element (the eviction target) or nil when empty.
func list_back(l List_Handle) (back Position) {
	defer func() { Position_Invariants(back, "list_back.back") }()
	List_Handle_Invariants(l, "list_back.list")
	return Position(l.Back)
}

// Links node at most-recently-used front.
func list_splice_front(l List_Handle, position Position) {
	Position_Invariants(position, "list_splice_front.position")
	List_Handle_Invariants(l, "list_splice_front.list")
	node := &l.Nodes[position]
	node.Previous = POSITION_NONE
	node.Next = Position(l.Front)
	node.Used = true
	if l.Front != Front_Position(POSITION_NONE) {
		l.Nodes[l.Front].Previous = position
	} else {
		l.Back = Back_Position(position)
	}
	l.Front = Front_Position(position)
	l.Count++
}

// Unlinks e from the active ring and returns it to the pool. Callers capture e's key and value
// first, since list_free zeroes them.
func list_remove(l List_Handle, position Position) {
	Position_Invariants(position, "list_remove.position")
	List_Handle_Invariants(l, "list_remove.list")
	e := &l.Nodes[position]
	if e.Previous != POSITION_NONE {
		l.Nodes[e.Previous].Next = e.Next
	} else {
		l.Front = Front_Position(e.Next)
	}
	if e.Next != POSITION_NONE {
		l.Nodes[e.Next].Previous = e.Previous
	} else {
		l.Back = Back_Position(e.Previous)
	}
	l.Count--
	list_free(l, position)
}

// Takes a node from the pool for a non-expiring (k, v) and splices it at the front.
func list_push_front(l List_Handle, key Key, value Value) (pushed Position) {
	defer func() { Position_Invariants(pushed, "list_push_front.pushed") }()
	List_Handle_Invariants(l, "list_push_front.list")
	position := list_allocate(l)
	if position == POSITION_NONE {
		return position
	}
	l.Nodes[position].Key = key
	l.Nodes[position].Value = value
	list_splice_front(l, position)
	return position
}

// Takes a node from the pool for (k, v) carrying its expiry Moment and splices it at the front.
func list_push_front_expirable(
	l List_Handle, key Key, value Value, expires_at time.Monotonic_Moment,
) (pushed Position) {
	defer func() { Position_Invariants(pushed, "list_push_front_expirable.pushed") }()
	List_Handle_Invariants(l, "list_push_front_expirable.list")
	time.Monotonic_Moment_Invariants(expires_at, "list_push_front_expirable.expires_at")
	position := list_allocate(l)
	if position == POSITION_NONE {
		return position
	}
	l.Nodes[position].Key = key
	l.Nodes[position].Value = value
	l.Nodes[position].Expires_At = expires_at
	list_splice_front(l, position)
	return position
}

// Promotes e to the most-recently-used front. It ignores an entry owned by another list or one
// already at the front, so a stray call cannot corrupt l.
func list_move_to_front(l List_Handle, position Position) {
	Position_Invariants(position, "list_move_to_front.position")
	List_Handle_Invariants(l, "list_move_to_front.list")
	if l.Front == Front_Position(position) {
		return
	}
	e := &l.Nodes[position]
	l.Nodes[e.Previous].Next = e.Next
	if e.Next != POSITION_NONE {
		l.Nodes[e.Next].Previous = e.Previous
	} else {
		l.Back = Back_Position(e.Previous)
	}
	e.Previous = POSITION_NONE
	e.Next = Position(l.Front)
	l.Nodes[l.Front].Previous = position
	l.Front = Front_Position(position)
}

// Returns entry one step toward newer end, or POSITION_NONE after front.
func entry_previous(l List_Handle, position Position) (previous Position) {
	defer func() { Position_Invariants(previous, "entry_previous.previous") }()
	Position_Invariants(position, "entry_previous.position")
	List_Handle_Invariants(l, "entry_previous.list")
	return l.Nodes[position].Previous
}

// Finds key without separate allocation-owning index.
func list_find(l List_Handle, key Key) (result Find_Result) {
	defer func() { Find_Result_Invariants(result, "list_find.result") }()
	List_Handle_Invariants(l, "list_find.list")
	for index := 0; index < int(l.Capacity); index++ {
		if l.Nodes[index].Used {
			if l.Nodes[index].Key == key {
				return Find_Result{Position: Position(index), Found: true}
			}
		}
	}
	return Find_Result{Position: POSITION_NONE}
}

// Find_Result holds one lookup position and presence result.
type Find_Result struct {
	// Position identifies matching node or absence.
	Position Position
	// Found reports whether node exists.
	Found Status
}

// Find_Result_Invariants composes lookup results.
func Find_Result_Invariants(result Find_Result, namespace aver.Namespace) {
	Position_Invariants(result.Position, namespace)
	Status_Invariants(result.Found, namespace)
}

// Simple is a fixed-capacity LRU cache. It is single-threaded like every cache here: the upstream
// split a lock-free base (simplelru.LRU) from a thread-safe wrapper (lru.Cache), but a
// deterministic package holds no lock, so the two collapse into this one type and the caller
// serializes access through the io loop. Its Capacity is pre-allocated and immutable.
type Simple struct {
	// Capacity is the fixed maximum, set once at construction and never changed; a full cache
	// evicts the oldest before an insert.
	Capacity Capacity
	// Evict_List orders entries by recency, most-recently-used at the front, over its own pool.
	Evict_List List
	// On_Evict, when non-nil, runs for every entry the cache discards.
	On_Evict Evict_Callback
}

// Simple_Invariants composes capacity, list, and key index.
func Simple_Invariants(cache Simple, namespace aver.Namespace) {
	Capacity_Invariants(cache.Capacity, namespace)
	List_Invariants(cache.Evict_List, namespace)
	aver.Always(
		cache.Capacity == Capacity(cache.Evict_List.Capacity),
		"Simple capacity matches list storage.",
	)
}

// Simple_Handle borrows mutable Simple state.
type Simple_Handle *Simple

// Simple_Handle_Invariants composes borrowed Simple state.
func Simple_Handle_Invariants(cache Simple_Handle, namespace aver.Namespace) {
	if cache == nil {
		return
	}
	Simple_Invariants(*cache, namespace)
}

// New_Simple builds a Simple of the given capacity; on_evict (may be nil) runs for each discarded
// entry. Capacity must be positive. Caller owns c and nodes, so initialization allocates nothing.
func New_Simple(
	c Simple_Handle, nodes Nodes, capacity_count Capacity, on_evict Evict_Callback,
) {
	Simple_Handle_Invariants(c, "new_simple.cache")
	Nodes_Invariants(nodes, "new_simple.nodes")
	Capacity_Invariants(capacity_count, "new_simple.capacity_count")
	aver.Always(capacity_count > 0, "A Simple cache has positive capacity.")
	*c = Simple{Capacity: capacity_count, On_Evict: on_evict}
	list_initialize(&c.Evict_List, nodes, Count(capacity_count))
}

// Simple_Add inserts or overwrites key, returning whether the insert forced an eviction. An
// overwrite renews recency without evicting. A new key into a full cache evicts the oldest first,
// so peak node use never exceeds Capacity and the pool holds.
func Simple_Add(c Simple_Handle, key Key, value Value) (evicted Status) {
	defer func() { Status_Invariants(evicted, "simple_add.evicted") }()
	Simple_Handle_Invariants(c, "simple_add.cache")
	aver.Always(c.Capacity > CAPACITY_MINIMUM, "Simple add cache is initialized.")
	found := list_find(&c.Evict_List, key)
	if found.Found {
		list_move_to_front(&c.Evict_List, found.Position)
		c.Evict_List.Nodes[found.Position].Value = value
		return false
	}
	evicted = c.Evict_List.Count >= Entry_Count(c.Capacity)
	if evicted {
		simple_remove_oldest(c)
	}
	list_push_front(&c.Evict_List, key, value)
	return evicted
}

// Simple_Get returns key's value and renews its recency; a miss returns the zero value and
// false.
func Simple_Get(c Simple_Handle, key Key) (value Value, ok Status) {
	defer func() { Status_Invariants(ok, "simple_get.ok") }()
	Simple_Handle_Invariants(c, "simple_get.cache")
	aver.Always(c.Capacity > CAPACITY_MINIMUM, "Simple get cache is initialized.")
	found := list_find(&c.Evict_List, key)
	if found.Found {
		list_move_to_front(&c.Evict_List, found.Position)
		return c.Evict_List.Nodes[found.Position].Value, true
	}
	return value, false
}

// Simple_Contains reports membership without renewing recency.
func Simple_Contains(c Simple_Handle, key Key) (ok Status) {
	defer func() { Status_Invariants(ok, "simple_contains.ok") }()
	Simple_Handle_Invariants(c, "simple_contains.cache")
	aver.Always(c.Capacity > CAPACITY_MINIMUM, "Simple contains cache is initialized.")
	found := list_find(&c.Evict_List, key)
	ok = found.Found
	return ok
}

// Simple_Peek returns key's value without renewing recency; a miss returns the zero value and
// false.
func Simple_Peek(c Simple_Handle, key Key) (value Value, ok Status) {
	defer func() { Status_Invariants(ok, "simple_peek.ok") }()
	Simple_Handle_Invariants(c, "simple_peek.cache")
	aver.Always(c.Capacity > CAPACITY_MINIMUM, "Simple peek cache is initialized.")
	found := list_find(&c.Evict_List, key)
	if found.Found {
		return c.Evict_List.Nodes[found.Position].Value, true
	}
	return value, false
}

// Simple_Contains_Or_Add inserts key only when absent, reporting whether it was already present
// and whether the insert evicted. A present key is left untouched, its recency unchanged.
func Simple_Contains_Or_Add(
	c Simple_Handle, key Key, value Value,
) (result Conditional_Result) {
	defer func() { Conditional_Result_Invariants(result, "simple_contains_or_add.result") }()
	Simple_Handle_Invariants(c, "simple_contains_or_add.cache")
	aver.Always(
		c.Capacity > CAPACITY_MINIMUM,
		"Simple contains or add cache is initialized.",
	)
	if Simple_Contains(c, key) {
		return Conditional_Result{Present: true}
	}
	result.Evicted = Evicted_Status(Simple_Add(c, key, value))
	return result
}

// Simple_Peek_Or_Add inserts key only when absent, returning the existing value when present
// and reporting whether it was present and whether the insert evicted. A present key's recency
// is left unchanged.
func Simple_Peek_Or_Add(
	c Simple_Handle, key Key, value Value,
) (result Peek_Or_Add_Result) {
	defer func() { Peek_Or_Add_Result_Invariants(result, "simple_peek_or_add.result") }()
	Simple_Handle_Invariants(c, "simple_peek_or_add.cache")
	aver.Always(c.Capacity > CAPACITY_MINIMUM, "Simple peek or add cache is initialized.")
	previous, found := Simple_Peek(c, key)
	result.Previous = previous
	result.Present = Present_Status(found)
	if found {
		return result
	}
	result.Evicted = Evicted_Status(Simple_Add(c, key, value))
	return result
}

// Simple_Remove deletes a present key, reporting whether it was there.
func Simple_Remove(c Simple_Handle, key Key) (present Status) {
	defer func() { Status_Invariants(present, "simple_remove.present") }()
	Simple_Handle_Invariants(c, "simple_remove.cache")
	aver.Always(c.Capacity > CAPACITY_MINIMUM, "Simple remove cache is initialized.")
	found := list_find(&c.Evict_List, key)
	if found.Found {
		simple_remove_element(c, found.Position)
		return true
	}
	return false
}

// Simple_Remove_Oldest deletes and returns the least-recently-used entry; ok is false when the
// cache is empty.
func Simple_Remove_Oldest(c Simple_Handle) (key Key, value Value, ok Status) {
	defer func() { Status_Invariants(ok, "simple_remove_oldest.ok") }()
	Simple_Handle_Invariants(c, "simple_remove_oldest.cache")
	aver.Always(
		c.Capacity > CAPACITY_MINIMUM,
		"Simple remove oldest cache is initialized.",
	)
	ent := list_back(&c.Evict_List)
	if ent == POSITION_NONE {
		return key, value, false
	}
	key = c.Evict_List.Nodes[ent].Key
	value = c.Evict_List.Nodes[ent].Value
	simple_remove_element(c, ent)
	return key, value, true
}

// Simple_Get_Oldest returns the least-recently-used entry without removing it; ok is false when
// the cache is empty.
func Simple_Get_Oldest(c Simple_Handle) (key Key, value Value, ok Status) {
	defer func() { Status_Invariants(ok, "simple_get_oldest.ok") }()
	Simple_Handle_Invariants(c, "simple_get_oldest.cache")
	aver.Always(c.Capacity > CAPACITY_MINIMUM, "Simple get oldest cache is initialized.")
	ent := list_back(&c.Evict_List)
	if ent != POSITION_NONE {
		return c.Evict_List.Nodes[ent].Key, c.Evict_List.Nodes[ent].Value, true
	}
	return key, value, false
}

// Simple_Keys writes the keys, oldest to newest, into buffer up to len(buffer) and returns how
// many it wrote. The caller sizes buffer — pass one of length Simple_Cap to receive them all —
// so the read allocates nothing and is bounded by the caller.
func Simple_Keys(c Simple_Handle, buffer Keys) (count Count) {
	defer func() { Count_Invariants(count, "simple_keys.count") }()
	Simple_Handle_Invariants(c, "simple_keys.cache")
	Keys_Invariants(buffer, "simple_keys.buffer")
	aver.Always(c.Capacity > CAPACITY_MINIMUM, "Simple keys cache is initialized.")
	for ent := list_back(&c.Evict_List); ent != POSITION_NONE; {
		if count >= Count(buffer.Count) {
			break
		}
		buffer.Storage[count] = c.Evict_List.Nodes[ent].Key
		count++
		ent = entry_previous(&c.Evict_List, ent)
	}
	return count
}

// Simple_Values writes the values, oldest to newest, into buffer up to len(buffer) and returns
// how many it wrote. Like Simple_Keys, the caller sizes and thus bounds the read.
func Simple_Values(c Simple_Handle, buffer Values) (count Count) {
	defer func() { Count_Invariants(count, "simple_values.count") }()
	Simple_Handle_Invariants(c, "simple_values.cache")
	Values_Invariants(buffer, "simple_values.buffer")
	aver.Always(c.Capacity > CAPACITY_MINIMUM, "Simple values cache is initialized.")
	for ent := list_back(&c.Evict_List); ent != POSITION_NONE; {
		if count >= Count(buffer.Count) {
			break
		}
		buffer.Storage[count] = c.Evict_List.Nodes[ent].Value
		count++
		ent = entry_previous(&c.Evict_List, ent)
	}
	return count
}

// Simple_Count returns the number of entries held.
func Simple_Count(c Simple_Handle) (count Count) {
	defer func() { Count_Invariants(count, "simple_count.count") }()
	Simple_Handle_Invariants(c, "simple_count.cache")
	aver.Always(c.Capacity > CAPACITY_MINIMUM, "Simple count cache is initialized.")
	return Count(c.Evict_List.Count)
}

// Simple_Cap returns the fixed capacity.
func Simple_Cap(c Simple_Handle) (capacity Capacity) {
	defer func() { Capacity_Invariants(capacity, "simple_cap.capacity") }()
	Simple_Handle_Invariants(c, "simple_cap.cache")
	aver.Always(c.Capacity > CAPACITY_MINIMUM, "Simple cap cache is initialized.")
	return c.Capacity
}

// Simple_Purge empties the cache, calling On_Evict for every discarded entry, and returns every
// node to the pool.
func Simple_Purge(c Simple_Handle) {
	Simple_Handle_Invariants(c, "simple_purge.cache")
	aver.Always(c.Capacity > CAPACITY_MINIMUM, "Simple purge cache is initialized.")
	if c.On_Evict != nil {
		for position := c.Evict_List.Back; position != Back_Position(POSITION_NONE); {
			entry := c.Evict_List.Nodes[position]
			position = Back_Position(entry.Previous)
			c.On_Evict(entry.Key, entry.Value)
		}
	}
	list_reset(&c.Evict_List)
}

// Drops the least-recently-used entry, if any.
func simple_remove_oldest(c Simple_Handle) {
	Simple_Handle_Invariants(c, "simple_remove_oldest_internal.cache")
	ent := list_back(&c.Evict_List)
	if ent != POSITION_NONE {
		simple_remove_element(c, ent)
	}
}

// Unlinks ent, deletes its Items entry, and fires On_Evict. It captures the key and value before
// list_remove frees (and zeroes) the node.
func simple_remove_element(c Simple_Handle, position Position) {
	Position_Invariants(position, "simple_remove_element.position")
	Simple_Handle_Invariants(c, "simple_remove_element.cache")
	key := c.Evict_List.Nodes[position].Key
	value := c.Evict_List.Nodes[position].Value
	list_remove(&c.Evict_List, position)
	if c.On_Evict != nil {
		c.On_Evict(key, value)
	}
}

// DEFAULT_RECENT_RATIO is the share of a Two_Queue held for entries seen once — the upstream
// Default2QRecentRatio of 0.25, as a fixed-point ratio (SCALE/4 is one-quarter of one whole).
const DEFAULT_RECENT_RATIO Recent_Ratio = fixedpoint.SCALE / 4

// DEFAULT_GHOST_RATIO is Two_Queue ghost-list share. It is upstream
// Default2QGhostEntries 0.50 as fixed-point ratio.
const DEFAULT_GHOST_RATIO Ghost_Ratio = fixedpoint.SCALE / 2

// RATIO_MINIMUM is empty capacity share.
const RATIO_MINIMUM int64 = 0

// RATIO_MAXIMUM is whole capacity share.
const RATIO_MAXIMUM int64 = int64(fixedpoint.SCALE)

// CONFIGURED_RATIO_MINIMUM gives one entry at largest cache capacity.
const CONFIGURED_RATIO_MINIMUM = (RATIO_MAXIMUM + int64(COUNT_MAXIMUM) - 1) / int64(COUNT_MAXIMUM)

// Ratio is validated dimensionless scaling input.
type Ratio fixedpoint.Ratio

// Ratio_Invariants bounds scaling from zero through one whole.
func Ratio_Invariants(ratio Ratio, namespace aver.Namespace) {
	aver.Tree(ratio, namespace).
		Range_Int64(int64(ratio), RATIO_MINIMUM, RATIO_MAXIMUM).
		Ensure()
}

// Recent_Ratio is capacity share held for first sightings.
type Recent_Ratio fixedpoint.Ratio

// Recent_Ratio_Invariants bounds recent capacity share.
func Recent_Ratio_Invariants(ratio Recent_Ratio, namespace aver.Namespace) {
	aver.Tree(ratio, namespace).
		Range_Int64(int64(ratio), RATIO_MINIMUM, RATIO_MAXIMUM).
		Ensure()
}

// Ghost_Ratio is capacity share held for ghost keys.
type Ghost_Ratio fixedpoint.Ratio

// Ghost_Ratio_Invariants bounds ghost capacity share.
func Ghost_Ratio_Invariants(ratio Ghost_Ratio, namespace aver.Namespace) {
	aver.Tree(ratio, namespace).
		Range_Int64(int64(ratio), RATIO_MINIMUM, RATIO_MAXIMUM).
		Ensure()
}

// Recent_Ratio_Input admits zero as default request.
type Recent_Ratio_Input fixedpoint.Ratio

// Recent_Ratio_Input_Invariants bounds recent ratio request.
func Recent_Ratio_Input_Invariants(ratio Recent_Ratio_Input, namespace aver.Namespace) {
	aver.Tree(ratio, namespace).
		Range_Int64(int64(ratio), RATIO_MINIMUM, RATIO_MAXIMUM).
		Ensure()
}

// Ghost_Ratio_Input admits zero as default request.
type Ghost_Ratio_Input fixedpoint.Ratio

// Ghost_Ratio_Input_Invariants bounds ghost ratio request.
func Ghost_Ratio_Input_Invariants(ratio Ghost_Ratio_Input, namespace aver.Namespace) {
	aver.Tree(ratio, namespace).
		Range_Int64(int64(ratio), RATIO_MINIMUM, RATIO_MAXIMUM).
		Ensure()
}

// Two_Queue is a scan-resistant 2Q cache (upstream lru.TwoQueueCache). It splits its capacity
// across a recent list (entries seen once) and a frequent list (entries seen again), plus a
// value-less ghost list tracking keys recently evicted from recent — so a burst of one-off keys
// churns only the recent list and cannot flush the frequently used entries. The upstream held
// its three sub-caches behind the LRUCache interface; here they are concrete *Simple, each with
// its own caller-owned pool. Capacity is fixed at construction.
type Two_Queue struct {
	// Capacity is the fixed total across the recent and frequent lists.
	Capacity Capacity
	// Recent_Size caps the recent list; once it is reached, ensure-space spills the oldest
	// recent entry to the ghost list rather than dropping a frequent one.
	Recent_Size Count
	// Recent_Ratio is the fixed-point share of Capacity the recent list may hold.
	Recent_Ratio Recent_Ratio
	// Ghost_Ratio is the fixed-point share of Capacity the ghost list tracks.
	Ghost_Ratio Ghost_Ratio
	// Live stores recent then frequent cache without duplicating one invariant subject type.
	Live Live_Caches
	// Ghost stores value-less recently evicted cache.
	Ghost Ghost_Caches
}

// Two_Queue_Invariants composes fixed split and owned stores.
func Two_Queue_Invariants(cache Two_Queue, namespace aver.Namespace) {
	Capacity_Invariants(cache.Capacity, namespace)
	Count_Invariants(cache.Recent_Size, namespace)
	Recent_Ratio_Invariants(cache.Recent_Ratio, namespace)
	Ghost_Ratio_Invariants(cache.Ghost_Ratio, namespace)
	Live_Caches_Invariants(cache.Live, namespace)
	Ghost_Caches_Invariants(cache.Ghost, namespace)
	aver.Always(
		cache.Recent_Size <= Count(cache.Capacity),
		"Recent limit does not exceed Two_Queue capacity.",
	)
	aver.Always(
		(cache.Recent_Ratio >= Recent_Ratio(CONFIGURED_RATIO_MINIMUM)) ==
			(cache.Capacity > CAPACITY_MINIMUM),
		"Recent ratio shares Two_Queue initialization state.",
	)
	aver.Always(
		(cache.Ghost_Ratio >= Ghost_Ratio(CONFIGURED_RATIO_MINIMUM)) ==
			(cache.Capacity > CAPACITY_MINIMUM),
		"Ghost ratio shares Two_Queue initialization state.",
	)
}

// Two_Queue_Handle borrows mutable Two_Queue state.
type Two_Queue_Handle *Two_Queue

// Two_Queue_Handle_Invariants composes borrowed Two_Queue state.
func Two_Queue_Handle_Invariants(cache Two_Queue_Handle, namespace aver.Namespace) {
	if cache == nil {
		return
	}
	Two_Queue_Invariants(*cache, namespace)
}

// LIVE_CACHE_COUNT holds recent and frequent stores.
const LIVE_CACHE_COUNT = 2

// Recent_List keeps recent cache assertions independent from frequent and ghost chains.
type Recent_List List

// Recent_List_Invariants states fields inline so shared List type occurs no second time.
func Recent_List_Invariants(list Recent_List, namespace aver.Namespace) {
	aver.Tree(list, namespace).
		Enum_Int(len(list.Nodes), COUNT_MINIMUM, COUNT_MAXIMUM).
		Range_Int(int(list.Capacity), CAPACITY_MINIMUM, CAPACITY_MAXIMUM).
		Range_Int(int(list.Free), POSITION_MINIMUM, POSITION_MAXIMUM).
		Range_Int(int(list.Front), POSITION_MINIMUM, POSITION_MAXIMUM).
		Range_Int(int(list.Back), POSITION_MINIMUM, POSITION_MAXIMUM).
		Range_Int(int(list.Count), COUNT_MINIMUM, COUNT_MAXIMUM).
		Ensure()
	aver.Always(
		(list.Capacity == 0) == (list.Nodes == nil),
		"Recent list capacity and node ownership share initialization state.",
	)
	aver.Always(
		list.Count <= Entry_Count(list.Capacity),
		"Recent list count does not exceed initialized storage.",
	)
}

// Recent_Cache keeps recent assertions distinct inside aggregate cache chain.
type Recent_Cache Simple

// Recent_Cache_Invariants composes recent list and inlines cache capacity.
func Recent_Cache_Invariants(cache Recent_Cache, namespace aver.Namespace) {
	Recent_List_Invariants(Recent_List(cache.Evict_List), namespace)
	aver.Tree(cache, namespace).
		Range_Int(int(cache.Capacity), CAPACITY_MINIMUM, CAPACITY_MAXIMUM).
		Ensure()
	aver.Always(
		cache.Capacity == Capacity(cache.Evict_List.Capacity),
		"Recent cache capacity matches list storage.",
	)
}

// Frequent_List keeps frequent cache assertions independent from other cache chains.
type Frequent_List List

// Frequent_List_Invariants states frequent list fields inline.
func Frequent_List_Invariants(list Frequent_List, namespace aver.Namespace) {
	aver.Tree(list, namespace).
		Enum_Int(len(list.Nodes), COUNT_MINIMUM, COUNT_MAXIMUM).
		Range_Int(int(list.Capacity), CAPACITY_MINIMUM, CAPACITY_MAXIMUM).
		Range_Int(int(list.Free), POSITION_MINIMUM, POSITION_MAXIMUM).
		Range_Int(int(list.Front), POSITION_MINIMUM, POSITION_MAXIMUM).
		Range_Int(int(list.Back), POSITION_MINIMUM, POSITION_MAXIMUM).
		Range_Int(int(list.Count), COUNT_MINIMUM, COUNT_MAXIMUM).
		Ensure()
	aver.Always(
		(list.Capacity == 0) == (list.Nodes == nil),
		"Frequent list capacity and node ownership share initialization state.",
	)
	aver.Always(
		list.Count <= Entry_Count(list.Capacity),
		"Frequent list count does not exceed initialized storage.",
	)
}

// Frequent_Cache keeps frequent assertions distinct inside aggregate cache chain.
type Frequent_Cache Simple

// Frequent_Cache_Invariants composes frequent list and inlines cache capacity.
func Frequent_Cache_Invariants(cache Frequent_Cache, namespace aver.Namespace) {
	Frequent_List_Invariants(Frequent_List(cache.Evict_List), namespace)
	aver.Tree(cache, namespace).
		Range_Int(int(cache.Capacity), CAPACITY_MINIMUM, CAPACITY_MAXIMUM).
		Ensure()
	aver.Always(
		cache.Capacity == Capacity(cache.Evict_List.Capacity),
		"Frequent cache capacity matches list storage.",
	)
}

// Live_Caches gives each list semantic type so one assertion chain repeats no subject.
type Live_Caches struct {
	// Recent owns entries seen one time.
	Recent Recent_Cache
	// Frequent owns entries seen more than one time.
	Frequent Frequent_Cache
}

// Live_Caches_Invariants fixes live-cache pair and requires both stores.
func Live_Caches_Invariants(caches Live_Caches, namespace aver.Namespace) {
	Recent_Cache_Invariants(caches.Recent, namespace)
	Frequent_Cache_Invariants(caches.Frequent, namespace)
}

// GHOST_CACHE_COUNT holds one ghost store.
const GHOST_CACHE_COUNT = 1

// Ghost_List keeps ghost cache assertions independent from live cache chains.
type Ghost_List List

// Ghost_List_Invariants states ghost list fields inline.
func Ghost_List_Invariants(list Ghost_List, namespace aver.Namespace) {
	aver.Tree(list, namespace).
		Enum_Int(len(list.Nodes), COUNT_MINIMUM, COUNT_MAXIMUM).
		Range_Int(int(list.Capacity), CAPACITY_MINIMUM, CAPACITY_MAXIMUM).
		Range_Int(int(list.Free), POSITION_MINIMUM, POSITION_MAXIMUM).
		Range_Int(int(list.Front), POSITION_MINIMUM, POSITION_MAXIMUM).
		Range_Int(int(list.Back), POSITION_MINIMUM, POSITION_MAXIMUM).
		Range_Int(int(list.Count), COUNT_MINIMUM, COUNT_MAXIMUM).
		Ensure()
	aver.Always(
		(list.Capacity == 0) == (list.Nodes == nil),
		"Ghost list capacity and node ownership share initialization state.",
	)
	aver.Always(
		list.Count <= Entry_Count(list.Capacity),
		"Ghost list count does not exceed initialized storage.",
	)
}

// Ghost_Cache keeps ghost assertions distinct inside aggregate cache chain.
type Ghost_Cache Simple

// Ghost_Cache_Invariants composes ghost list and inlines cache capacity.
func Ghost_Cache_Invariants(cache Ghost_Cache, namespace aver.Namespace) {
	Ghost_List_Invariants(Ghost_List(cache.Evict_List), namespace)
	aver.Tree(cache, namespace).
		Range_Int(int(cache.Capacity), CAPACITY_MINIMUM, CAPACITY_MAXIMUM).
		Ensure()
	aver.Always(
		cache.Capacity == Capacity(cache.Evict_List.Capacity),
		"Ghost cache capacity matches list storage.",
	)
}

// Ghost_Caches gives ghost storage one semantic cache type.
type Ghost_Caches struct {
	// Cache owns value-less recently evicted entries.
	Cache Ghost_Cache
}

// Ghost_Caches_Invariants requires ghost storage.
func Ghost_Caches_Invariants(caches Ghost_Caches, namespace aver.Namespace) {
	Ghost_Cache_Invariants(caches.Cache, namespace)
}

// Recent_Nodes owns recent list entries without sharing assertion identity.
type Recent_Nodes []Entry

// Recent_Nodes_Invariants admits zero state and complete caller storage.
func Recent_Nodes_Invariants(nodes Recent_Nodes, namespace aver.Namespace) {
	aver.Tree(nodes, namespace).
		Enum_Int(len(nodes), COUNT_MINIMUM, COUNT_MAXIMUM).
		Ensure()
}

// Frequent_Nodes owns frequent list entries without sharing assertion identity.
type Frequent_Nodes []Entry

// Frequent_Nodes_Invariants admits zero state and complete caller storage.
func Frequent_Nodes_Invariants(nodes Frequent_Nodes, namespace aver.Namespace) {
	aver.Tree(nodes, namespace).
		Enum_Int(len(nodes), COUNT_MINIMUM, COUNT_MAXIMUM).
		Ensure()
}

// Live_Nodes separates recent and frequent node assertion identities.
type Live_Nodes struct {
	// Recent backs recent cache.
	Recent Recent_Nodes
	// Frequent backs frequent cache.
	Frequent Frequent_Nodes
}

// Live_Nodes_Invariants admits empty storage before initialization or complete pair.
func Live_Nodes_Invariants(nodes Live_Nodes, namespace aver.Namespace) {
	Recent_Nodes_Invariants(nodes.Recent, namespace)
	Frequent_Nodes_Invariants(nodes.Frequent, namespace)
}

// Ghost_Node_Storage owns ghost entries without sharing live assertion identity.
type Ghost_Node_Storage []Entry

// Ghost_Node_Storage_Invariants admits zero state and complete caller storage.
func Ghost_Node_Storage_Invariants(nodes Ghost_Node_Storage, namespace aver.Namespace) {
	aver.Tree(nodes, namespace).
		Enum_Int(len(nodes), COUNT_MINIMUM, COUNT_MAXIMUM).
		Ensure()
}

// Ghost_Nodes stores one ghost node pool.
type Ghost_Nodes struct {
	// Cache backs ghost cache.
	Cache Ghost_Node_Storage
}

// Ghost_Nodes_Invariants admits empty storage before initialization or complete set.
func Ghost_Nodes_Invariants(nodes Ghost_Nodes, namespace aver.Namespace) {
	Ghost_Node_Storage_Invariants(nodes.Cache, namespace)
}

// Two_Queue_Nodes owns fixed node storage for recent, frequent, and ghost lists.
type Two_Queue_Nodes struct {
	// Live stores recent then frequent nodes.
	Live Live_Nodes
	// Ghost stores value-less ghost nodes.
	Ghost Ghost_Nodes
}

// Two_Queue_Nodes_Invariants requires caller storage.
func Two_Queue_Nodes_Invariants(nodes Two_Queue_Nodes, namespace aver.Namespace) {
	Live_Nodes_Invariants(nodes.Live, namespace)
	Ghost_Nodes_Invariants(nodes.Ghost, namespace)
}

// Two_Queue_Nodes_Handle borrows caller-owned Two_Queue storage.
type Two_Queue_Nodes_Handle *Two_Queue_Nodes

// Two_Queue_Nodes_Handle_Invariants composes caller-owned Two_Queue storage.
func Two_Queue_Nodes_Handle_Invariants(nodes Two_Queue_Nodes_Handle, namespace aver.Namespace) {
	if nodes == nil {
		return
	}
	Two_Queue_Nodes_Invariants(*nodes, namespace)
}

// Two_Queue_Input configures New_Two_Queue. It is not generic: neither the capacity nor the ratios
// carry a key or value type.
type Two_Queue_Input struct {
	// Capacity is the fixed total across the recent and frequent lists.
	Capacity Capacity
	// Recent_Ratio is the fixed-point share of Capacity the recent list may hold; a zero value
	// defaults to DEFAULT_RECENT_RATIO.
	Recent_Ratio Recent_Ratio_Input
	// Ghost_Ratio is the fixed-point share of Capacity the ghost list tracks; a zero value
	// defaults to DEFAULT_GHOST_RATIO.
	Ghost_Ratio Ghost_Ratio_Input
}

// Two_Queue_Input_Invariants composes requested capacity split.
func Two_Queue_Input_Invariants(input Two_Queue_Input, namespace aver.Namespace) {
	Capacity_Invariants(input.Capacity, namespace)
	Recent_Ratio_Input_Invariants(input.Recent_Ratio, namespace)
	Ghost_Ratio_Input_Invariants(input.Ghost_Ratio, namespace)
}

// New_Two_Queue initializes caller cache and node storage, defaulting unset ratios. Capacity
// must be positive and each ratio must lie in [0, 1]; the ghost
// list must come out non-empty, so a capacity too small for the ghost ratio panics — all
// programmer errors, asserted rather than returned.
func New_Two_Queue(c Two_Queue_Handle, nodes Two_Queue_Nodes_Handle, input Two_Queue_Input) {
	Two_Queue_Handle_Invariants(c, "new_two_queue.cache")
	Two_Queue_Nodes_Handle_Invariants(nodes, "new_two_queue.nodes")
	Two_Queue_Input_Invariants(input, "new_two_queue.input")
	recent_ratio := Recent_Ratio(input.Recent_Ratio)
	if recent_ratio == 0 {
		recent_ratio = DEFAULT_RECENT_RATIO
	}
	ghost_ratio := Ghost_Ratio(input.Ghost_Ratio)
	if ghost_ratio == 0 {
		ghost_ratio = DEFAULT_GHOST_RATIO
	}
	aver.Always(input.Capacity > 0, "A Two_Queue cache has positive capacity.")
	aver.Always(
		recent_ratio >= Recent_Ratio(CONFIGURED_RATIO_MINIMUM),
		"A recent ratio gives storage at maximum capacity.",
	)
	aver.Always(recent_ratio <= fixedpoint.SCALE, "A recent ratio does not exceed one.")
	aver.Always(
		ghost_ratio >= Ghost_Ratio(CONFIGURED_RATIO_MINIMUM),
		"A ghost ratio gives storage at maximum capacity.",
	)
	aver.Always(ghost_ratio <= fixedpoint.SCALE, "A ghost ratio does not exceed one.")
	recent_size := ratio_of(Count(input.Capacity), Ratio(recent_ratio))
	evict_size := ratio_of(Count(input.Capacity), Ratio(ghost_ratio))
	aver.Always(evict_size > 0, "A ghost ratio gives at least one ghost entry.")
	*c = Two_Queue{
		Capacity:     input.Capacity,
		Recent_Size:  recent_size,
		Recent_Ratio: recent_ratio,
		Ghost_Ratio:  ghost_ratio,
	}
	c.Live.Recent.Capacity = input.Capacity
	list_initialize(
		&c.Live.Recent.Evict_List, Nodes(nodes.Live.Recent), Count(input.Capacity),
	)
	c.Live.Frequent.Capacity = input.Capacity
	list_initialize(
		&c.Live.Frequent.Evict_List, Nodes(nodes.Live.Frequent), Count(input.Capacity),
	)
	c.Ghost.Cache.Capacity = Capacity(evict_size)
	list_initialize(&c.Ghost.Cache.Evict_List, Nodes(nodes.Ghost.Cache), evict_size)
}

// Two_Queue_Get returns key's value, promoting a recent hit to the frequent list; a miss returns
// the zero value and false. A frequent hit renews recency in place.
func Two_Queue_Get(c Two_Queue_Handle, key Key) (value Value, ok Status) {
	defer func() { Status_Invariants(ok, "two_queue_get.ok") }()
	Two_Queue_Handle_Invariants(c, "two_queue_get.cache")
	aver.Always(c.Capacity > CAPACITY_MINIMUM, "Two queue get cache is initialized.")
	frequent := list_find(&c.Live.Frequent.Evict_List, key)
	if frequent.Found {
		list_move_to_front(&c.Live.Frequent.Evict_List, frequent.Position)
		return c.Live.Frequent.Evict_List.Nodes[frequent.Position].Value, true
	}
	// A recent hit is the second access, so promote it into the frequent list.
	recent := list_find(&c.Live.Recent.Evict_List, key)
	if recent.Found {
		value = c.Live.Recent.Evict_List.Nodes[recent.Position].Value
		list_remove(&c.Live.Recent.Evict_List, recent.Position)
		list_push_front(&c.Live.Frequent.Evict_List, key, value)
		return value, true
	}
	return value, false
}

// Two_Queue_Add inserts or refreshes key. A frequent key is updated in place, a recent key is
// promoted to frequent, a ghost key returns as frequent, and a brand-new key lands in recent.
func Two_Queue_Add(c Two_Queue_Handle, key Key, value Value) {
	Two_Queue_Handle_Invariants(c, "two_queue_add.cache")
	aver.Always(c.Capacity > CAPACITY_MINIMUM, "Two queue add cache is initialized.")
	frequent := list_find(&c.Live.Frequent.Evict_List, key)
	if frequent.Found {
		list_move_to_front(&c.Live.Frequent.Evict_List, frequent.Position)
		c.Live.Frequent.Evict_List.Nodes[frequent.Position].Value = value
		return
	}
	recent := list_find(&c.Live.Recent.Evict_List, key)
	if recent.Found {
		list_remove(&c.Live.Recent.Evict_List, recent.Position)
		list_push_front(&c.Live.Frequent.Evict_List, key, value)
		return
	}
	// A ghost hit means the key was frequent-worthy before being evicted, so it returns as
	// frequent; ensure space treating this as a ghost-driven add.
	ghost := list_find(&c.Ghost.Cache.Evict_List, key)
	if ghost.Found {
		two_queue_ensure_space(c, true)
		list_remove(&c.Ghost.Cache.Evict_List, ghost.Position)
		list_push_front(&c.Live.Frequent.Evict_List, key, value)
		return
	}
	two_queue_ensure_space(c, false)
	list_push_front(&c.Live.Recent.Evict_List, key, value)
}

// Two_Queue_Contains reports membership across the frequent and recent lists (not the ghost list)
// without promoting the key.
func Two_Queue_Contains(c Two_Queue_Handle, key Key) (ok Status) {
	defer func() { Status_Invariants(ok, "two_queue_contains.ok") }()
	Two_Queue_Handle_Invariants(c, "two_queue_contains.cache")
	aver.Always(c.Capacity > CAPACITY_MINIMUM, "Two queue contains cache is initialized.")
	if list_find(&c.Live.Frequent.Evict_List, key).Found {
		return true
	}
	return list_find(&c.Live.Recent.Evict_List, key).Found
}

// Two_Queue_Peek returns key's value across the frequent and recent lists without promoting it; a
// miss returns the zero value and false.
func Two_Queue_Peek(c Two_Queue_Handle, key Key) (value Value, ok Status) {
	defer func() { Status_Invariants(ok, "two_queue_peek.ok") }()
	Two_Queue_Handle_Invariants(c, "two_queue_peek.cache")
	aver.Always(c.Capacity > CAPACITY_MINIMUM, "Two queue peek cache is initialized.")
	frequent := list_find(&c.Live.Frequent.Evict_List, key)
	if frequent.Found {
		return c.Live.Frequent.Evict_List.Nodes[frequent.Position].Value, true
	}
	recent := list_find(&c.Live.Recent.Evict_List, key)
	if recent.Found {
		return c.Live.Recent.Evict_List.Nodes[recent.Position].Value, true
	}
	return value, false
}

// Two_Queue_Remove deletes key from whichever of the frequent, recent, or ghost lists holds it.
func Two_Queue_Remove(c Two_Queue_Handle, key Key) {
	Two_Queue_Handle_Invariants(c, "two_queue_remove.cache")
	aver.Always(c.Capacity > CAPACITY_MINIMUM, "Two queue remove cache is initialized.")
	frequent := list_find(&c.Live.Frequent.Evict_List, key)
	if frequent.Found {
		list_remove(&c.Live.Frequent.Evict_List, frequent.Position)
		return
	}
	recent := list_find(&c.Live.Recent.Evict_List, key)
	if recent.Found {
		list_remove(&c.Live.Recent.Evict_List, recent.Position)
		return
	}
	ghost := list_find(&c.Ghost.Cache.Evict_List, key)
	if ghost.Found {
		list_remove(&c.Ghost.Cache.Evict_List, ghost.Position)
	}
}

// Two_Queue_Keys writes the keys into buffer — frequent entries first, then recent, each oldest
// to newest — up to len(buffer), and returns how many it wrote. Zero-alloc and caller-bounded.
func Two_Queue_Keys(c Two_Queue_Handle, buffer Keys) (count Count) {
	defer func() { Count_Invariants(count, "two_queue_keys.count") }()
	Two_Queue_Handle_Invariants(c, "two_queue_keys.cache")
	Keys_Invariants(buffer, "two_queue_keys.buffer")
	aver.Always(c.Capacity > CAPACITY_MINIMUM, "Two queue keys cache is initialized.")
	lists := [LIVE_CACHE_COUNT]List_Handle{
		&c.Live.Frequent.Evict_List,
		&c.Live.Recent.Evict_List,
	}
	for _, list := range lists {
		for position := list_back(list); position != POSITION_NONE; {
			if count >= Count(buffer.Count) {
				return count
			}
			buffer.Storage[count] = list.Nodes[position].Key
			count++
			position = entry_previous(list, position)
		}
	}
	return count
}

// Two_Queue_Values writes the values into buffer — frequent first, then recent, each oldest to
// newest — up to len(buffer), and returns how many it wrote.
func Two_Queue_Values(c Two_Queue_Handle, buffer Values) (count Count) {
	defer func() { Count_Invariants(count, "two_queue_values.count") }()
	Two_Queue_Handle_Invariants(c, "two_queue_values.cache")
	Values_Invariants(buffer, "two_queue_values.buffer")
	aver.Always(c.Capacity > CAPACITY_MINIMUM, "Two queue values cache is initialized.")
	lists := [LIVE_CACHE_COUNT]List_Handle{
		&c.Live.Frequent.Evict_List,
		&c.Live.Recent.Evict_List,
	}
	for _, list := range lists {
		for position := list_back(list); position != POSITION_NONE; {
			if count >= Count(buffer.Count) {
				return count
			}
			buffer.Storage[count] = list.Nodes[position].Value
			count++
			position = entry_previous(list, position)
		}
	}
	return count
}

// Two_Queue_Count returns the number of live entries (frequent plus recent, excluding ghosts).
func Two_Queue_Count(c Two_Queue_Handle) (count Count) {
	defer func() { Count_Invariants(count, "two_queue_count.count") }()
	Two_Queue_Handle_Invariants(c, "two_queue_count.cache")
	aver.Always(c.Capacity > CAPACITY_MINIMUM, "Two queue count cache is initialized.")
	return Count(c.Live.Frequent.Evict_List.Count + c.Live.Recent.Evict_List.Count)
}

// Two_Queue_Cap returns the fixed total capacity.
func Two_Queue_Cap(c Two_Queue_Handle) (capacity Capacity) {
	defer func() { Capacity_Invariants(capacity, "two_queue_cap.capacity") }()
	Two_Queue_Handle_Invariants(c, "two_queue_cap.cache")
	aver.Always(c.Capacity > CAPACITY_MINIMUM, "Two queue cap cache is initialized.")
	return c.Capacity
}

// Two_Queue_Purge empties the frequent, recent, and ghost lists.
func Two_Queue_Purge(c Two_Queue_Handle) {
	Two_Queue_Handle_Invariants(c, "two_queue_purge.cache")
	aver.Always(c.Capacity > CAPACITY_MINIMUM, "Two queue purge cache is initialized.")
	list_reset(&c.Live.Recent.Evict_List)
	list_reset(&c.Live.Frequent.Evict_List)
	list_reset(&c.Ghost.Cache.Evict_List)
}

// Makes room for one insert. When the recent and frequent lists together fill the cache, it
// evicts the oldest recent entry to the ghost list if recent is at or over its target, otherwise
// it drops the oldest frequent entry. A ghost-driven add (recent_evict) defers to frequent at the
// exact target, so a promotion from the ghost list does not immediately re-evict recent.
func two_queue_ensure_space(c Two_Queue_Handle, recent_evict Status) {
	Status_Invariants(recent_evict, "two_queue_ensure_space.recent_evict")
	Two_Queue_Handle_Invariants(c, "two_queue_ensure_space.cache")
	recent_count := Count(c.Live.Recent.Evict_List.Count)
	frequent_count := Count(c.Live.Frequent.Evict_List.Count)
	if recent_count+frequent_count < Count(c.Capacity) {
		return
	}
	if recent_count > 0 {
		evict_from_recent := false
		if recent_count > c.Recent_Size {
			evict_from_recent = true
		}
		if recent_count == c.Recent_Size {
			if !recent_evict {
				evict_from_recent = true
			}
		}
		if evict_from_recent {
			recent_back := list_back(&c.Live.Recent.Evict_List)
			key := c.Live.Recent.Evict_List.Nodes[recent_back].Key
			list_remove(&c.Live.Recent.Evict_List, recent_back)
			if c.Ghost.Cache.Evict_List.Count >= Entry_Count(c.Ghost.Cache.Capacity) {
				ghost_back := list_back(&c.Ghost.Cache.Evict_List)
				list_remove(&c.Ghost.Cache.Evict_List, ghost_back)
			}
			list_push_front(&c.Ghost.Cache.Evict_List, key, struct{}{})
			return
		}
	}
	frequent_back := list_back(&c.Live.Frequent.Evict_List)
	if frequent_back != POSITION_NONE {
		list_remove(&c.Live.Frequent.Evict_List, frequent_back)
	}
}

// Scales size by a fixed-point ratio, truncating toward zero — the deterministic replacement for
// the upstream int(float64(size) * ratio). fixedpoint is the house stand-in for float64; Apply
// scales a Number by a dimensionless Ratio.
func ratio_of(size Count, ratio Ratio) (scaled Count) {
	defer func() { Count_Invariants(scaled, "ratio_of.scaled") }()
	Count_Invariants(size, "ratio_of.size")
	Ratio_Invariants(ratio, "ratio_of.ratio")
	number := fixedpoint.Number(fixedpoint.From_Integer(fixedpoint.Whole_Integer(size)))
	product := fixedpoint.Apply(number, fixedpoint.Ratio(ratio))
	return Count(fixedpoint.Whole(product))
}

// BUCKET_COUNT is the size of Expirable's expiry ring. It must fit a uint8, since both
// Next_Cleanup_Bucket and Entry.Expire_Bucket are uint8, and it matches the upstream numBuckets.
const BUCKET_COUNT = 100

// TTL is positive cache lifetime inside the monotonic-clock domain.
type TTL time.Duration

// TTL_Invariants keeps expiry addition inside bounded monotonic time.
func TTL_Invariants(ttl TTL, namespace aver.Namespace) {
	aver.Tree(ttl, namespace).
		Range_Int64(int64(ttl), int64(TTL_MINIMUM), int64(TTL_MAXIMUM)).
		Ensure()
}

// TTL_MINIMUM is uninitialized caller-owned storage.
const TTL_MINIMUM = TTL(0)

// TTL_INITIALIZED_MINIMUM is one clock grain.
const TTL_INITIALIZED_MINIMUM = TTL_MINIMUM + TTL(time.NANOSECOND)

// TTL_MAXIMUM reaches the monotonic-clock bound from boot.
const TTL_MAXIMUM = TTL(time.MONOTONIC_MOMENT_MAXIMUM)

// Cleanup_Interval is one positive expiry-ring step.
type Cleanup_Interval time.Duration

// Cleanup_Interval_Invariants bounds one ring step by largest TTL divided across the ring.
func Cleanup_Interval_Invariants(interval Cleanup_Interval, namespace aver.Namespace) {
	aver.Tree(interval, namespace).
		Range_Int64(
			int64(interval),
			int64(CLEANUP_INTERVAL_MINIMUM),
			int64(CLEANUP_INTERVAL_MAXIMUM),
		).
		Ensure()
}

// CLEANUP_INTERVAL_MINIMUM advances the timeline by one clock grain.
const CLEANUP_INTERVAL_MINIMUM = Cleanup_Interval(time.NANOSECOND)

// CLEANUP_INTERVAL_MAXIMUM spreads largest admitted TTL across one ring lap.
const CLEANUP_INTERVAL_MAXIMUM = Cleanup_Interval(TTL_MAXIMUM / BUCKET_COUNT)

// Expirable_Bucket groups entries whose TTLs land in the same slice of the expiry ring, so the
// cleanup sweep reaps a whole cohort at once instead of scanning every entry. The entries chain
// intrusively through their nodes (Bucket_Next/Bucket_Previous), so the bucket allocates nothing.
type Expirable_Bucket struct {
	// Head is first entry in bucket chain, or POSITION_NONE when empty.
	Head Position
	// Newest_Entry is the latest Expires_At among the chain, so cleanup can tell when the whole
	// bucket has passed and is safe to reap in one pass.
	Newest_Entry time.Monotonic_Moment
}

// Expirable_Bucket_Invariants composes chain head and expiry bound.
func Expirable_Bucket_Invariants(bucket Expirable_Bucket, namespace aver.Namespace) {
	Position_Invariants(bucket.Head, namespace)
	time.Monotonic_Moment_Invariants(bucket.Newest_Entry, namespace)
}

// Cache_Clock is zero before initialization or one complete injected clock.
type Cache_Clock time.Clock

// Cache_Clock_Invariants admits zero storage and complete initialized clock.
func Cache_Clock_Invariants(clock Cache_Clock, _ aver.Namespace) {
	aver.Always(
		(clock.Now_Monotonic == nil) == (clock.Now_Realtime == nil),
		"A cache clock is empty or complete.",
	)
}

// Cache_Timeline is zero before initialization or one complete injected timeline.
type Cache_Timeline nbio.Timeline

// Cache_Timeline_Invariants admits zero storage and complete timeline vtable.
func Cache_Timeline_Invariants(timeline Cache_Timeline, _ aver.Namespace) {
	empty := timeline.Submit == nil
	aver.Always(
		(timeline.Open_Event == nil) == empty,
		"A cache timeline event opener matches initialization state.",
	)
	aver.Always(
		(timeline.Event_Listen == nil) == empty,
		"A cache timeline event listener matches initialization state.",
	)
	aver.Always(
		(timeline.Event_Trigger == nil) == empty,
		"A cache timeline event trigger matches initialization state.",
	)
	aver.Always(
		(timeline.Close_Event == nil) == empty,
		"A cache timeline event closer matches initialization state.",
	)
}

// Cleanup owns callback state beside Completion, whose address reaches this wrapper.
type Cleanup struct {
	// Completion must stay first so callback can recover Cleanup without pointer arithmetic.
	Completion nbio.Completion
	// Due records one fired timeout until cache access performs bounded generic cleanup.
	Due Status
}

// Cleanup_Invariants admits zero storage and complete initialized callback state.
func Cleanup_Invariants(cleanup Cleanup, namespace aver.Namespace) {
	Status_Invariants(cleanup.Due, namespace)
}

// Expirable is fixed-capacity LRU with TTL. Injected Clock removes wall time. Injected Timeline
// removes ticker goroutine. Timeout marks cleanup due. Next access reaps all ripe buckets before
// returning observable state, then re-arms timeout. Reads reject stale entry even before sweep.
// Caller keeps cache address stable because Timeline tracks embedded Completion by address.
type Expirable struct {
	// Capacity is the fixed maximum; adding past it evicts the least-recently-used entry.
	Capacity Capacity
	// TTL is how long past insertion an entry stays live.
	TTL TTL
	// Clock is the injected time source; expiry is measured against Now_Monotonic so it never
	// reads a wall clock and replays deterministically under a virtual clock.
	Clock Cache_Clock
	// Timeline is injected because only composition root or test may drive it.
	Timeline Cache_Timeline
	// Cleanup is caller-owned timeout storage and due state.
	Cleanup Cleanup
	// Evict_List orders entries by recency, most-recently-used at the front, over its own pool.
	Evict_List List
	// On_Evict, when non-nil, runs for every entry the cache discards.
	On_Evict Evict_Callback
	// Buckets is the expiry ring; an entry is filed by insertion time so cleanup reaps cohorts.
	Buckets Buckets
	// Next_Cleanup_Bucket is the ring position the cleanup sweep will consider next.
	Next_Cleanup_Bucket Bucket_Index
}

// Expirable_Invariants composes cache storage and injected timeline.
func Expirable_Invariants(cache Expirable, namespace aver.Namespace) {
	Capacity_Invariants(cache.Capacity, namespace)
	TTL_Invariants(cache.TTL, namespace)
	Cache_Clock_Invariants(cache.Clock, namespace)
	Cache_Timeline_Invariants(cache.Timeline, namespace)
	Cleanup_Invariants(cache.Cleanup, namespace)
	List_Invariants(cache.Evict_List, namespace)
	Buckets_Invariants(cache.Buckets, namespace)
	Bucket_Index_Invariants(cache.Next_Cleanup_Bucket, namespace)
	aver.Always(
		cache.Capacity == Capacity(cache.Evict_List.Capacity),
		"Expirable capacity matches list storage.",
	)
	aver.Always(
		(cache.TTL >= TTL_INITIALIZED_MINIMUM) ==
			(cache.Capacity > CAPACITY_MINIMUM),
		"TTL shares Expirable initialization state.",
	)
	aver.Always(
		(cache.Clock.Now_Monotonic != nil) ==
			(cache.Capacity > CAPACITY_MINIMUM),
		"Clock shares Expirable initialization state.",
	)
	aver.Always(
		(cache.Timeline.Submit != nil) ==
			(cache.Capacity > CAPACITY_MINIMUM),
		"Timeline shares Expirable initialization state.",
	)
}

// Expirable_Handle borrows mutable Expirable state.
type Expirable_Handle *Expirable

// Expirable_Handle_Invariants composes borrowed Expirable state.
func Expirable_Handle_Invariants(cache Expirable_Handle, namespace aver.Namespace) {
	if cache == nil {
		return
	}
	Expirable_Invariants(*cache, namespace)
}

// Expirable_Input configures New_Expirable.
type Expirable_Input struct {
	// Capacity is the fixed maximum; adding past it evicts the least-recently-used entry.
	Capacity Capacity
	// TTL is how long past insertion an entry stays live.
	TTL TTL
	// Clock is the injected time source expiry is measured against; required.
	Clock time.Clock
	// Timeline is the injected timer the cleanup sweep rides; required.
	Timeline nbio.Timeline
	// Buckets is caller-owned expiry-ring storage.
	Buckets Buckets
	// On_Evict, when non-nil, runs for every entry the cache discards.
	On_Evict Evict_Callback
}

// Expirable_Input_Invariants composes cache bounds and injected timeline.
func Expirable_Input_Invariants(input Expirable_Input, namespace aver.Namespace) {
	Capacity_Invariants(input.Capacity, namespace)
	TTL_Invariants(input.TTL, namespace)
	time.Clock_Invariants(input.Clock, namespace)
	nbio.Timeline_Invariants(input.Timeline, namespace)
	Buckets_Invariants(input.Buckets, namespace)
}

// New_Expirable initializes caller cache and node storage, then arms cleanup timer. Positive
// Capacity and TTL plus complete Clock and Timeline are programmer requirements. The upstream
// size<=0 "unlimited" and ttl<=0 "no expiry" modes are gone. No-expiry cache is Simple.
func New_Expirable(c Expirable_Handle, nodes Nodes, input Expirable_Input) {
	Expirable_Handle_Invariants(c, "new_expirable.cache")
	Nodes_Invariants(nodes, "new_expirable.nodes")
	Expirable_Input_Invariants(input, "new_expirable.input")
	aver.Always(
		!c.Cleanup.Completion.Armed,
		"Expirable initialization does not replace armed cleanup.",
	)
	aver.Always(input.Capacity > 0, "An Expirable cache has positive capacity.")
	aver.Always(
		input.TTL >= TTL_INITIALIZED_MINIMUM,
		"An Expirable cache has positive TTL.",
	)
	aver.Always(input.Clock.Now_Monotonic != nil, "An Expirable cache has a clock.")
	aver.Always(input.Timeline.Submit != nil, "An Expirable cache has a timeline.")
	aver.Always(len(input.Buckets) == BUCKET_COUNT, "An Expirable cache has one bucket ring.")
	*c = Expirable{
		Capacity: input.Capacity,
		TTL:      input.TTL,
		Clock:    Cache_Clock(input.Clock),
		Timeline: Cache_Timeline(input.Timeline),
		On_Evict: input.On_Evict,
		Buckets:  input.Buckets,
	}
	list_initialize(&c.Evict_List, nodes, Count(input.Capacity))
	for index := range c.Buckets {
		c.Buckets[index].Head = POSITION_NONE
	}
	expirable_arm_cleanup(c)
}

// Expirable_Add inserts or overwrites key, stamping its expiry TTL from now, and returns whether
// the insert forced a size eviction. An overwrite renews both recency and expiry. A new key into
// a full cache evicts the oldest first, so peak node use never exceeds Capacity.
func Expirable_Add(c Expirable_Handle, key Key, value Value) (evicted Status) {
	defer func() { Status_Invariants(evicted, "expirable_add.evicted") }()
	Expirable_Handle_Invariants(c, "expirable_add.cache")
	aver.Always(c.Capacity > CAPACITY_MINIMUM, "Expirable add cache is initialized.")
	expirable_reap_due(c)
	now := time.Clock_Now_Monotonic(time.Clock(c.Clock))
	expires_at := now + time.Monotonic_Moment(c.TTL)
	found := list_find(&c.Evict_List, key)
	if found.Found {
		list_move_to_front(&c.Evict_List, found.Position)
		// The renewed expiry belongs in a different bucket, so refile it.
		expirable_remove_from_bucket(c, found.Position)
		c.Evict_List.Nodes[found.Position].Value = value
		c.Evict_List.Nodes[found.Position].Expires_At = expires_at
		expirable_add_to_bucket(c, found.Position)
		return false
	}
	evicted = c.Evict_List.Count >= Entry_Count(c.Capacity)
	if evicted {
		expirable_remove_oldest(c)
	}
	ent := list_push_front_expirable(&c.Evict_List, key, value, expires_at)
	expirable_add_to_bucket(c, ent)
	return evicted
}

// Expirable_Get returns key's value and renews its recency; an entry past its TTL is rejected
// as a miss, and a genuine miss returns the zero value and false.
func Expirable_Get(c Expirable_Handle, key Key) (value Value, ok Status) {
	defer func() { Status_Invariants(ok, "expirable_get.ok") }()
	Expirable_Handle_Invariants(c, "expirable_get.cache")
	aver.Always(c.Capacity > CAPACITY_MINIMUM, "Expirable get cache is initialized.")
	expirable_reap_due(c)
	found := list_find(&c.Evict_List, key)
	if !found.Found {
		return value, false
	}
	// Cleanup runs on the harness's cadence, so a read must reject an already-expired entry
	// itself rather than hand back a value whose TTL has passed.
	if time.Clock_Now_Monotonic(time.Clock(c.Clock)) >
		c.Evict_List.Nodes[found.Position].Expires_At {
		return value, false
	}
	list_move_to_front(&c.Evict_List, found.Position)
	return c.Evict_List.Nodes[found.Position].Value, true
}

// Expirable_Contains reports membership without renewing recency and without checking expiry, so
// it may report an expired-but-not-yet-reaped entry (upstream parity).
func Expirable_Contains(c Expirable_Handle, key Key) (ok Status) {
	defer func() { Status_Invariants(ok, "expirable_contains.ok") }()
	Expirable_Handle_Invariants(c, "expirable_contains.cache")
	aver.Always(c.Capacity > CAPACITY_MINIMUM, "Expirable contains cache is initialized.")
	expirable_reap_due(c)
	found := list_find(&c.Evict_List, key)
	ok = found.Found
	return ok
}

// Expirable_Peek returns key's value without renewing recency; an entry past its TTL is rejected
// as a miss, and a genuine miss returns the zero value and false.
func Expirable_Peek(c Expirable_Handle, key Key) (value Value, ok Status) {
	defer func() { Status_Invariants(ok, "expirable_peek.ok") }()
	Expirable_Handle_Invariants(c, "expirable_peek.cache")
	aver.Always(c.Capacity > CAPACITY_MINIMUM, "Expirable peek cache is initialized.")
	expirable_reap_due(c)
	found := list_find(&c.Evict_List, key)
	if !found.Found {
		return value, false
	}
	if time.Clock_Now_Monotonic(time.Clock(c.Clock)) >
		c.Evict_List.Nodes[found.Position].Expires_At {
		return value, false
	}
	return c.Evict_List.Nodes[found.Position].Value, true
}

// Expirable_Remove deletes a present key, reporting whether it was there.
func Expirable_Remove(c Expirable_Handle, key Key) (present Status) {
	defer func() { Status_Invariants(present, "expirable_remove.present") }()
	Expirable_Handle_Invariants(c, "expirable_remove.cache")
	aver.Always(c.Capacity > CAPACITY_MINIMUM, "Expirable remove cache is initialized.")
	expirable_reap_due(c)
	found := list_find(&c.Evict_List, key)
	if found.Found {
		expirable_remove_element(c, found.Position)
		return true
	}
	return false
}

// Expirable_Remove_Oldest deletes and returns the least-recently-used entry; ok is false when
// the cache is empty.
func Expirable_Remove_Oldest(c Expirable_Handle) (key Key, value Value, ok Status) {
	defer func() { Status_Invariants(ok, "expirable_remove_oldest.ok") }()
	Expirable_Handle_Invariants(c, "expirable_remove_oldest.cache")
	aver.Always(
		c.Capacity > CAPACITY_MINIMUM,
		"Expirable remove oldest cache is initialized.",
	)
	expirable_reap_due(c)
	ent := list_back(&c.Evict_List)
	if ent == POSITION_NONE {
		return key, value, false
	}
	key = c.Evict_List.Nodes[ent].Key
	value = c.Evict_List.Nodes[ent].Value
	expirable_remove_element(c, ent)
	return key, value, true
}

// Expirable_Get_Oldest returns the least-recently-used entry without removing it; ok is false
// when the cache is empty.
func Expirable_Get_Oldest(c Expirable_Handle) (key Key, value Value, ok Status) {
	defer func() { Status_Invariants(ok, "expirable_get_oldest.ok") }()
	Expirable_Handle_Invariants(c, "expirable_get_oldest.cache")
	aver.Always(
		c.Capacity > CAPACITY_MINIMUM,
		"Expirable get oldest cache is initialized.",
	)
	expirable_reap_due(c)
	ent := list_back(&c.Evict_List)
	if ent != POSITION_NONE {
		return c.Evict_List.Nodes[ent].Key, c.Evict_List.Nodes[ent].Value, true
	}
	return key, value, false
}

// Expirable_Keys writes the unexpired keys, oldest to newest, into buffer up to len(buffer) and
// returns how many it wrote. Expired-but-unreaped entries are skipped. Caller bounds output.
func Expirable_Keys(c Expirable_Handle, buffer Keys) (count Count) {
	defer func() { Count_Invariants(count, "expirable_keys.count") }()
	Expirable_Handle_Invariants(c, "expirable_keys.cache")
	Keys_Invariants(buffer, "expirable_keys.buffer")
	aver.Always(c.Capacity > CAPACITY_MINIMUM, "Expirable keys cache is initialized.")
	expirable_reap_due(c)
	now := time.Clock_Now_Monotonic(time.Clock(c.Clock))
	for ent := list_back(&c.Evict_List); ent != POSITION_NONE; {
		if count >= Count(buffer.Count) {
			break
		}
		if now > c.Evict_List.Nodes[ent].Expires_At {
			ent = entry_previous(&c.Evict_List, ent)
			continue
		}
		buffer.Storage[count] = c.Evict_List.Nodes[ent].Key
		count++
		ent = entry_previous(&c.Evict_List, ent)
	}
	return count
}

// Expirable_Values writes the unexpired values, oldest to newest, into buffer up to len(buffer)
// and returns how many it wrote; expired-but-unreaped entries are skipped.
func Expirable_Values(c Expirable_Handle, buffer Values) (count Count) {
	defer func() { Count_Invariants(count, "expirable_values.count") }()
	Expirable_Handle_Invariants(c, "expirable_values.cache")
	Values_Invariants(buffer, "expirable_values.buffer")
	aver.Always(c.Capacity > CAPACITY_MINIMUM, "Expirable values cache is initialized.")
	expirable_reap_due(c)
	now := time.Clock_Now_Monotonic(time.Clock(c.Clock))
	for ent := list_back(&c.Evict_List); ent != POSITION_NONE; {
		if count >= Count(buffer.Count) {
			break
		}
		if now > c.Evict_List.Nodes[ent].Expires_At {
			ent = entry_previous(&c.Evict_List, ent)
			continue
		}
		buffer.Storage[count] = c.Evict_List.Nodes[ent].Value
		count++
		ent = entry_previous(&c.Evict_List, ent)
	}
	return count
}

// Expirable_Count returns the number of entries held, including any expired but not yet reaped
// (upstream parity).
func Expirable_Count(c Expirable_Handle) (count Count) {
	defer func() { Count_Invariants(count, "expirable_count.count") }()
	Expirable_Handle_Invariants(c, "expirable_count.cache")
	aver.Always(c.Capacity > CAPACITY_MINIMUM, "Expirable count cache is initialized.")
	expirable_reap_due(c)
	return Count(c.Evict_List.Count)
}

// Expirable_Cap returns the fixed capacity.
func Expirable_Cap(c Expirable_Handle) (capacity Capacity) {
	defer func() { Capacity_Invariants(capacity, "expirable_cap.capacity") }()
	Expirable_Handle_Invariants(c, "expirable_cap.cache")
	aver.Always(c.Capacity > CAPACITY_MINIMUM, "Expirable cap cache is initialized.")
	expirable_reap_due(c)
	return c.Capacity
}

// Expirable_Purge empties the cache and every expiry bucket, calling On_Evict for each entry, and
// returns every node to the pool.
func Expirable_Purge(c Expirable_Handle) {
	Expirable_Handle_Invariants(c, "expirable_purge.cache")
	aver.Always(c.Capacity > CAPACITY_MINIMUM, "Expirable purge cache is initialized.")
	expirable_reap_due(c)
	if c.On_Evict != nil {
		for position := c.Evict_List.Back; position != Back_Position(POSITION_NONE); {
			entry := c.Evict_List.Nodes[position]
			position = Back_Position(entry.Previous)
			c.On_Evict(entry.Key, entry.Value)
		}
	}
	for bucket_index := range c.Buckets {
		c.Buckets[bucket_index].Head = POSITION_NONE
		c.Buckets[bucket_index].Newest_Entry = 0
	}
	list_reset(&c.Evict_List)
}

// Arms one cleanup timeout. Callback cannot capture generic cache without allocation, so it only
// marks cleanup due. Next cache access does generic sweep and re-arms same completion. Cancelled
// timeout carries error and leaves cleanup idle.
func expirable_arm_cleanup(c Expirable_Handle) {
	Expirable_Handle_Invariants(c, "expirable_arm_cleanup.cache")
	nbio.Timeline_Timeout(
		nbio.Timeline(c.Timeline),
		&c.Cleanup.Completion,
		time.Duration(expirable_cleanup_interval(c)),
		expirable_cleanup_callback,
	)
}

// Expirable cleanup callback recovers wrapper because Completion is its first field.
func expirable_cleanup_callback(completion nbio.Completion_Handle) {
	cleanup := (*Cleanup)(unsafe.Pointer(completion))
	if completion.Error != nil {
		return
	}
	cleanup.Due = true
}

// Expirable reap due performs deferred generic cleanup before cache state becomes observable.
func expirable_reap_due(c Expirable_Handle) {
	Expirable_Handle_Invariants(c, "expirable_reap_due.cache")
	if !c.Cleanup.Due {
		return
	}
	c.Cleanup.Due = false
	for bucket_index := 0; bucket_index < BUCKET_COUNT; bucket_index++ {
		previous_bucket := c.Next_Cleanup_Bucket
		expirable_reap_ripe_bucket(c)
		if c.Next_Cleanup_Bucket == previous_bucket {
			break
		}
	}
	expirable_arm_cleanup(c)
}

// Returns the delay between cleanup sweeps: one TTL spread across the ring, so a full lap takes
// about one TTL — the upstream ticker's ttl/numBuckets. Clamped to at least one grain so the
// repeating timeout always advances the clock and never spins within a single tick.
func expirable_cleanup_interval(c Expirable_Handle) (interval Cleanup_Interval) {
	defer func() {
		Cleanup_Interval_Invariants(interval, "expirable_cleanup_interval.interval")
	}()
	Expirable_Handle_Invariants(c, "expirable_cleanup_interval.cache")
	interval = Cleanup_Interval(c.TTL / BUCKET_COUNT)
	if interval < CLEANUP_INTERVAL_MINIMUM {
		interval = CLEANUP_INTERVAL_MINIMUM
	}
	return interval
}

// Reaps the next expiry bucket when it has fully passed, advancing the ring cursor. A bucket is
// reaped only once the clock is past its newest entry — every entry filed there expires no later
// than that — so a partially live bucket is left alone and the cursor does not advance past it.
func expirable_reap_ripe_bucket(c Expirable_Handle) {
	Expirable_Handle_Invariants(c, "expirable_reap_ripe_bucket.cache")
	bucket := &c.Buckets[c.Next_Cleanup_Bucket]
	if time.Clock_Now_Monotonic(time.Clock(c.Clock)) <= bucket.Newest_Entry {
		return
	}
	// Capture each next-in-chain before remove_element unlinks and frees the node.
	ent := bucket.Head
	for ent != POSITION_NONE {
		next := c.Evict_List.Nodes[ent].Bucket_Next
		expirable_remove_element(c, ent)
		ent = next
	}
	bucket.Head = POSITION_NONE
	bucket.Newest_Entry = 0
	c.Next_Cleanup_Bucket = (c.Next_Cleanup_Bucket + 1) % BUCKET_COUNT
}

// Files ent at bucket head one step behind cleanup cursor, so cursor reaches it only after
// wrapping whole ring. This gives about one TTL of grace, matching upstream.
func expirable_add_to_bucket(c Expirable_Handle, position Position) {
	Position_Invariants(position, "expirable_add_to_bucket.position")
	Expirable_Handle_Invariants(c, "expirable_add_to_bucket.cache")
	bucket_identifier := Bucket_Index(
		(BUCKET_COUNT + int(c.Next_Cleanup_Bucket) - 1) % BUCKET_COUNT,
	)
	ent := &c.Evict_List.Nodes[position]
	ent.Expire_Bucket = bucket_identifier
	bucket := &c.Buckets[bucket_identifier]
	ent.Bucket_Previous = POSITION_NONE
	ent.Bucket_Next = bucket.Head
	if bucket.Head != POSITION_NONE {
		c.Evict_List.Nodes[bucket.Head].Bucket_Previous = position
	}
	bucket.Head = position
	if bucket.Newest_Entry < ent.Expires_At {
		bucket.Newest_Entry = ent.Expires_At
	}
}

// Unlinks ent from its bucket chain, so a renewed or removed entry does not linger there. It has
// the node in hand, so no key lookup is needed.
func expirable_remove_from_bucket(c Expirable_Handle, position Position) {
	Position_Invariants(position, "expirable_remove_from_bucket.position")
	Expirable_Handle_Invariants(c, "expirable_remove_from_bucket.cache")
	ent := &c.Evict_List.Nodes[position]
	bucket := &c.Buckets[ent.Expire_Bucket]
	if ent.Bucket_Previous != POSITION_NONE {
		c.Evict_List.Nodes[ent.Bucket_Previous].Bucket_Next = ent.Bucket_Next
	} else {
		bucket.Head = ent.Bucket_Next
	}
	if ent.Bucket_Next != POSITION_NONE {
		c.Evict_List.Nodes[ent.Bucket_Next].Bucket_Previous = ent.Bucket_Previous
	}
	ent.Bucket_Next = POSITION_NONE
	ent.Bucket_Previous = POSITION_NONE
}

// Drops the least-recently-used entry, if any.
func expirable_remove_oldest(c Expirable_Handle) {
	Expirable_Handle_Invariants(c, "expirable_remove_oldest_internal.cache")
	ent := list_back(&c.Evict_List)
	if ent != POSITION_NONE {
		expirable_remove_element(c, ent)
	}
}

// Unlinks ent from the list, its map entry, and its bucket, then fires On_Evict. It captures the
// key and value, and unlinks the bucket chain, before list_remove frees (and zeroes) the node.
func expirable_remove_element(c Expirable_Handle, position Position) {
	Position_Invariants(position, "expirable_remove_element.position")
	Expirable_Handle_Invariants(c, "expirable_remove_element.cache")
	key := c.Evict_List.Nodes[position].Key
	value := c.Evict_List.Nodes[position].Value
	expirable_remove_from_bucket(c, position)
	list_remove(&c.Evict_List, position)
	if c.On_Evict != nil {
		c.On_Evict(key, value)
	}
}
