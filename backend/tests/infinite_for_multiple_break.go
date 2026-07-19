package main

func appMain(args []string) int {
	i := 0
	for {
		i = i + 1
		if i == 3 {
			break
		}
		if i > 5 {
			break
		}
	}
	if i != 3 {
		print("RENVO-0447 multiple break failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
