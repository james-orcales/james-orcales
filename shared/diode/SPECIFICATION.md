
# Write Forwards To Sink

Bytes handed to Write are copied into the ring and, once the drain goroutine reads
them, written to the wrapped sink in a single Write call.

# Write Does Not Block

Write copies into the ring and returns immediately; it never calls the sink, so a
blocked or slow sink never stalls a producer.

# Overflow Drops Oldest

When producers outrun the drain past the ring's capacity, the oldest unread entries
are overwritten and the newest survive.

# Drop Count Is Reported

When the drain finds the writer has lapped it, the Alerter is called with the number
of entries that were overwritten unread.

# Order Is Preserved

Entries the drain delivers reach the sink in the order they were written.

# Poll Interval Is Configurable

The drain sleeps Poll_Interval on an empty ring via the injected clock; an unset
interval defaults to one hundred milliseconds.

# Close Flushes And Stops

Close drains the entries still buffered to the sink, stops the drain goroutine, and
closes the wrapped sink when it implements io.Closer.

# Dropping Does Not Allocate

A producer that overwrites an unread entry returns the dropped bucket to the pool, so a
diode shedding load under sustained overload allocates nothing per dropped line.
