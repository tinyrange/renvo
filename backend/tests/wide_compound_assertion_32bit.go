package main

func wideRecoveredCompound(value int64) (result int64) {
	defer func() {
		if recovered := recover(); recovered != nil {
			result += recovered.(int64)
		}
	}()
	panic(value)
}

func appMain(args []string) int {
	if wideRecoveredCompound(-1) != int64(-1) {
		print("FAIL: wide compound interface assertion\n")
		return 1
	}
	print("PASS\n")
	return 0
}
