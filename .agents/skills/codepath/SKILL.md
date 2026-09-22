---
name: codepath
description: >
  Load BEFORE you add a branch, a sum type, a stored reference, a cache, or an interface that
  makes the caller create and release an object. Also load it when a caller must check a
  result before it uses it, or when a stored reference points at a slot that was used again.
  Each branch and each input data configuration multiplies the codepaths. Make each function
  give the same guarantees in all cases, store a handle with a generation, and give the
  caller a key.
---

## Rules

1. Do not add a codepath that the problem does not require.
2. Count a codepath for each branch and for each input data configuration. A function table
   or a dispatch in place of a branch removes no codepath.
3. Make each function give the same guarantees in all of its cases. The caller must use the
   result without a check.
4. Do not return a sum type. Do not return a value that is safe to read in one case and not
   safe to read in a different case.
5. Do not store a reference to a slot that can be released or used again. Store a handle
   with a generation.
6. Do not make the caller create, change, or release an object. Give the caller a key.
7. Keep the stateful data structure in one thin layer. Do not mutate it from a different
   layer.

## Make a function give the same guarantees in all cases

1. List what the caller relies on: that the result is safe to read, the type of the result,
   and that a second call gives the same effect.
2. Find each case of the function where an item in the list does not hold.
3. Change that case so that the item holds. When a lookup finds nothing, return the nil
   instance (see the `errors` skill). When a key is not in a table, insert it with the
   default value and return that value.
4. Remove each check of the result from each caller.

## Store a handle with a generation

1. Give each slot a generation number.
2. Make the handle from the index of the slot and the generation number of the slot.
3. Increment the generation number of the slot when the slot is released or used again.
4. To resolve a handle, compare the generation number of the handle with the generation
   number of the slot. When they are different, return the nil instance.
5. Store the handle. Resolve it at each use. Do not keep the resolved reference.

## Give the caller a key

1. Replace each function that loads and each function that unloads with one function that
   maps a key to a value.
2. Make each function that uses the value take the key.
3. Put the cache behind the function that maps the key. The layer that owns the cache
   decides when to load, when to load again, and when to evict.
4. Do the heavy work of the cache off the codepath of the caller. The caller only looks up.
5. While a value is not ready, return a value that is safe to use.
