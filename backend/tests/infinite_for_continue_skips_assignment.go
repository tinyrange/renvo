package main

func appMain(args []string) int {
	i := 0
	bad := 0
	for {
		i = i + 1
		if i < 3 {
			continue
		}
		if i == 3 {
			break
		}
		bad = 1
	}
	if bad != 0 || i != 3 {
		print("RENVO-0430 continue skip failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
