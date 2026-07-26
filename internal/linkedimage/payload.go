package linkedimage

const (
	FormatUnknown = iota
	FormatELF
	FormatPE
	FormatMachO
	FormatWasm
	FormatBytecode
)

const (
	payloadOK = iota
	payloadHeader
	payloadChecksum
)

type LinuxSegment struct {
	FileOffset  int
	Address     int
	FileSize    int
	MemorySize  int
	Permissions int
}

// Payload validates an RNVI transport and returns its backend target, native
// format, and lossless linked payload. It deliberately uses only the language
// core so the self-hosted frontend can consume images without a format parser.
func Payload(src []byte) (target int, format int, native []byte, ok bool) {
	target, format, native, status := payload(src)
	return target, format, native, status == payloadOK
}

func payload(src []byte) (target int, format int, native []byte, status int) {
	if len(src) < 20 ||
		src[0] != 'R' || src[1] != 'N' || src[2] != 'V' || src[3] != 'I' ||
		imageRead16(src, 4) != 1 {
		return 0, 0, nil, payloadHeader
	}
	headerSize := imageRead16(src, 6)
	size := imageRead32(src, 12)
	if headerSize < 20 || headerSize > len(src) || size < 0 || size != len(src)-headerSize {
		return 0, 0, nil, payloadHeader
	}
	native = src[headerSize:]
	a, b := imageAdler(native)
	if imageRead16(src, 16) != a || imageRead16(src, 18) != b {
		return 0, 0, nil, payloadChecksum
	}
	return imageRead16(src, 8), int(src[10]), native, payloadOK
}

func imageRead16(src []byte, at int) int {
	return int(src[at]) | int(src[at+1])<<8
}

func imageRead32(src []byte, at int) int {
	return imageRead16(src, at) | imageRead16(src, at+2)<<16
}

func imageAdler(data []byte) (int, int) {
	a, b := 1, 0
	for i := 0; i < len(data); i++ {
		a = (a + int(data[i])) % 65521
		b = (b + a) % 65521
	}
	return a, b
}
