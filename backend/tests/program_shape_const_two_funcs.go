package main

const shapeNineUnit = 6

func shapeNineA() int { return shapeNineUnit + 1 }
func shapeNineB() int { return shapeNineUnit * 3 }
func appMain(args []string) int {
	if shapeNineA()+shapeNineB() != 25 {
		print("program_shape_09 const\n")
		return 1
	}
	print("PASS\n")
	return 0
}
