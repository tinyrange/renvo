package main

func wordStringsEqual(a string, b string) bool { return a == b }

func appMain() int {
	left := make([]byte, 35)
	right := make([]byte, 35)
	for i := 0; i < len(left); i++ {
		left[i] = byte(i*11 + 3)
		right[i] = left[i]
	}
	for start := 0; start < 3; start++ {
		for size := 0; size <= 32; size++ {
			a, b := string(left[start:start+size]), string(right[start:start+size])
			if !wordStringsEqual(a, b) || wordStringsEqual(a, b+"x") {
				panic("equal length or tail")
			}
			for at := 0; at < size; at++ {
				right[start+at] ^= 1
				if wordStringsEqual(a, string(right[start:start+size])) {
					panic("word or tail mismatch")
				}
				right[start+at] ^= 1
			}
		}
	}
	print("PASS\n")
	return 0
}
