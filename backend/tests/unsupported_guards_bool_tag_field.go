package main

type taggedValue struct {
	isByte bool
	n      int
	b      byte
}

func appMain(args []string) int {
	v := taggedValue{isByte: true, n: 91, b: 'Z'}
	got := 0
	if v.isByte {
		got = int(v.b)
	} else {
		got = v.n
	}
	if got != 90 {
		print("RENVO-0839 bool tag field failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
