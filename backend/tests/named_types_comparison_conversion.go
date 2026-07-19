package main

func appMain(args []string) int {
	var xs []byte
	xs = append(xs, byte(9))
	if int(xs[0]) == 9 {
		print("PASS\n")
		return 0
	}
	print("RENVO-0666 comparison conversion failed\n")
	return 1
}
