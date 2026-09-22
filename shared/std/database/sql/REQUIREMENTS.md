
# Initialization

Pool_Init binds one driver, data source, and caller-owned slot storage.
Zero slots are rejected; capacity never grows after initialization.
Package is single-threaded. Caller serializes all access to pools and handles.

# Pool

Pool acquisition reuses idle connections before opening empty slots. Exhaustion is scalar and never
allocates a waiter. Close refuses active leases and closes every idle driver connection.

# Connections

Connection is a generation-checked lease. Probe, execute, query, prepare, begin, and close reject
stale, closed, or handles with active dependent work.

# Rows

Rows owns driver cursor state and its slot reservation. Next writes exact caller storage; close or
exhaustion restores the owning connection, statement, transaction, or pool state.

# Statements

Statement pins one bounded pool slot. Execute and query reuse its driver statement until close.

# Transactions

Transaction pins one bounded pool slot. Commit and rollback are exclusive terminal operations.

# Allocation

Every public pool, connection, rows, statement, and transaction operation performs zero
allocations, including initialization and first use.

# Invariant Domains

Tests reach every registered pool bound, state, generation, status, and ownership branch.
