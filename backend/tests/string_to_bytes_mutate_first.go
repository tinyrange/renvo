package main

func appMain(args []string) int {
	bs := []byte("map")
	for {
		bs[0] = 't'
		break
	}
	if bs[0] != 't' || bs[1] != 'a' {
		print("RENVO-0585 mutate first failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
