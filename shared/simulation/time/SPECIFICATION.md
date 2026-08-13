
# Virtual Clock

A virtual clock is deterministic and tick-driven, modeled on TigerBeetle's TimeSim:
the clock is read-only and time advances only when the tick returned beside it is
called, so a simulation reaches a future Moment by ticking rather than by waiting.

### Monotonic

Now_Monotonic is exactly the tick count times the resolution and never regresses;
Now_Realtime is the epoch plus that elapsed span when no skew is modeled.

### Skew

A modeled skew bends Now_Realtime away from true elapsed time — linear drift, a
periodic wobble, or a step jump — while leaving Now_Monotonic untouched.

# Timeline

The timeline is the one queue every completion retires on, ordered by Ready_At. A backend
fills the Timeline vtable and returns a Driver beside it, so one order holds every event.
New_Virtual_Timeline builds the deterministic backend and never hands out its queue.

### Timeout

A timeout fires exactly when the virtual clock reaches its deadline, off the same
Ready_At queue the IO completions use, so every wait rides one timeline. The duration
must be positive; a caller that needs a deferred callback uses Next_Tick.

### Next Tick

Next_Tick appends a deferred callback to the completed queue without submitting kernel IO.
Reset_Next_Tick removes every queued next-tick completion for the selected source and returns
those completions to idle without invoking their callbacks.

### Event

Open_Event creates the TigerBeetle cross-thread event primitive. Event_Listen arms one
completion, Event_Trigger makes that completion ready, and Close_Event releases the event only
after its listener has drained.

### Run Until

Run_Until drives the loop until its predicate reports true, delivering completions each
step, so a straight-line caller can wait for its own operation inline.

### Reuse

Submitting a completion that is still armed panics as an illegal lifecycle transition:
only an idle completion may be armed. Delivery returns it to idle before the callback
runs, so reuse after — or from within — the callback is legal.

### Copy

Submitting a by-value copy of a completion panics: the loop tracks a completion by its own
address, so a copy carries the original's identity and fails loudly rather than splitting
the loop's view from the caller's.

### Reentrancy

Driving the loop from within a completion callback panics: Run, Run_For, and Run_Until are
top-level only, so a re-entrant drive fails loudly rather than corrupting the queue
mid-drain.
