package main

type RtgStructNamedLocalBranch struct {
	value int
	ok    bool
}

func rtgStructNamedLocalBranch(flag bool) RtgStructNamedLocalBranch {
	out := RtgStructNamedLocalBranch{value: 7, ok: true}
	if flag {
		return out
	}
	return RtgStructNamedLocalBranch{value: 0, ok: false}
}

func appMain(args []string) int {
	got := rtgStructNamedLocalBranch(true)
	if got.value != 7 || !got.ok {
		print("struct named local branch return failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
