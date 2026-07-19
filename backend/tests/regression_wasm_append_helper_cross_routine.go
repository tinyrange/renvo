package main

func addByte(xs []byte) []byte {
	xs = append(xs, byte('a'))
	return xs
}

func appMain() int {
	xs := make([]byte, 0, 16)
	xs = addByte(xs)
	xs = append(xs, byte('b'))
	if len(xs) == 2 {
		if xs[0] == byte('a') {
			if xs[1] == byte('b') {
				print("PASS\n")
				return 0
			}
		}
	}
	print("FAIL\n")
	return 1
}
