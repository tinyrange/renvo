package main

type Renvo0639Box struct{ p *int }

func renvo0639Same(p *int) *int {
	return p
}

func appMain(args []string) int {
	value := 22
	box := Renvo0639Box{p: renvo0639Same(&value)}
	if *box.p != 22 {
		print("RENVO-0639 pointer helper return failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
