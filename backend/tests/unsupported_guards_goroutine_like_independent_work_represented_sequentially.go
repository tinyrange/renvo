package main

func renvo0835WorkA() int {
	return 3
}

func renvo0835WorkB() int {
	return 4
}

func appMain(args []string) int {
	if renvo0835WorkA()*renvo0835WorkB() != 12 {
		print("RENVO-0835 sequential work failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
