package main

func renvoExplicitBareReturn(value int, out *int) {
	if value == 7 {
		*out = 11
		return;
	}
	*out = 99
}

func appMain() int {
	value := 0
	renvoExplicitBareReturn(7, &value)
	if value != 11 {
		return 1
	}
	print("PASS\n")
	return 0
}
