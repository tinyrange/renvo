package main

func appMain(args []string) int {
	buf := []byte("ABC")
	mask := int(buf[0]) ^ int(buf[1]) ^ int(buf[2])
	if !(mask == 64) {
		print("RENVO-0217 bitwise_checksum_over_byte_values failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
