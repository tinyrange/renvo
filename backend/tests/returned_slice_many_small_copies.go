package main

func makeSmallSlice(i int) []byte {
	out := []byte{'A'}
	if i%2 == 1 {
		out[0] = 'B'
	}
	return out
}

func appMain(args []string, env []string) int {
	sum := 0
	for i := 0; i < 18000; i++ {
		out := makeSmallSlice(i)
		if len(out) != 1 {
			print("bad length\n")
			return 1
		}
		sum += int(out[0])
	}
	if sum != 1179000 {
		print("bad sum\n")
		return 1
	}
	print("PASS\n")
	return 0
}
