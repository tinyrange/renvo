package main

type appendVariadicSession struct {
	source []byte
}

func appendVariadicReplace(s *appendVariadicSession, source []byte) {
	s.source = append(s.source[:0], source...)
}

func appMain() int {
	var s appendVariadicSession
	appendVariadicReplace(&s, []byte("PASS\n"))
	if string(s.source) != "PASS\n" {
		return 1
	}
	print(string(s.source))
	return 0
}
