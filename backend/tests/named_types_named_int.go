package main

type Renvo0651Count int

func appMain(args []string) int {
	var n Renvo0651Count = 4
	if n != 4 {
		print("RENVO-0651 named int failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
