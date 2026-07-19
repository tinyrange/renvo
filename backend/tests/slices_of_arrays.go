package main

func appMain() int {
	rows := [][2]int{{1, 2}, {3, 4}}
	if len(rows) == 2 && rows[0][0] == 1 && rows[0][1] == 2 && rows[1][0] == 3 && rows[1][1] == 4 {
		print("PASS\n")
		return 0
	}
	print("FAIL\n")
	return 1
}
