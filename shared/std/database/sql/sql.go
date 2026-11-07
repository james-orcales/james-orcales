// Package sql manages bounded database connections over caller-owned storage.
package sql

import "unsafe"

import "local/james-orcales/shared/std/database/driver"
import "local/james-orcales/shared/std/math/bits"
import "local/james-orcales/shared/std/sim/aver/default"
import "local/james-orcales/shared/std/slices"

// Status keeps pool and operation outcomes scalar and allocation-free.
type Status uint8

// Status_Invariants closes every SQL outcome.
func Status_Invariants(value Status, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint8(
    uint8(value),
    uint8(STATUS_OK),
    uint8(STATUS_GENERATION_EXHAUSTED),
  ).
    Ensure()
}

// Status_Pointer keeps paired operation status caller-owned.
type Status_Pointer *Status

// Status_Pointer_Invariants admits absent optional storage.
func Status_Pointer_Invariants(value Status_Pointer, namespace aver.Namespace) {
  if value == nil {
    return
  }
  Status_Invariants(*value, namespace)
}

// Initialization_Status reports successful initialization or missing slot storage.
type Initialization_Status Status

// Initialization_Status_Invariants keeps initialization outcomes exact.
func Initialization_Status_Invariants(
  value Initialization_Status, namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Enum_Uint8(
    uint8(value),
    uint8(STATUS_OK),
    uint8(STATUS_STORAGE_INVALID),
  ).
    Ensure()
}

// Operation_Status excludes generation exhaustion owned by bounded acquisition.
type Operation_Status Status

// Operation_Status_Invariants bounds connection-backed operation outcomes.
func Operation_Status_Invariants(
  value Operation_Status, namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint8(uint8(value), uint8(STATUS_OK), uint8(STATUS_BUSY)).
    Ensure()
}

// Reservation_Status reports one slot reservation decision.
type Reservation_Status Status

// Reservation_Status_Invariants keeps reservation outcomes exact.
func Reservation_Status_Invariants(
  value Reservation_Status, namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Enum_4_Uint8(
    uint8(value),
    uint8(STATUS_OK),
    uint8(STATUS_CLOSED),
    uint8(STATUS_HANDLE_INVALID),
    uint8(STATUS_BUSY),
  ).
    Ensure()
}

// Transition_Status reports one reserved slot transition decision.
type Transition_Status Status

// Transition_Status_Invariants keeps transition outcomes exact.
func Transition_Status_Invariants(
  value Transition_Status, namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Enum_3_Uint8(
    uint8(value),
    uint8(STATUS_OK),
    uint8(STATUS_HANDLE_INVALID),
    uint8(STATUS_BUSY),
  ).
    Ensure()
}

// Driver_Status is one driver outcome mapped into SQL status values.
type Driver_Status Status

// Driver_Status_Invariants excludes pool-owned outcomes.
func Driver_Status_Invariants(value Driver_Status, namespace aver.Namespace) {
  aver.Tree(value, namespace).
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

// CONNECTION_COUNT_MAXIMUM shares repository collection capacity.
const CONNECTION_COUNT_MAXIMUM = slices.SLICE_COUNT_MAXIMUM

// Connection_Count is one bounded pool quantity.
type Connection_Count int

// Connection_Count_Invariants bounds each pool quantity by fixed capacity.
func Connection_Count_Invariants(value Connection_Count, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(int(value), slices.COUNT_MINIMUM, CONNECTION_COUNT_MAXIMUM).
    Ensure()
}

// Capacity_Count is fixed caller slot count.
type Capacity_Count Connection_Count

// Capacity_Count_Invariants bounds fixed caller capacity.
func Capacity_Count_Invariants(value Capacity_Count, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(int(value), slices.COUNT_MINIMUM, CONNECTION_COUNT_MAXIMUM).
    Ensure()
}

// Open_Count counts slots holding driver connections.
type Open_Count Connection_Count

// Open_Count_Invariants bounds live driver connections.
func Open_Count_Invariants(value Open_Count, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(int(value), slices.COUNT_MINIMUM, CONNECTION_COUNT_MAXIMUM).
    Ensure()
}

// Idle_Count counts reusable driver connections.
type Idle_Count Connection_Count

// Idle_Count_Invariants bounds reusable driver connections.
func Idle_Count_Invariants(value Idle_Count, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(int(value), slices.COUNT_MINIMUM, CONNECTION_COUNT_MAXIMUM).
    Ensure()
}

// In_Use_Count counts leased driver connections.
type In_Use_Count Connection_Count

// In_Use_Count_Invariants bounds leased driver connections.
func In_Use_Count_Invariants(value In_Use_Count, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(int(value), slices.COUNT_MINIMUM, CONNECTION_COUNT_MAXIMUM).
    Ensure()
}

// SLOT_INDEX_MAXIMUM is final index in maximum bounded slot storage.
const SLOT_INDEX_MAXIMUM = CONNECTION_COUNT_MAXIMUM - 1

// Slot_Index identifies one caller-owned pool slot.
type Slot_Index int

// Slot_Index_Invariants bounds slot addressing.
func Slot_Index_Invariants(value Slot_Index, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(int(value), slices.COUNT_MINIMUM, SLOT_INDEX_MAXIMUM).
    Ensure()
}

// Generation prevents a stale handle from naming a reused slot.
type Generation uint64

// Generation_Invariants keeps complete finite lease identity domain.
func Generation_Invariants(value Generation, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint64(
    uint64(value),
    bits.WORD_64_MINIMUM,
    bits.WORD_64_MAXIMUM,
  ).
    Ensure()
}

// LEASE_GENERATION_MINIMUM reserves zero for empty handles and untouched slots.
const LEASE_GENERATION_MINIMUM = Generation(bits.WORD_64_MINIMUM) + 1

// Lease_Generation is one nonzero generation carried by a live handle.
type Lease_Generation Generation

// Lease_Generation_Invariants excludes the zero sentinel.
func Lease_Generation_Invariants(
  value Lease_Generation, namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Uint64(
    uint64(value),
    uint64(LEASE_GENERATION_MINIMUM),
    bits.WORD_64_MAXIMUM,
  ).
    Ensure()
}

// Slot_State is one state of the bounded pool slot machine.
type Slot_State uint8

// Slot_State_Invariants closes pool slot lifecycle.
func Slot_State_Invariants(value Slot_State, namespace aver.Namespace) {
  aver.Tree(value, namespace).
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
  value Reservation_State, namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Enum_4_Uint8(
    uint8(value),
    uint8(SLOT_CONNECTION),
    uint8(SLOT_ROWS),
    uint8(SLOT_STATEMENT),
    uint8(SLOT_TRANSACTION),
  ).
    Ensure()
}

// Activity_State is one exclusive driver-call state.
type Activity_State Slot_State

// Activity_State_Invariants closes active slot states.
func Activity_State_Invariants(value Activity_State, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_3_Uint8(
    uint8(value),
    uint8(SLOT_EXECUTION),
    uint8(SLOT_ROWS_READING),
    uint8(SLOT_CLOSE),
  ).
    Ensure()
}

// Discard_State is one driver-call state that can report a bad connection.
type Discard_State Slot_State

// Discard_State_Invariants closes discardable active states.
func Discard_State_Invariants(value Discard_State, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_3_Uint8(
    uint8(value),
    uint8(SLOT_EXECUTION),
    uint8(SLOT_ROWS_READING),
    uint8(SLOT_CLOSE),
  ).
    Ensure()
}

// SLOT_SIZE keeps caller-owned slot representation equal to explicit fields.
const SLOT_SIZE = unsafe.Sizeof(struct {
  Connection driver.Connection
  State      Slot_State
  Generation Generation
}{})

// Slot is caller-owned storage for one driver connection and lease identity.
type Slot struct {
  // Connection keeps driver resource in caller-owned storage.
  Connection driver.Connection
  // State prevents overlapping resource owners.
  State Slot_State
  // Generation prevents stale handle reuse.
  Generation Generation
}

// Slot_Invariants preserves storage while pool operations validate live fields.
func Slot_Invariants(value *Slot, namespace aver.Namespace) {
  driver.Connection_Invariants(&value.Connection, namespace)
  Slot_State_Invariants(value.State, namespace)
  Generation_Invariants(value.Generation, namespace)
  aver.Always(
    unsafe.Sizeof(*value) == SLOT_SIZE,
    "Explicit fields preserve pool slot storage size.",
  )
}

// Slot_Storage is bounded caller-owned pool capacity.
type Slot_Storage []Slot

// Slot_Storage_Invariants bounds fixed pool capacity.
func Slot_Storage_Invariants(value Slot_Storage, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(len(value), slices.COUNT_MINIMUM, CONNECTION_COUNT_MAXIMUM).
    Ensure()
}

// INITIALIZED_CONNECTION_COUNT_MINIMUM excludes rejected empty storage.
const INITIALIZED_CONNECTION_COUNT_MINIMUM = slices.COUNT_MINIMUM + 1

// Initialized_Slot_Storage is bound nonempty pool capacity.
type Initialized_Slot_Storage Slot_Storage

// Initialized_Slot_Storage_Invariants excludes rejected empty storage.
func Initialized_Slot_Storage_Invariants(
  value Initialized_Slot_Storage, namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Range_Int(
    len(value),
    INITIALIZED_CONNECTION_COUNT_MINIMUM,
    CONNECTION_COUNT_MAXIMUM,
  ).
    Ensure()
}

// Pool_Closed keeps zero value open so dependencies alone mark initialization.
type Pool_Closed uint8

// Pool_Closed_Invariants requires open and closed lifecycle observations.
func Pool_Closed_Invariants(value Pool_Closed, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_Uint8(uint8(value), uint8(POOL_OPEN), uint8(POOL_CLOSED)).
    Ensure()
}

// POOL_OPEN admits connection work.
const POOL_OPEN Pool_Closed = Pool_Closed(bits.WORD_8_MINIMUM)

// POOL_CLOSED refuses new connection work.
const POOL_CLOSED Pool_Closed = POOL_OPEN + 1

// POOL_SIZE keeps pool representation equal to injected driver and caller storage.
const POOL_SIZE = unsafe.Sizeof(struct {
  Driver      driver.Driver
  Data_Source driver.Data_Source
  Slots       Slot_Storage
  Closed      Pool_Closed
}{})

// Pool owns no dynamic storage; every connection slot belongs to caller.
type Pool struct {
  // Driver keeps resource creation injected.
  Driver driver.Driver
  // Data_Source remains validated before connection creation.
  Data_Source driver.Data_Source
  // Slots keeps capacity caller-owned and bounded.
  Slots Slot_Storage
  // Closed prevents acquisition after terminal resource release.
  Closed Pool_Closed
}

// Pool_Invariants preserves storage while pool_initialized validates dependencies.
func Pool_Invariants(value *Pool, namespace aver.Namespace) {
  driver.Driver_Invariants(&value.Driver, namespace)
  driver.Data_Source_Invariants(value.Data_Source, namespace)
  Slot_Storage_Invariants(value.Slots, namespace)
  Pool_Closed_Invariants(value.Closed, namespace)
  aver.Always(
    unsafe.Sizeof(*value) == POOL_SIZE,
    "Explicit fields preserve pool dependency storage size.",
  )
}

// Pool_Pointer names caller-owned Pool storage.
type Pool_Pointer *Pool

// Pool_Pointer_Invariants admits absent optional storage.
func Pool_Pointer_Invariants(value Pool_Pointer, namespace aver.Namespace) {
  if value == nil {
    return
  }
  Pool_Invariants(value, namespace)
}

// Pool_Destination names caller-owned storage before initialization.
type Pool_Destination *Pool

// Pool_Destination_Invariants admits absent optional storage.
func Pool_Destination_Invariants(
  value Pool_Destination, namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Pool_Invariants(value, namespace)
}

// Open_Pool excludes empty and closed pool states proven before internal work.
type Open_Pool Pool

// Open_Pool_Invariants preserves proof needed by internal pool operations.
func Open_Pool_Invariants(value *Open_Pool, namespace aver.Namespace) {
  driver.Driver_Invariants(&value.Driver, namespace)
  driver.Data_Source_Invariants(value.Data_Source, namespace)
  aver.Tree(*value, namespace).
    Range_Int(
    len(value.Slots),
    INITIALIZED_CONNECTION_COUNT_MINIMUM,
    CONNECTION_COUNT_MAXIMUM,
  ).
    Ensure()
  aver.Always(value.Closed == POOL_OPEN, "Open SQL Pool accepts work.")
  aver.Always(
    unsafe.Sizeof(*value) == POOL_SIZE,
    "Open SQL Pool preserves dependency storage size.",
  )
}

// Open_Pool_Pointer names proven open storage without copying Pool.
type Open_Pool_Pointer *Open_Pool

// Open_Pool_Pointer_Invariants admits absent optional storage.
func Open_Pool_Pointer_Invariants(
  value Open_Pool_Pointer, namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Open_Pool_Invariants(value, namespace)
}

// CONNECTION_SIZE keeps lease representation equal to owner, index, and generation.
const CONNECTION_SIZE = unsafe.Sizeof(struct {
  Pool       unsafe.Pointer
  Slot_Index Slot_Index
  Generation Generation
}{})

// Connection is one generation-checked pool lease.
type Connection struct {
  // Pool stays opaque so zero handle remains valid storage.
  Pool unsafe.Pointer
  // Slot_Index identifies caller-owned connection storage.
  Slot_Index Slot_Index
  // Generation leaves zero available for empty handle.
  Generation Generation
}

// Connection_Invariants preserves storage while reservation validates lease identity.
func Connection_Invariants(value *Connection, namespace aver.Namespace) {
  Slot_Index_Invariants(value.Slot_Index, namespace)
  Generation_Invariants(value.Generation, namespace)
  aver.Always(
    unsafe.Sizeof(*value) == CONNECTION_SIZE,
    "Explicit fields preserve connection handle storage size.",
  )
}

// Connection_Pointer names caller-owned Connection storage.
type Connection_Pointer *Connection

// Connection_Pointer_Invariants admits absent optional storage.
func Connection_Pointer_Invariants(
  value Connection_Pointer, namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Connection_Invariants(value, namespace)
}

// Live_Connection is one validated nonzero lease identity.
type Live_Connection Connection

// Live_Connection_Invariants excludes empty generation sentinel.
func Live_Connection_Invariants(
  value *Live_Connection, namespace aver.Namespace,
) {
  aver.Tree(*value, namespace).
    Range_Int(
    int(value.Slot_Index),
    slices.COUNT_MINIMUM,
    SLOT_INDEX_MAXIMUM,
  ).
    Range_Uint64(
    uint64(value.Generation),
    uint64(LEASE_GENERATION_MINIMUM),
    bits.WORD_64_MAXIMUM,
  ).
    Ensure()
  aver.Always(
    unsafe.Sizeof(*value) == CONNECTION_SIZE,
    "Live SQL Connection preserves handle storage size.",
  )
}

// Live_Connection_Pointer names validated lease storage without copying it.
type Live_Connection_Pointer *Live_Connection

// Live_Connection_Pointer_Invariants admits absent optional storage.
func Live_Connection_Pointer_Invariants(
  value Live_Connection_Pointer, namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Live_Connection_Invariants(value, namespace)
}

// Return_State is state restored after a dependent resource closes.
type Return_State Slot_State

// Return_State_Invariants permits pool, connection, statement, and transaction owners.
func Return_State_Invariants(value Return_State, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_4_Uint8(
    uint8(value),
    uint8(SLOT_IDLE),
    uint8(SLOT_CONNECTION),
    uint8(SLOT_STATEMENT),
    uint8(SLOT_TRANSACTION),
  ).
    Ensure()
}

// Connection_Return_State restores an internal operation to pool or explicit lease.
type Connection_Return_State Return_State

// Connection_Return_State_Invariants closes connection operation ownership.
func Connection_Return_State_Invariants(
  value Connection_Return_State, namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Enum_Uint8(
    uint8(value), uint8(SLOT_IDLE), uint8(SLOT_CONNECTION),
  ).
    Ensure()
}

// Transition_Kind selects rows ownership or one explicit return state.
type Transition_Kind uint8

// Transition_Kind_Invariants closes transition target representation.
func Transition_Kind_Invariants(value Transition_Kind, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_Uint8(
    uint8(value),
    uint8(TRANSITION_RETURN),
    uint8(TRANSITION_ROWS),
  ).
    Ensure()
}

// TRANSITION_RETURN restores one Return_State.
const TRANSITION_RETURN Transition_Kind = Transition_Kind(bits.WORD_8_MINIMUM)

// TRANSITION_ROWS transfers ownership to one rows cursor.
const TRANSITION_ROWS Transition_Kind = TRANSITION_RETURN + 1

// ROWS_SIZE keeps cursor handle representation equal to explicit fields.
const ROWS_SIZE = unsafe.Sizeof(struct {
  Connection   Connection
  Driver_Rows  driver.Rows
  Return_State Slot_State
}{})

// Rows carries one driver cursor and reserved pool lease.
type Rows struct {
  // Connection preserves reserved slot identity.
  Connection Connection
  // Driver_Rows keeps cursor state caller-owned.
  Driver_Rows driver.Rows
  // Return_State leaves zero available for empty handle.
  Return_State Slot_State
}

// Rows_Invariants preserves storage while rows operations validate live fields.
func Rows_Invariants(value *Rows, namespace aver.Namespace) {
  Connection_Invariants(&value.Connection, namespace)
  driver.Rows_Invariants(&value.Driver_Rows, namespace)
  Slot_State_Invariants(value.Return_State, namespace)
  aver.Always(
    unsafe.Sizeof(*value) == ROWS_SIZE,
    "Explicit fields preserve SQL rows storage size.",
  )
}

// Rows_Pointer names caller-owned Rows storage.
type Rows_Pointer *Rows

// Rows_Pointer_Invariants admits absent optional storage.
func Rows_Pointer_Invariants(value Rows_Pointer, namespace aver.Namespace) {
  if value == nil {
    return
  }
  Rows_Invariants(value, namespace)
}

// STATEMENT_SIZE keeps prepared handle representation equal to explicit fields.
const STATEMENT_SIZE = unsafe.Sizeof(struct {
  Connection       Connection
  Driver_Statement driver.Statement
  Return_State     Slot_State
}{})

// Statement carries one driver statement and reserved pool lease.
type Statement struct {
  // Connection preserves reserved slot identity.
  Connection Connection
  // Driver_Statement keeps prepared state caller-owned.
  Driver_Statement driver.Statement
  // Return_State leaves zero available for empty handle.
  Return_State Slot_State
}

// Statement_Invariants preserves storage while operations validate live fields.
func Statement_Invariants(value *Statement, namespace aver.Namespace) {
  Connection_Invariants(&value.Connection, namespace)
  driver.Statement_Invariants(&value.Driver_Statement, namespace)
  Slot_State_Invariants(value.Return_State, namespace)
  aver.Always(
    unsafe.Sizeof(*value) == STATEMENT_SIZE,
    "Explicit fields preserve SQL statement storage size.",
  )
}

// Statement_Pointer names caller-owned Statement storage.
type Statement_Pointer *Statement

// Statement_Pointer_Invariants admits absent optional storage.
func Statement_Pointer_Invariants(
  value Statement_Pointer, namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Statement_Invariants(value, namespace)
}

// TRANSACTION_SIZE keeps transaction handle representation equal to explicit fields.
const TRANSACTION_SIZE = unsafe.Sizeof(struct {
  Connection         Connection
  Driver_Transaction driver.Transaction
  Return_State       Slot_State
}{})

// Transaction carries one driver transaction and reserved pool lease.
type Transaction struct {
  // Connection preserves reserved slot identity.
  Connection Connection
  // Driver_Transaction keeps terminal state caller-owned.
  Driver_Transaction driver.Transaction
  // Return_State leaves zero available for empty handle.
  Return_State Slot_State
}

// Transaction_Invariants preserves storage while operations validate live fields.
func Transaction_Invariants(value *Transaction, namespace aver.Namespace) {
  Connection_Invariants(&value.Connection, namespace)
  driver.Transaction_Invariants(&value.Driver_Transaction, namespace)
  Slot_State_Invariants(value.Return_State, namespace)
  aver.Always(
    unsafe.Sizeof(*value) == TRANSACTION_SIZE,
    "Explicit fields preserve SQL transaction storage size.",
  )
}

// Transaction_Pointer names caller-owned Transaction storage.
type Transaction_Pointer *Transaction

// Transaction_Pointer_Invariants admits absent optional storage.
func Transaction_Pointer_Invariants(
  value Transaction_Pointer, namespace aver.Namespace,
) {
  if value == nil {
    return
  }
  Transaction_Invariants(value, namespace)
}

// Transaction_Terminal selects commit or rollback.
type Transaction_Terminal uint8

// Transaction_Terminal_Invariants closes terminal operation choice.
func Transaction_Terminal_Invariants(
  value Transaction_Terminal, namespace aver.Namespace,
) {
  aver.Tree(value, namespace).
    Enum_Uint8(
    uint8(value),
    uint8(TRANSACTION_ROLLBACK),
    uint8(TRANSACTION_COMMIT),
  ).
    Ensure()
}

// TRANSACTION_ROLLBACK abandons transaction changes.
const TRANSACTION_ROLLBACK Transaction_Terminal = Transaction_Terminal(bits.WORD_8_MINIMUM)

// TRANSACTION_COMMIT makes transaction changes durable.
const TRANSACTION_COMMIT Transaction_Terminal = TRANSACTION_ROLLBACK + 1

// RESULT_SIZE keeps SQL result representation equal to driver result.
const RESULT_SIZE = unsafe.Sizeof(struct {
  Driver_Result driver.Result
}{})

// Result carries one driver result without interface allocation.
type Result struct {
  // Driver_Result avoids interface allocation.
  Driver_Result driver.Result
}

// Result_Invariants preserves storage while accessors validate optional counters.
func Result_Invariants(value *Result, namespace aver.Namespace) {
  driver.Result_Invariants(&value.Driver_Result, namespace)
  aver.Always(
    unsafe.Sizeof(*value) == RESULT_SIZE,
    "Explicit field preserves SQL result storage size.",
  )
}

// Result_Pointer names caller-owned Result storage.
type Result_Pointer *Result

// Result_Pointer_Invariants admits absent optional storage.
func Result_Pointer_Invariants(value Result_Pointer, namespace aver.Namespace) {
  if value == nil {
    return
  }
  Result_Invariants(value, namespace)
}

// STATISTICS_SIZE keeps snapshot representation equal to four explicit counts.
const STATISTICS_SIZE = unsafe.Sizeof(struct {
  Capacity     Capacity_Count
  Open_Count   Open_Count
  Idle_Count   Idle_Count
  In_Use_Count In_Use_Count
}{})

// Statistics is one bounded snapshot of pool slot counts.
type Statistics struct {
  // Capacity preserves fixed caller capacity.
  Capacity Capacity_Count
  // Open_Count excludes untouched storage.
  Open_Count Open_Count
  // Idle_Count identifies reusable connections.
  Idle_Count Idle_Count
  // In_Use_Count excludes empty, idle, and in-flight creation storage.
  In_Use_Count In_Use_Count
}

// Statistics_Invariants preserves storage while accessors validate each count.
func Statistics_Invariants(value *Statistics, namespace aver.Namespace) {
  Capacity_Count_Invariants(value.Capacity, namespace)
  Open_Count_Invariants(value.Open_Count, namespace)
  Idle_Count_Invariants(value.Idle_Count, namespace)
  In_Use_Count_Invariants(value.In_Use_Count, namespace)
  aver.Always(
    unsafe.Sizeof(*value) == STATISTICS_SIZE,
    "Explicit counts preserve pool statistics storage size.",
  )
}

// Pool_Init binds injected driver to fixed caller-owned slots.
func Pool_Init(
  Pool Pool_Destination,
  injected driver.Driver,
  data_source driver.Data_Source,
  Slots Slot_Storage,
) (Status Initialization_Status) {
  defer func() {
    Initialization_Status_Invariants(Status, "pool_init.status")
  }()
  Pool_Destination_Invariants(Pool, "pool_init.pool")
  Pool_Invariants(Pool, "pool_init.pool_value")
  driver.Driver_Invariants(&injected, "pool_init.injected")
  driver.Data_Source_Invariants(data_source, "pool_init.data_source")
  Slot_Storage_Invariants(Slots, "pool_init.slots")
  if len(Slots) == slices.COUNT_MINIMUM {
    return Initialization_Status(STATUS_STORAGE_INVALID)
  }
  pool_driver_live(injected)
  for index := range Slots {
    Slot_Invariants(&Slots[index], "pool_init.slot")
    Slots[index] = Slot{}
  }
  *Pool = Pool{
    Driver:      injected,
    Data_Source: data_source,
    Slots:       Slots,
    Closed:      POOL_OPEN,
  }
  return Initialization_Status(STATUS_OK)
}

// Pool_Connection_Acquire reserves idle storage or opens one empty slot.
func Pool_Connection_Acquire(
  Pool Pool_Pointer, Status Status_Pointer,
) (Connection Connection) {
  defer func() {
    Connection_Invariants(&Connection, "pool_connection_acquire.connection")
  }()
  Pool_Pointer_Invariants(Pool, "pool_connection_acquire.pool")
  Status_Pointer_Invariants(Status, "pool_connection_acquire.status_storage")
  defer func() {
    Status_Pointer_Invariants(Status, "pool_connection_acquire.status")
  }()
  pool_initialized(Pool)
  open_pool := (*Open_Pool)((*Pool)(Pool))
  if Pool.Closed == POOL_CLOSED {
    *Status = STATUS_CLOSED
    return Connection{}
  }
  Slots := Pool.Slots
  generation_exhausted := false
  for index := range Slots {
    slot := &Slots[index]
    if slot.State != SLOT_IDLE {
      continue
    }
    if slot.Generation == Generation(bits.WORD_64_MAXIMUM) {
      generation_exhausted = true
      continue
    }
    slot.Generation++
    slot.State = SLOT_CONNECTION
    Connection = Connection(
      connection_of(
        open_pool, Slot_Index(index), Lease_Generation(slot.Generation),
      ),
    )
    *Status = STATUS_OK
    return Connection
  }
  for index := range Slots {
    slot := &Slots[index]
    if slot.State != SLOT_EMPTY {
      continue
    }
    if slot.Generation == Generation(bits.WORD_64_MAXIMUM) {
      generation_exhausted = true
      continue
    }
    slot.Generation++
    generation := slot.Generation
    slot.State = SLOT_CONNECTION_CREATION
    slot.Connection = driver.Connection{}
    driver_status := driver.Connect(
      Pool.Driver, Pool.Data_Source, &slot.Connection,
    )
    if driver_status != driver.STATUS_OK {
      slot.Connection = driver.Connection{}
      slot.State = SLOT_EMPTY
      *Status = Status(status_of_driver(driver_status))
      return Connection{}
    }
    slot.State = SLOT_CONNECTION
    Connection = Connection(
      connection_of(
        open_pool, Slot_Index(index), Lease_Generation(generation),
      ),
    )
    *Status = STATUS_OK
    return Connection
  }
  if generation_exhausted {
    *Status = STATUS_GENERATION_EXHAUSTED
    return Connection{}
  }
  *Status = STATUS_EXHAUSTED
  return Connection{}
}

// Connection_Close returns one explicit lease to idle pool state.
func Connection_Close(Connection Connection_Pointer) (Status Reservation_Status) {
  defer func() {
    Reservation_Status_Invariants(Status, "connection_close.status")
  }()
  Connection_Pointer_Invariants(Connection, "connection_close.connection")
  var table driver.Connection
  Status = connection_reserve(
    Connection,
    Reservation_State(SLOT_CONNECTION),
    Activity_State(SLOT_CLOSE),
    &table,
  )
  if Status != Reservation_Status(STATUS_OK) {
    return Status
  }
  transition_status := connection_transition(
    Live_Connection_Pointer((*Live_Connection)((*Connection)(Connection))),
    Activity_State(SLOT_CLOSE),
    TRANSITION_RETURN,
    Return_State(SLOT_IDLE),
  )
  if transition_status == Transition_Status(STATUS_OK) {
    *Connection = Connection{}
  }
  return Reservation_Status(transition_status)
}

// Pool_Close closes every idle connection after refusing active leases.
func Pool_Close(Pool Pool_Pointer) (Status Operation_Status) {
  defer func() { Operation_Status_Invariants(Status, "pool_close.status") }()
  Pool_Pointer_Invariants(Pool, "pool_close.pool")
  pool_initialized(Pool)
  if Pool.Closed == POOL_CLOSED {
    return Operation_Status(STATUS_OK)
  }
  Slots := Pool.Slots
  for index := range Slots {
    State := Slots[index].State
    if State != SLOT_EMPTY {
      if State != SLOT_IDLE {
        return Operation_Status(STATUS_BUSY)
      }
    }
  }
  Pool.Closed = POOL_CLOSED
  for index := range Slots {
    if Slots[index].State == SLOT_IDLE {
      Slots[index].State = SLOT_CLOSE
    }
  }
  Status = Operation_Status(STATUS_OK)
  for index := range Slots {
    slot := &Slots[index]
    if slot.State != SLOT_CLOSE {
      continue
    }
    driver_status := driver.Connection_Close(&slot.Connection)
    if Status == Operation_Status(STATUS_OK) {
      if driver_status != driver.STATUS_OK {
        Status = Operation_Status(status_of_driver(driver_status))
      }
    }
    slot.Connection = driver.Connection{}
    slot.State = SLOT_EMPTY
  }
  return Status
}

// Pool_Statistics returns one bounded count snapshot.
func Pool_Statistics(Pool Pool_Pointer) (statistics Statistics) {
  defer func() { Statistics_Invariants(&statistics, "pool_statistics.statistics") }()
  Pool_Pointer_Invariants(Pool, "pool_statistics.pool")
  pool_initialized(Pool)
  Slots := Pool.Slots
  statistics.Capacity = Capacity_Count(len(Slots))
  for index := range Slots {
    State := Slots[index].State
    if State != SLOT_EMPTY {
      statistics.Open_Count++
    }
    if State == SLOT_IDLE {
      statistics.Idle_Count++
    }
    if State != SLOT_EMPTY {
      if State != SLOT_IDLE {
        if State != SLOT_CONNECTION_CREATION {
          statistics.In_Use_Count++
        }
      }
    }
  }
  return statistics
}

// Statistics_Capacity reports fixed caller slot count.
func Statistics_Capacity(value Statistics) (count Connection_Count) {
  defer func() { Connection_Count_Invariants(count, "statistics_capacity.count") }()
  Statistics_Invariants(&value, "statistics_capacity.value")
  return Connection_Count(value.Capacity)
}

// Statistics_Open reports slots with driver connections.
func Statistics_Open(value Statistics) (count Connection_Count) {
  defer func() { Connection_Count_Invariants(count, "statistics_open.count") }()
  Statistics_Invariants(&value, "statistics_open.value")
  return Connection_Count(value.Open_Count)
}

// Statistics_Idle reports reusable driver connections.
func Statistics_Idle(value Statistics) (count Connection_Count) {
  defer func() { Connection_Count_Invariants(count, "statistics_idle.count") }()
  Statistics_Invariants(&value, "statistics_idle.value")
  return Connection_Count(value.Idle_Count)
}

// Statistics_In_Use reports reserved driver connections.
func Statistics_In_Use(value Statistics) (count Connection_Count) {
  defer func() { Connection_Count_Invariants(count, "statistics_in_use.count") }()
  Statistics_Invariants(&value, "statistics_in_use.value")
  return Connection_Count(value.In_Use_Count)
}

// Result_Rows_Affected reports driver affected count when supported.
func Result_Rows_Affected(
  Result Result,
  Status driver.Optional_Status_Pointer,
) (count driver.Rows_Affected) {
  defer func() { driver.Rows_Affected_Invariants(count, "result_rows_affected.count") }()
  Result_Invariants(&Result, "result_rows_affected.result")
  driver.Optional_Status_Pointer_Invariants(
    Status, "result_rows_affected.status_storage",
  )
  defer func() {
    driver.Optional_Status_Pointer_Invariants(Status, "result_rows_affected.status")
  }()
  return driver.Result_Rows_Affected(&Result.Driver_Result, Status)
}

// Result_Last_Insert_Identifier reports generated identity when supported.
func Result_Last_Insert_Identifier(
  Result Result,
  Status driver.Optional_Status_Pointer,
) (identifier driver.Last_Insert_Identifier) {
  defer func() {
    driver.Last_Insert_Identifier_Invariants(
      identifier, "result_last_insert_identifier.identifier",
    )
  }()
  Result_Invariants(&Result, "result_last_insert_identifier.result")
  driver.Optional_Status_Pointer_Invariants(
    Status, "result_last_insert_identifier.status_storage",
  )
  defer func() {
    driver.Optional_Status_Pointer_Invariants(
      Status, "result_last_insert_identifier.status",
    )
  }()
  return driver.Result_Last_Insert_Identifier(&Result.Driver_Result, Status)
}

// Pool_Probe probes one bounded connection and returns it to pool.
func Pool_Probe(Pool Pool_Pointer) (Status Status) {
  defer func() { Status_Invariants(Status, "pool_probe.status") }()
  Pool_Pointer_Invariants(Pool, "pool_probe.pool")
  lease := Pool_Connection_Acquire(Pool, &Status)
  if Status != STATUS_OK {
    return Status
  }
  return Status(connection_probe(
    &lease, Connection_Return_State(SLOT_IDLE),
  ))
}

// Connection_Probe probes one explicit connection lease.
func Connection_Probe(Connection Connection_Pointer) (Status Operation_Status) {
  defer func() { Operation_Status_Invariants(Status, "connection_probe.status") }()
  Connection_Pointer_Invariants(Connection, "connection_probe.connection")
  return connection_probe(Connection, Connection_Return_State(SLOT_CONNECTION))
}

// Pool_Exec executes one request into caller-owned result storage.
func Pool_Exec(Pool Pool_Pointer, request driver.Request, Result Result_Pointer) (Status Status) {
  defer func() { Status_Invariants(Status, "pool_exec.status") }()
  Pool_Pointer_Invariants(Pool, "pool_exec.pool")
  driver.Request_Invariants(&request, "pool_exec.request")
  Result_Pointer_Invariants(Result, "pool_exec.result")
  lease := Pool_Connection_Acquire(Pool, &Status)
  if Status != STATUS_OK {
    return Status
  }
  return Status(connection_exec(
    &lease, request, Connection_Return_State(SLOT_IDLE), Result,
  ))
}

// Connection_Exec executes one request through explicit lease.
func Connection_Exec(
  Connection Connection_Pointer,
  request driver.Request,
  Result Result_Pointer,
) (Status Operation_Status) {
  defer func() { Operation_Status_Invariants(Status, "connection_exec.status") }()
  Connection_Pointer_Invariants(Connection, "connection_exec.connection")
  driver.Request_Invariants(&request, "connection_exec.request")
  Result_Pointer_Invariants(Result, "connection_exec.result")
  return connection_exec(
    Connection, request, Connection_Return_State(SLOT_CONNECTION), Result,
  )
}

// Pool_Query opens rows into caller-owned cursor storage.
func Pool_Query(Pool Pool_Pointer, request driver.Request, Rows Rows_Pointer) (Status Status) {
  defer func() { Status_Invariants(Status, "pool_query.status") }()
  Pool_Pointer_Invariants(Pool, "pool_query.pool")
  driver.Request_Invariants(&request, "pool_query.request")
  Rows_Pointer_Invariants(Rows, "pool_query.rows")
  lease := Pool_Connection_Acquire(Pool, &Status)
  if Status != STATUS_OK {
    return Status
  }
  return Status(connection_query(
    &lease, request, Connection_Return_State(SLOT_IDLE), Rows,
  ))
}

// Connection_Query opens rows through explicit lease.
func Connection_Query(
  Connection Connection_Pointer,
  request driver.Request,
  Rows Rows_Pointer,
) (Status Operation_Status) {
  defer func() { Operation_Status_Invariants(Status, "connection_query.status") }()
  Connection_Pointer_Invariants(Connection, "connection_query.connection")
  driver.Request_Invariants(&request, "connection_query.request")
  Rows_Pointer_Invariants(Rows, "connection_query.rows")
  return connection_query(
    Connection, request, Connection_Return_State(SLOT_CONNECTION), Rows,
  )
}

// Rows_Next writes one exact-width row and closes automatically at exhaustion.
func Rows_Next(Rows Rows_Pointer, destination driver.Values) (Status Operation_Status) {
  defer func() { Operation_Status_Invariants(Status, "rows_next.status") }()
  Rows_Pointer_Invariants(Rows, "rows_next.rows")
  Rows_Invariants(Rows, "rows_next.rows_value")
  driver.Values_Invariants(destination, "rows_next.destination")
  lease := &Rows.Connection
  var table driver.Connection
  reservation_status := connection_reserve(
    lease,
    Reservation_State(SLOT_ROWS),
    Activity_State(SLOT_ROWS_READING),
    &table,
  )
  if reservation_status != Reservation_Status(STATUS_OK) {
    return Operation_Status(reservation_status)
  }
  driver_status := driver.Rows_Next(&Rows.Driver_Rows, destination)
  if driver_status == driver.STATUS_OK {
    return Operation_Status(
      connection_transition(
        Live_Connection_Pointer((*Live_Connection)(lease)),
        Activity_State(SLOT_ROWS_READING),
        TRANSITION_ROWS,
        Return_State(SLOT_IDLE),
      ),
    )
  }
  if driver_status == driver.STATUS_DONE {
    close_status := driver.Rows_Close(&Rows.Driver_Rows)
    if close_status != driver.STATUS_OK {
      Status = operation_complete(
        Live_Connection_Pointer((*Live_Connection)(lease)),
        &table,
        Activity_State(SLOT_ROWS_READING),
        Return_State(Rows.Return_State),
        close_status,
      )
      *Rows = Rows{}
      return Status
    }
    transition_status := connection_transition(
      Live_Connection_Pointer((*Live_Connection)(lease)),
      Activity_State(SLOT_ROWS_READING),
      TRANSITION_RETURN,
      Return_State(Rows.Return_State),
    )
    *Rows = Rows{}
    if transition_status != Transition_Status(STATUS_OK) {
      return Operation_Status(transition_status)
    }
    return Operation_Status(STATUS_DONE)
  }
  if driver_status == driver.STATUS_BAD_CONNECTION {
    driver.Connection_Close(&table)
    transition_status := connection_discard(
      Live_Connection_Pointer((*Live_Connection)(lease)),
      Discard_State(SLOT_ROWS_READING),
    )
    *Rows = Rows{}
    if transition_status != Transition_Status(STATUS_OK) {
      return Operation_Status(transition_status)
    }
    return Operation_Status(STATUS_BAD_CONNECTION)
  }
  transition_status := connection_transition(
    Live_Connection_Pointer((*Live_Connection)(lease)),
    Activity_State(SLOT_ROWS_READING),
    TRANSITION_ROWS,
    Return_State(SLOT_IDLE),
  )
  if transition_status != Transition_Status(STATUS_OK) {
    return Operation_Status(transition_status)
  }
  return Operation_Status(status_of_driver(driver_status))
}

// Rows_Close releases cursor and restores its owning slot state.
func Rows_Close(Rows Rows_Pointer) (Status Operation_Status) {
  defer func() { Operation_Status_Invariants(Status, "rows_close.status") }()
  Rows_Pointer_Invariants(Rows, "rows_close.rows")
  lease := &Rows.Connection
  var table driver.Connection
  reservation_status := connection_reserve(
    lease,
    Reservation_State(SLOT_ROWS),
    Activity_State(SLOT_CLOSE),
    &table,
  )
  if reservation_status != Reservation_Status(STATUS_OK) {
    return Operation_Status(reservation_status)
  }
  driver_status := driver.Rows_Close(&Rows.Driver_Rows)
  Status = operation_complete(
    Live_Connection_Pointer((*Live_Connection)(lease)),
    &table,
    Activity_State(SLOT_CLOSE),
    Return_State(Rows.Return_State),
    driver_status,
  )
  *Rows = Rows{}
  return Status
}

// Pool_Prepare pins one lease behind caller-owned prepared statement.
func Pool_Prepare(
  Pool Pool_Pointer,
  query driver.Query,
  Statement Statement_Pointer,
) (Status Status) {
  defer func() { Status_Invariants(Status, "pool_prepare.status") }()
  Pool_Pointer_Invariants(Pool, "pool_prepare.pool")
  driver.Query_Invariants(query, "pool_prepare.query")
  Statement_Pointer_Invariants(Statement, "pool_prepare.statement")
  lease := Pool_Connection_Acquire(Pool, &Status)
  if Status != STATUS_OK {
    return Status
  }
  return Status(
    connection_prepare(
      &lease, query, Connection_Return_State(SLOT_IDLE), Statement,
    ),
  )
}

// Connection_Prepare pins explicit lease behind prepared statement.
func Connection_Prepare(
  Connection Connection_Pointer,
  query driver.Query,
  Statement Statement_Pointer,
) (Status Operation_Status) {
  defer func() { Operation_Status_Invariants(Status, "connection_prepare.status") }()
  Connection_Pointer_Invariants(Connection, "connection_prepare.connection")
  driver.Query_Invariants(query, "connection_prepare.query")
  Statement_Pointer_Invariants(Statement, "connection_prepare.statement")
  return connection_prepare(
    Connection, query, Connection_Return_State(SLOT_CONNECTION), Statement,
  )
}

// Statement_Exec executes one prepared request.
func Statement_Exec(
  Statement Statement_Pointer,
  arguments driver.Arguments,
  Result Result_Pointer,
) (Status Operation_Status) {
  defer func() { Operation_Status_Invariants(Status, "statement_exec.status") }()
  Statement_Pointer_Invariants(Statement, "statement_exec.statement")
  Statement_Invariants(Statement, "statement_exec.statement_value")
  driver.Arguments_Invariants(arguments, "statement_exec.arguments")
  Result_Pointer_Invariants(Result, "statement_exec.result")
  lease := &Statement.Connection
  var table driver.Connection
  reservation_status := connection_reserve(
    lease,
    Reservation_State(SLOT_STATEMENT),
    Activity_State(SLOT_EXECUTION),
    &table,
  )
  if reservation_status != Reservation_Status(STATUS_OK) {
    return Operation_Status(reservation_status)
  }
  *Result = Result{}
  driver_status := driver.Statement_Exec(
    &Statement.Driver_Statement,
    arguments,
    &Result.Driver_Result,
  )
  Status = operation_complete(
    Live_Connection_Pointer((*Live_Connection)(lease)),
    &table,
    Activity_State(SLOT_EXECUTION),
    Return_State(SLOT_STATEMENT),
    driver_status,
  )
  return Status
}

// Statement_Query opens rows through one prepared statement.
func Statement_Query(
  Statement Statement_Pointer,
  arguments driver.Arguments,
  Rows Rows_Pointer,
) (Status Operation_Status) {
  defer func() { Operation_Status_Invariants(Status, "statement_query.status") }()
  Statement_Pointer_Invariants(Statement, "statement_query.statement")
  driver.Arguments_Invariants(arguments, "statement_query.arguments")
  Rows_Pointer_Invariants(Rows, "statement_query.rows")
  lease := &Statement.Connection
  var table driver.Connection
  reservation_status := connection_reserve(
    lease,
    Reservation_State(SLOT_STATEMENT),
    Activity_State(SLOT_EXECUTION),
    &table,
  )
  if reservation_status != Reservation_Status(STATUS_OK) {
    return Operation_Status(reservation_status)
  }
  rows_initialize(
    Rows,
    Live_Connection_Pointer((*Live_Connection)(lease)),
    Return_State(SLOT_STATEMENT),
  )
  driver_status := driver.Statement_Query(
    &Statement.Driver_Statement, arguments,
    &Rows.Driver_Rows,
  )
  if driver_status != driver.STATUS_OK {
    Status = operation_complete(
      Live_Connection_Pointer((*Live_Connection)(lease)),
      &table,
      Activity_State(SLOT_EXECUTION),
      Return_State(SLOT_STATEMENT),
      driver_status,
    )
    *Rows = Rows{}
    return Status
  }
  transition_status := connection_transition(
    Live_Connection_Pointer((*Live_Connection)(lease)),
    Activity_State(SLOT_EXECUTION),
    TRANSITION_ROWS,
    Return_State(SLOT_IDLE),
  )
  if transition_status != Transition_Status(STATUS_OK) {
    driver.Rows_Close(&Rows.Driver_Rows)
    *Rows = Rows{}
    return Operation_Status(transition_status)
  }
  return Operation_Status(STATUS_OK)
}

// Statement_Close releases prepared state and restores owning lease.
func Statement_Close(Statement Statement_Pointer) (Status Operation_Status) {
  defer func() { Operation_Status_Invariants(Status, "statement_close.status") }()
  Statement_Pointer_Invariants(Statement, "statement_close.statement")
  lease := &Statement.Connection
  var table driver.Connection
  reservation_status := connection_reserve(
    lease,
    Reservation_State(SLOT_STATEMENT),
    Activity_State(SLOT_CLOSE),
    &table,
  )
  if reservation_status != Reservation_Status(STATUS_OK) {
    return Operation_Status(reservation_status)
  }
  driver_status := driver.Statement_Close(&Statement.Driver_Statement)
  Status = operation_complete(
    Live_Connection_Pointer((*Live_Connection)(lease)),
    &table,
    Activity_State(SLOT_CLOSE),
    Return_State(Statement.Return_State),
    driver_status,
  )
  *Statement = Statement{}
  return Status
}

// Pool_Begin pins one lease behind a transaction.
func Pool_Begin(
  Pool Pool_Pointer,
  options driver.Transaction_Options,
  Transaction Transaction_Pointer,
) (Status Status) {
  defer func() { Status_Invariants(Status, "pool_begin.status") }()
  Pool_Pointer_Invariants(Pool, "pool_begin.pool")
  driver.Transaction_Options_Invariants(&options, "pool_begin.options")
  Transaction_Pointer_Invariants(Transaction, "pool_begin.transaction")
  lease := Pool_Connection_Acquire(Pool, &Status)
  if Status != STATUS_OK {
    return Status
  }
  return Status(
    connection_begin(
      &lease, options, Connection_Return_State(SLOT_IDLE), Transaction,
    ),
  )
}

// Connection_Begin pins explicit lease behind a transaction.
func Connection_Begin(
  Connection Connection_Pointer,
  options driver.Transaction_Options,
  Transaction Transaction_Pointer,
) (Status Operation_Status) {
  defer func() { Operation_Status_Invariants(Status, "connection_begin.status") }()
  Connection_Pointer_Invariants(Connection, "connection_begin.connection")
  driver.Transaction_Options_Invariants(&options, "connection_begin.options")
  Transaction_Pointer_Invariants(Transaction, "connection_begin.transaction")
  return connection_begin(
    Connection,
    options,
    Connection_Return_State(SLOT_CONNECTION),
    Transaction,
  )
}

// Transaction_Exec executes one request inside transaction.
func Transaction_Exec(
  Transaction Transaction_Pointer,
  request driver.Request,
  Result Result_Pointer,
) (Status Operation_Status) {
  defer func() { Operation_Status_Invariants(Status, "transaction_exec.status") }()
  Transaction_Pointer_Invariants(Transaction, "transaction_exec.transaction")
  Transaction_Invariants(Transaction, "transaction_exec.transaction_value")
  driver.Request_Invariants(&request, "transaction_exec.request")
  Result_Pointer_Invariants(Result, "transaction_exec.result")
  lease := &Transaction.Connection
  var table driver.Connection
  reservation_status := connection_reserve(
    lease,
    Reservation_State(SLOT_TRANSACTION),
    Activity_State(SLOT_EXECUTION),
    &table,
  )
  if reservation_status != Reservation_Status(STATUS_OK) {
    return Operation_Status(reservation_status)
  }
  *Result = Result{}
  driver_status := driver.Connection_Exec(
    &table, request, &Result.Driver_Result,
  )
  Status = operation_complete(
    Live_Connection_Pointer((*Live_Connection)(lease)),
    &table,
    Activity_State(SLOT_EXECUTION),
    Return_State(SLOT_TRANSACTION),
    driver_status,
  )
  return Status
}

// Transaction_Query opens rows inside transaction.
func Transaction_Query(
  Transaction Transaction_Pointer,
  request driver.Request,
  Rows Rows_Pointer,
) (Status Operation_Status) {
  defer func() { Operation_Status_Invariants(Status, "transaction_query.status") }()
  Transaction_Pointer_Invariants(Transaction, "transaction_query.transaction")
  driver.Request_Invariants(&request, "transaction_query.request")
  Rows_Pointer_Invariants(Rows, "transaction_query.rows")
  lease := &Transaction.Connection
  var table driver.Connection
  reservation_status := connection_reserve(
    lease,
    Reservation_State(SLOT_TRANSACTION),
    Activity_State(SLOT_EXECUTION),
    &table,
  )
  if reservation_status != Reservation_Status(STATUS_OK) {
    return Operation_Status(reservation_status)
  }
  rows_initialize(
    Rows,
    Live_Connection_Pointer((*Live_Connection)(lease)),
    Return_State(SLOT_TRANSACTION),
  )
  driver_status := driver.Connection_Query(
    &table, request, &Rows.Driver_Rows,
  )
  if driver_status != driver.STATUS_OK {
    Status = operation_complete(
      Live_Connection_Pointer((*Live_Connection)(lease)),
      &table,
      Activity_State(SLOT_EXECUTION),
      Return_State(SLOT_TRANSACTION),
      driver_status,
    )
    *Rows = Rows{}
    return Status
  }
  transition_status := connection_transition(
    Live_Connection_Pointer((*Live_Connection)(lease)),
    Activity_State(SLOT_EXECUTION),
    TRANSITION_ROWS,
    Return_State(SLOT_IDLE),
  )
  if transition_status != Transition_Status(STATUS_OK) {
    driver.Rows_Close(&Rows.Driver_Rows)
    *Rows = Rows{}
    return Operation_Status(transition_status)
  }
  return Operation_Status(STATUS_OK)
}

// Transaction_Commit makes transaction durable and releases owning lease.
func Transaction_Commit(Transaction Transaction_Pointer) (Status Operation_Status) {
  defer func() { Operation_Status_Invariants(Status, "transaction_commit.status") }()
  Transaction_Pointer_Invariants(Transaction, "transaction_commit.transaction")
  return transaction_finish(Transaction, TRANSACTION_COMMIT)
}

// Transaction_Rollback abandons transaction and releases owning lease.
func Transaction_Rollback(Transaction Transaction_Pointer) (Status Operation_Status) {
  defer func() { Operation_Status_Invariants(Status, "transaction_rollback.status") }()
  Transaction_Pointer_Invariants(Transaction, "transaction_rollback.transaction")
  return transaction_finish(Transaction, TRANSACTION_ROLLBACK)
}

func connection_probe(
  Connection Connection_Pointer,
  return_state Connection_Return_State,
) (Status Operation_Status) {
  defer func() {
    Operation_Status_Invariants(Status, "connection_probe_internal.status")
  }()
  Connection_Pointer_Invariants(
    Connection, "connection_probe_internal.connection",
  )
  Connection_Return_State_Invariants(
    return_state, "connection_probe_internal.return_state",
  )
  var table driver.Connection
  reservation_status := connection_reserve(
    Connection,
    Reservation_State(SLOT_CONNECTION),
    Activity_State(SLOT_EXECUTION),
    &table,
  )
  if reservation_status != Reservation_Status(STATUS_OK) {
    return Operation_Status(reservation_status)
  }
  driver_status := driver.Connection_Probe(&table)
  return operation_complete(
    Live_Connection_Pointer((*Live_Connection)((*Connection)(Connection))),
    &table,
    Activity_State(SLOT_EXECUTION),
    Return_State(return_state),
    driver_status,
  )
}

func connection_exec(
  Connection Connection_Pointer,
  request driver.Request,
  return_state Connection_Return_State,
  Result Result_Pointer,
) (Status Operation_Status) {
  defer func() {
    Operation_Status_Invariants(Status, "connection_exec_internal.status")
  }()
  Connection_Pointer_Invariants(
    Connection, "connection_exec_internal.connection",
  )
  driver.Request_Invariants(&request, "connection_exec_internal.request")
  Connection_Return_State_Invariants(
    return_state, "connection_exec_internal.return_state",
  )
  Result_Pointer_Invariants(Result, "connection_exec_internal.result")
  var table driver.Connection
  reservation_status := connection_reserve(
    Connection,
    Reservation_State(SLOT_CONNECTION),
    Activity_State(SLOT_EXECUTION),
    &table,
  )
  if reservation_status != Reservation_Status(STATUS_OK) {
    return Operation_Status(reservation_status)
  }
  *Result = Result{}
  driver_status := driver.Connection_Exec(
    &table, request, &Result.Driver_Result,
  )
  Status = operation_complete(
    Live_Connection_Pointer((*Live_Connection)((*Connection)(Connection))),
    &table,
    Activity_State(SLOT_EXECUTION),
    Return_State(return_state),
    driver_status,
  )
  return Status
}

func connection_query(
  Connection Connection_Pointer,
  request driver.Request,
  return_state Connection_Return_State,
  Rows Rows_Pointer,
) (Status Operation_Status) {
  defer func() {
    Operation_Status_Invariants(Status, "connection_query_internal.status")
  }()
  Connection_Pointer_Invariants(
    Connection, "connection_query_internal.connection",
  )
  driver.Request_Invariants(&request, "connection_query_internal.request")
  Connection_Return_State_Invariants(
    return_state, "connection_query_internal.return_state",
  )
  Rows_Pointer_Invariants(Rows, "connection_query_internal.rows")
  var table driver.Connection
  reservation_status := connection_reserve(
    Connection,
    Reservation_State(SLOT_CONNECTION),
    Activity_State(SLOT_EXECUTION),
    &table,
  )
  if reservation_status != Reservation_Status(STATUS_OK) {
    return Operation_Status(reservation_status)
  }
  rows_initialize(
    Rows,
    Live_Connection_Pointer((*Live_Connection)((*Connection)(Connection))),
    Return_State(return_state),
  )
  driver_status := driver.Connection_Query(
    &table, request, &Rows.Driver_Rows,
  )
  if driver_status != driver.STATUS_OK {
    Status = operation_complete(
      (*Live_Connection)((*Connection)(Connection)),
      &table,
      Activity_State(SLOT_EXECUTION),
      Return_State(return_state),
      driver_status,
    )
    *Rows = Rows{}
    return Status
  }
  transition_status := connection_transition(
    Live_Connection_Pointer((*Live_Connection)((*Connection)(Connection))),
    Activity_State(SLOT_EXECUTION),
    TRANSITION_ROWS,
    Return_State(SLOT_IDLE),
  )
  if transition_status != Transition_Status(STATUS_OK) {
    driver.Rows_Close(&Rows.Driver_Rows)
    *Rows = Rows{}
    return Operation_Status(transition_status)
  }
  return Operation_Status(STATUS_OK)
}

func connection_prepare(
  Connection Connection_Pointer,
  query driver.Query,
  return_state Connection_Return_State,
  Statement Statement_Pointer,
) (Status Operation_Status) {
  defer func() {
    Operation_Status_Invariants(Status, "connection_prepare_internal.status")
  }()
  Connection_Pointer_Invariants(
    Connection, "connection_prepare_internal.connection",
  )
  driver.Query_Invariants(query, "connection_prepare_internal.query")
  Connection_Return_State_Invariants(
    return_state, "connection_prepare_internal.return_state",
  )
  Statement_Pointer_Invariants(
    Statement, "connection_prepare_internal.statement",
  )
  var table driver.Connection
  reservation_status := connection_reserve(
    Connection,
    Reservation_State(SLOT_CONNECTION),
    Activity_State(SLOT_EXECUTION),
    &table,
  )
  if reservation_status != Reservation_Status(STATUS_OK) {
    return Operation_Status(reservation_status)
  }
  statement_initialize(
    Statement,
    Live_Connection_Pointer((*Live_Connection)((*Connection)(Connection))),
    return_state,
  )
  driver_status := driver.Connection_Prepare(
    &table, query, &Statement.Driver_Statement,
  )
  if driver_status != driver.STATUS_OK {
    Status = operation_complete(
      (*Live_Connection)((*Connection)(Connection)),
      &table,
      Activity_State(SLOT_EXECUTION),
      Return_State(return_state),
      driver_status,
    )
    *Statement = Statement{}
    return Status
  }
  transition_status := connection_transition(
    Live_Connection_Pointer((*Live_Connection)((*Connection)(Connection))),
    Activity_State(SLOT_EXECUTION),
    TRANSITION_RETURN,
    Return_State(SLOT_STATEMENT),
  )
  if transition_status != Transition_Status(STATUS_OK) {
    driver.Statement_Close(&Statement.Driver_Statement)
    *Statement = Statement{}
    return Operation_Status(transition_status)
  }
  return Operation_Status(STATUS_OK)
}

func connection_begin(
  Connection Connection_Pointer,
  options driver.Transaction_Options,
  return_state Connection_Return_State,
  Transaction Transaction_Pointer,
) (Status Operation_Status) {
  defer func() {
    Operation_Status_Invariants(Status, "connection_begin_internal.status")
  }()
  Connection_Pointer_Invariants(
    Connection, "connection_begin_internal.connection",
  )
  driver.Transaction_Options_Invariants(&options, "connection_begin_internal.options")
  Connection_Return_State_Invariants(
    return_state, "connection_begin_internal.return_state",
  )
  Transaction_Pointer_Invariants(
    Transaction, "connection_begin_internal.transaction",
  )
  var table driver.Connection
  reservation_status := connection_reserve(
    Connection,
    Reservation_State(SLOT_CONNECTION),
    Activity_State(SLOT_EXECUTION),
    &table,
  )
  if reservation_status != Reservation_Status(STATUS_OK) {
    return Operation_Status(reservation_status)
  }
  transaction_initialize(
    Transaction,
    Live_Connection_Pointer((*Live_Connection)((*Connection)(Connection))),
    return_state,
  )
  driver_status := driver.Connection_Begin(
    &table, options, &Transaction.Driver_Transaction,
  )
  if driver_status != driver.STATUS_OK {
    Status = operation_complete(
      (*Live_Connection)((*Connection)(Connection)),
      &table,
      Activity_State(SLOT_EXECUTION),
      Return_State(return_state),
      driver_status,
    )
    *Transaction = Transaction{}
    return Status
  }
  transition_status := connection_transition(
    Live_Connection_Pointer((*Live_Connection)((*Connection)(Connection))),
    Activity_State(SLOT_EXECUTION),
    TRANSITION_RETURN,
    Return_State(SLOT_TRANSACTION),
  )
  if transition_status != Transition_Status(STATUS_OK) {
    driver.Transaction_Rollback(&Transaction.Driver_Transaction)
    *Transaction = Transaction{}
    return Operation_Status(transition_status)
  }
  return Operation_Status(STATUS_OK)
}

func transaction_finish(
  Transaction Transaction_Pointer,
  terminal Transaction_Terminal,
) (Status Operation_Status) {
  defer func() { Operation_Status_Invariants(Status, "transaction_finish.status") }()
  Transaction_Pointer_Invariants(Transaction, "transaction_finish.transaction")
  Transaction_Terminal_Invariants(terminal, "transaction_finish.terminal")
  lease := &Transaction.Connection
  var table driver.Connection
  reservation_status := connection_reserve(
    lease,
    Reservation_State(SLOT_TRANSACTION),
    Activity_State(SLOT_CLOSE),
    &table,
  )
  if reservation_status != Reservation_Status(STATUS_OK) {
    return Operation_Status(reservation_status)
  }
  driver_status := driver.STATUS_OK
  if terminal == TRANSACTION_COMMIT {
    driver_status = driver.Transaction_Commit(
      &Transaction.Driver_Transaction,
    )
  } else {
    driver_status = driver.Transaction_Rollback(
      &Transaction.Driver_Transaction,
    )
  }
  Status = operation_complete(
    Live_Connection_Pointer((*Live_Connection)(lease)),
    &table,
    Activity_State(SLOT_CLOSE),
    Return_State(Transaction.Return_State),
    driver_status,
  )
  *Transaction = Transaction{}
  return Status
}

func connection_reserve(
  Connection Connection_Pointer,
  expected Reservation_State,
  next Activity_State,
  destination driver.Connection_Pointer,
) (Status Reservation_Status) {
  defer func() {
    Reservation_Status_Invariants(Status, "connection_reserve.status")
  }()
  Connection_Pointer_Invariants(Connection, "connection_reserve.connection")
  Reservation_State_Invariants(expected, "connection_reserve.expected")
  Activity_State_Invariants(next, "connection_reserve.next")
  driver.Connection_Pointer_Invariants(destination, "connection_reserve.destination")
  owner := (*Pool)(Connection.Pool)
  if owner == nil {
    return Reservation_Status(STATUS_HANDLE_INVALID)
  }
  Pool_Pointer_Invariants(Pool_Pointer(owner), "connection_reserve.pool")
  pool_initialized(owner)
  index := Connection.Slot_Index
  generation := Lease_Generation(Connection.Generation)
  Slot_Index_Invariants(index, "connection_reserve.index")
  Lease_Generation_Invariants(generation, "connection_reserve.generation")
  if owner.Closed == POOL_CLOSED {
    return Reservation_Status(STATUS_CLOSED)
  }
  Slots := owner.Slots
  if !(int(index) < len(Slots)) {
    return Reservation_Status(STATUS_HANDLE_INVALID)
  }
  slot := &Slots[index]
  if Lease_Generation(slot.Generation) != generation {
    return Reservation_Status(STATUS_HANDLE_INVALID)
  }
  if Reservation_State(slot.State) != expected {
    State := slot.State
    if State == SLOT_EMPTY {
      return Reservation_Status(STATUS_HANDLE_INVALID)
    }
    if State == SLOT_IDLE {
      return Reservation_Status(STATUS_HANDLE_INVALID)
    }
    return Reservation_Status(STATUS_BUSY)
  }
  *destination = slot.Connection
  slot.State = Slot_State(next)
  return Reservation_Status(STATUS_OK)
}

func connection_transition(
  Connection Live_Connection_Pointer,
  expected Activity_State,
  kind Transition_Kind,
  return_state Return_State,
) (Status Transition_Status) {
  defer func() {
    Transition_Status_Invariants(Status, "connection_transition.status")
  }()
  Live_Connection_Pointer_Invariants(
    Connection, "connection_transition.connection",
  )
  Activity_State_Invariants(expected, "connection_transition.expected")
  Transition_Kind_Invariants(kind, "connection_transition.kind")
  Return_State_Invariants(return_state, "connection_transition.return_state")
  owner := (*Pool)(Connection.Pool)
  if owner == nil {
    return Transition_Status(STATUS_HANDLE_INVALID)
  }
  Open_Pool_Pointer_Invariants(
    Open_Pool_Pointer((*Open_Pool)(owner)), "connection_transition.pool",
  )
  pool_initialized(owner)
  index := Connection.Slot_Index
  generation := Lease_Generation(Connection.Generation)
  Slot_Index_Invariants(index, "connection_transition.index")
  Lease_Generation_Invariants(generation, "connection_transition.generation")
  Slots := owner.Slots
  if !(int(index) < len(Slots)) {
    return Transition_Status(STATUS_HANDLE_INVALID)
  }
  slot := &Slots[index]
  if Lease_Generation(slot.Generation) != generation {
    return Transition_Status(STATUS_HANDLE_INVALID)
  }
  if Activity_State(slot.State) != expected {
    return Transition_Status(STATUS_BUSY)
  }
  next := Slot_State(return_state)
  if kind == TRANSITION_ROWS {
    next = SLOT_ROWS
  }
  slot.State = next
  if next == SLOT_IDLE {
    *Connection = Live_Connection{}
  }
  return Transition_Status(STATUS_OK)
}

func connection_discard(
  Connection Live_Connection_Pointer,
  expected Discard_State,
) (Status Transition_Status) {
  defer func() {
    Transition_Status_Invariants(Status, "connection_discard.status")
  }()
  Live_Connection_Pointer_Invariants(
    Connection, "connection_discard.connection",
  )
  Discard_State_Invariants(expected, "connection_discard.expected")
  owner := (*Pool)(Connection.Pool)
  if owner == nil {
    return Transition_Status(STATUS_HANDLE_INVALID)
  }
  Open_Pool_Pointer_Invariants(
    Open_Pool_Pointer((*Open_Pool)(owner)), "connection_discard.pool",
  )
  pool_initialized(owner)
  index := Connection.Slot_Index
  generation := Lease_Generation(Connection.Generation)
  Slot_Index_Invariants(index, "connection_discard.index")
  Lease_Generation_Invariants(generation, "connection_discard.generation")
  Slots := owner.Slots
  if !(int(index) < len(Slots)) {
    return Transition_Status(STATUS_HANDLE_INVALID)
  }
  slot := &Slots[index]
  if Lease_Generation(slot.Generation) != generation {
    return Transition_Status(STATUS_HANDLE_INVALID)
  }
  if Discard_State(slot.State) != expected {
    return Transition_Status(STATUS_BUSY)
  }
  slot.Connection = driver.Connection{}
  slot.State = SLOT_EMPTY
  *Connection = Live_Connection{}
  return Transition_Status(STATUS_OK)
}

func operation_complete(
  Connection Live_Connection_Pointer,
  table driver.Connection_Pointer,
  expected Activity_State,
  return_state Return_State,
  driver_status driver.Status,
) (Status Operation_Status) {
  defer func() { Operation_Status_Invariants(Status, "operation_complete.status") }()
  Live_Connection_Pointer_Invariants(Connection, "operation_complete.connection")
  driver.Connection_Pointer_Invariants(table, "operation_complete.table")
  Activity_State_Invariants(expected, "operation_complete.expected")
  Return_State_Invariants(return_state, "operation_complete.return_state")
  driver.Status_Invariants(driver_status, "operation_complete.driver_status")
  if driver_status == driver.STATUS_BAD_CONNECTION {
    driver.Connection_Close(table)
    transition_status := connection_discard(Connection, Discard_State(expected))
    if transition_status != Transition_Status(STATUS_OK) {
      return Operation_Status(transition_status)
    }
    return Operation_Status(STATUS_BAD_CONNECTION)
  }
  transition_status := connection_transition(
    Connection, expected, TRANSITION_RETURN, return_state,
  )
  if transition_status != Transition_Status(STATUS_OK) {
    return Operation_Status(transition_status)
  }
  return Operation_Status(status_of_driver(driver_status))
}

func connection_of(
  Pool Open_Pool_Pointer,
  index Slot_Index,
  generation Lease_Generation,
) (Connection Live_Connection) {
  defer func() {
    Live_Connection_Invariants(&Connection, "connection_of.connection")
  }()
  Open_Pool_Pointer_Invariants(Pool, "connection_of.pool")
  Slot_Index_Invariants(index, "connection_of.index")
  Lease_Generation_Invariants(generation, "connection_of.generation")
  return Live_Connection{
    Pool:       unsafe.Pointer(Pool),
    Slot_Index: index,
    Generation: Generation(generation),
  }
}

func rows_initialize(
  Rows Rows_Pointer,
  Connection Live_Connection_Pointer,
  return_state Return_State,
) {
  Rows_Pointer_Invariants(Rows, "rows_initialize.rows")
  Live_Connection_Pointer_Invariants(Connection, "rows_initialize.connection")
  Return_State_Invariants(return_state, "rows_initialize.return_state")
  *Rows = Rows{
    Connection:   Connection(*Connection),
    Return_State: Slot_State(return_state),
  }
}

func statement_initialize(
  Statement Statement_Pointer,
  Connection Live_Connection_Pointer,
  return_state Connection_Return_State,
) {
  Statement_Pointer_Invariants(Statement, "statement_initialize.statement")
  Live_Connection_Pointer_Invariants(
    Connection, "statement_initialize.connection",
  )
  Connection_Return_State_Invariants(return_state, "statement_initialize.return_state")
  *Statement = Statement{
    Connection:   Connection(*Connection),
    Return_State: Slot_State(return_state),
  }
}

func transaction_initialize(
  Transaction Transaction_Pointer,
  Connection Live_Connection_Pointer,
  return_state Connection_Return_State,
) {
  Transaction_Pointer_Invariants(Transaction, "transaction_initialize.transaction")
  Live_Connection_Pointer_Invariants(
    Connection, "transaction_initialize.connection",
  )
  Connection_Return_State_Invariants(
    return_state, "transaction_initialize.return_state",
  )
  *Transaction = Transaction{
    Connection:   Connection(*Connection),
    Return_State: Slot_State(return_state),
  }
}

func status_of_driver(driver_status driver.Status) (Status Driver_Status) {
  defer func() { Driver_Status_Invariants(Status, "status_of_driver.status") }()
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

func pool_initialized(Pool Pool_Pointer) {
  Pool_Pointer_Invariants(Pool, "pool_initialized.pool")
  driver.Driver_Invariants(&Pool.Driver, "pool_initialized.driver")
  driver.Data_Source_Invariants(
    Pool.Data_Source, "pool_initialized.data_source",
  )
  Initialized_Slot_Storage_Invariants(
    Initialized_Slot_Storage(Pool.Slots),
    "pool_initialized.slots",
  )
  pool_driver_live(Pool.Driver)
}

func pool_driver_live(value driver.Driver) {
  driver.Driver_Invariants(&value, "pool_driver_live.value")
  aver.Always(
    value.Connect_Procedure != nil,
    "Pool driver can open one connection.",
  )
}
