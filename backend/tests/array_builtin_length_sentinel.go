package main

func appMain() int {
	text := "-t"
	values := make([]byte, 2, 7)
	empty := [0]int{}
	if len(text) != 2 || text[0] != '-' || text[1] != 't' || len(values) != 2 || cap(values) != 7 || len(empty) != 0 {
		return 1
	}
	print("PASS\n")
	return 0
}
