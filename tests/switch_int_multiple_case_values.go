package main

func appMain(args []string) int {
	value := 4
	out := 0
	switch value {
	case 1, 3, 5:
		out = 15
	case 2, 4, 6:
		out = 24
	default:
		out = 99
	}
	if out != 24 {
		print("RTG-SWITCH-006 multiple int values failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
