package main

func appMain(args []string) int {
	bs := []byte("az")
	bs = append(bs, '!')
	if len(bs) != 3 || bs[1] != 'z' {
		print("RENVO-0592 char compare failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
