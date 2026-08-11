package main

func c89IndirectAdd(left int, right int) int {
	return left + right
}

func appMain() int {
	fn := c89IndirectAdd
	if fn(2, 3) != 5 {
		return 1
	}
	print("PASS\n")
	return 0
}
