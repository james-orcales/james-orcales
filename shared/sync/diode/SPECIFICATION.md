
# Write Forwards To Sink

Bytes handed to the diode Stream are copied into the ring and submitted to the inner Stream with
one nbio.Write per line.

# Write Does Not Block

A diode Write copies into the ring and retires at once. It never touches the inner Stream, so a
Write returns the same whether that Stream is idle or holds a submission in flight.

# Overflow Drops Oldest

When producers outrun the inner Stream past the ring's capacity, the oldest unread entries are
overwritten and the newest survive.

# Drop Count Is Reported

When the drain finds the writer has lapped it, the Alerter is called with the number of entries
that were overwritten unread and the cause DROP_OVERFLOW.

# Order Is Preserved

Entries the drain delivers reach the inner Stream in the order they were written.

# Root Rearms The Drain

Writer_Rearm submits at most one queued line, only when nothing is in flight, and reports whether
it did; the retirement records that the line left and continues nothing. Writer_Work_Queued
stays true while a line is in flight, so the root pumps until the diode is empty and idle.

# Close Flushes And Stops

Close records the caller's retirement and returns. Writer_Rearm keeps submitting buffered
entries, then sends Close to the inner Stream once the ring is empty, which retires the caller's
Close. Later Writes report Stream_Empty.

# Dropping Does Not Allocate

Every bucket is allocated once in New and recycled through a lock-free free list, so no Write
allocates: a producer that overwrites an unread entry pushes the dropped bucket back, and a
Write that finds BUCKET_SLACK producers already holding one reports ERR_NO_BUCKET.

# Rate Limit Sheds By Bytes

With a bytes-per-second rate limit, the drain delivers up to the burst of bytes and sheds the rest
without blocking, reporting each shed line with cause DROP_RATE_LIMIT; tokens refill over the
injected clock so later lines pass again.
