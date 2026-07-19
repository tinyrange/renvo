package main

type renvo0400State struct {
	n  int
	ok bool
}

func appMain(args []string) int {
	s := renvo0400State{}
	for s.n < 2 {
		if s.ok {
			print("RENVO-0400 bool zero failed\n")
			return 1
		}
		s.n = s.n + 1
	}
	if s.n != 2 {
		print("RENVO-0400 int zero failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
