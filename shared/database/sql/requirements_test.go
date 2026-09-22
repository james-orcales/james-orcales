package sql_test

import "testing"
import "unsafe"

import "local/james-orcales/shared/database/driver"
import "local/james-orcales/shared/database/sql"
import "local/james-orcales/shared/math/bits"
import "local/james-orcales/shared/slices"
import "local/james-orcales/shared/strings"
import "local/james-orcales/shared/testify"

// Test_Initialization protects fixed caller capacity and injected driver.
func Test_Initialization(t *testing.T) {
  State := fake_state{}
  data_source, validation := data_source_validate("memory")
  testify.Equal_Values(t, driver.STATUS_OK, validation, "data source")
  var Pool sql.Pool
  var Slots [POOL_CAPACITY]sql.Slot
  Status := sql.Pool_Init(
    &Pool, fake_driver(&State), data_source, Slots[:],
  )
  testify.Equal_Values(t, sql.STATUS_OK, Status, "pool init")
  statistics := sql.Pool_Statistics(&Pool)
  testify.Equal(
    t,
    sql.Connection_Count(POOL_CAPACITY),
    sql.Statistics_Capacity(statistics),
    "capacity",
  )

  var empty sql.Pool
  Status = sql.Pool_Init(
    &empty, fake_driver(&State), data_source, nil,
  )
  testify.Equal_Values(t, sql.STATUS_STORAGE_INVALID, Status, "empty capacity")
}

// Test_Pool protects reuse, exhaustion, stale lease rejection, and bounded close.
func Test_Pool(t *testing.T) {
  State := fake_state{}
  Pool := test_pool(t, &State)
  First, Status := pool_connection_acquire(&Pool)
  testify.Equal_Values(t, sql.STATUS_OK, Status, "first")
  stale := First
  Second, Status := pool_connection_acquire(&Pool)
  testify.Equal_Values(t, sql.STATUS_OK, Status, "second")
  _, Status = pool_connection_acquire(&Pool)
  testify.Equal_Values(t, sql.STATUS_EXHAUSTED, Status, "bounded exhaustion")
  testify.Equal_Values(t, sql.STATUS_BUSY, sql.Pool_Close(&Pool), "busy close")
  testify.Equal_Values(t, sql.STATUS_OK, sql.Connection_Close(&First), "release first")
  reused, Status := pool_connection_acquire(&Pool)
  testify.Equal_Values(t, sql.STATUS_OK, Status, "reuse")
  testify.Equal_Values(t, sql.STATUS_HANDLE_INVALID, sql.Connection_Probe(&stale), "stale")
  testify.Equal_Values(t, sql.STATUS_OK, sql.Connection_Close(&reused), "release reused")
  testify.Equal_Values(t, sql.STATUS_OK, sql.Connection_Close(&Second), "release second")
  testify.Equal_Values(t, sql.STATUS_OK, sql.Pool_Close(&Pool), "close")
  testify.Equal(t, POOL_CAPACITY, State.Close_Count, "driver close count")
}

// Test_Connections protects direct ping, execute, query, rows ownership, and release.
func Test_Connections(t *testing.T) {
  State := fake_state{}
  Pool := test_pool(t, &State)
  Connection, Status := pool_connection_acquire(&Pool)
  testify.Equal_Values(t, sql.STATUS_OK, Status, "acquire")
  testify.Equal_Values(t, sql.STATUS_OK, sql.Connection_Probe(&Connection), "probe")
  request := test_request(t)
  var Result sql.Result
  Status = sql.Status(sql.Connection_Exec(&Connection, request, &Result))
  testify.Equal_Values(t, sql.STATUS_OK, Status, "execute")
  _, result_status := result_rows_affected(Result)
  testify.Equal_Values(t, driver.STATUS_OK, result_status, "affected")
  var Rows sql.Rows
  Status = sql.Status(sql.Connection_Query(&Connection, request, &Rows))
  testify.Equal_Values(t, sql.STATUS_OK, Status, "query")
  testify.Equal_Values(t, sql.STATUS_BUSY, sql.Connection_Close(&Connection), "active rows")
  var Values [ROW_COLUMN_COUNT]driver.Value
  testify.Equal_Values(t, sql.STATUS_OK, sql.Rows_Next(&Rows, Values[:]), "row")
  testify.Equal_Values(t, sql.STATUS_DONE, sql.Rows_Next(&Rows, Values[:]), "done")
  testify.Equal_Values(t, sql.STATUS_OK, sql.Connection_Close(&Connection), "release")
  testify.Equal_Values(t, sql.STATUS_OK, sql.Pool_Close(&Pool), "pool close")
}

// Test_Rows protects exact destination size and explicit idempotent close rejection.
func Test_Rows(t *testing.T) {
  State := fake_state{}
  Pool := test_pool(t, &State)
  var Rows sql.Rows
  Status := sql.Pool_Query(&Pool, test_request(t), &Rows)
  testify.Equal_Values(t, sql.STATUS_OK, Status, "query")
  testify.Equal_Values(t, sql.STATUS_STORAGE_INVALID, sql.Rows_Next(&Rows, nil), "width")
  testify.Equal_Values(t, sql.STATUS_OK, sql.Rows_Close(&Rows), "close")
  testify.Equal_Values(t, sql.STATUS_HANDLE_INVALID, sql.Rows_Close(&Rows), "closed")
  testify.Equal_Values(t, sql.STATUS_OK, sql.Pool_Close(&Pool), "pool close")
}

// Test_Statements protects pinned prepare, execution, rows, and release.
func Test_Statements(t *testing.T) {
  State := fake_state{}
  Pool := test_pool(t, &State)
  query, _ := query_validate("select value")
  var Statement sql.Statement
  Status := sql.Pool_Prepare(&Pool, query, &Statement)
  testify.Equal_Values(t, sql.STATUS_OK, Status, "prepare")
  var Result sql.Result
  Status = sql.Status(sql.Statement_Exec(&Statement, nil, &Result))
  testify.Equal_Values(t, sql.STATUS_OK, Status, "execute")
  var Rows sql.Rows
  Status = sql.Status(sql.Statement_Query(&Statement, nil, &Rows))
  testify.Equal_Values(t, sql.STATUS_OK, Status, "query")
  testify.Equal_Values(t, sql.STATUS_BUSY, sql.Statement_Close(&Statement), "active rows")
  testify.Equal_Values(t, sql.STATUS_OK, sql.Rows_Close(&Rows), "rows close")
  testify.Equal_Values(t, sql.STATUS_OK, sql.Statement_Close(&Statement), "close")
  testify.Equal_Values(t, sql.STATUS_OK, sql.Pool_Close(&Pool), "pool close")
}

// Test_Transactions protects execution, rows, commit, rollback, and terminal state.
func Test_Transactions(t *testing.T) {
  State := fake_state{}
  Pool := test_pool(t, &State)
  options := driver.Transaction_Options_Of(driver.ISOLATION_SERIALIZABLE, false)
  var Transaction sql.Transaction
  Status := sql.Pool_Begin(&Pool, options, &Transaction)
  testify.Equal_Values(t, sql.STATUS_OK, Status, "begin commit")
  var Result sql.Result
  Status = sql.Status(sql.Transaction_Exec(&Transaction, test_request(t), &Result))
  testify.Equal_Values(t, sql.STATUS_OK, Status, "execute")
  var Rows sql.Rows
  Status = sql.Status(sql.Transaction_Query(&Transaction, test_request(t), &Rows))
  testify.Equal_Values(t, sql.STATUS_OK, Status, "query")
  testify.Equal_Values(
    t, sql.STATUS_BUSY, sql.Transaction_Commit(&Transaction), "active rows",
  )
  testify.Equal_Values(t, sql.STATUS_OK, sql.Rows_Close(&Rows), "rows close")
  testify.Equal_Values(t, sql.STATUS_OK, sql.Transaction_Commit(&Transaction), "commit")
  testify.Equal_Values(
    t,
    sql.STATUS_HANDLE_INVALID,
    sql.Transaction_Rollback(&Transaction),
    "terminal",
  )

  Status = sql.Pool_Begin(&Pool, options, &Transaction)
  testify.Equal_Values(t, sql.STATUS_OK, Status, "begin rollback")
  testify.Equal_Values(t, sql.STATUS_OK, sql.Transaction_Rollback(&Transaction), "rollback")
  testify.Equal_Values(t, sql.STATUS_OK, sql.Pool_Close(&Pool), "pool close")
}

// Test_Allocation protects every public operation without hidden ownership.
func Test_Allocation(t *testing.T) {
  data_source, validation := data_source_validate("memory")
  testify.Equal_Values(t, driver.STATUS_OK, validation, "data source")
  request := test_request(t)
  options := driver.Transaction_Options_Of(driver.ISOLATION_SERIALIZABLE, false)
  test_allocation_initialization(t, data_source)
  test_allocation_pool(t, data_source, request, options)
  test_allocation_connection(t, data_source, request, options)
  test_allocation_rejection(t, data_source, request, options)
}

// Test_Invariant_Domains reaches every registered pool boundary and state branch.
func Test_Invariant_Domains(t *testing.T) {
  test_pool_domains(t)
  test_status_storage_domains(t)
  test_connection_domains(t)
  test_transaction_option_domains(t)
  test_busy_domains(t)
  test_bad_connection_domains(t)
  test_handle_domains(t)
  test_collection_domains(t)
  test_result_domains()
  test_statistics_domains()
  test_pointer_storage_domains()
  test_request_operation_domains(t)
  test_pool_pointer_domains()
  test_zero_pool_pointer_domains(t)
  test_pool_initialization_storage_domains()
  test_output_storage_domains(t)
  test_statistics_result_domains(t)
  test_discard_pool_domains()
  test_initialization_connection_domains()
}

func test_status_storage_domains(t *testing.T) {
  State := fake_state{}
  Pool := test_pool(t, &State)
  Status := sql.STATUS_GENERATION_EXHAUSTED
  Connection := sql.Pool_Connection_Acquire(&Pool, &Status)
  sql.Connection_Close(&Connection)
  Status = sql.STATUS_DONE
  Connection = sql.Pool_Connection_Acquire(&Pool, &Status)
  sql.Connection_Close(&Connection)
  Status = sql.STATUS_INPUT_INVALID
  Connection = sql.Pool_Connection_Acquire(&Pool, &Status)
  sql.Connection_Close(&Connection)
  sql.Pool_Close(&Pool)

  var Result sql.Result
  optional := driver.Optional_Status(driver.STATUS_UNSUPPORTED)
  sql.Result_Rows_Affected(Result, &optional)
  optional = driver.Optional_Status(driver.STATUS_UNSUPPORTED)
  sql.Result_Last_Insert_Identifier(Result, &optional)
}

func test_transaction_option_domains(t *testing.T) {
  State := fake_state{}
  Pool := test_pool(t, &State)
  for _, isolation := range []driver.Isolation_Level{
    driver.ISOLATION_DEFAULT,
    driver.ISOLATION_READ_UNCOMMITTED,
    driver.ISOLATION_READ_COMMITTED,
    driver.ISOLATION_LINEARIZABLE,
  } {
    options := driver.Transaction_Options_Of(isolation, true)
    var Transaction sql.Transaction
    Status := sql.Pool_Begin(&Pool, options, &Transaction)
    testify.Equal_Values(t, sql.STATUS_OK, Status, "pool begin")
    testify.Equal_Values(
      t, sql.STATUS_OK, sql.Transaction_Rollback(&Transaction), "pool rollback",
    )
    Connection, Status := pool_connection_acquire(&Pool)
    testify.Equal_Values(t, sql.STATUS_OK, Status, "acquire")
    connection_status := sql.Connection_Begin(&Connection, options, &Transaction)
    testify.Equal_Values(t, sql.STATUS_OK, connection_status, "connection begin")
    connection_status = sql.Transaction_Rollback(&Transaction)
    testify.Equal_Values(
      t, sql.STATUS_OK, connection_status, "connection rollback",
    )
    testify.Equal_Values(t, sql.STATUS_OK, sql.Connection_Close(&Connection), "release")
  }
  testify.Equal_Values(t, sql.STATUS_OK, sql.Pool_Close(&Pool), "close")
}

type allocation_initialization_probe struct {
  State  fake_state
  Pool   sql.Pool
  Slots  []sql.Slot
  Status sql.Initialization_Status
}

func test_allocation_initialization(t *testing.T, data_source driver.Data_Source) {
  probe := allocation_initialization_probe{Slots: make([]sql.Slot, POOL_CAPACITY)}
  testify.Zero_Allocation(t, func() { allocation_initialization_run(&probe, data_source) })
  testify.Equal_Values(t, sql.STATUS_OK, probe.Status, "initialization")
}

func allocation_initialization_run(
  probe *allocation_initialization_probe,
  data_source driver.Data_Source,
) {
  clear(probe.Slots)
  *probe = allocation_initialization_probe{Slots: probe.Slots}
  probe.Status = sql.Pool_Init(
    &probe.Pool, fake_driver(&probe.State), data_source, probe.Slots,
  )
}

func allocation_statistics(Pool *sql.Pool) {
  statistics := sql.Pool_Statistics(Pool)
  sql.Statistics_Capacity(statistics)
  sql.Statistics_Open(statistics)
  sql.Statistics_Idle(statistics)
  sql.Statistics_In_Use(statistics)
}

type allocation_pool_probe struct {
  State       fake_state
  Pool        sql.Pool
  Slots       []sql.Slot
  Result      sql.Result
  Rows        sql.Rows
  Values      []driver.Value
  Statement   sql.Statement
  Transaction sql.Transaction
  Status      sql.Status
}

func test_allocation_pool(
  t *testing.T,
  data_source driver.Data_Source,
  request driver.Request,
  options driver.Transaction_Options,
) {
  probe := allocation_pool_probe{
    Slots:  make([]sql.Slot, POOL_CAPACITY),
    Values: make([]driver.Value, ROW_COLUMN_COUNT),
  }
  testify.Zero_Allocation(t, func() { allocation_pool_run(&probe, data_source, request, options) })
  testify.Equal_Values(t, sql.STATUS_OK, probe.Status, "pool public operations")
}

func allocation_pool_run(
  probe *allocation_pool_probe,
  data_source driver.Data_Source,
  request driver.Request,
  options driver.Transaction_Options,
) {
  clear(probe.Slots)
  clear(probe.Values)
  *probe = allocation_pool_probe{Slots: probe.Slots, Values: probe.Values}
  query := request.Query
  sql.Pool_Init(
    &probe.Pool, fake_driver(&probe.State), data_source, probe.Slots,
  )
  probe.Status = sql.Pool_Probe(&probe.Pool)
  probe.Status = sql.Pool_Exec(&probe.Pool, request, &probe.Result)
  result_rows_affected(probe.Result)
  result_last_insert_identifier(probe.Result)
  allocation_statistics(&probe.Pool)

  probe.Status = sql.Pool_Query(&probe.Pool, request, &probe.Rows)
  probe.Status = sql.Status(sql.Rows_Next(&probe.Rows, probe.Values))
  probe.Status = sql.Status(sql.Rows_Close(&probe.Rows))

  probe.Status = sql.Pool_Prepare(&probe.Pool, query, &probe.Statement)
  probe.Status = sql.Status(sql.Statement_Exec(&probe.Statement, nil, &probe.Result))
  probe.Status = sql.Status(sql.Statement_Query(&probe.Statement, nil, &probe.Rows))
  probe.Status = sql.Status(sql.Rows_Close(&probe.Rows))
  probe.Status = sql.Status(sql.Statement_Close(&probe.Statement))

  probe.Status = sql.Pool_Begin(&probe.Pool, options, &probe.Transaction)
  probe.Status = sql.Status(sql.Transaction_Exec(&probe.Transaction, request, &probe.Result))
  probe.Status = sql.Status(sql.Transaction_Query(&probe.Transaction, request, &probe.Rows))
  probe.Status = sql.Status(sql.Rows_Close(&probe.Rows))
  probe.Status = sql.Status(sql.Transaction_Commit(&probe.Transaction))
  probe.Status = sql.Pool_Begin(&probe.Pool, options, &probe.Transaction)
  probe.Status = sql.Status(sql.Transaction_Rollback(&probe.Transaction))
  probe.Status = sql.Status(sql.Pool_Close(&probe.Pool))
}

type allocation_connection_probe struct {
  State       fake_state
  Pool        sql.Pool
  Slots       []sql.Slot
  Connection  sql.Connection
  Result      sql.Result
  Rows        sql.Rows
  Values      []driver.Value
  Statement   sql.Statement
  Transaction sql.Transaction
  Status      sql.Status
}

func test_allocation_connection(
  t *testing.T,
  data_source driver.Data_Source,
  request driver.Request,
  options driver.Transaction_Options,
) {
  probe := allocation_connection_probe{
    Slots:  make([]sql.Slot, POOL_CAPACITY),
    Values: make([]driver.Value, ROW_COLUMN_COUNT),
  }
  testify.Zero_Allocation(t, func() {
    allocation_connection_run(&probe, data_source, request, options)
  })
  testify.Equal_Values(t, sql.STATUS_OK, probe.Status, "connection public operations")
}

func allocation_connection_run(
  probe *allocation_connection_probe,
  data_source driver.Data_Source,
  request driver.Request,
  options driver.Transaction_Options,
) {
  clear(probe.Slots)
  clear(probe.Values)
  *probe = allocation_connection_probe{Slots: probe.Slots, Values: probe.Values}
  query := request.Query
  sql.Pool_Init(
    &probe.Pool, fake_driver(&probe.State), data_source, probe.Slots,
  )
  allocation_acquire(&probe.Connection, &probe.Status, &probe.Pool)
  probe.Status = sql.Status(sql.Connection_Probe(&probe.Connection))
  probe.Status = sql.Status(sql.Connection_Exec(&probe.Connection, request, &probe.Result))
  probe.Status = sql.Status(sql.Connection_Query(&probe.Connection, request, &probe.Rows))
  probe.Status = sql.Status(sql.Rows_Next(&probe.Rows, probe.Values))
  probe.Status = sql.Status(sql.Rows_Close(&probe.Rows))

  probe.Status = sql.Status(sql.Connection_Prepare(&probe.Connection, query, &probe.Statement))
  probe.Status = sql.Status(sql.Statement_Close(&probe.Statement))
  probe.Status = sql.Status(sql.Connection_Begin(&probe.Connection, options, &probe.Transaction))
  probe.Status = sql.Status(sql.Transaction_Rollback(&probe.Transaction))
  probe.Status = sql.Status(sql.Connection_Close(&probe.Connection))
  probe.Status = sql.Status(sql.Pool_Close(&probe.Pool))
}

func allocation_acquire(Connection *sql.Connection, Status *sql.Status, Pool *sql.Pool) {
  *Connection, *Status = pool_connection_acquire(Pool)
}

func allocation_acquire_status(Status *sql.Status, Pool *sql.Pool) {
  _, *Status = pool_connection_acquire(Pool)
}

type allocation_rejection_probe struct {
  State           fake_state
  Pool            sql.Pool
  Slots           []sql.Slot
  First           sql.Connection
  Second          sql.Connection
  Zero_Connection sql.Connection
  Result          sql.Result
  Rows            sql.Rows
  Statement       sql.Statement
  Transaction     sql.Transaction
  Status          sql.Status
}

func test_allocation_rejection(
  t *testing.T,
  data_source driver.Data_Source,
  request driver.Request,
  options driver.Transaction_Options,
) {
  probe := allocation_rejection_probe{Slots: make([]sql.Slot, POOL_CAPACITY)}
  testify.Zero_Allocation(t, func() {
    allocation_rejection_run(&probe, data_source, request, options)
  })
  testify.Equal_Values(t, sql.STATUS_OK, probe.Status, "rejected public operations")
}

func allocation_rejection_run(
  probe *allocation_rejection_probe,
  data_source driver.Data_Source,
  request driver.Request,
  options driver.Transaction_Options,
) {
  clear(probe.Slots)
  *probe = allocation_rejection_probe{Slots: probe.Slots}
  query := request.Query
  sql.Pool_Init(
    &probe.Pool, fake_driver(&probe.State), data_source, probe.Slots,
  )
  allocation_acquire(&probe.First, &probe.Status, &probe.Pool)
  allocation_acquire(&probe.Second, &probe.Status, &probe.Pool)
  allocation_acquire_status(&probe.Status, &probe.Pool)
  probe.Status = sql.Status(sql.Connection_Probe(&probe.Zero_Connection))
  probe.Status = sql.Status(sql.Connection_Exec(&probe.Zero_Connection, request, &probe.Result))
  probe.Status = sql.Status(sql.Connection_Query(&probe.Zero_Connection, request, &probe.Rows))
  probe.Status = sql.Status(
    sql.Connection_Prepare(&probe.Zero_Connection, query, &probe.Statement),
  )
  probe.Status = sql.Status(
    sql.Connection_Begin(&probe.Zero_Connection, options, &probe.Transaction),
  )
  probe.Status = sql.Status(sql.Rows_Next(&probe.Rows, nil))
  probe.Status = sql.Status(sql.Rows_Close(&probe.Rows))
  probe.Status = sql.Status(sql.Statement_Exec(&probe.Statement, nil, &probe.Result))
  probe.Status = sql.Status(sql.Statement_Query(&probe.Statement, nil, &probe.Rows))
  probe.Status = sql.Status(sql.Statement_Close(&probe.Statement))
  probe.Status = sql.Status(sql.Transaction_Exec(&probe.Transaction, request, &probe.Result))
  probe.Status = sql.Status(sql.Transaction_Query(&probe.Transaction, request, &probe.Rows))
  probe.Status = sql.Status(sql.Transaction_Commit(&probe.Transaction))
  probe.Status = sql.Status(sql.Transaction_Rollback(&probe.Transaction))
  probe.Status = sql.Status(sql.Connection_Close(&probe.First))
  probe.Status = sql.Status(sql.Connection_Close(&probe.Second))
  probe.Status = sql.Status(sql.Pool_Close(&probe.Pool))
}

func test_pool_domains(t *testing.T) {
  request := test_request(t)
  query := request.Query
  options := driver.Transaction_Options_Of(driver.ISOLATION_SERIALIZABLE, false)
  for _, Status := range []driver.Status{
    driver.STATUS_DONE,
    driver.STATUS_INPUT_INVALID,
    driver.STATUS_UNSUPPORTED,
  } {
    State := fake_state{Return_Status: Status}
    Pool := test_pool(t, &State)
    var Result sql.Result
    var Rows sql.Rows
    var Statement sql.Statement
    var Transaction sql.Transaction
    pool_connection_acquire(&Pool)
    sql.Pool_Probe(&Pool)
    sql.Pool_Exec(&Pool, request, &Result)
    sql.Pool_Query(&Pool, request, &Rows)
    sql.Pool_Prepare(&Pool, query, &Statement)
    sql.Pool_Begin(&Pool, options, &Transaction)
  }

  State := fake_state{}
  Pool := test_pool(t, &State)
  for index := range Pool.Slots {
    Pool.Slots[index].Generation = sql.Generation(bits.WORD_64_MAXIMUM)
  }
  var Result sql.Result
  var Rows sql.Rows
  var Statement sql.Statement
  var Transaction sql.Transaction
  pool_connection_acquire(&Pool)
  sql.Pool_Probe(&Pool)
  sql.Pool_Exec(&Pool, request, &Result)
  sql.Pool_Query(&Pool, request, &Rows)
  sql.Pool_Prepare(&Pool, query, &Statement)
  sql.Pool_Begin(&Pool, options, &Transaction)
  test_pool_storage_domains(t)
  test_query_domains(t)
}

func test_pool_storage_domains(t *testing.T) {
  State := fake_state{}
  for _, count := range []int{
    NUMBER_ONE, NUMBER_TWO, sql.CONNECTION_COUNT_MAXIMUM,
  } {
    storage := make([]sql.Slot, count)
    var Pool sql.Pool
    data_source := driver.Data_Source(
      text_of(count % (strings.TEXT_SIZE_MAXIMUM + NUMBER_ONE)),
    )
    sql.Pool_Init(
      &Pool, fake_driver(&State), data_source, storage,
    )
    statistics := sql.Pool_Statistics(&Pool)
    sql.Statistics_Capacity(statistics)
  }
  for _, size := range []int{
    NUMBER_ZERO, NUMBER_ONE, NUMBER_TWO, strings.TEXT_SIZE_MAXIMUM,
  } {
    var Pool sql.Pool
    var Slots [POOL_CAPACITY]sql.Slot
    data_source := driver.Data_Source(text_of(size))
    sql.Pool_Init(
      &Pool, fake_driver(&State), data_source, Slots[:],
    )
    sql.Pool_Statistics(&Pool)
  }
}

func test_query_domains(t *testing.T) {
  for _, size := range []int{
    NUMBER_ZERO, NUMBER_ONE, NUMBER_TWO, strings.TEXT_SIZE_MAXIMUM,
  } {
    State := fake_state{}
    Pool := test_pool(t, &State)
    query := driver.Query(text_of(size))
    var Statement sql.Statement
    sql.Pool_Prepare(&Pool, query, &Statement)
    sql.Statement_Close(&Statement)
    Connection, _ := pool_connection_acquire(&Pool)
    sql.Connection_Prepare(&Connection, query, &Statement)
    sql.Statement_Close(&Statement)
    sql.Connection_Close(&Connection)
    sql.Pool_Close(&Pool)
  }
}

func test_connection_domains(t *testing.T) {
  request := test_request(t)
  query := request.Query
  options := driver.Transaction_Options_Of(driver.ISOLATION_SERIALIZABLE, false)
  for _, driver_status := range []driver.Status{
    driver.STATUS_DONE,
    driver.STATUS_INPUT_INVALID,
    driver.STATUS_UNSUPPORTED,
  } {
    State := fake_state{}
    Pool := test_pool(t, &State)
    Connection, _ := pool_connection_acquire(&Pool)
    State.Return_Status = driver_status
    var Result sql.Result
    var Rows sql.Rows
    var Statement sql.Statement
    var Transaction sql.Transaction
    sql.Connection_Probe(&Connection)
    sql.Connection_Exec(&Connection, request, &Result)
    sql.Connection_Query(&Connection, request, &Rows)
    sql.Connection_Prepare(&Connection, query, &Statement)
    sql.Connection_Begin(&Connection, options, &Transaction)
    State.Return_Status = driver.STATUS_OK
    test_rows_status(t, &Pool, &Connection, &State, request, driver_status)
    test_statement_status(&Connection, &State, query, driver_status)
    test_transaction_status(&Connection, &State, options, driver_status)
    sql.Connection_Close(&Connection)
    State.Return_Status = driver.STATUS_OK
    sql.Pool_Close(&Pool)
    test_pool_close_status(t, driver_status)
  }
}

func test_rows_status(
  t *testing.T,
  Pool *sql.Pool,
  Connection *sql.Connection,
  State *fake_state,
  request driver.Request,
  Status driver.Status,
) {
  var Rows sql.Rows
  sql.Connection_Query(Connection, request, &Rows)
  State.Return_Status = Status
  var Values [ROW_COLUMN_COUNT]driver.Value
  sql.Rows_Next(&Rows, Values[:])
  State.Return_Status = driver.STATUS_OK
  sql.Connection_Query(Connection, request, &Rows)
  State.Return_Status = Status
  sql.Rows_Close(&Rows)
  State.Return_Status = driver.STATUS_OK
  sql.Pool_Statistics(Pool)
}

func test_statement_status(
  Connection *sql.Connection,
  State *fake_state,
  query driver.Query,
  Status driver.Status,
) {
  var Statement sql.Statement
  var Result sql.Result
  var Rows sql.Rows
  sql.Connection_Prepare(Connection, query, &Statement)
  State.Return_Status = Status
  sql.Statement_Exec(&Statement, nil, &Result)
  sql.Statement_Query(&Statement, nil, &Rows)
  sql.Statement_Close(&Statement)
  State.Return_Status = driver.STATUS_OK
}

func test_transaction_status(
  Connection *sql.Connection,
  State *fake_state,
  options driver.Transaction_Options,
  Status driver.Status,
) {
  request := driver.Request{Query: ""}
  var Transaction sql.Transaction
  var Result sql.Result
  var Rows sql.Rows
  sql.Connection_Begin(Connection, options, &Transaction)
  State.Return_Status = Status
  sql.Transaction_Exec(&Transaction, request, &Result)
  sql.Transaction_Query(&Transaction, request, &Rows)
  sql.Transaction_Commit(&Transaction)
  State.Return_Status = driver.STATUS_OK
  sql.Connection_Begin(Connection, options, &Transaction)
  State.Return_Status = Status
  sql.Transaction_Rollback(&Transaction)
  State.Return_Status = driver.STATUS_OK
}

func test_pool_close_status(t *testing.T, Status driver.Status) {
  State := fake_state{}
  Pool := test_pool(t, &State)
  Connection, _ := pool_connection_acquire(&Pool)
  sql.Connection_Close(&Connection)
  State.Return_Status = Status
  sql.Pool_Close(&Pool)
}

func test_busy_domains(t *testing.T) {
  request := test_request(t)
  query := request.Query
  options := driver.Transaction_Options_Of(driver.ISOLATION_SERIALIZABLE, false)
  State := fake_state{}
  Pool := test_pool(t, &State)
  Connection, _ := pool_connection_acquire(&Pool)
  var Rows sql.Rows
  sql.Connection_Query(&Connection, request, &Rows)
  var Result sql.Result
  var Statement sql.Statement
  var Transaction sql.Transaction
  sql.Connection_Probe(&Connection)
  sql.Connection_Exec(&Connection, request, &Result)
  sql.Connection_Query(&Connection, request, &Rows)
  sql.Connection_Prepare(&Connection, query, &Statement)
  sql.Connection_Begin(&Connection, options, &Transaction)
  sql.Connection_Close(&Connection)
  sql.Rows_Close(&Rows)

  sql.Connection_Prepare(&Connection, query, &Statement)
  sql.Statement_Query(&Statement, nil, &Rows)
  sql.Statement_Exec(&Statement, nil, &Result)
  sql.Statement_Query(&Statement, nil, &Rows)
  sql.Statement_Close(&Statement)
  sql.Rows_Close(&Rows)
  sql.Statement_Close(&Statement)

  sql.Connection_Begin(&Connection, options, &Transaction)
  sql.Transaction_Query(&Transaction, request, &Rows)
  sql.Transaction_Exec(&Transaction, request, &Result)
  sql.Transaction_Query(&Transaction, request, &Rows)
  sql.Transaction_Commit(&Transaction)
  sql.Transaction_Rollback(&Transaction)
  sql.Rows_Close(&Rows)
  sql.Transaction_Rollback(&Transaction)

  sql.Connection_Query(&Connection, request, &Rows)
  slot := &Pool.Slots[NUMBER_ZERO]
  slot.State = sql.SLOT_TRANSACTION
  var Values [ROW_COLUMN_COUNT]driver.Value
  sql.Rows_Next(&Rows, Values[:])
  sql.Rows_Close(&Rows)
  slot.State = sql.SLOT_ROWS
  sql.Rows_Close(&Rows)
  sql.Connection_Close(&Connection)
  sql.Pool_Close(&Pool)
}

func test_bad_connection_domains(t *testing.T) {
  test_bad_connection_state(t, false, false)
  test_bad_connection_state(t, true, false)
  test_bad_connection_state(t, false, true)
  request := test_request(t)
  query := request.Query
  options := driver.Transaction_Options_Of(driver.ISOLATION_SERIALIZABLE, false)

  State := fake_state{}
  Pool := test_pool(t, &State)
  Connection, _ := pool_connection_acquire(&Pool)
  var Rows sql.Rows
  sql.Connection_Query(&Connection, request, &Rows)
  State.Return_Status = driver.STATUS_BAD_CONNECTION
  var Values [ROW_COLUMN_COUNT]driver.Value
  sql.Rows_Next(&Rows, Values[:])

  State = fake_state{}
  Pool = test_pool(t, &State)
  Connection, _ = pool_connection_acquire(&Pool)
  var closing_rows sql.Rows
  sql.Connection_Query(&Connection, request, &closing_rows)
  State.Return_Status = driver.STATUS_BAD_CONNECTION
  sql.Rows_Close(&closing_rows)

  State = fake_state{}
  Pool = test_pool(t, &State)
  Connection, _ = pool_connection_acquire(&Pool)
  var Statement sql.Statement
  sql.Connection_Prepare(&Connection, query, &Statement)
  State.Return_Status = driver.STATUS_BAD_CONNECTION
  var Result sql.Result
  sql.Statement_Exec(&Statement, nil, &Result)

  State = fake_state{}
  Pool = test_pool(t, &State)
  Connection, _ = pool_connection_acquire(&Pool)
  var Transaction sql.Transaction
  sql.Connection_Begin(&Connection, options, &Transaction)
  State.Return_Status = driver.STATUS_BAD_CONNECTION
  sql.Transaction_Exec(&Transaction, request, &Result)
}

func test_bad_connection_state(
  t *testing.T, mutate_generation bool, mutate_state bool,
) {
  State := fake_state{}
  Pool := test_pool(t, &State)
  Connection, _ := pool_connection_acquire(&Pool)
  State.Slots = Pool.Slots
  State.Mutate_Generation = mutate_generation
  State.Mutate_State = mutate_state
  State.Return_Status = driver.STATUS_BAD_CONNECTION
  sql.Connection_Probe(&Connection)
}

func test_handle_domains(t *testing.T) {
  State := fake_state{}
  var zero sql.Connection
  sql.Connection_Close(&zero)
  Pool := test_pool(t, &State)
  closed, _ := pool_connection_acquire(&Pool)
  Pool.Closed = sql.POOL_CLOSED
  sql.Connection_Close(&closed)
  test_transition_domains(t)
  test_handle_boundary_domains(t)
  test_connection_of_domains(t)
}

func test_transition_domains(t *testing.T) {
  for _, mutation := range []struct {
    Generation bool
    State      bool
  }{{Generation: true}, {State: true}} {
    State := fake_state{}
    Pool := test_pool(t, &State)
    Connection, _ := pool_connection_acquire(&Pool)
    State.Slots = Pool.Slots
    State.Mutate_Generation = mutation.Generation
    State.Mutate_State = mutation.State
    sql.Connection_Probe(&Connection)
  }
}

func test_handle_boundary_domains(t *testing.T) {
  State := fake_state{}
  storage := make([]sql.Slot, sql.CONNECTION_COUNT_MAXIMUM)
  var Pool sql.Pool
  data_source := driver.Data_Source("memory")
  sql.Pool_Init(
    &Pool, fake_driver(&State), data_source, storage,
  )
  for _, index := range []sql.Slot_Index{
    NUMBER_ZERO,
    NUMBER_ONE,
    NUMBER_TWO,
    sql.SLOT_INDEX_MAXIMUM,
  } {
    for _, generation := range []sql.Lease_Generation{
      NUMBER_ONE,
      NUMBER_TWO,
      sql.Lease_Generation(bits.WORD_64_MAXIMUM),
    } {
      slot := &storage[index]
      fake_connect(
        unsafe.Pointer(&State), data_source,
        &slot.Connection,
      )
      slot.State = sql.SLOT_CONNECTION
      slot.Generation = sql.Generation(generation)
      Connection := sql.Connection{
        Pool:       unsafe.Pointer(&Pool),
        Slot_Index: index,
        Generation: sql.Generation(generation),
      }
      State.Return_Status = driver.STATUS_OK
      sql.Connection_Probe(&Connection)
      State.Return_Status = driver.STATUS_BAD_CONNECTION
      sql.Connection_Probe(&Connection)
      State.Return_Status = driver.STATUS_OK
    }
  }
}

func test_connection_of_domains(t *testing.T) {
  State := fake_state{}
  storage := make([]sql.Slot, sql.CONNECTION_COUNT_MAXIMUM)
  for index := NUMBER_ZERO; index < len(storage)-NUMBER_ONE; index++ {
    storage[index].State = sql.SLOT_CONNECTION
  }
  var Pool sql.Pool
  sql.Pool_Init(
    &Pool, fake_driver(&State), driver.Data_Source("memory"),
    storage,
  )
  for index := NUMBER_ZERO; index < len(storage)-NUMBER_ONE; index++ {
    storage[index].State = sql.SLOT_CONNECTION
  }
  pool_connection_acquire(&Pool)

  Pool = sql.Pool{}
  sql.Pool_Init(
    &Pool, fake_driver(&State), driver.Data_Source("memory"),
    storage,
  )
  for index := range storage {
    if index != NUMBER_TWO {
      storage[index].State = sql.SLOT_CONNECTION
    }
  }
  pool_connection_acquire(&Pool)

  var single [sql.INITIALIZED_CONNECTION_COUNT_MINIMUM]sql.Slot
  Pool = sql.Pool{}
  sql.Pool_Init(
    &Pool, fake_driver(&State), driver.Data_Source("memory"),
    single[:],
  )
  fake_connect(
    unsafe.Pointer(&State),
    driver.Data_Source("memory"),
    &single[NUMBER_ZERO].Connection,
  )
  single[NUMBER_ZERO].State = sql.SLOT_IDLE
  single[NUMBER_ZERO].Generation = sql.Generation(
    bits.WORD_64_MAXIMUM - NUMBER_ONE,
  )
  pool_connection_acquire(&Pool)
}

func test_collection_domains(t *testing.T) {
  State := fake_state{}
  Pool := test_pool(t, &State)
  Connection, _ := pool_connection_acquire(&Pool)
  request := test_request(t)
  var Rows sql.Rows
  sql.Connection_Query(&Connection, request, &Rows)
  for _, count := range []int{NUMBER_TWO, sql.CONNECTION_COUNT_MAXIMUM} {
    sql.Rows_Next(&Rows, make(driver.Values, count))
  }
  sql.Rows_Close(&Rows)

  query := request.Query
  var Statement sql.Statement
  sql.Connection_Prepare(&Connection, query, &Statement)
  var Result sql.Result
  for _, count := range []int{
    NUMBER_ONE, NUMBER_TWO, driver.ARGUMENT_COUNT_MAXIMUM,
  } {
    arguments := make(driver.Arguments, count)
    sql.Statement_Exec(&Statement, arguments, &Result)
    sql.Statement_Query(&Statement, arguments, &Rows)
  }
  sql.Statement_Close(&Statement)
  sql.Connection_Close(&Connection)
  sql.Pool_Close(&Pool)
}

func test_result_domains() {
  for _, value := range []int64{
    bits.INTEGER_64_MINIMUM,
    NUMBER_NEGATIVE_ONE,
    NUMBER_ZERO,
    NUMBER_ONE,
    NUMBER_TWO,
    bits.INTEGER_64_MAXIMUM,
  } {
    var Result sql.Result
    driver.Result_Set_Last_Insert_Identifier(
      &Result.Driver_Result,
      driver.Last_Insert_Identifier(value),
    )
    result_last_insert_identifier(Result)
    driver.Result_Set_Rows_Affected(
      &Result.Driver_Result, driver.Rows_Affected(value),
    )
    result_last_insert_identifier(Result)
    result_rows_affected(Result)
  }
  var Result sql.Result
  result_last_insert_identifier(Result)
  result_rows_affected(Result)
}

func test_statistics_domains() {
  for _, count := range []int{
    NUMBER_ZERO, NUMBER_ONE, NUMBER_TWO, sql.CONNECTION_COUNT_MAXIMUM,
  } {
    statistics := sql.Statistics{
      Capacity:     sql.Capacity_Count(count),
      Open_Count:   sql.Open_Count(count),
      Idle_Count:   sql.Idle_Count(count),
      In_Use_Count: sql.In_Use_Count(count),
    }
    sql.Statistics_Capacity(statistics)
    sql.Statistics_Open(statistics)
    sql.Statistics_Idle(statistics)
    sql.Statistics_In_Use(statistics)
  }
}

func test_pointer_storage_domains() {
  test_connection_pointer_storage_domains()
  test_rows_pointer_storage_domains()
  test_statement_pointer_storage_domains()
  test_transaction_pointer_storage_domains()
}

func test_connection_pointer_storage_domains() {
  request := driver.Request{}
  options := driver.Transaction_Options_Of(driver.ISOLATION_DEFAULT, false)
  indexes := []sql.Slot_Index{
    NUMBER_ZERO, NUMBER_ONE, NUMBER_TWO, sql.SLOT_INDEX_MAXIMUM,
  }
  generations := []sql.Generation{
    sql.Generation(bits.WORD_64_MINIMUM),
    NUMBER_ONE,
    NUMBER_TWO,
    sql.Generation(bits.WORD_64_MAXIMUM),
  }
  for index := range indexes {
    Connection := sql.Connection{
      Slot_Index: indexes[index], Generation: generations[index],
    }
    var Result sql.Result
    var Rows sql.Rows
    var Statement sql.Statement
    var Transaction sql.Transaction
    sql.Connection_Probe(&Connection)
    sql.Connection_Exec(&Connection, request, &Result)
    sql.Connection_Query(&Connection, request, &Rows)
    sql.Connection_Prepare(&Connection, "", &Statement)
    sql.Connection_Begin(&Connection, options, &Transaction)
    sql.Connection_Close(&Connection)
  }
}

func test_rows_pointer_storage_domains() {
  indexes := []sql.Slot_Index{
    NUMBER_ZERO, NUMBER_ONE, NUMBER_TWO, sql.SLOT_INDEX_MAXIMUM,
  }
  generations := []sql.Generation{
    sql.Generation(bits.WORD_64_MINIMUM),
    NUMBER_ONE,
    NUMBER_TWO,
    sql.Generation(bits.WORD_64_MAXIMUM),
  }
  column_counts := []driver.Column_Count{
    NUMBER_ZERO,
    NUMBER_ONE,
    NUMBER_TWO,
    slices.SLICE_COUNT_MAXIMUM,
  }
  return_states := []sql.Slot_State{
    sql.SLOT_EMPTY,
    sql.SLOT_CONNECTION_CREATION,
    sql.SLOT_IDLE,
    sql.SLOT_CLOSE,
  }
  for index := range indexes {
    Connection := sql.Connection{
      Slot_Index: indexes[index], Generation: generations[index],
    }
    Rows := sql.Rows{
      Connection: Connection,
      Driver_Rows: driver.Rows{
        Column_Count: column_counts[index],
      },
      Return_State: return_states[index],
    }
    sql.Rows_Next(&Rows, nil)
    Rows.Connection = Connection
    sql.Rows_Close(&Rows)
  }
}

func test_statement_pointer_storage_domains() {
  indexes := []sql.Slot_Index{
    NUMBER_ZERO, NUMBER_ONE, NUMBER_TWO, sql.SLOT_INDEX_MAXIMUM,
  }
  generations := []sql.Generation{
    sql.Generation(bits.WORD_64_MINIMUM),
    NUMBER_ONE,
    NUMBER_TWO,
    sql.Generation(bits.WORD_64_MAXIMUM),
  }
  return_states := []sql.Slot_State{
    sql.SLOT_EMPTY,
    sql.SLOT_CONNECTION_CREATION,
    sql.SLOT_IDLE,
    sql.SLOT_CLOSE,
  }
  for index := range indexes {
    Connection := sql.Connection{
      Slot_Index: indexes[index], Generation: generations[index],
    }
    statement_count := driver.Statement_Argument_Count(indexes[index])
    if index == NUMBER_ZERO {
      statement_count = driver.STATEMENT_ARGUMENT_COUNT_UNKNOWN
    }
    Statement := sql.Statement{
      Connection: Connection,
      Driver_Statement: driver.Statement{
        Argument_Count: statement_count,
      },
      Return_State: return_states[index],
    }
    var Result sql.Result
    var Rows sql.Rows
    sql.Statement_Exec(&Statement, nil, &Result)
    Statement.Connection = Connection
    sql.Statement_Query(&Statement, nil, &Rows)
    Statement.Connection = Connection
    sql.Statement_Close(&Statement)
  }
  test_maximum_statement_pointer_storage_domain()
}

func test_transaction_pointer_storage_domains() {
  request := driver.Request{}
  indexes := []sql.Slot_Index{
    NUMBER_ZERO, NUMBER_ONE, NUMBER_TWO, sql.SLOT_INDEX_MAXIMUM,
  }
  generations := []sql.Generation{
    sql.Generation(bits.WORD_64_MINIMUM),
    NUMBER_ONE,
    NUMBER_TWO,
    sql.Generation(bits.WORD_64_MAXIMUM),
  }
  return_states := []sql.Slot_State{
    sql.SLOT_EMPTY,
    sql.SLOT_CONNECTION_CREATION,
    sql.SLOT_IDLE,
    sql.SLOT_CLOSE,
  }
  for index := range indexes {
    Connection := sql.Connection{
      Slot_Index: indexes[index], Generation: generations[index],
    }
    driver_transaction := driver.Transaction{}
    if index != NUMBER_ZERO {
      driver_transaction.Commit_Procedure = fake_commit
    }
    Transaction := sql.Transaction{
      Connection:         Connection,
      Driver_Transaction: driver_transaction,
      Return_State:       return_states[index],
    }
    var Result sql.Result
    var Rows sql.Rows
    sql.Transaction_Exec(&Transaction, request, &Result)
    Transaction.Connection = Connection
    sql.Transaction_Query(&Transaction, request, &Rows)
    Transaction.Connection = Connection
    sql.Transaction_Commit(&Transaction)
    Transaction.Connection = Connection
    sql.Transaction_Rollback(&Transaction)
  }
}

func test_maximum_statement_pointer_storage_domain() {
  Connection := sql.Connection{
    Slot_Index: sql.SLOT_INDEX_MAXIMUM,
    Generation: sql.Generation(bits.WORD_64_MAXIMUM),
  }
  Statement := sql.Statement{
    Connection: Connection,
    Driver_Statement: driver.Statement{
      Argument_Count: driver.ARGUMENT_COUNT_MAXIMUM,
    },
    Return_State: sql.SLOT_CLOSE,
  }
  var Result sql.Result
  var Rows sql.Rows
  sql.Statement_Exec(&Statement, nil, &Result)
  Statement.Connection = Connection
  sql.Statement_Query(&Statement, nil, &Rows)
  Statement.Connection = Connection
  sql.Statement_Close(&Statement)
}

func test_request_operation_domains(t *testing.T) {
  arguments := test_driver_arguments(driver.ARGUMENT_COUNT_MAXIMUM)
  for _, size := range []int{
    NUMBER_ZERO, NUMBER_ONE, NUMBER_TWO, strings.TEXT_SIZE_MAXIMUM,
  } {
    exercise_request_operations(t, driver.Request{Query: driver.Query(text_of(size))})
  }
  for _, size := range []int{
    NUMBER_ZERO, NUMBER_ONE, NUMBER_TWO, driver.ARGUMENT_COUNT_MAXIMUM,
  } {
    exercise_request_operations(t, driver.Request{Arguments: arguments[:size]})
  }
}

func exercise_request_operations(t *testing.T, request driver.Request) {
  State := fake_state{}
  Pool := test_pool(t, &State)
  var Result sql.Result
  var Rows sql.Rows
  sql.Pool_Exec(&Pool, request, &Result)
  Pool = test_pool(t, &State)
  sql.Pool_Query(&Pool, request, &Rows)

  var Connection sql.Connection
  sql.Connection_Exec(&Connection, request, &Result)
  sql.Connection_Query(&Connection, request, &Rows)
  var Transaction sql.Transaction
  sql.Transaction_Exec(&Transaction, request, &Result)
  sql.Transaction_Query(&Transaction, request, &Rows)
}

func test_pool_pointer_domains() {
  data_sizes := []int{
    NUMBER_ZERO,
    NUMBER_ONE,
    NUMBER_TWO,
    strings.TEXT_SIZE_MAXIMUM,
    strings.TEXT_SIZE_MAXIMUM,
  }
  slot_counts := []int{
    NUMBER_ONE, NUMBER_TWO, sql.CONNECTION_COUNT_MAXIMUM,
    NUMBER_ONE, NUMBER_ONE,
  }
  closed := []bool{false, false, false, false, true}
  for index := range data_sizes {
    exercise_pool_pointer_domain(
      data_sizes[index], slot_counts[index], closed[index],
    )
  }
}

// Nonnil zero pools reach structural bounds before initialized-state rejection.
func test_zero_pool_pointer_domains(t *testing.T) {
  testify.Panics(t, func() {
    pool_connection_acquire(&sql.Pool{})
  }, "zero pool acquire")
  testify.Panics(t, func() {
    sql.Pool_Probe(&sql.Pool{})
  }, "zero pool probe")
  testify.Panics(t, func() {
    sql.Pool_Exec(&sql.Pool{}, driver.Request{}, &sql.Result{})
  }, "zero pool execute")
  testify.Panics(t, func() {
    sql.Pool_Query(&sql.Pool{}, driver.Request{}, &sql.Rows{})
  }, "zero pool query")
  testify.Panics(t, func() {
    sql.Pool_Prepare(&sql.Pool{}, "", &sql.Statement{})
  }, "zero pool prepare")
  testify.Panics(t, func() {
    sql.Pool_Begin(
      &sql.Pool{},
      driver.Transaction_Options_Of(driver.ISOLATION_DEFAULT, false),
      &sql.Transaction{},
    )
  }, "zero pool begin")
  testify.Panics(t, func() {
    sql.Pool_Close(&sql.Pool{})
  }, "zero pool close")
  testify.Panics(t, func() {
    sql.Connection_Probe(&sql.Connection{Pool: unsafe.Pointer(&sql.Pool{})})
  }, "zero pool handle")
}

func exercise_pool_pointer_domain(
  data_size int, slot_count int, closed bool,
) {
  request := driver.Request{}
  options := driver.Transaction_Options_Of(driver.ISOLATION_DEFAULT, false)

  Pool := domain_pool(data_size, slot_count, closed)
  pool_connection_acquire(&Pool)
  Pool = domain_pool(data_size, slot_count, closed)
  sql.Pool_Probe(&Pool)
  Pool = domain_pool(data_size, slot_count, closed)
  var Result sql.Result
  sql.Pool_Exec(&Pool, request, &Result)
  Pool = domain_pool(data_size, slot_count, closed)
  var Rows sql.Rows
  sql.Pool_Query(&Pool, request, &Rows)
  Pool = domain_pool(data_size, slot_count, closed)
  var Statement sql.Statement
  sql.Pool_Prepare(&Pool, "", &Statement)
  Pool = domain_pool(data_size, slot_count, closed)
  var Transaction sql.Transaction
  sql.Pool_Begin(&Pool, options, &Transaction)
  Pool = domain_pool(data_size, slot_count, closed)
  sql.Pool_Close(&Pool)
  Pool = domain_pool(data_size, slot_count, closed)
  sql.Pool_Statistics(&Pool)
}

func domain_pool(data_size int, slot_count int, closed bool) (Pool sql.Pool) {
  State := fake_state{}
  storage := make([]sql.Slot, slot_count)
  sql.Pool_Init(
    &Pool, fake_driver(&State), driver.Data_Source(text_of(data_size)),
    storage,
  )
  Pool.Closed = sql.POOL_OPEN
  if closed {
    Pool.Closed = sql.POOL_CLOSED
  }
  return Pool
}

func test_pool_initialization_storage_domains() {
  State := fake_state{}
  data_sizes := []int{
    NUMBER_ZERO, NUMBER_ONE, NUMBER_TWO, strings.TEXT_SIZE_MAXIMUM,
  }
  slot_counts := []int{
    NUMBER_ZERO, NUMBER_ONE, NUMBER_TWO, sql.CONNECTION_COUNT_MAXIMUM,
  }
  for index := range data_sizes {
    closed := sql.POOL_OPEN
    if index == NUMBER_ONE {
      closed = sql.POOL_CLOSED
    }
    Pool := sql.Pool{
      Driver:      fake_driver(&State),
      Data_Source: driver.Data_Source(text_of(data_sizes[index])),
      Slots:       make(sql.Slot_Storage, slot_counts[index]),
      Closed:      closed,
    }
    var storage [POOL_CAPACITY]sql.Slot
    sql.Pool_Init(
      &Pool, fake_driver(&State), "", storage[:],
    )
  }

  states := []sql.Slot_State{
    sql.SLOT_EMPTY,
    sql.SLOT_CONNECTION_CREATION,
    sql.SLOT_IDLE,
    sql.SLOT_CLOSE,
  }
  generations := []sql.Generation{
    sql.Generation(bits.WORD_64_MINIMUM),
    NUMBER_ONE,
    NUMBER_TWO,
    sql.Generation(bits.WORD_64_MAXIMUM),
  }
  storage := make([]sql.Slot, len(states))
  for index := range storage {
    storage[index].State = states[index]
    storage[index].Generation = generations[index]
    if index != NUMBER_ZERO {
      fake_connect(
        unsafe.Pointer(&State), "", &storage[index].Connection,
      )
    }
  }
  var Pool sql.Pool
  sql.Pool_Init(
    &Pool, fake_driver(&State), "", storage,
  )
}

func test_output_storage_domains(t *testing.T) {
  test_result_output_storage_domains(t)
  test_rows_output_storage_domains(t)
  test_statement_output_storage_domains(t)
  test_transaction_output_storage_domains(t)
}

func test_result_output_storage_domains(t *testing.T) {
  request := driver.Request{}
  for index, value := range []int64{
    bits.INTEGER_64_MINIMUM,
    NUMBER_NEGATIVE_ONE,
    NUMBER_ZERO,
    NUMBER_ONE,
    NUMBER_TWO,
    bits.INTEGER_64_MAXIMUM,
  } {
    stored := sql.Result{
      Driver_Result: driver.Result{
        Last_Insert_Identifier:          driver.Last_Insert_Identifier(value),
        Rows_Affected:                   driver.Rows_Affected(value),
        Last_Insert_Identifier_Validity: index != NUMBER_ZERO,
        Rows_Affected_Validity:          index != NUMBER_ZERO,
      },
    }
    var Connection sql.Connection
    Result := stored
    sql.Connection_Exec(&Connection, request, &Result)
    Statement := sql.Statement{}
    Result = stored
    sql.Statement_Exec(&Statement, nil, &Result)
    Transaction := sql.Transaction{}
    Result = stored
    sql.Transaction_Exec(&Transaction, request, &Result)
    State := fake_state{}
    Pool := test_pool(t, &State)
    Result = stored
    sql.Pool_Exec(&Pool, request, &Result)
  }
}

func test_rows_output_storage_domains(t *testing.T) {
  request := driver.Request{}
  indexes := []sql.Slot_Index{
    NUMBER_ZERO, NUMBER_ONE, NUMBER_TWO, sql.SLOT_INDEX_MAXIMUM,
  }
  generations := []sql.Generation{
    sql.Generation(bits.WORD_64_MINIMUM),
    NUMBER_ONE,
    NUMBER_TWO,
    sql.Generation(bits.WORD_64_MAXIMUM),
  }
  states := []sql.Slot_State{
    sql.SLOT_EMPTY,
    sql.SLOT_CONNECTION_CREATION,
    sql.SLOT_IDLE,
    sql.SLOT_CLOSE,
  }
  column_counts := []driver.Column_Count{
    NUMBER_ZERO,
    NUMBER_ONE,
    NUMBER_TWO,
    slices.SLICE_COUNT_MAXIMUM,
  }
  for index := range indexes {
    stored := sql.Rows{
      Connection: sql.Connection{
        Slot_Index: indexes[index], Generation: generations[index],
      },
      Driver_Rows: driver.Rows{
        Column_Count: column_counts[index],
      },
      Return_State: states[index],
    }
    var Connection sql.Connection
    Rows := stored
    sql.Connection_Query(&Connection, request, &Rows)
    State := fake_state{}
    Pool := test_pool(t, &State)
    Rows = stored
    sql.Pool_Query(&Pool, request, &Rows)

    Pool = test_pool(t, &State)
    var Statement sql.Statement
    sql.Pool_Prepare(&Pool, "", &Statement)
    Rows = stored
    sql.Statement_Query(&Statement, nil, &Rows)

    Pool = test_pool(t, &State)
    var Transaction sql.Transaction
    options := driver.Transaction_Options_Of(driver.ISOLATION_DEFAULT, false)
    sql.Pool_Begin(&Pool, options, &Transaction)
    Rows = stored
    sql.Transaction_Query(&Transaction, request, &Rows)
  }
}

func test_statement_output_storage_domains(t *testing.T) {
  counts := []driver.Statement_Argument_Count{
    driver.STATEMENT_ARGUMENT_COUNT_UNKNOWN,
    NUMBER_ZERO,
    NUMBER_ONE,
    NUMBER_TWO,
    driver.ARGUMENT_COUNT_MAXIMUM,
  }
  indexes := []sql.Slot_Index{
    NUMBER_ZERO,
    NUMBER_ONE,
    NUMBER_TWO,
    NUMBER_ZERO,
    sql.SLOT_INDEX_MAXIMUM,
  }
  generations := []sql.Generation{
    sql.Generation(bits.WORD_64_MINIMUM),
    NUMBER_ONE,
    NUMBER_TWO,
    sql.Generation(bits.WORD_64_MINIMUM),
    sql.Generation(bits.WORD_64_MAXIMUM),
  }
  states := []sql.Slot_State{
    sql.SLOT_EMPTY,
    sql.SLOT_CONNECTION_CREATION,
    sql.SLOT_IDLE,
    sql.SLOT_EMPTY,
    sql.SLOT_CLOSE,
  }
  for index, count := range counts {
    stored := sql.Statement{
      Connection: sql.Connection{
        Slot_Index: indexes[index], Generation: generations[index],
      },
      Driver_Statement: driver.Statement{Argument_Count: count},
      Return_State:     states[index],
    }
    var Connection sql.Connection
    Statement := stored
    sql.Connection_Prepare(&Connection, "", &Statement)
    State := fake_state{}
    Pool := test_pool(t, &State)
    Statement = stored
    sql.Pool_Prepare(&Pool, "", &Statement)
  }
}

func test_transaction_output_storage_domains(t *testing.T) {
  options := driver.Transaction_Options_Of(driver.ISOLATION_DEFAULT, false)
  indexes := []sql.Slot_Index{
    NUMBER_ZERO, NUMBER_ONE, NUMBER_TWO, sql.SLOT_INDEX_MAXIMUM,
  }
  generations := []sql.Generation{
    sql.Generation(bits.WORD_64_MINIMUM),
    NUMBER_ONE,
    NUMBER_TWO,
    sql.Generation(bits.WORD_64_MAXIMUM),
  }
  states := []sql.Slot_State{
    sql.SLOT_EMPTY,
    sql.SLOT_CONNECTION_CREATION,
    sql.SLOT_IDLE,
    sql.SLOT_CLOSE,
  }
  for index := range indexes {
    driver_transaction := driver.Transaction{}
    if index != NUMBER_ZERO {
      driver_transaction.Commit_Procedure = fake_commit
    }
    stored := sql.Transaction{
      Connection: sql.Connection{
        Slot_Index: indexes[index], Generation: generations[index],
      },
      Driver_Transaction: driver_transaction,
      Return_State:       states[index],
    }
    var Connection sql.Connection
    Transaction := stored
    sql.Connection_Begin(&Connection, options, &Transaction)
    State := fake_state{}
    Pool := test_pool(t, &State)
    Transaction = stored
    sql.Pool_Begin(&Pool, options, &Transaction)
  }
}

func test_statistics_result_domains(t *testing.T) {
  testify.Panics(t, func() {
    sql.Pool_Statistics(&sql.Pool{})
  }, "zero pool statistics")
  for _, count := range []int{
    NUMBER_ONE, NUMBER_TWO, sql.CONNECTION_COUNT_MAXIMUM,
  } {
    State := fake_state{}
    Pool := domain_pool(NUMBER_ZERO, count, false)
    for index := range Pool.Slots {
      Pool.Slots[index].State = sql.SLOT_IDLE
      fake_connect(
        unsafe.Pointer(&State), "", &Pool.Slots[index].Connection,
      )
    }
    sql.Pool_Statistics(&Pool)

    Pool = domain_pool(NUMBER_ZERO, count, false)
    for index := range Pool.Slots {
      Pool.Slots[index].State = sql.SLOT_CONNECTION
    }
    sql.Pool_Statistics(&Pool)
  }
}

func test_discard_pool_domains() {
  data_sizes := []int{
    NUMBER_ZERO, NUMBER_ONE, NUMBER_TWO, strings.TEXT_SIZE_MAXIMUM,
  }
  slot_counts := []int{
    NUMBER_ONE, NUMBER_TWO, sql.CONNECTION_COUNT_MAXIMUM, NUMBER_ONE,
  }
  for index := range data_sizes {
    State := fake_state{}
    storage := make([]sql.Slot, slot_counts[index])
    var Pool sql.Pool
    sql.Pool_Init(
      &Pool,
      fake_driver(&State),
      driver.Data_Source(text_of(data_sizes[index])),
      storage,
    )
    Connection, _ := pool_connection_acquire(&Pool)
    State.Return_Status = driver.STATUS_BAD_CONNECTION
    sql.Connection_Probe(&Connection)
  }
}

func test_initialization_connection_domains() {
  indexes := []sql.Slot_Index{
    NUMBER_ZERO, NUMBER_ONE, NUMBER_TWO, sql.SLOT_INDEX_MAXIMUM,
  }
  generations := []sql.Generation{
    NUMBER_ONE,
    NUMBER_TWO,
    sql.Generation(bits.WORD_64_MAXIMUM),
    NUMBER_ONE,
  }
  request := driver.Request{}
  options := driver.Transaction_Options_Of(driver.ISOLATION_DEFAULT, false)
  for index := range indexes {
    State := fake_state{}
    storage := make([]sql.Slot, sql.CONNECTION_COUNT_MAXIMUM)
    Pool := sql.Pool{
      Driver:      fake_driver(&State),
      Data_Source: "",
      Slots:       storage,
      Closed:      sql.POOL_OPEN,
    }
    slot := &storage[indexes[index]]
    fake_connect(unsafe.Pointer(&State), "", &slot.Connection)
    slot.State = sql.SLOT_CONNECTION
    slot.Generation = generations[index]
    Connection := sql.Connection{
      Pool:       unsafe.Pointer(&Pool),
      Slot_Index: indexes[index],
      Generation: generations[index],
    }
    var Rows sql.Rows
    sql.Connection_Query(&Connection, request, &Rows)
    sql.Rows_Close(&Rows)
    var Statement sql.Statement
    sql.Connection_Prepare(&Connection, "", &Statement)
    sql.Statement_Close(&Statement)
    var Transaction sql.Transaction
    sql.Connection_Begin(&Connection, options, &Transaction)
    sql.Transaction_Rollback(&Transaction)
  }
}

func test_driver_arguments(count int) (arguments driver.Arguments) {
  arguments = make(driver.Arguments, count)
  for index := range arguments {
    var value driver.Value
    driver.Value_Null(&value)
    driver.Named_Value_Of(
      &arguments[index],
      "",
      driver.Argument_Ordinal_Unvalidated(index+NUMBER_ONE),
      value,
    )
  }
  return arguments
}

const NUMBER_ZERO = slices.COUNT_MINIMUM
const NUMBER_ONE = NUMBER_ZERO + 1
const NUMBER_TWO = NUMBER_ONE + NUMBER_ONE
const NUMBER_NEGATIVE_ONE = -NUMBER_ONE
const POOL_CAPACITY = NUMBER_TWO
const ROW_COLUMN_COUNT = NUMBER_ONE

type fake_state struct {
  Return_Status     driver.Status
  Row_Position      int
  Connect_Count     int
  Close_Count       int
  Slots             sql.Slot_Storage
  Mutate_Generation bool
  Mutate_State      bool
}

func text_of(size int) (text string) {
  return string(make([]byte, size))
}

func data_source_validate(
  unvalidated driver.Data_Source_Unvalidated,
) (data_source driver.Data_Source, Status driver.Validation_Status) {
  data_source = driver.Data_Source_Validate(unvalidated, &Status)
  return data_source, Status
}

func query_validate(
  unvalidated driver.Query_Unvalidated,
) (query driver.Query, Status driver.Validation_Status) {
  query = driver.Query_Validate(unvalidated, &Status)
  return query, Status
}

func request_of(
  query driver.Query,
  arguments driver.Arguments,
) (request driver.Request, Status driver.Validation_Status) {
  request = driver.Request_Of(query, arguments, &Status)
  return request, Status
}

func pool_connection_acquire(
  Pool sql.Pool_Pointer,
) (Connection sql.Connection, Status sql.Status) {
  Connection = sql.Pool_Connection_Acquire(Pool, &Status)
  return Connection, Status
}

func result_rows_affected(
  Result sql.Result,
) (count driver.Rows_Affected, Status driver.Optional_Status) {
  count = sql.Result_Rows_Affected(Result, &Status)
  return count, Status
}

func result_last_insert_identifier(
  Result sql.Result,
) (identifier driver.Last_Insert_Identifier, Status driver.Optional_Status) {
  identifier = sql.Result_Last_Insert_Identifier(Result, &Status)
  return identifier, Status
}

func test_pool(t *testing.T, State *fake_state) (Pool sql.Pool) {
  t.Helper()
  data_source, validation := data_source_validate("memory")
  testify.Equal_Values(t, driver.STATUS_OK, validation, "data source")
  var Slots [POOL_CAPACITY]sql.Slot
  Status := sql.Pool_Init(
    &Pool, fake_driver(State), data_source, Slots[:],
  )
  testify.Equal_Values(t, sql.STATUS_OK, Status, "pool init")
  return Pool
}

func test_request(t *testing.T) (request driver.Request) {
  t.Helper()
  query, validation := query_validate("select value")
  testify.Equal_Values(t, driver.STATUS_OK, validation, "query")
  request, validation = request_of(query, nil)
  testify.Equal_Values(t, driver.STATUS_OK, validation, "request")
  return request
}

func fake_driver(State *fake_state) (injected driver.Driver) {
  return driver.Driver{State: unsafe.Pointer(State), Connect_Procedure: fake_connect}
}

func fake_connect(
  State unsafe.Pointer,
  data_source driver.Data_Source,
  destination driver.Connection_Pointer,
) (Status driver.Status) {
  fake := (*fake_state)(State)
  fake.Connect_Count++
  destination.State = State
  destination.Close_Procedure = fake_close
  destination.Probe_Procedure = fake_probe
  destination.Exec_Procedure = fake_exec
  destination.Query_Procedure = fake_query
  destination.Prepare_Procedure = fake_prepare
  destination.Begin_Procedure = fake_begin
  return fake.Return_Status
}

func fake_close(State unsafe.Pointer) (Status driver.Status) {
  fake := (*fake_state)(State)
  fake.Close_Count++
  return fake.Return_Status
}

func fake_probe(State unsafe.Pointer) (Status driver.Status) {
  fake := (*fake_state)(State)
  if fake.Mutate_Generation {
    fake.Mutate_Generation = false
    fake.Slots[NUMBER_ZERO].Generation++
  }
  if fake.Mutate_State {
    fake.Mutate_State = false
    fake.Slots[NUMBER_ZERO].State = sql.SLOT_EMPTY
  }
  return fake.Return_Status
}

func fake_exec(
  State unsafe.Pointer,
  request driver.Request,
  Result driver.Result_Pointer,
) (Status driver.Status) {
  driver.Result_Set_Rows_Affected(Result, ROW_COLUMN_COUNT)
  return (*fake_state)(State).Return_Status
}

func fake_query(
  State unsafe.Pointer,
  request driver.Request,
  Rows driver.Rows_Pointer,
) (Status driver.Status) {
  (*fake_state)(State).Row_Position = NUMBER_ZERO
  fake_rows(State, Rows)
  return (*fake_state)(State).Return_Status
}

func fake_rows(State unsafe.Pointer, Rows driver.Rows_Pointer) {
  Rows.State = State
  Rows.Column_Count = ROW_COLUMN_COUNT
  Rows.Next_Procedure = fake_rows_next
  Rows.Close_Procedure = fake_rows_close
}

func fake_rows_next(State unsafe.Pointer, destination driver.Values) (Status driver.Status) {
  fake := (*fake_state)(State)
  if fake.Return_Status != driver.STATUS_OK {
    return fake.Return_Status
  }
  if fake.Row_Position != NUMBER_ZERO {
    return driver.STATUS_DONE
  }
  driver.Value_Of_Integer(&destination[NUMBER_ZERO], NUMBER_TWO)
  fake.Row_Position++
  return driver.STATUS_OK
}

func fake_rows_close(State unsafe.Pointer) (Status driver.Status) {
  return (*fake_state)(State).Return_Status
}

func fake_prepare(
  State unsafe.Pointer,
  query driver.Query,
  Statement driver.Statement_Pointer,
) (Status driver.Status) {
  Statement.State = State
  Statement.Argument_Count = NUMBER_ZERO
  Statement.Exec_Procedure = fake_statement_exec
  Statement.Query_Procedure = fake_statement_query
  Statement.Close_Procedure = fake_statement_close
  return (*fake_state)(State).Return_Status
}

func fake_statement_exec(
  State unsafe.Pointer,
  arguments driver.Arguments,
  Result driver.Result_Pointer,
) (Status driver.Status) {
  driver.Result_Set_Rows_Affected(Result, ROW_COLUMN_COUNT)
  return (*fake_state)(State).Return_Status
}

func fake_statement_query(
  State unsafe.Pointer,
  arguments driver.Arguments,
  Rows driver.Rows_Pointer,
) (Status driver.Status) {
  (*fake_state)(State).Row_Position = NUMBER_ZERO
  fake_rows(State, Rows)
  return (*fake_state)(State).Return_Status
}

func fake_statement_close(State unsafe.Pointer) (Status driver.Status) {
  return (*fake_state)(State).Return_Status
}

func fake_begin(
  State unsafe.Pointer,
  options driver.Transaction_Options,
  Transaction driver.Transaction_Pointer,
) (Status driver.Status) {
  Transaction.State = State
  Transaction.Commit_Procedure = fake_commit
  Transaction.Rollback_Procedure = fake_rollback
  return (*fake_state)(State).Return_Status
}

func fake_commit(State unsafe.Pointer) (Status driver.Status) {
  return (*fake_state)(State).Return_Status
}

func fake_rollback(State unsafe.Pointer) (Status driver.Status) {
  return (*fake_state)(State).Return_Status
}
