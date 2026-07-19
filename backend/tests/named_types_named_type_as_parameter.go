package main

type renvo0669Score int

func renvo0669Take(v renvo0669Score) int {
	return int(v) + 2
}

func appMain(args []string) int {
	if renvo0669Take(renvo0669Score(5)) != 7 {
		print("RENVO-0669 named parameter failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
