package sql_test

import (
	"testing"
	"unsafe"

	"local/james-orcales/shared/database/driver"
	"local/james-orcales/shared/database/sql"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/slices"
	"local/james-orcales/shared/strings"
	"local/james-orcales/shared/testify"
)

// Test_Initialization protects fixed caller capacity and injected driver.
func Test_Initialization(t *testing.T) {
	state := fake_state{}
	data_source, validation := driver.Data_Source_Validate("memory")
	testify.Equal_Values(t, driver.STATUS_OK, validation, "data source")
	var pool sql.Pool
	var slots [POOL_CAPACITY]sql.Slot
	status := sql.Pool_Init(
		&pool, fake_driver(&state), data_source, slots[:],
	)
	testify.Equal_Values(t, sql.STATUS_OK, status, "pool init")
	statistics := sql.Pool_Statistics(&pool)
	testify.Equal(t, sql.Connection_Count(POOL_CAPACITY),
		sql.Statistics_Capacity(statistics), "capacity")

	var empty sql.Pool
	status = sql.Pool_Init(
		&empty, fake_driver(&state), data_source, nil,
	)
	testify.Equal_Values(t, sql.STATUS_STORAGE_INVALID, status, "empty capacity")
}

// Test_Pool protects reuse, exhaustion, stale lease rejection, and bounded close.
func Test_Pool(t *testing.T) {
	state := fake_state{}
	pool := test_pool(t, &state)
	first, status := sql.Pool_Connection_Acquire(&pool)
	testify.Equal_Values(t, sql.STATUS_OK, status, "first")
	stale := first
	second, status := sql.Pool_Connection_Acquire(&pool)
	testify.Equal_Values(t, sql.STATUS_OK, status, "second")
	_, status = sql.Pool_Connection_Acquire(&pool)
	testify.Equal_Values(t, sql.STATUS_EXHAUSTED, status, "bounded exhaustion")
	testify.Equal_Values(t, sql.STATUS_BUSY, sql.Pool_Close(&pool), "busy close")
	testify.Equal_Values(t, sql.STATUS_OK, sql.Connection_Close(&first), "release first")
	reused, status := sql.Pool_Connection_Acquire(&pool)
	testify.Equal_Values(t, sql.STATUS_OK, status, "reuse")
	testify.Equal_Values(t, sql.STATUS_HANDLE_INVALID, sql.Connection_Probe(&stale), "stale")
	testify.Equal_Values(t, sql.STATUS_OK, sql.Connection_Close(&reused), "release reused")
	testify.Equal_Values(t, sql.STATUS_OK, sql.Connection_Close(&second), "release second")
	testify.Equal_Values(t, sql.STATUS_OK, sql.Pool_Close(&pool), "close")
	testify.Equal(t, POOL_CAPACITY, state.Close_Count, "driver close count")
}

// Test_Connections protects direct ping, execute, query, rows ownership, and release.
func Test_Connections(t *testing.T) {
	state := fake_state{}
	pool := test_pool(t, &state)
	connection, status := sql.Pool_Connection_Acquire(&pool)
	testify.Equal_Values(t, sql.STATUS_OK, status, "acquire")
	testify.Equal_Values(t, sql.STATUS_OK, sql.Connection_Probe(&connection), "probe")
	request := test_request(t)
	var result sql.Result
	status = sql.Status(sql.Connection_Exec(&connection, request, &result))
	testify.Equal_Values(t, sql.STATUS_OK, status, "execute")
	_, result_status := sql.Result_Rows_Affected(result)
	testify.Equal_Values(t, driver.STATUS_OK, result_status, "affected")
	var rows sql.Rows
	status = sql.Status(sql.Connection_Query(&connection, request, &rows))
	testify.Equal_Values(t, sql.STATUS_OK, status, "query")
	testify.Equal_Values(t, sql.STATUS_BUSY, sql.Connection_Close(&connection), "active rows")
	var values [ROW_COLUMN_COUNT]driver.Value
	testify.Equal_Values(t, sql.STATUS_OK, sql.Rows_Next(&rows, values[:]), "row")
	testify.Equal_Values(t, sql.STATUS_DONE, sql.Rows_Next(&rows, values[:]), "done")
	testify.Equal_Values(t, sql.STATUS_OK, sql.Connection_Close(&connection), "release")
	testify.Equal_Values(t, sql.STATUS_OK, sql.Pool_Close(&pool), "pool close")
}

// Test_Rows protects exact destination size and explicit idempotent close rejection.
func Test_Rows(t *testing.T) {
	state := fake_state{}
	pool := test_pool(t, &state)
	var rows sql.Rows
	status := sql.Pool_Query(&pool, test_request(t), &rows)
	testify.Equal_Values(t, sql.STATUS_OK, status, "query")
	testify.Equal_Values(t, sql.STATUS_STORAGE_INVALID, sql.Rows_Next(&rows, nil), "width")
	testify.Equal_Values(t, sql.STATUS_OK, sql.Rows_Close(&rows), "close")
	testify.Equal_Values(t, sql.STATUS_HANDLE_INVALID, sql.Rows_Close(&rows), "closed")
	testify.Equal_Values(t, sql.STATUS_OK, sql.Pool_Close(&pool), "pool close")
}

// Test_Statements protects pinned prepare, execution, rows, and release.
func Test_Statements(t *testing.T) {
	state := fake_state{}
	pool := test_pool(t, &state)
	query, _ := driver.Query_Validate("select value")
	var statement sql.Statement
	status := sql.Pool_Prepare(&pool, query, &statement)
	testify.Equal_Values(t, sql.STATUS_OK, status, "prepare")
	var result sql.Result
	status = sql.Status(sql.Statement_Exec(&statement, nil, &result))
	testify.Equal_Values(t, sql.STATUS_OK, status, "execute")
	var rows sql.Rows
	status = sql.Status(sql.Statement_Query(&statement, nil, &rows))
	testify.Equal_Values(t, sql.STATUS_OK, status, "query")
	testify.Equal_Values(t, sql.STATUS_BUSY, sql.Statement_Close(&statement), "active rows")
	testify.Equal_Values(t, sql.STATUS_OK, sql.Rows_Close(&rows), "rows close")
	testify.Equal_Values(t, sql.STATUS_OK, sql.Statement_Close(&statement), "close")
	testify.Equal_Values(t, sql.STATUS_OK, sql.Pool_Close(&pool), "pool close")
}

// Test_Transactions protects execution, rows, commit, rollback, and terminal state.
func Test_Transactions(t *testing.T) {
	state := fake_state{}
	pool := test_pool(t, &state)
	options := driver.Transaction_Options_Of(driver.ISOLATION_SERIALIZABLE, false)
	var transaction sql.Transaction
	status := sql.Pool_Begin(&pool, options, &transaction)
	testify.Equal_Values(t, sql.STATUS_OK, status, "begin commit")
	var result sql.Result
	status = sql.Status(sql.Transaction_Exec(&transaction, test_request(t), &result))
	testify.Equal_Values(t, sql.STATUS_OK, status, "execute")
	var rows sql.Rows
	status = sql.Status(sql.Transaction_Query(&transaction, test_request(t), &rows))
	testify.Equal_Values(t, sql.STATUS_OK, status, "query")
	testify.Equal_Values(
		t, sql.STATUS_BUSY, sql.Transaction_Commit(&transaction), "active rows",
	)
	testify.Equal_Values(t, sql.STATUS_OK, sql.Rows_Close(&rows), "rows close")
	testify.Equal_Values(t, sql.STATUS_OK, sql.Transaction_Commit(&transaction), "commit")
	testify.Equal_Values(t, sql.STATUS_HANDLE_INVALID,
		sql.Transaction_Rollback(&transaction), "terminal")

	status = sql.Pool_Begin(&pool, options, &transaction)
	testify.Equal_Values(t, sql.STATUS_OK, status, "begin rollback")
	testify.Equal_Values(t, sql.STATUS_OK, sql.Transaction_Rollback(&transaction), "rollback")
	testify.Equal_Values(t, sql.STATUS_OK, sql.Pool_Close(&pool), "pool close")
}

// Test_Allocation protects every public operation without hidden ownership.
func Test_Allocation(t *testing.T) {
	data_source, validation := driver.Data_Source_Validate("memory")
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

func test_transaction_option_domains(t *testing.T) {
	state := fake_state{}
	pool := test_pool(t, &state)
	for _, isolation := range []driver.Isolation_Level{
		driver.ISOLATION_DEFAULT, driver.ISOLATION_READ_UNCOMMITTED,
		driver.ISOLATION_READ_COMMITTED, driver.ISOLATION_LINEARIZABLE,
	} {
		options := driver.Transaction_Options_Of(isolation, true)
		var transaction sql.Transaction
		status := sql.Pool_Begin(&pool, options, &transaction)
		testify.Equal_Values(t, sql.STATUS_OK, status, "pool begin")
		testify.Equal_Values(
			t, sql.STATUS_OK, sql.Transaction_Rollback(&transaction), "pool rollback",
		)
		connection, status := sql.Pool_Connection_Acquire(&pool)
		testify.Equal_Values(t, sql.STATUS_OK, status, "acquire")
		connection_status := sql.Connection_Begin(&connection, options, &transaction)
		testify.Equal_Values(t, sql.STATUS_OK, connection_status, "connection begin")
		connection_status = sql.Transaction_Rollback(&transaction)
		testify.Equal_Values(
			t, sql.STATUS_OK, connection_status, "connection rollback",
		)
		testify.Equal_Values(t, sql.STATUS_OK, sql.Connection_Close(&connection), "release")
	}
	testify.Equal_Values(t, sql.STATUS_OK, sql.Pool_Close(&pool), "close")
}

func test_allocation_initialization(t *testing.T, data_source driver.Data_Source) {
	var status sql.Initialization_Status
	state := fake_state{}
	var pool sql.Pool
	var slots [POOL_CAPACITY]sql.Slot
	testify.Zero_Allocation(t, func() {
		state = fake_state{}
		pool = sql.Pool{}
		slots = [POOL_CAPACITY]sql.Slot{}
		status = sql.Pool_Init(
			&pool, fake_driver(&state), data_source, slots[:],
		)
	})
	testify.Equal_Values(t, sql.STATUS_OK, status, "initialization")
}

func test_allocation_pool(
	t *testing.T, data_source driver.Data_Source, request driver.Request,
	options driver.Transaction_Options,
) {
	var status sql.Status
	state := fake_state{}
	var pool sql.Pool
	var slots [POOL_CAPACITY]sql.Slot
	var result sql.Result
	var rows sql.Rows
	var values [ROW_COLUMN_COUNT]driver.Value
	var statement sql.Statement
	var transaction sql.Transaction
	query := request.Query
	testify.Zero_Allocation(t, func() {
		state = fake_state{}
		pool = sql.Pool{}
		slots = [POOL_CAPACITY]sql.Slot{}
		result = sql.Result{}
		rows = sql.Rows{}
		statement = sql.Statement{}
		transaction = sql.Transaction{}
		sql.Pool_Init(
			&pool, fake_driver(&state), data_source, slots[:],
		)
		status = sql.Pool_Probe(&pool)
		status = sql.Pool_Exec(&pool, request, &result)
		sql.Result_Rows_Affected(result)
		sql.Result_Last_Insert_Identifier(result)
		statistics := sql.Pool_Statistics(&pool)
		sql.Statistics_Capacity(statistics)
		sql.Statistics_Open(statistics)
		sql.Statistics_Idle(statistics)
		sql.Statistics_In_Use(statistics)

		status = sql.Pool_Query(&pool, request, &rows)
		status = sql.Status(sql.Rows_Next(&rows, values[:]))
		status = sql.Status(sql.Rows_Close(&rows))

		status = sql.Pool_Prepare(&pool, query, &statement)
		status = sql.Status(sql.Statement_Exec(&statement, nil, &result))
		status = sql.Status(sql.Statement_Query(&statement, nil, &rows))
		status = sql.Status(sql.Rows_Close(&rows))
		status = sql.Status(sql.Statement_Close(&statement))

		status = sql.Pool_Begin(&pool, options, &transaction)
		status = sql.Status(sql.Transaction_Exec(&transaction, request, &result))
		status = sql.Status(sql.Transaction_Query(&transaction, request, &rows))
		status = sql.Status(sql.Rows_Close(&rows))
		status = sql.Status(sql.Transaction_Commit(&transaction))
		status = sql.Pool_Begin(&pool, options, &transaction)
		status = sql.Status(sql.Transaction_Rollback(&transaction))
		status = sql.Status(sql.Pool_Close(&pool))
	})
	testify.Equal_Values(t, sql.STATUS_OK, status, "pool public operations")
}

func test_allocation_connection(
	t *testing.T, data_source driver.Data_Source, request driver.Request,
	options driver.Transaction_Options,
) {
	var status sql.Status
	state := fake_state{}
	var pool sql.Pool
	var slots [POOL_CAPACITY]sql.Slot
	var connection sql.Connection
	var result sql.Result
	var rows sql.Rows
	var values [ROW_COLUMN_COUNT]driver.Value
	var statement sql.Statement
	var transaction sql.Transaction
	query := request.Query
	testify.Zero_Allocation(t, func() {
		state = fake_state{}
		pool = sql.Pool{}
		slots = [POOL_CAPACITY]sql.Slot{}
		connection = sql.Connection{}
		result = sql.Result{}
		rows = sql.Rows{}
		statement = sql.Statement{}
		transaction = sql.Transaction{}
		sql.Pool_Init(
			&pool, fake_driver(&state), data_source, slots[:],
		)
		var acquired sql.Status
		connection, acquired = sql.Pool_Connection_Acquire(&pool)
		status = acquired
		status = sql.Status(sql.Connection_Probe(&connection))
		status = sql.Status(sql.Connection_Exec(&connection, request, &result))
		status = sql.Status(sql.Connection_Query(&connection, request, &rows))
		status = sql.Status(sql.Rows_Next(&rows, values[:]))
		status = sql.Status(sql.Rows_Close(&rows))

		status = sql.Status(sql.Connection_Prepare(&connection, query, &statement))
		status = sql.Status(sql.Statement_Close(&statement))
		status = sql.Status(sql.Connection_Begin(&connection, options, &transaction))
		status = sql.Status(sql.Transaction_Rollback(&transaction))
		status = sql.Status(sql.Connection_Close(&connection))
		status = sql.Status(sql.Pool_Close(&pool))
	})
	testify.Equal_Values(t, sql.STATUS_OK, status, "connection public operations")
}

func test_allocation_rejection(
	t *testing.T, data_source driver.Data_Source, request driver.Request,
	options driver.Transaction_Options,
) {
	var status sql.Status
	state := fake_state{}
	var pool sql.Pool
	var slots [POOL_CAPACITY]sql.Slot
	var first sql.Connection
	var second sql.Connection
	var zero_connection sql.Connection
	var result sql.Result
	var rows sql.Rows
	var statement sql.Statement
	var transaction sql.Transaction
	query := request.Query
	testify.Zero_Allocation(t, func() {
		state = fake_state{}
		pool = sql.Pool{}
		slots = [POOL_CAPACITY]sql.Slot{}
		first = sql.Connection{}
		second = sql.Connection{}
		zero_connection = sql.Connection{}
		result = sql.Result{}
		rows = sql.Rows{}
		statement = sql.Statement{}
		transaction = sql.Transaction{}
		sql.Pool_Init(
			&pool, fake_driver(&state), data_source, slots[:],
		)
		var acquired sql.Status
		first, acquired = sql.Pool_Connection_Acquire(&pool)
		status = acquired
		second, acquired = sql.Pool_Connection_Acquire(&pool)
		status = acquired
		_, status = sql.Pool_Connection_Acquire(&pool)
		status = sql.Status(sql.Connection_Probe(&zero_connection))
		status = sql.Status(sql.Connection_Exec(&zero_connection, request, &result))
		status = sql.Status(sql.Connection_Query(&zero_connection, request, &rows))
		status = sql.Status(sql.Connection_Prepare(&zero_connection, query, &statement))
		status = sql.Status(sql.Connection_Begin(&zero_connection, options, &transaction))
		status = sql.Status(sql.Rows_Next(&rows, nil))
		status = sql.Status(sql.Rows_Close(&rows))
		status = sql.Status(sql.Statement_Exec(&statement, nil, &result))
		status = sql.Status(sql.Statement_Query(&statement, nil, &rows))
		status = sql.Status(sql.Statement_Close(&statement))
		status = sql.Status(sql.Transaction_Exec(&transaction, request, &result))
		status = sql.Status(sql.Transaction_Query(&transaction, request, &rows))
		status = sql.Status(sql.Transaction_Commit(&transaction))
		status = sql.Status(sql.Transaction_Rollback(&transaction))
		status = sql.Status(sql.Connection_Close(&first))
		status = sql.Status(sql.Connection_Close(&second))
		status = sql.Status(sql.Pool_Close(&pool))
	})
	testify.Equal_Values(t, sql.STATUS_OK, status, "rejected public operations")
}

func test_pool_domains(t *testing.T) {
	request := test_request(t)
	query := request.Query
	options := driver.Transaction_Options_Of(driver.ISOLATION_SERIALIZABLE, false)
	for _, status := range []driver.Status{
		driver.STATUS_DONE, driver.STATUS_INPUT_INVALID, driver.STATUS_UNSUPPORTED,
	} {
		state := fake_state{Return_Status: status}
		pool := test_pool(t, &state)
		var result sql.Result
		var rows sql.Rows
		var statement sql.Statement
		var transaction sql.Transaction
		sql.Pool_Connection_Acquire(&pool)
		sql.Pool_Probe(&pool)
		sql.Pool_Exec(&pool, request, &result)
		sql.Pool_Query(&pool, request, &rows)
		sql.Pool_Prepare(&pool, query, &statement)
		sql.Pool_Begin(&pool, options, &transaction)
	}

	state := fake_state{}
	pool := test_pool(t, &state)
	for index := range pool.Slots {
		pool.Slots[index].Generation = sql.Generation(bits.WORD_64_MAXIMUM)
	}
	var result sql.Result
	var rows sql.Rows
	var statement sql.Statement
	var transaction sql.Transaction
	sql.Pool_Connection_Acquire(&pool)
	sql.Pool_Probe(&pool)
	sql.Pool_Exec(&pool, request, &result)
	sql.Pool_Query(&pool, request, &rows)
	sql.Pool_Prepare(&pool, query, &statement)
	sql.Pool_Begin(&pool, options, &transaction)
	test_pool_storage_domains(t)
	test_query_domains(t)
}

func test_pool_storage_domains(t *testing.T) {
	state := fake_state{}
	for _, count := range []int{
		NUMBER_ONE, NUMBER_TWO, sql.CONNECTION_COUNT_MAXIMUM,
	} {
		storage := make([]sql.Slot, count)
		var pool sql.Pool
		data_source := driver.Data_Source(
			text_of(count % (strings.TEXT_SIZE_MAXIMUM + NUMBER_ONE)),
		)
		sql.Pool_Init(
			&pool, fake_driver(&state), data_source, storage,
		)
		statistics := sql.Pool_Statistics(&pool)
		sql.Statistics_Capacity(statistics)
	}
	for _, size := range []int{
		NUMBER_ZERO, NUMBER_ONE, NUMBER_TWO, strings.TEXT_SIZE_MAXIMUM,
	} {
		var pool sql.Pool
		var slots [POOL_CAPACITY]sql.Slot
		data_source := driver.Data_Source(text_of(size))
		sql.Pool_Init(
			&pool, fake_driver(&state), data_source, slots[:],
		)
		sql.Pool_Statistics(&pool)
	}
}

func test_query_domains(t *testing.T) {
	for _, size := range []int{
		NUMBER_ZERO, NUMBER_ONE, NUMBER_TWO, strings.TEXT_SIZE_MAXIMUM,
	} {
		state := fake_state{}
		pool := test_pool(t, &state)
		query := driver.Query(text_of(size))
		var statement sql.Statement
		sql.Pool_Prepare(&pool, query, &statement)
		sql.Statement_Close(&statement)
		connection, _ := sql.Pool_Connection_Acquire(&pool)
		sql.Connection_Prepare(&connection, query, &statement)
		sql.Statement_Close(&statement)
		sql.Connection_Close(&connection)
		sql.Pool_Close(&pool)
	}
}

func test_connection_domains(t *testing.T) {
	request := test_request(t)
	query := request.Query
	options := driver.Transaction_Options_Of(driver.ISOLATION_SERIALIZABLE, false)
	for _, driver_status := range []driver.Status{
		driver.STATUS_DONE, driver.STATUS_INPUT_INVALID, driver.STATUS_UNSUPPORTED,
	} {
		state := fake_state{}
		pool := test_pool(t, &state)
		connection, _ := sql.Pool_Connection_Acquire(&pool)
		state.Return_Status = driver_status
		var result sql.Result
		var rows sql.Rows
		var statement sql.Statement
		var transaction sql.Transaction
		sql.Connection_Probe(&connection)
		sql.Connection_Exec(&connection, request, &result)
		sql.Connection_Query(&connection, request, &rows)
		sql.Connection_Prepare(&connection, query, &statement)
		sql.Connection_Begin(&connection, options, &transaction)
		state.Return_Status = driver.STATUS_OK
		test_rows_status(t, &pool, &connection, &state, request, driver_status)
		test_statement_status(&connection, &state, query, driver_status)
		test_transaction_status(&connection, &state, options, driver_status)
		sql.Connection_Close(&connection)
		state.Return_Status = driver.STATUS_OK
		sql.Pool_Close(&pool)
		test_pool_close_status(t, driver_status)
	}
}

func test_rows_status(
	t *testing.T, pool *sql.Pool, connection *sql.Connection, state *fake_state,
	request driver.Request, status driver.Status,
) {
	var rows sql.Rows
	sql.Connection_Query(connection, request, &rows)
	state.Return_Status = status
	var values [ROW_COLUMN_COUNT]driver.Value
	sql.Rows_Next(&rows, values[:])
	state.Return_Status = driver.STATUS_OK
	sql.Connection_Query(connection, request, &rows)
	state.Return_Status = status
	sql.Rows_Close(&rows)
	state.Return_Status = driver.STATUS_OK
	sql.Pool_Statistics(pool)
}

func test_statement_status(
	connection *sql.Connection, state *fake_state, query driver.Query,
	status driver.Status,
) {
	var statement sql.Statement
	var result sql.Result
	var rows sql.Rows
	sql.Connection_Prepare(connection, query, &statement)
	state.Return_Status = status
	sql.Statement_Exec(&statement, nil, &result)
	sql.Statement_Query(&statement, nil, &rows)
	sql.Statement_Close(&statement)
	state.Return_Status = driver.STATUS_OK
}

func test_transaction_status(
	connection *sql.Connection, state *fake_state, options driver.Transaction_Options,
	status driver.Status,
) {
	request := driver.Request{Query: ""}
	var transaction sql.Transaction
	var result sql.Result
	var rows sql.Rows
	sql.Connection_Begin(connection, options, &transaction)
	state.Return_Status = status
	sql.Transaction_Exec(&transaction, request, &result)
	sql.Transaction_Query(&transaction, request, &rows)
	sql.Transaction_Commit(&transaction)
	state.Return_Status = driver.STATUS_OK
	sql.Connection_Begin(connection, options, &transaction)
	state.Return_Status = status
	sql.Transaction_Rollback(&transaction)
	state.Return_Status = driver.STATUS_OK
}

func test_pool_close_status(t *testing.T, status driver.Status) {
	state := fake_state{}
	pool := test_pool(t, &state)
	connection, _ := sql.Pool_Connection_Acquire(&pool)
	sql.Connection_Close(&connection)
	state.Return_Status = status
	sql.Pool_Close(&pool)
}

func test_busy_domains(t *testing.T) {
	request := test_request(t)
	query := request.Query
	options := driver.Transaction_Options_Of(driver.ISOLATION_SERIALIZABLE, false)
	state := fake_state{}
	pool := test_pool(t, &state)
	connection, _ := sql.Pool_Connection_Acquire(&pool)
	var rows sql.Rows
	sql.Connection_Query(&connection, request, &rows)
	var result sql.Result
	var statement sql.Statement
	var transaction sql.Transaction
	sql.Connection_Probe(&connection)
	sql.Connection_Exec(&connection, request, &result)
	sql.Connection_Query(&connection, request, &rows)
	sql.Connection_Prepare(&connection, query, &statement)
	sql.Connection_Begin(&connection, options, &transaction)
	sql.Connection_Close(&connection)
	sql.Rows_Close(&rows)

	sql.Connection_Prepare(&connection, query, &statement)
	sql.Statement_Query(&statement, nil, &rows)
	sql.Statement_Exec(&statement, nil, &result)
	sql.Statement_Query(&statement, nil, &rows)
	sql.Statement_Close(&statement)
	sql.Rows_Close(&rows)
	sql.Statement_Close(&statement)

	sql.Connection_Begin(&connection, options, &transaction)
	sql.Transaction_Query(&transaction, request, &rows)
	sql.Transaction_Exec(&transaction, request, &result)
	sql.Transaction_Query(&transaction, request, &rows)
	sql.Transaction_Commit(&transaction)
	sql.Transaction_Rollback(&transaction)
	sql.Rows_Close(&rows)
	sql.Transaction_Rollback(&transaction)

	sql.Connection_Query(&connection, request, &rows)
	slot := &pool.Slots[NUMBER_ZERO]
	slot.State = sql.SLOT_TRANSACTION
	var values [ROW_COLUMN_COUNT]driver.Value
	sql.Rows_Next(&rows, values[:])
	sql.Rows_Close(&rows)
	slot.State = sql.SLOT_ROWS
	sql.Rows_Close(&rows)
	sql.Connection_Close(&connection)
	sql.Pool_Close(&pool)
}

func test_bad_connection_domains(t *testing.T) {
	test_bad_connection_state(t, false, false)
	test_bad_connection_state(t, true, false)
	test_bad_connection_state(t, false, true)
	request := test_request(t)
	query := request.Query
	options := driver.Transaction_Options_Of(driver.ISOLATION_SERIALIZABLE, false)

	state := fake_state{}
	pool := test_pool(t, &state)
	connection, _ := sql.Pool_Connection_Acquire(&pool)
	var rows sql.Rows
	sql.Connection_Query(&connection, request, &rows)
	state.Return_Status = driver.STATUS_BAD_CONNECTION
	var values [ROW_COLUMN_COUNT]driver.Value
	sql.Rows_Next(&rows, values[:])

	state = fake_state{}
	pool = test_pool(t, &state)
	connection, _ = sql.Pool_Connection_Acquire(&pool)
	var closing_rows sql.Rows
	sql.Connection_Query(&connection, request, &closing_rows)
	state.Return_Status = driver.STATUS_BAD_CONNECTION
	sql.Rows_Close(&closing_rows)

	state = fake_state{}
	pool = test_pool(t, &state)
	connection, _ = sql.Pool_Connection_Acquire(&pool)
	var statement sql.Statement
	sql.Connection_Prepare(&connection, query, &statement)
	state.Return_Status = driver.STATUS_BAD_CONNECTION
	var result sql.Result
	sql.Statement_Exec(&statement, nil, &result)

	state = fake_state{}
	pool = test_pool(t, &state)
	connection, _ = sql.Pool_Connection_Acquire(&pool)
	var transaction sql.Transaction
	sql.Connection_Begin(&connection, options, &transaction)
	state.Return_Status = driver.STATUS_BAD_CONNECTION
	sql.Transaction_Exec(&transaction, request, &result)
}

func test_bad_connection_state(
	t *testing.T, mutate_generation bool, mutate_state bool,
) {
	state := fake_state{}
	pool := test_pool(t, &state)
	connection, _ := sql.Pool_Connection_Acquire(&pool)
	state.Slots = pool.Slots
	state.Mutate_Generation = mutate_generation
	state.Mutate_State = mutate_state
	state.Return_Status = driver.STATUS_BAD_CONNECTION
	sql.Connection_Probe(&connection)
}

func test_handle_domains(t *testing.T) {
	state := fake_state{}
	var zero sql.Connection
	sql.Connection_Close(&zero)
	pool := test_pool(t, &state)
	closed, _ := sql.Pool_Connection_Acquire(&pool)
	pool.Closed = sql.POOL_CLOSED
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
		state := fake_state{}
		pool := test_pool(t, &state)
		connection, _ := sql.Pool_Connection_Acquire(&pool)
		state.Slots = pool.Slots
		state.Mutate_Generation = mutation.Generation
		state.Mutate_State = mutation.State
		sql.Connection_Probe(&connection)
	}
}

func test_handle_boundary_domains(t *testing.T) {
	state := fake_state{}
	storage := make([]sql.Slot, sql.CONNECTION_COUNT_MAXIMUM)
	var pool sql.Pool
	data_source := driver.Data_Source("memory")
	sql.Pool_Init(
		&pool, fake_driver(&state), data_source, storage,
	)
	for _, index := range []sql.Slot_Index{
		NUMBER_ZERO, NUMBER_ONE, NUMBER_TWO, sql.SLOT_INDEX_MAXIMUM,
	} {
		for _, generation := range []sql.Lease_Generation{
			NUMBER_ONE, NUMBER_TWO, sql.Lease_Generation(bits.WORD_64_MAXIMUM),
		} {
			slot := &storage[index]
			fake_connect(
				unsafe.Pointer(&state), data_source,
				&slot.Connection,
			)
			slot.State = sql.SLOT_CONNECTION
			slot.Generation = sql.Generation(generation)
			connection := sql.Connection{
				Pool:       unsafe.Pointer(&pool),
				Slot_Index: index,
				Generation: sql.Generation(generation),
			}
			state.Return_Status = driver.STATUS_OK
			sql.Connection_Probe(&connection)
			state.Return_Status = driver.STATUS_BAD_CONNECTION
			sql.Connection_Probe(&connection)
			state.Return_Status = driver.STATUS_OK
		}
	}
}

func test_connection_of_domains(t *testing.T) {
	state := fake_state{}
	storage := make([]sql.Slot, sql.CONNECTION_COUNT_MAXIMUM)
	for index := NUMBER_ZERO; index < len(storage)-NUMBER_ONE; index++ {
		storage[index].State = sql.SLOT_CONNECTION
	}
	var pool sql.Pool
	sql.Pool_Init(
		&pool, fake_driver(&state), driver.Data_Source("memory"),
		storage,
	)
	for index := NUMBER_ZERO; index < len(storage)-NUMBER_ONE; index++ {
		storage[index].State = sql.SLOT_CONNECTION
	}
	sql.Pool_Connection_Acquire(&pool)

	pool = sql.Pool{}
	sql.Pool_Init(
		&pool, fake_driver(&state), driver.Data_Source("memory"),
		storage,
	)
	for index := range storage {
		if index != NUMBER_TWO {
			storage[index].State = sql.SLOT_CONNECTION
		}
	}
	sql.Pool_Connection_Acquire(&pool)

	var single [sql.INITIALIZED_CONNECTION_COUNT_MINIMUM]sql.Slot
	pool = sql.Pool{}
	sql.Pool_Init(
		&pool, fake_driver(&state), driver.Data_Source("memory"),
		single[:],
	)
	fake_connect(
		unsafe.Pointer(&state), driver.Data_Source("memory"),
		&single[NUMBER_ZERO].Connection,
	)
	single[NUMBER_ZERO].State = sql.SLOT_IDLE
	single[NUMBER_ZERO].Generation = sql.Generation(
		bits.WORD_64_MAXIMUM - NUMBER_ONE,
	)
	sql.Pool_Connection_Acquire(&pool)
}

func test_collection_domains(t *testing.T) {
	state := fake_state{}
	pool := test_pool(t, &state)
	connection, _ := sql.Pool_Connection_Acquire(&pool)
	request := test_request(t)
	var rows sql.Rows
	sql.Connection_Query(&connection, request, &rows)
	for _, count := range []int{NUMBER_TWO, sql.CONNECTION_COUNT_MAXIMUM} {
		sql.Rows_Next(&rows, make(driver.Values, count))
	}
	sql.Rows_Close(&rows)

	query := request.Query
	var statement sql.Statement
	sql.Connection_Prepare(&connection, query, &statement)
	var result sql.Result
	for _, count := range []int{
		NUMBER_ONE, NUMBER_TWO, driver.ARGUMENT_COUNT_MAXIMUM,
	} {
		arguments := make(driver.Arguments, count)
		sql.Statement_Exec(&statement, arguments, &result)
		sql.Statement_Query(&statement, arguments, &rows)
	}
	sql.Statement_Close(&statement)
	sql.Connection_Close(&connection)
	sql.Pool_Close(&pool)
}

func test_result_domains() {
	for _, value := range []int64{
		bits.INTEGER_64_MINIMUM, NUMBER_NEGATIVE_ONE, NUMBER_ZERO,
		NUMBER_ONE, NUMBER_TWO, bits.INTEGER_64_MAXIMUM,
	} {
		var result sql.Result
		driver.Result_Set_Last_Insert_Identifier(
			&result.Driver_Result,
			driver.Last_Insert_Identifier(value),
		)
		sql.Result_Last_Insert_Identifier(result)
		driver.Result_Set_Rows_Affected(
			&result.Driver_Result, driver.Rows_Affected(value),
		)
		sql.Result_Last_Insert_Identifier(result)
		sql.Result_Rows_Affected(result)
	}
	var result sql.Result
	sql.Result_Last_Insert_Identifier(result)
	sql.Result_Rows_Affected(result)
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
		sql.Generation(bits.WORD_64_MINIMUM), NUMBER_ONE, NUMBER_TWO,
		sql.Generation(bits.WORD_64_MAXIMUM),
	}
	for index := range indexes {
		connection := sql.Connection{
			Slot_Index: indexes[index], Generation: generations[index],
		}
		var result sql.Result
		var rows sql.Rows
		var statement sql.Statement
		var transaction sql.Transaction
		sql.Connection_Probe(&connection)
		sql.Connection_Exec(&connection, request, &result)
		sql.Connection_Query(&connection, request, &rows)
		sql.Connection_Prepare(&connection, "", &statement)
		sql.Connection_Begin(&connection, options, &transaction)
		sql.Connection_Close(&connection)
	}
}

func test_rows_pointer_storage_domains() {
	indexes := []sql.Slot_Index{
		NUMBER_ZERO, NUMBER_ONE, NUMBER_TWO, sql.SLOT_INDEX_MAXIMUM,
	}
	generations := []sql.Generation{
		sql.Generation(bits.WORD_64_MINIMUM), NUMBER_ONE, NUMBER_TWO,
		sql.Generation(bits.WORD_64_MAXIMUM),
	}
	column_counts := []driver.Column_Count{
		NUMBER_ZERO, NUMBER_ONE, NUMBER_TWO, slices.SLICE_COUNT_MAXIMUM,
	}
	return_states := []sql.Slot_State{
		sql.SLOT_EMPTY, sql.SLOT_CONNECTION_CREATION, sql.SLOT_IDLE, sql.SLOT_CLOSE,
	}
	for index := range indexes {
		connection := sql.Connection{
			Slot_Index: indexes[index], Generation: generations[index],
		}
		rows := sql.Rows{
			Connection: connection,
			Driver_Rows: driver.Rows{
				Column_Count: column_counts[index],
			},
			Return_State: return_states[index],
		}
		sql.Rows_Next(&rows, nil)
		rows.Connection = connection
		sql.Rows_Close(&rows)
	}
}

func test_statement_pointer_storage_domains() {
	indexes := []sql.Slot_Index{
		NUMBER_ZERO, NUMBER_ONE, NUMBER_TWO, sql.SLOT_INDEX_MAXIMUM,
	}
	generations := []sql.Generation{
		sql.Generation(bits.WORD_64_MINIMUM), NUMBER_ONE, NUMBER_TWO,
		sql.Generation(bits.WORD_64_MAXIMUM),
	}
	return_states := []sql.Slot_State{
		sql.SLOT_EMPTY, sql.SLOT_CONNECTION_CREATION, sql.SLOT_IDLE, sql.SLOT_CLOSE,
	}
	for index := range indexes {
		connection := sql.Connection{
			Slot_Index: indexes[index], Generation: generations[index],
		}
		statement_count := driver.Statement_Argument_Count(indexes[index])
		if index == NUMBER_ZERO {
			statement_count = driver.STATEMENT_ARGUMENT_COUNT_UNKNOWN
		}
		statement := sql.Statement{
			Connection: connection,
			Driver_Statement: driver.Statement{
				Argument_Count: statement_count,
			},
			Return_State: return_states[index],
		}
		var result sql.Result
		var rows sql.Rows
		sql.Statement_Exec(&statement, nil, &result)
		statement.Connection = connection
		sql.Statement_Query(&statement, nil, &rows)
		statement.Connection = connection
		sql.Statement_Close(&statement)
	}
	test_maximum_statement_pointer_storage_domain()
}

func test_transaction_pointer_storage_domains() {
	request := driver.Request{}
	indexes := []sql.Slot_Index{
		NUMBER_ZERO, NUMBER_ONE, NUMBER_TWO, sql.SLOT_INDEX_MAXIMUM,
	}
	generations := []sql.Generation{
		sql.Generation(bits.WORD_64_MINIMUM), NUMBER_ONE, NUMBER_TWO,
		sql.Generation(bits.WORD_64_MAXIMUM),
	}
	return_states := []sql.Slot_State{
		sql.SLOT_EMPTY, sql.SLOT_CONNECTION_CREATION, sql.SLOT_IDLE, sql.SLOT_CLOSE,
	}
	for index := range indexes {
		connection := sql.Connection{
			Slot_Index: indexes[index], Generation: generations[index],
		}
		driver_transaction := driver.Transaction{}
		if index != NUMBER_ZERO {
			driver_transaction.Commit_Procedure = fake_commit
		}
		transaction := sql.Transaction{
			Connection: connection, Driver_Transaction: driver_transaction,
			Return_State: return_states[index],
		}
		var result sql.Result
		var rows sql.Rows
		sql.Transaction_Exec(&transaction, request, &result)
		transaction.Connection = connection
		sql.Transaction_Query(&transaction, request, &rows)
		transaction.Connection = connection
		sql.Transaction_Commit(&transaction)
		transaction.Connection = connection
		sql.Transaction_Rollback(&transaction)
	}
}

func test_maximum_statement_pointer_storage_domain() {
	connection := sql.Connection{
		Slot_Index: sql.SLOT_INDEX_MAXIMUM,
		Generation: sql.Generation(bits.WORD_64_MAXIMUM),
	}
	statement := sql.Statement{
		Connection: connection,
		Driver_Statement: driver.Statement{
			Argument_Count: driver.ARGUMENT_COUNT_MAXIMUM,
		},
		Return_State: sql.SLOT_CLOSE,
	}
	var result sql.Result
	var rows sql.Rows
	sql.Statement_Exec(&statement, nil, &result)
	statement.Connection = connection
	sql.Statement_Query(&statement, nil, &rows)
	statement.Connection = connection
	sql.Statement_Close(&statement)
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
	state := fake_state{}
	pool := test_pool(t, &state)
	var result sql.Result
	var rows sql.Rows
	sql.Pool_Exec(&pool, request, &result)
	pool = test_pool(t, &state)
	sql.Pool_Query(&pool, request, &rows)

	var connection sql.Connection
	sql.Connection_Exec(&connection, request, &result)
	sql.Connection_Query(&connection, request, &rows)
	var transaction sql.Transaction
	sql.Transaction_Exec(&transaction, request, &result)
	sql.Transaction_Query(&transaction, request, &rows)
}

func test_pool_pointer_domains() {
	data_sizes := []int{
		NUMBER_ZERO, NUMBER_ONE, NUMBER_TWO,
		strings.TEXT_SIZE_MAXIMUM, strings.TEXT_SIZE_MAXIMUM,
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
		var pool sql.Pool
		sql.Pool_Connection_Acquire(&pool)
	}, "zero pool acquire")
	testify.Panics(t, func() {
		var pool sql.Pool
		sql.Pool_Probe(&pool)
	}, "zero pool probe")
	testify.Panics(t, func() {
		var pool sql.Pool
		var result sql.Result
		sql.Pool_Exec(&pool, driver.Request{}, &result)
	}, "zero pool execute")
	testify.Panics(t, func() {
		var pool sql.Pool
		var rows sql.Rows
		sql.Pool_Query(&pool, driver.Request{}, &rows)
	}, "zero pool query")
	testify.Panics(t, func() {
		var pool sql.Pool
		var statement sql.Statement
		sql.Pool_Prepare(&pool, "", &statement)
	}, "zero pool prepare")
	testify.Panics(t, func() {
		var pool sql.Pool
		var transaction sql.Transaction
		options := driver.Transaction_Options_Of(driver.ISOLATION_DEFAULT, false)
		sql.Pool_Begin(&pool, options, &transaction)
	}, "zero pool begin")
	testify.Panics(t, func() {
		var pool sql.Pool
		sql.Pool_Close(&pool)
	}, "zero pool close")
	testify.Panics(t, func() {
		var pool sql.Pool
		connection := sql.Connection{Pool: unsafe.Pointer(&pool)}
		sql.Connection_Probe(&connection)
	}, "zero pool handle")
}

func exercise_pool_pointer_domain(
	data_size int, slot_count int, closed bool,
) {
	request := driver.Request{}
	options := driver.Transaction_Options_Of(driver.ISOLATION_DEFAULT, false)

	pool := domain_pool(data_size, slot_count, closed)
	sql.Pool_Connection_Acquire(&pool)
	pool = domain_pool(data_size, slot_count, closed)
	sql.Pool_Probe(&pool)
	pool = domain_pool(data_size, slot_count, closed)
	var result sql.Result
	sql.Pool_Exec(&pool, request, &result)
	pool = domain_pool(data_size, slot_count, closed)
	var rows sql.Rows
	sql.Pool_Query(&pool, request, &rows)
	pool = domain_pool(data_size, slot_count, closed)
	var statement sql.Statement
	sql.Pool_Prepare(&pool, "", &statement)
	pool = domain_pool(data_size, slot_count, closed)
	var transaction sql.Transaction
	sql.Pool_Begin(&pool, options, &transaction)
	pool = domain_pool(data_size, slot_count, closed)
	sql.Pool_Close(&pool)
	pool = domain_pool(data_size, slot_count, closed)
	sql.Pool_Statistics(&pool)
}

func domain_pool(data_size int, slot_count int, closed bool) (pool sql.Pool) {
	state := fake_state{}
	storage := make([]sql.Slot, slot_count)
	sql.Pool_Init(
		&pool, fake_driver(&state), driver.Data_Source(text_of(data_size)),
		storage,
	)
	pool.Closed = sql.POOL_OPEN
	if closed {
		pool.Closed = sql.POOL_CLOSED
	}
	return pool
}

func test_pool_initialization_storage_domains() {
	state := fake_state{}
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
		pool := sql.Pool{
			Driver:      fake_driver(&state),
			Data_Source: driver.Data_Source(text_of(data_sizes[index])),
			Slots:       make(sql.Slot_Storage, slot_counts[index]),
			Closed:      closed,
		}
		var storage [POOL_CAPACITY]sql.Slot
		sql.Pool_Init(
			&pool, fake_driver(&state), "", storage[:],
		)
	}

	states := []sql.Slot_State{
		sql.SLOT_EMPTY, sql.SLOT_CONNECTION_CREATION, sql.SLOT_IDLE, sql.SLOT_CLOSE,
	}
	generations := []sql.Generation{
		sql.Generation(bits.WORD_64_MINIMUM), NUMBER_ONE, NUMBER_TWO,
		sql.Generation(bits.WORD_64_MAXIMUM),
	}
	storage := make([]sql.Slot, len(states))
	for index := range storage {
		storage[index].State = states[index]
		storage[index].Generation = generations[index]
		if index != NUMBER_ZERO {
			fake_connect(
				unsafe.Pointer(&state), "", &storage[index].Connection,
			)
		}
	}
	var pool sql.Pool
	sql.Pool_Init(
		&pool, fake_driver(&state), "", storage,
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
		bits.INTEGER_64_MINIMUM, NUMBER_NEGATIVE_ONE, NUMBER_ZERO,
		NUMBER_ONE, NUMBER_TWO, bits.INTEGER_64_MAXIMUM,
	} {
		stored := sql.Result{Driver_Result: driver.Result{
			Last_Insert_Identifier:          driver.Last_Insert_Identifier(value),
			Rows_Affected:                   driver.Rows_Affected(value),
			Last_Insert_Identifier_Validity: index != NUMBER_ZERO,
			Rows_Affected_Validity:          index != NUMBER_ZERO,
		}}
		var connection sql.Connection
		result := stored
		sql.Connection_Exec(&connection, request, &result)
		statement := sql.Statement{}
		result = stored
		sql.Statement_Exec(&statement, nil, &result)
		transaction := sql.Transaction{}
		result = stored
		sql.Transaction_Exec(&transaction, request, &result)
		state := fake_state{}
		pool := test_pool(t, &state)
		result = stored
		sql.Pool_Exec(&pool, request, &result)
	}
}

func test_rows_output_storage_domains(t *testing.T) {
	request := driver.Request{}
	indexes := []sql.Slot_Index{
		NUMBER_ZERO, NUMBER_ONE, NUMBER_TWO, sql.SLOT_INDEX_MAXIMUM,
	}
	generations := []sql.Generation{
		sql.Generation(bits.WORD_64_MINIMUM), NUMBER_ONE, NUMBER_TWO,
		sql.Generation(bits.WORD_64_MAXIMUM),
	}
	states := []sql.Slot_State{
		sql.SLOT_EMPTY, sql.SLOT_CONNECTION_CREATION, sql.SLOT_IDLE, sql.SLOT_CLOSE,
	}
	column_counts := []driver.Column_Count{
		NUMBER_ZERO, NUMBER_ONE, NUMBER_TWO, slices.SLICE_COUNT_MAXIMUM,
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
		var connection sql.Connection
		rows := stored
		sql.Connection_Query(&connection, request, &rows)
		state := fake_state{}
		pool := test_pool(t, &state)
		rows = stored
		sql.Pool_Query(&pool, request, &rows)

		pool = test_pool(t, &state)
		var statement sql.Statement
		sql.Pool_Prepare(&pool, "", &statement)
		rows = stored
		sql.Statement_Query(&statement, nil, &rows)

		pool = test_pool(t, &state)
		var transaction sql.Transaction
		options := driver.Transaction_Options_Of(driver.ISOLATION_DEFAULT, false)
		sql.Pool_Begin(&pool, options, &transaction)
		rows = stored
		sql.Transaction_Query(&transaction, request, &rows)
	}
}

func test_statement_output_storage_domains(t *testing.T) {
	counts := []driver.Statement_Argument_Count{
		driver.STATEMENT_ARGUMENT_COUNT_UNKNOWN, NUMBER_ZERO, NUMBER_ONE,
		NUMBER_TWO, driver.ARGUMENT_COUNT_MAXIMUM,
	}
	indexes := []sql.Slot_Index{
		NUMBER_ZERO, NUMBER_ONE, NUMBER_TWO, NUMBER_ZERO, sql.SLOT_INDEX_MAXIMUM,
	}
	generations := []sql.Generation{
		sql.Generation(bits.WORD_64_MINIMUM), NUMBER_ONE, NUMBER_TWO,
		sql.Generation(bits.WORD_64_MINIMUM),
		sql.Generation(bits.WORD_64_MAXIMUM),
	}
	states := []sql.Slot_State{
		sql.SLOT_EMPTY, sql.SLOT_CONNECTION_CREATION, sql.SLOT_IDLE,
		sql.SLOT_EMPTY, sql.SLOT_CLOSE,
	}
	for index, count := range counts {
		stored := sql.Statement{
			Connection: sql.Connection{
				Slot_Index: indexes[index], Generation: generations[index],
			},
			Driver_Statement: driver.Statement{Argument_Count: count},
			Return_State:     states[index],
		}
		var connection sql.Connection
		statement := stored
		sql.Connection_Prepare(&connection, "", &statement)
		state := fake_state{}
		pool := test_pool(t, &state)
		statement = stored
		sql.Pool_Prepare(&pool, "", &statement)
	}
}

func test_transaction_output_storage_domains(t *testing.T) {
	options := driver.Transaction_Options_Of(driver.ISOLATION_DEFAULT, false)
	indexes := []sql.Slot_Index{
		NUMBER_ZERO, NUMBER_ONE, NUMBER_TWO, sql.SLOT_INDEX_MAXIMUM,
	}
	generations := []sql.Generation{
		sql.Generation(bits.WORD_64_MINIMUM), NUMBER_ONE, NUMBER_TWO,
		sql.Generation(bits.WORD_64_MAXIMUM),
	}
	states := []sql.Slot_State{
		sql.SLOT_EMPTY, sql.SLOT_CONNECTION_CREATION, sql.SLOT_IDLE, sql.SLOT_CLOSE,
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
		var connection sql.Connection
		transaction := stored
		sql.Connection_Begin(&connection, options, &transaction)
		state := fake_state{}
		pool := test_pool(t, &state)
		transaction = stored
		sql.Pool_Begin(&pool, options, &transaction)
	}
}

func test_statistics_result_domains(t *testing.T) {
	testify.Panics(t, func() {
		var pool sql.Pool
		sql.Pool_Statistics(&pool)
	}, "zero pool statistics")
	for _, count := range []int{
		NUMBER_ONE, NUMBER_TWO, sql.CONNECTION_COUNT_MAXIMUM,
	} {
		state := fake_state{}
		pool := domain_pool(NUMBER_ZERO, count, false)
		for index := range pool.Slots {
			pool.Slots[index].State = sql.SLOT_IDLE
			fake_connect(
				unsafe.Pointer(&state), "", &pool.Slots[index].Connection,
			)
		}
		sql.Pool_Statistics(&pool)

		pool = domain_pool(NUMBER_ZERO, count, false)
		for index := range pool.Slots {
			pool.Slots[index].State = sql.SLOT_CONNECTION
		}
		sql.Pool_Statistics(&pool)
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
		state := fake_state{}
		storage := make([]sql.Slot, slot_counts[index])
		var pool sql.Pool
		sql.Pool_Init(
			&pool, fake_driver(&state), driver.Data_Source(text_of(data_sizes[index])),
			storage,
		)
		connection, _ := sql.Pool_Connection_Acquire(&pool)
		state.Return_Status = driver.STATUS_BAD_CONNECTION
		sql.Connection_Probe(&connection)
	}
}

func test_initialization_connection_domains() {
	indexes := []sql.Slot_Index{
		NUMBER_ZERO, NUMBER_ONE, NUMBER_TWO, sql.SLOT_INDEX_MAXIMUM,
	}
	generations := []sql.Generation{
		NUMBER_ONE, NUMBER_TWO, sql.Generation(bits.WORD_64_MAXIMUM), NUMBER_ONE,
	}
	request := driver.Request{}
	options := driver.Transaction_Options_Of(driver.ISOLATION_DEFAULT, false)
	for index := range indexes {
		state := fake_state{}
		storage := make([]sql.Slot, sql.CONNECTION_COUNT_MAXIMUM)
		pool := sql.Pool{
			Driver:      fake_driver(&state),
			Data_Source: "",
			Slots:       storage,
			Closed:      sql.POOL_OPEN,
		}
		slot := &storage[indexes[index]]
		fake_connect(unsafe.Pointer(&state), "", &slot.Connection)
		slot.State = sql.SLOT_CONNECTION
		slot.Generation = generations[index]
		connection := sql.Connection{
			Pool:       unsafe.Pointer(&pool),
			Slot_Index: indexes[index],
			Generation: generations[index],
		}
		var rows sql.Rows
		sql.Connection_Query(&connection, request, &rows)
		sql.Rows_Close(&rows)
		var statement sql.Statement
		sql.Connection_Prepare(&connection, "", &statement)
		sql.Statement_Close(&statement)
		var transaction sql.Transaction
		sql.Connection_Begin(&connection, options, &transaction)
		sql.Transaction_Rollback(&transaction)
	}
}

func test_driver_arguments(count int) (arguments driver.Arguments) {
	arguments = make(driver.Arguments, count)
	for index := range arguments {
		var value driver.Value
		driver.Value_Null(&value)
		driver.Named_Value_Of(
			&arguments[index], "",
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

func test_pool(t *testing.T, state *fake_state) (pool sql.Pool) {
	t.Helper()
	data_source, validation := driver.Data_Source_Validate("memory")
	testify.Equal_Values(t, driver.STATUS_OK, validation, "data source")
	var slots [POOL_CAPACITY]sql.Slot
	status := sql.Pool_Init(
		&pool, fake_driver(state), data_source, slots[:],
	)
	testify.Equal_Values(t, sql.STATUS_OK, status, "pool init")
	return pool
}

func test_request(t *testing.T) (request driver.Request) {
	t.Helper()
	query, validation := driver.Query_Validate("select value")
	testify.Equal_Values(t, driver.STATUS_OK, validation, "query")
	request, validation = driver.Request_Of(query, nil)
	testify.Equal_Values(t, driver.STATUS_OK, validation, "request")
	return request
}

func fake_driver(state *fake_state) (injected driver.Driver) {
	return driver.Driver{State: unsafe.Pointer(state), Connect_Procedure: fake_connect}
}

func fake_connect(
	state unsafe.Pointer, data_source driver.Data_Source,
	destination driver.Connection_Pointer,
) (status driver.Status) {
	fake := (*fake_state)(state)
	fake.Connect_Count++
	destination.State = state
	destination.Close_Procedure = fake_close
	destination.Probe_Procedure = fake_probe
	destination.Exec_Procedure = fake_exec
	destination.Query_Procedure = fake_query
	destination.Prepare_Procedure = fake_prepare
	destination.Begin_Procedure = fake_begin
	return fake.Return_Status
}

func fake_close(state unsafe.Pointer) (status driver.Status) {
	fake := (*fake_state)(state)
	fake.Close_Count++
	return fake.Return_Status
}

func fake_probe(state unsafe.Pointer) (status driver.Status) {
	fake := (*fake_state)(state)
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
	state unsafe.Pointer, request driver.Request, result driver.Result_Pointer,
) (status driver.Status) {
	driver.Result_Set_Rows_Affected(result, ROW_COLUMN_COUNT)
	return (*fake_state)(state).Return_Status
}

func fake_query(
	state unsafe.Pointer, request driver.Request, rows driver.Rows_Pointer,
) (status driver.Status) {
	(*fake_state)(state).Row_Position = NUMBER_ZERO
	fake_rows(state, rows)
	return (*fake_state)(state).Return_Status
}

func fake_rows(state unsafe.Pointer, rows driver.Rows_Pointer) {
	rows.State = state
	rows.Column_Count = ROW_COLUMN_COUNT
	rows.Next_Procedure = fake_rows_next
	rows.Close_Procedure = fake_rows_close
}

func fake_rows_next(state unsafe.Pointer, destination driver.Values) (status driver.Status) {
	fake := (*fake_state)(state)
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

func fake_rows_close(state unsafe.Pointer) (status driver.Status) {
	return (*fake_state)(state).Return_Status
}

func fake_prepare(
	state unsafe.Pointer, query driver.Query, statement driver.Statement_Pointer,
) (status driver.Status) {
	statement.State = state
	statement.Argument_Count = NUMBER_ZERO
	statement.Exec_Procedure = fake_statement_exec
	statement.Query_Procedure = fake_statement_query
	statement.Close_Procedure = fake_statement_close
	return (*fake_state)(state).Return_Status
}

func fake_statement_exec(
	state unsafe.Pointer, arguments driver.Arguments, result driver.Result_Pointer,
) (status driver.Status) {
	driver.Result_Set_Rows_Affected(result, ROW_COLUMN_COUNT)
	return (*fake_state)(state).Return_Status
}

func fake_statement_query(
	state unsafe.Pointer, arguments driver.Arguments, rows driver.Rows_Pointer,
) (status driver.Status) {
	(*fake_state)(state).Row_Position = NUMBER_ZERO
	fake_rows(state, rows)
	return (*fake_state)(state).Return_Status
}

func fake_statement_close(state unsafe.Pointer) (status driver.Status) {
	return (*fake_state)(state).Return_Status
}

func fake_begin(
	state unsafe.Pointer, options driver.Transaction_Options,
	transaction driver.Transaction_Pointer,
) (status driver.Status) {
	transaction.State = state
	transaction.Commit_Procedure = fake_commit
	transaction.Rollback_Procedure = fake_rollback
	return (*fake_state)(state).Return_Status
}

func fake_commit(state unsafe.Pointer) (status driver.Status) {
	return (*fake_state)(state).Return_Status
}

func fake_rollback(state unsafe.Pointer) (status driver.Status) {
	return (*fake_state)(state).Return_Status
}
