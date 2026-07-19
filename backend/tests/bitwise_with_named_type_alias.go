package main

type bits int

func appMain(args []string) int {
	var x bits = 0x0f
	if !(int(x&bits(0x03)) == 3) {
		print("RENVO-0218 bitwise_with_named_type_alias failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
