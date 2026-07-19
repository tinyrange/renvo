package main

type RenvoStructNamedLocalBranch struct {
	value int
	ok    bool
}

func renvoStructNamedLocalBranch(flag bool) RenvoStructNamedLocalBranch {
	out := RenvoStructNamedLocalBranch{value: 7, ok: true}
	if flag {
		return out
	}
	return RenvoStructNamedLocalBranch{value: 0, ok: false}
}

func appMain(args []string) int {
	got := renvoStructNamedLocalBranch(true)
	if got.value != 7 || !got.ok {
		print("struct named local branch return failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
