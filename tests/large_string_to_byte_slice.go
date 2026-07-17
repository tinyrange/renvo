package main

func appMain() int {
	value := "0123456789abcdef"
	for len(value) < 65536 {
		value += value
	}
	data := []byte(value)
	if len(data) != 65536 || cap(data) != 65536 || data[0] != '0' || data[32768] != '0' || data[65535] != 'f' {
		print("FAIL\n")
		return 1
	}
	data[32768] = 'x'
	if value[32768] != '0' {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
