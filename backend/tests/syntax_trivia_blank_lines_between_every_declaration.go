package main

const renvo0801Value = 5

func appMain(args []string) int {

	if renvo0801Value != 5 {
		print("RENVO-0801 blank lines failed\n")
		return 1
	}

	print("PASS\n")
	return 0
}
