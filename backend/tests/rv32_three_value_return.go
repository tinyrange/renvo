package main

func renvoRV32ThreeValueReturn() (uint16, uint16, bool) {
	return 0x1234, 0x5678, true
}

func appMain(args []string) int {
	_, second, ok := renvoRV32ThreeValueReturn()
	if second != 0x5678 || !ok {
		print("FAIL: RV32 three-value return\n")
		return 1
	}
	print("PASS\n")
	return 0
}
