package main

func renvo_runtime_UnsafeInt32At(data []int32, index int) int32 {
	return data[index]
}

type unsafeDescriptor struct {
	data []int32
}

var unsafeGlobal = []int32{-17, 29}
var unsafeCalls int

func unsafeSource(value *unsafeDescriptor) *unsafeDescriptor {
	unsafeCalls++
	return value
}

func unsafeSlice(value *unsafeDescriptor) []int32 {
	unsafeCalls++
	return value.data
}

func unsafeIndex() int {
	unsafeCalls++
	return 1
}

func appMain() int {
	local := []int32{-17, 29}
	value := unsafeDescriptor{data: local}
	pointer := &value
	if renvo_runtime_UnsafeInt32At(local, 0) != -17 ||
		renvo_runtime_UnsafeInt32At(unsafeGlobal, 1) != 29 ||
		renvo_runtime_UnsafeInt32At(value.data, 1) != 29 ||
		renvo_runtime_UnsafeInt32At(pointer.data, 0) != -17 {
		return 1
	}
	if renvo_runtime_UnsafeInt32At(unsafeSource(pointer).data, unsafeIndex()) != 29 || unsafeCalls != 2 {
		return 2
	}
	if renvo_runtime_UnsafeInt32At(unsafeSlice(pointer), 0) != -17 || unsafeCalls != 3 {
		return 3
	}
	if renvo_runtime_UnsafeInt32At(local[1:], 0) != 29 {
		return 4
	}
	print("PASS\n")
	return 0
}
