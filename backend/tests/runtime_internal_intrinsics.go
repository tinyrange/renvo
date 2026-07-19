package main

func renvo_runtime_UnsafeByteAt(data []byte, index int) byte {
	return data[index]
}

func renvo_runtime_UnsafeInt32At(data []int32, index int) int32 {
	return data[index]
}

func renvo_runtime_UnsafeIntAt(data []int, index int) int {
	return data[index]
}

func renvoNonNil(values ...interface{}) {}

type renvoIntrinsicValue struct {
	value int
}

func appMain() int {
	bytes := []byte{3, 5, 7}
	words32 := []int32{-9, 11, 13}
	words := []int{17, 19, 23}
	value := &renvoIntrinsicValue{value: 29}
	renvoNonNil(value)
	if renvo_runtime_UnsafeByteAt(bytes, 2) != 7 {
		return 1
	}
	if renvo_runtime_UnsafeInt32At(words32, 1) != 11 {
		return 1
	}
	if renvo_runtime_UnsafeIntAt(words, 0) != 17 {
		return 1
	}
	if value.value != 29 {
		return 1
	}
	print("PASS\n")
	return 0
}
