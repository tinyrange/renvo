package main

const renvo0680Flag = true

func appMain(args []string) int {
	if !renvo0680Flag {
		print("RENVO-0680 bool const failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
