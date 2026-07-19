package main

func appMain() int {
	grid := [2][3]int{{1, 0, 3}, {4, 5, 2}}
	if grid[0][1]+grid[1][2] == 2 {
		print("PASS\n")
		return 0
	}
	print("FAIL\n")
	return 0
}
