
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
