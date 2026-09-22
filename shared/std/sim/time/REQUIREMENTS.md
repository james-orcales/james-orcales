
# Civil Calendar

Unix_Second_Split floors Unix seconds into Unix-relative Gregorian day and second inside day.
Civil_From_Days and Days_From_Civil convert between that day and proleptic Gregorian date. Pure
calendar arithmetic reads no ambient clock or timezone.

# Virtual Clock

Virtual clock is deterministic and tick-driven. Clock is read-only. Time advance only when
tick returned beside it run. Simulation thus reach future Moment by tick, never by wait.

### Monotonic

Now_Monotonic is exactly tick count times resolution. Never go backward. Now_Realtime is epoch
plus that elapsed span, when no skew modeled.

### Skew

Modeled skew bend Now_Realtime away from true elapsed time: linear drift, periodic wobble, or
step jump. Now_Monotonic stay untouched.
