package x509

import (
	"testing"

	"local/james-orcales/shared/crypto/ecdsa"
	"local/james-orcales/shared/crypto/elliptic"
	"local/james-orcales/shared/crypto/rsa"
	"local/james-orcales/shared/crypto/sha256"
	"local/james-orcales/shared/encoding/asn1"
	"local/james-orcales/shared/encoding/binary"
	"local/james-orcales/shared/math/bits"
)

// Test_Internal_Parser_Invariant_Domains reaches states forbidden by valid DER.
func Test_Internal_Parser_Invariant_Domains(_ *testing.T) {
	var storage_bytes [ENCODED_SIZE_MAXIMUM]byte
	storage := Storage(storage_bytes[:])
	positions := [...]int{
		ENCODED_SIZE_MINIMUM,
		binary.UINT_8_SIZE,
		binary.UINT_16_SIZE,
		ENCODED_SIZE_MAXIMUM,
	}
	words := [...]uint64{
		bits.WORD_64_MINIMUM,
		binary.UINT_8_SIZE,
		binary.UINT_16_SIZE,
		bits.WORD_64_MAXIMUM,
	}
	algorithms := [...]Public_Key_Algorithm{
		PUBLIC_KEY_ALGORITHM_RSA,
		PUBLIC_KEY_ALGORITHM_ECDSA,
		PUBLIC_KEY_ALGORITHM_ED25519,
		PUBLIC_KEY_ALGORITHM_RSA,
	}
	for index, position := range positions {
		span := Span{Start: Span_Start(position), End: Span_End(position)}
		identifier := identifier_edge(index)
		certificate := certificate_edge(words[index])
		algorithm := algorithms[index]
		parsed := parsed_certificate_edge(words[index], position, algorithm)
		parsed_public_key := parsed_public_key_edge(words[index], algorithm)

		Parse_Certificate(&certificate, nil)
		ignore_panic(func() { Verify_Signature_From(&certificate, &certificate) })

		parse(&parsed, storage, span)
		parse_certificate_content(&parsed, storage, span)
		parse_tbs(&parsed, storage, span, span)
		parse_tbs_fields(&parsed, storage, span, span)
		parse_tbs_subject(&parsed, storage, span)
		parse_public_key(&parsed_public_key, storage, span)
		public_key_parse(&parsed_public_key, storage, span, span)
		rsa_public_key_parse(&parsed_public_key.RSA_Key, storage, span)
		rsa_exponent_valid(storage, span)
		common_name := span
		parse_name(&common_name, storage, span)
		name_common_name(&common_name, storage, span)
		name_set_valid(storage, identifier, span)
		signature_algorithm_parse(storage, span)
		element_identifier := identifier
		element_content := span
		element_encoded := span
		element_tail := span
		take_element(
			&element_identifier,
			&element_content,
			&element_encoded,
			&element_tail,
			storage,
			span,
		)
	}
}

// Test_Internal_Value_Invariant_Domains reaches hostile scalar and identifier edges.
func Test_Internal_Value_Invariant_Domains(_ *testing.T) {
	var storage_bytes [ENCODED_SIZE_MAXIMUM]byte
	storage := Storage(storage_bytes[:])
	var signature_bytes [ecdsa.SIGNATURE_SIZE]byte
	signature := ecdsa.Signature(signature_bytes[:])
	var scalar_bytes [ecdsa.SCALAR_SIZE]byte
	scalar := ecdsa.Scalar_Encoding(scalar_bytes[:])
	var digest_bytes [sha256.DIGEST_256_SIZE]byte
	digest := Digest_Destination(digest_bytes[:])
	positions := [...]int{
		ENCODED_SIZE_MINIMUM,
		binary.UINT_8_SIZE,
		binary.UINT_16_SIZE,
		ENCODED_SIZE_MAXIMUM,
	}
	for index, position := range positions {
		span := Span{Start: Span_Start(position), End: Span_End(position)}
		identifier := identifier_edge(index)
		identifier_matches(identifier, identifier)
		expected := identifier
		sequence_identifier(&expected)
		expected = identifier
		set_identifier_expected(&expected)
		expected = identifier
		oid_identifier_expected(&expected)
		expected = identifier
		bit_string_identifier(&expected)
		expected = identifier
		integer_identifier(&expected)
		value := span
		bit_string_content(&value, storage, identifier, span)
		version_valid(storage, identifier, span)
		serial_valid(storage, identifier, span)
		positive_integer_valid(storage, identifier, span)
		validity_valid(storage, identifier, span)
		time_element_valid(identifier, span)
		optional_fields_valid(storage, span)
		digest_sha_256(digest, sha256.Source(storage[:position]))
		ecdsa_signature_parse(signature, storage, span)
		integer_scalar(scalar, storage, identifier, span)
	}
	span := Span{
		Start: Span_Start(ENCODED_SIZE_MINIMUM),
		End:   Span_End(binary.UINT_8_SIZE),
	}
	optional_fields_valid(storage, span)
}

// Test_Internal_Ready_Invariant_Domains reaches complete keys before parse replacement.
func Test_Internal_Ready_Invariant_Domains(_ *testing.T) {
	var storage_bytes [ENCODED_SIZE_MAXIMUM]byte
	storage := Storage(storage_bytes[:])
	var generator elliptic.Point
	elliptic.Point_Generator(&generator)
	parsed := parsed_certificate_edge(
		bits.WORD_64_MINIMUM, ENCODED_SIZE_MINIMUM, PUBLIC_KEY_ALGORITHM_ECDSA,
	)
	parsed.ECDSA_Public_Key = ecdsa.Public_Key{Point: generator, Ready: ecdsa.READY_COMPLETE}
	parsed_public_key := parsed_public_key_edge(
		bits.WORD_64_MINIMUM, PUBLIC_KEY_ALGORITHM_ECDSA,
	)
	parsed_public_key.ECDSA_Key = ecdsa.Public_Key{
		Point: generator, Ready: ecdsa.READY_COMPLETE,
	}
	empty := Span{}
	parse(&parsed, storage, empty)
	parse_certificate_content(&parsed, storage, empty)
	parse_tbs(&parsed, storage, empty, empty)
	parse_tbs_fields(&parsed, storage, empty, empty)
	parse_tbs_subject(&parsed, storage, empty)
	parse_public_key(&parsed_public_key, storage, empty)
	public_key_parse(&parsed_public_key, storage, empty, empty)
	parsed = parsed_certificate_edge(
		bits.WORD_64_MINIMUM, ENCODED_SIZE_MINIMUM, PUBLIC_KEY_ALGORITHM_RSA,
	)
	parsed.RSA_Public_Key.Ready = rsa.READY_COMPLETE
	parsed_public_key = parsed_public_key_edge(
		bits.WORD_64_MINIMUM, PUBLIC_KEY_ALGORITHM_RSA,
	)
	parsed_public_key.RSA_Key.Ready = rsa.READY_COMPLETE
	ignore_panic(func() { parse(&parsed, storage, empty) })
	ignore_panic(func() { parse_certificate_content(&parsed, storage, empty) })
	ignore_panic(func() { parse_tbs(&parsed, storage, empty, empty) })
	ignore_panic(func() { parse_tbs_fields(&parsed, storage, empty, empty) })
	ignore_panic(func() { parse_tbs_subject(&parsed, storage, empty) })
	ignore_panic(func() { parse_public_key(&parsed_public_key, storage, empty) })
	ignore_panic(func() { public_key_parse(&parsed_public_key, storage, empty, empty) })
	ignore_panic(func() { rsa_public_key_parse(&parsed_public_key.RSA_Key, storage, empty) })
}

func identifier_edge(index int) (result Identifier) {
	classes := [...]uint32{
		asn1.CLASS_UNIVERSAL,
		asn1.CLASS_APPLICATION,
		asn1.CLASS_CONTEXT_SPECIFIC,
		asn1.CLASS_PRIVATE,
	}
	tags := [...]uint32{
		bits.WORD_32_MINIMUM,
		binary.UINT_8_SIZE,
		binary.UINT_16_SIZE,
		asn1.TAG_MAXIMUM,
	}
	result = Identifier{
		Class:       Identifier_Class(classes[index]),
		Tag:         Identifier_Tag(tags[index]),
		Constructed: Constructed(index & binary.UINT_8_SIZE),
	}
	return result
}

func certificate_edge(word uint64) (result Certificate) {
	result = Certificate{
		Signature_Algorithm:  SIGNATURE_ALGORITHM_RSA_SHA_256,
		Public_Key_Algorithm: PUBLIC_KEY_ALGORITHM_RSA,
		RSA_Public_Key: rsa.Public_Key{
			Modulus: rsa_modulus_filled(word),
		},
		ECDSA_Public_Key: ecdsa.Public_Key{
			Point: ecdsa_point_filled(word),
		},
	}
	return result
}

func parsed_certificate_edge(
	word uint64, position int, algorithm Public_Key_Algorithm,
) (result Parsed_Certificate) {
	span := Span{Start: Span_Start(position), End: Span_End(position)}
	result = Parsed_Certificate{
		TBS:                  TBS_Span(span),
		Signature:            Signature_Span(span),
		Issuer_Raw:           Issuer_Raw_Span(span),
		Issuer_Common_Name:   Issuer_Common_Name_Span(span),
		Subject_Raw:          Subject_Raw_Span(span),
		Subject_Common_Name:  Subject_Common_Name_Span(span),
		Public_Key_Algorithm: algorithm,
		RSA_Public_Key: rsa.Public_Key{
			Modulus: rsa_modulus_filled(word),
		},
		ECDSA_Public_Key: ecdsa.Public_Key{
			Point: ecdsa_point_filled(word),
		},
	}
	return result
}

func parsed_public_key_edge(
	word uint64, algorithm Public_Key_Algorithm,
) (result Parsed_Public_Key) {
	result = Parsed_Public_Key{
		Algorithm: algorithm,
		RSA_Key: rsa.Public_Key{
			Modulus: rsa_modulus_filled(word),
		},
		ECDSA_Key: ecdsa.Public_Key{
			Point: ecdsa_point_filled(word),
		},
	}
	return result
}

func ecdsa_point_filled(word uint64) (result elliptic.Point) {
	result = elliptic.Point{
		X: elliptic.X_Coordinate{
			Limb_0: elliptic.X_Limb_0(word),
			Limb_1: elliptic.X_Limb_1(word),
			Limb_2: elliptic.X_Limb_2(word),
			Limb_3: elliptic.X_Limb_3(word),
		},
		Y: elliptic.Y_Coordinate{
			Limb_0: elliptic.Y_Limb_0(word),
			Limb_1: elliptic.Y_Limb_1(word),
			Limb_2: elliptic.Y_Limb_2(word),
			Limb_3: elliptic.Y_Limb_3(word),
		},
		Z: elliptic.Z_Coordinate{
			Limb_0: elliptic.Z_Limb_0(word),
			Limb_1: elliptic.Z_Limb_1(word),
			Limb_2: elliptic.Z_Limb_2(word),
			Limb_3: elliptic.Z_Limb_3(word),
		},
	}
	return result
}

func rsa_modulus_filled(word uint64) (result rsa.Modulus) {
	chunk := rsa.Integer_Chunk_Storage{
		Lane_0: rsa.Integer_Lane_0(word),
		Lane_1: rsa.Integer_Lane_1(word),
		Lane_2: rsa.Integer_Lane_2(word),
		Lane_3: rsa.Integer_Lane_3(word),
	}
	result = rsa.Modulus{
		Chunk_0: rsa.Integer_Chunk_0(chunk),
		Chunk_1: rsa.Integer_Chunk_1(chunk),
		Chunk_2: rsa.Integer_Chunk_2(chunk),
		Chunk_3: rsa.Integer_Chunk_3(chunk),
		Chunk_4: rsa.Integer_Chunk_4(chunk),
		Chunk_5: rsa.Integer_Chunk_5(chunk),
		Chunk_6: rsa.Integer_Chunk_6(chunk),
		Chunk_7: rsa.Integer_Chunk_7(chunk),
	}
	return result
}

func ignore_panic(operation func()) {
	defer func() { recover() }()
	operation()
}
