package main

func appMain() int {
	values := [...]int{1, 2, 3}
	keyed := [...]int{2: 40, 4: 42}
	if len(values) != 3 || values[0] != 1 || values[2] != 3 {
		print("ellipsis array sequential length failed\n")
		return 1
	}
	if len(keyed) != 5 || keyed[0] != 0 || keyed[2] != 40 || keyed[3] != 0 || keyed[4] != 42 {
		print("ellipsis array keyed length failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
