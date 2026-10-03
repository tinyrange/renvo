package main

func captureAlias(value int) func() int {
	type T = int
	return func() T { return value }
}

func captureStringAlias(value string) func() string {
	type T = string
	return func() T { return value }
}

func appMain(args []string) int {
	if captureAlias(42)() != 42 || captureStringAlias("yes")() != "yes" {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
