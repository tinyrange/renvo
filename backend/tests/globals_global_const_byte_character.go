package main

const renvo0681Byte = 'A'

func appMain(args []string) int {
	if int(renvo0681Byte) != 65 {
		print("RENVO-0681 byte const failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
