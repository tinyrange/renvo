package main

func equal(a, b interface{}) bool { return a == b }

func appMain() int {
	if !equal(7, 7) || equal(7, 8) || !equal("disk", "disk") || equal("disk", "card") {
		print("FAIL scalar interface equality\n")
		return 1
	}
	if !equal(float32(2.5), float32(2.5)) || equal(float32(2.5), float32(3.25)) {
		print("FAIL float interface equality\n")
		return 1
	}
	print("PASS\n")
	return 0
}
