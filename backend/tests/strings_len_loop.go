package main

func appMain(args []string) int {
	s := "loop"
	count := 0
	for i := 0; i < len(s); i = i + 1 {
		count += 1
	}
	if count != 4 {
		print("strings_12 loop\n")
		return 1
	}
	print("PASS\n")
	return 0
}
