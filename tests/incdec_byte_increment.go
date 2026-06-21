package main

func appMain(args []string) int {
	value := byte(64)
	value++
	if value != 'A' {
		print("RTG-INCDEC-011 byte increment failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
