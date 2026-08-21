
# Values

Value is closed union for null, Boolean, integer, binary64 bits, bytes, text, and realtime Moment.
Bytes and text borrow caller storage and never exceed shared repository bounds.

# Named Values

Named_Value has bounded one-based ordinal. Nonempty name starts with Unicode letter, matching
standard driver rule. Arguments remain caller-owned slice.

# Results

Result holds optional last insert identifier and rows affected as scalars. Absence returns
STATUS_UNSUPPORTED without error allocation.

# Driver Surface

Driver, Connection, and Rows are procedure tables over explicit state. Connect, execute, query,
advance, probe, and close validate all bounded inputs and outputs.

# Prepared Statements

Statement carries driver state and bounded argument count. Prepare, execute, query, and close use
caller-owned result and row storage.

# Transactions

Transaction carries commit and rollback procedures. Begin validates bounded isolation and read-only
options before driver entry.

# Allocation

Every exported operation performs zero heap allocations with zero-allocation injected driver.

# Invariant Domains

Tests reach every boundary and sentinel registered by public driver operations.
