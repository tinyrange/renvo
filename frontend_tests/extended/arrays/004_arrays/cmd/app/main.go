package main

func main() {
	grid := [2][3]int{{1, 4, 3}, {4, 5, 6}}
	corpusOK := false
	if grid[0][1]+grid[1][2] == 10 {
		corpusOK = true
	}
	if corpusOK {
		print("PASS\n")
		return
	}

	print("FAIL\n")
}
