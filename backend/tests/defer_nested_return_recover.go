package main

func nestedReturnRecovery() (outer, inner bool) {
	defer func() { outer = recover() != nil }()
	defer func() {
		defer func() { inner = recover() != nil }()
	}()
	panic("original")
}

func appMain() int {
	outer, inner := nestedReturnRecovery()
	if !outer || inner {
		panic("nested return recovered its caller's panic")
	}
	print("PASS\n")
	return 0
}
