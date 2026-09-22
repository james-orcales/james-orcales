---
name: errors
description: >
  Load BEFORE you write or change code that can fail, that returns an error, or that checks a
  result before it uses it. Also load it when a function has one branch for each failure. An
  error case is a case, and it goes through the same codepath as each other case. Return a
  value that is always safe to read, make the zero value valid, fail in the shallowest frame,
  return the error data together with the result, and collect error data in a log.
---

## Rules

1. Do not write a codepath for an error case. Send the error case through the codepath of
   the other cases.
2. Do not add a branch, an assertion, or an early return to check that a result is valid.
   Make the producer give a result that is always safe to read.
3. Make the zero value of each type valid. Each codepath that reads an input must do the
   sensible thing when that input is zero.
4. Put each operation that can fail in the shallowest frame of the call stack that can hold
   it. Remove each failure point that you can.
5. Return the error data together with the result. Do not return the error data in place of
   the result.
6. Read the error data only where you use it, such as where you show it to the user.
7. Collect error data in a log. Do not stop at the first error.

## Give a result that is always safe to read

1. For each type that a lookup returns by reference, make one nil instance before the
   program work starts.
2. Make the nil instance read-only.
3. Set each field of the nil instance to zero. Set each reference in the nil instance to the
   nil instance of the type that it refers to. A reference to its own type refers to itself.
4. When a lookup finds nothing, return the nil instance. Do not return a null reference.
5. Do not write to a nil instance.
6. When the data structure stores an index or a handle, resolve the zero index or handle to
   the nil instance.

Do not make a nil instance for a list that the caller reads fully in one loop. Let the loop
stop at the null reference or the zero count.

## Make the zero value valid

1. Give each sequence a count. Stop each loop at the count, or at the null reference.
2. Read a zero sequence as an empty sequence.
3. Do not write initialization code that sets a field to a value that is not zero, unless
   the field is a reference to a nil instance.
4. When the construction of a value fails, leave the value zero.

## Fail early

1. Get each resource that the operation needs before the operation starts.
2. When a resource is not available, tell the user immediately after the user starts the
   operation.
3. After the resource is there, do not check it again. Each codepath after that point is in
   the success path.
4. Allocate from a block that you reserved before the work started.

## Return the error data together with the result

1. Make the result one record with two fields: the value, and the list of messages.
2. When the operation fails, leave the value zero and add a message to the list.
3. Let a message refer to the parts of the value that it is about.
4. In the caller, use the value without a check. Read the list only to show it.

## Collect error data in a log

1. Give each thread one log: a list of messages and the storage for them.
2. When an error condition occurs, add a message to the log and continue.
3. Put in each message the call stack, the source location, and the text.
4. Read the log in a shallow frame, for example one time at the end of each loop of the
   program.
