package main

func renvoVFStringMatches(values ...string) bool {
	return len(values) == 3 && values[0] == "one" && values[1] == "two" && values[2] == "three"
}

func appMain(args []string) int {
	if !renvoVFStringMatches("one", "two", "three") {
		print("variadic_functions_strings literal call failed\n")
		return 1
	}
	values := []string{"one", "two", "three"}
	if !renvoVFStringMatches(values...) {
		print("variadic_functions_strings expanded call failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
