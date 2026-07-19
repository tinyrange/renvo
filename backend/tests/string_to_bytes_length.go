package main

func appMain(args []string) int {
	bs := []byte("zero")
	if len(bs) != 4 {
		print("RENVO-0598 length failed\n")
		return 1
	}
	if bs[0] == 0 {
		print("RENVO-0598 unexpected zero byte\n")
		return 2
	}
	print("PASS\n")
	return 0
}
