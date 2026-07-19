package main

type Renvo0607Bag struct{ xs []int }

func appMain(args []string) int {
	bag := Renvo0607Bag{}
	if len(bag.xs) != 0 {
		print("RENVO-0607 initial slice field failed\n")
		return 1
	} else {
		bag.xs = append(bag.xs, 6)
	}
	if bag.xs[0] != 6 {
		print("RENVO-0607 slice field failed\n")
		return 2
	}
	print("PASS\n")
	return 0
}
