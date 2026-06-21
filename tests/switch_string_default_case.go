package main

func appMain(args []string) int {
	word := "amber"
	out := 0
	switch word {
	case "red":
		out = 1
	case "green":
		out = 2
	default:
		out = 9
	}
	if out != 9 {
		print("RTG-SWITCH-012 string default failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
