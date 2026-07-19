package main

type shape14Count int

func appMain(args []string) int {
	if shape14Use(shape14Count(8)) != 16 {
		print("program_shape_14 named\n")
		return 1
	}
	print("PASS\n")
	return 0
}
func shape14Use(v shape14Count) int { return int(v) * 2 }
