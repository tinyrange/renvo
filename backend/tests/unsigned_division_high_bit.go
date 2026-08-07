package main

func appMain(args []string) int {
	var left32 uint32 = 3451514202
	var right32 uint32 = 23
	quotient32 := left32 / right32
	remainder32 := left32 % right32
	compound32 := left32
	compound32 += quotient32
	if quotient32 != uint32(150065834) || remainder32 != uint32(20) || compound32 != uint32(3601580036) {
		print("FAIL: uint32 division\n")
		return 1
	}
	var left64 uint64 = 0xf000000000000123
	var right64 uint64 = 31
	if left64/right64 != uint64(0x7bdef7bdef7bdf8) || left64%right64 != uint64(27) {
		print("FAIL: uint64 division\n")
		return 1
	}
	print("PASS\n")
	return 0
}
