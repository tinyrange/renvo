package apk

import "errors"

func Build(sharedObject []byte, config Config) ([]byte, error) {
	if err := validateConfig(config); err != nil {
		return nil, err
	}
	if !validAndroidSharedObject(sharedObject) {
		return nil, errors.New("input is not an AArch64 NativeActivity shared object")
	}
	manifest := buildManifest(config)
	entries := []zipEntry{
		{name: "AndroidManifest.xml", data: manifest},
		{name: "lib/arm64-v8a/librenvo.so", data: sharedObject},
	}
	local, central, end, ok := buildZIPSections(entries)
	if !ok {
		return nil, errors.New("APK contents exceed the ZIP32 limits")
	}
	signingBlock := buildV2SigningBlock(local, central, end)
	if len(signingBlock) == 0 {
		return nil, errors.New("could not sign APK")
	}
	centralOffset := len(local) + len(signingBlock)
	put32(end, 16, centralOffset)
	result := make([]byte, 0, centralOffset+len(central)+len(end))
	result = append(result, local...)
	result = append(result, signingBlock...)
	result = append(result, central...)
	result = append(result, end...)
	return result, nil
}

func validAndroidSharedObject(image []byte) bool {
	if len(image) < 64 || image[0] != 0x7f || image[1] != 'E' ||
		image[2] != 'L' || image[3] != 'F' || image[4] != 2 || image[5] != 1 {
		return false
	}
	if get16(image, 16) != 3 || get16(image, 18) != 183 {
		return false
	}
	needle := []byte("ANativeActivity_onCreate\x00")
	return containsBytes(image, needle)
}

func containsBytes(haystack []byte, needle []byte) bool {
	if len(needle) == 0 {
		return true
	}
	for at := 0; at+len(needle) <= len(haystack); at++ {
		matched := true
		for i := 0; i < len(needle); i++ {
			if haystack[at+i] != needle[i] {
				matched = false
				break
			}
		}
		if matched {
			return true
		}
	}
	return false
}

func append16(out []byte, value int) []byte {
	return append(out, byte(value), byte(value>>8))
}

func append32(out []byte, value int) []byte {
	return append(out, byte(value), byte(value>>8), byte(value>>16), byte(value>>24))
}

func append64(out []byte, value uint64) []byte {
	out = append32(out, int(uint32(value)))
	return append32(out, int(uint32(value>>32)))
}

func get16(data []byte, at int) int {
	return int(data[at]) | int(data[at+1])<<8
}

func get32(data []byte, at int) int {
	return int(data[at]) | int(data[at+1])<<8 |
		int(data[at+2])<<16 | int(data[at+3])<<24
}

func put16(data []byte, at int, value int) {
	data[at] = byte(value)
	data[at+1] = byte(value >> 8)
}

func put32(data []byte, at int, value int) {
	data[at] = byte(value)
	data[at+1] = byte(value >> 8)
	data[at+2] = byte(value >> 16)
	data[at+3] = byte(value >> 24)
}
