package main

var directReturnMagic = "RNVO"
var directReturnDefinitionBytes = []byte{
	0x77, 0xd3, 0xb7, 0x55, 0x41, 0x36, 0x35, 0x0b,
	0xa9, 0x5f, 0xf6, 0xe3, 0x86, 0x67, 0xd6, 0x55,
	0x4a, 0x70, 0x1c, 0xc3, 0xbe, 0x18, 0x1f, 0xb3,
	0xe9, 0x22, 0x37, 0x54, 0xce, 0x80, 0x23, 0x2f,
}

func directReturnBuiltInBinding(target int) (string, string, int, bool) {
	if target != 1 {
		return "", "", 0, false
	}
	return "linux/amd64", "\x77\xd3\xb7\x55\x41\x36\x35\v\xa9\x5f\xf6\xe3\x86\x67\xd6\x55\x4a\x70\x1c\xc3\xbe\x18\x1f\xb3\xe9\x22\x37\x54\xce\x80\x23\x2f", 3, true
}

func directReturnBinding(target int) (string, string, int, bool) {
	return directReturnBuiltInBinding(target)
}

func directReturnRead32(src []byte, at int) int {
	return int(src[at]) | int(src[at+1])<<8 | int(src[at+2])<<16 | int(src[at+3])<<24
}

func directReturnMatches(src []byte, target int) bool {
	expectedTarget, expectedDefinition, expectedVersion, ok := directReturnBinding(target)
	bindingStart := len(src) - 52 - len(expectedTarget)
	if !ok || len(expectedDefinition) != 32 || bindingStart < 14 ||
		src[0] != directReturnMagic[0] || src[1] != directReturnMagic[1] ||
		src[2] != directReturnMagic[2] || src[3] != directReturnMagic[3] {
		return false
	}
	if directReturnRead32(src, 10) != len(src)-14 {
		return false
	}
	targetData := bindingStart + 6
	hashHeader := targetData + len(expectedTarget)
	hashData := hashHeader + 6
	versionHeader := hashData + 32
	versionData := versionHeader + 6
	return int(src[bindingStart])|int(src[bindingStart+1])<<8 == 4 &&
		directReturnRead32(src, bindingStart+2) == len(expectedTarget) &&
		string(src[targetData:hashHeader]) == expectedTarget &&
		int(src[hashHeader])|int(src[hashHeader+1])<<8 == 5 &&
		directReturnRead32(src, hashHeader+2) == 32 &&
		string(src[hashData:versionHeader]) == expectedDefinition &&
		int(src[versionHeader])|int(src[versionHeader+1])<<8 == 6 &&
		directReturnRead32(src, versionHeader+2) == 2 &&
		int(src[versionData])|int(src[versionData+1])<<8 == expectedVersion
}

func directReturnWrite32(dst []byte, at int, value int) {
	dst[at] = byte(value)
	dst[at+1] = byte(value >> 8)
	dst[at+2] = byte(value >> 16)
	dst[at+3] = byte(value >> 24)
}

func appMain(args []string) int {
	target, _, version, _ := directReturnBinding(1)
	src := make([]byte, 14+52+len(target))
	src[0] = 'R'
	src[1] = 'N'
	src[2] = 'V'
	src[3] = 'O'
	directReturnWrite32(src, 10, len(src)-14)
	bindingStart := 14
	src[bindingStart] = 4
	directReturnWrite32(src, bindingStart+2, len(target))
	for i := 0; i < len(target); i++ {
		src[bindingStart+6+i] = target[i]
	}
	hashHeader := bindingStart + 6 + len(target)
	src[hashHeader] = 5
	directReturnWrite32(src, hashHeader+2, len(directReturnDefinitionBytes))
	for i := 0; i < len(directReturnDefinitionBytes); i++ {
		src[hashHeader+6+i] = directReturnDefinitionBytes[i]
	}
	versionHeader := hashHeader + 6 + len(directReturnDefinitionBytes)
	src[versionHeader] = 6
	directReturnWrite32(src, versionHeader+2, 2)
	src[versionHeader+6] = byte(version)
	if directReturnMatches(src, 1) {
		print("PASS\n")
		return 0
	}
	print("FAIL\n")
	return 1
}
