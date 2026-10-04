package main

type frameIndexValue struct{ a, b, c int }

func frameIndexMutate(data []byte, index *int) []byte {
	*index = 2
	return data
}

func appMain() int {
	data := []byte{11, 22, 33, 44}
	i, step := 0, 1
	if data[i+step+1] != 33 || data[i+3-step] != 33 || data[i+128-126] != 33 || data[0] != 11 {
		return 1
	}
	data[i+step+1] = 35
	if data[2] != 35 {
		return 2
	}
	values := []frameIndexValue{{1, 2, 3}, {4, 5, 6}, {7, 8, 9}}
	if values[i+step+1].c != 9 {
		return 3
	}
	values[i+step+1].b = 81
	if values[2].b != 81 {
		return 4
	}
	signed8 := []int8{-128, -3, 127}
	signed16 := []int16{-32768, -257, 32767}
	unsigned16 := []uint16{65535, 32768, 7}
	signed32 := []int32{-2147483647 - 1, -123456, 2147483647}
	unsigned32 := []uint32{4294967295, 2147483648, 9}
	if signed8[i] != -128 || signed16[i+step] != -257 || unsigned16[i] != 65535 || signed32[i] >= 0 || unsigned32[i] != 4294967295 {
		return 5
	}
	if signed8[0] != -128 || signed16[0] != -32768 || unsigned16[0] != 65535 || signed32[0] != -2147483647-1 || unsigned32[0] != 4294967295 {
		return 6
	}
	text := "wxyz"
	if text[i+step+1] != 'y' || text[0] != 'w' || text[i+3-step] != 'y' {
		return 7
	}
	largest := int(^uint(0) >> 1)
	if data[largest+step+largest+step] != 11 || text[largest+step+largest+step] != 'w' {
		return 8
	}
	if data[i+step+step+step+step+step+step+step+step+step+step-step-step-step-step-step-step-step-step] != 35 {
		return 9
	}
	index := 0
	if frameIndexMutate(data, &index)[index+step-1] != 35 || index != 2 {
		return 10
	}
	print("PASS\n")
	return 0
}
