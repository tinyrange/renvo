package main

var calls int

func nextGroupValue() int {
	calls++
	return calls
}

func appMain(args []string) int {
	seed := 9
	{
		var (
			seed = seed + 1
			a, b = nextGroupValue(), nextGroupValue()
			zero int
		)
		if seed != 10 || a != 1 || b != 2 || zero != 0 {
			return 1
		}
		zero = seed
		if zero != 10 {
			return 1
		}
	}
	if seed != 9 || calls != 2 {
		return 1
	}
	print("PASS\n")
	return 0
}
