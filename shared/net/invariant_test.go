//go:build !invariant_noop

package network_test

import (
	"testing"

	"local/james-orcales/shared/crypto/prng"
	"local/james-orcales/shared/net"
	"local/james-orcales/shared/sim/nbio"
	"local/james-orcales/shared/sim/time"
)

// Test_Invariants reaches boundaries through ownership APIs because direct helper calls would
// prove the assertion machinery instead of the resolver path.
func Test_Invariants(t *testing.T) {
	resolver_public_boundaries(t)
	resolver_state_boundaries(t, false)
	resolver_state_boundaries(t, true)
	resolver_dependency_boundaries(t)
	resolver_control_flow_boundaries(t)
}

func resolver_public_boundaries(t *testing.T) {
	t.Helper()
	resolver_boundary_failure(t)
	resolver_boundary_success(t, "a", 1, 1, time.NANOSECOND)
	resolver_boundary_success(t, "ab", 2, 2, 2*time.NANOSECOND)
	resolver_boundary_success(
		t, resolver_name_maximum(), network.PORT_MAXIMUM,
		network.DNS_ADDRESS_COUNT_MAXIMUM,
		time.Duration(time.MONOTONIC_MOMENT_MAXIMUM),
	)
}

func resolver_boundary_failure(t *testing.T) {
	t.Helper()
	var fixture resolver_fixture
	resolver_fixture_init(&fixture, nbio.FAMILY_IPV4)
	died := did_die(func() {
		network.Resolve(
			&fixture.Resolver, &fixture.Resolver.Completion, "", network.RECORD_TYPE_A,
			network.PORT_MINIMUM, nil, network.Timeout(time.NANOSECOND),
			resolver_allocation_complete,
		)
	})
	if !died {
		t.Fatal("Resolve accepted empty result storage")
	}
}

func resolver_boundary_success(
	t *testing.T, text string, port network.Port, result_capacity_count int,
	timeout time.Duration,
) {
	t.Helper()
	var fixture resolver_fixture
	resolver_fixture_init(&fixture, nbio.FAMILY_IPV4)
	name, name_error := network.Name_Validate(network.Name_Unvalidated(text))
	if name_error != nil {
		t.Fatal(name_error)
	}
	results := address_storage(result_capacity_count)
	network.Resolve(
		&fixture.Resolver, &fixture.Resolver.Completion, name, network.RECORD_TYPE_A,
		port, results, network.Timeout(timeout), resolver_allocation_complete,
	)
	if fixture.Resolver.Completion.Error != nil {
		t.Fatal(fixture.Resolver.Completion.Error)
	}
}

func resolver_state_boundaries(t *testing.T, resolve bool) {
	t.Helper()
	for boundary := uint16(1); boundary <= 2; boundary++ {
		resolver_state_boundary(t, boundary, resolve)
	}
	resolver_state_boundary(t, uint16(network.TRANSACTION_IDENTIFIER_MAXIMUM), resolve)
}

func resolver_state_boundary(t *testing.T, boundary uint16, resolve bool) {
	t.Helper()
	var fixture resolver_fixture
	resolver_fixture_init(&fixture, nbio.FAMILY_IPV4)
	results := address_storage(int(boundary))
	if boundary == uint16(network.TRANSACTION_IDENTIFIER_MAXIMUM) {
		results = address_storage(network.DNS_ADDRESS_COUNT_MAXIMUM)
	}
	resolver_state_seed(&fixture.Resolver, results, boundary)
	died := did_die(func() {
		if resolve {
			network.Resolve(
				&fixture.Resolver, &fixture.Resolver.Completion, "_",
				network.RECORD_TYPE_A, network.Port(boundary), fixture.Addresses[:],
				network.Timeout(time.SECOND), resolver_allocation_complete,
			)
			return
		}
		network.Resolver_Init(
			&fixture.Resolver, fake_dns_loop(&fixture.DNS),
			time.Virtual_Clock_To_Clock(&fixture.Virtual),
			prng.Chacha_To_Source(&fixture.Generator), &fixture.Workspace,
			resolver_configuration(nbio.FAMILY_IPV4),
		)
	})
	if !died {
		t.Fatal("Resolver accepted active retained state")
	}
}

func resolver_state_seed(
	resolver *network.Resolver, results []nbio.Address, boundary uint16,
) {
	byte_boundary := uint8(boundary)
	resolver.Results = network.Resolver_Results(results)
	resolver.Name = network.Resolver_Name(string(make([]byte, min(
		int(boundary), network.DNS_NAME_TEXT_BYTES_MAXIMUM,
	))))
	resolver.Deadline = network.Resolver_Deadline(boundary)
	resolver.Stage = network.Resolver_Stage(byte_boundary)
	resolver.Record_Type = network.Resolver_Record_Type(network.RECORD_TYPE_A)
	resolver.Port = network.Resolver_Port(boundary)
	resolver.Transaction = network.Transaction_Identifier(boundary)
	resolver.Query_Bytes = network.Query_Byte_Count(min(int(boundary),
		network.DNS_QUERY_BYTES_MAXIMUM))
	resolver.Transfer_Bytes = network.Transfer_Byte_Count(boundary)
	resolver.Response_Bytes = network.Response_Byte_Count(boundary)
	resolver.Result_Count = network.Result_Count(min(int(boundary),
		network.DNS_ADDRESS_COUNT_MAXIMUM))
	resolver.Flags = network.Resolver_Flags(byte_boundary)
	if boundary == uint16(network.TRANSACTION_IDENTIFIER_MAXIMUM) {
		resolver.Deadline = network.Resolver_Deadline(time.MONOTONIC_MOMENT_MAXIMUM)
		resolver.Stage = network.RESOLVER_STAGE_CLOSE_TCP
		resolver.Record_Type = network.Resolver_Record_Type(network.RECORD_TYPE_AAAA)
		resolver.Query_Bytes = network.QUERY_BYTE_COUNT_MAXIMUM
		resolver.Transfer_Bytes = network.TRANSFER_BYTE_COUNT_MAXIMUM
		resolver.Response_Bytes = network.RESPONSE_BYTE_COUNT_MAXIMUM
		resolver.Result_Count = network.RESULT_COUNT_MAXIMUM
		resolver.Flags = network.RESOLVER_FLAGS_ALL
	}
}

func resolver_dependency_boundaries(t *testing.T) {
	t.Helper()
	var fixture resolver_fixture
	resolver_fixture_init(&fixture, nbio.FAMILY_IPV6)
	died := did_die(func() {
		var resolver network.Resolver
		network.Resolver_Init(
			&resolver, fake_dns_loop(&fixture.DNS),
			time.Virtual_Clock_To_Clock(&fixture.Virtual),
			prng.Chacha_To_Source(&fixture.Generator), nil,
			resolver_configuration(nbio.FAMILY_IPV6),
		)
	})
	if !died {
		t.Fatal("Resolver_Init accepted missing workspace")
	}
	network.Resolver_Init(
		&fixture.Resolver, fake_dns_loop(&fixture.DNS),
		time.Virtual_Clock_To_Clock(&fixture.Virtual),
		prng.Chacha_To_Source(&fixture.Generator), &fixture.Workspace,
		resolver_configuration(nbio.FAMILY_IPV6),
	)
	network.Resolve(
		&fixture.Resolver, &fixture.Resolver.Completion, "_",
		network.RECORD_TYPE_AAAA, 0, fixture.Addresses[:],
		network.Timeout(time.SECOND), resolver_allocation_complete,
	)
	var zero network.Resolver
	died = did_die(func() {
		network.Resolve(
			&zero, &zero.Completion, "_", network.RECORD_TYPE_A, 0,
			fixture.Addresses[:], network.Timeout(time.SECOND),
			resolver_allocation_complete,
		)
	})
	if !died {
		t.Fatal("Resolve accepted uninitialized dependencies")
	}
}

func resolver_control_flow_boundaries(t *testing.T) {
	t.Helper()
	for _, fault := range [...]fake_dns_fault{
		FAKE_DNS_FAULT_ACTIVE_IDLE,
		FAKE_DNS_FAULT_COMPLETION_STAGE,
	} {
		var fixture resolver_fixture
		resolver_fixture_init(&fixture, nbio.FAMILY_IPV4)
		fixture.DNS.Fault = fault
		died := did_die(func() {
			network.Resolve(
				&fixture.Resolver, &fixture.Resolver.Completion, "example.com",
				network.RECORD_TYPE_A, network.PORT_MINIMUM, fixture.Addresses,
				network.Timeout(time.SECOND), resolver_allocation_complete,
			)
		})
		if !died {
			t.Fatalf("Resolver accepted control-flow fault %d", fault)
		}
	}
}

func resolver_fixture_init(fixture *resolver_fixture, family nbio.Address_Family) {
	resolver_fixture_storage_init(fixture)
	fixture.Virtual.Resolution = time.NANOSECOND
	resolver_generator_init(&fixture.Generator, 1)
	network.Resolver_Init(
		&fixture.Resolver, fake_dns_loop(&fixture.DNS),
		time.Virtual_Clock_To_Clock(&fixture.Virtual),
		prng.Chacha_To_Source(&fixture.Generator), &fixture.Workspace,
		resolver_configuration(family),
	)
}

// Enforcement panics at callsite, so recovery observes failure without changing global state.
func did_die(action func()) (died bool) {
	defer func() { died = recover() != nil }()
	action()
	return died
}
