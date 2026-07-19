package main

func appMain(args []string) int {
	words := make([]string, 2)
	words[0] = "go"
	words[1] = "renvo"
	if len(words) != 2 {
		print("make_slices_string_elements length failed\n")
		return 1
	}
	if words[0] != "go" || words[1] != "renvo" {
		print("make_slices_string_elements value failed\n")
		return 2
	}
	print("PASS\n")
	return 0
}
