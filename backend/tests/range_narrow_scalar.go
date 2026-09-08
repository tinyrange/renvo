package main

func appMain() int {
	bytes := []byte("ABC")
	sum := 0
	for _, b := range bytes {
		sum += int(b)
	}
	values := []int16{-1234, 7, 30000}
	for _, v := range values {
		sum += int(v)
	}
	if sum != 28971 {
		print("FAIL narrow range\n")
		return 1
	}
	print("PASS\n")
	return 0
}
