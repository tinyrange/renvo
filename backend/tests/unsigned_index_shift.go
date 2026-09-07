package main

func appMain(args []string) int {
	words := []uint32{2147483648, 4294967295}
	carry := words[0] >> 31
	if carry != 1 {
		panic("unsigned indexed right shift")
	}
	if words[1]>>16 != 65535 {
		panic("unsigned indexed high bits")
	}
	small := [1]uint16{32768}
	if small[0]>>15 != 1 {
		panic("unsigned array right shift")
	}
	print("PASS\n")
	return 0
}
