package main

func appMain(args []string) int {
	if len(args) == 0 {
		print("missing args\n")
		return 1
	}
	if len(args[0]) == 0 {
		print("empty arg0\n")
		return 1
	}
	for i := 0; i < len(args[0]); i++ {
		if args[0][i] == 0 {
			print("embedded nul\n")
			return 1
		}
	}
	print("PASS\n")
	return 0
}
