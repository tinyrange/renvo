package main

func unsignedShift(value uint32, count uint) uint32 { return value >> count }
func appMain() int {
	for count := uint(32); count <= 65; count++ {
		if unsignedShift(0xffffffff, count) != 0 {
			print("FAIL oversized unsigned shift\n")
			return 1
		}
	}
	print("PASS\n")
	return 0
}
