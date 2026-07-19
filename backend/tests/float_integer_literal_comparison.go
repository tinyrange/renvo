package main

func appMain(args []string) int {
	one := float64(1)
	negative := float64(-1.5)
	if one != 1 || 1 != one || one < 1 || one > 1 || negative >= -1 || negative < -2 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
