package main

type renvo0670Count int

func renvo0670Make() renvo0670Count {
	return renvo0670Count(12)
}

func appMain(args []string) int {
	if int(renvo0670Make()) != 12 {
		print("RENVO-0670 named return failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
