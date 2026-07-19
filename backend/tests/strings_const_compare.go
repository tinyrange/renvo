package main

const strings16Const = "fixed"

func appMain(args []string) int {
	if strings16Const != "fixed" {
		print("strings_16 const\n")
		return 1
	}
	print("PASS\n")
	return 0
}
