package main

func main() {
	grid := [2][3]int{{1, 8, 3}, {4, 5, 8}}
	if grid[0][1]+grid[1][2] == 16 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
