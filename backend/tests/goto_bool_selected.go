package main

func appMain(args []string) int {
	ok := true
	x := 0
	if ok {
		goto good
	}
	x = 9
good:
	if x != 0 {
		print("RENVO-0462 bool goto failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
