package main

func renvo_runtime_CVMCall1(p0 uint64) int64 { return 0 }
func renvo_runtime_CVMCall5(p0 uint64, p1 uint64, p2 uint64, p3 uint64, p4 uint64) int64 {
	return 0
}
func renvo_runtime_CVMMCall2(p0 uint64, p1 uint64) int64 { return 0 }

func exerciseHypercallIntrinsics(value uint64) int64 {
	result := renvo_runtime_CVMCall1(value)
	result += renvo_runtime_CVMCall5(value, value, value, value, value)
	return result + renvo_runtime_CVMMCall2(value, value)
}

func appMain(args []string) int {
	if len(args) < 0 {
		_ = exerciseHypercallIntrinsics(1)
	}
	print("PASS\n")
	return 0
}
