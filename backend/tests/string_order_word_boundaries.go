package main

func orderWords(a string, b string) bool {
	return a < b && a <= b && b > a && b >= a && !(a > b) && !(a >= b)
}

func appMain() int {
	left := make([]byte, 35)
	right := make([]byte, 35)
	for i := 0; i < len(left); i++ {
		left[i] = byte(128 + i)
		right[i] = left[i]
	}
	for start := 0; start < 3; start++ {
		for size := 0; size <= 32; size++ {
			a, b := string(left[start:start+size]), string(right[start:start+size])
			if a < b || a > b || !(a <= b) || !(a >= b) || !orderWords(a, b+"\x00") {
				panic("equal strings or prefix")
			}
			for at := 0; at < size; at++ {
				right[start+at]++
				if !orderWords(a, string(right[start:start+size])) {
					panic("unsigned word or byte order")
				}
				right[start+at] -= 2
				if !orderWords(string(right[start:start+size]), a) {
					panic("reverse word or byte order")
				}
				right[start+at]++
			}
		}
	}
	print("PASS\n")
	return 0
}
