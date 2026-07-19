package main

func appMain(args []string) int {
	s := "bad"
	goto choose
	s = "no"
choose:
	s = "goto"
	if s != "goto" {
		print("strings_24 goto\n")
		return 1
	}
	print("PASS\n")
	return 0
}
