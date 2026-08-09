package apk

import (
	"archive/zip"
	"bytes"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/binary"
	"io"
	"testing"
)

func TestBuildProducesVerifiedV2NativeActivityAPK(t *testing.T) {
	sharedObject := testSharedObject()
	config, err := ParseConfig([]byte(`
package=dev.renvo.testapp
name=Renvo Test
version_code=7
version_name=2.3
min_sdk=24
target_sdk=35
`))
	if err != nil {
		t.Fatal(err)
	}
	image, err := Build(sharedObject, config)
	if err != nil {
		t.Fatal(err)
	}
	verifyV2Signature(t, image)

	reader, err := zip.NewReader(bytes.NewReader(image), int64(len(image)))
	if err != nil {
		t.Fatalf("open APK as ZIP: %v", err)
	}
	if len(reader.File) != 2 {
		t.Fatalf("APK entries = %d, want 2", len(reader.File))
	}
	entries := make(map[string][]byte)
	for i := 0; i < len(reader.File); i++ {
		file := reader.File[i]
		if file.Method != zip.Store {
			t.Fatalf("APK entry %q uses compression method %d", file.Name, file.Method)
		}
		opened, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(opened)
		opened.Close()
		if err != nil {
			t.Fatal(err)
		}
		entries[file.Name] = data
	}
	if !bytes.Equal(entries["lib/arm64-v8a/librenvo.so"], sharedObject) {
		t.Fatal("APK did not preserve librenvo.so")
	}
	verifyNativeActivityManifest(t, entries["AndroidManifest.xml"], config)
}

func TestParseConfigRejectsUnsupportedAndUnsafeValues(t *testing.T) {
	cases := []string{
		"package=single\n",
		"package=dev.renvo.app\nmin_sdk=23\n",
		"package=dev.renvo.app\nname=bad\x01name\n",
		"package=dev.renvo.app\nunknown=value\n",
		"package=dev.renvo.app\nmin_sdk=35\ntarget_sdk=34\n",
	}
	for i := 0; i < len(cases); i++ {
		if _, err := ParseConfig([]byte(cases[i])); err == nil {
			t.Fatalf("case %d unexpectedly parsed", i)
		}
	}
}

func testSharedObject() []byte {
	image := make([]byte, 256)
	copy(image, []byte{0x7f, 'E', 'L', 'F', 2, 1, 1})
	binary.LittleEndian.PutUint16(image[16:18], 3)
	binary.LittleEndian.PutUint16(image[18:20], 183)
	copy(image[96:], []byte("ANativeActivity_onCreate\x00"))
	return image
}

func verifyV2Signature(t *testing.T, image []byte) {
	t.Helper()
	if len(image) < 22 {
		t.Fatal("APK is too small")
	}
	eocdOffset := len(image) - 22
	if binary.LittleEndian.Uint32(image[eocdOffset:eocdOffset+4]) != 0x06054b50 {
		t.Fatal("APK has no terminal ZIP EOCD")
	}
	centralOffset := int(binary.LittleEndian.Uint32(image[eocdOffset+16 : eocdOffset+20]))
	if centralOffset < 32 || centralOffset > eocdOffset {
		t.Fatal("APK central-directory offset is invalid")
	}
	footer := centralOffset - 24
	if !bytes.Equal(image[footer+8:centralOffset], []byte("APK Sig Block 42")) {
		t.Fatal("APK has no signing-block magic")
	}
	blockSize := int(binary.LittleEndian.Uint64(image[footer : footer+8]))
	blockStart := centralOffset - blockSize - 8
	if blockStart < 0 ||
		int(binary.LittleEndian.Uint64(image[blockStart:blockStart+8])) != blockSize {
		t.Fatal("APK signing-block size fields disagree")
	}
	pairAt := blockStart + 8
	pairSize := int(binary.LittleEndian.Uint64(image[pairAt : pairAt+8]))
	if pairAt+8+pairSize != footer ||
		binary.LittleEndian.Uint32(image[pairAt+8:pairAt+12]) != 0x7109871a {
		t.Fatal("APK signing block does not contain exactly one v2 pair")
	}
	v2 := image[pairAt+12 : pairAt+8+pairSize]
	signers, rest, ok := readLengthPrefixed(v2)
	if !ok || len(rest) != 0 {
		t.Fatal("invalid v2 signers sequence")
	}
	signer, rest, ok := readLengthPrefixed(signers)
	if !ok || len(rest) != 0 {
		t.Fatal("invalid v2 signer")
	}
	signedData, signer, ok := readLengthPrefixed(signer)
	if !ok {
		t.Fatal("invalid v2 signed data")
	}
	signatures, signer, ok := readLengthPrefixed(signer)
	if !ok {
		t.Fatal("invalid v2 signatures")
	}
	publicKey, signer, ok := readLengthPrefixed(signer)
	if !ok || len(signer) != 0 {
		t.Fatal("invalid v2 public key")
	}
	signatureRecord, signatures, ok := readLengthPrefixed(signatures)
	if !ok || len(signatures) != 0 || len(signatureRecord) < 8 ||
		binary.LittleEndian.Uint32(signatureRecord[:4]) != apkV2Algorithm {
		t.Fatal("invalid v2 signature record")
	}
	signature, rest, ok := readLengthPrefixed(signatureRecord[4:])
	if !ok || len(rest) != 0 {
		t.Fatal("invalid v2 RSA signature")
	}

	digests, signedRemainder, ok := readLengthPrefixed(signedData)
	if !ok {
		t.Fatal("invalid v2 digests")
	}
	certificates, signedRemainder, ok := readLengthPrefixed(signedRemainder)
	if !ok {
		t.Fatal("invalid v2 certificates")
	}
	attributes, signedRemainder, ok := readLengthPrefixed(signedRemainder)
	if !ok || len(signedRemainder) != 0 || len(attributes) != 0 {
		t.Fatal("invalid v2 additional attributes")
	}
	digestRecord, digests, ok := readLengthPrefixed(digests)
	if !ok || len(digests) != 0 || len(digestRecord) < 8 ||
		binary.LittleEndian.Uint32(digestRecord[:4]) != apkV2Algorithm {
		t.Fatal("invalid v2 digest record")
	}
	storedDigest, rest, ok := readLengthPrefixed(digestRecord[4:])
	if !ok || len(rest) != 0 {
		t.Fatal("invalid v2 content digest")
	}
	certificateDER, certificates, ok := readLengthPrefixed(certificates)
	if !ok || len(certificates) != 0 {
		t.Fatal("invalid v2 certificate sequence")
	}
	certificate, err := x509.ParseCertificate(certificateDER)
	if err != nil {
		t.Fatalf("parse signing certificate: %v", err)
	}
	if !bytes.Equal(certificate.RawSubjectPublicKeyInfo, publicKey) {
		t.Fatal("v2 public key does not match the signing certificate")
	}
	rsaKey, ok := certificate.PublicKey.(*rsa.PublicKey)
	if !ok {
		t.Fatal("v2 certificate is not RSA")
	}
	signedHash := sha256.Sum256(signedData)
	if err := rsa.VerifyPKCS1v15(rsaKey, crypto.SHA256, signedHash[:], signature); err != nil {
		t.Fatalf("verify v2 RSA signature: %v", err)
	}

	central := image[centralOffset:eocdOffset]
	digestEOCD := append([]byte(nil), image[eocdOffset:]...)
	binary.LittleEndian.PutUint32(digestEOCD[16:20], uint32(blockStart))
	wantDigest := hostAPKContentDigest(image[:blockStart], central, digestEOCD)
	if !bytes.Equal(storedDigest, wantDigest) {
		t.Fatalf("v2 content digest = %x, want %x", storedDigest, wantDigest)
	}
}

func hostAPKContentDigest(sections ...[]byte) []byte {
	var chunks []byte
	count := 0
	for _, section := range sections {
		for start := 0; start < len(section); start += 1 << 20 {
			end := start + (1 << 20)
			if end > len(section) {
				end = len(section)
			}
			framed := make([]byte, 5+end-start)
			framed[0] = 0xa5
			binary.LittleEndian.PutUint32(framed[1:5], uint32(end-start))
			copy(framed[5:], section[start:end])
			hash := sha256.Sum256(framed)
			chunks = append(chunks, hash[:]...)
			count++
		}
	}
	top := make([]byte, 5+len(chunks))
	top[0] = 0x5a
	binary.LittleEndian.PutUint32(top[1:5], uint32(count))
	copy(top[5:], chunks)
	hash := sha256.Sum256(top)
	return hash[:]
}

func readLengthPrefixed(data []byte) ([]byte, []byte, bool) {
	if len(data) < 4 {
		return nil, nil, false
	}
	size := int(binary.LittleEndian.Uint32(data[:4]))
	if size < 0 || 4+size > len(data) {
		return nil, nil, false
	}
	return data[4 : 4+size], data[4+size:], true
}

func verifyNativeActivityManifest(t *testing.T, manifest []byte, config Config) {
	t.Helper()
	if len(manifest) < 8 || binary.LittleEndian.Uint16(manifest[:2]) != 3 ||
		int(binary.LittleEndian.Uint32(manifest[4:8])) != len(manifest) {
		t.Fatal("AndroidManifest.xml is not a bounded binary XML document")
	}
	pool, next := readManifestPool(t, manifest, 8)
	if next+8 > len(manifest) || binary.LittleEndian.Uint16(manifest[next:next+2]) != 0x0180 {
		t.Fatal("AndroidManifest.xml has no resource map")
	}
	next += int(binary.LittleEndian.Uint32(manifest[next+4 : next+8]))
	found := make(map[string]bool)
	for next < len(manifest) {
		if next+8 > len(manifest) {
			t.Fatal("truncated binary XML node")
		}
		kind := binary.LittleEndian.Uint16(manifest[next : next+2])
		size := int(binary.LittleEndian.Uint32(manifest[next+4 : next+8]))
		if size < 8 || next+size > len(manifest) {
			t.Fatal("invalid binary XML node size")
		}
		if kind == 0x0102 {
			name := pool[int(binary.LittleEndian.Uint32(manifest[next+20:next+24]))]
			found[name] = true
			attributeStart := next + 16 + int(binary.LittleEndian.Uint16(manifest[next+24:next+26]))
			attributeSize := int(binary.LittleEndian.Uint16(manifest[next+26 : next+28]))
			attributeCount := int(binary.LittleEndian.Uint16(manifest[next+28 : next+30]))
			for i := 0; i < attributeCount; i++ {
				at := attributeStart + i*attributeSize
				attributeName := pool[int(binary.LittleEndian.Uint32(manifest[at+4:at+8]))]
				kind := manifest[at+15]
				value := int(binary.LittleEndian.Uint32(manifest[at+16 : at+20]))
				if name == "manifest" && attributeName == "package" &&
					(kind != manifestTypeString || pool[value] != config.Package) {
					t.Fatal("manifest package attribute is wrong")
				}
				if name == "application" && attributeName == "hasCode" &&
					(kind != manifestTypeBoolean || value != 0) {
					t.Fatal("NativeActivity APK must set hasCode=false")
				}
				if name == "application" && attributeName == "extractNativeLibs" &&
					(kind != manifestTypeBoolean || uint32(value) != 0xffffffff) {
					t.Fatal("NativeActivity APK must extract librenvo.so")
				}
				if name == "activity" && attributeName == "exported" &&
					(kind != manifestTypeBoolean || uint32(value) != 0xffffffff) {
					t.Fatal("launcher NativeActivity must be exported")
				}
			}
		}
		next += size
	}
	for _, name := range []string{
		"manifest", "uses-sdk", "application", "activity", "meta-data",
		"intent-filter", "action", "category",
	} {
		if !found[name] {
			t.Fatalf("AndroidManifest.xml omitted <%s>", name)
		}
	}
}

func readManifestPool(t *testing.T, data []byte, at int) ([]string, int) {
	t.Helper()
	if at+28 > len(data) || binary.LittleEndian.Uint16(data[at:at+2]) != 1 {
		t.Fatal("binary XML has no UTF-8 string pool")
	}
	size := int(binary.LittleEndian.Uint32(data[at+4 : at+8]))
	count := int(binary.LittleEndian.Uint32(data[at+8 : at+12]))
	stringsStart := int(binary.LittleEndian.Uint32(data[at+20 : at+24]))
	if size < 28+count*4 || at+size > len(data) {
		t.Fatal("invalid binary XML string pool")
	}
	result := make([]string, count)
	for i := 0; i < count; i++ {
		offset := int(binary.LittleEndian.Uint32(data[at+28+i*4 : at+32+i*4]))
		stringAt := at + stringsStart + offset
		if stringAt+2 > at+size {
			t.Fatal("invalid string-pool offset")
		}
		length := int(data[stringAt+1])
		if stringAt+2+length > at+size {
			t.Fatal("invalid string-pool value")
		}
		result[i] = string(data[stringAt+2 : stringAt+2+length])
	}
	return result, at + size
}
