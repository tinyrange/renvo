package main

func shape23RegionOne() int { return 14 }

func appMain(args []string) int {
	if shape23RegionOne()+shape23RegionTwo() != 39 {
		print("program_shape_23 regions\n")
		return 1
	}
	print("PASS\n")
	return 0
}

func shape23RegionTwo() int { return 25 }
