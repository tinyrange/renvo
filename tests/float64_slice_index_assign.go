package main

type float64SliceValues struct {
	items []float64
}

func setFloat64SliceIndex(v *float64SliceValues, index int, value float64) {
	v.items[index] = value
}

func appMain(args []string) int {
	v := float64SliceValues{items: make([]float64, 6)}
	setFloat64SliceIndex(&v, 3, 1.0)
	if v.items[3] == 1.0 {
		print("PASS\n")
		return 0
	}
	print("FAIL\n")
	return 1
}
