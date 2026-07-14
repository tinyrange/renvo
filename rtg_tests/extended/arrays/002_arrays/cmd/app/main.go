package main

func main() {
	grid := [2][3]int{{1, 2, 3}, {4, 5, 4}}
	if grid[0][1]+grid[1][2] == 6 {
		print("PASS\n")
		return
	} else {

		print("FAIL\n")
	}
}
