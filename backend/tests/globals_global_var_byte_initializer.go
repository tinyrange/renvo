package main

var renvo0687Byte byte = 'z'

func appMain(args []string) int {
	if int(renvo0687Byte) != 122 {
		print("RENVO-0687 byte global init failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
