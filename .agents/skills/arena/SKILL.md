---
name: arena
description: >
  Load BEFORE you write or change code that allocates memory, releases memory, or owns the
  lifetime of a value — an allocation for each object, a release routine, a cleanup path, a
  pool, or a temporary buffer. Also load it when a release path does not agree with its
  allocation path. Bind each allocation to an arena, and release the arena, never the
  allocation. The caller passes the arena, and thus the caller selects the lifetime.
---

## Rules

1. Do not allocate and release one object at a time. Allocate from an arena.
2. Make one arena for each lifetime. Put each allocation that has that lifetime in that
   arena.
3. Do not make one arena for each allocation.
4. Do not release one allocation. Release or clear the arena.
5. A function that allocates a result takes the arena as a parameter. The caller selects
   the arena.
6. Do not write a release routine for a structure that a function returns. Do not write
   cleanup code in the function that makes it.
7. Do not release memory that must stay for the full run of the program, or memory of a
   program that does one task and then exits.
8. Give each arena an upper bound.

## The arena

An arena is one block of memory and one allocation position.

| Operation | Procedure |
| --- | --- |
| Allocate | Return the address at the position, then add the size to the position |
| Allocate zeroed | Allocate, then set the bytes to zero |
| Get the position | Return the position |
| Set the position back | Set the position to a smaller value that you got before |
| Clear | Set the position to 0 |
| Release | Give the block back |

## Select the arena

| Lifetime of the allocation | Arena |
| --- | --- |
| The full run of the program | A permanent arena. Do not release it |
| One iteration of the top-level loop | An arena that you clear at the start of each iteration |
| One iteration, and the next iteration reads it | Two arenas. Alternate between them each iteration |
| Temporary work inside one function | A scratch arena |
| A result that the caller uses | The arena that the caller passed |
| A unit that you load and unload as one, such as a level or a document | One arena for the unit. Release it on unload |

## A temporary lifetime in an arena

1. Get the position of the arena.
2. Allocate.
3. Set the position back to the value from step 1.

## Scratch arenas

1. Give each thread two scratch arenas.
2. To get a scratch arena, pass each arena that is in scope for persistent allocations. Take
   a scratch arena that is not one of them.
3. Get the position of the scratch arena at the start. Set the position back before each
   return.
4. Do not allocate the result of the function in the scratch arena.
5. When you add an arena parameter to a function, find each place in the function that gets
   a scratch arena, and pass the new parameter there.

## Release one item from an arena

Use a pool on top of the arena. All items in the pool have one size.

1. Keep a free list of released items.
2. To allocate: when the free list is not empty, take the first item from it and set the item
   to zero. When the free list is empty, allocate a zeroed item from the arena.
3. To release: put the item at the start of the free list.

## When the arena is full

Select one:

| Condition | Procedure |
| --- | --- |
| Strict memory limit | Allocate one block of a fixed size before the work starts. Stop the program when the block is full |
| Contiguous memory is not necessary | Keep a linked list of large blocks. Get a new block when the current block is full |
| Contiguous memory is necessary, such as for an array that grows | Reserve one large range of virtual addresses. Commit pages as the position moves into them |

Before you select the reserve size, find the size of the virtual address space of each
target platform.
