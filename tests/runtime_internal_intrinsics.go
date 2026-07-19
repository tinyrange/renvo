package main

func rtg_runtime_UnsafeByteAt(data []byte, index int) byte {
	return data[index]
}

func rtg_runtime_UnsafeInt32At(data []int32, index int) int32 {
	return data[index]
}

func rtg_runtime_UnsafeIntAt(data []int, index int) int {
	return data[index]
}

func rtgTrustNonNil(values ...interface{}) {}

type rtgIntrinsicValue struct {
	value int
}

func appMain() int {
	bytes := []byte{3, 5, 7}
	words32 := []int32{-9, 11, 13}
	words := []int{17, 19, 23}
	value := &rtgIntrinsicValue{value: 29}
	rtgTrustNonNil(value)
	if rtg_runtime_UnsafeByteAt(bytes, 2) != 7 {
		return 1
	}
	if rtg_runtime_UnsafeInt32At(words32, 1) != 11 {
		return 1
	}
	if rtg_runtime_UnsafeIntAt(words, 0) != 17 {
		return 1
	}
	if value.value != 29 {
		return 1
	}
	print("PASS\n")
	return 0
}
