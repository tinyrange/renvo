package main

func nestedSliceFirst() []byte {
	return []byte{'P', 'A'}
}

func nestedSliceSecond() []byte {
	return []byte{'S', 'S', '\n'}
}

func appMain() int {
	values := [][]byte{nestedSliceFirst(), nestedSliceSecond()}
	first := values[0]
	second := values[1]
	if len(values) == 2 && len(first) == 2 && len(second) == 3 &&
		first[0] == 'P' && first[1] == 'A' &&
		second[0] == 'S' && second[1] == 'S' && second[2] == '\n' {
		print("PASS\n")
		return 0
	}
	print("FAIL\n")
	return 1
}
