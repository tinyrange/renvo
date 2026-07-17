package main

func rtgNestedSliceCallFirst() []byte {
	return []byte{'P', 'A', 'S', 'S', '\n'}
}

func rtgNestedSliceCallSecond() []byte {
	return []byte{'F', 'A', 'I', 'L', '\n'}
}

func appMain() int {
	values := [][]byte{rtgNestedSliceCallFirst(), rtgNestedSliceCallSecond()}
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
