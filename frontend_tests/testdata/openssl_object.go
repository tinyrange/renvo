package openssltest

// These declarations intentionally use several C ABI shapes: opaque pointer
// handles, byte pointers, mutable out-parameters, pointer returns, and both
// machine-word and fixed-width scalar values. The library label documents
// ownership for humans; an ELF relocatable object records undefined symbols
// for the system linker to resolve.

// renvo:linkstatic crypto,OpenSSL_version_num
func opensslVersionNumber() uint64 { return 0 }

// renvo:linkstatic crypto,OpenSSL_version
func opensslVersion(kind int32) uintptr { return 0 }

// renvo:linkstatic crypto,SHA256
func opensslSHA256(data *byte, size uintptr, output *byte) *byte { return nil }

// renvo:linkstatic crypto,CRYPTO_memcmp
func opensslMemcmp(left *byte, right *byte, size uintptr) int32 { return 0 }

// renvo:linkstatic crypto,RAND_bytes
func opensslRandom(output *byte, size int32) int32 { return 0 }

// renvo:linkstatic crypto,EVP_MD_CTX_new
func opensslDigestContextNew() uintptr { return 0 }

// renvo:linkstatic crypto,EVP_MD_CTX_free
func opensslDigestContextFree(context uintptr) {}

// renvo:linkstatic crypto,EVP_sha256
func opensslEVPSHA256() uintptr { return 0 }

// renvo:linkstatic crypto,EVP_DigestInit_ex
func opensslDigestInit(context uintptr, digest uintptr, engine uintptr) int32 {
	return 0
}

// renvo:linkstatic crypto,EVP_DigestUpdate
func opensslDigestUpdate(context uintptr, data *byte, size uintptr) int32 {
	return 0
}

// renvo:linkstatic crypto,EVP_DigestFinal_ex
func opensslDigestFinal(context uintptr, output *byte, size *uint32) int32 {
	return 0
}

// renvo:linkstatic crypto,BN_new
func opensslBigNumberNew() uintptr { return 0 }

// renvo:linkstatic crypto,BN_free
func opensslBigNumberFree(number uintptr) {}

// renvo:linkstatic crypto,BN_set_word
func opensslBigNumberSetWord(number uintptr, value uint64) int32 { return 0 }

// renvo:linkstatic crypto,BN_add_word
func opensslBigNumberAddWord(number uintptr, value uint64) int32 { return 0 }

// renvo:linkstatic crypto,BN_get_word
func opensslBigNumberWord(number uintptr) uint64 { return 0 }

// renvo:linkstatic crypto,BN_bn2binpad
func opensslBigNumberBytes(number uintptr, output *byte, size int32) int32 {
	return 0
}

//export renvo_openssl_self_test
func OpenSSLSelfTest() int32 {
	versionNumber := opensslVersionNumber()
	if versionNumber == 0 {
		return 1
	}
	// Exercise the prepared x86-64 compiler's division scratch register. An
	// object export must still preserve that System V callee-saved register.
	divisionInput := int(versionNumber)
	if divisionInput/7*7+divisionInput%7 != divisionInput {
		return 23
	}
	if opensslVersion(0) == 0 {
		return 2
	}

	message := [3]byte{'a', 'b', 'c'}
	expected := [32]byte{
		0xba, 0x78, 0x16, 0xbf, 0x8f, 0x01, 0xcf, 0xea,
		0x41, 0x41, 0x40, 0xde, 0x5d, 0xae, 0x22, 0x23,
		0xb0, 0x03, 0x61, 0xa3, 0x96, 0x17, 0x7a, 0x9c,
		0xb4, 0x10, 0xff, 0x61, 0xf2, 0x00, 0x15, 0xad,
	}
	var oneShot [32]byte
	if opensslSHA256(&message[0], 3, &oneShot[0]) == nil {
		return 3
	}
	if opensslMemcmp(&oneShot[0], &expected[0], 32) != 0 {
		return 4
	}

	context := opensslDigestContextNew()
	if context == 0 {
		return 5
	}
	digest := opensslEVPSHA256()
	if digest == 0 {
		opensslDigestContextFree(context)
		return 6
	}
	if opensslDigestInit(context, digest, 0) != 1 {
		opensslDigestContextFree(context)
		return 7
	}
	if opensslDigestUpdate(context, &message[0], 1) != 1 {
		opensslDigestContextFree(context)
		return 8
	}
	if opensslDigestUpdate(context, &message[1], 2) != 1 {
		opensslDigestContextFree(context)
		return 9
	}
	var streamed [32]byte
	var streamedSize uint32
	if opensslDigestFinal(context, &streamed[0], &streamedSize) != 1 {
		opensslDigestContextFree(context)
		return 10
	}
	opensslDigestContextFree(context)
	if streamedSize != 32 {
		return 11
	}
	if opensslMemcmp(&streamed[0], &oneShot[0], 32) != 0 {
		return 12
	}

	number := opensslBigNumberNew()
	if number == 0 {
		return 13
	}
	if opensslBigNumberSetWord(number, 0x0102030405060708) != 1 {
		opensslBigNumberFree(number)
		return 14
	}
	if opensslBigNumberAddWord(number, 1) != 1 {
		opensslBigNumberFree(number)
		return 15
	}
	if opensslBigNumberWord(number) != 0x0102030405060709 {
		opensslBigNumberFree(number)
		return 16
	}
	var encoded [8]byte
	if opensslBigNumberBytes(number, &encoded[0], 8) != 8 {
		opensslBigNumberFree(number)
		return 17
	}
	opensslBigNumberFree(number)
	expectedNumber := [8]byte{1, 2, 3, 4, 5, 6, 7, 9}
	for i := 0; i < len(encoded); i++ {
		if encoded[i] != expectedNumber[i] {
			return 18
		}
	}

	var randomA [32]byte
	var randomB [32]byte
	if opensslRandom(&randomA[0], int32(len(randomA))) != 1 {
		return 19
	}
	if opensslRandom(&randomB[0], int32(len(randomB))) != 1 {
		return 20
	}
	nonzero := false
	for i := 0; i < len(randomA); i++ {
		if randomA[i] != 0 {
			nonzero = true
		}
	}
	if !nonzero {
		return 21
	}
	if opensslMemcmp(&randomA[0], &randomB[0], uintptr(len(randomA))) == 0 {
		return 22
	}
	return 0
}
