// Package network resolves validated host names through injected bounded network IO.
package network

import (
	"errors"
	"unsafe"

	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/crypto/prng"
	"local/james-orcales/shared/simulation/aver/default"
	"local/james-orcales/shared/simulation/nbio"
	"local/james-orcales/shared/simulation/time"
)

// DNS_LABEL_SIZE_BITS is width available to one ordinary DNS label size.
const DNS_LABEL_SIZE_BITS = 6

// DNS_LABEL_BYTES_MINIMUM is shortest nonempty host label.
const DNS_LABEL_BYTES_MINIMUM = 1

// DNS_LABEL_BYTES_MAXIMUM is largest label encoded without compression marker bits.
const DNS_LABEL_BYTES_MAXIMUM = (1 << DNS_LABEL_SIZE_BITS) - 1

// DNS_NAME_SIZE_BITS is width of DNS message and expanded-name size domains.
const DNS_NAME_SIZE_BITS = 8

// DNS_NAME_WIRE_BYTES_MAXIMUM is RFC DNS expanded-name ceiling, root octet included.
const DNS_NAME_WIRE_BYTES_MAXIMUM = (1 << DNS_NAME_SIZE_BITS) - 1

// DNS_NAME_ROOT_BYTES is terminating zero-length root label.
const DNS_NAME_ROOT_BYTES = 1

// DNS_NAME_TEXT_BYTES_MINIMUM is shortest host name.
const DNS_NAME_TEXT_BYTES_MINIMUM = DNS_LABEL_BYTES_MINIMUM

// DNS_NAME_TEXT_BYTES_INVALID is empty validation result.
const DNS_NAME_TEXT_BYTES_INVALID = 0

// DNS_NAME_TEXT_BYTES_MAXIMUM omits encoded root octet from presentation.
const DNS_NAME_TEXT_BYTES_MAXIMUM = DNS_NAME_WIRE_BYTES_MAXIMUM - DNS_NAME_ROOT_BYTES

// DNS_NAME_TEXT_BYTES_UNVALIDATED_MINIMUM admits empty hostile text.
const DNS_NAME_TEXT_BYTES_UNVALIDATED_MINIMUM = DNS_NAME_TEXT_BYTES_INVALID

// DNS_NAME_TEXT_BYTES_UNVALIDATED_MAXIMUM admits first rejected size.
const DNS_NAME_TEXT_BYTES_UNVALIDATED_MAXIMUM = DNS_NAME_TEXT_BYTES_MAXIMUM + 1

// Error_Name_Invalid reports text outside bounded ASCII host-name grammar.
var Error_Name_Invalid = errors.New("net: invalid host name")

// Name_Unvalidated is untrusted presentation-form host text.
type Name_Unvalidated string

// Name_Unvalidated_Invariants bounds validation work itself.
func Name_Unvalidated_Invariants(value Name_Unvalidated, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), DNS_NAME_TEXT_BYTES_UNVALIDATED_MINIMUM,
			DNS_NAME_TEXT_BYTES_UNVALIDATED_MAXIMUM,
		).
		Ensure()
}

// Name is validated presentation-form host text.
type Name string

// Name_Invariants keeps validated text inside DNS host grammar.
func Name_Invariants(name Name, namespace aver.Namespace) {
	aver.Tree(name, namespace).
		Range_Int(
			len(name), DNS_NAME_TEXT_BYTES_INVALID, DNS_NAME_TEXT_BYTES_MAXIMUM,
		).
		Ensure()
}

// Name_Validate separates malicious text from wire-safe host name.
func Name_Validate(unvalidated Name_Unvalidated) (name Name, err error) {
	defer func() { Name_Invariants(name, "Name_Validate.name") }()
	Name_Unvalidated_Invariants(unvalidated, "Name_Validate.unvalidated")
	if !bool(name_is_valid(unvalidated)) {
		return "", Error_Name_Invalid
	}
	return Name(unvalidated), nil
}

func name_is_valid(text Name_Unvalidated) (valid bytes.Boolean) {
	defer func() { bytes.Boolean_Invariants(valid, "name_is_valid.valid") }()
	Name_Unvalidated_Invariants(text, "name_is_valid.text")
	if len(text) < DNS_NAME_TEXT_BYTES_MINIMUM {
		return false
	}
	if len(text) > DNS_NAME_TEXT_BYTES_MAXIMUM {
		return false
	}
	end_count := len(text)
	if text[end_count-1] == '.' {
		end_count--
	}
	if end_count == 0 {
		return false
	}
	label_start := 0
	wire_bytes := DNS_NAME_ROOT_BYTES
	for index := 0; index <= end_count; index++ {
		if index < end_count {
			if text[index] != '.' {
				octet := text[index]
				if octet >= 'a' {
					if octet <= 'z' {
						continue
					}
				}
				if octet >= 'A' {
					if octet <= 'Z' {
						continue
					}
				}
				if octet >= '0' {
					if octet <= '9' {
						continue
					}
				}
				if octet == '-' {
					continue
				}
				return false
			}
		}
		label_bytes := index - label_start
		if label_bytes < DNS_LABEL_BYTES_MINIMUM {
			return false
		}
		if label_bytes > DNS_LABEL_BYTES_MAXIMUM {
			return false
		}
		if text[label_start] == '-' {
			return false
		}
		if text[index-1] == '-' {
			return false
		}
		wire_bytes += DNS_NAME_ROOT_BYTES + label_bytes
		if wire_bytes > DNS_NAME_WIRE_BYTES_MAXIMUM {
			return false
		}
		label_start = index + 1
	}
	return true
}

// DNS_MESSAGE_SIZE_BITS is width of TCP DNS message-size prefix.
const DNS_MESSAGE_SIZE_BITS = 16

// DNS_MESSAGE_BYTES_MAXIMUM is largest message named by unsigned 16-bit TCP prefix.
const DNS_MESSAGE_BYTES_MAXIMUM = (1 << DNS_MESSAGE_SIZE_BITS) - 1

// DNS_HEADER_WORD_COUNT is count of 16-bit fields in fixed DNS header.
const DNS_HEADER_WORD_COUNT = 6

// DNS_WORD_BYTES follows message-size field width.
const DNS_WORD_BYTES = DNS_MESSAGE_SIZE_BITS / DNS_NAME_SIZE_BITS

// DNS_LABEL_SIZE_BYTES is one octet before label payload.
const DNS_LABEL_SIZE_BYTES = DNS_NAME_ROOT_BYTES

// DNS_HEADER_BYTES is fixed DNS header size.
const DNS_HEADER_BYTES = DNS_HEADER_WORD_COUNT * DNS_WORD_BYTES

// DNS_TRANSACTION_BYTES is transaction identifier width.
const DNS_TRANSACTION_BYTES = DNS_WORD_BYTES

// DNS_FLAGS_OFFSET follows transaction identifier.
const DNS_FLAGS_OFFSET = DNS_TRANSACTION_BYTES

// DNS_QUESTION_COUNT_OFFSET follows flags.
const DNS_QUESTION_COUNT_OFFSET = DNS_FLAGS_OFFSET + DNS_WORD_BYTES

// DNS_ANSWER_COUNT_OFFSET follows question count.
const DNS_ANSWER_COUNT_OFFSET = DNS_QUESTION_COUNT_OFFSET + DNS_WORD_BYTES

// DNS_AUTHORITY_COUNT_OFFSET follows answer count.
const DNS_AUTHORITY_COUNT_OFFSET = DNS_ANSWER_COUNT_OFFSET + DNS_WORD_BYTES

// DNS_ADDITIONAL_COUNT_OFFSET follows authority count.
const DNS_ADDITIONAL_COUNT_OFFSET = DNS_AUTHORITY_COUNT_OFFSET + DNS_WORD_BYTES

// DNS_RECORD_TYPE_BYTES is resource type width.
const DNS_RECORD_TYPE_BYTES = DNS_WORD_BYTES

// DNS_CLASS_BYTES is resource class width.
const DNS_CLASS_BYTES = DNS_WORD_BYTES

// DNS_QUESTION_FIXED_BYTES follows QNAME with type and class.
const DNS_QUESTION_FIXED_BYTES = DNS_RECORD_TYPE_BYTES + DNS_CLASS_BYTES

// DNS_TTL_BYTES is resource TTL width.
const DNS_TTL_BYTES = 2 * DNS_WORD_BYTES

// DNS_RESOURCE_SIZE_BYTES is resource-data size width.
const DNS_RESOURCE_SIZE_BYTES = DNS_WORD_BYTES

// DNS_RESOURCE_FIXED_BYTES follows owner name with type, class, TTL, and data size.
const DNS_RESOURCE_FIXED_BYTES = DNS_RECORD_TYPE_BYTES + DNS_CLASS_BYTES +
	DNS_TTL_BYTES + DNS_RESOURCE_SIZE_BYTES

// DNS_COMPRESSION_POINTER_BYTES is encoded owner pointer width.
const DNS_COMPRESSION_POINTER_BYTES = DNS_WORD_BYTES

// DNS_QUERY_BYTES_MAXIMUM holds fixed header and largest question.
const DNS_QUERY_BYTES_MAXIMUM = DNS_HEADER_BYTES + DNS_NAME_WIRE_BYTES_MAXIMUM +
	DNS_QUESTION_FIXED_BYTES

// DNS_TCP_SIZE_BYTES is TCP message-size prefix width.
const DNS_TCP_SIZE_BYTES = DNS_WORD_BYTES

// DNS_TCP_QUERY_BYTES_MAXIMUM includes TCP prefix before largest query.
const DNS_TCP_QUERY_BYTES_MAXIMUM = DNS_TCP_SIZE_BYTES + DNS_QUERY_BYTES_MAXIMUM

// DNS_UDP_MESSAGE_SIZE_BITS is classic DNS UDP message ceiling exponent.
const DNS_UDP_MESSAGE_SIZE_BITS = 9

// DNS_UDP_MESSAGE_BYTES_MAXIMUM is classic DNS UDP response ceiling.
const DNS_UDP_MESSAGE_BYTES_MAXIMUM = 1 << DNS_UDP_MESSAGE_SIZE_BITS

// DNS_IPV4_RESOURCE_BYTES_MINIMUM uses compressed owner and smallest address data.
const DNS_IPV4_RESOURCE_BYTES_MINIMUM = DNS_COMPRESSION_POINTER_BYTES +
	DNS_RESOURCE_FIXED_BYTES + nbio.IPV4_ADDRESS_BYTES

// DNS_NAME_WIRE_BYTES_MINIMUM holds one shortest label and root.
const DNS_NAME_WIRE_BYTES_MINIMUM = DNS_LABEL_SIZE_BYTES +
	DNS_LABEL_BYTES_MINIMUM + DNS_NAME_ROOT_BYTES

// DNS_QUERY_BYTES_MINIMUM holds header and shortest valid question.
const DNS_QUERY_BYTES_MINIMUM = DNS_HEADER_BYTES + DNS_NAME_WIRE_BYTES_MINIMUM +
	DNS_QUESTION_FIXED_BYTES

// DNS_ADDRESS_COUNT_MAXIMUM is largest possible address-record count in one message.
const DNS_ADDRESS_COUNT_MAXIMUM = (DNS_MESSAGE_BYTES_MAXIMUM - DNS_QUERY_BYTES_MINIMUM) /
	DNS_IPV4_RESOURCE_BYTES_MINIMUM

// DNS_CNAME_RESOURCE_BYTES_MINIMUM uses compressed owner and root target.
const DNS_CNAME_RESOURCE_BYTES_MINIMUM = DNS_COMPRESSION_POINTER_BYTES +
	DNS_RESOURCE_FIXED_BYTES + DNS_NAME_ROOT_BYTES

// DNS_CNAME_HOPS_MAXIMUM follows every CNAME that can fit after one valid question.
const DNS_CNAME_HOPS_MAXIMUM = (DNS_MESSAGE_BYTES_MAXIMUM - DNS_QUERY_BYTES_MINIMUM) /
	DNS_CNAME_RESOURCE_BYTES_MINIMUM

// DNS_QUESTION_POINTER targets QNAME at fixed header boundary.
const DNS_QUESTION_POINTER = DNS_COMPRESSION_POINTER_TAG | DNS_HEADER_BYTES

// DNS_CLASS_IN identifies Internet records.
const DNS_CLASS_IN = 1

// DNS_RESPONSE_FLAGS_SUCCESS is recursive successful response with recursion available.
const DNS_RESPONSE_FLAGS_SUCCESS = 1<<15 | 1<<8 | 1<<7

// DNS_RESPONSE_FLAGS_TRUNCATED adds truncated bit to successful response flags.
const DNS_RESPONSE_FLAGS_TRUNCATED = DNS_RESPONSE_FLAGS_SUCCESS | 1<<9

// DNS_QUERY_FLAGS_RECURSION_DESIRED requests recursive service.
const DNS_QUERY_FLAGS_RECURSION_DESIRED = 1 << 8

// DNS_RESPONSE_BIT distinguishes response from query.
const DNS_RESPONSE_BIT = 1 << 15

// DNS_TRUNCATED_BIT reports incomplete UDP response.
const DNS_TRUNCATED_BIT = 1 << 9

// DNS_RESPONSE_CODE_MASK selects low response-code bits.
const DNS_RESPONSE_CODE_MASK = 1<<4 - 1

// DNS_OPCODE_MASK selects response operation code.
const DNS_OPCODE_MASK = DNS_RESPONSE_CODE_MASK << 11

// DNS_COMPRESSION_TAG_MASK selects two compression marker bits.
const DNS_COMPRESSION_TAG_MASK = 3 << DNS_LABEL_SIZE_BITS

// DNS_COMPRESSION_TAG identifies compressed-name pointer.
const DNS_COMPRESSION_TAG = DNS_COMPRESSION_TAG_MASK

// DNS_COMPRESSION_POINTER_TAG places marker bits in 16-bit pointer field.
const DNS_COMPRESSION_POINTER_TAG = DNS_COMPRESSION_TAG << DNS_NAME_SIZE_BITS

// DNS_COMPRESSION_OFFSET_MASK removes pointer marker bits.
const DNS_COMPRESSION_OFFSET_MASK = (1 << 14) - 1

// DNS_NAME_COMPRESSION_HOPS_MAXIMUM bounds malicious pointer chains by expanded-name budget.
const DNS_NAME_COMPRESSION_HOPS_MAXIMUM = DNS_NAME_WIRE_BYTES_MAXIMUM /
	DNS_COMPRESSION_POINTER_BYTES

// DNS_NAME_DECODE_STEPS_MAXIMUM holds every expanded byte and compression hop.
const DNS_NAME_DECODE_STEPS_MAXIMUM = DNS_NAME_WIRE_BYTES_MAXIMUM +
	DNS_NAME_COMPRESSION_HOPS_MAXIMUM

// DNS_RESPONSE_CODE_SUCCESS reports successful server processing.
const DNS_RESPONSE_CODE_SUCCESS = 0

// DNS_RESPONSE_CODE_SERVER_FAILURE reports server failure.
const DNS_RESPONSE_CODE_SERVER_FAILURE = 2

// DNS_RESPONSE_CODE_NAME_ERROR reports nonexistent domain.
const DNS_RESPONSE_CODE_NAME_ERROR = 3

// DNS_SOCKET_INVALID marks no descriptor owned by Resolver.
const DNS_SOCKET_INVALID nbio.File = -1

// ADDRESS_STORAGE_COUNT_MINIMUM permits idle Resolver without retained results.
const ADDRESS_STORAGE_COUNT_MINIMUM = 0

// ADDRESS_STORAGE_COUNT_USABLE_MINIMUM is first caller result slot.
const ADDRESS_STORAGE_COUNT_USABLE_MINIMUM = ADDRESS_STORAGE_COUNT_MINIMUM + 1

// RECORD_TYPE_A is IPv4 address resource type.
const RECORD_TYPE_A Record_Type = 1

// RECORD_TYPE_AAAA is IPv6 address resource type.
const RECORD_TYPE_AAAA Record_Type = 28

// DNS_RECORD_TYPE_CNAME aliases one owner to another owner.
const DNS_RECORD_TYPE_CNAME = 5

// RECORD_TYPE_INVALID is idle Resolver value.
const RECORD_TYPE_INVALID Record_Type = 0

// PORT_MINIMUM is first transport port.
const PORT_MINIMUM Port = 0

// PORT_MAXIMUM closes unsigned transport-port field.
const PORT_MAXIMUM Port = (1 << DNS_MESSAGE_SIZE_BITS) - 1

// Error_Response_Malformed reports invalid DNS wire structure.
var Error_Response_Malformed = errors.New("net: malformed DNS response")

// Error_Response_Mismatch reports response for another transaction or question.
var Error_Response_Mismatch = errors.New("net: mismatched DNS response")

// Error_Name_Not_Found reports NXDOMAIN.
var Error_Name_Not_Found = errors.New("net: DNS name not found")

// Error_Server_Failure reports DNS server failure.
var Error_Server_Failure = errors.New("net: DNS server failure")

// Error_No_Address reports successful response without requested address.
var Error_No_Address = errors.New("net: DNS response has no requested address")

// Error_Result_Too_Small reports caller storage cannot hold every unique address.
var Error_Result_Too_Small = errors.New("net: address storage too small")

// Error_Transfer_No_Progress reports successful transport retirement with zero bytes.
var Error_Transfer_No_Progress = errors.New("net: DNS transfer made no progress")

// Error_Transfer_Count reports transport byte count outside submitted buffer.
var Error_Transfer_Count = errors.New("net: invalid DNS transfer count")

// Record_Type selects address family requested from DNS.
type Record_Type uint16

// Record_Type_Invariants admits only supported address resource types.
func Record_Type_Invariants(value Record_Type, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint16(
			uint16(value), uint16(RECORD_TYPE_A), uint16(RECORD_TYPE_AAAA),
		).
		Ensure()
}

// Address_Storage is caller-owned result capacity.
type Address_Storage []nbio.Address

// Address_Storage_Invariants keeps one result slot through maximum possible record count.
func Address_Storage_Invariants(value Address_Storage, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), ADDRESS_STORAGE_COUNT_MINIMUM, DNS_ADDRESS_COUNT_MAXIMUM).
		Ensure()
}

// Resolver_Configuration gives one upstream and explicit socket limits.
type Resolver_Configuration struct {
	// Server is injected IP endpoint. Host names cannot cross socket boundary.
	Server nbio.Address
	// UDP bounds first exchange socket.
	UDP nbio.UDP_Options
	// TCP bounds truncated-response fallback socket.
	TCP nbio.TCP_Options
}

// Resolver_Configuration_Invariants admits explicit supported address family.
func Resolver_Configuration_Invariants(
	value Resolver_Configuration, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_Int(
			int(value.Server.Family), int(nbio.FAMILY_IPV4), int(nbio.FAMILY_IPV6),
		).
		Ensure()
}

// Resolver_Workspace is complete caller-owned DNS message storage.
type Resolver_Workspace struct {
	// Query includes TCP prefix before maximum DNS question.
	Query [DNS_TCP_QUERY_BYTES_MAXIMUM]byte
	// Response holds any message named by TCP size prefix.
	Response [DNS_MESSAGE_BYTES_MAXIMUM]byte
	// Canonical_Name stores lowercase expanded query owner.
	Canonical_Name [DNS_NAME_WIRE_BYTES_MAXIMUM]byte
	// Decoded_Name stores one expanded response owner.
	Decoded_Name [DNS_NAME_WIRE_BYTES_MAXIMUM]byte
	// Alias_Name preserves one decoded target while answer scan continues.
	Alias_Name [DNS_NAME_WIRE_BYTES_MAXIMUM]byte
}

// Resolver_Workspace_Invariants fixes both protocol arrays to formulas.
func Resolver_Workspace_Invariants(value *Resolver_Workspace, _ aver.Namespace) {
	aver.Always(len(value.Query) == DNS_TCP_QUERY_BYTES_MAXIMUM,
		"Resolver query workspace has protocol capacity.")
	aver.Always(len(value.Response) == DNS_MESSAGE_BYTES_MAXIMUM,
		"Resolver response workspace has protocol capacity.")
	aver.Always(len(value.Alias_Name) == DNS_NAME_WIRE_BYTES_MAXIMUM,
		"Resolver alias workspace has expanded-name capacity.")
}

// Resolver_Workspace_Pointer keeps large scratch caller-owned and optional before init.
type Resolver_Workspace_Pointer *Resolver_Workspace

// Resolver_Workspace_Pointer_Invariants covers zero state and initialized workspace.
func Resolver_Workspace_Pointer_Invariants(
	value Resolver_Workspace_Pointer, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Sometimes(value != nil, "Resolver workspace is bound.").
		Ensure()
}

// Resolver_Stage is one bounded protocol state.
type Resolver_Stage uint8

// Resolver_Stage_Invariants keeps state machine inside declared stages.
func Resolver_Stage_Invariants(value Resolver_Stage, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(
			uint8(value), uint8(RESOLVER_STAGE_IDLE), uint8(RESOLVER_STAGE_CLOSE_TCP),
		).
		Ensure()
}

// RESOLVER_STAGE_IDLE holds no active operation.
const RESOLVER_STAGE_IDLE Resolver_Stage = 0

// RESOLVER_STAGE_OPEN_UDP creates first transport.
const RESOLVER_STAGE_OPEN_UDP Resolver_Stage = RESOLVER_STAGE_IDLE + 1

// RESOLVER_STAGE_CONNECT_UDP connects datagram peer.
const RESOLVER_STAGE_CONNECT_UDP Resolver_Stage = RESOLVER_STAGE_OPEN_UDP + 1

// RESOLVER_STAGE_SEND_UDP submits one complete datagram.
const RESOLVER_STAGE_SEND_UDP Resolver_Stage = RESOLVER_STAGE_CONNECT_UDP + 1

// RESOLVER_STAGE_RECEIVE_UDP waits for one datagram.
const RESOLVER_STAGE_RECEIVE_UDP Resolver_Stage = RESOLVER_STAGE_SEND_UDP + 1

// RESOLVER_STAGE_CLOSE_UDP releases datagram descriptor.
const RESOLVER_STAGE_CLOSE_UDP Resolver_Stage = RESOLVER_STAGE_RECEIVE_UDP + 1

// RESOLVER_STAGE_OPEN_TCP creates fallback transport.
const RESOLVER_STAGE_OPEN_TCP Resolver_Stage = RESOLVER_STAGE_CLOSE_UDP + 1

// RESOLVER_STAGE_CONNECT_TCP connects stream peer.
const RESOLVER_STAGE_CONNECT_TCP Resolver_Stage = RESOLVER_STAGE_OPEN_TCP + 1

// RESOLVER_STAGE_SEND_TCP submits framed query.
const RESOLVER_STAGE_SEND_TCP Resolver_Stage = RESOLVER_STAGE_CONNECT_TCP + 1

// RESOLVER_STAGE_RECEIVE_TCP_SIZE joins message prefix.
const RESOLVER_STAGE_RECEIVE_TCP_SIZE Resolver_Stage = RESOLVER_STAGE_SEND_TCP + 1

// RESOLVER_STAGE_RECEIVE_TCP_MESSAGE joins message body.
const RESOLVER_STAGE_RECEIVE_TCP_MESSAGE Resolver_Stage = RESOLVER_STAGE_RECEIVE_TCP_SIZE + 1

// RESOLVER_STAGE_CLOSE_TCP releases stream descriptor.
const RESOLVER_STAGE_CLOSE_TCP Resolver_Stage = RESOLVER_STAGE_RECEIVE_TCP_MESSAGE + 1

// Resolver_Entropy permits zero state before caller binds a source.
type Resolver_Entropy prng.Source

// Resolver_Entropy_Invariants covers uninitialized and bound state.
func Resolver_Entropy_Invariants(value Resolver_Entropy, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(value.State != nil, "Resolver entropy is bound.").
		Ensure()
}

// Resolver_Clock permits zero state before caller binds both readers.
type Resolver_Clock time.Clock

// Resolver_Clock_Invariants excludes partially bound clock vtable.
func Resolver_Clock_Invariants(value Resolver_Clock, namespace aver.Namespace) {
	aver.Always(
		(value.Now_Monotonic == nil) == (value.Now_Realtime == nil),
		"Resolver host binds both readers or neither reader.",
	)
	aver.Tree(value, namespace).
		Sometimes(value.Now_Monotonic != nil, "Resolver host is bound.").
		Ensure()
}

// Port is service port copied into resolved addresses.
type Port uint16

// Port_Invariants covers complete transport port domain.
func Port_Invariants(value Port, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value), uint16(PORT_MINIMUM), uint16(PORT_MAXIMUM)).
		Ensure()
}

// Transaction_Identifier distinguishes one DNS exchange.
type Transaction_Identifier uint16

// Transaction_Identifier_Invariants covers complete DNS identifier field.
func Transaction_Identifier_Invariants(
	value Transaction_Identifier, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(TRANSACTION_IDENTIFIER_MINIMUM),
			uint16(TRANSACTION_IDENTIFIER_MAXIMUM),
		).
		Ensure()
}

// TRANSACTION_IDENTIFIER_MINIMUM is first unsigned DNS identifier.
const TRANSACTION_IDENTIFIER_MINIMUM Transaction_Identifier = 0

// TRANSACTION_IDENTIFIER_MAXIMUM closes DNS identifier field.
const TRANSACTION_IDENTIFIER_MAXIMUM Transaction_Identifier = (1 << DNS_MESSAGE_SIZE_BITS) - 1

// Resolver_Flags packs state-machine decisions into one invariant subject.
type Resolver_Flags uint8

// Resolver_Flags_Invariants bounds every flag combination.
func Resolver_Flags_Invariants(value Resolver_Flags, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), uint8(RESOLVER_FLAGS_NONE), uint8(RESOLVER_FLAGS_ALL)).
		Ensure()
}

// RESOLVER_FLAGS_NONE marks idle scalar state.
const RESOLVER_FLAGS_NONE Resolver_Flags = 0

// RESOLVER_FLAG_ACTIVE protects retained callback and result storage.
const RESOLVER_FLAG_ACTIVE Resolver_Flags = 1 << 0

// RESOLVER_FLAG_WAIT_ACTIVE marks one nbio operation in flight.
const RESOLVER_FLAG_WAIT_ACTIVE Resolver_Flags = RESOLVER_FLAG_ACTIVE << 1

// RESOLVER_FLAG_SUBMISSION_ACTIVE marks backend procedure stack lifetime.
const RESOLVER_FLAG_SUBMISSION_ACTIVE Resolver_Flags = RESOLVER_FLAG_WAIT_ACTIVE << 1

// RESOLVER_FLAG_CONTINUE requests progress after inline retirement.
const RESOLVER_FLAG_CONTINUE Resolver_Flags = RESOLVER_FLAG_SUBMISSION_ACTIVE << 1

// RESOLVER_FLAG_TCP_FALLBACK records truncated UDP response.
const RESOLVER_FLAG_TCP_FALLBACK Resolver_Flags = RESOLVER_FLAG_CONTINUE << 1

// RESOLVER_FLAGS_ALL combines every independent flag.
const RESOLVER_FLAGS_ALL Resolver_Flags = RESOLVER_FLAG_ACTIVE |
	RESOLVER_FLAG_WAIT_ACTIVE | RESOLVER_FLAG_SUBMISSION_ACTIVE |
	RESOLVER_FLAG_CONTINUE | RESOLVER_FLAG_TCP_FALLBACK

// Resolver_Name retains valid or empty idle name without duplicating Name coverage chain.
type Resolver_Name Name

// Resolver_Name_Invariants bounds retained name storage.
func Resolver_Name_Invariants(value Resolver_Name, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), DNS_NAME_TEXT_BYTES_INVALID, DNS_NAME_TEXT_BYTES_MAXIMUM).
		Ensure()
}

// Resolver_Results retains caller storage or nil while idle.
type Resolver_Results Address_Storage

// Resolver_Results_Invariants bounds retained result capacity.
func Resolver_Results_Invariants(value Resolver_Results, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), ADDRESS_STORAGE_COUNT_MINIMUM, DNS_ADDRESS_COUNT_MAXIMUM).
		Ensure()
}

// Resolver_Deadline keeps idle zero and configured deadline in monotonic domain.
type Resolver_Deadline time.Monotonic_Moment

// Resolver_Deadline_Invariants bounds retained monotonic moment.
func Resolver_Deadline_Invariants(value Resolver_Deadline, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(
			int64(value), int64(time.MONOTONIC_MOMENT_MINIMUM),
			int64(time.MONOTONIC_MOMENT_MAXIMUM),
		).
		Ensure()
}

// Resolver_Record_Type keeps idle invalid value beside active validated type.
type Resolver_Record_Type Record_Type

// Resolver_Record_Type_Invariants bounds retained protocol number.
func Resolver_Record_Type_Invariants(
	value Resolver_Record_Type, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_3_Uint16(
			uint16(value), uint16(RECORD_TYPE_INVALID), uint16(RECORD_TYPE_A),
			uint16(RECORD_TYPE_AAAA),
		).
		Ensure()
}

// Resolver_Port keeps copied result port in protocol field.
type Resolver_Port Port

// Resolver_Port_Invariants bounds retained result port.
func Resolver_Port_Invariants(value Resolver_Port, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value), uint16(PORT_MINIMUM), uint16(PORT_MAXIMUM)).
		Ensure()
}

// Timeout is complete resolver-operation bound.
type Timeout time.Duration

// Timeout_Invariants requires positive duration fitting monotonic domain.
func Timeout_Invariants(value Timeout, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(
			int64(value), int64(time.NANOSECOND),
			int64(time.MONOTONIC_MOMENT_MAXIMUM),
		).
		Ensure()
}

// Resolver_Timeout is positive remaining operation span.
type Resolver_Timeout time.Duration

// Resolver_Timeout_Invariants bounds one submitted primitive.
func Resolver_Timeout_Invariants(value Resolver_Timeout, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(
			int64(value), int64(time.NANOSECOND),
			int64(time.MONOTONIC_MOMENT_MAXIMUM),
		).
		Ensure()
}

// Query_Byte_Count is zero while idle or encoded query size while active.
type Query_Byte_Count uint16

// Query_Byte_Count_Invariants bounds retained query size by query workspace.
func Query_Byte_Count_Invariants(value Query_Byte_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(QUERY_BYTE_COUNT_MINIMUM),
			uint16(QUERY_BYTE_COUNT_MAXIMUM),
		).
		Ensure()
}

// QUERY_BYTE_COUNT_MINIMUM is idle query size.
const QUERY_BYTE_COUNT_MINIMUM Query_Byte_Count = 0

// QUERY_BYTE_COUNT_MAXIMUM is largest encoded DNS query.
const QUERY_BYTE_COUNT_MAXIMUM Query_Byte_Count = DNS_QUERY_BYTES_MAXIMUM

// Transfer_Byte_Count is completed prefix of one TCP field.
type Transfer_Byte_Count uint16

// Transfer_Byte_Count_Invariants bounds progress by largest TCP message.
func Transfer_Byte_Count_Invariants(
	value Transfer_Byte_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(TRANSFER_BYTE_COUNT_MINIMUM),
			uint16(TRANSFER_BYTE_COUNT_MAXIMUM),
		).
		Ensure()
}

// TRANSFER_BYTE_COUNT_MINIMUM is first stream position.
const TRANSFER_BYTE_COUNT_MINIMUM Transfer_Byte_Count = 0

// TRANSFER_BYTE_COUNT_MAXIMUM is largest TCP DNS body position.
const TRANSFER_BYTE_COUNT_MAXIMUM Transfer_Byte_Count = DNS_MESSAGE_BYTES_MAXIMUM

// Response_Byte_Count is received DNS message size without TCP prefix.
type Response_Byte_Count uint16

// Response_Byte_Count_Invariants bounds retained response by caller workspace.
func Response_Byte_Count_Invariants(
	value Response_Byte_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(RESPONSE_BYTE_COUNT_MINIMUM),
			uint16(RESPONSE_BYTE_COUNT_MAXIMUM),
		).
		Ensure()
}

// RESPONSE_BYTE_COUNT_MINIMUM is idle response size.
const RESPONSE_BYTE_COUNT_MINIMUM Response_Byte_Count = 0

// RESPONSE_BYTE_COUNT_MAXIMUM is largest TCP DNS body.
const RESPONSE_BYTE_COUNT_MAXIMUM Response_Byte_Count = DNS_MESSAGE_BYTES_MAXIMUM

// Result_Count is unique address count written to caller storage.
type Result_Count uint16

// Result_Count_Invariants bounds retained result count by DNS message capacity.
func Result_Count_Invariants(value Result_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(RESULT_COUNT_MINIMUM), uint16(RESULT_COUNT_MAXIMUM),
		).
		Ensure()
}

// RESULT_COUNT_MINIMUM is idle result count.
const RESULT_COUNT_MINIMUM Result_Count = 0

// RESULT_COUNT_MAXIMUM is every address record fitting one DNS body.
const RESULT_COUNT_MAXIMUM Result_Count = DNS_ADDRESS_COUNT_MAXIMUM

// Resolver owns one active DNS exchange while every byte array stays in Workspace.
type Resolver struct {
	// Completion stays first so static callback recovers Resolver without allocation.
	Completion nbio.Completion
	// IO supplies socket operations and asynchronous close.
	IO nbio.IO
	// Clock bounds complete exchange instead of restarting timeout at each transfer.
	Clock Resolver_Clock
	// Entropy supplies unpredictable transaction identifiers.
	Entropy Resolver_Entropy
	// Workspace holds retained query and response bytes.
	Workspace Resolver_Workspace_Pointer
	// Configuration stays fixed across one exchange.
	Configuration Resolver_Configuration
	// Callback retires after owned descriptor closes.
	Callback nbio.Callback
	// Results borrows caller slots until Callback.
	Results Resolver_Results
	// Name borrows validated caller text until query construction finishes.
	Name Resolver_Name
	// Error survives descriptor teardown.
	Error error
	// Socket is current caller-owned descriptor.
	Socket nbio.File
	// Deadline is whole-exchange monotonic bound.
	Deadline Resolver_Deadline
	// Stage is current state-machine instruction.
	Stage Resolver_Stage
	// Record_Type selects A or AAAA parsing.
	Record_Type Resolver_Record_Type
	// Port is copied into every result address.
	Port Resolver_Port
	// Transaction identifies matching response.
	Transaction Transaction_Identifier
	// Query_Bytes is encoded DNS message size without TCP prefix.
	Query_Bytes Query_Byte_Count
	// Transfer_Bytes is completed portion of current stream field.
	Transfer_Bytes Transfer_Byte_Count
	// Response_Bytes is received DNS message size without TCP prefix.
	Response_Bytes Response_Byte_Count
	// Result_Count is unique address count written to caller storage.
	Result_Count Result_Count
	// Flags packs active, wait, inline-retirement, and fallback state.
	Flags Resolver_Flags
}

// Resolver_Invariants protects static completion recovery and injected ownership.
func Resolver_Invariants(value Resolver, namespace aver.Namespace) {
	aver.Always(
		unsafe.Pointer(&value) == unsafe.Pointer(&value.Completion),
		"Resolver completion stays first for static callback recovery.",
	)
	nbio.IO_Invariants(value.IO, namespace)
	Resolver_Entropy_Invariants(value.Entropy, namespace)
	Resolver_Clock_Invariants(value.Clock, namespace)
	Resolver_Workspace_Pointer_Invariants(value.Workspace, namespace)
	Resolver_Configuration_Invariants(value.Configuration, namespace)
	Resolver_Results_Invariants(value.Results, namespace)
	Resolver_Name_Invariants(value.Name, namespace)
	Resolver_Deadline_Invariants(value.Deadline, namespace)
	Resolver_Stage_Invariants(value.Stage, namespace)
	Resolver_Record_Type_Invariants(value.Record_Type, namespace)
	Resolver_Port_Invariants(value.Port, namespace)
	Transaction_Identifier_Invariants(value.Transaction, namespace)
	Query_Byte_Count_Invariants(value.Query_Bytes, namespace)
	Transfer_Byte_Count_Invariants(value.Transfer_Bytes, namespace)
	Response_Byte_Count_Invariants(value.Response_Bytes, namespace)
	Result_Count_Invariants(value.Result_Count, namespace)
	Resolver_Flags_Invariants(value.Flags, namespace)
	aver.Always(
		(value.Stage == RESOLVER_STAGE_IDLE) ==
			(value.Flags&RESOLVER_FLAG_ACTIVE == 0),
		"Resolver idle stage matches inactive ownership.",
	)
}

// Resolver_Init binds dependencies and caller workspace without opening socket.
func Resolver_Init(
	resolver *Resolver,
	loop nbio.IO, host time.Clock, entropy prng.Source,
	workspace Resolver_Workspace_Pointer, configuration Resolver_Configuration,
) {
	Resolver_Invariants(*resolver, "Resolver_Init.resolver")
	nbio.IO_Invariants(loop, "Resolver_Init.loop")
	time.Clock_Invariants(host, "Resolver_Init.host")
	prng.Source_Invariants(entropy, "Resolver_Init.entropy")
	Resolver_Workspace_Pointer_Invariants(workspace, "Resolver_Init.workspace")
	Resolver_Configuration_Invariants(configuration, "Resolver_Init.configuration")
	aver.Always(
		resolver.Flags == RESOLVER_FLAGS_NONE,
		"Resolver_Init owns inactive Resolver.",
	)
	aver.Always(workspace != nil, "Resolver_Init has caller-owned workspace.")
	Resolver_Workspace_Invariants(
		(*Resolver_Workspace)(workspace), "Resolver_Init.workspace_value",
	)
	aver.Always(configuration.Server.Port > 0, "Resolver server port is positive.")
	aver.Always(loop.Close_Procedure != nil, "Resolver has close procedure.")
	aver.Always(loop.Network.Socket_TCP_Procedure != nil,
		"Resolver has TCP socket procedure.")
	aver.Always(loop.Network.Socket_UDP_Procedure != nil,
		"Resolver has UDP socket procedure.")
	aver.Always(loop.Network.Connect_Procedure != nil,
		"Resolver has connect procedure.")
	aver.Always(loop.Network.Receive_Procedure != nil,
		"Resolver has receive procedure.")
	aver.Always(loop.Network.Send_Procedure != nil,
		"Resolver has send procedure.")
	*resolver = Resolver{
		IO: loop, Clock: Resolver_Clock(host), Entropy: Resolver_Entropy(entropy),
		Workspace: workspace, Configuration: configuration, Socket: DNS_SOCKET_INVALID,
		Stage: RESOLVER_STAGE_IDLE,
	}
}

// Resolve starts one bounded asynchronous address query.
func Resolve(
	resolver *Resolver, completion *nbio.Completion, name Name, record_type Record_Type,
	port Port, results Address_Storage, timeout Timeout, callback nbio.Callback,
) {
	Resolver_Invariants(*resolver, "Resolve.resolver")
	Name_Invariants(name, "Resolve.name")
	Record_Type_Invariants(record_type, "Resolve.record_type")
	Port_Invariants(port, "Resolve.port")
	Address_Storage_Invariants(results, "Resolve.results")
	Timeout_Invariants(timeout, "Resolve.timeout")
	time.Clock_Invariants(time.Clock(resolver.Clock), "Resolve.host")
	aver.Always(resolver.Entropy.State != nil, "Resolve has bound entropy.")
	aver.Always(resolver.Workspace != nil, "Resolve has bound workspace.")
	aver.Always(completion != nil, "Resolve has completion storage.")
	aver.Always(completion == &resolver.Completion,
		"Resolve submits Resolver-owned completion.")
	aver.Always(callback != nil, "Resolve has callback.")
	aver.Always(
		resolver.Flags&RESOLVER_FLAG_ACTIVE == 0, "Resolve owns idle Resolver.",
	)
	aver.Always(
		len(results) >= ADDRESS_STORAGE_COUNT_USABLE_MINIMUM,
		"Resolve has address result storage.",
	)
	if !bool(name_is_valid(Name_Unvalidated(name))) {
		completion.Data = 0
		completion.Error = Error_Name_Invalid
		callback(completion)
		return
	}
	now := time.Clock_Now_Monotonic(time.Clock(resolver.Clock))
	aver.Always(
		time.Monotonic_Moment(timeout) <= time.MONOTONIC_MOMENT_MAXIMUM-now,
		"Resolve deadline fits monotonic domain.",
	)
	resolver.Callback = callback
	resolver.Results = Resolver_Results(results)
	resolver.Name = Resolver_Name(name)
	resolver.Error = nil
	resolver.Socket = DNS_SOCKET_INVALID
	resolver.Deadline = Resolver_Deadline(now + time.Monotonic_Moment(timeout))
	resolver.Stage = RESOLVER_STAGE_OPEN_UDP
	resolver.Record_Type = Resolver_Record_Type(record_type)
	resolver.Port = Resolver_Port(port)
	resolver.Query_Bytes = 0
	resolver.Transfer_Bytes = 0
	resolver.Response_Bytes = 0
	resolver.Result_Count = 0
	resolver.Flags = RESOLVER_FLAG_ACTIVE
	completion.Data = 0
	completion.Error = nil
	resolver_query_build(completion)
	resolver_progress(completion)
}

// Query uses lowercase canonical copy so response owner comparison ignores presentation case.
func resolver_query_build(completion *nbio.Completion) {
	resolver := (*Resolver)(unsafe.Pointer(completion))
	query := resolver.Workspace.Query[:]
	for index := range query {
		query[index] = 0
	}
	base := DNS_TCP_SIZE_BYTES
	prng.Source_Read(
		prng.Source(resolver.Entropy),
		prng.Sink(query[base:base+DNS_TRANSACTION_BYTES]),
	)
	resolver.Transaction = Transaction_Identifier(
		uint16(query[base])<<8 | uint16(query[base+1]),
	)
	query[base+DNS_FLAGS_OFFSET] = byte(DNS_QUERY_FLAGS_RECURSION_DESIRED >> 8)
	query[base+DNS_QUESTION_COUNT_OFFSET+1] = 1
	query_offset := base + DNS_HEADER_BYTES
	canonical_count := 0
	name_count := len(resolver.Name)
	if resolver.Name[name_count-1] == '.' {
		name_count--
	}
	label_start := 0
	for index := 0; index <= name_count; index++ {
		if index < name_count {
			if resolver.Name[index] != '.' {
				continue
			}
		}
		label_count := index - label_start
		query[query_offset] = byte(label_count)
		resolver.Workspace.Canonical_Name[canonical_count] = byte(label_count)
		query_offset++
		canonical_count++
		for label_index := label_start; label_index < index; label_index++ {
			octet := resolver.Name[label_index]
			query[query_offset] = octet
			if octet >= 'A' {
				if octet <= 'Z' {
					octet += 'a' - 'A'
				}
			}
			resolver.Workspace.Canonical_Name[canonical_count] = octet
			query_offset++
			canonical_count++
		}
		label_start = index + 1
	}
	query[query_offset] = 0
	resolver.Workspace.Canonical_Name[canonical_count] = 0
	query_offset++
	query[query_offset] = byte(uint16(resolver.Record_Type) >> 8)
	query[query_offset+1] = byte(resolver.Record_Type)
	query_offset += DNS_RECORD_TYPE_BYTES
	query[query_offset+1] = byte(DNS_CLASS_IN)
	query_offset += DNS_CLASS_BYTES
	resolver.Query_Bytes = Query_Byte_Count(query_offset - base)
	query[0] = byte(resolver.Query_Bytes >> 8)
	query[1] = byte(resolver.Query_Bytes)
}

// Progress submits one operation at time and absorbs inline callback retirement without recursion.
func resolver_progress(completion *nbio.Completion) {
	resolver := (*Resolver)(unsafe.Pointer(completion))
	for resolver.Flags&RESOLVER_FLAG_ACTIVE != 0 {
		if resolver.Flags&RESOLVER_FLAG_WAIT_ACTIVE != 0 {
			return
		}
		resolver.Flags &^= RESOLVER_FLAG_CONTINUE
		switch resolver.Stage {
		case RESOLVER_STAGE_OPEN_UDP:
			resolver_udp_open(completion)
		case RESOLVER_STAGE_CONNECT_UDP, RESOLVER_STAGE_CONNECT_TCP:
			resolver_connect(completion)
		case RESOLVER_STAGE_SEND_UDP:
			resolver_udp_send(completion)
		case RESOLVER_STAGE_RECEIVE_UDP:
			resolver_udp_receive(completion)
		case RESOLVER_STAGE_CLOSE_UDP, RESOLVER_STAGE_CLOSE_TCP:
			resolver_close(completion)
		case RESOLVER_STAGE_OPEN_TCP:
			resolver_tcp_open(completion)
		case RESOLVER_STAGE_SEND_TCP:
			resolver_tcp_send(completion)
		case RESOLVER_STAGE_RECEIVE_TCP_SIZE:
			resolver_tcp_size_receive(completion)
		case RESOLVER_STAGE_RECEIVE_TCP_MESSAGE:
			resolver_tcp_message_receive(completion)
		case RESOLVER_STAGE_IDLE:
			panic("net: active Resolver is idle")
		}
		if resolver.Flags&RESOLVER_FLAG_WAIT_ACTIVE != 0 {
			return
		}
	}
}

func resolver_udp_open(completion *nbio.Completion) {
	resolver := (*Resolver)(unsafe.Pointer(completion))
	socket, socket_error := nbio.Network_Socket_UDP(
		resolver.IO.Network, resolver.Configuration.Server.Family,
		resolver.Configuration.UDP,
	)
	if socket_error != nil {
		resolver.Error = socket_error
		resolver_finish(completion)
		return
	}
	resolver.Socket = socket
	resolver.Stage = RESOLVER_STAGE_CONNECT_UDP
}

func resolver_tcp_open(completion *nbio.Completion) {
	resolver := (*Resolver)(unsafe.Pointer(completion))
	socket, socket_error := nbio.Network_Socket_TCP(
		resolver.IO.Network, resolver.Configuration.Server.Family,
		resolver.Configuration.TCP,
	)
	if socket_error != nil {
		resolver.Error = socket_error
		resolver_finish(completion)
		return
	}
	resolver.Socket = socket
	resolver.Transfer_Bytes = 0
	resolver.Stage = RESOLVER_STAGE_CONNECT_TCP
}

func resolver_connect(completion *nbio.Completion) {
	resolver := (*Resolver)(unsafe.Pointer(completion))
	if resolver_deadline_expired(completion) {
		return
	}
	resolver.Flags |= RESOLVER_FLAG_WAIT_ACTIVE | RESOLVER_FLAG_SUBMISSION_ACTIVE
	nbio.Network_Connect(
		resolver.IO.Network, completion, resolver.Socket,
		resolver.Configuration.Server, time.Duration(resolver_timeout(completion)),
		resolver_operation_complete,
	)
	resolver.Flags &^= RESOLVER_FLAG_SUBMISSION_ACTIVE
}

func resolver_udp_send(completion *nbio.Completion) {
	resolver := (*Resolver)(unsafe.Pointer(completion))
	if resolver_deadline_expired(completion) {
		return
	}
	resolver.Flags |= RESOLVER_FLAG_WAIT_ACTIVE | RESOLVER_FLAG_SUBMISSION_ACTIVE
	query_start := DNS_TCP_SIZE_BYTES
	query_end := query_start + int(resolver.Query_Bytes)
	nbio.Network_Send(
		resolver.IO.Network, completion, resolver.Socket,
		resolver.Workspace.Query[query_start:query_end],
		time.Duration(resolver_timeout(completion)), resolver_operation_complete,
	)
	resolver.Flags &^= RESOLVER_FLAG_SUBMISSION_ACTIVE
}

func resolver_udp_receive(completion *nbio.Completion) {
	resolver := (*Resolver)(unsafe.Pointer(completion))
	if resolver_deadline_expired(completion) {
		return
	}
	resolver.Flags |= RESOLVER_FLAG_WAIT_ACTIVE | RESOLVER_FLAG_SUBMISSION_ACTIVE
	nbio.Network_Receive(
		resolver.IO.Network, completion, resolver.Socket,
		resolver.Workspace.Response[:DNS_UDP_MESSAGE_BYTES_MAXIMUM],
		time.Duration(resolver_timeout(completion)), resolver_operation_complete,
	)
	resolver.Flags &^= RESOLVER_FLAG_SUBMISSION_ACTIVE
}

func resolver_close(completion *nbio.Completion) {
	resolver := (*Resolver)(unsafe.Pointer(completion))
	resolver.Flags |= RESOLVER_FLAG_WAIT_ACTIVE | RESOLVER_FLAG_SUBMISSION_ACTIVE
	nbio.IO_Close(resolver.IO, completion, resolver.Socket, resolver_operation_complete)
	resolver.Flags &^= RESOLVER_FLAG_SUBMISSION_ACTIVE
}

func resolver_tcp_send(completion *nbio.Completion) {
	resolver := (*Resolver)(unsafe.Pointer(completion))
	if resolver_deadline_expired(completion) {
		return
	}
	resolver.Flags |= RESOLVER_FLAG_WAIT_ACTIVE | RESOLVER_FLAG_SUBMISSION_ACTIVE
	nbio.Network_Send(
		resolver.IO.Network, completion, resolver.Socket,
		resolver.Workspace.Query[resolver.Transfer_Bytes:DNS_TCP_SIZE_BYTES+
			resolver.Query_Bytes],
		time.Duration(resolver_timeout(completion)), resolver_operation_complete,
	)
	resolver.Flags &^= RESOLVER_FLAG_SUBMISSION_ACTIVE
}

func resolver_tcp_size_receive(completion *nbio.Completion) {
	resolver := (*Resolver)(unsafe.Pointer(completion))
	if resolver_deadline_expired(completion) {
		return
	}
	resolver.Flags |= RESOLVER_FLAG_WAIT_ACTIVE | RESOLVER_FLAG_SUBMISSION_ACTIVE
	nbio.Network_Receive(
		resolver.IO.Network, completion, resolver.Socket,
		resolver.Workspace.Response[resolver.Transfer_Bytes:DNS_TCP_SIZE_BYTES],
		time.Duration(resolver_timeout(completion)), resolver_operation_complete,
	)
	resolver.Flags &^= RESOLVER_FLAG_SUBMISSION_ACTIVE
}

func resolver_tcp_message_receive(completion *nbio.Completion) {
	resolver := (*Resolver)(unsafe.Pointer(completion))
	if resolver_deadline_expired(completion) {
		return
	}
	resolver.Flags |= RESOLVER_FLAG_WAIT_ACTIVE | RESOLVER_FLAG_SUBMISSION_ACTIVE
	nbio.Network_Receive(
		resolver.IO.Network, completion, resolver.Socket,
		resolver.Workspace.Response[resolver.Transfer_Bytes:resolver.Response_Bytes],
		time.Duration(resolver_timeout(completion)), resolver_operation_complete,
	)
	resolver.Flags &^= RESOLVER_FLAG_SUBMISSION_ACTIVE
}

func resolver_timeout(completion *nbio.Completion) (timeout Resolver_Timeout) {
	defer func() { Resolver_Timeout_Invariants(timeout, "resolver_timeout.timeout") }()
	resolver := (*Resolver)(unsafe.Pointer(completion))
	now := time.Clock_Now_Monotonic(time.Clock(resolver.Clock))
	return Resolver_Timeout(time.Monotonic_Moment(resolver.Deadline) - now)
}

func resolver_deadline_expired(completion *nbio.Completion) (expired bytes.Boolean) {
	defer func() { bytes.Boolean_Invariants(expired, "resolver_deadline_expired.expired") }()
	resolver := (*Resolver)(unsafe.Pointer(completion))
	if time.Clock_Now_Monotonic(time.Clock(resolver.Clock)) <
		time.Monotonic_Moment(resolver.Deadline) {
		return false
	}
	resolver.Error = nbio.Deadline_Exceeded
	resolver_close_or_finish(completion)
	return true
}

func resolver_close_or_finish(completion *nbio.Completion) {
	resolver := (*Resolver)(unsafe.Pointer(completion))
	if resolver.Socket == DNS_SOCKET_INVALID {
		resolver_finish(completion)
		return
	}
	if resolver.Stage <= RESOLVER_STAGE_CLOSE_UDP {
		resolver.Stage = RESOLVER_STAGE_CLOSE_UDP
		return
	}
	resolver.Stage = RESOLVER_STAGE_CLOSE_TCP
}

func resolver_operation_complete(completion *nbio.Completion) {
	resolver := (*Resolver)(unsafe.Pointer(completion))
	aver.Always(
		resolver.Flags&RESOLVER_FLAG_ACTIVE != 0,
		"Resolver callback belongs to active operation.",
	)
	aver.Always(
		resolver.Flags&RESOLVER_FLAG_WAIT_ACTIVE != 0,
		"Resolver callback retires submitted operation.",
	)
	resolver.Flags &^= RESOLVER_FLAG_WAIT_ACTIVE
	if completion.Error != nil {
		close_stage := resolver.Stage == RESOLVER_STAGE_CLOSE_UDP
		if !close_stage {
			close_stage = resolver.Stage == RESOLVER_STAGE_CLOSE_TCP
		}
		if close_stage {
			resolver.Socket = DNS_SOCKET_INVALID
			if resolver.Error == nil {
				resolver.Error = completion.Error
			}
			resolver_finish(completion)
		} else {
			resolver.Error = completion.Error
			resolver_close_or_finish(completion)
		}
	} else {
		resolver_operation_succeeded(completion)
	}
	if resolver.Flags&RESOLVER_FLAG_SUBMISSION_ACTIVE != 0 {
		resolver.Flags |= RESOLVER_FLAG_CONTINUE
		return
	}
	resolver_progress(completion)
}

func resolver_operation_succeeded(completion *nbio.Completion) {
	resolver := (*Resolver)(unsafe.Pointer(completion))
	switch resolver.Stage {
	case RESOLVER_STAGE_CONNECT_UDP:
		resolver.Stage = RESOLVER_STAGE_SEND_UDP
	case RESOLVER_STAGE_SEND_UDP:
		count := resolver.Completion.Data
		if count <= 0 {
			if count == 0 {
				resolver.Error = Error_Transfer_No_Progress
			} else {
				resolver.Error = Error_Transfer_Count
			}
			resolver.Stage = RESOLVER_STAGE_CLOSE_UDP
			return
		}
		if count != int(resolver.Query_Bytes) {
			resolver.Error = Error_Transfer_Count
			resolver.Stage = RESOLVER_STAGE_CLOSE_UDP
			return
		}
		resolver.Stage = RESOLVER_STAGE_RECEIVE_UDP
	case RESOLVER_STAGE_RECEIVE_UDP:
		resolver_receive_udp_complete(completion)
	case RESOLVER_STAGE_CLOSE_UDP:
		resolver.Socket = DNS_SOCKET_INVALID
		if resolver.Flags&RESOLVER_FLAG_TCP_FALLBACK != 0 {
			if resolver.Error == nil {
				resolver.Stage = RESOLVER_STAGE_OPEN_TCP
				return
			}
		}
		resolver_finish(completion)
	case RESOLVER_STAGE_CONNECT_TCP:
		resolver.Transfer_Bytes = 0
		resolver.Stage = RESOLVER_STAGE_SEND_TCP
	case RESOLVER_STAGE_SEND_TCP:
		resolver_send_tcp_complete(completion)
	case RESOLVER_STAGE_RECEIVE_TCP_SIZE:
		resolver_receive_tcp_size_complete(completion)
	case RESOLVER_STAGE_RECEIVE_TCP_MESSAGE:
		resolver_receive_tcp_message_complete(completion)
	case RESOLVER_STAGE_CLOSE_TCP:
		resolver.Socket = DNS_SOCKET_INVALID
		resolver_finish(completion)
	default:
		panic("net: unknown Resolver completion stage")
	}
}

func resolver_receive_udp_complete(completion *nbio.Completion) {
	resolver := (*Resolver)(unsafe.Pointer(completion))
	count := resolver.Completion.Data
	if count <= 0 {
		if count == 0 {
			resolver.Error = Error_Transfer_No_Progress
		} else {
			resolver.Error = Error_Transfer_Count
		}
		resolver.Stage = RESOLVER_STAGE_CLOSE_UDP
		return
	}
	if count > DNS_UDP_MESSAGE_BYTES_MAXIMUM {
		resolver.Error = Error_Transfer_Count
		resolver.Stage = RESOLVER_STAGE_CLOSE_UDP
		return
	}
	resolver.Response_Bytes = Response_Byte_Count(count)
	if count < DNS_HEADER_BYTES {
		resolver.Error = Error_Response_Malformed
		resolver.Stage = RESOLVER_STAGE_CLOSE_UDP
		return
	}
	flags := uint16(resolver.Workspace.Response[DNS_FLAGS_OFFSET])<<8 |
		uint16(resolver.Workspace.Response[DNS_FLAGS_OFFSET+1])
	if flags&DNS_TRUNCATED_BIT != 0 {
		response_truncated_validate(completion)
		if resolver.Error != nil {
			resolver.Stage = RESOLVER_STAGE_CLOSE_UDP
			return
		}
		resolver.Flags |= RESOLVER_FLAG_TCP_FALLBACK
		resolver.Stage = RESOLVER_STAGE_CLOSE_UDP
		return
	}
	resolver_response_parse(completion)
	resolver.Stage = RESOLVER_STAGE_CLOSE_UDP
}

// TCP costs another descriptor and stream handshake, so an untrusted datagram must prove it owns
// this question before it can force fallback.
func response_truncated_validate(completion *nbio.Completion) {
	resolver := (*Resolver)(unsafe.Pointer(completion))
	message := resolver.Workspace.Response[:resolver.Response_Bytes]
	transaction := uint16(message[0])<<8 | uint16(message[1])
	if transaction != uint16(resolver.Transaction) {
		resolver.Error = Error_Response_Mismatch
		return
	}
	flags := uint16(message[DNS_FLAGS_OFFSET])<<8 | uint16(message[DNS_FLAGS_OFFSET+1])
	if flags&DNS_RESPONSE_BIT == 0 {
		resolver.Error = Error_Response_Mismatch
		return
	}
	if flags&DNS_OPCODE_MASK != 0 {
		resolver.Error = Error_Response_Mismatch
		return
	}
	question_count := uint16(message[DNS_QUESTION_COUNT_OFFSET])<<8 |
		uint16(message[DNS_QUESTION_COUNT_OFFSET+1])
	if question_count != 1 {
		resolver.Error = Error_Response_Mismatch
		return
	}
	response_question_parse(completion)
}

func resolver_send_tcp_complete(completion *nbio.Completion) {
	resolver := (*Resolver)(unsafe.Pointer(completion))
	count := resolver.Completion.Data
	remaining_count := DNS_TCP_SIZE_BYTES + int(resolver.Query_Bytes) -
		int(resolver.Transfer_Bytes)
	if count <= 0 {
		if count == 0 {
			resolver.Error = Error_Transfer_No_Progress
		} else {
			resolver.Error = Error_Transfer_Count
		}
		resolver.Stage = RESOLVER_STAGE_CLOSE_TCP
		return
	}
	if count > remaining_count {
		resolver.Error = Error_Transfer_Count
		resolver.Stage = RESOLVER_STAGE_CLOSE_TCP
		return
	}
	resolver.Transfer_Bytes += Transfer_Byte_Count(count)
	if int(resolver.Transfer_Bytes) == DNS_TCP_SIZE_BYTES+int(resolver.Query_Bytes) {
		resolver.Transfer_Bytes = 0
		resolver.Stage = RESOLVER_STAGE_RECEIVE_TCP_SIZE
	}
}

func resolver_receive_tcp_size_complete(completion *nbio.Completion) {
	resolver := (*Resolver)(unsafe.Pointer(completion))
	count := resolver.Completion.Data
	remaining_count := DNS_TCP_SIZE_BYTES - int(resolver.Transfer_Bytes)
	if count <= 0 {
		if count == 0 {
			resolver.Error = Error_Transfer_No_Progress
		} else {
			resolver.Error = Error_Transfer_Count
		}
		resolver.Stage = RESOLVER_STAGE_CLOSE_TCP
		return
	}
	if count > remaining_count {
		resolver.Error = Error_Transfer_Count
		resolver.Stage = RESOLVER_STAGE_CLOSE_TCP
		return
	}
	resolver.Transfer_Bytes += Transfer_Byte_Count(count)
	if resolver.Transfer_Bytes != DNS_TCP_SIZE_BYTES {
		return
	}
	resolver.Response_Bytes = Response_Byte_Count(
		int(resolver.Workspace.Response[0])<<8 | int(resolver.Workspace.Response[1]),
	)
	if resolver.Response_Bytes < DNS_HEADER_BYTES {
		resolver.Error = Error_Response_Malformed
		resolver.Stage = RESOLVER_STAGE_CLOSE_TCP
		return
	}
	resolver.Transfer_Bytes = 0
	resolver.Stage = RESOLVER_STAGE_RECEIVE_TCP_MESSAGE
}

func resolver_receive_tcp_message_complete(completion *nbio.Completion) {
	resolver := (*Resolver)(unsafe.Pointer(completion))
	count := resolver.Completion.Data
	remaining_count := int(resolver.Response_Bytes) - int(resolver.Transfer_Bytes)
	if count <= 0 {
		if count == 0 {
			resolver.Error = Error_Transfer_No_Progress
		} else {
			resolver.Error = Error_Transfer_Count
		}
		resolver.Stage = RESOLVER_STAGE_CLOSE_TCP
		return
	}
	if count > remaining_count {
		resolver.Error = Error_Transfer_Count
		resolver.Stage = RESOLVER_STAGE_CLOSE_TCP
		return
	}
	resolver.Transfer_Bytes += Transfer_Byte_Count(count)
	if int(resolver.Transfer_Bytes) != int(resolver.Response_Bytes) {
		return
	}
	resolver_response_parse(completion)
	resolver.Stage = RESOLVER_STAGE_CLOSE_TCP
}

func resolver_finish(completion *nbio.Completion) {
	resolver := (*Resolver)(unsafe.Pointer(completion))
	callback := resolver.Callback
	result_count := resolver.Result_Count
	err := resolver.Error
	resolver.Callback = nil
	resolver.Results = nil
	resolver.Name = ""
	resolver.Error = nil
	resolver.Socket = DNS_SOCKET_INVALID
	resolver.Deadline = 0
	resolver.Stage = RESOLVER_STAGE_IDLE
	resolver.Record_Type = Resolver_Record_Type(RECORD_TYPE_INVALID)
	resolver.Port = 0
	resolver.Transaction = 0
	resolver.Query_Bytes = 0
	resolver.Transfer_Bytes = 0
	resolver.Response_Bytes = 0
	resolver.Result_Count = 0
	resolver.Flags = RESOLVER_FLAGS_NONE
	resolver.Completion.Data = int(result_count)
	resolver.Completion.Error = err
	callback(&resolver.Completion)
}

// Response_Offset is current encoded-name position.
type Response_Offset uint16

// Response_Offset_Invariants bounds position by largest DNS body.
func Response_Offset_Invariants(value Response_Offset, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(RESPONSE_OFFSET_MINIMUM),
			uint16(RESPONSE_OFFSET_MAXIMUM),
		).
		Ensure()
}

// RESPONSE_OFFSET_MINIMUM is first response byte.
const RESPONSE_OFFSET_MINIMUM Response_Offset = DNS_HEADER_BYTES

// RESPONSE_OFFSET_MAXIMUM is last possible response boundary.
const RESPONSE_OFFSET_MAXIMUM Response_Offset = DNS_MESSAGE_BYTES_MAXIMUM

// Response_Answer_Offset is current answer-record boundary.
type Response_Answer_Offset uint16

// Response_Answer_Offset_Invariants starts after smallest valid question.
func Response_Answer_Offset_Invariants(
	value Response_Answer_Offset, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(RESPONSE_ANSWER_OFFSET_MINIMUM),
			uint16(RESPONSE_ANSWER_OFFSET_MAXIMUM),
		).
		Ensure()
}

// RESPONSE_ANSWER_OFFSET_MINIMUM follows smallest valid question.
const RESPONSE_ANSWER_OFFSET_MINIMUM Response_Answer_Offset = DNS_QUERY_BYTES_MINIMUM

// RESPONSE_ANSWER_OFFSET_MAXIMUM admits a malicious extra answer at message end.
const RESPONSE_ANSWER_OFFSET_MAXIMUM Response_Answer_Offset = DNS_MESSAGE_BYTES_MAXIMUM

// Response_Label_Offset is a label-size byte inside response body.
type Response_Label_Offset uint16

// Response_Label_Offset_Invariants leaves one byte for label-size field itself.
func Response_Label_Offset_Invariants(
	value Response_Label_Offset, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(RESPONSE_LABEL_OFFSET_MINIMUM),
			uint16(RESPONSE_LABEL_OFFSET_MAXIMUM),
		).
		Ensure()
}

// RESPONSE_LABEL_OFFSET_MINIMUM is first question-name byte.
const RESPONSE_LABEL_OFFSET_MINIMUM Response_Label_Offset = DNS_HEADER_BYTES

// RESPONSE_LABEL_OFFSET_MAXIMUM is last message byte.
const RESPONSE_LABEL_OFFSET_MAXIMUM Response_Label_Offset = DNS_MESSAGE_BYTES_MAXIMUM - 1

// Response_Encoded_Next is first byte after encoded name.
type Response_Encoded_Next uint16

// Response_Encoded_Next_Invariants bounds position by largest DNS body.
func Response_Encoded_Next_Invariants(
	value Response_Encoded_Next, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(RESPONSE_ENCODED_NEXT_MINIMUM),
			uint16(RESPONSE_ENCODED_NEXT_MAXIMUM),
		).
		Ensure()
}

// RESPONSE_ENCODED_NEXT_MINIMUM also marks no compressed continuation.
const RESPONSE_ENCODED_NEXT_MINIMUM Response_Encoded_Next = DNS_HEADER_BYTES

// RESPONSE_ENCODED_NEXT_MAXIMUM is last possible response boundary.
const RESPONSE_ENCODED_NEXT_MAXIMUM Response_Encoded_Next = DNS_MESSAGE_BYTES_MAXIMUM

// Response_Decoded_Byte_Count is expanded decoded name size.
type Response_Decoded_Byte_Count uint8

// Response_Decoded_Byte_Count_Invariants bounds expanded name by protocol field.
func Response_Decoded_Byte_Count_Invariants(
	value Response_Decoded_Byte_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Holed_Uint8(
			uint8(value), uint8(RESPONSE_DECODED_BYTE_COUNT_MINIMUM),
			uint8(RESPONSE_DECODED_BYTE_COUNT_MAXIMUM),
			DNS_NAME_WIRE_BYTES_IMPOSSIBLE, DNS_NAME_WIRE_BYTES_IMPOSSIBLE,
			DNS_NAME_WIRE_BYTES_IMPOSSIBLE,
		).
		Ensure()
}

// RESPONSE_DECODED_BYTE_COUNT_MINIMUM is empty parser scratch.
const RESPONSE_DECODED_BYTE_COUNT_MINIMUM Response_Decoded_Byte_Count = 0

// RESPONSE_DECODED_BYTE_COUNT_MAXIMUM is largest expanded DNS name.
const RESPONSE_DECODED_BYTE_COUNT_MAXIMUM Response_Decoded_Byte_Count = DNS_NAME_WIRE_BYTES_MAXIMUM

// DNS_NAME_WIRE_BYTES_IMPOSSIBLE cannot hold both a label-size byte and root byte.
const DNS_NAME_WIRE_BYTES_IMPOSSIBLE = DNS_NAME_ROOT_BYTES + DNS_NAME_ROOT_BYTES

// Response_Decoded_Prefix_Byte_Count is label bytes copied before encoded root.
type Response_Decoded_Prefix_Byte_Count uint8

// Response_Decoded_Prefix_Byte_Count_Invariants excludes one-byte partial label encoding.
func Response_Decoded_Prefix_Byte_Count_Invariants(
	value Response_Decoded_Prefix_Byte_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Holed_Uint8(
			uint8(value), uint8(RESPONSE_DECODED_PREFIX_BYTE_COUNT_MINIMUM),
			uint8(RESPONSE_DECODED_PREFIX_BYTE_COUNT_MAXIMUM),
			DNS_NAME_ROOT_BYTES, DNS_NAME_ROOT_BYTES, DNS_NAME_ROOT_BYTES,
		).
		Ensure()
}

// RESPONSE_DECODED_PREFIX_BYTE_COUNT_MINIMUM is empty decoded scratch.
const RESPONSE_DECODED_PREFIX_BYTE_COUNT_MINIMUM Response_Decoded_Prefix_Byte_Count = 0

// RESPONSE_DECODED_PREFIX_BYTE_COUNT_MAXIMUM leaves room for encoded root.
const RESPONSE_DECODED_PREFIX_BYTE_COUNT_MAXIMUM = Response_Decoded_Prefix_Byte_Count(
	DNS_NAME_WIRE_BYTES_MAXIMUM - DNS_NAME_ROOT_BYTES,
)

// Response_Name_Byte_Count is one successfully decoded expanded DNS name.
type Response_Name_Byte_Count uint8

// Response_Name_Byte_Count_Invariants excludes empty error output and impossible two-byte name.
func Response_Name_Byte_Count_Invariants(
	value Response_Name_Byte_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Holed_Uint8(
			uint8(value), uint8(RESPONSE_NAME_BYTE_COUNT_MINIMUM),
			uint8(RESPONSE_NAME_BYTE_COUNT_MAXIMUM),
			DNS_NAME_WIRE_BYTES_IMPOSSIBLE, DNS_NAME_WIRE_BYTES_IMPOSSIBLE,
			DNS_NAME_WIRE_BYTES_IMPOSSIBLE,
		).
		Ensure()
}

// RESPONSE_NAME_BYTE_COUNT_MINIMUM is root name.
const RESPONSE_NAME_BYTE_COUNT_MINIMUM Response_Name_Byte_Count = DNS_NAME_ROOT_BYTES

// RESPONSE_NAME_BYTE_COUNT_MAXIMUM is largest expanded DNS name.
const RESPONSE_NAME_BYTE_COUNT_MAXIMUM Response_Name_Byte_Count = DNS_NAME_WIRE_BYTES_MAXIMUM

// Response_Canonical_Byte_Count is current expanded target name size.
type Response_Canonical_Byte_Count uint8

// Response_Canonical_Byte_Count_Invariants bounds target by expanded-name ceiling.
func Response_Canonical_Byte_Count_Invariants(
	value Response_Canonical_Byte_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Holed_Uint8(
			uint8(value), uint8(RESPONSE_CANONICAL_BYTE_COUNT_MINIMUM),
			uint8(RESPONSE_CANONICAL_BYTES_MAXIMUM),
			DNS_NAME_WIRE_BYTES_IMPOSSIBLE, DNS_NAME_WIRE_BYTES_IMPOSSIBLE,
			DNS_NAME_WIRE_BYTES_IMPOSSIBLE,
		).
		Ensure()
}

// RESPONSE_CANONICAL_BYTE_COUNT_MINIMUM is idle parser target.
const RESPONSE_CANONICAL_BYTE_COUNT_MINIMUM = Response_Canonical_Byte_Count(
	DNS_NAME_ROOT_BYTES,
)

// RESPONSE_CANONICAL_BYTES_MAXIMUM is largest expanded DNS name.
const RESPONSE_CANONICAL_BYTES_MAXIMUM Response_Canonical_Byte_Count = DNS_NAME_WIRE_BYTES_MAXIMUM

// Response_Answer_Count is answer records named by header.
type Response_Answer_Count uint16

// Response_Answer_Count_Invariants covers complete header field.
func Response_Answer_Count_Invariants(
	value Response_Answer_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(RESPONSE_ANSWER_COUNT_MINIMUM),
			uint16(RESPONSE_ANSWER_COUNT_MAXIMUM),
		).
		Ensure()
}

// RESPONSE_ANSWER_COUNT_MINIMUM is response without answers.
const RESPONSE_ANSWER_COUNT_MINIMUM Response_Answer_Count = 0

// RESPONSE_ANSWER_COUNT_MAXIMUM closes unsigned header field.
const RESPONSE_ANSWER_COUNT_MAXIMUM Response_Answer_Count = (1 << DNS_MESSAGE_SIZE_BITS) - 1

// Response_Answer_Start is first answer-record position.
type Response_Answer_Start uint16

// Response_Answer_Start_Invariants bounds position by largest DNS body.
func Response_Answer_Start_Invariants(
	value Response_Answer_Start, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(RESPONSE_ANSWER_START_MINIMUM),
			uint16(RESPONSE_ANSWER_START_MAXIMUM),
		).
		Ensure()
}

// RESPONSE_ANSWER_START_MINIMUM is idle parser position.
const RESPONSE_ANSWER_START_MINIMUM Response_Answer_Start = DNS_QUERY_BYTES_MINIMUM

// RESPONSE_ANSWER_START_MAXIMUM is last possible response boundary.
const RESPONSE_ANSWER_START_MAXIMUM Response_Answer_Start = DNS_QUERY_BYTES_MAXIMUM

// Response_Resource_Offset is first resource-data byte.
type Response_Resource_Offset uint16

// Response_Resource_Offset_Invariants bounds position by largest DNS body.
func Response_Resource_Offset_Invariants(
	value Response_Resource_Offset, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(RESPONSE_RESOURCE_OFFSET_MINIMUM),
			uint16(RESPONSE_RESOURCE_OFFSET_MAXIMUM),
		).
		Ensure()
}

// RESPONSE_RESOURCE_OFFSET_MINIMUM is idle resource position.
const RESPONSE_RESOURCE_OFFSET_MINIMUM Response_Resource_Offset = DNS_QUERY_BYTES_MINIMUM +
	DNS_COMPRESSION_POINTER_BYTES + DNS_RESOURCE_FIXED_BYTES

// RESPONSE_RESOURCE_OFFSET_MAXIMUM is last possible response boundary.
const RESPONSE_RESOURCE_OFFSET_MAXIMUM Response_Resource_Offset = DNS_MESSAGE_BYTES_MAXIMUM -
	DNS_NAME_ROOT_BYTES

// Response_Address_Offset is first byte of validated A or AAAA payload.
type Response_Address_Offset uint16

// Response_Address_Offset_Invariants leaves room for smallest supported address.
func Response_Address_Offset_Invariants(
	value Response_Address_Offset, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(RESPONSE_ADDRESS_OFFSET_MINIMUM),
			uint16(RESPONSE_ADDRESS_OFFSET_MAXIMUM),
		).
		Ensure()
}

// RESPONSE_ADDRESS_OFFSET_MINIMUM follows smallest question and compressed resource header.
const RESPONSE_ADDRESS_OFFSET_MINIMUM Response_Address_Offset = DNS_QUERY_BYTES_MINIMUM +
	DNS_COMPRESSION_POINTER_BYTES + DNS_RESOURCE_FIXED_BYTES

// RESPONSE_ADDRESS_OFFSET_MAXIMUM leaves room for smallest supported address.
const RESPONSE_ADDRESS_OFFSET_MAXIMUM Response_Address_Offset = DNS_MESSAGE_BYTES_MAXIMUM -
	nbio.IPV4_ADDRESS_BYTES

// Response_Resource_End is first byte after complete CNAME resource.
type Response_Resource_End uint16

// Response_Resource_End_Invariants includes smallest compressed root CNAME through message end.
func Response_Resource_End_Invariants(
	value Response_Resource_End, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(RESPONSE_RESOURCE_END_MINIMUM),
			uint16(RESPONSE_RESOURCE_END_MAXIMUM),
		).
		Ensure()
}

// RESPONSE_RESOURCE_END_MINIMUM follows smallest question and compressed root CNAME.
const RESPONSE_RESOURCE_END_MINIMUM Response_Resource_End = DNS_QUERY_BYTES_MINIMUM +
	DNS_COMPRESSION_POINTER_BYTES + DNS_RESOURCE_FIXED_BYTES + DNS_NAME_ROOT_BYTES

// RESPONSE_RESOURCE_END_MAXIMUM is message boundary.
const RESPONSE_RESOURCE_END_MAXIMUM Response_Resource_End = DNS_MESSAGE_BYTES_MAXIMUM

// Response_Alias_Byte_Count is preserved CNAME target size.
type Response_Alias_Byte_Count uint8

// Response_Alias_Byte_Count_Invariants bounds target by expanded-name ceiling.
func Response_Alias_Byte_Count_Invariants(
	value Response_Alias_Byte_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Holed_Uint8(
			uint8(value), uint8(RESPONSE_ALIAS_BYTE_COUNT_MINIMUM),
			uint8(RESPONSE_ALIAS_BYTE_COUNT_MAXIMUM),
			DNS_NAME_WIRE_BYTES_IMPOSSIBLE, DNS_NAME_WIRE_BYTES_IMPOSSIBLE,
			DNS_NAME_WIRE_BYTES_IMPOSSIBLE,
		).
		Ensure()
}

// RESPONSE_ALIAS_BYTE_COUNT_MINIMUM reports no matching CNAME.
const RESPONSE_ALIAS_BYTE_COUNT_MINIMUM Response_Alias_Byte_Count = 0

// RESPONSE_ALIAS_BYTE_COUNT_MAXIMUM is largest expanded DNS name.
const RESPONSE_ALIAS_BYTE_COUNT_MAXIMUM Response_Alias_Byte_Count = DNS_NAME_WIRE_BYTES_MAXIMUM

// Response_Alias_Name_Byte_Count is one present expanded CNAME target.
type Response_Alias_Name_Byte_Count uint8

// Response_Alias_Name_Byte_Count_Invariants excludes absent and impossible name sizes.
func Response_Alias_Name_Byte_Count_Invariants(
	value Response_Alias_Name_Byte_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Holed_Uint8(
			uint8(value), uint8(RESPONSE_ALIAS_NAME_BYTE_COUNT_MINIMUM),
			uint8(RESPONSE_ALIAS_NAME_BYTE_COUNT_MAXIMUM),
			DNS_NAME_WIRE_BYTES_IMPOSSIBLE, DNS_NAME_WIRE_BYTES_IMPOSSIBLE,
			DNS_NAME_WIRE_BYTES_IMPOSSIBLE,
		).
		Ensure()
}

// RESPONSE_ALIAS_NAME_BYTE_COUNT_MINIMUM is root target.
const RESPONSE_ALIAS_NAME_BYTE_COUNT_MINIMUM = Response_Alias_Name_Byte_Count(
	DNS_NAME_ROOT_BYTES,
)

// RESPONSE_ALIAS_NAME_BYTE_COUNT_MAXIMUM is largest expanded CNAME target.
const RESPONSE_ALIAS_NAME_BYTE_COUNT_MAXIMUM = Response_Alias_Name_Byte_Count(
	DNS_NAME_WIRE_BYTES_MAXIMUM,
)

func resolver_response_parse(completion *nbio.Completion) {
	resolver := (*Resolver)(unsafe.Pointer(completion))
	answer_count := response_header_parse(completion)
	if resolver.Error != nil {
		return
	}
	answer_start := response_question_parse(completion)
	if resolver.Error != nil {
		return
	}
	response_answers_parse(completion, answer_start, answer_count)
}

func response_header_parse(
	completion *nbio.Completion,
) (answer_count Response_Answer_Count) {
	defer func() {
		Response_Answer_Count_Invariants(answer_count, "response_header_parse.answer_count")
	}()
	resolver := (*Resolver)(unsafe.Pointer(completion))
	message := resolver.Workspace.Response[:resolver.Response_Bytes]
	if len(message) < DNS_HEADER_BYTES {
		resolver.Error = Error_Response_Malformed
		return 0
	}
	transaction := uint16(message[0])<<8 | uint16(message[1])
	if transaction != uint16(resolver.Transaction) {
		resolver.Error = Error_Response_Mismatch
		return 0
	}
	flags := uint16(message[DNS_FLAGS_OFFSET])<<8 | uint16(message[DNS_FLAGS_OFFSET+1])
	if flags&DNS_RESPONSE_BIT == 0 {
		resolver.Error = Error_Response_Mismatch
		return 0
	}
	if flags&DNS_OPCODE_MASK != 0 {
		resolver.Error = Error_Response_Mismatch
		return 0
	}
	// TCP has no second transport fallback. Its length prefix already bounds complete bytes.
	if flags&DNS_TRUNCATED_BIT != 0 {
		if resolver.Flags&RESOLVER_FLAG_TCP_FALLBACK == 0 {
			resolver.Error = Error_Response_Malformed
			return 0
		}
	}
	response_code := flags & DNS_RESPONSE_CODE_MASK
	if response_code == DNS_RESPONSE_CODE_NAME_ERROR {
		resolver.Error = Error_Name_Not_Found
		return 0
	}
	if response_code == DNS_RESPONSE_CODE_SERVER_FAILURE {
		resolver.Error = Error_Server_Failure
		return 0
	}
	if response_code != DNS_RESPONSE_CODE_SUCCESS {
		resolver.Error = Error_Response_Malformed
		return 0
	}
	question_count := uint16(message[DNS_QUESTION_COUNT_OFFSET])<<8 |
		uint16(message[DNS_QUESTION_COUNT_OFFSET+1])
	if question_count != 1 {
		resolver.Error = Error_Response_Mismatch
		return 0
	}
	return Response_Answer_Count(
		uint16(message[DNS_ANSWER_COUNT_OFFSET])<<8 |
			uint16(message[DNS_ANSWER_COUNT_OFFSET+1]),
	)
}

func response_question_parse(
	completion *nbio.Completion,
) (answer_start Response_Answer_Start) {
	defer func() {
		Response_Answer_Start_Invariants(
			answer_start, "response_question_parse.answer_start",
		)
	}()
	resolver := (*Resolver)(unsafe.Pointer(completion))
	message := resolver.Workspace.Response[:resolver.Response_Bytes]
	next, decoded_bytes := response_name_decode(completion, DNS_HEADER_BYTES)
	if resolver.Error != nil {
		return RESPONSE_ANSWER_START_MINIMUM
	}
	canonical_bytes := Response_Canonical_Byte_Count(
		int(resolver.Query_Bytes) - DNS_HEADER_BYTES - DNS_QUESTION_FIXED_BYTES,
	)
	if !bool(response_decoded_match(
		completion, Response_Name_Byte_Count(decoded_bytes), canonical_bytes,
	)) {
		resolver.Error = Error_Response_Mismatch
		return RESPONSE_ANSWER_START_MINIMUM
	}
	offset := Response_Offset(next)
	if int(offset)+DNS_QUESTION_FIXED_BYTES > len(message) {
		resolver.Error = Error_Response_Malformed
		return RESPONSE_ANSWER_START_MINIMUM
	}
	question_type := uint16(message[offset])<<8 | uint16(message[offset+1])
	class_offset := offset + DNS_RECORD_TYPE_BYTES
	question_class := uint16(message[class_offset])<<8 |
		uint16(message[class_offset+1])
	if question_type != uint16(resolver.Record_Type) {
		resolver.Error = Error_Response_Mismatch
		return RESPONSE_ANSWER_START_MINIMUM
	}
	if question_class != DNS_CLASS_IN {
		resolver.Error = Error_Response_Mismatch
		return RESPONSE_ANSWER_START_MINIMUM
	}
	return Response_Answer_Start(int(offset) + DNS_QUESTION_FIXED_BYTES)
}

func response_answers_parse(
	completion *nbio.Completion, answer_start Response_Answer_Start,
	answer_count Response_Answer_Count,
) {
	Response_Answer_Start_Invariants(answer_start, "response_answers_parse.answer_start")
	Response_Answer_Count_Invariants(answer_count, "response_answers_parse.answer_count")
	resolver := (*Resolver)(unsafe.Pointer(completion))
	canonical_bytes := Response_Canonical_Byte_Count(
		int(resolver.Query_Bytes) - DNS_HEADER_BYTES - DNS_QUESTION_FIXED_BYTES,
	)
	for alias_hop := 0; alias_hop <= DNS_CNAME_HOPS_MAXIMUM; alias_hop++ {
		alias_bytes := RESPONSE_ALIAS_BYTE_COUNT_MINIMUM
		offset := Response_Answer_Offset(answer_start)
		answer_index := RESPONSE_ANSWER_COUNT_MINIMUM
		for answer_index < answer_count {
			offset, alias_bytes = response_resource_parse(
				completion, offset, canonical_bytes, alias_bytes,
			)
			if resolver.Error != nil {
				return
			}
			answer_index++
		}
		if resolver.Result_Count > 0 {
			return
		}
		if alias_bytes == 0 {
			resolver.Error = Error_No_Address
			return
		}
		if bool(response_alias_is_canonical(
			completion, Response_Alias_Name_Byte_Count(alias_bytes), canonical_bytes,
		)) {
			resolver.Error = Error_Response_Malformed
			return
		}
		copy(
			resolver.Workspace.Canonical_Name[:int(alias_bytes)],
			resolver.Workspace.Alias_Name[:int(alias_bytes)],
		)
		canonical_bytes = Response_Canonical_Byte_Count(alias_bytes)
	}
	resolver.Error = Error_Response_Malformed
}

func response_resource_parse(
	completion *nbio.Completion, offset Response_Answer_Offset,
	canonical_bytes Response_Canonical_Byte_Count,
	alias_bytes Response_Alias_Byte_Count,
) (next Response_Answer_Offset, next_alias Response_Alias_Byte_Count) {
	defer func() {
		Response_Answer_Offset_Invariants(next, "response_resource_parse.next")
		Response_Alias_Byte_Count_Invariants(
			next_alias, "response_resource_parse.next_alias",
		)
	}()
	Response_Answer_Offset_Invariants(offset, "response_resource_parse.offset")
	Response_Canonical_Byte_Count_Invariants(
		canonical_bytes, "response_resource_parse.canonical_bytes",
	)
	Response_Alias_Byte_Count_Invariants(alias_bytes, "response_resource_parse.alias_bytes")
	resolver := (*Resolver)(unsafe.Pointer(completion))
	message := resolver.Workspace.Response[:resolver.Response_Bytes]
	owner_next, owner_bytes := response_name_decode(completion, Response_Offset(offset))
	if resolver.Error != nil {
		return RESPONSE_ANSWER_OFFSET_MINIMUM, alias_bytes
	}
	owner_matches := response_decoded_match(
		completion, Response_Name_Byte_Count(owner_bytes), canonical_bytes,
	)
	offset = Response_Answer_Offset(owner_next)
	if int(offset)+DNS_RESOURCE_FIXED_BYTES > len(message) {
		resolver.Error = Error_Response_Malformed
		return RESPONSE_ANSWER_OFFSET_MINIMUM, alias_bytes
	}
	record_type := uint16(message[offset])<<8 | uint16(message[offset+1])
	class_offset := offset + DNS_RECORD_TYPE_BYTES
	record_class := uint16(message[class_offset])<<8 | uint16(message[class_offset+1])
	size_offset := class_offset + DNS_CLASS_BYTES + DNS_TTL_BYTES
	resource_size := uint16(message[size_offset])<<8 | uint16(message[size_offset+1])
	resource_offset := Response_Resource_Offset(int(size_offset) + DNS_RESOURCE_SIZE_BYTES)
	end_offset := int(resource_offset) + int(resource_size)
	if end_offset > len(message) {
		resolver.Error = Error_Response_Malformed
		return RESPONSE_ANSWER_OFFSET_MINIMUM, alias_bytes
	}
	next = Response_Answer_Offset(end_offset)
	if !bool(owner_matches) {
		return next, alias_bytes
	}
	if record_class != DNS_CLASS_IN {
		return next, alias_bytes
	}
	if record_type == uint16(resolver.Record_Type) {
		expected_size := nbio.IPV4_ADDRESS_BYTES
		if Record_Type(resolver.Record_Type) == RECORD_TYPE_AAAA {
			expected_size = nbio.IPV6_ADDRESS_BYTES
		}
		if int(resource_size) != expected_size {
			resolver.Error = Error_Response_Malformed
			return next, alias_bytes
		}
		response_address_append(completion, Response_Address_Offset(resource_offset))
		return next, alias_bytes
	}
	if record_type == DNS_RECORD_TYPE_CNAME {
		next_alias = response_alias_parse(
			completion, resource_offset, Response_Resource_End(next), alias_bytes,
		)
		return next, next_alias
	}
	return next, alias_bytes
}

func response_alias_parse(
	completion *nbio.Completion, resource_offset Response_Resource_Offset,
	end_offset Response_Resource_End, alias_bytes Response_Alias_Byte_Count,
) (next_alias Response_Alias_Byte_Count) {
	defer func() {
		Response_Alias_Byte_Count_Invariants(next_alias, "response_alias_parse.next_alias")
	}()
	Response_Resource_Offset_Invariants(resource_offset, "response_alias_parse.resource_offset")
	Response_Resource_End_Invariants(end_offset, "response_alias_parse.end_offset")
	Response_Alias_Byte_Count_Invariants(alias_bytes, "response_alias_parse.alias_bytes")
	resolver := (*Resolver)(unsafe.Pointer(completion))
	alias_next, decoded_bytes := response_name_decode(
		completion, Response_Offset(resource_offset),
	)
	if resolver.Error != nil {
		return alias_bytes
	}
	if alias_next != Response_Encoded_Next(end_offset) {
		resolver.Error = Error_Response_Malformed
		return alias_bytes
	}
	if alias_bytes == 0 {
		next_alias = Response_Alias_Byte_Count(decoded_bytes)
		copy(
			resolver.Workspace.Alias_Name[:int(next_alias)],
			resolver.Workspace.Decoded_Name[:int(next_alias)],
		)
		return next_alias
	}
	response_alias_match(
		completion, Response_Alias_Name_Byte_Count(alias_bytes),
		Response_Name_Byte_Count(decoded_bytes),
	)
	return alias_bytes
}

func response_alias_match(
	completion *nbio.Completion, alias_bytes Response_Alias_Name_Byte_Count,
	decoded_bytes Response_Name_Byte_Count,
) {
	Response_Alias_Name_Byte_Count_Invariants(
		alias_bytes, "response_alias_match.alias_bytes",
	)
	Response_Name_Byte_Count_Invariants(
		decoded_bytes, "response_alias_match.decoded_bytes",
	)
	resolver := (*Resolver)(unsafe.Pointer(completion))
	if int(alias_bytes) != int(decoded_bytes) {
		resolver.Error = Error_Response_Malformed
		return
	}
	for index := 0; index < int(alias_bytes); index++ {
		if resolver.Workspace.Alias_Name[index] !=
			resolver.Workspace.Decoded_Name[index] {
			resolver.Error = Error_Response_Malformed
			return
		}
	}
}

func response_name_decode(
	completion *nbio.Completion, start Response_Offset,
) (next Response_Encoded_Next, decoded Response_Decoded_Byte_Count) {
	defer func() {
		Response_Encoded_Next_Invariants(next, "response_name_decode.next")
		Response_Decoded_Byte_Count_Invariants(decoded, "response_name_decode.decoded")
	}()
	Response_Offset_Invariants(start, "response_name_decode.start")
	resolver := (*Resolver)(unsafe.Pointer(completion))
	message := resolver.Workspace.Response[:resolver.Response_Bytes]
	offset := int(start)
	next_offset := 0
	decoded_bytes := 0
	pointer_hops := 0
	for step_count := 0; step_count <= DNS_NAME_DECODE_STEPS_MAXIMUM; step_count++ {
		if offset >= len(message) {
			resolver.Error = Error_Response_Malformed
			return RESPONSE_ENCODED_NEXT_MINIMUM, 0
		}
		octet := message[offset]
		if octet&DNS_COMPRESSION_TAG_MASK == DNS_COMPRESSION_TAG {
			if offset+DNS_COMPRESSION_POINTER_BYTES > len(message) {
				resolver.Error = Error_Response_Malformed
				return RESPONSE_ENCODED_NEXT_MINIMUM, 0
			}
			pointer := int(octet&byte(DNS_COMPRESSION_OFFSET_MASK>>8))<<8 |
				int(message[offset+1])
			if pointer >= offset {
				resolver.Error = Error_Response_Malformed
				return RESPONSE_ENCODED_NEXT_MINIMUM, 0
			}
			if next_offset == 0 {
				next_offset = offset + DNS_COMPRESSION_POINTER_BYTES
			}
			offset = pointer
			pointer_hops++
			if pointer_hops > DNS_NAME_COMPRESSION_HOPS_MAXIMUM {
				resolver.Error = Error_Response_Malformed
				return RESPONSE_ENCODED_NEXT_MINIMUM, 0
			}
			continue
		}
		if octet&DNS_COMPRESSION_TAG_MASK != 0 {
			resolver.Error = Error_Response_Malformed
			return RESPONSE_ENCODED_NEXT_MINIMUM, 0
		}
		if octet == 0 {
			if decoded_bytes >= DNS_NAME_WIRE_BYTES_MAXIMUM {
				resolver.Error = Error_Response_Malformed
				return RESPONSE_ENCODED_NEXT_MINIMUM, 0
			}
			resolver.Workspace.Decoded_Name[decoded_bytes] = 0
			decoded_bytes++
			if next_offset == 0 {
				next_offset = offset + DNS_NAME_ROOT_BYTES
			}
			return Response_Encoded_Next(next_offset),
				Response_Decoded_Byte_Count(decoded_bytes)
		}
		offset_value, decoded_value := response_label_decode(
			completion, Response_Label_Offset(offset),
			Response_Decoded_Prefix_Byte_Count(decoded_bytes),
		)
		if resolver.Error != nil {
			return RESPONSE_ENCODED_NEXT_MINIMUM, 0
		}
		offset = int(offset_value)
		decoded_bytes = int(decoded_value)
	}
	resolver.Error = Error_Response_Malformed
	return RESPONSE_ENCODED_NEXT_MINIMUM, 0
}

func response_label_decode(
	completion *nbio.Completion, offset Response_Label_Offset,
	decoded Response_Decoded_Prefix_Byte_Count,
) (next Response_Offset, next_decoded Response_Decoded_Prefix_Byte_Count) {
	defer func() {
		Response_Offset_Invariants(next, "response_label_decode.next")
		Response_Decoded_Prefix_Byte_Count_Invariants(
			next_decoded, "response_label_decode.next_decoded",
		)
	}()
	Response_Label_Offset_Invariants(offset, "response_label_decode.offset")
	Response_Decoded_Prefix_Byte_Count_Invariants(
		decoded, "response_label_decode.decoded",
	)
	resolver := (*Resolver)(unsafe.Pointer(completion))
	message := resolver.Workspace.Response[:resolver.Response_Bytes]
	label_bytes := int(message[offset])
	if label_bytes > DNS_LABEL_BYTES_MAXIMUM {
		resolver.Error = Error_Response_Malformed
		return RESPONSE_OFFSET_MINIMUM, 0
	}
	if int(offset)+DNS_LABEL_SIZE_BYTES+label_bytes > len(message) {
		resolver.Error = Error_Response_Malformed
		return RESPONSE_OFFSET_MINIMUM, 0
	}
	if int(decoded)+DNS_LABEL_SIZE_BYTES+label_bytes > DNS_NAME_WIRE_BYTES_MAXIMUM {
		resolver.Error = Error_Response_Malformed
		return RESPONSE_OFFSET_MINIMUM, 0
	}
	resolver.Workspace.Decoded_Name[decoded] = byte(label_bytes)
	source_offset := int(offset) + DNS_LABEL_SIZE_BYTES
	target_offset := int(decoded) + DNS_LABEL_SIZE_BYTES
	source := message[source_offset : source_offset+label_bytes]
	target := resolver.Workspace.Decoded_Name[target_offset : target_offset+label_bytes]
	for label_index, value := range source {
		if value >= 'A' {
			if value <= 'Z' {
				value += 'a' - 'A'
			}
		}
		target[label_index] = value
	}
	return Response_Offset(source_offset + label_bytes),
		Response_Decoded_Prefix_Byte_Count(target_offset + label_bytes)
}

func response_decoded_match(
	completion *nbio.Completion, decoded_bytes Response_Name_Byte_Count,
	canonical_bytes Response_Canonical_Byte_Count,
) (same bytes.Boolean) {
	defer func() { bytes.Boolean_Invariants(same, "response_decoded_match.same") }()
	Response_Name_Byte_Count_Invariants(
		decoded_bytes, "response_decoded_match.decoded_bytes",
	)
	Response_Canonical_Byte_Count_Invariants(
		canonical_bytes, "response_decoded_match.canonical_bytes",
	)
	resolver := (*Resolver)(unsafe.Pointer(completion))
	if int(decoded_bytes) != int(canonical_bytes) {
		return false
	}
	for index := 0; index < int(canonical_bytes); index++ {
		if resolver.Workspace.Decoded_Name[index] !=
			resolver.Workspace.Canonical_Name[index] {
			return false
		}
	}
	return true
}

func response_alias_is_canonical(
	completion *nbio.Completion, alias_bytes Response_Alias_Name_Byte_Count,
	canonical_bytes Response_Canonical_Byte_Count,
) (same bytes.Boolean) {
	defer func() {
		bytes.Boolean_Invariants(same, "response_alias_is_canonical.same")
	}()
	Response_Alias_Name_Byte_Count_Invariants(
		alias_bytes, "response_alias_is_canonical.alias_bytes",
	)
	Response_Canonical_Byte_Count_Invariants(
		canonical_bytes, "response_alias_is_canonical.canonical_bytes",
	)
	resolver := (*Resolver)(unsafe.Pointer(completion))
	if int(alias_bytes) != int(canonical_bytes) {
		return false
	}
	for index := 0; index < int(canonical_bytes); index++ {
		if resolver.Workspace.Alias_Name[index] !=
			resolver.Workspace.Canonical_Name[index] {
			return false
		}
	}
	return true
}

func response_address_append(
	completion *nbio.Completion, resource_offset Response_Address_Offset,
) {
	Response_Address_Offset_Invariants(
		resource_offset, "response_address_append.resource_offset",
	)
	resolver := (*Resolver)(unsafe.Pointer(completion))
	message := resolver.Workspace.Response[:resolver.Response_Bytes]
	expected_size := nbio.IPV4_ADDRESS_BYTES
	if Record_Type(resolver.Record_Type) == RECORD_TYPE_AAAA {
		expected_size = nbio.IPV6_ADDRESS_BYTES
	}
	resource_end := int(resource_offset) + expected_size
	address := nbio.Address{Port: uint16(resolver.Port)}
	if Record_Type(resolver.Record_Type) == RECORD_TYPE_A {
		address.Family = nbio.FAMILY_IPV4
	} else {
		address.Family = nbio.FAMILY_IPV6
	}
	copy(address.IP[:expected_size], message[resource_offset:resource_end])
	for index := 0; index < int(resolver.Result_Count); index++ {
		if resolver.Results[index] == address {
			return
		}
	}
	if int(resolver.Result_Count) == len(resolver.Results) {
		resolver.Error = Error_Result_Too_Small
		return
	}
	resolver.Results[resolver.Result_Count] = address
	resolver.Result_Count++
}
