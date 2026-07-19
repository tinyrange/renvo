package main

var renvo0647Start int = 0

func appMain(args []string) int {
	p := &renvo0647Start
	for i := 0; i < 4; i = i + 1 {
		*p = *p + i
	}
	if *p != 6 {
		print("RENVO-0647 dereference loop body failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
