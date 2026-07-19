package main

func renvo0389More(i int) bool { return i < 3 }
func appMain(args []string) int {
	i := 0
	for renvo0389More(i) {
		i = i + 1
	}
	if i != 3 {
		print("RENVO-0389 helper condition loop failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
