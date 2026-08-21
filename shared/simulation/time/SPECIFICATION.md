
# Virtual Clock

Virtual clock is deterministic and tick-driven. Clock is read-only. Time advance only when
tick returned beside it run. Simulation thus reach future Moment by tick, never by wait.

### Monotonic

Now_Monotonic is exactly tick count times resolution. Never go backward. Now_Realtime is epoch
plus that elapsed span, when no skew modeled.

### Skew

Modeled skew bend Now_Realtime away from true elapsed time: linear drift, periodic wobble, or
step jump. Now_Monotonic stay untouched.

# Timeline

Timeline is one queue every completion retire on, ordered by Ready_At. Backend fill Timeline
vtable and return Driver beside it, thus one order hold every event. New_Virtual_Timeline build
deterministic backend, never hand out its queue.

### Timeout

Timeout fire exactly when virtual clock reach its deadline. Off same Ready_At queue IO
completions use, thus every wait ride one timeline. Duration must be positive.

### Callback Released

Completion releases its callback before call, so finished operation and borrowed buffer become
collectable. Clear-before-call lets callback arm same completion again without erasing new work.

### Event

Open_Event make cross-thread event primitive. Event_Listen arm one completion. Event_Trigger
make that completion ready. Close_Event release event only after its listener drain.

### Capacity

New_Virtual_Timeline receives caller-owned state, completion storage, and event storage. This
keeps construction allocation-free and gives one hard bound to every run. A full store rejects
new work before it changes completion or event lifecycle state.

### Run Until

Run_Until drive loop until its predicate report true. Deliver completions each step, thus
straight-line caller wait for own operation inline.

### Reuse

Submit of completion still armed panic as illegal lifecycle transition. Only idle completion
can be armed. Delivery return it to idle before callback run, thus reuse after callback, or
from inside callback, is legal.

### Copy

Submit of by-value copy panic. Loop track completion by own address, thus copy carry identity
of original. Fail loud, never split view of loop from view of caller.

### Reentrancy

Drive of loop from inside completion callback panic. Run, Run_For, Run_Until are top-level
only. Re-entrant drive thus fail loud, never corrupt queue mid-drain.
