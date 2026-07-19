package main

func panicRecoverString() (ok bool) {
	defer func() {
		value := recover()
		ok = value != nil && value.(string) == "string value"
	}()
	panic("string value")
}

func panicRecoverInt() (ok bool) {
	defer func() {
		value := recover()
		ok = value != nil && value.(int) == 47
	}()
	panic(47)
}

func panicRecoverPointer(expected *int) (ok bool) {
	defer func() {
		value := recover()
		ok = value != nil && value.(*int) == expected
	}()
	panic(expected)
}

func panicRecoverInterface() (ok bool) {
	defer func() {
		value := recover()
		ok = value != nil && value.(string) == "interface value"
	}()
	var value interface{} = "interface value"
	panic(value)
}

func panicIndirectRecover() interface{} {
	return recover()
}

func panicRecoverMustBeDirect() (ok bool) {
	defer func() {
		indirect := panicIndirectRecover()
		value := recover()
		ok = indirect == nil && value != nil && value.(string) == "direct"
	}()
	panic("direct")
}

func appMain(args []string) int {
	value := 11
	if panicRecoverString() && panicRecoverInt() && panicRecoverPointer(&value) && panicRecoverInterface() && panicRecoverMustBeDirect() {
		print("PASS\n")
		return 0
	}
	return 1
}
