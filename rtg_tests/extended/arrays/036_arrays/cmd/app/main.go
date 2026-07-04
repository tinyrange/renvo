package main

func main() {
	grid := [2][3]int{{1, 6, 3}, {4, 5, 3}}
	if grid[0][1]+grid[1][2] == 9 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
