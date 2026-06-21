package main

func appMain(args []string) int {
	value := byte('b')
	out := 0
	switch value {
	case 'a':
		out = 1
	case 'b':
		out = 2
	default:
		out = 3
	}
	if out != 2 {
		print("RTG-SWITCH-008 byte case failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
