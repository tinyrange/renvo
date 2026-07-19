package main

const (
	renvoIotaSwitchA = iota
	renvoIotaSwitchB
	renvoIotaSwitchC
)

func appMain(args []string) int {
	state := renvoIotaSwitchB
	value := 0
	switch state {
	case renvoIotaSwitchA:
		value = 4
	case renvoIotaSwitchB:
		value = 9
	default:
		value = 1
	}
	if value != 9 {
		print("RENVO-IOTA-015 switch cases failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
