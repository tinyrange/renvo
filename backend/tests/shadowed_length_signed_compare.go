package main

func len(data []byte) int { return -1 }

func appMain() int {
	data := []byte{1}
	maximum := int(^uint(0) >> 1)
	if maximum < len(data) {
		panic("shadowed length comparison")
	}
	print("PASS\n")
	return 0
}
