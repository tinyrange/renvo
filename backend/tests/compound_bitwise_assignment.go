package main

func compoundBitwiseDynamic(value int) int {
	return value
}

func appMain(args []string) int {
	left := compoundBitwiseDynamic(12)
	mask := compoundBitwiseDynamic(10)
	if left&^mask != 4 {
		print("FAIL dynamic &^\n")
		return 1
	}

	value := 16
	value >>= 2
	if value != 4 {
		print("FAIL >>=\n")
		return 1
	}
	value <<= 3
	if value != 32 {
		print("FAIL <<=\n")
		return 1
	}
	value |= 3
	if value != 35 {
		print("FAIL |=\n")
		return 1
	}
	value &= 31
	if value != 3 {
		print("FAIL &=\n")
		return 1
	}
	value ^= 7
	if value != 4 {
		print("FAIL ^=\n")
		return 1
	}
	value &^= 2
	if value != 4 {
		print("FAIL &^=\n")
		return 1
	}
	print("PASS\n")
	return 0
}
