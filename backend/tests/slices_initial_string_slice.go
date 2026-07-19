package main

func appMain(args []string) int {
	var xs []string
	if len(xs) != 0 {
		print("RENVO-0557 initial string slice failed\n")
		return 1
	} else {
		xs = append(xs, "red")
		xs = append(xs, "blue")
	}
	if len(xs) != 2 || xs[1] != "blue" {
		print("RENVO-0557 append strings failed\n")
		return 2
	}
	print("PASS\n")
	return 0
}
