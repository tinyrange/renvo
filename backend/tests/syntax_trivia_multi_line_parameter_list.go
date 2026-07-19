package main

func renvo0811Add(
	a int,
	b int,
) int {
	return a + b
}

func appMain(args []string) int {
	if renvo0811Add(3, 4) != 7 {
		print("RENVO-0811 multiline params failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
