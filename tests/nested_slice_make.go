package main

func appMain() int {
	values := make([][]byte, 2)
	values[0] = []byte("ab")
	values[1] = []byte("cde")
	first := values[0]
	second := values[1]
	if len(values) == 2 && len(first) == 2 && second[2] == 'e' {
		print("PASS\n")
		return 0
	}
	print("FAIL\n")
	return 1
}
