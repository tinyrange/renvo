package main

func appMain(args []string) int {
	bs := []byte("middle")
	for len(bs) < 3 {
		bs = append(bs, 'x')
	}
	if bs[3] != 'd' {
		print("RENVO-0583 middle byte failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
