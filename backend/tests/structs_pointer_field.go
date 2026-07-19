package main

type Renvo0608Ref struct{ p *int }

func appMain(args []string) int {
	value := 3
	box := Renvo0608Ref{p: &value}
	for *box.p < 7 {
		*box.p = *box.p + 2
	}
	if value != 7 {
		print("RENVO-0608 pointer field failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
