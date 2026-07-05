package main

var globalSliceMakeStoreData []int

func globalSliceMakeStoreAlloc(n int) {
	data := make([]int, n)
	globalSliceMakeStoreData = data
}

func globalSliceMakeStoreSet(index int, value int) {
	globalSliceMakeStoreData[index] = value
}

func appMain(args []string, env []string) int {
	globalSliceMakeStoreAlloc(4)
	globalSliceMakeStoreSet(0, 11)
	globalSliceMakeStoreSet(3, 17)
	if globalSliceMakeStoreData[0] != 11 {
		print("FAIL\n")
		return 1
	}
	if globalSliceMakeStoreData[3] != 17 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
