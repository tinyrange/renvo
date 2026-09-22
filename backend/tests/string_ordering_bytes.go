package main

func appMain(args []string) int {
	values := []string{"", "\x00", "\x00\xff", "\x7f", "\x80", "\xff"}
	for i := 0; i < len(values); i++ {
		for j := 0; j < len(values); j++ {
			a := values[i]
			b := values[j]
			if (a < b) != (i < j) || (a <= b) != (i <= j) {
				return 1
			}
			if (a > b) != (i > j) || (a >= b) != (i >= j) {
				return 2
			}
		}
	}
	print("PASS\n")
	return 0
}
