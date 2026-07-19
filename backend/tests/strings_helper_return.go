package main

func strings14() string { return "ret" }
func appMain(args []string) int {
	if strings14() != "ret" {
		print("strings_14 return\n")
		return 1
	}
	print("PASS\n")
	return 0
}
