package main

const rtgSwitchInt64A int64 = 11
const rtgSwitchInt64B int64 = 12

func appMain(args []string) int {
	var value int64 = 12
	out := 0
	switch value {
	case rtgSwitchInt64A:
		out = 1
	case rtgSwitchInt64B:
		out = 2
	default:
		out = 3
	}
	if out != 2 {
		print("RTG-SWITCH-007 int64 case failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
