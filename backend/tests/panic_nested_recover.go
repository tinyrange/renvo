package main

func panicRecoverNested() (outer bool, inner bool) {
	defer func() {
		value := recover()
		outer = value != nil && value.(string) == "outer"
	}()
	defer func() {
		defer func() {
			value := recover()
			inner = value != nil && value.(string) == "inner"
		}()
		panic("inner")
	}()
	panic("outer")
}

func panicReplaceNested() (ok bool) {
	defer func() {
		value := recover()
		ok = value != nil && value.(string) == "replacement"
	}()
	defer func() {
		panic("replacement")
	}()
	panic("original")
}

func panicNamedRecovered() (value int) {
	value = 73
	defer func() {
		ignored := recover()
		_ = ignored
	}()
	panic("named")
}

func panicUnnamedRecovered() int {
	defer func() {
		ignored := recover()
		_ = ignored
	}()
	panic("unnamed")
	return 99
}

func appMain(args []string) int {
	outer, inner := panicRecoverNested()
	if outer && inner && panicReplaceNested() && panicNamedRecovered() == 73 && panicUnnamedRecovered() == 0 {
		print("PASS\n")
		return 0
	}
	return 1
}
