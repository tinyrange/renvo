package main

func appendItems(out []int, values []int) []int {
	for i := 0; i < len(values); i++ {
		out = append(out, values[i])
	}
	return out
}

func buildItems() []int {
	var out []int
	var empty []int
	out = appendItems(out, empty)
	return out
}

func appMain(args []string, env []string) int {
	items := buildItems()
	if len(items) != 0 {
		print("bad length\n")
		return 1
	}
	print("PASS\n")
	return 0
}
