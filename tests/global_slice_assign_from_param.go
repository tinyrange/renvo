package main

var saved []string

func save(values []string) {
	saved = values
}

func appMain(args []string, env []string) int {
	var input []string
	input = append(input, "rtg")
	input = append(input, "PASS")
	save(input)
	values := saved
	if len(values) < 2 {
		print("FAIL len\n")
		return 1
	}
	if values[1] != "PASS" {
		print("FAIL value\n")
		return 1
	}
	print("PASS\n")
	return 0
}
