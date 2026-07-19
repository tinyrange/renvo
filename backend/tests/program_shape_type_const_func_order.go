package main

type shapeFiveInt int

const shapeFiveConst = 9

func shapeFiveValue(v shapeFiveInt) int { return int(v) + shapeFiveConst }
func appMain(args []string) int {
	if shapeFiveValue(shapeFiveInt(4)) != 13 {
		print("program_shape_05 order\n")
		return 1
	}
	print("PASS\n")
	return 0
}
