package main

func appMain() int {
	s := "PASS\n"
	if s == "" {
		print("FAIL\n")
		return 1
	}
	print(s)
	return 0
}
