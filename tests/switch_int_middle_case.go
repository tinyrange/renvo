package main

func appMain(args []string) int {
	value := 2
	out := 0
	switch value {
	case 1:
		out = 11
	case 2:
		out = 22
	case 3:
		out = 33
	}
	if out != 22 {
		print("RTG-SWITCH-002 middle case failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
