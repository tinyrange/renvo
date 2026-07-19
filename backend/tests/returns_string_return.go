package main

func renvo0532Same(s string) string {
	if len(s) > 2 {
		return s
	} else {
		return "small"
	}
}

func appMain(args []string) int {
	if renvo0532Same("north") != "north" {
		print("RENVO-0532 string return failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
