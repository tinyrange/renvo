package main

func appMain(args []string) int {
	word := "west"
	out := 0
	switch word {
	case "north", "south":
		out = 2
	case "east", "west":
		out = 4
	default:
		out = 8
	}
	if out != 4 {
		print("RTG-SWITCH-013 string multiple values failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
