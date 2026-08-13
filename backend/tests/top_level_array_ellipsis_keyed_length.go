package main

const topLevelArrayBase = 2

var topLevelKeyedArray = [...]int{topLevelArrayBase: 40, topLevelArrayBase + 2: 42}

func appMain() int {
	if len(topLevelKeyedArray) != 5 || topLevelKeyedArray[0] != 0 || topLevelKeyedArray[2] != 40 || topLevelKeyedArray[4] != 42 {
		print("top-level keyed ellipsis array length failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
