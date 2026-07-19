package main

func renvo1007Loop(n int) (int, int) {
	sum := 0
	i := 0
	for {
		if i == n {
			break
		}
		sum += i
		i = i + 1
	}
	return sum, i
}

func appMain(args []string) int {
	sum, count := renvo1007Loop(5)
	if sum != 10 || count != 5 {
		print("RENVO-1007 loop return pair failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
