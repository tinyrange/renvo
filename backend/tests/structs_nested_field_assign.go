package main

type Renvo0618Inner struct{ value int }
type Renvo0618Outer struct{ inner Renvo0618Inner }

func appMain(args []string) int {
	outer := Renvo0618Outer{}
	outer.inner.value = int(byte(12))
	if outer.inner.value != 12 {
		print("RENVO-0618 nested field assign failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
