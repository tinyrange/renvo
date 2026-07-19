package main

func appMain(args []string) int {
	var bs []byte
	i := 0
	for i < 3 {
		bs = append(bs, byte(65+i))
		i = i + 1
	}
	if len(bs) != 3 || bs[2] != byte(67) {
		print("RENVO-0385 append bytes loop failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
