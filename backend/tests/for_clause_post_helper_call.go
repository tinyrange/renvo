package main

func forClausePostHelperCallStep(value *int) int {
	*value = *value + 1
	return *value
}

func forClausePostHelperCallProxy(value *int) int {
	return forClausePostHelperCallStep(&(*value))
}

func appMain(args []string) int {
	total := 0
	for i := 0; i < 4; forClausePostHelperCallProxy(&i) {
		total = total + i
	}
	if total != 6 {
		return 1
	}
	print("PASS\n")
	return 0
}
