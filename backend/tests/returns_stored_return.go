package main

const renvo0546Want = 14

func renvo0546Value() int {
	return 14
}

func appMain(args []string) int {
	value := renvo0546Value()
	if value != renvo0546Want {
		print("RENVO-0546 stored return failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
