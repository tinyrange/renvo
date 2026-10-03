package main

func immediatePopLess(value int) bool { return value < 381301 }

func appMain() int {
	if !immediatePopLess(0) || !immediatePopLess(381300) || immediatePopLess(381301) || immediatePopLess(381302) {
		return 1
	}
	print("PASS\n")
	return 0
}
