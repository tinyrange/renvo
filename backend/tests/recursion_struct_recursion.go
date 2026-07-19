package main

type Renvo0509Box struct {
	count int
}

func renvo0509Fill(b *Renvo0509Box, n int) {
	if n == 0 {
		return
	}
	b.count = b.count + n
	renvo0509Fill(b, n-1)
}

func appMain(args []string) int {
	box := Renvo0509Box{count: 0}
	for i := 0; i < 1; i = i + 1 {
		renvo0509Fill(&box, 4)
	}
	if box.count != 10 {
		print("RENVO-0509 struct recursion failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
