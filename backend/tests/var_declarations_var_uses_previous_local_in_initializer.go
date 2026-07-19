package main

func appMain(args []string) int {
	var x int = 5
	var y int = x + 7
	if !(y == 12) {
		print("RENVO-0293 var_uses_previous_local_in_initializer failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
