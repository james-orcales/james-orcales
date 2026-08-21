// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by BSD-style license in Go source tree LICENSE file.

package network_test

import (
	"testing"

	"local/james-orcales/shared/crypto/prng"
	"local/james-orcales/shared/net"
	"local/james-orcales/shared/sim/aver/default"
	"local/james-orcales/shared/sim/nbio"
	"local/james-orcales/shared/sim/time"
)

// TestMain registers resolver invariant roots before all tests.
func TestMain(m *testing.M) {
	aver.Run_Test_Main(m)
}

// Test_Standard_Library_Resolver_Injected_IO ports A and AAAA parts of net.TestResolverDialFunc.
// SRV has no shared/net API. Two results prove injected transport preserves complete answer set.
func Test_Standard_Library_Resolver_Injected_IO(t *testing.T) {
	for _, record_type := range [...]network.Record_Type{
		network.RECORD_TYPE_A,
		network.RECORD_TYPE_AAAA,
	} {
		standard_library_resolve_two(t, record_type)
	}
}

func standard_library_resolve_two(t *testing.T, record_type network.Record_Type) {
	t.Helper()
	var fixture resolver_fixture
	resolver_fixture_storage_init(&fixture)
	fixture.DNS.Response_Kind = FAKE_DNS_RESPONSE_TWO_ADDRESSES
	fixture.Virtual.Resolution = time.NANOSECOND
	resolver_generator_init(
		&fixture.Generator, byte(network.ADDRESS_STORAGE_COUNT_USABLE_MINIMUM),
	)
	network.Resolver_Init(
		&fixture.Resolver, fake_dns_loop(&fixture.DNS),
		time.Virtual_Clock_To_Clock(&fixture.Virtual),
		prng.Chacha_To_Source(&fixture.Generator), &fixture.Workspace,
		resolver_configuration(nbio.FAMILY_IPV4),
	)
	network.Resolve(
		&fixture.Resolver, &fixture.Resolver.Completion, "example.com", record_type,
		network.PORT_MINIMUM, fixture.Addresses[:], network.Timeout(time.SECOND),
		resolver_allocation_complete,
	)
	if fixture.Resolver.Completion.Error != nil {
		t.Fatal(fixture.Resolver.Completion.Error)
	}
	if fixture.Resolver.Completion.Data != RESOLVER_RESULT_CAPACITY {
		t.Fatalf("address count = %d", fixture.Resolver.Completion.Data)
	}
	address_bytes := nbio.IPV4_ADDRESS_BYTES
	if record_type == network.RECORD_TYPE_AAAA {
		address_bytes = nbio.IPV6_ADDRESS_BYTES
	}
	first_index := network.ADDRESS_STORAGE_COUNT_MINIMUM
	second_index := first_index + network.ADDRESS_STORAGE_COUNT_USABLE_MINIMUM
	last_octet_index := address_bytes - network.ADDRESS_STORAGE_COUNT_USABLE_MINIMUM
	if fixture.Addresses[first_index].IP[last_octet_index] != FAKE_DNS_ADDRESS_LAST_OCTET {
		t.Fatalf("first address = %v", fixture.Addresses[first_index].IP[:address_bytes])
	}
	if fixture.Addresses[second_index].IP[last_octet_index] !=
		FAKE_DNS_ADDRESS_NEXT_LAST_OCTET {
		t.Fatalf("second address = %v", fixture.Addresses[second_index].IP[:address_bytes])
	}
}

// Test_Standard_Library_DNS_Transport_No_Fallback_On_TCP ports net.TestDNSTransportNoFallbackOnTCP.
// TCP truncation cannot fall back again, thus complete answer still retires current exchange.
func Test_Standard_Library_DNS_Transport_No_Fallback_On_TCP(t *testing.T) {
	var fixture resolver_fixture
	standard_library_tcp_truncated_init(&fixture)
	standard_library_tcp_truncated_resolve(&fixture)
	if fixture.Resolver.Completion.Error != nil {
		t.Fatal(fixture.Resolver.Completion.Error)
	}
	if fixture.Resolver.Completion.Data != network.ADDRESS_STORAGE_COUNT_USABLE_MINIMUM {
		t.Fatalf("address count = %d", fixture.Resolver.Completion.Data)
	}
	if fixture.DNS.Socket_Count != TCP_FALLBACK_SOCKET_COUNT {
		t.Fatalf("socket count = %d", fixture.DNS.Socket_Count)
	}

	var allocation_fixture resolver_fixture
	standard_library_tcp_truncated_init(&allocation_fixture)
	allocations := testing.AllocsPerRun(RESOLVER_ALLOCATION_RUN_COUNT, func() {
		standard_library_tcp_truncated_resolve(&allocation_fixture)
	})
	if allocations != 0 {
		t.Fatalf("TCP truncated response allocated %.1f times", allocations)
	}
}

func standard_library_tcp_truncated_init(fixture *resolver_fixture) {
	resolver_fixture_storage_init(fixture)
	fixture.DNS.Mode = FAKE_DNS_TCP
	fixture.DNS.TCP_Response_Truncated = true
	fixture.Virtual.Resolution = time.NANOSECOND
	resolver_generator_init(
		&fixture.Generator, byte(network.ADDRESS_STORAGE_COUNT_USABLE_MINIMUM),
	)
	network.Resolver_Init(
		&fixture.Resolver, fake_dns_loop(&fixture.DNS),
		time.Virtual_Clock_To_Clock(&fixture.Virtual),
		prng.Chacha_To_Source(&fixture.Generator), &fixture.Workspace,
		resolver_configuration(nbio.FAMILY_IPV4),
	)
}

func standard_library_tcp_truncated_resolve(fixture *resolver_fixture) {
	network.Resolve(
		&fixture.Resolver, &fixture.Resolver.Completion, "example.com",
		network.RECORD_TYPE_A, network.PORT_MINIMUM, fixture.Addresses[:],
		network.Timeout(time.SECOND), resolver_allocation_complete,
	)
}
