package main

func appMain() int {
	values := make([]byte, 700000)
	for i := 0; i < len(values); i++ {
		values[i] = 0
	}
	values[0] = 1
	values[12345] = 1
	values[234567] = 1
	values[699999] = 1
	count := 0
	for i := 0; i < len(values); i++ {
		if values[i] != 0 {
			count++
		}
	}
	if count == 4 {
		print("PASS\n")
		return 0
	}
	print("FAIL\n")
	return 1
}
