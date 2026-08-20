package main

func syscall() int {
	return 7
}

func appMain() int {
	if syscall() != 7 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
