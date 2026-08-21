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
	null := value_null()
	var kind_status driver.Validation_Status
	kind := driver.Value_Kind_Of(null, &kind_status)
	testify.Equal(t, driver.VALUE_NULL, kind, "null kind")
	testify.Equal_Values(t, driver.STATUS_OK, kind_status, "null kind status")

	boolean := value_boolean(true)
	var status driver.Validation_Status
	boolean_value := driver.Value_As_Boolean(boolean, &status)
	testify.Equal_Values(t, driver.STATUS_OK, status, "Boolean status")
	testify.True(t, bool(boolean_value), "Boolean value")

	integer := value_integer(driver.Integer(bits.INTEGER_64_MINIMUM))
	integer_value := driver.Value_As_Integer(integer, &status)
	testify.Equal_Values(t, driver.STATUS_OK, status, "integer status")
	testify.Equal(t, bits.INTEGER_64_MINIMUM, int64(integer_value), "integer value")
	value_integer(driver.Integer(bits.INTEGER_64_MAXIMUM))

	float := value_float(driver.Float(bits.WORD_64_MAXIMUM))
	float_value := driver.Value_As_Float(float, &status)
	testify.Equal_Values(t, driver.STATUS_OK, status, "float status")
	testify.Equal(t, bits.WORD_64_MAXIMUM, uint64(float_value), "float bits")
	value_float(driver.Float(bits.WORD_64_MINIMUM))

	bytes_value, status := value_bytes(driver.Bytes_Unvalidated("bytes"))
	testify.Equal_Values(t, driver.STATUS_OK, status, "bytes constructor")
	bytes_result := driver.Value_As_Bytes(bytes_value, &status)
	testify.Equal_Values(t, driver.STATUS_OK, status, "bytes status")
	testify.Equal(t, driver.Bytes("bytes"), bytes_result, "bytes value")

	text_value, status := value_text(driver.Text_Unvalidated("text"))
	testify.Equal_Values(t, driver.STATUS_OK, status, "text constructor")
	text_result := driver.Value_As_Text(text_value, &status)
	testify.Equal_Values(t, driver.STATUS_OK, status, "text status")
	testify.Equal(t, driver.Text("text"), text_result, "text value")

	moment := value_time(time.Moment(bits.INTEGER_64_MAXIMUM))
	moment_value := driver.Value_As_Time(moment, &status)
	testify.Equal_Values(t, driver.STATUS_OK, status, "time status")
	testify.Equal(t, time.Moment(bits.INTEGER_64_MAXIMUM), moment_value, "time value")
	value_time(time.Moment(bits.INTEGER_64_MINIMUM))

	too_large := make([]byte, strings.TEXT_SIZE_MAXIMUM+NUMBER_ONE)
	_, status = value_bytes(driver.Bytes_Unvalidated(too_large))
	testify.Equal_Values(t, driver.STATUS_INPUT_INVALID, status, "bytes bound")
	_, status = value_text(driver.Text_Unvalidated(string(too_large)))
	testify.Equal_Values(t, driver.STATUS_INPUT_INVALID, status, "text bound")
	driver.Value_As_Integer(text_value, &status)
	testify.Equal_Values(t, driver.STATUS_INPUT_INVALID, status, "wrong kind")
}

// Test_Named_Values protects standard name rule, ordinal bound, and request validation.
func Test_Named_Values(t *testing.T) {
	value := value_integer(NUMBER_ONE)
	_, status := named_value("1bad", NUMBER_ONE, value)
	testify.Equal_Values(t, driver.STATUS_INPUT_INVALID, status, "name head")
	_, status = named_value("good", NUMBER_ZERO, value)
	testify.Equal_Values(t, driver.STATUS_INPUT_INVALID, status, "ordinal minimum")
	_, status = named_value(
		"good", driver.ARGUMENT_COUNT_MAXIMUM+NUMBER_ONE, value,
	)
	testify.Equal_Values(t, driver.STATUS_INPUT_INVALID, status, "ordinal maximum")

	named, status := named_value("good", NUMBER_ONE, value)
	testify.Equal_Values(t, driver.STATUS_OK, status, "named value")
	testify.Equal(t, driver.Argument_Name("good"), driver.Named_Value_Name(named), "name")
	testify.Equal(
		t, driver.Argument_Ordinal(NUMBER_ONE),
		driver.Named_Value_Ordinal(named), "ordinal",
	)

	query, status := query_validate("select :good")
	testify.Equal_Values(t, driver.STATUS_OK, status, "query")
	request, status := request_of(query, driver.Arguments{named})
	testify.Equal_Values(t, driver.STATUS_OK, status, "request construction")
	testify.Equal_Values(t, driver.STATUS_OK, driver.Request_Validate(request), "request")
	request.Arguments[NUMBER_ZERO].Ordinal = NUMBER_TWO
	testify.Equal_Values(
		t, driver.STATUS_INPUT_INVALID, driver.Request_Validate(request), "sequence",
	)
}

// Test_Results protects optional standard result counters.
func Test_Results(t *testing.T) {
	var result driver.Result
	_, status := result_last_insert_identifier(&result)
	testify.Equal_Values(t, driver.STATUS_UNSUPPORTED, status, "absent identifier")
	_, status = result_rows_affected(&result)
	testify.Equal_Values(t, driver.STATUS_UNSUPPORTED, status, "absent affected count")

	driver.Result_Set_Last_Insert_Identifier(
		&result, driver.Last_Insert_Identifier(bits.INTEGER_64_MINIMUM),
	)
	driver.Result_Set_Rows_Affected(&result, driver.Rows_Affected(bits.INTEGER_64_MAXIMUM))
	identifier, status := result_last_insert_identifier(&result)
	testify.Equal_Values(t, driver.STATUS_OK, status, "identifier status")
	testify.Equal(t, bits.INTEGER_64_MINIMUM, int64(identifier), "identifier")
	affected, status := result_rows_affected(&result)
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
	data_source, validation := data_source_validate("memory")
	testify.Equal_Values(t, driver.STATUS_OK, validation, "data source")
	var connection driver.Connection
	status := driver.Connect(injected, data_source, &connection)
	testify.Equal(t, driver.STATUS_OK, status, "connect")
	testify.Equal(t, driver.STATUS_OK, driver.Connection_Probe(&connection), "probe")

	query, validation := query_validate("select value")
	testify.Equal_Values(t, driver.STATUS_OK, validation, "query")
	request, validation := request_of(query, nil)
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
	query, validation := query_validate("select value")
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
	test_value_allocation(t)
	test_metadata_allocation(t)
	test_result_allocation(t)

	state := fake_state{}
	injected := fake_driver(&state)
	data_source, validation := data_source_validate("memory")
	testify.Equal_Values(t, driver.STATUS_OK, validation, "data source")
	query, validation := query_validate("select value")
	testify.Equal_Values(t, driver.STATUS_OK, validation, "query")
	request, validation := request_of(query, nil)
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
		status = driver.Connection_Probe(&connection)
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
	test_status_storage_domains()
	test_value_domains()
	test_result_domains()
	test_named_value_domains()
	test_request_domains()
	test_statement_domains()
	test_transaction_domains()
	test_driver_status_domains()
	test_operation_storage_domains()
}

func test_status_storage_domains() {
	value := value_null()
	validation := driver.Validation_Status(driver.STATUS_INPUT_INVALID)
	driver.Value_Kind_Of(value, &validation)
	validation = driver.Validation_Status(driver.STATUS_INPUT_INVALID)
	driver.Value_As_Boolean(value, &validation)
	validation = driver.Validation_Status(driver.STATUS_INPUT_INVALID)
	driver.Value_As_Integer(value, &validation)
	validation = driver.Validation_Status(driver.STATUS_INPUT_INVALID)
	driver.Value_As_Float(value, &validation)
	validation = driver.Validation_Status(driver.STATUS_INPUT_INVALID)
	driver.Value_As_Bytes(value, &validation)
	validation = driver.Validation_Status(driver.STATUS_INPUT_INVALID)
	driver.Value_As_Text(value, &validation)
	validation = driver.Validation_Status(driver.STATUS_INPUT_INVALID)
	driver.Value_As_Time(value, &validation)
	validation = driver.Validation_Status(driver.STATUS_INPUT_INVALID)
	query := driver.Query_Validate("", &validation)
	validation = driver.Validation_Status(driver.STATUS_INPUT_INVALID)
	driver.Data_Source_Validate("", &validation)
	validation = driver.Validation_Status(driver.STATUS_INPUT_INVALID)
	driver.Request_Of(query, nil, &validation)

	var result driver.Result
	optional := driver.Optional_Status(driver.STATUS_UNSUPPORTED)
	driver.Result_Last_Insert_Identifier(&result, &optional)
	optional = driver.Optional_Status(driver.STATUS_UNSUPPORTED)
	driver.Result_Rows_Affected(&result, &optional)
}

func test_value_allocation(t *testing.T) {
	too_large := make([]byte, strings.TEXT_SIZE_MAXIMUM+NUMBER_ONE)
	bytes_value := driver.Bytes_Unvalidated("bytes")
	text_value := driver.Text_Unvalidated("text")
	invalid_text := driver.Text_Unvalidated(string(too_large))
	moment := time.Moment(bits.INTEGER_64_MAXIMUM)
	var value driver.Value
	testify.Zero_Allocation(t, func() {
		driver.Value_Null(&value)
		value_kind_of(value)
		value_as_boolean(value)
		value_as_integer(value)
		value_as_float(value)
		value_as_bytes(value)
		value_as_text(value)
		value_as_time(value)
		driver.Value_Of_Boolean(&value, true)
		value_as_boolean(value)
		driver.Value_Of_Integer(&value, driver.Integer(bits.INTEGER_64_MINIMUM))
		value_as_integer(value)
		driver.Value_Of_Float(&value, driver.Float(bits.WORD_64_MAXIMUM))
		value_as_float(value)
		driver.Value_Of_Bytes(&value, bytes_value)
		value_as_bytes(value)
		driver.Value_Of_Text(&value, text_value)
		value_as_text(value)
		driver.Value_Of_Time(&value, moment)
		value_as_time(value)
		driver.Value_Of_Bytes(&value, too_large)
		driver.Value_Of_Text(&value, invalid_text)
		value.Kind = driver.Value_Kind_Unvalidated(driver.VALUE_KIND_UNVALIDATED_MAXIMUM)
		value_kind_of(value)
	})
}

func test_metadata_allocation(t *testing.T) {
	too_large := make([]byte, strings.TEXT_SIZE_MAXIMUM+NUMBER_ONE)
	invalid_text := string(too_large)
	var value driver.Value
	var named driver.Named_Value
	var arguments [NUMBER_ONE]driver.Named_Value
	var request driver.Request
	testify.Zero_Allocation(t, func() {
		query, _ := query_validate("select :good")
		query_validate(driver.Query_Unvalidated(invalid_text))
		data_source_validate("memory")
		data_source_validate(driver.Data_Source_Unvalidated(invalid_text))
		driver.Value_Of_Integer(&value, NUMBER_ONE)
		driver.Named_Value_Of(&named, "1bad", NUMBER_ONE, value)
		driver.Named_Value_Of(&named, "good", NUMBER_ZERO, value)
		driver.Named_Value_Of(
			&named, "good", driver.ARGUMENT_COUNT_MAXIMUM+NUMBER_ONE, value,
		)
		driver.Named_Value_Of(&named, "good", NUMBER_ONE, value)
		driver.Named_Value_Name(named)
		driver.Named_Value_Ordinal(named)
		driver.Named_Value_Value(named)
		arguments[NUMBER_ZERO] = named
		request, _ = request_of(query, arguments[:])
		driver.Request_Validate(request)
		arguments[NUMBER_ZERO].Ordinal = NUMBER_TWO
		driver.Request_Validate(request)
		driver.Transaction_Options_Of(driver.ISOLATION_DEFAULT, false)
		driver.Transaction_Options_Of(driver.ISOLATION_LINEARIZABLE, true)
	})
}

func test_result_allocation(t *testing.T) {
	var result driver.Result
	testify.Zero_Allocation(t, func() {
		result = driver.Result{}
		result_last_insert_identifier(&result)
		result_rows_affected(&result)
		driver.Result_Set_Last_Insert_Identifier(
			&result, driver.Last_Insert_Identifier(bits.INTEGER_64_MINIMUM),
		)
		driver.Result_Set_Rows_Affected(
			&result, driver.Rows_Affected(bits.INTEGER_64_MAXIMUM),
		)
		result_last_insert_identifier(&result)
		result_rows_affected(&result)
	})
}

func test_validation_domains() {
	for _, size := range []int{
		NUMBER_ZERO, NUMBER_ONE, NUMBER_TWO, strings.TEXT_SIZE_MAXIMUM,
		strings.TEXT_SIZE_MAXIMUM + NUMBER_ONE,
	} {
		text := text_of(size, 'a')
		query_validate(driver.Query_Unvalidated(text))
		data_source_validate(driver.Data_Source_Unvalidated(text))
	}

	for _, ordinal := range []int{
		NUMBER_ZERO, NUMBER_ONE, NUMBER_TWO, driver.ARGUMENT_COUNT_MAXIMUM,
		driver.ARGUMENT_COUNT_MAXIMUM + NUMBER_ONE,
	} {
		named_value(
			"", driver.Argument_Ordinal_Unvalidated(ordinal), value_null(),
		)
	}
	for _, size := range []int{
		NUMBER_ZERO, NUMBER_ONE, NUMBER_TWO, strings.TEXT_SIZE_MAXIMUM,
		strings.TEXT_SIZE_MAXIMUM + NUMBER_ONE,
	} {
		name := text_of(size, 'a')
		named_value(
			driver.Argument_Name_Unvalidated(name), NUMBER_ONE, value_null(),
		)
	}
}

func test_value_domains() {
	test_value_storage_domains()
	for _, value := range []driver.Boolean{false, true} {
		encoded := value_boolean(value)
		value_as_boolean(encoded)
	}
	value_as_boolean(value_null())
	for _, value := range []driver.Integer{
		driver.Integer(bits.INTEGER_64_MINIMUM), NUMBER_NEGATIVE_ONE,
		NUMBER_ZERO, NUMBER_ONE, NUMBER_TWO,
		driver.Integer(bits.INTEGER_64_MAXIMUM),
	} {
		encoded := value_integer(value)
		value_as_integer(encoded)
	}
	for _, value := range []driver.Float{
		driver.Float(bits.WORD_64_MINIMUM), NUMBER_ONE, NUMBER_TWO,
		driver.Float(bits.WORD_64_MAXIMUM),
	} {
		encoded := value_float(value)
		value_as_float(encoded)
	}
	value_as_float(value_null())
	for _, size := range []int{
		NUMBER_ZERO, NUMBER_ONE, NUMBER_TWO, strings.TEXT_SIZE_MAXIMUM,
		strings.TEXT_SIZE_MAXIMUM + NUMBER_ONE,
	} {
		storage := make([]byte, size)
		encoded, status := value_bytes(driver.Bytes_Unvalidated(storage))
		if status == driver.Validation_Status(driver.STATUS_OK) {
			value_as_bytes(encoded)
		}
		text := driver.Text_Unvalidated(text_of(size, 'a'))
		encoded, status = value_text(text)
		if status == driver.Validation_Status(driver.STATUS_OK) {
			value_as_text(encoded)
		}
	}
	value_as_bytes(value_null())
	value_as_text(value_null())
	for _, value := range []time.Moment{
		time.Moment(bits.INTEGER_64_MINIMUM), NUMBER_NEGATIVE_ONE,
		NUMBER_ZERO, NUMBER_ONE, NUMBER_TWO,
		time.Moment(bits.INTEGER_64_MAXIMUM),
	} {
		encoded := value_time(value)
		value_as_time(encoded)
	}
	value_as_time(value_null())

	for _, value := range []driver.Value{
		value_null(),
		value_boolean(false),
		value_integer(NUMBER_ZERO),
		value_time(NUMBER_ZERO),
	} {
		value_kind_of(value)
	}
}

func test_value_storage_domains() {
	for _, stored := range value_storage_domains() {
		destination := stored
		driver.Value_Null(&destination)
		destination = stored
		driver.Value_Of_Boolean(&destination, stored.Boolean)
		destination = stored
		driver.Value_Of_Integer(&destination, stored.Integer)
		destination = stored
		driver.Value_Of_Float(&destination, stored.Float)
		destination = stored
		driver.Value_Of_Bytes(
			&destination, driver.Bytes_Unvalidated(stored.Bytes),
		)
		destination = stored
		driver.Value_Of_Text(
			&destination, driver.Text_Unvalidated(stored.Text),
		)
		destination = stored
		driver.Value_Of_Time(&destination, stored.Moment)

		value_kind_of(stored)
		value_as_boolean(stored)
		value_as_integer(stored)
		value_as_float(stored)
		value_as_bytes(stored)
		value_as_text(stored)
		value_as_time(stored)
	}
}

func value_storage_domains() (values []driver.Value) {
	type domain struct {
		Kind    driver.Value_Kind_Unvalidated
		Boolean driver.Boolean
		Integer driver.Integer
		Float   driver.Float
		Size    int
		Moment  time.Moment
	}
	domains := []domain{
		{
			Kind:    driver.Value_Kind_Unvalidated(driver.VALUE_NULL),
			Integer: driver.Integer(bits.INTEGER_64_MINIMUM),
			Float:   driver.Float(bits.WORD_64_MINIMUM),
			Moment:  time.Moment(bits.INTEGER_64_MINIMUM),
		},
		{
			Kind:    driver.Value_Kind_Unvalidated(driver.VALUE_BOOLEAN),
			Boolean: true, Integer: NUMBER_NEGATIVE_ONE,
			Float: NUMBER_ONE, Size: NUMBER_ONE, Moment: NUMBER_NEGATIVE_ONE,
		},
		{
			Kind:    driver.Value_Kind_Unvalidated(driver.VALUE_INTEGER),
			Integer: NUMBER_ZERO, Float: NUMBER_TWO,
			Size: NUMBER_TWO, Moment: NUMBER_ZERO,
		},
		{
			Kind:    driver.Value_Kind_Unvalidated(driver.VALUE_FLOAT),
			Integer: NUMBER_ONE, Float: driver.Float(bits.WORD_64_MAXIMUM),
			Size: strings.TEXT_SIZE_MAXIMUM, Moment: NUMBER_ONE,
		},
		{
			Kind:    driver.Value_Kind_Unvalidated(driver.VALUE_BYTES),
			Integer: NUMBER_TWO, Moment: NUMBER_TWO,
		},
		{
			Kind:    driver.Value_Kind_Unvalidated(driver.VALUE_TEXT),
			Integer: driver.Integer(bits.INTEGER_64_MAXIMUM),
			Moment:  time.Moment(bits.INTEGER_64_MAXIMUM),
		},
		{Kind: driver.Value_Kind_Unvalidated(driver.VALUE_TIME)},
		{Kind: driver.Value_Kind_Unvalidated(driver.VALUE_KIND_UNVALIDATED_MAXIMUM)},
	}
	values = make([]driver.Value, len(domains))
	for index := range domains {
		item := domains[index]
		values[index] = driver.Value{
			Kind: item.Kind, Boolean: item.Boolean, Integer: item.Integer,
			Float: item.Float, Bytes: make(driver.Bytes, item.Size),
			Text: driver.Text(text_of(item.Size, 'a')), Moment: item.Moment,
		}
	}
	return values
}

func test_result_domains() {
	for _, result := range result_storage_domains() {
		result_last_insert_identifier(&result)
		result_rows_affected(&result)
	}
	for index, value := range []int64{
		bits.INTEGER_64_MINIMUM, NUMBER_NEGATIVE_ONE, NUMBER_ZERO,
		NUMBER_ONE, NUMBER_TWO, bits.INTEGER_64_MAXIMUM,
	} {
		original := result_storage_domains()[index+NUMBER_ONE]
		result := original
		driver.Result_Set_Last_Insert_Identifier(
			&result, driver.Last_Insert_Identifier(value),
		)
		result_last_insert_identifier(&result)
		result = original
		driver.Result_Set_Rows_Affected(&result, driver.Rows_Affected(value))
		result_rows_affected(&result)
	}
}

func result_storage_domains() (results []driver.Result) {
	results = append(results, driver.Result{})
	for _, value := range []int64{
		bits.INTEGER_64_MINIMUM, NUMBER_NEGATIVE_ONE, NUMBER_ZERO,
		NUMBER_ONE, NUMBER_TWO, bits.INTEGER_64_MAXIMUM,
	} {
		results = append(results, driver.Result{
			Last_Insert_Identifier:          driver.Last_Insert_Identifier(value),
			Rows_Affected:                   driver.Rows_Affected(value),
			Last_Insert_Identifier_Validity: true,
			Rows_Affected_Validity:          true,
		})
	}
	return results
}

func test_named_value_domains() {
	for _, subject := range named_value_storage_domains() {
		destination := subject
		driver.Named_Value_Of(
			&destination, driver.Argument_Name_Unvalidated(subject.Name),
			driver.Argument_Ordinal_Unvalidated(subject.Ordinal), subject.Value,
		)
		driver.Named_Value_Name(subject)
		driver.Named_Value_Ordinal(subject)
		driver.Named_Value_Value(subject)
	}
	for _, domain := range []struct {
		Name_Size int
		Ordinal   int
	}{
		{Name_Size: NUMBER_ONE, Ordinal: NUMBER_ONE},
		{Name_Size: NUMBER_TWO, Ordinal: NUMBER_TWO},
		{Name_Size: strings.TEXT_SIZE_MAXIMUM, Ordinal: driver.ARGUMENT_COUNT_MAXIMUM},
	} {
		named, status := named_value(
			driver.Argument_Name_Unvalidated(text_of(domain.Name_Size, 'a')),
			driver.Argument_Ordinal_Unvalidated(domain.Ordinal), value_null(),
		)
		if status == driver.Validation_Status(driver.STATUS_OK) {
			driver.Named_Value_Name(named)
			driver.Named_Value_Ordinal(named)
		}
	}
}

func named_value_storage_domains() (values []driver.Named_Value) {
	storage := value_storage_domains()
	name_sizes := []int{
		NUMBER_ZERO, NUMBER_ONE, NUMBER_TWO, strings.TEXT_SIZE_MAXIMUM,
		NUMBER_ZERO, NUMBER_ONE, NUMBER_TWO, strings.TEXT_SIZE_MAXIMUM,
	}
	ordinals := []driver.Argument_Ordinal{
		NUMBER_ZERO, NUMBER_ONE, NUMBER_TWO, driver.ARGUMENT_COUNT_MAXIMUM,
		NUMBER_ZERO, NUMBER_ONE, NUMBER_TWO, driver.ARGUMENT_COUNT_MAXIMUM,
	}
	values = make([]driver.Named_Value, len(storage))
	for index := range storage {
		values[index] = driver.Named_Value{
			Name:    driver.Argument_Name(text_of(name_sizes[index], 'a')),
			Ordinal: ordinals[index], Value: storage[index],
			Validity: driver.Named_Value_Validity(index != NUMBER_ZERO),
		}
	}
	return values
}

func test_request_domains() {
	for _, size := range []int{
		NUMBER_ZERO, NUMBER_ONE, NUMBER_TWO, strings.TEXT_SIZE_MAXIMUM,
	} {
		query, _ := query_validate(driver.Query_Unvalidated(text_of(size, 'a')))
		request_of(query, nil)
	}

	arguments := make(driver.Arguments, driver.ARGUMENT_COUNT_MAXIMUM)
	for index := range arguments {
		ordinal := driver.Argument_Ordinal_Unvalidated(index + NUMBER_ONE)
		arguments[index], _ = named_value(
			"", ordinal, value_null(),
		)
	}
	query, _ := query_validate("")
	request_of(query, arguments[:NUMBER_TWO])
	request_of(query, arguments)

	invalid_value := driver.Value{
		Kind: driver.Value_Kind_Unvalidated(driver.VALUE_TIME + NUMBER_ONE),
	}
	invalid := driver.Named_Value{
		Ordinal:  NUMBER_ONE,
		Value:    invalid_value,
		Validity: true,
	}
	request_of(query, driver.Arguments{invalid})
}

func test_statement_domains() {
	state := fake_state{}
	for _, size := range []int{
		NUMBER_ZERO, NUMBER_ONE, NUMBER_TWO, strings.TEXT_SIZE_MAXIMUM,
	} {
		query, _ := query_validate(driver.Query_Unvalidated(text_of(size, 'a')))
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
		data_source, _ := data_source_validate(
			driver.Data_Source_Unvalidated(text_of(size, 'a')),
		)
		var connection driver.Connection
		driver.Connect(injected, data_source, &connection)
	}

	data_source, _ := data_source_validate("")
	query, _ := query_validate("")
	request, _ := request_of(query, nil)
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
	for _, value := range value_storage_domains() {
		state.Row_Position = NUMBER_ZERO
		state.Row_Value = value
		rows := fake_rows(&state, ROW_COLUMN_COUNT)
		driver.Rows_Next(&rows, make(driver.Values, ROW_COLUMN_COUNT))
	}
}

func test_operation_storage_domains() {
	state := fake_state{}
	query, _ := query_validate("")
	request, _ := request_of(query, nil)

	for _, candidate := range request_storage_domains() {
		connection := fake_connection(&state)
		var result driver.Result
		driver.Connection_Exec(&connection, candidate, &result)
		connection = fake_connection(&state)
		var rows driver.Rows
		driver.Connection_Query(&connection, candidate, &rows)
	}

	for _, stored := range result_storage_domains() {
		connection := fake_connection(&state)
		result := stored
		driver.Connection_Exec(&connection, request, &result)
		statement := fake_statement(&state, driver.STATEMENT_ARGUMENT_COUNT_UNKNOWN)
		result = stored
		driver.Statement_Exec(&statement, nil, &result)
	}

	for _, count := range []int{
		driver.STATEMENT_ARGUMENT_COUNT_UNKNOWN, NUMBER_ZERO, NUMBER_ONE,
		NUMBER_TWO, driver.ARGUMENT_COUNT_MAXIMUM,
	} {
		connection := fake_connection(&state)
		statement := fake_statement(&state, driver.Statement_Argument_Count(count))
		driver.Connection_Prepare(&connection, query, &statement)

		statement = fake_statement(&state, driver.Statement_Argument_Count(count))
		driver.Statement_Close(&statement)

		statement = fake_statement(&state, driver.Statement_Argument_Count(count))
		argument_count := count
		if count == driver.STATEMENT_ARGUMENT_COUNT_UNKNOWN {
			argument_count = NUMBER_ZERO
		}
		arguments := test_arguments(argument_count)
		var rows driver.Rows
		driver.Statement_Query(&statement, arguments, &rows)
	}

	for _, count := range []int{
		NUMBER_ZERO, NUMBER_ONE, NUMBER_TWO, slices.SLICE_COUNT_MAXIMUM,
	} {
		rows := fake_rows(&state, driver.Column_Count(count))
		driver.Rows_Close(&rows)

		connection := fake_connection(&state)
		rows = fake_rows(&state, driver.Column_Count(count))
		driver.Connection_Query(&connection, request, &rows)

		statement := fake_statement(&state, driver.STATEMENT_ARGUMENT_COUNT_UNKNOWN)
		rows = fake_rows(&state, driver.Column_Count(count))
		driver.Statement_Query(&statement, nil, &rows)
	}
}

func request_storage_domains() (requests []driver.Request) {
	arguments := test_arguments(driver.ARGUMENT_COUNT_MAXIMUM)
	for _, size := range []int{
		NUMBER_ZERO, NUMBER_ONE, NUMBER_TWO, strings.TEXT_SIZE_MAXIMUM,
	} {
		query, _ := query_validate(
			driver.Query_Unvalidated(text_of(size, 'a')),
		)
		request, _ := request_of(query, nil)
		requests = append(requests, request)
	}
	query, _ := query_validate("")
	for _, size := range []int{
		NUMBER_ZERO, NUMBER_ONE, NUMBER_TWO, driver.ARGUMENT_COUNT_MAXIMUM,
	} {
		request, _ := request_of(query, arguments[:size])
		requests = append(requests, request)
	}
	return requests
}

func test_arguments(count int) (arguments driver.Arguments) {
	arguments = make(driver.Arguments, count)
	for index := range arguments {
		ordinal := driver.Argument_Ordinal_Unvalidated(index + NUMBER_ONE)
		arguments[index], _ = named_value(
			"", ordinal, value_null(),
		)
	}
	return arguments
}

func value_null() (value driver.Value) {
	driver.Value_Null(&value)
	return value
}

func named_value(
	name driver.Argument_Name_Unvalidated,
	ordinal driver.Argument_Ordinal_Unvalidated, value driver.Value,
) (named driver.Named_Value, status driver.Validation_Status) {
	status = driver.Named_Value_Of(&named, name, ordinal, value)
	return named, status
}

func value_boolean(boolean driver.Boolean) (value driver.Value) {
	driver.Value_Of_Boolean(&value, boolean)
	return value
}

func value_integer(integer driver.Integer) (value driver.Value) {
	driver.Value_Of_Integer(&value, integer)
	return value
}

func value_float(float driver.Float) (value driver.Value) {
	driver.Value_Of_Float(&value, float)
	return value
}

func value_bytes(
	bytes driver.Bytes_Unvalidated,
) (value driver.Value, status driver.Validation_Status) {
	status = driver.Value_Of_Bytes(&value, bytes)
	return value, status
}

func value_text(
	text driver.Text_Unvalidated,
) (value driver.Value, status driver.Validation_Status) {
	status = driver.Value_Of_Text(&value, text)
	return value, status
}

func value_time(moment time.Moment) (value driver.Value) {
	driver.Value_Of_Time(&value, moment)
	return value
}

func value_kind_of(
	value driver.Value,
) (kind driver.Value_Kind, status driver.Validation_Status) {
	kind = driver.Value_Kind_Of(value, &status)
	return kind, status
}

func value_as_boolean(
	value driver.Value,
) (result driver.Boolean, status driver.Validation_Status) {
	result = driver.Value_As_Boolean(value, &status)
	return result, status
}

func value_as_integer(
	value driver.Value,
) (result driver.Integer, status driver.Validation_Status) {
	result = driver.Value_As_Integer(value, &status)
	return result, status
}

func value_as_float(
	value driver.Value,
) (result driver.Float, status driver.Validation_Status) {
	result = driver.Value_As_Float(value, &status)
	return result, status
}

func value_as_bytes(
	value driver.Value,
) (result driver.Bytes, status driver.Validation_Status) {
	result = driver.Value_As_Bytes(value, &status)
	return result, status
}

func value_as_text(
	value driver.Value,
) (result driver.Text, status driver.Validation_Status) {
	result = driver.Value_As_Text(value, &status)
	return result, status
}

func value_as_time(
	value driver.Value,
) (result time.Moment, status driver.Validation_Status) {
	result = driver.Value_As_Time(value, &status)
	return result, status
}

func query_validate(
	unvalidated driver.Query_Unvalidated,
) (query driver.Query, status driver.Validation_Status) {
	query = driver.Query_Validate(unvalidated, &status)
	return query, status
}

func data_source_validate(
	unvalidated driver.Data_Source_Unvalidated,
) (data_source driver.Data_Source, status driver.Validation_Status) {
	data_source = driver.Data_Source_Validate(unvalidated, &status)
	return data_source, status
}

func request_of(
	query driver.Query, arguments driver.Arguments,
) (request driver.Request, status driver.Validation_Status) {
	request = driver.Request_Of(query, arguments, &status)
	return request, status
}

func result_last_insert_identifier(
	result driver.Result_Pointer,
) (value driver.Last_Insert_Identifier, status driver.Optional_Status) {
	value = driver.Result_Last_Insert_Identifier(result, &status)
	return value, status
}

func result_rows_affected(
	result driver.Result_Pointer,
) (value driver.Rows_Affected, status driver.Optional_Status) {
	value = driver.Result_Rows_Affected(result, &status)
	return value, status
}

const NUMBER_ZERO = slices.COUNT_MINIMUM
const NUMBER_ONE = NUMBER_ZERO + 1
const NUMBER_TWO = NUMBER_ONE + NUMBER_ONE
const NUMBER_NEGATIVE_ONE = -NUMBER_ONE
const ROW_COLUMN_COUNT = NUMBER_ONE

type fake_state struct {
	Row_Position  int
	Row_Value     driver.Value
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
	state unsafe.Pointer, data_source driver.Data_Source,
	destination driver.Connection_Pointer,
) (status driver.Status) {
	destination.State = state
	destination.Close_Procedure = fake_close
	destination.Probe_Procedure = fake_probe
	destination.Exec_Procedure = fake_exec
	destination.Query_Procedure = fake_query
	destination.Prepare_Procedure = fake_prepare
	destination.Begin_Procedure = fake_begin
	return (*fake_state)(state).Return_Status
}

func fake_close(state unsafe.Pointer) (status driver.Status) {
	return (*fake_state)(state).Return_Status
}

func fake_probe(state unsafe.Pointer) (status driver.Status) {
	return (*fake_state)(state).Return_Status
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
	*rows = fake_rows((*fake_state)(state), ROW_COLUMN_COUNT)
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
	*rows = fake_rows((*fake_state)(state), ROW_COLUMN_COUNT)
	return (*fake_state)(state).Return_Status
}

func fake_statement_close(state unsafe.Pointer) (status driver.Status) {
	return (*fake_state)(state).Return_Status
}

func fake_statement(
	state *fake_state, count driver.Statement_Argument_Count,
) (statement driver.Statement) {
	statement.State = unsafe.Pointer(state)
	statement.Argument_Count = count
	statement.Exec_Procedure = fake_statement_exec
	statement.Query_Procedure = fake_statement_query
	statement.Close_Procedure = fake_statement_close
	return statement
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

func fake_transaction(state *fake_state) (transaction driver.Transaction) {
	transaction.State = unsafe.Pointer(state)
	transaction.Commit_Procedure = fake_commit
	transaction.Rollback_Procedure = fake_rollback
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
		destination[index] = fake.Row_Value
	}
	*position++
	return driver.STATUS_OK
}

func fake_rows_close(state unsafe.Pointer) (status driver.Status) {
	return (*fake_state)(state).Return_Status
}

func fake_rows(state *fake_state, count driver.Column_Count) (rows driver.Rows) {
	rows.State = unsafe.Pointer(state)
	rows.Column_Count = count
	rows.Next_Procedure = fake_rows_next
	rows.Close_Procedure = fake_rows_close
	return rows
}

func text_of(size int, value byte) (text string) {
	storage := make([]byte, size)
	for index := range storage {
		storage[index] = value
	}
	return string(storage)
}
