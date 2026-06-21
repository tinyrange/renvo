package main

func appMain(args []string) int {
	i := 4
	i++
	if i != 5 {
		print("RTG-INCDEC-001 local increment failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
