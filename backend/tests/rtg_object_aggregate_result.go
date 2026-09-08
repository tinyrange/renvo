package main

type pair struct { first, second int }

//export AddPair
func addPair(left, right pair) pair {
	return pair{left.first + right.first, left.second + right.second}
}

func appMain() int {
	result := addPair(pair{1, 2}, pair{10, 20})
	if result.first != 11 || result.second != 22 {
		return 1
	}
	print("PASS\n")
	return 0
}
