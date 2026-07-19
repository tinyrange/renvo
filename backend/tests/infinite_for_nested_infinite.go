package main

func appMain(args []string) int {
	total := 0
	for {
		for {
			total = total + 2
			break
		}
		total = total + 1
		break
	}
	if total != 3 {
		print("RENVO-0433 nested infinite failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
