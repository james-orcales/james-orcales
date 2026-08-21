package x509_test

import (
	standard_x509 "crypto/x509"
	"testing"

	"local/james-orcales/shared/crypto/x509"
	"local/james-orcales/shared/encoding/binary"
	"local/james-orcales/shared/encoding/hex"
	"local/james-orcales/shared/testify"
)

// Test_Certificate parses every supported interoperable public-key certificate.
func Test_Certificate(t *testing.T) {
	encoded := rsa_certificate(t)
	assert_certificate(t, encoded, x509.PUBLIC_KEY_ALGORITHM_RSA)
	encoded = ecdsa_certificate(t)
	assert_certificate(t, encoded, x509.PUBLIC_KEY_ALGORITHM_ECDSA)
	encoded = ed25519_certificate(t)
	assert_certificate(t, encoded, x509.PUBLIC_KEY_ALGORITHM_ED25519)
}

// Test_Verification accepts each supported signature and rejects changed bytes.
func Test_Verification(t *testing.T) {
	assert_verification(t, rsa_certificate(t))
	assert_verification(t, ecdsa_certificate(t))
	assert_verification(t, ed25519_certificate(t))
}

// Test_Bounds rejects oversized and bounded malformed input transactionally.
func Test_Bounds(t *testing.T) {
	var certificate x509.Certificate
	var oversized [x509.ENCODED_SIZE_MAXIMUM + binary.UINT_8_SIZE]byte
	testify.Panics(t, func() {
		x509.Parse_Certificate(&certificate, oversized[:])
	})
	before := certificate
	status := x509.Parse_Certificate(&certificate, nil)
	testify.Equal(t, x509.PARSE_STATUS_INPUT_INVALID, status)
	testify.Equal(t, before, certificate)
}

// Test_Allocation measures successful parsing and verification.
func Test_Allocation(t *testing.T) {
	for _, encoded := range [...]x509.Encoded{
		rsa_certificate(t), ecdsa_certificate(t), ed25519_certificate(t),
	} {
		var certificate x509.Certificate
		var parse_status x509.Parse_Status
		var verified x509.Verification
		testify.Zero_Allocation(t, func() {
			parse_status = x509.Parse_Certificate(&certificate, encoded)
		})
		testify.Zero_Allocation(t, func() {
			verified = x509.Verify_Signature_From(&certificate, &certificate)
		})
		testify.Equal(t, x509.PARSE_STATUS_OK, parse_status)
		testify.True(t, bool(verified))
	}
}

// Test_Invariant_Domains reaches all bounded source lengths and both outcomes.
func Test_Invariant_Domains(t *testing.T) {
	var certificate x509.Certificate
	var input [x509.ENCODED_SIZE_MAXIMUM]byte
	boundary_sizes := [...]int{
		x509.ENCODED_SIZE_MINIMUM,
		x509.ENCODED_SIZE_MINIMUM + binary.UINT_8_SIZE,
		x509.ENCODED_SIZE_MINIMUM + binary.UINT_16_SIZE,
		x509.ENCODED_SIZE_MAXIMUM - binary.UINT_8_SIZE,
		x509.ENCODED_SIZE_MAXIMUM,
	}
	test_parse_invariant_domains(&certificate, &input, boundary_sizes[:])
	test_verify_invariant_domains(t, &certificate, &input, boundary_sizes[:])
}

func test_parse_invariant_domains(
	certificate *x509.Certificate,
	input *[x509.ENCODED_SIZE_MAXIMUM]byte,
	boundary_sizes []int,
) {
	for _, size := range boundary_sizes {
		x509.Parse_Certificate(certificate, input[:size])
	}
	for _, size := range boundary_sizes {
		certificate.Raw = input[:size]
		certificate.TBS = input[:size]
		certificate.Issuer.Raw = input[:size]
		certificate.Subject.Raw = input[:size]
		x509.Parse_Certificate(certificate, nil)
	}
	for _, size := range [...]int{
		x509.ENCODED_SIZE_MINIMUM,
		x509.ENCODED_SIZE_MINIMUM + binary.UINT_8_SIZE,
		x509.ENCODED_SIZE_MINIMUM + binary.UINT_16_SIZE,
		x509.SIGNATURE_SIZE_MAXIMUM - binary.UINT_8_SIZE,
		x509.SIGNATURE_SIZE_MAXIMUM,
	} {
		certificate.Signature = input[:size]
		x509.Parse_Certificate(certificate, nil)
	}
	for _, size := range [...]int{
		x509.ENCODED_SIZE_MINIMUM,
		x509.ENCODED_SIZE_MINIMUM + binary.UINT_8_SIZE,
		x509.ENCODED_SIZE_MINIMUM + binary.UINT_16_SIZE,
		x509.COMMON_NAME_SIZE_MAXIMUM - binary.UINT_8_SIZE,
		x509.COMMON_NAME_SIZE_MAXIMUM,
	} {
		certificate.Issuer.Common_Name = input[:size]
		certificate.Subject.Common_Name = input[:size]
		x509.Parse_Certificate(certificate, nil)
	}
	maximum_algorithm := x509.PUBLIC_KEY_ALGORITHM_ED25519
	for algorithm := x509.PUBLIC_KEY_ALGORITHM_RSA; ; algorithm++ {
		if algorithm > maximum_algorithm {
			break
		}
		certificate.Public_Key_Algorithm = algorithm
		certificate.Signature_Algorithm = x509.Signature_Algorithm(algorithm)
		x509.Parse_Certificate(certificate, nil)
	}
}

func test_verify_invariant_domains(
	t *testing.T,
	certificate *x509.Certificate,
	input *[x509.ENCODED_SIZE_MAXIMUM]byte,
	boundary_sizes []int,
) {
	encoded := rsa_certificate(t)
	status := x509.Parse_Certificate(certificate, encoded)
	testify.Equal(t, x509.PARSE_STATUS_OK, status)
	for _, size := range boundary_sizes {
		certificate.Raw = input[:size]
		certificate.TBS = input[:size]
		certificate.Issuer.Raw = input[:size]
		certificate.Subject.Raw = input[:size]
		x509.Verify_Signature_From(certificate, certificate)
	}
	for _, size := range [...]int{
		x509.ENCODED_SIZE_MINIMUM,
		x509.ENCODED_SIZE_MINIMUM + binary.UINT_8_SIZE,
		x509.ENCODED_SIZE_MINIMUM + binary.UINT_16_SIZE,
		x509.SIGNATURE_SIZE_MAXIMUM - binary.UINT_8_SIZE,
		x509.SIGNATURE_SIZE_MAXIMUM,
	} {
		certificate.Signature = input[:size]
		x509.Verify_Signature_From(certificate, certificate)
	}
	for _, size := range [...]int{
		x509.ENCODED_SIZE_MINIMUM,
		x509.ENCODED_SIZE_MINIMUM + binary.UINT_8_SIZE,
		x509.ENCODED_SIZE_MINIMUM + binary.UINT_16_SIZE,
		x509.COMMON_NAME_SIZE_MAXIMUM - binary.UINT_8_SIZE,
		x509.COMMON_NAME_SIZE_MAXIMUM,
	} {
		certificate.Issuer.Common_Name = input[:size]
		certificate.Subject.Common_Name = input[:size]
		x509.Verify_Signature_From(certificate, certificate)
	}
}

func assert_certificate(
	t *testing.T, encoded []byte, public_key_algorithm x509.Public_Key_Algorithm,
) {
	var certificate x509.Certificate
	status := x509.Parse_Certificate(&certificate, encoded)
	testify.Equal(t, x509.PARSE_STATUS_OK, status)
	testify.Equal(t, public_key_algorithm, certificate.Public_Key_Algorithm)
	testify.Equal(t, "example", string(certificate.Subject.Common_Name))
}

func assert_verification(t *testing.T, encoded []byte) {
	var certificate x509.Certificate
	status := x509.Parse_Certificate(&certificate, encoded)
	testify.Equal(t, x509.PARSE_STATUS_OK, status)
	testify.True(t, bool(x509.Verify_Signature_From(&certificate, &certificate)))
	standard_certificate, err := standard_x509.ParseCertificate(encoded)
	testify.No_Error(t, err)
	err = standard_certificate.CheckSignature(
		standard_certificate.SignatureAlgorithm,
		standard_certificate.RawTBSCertificate,
		standard_certificate.Signature,
	)
	testify.No_Error(t, err)
	encoded[len(encoded)-binary.UINT_8_SIZE] ^= byte(binary.UINT_8_SIZE)
	testify.False(t, bool(x509.Verify_Signature_From(&certificate, &certificate)))
}

func rsa_certificate(t *testing.T) (encoded []byte) {
	return certificate_decode(t, RSA_CERTIFICATE_HEX)
}

func ecdsa_certificate(t *testing.T) (encoded []byte) {
	return certificate_decode(t, ECDSA_CERTIFICATE_HEX)
}

func ed25519_certificate(t *testing.T) (encoded []byte) {
	return certificate_decode(t, ED25519_CERTIFICATE_HEX)
}

func certificate_decode(t *testing.T, encoding string) (certificate []byte) {
	certificate = make([]byte, len(encoding)/hex.ENCODED_BYTE_SIZE)
	count, status := hex.Decode_Into(certificate, []byte(encoding))
	testify.Equal(t, hex.Decode_Status(hex.STATUS_OK), status)
	testify.Equal(t, len(certificate), int(count))
	return certificate
}

const RSA_CERTIFICATE_HEX = "308202e1308201c9a003020102020101300d06092a864886f70d01010b05003012" +
	"3110300e060355040313076578616d706c65301e170d3730303130313030303030" +
	"305a170d3730303130313031303030305a30123110300e06035504031307657861" +
	"6d706c6530820122300d06092a864886f70d01010105000382010f003082010a02" +
	"82010100a061042b12e3ac1d111bc5234fdc3ce38c2709abde88ed397bfa4de4" +
	"647642e75645b857e4a97f9ad9dda2c7e35d901f402cd3990cdc81068b72d477" +
	"3a20b8d3e3fe0bf976d08082a28ba5ecd7348ecef2637eee7779c4994d227f1b" +
	"1551497bd459d7bfc3281609f4f69152caca9db06d7107a63683243c6b0c3bc4" +
	"ead1cda88dc9a731cd0abe72015eb9fca7216ac0927bfe0d68b2566748deb8be" +
	"44c5ae628709cce2fe118a13b1a5c37ac65e3bb98ba3821f397f26e793e1dc29" +
	"1d501e1b81fa82d673bff9bd0cb8d7639a6b03c40ea9ff45e41f41d7dbedbb4" +
	"61de9365fd560457a64c595dbe1e3b7b9a5e251421d96ab4e798c4bb7c41e73b" +
	"3ade476d30203010001a3423040300e0603551d0f0101ff040403020780300f06" +
	"03551d130101ff040530030101ff301d0603551d0e04160414c644355c1747fd" +
	"fe439497dff2f55de1c1d45d82300d06092a864886f70d01010b050003820101" +
	"00603b49038ccb13fca1cee54e331175e356f322ce677872ae5f3bb9b8680ac0" +
	"f2add02bb58fb3d5cf3b9c8f5611c9c298dbf807d7c86b06029e0dd5fe80d931" +
	"e0ebef7f7e484204fb0d1bea69cc5707fd5c951241552c993e1f9dd673affac07" +
	"03bb1e504ebd8dd394c786992a41f013c6d12f095c45e1b8b8e84a3afc9c28b" +
	"31542448792357e7f8c4fc558826b2d7ef4019308da341274655dd187fea0c32e" +
	"b476cb602849c3181660bc065e712fd0962347646b55f1042e6ee2bb60e1f303f" +
	"57f2e2fd46ff0ce61672b9e1353d76476c84d6ad4a9c36b271db8cb01b84ed7" +
	"7dd88068b31f8f9eaacb87d2251bb0993b63f17a8da9a592b9c4358d41e021e4f"

const ECDSA_CERTIFICATE_HEX = "308201543081fba003020102020101300a06082a8648ce3d04030230123110300e" +
	"060355040313076578616d706c65301e170d3730303130313030303030305a170d" +
	"3730303130313031303030305a30123110300e060355040313076578616d706c65" +
	"3059301306072a8648ce3d020106082a8648ce3d030107034200044b51fcd18274" +
	"a0c35f3b477f88bfda3fe6f5be6e8766d3998955bdddd104e6a16eec02a114c3" +
	"9ac9fa750043348f49a7e3cf09b35f19477f9291e357c465cf9ba3423040300e06" +
	"03551d0f0101ff040403020780300f0603551d130101ff040530030101ff301d06" +
	"03551d0e0416041454442f94d969b4cbe2a305ad40b60dfc807492c6300a06082a" +
	"8648ce3d0403020348003045022074ab4c5d18ca81f8db4244073e43dc9c9004" +
	"a6a7c4116e930854bde948db6d1c022100daf4d73c2c1a79e48be1927d1491c2" +
	"6466546a896e02b3c0258b2b919824c593"

const ED25519_CERTIFICATE_HEX = "308201143081c7a003020102020101300506032b657030123110300e060355" +
	"040313076578616d706c65301e170d3730303130313030303030305a170d3730303130" +
	"313031303030305a30123110300e060355040313076578616d706c65302a300506" +
	"032b6570032100dadbd184a2d526f1ebdd5c06fdad9359b228759b4d7f79d666" +
	"89fa254aad8546a3423040300e0603551d0f0101ff040403020780300f0603551d" +
	"130101ff040530030101ff301d0603551d0e04160414db0743e2dcba9ebf2419bd" +
	"e0881beea966689a26300506032b65700341004518f1e6c1da31eface07aa57489" +
	"fb4b7b8508733eef0524694efd84dd985a10f7cad4066dcc680e08cae572cb610" +
	"8d4e75347794957ba34a60118f070021a09"
