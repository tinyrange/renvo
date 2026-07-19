package main

func appMain(args []string) int {
	enabled := true
	state := "run"
	out := 0
	if enabled {
		switch state {
		case "stop":
			out = 1
		case "run":
			out = 5
		default:
			out = 9
		}
	} else {
		out = 99
	}
	if out != 5 {
		print("RENVO-SWITCH-020 nested if failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
