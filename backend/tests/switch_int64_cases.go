package main

const renvoSwitchInt64A int64 = 11
const renvoSwitchInt64B int64 = 12

func appMain(args []string) int {
	var value int64 = 12
	out := 0
	switch value {
	case renvoSwitchInt64A:
		out = 1
	case renvoSwitchInt64B:
		out = 2
	default:
		out = 3
	}
	if out != 2 {
		print("RENVO-SWITCH-007 int64 case failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
