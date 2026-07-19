package main

func appMain(args []string) int {
	var data []byte
	i := 0
	for i < 1024 {
		data = append(data, byte(i))
		i += 1
	}
	if len(data) != 1024 || data[0] != 0 || data[1023] != 255 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
