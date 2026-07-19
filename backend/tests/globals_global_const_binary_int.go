package main

const renvo0678Value = 0b101010

func appMain(args []string) int {
	if renvo0678Value != 42 {
		print("RENVO-0678 binary const failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
