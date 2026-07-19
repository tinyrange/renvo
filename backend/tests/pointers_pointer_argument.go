package main

func renvo0637Set(p *int) {
	goto set
set:
	*p = 20
}

func appMain(args []string) int {
	value := 0
	renvo0637Set(&value)
	if value != 20 {
		print("RENVO-0637 pointer argument failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
