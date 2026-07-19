package main

func appMain(args []string) int {
	flag := false
	out := 0
	switch flag {
	case true:
		out = 8
	case false:
		out = 4
	}
	if out != 4 {
		print("RENVO-SWITCH-010 bool false failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
