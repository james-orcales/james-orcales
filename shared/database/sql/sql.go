// Package sql manages bounded database connections over caller-owned storage.
package sql

import (
	"unsafe"

	"local/james-orcales/shared/database/driver"
	"local/james-orcales/shared/invariant/default"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/slices"
)

// Status keeps pool and operation outcomes scalar and allocation-free.
type Status uint8

// Status_Invariants closes every SQL outcome.
func Status_Invariants(value Status, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint8(uint8(value), uint8(STATUS_OK), uint8(STATUS_GENERATION_EXHAUSTED)).
		Ensure()
}

// Initialization_Status reports successful initialization or missing slot storage.
type Initialization_Status Status

// Initialization_Status_Invariants keeps initialization outcomes exact.
func Initialization_Status_Invariants(
	value Initialization_Status, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_STORAGE_INVALID),
		).
		Ensure()
}

// Operation_Status excludes generation exhaustion owned by bounded acquisition.
type Operation_Status Status

// Operation_Status_Invariants bounds connection-backed operation outcomes.
func Operation_Status_Invariants(
	value Operation_Status, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Uint8(uint8(value), uint8(STATUS_OK), uint8(STATUS_BUSY)).
		Ensure()
}

// Reservation_Status reports one slot reservation decision.
type Reservation_Status Status

// Reservation_Status_Invariants keeps reservation outcomes exact.
func Reservation_Status_Invariants(
	value Reservation_Status, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Enum_4_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_CLOSED),
			uint8(STATUS_HANDLE_INVALID), uint8(STATUS_BUSY),
		).
		Ensure()
}

// Transition_Status reports one reserved slot transition decision.
type Transition_Status Status

// Transition_Status_Invariants keeps transition outcomes exact.
func Transition_Status_Invariants(
	value Transition_Status, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_HANDLE_INVALID),
			uint8(STATUS_BUSY),
		).
		Ensure()
}

// Driver_Status is one driver outcome mapped into SQL status values.
type Driver_Status Status

// Driver_Status_Invariants excludes pool-owned outcomes.
func Driver_Status_Invariants(value Driver_Status, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint8(uint8(value), uint8(STATUS_OK), uint8(STATUS_UNSUPPORTED)).
		Ensure()
}

// STATUS_OK reports completed SQL work.
const STATUS_OK Status = Status(driver.STATUS_OK)

// STATUS_DONE reports normal row exhaustion.
const STATUS_DONE Status = Status(driver.STATUS_DONE)

// STATUS_INPUT_INVALID rejects invalid request data.
const STATUS_INPUT_INVALID Status = Status(driver.STATUS_INPUT_INVALID)

// STATUS_STORAGE_INVALID rejects missing or wrongly sized caller storage.
const STATUS_STORAGE_INVALID Status = Status(driver.STATUS_STORAGE_INVALID)

// STATUS_ERROR reports driver failure.
const STATUS_ERROR Status = Status(driver.STATUS_ERROR)

// STATUS_BAD_CONNECTION reports one discarded driver connection.
const STATUS_BAD_CONNECTION Status = Status(driver.STATUS_BAD_CONNECTION)

// STATUS_UNSUPPORTED reports absent optional driver capability.
const STATUS_UNSUPPORTED Status = Status(driver.STATUS_UNSUPPORTED)

// STATUS_EXHAUSTED reports every bounded slot is reserved.
const STATUS_EXHAUSTED Status = STATUS_UNSUPPORTED + 1

// STATUS_CLOSED reports a closed pool.
const STATUS_CLOSED Status = STATUS_EXHAUSTED + 1

// STATUS_HANDLE_INVALID rejects stale or already closed handles.
const STATUS_HANDLE_INVALID Status = STATUS_CLOSED + 1

// STATUS_BUSY reports a live dependent operation.
const STATUS_BUSY Status = STATUS_HANDLE_INVALID + 1

// STATUS_GENERATION_EXHAUSTED refuses lease identity wraparound.
const STATUS_GENERATION_EXHAUSTED Status = STATUS_BUSY + 1

// Synchronizer injects exclusion without ambient synchronization state.
type Synchronizer struct {
	// State belongs to synchronization implementation.
	State unsafe.Pointer
	// Lock_Procedure enters pool critical section.
	Lock_Procedure func(state unsafe.Pointer)
	// Unlock_Procedure leaves pool critical section.
	Unlock_Procedure func(state unsafe.Pointer)
}

// Synchronizer_Invariants requires balanced critical-section capabilities.
func Synchronizer_Invariants(value Synchronizer, namespace invariant.Namespace) {
	invariant.Always(value.Lock_Procedure != nil, "A pool synchronizer can lock.")
	invariant.Always(value.Unlock_Procedure != nil, "A pool synchronizer can unlock.")
}

// CONNECTION_COUNT_MAXIMUM shares repository collection capacity.
const CONNECTION_COUNT_MAXIMUM = slices.SLICE_COUNT_MAXIMUM

// Connection_Count is one bounded pool quantity.
type Connection_Count int

// Connection_Count_Invariants bounds each pool quantity by fixed capacity.
func Connection_Count_Invariants(value Connection_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), slices.COUNT_MINIMUM, CONNECTION_COUNT_MAXIMUM).
		Ensure()
}

// SLOT_INDEX_MAXIMUM is final index in maximum bounded slot storage.
const SLOT_INDEX_MAXIMUM = CONNECTION_COUNT_MAXIMUM - 1

// Slot_Index identifies one caller-owned pool slot.
type Slot_Index int

// Slot_Index_Invariants bounds slot addressing.
func Slot_Index_Invariants(value Slot_Index, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), slices.COUNT_MINIMUM, SLOT_INDEX_MAXIMUM).
		Ensure()
}

// Generation prevents a stale handle from naming a reused slot.
type Generation uint64

// Generation_Invariants keeps complete finite lease identity domain.
func Generation_Invariants(value Generation, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// LEASE_GENERATION_MINIMUM reserves zero for empty handles and untouched slots.
const LEASE_GENERATION_MINIMUM = Generation(bits.WORD_64_MINIMUM) + 1

// Lease_Generation is one nonzero generation carried by a live handle.
type Lease_Generation Generation

// Lease_Generation_Invariants excludes the zero sentinel.
func Lease_Generation_Invariants(
	value Lease_Generation, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Uint64(
			uint64(value), uint64(LEASE_GENERATION_MINIMUM), bits.WORD_64_MAXIMUM,
		).
		Ensure()
}

// Slot_State is one state of the bounded pool slot machine.
type Slot_State uint8

// Slot_State_Invariants closes pool slot lifecycle.
func Slot_State_Invariants(value Slot_State, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint8(uint8(value), uint8(SLOT_EMPTY), uint8(SLOT_CLOSE)).
		Ensure()
}

// SLOT_EMPTY has no driver connection.
const SLOT_EMPTY Slot_State = Slot_State(bits.WORD_8_MINIMUM)

// SLOT_CONNECTION_CREATION reserves an empty slot during driver connection.
const SLOT_CONNECTION_CREATION Slot_State = SLOT_EMPTY + 1

// SLOT_IDLE holds one reusable driver connection.
const SLOT_IDLE Slot_State = SLOT_CONNECTION_CREATION + 1

// SLOT_CONNECTION belongs to one explicit or internal connection lease.
const SLOT_CONNECTION Slot_State = SLOT_IDLE + 1

// SLOT_EXECUTION reserves a connection during one driver call.
const SLOT_EXECUTION Slot_State = SLOT_CONNECTION + 1

// SLOT_ROWS belongs to one open rows cursor.
const SLOT_ROWS Slot_State = SLOT_EXECUTION + 1

// SLOT_ROWS_READING reserves rows during one advance call.
const SLOT_ROWS_READING Slot_State = SLOT_ROWS + 1

// SLOT_STATEMENT belongs to one prepared statement.
const SLOT_STATEMENT Slot_State = SLOT_ROWS_READING + 1

// SLOT_TRANSACTION belongs to one live transaction.
const SLOT_TRANSACTION Slot_State = SLOT_STATEMENT + 1

// SLOT_CLOSE reserves resource terminal work.
const SLOT_CLOSE Slot_State = SLOT_TRANSACTION + 1

// Reservation_State is one resource state that admits reservation.
type Reservation_State Slot_State

// Reservation_State_Invariants closes resource owner states.
func Reservation_State_Invariants(
	value Reservation_State, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Enum_4_Uint8(
			uint8(value), uint8(SLOT_CONNECTION), uint8(SLOT_ROWS),
			uint8(SLOT_STATEMENT), uint8(SLOT_TRANSACTION),
		).
		Ensure()
}

// Activity_State is one exclusive driver-call state.
type Activity_State Slot_State

// Activity_State_Invariants closes active slot states.
func Activity_State_Invariants(value Activity_State, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(SLOT_EXECUTION), uint8(SLOT_ROWS_READING),
			uint8(SLOT_CLOSE),
		).
		Ensure()
}

// Discard_State is one driver-call state that can report a bad connection.
type Discard_State Slot_State

// Discard_State_Invariants closes discardable active states.
func Discard_State_Invariants(value Discard_State, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(SLOT_EXECUTION), uint8(SLOT_ROWS_READING),
			uint8(SLOT_CLOSE),
		).
		Ensure()
}

// SLOT_FIELD is sole slot property position.
const SLOT_FIELD = slices.COUNT_MINIMUM

// SLOT_FIELD_COUNT keeps each pool slot aggregate fixed.
const SLOT_FIELD_COUNT = SLOT_FIELD + 1

// Slot is caller-owned storage for one driver connection and lease identity.
type Slot struct {
	// Connections holds one reusable driver connection.
	Connections [SLOT_FIELD_COUNT]driver.Connection
	// States holds one slot lifecycle state.
	States [SLOT_FIELD_COUNT]Slot_State
	// Generations holds current lease identity.
	Generations [SLOT_FIELD_COUNT]Generation
}

// Slot_Invariants states fixed caller-owned slot storage.
func Slot_Invariants(value Slot, namespace invariant.Namespace) {
	invariant.Always(
		len(value.Connections) == SLOT_FIELD_COUNT,
		"A pool slot holds one driver connection.",
	)
	invariant.Always(
		len(value.States) == SLOT_FIELD_COUNT,
		"A pool slot holds one lifecycle state.",
	)
	invariant.Always(
		len(value.Generations) == SLOT_FIELD_COUNT,
		"A pool slot holds one lease generation.",
	)
}

// Slot_Storage is bounded caller-owned pool capacity.
type Slot_Storage []Slot

// Slot_Storage_Invariants bounds fixed pool capacity.
func Slot_Storage_Invariants(value Slot_Storage, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, CONNECTION_COUNT_MAXIMUM).
		Ensure()
}

// INITIALIZED_CONNECTION_COUNT_MINIMUM excludes rejected empty storage.
const INITIALIZED_CONNECTION_COUNT_MINIMUM = slices.COUNT_MINIMUM + 1

// Initialized_Slot_Storage is bound nonempty pool capacity.
type Initialized_Slot_Storage Slot_Storage

// Initialized_Slot_Storage_Invariants excludes rejected empty storage.
func Initialized_Slot_Storage_Invariants(
	value Initialized_Slot_Storage, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(
			len(value), INITIALIZED_CONNECTION_COUNT_MINIMUM,
			CONNECTION_COUNT_MAXIMUM,
		).
		Ensure()
}

// Pool_State distinguishes zero, open, and closed pools.
type Pool_State uint8

// Pool_State_Invariants closes pool lifecycle.
func Pool_State_Invariants(value Pool_State, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(POOL_UNINITIALIZED), uint8(POOL_OPEN),
			uint8(POOL_CLOSED),
		).
		Ensure()
}

// POOL_UNINITIALIZED is zero pool before Pool_Init.
const POOL_UNINITIALIZED Pool_State = Pool_State(bits.WORD_8_MINIMUM)

// POOL_OPEN admits acquisition and operations.
const POOL_OPEN Pool_State = POOL_UNINITIALIZED + 1

// POOL_CLOSED rejects future acquisition.
const POOL_CLOSED Pool_State = POOL_OPEN + 1

// Initialized_Pool_State excludes a zero pool without bound dependencies.
type Initialized_Pool_State Pool_State

// Initialized_Pool_State_Invariants closes bound pool lifecycle.
func Initialized_Pool_State_Invariants(
	value Initialized_Pool_State, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(POOL_OPEN), uint8(POOL_CLOSED),
		).
		Ensure()
}

// POOL_FIELD is sole pool property position.
const POOL_FIELD = slices.COUNT_MINIMUM

// POOL_FIELD_COUNT keeps pool aggregate fixed.
const POOL_FIELD_COUNT = POOL_FIELD + 1

// Pool owns no dynamic storage; every connection slot belongs to caller.
type Pool struct {
	// Drivers holds injected driver root.
	Drivers [POOL_FIELD_COUNT]driver.Driver
	// Data_Sources holds validated driver connection text.
	Data_Sources [POOL_FIELD_COUNT]driver.Data_Source
	// Synchronizers holds injected exclusion.
	Synchronizers [POOL_FIELD_COUNT]Synchronizer
	// Slot_Sets borrows caller-owned fixed capacity.
	Slot_Sets [POOL_FIELD_COUNT]Slot_Storage
	// States holds pool lifecycle.
	States [POOL_FIELD_COUNT]Pool_State
}

// Pool_Invariants states fixed pool aggregate storage.
func Pool_Invariants(subject *Pool, namespace invariant.Namespace) {
	invariant.Always(subject != nil, "Pool storage exists.")
	invariant.Always(
		len(subject.Drivers) == POOL_FIELD_COUNT,
		"A pool holds one driver slot.",
	)
	invariant.Always(
		len(subject.Data_Sources) == POOL_FIELD_COUNT,
		"A pool holds one data-source slot.",
	)
	invariant.Always(
		len(subject.Synchronizers) == POOL_FIELD_COUNT,
		"A pool holds one synchronizer slot.",
	)
	invariant.Always(
		len(subject.Slot_Sets) == POOL_FIELD_COUNT,
		"A pool holds one slot-storage view.",
	)
	invariant.Always(
		len(subject.States) == POOL_FIELD_COUNT,
		"A pool holds one lifecycle slot.",
	)
}

// HANDLE_FIELD is sole handle property position.
const HANDLE_FIELD = slices.COUNT_MINIMUM

// HANDLE_FIELD_COUNT keeps every lease handle aggregate fixed.
const HANDLE_FIELD_COUNT = HANDLE_FIELD + 1

// Connection is one generation-checked pool lease.
type Connection struct {
	// Pools names owning pool; nil marks zero or closed handle.
	Pools [HANDLE_FIELD_COUNT]*Pool
	// Slot_Indices identifies one caller slot.
	Slot_Indices [HANDLE_FIELD_COUNT]Slot_Index
	// Generations identifies one acquisition of that slot.
	Generations [HANDLE_FIELD_COUNT]Lease_Generation
}

// Connection_Invariants states fixed lease handle storage.
func Connection_Invariants(subject Connection, namespace invariant.Namespace) {
	invariant.Always(
		len(subject.Pools) == HANDLE_FIELD_COUNT,
		"A connection handle holds one pool slot.",
	)
	invariant.Always(
		len(subject.Slot_Indices) == HANDLE_FIELD_COUNT,
		"A connection handle holds one index slot.",
	)
	invariant.Always(
		len(subject.Generations) == HANDLE_FIELD_COUNT,
		"A connection handle holds one generation slot.",
	)
}

// Return_State is state restored after a dependent resource closes.
type Return_State Slot_State

// Return_State_Invariants permits pool, connection, statement, and transaction owners.
func Return_State_Invariants(value Return_State, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_4_Uint8(
			uint8(value), uint8(SLOT_IDLE), uint8(SLOT_CONNECTION),
			uint8(SLOT_STATEMENT), uint8(SLOT_TRANSACTION),
		).
		Ensure()
}

// Connection_Return_State restores an internal operation to pool or explicit lease.
type Connection_Return_State Return_State

// Connection_Return_State_Invariants closes connection operation ownership.
func Connection_Return_State_Invariants(
	value Connection_Return_State, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(SLOT_IDLE), uint8(SLOT_CONNECTION),
		).
		Ensure()
}

// Transition_Kind selects rows ownership or one explicit return state.
type Transition_Kind uint8

// Transition_Kind_Invariants closes transition target representation.
func Transition_Kind_Invariants(value Transition_Kind, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(TRANSITION_RETURN), uint8(TRANSITION_ROWS),
		).
		Ensure()
}

// TRANSITION_RETURN restores one Return_State.
const TRANSITION_RETURN Transition_Kind = Transition_Kind(bits.WORD_8_MINIMUM)

// TRANSITION_ROWS transfers ownership to one rows cursor.
const TRANSITION_ROWS Transition_Kind = TRANSITION_RETURN + 1

// Rows carries one driver cursor and reserved pool lease.
type Rows struct {
	// Connections names reserved slot.
	Connections [HANDLE_FIELD_COUNT]Connection
	// Driver_Rows holds driver cursor state.
	Driver_Rows [HANDLE_FIELD_COUNT]driver.Rows
	// Return_States selects state restored by close or exhaustion.
	Return_States [HANDLE_FIELD_COUNT]Return_State
}

// Rows_Invariants states fixed rows handle storage.
func Rows_Invariants(subject Rows, namespace invariant.Namespace) {
	invariant.Always(
		len(subject.Connections) == HANDLE_FIELD_COUNT,
		"Rows holds one connection handle.",
	)
	invariant.Always(
		len(subject.Driver_Rows) == HANDLE_FIELD_COUNT,
		"Rows holds one driver cursor.",
	)
	invariant.Always(
		len(subject.Return_States) == HANDLE_FIELD_COUNT,
		"Rows holds one return state.",
	)
}

// Statement carries one driver statement and reserved pool lease.
type Statement struct {
	// Connections names reserved slot.
	Connections [HANDLE_FIELD_COUNT]Connection
	// Driver_Statements holds driver prepared state.
	Driver_Statements [HANDLE_FIELD_COUNT]driver.Statement
	// Return_States selects state restored by close.
	Return_States [HANDLE_FIELD_COUNT]Return_State
}

// Statement_Invariants states fixed statement handle storage.
func Statement_Invariants(subject Statement, namespace invariant.Namespace) {
	invariant.Always(
		len(subject.Connections) == HANDLE_FIELD_COUNT,
		"Statement holds one connection handle.",
	)
	invariant.Always(
		len(subject.Driver_Statements) == HANDLE_FIELD_COUNT,
		"Statement holds one driver statement.",
	)
	invariant.Always(
		len(subject.Return_States) == HANDLE_FIELD_COUNT,
		"Statement holds one return state.",
	)
}

// Transaction carries one driver transaction and reserved pool lease.
type Transaction struct {
	// Connections names reserved slot.
	Connections [HANDLE_FIELD_COUNT]Connection
	// Driver_Transactions holds driver transaction state.
	Driver_Transactions [HANDLE_FIELD_COUNT]driver.Transaction
	// Return_States selects state restored by commit or rollback.
	Return_States [HANDLE_FIELD_COUNT]Return_State
}

// Transaction_Invariants states fixed transaction handle storage.
func Transaction_Invariants(subject Transaction, namespace invariant.Namespace) {
	invariant.Always(
		len(subject.Connections) == HANDLE_FIELD_COUNT,
		"Transaction holds one connection handle.",
	)
	invariant.Always(
		len(subject.Driver_Transactions) == HANDLE_FIELD_COUNT,
		"Transaction holds one driver transaction.",
	)
	invariant.Always(
		len(subject.Return_States) == HANDLE_FIELD_COUNT,
		"Transaction holds one return state.",
	)
}

// Transaction_Terminal selects commit or rollback.
type Transaction_Terminal uint8

// Transaction_Terminal_Invariants closes terminal operation choice.
func Transaction_Terminal_Invariants(
	value Transaction_Terminal, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(TRANSACTION_ROLLBACK), uint8(TRANSACTION_COMMIT),
		).
		Ensure()
}

// TRANSACTION_ROLLBACK abandons transaction changes.
const TRANSACTION_ROLLBACK Transaction_Terminal = Transaction_Terminal(bits.WORD_8_MINIMUM)

// TRANSACTION_COMMIT makes transaction changes durable.
const TRANSACTION_COMMIT Transaction_Terminal = TRANSACTION_ROLLBACK + 1

// Result carries one driver result without interface allocation.
type Result struct {
	// Driver_Results holds one driver execution result.
	Driver_Results [HANDLE_FIELD_COUNT]driver.Result
}

// Result_Invariants states fixed result storage.
func Result_Invariants(value Result, namespace invariant.Namespace) {
	invariant.Always(
		len(value.Driver_Results) == HANDLE_FIELD_COUNT,
		"SQL Result holds one driver result.",
	)
}

// STATISTICS_FIELD is sole statistics property position.
const STATISTICS_FIELD = slices.COUNT_MINIMUM

// STATISTICS_FIELD_COUNT keeps statistics aggregate fixed.
const STATISTICS_FIELD_COUNT = STATISTICS_FIELD + 1

// Statistics is one bounded snapshot of pool slot counts.
type Statistics struct {
	// Capacities holds fixed caller capacity.
	Capacities [STATISTICS_FIELD_COUNT]Connection_Count
	// Open_Counts holds slots with driver connections.
	Open_Counts [STATISTICS_FIELD_COUNT]Connection_Count
	// Idle_Counts holds reusable connections.
	Idle_Counts [STATISTICS_FIELD_COUNT]Connection_Count
	// In_Use_Counts holds reserved connections.
	In_Use_Counts [STATISTICS_FIELD_COUNT]Connection_Count
}

// Statistics_Invariants states fixed statistics storage.
func Statistics_Invariants(value Statistics, namespace invariant.Namespace) {
	invariant.Always(
		len(value.Capacities) == STATISTICS_FIELD_COUNT,
		"Statistics holds one capacity slot.",
	)
	invariant.Always(
		len(value.Open_Counts) == STATISTICS_FIELD_COUNT,
		"Statistics holds one open-count slot.",
	)
	invariant.Always(
		len(value.Idle_Counts) == STATISTICS_FIELD_COUNT,
		"Statistics holds one idle-count slot.",
	)
	invariant.Always(
		len(value.In_Use_Counts) == STATISTICS_FIELD_COUNT,
		"Statistics holds one in-use-count slot.",
	)
}

// Pool_Init binds injected dependencies to fixed caller-owned slots.
func Pool_Init(
	pool *Pool, injected driver.Driver, data_source driver.Data_Source,
	synchronizer Synchronizer, slots Slot_Storage,
) (status Initialization_Status) {
	defer func() {
		Initialization_Status_Invariants(status, "pool_init.status")
	}()
	Pool_Invariants(pool, "pool_init.pool")
	driver.Driver_Invariants(injected, "pool_init.injected")
	driver.Data_Source_Invariants(data_source, "pool_init.data_source")
	Synchronizer_Invariants(synchronizer, "pool_init.synchronizer")
	Slot_Storage_Invariants(slots, "pool_init.slots")
	if len(slots) == slices.COUNT_MINIMUM {
		return Initialization_Status(STATUS_STORAGE_INVALID)
	}
	for index := range slots {
		Slot_Invariants(slots[index], "pool_init.slot")
		slots[index] = Slot{}
	}
	*pool = Pool{
		Drivers:       [POOL_FIELD_COUNT]driver.Driver{injected},
		Data_Sources:  [POOL_FIELD_COUNT]driver.Data_Source{data_source},
		Synchronizers: [POOL_FIELD_COUNT]Synchronizer{synchronizer},
		Slot_Sets:     [POOL_FIELD_COUNT]Slot_Storage{slots},
		States:        [POOL_FIELD_COUNT]Pool_State{POOL_OPEN},
	}
	return Initialization_Status(STATUS_OK)
}

// Pool_Connection_Acquire reserves idle storage or opens one empty slot.
func Pool_Connection_Acquire(pool *Pool) (connection Connection, status Status) {
	defer func() {
		Connection_Invariants(connection, "pool_connection_acquire.connection")
		Status_Invariants(status, "pool_connection_acquire.status")
	}()
	Pool_Invariants(pool, "pool_connection_acquire.pool")
	pool_initialized(pool)
	synchronizer := pool.Synchronizers[POOL_FIELD]
	synchronizer.Lock_Procedure(synchronizer.State)
	if pool.States[POOL_FIELD] == POOL_CLOSED {
		synchronizer.Unlock_Procedure(synchronizer.State)
		return Connection{}, STATUS_CLOSED
	}
	slots := pool.Slot_Sets[POOL_FIELD]
	generation_exhausted := false
	for index := range slots {
		slot := &slots[index]
		if slot.States[SLOT_FIELD] != SLOT_IDLE {
			continue
		}
		if slot.Generations[SLOT_FIELD] == Generation(bits.WORD_64_MAXIMUM) {
			generation_exhausted = true
			continue
		}
		slot.Generations[SLOT_FIELD]++
		slot.States[SLOT_FIELD] = SLOT_CONNECTION
		connection = connection_of(
			pool, Slot_Index(index), Lease_Generation(slot.Generations[SLOT_FIELD]),
		)
		synchronizer.Unlock_Procedure(synchronizer.State)
		return connection, STATUS_OK
	}
	for index := range slots {
		slot := &slots[index]
		if slot.States[SLOT_FIELD] != SLOT_EMPTY {
			continue
		}
		if slot.Generations[SLOT_FIELD] == Generation(bits.WORD_64_MAXIMUM) {
			generation_exhausted = true
			continue
		}
		slot.Generations[SLOT_FIELD]++
		generation := slot.Generations[SLOT_FIELD]
		slot.States[SLOT_FIELD] = SLOT_CONNECTION_CREATION
		slot.Connections[SLOT_FIELD] = driver.Connection{}
		synchronizer.Unlock_Procedure(synchronizer.State)
		driver_status := driver.Connect(
			pool.Drivers[POOL_FIELD], pool.Data_Sources[POOL_FIELD],
			&slot.Connections[SLOT_FIELD],
		)
		synchronizer.Lock_Procedure(synchronizer.State)
		if driver_status != driver.STATUS_OK {
			slot.Connections[SLOT_FIELD] = driver.Connection{}
			slot.States[SLOT_FIELD] = SLOT_EMPTY
			synchronizer.Unlock_Procedure(synchronizer.State)
			return Connection{}, Status(status_of_driver(driver_status))
		}
		slot.States[SLOT_FIELD] = SLOT_CONNECTION
		connection = connection_of(
			pool, Slot_Index(index), Lease_Generation(generation),
		)
		synchronizer.Unlock_Procedure(synchronizer.State)
		return connection, STATUS_OK
	}
	synchronizer.Unlock_Procedure(synchronizer.State)
	if generation_exhausted {
		return Connection{}, STATUS_GENERATION_EXHAUSTED
	}
	return Connection{}, STATUS_EXHAUSTED
}

// Connection_Close returns one explicit lease to idle pool state.
func Connection_Close(connection *Connection) (status Reservation_Status) {
	defer func() {
		Reservation_Status_Invariants(status, "connection_close.status")
	}()
	Connection_Invariants(*connection, "connection_close.connection")
	var table driver.Connection
	status = connection_reserve(
		connection, Reservation_State(SLOT_CONNECTION), Activity_State(SLOT_CLOSE), &table,
	)
	if status != Reservation_Status(STATUS_OK) {
		return status
	}
	transition_status := connection_transition(
		connection, Activity_State(SLOT_CLOSE), TRANSITION_RETURN,
		Return_State(SLOT_IDLE),
	)
	if transition_status == Transition_Status(STATUS_OK) {
		*connection = Connection{}
	}
	return Reservation_Status(transition_status)
}

// Pool_Close closes every idle connection after refusing active leases.
func Pool_Close(pool *Pool) (status Operation_Status) {
	defer func() { Operation_Status_Invariants(status, "pool_close.status") }()
	Pool_Invariants(pool, "pool_close.pool")
	pool_initialized(pool)
	synchronizer := pool.Synchronizers[POOL_FIELD]
	synchronizer.Lock_Procedure(synchronizer.State)
	if pool.States[POOL_FIELD] == POOL_CLOSED {
		synchronizer.Unlock_Procedure(synchronizer.State)
		return Operation_Status(STATUS_OK)
	}
	slots := pool.Slot_Sets[POOL_FIELD]
	for index := range slots {
		state := slots[index].States[SLOT_FIELD]
		if state != SLOT_EMPTY {
			if state != SLOT_IDLE {
				synchronizer.Unlock_Procedure(synchronizer.State)
				return Operation_Status(STATUS_BUSY)
			}
		}
	}
	pool.States[POOL_FIELD] = POOL_CLOSED
	for index := range slots {
		if slots[index].States[SLOT_FIELD] == SLOT_IDLE {
			slots[index].States[SLOT_FIELD] = SLOT_CLOSE
		}
	}
	synchronizer.Unlock_Procedure(synchronizer.State)
	status = Operation_Status(STATUS_OK)
	for index := range slots {
		slot := &slots[index]
		if slot.States[SLOT_FIELD] != SLOT_CLOSE {
			continue
		}
		driver_status := driver.Connection_Close(&slot.Connections[SLOT_FIELD])
		if status == Operation_Status(STATUS_OK) {
			if driver_status != driver.STATUS_OK {
				status = Operation_Status(status_of_driver(driver_status))
			}
		}
		slot.Connections[SLOT_FIELD] = driver.Connection{}
		slot.States[SLOT_FIELD] = SLOT_EMPTY
	}
	return status
}

// Pool_Statistics returns one synchronized bounded count snapshot.
func Pool_Statistics(pool *Pool) (statistics Statistics) {
	defer func() { Statistics_Invariants(statistics, "pool_statistics.statistics") }()
	Pool_Invariants(pool, "pool_statistics.pool")
	pool_initialized(pool)
	synchronizer := pool.Synchronizers[POOL_FIELD]
	synchronizer.Lock_Procedure(synchronizer.State)
	slots := pool.Slot_Sets[POOL_FIELD]
	statistics.Capacities[STATISTICS_FIELD] = Connection_Count(len(slots))
	for index := range slots {
		state := slots[index].States[SLOT_FIELD]
		if state != SLOT_EMPTY {
			statistics.Open_Counts[STATISTICS_FIELD]++
		}
		if state == SLOT_IDLE {
			statistics.Idle_Counts[STATISTICS_FIELD]++
		}
		if state != SLOT_EMPTY {
			if state != SLOT_IDLE {
				if state != SLOT_CONNECTION_CREATION {
					statistics.In_Use_Counts[STATISTICS_FIELD]++
				}
			}
		}
	}
	synchronizer.Unlock_Procedure(synchronizer.State)
	return statistics
}

// Statistics_Capacity reports fixed caller slot count.
func Statistics_Capacity(value Statistics) (count Connection_Count) {
	defer func() { Connection_Count_Invariants(count, "statistics_capacity.count") }()
	Statistics_Invariants(value, "statistics_capacity.value")
	return value.Capacities[STATISTICS_FIELD]
}

// Statistics_Open reports slots with driver connections.
func Statistics_Open(value Statistics) (count Connection_Count) {
	defer func() { Connection_Count_Invariants(count, "statistics_open.count") }()
	Statistics_Invariants(value, "statistics_open.value")
	return value.Open_Counts[STATISTICS_FIELD]
}

// Statistics_Idle reports reusable driver connections.
func Statistics_Idle(value Statistics) (count Connection_Count) {
	defer func() { Connection_Count_Invariants(count, "statistics_idle.count") }()
	Statistics_Invariants(value, "statistics_idle.value")
	return value.Idle_Counts[STATISTICS_FIELD]
}

// Statistics_In_Use reports reserved driver connections.
func Statistics_In_Use(value Statistics) (count Connection_Count) {
	defer func() { Connection_Count_Invariants(count, "statistics_in_use.count") }()
	Statistics_Invariants(value, "statistics_in_use.value")
	return value.In_Use_Counts[STATISTICS_FIELD]
}

// Result_Rows_Affected reports driver affected count when supported.
func Result_Rows_Affected(
	result Result,
) (count driver.Rows_Affected, status driver.Optional_Status) {
	defer func() {
		driver.Rows_Affected_Invariants(count, "result_rows_affected.count")
		driver.Optional_Status_Invariants(status, "result_rows_affected.status")
	}()
	Result_Invariants(result, "result_rows_affected.result")
	return driver.Result_Rows_Affected(&result.Driver_Results[HANDLE_FIELD])
}

// Result_Last_Insert_Identifier reports generated identity when supported.
func Result_Last_Insert_Identifier(
	result Result,
) (identifier driver.Last_Insert_Identifier, status driver.Optional_Status) {
	defer func() {
		driver.Last_Insert_Identifier_Invariants(
			identifier, "result_last_insert_identifier.identifier",
		)
		driver.Optional_Status_Invariants(status, "result_last_insert_identifier.status")
	}()
	Result_Invariants(result, "result_last_insert_identifier.result")
	return driver.Result_Last_Insert_Identifier(&result.Driver_Results[HANDLE_FIELD])
}

// Pool_Probe probes one bounded connection and returns it to pool.
func Pool_Probe(pool *Pool) (status Status) {
	defer func() { Status_Invariants(status, "pool_probe.status") }()
	Pool_Invariants(pool, "pool_probe.pool")
	connection, status := Pool_Connection_Acquire(pool)
	if status != STATUS_OK {
		return status
	}
	return Status(connection_probe(
		&connection, Connection_Return_State(SLOT_IDLE),
	))
}

// Connection_Probe probes one explicit connection lease.
func Connection_Probe(connection *Connection) (status Operation_Status) {
	defer func() { Operation_Status_Invariants(status, "connection_probe.status") }()
	Connection_Invariants(*connection, "connection_probe.connection")
	return connection_probe(connection, Connection_Return_State(SLOT_CONNECTION))
}

// Pool_Exec executes one request into caller-owned result storage.
func Pool_Exec(pool *Pool, request driver.Request, result *Result) (status Status) {
	defer func() { Status_Invariants(status, "pool_exec.status") }()
	Pool_Invariants(pool, "pool_exec.pool")
	driver.Request_Invariants(request, "pool_exec.request")
	Result_Invariants(*result, "pool_exec.result")
	connection, status := Pool_Connection_Acquire(pool)
	if status != STATUS_OK {
		return status
	}
	return Status(connection_exec(
		&connection, request, Connection_Return_State(SLOT_IDLE), result,
	))
}

// Connection_Exec executes one request through explicit lease.
func Connection_Exec(
	connection *Connection, request driver.Request, result *Result,
) (status Operation_Status) {
	defer func() { Operation_Status_Invariants(status, "connection_exec.status") }()
	Connection_Invariants(*connection, "connection_exec.connection")
	driver.Request_Invariants(request, "connection_exec.request")
	Result_Invariants(*result, "connection_exec.result")
	return connection_exec(
		connection, request, Connection_Return_State(SLOT_CONNECTION), result,
	)
}

// Pool_Query opens rows into caller-owned cursor storage.
func Pool_Query(pool *Pool, request driver.Request, rows *Rows) (status Status) {
	defer func() { Status_Invariants(status, "pool_query.status") }()
	Pool_Invariants(pool, "pool_query.pool")
	driver.Request_Invariants(request, "pool_query.request")
	Rows_Invariants(*rows, "pool_query.rows")
	connection, status := Pool_Connection_Acquire(pool)
	if status != STATUS_OK {
		return status
	}
	return Status(connection_query(
		&connection, request, Connection_Return_State(SLOT_IDLE), rows,
	))
}

// Connection_Query opens rows through explicit lease.
func Connection_Query(
	connection *Connection, request driver.Request, rows *Rows,
) (status Operation_Status) {
	defer func() { Operation_Status_Invariants(status, "connection_query.status") }()
	Connection_Invariants(*connection, "connection_query.connection")
	driver.Request_Invariants(request, "connection_query.request")
	Rows_Invariants(*rows, "connection_query.rows")
	return connection_query(
		connection, request, Connection_Return_State(SLOT_CONNECTION), rows,
	)
}

// Rows_Next writes one exact-width row and closes automatically at exhaustion.
func Rows_Next(rows *Rows, destination driver.Values) (status Operation_Status) {
	defer func() { Operation_Status_Invariants(status, "rows_next.status") }()
	Rows_Invariants(*rows, "rows_next.rows")
	driver.Values_Invariants(destination, "rows_next.destination")
	connection := &rows.Connections[HANDLE_FIELD]
	var table driver.Connection
	reservation_status := connection_reserve(
		connection, Reservation_State(SLOT_ROWS),
		Activity_State(SLOT_ROWS_READING), &table,
	)
	if reservation_status != Reservation_Status(STATUS_OK) {
		return Operation_Status(reservation_status)
	}
	driver_status := driver.Rows_Next(&rows.Driver_Rows[HANDLE_FIELD], destination)
	if driver_status == driver.STATUS_OK {
		return Operation_Status(
			connection_transition(
				connection, Activity_State(SLOT_ROWS_READING), TRANSITION_ROWS,
				Return_State(SLOT_IDLE),
			),
		)
	}
	if driver_status == driver.STATUS_DONE {
		close_status := driver.Rows_Close(&rows.Driver_Rows[HANDLE_FIELD])
		if close_status != driver.STATUS_OK {
			status = operation_complete(
				connection, &table, Activity_State(SLOT_ROWS_READING),
				rows.Return_States[HANDLE_FIELD], close_status,
			)
			*rows = Rows{}
			return status
		}
		transition_status := connection_transition(
			connection, Activity_State(SLOT_ROWS_READING), TRANSITION_RETURN,
			rows.Return_States[HANDLE_FIELD],
		)
		*rows = Rows{}
		if transition_status != Transition_Status(STATUS_OK) {
			return Operation_Status(transition_status)
		}
		return Operation_Status(STATUS_DONE)
	}
	if driver_status == driver.STATUS_BAD_CONNECTION {
		driver.Connection_Close(&table)
		transition_status := connection_discard(
			connection, Discard_State(SLOT_ROWS_READING),
		)
		*rows = Rows{}
		if transition_status != Transition_Status(STATUS_OK) {
			return Operation_Status(transition_status)
		}
		return Operation_Status(STATUS_BAD_CONNECTION)
	}
	transition_status := connection_transition(
		connection, Activity_State(SLOT_ROWS_READING), TRANSITION_ROWS,
		Return_State(SLOT_IDLE),
	)
	if transition_status != Transition_Status(STATUS_OK) {
		return Operation_Status(transition_status)
	}
	return Operation_Status(status_of_driver(driver_status))
}

// Rows_Close releases cursor and restores its owning slot state.
func Rows_Close(rows *Rows) (status Operation_Status) {
	defer func() { Operation_Status_Invariants(status, "rows_close.status") }()
	Rows_Invariants(*rows, "rows_close.rows")
	connection := &rows.Connections[HANDLE_FIELD]
	var table driver.Connection
	reservation_status := connection_reserve(
		connection, Reservation_State(SLOT_ROWS), Activity_State(SLOT_CLOSE), &table,
	)
	if reservation_status != Reservation_Status(STATUS_OK) {
		return Operation_Status(reservation_status)
	}
	driver_status := driver.Rows_Close(&rows.Driver_Rows[HANDLE_FIELD])
	status = operation_complete(
		connection, &table, Activity_State(SLOT_CLOSE),
		rows.Return_States[HANDLE_FIELD], driver_status,
	)
	*rows = Rows{}
	return status
}

// Pool_Prepare pins one lease behind caller-owned prepared statement.
func Pool_Prepare(
	pool *Pool, query driver.Query, statement *Statement,
) (status Status) {
	defer func() { Status_Invariants(status, "pool_prepare.status") }()
	Pool_Invariants(pool, "pool_prepare.pool")
	driver.Query_Invariants(query, "pool_prepare.query")
	Statement_Invariants(*statement, "pool_prepare.statement")
	connection, status := Pool_Connection_Acquire(pool)
	if status != STATUS_OK {
		return status
	}
	return Status(connection_prepare(
		&connection, query, Connection_Return_State(SLOT_IDLE), statement,
	))
}

// Connection_Prepare pins explicit lease behind prepared statement.
func Connection_Prepare(
	connection *Connection, query driver.Query, statement *Statement,
) (status Operation_Status) {
	defer func() { Operation_Status_Invariants(status, "connection_prepare.status") }()
	Connection_Invariants(*connection, "connection_prepare.connection")
	driver.Query_Invariants(query, "connection_prepare.query")
	Statement_Invariants(*statement, "connection_prepare.statement")
	return connection_prepare(
		connection, query, Connection_Return_State(SLOT_CONNECTION), statement,
	)
}

// Statement_Exec executes one prepared request.
func Statement_Exec(
	statement *Statement, arguments driver.Arguments, result *Result,
) (status Operation_Status) {
	defer func() { Operation_Status_Invariants(status, "statement_exec.status") }()
	Statement_Invariants(*statement, "statement_exec.statement")
	driver.Arguments_Invariants(arguments, "statement_exec.arguments")
	Result_Invariants(*result, "statement_exec.result")
	connection := &statement.Connections[HANDLE_FIELD]
	var table driver.Connection
	reservation_status := connection_reserve(
		connection, Reservation_State(SLOT_STATEMENT),
		Activity_State(SLOT_EXECUTION), &table,
	)
	if reservation_status != Reservation_Status(STATUS_OK) {
		return Operation_Status(reservation_status)
	}
	*result = Result{}
	driver_status := driver.Statement_Exec(
		&statement.Driver_Statements[HANDLE_FIELD], arguments,
		&result.Driver_Results[HANDLE_FIELD],
	)
	status = operation_complete(
		connection, &table, Activity_State(SLOT_EXECUTION),
		Return_State(SLOT_STATEMENT), driver_status,
	)
	return status
}

// Statement_Query opens rows through one prepared statement.
func Statement_Query(
	statement *Statement, arguments driver.Arguments, rows *Rows,
) (status Operation_Status) {
	defer func() { Operation_Status_Invariants(status, "statement_query.status") }()
	Statement_Invariants(*statement, "statement_query.statement")
	driver.Arguments_Invariants(arguments, "statement_query.arguments")
	Rows_Invariants(*rows, "statement_query.rows")
	connection := &statement.Connections[HANDLE_FIELD]
	var table driver.Connection
	reservation_status := connection_reserve(
		connection, Reservation_State(SLOT_STATEMENT),
		Activity_State(SLOT_EXECUTION), &table,
	)
	if reservation_status != Reservation_Status(STATUS_OK) {
		return Operation_Status(reservation_status)
	}
	*rows = rows_of(connection, Return_State(SLOT_STATEMENT))
	driver_status := driver.Statement_Query(
		&statement.Driver_Statements[HANDLE_FIELD], arguments,
		&rows.Driver_Rows[HANDLE_FIELD],
	)
	if driver_status != driver.STATUS_OK {
		status = operation_complete(
			connection, &table, Activity_State(SLOT_EXECUTION),
			Return_State(SLOT_STATEMENT), driver_status,
		)
		*rows = Rows{}
		return status
	}
	transition_status := connection_transition(
		connection, Activity_State(SLOT_EXECUTION), TRANSITION_ROWS,
		Return_State(SLOT_IDLE),
	)
	if transition_status != Transition_Status(STATUS_OK) {
		driver.Rows_Close(&rows.Driver_Rows[HANDLE_FIELD])
		*rows = Rows{}
		return Operation_Status(transition_status)
	}
	return Operation_Status(STATUS_OK)
}

// Statement_Close releases prepared state and restores owning lease.
func Statement_Close(statement *Statement) (status Operation_Status) {
	defer func() { Operation_Status_Invariants(status, "statement_close.status") }()
	Statement_Invariants(*statement, "statement_close.statement")
	connection := &statement.Connections[HANDLE_FIELD]
	var table driver.Connection
	reservation_status := connection_reserve(
		connection, Reservation_State(SLOT_STATEMENT), Activity_State(SLOT_CLOSE), &table,
	)
	if reservation_status != Reservation_Status(STATUS_OK) {
		return Operation_Status(reservation_status)
	}
	driver_status := driver.Statement_Close(&statement.Driver_Statements[HANDLE_FIELD])
	status = operation_complete(
		connection, &table, Activity_State(SLOT_CLOSE),
		statement.Return_States[HANDLE_FIELD], driver_status,
	)
	*statement = Statement{}
	return status
}

// Pool_Begin pins one lease behind a transaction.
func Pool_Begin(
	pool *Pool, options driver.Transaction_Options, transaction *Transaction,
) (status Status) {
	defer func() { Status_Invariants(status, "pool_begin.status") }()
	Pool_Invariants(pool, "pool_begin.pool")
	driver.Transaction_Options_Invariants(options, "pool_begin.options")
	Transaction_Invariants(*transaction, "pool_begin.transaction")
	connection, status := Pool_Connection_Acquire(pool)
	if status != STATUS_OK {
		return status
	}
	return Status(connection_begin(
		&connection, options, Connection_Return_State(SLOT_IDLE), transaction,
	))
}

// Connection_Begin pins explicit lease behind a transaction.
func Connection_Begin(
	connection *Connection, options driver.Transaction_Options,
	transaction *Transaction,
) (status Operation_Status) {
	defer func() { Operation_Status_Invariants(status, "connection_begin.status") }()
	Connection_Invariants(*connection, "connection_begin.connection")
	driver.Transaction_Options_Invariants(options, "connection_begin.options")
	Transaction_Invariants(*transaction, "connection_begin.transaction")
	return connection_begin(
		connection, options, Connection_Return_State(SLOT_CONNECTION), transaction,
	)
}

// Transaction_Exec executes one request inside transaction.
func Transaction_Exec(
	transaction *Transaction, request driver.Request, result *Result,
) (status Operation_Status) {
	defer func() { Operation_Status_Invariants(status, "transaction_exec.status") }()
	Transaction_Invariants(*transaction, "transaction_exec.transaction")
	driver.Request_Invariants(request, "transaction_exec.request")
	Result_Invariants(*result, "transaction_exec.result")
	connection := &transaction.Connections[HANDLE_FIELD]
	var table driver.Connection
	reservation_status := connection_reserve(
		connection, Reservation_State(SLOT_TRANSACTION),
		Activity_State(SLOT_EXECUTION), &table,
	)
	if reservation_status != Reservation_Status(STATUS_OK) {
		return Operation_Status(reservation_status)
	}
	*result = Result{}
	driver_status := driver.Connection_Exec(
		&table, request, &result.Driver_Results[HANDLE_FIELD],
	)
	status = operation_complete(
		connection, &table, Activity_State(SLOT_EXECUTION),
		Return_State(SLOT_TRANSACTION), driver_status,
	)
	return status
}

// Transaction_Query opens rows inside transaction.
func Transaction_Query(
	transaction *Transaction, request driver.Request, rows *Rows,
) (status Operation_Status) {
	defer func() { Operation_Status_Invariants(status, "transaction_query.status") }()
	Transaction_Invariants(*transaction, "transaction_query.transaction")
	driver.Request_Invariants(request, "transaction_query.request")
	Rows_Invariants(*rows, "transaction_query.rows")
	connection := &transaction.Connections[HANDLE_FIELD]
	var table driver.Connection
	reservation_status := connection_reserve(
		connection, Reservation_State(SLOT_TRANSACTION),
		Activity_State(SLOT_EXECUTION), &table,
	)
	if reservation_status != Reservation_Status(STATUS_OK) {
		return Operation_Status(reservation_status)
	}
	*rows = rows_of(connection, Return_State(SLOT_TRANSACTION))
	driver_status := driver.Connection_Query(
		&table, request, &rows.Driver_Rows[HANDLE_FIELD],
	)
	if driver_status != driver.STATUS_OK {
		status = operation_complete(
			connection, &table, Activity_State(SLOT_EXECUTION),
			Return_State(SLOT_TRANSACTION), driver_status,
		)
		*rows = Rows{}
		return status
	}
	transition_status := connection_transition(
		connection, Activity_State(SLOT_EXECUTION), TRANSITION_ROWS,
		Return_State(SLOT_IDLE),
	)
	if transition_status != Transition_Status(STATUS_OK) {
		driver.Rows_Close(&rows.Driver_Rows[HANDLE_FIELD])
		*rows = Rows{}
		return Operation_Status(transition_status)
	}
	return Operation_Status(STATUS_OK)
}

// Transaction_Commit makes transaction durable and releases owning lease.
func Transaction_Commit(transaction *Transaction) (status Operation_Status) {
	defer func() { Operation_Status_Invariants(status, "transaction_commit.status") }()
	Transaction_Invariants(*transaction, "transaction_commit.transaction")
	return transaction_finish(transaction, TRANSACTION_COMMIT)
}

// Transaction_Rollback abandons transaction and releases owning lease.
func Transaction_Rollback(transaction *Transaction) (status Operation_Status) {
	defer func() { Operation_Status_Invariants(status, "transaction_rollback.status") }()
	Transaction_Invariants(*transaction, "transaction_rollback.transaction")
	return transaction_finish(transaction, TRANSACTION_ROLLBACK)
}

func connection_probe(
	connection *Connection, return_state Connection_Return_State,
) (status Operation_Status) {
	defer func() {
		Operation_Status_Invariants(status, "connection_probe_internal.status")
	}()
	Connection_Invariants(*connection, "connection_probe_internal.connection")
	Connection_Return_State_Invariants(
		return_state, "connection_probe_internal.return_state",
	)
	var table driver.Connection
	reservation_status := connection_reserve(
		connection, Reservation_State(SLOT_CONNECTION),
		Activity_State(SLOT_EXECUTION), &table,
	)
	if reservation_status != Reservation_Status(STATUS_OK) {
		return Operation_Status(reservation_status)
	}
	driver_status := driver.Connection_Probe(&table)
	return operation_complete(
		connection, &table, Activity_State(SLOT_EXECUTION),
		Return_State(return_state), driver_status,
	)
}

func connection_exec(
	connection *Connection, request driver.Request,
	return_state Connection_Return_State,
	result *Result,
) (status Operation_Status) {
	defer func() {
		Operation_Status_Invariants(status, "connection_exec_internal.status")
	}()
	Connection_Invariants(*connection, "connection_exec_internal.connection")
	driver.Request_Invariants(request, "connection_exec_internal.request")
	Connection_Return_State_Invariants(
		return_state, "connection_exec_internal.return_state",
	)
	Result_Invariants(*result, "connection_exec_internal.result")
	var table driver.Connection
	reservation_status := connection_reserve(
		connection, Reservation_State(SLOT_CONNECTION),
		Activity_State(SLOT_EXECUTION), &table,
	)
	if reservation_status != Reservation_Status(STATUS_OK) {
		return Operation_Status(reservation_status)
	}
	*result = Result{}
	driver_status := driver.Connection_Exec(
		&table, request, &result.Driver_Results[HANDLE_FIELD],
	)
	status = operation_complete(
		connection, &table, Activity_State(SLOT_EXECUTION),
		Return_State(return_state), driver_status,
	)
	return status
}

func connection_query(
	connection *Connection, request driver.Request,
	return_state Connection_Return_State,
	rows *Rows,
) (status Operation_Status) {
	defer func() {
		Operation_Status_Invariants(status, "connection_query_internal.status")
	}()
	Connection_Invariants(*connection, "connection_query_internal.connection")
	driver.Request_Invariants(request, "connection_query_internal.request")
	Connection_Return_State_Invariants(
		return_state, "connection_query_internal.return_state",
	)
	Rows_Invariants(*rows, "connection_query_internal.rows")
	var table driver.Connection
	reservation_status := connection_reserve(
		connection, Reservation_State(SLOT_CONNECTION),
		Activity_State(SLOT_EXECUTION), &table,
	)
	if reservation_status != Reservation_Status(STATUS_OK) {
		return Operation_Status(reservation_status)
	}
	*rows = rows_of(connection, Return_State(return_state))
	driver_status := driver.Connection_Query(
		&table, request, &rows.Driver_Rows[HANDLE_FIELD],
	)
	if driver_status != driver.STATUS_OK {
		status = operation_complete(
			connection, &table, Activity_State(SLOT_EXECUTION),
			Return_State(return_state), driver_status,
		)
		*rows = Rows{}
		return status
	}
	transition_status := connection_transition(
		connection, Activity_State(SLOT_EXECUTION), TRANSITION_ROWS,
		Return_State(SLOT_IDLE),
	)
	if transition_status != Transition_Status(STATUS_OK) {
		driver.Rows_Close(&rows.Driver_Rows[HANDLE_FIELD])
		*rows = Rows{}
		return Operation_Status(transition_status)
	}
	return Operation_Status(STATUS_OK)
}

func connection_prepare(
	connection *Connection, query driver.Query,
	return_state Connection_Return_State,
	statement *Statement,
) (status Operation_Status) {
	defer func() {
		Operation_Status_Invariants(status, "connection_prepare_internal.status")
	}()
	Connection_Invariants(*connection, "connection_prepare_internal.connection")
	driver.Query_Invariants(query, "connection_prepare_internal.query")
	Connection_Return_State_Invariants(
		return_state, "connection_prepare_internal.return_state",
	)
	Statement_Invariants(*statement, "connection_prepare_internal.statement")
	var table driver.Connection
	reservation_status := connection_reserve(
		connection, Reservation_State(SLOT_CONNECTION),
		Activity_State(SLOT_EXECUTION), &table,
	)
	if reservation_status != Reservation_Status(STATUS_OK) {
		return Operation_Status(reservation_status)
	}
	*statement = statement_of(connection, return_state)
	driver_status := driver.Connection_Prepare(
		&table, query, &statement.Driver_Statements[HANDLE_FIELD],
	)
	if driver_status != driver.STATUS_OK {
		status = operation_complete(
			connection, &table, Activity_State(SLOT_EXECUTION),
			Return_State(return_state), driver_status,
		)
		*statement = Statement{}
		return status
	}
	transition_status := connection_transition(
		connection, Activity_State(SLOT_EXECUTION), TRANSITION_RETURN,
		Return_State(SLOT_STATEMENT),
	)
	if transition_status != Transition_Status(STATUS_OK) {
		driver.Statement_Close(&statement.Driver_Statements[HANDLE_FIELD])
		*statement = Statement{}
		return Operation_Status(transition_status)
	}
	return Operation_Status(STATUS_OK)
}

func connection_begin(
	connection *Connection, options driver.Transaction_Options,
	return_state Connection_Return_State, transaction *Transaction,
) (status Operation_Status) {
	defer func() {
		Operation_Status_Invariants(status, "connection_begin_internal.status")
	}()
	Connection_Invariants(*connection, "connection_begin_internal.connection")
	driver.Transaction_Options_Invariants(options, "connection_begin_internal.options")
	Connection_Return_State_Invariants(
		return_state, "connection_begin_internal.return_state",
	)
	Transaction_Invariants(*transaction, "connection_begin_internal.transaction")
	var table driver.Connection
	reservation_status := connection_reserve(
		connection, Reservation_State(SLOT_CONNECTION),
		Activity_State(SLOT_EXECUTION), &table,
	)
	if reservation_status != Reservation_Status(STATUS_OK) {
		return Operation_Status(reservation_status)
	}
	*transaction = transaction_of(connection, return_state)
	driver_status := driver.Connection_Begin(
		&table, options, &transaction.Driver_Transactions[HANDLE_FIELD],
	)
	if driver_status != driver.STATUS_OK {
		status = operation_complete(
			connection, &table, Activity_State(SLOT_EXECUTION),
			Return_State(return_state), driver_status,
		)
		*transaction = Transaction{}
		return status
	}
	transition_status := connection_transition(
		connection, Activity_State(SLOT_EXECUTION), TRANSITION_RETURN,
		Return_State(SLOT_TRANSACTION),
	)
	if transition_status != Transition_Status(STATUS_OK) {
		driver.Transaction_Rollback(&transaction.Driver_Transactions[HANDLE_FIELD])
		*transaction = Transaction{}
		return Operation_Status(transition_status)
	}
	return Operation_Status(STATUS_OK)
}

func transaction_finish(
	transaction *Transaction, terminal Transaction_Terminal,
) (status Operation_Status) {
	defer func() { Operation_Status_Invariants(status, "transaction_finish.status") }()
	Transaction_Invariants(*transaction, "transaction_finish.transaction")
	Transaction_Terminal_Invariants(terminal, "transaction_finish.terminal")
	connection := &transaction.Connections[HANDLE_FIELD]
	var table driver.Connection
	reservation_status := connection_reserve(
		connection, Reservation_State(SLOT_TRANSACTION), Activity_State(SLOT_CLOSE),
		&table,
	)
	if reservation_status != Reservation_Status(STATUS_OK) {
		return Operation_Status(reservation_status)
	}
	driver_status := driver.STATUS_OK
	if terminal == TRANSACTION_COMMIT {
		driver_status = driver.Transaction_Commit(
			&transaction.Driver_Transactions[HANDLE_FIELD],
		)
	} else {
		driver_status = driver.Transaction_Rollback(
			&transaction.Driver_Transactions[HANDLE_FIELD],
		)
	}
	status = operation_complete(
		connection, &table, Activity_State(SLOT_CLOSE),
		transaction.Return_States[HANDLE_FIELD], driver_status,
	)
	*transaction = Transaction{}
	return status
}

func connection_reserve(
	connection *Connection, expected Reservation_State, next Activity_State,
	destination *driver.Connection,
) (status Reservation_Status) {
	defer func() {
		Reservation_Status_Invariants(status, "connection_reserve.status")
	}()
	Connection_Invariants(*connection, "connection_reserve.connection")
	Reservation_State_Invariants(expected, "connection_reserve.expected")
	Activity_State_Invariants(next, "connection_reserve.next")
	driver.Connection_Invariants(destination, "connection_reserve.destination")
	pool := connection.Pools[HANDLE_FIELD]
	if pool == nil {
		return Reservation_Status(STATUS_HANDLE_INVALID)
	}
	Pool_Invariants(pool, "connection_reserve.pool")
	pool_initialized(pool)
	index := connection.Slot_Indices[HANDLE_FIELD]
	generation := connection.Generations[HANDLE_FIELD]
	Slot_Index_Invariants(index, "connection_reserve.index")
	Lease_Generation_Invariants(generation, "connection_reserve.generation")
	synchronizer := pool.Synchronizers[POOL_FIELD]
	synchronizer.Lock_Procedure(synchronizer.State)
	if pool.States[POOL_FIELD] == POOL_CLOSED {
		synchronizer.Unlock_Procedure(synchronizer.State)
		return Reservation_Status(STATUS_CLOSED)
	}
	slots := pool.Slot_Sets[POOL_FIELD]
	if int(index) >= len(slots) {
		synchronizer.Unlock_Procedure(synchronizer.State)
		return Reservation_Status(STATUS_HANDLE_INVALID)
	}
	slot := &slots[index]
	if Lease_Generation(slot.Generations[SLOT_FIELD]) != generation {
		synchronizer.Unlock_Procedure(synchronizer.State)
		return Reservation_Status(STATUS_HANDLE_INVALID)
	}
	if Reservation_State(slot.States[SLOT_FIELD]) != expected {
		state := slot.States[SLOT_FIELD]
		synchronizer.Unlock_Procedure(synchronizer.State)
		if state == SLOT_EMPTY {
			return Reservation_Status(STATUS_HANDLE_INVALID)
		}
		if state == SLOT_IDLE {
			return Reservation_Status(STATUS_HANDLE_INVALID)
		}
		return Reservation_Status(STATUS_BUSY)
	}
	*destination = slot.Connections[SLOT_FIELD]
	slot.States[SLOT_FIELD] = Slot_State(next)
	synchronizer.Unlock_Procedure(synchronizer.State)
	return Reservation_Status(STATUS_OK)
}

func connection_transition(
	connection *Connection, expected Activity_State, kind Transition_Kind,
	return_state Return_State,
) (status Transition_Status) {
	defer func() {
		Transition_Status_Invariants(status, "connection_transition.status")
	}()
	Connection_Invariants(*connection, "connection_transition.connection")
	Activity_State_Invariants(expected, "connection_transition.expected")
	Transition_Kind_Invariants(kind, "connection_transition.kind")
	Return_State_Invariants(return_state, "connection_transition.return_state")
	pool := connection.Pools[HANDLE_FIELD]
	if pool == nil {
		return Transition_Status(STATUS_HANDLE_INVALID)
	}
	Pool_Invariants(pool, "connection_transition.pool")
	pool_initialized(pool)
	index := connection.Slot_Indices[HANDLE_FIELD]
	generation := connection.Generations[HANDLE_FIELD]
	Slot_Index_Invariants(index, "connection_transition.index")
	Lease_Generation_Invariants(generation, "connection_transition.generation")
	synchronizer := pool.Synchronizers[POOL_FIELD]
	synchronizer.Lock_Procedure(synchronizer.State)
	slots := pool.Slot_Sets[POOL_FIELD]
	if int(index) >= len(slots) {
		synchronizer.Unlock_Procedure(synchronizer.State)
		return Transition_Status(STATUS_HANDLE_INVALID)
	}
	slot := &slots[index]
	if Lease_Generation(slot.Generations[SLOT_FIELD]) != generation {
		synchronizer.Unlock_Procedure(synchronizer.State)
		return Transition_Status(STATUS_HANDLE_INVALID)
	}
	if Activity_State(slot.States[SLOT_FIELD]) != expected {
		synchronizer.Unlock_Procedure(synchronizer.State)
		return Transition_Status(STATUS_BUSY)
	}
	next := Slot_State(return_state)
	if kind == TRANSITION_ROWS {
		next = SLOT_ROWS
	}
	slot.States[SLOT_FIELD] = next
	synchronizer.Unlock_Procedure(synchronizer.State)
	if next == SLOT_IDLE {
		*connection = Connection{}
	}
	return Transition_Status(STATUS_OK)
}

func connection_discard(
	connection *Connection, expected Discard_State,
) (status Transition_Status) {
	defer func() {
		Transition_Status_Invariants(status, "connection_discard.status")
	}()
	Connection_Invariants(*connection, "connection_discard.connection")
	Discard_State_Invariants(expected, "connection_discard.expected")
	pool := connection.Pools[HANDLE_FIELD]
	if pool == nil {
		return Transition_Status(STATUS_HANDLE_INVALID)
	}
	Pool_Invariants(pool, "connection_discard.pool")
	pool_initialized(pool)
	index := connection.Slot_Indices[HANDLE_FIELD]
	generation := connection.Generations[HANDLE_FIELD]
	Slot_Index_Invariants(index, "connection_discard.index")
	Lease_Generation_Invariants(generation, "connection_discard.generation")
	synchronizer := pool.Synchronizers[POOL_FIELD]
	synchronizer.Lock_Procedure(synchronizer.State)
	slots := pool.Slot_Sets[POOL_FIELD]
	if int(index) >= len(slots) {
		synchronizer.Unlock_Procedure(synchronizer.State)
		return Transition_Status(STATUS_HANDLE_INVALID)
	}
	slot := &slots[index]
	if Lease_Generation(slot.Generations[SLOT_FIELD]) != generation {
		synchronizer.Unlock_Procedure(synchronizer.State)
		return Transition_Status(STATUS_HANDLE_INVALID)
	}
	if Discard_State(slot.States[SLOT_FIELD]) != expected {
		synchronizer.Unlock_Procedure(synchronizer.State)
		return Transition_Status(STATUS_BUSY)
	}
	slot.Connections[SLOT_FIELD] = driver.Connection{}
	slot.States[SLOT_FIELD] = SLOT_EMPTY
	synchronizer.Unlock_Procedure(synchronizer.State)
	*connection = Connection{}
	return Transition_Status(STATUS_OK)
}

func operation_complete(
	connection *Connection, table *driver.Connection, expected Activity_State,
	return_state Return_State, driver_status driver.Status,
) (status Operation_Status) {
	defer func() { Operation_Status_Invariants(status, "operation_complete.status") }()
	Connection_Invariants(*connection, "operation_complete.connection")
	driver.Connection_Invariants(table, "operation_complete.table")
	Activity_State_Invariants(expected, "operation_complete.expected")
	Return_State_Invariants(return_state, "operation_complete.return_state")
	driver.Status_Invariants(driver_status, "operation_complete.driver_status")
	if driver_status == driver.STATUS_BAD_CONNECTION {
		driver.Connection_Close(table)
		transition_status := connection_discard(connection, Discard_State(expected))
		if transition_status != Transition_Status(STATUS_OK) {
			return Operation_Status(transition_status)
		}
		return Operation_Status(STATUS_BAD_CONNECTION)
	}
	transition_status := connection_transition(
		connection, expected, TRANSITION_RETURN, return_state,
	)
	if transition_status != Transition_Status(STATUS_OK) {
		return Operation_Status(transition_status)
	}
	return Operation_Status(status_of_driver(driver_status))
}

func connection_of(
	pool *Pool, index Slot_Index, generation Lease_Generation,
) (connection Connection) {
	defer func() { Connection_Invariants(connection, "connection_of.connection") }()
	Pool_Invariants(pool, "connection_of.pool")
	Slot_Index_Invariants(index, "connection_of.index")
	Lease_Generation_Invariants(generation, "connection_of.generation")
	return Connection{
		Pools:        [HANDLE_FIELD_COUNT]*Pool{pool},
		Slot_Indices: [HANDLE_FIELD_COUNT]Slot_Index{index},
		Generations:  [HANDLE_FIELD_COUNT]Lease_Generation{generation},
	}
}

func rows_of(
	connection *Connection, return_state Return_State,
) (rows Rows) {
	defer func() { Rows_Invariants(rows, "rows_of.rows") }()
	Connection_Invariants(*connection, "rows_of.connection")
	Return_State_Invariants(return_state, "rows_of.return_state")
	return Rows{
		Connections:   [HANDLE_FIELD_COUNT]Connection{*connection},
		Return_States: [HANDLE_FIELD_COUNT]Return_State{return_state},
	}
}

func statement_of(
	connection *Connection, return_state Connection_Return_State,
) (statement Statement) {
	defer func() { Statement_Invariants(statement, "statement_of.statement") }()
	Connection_Invariants(*connection, "statement_of.connection")
	Connection_Return_State_Invariants(return_state, "statement_of.return_state")
	return Statement{
		Connections:   [HANDLE_FIELD_COUNT]Connection{*connection},
		Return_States: [HANDLE_FIELD_COUNT]Return_State{Return_State(return_state)},
	}
}

func transaction_of(
	connection *Connection, return_state Connection_Return_State,
) (transaction Transaction) {
	defer func() { Transaction_Invariants(transaction, "transaction_of.transaction") }()
	Connection_Invariants(*connection, "transaction_of.connection")
	Connection_Return_State_Invariants(return_state, "transaction_of.return_state")
	return Transaction{
		Connections:   [HANDLE_FIELD_COUNT]Connection{*connection},
		Return_States: [HANDLE_FIELD_COUNT]Return_State{Return_State(return_state)},
	}
}

func status_of_driver(driver_status driver.Status) (status Driver_Status) {
	defer func() { Driver_Status_Invariants(status, "status_of_driver.status") }()
	driver.Status_Invariants(driver_status, "status_of_driver.driver_status")
	switch driver_status {
	case driver.STATUS_OK:
		return Driver_Status(STATUS_OK)
	case driver.STATUS_DONE:
		return Driver_Status(STATUS_DONE)
	case driver.STATUS_INPUT_INVALID:
		return Driver_Status(STATUS_INPUT_INVALID)
	case driver.STATUS_STORAGE_INVALID:
		return Driver_Status(STATUS_STORAGE_INVALID)
	case driver.STATUS_ERROR:
		return Driver_Status(STATUS_ERROR)
	case driver.STATUS_BAD_CONNECTION:
		return Driver_Status(STATUS_BAD_CONNECTION)
	case driver.STATUS_UNSUPPORTED:
		return Driver_Status(STATUS_UNSUPPORTED)
	default:
		return Driver_Status(STATUS_ERROR)
	}
}

func pool_initialized(pool *Pool) {
	Pool_Invariants(pool, "pool_initialized.pool")
	driver.Driver_Invariants(pool.Drivers[POOL_FIELD], "pool_initialized.driver")
	driver.Data_Source_Invariants(
		pool.Data_Sources[POOL_FIELD], "pool_initialized.data_source",
	)
	Synchronizer_Invariants(
		pool.Synchronizers[POOL_FIELD], "pool_initialized.synchronizer",
	)
	Initialized_Slot_Storage_Invariants(
		Initialized_Slot_Storage(pool.Slot_Sets[POOL_FIELD]),
		"pool_initialized.slots",
	)
	Initialized_Pool_State_Invariants(
		Initialized_Pool_State(pool.States[POOL_FIELD]), "pool_initialized.state",
	)
	invariant.Always(
		pool.States[POOL_FIELD] != POOL_UNINITIALIZED,
		"An initialized pool has bound dependencies.",
	)
}
