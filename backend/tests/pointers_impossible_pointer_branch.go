package main

func appMain(args []string) int {
	value := 31
	p := &value
	if p != &value && *p == 31 {
		print("RENVO-0645 impossible pointer branch\n")
		return 1
	}
	if *p+1 != 32 {
		print("RENVO-0645 dereference expression failed\n")
		return 2
	}
	print("PASS\n")
	return 0
}
