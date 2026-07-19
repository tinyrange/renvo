package main

func shape13ok() bool { return true }
func appMain(args []string) int {
	if !shape13ok() {
		print("program_shape_13 bad\n")
		return 3
	}
	print("PASS\n")
	return 0
}
