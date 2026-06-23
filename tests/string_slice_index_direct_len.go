package main

func appMain(args []string) int {
	if len(args) == 0 {
		print("RTG string slice index direct len missing args\n")
		return 1
	}
	i := 0
	arg := args[i]
	if len(arg) == 0 {
		print("RTG string slice index direct len empty arg\n")
		return 1
	}
	if len(args[i]) != len(arg) {
		print("RTG string slice index direct len mismatch\n")
		return 1
	}
	if args[i][0] != arg[0] {
		print("RTG string slice index direct byte mismatch\n")
		return 1
	}
	print("PASS\n")
	return 0
}
