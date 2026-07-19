package main

func renvo0831IntPath(x int) int {
	return x + 1
}

func renvo0831BytePath(x byte) int {
	return int(x) + 2
}

func appMain(args []string) int {
	if renvo0831IntPath(4)+renvo0831BytePath(byte(5)) != 12 {
		print("RENVO-0831 split helpers failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
