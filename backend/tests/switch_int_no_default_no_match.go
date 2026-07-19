package main

func appMain(args []string) int {
	value := 9
	out := 7
	switch value {
	case 1:
		out = 11
	case 2:
		out = 22
	}
	if out != 7 {
		print("RENVO-SWITCH-004 no default failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
