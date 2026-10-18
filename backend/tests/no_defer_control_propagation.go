package main

func plain(value int) int            { return value + 1 }
func forwarded(value int) (int, int) { return plain(value), plain(value + 1) }
func indirectRecover() bool          { return recover() != nil }
func throwNoDefers()                 { panic("original") }

func owner() (outer, stolen, complete bool) {
	defer func() { outer = recover() != nil }()
	defer func() {
		stolen = indirectRecover()
		first, second := forwarded(10)
		complete = first == 11 && second == 12
	}()
	throwNoDefers()
	return
}

func appMain() int {
	outer, stolen, complete := owner()
	if !outer || stolen || !complete {
		panic("empty-defer helper changed panic ownership")
	}
	first, second := forwarded(20)
	if first != 21 || second != 22 {
		panic("empty-defer helper changed tuple return")
	}
	print("PASS\n")
	return 0
}
