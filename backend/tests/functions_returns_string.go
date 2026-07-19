package main

func renvo0485Word() string { return "ok" }
func appMain(args []string) int {
	if renvo0485Word() != "ok" {
		print("RENVO-0485 string return failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
