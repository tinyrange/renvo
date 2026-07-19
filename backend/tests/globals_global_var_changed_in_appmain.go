package main

var renvo0693Value int = 4

func appMain(args []string) int {
	renvo0693Value = renvo0693Value + 6
	if renvo0693Value != 10 {
		print("RENVO-0693 appMain global mutation failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
