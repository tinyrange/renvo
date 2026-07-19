package main

func renvo0815Value() int { return 8 }

func appMain(args []string) int {
	if renvo0815Value() != 8 {
		print("RENVO-0815 same-line return failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
