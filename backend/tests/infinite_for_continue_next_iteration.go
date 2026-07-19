package main

func appMain(args []string) int {
	i := 0
	hits := 0
	for {
		i = i + 1
		if i < 3 {
			continue
		}
		hits = hits + 1
		if hits == 2 {
			break
		}
	}
	if i != 4 {
		print("RENVO-0431 continue next failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
