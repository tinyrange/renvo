package main

var m16ObjectWideInitializer = [3]uint64{
	0x00cf9b000000ffff,
	0x00cf93000000ffff,
	0x0000890010000067,
}

func appMain(args []string) int {
	if m16ObjectWideInitializer[0] != 0x00cf9b000000ffff ||
		m16ObjectWideInitializer[1] != 0x00cf93000000ffff ||
		m16ObjectWideInitializer[2] != 0x0000890010000067 {
		return 1
	}
	print("PASS\n")
	return 0
}
