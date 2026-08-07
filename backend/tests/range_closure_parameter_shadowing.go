package main

func appMain(args []string) int {
	values := []int{2, 3, 5}
	total := 0
	for _, value := range values {
		total += value
	}
	identity := func(value int) int { return value }
	if total+identity(7) != 17 {
		print("FAIL: nested closure parameter shadowed an earlier range variable\n")
		return 1
	}
	print("PASS\n")
	return 0
}
