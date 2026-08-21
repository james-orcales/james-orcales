package nbio_test

import (
	"testing"

	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/sim/aver/default"
	"local/james-orcales/shared/sim/nbio"
	"local/james-orcales/shared/sim/time"
	"local/james-orcales/shared/testify"
)

// TestMain register completion-machine assertions. Thus unused legal edge fail suite.
func TestMain(m *testing.M) {
	aver.Run_Test_Main(m)
}

// Lock the composition because copied decimal defaults hide the limits that produce them.
func Test_Socket_Option_Default_Formulas(t *testing.T) {
	testify.Equal(t, uint32(128*bits.KIBIBYTE_BYTES), nbio.TCP_RECEIVE_BUFFER_BYTES_DEFAULT)
	testify.Equal(t, uint32(128*bits.KIBIBYTE_BYTES), nbio.TCP_SEND_BUFFER_BYTES_DEFAULT)
	testify.Equal(t, nbio.IPV6_SOCKET_ADDRESS_BITS/bits.BIT_COUNT_8_MAXIMUM,
		nbio.IPV6_SOCKET_ADDRESS_BYTES)
	testify.Equal(t, bits.KIBIBYTE_BYTES+nbio.IPV6_SOCKET_ADDRESS_BYTES,
		nbio.UDP_RECEIVE_RECORD_BYTES)
	record_count := nbio.UDP_RECEIVE_BUFFER_SCALE_DEFAULT *
		((nbio.UDP_RECEIVE_BUFFER_BASELINE_BYTES + nbio.UDP_RECEIVE_RECORD_BYTES - 1) /
			nbio.UDP_RECEIVE_RECORD_BYTES)
	testify.Equal(t, record_count, nbio.UDP_RECEIVE_RECORD_COUNT_DEFAULT)
	testify.Equal(t, nbio.UDP_RECEIVE_RECORD_COUNT_DEFAULT*nbio.UDP_RECEIVE_RECORD_BYTES,
		nbio.UDP_RECEIVE_BUFFER_BYTES_DEFAULT)
	testify.Equal(t, nbio.UDP_SEND_BUFFER_KIBIBYTES_DEFAULT*bits.KIBIBYTE_BYTES,
		nbio.UDP_SEND_BUFFER_BYTES_DEFAULT)
	testify.Equal(t, nbio.IPV4_REASSEMBLY_BYTES_MINIMUM-nbio.IPV4_HEADER_BYTES_MINIMUM-
		nbio.TCP_HEADER_BYTES_MINIMUM, nbio.TCP_MAXIMUM_SEGMENT_BYTES_DEFAULT)
	testify.Equal(t, uint32(bits.KIBIBYTE_BYTES), nbio.TCP_NOT_SENT_LOW_WATER_BYTES_DEFAULT)
	testify.Equal(t, 2*time.HOUR, nbio.TCP_KEEPALIVE_IDLE_DEFAULT)
	testify.Equal(t, 75*time.SECOND, nbio.TCP_KEEPALIVE_INTERVAL_DEFAULT)
	testify.Equal(t, uint32(9), nbio.TCP_KEEPALIVE_PROBE_COUNT_DEFAULT)
	testify.Equal(t, uint32(1), nbio.SOCKET_RECEIVE_LOW_WATER_BYTES_DEFAULT)
	testify.Equal(t, time.SECOND, nbio.SOCKET_LINGER_TIMEOUT_DEFAULT)
	testify.True(t, nbio.TCP_NO_DELAY_DEFAULT)
}

// Lock bit-width formulas because decimal maxima conceal signed and unsigned option domains.
func Test_Socket_Option_Maximum_Formulas(t *testing.T) {
	testify.Equal(t, uint32(bits.INTEGER_32_MAXIMUM), nbio.SOCKET_OPTION_INTEGER_MAXIMUM)
	testify.Equal(t, uint32(bits.WORD_16_MAXIMUM), nbio.TCP_MAXIMUM_SEGMENT_BYTES_MAXIMUM)
	testify.Equal(t, uint32(bits.INTEGER_8_MAXIMUM),
		nbio.TCP_KEEPALIVE_PROBE_COUNT_MAXIMUM)
}
