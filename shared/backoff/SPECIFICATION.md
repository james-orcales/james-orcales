
# Constant Waits Fixed

Constant returns the same interval on every Next, and Reset leaves it unchanged.

# Zero Never Waits

Zero's Next is always zero, so the operation retries immediately.

# Stopped Never Retries

Stopped's Next is always STOP, so Retry gives up after the first failure.

# Exponential Grows And Caps

Exponential multiplies the interval by Multiplier on each Next, clamped at
Interval_Max, and Reset returns it to Initial_Interval.

# Jitter Stays Within Bounds

A jittered delay stays within Jitter of the interval either way, so it never drops
below the interval's lower spread nor exceeds its upper spread.

# Seed Reproduces Delays

Two Exponential policies over generators of the same seed emit an identical delay
sequence; a different seed diverges.

# Retry Returns First Success

Retry delivers the first successful result with a nil error and makes no further
attempts.

# Retry Stops On Permanent

A Permanent error ends Retry at once, matchable with errors.Is, with no wait and no
further attempt.

# Retry Exhausts After Tries

After Tries_Max failing attempts Retry delivers Error_Exhausted wrapping the last
failure, having run the operation exactly Tries_Max times.

# Retry Waits On Timeline

Between attempts Retry waits the policy delay on the io clock, so virtual time
advances by the sum of the delays.

# Retry After Overrides Delay

A Retry_After error makes the next wait its own Duration instead of the policy's, and
resets the policy.
