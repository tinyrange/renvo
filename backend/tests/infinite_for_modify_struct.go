package main

type renvo0441Box struct{ n int }

func appMain(args []string) int {
	b := renvo0441Box{}
	for {
		b.n = b.n + 2
		if b.n == 6 {
			break
		}
	}
	if b.n != 6 {
		print("RENVO-0441 struct infinite failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
