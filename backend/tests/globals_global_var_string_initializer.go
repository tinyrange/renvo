package main

var renvo0686Text string = "go"

func appMain(args []string) int {
	if renvo0686Text[1] != 'o' {
		print("RENVO-0686 string global init failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
