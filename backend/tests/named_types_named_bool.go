package main

type Renvo0654Flag bool

func renvo0654Make(v bool) Renvo0654Flag {
	return Renvo0654Flag(v)
}

func appMain(args []string) int {
	if !renvo0654Make(true) {
		print("RENVO-0654 named bool failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
