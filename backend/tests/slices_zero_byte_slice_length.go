package main

func appMain(args []string) int {
	var bs []byte
	length := len(bs)
	if length != 0 {
		print("RENVO-0552 zero byte slice length failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
