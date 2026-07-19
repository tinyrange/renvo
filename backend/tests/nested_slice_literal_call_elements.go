package main

func renvoNestedSliceCallFirst() []byte {
	return []byte{'P', 'A', 'S', 'S', '\n'}
}

func renvoNestedSliceCallSecond() []byte {
	return []byte{'F', 'A', 'I', 'L', '\n'}
}

func appMain() int {
	values := [][]byte{renvoNestedSliceCallFirst(), renvoNestedSliceCallSecond()}
	if len(values) != 2 {
		print("FAIL\n")
		return 1
	}
	first := values[0]
	second := values[1]
	if len(first) != 5 || len(second) != 5 || first[0] != 'P' || second[0] != 'F' {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
