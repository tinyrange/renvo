package main

func appMain(args []string) int {
	value := 5
	ptr := &value
	(*ptr)--
	(*ptr)--
	if value != 3 {
		print("RTG-INCDEC-009 pointer decrement failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
