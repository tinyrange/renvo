package main

type namedSliceConversionValues []int

func appMain() int {
	named := namedSliceConversionValues{19, 23}
	values := []int(named)
	if len(values) == 2 && values[0]+values[1] == 42 {
		print("PASS\n")
		return 0
	}
	print("FAIL\n")
	return 1
}
