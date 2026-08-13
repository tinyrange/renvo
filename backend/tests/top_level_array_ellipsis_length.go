package main

var topLevelArrays = [...][2]int{{7, 11}, {13, 17}, {19, 23}}

func appMain() int {
	if len(topLevelArrays) != 3 || topLevelArrays[0][1] != 11 || topLevelArrays[2][0] != 19 {
		print("top-level ellipsis array length failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
