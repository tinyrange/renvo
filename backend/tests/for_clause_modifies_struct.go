package main

type renvo0418Box struct{ n int }

func appMain(args []string) int {
	b := renvo0418Box{}
	for i := 0; i < 4; i = i + 1 {
		b.n = b.n + i
	}
	if b.n != 6 {
		print("RENVO-0418 struct for failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
