package main

const renvoSwitchExprBase = 3

func appMain(args []string) int {
	value := 5
	out := 0
	switch value {
	case renvoSwitchExprBase + 1:
		out = 10
	case renvoSwitchExprBase + 2:
		out = 20
	default:
		out = 30
	}
	if out != 20 {
		print("RENVO-SWITCH-005 case expression failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
