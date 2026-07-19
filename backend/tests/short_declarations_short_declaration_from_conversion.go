package main

func appMain(args []string) int {
	x := int(byte(70))
	if !(x == 70) {
		print("RENVO-0321 short_declaration_from_conversion failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
