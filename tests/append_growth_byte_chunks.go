package main

func appMain(args []string) int {
	capacity := 1200
	buf := make([]byte, 0, capacity)
	chunk := []byte("abcdefghijklmnopqrstuvwxyz")
	i := 0
	for i < 50 {
		buf = append(buf, chunk...)
		i++
	}
	if len(buf) != 1300 {
		print("FAIL\n")
		return 1
	}
	if buf[0] != 'a' {
		print("FAIL\n")
		return 1
	}
	if buf[25] != 'z' {
		print("FAIL\n")
		return 1
	}
	if buf[26] != 'a' {
		print("FAIL\n")
		return 1
	}
	if buf[1299] != 'z' {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
