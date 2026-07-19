package main

var renvo0553Items []int

func appMain(args []string) int {
	renvo0553Items = append(renvo0553Items, 8)
	if len(renvo0553Items) != 1 || renvo0553Items[0] != 8 {
		print("RENVO-0553 append one int failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
