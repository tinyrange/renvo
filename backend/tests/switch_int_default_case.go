package main

func appMain(args []string) int {
	value := 9
	out := 0
	switch value {
	case 1:
		out = 11
	case 2:
		out = 22
	default:
		out = 33
	}
	if out != 33 {
		print("RENVO-SWITCH-003 default case failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
