package main

type Renvo0660Count int

func appMain(args []string) int {
	for {
		n := Renvo0660Count(12)
		if n == 12 {
			print("PASS\n")
			return 0
		}
		break
	}
	print("RENVO-0660 int to named failed\n")
	return 1
}
