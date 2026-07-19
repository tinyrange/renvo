package main

type indexedArrayFields struct {
	ints   [16]int
	floats [16]float64
}

func setIndexedArrayFields(v *indexedArrayFields, index int, integer int, floating float64) {
	v.ints[index] = integer
	v.floats[index] = floating
}

func appMain(args []string) int {
	var v indexedArrayFields
	setIndexedArrayFields(&v, 3, 7, 1.5)
	if v.ints[3] == 7 && v.floats[3] == 1.5 {
		print("PASS\n")
		return 0
	}
	print("FAIL\n")
	return 1
}
