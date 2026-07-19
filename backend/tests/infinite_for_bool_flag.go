package main

func appMain(args []string) int {
	ok := true
	count := 0
	for {
		if !ok {
			break
		}
		count = count + 1
		if count == 2 {
			ok = false
		}
	}
	if count != 2 {
		print("RENVO-0435 bool infinite failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
