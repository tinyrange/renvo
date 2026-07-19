package main

func appMain(args []string) int {
	i := 4
	i--
	if i != 3 {
		print("RENVO-INCDEC-002 local decrement failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
