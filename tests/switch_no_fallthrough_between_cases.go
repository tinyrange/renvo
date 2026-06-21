package main

func appMain(args []string) int {
	value := 1
	out := 0
	switch value {
	case 1:
		out = out + 2
	case 2:
		out = out + 40
	default:
		out = out + 100
	}
	if out != 2 {
		print("RTG-SWITCH-014 no fallthrough failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
