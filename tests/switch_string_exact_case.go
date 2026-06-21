package main

func appMain(args []string) int {
	word := "green"
	out := 0
	switch word {
	case "red":
		out = 1
	case "green":
		out = 2
	case "blue":
		out = 3
	}
	if out != 2 {
		print("RTG-SWITCH-011 string exact failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
