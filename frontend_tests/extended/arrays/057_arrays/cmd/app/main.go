package main

func main() {
	grid := [2][3]int{{1, 7, 3}, {4, 5, 3}}
	if grid[0][1]+grid[1][2] == 10 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
