package main

type Renvo0603Flag struct {
	value int
	ok    bool
}

var renvo0603Global = Renvo0603Flag{value: 5, ok: true}

func appMain(args []string) int {
	if renvo0603Global.value != 5 || !renvo0603Global.ok {
		print("RENVO-0603 int bool struct failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
