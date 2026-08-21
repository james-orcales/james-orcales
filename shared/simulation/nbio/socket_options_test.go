package nbio

import (
	"testing"

	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/simulation/time"
	"local/james-orcales/shared/testify"
)

// Lock the composition because copied decimal defaults hide the limits that produce them.
func Test_Socket_Option_Default_Formulas(t *testing.T) {
	testify.Equal(t, uint32(128*bits.KIBIBYTE_BYTES), TCP_RECEIVE_BUFFER_BYTES_DEFAULT)
	testify.Equal(t, uint32(128*bits.KIBIBYTE_BYTES), TCP_SEND_BUFFER_BYTES_DEFAULT)
	testify.Equal(t, IPV6_SOCKET_ADDRESS_BITS/bits.BIT_COUNT_8_MAXIMUM,
		IPV6_SOCKET_ADDRESS_BYTES)
	testify.Equal(t, bits.KIBIBYTE_BYTES+IPV6_SOCKET_ADDRESS_BYTES,
		UDP_RECEIVE_RECORD_BYTES)
	record_count := UDP_RECEIVE_BUFFER_SCALE_DEFAULT *
		((UDP_RECEIVE_BUFFER_BASELINE_BYTES + UDP_RECEIVE_RECORD_BYTES - 1) /
			UDP_RECEIVE_RECORD_BYTES)
	testify.Equal(t, record_count, UDP_RECEIVE_RECORD_COUNT_DEFAULT)
	testify.Equal(t, UDP_RECEIVE_RECORD_COUNT_DEFAULT*UDP_RECEIVE_RECORD_BYTES,
		UDP_RECEIVE_BUFFER_BYTES_DEFAULT)
	testify.Equal(t, UDP_SEND_BUFFER_KIBIBYTES_DEFAULT*bits.KIBIBYTE_BYTES,
		UDP_SEND_BUFFER_BYTES_DEFAULT)
	testify.Equal(t, IPV4_REASSEMBLY_BYTES_MINIMUM-IPV4_HEADER_BYTES_MINIMUM-
		TCP_HEADER_BYTES_MINIMUM, TCP_MAXIMUM_SEGMENT_BYTES_DEFAULT)
	testify.Equal(t, uint32(bits.KIBIBYTE_BYTES), TCP_NOT_SENT_LOW_WATER_BYTES_DEFAULT)
	testify.Equal(t, 2*time.HOUR, TCP_KEEPALIVE_IDLE_DEFAULT)
	testify.Equal(t, 75*time.SECOND, TCP_KEEPALIVE_INTERVAL_DEFAULT)
	testify.Equal(t, uint32(9), TCP_KEEPALIVE_PROBE_COUNT_DEFAULT)
	testify.Equal(t, uint32(1), SOCKET_RECEIVE_LOW_WATER_BYTES_DEFAULT)
	testify.Equal(t, time.SECOND, SOCKET_LINGER_TIMEOUT_DEFAULT)
	testify.True(t, TCP_NO_DELAY_DEFAULT)
}

// Lock bit-width formulas because decimal maxima conceal signed and unsigned option domains.
func Test_Socket_Option_Maximum_Formulas(t *testing.T) {
	testify.Equal(t, uint32(bits.INTEGER_32_MAXIMUM), SOCKET_OPTION_INTEGER_MAXIMUM)
	testify.Equal(t, uint32(bits.WORD_16_MAXIMUM), TCP_MAXIMUM_SEGMENT_BYTES_MAXIMUM)
	testify.Equal(t, uint32(bits.INTEGER_8_MAXIMUM),
		TCP_KEEPALIVE_PROBE_COUNT_MAXIMUM)
}
