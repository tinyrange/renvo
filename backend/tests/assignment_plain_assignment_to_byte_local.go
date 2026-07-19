package main

func appMain(args []string) int {
	b := byte('x')
	b = 'y'
	if !(b == 'y') {
		print("RENVO-0329 plain_assignment_to_byte_local failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
