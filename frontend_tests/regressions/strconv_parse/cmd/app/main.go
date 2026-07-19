package main

import "strconv"

func main() {
	decimal, err1 := strconv.ParseInt("-42", 10, 64)
	hex, err2 := strconv.ParseUint("ff", 16, 64)
	auto, err3 := strconv.ParseUint("0b101", 0, 64)
	_, bad := strconv.ParseInt("no", 10, 64)
	if decimal == -42 && hex == 255 && auto == 5 && err1 == nil && err2 == nil && err3 == nil && bad != nil {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
