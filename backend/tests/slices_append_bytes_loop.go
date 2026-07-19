package main

func appMain(args []string) int {
	var bs []byte
	if len(bs) == 0 {
		for i := 0; i < 3; i = i + 1 {
			bs = append(bs, byte(65+i))
		}
	}
	if len(bs) != 3 || bs[2] != 'C' {
		print("RENVO-0556 append bytes loop failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
