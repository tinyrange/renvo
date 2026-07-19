package main

func appMain(args []string) int {
	s := "bad"
	if len(args) > 0 {
		s = "good"
	} else {
		s = "bad"
	}
	if s != "good" {
		print("strings_23 select\n")
		return 1
	}
	print("PASS\n")
	return 0
}
