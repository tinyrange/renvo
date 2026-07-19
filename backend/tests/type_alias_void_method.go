package main

type aliasMethodValue struct {
	n int
}

type aliasMethodFirst = aliasMethodValue
type aliasMethodSecond = aliasMethodFirst

func (v *aliasMethodValue) reset() {
	v.n = 0
}

func appMain(args []string) int {
	v := &aliasMethodSecond{n: 1}
	v.reset()
	if v.n != 0 {
		return 1
	}
	print("PASS\n")
	return 0
}
