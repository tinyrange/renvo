package main

func helper0(n int) string {
	if n == 0 {
		return "literal-0-0"
	}
	if n == 1 {
		return "literal-0-1"
	}
	if n == 2 {
		return "literal-0-2"
	}
	if n == 3 {
		return "literal-0-3"
	}
	if n == 4 {
		return "literal-0-4"
	}
	if n == 5 {
		return "literal-0-5"
	}
	if n == 6 {
		return "literal-0-6"
	}
	return "other"
}
func helper1(n int) string {
	if n == 0 {
		return "literal-1-0"
	}
	if n == 1 {
		return "literal-1-1"
	}
	if n == 2 {
		return "literal-1-2"
	}
	if n == 3 {
		return "literal-1-3"
	}
	if n == 4 {
		return "literal-1-4"
	}
	if n == 5 {
		return "literal-1-5"
	}
	if n == 6 {
		return "literal-1-6"
	}
	return "other"
}
func helper2(n int) string {
	if n == 0 {
		return "literal-2-0"
	}
	if n == 1 {
		return "literal-2-1"
	}
	if n == 2 {
		return "literal-2-2"
	}
	if n == 3 {
		return "literal-2-3"
	}
	if n == 4 {
		return "literal-2-4"
	}
	if n == 5 {
		return "literal-2-5"
	}
	if n == 6 {
		return "literal-2-6"
	}
	return "other"
}
func appMain(args []string) int {
	if helper0(0) != "literal-0-0" || helper1(0) != "literal-1-0" || helper2(0) != "literal-2-0" {
		return 1
	}
	print("PASS\n")
	return 0
}
