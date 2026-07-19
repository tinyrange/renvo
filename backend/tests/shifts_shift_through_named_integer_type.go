package main

type small int

func appMain(args []string) int {
	var x small = 2
	if !(int(x<<3) == 16) {
		print("RENVO-0246 shift_through_named_integer_type failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
