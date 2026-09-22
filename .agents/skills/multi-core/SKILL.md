---
name: multi-core
description: >
  Load BEFORE you write or change code that uses more than one core — a parallel for, a job
  system, a worker pool, or a thread launch. Also load it when cores wait at a barrier while
  others work. Launch every thread on the same entry point one time, then go narrow where the
  work has a serial dependency. Do not go wide from single-core code. Distribute work
  uniformly, and keep one timeline.
---

## Rules

1. Launch all threads one time, on the same entry point. Give each thread its index and the
   thread count. Put all program work in the entry point.
2. Do not write a parallel for, a job system, or a worker pool.
3. Do not launch a thread from program work.
4. Go wide: each thread does only its range of the input.
5. Go narrow: only the thread with index 0 does the work. Make the narrow block only the
   serial dependency.
6. Wait on the barrier before each read of data that a different thread wrote.
7. Write shared results with an atomic operation, or to a slot that only one thread owns.
8. Deduplicate only the dominating cost. Let each thread duplicate small work.
9. To run on one core, set the thread index to 0 and the thread count to 1. Do not write a
   second codepath.
10. Do not add a second timeline.

## The range of a thread

For `count` items, `thread_count` threads, and thread index `i`:

- `per_thread = count / thread_count`
- `leftover = count % thread_count`
- `first = per_thread * i + min(i, leftover)`
- `one_past_last = first + per_thread + (1 if i < leftover, else 0)`

## Share a value that one thread made

1. The thread with index 0 makes the value and writes it to storage that all threads can
   read.
2. All threads wait on the barrier.
3. Each thread reads the value.

## Combine the results

1. Each thread computes its result into a local variable.
2. Each thread adds its result to the shared total with an atomic add.
3. All threads wait on the barrier.
4. Each thread reads the total.

## Distribute the work

| Shape of the work | Do this |
| --- | --- |
| Uniform inputs give uniform work | Give each thread its range |
| Variable work, more tasks than threads | Take tasks from a shared atomic counter |
| Fewer tasks than threads, or some tasks very long | Change the algorithm |

The shared atomic counter:

1. Set the counter to 0 narrow, then wait on the barrier.
2. Each thread increments the counter atomically and takes the task at the index that it
   got.
3. A thread stops when its index is not less than the task count.

To change the algorithm, do the first of these that applies:

1. Divide one task into smaller tasks.
2. Replace the algorithm with one where all threads divide each pass as a range. Example:
   replace a comparison sort with a radix sort.
3. Do the serially-dependent part narrow, and make it write a layout. Do the remaining work
   wide from the layout. Example: walk a linked list narrow into an array.
