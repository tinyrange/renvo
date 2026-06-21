package main

func rtg1004Parts() (byte, int64) {
	return byte(65), int64(7)
}

func appMain(args []string) int {
	b, n := rtg1004Parts()
	if int(b)+int(n) != 72 {
		print("RTG-1004 byte int64 returns failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
