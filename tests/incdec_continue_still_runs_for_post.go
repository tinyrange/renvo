package main

func appMain(args []string) int {
	hits := 0
	for i := 0; i < 5; i++ {
		if i < 4 {
			continue
		}
		hits++
	}
	if hits != 1 {
		print("RTG-INCDEC-005 continue post failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
