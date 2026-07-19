package main

func main() {
	grid := [2][3]int{{1, 1, 3}, {4, 5, 3}}
	corpusOK := grid[0][1]+grid[1][2] == 4
	if !corpusOK {

		print("FAIL\n")
		return
	}
	print("PASS\n")

}
