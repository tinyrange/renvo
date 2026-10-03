package main

func headerPredicate(f func() bool) bool { return f() }

func appMain() int {
	if !headerPredicate(func() bool { v := true; return v }) ||
		!headerPredicate(func() bool { v := true; return v }) {
		return 1
	}
	count := 0
	for headerPredicate(func() bool { v := true; return v }) {
		count++
		break
	}
	switch headerPredicate(func() bool { v := true; return v }) {
	case true:
		count++
	default:
		return 2
	}
	if value := headerPredicate(func() bool { v := true; return v }); value {
		count++
	}
	for index := 0; index < 1 && headerPredicate(func() bool { v := true; return v }); index++ {
		count++
	}
	if count != 4 {
		return 3
	}
	print("PASS\n")
	return 0
}
