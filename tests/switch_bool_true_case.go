package main

func appMain(args []string) int {
	flag := true
	out := 0
	switch flag {
	case true:
		out = 8
	case false:
		out = 4
	}
	if out != 8 {
		print("RTG-SWITCH-009 bool true failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
