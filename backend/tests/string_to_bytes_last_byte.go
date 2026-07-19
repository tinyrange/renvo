package main

func appMain(args []string) int {
	bs := []byte("last")
	last := 0
	for i := 0; i < len(bs); i = i + 1 {
		last = i
	}
	if bs[last] != 't' {
		print("RENVO-0584 last byte failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
