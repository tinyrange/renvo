package apk

const debugCertificateBase64 = "MIIDKDCCAhCgAwIBAgIBATANBgkqhkiG9w0BAQsFADAsMRowGAYDVQQDDBFSZW52byBEZXZlbG9wbWVudDEOMAwGA1UECgwFUmVudm8wIBcNMjYwODA5MDQ0NjE3WhgPMjA1NjA4MDEwNDQ2MTdaMCwxGjAYBgNVBAMMEVJlbnZvIERldmVsb3BtZW50MQ4wDAYDVQQKDAVSZW52bzCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBALN4bDoMpsSjDs96qdeuzqlpwohvDQL0w6kj9fFlB66I1R/Fa9kLO+YdeH951ICR7szO05YlF8K4d4Dg788xC/AEWe4TfiIZ/INU0y9yWze1nibXrHkmbxulUc8BexTkm1NDJC0pmanQZG8crDOT0xUa2MUuryP80n3pf+iZnJDcqjoxvXVvLUTYgiFwuEh7qoA9W34W1udr2GhMUNNa7s23E9Z4yeAfCLQyBkMVeZoXATIcgch1n6CpOMdDLOe2iR+aeb/vnIVg91s+MUt9o8v46rqtTo92vLShvwf6dRRHZHvOnkSFZWUo9gdc1EXzYhoxhO6Ln9Ya1i8Uwq5TOOMCAwEAAaNTMFEwHQYDVR0OBBYEFGExO4pZQm7bgcOUjCzZPmfI6XUVMB8GA1UdIwQYMBaAFGExO4pZQm7bgcOUjCzZPmfI6XUVMA8GA1UdEwEB/wQFMAMBAf8wDQYJKoZIhvcNAQELBQADggEBAB/s6yQwb2WY3euSIdg426iISEyHTZp29aWmkmhyucocY/kkGp8N9gEE++Q613CMCaB1qoE7HHu7rqRIj7Rz3uJQNqNbHuiuLiDUTaCBS9zUy1iCpwzICGarJjWbE3MQBiTDb3jr9kqV8/F8g1mqyghmEQhi9rXzXoba7NRdyf/xq7L1aGEcnhDiUfk6sfZZ2qfPegPeS1Ch++eYC+1x778/ctjKEws9aZHDHkbLRl4WPCMEarY9xgj7Y5ILH9OYqjex4WlpKa9qVBaA/dOvCUua62Bj3t0UNvHK4Vva5wsR2Ynr1P0B1RNHCOFu6idIgNl6wDV2UcVLT8MHthHytZ4="

const debugPublicKeyBase64 = "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAs3hsOgymxKMOz3qp167OqWnCiG8NAvTDqSP18WUHrojVH8Vr2Qs75h14f3nUgJHuzM7TliUXwrh3gODvzzEL8ARZ7hN+Ihn8g1TTL3JbN7WeJteseSZvG6VRzwF7FOSbU0MkLSmZqdBkbxysM5PTFRrYxS6vI/zSfel/6JmckNyqOjG9dW8tRNiCIXC4SHuqgD1bfhbW52vYaExQ01ruzbcT1njJ4B8ItDIGQxV5mhcBMhyByHWfoKk4x0Ms57aJH5p5v++chWD3Wz4xS32jy/jquq1Oj3a8tKG/B/p1FEdke86eRIVlZSj2B1zURfNiGjGE7ouf1hrWLxTCrlM44wIDAQAB"

const apkV2Algorithm = 0x0103

func buildV2SigningBlock(local []byte, central []byte, end []byte) []byte {
	digest := apkContentDigest(local, central, end)
	certificate := decodeBase64(debugCertificateBase64)
	publicKey := decodeBase64(debugPublicKeyBase64)
	if len(digest) != 32 || len(certificate) == 0 || len(publicKey) == 0 {
		return nil
	}
	digestRecord := make([]byte, 0, 40)
	digestRecord = append32(digestRecord, apkV2Algorithm)
	digestRecord = appendLengthPrefixed(digestRecord, digest)
	digestSequence := appendLengthPrefixed(nil,
		appendLengthPrefixed(nil, digestRecord))
	certificateSequence := appendLengthPrefixed(nil,
		appendLengthPrefixed(nil, certificate))
	attributes := appendLengthPrefixed(nil, nil)
	signedData := make([]byte, 0,
		len(digestSequence)+len(certificateSequence)+len(attributes))
	signedData = append(signedData, digestSequence...)
	signedData = append(signedData, certificateSequence...)
	signedData = append(signedData, attributes...)

	signature := rsaSignSHA256(sha256Digest(signedData))
	if len(signature) != 256 {
		return nil
	}
	signatureRecord := make([]byte, 0, 264)
	signatureRecord = append32(signatureRecord, apkV2Algorithm)
	signatureRecord = appendLengthPrefixed(signatureRecord, signature)
	signatureSequence := appendLengthPrefixed(nil,
		appendLengthPrefixed(nil, signatureRecord))
	signer := appendLengthPrefixed(nil, signedData)
	signer = append(signer, signatureSequence...)
	signer = appendLengthPrefixed(signer, publicKey)
	v2Value := appendLengthPrefixed(nil, appendLengthPrefixed(nil, signer))

	pairLength := 4 + len(v2Value)
	pair := make([]byte, 0, 8+pairLength)
	pair = append64(pair, uint64(pairLength))
	pair = append32(pair, 0x7109871a)
	pair = append(pair, v2Value...)
	blockSize := uint64(len(pair) + 24)
	block := make([]byte, 0, int(blockSize)+8)
	block = append64(block, blockSize)
	block = append(block, pair...)
	block = append64(block, blockSize)
	block = append(block, []byte("APK Sig Block 42")...)
	return block
}

func apkContentDigest(local []byte, central []byte, end []byte) []byte {
	sections := [][]byte{local, central, end}
	chunkDigests := make([]byte, 0, 96)
	chunkCount := 0
	for section := 0; section < len(sections); section++ {
		data := sections[section]
		for start := 0; start < len(data); start += 1 << 20 {
			finish := start + (1 << 20)
			if finish > len(data) {
				finish = len(data)
			}
			framed := make([]byte, 0, 5+finish-start)
			framed = append(framed, 0xa5)
			framed = append32(framed, finish-start)
			framed = append(framed, data[start:finish]...)
			chunkDigests = append(chunkDigests, sha256Digest(framed)...)
			chunkCount++
		}
	}
	top := make([]byte, 0, 5+len(chunkDigests))
	top = append(top, 0x5a)
	top = append32(top, chunkCount)
	top = append(top, chunkDigests...)
	return sha256Digest(top)
}

func appendLengthPrefixed(out []byte, value []byte) []byte {
	out = append32(out, len(value))
	return append(out, value...)
}

func decodeBase64(value string) []byte {
	out := make([]byte, 0, len(value)*3/4)
	accumulator := 0
	bits := 0
	for i := 0; i < len(value); i++ {
		if value[i] == '=' {
			break
		}
		digit, ok := base64Digit(value[i])
		if !ok {
			return nil
		}
		accumulator = accumulator<<6 | digit
		bits += 6
		if bits >= 8 {
			bits -= 8
			out = append(out, byte(accumulator>>bits))
			accumulator &= (1 << bits) - 1
		}
	}
	return out
}

func base64Digit(value byte) (int, bool) {
	if value >= 'A' && value <= 'Z' {
		return int(value - 'A'), true
	}
	if value >= 'a' && value <= 'z' {
		return int(value-'a') + 26, true
	}
	if value >= '0' && value <= '9' {
		return int(value-'0') + 52, true
	}
	if value == '+' {
		return 62, true
	}
	if value == '/' {
		return 63, true
	}
	return 0, false
}
