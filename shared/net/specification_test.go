package network_test

import (
	"testing"
	"unsafe"

	"local/james-orcales/shared/crypto/prng"
	"local/james-orcales/shared/encoding/binary"
	"local/james-orcales/shared/net"
	"local/james-orcales/shared/simulation/nbio"
	"local/james-orcales/shared/simulation/time"
)

// Test_Name protects host grammar before untrusted text reaches wire encoder.
func Test_Name(t *testing.T) {
	const LABEL_MAXIMUM = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	const LABEL_NEAR_MAXIMUM = "ddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
	accepted := [...]string{
		"a",
		"ab",
		"example.com",
		"EXAMPLE.com.",
		"0.example",
		"a-b.example",
		LABEL_MAXIMUM + "." + LABEL_MAXIMUM + "." + LABEL_MAXIMUM + "." +
			LABEL_NEAR_MAXIMUM + ".",
	}
	for _, text := range accepted {
		name, err := network.Name_Validate(network.Name_Unvalidated(text))
		if err != nil {
			t.Fatalf("Name_Validate rejected %q: %v", text, err)
		}
		if string(name) != text {
			t.Fatalf("Name_Validate changed %q", text)
		}
	}

	rejected := [...]string{
		"",
		".",
		"example..com",
		"-example.com",
		"example-.com",
		"_service.example",
		"caf\u00e9.example",
		"abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijkl.example",
		LABEL_MAXIMUM + "." + LABEL_MAXIMUM + "." + LABEL_MAXIMUM + "." +
			LABEL_NEAR_MAXIMUM + ".a",
	}
	for _, text := range rejected {
		_, err := network.Name_Validate(network.Name_Unvalidated(text))
		if err != network.Error_Name_Invalid {
			t.Fatalf("Name_Validate(%q) error = %v", text, err)
		}
	}
}

// Test_Resolver proves UDP resolution and truncated-response TCP fallback.
func Test_Resolver(t *testing.T) {
	resolver_success(t, FAKE_DNS_UDP, network.RECORD_TYPE_A)
	resolver_success(t, FAKE_DNS_UDP, network.RECORD_TYPE_AAAA)
	resolver_success(t, FAKE_DNS_TCP, network.RECORD_TYPE_A)
}

// Test_Response rejects hostile envelopes and follows bounded CNAME chains.
func Test_Response(t *testing.T) {
	resolver_success_response(t, FAKE_DNS_RESPONSE_CNAME, 1)
	resolver_success_response(t, FAKE_DNS_RESPONSE_DUPLICATE, 1)
	resolver_error(t, FAKE_DNS_RESPONSE_NAME_ERROR, network.Error_Name_Not_Found, 2)
	resolver_error(t, FAKE_DNS_RESPONSE_SERVER_FAILURE, network.Error_Server_Failure, 2)
	resolver_error(t, FAKE_DNS_RESPONSE_NO_ADDRESS, network.Error_No_Address, 2)
	resolver_error(t, FAKE_DNS_RESPONSE_TRANSACTION_MISMATCH,
		network.Error_Response_Mismatch, 2)
	resolver_error(t, FAKE_DNS_RESPONSE_QUESTION_MISMATCH,
		network.Error_Response_Mismatch, 2)
	resolver_error(t, FAKE_DNS_RESPONSE_POINTER_FORWARD,
		network.Error_Response_Malformed, 2)
	resolver_response_case(t, FAKE_DNS_RESPONSE_TWO_ADDRESSES, 1,
		network.Error_Result_Too_Small, 1)
	resolver_error(t, FAKE_DNS_RESPONSE_CNAME_ROOT, network.Error_No_Address, 2)
	resolver_named_error(t, FAKE_DNS_RESPONSE_CNAME_ROOT, "a", network.Error_No_Address)
	resolver_error(t, FAKE_DNS_RESPONSE_CNAME_ROOT_DUPLICATE, network.Error_No_Address, 2)
	resolver_error(t, FAKE_DNS_RESPONSE_CNAME_ROOT_LOOP,
		network.Error_Response_Malformed, 2)
	resolver_error(t, FAKE_DNS_RESPONSE_CNAME_MALFORMED,
		network.Error_Response_Malformed, 2)
	resolver_error(t, FAKE_DNS_RESPONSE_CNAME_CONFLICT,
		network.Error_Response_Malformed, 2)
	resolver_named_error(t, FAKE_DNS_RESPONSE_CNAME_SELF, "a",
		network.Error_Response_Malformed)
	resolver_named_error(t, FAKE_DNS_RESPONSE_CNAME_SELF_DUPLICATE,
		resolver_name_maximum(), network.Error_Response_Malformed)
	resolver_error(t, FAKE_DNS_RESPONSE_ANSWER_COUNT_MAXIMUM,
		network.Error_Response_Malformed, 2)
	resolver_error(t, FAKE_DNS_RESPONSE_ADDRESS_SIZE_ZERO,
		network.Error_Response_Malformed, 2)
	resolver_error(t, FAKE_DNS_RESPONSE_ADDRESS_SIZE_ONE,
		network.Error_Response_Malformed, 2)
	resolver_error(t, FAKE_DNS_RESPONSE_ADDRESS_SIZE_TWO,
		network.Error_Response_Malformed, 2)
	resolver_error(t, FAKE_DNS_RESPONSE_END_RESOURCE,
		network.Error_Response_Malformed, 2)
	resolver_success_response(t, FAKE_DNS_RESPONSE_END_ADDRESS, 1)
	resolver_error(t, FAKE_DNS_RESPONSE_END_ALIAS, network.Error_No_Address, 2)
	resolver_error(t, FAKE_DNS_RESPONSE_END_ROOT,
		network.Error_Response_Malformed, 2)
	resolver_error(t, FAKE_DNS_RESPONSE_END_LABEL,
		network.Error_Response_Malformed, 2)
	resolver_error(t, FAKE_DNS_RESPONSE_END_LABEL_SHORT,
		network.Error_Response_Malformed, 2)
	resolver_error(t, FAKE_DNS_RESPONSE_PREFIX_MAXIMUM,
		network.Error_Response_Malformed, 2)
}

// Test_Bounds proves each public ceiling from protocol field formulas.
func Test_Bounds(t *testing.T) {
	if network.DNS_LABEL_BYTES_MAXIMUM != (1<<network.DNS_LABEL_SIZE_BITS)-1 {
		t.Fatal("label bound lost field-width formula")
	}
	if network.DNS_NAME_WIRE_BYTES_MAXIMUM != (1<<network.DNS_NAME_SIZE_BITS)-1 {
		t.Fatal("name bound lost field-width formula")
	}
	if network.DNS_MESSAGE_BYTES_MAXIMUM != (1<<network.DNS_MESSAGE_SIZE_BITS)-1 {
		t.Fatal("message bound lost field-width formula")
	}
	if network.DNS_QUERY_BYTES_MAXIMUM != network.DNS_HEADER_BYTES+
		network.DNS_NAME_WIRE_BYTES_MAXIMUM+network.DNS_QUESTION_FIXED_BYTES {
		t.Fatal("query bound lost component formula")
	}
	if network.DNS_TCP_QUERY_BYTES_MAXIMUM != network.DNS_TCP_SIZE_BYTES+
		network.DNS_QUERY_BYTES_MAXIMUM {
		t.Fatal("TCP query bound lost prefix formula")
	}
	if network.DNS_ADDRESS_COUNT_MAXIMUM != (network.DNS_MESSAGE_BYTES_MAXIMUM-
		network.DNS_QUERY_BYTES_MINIMUM)/network.DNS_IPV4_RESOURCE_BYTES_MINIMUM {
		t.Fatal("address count lost smallest-record formula")
	}
	if network.DNS_CNAME_RESOURCE_BYTES_MINIMUM != network.DNS_COMPRESSION_POINTER_BYTES+
		network.DNS_RESOURCE_FIXED_BYTES+network.DNS_NAME_ROOT_BYTES {
		t.Fatal("CNAME bound lost root-target formula")
	}
	if network.DNS_CNAME_HOPS_MAXIMUM != (network.DNS_MESSAGE_BYTES_MAXIMUM-
		network.DNS_QUERY_BYTES_MINIMUM)/network.DNS_CNAME_RESOURCE_BYTES_MINIMUM {
		t.Fatal("CNAME hop bound lost smallest-record formula")
	}
}

// Test_Allocation proves complete UDP and TCP paths use only retained caller storage.
func Test_Allocation(t *testing.T) {
	resolver_name_allocation(t, "example.com")
	resolver_name_allocation(t, "_service.example")
	resolver_init_allocation(t)
	resolver_allocation(t, FAKE_DNS_UDP)
	resolver_allocation(t, FAKE_DNS_TCP)
}

// Test_Deadline moves injected time between socket creation and connect because the complete
// exchange, not each primitive, owns one budget.
func Test_Deadline(t *testing.T) {
	var fixture resolver_fixture
	fixture.DNS.Virtual = &fixture.Virtual
	fixture.DNS.Tick_On_Socket = true
	fixture.Virtual.Resolution = time.NANOSECOND
	fixture.Generator = prng.New([prng.KEY_BYTES]byte{1}, prng.CURSOR_MIN)
	network.Resolver_Init(
		&fixture.Resolver, fake_dns_loop(&fixture.DNS),
		time.Virtual_Clock_To_Clock(&fixture.Virtual),
		prng.Chacha_To_Source(&fixture.Generator), &fixture.Workspace,
		resolver_configuration(nbio.FAMILY_IPV4),
	)
	network.Resolve(
		&fixture.Resolver, &fixture.Resolver.Completion, "a", network.RECORD_TYPE_A,
		0, fixture.Addresses[:], network.Timeout(time.NANOSECOND),
		resolver_allocation_complete,
	)
	if fixture.Resolver.Completion.Error != time.Deadline_Exceeded {
		t.Fatalf("error = %v", fixture.Resolver.Completion.Error)
	}
	if fixture.DNS.Close_Count != 1 {
		t.Fatalf("close count = %d", fixture.DNS.Close_Count)
	}
}

// Test_Truncated_Envelope blocks spoofed UDP from forcing an accepted TCP retry.
func Test_Truncated_Envelope(t *testing.T) {
	var fixture resolver_fixture
	fixture.DNS.Mode = FAKE_DNS_TCP
	fixture.DNS.Response_Kind = FAKE_DNS_RESPONSE_TRANSACTION_MISMATCH
	fixture.Virtual.Resolution = time.NANOSECOND
	fixture.Generator = prng.New([prng.KEY_BYTES]byte{1}, prng.CURSOR_MIN)
	network.Resolver_Init(
		&fixture.Resolver, fake_dns_loop(&fixture.DNS),
		time.Virtual_Clock_To_Clock(&fixture.Virtual),
		prng.Chacha_To_Source(&fixture.Generator), &fixture.Workspace,
		resolver_configuration(nbio.FAMILY_IPV4),
	)
	network.Resolve(
		&fixture.Resolver, &fixture.Resolver.Completion, "example.com",
		network.RECORD_TYPE_A, 0, fixture.Addresses[:], network.Timeout(time.SECOND),
		resolver_allocation_complete,
	)
	if fixture.Resolver.Completion.Error != network.Error_Response_Mismatch {
		t.Fatalf("error = %v", fixture.Resolver.Completion.Error)
	}
	if fixture.DNS.Socket_Count != 1 {
		t.Fatalf("socket count = %d", fixture.DNS.Socket_Count)
	}
}

// Test_TCP_Partial_Transfers forces every stream primitive to retire one byte because TCP never
// promises whole-buffer progress.
func Test_TCP_Partial_Transfers(t *testing.T) {
	var fixture resolver_fixture
	fixture.DNS.Mode = FAKE_DNS_TCP
	fixture.DNS.TCP_Send_Limit = 1
	fixture.DNS.TCP_Size_Limit = 1
	fixture.DNS.TCP_Response_Limit = 1
	fixture.Virtual.Resolution = time.NANOSECOND
	fixture.Generator = prng.New([prng.KEY_BYTES]byte{1}, prng.CURSOR_MIN)
	network.Resolver_Init(
		&fixture.Resolver, fake_dns_loop(&fixture.DNS),
		time.Virtual_Clock_To_Clock(&fixture.Virtual),
		prng.Chacha_To_Source(&fixture.Generator), &fixture.Workspace,
		resolver_configuration(nbio.FAMILY_IPV4),
	)
	network.Resolve(
		&fixture.Resolver, &fixture.Resolver.Completion, "example.com",
		network.RECORD_TYPE_A, 0, fixture.Addresses[:], network.Timeout(time.SECOND),
		resolver_allocation_complete,
	)
	if fixture.Resolver.Completion.Error != nil {
		t.Fatal(fixture.Resolver.Completion.Error)
	}
	if fixture.Resolver.Completion.Data != 1 {
		t.Fatalf("address count = %d", fixture.Resolver.Completion.Data)
	}
}

// Test_Transport_Errors makes each hostile completion retire exactly once and release every
// descriptor already owned by the resolver.
func Test_Transport_Errors(t *testing.T) {
	resolver_transport_error(t, FAKE_DNS_UDP, FAKE_DNS_FAULT_UDP_SOCKET,
		time.Deadline_Exceeded, 0)
	resolver_transport_error(t, FAKE_DNS_UDP, FAKE_DNS_FAULT_CONNECT,
		time.Deadline_Exceeded, 1)
	resolver_transport_error(t, FAKE_DNS_UDP, FAKE_DNS_FAULT_UDP_SEND,
		time.Deadline_Exceeded, 1)
	resolver_transport_error(t, FAKE_DNS_UDP, FAKE_DNS_FAULT_UDP_RECEIVE,
		time.Deadline_Exceeded, 1)
	resolver_transport_error(t, FAKE_DNS_UDP, FAKE_DNS_FAULT_UDP_SEND_ZERO,
		network.Error_Transfer_No_Progress, 1)
	resolver_transport_error(t, FAKE_DNS_UDP, FAKE_DNS_FAULT_UDP_SEND_OVERSIZE,
		network.Error_Transfer_Count, 1)
	resolver_transport_error(t, FAKE_DNS_UDP, FAKE_DNS_FAULT_UDP_RECEIVE_ZERO,
		network.Error_Transfer_No_Progress, 1)
	resolver_transport_error(t, FAKE_DNS_UDP, FAKE_DNS_FAULT_UDP_RECEIVE_OVERSIZE,
		network.Error_Transfer_Count, 1)
	resolver_transport_error(t, FAKE_DNS_TCP, FAKE_DNS_FAULT_TCP_SOCKET,
		time.Deadline_Exceeded, 1)
	resolver_transport_error(t, FAKE_DNS_TCP, FAKE_DNS_FAULT_TCP_CONNECT,
		time.Deadline_Exceeded, 2)
	resolver_transport_error(t, FAKE_DNS_TCP, FAKE_DNS_FAULT_TCP_SEND_ZERO,
		network.Error_Transfer_No_Progress, 2)
	resolver_transport_error(t, FAKE_DNS_TCP, FAKE_DNS_FAULT_TCP_SIZE_ZERO,
		network.Error_Transfer_No_Progress, 2)
	resolver_transport_error(t, FAKE_DNS_TCP, FAKE_DNS_FAULT_TCP_MESSAGE_ZERO,
		network.Error_Transfer_No_Progress, 2)
	resolver_transport_close_error(t)
}

// Test_Deferred_Completion keeps the first callback outside Resolve's stack because real nbio
// backends retire readiness later.
func Test_Deferred_Completion(t *testing.T) {
	var fixture resolver_fixture
	fixture.DNS.Defer_Connect = true
	fixture.Virtual.Resolution = time.NANOSECOND
	fixture.Generator = prng.New([prng.KEY_BYTES]byte{1}, prng.CURSOR_MIN)
	network.Resolver_Init(
		&fixture.Resolver, fake_dns_loop(&fixture.DNS),
		time.Virtual_Clock_To_Clock(&fixture.Virtual),
		prng.Chacha_To_Source(&fixture.Generator), &fixture.Workspace,
		resolver_configuration(nbio.FAMILY_IPV4),
	)
	callback_count := 0
	network.Resolve(
		&fixture.Resolver, &fixture.Resolver.Completion, "example.com",
		network.RECORD_TYPE_A, 0, fixture.Addresses[:], network.Timeout(time.SECOND),
		func(*time.Completion) { callback_count++ },
	)
	if callback_count != 0 {
		t.Fatal("resolver retired before deferred connect")
	}
	fixture.DNS.Defer_Connect = false
	fixture.DNS.Deferred_Completion.Data = 0
	fixture.DNS.Deferred_Completion.Error = nil
	fixture.DNS.Deferred_Callback(fixture.DNS.Deferred_Completion)
	if callback_count != 1 {
		t.Fatalf("callback count = %d", callback_count)
	}
}

type fake_dns_mode uint8

const FAKE_DNS_UDP fake_dns_mode = 0
const FAKE_DNS_TCP = FAKE_DNS_UDP + 1

const FAKE_DNS_SOCKET nbio.File = nbio.File(network.DNS_NAME_ROOT_BYTES)
const FAKE_DNS_TTL_SECONDS binary.Word_32 = network.DNS_NAME_ROOT_BYTES
const RESOLVER_RESULT_CAPACITY = network.ADDRESS_STORAGE_COUNT_USABLE_MINIMUM +
	network.ADDRESS_STORAGE_COUNT_USABLE_MINIMUM
const TCP_FALLBACK_SOCKET_COUNT = network.ADDRESS_STORAGE_COUNT_USABLE_MINIMUM +
	network.ADDRESS_STORAGE_COUNT_USABLE_MINIMUM
const RESOLVER_ALLOCATION_RUN_COUNT = network.ADDRESS_STORAGE_COUNT_USABLE_MINIMUM
const FAKE_DNS_ADDRESS_LAST_OCTET = byte(network.ADDRESS_STORAGE_COUNT_USABLE_MINIMUM)
const FAKE_DNS_ADDRESS_NEXT_LAST_OCTET = FAKE_DNS_ADDRESS_LAST_OCTET +
	byte(network.ADDRESS_STORAGE_COUNT_USABLE_MINIMUM)

type fake_dns_fault uint8

const FAKE_DNS_FAULT_NONE fake_dns_fault = 0
const FAKE_DNS_FAULT_UDP_SOCKET = FAKE_DNS_FAULT_NONE + 1
const FAKE_DNS_FAULT_CONNECT = FAKE_DNS_FAULT_UDP_SOCKET + 1
const FAKE_DNS_FAULT_UDP_SEND = FAKE_DNS_FAULT_CONNECT + 1
const FAKE_DNS_FAULT_UDP_RECEIVE = FAKE_DNS_FAULT_UDP_SEND + 1
const FAKE_DNS_FAULT_UDP_SEND_ZERO = FAKE_DNS_FAULT_UDP_RECEIVE + 1
const FAKE_DNS_FAULT_UDP_SEND_OVERSIZE = FAKE_DNS_FAULT_UDP_SEND_ZERO + 1
const FAKE_DNS_FAULT_UDP_RECEIVE_ZERO = FAKE_DNS_FAULT_UDP_SEND_OVERSIZE + 1
const FAKE_DNS_FAULT_UDP_RECEIVE_OVERSIZE = FAKE_DNS_FAULT_UDP_RECEIVE_ZERO + 1
const FAKE_DNS_FAULT_TCP_SOCKET = FAKE_DNS_FAULT_UDP_RECEIVE_OVERSIZE + 1
const FAKE_DNS_FAULT_TCP_CONNECT = FAKE_DNS_FAULT_TCP_SOCKET + 1
const FAKE_DNS_FAULT_TCP_SEND_ZERO = FAKE_DNS_FAULT_TCP_CONNECT + 1
const FAKE_DNS_FAULT_TCP_SIZE_ZERO = FAKE_DNS_FAULT_TCP_SEND_ZERO + 1
const FAKE_DNS_FAULT_TCP_MESSAGE_ZERO = FAKE_DNS_FAULT_TCP_SIZE_ZERO + 1

type fake_dns_response uint8

const FAKE_DNS_RESPONSE_ADDRESS fake_dns_response = 0
const FAKE_DNS_RESPONSE_CNAME = FAKE_DNS_RESPONSE_ADDRESS + 1
const FAKE_DNS_RESPONSE_DUPLICATE = FAKE_DNS_RESPONSE_CNAME + 1
const FAKE_DNS_RESPONSE_TWO_ADDRESSES = FAKE_DNS_RESPONSE_DUPLICATE + 1
const FAKE_DNS_RESPONSE_NAME_ERROR = FAKE_DNS_RESPONSE_TWO_ADDRESSES + 1
const FAKE_DNS_RESPONSE_SERVER_FAILURE = FAKE_DNS_RESPONSE_NAME_ERROR + 1
const FAKE_DNS_RESPONSE_NO_ADDRESS = FAKE_DNS_RESPONSE_SERVER_FAILURE + 1
const FAKE_DNS_RESPONSE_TRANSACTION_MISMATCH = FAKE_DNS_RESPONSE_NO_ADDRESS + 1
const FAKE_DNS_RESPONSE_QUESTION_MISMATCH = FAKE_DNS_RESPONSE_TRANSACTION_MISMATCH + 1
const FAKE_DNS_RESPONSE_POINTER_FORWARD = FAKE_DNS_RESPONSE_QUESTION_MISMATCH + 1
const FAKE_DNS_RESPONSE_CNAME_ROOT = FAKE_DNS_RESPONSE_POINTER_FORWARD + 1
const FAKE_DNS_RESPONSE_CNAME_ROOT_DUPLICATE = FAKE_DNS_RESPONSE_CNAME_ROOT + 1
const FAKE_DNS_RESPONSE_CNAME_CONFLICT = FAKE_DNS_RESPONSE_CNAME_ROOT_DUPLICATE + 1
const FAKE_DNS_RESPONSE_CNAME_SELF = FAKE_DNS_RESPONSE_CNAME_CONFLICT + 1
const FAKE_DNS_RESPONSE_CNAME_SELF_DUPLICATE = FAKE_DNS_RESPONSE_CNAME_SELF + 1
const FAKE_DNS_RESPONSE_ANSWER_COUNT_MAXIMUM = FAKE_DNS_RESPONSE_CNAME_SELF_DUPLICATE + 1
const FAKE_DNS_RESPONSE_ADDRESS_SIZE_ZERO = FAKE_DNS_RESPONSE_ANSWER_COUNT_MAXIMUM + 1
const FAKE_DNS_RESPONSE_ADDRESS_SIZE_ONE = FAKE_DNS_RESPONSE_ADDRESS_SIZE_ZERO + 1
const FAKE_DNS_RESPONSE_ADDRESS_SIZE_TWO = FAKE_DNS_RESPONSE_ADDRESS_SIZE_ONE + 1
const FAKE_DNS_RESPONSE_END_RESOURCE = FAKE_DNS_RESPONSE_ADDRESS_SIZE_TWO + 1
const FAKE_DNS_RESPONSE_END_ADDRESS = FAKE_DNS_RESPONSE_END_RESOURCE + 1
const FAKE_DNS_RESPONSE_END_ALIAS = FAKE_DNS_RESPONSE_END_ADDRESS + 1
const FAKE_DNS_RESPONSE_END_ROOT = FAKE_DNS_RESPONSE_END_ALIAS + 1
const FAKE_DNS_RESPONSE_END_LABEL = FAKE_DNS_RESPONSE_END_ROOT + 1
const FAKE_DNS_RESPONSE_END_LABEL_SHORT = FAKE_DNS_RESPONSE_END_LABEL + 1
const FAKE_DNS_RESPONSE_PREFIX_MAXIMUM = FAKE_DNS_RESPONSE_END_LABEL_SHORT + 1
const FAKE_DNS_RESPONSE_CNAME_ROOT_LOOP = FAKE_DNS_RESPONSE_PREFIX_MAXIMUM + 1
const FAKE_DNS_RESPONSE_CNAME_MALFORMED = FAKE_DNS_RESPONSE_CNAME_ROOT_LOOP + 1

type fake_dns struct {
	Mode                   fake_dns_mode
	Fault                  fake_dns_fault
	Close_Error            bool
	Defer_Connect          bool
	Deferred_Completion    *time.Completion
	Deferred_Callback      time.Callback
	Response_Kind          fake_dns_response
	Datagram               bool
	TCP_Send_Limit         int
	TCP_Size_Limit         int
	TCP_Response_Limit     int
	TCP_Response_Truncated bool
	TCP_Frame_Count        int
	TCP_Size_Count         int
	TCP_Response_Count     int
	Tick_On_Socket         bool
	Virtual                *time.Virtual_Clock
	Response               [network.DNS_MESSAGE_BYTES_MAXIMUM]byte
	Response_Count         int
	Socket_Count           int
	Close_Count            int
	Sent_Query             [network.DNS_QUERY_BYTES_MAXIMUM]byte
	TCP_Frame              [network.DNS_TCP_QUERY_BYTES_MAXIMUM]byte
	Sent_Query_Count       int
	Question_End           int
	Alias_Offset           int
}

type resolver_fixture struct {
	DNS       fake_dns
	Virtual   time.Virtual_Clock
	Generator prng.Chacha
	Workspace network.Resolver_Workspace
	Resolver  network.Resolver
	Addresses [RESOLVER_RESULT_CAPACITY]nbio.Address
}

func resolver_transport_error(
	t *testing.T, mode fake_dns_mode, fault fake_dns_fault,
	expected_error error, close_count int,
) {
	t.Helper()
	var fixture resolver_fixture
	fixture.DNS.Mode = mode
	fixture.DNS.Fault = fault
	fixture.Virtual.Resolution = time.NANOSECOND
	fixture.Generator = prng.New([prng.KEY_BYTES]byte{1}, prng.CURSOR_MIN)
	network.Resolver_Init(
		&fixture.Resolver, fake_dns_loop(&fixture.DNS),
		time.Virtual_Clock_To_Clock(&fixture.Virtual),
		prng.Chacha_To_Source(&fixture.Generator), &fixture.Workspace,
		resolver_configuration(nbio.FAMILY_IPV4),
	)
	callback_count := 0
	network.Resolve(
		&fixture.Resolver, &fixture.Resolver.Completion, "example.com",
		network.RECORD_TYPE_A, 0, fixture.Addresses[:], network.Timeout(time.SECOND),
		func(*time.Completion) { callback_count++ },
	)
	if callback_count != 1 {
		t.Fatalf("callback count = %d", callback_count)
	}
	if fixture.Resolver.Completion.Error != expected_error {
		t.Fatalf("error = %v, want %v", fixture.Resolver.Completion.Error, expected_error)
	}
	if fixture.DNS.Close_Count != close_count {
		t.Fatalf("close count = %d, want %d", fixture.DNS.Close_Count, close_count)
	}
}

func resolver_transport_close_error(t *testing.T) {
	t.Helper()
	var fixture resolver_fixture
	fixture.DNS.Fault = FAKE_DNS_FAULT_UDP_RECEIVE
	fixture.DNS.Close_Error = true
	fixture.Virtual.Resolution = time.NANOSECOND
	fixture.Generator = prng.New([prng.KEY_BYTES]byte{1}, prng.CURSOR_MIN)
	network.Resolver_Init(
		&fixture.Resolver, fake_dns_loop(&fixture.DNS),
		time.Virtual_Clock_To_Clock(&fixture.Virtual),
		prng.Chacha_To_Source(&fixture.Generator), &fixture.Workspace,
		resolver_configuration(nbio.FAMILY_IPV4),
	)
	network.Resolve(
		&fixture.Resolver, &fixture.Resolver.Completion, "example.com",
		network.RECORD_TYPE_A, 0, fixture.Addresses[:], network.Timeout(time.SECOND),
		resolver_allocation_complete,
	)
	if fixture.Resolver.Completion.Error != time.Deadline_Exceeded {
		t.Fatalf("error = %v", fixture.Resolver.Completion.Error)
	}
	if fixture.DNS.Close_Count != 1 {
		t.Fatalf("close count = %d", fixture.DNS.Close_Count)
	}
}

func resolver_success(t *testing.T, mode fake_dns_mode, record_type network.Record_Type) {
	t.Helper()
	var fixture resolver_fixture
	fixture.DNS.Mode = mode
	fixture.Virtual.Resolution = time.NANOSECOND
	clock := time.Virtual_Clock_To_Clock(&fixture.Virtual)
	fixture.Generator = prng.New([prng.KEY_BYTES]byte{1}, prng.CURSOR_MIN)
	loop := fake_dns_loop(&fixture.DNS)
	network.Resolver_Init(
		&fixture.Resolver, loop, clock, prng.Chacha_To_Source(&fixture.Generator),
		&fixture.Workspace, network.Resolver_Configuration{
			Server: nbio.Address_IPV4([nbio.IPV4_ADDRESS_BYTES]byte{192, 0, 2, 53}, 53),
			UDP:    resolver_udp_options(),
			TCP:    resolver_tcp_options(),
		},
	)
	name, name_err := network.Name_Validate("example.com")
	if name_err != nil {
		t.Fatal(name_err)
	}
	called := false
	network.Resolve(
		&fixture.Resolver, &fixture.Resolver.Completion, name, record_type,
		network.Port(443),
		fixture.Addresses[:], network.Timeout(time.SECOND),
		func(completion *time.Completion) { called = true },
	)
	if !called {
		t.Fatal("resolver callback did not retire")
	}
	if fixture.Resolver.Completion.Error != nil {
		t.Fatal(fixture.Resolver.Completion.Error)
	}
	if fixture.Resolver.Completion.Data != 1 {
		t.Fatalf("address count = %d", fixture.Resolver.Completion.Data)
	}
	resolver_success_address(t, record_type, fixture.Addresses[0])
	expected_socket_count := 1
	if mode == FAKE_DNS_TCP {
		expected_socket_count++
	}
	if fixture.DNS.Socket_Count != expected_socket_count {
		t.Fatalf("socket count = %d", fixture.DNS.Socket_Count)
	}
	if fixture.DNS.Close_Count != expected_socket_count {
		t.Fatalf("close count = %d", fixture.DNS.Close_Count)
	}
}

func resolver_success_address(
	t *testing.T, record_type network.Record_Type, address nbio.Address,
) {
	t.Helper()
	if address.Port != 443 {
		t.Fatalf("port = %d", address.Port)
	}
	if record_type == network.RECORD_TYPE_A {
		if address.Family != nbio.FAMILY_IPV4 {
			t.Fatalf("family = %d", address.Family)
		}
		valid := address.IP[0] == 203
		if valid {
			valid = address.IP[1] == 0
		}
		if valid {
			valid = address.IP[2] == 113
		}
		if valid {
			valid = address.IP[3] == FAKE_DNS_ADDRESS_LAST_OCTET
		}
		if !valid {
			t.Fatalf("IPv4 address = %v", address.IP[:nbio.IPV4_ADDRESS_BYTES])
		}
	} else {
		if address.Family != nbio.FAMILY_IPV6 {
			t.Fatalf("family = %d", address.Family)
		}
		valid := address.IP[0] == 0x20
		if valid {
			valid = address.IP[1] == 0x01
		}
		if valid {
			valid = address.IP[15] == FAKE_DNS_ADDRESS_LAST_OCTET
		}
		if !valid {
			t.Fatalf("IPv6 address = %v", address.IP)
		}
	}
}

func resolver_success_response(
	t *testing.T, response_kind fake_dns_response, result_count int,
) {
	resolver_response_case(t, response_kind, RESOLVER_RESULT_CAPACITY, nil, result_count)
}

func resolver_error(
	t *testing.T, response_kind fake_dns_response, expected_error error, result_capacity int,
) {
	resolver_response_case(t, response_kind, result_capacity, expected_error, 0)
}

func resolver_named_error(
	t *testing.T, response_kind fake_dns_response, text string, expected_error error,
) {
	t.Helper()
	var fixture resolver_fixture
	fixture.DNS.Response_Kind = response_kind
	if response_kind >= FAKE_DNS_RESPONSE_END_RESOURCE {
		if response_kind <= FAKE_DNS_RESPONSE_END_LABEL_SHORT {
			fixture.DNS.Mode = FAKE_DNS_TCP
		}
	}
	fixture.Virtual.Resolution = time.NANOSECOND
	fixture.Generator = prng.New([prng.KEY_BYTES]byte{1}, prng.CURSOR_MIN)
	network.Resolver_Init(
		&fixture.Resolver, fake_dns_loop(&fixture.DNS),
		time.Virtual_Clock_To_Clock(&fixture.Virtual),
		prng.Chacha_To_Source(&fixture.Generator), &fixture.Workspace,
		resolver_configuration(nbio.FAMILY_IPV4),
	)
	name, name_error := network.Name_Validate(network.Name_Unvalidated(text))
	if name_error != nil {
		t.Fatal(name_error)
	}
	network.Resolve(
		&fixture.Resolver, &fixture.Resolver.Completion, name, network.RECORD_TYPE_A,
		0, fixture.Addresses[:], network.Timeout(time.SECOND), resolver_allocation_complete,
	)
	if fixture.Resolver.Completion.Error != expected_error {
		t.Fatalf("error = %v, want %v", fixture.Resolver.Completion.Error, expected_error)
	}
}

func resolver_name_maximum() (name string) {
	const LABEL_MAXIMUM = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	const LABEL_NEAR_MAXIMUM = "ddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
	return LABEL_MAXIMUM + "." + LABEL_MAXIMUM + "." + LABEL_MAXIMUM + "." +
		LABEL_NEAR_MAXIMUM + "."
}

func resolver_response_case(
	t *testing.T, response_kind fake_dns_response, result_capacity int,
	expected_error error, result_count int,
) {
	t.Helper()
	var fixture resolver_fixture
	fixture.DNS.Response_Kind = response_kind
	if response_kind >= FAKE_DNS_RESPONSE_END_RESOURCE {
		if response_kind <= FAKE_DNS_RESPONSE_END_LABEL_SHORT {
			fixture.DNS.Mode = FAKE_DNS_TCP
		}
	}
	fixture.Virtual.Resolution = time.NANOSECOND
	fixture.Generator = prng.New([prng.KEY_BYTES]byte{1}, prng.CURSOR_MIN)
	network.Resolver_Init(
		&fixture.Resolver, fake_dns_loop(&fixture.DNS),
		time.Virtual_Clock_To_Clock(&fixture.Virtual),
		prng.Chacha_To_Source(&fixture.Generator), &fixture.Workspace,
		network.Resolver_Configuration{
			Server: nbio.Address_IPV4(
				[nbio.IPV4_ADDRESS_BYTES]byte{192, 0, 2, 53}, 53,
			),
			UDP: resolver_udp_options(), TCP: resolver_tcp_options(),
		},
	)
	name, name_error := network.Name_Validate("example.com")
	if name_error != nil {
		t.Fatal(name_error)
	}
	called := false
	network.Resolve(
		&fixture.Resolver, &fixture.Resolver.Completion, name, network.RECORD_TYPE_A,
		443, fixture.Addresses[:result_capacity], network.Timeout(time.SECOND),
		func(*time.Completion) { called = true },
	)
	if !called {
		t.Fatal("resolver callback did not retire")
	}
	if fixture.Resolver.Completion.Error != expected_error {
		t.Fatalf("error = %v, want %v", fixture.Resolver.Completion.Error, expected_error)
	}
	if fixture.Resolver.Completion.Data != result_count {
		t.Fatalf("address count = %d, want %d",
			fixture.Resolver.Completion.Data, result_count)
	}
}

func resolver_allocation(t *testing.T, mode fake_dns_mode) {
	t.Helper()
	var fixture resolver_fixture
	fixture.DNS.Mode = mode
	fixture.Virtual.Resolution = time.NANOSECOND
	clock := time.Virtual_Clock_To_Clock(&fixture.Virtual)
	fixture.Generator = prng.New([prng.KEY_BYTES]byte{1}, prng.CURSOR_MIN)
	loop := fake_dns_loop(&fixture.DNS)
	configuration := network.Resolver_Configuration{
		Server: nbio.Address_IPV4([nbio.IPV4_ADDRESS_BYTES]byte{192, 0, 2, 53}, 53),
		UDP:    resolver_udp_options(), TCP: resolver_tcp_options(),
	}
	network.Resolver_Init(
		&fixture.Resolver, loop, clock, prng.Chacha_To_Source(&fixture.Generator),
		&fixture.Workspace, configuration,
	)
	name, name_error := network.Name_Validate("example.com")
	if name_error != nil {
		t.Fatal(name_error)
	}
	allocations := testing.AllocsPerRun(RESOLVER_ALLOCATION_RUN_COUNT, func() {
		network.Resolve(
			&fixture.Resolver, &fixture.Resolver.Completion, name,
			network.RECORD_TYPE_A, 443, fixture.Addresses[:],
			network.Timeout(time.SECOND), resolver_allocation_complete,
		)
	})
	if allocations != 0 {
		t.Fatalf("mode %d allocated %.1f times per resolution", mode, allocations)
	}
	if fixture.Resolver.Completion.Error != nil {
		t.Fatal(fixture.Resolver.Completion.Error)
	}
	if fixture.Resolver.Completion.Data != 1 {
		t.Fatalf("address count = %d", fixture.Resolver.Completion.Data)
	}
}

func resolver_name_allocation(t *testing.T, text network.Name_Unvalidated) {
	t.Helper()
	var name network.Name
	var name_error error
	allocations := testing.AllocsPerRun(RESOLVER_ALLOCATION_RUN_COUNT, func() {
		name, name_error = network.Name_Validate(text)
	})
	if allocations != 0 {
		t.Fatalf("Name_Validate allocated %.1f times", allocations)
	}
	if name_error == nil {
		if name == "" {
			t.Fatal("Name_Validate lost accepted name")
		}
	}
}

func resolver_init_allocation(t *testing.T) {
	t.Helper()
	var fixture resolver_fixture
	fixture.Virtual.Resolution = time.NANOSECOND
	clock := time.Virtual_Clock_To_Clock(&fixture.Virtual)
	fixture.Generator = prng.New([prng.KEY_BYTES]byte{1}, prng.CURSOR_MIN)
	loop := fake_dns_loop(&fixture.DNS)
	configuration := resolver_configuration(nbio.FAMILY_IPV4)
	allocations := testing.AllocsPerRun(RESOLVER_ALLOCATION_RUN_COUNT, func() {
		network.Resolver_Init(
			&fixture.Resolver, loop, clock, prng.Chacha_To_Source(&fixture.Generator),
			&fixture.Workspace, configuration,
		)
	})
	if allocations != 0 {
		t.Fatalf("Resolver_Init allocated %.1f times", allocations)
	}
}

func resolver_allocation_complete(completion *time.Completion) {
	if completion == nil {
		panic("resolver allocation callback lost completion")
	}
}

func fake_dns_loop(state *fake_dns) (loop nbio.IO) {
	loop.State = unsafe.Pointer(state)
	loop.Close_Procedure = fake_dns_close
	loop.Network = nbio.Network{
		State:                unsafe.Pointer(state),
		Socket_TCP_Procedure: fake_dns_socket_tcp,
		Socket_UDP_Procedure: fake_dns_socket_udp,
		Connect_Procedure:    fake_dns_connect,
		Receive_Procedure:    fake_dns_receive,
		Send_Procedure:       fake_dns_send,
	}
	return loop
}

func fake_dns_socket_tcp(
	state_pointer unsafe.Pointer, _ nbio.Address_Family, _ nbio.TCP_Options,
) (socket nbio.File, err error) {
	state := (*fake_dns)(state_pointer)
	state.Datagram = false
	state.TCP_Frame_Count = 0
	state.TCP_Size_Count = 0
	state.TCP_Response_Count = 0
	state.Socket_Count++
	if state.Fault == FAKE_DNS_FAULT_TCP_SOCKET {
		return network.DNS_SOCKET_INVALID, time.Deadline_Exceeded
	}
	return FAKE_DNS_SOCKET, nil
}

func fake_dns_socket_udp(
	state_pointer unsafe.Pointer, _ nbio.Address_Family, _ nbio.UDP_Options,
) (socket nbio.File, err error) {
	state := (*fake_dns)(state_pointer)
	state.Datagram = true
	state.Socket_Count++
	if state.Fault == FAKE_DNS_FAULT_UDP_SOCKET {
		return network.DNS_SOCKET_INVALID, time.Deadline_Exceeded
	}
	if state.Tick_On_Socket {
		time.Virtual_Clock_Tick(state.Virtual)
	}
	return FAKE_DNS_SOCKET, nil
}

func fake_dns_connect(
	state_pointer unsafe.Pointer, completion *time.Completion, _ nbio.File, _ nbio.Address,
	_ time.Duration, callback time.Callback,
) {
	state := (*fake_dns)(state_pointer)
	if state.Defer_Connect {
		state.Deferred_Completion = completion
		state.Deferred_Callback = callback
		return
	}
	completion.Data = 0
	completion.Error = nil
	if state.Fault == FAKE_DNS_FAULT_CONNECT {
		completion.Error = time.Deadline_Exceeded
	}
	if state.Fault == FAKE_DNS_FAULT_TCP_CONNECT {
		if !state.Datagram {
			completion.Error = time.Deadline_Exceeded
		}
	}
	callback(completion)
}

func fake_dns_send(
	state_pointer unsafe.Pointer, completion *time.Completion, _ nbio.File, buffer []byte,
	_ time.Duration, callback time.Callback,
) {
	state := (*fake_dns)(state_pointer)
	if state.Fault == FAKE_DNS_FAULT_UDP_SEND {
		completion.Data = 0
		completion.Error = time.Deadline_Exceeded
		callback(completion)
		return
	}
	if state.Fault == FAKE_DNS_FAULT_UDP_SEND_ZERO {
		completion.Data = 0
		completion.Error = nil
		callback(completion)
		return
	}
	if state.Fault == FAKE_DNS_FAULT_UDP_SEND_OVERSIZE {
		completion.Data = len(buffer) + 1
		completion.Error = nil
		callback(completion)
		return
	}
	if state.Fault == FAKE_DNS_FAULT_TCP_SEND_ZERO {
		if !state.Datagram {
			completion.Data = 0
			completion.Error = nil
			callback(completion)
			return
		}
	}
	count := len(buffer)
	if !state.Datagram {
		if state.TCP_Send_Limit > 0 {
			count = min(count, state.TCP_Send_Limit)
		}
		copy(state.TCP_Frame[state.TCP_Frame_Count:], buffer[:count])
		state.TCP_Frame_Count += count
		if state.TCP_Frame_Count >= network.DNS_TCP_SIZE_BYTES {
			query_count := int(state.TCP_Frame[0])<<network.DNS_NAME_SIZE_BITS |
				int(state.TCP_Frame[1])
			frame_count := network.DNS_TCP_SIZE_BYTES + query_count
			if state.TCP_Frame_Count == frame_count {
				state.Sent_Query_Count = copy(
					state.Sent_Query[:],
					state.TCP_Frame[network.DNS_TCP_SIZE_BYTES:frame_count],
				)
				fake_dns_response_build(state)
			}
		}
	} else {
		state.Sent_Query_Count = copy(state.Sent_Query[:], buffer)
		fake_dns_response_build(state)
	}
	completion.Data = count
	completion.Error = nil
	callback(completion)
}

func fake_dns_response_build(state *fake_dns) {
	query := state.Sent_Query[:state.Sent_Query_Count]
	response := state.Response[:]
	for index := range response {
		response[index] = 0
	}
	copy(response[:network.DNS_TRANSACTION_BYTES], query[:network.DNS_TRANSACTION_BYTES])
	flags := binary.Word_16(network.DNS_RESPONSE_FLAGS_SUCCESS)
	if state.Response_Kind == FAKE_DNS_RESPONSE_NAME_ERROR {
		flags |= network.DNS_RESPONSE_CODE_NAME_ERROR
	}
	if state.Response_Kind == FAKE_DNS_RESPONSE_SERVER_FAILURE {
		flags |= network.DNS_RESPONSE_CODE_SERVER_FAILURE
	}
	if state.Datagram {
		if state.Mode == FAKE_DNS_TCP {
			flags = binary.Word_16(network.DNS_RESPONSE_FLAGS_TRUNCATED)
		}
	} else if state.TCP_Response_Truncated {
		flags |= network.DNS_TRUNCATED_BIT
	}
	binary.Put_Uint_16(
		response[network.DNS_FLAGS_OFFSET:network.DNS_FLAGS_OFFSET+network.DNS_WORD_BYTES],
		flags, binary.BIG_ENDIAN,
	)
	binary.Put_Uint_16(
		response[network.DNS_QUESTION_COUNT_OFFSET:network.DNS_QUESTION_COUNT_OFFSET+
			network.DNS_WORD_BYTES],
		1, binary.BIG_ENDIAN,
	)
	answer_count := fake_dns_answer_count(state, flags)
	binary.Put_Uint_16(
		response[network.DNS_ANSWER_COUNT_OFFSET:network.DNS_ANSWER_COUNT_OFFSET+
			network.DNS_WORD_BYTES],
		answer_count, binary.BIG_ENDIAN,
	)
	state.Question_End = network.DNS_HEADER_BYTES
	for query[state.Question_End] != 0 {
		state.Question_End += int(query[state.Question_End]) + network.DNS_LABEL_SIZE_BYTES
	}
	state.Question_End += network.DNS_NAME_ROOT_BYTES + network.DNS_QUESTION_FIXED_BYTES
	copy(response[network.DNS_HEADER_BYTES:state.Question_End],
		query[network.DNS_HEADER_BYTES:state.Question_End])
	state.Response_Count = state.Question_End
	if state.Response_Kind == FAKE_DNS_RESPONSE_QUESTION_MISMATCH {
		response[state.Question_End-network.DNS_QUESTION_FIXED_BYTES+1]++
	}
	if state.Response_Kind == FAKE_DNS_RESPONSE_TRANSACTION_MISMATCH {
		response[network.DNS_TRANSACTION_BYTES-1]++
	}
	fake_dns_answers_build(state, answer_count)
}

func fake_dns_answer_count(
	state *fake_dns, flags binary.Word_16,
) (answer_count binary.Word_16) {
	if state.Datagram {
		if flags == binary.Word_16(network.DNS_RESPONSE_FLAGS_TRUNCATED) {
			return 0
		}
	}
	switch state.Response_Kind {
	case FAKE_DNS_RESPONSE_NAME_ERROR, FAKE_DNS_RESPONSE_SERVER_FAILURE,
		FAKE_DNS_RESPONSE_NO_ADDRESS:
		return 0
	case FAKE_DNS_RESPONSE_ANSWER_COUNT_MAXIMUM:
		return binary.Word_16((1 << network.DNS_MESSAGE_SIZE_BITS) - 1)
	case FAKE_DNS_RESPONSE_CNAME, FAKE_DNS_RESPONSE_DUPLICATE,
		FAKE_DNS_RESPONSE_TWO_ADDRESSES, FAKE_DNS_RESPONSE_CNAME_ROOT_DUPLICATE,
		FAKE_DNS_RESPONSE_CNAME_ROOT_LOOP, FAKE_DNS_RESPONSE_CNAME_CONFLICT,
		FAKE_DNS_RESPONSE_CNAME_SELF_DUPLICATE,
		FAKE_DNS_RESPONSE_END_RESOURCE, FAKE_DNS_RESPONSE_END_ADDRESS,
		FAKE_DNS_RESPONSE_END_ALIAS, FAKE_DNS_RESPONSE_END_ROOT,
		FAKE_DNS_RESPONSE_END_LABEL, FAKE_DNS_RESPONSE_END_LABEL_SHORT:
		return 2
	default:
		return 1
	}
}

func fake_dns_answers_build(state *fake_dns, answer_count binary.Word_16) {
	if answer_count == 0 {
		return
	}
	if fake_dns_special_answers_build(state) {
		return
	}
	if state.Response_Kind == FAKE_DNS_RESPONSE_POINTER_FORWARD {
		pointer_start := state.Response_Count
		pointer_end := pointer_start + network.DNS_WORD_BYTES
		binary.Put_Uint_16(
			state.Response[pointer_start:pointer_end],
			binary.Word_16(network.DNS_COMPRESSION_POINTER_TAG|state.Response_Count),
			binary.BIG_ENDIAN,
		)
		state.Response_Count += network.DNS_COMPRESSION_POINTER_BYTES
		return
	}
	if state.Response_Kind == FAKE_DNS_RESPONSE_CNAME {
		fake_dns_cname_append(state)
		fake_dns_address_append(state, binary.Word_16(network.DNS_COMPRESSION_POINTER_TAG|
			state.Alias_Offset), FAKE_DNS_ADDRESS_LAST_OCTET)
		return
	}
	fake_dns_address_append(
		state, binary.Word_16(network.DNS_QUESTION_POINTER),
		FAKE_DNS_ADDRESS_LAST_OCTET,
	)
	if answer_count == 2 {
		last_octet := FAKE_DNS_ADDRESS_LAST_OCTET
		if state.Response_Kind == FAKE_DNS_RESPONSE_TWO_ADDRESSES {
			last_octet = FAKE_DNS_ADDRESS_NEXT_LAST_OCTET
		}
		fake_dns_address_append(
			state, binary.Word_16(network.DNS_QUESTION_POINTER), last_octet,
		)
	}
}

func fake_dns_special_answers_build(state *fake_dns) (built bool) {
	switch state.Response_Kind {
	case FAKE_DNS_RESPONSE_ANSWER_COUNT_MAXIMUM:
		return true
	case FAKE_DNS_RESPONSE_CNAME_ROOT:
		fake_dns_cname_root_append(state)
		return true
	case FAKE_DNS_RESPONSE_CNAME_ROOT_DUPLICATE:
		fake_dns_cname_root_append(state)
		fake_dns_cname_root_append(state)
		return true
	case FAKE_DNS_RESPONSE_CNAME_ROOT_LOOP:
		fake_dns_cname_root_append(state)
		fake_dns_root_cname_root_append(state)
		return true
	case FAKE_DNS_RESPONSE_CNAME_MALFORMED:
		fake_dns_resource_header(
			state, binary.Word_16(network.DNS_QUESTION_POINTER),
			binary.Word_16(network.DNS_RECORD_TYPE_CNAME), 0,
		)
		return true
	case FAKE_DNS_RESPONSE_CNAME_CONFLICT:
		fake_dns_cname_append(state)
		fake_dns_cname_conflict_append(state)
		return true
	case FAKE_DNS_RESPONSE_CNAME_SELF:
		fake_dns_cname_self_append(state)
		return true
	case FAKE_DNS_RESPONSE_CNAME_SELF_DUPLICATE:
		fake_dns_cname_self_append(state)
		fake_dns_cname_self_append(state)
		return true
	case FAKE_DNS_RESPONSE_ADDRESS_SIZE_ZERO,
		FAKE_DNS_RESPONSE_ADDRESS_SIZE_ONE, FAKE_DNS_RESPONSE_ADDRESS_SIZE_TWO:
		fake_dns_address_size_append(state)
		return true
	case FAKE_DNS_RESPONSE_END_RESOURCE, FAKE_DNS_RESPONSE_END_ADDRESS,
		FAKE_DNS_RESPONSE_END_ALIAS, FAKE_DNS_RESPONSE_END_ROOT,
		FAKE_DNS_RESPONSE_END_LABEL, FAKE_DNS_RESPONSE_END_LABEL_SHORT:
		fake_dns_end_answers_build(state)
		return true
	case FAKE_DNS_RESPONSE_PREFIX_MAXIMUM:
		fake_dns_prefix_maximum_append(state)
		return true
	}
	return false
}

func fake_dns_cname_append(state *fake_dns) {
	fake_dns_resource_header(
		state, binary.Word_16(network.DNS_QUESTION_POINTER),
		binary.Word_16(network.DNS_RECORD_TYPE_CNAME), 19,
	)
	state.Alias_Offset = state.Response_Count
	alias := state.Response[state.Response_Count:]
	alias[0] = 5
	copy(alias[1:], "alias")
	alias[6] = 7
	copy(alias[7:], "example")
	alias[14] = 3
	copy(alias[15:], "com")
	alias[18] = 0
	state.Response_Count += 19
}

func fake_dns_cname_root_append(state *fake_dns) {
	fake_dns_resource_header(
		state, binary.Word_16(network.DNS_QUESTION_POINTER),
		binary.Word_16(network.DNS_RECORD_TYPE_CNAME), network.DNS_NAME_ROOT_BYTES,
	)
	state.Response[state.Response_Count] = 0
	state.Response_Count += network.DNS_NAME_ROOT_BYTES
}

func fake_dns_root_cname_root_append(state *fake_dns) {
	response := state.Response[:]
	response[state.Response_Count] = 0
	state.Response_Count += network.DNS_NAME_ROOT_BYTES
	binary.Put_Uint_16(
		response[state.Response_Count:state.Response_Count+network.DNS_WORD_BYTES],
		network.DNS_RECORD_TYPE_CNAME, binary.BIG_ENDIAN,
	)
	state.Response_Count += network.DNS_RECORD_TYPE_BYTES
	binary.Put_Uint_16(
		response[state.Response_Count:state.Response_Count+network.DNS_WORD_BYTES],
		network.DNS_CLASS_IN, binary.BIG_ENDIAN,
	)
	state.Response_Count += network.DNS_CLASS_BYTES
	binary.Put_Uint_32(
		response[state.Response_Count:state.Response_Count+network.DNS_TTL_BYTES],
		FAKE_DNS_TTL_SECONDS, binary.BIG_ENDIAN,
	)
	state.Response_Count += network.DNS_TTL_BYTES
	binary.Put_Uint_16(
		response[state.Response_Count:state.Response_Count+network.DNS_WORD_BYTES],
		network.DNS_NAME_ROOT_BYTES, binary.BIG_ENDIAN,
	)
	state.Response_Count += network.DNS_RESOURCE_SIZE_BYTES
	response[state.Response_Count] = 0
	state.Response_Count += network.DNS_NAME_ROOT_BYTES
}

func fake_dns_cname_self_append(state *fake_dns) {
	fake_dns_resource_header(
		state, binary.Word_16(network.DNS_QUESTION_POINTER),
		binary.Word_16(network.DNS_RECORD_TYPE_CNAME),
		network.DNS_COMPRESSION_POINTER_BYTES,
	)
	binary.Put_Uint_16(
		state.Response[state.Response_Count:state.Response_Count+network.DNS_WORD_BYTES],
		binary.Word_16(network.DNS_QUESTION_POINTER), binary.BIG_ENDIAN,
	)
	state.Response_Count += network.DNS_COMPRESSION_POINTER_BYTES
}

func fake_dns_cname_conflict_append(state *fake_dns) {
	fake_dns_resource_header(
		state, binary.Word_16(network.DNS_QUESTION_POINTER),
		binary.Word_16(network.DNS_RECORD_TYPE_CNAME), 19,
	)
	alias := state.Response[state.Response_Count:]
	alias[0] = 5
	copy(alias[1:], "other")
	alias[6] = 7
	copy(alias[7:], "example")
	alias[14] = 3
	copy(alias[15:], "com")
	alias[18] = 0
	state.Response_Count += 19
}

func fake_dns_address_size_append(state *fake_dns) {
	resource_size := binary.Word_16(
		state.Response_Kind - FAKE_DNS_RESPONSE_ADDRESS_SIZE_ZERO,
	)
	fake_dns_resource_header(
		state, binary.Word_16(network.DNS_QUESTION_POINTER),
		binary.Word_16(network.RECORD_TYPE_A), resource_size,
	)
	state.Response_Count += int(resource_size)
}

func fake_dns_end_answers_build(state *fake_dns) {
	switch state.Response_Kind {
	case FAKE_DNS_RESPONSE_END_RESOURCE:
		fake_dns_padding_append(state, network.DNS_MESSAGE_BYTES_MAXIMUM)
	case FAKE_DNS_RESPONSE_END_ADDRESS:
		fake_dns_padding_append(
			state, network.DNS_MESSAGE_BYTES_MAXIMUM-
				network.DNS_IPV4_RESOURCE_BYTES_MINIMUM,
		)
		fake_dns_address_append(
			state, binary.Word_16(network.DNS_QUESTION_POINTER),
			FAKE_DNS_ADDRESS_LAST_OCTET,
		)
	case FAKE_DNS_RESPONSE_END_ALIAS:
		root_resource_bytes := network.DNS_COMPRESSION_POINTER_BYTES +
			network.DNS_RESOURCE_FIXED_BYTES + network.DNS_NAME_ROOT_BYTES
		fake_dns_padding_append(
			state, network.DNS_MESSAGE_BYTES_MAXIMUM-root_resource_bytes,
		)
		fake_dns_cname_root_append(state)
	case FAKE_DNS_RESPONSE_END_ROOT:
		fake_dns_padding_append(
			state, network.DNS_MESSAGE_BYTES_MAXIMUM-network.DNS_NAME_ROOT_BYTES,
		)
		state.Response[state.Response_Count] = 0
		state.Response_Count++
	case FAKE_DNS_RESPONSE_END_LABEL:
		fake_dns_padding_append(
			state, network.DNS_MESSAGE_BYTES_MAXIMUM-
				network.DNS_LABEL_SIZE_BYTES-network.DNS_LABEL_BYTES_MINIMUM,
		)
		state.Response[state.Response_Count] = network.DNS_LABEL_BYTES_MINIMUM
		state.Response[state.Response_Count+network.DNS_LABEL_SIZE_BYTES] = 'a'
		state.Response_Count += network.DNS_LABEL_SIZE_BYTES +
			network.DNS_LABEL_BYTES_MINIMUM
	case FAKE_DNS_RESPONSE_END_LABEL_SHORT:
		fake_dns_padding_append(
			state, network.DNS_MESSAGE_BYTES_MAXIMUM-network.DNS_LABEL_SIZE_BYTES,
		)
		state.Response[state.Response_Count] = network.DNS_LABEL_BYTES_MINIMUM
		state.Response_Count++
	}
}

func fake_dns_padding_append(state *fake_dns, end_count int) {
	header_start := state.Response_Count
	resource_size := end_count - header_start - network.DNS_COMPRESSION_POINTER_BYTES -
		network.DNS_RESOURCE_FIXED_BYTES
	fake_dns_resource_header(
		state, binary.Word_16(network.DNS_QUESTION_POINTER),
		binary.Word_16(network.RECORD_TYPE_A), binary.Word_16(resource_size),
	)
	class_offset := header_start + network.DNS_COMPRESSION_POINTER_BYTES +
		network.DNS_RECORD_TYPE_BYTES
	state.Response[class_offset+network.DNS_CLASS_BYTES-1] = 0
	state.Response_Count += resource_size
}

func fake_dns_prefix_maximum_append(state *fake_dns) {
	prefix_bytes := network.DNS_NAME_WIRE_BYTES_MAXIMUM - network.DNS_NAME_ROOT_BYTES
	label_count := prefix_bytes / (network.DNS_LABEL_SIZE_BYTES +
		network.DNS_LABEL_BYTES_MINIMUM)
	for label_index := 0; label_index < label_count; label_index++ {
		state.Response[state.Response_Count] = network.DNS_LABEL_BYTES_MINIMUM
		state.Response[state.Response_Count+network.DNS_LABEL_SIZE_BYTES] = 'a'
		state.Response_Count += network.DNS_LABEL_SIZE_BYTES +
			network.DNS_LABEL_BYTES_MINIMUM
	}
	state.Response[state.Response_Count] = network.DNS_LABEL_BYTES_MINIMUM
	state.Response_Count++
}

func fake_dns_address_append(
	state *fake_dns, owner binary.Word_16, last_octet byte,
) {
	query_type_offset := state.Question_End - network.DNS_QUESTION_FIXED_BYTES
	record_type := binary.Uint_16(
		state.Sent_Query[query_type_offset:query_type_offset+network.DNS_WORD_BYTES],
		binary.BIG_ENDIAN,
	)
	address_bytes := nbio.IPV4_ADDRESS_BYTES
	if record_type == binary.Word_16(network.RECORD_TYPE_AAAA) {
		address_bytes = nbio.IPV6_ADDRESS_BYTES
	}
	fake_dns_resource_header(state, owner, record_type, binary.Word_16(address_bytes))
	response := state.Response[:]
	if address_bytes == nbio.IPV4_ADDRESS_BYTES {
		response[state.Response_Count] = 203
		response[state.Response_Count+1] = 0
		response[state.Response_Count+2] = 113
		response[state.Response_Count+3] = last_octet
	} else {
		response[state.Response_Count] = 0x20
		response[state.Response_Count+1] = 0x01
		response[state.Response_Count+address_bytes-1] = last_octet
	}
	state.Response_Count += address_bytes
}

func fake_dns_resource_header(
	state *fake_dns, owner binary.Word_16, record_type binary.Word_16,
	resource_size binary.Word_16,
) {
	response := state.Response[:]
	binary.Put_Uint_16(
		response[state.Response_Count:state.Response_Count+network.DNS_WORD_BYTES],
		owner, binary.BIG_ENDIAN,
	)
	state.Response_Count += network.DNS_COMPRESSION_POINTER_BYTES
	binary.Put_Uint_16(
		response[state.Response_Count:state.Response_Count+network.DNS_WORD_BYTES],
		record_type, binary.BIG_ENDIAN,
	)
	state.Response_Count += network.DNS_RECORD_TYPE_BYTES
	binary.Put_Uint_16(
		response[state.Response_Count:state.Response_Count+network.DNS_WORD_BYTES],
		network.DNS_CLASS_IN, binary.BIG_ENDIAN,
	)
	state.Response_Count += network.DNS_CLASS_BYTES
	binary.Put_Uint_32(
		response[state.Response_Count:state.Response_Count+network.DNS_TTL_BYTES],
		FAKE_DNS_TTL_SECONDS, binary.BIG_ENDIAN,
	)
	state.Response_Count += network.DNS_TTL_BYTES
	binary.Put_Uint_16(
		response[state.Response_Count:state.Response_Count+network.DNS_WORD_BYTES],
		resource_size, binary.BIG_ENDIAN,
	)
	state.Response_Count += network.DNS_RESOURCE_SIZE_BYTES
}

func fake_dns_receive(
	state_pointer unsafe.Pointer, completion *time.Completion, _ nbio.File, buffer []byte,
	_ time.Duration, callback time.Callback,
) {
	state := (*fake_dns)(state_pointer)
	if fake_dns_receive_fault(state, completion, buffer, callback) {
		return
	}
	count := 0
	if state.Datagram {
		count = copy(buffer, state.Response[:state.Response_Count])
	} else if state.TCP_Size_Count < network.DNS_TCP_SIZE_BYTES {
		count = fake_dns_tcp_size_receive(state, buffer)
	} else {
		count = min(len(buffer), state.Response_Count-state.TCP_Response_Count)
		if state.TCP_Response_Limit > 0 {
			count = min(count, state.TCP_Response_Limit)
		}
		copy(
			buffer[:count],
			state.Response[state.TCP_Response_Count:state.TCP_Response_Count+count],
		)
		state.TCP_Response_Count += count
	}
	completion.Data = count
	completion.Error = nil
	callback(completion)
}

func fake_dns_receive_fault(
	state *fake_dns, completion *time.Completion, buffer []byte, callback time.Callback,
) (retired bool) {
	if state.Fault == FAKE_DNS_FAULT_UDP_RECEIVE {
		completion.Data = 0
		completion.Error = time.Deadline_Exceeded
		callback(completion)
		return true
	}
	if state.Fault == FAKE_DNS_FAULT_UDP_RECEIVE_ZERO {
		completion.Data = 0
		completion.Error = nil
		callback(completion)
		return true
	}
	if state.Fault == FAKE_DNS_FAULT_UDP_RECEIVE_OVERSIZE {
		completion.Data = len(buffer) + 1
		completion.Error = nil
		callback(completion)
		return true
	}
	if state.Fault == FAKE_DNS_FAULT_TCP_SIZE_ZERO {
		if !state.Datagram {
			if state.TCP_Size_Count < network.DNS_TCP_SIZE_BYTES {
				completion.Data = 0
				completion.Error = nil
				callback(completion)
				return true
			}
		}
	}
	if state.Fault == FAKE_DNS_FAULT_TCP_MESSAGE_ZERO {
		if !state.Datagram {
			if state.TCP_Size_Count == network.DNS_TCP_SIZE_BYTES {
				completion.Data = 0
				completion.Error = nil
				callback(completion)
				return true
			}
		}
	}
	return false
}

func fake_dns_tcp_size_receive(state *fake_dns, buffer []byte) (count int) {
	remaining_count := network.DNS_TCP_SIZE_BYTES - state.TCP_Size_Count
	count = min(len(buffer), remaining_count)
	if state.TCP_Size_Limit > 0 {
		count = min(count, state.TCP_Size_Limit)
	}
	for prefix_index := 0; prefix_index < count; prefix_index++ {
		prefix_offset := state.TCP_Size_Count + prefix_index
		if prefix_offset == 0 {
			buffer[prefix_index] = byte(
				state.Response_Count >> network.DNS_NAME_SIZE_BITS,
			)
		} else {
			buffer[prefix_index] = byte(state.Response_Count)
		}
	}
	state.TCP_Size_Count += count
	return count
}

func fake_dns_close(
	state_pointer unsafe.Pointer, completion *time.Completion, _ nbio.File,
	callback time.Callback,
) {
	state := (*fake_dns)(state_pointer)
	state.Close_Count++
	completion.Data = 0
	completion.Error = nil
	if state.Close_Error {
		completion.Error = time.Deadline_Exceeded
	}
	callback(completion)
}

func resolver_udp_options() (options nbio.UDP_Options) {
	return nbio.UDP_Options{
		Receive_Buffer_Bytes:    nbio.UDP_RECEIVE_BUFFER_BYTES_DEFAULT,
		Send_Buffer_Bytes:       nbio.UDP_SEND_BUFFER_BYTES_DEFAULT,
		Receive_Low_Water_Bytes: nbio.SOCKET_RECEIVE_LOW_WATER_BYTES_DEFAULT,
		Linger_Timeout:          nbio.SOCKET_LINGER_TIMEOUT_DEFAULT,
	}
}

func resolver_tcp_options() (options nbio.TCP_Options) {
	return nbio.TCP_Options{
		Receive_Buffer_Bytes:     nbio.TCP_RECEIVE_BUFFER_BYTES_DEFAULT,
		Send_Buffer_Bytes:        nbio.TCP_SEND_BUFFER_BYTES_DEFAULT,
		Receive_Low_Water_Bytes:  nbio.SOCKET_RECEIVE_LOW_WATER_BYTES_DEFAULT,
		Linger_Timeout:           nbio.SOCKET_LINGER_TIMEOUT_DEFAULT,
		Maximum_Segment_Bytes:    nbio.TCP_MAXIMUM_SEGMENT_BYTES_DEFAULT,
		Not_Sent_Low_Water_Bytes: nbio.TCP_NOT_SENT_LOW_WATER_BYTES_DEFAULT,
		Keepalive: nbio.TCP_Keepalive{
			Idle:        nbio.TCP_KEEPALIVE_IDLE_DEFAULT,
			Interval:    nbio.TCP_KEEPALIVE_INTERVAL_DEFAULT,
			Probe_Count: nbio.TCP_KEEPALIVE_PROBE_COUNT_DEFAULT,
		},
		No_Delay: nbio.TCP_NO_DELAY_DEFAULT,
	}
}

func resolver_configuration(
	family nbio.Address_Family,
) (configuration network.Resolver_Configuration) {
	server := nbio.Address_IPV4([nbio.IPV4_ADDRESS_BYTES]byte{192, 0, 2, 53}, 53)
	if family == nbio.FAMILY_IPV6 {
		server = nbio.Address_IPV6([nbio.IPV6_ADDRESS_BYTES]byte{0x20, 1, 0, 0}, 53)
	}
	return network.Resolver_Configuration{
		Server: server, UDP: resolver_udp_options(), TCP: resolver_tcp_options(),
	}
}
