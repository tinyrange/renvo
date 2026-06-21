package main

func rtgSwitchHelperValue(value int) int {
	switch value {
	case 0:
		return 3
	case 2:
		return 8
	default:
		return 1
	}
}

func appMain(args []string) int {
	if rtgSwitchHelperValue(2)+rtgSwitchHelperValue(5) != 9 {
		print("RTG-SWITCH-017 helper return failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
