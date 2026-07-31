package main

func appMain(args []string) int {
	if len(args) == 2 {
		print("PASS\n")
		return 0
	}
	if len(args) == 1 {
		print("PASS\n")
		return 0
	}
	if len(args) == 0 {
		print("PASS\n")
		return 0
	}
	print("RENVO-1474 prepared_backend_equality_branch failed\n")
	return 1
}
