package main

//export object_zero_probe
func ObjectZeroProbe() int {
	values := new([32]byte)
	for i := 0; i < 32; i++ {
		if values[i] != 0 {
			return 1
		}
	}
	return 0
}

func appMain() int {
	if ObjectZeroProbe() != 0 {
		panic("object allocation is not zeroed")
	}
	print("PASS\n")
	return 0
}
