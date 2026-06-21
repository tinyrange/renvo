package main

const (
	rtgIotaSwitchA = iota
	rtgIotaSwitchB
	rtgIotaSwitchC
)

func appMain(args []string) int {
	state := rtgIotaSwitchB
	value := 0
	switch state {
	case rtgIotaSwitchA:
		value = 4
	case rtgIotaSwitchB:
		value = 9
	default:
		value = 1
	}
	if value != 9 {
		print("RTG-IOTA-015 switch cases failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
