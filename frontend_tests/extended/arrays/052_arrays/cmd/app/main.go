package main

func main() {
	grid := [2][3]int{{1, 2, 3}, {4, 5, 5}}
	if grid[0][1]+grid[1][2] == 7 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
