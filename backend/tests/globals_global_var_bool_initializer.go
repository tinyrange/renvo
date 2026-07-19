package main

var renvo0685Flag bool = true

func appMain(args []string) int {
	if !renvo0685Flag {
		print("RENVO-0685 bool global init failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
