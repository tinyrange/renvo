package main

type Renvo0606Wide struct{ n int64 }

func appMain(args []string) int {
	w := Renvo0606Wide{n: int64(44)}
	if w.n == 44 {
		print("PASS\n")
		return 0
	}
	print("RENVO-0606 int64 field failed\n")
	return 1
}
