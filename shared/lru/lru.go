// Package lru is a dependency-injected port of github.com/hashicorp/golang-lru:
// three fixed-size caches — Simple (LRU), Two_Queue (scan-resistant 2Q), and Expirable
// (TTL) — over one shared doubly-linked list.
//
// The upstream leans on ambient state the house linter forbids and deterministic
// simulation cannot replay, all of which this port removes:
//   - Every cache is single-threaded. The upstream guarded each with a sync.Mutex, but a
//     deterministic package holds no lock; the caller serializes access through the io loop,
//     so there is no shared-memory concurrency to guard. This collapses the upstream's
//     lock-free base (simplelru.LRU) and thread-safe wrapper (lru.Cache) into one Simple —
//     the wrapper existed only to add a lock and buffer evictions past it.
//   - The LRUCache interface (a user interface) is gone; Two_Queue holds concrete *Simple
//     sub-caches — the house prefers one implementation to a substitution seam.
//   - Expirable read time.Now() directly; here it reads an injected time.Clock's
//     Now_Monotonic, so expiry replays bit-for-bit under a virtual clock in tests.
//   - Expirable drove cleanup from a self-owned time.Ticker goroutine; a deterministic package
//     starts none, so it arms a repeating io.Timeout the harness's io loop fires instead.
//   - Constructors returned an error on a non-positive size; that is a programmer bug, so they
//     assert instead — and the upstream size<=0 "unlimited" and ttl<=0 "no expiry" modes are gone.
//   - 2Q's recent/ghost split was float64, which a deterministic package bans; here it is a
//     fixedpoint.Ratio, the house stand-in for float64 (prng.Ratio is a probability, not this).
//   - Nothing is unbounded. Capacity is fixed at construction and pre-allocated: each cache owns
//     a pool of exactly Capacity entry nodes over which its list is threaded, so Add/Get/evict
//     allocate nothing in steady state. There is no Resize — capacity is a hard, up-front
//     commitment — and Keys/Values fill a caller-supplied buffer instead of allocating, so the
//     caller bounds its reads too.
//
// The linter also bans methods that do not satisfy a stdlib interface, so every cache
// operation is a free function prefixed by its type — Simple_Get(cache, key), not
// cache.Get(key) — mirroring shared/uuid's UUID_Version(u).
package lru

import (
	"local/james-orcales/shared/io"
	"local/james-orcales/shared/math/fixedpoint"
	"local/james-orcales/shared/time"
)

// Evict_Callback runs when a cache discards an entry — through eviction, removal, or purge.
// One type serves every cache here; the upstream declared an identical callback twice
// (simplelru and expirable), which the flat package collapses into this.
type Evict_Callback[K comparable, V any] func(key K, value V)

// Entry is one node of the intrusive lists shared by every cache in this package. Its storage is
// owned by a List's pre-allocated pool, not the heap, so an Add reuses a node rather than
// allocating one. Every node lives in the recency ring (Next/Previous); an Expirable node lives
// simultaneously in its expiry bucket chain (Bucket_Next/Bucket_Previous) over the same node.
type Entry[K comparable, V any] struct {
	// Next is the newer neighbor or the front Root, or the next free node while free.
	Next *Entry[K, V]
	// Previous points to the less-recently-used neighbor, or the sentinel Root at the back.
	Previous *Entry[K, V]
	// List owns this entry; a move on an entry from another List is rejected. Nil while free.
	List *List[K, V]
	// Bucket_Next chains toward the next entry in this Expirable expiry bucket; nil otherwise.
	Bucket_Next *Entry[K, V]
	// Bucket_Previous chains toward the previous entry in the expiry bucket; nil otherwise.
	Bucket_Previous *Entry[K, V]
	// Key is the lookup key.
	Key K
	// Value is the stored value.
	Value V
	// Expires_At is the Moment past which Expirable treats this entry as gone; zero for the
	// non-expiring caches.
	Expires_At time.Moment
	// Expire_Bucket is which of Expirable's ring buckets holds this entry, so a renew or a
	// remove finds it in O(1); zero for the non-expiring caches.
	Expire_Bucket uint8
}

// List is a fixed-capacity doubly-linked list: a house-style port of container/list (see
// LICENSE.bsd.golang) that owns its nodes. Build it with new_list, which allocates the whole
// Nodes pool once and threads every node onto Free; there is no zero-value path. The active
// list closes through Root — the last element's Next is &Root and the first's Previous is
// &Root — so insert and remove need no nil checks. Nodes come from Free (list_allocate) and
// return to it (list_free), so the list never touches the heap after construction.
type List[K comparable, V any] struct {
	// Root is the sentinel of the active ring, not a real element; only &Root, Root.Next, and
	// Root.Previous are used. Back is Root.Previous and Front is Root.Next.
	Root Entry[K, V]
	// Nodes is the pre-allocated pool — exactly capacity entries, allocated once. Every active
	// and free node is one of these; the list allocates no node after construction.
	Nodes []Entry[K, V]
	// Free heads the free-list threaded through the nodes' Next pointers.
	Free *Entry[K, V]
	// Count is the number of active elements, excluding the sentinel and the free-list.
	Count int
}

// Builds a fixed-capacity List: allocates the whole node pool once and readies the free-list and
// the empty sentinel ring.
func new_list[K comparable, V any](capacity_count int) (l *List[K, V]) {
	l = &List[K, V]{Nodes: make([]Entry[K, V], capacity_count)}
	list_reset(l)
	return l
}

// Empties l: zeroes every node, re-threads them all onto Free, and closes the sentinel ring. It
// runs at construction and on Purge, so a purge reclaims every node without touching the heap.
func list_reset[K comparable, V any](l *List[K, V]) {
	var zero_key K
	var zero_value V
	l.Free = nil
	for i_index := range l.Nodes {
		node := &l.Nodes[i_index]
		node.Key = zero_key
		node.Value = zero_value
		node.Expires_At = 0
		node.Expire_Bucket = 0
		node.List = nil
		node.Previous = nil
		node.Bucket_Next = nil
		node.Bucket_Previous = nil
		node.Next = l.Free
		l.Free = node
	}
	l.Root.Next = &l.Root
	l.Root.Previous = &l.Root
	l.Count = 0
}

// Pops a node from the free-list. It asserts the pool is not empty: every cache evicts before it
// inserts, so peak node use equals capacity and the pool cannot underflow — a nil here is a bug.
func list_allocate[K comparable, V any](l *List[K, V]) (node *Entry[K, V]) {
	node = l.Free
	assert(node != nil, "lru: list pool underflow")
	l.Free = node.Next
	return node
}

// Returns e to the free-list, zeroing its key and value first so a pooled node pins neither the
// key nor the value it used to hold — the references are released for GC even though the node is
// reused.
func list_free[K comparable, V any](l *List[K, V], e *Entry[K, V]) {
	var zero_key K
	var zero_value V
	e.Key = zero_key
	e.Value = zero_value
	e.Expires_At = 0
	e.Expire_Bucket = 0
	e.List = nil
	e.Previous = nil
	e.Bucket_Next = nil
	e.Bucket_Previous = nil
	e.Next = l.Free
	l.Free = e
}

// Returns the least-recently-used element (the eviction target) or nil when empty.
func list_back[K comparable, V any](l *List[K, V]) (back *Entry[K, V]) {
	if l.Count == 0 {
		return nil
	}
	return l.Root.Previous
}

// Links node just after the sentinel Root — the most-recently-used front — and adopts it.
func list_splice_front[K comparable, V any](l *List[K, V], node *Entry[K, V]) {
	at := &l.Root
	node.Previous = at
	node.Next = at.Next
	node.Previous.Next = node
	node.Next.Previous = node
	node.List = l
	l.Count++
}

// Unlinks e from the active ring and returns it to the pool. Callers capture e's key and value
// first, since list_free zeroes them.
func list_remove[K comparable, V any](l *List[K, V], e *Entry[K, V]) {
	e.Previous.Next = e.Next
	e.Next.Previous = e.Previous
	l.Count--
	list_free(l, e)
}

// Takes a node from the pool for a non-expiring (k, v) and splices it at the front.
func list_push_front[K comparable, V any](l *List[K, V], k K, v V) (pushed *Entry[K, V]) {
	node := list_allocate(l)
	node.Key = k
	node.Value = v
	list_splice_front(l, node)
	return node
}

// Takes a node from the pool for (k, v) carrying its expiry Moment and splices it at the front.
func list_push_front_expirable[K comparable, V any](
	l *List[K, V], k K, v V, expires_at time.Moment,
) (pushed *Entry[K, V]) {
	node := list_allocate(l)
	node.Key = k
	node.Value = v
	node.Expires_At = expires_at
	list_splice_front(l, node)
	return node
}

// Promotes e to the most-recently-used front. It ignores an entry owned by another list or one
// already at the front, so a stray call cannot corrupt l.
func list_move_to_front[K comparable, V any](l *List[K, V], e *Entry[K, V]) {
	if e.List != l {
		return
	}
	if l.Root.Next == e {
		return
	}
	// Splice e out of its current position, then in just after Root.
	e.Previous.Next = e.Next
	e.Next.Previous = e.Previous
	at := &l.Root
	e.Previous = at
	e.Next = at.Next
	e.Previous.Next = e
	e.Next.Previous = e
}

// Returns the entry one step toward the older end, or nil at the sentinel so a caller can walk
// from list_back to the front without seeing Root.
func entry_previous[K comparable, V any](e *Entry[K, V]) (previous *Entry[K, V]) {
	if e.List == nil {
		return nil
	}
	if e.Previous == &e.List.Root {
		return nil
	}
	return e.Previous
}

// Simple is a fixed-capacity LRU cache. It is single-threaded like every cache here: the upstream
// split a lock-free base (simplelru.LRU) from a thread-safe wrapper (lru.Cache), but a
// deterministic package holds no lock, so the two collapse into this one type and the caller
// serializes access through the io loop. Its Capacity is pre-allocated and immutable.
type Simple[K comparable, V any] struct {
	// Capacity is the fixed maximum, set once at construction and never changed; a full cache
	// evicts the oldest before an insert.
	Capacity int
	// Evict_List orders entries by recency, most-recently-used at the front, over its own pool.
	Evict_List *List[K, V]
	// Items indexes entries by key for O(1) lookup; pre-sized to Capacity, so it never grows.
	Items map[K]*Entry[K, V]
	// On_Evict, when non-nil, runs for every entry the cache discards.
	On_Evict Evict_Callback[K, V]
}

// New_Simple builds a Simple of the given capacity; on_evict (may be nil) runs for each discarded
// entry. Capacity must be positive — a non-positive capacity is a programmer error, not a runtime
// condition, so it panics rather than returning one. The whole node pool and index map are
// allocated here, so no operation afterward touches the heap.
func New_Simple[K comparable, V any](
	capacity_count int, on_evict Evict_Callback[K, V],
) (c *Simple[K, V]) {
	assert(capacity_count > 0, "lru: Simple capacity must be positive")
	return &Simple[K, V]{
		Capacity:   capacity_count,
		Evict_List: new_list[K, V](capacity_count),
		Items:      make(map[K]*Entry[K, V], capacity_count),
		On_Evict:   on_evict,
	}
}

// Simple_Add inserts or overwrites key, returning whether the insert forced an eviction. An
// overwrite renews recency without evicting. A new key into a full cache evicts the oldest first,
// so peak node use never exceeds Capacity and the pool holds.
func Simple_Add[K comparable, V any](c *Simple[K, V], key K, value V) (evicted bool) {
	if ent, found := c.Items[key]; found {
		list_move_to_front(c.Evict_List, ent)
		ent.Value = value
		return false
	}
	evicted = c.Evict_List.Count >= c.Capacity
	if evicted {
		simple_remove_oldest(c)
	}
	ent := list_push_front(c.Evict_List, key, value)
	c.Items[key] = ent
	return evicted
}

// Simple_Get returns key's value and renews its recency; a miss returns the zero value and
// false.
func Simple_Get[K comparable, V any](c *Simple[K, V], key K) (value V, ok bool) {
	ent, found := c.Items[key]
	if found {
		list_move_to_front(c.Evict_List, ent)
		return ent.Value, true
	}
	return value, false
}

// Simple_Contains reports membership without renewing recency.
func Simple_Contains[K comparable, V any](c *Simple[K, V], key K) (ok bool) {
	_, ok = c.Items[key]
	return ok
}

// Simple_Peek returns key's value without renewing recency; a miss returns the zero value and
// false.
func Simple_Peek[K comparable, V any](c *Simple[K, V], key K) (value V, ok bool) {
	ent, found := c.Items[key]
	if found {
		return ent.Value, true
	}
	return value, false
}

// Simple_Contains_Or_Add inserts key only when absent, reporting whether it was already present
// and whether the insert evicted. A present key is left untouched, its recency unchanged.
func Simple_Contains_Or_Add[K comparable, V any](
	c *Simple[K, V], key K, value V,
) (present bool, evicted bool) {
	if Simple_Contains(c, key) {
		return true, false
	}
	evicted = Simple_Add(c, key, value)
	return false, evicted
}

// Simple_Peek_Or_Add inserts key only when absent, returning the existing value when present
// and reporting whether it was present and whether the insert evicted. A present key's recency
// is left unchanged.
func Simple_Peek_Or_Add[K comparable, V any](
	c *Simple[K, V], key K, value V,
) (previous V, present bool, evicted bool) {
	previous, present = Simple_Peek(c, key)
	if present {
		return previous, true, false
	}
	evicted = Simple_Add(c, key, value)
	return previous, false, evicted
}

// Simple_Remove deletes a present key, reporting whether it was there.
func Simple_Remove[K comparable, V any](c *Simple[K, V], key K) (present bool) {
	ent, found := c.Items[key]
	if found {
		simple_remove_element(c, ent)
		return true
	}
	return false
}

// Simple_Remove_Oldest deletes and returns the least-recently-used entry; ok is false when the
// cache is empty.
func Simple_Remove_Oldest[K comparable, V any](c *Simple[K, V]) (key K, value V, ok bool) {
	ent := list_back(c.Evict_List)
	if ent == nil {
		return key, value, false
	}
	key = ent.Key
	value = ent.Value
	simple_remove_element(c, ent)
	return key, value, true
}

// Simple_Get_Oldest returns the least-recently-used entry without removing it; ok is false when
// the cache is empty.
func Simple_Get_Oldest[K comparable, V any](c *Simple[K, V]) (key K, value V, ok bool) {
	ent := list_back(c.Evict_List)
	if ent != nil {
		return ent.Key, ent.Value, true
	}
	return key, value, false
}

// Simple_Keys writes the keys, oldest to newest, into buffer up to len(buffer) and returns how
// many it wrote. The caller sizes buffer — pass one of length Simple_Cap to receive them all —
// so the read allocates nothing and is bounded by the caller.
func Simple_Keys[K comparable, V any](c *Simple[K, V], buffer []K) (count int) {
	for ent := list_back(c.Evict_List); ent != nil; ent = entry_previous(ent) {
		if count >= len(buffer) {
			break
		}
		buffer[count] = ent.Key
		count++
	}
	return count
}

// Simple_Values writes the values, oldest to newest, into buffer up to len(buffer) and returns
// how many it wrote. Like Simple_Keys, the caller sizes and thus bounds the read.
func Simple_Values[K comparable, V any](c *Simple[K, V], buffer []V) (count int) {
	for ent := list_back(c.Evict_List); ent != nil; ent = entry_previous(ent) {
		if count >= len(buffer) {
			break
		}
		buffer[count] = ent.Value
		count++
	}
	return count
}

// Simple_Count returns the number of entries held.
func Simple_Count[K comparable, V any](c *Simple[K, V]) (count int) {
	return c.Evict_List.Count
}

// Simple_Cap returns the fixed capacity.
func Simple_Cap[K comparable, V any](c *Simple[K, V]) (capacity int) {
	return c.Capacity
}

// Simple_Purge empties the cache, calling On_Evict for every discarded entry, and returns every
// node to the pool.
func Simple_Purge[K comparable, V any](c *Simple[K, V]) {
	for key, ent := range c.Items {
		if c.On_Evict != nil {
			c.On_Evict(key, ent.Value)
		}
		delete(c.Items, key)
	}
	list_reset(c.Evict_List)
}

// Drops the least-recently-used entry, if any.
func simple_remove_oldest[K comparable, V any](c *Simple[K, V]) {
	ent := list_back(c.Evict_List)
	if ent != nil {
		simple_remove_element(c, ent)
	}
}

// Unlinks ent, deletes its Items entry, and fires On_Evict. It captures the key and value before
// list_remove frees (and zeroes) the node.
func simple_remove_element[K comparable, V any](c *Simple[K, V], ent *Entry[K, V]) {
	key := ent.Key
	value := ent.Value
	delete(c.Items, key)
	list_remove(c.Evict_List, ent)
	if c.On_Evict != nil {
		c.On_Evict(key, value)
	}
}

// DEFAULT_RECENT_RATIO is the share of a Two_Queue held for entries seen once — the upstream
// Default2QRecentRatio of 0.25, as a fixed-point ratio (SCALE/4 is one-quarter of one whole).
const DEFAULT_RECENT_RATIO fixedpoint.Ratio = fixedpoint.SCALE / 4

// DEFAULT_GHOST_RATIO is the share of a Two_Queue's ghost list — the upstream Default2QGhostEntries
// of 0.50, as a fixed-point ratio (SCALE/2 is one-half of one whole).
const DEFAULT_GHOST_RATIO fixedpoint.Ratio = fixedpoint.SCALE / 2

// Two_Queue is a scan-resistant 2Q cache (upstream lru.TwoQueueCache). It splits its capacity
// across a recent list (entries seen once) and a frequent list (entries seen again), plus a
// value-less ghost list tracking keys recently evicted from recent — so a burst of one-off keys
// churns only the recent list and cannot flush the frequently used entries. The upstream held
// its three sub-caches behind the LRUCache interface; here they are concrete *Simple, each with
// its own pre-allocated pool. Capacity is fixed at construction.
type Two_Queue[K comparable, V any] struct {
	// Capacity is the fixed total across the recent and frequent lists.
	Capacity int
	// Recent_Size caps the recent list; once it is reached, ensure-space spills the oldest
	// recent entry to the ghost list rather than dropping a frequent one.
	Recent_Size int
	// Recent_Ratio is the fixed-point share of Capacity the recent list may hold.
	Recent_Ratio fixedpoint.Ratio
	// Ghost_Ratio is the fixed-point share of Capacity the ghost list tracks.
	Ghost_Ratio fixedpoint.Ratio
	// Recent holds entries seen once; a second access promotes them to Frequent.
	Recent *Simple[K, V]
	// Frequent holds entries seen more than once.
	Frequent *Simple[K, V]
	// Recent_Evict is the ghost list: keys recently evicted from Recent, tracked value-less
	// so a re-add can jump straight to Frequent.
	Recent_Evict *Simple[K, struct{}]
}

// Two_Queue_Input configures New_Two_Queue. It is not generic: neither the capacity nor the ratios
// carry a key or value type.
type Two_Queue_Input struct {
	// Capacity is the fixed total across the recent and frequent lists.
	Capacity int
	// Recent_Ratio is the fixed-point share of Capacity the recent list may hold; a zero value
	// defaults to DEFAULT_RECENT_RATIO.
	Recent_Ratio fixedpoint.Ratio
	// Ghost_Ratio is the fixed-point share of Capacity the ghost list tracks; a zero value
	// defaults to DEFAULT_GHOST_RATIO.
	Ghost_Ratio fixedpoint.Ratio
}

// New_Two_Queue builds a Two_Queue from input, defaulting unset ratios and pre-allocating all
// three sub-cache pools. Capacity must be positive and each ratio must lie in [0, 1]; the ghost
// list must come out non-empty, so a capacity too small for the ghost ratio panics — all
// programmer errors, asserted rather than returned.
func New_Two_Queue[K comparable, V any](input Two_Queue_Input) (c *Two_Queue[K, V]) {
	recent_ratio := input.Recent_Ratio
	if recent_ratio == 0 {
		recent_ratio = DEFAULT_RECENT_RATIO
	}
	ghost_ratio := input.Ghost_Ratio
	if ghost_ratio == 0 {
		ghost_ratio = DEFAULT_GHOST_RATIO
	}
	assert(input.Capacity > 0, "lru: Two_Queue capacity must be positive")
	assert(recent_ratio >= 0, "lru: Two_Queue recent ratio is negative")
	assert(recent_ratio <= fixedpoint.SCALE, "lru: Two_Queue recent ratio exceeds one")
	assert(ghost_ratio >= 0, "lru: Two_Queue ghost ratio is negative")
	assert(ghost_ratio <= fixedpoint.SCALE, "lru: Two_Queue ghost ratio exceeds one")
	recent_size := ratio_of(input.Capacity, recent_ratio)
	evict_size := ratio_of(input.Capacity, ghost_ratio)
	assert(evict_size > 0, "lru: Two_Queue capacity too small for its ghost ratio")
	return &Two_Queue[K, V]{
		Capacity:     input.Capacity,
		Recent_Size:  recent_size,
		Recent_Ratio: recent_ratio,
		Ghost_Ratio:  ghost_ratio,
		Recent:       New_Simple[K, V](input.Capacity, nil),
		Frequent:     New_Simple[K, V](input.Capacity, nil),
		Recent_Evict: New_Simple[K, struct{}](evict_size, nil),
	}
}

// Two_Queue_Get returns key's value, promoting a recent hit to the frequent list; a miss returns
// the zero value and false. A frequent hit renews recency in place.
func Two_Queue_Get[K comparable, V any](c *Two_Queue[K, V], key K) (value V, ok bool) {
	value, ok = Simple_Get(c.Frequent, key)
	if ok {
		return value, ok
	}
	// A recent hit is the second access, so promote it into the frequent list.
	value, ok = Simple_Peek(c.Recent, key)
	if ok {
		Simple_Remove(c.Recent, key)
		Simple_Add(c.Frequent, key, value)
		return value, ok
	}
	return value, false
}

// Two_Queue_Add inserts or refreshes key. A frequent key is updated in place, a recent key is
// promoted to frequent, a ghost key returns as frequent, and a brand-new key lands in recent.
func Two_Queue_Add[K comparable, V any](c *Two_Queue[K, V], key K, value V) {
	if Simple_Contains(c.Frequent, key) {
		Simple_Add(c.Frequent, key, value)
		return
	}
	if Simple_Contains(c.Recent, key) {
		Simple_Remove(c.Recent, key)
		Simple_Add(c.Frequent, key, value)
		return
	}
	// A ghost hit means the key was frequent-worthy before being evicted, so it returns as
	// frequent; ensure space treating this as a ghost-driven add.
	if Simple_Contains(c.Recent_Evict, key) {
		two_queue_ensure_space(c, true)
		Simple_Remove(c.Recent_Evict, key)
		Simple_Add(c.Frequent, key, value)
		return
	}
	two_queue_ensure_space(c, false)
	Simple_Add(c.Recent, key, value)
}

// Two_Queue_Contains reports membership across the frequent and recent lists (not the ghost list)
// without promoting the key.
func Two_Queue_Contains[K comparable, V any](c *Two_Queue[K, V], key K) (ok bool) {
	if Simple_Contains(c.Frequent, key) {
		return true
	}
	return Simple_Contains(c.Recent, key)
}

// Two_Queue_Peek returns key's value across the frequent and recent lists without promoting it; a
// miss returns the zero value and false.
func Two_Queue_Peek[K comparable, V any](c *Two_Queue[K, V], key K) (value V, ok bool) {
	value, ok = Simple_Peek(c.Frequent, key)
	if ok {
		return value, ok
	}
	return Simple_Peek(c.Recent, key)
}

// Two_Queue_Remove deletes key from whichever of the frequent, recent, or ghost lists holds it.
func Two_Queue_Remove[K comparable, V any](c *Two_Queue[K, V], key K) {
	if Simple_Remove(c.Frequent, key) {
		return
	}
	if Simple_Remove(c.Recent, key) {
		return
	}
	Simple_Remove(c.Recent_Evict, key)
}

// Two_Queue_Keys writes the keys into buffer — frequent entries first, then recent, each oldest
// to newest — up to len(buffer), and returns how many it wrote. Zero-alloc and caller-bounded.
func Two_Queue_Keys[K comparable, V any](c *Two_Queue[K, V], buffer []K) (count int) {
	count = Simple_Keys(c.Frequent, buffer)
	count += Simple_Keys(c.Recent, buffer[count:])
	return count
}

// Two_Queue_Values writes the values into buffer — frequent first, then recent, each oldest to
// newest — up to len(buffer), and returns how many it wrote.
func Two_Queue_Values[K comparable, V any](c *Two_Queue[K, V], buffer []V) (count int) {
	count = Simple_Values(c.Frequent, buffer)
	count += Simple_Values(c.Recent, buffer[count:])
	return count
}

// Two_Queue_Count returns the number of live entries (frequent plus recent, excluding ghosts).
func Two_Queue_Count[K comparable, V any](c *Two_Queue[K, V]) (count int) {
	return Simple_Count(c.Frequent) + Simple_Count(c.Recent)
}

// Two_Queue_Cap returns the fixed total capacity.
func Two_Queue_Cap[K comparable, V any](c *Two_Queue[K, V]) (capacity int) {
	return c.Capacity
}

// Two_Queue_Purge empties the frequent, recent, and ghost lists.
func Two_Queue_Purge[K comparable, V any](c *Two_Queue[K, V]) {
	Simple_Purge(c.Recent)
	Simple_Purge(c.Frequent)
	Simple_Purge(c.Recent_Evict)
}

// Makes room for one insert. When the recent and frequent lists together fill the cache, it
// evicts the oldest recent entry to the ghost list if recent is at or over its target, otherwise
// it drops the oldest frequent entry. A ghost-driven add (recent_evict) defers to frequent at the
// exact target, so a promotion from the ghost list does not immediately re-evict recent.
func two_queue_ensure_space[K comparable, V any](c *Two_Queue[K, V], recent_evict bool) {
	recent_count := Simple_Count(c.Recent)
	frequent_count := Simple_Count(c.Frequent)
	if recent_count+frequent_count < c.Capacity {
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
			key, _, _ := Simple_Remove_Oldest(c.Recent)
			Simple_Add(c.Recent_Evict, key, struct{}{})
			return
		}
	}
	Simple_Remove_Oldest(c.Frequent)
}

// Scales size by a fixed-point ratio, truncating toward zero — the deterministic replacement for
// the upstream int(float64(size) * ratio). fixedpoint is the house stand-in for float64; Apply
// scales a Number by a dimensionless Ratio.
func ratio_of(size int, ratio fixedpoint.Ratio) (scaled int) {
	product := fixedpoint.Apply(fixedpoint.From_Integer(int64(size)), ratio)
	return int(fixedpoint.Whole(product))
}

// BUCKET_COUNT is the size of Expirable's expiry ring. It must fit a uint8, since both
// Next_Cleanup_Bucket and Entry.Expire_Bucket are uint8, and it matches the upstream numBuckets.
const BUCKET_COUNT = 100

// Expirable_Bucket groups entries whose TTLs land in the same slice of the expiry ring, so the
// cleanup sweep reaps a whole cohort at once instead of scanning every entry. The entries chain
// intrusively through their nodes (Bucket_Next/Bucket_Previous), so the bucket allocates nothing.
type Expirable_Bucket[K comparable, V any] struct {
	// Head is the first entry in the bucket's chain, or nil when the bucket is empty.
	Head *Entry[K, V]
	// Newest_Entry is the latest Expires_At among the chain, so cleanup can tell when the whole
	// bucket has passed and is safe to reap in one pass.
	Newest_Entry time.Moment
}

// Expirable is a fixed-capacity LRU whose entries also expire after a TTL (upstream expirable.LRU).
// The upstream read time.Now() and swept expired entries from a self-owned time.Ticker
// goroutine; a deterministic package can do neither. Instead Expirable reads an injected
// time.Clock and arms a repeating io.Timeout on an injected io.IO — the sweep rides the io
// timeline the harness drives, so the library owns no goroutine and needs no Close or Restart
// (cancel the timeout via io.Cancel to stop it). Reads still reject expired entries lazily, so a
// stale entry is never served even before the timer reaps it. Always used by pointer, because it
// owns Cleanup_Completion, which the io loop tracks by address.
type Expirable[K comparable, V any] struct {
	// Capacity is the fixed maximum; adding past it evicts the least-recently-used entry.
	Capacity int
	// TTL is how long past insertion an entry stays live.
	TTL time.Duration
	// Clock is the injected time source; expiry is measured against Now_Monotonic so it never
	// reads a wall clock and replays deterministically under a virtual clock.
	Clock time.Clock
	// IO is the injected timer the cleanup sweep rides. The library submits an io.Timeout and
	// only the harness drives the loop that fires it, so the library owns no reaper.
	IO *io.IO
	// Cleanup_Completion is the caller-owned storage for the repeating cleanup timeout. Its own
	// callback re-arms it — the deterministic port of the upstream ticker goroutine.
	Cleanup_Completion io.Completion
	// Evict_List orders entries by recency, most-recently-used at the front, over its own pool.
	Evict_List *List[K, V]
	// Items indexes entries by key for O(1) lookup; pre-sized to Capacity, so it never grows.
	Items map[K]*Entry[K, V]
	// On_Evict, when non-nil, runs for every entry the cache discards.
	On_Evict Evict_Callback[K, V]
	// Buckets is the expiry ring; an entry is filed by insertion time so cleanup reaps cohorts.
	Buckets []Expirable_Bucket[K, V]
	// Next_Cleanup_Bucket is the ring position the cleanup sweep will consider next.
	Next_Cleanup_Bucket uint8
}

// Expirable_Input configures New_Expirable.
type Expirable_Input[K comparable, V any] struct {
	// Capacity is the fixed maximum; adding past it evicts the least-recently-used entry.
	Capacity int
	// TTL is how long past insertion an entry stays live.
	TTL time.Duration
	// Clock is the injected time source expiry is measured against; required.
	Clock time.Clock
	// IO is the injected timer the cleanup sweep rides; required.
	IO *io.IO
	// On_Evict, when non-nil, runs for every entry the cache discards.
	On_Evict Evict_Callback[K, V]
}

// New_Expirable builds an Expirable from input, pre-allocating the node pool, and arms its cleanup
// timer. Capacity and TTL must be positive and a Clock and an IO are required — all programmer
// errors, asserted rather than returned. The upstream size<=0 "unlimited" and ttl<=0 "no expiry"
// modes are gone: a no-expiry cache is just a Simple.
func New_Expirable[K comparable, V any](input Expirable_Input[K, V]) (c *Expirable[K, V]) {
	assert(input.Capacity > 0, "lru: Expirable capacity must be positive")
	assert(input.TTL > 0, "lru: Expirable ttl must be positive")
	assert(input.Clock.Now_Monotonic != nil, "lru: Expirable clock is required")
	assert(input.IO != nil, "lru: Expirable io is required")
	assert(input.IO.Timeout != nil, "lru: Expirable io timeout is required")
	c = &Expirable[K, V]{
		Capacity:   input.Capacity,
		TTL:        input.TTL,
		Clock:      input.Clock,
		IO:         input.IO,
		Evict_List: new_list[K, V](input.Capacity),
		Items:      make(map[K]*Entry[K, V], input.Capacity),
		On_Evict:   input.On_Evict,
		Buckets:    make([]Expirable_Bucket[K, V], BUCKET_COUNT),
	}
	expirable_arm_cleanup(c)
	return c
}

// Expirable_Add inserts or overwrites key, stamping its expiry TTL from now, and returns whether
// the insert forced a size eviction. An overwrite renews both recency and expiry. A new key into
// a full cache evicts the oldest first, so peak node use never exceeds Capacity.
func Expirable_Add[K comparable, V any](c *Expirable[K, V], key K, value V) (evicted bool) {
	expires_at := c.Clock.Now_Monotonic() + time.Moment(c.TTL)
	if ent, found := c.Items[key]; found {
		list_move_to_front(c.Evict_List, ent)
		// The renewed expiry belongs in a different bucket, so refile it.
		expirable_remove_from_bucket(c, ent)
		ent.Value = value
		ent.Expires_At = expires_at
		expirable_add_to_bucket(c, ent)
		return false
	}
	evicted = c.Evict_List.Count >= c.Capacity
	if evicted {
		expirable_remove_oldest(c)
	}
	ent := list_push_front_expirable(c.Evict_List, key, value, expires_at)
	c.Items[key] = ent
	expirable_add_to_bucket(c, ent)
	return evicted
}

// Expirable_Get returns key's value and renews its recency; an entry past its TTL is rejected
// as a miss, and a genuine miss returns the zero value and false.
func Expirable_Get[K comparable, V any](c *Expirable[K, V], key K) (value V, ok bool) {
	ent, found := c.Items[key]
	if !found {
		return value, false
	}
	// Cleanup runs on the harness's cadence, so a read must reject an already-expired entry
	// itself rather than hand back a value whose TTL has passed.
	if c.Clock.Now_Monotonic() > ent.Expires_At {
		return value, false
	}
	list_move_to_front(c.Evict_List, ent)
	return ent.Value, true
}

// Expirable_Contains reports membership without renewing recency and without checking expiry, so
// it may report an expired-but-not-yet-reaped entry (upstream parity).
func Expirable_Contains[K comparable, V any](c *Expirable[K, V], key K) (ok bool) {
	_, ok = c.Items[key]
	return ok
}

// Expirable_Peek returns key's value without renewing recency; an entry past its TTL is rejected
// as a miss, and a genuine miss returns the zero value and false.
func Expirable_Peek[K comparable, V any](c *Expirable[K, V], key K) (value V, ok bool) {
	ent, found := c.Items[key]
	if !found {
		return value, false
	}
	if c.Clock.Now_Monotonic() > ent.Expires_At {
		return value, false
	}
	return ent.Value, true
}

// Expirable_Remove deletes a present key, reporting whether it was there.
func Expirable_Remove[K comparable, V any](c *Expirable[K, V], key K) (present bool) {
	ent, found := c.Items[key]
	if found {
		expirable_remove_element(c, ent)
		return true
	}
	return false
}

// Expirable_Remove_Oldest deletes and returns the least-recently-used entry; ok is false when
// the cache is empty.
func Expirable_Remove_Oldest[K comparable, V any](c *Expirable[K, V]) (key K, value V, ok bool) {
	ent := list_back(c.Evict_List)
	if ent == nil {
		return key, value, false
	}
	key = ent.Key
	value = ent.Value
	expirable_remove_element(c, ent)
	return key, value, true
}

// Expirable_Get_Oldest returns the least-recently-used entry without removing it; ok is false
// when the cache is empty.
func Expirable_Get_Oldest[K comparable, V any](c *Expirable[K, V]) (key K, value V, ok bool) {
	ent := list_back(c.Evict_List)
	if ent != nil {
		return ent.Key, ent.Value, true
	}
	return key, value, false
}

// Expirable_Keys writes the unexpired keys, oldest to newest, into buffer up to len(buffer) and
// returns how many it wrote; expired-but-unreaped entries are skipped. Zero-alloc, caller-bounded.
func Expirable_Keys[K comparable, V any](c *Expirable[K, V], buffer []K) (count int) {
	now := c.Clock.Now_Monotonic()
	for ent := list_back(c.Evict_List); ent != nil; ent = entry_previous(ent) {
		if count >= len(buffer) {
			break
		}
		if now > ent.Expires_At {
			continue
		}
		buffer[count] = ent.Key
		count++
	}
	return count
}

// Expirable_Values writes the unexpired values, oldest to newest, into buffer up to len(buffer)
// and returns how many it wrote; expired-but-unreaped entries are skipped.
func Expirable_Values[K comparable, V any](c *Expirable[K, V], buffer []V) (count int) {
	now := c.Clock.Now_Monotonic()
	for ent := list_back(c.Evict_List); ent != nil; ent = entry_previous(ent) {
		if count >= len(buffer) {
			break
		}
		if now > ent.Expires_At {
			continue
		}
		buffer[count] = ent.Value
		count++
	}
	return count
}

// Expirable_Count returns the number of entries held, including any expired but not yet reaped
// (upstream parity).
func Expirable_Count[K comparable, V any](c *Expirable[K, V]) (count int) {
	return c.Evict_List.Count
}

// Expirable_Cap returns the fixed capacity.
func Expirable_Cap[K comparable, V any](c *Expirable[K, V]) (capacity int) {
	return c.Capacity
}

// Expirable_Purge empties the cache and every expiry bucket, calling On_Evict for each entry, and
// returns every node to the pool.
func Expirable_Purge[K comparable, V any](c *Expirable[K, V]) {
	for key, ent := range c.Items {
		if c.On_Evict != nil {
			c.On_Evict(key, ent.Value)
		}
		delete(c.Items, key)
	}
	for bucket_index := range c.Buckets {
		c.Buckets[bucket_index].Head = nil
		c.Buckets[bucket_index].Newest_Entry = 0
	}
	list_reset(c.Evict_List)
}

// Arms the repeating cleanup timeout — the deterministic port of the upstream time.Ticker
// goroutine. Each firing reaps one ripe bucket and re-arms the same completion (the io loop
// returns it to idle before the callback runs, so a callback may resubmit it), so the sweep
// advances on the io timeline the harness drives. A cancelled timeout — io.Cancel on
// Cleanup_Completion — delivers a non-nil error and stops the timer instead of re-arming.
func expirable_arm_cleanup[K comparable, V any](c *Expirable[K, V]) {
	var sweep io.Timeout_Callback
	sweep = func(completion *io.Completion, err error) {
		if err != nil {
			return
		}
		expirable_reap_ripe_bucket(c)
		c.IO.Timeout(completion, sweep, expirable_cleanup_interval(c))
	}
	c.IO.Timeout(&c.Cleanup_Completion, sweep, expirable_cleanup_interval(c))
}

// Returns the delay between cleanup sweeps: one TTL spread across the ring, so a full lap takes
// about one TTL — the upstream ticker's ttl/numBuckets. Clamped to at least one grain so the
// repeating timeout always advances the clock and never spins within a single tick.
func expirable_cleanup_interval[K comparable, V any](c *Expirable[K, V]) (interval time.Duration) {
	interval = c.TTL / BUCKET_COUNT
	if interval < 1 {
		interval = 1
	}
	return interval
}

// Reaps the next expiry bucket when it has fully passed, advancing the ring cursor. A bucket is
// reaped only once the clock is past its newest entry — every entry filed there expires no later
// than that — so a partially live bucket is left alone and the cursor does not advance past it.
func expirable_reap_ripe_bucket[K comparable, V any](c *Expirable[K, V]) {
	bucket := &c.Buckets[c.Next_Cleanup_Bucket]
	if c.Clock.Now_Monotonic() <= bucket.Newest_Entry {
		return
	}
	// Capture each next-in-chain before remove_element unlinks and frees the node.
	ent := bucket.Head
	for ent != nil {
		next := ent.Bucket_Next
		expirable_remove_element(c, ent)
		ent = next
	}
	bucket.Head = nil
	bucket.Newest_Entry = 0
	c.Next_Cleanup_Bucket = (c.Next_Cleanup_Bucket + 1) % BUCKET_COUNT
}

// Files ent at the head of the bucket one step behind the cleanup cursor, so the cursor reaches it
// only after wrapping the whole ring — about one TTL of grace, matching the upstream.
func expirable_add_to_bucket[K comparable, V any](c *Expirable[K, V], ent *Entry[K, V]) {
	bucket_identifier := uint8((BUCKET_COUNT + int(c.Next_Cleanup_Bucket) - 1) % BUCKET_COUNT)
	ent.Expire_Bucket = bucket_identifier
	bucket := &c.Buckets[bucket_identifier]
	ent.Bucket_Previous = nil
	ent.Bucket_Next = bucket.Head
	if bucket.Head != nil {
		bucket.Head.Bucket_Previous = ent
	}
	bucket.Head = ent
	if bucket.Newest_Entry < ent.Expires_At {
		bucket.Newest_Entry = ent.Expires_At
	}
}

// Unlinks ent from its bucket chain, so a renewed or removed entry does not linger there. It has
// the node in hand, so no key lookup is needed.
func expirable_remove_from_bucket[K comparable, V any](c *Expirable[K, V], ent *Entry[K, V]) {
	bucket := &c.Buckets[ent.Expire_Bucket]
	if ent.Bucket_Previous != nil {
		ent.Bucket_Previous.Bucket_Next = ent.Bucket_Next
	} else {
		bucket.Head = ent.Bucket_Next
	}
	if ent.Bucket_Next != nil {
		ent.Bucket_Next.Bucket_Previous = ent.Bucket_Previous
	}
	ent.Bucket_Next = nil
	ent.Bucket_Previous = nil
}

// Drops the least-recently-used entry, if any.
func expirable_remove_oldest[K comparable, V any](c *Expirable[K, V]) {
	ent := list_back(c.Evict_List)
	if ent != nil {
		expirable_remove_element(c, ent)
	}
}

// Unlinks ent from the list, its map entry, and its bucket, then fires On_Evict. It captures the
// key and value, and unlinks the bucket chain, before list_remove frees (and zeroes) the node.
func expirable_remove_element[K comparable, V any](c *Expirable[K, V], ent *Entry[K, V]) {
	key := ent.Key
	value := ent.Value
	delete(c.Items, key)
	expirable_remove_from_bucket(c, ent)
	list_remove(c.Evict_List, ent)
	if c.On_Evict != nil {
		c.On_Evict(key, value)
	}
}

// Panics with message when condition is false. Unlike the invariant framework it captures no
// caller site, so it inlines to a single predictable branch over a constant string — free on
// the hot path and allocation-free. The panic's stack trace carries the location; message
// names the invariant.
func assert(condition bool, message string) {
	if !condition {
		panic(message)
	}
}
