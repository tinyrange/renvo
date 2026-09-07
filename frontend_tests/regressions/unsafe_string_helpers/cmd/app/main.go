package main

import u "unsafe"

func badNil() (caught bool) {
	defer func() { caught = recover() != nil }()
	_ = u.String(nil, 1)
	return
}

func badLength(n int64) (caught bool) {
	defer func() { caught = recover() != nil }()
	var b byte
	_ = u.String(&b, n)
	return
}

func main() {
	bytes := []byte{'a', 0, 'b'}
	s := u.String(&bytes[0], uint16(len(bytes)))
	if s != "a\x00b" || u.StringData(s) != &bytes[0] {
		panic("storage or contents")
	}
	if u.String(nil, 0) != "" || u.String(&bytes[0], 0) != "" {
		panic("empty")
	}
	if !badNil() {
		panic("missing nil panic")
	}
	if !badLength(-1) {
		panic("missing length panic")
	}
	text := "hello world"
	if u.String(u.StringData(text), len(text)) != text {
		panic("roundtrip")
	}
	if u.StringData(text[6:]) != (*byte)(u.Add(u.Pointer(u.StringData(text)), 6)) {
		panic("subslice pointer")
	}
	println("PASS")
}
