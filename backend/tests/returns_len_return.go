package main

var renvo0547Text string = "abcde"

func renvo0547Len(s string) int {
	return len(s) + len(renvo0547Text)
}

func appMain(args []string) int {
	if renvo0547Len("xy") != 7 {
		print("RENVO-0547 len return failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
