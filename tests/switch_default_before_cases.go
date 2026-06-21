package main

func appMain(args []string) int {
	value := 2
	out := 0
	switch value {
	default:
		out = 90
	case 1:
		out = 10
	case 2:
		out = 20
	}
	if out != 20 {
		print("RTG-SWITCH-022 default before cases failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
