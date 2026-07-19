package main

func appMain(args []string) int {
	a := "copy"
	b := a
	if b != "copy" {
		print("strings_22 copy\n")
		return 1
	}
	print("PASS\n")
	return 0
}
