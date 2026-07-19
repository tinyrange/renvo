package main

func appMain(args []string) int {
	if !renvo0505Even(8) {
		print("RENVO-0505 even true failed\n")
		return 1
	}
	if renvo0505Even(7) {
		print("RENVO-0505 even false failed\n")
		return 2
	}
	print("PASS\n")
	return 0
}

func renvo0505Even(n int) bool {
	if n == 0 {
		return true
	}
	if n == 1 {
		return false
	}
	return renvo0505Even(n - 2)
}
