package main

func appMain() int {
	left := 137
	var data [65]byte
	right := 251
	leftPointer, rightPointer := &left, &right
	for i := 0; i < len(data); i++ {
		data[i] = byte(i + 1)
	}
	data = [65]byte{}
	for i := 0; i < len(data); i++ {
		if data[i] != 0 {
			print("FAIL\n")
			return 1
		}
	}
	if *leftPointer != 137 || *rightPointer != 251 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
