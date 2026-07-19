package main

func appMain(args []string) int {
	bs := []byte("az")
	if int(bs[0])+int(bs[1]) != 219 {
		print("RENVO-0593 byte int conversion failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
