package main

var loopVaryingValues = [3]int{11, 17, 23}

func loopVaryingLookup(index int) int {
	return loopVaryingValues[index]
}

func appMain(args []string) int {
	sum := 0
	for index := 0; index < 3; index++ {
		sum += loopVaryingLookup(index)
	}
	if sum == 51 {
		print("PASS\n")
		return 0
	}
	return 1
}
