package main

func renvo0511Even(n int) bool {
	if n == 0 {
		return true
	}
	return renvo0511Odd(n - 1)
}

func renvo0511Odd(n int) bool {
	if n == 0 {
		return false
	}
	return renvo0511Even(n - 1)
}

func appMain(args []string) int {
	ok := true
	for i := 0; i < 6; i = i + 1 {
		if i == 3 {
			continue
		}
		if renvo0511Even(i) == renvo0511Odd(i) {
			ok = false
		}
	}
	if !ok {
		print("RENVO-0511 mutual parity failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
