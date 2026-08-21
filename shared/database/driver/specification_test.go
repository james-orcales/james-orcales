package driver_test

import (
	"testing"
	"unsafe"

	"local/james-orcales/shared/database/driver"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/sim/time"
	"local/james-orcales/shared/slices"
	"local/james-orcales/shared/strings"
	"local/james-orcales/shared/testify"
)

// Test_Values protects closed typed value union and hostile value bounds.
func Test_Values(t *testing.T) {
	null := driver.Value_Null()
	testify.Equal(t, driver.VALUE_NULL, driver.Value_Kind_Of(null), "null kind")

	boolean := driver.Value_Of_Boolean(true)
	boolean_value, status := driver.Value_As_Boolean(boolean)
	testify.Equal_Values(t, driver.STATUS_OK, status, "Boolean status")
	testify.True(t, bool(boolean_value), "Boolean value")

	integer := driver.Value_Of_Integer(driver.Integer(bits.INTEGER_64_MINIMUM))
	integer_value, status := driver.Value_As_Integer(integer)
	testify.Equal_Values(t, driver.STATUS_OK, status, "integer status")
	testify.Equal(t, bits.INTEGER_64_MINIMUM, int64(integer_value), "integer value")
	driver.Value_Of_Integer(driver.Integer(bits.INTEGER_64_MAXIMUM))

	float := driver.Value_Of_Float(driver.Float(bits.WORD_64_MAXIMUM))
	float_value, status := driver.Value_As_Float(float)
	testify.Equal_Values(t, driver.STATUS_OK, status, "float status")
	testify.Equal(t, bits.WORD_64_MAXIMUM, uint64(float_value), "float bits")
	driver.Value_Of_Float(driver.Float(bits.WORD_64_MINIMUM))

	bytes_value, status := driver.Value_Of_Bytes(driver.Bytes_Unvalidated("bytes"))
	testify.Equal_Values(t, driver.STATUS_OK, status, "bytes constructor")
	bytes_result, status := driver.Value_As_Bytes(bytes_value)
	testify.Equal_Values(t, driver.STATUS_OK, status, "bytes status")
	testify.Equal(t, driver.Bytes("bytes"), bytes_result, "bytes value")

	text_value, status := driver.Value_Of_Text(driver.Text_Unvalidated("text"))
	testify.Equal_Values(t, driver.STATUS_OK, status, "text constructor")
	text_result, status := driver.Value_As_Text(text_value)
	testify.Equal_Values(t, driver.STATUS_OK, status, "text status")
	testify.Equal(t, driver.Text("text"), text_result, "text value")

	moment := driver.Value_Of_Time(time.Moment(bits.INTEGER_64_MAXIMUM))
	moment_value, status := driver.Value_As_Time(moment)
	testify.Equal_Values(t, driver.STATUS_OK, status, "time status")
	testify.Equal(t, time.Moment(bits.INTEGER_64_MAXIMUM), moment_value, "time value")
	driver.Value_Of_Time(time.Moment(bits.INTEGER_64_MINIMUM))

	too_large := make([]byte, strings.TEXT_SIZE_MAXIMUM+NUMBER_ONE)
	_, status = driver.Value_Of_Bytes(driver.Bytes_Unvalidated(too_large))
	testify.Equal_Values(t, driver.STATUS_INPUT_INVALID, status, "bytes bound")
	_, status = driver.Value_Of_Text(driver.Text_Unvalidated(string(too_large)))
	testify.Equal_Values(t, driver.STATUS_INPUT_INVALID, status, "text bound")
	_, status = driver.Value_As_Integer(text_value)
	testify.Equal_Values(t, driver.STATUS_INPUT_INVALID, status, "wrong kind")
}

// Test_Named_Values protects standard name rule, ordinal bound, and request validation.
func Test_Named_Values(t *testing.T) {
	value := driver.Value_Of_Integer(NUMBER_ONE)
	_, status := driver.Named_Value_Of("1bad", NUMBER_ONE, value)
	testify.Equal_Values(t, driver.STATUS_INPUT_INVALID, status, "name head")
	_, status = driver.Named_Value_Of("good", NUMBER_ZERO, value)
	testify.Equal_Values(t, driver.STATUS_INPUT_INVALID, status, "ordinal minimum")
	_, status = driver.Named_Value_Of(
		"good", driver.ARGUMENT_COUNT_MAXIMUM+NUMBER_ONE, value,
	)
	testify.Equal_Values(t, driver.STATUS_INPUT_INVALID, status, "ordinal maximum")

	named, status := driver.Named_Value_Of("good", NUMBER_ONE, value)
	testify.Equal_Values(t, driver.STATUS_OK, status, "named value")
	testify.Equal(t, driver.Argument_Name("good"), driver.Named_Value_Name(named), "name")
	testify.Equal(
		t, driver.Argument_Ordinal(NUMBER_ONE),
		driver.Named_Value_Ordinal(named), "ordinal",
	)

	query, status := driver.Query_Validate("select :good")
	testify.Equal_Values(t, driver.STATUS_OK, status, "query")
	request, status := driver.Request_Of(query, driver.Arguments{named})
	testify.Equal_Values(t, driver.STATUS_OK, status, "request construction")
	testify.Equal_Values(t, driver.STATUS_OK, driver.Request_Validate(request), "request")
	request.Argument_Sets[driver.REQUEST_SLOT][NUMBER_ZERO].
		Ordinals[driver.NAMED_VALUE_SLOT] = NUMBER_TWO
	testify.Equal_Values(
		t, driver.STATUS_INPUT_INVALID, driver.Request_Validate(request), "sequence",
	)
}

// Test_Results protects optional standard result counters.
func Test_Results(t *testing.T) {
	var result driver.Result
	_, status := driver.Result_Last_Insert_Identifier(&result)
	testify.Equal_Values(t, driver.STATUS_UNSUPPORTED, status, "absent identifier")
	_, status = driver.Result_Rows_Affected(&result)
	testify.Equal_Values(t, driver.STATUS_UNSUPPORTED, status, "absent affected count")

	driver.Result_Set_Last_Insert_Identifier(
		&result, driver.Last_Insert_Identifier(bits.INTEGER_64_MINIMUM),
	)
	driver.Result_Set_Rows_Affected(&result, driver.Rows_Affected(bits.INTEGER_64_MAXIMUM))
	identifier, status := driver.Result_Last_Insert_Identifier(&result)
	testify.Equal_Values(t, driver.STATUS_OK, status, "identifier status")
	testify.Equal(t, bits.INTEGER_64_MINIMUM, int64(identifier), "identifier")
	affected, status := driver.Result_Rows_Affected(&result)
	testify.Equal_Values(t, driver.STATUS_OK, status, "affected status")
	testify.Equal(t, bits.INTEGER_64_MAXIMUM, int64(affected), "affected")

	driver.Result_Set_Last_Insert_Identifier(
		&result, driver.Last_Insert_Identifier(bits.INTEGER_64_MAXIMUM),
	)
	driver.Result_Set_Rows_Affected(&result, driver.Rows_Affected(bits.INTEGER_64_MINIMUM))
}

// Test_Driver_Surface protects injected connection, execution, query, rows, and close path.
func Test_Driver_Surface(t *testing.T) {
	state := fake_state{}
	injected := fake_driver(&state)
	data_source, validation := driver.Data_Source_Validate("memory")
	testify.Equal_Values(t, driver.STATUS_OK, validation, "data source")
	var connection driver.Connection
	status := driver.Connect(injected, data_source, &connection)
	testify.Equal(t, driver.STATUS_OK, status, "connect")
	testify.Equal(t, driver.STATUS_OK, driver.Connection_Probe(&connection), "probe")

	query, validation := driver.Query_Validate("select value")
	testify.Equal_Values(t, driver.STATUS_OK, validation, "query")
	request, validation := driver.Request_Of(query, nil)
	testify.Equal_Values(t, driver.STATUS_OK, validation, "request")
	var result driver.Result
	status = driver.Connection_Exec(&connection, request, &result)
	testify.Equal(t, driver.STATUS_OK, status, "execute")

	var rows driver.Rows
	status = driver.Connection_Query(&connection, request, &rows)
	testify.Equal(t, driver.STATUS_OK, status, "query rows")
	values := [ROW_COLUMN_COUNT]driver.Value{}
	status = driver.Rows_Next(&rows, values[:])
	testify.Equal(t, driver.STATUS_OK, status, "first row")
	status = driver.Rows_Next(&rows, values[:])
	testify.Equal(t, driver.STATUS_DONE, status, "row end")
	testify.Equal(t, driver.STATUS_OK, driver.Rows_Close(&rows), "rows close")
	testify.Equal(t, driver.STATUS_OK, driver.Connection_Close(&connection), "connection close")
}

// Test_Prepared_Statements protects prepare, argument validation, execution, query, and close.
func Test_Prepared_Statements(t *testing.T) {
	state := fake_state{}
	connection := fake_connection(&state)
	query, validation := driver.Query_Validate("select value")
	testify.Equal_Values(t, driver.STATUS_OK, validation, "query")
	var statement driver.Statement
	status := driver.Connection_Prepare(&connection, query, &statement)
	testify.Equal(t, driver.STATUS_OK, status, "prepare")

	var result driver.Result
	status = driver.Statement_Exec(&statement, nil, &result)
	testify.Equal(t, driver.STATUS_OK, status, "execute")
	var rows driver.Rows
	status = driver.Statement_Query(&statement, nil, &rows)
	testify.Equal(t, driver.STATUS_OK, status, "query rows")
	testify.Equal(t, driver.STATUS_OK, driver.Rows_Close(&rows), "rows close")
	testify.Equal(t, driver.STATUS_OK, driver.Statement_Close(&statement), "statement close")
}

// Test_Transactions protects bounded begin options and exclusive terminal operation.
func Test_Transactions(t *testing.T) {
	state := fake_state{}
	connection := fake_connection(&state)
	options := driver.Transaction_Options_Of(driver.ISOLATION_SERIALIZABLE, true)
	var transaction driver.Transaction
	status := driver.Connection_Begin(&connection, options, &transaction)
	testify.Equal(t, driver.STATUS_OK, status, "begin commit")
	testify.Equal(t, driver.STATUS_OK, driver.Transaction_Commit(&transaction), "commit")

	transaction = driver.Transaction{}
	status = driver.Connection_Begin(&connection, options, &transaction)
	testify.Equal(t, driver.STATUS_OK, status, "begin rollback")
	testify.Equal(t, driver.STATUS_OK, driver.Transaction_Rollback(&transaction), "rollback")
}

// Test_Allocation protects each driver operation without hidden storage.
func Test_Allocation(t *testing.T) {
	state := fake_state{}
	injected := fake_driver(&state)
	data_source, validation := driver.Data_Source_Validate("memory")
	testify.Equal_Values(t, driver.STATUS_OK, validation, "data source")
	query, validation := driver.Query_Validate("select value")
	testify.Equal_Values(t, driver.STATUS_OK, validation, "query")
	request, validation := driver.Request_Of(query, nil)
	testify.Equal_Values(t, driver.STATUS_OK, validation, "request")
	var connection driver.Connection
	var result driver.Result
	var rows driver.Rows
	var statement driver.Statement
	var transaction driver.Transaction
	var values [ROW_COLUMN_COUNT]driver.Value
	var status driver.Status
	options := driver.Transaction_Options_Of(driver.ISOLATION_DEFAULT, false)

	testify.Zero_Allocation(t, func() {
		status = driver.Connect(injected, data_source, &connection)
	})
	testify.Zero_Allocation(t, func() {
		status = driver.Connection_Exec(&connection, request, &result)
	})
	testify.Zero_Allocation(t, func() {
		status = driver.Connection_Query(&connection, request, &rows)
	})
	testify.Zero_Allocation(t, func() { status = driver.Rows_Next(&rows, values[:]) })
	testify.Zero_Allocation(t, func() { status = driver.Rows_Close(&rows) })
	testify.Zero_Allocation(t, func() {
		status = driver.Connection_Prepare(&connection, query, &statement)
	})
	testify.Zero_Allocation(t, func() {
		status = driver.Statement_Exec(&statement, nil, &result)
	})
	testify.Zero_Allocation(t, func() {
		status = driver.Statement_Query(&statement, nil, &rows)
	})
	testify.Zero_Allocation(t, func() { status = driver.Statement_Close(&statement) })
	testify.Zero_Allocation(t, func() {
		status = driver.Connection_Begin(&connection, options, &transaction)
	})
	testify.Zero_Allocation(t, func() { status = driver.Transaction_Commit(&transaction) })
	testify.Zero_Allocation(t, func() {
		transaction = fake_transaction(&state)
		status = driver.Transaction_Rollback(&transaction)
	})
	testify.Zero_Allocation(t, func() { status = driver.Connection_Close(&connection) })
	testify.Equal(t, driver.STATUS_OK, status, "last operation")
}

// Test_Invariant_Domains reaches each registered scalar and collection boundary.
func Test_Invariant_Domains(t *testing.T) {
	t.Helper()
	test_validation_domains()
	test_value_domains()
	test_result_domains()
	test_named_value_domains()
	test_request_domains()
	test_statement_domains()
	test_transaction_domains()
	test_driver_status_domains()
}

func test_validation_domains() {
	for _, size := range []int{
		NUMBER_ZERO, NUMBER_ONE, NUMBER_TWO, strings.TEXT_SIZE_MAXIMUM,
		strings.TEXT_SIZE_MAXIMUM + NUMBER_ONE,
	} {
		text := text_of(size, 'a')
		driver.Query_Validate(driver.Query_Unvalidated(text))
		driver.Data_Source_Validate(driver.Data_Source_Unvalidated(text))
	}

	for _, ordinal := range []int{
		NUMBER_ZERO, NUMBER_ONE, NUMBER_TWO, driver.ARGUMENT_COUNT_MAXIMUM,
		driver.ARGUMENT_COUNT_MAXIMUM + NUMBER_ONE,
	} {
		driver.Named_Value_Of("", driver.Argument_Ordinal_Unvalidated(ordinal),
			driver.Value_Null())
	}
	for _, size := range []int{
		NUMBER_ZERO, NUMBER_ONE, NUMBER_TWO, strings.TEXT_SIZE_MAXIMUM,
		strings.TEXT_SIZE_MAXIMUM + NUMBER_ONE,
	} {
		name := text_of(size, 'a')
		driver.Named_Value_Of(driver.Argument_Name_Unvalidated(name), NUMBER_ONE,
			driver.Value_Null())
	}
}

func test_value_domains() {
	for _, value := range []driver.Boolean{false, true} {
		encoded := driver.Value_Of_Boolean(value)
		driver.Value_As_Boolean(encoded)
	}
	driver.Value_As_Boolean(driver.Value_Null())
	for _, value := range []driver.Integer{
		driver.Integer(bits.INTEGER_64_MINIMUM), NUMBER_NEGATIVE_ONE,
		NUMBER_ZERO, NUMBER_ONE, NUMBER_TWO,
		driver.Integer(bits.INTEGER_64_MAXIMUM),
	} {
		encoded := driver.Value_Of_Integer(value)
		driver.Value_As_Integer(encoded)
	}
	for _, value := range []driver.Float{
		driver.Float(bits.WORD_64_MINIMUM), NUMBER_ONE, NUMBER_TWO,
		driver.Float(bits.WORD_64_MAXIMUM),
	} {
		encoded := driver.Value_Of_Float(value)
		driver.Value_As_Float(encoded)
	}
	driver.Value_As_Float(driver.Value_Null())
	for _, size := range []int{
		NUMBER_ZERO, NUMBER_ONE, NUMBER_TWO, strings.TEXT_SIZE_MAXIMUM,
		strings.TEXT_SIZE_MAXIMUM + NUMBER_ONE,
	} {
		storage := make([]byte, size)
		encoded, status := driver.Value_Of_Bytes(driver.Bytes_Unvalidated(storage))
		if status == driver.Validation_Status(driver.STATUS_OK) {
			driver.Value_As_Bytes(encoded)
		}
		text := driver.Text_Unvalidated(text_of(size, 'a'))
		encoded, status = driver.Value_Of_Text(text)
		if status == driver.Validation_Status(driver.STATUS_OK) {
			driver.Value_As_Text(encoded)
		}
	}
	driver.Value_As_Bytes(driver.Value_Null())
	driver.Value_As_Text(driver.Value_Null())
	for _, value := range []time.Moment{
		time.Moment(bits.INTEGER_64_MINIMUM), NUMBER_NEGATIVE_ONE,
		NUMBER_ZERO, NUMBER_ONE, NUMBER_TWO,
		time.Moment(bits.INTEGER_64_MAXIMUM),
	} {
		encoded := driver.Value_Of_Time(value)
		driver.Value_As_Time(encoded)
	}
	driver.Value_As_Time(driver.Value_Null())

	for _, value := range []driver.Value{
		driver.Value_Null(),
		driver.Value_Of_Boolean(false),
		driver.Value_Of_Integer(NUMBER_ZERO),
		driver.Value_Of_Time(NUMBER_ZERO),
	} {
		driver.Value_Kind_Of(value)
	}
}

func test_result_domains() {
	for _, value := range []int64{
		bits.INTEGER_64_MINIMUM, NUMBER_NEGATIVE_ONE, NUMBER_ZERO,
		NUMBER_ONE, NUMBER_TWO, bits.INTEGER_64_MAXIMUM,
	} {
		var result driver.Result
		driver.Result_Set_Last_Insert_Identifier(
			&result, driver.Last_Insert_Identifier(value),
		)
		driver.Result_Last_Insert_Identifier(&result)
		driver.Result_Set_Rows_Affected(&result, driver.Rows_Affected(value))
		driver.Result_Rows_Affected(&result)
	}
}

func test_named_value_domains() {
	driver.Named_Value_Name(driver.Named_Value{})
	driver.Named_Value_Ordinal(driver.Named_Value{})
	for _, domain := range []struct {
		Name_Size int
		Ordinal   int
	}{
		{Name_Size: NUMBER_ONE, Ordinal: NUMBER_ONE},
		{Name_Size: NUMBER_TWO, Ordinal: NUMBER_TWO},
		{Name_Size: strings.TEXT_SIZE_MAXIMUM, Ordinal: driver.ARGUMENT_COUNT_MAXIMUM},
	} {
		named, status := driver.Named_Value_Of(
			driver.Argument_Name_Unvalidated(text_of(domain.Name_Size, 'a')),
			driver.Argument_Ordinal_Unvalidated(domain.Ordinal), driver.Value_Null(),
		)
		if status == driver.Validation_Status(driver.STATUS_OK) {
			driver.Named_Value_Name(named)
			driver.Named_Value_Ordinal(named)
		}
	}
}

func test_request_domains() {
	for _, size := range []int{
		NUMBER_ZERO, NUMBER_ONE, NUMBER_TWO, strings.TEXT_SIZE_MAXIMUM,
	} {
		query, _ := driver.Query_Validate(driver.Query_Unvalidated(text_of(size, 'a')))
		driver.Request_Of(query, nil)
	}

	arguments := make(driver.Arguments, driver.ARGUMENT_COUNT_MAXIMUM)
	for index := range arguments {
		ordinal := driver.Argument_Ordinal_Unvalidated(index + NUMBER_ONE)
		arguments[index], _ = driver.Named_Value_Of(
			"", ordinal, driver.Value_Null(),
		)
	}
	query, _ := driver.Query_Validate("")
	driver.Request_Of(query, arguments[:NUMBER_TWO])
	driver.Request_Of(query, arguments)

	invalid_value := driver.Value{
		Kinds: [driver.VALUE_SLOT_COUNT]driver.Value_Kind{
			driver.VALUE_TIME + NUMBER_ONE,
		},
	}
	invalid := driver.Named_Value{
		Ordinals: [driver.NAMED_VALUE_SLOT_COUNT]driver.Argument_Ordinal{
			NUMBER_ONE,
		},
		Values:     [driver.NAMED_VALUE_SLOT_COUNT]driver.Value{invalid_value},
		Validities: [driver.NAMED_VALUE_SLOT_COUNT]driver.Named_Value_Validity{true},
	}
	driver.Request_Of(query, driver.Arguments{invalid})
}

func test_statement_domains() {
	state := fake_state{}
	for _, size := range []int{
		NUMBER_ZERO, NUMBER_ONE, NUMBER_TWO, strings.TEXT_SIZE_MAXIMUM,
	} {
		query, _ := driver.Query_Validate(driver.Query_Unvalidated(text_of(size, 'a')))
		connection := fake_connection(&state)
		var statement driver.Statement
		driver.Connection_Prepare(&connection, query, &statement)
	}

	arguments := test_arguments(driver.ARGUMENT_COUNT_MAXIMUM)
	for _, size := range []int{
		NUMBER_ZERO, NUMBER_ONE, NUMBER_TWO, driver.ARGUMENT_COUNT_MAXIMUM,
	} {
		statement := fake_statement(&state, driver.STATEMENT_ARGUMENT_COUNT_UNKNOWN)
		var result driver.Result
		driver.Statement_Exec(&statement, arguments[:size], &result)
		var rows driver.Rows
		driver.Statement_Query(&statement, arguments[:size], &rows)
	}
	for _, count := range []int{
		driver.STATEMENT_ARGUMENT_COUNT_UNKNOWN, NUMBER_ZERO, NUMBER_ONE, NUMBER_TWO,
		driver.ARGUMENT_COUNT_MAXIMUM,
	} {
		statement := fake_statement(&state, driver.Statement_Argument_Count(count))
		size := count
		if count == driver.STATEMENT_ARGUMENT_COUNT_UNKNOWN {
			size = NUMBER_ZERO
		}
		var result driver.Result
		driver.Statement_Exec(&statement, arguments[:size], &result)
	}
	statement := fake_statement(&state, driver.STATEMENT_ARGUMENT_COUNT_UNKNOWN)
	var result driver.Result
	driver.Statement_Exec(&statement, driver.Arguments{{}}, &result)
	statement = fake_statement(&state, NUMBER_ONE)
	driver.Statement_Exec(&statement, nil, &result)
}

func test_transaction_domains() {
	state := fake_state{}
	connection := fake_connection(&state)
	for _, isolation := range []driver.Isolation_Level{
		driver.ISOLATION_DEFAULT, driver.ISOLATION_READ_UNCOMMITTED,
		driver.ISOLATION_READ_COMMITTED, driver.ISOLATION_LINEARIZABLE,
	} {
		for _, read_only := range []driver.Transaction_Read_Only{false, true} {
			options := driver.Transaction_Options_Of(isolation, read_only)
			var transaction driver.Transaction
			driver.Connection_Begin(&connection, options, &transaction)
		}
	}
}

func test_driver_status_domains() {
	state := fake_state{}
	injected := fake_driver(&state)
	for _, size := range []int{
		NUMBER_ZERO, NUMBER_ONE, NUMBER_TWO, strings.TEXT_SIZE_MAXIMUM,
	} {
		data_source, _ := driver.Data_Source_Validate(
			driver.Data_Source_Unvalidated(text_of(size, 'a')),
		)
		var connection driver.Connection
		driver.Connect(injected, data_source, &connection)
	}

	data_source, _ := driver.Data_Source_Validate("")
	query, _ := driver.Query_Validate("")
	request, _ := driver.Request_Of(query, nil)
	options := driver.Transaction_Options_Of(driver.ISOLATION_DEFAULT, false)
	for _, status := range []driver.Status{
		driver.STATUS_OK, driver.STATUS_DONE, driver.STATUS_INPUT_INVALID,
		driver.STATUS_UNSUPPORTED,
	} {
		state.Return_Status = status
		var connection driver.Connection
		driver.Connect(injected, data_source, &connection)
		driver.Connection_Probe(&connection)
		var result driver.Result
		driver.Connection_Exec(&connection, request, &result)
		var rows driver.Rows
		driver.Connection_Query(&connection, request, &rows)
		var statement driver.Statement
		driver.Connection_Prepare(&connection, query, &statement)
		var transaction driver.Transaction
		driver.Connection_Begin(&connection, options, &transaction)
		driver.Connection_Close(&connection)

		rows = fake_rows(&state, ROW_COLUMN_COUNT)
		var values [ROW_COLUMN_COUNT]driver.Value
		driver.Rows_Next(&rows, values[:])
		driver.Rows_Close(&rows)
		statement = fake_statement(&state, NUMBER_ZERO)
		driver.Statement_Exec(&statement, nil, &result)
		driver.Statement_Query(&statement, nil, &rows)
		driver.Statement_Close(&statement)
		transaction = fake_transaction(&state)
		driver.Transaction_Commit(&transaction)
		driver.Transaction_Rollback(&transaction)
	}

	state.Return_Status = driver.STATUS_OK
	for _, size := range []int{
		NUMBER_ZERO, NUMBER_ONE, NUMBER_TWO, driver.ARGUMENT_COUNT_MAXIMUM,
	} {
		state.Row_Position = NUMBER_ZERO
		rows := fake_rows(&state, driver.Column_Count(size))
		driver.Rows_Next(&rows, make(driver.Values, size))
	}
}

func test_arguments(count int) (arguments driver.Arguments) {
	arguments = make(driver.Arguments, count)
	for index := range arguments {
		ordinal := driver.Argument_Ordinal_Unvalidated(index + NUMBER_ONE)
		arguments[index], _ = driver.Named_Value_Of(
			"", ordinal, driver.Value_Null(),
		)
	}
	return arguments
}

const NUMBER_ZERO = slices.COUNT_MINIMUM
const NUMBER_ONE = NUMBER_ZERO + 1
const NUMBER_TWO = NUMBER_ONE + NUMBER_ONE
const NUMBER_NEGATIVE_ONE = -NUMBER_ONE
const ROW_COLUMN_COUNT = NUMBER_ONE

type fake_state struct {
	Row_Position  int
	Return_Status driver.Status
}

func fake_driver(state *fake_state) (injected driver.Driver) {
	return driver.Driver{State: unsafe.Pointer(state), Connect_Procedure: fake_connect}
}

func fake_connection(state *fake_state) (connection driver.Connection) {
	fake_connect(unsafe.Pointer(state), "", &connection)
	return connection
}

func fake_connect(
	state unsafe.Pointer, data_source driver.Data_Source, destination *driver.Connection,
) (status driver.Status) {
	destination.States[driver.CONNECTION_SLOT] = state
	destination.Close_Procedures[driver.CONNECTION_SLOT] = fake_close
	destination.Probe_Procedures[driver.CONNECTION_SLOT] = fake_probe
	destination.Exec_Procedures[driver.CONNECTION_SLOT] = fake_exec
	destination.Query_Procedures[driver.CONNECTION_SLOT] = fake_query
	destination.Prepare_Procedures[driver.CONNECTION_SLOT] = fake_prepare
	destination.Begin_Procedures[driver.CONNECTION_SLOT] = fake_begin
	return (*fake_state)(state).Return_Status
}

func fake_close(state unsafe.Pointer) (status driver.Status) {
	return (*fake_state)(state).Return_Status
}

func fake_probe(state unsafe.Pointer) (status driver.Status) {
	return (*fake_state)(state).Return_Status
}

func fake_exec(
	state unsafe.Pointer, request driver.Request, result *driver.Result,
) (status driver.Status) {
	driver.Result_Set_Rows_Affected(result, ROW_COLUMN_COUNT)
	return (*fake_state)(state).Return_Status
}

func fake_query(
	state unsafe.Pointer, request driver.Request, rows *driver.Rows,
) (status driver.Status) {
	*rows = fake_rows((*fake_state)(state), ROW_COLUMN_COUNT)
	return (*fake_state)(state).Return_Status
}

func fake_prepare(
	state unsafe.Pointer, query driver.Query, statement *driver.Statement,
) (status driver.Status) {
	statement.States[driver.STATEMENT_SLOT] = state
	statement.Argument_Counts[driver.STATEMENT_SLOT] = NUMBER_ZERO
	statement.Exec_Procedures[driver.STATEMENT_SLOT] = fake_statement_exec
	statement.Query_Procedures[driver.STATEMENT_SLOT] = fake_statement_query
	statement.Close_Procedures[driver.STATEMENT_SLOT] = fake_statement_close
	return (*fake_state)(state).Return_Status
}

func fake_statement_exec(
	state unsafe.Pointer, arguments driver.Arguments, result *driver.Result,
) (status driver.Status) {
	driver.Result_Set_Rows_Affected(result, ROW_COLUMN_COUNT)
	return (*fake_state)(state).Return_Status
}

func fake_statement_query(
	state unsafe.Pointer, arguments driver.Arguments, rows *driver.Rows,
) (status driver.Status) {
	*rows = fake_rows((*fake_state)(state), ROW_COLUMN_COUNT)
	return (*fake_state)(state).Return_Status
}

func fake_statement_close(state unsafe.Pointer) (status driver.Status) {
	return (*fake_state)(state).Return_Status
}

func fake_statement(
	state *fake_state, count driver.Statement_Argument_Count,
) (statement driver.Statement) {
	statement.States[driver.STATEMENT_SLOT] = unsafe.Pointer(state)
	statement.Argument_Counts[driver.STATEMENT_SLOT] = count
	statement.Exec_Procedures[driver.STATEMENT_SLOT] = fake_statement_exec
	statement.Query_Procedures[driver.STATEMENT_SLOT] = fake_statement_query
	statement.Close_Procedures[driver.STATEMENT_SLOT] = fake_statement_close
	return statement
}

func fake_begin(
	state unsafe.Pointer, options driver.Transaction_Options,
	transaction *driver.Transaction,
) (status driver.Status) {
	transaction.States[driver.TRANSACTION_SLOT] = state
	transaction.Commit_Procedures[driver.TRANSACTION_SLOT] = fake_commit
	transaction.Rollback_Procedures[driver.TRANSACTION_SLOT] = fake_rollback
	return (*fake_state)(state).Return_Status
}

func fake_commit(state unsafe.Pointer) (status driver.Status) {
	return (*fake_state)(state).Return_Status
}

func fake_rollback(state unsafe.Pointer) (status driver.Status) {
	return (*fake_state)(state).Return_Status
}

func fake_transaction(state *fake_state) (transaction driver.Transaction) {
	transaction.States[driver.TRANSACTION_SLOT] = unsafe.Pointer(state)
	transaction.Commit_Procedures[driver.TRANSACTION_SLOT] = fake_commit
	transaction.Rollback_Procedures[driver.TRANSACTION_SLOT] = fake_rollback
	return transaction
}

func fake_rows_next(state unsafe.Pointer, destination driver.Values) (status driver.Status) {
	fake := (*fake_state)(state)
	if fake.Return_Status != driver.STATUS_OK {
		return fake.Return_Status
	}
	position := &fake.Row_Position
	if *position != NUMBER_ZERO {
		return driver.STATUS_DONE
	}
	for index := range destination {
		destination[index] = driver.Value_Of_Integer(NUMBER_TWO)
	}
	*position++
	return driver.STATUS_OK
}

func fake_rows_close(state unsafe.Pointer) (status driver.Status) {
	return (*fake_state)(state).Return_Status
}

func fake_rows(state *fake_state, count driver.Column_Count) (rows driver.Rows) {
	rows.States[driver.ROWS_SLOT] = unsafe.Pointer(state)
	rows.Column_Counts[driver.ROWS_SLOT] = count
	rows.Next_Procedures[driver.ROWS_SLOT] = fake_rows_next
	rows.Close_Procedures[driver.ROWS_SLOT] = fake_rows_close
	return rows
}

func text_of(size int, value byte) (text string) {
	storage := make([]byte, size)
	for index := range storage {
		storage[index] = value
	}
	return string(storage)
}
