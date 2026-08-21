
# Simple Evicts Oldest

New_Simple caps the cache at its size: adding past the cap evicts the least-recently-used
entry and Simple_Add reports the eviction, while re-adding an existing key overwrites in
place without eviction. Simple_Count and Simple_Cap report the count and the cap.

# Simple Get Renews Recency

Simple_Get returns a present value and moves it to the most-recently-used end, so the next
eviction falls on a different entry; a miss returns the zero value and false.

# Simple Peek And Contains Leave Recency

Simple_Peek and Simple_Contains report membership without touching recency, so an inspected
entry stays the eviction target.

# Simple Contains Or Add Skips Present Keys

Simple_Contains_Or_Add inserts a key only when absent, reporting whether it was already present
and whether the insert evicted; Simple_Peek_Or_Add does the same but returns the existing value.
Neither renews the recency of a present key.

# Simple Remove Deletes

Simple_Remove deletes a present key and reports whether it was there; Simple_Get_Oldest
returns the least-recently-used entry and Simple_Remove_Oldest returns and deletes it.

# Simple Keys And Values Fill The Buffer

Simple_Keys and Simple_Values write entries oldest to newest into the caller's buffer and return
the count, writing at most len(buffer) so the caller bounds the read.

# Simple Capacity Is Fixed

Capacity is pre-allocated and immutable: a new key into a full cache evicts the oldest before
inserting, Simple_Cap never changes, and a steady-state churn of Add and Get allocates nothing.

# Simple Evict Callback Fires

The Evict_Callback passed to New_Simple runs for every entry that an eviction, a removal, or
Simple_Purge discards.

# Two Queue Promotes Recent To Frequent

A first Two_Queue_Add lands a key in the recent list; a second access promotes it to the frequent
list, where a later burst of one-off keys cannot evict it.

# Two Queue Resists Scan

A burst of one-off keys evicts only from the recent list, so keys already promoted to frequent
survive — the scan resistance 2Q buys over a plain LRU.

# Two Queue Ghost Promotes On Re Add

A key evicted from the recent list is tracked, value-less, in the ghost list; re-adding it while
it is a ghost sends it straight to the frequent list, so it then survives a scan.

# Two Queue Reads Do Not Mutate

Two_Queue_Peek and Two_Queue_Contains report a key across the frequent and recent lists without
promoting it, so a peeked recent key is still evicted by a scan. New_Two_Queue defaults unset
ratios (one-quarter recent, one-half ghost), and Two_Queue_Cap reports the total size.

# Expirable Get Rejects Expired

Expirable_Get and Expirable_Peek consult the injected clock and return a miss for an entry past
its TTL, even before cleanup removes it; Expirable_Contains does not check expiry, so it still
reports the not-yet-reaped entry.

# Expirable Timer Reaps Expired

Expirable arms a repeating io.Timeout that reaps expired entries as the harness pumps the io loop,
walking a ring of buckets so a full lap takes about one TTL — the deterministic port of the
upstream reaper goroutine, owning no goroutine of its own.

# Expirable Evicts Oldest By Size

Beyond its size Expirable evicts the least-recently-used entry like any LRU; re-adding a key
renews both its recency and its expiry.

# Expirable Reproduces Under Virtual Clock

Two Expirable caches driven by the same virtual-clock schedule produce identical results, because
expiry reads only the injected time.Clock and the package holds no wall clock.
