package main

func appMain(args []string) int {
	hits := 0
	for i := 0; i < 4; i = i + 1 {
		if i < 3 {
			continue
		}
		hits = hits + 1
	}
	if hits != 1 {
		print("RENVO-0425 continue post executes failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
