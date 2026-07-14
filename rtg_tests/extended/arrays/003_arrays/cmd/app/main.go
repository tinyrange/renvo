package main

func main() {
	grid := [2][3]int{{1, 3, 3}, {4, 5, 5}}
	for corpusAttempt := 0; corpusAttempt < 1; corpusAttempt++ {
		if grid[0][1]+grid[1][2] == 8 {
			print("PASS\n")
			return
		}
	}

	print("FAIL\n")
}
