package main

type rangeBytesEncoding struct {
	alphabet string
	pad      byte
}

func (e *rangeBytesEncoding) Encode(dst, src []byte) {
	_ = dst
	_ = src
}

func (e *rangeBytesEncoding) EncodeToString(src []byte) string {
	b := make([]byte, len(src))
	e.Encode(b, src)
	return string(b)
}

func rangeBytesEncodedLen(n int) int { return n * 2 }

func rangeBytesNarrowShift() int {
	dirty := 0x12345678
	b := byte(104)
	_ = dirty
	return int(b >> 4)
}

func rangeBytesEncode(src []byte) string {
	const digits = "0123456789abcdef"
	dst := make([]byte, rangeBytesEncodedLen(len(src)))
	for i, b := range src {
		hi := int(b >> 4)
		lo := int(b & 15)
		dst[2*i] = digits[hi]
		dst[2*i+1] = digits[lo]
	}
	return string(dst)
}

func appMain() int {
	if rangeBytesNarrowShift() != 6 {
		print("FAIL\n")
		return 1
	}
	if rangeBytesEncode([]byte("hello")) != "68656c6c6f" {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
