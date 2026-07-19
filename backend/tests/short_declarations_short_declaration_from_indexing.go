package main

func appMain(args []string) int {
	s := []byte("hi")
	b := s[1]
	if !(b == 'i') {
		print("RENVO-0322 short_declaration_from_indexing failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
