package main

func renvo0540Find(p *int) int {
	for i := 0; i < 5; i = i + 1 {
		*p = *p + i
		if i == 3 {
			return *p
		}
	}
	return -1
}

func appMain(args []string) int {
	value := 0
	if renvo0540Find(&value) != 6 {
		print("RENVO-0540 loop return failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
