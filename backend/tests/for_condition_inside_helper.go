package main

func renvo0397Count() int {
	i := 0
	for i < 6 {
		i = i + 2
	}
	return i
}
func appMain(args []string) int {
	if renvo0397Count() != 6 {
		print("RENVO-0397 helper loop failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
